import { useEffect, useMemo, useRef, useSyncExternalStore, type ReactNode } from "react";
import type { Locale } from "../../../../../shared/types";
import type { Session } from "../../../../../service/api";
import { ltr, stamp } from "../live-format";
import { setupReadClient } from "./client";
import { text, type SetupWord } from "./copy";
import { PartReadStore } from "./part-store";
import { partText } from "./part-copy";
import type { ProtectedRules } from "./protected";
import type { ReadState } from "./read-store";
import { duration, rulesText as r } from "./rules-copy";
import type { SetupScope } from "./scope";
import { RuleSteps, type ProtectedStore } from "./RuleWrites";

type Props = { l: Locale; session: Session; contract: string; scope: SetupScope; writes: ProtectedStore | undefined; settled: number; onSignedOut: () => void };
const retained: Record<string, SetupWord> = { registration_unknown: "proposal_unknown", proposed: "proposed", confirmation_unknown: "confirmation_unknown", confirmed: "confirmed" };

export function Rules({ l, session, contract, scope, writes, settled, onSignedOut }: Props) {
 const store = useMemo(() => { const client = setupReadClient(session); return new PartReadStore(() => client.protectedSummary({ contract, configuration: scope.configuration, digest: scope.digest })); }, [session.user, session.workspace, session.csrf, contract, scope.configuration, scope.digest]);
 const state = useSyncExternalStore(store.subscribe, store.getSnapshot), heading = useRef<HTMLHeadingElement>(null);
 const signOut = useRef(onSignedOut); signOut.current = onSignedOut;
 useEffect(() => { void store.load(); return () => store.dispose(); }, [store]);
 const first = useRef(true), sending = useSyncExternalStore(writes?.subscribe ?? idle, () => !!writes?.snapshot().entries.some(e => e.phase === "pending"));
 const reload = () => { heading.current?.focus(); void store.load(); };
 useEffect(() => { if (first.current) { first.current = false; return; } reload(); }, [settled]);
 useEffect(() => { if (state.error === "session_changed" || state.error === "unauthenticated") signOut.current(); }, [state.error]);
 const summary = state.value, rules = summary?.rules;
 return <section className="o03-part" aria-labelledby="o03-part-title" aria-busy={state.loading || undefined}>
  <div className="o03-docs-heading"><h3 id="o03-part-title" ref={heading} tabIndex={-1}>{partText(l, "rules")}</h3><button type="button" className="live-secondary" disabled={state.loading || sending} onClick={reload}>{text(l, "reload")}</button></div>
  <p>{r(l, "purpose")}</p>
  <p className="live-sub">{r(l, "installation")}</p>
  <PartFeedback l={l} state={state} />
  {summary && rules && <>
   <p className="live-meta">{r(l, "readAt")}: {stamp(l, summary.at)} · {text(l, "eligibility")}</p>
   {rules.state === "none" ? <p className="live-empty">{r(l, rules.observation === "unowned" ? "unowned" : "none")}</p> : <Owned l={l} rules={rules} />}
   {writes && <RuleSteps l={l} session={session} store={writes} scope={scope} summary={summary} onSettled={reload} />}
  </>}
 </section>;
}

const idle = () => () => undefined;

export function PartFeedback({ l, state }: { l: Locale; state: ReadState<unknown> }) {
 if (state.loading) return <p role="status">{text(l, "loading")}</p>;
 if (!state.error) return null;
 const message = state.error === "permission_denied" ? r(l, "denied") : state.error === "prerequisite_failed" ? r(l, "prerequisite") : text(l, state.error === "evidence_stale" ? "stale" : "failed");
 return <div className="live-error" role="alert"><p>{message}</p><p className="live-mono">{ltr(state.error)}</p></div>;
}
function Owned({ l, rules }: { l: Locale; rules: ProtectedRules }) {
 const settled = rules.state === "confirmed";
 const values = <div key="values"><h4>{r(l, settled || rules.observation === "active" ? "values" : "proposedValues")}</h4><dl className="live-facts">
  <Fact label={r(l, "approvals")}>{ltr(rules.approvals)}</Fact>
  <Fact label={r(l, "validity")}>{duration(l, rules.validitySeconds)}</Fact>
  <Fact label={r(l, "lead")}>{duration(l, rules.maxLeadSeconds)}</Fact>
  <Fact label={r(l, "channelAge")}>{duration(l, rules.channelAgeSeconds)}</Fact>
 </dl></div>;
 const record = <div key="record"><h4>{r(l, "record")}</h4><dl className="live-facts">
  <Fact label={r(l, "recordedState")}><span className="o03-tag">{text(l, retained[rules.state])}</span></Fact>
  <Fact label={r(l, "observed")}>{r(l, rules.observation as "absent" | "pending" | "active")}</Fact>
  {rules.pendingConfirmer && <Fact label={r(l, "pinned")}>{ltr(rules.pendingConfirmer)}</Fact>}
  <Fact label={r(l, "proposedBy")}>{ltr(rules.proposer)}</Fact>
  {rules.proposedAt && <Fact label={r(l, "proposedAt")}>{stamp(l, rules.proposedAt)}</Fact>}
  {rules.confirmer && <Fact label={r(l, "confirmedBy")}>{ltr(rules.confirmer)}</Fact>}
  {rules.confirmedAt && <Fact label={r(l, "confirmedAt")}>{stamp(l, rules.confirmedAt)}</Fact>}
 </dl></div>;
 const exact = ([["validity", rules.validitySeconds], ["lead", rules.maxLeadSeconds], ["channelAge", rules.channelAgeSeconds]] as const);
 return <><div className="o03-docs-scope">{settled ? [values, record] : [record, values]}</div>
  <details><summary>{text(l, "details")}</summary><p className="live-mono">{ltr(rules.id)}</p><p className="live-mono">{ltr(rules.digest)}</p>
   <p>{r(l, "exact")}</p><dl className="live-facts">{exact.map(([key, value]) => <Fact key={key} label={r(l, key)}>{ltr(value)}</Fact>)}</dl></details></>;
}
function Fact({ label, children }: { label: string; children: ReactNode }) { return <div><dt>{label}</dt><dd>{children}</dd></div>; }
