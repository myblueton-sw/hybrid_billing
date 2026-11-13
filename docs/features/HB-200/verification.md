---
type: VerificationRecord
title: VMware asset and meter history verification
description: Records what was executed and reviewed for the first model of work item W-13, with results, the corrections made on the way, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-200](https://linear.app/hybrid-billing/issue/HB-200/w-13-vmware-asset-and-meter-history-with-configuration-spans-energy) adds the Go package `modules/vmhistory` from main `3163a13`: per workspace and source, assets with a lifetime that is revised by revision; configuration spans with CPUs, memory and disk that may each be not stated, sizes converted exactly from a closed list of units; cumulative energy counters in Wh, kWh or MWh with readings that carry a generation and a revision; voids of spans, counters and readings; and `Usage`, which reads one asset over a period and returns, for the part of the period in which the asset was alive, exact integrals of configuration per organisation and the energy of pairs of readings that are next to each other, of one generation, in order, close enough and wholly inside, with covered and uncovered seconds, ordered reasons and a result of `complete`, `incomplete` or `not_applicable` for each.

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan twice before implementation and the implemented change afterwards (see Independent review). Machine: mac_mini.

Nothing calls the package and no vCenter is called. `complete` says that the reported history covers the window, not that it is true. The requirement that a present size is not turned into a past month is met only in part and by obligation: an adapter that knows only today's size can report one span from creation to now and nothing notices. Not delivered: counter kinds other than a cumulative energy counter, a local daily aggregate, query windows and checkpoints, power as a rate, parallel meters, attribution of energy to VMs or organisations. DC01, PO06 and SIM06 are NOT_RUN. W-13 is not complete, its design part stays open with the owner, and the independent verification by QA and a domain reviewer that W-13 asks for remains.

# Executed checks

Executed on 2026-11-13 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain, with `GOCACHE` set to a directory of the session. No other operating system was tested.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All twenty-two packages `ok`, including `modules/vmhistory`; run after the last correction |
| `go test -count=40 ./modules/vmhistory` | `ok`: forty consecutive runs without a failure, before each mutation run; the last on the final code |

The cases of the acceptance at the level of this package (`usage_test.go`): a size changed within a month enters the integrals by its two spans, and a half of a month without a span is uncovered and not filled; a reset by generation, a decrease, a gap and a meter that stood still; two months given as the instants of a zone with daylight saving time, 743 and 721 hours long, which shows only that the package takes the instants as given; one number in GB and in GiB giving different bytes, and eleven spellings of units refused; kW and seven other spellings refused as a counter unit, and Wh, kWh and MWh converted exactly; an asset with no history, incomplete with no entry and no total. The expected numbers were worked out apart from the code.

Literal tests with a test clock (`write_test.go`, `period_test.go`): construction and its limits; forms of identifiers, whole numbers, values and sizes at their edges; every refusal of an asset, a span, a counter, a batch of readings and each void, malformed before the lookup, with the order of error kinds; repeats and conflicts field by field, a repeat of a voided fact answered as voided; revisions of a lifetime, stale, replacing the kind, refused by a span or a reading outside, a reading on the deletion, a voided counter holding nothing back, and a widening that makes complete results incomplete; spans that touch, overlap, lie outside the life or end after the registry's time; revisions of a reading, a voided reading made live again, a late void of an earlier revision refused; generations out of order within a batch and against stored readings, judged on what the batch leaves; readings of a voided counter; segments of every kind at, inside and across the edges of a window, a window inside one segment, time before and after the readings; every reason of energy in one result and every reason of configuration in one; two and three counters in sequence, touching, a second apart, overlapping and beside a gap; a voided counter; windows cut by creation and deletion, empty windows and a period past the registry's time; dimensions not stated and a stated zero; the sequence and change numbers over sixteen writes and repeats, and refusals; pages of assets, spans, counters and readings; another workspace and another source; a clock that goes back; errors that are fixed values; a scan of the source for floating point and comments and, as a check of names only, for an input named for a utilisation, a rate or a price.

Property test (`property_test.go`, `model_test.go`, fixed seeds): 300 histories of 110 steps over five assets and four counters within about forty seconds of time. A reference model written in the test keeps every fact in a plain list, decides the outcome of every operation and computes a period second by second, with `math/big` for quantities; nineteen times per history every asset is read over a random period and compared with the model in full, with both numbers. A second oracle that enumerates no segments: when energy is complete, the total equals the sum over counters of the last value in the closed window less the first. Read from the registry besides: covered and uncovered seconds add up to the window, complete implies nothing uncovered and incomplete implies a reason, for both results; the configuration of two adjoining periods adds up to that of their union, per organisation and dimension; a counter's energy over a union is never less than over the parts, and equal when no paired segment of it crosses the inner boundary; live spans of an asset never overlap and lie within its life; listed readings equal the model's, with generations that never fall. The test fails unless named outcomes reached their minimum, among them each of three results for both, each of eleven reasons on thirty assets, the second oracle sixty times, and twenty-eight kinds of refusal fifteen times each, a refusal counted once per seed and target and a success per call.

An asset holding exactly its limit of 2,000 spans and 5,761 readings is read over forty days.

Concurrency under the race detector, forty rounds: one batch of readings from six goroutines, stored once; four spans of which each overlaps its neighbours, of which exactly one is stored; a void of a span against three readers of a period, who see the span or not, a batch whole or not at all, and numbers that never go back. Not exercised concurrently: revisions of an asset, voids of counters and readings, pages, a second workspace.

Limits of these tests: the model is the same rules written a second time by the same author. As the change review noted, it shares with the code the order of error kinds, the enumeration and classification of segments, the lists of reasons and the definition of the window; a misreading of the plan common to both would pass. What it adds is the second-by-second count of covered time, a pairwise check of overlapping counters, a scan of all facts instead of the first and last, and conversion by `big.Rat`. The second oracle fires only when a result is already complete. The model covers one source and one workspace and three of the nine units of size; the rest rests on the literal tests.

# Corrections made on the way

- The property test agreed with the registry on its first run; the generator was steered towards outcomes it reached too rarely.
- The first mutation run (265 rows) caught 247. Of the others, tests were added for five; eight did not compile and were rewritten; one was nonsense and removed; and four showed code that could not change a result, which was simplified: the overlap of counters and a skipped segment before the window.
- The change review led to one change a caller can see, a malformed position judged before the asset is looked up, and to an upper bound of two limits; see the plan's change review resolution.
- The second run, on the code after the change review, left two survivors, both code that could not change a result, which was removed as described in the plan. The third run had one row that did not compile after that removal; it was rewritten and run again.

# Mutation check

Single mutations of the five source files of `modules/vmhistory` were applied one at a time by a Python script, and the package's tests were run after each. The per-row results were appended to files that were never overwritten. The script and the result files are kept outside the repository, so the table below cannot be audited from the change alone.

- First run, 265 rows, before the change review: 247 caught; handled as described above.
- Second run, 265 rows, after the change review: 263 caught, two survivors handled as described.
- Third run, 263 rows, on the final code: 262 caught and one row that did not compile; that row was rewritten and run again and caught. The table reports the third run with that row's second result.

| No. | Fault introduced | Result |
| --- | --- | --- |
| 1 | Identifier length unbounded | Caught |
| 2 | An empty identifier | Caught |
| 3 | The longest identifier refused | Caught |
| 4 | A slash in an identifier | Caught |
| 5 | A space in an identifier | Caught |
| 6 | An at sign in an identifier | Caught |
| 7 | A number above the maximum | Caught |
| 8 | The highest number refused | Caught |
| 9 | The highest number of CPUs refused | Caught |
| 10 | The widest period refused | Caught |
| 11 | A size of nineteen digits | Caught |
| 12 | A size of eighteen digits refused | Caught |
| 13 | A value of sixteen digits | Caught |
| 14 | A value of fifteen digits refused | Caught |
| 15 | Any void reason | Caught |
| 16 | Limit not required: l.Sources > 0 | Caught |
| 17 | Limit not required: l.Assets > 0 | Caught |
| 18 | Limit not required: l.Spans > 0 | Caught |
| 19 | Limit not required: l.Counters > 0 | Caught |
| 20 | Limit not required: l.Readings > 0 | Caught |
| 21 | Limit not required: l.Batch > 0 | Caught |
| 22 | Limit not required: l.Page > 0 | Caught |
| 23 | Limit not required: l.MaxPeriod <= longestPeriod | Caught |
| 24 | Limit not required: l.MaxGap > 0 | Caught |
| 25 | Limit not required: l.MaxGap <= int64(l.MaxPeriod/time.Second) | Caught |
| 26 | A kilobyte of 1024 | Caught |
| 27 | A megabyte of 1024 squared | Caught |
| 28 | GB read as GiB | Caught |
| 29 | TB read as TiB | Caught |
| 30 | KiB read as kB | Caught |
| 31 | MiB read as MB | Caught |
| 32 | GiB read as GB | Caught |
| 33 | TiB read as TB | Caught |
| 34 | KWh read as Wh | Caught |
| 35 | MWh read as kWh | Caught |
| 36 | KW received as a counter unit | Caught |
| 37 | A unit in lower case received | Caught |
| 38 | Whole numbers of any length | Caught |
| 39 | A leading zero | Caught |
| 40 | A sign in a whole number | Caught |
| 41 | A unit without a value read as not stated | Caught |
| 42 | A size in an unknown unit | Caught |
| 43 | A size not converted to bytes | Caught |
| 44 | A value that ends in its point | Caught |
| 45 | A value of seven places | Caught |
| 46 | Letters after the point | Caught |
| 47 | A fraction read as a whole number | Caught |
| 48 | A value not converted from its unit | Caught |
| 49 | Whole watt-hours printed with a point | Caught |
| 50 | A small fraction printed without its zeros | Caught |
| 51 | Zeros kept at the end of a fraction | Caught |
| 52 | Construction without limits | Caught |
| 53 | Construction without a clock | Caught |
| 54 | Time going backwards | Caught |
| 55 | A registry that was not built | Caught |
| 56 | Prints without lengths or separators | Caught |
| 57 | No instant taken as one | Caught |
| 58 | An instant with a fraction | Caught |
| 59 | The change number of an asset never moves | Caught |
| 60 | The sequence number never moves | Caught |
| 61 | Facts before the creation | Caught |
| 62 | Facts after the deletion | Caught |
| 63 | A fact that ends with the life refused | Caught |
| 64 | A fact that starts with the life refused | Caught |
| 65 | An asset without an identifier | Caught |
| 66 | An asset of any kind | Caught |
| 67 | Revision zero of an asset | Caught |
| 68 | Revisions of an asset unbounded | Caught |
| 69 | An asset without a creation | Caught |
| 70 | A deletion with a fraction | Caught |
| 71 | A deletion at the creation | Caught |
| 72 | A creation kept in the caller's zone | Caught |
| 73 | Not part of an asset: the kind | Caught |
| 74 | Not part of an asset: the creation | Caught |
| 75 | Not part of an asset: the deletion | Caught |
| 76 | The stored revision with other content taken as a repeat | Caught |
| 77 | A stale revision of an asset replaces | Caught |
| 78 | A creation after the registry's time | Caught |
| 79 | A deletion after the registry's time | Caught |
| 80 | A revision that leaves a span outside the life | Caught |
| 81 | A revision judged by the first span alone | Caught |
| 82 | A revision judged by the last span alone | Caught |
| 83 | A revision that leaves a reading outside the life | Caught |
| 84 | A voided counter holds a revision back | Caught |
| 85 | A revision judged by the first reading alone | Caught |
| 86 | A revision judged by the last reading alone | Caught |
| 87 | A revision moves no number | Caught |
| 88 | Sources without limit | Caught |
| 89 | Assets without limit | Caught |
| 90 | One source over the limit | Caught |
| 91 | One asset over the limit | Caught |
| 92 | A span without an identifier | Caught |
| 93 | A span with a malformed asset | Caught |
| 94 | A malformed organisation | Caught |
| 95 | A span without a start | Caught |
| 96 | A span whose end has a fraction | Caught |
| 97 | A span of no length | Caught |
| 98 | A span kept in the caller's zone | Caught |
| 99 | CPUs unbounded | Caught |
| 100 | A malformed memory size | Caught |
| 101 | A malformed disk size | Caught |
| 102 | Not part of a span: the asset | Caught |
| 103 | Not part of a span: the start | Caught |
| 104 | Not part of a span: the end | Caught |
| 105 | Not part of a span: the CPUs | Caught |
| 106 | Not part of a span: the memory value | Caught |
| 107 | Not part of a span: the memory unit | Caught |
| 108 | Not part of a span: the disk value | Caught |
| 109 | Not part of a span: the disk unit | Caught |
| 110 | Not part of a span: the organisation | Caught |
| 111 | A changed span taken as a repeat | Caught |
| 112 | A repeat of a voided span answered as live | Caught |
| 113 | A span outside the life | Caught |
| 114 | A span that ends after the registry's time | Caught |
| 115 | Overlapping spans | Caught |
| 116 | Spans that touch refused | Caught |
| 117 | A span that starts where another ends refused or misplaced | Caught |
| 118 | Spans without limit | Caught |
| 119 | One span over the limit | Caught |
| 120 | Spans not counted towards the limit | Caught |
| 121 | A span moves no number | Caught |
| 122 | A second reason for a span accepted | Caught |
| 123 | A voided span still counts | Caught |
| 124 | A void of a span moves no number | Caught |
| 125 | The reason of a voided span not kept | Caught |
| 126 | A counter without an identifier | Caught |
| 127 | A counter with a malformed asset | Caught |
| 128 | A longest distance of zero | Caught |
| 129 | A longest distance above the limit | Caught |
| 130 | Not part of a counter: the asset | Caught |
| 131 | Not part of a counter: the unit | Caught |
| 132 | Not part of a counter: the distance | Caught |
| 133 | A changed counter taken as a repeat | Caught |
| 134 | Counters without limit | Caught |
| 135 | One counter over the limit | Caught |
| 136 | Counters in the order received | Caught |
| 137 | A counter moves no number | Caught |
| 138 | A second reason for a counter accepted | Caught |
| 139 | A void of a counter moves no number | Caught |
| 140 | An empty batch | Caught |
| 141 | A reading without an instant | Caught |
| 142 | An instant twice in a batch | Caught |
| 143 | Generation zero | Caught |
| 144 | Generations unbounded | Caught |
| 145 | Revision zero of a reading | Caught |
| 146 | Revisions of a reading unbounded | Caught |
| 147 | A malformed value | Caught |
| 148 | Batches of any size | Caught |
| 149 | A batch at the limit refused | Caught |
| 150 | A reading kept in the caller's zone | Caught |
| 151 | Not part of a reading: the generation | Caught |
| 152 | Not part of a reading: the value | Caught |
| 153 | The stored revision of a reading with another value taken as a repeat | Caught |
| 154 | A repeat of a voided reading answered as live | Caught |
| 155 | A stale revision of a reading replaces | Caught |
| 156 | A reading for a voided counter | Caught |
| 157 | A reading outside the life | Caught |
| 158 | A reading after the registry's time | Caught |
| 159 | A replaced reading kept beside its replacement | Caught |
| 160 | Readings in the order received | Caught |
| 161 | Generations that fall | Caught |
| 162 | Equal generations refused | Caught |
| 163 | Readings without limit | Caught |
| 164 | The limit of readings refused | Caught |
| 165 | Replacements counted towards the limit | Caught |
| 166 | A batch moves no number | Caught |
| 167 | A void at no instant | Caught |
| 168 | A void of revision zero | Caught |
| 169 | A void of a reading for any reason | Caught |
| 170 | A second reason for a reading accepted | Caught |
| 171 | A void of a reading of a voided counter | Caught |
| 172 | A late void of an earlier revision | Caught |
| 173 | A void of a revision not stored yet | Caught |
| 174 | A voided reading still counts | Caught |
| 175 | A void of a reading moves no number | Caught |
| 176 | The seconds of an entry not counted | Caught |
| 177 | Seconds of a dimension not stated not counted | Caught |
| 178 | A size not multiplied by its seconds | Caught |
| 179 | Not-stated seconds not returned | Caught |
| 180 | A dimension nobody stated returned as zero | Caught |
| 181 | Uncovered seconds of configuration not counted | Caught |
| 182 | A span that ends at the start of the window entered | Caught |
| 183 | A span that starts at the end of the window entered | Caught |
| 184 | A span not cut at the end of the window | Caught |
| 185 | A span not cut at the start of the window | Caught |
| 186 | Organisations summed together | Caught |
| 187 | Covered seconds of configuration not counted | Caught |
| 188 | Organisations in map order | Caught |
| 189 | No organisation listed among the organisations | Caught |
| 190 | Uncovered configuration complete | Caught |
| 191 | A dimension not stated complete | Caught |
| 192 | Memory and disk confused in the reasons | Caught |
| 193 | Configuration complete with reasons against it | Caught |
| 194 | Shared time counted twice or time before the window counted | Caught |
| 195 | Time after the window counted | Caught |
| 196 | Time counted once per segment | Caught |
| 197 | Overlapping counters unnoticed | Caught |
| 198 | Counters that touch said to overlap | Caught |
| 199 | A voided counter in a result | Caught |
| 200 | The segment across the start of the window missed | Caught |
| 201 | A segment that starts at the end of the window counts | Caught |
| 202 | Energy across a reset | Caught |
| 203 | A decrease summed as negative energy | Caught |
| 204 | Equal values called a decrease | Caught |
| 205 | Energy across a gap | Caught |
| 206 | The longest distance called a gap | Caught |
| 207 | A reset with a smaller value called a decrease | Caught |
| 208 | Energy of a segment across the start of the window | Caught |
| 209 | Energy of a segment across the end of the window | Caught |
| 210 | A boundary that was not read unnoticed | Caught |
| 211 | The energy of a counter is its last segment | Caught |
| 212 | The seconds of a counter not counted | Caught |
| 213 | A counter without a segment returned as zero | Caught |
| 214 | The total is the last counter | Caught |
| 215 | An asset without a counter judged as not read | Caught |
| 216 | Segments across an edge counted as covered | Caught |
| 217 | An overlap with a segment across an edge unnoticed | Caught |
| 218 | Time in a reset or a gap said to be not read | Caught |
| 219 | Time that was not read unnoticed | Caught |
| 220 | Reasons of energy in another order | Caught |
| 221 | A total under overlapping counters | Caught |
| 222 | A total of zero when nothing contributes | Caught |
| 223 | Energy complete with reasons against it | Caught |
| 224 | A period without a start | Caught |
| 225 | A period whose end has a fraction | Caught |
| 226 | A period that runs backwards | Caught |
| 227 | Periods of any length | Caught |
| 228 | The longest period refused | Caught |
| 229 | A window before the creation | Caught |
| 230 | A window in the caller's zone | Caught |
| 231 | A window after the deletion | Caught |
| 232 | A result without the sequence number | Caught |
| 233 | A result without the change number | Caught |
| 234 | A window of no length judged | Caught |
| 235 | The last asset read returned again | Caught |
| 236 | Assets in map order | Caught |
| 237 | More assets not told | Caught |
| 238 | A page of assets without the sequence number | Caught |
| 239 | A malformed position among assets | Caught |
| 240 | A page of no assets | Caught |
| 241 | A page of assets over the limit | Caught |
| 242 | A page of nothing of an asset | Caught |
| 243 | A page of an asset over the limit | Caught |
| 244 | A position among spans with a fraction | Caught |
| 245 | A malformed position among counters | Caught |
| 246 | A malformed position judged after the asset is looked up | Caught |
| 247 | The last span read returned again | Caught |
| 248 | A page of spans without the sequence number | Caught |
| 249 | A page of spans without the change number | Caught |
| 250 | More spans not told | Caught |
| 251 | The last counter read returned again | Caught |
| 252 | A page of counters without the sequence number | Caught |
| 253 | A page of counters without the change number | Caught |
| 254 | More counters not told | Caught |
| 255 | A voided counter listed as live | Caught |
| 256 | A position among readings with a fraction | Caught |
| 257 | A page of no readings | Caught |
| 258 | A page of readings over the limit | Caught |
| 259 | The readings of a voided counter listed | Caught |
| 260 | The last reading read returned again | Caught |
| 261 | A page of readings without the sequence number | Caught |
| 262 | A page of readings without the change number | Caught |
| 263 | More readings not told | Caught |

# Deviations from the plan

None beyond what the plan's change review resolution records: the corrected sentences of the design re-review and the code removed as unreachable.

# Independent review

- Design: two read-only passes on 2026-11-13 (FAIL with 3 HIGH and no redesign asked; then PASS with findings, no blocker). See the plan's two resolution sections.
- Change: one read-only pass on 2026-11-13, PASS with findings (7 findings, none HIGH), nothing executed by the reviewer. All findings were resolved or stated. The corrections made after it, and the two removals after the second mutation run, were not reviewed again by a separate reviewer; they are covered by the tests and the third mutation run.
- Not done: the independent verification by QA and a domain reviewer that W-13 asks for.

# Provisional choices awaiting the owner

Not conservative, stated: a span is believed for all of its time; two sources may report the same machine; an organisation is not checked against anything; a meter that stood still reads as zero energy. Costs to availability: energy is complete only when readings lie exactly on both edges of the window, so an adapter must read at period boundaries and at creation and deletion, or the owner must decide how a boundary is settled; every reset, every replacement with time between the meters and every late boundary makes its period incomplete with no rule to settle it; a gap discards a difference that may be good; one shared second of two counters withholds the total; a reading ahead of the registry's clock by a second is rejected with its batch; voided facts count towards the limits for good. Assumptions: energy counters are cumulative and never negative; one configuration per asset at a time; sizes are whole numbers of their unit; the shape of a counter, which is provisional until the vCenter decision. Undecided and not chosen here: the vCenter version and its counters, their kind, unit and retention; the local daily aggregate and who approves it; the billing unit; allocated against used; a basis of observation for a span; how a boundary reading is obtained.

# Not verified

Any real vCenter, an adapter, the units a source really means by its labels, a process that dies, persistence, the time one read takes at its stated bound, any operating system other than the one named, and everything listed as deferred in the plan. No deployment, payment, message, real account or customer data was involved.
