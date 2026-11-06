export const kinds = [
  "billing_result",
  "final_erp_amount",
  "adjustment",
  "policy",
  "monthly_recognition",
] as const;
export type Kind = (typeof kinds)[number];
export const views = [
  "pending",
  "decisions",
  "delegation",
  "history",
  "recognition",
] as const;
export type View = (typeof views)[number];
export type Principal = { id: string; label: string };
export type ApprovalAmount = {
  currency: string;
  basis: "net" | "tax" | "gross" | "expense";
  base: string | null;
  candidate: string | null;
  difference: string | null;
  erp: string | null;
};
export type ApprovalRecord = {
  id: string;
  kind: Kind;
  status:
    "pending" | "blocked" | "approved" | "rejected" | "expired" | "superseded";
  target: {
    id: string;
    label: string;
    entity: string | null;
    customer: string | null;
    period: string | null;
  };
  author: Principal | null;
  approver: Principal | null;
  updated_at: string;
  deadline: string | null;
  decided_at: string | null;
  amounts: ApprovalAmount[];
  erp: {
    applicability: "applicable" | "not_applicable" | "unknown";
    external_revision: string | null;
    observed_at: string | null;
  };
  evidence: {
    input_digest: string | null;
    result_digest: string | null;
    revision: string | null;
    dependencies: "current" | "stale" | "unknown";
    completeness: "complete" | "partial" | "unknown";
    validation: "passed" | "failed" | "unknown";
    blockers: { code: string; text: string }[];
  };
  authority: {
    separation: "satisfied" | "not_satisfied" | "unknown";
    limit: "within" | "exceeded" | "unknown";
    valid_until: string | null;
  };
};
export type Data = {
  schema: 1;
  numeric_format: "decimal-strings-read-v1";
  read_only: true;
  commands: "unconnected";
  source_mode: "synthetic" | "live";
  auth_scope: string;
  snapshot: string;
  permission_revision: string;
  actor: Principal;
  view: Exclude<View, "delegation">;
  selection: { kind: Kind | ""; q: string };
  cursor: string | null;
  next_cursor: string | null;
  observed_at: string;
  total: number | null;
  records: ApprovalRecord[];
};
export type ReadResult =
  { kind: "loaded"; data: Data } | { kind: "unconnected" };
