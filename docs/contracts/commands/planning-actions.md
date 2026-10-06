---
type: Command Contract
title: Planning action registry for confirmed inputs trust and operations
description: Defines shared proposed request result permission error and state contracts for the remediated screen actions.
status: draft
---

# Shared planning action contract

Tracking: [HB-16](https://linear.app/hybrid-billing/issue/HB-16). These are canonical proposed command IDs for the remediated planning paths, shared by UI/API/CLI/AI/workers. They are not deployed endpoints or permission grants. Remaining original commands keep their existing scope; this registry is not a claim that every product API is finished. The [domain contract](../../planning/v1.16/domain-contracts.md) defines the common boundary. Financial authority follows [Claim/numeric](../billing/claim-and-numeric.md) and [financial gates](../billing/inputs-pricing-and-erp.md).

## Common request and result

All mutations require `command_id`, `idempotency_key`, `expected_revision`, `target_id`, a typed command-specific body and a user rationale where the command changes authority, eligibility or a business deadline. Server resolves installation/workspace, authenticated user/service/client, current grants, role ceiling, current authority epoch and permitted target scope. A caller-supplied workspace/actor/passed-validation flag is never authoritative. Use canonical IDs rather than labels; decimal payloads use the numeric contract and timestamps have explicit timezone/offset semantics.

Idempotency identity is `(installation, client, command_id, idempotency_key)` plus canonical payload digest and bound authenticated initiating principal. Cross-principal reuse is a conflict; same-key/different-body conflicts. Same authorized principal/key/body returns the existing operation, with current access checked before protected result disclosure. It does not rerun the economic action. A compare-and-swap conflict requires reinspection and a new reviewed intent; do not silently apply against a new revision.

Successful synchronous mutation returns `operation_id`, `target_id`, `result_revision`, `business_state`, permitted evidence references and `as_of`. Durable asynchronous acceptance returns `job_id`, operation ID, technical state and authorized status/cancellation capabilities; it is not business success. A failed authority check cannot reveal forbidden target existence or cached prior data. Read replies include fixed filter/scope/revision, stable sort/cursor, completeness/freshness and next-page information. Bulk operations capture a target manifest and recheck current authorization; never send unbounded detail in one response.

| Error family | HTTP mapping proposal | Meaning and next action |
| --- | --- | --- |
| `unauthenticated` | 401 | Obtain valid authentication; no identity disclosure |
| `permission_denied`, `self_approval_denied` | 403 | Current permission/independence fails; generic unavailable response for forbidden target |
| `revision_conflict`, `idempotency_conflict`, `authority_epoch_conflict`, `evidence_stale`, `approval_stale` | 409 | Reinspect permitted current state; no automatic changed-target retry |
| `confirmation_prerequisite_failed`, `independent_approval_required`, `identity_proof_invalid`, `policy_incomplete`, `invalid_stage`, numeric/coverage validation errors | 422 | Show permitted field or prerequisite explanation; correcting a blocker requires new evidence/intent |
| `destination_denied`, `destination_unclassified`, `credential_audience_mismatch`, `retrieval_budget_exceeded` | 422 for synchronous configuration; explicit blocked job reason otherwise | Retain partial/quarantined evidence; expose no secret URL/address to unauthorized users |
| `rate_limited` | 429 | Bounded retry guidance; no duplicate operation |
| `authority_unavailable`, `dependency_unavailable` | 503 | Fail closed; preserve known state and use permitted reconciliation/status path |

External `outcome_unknown` is an explicit business/operation status with last confirmed evidence, not success and not a generic retry invitation. A successful query may return it with 200; accepted dispatch returns 202; synchronous completed commands return 200/201. All errors include request ID, stable code, permitted field errors, retryability and next action; no secrets/customer bodies. Transport choice does not weaken command semantics.

## Input confirmation

| ID | Required body / permission | Outcome |
| --- | --- | --- |
| `input.inspect` | Candidate ID and optional permitted fixed revision; scoped collection/evidence read | Server-built candidate, validation findings, independent comparison and current eligibility; customer-safe projection |
| `input.confirm_publish` | Candidate and scope expected revisions, server-issued evidence digest, explicit confirmation intent/rationale; scoped `input.confirm` and publication authority under one approved role mapping | Atomic immutable decision record plus eligible manifest publication |
| `input.reject` | Candidate expected revision, digest and rationale; scoped `input.confirm` | Rejected candidate with retained evidence; no deletion or alteration of prior eligible/issued history |

Proposed states: `received → normalized_candidate → validating → blocked | awaiting_confirmation → eligible`; rejected/stale/superseded candidates require a new valid candidate/evidence review. Candidate server digest covers input manifest, mapping, attribution, source profile, comparison method/version, expected/observed evidence and resolved findings. Mandatory independent full cross-validation must pass and all unwaivable blockers must be absent before awaiting-confirmation. An uploaded replacement total is not a delta; all snapshot/append/hybrid modes obey the gate.

`input.reject` is legal only for unpublished normalized_candidate, validating, blocked or awaiting_confirmation states at the expected revision. It cannot change an eligible manifest/decision to rejected. Withdrawal or correction of published eligibility requires its own governed impact/financial path preserving the historical decision. Received originals remain evidence; rejection is not source deletion.

At the atomic publication boundary recheck current actor authority, candidate and scope revisions, evidence digest and every prerequisite. Concurrency permits one eligible publication per expected scope revision; losers conflict. Background processing must apply the same checks when it actually publishes. Durable acceptance alone never reports eligible. Revoked authority cannot confirm or expose protected replay results; prior historical decisions remain immutable. Subsequent affected mapping/attribution/input/profile/comparison changes invalidate unpublished confirmation and require new review; they do not rewrite previously issued history.

S03 presents received/candidate history. S06 presents comparison, blockers and explicit confirm/reject. B09 consumes the bound input decision as a prerequisite to its separate monetary approval. This command does not add a second-human input confirmer requirement; existing financial segregation remains mandatory. No optional auto-confirmation exception is inferred.

## Authentication trust replacement

| ID | Body / distinct permission | State/effect |
| --- | --- | --- |
| `trust.propose` | Connection, old configuration revision/epoch, old/new digests, identity mapping, affected scope, legitimate-authority proof, session impact, recovery evidence, rationale; `trust.propose` | New draft change; no live activation |
| `trust.validate` | Change expected revision and bound evidence; `trust.validate` | Validated evidence or explicit blocked findings; validation is not approval |
| `trust.approve` | Validated change revision/digest, old epoch, decision rationale; independently granted `trust.approve` | Independently approved immutable record; proposer cannot self-approve |
| `trust.activate` | Approved change revision/digest and expected old epoch; `trust.activate` plus current independent approved authority | Atomic configuration publication, monotonic new epoch and durable invalidation intent |
| `trust.reject`, `trust.cancel` | Current change revision and rationale; authorized approval or proposal cancellation scope respectively | Rejected/cancelled only before activation; does not roll back live trust |

M02 must show proposer, independent approver, activation permission, proof status, old/new digest, affected principals and session impact. Generic identity/membership administration and infrastructure access confer none of these sensitive rights. Proposer/approver separation and current approver/activator rights are rechecked at activation. Activation cannot be self-approved by changing identities or using an unrelated membership grant. Actual independent authority enrollment/recovery procedure remains an approved installation policy.

Changing configuration/proof invalidates validation/approval. Old sessions, refresh families, step-up evidence and queued privileged commands cannot proceed on the old authority epoch. Consumers must verify current epoch/deny freshness; propagation uncertainty fails closed rather than relying on stale caches. Durable activation plus invalidation intent is not a claim that asynchronous consumers finished; report propagation state honestly. Rollback is a fresh governed change and a newer epoch, never restoration of prior session validity. Recovery uses independent legitimate-authority proof and cannot bypass separation.

## Holds and stage-specific schedules

`holds.request` and `holds.release` use typed `hold_type` (`billing`, `delivery`, `collections`), hold/case ID, immutable target manifest and partial scope, expected target/hold revisions, rationale and required approval evidence. Map permissions to distinct `hold.billing.request/release`, `hold.delivery.request/release`, `hold.collections.request/release` and scoped read. `holds.status` requires only scoped read for the currently authorized hold/case and target. It returns current revisions, as-of, effects and external uncertainty without mutation, new approval evidence or mutation confirmation. Generic case resolution grants none of the mutation permissions. Commands follow the existing [independent hold contract](../operations/holds-onboarding-and-favorites.md); B11 requests/reviews while I04 shows dispatch, effectiveness and external reconciliation.

Local active/released and external unsupported/requested/pending/confirmed-active/release-pending/confirmed-released/failed/unknown remain distinct. Release of one hold leaves every other hold effective. Local request/release and controllable dispatch serialize; a delayed external acknowledgement must match current operation/revision. Unknown release retains uncertainty and last confirmed state. No hold release sends a document, resumes billing, waives debt, changes payment terms or removes legal retention holds. Separate authorized operations decide those effects.

Each `StageSchedule` has `(cycle_id, schedule_id, stage)` with stage one of collection/calculation/approval/issuance/delivery/payment; original/current deadline, timezone/calendar-policy revision, owner, completion predicate, independent history, state and revision. Original deadline never changes. Several named schedules may exist within one stage; stage alone is not identity. U02 rows and detail context must show stage/schedule before mutation.

`schedule.change_request` requires cycle/schedule/stage, expected schedule revision, old/current deadline digest, proposed instant plus displayed local timezone/calendar revision, rationale, impact summary and `closing.schedule.write` within target scope. Validate stage/identity, calendar rules/DST interpretation and current authority. The command creates a proposed revision/approval request only. `schedule.apply_approved` atomically applies the exact approved digest and expected revision using separately authorized approval/execution permission; it does not recompute unrelated schedules or perform approval/issuance/delivery business actions. Dependency-derived suggestions require separate reviewed changes for each target.

Issued invoice payment due dates are governed by `payment.extension_request` / separately approved payment extension application from the payment contract, with distinct finance authority. Generic schedule permission cannot change them. A linked payment schedule updates only from that approved financial event, preserving original due date, effective extension and audit linkage. For unissued planning payment schedules, changing a planned date cannot pre-approve invoice terms. Delaying delivery leaves approval and payment dates unchanged unless their own authorized revisions are applied.

## Onboarding metrics and personal favorites

`onboarding.metrics.read` requires scoped onboarding read, cohort ID/revision, supported outcome policy and fixed observation cutoff. Return immutable eligible attempt unit IDs under current permissions, denominator/numerator definitions, connected and first-verified outcomes separately, exclusions/censoring, as-of and evidence. Query filtering creates an explicitly labeled authorized subset within that frozen cohort; it does not rewrite the cohort or compare mismatched numerators/denominators. Missing milestone/eligibility policy means unavailable, not 0%. Retries reuse the attempt unit. Classified waiting intervals use union, preserve overlap and missing timestamps; never sum overlapping categories into elapsed total. S02 must display policy/cutoff/denominator and cannot equate connection success with billing verification.

`favorites.add/remove/list` always derive user/workspace server-side and target a canonical currently authorized customer. Personal-preference write and customer read are both required; preference permission grants no customer access. Add/remove are idempotent; list/count/cache/search apply current rights and generic denied/unavailable output. Rename preserves identity. A versioned policy chooses purge or bounded hidden-reference retention on revocation and explicit regrant restoration. If policy is missing, activation is blocked; no indefinite retention or automatic resurrection default is invented. Termination never transfers favorites to a recreated identity. U01 hides unauthorized identifiers and removing a favorite changes no customer/grant.

## Planned contract cases

All cases below are NOT_RUN, linked to canonical screen/work records by the [remediation trace](../../planning/v1.16/remediation-trace.json). Existing H13, TR, FP, DH, OB, FV and PT cases remain applicable.

| ID | Independent expected behavior |
| --- | --- |
| IC-01 | Candidate with complete independent comparison publishes once with its decision; B09 still requires separate amount approval |
| IC-02 | Missing coverage/unresolved finding blocks despite confirmation intent; a server rejects caller-provided passed-validation assertions |
| IC-03 | Mapping/profile/evidence change or racing scope revision rejects old confirmation; no stale publication |
| IC-04 | Revocation/cross-workspace ID or inaccessible idempotent prior result reveals no protected data and grants no publication |
| IC-05 | Same command key with changed payload conflicts; concurrent equivalent confirmations never publish twice |
| IC-06 | Reject racing publication either rejects the still-unpublished candidate or conflicts after publication; eligible history never becomes rejected |
| TC-01 | Self approval or generic IdP-admin activation fails; independent valid approval plus current authority is required |
| TC-02 | Activation creates a newer epoch and durable invalidation; queued/session/refresh work on old epoch blocks even if propagation lags |
| TC-03 | Changed proof/old epoch, concurrent replacement or rollback to old configuration requires fresh approval and newer epoch |
| SC-01 | Approved delivery postponement changes only the selected cycle/schedule revision; approval/payment original/effective dates remain intact |
| SC-02 | Stale revision, ambiguous DST/calendar or stage/ID mismatch blocks; request acceptance is not schedule application |
| SC-03 | Generic schedule permission cannot extend issued invoice due date; authorized finance extension produces separately linked update |

No numeric defaults, deployed HTTP paths or completed compatibility claims follow from these proposals. Required permission mappings must be reviewed against actual role ceilings before activation.
