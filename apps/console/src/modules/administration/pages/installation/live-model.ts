import { Refusals } from "../../../../service/refusal";
import type { RowDiagnostic } from "../../../../service/command-diagnostics";
import type { InstallationPage, InstallProposal } from "../../../../service/installation";

export const views = ["plan", "diagnostics", "support", "access"];

export function viewOf(v: string): string {
  return views.includes(v) ? v : "plan";
}

export const hashPattern = /^[0-9a-f]{64}$/;
export const targetPattern = /^installation-proposal-[1-9][0-9]*$/;
export const planTarget = "installation";
export const shownLimit = 50;
export const maxReason = 256;

export type CommandKey = "installation.approve" | "installation.withdraw" | "installation.propose_plan";

export function commandTarget(p: InstallProposal): string {
  if (p.number === null || !targetPattern.test(p.target)) return "";
  return p.target === "installation-proposal-" + p.number ? p.target : "";
}

export function timeKnown(iso: string): boolean {
  if (!iso) return false;
  const d = new Date(iso);
  return !Number.isNaN(d.getTime()) && d.getUTCFullYear() > 1;
}

export const observationStates = ["fresh", "stale", "missing", "unknown"];

export function observationBlocks(state: string): boolean {
  return state === "stale" || state === "missing";
}

export function ordered(list: InstallProposal[]): { pending: InstallProposal[]; closed: InstallProposal[] } {
  const sorted = [...list].sort((a, b) => (b.number ?? -1) - (a.number ?? -1));
  return { pending: sorted.filter((p) => p.state === "pending"), closed: sorted.filter((p) => p.state !== "pending") };
}

export const proposalKinds = ["key", "release", "site", "probe", "config", "destination", "plan", "upgrade", "rollback"];

export type Summary = { kind: string; known: boolean; values: string[] };

export function summaryParts(p: InstallProposal): Summary {
  const s = p.summary;
  const known = proposalKinds.includes(p.kind);
  switch (p.kind) {
    case "key":
      return { kind: p.kind, known, values: [s.key_version] };
    case "release":
      return { kind: p.kind, known, values: [s.release_version] };
    case "site":
    case "probe":
    case "config":
      return { kind: p.kind, known, values: [s.site] };
    case "destination":
      return { kind: p.kind, known, values: [s.host, s.port === null ? "" : String(s.port), s.ranges.join(", ")] };
    case "plan":
    case "upgrade":
    case "rollback":
      return { kind: p.kind, known, values: [s.plan_version] };
    default:
      return { kind: p.kind, known: false, values: [] };
  }
}

export const acceptanceKinds = ["plan", "upgrade", "rollback"];

export function isAcceptance(p: InstallProposal): boolean {
  return acceptanceKinds.includes(p.kind);
}

export function hashChanged(p: InstallProposal): boolean {
  return p.current_hash !== "" && p.current_hash !== p.summary.plan_hash;
}

export const knownApproveBlockers = ["own_proposal", "not_pending", "plan_changed", "observation_stale", "not_observed", "conflicts"];

export function withdrawSentence(p: InstallProposal): boolean {
  return !p.can_withdraw && p.withdraw_blocked !== "not_proposer" && p.withdraw_blocked !== "not_pending";
}

export function actor(login: string, id: string): string {
  return login || id;
}

export type ProposeBlocker = "" | "unavailable" | "not_applicable" | "unknown_state" | "conflicts" | "conflicts_unknown" | "hash" | "accepted" | "waiting";

export type ProposeState = { offered: boolean; blocker: ProposeBlocker; stale: InstallProposal | null };

export function proposeState(page: InstallationPage, me: string): ProposeState {
  const p = page.preview;
  const none = (blocker: ProposeBlocker): ProposeState => ({ offered: false, blocker, stale: null });
  if (p.state === "unavailable") return none("unavailable");
  if (p.state === "not_applicable") return none("not_applicable");
  if (p.state !== "available" || !p.version) return none("unknown_state");
  if (p.conflicts === null) return none("conflicts_unknown");
  if (p.conflicts.length > 0) return none("conflicts");
  if (!hashPattern.test(p.hash)) return none("hash");
  const plans = page.proposals.filter((x) => x.kind === "plan" && x.summary.plan_version === p.version);
  if (plans.some((x) => x.state === "applied")) return none("accepted");
  const mine = plans.filter((x) => x.state === "pending" && !!me && x.proposer === me);
  if (mine.some((x) => x.summary.plan_hash === p.hash)) return none("waiting");
  return { offered: true, blocker: "", stale: mine[0] ?? null };
}

export function previewBasis(page: InstallationPage): "" | "pending" | "release" {
  const v = page.preview.version;
  if (!v) return "";
  const newest = ordered(page.proposals).pending.find((x) => x.kind === "plan");
  return newest && newest.summary.plan_version === v ? "pending" : "release";
}

export function latestAcceptance(list: InstallProposal[]): InstallProposal | null {
  return ordered(list).closed.find((x) => x.kind === "plan" && x.state === "applied") ?? null;
}

export function duplicateOf(before: InstallProposal[], number: string): boolean {
  return /^[1-9][0-9]*$/.test(number) && before.some((x) => x.number !== null && String(x.number) === number);
}

export function showsEffects(state: string): boolean {
  return state === "available";
}

export type NotApplicable = "no_release" | "nothing_installed" | "no_newer" | "no_earlier";

export function notApplicableOf(kind: "install" | "upgrade" | "rollback", installed: string): NotApplicable {
  if (kind === "install") return "no_release";
  if (installed === "not_installed") return "nothing_installed";
  return kind === "upgrade" ? "no_newer" : "no_earlier";
}

export function endpoint(host: string, port: string, unknown: string): string {
  return (host || unknown) + ":" + (port || unknown);
}

export function listCount(v: unknown[] | null): "unknown" | "none" | number {
  if (v === null) return "unknown";
  return v.length === 0 ? "none" : v.length;
}

export function reportPassed(failure: string): boolean {
  return failure === "ok";
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

export function planChangedField(fields: string[]): boolean {
  return fields.includes("version") || fields.includes("hash");
}

export function utf8Bytes(v: string): number {
  return new TextEncoder().encode(v).length;
}

export function reasonTooLong(v: string): boolean {
  return utf8Bytes(v) > maxReason;
}

export type Pending = { key: string; request: string; expected: string; values: string };
const pending = new Map<string, Pending>();
const drafts = new Map<string, Record<string, string>>();
const refusals = new Refusals<boolean>();

let scope = "";
let scopeGeneration = 0;

export function currentScope(): number { return scopeGeneration; }

export function setScope(workspace: string, user: string, session: string): void {
  const next = workspace + "\u0000" + user + "\u0000" + session;
  if (next !== scope) {
    scopeGeneration++;
    pending.clear();
    drafts.clear();
    refusals.clear();
    scope = next;
  }
}

export function forget(): void {
  scopeGeneration++;
  pending.clear();
  drafts.clear();
  refusals.clear();
  scope = "";
}

function slot(command: string, target: string): string {
  return scope + "\u0001" + command + "\u0000" + target;
}

export function pendingNow(command: string, target: string): Pending | undefined {
  return pending.get(slot(command, target));
}

export function pendingFor(command: string, target: string, values: Record<string, string>, expected: string, fresh: () => { request: string; key: string }): Pending {
  const id = slot(command, target);
  const frozen = JSON.stringify(values);
  const kept = pending.get(id);
  if (kept && kept.values === frozen) return kept;
  refusals.delete(id);
  const tried = fresh();
  const next = { key: tried.key, request: tried.request, expected, values: frozen };
  pending.set(id, next);
  return next;
}

export function discard(command: string, target: string, values: Record<string, string>): void {
  const kept = pending.get(slot(command, target));
  if (kept && kept.values !== JSON.stringify(values)) pending.delete(slot(command, target));
}

export function settle(command: string, target: string, reaction: Reaction | "ok", diagnostic?: RowDiagnostic): void {
  if (reaction !== "ok" && keepsAttempt(reaction)) return;
  pending.delete(slot(command, target));
  if (reaction === "hint") refusals.set(slot(command, target), true, diagnostic);
  else refusals.delete(slot(command, target));
}

export function refusedNow(command: string, target: string): boolean {
  return refusals.has(slot(command, target));
}

export function pruneRefusals(page: InstallationPage, me: string): void {
  const still = (command: CommandKey, target: string): boolean => {
    if (command === "installation.propose_plan") return proposeState(page, me).offered;
    const row = page.proposals.find((p) => p.target === target);
    if (!row || row.state !== "pending") return false;
    return command === "installation.approve" ? row.can_approve : row.can_withdraw;
  };
  for (const command of ["installation.approve", "installation.withdraw", "installation.propose_plan"] as CommandKey[]) {
    const targets = command === "installation.propose_plan" ? [planTarget] : page.proposals.map((p) => p.target);
    for (const target of targets) if (refusals.has(slot(command, target)) && !still(command, target)) refusals.delete(slot(command, target));
  }
  for (const id of [...refusals.keys()]) {
    const target = id.slice(id.indexOf("\u0000", id.indexOf("\u0001")) + 1);
    if (target !== planTarget && !page.proposals.some((p) => p.target === target)) refusals.delete(id);
  }
}

export function draftOf(command: string, target: string): Record<string, string> {
  return drafts.get(slot(command, target)) ?? {};
}

export function keepDraft(command: string, target: string, values: Record<string, string>): void {
  drafts.set(slot(command, target), values);
}

export function clearDraft(command: string, target: string): void {
  drafts.delete(slot(command, target));
}

export function refusalDiagnostics(target: string): RowDiagnostic[] {
  return ["installation.approve", "installation.withdraw", "installation.propose_plan"].flatMap((command) => {
    const entry = refusals.detail(slot(command, target));
    return entry && entry.command === command && entry.detail.target === target ? [entry] : [];
  });
}
