---
type: VerificationRecord
title: TokenMeter snapshot exports and eligibility verification
description: Records what was executed and reviewed for the first model of work item W-11, with results, the corrections made on the way, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-196](https://linear.app/hybrid-billing/issue/HB-196/w-11-tokenmeter-snapshot-exports-with-usage-events-cost-classes-and-an) adds the Go package `modules/tokenmeter` from main `5b29bdd`: receptions of snapshot exports from TokenMeter, at most one that is not voided per export; an immutable header with period, how the export was produced, the kind of numeric, declared events, token totals and cost totals and where that declaration comes from; usage events that keep event identifier, request and attempt, outcome, four token kinds that may be not known, a usage grade, a cache relation and a cost with its original text, currency, grade, origin and document; findings per event; at closing, token sums per usage grade, a net input, and cost sums apart for the classes `estimated`, `self_hosted`, `invoice_referenced` and `unconfirmed`, with sums per document; and an eligibility verdict with ordered reasons. `Counting` reads the eligible reception of an exact period with a sequence number of the source.

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan twice before implementation and the implemented change afterwards (see Independent review). Machine: mac_mini.

Nothing calls the package and no TokenMeter is called; the TokenMeter connection is not opened. The names of grades, outcomes and origins are assumptions, since no TokenMeter interface is in the planning files. `invoice_referenced` means that the source named a document, not that the cost was reconciled with it. `eligible` is necessary for using an export in a month close and not sufficient. The acceptance item about duplication with HawkEye's cost for self-hosted GPUs is not met by this package. BILL01, COM01, SIM03, AUD-AR-02, AUD-AR-04, AUD-AR-05 and AUD-RT-01 are NOT_RUN. W-11 is not complete.

# Executed checks

Executed on 2026-11-13 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain, with `GOCACHE` set to a directory of the session. No other operating system was tested.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All twenty packages `ok`, including `modules/tokenmeter`; run after the last correction |
| `go test -count=40 ./modules/tokenmeter` | `ok`: forty consecutive runs without a failure, before each of the second and third mutation runs; the third on the code after the change review |

The cases of the acceptance at the level of this package (`event_test.go`): a request with a failed, a cancelled and a succeeded attempt, each kept, and the same attempt under a second event identifier refused; cache `included`, `separate` and `not_known` with the net input of each, an unknown cache count and a broken relation; counts that are not known beside zero, in sums and in numbers of unknowns, per usage grade; an estimated, a billed, an invoiced and a self-hosted cost in four sums apart, with a credit; sums per document; the fee of TokenMeter itself refused. The expected figures were worked out by hand.

Literal tests with a test clock (`open_test.go`, `event_test.go`, `read_test.go`): construction and the rules for identifiers and currencies; amount texts, malformed and well formed, with value, printed form and significant digits; every refusal of a header and of an event; repeats and conflicts field by field, texts compared as texts (`1.50` against `1.5`, `-0` against `0`), a colon that moves between two fields; batches applied whole, reported before in every state, partly reported, conflicting, over the limit; the limits of currencies and documents; every finding alone and all four in order, with the edges of the period and the fifteenth and sixteenth digit; every class of cost; every reason of eligibility alone, in combinations, and the eleven other than `voided` in one verdict, while open and after closing; a document sum outside the profile while its class sums to zero; periods that touch and overlap; an event identifier and a pair of request and attempt held by two receptions, symmetric, listed, removed by a void; `Counting` with its sequence number moving by one per write and not on repeats or refusals, and with malformed periods; `Void` with every reason from an open and a closed reception; pages of events and of receptions; the tenant boundary; times in UTC; errors that are fixed values; a scan of the source for floating point, for any call of the decimal primitive other than its constructor, for multiplication or division by anything but ten, and for comments.

The parser of amount texts is compared over 20,000 generated texts with `math/big.Rat`, used in the test only: value, printed form, significant digits and whether the profile holds it.

Property test (`property_test.go`, `model_test.go`, fixed seeds): 260 histories of 90 steps over two workspaces and two sources, with periods, snapshots and some event identifiers drawn from small sets so that receptions collide. A reference model written in the test decides the outcome of every operation and a verdict for the reception concerned after every step; seven times per history, for every reception, findings and classes are derived by the test from the events that `Events` returns, figures are recomputed from them, and a second verdict is computed from what `Reception`, `Events` and `Receptions` return; all must equal the registry's. Read from the registry besides: no two eligible receptions of a source overlap or share an event; an eligible reception is closed, complete and without an event in error; at most one reception that is not voided per export; `Counting` of every period returns a reception exactly when it is eligible and has that period, with the model's sequence number. The test fails unless named outcomes reached their minimum, among them 300 eligible receptions, every one of twelve reasons, four findings and four classes on at least twenty different receptions or events, and thirteen kinds of refusal at least twenty times.

An export of 100,000 events in batches of 1,000 is received, closed and judged, with its sums equal to the expected ones; it has one class of cost and events of one shape.

Concurrency under the race detector: six goroutines reporting batches to one reception while three close it and three read, over thirty rounds with the close delayed differently, after which the events kept are exactly those of the accepted batches; and over forty rounds, one batch reported by eight goroutines while another closes, counted once, eight receptions of one export of which one succeeds and the others name it, and a void against four readers of eligibility and of what counts.

Limits of these tests: the model is the same rules written a second time by the same author. As the change review noted, its classes, its figures, the order of refusals in a batch and the order of reasons repeat the structure of the code; what pins those independently are the literal tests. Independent in the model are the value of an amount, by `big.Rat`, and sharing, computed from lists of events against the registry's index. A second verdict cannot know sharing with a reception that is still open, since its events cannot be read; there it is compared apart from sharing. The model does not cover malformed identifiers, limits of pages, or more than the generator's forms of malformed requests.

# Corrections made on the way

- The property test agreed with the registry on its first run; two reasons were reached too rarely and the generator was steered towards them.
- The first mutation run (296 rows) caught 269. Fourteen of the others showed missing tests, which were added, among them a cost of unknown grade or origin that another rule had refused, a stored event sharing the caller's cost, a page sharing findings, and sharing by a request alone. Four rows could not differ from the code and were removed: a malformed token total read as not stated, which the count of stated totals still refuses; two marks in prints that are redundant under length prefixes; and judging by the presence of figures, which only a voided reception has without being closed. Ten did not compile and were rewritten.
- The change review led to one change of behavior, a document sum outside the profile, and to tests: see the plan's change review resolution. A sentence in the request for that review said fifteen first-run survivors; the number is fourteen.

# Mutation check

Single mutations of the four source files of `modules/tokenmeter` were applied one at a time by a Python script, and the package's tests were run after each. The per-row results were appended to files that were never overwritten. The script and the result files are kept outside the repository, so the table below cannot be audited from the change alone.

- First run, 296 rows: 269 caught; handled as described above.
- Second run, 292 rows, before the change review: 292 of 292 caught.
- Third run, 293 rows, all against the code after the change review: 293 of 293 caught. Only that run is reported below.

| No. | Fault introduced | Result |
| --- | --- | --- |
| 1 | Identifier length unbounded | Caught |
| 2 | An empty identifier | Caught |
| 3 | A slash in an identifier | Caught |
| 4 | A space in an identifier | Caught |
| 5 | An at sign in an identifier | Caught |
| 6 | Third currency letter unchecked | Caught |
| 7 | First currency letter unchecked | Caught |
| 8 | Second currency letter unchecked | Caught |
| 9 | A lower-case currency letter | Caught |
| 10 | A revision above the maximum | Caught |
| 11 | The longest identifier refused | Caught |
| 12 | Any outcome | Caught |
| 13 | Any usage grade | Caught |
| 14 | Any cache relation | Caught |
| 15 | Any cost grade | Caught |
| 16 | Any cost origin | Caught |
| 17 | Any void reason | Caught |
| 18 | Limit not required: l.Receptions > 0 | Caught |
| 19 | Limit not required: l.Events > 0 | Caught |
| 20 | Limit not required: l.Batch > 0 | Caught |
| 21 | Limit not required: l.Currencies > 0 | Caught |
| 22 | Limit not required: l.Documents > 0 | Caught |
| 23 | Limit not required: l.Page > 0 | Caught |
| 24 | Limit not required: l.Listed > 0 | Caught |
| 25 | Limit not required: l.MaxPeriod > 0 | Caught |
| 26 | A sum not rescaled to a finer amount | Caught |
| 27 | A coarser amount not rescaled | Caught |
| 28 | Sums equal up to a power of ten | Caught |
| 29 | Numbers not reduced | Caught |
| 30 | A number below one without its leading zero | Caught |
| 31 | A negative sum printed without its sign | Caught |
| 32 | Every amount fits the profile | Caught |
| 33 | Amount texts of any length | Caught |
| 34 | An amount text of forty bytes refused | Caught |
| 35 | A count of nineteen digits | Caught |
| 36 | A count of eighteen digits refused | Caught |
| 37 | A total of thirty-one digits | Caught |
| 38 | A total of thirty digits refused | Caught |
| 39 | Sixteen digits from a float | Caught |
| 40 | Fifteen digits from a float refused | Caught |
| 41 | An amount with a leading zero | Caught |
| 42 | An amount with a plus sign | Caught |
| 43 | An amount that ends in its point | Caught |
| 44 | An exponent of three digits | Caught |
| 45 | A capital exponent refused | Caught |
| 46 | Text after an amount | Caught |
| 47 | A whole number with a leading zero | Caught |
| 48 | The exponent applied the wrong way | Caught |
| 49 | The exponent ignored | Caught |
| 50 | A positive exponent not expanded | Caught |
| 51 | A negative amount read as positive | Caught |
| 52 | Zeros at the end significant | Caught |
| 53 | Zeros at the front significant | Caught |
| 54 | Zero has no significant digit | Caught |
| 55 | Whole numbers of any length | Caught |
| 56 | Construction without limits | Caught |
| 57 | Construction without a profile | Caught |
| 58 | Construction without a clock | Caught |
| 59 | Time going backwards | Caught |
| 60 | The registry's time kept in the clock's zone | Caught |
| 61 | A registry that was not built | Caught |
| 62 | Prints without lengths | Caught |
| 63 | A start that is not a whole second | Caught |
| 64 | An end that is not a whole second | Caught |
| 65 | A period of no length | Caught |
| 66 | Periods of any length | Caught |
| 67 | The longest period refused | Caught |
| 68 | A header without a workspace | Caught |
| 69 | A header without a source | Caught |
| 70 | A header without a snapshot | Caught |
| 71 | A header without a schema version | Caught |
| 72 | A header without a connection | Caught |
| 73 | A header with any period | Caught |
| 74 | Revision zero | Caught |
| 75 | Revisions unbounded | Caught |
| 76 | Any unit | Caught |
| 77 | A header that declares no event | Caught |
| 78 | Any way of producing | Caught |
| 79 | Any kind of numeric | Caught |
| 80 | Any declaration | Caught |
| 81 | A period kept in the caller's zone | Caught |
| 82 | Not part of a header: the source | Caught |
| 83 | Not part of a header: the snapshot | Caught |
| 84 | Not part of a header: the start | Caught |
| 85 | Not part of a header: the end | Caught |
| 86 | Not part of a header: the revision | Caught |
| 87 | Not part of a header: how it was produced | Caught |
| 88 | Not part of a header: the kind of numeric | Caught |
| 89 | Not part of a header: the number of events | Caught |
| 90 | Not part of a header: the declaration | Caught |
| 91 | A total not stated leaves no mark in the header | Caught |
| 92 | Not part of a header: a token total | Caught |
| 93 | A total for a kind that does not exist | Caught |
| 94 | A declared cost in a malformed currency | Caught |
| 95 | A malformed declared cost dropped | Caught |
| 96 | An imprecise float total never noted | Caught |
| 97 | A long decimal total called imprecise | Caught |
| 98 | Declared costs compared by value | Caught |
| 99 | Not part of a header: the currency of a declared cost | Caught |
| 100 | Declared currencies not counted towards the limit | Caught |
| 101 | More events declared than the limit | Caught |
| 102 | More currencies declared than the limit | Caught |
| 103 | The limit of events refused | Caught |
| 104 | The limit of currencies refused | Caught |
| 105 | A different header under a known identifier | Caught |
| 106 | A repeated header answered as open | Caught |
| 107 | Two receptions of one export | Caught |
| 108 | The occupant not named | Caught |
| 109 | Receptions without limit | Caught |
| 110 | One reception over the limit | Caught |
| 111 | A new reception without the time of its change | Caught |
| 112 | A new reception without the time it was opened | Caught |
| 113 | Opening does not move the sequence number | Caught |
| 114 | An event without a request | Caught |
| 115 | An event without an attempt | Caught |
| 116 | An event without a consumer | Caught |
| 117 | An event without a provider | Caught |
| 118 | An event without a model | Caught |
| 119 | A malformed version | Caught |
| 120 | A malformed endpoint | Caught |
| 121 | A version required | Caught |
| 122 | An endpoint required | Caught |
| 123 | An event without a time | Caught |
| 124 | An event time with a fraction | Caught |
| 125 | An event with any outcome | Caught |
| 126 | An event with any usage grade | Caught |
| 127 | An event with any cache relation | Caught |
| 128 | An event time kept in the caller's zone | Caught |
| 129 | Not part of an event: the request | Caught |
| 130 | Not part of an event: the attempt | Caught |
| 131 | Not part of an event: the time | Caught |
| 132 | Not part of an event: the consumer | Caught |
| 133 | Not part of an event: the provider | Caught |
| 134 | Not part of an event: the model | Caught |
| 135 | Not part of an event: the version | Caught |
| 136 | Not part of an event: the endpoint | Caught |
| 137 | Not part of an event: the outcome | Caught |
| 138 | Not part of an event: the usage grade | Caught |
| 139 | Not part of an event: the cache relation | Caught |
| 140 | Not part of an event: the counts | Caught |
| 141 | A malformed count read as not known | Caught |
| 142 | Every count taken as input | Caught |
| 143 | A malformed amount | Caught |
| 144 | A cost in a malformed currency | Caught |
| 145 | A cost of any grade | Caught |
| 146 | A cost of any origin | Caught |
| 147 | Invoiced from another origin | Caught |
| 148 | Invoiced without a document | Caught |
| 149 | A document on a cost that is not invoiced | Caught |
| 150 | An estimated cost classed by its origin | Caught |
| 151 | A self-hosted cost among the provider's | Caught |
| 152 | A billed cost among the invoiced | Caught |
| 153 | A computed cost among the invoiced | Caught |
| 154 | A cost of unknown grade among the invoiced | Caught |
| 155 | An estimated self-hosted cost classed as self-hosted | Caught |
| 156 | Not part of an event: the amount | Caught |
| 157 | Not part of an event: the currency | Caught |
| 158 | Not part of an event: the grade | Caught |
| 159 | Not part of an event: the origin | Caught |
| 160 | Not part of an event: the document | Caught |
| 161 | Amounts compared by value | Caught |
| 162 | A stored event shares the caller's cost | Caught |
| 163 | An event before the period | Caught |
| 164 | An event at the end of the period | Caught |
| 165 | An event at the start of the period refused | Caught |
| 166 | A separate cache held against the input | Caught |
| 167 | An included cache beside an input that is not known | Caught |
| 168 | A cache equal to its input broken | Caught |
| 169 | A cache larger than the input that includes it | Caught |
| 170 | A long decimal amount found imprecise | Caught |
| 171 | A float amount of sixteen digits without a finding | Caught |
| 172 | A cost outside the profile without a finding | Caught |
| 173 | A shared pair of request and attempt unseen | Caught |
| 174 | A shared event identifier unseen | Caught |
| 175 | Sharing by the request alone | Caught |
| 176 | An empty batch judged before the reception is looked up | Caught |
| 177 | An empty batch | Caught |
| 178 | An event twice in a batch | Caught |
| 179 | A pair twice in a batch | Caught |
| 180 | Batches of any size | Caught |
| 181 | A batch at the limit refused | Caught |
| 182 | A changed event taken as a repeat | Caught |
| 183 | One attempt under two event identifiers | Caught |
| 184 | A repeat judged by the state | Caught |
| 185 | An event for a reception that is not open | Caught |
| 186 | More events than declared | Caught |
| 187 | The last declared event refused | Caught |
| 188 | Repeats counted towards the declared number | Caught |
| 189 | The fee of TokenMeter received | Caught |
| 190 | A known currency counted again | Caught |
| 191 | A new currency counted per event | Caught |
| 192 | A known document counted again | Caught |
| 193 | A new document counted per event | Caught |
| 194 | No document counted as one | Caught |
| 195 | Currencies without limit | Caught |
| 196 | Documents without limit | Caught |
| 197 | The limit of currencies refused for events | Caught |
| 198 | The limit of documents refused | Caught |
| 199 | Events never examined | Caught |
| 200 | Pairs not remembered | Caught |
| 201 | Currencies of events not remembered | Caught |
| 202 | Documents not remembered | Caught |
| 203 | Sharing known to the later reception only | Caught |
| 204 | Sharing known to the earlier reception only | Caught |
| 205 | Identities not indexed | Caught |
| 206 | Events do not move the sequence number | Caught |
| 207 | Events in map order | Caught |
| 208 | Closing what was voided open | Caught |
| 209 | Closing again refused or redone | Caught |
| 210 | Closing does not move the sequence number | Caught |
| 211 | Closing without the time of the change | Caught |
| 212 | Outcomes counted without events in error | Caught |
| 213 | Events in error not counted | Caught |
| 214 | Unknown counts not counted | Caught |
| 215 | An unknown count summed as zero | Caught |
| 216 | Token sums hold the last count | Caught |
| 217 | Estimated usage left out of the comparison with the manifest | Caught |
| 218 | A broken cache relation enters the net input | Caught |
| 219 | Included cache not taken off the net input | Caught |
| 220 | Separate cache taken off the net input | Caught |
| 221 | An unknown relation read as separate | Caught |
| 222 | An unknown cache count read as zero | Caught |
| 223 | The cost of an event in error summed | Caught |
| 224 | Costs of all currencies in one sum | Caught |
| 225 | Estimated costs left out of the comparison with the manifest | Caught |
| 226 | Document sums empty | Caught |
| 227 | Events per document not counted | Caught |
| 228 | A sum outside the profile unnoticed | Caught |
| 229 | A document sum outside the profile unnoticed | Caught |
| 230 | A declared currency without a cost accepted | Caught |
| 231 | A cost in an undeclared currency accepted | Caught |
| 232 | A differing cost total accepted | Caught |
| 233 | A differing token total accepted | Caught |
| 234 | A declared total without any known count accepted | Caught |
| 235 | A void for any reason | Caught |
| 236 | A second void reason accepted | Caught |
| 237 | The events of a reception voided open not shown | Caught |
| 238 | A void without the time of the change | Caught |
| 239 | A void does not free the export | Caught |
| 240 | A void does not move the sequence number | Caught |
| 241 | Listed receptions in the order received | Caught |
| 242 | Receptions listed without bound | Caught |
| 243 | An open reception eligible | Caught |
| 244 | A list read eligible | Caught |
| 245 | A derived declaration eligible | Caught |
| 246 | An imprecise float total eligible | Caught |
| 247 | Reasons in another order | Caught |
| 248 | A voided reception still conflicts and shares | Caught |
| 249 | Periods that touch overlap | Caught |
| 250 | Only identical periods conflict | Caught |
| 251 | A list read does not conflict | Caught |
| 252 | An open reception does not conflict | Caught |
| 253 | An open reception does not share | Caught |
| 254 | A period conflict eligible | Caught |
| 255 | A shared event eligible | Caught |
| 256 | Events in error eligible | Caught |
| 257 | An incomplete export eligible | Caught |
| 258 | Totals not stated eligible | Caught |
| 259 | Differing totals eligible | Caught |
| 260 | A sum outside the profile eligible | Caught |
| 261 | Eligible with reasons against it | Caught |
| 262 | A voided reception judged further | Caught |
| 263 | Counting for a malformed period | Caught |
| 264 | Counting whatever the start | Caught |
| 265 | Counting whatever the end | Caught |
| 266 | A reception that is not eligible counts | Caught |
| 267 | What counts returned without the sequence number | Caught |
| 268 | What counts returned without its snapshot | Caught |
| 269 | The sequence number not returned when nothing counts | Caught |
| 270 | The view shares its declared totals | Caught |
| 271 | The view shares its figures | Caught |
| 272 | The void reason not shown | Caught |
| 273 | That it was closed not shown | Caught |
| 274 | A negative position | Caught |
| 275 | A page of nothing | Caught |
| 276 | A page over the limit | Caught |
| 277 | The events of an open reception read | Caught |
| 278 | A position beyond the end | Caught |
| 279 | Pages of events not cut | Caught |
| 280 | A page shares the findings | Caught |
| 281 | A page shares the cost | Caught |
| 282 | A malformed lower bound | Caught |
| 283 | A page of no receptions | Caught |
| 284 | A page of receptions over the limit | Caught |
| 285 | The last identifier read returned again | Caught |
| 286 | Receptions in the order received | Caught |
| 287 | Pages of receptions not cut | Caught |
| 288 | Copied figures without token sums | Caught |
| 289 | Copied figures without cost sums | Caught |
| 290 | Copied figures without document sums | Caught |
| 291 | Copied figures without unknown counts | Caught |
| 292 | Copied figures without outcomes | Caught |
| 293 | Copied figures without the unknown net inputs | Caught |

# Deviations from the plan

None after the plan was amended with the change review resolution. Where the rules of the design re-review are more precise than the sentences above them, the code follows the rules.

# Independent review

- Design: two read-only passes on 2026-11-13 (FAIL with 5 HIGH and no redesign asked; then PASS with findings and no HIGH). See the plan's two resolution sections.
- Change: one read-only pass on 2026-11-13, PASS with findings (2 MEDIUM about tests, 8 LOW), nothing executed by the reviewer. All findings were resolved or stated. The corrections made after it, one of them a change of behavior, were not reviewed again by a separate reviewer; they are covered by the tests and the third mutation run.

# Provisional choices awaiting the owner

Not the conservative choices, listed in the plan: `Close` accepts a reception with fewer events than declared; how the export was produced, the kind of numeric and whether the declaration is TokenMeter's own are the adapter's flags; an eligible reception may consist of estimated usage only; a void works as a choice between two conflicting receptions. Conservative choices that cost availability: any overlap of periods and any shared event make every reception concerned not eligible; a float source whose total has more than fifteen significant digits is never eligible; one event with a finding makes its export not eligible; a document sum outside the profile does too; an export without any event cannot be reported; an export that names more documents or currencies than the limit cannot be received whole. Assumptions: the names of outcomes, usage grades, cost grades and origins; event identifiers and pairs of request and attempt that are unique and stable across the exports of a source, without which only one reception per source can ever be eligible; one event per attempt; a manifest total that covers all four classes. Undecided and not chosen here: TokenMeter's interface, its snapshot export and authentication; rates, denominators, model aliases and which outcome may be sold; how an invoice-referenced cost is reconciled with its document, reaches a billing and is kept from being used twice; how a self-hosted cost meets HawkEye's; who may void.

# Not verified

Any real TokenMeter, an adapter, authentication and tenant binding of the connection, a process that dies, persistence, memory under many large exports, any operating system other than the one named, and everything listed as deferred in the plan. No deployment, payment, message, real account or customer data was involved.
