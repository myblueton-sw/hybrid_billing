import { useEffect, useState, useSyncExternalStore } from "react";
import { useService } from "../../../../../service/context";
import type { ApprovalRecord, Data } from "../contract";
import { decisionFamily } from "../decision-api";
import { DocumentError } from "./contract";
import { preparationSettled, sessionDocumentStore, type DocumentState, type DocumentStore } from "./store";
const empty: DocumentState = Object.freeze<DocumentState>({ selection: "", context: { phase: "idle" }, documents: { phase: "idle" }, preparations: { phase: "idle" }, document: { phase: "idle" }, preparation: { phase: "idle" }, invalidFields: [], entries: [] });
const subscribeIdle = () => () => {}, snapshotIdle = () => empty;
export function useDocuments(data: Data | undefined, record: ApprovalRecord | undefined, scope: string, refreshing: boolean) {
 const service = useService(), session = service?.session;
 const sessionKey = session ? JSON.stringify([session.user, session.workspace, session.csrf]) : "";
 const [binding, setBinding] = useState<{ key: string; store?: DocumentStore; error?: string }>();
 const [rationale, setRationale] = useState("");
 const capability = (data?.commands === "financial_charge_v1" || data?.commands === "financial_approval_v2") && data.actor.id === session?.user;
 const identity = capability && session && record && decisionFamily(record) === "financial_charge" ? { request: record.id, digest: record.evidence.revision!, workspace: session.workspace } : undefined;
 const identityKey = JSON.stringify(identity);
 useEffect(() => {
  if (!session) { setBinding(undefined); return; }
  let store: DocumentStore | undefined;
  try { store = sessionDocumentStore(session); store.invalidate(); setBinding({ key: sessionKey, store }); }
  catch (e) { setBinding({ key: sessionKey, error: e instanceof DocumentError ? e.code : "session_unavailable" }); }
  return () => store?.invalidate();
 }, [sessionKey]);
 const active = binding?.key === sessionKey ? binding : undefined, store = active?.store;
 const state = useSyncExternalStore(store?.subscribe ?? subscribeIdle, store?.snapshot ?? snapshotIdle, snapshotIdle);
 useEffect(() => { store?.select(scope, identity); setRationale(""); }, [store, scope, identityKey]);
 useEffect(() => { if (refreshing) { store?.invalidate(); setRationale(""); } }, [store, refreshing]);
 const enabled = !!identity && !refreshing && state.selection === JSON.stringify([scope, identity]);
 const reauthenticate = service?.stepUp ? async () => { store?.invalidate(); await service.stepUp!(); } : undefined;
 const unresolved = state.entries.some(e => e.phase === "pending" || e.phase === "unknown" && !preparationSettled(state.entries,e));
 return { store, state, unresolved, enabled, capability, rationale, setRationale, reauthenticate, actor: session?.user, error: active?.error };
}
export type DocumentBinding = ReturnType<typeof useDocuments>;
