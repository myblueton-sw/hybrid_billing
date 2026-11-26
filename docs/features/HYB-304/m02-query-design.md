---
type: DesignPlan
title: M02 authentication policy — bounded reads of trust changes
description: What the M02 screen reads today, which of the specification's filters have a source, and how the list of trust changes becomes a bounded, filtered and paged read.
status: draft
---

# Scope and ownership

HYB-304 increment D of [the plan](plan.md), machine:mac_mini. Root wrote this on 2026-11-27 from the code on main `89456b80`; nothing was built for it. It covers the M02 screen "Authentication policy and trust changes". It follows the common contract of the plan and the mechanisms built for M03 ([M03 query design](m03-query-design.md)): sealed cursors bound to the query, the session, a reference time and a source version; a scan budget; an exact or lower-bound count.

# Facts

- The screen has six views (`apps/console/src/modules/identity/pages/authentication/Live.tsx:117-122`). Only "policies" reads the service. "providers" shows fixed text; "test", "keys", "sync" and "recovery" say they are not available before launch.
- The specification (`docs/ui/screens/M02/spec.md:77-83`) gives tabs for provider connections, organization sign-in policy, test sign-in, keys and certificates, recovery paths and external status synchronization; the filters Method, Workspace, Status and Certificate expiry; and the columns Connection, Method, workspace route, Diagnostics, MFA evidence, Key expiry and Active revision. Those filters and columns describe a list of provider connections.
- There is no producer of provider connections. The plan already records it: "External provider connection/certificate inventory is absent; trust-change state or local method cannot be substituted for those fields" (`plan.md:27`).
- The one read is `GET /api/v1/identity/policy` (`apps/api/server/policy.go` `listPolicy`, page `policyPage`). It answers the current policy, the trust row, the ledger row, the caller's rights, the holders summary, session times, and `changes`: every trust change the registry keeps for the workspace, newest first (`modules/trustchange/registry.go` `List`).
- The registry keeps at most 200 changes (`modules/trustchange/types.go` `MaxKept`) and prunes closed ones first. A change is open while its state is `draft`, `validated` or `independently_approved`; the other states are `activated`, `rejected`, `cancelled`, `superseded` and `activation_unknown`.
- Each change is a full row (`apps/api/server/trustrows.go` `changeRow`): identifiers, digests, the candidate revision, the four actors, validation with its items, impact, activation, job, invalidation progress, what the caller can do and why not, and the whole history.
- Building a row for an **open** change calls `trustFacts` (`trustrows.go` `row`), which loads the whole roster, every user and every grant (`trustevidence.go` `trustFacts` through `accessFacts`). The page calls it once more for the holders summary. So one read of the page costs one whole-population load for the holders and one more per open change.
- The screen takes from `changes`: the selected change by identifier (`Live.tsx:136`), the first change in state `activation_unknown` for a notice (`TrustChanges.tsx:30`), whether any change is from an earlier run (`:105`), and the change a superseded one names (`:369`). It has no filter and no paging; it lists them all.
- The screen groups the list in the browser (`trust-model.ts:110-114`, `grouped`): open and closed, each sorted by state priority, then number, then identifier. The console's open group is `draft`, `validated`, `independently_approved` **and `activation_unknown`** (`trust-model.ts:7`); the module's `Open` (`modules/trustchange/types.go:160-162`) does not include `activation_unknown`. A change's number is the counter of one server run and repeats across runs (`registry.go:155-156`), which is why the identifier is the last tie-breaker.
- A list row shows more than identity and state: whether validation was blocked (`validation.result`), whether its evidence is stale (`validation.evidence_current`) and the progress of invalidation (`invalidation.state`) (`trust-model.ts:116-137`; `TrustChanges.tsx:135-139`, `:170`, `:177`).
- What the caller can do splits by cost (`trustrows.go:152-226`): `can.revise`, `validate`, `reject` and `cancel` need only the state, the rights, the actors and whether trust is unavailable; `can.approve`, `can.activate`, their last reason `evidence_stale`, `evidence_current` and `impact` need the whole-population facts for that change's candidate. At most 100 changes can be open (`MaxOpenPerWorkspace`), so one read of the page can cost up to 100 such loads, and one more for the holders.
- Pruning also removes changes in `activation_unknown`, which the console counts as open; a waiting activation can therefore be pruned once more than 200 changes are kept, and the additive field that names one can lose it that way.
- The registry has no version. Every mutation goes through `record` (`registry.go:95`), `prune` (`:162`) or `Restore` (`:384-388`).

# What has a source and what does not

| Specification item | Source | Outcome |
| --- | --- | --- |
| The list of trust changes of the installation's sign-in policy | the trust change registry | bounded, filtered and paged in this increment |
| Filter Status | a change's state | offered on the changes list |
| Search | a change's identifier | offered on the changes list |
| Provider connections: the filters Method, Workspace, Certificate expiry and the connection columns | none | not built |
| Keys and certificates | none | not built |
| Recovery paths | none | not built |
| Test sign-in | none | not built |
| External status synchronization | none | not built |

The authentication policy module only names the sign-in methods; the server reports the external ones as not connected; no adapter, connection, certificate or synchronization inventory exists in `modules/` or `apps/api`. None of the five areas is imitated from the local policy's fields.

**Increment D therefore delivers only the trust-change list.** The plan requires a real inventory source and contract before those filters and says the parent's completion "remains unproven while applicable filters lack a real source" (`plan.md:27`, `:53`). M02 and the HYB-304 parent are not complete by this increment; the five areas are recorded as a dependency in a ticket of their own, and whether HYB-304 may close with that dependency recorded or must stay open is put to the owner. This matches the owner's direction of 2026-11-17 to build what the planning says and to block truthfully when a dependency is missing.

# Design

**Reads.**

| Read | Lists | Filters | Order |
| --- | --- | --- | --- |
| `GET /identity/policy/changes` | the workspace's kept trust changes | `state`: one of the eight states, or a group; `q` over the identifier; `limit`; `after` | newest first, the registry's order |
| `GET /identity/policy/changes/{id}` | one change by identifier | none | none |

The groups are the screen's: `open` is `draft`, `validated`, `independently_approved` and `activation_unknown`; `closed` is `activated`, `rejected`, `cancelled` and `superseded`. This differs from `trustchange.Open` on purpose and is named so in the API description. Search is over the identifier only, since numbers repeat across runs.

Order is creation order, newest first: it never changes when a change moves state, and pruning only deletes, so a cursor is stable; a cursor whose anchor was pruned answers 409. The screen's sort by state priority cannot be kept across pages. Each group is read on its own; a group that fits one page (the open group holds at most 100 changes awaiting a decision, by `MaxOpenPerWorkspace`, plus any in `activation_unknown`, which that limit does not count) can still be sorted by state priority in the browser as today, and what a group of more than one page shows is the ui-designer's composition decision, recorded with the composition.

Admission is the policy read's. An unknown identifier and one of another workspace answer alike (`Get(id, workspace)`, `registry.go:353-361`). A page carries `rows`, `next`, `count` with its certainty, `reference_at`, `observed_at` and the echoed query, as the M03 lists do.

**Rows.** Both reads build rows through the same `rowContext` the page uses today; there is no second copy of the rules.

- A list row carries: identifier, number, state, change revision, the four actors, the candidate's summary, `earlier_run`, `superseded_by`, `validation_result`, `invalidation_state`, and for the caller `can.revise`, `can.validate`, `can.reject`, `can.cancel` exactly. `why_not.approve` and `why_not.activate` are computed by `whyApprove` and `whyActivate` with evidence currency not evaluated, so every reason before `evidence_stale` is exact. For an open change `evidence_current` is `not_evaluated` and `can.approve` and `can.activate` are not answered; the row says they are answered in the detail.
- The detail read answers the full row, including evidence currency, `impact`, `can.approve` and `can.activate`, at the cost of one whole-population load for that one change.
- A superseded change's successor is resolved through the detail read by identifier, not from the current page.
- What the caller may do and the progress of invalidation are evaluated when the page is read, as the M03 lists do; the command decides.

**The whole policy read.** It keeps what is not a list: the policy, the trust row, the ledger, rights, holders and times. Two facts the screen takes from the whole list move to it as additive fields: the identifier of a change in state `activation_unknown`, if there is one, and whether any kept change is from an earlier run. Its `changes` field is removed only after the screen no longer reads it, and removing it is put to the owner first, as with M03.

**Module.** The trust change registry gains a version advanced in `record`, `prune` and `Restore`, which covers every mutation; a page read newest first with a limit and a continuation; and the two summary facts above. A read by identifier exists (`Get`). The cursor's basis is that version.

**Screen.** The policies view lists changes from the bounded read with a state filter and paging, opens a change from the detail read, and takes the notice and the "earlier run" note from the whole read's new fields. Composition is the ui-designer's.

# Increments

1. **M2a.** Trust change registry: version, page, summary facts; tests against `List` and `Get`.
2. **M2b.** The two reads with cursors, filters and counts; the two additive fields on the policy read; API description.
3. **M2c.** The console's strict client for the two reads and the new fields.
4. **M2d.** ui-designer composition; the screen on the bounded reads; rendered check.
5. **M2e.** After the screen reads no list from the policy read: ask the owner about removing `changes`, then remove it and bound the remaining whole-population loads of the policy read (the holders summary).

Each increment has an independent change review; M2b and M2e change a public read.

# Progress

- **M2a** (2026-11-27): `modules/trustchange/pages.go` adds `Version`, `Page(Query)` and `Summary`; the registry's version advances in `record`, when `prune` drops a change and for each change `Restore` takes. A page is newest first, filtered by workspace, a set of states and a substring of the identifier, continued after an identifier, at most `MaxPage` (100) rows; it states the exact number of matches, whether more follow, and whether its anchor is still kept. The module walks its kept changes (at most 200 closed plus the open ones) without copying and copies only the rows it returns, so the count is exact. Nothing reads these yet. Verified: the module's tests pass, among them pages against `List` for five state filters, four searches and four limits; 19 mutants of the new code, 18 killed and one equivalent (a `continue` at the end of the loop body). The state groups `open` and `closed` are the server's to define in M2b; the module takes a set of states. `Known` is false for a pruned anchor, an anchor of another workspace (answered exactly as an unknown one) and a limit outside 1 to 100. The whole server package passes with the changed module (442 s). Independent change review (by reading; the reviewer ran nothing): approve with conditions, no defect in the code, three test gaps; each is closed and confirmed by the wrong implementation the reviewer described failing the tests (a total counted only after the anchor; an anchor honoured only when the filter admits it; only the first of several states honoured). Carried to M2b from the review: bind the cursor to the registry's run as well as its version, since the version restarts with the server, and do not expose the raw number; bound the search's length and validate state names before calling; never pass an empty workspace; reads wait while an activation holds the registry during its commit. Existing and not changed here: `apps/api/server/trustledger.go` discards the error of `Restore`, which can leave restored changes unpruned.
- **M2b** (2026-11-27): `apps/api/server/trustlists.go` adds `GET /identity/policy/changes` and `GET /identity/policy/changes/{id}`; the policy read gains `waiting_activation` and `earlier_run_kept`. The list reuses the access lists' query parsing and sealed cursor (`openList` now takes the source of its basis); its basis is the registry's run and version, its count is exact on every page, and its limit is the service page limit capped at 100. A list row is built by `rowContext.listed`, which takes what the caller may do from the same `actions` the full row uses, with evidence currency assumed, and drops `can.approve` and `can.activate`; `decide_in_detail` is true for a change in `draft`, `validated` or `independently_approved`. The list makes no whole-population load. The detail read answers the full row through the unchanged `row`. An unknown identifier, a malformed one and one of another workspace answer `invalid_request` alike. Unlike the policy read, both reads refuse a session whose workspace is not the installation's. Verified by tests that compare every listed row and every detail with the policy read's `changes` for three administrators, every state and group filter, paging by one, the search, the refusals and the cursor's bindings; 19 mutants, 16 killed, 3 not: two equivalent (`cancel` and `revise` have one rule; the module's unknown-anchor answer cannot be reached because a pruned anchor already changes the basis) and one that showed the access lists' lower-bound count had no test, now added and killed. The whole server package passes (427 s; run before the review's test additions, which were then run with the trust, description and source-rule tests). Independent change review (by reading; the reviewer ran nothing): approve with one condition, no code defect: the first test compared rows for seven of the eight states only. A row and a detail in `activation_unknown` on a held workspace and an earlier run's row are now compared with the policy read as well. Known and not changed: a session of another workspace gets `unauthenticated` from the list and `invalid_request` from the detail, and whether such a session can exist was not verified; `scan_stopped` is true on a full page of 100 when the scan limit is 100; `work_units` counts returned rows. Not tested: an expired cursor on this list; an access-list cursor replayed on this list (the reverse is); the two description entries.
- **M2c** (2026-11-27): `apps/console/src/service/trust-changes.ts` adds `trustChanges(query)` and `trustChange(id)` to the admin client, and the policy page type gains `waiting_activation` and `earlier_run_kept`. The list parser is closed: unknown or missing fields, value sets, and the relations the service keeps (a change awaiting a decision is left to the detail; evidence is `not_evaluated` only on such a change with a validation; an ended change allows nothing; a successor only on a superseded change; invalidation only on an activated one; an exact count; a continuation only after a full page) are checked, and a page is refused when its echoed query, its rows' states or its rows' identifiers do not fit what was asked. The detail read checks its envelope, the identifier and the state, and parses the row with the parser the screen already uses for the policy read's `changes`. No screen calls them yet. The tests' fixtures (`trust-changes.sample.ts`) are answers captured from the server's own tests (a first page of three, the open group with a search, a held workspace with a waiting activation, a detail), with the cursor and the revision replaced by placeholders. Verified: TypeScript clean; 1487 console tests, 1482 passed, 5 skipped; 31 mutants of the client's checks, all killed after four cases that had been refused for another reason than they named were corrected. Independent change review (by reading; the reviewer ran nothing): approve with two conditions, no code defect. A search with characters outside ASCII is lower-cased differently by the service and the browser, so for such a search the client now checks only that the echoed search is not empty; an inert linter directive was removed. On the review's advice the fixtures gained a captured answer of the closed group (an activated change with its invalidation, a superseded one with its successor, a rejected, a cancelled and an earlier run's change with number 0), the refusals assert the parser's own error, and six checks that had no case have one; nine further mutants killed; one of the nine was confirmed only after the merge of M2c, because the case for a search outside ASCII had used the browser's lower-casing as the service's answer and so could not tell the two apart (corrected in a follow-up change the same day). Found while preparing M2d and not built here: the screen's list shows what a change is based on (`old_epoch`, `old_policy`), which the list row does not carry; it is added to the row in the next increment.
- **M2b-2** (2026-11-27): the list row gains `old_epoch` and `old_policy`, what a change is based on, which the screen's list shows in its "Based on" column and the first row design left out. Both are the change's own facts and need no whole-population load. The console's parser requires them and checks that an epoch's value is its generation and number combined. The fixtures were captured again from the server's tests. Verified: the server's comparison of every listed row with the policy read now includes the two fields, and removing them from the row fails it; console TypeScript clean, 1482 tests passed; four mutants of the new parser checks killed. The whole server package passes (406 s). Independent change review (by reading; the reviewer ran nothing): pass, no blocking finding; it confirmed that the list cells and their helpers read no fact the row lacks, three of them under another name or shape (`validation_result`, `invalidation_state`, and evidence currency, which the list does not have), which M2d adapts. On its advice an epoch of a later generation has an accepted and a refused case; a wrong multiplier is killed.

# Risk

- Trust changes govern who can sign in and how. The reads are administration reads and decide nothing, but `can` and `why_not` tell an administrator what they may do; they must come from the same code the commands use, not from a second copy.
- With at most 200 kept changes the list is small today. The bound is still required by the plan, and the larger cost is the whole-population load per open change, which the split between list and detail removes.

# Follow-up

- **The five areas without a source.** HYB-374 records that provider connections, keys and certificates, recovery paths, test sign-in and external status synchronization have no producer, that M02's specified filters and columns belong to them, and that HYB-304 does not build them.

# Reviewed

Independent design review (role-based AI reviewer separate from the writer, by reading, nothing built), 2026-11-27: approve with conditions. Three blocking findings, taken into this revision and resolved on a bounded re-review, which also corrected a false statement about the size of the open group: the first draft missed how the screen groups, orders and fills its list rows, so its rows would have dropped displayed facts and its `open` group would have put waiting activations among the closed; the split of what a list row can answer without the whole-population facts was already decidable from the code and is now stated; and the plan requires more than a follow-up ticket before M02 is called done, so all five areas without a source are named and completion is stated as unproven. The reviewer confirmed the facts about the views, the one read, the registry's order, cap and pruning, and that the two additive fields suffice for the notice and the note.

# Unknown

- Whether the holders summary can be answered without the whole roster (read in M2e).
- Whether the facts behind evidence currency depend on the candidate; if they do not, one load could serve every open change (the reviewer did not verify it).
- The order `Restore` gives restored changes relative to those created in the current run.
