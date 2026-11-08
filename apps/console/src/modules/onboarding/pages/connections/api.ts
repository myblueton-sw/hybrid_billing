import { request, ServiceError } from "../../../sources/pages/live-imports/api";
import { uuid, digest } from "../../../sources/pages/source-data/api";
export { ServiceError, uuid };
export type Connection = {
  classification: string;
  connection: string;
  workspace: string;
  account: string;
  subaccount: string | null;
  period_start: string;
  period_end: string;
  policy_revision: string;
  observed_at: string;
  contracts: { version: string; revision: string; digest: string }[];
  max_bytes: number;
  max_rows: number;
  completeness: string;
  provider_authentication: string;
  provider_reads: string;
  invoices: string;
  holds: { quarantine: boolean; unresolved: boolean };
  actions: ("imports" | "source-data" | "reconciliation")[];
  latest_candidate: null | {
    release_id: string;
    source_revision: string;
    state: string;
    created_at: string;
    manifest_digest: string;
    version: string;
  };
};
const bounded = (v: unknown, max = 256): v is string =>
  typeof v === "string" && v.length > 0 && v.length <= max;
const timestamp = (v: unknown): v is string =>
  bounded(v, 64) &&
  /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?(?:Z|[+-]\d{2}:\d{2})$/.test(
    v,
  ) &&
  Number.isFinite(Date.parse(v));
const revision = (v: unknown) => bounded(v, 19) && /^[1-9][0-9]*$/.test(v);
export function validConnection(c: Connection, expected?: string): boolean {
  const r = c?.latest_candidate;
  return (
    !!c &&
    c.classification === "local-connection-metadata" &&
    bounded(c.connection, 36) &&
    uuid.test(c.connection) &&
    (!expected || c.connection === expected) &&
    bounded(c.workspace, 36) &&
    uuid.test(c.workspace) &&
    bounded(c.account, 4096) &&
    (c.subaccount === null || bounded(c.subaccount, 4096)) &&
    timestamp(c.period_start) &&
    timestamp(c.period_end) &&
    revision(c.policy_revision) &&
    timestamp(c.observed_at) &&
    Array.isArray(c.contracts) &&
    c.contracts.length > 0 &&
    c.contracts.length <= 100 &&
    new Set(c.contracts.map((x) => x?.version)).size === c.contracts.length &&
    c.contracts.every(
      (x) =>
        !!x &&
        bounded(x.version, 32) &&
        revision(x.revision) &&
        bounded(x.digest, 64) &&
        digest.test(x.digest),
    ) &&
    c.max_bytes === 8 * 1024 * 1024 &&
    c.max_rows === 10000 &&
    c.completeness === "unknown" &&
    c.provider_authentication === "unconnected" &&
    c.provider_reads === "unconnected" &&
    c.invoices === "unconnected" &&
    !!c.holds &&
    typeof c.holds.quarantine === "boolean" &&
    typeof c.holds.unresolved === "boolean" &&
    Array.isArray(c.actions) &&
    ["imports", "source-data|reconciliation"].includes(c.actions.join("|")) &&
    (r === null ||
      (!!r &&
        bounded(r.release_id, 36) &&
        uuid.test(r.release_id) &&
        revision(r.source_revision) &&
        ["pending", "verified", "sealed"].includes(r.state) &&
        timestamp(r.created_at) &&
        bounded(r.manifest_digest, 64) &&
        digest.test(r.manifest_digest) &&
        bounded(r.version, 32)))
  );
}
export async function connection(
  expected: string | undefined,
  signal: AbortSignal,
) {
  if (expected && !uuid.test(expected))
    throw new ServiceError("request_invalid", "");
  const data = await request<Connection>(
    expected ? "/" + expected : "",
    signal,
    {},
    "/api/local-connections",
  );
  if (!validConnection(data, expected))
    throw new ServiceError("response_invalid", "");
  return data;
}
export function actionUrl(
  action: Connection["actions"][number],
  data: Connection,
  locale: string,
) {
  const q = new URLSearchParams({ lang: locale, service: "local" }),
    r = data.latest_candidate;
  if (action === "source-data" && r) q.set("snapshot", r.manifest_digest);
  return "#/" + action + (r ? "/" + r.release_id : "") + "?" + q;
}
