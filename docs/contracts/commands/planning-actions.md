---
type: Command Contract
title: Planning action registry for confirmed inputs trust and operations
description: Defines shared proposed request result permission error and state contracts for the remediated screen actions, adjustment and dependent-period lifecycle, operational notification rules, demo scenario runs and ERP route switching.
status: draft
---

# Shared planning action contract

Tracking: [HB-16](https://linear.app/hybrid-billing/issue/HB-16); [HB-23](https://linear.app/hybrid-billing/issue/HB-23) adds the adjustment and dependent-period, operational notification rule, demo scenario and ERP route families below; bindings for screens whose specifications are local companions are recorded in the [screen binding register](../../ui/screen-binding-register.md) and are not adopted until [HB-24](https://linear.app/hybrid-billing/issue/HB-24). These are canonical proposed command IDs for the remediated planning paths, shared by UI/API/CLI/AI/workers. They are not deployed endpoints or permission grants. Remaining original commands keep their existing scope; this registry is not a claim that every product API is finished. The [domain contract](../../planning/v1.16/domain-contracts.md) defines the common boundary. Financial authority follows [Claim/numeric](../billing/claim-and-numeric.md) and [financial gates](../billing/inputs-pricing-and-erp.md).

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

## Pre-issuance adjustment and dependent-period lifecycle

Source: B0486.R007, B0495; local AR-07 and REV01 as cited by the [monthly settlement schema](../../architecture/data-ownership/monthly-settlement-schema.md), whose AdjustmentDraft, AdjustmentLine/AdjustmentComputation, AdjustmentDecision/SettlementAdjustmentLink, FinalSettlementSnapshot and dependency-edge rules these commands operate. Supplies the command part of HB-15 FA-04. Schema names are logical proposals; the approval-subject typing of FA-01 belongs to the schema documents and is referenced, not redefined, here.

| ID | Required body / permission | Outcome |
| --- | --- | --- |
| `adjustment.draft` | Unissued original close/run/result/line refs, affected item/pool/period manifest, kind `quantity`, `fixed_amount` or `target_total`, old and proposed Quantity or Money, declared basis (`net`, `tax` or `gross`), currency, reason and evidence refs, expected base result revision and digest, expected draft revision when revising; scoped `adjustment.write` | New or revised AdjustmentDraft. No computation, approval or financial effect. A generic edit of an invoice total is not accepted |
| `adjustment.compute` | Draft ID and expected draft revision; scoped `adjustment.compute` | Job computing the explicit signed delta, target minus current **at the same declared basis**, tax/FX/pool recomputation and full impact manifest. Ends `validated` with a computation digest or `blocked` with typed reasons. Acceptance of the job is not validation |
| `adjustment.approve`, `adjustment.reject` | Draft ID, exact computation digest, expected draft revision, decision rationale; independently granted `adjustment.approve` within the approver's ceiling | Immutable AdjustmentDecision. The draft author is refused with `self_approval_denied`. Adjustment approval never substitutes for final amount approval |
| `adjustment.cancel` | Draft ID, expected draft revision, rationale; author's `adjustment.write` or authorized cancellation scope | Cancelled before application only; no effect on any result |
| `adjustment.apply` | Approved decision ID and digest, expected base result revision, expected draft revision; scoped `adjustment.apply` plus current independent approved decision | Atomically, once: new unissued result revision with immutable AdjustmentLine and SettlementAdjustmentLink. Replay of the same intent returns the existing revision. Every FinalSettlementSnapshot approval bound to the previous result becomes stale |
| `settlement.dependency_status` | FinalSettlementSnapshot or BillingCycle ID; scoped settlement read | Pinned dependency set, current authoritative revisions, stale or pinned flags, traversal completeness and as-of. No mutation |
| `close.publish_replacement` | Settlement period ID, expected MonthlyCloseRevision, replacement input/manifest refs, the eligible input decision refs published by `input.confirm_publish` for every replaced input, replacement reason and evidence; scoped `close.publish` | Publication rechecks that each bound input decision is still eligible and current, otherwise `confirmation_prerequisite_failed`; it cannot bypass input confirmation. New MonthlyCloseRevision; the prior revision stays immutable. Every registered dependent unissued FinalSettlementSnapshot, invoice cycle and approval is marked stale in the same transaction or by durable intent that blocks issuance until traversal completes |
| `settlement.recompute_aggregate` | Cycle or aggregate scope ID, expected aggregate revision, current dependency set from `settlement.dependency_status`; scoped `settlement.compute` | New complete aggregate revision (for example a quarterly snapshot over every dependent month) with a new digest. Never swaps one month inside an old approved aggregate |
| `settlement.approve_final` | Snapshot ID, expected snapshot digest and revision, rationale; independently granted `settlement.approve_final` within ceiling. The server derives the approval subject (`ApprovalSubject` with purpose and snapshot type per the logical schema) from the stored snapshot; a caller-supplied subject is never authoritative | Final amount approval of that digest only. Stale, superseded or digest-mismatched snapshots return `approval_stale` or `revision_conflict` |

Proposed adjustment states: `draft → computing → validated | blocked → approved | rejected | cancelled → applied`. A change to any parameter, evidence or base result makes an unapplied draft `stale`; it needs a new computation and decision. Proposed snapshot approval states: `awaiting_approval → approved`, and `approved → approval_stale` when a dependency is superseded; an `approval_stale` snapshot is replaced by a newly computed revision, never revived.

At `adjustment.apply` recheck current authority and separation, expected base revision and all bound evidence. Application serializes with issuance and submission pinning. When any target is issued, submitted, potentially dispatched or in `outcome_unknown`, apply returns `invalid_stage` with the governed correction or reconciliation path as next action; unknown dispatch is never proof that a document is unissued. A second target-total request binds the revised current result, so the earlier applied delta is not added twice. Gross targets require an approved deterministic tax/rounding allocation and a feasible exact target, otherwise `policy_incomplete`. Mixed currency or basis is a validation error.

`close.publish_replacement` never edits an issued or externally pinned cycle. An issued, pinned or unknown-outcome dependent receives a linked correction or reconciliation work item; it is not released, reissued or silently repriced. Incomplete or failed dependency traversal blocks issuance of every possibly affected aggregate (`evidence_stale` or `authority_unavailable`) rather than allowing it. Reapproval is a fresh `settlement.approve_final` on the recomputed digest.

B04, B11 (unissued targets only) and S06 originate adjustment drafts and their authors cancel them there with `adjustment.cancel`, B08 shows computation, application and stale dependency results, B09 presents adjustment decisions and final amount approval, and U02 shows the cycle stage blocked by stale dependencies. Cut-off, late-arrival, tax allocation and cumulative correction values remain open policy decisions.

## FX component validation

Source: B0463, B0464, B0467, B0474, B0477.R002; local AR-06. Supplies the command-validation part of HB-15 FA-05 for any FX policy draft or approval command, including the existing B05 approval request. A published or selected raw exchange rate must be greater than zero, with a positive quotation unit, correct pair and direction, approval and effective time. Selling FX markup and conversion fee are separate fields recorded as contract rules, not part of the raw rate. Zero markup and zero fee are valid. A negative markup or fee is accepted or rejected only by that component's approved sign policy under the [Claim/numeric contract](../billing/claim-and-numeric.md); no blanket zero/negative rule applies to them. A raw-rate value that already contains a markup is a validation finding, not a rate.

## Operational notification rules

Source: B1381, B1387, B1388.R001–R005, B1391, B1392; OPSUI03, OPSUI04 (B1396). Supplies the command part of HB-15 FA-13. An operational notification rule revision belongs to the installation or workspace and governs who is told what, when and through which escalation. It is distinct from personal preferences (quiet hours, digest, snooze, personal channel) managed in M05 and from deadline identity managed by `schedule.*`.

| ID | Required body / permission | Outcome |
| --- | --- | --- |
| `notification_rule.draft` | Rule or template ID, expected rule revision when revising, scope (workspace, contract or cycle set, stage and schedule kinds), notification type from B1388 (deadline pre-notice, approval or handoff waiting, closing blocker or failure, overdue or source delay, completion, correction or schedule change), trigger offsets relative to the stage schedule, recipient roles, ordered escalation steps (target role or principal and condition), alternate owner, channel binding refs, template content revision, rationale; scoped `notification.rule.write` | New draft revision; no live effect. Source offsets D-7, D-3, D-1 and same day are an initial template; actual values are profile choices |
| `notification_rule.preview_impact` | Draft ID and expected revision; scoped `notification.rule.read` | Server-built impact: affected contracts, cycles and schedules, recipients resolved under current permissions, steps suppressed by holds or confirmed receipt, unresolved recipients and counts, with as-of. Unauthorized targets appear only as a generic unresolved count. No mutation |
| `notification_rule.approve`, `notification_rule.reject` | Draft ID, exact preview digest, expected revision, rationale; `notification.rule.approve` under the approved role mapping | Immutable decision. Whether the approver must differ from the author is an installation policy decision; when the policy requires separation the author is refused with `self_approval_denied` |
| `notification_rule.apply` | Approved decision ID and digest, expected rule revision; scoped `notification.rule.apply` | Atomically activates the exact approved revision for events created after activation. Events already created keep their rule revision. Deadlines, blockers, holds and approvals are unchanged |

At each dispatch the notification service rechecks the current target state, recipient activity and permission, applicable hold and receipt state. A recipient who is inactive or lacks current permission for the target is not notified; the event records an unresolved-recipient state and proceeds to the rule's alternate owner or next escalation step, never to an unauthorized principal. A collections hold suppresses the reminder kinds it covers. Confirmed receipt suppresses payment reminders for the settled balance; unknown receipt is shown as unknown and is never treated as paid. Read, acknowledgement, owner acceptance and resolution remain separate (B1391); acknowledging a notification never resolves a blocker.

Personal preferences may delay, group or route a user's own non-mandatory notifications within the organization's ceiling. They cannot edit an operational rule, remove an escalation step or replace the rule's alternate owner. A personal alternate chosen in M05 is a personal preference limited to permitted roles; precedence between it and the operational alternate owner is a profile decision. Channel connection or notification receipt grants no billing permission (B1387). U02 presents rule drafting, preview, approval state and application; P02 supplies channel bindings; M05 and U03 remain personal.

## Demo scenario runs

Source: B1286, B1293, B1317, B1318.R001–R008, B1328.R001–R010 (SIM01–SIM10), B1303. Supplies the command part of HB-15 SR-03 for P06.

| ID | Required body / permission | Outcome |
| --- | --- | --- |
| `demo.scenario_run` | Demo environment ID, operating profile `msp`, `standalone_enterprise` or `corporate_group`, purpose `public_sample_view`, `synthetic_settlement` or `failure_reproduction`, scenario ID and revision, fixture revision, seed and virtual clock, expected-result bundle revision (required for `synthetic_settlement` and `failure_reproduction`, not applicable to `public_sample_view`), fault-injection set (only for `failure_reproduction`); scoped `demo.run` | Mock-only job in the isolated demo environment. Rejects real credentials, secret-store links, operational targets and unregistered destinations |
| `demo.compare_expected` | Run ID, expected-result bundle revision; scoped `demo.read` | Amount, state, blocker and permission differences against the independent expected bundle. Not available for `public_sample_view`, whose output is labelled as sample viewing with missing documents and partial months shown as such |
| `demo.reset` | Demo environment ID, expected environment revision, rationale; scoped `demo.reset` | Job resetting only that demo environment's data; rejects operational connections, volumes or any non-demo target |

Each profile uses the same account, contract and permission model with different customer relationships (B1317). A public sample view never produces a validated invoice answer or restored CSP invoice (B1293). A demo pass is recorded separately from CSP, ERP or product connection verification (SIM10). Fixture versions, expected-result approver and training hardware remain open decisions.

## ERP route switching

Source: B1044, B1047, B1048. Registers the commands proposed in [source ERP and issuance obligations](../billing/source-erp-and-issuance-obligations.md) EI-06 (HB-22 follow-up 2). Names are unchanged from that table. HB-23 refines it: the separate approval operation names the designated-owner approval it already requires, the request permission is named `erp_route.assign_manual.request`, and both manual-assignment commands require route state `dispatch_locked` at an expected route revision (B1047). Where they differ, this registry governs.

| ID | Required body / permission | Outcome |
| --- | --- | --- |
| `erp_route.lock_dispatch` | ERP connection and scope, expected route revision, rationale; scoped `erp_route.lock` | Route `dispatch_locked`. New automatic dispatch stops; queued outbox rows are held, not dropped, and stay in the shared submission ledger |
| `erp_route.assign_manual` | ERP connection, expected route revision, submission IDs, each with confirmed not-applied evidence and expected revision, rationale; scoped `erp_route.assign_manual.request` | Allowed only while the route is `dispatch_locked` at the expected route revision, otherwise `invalid_stage` (B1047: lock dispatch first). Pending manual assignment request bound to that exact submission set and digest; no route change yet |
| `erp_route.approve_manual_assignment` | Assignment request ID and digest, expected route revision, expected submission revisions, rationale; designated owner holding `erp_route.assign_manual.approve` | Allowed only while the route is still `dispatch_locked` at the expected revision, otherwise `invalid_stage`. Rechecks each submission's not-applied evidence against current ledger and lookup state; a submission now applied or unknown is excluded and reported. Executes the assignment: listed not-applied submissions move to the manual route; unknown-outcome items receive a review-only file and re-entry submissions that risk duplication are held. The requester is refused with `self_approval_denied` |
| `erp_route.register_manual_result` | Submission ID, ERP document number, date, amount and status, or an ERP result file reference, expected submission revision; scoped `erp_route.register_result` | Reconciled against the original statement and recorded as operator-confirmed, distinct from ERP-confirmed. Differences open reconciliation; approved amounts are never edited by this command |
| `erp_route.sync_after_recovery` | ERP connection, expected route revision; scoped `erp_route.lock` | Job synchronizing manually applied ERP numbers and statuses first; completed transactions are never recreated through the API; duplicate candidates are flagged |
| `erp_route.unlock_dispatch` | ERP connection, expected route revision, rationale; scoped `erp_route.lock` | Allowed only after `sync_after_recovery` finished and no unknown-outcome item remains unassigned, otherwise `invalid_stage`. Held rows for not-applied submissions resume; others stay held for review |

Proposed route states: `api_active → dispatch_locked → recovery_sync → api_active`; per submission `queued | held | manual_pending_approval | manual_assigned | operator_confirmed | erp_confirmed | review_only | outcome_unknown`. Manual handoff is never an unconditional retry path. Where the ERP does not enforce reference uniqueness, submission history, duplicate-candidate checks and result reconciliation compensate; operator re-entry cannot be fully prevented technically. I02 presents lock, assignment, approval, synchronization and unlock; I03 presents review-only files and manual result registration.

## Planned contract cases

All cases in the following table are NOT_RUN, linked to canonical screen/work records by the [remediation trace](../../planning/v1.16/remediation-trace.json). Existing H13, TR, FP, DH, OB, FV and PT cases remain applicable.

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

HB-23 cases are also NOT_RUN. Numbers restate the schema's synthetic example; they are not chosen policies. MS references are the schema's editorial cases.

| ID | Independent expected behavior |
| --- | --- |
| H23-C01 | MS-05: net target 900 against base net 1,000 computes −100 at net basis; independent approval; one application yields a new result revision; replay returns the same revision; the prior final approval is stale and B09 requires `settlement.approve_final` on the new digest |
| H23-C02 | The draft author calling `adjustment.approve` receives `self_approval_denied`; an approver above ceiling receives `permission_denied`; no decision is recorded |
| H23-C03 | A new base run after computation makes `adjustment.apply` fail with `revision_conflict` or `approval_stale`; a new draft computation and decision are required |
| H23-C04 | MS-05: a later target net 850 against revised net 900 adds −50; both applied lines and the original base remain; −100 is not added again |
| H23-C05 | Gross target without an approved tax allocation rule returns `policy_incomplete`; mixed currency or basis returns a validation error |
| H23-C06 | MS-07: apply racing issuance, or against a target in `outcome_unknown`, returns `invalid_stage` with the correction path; no pre-issuance effect is created |
| H23-C07 | MS-09: replacement month published before quarterly issuance marks every dependent unissued quarter snapshot and approval stale; issuance is blocked until `settlement.recompute_aggregate` and a fresh `settlement.approve_final` |
| H23-C08 | MS-09: replacement month after the quarter is issued or pinned unknown leaves the quarter immutable and opens a linked correction or reconciliation item; no release or reissue |
| H23-C09 | Incomplete dependency traversal blocks issuance of every possibly affected aggregate; `settlement.dependency_status` reports traversal incomplete |
| H23-C10 | FA-05: raw rate 0 or negative, wrong pair or zero quotation unit is rejected; zero markup and zero fee are accepted; a negative fee or markup follows its component sign policy; a raw rate containing markup is flagged |
| H23-C11 | Rule revision changing a source example offset: preview lists affected cycles and resolved recipients; no effect before approval and apply; deadlines and blockers unchanged; events created earlier keep their rule revision |
| H23-C12 | Escalation target inactive or without current target permission at dispatch is not notified; the event records unresolved recipient and proceeds to the alternate owner or next step; no protected detail reaches an unauthorized principal |
| H23-C13 | Personal quiet hours delay a user's own non-mandatory notification but cannot remove an operational escalation step; a personal-preference permission cannot call `notification_rule.draft` or `notification_rule.apply` |
| H23-C14 | Active collections hold suppresses covered reminders; confirmed receipt suppresses the payment reminder; unknown receipt is displayed as unknown and is not treated as paid |
| H23-C15 | `notification_rule.apply` with a stale rule revision conflicts; same key and payload returns the same operation; changed payload conflicts |
| H23-C16 | `demo.scenario_run` without operating profile or purpose is blocked; `public_sample_view` has no expected bundle and its output is never labelled an invoice answer; `synthetic_settlement` requires a fixed expected bundle revision |
| H23-C17 | Demo run with a real credential, secret-store link or unregistered destination is blocked; `demo.reset` touches only the named demo environment and rejects any operational target |
| H23-C18 | SIM02: same profile, scenario, fixture, seed and clock reproduce identical sorted statements and amounts; a demo pass is recorded separately from connection verification (SIM10) |
| H23-C19 | `erp_route.unlock_dispatch` with a stale route revision conflicts; same-key replay of `erp_route.lock_dispatch` returns the same operation; a manual result whose amount differs from the original statement opens reconciliation and is recorded operator-confirmed, not ERP-confirmed |
| H23-C20 | A submission ID from another workspace in `erp_route.assign_manual` returns a generic denial without existence disclosure; `erp_route.approve_manual_assignment` by the requester returns `self_approval_denied` |
| H23-C21 | EI-06, B1047: `erp_route.assign_manual` while the route is not `dispatch_locked`, or with a stale route revision, returns `invalid_stage` or `revision_conflict`; a submission whose lookup shows it applied after the request is excluded at `erp_route.approve_manual_assignment` and is not assigned |
| H23-C22 | `close.publish_replacement` referencing an input that is not eligible, or whose decision was superseded before publication, returns `confirmation_prerequisite_failed`; no replacement month is published and no dependent becomes stale |
| H23-C23 | `settlement.approve_final` with a caller-supplied approval subject differing from the stored snapshot's derived subject records the approval only against the server-derived subject and does not persist the supplied value; a digest that differs from the current snapshot returns `revision_conflict` |

No numeric defaults, deployed HTTP paths or completed compatibility claims follow from these proposals. Required permission mappings must be reviewed against actual role ceilings before activation.
