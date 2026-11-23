import { useEffect, useMemo, useRef, useState, useSyncExternalStore, type ReactNode } from "react";
import { attempt } from "../../../../../service/admin";
import type { Session } from "../../../../../service/api";
import type { Locale } from "../../../../../shared/types";
import { ltr } from "../live-format";
import type { AttestationPurpose, AttestationReceipt } from "./attestation";
import { admitChannelIntent, channelOffer, channelReferences, registerChannelIntent, reusableStatement, statementIntent, type StatementStep } from "./channel-flow";
import { channelError, channelText as c, type ChannelWord } from "./channels-copy";
import { setupReadClient } from "./client";
import { text } from "./copy";
import { PartReadStore } from "./part-store";
import type { ProtectedChannel, ProtectedPage } from "./protected";
import type { ProtectedWriteIntent } from "./protected-writes";
import type { ProtectedStore } from "./RuleWrites";
import { sampleLabel, statementError, statementText as st } from "./statements-copy";
import { sendChained, type ChainOutcome } from "./statement-chain";
import { codeOf } from "./StepResults";
import { stepText as t } from "./step-copy";
import type { StepWriteStore } from "./step-store";

export type StatementStore = StepWriteStore<StatementStep, AttestationReceipt>;
type Prepared = Readonly<{
 action: ChannelWord; subject: string; form: string; purpose: AttestationPurpose; digest: string; reuse?: Readonly<{ id: string; text: string; locator: string }>;
 statement?: StatementStep; build: (attestation: string, revision: string) => ProtectedWriteIntent; facts: readonly (readonly [string, ReactNode])[]; notices: readonly ChannelWord[];
}>;
type Props = { l: Locale; session: Session; store: ProtectedStore; statements: StatementStore; page: ProtectedPage<ProtectedChannel>; row?: ProtectedChannel; onRowDone: () => void; onSettled: () => void };

export function StatementFields({ l, id, draft, invalid, onEdit }: { l: Locale; id: string; draft: Readonly<Record<string, string>>; invalid: boolean; onEdit: (patch: Readonly<Record<string, string>>) => void }) {
 return <fieldset><legend>{st(l, "groupFields")}</legend>
  <label htmlFor={id + "-statement"}>{st(l, "fieldStatement")}</label><p className="live-sub" id={id + "-statement-hint"}>{st(l, "hintStatement")}</p>
  <textarea id={id + "-statement"} maxLength={2000} aria-describedby={id + "-statement-hint"} aria-invalid={invalid || undefined} value={draft.statement ?? ""} onChange={e => onEdit({ statement: e.target.value })} />
  <label htmlFor={id + "-locator"}>{st(l, "fieldLocator")}</label><p className="live-sub" id={id + "-locator-hint"}>{st(l, "hintLocator")}</p>
  <input id={id + "-locator"} className="o03-wide" maxLength={512} autoComplete="off" aria-describedby={id + "-locator-hint"} value={draft.locator ?? ""} onChange={e => onEdit({ locator: e.target.value })} />
 </fieldset>;
}

export function StatementReview({ l, prepared }: { l: Locale; prepared: Pick<Prepared, "reuse" | "statement"> }) {
 const body = prepared.reuse ?? { text: prepared.statement!.statement, locator: prepared.statement!.locator };
 return <>
  <h6>{st(l, prepared.reuse ? "groupReuse" : "groupStatement")}</h6>
  <p className="o03-statement" dir="auto">{body.text}</p>
  <p className="live-sub">{st(l, "locator")}: {body.locator ? <bdi>{body.locator}</bdi> : st(l, "noLocator")}</p>
 </>;
}

export function ChannelSteps({ l, session, store, statements, page, row, onRowDone, onSettled }: Props) {
 const state = useSyncExternalStore(store.subscribe, store.snapshot), claims = useSyncExternalStore(statements.subscribe, statements.snapshot);
 const catalog = useMemo(() => { const client = setupReadClient(session); return new PartReadStore(() => client.references(page.scope)); }, [session.user, session.workspace, session.csrf, page.scope]);
 const samples = useSyncExternalStore(catalog.subscribe, catalog.getSnapshot);
 const [form, setForm] = useState<"register" | "">(""), [prepared, setPrepared] = useState<Prepared>(), [error, setError] = useState(""), [source, setSource] = useState<"channel" | "statement">("channel");
 const heading = useRef<HTMLHeadingElement>(null), review = useRef<HTMLHeadingElement>(null);
 const scopeKey = page.scope.contract + "/" + page.scope.configuration, registerName = "register:" + scopeKey, acceptName = (channel: string) => "accept:" + scopeKey + "/" + channel;
 const blocked = store.unresolved() || statements.unresolved(), describe = blocked ? "o03-channel-blocked" : undefined;
 useEffect(() => { void catalog.load(); return () => catalog.dispose(); }, [catalog]);
 useEffect(() => { if (prepared) review.current?.focus(); }, [prepared]);
 const fail = (e: unknown, invalid: string) => { const code = codeOf(e); setSource("channel"); setError(code === "invalid_setup_request" ? invalid : code); setPrepared(undefined); heading.current?.focus(); };
 const reuseOf = (purpose: AttestationPurpose, target = "", digest = "") => reusableStatement(claims.entries, state.entries, page.scope, session.user, purpose, target, digest);
 const registerStep = (): Prepared => {
  const draft = store.draft(registerName), kept = reuseOf("channel_registration"), channel = kept?.intent.target ?? attempt().request, id = kept?.intent.attestation ?? attempt().request, act = attempt();
  const statement = kept ? undefined : statementIntent(session, page, "channel_registration", id, channel, "", draft, attempt());
  const build = (attestation: string, revision: string) => registerChannelIntent(session, page, revision, channel, attestation, draft, samples.value?.entries ?? [], act);
  const intent = build(id, page.revision), sample = samples.value!.entries.find(e => e.code === intent.args.reference)!;
  return { action: "actRegister", subject: channel, form: registerName, purpose: "channel_registration", digest: "", statement, build, notices: [], reuse: kept && { id, text: kept.intent.statement, locator: kept.intent.locator },
   facts: [[c(l, "fieldSubject"), c(l, intent.args.subject_kind as ChannelWord)], [c(l, "fieldKind"), c(l, intent.args.kind as ChannelWord)], [c(l, "fieldReference"), sampleLabel(l, sample)], [c(l, "fieldReason"), <bdi>{intent.rationale}</bdi>]] };
 };
 const acceptStep = (r: ProtectedChannel): Prepared => {
  const draft = store.draft(acceptName(r.channel)), kept = reuseOf("channel_admission", r.channel, r.digest), id = kept?.intent.attestation ?? attempt().request, act = attempt();
  const statement = kept ? undefined : statementIntent(session, page, "channel_admission", id, r.channel, r.digest, draft, attempt());
  const build = (attestation: string, revision: string) => admitChannelIntent(session, page, revision, r, attestation, act);
  build(id, page.revision);
  return { action: "actAccept", subject: r.channel, form: acceptName(r.channel), purpose: "channel_admission", digest: r.digest, statement, build, notices: [], reuse: kept && { id, text: kept.intent.statement, locator: kept.intent.locator },
   facts: [[c(l, "channel"), <>{c(l, r.subject_kind as ChannelWord)} · {c(l, r.kind as ChannelWord)}</>], [c(l, "registeredBy"), ltr(r.registrar)]] };
 };
 const prepare = (build: () => Prepared, invalid: string) => { setError(""); try { setPrepared(build()); } catch (e) { fail(e, invalid); } };
 const close = () => { if (form === "register") store.clear(registerName); if (row) store.clear(acceptName(row.channel)); setPrepared(undefined); setForm(""); setError(""); onRowDone(); heading.current?.focus(); };
 const send = async () => {
  if (!prepared) return;
  const current = prepared; setPrepared(undefined); setError(""); heading.current?.focus();
  let outcome: ChainOutcome;
  try { outcome = await sendChained(statements, store, current, page.revision); } catch (e) { setSource(current.statement && !statements.snapshot().entries.some(v => v.intent === current.statement) ? "statement" : "channel"); setError(codeOf(e)); return; }
  if (outcome === "statement_unsettled") { onSettled(); return; }
  setForm(""); onRowDone(); onSettled();
 };
 const register = store.draft(registerName), accept = row ? store.draft(acceptName(row.channel)) : {};
 const edit = (name: string, patch: Readonly<Record<string, string>>) => { setError(""); store.edit(name, patch); };
 const busy = [...state.entries, ...claims.entries].some(e => e.phase === "pending"), options = channelReferences(samples.value?.entries ?? [], register.subject ?? "");
 const keptRegister = reuseOf("channel_registration"), keptAccept = row && reuseOf("channel_admission", row.channel, row.digest);
 const kept = (entry: NonNullable<typeof keptRegister>) => <div className="o03-kept"><p>{st(l, "groupReuse")}</p><p className="o03-statement" dir="auto">{entry.intent.statement}</p><p className="live-sub">{st(l, "keptHint")}</p></div>;
 return <div className="o03-part-sub o03-steps" aria-busy={busy || undefined}>
  <h4 id="o03-channel-steps" ref={heading} tabIndex={-1}>{t(l, "actions")}</h4>
  <p className="live-sub">{c(l, "stepNote")} {t(l, "permissionNote")}</p>
  {blocked && <p id="o03-channel-blocked"><strong>{t(l, "blocked")}</strong></p>}
  {error && <div className="live-error" role="alert"><p>{source === "statement" ? statementError(l, error) : channelError(l, error)}</p><p className="live-mono">{t(l, "code")}: {ltr(error)}</p></div>}
  {prepared ? <div className="o03-setup-review">
   <h5 ref={review} tabIndex={-1}>{t(l, "reviewTitle")}</h5>
   <StatementReview l={l} prepared={prepared} />
   <h6>{st(l, "groupAction")}</h6>
   <dl className="live-facts live-facts-grid"><div><dt>{t(l, "step")}</dt><dd>{c(l, prepared.action)}</dd></div>{prepared.facts.map(([label, value]) => <div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}</dl>
   <p><strong>{st(l, prepared.reuse ? "oneRequest" : "twoRequests")}</strong></p>
   <p><strong>{st(l, "noticeUsed")}</strong></p>
   <p className="live-sub">{t(l, "reviewNote")}</p>
   <div className="o03-docs-pages"><button type="button" className="live-primary" disabled={blocked} aria-describedby={describe} onClick={() => void send()}>{c(l, prepared.action)}</button><button type="button" className="live-secondary" onClick={() => { setPrepared(undefined); heading.current?.focus(); }}>{t(l, "back")}</button></div>
  </div> : row ? <form className="o03-setup-form" noValidate onSubmit={e => { e.preventDefault(); prepare(() => acceptStep(row), "invalid_accept"); }}>
   <h5>{c(l, "actAccept")}</h5>
   <dl className="live-facts live-facts-grid"><div><dt>{c(l, "channel")}</dt><dd>{c(l, row.subject_kind as ChannelWord)} · {c(l, row.kind as ChannelWord)}</dd></div><div><dt>{c(l, "registeredBy")}</dt><dd>{ltr(row.registrar)}</dd></div></dl>
   {keptAccept ? kept(keptAccept) : <StatementFields l={l} id="o03-accept" draft={accept} invalid={error === "invalid_accept"} onEdit={patch => edit(acceptName(row.channel), patch)} />}
   <p><strong>{st(l, "noticeUsed")}</strong></p>
   <div className="o03-docs-pages"><button type="submit" className="live-primary" disabled={blocked} aria-describedby={describe}>{t(l, "review")}</button><button type="button" className="live-secondary" onClick={close}>{t(l, "cancel")}</button></div>
  </form> : form === "register" ? <form className="o03-setup-form" noValidate onSubmit={e => { e.preventDefault(); prepare(registerStep, "invalid_setup_request"); }}>
   <h5>{c(l, "actRegister")}</h5>
   <fieldset><legend>{c(l, "fieldSubject")}</legend>{(["customer", "entity", "document_kind"] as const).map(key => <label className="live-check" key={key}><input type="radio" name="o03-channel-subject" checked={register.subject === key} onChange={() => edit(registerName, { subject: key, reference: "" })} />{c(l, key)}</label>)}</fieldset>
   <fieldset><legend>{c(l, "fieldKind")}</legend>{(["email", "phone", "portal", "postal"] as const).map(key => <label className="live-check" key={key}><input type="radio" name="o03-channel-kind" checked={register.kind === key} onChange={() => edit(registerName, { kind: key })} />{c(l, key)}</label>)}</fieldset>
   <fieldset><legend>{c(l, "fieldReference")}</legend><p className="live-sub">{c(l, "sampleOnly")}</p>
    {samples.error ? <p role="alert">{text(l, "failed")} <span className="live-mono">{ltr(samples.error)}</span></p> : !samples.value ? <p role="status">{text(l, "loading")}</p> : !register.subject ? <p>{c(l, "subjectFirst")}</p> : options.map(option => <label className="live-check" key={option.code}><input type="radio" name="o03-channel-reference" checked={register.reference === option.code} onChange={() => edit(registerName, { reference: option.code })} />{sampleLabel(l, option)}</label>)}
   </fieldset>
   <label htmlFor="o03-register-reason">{c(l, "fieldReason")}</label><textarea id="o03-register-reason" maxLength={2000} aria-invalid={error === "invalid_setup_request" || undefined} value={register.reason ?? ""} onChange={e => edit(registerName, { reason: e.target.value })} />
   {keptRegister ? kept(keptRegister) : <StatementFields l={l} id="o03-register" draft={register} invalid={error === "invalid_setup_request"} onEdit={patch => edit(registerName, patch)} />}
   <p><strong>{st(l, "noticeUsed")}</strong></p>
   <div className="o03-docs-pages"><button type="submit" className="live-primary" disabled={blocked || !samples.value} aria-describedby={describe}>{t(l, "review")}</button><button type="button" className="live-secondary" onClick={close}>{t(l, "cancel")}</button></div>
  </form> : <div className="o03-docs-pages"><button type="button" className="live-secondary" disabled={blocked} aria-describedby={describe} onClick={() => setForm("register")}>{c(l, "actRegister")}</button></div>}
 </div>;
}

export { channelOffer };
