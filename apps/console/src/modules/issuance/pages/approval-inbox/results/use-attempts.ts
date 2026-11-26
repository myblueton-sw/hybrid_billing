import { useEffect, useState, useSyncExternalStore } from "react";
import { useService } from "../../../../../service/context";
import { sessionResultAttempts, type ResultAttempt, type ResultAttempts } from "./attempts";
import { ResultError } from "./contract";
const idle: readonly ResultAttempt[] = Object.freeze([]);
const noSubscribe = () => () => {}, snapshotIdle = () => idle;
export function useResultAttempts() {
 const service = useService(), session = service?.session;
 const key = session ? JSON.stringify([session.user,session.workspace,session.csrf]) : "";
 const [binding,setBinding] = useState<{ key: string; store?: ResultAttempts; error?: string }>();
 useEffect(() => {
  if (!session) { setBinding(undefined); return; }
  try { setBinding({ key, store: sessionResultAttempts(session) }); }
  catch (e) { setBinding({ key, error: e instanceof ResultError ? e.code : "session_unavailable" }); }
 }, [key]);
 const current = binding?.key === key ? binding : undefined, store = current?.store;
 const entries = useSyncExternalStore(store?.subscribe ?? noSubscribe,store?.snapshot ?? snapshotIdle,snapshotIdle);
 return { store, entries, error: current?.error };
}
