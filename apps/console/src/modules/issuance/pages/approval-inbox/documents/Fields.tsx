import { useEffect, useRef } from "react";
import type { Locale } from "../../../../../shared/types";
import type { DocumentBinding } from "./use-documents";
import { copy, type CopyKey } from "./copy";
export function Identifier({ children }: { children: string }) { return <bdi dir="ltr" className="b09-doc-id">{children}</bdi>; }
export function Failure({ locale, code }: { locale: Locale; code?: string }) { return code ? <div role="alert" className="b09-doc-feedback"><strong>{copy(locale,"failed")}</strong><p>{copy(locale,"errorHelp")}</p><Identifier>{code}</Identifier></div> : null; }
export function PacketFields({ locale: l, binding: b, prefix }: { locale: Locale; binding: DocumentBinding; prefix: string }) {
 const root = useRef<HTMLFormElement>(null), summary = useRef<HTMLDivElement>(null), heading = useRef<HTMLHeadingElement>(null), c = b.state.context.value, d = b.state.draft;
 useEffect(() => { if (b.state.invalidFields.length && root.current?.getClientRects().length) summary.current?.focus(); }, [b.state.invalidFields]);
 useEffect(() => { if (root.current?.getClientRects().length && !b.state.review) heading.current?.focus(); }, [c, b.state.review]);
 if (!c || !d || b.state.context.phase !== "ready" || b.state.review) return null;
 const invalid = (key: string) => b.state.invalidFields.includes(key), fieldId = (key: string) => `${prefix}-${encodeURIComponent(key)}`;
 const organizationLabels: Record<string,CopyKey> = { WorkspaceID:"workspace",IssuerLegalEntityID:"issuer",BuyerLegalEntityID:"buyer",PayerID:"payer",Currency:"currency",Path:"documentPath" };
 const labels: Record<string, CopyKey> = { erp:"erp",unit:"unit",evidence:"evidence",net:"net",tax:"tax",rationale:"rationale",quantity:"quantity",lines:"context",packet_size:"packet",context:"context" };
 const label = (key: string) => copy(l, key.startsWith("tax:") ? "lineTax" : labels[key.split(":")[0]] ?? "evidence");
 const order = ["erp","unit","evidence",...c.lines.flatMap(line => ["quantity","tax","evidence"].map(key => `${key}:${line.line}`)),"net","tax","rationale","lines","context","packet_size"];
 const errors = [...b.state.invalidFields].sort((a,b) => order.indexOf(a)-order.indexOf(b));
 const problem = (key: string) => {
  if (key === "erp") return copy(l,"chooseERP");
  const [kind,line] = key.split(":"), value = line ? d.lines[key.slice(kind.length+1)]?.[kind as "quantity"|"tax"|"evidence"] : d[key as "unit"|"evidence"|"net"|"tax"|"rationale"];
  if (!value?.trim()) return copy(l,"required");
  if (kind === "quantity") return copy(l,"invalidQuantity");
  if (kind === "tax" && line) return copy(l,"invalidTax");
  if (key === "net") return `${copy(l,"matchNet")}: ${c.header.currency} ${c.header.total}`;
  if (key === "tax") return copy(l,"matchTax");
  if (key === "rationale") return copy(l,"invalidReason");
  return copy(l,"invalidID");
 };
 const inlineError = (key: string) => invalid(key) ? <span id={fieldId(key)+"-error"} className="b09-doc-field-error">{problem(key)}</span> : null;
 function field(key: "unit" | "evidence" | "net" | "tax" | "rationale") {
  const props = { id: fieldId(key), "aria-labelledby": fieldId(key)+"-label", "aria-describedby": [key === "rationale" ? fieldId(key)+"-hint" : "", invalid(key) ? fieldId(key)+"-error" : ""].filter(Boolean).join(" ") || undefined, value: d![key], onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => b.store?.edit({ ...d!, [key]: e.target.value }), "aria-invalid": invalid(key) || undefined, required: true };
  return <label className="b09-doc-field" htmlFor={props.id} key={key}><span id={fieldId(key)+"-label"}>{label(key)}</span>{key === "rationale" ? <><textarea {...props} rows={3} maxLength={2000} /><small id={fieldId(key)+"-hint"}>{copy(l,"reasonLimit")}</small></> : <input {...props} dir="ltr" inputMode={key === "net" || key === "tax" ? "decimal" : "text"} maxLength={key === "net" || key === "tax" ? 64 : 128} />}{inlineError(key)}</label>;
 }
 return <form ref={root} className="b09-doc-form" noValidate onSubmit={e => { e.preventDefault(); b.store?.reviewProposal(); }}>
  <h4 ref={heading} tabIndex={-1}>{copy(l,"context")}</h4>
  <dl className="b09-doc-facts"><div><dt>{copy(l,"amount")}</dt><dd><Identifier>{`${c.header.currency} ${c.header.total}`}</Identifier></dd></div><div><dt>{copy(l,"month")}</dt><dd><Identifier>{c.header.period}</Identifier></dd></div><div><dt>{copy(l,"organization")}</dt><dd>{Object.entries(c.header.organization).filter(([key]) => !["Currency","Path"].includes(key)).map(([key,value]) => <div key={key}><span>{copy(l,organizationLabels[key])}: </span>{key === "Path" ? copy(l,"customerInvoice") : <Identifier>{value}</Identifier>}</div>)}</dd></div><div><dt>{copy(l,"currency")}</dt><dd><Identifier>{c.header.currency}</Identifier></dd></div><div><dt>{copy(l,"documentPath")}</dt><dd>{copy(l,"customerInvoice")}</dd></div></dl>
  <p className="b09-doc-input-help">{copy(l,"inputHelp")}</p><p>{copy(l,"requiredAll")} {copy(l,"scale")}: <Identifier>{String(c.header.erp.scale)}</Identifier></p>
  <p>{copy(l,"permission")}: <strong>{copy(l,c.permission.state)}</strong>{c.permission.code && <> · <Identifier>{c.permission.code}</Identifier></>}</p>
  {!!b.state.invalidFields.length && <div ref={summary} tabIndex={-1} role="alert" className="b09-doc-feedback"><strong>{copy(l,"invalid")}</strong><ul>{errors.map(key => <li key={key}><a href={`#${fieldId(key)}`} onClick={e => { e.preventDefault(); (document.getElementById(fieldId(key)) ?? root.current)?.focus(); }}>{label(key)}{key.includes(":") && <> · <span>{copy(l,"line")} {c.lines.findIndex(line => line.line === key.slice(key.indexOf(":")+1))+1}</span></>}</a></li>)}</ul></div>}
  <fieldset id={fieldId("erp")} tabIndex={-1}><legend>{copy(l,"erp")}</legend><label><input type="radio" name={`${prefix}-erp`} checked={d.selectedERP} onChange={() => b.store?.edit({ ...d, selectedERP: true })} aria-invalid={invalid("erp") || undefined} aria-describedby={invalid("erp") ? fieldId("erp")+"-error" : undefined} /><Identifier>{`${c.header.erp.connection} · ${c.header.erp.product} · ${c.header.erp.tax_code}`}</Identifier></label>{inlineError("erp")}</fieldset>
  <div className="b09-doc-field-grid">{field("unit")}{field("evidence")}</div>
  <div id={fieldId("lines")} tabIndex={-1} className="b09-doc-lines">{c.lines.map((line,i) => <fieldset key={line.line}><legend>{copy(l,"line")} {i+1}</legend><p className="b09-doc-line-id"><Identifier>{line.line}</Identifier></p><p>{copy(l,"amount")}: <Identifier>{`${c.header.currency} ${line.amount}`}</Identifier></p><div className="b09-doc-field-grid">{(["quantity","tax","evidence"] as const).map(key => { const name = `${key}:${line.line}`; return <label className="b09-doc-field" key={key} htmlFor={fieldId(name)}><span id={fieldId(name)+"-label"}>{copy(l,key === "tax" ? "lineTax" : key)}</span><input aria-labelledby={fieldId(name)+"-label"} id={fieldId(name)} dir="ltr" inputMode={key === "evidence" ? "text" : "decimal"} maxLength={key === "evidence" ? 128 : 64} value={d.lines[line.line]?.[key] ?? ""} required aria-invalid={invalid(name) || undefined} aria-describedby={invalid(name) ? fieldId(name)+"-error" : undefined} onChange={e => b.store?.edit({ ...d, lines: { ...d.lines, [line.line]: { ...d.lines[line.line], [key]: e.target.value } } })} />{inlineError(name)}</label>; })}</div></fieldset>)}</div>
  <div className="b09-doc-field-grid">{field("net")}{field("tax")}</div>{field("rationale")}
  <div id={fieldId("packet_size")} tabIndex={-1}><button type="submit" disabled={c.permission.state !== "allowed"}>{copy(l,"review")}</button></div>
 </form>;
}
