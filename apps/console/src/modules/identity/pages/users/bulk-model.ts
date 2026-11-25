import type { UserRow } from "../../../../service/admin";
import type { BulkCounts, BulkItem, BulkJob, Cleanup } from "../../../../service/identity-bulk";

export type Selection = ReadonlySet<string>;

export function toggle(s: Selection, id: string, limit: number): { next: Selection; refused: boolean } {
  const next = new Set(s);
  if (next.has(id)) {
    next.delete(id);
    return { next, refused: false };
  }
  if (next.size >= limit) return { next: s, refused: true };
  next.add(id);
  return { next, refused: false };
}

export type HeaderState = "checked" | "mixed" | "none";

export function headerState(s: Selection, loaded: string[]): HeaderState {
  if (loaded.length === 0) return "none";
  const n = loaded.filter((id) => s.has(id)).length;
  return n === 0 ? "none" : n === loaded.length ? "checked" : "mixed";
}

export function selectLoaded(s: Selection, loaded: string[], limit: number): { next: Selection; refused: boolean } {
  if (headerState(s, loaded) === "checked") {
    const next = new Set(s);
    for (const id of loaded) next.delete(id);
    return { next, refused: false };
  }
  const next = new Set(s);
  for (const id of loaded) next.add(id);
  if (next.size > limit) return { next: s, refused: true };
  return { next, refused: false };
}

export function notLoaded(s: Selection, loaded: string[]): string[] {
  const here = new Set(loaded);
  return [...s].filter((id) => !here.has(id)).sort();
}

export function without(s: Selection, ids: string[]): Selection {
  const next = new Set(s);
  for (const id of ids) next.delete(id);
  return next;
}

export function invariant(c: BulkCounts): boolean {
  return c.targets === c.succeeded + c.held + c.failed + c.unprocessed;
}

export type JobWords = "draft" | "prepared" | "running" | "unconfirmed" | "not_started" | "waiting" | "finished" | "partial" | "discarded" | "expired";

export function jobWords(j: BulkJob): JobWords {
  switch (j.state) {
    case "draft":
    case "prepared":
    case "discarded":
    case "expired":
      return j.state;
  }
  if (j.job_state === "waiting") return j.job_reason === "decision_unconfirmed" ? "unconfirmed" : j.job_reason === "cleanup_not_started" ? "not_started" : "waiting";
  if (j.job_state === "succeeded") return j.counts.failed > 0 ? "partial" : "finished";
  return "running";
}

export type ItemWords = "complete" | "held" | "failed" | "not_processed" | "not_recorded" | "excluded" | "unconfirmed";

export function itemWords(i: BulkItem): ItemWords {
  switch (i.outcome) {
    case "succeeded":
      return "complete";
    case "held":
      return "held";
    case "failed":
      return "failed";
    case "excluded":
      return "excluded";
  }
  if (i.state === "decision_unconfirmed") return i.receipt.reason === "record_not_confirmed" ? "not_recorded" : "unconfirmed";
  return "not_processed";
}

export function groups(cleanup: Cleanup[]): { done: Cleanup[]; not_applicable: Cleanup[]; outside_scope: Cleanup[] } {
  return {
    done: cleanup.filter((c) => c.state === "done"),
    not_applicable: cleanup.filter((c) => c.state === "not_applicable"),
    outside_scope: cleanup.filter((c) => c.state === "outside_scope"),
  };
}

export function split(jobs: BulkJob[]): { open: BulkJob[]; executed: BulkJob[]; ended: BulkJob[] } {
  return {
    open: jobs.filter((j) => j.state === "draft" || j.state === "prepared"),
    executed: jobs.filter((j) => j.state === "executed"),
    ended: jobs.filter((j) => j.state === "discarded" || j.state === "expired"),
  };
}

export function expired(j: BulkJob, serverTime: string): boolean {
  return j.state === "expired" || (j.state === "prepared" && new Date(serverTime).getTime() >= new Date(j.expires_at).getTime());
}

export const digestShown = (digest: string) => digest.slice(0, 12);

export function number(id: string): string {
  const m = /^bulk-(?:revocation|draft)-(\d+)$/.exec(id);
  return m ? m[1] : id;
}

export function mayBeAdministrator(u: UserRow | undefined): boolean {
  return u?.membership?.role === "administrator" || u?.membership?.role === "unknown";
}
