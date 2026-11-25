import { useEffect, useMemo, useRef, useState, useSyncExternalStore } from "react";
import type { Locale } from "../../../../../shared/types";
import type { Session } from "../../../../../service/api";
import { ltr, stamp } from "../live-format";
import { changeText as c, type ChangeWord } from "./changes-copy";
import { setupReadClient } from "./client";
import { text } from "./copy";
import { PageReadStore } from "./page-store";
import { partText } from "./part-copy";
import { PartReadStore } from "./part-store";
import { protectedFields, type ProtectedChange, type ProtectedField } from "./protected";
import { PartFeedback } from "./Rules";
import { rulesText } from "./rules-copy";
import type { SetupScope } from "./scope";
import { decideOffer, verifyOffer } from "./change-flow";
import { ChangeSteps, type ChangeChoice } from "./ChangeWrites";
import type { StatementStore } from "./ChannelWrites";
import type { ProtectedStore } from "./RuleWrites";

type Props = { l: Locale; session: Session; contract: string; scope: SetupScope; writes: ProtectedStore | undefined; statements: StatementStore | undefined; settled: number; onSignedOut: () => void };
const idle = () => () => undefined;

export function Changes({ l, session, contract, scope, writes, statements, settled, onSignedOut }: Props) {
 const [choice, setChoice] = useState<ChangeChoice>();
 const [field, setField] = useState<ProtectedField>("document_recipient");
 const target = useMemo(() => ({ contract, configuration: scope.configuration, digest: scope.digest }), [contract, scope.configuration, scope.digest]);
 const client = useMemo(() => setupReadClient(session), [session.user, session.workspace, session.csrf]);
 const summary = useMemo(() => new PartReadStore(() => client.protectedSummary(target)), [client, target]);
 const rows = useMemo(() => new PageReadStore((after, snapshot) => client.protectedChanges(target, field, after, snapshot)), [client, target, field]);
 const cards = useSyncExternalStore(summary.subscribe, summary.getSnapshot), state = useSyncExternalStore(rows.subscribe, rows.getSnapshot);
 const heading = useRef<HTMLHeadingElement>(null), table = useRef<HTMLHeadingElement>(null), shown = useRef(false);
 const signOut = useRef(onSignedOut); signOut.current = onSignedOut;
 useEffect(() => { void summary.load(); return () => summary.dispose(); }, [summary]);
 useEffect(() => { void rows.load(); if (shown.current) { shown.current = false; table.current?.focus(); } return () => rows.dispose(); }, [rows]);
 useEffect(() => { if ([cards.error, state.error].some(v => v === "session_changed" || v === "unauthenticated")) signOut.current(); }, [cards.error, state.error]);
 const reload = () => { heading.current?.focus(); void summary.load(); void rows.load(); };
 const first = useRef(true), unresolved = () => !!writes?.unresolved() || !!statements?.unresolved(), pending = () => [writes, statements].some(s => s?.snapshot().entries.some(e => e.phase === "pending"));
 const blockedA = useSyncExternalStore(writes?.subscribe ?? idle, unresolved), blockedB = useSyncExternalStore(statements?.subscribe ?? idle, unresolved), blocked = blockedA || blockedB;
 const sendingA = useSyncExternalStore(writes?.subscribe ?? idle, pending), sendingB = useSyncExternalStore(statements?.subscribe ?? idle, pending), sending = sendingA || sendingB;
 useEffect(() => { if (first.current) { first.current = false; return; } reload(); }, [settled]);
 useEffect(() => { setChoice(undefined); }, [state.value, field]);
 const choose = (row: ProtectedChange, mode: ChangeChoice["mode"]) => { setChoice({ row, mode }); document.getElementById("o03-change-steps")?.focus(); };
 const turn = (next = false) => { table.current?.focus(); void rows.load(next); };
 const page = state.value;
 return <section className="o03-part" aria-labelledby="o03-part-title" aria-busy={cards.loading || state.loading || undefined}>
  <div className="o03-docs-heading"><h3 id="o03-part-title" ref={heading} tabIndex={-1}>{partText(l, "changes")}</h3><button type="button" className="live-secondary" disabled={cards.loading || state.loading || sending} onClick={reload}>{text(l, "reload")}</button></div>
  <p>{c(l, "purpose")}</p>
  <PartFeedback l={l} state={cards} />
  {cards.value && <>
   <p className="live-meta">{rulesText(l, "readAt")}: {stamp(l, cards.value.at)} · {text(l, "eligibility")}</p>
   <ul className="o03-field-cards" aria-label={c(l, "fields")}>{protectedFields.map(key => {
    const f = cards.value!.fields[key];
    return <li key={key} aria-current={key === field ? "true" : undefined}>
     <h4>{c(l, key)}</h4>
     <dl className="live-facts">
      <div><dt>{c(l, "subject")}</dt><dd>{c(l, f.subjectKind as ChangeWord)}</dd></div>
      <div><dt>{c(l, "changes")}</dt><dd>{f.changes}</dd></div>
      {f.otherProvenance > 0 && <div><dt>{c(l, "other")}</dt><dd>{f.otherProvenance}</dd></div>}
      {f.unownedModule > 0 && <div><dt>{c(l, "unowned")}</dt><dd>{f.unownedModule}</dd></div>}
      <div><dt>{c(l, "latest")}</dt><dd>{f.latestScheduled ? <>{ltr(f.latestScheduled)}<span className="live-sub">{c(l, "effective")}: {stamp(l, f.latestScheduledEffective)}</span></> : c(l, "latestNone")}</dd></div>
     </dl>
     {f.latestScheduled && <p className="live-sub">{c(l, "latestNote")}</p>}
     {f.unownedModule > 0 && <p>{c(l, "blockingUnowned")}</p>}
     {key === field ? <p className="o03-docs-selected">{c(l, "shown")}</p> : <button type="button" className="live-secondary" aria-label={c(l, "show") + ": " + c(l, key)} onClick={() => { shown.current = true; setField(key); }}>{c(l, "show")}</button>}
    </li>;
   })}</ul>
  </>}
  <div className="o03-part-sub" aria-busy={state.loading || undefined}>
   <h4 id="o03-changes-title" ref={table} tabIndex={-1}>{c(l, "tableFor")}: {c(l, field)}</h4>
   <PartFeedback l={l} state={state} />
   {page && <>
    <p className="live-meta">{rulesText(l, "readAt")}: {stamp(l, page.at)} · {text(l, "eligibility")}</p>
    <dl className="live-facts live-facts-grid o03-part-counts">
     <div><dt>{c(l, "changes")}</dt><dd>{page.count}</dd></div>
     <div><dt>{c(l, "showing")}</dt><dd>{page.rows.length}</dd></div>
    </dl>
    {page.rows.length === 0 ? <p className="live-empty">{c(l, "empty")}</p> : <div className="live-table"><table className="o03-part-table o03-changes-table" aria-labelledby="o03-changes-title"><thead><tr>{(["change", "state", "effective", "channel", "action"] as const).map(key => <th key={key} scope="col">{c(l, key)}</th>)}</tr></thead><tbody>{page.rows.map(row => <ChangeRow key={row.change} l={l} row={row} user={writes && statements ? session.user : ""} blocked={blocked} onChoose={mode => choose(row, mode)} />)}</tbody></table></div>}
    <div className="o03-docs-pages">{state.after && <button type="button" className="live-secondary" disabled={sending} onClick={() => turn()}>{text(l, "first")}</button>}{page.next && <button type="button" className="live-secondary" disabled={sending} onClick={() => turn(true)}>{text(l, "next")}</button>}</div>
    {writes && statements && <ChangeSteps l={l} session={session} store={writes} statements={statements} field={field} page={page} choice={choice} onChoiceDone={() => setChoice(undefined)} onSettled={reload} />}
   </>}
  </div>
 </section>;
}

function ChangeRow({ l, row: r, user, blocked, onChoose }: { l: Locale; row: ProtectedChange; user: string; blocked: boolean; onChoose: (mode: ChangeChoice["mode"]) => void }) {
 const verify = user ? verifyOffer(r, user) : "none", decide = user ? decideOffer(r, user) : "none", described = blocked ? "o03-change-blocked" : undefined;
 const layer = (label: ChangeWord, value: React.ReactNode) => <span className="o03-part-line"><span className="live-sub">{c(l, label)}</span>{value}</span>;
 const event = (key: string, label: ChangeWord, actor: string, at: string) => <li key={key}>{c(l, label)}{actor && <>: {ltr(actor)}</>}<span className="live-sub">{at ? <>{c(l, "at")}: {stamp(l, at)}</> : c(l, "noTime")}</span></li>;
 const events = [
  ...(r.ended_at && !r.decisions.some(d => d.approver === r.ended_by && d.at === r.ended_at) ? [event("ended", r.ended_by ? "evEndedBy" : "evEnded", r.ended_by, r.ended_at)] : []),
  ...[...r.decisions].reverse().map((d, i) => event("decision-" + i, d.verdict === "approve" ? "evApproved" : "evRejected", d.approver, d.at)),
  ...(r.verifier ? [event("verified", r.verification_outcome === "confirmed" ? "evVerifiedConfirmed" : "evVerifiedDenied", r.verifier, r.verified_at)] : []),
  event("requested", r.observation === "absent" ? "evAttempted" : "evRequested", r.requester, r.requested_at),
 ];
 return <tr>
  <td data-label={c(l, "change")}>{c(l, r.provenance as ChangeWord)}<span className="live-sub">{c(l, "instruction")}: {c(l, r.instruction_kind as ChangeWord)}</span><details><summary>{text(l, "details")}</summary><p className="live-mono">{ltr(r.change)}</p><p className="live-mono">{ltr(r.digest)}</p></details></td>
  <td data-label={c(l, "state")}>
   {r.phase !== "recorded" && layer("phase", <span className="o03-tag">{c(l, r.phase as ChangeWord)}</span>)}
   {r.state && layer("recordedState", <span className="o03-tag">{c(l, r.state as ChangeWord)}</span>)}
   {layer("observed", c(l, r.observation as ChangeWord))}
   {r.pending_verifier && layer("pinnedVerifier", ltr(r.pending_verifier))}
   {r.pending_decision_actor && layer("pinnedDecider", ltr(r.pending_decision_actor))}
   {r.approvals_required && layer("approvals", ltr(r.approvals_required))}
   {r.verifier && layer("decisions", ltr(String(r.decisions.length)))}
   {r.rules === "other" && <span className="live-sub">{c(l, "otherRules")}</span>}
   {r.phase !== "recorded" && <p className="o03-part-notice">{c(l, "blockingUnknown")}</p>}
   <details><summary>{c(l, "events")}</summary><ol className="o03-events" reversed>{events}</ol></details>
  </td>
  <td data-label={c(l, "effective")}>{stamp(l, r.effective)}</td>
  <td data-label={c(l, "channel")}>{ltr(r.channel)}</td>
  <td data-label={c(l, "action")}>{verify === "offer" ? <button type="button" className="live-secondary" disabled={blocked} aria-describedby={described} aria-label={c(l, "actVerify") + ": " + r.change} onClick={() => onChoose("verify")}>{c(l, "actVerifyShort")}</button>
   : decide === "offer" ? <button type="button" className="live-secondary" disabled={blocked} aria-describedby={described} aria-label={c(l, "actDecide") + ": " + r.change} onClick={() => onChoose("decide")}>{c(l, "actDecideShort")}</button>
   : verify === "own" || decide === "requester" ? c(l, "ownRequest") : decide === "verifier" ? c(l, "ownVerify") : decide === "decided" ? c(l, "ownDecision") : null}</td>
 </tr>;
}
