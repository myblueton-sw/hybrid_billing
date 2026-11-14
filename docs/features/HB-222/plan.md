---
type: ImplementationPlan
title: Adjustment drafts over immutable sources with explicit difference lines, re-validation and release only under the approved digest
description: Defines the AR-07 increment of work item W-23, an adjustment register in which quantity corrections, fixed amounts and target totals are recorded as a draft over a frozen source snapshot; every entry becomes an explicit difference line and never overwrites a source line; tax, allocation and final amounts are re-validated for each revision; and a release needs a human approver separate from the authors and the digest of the current revision, producing a new run before issue or a referenced correction after issue.
status: draft
---

# Scope and ownership

[HB-222](https://linear.app/hybrid-billing/issue/HB-222) continues work item W-23 (requirement HB-23, detail contract AR-07, acceptance AUD-AR-07) after HB-221 (`modules/approvals`). It is built in delivery order under the owner's instructions of 2026-11-12 and 2026-11-14. Base: main `8c9168e`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the register changes billing amounts. Machine: mac_mini.

This increment builds only AR-07. QA-01 (protected payment and recipient changes) and the ERP recognition cadence (ERPC-01 to ERPC-08) are later increments.

# Observed baseline

- AR-07 requires the following:
  - quantity corrections, fixed amounts and target totals are an AdjustmentDraft over unchanged sources;
  - the draft shows target minus current as explicit adjustment lines;
  - tax, allocation and final amounts are re-validated and re-approved;
  - reason, evidence, period, author and approver are kept, and source errors stay visible;
  - a retroactive change before issue is a new run, and a change after issue is a correction that references the original.
- AUD-AR-07 checks that a target total never overwrites a source or charge line but becomes an explainable difference line. It also checks that a changed draft is never submitted under an earlier approval.
- `modules/rating` and `modules/marketplace` produce charges. `modules/approvals` binds approvals to digests and gates execution. `modules/cases` records billing findings. No module records manual adjustments.
- The sibling registers from HB-216 onwards (`modules/approvals`, `modules/components`, `modules/receivables`) scan their own non-test sources and import only `packages/primitives`. Some older modules import others: `modules/gate` imports `modules/auth` and `modules/identity`, and `modules/rating` imports `modules/allocation`. No package imports `modules/approvals` today.
- This register follows the newer rule. Its non-test sources import only `packages/primitives`. It takes source lines, finding identifiers and approval references as data, and exposes digests for `modules/approvals` to bind. Its test files may import `modules/approvals`, and the integration test lives there. The production integration layer is a later increment.

# Decision

A new package `modules/adjustments` holds one registry per process. It has an injected clock (the highest reading, in UTC, rounded to zero) and one mutex. Workspaces are isolated: a lookup of another workspace's draft is `ErrNotFound`. Errors are `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (a `Refusal{Reason}` that unwraps to it) and `ErrLimit`. Checks run in this order: form, lookup, repeat or conflict, rule, room.

## Drafts and sources

`Open(workspace, Draft, actor)` records a draft with the following fields:

- an identifier, a customer, a contract and a period (`From` ≤ `To`, both dates);
- a base:
  - the run identifier and the run digest (64 lowercase hex);
  - `Issued`, and the issued document identifier, which is required exactly when `Issued` is true;
  - the currency, the amount scale (0 to 8) and the rounding mode, one of `toward_zero`, `away_from_zero`, `floor`, `ceiling`, `half_up` and `half_even`;
- the source lines. Each line has:
  - an identifier, unique in the draft;
  - the charge line it reproduces;
  - a quantity (zero or more);
  - an optional unit price (zero or more);
  - a net amount on the amount scale (any sign).

All amounts in the register are net of tax. At least one line is required. The registry copies the lines and computes a source digest over them. Neither the lines nor the base can change after opening.

The lines are caller data. The integration layer must check that they reproduce the charge lines of the run named by the run digest before it opens a draft. The register cannot verify this itself, because modules do not import each other.

A second `Open` with the same identifier, the same content and the same opener returns the first view. Anything else under that identifier is `ErrConflict`.

Every actor is `Actor{ID, Kind}`, where the kind is human, assistant or system. Who may do what:

- Assistants and systems may open drafts and add or remove entries.
- Only humans and systems may validate, release or withdraw; an assistant gets `ai_not_allowed`.
- Only a human may approve.

## Entries

`Add(workspace, draft, revision, Entry, actor)` needs the current revision. Each entry has the following fields:

- an identifier, unique in the draft for its lifetime, including after removal;
- a kind and a scope: `Line` names a source line, and an empty `Line` means the whole draft. Line identifiers cannot be empty, so the two never collide;
- the target quantity (`quantity`) or the amount (`fixed` and `target_total`);
- a reason, and the evidence;
- an optional finding, and a period inside the draft period.

The kinds are:

- `quantity`: needs a line scope whose line has a unit price. The target quantity is zero or more.
  - The current quantity is the source quantity plus the quantity differences of earlier active entries on that line.
  - The difference amount is round(target × price) − round(current × price), with the amount scale and the base mode. This telescopes, so the same final quantity gives the same amount whatever the steps.
  - The source amount may differ from round(source quantity × price), for example after discounts. That gap stays in the source line, unchanged.
- `fixed`: a non-zero signed amount on the amount scale. The scope is one line or the whole draft.
- `target_total`: a signed target amount on the amount scale. The scope is one line or the whole draft.
  - The current amount of a line is its source amount plus the differences of earlier active entries on that line.
  - The current amount of the whole draft is the sum of all source amounts and all earlier active differences.
  - The difference is target − current.

Further rules:

- Entries apply in the order they were added. After every change the whole draft is recomputed from the source lines, so a later target still reaches its target when an earlier entry is removed.
- A line's adjusted amount includes only the differences scoped to that line. Differences scoped to the whole draft appear as their own difference lines and in the totals.
- Reasons are `source_error`, `contract_term`, `goodwill`, `rounding`, `measurement` and `other`. A `source_error` entry needs a finding identifier, and the view lists every finding of its active entries, so a source error is never absorbed silently. Other reasons may carry a finding.
- Evidence is one or more identifiers, without duplicates, within the limit.

`Add` runs these checks in order:

1. form;
2. lookup of the draft, then the amount scale of a fixed or target amount (`ErrInvalid`);
3. repeat or conflict:
   - an entry with the same identifier, the same content and the same author, sent with the revision it was added at, returns the current view;
   - any other use of a known identifier (active or removed) is `ErrConflict`;
4. lookup of the scoped line (`ErrNotFound`);
5. rules:
   - `released` or `withdrawn`;
   - `stale_revision`;
   - `outside_period`;
   - `no_unit_price`;
6. room (`ErrLimit` for evidence or entries);
7. arithmetic:
   - `ErrLimit` when a value leaves the decimal profile;
   - `zero_difference` when the new entry's difference is zero;
   - `negative_total` when the adjusted total would be below zero.

`Remove(workspace, draft, revision, entry, actor)` deactivates an entry. Its checks:

- form, including the actor;
- lookup of the draft, then of the entry (`ErrNotFound`);
- repeat: the same entry, the same actor and the revision it was removed at returns the current view; any other removal of a removed entry is `entry_removed`;
- `released` or `withdrawn`, then `stale_revision`.

The entry stays in the view with who removed it and when. A removal may leave a zero difference or a negative total; validation and release then refuse.

Both commands increase the revision by one.

## Difference lines and digests

The view shows the following:

- the source lines unchanged;
- one difference line per active entry, carrying the entry, kind, scope and its charge line, the current value, the target value, the quantity difference (for `quantity`) and the amount difference;
- the source total, the adjustment total (the sum of the differences) and the adjusted total, with each line's adjusted amount;
- the gross change: the sum of the absolute differences, so a +X/−X pair is not hidden by netting to zero;
- the revision and the state: `open`, `released` or `withdrawn`.

All amounts are exact decimals with the fewest digits. A result outside the decimal profile is `ErrLimit`, and nothing is recorded.

The draft digest is a SHA-256 over length-prefixed fields, under the label `adjustment_draft_v1`. It covers:

- the workspace;
- the draft identifier, customer, contract and period;
- the source digest (label `adjustment_source_v1`), which covers the base and every line field;
- the revision and the number of active entries;
- for each active entry in order:
  - the identifier, kind, scope, target, amount, reason, finding, period, author identifier and kind, and the evidence;
  - the computed current, target, quantity difference and amount difference;
- the source, adjustment, adjusted and gross totals.

## Re-validation

`Validate(workspace, draft, Check, actor)` records that tax, allocation and the final amount were recomputed for one revision. A check carries:

- the revision and the draft digest;
- the tax result digest and the allocation result digest (64 lowercase hex each);
- the recomputed tax amount and the final amount, both signed and on the amount scale. A final amount may be negative when a credit outweighs the net, for example net 1 with tax −20.

Checks in order:

1. form;
2. lookup, then the amount scale of the tax and final amounts (`ErrInvalid`);
3. repeat: the same check by the same actor for the same revision returns the view; a different check for that revision is `ErrConflict`;
4. `ai_not_allowed`;
5. `released` or `withdrawn`;
6. `stale_revision`, then `digest_changed`;
7. `nothing_to_adjust`, `zero_difference` and `negative_total`;
8. `final_mismatch` unless final = adjusted total + tax, exactly.

A validation belongs to one revision only. Any later change leaves the draft unvalidated until a new check.

After a check, the view carries the release subject digest, under the label `adjustment_release_v1`. It covers:

- the draft digest;
- the base: run, run digest, issued state and document;
- the check: tax digest, allocation digest, tax amount and final amount;
- the gross change and the currency.

The release subject digest is what an approval binds. It exists only after a validation, so approval always follows re-validation.

## Approval binding

The register does not import `modules/approvals`. The integration layer joins the two:

1. It requests an approval subject with these fields:
   - the stage `charge_approval`;
   - the target `adjustment:<draft>:<revision>`. It never collides with the run's own approval, and each revision gets its own target, so a draft refused after `Execute` can be changed, validated and approved again. The register's `already_released` and `base_consumed` still allow only one release;
   - the digest kind `result`, whose value is the release subject digest;
   - the gross change as the amount, in the draft currency.

   For a new revision, the integration layer first withdraws the unexecuted requests of earlier revisions. Their targets differ, so the approvals register does not supersede them by itself.

2. The approvals register checks eligible human approvers and limits.
3. Before `Execute`, the integration layer calls `Ready(workspace, draft, Release, actor)`. It runs every `Release` check and records nothing, so an approval is not consumed for a release that would be refused.
4. `Execute` re-checks the live digest and consumes the approval once.
5. The integration layer then calls `Release`. It passes the execution identifier as the approval reference, and as approvers exactly the deciding actors of the approval request. The register cannot verify the approvers itself, so this is a caller duty.
6. If `Release` still refuses after `Execute` (a race between steps 3 and 5), the integration layer reports the execution as failed with `refused_downstream`. A new approval on the new revision's target is needed.

## Release

`Release(workspace, draft, Release, actor)` carries:

- a request identifier, unique per workspace;
- the revision and the release subject digest;
- the full base as it stands now: run, run digest, issued state, document, currency, amount scale and mode;
- the approval reference, unique per workspace;
- one to three approvers.

Checks run in this order:

1. form;
2. lookup;
3. repeat:
   - the same request with the same content and actor returns the recorded outcome;
   - a recorded request identifier with other content or another actor, on any draft, is `ErrConflict`;
   - an approval reference already used on another draft is `ErrConflict`;
   - a different request on a released draft is `already_released`;
4. `ai_not_allowed` for the releasing actor;
5. `withdrawn`;
6. `stale_revision`, then `not_validated` when the current revision has no check, then `digest_changed` against the release subject digest;
7. `base_changed` when any field of the quoted base differs from the draft base, for example because the run was issued after the draft was opened. The draft must then be withdrawn and reopened over the issued base, as a correction;
8. `base_consumed` when another draft over the same workspace, run and run digest was already released;
9. `ai_not_allowed` when an approver is not a human;
10. `self_approval` when an approver opened the draft, authored or removed any entry, or recorded a check on any revision;
11. `nothing_to_adjust`, `zero_difference` and `negative_total`.

The outcome holds:

- the disposition:
  - `new_run` with the base run when the base is not issued;
  - `correction` with the base run and the original document when it is issued;
- the currency and the amount scale of the draft base;
- the difference lines and the totals, including the gross change;
- the draft digest and the release subject digest;
- the tax and allocation digests and amounts of the check;
- the approval reference, the approvers, the releasing actor and the time.

`Ready` runs the same checks and records nothing. A replay that `Release` would answer with the recorded outcome is also accepted by `Ready`.

A released draft accepts no `Add`, `Remove`, `Validate` or `Withdraw`. The register records the release; producing the run or the correction is the caller's task.

`Withdraw(workspace, draft, actor)` closes an open draft without a release, also after a validation. A human or a system may withdraw. Repeating it returns the view.

## Limits

`Limits{Workspaces, Drafts, Lines, Entries, Evidence}` sets the following:

- workspaces, up to 100,000;
- drafts per workspace, up to 1,000,000;
- lines per draft, up to 10,000;
- entries per draft, including removed entries, up to 1,000;
- evidence per entry, up to 64.

Each must be positive. Exceeding one is `ErrLimit`. An oversized line list is refused before its lines are parsed. Identifiers are at most 128 bytes from `[A-Za-z0-9._:-]`, and numbers are at most 64 bytes.

# Provisional choices awaiting the owner

- The reason list, and the rule that only `source_error` must link a finding.
- Assistants and systems may draft entries; humans and systems validate, release and withdraw; only a human approves.
- A zero difference is refused. A negative adjusted total is refused, while negative lines are allowed.
- Quantity differences telescope: round(target × price) − round(current × price).
- Entries apply in addition order, and targets are recomputed after a removal.
- Amounts are net of tax, and the final amount is the adjusted net plus the recomputed tax. It may be negative.
- The approval uses the `charge_approval` stage and its policy (validity, approver count and required kinds) rather than a stage of its own. It uses the target `adjustment:<draft>:<revision>`, the digest kind `result`, and the gross change as its amount.
- Only one draft over the same run and run digest may be released.

# Verification

- Literal tests:
  - every kind, scope and reason rule, and the period and evidence rules;
  - the check and release orders and replay, and isolation between workspaces;
  - the AUD-AR-07 scenario: a target total over sources leaves every source line unchanged and adds one explainable difference line. A change after approval refuses the release under the old digest, and a new check and approval are needed.
  - an integration test through `modules/approvals` covers these paths:
    - `Ready`, then `Execute`, then `Release` succeeds, with the approvers equal to the deciding actors;
    - a changed draft cannot be executed or released under the old approval;
    - `Ready` refuses before `Execute`, so the approval stays unconsumed;
    - a `Release` refused after `Execute` is reported as `refused_downstream`;
    - after a late `stale_revision`, the changed draft is validated, approved on its new target, executed and released.
- A property test runs random entry sequences, including removals, against an independent `big.Rat` model of the running quantities and amounts. It checks the following:
  - the differences, totals, gross change and adjusted lines match the model;
  - every source line stays unchanged;
  - quantity paths that reach the same final quantity give the same amount.
- Also run:
  - the race detector with count=10, and a 386 build;
  - gofmt and vet, and the whole repository test run;
  - the mutation run and the source scan.
- Not covered: QA-01, ERPC, the stage prerequisites, the check that source lines reproduce the run's charge lines, actual tax and allocation recomputation (taken as digests), producing the new run or the correction, persistence, screens, and independent DBA and QA verification.

# Review history

The first design review returned FAIL with thirteen findings. The body above has absorbed them:

- approvals bound by a release subject digest, single-use references and an integration test;
- the base re-quoted at release (`base_changed`);
- one release per run (`base_consumed`);
- the check in the subject digest, so approval follows validation;
- net amounts with the final amount equal to net plus tax;
- telescoping quantity rounding;
- exact replay rules for `Add`, `Remove`, `Validate` and `Release`;
- charge line references and the integration check on sources;
- check orders and actor kinds;
- the gross change as the approval amount;
- the digest field list;
- signed line targets and the withdraw rule;
- upper limits and request scope.

The re-review returned PASS on condition that seven text-level findings were written in. The body now covers them:

- the actual import rule and where the integration test lives;
- `Ready` before `Execute`, and `refused_downstream` after a late refusal;
- the namespaced target and the shared stage policy as an owner choice;
- `ErrConflict` for a reused request identifier;
- the empty-line scope;
- approvers as a caller duty checked in the integration test;
- the full `Remove` check order.

# Change review resolution

The first independent change review returned FAIL with two medium findings and six lower ones. All were fixed before the re-review:

- The approval target carries the revision, so a draft refused after `Execute` can be approved again. The flow test covers the recovery.
- The release compares every base field, and the outcome carries the draft currency and scale.
- The scale checks run right after lookup, as documented.
- A table test shows that each field reaches the draft digest.
- The property test compares every difference field with the model over scales 0 to 8. The path test allows only `zero_difference`.
- The final amount is signed.
- An oversized line list is refused before parsing.
- Checkers of earlier revisions are excluded from approval.

The re-review returned PASS with two low notes, both written into the plan: the release quotes the full base, and the integration layer withdraws unexecuted requests of earlier revisions.
