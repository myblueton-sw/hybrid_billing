import { useEffect, useMemo, useRef, useState, useSyncExternalStore } from "react";
import type { Locale } from "../../../../../shared/types";
import type { Session } from "../../../../../service/api";
import { ltr, stamp } from "../live-format";
import { channelText as c, type ChannelWord } from "./channels-copy";
import { setupReadClient } from "./client";
import { text } from "./copy";
import { PageReadStore } from "./page-store";
import { partText } from "./part-copy";
import type { ProtectedChannel } from "./protected";
import { PartFeedback } from "./Rules";
import { rulesText } from "./rules-copy";
import type { SetupScope } from "./scope";
import { ChannelSteps, channelOffer, type StatementStore } from "./ChannelWrites";
import type { Offer } from "./channel-flow";
import type { ProtectedStore } from "./RuleWrites";

type Props = { l: Locale; session: Session; contract: string; scope: SetupScope; writes: ProtectedStore | undefined; statements: StatementStore | undefined; settled: number; onSignedOut: () => void };
const idle = () => () => undefined;

export function Channels({ l, session, contract, scope, writes, statements, settled, onSignedOut }: Props) {
 const [chosen, setChosen] = useState<ProtectedChannel>();
 const store = useMemo(() => { const client = setupReadClient(session); return new PageReadStore((after, snapshot) => client.protectedChannels({ contract, configuration: scope.configuration, digest: scope.digest }, after, snapshot)); }, [session.user, session.workspace, session.csrf, contract, scope.configuration, scope.digest]);
 const state = useSyncExternalStore(store.subscribe, store.getSnapshot), heading = useRef<HTMLHeadingElement>(null);
 const signOut = useRef(onSignedOut); signOut.current = onSignedOut;
 useEffect(() => { void store.load(); return () => store.dispose(); }, [store]);
 useEffect(() => { if (state.error === "session_changed" || state.error === "unauthenticated") signOut.current(); }, [state.error]);
 const load = (next = false) => { heading.current?.focus(); void store.load(next); };
 const first = useRef(true), unresolved = () => !!writes?.unresolved() || !!statements?.unresolved();
 const blockedA = useSyncExternalStore(writes?.subscribe ?? idle, unresolved), blockedB = useSyncExternalStore(statements?.subscribe ?? idle, unresolved), blocked = blockedA || blockedB;
 const sending = [useSyncExternalStore(writes?.subscribe ?? idle, () => !!writes?.snapshot().entries.some(e => e.phase === "pending")), useSyncExternalStore(statements?.subscribe ?? idle, () => !!statements?.snapshot().entries.some(e => e.phase === "pending"))].some(Boolean);
 useEffect(() => { if (first.current) { first.current = false; return; } load(); }, [settled]);
 useEffect(() => { setChosen(undefined); }, [state.value]);
 const page = state.value;
 return <section className="o03-part" aria-labelledby="o03-part-title" aria-busy={state.loading || undefined}>
  <div className="o03-docs-heading"><h3 id="o03-part-title" ref={heading} tabIndex={-1}>{partText(l, "channels")}</h3><button type="button" className="live-secondary" disabled={state.loading || sending} onClick={() => load()}>{text(l, "reload")}</button></div>
  <p>{c(l, "purpose")}</p>
  <PartFeedback l={l} state={state} />
  {page && <>
   <p className="live-meta">{rulesText(l, "readAt")}: {stamp(l, page.at)} · {text(l, "eligibility")}</p>
   <dl className="live-facts live-facts-grid o03-part-counts">
    <div><dt>{c(l, "total")}</dt><dd>{page.count}</dd></div>
    <div><dt>{c(l, "showing")}</dt><dd>{page.rows.length}</dd></div>
    {page.otherProvenance > 0 && <div><dt>{c(l, "other")}</dt><dd>{page.otherProvenance}</dd></div>}
    {page.unownedModule > 0 && <div><dt>{c(l, "unowned")}</dt><dd>{page.unownedModule}</dd></div>}
   </dl>
   {page.rows.length > 0 && <p className="live-sub">{c(l, "earliestNote")}</p>}
   {page.rows.length === 0 ? <p className="live-empty">{c(l, "empty")}</p> : <div className="live-table"><table className="o03-part-table"><thead><tr>{(["channel", "state", "people", "earliest", "action"] as const).map(key => <th key={key} scope="col">{c(l, key)}</th>)}</tr></thead><tbody>{page.rows.map(row => <ChannelRow key={row.channel} l={l} row={row} offer={writes && statements ? channelOffer(row, session.user) : "none"} blocked={blocked} onAccept={() => { setChosen(row); document.getElementById("o03-channel-steps")?.focus(); }} />)}</tbody></table></div>}
   <div className="o03-docs-pages">{state.after && <button type="button" className="live-secondary" disabled={sending} onClick={() => load()}>{text(l, "first")}</button>}{page.next && <button type="button" className="live-secondary" disabled={sending} onClick={() => load(true)}>{text(l, "next")}</button>}</div>
   {writes && statements && <ChannelSteps l={l} session={session} store={writes} statements={statements} page={page} row={chosen} onRowDone={() => setChosen(undefined)} onSettled={() => load()} />}
  </>}
 </section>;
}

function ChannelRow({ l, row: r, offer, blocked, onAccept }: { l: Locale; row: ProtectedChannel; offer: Offer; blocked: boolean; onAccept: () => void }) {
 const line = (label: ChannelWord, actor: string, at: ChannelWord, instant: string) => actor || instant ? <span className="o03-part-line">{actor && <>{c(l, label)}: {ltr(actor)}</>}{instant && <span className="live-sub">{c(l, at)}: {stamp(l, instant)}</span>}</span> : null;
 return <tr>
  <td data-label={c(l, "channel")}>{c(l, r.subject_kind as ChannelWord)} · {c(l, r.kind as ChannelWord)}<details><summary>{text(l, "details")}</summary><p className="live-mono">{ltr(r.channel)}</p><p className="live-mono">{ltr(r.digest)}</p></details></td>
  <td data-label={c(l, "state")}>
   <span className="o03-part-line"><span className="live-sub">{c(l, "recordedState")}</span><span className="o03-tag">{c(l, r.state as ChannelWord)}</span></span>
   <span className="o03-part-line"><span className="live-sub">{c(l, "observed")}</span>{c(l, ("observed_" + r.observation) as ChannelWord)}</span>
   {r.pending_admitter && <span className="o03-part-line"><span className="live-sub">{c(l, "pinned")}</span>{ltr(r.pending_admitter)}</span>}
   {r.rules === "other" && <span className="live-sub">{c(l, "otherRules")}</span>}
  </td>
  <td data-label={c(l, "people")}>{line("registeredBy", r.registrar, "registeredAt", r.registered_at)}{line("acceptedBy", r.admitter, "acceptedAt", r.admitted_at)}{line("retiredBy", r.retired_by, "retiredAt", r.retired_at)}</td>
  <td data-label={c(l, "earliest")}>{r.request_not_before ? stamp(l, r.request_not_before) : c(l, "notStated")}</td>
  <td data-label={c(l, "action")}>{offer === "offer" ? <button type="button" className="live-secondary" disabled={blocked} aria-describedby={blocked ? "o03-channel-blocked" : undefined} aria-label={c(l, "actAccept") + ": " + r.channel} onClick={onAccept}>{c(l, "actAcceptShort")}</button> : offer === "own" ? c(l, "ownChannel") : null}</td>
 </tr>;
}
