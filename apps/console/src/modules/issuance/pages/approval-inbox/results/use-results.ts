import { useEffect, useState, useSyncExternalStore } from "react";
import { useService } from "../../../../../service/context";
import { freeze } from "./contract";
import { ResultReadStore, type ResultReadState } from "./read-store";
import { resultReadTransport } from "./transport";
const idle = freeze<ResultReadState>({ preparations: { phase: "idle" }, context: { phase: "idle" }, history: { phase: "idle" } });
const subscribeIdle = () => () => {}, snapshotIdle = () => idle;
export function useResults(enabled: boolean) {
 const service = useService(), session = service?.session;
 const key = enabled && session ? JSON.stringify([session.user, session.workspace, session.csrf]) : "";
 const [binding, setBinding] = useState<{ key: string; store: ResultReadStore }>();
 useEffect(() => {
  if (!key || !session) { setBinding(undefined); return; }
  const store = new ResultReadStore(session.workspace, resultReadTransport(session));
  setBinding({ key, store });
  void store.refresh();
  return () => store.dispose();
 }, [key]);
 const store = key && binding?.key === key ? binding.store : undefined;
 const state = useSyncExternalStore(store?.subscribe ?? subscribeIdle, store?.snapshot ?? snapshotIdle, snapshotIdle);
 return { store, state, authenticated: !!session, actor: session?.user };
}
