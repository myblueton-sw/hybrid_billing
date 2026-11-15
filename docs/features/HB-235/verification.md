---
type: VerificationRecord
title: Closing cycles verification
description: Records what was executed and reviewed for the first increment of work item W-31, a register of closing cycles bound to approved calendar revisions, with separate stage deadlines and completion bases, fixed target manifests, work items that complete in stage order, reopen, hold and are waived under dual control, postponements that keep the original due date, progress as of any cycle sequence, and due claims that are signals only. The record lists results, the mutation check, provisional choices and what remains unverified.
status: draft
---

# Delivered behavior

[HB-235](https://linear.app/hybrid-billing/issue/HB-235) adds the Go package `modules/closing` from main `dc3cf98`.

- **Calendars.** Dual-controlled revisions numbered by the register, with an IANA zone, up to six weekend days, up to 1,000 holidays, a local close time (1440 meaning the end of the date) and a covered range; revisions never change.
- **Deadlines.** Each stage's deadline is the N-th business or calendar day after the period's last day at the close time; DST gaps move to the first valid instant after the gap and overlaps take the earlier instant; searches stay within the covered range and are bounded; deadlines must not decrease along the stage order and are stored as instants.
- **Completion bases.** `internal_approval` (humans only), `erp_posting` (actually posted) and `file_receipt` (confirmed receipt); `settle` means internal settlement only and never customer payment status.
- **Manifest and items.** Every change takes the next cycle sequence (opening is 1). Targets are added in revisions and never removed. Items complete in stage order with matching evidence and lateness against both due dates; reopening cascades to later done stages and keeps history; holds and dual-controlled waivers (with a pending state that blocks other changes) never count as done.
- **Postponements.** Dual-controlled, one pending per stage, later than the current due and now, never past the next stage's current due; the original due never changes.
- **Progress.** As of any sequence and instant: denominator (targets minus waived), done, waived, held, open, open and held past due, done late against the original due from the latest completion, and the original and then-current due.
- **Claims.** System actors claim due-soon, due and overdue events (24 hours after the due) per item once per current due; each stage keeps the set of items still owed a step; events carry the due date and step instant, are ordered by due date, cycle, stage and target, grouped, bounded and record skipped steps; claiming changes nothing.

# Commands and results

All runs were made in the worktree `/private/tmp/hb235` on 2026-11-16 with Go 1.25.7 on darwin/arm64.

| Check | Result |
| --- | --- |
| `gofmt -l modules/` | no output |
| `go vet ./modules/closing` | pass |
| `go test -race -count=10 ./modules/closing` | pass |
| `GOOS=linux GOARCH=386 go test -c ./modules/closing` | builds |
| `go test -race -count=1 ./...` | 48 packages, all pass |
| Coverage | `modules/closing` 100.0% |
| Mutation testing | 559 mutants of `calendar.go`, `registry.go` and `items.go` on the final code: 516 killed, 20 did not build, 23 survived |

The 23 survivors were read and judged equivalent: in the DST helper, any gap-scan start at or before the transition gives the same result, so the offset bounds, hour range and the nanosecond in the wall clock do not change outcomes; `parseDate` with `||`; the search bound at 400 or 401 days; a `[][3]` limits array; `Round(1)` on clocks and due dates; an approved postponement entry can never be at index 0; a step of -2 instead of -1 is treated alike; `<=` in comparators reached only for distinct keys; a truncation at exactly the limit; and a settle order value of 7 that keeps the order.

A property test over 40 seeds of 150 random steps (additions, completions, reopenings, holds, releases, waiver requests and approvals) predicts every refusal and compares targets, done, waived, held, open and the denominator at every recorded sequence with an independent model.

# Reviews

- Design review: two rounds by a separate read-only reviewer; FAIL, PASS ([plan](plan.md)).
- Change review: two rounds by a fresh read-only reviewer; FAIL (claim due dates and full scans), PASS.

# Provisional choices for the owner

- A target added after a stage's original due date is late from the start.
- Stage names, completion bases and evidence kinds; the 24-hour overdue step and the due-soon lead per registry.
- Reads take no actor until the `modules/auth` adapter exists.

# Not verified and residuals

Due events into `modules/incidents` and escalation chains, NotificationEvent and Attempt, external messages with authenticated links, a persistent outbox with leases and acknowledgements, waiver authority through `modules/auth`, quarterly and non-calendar periods and the quarterly billing close, per-item postponement, customer payment due and overdue status (`modules/receivables`), bulk commands, persistence and screens. Each claim still passes over all stages of the workspace.
