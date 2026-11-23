import { useEffect, useMemo, useRef, useSyncExternalStore } from "react";
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
import { sampleLabel, statementText as c, type StatementWord } from "./statements-copy";

type Props = { l: Locale; session: Session; contract: string; scope: SetupScope; onSignedOut: () => void };

export function Statements({ l, session, contract, scope, onSignedOut }: Props) {
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
 const page = state.value;
 return <section className="o03-part" aria-labelledby="o03-part-title" aria-busy={state.loading || undefined}>
  <div className="o03-docs-heading"><h3 id="o03-part-title" ref={heading} tabIndex={-1}>{partText(l, "statements")}</h3><button type="button" className="live-secondary" disabled={state.loading || samples.loading} onClick={() => load()}>{text(l, "reload")}</button></div>
  <p>{c(l, "purpose")}</p>
  <p className="live-sub">{c(l, "assurance")}</p>
  <PartFeedback l={l} state={state} />
  {page && <>
   <p className="live-meta">{rulesText(l, "readAt")}: {stamp(l, page.at)} · {text(l, "eligibility")}</p>
   <dl className="live-facts live-facts-grid o03-part-counts">
    <div><dt>{c(l, "total")}</dt><dd>{page.count}</dd></div>
    <div><dt>{c(l, "showing")}</dt><dd>{page.rows.length}</dd></div>
   </dl>
   {page.rows.length === 0 ? <p className="live-empty">{c(l, "empty")}</p> : <div className="live-table"><table className="o03-part-table o03-statements-table"><thead><tr>{(["step", "state", "names", "author", "recorded"] as const).map(key => <th key={key} scope="col">{c(l, key)}</th>)}</tr></thead><tbody>{page.rows.map(row => <StatementRow key={row.attestation} l={l} row={row} />)}</tbody></table></div>}
   <div className="o03-docs-pages">{state.after && <button type="button" className="live-secondary" onClick={() => load()}>{text(l, "first")}</button>}{page.next && <button type="button" className="live-secondary" onClick={() => load(true)}>{text(l, "next")}</button>}</div>
  </>}
  <div className="o03-part-sub" aria-busy={samples.loading || undefined}>
   <h4>{c(l, "samples")}</h4>
   <p className="live-sub">{c(l, "samplesNote")}</p>
   <PartFeedback l={l} state={samples} />
   {samples.value && <dl className="live-facts live-facts-grid o03-samples">{samples.value.entries.map(entry => <div key={entry.code}><dt>{sampleLabel(l, entry)}</dt><dd>{c(l, ("use_" + entry.use) as StatementWord)}</dd></div>)}</dl>}
  </div>
 </section>;
}

function StatementRow({ l, row: r }: { l: Locale; row: Attestation }) {
 const kind = r.purpose.startsWith("channel_") ? "channel" : "change";
 return <tr>
  <td data-label={c(l, "step")}>{c(l, r.purpose as StatementWord)}<details><summary>{c(l, "text")}</summary><p className="o03-statement" dir="auto">{r.statement}</p><p className="live-sub">{c(l, "locator")}: {r.locator ? <bdi>{r.locator}</bdi> : c(l, "noLocator")}</p><p className="live-mono">{ltr(r.attestation)}</p></details></td>
  <td data-label={c(l, "state")}>{r.state === "consumed" ? <span className="o03-tag">{c(l, "consumed")}</span> : c(l, r.state as StatementWord)}</td>
  <td data-label={c(l, "names")}>{c(l, kind)}<span className="live-sub">{ltr(r.target)}</span></td>
  <td data-label={c(l, "author")}>{ltr(r.actor)}</td>
  <td data-label={c(l, "recorded")}>{stamp(l, r.recorded_at)}</td>
 </tr>;
}
