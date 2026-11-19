import { Refusals, type RefusalDetail } from "../../../../service/refusal";
import type { Budget, Candidate, CountOf, EvidencePage, PathRow, Problem, ProfileRow, Proposal, RequirementRow, TargetRow, TestRow } from "../../../../service/release-evidence";
import { failingVerdicts, has, verdictWords } from "./live-codes";

export const views = ["matrix", "requirements", "artifacts", "load", "decisions"];

export function viewOf(v: string): string {
  return views.includes(v) ? v : "matrix";
}

export const splitWidth = 940;
export const maxReason = 256;
export const maxText = 200;

export type Mode = "split" | "step";

export function modeOf(width: number): Mode {
  return Number.isFinite(width) && Math.floor(width) >= splitWidth ? "split" : "step";
}

export type CommandKey = "release_evidence.approve" | "release_evidence.withdraw" | "release_evidence.retire_requirement" | "release_evidence.retire_profile";
export const commandKeys: CommandKey[] = ["release_evidence.approve", "release_evidence.withdraw", "release_evidence.retire_requirement", "release_evidence.retire_profile"];

export function isProposalCommand(command: CommandKey): boolean {
  return command === "release_evidence.approve" || command === "release_evidence.withdraw";
}

const idChars = /^[A-Za-z0-9._:-]{1,128}$/;
export const tracePrefix = "trace-proposal-";
export const perfPrefix = "perf-proposal-";
export const requirementPrefix = "trace-requirement-";
export const profilePrefix = "perf-profile:";
const requirementPattern = /^trace-requirement-([1-9][0-9]*)$/;

export function utf8Bytes(v: string): number {
  return new TextEncoder().encode(v).length;
}

export function validTarget(t: string): boolean {
  return idChars.test(t) && utf8Bytes(t) <= 128;
}

export function modulePrefix(module: string): string {
  return module === "release_trace" ? tracePrefix : module === "perf_gate" ? perfPrefix : "";
}

export function proposalTarget(p: Proposal): string {
  const prefix = modulePrefix(p.module);
  if (!prefix || p.number === null || !Number.isSafeInteger(p.number)) return "";
  return p.target === prefix + p.number && validTarget(p.target) ? p.target : "";
}

export function requirementNumber(target: string): string {
  const m = requirementPattern.exec(target);
  return m && Number.isSafeInteger(Number(m[1])) ? m[1] : "";
}

export function requirementTarget(r: RequirementRow): string {
  return requirementNumber(r.target) && validTarget(r.target) ? r.target : "";
}

export function profileTarget(p: ProfileRow): string {
  if (!p.id) return "";
  return p.target === profilePrefix + p.id && validTarget(p.target) ? p.target : "";
}

export function replacementOptions(r: RequirementRow): TargetRow[] {
  return (r.replacements.rows ?? []).filter((x) => !!requirementNumber(x.target) && validTarget(x.target) && x.target !== r.target && !!x.label);
}

export function frozenTargetOk(command: CommandKey, target: string): boolean {
  if (!validTarget(target)) return false;
  switch (command) {
    case "release_evidence.approve":
    case "release_evidence.withdraw":
      return /^(trace|perf)-proposal-[1-9][0-9]*$/.test(target);
    case "release_evidence.retire_requirement":
      return !!requirementNumber(target);
    default:
      return target.startsWith(profilePrefix) && target.length > profilePrefix.length;
  }
}

export function commandsAllowed(page: EvidencePage): boolean {
  return page.revision !== "";
}

export function timeKnown(iso: string | null): iso is string {
  if (!iso) return false;
  const d = new Date(iso);
  return !Number.isNaN(d.getTime()) && d.getUTCFullYear() > 1;
}

export const digestPattern = /^[0-9a-f]{64}$/;
export const commitPattern = /^[0-9a-f]{40}([0-9a-f]{24})?$/;

export type Absent = "not_evaluated" | "unknown";

export function absentOf(candidate: Candidate): Absent {
  return candidate === null ? "not_evaluated" : "unknown";
}

export type Locator = "gate.ready" | "readiness.ready" | "path.state" | "path.build" | "path.blockers" | "requirement.state" | "run.verdict" | "gate.status" | "budget.state" | "budget.approval" | "customer_proof.count" | "gate.templates" | "proposal.state" | "release" | "activation" | "support_decision" | "intake" | "revocation" | "artifacts" | "ledger" | "origin" | "measurement" | "page.state" | "planning_targets" | "test.outcome";

export type Shown = { kind: "unexpected"; code: Locator } | { kind: "value"; code: string };

export function pathState(p: PathRow): Shown {
  if (p.state === "active") return { kind: "unexpected", code: "path.state" };
  if (p.build_set) return { kind: "unexpected", code: "path.build" };
  return { kind: "value", code: p.state };
}

export function requirementState(state: string | null): Shown | null {
  if (state === null) return null;
  return state === "customer_proven" ? { kind: "unexpected", code: "requirement.state" } : { kind: "value", code: state };
}

export function verdictShown(v: string): Shown {
  return v === "passed" ? { kind: "unexpected", code: "run.verdict" } : { kind: "value", code: v };
}

export function budgetUnexpected(b: Budget): Locator | "" {
  if (!b) return "";
  if (b.state === "approved") return "budget.state";
  return b.approval_set ? "budget.approval" : "";
}

export function proofUnexpected(c: CountOf): boolean {
  return !!c && c.count !== null && c.count > 0;
}

export function blockersUnexpected(b: Problem[] | null): boolean {
  return b !== null && b.length === 0;
}

export function problemUnexpected(code: string): boolean {
  return code === "path_build";
}

export function proposalUnexpected(p: Proposal): boolean {
  if (p.state !== "applied") return false;
  return (p.module === "release_trace" && (p.kind === "activate" || p.kind === "proof")) || (p.module === "perf_gate" && p.kind === "approval");
}

export function failingVerdict(v: string): boolean {
  return failingVerdicts.includes(v);
}

export type VerdictLine = { kind: "verdict"; verdict: string; failing: boolean } | { kind: "not_run" } | { kind: "not_evaluated" } | { kind: "unexpected" } | { kind: "unknown"; code: string };

export function verdictLine(status: string | null, candidate: Candidate): VerdictLine {
  if (status === null) return candidate === null ? { kind: "not_evaluated" } : { kind: "unknown", code: "" };
  if (status === "passed") return { kind: "unexpected" };
  if (status === "not_run") return { kind: "not_run" };
  if (has(verdictWords, status)) return { kind: "verdict", verdict: status, failing: failingVerdict(status) };
  return { kind: "unknown", code: status };
}

export function stateAtCandidate(r: RequirementRow, candidate: Candidate): string | null {
  if (!candidate || !r.states) return null;
  return has(r.states, candidate.build) ? r.states[candidate.build] : null;
}

export type TestCounts = { total: number | null; pass: number; fail: number; none: number };

export function testCounts(r: RequirementRow): TestCounts | null {
  if (!r.tests || r.tests.rows === null) return null;
  const rows = r.tests.rows;
  return {
    total: r.tests.total,
    pass: rows.filter((x) => x.latest_outcome === "pass").length,
    fail: rows.filter((x) => x.latest_outcome === "fail").length,
    none: rows.filter((x) => x.latest_outcome === null).length,
  };
}

export type ReasonLine = { kind: "none" } | { kind: "not_run"; code: string } | { kind: "not_counted"; code: string } | { kind: "unknown" };

export function reasonLine(x: TestRow): ReasonLine {
  if (x.latest_outcome === null) return x.not_run_reason ? { kind: "not_run", code: x.not_run_reason } : { kind: "unknown" };
  if (x.not_counted_reason) return { kind: "not_counted", code: x.not_counted_reason };
  return { kind: "none" };
}

export function textProblem(v: string): "" | "missing" | "long" | "control" {
  if (/[\u0000-\u001F\u007F-\u009F]/.test(v)) return "control";
  const t = v.trim();
  if (!t) return "missing";
  return utf8Bytes(t) > maxText ? "long" : "";
}

export function reasonProblem(v: string): "" | "missing" | "long" {
  const t = v.trim();
  if (!t) return "missing";
  return utf8Bytes(t) > maxReason ? "long" : "";
}

export const noReplacement = "none";

export function replacementProblem(choice: string, required: boolean, options: TargetRow[]): "" | "choose" | "required" {
  if (choice === noReplacement) return required ? "required" : "";
  if (!choice || !options.some((o) => o.target === choice)) return required ? "required" : "choose";
  return "";
}

export function retireValues(d: Record<string, string>): Record<string, string> {
  const out: Record<string, string> = { reason: (d.reason ?? "").trim() };
  if (d.replacement && d.replacement !== noReplacement) out.replacement = d.replacement;
  return out;
}

export type RetireGate = { usable: boolean; note: "" | "no_command" | "no_usable" };

export function requirementRetire(r: RequirementRow, canCommand: boolean, pending: boolean): RetireGate {
  const target = requirementTarget(r);
  if (!target) return { usable: false, note: "no_command" };
  if (pending && frozenTargetOk("release_evidence.retire_requirement", target)) return { usable: canCommand, note: "" };
  if (!r.can_retire) return { usable: false, note: "" };
  if (r.replacement_required === null) return { usable: false, note: "no_command" };
  if (r.replacement_required && replacementOptions(r).length === 0) return { usable: false, note: "no_usable" };
  return { usable: canCommand, note: "" };
}

export function profileRetire(p: ProfileRow, canCommand: boolean, pending: boolean): RetireGate {
  const target = profileTarget(p);
  if (!target) return { usable: false, note: "no_command" };
  if (pending && frozenTargetOk("release_evidence.retire_profile", target)) return { usable: canCommand, note: "" };
  if (!p.can_retire) return { usable: false, note: "" };
  return { usable: canCommand, note: "" };
}

export function approveUsable(p: Proposal): boolean {
  return p.can_approve && p.kind === "retire" && !!proposalTarget(p) && p.state === "pending";
}

export function withdrawUsable(p: Proposal): boolean {
  return p.can_withdraw && p.kind === "retire" && !!proposalTarget(p) && p.state === "pending";
}

export type Explain = "hidden" | "explain";

export function blockerPolicy(code: string, ctx: { pendingRow?: boolean; retiredRow?: boolean } = {}): Explain {
  if (code === "not_pending") return ctx.pendingRow ? "explain" : "hidden";
  if (code === "not_proposer" || code === "store_retired") return "hidden";
  if (code === "retired" && ctx.retiredRow) return "hidden";
  return "explain";
}

export function stateSentence(code: string, ctx: { pendingRow?: boolean; retiredRow?: boolean } = {}): string {
  if (code === "not_proposer" && ctx.pendingRow) return "not_proposer";
  if (code === "retired" && ctx.retiredRow) return "retired";
  return "";
}

export function proposalOf(page: EvidencePage, module: string, n: number | null): Proposal | undefined {
  if (n === null) return undefined;
  return (page.decisions.proposals.rows ?? []).find((p) => p.module === module && p.number === n);
}

export type AlreadyProposed = { number: string; own: boolean; link: string } | null;

export function pendingRetire(page: EvidencePage, module: "release_trace" | "perf_gate", n: number | null): AlreadyProposed {
  if (n === null) return null;
  const p = proposalOf(page, module, n);
  return { number: String(n), own: !!p && (p.approve_blocked === "own_proposal" || p.can_withdraw), link: (module === "release_trace" ? "trace-" : "perf-") + n };
}

export function requirementByTarget(page: EvidencePage, target: string | null): RequirementRow | undefined {
  if (!target) return undefined;
  return (page.requirements.requirements.rows ?? []).find((r) => r.target === target);
}

export function requirementByLabel(page: EvidencePage, label: string): RequirementRow | undefined {
  const found = (page.requirements.requirements.rows ?? []).filter((r) => r.label === label && !!requirementTarget(r));
  return found.length === 1 ? found[0] : undefined;
}

export function requirementByTest(page: EvidencePage, namespace: string, test: string): RequirementRow | undefined {
  const found = (page.requirements.requirements.rows ?? []).filter((r) => r.namespace === namespace && !!requirementTarget(r) && (r.tests?.rows ?? []).some((x) => x.id === test));
  return found.length === 1 ? found[0] : undefined;
}

export function profileById(page: EvidencePage, id: string): ProfileRow | undefined {
  return (page.load.profiles.rows ?? []).find((p) => p.id === id);
}

const semver = /^[0-9]+\.[0-9]+\.[0-9]+$/;

function compareBuilds(a: string, b: string): number {
  const x = a.split(".").map(Number);
  const y = b.split(".").map(Number);
  for (let i = 0; i < 3; i++) if (x[i] !== y[i]) return y[i] - x[i];
  return 0;
}

export function sortBuilds(keys: string[]): string[] {
  const valid = keys.filter((x) => semver.test(x)).sort(compareBuilds);
  return [...valid, ...keys.filter((x) => !semver.test(x)).sort()];
}

export function buildsOf(page: EvidencePage): string[] {
  const all = new Set<string>();
  for (const r of page.requirements.requirements.rows ?? []) for (const key of Object.keys(r.states ?? {})) if (semver.test(key)) all.add(key);
  return [...all].sort(compareBuilds);
}

export type Pair = { from: string; to: string };

export function pairOf(page: EvidencePage, from: string, to: string): Pair | null {
  const builds = buildsOf(page);
  if (builds.length < 2) return null;
  if (semver.test(from) && semver.test(to) && from !== to && builds.includes(from) && builds.includes(to)) return { from, to };
  return { from: builds[1], to: builds[0] };
}

export type Difference = { label: string; from: string | null; to: string | null };

export function differences(page: EvidencePage, pair: Pair): Difference[] {
  return (page.requirements.requirements.rows ?? []).flatMap((r) => {
    const s = r.states ?? {};
    const a = has(s, pair.from) ? s[pair.from] : null;
    const b = has(s, pair.to) ? s[pair.to] : null;
    return a === b ? [] : [{ label: r.label, from: a, to: b }];
  });
}

export function buildsMore(page: EvidencePage): boolean {
  return (page.requirements.requirements.rows ?? []).some((r) => r.builds_more !== false);
}

export type Selection = { path: string; requirement: string; from: string; to: string; build: string; profile: string; proposal: string };

export function selectionOf(params: URLSearchParams, page: EvidencePage | null, view: string): Selection {
  const out: Selection = { path: "", requirement: "", from: "", to: "", build: "", profile: "", proposal: "" };
  if (!page) return out;
  const path = params.get("path") ?? "";
  if (view === "matrix" && idChars.test(path) && (page.matrix.paths.rows ?? []).some((p) => p.id === path)) out.path = path;
  const req = params.get("requirement") ?? "";
  if (view === "requirements" && /^[1-9][0-9]*$/.test(req) && requirementByTarget(page, requirementPrefix + req)) out.requirement = req;
  const from = params.get("from") ?? "";
  const to = params.get("to") ?? "";
  if (view === "requirements" && semver.test(from) && semver.test(to) && from !== to && buildsOf(page).includes(from) && buildsOf(page).includes(to)) {
    out.from = from;
    out.to = to;
  }
  const build = params.get("build") ?? "";
  if (view === "artifacts" && idChars.test(build) && (page.artifacts.builds.rows ?? []).some((b) => b.build === build)) out.build = build;
  const profile = params.get("profile") ?? "";
  if (view === "load" && idChars.test(profile) && profileById(page, profile)) out.profile = profile;
  const m = /^(trace|perf)-([1-9][0-9]*)$/.exec(params.get("proposal") ?? "");
  if (view === "decisions" && m && Number.isSafeInteger(Number(m[2])) && proposalOf(page, m[1] === "trace" ? "release_trace" : "perf_gate", Number(m[2]))) out.proposal = m[0];
  return out;
}

export function proposalKey(p: Proposal): string {
  const prefix = p.module === "release_trace" ? "trace-" : p.module === "perf_gate" ? "perf-" : "";
  return prefix && p.number !== null ? prefix + p.number : "";
}

export function safeId(id: string): string {
  return id.replace(/[^A-Za-z0-9]/g, (c) => "_" + (c.codePointAt(0) ?? 0).toString(16) + "_");
}

export function rowId(prefix: string, id: string): string {
  return "p05-" + prefix + "-" + safeId(id);
}

export function ordered(list: Proposal[]): { pending: Proposal[]; closed: Proposal[] } {
  return { pending: list.filter((p) => p.state === "pending"), closed: list.filter((p) => p.state !== "pending") };
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

export function keepsAttempt(r: Reaction | "ok"): boolean {
  return r === "retry" || r === "wait" || r === "reauth";
}

export type FieldReaction = "built" | "text" | "replacement" | "rationale";

const ownFields: Record<CommandKey, string[]> = {
  "release_evidence.approve": [],
  "release_evidence.withdraw": [],
  "release_evidence.retire_requirement": ["reason", "replacement"],
  "release_evidence.retire_profile": [],
};

export function fieldReaction(command: CommandKey, fields: string[]): FieldReaction {
  const own = ownFields[command];
  if (fields.some((f) => f !== "rationale" && !own.includes(f))) return "built";
  if (fields.includes("replacement")) return "replacement";
  if (fields.includes("reason")) return "text";
  return "rationale";
}

export type Pending = { key: string; request: string; expected: string; values: string; snapshot: string[]; context: Record<string, string> };
const pending = new Map<string, Pending>();
const drafts = new Map<string, Record<string, string>>();
const refusals = new Refusals<string>();
const earlier = new Set<string>();
let scope = "";
let generation = 0;

export function currentScope(): number {
  return generation;
}
let seen = false;
let revoked = false;

function clearAll(): void {
  generation += 1;
  pending.clear();
  drafts.clear();
  refusals.clear();
  earlier.clear();
}

export function setScope(workspace: string, user: string, session: string): void {
  const next = workspace + "\u0000" + user + "\u0000" + session;
  if (next !== scope) {
    clearAll();
    seen = false;
    revoked = false;
    scope = next;
  }
}

export function forget(): void {
  clearAll();
  seen = false;
  revoked = false;
  scope = "";
}

export function readSucceeded(): void {
  seen = true;
  revoked = false;
}

export function readDenied(): "revoked" | "denied" {
  if (seen) revoked = true;
  if (revoked) {
    clearAll();
    seen = false;
    return "revoked";
  }
  return "denied";
}

export function revokedNow(): boolean {
  return revoked;
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

export function hadEarlier(command: string, target: string): boolean {
  return pending.has(slot(command, target)) || earlier.has(slot(command, target));
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
  const id = slot(command, target);
  const kept = pending.get(id);
  if (kept && kept.values !== JSON.stringify(values)) {
    pending.delete(id);
    earlier.add(id);
  }
}

export function drop(command: string, target: string): void {
  pending.delete(slot(command, target));
}

export function renew(command: string, target: string): void {
  const id = slot(command, target);
  pending.delete(id);
  earlier.add(id);
}

export function cancelled(command: string, target: string): void {
  earlier.delete(slot(command, target));
}

export function settle(command: string, target: string, reaction: Reaction | "ok", failure = "", had = false, detail?: RefusalDetail): void {
  const id = slot(command, target);
  if (reaction === "ok") {
    pending.delete(id);
    earlier.delete(id);
    refusals.delete(id);
    return;
  }
  if (keepsAttempt(reaction)) return;
  pending.delete(id);
  if (had) earlier.add(id);
  if (reaction === "hint") refusals.set(id, had ? "earlier" : failure === "refused" ? "refused" : "stage", detail);
  else refusals.delete(id);
}

export function refusedDetail(command: string, target: string): RefusalDetail | undefined {
  return refusals.detail(slot(command, target));
}

export function refusedNow(command: string, target: string): string {
  return refusals.get(slot(command, target)) ?? "";
}

export function keptKey(command: CommandKey, kind: string): string {
  return kind === "earlier" ? "earlier" : command + ":" + (kind === "refused" ? "refused" : "stage");
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

export function listed(page: EvidencePage, command: CommandKey, target: string): boolean {
  switch (command) {
    case "release_evidence.approve":
    case "release_evidence.withdraw":
      return (page.decisions.proposals.rows ?? []).some((p) => p.target === target);
    case "release_evidence.retire_requirement":
      return (page.requirements.requirements.rows ?? []).some((r) => r.target === target);
    default:
      return (page.load.profiles.rows ?? []).some((p) => p.target === target);
  }
}

export function stillAllowed(page: EvidencePage, command: CommandKey, target: string): boolean {
  const canCommand = commandsAllowed(page);
  switch (command) {
    case "release_evidence.approve":
    case "release_evidence.withdraw": {
      const p = (page.decisions.proposals.rows ?? []).find((x) => x.target === target);
      if (!p) return false;
      return command === "release_evidence.approve" ? approveUsable(p) : withdrawUsable(p);
    }
    case "release_evidence.retire_requirement": {
      const r = (page.requirements.requirements.rows ?? []).find((x) => x.target === target);
      return !!r && requirementRetire(r, canCommand, false).usable;
    }
    default: {
      const p = (page.load.profiles.rows ?? []).find((x) => x.target === target);
      return !!p && profileRetire(p, canCommand, false).usable;
    }
  }
}

export function closedProposal(page: EvidencePage, command: CommandKey, target: string): boolean {
  if (!isProposalCommand(command)) return false;
  const p = (page.decisions.proposals.rows ?? []).find((x) => x.target === target);
  return !!p && p.state !== "pending";
}

export function pruneRefusals(page: EvidencePage): void {
  for (const id of [...refusals.keys()]) {
    const at = unslot(id);
    if (!at || !listed(page, at.command, at.target)) {
      refusals.delete(id);
      continue;
    }
    if (refusals.get(id) === "earlier" || closedProposal(page, at.command, at.target)) continue;
    if (!stillAllowed(page, at.command, at.target)) refusals.delete(id);
  }
  for (const id of [...earlier]) {
    const at = unslot(id);
    if (!at || !listed(page, at.command, at.target) || !stillAllowed(page, at.command, at.target)) earlier.delete(id);
  }
}

export type Vanished = { command: CommandKey; target: string; context: Record<string, string> };

export function vanished(page: EvidencePage, view: string): Vanished[] {
  const out: Vanished[] = [];
  for (const [id, p] of pending) {
    const at = unslot(id);
    if (!at || (p.context.view ?? "") !== view || listed(page, at.command, at.target)) continue;
    out.push({ command: at.command, target: at.target, context: p.context });
  }
  return out;
}

export function snapshotOf(page: EvidencePage, command: CommandKey): string[] {
  const module = command === "release_evidence.retire_requirement" ? "release_trace" : command === "release_evidence.retire_profile" ? "perf_gate" : "";
  if (!module) return [];
  return (page.decisions.proposals.rows ?? []).flatMap((p) => (p.module === module && p.number !== null ? [String(p.number)] : []));
}

export function duplicateIn(snapshot: string[], id: string): boolean {
  return !!id && snapshot.includes(id);
}

export type Denied = "applied_proposal" | "gone";

export function deniedOutcome(command: CommandKey, target: string, hadPending: boolean, page: EvidencePage): Denied {
  if (hadPending && isProposalCommand(command) && !listed(page, command, target)) return "applied_proposal";
  return "gone";
}

export type GoneKind = "proposal" | "requirement" | "profile";

export function goneKind(command: CommandKey): GoneKind {
  if (isProposalCommand(command)) return "proposal";
  return command === "release_evidence.retire_requirement" ? "requirement" : "profile";
}

export function retiredAfter(page: EvidencePage | null, p: Proposal): boolean {
  if (!page) return false;
  if (p.module === "release_trace") {
    const label = typeof p.summary.label === "string" ? p.summary.label : "";
    const r = label ? requirementByLabel(page, label) : undefined;
    return r?.retired === true;
  }
  const id = typeof p.summary.profile === "string" ? p.summary.profile : "";
  return id ? profileById(page, id)?.retired === true : false;
}
