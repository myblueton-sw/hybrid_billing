import { Refusals } from "../../../../service/refusal";
import type { RowDiagnostic } from "../../../../service/command-diagnostics";
import type { Owner, ArtifactEntry, ArtifactRow, ClassGroup, Reference, HoldRow, ImpactNow, Person, PolicyRow, PolicyView, Proposal, RetentionPage, RunRef, RunRow, StoreRow } from "../../../../service/retention";
import { classes } from "./live-codes";

export const views = ["policies", "expiry", "holds", "deletion", "restore", "decisions"];

export function viewOf(v: string): string {
  return views.includes(v) ? v : "policies";
}

export const shownLimit = 50;
export const maxReason = 256;
export const maxText = 200;
export const maxPeriod = 36500;
export const maxGrace = 365;

export type CommandKey = "retention.approve" | "retention.withdraw" | "retention.propose_policy" | "retention.place_hold" | "retention.propose_release" | "retention.stop_run";
export const commandKeys: CommandKey[] = ["retention.approve", "retention.withdraw", "retention.propose_policy", "retention.place_hold", "retention.propose_release", "retention.stop_run"];

export function isProposalCommand(command: CommandKey): boolean {
  return command === "retention.approve" || command === "retention.withdraw";
}

const idChars = /^[A-Za-z0-9._:-]{1,128}$/;
export const proposalPrefix = "retention-proposal-";
export const classPrefix = "retention-class:";
export const artifactPrefix = "retention-artifact:";
export const tenantPrefix = "retention-tenant:";
export const holdPrefix = "retention-hold:";
export const runPrefix = "retention-run-";

export function validTarget(t: string): boolean {
  return idChars.test(t);
}

export function validId(id: string): boolean {
  return idChars.test(id);
}

export function proposalTarget(p: Proposal): string {
  if (p.number === null || !/^retention-proposal-[1-9][0-9]*$/.test(p.target)) return "";
  return p.target === proposalPrefix + p.number && validTarget(p.target) ? p.target : "";
}

export function classTarget(row: PolicyRow): string {
  if (!classes.includes(row.class)) return "";
  return row.target === classPrefix + row.class && validTarget(row.target) ? row.target : "";
}

export function artifactTarget(a: ArtifactRow): string {
  if (!a.id || !a.hold_target) return "";
  return a.hold_target === artifactPrefix + a.id && validTarget(a.hold_target) ? a.hold_target : "";
}

export function tenantTarget(page: RetentionPage, workspace: string): string {
  const t = page.holds.tenant_target;
  if (!/^retention-tenant:[A-Za-z0-9._:-]+$/.test(t) || !workspace) return "";
  return t === tenantPrefix + workspace && validTarget(t) ? t : "";
}

export function holdTarget(h: HoldRow): string {
  if (!h.id) return "";
  return h.target === holdPrefix + h.id && validTarget(h.target) ? h.target : "";
}

export function runNumber(run: RunRef): string {
  if (!run) return "";
  const m = /^retention-run-([1-9][0-9]*)$/.exec(run.target);
  if (!m || run.label !== "run/" + m[1] || !Number.isSafeInteger(Number(m[1]))) return "";
  return m[1];
}

export function runTarget(r: RunRow): string {
  const n = runNumber(r.run);
  return n && r.run && validTarget(r.run.target) ? r.run.target : "";
}

export function frozenTargetOk(command: CommandKey, target: string): boolean {
  if (!validTarget(target)) return false;
  switch (command) {
    case "retention.approve":
    case "retention.withdraw":
      return /^retention-proposal-[1-9][0-9]*$/.test(target);
    case "retention.propose_policy":
      return classes.some((c) => target === classPrefix + c);
    case "retention.place_hold":
      return (target.startsWith(artifactPrefix) && target.length > artifactPrefix.length) || (target.startsWith(tenantPrefix) && target.length > tenantPrefix.length);
    case "retention.propose_release":
      return target.startsWith(holdPrefix) && target.length > holdPrefix.length;
    default:
      return /^retention-run-[1-9][0-9]*$/.test(target);
  }
}

export function commandsAllowed(page: RetentionPage): boolean {
  return page.revision !== "";
}

export function timeKnown(iso: string): boolean {
  if (!iso) return false;
  const d = new Date(iso);
  return !Number.isNaN(d.getTime()) && d.getUTCFullYear() > 1;
}

export function actor(p: Person): string {
  return p ? p.login || p.id : "";
}

export const digestPattern = /^[0-9a-f]{64}$/;

export function policyKnown(p: PolicyView | null): p is PolicyView & { revision: number; period_days: number; grace_days: number } {
  return !!p && p.revision !== null && p.period_days !== null && p.grace_days !== null;
}

export type Current = { kind: "none" } | { kind: "unknown" } | { kind: "known"; revision: number; period: number; grace: number };

export function currentOf(row: PolicyRow | undefined): Current {
  if (!row) return { kind: "unknown" };
  if (row.policy === null) return { kind: "none" };
  return policyKnown(row.policy) ? { kind: "known", revision: row.policy.revision, period: row.policy.period_days, grace: row.policy.grace_days } : { kind: "unknown" };
}

export function currentContext(c: Current): Record<string, string> {
  return c.kind === "known" ? { current: "known", revision: String(c.revision), period: String(c.period), grace: String(c.grace) } : { current: c.kind };
}

export function currentFrom(ctx: Record<string, string>): Current {
  if (ctx.current === "none") return { kind: "none" };
  const n = (k: string) => (/^(0|[1-9][0-9]*)$/.test(ctx[k] ?? "") ? Number(ctx[k]) : null);
  const [r, p, g] = [n("revision"), n("period"), n("grace")];
  if (ctx.current === "known" && r !== null && p !== null && g !== null) return { kind: "known", revision: r, period: p, grace: g };
  return { kind: "unknown" };
}

export function impactValid(i: ImpactNow | null | "invalid"): i is ImpactNow & { count: number; workspace_count: number } {
  return !!i && i !== "invalid" && timeKnown(i.as_of) && i.count !== null && i.workspace_count !== null && digestPattern.test(i.digest);
}

export type Narrow = "" | "no_command" | "impact_unreadable";

export function approveNarrow(p: Proposal): Narrow {
  if (!proposalTarget(p)) return "no_command";
  if (p.kind !== "policy" && p.kind !== "release") return "no_command";
  if (p.kind === "policy") {
    if (p.impact_required === true) return impactValid(p.impact_now) ? "" : "impact_unreadable";
    if (p.impact_required === false) return p.impact_now === null ? "" : "impact_unreadable";
    return "impact_unreadable";
  }
  return "";
}

export function impactValues(p: Proposal): Record<string, string> {
  if (p.kind !== "policy" || p.impact_required !== true || !impactValid(p.impact_now)) return {};
  return { impact_as_of: p.impact_now.as_of, impact_count: String(p.impact_now.count), impact_digest: p.impact_now.digest };
}

export type NumberProblem = "" | "missing" | "range";

export function periodProblem(v: string): NumberProblem {
  const s = v.trim();
  if (!s) return "missing";
  if (!/^[0-9]{1,5}$/.test(s)) return "range";
  const n = Number(s);
  return n >= 1 && n <= maxPeriod ? "" : "range";
}

export function graceProblem(v: string): NumberProblem {
  const s = v.trim();
  if (!s) return "missing";
  if (!/^[0-9]{1,3}$/.test(s)) return "range";
  const n = Number(s);
  return n >= 0 && n <= maxGrace ? "" : "range";
}

export function canonical(v: string): string {
  return String(Number(v.trim()));
}

export function sameAsCurrent(c: Current, period: string, grace: string): boolean {
  if (c.kind !== "known" || periodProblem(period) || graceProblem(grace)) return false;
  return Number(canonical(period)) === c.period && Number(canonical(grace)) === c.grace;
}

export type Change = { kind: "unknown" } | { kind: "shorter"; days: number } | { kind: "longer"; days: number } | { kind: "same" } | { kind: "new" };

export function changeOf(c: Current, period: number, grace: number): Change {
  if (c.kind === "unknown") return { kind: "unknown" };
  if (c.kind === "none") return { kind: "new" };
  const d = period + grace - (c.period + c.grace);
  return d < 0 ? { kind: "shorter", days: -d } : d > 0 ? { kind: "longer", days: d } : { kind: "same" };
}

export function utf8Bytes(v: string): number {
  return new TextEncoder().encode(v).length;
}

export function reasonProblem(v: string): "" | "missing" | "long" {
  const t = v.trim();
  if (!t) return "missing";
  return utf8Bytes(t) > maxReason ? "long" : "";
}

export function textProblem(v: string): "" | "missing" | "long" | "control" {
  if (/[\u0000-\u001F\u007F-\u009F]/.test(v)) return "control";
  const t = v.trim();
  if (!t) return "missing";
  return utf8Bytes(t) > maxText ? "long" : "";
}

export function ordered(list: Proposal[]): { pending: Proposal[]; closed: Proposal[] } {
  const sorted = [...list].sort((a, b) => (b.number ?? -1) - (a.number ?? -1));
  return { pending: sorted.filter((p) => p.state === "pending"), closed: sorted.filter((p) => p.state !== "pending") };
}

export function proposalsOfKind(page: RetentionPage, kind: string): Proposal[] | null {
  return page.proposals.rows === null ? null : page.proposals.rows.filter((p) => p.kind === kind);
}

export function viewOfKind(kind: string): string {
  return kind === "policy" ? "policies" : kind === "release" ? "holds" : "deletion";
}

export type AlreadyProposed = { number: string; own: boolean } | null;

function ownProposal(p: Proposal): boolean {
  return p.approve_blocked === "own_proposal" || p.can_withdraw;
}

export function pendingPolicyFor(page: RetentionPage, row: PolicyRow): AlreadyProposed {
  if (row.pending === null) return null;
  const p = (page.proposals.rows ?? []).find((x) => x.number === row.pending && x.kind === "policy");
  return { number: String(row.pending), own: !!p && ownProposal(p) };
}

export function pendingReleaseFor(page: RetentionPage, h: HoldRow): AlreadyProposed {
  if (h.release === null) return null;
  const p = (page.proposals.rows ?? []).find((x) => x.number === h.release && x.kind === "release");
  return { number: String(h.release), own: !!p && ownProposal(p) };
}

export type Action = "approve" | "withdraw" | "propose_policy" | "place_hold" | "propose_release" | "stop_run";
export type Explain = "hidden" | "explain";

export function blockerPolicy(code: string, action: Action, ctx: { pendingRow?: boolean } = {}): Explain {
  if (code === "not_pending") return ctx.pendingRow ? "explain" : "hidden";
  if (code === "not_proposer") return "hidden";
  if (code === "released" && action === "propose_release") return "hidden";
  if ((code === "stopped" || code === "finished") && action === "stop_run") return "hidden";
  return "explain";
}

export function stateSentence(code: string, action: Action, pendingRow = true): string {
  if (code === "not_proposer" && action === "withdraw" && pendingRow) return "not_proposer";
  if (code === "released" && action === "propose_release") return "released";
  if ((code === "stopped" || code === "finished") && action === "stop_run") return code;
  return "";
}

export type Sum = "ok" | "mismatch";

export function storeSum(s: StoreRow): Sum {
  const parts = [s.target, s.deleted, s.failed, s.excluded, s.pending];
  if (parts.some((x) => x === null)) return "mismatch";
  return (s.deleted as number) + (s.failed as number) + (s.excluded as number) + (s.pending as number) === s.target ? "ok" : "mismatch";
}

export function storesBalanced(list: StoreRow[] | null): boolean {
  return list !== null && list.every((s) => storeSum(s) === "ok");
}

export type DecisionLine = { kind: "none" } | { kind: "not_connected" } | { kind: "code"; code: string } | { kind: "unknown" };

export function decisionLine(decision: string, needed: boolean): DecisionLine {
  if (decision === "not_connected") return { kind: "not_connected" };
  if (decision) return { kind: "code", code: decision };
  return needed ? { kind: "unknown" } : { kind: "none" };
}

export function more(shown: number, total: number | null): boolean {
  return total !== null && total > shown;
}

export function artifactOf(page: RetentionPage, id: string): { group: ClassGroup; artifact: ArtifactRow } | undefined {
  for (const group of page.expiry ?? []) {
    const artifact = (group.artifacts ?? []).find((a) => a.id === id);
    if (artifact) return { group, artifact };
  }
  return undefined;
}

export function holdOf(page: RetentionPage, id: string): HoldRow | undefined {
  return (page.holds.rows ?? []).find((h) => h.id === id);
}

export function runOf(page: RetentionPage, n: string): RunRow | undefined {
  return (page.deletion.rows ?? []).find((r) => runNumber(r.run) === n);
}

export function policyOf(page: RetentionPage, cls: string): PolicyRow | undefined {
  return (page.policies ?? []).find((r) => r.class === cls);
}

export function proposalOfNumber(page: RetentionPage, n: string): Proposal | undefined {
  return (page.proposals.rows ?? []).find((p) => p.number !== null && String(p.number) === n);
}

export type Selection = { cls: string; artifact: string; hold: string; run: string; proposal: string };

export function selectionOf(params: URLSearchParams, page: RetentionPage | null, view: string): Selection {
  const out: Selection = { cls: "", artifact: "", hold: "", run: "", proposal: "" };
  if (!page) return out;
  const cls = params.get("class") ?? "";
  if (classes.includes(cls)) {
    if (view === "policies" && (page.policies ?? []).some((r) => r.class === cls)) out.cls = cls;
    if (view === "expiry" && (page.expiry ?? []).some((g) => g.class === cls)) out.cls = cls;
  }
  const artifact = params.get("artifact") ?? "";
  if (view === "expiry" && validId(artifact) && artifactOf(page, artifact)) out.artifact = artifact;
  const hold = params.get("hold") ?? "";
  if (view === "holds" && validId(hold) && holdOf(page, hold)) out.hold = hold;
  const run = params.get("run") ?? "";
  if (view === "deletion" && /^[1-9][0-9]*$/.test(run) && runOf(page, run)) out.run = run;
  const proposal = params.get("proposal") ?? "";
  const kind = view === "policies" ? "policy" : view === "holds" ? "release" : view === "deletion" ? "plan" : "";
  if (kind && /^[1-9][0-9]*$/.test(proposal) && proposalOfNumber(page, proposal)?.kind === kind) out.proposal = proposal;
  return out;
}

export function rowId(prefix: string, id: string): string {
  return "p04-" + prefix + "-" + id.replace(/[^A-Za-z0-9_-]/g, "_");
}

export function sameClasses(a: string[] | null, b: string[]): boolean {
  const x = [...(a ?? [])].sort();
  const y = [...b].sort();
  return x.length === y.length && x.every((v, i) => v === y[i]);
}

export function holdMatch(page: RetentionPage, me: string, target: string, frozen: Record<string, string>): string {
  const owner = frozen.owner ?? "";
  const reason = frozen.reason ?? "";
  const cls = frozen.class ? [frozen.class] : [];
  const found = (page.holds.rows ?? []).find((h) => {
    if (!h.placer || h.placer.id !== me || !h.owner || h.owner.id !== owner || h.reason !== reason) return false;
    if (target.startsWith(artifactPrefix)) return (h.scope.artifacts ?? []).includes(target.slice(artifactPrefix.length));
    return h.scope.workspace === true && sameClasses(h.scope.classes, cls);
  });
  return found?.id ?? "";
}

export type Reaction = "field" | "reason" | "hint" | "conflict" | "denied" | "signin" | "wait" | "retry" | "renew" | "partial" | "reauth" | "failed";

export function react(failure: string, fields: string[]): Reaction {
  switch (failure) {
    case "invalid":
      return fields.length > 0 ? "field" : "reason";
    case "refused":
    case "stage":
      return "hint";
    case "conflict":
      return "conflict";
    case "denied":
      return "denied";
    case "unauthenticated":
      return "signin";
    case "limited":
      return "wait";
    case "busy":
    case "unavailable":
      return "retry";
    case "stale":
      return "renew";
    case "partial":
      return "partial";
    case "reauth":
    case "reauth_cancelled":
      return "reauth";
    default:
      return "failed";
  }
}

export function keepsAttempt(r: Reaction): boolean {
  return r === "retry" || r === "wait" || r === "reauth";
}

export type FieldReaction = "built" | "impact" | "reason" | "period" | "grace" | "owner" | "text";

const ownFields: Record<CommandKey, string[]> = {
  "retention.approve": ["impact_as_of", "impact_count", "impact_digest"],
  "retention.withdraw": [],
  "retention.propose_policy": ["period", "grace"],
  "retention.place_hold": ["owner", "reason"],
  "retention.propose_release": ["reason"],
  "retention.stop_run": [],
};

export function fieldReaction(command: CommandKey, fields: string[]): FieldReaction {
  const own = ownFields[command];
  if (fields.some((f) => f !== "rationale" && !own.includes(f))) return "built";
  if (fields.some((f) => f.startsWith("impact_"))) return "impact";
  if (fields.includes("owner")) return "owner";
  if (fields.includes("period")) return "period";
  if (fields.includes("grace")) return "grace";
  if (fields.includes("reason")) return "text";
  return "reason";
}

export type Pending = { key: string; request: string; expected: string; values: string; snapshot: string[]; context: Record<string, string>; stale: boolean };
const pending = new Map<string, Pending>();
const drafts = new Map<string, Record<string, string>>();
const refusals = new Refusals<string>();

let scope = "";
let scopeGeneration = 0;

export function currentScope(): number { return scopeGeneration; }

export function setScope(workspace: string, user: string, session: string): void {
  const next = workspace + "\u0000" + user + "\u0000" + session;
  if (next !== scope) {
    scopeGeneration++;
    pending.clear();
    drafts.clear();
    refusals.clear();
    scope = next;
  }
}

export function forget(): void {
  scopeGeneration++;
  pending.clear();
  drafts.clear();
  refusals.clear();
  scope = "";
}

export function scopeWorkspace(): string {
  const at = scope.indexOf("\u0000");
  return at < 0 ? "" : scope.slice(0, at);
}

export function approveValuesFor(p: Proposal, kept: Pending | undefined): Record<string, string> {
  if (!kept) return impactValues(p);
  return Object.fromEntries(Object.entries(frozenValues(kept) ?? {}).filter(([key]) => key !== "rationale"));
}

function slot(command: string, target: string): string {
  return scope + "\u0001" + command + "\u0000" + target;
}

function unslot(id: string): { command: CommandKey; target: string } | null {
  if (!id.startsWith(scope + "\u0001")) return null;
  const rest = id.slice(scope.length + 1);
  const at = rest.indexOf("\u0000");
  return { command: rest.slice(0, at) as CommandKey, target: rest.slice(at + 1) };
}

export function pendingNow(command: string, target: string): Pending | undefined {
  return pending.get(slot(command, target));
}

export function pendingFor(
  command: string,
  target: string,
  values: Record<string, string>,
  expected: string,
  fresh: () => { request: string; key: string },
  snapshot: string[] = [],
  context: Record<string, string> = {},
): Pending {
  const id = slot(command, target);
  const frozen = JSON.stringify(values);
  const kept = pending.get(id);
  if (kept && kept.values === frozen && !kept.stale) return kept;
  refusals.delete(id);
  const tried = fresh();
  const next = { key: tried.key, request: tried.request, expected, values: frozen, snapshot: [...snapshot], context: { ...context }, stale: false };
  pending.set(id, next);
  return next;
}

export function frozenValues(p: Pending | undefined): Record<string, string> | null {
  if (!p) return null;
  try {
    const v = JSON.parse(p.values) as unknown;
    return v && typeof v === "object" ? (v as Record<string, string>) : null;
  } catch {
    return null;
  }
}

export function discard(command: string, target: string, values: Record<string, string>): void {
  const kept = pending.get(slot(command, target));
  if (kept && kept.values !== JSON.stringify(values)) pending.delete(slot(command, target));
}

export function drop(command: string, target: string): void {
  pending.delete(slot(command, target));
}

export function markStale(command: string, target: string): void {
  const kept = pending.get(slot(command, target));
  if (kept) kept.stale = true;
}

export function settle(command: string, target: string, reaction: Reaction | "ok", failure = "", diagnostic?: RowDiagnostic): void {
  if (reaction !== "ok" && keepsAttempt(reaction)) return;
  if (reaction === "renew" && command === "retention.place_hold" && pending.has(slot(command, target))) {
    markStale(command, target);
    return;
  }
  pending.delete(slot(command, target));
  if (reaction === "hint") refusals.set(slot(command, target), failure === "refused" ? "refused" : "stage", diagnostic);
  else refusals.delete(slot(command, target));
}

export function refusedNow(command: string, target: string): string {
  return refusals.get(slot(command, target)) ?? "";
}

export function keptKey(command: CommandKey, kind: string): string {
  return command === "retention.place_hold" ? command + ":" + (kind === "refused" ? "refused" : "stage") : command;
}

export function listed(page: RetentionPage, command: CommandKey, target: string): boolean {
  switch (command) {
    case "retention.approve":
    case "retention.withdraw":
      return (page.proposals.rows ?? []).some((p) => p.target === target);
    case "retention.propose_policy":
      return (page.policies ?? []).some((r) => r.target === target);
    case "retention.place_hold":
      return page.holds.tenant_target === target || (page.expiry ?? []).some((g) => (g.artifacts ?? []).some((a) => a.hold_target === target));
    case "retention.propose_release":
      return (page.holds.rows ?? []).some((h) => h.target === target);
    default:
      return (page.deletion.rows ?? []).some((r) => r.run?.target === target);
  }
}

export function stillAllowed(page: RetentionPage, command: CommandKey, target: string): boolean {
  switch (command) {
    case "retention.approve":
    case "retention.withdraw": {
      const p = (page.proposals.rows ?? []).find((x) => x.target === target);
      if (!p || p.state !== "pending" || p.kind === "plan") return false;
      return command === "retention.approve" ? p.can_approve && approveNarrow(p) === "" : p.can_withdraw;
    }
    case "retention.propose_policy":
      return !!(page.policies ?? []).find((r) => r.target === target && r.can_propose && classTarget(r));
    case "retention.place_hold":
      if (target.startsWith(tenantPrefix)) return page.holds.tenant_target === target && tenantTarget(page, scopeWorkspace()) === target;
      return (page.expiry ?? []).some((g) => (g.artifacts ?? []).some((a) => a.hold_target === target && a.holdable && artifactTarget(a)));
    case "retention.propose_release":
      return !!(page.holds.rows ?? []).find((h) => h.target === target && h.can_propose_release && holdTarget(h));
    default:
      return !!(page.deletion.rows ?? []).find((r) => r.run?.target === target && r.can_stop && runTarget(r));
  }
}

export function pruneRefusals(page: RetentionPage): void {
  for (const id of [...refusals.keys()]) {
    const at = unslot(id);
    if (!at) {
      refusals.delete(id);
      continue;
    }
    if (!listed(page, at.command, at.target) || !stillAllowed(page, at.command, at.target)) refusals.delete(id);
  }
}

export type Vanished = { command: CommandKey; target: string; context: Record<string, string> };

export function vanished(page: RetentionPage, view: string): Vanished[] {
  const out: Vanished[] = [];
  for (const [id, p] of pending) {
    const at = unslot(id);
    if (!at || (p.context.view ?? "") !== view || listed(page, at.command, at.target)) continue;
    out.push({ command: at.command, target: at.target, context: p.context });
  }
  return out;
}

export function draftOf(command: string, target: string): Record<string, string> {
  return drafts.get(slot(command, target)) ?? {};
}

export function keepDraft(command: string, target: string, values: Record<string, string>): void {
  drafts.set(slot(command, target), { ...draftOf(command, target), ...values });
}

export function clearDraft(command: string, target: string): void {
  drafts.delete(slot(command, target));
}

export type Reviewed = { count: string; digest: string } | null;

export function reviewedOf(target: string): Reviewed {
  const d = draftOf("retention.approve#reviewed", target);
  return d.count !== undefined && d.digest !== undefined ? { count: d.count, digest: d.digest } : null;
}

export function keepReviewed(target: string, count: string, digest: string): void {
  drafts.set(slot("retention.approve#reviewed", target), { count, digest });
}

export function clearReviewed(target: string): void {
  drafts.delete(slot("retention.approve#reviewed", target));
}

export function reviewedChanged(r: Reviewed, count: string, digest: string): boolean {
  return !!r && (r.count !== count || r.digest !== digest);
}

export function snapshotOf(page: RetentionPage, command: CommandKey): string[] {
  if (command === "retention.propose_policy" || command === "retention.propose_release") return (page.proposals.rows ?? []).flatMap((p) => (p.number === null ? [] : [String(p.number)]));
  return [];
}

export function duplicateIn(snapshot: string[], id: string): boolean {
  return !!id && snapshot.includes(id);
}

export type Denied = "applied_proposal" | "gone";

export function deniedOutcome(command: CommandKey, target: string, hadPending: boolean, page: RetentionPage): Denied {
  if (hadPending && isProposalCommand(command) && !listed(page, command, target)) return "applied_proposal";
  return "gone";
}

export type GoneKind = "proposal" | "hold" | "run" | "item" | "class" | "workspace";

export function goneKind(command: CommandKey, target: string): GoneKind {
  if (isProposalCommand(command)) return "proposal";
  if (command === "retention.propose_release") return "hold";
  if (command === "retention.stop_run") return "run";
  if (command === "retention.propose_policy") return "class";
  return target.startsWith(tenantPrefix) ? "workspace" : "item";
}

export function partialCounts(fields: Record<string, string>): { a: number | null; b: number | null; c: number | null } {
  const n = (v: string | undefined) => (v !== undefined && /^(0|[1-9][0-9]*)$/.test(v) && Number.isSafeInteger(Number(v)) ? Number(v) : null);
  return { a: n(fields.not_preserved), b: n(fields.outcome_pending), c: n(fields.partially_deleted) };
}

export function holdValues(d: Record<string, string>, tenant: boolean): Record<string, string> {
  const out: Record<string, string> = { owner: d.owner ?? "", reason: (d.reason ?? "").trim() };
  if (tenant && d.class && classes.includes(d.class)) out.class = d.class;
  return out;
}

export function refusalDiagnostics(target: string): RowDiagnostic[] {
  return commandKeys.flatMap((command) => {
    const entry = refusals.detail(slot(command, target));
    return entry && entry.command === command && entry.detail.target === target ? [entry] : [];
  });
}

export function withHeldArtifacts(page: RetentionPage, entries: ArtifactEntry[]): RetentionPage {
  if (page.expiry === null || entries.length === 0) return page;
  let changed = false;
  const expiry = page.expiry.map((g) => {
    if (g.artifacts === null) return g;
    const have = new Set(g.artifacts.map((a) => a.id));
    const held = [...g.artifacts];
    for (const e of entries) {
      if (e.class !== g.class || e.artifact.id === "" || have.has(e.artifact.id)) continue;
      have.add(e.artifact.id);
      held.push(e.artifact);
    }
    if (held.length === g.artifacts.length) return g;
    changed = true;
    held.sort((a, b) => (a.id < b.id ? -1 : a.id > b.id ? 1 : 0));
    return { ...g, artifacts: held, more: g.total === null ? g.more : held.length < g.total };
  });
  return changed ? { ...page, expiry } : page;
}

export function withMoreReferences(page: RetentionPage, rows: Reference[]): RetentionPage {
  const refs = page.holds.references;
  if (refs.rows === null || rows.length === 0) return page;
  const have = new Set(refs.rows.map((r) => r.id));
  const held = [...refs.rows];
  for (const r of rows) {
    if (r.id === "" || have.has(r.id)) continue;
    have.add(r.id);
    held.push(r);
  }
  return { ...page, holds: { ...page.holds, references: { ...refs, rows: held, more: refs.total === null ? refs.more : held.length < refs.total } } };
}

export function artifactIdsInUse(): string[] {
  const out = new Set<string>();
  for (const id of [...pending.keys(), ...refusals.keys()]) {
    const at = unslot(id);
    if (at && at.command === "retention.place_hold" && at.target.startsWith(artifactPrefix)) out.add(at.target.slice(artifactPrefix.length));
  }
  return [...out];
}

export function missingArtifacts(page: RetentionPage, ids: string[]): string[] {
  if (page.expiry === null) return [];
  return [...new Set(ids)].filter((id) => validId(id) && !artifactOf(page, id));
}
