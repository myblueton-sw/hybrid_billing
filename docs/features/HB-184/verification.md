---
type: VerificationRecord
title: Connector capabilities and collection ledger verification
description: Records what was executed and reviewed for the first model of work item W-05, with results, a test fault found on the way, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-184](https://linear.app/hybrid-billing/issue/HB-184/w-05-verified-connector-capabilities-and-a-collection-ledger-that) adds the Go package `modules/connectors` from main `283118c`: capability declarations per adapter version, operation, data kind and condition, of which only a verified one lets anything proceed; and a collection ledger per workspace in which a file collection is checked against a manifest and a paged one against a cursor chain, a contradiction about something already accepted ends a collection as `inconsistent`, the highest known source revision of a subject supersedes whatever runs below it, a collection that cannot finish is abandoned, and what counts for a subject changes only by its first completion, an explicit adoption or a retirement. See the [plan](plan.md).

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan twice before implementation and the implemented change afterwards (see Independent review). Machine: mac_mini.

Nothing calls the package: no adapter exists, no provider is called and no data is stored. The ledger checks that what a caller reports is complete and consistent; it cannot know that a report is true. It is therefore a precondition for the acceptance of HB-05, which injects faults into real collection and is NOT_RUN, as are BILL01, COM01, PO07, FD03, AUD-AR-02, AUD-DB-02, AUD-QA-02, AUD-QA-07 and AUD-RT-01. W-05 is not complete.

# Executed checks

Executed on 2026-11-13 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain. No other operating system was tested.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All fourteen packages `ok`, including `modules/connectors`; run again after the last correction |
| `go test -count=60 ./modules/connectors/` | `ok`: sixty consecutive runs without a failure, after the fault described below was fixed |

The injected faults of the acceptance, as the ledger sees them (`registry_test.go`, with a test clock):

| Fault | What the test shows |
| --- | --- |
| One split file missing | `Complete` refused, naming each missing part |
| One split file with a wrong byte count, row count or checksum | The part stays unverified with the count of mismatches and the last values kept; `Complete` refused naming it; a correct receipt afterwards verifies it |
| A mismatch after the part was verified | The collection ends `inconsistent` with the part recorded; the delayed correct report afterwards changes nothing |
| A repeated page | Counted once, also after the chain moved on |
| A page repeated with other content, a loop in three forms, the same content under a new cursor, totals that the chain contradicts | The collection ends `inconsistent` with the cause recorded |
| A 429 | A counter and the last wait; the state unchanged |
| A withdrawn permission | The collection is interrupted and can complete only after a resumption under an unchanged capability; a delayed copy of the loss report does not interrupt again, and a delayed resumption does not clear a newer interruption |
| A changed source | Every running collection of the subject below the new revision is superseded, whichever collection the news came through and whether it was interrupted or final; a start below the known revision is refused |

Further literal tests: every answer of the capability gate, with a neighbouring contract, region and source version unknown; nineteen malformed declarations; a return to earlier content not returning to its revision; a withdrawal at the capability limit; thirteen malformed starts, each refusal of a start, a connection under a second provider and the same connection identifier in another workspace; the manifest in another order, with another origin, other values or more parts than allowed, empty and derived by the adapter; operations of the other shape; declared totals met and not met; empty pages and pages sharing a checksum with an empty one; the page limit and each reason to abandon; contradictions arriving while a collection is interrupted, for a part, a manifest, a total and a page; the capability declared again, also back to its first content, blocking `Complete` and `Resume`; what counts through three completions of one revision, an empty higher revision that waits, adoption by pointer number with a delayed adoption refused, a mistaken adoption undone by retirement and adoption of the older data, staleness, every standing, and the capability's present answer beside the stored one; costs, invoices, a pro forma and a distributor's sales data as separate subjects; chains and manifests that differ only in a checksum standing as conflicting; the other workspace seeing nothing, tested against a workspace that exists; the registry's time after the clock went back; a registry that is not built.

Property test (`property_test.go`, fixed seed): 400 histories of 140 raw attempts over starts, manifests, parts, pages, totals, losses, resumptions, changed sources, abandonments, completions, adoptions, retirements, new declarations of a capability and throttling, half of the steps steered so that collections do finish. A reference model written in the test decides from each attempt whether it is accepted; after every step the registry's acceptance, the state of every collection, the rows of a completed one and the standings of every subject equal the model's. From the registry's own reports: no collection leaves a final state except a completed one that is retired; no running collection is below its subject's known revision; what counts for a subject moves only in a step in which a completion, an adoption or a retirement was accepted; the standings are the same on a second read. Reached: 1,023 starts, 522 completions (277 of files, 245 of pages), 81 collections ended inconsistent, 13 of them while interrupted, 57 superseded, 46 abandoned, 355 retired; the pointer moved 478 times by a completion, 32 by an adoption and 342 by a retirement.

Concurrency under the race detector: eight goroutines reporting the same forty parts and forty pages, each counted once.

Limits of these tests: the model is the same rules written a second time by the same author and shares types and test helpers with the package, so a shared misreading is invisible to it, as the change review showed for one rule (contradictions while interrupted, since corrected in both); the property looks at no error kind, no gap list, no byte totals and one workspace; the pointer oracle excuses a move in any subject in a step with an accepted completion, adoption or retirement; expectations in the tables were worked out by hand.

# A fault in the tests, found by the mutation runs

One literal test iterated over a map of three mismatching receipts and asserted the last values kept, which depends on the order of iteration. It failed in 8 of 40 runs for no fault of the code. It was noticed because two mutation rows changed their result between runs. Until it was fixed, any mutation row could have been recorded as caught because that test happened to fail, so the results of the first three runs are not used. The test now walks an ordered list; sixty consecutive runs passed; the whole mutation list was then run again and only that run is reported below. The four packages merged before this one (`contracts`, `jobs`, `onboarding`, `commands`) were each run 25 times without a failure.

# Mutation check

125 single-line mutations of `modules/connectors` were applied one at a time by a Python script, and the package's tests were run after each. The per-row results were appended to files that were never overwritten.

History:

- The list had 116 rows before the change review and 125 after it. Three runs were made while the order-dependent test described above was still in the suite; their results are not used, though they did their work: they showed tests that did not test what they named (chains and manifests differing only in a checksum, empty pages sharing a checksum with a page that had rows, the registry's time after the clock went back, an unknown operation without a data kind), which were added, and three conditions that could not change a result, which were removed.
- Fourth run, of all 125 rows, with the corrected tests.
- Result: 121 of 125 caught, 4 not caught.

The 4 that are not caught are checks kept as defence; the change review confirmed by reading that nothing reaches three of them, and the fourth was added at its request:

| Mutation | Why no test distinguishes it |
| --- | --- |
| 28, a capability's answer not checked beside its revision | Any declaration with other content raises the revision, so an equal revision already means the same, verified, content |
| 53, a part reported on a paged collection | A paged collection has no manifest and so no part to match; the refusal is the same |
| 66, a page reported on a file collection | A file collection waits for no cursor, so every page is refused for that reason |
| 123, the gap for totals that differ in `Complete` | A chain that contradicts its declared totals has already ended the collection as `inconsistent` |

| No. | Fault introduced | Result |
| --- | --- | --- |
| 1 | A short checksum | Caught |
| 2 | Checksum length unbounded | Caught |
| 3 | A checksum in capitals | Caught |
| 4 | Any pairing of operation and kind | Caught |
| 5 | An unknown operation | Caught |
| 6 | A condition without a source version | Caught |
| 7 | A condition without a contract | Caught |
| 8 | Verified without evidence | Caught |
| 9 | Verified with a reason | Caught |
| 10 | Limited without evidence | Caught |
| 11 | Limited without a reason | Caught |
| 12 | Unsupported with evidence | Caught |
| 13 | An alternative that is text | Caught |
| 14 | A subject of an operation that fetches nothing | Caught |
| 15 | A subject without a data kind | Caught |
| 16 | A source revision of zero | Caught |
| 17 | A cursor on a file collection | Caught |
| 18 | Pages without a start cursor | Caught |
| 19 | An interrupted collection does not count as running | Caught |
| 20 | Byte totals may overflow | Caught |
| 21 | Row totals may overflow | Caught |
| 22 | The registry's time goes backwards | Caught |
| 23 | Capabilities unbounded | Caught |
| 24 | A new declaration keeps the revision | Caught |
| 25 | An identical declaration raises the revision | Caught |
| 26 | A limited capability allowed | Caught |
| 27 | A capability declared again still permits | Caught |
| 28 | The answer not checked | Not caught |
| 29 | The known revision can go down | Caught |
| 30 | An interrupted collection survives a changed source | Caught |
| 31 | A completed collection superseded | Caught |
| 32 | Siblings not superseded | Caught |
| 33 | A start with other content accepted as a repeat | Caught |
| 34 | A start without a verified capability | Caught |
| 35 | A connection under two providers | Caught |
| 36 | A start below the known revision | Caught |
| 37 | Collections unbounded | Caught |
| 38 | A start does not raise the known revision | Caught |
| 39 | Collections found across workspaces | Caught |
| 40 | A negative byte count | Caught |
| 41 | Byte counts unbounded | Caught |
| 42 | Row counts unbounded | Caught |
| 43 | Checksums of parts unchecked | Caught |
| 44 | Any manifest origin | Caught |
| 45 | A part twice in a manifest | Caught |
| 46 | A manifest on a paged collection | Caught |
| 47 | Another origin taken for the same manifest | Caught |
| 48 | A shorter manifest taken for the same | Caught |
| 49 | Other values taken for the same manifest | Caught |
| 50 | Another manifest makes a completed collection inconsistent | Caught |
| 51 | A changed manifest refused and forgotten | Caught |
| 52 | Parts per manifest unbounded | Caught |
| 53 | A part on a paged collection | Not caught |
| 54 | Parts received after a final state | Caught |
| 55 | A mismatch after verification only unverifies the part | Caught |
| 56 | Parts never verified | Caught |
| 57 | Mismatches not counted | Caught |
| 58 | The last mismatched values not kept | Caught |
| 59 | More pages than declared tolerated until the end | Caught |
| 60 | Declared rows not compared | Caught |
| 61 | Fewer pages than declared tolerated | Caught |
| 62 | A total of no pages | Caught |
| 63 | A total on a file collection | Caught |
| 64 | A second total replaces the first | Caught |
| 65 | A total declared after a contradicting chain | Caught |
| 66 | A page on a file collection | Not caught |
| 67 | A page with other content makes a completed collection inconsistent | Caught |
| 68 | A page repeated with other content ignored | Caught |
| 69 | Any new cursor accepted | Caught |
| 70 | New pages after a final state | Caught |
| 71 | Pages unbounded | Caught |
| 72 | A page pointing to itself | Caught |
| 73 | A page pointing to the start or an earlier one | Caught |
| 74 | A page pointing to an earlier one | Caught |
| 75 | The same content under a new cursor | Caught |
| 76 | Empty pages with the checksum of a page with rows | Caught |
| 77 | An empty page marks its checksum as content | Caught |
| 78 | A chain that contradicts its declared total | Caught |
| 79 | Waits unbounded | Caught |
| 80 | Throttling recorded in any state | Caught |
| 81 | A repeated loss report with another reason | Caught |
| 82 | Access lost in any state | Caught |
| 83 | Lost access does not interrupt | Caught |
| 84 | A repeated resumption | Caught |
| 85 | Resuming under a changed capability | Caught |
| 86 | Resuming a final collection | Caught |
| 87 | A resumption that resumes nothing | Caught |
| 88 | A changed source at the collection's own revision | Caught |
| 89 | A changed source forgotten | Caught |
| 90 | A final collection abandoned | Caught |
| 91 | An abandonment repeated with another reason | Caught |
| 92 | Completing under a changed capability | Caught |
| 93 | Completing a chain that has not ended | Caught |
| 94 | Completing without a manifest | Caught |
| 95 | Completing with a mismatched part | Caught |
| 96 | Completing with a missing part | Caught |
| 97 | A mismatched part reported as missing | Caught |
| 98 | Mismatched receipts added to the rows | Caught |
| 99 | The digest of files without their checksums | Caught |
| 100 | The digest of pages without their checksums | Caught |
| 101 | A collection of empty pages not called empty | Caught |
| 102 | A repeated completion | Caught |
| 103 | Completing in any state | Caught |
| 104 | A completion after a retirement becomes effective by itself | Caught |
| 105 | Every completion replaces what counts | Caught |
| 106 | Adopting a collection that is not completed | Caught |
| 107 | A delayed adoption moves what counts | Caught |
| 108 | Adopting an older revision over a newer one | Caught |
| 109 | Adopting the effective collection moves the pointer number | Caught |
| 110 | Retiring a collection that never completed | Caught |
| 111 | A retired collection still counts | Caught |
| 112 | Staleness not shown | Caught |
| 113 | Duplicates shown as conflicting | Caught |
| 114 | Conflicting collections shown as duplicates | Caught |
| 115 | Retired collections shown with another standing | Caught |
| 116 | Running collections listed among the completed | Caught |
| 117 | A mismatch after verification forgotten while interrupted | Caught |
| 118 | Another manifest forgotten while interrupted | Caught |
| 119 | Another total forgotten while interrupted | Caught |
| 120 | A page with other content forgotten while interrupted | Caught |
| 121 | The cause of a contradiction not recorded | Caught |
| 122 | The time of a change of state not recorded | Caught |
| 123 | The defensive gap for differing totals | Not caught |
| 124 | The origin missing among the other collections | Caught |
| 125 | A wait at the bound refused | Caught |

# Deviations from the plan

- A contradiction is answered with the state `inconsistent` and no error; the reviewed text did not say how it is answered, and the plan was completed after the change review.
- Not every record is timed: a collection carries its start and its last change of state, a capability its last declaration, a subject its last pointer move; the plan was narrowed to that.
- `Complete` on a collection that completed returns no state.
- Contradictions arriving while a collection is interrupted end it as `inconsistent`; the reviewed text refused and forgot them, which the change review found.

# Provisional choices and undecided matters

The owner has not decided these; the implementation uses the labelled rules of the plan: only a verified capability proceeds, a limited one is refused; evidence is a bare reference without validity, product version or verifier; the condition is whatever the caller of `Start` states; any new declaration of a capability blocks the collections running under it; a later completion never replaces what counts without an adoption; a stale, conflicted or capability-withdrawn collection is reported and whether it is counted is left to whoever counts; the connection is part of the subject.

# Limits

- The package authorizes nothing and is reachable only in process; it must not be exposed before commands and adapters exist for it, and its views show a registry-wide completion number.
- A report is believed: a wrong page that never repeats is not detected, a chain without declared totals is summed against nothing, and a manifest derived by the adapter verifies nothing.
- Overlap across subjects is not detected: a parent payer's original and a member account's export, a month and a part of it, or two connections to one source each have their own effective collection.
- One report of a revision that is too high supersedes what runs and blocks lower starts of that subject for good.
- A contradiction that arrives after completion is refused and recorded nowhere.
- Two honest pages with rows and identical content end a collection as `inconsistent`; two honest collections paged differently stand as conflicting.
- Collections are never evicted and final ones count towards the limit; interruptions per collection and workspaces are not bounded.
- The Python import path is separate, and nothing may read both for one period.
- The clock is called while the registry's lock is held.
- State is in memory and lost on restart.
- The per-row mutation results are in the session's scratch files, not in the repository.

# Independent review

The design review of 2026-11-13 returned FAIL (10 HIGH, 7 MEDIUM, 1 LOW), traced to five structural holes, among them contradictions that could be undone and source revisions known only per collection; the plan was rewritten around rules for those. The re-review of the same day returned PASS with findings (1 HIGH, 8 MEDIUM, 5 LOW), said implementation may start once they were written into the text, and found no sequence giving a false completion, a double count or two effective collections; all were adopted.

The change review of the same day returned PASS with findings: no blocker, no HIGH, 3 MEDIUM and 9 LOW. It found no call sequence that gives a false `Complete`, a double count, two effective collections or a reach into another workspace, and confirmed the simplifications made after the first mutation run. All twelve findings were handled as the plan's change review resolution lists. The corrections made after it (contradictions while interrupted, their recorded cause, the recorded times, the origin in the list of other collections, a defensive gap, tests) were not reviewed again; they are covered by the tests and by the full mutation run after them.

All three reviews were read-only and executed nothing: every executed result in this record is the author's.
