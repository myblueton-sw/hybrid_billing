import type { Session } from "../../../../service/api";
import type { ApprovalRecord } from "./contract";
import { decisionClient, DecisionError, submissionOf, type DecisionAction, type Preparation, type Receipt, type Submission } from "./decision-api";

type Client = ReturnType<typeof decisionClient>;
function frozenRecord(record: ApprovalRecord): ApprovalRecord {
  const freeze = (value: unknown) => {
    if (value && typeof value === "object") {
      for (const child of Object.values(value)) freeze(child);
      Object.freeze(value);
    }
  };
  const result = structuredClone(record);
  freeze(result);
  return result;
}
export type DecisionEntry = Readonly<{ phase: "pending" | "unknown" | "refused" | "recorded"; submission: Submission; receipt?: Receipt; error?: string; operation?: string }>;
export type DecisionState = Readonly<{
  selection: string; record?: ApprovalRecord; checking: boolean; preparation?: Preparation; error?: string;
  action?: DecisionAction; rationale: string; reason: string; entries: Readonly<Record<string, DecisionEntry>>;
}>;
export class DecisionStore {
  private state: DecisionState = Object.freeze({ selection: "", checking: false, rationale: "", reason: "", entries: Object.freeze(Object.create(null)) });
  private listeners = new Set<() => void>();
  private generation = 0;
  constructor(private client: Client) {}
  snapshot = () => this.state;
  subscribe = (listener: () => void) => { this.listeners.add(listener); return () => { this.listeners.delete(listener); }; };
  private update(patch: Partial<DecisionState>) {
    this.state = Object.freeze({ ...this.state, ...patch });
    for (const listener of this.listeners) listener();
  }
  private entry(target: string, entry: DecisionEntry) {
    this.update({ entries: Object.freeze(Object.assign(Object.create(null), this.state.entries, { [target]: Object.freeze(entry) })) });
  }
  select(scope: string, record?: ApprovalRecord) {
    const selection = JSON.stringify([scope, record?.id, record?.evidence.revision]);
    if (selection === this.state.selection) return;
    this.generation++;
    this.update({ selection, record: record ? frozenRecord(record) : undefined, checking: false, preparation: undefined, error: undefined, action: undefined, rationale: "", reason: "" });
  }
  invalidatePreparation() {
    this.generation++;
    this.update({ checking: false, preparation: undefined, error: undefined, action: undefined, rationale: "", reason: "" });
  }
  async check() {
    const record = this.state.record;
    if (!record || this.state.checking) return;
    const entry = this.state.entries[record.id];
    if (entry && entry.phase !== "refused") return;
    const generation = ++this.generation;
    this.update({ checking: true, preparation: undefined, error: undefined, action: undefined, rationale: "", reason: "" });
    try {
      const preparation = await this.client.prepare(record);
      if (generation === this.generation) this.update({ checking: false, preparation });
    } catch (error) {
      if (generation === this.generation) this.update({ checking: false, error: error instanceof DecisionError ? error.code : "service_unavailable" });
    }
  }
  choose(action: DecisionAction) {
    if (!this.state.preparation || this.state.preparation[action] !== "eligible") return;
    const entry = this.state.entries[this.state.preparation.request];
    if (entry && entry.phase !== "refused") return;
    this.update({ action, rationale: "", reason: "", error: undefined });
  }
  edit(rationale: string, reason: string) {
    if (!this.state.action) return;
    this.update({ rationale, reason, error: undefined });
  }
  cancel() { this.update({ action: undefined, rationale: "", reason: "", error: undefined }); }
  async submit() {
    const { preparation, action, rationale, reason } = this.state;
    if (!preparation || !action) return;
    const previous = this.state.entries[preparation.request];
    if (previous && previous.phase !== "refused") return;
    if (!previous && Object.keys(this.state.entries).length >= 100) { this.update({ error: "attempt_capacity" }); return; }
    let submission: Submission;
    try { submission = submissionOf(preparation, action, rationale, reason); }
    catch (error) { this.update({ error: error instanceof DecisionError ? error.code : "invalid_decision" }); return; }
    this.update({ preparation: undefined, action: undefined, error: undefined, rationale: "", reason: "" });
    return this.run(submission, false);
  }
  async retry(target: string) {
    const entry = this.state.entries[target];
    if (!entry || entry.phase !== "unknown") return;
    return this.run(entry.submission, true, entry.operation);
  }
  private async run(submission: Submission, wasUnknown: boolean, operation?: string) {
    this.entry(submission.target, { phase: "pending", submission, operation });
    try {
      const receipt = await this.client.submit(submission);
      this.entry(submission.target, { phase: "recorded", submission, receipt, operation });
    } catch (error) {
      const known = error instanceof DecisionError;
      this.entry(submission.target, { phase: wasUnknown || !known || error.effect === "unknown" ? "unknown" : "refused", submission, error: known ? error.code : "service_unavailable", operation: operation || (known ? error.operation : "") });
    }
  }
}
const stores = new Map<string, DecisionStore>();
export function sessionDecisionStore(session: Session): DecisionStore {
  const key = JSON.stringify([session.user, session.workspace, session.csrf]);
  const existing = stores.get(key);
  if (existing) return existing;
  if (stores.size >= 16) throw new DecisionError("session_capacity", "none");
  const store = new DecisionStore(decisionClient(session));
  stores.set(key, store);
  return store;
}
