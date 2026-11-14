---
type: VerificationRecord
title: Data-centre components and SLA credits verification
description: Records what was executed and reviewed for the AR-10 increment of work item W-22, the component register with inclusions, monthly statements that never charge included quantities, incidents as evidence, and SLA credits under approved clauses with operator approval and links, with results, the mutation check, provisional choices and what remains unverified.
status: draft
---

# Delivered behavior

[HB-219](https://linear.app/hybrid-billing/issue/HB-219) adds the Go package `modules/components` from main `dcff4ca`. It uses the standard library and the primitives `decimal` and `exact`, and imports no other module. A registry with an injected clock and one mutex keeps, per workspace:

- **Pricing.** Each contract has a scale (0 to 8) and a rounding mode, written once.
- **Components (AR-10).** These are fixed rack, space, line, reserved GPU and service fees, and usage add-ons on a meter.
  - Fixed components declare inclusions: a quantity per meter, or `all`.
  - Each component is approved by another actor.
  - A meter has at most one approved usage component (`meter_overlap`) and at most one including fixed component (`inclusion_overlap`) at any time, whatever the currency.
- **Monthly statements.** A statement covers exactly one calendar month and is a preview that records nothing.
  - Fixed lines carry their fee.
  - Usage lines charge max(0, measured − included), or nothing when the meter is fully included. Each line is rounded with the contract pricing.
  - Readings with no usage component are shown as included, as unpriced excess, or both. Unpriced quantities are never hidden and never charged.
  - A component covering only part of the month (`partial_period`), two currencies (`currency_mismatch`) and missing readings (`missing_reading`) are refused.
  - `Net` = `Charges`.
- **Incidents.** They are evidence only and never change a statement.
- **SLA clauses.** A clause has a rate in (0, 1] of a fixed fee and sits within the component dates. It is approved by another actor, and clauses must not overlap.
- **SLA credits.**
  - A credit covers one month under an approved clause on an approved component. It is backed by incidents that reach into the month and lie within the clause dates.
  - Each incident backs at most one credit. There is at most one credit per component and month.
  - Its amount is the fee × the rate, rounded. A zero amount is refused.
  - A credit is approved by an actor other than its proposer, which is the operator approval.
  - A link to a credit document and to a balance increase is unique per workspace.
  - Approved credits appear on the statement as informational lines (`linked` or `unlinked`) and are excluded from `Net`, so they settle once, through the balance.

Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviewed the plan twice and the change once (see the plan). Machine: mac_mini. Nothing calls the package.

# Executed checks

All checks ran on 2026-11-14 on the mac_mini (darwin/arm64, Go 1.25.7) in the ticket worktree.

| Check | Result |
| --- | --- |
| `go test ./modules/components` | ok |
| `go test -race -count=10 ./modules/components` | ok |
| `GOOS=linux GOARCH=386 go test -c ./modules/components` | builds |
| `gofmt -l modules`, `go vet ./modules/components` | clean |
| `go test -race -count=1 ./...` | 34 packages ok, no failures |
| Statement coverage of `modules/components` | 99.3% |
| Source scan (no binary fractions, no comments, imports only the standard library and the primitives, under 1,500 lines) | ok |

The tests include:

- **AUD-AR-10 at package level:**
  - a rack that includes all power, with 8000 units of power usage charged 0;
  - a second fixed charge including power, refused;
  - an incident alone, which creates no line;
  - a credit under an approved clause, approved by another actor, linked, and shown once without changing `Net`.
- **Inclusions:** partial inclusion at its bounds, and half-even rounding.
- **Statements:** uncovered meters, whole-month and partial-month components, leap February.
- **Overlap rules:** components sharing one day, concurrent approvals that leave one usage component per meter, and clause, credit, incident and link uniqueness.
- **Forms and repeats:** every field of the repeat comparisons, malformed arguments, and room limits.
- **A pinned digest.**

A property test of 400 seeds against a `math/big.Rat` model checks:

- random fixed components, inclusions, usage components and readings, with a random scale (0 to 4) and all six rounding modes;
- random credits that are withdrawn, approved, or approved and linked;
- charges, line sums, unpriced excess, credit amounts and the set of listed credits.

Mutation check (`mutate219.py` on the final production code, with survivors rerun after the last tests): 619 mutants, 593 killed and 9 did not build. 17 survived, all classified as equivalent:

- 7 bounds already enforced by the parser, or two-sided date checks;
- 3 `Round(0)`→`Round(1)` changes;
- 2 changes to the sentinel date;
- 1 sort over unique identifiers;
- 3 details of the format routine;
- 1 unreachable guard: credits whose period lies outside the component dates. Clause dates lie inside component dates, so this cannot happen; the guard is kept.

# Acceptance status

- AUD-AR-10 is covered at package level, with two limits:
  - a fixed fee that charges for power without declaring the meter in `Includes` is not detected;
  - "linked to the balance" is a recorded reference only, not checked against `modules/balances`, and is UNVERIFIED.
- NOT_RUN as a system test.

# Provisional choices awaiting the owner

- Only calendar-month cadence is supported, with no proration.
- SLA credits use one formula, rate × fixed fee. Tiered SLA levels and availability computation are not built.
- At most one fixed component includes a meter at a time, which refuses stacked bundles.
- Unpriced quantities are shown and never charged.
- SLA credits are settled only through the credit document and the balance, never as a statement deduction.
- Approved credits cannot be reversed in this increment.

# Not done or unverified

- The change-review fixes were not reviewed again.
- Independent DBA and QA verification were not done.
- Not built: documents, persistence, screens (O03, B01, B04, B11), and the integration that checks the balance link.
- The remaining W-22 scope (DB-10 remainder, refunds, external authority locks, included pools, PT-05 and full PT-08) is not in this ticket.
