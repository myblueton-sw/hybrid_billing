import { request } from "../live-imports/api";
export { ServiceError } from "../live-imports/api";
export type Candidate = {
  release_id: string;
  source_revision: string;
  manifest_digest: string;
  row_count: number;
  state: string;
  version: string;
  created_at: string;
  blockers: string[];
};
export type Profile = {
  account: string;
  workspace: string;
  connection: string;
  period_start: string;
  period_end: string;
};
export type Candidates = {
  candidates: Candidate[];
  more: boolean;
  blockers: string[];
};
export type Comparison = {
  currency: string;
  consumed_unit: string | null;
  pricing_unit: string | null;
  measure: string;
  expected: string | null;
  actual: string | null;
  difference: string | null;
  expected_present: number;
  actual_present: number;
  expected_missing: number | null;
  actual_missing: number | null;
  state: string;
};
export type Result = {
  candidate: Candidate;
  observed_at: string;
  state: string;
  blockers: string[];
  checks: { id: string; passed: boolean }[];
  comparisons: Comparison[];
  comparison_count: number;
  mismatch_count: number;
  incomplete_count: number;
  truncated: boolean;
  evidence?: Record<string, string>;
};
export function reconcile<T>(
  path: string,
  signal: AbortSignal,
  init: RequestInit = {},
) {
  return request<T>(path, signal, init, "/api/local-reconciliation");
}
