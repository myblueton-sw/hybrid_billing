import { useCallback, useEffect, useRef, useState, type FormEvent, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, attempt, sequence, type AccessGrant, type AccessPage, type AdminClient, type Attempt, type Preview } from "../../../../service/admin";
import { useAdminClient } from "../../../../service/context";
import type { Session } from "../../../../service/api";
import { c } from "../../../../service/copy";
import { Summary, type Problems } from "../../../../service/Service";
import { FailureBox, Reason, reasonProblem, settles, useWide } from "../users/Live";
import { w } from "../users/live-copy";
import { failureText } from "../users/live-model";
import { when } from "../authentication/live-model";
import { a, accessPhrase, actionLabel, fieldLabel, kindLabel } from "./live-copy";
import { accessText, ceilingGrants, delegationsOf, maxDays, people, reasonText, revocable, scopeText, stateLine, wholeDays } from "./live-model";
import { q, v } from "./request-copy";
import { additionsPaused, descendantsOf, keepAttempt, pendingRevokeFor, unconfirmedFor } from "./request-model";
import { DecidePanel, describeRequest, RequestSection, type Decision } from "./Requests";
import { breakable, loginOf } from "./who";
import { RevocationSection } from "./Revocations";
import "../users/live.css";
import "./access.css";

type Target = { kind: "grant"; row: AccessGrant } | { kind: "delegation"; id: string; who: string; access: string; until: string };
type Mode = { kind: "form" } | { kind: "revoke"; target: Target } | { kind: "decide"; action: Decision; id: string };

function describeFor(kind: "grant" | "revoke" | "load") {
  return (l: Locale, e: unknown) => {
    if (kind === "grant") return describeRequest(l, e);
    if (e instanceof AdminError && kind !== "load") {
      if (e.failure === "refused") return { title: a.refusedTitle(l), body: a.refusedRevoke(l) };
      if (e.failure === "stage") return { title: a.refusedTitle(l), body: a.stageBody(l) };
      if (e.failure === "invalid") return { title: a.refusedTitle(l), body: a.invalidBody(l) };
      if (e.failure === "denied" && kind === "revoke") return { title: a.refusedTitle(l), body: a.goneBody(l) };
      if (e.failure === "conflict" || e.failure === "stale") return { title: w.saveFailed(l), body: a.conflictBody(l) };
    }
    if (e instanceof AdminError && (e.failure === "unavailable" || e.failure === "busy")) return { title: w.saveFailed(l), body: a.unavailableBody(l) };
    return failureText(l, e);
  };
}

function GrantForm({ l, api, page, self, onDone, onCancel, onReload, stepping }: {
  l: Locale; api: AdminClient; page: AccessPage; self: string; onDone: (login: string) => void; onCancel: () => void; onReload: () => void; stepping: boolean;
}) {
  const pair = page.catalog[0];
  const choices = page.subjects.filter((s) => s.user_id !== self);
  const most = pair ? maxDays(pair.until, page.now) : 0;
  const [subject, setSubject] = useState("");
  const [days, setDays] = useState(most > 0 ? String(Math.min(30, most)) : "");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [problems, setProblems] = useState<Problems>([]);
  const [error, setError] = useState<unknown>(null);
  const [focusKey, setFocusKey] = useState(0);
  const tried = useRef<Attempt>(attempt());
  const kept = useRef(false);
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => {
    if (!kept.current) tried.current = attempt();
    kept.current = false;
  }, [page.revision]);
  useEffect(() => { if (stepping) heading.current?.focus(); }, [stepping]);
  const paused = additionsPaused(page.ledger);
  const edit = (f: (v: string) => void) => (v: string) => { tried.current = attempt(); f(v); };
  if (!pair || most < 1) return null;
  const n = wholeDays(days, most);
  const login = choices.find((s) => s.user_id === subject)?.login ?? "";
  const phrase = accessPhrase(l, pair.action, pair.kind, pair.fields) ?? accessText(l, pair.action, pair.fields);
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    const found: Problems = [];
    if (!subject) found.push({ field: "access-person", message: a.errPerson(l) });
    if (n === null) found.push({ field: "access-days", message: a.errDays(l, most) });
    const r = reasonProblem(l, reason);
    if (r) found.push({ field: "live-reason", message: r });
    setProblems(found);
    setError(null);
    if (found.length) { setFocusKey((k) => k + 1); return; }
    setBusy(true);
    try {
      await api.command("access.request_grant", "grants", page.revision, reason,
        { subject, action: pair.action, kind: pair.kind, fields: pair.fields.join(","), customer: "all", days: String(n) }, tried.current);
      onDone(login);
    } catch (err) {
      if (err instanceof AdminError && err.failure === "invalid" && err.fields.includes("days")) setProblems([{ field: "access-days", message: a.errDays(l, most) }]);
      else setError(err);
      kept.current = keepAttempt(err);
      if (err instanceof AdminError && err.failure === "stale") tried.current = attempt();
      if (settles(err)) onReload();
      setFocusKey((k) => k + 1);
    } finally {
      setBusy(false);
    }
  };
  const field = (id: string) => problems.find((x) => x.field === id)?.message;
  return (
    <form className="live-step" onSubmit={submit} noValidate>
      <h2 tabIndex={-1} ref={heading}>{q.requestTitle(l)}</h2>
      <p className="live-impact">{q.requestLead(l)}</p>
      {paused && <p className="live-strong" id="access-paused">{q.additionsPaused(l)}</p>}
      {problems.length > 0 && <Summary locale={l} title={c.check(l)} problems={problems} focusKey={focusKey} />}
      {error !== null && <FailureBox l={l} error={error} focusKey={focusKey} describe={describeFor("grant")} />}
      {choices.length === 0 ? <p className="live-note">{a.noneEligible(l)}</p> : (
        <div className="service-field">
          <label htmlFor="access-person">{a.person(l)}</label>
          {field("access-person") && <p className="service-field-error" id="access-person-error"><span className="visually-hidden">{c.errorPrefix(l)} </span>{field("access-person")}</p>}
          <select id="access-person" className="live-select" value={subject} aria-invalid={field("access-person") ? true : undefined}
            aria-describedby={field("access-person") ? "access-person-error" : undefined} onChange={(e) => edit(setSubject)(e.target.value)}>
            <option value="">{a.choosePerson(l)}</option>
            {choices.map((s) => <option key={s.user_id} value={s.user_id}>{s.login || s.user_id}</option>)}
          </select>
        </div>
      )}
      <dl className="live-facts">
        <div><dt>{a.access(l)}</dt><dd>{accessText(l, pair.action, pair.fields)} · {kindLabel(l, pair.kind) ?? pair.kind}</dd><dd className="live-note">{a.onlyPair(l)}</dd></div>
        <div><dt>{a.customer(l)}</dt><dd>{a.allCustomers(l)}</dd>{pair.kind === "focus_data" && <dd className="live-note">{a.focusAll(l)}</dd>}</div>
      </dl>
      <div className="service-field live-minutes">
        <label htmlFor="access-days">{a.duration(l)}</label>
        <p className="service-hint" id="access-days-hint">{a.daysHint(l, most, when(l, pair.until))}</p>
        {field("access-days") && <p className="service-field-error" id="access-days-error"><span className="visually-hidden">{c.errorPrefix(l)} </span>{field("access-days")}</p>}
        <span className="live-unit">
          <input id="access-days" inputMode="numeric" dir="ltr" value={days} aria-invalid={field("access-days") ? true : undefined}
            aria-describedby={["access-days-hint", field("access-days") ? "access-days-error" : ""].filter(Boolean).join(" ")} onChange={(e) => edit(setDays)(e.target.value)} />
          <span>{a.days(l)}</span>
        </span>
      </div>
      {login && n && <p className="live-impact">{q.requestImpact(l, login, phrase, n)}</p>}
      <Reason l={l} value={reason} onChange={edit(setReason)} error={field("live-reason") ?? ""} />
      <div className="live-buttons">
        <button type="submit" className={paused ? "live-primary access-paused" : "live-primary"} aria-busy={busy || undefined} aria-describedby={paused ? "access-paused" : undefined}
          disabled={busy || choices.length === 0 || paused}>{busy ? w.saving(l) : q.requestTitle(l)}</button>
        {stepping && <button type="button" className="live-secondary" onClick={onCancel}>{w.cancel(l)}</button>}
      </div>
    </form>
  );
}

function Revoke({ l, api, page, target, onDone, onCancel, onReload }: {
  l: Locale; api: AdminClient; page: AccessPage; target: Target; onDone: (message: string) => void; onCancel: () => void; onReload: () => void;
}) {
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [problem, setProblem] = useState("");
  const [focusKey, setFocusKey] = useState(0);
  const tried = useRef<Attempt>(attempt());
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => heading.current?.focus(), []);
  useEffect(() => { tried.current = attempt(); }, [page.revision]);
  const grant = target.kind === "grant";
  const who = grant ? loginOf(page, target.row.subject) : target.who;
  const id = grant ? target.row.id : target.id;
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    const r = reasonProblem(l, reason);
    setProblem(r);
    if (r) { setError(null); setFocusKey((k) => k + 1); return; }
    setBusy(true);
    setError(null);
    try {
      await api.command(grant ? "access.request_revoke" : "access.revoke_delegation", id, page.revision, reason, {}, tried.current);
      onDone(grant ? q.revocationRequested(l) : q.doneRevokedDeleg(l));
    } catch (err) {
      setError(err);
      setFocusKey((k) => k + 1);
      if (err instanceof AdminError && err.failure === "stale") tried.current = attempt();
      if (settles(err)) onReload();
    } finally {
      setBusy(false);
    }
  };
  const title = grant ? q.requestRevocation(l) : a.revokeDelegTitle(l);
  const below = grant ? [] : descendantsOf(page, id);
  return (
    <form className="live-step" onSubmit={submit} noValidate>
      <h2 tabIndex={-1} ref={heading}>{title}</h2>
      <p className="live-who"><bdi dir="ltr">{who}</bdi></p>
      {problem && <Summary locale={l} title={c.check(l)} problems={[{ field: "live-reason", message: problem }]} focusKey={focusKey} />}
      {error !== null && <FailureBox l={l} error={error} focusKey={focusKey} describe={grant ? describeRequest : describeFor("revoke")} />}
      <dl className="live-facts">
        <div><dt>{a.access(l)}</dt><dd>{grant ? `${accessText(l, target.row.action, target.row.fields)} · ${kindLabel(l, target.row.kind) ?? target.row.kind}` : target.access}</dd></div>
        {grant && <div><dt>{a.customer(l)}</dt><dd>{target.row.scope.customer === "all" ? a.allCustomers(l) : target.row.scope.customer}</dd></div>}
        <div><dt>{a.colStatus(l)}</dt><dd>{grant ? stateLine(l, target.row).join(" · ") : target.until}</dd></div>
      </dl>
      <p className="live-impact">{grant ? q.revokeRequestLead(l, who) : a.revokeDelegImpact(l, who)}</p>
      {!grant && (
        <div className="access-impact-list">
          <h3>{q.descendants(l)}</h3>
          {below.length === 0 ? <p className="live-note">{q.noDescendants(l)}</p> : <p className="live-mono" dir="ltr">{below.join(", ")}</p>}
        </div>
      )}
      <Reason l={l} value={reason} onChange={(v) => { tried.current = attempt(); setReason(v); }} error={problem} />
      <div className="live-buttons">
        <button type="submit" className="live-primary" aria-busy={busy || undefined} disabled={busy}>{busy ? w.saving(l) : title}</button>
        <button type="button" className="live-secondary" onClick={onCancel}>{w.cancel(l)}</button>
      </div>
    </form>
  );
}

function GrantTable({ l, page, rows, self, selected, onRevoke }: {
  l: Locale; page: AccessPage; rows: AccessGrant[]; self: string; selected: string; onRevoke?: (row: AccessGrant) => void;
}) {
  return (
    <div className="security-table-scroll live-table live-access access-grants">
      <table role="table">
        <colgroup><col className="live-col-a" /><col className="live-col-b" /><col className="live-col-c" />{onRevoke && <col className="live-col-d" />}</colgroup>
        <thead role="rowgroup"><tr role="row">
          <th scope="col" role="columnheader">{a.colPerson(l)}</th>
          <th scope="col" role="columnheader">{a.colAccess(l)}</th>
          <th scope="col" role="columnheader">{a.colStatus(l)}</th>
          {onRevoke && <th scope="col" role="columnheader"><span className="visually-hidden">{a.colActions(l)}</span></th>}
        </tr></thead>
        <tbody role="rowgroup">
          {rows.map((g) => {
            const [state, sub] = stateLine(l, g);
            const asked = pendingRevokeFor(page, g.id);
            return (
              <tr key={g.id} role="row" className={selected === g.id ? "live-selected" : undefined}>
                <td role="cell" className="live-user"><bdi dir="ltr">{breakable(loginOf(page, g.subject))}</bdi></td>
                <td role="cell" data-label={a.colAccess(l)}>{accessText(l, g.action, g.fields)}<span className="live-sub">{scopeText(l, g.kind, g.scope.customer)}</span></td>
                <td role="cell" data-label={a.colStatus(l)} data-state={g.state} className={g.effective === false || g.state !== "active" ? "live-strong" : ""}>{state}<span className="live-sub">{sub}</span></td>
                {onRevoke && (
                  <td role="cell">
                    {g.subject === self ? <span className="live-quiet">{a.yourGrant(l)}</span>
                      : asked ? <span className="live-quiet" data-requested={g.id}>{q.revocationRequested(l)}</span>
                      : !revocable(g.state) ? null
                      : <button type="button" className="live-secondary live-revoke" data-grant={g.id} onClick={() => onRevoke(g)}>{q.requestRevocation(l)}<span className="visually-hidden"> {loginOf(page, g.subject)}</span></button>}
                  </td>
                )}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function Effects({ l, api, page }: { l: Locale; api: AdminClient; page: AccessPage }) {
  const kinds = Array.from(new Set(page.catalog.map((x) => x.kind)));
  const [subject, setSubject] = useState("");
  const [kind, setKind] = useState(kinds[0] ?? "");
  const [customer, setCustomer] = useState("");
  const [result, setResult] = useState<{ rows: Preview[]; at: string; login: string; customer: string; kind: string } | null>(null);
  const [state, setState] = useState<"start" | "cleared" | "loading" | "error" | "shown">("start");
  const [error, setError] = useState<unknown>(null);
  const order = useRef(sequence());
  const change = (f: (v: string) => void) => (v: string) => { order.current.next(); f(v); setResult(null); setState((s) => (s === "start" ? "start" : "cleared")); };
  const run = async (e: FormEvent) => {
    e.preventDefault();
    if (!subject || !kind) return;
    const n = order.current.next();
    setState("loading");
    try {
      const rows = await api.preview(subject, kind, customer.trim());
      if (!order.current.latest(n)) return;
      setResult({ rows, at: new Date().toISOString(), login: loginOf(page, subject), customer: customer.trim(), kind });
      setState("shown");
    } catch (err) {
      if (!order.current.latest(n)) return;
      setError(err);
      setState("error");
    }
  };
  let last = "";
  return (
    <div className="live-users live-effects" data-live="effects">
      <form className="live-effects-form" onSubmit={run} noValidate>
        <div className="service-field">
          <label htmlFor="effect-person">{a.person(l)}</label>
          <select id="effect-person" className="live-select" value={subject} onChange={(e) => change(setSubject)(e.target.value)}>
            <option value="">{a.choosePerson(l)}</option>
            {page.subjects.map((s) => <option key={s.user_id} value={s.user_id}>{s.login || s.user_id}</option>)}
          </select>
        </div>
        <div className="service-field">
          {kinds.length > 1 ? (
            <>
              <label htmlFor="effect-kind">{a.data(l)}</label>
              <select id="effect-kind" className="live-select" value={kind} onChange={(e) => change(setKind)(e.target.value)}>
                {kinds.map((k) => <option key={k} value={k}>{kindLabel(l, k) ?? k}</option>)}
              </select>
            </>
          ) : (
            <>
              <span className="live-fact-label">{a.data(l)}</span>
              <span className="live-fact-box">{kindLabel(l, kind) ?? kind}</span>
            </>
          )}
        </div>
        <div className="service-field">
          <label htmlFor="effect-customer">{a.customerOpt(l)}</label>
          <input id="effect-customer" className="live-mono live-customer" dir="ltr" value={customer} aria-describedby="effect-customer-hint" onChange={(e) => change(setCustomer)(e.target.value)} />
          <p className="service-hint live-hint-below" id="effect-customer-hint">{a.customerOptHint(l)}</p>
        </div>
        <button type="submit" className="live-primary" disabled={!subject || state === "loading"}>{a.showEffect(l)}</button>
      </form>
      <p className="live-preview-note"><strong>{a.previewLead(l)}</strong> {a.previewNote(l)}</p>
      {subject && unconfirmedFor(page, subject) && <p className="live-strong" data-unconfirmed="true">{v.previewUnconfirmed(l)}</p>}
      {state === "start" && <p className="live-empty">{a.startHint(l)}</p>}
      {state === "cleared" && <p className="live-empty">{a.cleared(l)}</p>}
      <p className="visually-hidden" role="status" aria-live="polite">{state === "loading" ? a.loading(l) : state === "shown" && result ? a.allowedCount(l, result.rows.filter((r) => r.allowed).length, result.rows.length) : ""}</p>
      {state === "loading" && <p className="live-meta">{a.loading(l)}</p>}
      {state === "error" && (
        <section className="live-error" role="alert">
          <h2>{a.previewFailed(l)}</h2>
          <p>{describeFor("load")(l, error).body}</p>
        </section>
      )}
      {state === "shown" && result && (
        <section aria-labelledby="effect-title">
          <div className="live-heading live-section-head">
            <h2 id="effect-title">{a.effectFor(l, "\u2068" + result.login + "\u2069")}</h2>
          </div>
          <p className="live-meta">{kindLabel(l, result.kind) ?? result.kind} · {result.customer ? <bdi dir="ltr">{result.customer}</bdi> : a.noCustomer(l)} · {a.checkedAt(l, when(l, result.at))}</p>
          <p className="live-meta live-count">{a.allowedCount(l, result.rows.filter((r) => r.allowed).length, result.rows.length)}</p>
          <div className="security-table-scroll live-table live-effect-table">
            <table role="table">
              <colgroup><col className="live-col-e1" /><col className="live-col-e2" /><col className="live-col-e3" /><col /></colgroup>
              <thead role="rowgroup"><tr role="row">
                <th scope="col" role="columnheader">{a.colAction(l)}</th>
                <th scope="col" role="columnheader">{a.colField(l)}</th>
                <th scope="col" role="columnheader">{a.colResult(l)}</th>
                <th scope="col" role="columnheader">{a.colWhy(l)}</th>
              </tr></thead>
              <tbody role="rowgroup">
                {result.rows.map((r) => {
                  const repeat = r.action === last;
                  last = r.action;
                  return (
                    <tr key={r.action + r.field} role="row">
                      <td role="cell" className="live-user">{repeat ? <span className="live-quiet" aria-hidden="true">″</span> : null}<span className={repeat ? "visually-hidden" : undefined}>{actionLabel(l, r.action) ?? r.action}</span></td>
                      <td role="cell" data-label={a.colField(l)}>{r.field ? fieldLabel(l, r.field) ?? r.field : <><span aria-hidden="true">—</span><span className="visually-hidden">{a.notField(l)}</span></>}</td>
                      <td role="cell" data-label={a.colResult(l)} className={r.allowed ? "live-strong" : ""}>{r.allowed ? a.allowed(l) : a.notAllowed(l)}</td>
                      <td role="cell" data-label={a.colWhy(l)}>{reasonText(l, r)}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </section>
      )}
    </div>
  );
}

function Unavailable({ title, body, action }: { title: string; body: string; action: { label: string; go: () => void } }) {
  return (
    <div className="live-users">
      <section className="live-unavailable" aria-labelledby="access-unavailable">
        <h2 id="access-unavailable">{title}</h2>
        <p>{body}</p>
        <button type="button" className="live-secondary" onClick={action.go}>{action.label}</button>
      </section>
    </div>
  );
}

export default function LiveAccess({ locale: l, view, session, onSignOut, onView }: {
  locale: Locale; view: string; session: Session; onSignOut: () => void; onView: (view: string) => void;
}) {
  const api = useAdminClient();
  const wide = useWide();
  const order = useRef(sequence());
  const [page, setPage] = useState<AccessPage | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState<unknown>(null);
  const [mode, setMode] = useState<Mode | null>(null);
  const [notice, setNotice] = useState("");
  const [announce, setAnnounce] = useState("");
  const [focusTo, setFocusTo] = useState("");
  const [formKey, setFormKey] = useState(0);
  useEffect(() => {
    if (!focusTo) return;
    (document.querySelector<HTMLElement>(focusTo) ?? document.querySelector<HTMLElement>("#access-people"))?.focus();
    setFocusTo("");
  }, [focusTo]);
  const say = (text: string) => { setNotice(text); setAnnounce(""); setTimeout(() => setAnnounce(text), 50); };
  const load = useCallback(() => {
    const n = order.current.next();
    setStatus("loading");
    api.access().then(
      (got) => { if (order.current.latest(n)) { setPage(got); setStatus("ready"); setError(null); } },
      (e) => { if (order.current.latest(n)) { if (e instanceof AdminError && e.failure === "invalid") setPage(null); setError(e); setStatus("error"); } },
    );
  }, [api]);
  useEffect(() => { if (view === "grants" || view === "effects") load(); }, [load, view]);
  useEffect(() => { setMode(null); setNotice(""); }, [view]);
  const toGrants = { label: a.goGrants(l), go: () => onView("grants") };
  if (view === "roles") return <Unavailable title={a.rolesTitle(l)} body={a.rolesBody(l)} action={toGrants} />;
  if (view === "support") return <Unavailable title={a.supportTitle(l)} body={a.supportBody(l)} action={toGrants} />;
  if (status === "error" && error instanceof AdminError && error.failure === "denied") {
    return (
      <div className="live-users"><section className="live-unavailable" aria-labelledby="access-denied"><h2 id="access-denied">{a.deniedTitle(l)}</h2><p>{a.denied(l)}</p></section></div>
    );
  }
  const loadError = status === "error" && (
    <section className="live-error" role="alert">
      <h2>{a.loadFailed(l)}</h2>
      <p>{describeFor("load")(l, error).body}</p>
      {error instanceof AdminError && error.failure === "unauthenticated"
        ? <button type="button" className="live-secondary" onClick={onSignOut}>{w.signInAgain(l)}</button>
        : <button type="button" className="live-secondary" onClick={load}>{c.tryAgain(l)}</button>}
    </section>
  );
  if (view === "effects") {
    if (!page) return <div className="live-users">{loadError || <p className="live-empty">{a.loading(l)}</p>}</div>;
    return <Effects l={l} api={api} page={page} />;
  }
  const self = session.user;
  const grants = page ? people(page) : { current: [], ended: [] };
  const ceiling = page ? ceilingGrants(page) : [];
  const delegations = page ? delegationsOf(page) : { current: [], ended: [] };
  const selected = mode?.kind === "revoke" ? (mode.target.kind === "grant" ? mode.target.row.id : mode.target.id) : mode?.kind === "decide" ? mode.id : "";
  const deciding = page && mode?.kind === "decide" ? page.requests.find((r) => r.id === mode.id) : undefined;
  const decide = (action: Decision, id: string) => { setNotice(""); setMode({ kind: "decide", action, id }); };
  const revokeGrant = (row: AccessGrant) => { setNotice(""); setMode({ kind: "revoke", target: { kind: "grant", row } }); };
  let panel: ReactNode = null;
  if (page && mode?.kind === "decide" && deciding) {
    const back = `[data-request="${mode.id}"] [data-act="${mode.action}"]`;
    panel = <DecidePanel key={mode.action + mode.id} l={l} api={api} page={page} request={deciding} action={mode.action} onReload={load}
      onCancel={() => { setMode(null); setFocusTo(back); }}
      onDone={(message) => { say(message); setMode(null); load(); setFocusTo(mode.action === "approve" && deciding.change === "revoke" ? "#access-revocations" : "#access-requests"); }} />;
  } else if (page && mode?.kind === "revoke") {
    const back = mode.target.kind === "grant" ? `[data-grant="${mode.target.row.id}"]` : `[data-delegation="${mode.target.id}"]`;
    panel = <Revoke key={selected} l={l} api={api} page={page} target={mode.target} onReload={load}
      onCancel={() => { setMode(null); setFocusTo(back); }}
      onDone={(message) => { say(message); setMode(null); setFormKey((k) => k + 1); load(); setFocusTo(mode.target.kind === "grant" ? "#access-requests" : "#access-revocations"); }} />;
  } else if (page && page.catalog.length > 0 && maxDays(page.catalog[0].until, page.now) > 0 && (wide || mode?.kind === "form")) {
    panel = <GrantForm key={formKey} l={l} api={api} page={page} self={self} stepping={!wide} onReload={load}
      onCancel={() => { setMode(null); setFocusTo(".access-grant-open"); }}
      onDone={() => { say(q.requestSent(l)); setMode(null); setFormKey((k) => k + 1); load(); setFocusTo("#access-requests"); }} />;
  }
  const stepping = !wide && mode !== null;
  const backTo = mode?.kind === "revoke" ? (mode.target.kind === "grant" ? `[data-grant="${mode.target.row.id}"]` : `[data-delegation="${mode.target.id}"]`)
    : mode?.kind === "decide" ? `[data-request="${mode.id}"] [data-act="${mode.action}"]` : ".access-grant-open";
  return (
    <div className="live-users" data-live="grants" data-wide={wide ? "true" : "false"}>
      <p className="visually-hidden" role="status" aria-live="polite">{announce}</p>
      {!stepping && (
        <div className="live-list live-sections">
          <section aria-labelledby="access-people">
            <div className="live-toolbar">
              <div className="live-heading">
                <h2 id="access-people" tabIndex={-1}>{a.peopleTitle(l)}</h2>
                {page && <p className="live-meta">{a.activeGrants(l, grants.current.filter((g) => g.state === "active" && g.effective).length)}</p>}
                {page && <p className="live-meta">{w.loadedAt(l, when(l, page.now))}</p>}
              </div>
              <div className="live-tools">
                {!wide && page && page.catalog.length > 0 && maxDays(page.catalog[0].until, page.now) > 0 && <button type="button" className="live-primary access-grant-open" onClick={() => { setNotice(""); setMode({ kind: "form" }); }}>{q.requestTitle(l)}</button>}
                <button type="button" className="live-secondary" onClick={() => { setNotice(""); load(); }}>{w.reload(l)}</button>
              </div>
            </div>
            {notice && <p className="service-status">{notice}</p>}
            {status === "loading" && !page && <p role="status" className="live-meta">{a.loading(l)}</p>}
            {loadError}
            {page && (grants.current.length === 0 ? <p className="live-empty">{a.noGrants(l)}</p> : <GrantTable l={l} page={page} rows={grants.current} self={self} selected={selected} onRevoke={revokeGrant} />)}
            {page && grants.ended.length > 0 && (
              <details className="live-ended"><summary>{a.endedGrants(l, grants.ended.length)}</summary><GrantTable l={l} page={page} rows={grants.ended} self={self} selected="" /></details>
            )}
          </section>
          {page && <RequestSection l={l} page={page} selected={selected} onAct={(action, r) => decide(action, r.id)} />}
          {page && <RevocationSection l={l} page={page} />}
          {page && (
            <section aria-labelledby="access-ceiling">
              <h2 id="access-ceiling" className="live-title">{a.ceilingTitle(l)}</h2>
              <p className="live-impact">{a.ceilingLead(l)}</p>
              <div className="security-table-scroll live-table">
                <table role="table">
                  <thead role="rowgroup"><tr role="row"><th scope="col" role="columnheader">{a.colAccess(l)}</th><th scope="col" role="columnheader">{a.colCustomer(l)}</th><th scope="col" role="columnheader">{a.colStatus(l)}</th></tr></thead>
                  <tbody role="rowgroup">
                    {ceiling.map((g) => (
                      <tr key={g.id} role="row">
                        <td role="cell" className="live-user">{accessText(l, g.action, g.fields)}<span className="live-sub">{kindLabel(l, g.kind) ?? g.kind}</span></td>
                        <td role="cell" data-label={a.colCustomer(l)}>{g.scope.customer === "all" ? a.allCustomers(l) : g.scope.customer}</td>
                        <td role="cell" data-label={a.colStatus(l)}>{a.stActive(l)}<span className="live-sub">{a.until(l, when(l, g.not_after))}</span></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          )}
          {page && (
            <section aria-labelledby="access-delegations">
              <div className="live-heading live-section-head">
                <h2 id="access-delegations">{a.delegTitle(l)}</h2>
                <p className="live-meta">{a.activeDelegations(l, delegations.current.length)}</p>
              </div>
              {delegations.current.length === 0 ? <p className="live-empty">{a.noDeleg(l)}</p> : (
                <div className="security-table-scroll live-table live-access">
                  <table role="table">
                    <thead role="rowgroup"><tr role="row">
                      <th scope="col" role="columnheader">{a.colDelegation(l)}</th>
                      <th scope="col" role="columnheader">{a.colAccess(l)}</th>
                      <th scope="col" role="columnheader">{a.colStatus(l)}</th>
                      <th scope="col" role="columnheader"><span className="visually-hidden">{a.colActions(l)}</span></th>
                    </tr></thead>
                    <tbody role="rowgroup">
                      {delegations.current.map((d) => {
                        const [state, sub] = stateLine(l, d);
                        const who = loginOf(page, d.delegate);
                        return (
                          <tr key={d.id} role="row" className={selected === d.id ? "live-selected" : undefined}>
                            <td role="cell" className="live-user"><bdi dir="ltr">{loginOf(page, d.delegator)}</bdi> {l === "ar" ? "←" : "→"} <bdi dir="ltr">{who}</bdi>
                              {d.parent && <span className="live-sub live-mono">{a.fromParent(l, d.parent)}</span>}{d.redelegate && <span className="live-sub">{a.canRedelegate(l)}</span>}</td>
                            <td role="cell" data-label={a.colAccess(l)}>{accessText(l, d.action, d.fields)}<span className="live-sub">{scopeText(l, d.kind, d.scope.customer)}</span></td>
                            <td role="cell" data-label={a.colStatus(l)} className={d.state !== "active" ? "live-strong" : ""} data-state={d.state}>{state}<span className="live-sub">{sub}</span></td>
                            <td role="cell">{revocable(d.state) && <button type="button" className="live-secondary live-revoke" data-delegation={d.id}
                              onClick={() => { setNotice(""); setMode({ kind: "revoke", target: { kind: "delegation", id: d.id, who, access: accessText(l, d.action, d.fields), until: [state, sub].join(" · ") } }); }}>{a.revoke(l)}</button>}</td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              )}
              {delegations.ended.length > 0 && (
                <details className="live-ended"><summary>{a.endedDeleg(l, delegations.ended.length)}</summary>
                  <ul role="list" className="access-ended-delegations access-bullets">
                    {delegations.ended.map((d) => (
                      <li key={d.id} data-ended={d.id}><span className="live-mono" dir="ltr">{d.id}</span> · {d.state === "ended" ? v.endedByChain(l) : stateLine(l, d)[0]}</li>
                    ))}
                  </ul>
                </details>
              )}
            </section>
          )}
        </div>
      )}
      {(wide || stepping) && panel && (
        <aside className="live-panel" aria-label={mode?.kind === "revoke" ? (mode.target.kind === "grant" ? q.requestRevocation(l) : a.revokeDelegTitle(l))
          : mode?.kind === "decide" ? (mode.action === "approve" ? q.approveTitle(l) : mode.action === "reject" ? q.rejectTitle(l) : q.withdrawTitle(l)) : q.requestTitle(l)}>
          {stepping && <button type="button" className="service-button-link live-back" onClick={() => { setMode(null); setFocusTo(backTo); }}>{a.backToGrants(l)}</button>}
          {panel}
        </aside>
      )}
    </div>
  );
}
