import type { UsersFilter } from "../../../../service/admin";
import type { BulkJob, BulkPage, MatchCount } from "../../../../service/identity-bulk";
import { noFilters, sameFilters, type UserFilters } from "./filter-model";

export type Fixed = { draft: string; matched: number; fixed_at: string; fixed_revision: string; filter: UsersFilter; counted: number | null; countedAtLeast?: boolean };

export type CountView = "exact" | "atLeast" | "over" | "none";

export function countView(c: MatchCount): CountView {
  if (c.over_limit) return "over";
  if (c.complete && c.count === 0) return "none";
  return c.complete ? "exact" : "atLeast";
}

export function confirmable(c: MatchCount): boolean {
  const v = countView(c);
  return v === "exact" || v === "atLeast";
}

export function storedFilters(f: UsersFilter): UserFilters {
  return { ...noFilters, q: f.q, st: f.status, ms: f.membership, role: f.role, per: f.period, si: f.signin, adm: f.holder === "true" };
}

export function fixedOfJob(j: BulkJob, self: string, counted: number | null = null): Fixed | null {
  if (j.state !== "draft" || j.selection !== "filter" || j.preparer !== self || !j.filter) return null;
  return { draft: j.id, matched: j.matched, fixed_at: j.fixed_at, fixed_revision: j.fixed_revision, filter: j.filter, counted };
}

export function fixedOfFields(fields: Record<string, string>, counted: number | null, countedAtLeast = false): Fixed {
  const filter: UsersFilter = { q: "", status: "", membership: "", role: "", period: "", signin: "", invitation: "", holder: "" };
  for (const key of Object.keys(filter) as (keyof UsersFilter)[]) filter[key] = fields["filter." + key] ?? "";
  return { draft: fields.draft ?? "", matched: Number(fields.matched ?? 0), fixed_at: fields.fixed_at ?? "", fixed_revision: fields.fixed_revision ?? "", filter, counted, ...(countedAtLeast ? { countedAtLeast } : {}) };
}

export type CountNote = "none" | "differs" | "below";

export function countNote(f: Fixed): CountNote {
  if (f.counted === null) return "none";
  if (f.countedAtLeast) return f.matched < f.counted ? "below" : "none";
  return f.counted !== f.matched ? "differs" : "none";
}

export function differsFromCount(f: Fixed): boolean {
  return countNote(f) !== "none";
}

export function keepsOwn(f: Fixed, current: UserFilters): boolean {
  return !sameFilters(storedFilters(f.filter), current);
}

export type RefusalCause = "openDraft" | "none" | "over" | "pending" | "unknown";

export function refusalCause(page: BulkPage | null, pendingLimit: number, count: MatchCount | null): RefusalCause {
  if (page?.open_draft) return "openDraft";
  if (count && count.complete && count.count === 0) return "none";
  if (count?.over_limit) return "over";
  if (page && page.pending >= pendingLimit) return "pending";
  return "unknown";
}

export function emptyFilter(f: UsersFilter): boolean {
  return Object.values(f).every((v) => v === "");
}
