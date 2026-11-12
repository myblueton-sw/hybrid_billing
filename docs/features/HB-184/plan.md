---
type: ImplementationPlan
title: Verified connector capabilities and a collection ledger that prevents false completion and double counting
description: Defines the first model of work item W-05, capability declarations per adapter version that are active only with verification evidence, and a collection ledger per workspace and connection in which contradicting evidence is final, source revisions are known per subject, and counted data is replaced only by an explicit step.
status: draft
---

# Scope and ownership

[HB-184](https://linear.app/hybrid-billing/issue/HB-184/w-05-verified-connector-capabilities-and-a-collection-ledger-that) is the first increment of work item W-05 (requirement HB-05). It is the eleventh increment built under the owner's instruction of 2026-11-12. Base: main `283118c`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the model decides whether received cost data counts as complete and whether it counts once, and is keyed by tenant. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-13:

- Requirement HB-05: AWS, Azure, GCP, OCI, Alibaba, VMware, HawkEye and TokenMeter are in the common design scope, but only combinations that were actually verified are activated. Provider originals are kept apart from pro forma and distributor sales data. Proposed direction: versioned adapters with `discover`, `validate_connection`, `fetch_costs`, `fetch_invoices`, `checkpoint` and support evidence; capability varies with contract, region and source version; an unsupported call is refused explicitly. Scale: the original of a parent payer and period is collected once and attribution is derived from it; page cursor, file manifest, byte and row count and checksum are verified; concurrency and call cost are limited per provider.
- Acceptance of HB-05: with one split file missing, a repeated page, a 429, a withdrawn permission and a changed source injected, a false "receipt complete" and duplicated cost are prevented. Linked source cases, all NOT_RUN: BILL01, COM01, PO07, FD03 (target accounts, period, split files, revisions, duplicates and omissions are checked).
- The neighbouring requirement on receipt idempotency states the rule this plan follows: the same key with the same content is a repeated receipt, the same key with other content is a conflict.
- Open decisions of HB-05, not made: the original invoice API and file paths, external call limits, and the full export contracts of TokenMeter and HawkEye.
- Mandatory detail contracts of W-05 that concern other things than this increment: AR-02 (mutual authentication, tenant binding of connections, endpoint restrictions, push signatures), DB-02 (provider profiles), QA-02 (final evidence before access is withdrawn), QA-07 (role separation), RT-01 (source event envelope; a late source after issue is a later adjustment).
- Existing code: the Python modules under `modules/focus_import` hold a working local FOCUS import path with its own connections, deliveries, receipts and publication, for files brought in by an operator. No Go package models connectors or collections. Everything is in memory or local files; the business database is not selected.
- The planning sources are untracked local files of the primary checkout and were read there.

# Decision

One Go package, `modules/connectors`, standard library only, with an in-memory registry. It holds what adapters are declared to support and what a caller reports that a collection received; it calls no provider, stores no credential and no received data, and has no entry point wired in this increment.

What this can and cannot show: an adapter, which does not exist yet, is a caller that reports what it says it received. The ledger therefore gives the precondition for the acceptance, not the acceptance: it refuses to call a collection complete unless the reports are complete and consistent, and it cannot know whether a report is true. A wrong page is detectable only when it repeats; a manifest verifies something only when it is the source's own listing.

The Python import path is not changed and not connected. Until the two are joined, nothing may read both ledgers for one period, because that would count the same data twice (stated).

Not in this increment, each mandatory for W-05: real adapters and network calls; connection authentication and push verification (AR-02); the source event envelope (RT-01); provider profiles (DB-02); call budgets and concurrency limits per provider.

## Capabilities: active only with evidence

A capability is a statement about one adapter version: (provider, adapter version, operation, data kind, condition) with a support state. All parts are identifiers or closed lists:

- operation: `discover`, `validate_connection`, `fetch_costs`, `fetch_invoices`, `checkpoint`;
- data kind: `provider_original`, `pro_forma`, `distributor_sales` for the two fetch operations, and `none` for the other three; any other pairing is refused;
- condition: contract type, region and source version, each an identifier, all three required and no wildcard, so that support under one says nothing about another. An operation that runs before these are known, such as discovery, must still be declared per condition (stated limit);
- state: `verified`, with an evidence reference; `limited`, with a reason code and an evidence reference; `unsupported`, with a reason code and, optionally, an alternative as an identifier.

`Declare` stores a capability under its five-part key. Declaring the same key again with identical content writes nothing; with other content it replaces the capability and increases its revision, which never goes back, also when the content returns to what it was. A new key is refused when the capability limit is reached; a replacement of an existing key, in particular a withdrawal, is never refused for room. Declarations belong to the installation, not to a workspace, and an evidence reference must not name a tenant.

`Permit(provider, version, operation, kind, condition)` answers one of: `allowed`, only for a `verified` capability; `limited`, with the reason; `unsupported`, with the reason and alternative; `unknown`, when nothing is declared for that exact key. Only `allowed` lets an operation proceed here; `limited` and `unknown` are refusals too, because the requirement activates only what was actually verified. The answer carries the capability's revision.

Stated limits of the gate, each provisional until the work that owns it: evidence is a bare reference with no validity period, no binding to a product version and no verifier, unlike onboarding evidence, because what counts as verification is the owner's open decision; and the condition is asserted by whoever starts a collection, so nothing here stops a connection in one region from being started under another region's verified capability until provider profiles or onboarding bind the condition to the connection.

## Subjects and collections

The subject of a collection is (workspace, connection, operation, scope, period, data kind), all identifiers or closed lists. Operation and data kind are in the subject, so costs and invoices, and a provider original, a pro forma and a distributor's sales data, never share a subject and are never merged or substituted for one another.

A connection of a workspace is bound to the provider of the first collection that was accepted on it; a start under another provider is a conflict. The binding is kept per workspace and connection and is written only by an accepted `Start`, so the same connection identifier in another workspace is unrelated.

A collection is one attempt to receive a subject's data at one source revision. `Start` creates it with a caller-assigned identifier unique in the workspace, the subject, the adapter (provider and version), the condition, the source revision and the shape, `files` or `pages` (with a start cursor). The source revision is a number that the adapter takes from the source, never from the time of collection, and that is higher for a later state of the same source; what it is for each provider is undecided, and an adapter that cannot produce one cannot use this ledger (stated limit).

`Start` asks `Permit` and is refused unless the answer is `allowed`; the capability's revision is stored. Starting a collection whose identifier exists succeeds without writing when everything is identical and is a conflict otherwise; this is decided before anything else, so a repeat still succeeds after the subject's known revision has moved on. A refused `Start` writes nothing at all, not even the revision it named.

States: `collecting`, `interrupted`, and five final states: `completed`, `superseded`, `inconsistent`, `abandoned`, and `retired`, which only a completed collection can reach.

### Source revisions are known per subject

Each subject keeps the highest source revision anyone has reported for it. It is raised by an accepted `Start`, with the new collection's revision, and by `SourceChanged(collection, revision)`, in every state of the named collection, final ones included: a revision at or below that collection's own is refused; one above it and at or below the known revision writes nothing and succeeds; one above the known revision raises it.

Whenever the known revision rises, every collection of the subject that is `collecting` or `interrupted` at a lower revision becomes `superseded` in the same write: what it was receiving describes a source state that no longer exists, whichever collection the news came through. `Start` at a revision below the known one is refused, so a running collection is always at the known revision. The known revision is never lowered: one report of a revision that is too high supersedes what is running and blocks every lower start of that subject for good, and a correction by an operator is not built (stated limit).

### Contradicting evidence is final

The same key with the same content is a repeat and writes nothing. The same key with other content is a contradiction, and a contradiction about something already accepted makes a collection that is `collecting` or `interrupted` `inconsistent`, finally: the registry cannot know which report was right, and a later report that agrees again proves nothing. The cases are listed with the shapes below. The order of arrival therefore matters: a wrong report, the right one, and then a delayed copy of the wrong one end in `inconsistent` although the source did not change; a new collection at the same revision can be started (stated).

How a contradiction is answered: the operation that brings it returns the state `inconsistent` without an error, because it was applied, and the collection records what contradicted (the manifest, a part, the total, a page, a loop, repeated content or the declared totals) and when. A caller must read the returned state and not only the error; the same report again is then refused like anything else on a final collection.

An operation of the other shape, a manifest or a part on a paged collection, a page or a total on a file collection, is refused.

### Files

`DeclareManifest(origin, parts)` fixes the complete list of parts once, each with an identifier, a byte count, a row count and a checksum, together with its origin: `source_listed`, when the list is the source's own, or `adapter_derived`, when the adapter built it from what it downloaded, in which case it verifies nothing and is reported as such wherever the collection is shown. An empty list is allowed and means the source says there is nothing; a list that names a part twice is refused. The order of the list is not content: declaring the same parts again in any order writes nothing; other parts or values make the collection `inconsistent`.

`ReceivePart(part, bytes, rows, checksum)` records what arrived for a part of the manifest. A part that is not in the manifest, or any part before the manifest is declared, is refused with nothing written. Otherwise:

- all three values equal the manifest's: the part is verified; receiving a verified part again with the same values writes nothing;
- a value differs and the part is not verified: the part stays unverified, and the collection keeps a count of mismatches for that part and the last values that arrived; a later receipt with the right values verifies it, since a transfer that failed and was repeated is ordinary;
- a value differs and the part is verified: the collection becomes `inconsistent`.

### Pages

`ReceivePage(cursor, next, rows, checksum)` records one page; `next` is empty on the last page. `DeclareTotal(pages, rows)` may be called once, when the source states totals, with at least one page; again with the same values it writes nothing, with others the collection becomes `inconsistent`. So does a chain that contradicts declared totals, whichever comes last: more pages received than declared, or a chain that has ended with other pages or rows than declared.

- a cursor that was already received with the same `next`, rows and checksum is a repeated page: nothing is written and nothing is counted twice;
- a cursor that was already received with other content makes the collection `inconsistent`;
- a cursor that was not received must be the one the collection is waiting for, the start cursor at first and then the `next` of the last page; any other is refused with nothing written;
- an accepted new page (the rule before this one comes first) whose `next` is its own cursor, the start cursor or any cursor already received is a loop, and a new page with rows whose checksum was already received under another cursor for a page with rows is the same content served twice; both make the collection `inconsistent`.

Two honest pages with rows and identical content also end in `inconsistent` by this rule; the adapter is expected to digest a page together with its position when a source can legitimately repeat content (stated limit). Nothing declares a page's content in advance, so a wrong page that is never repeated is not detected, and without declared totals the sum of a chain is checked against nothing (stated limits). The cursor an adapter reports must be a deterministic digest of the provider's cursor: an adapter that numbers pages as they come defeats the detection of repeats and loops (stated obligation).

### What can happen while collecting

- `Throttled(retryAfter)` records that the source asked to wait (a 429): a saturating counter and the last wait, a non-negative duration up to a bound, stated by the caller. It is a diagnostic that may be reported more than once for one event and is never used for a decision; the state does not change. Nothing here limits how long a caller keeps retrying; call budgets are deferred.
- `AccessLost(interruption, reason)` names the number of the interruption it opens, with a reason from a closed list (`permission_withdrawn`, `credential_expired`, `connection_disabled`). A collection that is `collecting` accepts the number after its current one and becomes `interrupted`; what was received is kept. A number at or below the current one with the reason that interruption had writes nothing and succeeds, whatever the state, so a delayed repeat cannot interrupt a collection that was resumed; another reason for a known number, or a number that skips one, is a conflict, and the next number on a collection that is not `collecting` is refused.
- `Resume(interruption)` names the interruption it ends:

| Number | Collection is `interrupted` | Collection is `collecting` |
| --- | --- | --- |
| The current one | Resumed, when the capability is still `allowed` under the stored revision; else refused | Writes nothing and succeeds: it was already resumed |
| Below the current one | Writes nothing and succeeds: that interruption was resumed, and the newer one stays in force | Writes nothing and succeeds |
| Above the current one | Refused | Refused |

- While interrupted, the operations that change anything are `Resume`, `SourceChanged`, `Abandon` and a report that contradicts something already accepted; repeats are answered as described under Repeats.

### Leaving without success

`Abandon(reason)` makes a collection that is `collecting` or `interrupted` `abandoned`, with a reason from a closed list (`capability_changed`, `cursor_expired`, `limit_reached`, `part_unverifiable`, `operator`). It is never refused for room. It is the exit for every collection that cannot finish: one whose capability was declared again (which blocks its completion for good, below), one whose cursor expired while it was interrupted, one that reached the page limit before the last page.

### Completion

`Complete` is accepted only when the collection is `collecting`, the capability's answer is `allowed` under the same revision as at the start, and everything declared arrived: for files, the manifest is declared and every part is verified; for pages, the last page was received and, when totals were declared, the pages and rows equal them. The capability is read and the state written in one step. Otherwise it is refused and the answer lists what is missing: no manifest, each unverified part with whether it is missing or mismatched, an unfinished chain, totals that differ, a changed capability.

Any new declaration of the capability, also an evidence refresh that leaves it `verified`, changes its revision and so blocks `Complete` and `Resume` for every running collection of that key: they are abandoned and started again. This is strict on purpose: the registry cannot tell a refresh from a correction (provisional).

A completed collection stores its totals of parts or pages, bytes and rows, its content digest, computed by the registry (for files over the parts sorted by identifier, each with identifier, bytes, rows and checksum; for pages over the chain in order, each page with rows and checksum and without its cursors), whether it is empty, and a completion number from one counter of the registry, which, not time, orders completions. `Complete` on a collection that completed, also one retired since, writes nothing and succeeds; it returns no state, and the caller reads the collection. After completion nothing more is received: a late report for a completed collection is refused and recorded nowhere (stated limit).

## What counts: one effective collection, changed only on purpose

Each subject has at most one effective collection, and the pointer moves in three ways only:

- a collection that completes when no collection of the subject has completed before becomes effective;
- `Adopt(collection, pointer)` makes a completed collection of the subject effective. The subject numbers every move of its pointer, and `Adopt` names the number it saw: any other number is a conflict, so a delayed repeat of an adoption cannot undo a later decision. When a collection is effective, the adopted one's revision must not be lower; when none is, after a retirement, any completed collection that is not retired may be adopted. The one it replaces stays completed and is no longer effective;
- `Retire(collection, reason)` makes a completed collection `retired`, with a reason from a closed list (`conflict`, `capability_withdrawn`, `wrong_source`, `operator`); if it was effective, the pointer moves to none and the subject has no effective collection until one is adopted: a collection that completes later waits like any other.

So a later completion never replaces counted data by itself, whether it is larger, smaller or empty: it is reported and waits for a decision. This is the conservative reading of "a late source is a later adjustment", and it is why an adapter that lists an export too early and completes an empty higher revision cannot make a period cost nothing.

`Effective(subject)` returns the effective collection, if any, with: its revision, totals, digest, emptiness and manifest origin; the capability's stored revision beside its present answer and revision, so that a withdrawal after completion is visible; `stale`, when the subject knows a higher revision than the effective one's; the pointer's number; and the other collections of the subject that completed, each with its standing against the effective one: `duplicate` (same revision, same digest), `conflicting` (same revision, another digest), `newer` (higher revision, waiting to be adopted), `older`, or, when nothing is effective, `candidate`; retired ones are listed with their reason. Two honest collections of one revision that were paged differently have other digests and stand as `conflicting` (stated limit). Whether a stale, conflicted or capability-withdrawn effective collection may still be counted is decided by whoever counts; the ledger reports and does not decide (stated).

What this prevents and what it does not: within one subject, nothing is counted twice and nothing replaces what counts without a step. Across subjects nothing is compared: a parent payer's original and a member account's own export, a month and a part of it, or two connections to the same source are different subjects and each has its effective collection. The requirement's "collect the parent original once and derive attribution" is not enforced here; whether the connection belongs in the subject at all is a question for the owner (stated limit).

## Repeats

No write identifiers are used: every operation is its own key. A repeat of an operation that was applied, with the same content, writes nothing and succeeds with the stored result, in every state the collection has since reached, final ones included: `Start`, `DeclareManifest`, `DeclareTotal`, `ReceivePart` of a verified part, `ReceivePage` of a received page, `AccessLost` with the same reason while interrupted, `Resume` of an ended interruption, `SourceChanged` with a revision the subject already knows, `Abandon` and `Retire` with the same reason, `Complete`. An `Adopt` is not repeatable by content: its pointer number makes a repeat a conflict once the pointer has moved, and the caller reads the state. `Throttled` is the stated exception. The same operation with other content is a conflict or, where the rules above say so, makes the collection `inconsistent`. Anything else on a final collection is refused.

## Tenant boundary, time and limits

Every collection operation takes the workspace and addresses only that workspace's collections and subjects; those of another workspace are not found, in the same way as ones that do not exist. Capabilities are installation-wide and hold nothing of a tenant. The registry authorizes nothing and must not be exposed directly (stated limit). Connection, scope, period, part and cursor identifiers, evidence references and reason codes are identifiers under the project's identifier rule (letters, digits and `. _ : -`, at most 128 bytes), never free text, a path, an address or a secret of another shape; the package has its own copy of the rule (stated). A checksum is lowercase hexadecimal of 8 to 128 characters, one canonical form, so that the same digest in another spelling is not taken for other content.

The registry's clock is injected, read once per operation, and never goes backwards. A collection carries the time it was started and the time of its last change of state; a capability the time of its last declaration; a subject the time its pointer last moved. Single receipts are not timed.

Required and positive at construction, without defaults: capabilities; collections per workspace; parts per manifest; pages per collection; the largest byte count and row count of one part or page, bounded so that the totals of a full collection cannot overflow, which is checked at construction; the longest retry wait. The collection limit counts every collection of the workspace, final ones included, so abandoning frees no room. At a limit a new key is refused and nothing is evicted; `AccessLost`, `SourceChanged`, `Abandon`, `Retire` and a replacing `Declare` change existing records only and are never refused for room. Two things grow without a limit of their own: the interruptions of one collection, one entry per loss of access, and the number of workspaces (stated). The completion counter is one for the whole registry, so a completion number shown to a tenant would tell it about other tenants' activity; the views are not for exposure as they are (stated). Retention is the owner's open decision and belongs with persistence (stated limit, an exposure blocker like those of the other registries).

## What is not decided here

Which combinations are verified, by what evidence and for how long; what a source revision is for each provider; the original invoice API and file paths; call limits; whether a stale, conflicted or withdrawn collection may be counted; whether the connection belongs in the subject, and how overlapping scopes and periods are found; who may declare capabilities, operate collections, adopt and retire.

# Allowed files

`modules/connectors/*` (new); `docs/features/HB-184/*`.

# Verification

- Literal table tests with a test clock: every `Permit` answer and a neighbouring contract, region and source version; refused pairings of operation and kind; `Start` refused for each answer but `allowed`; a capability declared again, also with returning content, blocking `Complete` and `Resume`; a replacement at the capability limit.
- The sequences the design review named, each as a test: a mismatch after verification, with the delayed correct repeat arriving afterwards; a second manifest with other content; `SourceChanged` while interrupted; a sibling collection of a subject superseded through another collection's news, and a start below the known revision; two completions of one revision with other content reported as conflicting; a collection blocked by a new declaration, by an expired cursor and by the page limit, each leaving by `Abandon`; costs and invoices as separate subjects; a connection started under a second provider; the same page under a new cursor; an empty higher revision that completes and does not replace what counts until adopted. And those of the re-review: a delayed `Adopt` after a later decision; a delayed `AccessLost` after a resumption; a retirement followed by an adoption with nothing effective; a wrong report, the right one and a delayed wrong one; operations of the other shape; a connection identifier used in two workspaces; `SourceChanged` on each final state and at or below the known revision; a start at the known revision and a repeated start after supersession; a late report on a completed collection; totals contradicted by an ended chain.
- The injected faults of the acceptance as ledger preconditions: one split file missing or mismatched; a repeated page; a 429; lost access and its resumption, with a delayed `Resume` not clearing a newer interruption; a changed source; in each, `Complete` refused with the reason listed until the condition is met.
- Pages: the three loop forms; declared totals met and not met; an empty first page. Manifests: empty, derived by the adapter, the limits.
- What counts: first completion, adoption, retirement, staleness, each standing; the capability's present answer beside the stored one; three data kinds and two operations as separate subjects.
- Repeats of every operation after every state returning the same answer and writing nothing; refusals writing nothing.
- The tenant boundary: operations in one workspace leave every observation of another unchanged, and another workspace's collection is not distinguishable from one that does not exist.
- Properties with a fixed seed over random histories of raw attempts, with capability declarations among them, against a reference model written in the test that decides acceptance itself: the same attempts are accepted; a collection completes only when the model says everything declared arrived; totals equal the model's sums with every part and page counted once; each subject has at most one effective collection, it changes only through a first completion, an adoption or a retirement, and it is the same on every read; every final state is absorbing; no running collection is below its subject's known revision.
- Concurrency under the race detector: concurrent receipts of the same parts and pages, each counted once.
- Mutation check by a script in Python with per-row results appended to files that are never overwritten.
- `gofmt -l`, `go vet ./...`, `go test -race -count=1 ./...`.

Remaining NOT_RUN at system level after this ticket: BILL01, COM01, PO07, FD03, AUD-AR-02, AUD-DB-02, AUD-QA-02, AUD-QA-07, AUD-RT-01. This ticket does not complete W-05, and the acceptance of HB-05 itself, which injects faults into real collection, is NOT_RUN.

# Deferred

Real adapters for any provider, with their evidence; connection authentication, endpoint restrictions and push verification (AR-02); the source event envelope (RT-01); provider profiles and the binding of a condition to a connection (DB-02); validity and verifier of capability evidence; call budgets and provider concurrency; overlap across subjects and deriving attribution from a parent payer's original; joining the Python import path; commands, permissions and screens S01, S03; retention; persistence.

# Design review resolution

The independent design review of 2026-11-13 (read-only, nothing executed) returned FAIL, not implementable as written, without asking for a redesign: 10 HIGH, 7 MEDIUM and 1 LOW, traced to five structural holes. All eighteen were adopted, the structural ones as rules of their own.

| Finding | Resolution |
| --- | --- |
| S1, D1, D2, D9 contradicting evidence not sticky | A contradiction about something accepted makes the collection `inconsistent`: a mismatch after verification, another manifest, another total, a page with other content, a loop, the same content under another cursor |
| S2, D3, D4 source revisions known per collection only | The subject keeps the highest known revision; news through any collection in any state supersedes every running collection below it; a start below it is refused |
| D5 a same-revision completion with other content called a duplicate | A content digest at completion; `conflicting` as its own standing |
| S3, D6 no exit without success | `Abandon` with closed reasons, never refused for room; the blocking effect of a new declaration stated |
| S4, D7, D8 the subject | The operation is in the subject; a connection is bound to its first provider; overlap across subjects stated as not detected, with the question for the owner |
| D10 an empty higher revision replacing data at once | Nothing replaces what counts without `Adopt`; emptiness reported |
| S5, D11, D12 repeats | Defined for every operation; `Resume` names its interruption; `Throttled` a stated diagnostic |
| D13 a capability withdrawn after completion | The present answer shown beside the stored revision; `Retire` |
| D14 ties between completions | A completion counter, not time |
| D15 evidence and condition unbound | Stated as limits; pairing of operation and kind enforced; replacements never refused for room |
| D16 the reference model | It decides acceptance from raw attempts; the named sequences and properties added |
| D17 overclaims | The ledger called a precondition; undetectable cases stated; manifest origin and optional totals added |
| D18 smaller items | Bounded mismatch record, receipts after completion, one checksum form, the Python path, the origin of a revision and atomic completion stated |

# Design re-review resolution

The narrow re-review of 2026-11-13 (read-only, nothing executed) returned PASS with findings and said implementation may start once its findings were written into the text: sixteen of the eighteen first findings resolved and two partly; fourteen new findings, 1 HIGH, 8 MEDIUM and 5 LOW. It found no sequence that produces a false `Complete`, a double count or two effective collections. All fourteen were adopted above; the text after them was not reviewed again as a design, and the change review reads it against the code.

| Finding | Resolution |
| --- | --- |
| E1 a delayed `Adopt` moving what counts | The pointer is numbered and `Adopt` names the number it saw |
| E2 adoption and listing with nothing effective | Any completed, not retired collection may be adopted; candidates and retired ones listed |
| E3 a delayed `AccessLost` interrupting again | `AccessLost` names the interruption it opens |
| E4 the scope of the provider binding | Per workspace and connection, written only by an accepted `Start` |
| E5 the digest undefined | Its inputs stated; manifest order is not content; other pagination stands as conflicting |
| E6 a revision that is too high | Stated as a limit |
| E7 the order of arrival deciding `inconsistent` | Stated |
| E8 blocks without a fitting exit | A chain against declared totals is `inconsistent`; `part_unverifiable` added |
| E9 what the collection limit counts | Stated |
| E10 `SourceChanged` at or below known levels | One rule |
| E11 the interruption rules | A table; repeats while interrupted |
| E12 small rules | Other shapes, repeated part identifiers, a total of no pages, the order of checks at `Start`, precedence of the page rules |
| E13 "first to complete" after a retirement | Reworded |
| E14 identical honest pages; contradictions after completion | Stated as limits |

# Change review resolution

The independent change review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker, no HIGH, 3 MEDIUM and 9 LOW. It found no call sequence that gives a false `Complete`, a double count, two effective collections or a reach into another workspace, and it confirmed each of the five simplifications made after the first mutation run.

Simplifications, each confirmed by the review: `ReceivePart` needs no separate check that the manifest is declared, because no part exists before it; `ReceivePage` needs none that the chain ended, because the awaited cursor is then empty and no cursor is empty; the loop rule needs no separate comparison with the start cursor, because that cursor is a received page as soon as any page is accepted. Kept as defence although nothing reaches them: the answer beside the revision when a capability is checked, the shape check of `ReceivePage`, and the gap for differing totals in `Complete`.

| Finding | Resolution |
| --- | --- |
| C1 a contradiction while interrupted refused and forgotten | It makes the collection `inconsistent` while interrupted too; tests for all four kinds; the model follows |
| C2 the answer to a contradiction | Stated above; the collection records what contradicted and when |
| C3 "every record carries its time" | Narrowed to what is timed; start, last change, declaration and pointer move are recorded |
| C4 the origin missing among the other collections | Added |
| C5 the gap for differing totals | Added as a defensive check |
| C6 the same-content rule narrower than its text | The text now says both pages have rows |
| C7 `Complete` on a retired collection | Stated |
| C8 the error for the next interruption on a collection that is not collecting | The text aligned |
| C9 unbounded interruptions and workspaces | Stated |
| C10 the registry-wide completion number | Stated |
| C11 the bound of a wait | Inclusive, stated and tested |
| C12 test gaps | Added: both remaining reasons to abandon, a changed source through inconsistent, abandoned and retired collections, a distributor's sales as a subject, completing a retired collection, a second manifest above the limit; three assertions that could not fail corrected; the tenant test now runs against a workspace that exists |
