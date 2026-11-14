---
type: VerificationRecord
title: Contract payment terms, due dates, overdue status and aging verification
description: Records what was executed and reviewed for the payment-term increment of work item W-22, the receivable register with term bindings, issue snapshots, extensions, allocations, reversals, freshness, status and aging, with results, the mutation check, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-218](https://linear.app/hybrid-billing/issue/HB-218) adds the Go package `modules/receivables` from main `5dcffc2`. It uses the standard library and the primitives `decimal` and `exact`, and imports no other module. Zone data comes from an injected resolver. A registry with an injected clock and one mutex keeps, per workspace:

- **Terms**: net days and calendar mode, approved by another actor. Business-day terms can never be approved.
- **Contract bindings**: a term revision for an effective date range in one zone per contract. Overlap is refused at approval. Ends shorten a binding, are approved by another actor, and are re-checked against issued documents.
- **Issue snapshots (PT-01, PT-02)**:
  - The issue date is in the contract zone, and the original and effective due date is the issue date plus net days (day 0).
  - When no binding covers the issue date, the issue is refused with `term_missing`, `term_pending` or `term_conflict`.
  - A redelivery returns the stored snapshot, and later bindings never change it.
  - A replacement inherits or reissues the due date, at most once per document.
  - Internal documents are `not_applicable`.
- **Extensions**: they only lengthen the due date and are approved by another actor. Each keeps the previous due date and the days overdue at approval, and the original due date stays as it was.
- **Allocations and reversals (PT-03)**:
  - Authority allocations take effect when recorded. Manual allocations need evidence and a second approver.
  - Allocations are refused when they exceed the outstanding amount or are dated in the future.
  - Reversals are full, once per allocation, from the same source as the allocation, and not dated before it.
  - Outstanding = amount − approved, unreversed allocations, exact.
- **Freshness (PT-04)**: a confirmation per entity and currency never moves back.
- **Status**: settled, not due and due today come first, with a stale flag. Once past the due date, the status is unknown when confirmation is missing or stale, and overdue otherwise. Day counts are signed, in the contract zone, and immune to DST.
- **Holds (PT-07 axis)**: dispute and collection holds are kept beside the status and never change it.
- **Aging (PT-06)**: per currency, the buckets are not due, due today, 1-30, 31-60, 61-90, 91 or more, unknown and replaced. Each bucket reports a count, a sum, and the stale count and stale sum. The total equals the sum of the buckets.

Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviewed the plan twice and the change once (see the plan). Machine: mac_mini. Nothing calls the package.

# Executed checks

All checks ran on 2026-11-14 on the mac_mini (darwin/arm64, Go 1.25.7) in the ticket worktree.

| Check | Result |
| --- | --- |
| `go test ./modules/receivables` | ok |
| `go test -race -count=10 ./modules/receivables` | ok |
| `GOOS=linux GOARCH=386 go test -c ./modules/receivables` | builds |
| `gofmt -l modules`, `go vet ./modules/receivables` | clean |
| `go test -race -count=1 ./...` | 33 packages ok, no failures |
| Statement coverage of `modules/receivables` | 99.3% |
| Source scan (no binary fractions, no comments, imports only the standard library and the primitives, no embedded zone data in production code, under 1,500 lines) | ok |

The tests include:

- the planning examples (2026-10-01 plus 30 days is 2026-10-31, plus 45 days is 2026-11-15);
- midnight on either side in Seoul and New York;
- month ends, 2028-02-29, and the DST changes in New York and Berlin;
- every refusal combination at issue;
- extensions of an overdue document;
- partial payments, credits, cancellations, chargebacks and refused overpayment;
- mismatched reversal sources;
- missing, stale and exact-limit freshness;
- aging boundaries at 0, 1, 30, 31, 60, 61, 90 and 91 days;
- the replacement window;
- holds that change nothing;
- concurrent allocations that never exceed the amount;
- a form check of every argument;
- a pinned term digest.

A property test of 300 seeds × 60 calls in Seoul, New York and Berlin checks, after every call, outstanding against a `math/big.Rat` model, status, days, stale flag and effective due date against an independent model, and aging totals against the model sum.

Mutation check (`mutate218.py` on the final production code; the survivors were rerun after the last tests were added): 863 mutants. 821 were killed and 26 did not build. 16 survived, and all 16 were classified as equivalent:

- 5 number-length or date-range bounds that the parser already enforces;
- 3 `Round(0)`→`Round(1)` changes, which strip the monotonic reading the same way;
- 2 changes to the open-end sentinel date, which stays above every valid date;
- a maximum scan where equality changes nothing;
- 2 clamps to zero that give the same result;
- 3 format details where the index can never be 0 or the scales are equal.

# Acceptance status

- PT-01, PT-02, PT-03, PT-04, PT-06 and PT-07 are covered at package level, without ERP synchronization, notifications or screens. NOT_RUN as system tests.
- PT-05 (portal, search, export and permission boundaries) is not covered.
- PT-08 is partial: aging scans the workspace under the lock. There is no indexed projection, no catch-up after a failure and no projection-lag display.

# Provisional choices awaiting the owner

- Only calendar days are supported. Business-day terms are recorded but never approved.
- The maximum net days and the freshness limit are registry limits.
- There is one zone per contract.
- Staleness is judged per entity and currency, and applies only after the due date. A settled document stays settled even when the confirmation is stale.
- Extensions only lengthen the due date, take effect at approval, and cannot be cancelled.
- The caller chooses whether a replacement inherits or reissues the due date. A replaced document is shown in a separate aging bucket until it is cancelled.
- Overpayment is refused here; it belongs to the cash ledger.
- The ledger revision is recorded but not used for ordering.
- Amounts print as exact values with the fewest digits.

# Not done or unverified

- The change-review fixes were not reviewed again.
- Independent DBA and QA verification were not done.
- Pinning the zone data version is a deployment obligation and is UNVERIFIED.
- Not built: persistence, screens (O03, I01, I02, B11, U01, R01, U03, C01, C02), notifications, ERP synchronization and reconciliation findings for due-date disagreements.
- The remaining W-22 scope (AR-10, DB-10, refunds, external authority locks, included pools) is not in this ticket.
