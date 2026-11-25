import { useEffect, useRef, useState, useSyncExternalStore, type ReactNode } from "react";
import { attempt } from "../../../../../service/admin";
import type { Session } from "../../../../../service/api";
import type { Locale } from "../../../../../shared/types";
import { day, ltr } from "../live-format";
import type { ProtectedSummary } from "./protected";
import type { ProtectedWriteIntent, ProtectedWriteReceipt } from "./protected-writes";
import { duration, rulesError, rulesText as r, type RulesWord } from "./rules-copy";
import { confirmRulesIntent, proposeRulesIntent, ruleOffers, type RuleDuration } from "./rules-flow";
import type { SetupScope } from "./scope";
import { codeOf } from "./StepResults";
import { stepText as t } from "./step-copy";
import type { StepWriteStore } from "./step-store";

type Form = "propose" | "confirm";
type Prepared = Readonly<{ intent: ProtectedWriteIntent; subject: string; action: RulesWord; facts: readonly (readonly [string, ReactNode])[]; notices: readonly RulesWord[]; form: Form }>;
export type ProtectedStore = StepWriteStore<ProtectedWriteIntent, ProtectedWriteReceipt>;
export const ruleActionOf: Readonly<Record<string, RulesWord>> = { propose_protected_rules: "actPropose", confirm_protected_rules: "actConfirm" };
type Props = { l: Locale; session: Session; store: ProtectedStore; scope: SetupScope; summary: ProtectedSummary; onSettled: () => void };
const durations: readonly (readonly [RuleDuration, RulesWord, RulesWord])[] = [["validity", "validity", "hintValidity"], ["lead", "lead", "hintLead"], ["channelAge", "channelAge", "hintChannelAge"]];

export function RuleSteps({ l, session, store, scope, summary, onSettled }: Props) {
 const state = useSyncExternalStore(store.subscribe, store.snapshot);
 const [form, setForm] = useState<Form | "">(""), [prepared, setPrepared] = useState<Prepared>(), [error, setError] = useState("");
 const heading = useRef<HTMLHeadingElement>(null), review = useRef<HTMLHeadingElement>(null), offers = ruleOffers(summary, session.user), rules = summary.rules;
 const name = (f: Form) => f + "-rules:" + summary.scope.contract + "/" + summary.scope.configuration, blocked = store.unresolved();
 useEffect(() => { if (prepared) review.current?.focus(); }, [prepared]);
 const prepare = (build: () => Prepared, invalid: string) => { setError(""); try { setPrepared(build()); } catch (e) { setPrepared(undefined); const code = codeOf(e); setError(code === "invalid_setup_request" ? invalid : code); heading.current?.focus(); } };
 const close = (f: Form) => { store.clear(name(f)); setPrepared(undefined); setForm(""); setError(""); heading.current?.focus(); };
 const send = async () => {
  if (!prepared) return;
  const current = prepared; setPrepared(undefined); heading.current?.focus();
  try { await store.submit(current.intent, current.subject, name(current.form)); } catch (e) { setError(codeOf(e)); return; }
  onSettled();
 };
 const settings: readonly (readonly [string, ReactNode])[] = [[r(l, "appliesTo"), r(l, "wholeInstallation")], [r(l, "reachedThrough"), day(l, scope.setupOn)]];
 const values = (approvals: string, validity: string, lead: string, age: string): (readonly [string, ReactNode])[] => [[r(l, "approvals"), ltr(approvals)], [r(l, "validity"), duration(l, validity)], [r(l, "lead"), duration(l, lead)], [r(l, "channelAge"), duration(l, age)]];
 const proposeStep = (): Prepared => { const id = attempt().request, intent = proposeRulesIntent(session, summary, id, store.draft(name("propose")), attempt()), a = intent.args; return { intent, subject: id, action: "actPropose", form: "propose", notices: ["noticeFirst"], facts: [...settings, ...values(a.approvals, a.validity_seconds, a.max_lead_seconds, a.channel_age_seconds), [r(l, "fieldReason"), <bdi>{intent.rationale}</bdi>]] }; };
 const confirmStep = (): Prepared => { const intent = confirmRulesIntent(session, summary, store.draft(name("confirm")).reason ?? "", attempt()); return { intent, subject: rules.id, action: "actConfirm", form: "confirm", notices: ["noticeConfirm"], facts: [...settings, ...values(rules.approvals, rules.validitySeconds, rules.maxLeadSeconds, rules.channelAgeSeconds), [r(l, "proposedBy"), ltr(rules.proposer)], [r(l, "confirmReason"), <bdi>{intent.rationale}</bdi>]] }; };
 const propose = store.draft(name("propose")), confirm = store.draft(name("confirm"));
 const edit = (f: Form, patch: Readonly<Record<string, string>>) => { setError(""); store.edit(name(f), patch); };
 const busy = state.entries.some(e => e.phase === "pending"), describe = blocked ? "o03-rule-blocked" : undefined;
 if (!offers.propose && offers.confirm === "none") return null;
 return <div className="o03-part-sub o03-steps" aria-busy={busy || undefined}>
  <h4 id="o03-rule-steps" ref={heading} tabIndex={-1}>{t(l, "actions")}</h4>
  <p className="live-sub">{r(l, "adminNote")} {t(l, "permissionNote")}</p>
  {blocked && <p id="o03-rule-blocked"><strong>{t(l, "blocked")}</strong></p>}
  {error && <div className="live-error" role="alert"><p>{rulesError(l, error)}</p><p className="live-mono">{t(l, "code")}: {ltr(error)}</p></div>}
  {prepared ? <div className="o03-setup-review">
   <h5 ref={review} tabIndex={-1}>{t(l, "reviewTitle")}</h5>
   <dl className="live-facts live-facts-grid"><div><dt>{t(l, "step")}</dt><dd>{r(l, prepared.action)}</dd></div>{prepared.facts.map(([label, value]) => <div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}</dl>
   {prepared.notices.map(key => <p key={key}><strong>{r(l, key)}</strong></p>)}
   <p className="live-sub">{t(l, "reviewNote")}</p>
   <div className="o03-docs-pages"><button type="button" className="live-primary" disabled={blocked} aria-describedby={describe} onClick={() => void send()}>{r(l, prepared.action)}</button><button type="button" className="live-secondary" onClick={() => { setPrepared(undefined); heading.current?.focus(); }}>{t(l, "back")}</button></div>
  </div> : form === "propose" ? <form className="o03-setup-form" noValidate onSubmit={e => { e.preventDefault(); prepare(proposeStep, "invalid_setup_request"); }}>
   <h5>{r(l, "actPropose")}</h5>
   <fieldset><legend>{r(l, "approvals")}</legend>{["1", "2", "3"].map(n => <label className="live-check" key={n}><input type="radio" name="o03-rule-approvals" checked={propose.approvals === n} onChange={() => edit("propose", { approvals: n })} />{ltr(n)}</label>)}</fieldset>
   {durations.map(([key, label, hint]) => <fieldset key={key}><legend>{r(l, label)}</legend>
    <p className="live-sub" id={"o03-rule-" + key + "-hint"}>{r(l, hint)}</p>
    <div className="o03-duration">
     <input className="o03-short" inputMode="numeric" autoComplete="off" aria-label={r(l, label)} aria-describedby={"o03-rule-" + key + "-hint"} aria-invalid={error === "invalid_setup_request" || undefined} value={propose[key] ?? ""} onChange={e => edit("propose", { [key]: e.target.value })} />
     <span className="live-sub">{r(l, "unitLegend")}</span>
     {(["hours", "days"] as const).map(unit => <label className="live-check" key={unit}><input type="radio" name={"o03-rule-" + key + "-unit"} aria-label={r(l, label) + ": " + r(l, unit)} checked={propose[key + "Unit"] === unit} onChange={() => edit("propose", { [key + "Unit"]: unit })} />{r(l, unit)}</label>)}
    </div>
   </fieldset>)}
   <label htmlFor="o03-rule-reason">{r(l, "fieldReason")}</label><textarea id="o03-rule-reason" maxLength={2000} aria-invalid={error === "invalid_setup_request" || undefined} value={propose.reason ?? ""} onChange={e => edit("propose", { reason: e.target.value })} />
   <p><strong>{r(l, "noticeFirst")}</strong></p>
   <div className="o03-docs-pages"><button type="submit" className="live-primary" disabled={blocked} aria-describedby={describe}>{t(l, "review")}</button><button type="button" className="live-secondary" onClick={() => close("propose")}>{t(l, "cancel")}</button></div>
  </form> : form === "confirm" ? <form className="o03-setup-form" noValidate onSubmit={e => { e.preventDefault(); prepare(confirmStep, "invalid_confirm"); }}>
   <h5>{r(l, "actConfirm")}</h5>
   <p><strong>{r(l, "noticeConfirm")}</strong></p>
   <label htmlFor="o03-rule-confirm-reason">{r(l, "confirmReason")}</label><textarea id="o03-rule-confirm-reason" maxLength={2000} aria-invalid={error === "invalid_confirm" || undefined} value={confirm.reason ?? ""} onChange={e => edit("confirm", { reason: e.target.value })} />
   <div className="o03-docs-pages"><button type="submit" className="live-primary" disabled={blocked} aria-describedby={describe}>{t(l, "review")}</button><button type="button" className="live-secondary" onClick={() => close("confirm")}>{t(l, "cancel")}</button></div>
  </form> : <div className="o03-docs-pages">
   {offers.propose && <button type="button" className="live-secondary" disabled={blocked} aria-describedby={describe} onClick={() => setForm("propose")}>{r(l, "actPropose")}</button>}
   {offers.confirm === "offer" && <button type="button" className="live-secondary" disabled={blocked} aria-describedby={describe} onClick={() => setForm("confirm")}>{r(l, "actConfirm")}</button>}
   {offers.confirm === "own" && <p>{r(l, "ownRules")}</p>}
  </div>}
 </div>;
}
