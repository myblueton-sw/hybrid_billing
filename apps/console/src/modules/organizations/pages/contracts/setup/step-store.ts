import type { Attempt } from "../../../../../service/admin";
import type { Session } from "../../../../../service/api";
import { freeze, id, revision } from "../../../../issuance/pages/approval-inbox/results/contract";
import { SetupError } from "./context";

export type StepIntent = Readonly<{ operation: string; actor: string; workspace: string; sessionCSRF: string; revision: string; attempt: Readonly<Attempt> }>;
export type StepPhase = "pending" | "unknown" | "refused" | "recorded" | "reconciled";
export type StepEntry<I extends StepIntent, R> = Readonly<{ intent: I; subject: string; phase: StepPhase; error?: string; operationId?: string; receipt?: R; recovers?: string; form?: string; at: string; noRecord?: true }>;
export type StepState<I extends StepIntent, R> = Readonly<{ entries: readonly StepEntry<I, R>[]; drafts: Readonly<Record<string, Readonly<Record<string, string>>>> }>;
type Owner = Pick<Session, "user" | "workspace" | "csrf">;
export const noRecordBoundMs = 30 * 60 * 1000;

export class StepWriteStore<I extends StepIntent, R> {
  private state: StepState<I, R> = freeze({ entries: [], drafts: {} });
  private listeners = new Set<() => void>();
  private started = new Map<string, number>();
  constructor(private readonly owner: Readonly<Owner>, private readonly send: (intent: I) => Promise<R>, private readonly now: () => number = Date.now, private readonly elapsed: () => number = () => performance.now(), private readonly siblings: () => readonly StepEntry<I, R>[] = () => []) {}
  earlier(): readonly StepEntry<I, R>[] { return this.siblings().filter(e => e.phase === "pending" || e.phase === "unknown"); }
  private within(age: number): boolean { return Number.isFinite(age) && age >= 0 && age < noRecordBoundMs; }
  private fresh(entry: StepEntry<I, R>): boolean { const began = this.started.get(entry.intent.attempt.key); return began !== undefined && this.within(this.elapsed() - began) && this.within(this.now() - Date.parse(entry.at)); }
  snapshot = () => this.state;
  subscribe = (listener: () => void) => { this.listeners.add(listener); return () => { this.listeners.delete(listener); }; };
  private update(patch: Partial<StepState<I, R>>) { this.state = freeze({ ...this.state, ...patch }); this.listeners.forEach(listener => listener()); }
  private put(value: StepEntry<I, R>) { this.update({ entries: [...this.state.entries.filter(e => e.intent.attempt.key !== value.intent.attempt.key), value] }); }
  draft(name: string): Readonly<Record<string, string>> { return this.state.drafts[name] ?? {}; }
  edit(name: string, patch: Readonly<Record<string, string>>) { this.update({ drafts: { ...this.state.drafts, [name]: freeze({ ...this.draft(name), ...patch }) } }); }
  clear(name: string) { const { [name]: _, ...drafts } = this.state.drafts; this.update({ drafts }); }
  dirty(): boolean { return Object.values(this.state.drafts).some(d => Object.values(d).some(v => v !== "")) || this.unresolved(); }
  unresolved(): boolean { return this.state.entries.some(e => e.phase === "pending" || e.phase === "unknown"); }
  private admit(intent: I) {
    if (intent.actor !== this.owner.user || intent.workspace !== this.owner.workspace || intent.sessionCSRF !== this.owner.csrf) throw new SetupError("session_changed");
    if (![intent.attempt?.request, intent.attempt?.key].every(id) || !revision(intent.revision)) throw new SetupError("invalid_setup_request");
    if (this.state.entries.length >= 100) throw new SetupError("attempt_capacity");
    if (this.state.entries.some(e => e.intent.attempt.key === intent.attempt.key || e.intent.attempt.request === intent.attempt.request)) throw new SetupError("duplicate_attempt");
  }
  async submit(intent: I, subject: string, form?: string) {
    this.admit(intent);
    if (this.unresolved()) throw new SetupError("unresolved_operation");
    await this.run({ intent, subject, phase: "pending", form, at: new Date(this.now()).toISOString() });
  }
  async retry(key: string) {
    const entry = this.state.entries.find(e => e.intent.attempt.key === key);
    if (!entry || entry.phase !== "unknown" || this.state.entries.some(e => e.phase === "pending")) throw new SetupError("retry_unavailable");
    await this.run({ ...entry, phase: "pending", error: undefined }, true);
  }
  async recover(key: string, attempt: Attempt, current: string) {
    const entry = this.state.entries.find(e => e.intent.attempt.key === key);
    if (!entry || entry.phase !== "unknown" || entry.recovers || this.state.entries.some(e => e.phase === "pending" || e.phase === "unknown" && e !== entry)) throw new SetupError("recovery_unavailable");
    const intent = freeze({ ...entry.intent, attempt: freeze({ ...attempt }), revision: current }) as I;
    this.admit(intent);
    await this.run({ intent, subject: entry.subject, phase: "pending", recovers: key, form: entry.form, at: new Date(this.now()).toISOString() });
  }
  private async run(entry: StepEntry<I, R>, repeated = false) {
    if (!repeated) this.started.set(entry.intent.attempt.key, this.elapsed());
    this.put(entry);
    try {
      const receipt = await this.send(entry.intent);
      this.put({ ...entry, phase: "recorded", receipt, error: undefined });
      const key = entry.intent.attempt.key, origin = entry.recovers;
      this.update({ entries: this.state.entries.map(e => e.phase === "unknown" && (e.recovers === key || origin !== undefined && (e.intent.attempt.key === origin || e.recovers === origin)) ? { ...e, phase: "reconciled" as const, receipt } : e) });
      if (entry.form) this.clear(entry.form);
    } catch (error) {
      const known = error instanceof SetupError;
      const noRecord = repeated && known && error.code === "revision_conflict" && error.effect === "none" && !error.operation && !entry.operationId && this.fresh(entry);
      this.put({ ...entry, phase: !noRecord && (repeated || !known || error.effect === "unknown") ? "unknown" : "refused", error: known ? error.code : "service_unavailable", operationId: known && error.operation ? error.operation : entry.operationId, ...(noRecord ? { noRecord: true as const } : {}) });
    }
  }
}

const stores = new Map<string, StepWriteStore<StepIntent, unknown>>();
export function sessionStepStore<I extends StepIntent, R>(kind: string, session: Owner, send: () => (intent: I) => Promise<R>): StepWriteStore<I, R> {
  const key = JSON.stringify([kind, session.user, session.workspace, session.csrf]);
  let store = stores.get(key) as StepWriteStore<I, R> | undefined;
  if (!store) {
    if (stores.size >= 32) throw new SetupError("session_capacity");
    const others = () => [...stores.entries()].filter(([k]) => { const [otherKind, user, workspace, csrf] = JSON.parse(k) as string[]; return otherKind === kind && user === session.user && workspace === session.workspace && csrf !== session.csrf; }).flatMap(([, other]) => other.snapshot().entries as readonly StepEntry<I, R>[]);
    store = new StepWriteStore<I, R>(freeze({ user: session.user, workspace: session.workspace, csrf: session.csrf }), send(), Date.now, () => performance.now(), others);
    stores.set(key, store as unknown as StepWriteStore<StepIntent, unknown>);
  }
  return store;
}
export function anyStepDirty(): boolean { return [...stores.values()].some(store => store.dirty()); }
