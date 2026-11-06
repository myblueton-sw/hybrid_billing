export const views = [
  "balances",
  "reservations",
  "cases",
  "adjustments",
  "payment-terms",
] as const;
export type View = (typeof views)[number];
export type Authority = {
  kind: "platform" | "external" | "unknown";
  reference: string | null;
  observed_at: string | null;
  freshness: "current" | "stale" | "unknown";
};
export type Dimension =
  | { kind: "money"; currency: string; unit: null }
  | { kind: "quantity"; currency: null; unit: string };
export type Money = { currency: string; value: string | null };
export type Hold = {
  type: "billing" | "delivery" | "collections";
  effect: "requested" | "effective" | "released" | "pending" | "unknown";
  reference: string | null;
  revision: string | null;
  target_ref: string | null;
  observed_at: string | null;
  external_effect: "confirmed" | "pending" | "unknown" | "not_applicable";
};
type Common = {
  id: string;
  revision: string | null;
  label: string;
  target: {
    entity: string | null;
    customer: string | null;
    contract: string | null;
    period: string | null;
  };
  updated_at: string;
  authority: Authority;
  blockers: { code: string; text: string }[];
};
export type BalanceRecord = Common & {
  kind: "balance";
  dimension: Dimension;
  opening: string | null;
  increases: string | null;
  applied: string | null;
  expired: string | null;
  reversed: string | null;
  closing: string | null;
  available: string | null;
  reserved: string | null;
  migration_ref: string | null;
  included_history_ref: string | null;
};
export type ReservationRecord = Common & {
  kind: "reservation";
  dimension: Dimension;
  value: string | null;
  credit_ref: string | null;
  document_ref: string | null;
  original_ref: string | null;
  state:
    | "reserved"
    | "applied"
    | "released"
    | "outcome_unknown"
    | "reversed"
    | "unknown";
};
export type CaseRecord = Common & {
  kind: "case";
  state: "open" | "in_review" | "resolved" | "closed" | "unknown";
  original_document_ref: string | null;
  amounts: Money[];
  owner_label: string | null;
  received_at: string | null;
  deadline: string | null;
  internal_note: string | null;
  public_reply: string | null;
  resolution:
    | "confirmed_normal"
    | "correction_requested"
    | "corrected"
    | "unresolved"
    | "unknown";
  resolution_ref: string | null;
  consent: {
    state:
      "recorded" | "declined" | "not_recorded" | "not_applicable" | "unknown";
    evidence_ref: string | null;
  };
  holds: Hold[];
};
export type AdjustmentRecord = Common & {
  kind: "adjustment";
  original_document_ref: string | null;
  original_claim_ref: string | null;
  correction_ref: string | null;
  reason: string | null;
  evidence_ref: string | null;
  approval_ref: string | null;
  adjustment_kind:
    "credit" | "refund" | "quantity_reversal" | "other" | "unknown";
  state: "draft" | "pending" | "applied" | "rejected" | "unknown";
  amounts: Money[];
};
export type PaymentRecord = Common & {
  kind: "payment";
  original_document_ref: string | null;
  issuance: "issued" | "not_issued" | "internal" | "unknown";
  term_ref: string | null;
  term_label: string | null;
  time_zone: string | null;
  issue_date: string | null;
  original_due_date: string | null;
  effective_due_date: string | null;
  receivable: Money;
  receipt: {
    state: "confirmed" | "unknown" | "not_provided";
    reference: string | null;
    observed_at: string | null;
    freshness: "current" | "stale" | "unknown";
  };
  overdue:
    | "on_time"
    | "due_today"
    | "overdue"
    | "fully_paid"
    | "unknown"
    | "not_applicable";
  dispute: "open" | "none" | "unknown";
  holds: Hold[];
};
export type EvidenceRecord =
  | BalanceRecord
  | ReservationRecord
  | CaseRecord
  | AdjustmentRecord
  | PaymentRecord;
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
  selection: { q: string };
  cursor: string | null;
  next_cursor: string | null;
  total: number | null;
  observed_at: string;
  records: EvidenceRecord[];
};
export type ReadResult =
  { kind: "loaded"; data: Data } | { kind: "unconnected" };
