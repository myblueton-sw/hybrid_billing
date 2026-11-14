---
type: ImplementationPlan
title: Approval subjects bound to digests, separated stages, eligible human approvers with limits and execution that re-checks and consumes
description: Defines the first increment of work item W-23, an approval register in which a subject binds a stage, a target, the digests of its inputs, rules, mappings, exchange rates, attribution, ERP draft and result, its amount and an optional bulk manifest; requests and decisions are separate commands; decisions need eligible human approvers other than the requester with an active authority and amount limit; changed digests supersede earlier requests; and execution re-checks digests and authority and consumes an approval once.
status: draft
---

# Scope and ownership

[HB-221](https://linear.app/hybrid-billing/issue/HB-221) is the first increment of work item W-23 (requirement HB-23; the implementation deliverable HB-23-I and the verification HB-23-V). It is built in delivery order under the owner's instructions of 2026-11-12 and 2026-11-14. Base: main `3aee5bd`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the register decides who may release billing amounts (permission boundary). Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-14:

- **HB-23-D.** Charge approval, ERP final amount approval, issuance and delivery are separate. Explicit human approval and separation of duties are kept. A change of input, policy, FX, attribution or ERP draft requires a new approval. The approvers per stage, the amount limits and the trusted external approval method are open.
- **HB-23-I.** An ApprovalSubject binds target, input, rule, mapping and result hashes to the act, the eligible approvers and the limits. Long-running work re-checks the authority and the target hash. The approval request and the approval decision are different commands.
- **HB-23-V.** External execution is blocked after a concurrent change, a withdrawal, a self-approval, an AI-generated approval, or an ERP draft change right after approval. A bulk approval is bound to the digest of a fixed target manifest, with per-customer or per-pool eligibility and partial failures recorded. No global approve button bypasses blocking errors.
- **Other detail contracts.**
  - AR-06 (FX precedence) was delivered by HB-208 (`modules/fx`).
  - AR-07 (adjustment drafts), QA-01 (protected payment and recipient changes) and the ERP recognition cadence (ERPC-01 to ERPC-08) are later increments.
- **Existing code.**
  - `modules/auth` decides grants, including `RequestApproval`, `Approve` and the self-approval denial for an action.
  - Nothing binds an approval to the digests of what was approved, or consumes it at execution.

# Decision

One Go package, `modules/approvals`. It uses the standard library and the primitives `decimal` and `exact`. A registry with an injected clock and one mutex holds the data per workspace. Nothing calls it and no entry point is wired.

The registry keeps its own administrators and approver authorities. Linking them to `modules/auth` grants is a later integration; the caller must keep the two consistent until then (stated obligation). Kinds map as follows: `human` ↔ `auth.Human`, `assistant`/`system` ↔ `auth.Service`.

Strings are identifiers, three-letter currency codes, dates, decimal texts read by the registry's profile, 64-character lowercase hexadecimal digests, or values of closed lists. There is no free text and no floating point.

Errors are `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (with a named reason where one is listed) and `ErrLimit`. Checks run in the order form, lookup, identical repeat, rule, room, except where an order is stated below.

## Actors

Every call names an `Actor{ID, Kind}`, where the kind is `human`, `assistant` or `system`. The caller authenticates the actor and its kind (stated obligation).

Only a human may open a workspace, administer it, decide or report as an administrator (`ai_not_allowed` for others). Any kind may request an approval and execute.

## Administration

- `Open(ws, admins, actor)` opens a workspace once. It needs a human actor and two to `Limits.Admins` distinct administrator identifiers, which are humans by caller obligation. This is an installation step done under the caller's own controls (stated obligation).
- Policies, authority grants and administrator changes are proposals. `Propose(ws, Proposal{ID, Kind, Policy, Authority, Admin}, actor)` needs an administrator. The kind is `policy`, `grant`, `add_admin` or `remove_admin`.
- `Confirm(ws, id, actor)` needs a different administrator. It re-checks that:
  - the proposer is still an administrator;
  - a policy proposal's revision is still the next one;
  - a grantee is neither the proposer nor the confirmer;
  - a removal leaves at least two administrators.
- A confirmed proposal takes effect at once. A proposal that fails these re-checks stays unconfirmed. `Limits.Proposals` bounds proposals per workspace.
- `Revoke(ws, authority, actor)` ends an authority at once, by any administrator or by its grantee. Revocation is the safe direction.
- Residual risk, recorded: two colluding administrators.

## Stages and policies

- The stages are `charge_approval`, `erp_final_approval`, `issue` and `deliver`. An approval for one stage never authorizes another.
- Prerequisites between stages are not enforced here (not covered).
- `Policy{Stage, Revision, Approvals, Required, Validity}`:
  - `Revision` must be the stage's current revision + 1;
  - `Approvals` is the number of distinct approvers, from 1 to 3;
  - `Required` is a non-empty set of digest kinds, which must contain `result`, from `input`, `rule`, `mapping`, `fx`, `attribution`, `erp_draft`, `result`;
  - `Validity` is from 1 hour to 90 days.
- Confirming a new revision supersedes every pending or approved request of that stage, including the unexecuted items of a partly executed bulk request.

## Authorities

`Authority{ID, Approver, Stage, Currency, Limit, From, To}` lets a human approver approve subjects of a stage while their covered amount, in absolute value, is at most the limit in that currency, on the dates From to To. An authority is never edited; a new limit is a new grant.

## Requests

`Request(ws, Subject{ID, Stage, Target, Author, Digests, Amount, Currency, Entries}, actor)` opens a pending request under the stage's current policy revision.

- `Author` is the accountable preparer, a human by caller obligation.
- `Digests` holds exactly the required kinds.
- `Entries` is optional. Each entry is `{Item, Scope, Amount, Digest}`, there are 1 to `Limits.Items` of them, and items are distinct. With entries, `Amount` equals their sum.
- Amounts may be negative.
- The subject's digest covers every field and the policy revision.
- A new request for the same stage and target supersedes every pending or approved earlier request for that stage and target, including the unexecuted items of a partly executed one.

## Decisions

`Decide(ws, id, Decision{Verdict, Reason, Exclude}, actor)`:

1. The actor must be human (`ai_not_allowed`).
2. The request must be pending (`not_pending`).
3. The actor must be neither the author nor the requester (`self_approval`), and must not have decided this request already (`ErrConflict` if the content differs).
4. For `approve`, the actor must hold an authority for the stage and currency, active at now, covering the absolute sum of the entries left after the union of all exclusions so far and this decision's exclusions (`no_authority`).
5. Leaving no entry is refused (`nothing_left`).

Each exclusion is `{Item, Reason}`, with reason codes `not_eligible`, `blocked_error`, `customer_hold` and `data_missing`. `reject` needs a reason code from `incorrect_amount`, `missing_evidence`, `policy_violation` and `other`, and ends the request. When the number of distinct approvers reaches the policy count, the request becomes `approved` at that time. Decisions per request are bounded by `Limits.Decisions`.

`Withdraw(ws, id, actor)` by the requester works while the request is pending or approved. It ends any unexecuted items.

## Execution

`Execute(ws, Execution{ID, Request, Stage, Target, Item, Digests, ItemDigest, Amount, Currency}, actor)` is the last gate before an external act, and the act must use exactly what was checked. The caller computes `Digests` and `ItemDigest` from the live target at that moment (stated obligation). The execution identifier must be the idempotency key of the external call (stated obligation).

Checks, in this order:

1. The request is pending or rejected (`not_approved`).
2. It was superseded (`superseded`) or withdrawn (`withdrawn`).
3. The policy revision is no longer current (`policy_changed`).
4. More than `Validity` has passed since approval (`expired`).
5. The stage, target, digests and currency differ from the subject, or the amount differs from the subject's amount or, for an item, from the entry's amount and digest (`subject_changed`).
6. An approver's authority is no longer active, or no longer covers the covered sum (`authority_lost`).
7. For a bulk subject, the item is missing from the manifest, excluded, or ended (`item_ineligible`).
8. The stage and key were already executed by another execution (`already_executed`). The key is the target, or the item for a bulk subject. Targets and items share one identifier space, the identifier of the thing acted on (stated obligation).

A success records the execution. Execution identifiers are unique per workspace, and `Limits.Executions` bounds them.

- A repeat with the same content returns the stored record when it was already reported `succeeded`. Otherwise checks 1 to 7 run again, and the record is returned only if they pass.
- A repeat with other content is `ErrConflict`.
- After a failed act, only a replay of the same identifier can proceed. A key stays consumed even if the replay later fails its checks, for example after expiry. This is a known operating consequence.

`Report(ws, execution, Outcome{Result, Reason}, actor)` is written once by the executing actor. `Result` is `succeeded` or `failed`, with a reason code for a failure from `external_error`, `timeout` and `refused_downstream`. It records partial bulk failures item by item.

## Limits

`Limits{Workspaces, Admins, Proposals, Authorities, Requests, Items, Decisions, Executions}`. Each is positive and capped. `Items` counts per request, `Decisions` per request, and the others per workspace.

# Provisional choices awaiting the owner

- Approver count per stage, the digest kinds required, the validity, and the amount limits per approver, currency and dates.
- Assistants and systems may request and execute, but never administer or decide.
- Two administrators for every administrative change, with revocation by one.
- Withdrawal by the requester is allowed until execution.
- No re-run of an executed stage and key.
- The registry's administrators and authorities are separate from `modules/auth` until integration.

# Verification

- Literal tests for every rule, including the HB-23-V scenarios:
  - a concurrent new request between approval and execution;
  - withdrawal;
  - self-approval by the requester and by the author;
  - an assistant's decision;
  - an ERP draft digest change with no new request;
  - a policy change;
  - revoked authority;
  - expiry;
  - a bulk manifest with union exclusions and partial execution and failure;
  - replays before and after reports;
  - a single administrator acting alone;
  - a grantee confirming its own grant.
- The check order is pinned by a test.
- A property test runs against an independent model over random call sequences.
- Race detector, count=10, a 386 build, gofmt, vet, the whole repository test run, a mutation run and a source scan.
- Not covered: stage prerequisites, AR-07, QA-01, ERPC, the link to `modules/auth`, persistence, screens and external approval methods.

# Review history

The first design review returned FAIL with twelve findings. They covered administration bypass, re-approval on policy change, replay re-checks, consumption across requests, per-item bulk binding, negative amounts, author exclusion, live digests, the last gate and reports, lifecycle edges, stage order, and check order and limits.

The re-review returned PASS with seven low and medium findings:

- supersede of partly executed bulk requests;
- union exclusions leaving nothing;
- pending administrative changes;
- replay after success;
- check-order wording;
- stale text;
- the author field.

All are folded into the text above, which replaces the earlier body.

# Change review resolution

The independent change review returned PASS with findings F1 to F6 and four hypotheses. Each is resolved as follows:

- F1, stated: the covered amount of a bulk subject is the gross sum of the absolute amounts of the entries that remain. Credits therefore never hide large charges from an approver's limit. For a single subject, the covered amount is |amount|.
- F2, changed: a report of `failed` may be followed once by `succeeded` from the same actor, for example after a successful replay. The earlier outcome is kept (`Earlier`). Any other second report is `ErrConflict`. This is tested.
- F3, changed: `Revoke` requires a human actor (`ai_not_allowed`). Tested for an assistant administrator and a system grantee.
- F4, changed: `Limits.Decisions` is removed. Decisions per request are bounded by construction: at most three distinct approvers plus one reject.
- F5, stated: in `Decide`, the identical-repeat and conflicting-repeat check runs before the actor-kind and state rules. A repeat of an earlier decision on a request that has since ended therefore returns the stored view, or `ErrConflict` when the content differs.
- F6, closed: tests now cover:
  - a new request superseding a partly executed bulk request and blocking its remaining items;
  - `policy_changed` reported before `expired`;
  - a non-human `Revoke`;
  - the property model's assistant withdrawal, which is now exercised directly.

Hypotheses recorded:

- H1 and H2: targets and items share one identifier space by caller obligation, and `already_executed` prevents a double act.
- H3: any kind of actor may request, and repeated requests can keep superseding approvals. This is the safe direction, and rate or role limits are left to the owner.
- H4: any human other than the author and the requester may reject without an authority. This is the safe direction and is listed as provisional.

These fixes were not reviewed again.
