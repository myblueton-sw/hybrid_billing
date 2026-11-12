---
type: VerificationRecord
title: Source artifacts, chunked staging and input manifests verification
description: Records what was executed and reviewed for the first model of work item W-06, with results, the corrections made on the way, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-186](https://linear.app/hybrid-billing/issue/HB-186/w-06-source-artifacts-staged-in-chunks-record-keys-with-repeats) adds the Go package `modules/intake` from main `1e7e33d`: source documents of a subject received as versions; artifacts registered once with their declared content and the results of three checks; rows staged in chunks that can be sent again in any order; sealing only when what was staged equals what was declared; extractions bound to a verification of exactly their content; the highest live version of a document replacing earlier ones as a whole; and numbered, immutable input manifests published only when no document of the subject is in conflict or unreadable, no event key is in conflict and no sum leaves the decimal profile.

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan three times before implementation and the implemented change afterwards (see Independent review). Machine: mac_mini.

Nothing calls the package: no bytes are stored, scanned or parsed, no provider is called, and the Python import path is not connected. The registry checks that what a caller reports is complete, consistent and equal to what it declared; it cannot know that a row was read correctly. The acceptance of HB-06 is about real files and is NOT_RUN, as are BILL01, COM01, FD03, DAT01, AUD-AR-02, AUD-AR-03, AUD-AR-05, AUD-DB-06 and AUD-RT-01. W-06 is not complete.

# Executed checks

Executed on 2026-11-13 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain. No other operating system was tested.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All fifteen packages `ok`, including `modules/intake`; run again after the last correction |
| `go test -count=40 ./modules/intake/` | `ok`: forty consecutive runs without a failure, before the first mutation run and again after the corrections that followed the change review |

The acceptance at package level (`acceptance_test.go`):

| Test | What it shows |
| --- | --- |
| 120,000 records in 120 chunks, four currencies, negative amounts, one currency summing to zero | Staged straight through in one registry and, in another, broken off after 50 chunks, resumed from chunk 31, then in a shuffled order with repeats: the same row count, sums and record digest; a manifest with the same content digest; totals equal to the generator's; both digests equal to a second writing of the stated encoding in the test; every record read back in pages |
| A second receipt of the same version with one amount changed | Sealed, and the subject is blocked as a document in conflict; the published manifest is unchanged |
| A provider's final state, 60 rounds | Documents with corrections, removed rows, rows that reappear in another document, shifted locator rows and versions that end empty, delivered as versions and sealed in a random order with publications in between: once every version is sealed, the latest manifests of all subjects hold exactly the provider's final rows, compared as a multiset of subject, amount, currency and row digest |

Two in-memory registries show independence from the order and repetition of calls, not a process that died. The ground-truth test gives every document its own event keys, so the single count of an event key shared by two documents rests on the literal test and the property test.

Literal tests with a test clock (`register_test.go`, `stage_test.go`, `manifest_test.go`): every refusal and every reason for rejection at registration, alone and together, with the boundaries of size, ratio, chunk count and chunk capacity; a document's subject and key kind fixed by its first recorded artifact and not by a refused one; identical and differing registrations field by field; the originals a derived artifact may name; chunk refusals; repeats in any order; ten ways to contradict a staged chunk; every reason `Seal` is refused, listed together; the sealed digest against the stated encoding and independent of chunking; sums whose partial sums leave the profile in one order of arrival; verification before all chunks, with another digest, repeated, by another verifier, and other content afterwards; abandonment and retirement with every reason; the record limit with records given back; time that does not go backwards; a later version dropping and shifting rows and the earlier one counting again after a retirement; equal and differing receipts of one version and attribution to the earliest sealed; a scan unreadable until its extraction is sealed, two extractions that disagree, an extraction not counting after its original is retired; mixed artifacts at one revision; event keys across documents and locators kept apart; a subject sum outside the profile; bounded lists with totals; documents behind for each cause; documents without a live artifact; an emptied document published; equal bytes under two documents; the six checks of publication in order; paging; views that share no state; errors that repeat nothing of the request.

Property tests (`property_test.go`, `model_test.go`, fixed seeds): 700 histories of 110 raw attempts over registration, staging, verification, sealing, abandonment, retirement and publication in two workspaces, sixty percent of the steps steered so that artifacts do get sealed and published. A reference model written in the test decides from each attempt the outcome, including the list of gaps and blockers; after every step the registry's outcome equals the model's, every written manifest equals the model's rows, attribution, sums and annotations, holds no two rows of one key and no row of an artifact that is not live, and no published manifest changes later; `Status` is compared every seventh step; at the end every artifact's state and every workspace's record count equal the model's. The test fails unless the histories reached 46 named outcomes, among them the error kinds of each operation that the histories can produce, each kind of blocker but the profile bound, and each answer of `Status`. A second property seals the same artifacts in two random orders and compares blockers and manifest content.

Concurrency under the race detector (`concurrency_test.go`): eight goroutines staging the same twenty chunks, each counted once; sixteen publications with one expected number, of which one is written and all are answered; six goroutines registering, sealing, retiring and publishing against one subject.

Limits of these tests: the model is the same rules written a second time by the same author; it is independent in its arithmetic (64-bit integers), its digest (the second writing of the encoding), its validation and its grouping of rows, but the order in which refusals take precedence mirrors the code, so agreement there shows consistency only; it does not compare which document a key conflict names and knows no profile bound; expectations in the tables were worked out by hand.

# Corrections made on the way

- Before the mutation runs, five pieces of code that could not change a result were removed (a zero check before a loop that handles zero, a negative row count caught by two later checks, a role check implied by what an original holds, a content comparison implied by the attribution comparison, and the second half of the empty-manifest check) and seven tests were added for cases the planned mutants showed to be untested.
- The first mutation run left four mutants alive: two event-key comparisons (the test rows also differed in their row digest), an original that does not say what it holds, and the order of blockers by kind. The tests were corrected; ten further mutants did not compile and were rewritten.
- The change review led to three changes of behaviour: a subject sum outside the profile blocks publication, the sealing sequence is per workspace, and a key conflict names the document that holds the key.

# Mutation check

253 single-line mutations of `modules/intake` were applied one at a time by a Python script, and the package's tests were run after each. The per-row results were appended to files that were never overwritten.

- First run, 250 rows: 236 caught, 4 not caught, 10 not compiling. After the corrections above those 14 rows were run again: all caught.
- After the change review the list had 253 rows and all were run again against the final code. Only that run is reported below: 253 of 253 caught.

| No. | Fault introduced | Result |
| --- | --- | --- |
| 1 | Identifier length unbounded | Caught |
| 2 | An empty identifier | Caught |
| 3 | A slash in an identifier | Caught |
| 4 | A short digest | Caught |
| 5 | Digest length unbounded | Caught |
| 6 | A digest in capitals | Caught |
| 7 | Third currency letter unchecked | Caught |
| 8 | First currency letter unchecked | Caught |
| 9 | Second currency letter unchecked | Caught |
| 10 | Subject without workspace | Caught |
| 11 | Subject without provider | Caught |
| 12 | Subject without scope | Caught |
| 13 | Subject without period | Caught |
| 14 | A derived artifact treated as holding no rows | Caught |
| 15 | Any abandon reason | Caught |
| 16 | The operator abandon reason unknown | Caught |
| 17 | Any retire reason | Caught |
| 18 | The wrong-source retire reason unknown | Caught |
| 19 | Limit: size not required | Caught |
| 20 | Limit: ratio not required | Caught |
| 21 | Limit: chunks not required | Caught |
| 22 | Limit: records per chunk not required | Caught |
| 23 | Limit: artifacts not required | Caught |
| 24 | Limit: records not required | Caught |
| 25 | Limit: manifests not required | Caught |
| 26 | Limit: page not required | Caught |
| 27 | Limit: list bound not required | Caught |
| 28 | Sum not rescaled to a finer amount | Caught |
| 29 | A coarser amount not rescaled | Caught |
| 30 | Sums not reduced to canonical form | Caught |
| 31 | The sign of a negative sum lost | Caught |
| 32 | A sum below one without its leading zero | Caught |
| 33 | Sums equal up to a power of ten | Caught |
| 34 | Fields without their length | Caught |
| 35 | An event key with another currency is the same row | Caught |
| 36 | An event key with another row digest is the same row | Caught |
| 37 | An event key with another amount is the same row | Caught |
| 38 | Record digest in map order | Caught |
| 39 | Record digest without keys | Caught |
| 40 | Record digest without amounts | Caught |
| 41 | Record digest without currencies | Caught |
| 42 | Record digest without row digests | Caught |
| 43 | Chunk comparison ignores keys | Caught |
| 44 | Chunk comparison ignores amounts | Caught |
| 45 | Chunk comparison ignores currencies | Caught |
| 46 | Chunk comparison ignores row digests | Caught |
| 47 | Construction without limits | Caught |
| 48 | Construction without a profile | Caught |
| 49 | Construction without a clock | Caught |
| 50 | Time going backwards | Caught |
| 51 | Registration without a valid subject | Caught |
| 52 | Registration with a malformed identifier | Caught |
| 53 | Registration without a document | Caught |
| 54 | Registration without a byte digest | Caught |
| 55 | Registration without a location | Caught |
| 56 | An unknown key kind | Caught |
| 57 | Revision zero | Caught |
| 58 | Size zero | Caught |
| 59 | Expanded size below size | Caught |
| 60 | Negative chunk count | Caught |
| 61 | A malformed collection reference | Caught |
| 62 | An unknown scan result | Caught |
| 63 | An unknown syntax result | Caught |
| 64 | An unknown bodies result | Caught |
| 65 | An original that does not say what it holds | Caught |
| 66 | An original naming an original | Caught |
| 67 | An original naming a method | Caught |
| 68 | A derived artifact saying what it holds | Caught |
| 69 | A derived artifact without an original | Caught |
| 70 | A derived artifact without a method | Caught |
| 71 | An unknown role | Caught |
| 72 | A malformed currency of a declared sum | Caught |
| 73 | An empty artifact declaring a chunk | Caught |
| 74 | An empty artifact declaring a row | Caught |
| 75 | An empty artifact declaring a sum | Caught |
| 76 | An empty artifact stating an origin of totals | Caught |
| 77 | Totals without an origin | Caught |
| 78 | Rows without a sum | Caught |
| 79 | Fewer rows than chunks | Caught |
| 80 | More currencies than rows | Caught |
| 81 | Chunk limit not applied | Caught |
| 82 | Chunk limit one too low | Caught |
| 83 | Rows over the capacity of the chunks | Caught |
| 84 | One row over capacity | Caught |
| 85 | Exactly full chunks refused | Caught |
| 86 | Size at the limit rejected | Caught |
| 87 | Size limit not applied | Caught |
| 88 | A ratio far over the limit | Caught |
| 89 | A ratio just over the limit | Caught |
| 90 | A ratio at the limit rejected | Caught |
| 91 | Scan reasons swapped | Caught |
| 92 | Syntax check ignored | Caught |
| 93 | Bodies check ignored | Caught |
| 94 | A check that was not run accepted | Caught |
| 95 | An identifier registered again with other content | Caught |
| 96 | A document under another subject | Caught |
| 97 | A document with another key kind | Caught |
| 98 | An extraction of an original that holds content | Caught |
| 99 | An extraction of another document's original | Caught |
| 100 | An extraction of another revision | Caught |
| 101 | An extraction of a staging original refused | Caught |
| 102 | An extraction of a sealed original refused | Caught |
| 103 | An extraction of an ended original | Caught |
| 104 | One artifact over the limit | Caught |
| 105 | The highest revision lowered | Caught |
| 106 | Declared sums stored as written | Caught |
| 107 | Registration time not recorded | Caught |
| 108 | Change time not set at registration | Caught |
| 109 | A rejected artifact left staging | Caught |
| 110 | A rejected artifact shows staged content | Caught |
| 111 | Records not given back | Caught |
| 112 | Rows of an ended artifact kept | Caught |
| 113 | Chunk number zero | Caught |
| 114 | An empty chunk | Caught |
| 115 | A chunk over the limit | Caught |
| 116 | A full chunk refused | Caught |
| 117 | A malformed row key | Caught |
| 118 | A malformed row digest | Caught |
| 119 | A malformed row currency | Caught |
| 120 | A chunk number over the declared count | Caught |
| 121 | The last chunk refused | Caught |
| 122 | A repeated chunk is a contradiction | Caught |
| 123 | A chunk number staged again with other records accepted | Caught |
| 124 | Staging into an ended artifact | Caught |
| 125 | A staged chunk replaced by other rows | Caught |
| 126 | A row key of another chunk accepted | Caught |
| 127 | A row key twice in one chunk accepted | Caught |
| 128 | A contradiction leaves the artifact staging | Caught |
| 129 | The contradicting chunk not recorded | Caught |
| 130 | The time of a contradiction not recorded | Caught |
| 131 | An inconsistent artifact keeps its records | Caught |
| 132 | Record limit not applied | Caught |
| 133 | Record limit one too low | Caught |
| 134 | Staged amounts not summed | Caught |
| 135 | A staged chunk not remembered | Caught |
| 136 | Staged rows not counted | Caught |
| 137 | Staged rows not counted against the artifact | Caught |
| 138 | Staged rows not counted against the workspace | Caught |
| 139 | Staged content of an ended artifact shown | Caught |
| 140 | Always complete | Caught |
| 141 | Verification with a malformed digest | Caught |
| 142 | Verification without a verifier | Caught |
| 143 | Verification with free text as evidence | Caught |
| 144 | Verification of an original | Caught |
| 145 | A second verification of another digest | Caught |
| 146 | A second verification by another verifier | Caught |
| 147 | A second verification with other evidence | Caught |
| 148 | Verification of an ended artifact | Caught |
| 149 | Verification before every chunk is staged | Caught |
| 150 | Verification of other content | Caught |
| 151 | Verification time not recorded | Caught |
| 152 | Sealing with chunks missing | Caught |
| 153 | Sealing with another row count | Caught |
| 154 | Sealing with an undeclared currency | Caught |
| 155 | Sealing with a declared currency that has no rows | Caught |
| 156 | Sealing with another sum | Caught |
| 157 | Gaps in map order | Caught |
| 158 | Sealing an extraction before its original | Caught |
| 159 | Sealing an unverified extraction | Caught |
| 160 | Sealing again after retirement refused | Caught |
| 161 | Sealing again refused | Caught |
| 162 | Sealing an ended artifact | Caught |
| 163 | Sealing despite gaps | Caught |
| 164 | Sealing sequence not advanced | Caught |
| 165 | Sealing sequence not assigned | Caught |
| 166 | Record digest not stored | Caught |
| 167 | Sealing time not recorded | Caught |
| 168 | Abandoning with an unknown reason | Caught |
| 169 | Abandoning again with another reason | Caught |
| 170 | Abandoning again refused | Caught |
| 171 | Abandoning a sealed artifact | Caught |
| 172 | An abandoned artifact keeps its records | Caught |
| 173 | Abandon time not recorded | Caught |
| 174 | Abandoning changes nothing | Caught |
| 175 | Retiring with an unknown reason | Caught |
| 176 | Retiring again with another reason | Caught |
| 177 | Retiring again refused | Caught |
| 178 | Retiring a staging artifact | Caught |
| 179 | A retired artifact keeps its records | Caught |
| 180 | Retire time not recorded | Caught |
| 181 | Retiring changes nothing | Caught |
| 182 | The artifact view shares its declared sums | Caught |
| 183 | The artifact view shares its verification | Caught |
| 184 | An artifact that is not sealed counts | Caught |
| 185 | An extraction of a retired original counts | Caught |
| 186 | A scan awaiting extraction carries content | Caught |
| 187 | Staging artifacts not counted | Caught |
| 188 | The last registered live revision counts | Caught |
| 189 | Every document behind | Caught |
| 190 | Behind documents not reported | Caught |
| 191 | A document without a live artifact blocks | Caught |
| 192 | An ended artifact of the counting revision counts | Caught |
| 193 | Earlier revisions mixed in | Caught |
| 194 | A scan compared as content | Caught |
| 195 | Differing receipts of one version accepted | Caught |
| 196 | Rows attributed to the first registered receipt | Caught |
| 197 | An unreadable document does not block | Caught |
| 198 | A document in conflict does not block | Caught |
| 199 | A document in conflict is read from its first receipt | Caught |
| 200 | Behind documents in registration order | Caught |
| 201 | A document paired with itself | Caught |
| 202 | A pair reported once per shared file | Caught |
| 203 | Event rows attributed in document order | Caught |
| 204 | Locators compared across documents | Caught |
| 205 | A locator and an event key of the same text meet | Caught |
| 206 | One key conflict listed once per document | Caught |
| 207 | An event key in conflict does not block | Caught |
| 208 | Blockers not ordered by kind | Caught |
| 209 | Manifest records not ordered by document | Caught |
| 210 | Manifest records in reverse key order | Caught |
| 211 | Manifest sums not added | Caught |
| 212 | Attribution digest without the artifact | Caught |
| 213 | Manifest digests without the document | Caught |
| 214 | Manifest digests without the key kind | Caught |
| 215 | Manifest digests without the row digest | Caught |
| 216 | Lists not bounded | Caught |
| 217 | The total of a bounded list | Caught |
| 218 | Blockers not bounded | Caught |
| 219 | Publication for a malformed subject | Caught |
| 220 | Publication despite blockers | Caught |
| 221 | The total of bounded blockers | Caught |
| 222 | A repeated publication is checked against the expected number | Caught |
| 223 | A change of attribution alone publishes nothing | Caught |
| 224 | Publication with a stale expected number | Caught |
| 225 | An empty first manifest | Caught |
| 226 | An empty successor refused | Caught |
| 227 | Manifest limit not applied | Caught |
| 228 | One manifest over the limit | Caught |
| 229 | Manifest without its record count | Caught |
| 230 | Manifest without sums | Caught |
| 231 | Manifest without the staging count | Caught |
| 232 | Manifest without behind documents | Caught |
| 233 | Manifest without equal-bytes pairs | Caught |
| 234 | Manifest without its time | Caught |
| 235 | A written manifest not reported as written | Caught |
| 236 | Status without the staging count | Caught |
| 237 | Status of a blocked subject compared anyway | Caught |
| 238 | A subject in which nothing counts differs | Caught |
| 239 | A change of content shown as attribution | Caught |
| 240 | A change of attribution not shown | Caught |
| 241 | Reading a manifest of a malformed subject | Caught |
| 242 | The latest manifest not found | Caught |
| 243 | The manifest view shares its sums | Caught |
| 244 | The manifest view shares its behind list | Caught |
| 245 | The manifest view shares its pairs | Caught |
| 246 | A page of no records | Caught |
| 247 | A page over the limit | Caught |
| 248 | A full page refused | Caught |
| 249 | Pages not cut | Caught |
| 250 | A page shares the manifest's records | Caught |
| 251 | A subject sum outside the profile is published | Caught |
| 252 | A key conflict without the document that holds the key | Caught |
| 253 | The holder of an event key not recorded | Caught |

# Deviations from the plan

None in behaviour after the plan was amended with the change review resolution. The plan's "sum outside the profile" as a reason `Seal` refuses does not exist as its own gap: a staged sum must equal a declared sum that the profile accepted.

# Independent review

- Design: three read-only passes on 2026-11-13 (FAIL, FAIL on two points, PASS with findings); the first replaced the model of per-event revisions by whole-version replacement of documents. See the plan's three resolution sections.
- Change: one read-only pass on 2026-11-13, PASS with findings (1 MEDIUM, 6 LOW), nothing executed by the reviewer. All findings were resolved or stated; see the plan's change review resolution. The corrections made after it were not reviewed again by a separate reviewer; they are covered by the tests and the final mutation run above.

# Provisional choices awaiting the owner

Conservative defaults, labelled provisional: a conflict or an unreadable document blocks its whole subject; a document that is behind and two documents with equal bytes are reported and do not block (not the conservative choice, listed in the plan as undecided); a check that was not run rejects an artifact; a subject sum outside the profile blocks. Undecided and not chosen here: the stable identifier of each source and whether its rows have event keys; the size and ratio limits; the preservation store; what counts as verification of an extraction and who may give it; who may retire and publish; retention.

# Not verified

Real files, real providers, a process that dies during a transfer, persistence, any operating system other than the one named, load beyond the tests above, and everything listed as deferred in the plan. No deployment, payment, message, real account or customer data was involved.
