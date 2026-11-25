import { useCallback, useEffect, useRef, useState, type FormEvent, type ReactNode } from "react";
import { diagnosticCopy } from "../../../../service/diagnostic-copy";
import type { Locale } from "../../../../shared/types";
import { AdminError, attempt, pickupGone, sequence, type AdminClient, type Attempt, type ClockState, type CommandId, type SettingRow, type UserRow, type UsersPage } from "../../../../service/admin";
import { useAdminClient } from "../../../../service/context";
import type { Session } from "../../../../service/api";
import { c } from "../../../../service/copy";
import { emailValid, loginValid, utf8Length } from "../../../../service/helpers";
import { Summary, TextField, type Problems } from "../../../../service/Service";
import { actionDone, actionLabel, impact, w, type Action } from "./live-copy";
import { accountName, failureText, offers, roleName, setup, tags } from "./live-model";
import { b } from "./bulk-copy";
import { mayBeAdministrator, number, selectLoaded, toggle, without, type Selection } from "./bulk-model";
import { batches, bulkClient } from "../../../../service/identity-bulk";
import { FixedBar, MatchAction, plainFixedLine } from "./Matching";
import { m } from "./match-copy";
import { fixedOfJob, removalOf, type Fixed, type RemovalSum } from "./match-model";
import { dateParts } from "./period-model";
import { BulkFlow } from "./BulkPrepare";
import RevocationJobs from "./RevocationJobs";
import { endError, endFields, initialEnd, PeriodCell, PeriodFact, PeriodForm, RoleEnds, type EndValue } from "./RolePeriod";
import { CheckCell, HeaderCheck, SelectionBar } from "./Selection";
import { p } from "./period-copy";
import { setting } from "./period-model";
import "./live.css";
import "./bulk.css";
import { At, fill, Stamp } from "./Stamp";
import { bare, fc } from "./filter-copy";
import { appliedKeys, filterViews, linkOf, selectableIds, loginParts, lostPerson, membershipEnded, merge, noFilters, parseLink, queryOf, sameInstant, without as dropFilter, type FilterKey, type UserFilters } from "./filter-model";
import { FilterBand, PagingArea, ScopeLine, type Paging } from "./filters";

type Mode =
  | { kind: "none" }
  | { kind: "detail" }
  | { kind: "confirm"; action: Action }
  | { kind: "create" }
  | { kind: "link"; login: string; purpose: "set" | "reset"; ref: string }
  | { kind: "period" };

type Loaded = Omit<UsersPage, "clock"> & { at: Date; clock: ClockState; first: string };

type Context = { clock: ClockState; settings: SettingRow[]; own: string };

let openJobNext = "";
let kept: { user: string; selection: Selection } = { user: "", selection: new Set() };
let keptFixed: { user: string; fixed: Fixed | null } = { user: "", fixed: null };

export function useWide(): boolean {
  const query = "(min-width: 1280px)";
  const [wide, setWide] = useState(() => typeof matchMedia === "function" && matchMedia(query).matches);
  useEffect(() => {
    if (typeof matchMedia !== "function") return;
    const m = matchMedia(query);
    const update = () => setWide(m.matches);
    m.addEventListener("change", update);
    return () => m.removeEventListener("change", update);
  }, []);
  return wide;
}

const commandOf = (a: Action): CommandId => ("identity." + a) as CommandId;

export function Reason({ l, value, onChange, error }: { l: Locale; value: string; onChange: (v: string) => void; error: string }) {
  const n = utf8Length(value);
  return (
    <div className="service-field live-reason">
      <label htmlFor="live-reason">{w.reason(l)}</label>
      <p className="service-hint" id="live-reason-hint">{w.reasonHint(l)}</p>
      {error && (
        <p className="service-field-error" id="live-reason-error">
          <span className="visually-hidden">{c.errorPrefix(l)} </span>
          {error}
        </p>
      )}
      <textarea
        id="live-reason" rows={3} value={value} aria-invalid={error ? true : undefined}
        aria-describedby={["live-reason-hint", error ? "live-reason-error" : ""].filter(Boolean).join(" ")}
        onChange={(e) => onChange(e.target.value)}
      />
      <p className="live-counter" aria-live={n > 230 ? "polite" : "off"}>{w.bytes(l, n)}</p>
    </div>
  );
}

function RoleChoice({ l, value, onChange, allowAdmin }: { l: Locale; value: string; onChange: (v: string) => void; allowAdmin: boolean }) {
  return (
    <fieldset className="live-roles">
      <legend>{w.workspaceRole(l)}</legend>
      {(allowAdmin ? ["member", "administrator"] : ["member"]).map((r) => (
        <label key={r}>
          <input type="radio" name="live-role" value={r} checked={value === r} onChange={() => onChange(r)} />
          {roleName(l, r)}
        </label>
      ))}
    </fieldset>
  );
}

export function reasonProblem(l: Locale, reason: string): string {
  if (!reason.trim()) return w.enterReason(l);
  return utf8Length(reason) > 256 ? w.longReason(l) : "";
}

export function FailureBox({ l, error, focusKey, describe = failureText }: {
  l: Locale; error: unknown; focusKey: number; describe?: (l: Locale, e: unknown) => { title: string; body: string };
}) {
  const box = useRef<HTMLDivElement>(null);
  useEffect(() => { if (focusKey) box.current?.focus(); }, [focusKey]);
  const shown = describe(l, error);
  return (
    <div className="service-summary" role="alert" tabIndex={-1} ref={box}>
      <h2>{shown.title}</h2>
      {shown.body && <p>{shown.body}</p>}
    </div>
  );
}

export function settles(e: unknown): boolean {
  return !(e instanceof AdminError) || !["unavailable", "limited", "unauthenticated", "reauth", "reauth_cancelled"].includes(e.failure);
}

function Confirm({ l, user, action, api, revision, ctx, onDone, onCancel, onReload }: {
  l: Locale; user: UserRow; action: Action; api: AdminClient; revision: string; ctx: Context;
  onDone: (user: string, action: Action, ref: string) => void; onCancel: () => void; onReload: () => void;
}) {
  const ends = action === "change_role" || action === "invite";
  const [end, setEnd] = useState<EndValue>(() => initialEnd(action === "change_role" ? user.membership?.role_until ?? "" : "", ctx.own));
  const [endProblemText, setEndProblem] = useState("");
  const current = user.membership?.role ?? "";
  const allowAdmin = action === "invite" || !user.link_purpose;
  const [role, setRole] = useState(action === "change_role" && current === "member" && allowAdmin ? "administrator" : "member");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [problem, setProblem] = useState("");
  const [focusKey, setFocusKey] = useState(0);
  const tried = useRef<Attempt>(attempt());
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => heading.current?.focus(), []);
  useEffect(() => { tried.current = attempt(); }, [revision]);
  const edit = (f: (v: string) => void) => (v: string) => { tried.current = attempt(); f(v); };
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    const p = reasonProblem(l, reason);
    const ep = ends ? endError(l, end, ctx.clock, ctx.settings, ctx.own) : "";
    setProblem(p);
    setEndProblem(ep);
    if (p || ep) { setError(null); setFocusKey((k) => k + 1); return; }
    setBusy(true);
    setError(null);
    try {
      const body: Record<string, string> = ends ? { role, ...endFields(end).values, ...(action === "change_role" ? { end: end.kind } : {}) } : {};
      const out = await api.command(commandOf(action), user.user_id, revision, reason, body, tried.current);
      onDone(user.user_id, action, out.fields.link_ref ?? "");
    } catch (err) {
      setError(err);
      setFocusKey((k) => k + 1);
      if (err instanceof AdminError && err.failure === "stale") tried.current = attempt();
      if (settles(err)) onReload();
    } finally {
      setBusy(false);
    }
  };
  return (
    <form className="live-step" onSubmit={submit} noValidate>
      <h2 tabIndex={-1} ref={heading}>{actionLabel[action](l)}</h2>
      <p className="live-who"><bdi dir="ltr">{user.login || user.user_id}</bdi></p>
      {(problem || endProblemText) && <Summary locale={l} title={c.check(l)} problems={[endProblemText && { field: "m01-role-end", message: endProblemText }, problem && { field: "live-reason", message: problem }].filter(Boolean) as Problems} focusKey={focusKey} />}
      {error !== null && <FailureBox l={l} error={error} focusKey={focusKey} />}
      <p className="live-impact">{impact(l, action, user.login || user.user_id, roleName(l, current), roleName(l, role))}</p>
      {ends && (
        <>
          <RoleChoice l={l} value={role} onChange={edit(setRole)} allowAdmin={allowAdmin} />
          <RoleEnds l={l} id="m01-role-end" value={end} onChange={(v) => { tried.current = attempt(); setEnd(v); }} own={ctx.own} clock={ctx.clock} settings={ctx.settings} error={endProblemText} />
        </>
      )}
      <Reason l={l} value={reason} onChange={edit(setReason)} error={problem} />
      <div className="live-buttons">
        <button type="submit" className="live-primary" aria-busy={busy || undefined} disabled={busy}>{busy ? w.saving(l) : actionLabel[action](l)}</button>
        <button type="button" className="live-secondary" onClick={onCancel}>{w.cancel(l)}</button>
      </div>
    </form>
  );
}

function Create({ l, api, revision, ctx, onDone, onCancel, onReload }: {
  l: Locale; api: AdminClient; revision: string; ctx: Context;
  onDone: (user: string, login: string, ref: string) => void; onCancel: () => void; onReload: () => void;
}) {
  const [end, setEnd] = useState<EndValue>(() => initialEnd("", ctx.own));
  const [login, setLogin] = useState("");
  const [email, setEmail] = useState("");
  const [role, setRole] = useState("member");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [problems, setProblems] = useState<Problems>([]);
  const [error, setError] = useState<unknown>(null);
  const [focusKey, setFocusKey] = useState(0);
  const tried = useRef<Attempt>(attempt());
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => heading.current?.focus(), []);
  useEffect(() => { tried.current = attempt(); }, [revision]);
  const edit = (f: (v: string) => void) => (v: string) => { tried.current = attempt(); f(v); };
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    const found: Problems = [];
    if (!login.trim()) found.push({ field: "live-login", message: c.enterSignInEmail(l) });
    else if (!loginValid(login) || !emailValid(login) || login.trim() !== login) found.push({ field: "live-login", message: c.badSignInEmail(l) });
    if (!email.trim()) found.push({ field: "live-email", message: c.enterContact(l) });
    else if (!emailValid(email) || email.trim() !== email) found.push({ field: "live-email", message: c.badContact(l) });
    const ep = endError(l, end, ctx.clock, ctx.settings, ctx.own);
    if (ep) found.push({ field: "m01-create-end", message: ep });
    const r = reasonProblem(l, reason);
    if (r) found.push({ field: "live-reason", message: r });
    setProblems(found);
    setError(null);
    if (found.length) { setFocusKey((k) => k + 1); return; }
    setBusy(true);
    try {
      const out = await api.command("identity.create_user", "users", revision, reason, { login, email, role, ...endFields(end).values }, tried.current);
      onDone(out.fields.user_id ?? "", login.toLowerCase(), out.fields.link_ref ?? "");
    } catch (err) {
      if (err instanceof AdminError && err.failure === "refused") setProblems([{ field: "live-login", message: w.loginRefused(l) }]);
      else if (err instanceof AdminError && err.failure === "invalid" && err.fields.length) {
        setProblems(err.fields.map((f) => f === "role_until" ? { field: "m01-create-end", message: p.capRefused(l) }
          : { field: f === "email" ? "live-email" : "live-login", message: f === "email" ? c.badContact(l) : c.badSignInEmail(l) }));
      } else setError(err);
      if (err instanceof AdminError && err.failure === "stale") tried.current = attempt();
      if (settles(err)) onReload();
      setFocusKey((k) => k + 1);
    } finally {
      setBusy(false);
    }
  };
  const field = (id: string) => problems.find((p) => p.field === id)?.message;
  return (
    <form className="live-step" onSubmit={submit} noValidate>
      <h2 tabIndex={-1} ref={heading}>{w.create(l)}</h2>
      <p className="live-impact">{w.createLead(l)}</p>
      {problems.length > 0 && <Summary locale={l} title={c.check(l)} problems={problems} focusKey={focusKey} />}
      {error !== null && <FailureBox l={l} error={error} focusKey={focusKey} />}
      <TextField id="live-login" label={c.signInEmail(l)} value={login} onChange={edit(setLogin)} type="email" autoComplete="off" error={field("live-login")} locale={l} />
      <TextField id="live-email" label={c.contactEmail(l)} value={email} onChange={edit(setEmail)} type="email" autoComplete="off" error={field("live-email")} locale={l} />
      <RoleChoice l={l} value={role} onChange={edit(setRole)} allowAdmin />
      <RoleEnds l={l} id="m01-create-end" value={end} onChange={(v) => { tried.current = attempt(); setEnd(v); }} own={ctx.own} clock={ctx.clock} settings={ctx.settings} error={field("m01-create-end") ?? ""} />
      <Reason l={l} value={reason} onChange={edit(setReason)} error={field("live-reason") ?? ""} />
      <div className="live-buttons">
        <button type="submit" className="live-primary" aria-busy={busy || undefined} disabled={busy}>{busy ? w.creating(l) : w.create(l)}</button>
        <button type="button" className="live-secondary" onClick={onCancel}>{w.cancel(l)}</button>
      </div>
    </form>
  );
}

function Pickup({ l, api, login, purpose, refId, expires, notice, onDone }: {
  l: Locale; api: AdminClient; login: string; purpose: "set" | "reset"; refId: string; expires: string; notice: string; onDone: () => void;
}) {
  const [url, setUrl] = useState("");
  const [state, setState] = useState<"loading" | "shown" | "retry" | "gone">("loading");
  const [copy, setCopy] = useState<"" | "done" | "failed">("");
  const asked = useRef(false);
  const heading = useRef<HTMLHeadingElement>(null);
  const fetchLink = useCallback(() => {
    setState("loading");
    api.collect(refId).then(
      (u) => { setUrl(u); setState("shown"); },
      (e) => setState(pickupGone(e) ? "gone" : "retry"),
    );
  }, [api, refId]);
  useEffect(() => {
    if (asked.current) return;
    asked.current = true;
    if (refId) fetchLink();
    else setState("gone");
  }, [fetchLink, refId]);
  useEffect(() => heading.current?.focus(), []);
  const doCopy = async () => {
    try {
      await navigator.clipboard.writeText(url);
      setCopy("done");
    } catch {
      setCopy("failed");
    }
  };
  return (
    <section className="live-step" aria-labelledby="live-pickup-title">
      {notice && <p className="service-status">{notice}</p>}
      <h2 id="live-pickup-title" tabIndex={-1} ref={heading}>{purpose === "set" ? w.copySet(l) : w.copyReset(l)}</h2>
      <p className="live-who">{w.forUser(l, login)}</p>
      {state === "loading" && <p role="status">{c.loading(l)}</p>}
      {state === "shown" && (
        <>
          <div className="service-field">
            <label htmlFor="live-link">{w.oneTime(l)}</label>
            <input id="live-link" className="live-link" readOnly dir="ltr" value={url} onFocus={(e) => e.currentTarget.select()} />
          </div>
          <div className="live-buttons">
            <button type="button" className="live-primary" onClick={doCopy}>{w.copy(l)}</button>
          </div>
          <p role="status" className="live-copy-state">{copy === "done" ? w.copied(l) : copy === "failed" ? w.copyFailed(l) : ""}</p>
          {expires && <p><At l={l} template={w.linkExpires(l, "{at}")} iso={expires} zone={false} /></p>}
          <p className="live-impact">{w.shownOnce(l)}</p>
          <p className="live-impact">{w.handOver(l, login)}</p>
        </>
      )}
      {state === "retry" && (
        <div className="service-summary" role="alert">
          <p>{w.unavailable(l)}</p>
          <button type="button" className="live-secondary" onClick={fetchLink}>{w.showLink(l)}</button>
        </div>
      )}
      {state === "gone" && <p className="live-impact">{w.linkGone(l)}</p>}
      <div className="live-buttons">
        <button type="button" className="live-secondary" onClick={() => { setUrl(""); onDone(); }}>{w.done(l)}</button>
      </div>
    </section>
  );
}

function Detail({ l, user, self, notice, onAction, serverTime, onJob }: {
  l: Locale; user: UserRow; self: string; notice: string; onAction: (a: Action) => void; serverTime: string; onJob: (id: string) => void;
}) {
  const offer = offers(l, user, self);
  const group = (title: string, actions: Action[], notes: string[] = []) =>
    (actions.length > 0 || notes.length > 0) && (
      <section className="live-group">
        <h3>{title}</h3>
        <div className="live-group-actions">
          {actions.map((a) => (
            <button key={a} type="button" className="live-secondary" data-action={a} onClick={() => onAction(a)}>{actionLabel[a](l)}</button>
          ))}
        </div>
        {notes.map((n) => <p key={n} className="live-note">{n}</p>)}
      </section>
    );
  return (
    <section className="live-detail-body" aria-labelledby="live-detail-title">
      {notice && <div className="service-status">{notice.split("\n").map((line, i) => <p key={i} className={i === 0 && notice.includes("\n") ? "live-strong" : undefined}>{line}</p>)}</div>}
      <h2 id="live-detail-title" tabIndex={-1}><bdi dir="ltr">{user.login || user.user_id}</bdi></h2>
      <dl className="live-facts">
        <div><dt>{c.contactEmail(l)}</dt><dd><bdi dir="ltr">{user.email || "—"}</bdi></dd></div>
        <div><dt>{w.accountStatus(l)}</dt><dd className={user.state === "active" ? "" : "live-strong"}>{accountName(l, user.state)}</dd></div>
        <div><dt>{w.workspaceRole(l)}</dt><dd>{roleName(l, user.membership?.role ?? "")}</dd></div>
        <PeriodFact l={l} u={user} serverTime={serverTime} onJob={onJob} />
        <div><dt>{w.membershipStatus(l)}</dt><dd>{membershipWord(l, user)}</dd></div>
        <div><dt>{w.signInSetup(l)}</dt><dd><SetupText l={l} u={user} /></dd></div>
        {user.installation_administrator && <div><dt>{w.installationAdmin(l)}</dt><dd>{w.yes(l)}</dd></div>}
        <div><dt>{w.userId(l)}</dt><dd className="live-mono"><bdi dir="ltr">{user.user_id}</bdi></dd></div>
      </dl>
      {offer.blocked ? (
        <p className="live-note live-blocked">{offer.blocked}</p>
      ) : (
        <>
          {group(w.account(l), offer.account)}
          {group(w.membershipTitle(l), offer.membership, offer.notes.filter((n) => n === w.promotionReason(l) || n === w.unknownStateReason(l) || n === w.unknownRoleReason(l)))}
          {group(w.sign(l), offer.sign, offer.notes.filter((n) => n !== w.promotionReason(l) && n !== w.unknownStateReason(l) && n !== w.unknownRoleReason(l)))}
        </>
      )}
    </section>
  );
}

function SetupText({ l, u }: { l: Locale; u: UserRow }) {
  if (u.link_purpose === "set") return <At l={l} template={fc.setLinkAt(l)} iso={u.link_expires} zone={false} />;
  if (u.link_purpose === "reset") return <At l={l} template={fc.resetLinkAt(l)} iso={u.link_expires} zone={false} />;
  if (u.invitation_expires) return <At l={l} template={w.invitationPending(l, "{at}")} iso={u.invitation_expires} zone={false} />;
  return <>{setup(l, u)}</>;
}

function Breakable({ text }: { text: string }) {
  return <>{loginParts(text).map((part, i) => <span key={i}>{i > 0 && <wbr />}{part}</span>)}</>;
}

function membershipWord(l: Locale, u: UserRow): string {
  if (!u.membership) return w.noMembership(l);
  if (membershipEnded(u)) return fc.membershipEnded(l);
  return u.membership.state === "active" ? w.active(l) : u.membership.state === "suspended" ? w.membershipSuspended(l) : w.unknownState(l);
}

function RoleBeside({ l, u }: { l: Locale; u: UserRow }) {
  if (!u.membership) return <><span aria-hidden="true">—</span><span className="visually-hidden">{w.noRole(l)}</span></>;
  return <>{roleName(l, u.membership.role)}</>;
}

function Cell({ label, className, children }: { label: string; className?: string; children: ReactNode }) {
  return <td role="cell" data-label={label} className={className}>{children}</td>;
}

type Select = { on: boolean; keeping: boolean; selected: Selection; self: string; atLimit: boolean; loaded: string[]; onToggle: (id: string) => void; onAll: () => void };

function Effect({ l, u }: { l: Locale; u: UserRow }) {
  if (!u.revocation || u.membership?.suspended_for !== "bulk_revocation" && u.state !== "suspended") return null;
  return (
    <span className="live-sub m01-effect">
      {b.suspendedBy(l, u.revocation.bulk.replace("bulk-revocation-", ""))}
      {u.revocation.state === "decision_unconfirmed" && <span className="live-strong"> · {p.unconfirmed(l)}</span>}
    </span>
  );
}

export function Table({ l, view, users, selected, onOpen, sel, serverTime }: {
  l: Locale; view: string; users: UserRow[]; selected: string; onOpen: (id: string) => void; sel: Select; serverTime: string;
}) {
  const check = sel.on && view !== "invitations";
  const head = view === "invitations"
    ? [w.user(l), w.role(l), w.invitationExpires(l), w.setLink(l)]
    : view === "membership"
      ? [w.user(l), w.workspaceRole(l), p.rolePeriod(l), w.membershipStatus(l), w.installationAdmin(l), p.roleAndPeriod(l)]
      : [w.user(l), w.accountStatus(l), w.workspaceRole(l), p.rolePeriod(l), w.signInSetup(l), p.roleAndPeriod(l)];
  const folded = (i: number) => (view !== "invitations" && (view === "membership" ? i === 1 || i === 2 || i === 4 : i === 2 || i === 3 || i === 4) ? "m01-l" : view !== "invitations" && i === 5 ? "m01-m" : "");
  return (
    <div className="security-table-scroll live-table" data-view={view} data-check={check ? "true" : "false"}>
      <table role="table">
        <colgroup>{check && <col className="m01-col-check" />}<col className="live-col-user" />{head.slice(1).map((h, i) => <col key={h} className={folded(i + 1)} />)}</colgroup>
        <thead role="rowgroup"><tr role="row">
          {check && <HeaderCheck l={l} selected={sel.selected} loaded={sel.loaded} onAll={sel.onAll} cell keeping={sel.keeping} />}
          {head.map((h, i) => <th key={h} scope="col" role="columnheader" className={folded(i)}>{h}</th>)}
        </tr></thead>
        <tbody role="rowgroup">
          {users.map((u) => (
            <tr key={u.user_id} role="row" className={u.user_id === selected ? "live-selected" : undefined}>
              {check && <CheckCell l={l} u={u} self={sel.self} selected={sel.selected} atLimit={sel.atLimit} onToggle={sel.onToggle} keeping={sel.keeping} />}
              <td role="cell" className="live-user">
                <button type="button" className="live-open" data-user-id={u.user_id} aria-current={u.user_id === selected ? "true" : undefined} onClick={() => onOpen(u.user_id)}>
                  <bdi dir="ltr" className={u.login ? undefined : "live-mono"}>{u.login ? <Breakable text={u.login} /> : u.user_id}</bdi>
                  <span className="visually-hidden">, {w.openDetails(l)}{u.user_id === selected ? " " + w.selected(l) : ""}</span>
                </button>
                {u.user_id === selected && <span className="m01-tag" aria-hidden="true">{b.open(l)}</span>}
                {u.email && u.email !== u.login && <span className="live-sub"><bdi dir="ltr">{u.email}</bdi></span>}
                {u.state === "departed" && <span className="live-sub m01-departed">{fc.departedRow(l)}</span>}
                {view === "users" && <span className="live-sub m01-fold live-quiet"><SetupText l={l} u={u} /></span>}
              </td>
              {view === "invitations" ? (
                <>
                  <Cell label={head[1]}>{roleName(l, u.membership?.role ?? "")}</Cell>
                  <Cell label={head[2]}><Stamp l={l} iso={u.invitation_expires} zone={false} /></Cell>
                  <Cell label={head[3]} className="live-quiet">{u.link_purpose !== "set" ? w.notIssued(l) : sameInstant(u.link_expires, u.invitation_expires) ? fc.linkWithInvitation(l)
                    : <At l={l} template={fc.linkIssued(l)} iso={u.link_expires} zone={false} />}</Cell>
                </>
              ) : view === "membership" ? (
                <>
                  <Cell label={head[1]} className="m01-l"><RoleBeside l={l} u={u} /></Cell>
                  <Cell label={head[2]} className="m01-l"><PeriodCell l={l} u={u} serverTime={serverTime} /></Cell>
                  <Cell label={head[3]}>{membershipWord(l, u)}<Effect l={l} u={u} /></Cell>
                  <Cell label={head[4]} className="m01-l">{u.installation_administrator ? w.yes(l) : <><span aria-hidden="true">—</span><span className="visually-hidden">{w.no(l)}</span></>}</Cell>
                  <Cell label={head[5]} className="m01-m">
                    <RoleBeside l={l} u={u} />{u.membership && <span className="live-sub"><PeriodCell l={l} u={u} serverTime={serverTime} /></span>}
                    {u.installation_administrator && <span className="live-sub">{w.installationAdmin(l)}</span>}
                  </Cell>
                </>
              ) : (
                <>
                  <Cell label={head[1]} className={u.state === "active" ? "" : "live-strong"}>{accountName(l, u.state)}{u.state === "suspended" && u.membership?.suspended_for !== "bulk_revocation" && <Effect l={l} u={u} />}</Cell>
                  <Cell label={head[2]} className="m01-l">
                    {roleName(l, u.membership?.role ?? "")}
                    {u.membership?.state === "suspended" && <span className="live-sub">{u.membership.suspended_for === "role_period_ended" ? p.ended(l) : w.membershipSuspended(l)}</span>}
                    {u.membership?.state === "unknown" && <span className="live-sub">{w.unknownState(l)}</span>}
                    {u.membership?.suspended_for === "bulk_revocation" && <Effect l={l} u={u} />}
                    {u.installation_administrator && <span className="live-sub">{w.installationAdmin(l)}</span>}
                  </Cell>
                  <Cell label={head[3]} className="m01-l"><PeriodCell l={l} u={u} serverTime={serverTime} /></Cell>
                  <Cell label={head[4]} className="live-quiet m01-l"><SetupText l={l} u={u} /></Cell>
                  <Cell label={head[5]} className="m01-m">
                    {roleName(l, u.membership?.role ?? "")}{u.membership && <span className="live-sub"><PeriodCell l={l} u={u} serverTime={serverTime} /></span>}
                    {u.membership?.state === "suspended" && <span className="live-sub">{w.membershipSuspended(l)}</span>}
                    {u.membership?.state === "unknown" && <span className="live-sub">{w.unknownState(l)}</span>}
                    {u.membership?.suspended_for === "bulk_revocation" && <Effect l={l} u={u} />}
                    {u.installation_administrator && <span className="live-sub">{w.installationAdmin(l)}</span>}
                  </Cell>
                </>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function Conditions({ l, data }: { l: Locale; data: Loaded }) {
  const cl = data.clock;
  const down = ["isolated", "stopped", "full"].includes(data.ledger_state);
  const baseNotRecorded = cl.clock_base.choice !== "default" && !cl.clock_base.recorded;
  if (!cl.wall_lead && !down && !baseNotRecorded && !cl.clock_behind && !cl.clock_ahead && !cl.recovery.stranded) return null;
  return (
    <section className="m01-conditions" aria-labelledby="m01-conditions-h">
      <h2 id="m01-conditions-h" className="visually-hidden">{b.notices(l)}</h2>
      {cl.recovery.stranded && <div className="m01-condition"><p><strong>{b.recovery(l)}</strong></p></div>}
      {cl.wall_lead && (
        <div className="m01-condition">
          {b.wallLead(l).map((line, i) => <p key={i}>{i === 0 ? <strong>{line}</strong> : line}</p>)}
          <dl className="m01-inline"><div><dt>{b.hostClock(l)}</dt><dd><bdi dir="ltr">{cl.wall_lead.wall_time}</bdi></dd></div>
            <div><dt>{b.serverTime(l)}</dt><dd><bdi dir="ltr">{cl.wall_lead.server_time}</bdi></dd></div></dl>
        </div>
      )}
      {down && <div className="m01-condition">{b.ledgerDown(l).map((line, i) => <p key={i}>{i === 0 ? <strong>{line}</strong> : line}</p>)}</div>}
      {baseNotRecorded && <div className="m01-condition"><p><strong>{b.baseNotRecorded(l, cl.clock_base.choice)}</strong></p></div>}
      {(cl.clock_behind || cl.clock_ahead) && <div className="m01-condition"><p>{p.clockDiffers(l)}</p></div>}
    </section>
  );
}

export default function LiveUsers({ locale: l, view, session, onSignOut, onView, params = new URLSearchParams(), onParams = () => {} }: {
  locale: Locale; view: string; session: Session; onSignOut: () => void; onView: (view: string) => void;
  params?: URLSearchParams; onParams?: (next: Record<string, string | null>) => void;
}) {
  const api = useAdminClient();
  const wide = useWide();
  const order = useRef(sequence());
  const link = parseLink(params);
  const filters = link.filters;
  const query = queryOf(view, filters);
  const queryKey = JSON.stringify(query);
  const queryRef = useRef(query);
  queryRef.current = query;
  const [data, setData] = useState<Loaded | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState<unknown>(null);
  const [paging, setPaging] = useState<Paging>("idle");
  const [changed, setChanged] = useState(false);
  const [badLink, setBadLink] = useState(false);
  const [bandOpen, setBandOpen] = useState(false);
  const [freshRead, setFreshRead] = useState(true);
  const [actionResult, setActionResult] = useState<{ who: string; login: boolean; text: string } | null>(null);
  const [selected, setSelected] = useState("");
  const [mode, setMode] = useState<Mode>({ kind: "none" });
  const [notice, setNotice] = useState("");
  const [closedNote, setClosedNote] = useState(false);
  const [announce, setAnnounce] = useState("");
  const [focusTo, setFocusTo] = useState("");
  const [picked, setPicked] = useState<Selection>(() => (kept.user === session.user ? kept.selection : new Set()));
  useEffect(() => { kept = { user: session.user, selection: picked }; }, [picked, session.user]);
  const [fixed, setFixed] = useState<Fixed | null>(() => (keptFixed.user === session.user ? keptFixed.fixed : null));
  const [removing, setRemoving] = useState(false);
  const [removePicked, setRemovePicked] = useState<Selection>(new Set());
  const [removeBusy, setRemoveBusy] = useState(false);
  const [removeNote, setRemoveNote] = useState("");
  const fixedDraft = fixed?.draft ?? "";
  useEffect(() => { setRemoving(false); setRemovePicked(new Set()); setRemoveNote(""); setLimitHit(false); }, [fixedDraft]); // eslint-disable-line react-hooks/exhaustive-deps
  const [fixError, setFixError] = useState<unknown>(null);
  useEffect(() => { keptFixed = { user: session.user, fixed }; }, [fixed, session.user]);
  const [bulk, setBulk] = useState(false);
  const [job, setJob] = useState(() => { const id = openJobNext; openJobNext = ""; return id; });
  const [limitHit, setLimitHit] = useState(false);
  const names = useRef<Record<string, string>>({});
  const selectedRef = useRef("");
  selectedRef.current = selected;
  const modeRef = useRef("none");
  modeRef.current = mode.kind;
  const noticeRef = useRef("");
  noticeRef.current = notice;
  const dataRef = useRef<Loaded | null>(null);
  dataRef.current = data;
  const pending = useRef<"" | "filter" | "removed">("");
  const shownQuery = useRef("");
  useEffect(() => {
    if (!focusTo) return;
    document.querySelector<HTMLElement>(focusTo)?.focus();
    setFocusTo("");
  }, [focusTo]);
  const paramsKey = params.toString();
  useEffect(() => {
    if (!link.bad) return;
    setBadLink(true);
    onParams(linkOf(filters));
  }, [paramsKey]);
  const say = (text: string) => {
    setAnnounce("");
    setTimeout(() => setAnnounce(text), 50);
  };
  const tell = (text: string) => { setNotice(text); say(text); };
  const countLine = (page: { next: string; users: UserRow[] }) => (page.users.length === 0 ? fc.noneShown(l) : page.next ? fc.shownMore(l, page.users.length) : fc.shownAll(l, page.users.length));
  const load = useCallback((keepNotice = false) => {
    const n = order.current.next();
    const asked = queryRef.current;
    const key = JSON.stringify(asked);
    const fresh = shownQuery.current !== key;
    setFreshRead(fresh);
    if (fresh) setActionResult(null);
    setStatus("loading");
    setPaging("idle");
    if (fresh) setData((d) => (d ? { ...d, users: [], next: "", next_login: "", scan_stopped: false } : d));
    if (!keepNotice) setNotice("");
    return api.users(asked).then(
      (page) => {
        if (!order.current.latest(n)) return page.revision;
        for (const u of page.users) names.current[u.user_id] = u.login || u.user_id;
        shownQuery.current = key;
        setFreshRead(false);
        setData({ ...page, at: new Date(), first: page.list_version });
        setChanged(false);
        setStatus("ready");
        setError(null);
        return page.revision;
      },
      (e) => {
        if (!order.current.latest(n)) return "";
        setError(e);
        setStatus("error");
        return "";
      },
    );
  }, [api, queryKey]);
  useEffect(() => { void load(); }, [load]);
  const canBulk = !!data?.can_bulk_revoke;
  useEffect(() => {
    if (!canBulk) return;
    let live = true;
    bulkClient(api).list("", 1).then((page) => {
      if (!live) return;
      const found = page.jobs[0] ? fixedOfJob(page.jobs[0], session.user) : null;
      setFixed((was) => (found && was && was.draft === found.draft ? { ...found, counted: was.counted, countedAtLeast: was.countedAtLeast } : found));
    }, () => undefined);
    return () => { live = false; };
  }, [canBulk, api, session.user]);
  useEffect(() => { if (fixed || bulk) setClosedNote(false); }, [fixed, bulk]);
  useEffect(() => { setMode((m) => ({ kind: m.kind === "none" ? "none" : "detail" })); setNotice(""); setClosedNote(false); setActionResult(null); setBulk(false); setBandOpen(false); }, [view]);
  const users = data?.users ?? [];
  const applied = appliedKeys(filters).length;
  useEffect(() => {
    if (status !== "ready" || !data) return;
    const kind = pending.current;
    pending.current = "";
    const lostId = selectedRef.current;
    const lost = lostPerson(modeRef.current, lostId, data.users);
    if (lost) {
      const inPanel = !!document.activeElement?.closest("aside.live-panel") || document.activeElement === document.body;
      setMode({ kind: "none" });
      setSelected("");
      if (inPanel) setFocusTo(wide ? "#m01-q" : filterViews.includes(view) ? "#m01-filters-toggle" : ".live-toolbar h2");
    }
    const count = countLine(data);
    const gone = lost ? (appliedKeys(filters).length > 0 ? fc.openFiltered(l) : fc.openNotLoaded(l)) : "";
    if (!kind && !gone) return;
    const result = lost && !kind ? noticeRef.current : "";
    const who = names.current[lostId] || lostId;
    if (result) setActionResult({ who, login: who !== lostId, text: result });
    say([kind === "removed" ? fc.removed(l, count) : kind ? fc.updated(l, count, appliedKeys(filters).length) : "", result ? `${who}: ${bare(result.replace(/\n/g, " "))}.` : "", gone].filter(Boolean).join(" "));
  }, [status, data]);
  const apply = (next: UserFilters, kind: "filter" | "removed" = "filter") => {
    pending.current = kind;
    setBadLink(false);
    onParams(linkOf(next));
  };
  const removeChip = (key: FilterKey) => {
    const keys = appliedKeys(filters);
    const i = keys.indexOf(key);
    const target = keys[i + 1] ?? keys[i - 1];
    setFocusTo(target ? "#m01-chip-" + target : wide ? "#m01-q" : "#m01-filters-toggle");
    apply(dropFilter(filters, key), "removed");
  };
  const clearAll = () => { setFocusTo(wide ? "#m01-q" : "#m01-filters-toggle"); apply(noFilters, "removed"); };
  const row = (id: string) => `.live-open[data-user-id="${CSS.escape(id)}"]`;
  const more = () => {
    const d = dataRef.current;
    if (!d?.next || paging === "more") return;
    const n = order.current.next();
    setPaging("more");
    api.users({ ...queryRef.current, after: d.next }).then(
      (page) => {
        if (!order.current.latest(n)) return;
        for (const u of page.users) names.current[u.user_id] = u.login || u.user_id;
        const before = new Set(d.users.map((u) => u.user_id));
        const added = page.users.find((u) => !before.has(u.user_id));
        setData((x) => (x ? { ...page, users: merge(x.users, page.users), at: new Date(), first: x.first } : x));
        if (page.list_version !== d.first) setChanged(true);
        setPaging("idle");
        if (added) setFocusTo(row(added.user_id));
      },
      (e) => {
        if (!order.current.latest(n)) return;
        const restart = e instanceof AdminError && e.failure === "conflict" && e.fields.includes("after");
        setPaging(restart ? "restart" : "failed");
        if (restart) setFocusTo("#m01-start-again");
      },
    );
  };
  const self = users.find((u) => u.user_id === session.user);
  const user = users.find((u) => u.user_id === selected) ?? null;
  const ctx: Context = { clock: data?.clock ?? { server_time: "", wall_time: "", clock_behind: false, clock_ahead: false, clock_base: { choice: "default", ledger_time: "", wall_time: "", recorded: false }, wall_lead: null, recovery: { stranded: false, records: [] } },
    settings: data?.settings ?? [], own: self?.membership?.role_until ?? "" };
  const limit = setting(ctx.settings, "bulk_revocation_max_targets", 500);
  const days = data?.warning_days || setting(ctx.settings, "role_period_warning_days", 14);
  const open = (id: string) => { setSelected(id); setNotice(""); setActionResult(null); setMode({ kind: "detail" }); setFocusTo(wide ? "" : "#live-detail-title"); };
  const close = () => { const id = selected; setMode({ kind: "none" }); setSelected(""); setFocusTo(row(id)); };
  const backToDetail = (action?: Action) => { setMode({ kind: "detail" }); setFocusTo(action ? `[data-action="${action}"]` : "#live-detail-title"); };
  const stranded = ctx.clock.recovery.stranded;
  const scope = <ScopeLine l={l} workspace={session.workspace} />;
  if (view === "revocation") {
    return (
      <div className="live-users m01-live" data-live="revocation" data-wide={wide ? "true" : "false"}>
        <p className="visually-hidden" role="status" aria-live="polite">{announce}</p>
        {scope}
        {data && <Conditions l={l} data={data} />}
        {applied > 0 && <p className="m01-note">{fc.jobsNote(l)}</p>}
        <RevocationJobs l={l} api={api} wide={wide} initial={job} self={session.user} revision={data?.revision ?? ""} onSignOut={onSignOut} />
      </div>
    );
  }
  if (status === "error" && error instanceof AdminError && error.failure === "denied") {
    return (
      <div className="live-users" data-live={view} data-state="denied">
        <section className="live-unavailable" aria-labelledby="live-denied">
          <h2 id="live-denied">{w.deniedTitle(l)}</h2>
          <p>{w.denied(l)}</p>
        </section>
      </div>
    );
  }
  const title = view === "invitations" ? w.invitationsTitle(l) : view === "membership" ? w.membershipTitle(l) : w.usersTitle(l);
  const selectable = !!data?.can_bulk_revoke && !stranded;
  const pendingLimit = setting(ctx.settings, "bulk_revocation_pending", 20);
  const fixedNow = selectable ? fixed : null;
  const reloadRevision = async () => (await load(true)) || dataRef.current?.revision || "";
  const removeSelected = async () => {
    if (!fixedNow || removePicked.size === 0 || removeBusy) return;
    const draft = fixedNow.draft;
    setRemoveBusy(true);
    setFixError(null);
    let sum: RemovalSum = { removed: 0, already: 0, ignored: 0, chosen: fixedNow.chosen };
    let failed: unknown = null;
    try {
      for (const batch of batches([...removePicked])) {
        const done = await api.command("bulk_revocation.remove_targets", draft, await reloadRevision(), m.removeReason(l), { users: batch.join("\n") }, attempt());
        sum = removalOf(done.fields ?? {}, sum);
      }
    } catch (e) {
      failed = e;
    }
    setRemoveBusy(false);
    setFixed((was) => (was && was.draft === draft ? { ...was, chosen: sum.chosen } : was));
    const note = [sum.removed > 0 ? m.removedNow(l, sum.removed) : "", sum.already > 0 ? m.removedAlready(l, sum.already) : "", sum.ignored > 0 ? m.removedIgnored(l, sum.ignored) : ""].filter(Boolean).join(" ");
    setRemoveNote(note);
    if (failed !== null) {
      if (failed instanceof AdminError && failed.failure === "stage") { closedNow(); return; }
      setFixError(failed);
      if (note) say(note);
      return;
    }
    setRemovePicked(new Set());
    setRemoving(false);
    setFocusTo(sum.chosen > 0 ? ".m01-removed" : ".m01-selcount");
    say(note || m.removedNow(l, 0));
  };
  const gone = (text: string) => { setFixed(null); setFixError(null); setFocusTo(".m01-match-open"); say(text); };
  const closedNow = () => { setFixed(null); setFixError(null); setFocusTo(".m01-match-open"); setClosedNote(true); say(m.alreadyClosed(l)); };
  const discardFixed = async () => {
    if (!fixedNow) return;
    try {
      await api.command("bulk_revocation.discard", fixedNow.draft, await reloadRevision(), b.discardReason(l), {}, attempt());
    } catch (e) {
      if (!(e instanceof AdminError && e.failure === "stage")) { setFixError(e); return; }
      closedNow();
      return;
    }
    gone(m.discarded(l));
  };
  const loaded = selectableIds(status === "ready" ? "ready" : freshRead ? "fresh" : "reloading", users, session.user);
  const sel: Select = {
    on: selectable && (!fixedNow || removing), keeping: !!fixedNow, selected: fixedNow ? removePicked : picked, self: session.user, atLimit: (fixedNow ? removePicked : picked).size >= limit, loaded,
    onToggle: (id) => {
      const out = toggle(fixedNow ? removePicked : picked, id, limit);
      setLimitHit(out.refused);
      if (out.refused) say(b.limit(l, limit));
      else if (fixedNow) setRemovePicked(out.next);
      else setPicked(out.next);
    },
    onAll: () => {
      const out = selectLoaded(fixedNow ? removePicked : picked, loaded, limit);
      setLimitHit(out.refused);
      if (out.refused) say(b.limit(l, limit));
      else if (fixedNow) setRemovePicked(out.next);
      else setPicked(out.next);
    },
  };
  const doneAction = (id: string, action: Action, ref: string) => {
    void load(true);
    if (selectedRef.current !== id) return;
    const target = users.find((u) => u.user_id === id);
    tell(actionDone[action](l));
    if (target && (action === "issue_set_link" || action === "issue_reset_link")) {
      setMode({ kind: "link", login: target.login, purpose: action === "issue_set_link" ? "set" : "reset", ref });
    } else {
      setMode({ kind: "detail" });
      setFocusTo("#live-detail-title");
    }
  };
  if (bulk && data && selectable) {
    const ids = fixedNow ? [] : [...picked];
    const admins = ids.filter((id) => mayBeAdministrator(users.find((u) => u.user_id === id)));
    return (
      <div className="live-users m01-live" data-live={view} data-wide={wide ? "true" : "false"} data-bulk="effect">
        <p className="visually-hidden" role="status" aria-live="polite">{announce}</p>
        <Conditions l={l} data={data} />
        <BulkFlow l={l} api={api} ids={ids} names={names.current} admins={admins} rows={users} serverTime={ctx.clock.server_time} revision={data.revision}
          fixed={fixedNow} days={days} onPrepared={() => setFixed(null)} onDiscarded={(closed) => { setBulk(false); if (closed) closedNow(); else gone(m.discarded(l)); }}
          onJobs={() => { setBulk(false); onView("revocation"); }} reload={reloadRevision}
          onBack={() => { setBulk(false); setFocusTo(".m01-prepare"); }} say={say}
          onDeselect={(id) => setPicked(without(picked, [id]))}
          onExecuted={(id, targets) => { setPicked(without(picked, targets)); setBulk(false); setJob(id); openJobNext = id; void load(true); onView("revocation"); }} />
      </div>
    );
  }
  let panel: ReactNode = null;
  if (mode.kind === "create") {
    panel = <Create l={l} api={api} revision={data?.revision ?? ""} ctx={ctx} onReload={() => void load(true)}
      onCancel={() => { if (selected) backToDetail(); else { setMode({ kind: "none" }); setFocusTo(".live-create"); } }}
      onDone={(id, login, ref) => { setSelected(id); tell(w.created(l)); setMode({ kind: "link", login, purpose: "set", ref }); void load(true); }} />;
  } else if (mode.kind === "link") {
    const target = users.find((u) => u.login === mode.login);
    panel = <Pickup key={mode.ref} l={l} api={api} login={mode.login} purpose={mode.purpose} refId={mode.ref} expires={target?.link_expires ?? ""} notice={notice}
      onDone={() => { setNotice(""); if (selected) backToDetail(); else { setMode({ kind: "none" }); setFocusTo(".live-create"); } }} />;
  } else if (mode.kind === "period" && user && user.membership) {
    panel = <PeriodForm l={l} user={user} own={ctx.own} clock={ctx.clock} settings={ctx.settings} api={api} revision={data?.revision ?? ""} onReload={() => void load(true)}
      onCancel={() => backToDetail("set_role_period")}
      onDone={(lines) => { void load(true); tell(lines.join("\n")); setMode({ kind: "detail" }); setFocusTo("#live-detail-title"); }} />;
  } else if (mode.kind === "confirm" && user) {
    const action = mode.action;
    panel = <Confirm key={action + user.user_id} l={l} user={user} action={action} api={api} revision={data?.revision ?? ""} ctx={ctx}
      onDone={doneAction} onReload={() => void load(true)} onCancel={() => backToDetail(action)} />;
  } else if (user && mode.kind !== "none") {
    panel = <Detail l={l} user={user} self={session.user} notice={notice} serverTime={ctx.clock.server_time} onJob={(id) => { setJob(id); openJobNext = id; onView("revocation"); }}
      onAction={(a) => { setNotice(""); setMode(a === "set_role_period" ? { kind: "period" } : { kind: "confirm", action: a }); }} />;
  }
  const stepping = !wide && panel !== null;
  const failure = error ? failureText(l, error) : null;
  const count = data ? countLine(data) : "";
  const paging_ = data && (
    <PagingArea l={l} next={data.next} nextLogin={data.next_login} stopped={data.scan_stopped} paging={paging} changed={changed} count={count} compact={!wide && typeof matchMedia === "function" && matchMedia("(max-width: 767px)").matches}
      onMore={more} onReload={() => void load()} onRestart={() => void load()} />
  );
  const filtered = applied > 0;
  const empty = data?.scan_stopped ? fc.noMatchesYet(l) : filtered ? fc.filteredEmpty(l) : view === "invitations" ? w.noInvitations(l) : w.empty(l);
  return (
    <div className="live-users m01-live" data-live={view} data-wide={wide ? "true" : "false"} data-open={panel ? "true" : "false"} data-bulk="none">
      <p className="visually-hidden" role="status" aria-live="polite">{announce}</p>
      {scope}
      {data && <Conditions l={l} data={data} />}
      {!stepping && badLink && <p className="service-status m01-badlink" role="status">{fc.badLink(l)}</p>}
      {!stepping && closedNote && <p className="service-status m01-closed-note">{m.alreadyClosed(l)}</p>}
      {!stepping && filterViews.includes(view) && (
        <FilterBand l={l} view={view} wide={wide} open={bandOpen} filters={filters} days={days} auth={data?.auth_methods ?? { connected: [], not_connected: [] }} onToggle={() => setBandOpen((o) => !o)}
          onChange={(next) => apply(next)} onRemove={removeChip} onClearAll={clearAll} />
      )}
      {!stepping && (
        <div className="live-list m01-list">
          <div className="live-toolbar">
            <div className="live-heading">
              <h2>{title}</h2>
              {status === "ready" && users.length > 0 && <p className="live-meta m01-count">{count}</p>}
              <p className="live-meta m01-order">{fc.order(l)}</p>
              {data && <p className="live-meta">{fill(w.loadedAt(l, "{at}"), <bdi className="m01-part">{new Intl.DateTimeFormat(tags[l], { timeStyle: "short" }).format(data.at)}</bdi>)}</p>}
              {wide && !panel && <p className="live-meta">{w.pick(l)}</p>}
            </div>
            <div className="live-tools">
              <button type="button" className="live-secondary" onClick={() => void load()}>{w.reload(l)}</button>
              {view !== "membership" && !stranded && (
                <button type="button" className="live-primary live-create" onClick={() => { setNotice(""); setMode({ kind: "create" }); }}>{w.create(l)}</button>
              )}
            </div>
          </div>
          {actionResult && (
            <div className="service-status m01-actionresult" role="status">
              {actionResult.text.split("\n").map((line, i) => (
                <p key={i}>{i === 0 && <><bdi dir="ltr" className={actionResult.login ? undefined : "live-mono"}>{actionResult.who}</bdi>: </>}{line}</p>
              ))}
            </div>
          )}
          {selectable && view !== "invitations" && fixedNow && (
            <FixedBar l={l} f={fixedNow} filters={filters} days={days} number={number(fixedNow.draft)} error={fixError} say={say}
              onPrepare={() => setBulk(true)} onDiscard={() => void discardFixed()}
              removal={{ on: removing, selected: removePicked.size, busy: removeBusy, note: removeNote,
                onStart: () => { setRemoving(true); setRemoveNote(""); setFocusTo(".m01-remove-cancel"); },
                onCancel: () => { setRemoving(false); setRemovePicked(new Set()); setFocusTo(".m01-remove-open"); },
                onRemove: () => void removeSelected() }} />
          )}
          {selectable && view !== "invitations" && !fixedNow && (
            <SelectionBar l={l} selected={picked} loaded={loaded} limit={limit} atLimit={limitHit || picked.size >= limit} names={names.current}
              match={<MatchAction l={l} api={api} query={query} filters={filters} picked={picked.size} pendingLimit={pendingLimit} reload={reloadRevision} say={say}
                onJobs={() => onView("revocation")}
                onFixed={(f) => { setPicked(new Set()); setFixed(f); setFixError(null); setFocusTo(".m01-selcount"); say(m.afterFix(l, plainFixedLine(l, f, dateParts(l, f.fixed_at).time))); }} />}
              onPrepare={() => setBulk(true)} onClear={() => { setPicked(new Set()); setFocusTo(".m01-selcount"); }} onRemove={(id) => setPicked(without(picked, [id]))} onAll={sel.onAll} />
          )}
          {selectable && view === "invitations" && picked.size > 0 && <p className="m01-selline">{b.elsewhere(l, picked.size)}</p>}
          {status === "loading" && users.length === 0 && <p role="status" className="live-meta">{w.loading(l)}</p>}
          {status === "error" && failure && (
            <section className="live-error" role="alert">
              <h2>{w.loadFailed(l)}</h2>
              <p>{failure.body}</p>
              {error instanceof AdminError && error.failure === "unauthenticated" ? (
                <button type="button" className="live-secondary" onClick={onSignOut}>{w.signInAgain(l)}</button>
              ) : (
                <button type="button" className="live-secondary" onClick={() => void load()}>{c.tryAgain(l)}</button>
              )}
            </section>
          )}
          {data && status === "ready" && users.length === 0 ? (
            <div className={"live-empty m01-empty" + (data.scan_stopped ? " m01-stop-empty" : "")}>
              <p>{empty}</p>
              {filtered && !data.scan_stopped && <button type="button" className="service-button-link" onClick={clearAll}>{fc.clearAll(l)}</button>}
              {data.scan_stopped && paging_}
            </div>
          ) : users.length > 0 && status !== "error" && (
            <div aria-busy={status === "loading" || undefined}>
              <Table l={l} view={view} users={users} selected={selected} onOpen={open} sel={sel} serverTime={ctx.clock.server_time} />
            </div>
          )}
          {data && status !== "error" && !(status === "ready" && users.length === 0 && data.scan_stopped) && paging_}
        </div>
      )}
      {(stepping || (wide && panel)) && (
        <aside className="live-panel" aria-label={title}>
          {stepping && (
            <button type="button" className="service-button-link live-back" onClick={close}>{w.back(l)}</button>
          )}
          {stepping && status === "error" && failure && (
            <section className="live-error m01-panel-kept" role="alert">
              <h2>{w.loadFailed(l)}</h2>
              <p>{diagnosticCopy.kept(l)}</p>
              {error instanceof AdminError && error.failure === "unauthenticated" ? (
                <button type="button" className="live-secondary" onClick={onSignOut}>{w.signInAgain(l)}</button>
              ) : (
                <button type="button" className="live-secondary" onClick={() => void load()}>{c.tryAgain(l)}</button>
              )}
            </section>
          )}
          {stranded && panel ? <p className="live-unavailable">{b.recovery(l)}</p> : panel}
        </aside>
      )}
    </div>
  );
}
