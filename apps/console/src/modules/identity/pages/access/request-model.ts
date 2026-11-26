import type { Locale } from "../../../../shared/types";
import { AdminError, type AccessPage, type AccessRequest, type AccessSetting, type CleanupItem, type LedgerStatus, type Revocation } from "../../../../service/admin";
import { q, s, v } from "./request-copy";

export function additionsPaused(ledger: LedgerStatus): boolean {
  return ledger.state !== "ok" && ledger.state !== "recovered";
}

export type Groups = Record<CleanupItem["state"], CleanupItem[]>;

export function cleanupGroups(items: CleanupItem[]): Groups {
  const out: Groups = { done: [], not_applicable: [], outside_scope: [], not_started: [] };
  for (const it of items) (out[it.state] ?? out.not_started).push(it);
  return out;
}

export function settled(items: CleanupItem[]): number {
  return items.filter((i) => i.state === "done" || i.state === "not_applicable").length;
}

export function shortDigest(digest: string): string {
  return digest.slice(0, 12);
}

export function stateWords(l: Locale, r: Revocation): { name: string; copy: string } {
  const key = r.state in v.name ? r.state : "decision_unconfirmed";
  return { name: v.name[key](l), copy: v.copy[key](l) };
}

export function jobWords(l: Locale, r: Revocation): string {
  return (v.job[r.job_state] ?? v.job.running)(l);
}

export function failureWords(l: Locale, reason: string): string {
  const found = (v.reason as Record<string, (l: Locale) => string>)[reason];
  return found ? found(l) : reason;
}

export function itemWords(l: Locale, it: CleanupItem): { name: string; why: string } {
  const name = (v.item as Record<string, (l: Locale) => string>)[it.item];
  const why = (v.why as Record<string, (l: Locale) => string>)[it.reason];
  return { name: name ? name(l) : it.item, why: why ? why(l) : it.reason };
}

export function requestState(l: Locale, r: AccessRequest): string {
  return (q.st[r.state] ?? q.st.pending)(l);
}

export function settingName(l: Locale, setting: AccessSetting): string {
  const found = (s.name as Record<string, (l: Locale) => string>)[setting.name];
  return found ? found(l) : setting.name;
}

export function settingValue(l: Locale, setting: AccessSetting): string {
  const found = (s.unit as Record<string, (l: Locale, n: number) => string>)[setting.unit];
  return found ? found(l, setting.value) : `${setting.value} ${setting.unit}`;
}

export function settingSource(l: Locale, setting: AccessSetting): string {
  return setting.source === "start_flag" ? s.source.start_flag(l) : s.source.default(l);
}

export function ledgerLines(l: Locale, ledger: LedgerStatus, when: (at: string) => string): string[] {
  const out: string[] = [];
  switch (ledger.state) {
    case "isolated": {
      const reason = ledger.isolation_reason;
      out.push(reason === "missing" ? v.ledgerIsolated.missing(l) : reason === "id_mismatch" ? v.ledgerIsolated.id_mismatch(l) : v.ledgerIsolated.integrity_failed(l, ledger.problem_line));
      break;
    }
    case "stopped":
      out.push(ledger.not_durable ? v.ledgerStoppedStartSync(l) : v.ledgerStopped(l));
      break;
    case "full":
      out.push(v.ledgerFull(l));
      break;
    case "unknown":
      out.push(v.ledgerUnknown(l));
      break;
    default:
      out.push(v.ledgerOk(l, ledger.records, ledger.watermark.split(":")[0] || "0"));
  }
  if (ledger.not_durable) out.push(ledger.state === "isolated" ? v.ledgerNotDurable.isolated(l) : v.ledgerNotDurable.stopped(l));
  if (ledger.recovery) out.push(v.ledgerRecovered(l, when(ledger.recovery.at), ledger.recovery.truncated_bytes));
  if (ledger.replaces) out.push(v.ledgerReplaced(l, ledger.replaces === "unknown" ? v.unknownLedger(l) : ledger.replaces));
  if (additionsPaused(ledger)) out.push(v.stillAvailable(l));
  return out;
}

export function keepAttempt(e: unknown): boolean {
  return e instanceof AdminError && e.failure === "partial";
}
