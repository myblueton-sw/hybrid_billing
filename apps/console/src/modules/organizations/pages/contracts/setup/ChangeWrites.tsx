import { useEffect, useMemo, useRef, useState, useSyncExternalStore, type ReactNode } from "react";
import { attempt } from "../../../../../service/admin";
import type { Session } from "../../../../../service/api";
import type { Locale } from "../../../../../shared/types";
import { ltr, stamp } from "../live-format";
import type { AttestationPurpose } from "./attestation";
import { decideChangeIntent, requestChangeIntent, usableChannels, verifyChangeIntent } from "./change-flow";
import { changeError, changeText as c, type ChangeWord } from "./changes-copy";
import { reusableStatement, statementIntent } from "./channel-flow";
import { channelText, type ChannelWord } from "./channels-copy";
import { StatementFields, StatementReview, type StatementStore } from "./ChannelWrites";
import { setupReadClient } from "./client";
import { text } from "./copy";
import { PartReadStore } from "./part-store";
import type { ProtectedChange, ProtectedChannel, ProtectedField, ProtectedPage } from "./protected";
import type { ProtectedWriteIntent } from "./protected-writes";
import type { ProtectedStore } from "./RuleWrites";
import { sendChained, type ChainedStep } from "./statement-chain";
import { sampleLabel, statementError, statementText as st } from "./statements-copy";
import { codeOf } from "./StepResults";
import { stepText as t } from "./step-copy";

export type ChangeChoice = Readonly<{ row: ProtectedChange; mode: "verify" | "decide" }>;
type Facts = readonly (readonly [string, ReactNode])[];
type Prepared = Readonly<{ action: ChangeWord; facts: Facts; notices: readonly ChangeWord[]; purpose?: AttestationPurpose; chained?: ChainedStep; direct?: Readonly<{ intent: ProtectedWriteIntent; subject: string; form: string }> }>;
type Props = { l: Locale; session: Session; store: ProtectedStore; statements: StatementStore; field: ProtectedField; page: ProtectedPage<ProtectedChange>; choice?: ChangeChoice; onChoiceDone: () => void; onSettled: () => void };

export function ChangeSteps({ l, session, store, statements, field, page, choice, onChoiceDone, onSettled }: Props) {
 const state = useSyncExternalStore(store.subscribe, store.snapshot), claims = useSyncExternalStore(statements.subscribe, statements.snapshot);
 const needs = useMemo(() => { const client = setupReadClient(session); return new PartReadStore(async () => {
  const channels: ProtectedChannel[] = []; let after = "", snapshot = "";
  for (let i = 0; i < 10; i++) { const next = await client.protectedChannels(page.scope, after, snapshot); channels.push(...next.rows); if (!next.next) break; after = next.next; snapshot = next.snapshot; }
  return { channels, references: (await client.references(page.scope)).entries };
 }); }, [session.user, session.workspace, session.csrf, page.scope]);
 const loaded = useSyncExternalStore(needs.subscribe, needs.getSnapshot);
 const [form, setForm] = useState<"request" | "">(""), [prepared, setPrepared] = useState<Prepared>(), [error, setError] = useState(""), [source, setSource] = useState<"change" | "statement">("change");
 const heading = useRef<HTMLHeadingElement>(null), review = useRef<HTMLHeadingElement>(null);
 const scopeKey = page.scope.contract + "/" + page.scope.configuration, requestName = "request:" + scopeKey + "/" + field, choiceName = choice ? choice.mode + ":" + scopeKey + "/" + choice.row.change : "";
 const blocked = store.unresolved() || statements.unresolved(), describe = blocked ? "o03-change-blocked" : undefined;
 useEffect(() => { void needs.load(); return () => needs.dispose(); }, [needs]);
 useEffect(() => { if (prepared) review.current?.focus(); }, [prepared]);
 useEffect(() => { setForm(""); setPrepared(undefined); setError(""); }, [field, choice]);
 const channels = usableChannels(loaded.value?.channels ?? [], field, session.user), arrivals = usableChannels(loaded.value?.channels ?? [], field), samples = (loaded.value?.references ?? []).filter(e => e.use === field);
 const channelLabel = (ch: ProtectedChannel) => channelText(l, ch.subject_kind as ChannelWord) + " · " + channelText(l, ch.kind as ChannelWord) + " (" + c(l, "earliest") + ": " + (ch.request_not_before ? stamp(l, ch.request_not_before) : c(l, "notStated")) + ")";
 const reuseOf = (purpose: AttestationPurpose, target = "", digest = "") => reusableStatement(claims.entries, state.entries, page.scope, session.user, purpose, target, digest);
 const settingFact: readonly [string, ReactNode] = [c(l, "setting"), c(l, field)];
 const requestStep = (): Prepared => {
  const draft = store.draft(requestName), kept = reuseOf("change_instruction"), change = kept?.intent.target ?? attempt().request, id = kept?.intent.attestation ?? attempt().request, act = attempt();
  const statement = kept ? undefined : statementIntent(session, page, "change_instruction", id, change, "", draft, attempt());
  const build = (attestation: string, revision: string) => requestChangeIntent(session, page, revision, change, attestation, field, draft, loaded.value?.references ?? [], loaded.value?.channels ?? [], act);
  const a = build(id, page.revision).args, chosen = channels.find(ch => ch.channel === a.channel)!, via = a.instruction_channel ? arrivals.find(ch => ch.channel === a.instruction_channel) : undefined;
  return { action: "actRequest", purpose: "change_instruction", notices: ["noticeRequest"], chained: { subject: change, form: requestName, statement, build, reuse: kept && { id, text: kept.intent.statement, locator: kept.intent.locator } },
   facts: [settingFact, [c(l, "fieldReference"), sampleLabel(l, samples.find(e => e.code === a.reference)!)], [c(l, "fieldChannel"), channelLabel(chosen)], [c(l, "fieldProvenance"), c(l, a.provenance as ChangeWord)],
    [c(l, "fieldInstruction"), c(l, (a.instruction === "in_person" ? "in_person_form" : a.instruction) as ChangeWord)], ...(via ? [[c(l, "fieldInstructionChannel"), channelLabel(via)] as const] : []), [c(l, "fieldEffective"), stamp(l, a.effective)], [c(l, "fieldReason"), <bdi>{(draft.reason ?? "").trim()}</bdi>]] };
 };
 const named = (id: string): ReactNode => { const ch = loaded.value?.channels.find(v => v.channel === id); return <>{ch ? channelText(l, ch.subject_kind as ChannelWord) + " · " + channelText(l, ch.kind as ChannelWord) : c(l, "channelUnknown")}<span className="live-sub">{ltr(id)}</span></>; };
 const rowFacts = (r: ProtectedChange): Facts => [settingFact, [c(l, "effective"), stamp(l, r.effective)], [c(l, "evRequested"), ltr(r.requester)], [c(l, "channel"), named(r.channel)]];
 const verifyStep = (r: ProtectedChange): Prepared => {
  const draft = store.draft(choiceName), kept = reuseOf("change_verification", r.change, r.digest), id = kept?.intent.attestation ?? attempt().request, act = attempt();
  const statement = kept ? undefined : statementIntent(session, page, "change_verification", id, r.change, r.digest, draft, attempt());
  const build = (attestation: string, revision: string) => verifyChangeIntent(session, page, revision, r, attestation, draft.outcome ?? "", act);
  const a = build(id, page.revision).args;
  return { action: "actVerify", purpose: "change_verification", notices: a.outcome === "denied" ? ["noticeVerify"] : [], chained: { subject: r.change, form: choiceName, statement, build, reuse: kept && { id, text: kept.intent.statement, locator: kept.intent.locator } },
   facts: [...rowFacts(r), [c(l, "fieldOutcome"), c(l, ("outcome_" + a.outcome) as ChangeWord)]] };
 };
 const decideStep = (r: ProtectedChange): Prepared => {
  const intent = decideChangeIntent(session, page, r, store.draft(choiceName), attempt()), a = intent.args;
  return { action: "actDecide", notices: ["noticeDecide"], direct: { intent, subject: r.change, form: choiceName },
   facts: [...rowFacts(r), [c(l, "approvals"), ltr(r.approvals_required)], [c(l, "decisions"), ltr(String(r.decisions.length))], [c(l, "fieldVerdict"), c(l, a.verdict as ChangeWord)], ...(a.reason ? [[c(l, "fieldRejection"), c(l, ("reason_" + a.reason) as ChangeWord)] as const] : [])] };
 };
 const prepare = (build: () => Prepared, invalid: string) => { setError(""); try { setPrepared(build()); } catch (e) { const code = codeOf(e); setSource("change"); setError(code === "invalid_setup_request" ? invalid : code); setPrepared(undefined); heading.current?.focus(); } };
 const close = () => { store.clear(choice ? choiceName : requestName); setPrepared(undefined); setForm(""); setError(""); onChoiceDone(); heading.current?.focus(); };
 const send = async () => {
  if (!prepared) return;
  const current = prepared; setPrepared(undefined); setError(""); heading.current?.focus();
  try {
   if (current.direct) await store.submit(current.direct.intent, current.direct.subject, current.direct.form);
   else if (await sendChained(statements, store, current.chained!, page.revision) === "statement_unsettled") { onSettled(); return; }
  } catch (e) { setSource(current.chained?.statement && !statements.snapshot().entries.some(v => v.intent === current.chained!.statement) ? "statement" : "change"); setError(codeOf(e)); return; }
  setForm(""); onChoiceDone(); onSettled();
 };
 const draft = store.draft(choice ? choiceName : requestName), edit = (patch: Readonly<Record<string, string>>) => { setError(""); store.edit(choice ? choiceName : requestName, patch); };
 const busy = [...state.entries, ...claims.entries].some(e => e.phase === "pending");
 const keptRequest = reuseOf("change_instruction"), keptVerify = choice?.mode === "verify" ? reuseOf("change_verification", choice.row.change, choice.row.digest) : undefined;
 const kept = (entry: NonNullable<typeof keptRequest>) => <div className="o03-kept"><p>{st(l, "groupReuse")}</p><p className="o03-statement" dir="auto">{entry.intent.statement}</p><p className="live-sub">{st(l, "keptHint")}</p></div>;
 const radios = (name: string, key: string, values: readonly string[], label: (value: string) => string, extra: Readonly<Record<string, string>> = {}) => values.map(value => <label className="live-check" key={value}><input type="radio" name={name} checked={draft[key] === value} onChange={() => edit({ [key]: value, ...extra })} />{label(value)}</label>);
 const buttons = (invalid: boolean) => <div className="o03-docs-pages"><button type="submit" className="live-primary" disabled={blocked || invalid} aria-describedby={describe}>{t(l, "review")}</button><button type="button" className="live-secondary" onClick={close}>{t(l, "cancel")}</button></div>;
 const target = choice && <dl className="live-facts live-facts-grid">{rowFacts(choice.row).map(([label, value]) => <div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}</dl>;
 return <div className="o03-part-sub o03-steps" aria-busy={busy || undefined}>
  <h4 id="o03-change-steps" ref={heading} tabIndex={-1}>{t(l, "actions")}</h4>
  <p className="live-sub">{c(l, "stepNote")} {t(l, "permissionNote")}</p>
  {blocked && <p id="o03-change-blocked"><strong>{t(l, "blocked")}</strong></p>}
  {error && <div className="live-error" role="alert"><p>{source === "statement" ? statementError(l, error) : changeError(l, error)}</p><p className="live-mono">{t(l, "code")}: {ltr(error)}</p></div>}
  {prepared ? <div className="o03-setup-review">
   <h5 ref={review} tabIndex={-1}>{t(l, "reviewTitle")}</h5>
   {prepared.chained && <><StatementReview l={l} prepared={prepared.chained} /><h6>{st(l, "groupAction")}</h6></>}
   <dl className="live-facts live-facts-grid"><div><dt>{t(l, "step")}</dt><dd>{c(l, prepared.action)}</dd></div>{prepared.facts.map(([label, value]) => <div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}</dl>
   {prepared.chained && <><p><strong>{st(l, prepared.chained.reuse ? "oneRequest" : "twoRequests")}</strong></p><p><strong>{st(l, "noticeUsed")}</strong></p></>}
   {prepared.notices.map(key => <p key={key}><strong>{c(l, key)}</strong></p>)}
   <p className="live-sub">{t(l, "reviewNote")}</p>
   <div className="o03-docs-pages"><button type="button" className="live-primary" disabled={blocked} aria-describedby={describe} onClick={() => void send()}>{c(l, prepared.action)}</button><button type="button" className="live-secondary" onClick={() => { setPrepared(undefined); heading.current?.focus(); }}>{t(l, "back")}</button></div>
  </div> : choice?.mode === "verify" ? <form className="o03-setup-form" noValidate onSubmit={e => { e.preventDefault(); prepare(() => verifyStep(choice.row), "invalid_verify"); }}>
   <h5>{c(l, "actVerify")}</h5>{target}
   <fieldset><legend>{c(l, "fieldOutcome")}</legend>{radios("o03-change-outcome", "outcome", ["confirmed", "denied"], v => c(l, ("outcome_" + v) as ChangeWord))}</fieldset>
   {draft.outcome === "denied" && <p><strong>{c(l, "noticeVerify")}</strong></p>}
   {keptVerify ? kept(keptVerify) : <StatementFields l={l} id="o03-verify" draft={draft} invalid={error === "invalid_verify"} onEdit={edit} />}
   <p><strong>{st(l, "noticeUsed")}</strong></p>
   {buttons(false)}
  </form> : choice?.mode === "decide" ? <form className="o03-setup-form" noValidate onSubmit={e => { e.preventDefault(); prepare(() => decideStep(choice.row), "invalid_decide"); }}>
   <h5>{c(l, "actDecide")}</h5>{target}
   <fieldset><legend>{c(l, "fieldVerdict")}</legend>{radios("o03-change-verdict", "verdict", ["approve", "reject"], v => c(l, v as ChangeWord), { rejection: "" })}</fieldset>
   {draft.verdict === "reject" && <fieldset><legend>{c(l, "fieldRejection")}</legend>{radios("o03-change-rejection", "rejection", ["incorrect_details", "missing_evidence", "suspected_fraud", "other"], v => c(l, ("reason_" + v) as ChangeWord))}</fieldset>}
   <p><strong>{c(l, "noticeDecide")}</strong></p>
   {buttons(false)}
  </form> : form === "request" ? <form className="o03-setup-form" noValidate onSubmit={e => { e.preventDefault(); prepare(requestStep, "invalid_setup_request"); }}>
   <h5>{c(l, "actRequest")}: {c(l, field)}</h5>
   {loaded.loading && <p role="status">{text(l, "loading")}</p>}
   {loaded.error && <div className="live-error" role="alert"><p>{text(l, loaded.error === "permission_denied" ? "denied" : "failed")}</p><p className="live-mono">{ltr(loaded.error)}</p></div>}
   {loaded.value && <>
    <fieldset><legend>{c(l, "fieldReference")}</legend><p className="live-sub">{c(l, "sampleOnly")}</p>{radios("o03-change-reference", "reference", samples.map(e => e.code), code => sampleLabel(l, samples.find(e => e.code === code)!))}</fieldset>
    <label htmlFor="o03-change-channel">{c(l, "fieldChannel")}</label><p className="live-sub" id="o03-change-channel-hint">{c(l, "channelHint")}</p>
    <p className="live-sub">{c(l, "ownChannels")}</p>
    {channels.length === 0 ? <p>{c(l, "noChannels")}</p> : <select id="o03-change-channel" aria-describedby="o03-change-channel-hint" value={draft.channel ?? ""} onChange={e => edit({ channel: e.target.value, instructionChannel: "" })}><option value="">—</option>{channels.map(ch => <option key={ch.channel} value={ch.channel}>{channelLabel(ch)}</option>)}</select>}
    <fieldset><legend>{c(l, "fieldProvenance")}</legend>{radios("o03-change-provenance", "provenance", ["staff_entry", "customer_document", "customer_portal"], v => c(l, v as ChangeWord))}</fieldset>
    <fieldset><legend>{c(l, "fieldInstruction")}</legend>{radios("o03-change-instruction", "instruction", ["in_person", "registered_channel", "unregistered_sender"], v => c(l, (v === "in_person" ? "in_person_form" : v) as ChangeWord), { instructionChannel: "" })}</fieldset>
    {draft.instruction === "registered_channel" && <><label htmlFor="o03-change-instruction-channel">{c(l, "fieldInstructionChannel")}</label><p className="live-sub" id="o03-change-instruction-hint">{c(l, "instructionChannelHint")}</p>
     <select id="o03-change-instruction-channel" aria-describedby="o03-change-instruction-hint" value={draft.instructionChannel ?? ""} onChange={e => edit({ instructionChannel: e.target.value })}><option value="">—</option>{arrivals.filter(ch => ch.channel !== draft.channel).map(ch => <option key={ch.channel} value={ch.channel}>{channelLabel(ch)}</option>)}</select></>}
    <label htmlFor="o03-change-effective">{c(l, "fieldEffective")}</label><p className="live-sub" id="o03-change-effective-hint">{c(l, "effectiveHint")}</p>
    <input id="o03-change-effective" className="o03-short o03-datetime" type="datetime-local" aria-describedby="o03-change-effective-hint" value={draft.effective ?? ""} onChange={e => edit({ effective: e.target.value })} />
    <label htmlFor="o03-change-reason">{c(l, "fieldReason")}</label><textarea id="o03-change-reason" maxLength={2000} aria-invalid={error === "invalid_setup_request" || undefined} value={draft.reason ?? ""} onChange={e => edit({ reason: e.target.value })} />
    {keptRequest ? kept(keptRequest) : <StatementFields l={l} id="o03-request" draft={draft} invalid={error === "invalid_setup_request"} onEdit={edit} />}
    <p><strong>{st(l, "noticeUsed")}</strong></p>
   </>}
   {buttons(!loaded.value)}
  </form> : <div className="o03-docs-pages"><button type="button" className="live-secondary" disabled={blocked} aria-describedby={describe} onClick={() => setForm("request")}>{c(l, "actRequest")}: {c(l, field)}</button></div>}
 </div>;
}
