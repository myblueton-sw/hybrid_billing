import { useEffect, useMemo, useRef, useState, useSyncExternalStore, type ReactNode } from "react";
import type { Locale } from "../../../../../shared/types";
import type { Session } from "../../../../../service/api";
import type { OrgParty } from "../../../../../service/admin";
import { day, ltr, nameOf, stamp } from "../live-format";
import { setupReadClient } from "./client";
import { SetupReadStore, type ReadState } from "./read-store";
import type { Configuration } from "./history";
import { text, type SetupWord } from "./copy";
import { Writes } from "./Writes";
import { PartFrame, PartNav } from "./Parts";
import { Rules } from "./Rules";
import { partText } from "./part-copy";
import { scopeEligible, sessionScopeStore, type SetupPart, type SetupScope } from "./scope";
import "./documents.css";

type Props = { l: Locale; session: Session; contract: string; on: string; parties: OrgParty[]; part: SetupPart; onPart: (part: SetupPart) => void; onSignedOut: () => void };
export function Documents({ l, session, contract, on, parties, part, onPart, onSignedOut }: Props) {
 const store = useMemo(() => new SetupReadStore(setupReadClient(session), contract, on), [session.user, session.workspace, session.csrf, contract, on]);
 const state = useSyncExternalStore(store.subscribe, store.getSnapshot);
 const scopes = useMemo(() => sessionScopeStore(session, contract, on), [session.user, session.workspace, session.csrf, contract, on]);
 const scope = useSyncExternalStore(scopes.subscribe, scopes.getSnapshot);
 const scopeHeading = useRef<HTMLHeadingElement>(null), historyHeading = useRef<HTMLHeadingElement>(null);
 const reloadSource = () => { scopeHeading.current?.focus(); void store.source(); };
 const reloadHistory = (next = false) => { historyHeading.current?.focus(); void store.history(next); };
 const signOut = useRef(onSignedOut); signOut.current = onSignedOut;
 useEffect(() => { void store.load(); return () => store.dispose(); }, [store]);
 useEffect(() => { if ([state.context.error, state.options.error, state.history.error].some(v => v === "session_changed" || v === "unauthenticated")) signOut.current(); }, [state]);
 const context = state.context.value, page = state.history.value;
 useEffect(() => { scopes.reconcile(page, state.historyAfter); }, [scopes, page, state.historyAfter]);
 const person = (id: string) => <bdi>{nameOf(id, parties)}</bdi>;
 const toSettings = useRef(false);
 useEffect(() => { if (part === "settings" && toSettings.current) { toSettings.current = false; historyHeading.current?.focus(); } }, [part]);
 if (part !== "settings") return <div className="o03-documents">
  <PartNav l={l} part={part} onPart={onPart} />
  <PartFrame l={l} part={part} onPart={next => { toSettings.current = next === "settings"; onPart(next); }} scope={scope} loading={state.history.loading || !page && !state.history.error} feedback={!scope && state.history.error ? <ReadFeedback l={l} state={state.history} /> : null}>{chosen => part === "rules" ? <Rules key={chosen.configuration + chosen.digest} l={l} session={session} contract={contract} scope={chosen} onSignedOut={onSignedOut} /> : null}</PartFrame>
 </div>;
 return <div className="o03-documents">
  <PartNav l={l} part={part} onPart={onPart} />
  <p className="live-note-rule">{text(l, "sampleNote")}</p>
  <section aria-labelledby="o03-docs-scope" aria-busy={state.context.loading || undefined}>
   <div className="o03-docs-heading"><h3 id="o03-docs-scope" ref={scopeHeading} tabIndex={-1}>{text(l, "scope")}</h3><button type="button" className="live-secondary" disabled={state.context.loading || state.options.loading} onClick={reloadSource}>{text(l, "reload")}</button></div>
   <p>{text(l, "selectedDate")}: {on ? day(l, on) : text(l, "unknown")}</p>
   <ReadFeedback l={l} state={state.context} />
   {context && <div className="o03-docs-scope">
    <div><h4>{text(l, "contractBasis")}</h4><dl className="live-facts">
     <Fact label={text(l, "contractRevision")}>{ltr(context.contractRevision)}</Fact>
     <Fact label={text(l, "lifecycle")}>{ltr(context.lifecycleSequence)}</Fact>
    </dl></div>
    <div><h4>{text(l, "profile")}</h4><p><bdi>{context.billingProfileName || context.billingProfile}</bdi></p><p>{text(l, "profileVersion")}: {ltr(context.billingRevision)}</p><dl className="live-facts"><Fact label={text(l, "buyer")}>{person(context.buyer)}</Fact><Fact label={text(l, "payer")}>{person(context.payer)}</Fact><Fact label={text(l, "issuer")}>{person(context.issuer)}</Fact><Fact label={text(l, "currency")}>{ltr(context.currency)}</Fact></dl><p className="live-sub">{text(l, "profileNote")}</p><p>{text(l, "eligibility")}</p><details><summary>{text(l, "details")}</summary><p className="live-mono">{ltr(context.billingProfile)}</p><p className="live-mono">{ltr(context.scopeDigest)}</p></details></div>
   </div>}
  </section>
  <section aria-labelledby="o03-docs-history" aria-busy={state.history.loading || undefined}>
   <div className="o03-docs-heading"><h3 id="o03-docs-history" ref={historyHeading} tabIndex={-1}>{text(l, "history")}</h3><button type="button" className="live-secondary" disabled={state.history.loading} onClick={() => reloadHistory()}>{text(l, "reload")}</button></div>
   <ReadFeedback l={l} state={state.history} />
   {page && <><p className="live-meta">{text(l, "total")}: {page.count} · {text(l, "showing")}: {page.rows.length}</p>
    {page.rows.length === 0 ? <p className="live-empty">{text(l, "empty")}</p> : <div className="live-table"><table className="o03-docs-table"><thead><tr>{(["state", "settings", "scope", "people", "recorded"] as const).map(key => <th key={key} scope="col">{text(l, key)}</th>)}<th scope="col">{partText(l, "action")}</th></tr></thead><tbody>{page.rows.map(row => <HistoryRow key={row.configuration} l={l} row={row} scope={scope} onUse={() => scopes.choose(row)} />)}</tbody></table></div>}
    <div className="o03-docs-pages">{state.historyAfter && <button type="button" className="live-secondary" onClick={() => reloadHistory()}>{text(l, "first")}</button>}{page.next && <button type="button" className="live-secondary" onClick={() => reloadHistory(true)}>{text(l, "next")}</button>}</div>
   </>}
  </section>
  <section aria-labelledby="o03-docs-options" aria-busy={state.options.loading || undefined}><h3 id="o03-docs-options">{text(l, "sample")}</h3><ReadFeedback l={l} state={state.options} />{state.options.value && <><dl className="live-facts live-facts-grid">
   <Fact label={text(l, "mode")}>{ltr(state.options.value.mode)}</Fact><Fact label={text(l, "documentProfile")}>{ltr(state.options.value.profile)}</Fact>
   <Fact label={text(l, "region")}>{ltr(state.options.value.region)}</Fact><Fact label={text(l, "tax")}>{ltr(state.options.value.tax)}</Fact><Fact label={text(l, "zone")}>{ltr(state.options.value.zone)}</Fact><Fact label={text(l, "languages")}>{ltr(state.options.value.languages.join(", "))}</Fact>
  </dl><p className="live-sub">{text(l, "presetNote")}</p></>}</section>
  <Writes l={l} session={session} reads={store} />
 </div>;
}
function Fact({ label, children }: { label: string; children: ReactNode }) { return <div><dt>{label}</dt><dd>{children}</dd></div>; }
function ReadFeedback({ l, state }: { l: Locale; state: ReadState<unknown> }) {
 if (state.loading) return <p role="status">{text(l, "loading")}</p>;
 if (!state.error) return null;
 const key = state.error === "permission_denied" ? "denied" : state.error === "evidence_stale" ? "stale" : "failed";
 return <div className="live-error" role="alert"><p>{text(l, key)}</p><p className="live-mono">{ltr(state.error)}</p></div>;
}
function HistoryRow({ l, row: r, scope, onUse }: { l: Locale; row: Configuration; scope: SetupScope | undefined; onUse: () => void }) {
 const selected = scope?.configuration === r.configuration && scope.digest === r.digest, mark = useRef<HTMLSpanElement>(null), [used, setUsed] = useState(false);
 useEffect(() => { if (selected && used) { setUsed(false); mark.current?.focus(); } }, [selected, used]);
 return <tr>
  <td data-label={text(l, "state")}><span className="o03-tag">{text(l, r.state as SetupWord)}</span><span className="live-sub live-mono">{ltr(r.configuration)}</span>{!r.setup_on && <span className="live-sub">{text(l, "legacy")}</span>}</td>
  <td data-label={text(l, "settings")}>{ltr(r.region)} · {ltr(r.tax)}<span className="live-sub">{ltr(r.zone)} · {ltr(r.languages)}</span><span className="live-sub">{ltr(r.mode)} · {ltr(r.profile)}</span></td>
  <td data-label={text(l, "scope")}>{r.setup_on ? day(l, r.setup_on) : text(l, "unknown")}<span className="live-sub">{text(l, "recordedContractRevision")}: {ltr(r.contract_revision)}</span><span className="live-sub">{text(l, "recordedProfileVersion")}: {ltr(r.billing_revision)}</span><details><summary>{text(l, "details")}</summary><p className="live-mono">{ltr(r.billing_profile)}</p><p className="live-mono">{ltr(r.digest)}</p></details></td>
  <td data-label={text(l, "people")}>{text(l, "proposer")}: {ltr(r.proposer)}{r.confirmer && <span className="live-sub">{text(l, r.state === "confirmed" ? "confirmer" : "confirmerAttempt")}: {ltr(r.confirmer)}</span>}</td>
  <td data-label={text(l, "recorded")}>{r.proposed_at ? <span>{text(l, "proposedAt")}: {stamp(l, r.proposed_at)}</span> : text(l, "unknown")}{r.activated_at ? <span className="live-sub">{text(l, "confirmedAt")}: {stamp(l, r.activated_at)}</span> : r.approval_observed_at && <span className="live-sub">{text(l, "approvalObserved")}: {stamp(l, r.approval_observed_at)}</span>}</td>
  <td data-label={partText(l, "action")}>{!scopeEligible(r) ? null : selected ? <span className="o03-docs-selected" ref={mark} tabIndex={-1}>{partText(l, "selected")}</span> : <button type="button" className="live-secondary o03-docs-use" aria-label={partText(l, "use") + ": " + r.configuration} onClick={() => { setUsed(true); onUse(); }}>{partText(l, "use")}</button>}</td>
 </tr>;
}
