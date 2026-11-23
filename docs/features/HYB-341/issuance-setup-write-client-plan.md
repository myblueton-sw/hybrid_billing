---
type: ImplementationPlan
title: Contract setup proposal and confirmation client
description: Session-bound immutable setup submissions with exact retries and actor-owned recovery.
status: draft
---

# Outcome

HYB-341, machine:mac_mini. Root implements O03 proposal/confirmation within the adopted inline check-answers design. Independent refusal_review reviews the new client boundary; Claude reviews changed composition/copy only. Existing read-view and backend reviews remain reused. Owner performs browser verification.

# Immutable operation boundary

Generate a new configuration ID internally once per proposal form. Require deliberate sample preset and nonempty language selection plus rationale. Resolve context/options and inspect the exact operation/target before displaying the final check step. Never enable sending from role guesses, recorded action or unchecked source alone; action must be recheck_required with allowed permission and satisfied step-up. Use that inspection's current server revision for the new envelope. Every actual write rechecks on the server.

Freeze action, target, contract, actor/workspace, revision, argument map, rationale, attempt request/idempotency IDs and expected immutable receipt facts. Exact retry reuses the entire original envelope. Network failure, malformed success, an operation-bearing error or an unrecognized error contract remains unknown. Only a recognized correlated no-operation refusal is clean; a clean refusal to retry cannot disprove an earlier uncertain effect.

Validate receipt request/target/operation/revision/state and every captured immutable configuration/source fact. Proposal receipts may reflect a retained confirmed record on replay; never infer current eligibility from historical receipt. Confirmation must match the captured configuration digest and current actor as retained confirmer. Any receipt mismatch remains unknown.

# Retention and recovery

Keep attempts in a session-bound store above tab/contract remounts, with one unresolved write at a time and bounded retained entries. Scope changes clear current reads, not unresolved attempts. A different session cannot submit or retry another session's operation. Reauthentication requires a fresh inspection for new submissions; exact retries keep original identity.

For fresh owned recovery, require freshly inspected retained unknown history and matching action guidance. Proposal recovery copies the original setup date/digest, settings and proposal rationale. Confirmation recovery uses the retained configuration digest and reserved actor; it cannot take over another confirmer. Reconcile only a validated recovery receipt; reads and changed permissions never erase uncertainty. Legacy records have no public recovery controls.

# Verification and limits

Logic tests cover closed action-response combinations, exact frozen retries, cross-session refusal, correlated clean versus uncertain failures, malformed/mismatched receipts, late reads, one unresolved operation, unknown preservation after retry denial and exact owned recovery. UI will retain unsent input or use the existing discard guard; session pending outcomes remain visible across navigation. No actual issuance, payment, deployment, merge or browser acceptance is implied.

# Implemented client boundary

The action decoder, intent constructors, mutation transport and session-bound write store are implemented. Independent refusal_review returned scoped design PASS and source PASS. Requested successful confirmation and confirmed historical proposal-receipt cases were added. A focused corrective review also passed explicit same-actor/new-session binding and invalidation of guidance obtained during a pending write. The session binding exists only in runtime memory and is excluded from the serialized command body; no credential values were committed.

Initial setup logic tests passed (23). After adding confirmation/replay, new-session and pending-guidance cases, the focused action/mutation/store suite passed (12). Type checking identified inferred optional undefined properties in heterogeneous object unions; explicit string-map contextual types resolved that compiler issue without changing wire behavior. Final console typecheck passed; the six mutation tests also passed after checking that the runtime session binding is absent from the command body. The reviewer ran no tests. This checkpoint does not claim write buttons, session-remount UI behavior or complete operator setup; those remain the next integration step.
