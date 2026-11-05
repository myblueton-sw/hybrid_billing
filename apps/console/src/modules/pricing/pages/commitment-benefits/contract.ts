export const profile = "commitment-source-evidence-sums-v1";
export const fieldNames = [
  "id",
  "name",
  "type",
  "category",
  "status",
  "charge_category",
  "charge_class",
  "charge_frequency",
] as const;
export type FieldName = (typeof fieldNames)[number];
export type FieldState =
  | "value"
  | "source_null"
  | "absent"
  | "blank"
  | "unmapped_version"
  | "invalid_type";
export type Field = { state: FieldState; value: string | null };
export type Measure = {
  coefficient: string | null;
  present: number;
  missing: number;
};
export type Costs = {
  rows: number;
  billed_cost: Measure;
  effective_cost: Measure;
};
export type Amount = Costs & { currency: string };
export type Metadata = {
  values: string[];
  distinct: number;
  truncated: boolean;
};
export type Group = Costs & {
  id: string;
  source_id: string;
  tenant: string;
  account: string;
  provider: string;
  billing_account: string;
  sub_account: string | null;
  currency: string;
  metadata: Record<Exclude<FieldName, "id">, Metadata>;
  statuses: Record<"used" | "unused" | "unknown", Costs>;
  versions: string[];
  mapping: { version_mapped_rows: number; version_unmapped_rows: number };
  intervals: {
    charge_start: string;
    charge_end: string;
    billing_start: string;
    billing_end: string;
  };
};
export type Charge = {
  id: string;
  group: string;
  tenant: string;
  account: string;
  provider: string;
  billing_account: string;
  sub_account: string | null;
  currency: string;
  service: string | null;
  billed_cost: string | null;
  effective_cost: string | null;
  charge_start: string;
  charge_end: string;
  billing_start: string;
  billing_end: string;
  version: string;
  batch: string;
  ordinal: string;
  fields: Record<FieldName, Field>;
  evidence_class: string;
};
export type Detail = {
  group: string;
  rows: Charge[];
  after: number;
  next_after: number | null;
  total_rows: number;
};
export type Account = {
  id: string;
  tenant: string;
  provider: string;
  billing_account: string;
  sub_account: string | null;
  rows: number;
};
export type Data = {
  schema: 1;
  scale: 18;
  profile: typeof profile;
  classification: "collected-commitment-evidence-only";
  financial_authority: false;
  snapshot: string;
  source_snapshot: string;
  mapping_digest: string;
  company: string | null;
  versions: string[];
  observed_at: string;
  selection: { period: string; provider: string; account: string; q: string };
  period_basis: "charge-start-in-tenant-timezone-no-proration";
  coverage: {
    slices: number;
    finalized_slices: number;
    complete_through: string | null;
    expectations: string;
  };
  options: { providers: string[]; periods: string[]; currencies: string[] };
  authorized_rows: number;
  row_count: number;
  identified_rows: number;
  unidentified_rows: number;
  group_count: number;
  identified_amounts: Amount[];
  unidentified_amounts: Amount[];
  unidentified_reasons: Record<Exclude<FieldState, "value">, number>;
  groups: Group[];
  after: number;
  next_after: number | null;
  accounts: Account[];
  account_count: number;
  account_matches: number;
  account_after: number;
  next_account_after: number | null;
  selected: Detail | null;
};
