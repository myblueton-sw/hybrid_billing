import { useEffect, useMemo, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, type AdminClient } from "../../../../service/admin";
import type { OperationItems, OperationPage, SessionOperation } from "../../../../service/session-query";
import type { SelfEndRow } from "../../../../service/self-end";
import { rowOutcome, groupOf, groupOrder } from "./live-model";
import { e, whenSeconds } from "./end-copy";
import { m } from "./live-copy";
import { sq } from "./query-copy";
import { useScopedRead, type ScopedValue } from "./scoped-read";
import "./end.css";

export function summarySentence(l: Locale, op: SessionOperation): string {
  if (op.state === "unconfirmed" || op.ended_count === null || op.cleanup === null) return e.titleUnknown(l);
  if (op.ended_count === 0) return e.completeNone(l);
  if (op.state === "cleanup_running") return e.running(l);
  if (op.state === "cleanup_incomplete") return e.incomplete(l,op.ended_count)+" "+e.incompleteCount(l,op.cleanup.failed);
  return e.complete(l,op.ended_count);
}

export function CleanupGroups({ l,row }: { l: Locale; row: SelfEndRow }) {
  const labels = { done:e.groupDone,pending:e.groupPending,failed:e.groupFailed,not_applicable:e.groupNotApplicable,outside_scope:e.groupOutside };
  return <>{groupOrder.map(group => {
    const items = row.items.filter(item => groupOf(item) === group);
    return items.length > 0 ? <section key={group}><h5>{labels[group](l)}</h5><ul>{items.map(item => <li key={item.item}><strong>{e.item(l,item.item)}</strong>{item.state === "already_ended" ? " · "+e.countCleared(l) : ""}<p className="live-meta">{e.reason(l,item.reason)}</p></li>)}</ul></section> : null;
  })}</>;
}

function ResultRow({ l,row }: { l: Locale; row: SelfEndRow }) {
  const outcome = rowOutcome(row);
  const label = { gone:e.rowGone,complete:e.rowComplete,running:e.rowRunning,failed:e.rowFailed }[outcome](l);
  return <li><details><summary><bdi dir="ltr" className="live-mono">{row.handle}</bdi>{" · "+label}</summary>
    {row.result === "already_gone" ? <p>{e.nothingSession(l)}</p> : <CleanupGroups l={l} row={row} />}
  </details></li>;
}

function OperationRows({ l,api,op,generation,view,onEnd,locked,onSignedOut }: {
  l: Locale; api: AdminClient; op: SessionOperation; generation: string; view: "results" | "not_included"; onEnd: (handle: string) => void; locked: boolean; onSignedOut: () => void;
}) {
  const serial = useRef(0);
  const readKey = useMemo(() => ({ api,id:op.operation,generation,view }),[api,op.operation,generation,view]);
  const guard = useScopedRead(readKey);
  const [pageData,setPageData] = useState<ScopedValue<OperationItems> | null>(null);
  const page = pageData?.scope === guard.token ? pageData.value : null;
  const setPage = (value: OperationItems | null) => setPageData(value ? { scope:guard.token,value } : null);
  const [after,setAfter] = useState("");
  const [trail,setTrail] = useState<string[]>([]);
  const [busy,setBusy] = useState(false);
  const [failure,setFailure] = useState<unknown>(null);
  const load = (cursor = "",history: string[] = []) => {
    const n = ++serial.current;
    const valid = guard.begin();
    setBusy(true); setFailure(null); setPage(null);
    api.sessionOperationItems(op.operation,view,cursor ? { after:cursor } : {}).then(got => {
      if (n !== serial.current || !valid()) return;
      if (got.operation !== op.operation || got.view !== view) throw new AdminError("invalid");
      setPage(got); setAfter(cursor); setTrail(history); setBusy(false);
    }).catch(error => { if (n !== serial.current || !valid()) return; setBusy(false); setFailure(error); if (error instanceof AdminError && error.failure === "unauthenticated") onSignedOut(); });
  };
  useEffect(() => { load(); return () => { serial.current++; }; },[guard]);
  return <div className="mysec-operation-items" aria-busy={busy}>
    {busy && <p role="status">{e.reading(l)}</p>}
    {failure !== null && <div role="alert"><p>{failure instanceof AdminError && failure.code === "revision_conflict" ? sq.expired(l) : m.loadFailed(l)}</p><button type="button" className="live-secondary" onClick={() => load()}>{sq.first(l)}</button></div>}
    {page && <><p className="live-meta">{sq.shown(l,page.rows.length)} · {sq.viewCount(l,page.count)}</p>
      <ul className="mysec-result-rows">{page.rows.map(row => view === "results" ? <ResultRow key={row.handle} l={l} row={row as SelfEndRow} /> : <li key={row.handle}><bdi dir="ltr" className="live-mono">{row.handle}</bdi> · <bdi dir="ltr">{whenSeconds(l,row.signed_in)}</bdi>{" "}<button type="button" className="live-secondary" disabled={locked} onClick={() => onEnd(row.handle)}>{m.end(l)}</button></li>)}</ul>
      <div className="live-buttons"><button type="button" className="live-secondary" disabled={busy || trail.length === 0} onClick={() => load(trail[trail.length-1],trail.slice(0,-1))}>{sq.previous(l)}</button>
        <button type="button" className="live-secondary" disabled={busy || !page.next} onClick={() => load(page.next,[...trail,after].slice(-32))}>{sq.next(l)}</button></div></>}
  </div>;
}

function OperationCard({ l,api,op,generation,onEnd,locked,onSignedOut,idPrefix = "mysec-op" }: {
  l: Locale; api: AdminClient; op: SessionOperation; generation: string; onEnd: (handle: string) => void; locked: boolean; onSignedOut: () => void; idPrefix?: string;
}) {
  const [open,setOpen] = useState(false);
  const [view,setView] = useState<"results" | "not_included">("results");
  return <li><section className="mysec-op" aria-labelledby={idPrefix+"-"+op.operation}>
    <h4 id={idPrefix+"-"+op.operation} tabIndex={-1}>{op.kind === "one" ? e.opOne(l) : e.opOthers(l)}</h4>
    <p className="live-meta"><bdi dir="ltr">{op.operation}</bdi>{op.blocked_at ? " · "+whenSeconds(l,op.blocked_at) : ""}</p><p>{summarySentence(l,op)}</p>
    <p>{sq.recorded(l,op.recorded_count)}</p>{op.ended_count !== null && <p>{sq.access(l,op.ended_count)} · {sq.gone(l,op.gone_count!)}</p>}
    {op.cleanup && <p>{e.equation(l,op.ended_count!,op.cleanup.done,op.cleanup.already_ended,op.cleanup.pending,op.cleanup.failed)}</p>}
    {op.generation_changed === true && <p>{e.changedOnly(l)}</p>}
    {op.not_included_count !== null && op.not_included_count > 0 && <p>{sq.excluded(l,op.not_included_count)}</p>}
    <div className="live-buttons"><button type="button" className="live-secondary" aria-expanded={open && view === "results"} onClick={() => { setView("results");setOpen(current => view !== "results" || !current); }}>{sq.results(l)}</button>
      {op.kind === "others" && op.not_included_count !== null && op.not_included_count > 0 && <button type="button" className="live-secondary" aria-expanded={open && view === "not_included"} onClick={() => { setView("not_included");setOpen(current => view !== "not_included" || !current); }}>{sq.notIncluded(l)}</button>}</div>
    {open && op.state !== "unconfirmed" && <OperationRows l={l} api={api} op={op} generation={generation} view={view} onEnd={onEnd} locked={locked} onSignedOut={onSignedOut} />}
    {open && op.state === "unconfirmed" && <p role="status">{sq.resultsUnconfirmed(l)}</p>}
  </section></li>;
}

export default function SessionOperations({ l,api,refreshKey,onEnd,locked,onSignedOut }: {
  l: Locale; api: AdminClient; refreshKey: number; onEnd: (handle: string) => void; locked: boolean; onSignedOut: () => void;
}) {
  const serial = useRef(0);
  const pollUntil = useRef(0);
  const [stopped,setStopped] = useState(false);
  const readKey = useMemo(() => ({ api,refreshKey }),[api,refreshKey]);
  const guard = useScopedRead(readKey);
  const [pageData,setPageData] = useState<ScopedValue<OperationPage> | null>(null);
  const page = pageData?.scope === guard.token ? pageData.value : null;
  const setPage = (value: OperationPage | null) => setPageData(value ? { scope:guard.token,value } : null);
  const [after,setAfter] = useState("");
  const [trail,setTrail] = useState<string[]>([]);
  const [busy,setBusy] = useState(false);
  const [failure,setFailure] = useState<unknown>(null);
  const load = (cursor = "",history: string[] = [],poll = false) => {
    const n = ++serial.current;
    const valid = guard.begin();
    if (!poll) { pollUntil.current = Date.now()+30000; setStopped(false); setPage(null); }
    setBusy(true); setFailure(null);
    api.sessionOperations(cursor ? { after:cursor } : {}).then(got => {
      if (n !== serial.current || !valid()) return;
      setPage(got); setAfter(cursor); setTrail(history); setBusy(false);
    }).catch(error => { if (n !== serial.current || !valid()) return; setBusy(false); setFailure(error); if (error instanceof AdminError && error.failure === "unauthenticated") onSignedOut(); });
  };
  useEffect(() => { load(); return () => { serial.current++; }; },[guard]);
  useEffect(() => {
    if (!page || busy || after || failure || !page.rows.some(op => op.state === "cleanup_running")) return;
    if (Date.now() >= pollUntil.current) { setStopped(true); return; }
    const timer = setTimeout(() => load("",[],true),2000);
    return () => clearTimeout(timer);
  },[page,busy,after,failure]);
  return <section className="mysec-ends" aria-labelledby="mysec-ends-title" aria-busy={busy}>
    <div className="live-toolbar"><div><h3 id="mysec-ends-title">{e.endsTitle(l)}</h3>{page && <><p className="live-meta">{sq.totalOps(l,page.accepted_count)}</p><p className="mysec-checked">{e.checkedAt(l,whenSeconds(l,page.reference_at))}</p></>}</div><button type="button" className="live-secondary" disabled={busy} onClick={() => load()}>{e.checkAgain(l)}</button></div>
    {busy && <p role="status">{e.reading(l)}</p>}{failure !== null && <div role="alert"><p>{failure instanceof AdminError && failure.code === "revision_conflict" ? sq.expired(l) : m.loadFailed(l)}</p><button type="button" className="live-secondary" onClick={() => load()}>{sq.first(l)}</button></div>}
    {stopped && page && <p role="status">{e.stillRunning(l,whenSeconds(l,page.reference_at))}</p>}
    {page && <><ol className="mysec-ops">{page.rows.map(op => <OperationCard key={op.operation} l={l} api={api} op={op} generation={page.generation} onEnd={onEnd} locked={locked} onSignedOut={onSignedOut} />)}</ol>
      <div className="live-buttons"><button type="button" className="live-secondary" disabled={busy || trail.length === 0} onClick={() => load(trail[trail.length-1],trail.slice(0,-1))}>{sq.previous(l)}</button><button type="button" className="live-secondary" disabled={busy || !page.next} onClick={() => load(page.next,[...trail,after].slice(-32))}>{sq.next(l)}</button></div></>}
  </section>;
}

export function ReferencedOperation({ l,api,reference,onEnd,locked,onSignedOut }: {
  l: Locale; api: AdminClient; reference: string; onEnd: (handle: string) => void; locked: boolean; onSignedOut: () => void;
}) {
  const [reload,setReload] = useState(0);
  const key = useMemo(() => ({ api,reference,reload }),[api,reference,reload]);
  const guard = useScopedRead(key);
  const [data,setData] = useState<ScopedValue<{ op?: SessionOperation; failed?: "unavailable" | "unconfirmed" }> | null>(null);
  const value = data?.scope === guard.token ? data.value : null;
  const title = useRef<HTMLHeadingElement>(null);
  useEffect(() => {
    const valid = guard.begin();
    api.sessionOperation(reference).then(op => {
      if (!valid()) return;
      if (op.operation !== reference || op.kind !== "others") throw new AdminError("invalid");
      setData({ scope:guard.token,value:op.state === "unconfirmed" ? { failed:"unconfirmed" } : { op } });
    }).catch(error => { if (!valid()) return;setData({ scope:guard.token,value:{ failed:"unavailable" } });if (error instanceof AdminError && error.failure === "unauthenticated") onSignedOut(); });
  },[guard]);
  useEffect(() => { if (value) title.current?.focus(); },[value]);
  return <section className="mysec-referenced-operation" aria-labelledby="mysec-referenced-title"><h3 id="mysec-referenced-title" tabIndex={-1} ref={title}>{sq.viewOperation(l)}</h3>
    {!value && <p role="status">{e.reading(l)}</p>}{value?.failed && <p role={value.failed === "unconfirmed" ? "status" : "alert"}>{value.failed === "unconfirmed" ? sq.operationUnconfirmed(l) : sq.operationUnavailable(l)}</p>}
    <button type="button" className="live-secondary" disabled={!value} onClick={() => setReload(n => n+1)}>{e.checkAgain(l)}</button>
    {value?.op && <ol className="mysec-ops"><OperationCard idPrefix="mysec-referenced-op" l={l} api={api} op={value.op} generation={JSON.stringify([value.op.state,value.op.cleanup])} onEnd={onEnd} locked={locked} onSignedOut={onSignedOut} /></ol>}
  </section>;
}
