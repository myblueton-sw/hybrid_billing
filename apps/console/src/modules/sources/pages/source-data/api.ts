import { request, ServiceError } from "../live-imports/api";
import type { Candidate, Candidates, Profile } from "../reconciliation/api";
export { ServiceError };
export type { Candidate, Candidates, Profile };
export const uuid =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;
export const digest = /^[0-9a-f]{64}$/;
export const ordinal = /^(0|[1-9][0-9]{0,3})$/;
export const rowOrdinal = /^(?:[1-9][0-9]{0,3}|10000)$/;
export type Row = { ordinal: string; values: Record<string, string | null> };
export type Page = {
  candidate: Candidate;
  snapshot: string;
  after: string;
  next_after: string | null;
  rows: Row[];
  fields: string[];
  observed_at: string;
  classification: string;
  content_integrity: string;
};
const fields = new Set([
  "billing_account_id",
  "sub_account_id",
  "service_name",
  "resource_id",
  "charge_period_start",
  "charge_period_end",
  "consumed_quantity",
  "consumed_unit",
  "pricing_quantity",
  "pricing_unit",
  "billed_cost",
  "effective_cost",
  "list_cost",
  "contracted_cost",
  "billing_currency",
  "charge_category",
]);
const decimals = new Set([
  "consumed_quantity",
  "pricing_quantity",
  "billed_cost",
  "effective_cost",
  "list_cost",
  "contracted_cost",
]);
export function validPage(
  p: Page,
  release: string,
  snapshot: string,
  after: string,
): boolean {
  return (
    !!p &&
    validCandidate(p.candidate) &&
    bounded(p.observed_at, 64) &&
    p.candidate?.release_id === release &&
    p.candidate.manifest_digest === snapshot &&
    p.snapshot === snapshot &&
    p.after === after &&
    p.classification === "local-source-inspection" &&
    p.content_integrity === "not-revalidated" &&
    Array.isArray(p.fields) &&
    [
      "billing_account_id",
      "billing_currency",
      "billed_cost",
      "effective_cost",
      "charge_period_start",
      "charge_period_end",
    ].every((f) => p.fields.includes(f)) &&
    p.fields.every((f) => fields.has(f)) &&
    new Set(p.fields).size === p.fields.length &&
    Array.isArray(p.rows) &&
    p.rows.length <= 50 &&
    Number.isInteger(p.candidate.row_count) &&
    p.candidate.row_count > 0 &&
    p.candidate.row_count <= 10000 &&
    p.rows.length === Math.min(50, p.candidate.row_count - Number(after)) &&
    p.rows.every(
      (r, i) =>
        r?.ordinal === String(Number(after) + i + 1) &&
        !!r.values &&
        Object.keys(r.values).length === p.fields.length &&
        p.fields.every(
          (k) =>
            Object.hasOwn(r.values, k) &&
            (r.values[k] === null ||
              (typeof r.values[k] === "string" &&
                r.values[k]!.length <= 4096 &&
                (!decimals.has(k) ||
                  /^-?\d{1,12}\.\d{6}$/.test(r.values[k]!)))),
        ),
    ) &&
    p.next_after ===
      (Number(after) + p.rows.length < p.candidate.row_count
        ? String(Number(after) + 50)
        : null)
  );
}
export function source<T>(path: string, signal: AbortSignal) {
  return request<T>(path, signal, {}, "/api/local-source-data");
}
export async function page(
  release: string,
  snapshot: string,
  after: string,
  signal: AbortSignal,
) {
  const result = await source<Page>(
    `/pages/${release}/${snapshot}/${after}`,
    signal,
  );
  if (!validPage(result, release, snapshot, after))
    throw new ServiceError("response_invalid", "");
  return result;
}

const bounded = (v: unknown, max = 256): v is string =>
  typeof v === "string" && v.length > 0 && v.length <= max;
export function validCandidate(c: Candidate): boolean {
  return (
    !!c &&
    typeof c === "object" &&
    bounded(c.release_id) &&
    uuid.test(c.release_id) &&
    bounded(c.manifest_digest) &&
    digest.test(c.manifest_digest) &&
    bounded(c.source_revision, 19) &&
    /^[1-9][0-9]*$/.test(c.source_revision) &&
    bounded(c.version, 32) &&
    bounded(c.state, 32) &&
    bounded(c.created_at, 64) &&
    Number.isInteger(c.row_count) &&
    c.row_count > 0 &&
    c.row_count <= 10000 &&
    Array.isArray(c.blockers) &&
    c.blockers.length <= 20 &&
    c.blockers.every((v) => bounded(v, 128))
  );
}
export function validCandidates(c: Candidates): boolean {
  return (
    !!c &&
    Array.isArray(c.candidates) &&
    c.candidates.length <= 25 &&
    c.candidates.every(validCandidate) &&
    typeof c.more === "boolean" &&
    Array.isArray(c.blockers) &&
    c.blockers.length <= 20 &&
    c.blockers.every((v) => bounded(v, 128))
  );
}
export function validProfile(p: Profile): boolean {
  return (
    !!p &&
    bounded(p.account, 4096) &&
    bounded(p.workspace) &&
    uuid.test(p.workspace) &&
    bounded(p.connection) &&
    uuid.test(p.connection) &&
    bounded(p.period_start, 64) &&
    bounded(p.period_end, 64)
  );
}
