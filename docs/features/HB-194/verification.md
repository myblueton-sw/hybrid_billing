---
type: VerificationRecord
title: HawkEye statement receptions and eligibility verification
description: Records what was executed and reviewed for the first model of work item W-10, with results, the corrections made on the way, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-194](https://linear.app/hybrid-billing/issue/HB-194/w-10-hawkeye-statements-with-an-eligibility-verdict-and-one-statement) adds the Go package `modules/hawkeye` from main `f838a60`: receptions of cost statements from HawkEye, at most one that is not voided per statement; an immutable header with period, supersession, mock flag, cost status, measurement tier, coverage, declared lines and totals and where that declaration comes from; supersession as a fact about statements that is never undone; team lines whose count and exact sums are compared with the declaration; and an eligibility verdict with ordered reasons under which, at any one reading, at most one reception of a source is eligible for a moment of time. `Counting` reads the eligible reception of an exact period with a sequence number of the source.

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan twice before implementation and the implemented change afterwards (see Independent review). Machine: mac_mini.

Nothing calls the package and no HawkEye is called; the HawkEye connection is not opened. The names of tiers and statuses are assumptions, since no HawkEye interface is in the planning files. `eligible` is necessary for using a power cost and not sufficient. The acceptance sentence "the same cost is not reflected twice" is not met by this package: it does not know that a statement was used, and what counts for a period can change afterwards. COM01, DC01, SIM06, AUD-AR-02, AUD-AR-10 and AUD-RT-01 are NOT_RUN. W-10 is not complete, and its design part, which asks for the deployed version, the supersession relation, the export API and authentication, stays open with the owner.

# Executed checks

Executed on 2026-11-13 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain, with `GOCACHE` set to a directory of the session because the machine's shared build cache had lost entries earlier that day. No other operating system was tested.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All nineteen packages `ok`, including `modules/hawkeye`; run after the last correction |
| `go test -count=40 ./modules/hawkeye/` | `ok`: forty consecutive runs without a failure, before the first mutation run and again on the code after the change review |

The cases of the acceptance at the level of this package (`run_test.go`): another statement identifier in the same month makes both receptions not eligible and nothing counts, and after one supersedes the other exactly one counts; a team flagged as unmeasured and reported as zero makes its reception not eligible; a list cut off at 500 and at 2000 lines of more is not eligible when the declaration is the source's, is not eligible for being derived when the adapter counted the lines it received, and the whole list reported again under a new reception is eligible; a mock neither counts nor conflicts.

Literal tests with a test clock (`open_test.go`, `run_test.go`): every refusal of a header and its limits; identical and differing headers field by field, the same header through another connection and schema version, canonical totals; one reception per statement, the occupant returned, a voided reception freeing its statement, a mock occupying one; supersession references that are not right and right ones, a correction that moves the period, a superseded statement staying so after a void and for a later reception, the second reception of a superseding statement in four variants, chains and circles of two and three, a predecessor fetched late, statements whose reception was a mock or voided; lines with every refusal, repeats, conflicts, an undeclared currency and the limit of currencies; every reason of eligibility alone, combined and in order, while open and after closing, with coverage at and below the requirement and a requirement of zero, sums at mixed scales and outside a small profile; periods that touch, nest and coincide, conflicts with open receptions and ones that can never be eligible, their list cut at the bound, removed by a void and by a supersession; `Counting` with its sequence number moving by one per write and not on repeats or refusals; `Void` with every reason from each state; pages of lines and of receptions while receptions are added; the tenant boundary; times in UTC; errors that are fixed values; and a scan of the source for floating point and for multiplication or division of the decimal primitive.

Property test (`property_test.go`, fixed seed): 300 histories of 70 steps over receptions, lines, closing, voiding and reading, with sources and periods drawn from a small set so that receptions collide. A reference model written in the test decides the outcome of every operation and, from the lines it was given, a verdict with reasons and conflicting receptions for every reception after every step; a second verdict is recomputed by the test from what `Reception`, `Lines` and `Receptions` return; both must equal `Eligibility`. Read from the registry besides: no two eligible receptions of a source overlap; `Counting` of every period returns a reception exactly when it is eligible and has that period, with the model's sequence number; being superseded is never undone; at most one reception that is not voided per statement; no statement supersedes two or is superseded by two, and no walk along supersessions returns to its start. The test fails unless thirty-one named outcomes reached their minimum, among them 500 eligible receptions, 100 supersessions, 20 supersessions followed by a void, 20 statements received again after a void, 50 conflicts removed, 20 of them by a void, and each of twelve reasons at least twenty times.

Concurrency under the race detector: eight goroutines reporting twenty lines while another closes the reception, after which the lines kept are exactly the ones accepted; 200 rounds of two receptions of one statement and two statements superseding one, of which one succeeds each, while a first close and a void race them and must agree with what the reception then shows.

Limits of these tests: the model is the same rules written a second time by the same author, and its handling of supersession follows the code's; it does not cover the limits, a sum outside the profile, energy, more than two currencies or a second workspace, and it does not count the kinds of a rejected reception apart; these rest on the literal tests. Expectations in the tables were worked out by hand.

# Corrections made on the way

- The first mutation run (162 rows) caught 155. Five of the others showed missing tests, two of them checks of an order that a map of three elements passed by chance; tests were added with more elements. Two mutants did not compile and were rewritten. All seven rows were run again and caught.
- The change review found that the model computed no reasons, that no close really raced a report, and that an assertion could not fail; all three were corrected, with nine smaller points. When the model's own verdict was added it disagreed once with the registry, for a mock whose statement was superseded; the error was in the model.

# Mutation check

162 single-line mutations of `modules/hawkeye` were applied one at a time by a Python script, and the package's tests were run after each. The per-row results were appended to files that were never overwritten.

- First run, before the change review: 155 caught; handled as described above.
- Second run, all 162 rows against the code after the change review: 162 of 162 caught. Only that run is reported below.

| No. | Fault introduced | Result |
| --- | --- | --- |
| 1 | Identifier length unbounded | Caught |
| 2 | A slash in an identifier | Caught |
| 3 | Third currency letter unchecked | Caught |
| 4 | First currency letter unchecked | Caught |
| 5 | A revision above the maximum | Caught |
| 6 | A coverage above the whole | Caught |
| 7 | Any void reason | Caught |
| 8 | Limit not required: l.Receptions > 0 | Caught |
| 9 | Limit not required: l.Lines > 0 | Caught |
| 10 | Limit not required: l.Currencies > 0 | Caught |
| 11 | Limit not required: l.Page > 0 | Caught |
| 12 | Limit not required: l.Listed > 0 | Caught |
| 13 | Limit not required: l.MaxPeriod > 0 | Caught |
| 14 | A sum not rescaled to a finer amount | Caught |
| 15 | A coarser amount not rescaled | Caught |
| 16 | Sums equal up to a power of ten | Caught |
| 17 | Numbers not reduced | Caught |
| 18 | A number below one without its leading zero | Caught |
| 19 | Every sum fits the profile | Caught |
| 20 | Construction without limits | Caught |
| 21 | A negative required coverage | Caught |
| 22 | A required coverage above the whole | Caught |
| 23 | Construction without a profile | Caught |
| 24 | Construction without a clock | Caught |
| 25 | Time going backwards | Caught |
| 26 | The registry's time kept in the clock's zone | Caught |
| 27 | A start that is not a whole second | Caught |
| 28 | An end that is not a whole second | Caught |
| 29 | A period of no length | Caught |
| 30 | Periods of any length | Caught |
| 31 | The longest period refused | Caught |
| 32 | A header without the origin of the cost | Caught |
| 33 | A header without a schema version | Caught |
| 34 | A header without a connection | Caught |
| 35 | Revision zero | Caught |
| 36 | Revisions unbounded | Caught |
| 37 | A negative coverage | Caught |
| 38 | A coverage over the whole | Caught |
| 39 | A header that declares no line | Caught |
| 40 | A header without totals | Caught |
| 41 | A supersession without a reason, or a reason without one | Caught |
| 42 | A statement superseding itself | Caught |
| 43 | A mock that supersedes | Caught |
| 44 | A malformed superseded statement | Caught |
| 45 | A correction reason that is free text | Caught |
| 46 | An unknown cost status | Caught |
| 47 | An unknown tier | Caught |
| 48 | An unknown origin of the declaration | Caught |
| 49 | A period kept in the caller's zone | Caught |
| 50 | A total in a malformed currency | Caught |
| 51 | A negative declared total | Caught |
| 52 | Declared totals kept as written | Caught |
| 53 | A header compared by the spelling of its totals | Caught |
| 54 | The currency not part of a header | Caught |
| 55 | Not part of a header: the origin of the cost | Caught |
| 56 | Not part of a header: the start of the period | Caught |
| 57 | Not part of a header: the end of the period | Caught |
| 58 | Not part of a header: the revision | Caught |
| 59 | Not part of a header: the correction reason | Caught |
| 60 | Not part of a header: the mock flag | Caught |
| 61 | Not part of a header: the cost status | Caught |
| 62 | Not part of a header: the tier | Caught |
| 63 | Not part of a header: the coverage | Caught |
| 64 | Not part of a header: the declared number of lines | Caught |
| 65 | Not part of a header: the origin of the declaration | Caught |
| 66 | Lines per reception unbounded | Caught |
| 67 | Line limit one too low | Caught |
| 68 | Declared currencies unbounded | Caught |
| 69 | Currency limit one too low | Caught |
| 70 | A reception opened again with another header | Caught |
| 71 | A second reception of a statement | Caught |
| 72 | The occupant of a statement not returned | Caught |
| 73 | A superseding statement received again with another reference | Caught |
| 74 | A statement superseded twice | Caught |
| 75 | A circle of superseding statements | Caught |
| 76 | Receptions unbounded | Caught |
| 77 | One reception over the limit | Caught |
| 78 | A named statement not superseded | Caught |
| 79 | What a statement supersedes not remembered | Caught |
| 80 | Opening time not recorded | Caught |
| 81 | A statement not occupied by its reception | Caught |
| 82 | A reception not listed under its source | Caught |
| 83 | The sequence number not raised by a reception | Caught |
| 84 | A line with a malformed team | Caught |
| 85 | A line in a malformed currency | Caught |
| 86 | A negative cost | Caught |
| 87 | Negative energy | Caught |
| 88 | A line compared by the spelling of its cost | Caught |
| 89 | The measured flag not part of a line | Caught |
| 90 | The energy not part of a line | Caught |
| 91 | A line reported again with other content | Caught |
| 92 | A line for a reception that is not open | Caught |
| 93 | A line for a superseded statement | Caught |
| 94 | A line beyond the declared number | Caught |
| 95 | Currencies of lines unbounded | Caught |
| 96 | A line kept as it was spelled | Caught |
| 97 | The sequence number not raised by a line | Caught |
| 98 | The currencies of lines not remembered | Caught |
| 99 | Closing again refused | Caught |
| 100 | Closing a reception voided while open | Caught |
| 101 | Closing a reception whose statement is superseded | Caught |
| 102 | Costs not summed | Caught |
| 103 | Unmeasured lines not noticed | Caught |
| 104 | Lines in map order | Caught |
| 105 | Closing time not recorded | Caught |
| 106 | The sequence number not raised by closing | Caught |
| 107 | Voiding with an unknown reason | Caught |
| 108 | Voided again with another reason | Caught |
| 109 | The lines of a voided reception in map order | Caught |
| 110 | Void time not recorded | Caught |
| 111 | A voided reception still occupies its statement | Caught |
| 112 | The sequence number not raised by a void | Caught |
| 113 | A voided reception conflicts | Caught |
| 114 | A mock conflicts | Caught |
| 115 | A superseded reception conflicts | Caught |
| 116 | A voided reception judged like any other | Caught |
| 117 | An open reception eligible | Caught |
| 118 | A mock eligible | Caught |
| 119 | A provisional statement eligible | Caught |
| 120 | An unmeasured tier eligible | Caught |
| 121 | Coverage below the requirement eligible | Caught |
| 122 | Coverage at the requirement refused | Caught |
| 123 | A derived declaration eligible | Caught |
| 124 | A superseded statement eligible | Caught |
| 125 | A conflict with a reception that does not contend | Caught |
| 126 | A period that ends where this one starts conflicts | Caught |
| 127 | A period that starts where this one ends conflicts | Caught |
| 128 | Only identical periods conflict | Caught |
| 129 | Conflicting receptions in the order received | Caught |
| 130 | Conflicting receptions listed without bound | Caught |
| 131 | A period conflict eligible | Caught |
| 132 | Unmeasured lines eligible | Caught |
| 133 | An incomplete list eligible | Caught |
| 134 | A declared currency without a line accepted | Caught |
| 135 | A line in an undeclared currency accepted | Caught |
| 136 | A differing sum accepted | Caught |
| 137 | Differing totals eligible | Caught |
| 138 | A sum outside the profile eligible | Caught |
| 139 | An open reception judged by its lines | Caught |
| 140 | Eligible with reasons against it | Caught |
| 141 | Counting for a malformed period | Caught |
| 142 | Counting whatever the start | Caught |
| 143 | Counting whatever the end | Caught |
| 144 | A reception that is not eligible counts | Caught |
| 145 | What counts returned without the sequence number | Caught |
| 146 | The sequence number not returned when nothing counts | Caught |
| 147 | By which statement a reception is superseded not shown | Caught |
| 148 | The view shares its declared totals | Caught |
| 149 | A page of nothing | Caught |
| 150 | A page over the limit | Caught |
| 151 | A negative position | Caught |
| 152 | The lines of an open reception read | Caught |
| 153 | A position beyond the end | Caught |
| 154 | Pages of lines not cut | Caught |
| 155 | A malformed lower bound | Caught |
| 156 | A page of no receptions | Caught |
| 157 | A page of receptions over the limit | Caught |
| 158 | The last identifier read returned again | Caught |
| 159 | Receptions in the order received | Caught |
| 160 | Pages of receptions not cut | Caught |
| 161 | The mock flag not shown in a row | Caught |
| 162 | The superseded mark not shown in a row | Caught |

# Deviations from the plan

None after the plan was amended with the change review resolution. Where the rules of the design re-review are more precise than the sentences above them, the code follows the rules. One line of defence has no test: a monotonic clock reading is removed from a period before it is checked.

# Independent review

- Design: two read-only passes on 2026-11-13 (FAIL with 4 HIGH and no redesign asked; then PASS with findings and no HIGH). See the plan's two resolution sections.
- Change: one read-only pass on 2026-11-13, PASS with findings (3 MEDIUM about tests, 9 LOW), nothing executed by the reviewer. All findings were resolved or stated. The corrections made after it were not reviewed again by a separate reviewer; they are covered by the tests and the second mutation run.

# Provisional choices awaiting the owner

Not the conservative choices, listed in the plan: `Close` accepts a reception with fewer lines than declared; the required coverage may be zero; whether a line was measured and whether the declaration is HawkEye's own are the adapter's flags; a void works as a choice between two conflicting receptions. Conservative choices that cost availability: a superseded statement never counts again, also when it was named by mistake, for which there is no exit; any overlap of periods makes every reception concerned not eligible; if HawkEye states no count and totals apart from its list, no reception is ever eligible. Assumptions: the names of cost statuses and tiers; one superseded statement per statement; one coverage figure per statement; costs that are never negative; energy in kilowatt-hours. Undecided and not chosen here: the deployed version of HawkEye, how its statements supersede one another, its export API and authentication; the required coverage; whether an estimated tier may be used; how an eligible cost reaches a billing and what keeps it from being used twice; who may void.

# Not verified

Any real HawkEye, an adapter, authentication and tenant binding of the connection, a bulk export, a process that dies, persistence, any operating system other than the one named, and everything listed as deferred in the plan. No deployment, payment, message, real account or customer data was involved.
