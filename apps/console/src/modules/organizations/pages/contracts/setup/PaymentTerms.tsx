import { useEffect, useMemo, useRef, useState, useSyncExternalStore, type ReactNode } from "react";
import type { Locale } from "../../../../../shared/types";
import type { Session } from "../../../../../service/api";
import { day, ltr, stamp } from "../live-format";
import { setupReadClient } from "./client";
import { text } from "./copy";
import { PageReadStore } from "./page-store";
import { ScopeBar } from "./Parts";
import { PartReadStore } from "./part-store";
import type { ReadState } from "./read-store";
import { rulesText } from "./rules-copy";
import { sessionScopeStore, type SetupScope } from "./scope";
import type { PaymentTerm, TermBinding } from "./terms";
import { netDays, termText as c, type TermWord } from "./terms-copy";
import { termOffers, type TermOffer } from "./term-flow";
import { TermSteps, termActionOf, type TermStore } from "./TermWrites";
import { termError } from "./terms-copy";
import { StepResults, useStepStore } from "./StepResults";
import { stepError } from "./step-copy";
import { termWriteClient, type TermWriteIntent, type TermWriteReceipt } from "./terms";
import "./documents.css";

type Props = { l: Locale; session: Session; contract: string; on: string; onSignedOut: () => void };

function settingsHref(): string {
 if (typeof location === "undefined") return "#";
 const [path, query = ""] = location.hash.replace(/^#/, "").split("?"), q = new URLSearchParams(query);
 q.set("view", "documents"); q.delete("part");
 return "#" + path + "?" + q.toString();
}

export function PaymentTerms({ l, session, contract, on, onSignedOut }: Props) {
 const client = useMemo(() => setupReadClient(session), [session.user, session.workspace, session.csrf]);
 const scopes = useMemo(() => sessionScopeStore(session, contract, on), [session.user, session.workspace, session.csrf, contract, on]);
 const history = useMemo(() => new PartReadStore(() => client.configurations(contract)), [client, contract]);
 const scope = useSyncExternalStore(scopes.subscribe, scopes.getSnapshot), found = useSyncExternalStore(history.subscribe, history.getSnapshot);
 const signOut = useRef(onSignedOut); signOut.current = onSignedOut;
 useEffect(() => { void history.load(); return () => history.dispose(); }, [history]);
 useEffect(() => { scopes.reconcile(found.value, ""); }, [scopes, found.value]);
 useEffect(() => { if (found.error === "session_changed" || found.error === "unauthenticated") signOut.current(); }, [found.error]);
 const writes = useStepStore<TermWriteIntent, TermWriteReceipt>("terms", session, () => termWriteClient(session)), [settled, setSettled] = useState(0);
 const results = writes ? <StepResults l={l} id="o03-term-results" store={writes} contract={contract} onSettled={() => setSettled(n => n + 1)} explain={code => termError(l, code)} revision={async intent => (await client.terms(intent.scope)).revision}
  describe={entry => ({ action: c(l, termActionOf[entry.intent.operation]), target: <>{c(l, entry.intent.operation.includes("bind") ? "targetBinding" : "targetTerm")}: {ltr(entry.subject)}</>, state: entry.receipt && <span className="o03-tag">{c(l, entry.receipt.state as TermWord)}</span> })} /> : <p role="alert">{stepError(l, "session_capacity")}</p>;
 if (scope) return <div className="o03-documents"><ScopeBar l={l} scope={scope} /><Terms key={scope.configuration + scope.digest} l={l} session={session} client={client} contract={contract} scope={scope} writes={writes} settled={settled} onSignedOut={onSignedOut} />{results}</div>;
 return <div className="o03-documents"><section className="o03-part" aria-labelledby="o03-part-title">
  <h3 id="o03-part-title">{c(l, "title")}</h3>
  {found.loading || !found.value && !found.error ? <p role="status">{text(l, "loading")}</p> : <>
   {found.error ? <div className="live-error" role="alert"><p>{c(l, "historyDenied")}</p><p className="live-mono">{ltr(found.error)}</p></div> : <p>{c(l, "noScope")}</p>}
   <p><a href={settingsHref()}>{c(l, "toSettings")}</a></p>
  </>}
 </section>{results}</div>;
}

type TermsProps = { l: Locale; session: Session; client: ReturnType<typeof setupReadClient>; contract: string; scope: SetupScope; writes: TermStore | undefined; settled: number; onSignedOut: () => void };
function Terms({ l, session, client, contract, scope, writes, settled, onSignedOut }: TermsProps) {
 const [chosen, setChosen] = useState<TermBinding>();
 const store = useMemo(() => new PageReadStore((after, snapshot) => client.terms({ contract, configuration: scope.configuration, digest: scope.digest }, after, snapshot)), [client, contract, scope.configuration, scope.digest]);
 const state = useSyncExternalStore(store.subscribe, store.getSnapshot), heading = useRef<HTMLHeadingElement>(null);
 const signOut = useRef(onSignedOut); signOut.current = onSignedOut;
 useEffect(() => { void store.load(); return () => store.dispose(); }, [store]);
 const first = useRef(true), sending = useSyncExternalStore(writes?.subscribe ?? noStore, () => !!writes?.snapshot().entries.some(e => e.phase === "pending")), blocked = useSyncExternalStore(writes?.subscribe ?? noStore, () => !!writes?.unresolved());
 useEffect(() => { if (first.current) { first.current = false; return; } heading.current?.focus(); void store.load(); }, [settled]);
 useEffect(() => { if (state.error === "session_changed" || state.error === "unauthenticated") signOut.current(); }, [state.error]);
 const page = state.value, offers = page ? termOffers(page, session.user).bindings : {};
 useEffect(() => { setChosen(undefined); }, [page]);
 return <section className="o03-part" aria-labelledby="o03-part-title" aria-busy={state.loading || undefined}>
  <div className="o03-docs-heading"><h3 id="o03-part-title" ref={heading} tabIndex={-1}>{c(l, "part")}</h3><button type="button" className="live-secondary" disabled={state.loading || sending} onClick={() => { heading.current?.focus(); void store.load(); }}>{text(l, "reload")}</button></div>
  <p>{c(l, "purpose")}</p>
  <p className="live-sub">{c(l, "notComputed")}</p>
  <TermsFeedback l={l} state={state} />
  {page && <>
   <p className="live-meta">{rulesText(l, "readAt")}: {stamp(l, page.at)} · {text(l, "eligibility")}</p>
   {page.term ? <TermFacts l={l} term={page.term} /> : <p className="live-empty">{c(l, "none")}</p>}
   <div className="o03-part-sub">
    <h4 id="o03-bindings-title">{c(l, "bindings")}</h4>
    <p className="live-sub">{c(l, "bindingsLead")}</p>
    <dl className="live-facts live-facts-grid o03-part-counts">
     <div><dt>{c(l, "zone")}</dt><dd>{ltr(page.zone)}<span className="live-sub">{c(l, "zoneNote")}</span></dd></div>
     <div><dt>{c(l, "total")}</dt><dd>{page.count}</dd></div>
     <div><dt>{c(l, "showing")}</dt><dd>{page.rows.length}</dd></div>
    </dl>
    {page.rows.length === 0 ? <p className="live-empty">{c(l, "empty")}</p> : <div className="live-table"><table className="o03-part-table o03-bindings-table" aria-labelledby="o03-bindings-title"><thead><tr>{(["period", "terms", "state", "people", "basis", "action"] as const).map(key => <th key={key} scope="col">{c(l, key)}</th>)}</tr></thead><tbody>{page.rows.map(row => <BindingRow key={row.binding} l={l} row={row} scope={scope} offer={offers[row.binding] ?? "none"} blocked={blocked} onApprove={() => { setChosen(row); document.getElementById("o03-term-steps")?.focus(); }} />)}</tbody></table></div>}
    <div className="o03-docs-pages">{state.after && <button type="button" className="live-secondary" disabled={sending} onClick={() => { heading.current?.focus(); void store.load(); }}>{text(l, "first")}</button>}{page.next && <button type="button" className="live-secondary" disabled={sending} onClick={() => { heading.current?.focus(); void store.load(true); }}>{text(l, "next")}</button>}</div>
   </div>
   {writes && <TermSteps l={l} session={session} store={writes} scope={scope} page={page} row={chosen} onRowDone={() => setChosen(undefined)} onSettled={() => { heading.current?.focus(); void store.load(); }} />}
  </>}
 </section>;
}

const noStore = () => () => undefined;

function TermsFeedback({ l, state }: { l: Locale; state: ReadState<unknown> }) {
 if (state.loading) return <p role="status">{text(l, "loading")}</p>;
 if (!state.error) return null;
 const message = state.error === "permission_denied" ? c(l, "denied") : state.error === "prerequisite_failed" ? rulesText(l, "prerequisite") : text(l, state.error === "evidence_stale" ? "stale" : "failed");
 return <div className="live-error" role="alert"><p>{message}</p><p className="live-mono">{ltr(state.error)}</p></div>;
}

const layer = (l: Locale, label: TermWord, value: ReactNode) => <span className="o03-part-line"><span className="live-sub">{c(l, label)}</span>{value}</span>;
const person = (l: Locale, label: TermWord, actor: string, at: TermWord, instant: string) => actor || instant ? <span className="o03-part-line">{actor && <>{c(l, label)}: {ltr(actor)}</>}{instant && <span className="live-sub">{c(l, at)}: {stamp(l, instant)}</span>}</span> : null;

function TermFacts({ l, term: t }: { l: Locale; term: PaymentTerm }) {
 return <><div className="o03-docs-scope">
  <div><h4>{c(l, "values")}</h4><dl className="live-facts">
   <Fact label={c(l, "netDays")}>{netDays(l, t.netDays)}</Fact>
   <Fact label={c(l, "revision")}>{ltr(t.revision)}</Fact>
  </dl></div>
  <div><h4>{c(l, "record")}</h4><dl className="live-facts">
   <Fact label={c(l, "recordedState")}><span className="o03-tag">{c(l, t.state as TermWord)}</span></Fact>
   <Fact label={c(l, "observed")}>{c(l, ("observed_" + t.observation) as TermWord)}</Fact>
   {t.pendingApprover && <Fact label={c(l, "pinned")}>{ltr(t.pendingApprover)}</Fact>}
   <Fact label={c(l, "author")}>{ltr(t.author)}</Fact>
   {t.writtenAt && <Fact label={c(l, "writtenAt")}>{stamp(l, t.writtenAt)}</Fact>}
   {t.approver && <Fact label={c(l, "approver")}>{ltr(t.approver)}</Fact>}
   {t.approvedAt && <Fact label={c(l, "approvedAt")}>{stamp(l, t.approvedAt)}</Fact>}
  </dl></div>
 </div>
 <details><summary>{text(l, "details")}</summary><p className="live-mono">{ltr(t.id)}</p><p className="live-mono">{ltr(t.digest)}</p></details></>;
}

function BindingRow({ l, row: r, scope, offer, blocked, onApprove }: { l: Locale; row: TermBinding; scope: SetupScope; offer: TermOffer; blocked: boolean; onApprove: () => void }) {
 const origin = r.origin === "profile" ? c(l, "originProfile") : r.configuration === scope.configuration ? c(l, "originThese") : c(l, "originOther");
 return <tr>
  <td data-label={c(l, "period")}><span className="o03-part-line">{c(l, "from")}: {day(l, r.from)}</span><span className="o03-part-line">{r.to ? <>{c(l, "to")}: {day(l, r.to)}</> : c(l, "noEnd")}</span><details><summary>{text(l, "details")}</summary><p className="live-mono">{ltr(r.binding)}</p><p className="live-mono">{ltr(r.digest)}</p></details></td>
  <td data-label={c(l, "terms")}>{netDays(l, r.net_days)}<span className="live-sub">{c(l, "days")}</span><span className="live-sub">{c(l, "revision")}: {ltr(r.term_revision)}</span><span className="live-sub">{c(l, "timeZone")}: {ltr(r.zone)}</span></td>
  <td data-label={c(l, "state")}>
   {layer(l, "recordedState", <span className="o03-tag">{c(l, r.state as TermWord)}</span>)}
   {layer(l, "observed", c(l, ("observed_" + r.observation) as TermWord))}
   {r.pending_approver && layer(l, "pinned", ltr(r.pending_approver))}
  </td>
  <td data-label={c(l, "people")}>{person(l, "appliedBy", r.author, "appliedAt", r.written_at)}{person(l, "approvedRow", r.approver, "approvedAt", r.approved_at)}</td>
  <td data-label={c(l, "basis")}><span className="o03-part-line">{c(l, "reason")}: {c(l, r.reason as TermWord)}</span><span className="o03-part-line">{c(l, "origin")}: {origin}{r.origin === "configuration" && r.configuration !== scope.configuration && <span className="live-sub">{ltr(r.configuration)}</span>}</span></td>
  <td data-label={c(l, "action")}>{offer === "offer" ? <button type="button" className="live-secondary" disabled={blocked} aria-describedby={blocked ? "o03-term-blocked" : undefined} aria-label={c(l, "actApproveApply") + ": " + r.binding} onClick={onApprove}>{c(l, "actApproveShort")}</button> : offer === "own" ? c(l, "ownBinding") : null}</td>
 </tr>;
}
function Fact({ label, children }: { label: string; children: ReactNode }) { return <div><dt>{label}</dt><dd>{children}</dd></div>; }
