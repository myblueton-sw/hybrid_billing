import type { AccessGrant, AccessDelegation, AccessRequest, AccessSetting, Revocation, CleanupItem, Preview, Ceiling, ScopeRow } from "./admin";
import { array, boolean, choice, integer, shape, strings, text } from "./strict-read";

const scope = shape<ScopeRow>({ customer: text, legal_entity: text, contract: text, period: text });
const terms = { action: text, kind: text, fields: strings, scope, not_before: text, not_after: text, effective: boolean, version: integer };
export const grantsOf = array(shape<AccessGrant>({ ...terms, id: text, subject: text, client: text,
  state: choice("active", "revoked", "expired", "pending", "revocation_unconfirmed") }));
export const delegationsOf = array(shape<AccessDelegation>({ ...terms, id: text, delegator: text, delegate: text, parent: text, redelegate: boolean,
  state: choice("active", "revoked", "expired", "pending", "ended", "revocation_unconfirmed") }));
export const catalogOf = array(shape<Ceiling>({ action: text, kind: text, fields: strings, until: text }));
export const subjectsOf = array(shape({ user_id: text, login: text }));
export const settingsOf = array(shape<AccessSetting>({ name: text, value: integer, unit: text, default: integer, min: integer, max: integer, source: text }));
export const requestsOf = array(shape<AccessRequest>({
  id: text, change: choice("add", "revoke"), subject: text, grant: text, action: text, kind: text, fields: strings, customer: text, days: integer,
  digest: text, state: choice("pending", "applied", "rejected", "withdrawn", "expired", "superseded"), current: boolean,
  proposer: text, proposed_at: text, expires_at: text, reason: text, decided_by: text, decided_at: text, decision_reason: text, result: text,
  eligible_approvers: integer, can_approve: boolean, can_reject: boolean, can_withdraw: boolean, may_end: strings,
}));
const cleanup = shape<CleanupItem>({ item: text, state: choice("done", "not_applicable", "outside_scope", "not_started"), reason: text, refs: strings });
export const revocationsOf = array(shape<Revocation>({
  id: text, origin: choice("this_process", "ledger_replay"), recorded_at: text, target_kind: choice("grant", "delegation"), target: text,
  target_present: boolean, replay: choice("", "reapplied", "absent", "protected"),
  state: choice("received", "deny_confirmed", "cleanup_running", "revocation_complete", "decision_unconfirmed"),
  history: array(shape({ state: text, at: text })), ended_delegations: strings, deny_generation: integer,
  job_state: choice("running", "waiting", "succeeded"), job_reason: text, requested_by: text, approved_by: text, request: text,
  received_at: text, deny_at: text, receipt: shape({ sequence: integer, watermark: text, reason: text }), verified: boolean,
  cleanup: array(cleanup), completed_at: text,
}));
export const previewsOf = array(shape<Preview>({ action: text, field: text, allowed: boolean, reason: text, delegations: strings }));
