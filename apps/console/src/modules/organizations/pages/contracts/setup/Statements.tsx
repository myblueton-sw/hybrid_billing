import { useEffect, useMemo, useRef, useState, useSyncExternalStore } from "react";
import { attempt } from "../../../../../service/admin";
import type { Locale } from "../../../../../shared/types";
import type { Session } from "../../../../../service/api";
import { ltr, stamp } from "../live-format";
import type { Attestation } from "./attestation";
import { setupReadClient } from "./client";
import { text } from "./copy";
import { PageReadStore } from "./page-store";
import { partText } from "./part-copy";
import { PartReadStore } from "./part-store";
import { PartFeedback } from "./Rules";
import { rulesText } from "./rules-copy";
import type { SetupScope } from "./scope";
import { sampleLabel, statementError, statementText as c, type StatementWord } from "./statements-copy";
import { withdrawIntent } from "./channel-flow";
import type { StatementStore } from "./ChannelWrites";
import type { ProtectedStore } from "./RuleWrites";
import { codeOf } from "./StepResults";
import { stepText as t } from "./step-copy";

type Props = { l: Locale; session: Session; contract: string; scope: SetupScope; writes: ProtectedStore | undefined; statements: StatementStore | undefined; settled: number; onSignedOut: () => void };
const idle = () => () => undefined;

export function Statements({ l, session, contract, scope, writes, statements, settled, onSignedOut }: Props) {
 const [chosen, setChosen] = useState<Attestation>(), [error, setError] = useState("");
 const target = useMemo(() => ({ contract, configuration: scope.configuration, digest: scope.digest }), [contract, scope.configuration, scope.digest]);
 const client = useMemo(() => setupReadClient(session), [session.user, session.workspace, session.csrf]);
 const store = useMemo(() => new PageReadStore((after, snapshot) => client.attestations(target, after, snapshot)), [client, target]);
 const catalog = useMemo(() => new PartReadStore(() => client.references(target)), [client, target]);
 const state = useSyncExternalStore(store.subscribe, store.getSnapshot), samples = useSyncExternalStore(catalog.subscribe, catalog.getSnapshot);
 const heading = useRef<HTMLHeadingElement>(null), signOut = useRef(onSignedOut); signOut.current = onSignedOut;
 useEffect(() => { void store.load(); return () => store.dispose(); }, [store]);
 useEffect(() => { void catalog.load(); return () => catalog.dispose(); }, [catalog]);
 useEffect(() => { if ([state.error, samples.error].some(v => v === "session_changed" || v === "unauthenticated")) signOut.current(); }, [state.error, samples.error]);
 const load = (next = false) => { heading.current?.focus(); void store.load(next); if (!next) void catalog.load(); };
 const first = useRef(true), steps = useRef<HTMLHeadingElement>(null), unresolved = () => !!writes?.unresolved() || !!statements?.unresolved();
 const blockedA = useSyncExternalStore(writes?.subscribe ?? idle, unresolved), blockedB = useSyncExternalStore(statements?.subscribe ?? idle, unresolved), blocked = blockedA || blockedB;
 const sending = useSyncExternalStore(statements?.subscribe ?? idle, () => !!statements?.snapshot().entries.some(e => e.phase === "pending"));
 useEffect(() => { if (first.current) { first.current = false; return; } load(); }, [settled]);
 useEffect(() => { setChosen(undefined); }, [state.value]);
 useEffect(() => { if (chosen) steps.current?.focus(); }, [chosen]);
 const withdraw = async () => {
  if (!chosen || !statements || !state.value) return;
  const row = chosen; setChosen(undefined); setError(""); heading.current?.focus();
  try { await statements.submit(withdrawIntent(session, state.value, row, attempt()), row.attestation); } catch (e) { const code = codeOf(e); setError(code === "invalid_setup_request" ? "evidence_stale" : code); return; }
  load();
 };
 const page = state.value;
 return <section className="o03-part" aria-labelledby="o03-part-title" aria-busy={state.loading || undefined}>
  <div className="o03-docs-heading"><h3 id="o03-part-title" ref={heading} tabIndex={-1}>{partText(l, "statements")}</h3><button type="button" className="live-secondary" disabled={state.loading || samples.loading || sending} onClick={() => load()}>{text(l, "reload")}</button></div>
  <p>{c(l, "purpose")}</p>
  <p className="live-sub">{c(l, "assurance")}</p>
  <PartFeedback l={l} state={state} />
  {error && <div className="live-error" role="alert"><p>{statementError(l, error)}</p><p className="live-mono">{t(l, "code")}: {ltr(error)}</p></div>}
  {page && <>
   <p className="live-meta">{rulesText(l, "readAt")}: {stamp(l, page.at)} · {text(l, "eligibility")}</p>
   <dl className="live-facts live-facts-grid o03-part-counts">
    <div><dt>{c(l, "total")}</dt><dd>{page.count}</dd></div>
    <div><dt>{c(l, "showing")}</dt><dd>{page.rows.length}</dd></div>
   </dl>
   {page.rows.length === 0 ? <p className="live-empty">{c(l, "empty")}</p> : <div className="live-table"><table className="o03-part-table o03-statements-table"><thead><tr>{(["step", "state", "names", "author", "recorded", "action"] as const).map(key => <th key={key} scope="col">{c(l, key)}</th>)}</tr></thead><tbody>{page.rows.map(row => <StatementRow key={row.attestation} l={l} row={row} own={!!statements && row.actor === session.user && row.state === "unconsumed"} blocked={blocked} onWithdraw={() => { setError(""); setChosen(row); }} />)}</tbody></table></div>}
   <div className="o03-docs-pages">{state.after && <button type="button" className="live-secondary" disabled={sending} onClick={() => load()}>{text(l, "first")}</button>}{page.next && <button type="button" className="live-secondary" disabled={sending} onClick={() => load(true)}>{text(l, "next")}</button>}</div>
   {statements && <div className="o03-part-sub o03-steps">
    <h4 id="o03-statement-steps" ref={steps} tabIndex={-1}>{t(l, "actions")}</h4>
    <p className="live-sub">{c(l, "ownOnly")} {t(l, "permissionNote")}</p>
    {blocked && <p id="o03-statement-blocked"><strong>{t(l, "blocked")}</strong></p>}
    {chosen && <div className="o03-setup-review">
     <h5>{t(l, "reviewTitle")}</h5>
     <dl className="live-facts live-facts-grid"><div><dt>{t(l, "step")}</dt><dd>{c(l, "actWithdraw")}</dd></div><div><dt>{c(l, "step")}</dt><dd>{c(l, chosen.purpose as StatementWord)}</dd></div><div><dt>{c(l, "names")}</dt><dd>{ltr(chosen.target)}</dd></div><div><dt>{c(l, "recorded")}</dt><dd>{stamp(l, chosen.recorded_at)}</dd></div></dl>
     <p className="o03-statement" dir="auto">{chosen.statement}</p>
     <p><strong>{c(l, "noticeWithdraw")}</strong></p>
     <p className="live-sub">{t(l, "reviewNote")}</p>
     <div className="o03-docs-pages"><button type="button" className="live-primary" disabled={blocked} aria-describedby={blocked ? "o03-statement-blocked" : undefined} onClick={() => void withdraw()}>{c(l, "actWithdraw")}</button><button type="button" className="live-secondary" onClick={() => { setChosen(undefined); heading.current?.focus(); }}>{t(l, "cancel")}</button></div>
    </div>}
   </div>}
  </>}
  <div className="o03-part-sub" aria-busy={samples.loading || undefined}>
   <h4>{c(l, "samples")}</h4>
   <p className="live-sub">{c(l, "samplesNote")}</p>
   <PartFeedback l={l} state={samples} />
   {samples.value && <dl className="live-facts live-facts-grid o03-samples">{samples.value.entries.map(entry => <div key={entry.code}><dt>{sampleLabel(l, entry)}</dt><dd>{c(l, ("use_" + entry.use) as StatementWord)}</dd></div>)}</dl>}
  </div>
 </section>;
}

function StatementRow({ l, row: r, own, blocked, onWithdraw }: { l: Locale; row: Attestation; own: boolean; blocked: boolean; onWithdraw: () => void }) {
 const kind = r.purpose.startsWith("channel_") ? "channel" : "change";
 return <tr>
  <td data-label={c(l, "step")}>{c(l, r.purpose as StatementWord)}<details><summary>{c(l, "text")}</summary><p className="o03-statement" dir="auto">{r.statement}</p><p className="live-sub">{c(l, "locator")}: {r.locator ? <bdi>{r.locator}</bdi> : c(l, "noLocator")}</p><p className="live-mono">{ltr(r.attestation)}</p></details></td>
  <td data-label={c(l, "state")}>{r.state === "consumed" ? <span className="o03-tag">{c(l, "consumed")}</span> : c(l, r.state as StatementWord)}</td>
  <td data-label={c(l, "names")}>{c(l, kind)}<span className="live-sub">{ltr(r.target)}</span></td>
  <td data-label={c(l, "author")}>{ltr(r.actor)}</td>
  <td data-label={c(l, "recorded")}>{stamp(l, r.recorded_at)}</td>
  <td data-label={c(l, "action")}>{own ? <button type="button" className="live-secondary" disabled={blocked} aria-describedby={blocked ? "o03-statement-blocked" : undefined} aria-label={c(l, "actWithdraw") + ": " + r.attestation} onClick={onWithdraw}>{c(l, "actWithdraw")}</button> : null}</td>
 </tr>;
}
