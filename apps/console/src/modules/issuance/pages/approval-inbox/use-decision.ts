import { useEffect, useRef, useState, useSyncExternalStore } from "react";
import { useService } from "../../../../service/context";
import type { ApprovalRecord, Data } from "./contract";
import { DecisionError, serviceDecisionRecord } from "./decision-api";
import { sessionDecisionStore, type DecisionState, type DecisionStore } from "./decision-store";
const empty: DecisionState = Object.freeze({ selection: "", checking: false, rationale: "", reason: "", entries: Object.freeze({}) });
const idleSubscribe = () => () => {};
const idleSnapshot = () => empty;
export function useDecision(data: Data | undefined, record: ApprovalRecord | undefined, scope: string, refreshing: boolean, refresh: () => void) {
  const service = useService();
  const session = service?.session;
  const sessionKey = session ? JSON.stringify([session.user, session.workspace, session.csrf]) : "";
  const [binding, setBinding] = useState<{ key: string; store?: DecisionStore; error?: string }>();
  const refreshed = useRef(new Set<string>());
  const refreshRef = useRef(refresh); refreshRef.current = refresh;
  useEffect(() => {
    if (!session) { setBinding(undefined); return; }
    let store: DecisionStore;
    try {
      store = sessionDecisionStore(session);
      refreshed.current = new Set(Object.values(store.snapshot().entries).filter(e => e.phase === "recorded").map(e => e.submission.attempt.key));
      store.invalidatePreparation();
      setBinding({ key: sessionKey, store });
    } catch (error) { setBinding({ key: sessionKey, error: error instanceof DecisionError ? error.code : "session_unavailable" }); }
    return () => { store?.invalidatePreparation(); };
  }, [sessionKey]);
  const active = binding?.key === sessionKey ? binding : undefined;
  const store = active?.store;
  const state = useSyncExternalStore(store?.subscribe ?? idleSubscribe, store?.snapshot ?? idleSnapshot, idleSnapshot);
  const capability = (data?.commands === "financial_approval_v2" || (data?.commands === "financial_charge_v1" && record?.kind === "billing_result")) && data.actor.id === session?.user;
  useEffect(() => { store?.select(scope, capability && record && serviceDecisionRecord(record) ? record : undefined); }, [store, scope, capability, record?.kind, record?.amounts[0]?.basis, record?.id, record?.evidence.revision]);
  useEffect(() => { if (refreshing) store?.invalidatePreparation(); }, [store, refreshing]);
  useEffect(() => {
    const newSuccess = Object.values(state.entries).filter(e => e.phase === "recorded" && !refreshed.current.has(e.submission.attempt.key));
    if (!newSuccess.length) return;
    for (const entry of newSuccess) refreshed.current.add(entry.submission.attempt.key);
    refreshRef.current();
  }, [state.entries]);
  const reauthenticate = service?.stepUp ? async () => { store?.invalidatePreparation(); await service.stepUp!(); } : undefined;
  return { store, state, capability, error: active?.error ?? (!session ? "session_unavailable" : undefined), reauthenticate };
}
export type DecisionBinding = ReturnType<typeof useDecision>;
