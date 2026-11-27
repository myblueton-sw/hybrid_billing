import { lc } from "../../../../shared/list/list-copy";
import { ListSettings } from "../../../../shared/list/ListParts";
import { pageLimitOf } from "../../../../service/query-settings";
import { useCallback, useEffect, useRef, useState, type FormEvent, type ReactNode } from "react";
import { diagnosticCopy } from "../../../../service/diagnostic-copy";
import type { Locale } from "../../../../shared/types";
import { AdminError, attempt, sequence, type AccessPage, type AdminClient, type Attempt, type Preview } from "../../../../service/admin";
import { useAdminClient } from "../../../../service/context";
import type { Session } from "../../../../service/api";
import { c } from "../../../../service/copy";
import { Summary, type Problems } from "../../../../service/Service";
import { FailureBox, Reason, reasonProblem, settles, useWide } from "../users/Live";
import { w } from "../users/live-copy";
import { failureText } from "../users/live-model";
import { when } from "../authentication/live-model";
import { a, accessPhrase, actionLabel, fieldLabel, kindLabel } from "./live-copy";
import { accessText, capabilityKey, grantCapabilities, selectedCapability, maxDays, reasonText, wholeDays } from "./live-model";
import { q, v } from "./request-copy";
import { additionsPaused, keepAttempt } from "./request-model";
import { DecidePanel, describeRequest, RequestSection, type Decision } from "./Requests";
import { RevocationSection } from "./Revocations";
import { PermissionList, effectiveLines } from "./Permissions";
import { p } from "./permission-copy";
import type { Permission } from "../../../../service/access-query";
import type { AccessSubject, ListedRequest } from "../../../../service/access-lists";
import { PersonPicker } from "./PersonPicker";
import { pc } from "./person-copy";
import "../users/live.css";
import "../../../../shared/list/list-pattern.css";
import "./access.css";

type Target = { kind: "grant"; id: string; who: string; access: string; customer: string; status: string } | { kind: "delegation"; id: string; who: string; access: string; until: string };
type Mode = { kind: "form" } | { kind: "revoke"; target: Target } | { kind: "decide"; action: Decision; row: ListedRequest };

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

function GrantForm({ l, api, page, self, onDone, onCancel, onReload, onRefresh, stepping }: {
  l: Locale; api: AdminClient; page: AccessPage; self: string; onDone: (login: string) => void; onCancel: () => void; onReload: () => void; onRefresh: () => void; stepping: boolean;
}) {
  const capabilities = grantCapabilities(page);
  const [capability, setCapability] = useState(() => capabilities.length === 1 ? capabilityKey(capabilities[0]) : "");
  const pair = selectedCapability(page, capability);
  const most = pair ? maxDays(pair.until, page.now) : 0;
  const [person, setPerson] = useState<AccessSubject | null>(null);
  const subject = person?.user_id ?? "";
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
  useEffect(() => { heading.current?.focus(); }, []);
  const paused = additionsPaused(page.ledger);
  const edit = (f: (v: string) => void) => (v: string) => { tried.current = attempt(); f(v); };
  useEffect(() => {
    if (capability && !pair) {
      setCapability("");
      setDays("");
      setProblems([{ field: "access-capability", message: a.accessGone(l) }]);
      setError(null);
      tried.current = attempt();
      kept.current = false;
    }
  }, [capability, pair, l]);
  const changeCapability = (key: string) => {
    const next = selectedCapability(page, key);
    setCapability(key);
    setDays(next ? String(Math.min(30, maxDays(next.until, page.now))) : "");
    setProblems([]);
    setError(null);
    tried.current = attempt();
    kept.current = false;
  };
  const n = wholeDays(days, most);
  const login = person?.login ?? "";
  const phrase = pair ? accessPhrase(l, pair.action, pair.kind, pair.fields) ?? accessText(l, pair.action, pair.fields) : "";
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    const found: Problems = [];
    if (!subject) found.push({ field: "access-person", message: a.errPerson(l) });
    if (!pair) found.push({ field: "access-capability", message: a.chooseAccess(l) });
    if (pair && n === null) found.push({ field: "access-days", message: a.errDays(l, most) });
    const r = reasonProblem(l, reason);
    if (r) found.push({ field: "live-reason", message: r });
    setProblems(found);
    setError(null);
    if (found.length || !pair) { setFocusKey((k) => k + 1); return; }
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
      <PersonPicker l={l} api={api} id="access-person" self={self} pageLimit={pageLimitOf(page.query_settings)} onRefresh={onRefresh} chosen={person} error={field("access-person") ?? ""} disabled={busy}
        flagLine={pc.requestUnconfirmed(l)} onChoose={(next) => { tried.current = attempt(); setPerson(next); }} />
      <div className="service-field">
        <label htmlFor="access-capability">{a.access(l)}</label>
        <p className="service-hint" id="access-capability-hint">{a.accessHint(l)}</p>
        {page.catalog_incomplete && <p className="service-hint" id="access-capability-short" data-catalog-short="true">{a.catalogShort(l)}</p>}
        {field("access-capability") && <p className="service-field-error" id="access-capability-error"><span className="visually-hidden">{c.errorPrefix(l)} </span>{field("access-capability")}</p>}
        <select id="access-capability" className="live-select" value={pair ? capability : ""} disabled={busy}
          aria-invalid={field("access-capability") ? true : undefined} aria-describedby={["access-capability-hint", page.catalog_incomplete ? "access-capability-short" : "", field("access-capability") ? "access-capability-error" : ""].filter(Boolean).join(" ")}
          onChange={(e) => changeCapability(e.target.value)}>
          <option value="">{a.chooseAccess(l)}</option>
          {capabilities.map((row) => <option key={capabilityKey(row)} value={capabilityKey(row)}>{accessText(l, row.action, row.fields)} · {kindLabel(l, row.kind)}</option>)}
        </select>
        {pair && <p className="service-hint">{accessText(l, pair.action, pair.fields)} · {kindLabel(l, pair.kind)}</p>}
      </div>
      {pair && <dl className="live-facts">
        <div><dt>{a.customer(l)}</dt><dd>{a.allCustomers(l)}</dd>{pair.kind === "focus_data" && <dd className="live-note">{a.focusAll(l)}</dd>}</div>
      </dl>}
      <div className="service-field live-minutes">
        <label htmlFor="access-days">{a.duration(l)}</label>
        <p className="service-hint" id="access-days-hint">{pair ? a.daysHint(l, most, when(l, pair.until)) : a.chooseDurationHint(l)}</p>
        {field("access-days") && <p className="service-field-error" id="access-days-error"><span className="visually-hidden">{c.errorPrefix(l)} </span>{field("access-days")}</p>}
        <span className="live-unit">
          <input id="access-days" inputMode="numeric" dir="ltr" value={days} disabled={busy} aria-invalid={field("access-days") ? true : undefined}
            aria-describedby={["access-days-hint", field("access-days") ? "access-days-error" : ""].filter(Boolean).join(" ")} onChange={(e) => edit(setDays)(e.target.value)} />
          <span>{a.days(l)}</span>
        </span>
      </div>
      {login && n && <p className="live-impact">{q.requestImpact(l, login, phrase, n)}</p>}
      <Reason l={l} value={reason} onChange={edit(setReason)} error={field("live-reason") ?? ""} />
      <div className="live-buttons">
        <button type="submit" className={paused ? "live-primary access-paused" : "live-primary"} aria-busy={busy || undefined} aria-describedby={paused ? "access-paused" : undefined}
          disabled={busy || paused}>{busy ? w.saving(l) : q.requestTitle(l)}</button>
        {stepping && <button type="button" className="live-secondary" onClick={onCancel}>{w.cancel(l)}</button>}
      </div>
    </form>
  );
}

function WouldStop({ l, api, source, id }: { l: Locale; api: AdminClient; source: "grant" | "delegation"; id: string }) {
  const [tries, setTries] = useState(0);
  const [read, setRead] = useState<{ state: "loading" } | { state: "error" } | { state: "ready"; ids: string[]; at: string }>({ state: "loading" });
  useEffect(() => {
    let active = true;
    setRead({ state: "loading" });
    api.accessPermission(source, id).then(
      (detail) => { if (active) setRead({ state: "ready", ids: detail.would_end, at: detail.observed_at }); },
      () => { if (active) setRead({ state: "error" }); },
    );
    return () => { active = false; };
  }, [api, source, id, tries]);
  const shown = read.state === "ready" ? read.ids.slice(0, 10) : [];
  const rest = read.state === "ready" ? read.ids.slice(10) : [];
  const line = (x: string) => <li key={x}><bdi dir="ltr" className="live-mono" translate="no">{x}</bdi></li>;
  return (
    <div className="access-impact-list access-would-stop" data-would-stop={read.state}>
      <div className="live-heading live-section-head">
        <h3>{q.stopTitle(l)}</h3>
        {read.state === "ready" && read.ids.length > 0 && <p className="live-meta" data-stop-count={read.ids.length}>{q.stopCount(l, read.ids.length)}</p>}
      </div>
      <p className="live-note">{source === "grant" ? q.stopLeadGrant(l) : q.stopLeadDeleg(l)}</p>
      {read.state === "loading" && <p role="status" className="live-meta">{q.stopLoading(l)}</p>}
      {read.state === "error" && (
        <div role="alert" className="access-stop-failed">
          <p>{q.stopFailed(l)}</p>
          <button type="button" className="live-secondary" onClick={() => setTries((n) => n + 1)}>{c.tryAgain(l)}</button>
        </div>
      )}
      {read.state === "ready" && <p className="live-meta">{a.checkedAt(l, when(l, read.at))}</p>}
      {read.state === "ready" && (read.ids.length === 0 ? <p className="live-note">{q.stopNone(l)}</p> : (
        <>
          <ul role="list" className="access-stop-list">{shown.map(line)}</ul>
          {rest.length > 0 && <details className="live-ended"><summary>{q.stopMore(l, rest.length)}</summary><ul role="list" className="access-stop-list">{rest.map(line)}</ul></details>}
        </>
      ))}
    </div>
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
  const who = target.who;
  const id = target.id;
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
  return (
    <form className="live-step" onSubmit={submit} noValidate>
      <h2 tabIndex={-1} ref={heading}>{title}</h2>
      <p className="live-who"><bdi dir="ltr">{who}</bdi></p>
      {problem && <Summary locale={l} title={c.check(l)} problems={[{ field: "live-reason", message: problem }]} focusKey={focusKey} />}
      {error !== null && <FailureBox l={l} error={error} focusKey={focusKey} describe={grant ? describeRequest : describeFor("revoke")} />}
      <dl className="live-facts">
        <div><dt>{a.access(l)}</dt><dd>{target.access}</dd></div>
        {target.kind === "grant" && <div><dt>{a.customer(l)}</dt><dd>{target.customer}</dd></div>}
        <div><dt>{a.colStatus(l)}</dt><dd>{target.kind === "grant" ? target.status : target.until}</dd></div>
      </dl>
      <p className="live-impact">{grant ? q.revokeRequestLead(l, who) : a.revokeDelegImpact(l, who)}</p>
      <WouldStop l={l} api={api} source={grant ? "grant" : "delegation"} id={id} />
      <Reason l={l} value={reason} onChange={(v) => { tried.current = attempt(); setReason(v); }} error={problem} />
      <div className="live-buttons">
        <button type="submit" className="live-primary" aria-busy={busy || undefined} disabled={busy}>{busy ? w.saving(l) : title}</button>
        <button type="button" className="live-secondary" onClick={onCancel}>{w.cancel(l)}</button>
      </div>
    </form>
  );
}

function Effects({ l, api, page, onRefresh }: { l: Locale; api: AdminClient; page: AccessPage; onRefresh: () => void }) {
  const kinds = Array.from(new Set(page.catalog.map((x) => x.kind)));
  const [person, setPerson] = useState<AccessSubject | null>(null);
  const subject = person?.user_id ?? "";
  const [picked, setKind] = useState(kinds[0] ?? "");
  const kind = kinds.includes(picked) ? picked : kinds[0] ?? "";
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
      setResult({ rows, at: new Date().toISOString(), login: person?.login || subject, customer: customer.trim(), kind });
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
        <PersonPicker l={l} api={api} id="effect-person" self="" pageLimit={pageLimitOf(page.query_settings)} onRefresh={onRefresh} chosen={person} error="" flagLine={v.previewUnconfirmed(l)}
          onChoose={(next) => { order.current.next(); setPerson(next); setResult(null); setState((x) => (x === "start" ? "start" : "cleared")); }} />
        <div className="service-field">
          {kinds.length > 1 ? (
            <>
              <label htmlFor="effect-kind">{a.data(l)}</label>
              <select id="effect-kind" className="live-select" value={kind} aria-describedby={page.catalog_incomplete ? "effect-kind-short" : undefined} onChange={(e) => change(setKind)(e.target.value)}>
                {kinds.map((k) => <option key={k} value={k}>{kindLabel(l, k) ?? k}</option>)}
              </select>
            </>
          ) : (
            <>
              <span className="live-fact-label">{a.data(l)}</span>
              <span className="live-fact-box">{kindLabel(l, kind) ?? kind}</span>
            </>
          )}
          {page.catalog_incomplete && kind && <p className="service-hint live-hint-below" id="effect-kind-short" data-catalog-short="true">{a.kindsShort(l)}</p>}
        </div>
        <div className="service-field">
          <label htmlFor="effect-customer">{a.customerOpt(l)}</label>
          <input id="effect-customer" className="live-mono live-customer" dir="ltr" value={customer} aria-describedby="effect-customer-hint" onChange={(e) => change(setCustomer)(e.target.value)} />
          <p className="service-hint live-hint-below" id="effect-customer-hint">{a.customerOptHint(l)}</p>
        </div>
        <button type="submit" className="live-primary" disabled={!subject || !kind || state === "loading"} aria-describedby={kind ? undefined : "effect-no-kind"}>{a.showEffect(l)}</button>
        {!kind && <p className="live-error" role="alert" id="effect-no-kind">{page.catalog_incomplete ? a.kindsNone(l) : a.noKind(l)}</p>}
      </form>
      <p className="live-preview-note"><strong>{a.previewLead(l)}</strong> {a.previewNote(l)}</p>
      {state === "start" && kind && <p className="live-empty">{a.startHint(l)}</p>}
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

export default function LiveAccess({ locale: l, view, session, onSignOut, onView, params, onParams }: {
  locale: Locale; view: string; session: Session; onSignOut: () => void; onView: (view: string) => void;
  params: URLSearchParams; onParams: (extra: Record<string, string | null>, push?: boolean) => void;
}) {
  const api = useAdminClient();
  const wide = useWide();
  const order = useRef(sequence());
  const [page, setPage] = useState<AccessPage | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState<unknown>(null);
  const [mode, setMode] = useState<Mode | null>(null);
  const [notice, setNotice] = useState("");
  const [noticeAt, setNoticeAt] = useState<"list" | "requests">("list");
  const [announce, setAnnounce] = useState("");
  const [focusTo, setFocusTo] = useState("");
  const [formKey, setFormKey] = useState(0);
  const [listed, setListed] = useState(0);
  useEffect(() => {
    if (!focusTo) return;
    (document.querySelector<HTMLElement>(focusTo) ?? document.querySelector<HTMLElement>("#access-list"))?.focus();
    setFocusTo("");
  }, [focusTo]);
  const say = (text: string, at: "list" | "requests" = "list") => { setNotice(text); setNoticeAt(at); setAnnounce(""); setTimeout(() => setAnnounce(text), 50); };
  const read = useCallback((listsAfter: boolean) => {
    const n = order.current.next();
    if (!listsAfter) setListed((x) => x + 1);
    setStatus("loading");
    api.access().then(
      (got) => { if (order.current.latest(n)) { setPage(got); setStatus("ready"); setError(null); if (listsAfter) setListed((x) => x + 1); } },
      (e) => {
        if (!order.current.latest(n)) return;
        if (e instanceof AdminError && e.failure === "invalid") setPage(null);
        setError(e); setStatus("error");
        if (listsAfter) setListed((x) => x + 1);
      },
    );
  }, [api]);
  const load = useCallback(() => read(false), [read]);
  useEffect(() => { if (view === "grants" || view === "effects") load(); }, [load, view]);
  useEffect(() => { setMode(null); setNotice(""); }, [view]);
  const toGrants = { label: a.goGrants(l), go: () => onView("grants") };
  if (view === "roles") return <Unavailable title={a.rolesTitle(l)} body={a.rolesBody(l)} action={toGrants} />;
  if (view === "support") return <Unavailable title={a.supportTitle(l)} body={a.supportBody(l)} action={toGrants} />;
  const refresh = () => read(true);
  if (status === "error" && error instanceof AdminError && error.failure === "denied") {
    return (
      <div className="live-users"><section className="live-unavailable" aria-labelledby="access-denied"><h2 id="access-denied">{a.deniedTitle(l)}</h2><p>{a.denied(l)}</p></section></div>
    );
  }
  const loadError = status === "error" && (
    <section className="live-error" role="alert">
      <h2>{a.loadFailed(l)}</h2>
      <p>{describeFor("load")(l, error).body}</p>
      {page && <p>{diagnosticCopy.kept(l)}</p>}
      {error instanceof AdminError && error.failure === "unauthenticated"
        ? <button type="button" className="live-secondary" onClick={onSignOut}>{w.signInAgain(l)}</button>
        : <button type="button" className="live-secondary" onClick={load}>{c.tryAgain(l)}</button>}
    </section>
  );
  if (view === "effects") {
    if (!page) return <div className="live-users">{loadError || <p className="live-empty">{a.loading(l)}</p>}</div>;
    return <>{loadError && <div className="live-users">{loadError}</div>}<Effects l={l} api={api} page={page} onRefresh={refresh} /></>;
  }
  const self = session.user;
  const selected = mode?.kind === "revoke" ? mode.target.id : mode?.kind === "decide" ? mode.row.id : "";
  const decide = (action: Decision, row: ListedRequest) => { setNotice(""); setMode({ kind: "decide", action, row }); };
  const customerOf = (row: Permission) => (row.scope.customer.any ? a.allCustomers(l) : row.scope.customer.id);
  const revokeGrant = (row: Permission) => { setNotice(""); setMode({ kind: "revoke", target: { kind: "grant", id: row.id, who: row.subject_login || row.subject,
    access: `${accessText(l, row.action, row.fields)} · ${kindLabel(l, row.kind) ?? row.kind}`, customer: customerOf(row), status: effectiveLines(l, row).join(" · ") } }); };
  const revokeDelegation = (row: Permission) => { setNotice(""); setMode({ kind: "revoke", target: { kind: "delegation", id: row.id, who: row.subject_login || row.subject,
    access: accessText(l, row.action, row.fields), until: effectiveLines(l, row).join(" · ") } }); };
  let panel: ReactNode = null;
  if (page && mode?.kind === "decide") {
    const back = `[data-request="${mode.row.id}"] [data-act="${mode.action}"]`;
    panel = <DecidePanel key={mode.action + mode.row.id} l={l} api={api} revision={page.revision} request={mode.row} action={mode.action} onReload={load}
      onCancel={() => { setMode(null); setFocusTo(back); }}
      onDone={(message) => { say(message, "requests"); setMode(null); load(); setFocusTo("#access-requests"); }} />;
  } else if (page && mode?.kind === "revoke") {
    const back = mode.target.kind === "grant" ? `[data-grant="${mode.target.id}"]` : `[data-delegation="${mode.target.id}"]`;
    panel = <Revoke key={selected} l={l} api={api} page={page} target={mode.target} onReload={load}
      onCancel={() => { setMode(null); setFocusTo(back); }}
      onDone={(message) => { say(message, mode.target.kind === "grant" ? "requests" : "list"); setMode(null); setFormKey((k) => k + 1); load(); setFocusTo(mode.target.kind === "grant" ? "#access-requests" : "#access-revocations"); }} />;
  } else if (page && grantCapabilities(page).length > 0 && mode?.kind === "form") {
    panel = <GrantForm key={formKey} l={l} api={api} page={page} self={self} stepping={!wide} onReload={load} onRefresh={refresh}
      onCancel={() => { setMode(null); setFocusTo(".access-grant-open"); }}
      onDone={() => { say(q.requestSent(l), "requests"); setMode(null); setFormKey((k) => k + 1); load(); setFocusTo("#access-requests"); }} />;
  }
  const stepping = !wide && mode !== null;
  const backTo = mode?.kind === "revoke" ? (mode.target.kind === "grant" ? `[data-grant="${mode.target.id}"]` : `[data-delegation="${mode.target.id}"]`)
    : mode?.kind === "decide" ? `[data-request="${mode.row.id}"] [data-act="${mode.action}"]` : ".access-grant-open";
  return (
    <div className="live-users" data-live="grants" data-wide={wide ? "true" : "false"} data-panel={panel ? "true" : "false"}>
      <p className="visually-hidden" role="status" aria-live="polite">{announce}</p>
      {!stepping && (
        <div className="live-list live-sections">
          <section aria-labelledby="access-list">
            <div className="live-toolbar">
              <div className="live-heading">
                <h2 id="access-list" tabIndex={-1}>{p.title(l)}</h2>
              </div>
              <div className="live-tools">
                {page && grantCapabilities(page).length > 0 && <button type="button" className="live-primary access-grant-open" aria-expanded={mode?.kind === "form"} aria-controls="access-panel"
                  onClick={() => { setNotice(""); if (mode?.kind === "form") document.querySelector<HTMLElement>("#access-panel h2")?.focus(); else { setMode({ kind: "form" }); setFocusTo("#access-panel h2"); } }}>{q.requestTitle(l)}</button>}
                <button type="button" className="live-secondary" onClick={() => { setNotice(""); load(); }}>{w.reload(l)}</button>
              </div>
            </div>
            {notice && noticeAt === "list" && <p className="service-status">{notice}</p>}
            {page && page.catalog_incomplete && grantCapabilities(page).length === 0 && <p className="live-note" data-catalog-none="true">{a.catalogNone(l)}</p>}
            {status === "loading" && !page && <p role="status" className="live-meta">{a.loading(l)}</p>}
            {loadError}
            <PermissionList l={l} api={api} self={self} params={params} onParams={onParams} generation={listed} selected={selected} onSignOut={onSignOut}
              pageLimit={page ? pageLimitOf(page.query_settings) : null} settled={page !== null || status !== "loading"} onRefresh={refresh}
              actions={page ? [...new Set(page.catalog.map((x) => x.action))] : []} onRevokeGrant={revokeGrant} onRevokeDelegation={revokeDelegation} />
          </section>
          {page && <RequestSection l={l} api={api} params={params} onParams={onParams} generation={listed} pageLimit={pageLimitOf(page.query_settings)} onRefresh={refresh} startedAt={page.started_at} settings={page.settings} selected={selected}
            notice={noticeAt === "requests" ? notice : ""} onAct={decide} onSignOut={onSignOut} />}
          {page && <RevocationSection l={l} api={api} params={params} onParams={onParams} generation={listed} pageLimit={pageLimitOf(page.query_settings)} onRefresh={refresh} ledger={page.ledger} onSignOut={onSignOut} />}
          {page && <ListSettings l={l} rows={page.query_settings} />}
        </div>
      )}
      {(wide || stepping) && panel && (
        <aside className="live-panel" id="access-panel" data-closable={wide ? "true" : undefined} aria-label={mode?.kind === "revoke" ? (mode.target.kind === "grant" ? q.requestRevocation(l) : a.revokeDelegTitle(l))
          : mode?.kind === "decide" ? (mode.action === "approve" ? q.approveTitle(l) : mode.action === "reject" ? q.rejectTitle(l) : q.withdrawTitle(l)) : q.requestTitle(l)}>
          {stepping && <button type="button" className="service-button-link live-back" onClick={() => { setMode(null); setFocusTo(backTo); }}>{a.backToGrants(l)}</button>}
          {wide && <div className="access-panel-close"><button type="button" className="live-secondary" onClick={() => { setMode(null); setFocusTo(backTo); }}>{lc.close(l)}</button></div>}
          {panel}
        </aside>
      )}
    </div>
  );
}
