---
type: VerificationRecord
title: Reconciliation runs and billing readiness verification
description: Records what was executed and reviewed for the first model of work item W-09, with results, the corrections made on the way, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-192](https://linear.app/hybrid-billing/issue/HB-192/w-09-reconciliation-runs-with-explained-residuals-and-billing) adds the Go package `modules/reconcile` from main `d4e9529`: runs of three billing bases, each with an exact set of fields and a subject key that at most one run that is not voided holds in a workspace; for a provider document a total, internal parts of the billed cost kind and typed explanation lines whose items are linked once among the runs that count, with rounding bounded per currency; a residual computed exactly and the verdict `explained` only at zero; revisions of a document that supersede earlier ones from the moment a run for them is opened; and a readiness verdict with reasons and a separate cost state, `determined`, `declared` or `undetermined`.

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan twice before implementation and the implemented change afterwards (see Independent review). Machine: mac_mini.

Nothing calls the package and it reads no document: a caller posts the figures. `explained` means that the posted lines sum to the difference, not that any line is the real cause; any line of another category than rounding can balance a run. REC01, REC02, COM02, FD07 and AUD-AR-03 are NOT_RUN. W-09 is not complete, and its design part, which asks for the reconciliation scope, the rounding policy and the evidence an independent price needs, stays open with the owner.

# Executed checks

Executed on 2026-11-13 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain, with `GOCACHE` set to a directory of the session because the machine's shared build cache had lost entries earlier that day. No other operating system was tested.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All eighteen packages `ok`, including `modules/reconcile`; run again after the last correction |
| `go test -count=40 ./modules/reconcile/` | `ok`: forty consecutive runs without a failure, before the first mutation run and again on the code after the change review |

Literal tests with a test clock (`open_test.go`, `run_test.go`): construction with every limit and rounding maximum; the exact field set of each basis, every field of a document on the bases without one and each malformed or foreign field per basis; refusals after the shape; one run per subject key on each basis, conflicts and repeats field by field, the same key in two workspaces, a voided run freeing its key and an identical open of it answered afterwards; figures kept in canonical form; parts with every other cost kind, another currency, a source used twice, a part beyond the declared number, zero and negative parts; lines of every category, both item types, every refusal, an item linked twice in one and in two categories, rounding at, below and above the allowed amount with either sign and in three currencies; items linked once among runs that count, across documents, across currencies of one revision, across workspaces, after a void and after supersession; nine residuals with their sums per category; sums outside the profile with the order of reasons; revisions superseding while the later run is open, unexplained and voided, lower revisions refused, what `Current` lists; readiness of every basis with the cost state, the case of the acceptance among them, a complete quantity that is `ready` with an `undetermined` cost; a voided run of every basis making no claim; pages of parts and lines; the tenant boundary; times; errors that are fixed values; a scan of the source for floating point.

Property test (`property_test.go`, fixed seed): 400 histories of 90 steps over opening runs of every basis, posting, explaining, balancing, closing, voiding and reading readiness. A reference model written in the test, with 64-bit integers at one scale and its own spelling of amounts, decides every outcome. Read from the registry after every step, independent of the model: at most one run that is not voided per subject key; no voided or superseded run is `ready` or has a cost other than `undetermined`; no item is linked by two runs that count; and for every closed run the residual recomputed from the pages of parts and lines equals the stored one and is zero exactly when the verdict is `explained`. The test fails unless twenty-four named outcomes were each reached at least three times and the generator constructed at least 200 balanced runs and 60 near misses of one smallest unit.

Metamorphic test: 300 rounds in which a run, the same run with one part split in two and its parts and lines in another order, and the same run opened again after a void give the same sums, residual and verdict, one in five of them one smallest unit off balance.

Scale: 50,000 parts at mixed scales and signs, posted in random order, whose sum equals the generator's and which are read back in order.

Concurrency under the race detector: eight goroutines posting the same twenty parts while closing, explaining and reading; 200 rounds of two runs opened for one subject key, of which one succeeds, followed by a close, a void and a later revision at once.

Limits of these tests: the model is the same rules written a second time by the same author, and the order of its checks and its readiness follow the code's; it does not cover malformed input, limits, a currency without a rounding maximum, sums outside the profile, `Current`, `Holder`, pages, a second workspace or times, which rest on the literal tests. The read of readiness that races a later revision is exercised for the race detector only. Expectations in the tables were worked out by hand.

# Corrections made on the way

- The first property run agreed with the model everywhere but constructed too few balanced runs; the generator was steered towards runs that can still be written to.
- The first mutation run (164 rows) caught 150. Eight of the others showed tests that were missing; tests were added. One showed a check that could change no result, which was removed and is stated in the plan; five did not compile and were rewritten.
- The change review led to two changes of behaviour: a subject without a document that names a currency is invalid in `Holder`, and a run's own figures are kept in canonical form. It also found a test that paired reasons with runs in the order of a map, which made mutation rows not reproducible; it walks a fixed list now.

# Mutation check

168 single-line mutations of `modules/reconcile` were applied one at a time by a Python script, and the package's tests were run after each. The per-row results were appended to files that were never overwritten.

- First run, 164 rows, before the change review: 150 caught; handled as described above.
- Second run, all 168 rows against the code after the change review: 168 of 168 caught. Only that run is reported below.

| No. | Fault introduced | Result |
| --- | --- | --- |
| 1 | Identifier length unbounded | Caught |
| 2 | A slash in an identifier | Caught |
| 3 | Third currency letter unchecked | Caught |
| 4 | First currency letter unchecked | Caught |
| 5 | A revision above the maximum | Caught |
| 6 | Any cost kind known | Caught |
| 7 | A sale amount is malformed instead of refused | Caught |
| 8 | Categories in another order | Caught |
| 9 | Any void reason | Caught |
| 10 | Limit not required: runs | Caught |
| 11 | Limit not required: parts | Caught |
| 12 | Limit not required: lines | Caught |
| 13 | Limit not required: page | Caught |
| 14 | Limit not required: list bound | Caught |
| 15 | A sum not rescaled to a finer amount | Caught |
| 16 | A coarser amount not rescaled | Caught |
| 17 | Negative rounding of any size | Caught |
| 18 | Rounding exactly at the allowed amount refused | Caught |
| 19 | Numbers not reduced | Caught |
| 20 | The sign of a negative number lost | Caught |
| 21 | A number below one without its leading zero | Caught |
| 22 | Every sum fits the profile | Caught |
| 23 | Construction without limits | Caught |
| 24 | Construction without a profile | Caught |
| 25 | Construction without a clock | Caught |
| 26 | A rounding maximum for a malformed currency | Caught |
| 27 | A negative rounding maximum | Caught |
| 28 | Time going backwards | Caught |
| 29 | A document run for a customer | Caught |
| 30 | A document run with a period | Caught |
| 31 | A document run with a charge | Caught |
| 32 | A document run with a pool | Caught |
| 33 | An issuer on a basis without a document | Caught |
| 34 | An account on a basis without a document | Caught |
| 35 | A document on a basis without one | Caught |
| 36 | A revision on a basis without a document | Caught |
| 37 | Revision zero | Caught |
| 38 | Revisions unbounded | Caught |
| 39 | A document run without a currency | Caught |
| 40 | One run per revision whatever the currency | Caught |
| 41 | A price run with a pool | Caught |
| 42 | A price run without a charge | Caught |
| 43 | One price run per customer and period whatever the charge | Caught |
| 44 | A pool run with a charge | Caught |
| 45 | A pool run without a pool | Caught |
| 46 | One pool run per customer whatever the period | Caught |
| 47 | A run without a workspace | Caught |
| 48 | A run with a malformed identifier | Caught |
| 49 | A document run with a quantity flag | Caught |
| 50 | A document run with a cost | Caught |
| 51 | A negative allowed rounding | Caught |
| 52 | A document total without a reference to its input | Caught |
| 53 | A run that declares no part | Caught |
| 54 | A repeat compared by the spelling of the total | Caught |
| 55 | The allowed rounding not part of what was opened | Caught |
| 56 | The input not part of what was opened | Caught |
| 57 | A price run without quantity evidence | Caught |
| 58 | A price run without price evidence | Caught |
| 59 | A currency or evidence without a cost | Caught |
| 60 | A negative cost of a price run | Caught |
| 61 | A cost of a price run without evidence | Caught |
| 62 | The quantity flag not part of what was opened | Caught |
| 63 | A negative pool cost | Caught |
| 64 | A pool cost without pool evidence | Caught |
| 65 | A pool cost without allocation evidence | Caught |
| 66 | Nothing is ever superseded | Caught |
| 67 | A document line unique in the whole workspace | Caught |
| 68 | A document line named once per currency | Caught |
| 69 | Releasing an item another run took since | Caught |
| 70 | A declared number of parts that can never be met | Caught |
| 71 | Part limit one too low | Caught |
| 72 | A run opened again with other content | Caught |
| 73 | A second run for a subject key | Caught |
| 74 | A revision below the highest opened | Caught |
| 75 | Rounding for a currency without a maximum | Caught |
| 76 | Allowed rounding over the maximum | Caught |
| 77 | Runs unbounded | Caught |
| 78 | One run over the limit | Caught |
| 79 | Opening time not recorded | Caught |
| 80 | A run without a document left open | Caught |
| 81 | The highest revision never raised | Caught |
| 82 | A superseded run keeps its items | Caught |
| 83 | Runs not listed under their document | Caught |
| 84 | A subject key not taken | Caught |
| 85 | Writing to a closed run | Caught |
| 86 | Writing to a superseded run | Caught |
| 87 | An amount in another currency | Caught |
| 88 | A part with a malformed identifier | Caught |
| 89 | A part without a source | Caught |
| 90 | A part compared by its spelling | Caught |
| 91 | The source not part of a part | Caught |
| 92 | A part posted again with other content | Caught |
| 93 | One source under two parts | Caught |
| 94 | A cost kind other than billed accepted | Caught |
| 95 | A part beyond the declared number | Caught |
| 96 | A part stored as it was spelled | Caught |
| 97 | A used source not remembered | Caught |
| 98 | A line of an unknown category | Caught |
| 99 | A line without evidence | Caught |
| 100 | A line of zero | Caught |
| 101 | A rounding line with an item | Caught |
| 102 | A line with an unknown item type | Caught |
| 103 | A line without an item | Caught |
| 104 | The evidence not part of a line | Caught |
| 105 | A line recorded again with other content | Caught |
| 106 | A second rounding line | Caught |
| 107 | Rounding above the allowed amount | Caught |
| 108 | An item linked twice | Caught |
| 109 | Lines unbounded | Caught |
| 110 | One line over the limit | Caught |
| 111 | A line stored as it was spelled | Caught |
| 112 | The rounding line not remembered | Caught |
| 113 | A linked item not remembered | Caught |
| 114 | Closing again refused | Caught |
| 115 | Closing a superseded run | Caught |
| 116 | Closing with parts missing | Caught |
| 117 | Closing a run voided while open | Caught |
| 118 | Parts not summed | Caught |
| 119 | Category sums not added | Caught |
| 120 | Lines per category not counted | Caught |
| 121 | Lines not summed | Caught |
| 122 | The internal side added to the residual | Caught |
| 123 | Explanations not taken from the residual | Caught |
| 124 | A positive residual called explained | Caught |
| 125 | A negative residual called explained | Caught |
| 126 | Closing time not recorded | Caught |
| 127 | Voiding with an unknown reason | Caught |
| 128 | Voided again with another reason | Caught |
| 129 | Void time not recorded | Caught |
| 130 | A voided run keeps its subject key | Caught |
| 131 | A voided run keeps its items | Caught |
| 132 | An internal sum outside the profile ready | Caught |
| 133 | A category sum outside the profile ready | Caught |
| 134 | A voided run makes a claim | Caught |
| 135 | A pool cost called determined | Caught |
| 136 | An incomplete quantity ready | Caught |
| 137 | A declared cost of a price run not shown | Caught |
| 138 | A price run without a cost shown as declared | Caught |
| 139 | An open run judged by a residual | Caught |
| 140 | An unexplained run ready | Caught |
| 141 | A superseded run ready | Caught |
| 142 | Sums outside the profile ready | Caught |
| 143 | A cost determined without being ready | Caught |
| 144 | Ready with reasons against it | Caught |
| 145 | Supersession not shown | Caught |
| 146 | A category whose lines cancel not shown | Caught |
| 147 | A page of nothing | Caught |
| 148 | A page over the limit | Caught |
| 149 | A negative position | Caught |
| 150 | Parts of a run without a document read | Caught |
| 151 | Parts of an open run read | Caught |
| 152 | Parts in map order | Caught |
| 153 | Lines in map order | Caught |
| 154 | A position beyond the end | Caught |
| 155 | Pages not cut | Caught |
| 156 | A holder asked without a workspace | Caught |
| 157 | An unknown document has a current view | Caught |
| 158 | Voided runs listed as current | Caught |
| 159 | Superseded runs listed as current | Caught |
| 160 | Current runs in the order opened | Caught |
| 161 | Superseded runs ordered by identifier alone | Caught |
| 162 | The total of superseded runs not shown | Caught |
| 163 | Superseded runs listed without bound | Caught |
| 164 | A currency on the subject of a price run | Caught |
| 165 | A currency on the subject of a pool run | Caught |
| 166 | The figures of a document run kept as written | Caught |
| 167 | A pool cost kept as written | Caught |
| 168 | Two reasons in the other order | Caught |

# Deviations from the plan

None after the plan was amended with the change review resolution. Where the rules of the design re-review are more precise than the sentences above them, the code follows the rules.

# Independent review

- Design: two read-only passes on 2026-11-13 (FAIL with 5 HIGH and no redesign asked; then PASS with findings and no HIGH). See the plan's two resolution sections.
- Change: one read-only pass on 2026-11-13, PASS with findings (9 LOW), nothing executed by the reviewer. All findings were resolved or stated. The corrections made after it were not reviewed again by a separate reviewer; they are covered by the tests and the second mutation run.

# Provisional choices awaiting the owner

Not the conservative choices, listed in the plan: a run is `explained`, and then `ready` with a `determined` cost, when lines of any category sum to the difference, without any check of a line against its evidence; the allowed rounding is chosen by the caller inside the maximum; the number of parts is the caller's own declaration; runs of `independent_price` and `internal_cost` are `ready` on references alone; the completeness of a quantity is a caller's flag; the basis is a caller's declaration; the document total is not checked to be an eligible value. Assumptions: the three billing bases, since the planning files name none; only a billed cost is balanced against a document; the six categories; revisions as numbers. Undecided and not chosen here: the reconciliation scope of a document, the rounding policy, the evidence an independent unit price needs; whether one document may be balanced in several runs per shared pool; how a run is linked to the billing of a customer and period; who may void.

# Not verified

Real documents and real costs, the internal side computed from normalized rows, a process that dies, persistence, any operating system other than the one named, and everything listed as deferred in the plan. A revision opened too high by mistake supersedes a document's real revisions for good and has no exit here. No deployment, payment, message, real account or customer data was involved.
