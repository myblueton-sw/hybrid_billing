import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type FormEvent, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, sequence, type AdminClient, type SelfPage } from "../../../../service/admin";
import { useAdminClient } from "../../../../service/context";
import type { Session } from "../../../../service/api";
import { c } from "../../../../service/copy";
import { passwordProblem } from "../../../../service/helpers";
import { Summary, type Problems } from "../../../../service/Service";
import { useWide } from "../users/Live";
import { w } from "../users/live-copy";
import { roleName } from "../users/live-model";
import { when } from "../authentication/live-model";
import { m } from "./live-copy";
import { canChange, changeOutcome, checkChange } from "./live-model";
import LiveSessions from "./LiveSessions";
import { useScopedRead, type ScopedValue } from "./scoped-read";
import "../users/live.css";

type Mode = { kind: "list" } | { kind: "password" };


const ltr = (s: string) => <bdi dir="ltr">{s}</bdi>;

function Facts({ items }: { items: [string, ReactNode][] }) {
  return (
    <dl className="live-facts">
      {items.map(([k, value]) => <div key={k}><dt>{k}</dt><dd>{value}</dd></div>)}
    </dl>
  );
}

function SecretField({ l, id, label, autoComplete, hint, error, field }: {
  l: Locale; id: string; label: string; autoComplete: string; hint?: string; error?: string; field: React.RefObject<HTMLInputElement | null>;
}) {
  const [shown, setShown] = useState(false);
  const described = [hint ? id + "-hint" : "", error ? id + "-error" : ""].filter(Boolean).join(" ") || undefined;
  return (
    <div className="service-field">
      <label htmlFor={id}>{label}</label>
      {hint && <p className="service-hint" id={id + "-hint"}>{hint}</p>}
      {error && <p className="service-field-error" id={id + "-error"}><span className="visually-hidden">{c.errorPrefix(l)} </span>{error}</p>}
      <div className="service-password">
        <input id={id} ref={field} type={shown ? "text" : "password"} dir="ltr" autoComplete={autoComplete} spellCheck={false} autoCapitalize="none"
          aria-invalid={error ? true : undefined} aria-describedby={described} defaultValue="" />
        <button type="button" className="service-toggle" aria-pressed={shown} aria-label={shown ? c.hideName(l, label) : c.showName(l, label)} onClick={() => setShown((s) => !s)}>
          {shown ? c.hide(l) : c.show(l)}
        </button>
      </div>
    </div>
  );
}

function PasswordForm({ l, api, page, stepping, hadWrong, pending, onWrong, onDone, onSignedOut, onCancel }: {
  l: Locale; api: AdminClient; page: SelfPage; stepping: boolean; hadWrong: boolean; pending: React.MutableRefObject<boolean>;
  onWrong: () => void; onDone: () => void; onSignedOut: (locked: boolean) => void; onCancel: () => void;
}) {
  const current = useRef<HTMLInputElement>(null);
  const next = useRef<HTMLInputElement>(null);
  const confirm = useRef<HTMLInputElement>(null);
  const heading = useRef<HTMLHeadingElement>(null);
  const result = useRef<HTMLHeadingElement>(null);
  const [busy, setBusy] = useState(false);
  const [problems, setProblems] = useState<Problems>([]);
  const [server, setServer] = useState<Problems>([]);
  const [failure, setFailure] = useState("");
  const [changed, setChanged] = useState(false);
  const [focusKey, setFocusKey] = useState(0);
  const clear = () => { for (const r of [current, next, confirm]) if (r.current) r.current.value = ""; };
  useEffect(() => { if (stepping) heading.current?.focus(); }, [stepping]);
  useLayoutEffect(() => clear, []);
  useEffect(() => { if (changed) result.current?.focus(); }, [changed]);
  const focus = (id: string) => setTimeout(() => document.getElementById(id)?.focus(), 0);
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (pending.current) return;
    const values = [current.current?.value ?? "", next.current?.value ?? "", confirm.current?.value ?? ""] as const;
    const found: Problems = [];
    const local = checkChange(...values);
    if (!values[0]) found.push({ field: "mysec-current", message: m.enterCurrent(l) });
    const rule = passwordProblem(l, values[1], page.login);
    if (rule) found.push({ field: "mysec-new", message: rule });
    else if (local === "same") found.push({ field: "mysec-new", message: m.sameAsCurrent(l) });
    if (!rule && local === "mismatch") found.push({ field: "mysec-confirm", message: c.mismatch(l) });
    if (!rule && local === "missing" && values[0] && values[1] && !values[2]) found.push({ field: "mysec-confirm", message: c.mismatch(l) });
    setProblems(found);
    setServer([]);
    setFailure("");
    if (found.length) { setFocusKey((k) => k + 1); return; }
    pending.current = true;
    setBusy(true);
    try {
      await api.changePassword(values[0], values[1]);
      clear();
      setChanged(true);
      onDone();
    } catch (err) {
      clear();
      const outcome = changeOutcome(err, hadWrong);
      if (outcome === "lockedOut" || outcome === "signedOut") { onSignedOut(outcome === "lockedOut"); return; }
      if (outcome === "wrong") {
        onWrong();
        setServer([{ field: "mysec-current", message: m.errWrong(l) }]);
        focus("mysec-current");
      } else if (outcome === "form") {
        setServer([{ field: "mysec-new", message: m.errForm(l) }]);
        focus("mysec-new");
      } else {
        setFailure(outcome === "locked" ? m.errLocked(l) : outcome === "limited" ? m.errLimited(l, err instanceof AdminError ? err.retryAfter : 0)
          : outcome === "conflict" ? m.errConflict(l) : outcome === "busy" ? m.errBusy(l) : m.changeUnknown(l));
        setFocusKey((k) => k + 1);
      }
    } finally {
      pending.current = false;
      setBusy(false);
    }
  };
  if (!canChange(page)) return <section className="live-step"><h2>{m.changeTitle(l)}</h2><p className="live-impact">{m.noPassword(l)}</p></section>;
  if (changed) {
    return (
      <section className="live-step" aria-labelledby="mysec-changed">
        <div className="service-status">
          <h3 id="mysec-changed" tabIndex={-1} ref={result}>{m.changedTitle(l)}</h3>
          <p>{m.changedBody(l)}</p>
        </div>
        <div className="live-buttons"><button type="button" className="live-secondary" onClick={() => { setChanged(false); if (stepping) onCancel(); }}>{m.done(l)}</button></div>
      </section>
    );
  }
  const field = (id: string) => [...problems, ...server].find((p) => p.field === id)?.message;
  return (
    <form className="live-step" method="post" onSubmit={submit} noValidate>
      <h2 id="mysec-pw-title" tabIndex={-1} ref={heading}>{m.changeTitle(l)}</h2>
      <p className="live-impact">{m.changeImpact(l)}</p>
      {problems.length > 0 && <Summary locale={l} title={c.check(l)} problems={problems} focusKey={focusKey} />}
      {failure && <FailureNote focusKey={focusKey} title={w.saveFailed(l)} body={failure} />}
      <span className="visually-hidden" aria-hidden="true"><input type="text" autoComplete="username" readOnly tabIndex={-1} value={page.login} /></span>
      <SecretField l={l} id="mysec-current" label={m.currentPassword(l)} autoComplete="current-password" error={field("mysec-current")} field={current} />
      <SecretField l={l} id="mysec-new" label={c.newPassword(l)} autoComplete="new-password" hint={`${c.rule(l)} ${m.differ(l)}`} error={field("mysec-new")} field={next} />
      <SecretField l={l} id="mysec-confirm" label={c.confirmPassword(l)} autoComplete="new-password" error={field("mysec-confirm")} field={confirm} />
      <p className="live-note">{m.attemptsNote(l)}</p>
      <div className="live-buttons">
        <button type="submit" className="live-primary" aria-busy={busy || undefined} disabled={busy}>{busy ? w.saving(l) : m.changeTitle(l)}</button>
        <button type="button" className="live-secondary" onClick={() => { clear(); setProblems([]); setServer([]); setFailure(""); onCancel(); }}>{w.cancel(l)}</button>
      </div>
    </form>
  );
}

function FailureNote({ focusKey, title, body }: { focusKey: number; title: string; body: string }) {
  const box = useRef<HTMLDivElement>(null);
  useEffect(() => { if (focusKey) box.current?.focus(); }, [focusKey]);
  return (
    <div className="service-summary" role="alert" tabIndex={-1} ref={box}>
      <h2>{title}</h2>
      <p>{body}</p>
    </div>
  );
}

function Unavailable({ title, body, extra, action }: { title: string; body: string; extra?: string; action: { label: string; go: () => void } }) {
  return (
    <div className="live-users">
      <section className="live-unavailable" aria-labelledby="mysec-unavailable">
        <h2 id="mysec-unavailable">{title}</h2>
        <p>{body}</p>
        {extra && <p>{extra}</p>}
        <button type="button" className="live-secondary" onClick={action.go}>{action.label}</button>
      </section>
    </div>
  );
}

export default function LiveMySecurity({ locale: l, view, session, onSignOut, onView, params = new URLSearchParams(), onParams = () => {} }: {
  locale: Locale; view: string; session: Session; onSignOut: () => void; onView: (view: string) => void; params?: URLSearchParams; onParams?: (extra: Record<string,string | null>) => void;
}) {
  const api = useAdminClient();
  const wide = useWide();
  const identityScope = useMemo(() => ({ api,user:session.user,workspace:session.workspace }),[api,session.user,session.workspace]);
  const readKey = useMemo(() => ({ identityScope,view }),[identityScope,view]);
  const guard = useScopedRead(readKey);
  const [pageData,setPageData] = useState<ScopedValue<SelfPage> | null>(null);
  const page = pageData?.scope === guard.token ? pageData.value : null;
  const setPage = (value: SelfPage | null) => { if (guard.token.active) setPageData(value ? { scope:guard.token,value } : null); };
  const [status,setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [mode,setMode] = useState<Mode>({ kind:"list" });
  const [blocked,setBlocked] = useState<{ scope:object; value:"" | "plain" | "locked" } | null>(null);
  const signedOut = blocked?.scope === identityScope ? blocked.value : "";
  const setSignedOut = (value: "" | "plain" | "locked") => { if (guard.token.active) setBlocked({ scope:identityScope,value }); };
  const [hadWrong,setHadWrong] = useState(false);
  const [notice,setNotice] = useState("");
  const [formKey,setFormKey] = useState(0);
  const pending = useRef(false);
  const load = useCallback(() => {
    const valid = guard.begin();
    api.me().then(got => { if (!valid()) return;if (got.user_id !== identityScope.user || got.workspace !== identityScope.workspace) { setPage(null);setStatus("error");return; }setPage(got);setStatus("ready"); },error => {
      if (!valid()) return;
      if (error instanceof AdminError && error.failure === "unauthenticated") { setSignedOut("plain");setPage(null); }
      if (error instanceof AdminError && error.failure === "invalid") setPage(null);
      setStatus("error");
    });
  },[api,guard,identityScope]);
  useEffect(() => { setPage(null);setStatus("loading");if (view !== "recovery" && view !== "notifications") load();},[guard,view]);
  useEffect(() => { setMode({ kind:"list" });setNotice("");if (!pending.current) setFormKey(key => key+1); },[view,l]);
  if (signedOut) return <div className="live-users"><section className="live-unavailable"><h2>{m.signedOutTitle(l)}</h2><p>{signedOut === "locked" ? m.lockedOutBody(l) : m.signedOutBody(l)}</p><button type="button" className="live-primary" autoFocus onClick={onSignOut}>{w.signInAgain(l)}</button></section></div>;
  if (view === "recovery") return <Unavailable title={m.recoveryTitle(l)} body={m.recoveryBody(l)} extra={m.recoveryForgot(l)} action={{ label:m.goAccount(l),go:() => onView("account") }} />;
  if (view === "notifications") return <Unavailable title={m.notifTitle(l)} body={m.notifBody(l)} action={{ label:m.goSessions(l),go:() => onView("sessions") }} />;
  const say = (text: string) => { if (guard.token.active) setNotice(text); };
  const announcer = <p role="status" aria-live="polite" className="visually-hidden">{notice}</p>;
  const toolbar = (id: string,title: string,meta: ReactNode,tools?: ReactNode) => <div className="live-toolbar"><div className="live-heading"><h2 id={id} tabIndex={-1}>{title}</h2>{meta}{page && <p className="live-meta">{w.loadedAt(l,when(l,page.now))}</p>}</div><div className="live-tools">{tools}<button type="button" className="live-secondary" onClick={load}>{w.reload(l)}</button></div></div>;
  const loadState = <>{status === "loading" && !page && <p role="status">{m.loading(l)}</p>}{status === "error" && <section className="live-error" role="alert"><h2>{m.loadFailed(l)}</h2><p>{m.loadRetry(l)}</p><button type="button" className="live-secondary" onClick={load}>{c.tryAgain(l)}</button></section>}</>;
  const prelaunch = (text: string) => <div className="live-note-rule"><p><strong>{m.prelaunch(l)}</strong> {text}</p></div>;
  if (view === "account") {
    return (
      <div className="live-users" data-live="account" data-wide={wide ? "true" : "false"}>
        <div className="live-list live-sections">
          <section aria-labelledby="mysec-account-title">
            {toolbar("mysec-account-title", m.accountTitle(l), null)}
            {loadState}
            {page && (
              <>
                <Facts items={[
                  [c.signInEmail(l), ltr(page.login)],
                  [c.contactEmail(l), page.email ? ltr(page.email) : m.notRecorded(l)],
                  [m.userId(l), <><span className="live-mono">{ltr(page.user_id)}</span><span className="live-sub">{m.userIdSub(l)}</span></>],
                  [m.password(l), page.password_set ? m.set(l) : m.notSet(l)],
                ]} />
                {prelaunch(m.profileNote(l))}
              </>
            )}
          </section>
          {page && !wide && <AccessFacts l={l} page={page} />}
        </div>
        {page && wide && <aside className="live-panel" aria-label={m.accessTitle(l)}><AccessFacts l={l} page={page} /></aside>}
      </div>
    );
  }
  const stepping = !wide && mode.kind !== "list";
  const back = () => { setMode({ kind: "list" }); requestAnimationFrame(() => document.querySelector<HTMLElement>(".mysec-password-open")?.focus()); };
  if (view === "methods") {
    const form = page && (
      <PasswordForm key={formKey} l={l} api={api} page={page} stepping={stepping} hadWrong={hadWrong} pending={pending} onWrong={() => { if (guard.token.active) setHadWrong(true); }}
        onDone={() => { if (!guard.token.active) return;say(m.changedTitle(l)); setHadWrong(false); load(); }} onSignedOut={(locked) => setSignedOut(locked ? "locked" : "plain")}
        onCancel={() => { if (!wide) back(); else { setFormKey((k) => k + 1); requestAnimationFrame(() => document.getElementById("mysec-current")?.focus()); } }} />
    );
    return (
      <div className="live-users" data-live="methods" data-wide={wide ? "true" : "false"}>
        {announcer}
        {!stepping && (
          <div className="live-list live-sections">
            <section aria-labelledby="mysec-methods-title">
              {toolbar("mysec-methods-title", m.methodsTitle(l), null,
                !wide && page && <button type="button" className="live-primary mysec-password-open" onClick={() => setMode({ kind: "password" })}>{m.changeTitle(l)}</button>)}
              {loadState}
              {page && (
                <>
                  <div className="security-table-scroll live-table">
                    <table role="table">
                      <thead role="rowgroup"><tr role="row">
                        <th scope="col" role="columnheader">{m.colMethod(l)}</th>
                        <th scope="col" role="columnheader">{m.colSignsIn(l)}</th>
                        <th scope="col" role="columnheader">{m.colStatus(l)}</th>
                      </tr></thead>
                      <tbody role="rowgroup">
                        {page.identities.map((i) => (
                          <tr key={i.method + i.login} role="row">
                            <td role="cell" className="live-user">{m.password(l)}</td>
                            <td role="cell" data-label={m.colSignsIn(l)}>{ltr(i.login)}</td>
                            <td role="cell" data-label={m.colStatus(l)}>{page.password_set ? m.pwSet(l) : m.pwNotSet(l)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                  {prelaunch(m.linkNote(l))}
                </>
              )}
            </section>
          </div>
        )}
        {page && (wide || stepping) && (
          <aside className="live-panel" aria-label={m.changeTitle(l)}>
            {stepping && <button type="button" className="service-button-link live-back" onClick={back}>{m.backMethods(l)}</button>}
            {form}
          </aside>
        )}
      </div>
    );
  }
  return <>{loadState}{page && <LiveSessions key={JSON.stringify([session.user,session.workspace,page.current_session.handle])} l={l} api={api} self={page} params={params} onParams={onParams} onRefresh={load} onSignOut={onSignOut} onSignedOut={() => { setSignedOut("plain");setPage(null); }} />}</>;
}

function AccessFacts({ l, page }: { l: Locale; page: SelfPage }) {
  return (
    <section aria-labelledby="mysec-access-title">
      <h2 id="mysec-access-title" className="live-title">{m.accessTitle(l)}</h2>
      <Facts items={[
        [m.workspace(l), ltr(page.workspace)],
        [m.role(l), roleName(l, page.role)],
        [m.installAdmin(l), page.installation_administrator ? m.yes(l) : m.no(l)],
      ]} />
      <p className="live-note">{m.roleNote(l)}</p>
    </section>
  );
}
