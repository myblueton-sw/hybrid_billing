---
type: ImplementationPlan
title: Admin status projection from work results with versioned as-of snapshots, scope-first filtering, cursors, per-meaning totals and a per-subject timeline
description: Defines the first increment of work item W-30, a register that ingests stage facts per customer, contract, entity, period and currency idempotently, keeps occurrence and receipt times and superseded source revisions in a timeline, and answers list, detail and total queries from one as-of sequence, applying the viewer's customer scope before any count, keeping stale, unknown and zero apart, never adding different amount meanings or currencies, paging with cursors bound to the query and the as-of, and storing shared saved filters that always run in the viewer's own scope.
status: draft
---

# Scope and ownership

[HB-233](https://linear.app/hybrid-billing/issue/HB-233) starts work item W-30 (requirement HB-30; HB-30-I; HB-30-V) after W-29. Base: main `397083a`. Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviews the plan and the change, because the projection decides which customer data a staff member can see and count. Machine: mac_mini.

Already delivered elsewhere: AR-08 billing findings and cases (HB-217 `modules/cases`), DB-10 benefits and seller fees (HB-212, HB-216 to HB-219). Out of scope, recorded as residuals (their HB-30-V acceptances AUD-QA-03 and AUD-QA-08 stay NOT_RUN): notification incidents (QA-08), closing cycles (QA-03, also W-31), bulk commands over fixed targets with partial-success display, the U01 payment-term view (outstanding, due today, overdue and unknown by currency must come from `modules/receivables`, never from the overview's due dates), free-text search, void facts, adapters that turn module events into facts, the `modules/auth` scope adapter, persistent indexes, screen virtualisation and screens.

# Observed baseline

- HB-30-D: the home view shows stage, deadline, owner and next action per customer, contract, entity, period and currency; it never adds cost, sales and allocation; past versions, corrections and evidence are traceable; saved filters can be shared; search scope is defined.
- HB-30-I: a status projection and timeline built from work results; occurrence and receipt times are distinguished; list, detail and totals use the same as_of; filters, sorting and recent work.
- HB-30-V: over 100,000 accumulated rows, totals match details; stale, unknown and zero are distinguished; restricted customers leak no counts; server-side filters and aggregation with cursors, never a full client load or N+1 detail calls.

# Decision

A new package `modules/overview` follows the sibling conventions: injected clock, one mutex, workspace isolation, `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (`Refusal{Reason}`), `ErrLimit`; check order form, lookup, repeat, rule, room. Assistants are refused everywhere; facts are ingested by system actors only; humans and systems may read. `NewRegistry(Config{Scope, Random}, Limits{Workspaces, Subjects, Facts, Filters, Page, Stale}, clock)`, with at most 1,000,000 subjects and 10,000,000 facts per workspace; every query scans the matching workspace in memory under the mutex, which is acceptable for this increment's 100,000-row target and recorded as a known limit (persistent indexes are a residual). `Random` seeds a per-registry secret used to encrypt and authenticate as-of and cursor tokens.

## Scope

`Scope.Visible(workspace, viewer)` returns `Visibility{All, Selectors, Cost}`: a subject is visible when `All` is set or any selector matches it, a selector naming optionally a customer, an entity, a contract and a period range `From`–`To` (`YYYY-MM`, both months included; the `modules/auth` adapter converts its end-exclusive ranges; empty means any), mirroring the `modules/auth` grant dimensions; `Cost` tells whether the viewer may see cost. An error is `scope_unavailable`. Without `Cost`, the `cost` meaning is removed from rows, totals and timelines entirely (not reported as unknown). Scope is applied before any filter, count, total, sort or page, and to detail and timelines; the `modules/auth` adapter is a residual.

## Facts

`Ingest(workspace, Fact{ID, Subject{Customer, Contract, Entity, Period, Currency}, Source{ID, Revision}, Stage, Due, Owner, Next, Amounts, Evidence, Occurred}, actor)` by a system actor. All subject fields are required; period is `YYYY-MM`, currency three upper-case letters; stage is one of `collected`, `rated`, `reviewed`, `approved`, `issued`, `delivered`, `settled`, `blocked`; due is a calendar date or empty (not applicable); owner and next action are identifiers or empty; amounts map a meaning (`cost`, `sales`, `allocation`, `outstanding`) to a canonical signed decimal with at most 6 fraction digits, and an absent meaning is unknown; evidence is a list of up to 16 reference identifiers (documents, corrections, cases). `Occurred` may not be after the receipt time (`future_occurrence`). The same fact identifier with the same content returns the record; with other content it is `ErrConflict`. A source revision has exactly one fact (`revision_conflict` otherwise).

Each accepted fact gets the next workspace sequence. Source revisions are ordered workspace-wide: a source's current fact is its highest revision, wherever its subject is, so a source that corrects a misattributed subject moves out of the old subject. Lower revisions arriving late are kept in the timeline and never become current. Explicit void facts are a residual.

## Combining sources

The state of a subject at an as-of is computed only from facts accepted up to that as-of, from the current facts of the sources whose current fact is in that subject:

- stage: `blocked` while any of them reports it, otherwise the stage of the one with the latest occurrence (ties by sequence); due, owner and next action follow the same fact;
- amounts: at most one source reports a meaning for a subject at any as-of: a new fact that would become its source's current fact is refused (`meaning_owned`) when another source's current fact in the target subject reports the same meaning, and ownership ends when that source's current fact leaves the subject or stops reporting the meaning; a correction moving into an owned subject is refused until the other source moves or stops; the value comes from the reporting source's current fact, and a meaning no current fact reports is unknown;
- status: `stale` when the latest receipt among those facts is older than `Limits.Stale` before the evaluation instant, otherwise `current`.

## As-of tokens and queries

Queries never expose workspace sequence numbers: as-of and cursor tokens are encrypted and authenticated (AES-GCM under the registry secret, with a fresh nonce from `Random`; a nonce read failure is `random_unavailable`), so their content is neither readable nor forgeable, and rows and timeline entries carry no sequence numbers. `Query(workspace, Query{AsOf, Filter, FilterOwner, FilterID, Sort, Cursor, Limit}, actor)` with an empty `AsOf` evaluates the latest sequence at the clock's current time and returns an encrypted and authenticated as-of token binding the workspace, sequence and evaluation instant; passing that token later re-evaluates the same version at the same instant. Unknown or forged tokens are `ErrInvalid`. An empty workspace returns an empty page with a token.

Filters select by sets of customers, contracts, entities, periods, currencies, stages and owners, a due date upper bound (subjects without a due date are excluded from a bound) and status. `Filter` and a saved filter (`FilterOwner`, `FilterID`) are mutually exclusive. Sorts are by due date (subjects without one last), customer, period or receipt time (descending receipt time is the recent-work view), always tie-broken by the subject key. Pages hold 1 to `Limits.Page` rows.

Totals cover all matching visible rows, keyed by currency and meaning: the current amount (rows that are current and know the meaning), the stale amount (stale rows that know it), and counts of rows, stale rows and unknown rows; an amount with no known value is empty, not `0`. Signed values are summed as reported; the overview never reclassifies overpayments or computes overdue status.

A cursor is an encrypted and authenticated token binding the workspace, as-of token, viewer, a digest of the resolved visibility, the resolved filter content and the sort; any difference, including a scope change or an edited saved filter between pages, is `cursor_mismatch`. A cursor carries its own as-of.

`Detail(workspace, subject, asOf, actor)` takes the same as-of token (empty means the latest version evaluated now) and returns a token, the subject's state and its timeline up to that as-of: only facts whose own subject is this subject, in acceptance order with occurrence and receipt times, source revision, evidence and whether each was superseded at that as-of. A subject with no current fact at the as-of (every source moved out) is absent from lists and totals, and its detail shows the timeline with an empty state and status `absent`. A subject outside scope is `ErrNotFound`, like a missing one.

## Saved filters

`Save(workspace, Saved{ID, Filter, Sort, Shared}, actor)` by a human stores a filter in that human's own namespace; repeats by the owner replace it. A shared filter may not name customers, contracts, entities or owners (`shared_identifiers`), so sharing reveals no identifiers. `Filters(workspace, actor)` lists the actor's own and all shared filters. A query using a saved filter the actor may not see is `ErrNotFound`, and it always runs in the querying actor's own scope.

# Verification plan

Literal tests for every refusal and repeat; out-of-order and superseded facts; as-of queries returning earlier versions after later facts; stale, unknown and zero cases; scope with hidden customers (counts, totals, detail and cursors leak nothing); shared filters in another viewer's scope; a 100,000-subject test whose cursor pages concatenate to the full set and whose totals equal the sum of the rows; a property test against an independent model of state at random as-ofs; `-race -count=10`, a 386 build, gofmt, vet, the whole repository with `-race`, 100% coverage and mutation testing.

# Review history

- Design review round 1: FAIL (3 high, 7 medium, 4 low). Folded in: scope selectors on customer, entity, contract and period with cost visibility (1); staleness against an evaluation instant bound into signed as-of tokens (2, 6); `blocked` precedence and one owning source per amount meaning (3); workspace-wide source revisions and as-of-relative supersession (4); system-only ingestion (5); stale amounts kept apart and empty unknown totals (7); U01, search, recent work, evidence and partial success placed (8, 10); identifier-free shared filters in per-owner namespaces (9); signed cursors over the resolved visibility and filter (11); actors on reads (12); field edge cases (13); lower limits and scan cost stated (14).
- Design review round 2: FAIL (2 medium, 4 low). Folded in: meaning ownership over current facts (1); encrypted tokens and no sequence numbers in results (2); timelines hold only the subject's own facts (3); absent subjects (4); optional customer and period ranges in selectors (5); detail as-of token (6).
- Design review round 3: PASS. Folded in during implementation: tokens are described as encrypted and authenticated everywhere (L1); the ownership check applies only to facts that would become current, so late lower revisions keep their history (L2); period ranges include both months (L3). Known limit: tokens stop working when the registry restarts, because the secret lives with the registry instance.

# Change review resolution

Change review round 1: PASS with low findings, applied: as-of and cursor tokens seal fixed-width fields and a workspace digest, so their length reveals no activity (1); tests for moves into owned subjects, ownership ending when the owner moves or stops reporting (2); the property model predicts every refusal and compares owner, next action and receipt time (3); the 100,000-subject test checks uniqueness, both currency totals and an independently computed sum (4); the stray test binary was removed (5); saved filters are owned by human actors only, so a system actor with the same identifier sees none, and an all-empty selector matches nothing (6b, 6c). Recorded as a contract for the `modules/auth` adapter: `Scope.Visible` runs under the registry mutex and must not call back into the overview or block (6a). A timeline's superseded flag can show that a source moved elsewhere, without identifiers (6d, accepted by design).
