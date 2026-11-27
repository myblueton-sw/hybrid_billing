import {
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  useSyncExternalStore,
  type ReactNode,
} from "react";
import type { Locale } from "../../../../shared/types";
import { useAdminClient } from "../../../../service/context";
import type { Session } from "../../../../service/api";
import type { AdminClient } from "../../../../service/admin";
import type {
  ServiceAccount,
  ServiceCredential,
  ServicePage,
} from "../../../../service/service-query-types";
import { w } from "../users/live-copy";
import { useWide } from "../users/Live";
import { v } from "./live-copy";
import { qcopy as t } from "./query-copy";
import { useServiceAccountRecovery, type Recovery } from "./recovery-context";
import { readIdentity, useScopedRead } from "./scoped-read";
import {
  accountKeys,
  credentialKeys,
  clearOwned,
  cursorTrail,
  navigation,
  queryKey,
} from "./query-model";
import { Facts, Moment, PagePager, PageSummary, Picker, ReadBoundary, bdi, usePageFocus } from "./query-ui";
import { QueryFilters, filterValue } from "./QueryFilters";
import { DeclarationForm } from "./DeclarationForm";
import { AccountDetailView } from "./AccountDetail";
import { ActionForm, IssueControl, type AccountAction } from "./AccountActions";
import { Reauthenticate, RetainedCommands } from "./command-ui";
import { Reveal } from "./Reveal";
import { CredentialPeriod, CredentialStatus } from "./CredentialStatus";
import { RotateControl } from "./RotateControl";
import { issueExec } from "./issue-send";
import { r } from "./rotate-copy";
import "../users/live.css";
import "../../../../shared/list/list-pattern.css";
import "./service-accounts.css";
export { Reveal } from "./Reveal";

export type LiveProps = {
  locale: Locale;
  view: string;
  params: URLSearchParams;
  onParams: (extra: Record<string, string | null>, push?: boolean) => void;
  session: Session;
  onSignOut: () => void;
  onView: (view: string) => void;
};
export default function LiveServices(props: LiveProps) {
  const recovery = useServiceAccountRecovery();
  if (!recovery) return <p role="status">{v.loading(props.locale)}</p>;
  return <RecoveredServices {...props} recovery={recovery} />;
}
type RevealTarget = {
  estimated: boolean;
  service: string;
  credential: string;
  ref: string;
  until: string;
};
type RevokeTarget = { id: string; service: string; reason: string };

function RecoveredServices({
  locale: l,
  view,
  params,
  onParams,
  session,
  onView,
  recovery,
}: LiveProps & { recovery: Recovery }) {
  const api = useAdminClient(),
    wide = useWide();
  const [generation, setGeneration] = useState(0),
    [reveal, setReveal] = useState<RevealTarget | null>(null),
    [revoke, setRevoke] = useState<RevokeTarget | null>(null);
  const opener = useRef<HTMLElement | null>(null);
  const credentials = view === "credentials";
  const reset = () =>
    onParams(
      clearOwned(params, credentials ? "credentials" : "accounts"),
      true,
    );
  const refresh = () => setGeneration((n) => n + 1);
  const show = (service: string) => {
    const issued = recovery.issues.open(session.user, service);
    if (!issued) return;
    remember();
    setReveal({ ...issued, service, estimated: true });
    void api.serviceCredential(issued.credential, session.workspace).then(
      (detail) => {
        if (!recovery.issues.current() || detail.row.service !== service)
          return;
        setReveal((prior) =>
          prior?.ref === issued.ref &&
          prior.service === service &&
          prior.credential === issued.credential
            ? { ...prior, until: detail.row.not_after, estimated: false }
            : prior,
        );
      },
      () => {},
    );
  };
  if (view !== "accounts" && view !== "credentials")
    return (
      <section className="live-unavailable">
        <h2>{view === "usage" ? v.usageTitle(l) : v.transferTitle(l)}</h2>
        <p>{view === "usage" ? v.usageBody(l) : v.transferBody(l)}</p>
        <button
          type="button"
          className="live-secondary"
          onClick={() => onView("accounts")}
        >
          {v.goAccounts(l)}
        </button>
      </section>
    );
  const remember = () => {
    const active = document.activeElement;
    if (!(active instanceof HTMLElement) || active === document.body || active.closest(".live-panel")) return;
    opener.current = active;
  };
  const back = (completed: boolean, credential = "") => {
    const target = opener.current;
    opener.current = null;
    requestAnimationFrame(() => {
      if (!completed && target?.isConnected && target.offsetParent !== null) return target.focus();
      const row = completed || !credential ? null : [...document.querySelectorAll<HTMLTableRowElement>("section[aria-labelledby='svc-cred-title'] tbody tr")]
        .find((tr) => tr.textContent?.includes(credential));
      const control = row?.querySelector<HTMLElement>("button, a[href]");
      if (control) return control.focus();
      document.getElementById(credentials ? "svc-cred-title" : "svc-list-title")?.focus();
    });
  };
  const onRevoke = (id: string, service: string, reason = "") => {
    remember();
    setReveal(null);
    setRevoke({ id, service, reason });
  };
  const common = {
    l,
    api,
    recovery,
    workspace: session.workspace,
    onParams,
    refresh,
    generation,
    wide,
  };
  return (
    <div
      className="live-users"
      data-live={credentials ? "credentials" : "accounts"}
      data-wide={wide && (reveal || revoke) ? "true" : "false"}
    >
      <div className="live-list live-sections">
        <RetainedCommands l={l} api={api} recovery={recovery} />
        <RecoveryRegion
          l={l}
          api={api}
          recovery={recovery}
          user={session.user}
          workspace={session.workspace}
          generation={generation}
          show={show}
          issue={(id) => {
            onParams({ svc_c_issue: id, view: "credentials" }, true);
          }}
          rotation={(id) => {
            onParams({ svc_c_selected: id, svc_c_rotate: id, view: "credentials" }, true);
          }}
          revoke={onRevoke}
        />
        <div className="live-sections" hidden={!!(reveal || revoke)}>
          {credentials ? (
            <CredentialWorkspace
              key={"credentials"}
              {...common}
              params={params}
              user={session.user}
              show={show}
              revoke={onRevoke}
            />
          ) : (
            <AccountWorkspace
              key={"accounts"}
              {...common}
              params={params}
              reset={reset}
            />
          )}
        </div>
      </div>
      {(reveal || revoke) && (
        <aside className="live-panel">
          {reveal && (
            <Reveal
              key={reveal.ref}
              l={l}
              api={api}
              issues={recovery.issues}
              user={session.user}
              service={reveal.service}
              refId={reveal.ref}
              credential={reveal.credential}
              until={reveal.until}
              untilEstimated={reveal.estimated}
              onDone={() => {
                recovery.issues.unmark(session.user, reveal.credential);
                setReveal(null);
                refresh();
                back(true);
              }}
              onLeave={() => {
                setReveal(null);
                back(false, reveal.credential);
              }}
              onRevoke={(notStored) =>
                onRevoke(
                  reveal.credential,
                  reveal.service,
                  notStored ? v.revokeUnstoredReason(l) : "",
                )
              }
            />
          )}
          {revoke && (
            <KnownRevoke
              key={revoke.id}
              l={l}
              api={api}
              recovery={recovery}
              workspace={session.workspace}
              target={revoke}
              generation={generation}
              refresh={refresh}
              close={() => {
                setRevoke(null);
                back(false, revoke.id);
              }}
              done={() => {
                recovery.issues.revoked(session.user, revoke.id);
                setRevoke(null);
                refresh();
                back(true);
              }}
            />
          )}
        </aside>
      )}
    </div>
  );
}

type WorkspaceProps = {
  l: Locale;
  api: AdminClient;
  recovery: Recovery;
  workspace: string;
  onParams: LiveProps["onParams"];
  refresh: () => void;
  generation: number;
  wide: boolean;
  params: URLSearchParams;
};
function AccountWorkspace({
  l,
  api,
  recovery,
  workspace,
  params,
  onParams,
  refresh,
  generation,
  wide,
  reset,
}: WorkspaceProps & { reset: () => void }) {
  const nav = navigation(params, workspace, "accounts");
  const query = nav.valid ? nav.query : { workspace, limit: 25 };
  const [trail] = useState(cursorTrail);
  const [mode, setMode] = useState<{
    target: string;
    kind: "detail" | "edit" | "action";
    action?: AccountAction;
  } | null>(null);
  const [create, setCreate] = useState(false);
  const [restore, setRestore] = useState<{
    id: string;
    query: string;
    create: boolean;
  } | null>(null);
  const read = useScopedRead(
    recovery,
    readIdentity("accounts", JSON.stringify(query), generation),
    () => api.serviceAccounts(query),
    nav.valid,
  );
  const selected = nav.valid ? nav.selected : "";
  useLayoutEffect(() => {
    setMode(null);
    setCreate(false);
  }, [selected]);
  const detail = useScopedRead(
    recovery,
    readIdentity("account", workspace, selected, generation),
    () => api.serviceAccount(selected, workspace),
    nav.valid && !!selected,
  );
  const key = queryKey(query);
  const focus = usePageFocus(read.result.state, read.result.state === "ready" ? read.result.value : null);
  useLayoutEffect(() => {
    if (!restore || selected || create) return;
    const screen = document.querySelector('[data-security-screen="M04"]');
    const target = restore.create
      ? screen?.querySelector<HTMLButtonElement>("[data-svc-create]")
      : restore.query === key
        ? [
            ...(screen?.querySelectorAll<HTMLButtonElement>("[data-service]") ??
              []),
          ].find((button) => button.dataset.service === restore.id)
        : null;
    (target ?? document.getElementById("svc-list-title"))?.focus();
    setRestore(null);
  }, [restore, selected, create, key]);
  const select = (id: string) => {
    setCreate(false);
    setMode({ target: id, kind: "detail" });
    onParams({ svc_a_selected: id }, true);
  };
  const done = (id: string) => {
    setCreate(false);
    select(id);
    refresh();
  };
  if (!nav.valid)
    return (
      <section>
        <p role="alert">{t(l, "invalidQuery")}</p>
        <button className="live-secondary" type="button" onClick={reset}>
          {t(l, "reset")}
        </button>
      </section>
    );
  let panel: ReactNode = null;
  if (create && read.result.state === "ready")
    panel = (
      <DeclarationForm
        key="create"
        l={l}
        api={api}
        recovery={recovery}
        workspace={workspace}
        metadata={read.result.value}
        refresh={refresh}
        onDone={done}
        onKeep={select}
      />
    );
  else if (selected)
    panel = (
      <ReadBoundary l={l} read={detail.result} reload={detail.reload}>
        {(value) =>
          mode?.target === selected && mode.kind === "edit" ? (
            <DeclarationForm
              key={selected}
              l={l}
              api={api}
              recovery={recovery}
              workspace={workspace}
              metadata={value}
              detail={value}
              refresh={detail.reload}
              onDone={done}
              onKeep={select}
            />
          ) : mode?.target === selected &&
            mode.kind === "action" &&
            mode.action ? (
            <ActionForm
              key={selected + mode.action}
              l={l}
              api={api}
              recovery={recovery}
              detail={value}
              action={mode.action}
              refresh={detail.reload}
              onDone={() => done(selected)}
              onClose={() => setMode({ target: selected, kind: "detail" })}
            />
          ) : (
            <AccountDetailView
              key={selected}
              l={l}
              api={api}
              recovery={recovery}
              detail={value}
              outside={
                read.result.state !== "ready" ||
                !read.result.value.rows.some((row) => row.id === selected)
              }
              refresh={detail.reload}
              onEdit={() => setMode({ target: selected, kind: "edit" })}
              onAction={(action) =>
                setMode({ target: selected, kind: "action", action })
              }
              onCredentials={() =>
                onParams(
                  {
                    svc_c_service: selected,
                    svc_c_issue: selected,
                    svc_c_after: null,
                    view: "credentials",
                  },
                  true,
                )
              }
            />
          )
        }
      </ReadBoundary>
    );
  return (
    <section className="live-sections" aria-labelledby="svc-list-title">
      <div className="live-toolbar">
        <h2 id="svc-list-title" tabIndex={-1}>
          {v.listTitle(l)}
        </h2>
        <div className="live-buttons">
          <button
            className="live-primary"
            type="button"
            data-svc-create
            onClick={() => setCreate(true)}
          >
            {v.createTitle(l)}
          </button>
          <button className="live-secondary" type="button" onClick={refresh}>
            {w.reload(l)}
          </button>
        </div>
      </div>
      <div className="live-users" data-wide={wide && panel ? "true" : "false"}>
        <div className="live-list live-sections" hidden={!!panel && !wide}>
          <div className="svc-list-stack">
          <QueryFilters
            l={l}
            area="accounts"
            query={query}
            onParams={onParams}
          />
          <ReadBoundary
            l={l}
            read={read.result}
            reload={read.reload}
            restart={() => {
              trail.clear();
              onParams({ svc_a_after: null }, true);
              read.reload();
            }}
          >
            {(page) => (
              <div className="svc-list-body">
                <Facts
                  items={[
                    [
                      t(l, "admission"),
                      bdi(
                        page.admission.recorded + " / " + page.admission.limit,
                      ),
                    ],
                  ]}
                />
                <PageSummary l={l} page={page} summaryRef={focus.summary} />
                {page.rows.length > 0 && (
                <AccountTable
                  l={l}
                  rows={page.rows}
                  selected={selected}
                  select={select}
                />)}
                <PagePager
                  l={l}
                  title={v.listTitle(l)}
                  page={page}
                  first={!query.after}
                  onFirst={() => {
                    focus.want.current = "summary";
                    trail.clear();
                    onParams({ svc_a_after: null }, true);
                  }}
                  onPrevious={() => {
                    focus.want.current = "summary";
                    onParams({ svc_a_after: trail.previous(key) || null }, true);
                  }}
                  onNext={() => {
                                        if (trail.next(key, query.after ?? "", page.next)) {
                      focus.want.current = "summary";
 onParams({ svc_a_after: page.next }, true);
 }
                  }}
                  size={{
                    id: "svc-a-size",
                    requested: query.limit ?? page.query.limit,
                    onChange: (n) => {
                      focus.want.current = "svc-a-size";
                      trail.clear();
                      onParams({ svc_a_limit: String(n), svc_a_after: null }, true);
                    },
                  }}
                />
              </div>
            )}
          </ReadBoundary>
          </div>
        </div>
        {panel && (
          <aside className="live-panel">
            <button
              className="live-secondary"
              type="button"
              onClick={() => {
                setRestore({ id: selected, query: key, create });
                setCreate(false);
                onParams({ svc_a_selected: null }, true);
              }}
            >
              {v.close(l)}
            </button>
            {panel}
          </aside>
        )}
      </div>
    </section>
  );
}
function AccountTable({
  l,
  rows,
  selected,
  select,
}: {
  l: Locale;
  rows: ServiceAccount[];
  selected: string;
  select: (id: string) => void;
}) {
  return (
    <div className="security-table-scroll live-table live-access">
      <table>
        <thead>
          <tr>
            <th scope="col">{v.colService(l)}</th>
            <th scope="col">{t(l, "purpose")}</th>
            <th scope="col">{t(l, "owner")}</th>
            <th scope="col">{t(l, "state")}</th>
            <th scope="col">{t(l, "select")}</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr
              key={row.id}
              className={row.id === selected ? "live-selected" : undefined}
            >
              <td className="live-mono">{bdi(row.id)}</td>
              <td>
                {row.declaration.state === "recorded"
                  ? row.declaration.purpose
                  : t(
                      l,
                      row.declaration.state === "missing"
                        ? "missing"
                        : "unavailable",
                    )}
              </td>
              <td>{bdi(row.owner_login || row.owner)}</td>
              <td>{filterValue(l, row.state)}</td>
              <td>
                <button
                  type="button"
                  className="live-secondary"
                  data-service={row.id}
                  aria-pressed={selected === row.id}
                  onClick={() => select(row.id)}
                >
                  {t(l, "select")}
                  <span className="visually-hidden"> {row.id}</span>
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function CredentialWorkspace({
  l,
  api,
  recovery,
  workspace,
  params,
  onParams,
  refresh,
  generation,
  user,
  show,
  revoke,
}: WorkspaceProps & {
  user: string;
  show: (service: string) => void;
  revoke: (id: string, service: string, reason?: string) => void;
}) {
  const nav = navigation(params, workspace, "credentials");
  const query = nav.valid ? nav.query : { workspace, limit: 25 };
  const selected = nav.valid ? nav.selected : "",
    issue = nav.valid ? nav.issue : "",
    rotate = nav.valid ? nav.rotate : "";
  const [trail] = useState(cursorTrail);
  const read = useScopedRead(
    recovery,
    readIdentity("credentials", JSON.stringify(query), generation),
    () => api.serviceCredentials(query),
    nav.valid,
  );
  const known = useScopedRead(
    recovery,
    readIdentity("credential", workspace, selected, generation),
    () => api.serviceCredential(selected, workspace),
    nav.valid && !!selected,
  );
  const account = useScopedRead(
    recovery,
    readIdentity("issue", workspace, issue, generation),
    () => api.serviceAccount(issue, workspace),
    nav.valid && !!issue,
  );
  const key = queryKey(query);
  const focus = usePageFocus(read.result.state, read.result.state === "ready" ? read.result.value : null);
  if (!nav.valid)
    return (
      <section>
        <p role="alert">{t(l, "invalidQuery")}</p>
        <button
          className="live-secondary"
          type="button"
          onClick={() => onParams(clearOwned(params, "credentials"), true)}
        >
          {t(l, "reset")}
        </button>
      </section>
    );
  return (
    <section className="live-sections" aria-labelledby="svc-cred-title">
      <div className="live-toolbar">
        <h2 id="svc-cred-title" tabIndex={-1}>
          {v.credTitle(l)}
        </h2>
        <button type="button" className="live-secondary" onClick={refresh}>
          {w.reload(l)}
        </button>
      </div>
      <p className="live-meta">{t(l, "noGrant")}</p>
      <div className="svc-list-stack">
      <QueryFilters
        l={l}
        area="credentials"
        query={query}
        onParams={onParams}
      />
      <ReadBoundary
        l={l}
        read={read.result}
        reload={read.reload}
        restart={() => {
          trail.clear();
          onParams({ svc_c_after: null }, true);
          read.reload();
        }}
      >
        {(page) => (
          <div className="svc-list-body">
            <PageSummary l={l} page={page} summaryRef={focus.summary} />
            {page.rows.length > 0 && (
            <CredentialTable
              l={l}
              rows={page.rows}
              selected={selected}
              select={(id) => onParams({ svc_c_selected: id, svc_c_rotate: null }, true)}
            />)}
            <PagePager
              l={l}
              title={v.credTitle(l)}
              page={page}
              first={!query.after}
              onFirst={() => {
                focus.want.current = "summary";
                trail.clear();
                onParams({ svc_c_after: null }, true);
              }}
              onPrevious={() => {
                focus.want.current = "summary";
                onParams({ svc_c_after: trail.previous(key) || null }, true);
              }}
              onNext={() => {
                                if (trail.next(key, query.after ?? "", page.next)) {
                  focus.want.current = "summary";
 onParams({ svc_c_after: page.next }, true);
 }
              }}
              size={{
                id: "svc-c-size",
                requested: query.limit ?? page.query.limit,
                onChange: (n) => {
                  focus.want.current = "svc-c-size";
                  trail.clear();
                  onParams({ svc_c_limit: String(n), svc_c_after: null }, true);
                },
              }}
            />
          </div>
        )}
      </ReadBoundary>
      </div>
      {selected && (
        <ReadBoundary l={l} read={known.result} reload={known.reload}>
          {(detail) => (
            <section className="live-step">
              <h3>{bdi(detail.row.id)}</h3>
              {(read.result.state !== "ready" ||
                !read.result.value.rows.some((row) => row.id === selected)) && (
                <p>{t(l, "pinned")}</p>
              )}
              <Facts
                items={[
                  [v.colService(l), bdi(detail.row.service)],
                  [t(l, "state"), <CredentialStatus l={l} row={detail.row} select={(id) => onParams({ svc_c_selected: id, svc_c_rotate: null }, true)} />],
                  [v.colValid(l), <CredentialPeriod l={l} row={detail.row} />],
                  [t(l, "observed"), <Moment l={l} iso={detail.observed_at} />],
                ]}
              />
              <Reauthenticate
                l={l}
                state={detail.eligibility.step_up_state}
                refresh={known.reload}
              />
              <div className="live-buttons">
                <button
                  type="button"
                  className="live-secondary live-rotate"
                  disabled={!detail.eligibility.can_rotate}
                  onClick={() => onParams({ svc_c_rotate: detail.row.id }, true)}
                >
                  {r.rotate(l)}
                </button>
                <button
                  type="button"
                  className="live-secondary"
                  disabled={!detail.eligibility.can_revoke}
                  onClick={() => revoke(detail.row.id, detail.row.service)}
                >
                  {v.revoke(l)}
                </button>
              </div>
              {rotate === detail.row.id && (
                <RotateControl
                  key={rotate}
                  l={l}
                  api={api}
                  recovery={recovery}
                  detail={detail}
                  user={user}
                  refresh={() => {
                    known.reload();
                    refresh();
                  }}
                  show={show}
                  cancel={() => onParams({ svc_c_rotate: null }, true)}
                />
              )}
            </section>
          )}
        </ReadBoundary>
      )}
      <Picker
        l={l}
        recovery={recovery}
        name={t(l, "issueTarget")}
        scope={workspace}
        value={issue}
        load={(q) => api.serviceAccounts(q)}
        resolve={(id) => api.serviceAccount(id, workspace)}
        label={(row) => row.id}
        onChange={(id) => onParams({ svc_c_issue: id || null }, true)}
      />
      {issue ? (
        <ReadBoundary l={l} read={account.result} reload={account.reload}>
          {(detail) => (
            <IssueControl
              key={issue}
              l={l}
              api={api}
              recovery={recovery}
              detail={detail}
              user={user}
              refresh={() => {
                account.reload();
                refresh();
              }}
              show={() => show(issue)}
              onIssue={(target) => onParams({ svc_c_issue: target }, true)}
            />
          )}
        </ReadBoundary>
      ) : (
        <p>{t(l, "noIssueTarget")}</p>
      )}
      <PendingCollection
        l={l}
        api={api}
        recovery={recovery}
        workspace={workspace}
        generation={generation}
        select={(row) =>
          revoke(
            row.id,
            row.service,
            row.secret_pickup === "never_collected" ? v.revokeNeverReason(l) : "",
          )
        }
      />
    </section>
  );
}
function CredentialTable({
  l,
  rows,
  selected,
  select,
}: {
  l: Locale;
  rows: ServiceCredential[];
  selected: string;
  select: (id: string) => void;
}) {
  return (
    <div className="security-table-scroll live-table">
      <table>
        <thead>
          <tr>
            <th scope="col">{v.colCred(l)}</th>
            <th scope="col">{v.colService(l)}</th>
            <th scope="col">{t(l, "state")}</th>
            <th scope="col">{v.colValid(l)}</th>
            <th scope="col">{t(l, "select")}</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr
              key={row.id}
              className={selected === row.id ? "live-selected" : undefined}
            >
              <td>{bdi(row.id)}</td>
              <td>{bdi(row.service)}</td>
              <td>
                <CredentialStatus l={l} row={row} />
                {row.secret_pickup && (
                  <p>
                    {row.secret_pickup === "open" ? (
                      <>
                        {v.pickupOpen(l)} <Moment l={l} iso={row.secret_pickup_until!} />
                      </>
                    ) : row.secret_pickup === "collected" ? (
                      t(l, "collected")
                    ) : (
                      v.pickupNever(l)
                    )}
                  </p>
                )}
              </td>
              <td>
                <CredentialPeriod l={l} row={row} />
              </td>
              <td>
                <button
                  type="button"
                  className="live-secondary"
                  aria-pressed={selected === row.id}
                  onClick={() => select(row.id)}
                >
                  {t(l, "select")}
                  <span className="visually-hidden"> {row.id}</span>
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
function PendingCollection({
  l,
  api,
  recovery,
  workspace,
  generation,
  select,
}: {
  l: Locale;
  api: AdminClient;
  recovery: Recovery;
  workspace: string;
  generation: number;
  select: (row: ServiceCredential) => void;
}) {
  const [after, setAfter] = useState(""),
    [trail] = useState(cursorTrail);
  const read = useScopedRead(
    recovery,
    readIdentity("pending", workspace, after, generation),
    () =>
      api.serviceCredentials({
        workspace,
        collection: "uncollected",
        limit: 25,
        after,
      }),
  );
  const focus = usePageFocus(read.result.state, read.result.state === "ready" ? read.result.value : null);
  return (
    <section className="live-sections">
      <h3>{t(l, "pending")}</h3>
      <p>{t(l, "pendingLead")}</p>
      <ReadBoundary
        l={l}
        read={read.result}
        reload={read.reload}
        restart={() => {
          trail.clear();
          setAfter("");
          read.reload();
        }}
      >
        {(page) => (
          <div className="svc-list-body">
            <PageSummary l={l} page={page} summaryRef={focus.summary} />
            {page.rows.length > 0 && <ul>
              {page.rows.map((row) => (
                <li key={row.id}>
                  {bdi(row.id)} · {bdi(row.service)}{" "}
                  <span>
                    {row.secret_pickup === "open" ? (
                      <>
                        {v.pickupOpen(l)} <Moment l={l} iso={row.secret_pickup_until!} />
                      </>
                    ) : row.secret_pickup === "never_collected" ? (
                      v.pickupNever(l)
                    ) : row.secret_pickup === "collected" ? (
                      t(l, "collected")
                    ) : (
                      t(l, "unavailable")
                    )}
                  </span>{" "}
                  <button
                    className="live-secondary"
                    type="button"
                    onClick={() => select(row)}
                  >
                    {v.revoke(l)}
                  </button>
                </li>
              ))}
            </ul>}
            <PagePager
              l={l}
              title={t(l, "pending")}
              page={page}
              first={!after}
              onFirst={() => {
                focus.want.current = "summary";
                trail.clear();
                setAfter("");
              }}
              onPrevious={() => {
                focus.want.current = "summary";
                setAfter(trail.previous(workspace));
              }}
              onNext={() => {
                focus.want.current = "summary";
                if (trail.next(workspace, after, page.next))
                  setAfter(page.next);
              }}
            />
          </div>
        )}
      </ReadBoundary>
    </section>
  );
}
function KnownRevoke({
  l,
  api,
  recovery,
  workspace,
  target,
  generation,
  refresh,
  close,
  done,
}: {
  l: Locale;
  api: AdminClient;
  recovery: Recovery;
  workspace: string;
  target: RevokeTarget;
  generation: number;
  refresh: () => void;
  close: () => void;
  done: () => void;
}) {
  const read = useScopedRead(
    recovery,
    readIdentity("revoke", workspace, target.id, generation),
    () => api.serviceCredential(target.id, workspace),
  );
  return (
    <ReadBoundary l={l} read={read.result} reload={read.reload}>
      {(detail) =>
        detail.row.service === target.service ? (
          <ActionForm
            l={l}
            api={api}
            recovery={recovery}
            detail={detail}
            action="revoke"
            initial={target.reason}
            refresh={read.reload}
            onDone={done}
            onClose={close}
          />
        ) : (
          <p>{t(l, "unavailable")}</p>
        )
      }
    </ReadBoundary>
  );
}
function RecoveryRegion({
  l,
  api,
  recovery,
  user,
  workspace,
  generation,
  show,
  issue,
  rotation,
  revoke,
}: {
  l: Locale;
  api: AdminClient;
  recovery: Recovery;
  user: string;
  workspace: string;
  generation: number;
  show: (service: string) => void;
  issue: (service: string) => void;
  rotation: (credential: string) => void;
  revoke: (id: string, service: string, reason: string) => void;
}) {
  const { issues } = recovery;
  useSyncExternalStore(issues.subscribe, issues.version);
  const entries = issues.list(user),
    marks = issues.marks(user);
  useEffect(() => {
    const seq = issues.readStarted();
    for (const entry of issues.list(user))
      if (entry.issued)
        void api.serviceCredential(entry.issued.credential, workspace).then(
          (detail) => {
            if (issues.current()) issues.observe(user, detail.row, seq);
          },
          () => {},
        );
  }, [api, issues, user, workspace, generation]);
  if (!entries.length && !marks.length) return null;
  return (
    <section className="live-attempts" aria-labelledby="svc-attempts-title">
      <h3 id="svc-attempts-title" tabIndex={-1}>
        {v.attemptsTitle(l)}
      </h3>
      <p>{v.attemptsDiscardNote(l)}</p>
      <ul>
        {entries.map((entry) => (
          <li key={entry.service}>
            <p>
              {bdi(entry.service)} ·{" "}
              {entry.phase === "unknown"
                ? t(l, "unknown")
                : entry.phase === "issued"
                  ? entry.rotate
                    ? r.rotated(l, entry.issued!.credential, entry.rotate.credential, entry.service)
                    : v.attemptIssued(l, entry.issued!.credential, entry.service)
                  : entry.phase === "sending"
                    ? entry.rotate
                      ? r.sending(l, entry.service)
                      : v.sendingLead(l)
                    : entry.rotate
                      ? r.lapsed(l, entry.service)[0]
                      : v.lapsedTitle(l)}
            </p>
            <div className="live-buttons">
              {entry.phase === "unknown" && (
                <>
                  <button
                    type="button"
                    className="live-secondary"
                    onClick={() => (entry.rotate ? rotation(entry.rotate.credential) : issue(entry.service))}
                  >
                    {entry.rotate ? r.open(l) : v.openIssue(l)}
                  </button>
                  <button
                    type="button"
                    className="live-secondary"
                    onClick={() =>
                      issues.submit(
                        user,
                        entry.service,
                        { days: entry.days, reason: entry.reason, ...(entry.rotate ? { rotate: entry.rotate } : {}) },
                        entry.expected,
                        entry.replay,
                        issueExec(api, entry.service),
                      )
                    }
                  >
                    {t(l, "retryOriginal")}
                  </button>
                </>
              )}
              {entry.phase === "issued" && (
                <>
                  <button
                    type="button"
                    className="live-primary"
                    onClick={() => show(entry.service)}
                  >
                    {v.showSecret(l)}
                  </button>
                  <button
                    type="button"
                    className="live-secondary"
                    onClick={() =>
                      revoke(
                        entry.issued!.credential,
                        entry.service,
                        v.revokeNotShownReason(l),
                      )
                    }
                  >
                    {v.revoke(l)}
                  </button>
                </>
              )}
              {(entry.phase === "unknown" || entry.phase === "lapsed") && (
                <button
                  type="button"
                  className="live-secondary"
                  onClick={() => issues.discard(user, entry.service)}
                >
                  {v.discardAttempt(l)}
                </button>
              )}
            </div>
          </li>
        ))}
      </ul>
      {marks.length > 0 && (
        <>
          <h3>{v.unstoredTitle(l)}</h3>
          <p>{v.unstoredLead(l)}</p>
          <ul>
            {marks.map((mark) => (
              <li key={mark.credential}>
                {bdi(mark.credential)} · {bdi(mark.service)}{" "}
                <button
                  type="button"
                  className="live-secondary"
                  onClick={() =>
                    revoke(
                      mark.credential,
                      mark.service,
                      v.revokeUnstoredReason(l),
                    )
                  }
                >
                  {v.revoke(l)}
                </button>
                <button
                  type="button"
                  className="live-secondary"
                  onClick={() => issues.unmark(user, mark.credential)}
                >
                  {v.dismiss(l)}
                </button>
              </li>
            ))}
          </ul>
        </>
      )}
    </section>
  );
}
