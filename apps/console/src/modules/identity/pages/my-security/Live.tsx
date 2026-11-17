import { useCallback, useEffect, useLayoutEffect, useRef, useState, type FormEvent, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, sequence, type AdminClient, type SelfPage, type SelfSession } from "../../../../service/admin";
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
import { canChange, changeOutcome, checkChange, endOutcome, others } from "./live-model";
import "../users/live.css";

type Mode = { kind: "list" } | { kind: "password" } | { kind: "end"; handle: string } | { kind: "endAll" };

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

function EndStep({ l, api, page, target, onDone, onCancel, onSignedOut }: {
  l: Locale; api: AdminClient; page: SelfPage; target: Mode; onDone: (message: string) => void; onCancel: () => void; onSignedOut: () => void;
}) {
  const heading = useRef<HTMLHeadingElement>(null);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState("");
  const [focusKey, setFocusKey] = useState(0);
  useEffect(() => heading.current?.focus(), []);
  const one = target.kind === "end" ? page.sessions.find((s) => s.handle === target.handle) : undefined;
  const count = others(page).length;
  const go = async () => {
    if (busy) return;
    setBusy(true);
    setFailure("");
    try {
      const n = await api.endSessions(target.kind === "end" ? target.handle : "others");
      onDone(m.ended(l, n));
    } catch (err) {
      const outcome = endOutcome(err);
      if (outcome === "signedOut") { onSignedOut(); return; }
      if (outcome === "gone") { onDone(m.endGone(l)); return; }
      setFailure(outcome === "limited" ? m.endLimited(l, err instanceof AdminError ? err.retryAfter : 0) : outcome === "reauth" ? c.reauthText(l, err instanceof AdminError ? err.failure : "") : m.unknownResult(l));
      setFocusKey((k) => k + 1);
    } finally {
      setBusy(false);
    }
  };
  const title = target.kind === "end" ? m.endOneTitle(l) : m.endAllTitle(l);
  return (
    <section className="live-step" aria-labelledby="mysec-end-title">
      <h2 id="mysec-end-title" tabIndex={-1} ref={heading}>{title}</h2>
      {failure && <FailureNote focusKey={focusKey} title={w.saveFailed(l)} body={failure} />}
      {one && <Facts items={[[m.reference(l), <span className="live-mono">{ltr(one.handle)}</span>], [m.colSignedIn(l), when(l, one.signed_in)], [m.colEnds(l), when(l, one.ends)]]} />}
      <p className="live-impact">{target.kind === "end" ? m.endOneImpact(l) : m.endAllImpact(l, count)}</p>
      <div className="live-buttons">
        <button type="button" className="live-primary" aria-busy={busy || undefined} disabled={busy} onClick={go}>{busy ? w.saving(l) : target.kind === "end" ? m.endOneButton(l) : m.endAllButton(l)}</button>
        <button type="button" className="live-secondary" onClick={onCancel}>{w.cancel(l)}</button>
      </div>
    </section>
  );
}

function SessionTable({ l, rows, selected, onEnd, onSignOut }: { l: Locale; rows: SelfSession[]; selected: string; onEnd: (s: SelfSession) => void; onSignOut: () => void }) {
  return (
    <div className="security-table-scroll live-table mysec-sessions">
      <table role="table">
        <colgroup><col className="mysec-col-a" /><col className="mysec-col-b" /><col className="mysec-col-c" /><col className="mysec-col-d" /></colgroup>
        <thead role="rowgroup"><tr role="row">
          <th scope="col" role="columnheader">{m.colSession(l)}</th>
          <th scope="col" role="columnheader">{m.colSignedIn(l)}</th>
          <th scope="col" role="columnheader">{m.colEnds(l)}</th>
          <th scope="col" role="columnheader"><span className="visually-hidden">{m.colActions(l)}</span></th>
        </tr></thead>
        <tbody role="rowgroup">
          {rows.map((s) => (
            <tr key={s.handle} role="row" className={selected === s.handle ? "live-selected" : undefined}>
              <td role="cell" className="live-user">{s.current ? m.thisSession(l) : m.otherSession(l)}
                <span className="live-sub">{m.reference(l)} <span className="live-mono">{ltr(s.handle)}</span></span></td>
              <td role="cell" data-label={m.colSignedIn(l)}>{when(l, s.signed_in)}</td>
              <td role="cell" data-label={m.colEnds(l)}>{when(l, s.ends)}</td>
              <td role="cell">
                {s.current
                  ? <button type="button" className="live-secondary live-revoke" onClick={onSignOut}>{m.signOut(l)}</button>
                  : <button type="button" className="live-secondary live-revoke" data-session={s.handle} onClick={() => onEnd(s)}>{m.end(l)}<span className="visually-hidden"> {m.reference(l)} {s.handle}</span></button>}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
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

export default function LiveMySecurity({ locale: l, view, session, onSignOut, onView }: {
  locale: Locale; view: string; session: Session; onSignOut: () => void; onView: (view: string) => void;
}) {
  const api = useAdminClient();
  const wide = useWide();
  const order = useRef(sequence());
  const [page, setPage] = useState<SelfPage | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState<unknown>(null);
  const [mode, setMode] = useState<Mode>({ kind: "list" });
  const [signedOut, setSignedOut] = useState<"" | "plain" | "locked">("");
  const [hadWrong, setHadWrong] = useState(false);
  const [notice, setNotice] = useState("");
  const [announce, setAnnounce] = useState("");
  const [focusTo, setFocusTo] = useState("");
  const [formKey, setFormKey] = useState(0);
  const pending = useRef(false);
  useEffect(() => {
    if (!focusTo) return;
    document.querySelector<HTMLElement>(focusTo)?.focus();
    setFocusTo("");
  }, [focusTo]);
  const say = (text: string) => { setNotice(text); setAnnounce(""); setTimeout(() => setAnnounce(text), 50); };
  const load = useCallback(() => {
    const n = order.current.next();
    api.me().then(
      (got) => { if (order.current.latest(n)) { setPage(got); setStatus("ready"); setError(null); } },
      (e) => {
        if (!order.current.latest(n)) return;
        if (e instanceof AdminError && e.failure === "unauthenticated") setSignedOut((s) => s || "plain");
        setError(e);
        setStatus("error");
      },
    );
  }, [api]);
  useEffect(() => { if (view !== "recovery" && view !== "notifications") load(); }, [load, view]);
  const viewRef = useRef(view);
  useEffect(() => {
    if (viewRef.current !== view) { setMode({ kind: "list" }); setNotice(""); if (!pending.current) setFormKey((k) => k + 1); }
    viewRef.current = view;
  }, [view]);
  useEffect(() => { if (!pending.current) setFormKey((k) => k + 1); }, [l]);
  if (signedOut) {
    return (
      <div className="live-users">
        <section className="live-unavailable" aria-labelledby="mysec-out">
          <h2 id="mysec-out">{m.signedOutTitle(l)}</h2>
          <p>{signedOut === "locked" ? m.lockedOutBody(l) : m.signedOutBody(l)}</p>
          <button type="button" className="live-primary" autoFocus onClick={onSignOut}>{w.signInAgain(l)}</button>
        </section>
      </div>
    );
  }
  if (view === "recovery") return <Unavailable title={m.recoveryTitle(l)} body={m.recoveryBody(l)} extra={m.recoveryForgot(l)} action={{ label: m.goAccount(l), go: () => onView("account") }} />;
  if (view === "notifications") return <Unavailable title={m.notifTitle(l)} body={m.notifBody(l)} action={{ label: m.goSessions(l), go: () => onView("sessions") }} />;
  const toolbar = (id: string, title: string, meta: ReactNode, tools?: ReactNode) => (
    <div className="live-toolbar">
      <div className="live-heading">
        <h2 id={id} tabIndex={-1}>{title}</h2>
        {meta}
        {page && <p className="live-meta">{w.loadedAt(l, when(l, page.now))}</p>}
      </div>
      <div className="live-tools">
        {tools}
        <button type="button" className="live-secondary" onClick={() => { setNotice(""); load(); }}>{w.reload(l)}</button>
      </div>
    </div>
  );
  const loadState = (
    <>
      {status === "loading" && !page && <p role="status" className="live-meta">{m.loading(l)}</p>}
      {status === "error" && (
        <section className="live-error" role="alert">
          <h2>{m.loadFailed(l)}</h2>
          <p>{m.loadRetry(l)}</p>
          <button type="button" className="live-secondary" onClick={load}>{c.tryAgain(l)}</button>
        </section>
      )}
    </>
  );
  const announcer = <p className="visually-hidden" role="status" aria-live="polite">{announce}</p>;
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
  const backTo = mode.kind === "end" ? `[data-session="${mode.handle}"]` : mode.kind === "endAll" ? ".mysec-end-others" : ".mysec-password-open";
  const back = () => { setMode({ kind: "list" }); setFocusTo(backTo); };
  if (view === "methods") {
    const form = page && (
      <PasswordForm key={formKey} l={l} api={api} page={page} stepping={stepping} hadWrong={hadWrong} pending={pending} onWrong={() => setHadWrong(true)}
        onDone={() => { say(m.changedTitle(l)); setHadWrong(false); load(); }} onSignedOut={(locked) => setSignedOut(locked ? "locked" : "plain")}
        onCancel={() => { if (!wide) back(); else setFormKey((k) => k + 1); }} />
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
  const rows = page?.sessions ?? [];
  const selected = mode.kind === "end" ? mode.handle : "";
  const finishEnd = (message: string) => { say(message); setMode({ kind: "list" }); load(); setFocusTo("#mysec-sessions-title"); };
  const step = page && (mode.kind === "end" || mode.kind === "endAll")
    ? <EndStep key={mode.kind === "end" ? mode.handle : "all"} l={l} api={api} page={page} target={mode} onDone={finishEnd} onCancel={back} onSignedOut={() => setSignedOut("plain")} />
    : null;
  const limits = page && (
    <section aria-labelledby="mysec-limits-title">
      <h2 id="mysec-limits-title" className="live-title">{m.limitsTitle(l)}</h2>
      <Facts items={[
        [m.idle(l), m.minutes(l, page.idle_minutes)],
        [m.longest(l), m.longestValue(l)],
        [m.maxAge(l), page.max_age_seconds > 0 ? m.age(l, page.max_age_seconds) : m.noMaxAge(l)],
        [m.youSignedIn(l), when(l, page.signed_in_at)],
        [m.lastConfirmed(l), when(l, page.authenticated_at)],
      ]} />
      {prelaunch(m.historyNote(l))}
    </section>
  );
  return (
    <div className="live-users" data-live="sessions" data-wide={wide ? "true" : "false"}>
      {announcer}
      {!stepping && (
        <div className="live-list live-sections">
          <section aria-labelledby="mysec-sessions-title">
            {toolbar("mysec-sessions-title", m.sessionsTitle(l), page && <p className="live-meta">{m.count(l, rows.length)}</p>,
              page && others(page).length > 0 && <button type="button" className="live-secondary mysec-end-others" onClick={() => { setNotice(""); setMode({ kind: "endAll" }); }}>{m.endOthers(l)}</button>)}
            {notice && <p className="service-status">{notice}</p>}
            {loadState}
            {page && (
              <>
                {prelaunch(m.staleNote(l, page.idle_minutes))}
                <SessionTable l={l} rows={rows} selected={selected} onSignOut={onSignOut} onEnd={(s) => { setNotice(""); setMode({ kind: "end", handle: s.handle }); }} />
                {others(page).length === 0 && <p className="live-empty">{m.noOthers(l)}</p>}
              </>
            )}
          </section>
          {!wide && limits}
        </div>
      )}
      {page && wide && <aside className="live-panel" aria-label={m.limitsTitle(l)}>{step ?? limits}</aside>}
      {stepping && step && (
        <aside className="live-panel" aria-label={m.sessionsTitle(l)}>
          <button type="button" className="service-button-link live-back" onClick={back}>{m.backSessions(l)}</button>
          {step}
        </aside>
      )}
    </div>
  );
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
