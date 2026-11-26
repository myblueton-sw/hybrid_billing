import { useState } from "react";
import type { Locale } from "../../../../shared/types";
import type { AdminClient } from "../../../../service/admin";
import type { AccountDetail as Detail } from "../../../../service/service-query-types";
import type { Recovery } from "./recovery-context";
import { useScopedRead } from "./scoped-read";
import {
  Facts,
  PageInfo,
  ReadBoundary,
  bdi,
  usePanelHeading,
} from "./query-ui";
import { qcopy as t } from "./query-copy";
import { v } from "./live-copy";
import { ScopeDefinition } from "./DeclarationForm";
import { filterValue } from "./QueryFilters";
import { Reauthenticate } from "./command-ui";
import type { AccountAction } from "./AccountActions";
import { cursorTrail } from "./query-model";

export function DeclarationHistory({
  l,
  api,
  recovery,
  workspace,
  id,
}: {
  l: Locale;
  api: AdminClient;
  recovery: Recovery;
  workspace: string;
  id: string;
}) {
  const [after, setAfter] = useState(""),
    [version, setVersion] = useState(""),
    [applied, setApplied] = useState("");
  const [trail] = useState(cursorTrail);
  const read = useScopedRead(
    recovery,
    ["history", workspace, id, after, applied].join("\n"),
    () =>
      api.serviceDeclarations(id, {
        workspace,
        after,
        limit: 10,
        version: applied,
      }),
  );
  return (
    <section className="live-sections">
      <h3>{t(l, "history")}</h3>
      <label>
        {t(l, "version")}
        <input
          value={version}
          dir="ltr"
          inputMode="numeric"
          onChange={(e) => {
            if (e.target.value.length <= 20) setVersion(e.target.value);
          }}
        />
      </label>
      <button
        className="live-secondary"
        type="button"
        onClick={() => {
          setApplied(version);
          setAfter("");
          trail.clear();
        }}
      >
        {t(l, "search")}
      </button>
      <ReadBoundary
        l={l}
        read={read.result}
        reload={read.reload}
        restart={() => {
          setAfter("");
          trail.clear();
          read.reload();
        }}
      >
        {(page) => (
          <>
            {page.availability !== "recorded" && (
              <p>
                {t(
                  l,
                  page.availability === "not_recorded"
                    ? "missing"
                    : "versionUnavailable",
                )}
              </p>
            )}
            {page.rows.map((row) => (
              <Facts
                key={row.version}
                items={[
                  [t(l, "version"), row.version],
                  [t(l, "purpose"), row.purpose],
                  [t(l, "organization"), bdi(row.organization)],
                  [t(l, "catalog"), bdi(row.catalog_version)],
                  [t(l, "recordedAt"), bdi(row.recorded_at)],
                  [
                    t(l, "scopes"),
                    row.scopes.map((s) => (
                      <p key={JSON.stringify([s.audience, s.scope])}>
                        {bdi(s.audience)} / {bdi(s.scope)}
                      </p>
                    )),
                  ],
                ]}
              />
            ))}
            <PageInfo
              l={l}
              page={page}
              first={() => {
                setAfter("");
                trail.clear();
              }}
              previous={() => setAfter(trail.previous(applied))}
              next={() => {
                if (trail.next(applied, after, page.next)) setAfter(page.next);
              }}
            />
          </>
        )}
      </ReadBoundary>
    </section>
  );
}
export function AccountDetailView({
  l,
  api,
  recovery,
  detail,
  outside,
  refresh,
  onEdit,
  onAction,
  onCredentials,
}: {
  l: Locale;
  api: AdminClient;
  recovery: Recovery;
  detail: Detail;
  outside: boolean;
  refresh: () => void;
  onEdit: () => void;
  onAction: (action: AccountAction) => void;
  onCredentials: () => void;
}) {
  const heading = usePanelHeading(detail.row.id);
  const row = detail.row,
    declaration = row.declaration,
    eligible = detail.eligibility;
  return (
    <section className="live-step">
      <h2 tabIndex={-1} ref={heading}>
        {bdi(row.id)}
      </h2>
      {outside && <p>{t(l, "pinned")}</p>}
      <Facts
        items={[
          [t(l, "state"), filterValue(l, row.state)],
          [t(l, "owner"), bdi(row.owner_login || row.owner)],
          [v.creator(l), bdi(row.creator_login || row.creator)],
          [t(l, "observed"), bdi(detail.observed_at)],
        ]}
      />
      <h3>{t(l, "declaration")}</h3>
      <p>{t(l, "metadata")}</p>
      {declaration.state !== "recorded" ? (
        <p>
          {t(l, declaration.state === "missing" ? "missing" : "unavailable")}
        </p>
      ) : (
        <>
          <Facts
            items={[
              [t(l, "purpose"), declaration.purpose],
              [
                t(l, "organization"),
                declaration.organization_name_state === "available" ? (
                  declaration.organization_name
                ) : (
                  <>
                    {bdi(declaration.organization)} · {t(l, "nameUnavailable")}
                  </>
                ),
              ],
              [t(l, "version"), declaration.version],
              [t(l, "catalog"), bdi(declaration.catalog_version)],
            ]}
          />
          {declaration.scopes.map((ref) => (
            <ScopeDefinition
              key={JSON.stringify([ref.audience, ref.scope])}
              l={l}
              api={api}
              recovery={recovery}
              workspace={detail.workspace}
              refId={ref}
              catalog={declaration.catalog_version}
            />
          ))}
        </>
      )}
      <p>{t(l, "noGrant")}</p>
      <Reauthenticate l={l} state={eligible.step_up_state} refresh={refresh} />
      <div className="live-buttons">
        <button type="button" className="live-secondary" onClick={onEdit}>
          {t(l, "edit")}
        </button>
        <button
          type="button"
          className="live-secondary"
          onClick={onCredentials}
        >
          {v.manageCreds(l)}
        </button>
        {(["suspend", "reinstate", "retire"] as const).map((action) => (
          <button
            key={action}
            type="button"
            className="live-secondary"
            disabled={
              !eligible[
                action === "suspend"
                  ? "can_suspend"
                  : action === "reinstate"
                    ? "can_reinstate"
                    : "can_retire"
              ]
            }
            onClick={() => onAction(action)}
          >
            {action === "suspend"
              ? v.suspendTitle(l)
              : action === "reinstate"
                ? v.reinstateTitle(l)
                : v.retireTitle(l)}
          </button>
        ))}
      </div>
      <DeclarationHistory
        key={row.id}
        l={l}
        api={api}
        recovery={recovery}
        workspace={detail.workspace}
        id={row.id}
      />
    </section>
  );
}
