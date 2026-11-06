import type { Locale } from "../../types";
export type WorkId = "O01" | "O02" | "O03" | "U03" | "U04" | "S03" | "A01";
export type Words = readonly [string, string, string];
export type Props = {
  screenId: string;
  audience?: "admin" | "customer" | "unauthenticated";
  locale: Locale;
  params: URLSearchParams;
  view: string;
  onState: (state: string, items: string[]) => void;
  onDirty: (dirty: boolean) => void;
};
export const text = (l: Locale, words: Words) =>
  words[l === "ko" ? 0 : l === "en" ? 1 : 2];
export const fields = {
  O01: {
    tree: ["as_of", "organization_id", "parent", "relationship", "interval"],
    accounts: [
      "account_id",
      "source_scope",
      "payer_role",
      "relationship",
      "unassigned_reason",
    ],
    groups: ["group_id", "members", "view_scope", "revision"],
    history: [
      "organization_id",
      "relationship",
      "valid_from",
      "valid_until",
      "revision",
    ],
  },
  O02: {
    targets: [
      "change_kind",
      "target_manifest",
      "expected_revision",
      "effective_at",
    ],
    impact: [
      "pool_dependencies",
      "contract_dependencies",
      "amount_currency",
      "disclosure_impact",
      "erp_impact",
    ],
    approval: ["approval_hash", "current_authority", "schedule_at", "revision"],
    result: ["job_id", "dependency_unit", "result", "checkpoint"],
  },
  O03: {
    parties: [
      "transaction_path",
      "buyer",
      "payer",
      "issuer",
      "legal_entity",
      "currency",
      "timezone",
    ],
    schedule: [
      "service_interval",
      "billing_interval",
      "billing_cycle",
      "proration_policy",
      "fx_policy",
    ],
    products: [
      "components",
      "allowances",
      "service_fee_eligibility",
      "rate_revision",
    ],
    documents: [
      "erp_order",
      "prepayment_reference",
      "payment_reference",
      "recipient_reference",
      "template_revision",
    ],
    termination: [
      "contract_status",
      "final_settlement_status",
      "termination_at",
      "final_evidence",
    ],
    "payment-terms": [
      "term_revision",
      "days_from_issue",
      "calendar_policy",
      "timezone",
      "effective_interval",
    ],
    recognition: [
      "billing_cadence",
      "recognition_cadence",
      "legal_entity",
      "amount_basis",
      "monthly_batches",
      "reconciliation",
    ],
  },
  U03: {
    tasks: [
      "job_id",
      "job_state",
      "waiting_reason",
      "result_counts",
      "checkpoint",
    ],
    notifications: [
      "incident_id",
      "read_state",
      "acknowledgement",
      "assignment",
      "resolution",
      "occurred_at",
    ],
    detail: [
      "job_id",
      "target_manifest",
      "waiting_reason",
      "safe_error",
      "next_action",
    ],
    delivery: [
      "channel_acceptance",
      "delivery_state",
      "user_read_state",
      "observed_at",
    ],
    "payment-terms": [
      "due_date",
      "receipt_freshness",
      "reminder_stage",
      "collection_hold",
    ],
  },
  U04: {
    findings: [
      "finding_id",
      "kind",
      "scope_period",
      "comparison_basis",
      "amount_meaning",
      "currency",
      "impact_certainty",
      "review_state",
      "assignee",
    ],
    review: [
      "evidence_reference",
      "source_revision",
      "comparison_basis",
      "review_reason",
      "resolution_reference",
    ],
    evidence: [
      "source_reference",
      "contract_reference",
      "correction_reference",
      "current_access",
    ],
  },
  S03: {
    imports: [
      "receipt_id",
      "source_period",
      "original_hash",
      "revision",
      "rows_bytes",
      "job_state",
    ],
    quarantine: [
      "receipt_id",
      "safe_rejection_reason",
      "metadata_reference",
      "conflict_revision",
    ],
    completeness: [
      "expected_manifest",
      "received_partitions",
      "watermark",
      "missing_scope",
      "cross_validation",
    ],
    extraction: [
      "original_reference",
      "candidate_revision",
      "comparison_digest",
      "verifier_evidence",
      "mapping_profile_revision",
      "eligibility",
    ],
  },
  A01: {
    conversation: ["scope", "period", "currency", "request", "source_as_of"],
    plan: [
      "command_reference",
      "target_manifest",
      "input_hash",
      "policy_revision",
      "required_approval",
    ],
    results: ["job_id", "acceptance", "job_state", "result_reference"],
    retention: [
      "conversation_policy",
      "hold",
      "deletion_request",
      "audit_retention",
    ],
  },
} as const;
export const validReference = (s: string) =>
  /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/.test(s);
export const validDate = (s: string) =>
  /^\d{4}-\d{2}-\d{2}$/.test(s) &&
  Number.isFinite(Date.parse(s + "T00:00:00Z")) &&
  new Date(s + "T00:00:00Z").toISOString().slice(0, 10) === s;
export type EvidenceState = {
  kind: "unconnected";
  authoritativeScope: null;
  asOf: null;
  exactTotal: null;
};
export const unavailable: EvidenceState = {
  kind: "unconnected",
  authoritativeScope: null,
  asOf: null,
  exactTotal: null,
};
