---
type: VerificationRecord
title: Mapping revisions and normalization runs verification
description: Records what was executed and reviewed for the first model of work item W-07, with results, the corrections made on the way, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-188](https://linear.app/hybrid-billing/issue/HB-188/w-07-mapping-revisions-and-normalization-runs-with-separate) adds the Go package `modules/normalize` from main `e858f6b`: declared target schemas; numbered immutable mapping revisions in which every source field has a declared kind and cost kind and an entry is accepted only when they equal the target's, so that a sale amount, another cost kind or a quantity cannot be mapped into a cost; and, per workspace, normalization runs that check rows in chunks that can be sent again, keep null, not provided and zero apart, keep source decimals exactly, class every row as normal, error or quarantined, and at closing give two separate verdicts, format conformance and billing readiness, with exact sums per cost field and currency.

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan twice before implementation and the implemented change afterwards (see Independent review). Machine: mac_mini.

Nothing calls the package: no file is parsed, no target that copies a FOCUS version is declared, no mapping of a real provider exists, and the Python import path is not connected. "Format conformance" means conformance to a declared target as far as the listed checks go; what a source field is, and that a field is not applicable, are the caller's declarations and are not verified. FD04, FD05, FD06, SIM01 and AUD-QA-07 are NOT_RUN, and W-07 is not complete.

# Executed checks

Executed on 2026-11-13 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain. No other operating system was tested.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All sixteen packages `ok`, including `modules/normalize`; run again after the last correction |
| `go test -count=40 ./modules/normalize/` | `ok`: forty consecutive runs without a failure, before the first mutation run and again on the code after the change review; ten more after the last added test |

Literal tests with a test clock (`mapping_test.go`, `run_test.go`, `open_test.go`): every refusal of a target and of a mapping, among them a sale amount as a billed cost, another cost kind, a quantity as a cost, one source field with two kinds, `not_applicable` on a required field, a cost without its currency and half a period; the guard of `DeclareMapping` with a repeated, a stale and an early expected number and at both limits; computed support, with a field that is not required and not supported making the mapping partial, and a target without a required cost never ready; null, not provided and zero through every transformation; twenty-six decimal cases through costs and quantities, with the input bound, the result bound after scaling both ways, negative zero, an exponent and an over-long text; times with offsets, fractions, the edges of the offset and of the year; every finding code with the field it is attached to and the order within a row; the period at each edge of the window and without a window; duplicates in one chunk, across chunks, three times and of a row in error; a locator and an event key of the same text; malformed chunks refused whole; ten ways a chunk sent again differs, each making the run inconsistent, with what such a run then refuses; the chunk digest keeping fields apart by their length; `Close` refused with chunks missing and with another row count; twenty-one cases of the two verdicts with their reasons, reaching the five combinations that can occur; sums outside the profile, alone and with other reasons; every refusal of `Open`; an identical `Open` after a newer revision and at the run limit; a run bound to its revision; abandoning and discarding with every reason, rows given back, what stays after discarding; bounds of findings and unused names with totals and order; paging; views that share no state; the tenant boundary and time; errors that repeat nothing and findings that hold no value; and a scan of the package's source for floating point types, conversions, parsing and formatting, which also checks that the pattern finds what it bans.

Property tests (`property_test.go`, fixed seeds): 1,500 runs over random targets, mappings and rows against a reference model written in the test, which decides each row's class, the number of findings, the verdicts with their reasons and the sums with 64-bit integers; after every chunk the three counts add up and equal the model's, with every chunk sent twice; at closing the verdicts, counts, findings total, sums and the normalized amounts of every normal row equal the model's. The test fails unless every reason but the profile bound, the five combinations of verdicts and runs with all three classes were each reached at least five times, and if a nonconformant run was ever ready. A second property runs the same rows in two random chunkings and orders of arrival and compares verdicts, counts, sums, unused names and the normal rows by key.

Scale (`scale_test.go`): 100,000 rows in 100 chunks arriving in random order with repeats, 100 rows in error and 40 quarantined spread over them: the counts and the sums per currency equal the generator's, rows and findings come back in order.

Concurrency under the race detector: eight goroutines checking the same twenty chunks while closing and reading, each chunk counted once; twelve declarers of one mapping with one expected number, of which one appends; six goroutines opening, checking, closing and discarding runs in two workspaces.

Limits of these tests: the model is the same rules written a second time by the same author, and its judgement of a row and its order of reasons follow the code's structure, so agreement there shows consistency; it counts findings and does not compare their codes, knows two currencies, a table of times and small numbers, and does not cover the profile bound, unused names, the quarantined list or runs that end without closing. The second property compares the code with itself. Expectations in the tables were worked out by hand.

# Corrections made on the way

- The first mutation run (270 rows) caught 245. Of the 25 others, eleven showed tests that did not test what they named or were missing (identifier and currency rules, the registry keeping a caller's fields, `not_applicable` on a field that only billing requires, lengths and the document in the chunk digest, the order of findings within a row, closing a run that ended otherwise, a target without costs, and others); tests were added. Two showed code that could change no result, which was removed; two are equivalent and were taken off the list (see the plan's change review resolution); ten did not compile and were rewritten.
- The change review found that the property test drew random numbers while ranging over a map, so that its seed did not reproduce a run, and that an abandoned run kept the names it had collected; both were corrected, with ten smaller points.
- While the second mutation run was being prepared, forty consecutive runs took about six times as long as before. The machine's load average was above 13 at the time; a single run afterwards took 1.4 seconds. No change was made for it.

# Mutation check

276 single-line mutations of `modules/normalize` were applied one at a time by a Python script, and the package's tests were run after each. The per-row results were appended to files that were never overwritten.

- First run, 270 rows, before the change review: 245 caught; handled as described above.
- Second run, all 276 rows against the code after the change review: 275 caught, 1 not caught (a new revision of an existing mapping at the mapping limit, which no test reached). A test was added and that row run again: caught.
- Result reported below: 276 of 276 caught, 275 in the second run and one in the third.

| No. | Fault introduced | Result |
| --- | --- | --- |
| 1 | Identifier length unbounded | Caught |
| 2 | Third currency letter unchecked | Caught |
| 3 | First currency letter unchecked | Caught |
| 4 | Any field kind | Caught |
| 5 | Transformations on every kind | Caught |
| 6 | Quantities not numeric | Caught |
| 7 | A sale amount as a target cost kind | Caught |
| 8 | Any missing reason | Caught |
| 9 | Mapping without provider | Caught |
| 10 | Mapping without source version | Caught |
| 11 | Mapping without target | Caught |
| 12 | An event key with a document | Caught |
| 13 | An event key that is not an identifier | Caught |
| 14 | A locator without a document | Caught |
| 15 | A locator key that is not an identifier | Caught |
| 16 | Any abandon reason | Caught |
| 17 | The operator abandon reason unknown | Caught |
| 18 | Any discard reason | Caught |
| 19 | The operator discard reason unknown | Caught |
| 20 | Limit not required: l.Targets > 0 | Caught |
| 21 | Limit not required: l.Fields > 0 | Caught |
| 22 | Limit not required: l.Mappings > 0 | Caught |
| 23 | Limit not required: l.Revisions > 0 | Caught |
| 24 | Limit not required: l.Runs > 0 | Caught |
| 25 | Limit not required: l.Chunks > 0 | Caught |
| 26 | Limit not required: l.ChunkRows > 0 | Caught |
| 27 | Limit not required: l.Rows > 0 | Caught |
| 28 | Limit not required: l.Cells > 0 | Caught |
| 29 | Limit not required: l.Text > 0 | Caught |
| 30 | Limit not required: l.Findings > 0 | Caught |
| 31 | Limit not required: l.Unused > 0 | Caught |
| 32 | Limit not required: l.Page > 0 | Caught |
| 33 | Construction without limits | Caught |
| 34 | Construction without a profile | Caught |
| 35 | Construction without a clock | Caught |
| 36 | Time going backwards | Caught |
| 37 | A target with a malformed identifier | Caught |
| 38 | A target without fields | Caught |
| 39 | Fields per target unbounded | Caught |
| 40 | Field limit one too low | Caught |
| 41 | A field name twice | Caught |
| 42 | A malformed field name | Caught |
| 43 | An unknown field kind | Caught |
| 44 | A cost field without a cost kind | Caught |
| 45 | A sale kind on a field that is not a cost | Caught |
| 46 | A named field of another kind | Caught |
| 47 | Costs without a currency of costs | Caught |
| 48 | A currency of costs without costs | Caught |
| 49 | A period end without a start | Caught |
| 50 | A period start without an end | Caught |
| 51 | One field as both ends of the period | Caught |
| 52 | An unknown period end | Caught |
| 53 | A target declared again with other content | Caught |
| 54 | Targets unbounded | Caught |
| 55 | One target over the limit | Caught |
| 56 | The target keeps the caller's fields | Caught |
| 57 | The billing flag not part of a target's identity | Caught |
| 58 | The format flag not part of a target's identity | Caught |
| 59 | An entry fewer than fields | Caught |
| 60 | Two entries for one field | Caught |
| 61 | An unknown missing reason | Caught |
| 62 | A missing entry with a source | Caught |
| 63 | A missing entry with a kind | Caught |
| 64 | A missing entry with a cost kind | Caught |
| 65 | A missing entry with a transformation | Caught |
| 66 | A missing entry with a power | Caught |
| 67 | A malformed source field name | Caught |
| 68 | A cost source without a cost kind | Caught |
| 69 | A power on copy or negate | Caught |
| 70 | Scaling by nothing | Caught |
| 71 | Scaling down without bound | Caught |
| 72 | Scaling up without bound | Caught |
| 73 | Scaling by ten to the minus twelve refused | Caught |
| 74 | A field that is not required and not supported leaves the mapping complete | Caught |
| 75 | Not applicable on a field that format requires | Caught |
| 76 | Not applicable on a field that billing requires | Caught |
| 77 | Not applicable fields not listed | Caught |
| 78 | A missing field that billing requires is no gap | Caught |
| 79 | A source of another kind | Caught |
| 80 | A source of another cost kind, a sale amount included | Caught |
| 81 | A transformation of something that is not a number | Caught |
| 82 | One source field declared with two kinds | Caught |
| 83 | Readiness possible without a required cost | Caught |
| 84 | A cost mapped without its currency | Caught |
| 85 | A revision that maps nothing | Caught |
| 86 | Half a period mapped | Caught |
| 87 | Every revision complete | Caught |
| 88 | Readiness possible despite a gap | Caught |
| 89 | A repeated declaration checked against the expected number | Caught |
| 90 | A declaration with a stale expected number | Caught |
| 91 | Mappings unbounded | Caught |
| 92 | Revisions unbounded | Caught |
| 93 | A new revision counted as a new mapping | Caught |
| 94 | Revision time not recorded | Caught |
| 95 | The revision view shares its entries | Caught |
| 96 | The revision view shares its list | Caught |
| 97 | Reading a revision of a malformed mapping | Caught |
| 98 | The latest revision not found by number | Caught |
| 99 | A sum not rescaled to a finer amount | Caught |
| 100 | A coarser amount not rescaled | Caught |
| 101 | Numbers not reduced | Caught |
| 102 | Sums equal up to a power of ten | Caught |
| 103 | The sign of a negative number lost | Caught |
| 104 | A number below one without its leading zero | Caught |
| 105 | A text that is no number reported as outside the profile | Caught |
| 106 | A number outside the input bound reported as no number | Caught |
| 107 | Negate does nothing | Caught |
| 108 | Scale in the wrong direction | Caught |
| 109 | Scaling up loses the factor | Caught |
| 110 | Any form the time parser accepts | Caught |
| 111 | Ten digits of a fraction | Caught |
| 112 | Times kept at their offset | Caught |
| 113 | A time that does not exist | Caught |
| 114 | A run on a malformed mapping key | Caught |
| 115 | A run without a workspace | Caught |
| 116 | A run with a malformed identifier | Caught |
| 117 | A run without an input scope | Caught |
| 118 | A run without an input period | Caught |
| 119 | A run without a manifest number | Caught |
| 120 | Fewer rows than chunks | Caught |
| 121 | An empty or inverted window | Caught |
| 122 | The window's end not part of a run's identity | Caught |
| 123 | The window's start not part of a run's identity | Caught |
| 124 | The expected field not part of a run's identity | Caught |
| 125 | The expected sum not part of a run's identity | Caught |
| 126 | An expected sum compared by its spelling | Caught |
| 127 | An expected sum for a malformed currency | Caught |
| 128 | The row count not part of a run's identity | Caught |
| 129 | A run opened again with other content | Caught |
| 130 | An unknown revision reported as rejected | Caught |
| 131 | A run on a revision that is not the latest | Caught |
| 132 | A window without a mapped period | Caught |
| 133 | Expected sums for a field that is not a cost | Caught |
| 134 | Expected sums for a missing field | Caught |
| 135 | Chunks per run unbounded | Caught |
| 136 | Chunk limit one too low | Caught |
| 137 | One row over the capacity of the chunks | Caught |
| 138 | Rows far over the capacity | Caught |
| 139 | Exactly full chunks refused | Caught |
| 140 | Runs unbounded | Caught |
| 141 | One run over the limit | Caught |
| 142 | The run keeps the caller's window | Caught |
| 143 | Expected sums stored as written | Caught |
| 144 | Opening time not recorded | Caught |
| 145 | Chunk fields without their length | Caught |
| 146 | A malformed row key | Caught |
| 147 | Cells per row unbounded | Caught |
| 148 | Cell limit one too low | Caught |
| 149 | A malformed source field name in a row | Caught |
| 150 | Text length unbounded | Caught |
| 151 | Text limit one too low | Caught |
| 152 | A null cell with a text | Caught |
| 153 | The chunk digest in map order | Caught |
| 154 | The chunk digest without the document | Caught |
| 155 | The chunk digest without the row key | Caught |
| 156 | The chunk digest without cell names | Caught |
| 157 | Null and an empty text the same in a chunk | Caught |
| 158 | The chunk digest without cell texts | Caught |
| 159 | Rows not given back | Caught |
| 160 | Chunk number zero | Caught |
| 161 | An empty chunk | Caught |
| 162 | A chunk over the limit | Caught |
| 163 | A chunk number over the declared count | Caught |
| 164 | The last chunk refused | Caught |
| 165 | A repeated chunk is a contradiction | Caught |
| 166 | A chunk sent again with other rows accepted as a repeat | Caught |
| 167 | Checking into a run that is not open | Caught |
| 168 | An inconsistent run keeps its rows | Caught |
| 169 | The time of a contradiction not recorded | Caught |
| 170 | The contradicting chunk not recorded | Caught |
| 171 | Rows per workspace unbounded | Caught |
| 172 | Row limit one too low | Caught |
| 173 | Duplicate keys not quarantined | Caught |
| 174 | The earlier rows of a duplicated key stay as they were | Caught |
| 175 | Keys seen not remembered | Caught |
| 176 | A checked chunk not remembered | Caught |
| 177 | Checked rows not counted | Caught |
| 178 | Rows not counted against the run | Caught |
| 179 | Rows not counted against the workspace | Caught |
| 180 | Every cell name reported as unused | Caught |
| 181 | Not provided shown as null | Caught |
| 182 | Null shown as not provided | Caught |
| 183 | An unmapped field shown as not provided | Caught |
| 184 | Any text cell asks for the currency of costs | Caught |
| 185 | An empty text judged by its kind | Caught |
| 186 | An empty text field in error | Caught |
| 187 | Transformations not applied | Caught |
| 188 | A transformed value outside the profile accepted | Caught |
| 189 | Amounts kept as they were spelled | Caught |
| 190 | Times kept as they were spelled | Caught |
| 191 | A malformed currency accepted | Caught |
| 192 | Required null and not provided swapped | Caught |
| 193 | A required field that is not provided accepted | Caught |
| 194 | A required null reported as not provided | Caught |
| 195 | Rows never lack anything | Caught |
| 196 | A cost without its currency accepted | Caught |
| 197 | The currency demanded of rows without a cost | Caught |
| 198 | Period values demanded without a window | Caught |
| 199 | A row without its period accepted under a window | Caught |
| 200 | A missing period end accepted | Caught |
| 201 | An inverted period accepted | Caught |
| 202 | A period of no length accepted | Caught |
| 203 | A row before the window accepted | Caught |
| 204 | A row after the window accepted | Caught |
| 205 | A row at the window's start refused | Caught |
| 206 | A row at the window's end refused | Caught |
| 207 | Findings of a row in the order they were found | Caught |
| 208 | Rows with findings stay normal | Caught |
| 209 | Closing again after discarding refused | Caught |
| 210 | Closing a run that ended otherwise | Caught |
| 211 | Closing with chunks missing | Caught |
| 212 | Closing with another row count | Caught |
| 213 | Findings kept without bound | Caught |
| 214 | Findings not counted | Caught |
| 215 | Rows lacking a required value not noticed | Caught |
| 216 | Quantities summed as costs | Caught |
| 217 | Sums not added | Caught |
| 218 | Unused names in map order | Caught |
| 219 | Unused names unbounded | Caught |
| 220 | A partial mapping called conformant | Caught |
| 221 | A mapping gap called ready | Caught |
| 222 | Rows in error called ready | Caught |
| 223 | Rows in error called conformant | Caught |
| 224 | Quarantined rows called ready | Caught |
| 225 | Quarantined rows called conformant | Caught |
| 226 | Rows lacking a required value called ready | Caught |
| 227 | Expected sums demanded for a missing field | Caught |
| 228 | Ready without expected sums | Caught |
| 229 | Expected sums demanded for a field that billing does not require | Caught |
| 230 | An expected currency without a value accepted | Caught |
| 231 | A sum outside the profile called ready | Caught |
| 232 | An unexpected currency accepted | Caught |
| 233 | A differing sum accepted | Caught |
| 234 | Totals not expected not reported | Caught |
| 235 | Differing totals called ready | Caught |
| 236 | Ready without a window | Caught |
| 237 | A sum outside the profile not reported | Caught |
| 238 | Ready with reasons against it | Caught |
| 239 | Closing time not recorded | Caught |
| 240 | Abandoning again with another reason | Caught |
| 241 | Abandoning a run that is not open | Caught |
| 242 | An abandoned run keeps its rows | Caught |
| 243 | Abandon time not recorded | Caught |
| 244 | Discarding again with another reason | Caught |
| 245 | Discarding a run that is not closed | Caught |
| 246 | A discarded run keeps its rows | Caught |
| 247 | Discard time not recorded | Caught |
| 248 | Abandoning with an unknown reason | Caught |
| 249 | Discarding with an unknown reason | Caught |
| 250 | A verdict view shares its reasons | Caught |
| 251 | The run view shares its window | Caught |
| 252 | The run view shares its expected sums | Caught |
| 253 | The run view shares its sums | Caught |
| 254 | The profile revision not shown | Caught |
| 255 | A page of nothing | Caught |
| 256 | A page over the limit | Caught |
| 257 | A full page refused | Caught |
| 258 | A negative position | Caught |
| 259 | Reading a run that is not closed | Caught |
| 260 | Pages not cut | Caught |
| 261 | A position beyond the end | Caught |
| 262 | A page of rows without the billing verdict | Caught |
| 263 | A page of rows without the format verdict | Caught |
| 264 | The fields a row lacks not shown | Caught |
| 265 | A page shares the run's findings | Caught |
| 266 | A page shares the run's quarantined rows | Caught |
| 267 | A canonical time outside four digits of a year | Caught |
| 268 | The year zero refused | Caught |
| 269 | The year 9999 refused | Caught |
| 270 | An offset of more than a day | Caught |
| 271 | An offset of sixty minutes | Caught |
| 272 | A run that declares more rows than the workspace may hold | Caught |
| 273 | A run of exactly the workspace's rows refused | Caught |
| 274 | An ended run keeps the names it collected | Caught |
| 275 | A malformed expected field name | Caught |
| 276 | The profile identifier not recorded | Caught |

# Deviations from the plan

None after the plan was amended with the change review resolution. Where the rules of the design re-review are more precise than the sentences above them, the code follows the rules.

# Independent review

- Design: two read-only passes on 2026-11-13 (FAIL with 4 HIGH and no redesign asked; then PASS with findings). See the plan's two resolution sections.
- Change: one read-only pass on 2026-11-13, PASS with findings (2 MEDIUM, 10 LOW), nothing executed by the reviewer. All findings were resolved or stated. The corrections made after it were not reviewed again by a separate reviewer; they are covered by the tests and the second mutation run.

# Provisional choices awaiting the owner

Labelled provisional: a duplicated key quarantines every row that has it; a row that straddles the window is in error; a run can only be opened on the latest revision. Not the conservative choices, listed in the plan: a run that is `partial` may be `ready`; mappings are installation-wide; a mapped cost that billing readiness does not require needs no expected sums; a window turns rows outside it into error rows and so enters the format verdict; unused cells are ignored; rows are not reserved at `Open`. Undecided and not chosen here: the internal target version and dataset; the mapping of any real provider and what each source field is; the precision limits; which period pair is checked; which run counts for a subject and what happens to a run when its input is superseded; retention.

# Not verified

Real exports, a target that is a verified copy of a FOCUS version, any real mapping, a process that dies during a run, persistence, any operating system other than the one named, and everything listed as deferred in the plan. No deployment, payment, message, real account or customer data was involved.
