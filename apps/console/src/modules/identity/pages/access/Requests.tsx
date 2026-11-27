import { sized } from "../../../../service/query-settings";
import { useEffect, useRef, useState, type FormEvent } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, attempt, type AccessSetting, type AdminClient, type Attempt } from "../../../../service/admin";
import type { ListedRequest } from "../../../../service/access-lists";
import { c } from "../../../../service/copy";
import { Summary } from "../../../../service/Service";
import { FailureBox, Reason, reasonProblem, settles } from "../users/Live";
import { w } from "../users/live-copy";
import { failureText } from "../users/live-model";
import { when } from "../authentication/live-model";
import { a, kindLabel } from "./live-copy";
import { accessText } from "./live-model";
import { q } from "./request-copy";
import { keepAttempt, requestState, shortDigest } from "./request-model";
import { qcopy } from "../service-accounts/query-copy";
import { ListFailure, Paging, Reference, addressCursor, addressSize, useCursorReset, usePagedList, type AddressChange } from "./paged";
import { SettingsDisclosure } from "./Settings";
import { breakable, withWho } from "./who";

export type Decision = "approve" | "reject" | "withdraw";

const mark = "\u0000who\u0000";

export function describeRequest(l: Locale, e: unknown): { title: string; body: string } {
  if (e instanceof AdminError) {
    if (e.failure === "refused") return { title: a.refusedTitle(l), body: q.refusedRequest(l) };
    if (e.failure === "stage") return { title: a.refusedTitle(l), body: q.stageRequest(l) };
    if (e.failure === "busy") return { title: w.saveFailed(l), body: q.busyIncrease(l) };
    if (e.failure === "partial") return { title: w.saveFailed(l), body: q.outcomeUnknown(l) };
    if (e.failure === "invalid") return { title: a.refusedTitle(l), body: a.invalidBody(l) };
    if (e.failure === "denied") return { title: a.refusedTitle(l), body: a.goneBody(l) };
    if (e.failure === "conflict" || e.failure === "stale") return { title: w.saveFailed(l), body: a.conflictBody(l) };
  }
  return failureText(l, e);
}

function scopeLine(l: Locale, r: ListedRequest): string {
  const what = `${accessText(l, r.action, r.fields)} · ${kindLabel(l, r.kind) ?? r.kind} · ${r.customer === "all" || r.customer === "" ? a.allCustomers(l) : r.customer}`;
  return r.change === "add" ? q.adds(l, what) : q.removes(l, what);
}

function Facts({ l, r }: { l: Locale; r: ListedRequest }) {
  return (
    <dl className="live-facts">
      <div><dt>{q.colChange(l)}</dt><dd>{r.change === "add" ? q.changeAdd(l) : q.changeRevoke(l)}</dd></div>
      <div><dt>{a.person(l)}</dt><dd><bdi dir="ltr">{r.subject_login || r.subject}</bdi></dd></div>
      <div><dt>{a.access(l)}</dt><dd className="live-strong">{scopeLine(l, r)}</dd>{r.change === "add" && <dd>{q.forDays(l, r.days)}</dd>}</div>
      <div><dt>{q.colRequest(l)}</dt><dd>{withWho(q.proposedBy(l, mark, when(l, r.proposed_at)), mark, r.proposer_login || r.proposer)}</dd><dd>{q.expires(l, when(l, r.expires_at))}</dd><dd>{r.reason}</dd></div>
      <div><dt>{q.revision(l)}</dt><dd className="live-mono access-wrap" dir="ltr">{r.digest}</dd></div>
    </dl>
  );
}

export function DecidePanel({ l, api, revision, request, action, onDone, onCancel, onReload }: {
  l: Locale; api: AdminClient; revision: string; request: ListedRequest; action: Decision; onDone: (message: string) => void; onCancel: () => void; onReload: () => void;
}) {
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [problem, setProblem] = useState("");
  const [focusKey, setFocusKey] = useState(0);
  const tried = useRef<Attempt>(attempt());
  const kept = useRef(false);
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => heading.current?.focus(), []);
  useEffect(() => {
    if (!kept.current) tried.current = attempt();
    kept.current = false;
  }, [revision]);
  const title = action === "approve" ? q.approveTitle(l) : action === "reject" ? q.rejectTitle(l) : q.withdrawTitle(l);
  const command = action === "approve" ? "access.approve_request" : action === "reject" ? "access.reject_request" : "access.withdraw_request";
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    const r = reasonProblem(l, reason);
    setProblem(r);
    if (r) { setError(null); setFocusKey((k) => k + 1); return; }
    setBusy(true);
    setError(null);
    try {
      await api.command(command, request.id, revision, reason, action === "approve" ? { digest: request.digest } : {}, tried.current);
      const login = request.subject_login || request.subject;
      onDone(action === "approve" ? (request.change === "add" ? q.doneApprovedAdd(l, login) : q.doneApprovedRevoke(l)) : action === "reject" ? q.doneRejected(l) : q.doneWithdrawn(l));
    } catch (err) {
      setError(err);
      setFocusKey((k) => k + 1);
      kept.current = keepAttempt(err);
      if (err instanceof AdminError && err.failure === "stale") tried.current = attempt();
      if (settles(err)) onReload();
    } finally {
      setBusy(false);
    }
  };
  return (
    <form className="live-step" onSubmit={submit} noValidate>
      <h2 tabIndex={-1} ref={heading}>{title}</h2>
      <p className="live-impact">{action === "approve" ? (request.change === "add" ? q.approveLead(l) : q.approveRevokeLead(l)) : action === "reject" ? q.rejectLead(l) : q.withdrawLead(l)}</p>
      {problem && <Summary locale={l} title={c.check(l)} problems={[{ field: "live-reason", message: problem }]} focusKey={focusKey} />}
      {error !== null && <FailureBox l={l} error={error} focusKey={focusKey} describe={describeRequest} />}
      <Facts l={l} r={request} />
      {action === "approve" && request.change === "revoke" && (
        <div className="access-impact-list">
          <h3>{q.mayEnd(l)}</h3>
          {request.may_end.length === 0 ? <p className="live-note">{q.mayEndNone(l)}</p> : <p className="live-mono" dir="ltr">{request.may_end.join(", ")}</p>}
        </div>
      )}
      <Reason l={l} value={reason} onChange={(x) => { tried.current = attempt(); setReason(x); }} error={problem} />
      <div className="live-buttons">
        <button type="submit" className="live-primary" aria-busy={busy || undefined} disabled={busy}>{busy ? w.saving(l) : title}</button>
        <button type="button" className="live-secondary" onClick={onCancel}>{w.cancel(l)}</button>
      </div>
    </form>
  );
}

function Rows({ l, rows, selected, onAct }: { l: Locale; rows: ListedRequest[]; selected: string; onAct?: (action: Decision, r: ListedRequest) => void }) {
  return (
    <div className={"security-table-scroll live-table live-access access-requests" + (onAct ? " access-requests-acting" : "")}>
      <table role="table">
        <colgroup><col className="req-col-a" /><col className="req-col-b" /><col className="req-col-c" />{onAct && <col className="req-col-d" />}</colgroup>
        <thead role="rowgroup"><tr role="row">
          <th scope="col" role="columnheader">{q.colChange(l)}</th>
          <th scope="col" role="columnheader">{a.colAccess(l)}</th>
          <th scope="col" role="columnheader">{a.colStatus(l)}</th>
          {onAct && <th scope="col" role="columnheader"><span className="visually-hidden">{a.colActions(l)}</span></th>}
        </tr></thead>
        <tbody role="rowgroup">
          {rows.map((r) => {
            const pending = r.state === "pending";
            return (
              <tr key={r.id} role="row" data-request={r.id} className={selected === r.id ? "live-selected" : undefined}>
                <td role="cell" className="live-user">
                  {r.change === "add" ? q.changeAdd(l) : q.changeRevoke(l)}
                  <span className="access-who"><bdi dir="ltr">{breakable(r.subject_login || r.subject)}</bdi></span>
                  <span className="live-sub">{withWho(q.proposedBy(l, mark, when(l, r.proposed_at)), mark, r.proposer_login || r.proposer)}</span>
                  <span className="live-sub live-mono" translate="no"><bdi dir="ltr">{r.id} · {shortDigest(r.digest)}</bdi></span>
                </td>
                <td role="cell" data-label={a.colAccess(l)}>{scopeLine(l, r)}{r.change === "add" && <span className="live-sub">{q.forDays(l, r.days)}</span>}</td>
                <td role="cell" data-label={a.colStatus(l)}>
                  <span className="live-strong">{requestState(l, r)}</span>
                  {pending && <span className="live-sub">{q.expires(l, when(l, r.expires_at))}</span>}
                  {pending && <span className="live-sub">{q.approvers(l, r.eligible_approvers)}</span>}
                  {pending && !r.current && <span className="live-sub live-strong">{q.outOfDate(l)}</span>}
                  {pending && r.current && r.eligible_approvers === 0 && <span className="live-sub live-strong">{q.noApprover(l)}</span>}
                  {!pending && r.decided_at && <span className="live-sub">{q.decidedBy(l, r.decided_by ? r.decided_by_login || r.decided_by : "—", when(l, r.decided_at))}</span>}
                  {r.decision_reason && <span className="live-sub">{r.decision_reason}</span>}
                </td>
                {onAct && (
                  <td role="cell"><div className="access-actions">
                    {r.can_approve && <button type="button" className="live-secondary" data-act="approve" onClick={() => onAct("approve", r)}>{q.approve(l)}</button>}
                    {r.can_reject && <button type="button" className="live-secondary" data-act="reject" onClick={() => onAct("reject", r)}>{q.reject(l)}</button>}
                    {r.can_withdraw && <button type="button" className="live-secondary" data-act="withdraw" onClick={() => onAct("withdraw", r)}>{q.withdraw(l)}</button>}
                  </div></td>
                )}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

export function RequestSection({ l, api, generation, pageLimit, startedAt, settings, selected, notice, onAct, onSignOut, onRefresh, params, onParams }: {
  l: Locale; api: AdminClient; generation: number; pageLimit: number | null; startedAt: string; settings: AccessSetting[]; selected: string; notice: string;
  onAct: (action: Decision, r: ListedRequest) => void; onSignOut: () => void; onRefresh: () => void; params: URLSearchParams; onParams: AddressChange;
}) {
  const endedSize = addressSize(params.get("req_el"), 25);
  const [openEnded, setOpenEnded] = useState(() => !!addressCursor(params.get("req_ea")));
  useCursorReset(generation, ["req_wa", "req_ea"], onParams);
  const pending = usePagedList((after) => api.accessRequests({ state: "pending", ...sized(50, pageLimit), ...(after ? { after } : {}) }), "pending/" + String(pageLimit), generation,
    { after: addressCursor(params.get("req_wa")), go: (next) => onParams({ req_wa: next || null }, true) });
  const ended = usePagedList((after) => api.accessRequests({ state: "ended", ...sized(endedSize, pageLimit), ...(after ? { after } : {}) }), "ended/" + String(pageLimit) + "/" + endedSize, generation,
    { after: addressCursor(params.get("req_ea")), go: (next) => onParams({ req_ea: next || null }, true) });
  const waiting = pending.read.state === "ready" ? pending.read.page : null;
  const past = ended.read.state === "ready" ? ended.read.page : null;
  const summary = ended.read.state === "error" ? q.endedFailed(l) : !past ? q.endedLoading(l)
    : past.count.certainty === "exact" ? q.endedRequests(l, Number(past.count.value)) : q.endedAtLeast(l, Number(past.count.value));
  const hidden = !!past && ended.first && past.rows.length === 0 && !past.next;
  return (
    <section aria-labelledby="access-requests" className="access-request-section">
      <div className="live-heading live-section-head">
        <h2 id="access-requests" tabIndex={-1}>{q.requestsTitle(l)}</h2>
        {waiting && <p className="live-meta" data-waiting={waiting.count.value}>{waiting.count.certainty === "exact" ? q.waiting(l, Number(waiting.count.value)) : q.waitingAtLeast(l, Number(waiting.count.value))}</p>}
      </div>
      {notice && <p className="service-status" data-request-notice="true">{notice}</p>}
      <p className="live-impact">{q.requestsLead(l)}</p>
      {pending.read.state === "loading" && <p role="status" className="live-meta">{a.loading(l)}</p>}
      <ListFailure l={l} title={q.requestsFailed(l)} list={pending} onSignOut={onSignOut} onRefresh={onRefresh} />
      {waiting && <Reference l={l} page={waiting} list={pending} />}
      {waiting && (waiting.rows.length === 0
        ? <p className="live-empty">{waiting.next ? qcopy(l, "incomplete") : q.noRequests(l)}</p>
        : <Rows l={l} rows={waiting.rows} selected={selected} onAct={onAct} />)}
      <Paging l={l} label={q.requestsTitle(l)} list={pending} />
      <p className="live-note">{q.notKept(l, when(l, startedAt))}</p>
      {!hidden && (
        <details className="live-ended access-ended-requests" open={openEnded || undefined} onToggle={(e) => setOpenEnded(e.currentTarget.open)}>
          <summary>{summary}</summary>
          {ended.read.state === "loading" && <p role="status" className="live-meta">{a.loading(l)}</p>}
          <ListFailure l={l} title={q.endedFailedTitle(l)} list={ended} onSignOut={onSignOut} onRefresh={onRefresh} />
          {past && <Reference l={l} page={past} list={ended} />}
          {past && (past.rows.length === 0 ? <p className="live-empty">{qcopy(l, past.next ? "incomplete" : "empty")}</p> : <Rows l={l} rows={past.rows} selected="" />)}
          <Paging l={l} label={summary} list={ended} size={{ id: "access-ended-size", requested: endedSize, limit: pageLimit, onChange: (n) => onParams({ req_el: n === 25 ? null : String(n), req_ea: null }, true) }} />
        </details>
      )}
      <SettingsDisclosure l={l} settings={settings} />
    </section>
  );
}
