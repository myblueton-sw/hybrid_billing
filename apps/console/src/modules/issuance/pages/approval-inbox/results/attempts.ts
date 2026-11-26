import { reviewAction, type ActionReview } from "./actions";
import type { Session } from "../../../../../service/api";
import { freeze, json, object, preparationOf, ResultError, type Preparation } from "./contract";
import { freezeIntent, resultMutationTransport, type Intent, type Receipt } from "./mutations";
export type ResultAttempt = Readonly<{ scope: Preparation; intent: Intent; phase: "pending" | "unknown" | "refused" | "recorded"; receipt?: Receipt; error?: string; operation?: string }>;
export class ResultAttempts {
 private entries: readonly ResultAttempt[] = freeze([]);
 private listeners = new Set<() => void>();
 constructor(private readonly send: (intent: Intent) => Promise<Receipt>, private readonly actor = "") {}
 snapshot = () => this.entries;
 subscribe = (listener: () => void) => { this.listeners.add(listener); return () => { this.listeners.delete(listener); }; };
 private update(entry: ResultAttempt) {
  this.entries = freeze([...this.entries.filter(e => e.intent.attempt.key !== entry.intent.attempt.key), entry]);
  for (const listener of this.listeners) listener();
 }
 submit(scope: Preparation, input: Intent) {
  const selected = preparationOf(JSON.stringify(scope)), intent = freezeIntent(input);
  if (intent.action !== "propose") throw new ResultError("invalid_result_request");
  if (intent.action === "propose") {
   const packet = json(intent.fields.packet);
   if (!object(packet) || packet.Preparation !== selected.preparation || packet.PreparationDigest !== selected.preparation_digest || packet.Connection !== selected.connection || packet.Currency !== selected.currency || packet.Customer !== selected.customer || packet.Period !== selected.period) throw new ResultError("invalid_result_scope");
  }
  if (this.entries.length >= 100) throw new ResultError("attempt_capacity");
  if (this.entries.some(e => e.intent.attempt.key === intent.attempt.key || e.scope.preparation === selected.preparation && this.unresolved(e))) throw new ResultError("unresolved_result_attempt");
  return this.run(freeze({ scope: selected, intent, phase: "pending" }));
 }
 unresolved(entry: ResultAttempt) { return entry.phase === "pending" || entry.phase === "unknown" && !this.settled(entry); }
 settled(entry: ResultAttempt) {
  return entry.intent.action === "apply" && this.entries.some(e=>e.phase==="recorded"&&e.intent.action==="apply"&&e.intent.target===entry.intent.target&&e.intent.fields.digest===entry.intent.fields.digest&&e.receipt?.applicationState==="recorded");
 }
 submitAction(input: ActionReview) {
  const value=reviewAction(input.scope,input.detail,this.actor,input.action,input.intent.rationale,input.intent.attempt);
  if(JSON.stringify(value.intent)!==JSON.stringify(input.intent))throw new ResultError("invalid_result_request");
  if(this.entries.length>=100)throw new ResultError("attempt_capacity");
  if(this.entries.some(e=>e.intent.attempt.key===value.intent.attempt.key||e.scope.preparation===value.scope.preparation&&(e.phase==="pending"||e.phase==="unknown"&&!this.settled(e)&&((value.action!=="recover"&&value.action!=="revoke")||e.intent.action!=="apply"||e.intent.target!==value.intent.target||e.intent.fields.digest!==value.intent.fields.digest))))throw new ResultError("unresolved_result_attempt");
  return this.run(freeze({scope:value.scope,intent:value.intent,phase:"pending"}));
 }
 async retry(key: string) {
  const entry = this.entries.find(e => e.intent.attempt.key === key);
  if (!entry || entry.phase !== "unknown" || this.settled(entry) || this.entries.some(e => e.phase === "pending" && e.scope.preparation === entry.scope.preparation)) return;
  await this.run(entry);
 }
 private async run(entry: ResultAttempt) {
  const wasUnknown = entry.phase === "unknown";
  this.update({ ...entry, phase: "pending" });
  try { const receipt = await this.send(entry.intent); this.update({ ...entry, phase: "recorded", receipt, error: undefined }); }
  catch (e) { this.update({ ...entry, phase: wasUnknown || !(e instanceof ResultError) || e.effect === "unknown" ? "unknown" : "refused", error: e instanceof ResultError ? e.code : "service_unavailable", operation: entry.operation || (e instanceof ResultError ? e.operation : "") }); }
 }
}
const stores = new Map<string, ResultAttempts>();
export function sessionResultAttempts(session: Session) {
 const key = JSON.stringify([session.user, session.workspace, session.csrf]);
 const found = stores.get(key); if (found) return found;
 if (stores.size >= 16) throw new ResultError("session_capacity");
 const store = new ResultAttempts(resultMutationTransport(session),session.user); stores.set(key, store); return store;
}
