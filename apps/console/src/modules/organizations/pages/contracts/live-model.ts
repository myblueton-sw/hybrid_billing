import type { ContractAction, ContractActionKey, ContractDetail, ContractEvent, ContractEvidence, ContractRow } from "../../../../service/contracts";

export type Filters = { state: string; settlement: string; path: string; buyer: string };

export const states = ["not_started", "active", "suspended", "ended"];
export const settlements = ["none", "pending_final", "settled"];
export const paths = ["external_invoice", "internal_chargeback"];

export function filtered(rows: ContractRow[], f: Filters): ContractRow[] {
  return rows.filter(
    (r) => (!f.state || r.state === f.state) && (!f.settlement || r.settlement === f.settlement) && (!f.path || r.path === f.path) && (!f.buyer || r.buyer === f.buyer),
  );
}

export function activeFilters(f: Filters): number {
  return [f.state, f.settlement, f.path, f.buyer].filter(Boolean).length;
}

export function filtersOf(params: URLSearchParams): Filters {
  const pick = (key: string, allowed: string[]) => {
    const v = params.get(key) ?? "";
    return allowed.includes(v) ? v : "";
  };
  const buyer = params.get("buyer") ?? "";
  return { state: pick("state", states), settlement: pick("settlement", settlements), path: pick("path", paths), buyer: idPattern.test(buyer) ? buyer : "" };
}

const dayPattern = /^\d{4}-\d{2}-\d{2}$/;

export function validDay(v: string): boolean {
  if (!dayPattern.test(v)) return false;
  const d = new Date(v + "T00:00:00Z");
  return !Number.isNaN(d.getTime()) && d.toISOString().slice(0, 10) === v;
}

export function addDays(day: string, n: number): string {
  const d = new Date(day + "T00:00:00Z");
  d.setUTCDate(d.getUTCDate() + n);
  return d.toISOString().slice(0, 10);
}

export function spanDays(from: string, until: string): number {
  return Math.round((Date.parse(until + "T00:00:00Z") - Date.parse(from + "T00:00:00Z")) / 86400000);
}

export const maxPeriod = 366;

export type Period = { from: string; until: string; problem: "" | "from" | "until" | "order" | "span" };

export function periodOf(params: URLSearchParams): Period {
  const from = params.get("from") ?? "";
  const until = params.get("until") ?? "";
  if (!from && !until) return { from: "", until: "", problem: "" };
  if (from && !validDay(from)) return { from, until, problem: "from" };
  if (until && !validDay(until)) return { from, until, problem: "until" };
  if (from && until && spanDays(from, until) <= 0) return { from, until, problem: "order" };
  if (from && until && spanDays(from, until) > maxPeriod) return { from, until, problem: "span" };
  return { from, until, problem: "" };
}

export const lifecycles: string[] = ["contract.suspend", "contract.resume", "contract.renew", "contract.end"];

export function offered(action: ContractAction | undefined, lifecycle = false): boolean {
  if (!action || !action.allowed || action.blocked !== "") return false;
  return !lifecycle || validDay(action.earliest);
}

export const knownBlockers = [
  "settled",
  "ended",
  "not_active",
  "not_suspended",
  "horizon",
  "write_limit",
  "nothing_withdrawable",
  "not_withdrawable",
  "breaks_lifecycle",
  "not_pending_final",
  "end_not_reached",
  "open_gaps",
  "evidence_needed",
  "not_settled",
];

export function withdrawable(e: ContractEvent): boolean {
  return e.withdraw_blocked === "";
}

export function openGaps(evidence: ContractEvidence[]): ContractEvidence[] {
  return evidence.filter((e) => e.kind === "missing" && !e.resolved);
}

export function minDay(today: string, earliest: string): string {
  if (validDay(earliest) && validDay(today)) return earliest > today ? earliest : today;
  return validDay(earliest) ? earliest : today;
}

export function dayProblem(value: string, min: string, max: string): "" | "format" | "min" | "max" {
  if (!validDay(value)) return "format";
  if (min && value < min) return "min";
  if (max && value > max) return "max";
  return "";
}

export const idPattern = /^[A-Za-z0-9._:-]{1,128}$/;

export function idProblem(value: string): boolean {
  return !idPattern.test(value);
}

export const gapReasons = ["source_unavailable", "access_withdrawn_emergency", "provider_refused", "not_attempted", "late_source_received"];
export const reopenReasons = ["late_source", "late_credit", "correction"];
export const cancelReasons = ["recorded_in_error", "contract_continues"];

export type Reaction = "field" | "reason" | "hint" | "conflict" | "denied" | "signin" | "wait" | "retry" | "renew" | "partial" | "failed";

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
    default:
      return "failed";
  }
}

export function keepsAttempt(r: Reaction): boolean {
  return r === "retry";
}

export function actionOf(detail: ContractDetail | null, key: ContractActionKey): ContractAction {
  const a = detail?.actions[key];
  if (!a) return { allowed: false, blocked: "missing", earliest: "" };
  if (a.allowed && a.blocked !== "") return { allowed: false, blocked: a.blocked, earliest: "" };
  return a;
}

export function utf8Bytes(v: string): number {
  return new TextEncoder().encode(v).length;
}

export function reasonTooLong(v: string): boolean {
  return utf8Bytes(v) > maxReason;
}

export function revisionText(v: number | null): "na" | "unknown" | string {
  if (v === null) return "unknown";
  return v === 0 ? "na" : String(v);
}

export function actorText(login: string, actor: string): string {
  return login || actor;
}

export const horizonDays = 3660;

export function lifecycleBounds(today: string, earliest: string): { min: string; max: string } {
  return { min: minDay(today, earliest), max: validDay(today) ? addDays(today, horizonDays) : "" };
}

export function settleBounds(today: string, end: string): { min: string; max: string } {
  return { min: validDay(end) ? end : "", max: today };
}

export function coversEnd(covered: string, end: string): boolean {
  return !validDay(end) || (validDay(covered) && covered >= end);
}

export function legalEntities(rows: ContractRow[]): string[] {
  return [...new Set(rows.map((r) => r.buyer).filter(Boolean))].sort();
}

export type Pending = { key: string; request: string; expected: string; values: string };
const pending = new Map<string, Pending>();
const drafts = new Map<string, Record<string, string>>();

let scope = "";

export function setScope(workspace: string, user: string, session: string): void {
  const next = workspace + "\u0000" + user + "\u0000" + session;
  if (next !== scope) {
    pending.clear();
    drafts.clear();
    scope = next;
  }
}

export function forget(): void {
  pending.clear();
  drafts.clear();
  scope = "";
}

export function slot(command: string, contract: string): string {
  return scope + "\u0001" + command + "\u0000" + contract;
}

export function discard(command: string, contract: string, values: Record<string, string>): void {
  const kept = pending.get(slot(command, contract));
  if (kept && kept.values !== JSON.stringify(values)) pending.delete(slot(command, contract));
}

export const maxResolves = 8;

export type FormKey =
  | "contract.suspend"
  | "contract.resume"
  | "contract.renew"
  | "contract.end"
  | "contract.withdraw"
  | "contract.evidence.secured"
  | "contract.evidence.missing"
  | "contract.settle"
  | "contract.reopen";

export function commandFor(key: FormKey): string {
  return key.startsWith("contract.evidence") ? "contract.evidence" : key;
}

export function bodyFor(key: FormKey, v: Record<string, string>, event?: ContractEvent): Record<string, string> {
  if (lifecycles.includes(key)) return { from: v.from ?? "" };
  if (key === "contract.withdraw") return event?.cancel_required && v.cancel ? { event: event.id, cancel: v.cancel } : { event: event?.id ?? "" };
  if (key === "contract.evidence.secured") {
    const out: Record<string, string> = { kind: "secured", reference: v.reference ?? "", covered_until: v.covered_until ?? "" };
    if (v.resolves) out.resolves = v.resolves;
    return out;
  }
  if (key === "contract.evidence.missing") return { kind: "missing", reason: v.reason ?? "", follow_up: v.follow_up ?? "" };
  if (key === "contract.settle") return { as_of: v.as_of ?? "" };
  return { reason: v.reason ?? "" };
}

export const formFields = ["from", "cancel", "reference", "covered_until", "resolves", "reason", "follow_up", "as_of"];

export function controlFor(server: string): string {
  return formFields.includes(server) ? server : "rationale";
}

export function canWithdraw(detail: ContractDetail, e: ContractEvent): boolean {
  return e.withdraw_blocked === "" && offered(actionOf(detail, "contract.withdraw"));
}

export function endPassed(end: string, today: string): boolean {
  return validDay(end) && validDay(today) && end <= today;
}

export function pendingNow(command: string, contract: string): Pending | undefined {
  return pending.get(slot(command, contract));
}

export function pendingFor(command: string, contract: string, values: Record<string, string>, expected: string, fresh: () => { request: string; key: string }): Pending {
  const id = slot(command, contract);
  const frozen = JSON.stringify(values);
  const kept = pending.get(id);
  if (kept && kept.values === frozen) return kept;
  const tried = fresh();
  const next = { key: tried.key, request: tried.request, expected, values: frozen };
  pending.set(id, next);
  return next;
}

export function settle(command: string, contract: string, reaction: Reaction | "ok"): void {
  if (reaction === "ok" || !keepsAttempt(reaction as Reaction)) pending.delete(slot(command, contract));
}

export function draftOf(command: string, contract: string): Record<string, string> {
  return drafts.get(slot(command, contract)) ?? {};
}

export function keepDraft(command: string, contract: string, values: Record<string, string>): void {
  drafts.set(slot(command, contract), values);
}

export function clearDraft(command: string, contract: string): void {
  drafts.delete(slot(command, contract));
}

export function count(v: number | null): string {
  return v === null ? "" : String(v);
}

export const maxReason = 256;
