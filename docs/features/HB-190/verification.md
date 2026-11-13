---
type: VerificationRecord
title: Evidence artifacts, bundles, retention, holds and deletion verification
description: Records what was executed and reviewed for the first model of work item W-08, with results, the corrections made on the way, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-190](https://linear.app/hybrid-billing/issue/HB-190/w-08-evidence-artifacts-sealed-bundles-retention-holds-and-dependency) adds the Go package `modules/evidence` from main `7f4362f`: artifacts whose custody is recorded as instances that are only ever added, a product copy or a remote location with an optional verification; one predicate, held until a time; bundles of a confirmed calculation that are confirmed only when the required roles are present and every artifact is held until the retention time; legal holds; purge refused while a bundle is protected; deletion of an artifact refused while any bundle that was not purged names it; verdicts `held` and `broken` with items at risk; downloads whose digest and size never change; versions of an engine or module restricted for the whole installation; and evidence gaps bound into a bundle at confirmation.

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan twice before implementation and the implemented change afterwards (see Independent review). Machine: mac_mini.

Nothing calls the package and it stores no bytes: it records what a caller reports about copies, remote locations and losses. `held` means that every artifact a bundle names is held under the predicate, not that those artifacts are enough to reproduce a calculation. The acceptance of HB-08 is about a real store and real sources and is NOT_RUN, as are DAT01, EX01, EX09, REV04, AUD-AR-03, AUD-QA-02 and AUD-QA-06. W-08 is not complete, and its design part, which asks for the retention period, the responsibility for keys and the conditions of a remote store, stays open with the owner.

This change also corrects one sentence of `docs/features/HB-188/verification.md`, which named 2026-11-14 as a day of execution although every check of that ticket ran on 2026-11-13.

# Executed checks

Executed on 2026-11-13 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain. No other operating system was tested. During this ticket the shared Go build cache of the machine lost entries and builds failed with missing cache files; the cause was not established. From then on every Go command ran with `GOCACHE` set to a directory of the session; the owner's cache was not changed.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All seventeen packages `ok`, including `modules/evidence`; run again after the last correction |
| `go test -count=40 ./modules/evidence/` | `ok`: forty consecutive runs without a failure, before the first mutation run and again on the code after the change review; ten more after the last added test |

The acceptance at package level (`store_test.go`), against a store simulated in the test: eight artifacts of an invoice with unverified remote locations cannot be confirmed; after product copies are added the bundle is confirmed; every remote location is then removed from the store and reported expired, deleted or revoked, with the clock moved 100 days each time, and the bundle stays `held`, every download returns the same identifier, version, digest and size through the product copy, and none of the artifacts can be deleted. A bundle that rests on a verified remote location alone is `broken` once it is lost. Two artifacts cannot rest on one product location, within a workspace or across workspaces, and deleting an artifact of another workspace leaves the bytes of the first. In the simulation a product object exists until it is removed, a verified remote object expires at the end of its verification without any report, and an unverified remote object is not kept; a `held` bundle must have, for every item, an instance that is not lost and whose object exists at that moment.

Literal tests with a test clock (`artifact_test.go`, `bundle_test.go`, `read_test.go`): every refusal of an artifact and of a custody, a location that is a URL among them; repeats and conflicts of registration field by field, a repeat compared with what was registered after the verification moved; the exclusivity of product locations; thirteen steps of adding custody and twelve of reporting loss, with what each writes; a lost instance not coming back and the same artifact held again at a new location; a deleted artifact final; times kept in UTC; twenty-one cases of a refused `Confirm`, alone and all reasons together in order; the header digest by a second writing of the encoding for two bundles; items in the order of roles; gaps bound at confirmation and not those of another customer, period, workspace or already closed; repeats, conflicts and limits of `Confirm`, a repeat after an extension and after a purge; deletion refused for one and for several bundles, cut at the bound, and the order purge then delete; purge refused under retention, under a hold, under both and after an extension, and accepted at the moment protection ends; a hold and an extension arriving after the retention time and finding every artifact; everything a purged bundle refuses; holds with every refusal, positions, repeats, release and a second hold; five extensions with what they return; the edges of a verification and of protection to the nanosecond; verdicts, causes with the reason of the instance lost last, lists cut at the bound, `Exposed` and what `ReportLoss` returns, an emergency withdrawal not refused; downloads through product and verified remote instances in order, not through a lost or unverified one, and reading a document after its engine was lost and restricted; restriction by kind and version across workspaces, before registration; gaps with every refusal, closing, and what `Inspect` shows; the tenant boundary with the same identifiers in two workspaces; errors that are the package's fixed values.

Property test (`property_test.go`, fixed seed): 300 histories of 120 steps over registration, custody, loss, confirmation, extension, holds, release, purge, deletion and a clock that moves across retention and verification times, four fifths of them starting with nine registered artifacts. A reference model written in the test decides the outcome of each step, for `Confirm` with its reason codes, and the verdict, counts and protection of every bundle after every step. Read from the registry after every step, independent of the model: no bundle that was not purged names an artifact deleted through the registry; a purged bundle has no hold and no items; held until a later time implies held until an earlier one; a download never changes; and a `held` bundle has bytes in the simulated store. The test fails unless forty named outcomes were each reached at least three times.

Concurrency under the race detector: deletion against confirmation of a bundle that names the artifact, 200 rounds, exactly one succeeding; a hold or an extension against a purge, 200 rounds, exactly one succeeding and the state agreeing; six goroutines registering, confirming, losing, inspecting and restoring in one workspace.

Limits of these tests: the model is the same rules written a second time by the same author, and its predicate and the structure of its confirmation follow the code's; it does not cover gaps, restriction, a second workspace, the bounds of lists, the instances of a download, the values that `AddCustody`, `ReportLoss`, `Extend` and `PlaceHold` return, the roles and artifacts of reasons or malformed requests, which rest on the literal tests. The simulated store is driven by the test: a loss is reported whenever an object is removed, so it shows that the registry does not call something held that the test took away or let expire, not that a real store behaves so. Expectations in the tables were worked out by hand.

# Corrections made on the way

- The first property run failed because two artifacts shared one remote location in the simulation, so a loss of one removed the bytes of the other; remote objects are kept per artifact, since the registry does not make remote locations exclusive. A second failure showed that the model lacked the limit of items. After that the histories almost never confirmed a bundle, and the generator was steered.
- The first mutation run (212 rows) caught 192. Twelve of the others showed tests that were missing or did not test what they named; tests were added. Two showed checks that could change no result, which were removed and are stated in the plan; six did not compile and were rewritten.
- The change review found an invariant in the property test that could not fail and a simulated store that only followed the reports; both were corrected, with nine smaller points.

# Mutation check

211 single-line mutations of `modules/evidence` were applied one at a time by a Python script, and the package's tests were run after each. The per-row results were appended to files that were never overwritten.

- First run, 212 rows, before the change review: 192 caught; handled as described above.
- Second run, all 211 rows against the code after the change review: 210 caught, 1 not caught (the role of an item not being part of what a repeat is compared with; the test changed the order of artifacts together with the role). A case was added and that row run again: caught.
- Result reported below: 211 of 211 caught, 210 in the second run and one in the third.

| No. | Fault introduced | Result |
| --- | --- | --- |
| 1 | Identifier length unbounded | Caught |
| 2 | A URL as an identifier | Caught |
| 3 | A short digest | Caught |
| 4 | Digest length unbounded | Caught |
| 5 | A digest in capitals | Caught |
| 6 | A bundle without a result | Caught |
| 7 | A bundle without a source | Caught |
| 8 | Another order of roles | Caught |
| 9 | A location that is not an identifier | Caught |
| 10 | A product copy without a receipt | Caught |
| 11 | A product copy with a verification time | Caught |
| 12 | A product copy with evidence | Caught |
| 13 | A remote location with a receipt | Caught |
| 14 | A verification without its evidence or its time | Caught |
| 15 | Evidence that is free text | Caught |
| 16 | Any loss reason | Caught |
| 17 | Any delete reason | Caught |
| 18 | Any purge reason | Caught |
| 19 | Any release reason | Caught |
| 20 | Any gap reason | Caught |
| 21 | Limit not required: l.Artifacts > 0 | Caught |
| 22 | Limit not required: l.Bundles > 0 | Caught |
| 23 | Limit not required: l.Holds > 0 | Caught |
| 24 | Limit not required: l.Gaps > 0 | Caught |
| 25 | Limit not required: l.Instances > 0 | Caught |
| 26 | Limit not required: l.Items > 0 | Caught |
| 27 | Limit not required: l.HoldBundles > 0 | Caught |
| 28 | Limit not required: l.Restricted > 0 | Caught |
| 29 | Limit not required: l.Page > 0 | Caught |
| 30 | Limit not required: l.Listed > 0 | Caught |
| 31 | Construction without limits | Caught |
| 32 | Construction without a clock | Caught |
| 33 | Time going backwards | Caught |
| 34 | A remote location taken by a product location of the same text | Caught |
| 35 | A product location shared across workspaces | Caught |
| 36 | A product location shared inside a workspace | Caught |
| 37 | Lists in map order | Caught |
| 38 | Lists not bounded | Caught |
| 39 | The total of a bounded list | Caught |
| 40 | A deleted artifact still held | Caught |
| 41 | Held until a time in the past by a verification that ended | Caught |
| 42 | A lost instance still holds | Caught |
| 43 | A product copy does not hold | Caught |
| 44 | A verification of any length holds | Caught |
| 45 | A verification that ends exactly then does not hold | Caught |
| 46 | A lost copy counts as a copy | Caught |
| 47 | A remote instance counts as a copy | Caught |
| 48 | The cause lost while an instance remains | Caught |
| 49 | The reason of the first loss shown | Caught |
| 50 | An artifact with a malformed identifier | Caught |
| 51 | An artifact of an unknown kind | Caught |
| 52 | An artifact without a version | Caught |
| 53 | An artifact without a digest | Caught |
| 54 | An artifact of size zero | Caught |
| 55 | An artifact without a custody | Caught |
| 56 | An artifact registered again with other content | Caught |
| 57 | The evidence not part of what was registered | Caught |
| 58 | The verification time not part of what was registered | Caught |
| 59 | Registration at a taken product location | Caught |
| 60 | One artifact over the limit | Caught |
| 61 | The first product location not reserved | Caught |
| 62 | Registration time not recorded | Caught |
| 63 | Instances found by location alone | Caught |
| 64 | Custody added to a deleted artifact | Caught |
| 65 | A lost instance comes back | Caught |
| 66 | A product copy with another receipt accepted | Caught |
| 67 | A verification moved earlier | Caught |
| 68 | The same end with other evidence replaces the verification | Caught |
| 69 | The time of a new verification not recorded | Caught |
| 70 | The evidence of a new verification not stored | Caught |
| 71 | A copy added at a taken product location | Caught |
| 72 | Instances unbounded | Caught |
| 73 | One instance over the limit | Caught |
| 74 | An added product location not reserved | Caught |
| 75 | The time of an added custody not recorded | Caught |
| 76 | A loss of an unknown kind of custody | Caught |
| 77 | A loss with an unknown reason | Caught |
| 78 | A loss reported for a deleted artifact | Caught |
| 79 | A second loss with another reason | Caught |
| 80 | A verification withdrawn from a product copy | Caught |
| 81 | A reported loss ends nothing | Caught |
| 82 | Losses not ordered | Caught |
| 83 | The time of a loss not recorded | Caught |
| 84 | A loss that puts a bundle at risk not returned | Caught |
| 85 | A loss that breaks a bundle not returned | Caught |
| 86 | Deleted again with another reason | Caught |
| 87 | An artifact deleted while a bundle names it | Caught |
| 88 | Deletion time not recorded | Caught |
| 89 | Restricting a kind that is not run | Caught |
| 90 | A restriction without a version | Caught |
| 91 | A restriction with free text | Caught |
| 92 | Restricted again with another reason | Caught |
| 93 | Restricted versions unbounded | Caught |
| 94 | A restriction that ignores the kind when stored | Caught |
| 95 | Every artifact matched as a module | Caught |
| 96 | A hold does not protect | Caught |
| 97 | Retention does not protect | Caught |
| 98 | Protected at the retention time itself | Caught |
| 99 | A hold asks for nothing more than retention | Caught |
| 100 | Nothing at risk under retention | Caught |
| 101 | Broken items not counted | Caught |
| 102 | Items at risk not counted | Caught |
| 103 | Digest fields without their length | Caught |
| 104 | Items ordered by artifact alone | Caught |
| 105 | Items of a role in the order sent | Caught |
| 106 | A bundle without a customer | Caught |
| 107 | A bundle without a period | Caught |
| 108 | A bundle without a retention time | Caught |
| 109 | A bundle without items | Caught |
| 110 | An item of an unknown role | Caught |
| 111 | An item with a malformed artifact | Caught |
| 112 | Items per bundle unbounded | Caught |
| 113 | Item limit one too low | Caught |
| 114 | A bundle confirmed again with other content | Caught |
| 115 | The modules flag not part of what was confirmed | Caught |
| 116 | The retention time not part of what was confirmed | Caught |
| 117 | The role not part of what was confirmed | Caught |
| 118 | A deleted artifact reported as not held | Caught |
| 119 | An artifact of another kind accepted | Caught |
| 120 | An artifact that is not held accepted | Caught |
| 121 | Held now is enough to confirm | Caught |
| 122 | An artifact named twice accepted | Caught |
| 123 | Artifacts named twice in map order | Caught |
| 124 | Missing roles accepted | Caught |
| 125 | An FX item without the flag accepted | Caught |
| 126 | FX used without an item accepted | Caught |
| 127 | A module item without the flag accepted | Caught |
| 128 | Modules used without an item accepted | Caught |
| 129 | A retention time that is not ahead accepted | Caught |
| 130 | A retention time of now accepted | Caught |
| 131 | Reasons in another order | Caught |
| 132 | A refused confirmation without its reasons | Caught |
| 133 | Bundles unbounded | Caught |
| 134 | One bundle over the limit | Caught |
| 135 | The digest without its format tag | Caught |
| 136 | The digest without the workspace | Caught |
| 137 | The digest without the period | Caught |
| 138 | The digest without the modules flag | Caught |
| 139 | The digest without roles | Caught |
| 140 | The digest without artifact digests | Caught |
| 141 | Closed gaps bound at confirmation | Caught |
| 142 | Gaps of another customer bound | Caught |
| 143 | Gaps of another period bound | Caught |
| 144 | Bound gaps in map order | Caught |
| 145 | Confirmation time not recorded | Caught |
| 146 | A confirmed bundle does not keep its artifacts from deletion | Caught |
| 147 | An extension to no time | Caught |
| 148 | A purged bundle extended | Caught |
| 149 | Retention moved earlier | Caught |
| 150 | Extension time not recorded | Caught |
| 151 | An extension does not say what it put at risk | Caught |
| 152 | Purged again with another reason | Caught |
| 153 | A protected bundle purged | Caught |
| 154 | A purged bundle still keeps its artifacts from deletion | Caught |
| 155 | Purge time not recorded | Caught |
| 156 | A purged bundle keeps its items | Caught |
| 157 | Bundles at risk not listed as exposed | Caught |
| 158 | Broken bundles not listed as exposed | Caught |
| 159 | A hold with free text as its reason | Caught |
| 160 | A hold on nothing | Caught |
| 161 | A hold naming a bundle twice | Caught |
| 162 | Bundles per hold unbounded | Caught |
| 163 | Hold bundle limit one too low | Caught |
| 164 | A hold with its bundles in another order is another hold | Caught |
| 165 | A hold placed again with other content | Caught |
| 166 | A hold on a purged bundle | Caught |
| 167 | A refused hold without positions | Caught |
| 168 | Holds unbounded | Caught |
| 169 | One hold over the limit | Caught |
| 170 | A placed hold protects nothing | Caught |
| 171 | A hold does not say what is exposed | Caught |
| 172 | Released again with another reason | Caught |
| 173 | A released hold still protects | Caught |
| 174 | A gap of an unknown role | Caught |
| 175 | A gap without an owner | Caught |
| 176 | A gap without a path | Caught |
| 177 | A gap recorded again with other content | Caught |
| 178 | Gaps unbounded | Caught |
| 179 | A gap closed again with another artifact | Caught |
| 180 | A gap closed with an artifact of another kind | Caught |
| 181 | A gap closed with an artifact that is not held | Caught |
| 182 | Closed gaps listed as open | Caught |
| 183 | Open gaps of another customer listed | Caught |
| 184 | Open gaps of another period listed | Caught |
| 185 | A purged bundle read | Caught |
| 186 | A restricted module still run in operation | Caught |
| 187 | A restricted engine still run in operation | Caught |
| 188 | A bundle with an item that is not held called held | Caught |
| 189 | Broken items not counted in the verdict | Caught |
| 190 | Broken items listed without bound | Caught |
| 191 | Items at risk not counted in the verdict | Caught |
| 192 | Items at risk listed without bound | Caught |
| 193 | Bound gaps listed without bound | Caught |
| 194 | The artifact that closed a gap not shown | Caught |
| 195 | The total of bound gaps not shown | Caught |
| 196 | The gaps open now not shown | Caught |
| 197 | A download for an unknown role | Caught |
| 198 | A download under another role | Caught |
| 199 | A download of an item that is not held | Caught |
| 200 | A lost instance offered for download | Caught |
| 201 | An unverified remote instance offered for download | Caught |
| 202 | Download instances ordered by location alone | Caught |
| 203 | A page of nothing | Caught |
| 204 | A page over the limit | Caught |
| 205 | A negative position | Caught |
| 206 | A position beyond the end | Caught |
| 207 | Pages not cut | Caught |
| 208 | A page shares the bundle's items | Caught |
| 209 | The bundle view shares its gaps | Caught |
| 210 | The bundle view shares its counts | Caught |
| 211 | The registry's time kept in the clock's zone | Caught |

# Deviations from the plan

None after the plan was amended with the change review resolution. Where the rules of the design re-review are more precise than the sentences above them, the code follows the rules.

# Independent review

- Design: two read-only passes on 2026-11-13 (FAIL with 4 HIGH and no redesign asked; then PASS with findings, 1 HIGH among them, settled in the text). See the plan's two resolution sections.
- Change: one read-only pass on 2026-11-13, PASS with findings (2 MEDIUM, 9 LOW), nothing executed by the reviewer. All findings were resolved or stated. The corrections made after it were not reviewed again by a separate reviewer; they are covered by the tests and the second mutation run.

# Provisional choices awaiting the owner

Not the conservative choices, listed in the plan: `Extend` and `PlaceHold` succeed although evidence is then at risk; `Confirm` succeeds with open gaps and binds them. Assumptions: the seven roles every confirmed calculation must have, with several of each allowed; a decimal profile, a target schema and contract terms registered as `policy` artifacts. Undecided and not chosen here: the retention period, the responsibility for encryption keys, what makes a remote store verified and who may say so; whether a hold forces a copy; a hold by customer and period; how a mistaken bundle is corrected; which bundle of a customer and period is current; who may delete, purge, hold, release and restrict.

# Not verified

A real store, real sources, copying and comparing bytes, encryption, a process that dies, persistence, any operating system other than the one named, and everything listed as deferred in the plan. No deployment, payment, message, real account or customer data was involved.
