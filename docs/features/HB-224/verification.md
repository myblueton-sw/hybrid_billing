---
type: VerificationRecord
title: Monthly ERP recognition and quarterly reconciliation verification
description: Records what was executed and reviewed for the ERP recognition cadence increment of work item W-23, the register of immutable monthly recognition batches with separate approval, purpose-bound submissions and posting results, corrections that never overwrite a month, approved recognition adjustments counted once, a quarterly reconciliation with explicit states and a clearance that refuses without a double-recognition method, with results, the mutation check, provisional choices and what remains unverified.
status: draft
---

# Delivered behavior

[HB-224](https://linear.app/hybrid-billing/issue/HB-224) adds the Go package `modules/recognition` from main `e714b02`. Its non-test sources use the standard library and the primitive `decimal` only. A registry with an injected clock and one mutex keeps, per workspace:

- **Profiles.** Entity, ledger, contract, amount basis, tax basis, currency, scale, whether provisional batches are allowed and the double-recognition method. Profiles are immutable, and amounts never add across profiles. `Close` closes posting months permanently.
- **Batches.** `initial`, `amendment` and `reversal` batches carry the attribution month, posting month, snapshot digest and lines with lineage. Corrections post in an open month and link to the original month. An amendment needs a posted live initial with no pending reversal. A reversal negates a posted, non-reversal batch once and takes its provisional flag. An initial can be reversed only after its amendments. Batches are approved by a human other than the author against the batch digest; open batches can be voided, and approved, never-posted batches can be abandoned.
- **Submissions and results.** Each `monthly_recognition` submission has a single-use idempotency key. No new key is accepted while a result is `sent`, `accepted` or `unknown`, or once a batch is posted. Result transitions are explicit, `posted` needs a workspace-unique external reference, and only posted batches count.
- **Recognition adjustments.** Each adjustment has a cause, before and after amounts, approval by a second human, withdrawal, and at most one carrying amendment with equal amounts and no reversal. It counts in the adjustment term until its carrier posts and in the monthly net afterwards, never both.
- **Reconciliation and clearance.** Net, tax and gross are reconciled separately per profile, over 1–12 non-calendar months. The states are `collecting`, `posting_unconfirmed`, `mixed_basis`, `adjustment_pending`, `amount_difference`, `adjustment_posting` and `matched`, and the result is bound to a reconciliation digest. `Clear` refuses without a method, without a match, for a changed digest and for overlapping quarters, and then freezes the cleared months. It issues nothing.

# Commands and results

All runs were made in the worktree `/private/tmp/hb224` on 2026-11-14 with Go 1.25.7 on darwin/arm64.

| Check | Result |
| --- | --- |
| `gofmt -l modules/recognition` | no output |
| `go vet ./modules/recognition` | pass |
| `go test -race -count=10 ./modules/recognition/` | pass (121.9 s) |
| `GOOS=linux GOARCH=386 go test -c -o /dev/null` in the package | builds |
| `go test -race -count=1 ./...` | pass, 38 packages with tests |
| statement coverage of `modules/recognition` | 100.0% |
| property test against an independent `big.Rat` model | 400 seeds × 120 steps pass; every state reached at least 50 times |
| mutation testing (`mutate224.py`, 796 mutants) | 738 killed, 53 did not build, 5 survived (all equivalent) |

The property model keeps its own batches, results and adjustments. It predicts which operations are accepted and computes the reconciliation with `big.Rat`. It also checks a counted-once invariant that uses none of the production predicates: the base must equal every approved adjustment of the months, plus every posted batch that neither carries an approved adjustment nor reverses such a carrier. When the change review's high finding was put back into the code, the property test failed on this invariant.

An earlier mutation run on the reviewed code left 18 survivors. Tests were added for the transition from `accepted` to `posted`, scaling below the profile scale, the hash framing, the flag encoding and an initial batch with matching amounts offered as a carrier. Two redundant conditions were removed: the gross difference, which is implied by the net and tax differences, and positional digit checks of the month, which the bounds already covered.

The 5 final survivors are equivalent:

- the input length guard at 64 or 65 bytes, and at `>` or `>=`, because no number the decimal profile accepts is longer than 24 bytes;
- `Round(1)` instead of `Round(0)` on the clock reading;
- `>=` in the sorted insertion of closed months, because a month is never inserted twice;
- the size of a fixed four-element array.

The 53 mutants that did not build were rejected by the compiler or `go vet`.

# Acceptance

| Criterion | Evidence |
| --- | --- |
| ERPC-01 monthly recognition without customer invoicing | `TestMonthlyRecognitionWithoutCustomerInvoice`: submissions carry purpose `monthly_recognition`, and the register has no issue, delivery or payment-term path |
| ERPC-02 100 + 120 + 80 = 300; 290 with a −10 adjustment counted once | `TestQuarterlyDiscountAdjustmentCountedOnce` |
| ERPC-03 missing, unconfirmed and partial months; idempotent resumption | `TestMissingOrUnconfirmedMonthIsNotComplete`, `TestUnknownResultBlocksNewKey` (batch level only) |
| ERPC-04 no automatic quarterly submission without a method | `TestClearanceRequiresMethodMatchAndDigest`. The provider-side external net check is not run. |
| ERPC-05 closed periods, corrections, reversals, final results | `TestClosedPeriodsAndCorrections`, `TestInitialReplacementRules`, `TestReversalTargets` |
| ERPC-06 entity, basis, currency and tax basis separation; provisional and final | `TestBasisSeparation`, `TestProvisionalAndFinalNeverMix` |
| ERPC-07 non-calendar quarters, approval separation, immutable snapshots | `TestNonCalendarQuarterAndSeparation`, form and conflict tests |
| ERPC-08 bulk month-end, projection lag, provider combinations | NOT RUN; outside this increment |

# Reviews

- **Design review.** Three rounds by an independent read-only reviewer:
  1. CONDITIONAL PASS: 6 high, 6 medium and 3 low findings.
  2. CONDITIONAL PASS: 1 high and 3 medium findings.
  3. CONDITIONAL PASS: 2 medium findings, to be treated as PASS once folded in, which was done.
- **Change review.** Two rounds by a separate independent read-only reviewer:
  1. FAIL: 1 high (a reversed amendment usable as a carrier), 2 medium and 3 low findings. All were fixed.
  2. PASS.

# Provisional choices

These are my choices, not the owner's, and need owner confirmation:

- the amount bases, tax bases, methods, causes, states, ranges and limits;
- gross is net + tax;
- submissions are batch-level;
- a reversal negates a whole batch, and an explicit zero initial counts as evidence;
- any human may void, close or withdraw;
- a posted result after closure is accepted;
- the clearance names the method, and provider support is verified later.

# Not verified or not built

- ERP connectors, provider profiles and the external net check of ERPC-04 (W-24, W-25); line-level partial resumption; detection of manual ERP edits.
- FX conversion, persistence, screens O03, U02, I01, I02, B08, B09, I03 and R01, DBA and QA sign-off, and the ERPC-08 scale runs.
- The W-24 common submission ledger, which will absorb the recognition-purpose submission record.
