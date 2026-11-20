import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type FormEvent, type ReactNode } from "react";
import type { Locale } from "../shared/types";
import HeaderSelect from "../app/HeaderSelect";
import logo from "../../../../packages/design-system/assets/logo/hb-lockup-color.svg";
import { c, retryPhrase } from "./copy";
import { fragmentToken, serviceClient, ServiceError, type LinkInfo, type ServiceClient, type Session } from "./api";
import { ServiceContext } from "./context";
import { Reauthenticate } from "./Reauthenticate";
import { stepUpGate } from "./reauth-model";
import { leaveDeliberately, stayAgain, watchRestore } from "../shared/unload-intent";
import "./service.css";
import { localPreviewLogin } from "./local-preview";
import { afterSignIn, emailValid, localeOf, loginValid, passwordProblem, serviceUrl, shellLocale } from "./helpers";
export { serviceMode } from "./helpers";

export function Notice({ locale: l }: { locale: Locale }) {
  return (
    <div className="prelaunch-notice" role="note">
      <strong>{c.noticeLead(l)}</strong> {c.notice(l)}
    </div>
  );
}

function Frame({ locale: l, children, onLocale }: { locale: Locale; children: ReactNode; onLocale: (l: Locale) => void }) {
  return (
    <div className="entry-shell service-entry" lang={l} dir={l === "ar" ? "rtl" : "ltr"}>
      <a className="skip" href="#main" onClick={(e) => { e.preventDefault(); document.getElementById("main")?.focus(); }}>
        {c.skip(l)}
      </a>
      <header className="entry-header">
        <a className="brand" dir="ltr" href={serviceUrl("signin", l)}>
          <img className="brand-logo" src={logo} width="155" height="32" alt="Hybrid Billing" />
        </a>
        <HeaderSelect
          label={c.language(l)}
          value={l}
          options={[
            { value: "en", label: "English (US)" },
            { value: "ko", label: "한국어" },
            { value: "ar", label: "العربية (الإمارات)" },
          ]}
          onChange={(v) => onLocale(v as Locale)}
        />
      </header>
      <Notice locale={l} />
      <main id="main" className="service-main" tabIndex={-1}>
        {children}
      </main>
      <footer className="entry-footer service-footer">
        <span className="service-footer-group">
          <span className="entry-footer-brand">Hybrid Billing</span>
          <span className="entry-environment">{c.prelaunch(l)}</span>
        </span>
        <span className="service-footer-group">
          <a href={serviceUrl("signin", l)}>{c.signIn(l)}</a>
          <a href={serviceUrl("help", l)}>{c.helpSignIn(l)}</a>
        </span>
      </footer>
    </div>
  );
}

export function PasswordField({
  locale: l, id, label, value, onChange, autoComplete, hint, error,
}: {
  locale: Locale; id: string; label: string; value: string; onChange: (v: string) => void;
  autoComplete: string; hint?: string; error?: string;
}) {
  const [shown, setShown] = useState(false);
  const described = [hint ? id + "-hint" : "", error ? id + "-error" : ""].filter(Boolean).join(" ") || undefined;
  return (
    <div className="service-field">
      <label htmlFor={id}>{label}</label>
      {hint && <p className="service-hint" id={id + "-hint"}>{hint}</p>}
      {error && (
        <p className="service-field-error" id={id + "-error"}>
          <span className="visually-hidden">{c.errorPrefix(l)} </span>
          {error}
        </p>
      )}
      <div className="service-password">
        <input
          id={id} type={shown ? "text" : "password"} value={value} dir="ltr"
          autoComplete={autoComplete} spellCheck={false} autoCapitalize="none"
          aria-invalid={error ? true : undefined} aria-describedby={described}
          onChange={(e) => onChange(e.target.value)}
        />
        <button
          type="button" className="service-toggle"
          aria-label={shown ? c.hideName(l, label) : c.showName(l, label)}
          onClick={() => setShown((s) => !s)}
        >
          {shown ? c.hide(l) : c.show(l)}
        </button>
      </div>
      <span className="visually-hidden" role="status">{shown ? c.visible(l, label) : c.hidden(l, label)}</span>
    </div>
  );
}

export function TextField({
  id, label, value, onChange, type = "text", autoComplete, error, locale: l,
}: {
  id: string; label: string; value: string; onChange: (v: string) => void; type?: string;
  autoComplete: string; error?: string; locale: Locale;
}) {
  return (
    <div className="service-field">
      <label htmlFor={id}>{label}</label>
      {error && (
        <p className="service-field-error" id={id + "-error"}>
          <span className="visually-hidden">{c.errorPrefix(l)} </span>
          {error}
        </p>
      )}
      <input
        id={id} type={type} value={value} dir="ltr" autoComplete={autoComplete}
        spellCheck={false} autoCapitalize="none" inputMode={type === "email" ? "email" : undefined}
        aria-invalid={error ? true : undefined} aria-describedby={error ? id + "-error" : undefined}
        onChange={(e) => onChange(e.target.value)}
      />
    </div>
  );
}

export type Problems = Array<{ field: string; message: string }>;

export function Summary({ locale: l, title, problems, focusKey }: { locale: Locale; title: string; problems: Problems; focusKey: number }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => { if (focusKey) ref.current?.focus(); }, [focusKey]);
  return (
    <div className="service-summary" role="alert" tabIndex={-1} ref={ref}>
      <h2>{title}</h2>
      {problems.length > 0 && (
        <ul>
          {problems.map((p) => (
            <li key={p.field + p.message}>
              <a href={"#" + p.field} onClick={(e) => { e.preventDefault(); document.getElementById(p.field)?.focus(); }}>
                {p.message}
              </a>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function failureMessage(l: Locale, e: unknown, fallback = c.failed): string {
  if (e instanceof ServiceError) {
    if (e.failure === "limited") return c.limited(l, retryPhrase(l, e.retryAfter || 60));
    if (e.failure === "unavailable" || e.failure === "expired") return c.unavailable(l);
  }
  return fallback(l);
}

export function SignIn({ locale: l, client, status, onSession, notice }: {
  locale: Locale; client: ServiceClient; status: string; onSession: (s: Session) => void; notice?: ReactNode;
}) {
  const [workspace, setWorkspace] = useState(""), [email, setEmail] = useState(""), [password, setPassword] = useState("");
  const [problems, setProblems] = useState<Problems>([]), [message, setMessage] = useState(""), [busy, setBusy] = useState(false), [tick, setTick] = useState(0);
  const loginButton = useRef<HTMLButtonElement>(null);
  const preview = localPreviewLogin(typeof location === "undefined" ? "" : location.hostname, typeof __LOCAL_PREVIEW_LOGIN__ === "undefined" ? null : __LOCAL_PREVIEW_LOGIN__);
  const fillPreview = () => {
    if (!preview || busy) return;
    setWorkspace(preview.workspace);
    setEmail(preview.email);
    setPassword(preview.password);
    setProblems([]);
    setMessage("");
    loginButton.current?.focus();
  };
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (busy) return;
    const found: Problems = [];
    if (!workspace.trim()) found.push({ field: "signin-workspace", message: c.enterWorkspace(l) });
    if (!email.trim()) found.push({ field: "signin-email", message: c.enterEmail(l) });
    if (!password) found.push({ field: "signin-password", message: c.enterPassword(l) });
    setProblems(found);
    setMessage("");
    if (found.length) { setTick((n) => n + 1); return; }
    setBusy(true);
    try {
      onSession(await client.login(workspace.trim(), email.trim(), password));
    } catch (err) {
      setPassword("");
      setMessage(failureMessage(l, err));
      setTick((n) => n + 1);
    } finally {
      setBusy(false);
    }
  };
  const error = (f: string) => problems.find((p) => p.field === f)?.message;
  return (
    <div className="service-columns">
      <section className="service-task">
        {notice}
        {status === "signed-out" && <p className="service-status" role="status">{c.signedOut(l)}</p>}
        {status === "ended" && <p className="service-status" role="status">{c.ended(l)}</p>}
        <h1>{c.signIn(l)}</h1>
        <p className="service-lead">{c.lead(l)}</p>
        {(problems.length > 0 || message) && (
          <Summary locale={l} title={message || c.check(l)} problems={problems} focusKey={tick} />
        )}
        {preview && <button type="button" className="service-secondary" disabled={busy} onClick={fillPreview}>{c.fillLocalSample(l)}</button>}
        <form noValidate onSubmit={submit} aria-busy={busy}>
          <TextField locale={l} id="signin-workspace" label={c.workspace(l)} value={workspace} onChange={setWorkspace} autoComplete="off" error={error("signin-workspace")} />
          <TextField locale={l} id="signin-email" label={c.email(l)} value={email} onChange={setEmail} type="email" autoComplete="username" error={error("signin-email")} />
          <PasswordField locale={l} id="signin-password" label={c.password(l)} value={password} onChange={setPassword} autoComplete="current-password" error={error("signin-password")} />
          <button ref={loginButton} className="primary service-submit" type="submit" aria-disabled={busy}>
            {busy ? c.signingIn(l) : c.signIn(l)}
          </button>
        </form>
        <a className="service-secondary" href={serviceUrl("help", l)}>{c.cantSignIn(l)}</a>
      </section>
      <aside className="service-help">
        <h2>{c.helpTitle(l)}</h2>
        <p>{c.helpBody(l)}</p>
      </aside>
    </div>
  );
}

export function Help({ locale: l }: { locale: Locale }) {
  return (
    <div className="service-columns">
      <section className="service-task">
        <h1>{c.helpSignIn(l)}</h1>
        <div className="service-sections">
          <section><h2>{c.forgot(l)}</h2><p>{c.forgotBody(l)}</p></section>
          <section><h2>{c.attempts(l)}</h2><p>{c.attemptsBody(l)}</p></section>
          <section><h2>{c.restart(l)}</h2><p>{c.restartBody(l)}</p></section>
        </div>
        <a className="primary service-submit service-button-link" href={serviceUrl("signin", l)}>{c.backSignIn(l)}</a>
      </section>
      <aside className="service-help"><h2>{c.recoveryTitle(l)}</h2><p>{c.selfService(l)}</p></aside>
    </div>
  );
}

type LinkState =
  | { kind: "checking" }
  | { kind: "invalid" }
  | { kind: "retry"; error: unknown }
  | { kind: "ready"; info: LinkInfo }
  | { kind: "done"; info: LinkInfo };

export function Activate({ locale: l, client, token }: { locale: Locale; client: ServiceClient; token: string }) {
  const [state, setState] = useState<LinkState>(token ? { kind: "checking" } : { kind: "invalid" });
  const [login, setLogin] = useState(""), [contact, setContact] = useState(""), [first, setFirst] = useState(""), [second, setSecond] = useState("");
  const [problems, setProblems] = useState<Problems>([]), [message, setMessage] = useState(""), [busy, setBusy] = useState(false), [tick, setTick] = useState(0);
  const heading = useRef<HTMLHeadingElement>(null);
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    if (!token) return;
    let live = true;
    client.peek(token).then(
      (info) => live && setState({ kind: "ready", info }),
      (e) => {
        if (!live) return;
        if (e instanceof ServiceError && e.failure === "link") setState({ kind: "invalid" });
        else setState({ kind: "retry", error: e });
      },
    );
    return () => { live = false; };
  }, [client, token, attempt]);
  useEffect(() => { if (state.kind === "done") heading.current?.focus(); }, [state.kind]);
  if (state.kind === "checking") return <div className="service-columns"><section className="service-task"><h1 role="status">{c.checking(l)}</h1></section></div>;
  if (state.kind === "retry")
    return (
      <div className="service-columns">
        <section className="service-task">
          <h1>{c.checkFailed(l)}</h1>
          <p className="service-lead" role="alert">{failureMessage(l, state.error)}</p>
          <button type="button" className="primary service-submit" onClick={() => { setState({ kind: "checking" }); setAttempt((n) => n + 1); }}>
            {c.tryAgain(l)}
          </button>
        </section>
      </div>
    );
  if (state.kind === "invalid")
    return (
      <div className="service-columns">
        <section className="service-task">
          <h1>{c.linkTitle(l)}</h1>
          <p className="service-lead">{c.linkBody(l)}</p>
          <a className="primary service-submit service-button-link" href={serviceUrl("signin", l)}>{c.goSignIn(l)}</a>
        </section>
      </div>
    );
  const info = state.info;
  const titles = { set: [c.setTitle, c.setDone], reset: [c.resetTitle, c.resetDone], activate: [c.activateTitle, c.activateDone] }[info.purpose];
  if (state.kind === "done")
    return (
      <div className="service-columns">
        <section className="service-task">
          <h1 tabIndex={-1} ref={heading}>{titles[1](l)}</h1>
          <p className="service-lead">{c.doneBody(l)}</p>
          <a className="primary service-submit service-button-link" href={serviceUrl("signin", l)}>{c.signIn(l)}</a>
        </section>
      </div>
    );
  const account = info.purpose === "activate" ? login : info.login;
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (busy) return;
    const found: Problems = [];
    if (info.purpose === "activate") {
      if (!login.trim()) found.push({ field: "link-login", message: c.enterSignInEmail(l) });
      else if (!loginValid(login.trim())) found.push({ field: "link-login", message: c.badSignInEmail(l) });
      if (!contact.trim()) found.push({ field: "link-contact", message: c.enterContact(l) });
      else if (!emailValid(contact.trim())) found.push({ field: "link-contact", message: c.badContact(l) });
    }
    const p = passwordProblem(l, first, account.trim());
    if (p) found.push({ field: "link-new", message: p });
    if (!second) found.push({ field: "link-confirm", message: c.enterAgain(l) });
    else if (first !== second) found.push({ field: "link-confirm", message: c.mismatch(l) });
    setProblems(found);
    setMessage("");
    if (found.length) { setFirst(""); setSecond(""); setTick((n) => n + 1); return; }
    setBusy(true);
    try {
      await client.redeem(token, first, info.purpose === "activate" ? { login: login.trim(), email: contact.trim() } : undefined);
      setFirst("");
      setSecond("");
      setState({ kind: "done", info });
    } catch (err) {
      setFirst("");
      setSecond("");
      if (err instanceof ServiceError && err.failure === "link") setState({ kind: "invalid" });
      else { setMessage(failureMessage(l, err, c.notCompleted)); setTick((n) => n + 1); }
    } finally {
      setBusy(false);
    }
  };
  const error = (f: string) => problems.find((p) => p.field === f)?.message;
  return (
    <div className="service-columns">
      <section className="service-task">
        <h1>{titles[0](l)}</h1>
        {info.purpose !== "activate" && (
          <dl className="service-account"><dt>{c.email(l)}</dt><dd><bdi dir="ltr">{info.login}</bdi></dd></dl>
        )}
        {(problems.length > 0 || message) && <Summary locale={l} title={message || c.check(l)} problems={problems} focusKey={tick} />}
        <form noValidate onSubmit={submit} aria-busy={busy}>
          {info.purpose !== "activate" && (
            <input className="visually-hidden" tabIndex={-1} aria-hidden="true" autoComplete="username" value={info.login} readOnly />
          )}
          {info.purpose === "activate" && (
            <>
              <TextField locale={l} id="link-login" label={c.signInEmail(l)} value={login} onChange={setLogin} type="email" autoComplete="username" error={error("link-login")} />
              <TextField locale={l} id="link-contact" label={c.contactEmail(l)} value={contact} onChange={setContact} type="email" autoComplete="email" error={error("link-contact")} />
            </>
          )}
          <PasswordField locale={l} id="link-new" label={c.newPassword(l)} value={first} onChange={setFirst} autoComplete="new-password" hint={c.rule(l)} error={error("link-new")} />
          <PasswordField locale={l} id="link-confirm" label={c.confirmPassword(l)} value={second} onChange={setSecond} autoComplete="new-password" error={error("link-confirm")} />
          <button className="primary service-submit" type="submit" aria-disabled={busy}>
            {busy ? c.saving(l) : info.purpose === "activate" ? c.activateButton(l) : c.setButton(l)}
          </button>
        </form>
      </section>
      <aside className="service-help"><h2>{c.aboutLink(l)}</h2><p>{c.aboutLinkBody(l)}</p></aside>
    </div>
  );
}

export function SessionStrip({ locale: l, session, onSignOut }: { locale: Locale; session: Session; onSignOut: () => void }) {
  return (
    <div className="service-session" lang={l} dir={l === "ar" ? "rtl" : "ltr"}>
      <Notice locale={l} />
      <div className="service-session-row">
        <span className="service-session-item"><span className="service-session-label">{c.workspace(l)}</span> <bdi dir="ltr">{session.workspace}</bdi></span>
        <span className="service-session-item service-session-user"><bdi dir="ltr">{session.user}</bdi></span>
        <button type="button" className="header-control service-signout" onClick={onSignOut}>{c.signOut(l)}</button>
      </div>
    </div>
  );
}

export function ServiceRoot({ app, client: given }: { app: ReactNode; client?: ServiceClient }) {
  const client = useMemo(() => given ?? serviceClient(), [given]);
  const [locale, setLocale] = useState<Locale>(() => localeOf(location.search));
  const [session, setSession] = useState<Session | null | undefined>(undefined);
  const [failed, setFailed] = useState(false);
  const [hash, setHash] = useState(location.hash);
  useEffect(() => {
    const update = () => setHash(location.hash);
    window.addEventListener("hashchange", update);
    return () => window.removeEventListener("hashchange", update);
  }, []);
  const params = new URLSearchParams(location.search);
  const activating = location.pathname === "/activate";
  const [token] = useState(() => fragmentToken(location.hash));
  const [status] = useState(() => params.get("status") ?? "");
  const [check, setCheck] = useState(0);
  const [prompt, setPrompt] = useState<{ opener: HTMLElement | null; n: number } | null>(null);
  const lastFocus = useRef<HTMLElement | null>(null);
  useEffect(() => {
    const seen = (e: FocusEvent) => { if (e.target instanceof HTMLElement && !e.target.closest("dialog.service-reauth")) lastFocus.current = e.target; };
    document.addEventListener("focusin", seen);
    return () => document.removeEventListener("focusin", seen);
  }, []);
  const gate = useMemo(() => stepUpGate(() => {
    const active = document.activeElement;
    const opener = active instanceof HTMLElement && active !== document.body ? active : lastFocus.current;
    setPrompt((p) => ({ opener, n: (p?.n ?? 0) + 1 }));
  }), []);
  const stepUp = useCallback(() => gate.request(), [gate]);
  useEffect(() => watchRestore(window), []);
  useEffect(() => { if (session) stayAgain(); }, [session]);
  useEffect(() => () => gate.settle(false), [gate]);
  useLayoutEffect(() => {
    if (activating && location.hash) history.replaceState(null, "", location.pathname + location.search);
    if (!activating && params.has("status")) {
      const q = new URLSearchParams(location.search);
      q.delete("status");
      history.replaceState(null, "", location.pathname + "?" + q.toString() + location.hash);
    }
  }, []);
  useEffect(() => {
    if (activating) return;
    client.current().then(
      (s) => { setFailed(false); setSession(s); },
      () => setFailed(true),
    );
  }, [client, activating, check]);
  const changeLocale = (v: Locale) => {
    const q = new URLSearchParams(location.search);
    q.set("lang", v);
    history.replaceState(null, "", location.pathname + "?" + q.toString() + location.hash);
    setLocale(v);
  };
  const signOutIn = async (l: Locale) => {
    if (session) await client.logout(session).catch(() => undefined);
    leaveDeliberately();
    location.assign(serviceUrl("signin", l, "signed-out"));
  };
  if (activating) {
    return <Frame locale={locale} onLocale={changeLocale}><Activate locale={locale} client={client} token={token} /></Frame>;
  }
  if (session) {
    const shown = shellLocale(hash, locale);
    const signOut = () => { setLocale(shown); void signOutIn(shown); };
    return (
      <ServiceContext.Provider value={{ session, signOut, stepUp }}>
        <SessionStrip locale={shown} session={session} onSignOut={signOut} />
        {app}
        {prompt && (
          <Reauthenticate
            key={prompt.n} locale={shown} login={session.login} csrf={session.csrf} opener={prompt.opener}
            onDone={(ok) => { setPrompt(null); gate.settle(ok); }}
            onSignedOut={() => { setPrompt(null); gate.settle(false); leaveDeliberately(); location.assign(serviceUrl("signin", shown, "ended")); }}
          />
        )}
      </ServiceContext.Provider>
    );
  }
  if (params.get("view") === "help") return <Frame locale={locale} onLocale={changeLocale}><Help locale={locale} /></Frame>;
  if (session === undefined && !failed) return <Frame locale={locale} onLocale={changeLocale}><p className="service-loading" role="status">{c.loading(locale)}</p></Frame>;
  return (
    <Frame locale={locale} onLocale={changeLocale}>
      <SignIn
        locale={locale} client={client} status={failed ? "" : status}
        notice={failed ? (
          <div className="service-status service-unavailable" role="alert" key={check}>
            <p>{c.unavailable(locale)}</p>
            <button type="button" className="service-retry" onClick={() => setCheck((n) => n + 1)}>{c.tryAgain(locale)}</button>
          </div>
        ) : undefined} onSession={(s) => { history.replaceState(null, "", "/?service=api&lang=" + locale + afterSignIn(location.hash, locale)); setHash(location.hash); setSession(s); }} />
    </Frame>
  );
}
