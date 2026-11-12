---
type: ImplementationPlan
title: Source documents received in versions, staged in chunks, and published as immutable input manifests
description: Defines the first model of work item W-06, source documents that belong to one subject and are received as versions whose artifacts are staged in repeatable chunks and sealed only when what was staged equals what they declare, with the latest sealed version replacing the earlier ones as a whole, and numbered input manifests published only when nothing in the subject is in conflict or unreadable.
status: draft
---

# Scope and ownership

[HB-186](https://linear.app/hybrid-billing/issue/HB-186/w-06-source-artifacts-staged-in-chunks-record-keys-with-repeats) is the first increment of work item W-06 (requirement HB-06, with the eligibility part of AR-03 and the exclusion rule of AR-05). It is the twelfth increment built under the owner's instruction of 2026-11-12. Base: main `1e7e33d`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the model decides which received amounts enter a confirmed calculation and is keyed by tenant. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-13:

- Requirement HB-06: an original is immutable per received version. The same key with the same content is a repeated receipt, the same key with other content is a conflict, a provider's correction is a new revision. Normal repeated use of an identical amount is not removed. Proposed direction: `SourceArtifact` and `SourceRecord` are separate; the receipt key is installation, provider, source scope, source event identifier and revision, and where an original row has no identifier, the file revision and a row locator are used; an upload is published from staging after checks of size, decompressed size, malicious content and syntax. Scale: file stream, immutable store, chunked staging, set-based duplicate and conflict checks, a completeness verdict, publication of an input manifest.
- Acceptance of HB-06: more than 100,000 inputs, interrupted and received again, keep their original row count, currency sums and hash, and conflicting data does not enter confirmed calculation. Linked source cases, all NOT_RUN: BILL01, COM01, FD03, DAT01 (a remote source modified, deleted or withdrawn).
- Open decisions of HB-06, not made: stable identifiers per source, the largest size and compression ratio of one file, and the preservation store.
- AR-03: an original and an OCR or parsing candidate are separate artifacts; extraction method, verifier and verification evidence are recorded; an unverified extracted value is not in the input manifest of a confirmed calculation; a file in which OCR misread a decimal point or a currency is blocked from confirmation although its upload succeeded. AUD-AR-03 is NOT_RUN.
- AR-05: prompt and response bodies are excluded from billing source fields; an input that carries them is not accepted as a normal billing artifact and the permitted fields are obtained again; staging and error logs follow the same rule. AUD-AR-05 is NOT_RUN.
- Mandatory detail contracts of W-06 that concern other things than this increment: AR-02 (connection authentication and push verification), DB-06 (documents that execute or reach outside), RT-01 (source event envelope, in which a correction is its own event that names the one it supersedes).
- Existing code: `modules/connectors` (HB-184) says whether a collection received everything and which collection counts, and holds no rows. The Python modules under `modules/focus_import` hold a working local import path with its own receipts. `packages/primitives/decimal` gives exact decimal arithmetic under a profile and refuses an intermediate result outside the profile. Everything is in memory or local files; the business database and the preservation store are not selected.
- The planning sources are untracked local files of the primary checkout and were read there.

# Decision

One Go package, `modules/intake`, standard library and the decimal primitive only, with an in-memory registry. It records what a caller says an artifact contains and decides what may enter a calculation; it stores no bytes of an original, scans and parses nothing, calls no provider and has no entry point wired in this increment.

What this can and cannot show: the registry checks that what was staged is complete, internally consistent and equal to what the artifact declares, that a later version replaces an earlier one as a whole, and that nothing in conflict is published. It cannot know that a row was read correctly: rows, digests, sizes, check results and declared totals all come from the caller. Two rows whose errors cancel, 150 and 150 for 100 and 200, pass every check here when the declared totals come from the same reading; totals stated by the source itself, and for extracted values the verification of AR-03, are what stands against that (stated limit).

Scope of the model: sources that deliver documents, each a complete statement of its subject that a later version restates in full. A source that delivers single events, each correction its own event (RT-01), is not modelled; an adapter would have to fold such events into document versions, and that work is deferred with RT-01 (stated).

The Python import path is not changed and not connected; nothing may read both for one period (stated).

## Subjects and documents

The subject of a document, and of a manifest, is (workspace, provider, source scope, period), all identifiers. A period is an identifier that the caller chooses consistently, such as a billing month.

A document is one original as the source knows it, across all its versions. Its identifier is unique in the workspace, and the first artifact registered for it, whatever becomes of that artifact, fixes its subject and its row key kind; a later artifact that names the document with another provider, scope, period or kind is refused as a conflict. A registration that is refused writes nothing and fixes nothing; one that is recorded as rejected does fix them. A document registered under the wrong subject is left by retiring what was sealed and receiving the original under a new document identifier. The installation of the requirement's key is the registry; the workspace is the tenant inside it. Rows carry no period of their own, so an amount cannot be in two periods: a file that covers several periods is split by the caller into one document per period, each with its own identifier (stated obligation).

A document has a row key kind, fixed by its first artifact: `event`, when the source gives each row a stable identifier, or `locator`, when a row is known only by its position. An event key means the same row wherever it appears in the subject; a locator means something only inside its own document. Rows of an event document and of a locator document are never compared, so a subject that receives the same rows in both kinds counts them twice (stated limit).

## Artifacts

An artifact is one received version of a document, or one candidate extracted from such a version. It is registered once, with a caller-assigned identifier unique in the workspace, and never changes:

- its document, its source revision (a number from the source, at least 1, higher for a later state), and a digest of its bytes;
- its role: `original`, with what it holds: `rows`, `empty` (the source states nothing for this version) or `awaiting_extraction` (a scan whose content will come through a derived artifact); or `derived`, which names the original artifact it was extracted from and the extraction method, an identifier;
- what it declares about its content: the number of chunks, the number of rows, the sum of amounts for every currency that occurs, and where those totals come from (`source_stated` or `caller_computed`);
- its size in bytes and its expanded size, an opaque reference to where its bytes are kept, an optional reference to the collection it came from, and the results of three checks made on its bytes: the scan for malicious content, the syntax check, and the search for prompt or response bodies, each `passed`, `failed` or `not_run`.

Registering an identifier that exists in the workspace succeeds without writing when everything is identical and is a conflict otherwise; the same identifier in another workspace is unrelated.

Refused at registration with nothing written, because the artifact could never be sealed: a size that is not positive, an expanded size below it, more chunks than the limit, more rows than chunks times the records allowed in one, fewer rows than chunks, an original or derived artifact with rows that declares no row, no chunk or no sum, more currencies than rows, an `empty` or `awaiting_extraction` original that declares a chunk, a row or a sum, a sum that the profile does not accept as an input (its input bound, not its result bound: an artifact whose true sum is longer must be split by the caller), a row artifact that does not say where its totals come from, an artifact without a reference to its bytes.

Registered as `rejected`, with every reason that applies and accepting nothing afterwards: larger than the size limit; expanded size over the ratio limit times its size; a check that `failed`; a check that was `not_run`. A rejected artifact is kept as a record that something arrived, and like every artifact that ended without being sealed, or was retired, it is received again only under a new identifier; when the reason is that bodies were found, the caller must not keep the bytes as a collected original (AR-05; the registry holds none). A check made later needs a new artifact.

An `empty` original and an `awaiting_extraction` original have no chunks; `Seal` seals them as they are. Both still need bytes, a digest and passed checks, so a provider that withdraws a document without delivering anything is represented by a notice that the adapter supplies as the bytes of an `empty` version (stated obligation). They mean different things: the first is content, namely none, which is how a document that the provider voided or emptied is received; the second is not content yet. A `derived` artifact has rows and names an `awaiting_extraction` original of the same document and revision that is `staging` or `sealed`; no other original can have one, so an original and its extraction never both claim the same content. Rows of a derived artifact of a locator document carry the locator in the original; for an event document nothing here records where in the original a value was read (AR-03 asks for it; stated gap).

States: `staging`; and, final: `sealed`, `rejected`, `inconsistent`, `abandoned`, and `retired`, the last only from `sealed`.

## Records

A record is one row as the caller read it: its row key, an event identifier or a locator according to the document's kind; a digest of the row's content; and its amount with a currency. No free text: identifiers, a digest, a decimal and a currency code. Prompt and response bodies therefore cannot be staged, and an artifact found to contain them is rejected at registration.

Two rows with the same amount, or the same whole content, under different row keys are two records. Nothing here removes a row because it equals another.

## Staging in chunks that can be sent again

`Stage(artifact, number, records)` stages one chunk of a `staging` artifact. The number is from 1 to the declared count and the chunk holds at least one record and at most the limit; anything else is refused with nothing written.

- A number that was staged with the same records in the same order writes nothing and succeeds, so an interrupted transfer is continued by sending chunks again from anywhere, in any order.
- A number that was staged with other records makes the artifact `inconsistent`, finally. So does a row key that appears twice in the artifact, in one chunk or in two.
- Staged records are visible to nothing outside the artifact until it is sealed.

Sums are kept as exact integers while staging, each currency at the largest scale among its amounts, and only the final sum of each currency is checked against the profile, so whether an artifact can be sealed never depends on the order in which its chunks arrived.

`Abandon(artifact, reason)` ends a `staging` artifact, with a reason from a closed list (`transfer_failed`, `superseded`, `unsealable`, `operator`); it is never refused for room. It is the exit for an artifact that cannot be sealed.

## Sealing

`Seal(artifact)` makes a `staging` artifact `sealed` when all of these hold, and is refused otherwise with every reason listed and nothing written:

- every declared chunk was staged and the number of staged rows equals the declared number;
- the currencies of the staged rows are exactly the declared ones and each exact sum equals the declared sum, a declared sum of zero included;
- for a `derived` artifact: its original is `sealed`, and a verification is recorded whose digest equals the digest of what is staged (below).

A sealed artifact stores its row count, its sums and its record digest: SHA-256 over its records sorted by row key, each as four fields, the row key, the amount in the primitive's canonical text, the currency and the row digest, every field preceded by its length as eight bytes, big-endian. Keys are compared bytewise. An artifact without rows has the digest of no records. The same content staged in any chunking and any order of arrival has the same digest. Together with the byte digest given at registration, this is the acceptance's "row count, currency sums and hash preserved" at the level of this package. Sealing gives the artifact the next number of the sealing sequence of its workspace, which, not time, orders artifacts; the sequence is per workspace so that one tenant cannot count another's seals.

`Verify(artifact, digest, verifier, evidence)` is accepted for a `derived` artifact that is `staging` with all its chunks staged, when `digest` is the record digest of what is staged, which `Staged(artifact)` reports together with the rows and sums staged so far: the verifier attests exactly that content. Again with the same values it writes nothing; with others it is a conflict. Because a staged chunk cannot change, what was verified is what is sealed. An extraction in which a decimal point was misread is caught only if the verifier compared the rows with the original; the registry records who says so and for which content, and makes it impossible to seal anything else (AR-03 at the level of this package).

`Retire(artifact, reason)` makes a `sealed` artifact `retired`, with a reason from a closed list (`wrong_source`, `reissued`, `verification_withdrawn`, `operator`); it is never refused for room. `Seal` of a sealed or retired artifact writes nothing and succeeds.

## What counts: the latest live version of each document

An artifact is live when it is `sealed` and, if it is derived, its original is `sealed` too: what was extracted from an original that was retired counts for nothing, though its records stay counted against the limit until it is retired itself. A live artifact carries content when it is an original that holds `rows` or is `empty`, or a derived artifact.

For each document, only the highest revision that has a live artifact counts, and it replaces every lower revision as a whole: a row that the later version no longer contains is gone, a row that moved is where the later version says, and nothing of an earlier version is mixed in. A provider's correction, and a provider's deletion, are therefore the same thing here: a new version of the document. Consequence, stated: when the artifacts of the highest revision are retired, the next lower live revision counts again.

The revision that counts is the highest that has any live artifact. A document without a live artifact has the counting revision 0, contributes nothing and blocks nothing. Otherwise, at the revision that counts, exactly one of these holds, tested in this order:

- unreadable: no live artifact of that revision carries content, a scan whose extraction was not sealed. The earlier version it replaced does not come back to fill the gap;
- in conflict: the live artifacts that carry content do not all have the same record digest, two receipts of one version that disagree, an empty statement beside rows, or two extractions that disagree;
- readable: otherwise. The earliest sealed of the artifacts that carry content is the one the document's rows are attributed to; the others are repeated receipts and add nothing.

The registry also knows, for each document, the highest revision anyone ever registered for it, in whatever state. A document whose counting revision is below that is behind: a later version arrived and was rejected, abandoned, found inconsistent, retired, or is still being staged, and what counts is the version before it, or nothing at all when no artifact of the document is live. A revision registered too high by mistake leaves its document behind for good, which dulls the sign; the exit is a new document identifier (stated limit). Being behind does not block; it is reported wherever the subject is shown (below). Likewise reported and not blocking: two live artifacts of different documents of the subject with the same byte digest, which is what the same file registered under two document identifiers looks like, and is seen only inside one subject, not when the two documents name different periods; for locator documents the document identifier is the only defence against counting such a file twice (stated limit).

Across the documents of one subject, an event key that occurs in two readable documents is one row when amount, currency and row digest are equal, counted once and attributed to the earlier sealed artifact, and a conflict when they differ. Locators are never compared across documents.

## Input manifests

`Publish(subject, expected)` creates the next manifest of a subject from what counts at that moment. The checks run in this order:

1. a subject with no document is not found;
2. it is refused while any document of the subject is in conflict or unreadable, any event key is in conflict, or the sum of what counts in one currency is outside the profile's result bound, listing them up to a bound with the total: conflicting data never enters, and is not silently left out either, because leaving a row out changes the total as much as a wrong row does;
3. when content and attribution equal the latest manifest's (judged by the attribution digest, which covers every content field), nothing is written and that manifest's number is returned, whatever `expected` says, so a repeat after a lost response is answered;
4. `expected` must be the number of the latest manifest, 0 for none, else it is a conflict;
5. a first manifest without records is refused, so that nothing is published about a subject in which nothing counts yet (an empty manifest after an empty one is already answered by check 3);
6. the manifest limit of the subject is checked.

The key of a record in a manifest is (key kind, the document identifier for a locator row and nothing for an event row, row key), so a locator of one document, the same locator of another and an event key of the same text are three records. A manifest is immutable and holds: its number; the number of records and the sum per currency; a content digest, SHA-256 over its records sorted by that key, each as the three key fields (the kind as the text `event` or `locator`, the document identifier or a field of length zero, the row key), the canonical amount, the currency and the row digest, every field preceded by its length as eight bytes, big-endian, with records ordered by comparing the three key fields bytewise in turn; an attribution digest, the same with the identifier of the artifact each record rests on added; the number of artifacts of the subject that were still `staging`, the documents that were behind and the pairs with equal bytes when it was published, each as a total count and a list up to a bound; the records, each with key, content and the artifact it rests on; and the time. Those three annotations are a picture of the moment of publication: they are in neither digest, a change in them alone publishes nothing, and what is true now is read from `Status`, never from an old manifest. A change of attribution alone, such as the retirement of the earlier of two equal receipts, publishes a new manifest with the same content digest. A manifest may be empty when its predecessor was not, so that a removal is published.

`Status(subject)` says, without writing: the latest manifest's number; whether what counts now differs from it, in content or only in attribution, or `blocked` when a blocker makes the comparison meaningless; the blockers of check 2, up to the bound with their total; the number of artifacts still staging; and the documents that are behind and the pairs with equal bytes. It does not say that a publication would be refused as an empty first manifest or at the manifest limit (stated limit). A consumer that holds manifest N reads this to learn that N is no longer current. `Manifest(subject, number)` returns a manifest's summary and `Records(subject, number, after, limit)` its records in key order, in pages of at most a limit given at construction, `after` being the number of records already read.

A manifest says nothing about whether everything that should have been received was received: it is made from the sealed artifacts that exist, it records how many were still being staged, and nothing ties it to the collection that `modules/connectors` calls effective. In particular a row that a correction moved to another period is in both subjects' manifests from the moment the new version of the one document is sealed until the new version of the other is (stated limit).

## Repeats

No write identifiers are used: every operation is its own key. A repeat with the same content writes nothing and succeeds in every state the artifact has since reached: `Register`, `Stage` of a staged chunk, `Verify`, `Seal`, `Abandon` and `Retire` with the same reason. The same operation with other content is a conflict or, for a chunk or a row key, makes a staging artifact `inconsistent`; that operation returns the state `inconsistent` without an error, because it was applied, the artifact records which chunk contradicted and when, and every later operation on it other than a repeat is refused. A caller must read the returned state. `Publish` is guarded as described. The registry keeps one digest per staged chunk also after the rows of a dead artifact were dropped, so a repeated chunk is still recognised; the chunk that caused the contradiction was never staged and is refused when sent again. No error and no record of a contradiction repeats a string of the refused request; the blockers of a publication name stored document identifiers, row keys and currencies of the workspace itself.

## Tenant boundary, time and limits

Every operation takes the workspace and addresses only that workspace's artifacts, documents and manifests; those of another workspace are not found, in the same way as ones that do not exist, and the same keys with other content in two workspaces never meet. The registry authorizes nothing and must not be exposed directly (stated limit). Every caller-supplied string is an identifier under the project's identifier rule, a lowercase hexadecimal digest of 8 to 128 characters, a three-letter uppercase currency code, or a decimal that the profile given at construction accepts; the package has its own copy of the identifier rule (stated).

The registry's clock is injected, read once per operation, and never goes backwards. An artifact carries the time of its registration and of its last change of state; a verification and a manifest carry their time.

Required and positive at construction, without defaults: the largest size of an artifact; the largest ratio of expanded to stored size; chunks per artifact; records per chunk; artifacts per workspace; records per workspace; manifests per subject; the page size; the bound of the lists of blockers, behind documents and pairs. Records are counted when staged, for artifacts that are `staging` or `sealed`; an artifact that becomes `abandoned`, `inconsistent` or `retired` gives its records back, and its staged rows are dropped, since nothing can use them. So `Seal` never needs room, and retiring or abandoning frees it. Artifacts are counted for good, rejected ones included, and manifests are copies that are counted only by their number: a source that keeps registering artifacts that are rejected stops intake for its workspace, and a subject republished to its limit, changes of attribution included, can publish no further manifest; neither has an exit here (stated limit, an exposure blocker; retention and the preservation store are the owner's open decisions).

## What is not decided here

The stable identifier of each source and whether its rows have event keys; the size and ratio limits; the preservation store; whether a conflict or an unreadable document blocks its whole subject; whether a document that is behind, or two documents with the same bytes, must block as well (they are only reported, which is not the conservative choice); what counts as verification of an extraction and who may give it; who may retire and publish.

# Allowed files

`modules/intake/*` (new); `docs/features/HB-186/*`.

# Verification

- The acceptance at package level: 120,000 records with several currencies, negative amounts and one currency that sums to zero, in 120 chunks, staged straight through in one registry and, in another, with the transfer broken off, resumed from an earlier chunk, chunks out of order and repeated, give the same row count, sums and record digest, a published manifest with the same content digest, and totals equal to those of the generator; the digest is also computed in the test by a second writing of the stated encoding. Stated with it: two in-memory registries show independence from the order and repetition of calls, not a process that died.
- A ground truth that does not share the model: the test generates what a provider knows, documents with their final rows after corrections, deletions, re-exports in which rows shift, rows moved to another period and documents that end empty, derives the versions an adapter would deliver, seals them in a random order, and asserts that once every version is sealed the latest manifests of all subjects together hold exactly the provider's final rows, compared as a multiset of subject, amount, currency and row digest.
- Literal table tests with a test clock: every refusal and every reason for rejection at registration, several at once; a document's subject and key kind fixed by its first artifact; each way a staging artifact becomes inconsistent; each reason `Seal` is refused, listed; chunk numbers out of range, an empty chunk, a chunk over the limit; sums whose partial sums leave the profile in one order of arrival and not in another; rows with equal content under different keys both kept; a later version dropping and shifting rows; the lower version counting again after a retirement; two receipts of one version, equal and different; a scanned original without rows, unreadable until its extraction is verified and sealed; verification before all chunks are staged, with another digest, and of other content afterwards; two extractions that disagree; an extraction not counting once its original is retired; an event key in two documents, equal and different; locators not compared across documents, and a locator and an event key of the same text as separate records; a later version that is rejected, abandoned, inconsistent, retired or still staging reported as behind; the same bytes under two documents reported; a document none of whose artifacts is live reported as behind, both when its only artifact was rejected and when its sealed version was retired while the next one failed; a later version rejected after a manifest was published, seen in `Status` and not in that manifest; a mistaken high revision; more documents behind than the bound, with the total; mixed artifacts at one revision in each of the three conditions; a document without a live artifact; an empty first manifest refused; an emptied document received and published; the manifest limit; a repeated chunk after the rows were dropped; publication in each of its six checks, with an old expected number, after a lost response, with unchanged content, with a change of attribution only, and of an empty successor; `Status` after each kind of change; paging.
- Repeats of every operation after every state; refusals writing nothing and echoing nothing; the tenant boundary against a workspace that exists, with the same keys and other content in two workspaces; limits, and records given back.
- Properties with a fixed seed over random histories of raw attempts against a reference model written in the test that decides acceptance itself: the same attempts are accepted; every manifest equals the model's; sealing the same artifacts in another order gives the same manifest content; no manifest holds a row of an artifact that was not live, two rows of one key, or anything of a subject that had a blocker; no manifest changes after it was published.
- The package's tests are run many times in a row before any mutation run.
- Concurrency under the race detector: concurrent staging of the same chunks, each counted once, concurrent publication with one expected number, of which one is written, and retirement and sealing racing publication.
- Mutation check by a script in Python with per-row results appended to files that are never overwritten.
- `gofmt -l`, `go vet ./...`, `go test -race -count=1 ./...`.

Remaining NOT_RUN at system level after this ticket: BILL01, COM01, FD03, DAT01, AUD-AR-02, AUD-AR-03, AUD-AR-05, AUD-DB-06, AUD-RT-01. This ticket does not complete W-06, and the acceptance of HB-06 itself, which is about real files, is NOT_RUN.

# Deferred

Sources that deliver single events with corrections of their own (RT-01); storing, preserving and scanning the bytes; parsing; AR-02; DB-06; a checked link to collections of `modules/connectors`; publishing without the rows in conflict; joining the Python import path; commands, permissions and screens S03, S05; retention; persistence.

# Design review resolution

The independent design review of 2026-11-13 (read-only, nothing executed) returned FAIL: three blockers and three HIGH structural holes, and eleven further findings. The first draft keyed every row by event and revision and let the highest revision of each event count; amounts were then counted twice when a correction moved an event to another period, and miscounted when a file without row identifiers was exported again with rows shifted or removed. The model was replaced, not patched.

| Finding | Resolution |
| --- | --- |
| S1 an event in two periods | A document belongs to one subject and rows carry no period |
| S2 locator keys under per-event revisions; no identity of the original | Documents with an identifier; a later version replaces an earlier one as a whole; locators are local to their document |
| S3 deletion by the provider | A new version without the row; the return of a lower version after a retirement stated and tested |
| S4 an original and its extraction claiming the same content | Only an original that awaits extraction can have a derived artifact |
| S5 no bond to the bytes; self-referential checks; verification not bound to content | A byte digest at registration; the origin of the totals recorded and the limit stated; `Verify` attests the digest of what is staged |
| S6 an extraction living on after its original | Not live unless its original is sealed |
| D1 sums depending on the order of arrival | Exact integer sums; only the final sum is checked |
| D2 digests and manifest content | Inputs stated; content and attribution digests; a sealing sequence |
| D3 setting a claim aside | Removed: a conflict is resolved by a new version or a retirement |
| D4 stale manifests | `Status` |
| D5 publication | The order of checks; no first manifest of nothing; empty successors; a manifest limit |
| D6 completeness | Stated; the number of artifacts still staging recorded |
| D7 registration | Every reason kept; `not_run` for all three checks; the purge obligation; originals checked; unsealable declarations refused |
| D8 limits without an exit | Records counted while they can be used and given back |
| D9 chunk edge cases | Stated |
| D10 revisions and RT-01 | The revision is the document's; event sources deferred |
| D11 the workspace in the key; echoes | Stated and tested |

# Design re-review resolution

The narrow re-review of 2026-11-13 (read-only, nothing executed) returned FAIL on two points and said the replaced model is sound, closes the double counts of the first draft, and needs amendment of paragraphs, not another redesign: thirteen of the seventeen first findings resolved and four partly; nine new findings, 2 HIGH, 5 MEDIUM and 2 LOW. It judged a model of whole-version replacement a legitimate reading of the requirement for file and document sources. All nine were adopted above.

| Finding | Resolution |
| --- | --- |
| E1 an empty version could not be received | An original says whether it holds rows, is empty or awaits extraction; an empty one is content |
| E2 a superseded version published without a sign | The highest revision ever registered is kept per document; a document that is behind is reported in `Status` and in every manifest; whether it blocks is listed as undecided |
| E3 the three conditions not exclusive | "Carries content" defined; the conditions tested in a stated order |
| E4 the identity of a document | Unique in the workspace; fixed by its first artifact |
| E5 manifest keys and digests | The key and both encodings stated, every field length-prefixed |
| E6 one file under two document identifiers | Reported by equal byte digests; the limit stated; listed as undecided |
| E7 a subject in which nothing counts | A document without a live artifact contributes and blocks nothing; an empty first manifest refused |
| E8 repeats after rows were dropped | One digest per staged chunk is kept |
| E9 wording and small gaps | The manifest limit, records of a dead extraction, `Staged`, sealing without rows, receiving again under a new identifier and the missing locator of AR-03 stated |

# Design third review resolution

A third pass, limited to the amended paragraphs, returned PASS with findings on 2026-11-13: implementation may start once two one-sentence points were settled in the text; no blocker or HIGH, 3 MEDIUM and 6 LOW; all nine second-round findings resolved. It found no new double count inside the model, one silent drop (a document with no live artifact was not reported as behind) and one weakening of the report (annotations that a published manifest never receives). All nine were adopted above: a document without a live artifact has the counting revision 0 and is behind; the annotations of a manifest are a picture of its moment and `Status` is the current source; the consequence of a mistaken high revision, the exits for a wrong first registration and for a withdrawal without bytes, the reach of the equal-bytes report, totals beside bounded lists and the byte-level encoding are stated. The text after this pass was not reviewed again as a design; the change review reads it against the code.

# Change review resolution

The independent change review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker or HIGH, 1 MEDIUM and 6 LOW. It found no path by which an amount is counted twice or dropped, or by which a conflicting, unverified or non-live row reaches a manifest, beyond the stated limits, and confirmed by reading the arithmetic, the digest encodings, the record accounting, the tenant boundary and the absence of shared state in views.

| Finding | Resolution |
| --- | --- |
| 1 MEDIUM a subject's sum may leave the profile although every artifact's sum is inside it; declared sums are bound by the input bound | A subject sum outside the result bound blocks publication (`sum_outside_profile`, per currency); the registration paragraph says input bound |
| 2 LOW the registry-wide sealing sequence shows one tenant the seals of another | The sequence is per workspace |
| 3 LOW simplifications and unstated behaviour | Stated above (checks 3 and 5, paging, the list bound, `blocked`) and below |
| 4 LOW a key conflict names only the later document | The blocker also names the document that holds the key; one blocker per key, the later document being the first in sealing order that disagrees |
| 5 LOW `Status` does not announce two refusals | Stated as a limit |
| 6 LOW one lock for all workspaces, digests recomputed on every read | Added to the exposure blockers below |
| 7 LOW tests | An assertion that could not fail replaced; repeats of `Register` after `abandoned` and `inconsistent` asserted; a test that errors repeat nothing of the request added; the limits of the reference model and of the ground-truth test recorded in the verification record |

Stated with this review, as the code behaves:

- Error kinds: a malformed request is invalid, including a chunk number above the declared count and rows over the capacity of the declared chunks; more chunks than the limit, and no room for artifacts, records or manifests, are limit errors; a derived artifact that names a missing original is not found, one that names an unsuitable original is rejected. A publication refused as an empty first manifest and one refused for blockers are both rejected and differ by the count of blockers.
- An equal amount written otherwise (`1.50`, `1.5`) is the same content everywhere: registration, a repeated chunk, digests.
- `Stage` decides a contradiction before it looks at the record limit, so whether an artifact becomes inconsistent never depends on room.
- `Seal` requires that a verification exists; it need not compare digests again, because verification is accepted only when every chunk is staged and a staged chunk cannot change.
- `Staged` is refused once the rows were dropped. `Verify` does not change the artifact's time of last change. The view of an abandoned, inconsistent or retired artifact keeps the row count and sums it had.
- Exposure blockers added: one lock serves every workspace, and `Publish`, `Status` and `Staged` sort and hash all rows concerned while holding it, so a large subject of one tenant delays all tenants.
