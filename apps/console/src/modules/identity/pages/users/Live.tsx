import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { adminClient, AdminError, attempt, sequence, type AdminClient, type Attempt, type CommandId, type UserRow } from "../../../../service/admin";
import type { Session } from "../../../../service/api";
import { c } from "../../../../service/copy";
import { emailValid, loginValid, utf8Length } from "../../../../service/helpers";
import { Summary, TextField, type Problems } from "../../../../service/Service";
import { actionDone, actionLabel, impact, w, type Action } from "./live-copy";
import { accountName, failureText, offers, roleName, setup, tags, when } from "./live-model";
import "./live.css";

type Mode =
  | { kind: "none" }
  | { kind: "detail" }
  | { kind: "confirm"; action: Action }
  | { kind: "create" }
  | { kind: "link"; login: string; purpose: "set" | "reset"; ref: string };

type Loaded = { revision: string; next: string; users: UserRow[]; at: Date };

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
  return !(e instanceof AdminError) || !["unavailable", "limited", "unauthenticated"].includes(e.failure);
}

function Confirm({ l, user, action, api, revision, onDone, onCancel, onReload }: {
  l: Locale; user: UserRow; action: Action; api: AdminClient; revision: string;
  onDone: (user: string, action: Action, ref: string) => void; onCancel: () => void; onReload: () => void;
}) {
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
    setProblem(p);
    if (p) { setError(null); setFocusKey((k) => k + 1); return; }
    setBusy(true);
    setError(null);
    try {
      const body: Record<string, string> = action === "change_role" || action === "invite" ? { role } : {};
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
      {problem && <Summary locale={l} title={c.check(l)} problems={[{ field: "live-reason", message: problem }]} focusKey={focusKey} />}
      {error !== null && <FailureBox l={l} error={error} focusKey={focusKey} />}
      <p className="live-impact">{impact(l, action, user.login || user.user_id, roleName(l, current), roleName(l, role))}</p>
      {(action === "change_role" || action === "invite") && (
        <RoleChoice l={l} value={role} onChange={edit(setRole)} allowAdmin={allowAdmin} />
      )}
      <Reason l={l} value={reason} onChange={edit(setReason)} error={problem} />
      <div className="live-buttons">
        <button type="submit" className="live-primary" aria-busy={busy || undefined} disabled={busy}>{busy ? w.saving(l) : actionLabel[action](l)}</button>
        <button type="button" className="live-secondary" onClick={onCancel}>{w.cancel(l)}</button>
      </div>
    </form>
  );
}

function Create({ l, api, revision, onDone, onCancel, onReload }: {
  l: Locale; api: AdminClient; revision: string;
  onDone: (user: string, login: string, ref: string) => void; onCancel: () => void; onReload: () => void;
}) {
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
    const r = reasonProblem(l, reason);
    if (r) found.push({ field: "live-reason", message: r });
    setProblems(found);
    setError(null);
    if (found.length) { setFocusKey((k) => k + 1); return; }
    setBusy(true);
    try {
      const out = await api.command("identity.create_user", "users", revision, reason, { login, email, role }, tried.current);
      onDone(out.fields.user_id ?? "", login.toLowerCase(), out.fields.link_ref ?? "");
    } catch (err) {
      if (err instanceof AdminError && err.failure === "refused") setProblems([{ field: "live-login", message: w.loginRefused(l) }]);
      else if (err instanceof AdminError && err.failure === "invalid" && err.fields.length) {
        setProblems(err.fields.map((f) => ({ field: f === "email" ? "live-email" : "live-login", message: f === "email" ? c.badContact(l) : c.badSignInEmail(l) })));
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
      (e) => setState(e instanceof AdminError && e.failure === "unavailable" ? "retry" : "gone"),
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
          {expires && <p>{w.linkExpires(l, when(l, expires))}</p>}
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

function Detail({ l, user, self, notice, onAction }: {
  l: Locale; user: UserRow; self: string; notice: string; onAction: (a: Action) => void;
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
      {notice && <p className="service-status">{notice}</p>}
      <h2 id="live-detail-title" tabIndex={-1}><bdi dir="ltr">{user.login || user.user_id}</bdi></h2>
      <dl className="live-facts">
        <div><dt>{c.contactEmail(l)}</dt><dd><bdi dir="ltr">{user.email || "—"}</bdi></dd></div>
        <div><dt>{w.accountStatus(l)}</dt><dd className={user.state === "active" ? "" : "live-strong"}>{accountName(l, user.state)}</dd></div>
        <div><dt>{w.workspaceRole(l)}</dt><dd>{roleName(l, user.membership?.role ?? "")}</dd></div>
        <div><dt>{w.membershipStatus(l)}</dt><dd>{user.membership ? (user.membership.state === "active" ? w.active(l) : w.membershipSuspended(l)) : w.noMembership(l)}</dd></div>
        <div><dt>{w.signInSetup(l)}</dt><dd>{setup(l, user)}</dd></div>
        {user.installation_administrator && <div><dt>{w.installationAdmin(l)}</dt><dd>{w.yes(l)}</dd></div>}
        <div><dt>{w.userId(l)}</dt><dd className="live-mono"><bdi dir="ltr">{user.user_id}</bdi></dd></div>
      </dl>
      {offer.blocked ? (
        <p className="live-note live-blocked">{offer.blocked}</p>
      ) : (
        <>
          {group(w.account(l), offer.account)}
          {group(w.membershipTitle(l), offer.membership, offer.notes.filter((n) => n === w.promotionReason(l)))}
          {group(w.sign(l), offer.sign, offer.notes.filter((n) => n !== w.promotionReason(l)))}
        </>
      )}
    </section>
  );
}

function rowsFor(view: string, users: UserRow[]): UserRow[] {
  return view === "invitations" ? users.filter((u) => u.invitation_expires || u.link_purpose === "set") : users;
}

function Cell({ label, className, children }: { label: string; className?: string; children: ReactNode }) {
  return <td role="cell" data-label={label} className={className}>{children}</td>;
}

function Table({ l, view, users, selected, onOpen }: { l: Locale; view: string; users: UserRow[]; selected: string; onOpen: (id: string) => void }) {
  const head = view === "invitations"
    ? [w.user(l), w.role(l), w.invitationExpires(l), w.setLink(l)]
    : view === "membership"
      ? [w.user(l), w.workspaceRole(l), w.membershipStatus(l), w.installationAdmin(l)]
      : [w.user(l), w.accountStatus(l), w.workspaceRole(l), w.signInSetup(l)];
  return (
    <div className="security-table-scroll live-table" data-view={view}>
      <table role="table">
        <colgroup><col className="live-col-user" /><col /><col /><col /></colgroup>
        <thead role="rowgroup"><tr role="row">{head.map((h) => <th key={h} scope="col" role="columnheader">{h}</th>)}</tr></thead>
        <tbody role="rowgroup">
          {users.map((u) => (
            <tr key={u.user_id} role="row" className={u.user_id === selected ? "live-selected" : undefined}>
              <td role="cell" className="live-user">
                <button type="button" className="live-open" data-user-id={u.user_id} aria-current={u.user_id === selected ? "true" : undefined} onClick={() => onOpen(u.user_id)}>
                  <bdi dir="ltr">{u.login || u.user_id}</bdi>
                  <span className="visually-hidden">, {w.openDetails(l)}{u.user_id === selected ? " " + w.selected(l) : ""}</span>
                </button>
                {u.email && u.email !== u.login && <span className="live-sub"><bdi dir="ltr">{u.email}</bdi></span>}
              </td>
              {view === "invitations" ? (
                <>
                  <Cell label={head[1]}>{roleName(l, u.membership?.role ?? "")}</Cell>
                  <Cell label={head[2]}>{u.invitation_expires ? when(l, u.invitation_expires) : "—"}</Cell>
                  <Cell label={head[3]} className="live-quiet">{u.link_purpose === "set" ? w.setIssued(l, when(l, u.link_expires)) : w.notIssued(l)}</Cell>
                </>
              ) : view === "membership" ? (
                <>
                  <Cell label={head[1]}>{roleName(l, u.membership?.role ?? "")}</Cell>
                  <Cell label={head[2]}>{u.membership ? (u.membership.state === "active" ? w.active(l) : w.membershipSuspended(l)) : w.noMembership(l)}</Cell>
                  <Cell label={head[3]}>{u.installation_administrator ? w.yes(l) : <><span aria-hidden="true">—</span><span className="visually-hidden">{w.no(l)}</span></>}</Cell>
                </>
              ) : (
                <>
                  <Cell label={head[1]} className={u.state === "active" ? "" : "live-strong"}>{accountName(l, u.state)}</Cell>
                  <Cell label={head[2]}>
                    {roleName(l, u.membership?.role ?? "")}
                    {u.membership?.state === "suspended" && <span className="live-sub">{w.membershipSuspended(l)}</span>}
                    {u.installation_administrator && <span className="live-sub">{w.installationAdmin(l)}</span>}
                  </Cell>
                  <Cell label={head[3]} className="live-quiet">{setup(l, u)}</Cell>
                </>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default function LiveUsers({ locale: l, view, session, onSignOut, onView }: {
  locale: Locale; view: string; session: Session; onSignOut: () => void; onView: (view: string) => void;
}) {
  const api = useMemo(() => adminClient(session.csrf), [session.csrf]);
  const wide = useWide();
  const order = useRef(sequence());
  const [data, setData] = useState<Loaded | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState<unknown>(null);
  const [selected, setSelected] = useState("");
  const [mode, setMode] = useState<Mode>({ kind: "none" });
  const [notice, setNotice] = useState("");
  const [announce, setAnnounce] = useState("");
  const [focusTo, setFocusTo] = useState("");
  const selectedRef = useRef("");
  selectedRef.current = selected;
  useEffect(() => {
    if (!focusTo) return;
    document.querySelector<HTMLElement>(focusTo)?.focus();
    setFocusTo("");
  }, [focusTo]);
  const say = (text: string) => {
    setNotice(text);
    setAnnounce("");
    setTimeout(() => setAnnounce(text), 50);
  };
  const load = useCallback((keepNotice = false) => {
    const n = order.current.next();
    setStatus("loading");
    if (!keepNotice) setNotice("");
    api.users("", 100).then(
      (page) => {
        if (!order.current.latest(n)) return;
        setData({ ...page, at: new Date() });
        setStatus("ready");
        setError(null);
      },
      (e) => {
        if (!order.current.latest(n)) return;
        setError(e);
        setStatus("error");
      },
    );
  }, [api]);
  const more = () => {
    if (!data?.next) return;
    const n = order.current.next();
    api.users(data.next, 100).then(
      (page) => {
        if (!order.current.latest(n)) return;
        setData((d) => (d ? { ...page, users: [...d.users, ...page.users], at: new Date() } : d));
      },
      (e) => { if (order.current.latest(n)) { setError(e); setStatus("error"); } },
    );
  };
  useEffect(() => { load(); }, [load]);
  useEffect(() => { setMode((m) => ({ kind: m.kind === "none" ? "none" : "detail" })); setNotice(""); }, [view]);
  const users = data?.users ?? [];
  const user = users.find((u) => u.user_id === selected) ?? null;
  const row = (id: string) => `.live-open[data-user-id="${CSS.escape(id)}"]`;
  const open = (id: string) => { setSelected(id); setNotice(""); setMode({ kind: "detail" }); setFocusTo(wide ? "" : "#live-detail-title"); };
  const close = () => { const id = selected; setMode({ kind: "none" }); setSelected(""); setFocusTo(row(id)); };
  const backToDetail = (action?: Action) => { setMode({ kind: "detail" }); setFocusTo(action ? `[data-action="${action}"]` : "#live-detail-title"); };
  if (view === "revocation") {
    return (
      <div className="live-users" data-live="revocation">
        <section className="live-unavailable" aria-labelledby="live-revocation">
          <h2 id="live-revocation">{w.revocationTitle(l)}</h2>
          <p>{w.revocationBody(l)}</p>
          <button type="button" className="live-secondary" onClick={() => onView("users")}>{w.goUsers(l)}</button>
        </section>
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
  const shown = rowsFor(view, users);
  const title = view === "invitations" ? w.invitationsTitle(l) : view === "membership" ? w.membershipTitle(l) : w.usersTitle(l);
  const doneAction = (id: string, action: Action, ref: string) => {
    load(true);
    if (selectedRef.current !== id) return;
    const target = users.find((u) => u.user_id === id);
    say(actionDone[action](l));
    if (target && (action === "issue_set_link" || action === "issue_reset_link")) {
      setMode({ kind: "link", login: target.login, purpose: action === "issue_set_link" ? "set" : "reset", ref });
    } else {
      setMode({ kind: "detail" });
      setFocusTo("#live-detail-title");
    }
  };
  let panel: ReactNode = null;
  if (mode.kind === "create") {
    panel = <Create l={l} api={api} revision={data?.revision ?? ""} onReload={() => load(true)}
      onCancel={() => { if (selected) backToDetail(); else { setMode({ kind: "none" }); setFocusTo(".live-create"); } }}
      onDone={(id, login, ref) => { setSelected(id); say(w.created(l)); setMode({ kind: "link", login, purpose: "set", ref }); load(true); }} />;
  } else if (mode.kind === "link") {
    const target = users.find((u) => u.login === mode.login);
    panel = <Pickup key={mode.ref} l={l} api={api} login={mode.login} purpose={mode.purpose} refId={mode.ref} expires={target?.link_expires ?? ""} notice={notice}
      onDone={() => { setNotice(""); if (selected) backToDetail(); else { setMode({ kind: "none" }); setFocusTo(".live-create"); } }} />;
  } else if (mode.kind === "confirm" && user) {
    const action = mode.action;
    panel = <Confirm key={action + user.user_id} l={l} user={user} action={action} api={api} revision={data?.revision ?? ""}
      onDone={doneAction} onReload={() => load(true)} onCancel={() => backToDetail(action)} />;
  } else if (user && mode.kind !== "none") {
    panel = <Detail l={l} user={user} self={session.user} notice={notice} onAction={(a) => { setNotice(""); setMode({ kind: "confirm", action: a }); }} />;
  }
  const stepping = !wide && panel !== null;
  const failure = error ? failureText(l, error) : null;
  const count = view === "invitations" ? w.invitationCount(l, shown.length) : w.count(l, shown.length);
  return (
    <div className="live-users" data-live={view} data-wide={wide ? "true" : "false"}>
      <p className="visually-hidden" role="status" aria-live="polite">{announce}</p>
      {!stepping && (
        <div className="live-list">
          <div className="live-toolbar">
            <div className="live-heading">
              <h2>{title}</h2>
              {status === "ready" && <p className="live-meta">{count}</p>}
              {data && <p className="live-meta">{w.loadedAt(l, new Intl.DateTimeFormat(tags[l], { timeStyle: "short" }).format(data.at))}</p>}
            </div>
            <div className="live-tools">
              <button type="button" className="live-secondary" onClick={() => load()}>{w.reload(l)}</button>
              {view !== "membership" && (
                <button type="button" className="live-primary live-create" onClick={() => { setNotice(""); setMode({ kind: "create" }); }}>{w.create(l)}</button>
              )}
            </div>
          </div>
          {status === "loading" && !data && <p role="status" className="live-meta">{w.loading(l)}</p>}
          {status === "error" && failure && (
            <section className="live-error" role="alert">
              <h2>{w.loadFailed(l)}</h2>
              <p>{failure.body}</p>
              {error instanceof AdminError && error.failure === "unauthenticated" ? (
                <button type="button" className="live-secondary" onClick={onSignOut}>{w.signInAgain(l)}</button>
              ) : (
                <button type="button" className="live-secondary" onClick={() => load()}>{c.tryAgain(l)}</button>
              )}
            </section>
          )}
          {data && status !== "error" && (shown.length === 0 ? (
            <p className="live-empty">{view === "invitations" ? w.noInvitations(l) : w.empty(l)}</p>
          ) : (
            <div aria-busy={status === "loading" || undefined}>
              <Table l={l} view={view} users={shown} selected={selected} onOpen={open} />
            </div>
          ))}
          {data?.next && <button type="button" className="live-secondary live-more" onClick={more}>{w.more(l)}</button>}
        </div>
      )}
      {(wide || stepping) && (
        <aside className="live-panel" aria-label={title}>
          {stepping && (
            <button type="button" className="service-button-link live-back" onClick={close}>{w.back(l)}</button>
          )}
          {panel ?? <p className="live-empty">{w.pick(l)}</p>}
        </aside>
      )}
    </div>
  );
}
