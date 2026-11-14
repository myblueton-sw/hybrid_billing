---
type: VerificationRecord
title: Adjustment drafts, re-validation and digest-bound release verification
description: Records what was executed and reviewed for the AR-07 increment of work item W-23, the adjustment register that keeps sources unchanged, shows every quantity correction, fixed amount and target total as a difference line, re-validates tax, allocation and final amounts per revision, and releases only under the approved subject digest as a new run or a referenced correction, with results, the mutation check, provisional choices and what remains unverified.
status: draft
---

# Delivered behavior

[HB-222](https://linear.app/hybrid-billing/issue/HB-222) adds the Go package `modules/adjustments` from main `8c9168e`. Its non-test sources use the standard library and the primitives `decimal` and `exact` only; its tests also import `modules/approvals` for the integration scenario. A registry with an injected clock and one mutex keeps, per workspace:

- **Drafts over frozen sources.**
  - A draft has a base: the run, its digest, the issued state and document, the currency, the amount scale and the rounding mode.
  - It also holds source lines with charge references, quantities, optional unit prices and net amounts.
  - Neither the base nor the lines ever change.
- **Entries.** There are three kinds, each scoped to a line or to the whole draft:
  - `quantity`: round(target × price) − round(current × price);
  - `fixed`: a signed amount;
  - `target_total`: a signed target minus the current amount.

  Each entry has a reason, evidence, an optional finding (required for `source_error`) and a period inside the draft period. Entries apply in order and are recomputed from the sources after every change. A removed entry stays visible.
- **Difference lines.** Each difference line shows the current value, the target, the quantity difference and the amount difference. The view gives the source, adjustment, adjusted and gross totals, plus the findings of active entries.
- **Re-validation.** A check binds the tax and allocation result digests, the tax amount and the final amount (adjusted net + tax, exactly) to one revision and digest. It produces the release subject digest that an approval binds.
- **Release.** The checks run in a fixed order:
  - replay and single use of request, approval and run;
  - actor kind;
  - state, stale revision, not validated and subject digest;
  - the full base (`base_changed`) and `base_consumed`;
  - approver kind and separation of duties (opener, authors, removers, every checker);
  - arithmetic.

  The outcome is a `new_run` before issue or a `correction` referencing the issued document. `Ready` runs the same checks without recording anything.

# Commands and results

All runs were made in the worktree `/private/tmp/hb222` on 2026-11-14 with Go 1.25.7 on darwin/arm64.

| Check | Result |
| --- | --- |
| `gofmt -l modules/` | no output |
| `go vet ./...` | no findings |
| `go test -race -count=10 ./modules/adjustments/` | ok (207.8 s) |
| `GOOS=linux GOARCH=386 go test -c ./modules/adjustments/` | builds |
| `go test -race -count=1 ./...` | 36 packages ok, 0 failures |
| `go test -cover ./modules/adjustments/` | 99.4% of statements |

The tests cover the following:

- **Literal scenarios:** every kind and scope, telescoping rounding, recomputation after removal, signed line targets and credits, and source-error findings.
- **AUD-AR-07:** a target total leaves the sources unchanged as a difference line, and a change after approval refuses the old subject.
- **Dispositions:** `new_run` before issue, `correction` after issue, and refusal when the run was issued after the draft opened.
- **Release rules:** one release per run, withdrawal, closure after release, the check and release orders, and separation of duties including earlier checkers.
- **Forms, limits and bounds:** forms, limits including caps, profile bounds including a removal that would leave the total beyond the profile, and the replay field comparisons.
- **Digests and concurrency:** a one-field-difference digest table, clock monotonicity, and concurrent releases and adds.
- **Property tests:** 400 random sequences of adds and removals at scales 0 to 8 and all six rounding modes, against an independent `big.Rat` model with its own rounding. Every difference field, every total and every adjusted line matches the model, and the source lines never change. A second property test checks that quantity paths reaching the same final quantity give the same amount.
- **Integration with `modules/approvals`:**
  - `Ready`, `Execute` and `Release` succeed;
  - an old approval cannot execute or release a changed draft;
  - `Ready` refuses before an approval is consumed;
  - a late refusal is reported as `refused_downstream`;
  - recovery after a late `stale_revision` goes through a new approval on the new revision's target.

# Mutation check

The script swaps comparison and logical operators, error kinds, arithmetic, state, kind, entry and disposition constants, negations and integer literals in the six production files. It ran 526 mutants:

- 495 killed;
- 18 did not build;
- 13 survived, all equivalent:
  - `Round(0)` → `Round(1)` on the clock;
  - the dot index and replace count in `bounded`, whose formatted text never starts with or repeats a dot;
  - the upper date bounds `2099-13-31` and `2099-12-32`, and `&&` → `||` after `time.Parse`. The parser already rejects these dates;
  - the sign test for adding to the gross change at zero;
  - the length prefix width and the slash index in `format`;
  - `fives >= scale`, where equal counts give the same scale.

Three survivors of the first run were killed by new tests:

- invalid dates that sort before the end date;
- an entry replay that changes only its kind.

# Reviews

- **Design review** (independent reviewer, read-only):
  - The first pass returned FAIL with 13 findings.
  - The second pass returned PASS on condition of 7 text-level findings.
  - All were folded into the plan.
- **Change review** (separate read-only reviewer):
  - The first pass returned FAIL with 2 medium and 6 lower findings.
  - All were fixed, and the re-review returned PASS with two low plan notes, which were applied.
- See the review history and change review resolution in the [plan](./plan.md).

# Acceptance status

AUD-AR-07 is covered at package level. NOT_RUN as a system test. AUD-QA-01 and ERPC-01 to ERPC-08 are not covered by this ticket.

# Provisional choices awaiting the owner

- The reason list, and the rule that only `source_error` must link a finding.
- Roles by actor kind:
  - assistants and systems may draft;
  - humans and systems validate, release and withdraw;
  - only humans approve.
- Zero differences and negative adjusted totals are refused. Negative lines and negative final amounts are allowed.
- Quantity differences telescope with the base rounding mode.
- Amounts are net, and the final amount is net plus the recomputed tax.
- Approval shares the `charge_approval` stage policy. The target is `adjustment:<draft>:<revision>`, the digest kind is `result`, and the amount is the gross change.
- Only one draft over the same run and run digest may be released.

# Not verified or not built

- The integration layer that joins the register and `modules/approvals` is not built. It also has to:
  - check that source lines reproduce the run's charge lines;
  - pass the deciding approvers;
  - withdraw requests for earlier revisions.
- Not built:
  - actual tax and allocation recomputation (taken as digests);
  - producing the new run or the correction;
  - persistence, screens, and the stage prerequisites.
- Independent DBA and QA verification has not been done.
- QA-01 (protected payment and recipient changes) and the ERP recognition cadence (ERPC-01 to ERPC-08) remain in the W-23 follow-up.
