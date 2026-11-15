---
type: ImplementationPlan
title: Closing cycles with approved calendar revisions, separate stage deadlines and completion bases, fixed target manifests, waivers and holds outside the done count, postponements that keep the original due date, reopening and due claims that are signals only
description: Defines the first increment of work item W-31, a register of closing cycles per period bound to an approved calendar revision, with separate deadlines and completion bases for collection, validation, approval, issue, delivery and settlement; fixed target manifests whose revisions keep every earlier denominator; work items completed with matching evidence in stage order, reopened, waived under dual control or held; original and current due dates with dual-controlled postponements; and a due index that hands out bounded claims of due-soon, due and overdue events that never act as commands.
status: draft
---

# Scope and ownership

[HB-235](https://linear.app/hybrid-billing/issue/HB-235) starts work item W-31 (requirement HB-31; QA-03; AUD-QA-03; HB-31-I and HB-31-V closing parts) after W-30. Base: main `dc3cf98`. Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviews the plan and the change, because the register decides what counts as closed and on time. Machine: mac_mini.

Already delivered: AR-08 (HB-217), DB-10 (HB-212 to HB-219), QA-08 notification incidents (HB-234). Out of scope, recorded as residuals: feeding due events into `modules/incidents` and the escalation chain to owners, NotificationEvent and Attempt with snooze, minimal-identifier external messages with authenticated links, a persistent outbox with leases and acknowledgements, virtual-clock tests of absent owners, channel failures and permission revocation (covered by HB-234 for notifications), the `modules/auth` permission adapter and waiver authority, quarterly and non-calendar periods and the quarterly billing close (`erp-recognition-cadence.md`), per-item postponement, customer payment due and overdue status (`modules/receivables`), bulk commands, persistence and screens (U01 to U03).

# Observed baseline

- HB-31-D: collection, validation, approval, issue, delivery and payment deadlines differ; a deadline arriving is not an issue command; reading, accepting and resolving a notification are separate; closing calendar, business hours, channels and escalation.
- HB-31-I: ClosingCycle, CalendarRevision, WorkItem, NotificationEvent and Attempt; holidays, month end, DST, changes, snooze, alternates and re-checks after interruptions; external messages carry minimal identifiers and authenticated links.
- HB-31-V: a virtual clock tests missed schedules, postponements, reopening, absent owners, channel failures and permission revocation; internal settlement and ERP or file completion use different criteria; a due-at index with small claims and a persistent outbox, never a full customer scan; grouping by cause, target, period and stage.
- QA-03 and AUD-QA-03: original and current due dates with change history and a fixed target manifest; a waiver is a state separate from completion and needs a reason, authority and evidence; denominator changes from new targets, waivers and holds keep their revisions; postponement never erases the first due date or the lateness history; waivers are not counted as done, and earlier denominators, first due dates and lateness stay readable after additions, postponements and reopening.

# Decision

A new package `modules/closing` follows the sibling conventions: injected clock, one mutex, workspace isolation, `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (`Refusal{Reason}`), `ErrLimit`; check order form, lookup, repeat, rule, room; assistants refused everywhere; humans for decisions, system actors only where stated. `NewRegistry(Config{Zones}, Limits{Workspaces, Calendars, Cycles, Targets, History, Claims, Lead}, clock)`; `History` bounds the events kept per cycle and `Lead` (1 hour to 7 days) is the due-soon lead time.

## Calendars

`ProposeCalendar(workspace, Calendar{ID, Zone, Weekend, Holidays, Close, Covers}, actor)` by a human proposes the next revision of a calendar (numbered by the register; the same content by the same human while pending returns it): an IANA zone, at most 6 weekend weekdays, at most 1,000 holiday dates, the local close time as minutes after midnight (1 to 1440, where 1440 means 00:00 of the next day, the end of the date), and the covered date range. `ApproveCalendar(workspace, id, revision, actor)` by another human activates a revision. Revisions never change.

## Cycles and deadlines

`OpenCycle(workspace, Cycle{ID, Period, Calendar, Revision, Stages}, actor)` by a human opens a cycle for a monthly `YYYY-MM` period, bound to an approved revision of a calendar in the same workspace (`calendar_not_approved`). Stages come from `collect`, `validate`, `approve`, `issue`, `deliver`, `settle` in that order (a subset, each at most once). Each stage has a rule `{Days, Business}` with `Days` 1 to 31: the deadline falls on the `Days`-th business day (or calendar day) after the period's last day, at the close time in the calendar zone. Local times in a DST gap move to the first valid instant after the gap; in an overlap the earlier instant is used. The business-day search never leaves the covered range (`outside_calendar`) and is bounded. Deadlines must not decrease along the stage order (`stage_order`). The computed instants are stored as `original_due` and `current_due`, so later zone data never moves them. A deadline arriving changes no state and issues nothing.

Each stage names its completion basis and the evidence kind it needs: `internal` (`internal_approval`, by a human only), `erp` (`erp_posting`, an actually posted ERP reference, never a submission or acceptance) or `file` (`file_receipt`, a confirmed receipt, not a download). Which stage uses which basis is chosen per cycle. `settle` means internal settlement only: it never represents customer payment, due dates or overdue status, which belong to `modules/receivables`.

## Sequence and manifest

Opening a cycle takes sequence 1. Every later change to a cycle (target additions, completions, reopenings, holds, releases, waivers, postponements) gets the next cycle sequence number, returned to the caller. `AddTargets(workspace, cycle, Addition{Targets, Reason}, actor)` by a human adds targets (each at most once per cycle); every target gets one work item per stage, with the stage's original due date (a target added after that date is late from the start; recorded as a provisional choice). Targets are never removed; they leave the denominator only by waiver.

## Work items

States: `open`, `done`, `waived`, `held`, plus a pending waiver request on an open or held item.

- `Complete(workspace, cycle, target, stage, Evidence{Kind, Reference}, actor)` by a human (or a system actor for `erp` and `file`) completes an open item without a pending waiver when the evidence kind matches the stage basis (`basis_mismatch`) and every earlier stage present in the cycle is done or waived for that target (`stage_order`). It records the completion time, the lateness against `original_due` and against the then `current_due` (completing exactly at a due instant is on time). The same evidence again returns the item; other evidence on a done item is `ErrConflict`.
- `Reopen(…, reason, actor)` by a human reopens a done item and every later done stage of that target; later waived stages stay waived; every completion stays in the item's history.
- `Hold(…, reason, actor)` and `Release(…, actor)` by a human hold an open item and return it to open.
- `RequestWaiver(…, Waiver{Reason, Evidence}, actor)` by a human on an open or held item (one pending request per item), `ApproveWaiver(…, actor)` by another human, `WithdrawWaiver(…, actor)` by the requester. While a waiver is pending, completion, hold and release are refused (`waiver_pending`). A waiver is final in this increment; checking the waiving person's authority through `modules/auth` is a residual.

## Postponements

`RequestPostponement(workspace, cycle, stage, Postponement{Due, Reason}, actor)` by a human (one pending per stage), `ApprovePostponement(…, actor)` by another human, `WithdrawPostponement(…, actor)` by the requester. The new due must be after the current `current_due` and after the approval time (`not_later`), and current due dates must still not decrease along the stage order (`stage_order`). `original_due` never changes; every change stays in the stage's history. Postponing a single item is a residual.

## Progress and items

`Progress(workspace, cycle, stage, AsOf{Sequence, At})` returns the stage as it was after cycle sequence `Sequence` (0 means the latest; beyond it is `ErrNotFound`), evaluated at instant `At` (zero means now): the denominator (targets minus waived; held items stay in it and are reported separately), and the counts of targets, done, waived, held, open, open past due, held past due, done late against `original_due` (from each item's latest completion), the original due and the current due rebuilt from the postponements up to that sequence. Past due means strictly after that `current_due`. `Item(…)` returns an item's state, history and, for an open item, its lateness against `original_due` at the instant.

## Due claims

`Claim(workspace, limit, actor)` by a system actor hands out at most `limit` (1 to `Limits.Claims`) item events from a due index with one entry per cycle stage keyed by its current due date and, per stage, the set of open items; done, waived and held items leave the set, reopened and released items return to it, and only stages whose next step is due are visited; within such a stage the claim walks that stage's open set, which is the bound of this increment. Steps are `due_soon` (from `Lead` before `current_due`), `due` (at it) and `overdue` (24 hours after it). Each item is handed each step once per `current_due` value; after a missed run only the latest step that is due is handed out and earlier unhanded steps for that due date are recorded as skipped; a postponement re-arms all steps for the new due date. Events come in a stable order (due instant, cycle, stage, step, target) and are grouped by cycle, stage and step with their targets. Claiming changes no work item. There is no lease or acknowledgement: a consumer that crashes loses its claimed events (residual, with the persistent outbox).

# Verification plan

Literal tests for every refusal and repeat; DST gap and overlap, the 1440 close, `outside_calendar`, `waiver_pending`, `not_later`, a postponement that would break stage order, steps skipped after a missed run, re-arming after a release or late addition, and the sequence boundaries of progress; deadlines across month ends, leap February, holidays, weekends and a DST zone; stage order and completion bases; waivers and holds outside the done count; postponements keeping the original due and lateness; reopening keeping history; progress at earlier manifest revisions after additions, waivers, holds, postponements and reopening; claims with a virtual clock (missed runs, postponements, no repeats, grouping, limits); a property test of random operations against an independent model of item states and progress; `-race -count=10`, a 386 build, gofmt, vet, the whole repository with `-race`, 100% coverage and mutation testing.

# Review history

- Design review round 1: FAIL (3 high, 7 medium, 3 low). Folded in: cycle sequence on every change and a defined denominator (1); progress as of a sequence and an instant with rebuilt due dates (2); DST gap and overlap rules, bounded searches, covered ranges, 1440 close (3); `settle` as internal settlement only (4); typed evidence per basis, human-only internal approvals, conflicting repeats (5); lateness against original and current dues, boundaries, late-added targets (6); pending waivers, reopen cascade, held past due (7); postponement rules and stage order (8); claim index, steps, missed runs, re-arming, ordering, no lease (9); monthly periods only, others a residual (10); calendar revision numbering and stored instants (11); history limit (12); residuals (13).
- Design review round 2: PASS. Folded in: the per-stage walk is the stated claim bound (R1), opening takes sequence 1 (R2), the new rules are named in the verification plan (R3).

# Change review resolution

Change review round 1: FAIL (2 medium, 5 low). Resolved:

- Claim events carry the stage's current due date as `Due` and the step instant as `At`, and are ordered by due date, cycle, stage, step and target (1).
- Each stage keeps the set of open items still owed a step; claims skip stages whose set is empty and walk only that set, items leave it when they stop being open or have been handed every step for the current due, and postponements, reopenings, releases and additions put them back (2). The remaining bound is one pass over the workspace's stages per claim.
- The human-only rule for internal approvals also applies to repeats (3); identical repeats of target additions and of waiver and postponement approvals by the same person return the stored view (4); a pending calendar proposal refuses others with `calendar_pending`, and the claim events themselves are the record of skipped steps (5).
- Reads (`CycleOf`, `Progress`, `Item`) take no actor, as in the plan; actor checks on reads belong to the `modules/auth` adapter residual (6).
- Tests added for due-ordered claims, empty owed sets, idempotent repeats, cross-workspace isolation, a cycle on a DST calendar and stages sharing a due date; a mislabelled test was fixed (7).

Change review round 2: PASS. The cross-workspace test now gives the first workspace an approved revision 2 that the second lacks.
