import { useState, type FormEvent } from "react";
import type { Locale } from "../../../../shared/types";
import type { AdminClient } from "../../../../service/admin";
import type {
  AccountDetail,
  AccountPage,
  ScopeRef,
  ServiceSelector,
  ServiceRestriction,
} from "../../../../service/service-query-types";
import { reasonProblem } from "../users/Live";
import { w } from "../users/live-copy";
import { c } from "../../../../service/copy";
import { v } from "./live-copy";
import { qcopy as t } from "./query-copy";
import {
  admitsDeclaration,
  declarationFields,
  settingValue,
  type DeclarationDraft,
} from "./command-intents";
import {
  BoundedReason,
  IntentNotice,
  Reauthenticate,
  useIntents,
  usePresent,
} from "./command-ui";
import { Facts, Picker, ReadBoundary, bdi, usePanelHeading } from "./query-ui";
import { useScopedRead } from "./scoped-read";
import type { Recovery } from "./recovery-context";
import { validName } from "./live-model";

function Restrictions({ l, value }: { l: Locale; value: ServiceRestriction }) {
  const selector = (s: ServiceSelector) =>
    "any" in s ? (
      t(l, "all")
    ) : (
      <>
        {t(l, "exactId")}: {bdi(s.id)}
      </>
    );
  const actions: Record<
    ServiceRestriction["action"],
    [string, string, string]
  > = {
    view: ["View", "조회", "عرض"],
    view_cost: ["View cost", "원가 조회", "عرض التكلفة"],
    change_customer_policy: [
      "Change customer policy",
      "고객 정책 변경",
      "تغيير سياسة العميل",
    ],
    change_selling_policy: [
      "Change selling policy",
      "판매 정책 변경",
      "تغيير سياسة البيع",
    ],
    request_approval: ["Request approval", "승인 요청", "طلب الموافقة"],
    issue: ["Issue", "발행", "إصدار"],
    download: ["Download", "다운로드", "تنزيل"],
    deliver: ["Deliver", "전달", "تسليم"],
  };
  return (
    <Facts
      items={[
        [
          t(l, "action"),
          actions[value.action][l === "ko" ? 1 : l === "ar" ? 2 : 0],
        ],
        [t(l, "kind"), bdi(value.kind)],
        [
          t(l, "fields"),
          value.fields.map((field) => <p key={field}>{bdi(field)}</p>),
        ],
        [t(l, "customer"), selector(value.customer)],
        [t(l, "legalEntity"), selector(value.legal_entity)],
        [t(l, "contract"), selector(value.contract)],
        [
          t(l, "period"),
          "any" in value.period ? (
            t(l, "all")
          ) : (
            <>
              {bdi(value.period.from)} / {bdi(value.period.until)}
              <p>{t(l, "exclusiveEnd")}</p>
            </>
          ),
        ],
      ]}
    />
  );
}

export function ScopeDefinition({
  l,
  api,
  recovery,
  workspace,
  refId,
  catalog,
}: {
  l: Locale;
  api: AdminClient;
  recovery: Recovery;
  workspace: string;
  refId: ScopeRef;
  catalog: string;
}) {
  const read = useScopedRead(
    recovery,
    ["scope", workspace, refId.audience, refId.scope].join("\n"),
    () => api.serviceScope(refId.audience, refId.scope, workspace),
  );
  return (
    <ReadBoundary l={l} read={read.result} reload={read.reload}>
      {(detail) => (
        <div className="live-note-rule">
          <p>
            {bdi(refId.audience)} / {bdi(refId.scope)}
          </p>
          {catalog !== detail.catalog_version && <p>{t(l, "currentOnly")}</p>}
          <Facts
            items={[
              [t(l, "resource"), bdi(detail.row.resource)],
              [t(l, "state"), t(l, detail.eligibility)],
              [
                t(l, "restrictions"),
                <details>
                  <summary>{t(l, "restrictions")}</summary>
                  <Restrictions l={l} value={detail.row.restrictions} />
                </details>,
              ],
            ]}
          />
        </div>
      )}
    </ReadBoundary>
  );
}

export function DeclarationForm({
  l,
  api,
  recovery,
  workspace,
  metadata,
  detail,
  refresh,
  onDone,
  onKeep,
}: {
  l: Locale;
  api: AdminClient;
  recovery: Recovery;
  workspace: string;
  metadata: AccountPage | AccountDetail;
  detail?: AccountDetail;
  refresh: () => void;
  onDone: (id: string) => void;
  onKeep: (id: string) => void;
}) {
  useIntents(recovery);
  const present = usePresent();
  const heading = usePanelHeading(detail?.row.id ?? "create");
  const kind = detail ? "declare" : "create";
  const saved = recovery.commands.draft(kind),
    entry = recovery.commands.get(kind);
  const row = detail?.row,
    declaration = row?.declaration;
  const initial: DeclarationDraft = {
    target: row?.id ?? "",
    name: "",
    owner: row?.owner ?? "",
    purpose: declaration?.purpose ?? "",
    organization: declaration?.organization ?? "",
    scopes: declaration?.scopes ?? [],
    catalog: metadata.catalog_version,
    version: declaration?.version || "0",
    reason: "",
  };
  const draft = saved ?? initial;
  const locked = !!entry;
  const [problem, setProblem] = useState(""),
    [busy, setBusy] = useState(false),
    [audience, setAudience] = useState(""),
    [scope, setScope] = useState("");
  const wrongTarget = !!saved && saved.target !== initial.target;
  const edit = (patch: Partial<DeclarationDraft>) => {
    if (locked || busy) return;
    if (
      !recovery.commands.save(
        kind,
        { ...draft, ...patch },
        metadata.limits,
        metadata.settings,
      )
    )
      setProblem(t(l, "bounds"));
    else setProblem("");
  };
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (busy || !recovery.issues.current()) return;
    if (entry) {
      if (entry.phase !== "unknown") return;
    } else {
      const reason = reasonProblem(l, draft.reason);
      if (
        (!detail && !validName(draft.name)) ||
        !draft.owner ||
        !draft.purpose.trim() ||
        !draft.organization ||
        !draft.scopes.length ||
        reason ||
        !admitsDeclaration(draft, metadata.limits, metadata.settings) ||
        draft.catalog !== metadata.catalog_version ||
        (detail &&
          (draft.version !== (detail.row.declaration.version || "0") ||
            !detail.eligibility.can_declare))
      ) {
        setProblem(reason || t(l, "required"));
        return;
      }
      setBusy(true);
      setProblem("");
      try {
        const [owner, organization, ...definitions] = await Promise.all([
          api.serviceOwner(draft.owner, workspace),
          api.serviceOrganization(draft.organization, workspace),
          ...draft.scopes.map((ref) =>
            api.serviceScope(ref.audience, ref.scope, workspace),
          ),
        ]);
        if (!present.current || !recovery.issues.current()) return;
        if (
          !owner.row.eligible_at_reference ||
          organization.eligibility !== "eligible" ||
          definitions.some(
            (d) =>
              d.eligibility !== "eligible" ||
              d.catalog_version !== draft.catalog,
          )
        ) {
          setProblem(t(l, "required"));
          return;
        }
      } catch {
        if (present.current) setProblem(t(l, "unavailable"));
        return;
      } finally {
        if (present.current) setBusy(false);
      }
    }
    const done = await recovery.commands.submit(
      kind,
      entry
        ? undefined
        : {
            command: detail ? "service.declare" : "service.create",
            target: draft.target || "services",
            reason: draft.reason,
            fields: declarationFields(draft),
            revision: metadata.revision,
            replay: metadata.limits.replay_seconds,
          },
      metadata.limits,
      (d, tried) =>
        api.command(
          d.command,
          d.target,
          d.revision,
          d.reason,
          { ...d.fields },
          tried,
        ),
    );
    if (done && present.current && recovery.issues.current()) {
      recovery.commands.discardDraft(kind);
      recovery.commands.discard(kind);
      onDone(draft.target || "svc-" + draft.name);
    }
  };
  if (wrongTarget)
    return (
      <section className="live-step">
        <p>
          {t(l, "otherDraft")} {bdi(saved.target)}
        </p>
        <div className="live-buttons">
          <button
            type="button"
            className="live-secondary"
            onClick={() => onKeep(saved.target)}
          >
            {t(l, "keepDraft")}
          </button>
          <button
            type="button"
            className="live-secondary"
            onClick={() => recovery.commands.discardDraft(kind)}
          >
            {t(l, "discardDraft")}
          </button>
        </div>
      </section>
    );
  if (entry && entry.draft.target !== (draft.target || "services"))
    return <IntentNotice l={l} recovery={recovery} kind={kind} />;
  const purposeLimit = settingValue(
      metadata.settings,
      "service_declaration_purpose_bytes",
    ),
    scopeLimit = settingValue(metadata.settings, "service_declaration_scopes");
  return (
    <form className="live-step" noValidate onSubmit={submit}>
      <h2 tabIndex={-1} ref={heading}>
        {detail ? t(l, "edit") : v.createTitle(l)}
      </h2>
      <p>{t(l, "metadata")}</p>
      <IntentNotice l={l} recovery={recovery} kind={kind} />
      {problem && (
        <p role="alert" className="service-field-error">
          {problem}
        </p>
      )}
      {detail && (
        <Reauthenticate
          l={l}
          state={detail.eligibility.step_up_state}
          refresh={refresh}
        />
      )}
      <fieldset
        disabled={locked || busy || !purposeLimit || !scopeLimit}
        className="live-sections"
      >
        {!detail && (
          <label>
            {v.name(l)}
            <span className="live-unit live-prefixed" dir="ltr">
              <span>svc-</span>
              <input
                value={draft.name}
                onChange={(e) => edit({ name: e.target.value })}
                autoComplete="off"
              />
            </span>
          </label>
        )}
        {detail ? (
          <Facts
            items={[
              [v.colService(l), bdi(detail.row.id)],
              [t(l, "owner"), bdi(detail.row.owner_login || detail.row.owner)],
            ]}
          />
        ) : (
          <Picker
            l={l}
            recovery={recovery}
            name={t(l, "owner")}
            scope={workspace}
            value={draft.owner}
            load={(q) => api.serviceOwners(q)}
            resolve={(id) => api.serviceOwner(id, workspace)}
            label={(r) => r.login || r.id}
            onChange={(owner) => edit({ owner })}
          />
        )}
        <div className="service-field">
          <label htmlFor="svc-purpose">{t(l, "purpose")}</label>
          <textarea
            id="svc-purpose"
            rows={3}
            value={draft.purpose}
            onChange={(e) => edit({ purpose: e.target.value })}
          />
        </div>
        <Picker
          l={l}
          recovery={recovery}
          name={t(l, "organization")}
          scope={workspace}
          value={draft.organization}
          load={(q) => api.serviceOrganizations(q)}
          resolve={(id) => api.serviceOrganization(id, workspace)}
          label={(r) =>
            r.name_state === "available"
              ? r.name
              : r.id + " · " + t(l, "nameUnavailable")
          }
          onChange={(organization) => edit({ organization })}
        />
        <Picker
          l={l}
          recovery={recovery}
          name={t(l, "audience")}
          emptyCatalog
          scope={workspace}
          value={audience}
          load={(q) => api.serviceAudiences(q)}
          resolve={(id) => api.serviceAudience(id, workspace)}
          label={(r) => r.id + " · " + r.resource}
          onChange={(id) => {
            setAudience(id);
            setScope("");
          }}
        />
        {audience && (
          <Picker
            key={audience}
            l={l}
            recovery={recovery}
            name={t(l, "scopes")}
            scope={workspace + "\n" + audience}
            value={scope}
            load={(q) => api.serviceScopes(audience, q)}
            resolve={(id) => api.serviceScope(audience, id, workspace)}
            label={(r) => r.id}
            onChange={setScope}
          />
        )}
        <button
          type="button"
          className="live-secondary"
          disabled={!audience || !scope}
          onClick={() => {
            if (
              !draft.scopes.some(
                (s) => s.audience === audience && s.scope === scope,
              )
            )
              edit({ scopes: [...draft.scopes, { audience, scope }] });
          }}
        >
          {t(l, "add")}
        </button>
        <ul>
          {draft.scopes.map((ref) => (
            <li key={JSON.stringify([ref.audience, ref.scope])}>
              <ScopeDefinition
                l={l}
                api={api}
                recovery={recovery}
                workspace={workspace}
                refId={ref}
                catalog={draft.catalog}
              />
              <button
                type="button"
                className="live-secondary"
                onClick={() =>
                  edit({ scopes: draft.scopes.filter((s) => s !== ref) })
                }
              >
                {t(l, "remove")} {bdi(ref.scope)}
              </button>
            </li>
          ))}
        </ul>
        <Facts
          items={[
            [t(l, "catalog"), bdi(draft.catalog)],
            [t(l, "version"), draft.version],
          ]}
        />
        {draft.catalog !== metadata.catalog_version && (
          <button
            type="button"
            className="live-secondary"
            onClick={() => edit({ catalog: metadata.catalog_version })}
          >
            {t(l, "useCatalog")}
          </button>
        )}
        <BoundedReason
          l={l}
          limit={metadata.limits.rationale_bytes}
          value={draft.reason}
          onChange={(reason) => edit({ reason })}
          error=""
        />
      </fieldset>
      <div className="live-buttons">
        <button
          type="submit"
          className="live-primary"
          disabled={
            busy ||
            (!!entry && entry.phase !== "unknown") ||
            (!entry && !!detail && !detail.eligibility.can_declare)
          }
        >
          {entry?.phase === "unknown"
            ? c.tryAgain(l)
            : busy
              ? w.saving(l)
              : detail
                ? t(l, "edit")
                : v.createTitle(l)}
        </button>
        {saved && !locked && (
          <button
            type="button"
            className="live-secondary"
            onClick={() => recovery.commands.discardDraft(kind)}
          >
            {t(l, "discardDraft")}
          </button>
        )}
      </div>
    </form>
  );
}
