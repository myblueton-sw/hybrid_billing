---
type: VerificationRecord
title: Contract lifecycle, settlement state and proration verification
description: Records what was executed and reviewed for the first model of work item W-15, with results, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-178](https://linear.app/hybrid-billing/issue/HB-178/w-15-contract-lifecycle-events-independent-settlement-state-period) adds the Go package `modules/contracts` from main `eccf3b6`: an append-only, dated lifecycle per contract and workspace with one sequence of knowledge points; a settlement state stored on its own, with guarded transitions and evidence records in which a gap keeps its reason, owner and follow-up until a secured record names it; the split of a billing period into segments by the lifecycle as known at a point; the proration of a fixed periodic amount with cumulative rounding; and the difference between two proration results. See the [plan](plan.md).

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan twice before implementation and the implemented change afterwards (see Independent review). Machine: mac_mini.

Nothing calls the package: there is no command, permission check, screen or storage for it, and it must not be exposed directly. Ending a contract in it has no outward effect. The cases REV01, GH05, GH06, AUD-AR-01, AUD-AR-10, AUD-QA-02, PT-01 to PT-08 and ERPC-01 to ERPC-08 remain NOT_RUN at system level, and W-15 is not complete.

# Executed checks

Executed on 2026-11-13 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain. No other operating system was tested.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All eleven packages `ok`, including `modules/contracts`; run again after the last correction |

The acceptance of HB-15, with expected values written by hand (`prorate_test.go`):

| Case | Expected | Result |
| --- | --- | --- |
| 300,000 for a 30-day period, 10 days served | 100,000 | 100,000 |
| That amount with an arrears addition of 50,000 (the decimal primitive's addition) | 150,000 | 150,000 |
| Correction of the advance bill of 300,000 once the end is known | −200,000 | −200,000 |

Literal table tests:

- days and intervals (`lifecycle_test.go`): fourteen dates including leap years and both ends of the range; nine intervals with their day counts, among them 28, 29, 30, 31, 365 and 366 days and the whole range; moving a day forward and back; moves beyond either end and moves large enough to wrap an integer, all refused; four invalid intervals;
- lifecycle rules: seven valid and twenty-five invalid lifecycles (order, repeated or missing start, events after the end, two events on one day in three combinations, terms present or absent on the wrong kind, an unknown kind, an identifier that is text), each also refused by `Segments`;
- segments: fourteen periods against one lifecycle, among them an event on the first day of the period, on its last day and on the day after it, a renewal that cuts nothing, a period before the start and after the end, and no events; independence of the order in which events are given; the state on eight days;
- proration (`prorate_test.go`): thirty-three tables with each line, the total and the totals per revision, among them a price change in mid-period, suspended days billed and skipped, periods of 28, 29 and 31 days, a period over 29 February and over a year boundary, 100,000 over 31 days in three segments under each of the six rounding modes, the same as a credit under four modes, the `floor` asymmetry between a charge (32,258) and its mirrored credit (−32,259), a zero line for a share that is not zero, a zero amount, an unused amount, terms that return after a change, later terms with the lower number, four decimals; a line off by exactly one unit under `half_even`;
- proration refusals: thirteen missing or invalid members of the basis; a scale beyond the profile, also when nothing is billed; knowledge point 0 and scale 0 accepted; no profile; twelve segment lists that do not cover the period (none, the billable part only, one chunk, the tail, the wrong order, a gap, an overlap, an unknown state, an empty or a reversed segment between contiguous neighbours, a segment twice); a missing amount, an unset amount, more decimals than the scale, amounts of both signs, with a zero amount beside either sign accepted; an amount whose product with the days passes the profile's bound refused with the primitive's error, and the largest input over a month accepted;
- corrections: the netting case (+40,000 made of −200,000 and +240,000); corrections over three knowledge points adding up to the correction over the span; the reverse; a result against itself; three terms returned in order on forty repetitions; another profile and another profile revision; seven mismatches of the basis in both directions; another price for terms billed in both; a result whose totals per revision do not sum to its total, that lists a revision twice or without a price; an empty result;
- batches (`registry_test.go`): backdated events, a withdrawal with its replacement, and the lifecycle, the state and the segments at each earlier knowledge point; twenty-one refused batches with nothing written and the write identifier not consumed; the log's order, times and independence from the caller's and the reader's slices; the same contract identifier in two workspaces; unknown workspaces and contracts not found;
- repeated writes: every kind of write repeated with identical content returning its stored sequence, also after the state moved on and before the expected sequence is looked at; other content, another order of events and another kind of write under a stored identifier as conflicts;
- settlement: the three situations of AUD-QA-02 at package level (the last collection fails; an emergency withdrawal on a running contract; a source received after the end), each leaving a record with reason, owner and follow-up that is still visible, and resolved only by name, at its own knowledge point; an end opening the final settlement in the same write; `Settle` refused without evidence, with only gaps, with a gap recorded before the end still open, with evidence that stops short of the end, before the end day, on evidence from before a reopening with no gap open, and on evidence older than the last batch although it covers the new end; secured evidence refused on a running and on a settled contract and when it names an unknown gap, a resolved gap, a secured record, or an open and an unknown gap together, with nothing resolved by a refused record; every batch refused while settled; a gap recorded while settled not reopening by itself; the mistaken end cancelled only with a reason, and the reason refused in each case where it is not due; a voided contract settled without evidence only when no gap is open, and started again only with a cancellation; eighteen malformed evidence records;
- limits: required limits; ordinary writes, a batch with a withdrawal, an end beside another event, secured evidence, `Settle` and the cancellation of an end refused at the limit; a suspension, an end, a gap record and a reopening written from the reserve, and each refused when the reserve is used up; the contract limit per workspace, with a refused first batch creating nothing;
- a registry that is not built.

Property tests (`property_test.go`, fixed seeds), against a reference model written in the test that knows nothing of segments: it steps through the period day by day with the standard library's calendar, decides each day's state with its own function, sums exact shares with `math/big.Rat` and rounds with its own rounding function, which is checked by eighteen literal cases.

- 1,500 generated lifecycles and periods of 1 to 366 days: the segments cover the period exactly and agree with the model on every day; the total equals the model's rounded total; each line equals the model's own cumulative rounding, has the sign of its exact share or is zero, and differs from it by at most one unit; the lines and the totals per revision sum exactly; cutting segments further never changes the total; a full active period bills exactly the price; the correction against an earlier part of the same history equals the difference of two model totals. Reached: 1,006 single-segment and 494 multi-segment cases, each rounding mode 239 to 265 times, 411 credits, 536 periods with nothing billed, 178 with several revisions billed, 687 non-zero corrections, 526 refinements, 14 zero lines for a non-zero share.
- 450 histories of 60 random writes: 888 batches, 1,299 gap records, 1,015 secured records, 90 settlements and 63 reopenings written, and 23,645 writes refused. After every write the settlement is none exactly when the contract is neither ended (read from the state on the last representable day) nor voided; a refused write never moves the sequence; every settlement of an ended contract (32) had, read from the log, a secured last record later than the last batch and the last reopening, covering through the end, with the end not after the day given; and the lifecycle, settlement and state at every earlier knowledge point are unchanged at the end.

Concurrency under the race detector: sixteen batches and sixteen settlements with the same expected sequence, of which exactly one is written, and never a settlement of a lifecycle its caller did not see.

Limits of these tests: the model's day-state function implements the same rules a second time and is fed only valid, ordered histories; the random writes are mostly refused (random kinds on random days), so deep lifecycles are reached by the table tests rather than by that property; rounding expectations in the tables were computed by hand.

# Mutation check

147 single-line mutations of `modules/contracts` were applied one at a time by a script, and the package's tests were run after each. The per-row results were appended to files that were never overwritten.

History:

- Before the change review the list had 139 rows. First run: 120 caught, 11 not caught, 8 did not compile because removing a check left a name unused. Six of the 11 showed tests that did not test what they named, which were corrected: settling at once after a reopening with no gap open; settling on evidence older than a batch although it covers the new end; a secured record named as a gap (the test had named the wrong write); an empty or reversed segment between contiguous neighbours; a scale beyond the profile when nothing is billed; revisions returned out of order. Three showed code that could not change a result and was removed (a second note of the entry into the final settlement, a second test for a missing amount) or a row that was itself a refusal. The 8 were rewritten so that they compile. After that: 137 of 139 caught.
- After the change review's corrections, 5 rows were rewritten for the changed code and 8 were added. All 147 were run again from the start: 141 caught, 5 not caught, 1 no longer matching the code. Of the 5, one row (differences returned in map order) is caught only when the map happens to iterate out of order, which the first run had hidden; the test now repeats the case forty times, and the row was caught in four consecutive runs afterwards. One showed a missing case (a revision without a price that the other result does not bill), added. One showed a bound in `Day.Add` that the range check after it already implies, removed. The affected rows were run again: all caught.
- Result: 145 of 147 caught, 2 not caught.

The 2 that are not caught, both because another condition already decides the same cases:

| Mutation | Why no test distinguishes it |
| --- | --- |
| 73, the last batch noted one knowledge point early | An evidence record never has the sequence of a batch, so "later than the batch" and "later than the point before it" select the same records |
| 86, settling on a gap record as the last evidence | A gap record that is last is unresolved, and an unresolved gap already refuses the settlement; the test of the kind is a second line of defence |

| No. | Fault introduced | Result |
| --- | --- | --- |
| 1 | Year zero accepted | Caught |
| 2 | A day that the calendar does not have accepted | Caught |
| 3 | The first day is the zero day | Caught |
| 4 | A day is before itself | Caught |
| 5 | The zero day can be moved | Caught |
| 6 | A day after the last year | Caught |
| 7 | An empty interval accepted | Caught |
| 8 | The end day counted | Caught |
| 9 | A start without terms | Caught |
| 10 | Terms on an event that has none | Caught |
| 11 | Event identifier unchecked | Caught |
| 12 | An event without a day | Caught |
| 13 | Secured without a reference | Caught |
| 14 | Secured without a covered day | Caught |
| 15 | Secured with gap members | Caught |
| 16 | Resolved gaps unchecked | Caught |
| 17 | Any reason for a gap | Caught |
| 18 | A gap without an owner | Caught |
| 19 | A gap without a follow-up | Caught |
| 20 | A gap with secured members | Caught |
| 21 | No reserve required | Caught |
| 22 | Identifier length unbounded | Caught |
| 23 | A second start accepted | Caught |
| 24 | A change to the same terms accepted | Caught |
| 25 | A change outside a running contract | Caught |
| 26 | A renewal outside a running contract | Caught |
| 27 | Suspended twice | Caught |
| 28 | A suspension loses the terms | Caught |
| 29 | Resumed while active | Caught |
| 30 | An end outside a running contract | Caught |
| 31 | An ended contract keeps its terms | Caught |
| 32 | Two events on one day | Caught |
| 33 | Malformed events in a lifecycle | Caught |
| 34 | Every lifecycle counts as ended | Caught |
| 35 | An event does not hold on its own day | Caught |
| 36 | Segments without a period | Caught |
| 37 | Segments of an invalid lifecycle | Caught |
| 38 | An event on the day after the period cuts it | Caught |
| 39 | An event on the first day cuts an empty segment | Caught |
| 40 | A renewal cuts a segment | Caught |
| 41 | Segments overlap after a cut | Caught |
| 42 | Segments depend on the order given | Caught |
| 43 | An event known before it was written | Caught |
| 44 | A withdrawal rewrites the past | Caught |
| 45 | A withdrawn event holds at its withdrawal | Caught |
| 46 | Identifiers of a write unchecked | Caught |
| 47 | A stored identifier replays any content | Caught |
| 48 | A stored identifier with other content written again | Caught |
| 49 | A replay answers the present sequence | Caught |
| 50 | A replay not marked | Caught |
| 51 | Expected sequence unchecked | Caught |
| 52 | The reserve unbounded | Caught |
| 53 | One write over the limit | Caught |
| 54 | A batch with a withdrawal admitted to the reserve | Caught |
| 55 | Any event admitted to the reserve | Caught |
| 56 | An empty batch written | Caught |
| 57 | An unknown cancellation reason | Caught |
| 58 | Batch size unbounded | Caught |
| 59 | An event identifier twice in a batch | Caught |
| 60 | The same withdrawal twice | Caught |
| 61 | An event identifier reused | Caught |
| 62 | A batch while settled | Caught |
| 63 | A withdrawal of nothing | Caught |
| 64 | An invalid lifecycle written | Caught |
| 65 | A voided contract is not final | Caught |
| 66 | A cancellation accepted where none is due | Caught |
| 67 | An end removed without a reason | Caught |
| 68 | Contracts per workspace unbounded | Caught |
| 69 | Contracts counted across workspaces | Caught |
| 70 | Withdrawals not applied | Caught |
| 71 | An event known one point early | Caught |
| 72 | An end does not open the final settlement | Caught |
| 73 | The last batch noted one point early | Not caught |
| 74 | Evidence older than the last batch settles | Caught |
| 75 | Evidence shape unchecked | Caught |
| 76 | Secured evidence in any state | Caught |
| 77 | A resolved gap resolved again | Caught |
| 78 | An unknown gap resolved | Caught |
| 79 | A secured record named as a gap | Caught |
| 80 | Secured evidence admitted to the reserve | Caught |
| 81 | A gap record not admitted to the reserve | Caught |
| 82 | Gaps never resolved | Caught |
| 83 | Settling from any state | Caught |
| 84 | Settled with an open gap | Caught |
| 85 | An ended contract settled like a voided one | Caught |
| 86 | Settled on a gap record | Not caught |
| 87 | Settled on evidence older than the reopening | Caught |
| 88 | Settled on evidence older than the last batch | Caught |
| 89 | Settled on evidence that stops short | Caught |
| 90 | A future end settled | Caught |
| 91 | Settling beyond the limit | Caught |
| 92 | Any reopening reason | Caught |
| 93 | Reopening from any state | Caught |
| 94 | A reopening not noted as an entry | Caught |
| 95 | A knowledge point of the future answered | Caught |
| 96 | The past shows the present settlement | Caught |
| 97 | The past shows later evidence | Caught |
| 98 | The past shows a later resolution | Caught |
| 99 | Reopenings not counted | Caught |
| 100 | The log aliases the caller's batch | Caught |
| 101 | The log aliases what it returns | Caught |
| 102 | Content ignores the reasons and the day | Caught |
| 103 | Any day basis | Caught |
| 104 | No rule for suspended days | Caught |
| 105 | A negative scale | Caught |
| 106 | No charge identifier | Caught |
| 107 | No workspace or contract | Caught |
| 108 | Segments that start elsewhere | Caught |
| 109 | Segments that end elsewhere | Caught |
| 110 | A gap between segments | Caught |
| 111 | An overlap of segments | Caught |
| 112 | An empty segment | Caught |
| 113 | An unknown state of a segment | Caught |
| 114 | The cover check skipped | Caught |
| 115 | Scale and mode checked only when something is billed | Caught |
| 116 | Suspended days always billed | Caught |
| 117 | Suspended days never billed | Caught |
| 118 | Ended days billed | Caught |
| 119 | More decimals than the scale | Caught |
| 120 | A missing amount billed | Caught |
| 121 | Amounts of both signs | Caught |
| 122 | Each line rounded on its own | Caught |
| 123 | The rounding mode ignored | Caught |
| 124 | The scale ignored | Caught |
| 125 | A thirty-day denominator | Caught |
| 126 | A segment one day longer | Caught |
| 127 | Lines are running totals | Caught |
| 128 | The total left at zero | Caught |
| 129 | The total of terms is their last line | Caught |
| 130 | Terms in the order they were billed | Caught |
| 131 | Results of two knowledge points not comparable | Caught |
| 132 | Results of another basis compared | Caught |
| 133 | Another revision of the profile | Caught |
| 134 | Another profile | Caught |
| 135 | Another price for the same terms | Caught |
| 136 | The correction reversed | Caught |
| 137 | The difference of terms reversed | Caught |
| 138 | Terms billed only before left out | Caught |
| 139 | Differences in map order | Caught |
| 140 | A day before the first | Caught |
| 141 | The last day cannot be reached | Caught |
| 142 | An inconsistent later result compared | Caught |
| 143 | An inconsistent earlier result compared | Caught |
| 144 | Terms listed twice in a result | Caught |
| 145 | Terms without a price in a result | Caught |
| 146 | Totals per terms that do not sum to the total | Caught |
| 147 | An end not admitted to the reserve | Caught |

# Deviations from the plan

- The entry into `pending_final` that a batch causes is not tracked on its own; "after the most recent lifecycle batch" covers it, and only a reopening is noted separately. The change review checked the argument.
- A contract without any write is not found at knowledge point 0; the plan now says so.
- The time of a record is read before the write is validated, so a refused write also reads the clock.
- `Correction` checks a result's internal consistency, which the reviewed plan did not ask for; the plan now says so.

# Provisional choices and undecided matters

The owner has not decided these; the implementation uses the labelled rules of the plan: `actual_days` as the only day basis (not conservative: ten days of 300,000 are 107,143 in a 28-day and 96,774 in a 31-day period); billing of suspended days as a required parameter without a default; cumulative rounding as the remainder allocation; a contract without an end continuing as active, with term expiry not modelled (not conservative: it bills more than a term that lapses); a re-start after an end under a new identifier; one event per day; no refund decision, only the arithmetic difference; no final grace period; `Settle` refused while any gap is unresolved; a voided contract settled without evidence; the contract time zone left to the caller.

# Limits

- The package authorizes nothing and is reachable only in process; it must not be exposed before commands behind the gate exist for it.
- It does not check that a contract, a revision, a workspace, an owner or an evidence reference exists anywhere, and it is not bound to `modules/organizations`.
- Identifiers can still resemble names; that no personal or customer text is put into one is the caller's obligation.
- A result passed to `Correction` is trusted beyond its internal consistency.
- `asOf` and `CoveredUntil` are the caller's statements and are compared with nothing but the lifecycle.
- At the write limit plus the reserve a running contract can no longer be ended in this registry; nothing but the refusal signals it.
- The number of workspaces is not bounded.
- An amount whose product with the days of the period passes the profile's bound is refused although its result would fit.
- `floor` and `ceiling` give a charge and its reversing credit that can differ by one unit.
- State is in memory and lost on restart.
- The per-row mutation results are in the session's scratch files, not in the repository.

# Independent review

The design review of 2026-11-13 returned PASS with findings (4 HIGH, 8 MEDIUM, 4 LOW) and judged the settlement section as first written to fail AUD-QA-02; the plan was rewritten. The re-review of the same day returned PASS with findings (1 HIGH, 6 MEDIUM, 5 LOW), all adopted; the text after it was not reviewed again as a design, but the change review read the final plan against the code.

The change review of the same day returned PASS with findings: no blocker, no HIGH, 2 MEDIUM and 7 LOW. It found every rule of the plan implemented, no input sequence that lets the settlement state none coexist with an ended or voided lifecycle, no path to settled for an ended contract without qualifying evidence, no change of a past knowledge point's answers, and the model's rounding correct and independent. All nine findings were handled as the plan's change review resolution lists. The corrections made after it (the consistency check in `Correction`, the bound in `Day.Add`, the clock read moved before the first change, and tests) were not reviewed again; each only adds a refusal or moves a read, and they are covered by the tests and by the mutation rows run after them.

All three reviews were read-only and executed nothing: every executed result in this record is the author's.
