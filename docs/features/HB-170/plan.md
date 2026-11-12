---
type: ImplementationPlan
title: Relationship revisions with effective intervals, change plans and atomic activation
description: Defines the first model of work item W-02, five separate relationship kinds held as immutable revisions of dated timelines, changed only through a sealed plan that is validated as a whole and activated atomically, with queries pinned to a revision.
status: draft
---

# Scope and ownership

[HB-170](https://linear.app/hybrid-billing/issue/HB-170/w-02-relationship-revisions-with-effective-intervals-change-plans-and) is work item W-02 (requirement HB-02). It is the fourth increment built under the owner's instruction of 2026-11-12. Base: main `ea7dbba`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the model decides tenant boundaries and billing attribution. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-13:

- Requirement HB-02: organization hierarchy, viewing groups, CSP payer relationships, contract pools and allocation pools are separate. A move or release has an effective date and a recorded time and does not change a past confirmed attribution. Proposed direction: relationship revisions with `[from, to)` intervals and cycle and overlap constraints; a change plan with a fixed target manifest and an impact preview is approved and published atomically per dependent pool; the current organization analysis is a view separate from the settlement of that time. Scale: 100,000 relationship changes are prepared in chunks, the whole new revision is validated, and then the active pointer is switched so that no partial state of a shared pool is visible; a viewing group is the de-duplicated union of source and allocation identifiers. Acceptance: mid-month move, retroactive change, concurrent modification and group release preserve billing, permission and included quantity, and a failed unit can be resumed.
- Open decisions of HB-02: organization depth, attribution priority of shared accounts, contract handling when mid-month history is missing.
- Acceptance cases linked to HB-02, all NOT_RUN: GH03 (shared-source allocation and remainder totals are preserved and a duplicated viewing group does not increase the billed amount), GH04 (exposure of permissions, past documents and waiting files after a group move or release), GH05 (confirmed documents stay and only the unissued effect is re-approved after a mid-month move or retroactive change), GH06 (group included quantity, discount and minimum are preserved by contract after joining and leaving).
- Existing code: `modules/organizations` (HB-168) holds workspaces, parties with one immutable parent pointer, billing profiles and contract revisions, without dated relationships; its plan deferred the reconciliation of that pointer to W-02. `modules/auth` (HB-169) decides permissions and does not read relationships. `modules/allocation` holds a calculator and no pool membership. All are in memory; the business database is not selected.
- The planning set names multi-level payment structures (AWS Billing Transfer, GCP reseller subaccounts) and account attribution with validity periods on the screens.

# Decision

One Go package, `modules/relationships`, standard library only, with an in-memory registry. It records and answers; it authorizes nothing, computes no amount and issues nothing. No entry point is wired to it in this increment.

Deviation, stated: the repository structure places organization nodes, reporting groups and account relationships in `organizations`. A separate package is chosen because `modules/organizations` keeps current-state records with a version per record, while relationships are workspace-wide immutable revisions switched by one pointer; the two concurrency models are kept apart until persistence is designed.

## Relationship to `modules/organizations`

`Party.ParentID` in `modules/organizations` remains the only authority for the legal entity of a billing document. The `hierarchy` kind is the dated organizational tree for analysis and attribution and must never be used to derive a legal entity. The package does not check the two against each other, does not check that an identifier or a workspace exists or is active there, and does not bind `contract_pool` to a contract identifier. Reconciling the two hierarchies, which HB-168 deferred to W-02, is not done by this ticket; a follow-up ticket is created when this ticket closes. "Revision" here is a workspace relationship revision and is unrelated to `ContractRevision`.

## Relationship kinds

A closed set of five kinds. Each relates a member identifier to a target identifier inside one workspace for an interval of days. Identifiers are opaque and member and target of one entry differ. Workspace, plan, member and target identifiers follow the identifier rule of `modules/organizations` (its alphabet, at most 128 bytes, compared bytewise), duplicated in this package with a test that pins the bound and the alphabet. A kind never implies another, and a query for one kind never reads another.

| Kind | Member and target | Targets per member on one day | Further rule |
| --- | --- | --- | --- |
| `hierarchy` | unit, parent unit | at most one | no cycle; chain length at most the configured depth |
| `payer` | account, payer account | at most one | a payer has no payer on the same day |
| `contract_pool` | member, pool | at most one | an identifier is never both member and pool in this kind |
| `allocation_pool` | member, pool | at most one | an identifier is never both member and pool in this kind |
| `view_group` | member, group | any number | an identifier is never both member and group in this kind |

`allocation_pool` and `contract_pool` are membership only; the pool versions of later work items will reference a relationship revision number.

Assumption, labelled: "at most one pool per member per day" stands in for the open question of shared-account attribution priority. With one pool there is no priority to apply, and a shared account that must count in two pools on the same day is rejected until the owner decides a rule. Attribution of a source account to an organization unit with validity periods is not modelled by this ticket and is deferred.

## Days and intervals

A day is a civil date without time zone, built by `NewDay(year, month, day)` for the years 1 to 9999 and rejected when it is not a real date; days are totally ordered. An interval is half-open, `[from, to)` with `from` before `to`, or open-ended, `[from, ∞)`, built by explicit constructors; the zero value is invalid, so a missing end can never be read as "forever". Two entries touch when one's `to` equals the other's `from`.

## Timelines and revisions

The unit of record is a timeline: for one kind and one member, the complete list of (target, interval) entries, stored sorted by (from, target). Within one timeline no two entries with the same target overlap or touch (they must be written as one entry, which gives a canonical form), and for the four exclusive kinds no two entries overlap at all; entries with different targets may touch, which is how a move on a day is written. An empty timeline is identical to an absent one: it is never stored, and an identifier is a member or a target of a kind only through a stored entry.

A gap means that no relationship is recorded for that day. The package cannot distinguish "belongs to none" from "history is missing" and does not answer the open decision on missing mid-month history. Until the owner decides, consumers in the allocation and contract work items must not derive an amount from "none" for a member that has any entry of the same kind anywhere in the pinned revision without an explicit operator confirmation; that rule belongs to those work items and is NOT_RUN here.

A revision is the immutable set of all timelines of one workspace, numbered from 1 without gaps, with the recorded time from the registry's injected clock, the plan that produced it and that plan's manifest digest. A revision shares unchanged timeline values with its base, and a stored timeline is never mutated. Revisions are never changed or deleted. Revision 0 exists for every workspace, is empty and has the zero digest. A workspace has one active revision, initially 0. Actor and reason of a change are not recorded here; they belong to the audit record of the command layer (W-33).

Past attribution is preserved by pinning: every query names the revision it reads. A settlement stores the revision number it used and reads the same answers forever, whatever is activated later, including a retroactive change. The "current organization" view reads the active revision. Deciding which unissued results must be recomputed or re-approved after a retroactive change belongs to the billing work items; this package supplies the preview and the pinned reads they need.

## Change plans

The only way to change relationships is a plan. Plan identifiers are create-only per workspace, forever. States: draft → sealed → activated; draft → rejected; draft or sealed → discarded. No other transition exists.

1. `Draft(workspace, plan, base)` opens a plan. The base must be the active revision, and the number of open (draft or sealed) plans of the workspace must be below its limit.
2. `AddChunk(workspace, plan, index, timelines)` is allowed only in draft. Each timeline states the complete new list for its kind and member; an empty list removes the member from that kind. The input is deep-copied and its entries sorted before validation and storage. Adding an index again with identical content is accepted and changes nothing, so an interrupted upload resumes at the failed chunk; an existing index with different content is a conflict. Chunk content is compared after it is put in canonical order (timelines by (kind, member), entries by (from, target)). A chunk is refused, without being stored and without changing the plan's state, when it is malformed in itself (unknown kind, invalid identifier or interval, member equal to target, a (kind, member) twice inside the chunk, the chunk index, timelines per chunk or entries per timeline over its limit), when one of its timelines violates the per-timeline rules (same-target overlap or touch, any overlap in an exclusive kind), or when it names a (kind, member) already stored under another index. `AddChunk` does not check the base.
3. `Seal(workspace, plan, chunks)` distinguishes two failures. A precondition failure — the plan is not in draft, an index below `chunks` is missing, an index at or above `chunks` is stored, or the base is no longer the active revision — returns an error and leaves the plan unchanged (a plan on a stale base can never be sealed or activated; it counts as open until it is discarded). Otherwise the candidate is built as the base with the named timelines replaced and is validated as a whole. A validation failure — a cross-timeline rule violation, timelines per plan or entries per plan over its limit, or a plan in which no timeline differs from the base — moves the plan to rejected for good and returns the violations as data. A plan over a plan limit or without any change is rejected with that single rule before the cross-timeline rules are evaluated. On success the plan is sealed and immutable and a preview is returned.
4. `Activate(workspace, plan, digest, expected)` requires a sealed plan, the same digest, and `expected` equal to both the plan's base and the workspace's active revision; `expected` repeats the base deliberately, as the caller's read-back. It then makes the candidate the next revision and the active one in a single step under the registry lock. A failed activation changes nothing and consumes no revision number. If another plan was activated in between, activation fails with a conflict; the plan is stale and can only be discarded and drafted again on the new base. There is no automatic rebase.
5. `Discard` closes a draft or sealed plan. On a rejected, activated or discarded plan it is a state error and changes nothing. Rejected and discarded plans keep only their identifier, state and base, and a rejected plan its violations. A draft that is refused creates nothing, not even a workspace record.

Readers see the old revision or the new one, never a mixture, for every pool of the workspace at once. This workspace-wide switch is stronger than atomic publication per dependent pool and is a provisional deviation with a consequence: one plan at a time per workspace can be approved and activated, and any activation invalidates every other open plan of that workspace together with its approval. Revisions per pool, or rebase, are a follow-up.

The manifest is a target state, not a list of edits. It is independent of order, a repeated chunk cannot apply twice, and the preview is a plain difference between two states. Its cost, stated: every change to a member resubmits that member's whole timeline and can rewrite its past entries; the only guard is the preview's earliest changed day, which the approval layer must show.

The digest is what an approver approves; approval itself belongs to `modules/auth` and the later approvals work. The digest is SHA-256 over: a fixed version tag; the workspace; the base revision number and the base revision's digest; the timeline count; then, per timeline sorted by (kind, member), the kind, the member, the entry count and, per entry in stored order, the target, `from` as an eight-digit date, and an end marker byte followed by the `to` date or by nothing. Every integer (revision number, counts, string lengths) is 8 bytes big-endian unsigned; every string, including the kind's name, is prefixed by its length; a date is exactly eight ASCII digits `YYYYMMDD`; the end marker is `0x00` for an open-ended entry with nothing after it, or `0x01` followed by the eight-digit `to`; the base digest is its 32 raw bytes. Timelines equal to the base are included; the plan identifier and the chunking are not. A sealed plan stores its digest, its summary, the changed timelines and the preview lists derived from them (one change record per changed timeline and the affected targets); the candidate shares unchanged timelines with the base. An activated plan drops the changed timelines and keeps the preview lists for as long as the process lives (a stated limit).

Violations are returned sorted by (kind, member, rule), at most one per (kind, member, rule) and at most the configured number, with a truncation flag; error text carries no identifiers.

## Limits

All required and positive at construction, none with a default: hierarchy depth (between 1 and a fixed package maximum of 32, because the organization depth is the owner's open decision), entries per timeline, timelines per chunk, chunks per plan, timelines per plan, entries per plan, open plans per workspace, violations returned, and page size. A chunk index must be below the chunk limit. `AddChunk` enforces the chunk index, timelines per chunk and entries per timeline; `Seal` enforces timelines and entries per plan. The total size of a revision is not bounded in this increment.

## Preview

`Seal` returns, per kind: the number of timelines that differ from the base, the entries of those timelines before and after, and the earliest day on which any member's target set differs (or none). `ChangedMembers(workspace, plan, kind, after, limit)` pages through the changed members in identifier order with, for each, the maximal day intervals on which the target set differs and the sets of all targets of the member's timeline before and after (whole timeline, not per interval). `ChangedTargets(workspace, plan, kind, after, limit)` pages through the targets that gain or lose a member on any day. Both are readable while the plan is sealed or activated.

## Queries

Every query takes the workspace and a revision number. Revision 0 answers none or empty; a number above the latest is not found. "None" is returned as an explicit flag, never as an empty identifier. A query on a kind that does not suit it is an invalid request. Returned slices are copies.

- `TargetOn(kind, member, day)`, exclusive kinds only: the target or none.
- `Segments(kind, member, interval)`, exclusive kinds only, bounded interval only: the ordered pieces that cover the interval exactly, each with its target or none. Entries, including open-ended ones, are clipped to the interval, and pieces are maximal (adjacent pieces never have the same answer).
- `TargetsOn(kind, member, day)` and `MembersOn(kind, target, day)`: sorted identifier lists.
- `Chain(member, day)` for `hierarchy`: the ancestors from parent to root.
- `Union(groups, day)` for `view_group`: the sorted, de-duplicated set of members of all named groups on that day. De-duplication on the identifier is necessary, not sufficient, for GH03: without a typed identifier namespace, two identifiers that denote the same thing are not merged (a stated limit).
- `Revision(n)`: recorded time, plan and digest. `Active()`: the active revision number.
- `Plan(plan)`: the state, the base, the stored chunk indices and, when sealed or activated, the digest and the per-kind summary, when activated the revision number, and when rejected the stored violations. A caller that lost the answer of `Seal` or `Activate` reads it here.

`MembersOn`, `TargetsOn` and `Union` return whole lists in this increment; paging is added with the command layer.

## Validation of the candidate

- Per timeline: the ordering, overlap and touching rules; the entry limit.
- Member and target disjointness for `contract_pool`, `allocation_pool` and `view_group` over the whole candidate regardless of days.
- `payer`: for every entry, the target has no payer entry that overlaps the entry's interval. Touching is allowed.
- `hierarchy`: depth D means that `Chain` returns at most D identifiers on every day; a unit without a parent has chain length 0. For every entry the chain above the member is walked within the entry's interval through overlapping parent entries, narrowing the interval at each step. Each branch of the walk stops at the first of: an identifier already on the path, including the start member, recurs for a non-empty interval, which is `cycle`; the number of ancestors on the path, excluding the start member, would exceed D, which is `depth`. A cycle of more than D+1 identifiers is therefore reported as `depth`; both reject. The walk continues over the other day ranges of the entry, so a member is reported with every rule it breaks on some day (at most one violation per rule), which is what a day-by-day reading gives.

Validation covers the whole candidate, not only the changed timelines, in this increment. Its cost is bounded by entries × depth × the ancestor entries overlapped, hence by the limits; a parent with very many dated entries above very many members is slow, and incremental validation is deferred. The scale test records its duration and asserts none.

## Registry and isolation

In memory, one read-write lock, an injected clock. `Seal` validates under the registry lock in this increment, so a slow seal delays every workspace (a stated limit that goes with persistence). Plans and revisions are keyed by workspace; an identifier in one workspace never meets one in another, and every call names its workspace, which the caller must take from the authenticated context. Writes are not authorized by this package.

The same-workspace check is not customer isolation: queries, previews and violation lists can name any identifier of the workspace, including another customer's. No layer may expose them without the scope decision of `modules/auth`.

## What is not decided here

Organization depth; attribution priority of shared accounts; contract handling when mid-month history is missing; the calendar and time zone of an effective day; the database; retention of old revisions.

Provisional choices, conservative and each refusing rather than guessing: at most one parent per unit and one payer per account per day; payer chains of length one (this rejects the documented multi-level payment structures until the owner decides how they are modelled); at most one contract pool and one allocation pool per member per day; no nested viewing groups; member and target differ in every kind; a role (member or pool or group) that is permanent across the whole history of a kind, so that an identifier that was once a pool can become a member only by rewriting that history; civil-day granularity; the workspace-wide atomic switch without rebase; rejection of a plan for good after a validation failure; identifier-based de-duplication.

This ticket does not complete W-02: the effect on settled and unissued billing results, permissions after a move (GH04), included quantities across joining and leaving (GH06), account attribution, reconciliation with `modules/organizations`, the approval of a plan, persistence and the acceptance cases GH03 to GH06 remain NOT_RUN at system level.

# Allowed files

`modules/relationships/*` (new); `docs/features/HB-170/*`.

# Verification

Literal table tests are the primary evidence.

- Days and intervals: invalid dates, zero values, inverted and empty intervals, half-open boundaries on the exact day.
- Timelines: overlap in exclusive and non-exclusive kinds; a same-target touching entry refused and a different-target touching entry allowed; an open-ended entry followed by a later entry; a gap answered as none; an empty timeline equal to an absent one; removal followed by a role change.
- Mid-month move: `TargetOn` on the day before, the day and the day after; `Segments` over the month covers it exactly with two maximal pieces; `Segments` with an open-ended query refused and an open-ended entry clipped.
- Retroactive change: a move re-dated to an earlier day in a new revision; the old revision still answers as before; the preview reports the earliest changed day, the changed members and the affected pools.
- Release: a member leaves a group; `Union` before and after; an identifier in two groups appears once.
- Concurrent modification: two plans on the same base, the second activation fails with a conflict and changes nothing; a wrong digest; a wrong expected revision; `Seal` on a stale base leaves the plan unchanged; revision numbers without gaps after failures.
- Plan life cycle: chunk resume (identical accepted, different refused); `Seal` with a missing chunk or a stored index at or above `chunks` leaves the plan in draft and it can then be completed and sealed; a member named in two chunks refused at `AddChunk`; a chunk with a per-timeline violation refused at `AddChunk` and replaceable; `Plan` read-back in every state; `Discard` in every state; `AddChunk` after seal; a rejected plan cannot be sealed or activated; a discarded plan; an activated plan cannot be activated twice; identifier reuse after each terminal state; a plan identifier of another workspace; a no-op plan; every limit at and above its value; the violation list's order, bound and truncation flag.
- Digest: a golden vector with a literal value; the same manifest under different chunking and input order gives an equal digest; a boundary-shift pair (member `ab` with target `c`, member `a` with target `bc`) gives different digests; a different base gives a different digest; the caller mutating its slices after `AddChunk` and after `Seal` changes nothing.
- Hierarchy: direct and indirect cycles; a cycle that excludes the start member is classified as a cycle; a cycle longer than D+1 is rejected as depth; a cycle on some days only; two parents on disjoint days without a cycle; depth exactly D and D+1; depth exceeded on some days only; an ancestor with many entries.
- Payer: touching intervals allowed, an overlap of one day refused, mutual payers refused.
- Disjointness of member and target per kind; the same identifier used freely across kinds.
- Queries: revision 0, a non-existent revision, a kind that does not suit the query, `Chain`.
- Workspace isolation: identical identifiers in two workspaces; a plan or revision of one workspace is not found in another.
- Atomicity under the race detector: while plans that move many members between two pools are activated, readers call `Active()` and then read both pools at the returned number; that revision is always readable and its totals equal exactly that revision's; two goroutines activating two sealed plans on one base, exactly one succeeds.
- Properties with a fixed-seed generator and a reference model written in the test. The model owns validity (exclusivity, payer chain, cycle, depth, disjointness, each decided by brute force per day) and answers every query by scanning all entries on every day of a small universe of days, the last of which stands for all later days; every kind, identifier and day of the universe is compared after every fifth generated plan, and a random twentieth of them after the others and again for every earlier revision at the end. `Seal` accepts if and only if the model finds no violation, for generated candidates of both kinds (the model asserts the returned rule only for cycles of at most D+1 identifiers), and the test fails when a category is rarely reached. Applied in sequence, the answers of `TargetOn`, `TargetsOn`, `MembersOn`, `Segments`, `Chain` and `Union` equal the model's; every earlier revision still answers as recorded; the preview's changed members, days, affected targets and earliest day equal the model's difference.
- Scale: 100,000 timelines in chunks, sealed and activated; duration recorded.
- Mutation check by a script, results in the verification record, covering at least: each interval comparison (overlap, touch, containment of a day), the exclusive-kind test, each disjointness check, the payer overlap test, the cycle test, the depth bound, each precondition and each state transition of the plan, chunk identity, each element of the digest, the base and expected checks of activation, the copy on input and on output, the empty-timeline rule, the query kind checks, each limit, and the workspace key.
- `gofmt -l`, `go vet ./...`, `go test -race ./...`.
- Independent review of the actual change.

# Deferred

Persistence and the database; commands, HTTP and audit records with actor and reason (W-33); approval of plans; account-to-unit attribution; binding identifiers to and reconciling the hierarchy with `modules/organizations`; the effect of a new revision on permissions, waiting jobs and unissued results; a typed identifier namespace; incremental validation; revisions per pool and rebase of stale plans; multi-level payer structures; retention.

# Design review resolution

The independent design review of 2026-11-13 (read-only, nothing executed) returned FAIL (narrow) without a blocker: 4 HIGH, 12 MEDIUM, 8 LOW, all fixable by wording, and stated that no owner decision blocks implementation. It could not construct a lost update, an altered pinned answer, a chunk applied twice, a mixed-revision read or a cross-workspace read. Its corrected rules were adopted as written in the sections above.

| Finding | Resolution |
| --- | --- |
| D1 a seal precondition failure destroyed the plan | Precondition failures leave the plan in draft; only validation failures reject |
| D2 hierarchy walk termination and cost | Any repeated identifier on the path is a cycle; depth bounded by a package maximum; cost stated honestly |
| D3 a gap is not fail-closed | Restated: the package cannot tell none from unknown; the consumer rule is assigned to the later work items |
| D4 two sources of truth for the hierarchy | Authority sentence, package deviation and follow-up ticket stated |
| D5 account attribution | Deferred explicitly; the pool reading labelled as an assumption |
| D6 workspace-wide switch | Consequence stated as provisional |
| D7 digest encoding | Specified, bound to the base digest |
| D8 input aliasing | Deep copy before validation and storage |
| D9 empty timelines | Identical to absent |
| D10 plan identity and states | State machine stated |
| D11 unbounded plans | Limits listed; sealed plans store digest, summary, changed timelines and preview lists |
| D12 full copy per revision | Structural sharing |
| D13 depth off-by-one | Defined through `Chain` |
| D14 query contracts | Stated |
| D15 preview size | Paged targets; no-op plan rejected |
| D16 customer isolation | Stated as not provided |
| D17 unlabelled provisional choices | Listed, with the multi-level payer note |
| D18 to D25 | Identifier alphabet, violation order, touch, expected, GH03 wording, provenance, naming and allocation limits stated |
| Verification | Model-owned validity, boundary-day evaluation, a non-vacuous atomicity test, the missing cases and the mutation areas added |

# Design re-review resolution

The narrow re-review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker or high finding, 5 MEDIUM and 7 LOW, and judged the document implementable once the five were folded in. It found D1, D4 to D6, D8, D9, D12 and D14 to D17 resolved and the others partly resolved by the findings below, all adopted as written in the sections above: the cycle and depth classification within the stated bound (E1); a `Plan` read-back so that a lost answer can be recovered (E2); the exact digest encoding and the year range (E3); errors detectable inside one chunk refused at `AddChunk` without destroying the plan (E4); the consumer rule for gaps widened to the whole pinned revision (E5); limits and their enforcement points (E6); the remaining state transitions (E7); chunk identity after canonical ordering (E8); the storage sentence (E9); unpaged list queries stated (E10); the identifier rule duplicated (E11); validation under the lock stated as a limit (E12). The folded text was not reviewed a third time; the change review of the implementation is the check on it.

Surfaced to the owner, not blocking: payer chains of length one reject the documented multi-level payment structures, and the workspace-wide switch invalidates every other open plan on activation.
