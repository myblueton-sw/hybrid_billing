import {
  AdminError,
  type Attempt,
  type CommandId,
  type Outcome,
  type AccessSetting,
} from "../../../../service/admin";
import type {
  ScopeRef,
  ServiceLimits,
} from "../../../../service/service-query-types";
import { utf8Length } from "../../../../service/helpers";
import { nextPhase, outsideWindow } from "./live-model";
import type { Clocks } from "./issue-attempts";

export type IntentKind = "create" | "declare" | "state" | "revoke";
export type CommandDraft = Readonly<{
  command: CommandId;
  target: string;
  reason: string;
  fields: Readonly<Record<string, string>>;
  revision: string;
  replay: number;
}>;
export type Intent = {
  readonly draft: CommandDraft;
  readonly attempt: Readonly<Attempt>;
  readonly first: { mono: number; wall: number };
  phase: "sending" | "unknown" | "lapsed" | "done" | "refused";
  error?: unknown;
  outcome?: Outcome;
};
export type DeclarationDraft = Readonly<{
  target: string;
  name: string;
  owner: string;
  purpose: string;
  organization: string;
  scopes: readonly ScopeRef[];
  catalog: string;
  version: string;
  reason: string;
}>;
export function settingValue(
  settings: AccessSetting[],
  name: string,
): number | null {
  const matching = settings.filter((s) => s.name === name);
  return matching.length === 1 &&
    Number.isSafeInteger(matching[0].value) &&
    matching[0].value > 0
    ? matching[0].value
    : null;
}
export function declarationFields(
  draft: DeclarationDraft,
): Record<string, string> {
  const scopes = draft.scopes
    .map((s) => ({ audience: s.audience, scope: s.scope }))
    .sort(
      (a, b) =>
        a.audience.localeCompare(b.audience) || a.scope.localeCompare(b.scope),
    );
  const fields: Record<string, string> = {
    purpose: draft.purpose,
    organization: draft.organization,
    scopes: JSON.stringify(scopes),
    catalog_version: draft.catalog,
  };
  if (draft.target) fields.declaration_version = draft.version;
  else Object.assign(fields, { name: draft.name, owner: draft.owner });
  return fields;
}
export function admitsCommand(
  draft: CommandDraft,
  limits: ServiceLimits,
  tried: Attempt,
): boolean {
  if (utf8Length(draft.reason) > limits.rationale_bytes) return false;
  if (
    Object.entries(draft.fields).some(
      ([key, value]) =>
        utf8Length(key) > 128 ||
        BigInt(utf8Length(value)) > BigInt(limits.command_value_bytes),
    )
  )
    return false;
  const body = JSON.stringify({
    request_id: tried.request,
    target_id: draft.target,
    idempotency_key: tried.key,
    expected_revision: draft.revision,
    rationale: draft.reason,
    body: draft.fields,
  });
  return utf8Length(body) <= limits.body_bytes;
}
export function admitsDeclaration(
  draft: DeclarationDraft,
  limits: ServiceLimits,
  settings: AccessSetting[],
): boolean {
  const purpose = settingValue(settings, "service_declaration_purpose_bytes"),
    scopes = settingValue(settings, "service_declaration_scopes");
  if (
    !purpose ||
    !scopes ||
    utf8Length(draft.purpose) > purpose ||
    draft.scopes.length > scopes
  )
    return false;
  if (
    [
      draft.target,
      draft.owner,
      draft.organization,
      ...draft.scopes.flatMap((s) => [s.audience, s.scope]),
    ].some((v) => v !== "" && !/^[A-Za-z0-9._:-]{1,128}$/.test(v))
  )
    return false;
  if (
    draft.name.length > 48 ||
    draft.catalog.length > 64 ||
    draft.version.length > 20
  )
    return false;
  return admitsCommand(
    {
      command: draft.target ? "service.declare" : "service.create",
      target: draft.target || "services",
      reason: draft.reason,
      fields: declarationFields(draft),
      revision: "r".repeat(128),
      replay: 1,
    },
    limits,
    { request: "r".repeat(128), key: "k".repeat(128) },
  );
}
function completed(draft: CommandDraft, outcome: Outcome): boolean {
  if (
    outcome.status !== "completed" ||
    outcome.job_id !== "" ||
    !outcome.revision
  )
    return false;
  const states: Partial<Record<CommandId, string>> = {
    "service.create": "service_created",
    "service.declare": "service_declared",
    "service.suspend": "service_suspended",
    "service.reinstate": "service_active",
    "service.retire": "service_retired",
    "service.revoke_credential": "credential_revoked",
  };
  if (outcome.state !== states[draft.command]) return false;
  if (draft.command === "service.create")
    return outcome.fields.declaration_version === "1";
  if (draft.command === "service.declare")
    return (
      outcome.fields.declaration_version ===
      String(BigInt(draft.fields.declaration_version) + 1n)
    );
  return Object.keys(outcome.fields).length === 0;
}
export function commandIntents(
  fresh: () => Attempt,
  clocks: Clocks,
  current: () => boolean,
) {
  const intents = new Map<IntentKind, Intent>(),
    drafts = new Map<"create" | "declare", DeclarationDraft>();
  const listeners = new Set<() => void>();
  let disposed = false,
    version = 0;
  const active = () => !disposed && current();
  const changed = () => {
    version++;
    for (const fn of listeners) fn();
  };
  const elapsed = (entry: Intent) =>
    Math.max(
      clocks.mono() - entry.first.mono,
      clocks.wall() - entry.first.wall,
    );
  return {
    subscribe(fn: () => void) {
      listeners.add(fn);
      return () => {
        listeners.delete(fn);
      };
    },
    version: () => version,
    guarded: () =>
      active() &&
      (drafts.size > 0 ||
        [...intents.values()].some(
          (entry) => entry.phase !== "done" && entry.phase !== "refused",
        )),
    get: (kind: IntentKind) => (active() ? intents.get(kind) : undefined),
    draft: (kind: "create" | "declare") =>
      active() ? drafts.get(kind) : undefined,
    save(
      kind: "create" | "declare",
      draft: DeclarationDraft,
      limits: ServiceLimits,
      settings: AccessSetting[],
    ) {
      if (!active() || !admitsDeclaration(draft, limits, settings))
        return false;
      const prior = drafts.get(kind);
      if (kind === "declare" && prior && prior.target !== draft.target)
        return false;
      drafts.set(
        kind,
        Object.freeze({
          ...draft,
          scopes: Object.freeze(
            draft.scopes.map((s) => Object.freeze({ ...s })),
          ),
        }),
      );
      changed();
      return true;
    },
    discardDraft(kind: "create" | "declare") {
      if (active()) {
        drafts.delete(kind);
        changed();
      }
    },
    discard(kind: IntentKind) {
      if (!active() || intents.get(kind)?.phase === "sending") return false;
      intents.delete(kind);
      changed();
      return true;
    },
    dispose() {
      disposed = true;
      intents.clear();
      drafts.clear();
      changed();
      listeners.clear();
    },
    async submit(
      kind: IntentKind,
      proposed: CommandDraft | undefined,
      limits: ServiceLimits | undefined,
      exec: (draft: CommandDraft, tried: Attempt) => Promise<Outcome>,
    ) {
      if (!active()) return undefined;
      let entry = intents.get(kind);
      const retry = !!entry;
      if (entry && (proposed || entry.phase !== "unknown")) return undefined;
      if (entry && outsideWindow(elapsed(entry), entry.draft.replay)) {
        entry.phase = "lapsed";
        changed();
        return undefined;
      }
      if (!entry) {
        if (!proposed || !limits) return undefined;
        const tried = Object.freeze({ ...fresh() });
        if (!admitsCommand(proposed, limits, tried)) return undefined;
        entry = {
          draft: Object.freeze({
            ...proposed,
            fields: Object.freeze({ ...proposed.fields }),
          }),
          attempt: tried,
          first: { mono: clocks.mono(), wall: clocks.wall() },
          phase: "sending",
        };
        intents.set(kind, entry);
      }
      entry.phase = "sending";
      entry.error = undefined;
      changed();
      let outcome: Outcome | undefined, error: unknown;
      try {
        outcome = await exec(entry.draft, entry.attempt);
      } catch (err) {
        error = err;
      }
      if (!active() || intents.get(kind) !== entry) return undefined;
      if (outcome && completed(entry.draft, outcome)) {
        entry.phase = "done";
        entry.outcome = outcome;
        changed();
        return outcome;
      }
      entry.error = error ?? new AdminError("unavailable");
      const next = nextPhase(
        retry,
        entry.error,
        outsideWindow(elapsed(entry), entry.draft.replay, true),
      );
      entry.phase = next === "remove" ? "refused" : next;
      changed();
      return undefined;
    },
  };
}
export type CommandIntents = ReturnType<typeof commandIntents>;
