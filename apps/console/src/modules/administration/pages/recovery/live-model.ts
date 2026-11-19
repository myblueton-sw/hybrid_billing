import { Refusals, type RefusalDetail } from "../../../../service/refusal";
import type { IsolationRow, ReconciliationRow, RecoveryPage, RecoveryProposal, ResumeRow, SetRow } from "../../../../service/recovery";

export const views = ["sets", "isolation", "reconciliation", "migration", "resume"];

export function viewOf(v: string): string {
  return views.includes(v) ? v : "sets";
}

export const shownLimit = 50;
export const maxReason = 256;
export const chunkBound = 100;
export const lostShown = 50;

export type CommandKey = "recovery.approve" | "recovery.withdraw" | "recovery.lock" | "recovery.propose_unlock";
export const commandKeys: CommandKey[] = ["recovery.approve", "recovery.withdraw", "recovery.lock", "recovery.propose_unlock"];

export function isProposalCommand(command: CommandKey): boolean {
  return command === "recovery.approve" || command === "recovery.withdraw";
}

export function viewOfCommand(command: CommandKey): string {
  return command === "recovery.lock" ? "isolation" : "resume";
}

const idChars = /^[A-Za-z0-9._:-]{1,128}$/;
export const installPrefix = "recovery-installation:";
export const proposalPrefix = "recovery-proposal-";

export function validTarget(t: string): boolean {
  return idChars.test(t);
}

export function validId(id: string): boolean {
  return idChars.test(id);
}

export function proposalTarget(p: RecoveryProposal): string {
  if (p.number === null || !/^recovery-proposal-[1-9][0-9]*$/.test(p.target)) return "";
  return p.target === proposalPrefix + p.number && validTarget(p.target) ? p.target : "";
}

export function installTarget(id: string): string {
  const t = installPrefix + id;
  return id && validTarget(t) ? t : "";
}

export function frozenTargetOk(command: CommandKey, target: string): boolean {
  if (!validTarget(target)) return false;
  return isProposalCommand(command) ? /^recovery-proposal-[1-9][0-9]*$/.test(target) : target.startsWith(installPrefix) && target.length > installPrefix.length;
}

export function commandsAllowed(page: RecoveryPage): boolean {
  return page.revision !== "";
}

export function timeKnown(iso: string): boolean {
  if (!iso) return false;
  const d = new Date(iso);
  return !Number.isNaN(d.getTime()) && d.getUTCFullYear() > 1;
}

export function actor(login: string, id: string): string {
  return login || id;
}

export const digestPattern = /^[0-9a-f]{64}$/;

export function setOf(page: RecoveryPage, id: string): SetRow | undefined {
  return (page.sets.rows ?? []).find((r) => r.installation === id);
}

export function isolationOf(page: RecoveryPage, id: string): IsolationRow | undefined {
  return (page.isolation.rows ?? []).find((r) => r.installation === id);
}

export function reconciliationOf(page: RecoveryPage, id: string): ReconciliationRow | undefined {
  return (page.reconciliation.rows ?? []).find((r) => r.installation === id);
}

export function resumeOf(page: RecoveryPage, id: string): ResumeRow | undefined {
  return (page.resume.rows ?? []).find((r) => r.installation === id);
}

export function proposalInstallation(p: RecoveryProposal): string {
  return p.summary.installation || p.installation;
}

export function generationOf(page: RecoveryPage, id: string): number | null {
  const i = isolationOf(page, id);
  if (i) return i.generation;
  const s = setOf(page, id);
  return s ? s.generation : null;
}

export function stateOf(page: RecoveryPage, id: string): string {
  return isolationOf(page, id)?.state ?? setOf(page, id)?.state ?? "";
}

export function noRestorePoint(row: SetRow): boolean {
  return row.recovery.state === "open" && row.recovery.domains !== null && row.recovery.domains.length === 0;
}

export type Evidence = { kind: "not_applicable" } | { kind: "no_point" } | { kind: "unknown" } | { kind: "counts"; verified: number; total: number; mismatched: number; pending: number };

export function evidenceOf(row: SetRow): Evidence {
  if (row.recovery.state === "none") return { kind: "not_applicable" };
  if (row.recovery.state !== "open" || row.recovery.domains === null) return { kind: "unknown" };
  const d = row.recovery.domains;
  if (d.length === 0) return { kind: "no_point" };
  return {
    kind: "counts",
    verified: d.filter((x) => x.status === "verified").length,
    total: d.length,
    mismatched: d.filter((x) => x.status === "mismatch").length,
    pending: d.filter((x) => x.status === "pending").length,
  };
}

export type Watermark =
  | { kind: "unknown" }
  | { kind: "outside" }
  | { kind: "mode"; mode: string }
  | { kind: "pending"; total: number | null; lost: number | null };

export function watermarkOf(page: RecoveryPage, id: string): Watermark {
  const r = resumeOf(page, id);
  if (!r) return { kind: "unknown" };
  if (r.mode === "lock_release" || r.mode === "none") return { kind: "outside" };
  if (r.mode !== "recovery_resume") return { kind: "mode", mode: r.mode };
  return { kind: "pending", total: r.pending_total, lost: r.lost === null ? null : r.lost.length };
}

export type HistoryView = { kind: "unknown" } | { kind: "empty" } | { kind: "list" } | { kind: "more"; total: number };

export function historyOf(row: SetRow): HistoryView {
  const list = row.history;
  if (list === null) return { kind: "unknown" };
  if (list.length === 0) return row.history_total === 0 ? { kind: "empty" } : { kind: "unknown" };
  return row.history_total !== null && row.history_total > list.length ? { kind: "more", total: row.history_total } : { kind: "list" };
}

export type Reconciled = { kind: "none" } | { kind: "no_point" } | { kind: "unknown" } | { kind: "counts"; reconciled: number; total: number };

export function reconciledOf(page: RecoveryPage, row: SetRow): Reconciled {
  if (noRestorePoint(row)) return { kind: "no_point" };
  const r = reconciliationOf(page, row.installation);
  if (!r) return { kind: "none" };
  if (r.externals === null) return { kind: "unknown" };
  return { kind: "counts", reconciled: r.externals.filter((x) => x.status === "reconciled").length, total: r.externals.length };
}

export type Chunks = { kind: "none" } | { kind: "unknown" } | { kind: "count_differs" } | { kind: "list"; first100: boolean; indices: number[] };

export function chunksOf(status: string, list: number[] | null): Chunks {
  if (status !== "mismatch") return { kind: "none" };
  if (list === null) return { kind: "unknown" };
  if (list.length === 0) return { kind: "count_differs" };
  return { kind: "list", first100: list.length === chunkBound, indices: list };
}

export type HighWater = { kind: "number"; n: number } | { kind: "no_series" } | { kind: "unknown" };

export function highWaterOf(series: string, n: number | null): HighWater {
  if (n !== null) return { kind: "number", n };
  return series ? { kind: "unknown" } : { kind: "no_series" };
}

export type Stopped = { kind: "time"; at: string } | { kind: "not_stopped" } | { kind: "unknown" };

export function stoppedOf(state: string, at: string): Stopped {
  if (timeKnown(at)) return { kind: "time", at };
  return !at && state === "serving" ? { kind: "not_stopped" } : { kind: "unknown" };
}

export type Calls = { kind: "listed" } | { kind: "empty" } | { kind: "not_shown" } | { kind: "unknown"; code: string };

export function callsOf(r: IsolationRow): Calls {
  if (r.open_calls_source === "not_shown") return { kind: "not_shown" };
  if (r.open_calls_source !== "recovery") return { kind: "unknown", code: r.open_calls_source };
  if (r.open_calls === null) return { kind: "unknown", code: "" };
  if (r.open_calls.length === 0) return r.open_calls_total === 0 ? { kind: "empty" } : { kind: "unknown", code: "" };
  return { kind: "listed" };
}

export function more(shown: number, total: number | null): boolean {
  return total !== null && total > shown;
}

export type LostText = { kind: "unknown" } | { kind: "none" } | { kind: "list"; shown: number[]; rest: number };

export function lostOf(list: number[] | null): LostText {
  if (list === null) return { kind: "unknown" };
  if (list.length === 0) return { kind: "none" };
  return { kind: "list", shown: list.slice(0, lostShown), rest: Math.max(0, list.length - lostShown) };
}

export function ordered(list: RecoveryProposal[]): { pending: RecoveryProposal[]; closed: RecoveryProposal[] } {
  const sorted = [...list].sort((a, b) => (b.number ?? -1) - (a.number ?? -1));
  return { pending: sorted.filter((p) => p.state === "pending"), closed: sorted.filter((p) => p.state !== "pending") };
}

export const proposalKinds = ["installation", "writer", "external", "verifier", "begin", "unlock", "prune", "abandon"];

export type SummaryPart = { id: string } | { num: number | null } | { date: string } | { skew: number | null } | { writerKind: string };
export type SummaryParts = { kind: string; known: boolean; parts: Record<string, SummaryPart> };

export function summaryOf(p: RecoveryProposal): SummaryParts {
  const s = p.summary;
  const known = proposalKinds.includes(p.kind);
  const parts: Record<string, SummaryPart> = { installation: { id: proposalInstallation(p) } };
  if (p.kind === "installation") parts.skew = { skew: s.skew_seconds };
  if (p.kind === "writer") parts.writer = { id: s.writer };
  if (p.kind === "external") {
    parts.external = { id: s.external };
    parts.kind = { writerKind: s.external_kind };
  }
  if (p.kind === "verifier") parts.verifier = { id: s.verifier };
  if (p.kind === "begin") {
    parts.as_of = { date: s.as_of };
    parts.point_generation = { num: s.point_generation };
    parts.manifest_digest = { id: s.manifest_digest };
  }
  if (p.kind === "unlock") {
    parts.unlock_generation = { num: s.unlock_generation };
    parts.lost_digest = { id: s.lost_digest };
  }
  return { kind: p.kind, known, parts };
}

export function pendingUnlockFor(page: RecoveryPage, id: string): RecoveryProposal | undefined {
  return ordered(page.proposals ?? []).pending.find((p) => p.kind === "unlock" && proposalInstallation(p) === id);
}

export function pendingBeginsFor(page: RecoveryPage, id: string): number[] {
  return ordered(page.proposals ?? []).pending.filter((p) => p.kind === "begin" && proposalInstallation(p) === id && p.number !== null).map((p) => p.number as number);
}

export type AlreadyProposed = { number: string; own: boolean } | null;

export function alreadyProposed(page: RecoveryPage, id: string): AlreadyProposed {
  const p = pendingUnlockFor(page, id);
  if (!p || p.number === null) return null;
  return { number: String(p.number), own: p.approve_blocked === "own_proposal" || p.can_withdraw };
}

export type Action = "approve" | "withdraw" | "lock" | "propose_unlock";
export type Explain = "hidden" | "explain";

export function blockerPolicy(code: string, action: Action, ctx: { pendingRow?: boolean; mode?: string } = {}): Explain {
  if (code === "not_pending") return ctx.pendingRow ? "explain" : "hidden";
  if (code === "not_proposer") return "hidden";
  if (code === "locked" && action === "lock") return "hidden";
  if (code === "not_recovering" && action === "propose_unlock" && ctx.mode === "none") return "hidden";
  if (code === "not_console_kind" && !ctx.pendingRow) return "hidden";
  return "explain";
}

export type Narrow = "" | "no_command" | "unreadable" | "resume_unknown" | "generation_unknown" | "changed";

export const resumeModes = ["lock_release", "recovery_resume"];

export function proposeNarrow(r: ResumeRow): Narrow {
  if (!installTarget(r.installation)) return "no_command";
  if (!digestPattern.test(r.lost_digest)) return "unreadable";
  if (!resumeModes.includes(r.mode)) return "unreadable";
  if (r.mode === "recovery_resume" && r.lost === null) return "unreadable";
  return "";
}

export function approveNarrow(page: RecoveryPage, p: RecoveryProposal): Narrow {
  if (!proposalTarget(p)) return "no_command";
  const id = proposalInstallation(p);
  const r = resumeOf(page, id);
  if (!r || !resumeModes.includes(r.mode)) return "resume_unknown";
  const g = generationOf(page, id);
  if (g === null) return "generation_unknown";
  if (p.summary.unlock_generation !== g || p.summary.lost_digest !== r.lost_digest) return "changed";
  return "";
}

export type Basis = {
  installation: string;
  mode: string;
  state: string;
  generation: string;
  lost: string;
  digest: string;
  recovery: string;
  begins: string;
};

export function basisOf(page: RecoveryPage, id: string): Basis {
  const r = resumeOf(page, id);
  const g = generationOf(page, id);
  const s = setOf(page, id);
  return {
    installation: id,
    mode: r?.mode ?? "",
    state: stateOf(page, id),
    generation: g === null ? "" : String(g),
    lost: r && r.lost !== null ? JSON.stringify(r.lost) : "",
    digest: r?.lost_digest ?? "",
    recovery: s && s.recovery.state === "open" ? s.recovery.id : "",
    begins: pendingBeginsFor(page, id).join(","),
  };
}

export function basisFrom(context: Record<string, string>): Basis {
  const g = (k: string) => context[k] ?? "";
  return { installation: g("installation"), mode: g("mode"), state: g("state"), generation: g("generation"), lost: g("lost"), digest: g("digest"), recovery: g("recovery"), begins: g("begins") };
}

export function basisContext(b: Basis, extra: Record<string, string> = {}): Record<string, string> {
  return { ...b, ...extra };
}

export function lostList(b: Basis): number[] | null {
  if (!b.lost) return null;
  try {
    const v = JSON.parse(b.lost) as unknown;
    return Array.isArray(v) && v.every((x) => typeof x === "number" && Number.isSafeInteger(x) && x >= 0) ? (v as number[]) : null;
  } catch {
    return null;
  }
}

export function generationNumber(b: Basis): number | null {
  return /^(0|[1-9][0-9]*)$/.test(b.generation) ? Number(b.generation) : null;
}

export function listed(page: RecoveryPage, command: CommandKey, target: string): boolean {
  if (isProposalCommand(command)) return (page.proposals ?? []).some((p) => p.target === target);
  if (command === "recovery.lock") return (page.isolation.rows ?? []).some((r) => installTarget(r.installation) === target);
  return (page.resume.rows ?? []).some((r) => installTarget(r.installation) === target);
}

export function stillAllowed(page: RecoveryPage, command: CommandKey, target: string): boolean {
  if (isProposalCommand(command)) {
    const p = (page.proposals ?? []).find((x) => x.target === target);
    if (!p || p.state !== "pending") return false;
    return command === "recovery.approve" ? p.can_approve && approveNarrow(page, p) === "" : p.can_withdraw;
  }
  if (command === "recovery.lock") return !!(page.isolation.rows ?? []).find((r) => installTarget(r.installation) === target)?.can_lock;
  const r = (page.resume.rows ?? []).find((x) => installTarget(x.installation) === target);
  return !!r && r.can_propose_unlock && proposeNarrow(r) === "";
}

export type Selection = { installation: string; proposal: string };

export function selectionOf(params: URLSearchParams, page: RecoveryPage | null, view: string): Selection {
  const out: Selection = { installation: "", proposal: "" };
  if (!page) return out;
  const id = params.get("installation") ?? "";
  if (validId(id)) {
    const section = view === "sets" ? page.sets.rows : view === "isolation" ? page.isolation.rows : view === "reconciliation" ? page.reconciliation.rows : view === "resume" ? page.resume.rows : null;
    if ((section ?? []).some((r) => r.installation === id)) out.installation = id;
  }
  const proposal = params.get("proposal") ?? "";
  if (view === "resume" && /^[1-9][0-9]*$/.test(proposal) && (page.proposals ?? []).some((p) => p.number !== null && String(p.number) === proposal)) out.proposal = proposal;
  return out;
}

export function rowId(prefix: string, id: string): string {
  return "p03-" + prefix + "-" + id;
}

export type Reaction = "field" | "reason" | "hint" | "conflict" | "denied" | "signin" | "wait" | "retry" | "renew" | "partial" | "reauth" | "failed";

export function react(failure: string, fields: string[]): Reaction {
  switch (failure) {
    case "invalid":
      return fields.length > 0 ? "field" : "reason";
    case "refused":
    case "stage":
      return "hint";
    case "conflict":
      return "conflict";
    case "denied":
      return "denied";
    case "unauthenticated":
      return "signin";
    case "limited":
      return "wait";
    case "busy":
    case "unavailable":
      return "retry";
    case "stale":
      return "renew";
    case "partial":
      return "partial";
    case "reauth":
    case "reauth_cancelled":
      return "reauth";
    default:
      return "failed";
  }
}

export function keepsAttempt(r: Reaction): boolean {
  return r === "retry" || r === "reauth";
}

export type FieldReaction = "built" | "changed" | "reason";

export function fieldReaction(command: CommandKey, fields: string[]): FieldReaction {
  const own = command === "recovery.propose_unlock" ? ["lost"] : [];
  if (fields.some((f) => f !== "rationale" && !own.includes(f))) return "built";
  if (fields.includes("lost")) return "changed";
  return "reason";
}

export function utf8Bytes(v: string): number {
  return new TextEncoder().encode(v).length;
}

export function reasonProblem(v: string): "" | "missing" | "long" {
  const t = v.trim();
  if (!t) return "missing";
  return utf8Bytes(t) > maxReason ? "long" : "";
}

export type Pending = { key: string; request: string; expected: string; values: string; snapshot: string[]; context: Record<string, string> };
const pending = new Map<string, Pending>();
const drafts = new Map<string, Record<string, string>>();
const refusals = new Refusals<boolean>();

let scope = "";
let generation = 0;

export function currentScope(): number {
  return generation;
}

export function setScope(workspace: string, user: string, session: string): void {
  const next = workspace + "\u0000" + user + "\u0000" + session;
  if (next !== scope) {
    generation += 1;
    pending.clear();
    drafts.clear();
    refusals.clear();
    scope = next;
  }
}

export function forget(): void {
  generation += 1;
  pending.clear();
  drafts.clear();
  refusals.clear();
  scope = "";
}

function slot(command: string, target: string): string {
  return scope + "\u0001" + command + "\u0000" + target;
}

function unslot(id: string): { command: CommandKey; target: string } | null {
  if (!id.startsWith(scope + "\u0001")) return null;
  const rest = id.slice(scope.length + 1);
  const at = rest.indexOf("\u0000");
  return { command: rest.slice(0, at) as CommandKey, target: rest.slice(at + 1) };
}

export function pendingNow(command: string, target: string): Pending | undefined {
  return pending.get(slot(command, target));
}

export function pendingFor(
  command: string,
  target: string,
  values: Record<string, string>,
  expected: string,
  fresh: () => { request: string; key: string },
  snapshot: string[] = [],
  context: Record<string, string> = {},
): Pending {
  const id = slot(command, target);
  const frozen = JSON.stringify(values);
  const kept = pending.get(id);
  if (kept && kept.values === frozen) return kept;
  refusals.delete(id);
  const tried = fresh();
  const next = { key: tried.key, request: tried.request, expected, values: frozen, snapshot: [...snapshot], context: { ...context } };
  pending.set(id, next);
  return next;
}

export function frozenValues(p: Pending | undefined): Record<string, string> | null {
  if (!p) return null;
  try {
    const v = JSON.parse(p.values) as unknown;
    return v && typeof v === "object" ? (v as Record<string, string>) : null;
  } catch {
    return null;
  }
}

export function discard(command: string, target: string, values: Record<string, string>): void {
  const kept = pending.get(slot(command, target));
  if (kept && kept.values !== JSON.stringify(values)) pending.delete(slot(command, target));
}

export function drop(command: string, target: string): void {
  pending.delete(slot(command, target));
}

export function settle(command: string, target: string, reaction: Reaction | "ok", detail?: RefusalDetail): void {
  if (reaction !== "ok" && keepsAttempt(reaction)) return;
  pending.delete(slot(command, target));
  if (reaction === "hint") refusals.set(slot(command, target), true, detail);
  else refusals.delete(slot(command, target));
}

export function refusedDetail(command: string, target: string): RefusalDetail | undefined {
  return refusals.detail(slot(command, target));
}

export function refusedNow(command: string, target: string): boolean {
  return refusals.has(slot(command, target));
}

export function pruneRefusals(page: RecoveryPage): void {
  for (const id of [...refusals]) {
    const at = unslot(id);
    if (!at) {
      refusals.delete(id);
      continue;
    }
    if (!listed(page, at.command, at.target) || !stillAllowed(page, at.command, at.target)) refusals.delete(id);
  }
}

export type Vanished = { command: CommandKey; target: string; context: Record<string, string> };

export function vanished(page: RecoveryPage, view: string): Vanished[] {
  const out: Vanished[] = [];
  for (const [id, p] of pending) {
    const at = unslot(id);
    if (!at || viewOfCommand(at.command) !== view || listed(page, at.command, at.target)) continue;
    out.push({ command: at.command, target: at.target, context: p.context });
  }
  return out;
}

export function draftOf(command: string, target: string): Record<string, string> {
  return drafts.get(slot(command, target)) ?? {};
}

export function keepDraft(command: string, target: string, values: Record<string, string>): void {
  drafts.set(slot(command, target), { ...draftOf(command, target), ...values });
}

export function clearDraft(command: string, target: string): void {
  drafts.delete(slot(command, target));
}

export function snapshotOf(page: RecoveryPage, command: CommandKey): string[] {
  if (command === "recovery.propose_unlock") return (page.proposals ?? []).flatMap((p) => (p.number === null ? [] : [String(p.number)]));
  return [];
}

export function duplicateIn(snapshot: string[], id: string): boolean {
  return !!id && snapshot.includes(id);
}

export type Denied = "applied_proposal" | "gone";

export function deniedOutcome(command: CommandKey, target: string, hadPending: boolean, page: RecoveryPage): Denied {
  if (hadPending && isProposalCommand(command) && !listed(page, command, target)) return "applied_proposal";
  return "gone";
}
