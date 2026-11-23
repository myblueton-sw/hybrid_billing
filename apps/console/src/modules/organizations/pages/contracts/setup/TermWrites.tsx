import { useEffect, useRef, useState, useSyncExternalStore, type ReactNode } from "react";
import { attempt } from "../../../../../service/admin";
import type { Session } from "../../../../../service/api";
import type { Locale } from "../../../../../shared/types";
import { day, ltr } from "../live-format";
import type { SetupScope } from "./scope";
import type { StepWriteStore } from "./step-store";
import { approveBindingIntent, approveTermIntent, bindTermIntent, recordTermIntent, termOffers } from "./term-flow";
import { type PaymentTerms, type TermBinding, type TermWriteIntent, type TermWriteReceipt } from "./terms";
import { netDays, termError, termText as c, type TermWord } from "./terms-copy";
import { codeOf } from "./StepResults";
import { stepText as t } from "./step-copy";

type Form = "record" | "apply";
type Prepared = Readonly<{ intent: TermWriteIntent; subject: string; action: TermWord; facts: readonly (readonly [string, ReactNode])[]; notices: readonly TermWord[]; form?: Form }>;
export type TermStore = StepWriteStore<TermWriteIntent, TermWriteReceipt>;
export const termActionOf: Record<string, TermWord> = { record_term: "actRecord", approve_term: "actApprove", bind_term: "actApply", approve_term_binding: "actApproveApply" };

type StepsProps = { l: Locale; session: Session; store: TermStore; scope: SetupScope; page: PaymentTerms; row?: TermBinding; onRowDone: () => void; onSettled: () => void };
export function TermSteps({ l, session, store, scope, page, row, onRowDone, onSettled }: StepsProps) {
 const state = useSyncExternalStore(store.subscribe, store.snapshot);
 const [form, setForm] = useState<Form | "">(""), [prepared, setPrepared] = useState<Prepared>(), [error, setError] = useState("");
 const heading = useRef<HTMLHeadingElement>(null), review = useRef<HTMLHeadingElement>(null), offers = termOffers(page, session.user);
 const name = (f: Form) => f + ":" + page.scope.contract + "/" + page.scope.configuration;
 const blocked = store.unresolved();
 useEffect(() => { if (prepared) review.current?.focus(); }, [prepared]);
 const prepare = (build: () => Prepared, invalid = "invalid_setup_request") => { setError(""); try { setPrepared(build()); } catch (e) { setPrepared(undefined); const code = codeOf(e); setError(code === "invalid_setup_request" ? invalid : code); onRowDone(); heading.current?.focus(); } };
 useEffect(() => { if (row) prepare(() => bindingApproval(row)); }, [row]);
 const close = (f?: Form) => { if (f) store.clear(name(f)); setPrepared(undefined); setForm(""); setError(""); onRowDone(); heading.current?.focus(); };
 const send = async () => {
  if (!prepared) return;
  const current = prepared; setPrepared(undefined); onRowDone(); heading.current?.focus();
  try { await store.submit(current.intent, current.subject, current.form && name(current.form)); } catch (e) { setError(codeOf(e)); return; }
  onSettled();
 };
 const days = (value: string) => netDays(l, value), settings: readonly [string, ReactNode] = [t(l, "settings"), day(l, scope.setupOn)];
 const recordStep = (): Prepared => { const term = attempt().request, intent = recordTermIntent(session, page, term, store.draft(name("record")), attempt()); return { intent, subject: term, action: "actRecord", form: "record", notices: ["noticeDays"], facts: [settings, [c(l, "fieldDays"), days(intent.args.net_days)], [c(l, "fieldReason"), <bdi>{intent.rationale}</bdi>]] }; };
 const approveStep = (): Prepared => ({ intent: approveTermIntent(session, page, attempt()), subject: page.term!.id, action: "actApprove", notices: [], facts: [settings, [c(l, "fieldDays"), days(page.term!.netDays)], [c(l, "author"), ltr(page.term!.author)]] });
 const applyStep = (): Prepared => { const binding = attempt().request, intent = bindTermIntent(session, page, binding, store.draft(name("apply")), attempt()); return { intent, subject: binding, action: "actApply", form: "apply", notices: intent.args.to ? ["noticeBind"] : ["noticeBind", "openNotice"], facts: [settings, [c(l, "fieldDays"), days(page.term!.netDays)], [c(l, "fieldFrom"), day(l, intent.args.from)], [c(l, "endLegend"), intent.args.to ? day(l, intent.args.to) : c(l, "endOpen")], [c(l, "reasonLegend"), c(l, intent.args.reason as TermWord)], [c(l, "timeZone"), ltr(page.zone)]] }; };
 const bindingApproval = (r: TermBinding): Prepared => ({ intent: approveBindingIntent(session, page, r, attempt()), subject: r.binding, action: "actApproveApply", notices: [], facts: [settings, [c(l, "fieldDays"), days(r.net_days)], [c(l, "fieldFrom"), day(l, r.from)], [c(l, "endLegend"), r.to ? day(l, r.to) : c(l, "endOpen")], [c(l, "reasonLegend"), c(l, r.reason as TermWord)], [c(l, "appliedBy"), ltr(r.author)]] });
 const record = store.draft(name("record")), apply = store.draft(name("apply"));
 const edit = (f: Form, patch: Readonly<Record<string, string>>) => { setError(""); store.edit(name(f), patch); };
 const busy = state.entries.some(e => e.phase === "pending");
 return <div className="o03-part-sub o03-steps" aria-busy={busy || undefined}>
  <h4 id="o03-term-steps" ref={heading} tabIndex={-1}>{t(l, "actions")}</h4>
  <p className="live-sub">{t(l, "permissionNote")}</p>
  {blocked && <p id="o03-term-blocked"><strong>{t(l, "blocked")}</strong></p>}
  {error && <div className="live-error" role="alert"><p>{termError(l, error)}</p><p className="live-mono">{t(l, "code")}: {ltr(error)}</p></div>}
  {prepared ? <div className="o03-setup-review">
   <h5 ref={review} tabIndex={-1}>{t(l, "reviewTitle")}</h5>
   <dl className="live-facts live-facts-grid"><div><dt>{t(l, "step")}</dt><dd>{c(l, prepared.action)}</dd></div>{prepared.facts.map(([label, value]) => <div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}</dl>
   {prepared.notices.map(key => <p key={key}><strong>{c(l, key)}</strong></p>)}
   <p className="live-sub">{t(l, "reviewNote")}</p>
   <div className="o03-docs-pages"><button type="button" className="live-primary" disabled={blocked} aria-describedby={blocked ? "o03-term-blocked" : undefined} onClick={() => void send()}>{c(l, prepared.action)}</button><button type="button" className="live-secondary" onClick={() => { if (prepared.form) { setPrepared(undefined); heading.current?.focus(); } else close(); }}>{t(l, prepared.form ? "back" : "cancel")}</button></div>
  </div> : form === "record" ? <form className="o03-setup-form" noValidate onSubmit={e => { e.preventDefault(); prepare(recordStep); }}>
   <h5>{c(l, "actRecord")}</h5>
   <label htmlFor="o03-term-days">{c(l, "fieldDays")}</label><p className="live-sub" id="o03-term-days-hint">{c(l, "hintDays")}</p>
   <input id="o03-term-days" className="o03-short" inputMode="numeric" autoComplete="off" aria-describedby="o03-term-days-hint" aria-invalid={error === "invalid_setup_request" || undefined} value={record.days ?? ""} onChange={e => edit("record", { days: e.target.value })} />
   <label htmlFor="o03-term-reason">{c(l, "fieldReason")}</label><textarea id="o03-term-reason" maxLength={2000} aria-invalid={error === "invalid_setup_request" || undefined} value={record.reason ?? ""} onChange={e => edit("record", { reason: e.target.value })} />
   <p><strong>{c(l, "noticeDays")}</strong></p>
   <div className="o03-docs-pages"><button type="submit" className="live-primary" disabled={blocked} aria-describedby={blocked ? "o03-term-blocked" : undefined}>{t(l, "review")}</button><button type="button" className="live-secondary" onClick={() => close("record")}>{t(l, "cancel")}</button></div>
  </form> : form === "apply" ? <form className="o03-setup-form" noValidate onSubmit={e => { e.preventDefault(); prepare(applyStep, "invalid_apply"); }}>
   <h5>{c(l, "actApply")}</h5>
   <label htmlFor="o03-bind-from">{c(l, "fieldFrom")}</label><input id="o03-bind-from" className="o03-short" type="date" aria-invalid={error === "invalid_apply" || undefined} value={apply.from ?? ""} onChange={e => edit("apply", { from: e.target.value })} />
   <fieldset><legend>{c(l, "endLegend")}</legend>
    <label className="live-check"><input type="radio" name="o03-bind-end" checked={apply.end === "date"} onChange={() => edit("apply", { end: "date" })} />{c(l, "endDate")}</label>
    {apply.end === "date" && <><label htmlFor="o03-bind-to">{c(l, "fieldTo")}</label><input id="o03-bind-to" className="o03-short" type="date" value={apply.to ?? ""} onChange={e => edit("apply", { to: e.target.value })} /></>}
    <label className="live-check"><input type="radio" name="o03-bind-end" checked={apply.end === "open"} onChange={() => edit("apply", { end: "open", to: "" })} />{c(l, "endOpen")}</label>
    {apply.end === "open" && <p><strong>{c(l, "openNotice")}</strong></p>}
   </fieldset>
   <fieldset><legend>{c(l, "reasonLegend")}</legend>{(["initial", "renegotiation", "correction"] as const).map(key => <label className="live-check" key={key}><input type="radio" name="o03-bind-reason" checked={apply.reason === key} onChange={() => edit("apply", { reason: key })} />{c(l, key)}</label>)}</fieldset>
   <p>{c(l, "timeZone")}: {ltr(page.zone)}</p>
   <p><strong>{c(l, "noticeBind")}</strong></p>
   <div className="o03-docs-pages"><button type="submit" className="live-primary" disabled={blocked} aria-describedby={blocked ? "o03-term-blocked" : undefined}>{t(l, "review")}</button><button type="button" className="live-secondary" onClick={() => close("apply")}>{t(l, "cancel")}</button></div>
  </form> : <div className="o03-docs-pages">
   {offers.record && <button type="button" className="live-secondary" disabled={blocked} aria-describedby={blocked ? "o03-term-blocked" : undefined} onClick={() => setForm("record")}>{c(l, "actRecord")}</button>}
   {offers.approveTerm === "offer" && <button type="button" className="live-secondary" disabled={blocked} aria-describedby={blocked ? "o03-term-blocked" : undefined} onClick={() => prepare(approveStep)}>{c(l, "actApprove")}</button>}
   {offers.approveTerm === "own" && <p>{c(l, "ownTerm")}</p>}
   {offers.apply && <button type="button" className="live-secondary" disabled={blocked} aria-describedby={blocked ? "o03-term-blocked" : undefined} onClick={() => setForm("apply")}>{c(l, "actApply")}</button>}
  </div>}
 </div>;
}
