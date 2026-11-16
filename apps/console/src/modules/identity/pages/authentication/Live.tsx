import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { adminClient, AdminError, attempt, sequence, type AdminClient, type Attempt, type PolicyPage, type ProposalRow, type RevisionRow } from "../../../../service/admin";
import type { Session } from "../../../../service/api";
import { c } from "../../../../service/copy";
import { Summary, type Problems } from "../../../../service/Service";
import { FailureBox, Reason, reasonProblem, settles, useWide } from "../users/Live";
import { w } from "../users/live-copy";
import { p } from "./live-copy";
import { canAct, duration, history, methodsText, pending, policyFailure, policyStep, stale, stepText, wholeMinutes, when } from "./live-model";
import "../users/live.css";

type Mode = { kind: "propose" } | { kind: "approve" | "withdraw"; row: ProposalRow };

function Compare({ l, current, next, stacked = false }: { l: Locale; current: RevisionRow | null; next: RevisionRow; stacked?: boolean }) {
  const rows: [string, string, string][] = [
    [p.maxAge(l), current ? duration(l, current.max_age_seconds) : "—", duration(l, next.max_age_seconds)],
    [p.stepUp(l), current ? stepText(l, current) : "—", stepText(l, next)],
  ];
  if (stacked)
    return (
      <dl className="live-facts live-compare-stack">
        {rows.map(([label, before, after]) => (
          <div key={label}>
            <dt>{label}</dt>
            <dd>{p.current(l)}: {before}</dd>
            <dd className={before !== after ? "live-strong" : ""}>{p.proposed(l)}: {after}{before !== after && <> {p.changed(l)}</>}</dd>
          </div>
        ))}
      </dl>
    );
  return (
    <div className="security-table-scroll live-table live-compare">
      <table role="table">
        <thead role="rowgroup"><tr role="row"><th scope="col" role="columnheader">{p.setting(l)}</th><th scope="col" role="columnheader">{p.current(l)}</th><th scope="col" role="columnheader">{p.proposed(l)}</th></tr></thead>
        <tbody role="rowgroup">
          {rows.map(([label, before, after]) => (
            <tr key={label} role="row">
              <td role="cell" className="live-user">{label}</td>
              <td role="cell" data-label={p.current(l)}>{before}</td>
              <td role="cell" data-label={p.proposed(l)} className={before !== after ? "live-strong" : ""}>{after}{before !== after && <> {p.changed(l)}</>}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function Propose({ l, api, page, onDone, onCancel, onReload, stepping }: {
  l: Locale; api: AdminClient; page: PolicyPage; onDone: (target: string) => void; onCancel: () => void; onReload: () => void; stepping: boolean;
}) {
  const current = page.policy;
  const [age, setAge] = useState(current ? String(Math.round(current.max_age_seconds / 60)) : "");
  const [stepOn, setStepOn] = useState(current ? policyStep(current) > 0 : false);
  const [step, setStep] = useState(current && policyStep(current) ? String(Math.round(policyStep(current) / 60)) : "");
  const [reason, setReason] = useState("");
  const [touched, setTouched] = useState(false);
  const [busy, setBusy] = useState(false);
  const [problems, setProblems] = useState<Problems>([]);
  const [error, setError] = useState<unknown>(null);
  const [focusKey, setFocusKey] = useState(0);
  const tried = useRef<Attempt>(attempt());
  useEffect(() => { tried.current = attempt(); }, [page.revision]);
  const edit = (f: (v: string) => void) => (v: string) => { tried.current = attempt(); f(v); };
  const maxAge = wholeMinutes(age, 5, 720);
  const stepValue = stepOn ? wholeMinutes(step, 5, maxAge ?? 720) : 0;
  const same = current !== null && maxAge !== null && stepValue !== null && maxAge * 60 === current.max_age_seconds && (stepValue ?? 0) * 60 === policyStep(current);
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    const found: Problems = [];
    if (maxAge === null) found.push({ field: "policy-age", message: p.errMaxAge(l) });
    if (stepOn && stepValue === null) found.push({ field: "policy-step", message: p.errStep(l, maxAge ?? 720) });
    const r = reasonProblem(l, reason);
    if (r) found.push({ field: "live-reason", message: r });
    setProblems(found);
    setError(null);
    if (found.length) { setFocusKey((k) => k + 1); return; }
    setBusy(true);
    try {
      const out = await api.command("policy.propose", "policy", page.revision, reason, { max_age_minutes: String(maxAge), step_up_minutes: String(stepValue ?? 0) }, tried.current);
      onDone(out.fields.proposal ?? "");
    } catch (err) {
      if (err instanceof AdminError && err.failure === "invalid" && err.fields.length) {
        setProblems(err.fields.map((f) => (f === "step_up_minutes" ? { field: "policy-step", message: p.errStep(l, maxAge ?? 720) } : { field: "policy-age", message: p.errMaxAge(l) })));
      } else setError(err);
      if (err instanceof AdminError && err.failure === "stale") tried.current = attempt();
      if (settles(err)) onReload();
      setFocusKey((k) => k + 1);
    } finally {
      setBusy(false);
    }
  };
  const field = (id: string) => problems.find((x) => x.field === id)?.message;
  const minutesField = (id: string, label: string, value: string, set: (v: string) => void, hint: string) => (
    <div className="service-field live-minutes">
      <label htmlFor={id}>{label}</label>
      <p className="service-hint" id={id + "-hint"}>{hint}</p>
      {field(id) && <p className="service-field-error" id={id + "-error"}><span className="visually-hidden">{c.errorPrefix(l)} </span>{field(id)}</p>}
      <span className="live-unit">
        <input id={id} inputMode="numeric" dir="ltr" value={value} aria-invalid={field(id) ? true : undefined}
          aria-describedby={[id + "-hint", field(id) ? id + "-error" : ""].filter(Boolean).join(" ")} onChange={(e) => { setTouched(true); edit(set)(e.target.value); }} />
        <span>{p.minutes(l)}</span>
      </span>
    </div>
  );
  return (
    <form className="live-step" onSubmit={submit} noValidate>
      <h2>{p.proposeTitle(l)}</h2>
      <p className="live-impact">{p.proposeLead(l)}</p>
      {problems.length > 0 && <Summary locale={l} title={c.check(l)} problems={problems} focusKey={focusKey} />}
      {error !== null && <FailureBox l={l} error={error} focusKey={focusKey} describe={policyFailure} />}
      {minutesField("policy-age", p.maxAge(l), age, setAge, p.maxAgeHint(l))}
      <fieldset className="live-roles">
        <legend>{p.stepUp(l)}</legend>
        <label><input type="radio" name="policy-step-on" checked={!stepOn} onChange={() => { setTouched(true); edit(() => setStepOn(false))(""); }} />{p.stepNone(l)}</label>
        <label><input type="radio" name="policy-step-on" checked={stepOn} onChange={() => { setTouched(true); edit(() => setStepOn(true))(""); }} />{p.stepWithin(l)}</label>
      </fieldset>
      {stepOn && minutesField("policy-step", p.stepMinutes(l), step, setStep, p.stepHint(l))}
      {same && touched && <p className="live-note">{p.same(l)}</p>}
      <Reason l={l} value={reason} onChange={edit(setReason)} error={field("live-reason") ?? ""} />
      <div className="live-buttons">
        <button type="submit" className="live-primary" aria-busy={busy || undefined} disabled={busy}>{busy ? w.saving(l) : p.submit(l)}</button>
        {stepping && <button type="button" className="live-secondary" onClick={onCancel}>{w.cancel(l)}</button>}
      </div>
    </form>
  );
}

function Decide({ l, api, page, mode, onDone, onCancel, onReload }: {
  l: Locale; api: AdminClient; page: PolicyPage; mode: { kind: "approve" | "withdraw"; row: ProposalRow };
  onDone: (n: number, kind: "approve" | "withdraw") => void; onCancel: () => void; onReload: () => void;
}) {
  const { row, kind } = mode;
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [problem, setProblem] = useState("");
  const [focusKey, setFocusKey] = useState(0);
  const tried = useRef<Attempt>(attempt());
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => heading.current?.focus(), []);
  useEffect(() => { tried.current = attempt(); }, [page.revision]);
  const others = pending(page).filter((x) => x.number !== row.number).map((x) => String(x.number));
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    const r = reasonProblem(l, reason);
    setProblem(r);
    if (r) { setError(null); setFocusKey((k) => k + 1); return; }
    setBusy(true);
    setError(null);
    try {
      await api.command(kind === "approve" ? "policy.approve" : "policy.withdraw", row.target, page.revision, reason, {}, tried.current);
      onDone(row.number, kind);
    } catch (err) {
      setError(err);
      setFocusKey((k) => k + 1);
      if (err instanceof AdminError && err.failure === "stale") tried.current = attempt();
      if (settles(err)) onReload();
    } finally {
      setBusy(false);
    }
  };
  const title = kind === "approve" ? p.approveTitle(l, row.number) : p.withdrawTitle(l, row.number);
  return (
    <form className="live-step" onSubmit={submit} noValidate>
      <h2 tabIndex={-1} ref={heading}>{title}</h2>
      <p className="live-who">{p.proposedBy(l)} <bdi dir="ltr" className="live-mono">{row.proposer}</bdi></p>
      {problem && <Summary locale={l} title={c.check(l)} problems={[{ field: "live-reason", message: problem }]} focusKey={focusKey} />}
      {error !== null && <FailureBox l={l} error={error} focusKey={focusKey} describe={policyFailure} />}
      <Compare l={l} current={page.policy} next={row.revision} stacked />
      {kind === "approve" ? (
        <>
          <p className="live-impact">{p.approveNow(l, duration(l, row.revision.max_age_seconds))}</p>
          <p className="live-impact">{p.approveSelf(l)}</p>
          {others.length > 0 && <p className="live-impact live-strong">{p.approveDiscard(l, others.join(", "))}</p>}
        </>
      ) : (
        <p className="live-impact">{p.withdrawImpact(l)}</p>
      )}
      <Reason l={l} value={reason} onChange={(v) => { tried.current = attempt(); setReason(v); }} error={problem} />
      <div className="live-buttons">
        <button type="submit" className="live-primary" aria-busy={busy || undefined} disabled={busy}>{busy ? w.saving(l) : title}</button>
        <button type="button" className="live-secondary" onClick={onCancel}>{w.cancel(l)}</button>
      </div>
    </form>
  );
}

function Stale({ l, page, onSignOut }: { l: Locale; page: PolicyPage; onSignOut: () => void }) {
  return (
    <section className="live-step" aria-labelledby="policy-stale">
      <h2 id="policy-stale">{p.staleTitle(l)}</h2>
      <p className="live-impact">{p.staleBody(l, when(l, page.signed_in_at))}</p>
      <div className="live-buttons">
        <button type="button" className="live-primary" onClick={onSignOut}>{p.staleAction(l)}</button>
      </div>
    </section>
  );
}

function Unavailable({ l, title, body, action }: { l: Locale; title: string; body: string; action?: { label: string; go: () => void } }) {
  return (
    <div className="live-users">
      <section className="live-unavailable" aria-labelledby="policy-unavailable">
        <h2 id="policy-unavailable">{title}</h2>
        <p>{body}</p>
        {action && <button type="button" className="live-secondary" onClick={action.go}>{action.label}</button>}
      </section>
    </div>
  );
}

function Providers({ l, onView }: { l: Locale; onView: (view: string) => void }) {
  const rows: [string, boolean, string][] = [
    [p.password(l), true, p.passwordNote(l)],
    ["SAML", false, ""],
    ["OIDC", false, ""],
    [p.ldap(l), false, ""],
  ];
  return (
    <div className="live-users" data-live="providers">
      <section className="live-list" aria-labelledby="policy-methods">
        <h2 id="policy-methods" className="live-title">{p.methodsTitle(l)}</h2>
        <p className="live-impact">{p.methodsLead(l)}</p>
        <div className="security-table-scroll live-table">
          <table role="table">
            <thead role="rowgroup"><tr role="row"><th scope="col" role="columnheader">{p.colMethod(l)}</th><th scope="col" role="columnheader">{p.colStatus(l)}</th><th scope="col" role="columnheader">{p.colNotes(l)}</th></tr></thead>
            <tbody role="rowgroup">
              {rows.map(([name, on, note]) => (
                <tr key={name} role="row">
                  <td role="cell" className="live-user"><bdi>{name}</bdi></td>
                  <td role="cell" data-label={p.colStatus(l)} className={on ? "live-strong" : "live-quiet"}>{on ? p.active(l) : p.notAvail(l)}</td>
                  <td role="cell" data-label={p.colNotes(l)} className="live-quiet">{note || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className="live-buttons"><button type="button" className="live-secondary" onClick={() => onView("policies")}>{p.goPolicy(l)}</button></div>
      </section>
    </div>
  );
}

export default function LivePolicy({ locale: l, view, session, onSignOut, onView, onUsers }: {
  locale: Locale; view: string; session: Session; onSignOut: () => void; onView: (view: string) => void; onUsers: () => void;
}) {
  const api = useMemo(() => adminClient(session.csrf), [session.csrf]);
  const wide = useWide();
  const order = useRef(sequence());
  const [page, setPage] = useState<PolicyPage | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState<unknown>(null);
  const [mode, setMode] = useState<Mode | null>(null);
  const [notice, setNotice] = useState("");
  const [announce, setAnnounce] = useState("");
  const [focusTo, setFocusTo] = useState("");
  const [formKey, setFormKey] = useState(0);
  const afterLoad = useRef("");
  useEffect(() => {
    if (!focusTo) return;
    (document.querySelector<HTMLElement>(focusTo) ?? document.querySelector<HTMLElement>("#policy-pending"))?.focus();
    setFocusTo("");
  }, [focusTo]);
  const say = (text: string) => { setNotice(text); setAnnounce(""); setTimeout(() => setAnnounce(text), 50); };
  const load = useCallback(() => {
    const n = order.current.next();
    setStatus("loading");
    api.policy().then(
      (got) => {
        if (!order.current.latest(n)) return;
        setPage(got);
        setStatus("ready");
        setError(null);
        if (afterLoad.current) { setFocusTo(afterLoad.current); afterLoad.current = ""; }
      },
      (e) => { afterLoad.current = ""; if (order.current.latest(n)) { setError(e); setStatus("error"); } },
    );
  }, [api]);
  useEffect(() => { if (view === "policies") load(); }, [load, view]);
  if (view === "providers") return <Providers l={l} onView={onView} />;
  if (view === "test") return <Unavailable l={l} title={p.testTitle(l)} body={p.testBody(l)} />;
  if (view === "keys") return <Unavailable l={l} title={p.keysTitle(l)} body={p.keysBody(l)} />;
  if (view === "sync") return <Unavailable l={l} title={p.syncTitle(l)} body={p.syncBody(l)} action={{ label: p.goUsers(l), go: onUsers }} />;
  if (view === "recovery") return <Unavailable l={l} title={p.recoveryTitle(l)} body={p.recoveryBody(l)} action={{ label: p.goUsers(l), go: onUsers }} />;
  if (status === "error" && error instanceof AdminError && error.failure === "denied") return <Unavailable l={l} title={p.deniedTitle(l)} body={p.denied(l)} />;
  const late = page ? stale(page) : false;
  const back = (selector: string) => { setMode(null); setFocusTo(selector); };
  let panel: ReactNode = null;
  if (page && mode && mode.kind !== "propose") {
    panel = <Decide key={mode.kind + mode.row.number} l={l} api={api} page={page} mode={mode} onReload={load}
      onCancel={() => back(`[data-proposal="${mode.row.number}"][data-action="${mode.kind}"]`)}
      onDone={(n, kind) => { say(kind === "approve" ? p.doneApproved(l, n) : p.doneWithdrawn(l, n)); load(); back(kind === "approve" ? "#policy-current" : "#policy-pending"); }} />;
  } else if (page && late && (wide || mode)) {
    panel = <Stale l={l} page={page} onSignOut={onSignOut} />;
  } else if (page && (wide || mode?.kind === "propose")) {
    panel = <Propose key={`${page.policy?.number ?? 0}-${formKey}`} l={l} api={api} page={page} onReload={load} stepping={!wide} onCancel={() => back(".policy-propose")}
      onDone={(target) => { say(p.doneProposed(l, p.proposal(l, Number(target.replace("proposal-", ""))))); afterLoad.current = `#${target}`; setFormKey((k) => k + 1); setMode(null); load(); }} />;
  }
  const stepping = !wide && mode !== null;
  const rows = page ? pending(page) : [];
  const done = page ? history(page) : [];
  const current = page?.policy ?? null;
  return (
    <div className="live-users" data-live="policies" data-wide={wide ? "true" : "false"}>
      <p className="visually-hidden" role="status" aria-live="polite">{announce}</p>
      {!stepping && (
        <div className="live-list live-sections">
          <section aria-labelledby="policy-current">
            <div className="live-toolbar">
              <div className="live-heading">
                <h2 id="policy-current" tabIndex={-1}>{p.currentTitle(l)}</h2>
                {page && <p className="live-meta">{w.loadedAt(l, when(l, page.now))}</p>}
              </div>
              <div className="live-tools">
                {!wide && page && !late && <button type="button" className="live-primary policy-propose" onClick={() => { setNotice(""); setMode({ kind: "propose" }); }}>{p.proposeTitle(l)}</button>}
                <button type="button" className="live-secondary" onClick={() => { setNotice(""); load(); }}>{w.reload(l)}</button>
              </div>
            </div>
            {notice && <p className="service-status">{notice}</p>}
            {!wide && page && late && <Stale l={l} page={page} onSignOut={onSignOut} />}
            {status === "loading" && !page && <p role="status" className="live-meta">{p.loading(l)}</p>}
            {status === "error" && (
              <section className="live-error" role="alert">
                <h2>{p.loadFailed(l)}</h2>
                <p>{policyFailure(l, error).body}</p>
                {error instanceof AdminError && error.failure === "unauthenticated"
                  ? <button type="button" className="live-secondary" onClick={onSignOut}>{w.signInAgain(l)}</button>
                  : <button type="button" className="live-secondary" onClick={load}>{c.tryAgain(l)}</button>}
              </section>
            )}
            {page && !current && <p className="live-empty">{p.noPolicy(l)}</p>}
            {page && current && (
              <>
                <dl className="live-facts live-facts-grid">
                  <div><dt>{p.revision(l)}</dt><dd>{current.number}</dd></div>
                  <div><dt>{p.methods(l)}</dt><dd>{methodsText(l, current)}</dd></div>
                  <div><dt>{p.mfa(l)}</dt><dd>{current.mfa === "none" ? p.notRequired(l) : <span className="live-mono">{current.mfa}</span>}</dd></div>
                  <div><dt>{p.maxAge(l)}</dt><dd>{duration(l, current.max_age_seconds)}</dd></div>
                  <div><dt>{p.stepUp(l)}</dt><dd>{stepText(l, current)}</dd></div>
                  <div><dt>{p.proposedBy(l)}</dt><dd><bdi dir="ltr" className="live-mono">{current.proposer}</bdi></dd></div>
                  <div><dt>{p.approvedBy(l)}</dt><dd><bdi dir="ltr" className="live-mono">{current.approver}</bdi></dd></div>
                  <div><dt>{p.applied(l)}</dt><dd>{when(l, current.at)}</dd></div>
                </dl>
                {page.act_until && <p className="live-meta live-window">{p.actUntil(l, when(l, page.act_until))}</p>}
              </>
            )}
          </section>
          {page && (
            <section aria-labelledby="policy-pending">
              <div className="live-heading live-section-head">
                <h2 id="policy-pending" tabIndex={-1}>{p.pendingTitle(l)}</h2>
                <p className="live-meta">{p.pendingCount(l, rows.length)}</p>
              </div>
              {rows.length === 0 && <p className="live-empty">{p.noPending(l)}</p>}
              {rows.map((row) => {
                const act = canAct(page, row);
                const open = mode && mode.kind !== "propose" && mode.row.number === row.number;
                return (
                  <article key={row.number} className={"live-proposal" + (open ? " live-selected" : "")} aria-labelledby={row.target}>
                    <h3 id={row.target} tabIndex={-1}>{p.proposal(l, row.number)}{open && <span className="visually-hidden"> {w.selected(l)}</span>}</h3>
                    <p className="live-meta">{p.proposedBy(l)} <bdi dir="ltr" className="live-mono">{row.proposer}</bdi></p>
                    <Compare l={l} current={current} next={row.revision} />
                    {row.reason && <p className="live-reason-text"><span className="live-meta">{w.reason(l)}</span><br /><bdi>{row.reason}</bdi></p>}
                    <div className="live-tools">
                      {act.approve && <button type="button" className="live-secondary" data-proposal={row.number} data-action="approve" onClick={() => { setNotice(""); setMode({ kind: "approve", row }); }}>{p.approve(l)}</button>}
                      {act.withdraw && <button type="button" className="live-secondary" data-proposal={row.number} data-action="withdraw" onClick={() => { setNotice(""); setMode({ kind: "withdraw", row }); }}>{p.withdraw(l)}</button>}
                    </div>
                    {(late || act.note) && <p className="live-note">{late && !act.approve ? p.staleRow(l) : act.note === "needOther" ? p.needOther(l) : act.note === "staleRow" ? p.staleRow(l) : act.note ? p.cannotApprove(l) : ""}</p>}
                  </article>
                );
              })}
            </section>
          )}
          {page && (
            <section aria-labelledby="policy-history">
              <div className="live-heading live-section-head">
                <h2 id="policy-history">{p.historyTitle(l)}</h2>
                <p className="live-meta">{p.last10(l)}</p>
              </div>
              {done.length === 0 ? <p className="live-empty">{p.noHistory(l)}</p> : (
                <div className="security-table-scroll live-table live-history">
                  <table role="table">
                    <colgroup><col className="live-col-number" /><col /><col /><col /><col /></colgroup>
                    <thead role="rowgroup"><tr role="row">
                      <th scope="col" role="columnheader">{p.proposalColumn(l)}</th>
                      <th scope="col" role="columnheader">{p.maxAge(l)}</th>
                      <th scope="col" role="columnheader">{p.stepUp(l)}</th>
                      <th scope="col" role="columnheader">{p.proposedBy(l)}</th>
                      <th scope="col" role="columnheader">{p.approvedBy(l)}</th>
                    </tr></thead>
                    <tbody role="rowgroup">
                      {done.map((row) => (
                        <tr key={row.number} role="row">
                          <td role="cell" className="live-user">{p.proposal(l, row.number)}</td>
                          <td role="cell" data-label={p.maxAge(l)}>{duration(l, row.revision.max_age_seconds)}</td>
                          <td role="cell" data-label={p.stepUp(l)}>{stepText(l, row.revision)}</td>
                          <td role="cell" data-label={p.proposedBy(l)}><bdi dir="ltr" className="live-mono">{row.proposer}</bdi></td>
                          <td role="cell" data-label={p.approvedBy(l)}><bdi dir="ltr" className="live-mono">{row.approver}</bdi></td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </section>
          )}
        </div>
      )}
      {(wide || stepping) && panel && (
        <aside className="live-panel" aria-label={p.proposeTitle(l)}>
          {stepping && <button type="button" className="service-button-link live-back" onClick={() => back(".policy-propose")}>{p.back(l)}</button>}
          {panel}
        </aside>
      )}
    </div>
  );
}
