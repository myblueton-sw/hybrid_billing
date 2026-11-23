import { AdminError, type AdminClient } from "../../../../service/admin";
import type { SelectionFingerprint, SessionOperation } from "../../../../service/session-query";

export type EndIntent = { id: number; kind: "one" | "others"; fingerprint: SelectionFingerprint; state: "submitting" | "unknown" | "recovering" | "unavailable" | "accepted"; operation?: SessionOperation };
export type IntentSnapshot = { intent: EndIntent | null; error: unknown; version: number };
export type IntentStore = { scope: string; get: () => IntentSnapshot; subscribe: (listener: () => void) => () => void; update: (snapshot: IntentSnapshot) => void };
const stores = new WeakMap<AdminClient, IntentStore>();

export function intentStore(api: AdminClient, scope: string): IntentStore | null {
  const existing = stores.get(api);
  if (existing) return existing.scope === scope ? existing : null;
  let snapshot: IntentSnapshot = { intent: null, error: null, version: 0 };
  const listeners = new Set<() => void>();
  const store: IntentStore = { scope, get: () => snapshot, subscribe(listener) { listeners.add(listener); return () => { listeners.delete(listener); }; },
    update(next) { snapshot = next; for (const listener of listeners) listener(); } };
  stores.set(api, store);
  return store;
}

export function intentLocked(snapshot: IntentSnapshot): boolean {
  return !!snapshot.intent && snapshot.intent.state !== "accepted";
}
function commit(store: IntentStore, prior: EndIntent, next: EndIntent, error: unknown = null): boolean {
  const current = store.get();
  if (current.intent !== prior) return false;
  store.update({ intent: next, error, version: current.version + 1 });
  return true;
}
function matching(op: SessionOperation, intent: EndIntent): boolean {
  return op.selection === intent.fingerprint.selection && op.digest === intent.fingerprint.digest && op.recorded_count === intent.fingerprint.count && op.kind === intent.kind;
}
function accept(store: IntentStore, prior: EndIntent, op: SessionOperation): void {
  if (!matching(op, prior)) { commit(store, prior, { ...prior, state: "unknown" }, new AdminError("invalid")); return; }
  commit(store, prior, { ...prior, state: op.state === "unconfirmed" ? "unknown" : "accepted", operation: op });
}
function noEffect(error: unknown): boolean {
  return error instanceof AdminError && (["invalid_request", "revision_conflict", "rate_limited", "reauthentication_required", "prerequisite_failed", "unauthenticated", "csrf_refused"].includes(error.code) || error.failure === "reauth_cancelled");
}

export async function submitSelected(store: IntentStore, api: AdminClient, fingerprint: SelectionFingerprint, kind: "one" | "others"): Promise<void> {
  const current = store.get();
  if (intentLocked(current)) return;
  const intent: EndIntent = { id: current.version + 1, kind, fingerprint: { ...fingerprint }, state: "submitting" };
  store.update({ intent, error: null, version: current.version + 1 });
  try { accept(store, intent, await api.endSelectedSessions(intent.fingerprint)); }
  catch (error) {
    if (store.get().intent !== intent) return;
    if (noEffect(error)) store.update({ intent: null, error, version: store.get().version + 1 });
    else commit(store, intent, { ...intent, state: "unknown" }, error);
  }
}

export async function recoverSelected(store: IntentStore, api: AdminClient): Promise<void> {
  const current = store.get().intent;
  if (!current || current.state === "submitting" || current.state === "recovering") return;
  const intent: EndIntent = { ...current, state: "recovering" };
  if (!commit(store, current, intent)) return;
  try {
    const preview = await api.previewSessionSelection(intent.fingerprint.selection, { limit: 1 });
    if (store.get().intent !== intent) return;
    if (preview.selection !== intent.fingerprint.selection || preview.digest !== intent.fingerprint.digest || preview.count !== intent.fingerprint.count || preview.kind !== intent.kind) {
      commit(store, intent, { ...intent, state: "unavailable" }, new AdminError("invalid")); return;
    }
    if (preview.operation) accept(store, intent, await api.sessionOperation(preview.operation));
    else commit(store, intent, { ...intent, state: "unknown" });
  } catch (error) { commit(store, intent, { ...intent, state: "unavailable" }, error); }
}

export async function replaySelected(store: IntentStore, api: AdminClient): Promise<void> {
  const current = store.get().intent;
  if (!current || current.state === "submitting" || current.state === "recovering" || current.state === "accepted") return;
  const intent: EndIntent = { ...current, state: "submitting" };
  if (!commit(store, current, intent)) return;
  try { accept(store, intent, await api.endSelectedSessions(intent.fingerprint)); }
  catch (error) { commit(store, intent, { ...intent, state: "unavailable" }, error); }
}
