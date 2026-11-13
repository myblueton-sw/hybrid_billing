---
type: VerificationRecord
title: On-premises cost pool versions verification
description: Records what was executed and reviewed for the first model of work item W-12, with results, the corrections made on the way, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-198](https://linear.app/hybrid-billing/issue/HB-198/w-12-on-premises-cost-pool-versions-with-components-an-explicit) adds the Go package `modules/costpool` from main `f5cf009`: versions of an on-premises cost pool with an immutable header; assets whose acquisition price is kept and enters no sum; cost components of six kinds with an amount that may be not known, an asset, evidence, a statement of inclusion in a flat fee and, for power, a model with a reported grade; one allocation basis per version whose weights must add up to the stated capacity and whose idle targets are not borne by an organisation; a recorded approval that sums the pool exactly, refuses a total that is not a whole multiple of the quantum, allocates it with the existing largest-remainder allocation and checks that the shares add up to the total; supersession of a version with the same period and currency; and a standing of `current` or `not_current`, with the stored invariant that the current versions of a pool do not overlap. `Counting` reads the current version of an exact period with a sequence number of the pool.

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan twice before implementation and the implemented change afterwards (see Independent review). Machine: mac_mini.

Nothing calls the package, and no inventory or meter is read. A recorded approval is a caller's statement of an approver and a reference: that the customer approved, which HB-12 makes the precondition of an internal cost, is not met here. The acceptance sentence about a separate sale of 1,000,000 won is not violated and not met: the package has no input or output for a sale. `current` is necessary for using a pool's shares and not sufficient. DC01, COM06, SIM06 and AUD-AR-10 are NOT_RUN. W-12 is not complete, its design part (how a pool is split, idle capacity, the approval policy) stays open with the owner, and the independent verification by a DBA and by QA that W-12 asks for remains.

# Executed checks

Executed on 2026-11-13 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain, with `GOCACHE` set to a directory of the session. No other operating system was tested.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All twenty-one packages `ok`, including `modules/costpool`; run after the last correction |
| `go test -count=40 ./modules/costpool` | `ok`: forty consecutive runs without a failure on the code after the change review, before the third mutation run. An earlier run of forty failed, see Corrections |

The cases of the acceptance at the level of this package (`approve_test.go`): a pool of 1,200,000 won from eight components with weights 60, 20 and 20 gives 720,000, 240,000 and 240,000, the last an idle target borne by the MSP; an asset with a purchase price of 20,000,000 and a depreciation of 400,000 is summed once; power included in a flat facility fee and an operating cost included in a lease add nothing; a depreciation whose amount is not known and an asset without its component each refuse the record. The weights are inferred from the amounts; the source chapter is not in the planning files read.

Literal tests with a test clock (`open_test.go`, `parts_test.go`, `approve_test.go`): construction and the rules for identifiers, compared with the allocation's at the edges, and for currencies; every refusal of a header, an asset, a component with its model, a basis, an approval and a withdrawal, malformed apart from rejected, with the order of error kinds; repeats and conflicts field by field, texts compared as texts, negative zero, the order of targets; the limits of pools, versions, assets, components and targets; the figures of versions with every kind and grade; thirteen cases of quantum, amounts and weights with the shares of each, a total that is not a multiple, a total of zero and a unit left over; a total the profile cannot hold, the primitive's and the allocation's bound of intermediates, each a limit error that writes nothing; an amount of five places under a profile that reads six; supersession, its refusals at opening, the second of two drafts, permanence after a withdrawal, a withdrawn version named again and the period free again; overlapping, touching and nested periods against a current version, with what `Version` tells a draft; `Counting` with its sequence number, malformed periods and a pool that does not exist; every page; the tenant boundary; times in UTC; errors that are fixed values; and a scan of the source.

Property test (`property_test.go`, `model_test.go`, fixed seeds): 300 histories of 120 steps over two workspaces and two pools with periods drawn from five that touch and overlap. A reference model written in the test with `math/big.Rat` decides the outcome of every operation; seven times per history, for every version, the view, the standing, what stands against an approval, the basis with its pages of targets, the figures and the sums per asset are compared with the model, and the standing is derived a second time from the view alone. At every recorded approval the shares are read and checked against properties: they add up to the model's total; each is a multiple of the quantum and differs from its exact proportion by less than one quantum; a larger weight never has a smaller share; a weight of zero has a share of zero; equal weights differ by at most one quantum; identifier, bearer, weight and idle mark are the basis's. Read from the registry besides: no two current versions of a pool overlap; `Counting` of every period returns a version exactly when it is current and has that period, with the model's sequence number; being superseded is never undone; a superseded version names its superseder and the superseder names it. The test fails unless named outcomes reached their minimum, among them 600 recorded approvals, 60 supersessions, each standing, and twenty-six kinds of refusal on at least fifteen different versions each.

A version with 300 assets, 900 components and 200 targets has its approval recorded; every share lies between 167 and 168 and they add up to the total.

Concurrency under the race detector, forty rounds: two versions naming the same one recorded at once, of which one succeeds; two overlapping versions recorded at once, of which one succeeds; one component reported by eight goroutines and stored once; readers of what counts throughout; then a withdrawal against four readers of standing and of what counts, who never see `current` after `withdrawn`. The order of the record of a version and its components is fixed by gates in two of three rounds, record first in one and components first in the other, and free in the third, whose outcome is not required.

Limits of these tests: the model is the same rules written a second time by the same author. As the change review noted, it uses the package's own lists of kinds and bearers, takes the form of an amount from a table and not from a parser, follows the code's order of refusals, and shares the formula for overlapping periods; it does not judge identifiers, currencies, bounds of the profile or a limit error at the record. Independent in it are the multiple test, all sums, the printing of numbers and the keys of repeats. The shares are not computed a second time.

# Corrections made on the way

- The property test agreed with the registry on its first run; the generator was steered several times towards outcomes it reached too rarely, and one error was in the generator itself, which withdrew a version whenever its plan was exhausted.
- The first mutation run (269 rows) caught 250. Eight of the others showed missing tests or assertions, which were added, among them a model kept in the caller's zone, two power components of one grade, a weight shown as written and a page that shared its targets. Three showed checks that could not be reached and were removed from the code: a version number below one, a basis without targets and a version overlapping itself. One row could not differ from the code, a malformed weight, which fails the addition anyway, and was removed. Nine did not compile and were rewritten; one of those then survived the second run, the rule of four places, which a test under a profile reading six places now covers.
- A run of forty repetitions failed in the concurrency test: a minimum number of rounds in which the record lost a race depended on scheduling. The order is now fixed by gates in two of three rounds.
- The change review led to two changes that a caller can see, the time of change of a superseded version and the field that says whether the named version is superseded: see the plan's change review resolution. One mutant written after it was by mistake equal to the code and was reported as surviving; it was corrected and run again.

# Mutation check

Single mutations of the four source files of `modules/costpool` were applied one at a time by a Python script, and the package's tests were run after each. The per-row results were appended to files that were never overwritten. The script and the result files are kept outside the repository, so the table below cannot be audited from the change alone.

- First run, 269 rows: 250 caught; handled as described above.
- Second run, 265 rows, before the change review: 264 caught, one survivor handled as described.
- Third run, 267 rows, all against the code after the change review: 266 caught and one row that equalled the code; that row was corrected and run again and caught. The table reports the third run with that row's second result.

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
| 13 | A quantum of ten | Caught |
| 14 | A quantum of two | Caught |
| 15 | Any kind of asset | Caught |
| 16 | Any kind of component | Caught |
| 17 | Any bearer | Caught |
| 18 | The quantum 0.1 refused | Caught |
| 19 | The quantum 0.01 refused | Caught |
| 20 | The quantum 0.001 refused | Caught |
| 21 | The quantum 0.0001 refused | Caught |
| 22 | The quantum 1 refused | Caught |
| 23 | Limit not required: l.Pools > 0 | Caught |
| 24 | Limit not required: l.Versions > 0 | Caught |
| 25 | Limit not required: l.Assets > 0 | Caught |
| 26 | Limit not required: l.Components > 0 | Caught |
| 27 | Limit not required: l.Targets > 0 | Caught |
| 28 | Limit not required: l.Page > 0 | Caught |
| 29 | Limit not required: l.Listed > 0 | Caught |
| 30 | Limit not required: l.MaxPeriod > 0 | Caught |
| 31 | Limit not required: l.Intermediate > 0 | Caught |
| 32 | Construction without limits | Caught |
| 33 | Construction without a clock | Caught |
| 34 | Time going backwards | Caught |
| 35 | A registry that was not built | Caught |
| 36 | Prints without lengths or separators | Caught |
| 37 | A start that is not a whole second | Caught |
| 38 | An end that is not a whole second | Caught |
| 39 | A period of no length | Caught |
| 40 | Periods of any length | Caught |
| 41 | The longest period refused | Caught |
| 42 | Negative numbers read | Caught |
| 43 | A PUE of one refused | Caught |
| 44 | A superseded version current | Caught |
| 45 | A withdrawn version current | Caught |
| 46 | Periods that touch overlap | Caught |
| 47 | Only identical periods overlap | Caught |
| 48 | The time of a change not kept | Caught |
| 49 | The sequence number never moves | Caught |
| 50 | A header without a workspace | Caught |
| 51 | A header without a pool | Caught |
| 52 | A header without an owner | Caught |
| 53 | Versions unbounded | Caught |
| 54 | A header in a malformed currency | Caught |
| 55 | Any quantum the profile reads | Caught |
| 56 | A quantum finer than the profile reads | Caught |
| 57 | A header with any period | Caught |
| 58 | Superseding a negative version | Caught |
| 59 | Superseding itself | Caught |
| 60 | A period kept in the caller's zone | Caught |
| 61 | Not part of a header: the currency | Caught |
| 62 | Not part of a header: the start | Caught |
| 63 | Not part of a header: the end | Caught |
| 64 | Not part of a header: the owner | Caught |
| 65 | Not part of a header: the quantum | Caught |
| 66 | Not part of a header: what it supersedes | Caught |
| 67 | A different header under a known identity | Caught |
| 68 | A repeated header answered as a draft | Caught |
| 69 | Naming a version that does not exist | Caught |
| 70 | Naming a version without a recorded approval | Caught |
| 71 | Naming a version that is superseded | Caught |
| 72 | Superseding in another currency | Caught |
| 73 | Superseding with another start | Caught |
| 74 | Superseding with another end | Caught |
| 75 | Pools without limit | Caught |
| 76 | Versions without limit | Caught |
| 77 | One pool over the limit | Caught |
| 78 | One version over the limit | Caught |
| 79 | The time of opening not kept | Caught |
| 80 | An asset without an identifier | Caught |
| 81 | An asset of any kind | Caught |
| 82 | An asset with any method | Caught |
| 83 | A malformed acquisition price | Caught |
| 84 | Not part of an asset: the kind | Caught |
| 85 | Not part of an asset: the method | Caught |
| 86 | Not part of an asset: the price | Caught |
| 87 | A changed asset taken as a repeat | Caught |
| 88 | An asset for a version that is not a draft | Caught |
| 89 | Assets without limit | Caught |
| 90 | One asset over the limit | Caught |
| 91 | An asset does not move the sequence number | Caught |
| 92 | A component without an identifier | Caught |
| 93 | A component of any kind | Caught |
| 94 | A component without evidence | Caught |
| 95 | A component with a malformed asset | Caught |
| 96 | An amount with more than four places | Caught |
| 97 | A malformed amount | Caught |
| 98 | A known amount kept as not known | Caught |
| 99 | An amount required | Caught |
| 100 | A depreciation or lease without an asset | Caught |
| 101 | A malformed container | Caught |
| 102 | A component included in itself | Caught |
| 103 | A component of any kind included | Caught |
| 104 | A licence cannot be included | Caught |
| 105 | Power cannot be included | Caught |
| 106 | Operations cannot be included | Caught |
| 107 | Not part of a component: the kind | Caught |
| 108 | Not part of a component: the amount | Caught |
| 109 | Not part of a component: the asset | Caught |
| 110 | Not part of a component: the evidence | Caught |
| 111 | Not part of a component: the container | Caught |
| 112 | A model on another kind | Caught |
| 113 | Power without a model | Caught |
| 114 | A model of any grade | Caught |
| 115 | A model with any period | Caught |
| 116 | A stated grade without a source | Caught |
| 117 | A malformed source for an unknown grade | Caught |
| 118 | A stated grade without a PUE | Caught |
| 119 | A PUE below one for an unknown grade | Caught |
| 120 | A PUE below one | Caught |
| 121 | An estimated grade without source or PUE | Caught |
| 122 | A model period kept in the caller's zone | Caught |
| 123 | A stored component shares the caller's model | Caught |
| 124 | Not part of a component: the grade | Caught |
| 125 | Not part of a component: the source | Caught |
| 126 | Not part of a component: the PUE | Caught |
| 127 | Not part of a component: the start of the model | Caught |
| 128 | Not part of a component: the end of the model | Caught |
| 129 | A changed component taken as a repeat | Caught |
| 130 | A component for a version that is not a draft | Caught |
| 131 | A depreciation for a leased asset | Caught |
| 132 | A second depreciation or lease of an asset | Caught |
| 133 | A model that starts before the version | Caught |
| 134 | A model that ends after the version | Caught |
| 135 | A model that ends with the version refused | Caught |
| 136 | Included in a component of any kind | Caught |
| 137 | Included in a lease refused | Caught |
| 138 | Included in a facility fee refused | Caught |
| 139 | Components without limit | Caught |
| 140 | One component over the limit | Caught |
| 141 | That an asset has its component not remembered | Caught |
| 142 | A component does not move the sequence number | Caught |
| 143 | A basis without an identifier | Caught |
| 144 | A basis with any measure | Caught |
| 145 | A capacity of zero | Caught |
| 146 | Targets without limit | Caught |
| 147 | The limit of targets refused | Caught |
| 148 | A basis shares and sorts the caller's targets | Caught |
| 149 | Targets in the caller's order | Caught |
| 150 | A target without an identifier | Caught |
| 151 | A target twice | Caught |
| 152 | A target with any bearer | Caught |
| 153 | Idle capacity borne by an organisation | Caught |
| 154 | Not part of a basis: a target's identifier | Caught |
| 155 | Not part of a basis: a bearer | Caught |
| 156 | Not part of a basis: a weight | Caught |
| 157 | Not part of a basis: the idle mark | Caught |
| 158 | Not part of a basis: its identifier | Caught |
| 159 | Not part of a basis: the measure | Caught |
| 160 | Not part of a basis: the capacity | Caught |
| 161 | Weights below the capacity | Caught |
| 162 | Weights above the capacity | Caught |
| 163 | Another basis taken as a repeat | Caught |
| 164 | A basis for a version that is not a draft | Caught |
| 165 | A basis does not move the sequence number | Caught |
| 166 | An included component summed | Caught |
| 167 | An unknown amount read as zero | Caught |
| 168 | Licences left out of the total | Caught |
| 169 | A kind sum holds the last amount | Caught |
| 170 | The pool-level sum not marked | Caught |
| 171 | Pool-level components not summed | Caught |
| 172 | An asset sum holds the last amount | Caught |
| 173 | A grade sum holds the last amount | Caught |
| 174 | Assets without their component not counted | Caught |
| 175 | The named version counted as overlapping | Caught |
| 176 | A version that is not current overlaps | Caught |
| 177 | Every current version overlaps | Caught |
| 178 | Overlapping versions in map order | Caught |
| 179 | An approval without an approver | Caught |
| 180 | An approval without a reference | Caught |
| 181 | A second approval accepted | Caught |
| 182 | An approval for a withdrawn draft | Caught |
| 183 | An approval without a summed component | Caught |
| 184 | An approval with an unknown amount | Caught |
| 185 | An approval with an asset without its component | Caught |
| 186 | A version superseded twice | Caught |
| 187 | An approval that overlaps a current version | Caught |
| 188 | A bound of the profile reported as rejected | Caught |
| 189 | A total that is not a multiple of the quantum | Caught |
| 190 | A total at the scale of the quantum refused | Caught |
| 191 | The primitive's bound of intermediates reported as a failed self-check | Caught |
| 192 | A bound of the allocation reported as a failed self-check | Caught |
| 193 | Shares in whole units whatever the quantum | Caught |
| 194 | Equal shares whatever the weights | Caught |
| 195 | Shares joined to the wrong targets | Caught |
| 196 | A weight shown as printed | Caught |
| 197 | The idle mark not shown on a share | Caught |
| 198 | The idle sum not marked | Caught |
| 199 | The idle sum holds the last share | Caught |
| 200 | A class sum holds the last share | Caught |
| 201 | Sums per kind not stored | Caught |
| 202 | Sums per grade not stored | Caught |
| 203 | The pool-level sum not stored | Caught |
| 204 | The idle sum not stored | Caught |
| 205 | An included component without an amount counted as known | Caught |
| 206 | Summed components counted as included | Caught |
| 207 | Every price counted as stated | Caught |
| 208 | Asset sums in map order | Caught |
| 209 | The number of asset sums not stored | Caught |
| 210 | The approval not kept | Caught |
| 211 | That an approval was recorded not remembered | Caught |
| 212 | By which version not remembered | Caught |
| 213 | Being superseded is not a change | Caught |
| 214 | The named version not superseded | Caught |
| 215 | An approval does not move the sequence number | Caught |
| 216 | A withdrawal for any reason | Caught |
| 217 | A second reason accepted | Caught |
| 218 | A withdrawal repeated as a new one | Caught |
| 219 | The reason of a withdrawal not kept | Caught |
| 220 | A withdrawal does not move the sequence number | Caught |
| 221 | A withdrawn version judged further | Caught |
| 222 | A draft current | Caught |
| 223 | A superseded version stands as current | Caught |
| 224 | Counting for a malformed period | Caught |
| 225 | A version that is not current counts | Caught |
| 226 | Counting whatever the start | Caught |
| 227 | Counting whatever the end | Caught |
| 228 | What counts returned without the sequence number | Caught |
| 229 | The sequence number not returned when nothing counts | Caught |
| 230 | The number of summed components not told | Caught |
| 231 | Components without an amount not told | Caught |
| 232 | Assets without their component not told | Caught |
| 233 | Whether a basis is set not told | Caught |
| 234 | The named version always told as standing | Caught |
| 235 | A multiple determined beyond a bound | Caught |
| 236 | A multiple determined with an unknown amount | Caught |
| 237 | Every total told as a multiple | Caught |
| 238 | The number of overlapping versions not told | Caught |
| 239 | Overlapping versions listed without bound | Caught |
| 240 | That an approval was recorded not shown | Caught |
| 241 | By which version not shown | Caught |
| 242 | The approval not shown | Caught |
| 243 | The reason of a withdrawal not shown | Caught |
| 244 | The view carries every target | Caught |
| 245 | What stands against an approval told for any state | Caught |
| 246 | The view shares its figures | Caught |
| 247 | A malformed position | Caught |
| 248 | A page of nothing | Caught |
| 249 | A page over the limit | Caught |
| 250 | Identifiers in map order | Caught |
| 251 | The last identifier read returned again | Caught |
| 252 | Pages of identifiers not cut | Caught |
| 253 | A page shares a model | Caught |
| 254 | The targets of a version without a basis | Caught |
| 255 | A page shares the targets | Caught |
| 256 | A page shares the asset sums | Caught |
| 257 | A page shares the shares | Caught |
| 258 | The last target read returned again | Caught |
| 259 | The last asset sum read returned again | Caught |
| 260 | The last share read returned again | Caught |
| 261 | A negative position among versions | Caught |
| 262 | A page of no versions | Caught |
| 263 | A page of versions over the limit | Caught |
| 264 | The last version read returned again | Caught |
| 265 | Versions in map order | Caught |
| 266 | Pages of versions not cut | Caught |
| 267 | The superseded mark not shown in a row | Caught |

# Deviations from the plan

Recorded in the plan's change review resolution: a read `Targets` was added and `Version` returns the basis without its targets; three unreachable checks were removed. Where the rules of the design re-review are more precise than the sentences above them, the code follows the rules.

# Independent review

- Design: two read-only passes on 2026-11-13 (FAIL with 3 HIGH and no redesign asked; then PASS with findings, no blocker). See the plan's two resolution sections.
- Change: one read-only pass on 2026-11-13, PASS with findings (3 MEDIUM about what the verification claims, 11 LOW), nothing executed by the reviewer. All findings were resolved or stated. The corrections made after it were not reviewed again by a separate reviewer; they are covered by the tests and the third mutation run.
- Not done: the independent verification by a DBA and by QA that W-12 asks for.

# Provisional choices awaiting the owner

Provisional defaults: idle capacity is not borne by an organisation; one pool is split by one basis of weights the caller states; an approval is a recorded approver and reference without a check of who or what; a total must be a whole multiple of the quantum, so nothing is left to rounding; a version with an estimated or unknown power grade can have an approval recorded. Not conservative: nothing checks that depreciation over the versions of a pool does not exceed an acquisition price; a purchase or a perpetual licence reported as a component beside its amortisation is summed twice; the same asset may appear in two pools; a component wrongly stated as included is not summed; a caller who states the used sum as the capacity has normalised and nothing notices. Costs to availability: one unknown amount, one asset without its component or one overlapping current version blocks the record; a correction that changes the period goes by withdrawing and opening, which leaves an interval without a current version; an exhausted pool has no exit but another pool identifier. Assumptions: costs are never negative; one currency per version; one depreciation or lease per asset and period; the weights 60, 20 and 20 behind the acceptance amounts. Undecided and not chosen here: how a pool is split, who bears idle capacity, the customer's approval policy and who may record or withdraw; what a workspace is in relation to the MSP and its customers; how shares reach a billing and what keeps a version from being used twice.

# Not verified

A failed self-check, which no test produces, and the mapping of the allocation's other errors to it; any inventory, depreciation schedule, meter or PUE source; a customer's approval; exposure of shares to a customer-facing read, which must not happen; a process that dies, persistence, any operating system other than the one named, and everything listed as deferred in the plan. No deployment, payment, message, real account or customer data was involved.
