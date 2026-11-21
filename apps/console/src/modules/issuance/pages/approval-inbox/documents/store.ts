import type { Session } from "../../../../../service/api";
import { attempt } from "../../../../../service/admin";
import { DocumentError, freeze, type DocumentRow, type FormContext, type HistoryRow, type PreparationRow, type RequestIdentity } from "./contract";
import { documentGateway, type DocumentGateway } from "./gateway";
import type { PreparationInspection } from "./inspection";
import { emptyDraft, packetOf, PacketError, type PacketDraft } from "./packet";
import { freezeIntent, type DocumentAction, type Intent, type Receipt } from "./transport";
export type Query<T> = Readonly<{ phase: "idle" | "loading" | "ready" | "failed"; value?: T; error?: string }>;
type History = Readonly<{ revision: string; rows: readonly HistoryRow[] }>;
type Detail = Awaited<ReturnType<DocumentGateway["document"]>>;
export type DocumentAttempt = Readonly<{ scope: RequestIdentity; intent: Intent; phase: "pending" | "unknown" | "refused" | "recorded"; receipt?: Receipt; error?: string; operation?: string }>;
export type DocumentState = Readonly<{
 selection: string; scope?: RequestIdentity; context: Query<FormContext>; documents: Query<History>; preparations: Query<History>;
 document: Query<Detail>; preparation: Query<PreparationInspection>; draft?: PacketDraft; review?: Intent;
 error?: string; invalidFields: readonly string[]; entries: readonly DocumentAttempt[];
}>;
const idle = <T>(): Query<T> => ({ phase: "idle" });
const code = (e: unknown) => e instanceof DocumentError ? e.code : "service_unavailable";
export class DocumentStore {
 private state: DocumentState = freeze({ selection: "", context: idle(), documents: idle(), preparations: idle(), document: idle(), preparation: idle(), invalidFields: [], entries: [] });
 private listeners = new Set<() => void>();
 private epochs = { context: 0, documents: 0, preparations: 0, document: 0, preparation: 0 };
 constructor(private session: Readonly<Session>, private client: DocumentGateway = documentGateway(session)) { this.session = freeze({ ...session }); }
 snapshot = () => this.state;
 subscribe = (listener: () => void) => { this.listeners.add(listener); return () => { this.listeners.delete(listener); }; };
 private update(patch: Partial<DocumentState>) { this.state = freeze({ ...this.state, ...patch }); for (const listener of this.listeners) listener(); }
 select(selection: string, scope?: RequestIdentity) {
  const next = JSON.stringify([selection, scope]);
  if (next === this.state.selection) return;
  this.invalidate();
  this.update({ selection: next, scope: scope ? freeze({ ...scope }) : undefined, documents: idle(), preparations: idle() });
 }
 invalidate() {
  for (const key of Object.keys(this.epochs) as (keyof typeof this.epochs)[]) this.epochs[key]++;
  this.update({ context: idle(), document: idle(), preparation: idle(), draft: undefined, review: undefined, error: undefined, invalidFields: [], documents: idle(), preparations: idle() });
 }
 async loadContext() {
  const scope = this.state.scope; if (!scope || this.state.context.phase === "loading") return;
  const epoch = ++this.epochs.context;
  this.update({ context: { phase: "loading" }, draft: undefined, review: undefined, error: undefined, invalidFields: [] });
  try { const value = await this.client.context(scope); if (epoch === this.epochs.context) this.update({ context: { phase: "ready", value }, draft: emptyDraft(value) }); }
  catch (e) { if (epoch === this.epochs.context) this.update({ context: { phase: "failed", error: code(e) } }); }
 }
 async loadHistory(kind: "documents" | "preparations") {
  const scope = this.state.scope; if (!scope || this.state[kind].phase === "loading") return;
  const epoch = ++this.epochs[kind];
  if (kind === "documents") { this.epochs.document++; this.update({ document: idle(), review: undefined }); }
  else { this.epochs.preparation++; this.update({ preparation: idle(), review: undefined }); }
  this.update({ [kind]: { phase: "loading" } });
  try { const value = await this.client.history(scope, kind); if (epoch === this.epochs[kind]) this.update({ [kind]: { phase: "ready", value } }); }
  catch (e) { if (epoch === this.epochs[kind]) this.update({ [kind]: { phase: "failed", error: code(e) } }); }
 }
 async inspectDocument(target: string) {
  const scope = this.state.scope, row = this.state.documents.value?.rows.find(r => "digest" in r && r.document === target) as DocumentRow | undefined;
  if (!scope || !row || this.state.documents.phase !== "ready") return;
  const epoch = ++this.epochs.document; this.update({ document: { phase: "loading" }, review: undefined, error: undefined });
  try { const value = await this.client.document(scope, row); if (epoch === this.epochs.document) this.update({ document: { phase: "ready", value } }); }
  catch (e) { if (epoch === this.epochs.document) this.update({ document: { phase: "failed", error: code(e) } }); }
 }
 async inspectPreparation(target: string) {
  const row = this.state.preparations.value?.rows.find(r => "preparation" in r && r.preparation === target) as PreparationRow | undefined;
  if (!row || this.state.preparations.phase !== "ready") return;
  const epoch = ++this.epochs.preparation; this.update({ preparation: { phase: "loading" }, review: undefined, error: undefined });
  try { const value = await this.client.preparation(row); if (epoch === this.epochs.preparation) this.update({ preparation: { phase: "ready", value } }); }
  catch (e) { if (epoch === this.epochs.preparation) this.update({ preparation: { phase: "failed", error: code(e) } }); }
 }
 edit(draft: PacketDraft) { if (this.state.context.phase === "ready" && !this.state.review) this.update({ draft: freeze(structuredClone(draft)), error: undefined, invalidFields: [] }); }
 cancelReview() { this.update({ review: undefined, error: undefined, invalidFields: [] }); }
 private guarded(work: () => Intent) {
  try { this.update({ review: freezeIntent(work()), error: undefined, invalidFields: [] }); }
  catch (e) { this.update({ review: undefined, error: code(e), invalidFields: e instanceof PacketError ? e.fields : [] }); }
 }
 reviewProposal() {
  const { context, draft } = this.state; if (context.phase !== "ready" || !context.value || !draft) return;
  this.guarded((): Intent => {
   if (this.state.entries.some(e => e.scope.request === context.value!.request && e.scope.digest === context.value!.digest && e.intent.action === "propose" && (e.phase === "pending" || e.phase === "unknown"))) throw new DocumentError("unresolved_document_attempt");
   const packet = packetOf(context.value!, draft), tried = attempt();
   return { action: "propose", target: `document-${tried.request}`, fields: { packet: packet.text }, rationale: packet.rationale, revision: context.value!.revision, attempt: tried };
  });
 }
 private settled(entry: DocumentAttempt) {
  return this.state.entries.some(e => e.phase === "recorded" && e.intent.action === "prepare" && entry.intent.action === "prepare" && e.intent.target === entry.intent.target && e.intent.fields.document === entry.intent.fields.document && e.intent.fields.digest === entry.intent.fields.digest);
 }
 private unresolved(document: string) {
  return this.state.entries.some(e => (e.phase === "pending" || e.phase === "unknown" && !this.settled(e)) && (e.intent.target === document || e.intent.fields.document === document));
 }
 reviewAction(action: Exclude<DocumentAction, "propose">, rationale: string, recovery = false) {
  this.guarded((): Intent => {
   const d = this.state.document.value;
   if (this.state.document.phase !== "ready" || !d || d.advice.readiness[action].state !== "ready" || d.advice.permissions[action].state !== "allowed") throw new DocumentError("prerequisite_failed");
   const document = d.inspection.row.document, hash = d.inspection.row.digest, tried = attempt();
   if (action !== "prepare") {
    if (this.unresolved(document)) throw new DocumentError("unresolved_document_attempt");
    return { action, target: document, fields: { digest: hash }, rationale, revision: d.advice.revision, attempt: tried };
   }
   const history = this.state.preparations;
   if (history.phase !== "ready" || !history.value || history.value.revision !== d.advice.revision) throw new DocumentError("revision_conflict");
   if (recovery) {
    const observed = this.state.preparation.value;
    if (this.state.preparation.phase !== "ready" || !observed || observed.revision !== d.advice.revision || observed.row.document !== document || observed.row.document_digest !== hash || observed.row.actor !== this.session.user || observed.row.recorded_receipt || observed.observation === "conflict") throw new DocumentError("prerequisite_failed");
    const target = observed.row.preparation;
    if (this.state.entries.some(e => e.phase === "pending" && e.intent.target === target)) throw new DocumentError("attempt_pending");
    return { action, target, fields: { document, digest: hash }, rationale, revision: d.advice.revision, attempt: tried };
   }
   if (this.unresolved(document) || history.value.rows.some(r => "preparation" in r && r.document === document && !r.recorded_receipt)) throw new DocumentError("unresolved_document_attempt");
   return { action, target: `preparation-${tried.request}`, fields: { document, digest: hash }, rationale, revision: d.advice.revision, attempt: tried };
  });
 }
 async submit() {
  const { review, scope } = this.state; if (!review || !scope) return;
  if (this.state.entries.length >= 100) { this.update({ error: "attempt_capacity" }); return; }
  if (this.state.entries.some(e => e.phase === "pending" && (e.intent.target === review.target || e.scope.request === scope.request))) { this.update({ error: "attempt_pending" }); return; }
  this.update({ review: undefined });
  return this.run(freeze({ scope, intent: review, phase: "pending" }));
 }
 async retry(key: string) {
  const entry = this.state.entries.find(e => e.intent.attempt.key === key);
  if (!entry || entry.phase !== "unknown" || this.settled(entry) || this.state.entries.some(e => e.phase === "pending" && (e.intent.target === entry.intent.target || e.scope.request === entry.scope.request))) return;
  return this.run(entry);
 }
 private setEntry(entry: DocumentAttempt) {
  const entries = this.state.entries.filter(e => e.intent.attempt.key !== entry.intent.attempt.key);
  this.update({ entries: [...entries, entry] });
 }
 private async run(entry: DocumentAttempt) {
  const wasUnknown = entry.phase === "unknown";
  this.setEntry({ ...entry, phase: "pending" });
  this.invalidate();
  try { const receipt = await this.client.submit(entry.intent); this.setEntry({ ...entry, phase: "recorded", receipt, error: undefined }); }
  catch (e) { this.setEntry({ ...entry, phase: wasUnknown || !(e instanceof DocumentError) || e.effect === "unknown" ? "unknown" : "refused", error: code(e), operation: entry.operation || (e instanceof DocumentError ? e.operation : "") }); }
  finally { if (this.state.scope?.request === entry.scope.request && this.state.scope?.digest === entry.scope.digest) this.invalidate(); }
 }
}
const stores = new Map<string, DocumentStore>();
export function sessionDocumentStore(session: Session): DocumentStore {
 const key = JSON.stringify([session.user, session.workspace, session.csrf]);
 const found = stores.get(key); if (found) return found;
 if (stores.size >= 16) throw new DocumentError("session_capacity");
 const store = new DocumentStore(session); stores.set(key, store); return store;
}
