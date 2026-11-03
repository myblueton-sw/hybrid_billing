import { request, ServiceError } from "../live-imports/api";
import {
  uuid,
  digest,
  rowOrdinal,
  ordinal,
  sourceFieldNames,
  decimalFieldNames,
} from "../source-data/api";
export { ServiceError };
export type Context = {
  release: string;
  snapshot: string;
  row: string;
  after: string;
};
export type Field = {
  name: string;
  source_name: string | null;
  source_present: boolean;
  source_text: string | null;
  normalized: string | null;
};
export type Evidence = {
  release_id: string;
  snapshot: string;
  ordinal: string;
  row_count: number;
  source_revision: string;
  version: string;
  registry_digest: string;
  original_digest: string;
  original_bytes: number;
  method: string;
  classification: string;
  warehouse_integrity: string;
  retention: string;
  public_document: string;
  observed_at: string;
  scope: {
    workspace: string;
    account: string;
    period_start: string;
    period_end: string;
  };
  fields: Field[];
};
export function context(
  release: string | undefined,
  q: URLSearchParams,
): Context | null {
  const snapshot = q.get("snapshot"),
    row = q.get("row"),
    after = q.get("after");
  if (!release && snapshot === null && row === null && after === null)
    return null;
  if (
    !release ||
    !uuid.test(release) ||
    !snapshot ||
    !digest.test(snapshot) ||
    !row ||
    !rowOrdinal.test(row) ||
    after === null ||
    !ordinal.test(after) ||
    Number(row) <= Number(after) ||
    Number(row) > Number(after) + 50
  )
    throw new ServiceError("request_invalid", "");
  return { release, snapshot, row, after };
}
const bounded = (v: unknown, max = 256): v is string =>
  typeof v === "string" && v.length > 0 && v.length <= max;
const nullable = (v: unknown, max = 4096): v is string | null =>
  v === null || (typeof v === "string" && v.length <= max);
export function validEvidence(e: Evidence, c: Context): boolean {
  return (
    !!e &&
    e.release_id === c.release &&
    e.snapshot === c.snapshot &&
    e.ordinal === c.row &&
    Number.isInteger(e.row_count) &&
    e.row_count >= Number(c.row) &&
    e.row_count <= 10000 &&
    bounded(e.source_revision, 19) &&
    /^[1-9][0-9]*$/.test(e.source_revision) &&
    bounded(e.version, 32) &&
    bounded(e.registry_digest, 64) &&
    digest.test(e.registry_digest) &&
    bounded(e.original_digest, 64) &&
    digest.test(e.original_digest) &&
    Number.isInteger(e.original_bytes) &&
    e.original_bytes > 0 &&
    e.original_bytes <= 8 * 1024 * 1024 &&
    e.method === "pinned-original-record-extract-v1" &&
    e.classification === "local-source-evidence" &&
    e.warehouse_integrity === "not-revalidated" &&
    e.retention === "unknown" &&
    e.public_document === "unconnected" &&
    bounded(e.observed_at, 64) &&
    !!e.scope &&
    bounded(e.scope.workspace, 36) &&
    uuid.test(e.scope.workspace) &&
    bounded(e.scope.account, 4096) &&
    bounded(e.scope.period_start, 64) &&
    bounded(e.scope.period_end, 64) &&
    Array.isArray(e.fields) &&
    e.fields.length <= 16 &&
    new Set(e.fields.map((f) => f?.name)).size === e.fields.length &&
    [
      "billing_account_id",
      "billing_currency",
      "billed_cost",
      "effective_cost",
      "charge_period_start",
      "charge_period_end",
    ].every((name) => e.fields.some((f) => f?.name === name)) &&
    e.fields.every(
      (f) =>
        !!f &&
        sourceFieldNames.has(f.name) &&
        nullable(f.source_name, 64) &&
        (f.source_name === null ||
          /^[A-Za-z][A-Za-z0-9_]{0,63}$/.test(f.source_name)) &&
        typeof f.source_present === "boolean" &&
        (f.source_present
          ? f.source_name !== null &&
            typeof f.source_text === "string" &&
            f.source_text.length <= 4096
          : f.source_text === null) &&
        nullable(f.normalized) &&
        (!decimalFieldNames.has(f.name) ||
          f.normalized === null ||
          /^-?\d{1,12}\.\d{6}$/.test(f.normalized)),
    )
  );
}
export async function inspect(c: Context, signal: AbortSignal) {
  const result = await request<Evidence>(
    `/rows/${c.release}/${c.snapshot}/${c.row}`,
    signal,
    {},
    "/api/local-evidence",
  );
  if (!validEvidence(result, c)) throw new ServiceError("response_invalid", "");
  return result;
}
export function sourceReturn(c: Context, locale: string) {
  return (
    "#/source-data/" +
    c.release +
    "?" +
    new URLSearchParams({
      lang: locale,
      service: "local",
      snapshot: c.snapshot,
      after: c.after,
      row: c.row,
      focus: "evidence",
    })
  );
}
