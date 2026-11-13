---
type: VerificationRecord
title: Shared allocation pools and runs verification
description: Records what was executed and reviewed for the first model of work item W-14, with results, the corrections made on the way, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-202](https://linear.app/hybrid-billing/issue/HB-202/w-14-shared-allocation-pools-and-runs-with-staged-chunks-held) adds the Go package `modules/sharedpool` from main `0012be3`: versions of a shared allocation pool with an amount, a quantum and either a stated capacity with a residual bearer or normalisation; runs with a purpose of showback, simulation or chargeback that receive the weights of their targets in chunks; a seal that judges only when every declared chunk is there, holds on a weight that is not known, a zero denominator or weights over the capacity, and otherwise allocates the amount in one call of the existing largest-remainder allocation, with what lies below a stated capacity as a share of its own; a recorded alternative for a zero denominator; publication, before which no share, class sum or residual can be read; and the stored invariant that at most one published chargeback run of a pool covers a moment of time. `Chargeback` reads the published chargeback run of an exact period with a sequence number of the pool.

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan before implementation and the implemented change afterwards (see Independent review). Machine: mac_mini.

Nothing calls the package. A published chargeback run stands in the place of what the requirement calls confirmed; that an authorised approver confirmed is not met here, a publication being a caller's statement of an approver and a reference. Not delivered: the fixed priority of direct attribution, tag, ratio, usage, reservation and common cost, a pool version having one basis; group rules; re-approval after a retroactive change. COM04 and COM06 hold as arithmetic identities of a run only. COM04, COM06, GH03 and GH06 are NOT_RUN. W-14 is not complete, its design part stays open with the owner, and the independent verification by a DBA and by QA that W-14 asks for remains.

# Executed checks

Executed on 2026-11-13 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain, with `GOCACHE` set to a directory of the session. No other operating system was tested.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All twenty-three packages `ok`, including `modules/sharedpool`; run on the final code |
| `go test -count=40 ./modules/sharedpool` | `ok`: forty consecutive runs without a failure, before each mutation run; the last on the final code |

The cases of the acceptance at the level of this package (`run_test.go`): five targets staged in five different orders and cuts, from one chunk to four, give the same shares, which were worked out by hand; a run with a chunk missing is not sealed; a sealed run shows no class sum and no residual and its shares cannot be read until it is published; weights of 85 against a capacity of 100 leave a residual share of 150,000 of 1,000,000 borne by the MSP; a zero denominator is held, has no share, and is settled by a recorded alternative that gives every staged target zero and the residual the whole amount.

Literal tests with a test clock (`run_test.go`, `stage_test.go`): construction; identifiers compared with the allocation's at the edges; every refusal of a pool version, a run, a chunk, a seal, a settlement, a publication and a withdrawal, with the order of error kinds, the size of a chunk judged before its weights and before the run; repeats and conflicts field by field, the order of weights in a chunk; both ways of forming the denominator; each hold and their order; a stated capacity with weights of zero; weights that use the capacity up; an amount of zero with a residual of zero; a unit left over between the residual and targets that start with a digit, a hyphen, a point, an underscore and a letter; five quanta; a run with the limit of targets, the limit of chunks and a residual; a sum the profile cannot hold, the primitive's and the allocation's bound of intermediates, each a limit error that leaves the run staging, and its exit; publication with and without an approval per purpose; chargeback runs of the same, of touching and of overlapping periods across versions and currencies; withdrawal from each state, a published run readable after it and a sealed one not; `Chargeback` with its sequence number; pages of versions, runs, weights and shares, the residual as a position; the sequence number over fourteen writes and repeats, and refusals; another workspace; a clock that goes back and one in another zone; errors that are fixed values; a scan of the source.

Property test (`property_test.go`, fixed seeds): 300 histories of 180 steps over two pools, five versions and periods that touch and overlap. A reference model written in the test with `math/big.Rat` decides the outcome of every operation and the state and hold of every run. At every publication the shares are read and checked against properties: they add up to the amount, also as the shares that are not the residual plus the residual; each is a multiple of the quantum and differs from its exact proportion by less than one quantum; a larger weight never has a smaller share; a weight of zero has a share of zero; target, bearer and weight are what was staged; the residual is there exactly when a stated capacity is not used up, with the unused capacity as its weight; the class sums of the run are the sums of the shares read. For every run sealed by allocation and published, a second registry is given the same targets shuffled and cut into another number of chunks and must return the same shares. Ten times per history: no share is readable from a run that was never published, and such a run shows no class sum; the stored shares of sealed runs that were not published add up to the amount; published chargeback runs of a pool do not overlap; `Chargeback` of every period equals the model's. The test fails unless named outcomes reached their minimum, among them 300 comparisons with a second registry, each of five states, each of three holds, each purpose published sixty times, and nineteen kinds of refusal fifteen times each, a refusal counted once per seed and target and a success per call.

Concurrency under the race detector, forty rounds: the four chunks of a run staged twice over by eight goroutines in any order while three try to seal and three read, who never see a share or a class sum, after which the shares equal those of a run staged in order; two chargeback runs of one period published at once, of which exactly one succeeds; a withdrawal against four readers of `Chargeback` and of pages of shares, who never see the run billed after the withdrawal was seen.

Limits of these tests: the model is the same rules written a second time by the same author. As the change review noted, it uses the package's own predicates for bearers, repeats the code's flags and its condition for a publication, follows the same order of judging, takes the form of a weight from a table and knows three quanta; a misreading of the plan common to both would pass. The shares are not computed a second time, and the second registry is the same code, so it finds dependence on order and cut and no defect common to both. The order of ties is pinned by the literal tests only. The property test does not reach the limit of targets of a run.

# Corrections made on the way

- The property test agreed with the registry on its first run that reached runs; before that, an error in the generator kept it from starting any run, which the thresholds showed.
- One expected value in the acceptance test was wrong by hand and was corrected against the arithmetic, not against the output alone: 1,000,000 over weights 60, 25, 10, 3 and 1 leaves one unit, which goes to the largest remainder.
- The first mutation run (214 rows) caught 201. Tests were added for four of the others; eight did not compile and were rewritten; one showed a sort before the allocation that could not change a result, which was removed.
- The change review led to no change of behavior; a type was renamed, tests were added and corrected, and the plan records one rule as built. See the plan's change review resolution.

# Mutation check

Single mutations of the four source files of `modules/sharedpool` were applied one at a time by a Python script, and the package's tests were run after each. The per-row results were appended to files that were never overwritten. The script and the result files are kept outside the repository, so the table below cannot be audited from the change alone.

- First run, 214 rows, before the change review: 201 caught; handled as described above.
- Second run, 214 rows, on the final code: 213 caught. The one not caught writes the fields of a print without their lengths. Under the present forms no two different requests then have the same print, since amounts, quanta, bearers and periods are of closed forms and no field may hold the mark that separates a weight; the lengths are kept as a defence, and no test distinguishes them.

| No. | Fault introduced | Result |
| --- | --- | --- |
| 1 | Identifier length unbounded | Caught |
| 2 | An empty identifier | Caught |
| 3 | The longest identifier refused | Caught |
| 4 | A slash in an identifier | Caught |
| 5 | A space in an identifier | Caught |
| 6 | An at sign in an identifier | Caught |
| 7 | Third currency letter unchecked | Caught |
| 8 | First currency letter unchecked | Caught |
| 9 | Second currency letter unchecked | Caught |
| 10 | A version above the maximum | Caught |
| 11 | The highest version refused | Caught |
| 12 | An amount with four places refused | Caught |
| 13 | An amount with five places | Caught |
| 14 | The residual under an identifier a caller may use | Caught |
| 15 | A quantum of ten | Caught |
| 16 | A quantum of two | Caught |
| 17 | The quantum 0.1 refused | Caught |
| 18 | The quantum 0.01 refused | Caught |
| 19 | The quantum 0.001 refused | Caught |
| 20 | The quantum 0.0001 refused | Caught |
| 21 | The quantum 1 refused | Caught |
| 22 | Any bearer | Caught |
| 23 | A residual borne by an organisation | Caught |
| 24 | An unattributed residual refused | Caught |
| 25 | A residual borne by the MSP refused | Caught |
| 26 | Limit not required: l.Pools > 0 | Caught |
| 27 | Limit not required: l.Versions > 0 | Caught |
| 28 | Limit not required: l.Runs > 0 | Caught |
| 29 | Limit not required: l.Chunks > 0 | Caught |
| 30 | Limit not required: l.ChunkWeights > 0 | Caught |
| 31 | Limit not required: l.Targets > 0 | Caught |
| 32 | Limit not required: l.Page > 0 | Caught |
| 33 | Limit not required: l.MaxPeriod > 0 | Caught |
| 34 | Limit not required: l.Intermediate > 0 | Caught |
| 35 | Construction without limits | Caught |
| 36 | Construction without a clock | Caught |
| 37 | Time going backwards | Caught |
| 38 | The registry's time kept in the clock's zone | Caught |
| 39 | A registry that was not built | Caught |
| 40 | Prints without lengths or separators | Not caught; cannot differ, see above |
| 41 | A start that is not a whole second | Caught |
| 42 | An end that is not a whole second | Caught |
| 43 | A period of no length | Caught |
| 44 | Periods of any length | Caught |
| 45 | The longest period refused | Caught |
| 46 | Negative numbers read | Caught |
| 47 | The time of a change not kept | Caught |
| 48 | A run moves no number | Caught |
| 49 | A pool without a workspace | Caught |
| 50 | A pool without an identifier | Caught |
| 51 | Version zero | Caught |
| 52 | Versions unbounded | Caught |
| 53 | A pool in a malformed currency | Caught |
| 54 | A pool with any period | Caught |
| 55 | Any quantum the profile reads | Caught |
| 56 | A quantum finer than the profile reads | Caught |
| 57 | A malformed amount | Caught |
| 58 | An amount with more than four places | Caught |
| 59 | An amount that is not a multiple of the quantum | Caught |
| 60 | An amount at the scale of the quantum refused | Caught |
| 61 | A bearer without a capacity | Caught |
| 62 | A capacity of zero | Caught |
| 63 | A capacity with any bearer | Caught |
| 64 | A period kept in the caller's zone | Caught |
| 65 | Not part of a pool: the currency | Caught |
| 66 | Not part of a pool: the start | Caught |
| 67 | Not part of a pool: the end | Caught |
| 68 | Not part of a pool: the amount | Caught |
| 69 | Not part of a pool: the quantum | Caught |
| 70 | Not part of a pool: the capacity | Caught |
| 71 | Not part of a pool: the bearer | Caught |
| 72 | A different pool under a known identity | Caught |
| 73 | Pools without limit | Caught |
| 74 | Versions without limit | Caught |
| 75 | One pool over the limit | Caught |
| 76 | One version over the limit | Caught |
| 77 | A pool version moves no number | Caught |
| 78 | A run without an identifier | Caught |
| 79 | A run of any purpose | Caught |
| 80 | A run without a chunk | Caught |
| 81 | Chunks without limit | Caught |
| 82 | The limit of chunks refused | Caught |
| 83 | A run started again with other content | Caught |
| 84 | A repeated start answered as staging | Caught |
| 85 | Runs without limit | Caught |
| 86 | One run over the limit | Caught |
| 87 | The time a run was started not kept | Caught |
| 88 | A chunk at position zero | Caught |
| 89 | An empty chunk | Caught |
| 90 | Chunks of any size | Caught |
| 91 | A chunk at its limit refused | Caught |
| 92 | A malformed target | Caught |
| 93 | A target that starts with a colon | Caught |
| 94 | A target twice in a chunk | Caught |
| 95 | A weight with any bearer | Caught |
| 96 | A malformed weight read as not known | Caught |
| 97 | A known weight kept as not known | Caught |
| 98 | The order of weights part of a chunk | Caught |
| 99 | Not part of a chunk: the target | Caught |
| 100 | Not part of a chunk: the bearer | Caught |
| 101 | Not part of a chunk: the weight | Caught |
| 102 | A changed chunk taken as a repeat | Caught |
| 103 | A chunk for a run that is not staging | Caught |
| 104 | A chunk beyond the declared number | Caught |
| 105 | The last declared chunk refused | Caught |
| 106 | A target in two chunks | Caught |
| 107 | Targets without limit | Caught |
| 108 | The limit of targets refused | Caught |
| 109 | A chunk moves no number | Caught |
| 110 | A withdrawal for any reason | Caught |
| 111 | A second reason accepted | Caught |
| 112 | A withdrawn chargeback still excludes and is billed | Caught |
| 113 | The reason of a withdrawal not kept | Caught |
| 114 | A withdrawal moves no number | Caught |
| 115 | Weights that are not known not counted | Caught |
| 116 | The self-check accepts any sum | Caught |
| 117 | Every share summed under one class | Caught |
| 118 | The residual counted among the other shares | Caught |
| 119 | No residual shown as a residual of zero | Caught |
| 120 | The residual not stored | Caught |
| 121 | Sums per class not stored | Caught |
| 122 | A seal repeated as a new seal | Caught |
| 123 | A seal of a withdrawn run | Caught |
| 124 | A seal with a chunk missing | Caught |
| 125 | A weight that is not known read as zero | Caught |
| 126 | A hold moves no number | Caught |
| 127 | A held run not marked as judged | Caught |
| 128 | A bound of the profile reported as rejected | Caught |
| 129 | Weights that use the capacity up held | Caught |
| 130 | Weights over the capacity allocated | Caught |
| 131 | A zero denominator divided | Caught |
| 132 | Weights of zero under a stated capacity held | Caught |
| 133 | A run over its capacity held as a zero denominator | Caught |
| 134 | The weight sum of a held run not shown | Caught |
| 135 | A weight shown as printed | Caught |
| 136 | Every share shown as an organisation's | Caught |
| 137 | Weights ignored | Caught |
| 138 | A residual of no weight when the capacity is used up | Caught |
| 139 | What lies below the capacity spread over the targets | Caught |
| 140 | The residual not marked | Caught |
| 141 | The residual always borne by the MSP | Caught |
| 142 | The residual shown with the whole capacity | Caught |
| 143 | The reason of a residual not kept | Caught |
| 144 | No room for the residual beside the limit of targets | Caught |
| 145 | The primitive's bound of intermediates reported as a failed self-check | Caught |
| 146 | The allocation's bound reported as a failed self-check | Caught |
| 147 | The residual joined to the last share | Caught |
| 148 | A seal moves no number | Caught |
| 149 | The denominator shown as the weight sum | Caught |
| 150 | An alternative without an approver | Caught |
| 151 | An alternative without a reference | Caught |
| 152 | An alternative with any bearer | Caught |
| 153 | A second alternative accepted | Caught |
| 154 | An alternative for another hold | Caught |
| 155 | An alternative for a run that is not held | Caught |
| 156 | A settled amount always borne by the MSP | Caught |
| 157 | The settled share not marked as the residual | Caught |
| 158 | The shares of a settled run in the order of arrival | Caught |
| 159 | A settled residual called below capacity | Caught |
| 160 | A settlement moves no number | Caught |
| 161 | A malformed approver | Caught |
| 162 | A malformed reference | Caught |
| 163 | A second approval accepted | Caught |
| 164 | A run published that is not sealed | Caught |
| 165 | A chargeback without an approver, or a showback with one | Caught |
| 166 | A chargeback without a reference, or a showback with one | Caught |
| 167 | A showback excluded by a chargeback, and excluding | Caught |
| 168 | Two chargeback runs of one period | Caught |
| 169 | Periods that touch exclude each other | Caught |
| 170 | Only identical periods exclude each other | Caught |
| 171 | Chargeback runs of other versions do not exclude | Caught |
| 172 | The approval not kept | Caught |
| 173 | A publication moves no number | Caught |
| 174 | The last version read returned again | Caught |
| 175 | Versions in map order | Caught |
| 176 | Pages of versions not cut | Caught |
| 177 | A negative position among versions | Caught |
| 178 | A page of no versions | Caught |
| 179 | A page of versions over the limit | Caught |
| 180 | A version row without its currency | Caught |
| 181 | The number of staged chunks not shown | Caught |
| 182 | The number of targets not shown | Caught |
| 183 | The number of weights not known not shown | Caught |
| 184 | The reason of a hold not shown | Caught |
| 185 | The weight sum not shown | Caught |
| 186 | The amount not shown | Caught |
| 187 | The denominator not shown | Caught |
| 188 | The approval not shown | Caught |
| 189 | That a run was published not shown | Caught |
| 190 | The reason of a withdrawal not shown | Caught |
| 191 | The view shares the alternative | Caught |
| 192 | Sums of a sealed run shown before publication | Caught |
| 193 | The view shares its sums | Caught |
| 194 | The reason of a residual not shown | Caught |
| 195 | The shares of a sealed run read before publication | Caught |
| 196 | A page of shares without the state | Caught |
| 197 | A page of shares without the purpose | Caught |
| 198 | A page of shares without the sequence number | Caught |
| 199 | The last share read returned again | Caught |
| 200 | More shares not told | Caught |
| 201 | The last weight read returned again | Caught |
| 202 | More weights not told | Caught |
| 203 | A chunk that is not staged read | Caught |
| 204 | The last run read returned again | Caught |
| 205 | Runs in map order | Caught |
| 206 | Pages of runs not cut | Caught |
| 207 | A run row without its purpose | Caught |
| 208 | A run row without its state | Caught |
| 209 | Chargeback for a malformed period | Caught |
| 210 | Chargeback whatever the start | Caught |
| 211 | Chargeback whatever the end | Caught |
| 212 | What is billed returned without its version | Caught |
| 213 | What is billed returned without the sequence number | Caught |
| 214 | The sequence number not returned when nothing is billed | Caught |

# Deviations from the plan

Recorded in the plan's change review resolution: the size of a chunk is judged before the run is looked up, and the staged entries are not sorted before the allocation.

# Independent review

- Design: one read-only pass on 2026-11-13, PASS with findings (no HIGH, 12 MEDIUM, 10 LOW), all adopted as rules in the plan before implementation. The rules written after it were not read again by a separate reviewer before implementation; the change review read them against the code.
- Change: one read-only pass on 2026-11-13, PASS with findings (1 MEDIUM, 9 LOW), nothing executed by the reviewer. All findings were resolved or stated. The corrections made after it were not reviewed again by a separate reviewer; they are covered by the tests and the second mutation run.
- Not done: the independent verification by a DBA and by QA that W-14 asks for.

# Provisional choices awaiting the owner

Provisional defaults: a residual is borne by the MSP or left unattributed unless the pool is opened as normalising; a zero denominator is held and its alternative gives the whole amount to one bearer; an amount must be a whole multiple of the quantum; purposes do not differ in arithmetic; one chargeback per pool and overlapping period, whatever the currency; on equal remainders a unit left over goes to the residual before any target named with a letter and after any named with a digit. Not conservative, stated: a pool opened as normalising spreads unused capacity over its targets; a stated capacity with weights of zero gives the whole amount to the residual without a hold; nothing relates two pools; a weight of zero is a stated zero. Costs to availability: one weight that is not known holds the whole run; a missing chunk keeps every target waiting; a chargeback run is published without its shares having been read, the route for a review being a simulation run of the same weights; a run whose seal ends on a bound can only be withdrawn; with an inconsistent source the first chunk to arrive is kept. Known divergence: `modules/costpool` makes the caller name unused capacity as a target, this package makes the residual itself. Undecided and not chosen here: the denominator, the assignment of a residual and of a unit of rounding, the boundaries of a pool, the order of allocation steps, who may publish and withdraw, what a workspace is in relation to the MSP and its customers.

# Not verified

A failed self-check, which no test produces; exposure of shares to a customer-facing read, which must not happen; processes as workers and their failure; persistence; any operating system other than the one named; and everything listed as deferred in the plan. No deployment, payment, message, real account or customer data was involved.
