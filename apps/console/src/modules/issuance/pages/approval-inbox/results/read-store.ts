import { inspectWithAdvice } from "./inspection";
import { freeze, id, ResultError, type Context, type Preparation, type ReadCommand } from "./contract";
import { loadContext, loadPreparations } from "./reads";
import { loadHistory } from "./history";
export type ReadState<T> = Readonly<{ phase: "idle" | "loading" }> | Readonly<{ phase: "loaded"; data: T }> | Readonly<{ phase: "error"; code: string }>;
export type ResultReadState = Readonly<{
 preparations: ReadState<Awaited<ReturnType<typeof loadPreparations>>>;
 selected?: Preparation;
 context: ReadState<Context>;
 history: ReadState<Awaited<ReturnType<typeof loadHistory>>>;
 detail?: ReadState<Awaited<ReturnType<typeof inspectWithAdvice>>>;
}>;
const empty = (): ResultReadState => freeze({ preparations: { phase: "idle" }, context: { phase: "idle" }, history: { phase: "idle" } });
const failure = (e: unknown): ReadState<never> => ({ phase: "error", code: e instanceof ResultError ? e.code : "service_unavailable" });
export class ResultReadStore {
 private state: ResultReadState = empty();
 private listeners = new Set<() => void>();
 private epoch = 0;
 private selection = 0;
 private inspection = 0;
 private disposed = false;
 constructor(private readonly workspace: string, private readonly read: ReadCommand) {
  if (!id(workspace)) throw new ResultError("invalid_result_request");
 }
 snapshot = () => this.state;
 subscribe = (listener: () => void) => { this.listeners.add(listener); return () => { this.listeners.delete(listener); }; };
 private publish(state: ResultReadState) { this.state = freeze(state); for (const listener of this.listeners) listener(); }
 invalidate() {
  this.epoch++; this.selection++; this.inspection++;
  this.publish(empty());
 }
 dispose() { this.disposed = true; this.invalidate(); this.listeners.clear(); }
 async followMutation(scope: Preparation, result: string, work: Promise<void>) {
  if (this.disposed) { await work; return; }
  this.invalidate();
  const pendingEpoch = this.epoch, pendingSelection = this.selection;
  await work;
  if (this.disposed || this.epoch !== pendingEpoch || this.selection !== pendingSelection) return;
  const refreshing = this.refresh(), epoch = this.epoch, selection = this.selection;
  await refreshing;
  if (this.disposed || this.epoch !== epoch || this.selection !== selection || this.state.preparations.phase !== "loaded") return;
  const fresh = this.state.preparations.data.rows.find(row => row.preparation === scope.preparation);
  if (!fresh || fresh.preparation_digest !== scope.preparation_digest || fresh.document_digest !== scope.document_digest) return;
  const selecting = this.select(scope.preparation), selectedGeneration = this.selection, inspectedGeneration = this.inspection;
  await selecting;
  if (this.disposed || this.epoch !== epoch || this.selection !== selectedGeneration || this.inspection !== inspectedGeneration) return;
  await this.inspect(result);
 }
 async refresh() {
  if (this.disposed) return;
  const epoch = ++this.epoch; this.selection++;
  this.publish({ preparations: { phase: "loading" }, context: { phase: "idle" }, history: { phase: "idle" } });
  try {
   const data = await loadPreparations(this.read, this.workspace);
   if (!this.disposed && this.epoch === epoch) this.publish({ ...this.state, preparations: { phase: "loaded", data } });
  } catch (e) {
   if (!this.disposed && this.epoch === epoch) this.publish({ ...this.state, preparations: failure(e) });
  }
 }
 async select(preparation: string | undefined) {
  if (this.disposed) return;
  const selected = this.state.preparations.phase === "loaded" ? this.state.preparations.data.rows.find(v => v.preparation === preparation) : undefined;
  const epoch = this.epoch, selection = ++this.selection;
  this.inspection++;
  this.publish({ ...this.state, selected, detail: undefined, context: { phase: selected ? "loading" : "idle" }, history: { phase: selected ? "loading" : "idle" } });
  if (!selected) return;
  const current = () => !this.disposed && this.epoch === epoch && this.selection === selection;
  const context = loadContext(this.read, selected).then(data => { if (current()) this.publish({ ...this.state, context: { phase: "loaded", data } }); }, e => { if (current()) this.publish({ ...this.state, context: failure(e) }); });
  const history = loadHistory(this.read, selected).then(data => { if (current()) this.publish({ ...this.state, history: { phase: "loaded", data } }); }, e => { if (current()) this.publish({ ...this.state, history: failure(e) }); });
  await Promise.all([context, history]);
 }
 async inspect(result: string) {
  if(this.disposed||!this.state.selected||this.state.history.phase!=="loaded")return;
  const scope=this.state.selected,row=this.state.history.data.rows.find(v=>v.result===result);if(!row)return;
  const epoch=this.epoch,selection=this.selection,inspection=++this.inspection;
  this.publish({...this.state,detail:{phase:"loading"}});
  const current=()=>!this.disposed&&epoch===this.epoch&&selection===this.selection&&inspection===this.inspection;
  try{const data=await inspectWithAdvice(this.read,scope,row,this.workspace);if(current())this.publish({...this.state,detail:{phase:"loaded",data}});}
  catch(e){if(current())this.publish({...this.state,detail:failure(e)});}
 }

}
