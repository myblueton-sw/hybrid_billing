---
type: VerificationRecord
title: HYB-323 M01 select everyone matching a filter, implementation verification
description: What HYB-323 built against the approved plan and the binding visual addendum, the checks run with their observed results, the scale measurements at 5,000, 20,000 and 100,000, browser QA in EN, KO and AR with every new check shown failing first, regressions, mutation, deviations, [TW] strings and residuals. Written by the implementer; no change, render or technical-writing review has seen this build.
status: stable
tags: [HYB-323, HYB-294, verification, M01, bulk-revocation, filter-wide-selection, IAM-07, UI-04]
sources:
  - resource: plan.md
  - resource: visual-spec.md
  - resource: ../HYB-294/verification.md
  - resource: ../HYB-299/verification.md
---

# Scope of this record

- Branch `feat/HYB-323-m01-select-matching`, worktree `/private/tmp/hb323`, built on main `3bdb750` and rebased by root onto `113c0d5` (HYB-309; conflicts in `cmd/hb-api/main.go` and `main_test.go` resolved by root). Nothing is committed. Results marked "round 2" were observed on the rebased tree; earlier numbers (scale, first QA pass) were observed on `3bdb750` and the rebase brought no change to the measured code.
- Everything below was observed on 2026-11-19 on the Mac mini (darwin/arm64). The host runs unrelated workloads; load averages during the runs are noted where timings are given. A check that did not run says so.
- Scratchpad (`/private/tmp/claude-501/-Users-home-project-hybrid-billing/ff7545a1-604d-4178-88ba-25b4b88aab02/scratchpad`, below `$SP`) holds the runners, outputs and screenshots; nothing from it is in the worktree.

# Round 2: review fixes (2026-11-19, on the rebased tree)

Change review FAIL narrowly (test evidence); render and copy review FAIL on copy. Applied per root's `hb323-fixes.md` and the exact strings in `hb323-render.md`:

- **C1.** New test `TestPeopleFailureInTheLoginSegmentFailsTheWholeWalk` (seed world, no departed people; the `People` fault on every call and on the first call only): the count read and `add_matching` answer `dependency_unavailable` and no draft is created. Mutant 212 (`userwalk.go:62`) is killed by it (see Mutation).
- **C2.** Memory figures corrected (see Scale).
- **Q-B6 (root decision).** Default `-bulk-revocation-retained-people-max` = 1,100,000; `settings.go`, the refusal-message and usage-text tests, `users_scale_test.go` (its own world now sets the cap it needs), plan Q-B6 and section 4.
- **C3.** When the list was counted as a lower bound and fewer people are fixed, the fixed line says "{m} people fixed at {time} (at least {n} when counted)" [TW] (KO "…(집계 당시 최소 {n}명)", AR "…(كان العدد {n} على الأقل عند العدّ)"); a larger fixed number after a lower bound is not called a difference. `match-model.ts` `countNote`, `Matching.tsx`, recovery keeps the flag. The unit and render tests were shown failing before the change (`$SP/hyb323-c3-before.out`: 2 failing tests).
- **C4.** Accepted, no change.
- **Copy fixes 1–6** exactly as given: plural-aware `noLongerMatch`, label-style `arithmetic`, `checking`; KO 버리기 wording (discardList, discarded, fixedNoChecks, listClosed); AR «القوائم بانتظار التنفيذ» in the setting name and `pendingLimit` (R-C4); Base revision KO "기준 개정 번호", AR «رقم التعديل الأساسي»; setting hints built from `fc.searchedUpTo` and `fc.continueSearching`; KO countedAt and lowerBound. Glossary updated.
- **L1.** `.m01-selsum .m01-fixedmeta + p { margin-block-start: 8px }`.
- **L2.** Accepted as deviation V-9: the fixed line always shows the date.
- **L3.** The search chip uses the Part A `fc.chip` template inside one span (no flex gap between label and value).
- **L4.** Removable chips computed 16px (the shell rule `#content [data-remaining-screen] :is(…button…)` outranked `.m01-chip`); `match.css` sets 14px with a selector that outranks it. Shown failing first: QA phase K on the pre-fix build, 8/9 with the L4 check failing; 9/9 after.
- **QA runner.** `.m01-settings td.m01-num` is exempt from the repeated-text check (Value and Default may legitimately be equal); the M harness reads the count instead of the scan-limited list; new phase K (§12 item 17: Tab order in states E and F, Enter keeps focus on the trigger, Escape closes and focuses the trigger, focus after fixing, after a prepare refusal and after discarding; L4 font size); new phase E (§12 item 10: the explicit-list preview HTML, digits and list numbers normalised, equals the snapshot from a `113c0d5` server and console, with a self-test that a changed preview is flagged).
- **Shown failing first.** Against `113c0d5`: K stops at its first check (no trigger), Z 0/15 (`$SP/qa323b2-*.out`); E's comparison is checked by its self-test.

## Round 2 results

| Check | Result |
| --- | --- |
| gofmt; vet darwin, linux, windows; linux/386 and windows/amd64 builds | clean; pass; pass |
| `go test -short -count=1 ./...` | 73 packages `ok` |
| `-race -short` on `apps/api/server`, `modules/bulkrevoke`, `cmd/hb-api` | all `ok` |
| Coverage | server 100.0%, bulkrevoke 100.0% |
| Console | tsc clean; `npm test` 786/786; build passes |
| C1 mutant (`userwalk.go:62`, `if false &&`) | killed |
| QA (fresh servers) | S 22/22; X 11/11; M 1/1 (24 members); F 684/696; P 72/72; G 95/101; K 9/9; Z 15/15; E 2/2 (fresh server, after a 65 s pause so the command burst of the harness has passed); T 30/30; L 27/27 (S 23/23) |
| Regressions (round 2) | `qa299` copy 784/784; `qa294` 22, 27, 1497, 70 (2 page errors: the aborted request), 33; `qa264on294` 52/52; M02 all groups 0 failed; M03 157/157; HYB-293 134/134; P01–P05 828/828 |

F and G failures are only the inherited classes, never counted as passed: F 12 × Membership-tab "No membership" repetition (HYB-331); G 4 × job-row/item text escapes and 2 × AR 390 jobs table (HYB-332). The settings repeated-number failures are gone with the exemption.

# What was built

## modules/bulkrevoke

- `types.go`: selections `filter` and `not_recorded`, `FilterTerm`, `CanonicalFilter` (`len:key len:value` per term), View fields `Filter`, `Matched`, `FixedAt`, `FixedRevision`, `Limits.MaxMatching`, exclusion reason `no_longer_matches`.
- `registry.go`: `AddMatching` (validation, `ErrEmpty`, `ErrTooMany`, open draft, pending limit), `DraftView`; `Add` on a filter draft answers `ErrState`; `List` builds summaries without copying items, targets and exclusions; `Restore` and `prune` keep the index (below).
- `manifest.go`: the digest takes the draft's own selection; for filter drafts it adds the canonical filter, matched count, fixed time and base revision (`ListDigest`); `Digest` for explicit lists is unchanged.
- `latestindex.go`: incremental latest-outcome index (`refs[workspace][user]`, newest number first, first item of the highest-numbered job); `Execute` indexes the new job, eviction un-indexes by binary search, `Restore` replaces an entry's refs.
- `job.go`: `Span` copies only a task's items; `Cleaned` and `Failed` keep `Counts` incrementally instead of recounting every item.

## apps/api/server

- `bulkstage.go`: `bulk_revocation.add_matching` (enum filter fields from `userFilterValues`, `q` and `method` as text), shared parser, `prerequisite_failed` refusals, answer fields, log line `bulk_revocation_matched` with the filter keys only.
- `userwalk.go` (new): one walker over the two segments for the list read, the count read and `add_matching`.
- `userfilter.go`: `parseUsersQuery` also returns the refused parameter's name (sorted keys); terms and filters convert both ways.
- `userscount.go` (new) and the route in `boundary.go`: `GET /api/v1/identity/users/count`.
- `bulk.go`: prepare re-checks a filter draft (`stillMatches`; nil member for non-members); nothing left on a filter list answers `invalid_stage`.
- `bulkimpact.go`: `no_longer_matches` between `not_found` and `no_membership`; per-prepare side indexes for sessions and pickups; proposal stores enumerate once and tally by proposer.
- `bulkread.go`: row fields `selection`, `matched`, `fixed_at`, `fixed_revision`, `filter`, `search_masked`; the detail adds `exclusions.no_longer_matches` for filter lists only.
- `bulkcleanup.go`, `session.go`, `directory.go`: task limit from `max_matching`, `Span` per task, per-task session and pickup lookups.
- `identityledger.go`: replayed jobs report `not_recorded`.
- `settings.go`, `server.go`, `cmd/hb-api/main.go`: `bulk_revocation_max_matching` (5,000, 1–100,000, at least `max_targets`), `user_count_scan` (20,000, 1,000–100,000), `bulk_revocation_retained_people_max` (1,100,000, 10,000–20,000,000) with the start check `(2 × jobs_kept + pending) × max_matching`; the refusal and the flag usage name the cap and all three inputs; flags accept up to eight digits; the users and bulk reads return the users settings rows too.
- `openapi.go`: `x-reads` for the count read and the bulk reads (selection values, the replay change, the masking note); `trusthold.go` row.

## apps/console

- New `Matching.tsx` (trigger and count region, fixed bar, read-only chips, basis line and facts, fixed facts in the preview), `match-model.ts`, `match-copy.ts`, `match.css`.
- `Selection.tsx` (action after the actions, R-C8 sentence, `filterWide` removed), `Live.tsx` (fixed list state kept per user and recovered from `bulk-revocations?limit=1`, no checkbox column while fixed, discard), `BulkPrepare.tsx` (filter mode: one prepare call, fixed summary, anchor dead-end, list closed, fixed facts and the five-reason exclusion summary), `RevocationJobs.tsx` (basis line, "Fixed {at}", basis facts in the detail), `RolePeriod.tsx` (setting hints), `bulk-copy.ts` (new setting names and reason; `filterWide` deleted), `service/identity-bulk.ts` (row fields, `count`, `countPath`, `matchValues`), `service/admin.ts` (command ID, exported `filterOf`).

## Documents

- `docs/ui/terminology/product-glossary.md`: section "M01 select everyone matching (HYB-323, HYB-294 part B)".
- `docs/ui/screen-binding-register.md` (M01 row, reads, commands, residual line) and `docs/ui/screen-binding-fields/identity.md` (new subsection; `jobs[].selection` removed from the unread list).
- `plan.md`: round 3 note folded in, D-7 amended per the addendum, status `stable` (root instruction).

# Checks and observed results

| Check | Command | Result |
| --- | --- | --- |
| Format, vet | `gofmt -l apps modules`; `go vet ./...` on darwin, `GOOS=linux`, `GOOS=windows` | no files listed; all three pass |
| Cross builds | `GOOS=linux GOARCH=386 go build ./...`; `GOOS=windows GOARCH=amd64 go build ./...` | both pass |
| Go tests | `go test -short -count=1 ./...` | 73 packages `ok`, none failing; the two scale tests pass in their own non-short runs |
| Coverage | `go test -short -cover` | `apps/api/server` 100.0%, `modules/bulkrevoke` 100.0%, `apps/api/cmd/hb-api` 92.9% (unchanged from `3bdb750`) |
| Race | `go test -race -short -count=40` on the new bulkrevoke tests; `-race -short -count=10` on the new server tests | both `ok` (19.4 s, 70.7 s) |
| Console | `npx tsc --noEmit`, `npm test`, `npm run build` | tsc clean; 778/778; build passes |
| Source rules | bulkrevoke and server `TestSourceRules` (no comments, no binary-fraction type in Go tests, under 1,500 lines) | pass; largest changed source `admin.ts` 1,153 lines, `Live.tsx` 855 |

## New tests

- `modules/bulkrevoke`: `matching_test.go` (every `AddMatching` refusal including a 64-byte value and a 128-byte revision; filter digest moves with terms, matched, time and revision; an explicit digest is refused for a filter list; the explicit digest pinned to `d2166a50…0dedd`, the value `3bdb750` produces, and equal to a test-local copy of the previous algorithm over 300 random inputs; `CanonicalFilter` injective over 20,000 random term lists; list summaries equal the full views; `Span`), `index_property_test.go` (60 seeds × 120 random steps of prepare, execute in any order, finish, restore with changed outcomes and with new numbers, discard, expiry; `LatestFor` equals the full-scan oracle after every step; work count equals the new job's items plus the evicted job's items; first item of a person in a job), `property_test.go` (incremental `Counts` equal `count(v)` after every step), `scale_test.go` (index timing).
- `apps/api/server`: `bulk_matching_test.go` (fixed set equals the list's set for 11 filters; refusals with field names; limits exactly at and one over; recheck with drift and a member who changed but still matches; classification order including `suspend_user` without membership; anchor stranding; masking for a bulk-right holder and an administrator without it on rows, detail and items; log keys only; roster rows equal People rows for every key and value including non-members, removed, suspended, ended and departed; settings and refusal text), `bulk_property_test.go` (independent model: 4 populations up to 300, 40 filters; fixed set, count answer under random budgets and limits, every exclusion reason after random drift, and the digest recomputed independently), `users_count_test.go` (exact, lower bound, over limit, the tie at limit + 1, refusals without fields, faults, concurrency under a held read lock, the document), `bulk_scaling_test.go` (cleanup of 10,001 items, impact indexes equal the per-person scans on 40 random worlds, cleanup ends the same sessions and pickups as before), `bulk_scale_http_test.go` (scale), `identityledger_test.go` (replayed rows `not_recorded`).
- Console: `match-model.test.ts`, `match-render.test.tsx` (EN, KO, AR), `identity-bulk.test.ts` additions; the Part A render test now asserts the `filterWide` note is gone.

## Shown failing first

- Cleanup of 10,001 items on `3bdb750` (same synthetic job, scratch test in a detached base worktree): the cleanup job is not created, job state `waiting` (`$SP/hyb323-base-cleanup.out`).
- Browser checks: phase Z (15 probes, one per new behaviour) against a `3bdb750` server and console: 0/15; against this build: 15/15 (`$SP/qa323-Z-4523.out`, `-4522.out`). Phases X, F, P and G against `3bdb750` stop at their first Part B step (no trigger: `locator.click`/`innerText` timeouts, `$SP/qa323base-prove-*.out`). The self-test phase X breaks each layout and content property on a live fixed bar and confirms the probe flags it: 11/11.
- Mutation (below) killed the mutants in the walker's stop rules and the classification order, which is what the property test was required to catch.

# Scale measurements (recorded numbers, not thresholds)

`TestScaleFilterListsOverHTTP` (`$SP/hyb323-scale2.out`), 100,000 people with memberships built through the identity API in 1 m 29 s, side data 9,900 sessions and 1,001 pickups plus pending relationship plans and release evidence for 1% (two of the eight proposal stores; the others are counted by the same single pass). Load average 7.6 at start, 6.0 at the end.

| | 5,000 | 20,000 | 100,000 |
| --- | --- | --- | --- |
| `add_matching` (HTTP) | 113 ms | 119 ms | 108 ms |
| prepare (HTTP) | 117 ms | 208 ms | 628 ms |
| execute (HTTP, to the answer) | 216 ms | 590 ms | 2.29 s |
| phase A (deny to first ledger sync) | 18 ms | 47 ms | 102 ms |
| phase B (ledger header and items) | 115 ms, 21 syncs | 418 ms, 80 syncs | 2.05 s, 392 syncs |
| users read issued during execute answered after | 196 ms | 571 ms | 2.27 s |
| ledger bytes per item | 533 | 535 | 536 |
| cleanup tasks, drain, longest lock hold per task | 50, 69 ms, 7.7 ms | 200, 232 ms, 6.6 ms | 1,000, 711 ms, 4.7 ms |
| jobs list read with the list kept | 0.71 ms | 4.5 ms | 42.9 ms |
| job detail read | 0.60 ms | 2.9 ms | 25.2 ms |
| users first page | 0.29 ms | 0.31 ms | 0.30 ms |

Count read over everyone: 14.6 ms with `user_count_scan` 20,000 and 80.9 ms with 100,000; a write command started during the count answered after 16.2 ms and 88.7 ms (it waits for the read lock).

Memory (Go heap after two collections): draft ID 34–45 bytes; open prepared person 163–175; ended (discarded) prepared person 163; retained executed item after cleanup 1,481–1,542.
- The start check `(2 × jobs_kept + pending) × max_matching ≤ cap` bounds executed items to `jobs_kept × max_matching ≤ cap / 2`; the other half covers ended lists and lists waiting to run at about 163–175 bytes per person (corrected after the change review, C2: the first version of this record multiplied the whole cap by the executed-item size).
- Worst case ≈ cap/2 × 1,542 + cap/2 × 175: about 870 MB at 1,100,000 and about 1.7 GB at 2,000,000.
- **Q-B6, root decision:** the default is 1,100,000, the smallest value that keeps the shipped defaults (100 kept, 20 pending, 5,000) starting. Settings, the refusal test, the usage text and plan Q-B6 carry it.

Cleanup before and after (synthetic job, no sessions; `$SP/hyb323-drain.out`): `3bdb750` 5,000 items 293 ms with 12.4 ms per task, 10,000 items 1.08 s with 21.6 ms per task (quadratic recount); above 10,000 the job is not created. This build: 3.5 ms and 6.3 ms.

Latest-outcome index (`modules/bulkrevoke` `TestScaleIndexPerExecute`, `$SP/hyb323-index-*.out`):

| | `3bdb750` (rebuild) | this build (incremental) |
| --- | --- | --- |
| 100 executes of 5,000, up to 100 kept | 2.92 s in total | 0.24 s |
| one execute of 5,000 with 100 kept jobs | 42.1 ms | 3.1 ms |
| execute of 100,000 with 100 kept jobs | 49.4 ms, 33.0 ms (third run not reached: finishing its 100,000 items with the per-item recount exceeded the 600 s test timeout) | 18.3 ms, 15.3 ms, 25.9 ms |

# Browser QA

Runner `$SP/qa323.mjs` (helpers copied from `qa294.mjs`), live `hb-api` built from this tree, `-user-filter-scan 100`, the sample seed, a harness of 300 people (10 suspended, 10 departed), 24 people who accepted their invitations (phase M), Chromium headless shell. Each phase name, servers and results:

| Phase | What | Result |
| --- | --- | --- |
| S | harness | 22/22 |
| M | 24 members through invitation links | members created; its own count check failed (0/1) on the first run because it used the paged read limited by the scan budget; P and G then fixed lists of those members |
| X | self-test on a live fixed bar | 11/11 |
| F | EN, KO, AR × 1440, 1280, 1024, 390: trigger labels, counting without digits, exact label, layout and style sweeps, Escape, fixing, focus, no checkbox column, fixed bar facts, filter change keeps the list, prepare summary, discard and focus | 684/696; the 12 failures are below |
| P | EN, KO, AR × 1440, 390: drift between fixing and preparing ("1 no longer match"), fixed facts, the exclusion row | 72/72 |
| G | functional EN 1440 and others | 94/101; the 7 failures are below |
| T | `-bulk-revocation-max-matching 2`: too many shown before trying without a number; none match | 30/30 |
| L | about 1,100 people (300 with pending invitations, 800 more whose invitations were revoked to stay under the 1,000 pending-invitation cap), `-user-count-scan 1000`: the lower-bound line names the scan and the limit in EN, KO and AR, with layout checks | S 23/23, L 27/27 (two earlier attempts were invalid: one stopped at the pending-invitation cap, one ran without `-user-count-scan 1000`) |
| Z | proof probes | 15/15 (base 0/15) |

G covers: no trigger on Invitations and Revocation jobs and without the bulk right; replace confirmation with `aria-describedby`; Tab order trigger → confirm → Cancel; a person created between counting and fixing gives "({n} when counted)"; picks replaced; the fixed list survives a tab change and a reload (recovered from the server); the discard announcement; the open-draft refusal named and focused; list closed with `suspend_user` on suspended people; discard from the refusal returns to the trigger with the filters; the anchor dead-end in EN, KO and AR with the rendered role label and no deselect; execution; the job row basis for the preparer and "Search applied" for another administrator with the text absent from the page source; the settings rows and hint; seven-digit values whole at 390.

(First pass on `3bdb750`; round-2 results are in "Round 2 results" above.) Failures, all inherited or check limitations, shown on `3bdb750` with the probe phase B (`$SP/qa323base-B.out`, 28/33 on base, the same classes):
1. F, 12 × "(a) repeated text block" on the Membership tab: rows without a membership say "No membership" in two columns. Present on `3bdb750` (`baseprobe-membership`).
2. G, 4 × "text escapes" on job rows (`span.m01-opener`, the "Open" tag) and job items (`li.m01-item`). Present on `3bdb750`.
3. G, AR 390 jobs table scroll and header word breaks. Present on `3bdb750`.
4. G, AR 390 settings "(a) repeated text": value and default columns hold the same number (5000, 2000000, 20000). The check ignores texts under four characters, so the older three-digit rows never tripped it; these are two separate facts. Not changed.

# Regressions

Each on its own ports against this build (`hb-api323`, this console):

| Suite | Result |
| --- | --- |
| M01 `qa299` copy (the `filterWide` check replaced by "no note, action offered") A, C, B, D, E, L, P, N | 334, 3, 5, 5, 53, 352, 1, 31: 784/784 |
| M01 Part A `qa294` S, X, F, G, W | 22/22, 27/27, 1497/1497, 70/70 (2 page errors: the deliberately aborted next-page request), 33/33 |
| M01 `qa264on294` | 52/52 |
| M02 `qa297` A, B, C, D, E, I, K, L, F1, F, G, H | 17, 3, 3, 3, 2, 3, 130, 21, 2, 2, 3, 2, 0 failed |
| M03 `qa298on297` A–E | 67, 21, 19, 30, 20, 0 failed |
| HYB-293 `qa293on297` B, C, D, A | 32, 27, 54, 21, 0 failed |
| P01–P05 `qa315bodyon297-full` A, B, C1, C2, C3, F, P, R (sample seed) and E (`--seed none`) | 17, 9, 2, 3, 30, 132, 12, 481 and 142, 0 failed |

# Mutation

Driver `$SP/mutate323.py`: every changed or new line (against `3bdb750`) of the five bulkrevoke files, sixteen server files and `cmd/hb-api/main.go`; operator, comparison, boolean, `min`/`max`, error-code and selection swaps, removed negations and `if false &&`. Three parts, **each on its own copy of the worktree** with its own `GOCACHE`, removed after the batch (62 GiB free afterwards). A first attempt ran the three parts on one shared copy; parallel parts restored each other's mutated files, so its results were discarded and the run repeated on separate copies. The first pass ran the server mutants against a `-run` subset of the HYB-299, HYB-294 and HYB-323 tests; every survivor was then re-run against the full `-short` package.

- 248 mutants: 213 killed, 12 did not compile (an `if false &&` before a declaration, a constant rename), 23 survived the first pass.
- Two survivors of an earlier pass (`AddMatching` with a 128-byte revision, a 64-byte filter value) were killed by boundary tests added for them.
- Re-run against the full package: 14 killed by existing tests outside the subset (the proposal-store predicates); 3 more killed by tests added for them (the detail's `no_longer_matches` key only for filter lists, 2 mutants; an emptied index list is deleted, 1 mutant).
- Round 2 (change review C1): the survivor `userwalk.go:62` (`err != nil` dropped from the early return) is not equivalent: a `People` failure in the login segment with no departed people was swallowed. `TestPeopleFailureInTheLoginSegmentFailsTheWholeWalk` (fault on every call and fault on the first call only; `add_matching` and the count read answer `dependency_unavailable` and no draft is created) kills it: mutant 212 re-run against the full package on the rebased tree, `killed` (`$SP/mut323-c1.out`).
- Survivors, each equivalent:
  1. `latestindex.go:47`, the `indexed(item)` test in `unindex` removed: an excluded or held item has no ref with that job's number, and a person appears once per job, so the number test already skips it.
  2. `userwalk.go:19`, `len(ids) > 0` → `>= 0`: with no IDs the loop computes a size of 0 and returns the empty rest.
  3. `userwalk.go:61` and `:65`, the segment flag swapped in `matchWalk`: the cursor it sets is never read by the count or the command.
  4. `cmd/hb-api/main.go:154`, the empty-string test removed from `settingNumber`: `strconv.Atoi("")` fails and the value is refused anyway.

# Deviations

- **V-1 (F2).** Cleanup lookups are built per task (one pass over sessions and pickups for the task's people) instead of persistent indexes inside `sessions` and `directory`; the scaling goal holds (1,000 tasks at 100,000 in 711 ms, at most 7.7 ms per task) and the pickups map keeps its existing writers. Counts are kept incrementally rather than batching `Cleaning`/`Cleaned`; the per-item checkpoint stays.
- **V-2.** `RolePeriod.tsx` (the shared `Settings` component) carries the setting hints; it was not in the plan's file list.
- **V-3.** `cmd/hb-api` accepts setting values of up to eight digits (was six), because the retained-people cap reaches 20,000,000; the refusal is the server error printed at start (`hb-api: configuration is invalid: …`), not a separate log line.
- **V-4.** The users read and the bulk read both return the users settings rows (`user_filter_scan`, `user_count_scan`) after the identity rows, so the M01 settings table and the role period form show them (Q-B3 default).
- **V-5.** `admin.ts` exports `filterOf` and gains the command ID (one line each).
- **V-6.** A revision conflict on `add_matching` (a role sweep moved the revision between the console's read and the command) is retried once with a fresh revision and the same request key.
- **V-7.** The fixed line's `({n} when counted)` uses the last count only when it was complete; a lower bound is not compared.
- **V-9 (render L2).** The fixed line always shows the date, not only when it is not today.
- **V-10 (render L4).** The 14px chip size for removable chips is set in `match.css` (shared by Part A chips) because the shell's control rule outranked `.m01-chip`.
- **V-8.** `x-reads` gains a `/identity/bulk-revocations` entry that records the selection values, the replay change and the masking note; the bulk reads had no `x-reads` entry before.

# [TW] strings needing review

All KO and AR strings are AI drafts. New EN strings from the addendum and this build: the counted-at and lower-bound meta lines; "Selecting…"; the combined refusal sentence; "(at least {n} when counted)" (C3); the fixed line variants (everyone, short, when counted); "Fixed filters:"; "Base revision"; the discarded sentence; the after-fixing announcement; "Everyone matching filters ({n})"; the filter-mode server-checks sentence; "Checking {n} people…"; "Fixed list"; "Filters"; "Fixed at"; "Matched when fixed"; the arithmetic sentence; "{k} no longer match the filters and are excluded." (R-C2); "Everyone in the list"; "Search applied"; "Fixed {at}"; the basis words; "Selection basis"; the system rationale "Selected everyone who matched the filters on the revocation screen"; the four setting names and three hints; R-C8 "No one selected. Use the checkboxes, or select everyone who matches."; "Select everyone in the list". R-C4 resolved («القوائم بانتظار التنفيذ»). Open: the KO and AR singular and dual forms of the selection counts; native review of all AR strings.

# Residuals

- **HYB-331:** Membership tab repeats "No membership" for people without a membership (12 F failures; also on `3bdb750` and `113c0d5`).
- **HYB-332:** AR 390 jobs table overflow and header word breaks; the `span.m01-opener` and `li.m01-item` text-escape results need investigation (6 G failures; also on the base).
- Plan R-B1 (execute does not re-run the filter), R-B2 (masking is a display rule), R-B3 (unfinished executed jobs above `jobs_kept`) stand. HYB-294 R3–R8, R11 stand.
- Q4: removing chosen people from a filter list is HYB-324; `ui-design.md:421` exclusions stay partial.
- Phase B holds the domain lock for 2.05 s at 100,000 (Q3); a users read waited 2.27 s.
- The jobs list read copies every shown job's items through `Get` (42.9 ms with one 100,000 list kept); not changed.
- The browser harness holds about 350 people (about 1,100 for phase L); 100,000 is measured in Go only.
- No change, render, technical-writing or QA review has run on this build; no owner acceptance.

# Planning cross-check

| ID | Source | Evidence in this build | Status |
| --- | --- | --- | --- |
| M01 large data | `M01/spec.md:104` | separate page and filter-wide actions; server-fixed set, count and scope (`add_matching`, digest); drift shown as `no_longer_matches`, never widened | covered |
| M01 bulk action | `M01/spec.md:75` | fixed manifest, chunked cleanup after the deny at 100,000, targets/exclusions/effect in the preview | covered |
| M01 mobile and tabs | `M01/spec.md:112`, `:81` | summary before confirmation at 390; the fixed list survives tab changes and reloads | covered |
| M01 scope | `M01/spec.md:25` | workspace from the session only; `workspace` refused; same admission for the count | covered |
| IAM-07 | `retention-access.md:116` | revocation of 100,000 measured over HTTP; isolation and counts invariant asserted | revocation covered; invitation and sync outside |
| UI-04 | `ui-design.md:547`, `:420-423` | bounded count with "at least"; waiting instead of a number; no client ID array; base revision fixed | covered for M01 |
| `ui-design.md:421` | | targets, system exclusions and base revision fixed; chosen exclusions open | partial (HYB-324) |
| RET-05 | `retention-access.md:98` | phase B before the answer | covered |
| HB-03 | `requirements.md:53` | server-side permission and scope checks, negative tests | covered |

## Root final verification (2026-11-19, rebased on main 113c0d5)

- Reviews: design review rounds 1–2 CHANGES REQUIRED → round 3 APPROVE; change review round 1 FAIL (C1 test evidence) → round 2 PASS; render and copy review round 1 FAIL (copy) → round 2 PASS. Root applied the round-2 consistency wording "(at least {n} when counted)" in EN and AR (KO already used 최소) in code, test, glossary and this record.
- Rebase conflicts in `apps/api/cmd/hb-api/main.go` and `main_test.go` (HYB-309 plugin flags vs this ticket's settings flags): both sides kept.
- Root re-ran: gofmt clean; `go vet ./...` clean; `go test -short ./...` all packages ok; coverage server 100.0%, bulkrevoke 100.0%, identity 99.0%; console tsc clean, npm test 786/786, build ok; OKF status test ok.
- Browser QA and regressions are the implementer's round-2 runs above; the only failing QA checks are the inherited ones ticketed as HYB-331 and HYB-332, not counted as passed.
- V-10 (removable chips at 14px) brings HYB-294 Part A chips into line with HYB-294 visual-spec §4.4; recorded here as a correction toward that spec.
