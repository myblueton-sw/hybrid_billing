---
type: ImplementationPlan
title: Protected payment, recipient, tax identifier and template changes with existing-channel verification, separate approval and forward-only effective dates
description: Defines the QA-01 increment of work item W-23, a register in which a change to a payment account, a document recipient, a tax identifier or a document template is a request over protected references that cannot take effect until a human other than the requester confirms it through a channel registered before the request, separate humans approve the exact request digest, and the change applies only to documents dated on or after its effective time.
status: draft
---

# Scope and ownership

[HB-223](https://linear.app/hybrid-billing/issue/HB-223) continues work item W-23 (requirements HB-23, HB-28 and HB-32; detail contract QA-01; acceptance AUD-QA-01) after HB-222 (`modules/adjustments`). It is built in delivery order under the owner's instructions of 2026-11-12 and 2026-11-14. Base: main `ee60476`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the register controls where money is paid and who receives billing documents, which is a permission and tenant-boundary concern. Machine: mac_mini.

This increment builds only QA-01. The ERP recognition cadence (ERPC-01 to ERPC-08), the integration layer joining registers and `modules/auth`, persistence and screens O03, I01 and B09 are later increments.

# Observed baseline

- QA-01 requires the following:
  - changes to a payment account, a document recipient, a tax identifier or a template need separate verification, approval and a validity period;
  - the protected references before and after, the verifier and approver, and evidence of confirmation through an existing registered channel are kept;
  - an instruction from a new email alone never applies an account;
  - an LLM never generates or replaces payment information;
  - the control is mandatory even though the owners and concrete periods are not yet decided.
- AUD-QA-01 checks that an account change backed only by a new email and an LLM-generated account are refused, and that a change reaches documents only from its effective date after existing-channel confirmation and separate approval.
- `docs/planning/v1.16/domain-contracts.md` lists QA-01 under Access as "existing-channel payment-detail verification and no LLM invention". `docs/planning/v1.16/ui-design.md` maps O03, I01 and B09 to change review, existing-channel confirmation, separate approval and validity.
- No module records protected-field changes. `modules/approvals` has the stages `charge_approval`, `erp_final_approval`, `issue` and `deliver`; none fits a payee change, and adding one would change its public contract.
- The newer registers (`modules/approvals`, `modules/adjustments`) import only `packages/primitives` in non-test sources. This register imports nothing outside the standard library.

# Decision


A new package `modules/protected` holds one registry per process. It has an injected clock (the highest reading, in UTC, rounded to zero) and one mutex. Workspaces are isolated. Change and channel identifiers are scoped per workspace, and a change of another workspace is `ErrNotFound`. Errors are `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (a `Refusal{Reason}` that unwraps to it) and `ErrLimit`. Checks run in this order: form, lookup, repeat or conflict, rule, room. An identical repeat of a write by the same human actor returns the current view in every state; an assistant or system actor reusing a human's identifier gets no view.

The register holds only protected references, never the protected values. A `Ref` is a vault reference identifier and a 64-hex fingerprint of the value. The vault, the value format checks and the rendering of values are outside this register.

## Actors

Every actor is `Actor{ID, Kind}`, where the kind is human, assistant or system. Every write in this register needs a human. An assistant gets `ai_not_allowed` and a system gets `not_human`. Reads have no actor; permission checks belong to the integration layer with `modules/auth`.

## Rules under dual control

Rules are:

- `Approvals`: the number of distinct approvers, 1 to 3;
- `Validity`: how long a confirmation stays usable for approval, 1 hour to 30 days;
- `MaxLead`: the latest effective time allowed after the request, 1 hour to 366 days;
- `ChannelAge`: how long a channel must have been admitted before the request, 1 day to 90 days.

`Propose(workspace, Rules, actor)` records a proposal; every proposal takes a new revision number, and the first proposal creates the workspace. An identical repeat by the proposer of the pending or active revision returns it. Refusals, in order: `ai_not_allowed` or `not_human`; `no_change` when the rules equal the active rules; `proposal_pending` when another human's proposal is pending (a proposer may replace their own). Creating a workspace beyond the limit, or a workspace's 1001st proposal, is `ErrLimit`.

`Confirm(workspace, revision, actor)` activates the pending proposal. A missing workspace is `ErrNotFound`; a repeat by the confirmer of the active revision returns it. Refusals, in order: `ai_not_allowed` or `not_human`; `stale_proposal` when nothing is pending or the revision is not the pending one, so a replaced proposal can never be confirmed under an older number; `self_approval` for the proposer. `Discard(workspace, revision, actor)` drops the pending proposal and records who discarded it; any human may discard, so an abandoned proposal cannot lock the rules. A missing workspace is `ErrNotFound`; a repeat by the discarder of the last discarded revision returns it. Refusals, in order: `ai_not_allowed` or `not_human`; `stale_proposal` when the revision is not pending.

Until a first confirmation the workspace has no rules. `RulesOf(workspace)` returns the active and pending revisions and the history of every proposal in revision order, each with its proposer, confirmer or discarder and their times.

A request copies the active rules and revision. Later rule changes do not affect it.

## Channels under dual control

`Register(workspace, Channel{ID, Subject, Kind, Ref, Evidence}, actor)` records a contact channel of a subject (a customer or a vendor account) as pending, with an evidence reference for why the address is trusted. The kind is `email`, `phone`, `portal` or `postal`. A missing workspace is `ErrNotFound`; an identical repeat by the registrar returns the view and any other registration under that identifier is `ErrConflict`; then `ai_not_allowed` or `not_human`; then `ErrLimit`.

`Admit(workspace, channel, evidence, actor)` admits a pending channel and records the admission evidence. A missing channel is `ErrNotFound`; a repeat by the admitter returns the view and an admission by another human of an admitted channel is `ErrConflict`. Refusals, in order: `ai_not_allowed` or `not_human`; `channel_retired`; `self_approval` for the registrar. Only an admitted channel can confirm a change, and its age is counted from admission.

`Retire(workspace, channel, actor)` ends a pending or admitted channel at the current time; one human is enough because retiring only removes a confirmation path. A missing channel is `ErrNotFound`; a repeat by the retirer returns the view and retiring a channel retired by someone else is `ErrConflict`; then `ai_not_allowed` or `not_human`.

The registrar and the admitter of a channel are its custodians. Neither may request a change confirmed through it, confirm through it, or approve a change confirmed through it.

## Requests

`Request(workspace, Change, actor)` records a change with:

- an identifier, the subject and the field: `payment_account`, `document_recipient`, `tax_identifier` or `document_template`;
- `After`, the new protected reference;
- `Provenance`: `staff_entry`, `customer_document`, `customer_portal` or `assistant_generated`;
- `Instruction`: the kind (`email`, `phone`, `portal`, `postal` or `in_person`), the registered channel it arrived through or empty, the fingerprint of the sender's address (required for every kind except `in_person`, where it is empty), and an evidence reference;
- `Effective`, the time from which documents use the new reference.

Refusals, in order:

- `ai_not_allowed` or `not_human` for a non-human actor;
- `llm_generated` when the provenance is `assistant_generated`;
- `no_rules` when the workspace has no active rules;
- `unknown_channel` when the instruction names a channel that does not exist in this workspace, belongs to another subject, has another kind or another fingerprint than the sender;
- `undeclared_channel` when the instruction names no channel but the sender's fingerprint equals the fingerprint of any channel of the subject, pending, admitted or retired;
- `change_pending` when another change of the same subject and field is requested or confirmed and has not lapsed;
- `effective_out_of_range` when `Effective` is not after now or is after now plus `MaxLead`;
- `out_of_order` when `Effective` is not after the effective time of every scheduled change of the same subject and field;
- `no_change` when `After` equals the reference that would be in force just before `Effective`.

`Before` is the reference of the latest scheduled change of the same subject and field. The first change of a subject and field is its baseline: `Before` is empty, and the same confirmation and approval apply, so onboarding needs admitted channels first.

The request digest (label `protected_change_v1`) covers the workspace, identifier, subject, field, before, after, provenance, instruction, effective time, requester, request time, rules revision and rules. An identical repeat by the same requester returns the view; anything else under that identifier is `ErrConflict`.

## Verification

`Verify(workspace, change, Verification{Channel, Evidence, Outcome, Digest}, actor)` records the existing-channel confirmation. The outcome is `confirmed` or `denied`. Refusals, in order:

- `ai_not_allowed` or `not_human`;
- `lapsed` when the change has lapsed, `not_pending` unless the change is requested;
- `digest_changed` unless the digest equals the request digest;
- `self_verification` when the verifier is the requester;
- `unknown_channel` unless the channel exists in this workspace and belongs to the change's subject;
- `channel_not_admitted` when it is not admitted;
- `same_channel` when it is the instruction's channel or its fingerprint equals the sender's;
- `channel_retired` when it is retired;
- `channel_too_new` unless it was admitted at or before the request time minus `ChannelAge`;
- `self_channel` when the requester or the verifier is a custodian of the channel.

`denied` ends the change as rejected. `confirmed` moves it to confirmed and records the time and a confirmation digest (label `protected_confirmation_v1`) over the request digest, verifier, channel, evidence, outcome and time. A new email alone therefore never applies an account: the confirmation must come through an older admitted channel that is not the sender's.

## Approval

`Approve(workspace, change, Approval{Verdict, Reason, Digest}, actor)` records one decision against the confirmation digest, so an approver is bound to the confirmation they reviewed. The verdict is `approve` or `reject`; a rejection needs a reason (`incorrect_details`, `missing_evidence`, `suspected_fraud` or `other`). Refusals, in order:

- `ai_not_allowed` or `not_human`;
- `lapsed` when the change has lapsed, `not_confirmed` when it is requested, `not_pending` otherwise unless it is confirmed;
- `digest_changed` unless the digest equals the confirmation digest;
- `channel_retired` when the confirming channel has been retired since the confirmation;
- `self_approval` when the approver is the requester, the verifier or a custodian of the confirming channel;
- `already_decided` when the approver has approved this change.

A rejection ends the change as rejected. When the number of approvals reaches `Approvals`, the change becomes scheduled and the time is recorded.

## Lapse

A requested change lapses after its effective time, and a confirmed change lapses after its effective time or after its confirmation time plus `Validity`, whichever comes first. Each call settles only the changes it touches before its checks (the target change, and for `Request`, `Withdraw` and `Trail` every change of the same subject and field): the state becomes `lapsed`, with the deadline as its end time and no actor. A lapsed change does not block a new request. An approval therefore never applies a change to a document dated before the approval.

## Withdrawal

`Withdraw(workspace, change, actor)` ends a requested, confirmed or scheduled change as withdrawn. Any human may withdraw; the integration layer narrows who. A scheduled change can be withdrawn only before its effective time (`already_effective`) and only when no later change of the same subject and field is requested, confirmed or scheduled (`has_successor`), because the later change's `Before` depends on it. Other states are `not_pending`.

## Reads

- `Of(workspace, change)` returns the view: the change, the state, `Before`, the requester and request time, the rules and revision, the confirmation (verifier, channel, evidence, outcome, time, digest), the decisions (approver, verdict, reason, time), the scheduled time, who ended it and when, and the request digest.
- `Trail(workspace, subject, field)` returns the views of every change of that subject and field in request order.
- `Resolve(workspace, subject, field, at)` returns the change identifier, reference and effective time of the latest scheduled change whose effective time is at or before `at`. None is `ErrNotFound`. Documents dated before an effective time keep the earlier reference.
- `ChannelOf(workspace, channel)` returns the channel with its custodians and times.

## Limits

`Limits{Workspaces ≤ 100000, Channels ≤ 1000000, Changes ≤ 1000000}`. Channels and changes are counted per workspace. Exceeding a limit is `ErrLimit` and changes nothing.

# Tests

- AUD-QA-01 scenarios: an account change from a new email with no confirmation is never resolvable; confirmation through the instruction's channel or sender, a retired, unadmitted or too-new channel, or a channel whose custodian is the requester, verifier or approver is refused; an `assistant_generated` provenance and assistant actors are refused; after confirmation and separate approval, `Resolve` returns the old reference before `Effective` and the new one from `Effective`.
- Dual control of rules and channels, every refusal reason, every form error and limit, idempotent repeats and conflicts, workspace isolation, the rules snapshot, lapse, withdrawal rules, and digest coverage of each field.
- A property test against an independent model of the state machine and of `Resolve` over random operation sequences.
- `-race -count=10`, a 386 build, gofmt, vet, the repository race run, coverage and mutation testing.

# Provisional choices

- The rule ranges, the field, provenance and channel kinds, the rejection reasons and requiring humans for every write.
- `Validity` is read as how long a confirmation stays usable for approval. QA-01's validity period could also mean how long evidence or an approved change stays valid; that reading is not built.
- A request snapshots the active rules.
- The separation sets for requester, verifier, approvers and channel custodians.
- Provenance is declared by the caller; the integration layer must derive it from the system origin, never from free user input, because a human could re-key generated output.
- Effective times are instants. The integration layer converts document dates and stores the resolved change identifier on each document, because a later scheduling can change what a date resolves to.
- Any human may withdraw.
- The sender fingerprint is declared by the caller; the integration layer must derive it from message metadata, not from what the requester types. The separate verifier is the backstop.
- A retired confirming channel blocks further approvals but does not end the change; it lapses or is withdrawn.
- Only the proposer of a pending rules proposal may replace it.
- A proposer retrying a proposal that someone else discarded creates a new revision.
- A change keeps its rules and revision; who proposed and confirmed that revision is read from the rules history.

# Review history

- Design review 1 (independent reviewer, read-only): CONDITIONAL PASS with 3 high, 6 medium and 4 low findings. Folded in: dual control for channels (`Admit`, custodians excluded from every role, `ChannelAge` at least 1 day) and for rules (`Propose`/`Confirm`); the first change as an explicit baseline; the sender fingerprint with `undeclared_channel` and the fingerprint test in `same_channel`; identical repeats return the view in every state; `unknown_channel` for missing, foreign and other-subject channels and `ErrNotFound` for `Register` before a workspace exists; approvals bound to a confirmation digest; lapse instead of blocked confirmed changes; `Effective` strictly after now; provisional notes on validity, provenance, document snapshots and withdrawal.
- Design review 2 (same reviewer): CONDITIONAL PASS; all 13 earlier findings resolved, 3 medium and 4 low new findings. Folded in: ordered refusal lists for `Propose`, `Confirm`, `Register`, `Admit` and `Retire`; a new revision number for every proposal; settlement limited to the touched changes; sender fingerprint form rules and matching against channels in any state; evidence on registration and admission; `channel_retired` in `Approve`; `proposal_pending` against replacing another human's proposal.
- Design review 3 (same reviewer): PASS with one optional low finding, applied: `Discard` lets any human drop a pending proposal.

# Change review resolution

Change review 1 (a separate independent reviewer, read-only): PASS with 2 medium and 7 low findings. All were addressed:

1. The no-comments guard parsed imports only; it now parses whole files.
2. Refusal order between adjacent refusals was not asserted; `orders_test.go` now covers each adjacent pair in `Request`, `Verify` and `Approve`.
3. Identical repeats compared only the actor identifier; they now also require a human actor, and the plan says so.
4. A proposer's retry after confirmation returned `no_change`; it now returns the active revision. A retry after a discard is recorded as provisional.
5. Discard and earlier revisions were not readable; `RulesOf` now returns the full proposal history.
6. `Withdraw` settled the subject and field only for scheduled changes; it now settles them first in every state.
7. The property model now includes system actors, wrong digests, denied confirmations and checks of `EndedBy` and `Scheduled`, over 600 seeds. Channel admission and age remain covered by example tests.
8. Added tests for the `channel_too_new` boundary, an instruction channel of another subject, a retired channel as sender and concurrent calls.
9. The unused loop index was removed.

Change review 2 (same reviewer): PASS; all 9 findings resolved. One new low finding, applied: proposals are capped at 1,000 per workspace (`ErrLimit`), so one human cannot grow the rules history without bound.
