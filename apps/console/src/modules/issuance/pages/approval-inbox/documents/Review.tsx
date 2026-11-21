import { preparationSettled } from "./store";
import { useEffect, useRef } from "react";
import type { Locale } from "../../../../../shared/types";
import type { DocumentBinding } from "./use-documents";
import type { Packet } from "./packet";
import { copy } from "./copy";
import { Identifier } from "./Fields";
export function Review({ locale: l, binding: b }: { locale: Locale; binding: DocumentBinding }) {
 const heading = useRef<HTMLHeadingElement>(null);
 const i = b.state.review;
 useEffect(() => { if (i && heading.current?.getClientRects().length) heading.current.focus(); }, [i]);
 if (!i) return null;
 const p = i.action === "propose" ? JSON.parse(i.fields.packet) as Packet : undefined;
 return <section className="b09-doc-review" aria-label={copy(l,"review")}>
  <h4 ref={heading} tabIndex={-1}>{copy(l,"review")}: {copy(l,i.action)}</h4><p>{copy(l,"frozen")}</p>{i.action === "confirm" && <p>{copy(l,"confirmationBoundary")}</p>}
  <dl className="b09-doc-facts"><div><dt>{copy(l,"id")}</dt><dd><Identifier>{i.target}</Identifier></dd></div><div><dt>{copy(l,"rationale")}</dt><dd className="b09-doc-reason">{i.rationale}</dd></div></dl>
  {p ? <><dl className="b09-doc-facts"><div><dt>{copy(l,"net")}</dt><dd><Identifier>{`${b.state.context.value?.header.currency ?? ""} ${p.ExpectedNet}`}</Identifier></dd></div><div><dt>{copy(l,"tax")}</dt><dd><Identifier>{`${b.state.context.value?.header.currency ?? ""} ${p.ExpectedTax}`}</Identifier></dd></div><div><dt>{copy(l,"erp")}</dt><dd><Identifier>{p.Connection}</Identifier></dd></div><div><dt>{copy(l,"evidence")}</dt><dd><Identifier>{p.Evidence}</Identifier></dd></div></dl><ul className="b09-doc-list">{p.Lines.map((line,index) => <li key={line.Line}><h5>{copy(l,"line")} {index+1}</h5><p className="b09-doc-line-id"><Identifier>{line.Line}</Identifier></p><p>{copy(l,"amount")}: <Identifier>{`${b.state.context.value?.header.currency ?? ""} ${b.state.context.value?.lines.find(l => l.line === line.Line)?.amount ?? ""}`}</Identifier></p><dl className="b09-doc-facts">{([["product",line.Product],["taxCode",line.TaxCode],["quantity",line.Quantity],["unit",line.Unit],["lineTax",line.Tax],["evidence",line.Evidence]] as const).map(([key,value]) => <div key={key}><dt>{copy(l,key)}</dt><dd><Identifier>{key === "lineTax" ? `${b.state.context.value?.header.currency ?? ""} ${value}` : value}</Identifier></dd></div>)}</dl></li>)}</ul></> : <dl className="b09-doc-facts">{Object.entries(i.fields).map(([key,value]) => <div key={key}><dt>{copy(l,key === "document" ? "documentRef" : "digest")}</dt><dd><Identifier>{value}</Identifier></dd></div>)}</dl>}
  <details><summary>{copy(l,"packet")}</summary><pre dir="ltr">{p ? JSON.stringify(p,null,2) : JSON.stringify(i.fields,null,2)}</pre></details>
  <div className="b09-doc-actions"><button onClick={() => b.store?.cancelReview()}>{copy(l,"edit")}</button><button onClick={() => void b.store?.submit()} disabled={b.state.entries.some(e => e.phase === "pending" && e.scope.request === b.state.scope?.request)}>{copy(l,"submit")}</button></div>
 </section>;
}
export function Attempts({ locale: l, binding: b }: { locale: Locale; binding: DocumentBinding }) {
 const root = useRef<HTMLElement>(null), seen = useRef("");
 const signature = JSON.stringify(b.state.entries.map(e => [e.intent.attempt.key,e.phase]));
 useEffect(() => {
  const prior = new Map<string,string>(seen.current ? JSON.parse(seen.current) : []);
  const newPending = b.state.entries.some(e => e.phase === "pending" && prior.get(e.intent.attempt.key) !== "pending");
  if (seen.current && seen.current !== signature && root.current?.getClientRects().length && (newPending || root.current.contains(document.activeElement))) root.current.focus();
  seen.current = signature;
 }, [signature]);
 if (!b.state.entries.length) return null;
 const resolved = (key: string) => preparationSettled(b.state.entries,b.state.entries.find(e => e.intent.attempt.key === key)!);
 return <section ref={root} tabIndex={-1} className="b09-doc-attempts" aria-label={copy(l,"attempts")}><h2>{copy(l,"attempts")}</h2><p>{copy(l,"memory")}</p><ul className="b09-doc-list">{b.state.entries.map(e => <li key={e.intent.attempt.key} data-document-attempt={e.phase}>
  <h3>{copy(l,e.intent.action)}</h3><p role="status"><strong>{copy(l,e.phase === "pending" ? "sending" : e.phase)}</strong>{e.receipt && <> · {copy(l,e.receipt.state as "prepared" | "confirmed" | "pending" | "revoked")}</>}</p>
  <dl className="b09-doc-facts"><div><dt>{copy(l,"id")}</dt><dd><Identifier>{e.intent.target}</Identifier></dd></div><div><dt>{copy(l,"chargeRequest")}</dt><dd><Identifier>{e.scope.request}</Identifier></dd></div><div><dt>{copy(l,"rationale")}</dt><dd className="b09-doc-reason">{e.intent.rationale}</dd></div></dl>
  <dl className="b09-doc-facts b09-doc-references"><div><dt>{copy(l,"request")}</dt><dd><Identifier>{e.intent.attempt.request}</Identifier></dd></div>{e.receipt && <div><dt>{copy(l,"receipt")}</dt><dd><Identifier>{e.receipt.digest}</Identifier></dd></div>}</dl>{e.receipt?.replay && <p className="b09-doc-replay">{copy(l,"replay")}</p>}
  {e.error && <p>{copy(l,e.phase === "refused" ? e.error === "invalid_request" ? "invalidRefusal" : "refusedHelp" : "failed")}<span className="b09-doc-diagnostic"><Identifier>{e.error}</Identifier></span></p>}{e.operation && <p><Identifier>{e.operation}</Identifier></p>}
  {e.phase === "unknown" && resolved(e.intent.attempt.key) && <p>{copy(l,"recoveredOriginal")}</p>}
  {e.phase === "unknown" && !resolved(e.intent.attempt.key) && <><p>{copy(l,"unknownHelp")}</p><button disabled={b.state.entries.some(other => other.phase === "pending" && other.scope.request === e.scope.request)} onClick={() => void b.store?.retry(e.intent.attempt.key)}>{copy(l,"retry")}</button></>}
  {e.error === "reauthentication_required" && b.reauthenticate && <button onClick={() => void b.reauthenticate?.()}>{copy(l,"reauth")}</button>}
 </li>)}</ul></section>;
}
