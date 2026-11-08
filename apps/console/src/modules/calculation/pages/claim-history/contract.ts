export const views = ["ledger", "partial", "reversals", "references"] as const;
export type View = (typeof views)[number];
export const protections = [
  "available",
  "reserved",
  "outcome_unknown",
  "applied",
  "release_pending",
  "released",
  "replacement_pending",
  "cancelled",
  "superseded",
  "unknown",
] as const;
export type Protection = (typeof protections)[number];
export type ClaimRecord = {
  id: string;
  revision: string | null;
  application_id: string | null;
  obligation_id: string | null;
  purpose: string | null;
  target: {
    label: string;
    entity: string | null;
    customer: string | null;
    contract: string | null;
    component: string | null;
    period: string | null;
  };
  protection: Protection;
  partial: boolean | null;
  updated_at: string;
  coverage: {
    identity: string | null;
    universe: string | null;
    unit: string | null;
    unit_version: string | null;
    description: string | null;
    reserved: string | null;
    applied: string | null;
    reversed: string | null;
    available: string | null;
  };
  amounts: {
    currency: string;
    basis: "net" | "tax" | "gross" | "expense";
    applied: string | null;
    remaining: string | null;
    reversed: string | null;
  }[];
  lineage: {
    source: string | null;
    allocation: string | null;
    sales: string | null;
    charge_line: string | null;
    submission: string | null;
  };
  correction: {
    original_claim_id: string | null;
    linked_claim_ids: string[];
    disposition:
      | "none"
      | "money_only"
      | "quantity_reversal"
      | "replacement_pending"
      | "cancelled"
      | "superseded"
      | "unknown";
  };
  external: {
    kind: "erp_order" | "prepayment" | "invoice" | "submission" | "other";
    id: string;
    revision: string | null;
    observed_at: string | null;
    outcome: "pending" | "unknown" | "applied" | "not_applied" | "cancelled";
  }[];
  blockers: { code: string; text: string }[];
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
  view: View;
  selection: { state: Protection | ""; q: string };
  cursor: string | null;
  next_cursor: string | null;
  total: number | null;
  observed_at: string;
  records: ClaimRecord[];
};
export type ReadResult =
  { kind: "loaded"; data: Data } | { kind: "unconnected" };
