import { useEffect, useRef } from "react";
import type { Locale } from "../../../../../shared/types";
import type { DocumentBinding } from "./use-documents";
import type { Query } from "./store";
import type { CopyKey } from "./copy";
import { copy } from "./copy";
import { Failure, Identifier, PacketFields } from "./Fields";
import { Review } from "./Review";
import "./documents.css";
function State({ locale: l, query }: { locale: Locale; query: Query<unknown> }) { return query.phase === "failed" ? <Failure locale={l} code={query.error} /> : query.phase !== "ready" ? <p role="status">{copy(l,query.phase)}</p> : null; }
export default function DocumentPanel({ locale: l, binding: b, surface }: { locale: Locale; binding: DocumentBinding; surface: string }) {
 const actionReason = useRef<HTMLTextAreaElement>(null), previousReview = useRef<string | undefined>(undefined);
 useEffect(() => {
  if (previousReview.current && previousReview.current !== "propose" && !b.state.review && actionReason.current?.getClientRects().length) actionReason.current.focus();
  previousReview.current = b.state.review?.action;
 }, [b.state.review]);
 if (!b.enabled) return null;
 const s = b.state, d = s.document.value, prep = s.preparation.value, prefix = `b09-doc-${surface}`;
 const busy = s.entries.some(e => e.phase === "pending" && e.scope.request === s.scope?.request);
 const readHistory = () => { void b.store?.loadHistory("documents"); void b.store?.loadHistory("preparations"); };
 const readyPreparation = !!d && s.preparations.phase === "ready" && s.preparations.value?.revision === d.advice.revision;
 const recovery = !!prep && !!d && prep.row.document === d.inspection.row.document && prep.row.document_digest === d.inspection.row.digest && prep.row.actor === b.actor && !prep.row.recorded_receipt && prep.observation !== "conflict" && prep.revision === d.advice.revision;
 return <section className="b09-documents" data-document-panel>
  <h3>{copy(l,"title")}</h3><p>{copy(l,"boundary")}</p><p>{copy(l,"confirmationBoundary")}</p>
  <button onClick={readHistory} disabled={s.documents.phase === "loading" || s.preparations.phase === "loading"}>{copy(l,"histories")}</button>
  <div className="b09-doc-history-grid">{(["documents","preparations"] as const).map(kind => <section key={kind}><h4>{copy(l,kind)}</h4><p>{copy(l,"unchecked")}</p><State locale={l} query={s[kind]} />{s[kind].phase === "ready" && <>{!s[kind].value?.rows.length ? <p>{copy(l,"empty")}</p> : <ul className="b09-doc-list">{s[kind].value?.rows.map(row => "digest" in row ? <li key={row.document}><Identifier>{row.document}</Identifier><p>{copy(l,row.recorded_state)}</p><p>{copy(l,"actor")}: <Identifier>{row.proposed_by}</Identifier></p><button onClick={() => void b.store?.inspectDocument(row.document)}>{copy(l,"inspect")} · <Identifier>{row.document}</Identifier></button></li> : <li key={row.preparation}><Identifier>{row.preparation}</Identifier><p>{row.recorded_receipt ? copy(l,"recorded") : copy(l,"noReceipt")}</p><p>{copy(l,"documents")}: <Identifier>{row.document}</Identifier></p><button onClick={() => void b.store?.inspectPreparation(row.preparation)}>{copy(l,"inspect")} · <Identifier>{row.preparation}</Identifier></button></li>)}</ul>}</>}</section>)}</div>
  {s.document.phase !== "idle" && <State locale={l} query={s.document} />}
  {s.document.phase === "ready" && d && <section className="b09-doc-detail"><h4>{copy(l,"basis")} · <Identifier>{d.inspection.row.document}</Identifier></h4>
   <p>{copy(l,d.inspection.row.recorded_state)}</p><dl className="b09-doc-facts"><div><dt>{copy(l,"actor")}</dt><dd><Identifier>{d.inspection.row.proposed_by}</Identifier></dd></div><div><dt>{copy(l,"rationale")}</dt><dd className="b09-doc-reason">{d.inspection.reason}</dd></div><div><dt>{copy(l,"net")}</dt><dd><Identifier>{`${d.inspection.basis.Document.Currency} ${d.inspection.basis.ExpectedNet}`}</Identifier></dd></div><div><dt>{copy(l,"tax")}</dt><dd><Identifier>{`${d.inspection.basis.Document.Currency} ${d.inspection.basis.ExpectedTax}`}</Identifier></dd></div></dl>
   {(["confirmation","revocation"] as const).map(key => { const event = d.inspection[key]; return event ? <section key={key}><h5>{copy(l,key === "confirmation" ? "confirmed" : "revoked")}</h5><p><Identifier>{event.actor}</Identifier> · <Identifier>{event.at}</Identifier></p><p className="b09-doc-reason">{event.reason}</p></section> : null; })}
   <details><summary>{copy(l,"basis")}</summary><pre dir="ltr">{JSON.stringify(d.inspection.basis,null,2)}</pre></details>
   <p>{copy(l,"readinessScope")}</p><ul className="b09-doc-list">{(["confirm","revoke","prepare"] as const).map(action => <li key={action}><strong>{copy(l,action)}</strong><p>{copy(l,"permission")}: {copy(l,d.advice.permissions[action].state)}{d.advice.permissions[action].code && <> · <Identifier>{d.advice.permissions[action].code}</Identifier></>}</p><p>{copy(l,"readiness")}: {copy(l,d.advice.readiness[action].state === "unchecked" ? "uncheckedState" : d.advice.readiness[action].state as CopyKey)}{d.advice.readiness[action].code && <> · <Identifier>{d.advice.readiness[action].code}</Identifier></>}</p></li>)}</ul>
   {!s.review && <><label className="b09-doc-field" htmlFor={`${prefix}-action-reason`}><span id={`${prefix}-reason-label`}>{copy(l,"rationale")}</span><textarea aria-labelledby={`${prefix}-reason-label`} aria-describedby={`${prefix}-reason-hint`} ref={actionReason} id={`${prefix}-action-reason`} rows={3} maxLength={2000} value={b.rationale} onChange={e => b.setRationale(e.target.value)} /><small id={`${prefix}-reason-hint`}>{copy(l,"reasonLimit")}</small></label><div className="b09-doc-actions">{(["confirm","revoke","prepare"] as const).map(action => <button key={action} disabled={busy || !b.rationale.trim() || d.advice.permissions[action].state !== "allowed" || d.advice.readiness[action].state !== "ready" || action === "prepare" && !readyPreparation} onClick={() => b.store?.reviewAction(action,b.rationale)}>{copy(l,"review")} · {copy(l,action)}</button>)}</div></>}
  </section>}
  {s.preparation.phase !== "idle" && <State locale={l} query={s.preparation} />}
  {s.preparation.phase === "ready" && prep && <section className="b09-doc-detail"><h4>{copy(l,"preparations")} · <Identifier>{prep.row.preparation}</Identifier></h4><p>{copy(l,"observation")}: {copy(l,prep.observation)}</p><p>{copy(l,"observationHelp")}</p><p>{copy(l,"receipt")}: {prep.row.recorded_receipt ? <Identifier>{prep.row.recorded_receipt}</Identifier> : copy(l,"noReceipt")}</p><dl className="b09-doc-facts">{Object.entries(prep.total).map(([key,value]) => <div key={key}><dt>{copy(l,key === "Net" ? "net" : key === "Tax" ? "tax" : "quantity")}</dt><dd><Identifier>{key === "Quantity" ? value : `${d?.inspection.row.document === prep.row.document && d.inspection.row.digest === prep.row.document_digest ? d.inspection.basis.Document.Currency + " " : ""}${value}`}</Identifier></dd></div>)}</dl>{(!d || d.inspection.row.document !== prep.row.document || d.inspection.row.digest !== prep.row.document_digest) && <p>{copy(l,"currencyUnavailable")}</p>}<details><summary>{copy(l,"packet")}</summary><pre dir="ltr">{JSON.stringify(prep.submission,null,2)}</pre></details>{!prep.row.recorded_receipt && <><p>{copy(l,"recoveryHelp")}</p><button disabled={busy || !!s.review || !recovery || !readyPreparation || !b.rationale.trim() || d?.advice.readiness.prepare.state !== "ready"} onClick={() => b.store?.reviewAction("prepare",b.rationale,true)}>{copy(l,"review")} · {copy(l,"recovery")}</button></>}</section>}
  <Failure locale={l} code={s.error === "invalid_document_packet" ? undefined : s.error ?? b.error} />
  {[s.error,s.context.error,s.documents.error,s.preparations.error,s.document.error,s.preparation.error].includes("reauthentication_required") && b.reauthenticate && <button onClick={() => void b.reauthenticate?.()}>{copy(l,"reauth")}</button>}
  {!s.review && <button data-document-new aria-expanded={s.context.phase === "ready"} disabled={busy || s.context.phase === "loading"} onClick={() => void b.store?.loadContext()}>{copy(l,s.context.phase === "ready" ? "reloadContext" : "new")}</button>}
  {s.context.phase !== "idle" && <State locale={l} query={s.context} />}<PacketFields locale={l} binding={b} prefix={prefix} /><Review locale={l} binding={b} />
 </section>;
}
