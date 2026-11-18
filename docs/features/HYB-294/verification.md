---
type: VerificationRecord
title: HYB-294 Part A M01 server search, filters and paging, implementation verification
description: What Part A built against the approved plan and the final visual spec, the checks run with their observed results, the measured scale numbers before and after the indexed listings, browser QA, regressions, mutation, deviations, plan-text changes for root, residuals and the planning cross-check. Written by the implementer; no independent review has seen this build.
status: stable
tags: [HYB-294, verification, M01, identity, search, filters, paging, IAM-07, UI-04]
sources:
  - resource: plan.md
  - resource: visual-spec.md
  - resource: ../HYB-299/verification.md
  - resource: ../HYB-300/verification.md
---

# Scope of this record

- Branch `feat/HYB-294-m01-server-search`, worktree `/private/tmp/hb294`, built on main `e360bf7` and rebased by root onto `bc0d03e`. Nothing is committed; root owns review, commit and merge.
- Part A only (plan "Part A", D1–D6 and the Part A parts of D4/D5). Part B (HYB-323: `add_matching`, the count read, the filter digest, the selection-bar action) is not built and has no placeholder.
- Every result below was observed on 2026-11-18 on the Mac mini (darwin/arm64). Where a check was not run, it says so.
- The change review (PASS), render review (FAIL) and technical-writing review (FAIL) ran on the first build; their fixes are listed under "Review fixes". No review has re-run on the fixed build.

# Review fixes (2026-11-18, tree rebased by root onto main `bc0d03e`)

Root's decisions from the change review (PASS, 1 medium, 4 low), the render review (FAIL, one major) and the technical-writing review (FAIL), all applied:

- **M1. Page-scoped latest outcome.** New `modules/bulkrevoke/latestindex.go`: a per-workspace `user → (job, item)` index rebuilt whenever the executed set changes (`prune`, which `Execute` and `Restore` call); `LatestFor` reads the item's current state through it in O(page). `Latest` keeps the full scan and is now only the test oracle. Tests: index after execute, prune (kept 2 of 5), header state change, restore.
- **L1.** `x-reads` now records 401, 403, 409 (field `after`), 422 (field `after`), 501 and 503 with their next actions; the pinning test checks each.
- **L2.** While a read is in flight the selection bar gets no loaded people (`loadedIds`), so "Select the loaded people" selects nobody; unit test and QA step.
- **L3.** Every ready load (filter change, Reload, an action's reload) closes a row panel whose person is no longer in the rows, moves focus when it was in the panel, and announces it ("not in the filtered list" with filters, "not in the loaded rows" without). Pickup and create panels are not closed. Unit test (`lostPerson`) and QA step.
- **L4.** The property walk compares the generated people's rows with the independent model over every filter key in the trial (second oracle); shown to fail on a broken holder predicate.
- **R1.** Fixed field widths (`inline-size: Npx; max-inline-size: 100%`). The technical-writing strings are longer: "Role period ended, suspension pending" was cut at 320px in the first render of this fix. Two widths therefore follow the spec's own rule (longest EN option + 52px): membership 336px in every language (was 320; AR already 336), period 242px (was 232). QA checks every option's text against its closed select and the widths.
- **R2.** M01-only toolbar grid keeps Reload and Create user at inline-end on the heading row (`bulk.css`).
- **R3.** Applied chips take `flex: 1 1 16rem` beside the "Applied filters:" label.
- **R4.** Logins in rows get `<wbr>` before "@" and after each "."; the 32-character harness login no longer breaks inside a word.
- **R5.** Resolved by TW row 10.
- **R6.** The merged role-and-period cell omits the period line when there is no membership.
- **R7.** The stopped empty state renders "No matches yet." and the searched-up-to block as one block (8px gap).
- **R8.** Viewport-only captures (`viewport-users-<lang>-<width>.png`) taken; see Browser QA.
- **TW rows 1–14** adopted as proposed with root's decisions: jobs-note tab names built from the shell's tab label source (`tabLabel` + `screens.json`); plural-aware "Ends within {n} day(s)" and "Ends in more than {n} day(s)"; "List updated. {count}. Filters applied: {k}."; no count line and "No people shown." with zero rows; "Role period ended, suspension pending"; "Password set/reset link issued"; "login ID" (KO 로그인 ID) with a glossary row; "Departed (login and email removed)"; Invitations set-link cell "Issued, expires {time}" or "Issued, expires with the invitation" when the two instants are equal; the method line built from `auth_methods` (AR الدخول الموحد); KO "지금까지 일치하는 사람이 없습니다."; short membership options with "Membership suspended" kept in cells; the superseded `plural`/`count`/`invitationCount` copy removed (no consumer in source, tests or the required QA runners; the older `qa287*` runners only mention the name). Glossary section rewritten accordingly.

## Results on the fixed build

| Check | Result |
| --- | --- |
| gofmt; go vet darwin, linux, windows | clean; all three pass |
| `go test -short ./...` | 73 packages ok, none failing |
| `-race` on `apps/api/server`, `modules/identity`, `modules/bulkrevoke`, `cmd/hb-api`; `-race -count=20` on the latest-index tests | all pass |
| Coverage | server 100.0%, bulkrevoke 100.0%, identity 99.0% |
| Console | tsc clean; `npm test` 760/760; build passes |
| Browser QA (fresh API, `-user-filter-scan 100`, harness of 301 people incl. a 32-character login) | harness 22/22; self-test 25/25; state matrix EN/KO/AR × 1440/1280/1024/390 **1425/1425**; functional **69/69** (2 page errors are the deliberately aborted next-page request) |
| Regressions | M01 `qa299` A/C/B/D/E/L/P/N 784/784; M01 `qa264on294` 52/52 (its invitations wait now accepts the empty state, because zero rows show no count line); M03 157/157; M02 191/191; HYB-293 134/134; P01–P05 A, B, C1, C2, C3, F, P, R and E (`--seed none`) 828/828 |

New QA checks, each shown to fail on a deliberately broken DOM state in the self-test (phase X) before use: every option's text fits its closed select; the fixed widths; field rows at 1440; Reload and Create user at inline-end on the heading row; chips beside the "Applied filters:" label; no stray dash in merged cells without a membership; the stopped empty state as one block; a long login breaking inside the word without break points (and not with them). The checks were not run against the pre-fix console build, because that build was not kept after root's rebase; the self-test breaks the same properties instead.

Observed layout at 1280 with the fixed widths (recorded, not asserted): EN and AR rows `1122233` (the search form with its button is 404–418px wide, so membership wraps), KO `1112222`. The spec's calculated "#7 alone on row 3 at 1280" assumed a 304px search block and a 320px membership field; at 1440 every language matches "1–3 then 4–7". Row-open captures at 1440 and 1280 confirm the band keeps its width and the list stays beside the panel (§15.2). Viewport-only captures (`viewport-users-*.png`) show the sidebar in place; the earlier offset is a full-page capture artifact.

Executed-job history in the scale test (100 jobs × 5,000 targets, recorded numbers):

| People | First page, indexed `LatestFor` | First page, previous full-scan `LatestFor` (scratch copy) | Full-history scan alone |
| --- | --- | --- | --- |
| 10,000 | 234 µs, 448 KiB | 9.75 ms, 2,240 KiB | 8.5 ms |
| 100,000 | 235 µs, 448 KiB | 16.6 ms, 14,793 KiB | 15.1 ms |

The other scale numbers on this build: first page 248 µs / 304 µs, deep page 228 µs / 270 µs, sparse filter 307 µs / 320 µs at 10,000 / 100,000.

# Round 3 review fixes (2026-11-18)

Render re-review PASS; technical-writing re-review FAIL on one item. Applied:

- **W1.** The kept action result names the person: status line "{login}: {result}" with the login in an LTR isolate (or the user ID in mono when there is no login), text capped at 68ch, 12px block padding. The announcement is "{login}: {result}." followed by the open-person sentence (filtered or loaded-rows variant).
- **W2.** The superseded `w.setIssued` / `w.resetIssued` strings are removed from `live-copy.ts`; `setup()` in `live-model.ts` now uses `fc.setLinkAt` / `fc.resetLinkAt`, and its test asserts the new wording and that the old keys are gone. No other consumer (grep over the console source and the M01 runners; `live-copy.ts` `w` is also imported by M02–M05 and O01/O02, none used these keys, tsc confirms).
- **W3.** Glossary row `action-result-kept`.
- **W4.** A tab change clears the kept result (`[view]` effect).
- **W5.** Captures in EN, KO and AR at 1440 and 390 (`w5-r6-*`, `w5-n1-*`, `w5-inflight-*`) with the live-region text logged; filtered-empty captures are in the state matrix (`filtered-empty-*`).

Shown failing on the pre-fix build first (`qa294-p3-W.out`): W1 "the kept result names the person" failed in all six language/width combinations; the W2 unit test failed. Two checks passed before the fix and could not be shown failing: the 12px padding and 68ch cap (the shared status style already gave them) and W4 (the shell remounts the screen on a tab change, so the old result was already gone; the explicit clear is still added). R-6 is not reachable below 1280: the step view hides the list and Reload, so the person is closed with Back first; recorded as INFO, not as a check.

## Results on the round-3 build

| Check | Result |
| --- | --- |
| Console | tsc clean; `npm test` 763/763; build passes |
| Browser QA (fresh API and harness) | harness 22/22; self-test 27/27; state matrix 1497/1497; round-3 phase W (EN/KO/AR × 1440/390) 33/33; functional 70/70 on a fresh harness (2 page errors: the deliberately aborted next-page request) |
| M01 regressions | `qa299` 784/784; `qa264on294` 52/52 |
| Shared-file regressions (`live-copy.ts` is imported by M02–M05 and O01/O02) | M03 157/157; M02 191/191; HYB-293 134/134; P01–P05 828/828 |

The first functional run of this round ran after phase W on the same API, which had suspended twelve harness people, and stopped at its own "10 suspended rows" wait; on a fresh harness it passed 70/70. Live-region text logged in phase W, e.g. EN "qa021@example.invalid: Account suspended. The person you had open is not in the filtered list.", KO "qa025@example.invalid: 계정을 정지했습니다. 열어 둔 사람이 필터 결과 목록에 없습니다.", AR "qa029@example.invalid: تم تعليق الحساب. الشخص الذي فتحته ليس في القائمة المصفّاة.". Go code did not change in this round.

# Round 2 review fixes (2026-11-18)

Change re-review PASS (lows), render re-review FAIL (2 must-fix), technical-writing re-review FAIL (1 blocking). Applied:

- **N1.** When the R-6 check follows an action's reload, the action result stays as a status line in the list area (`.m01-actionresult`) and the live region says "<result>. <gone>" in one announcement.
- **N2.** `x-reads` lists every code per status in `x-codes`: 401 `unauthenticated`; 403 `csrf_refused`, `permission_denied`; 409 `revision_conflict`; 422 `invalid_request` with `x-fields: [after]` marked `x-fields-when: cursor`; 501 `unsupported_capability`; 503 `authority_unavailable`, `dependency_unavailable`. Pinned in the test.
- **N3 + V1.** `selectableIds` gives no loaded people only on a fresh read (the query changed); a same-query reload keeps the visible rows loaded. `Selection.tsx` renders no select-loaded control (bar label or header-cell checkbox; the header cell stays as an empty `th` for parity) when nothing is loaded: fresh read in flight, filtered empty, zero-row scan stop. Count line, the filter-wide note and the not-loaded disclosure stay.
- **N4.** `bulkrevoke.Latest` moved to `latest_oracle_test.go`; production keeps only the index.
- **N5.** Recorded for HYB-323: `reindex` walks every kept executed item on each prune (100 jobs × 5,000 targets were recorded in 0.9–1.4 s in total in the scale test, about 9–14 ms per execute); measure it in the Part B scale run with the 100,000-target lists.
- **Eviction test.** Finished jobs (header, records, cleaning, cleaned) evicted beyond `Kept`: `LatestFor` equals the oracle after every execute, an evicted person's only job disappears, others fall back to the latest kept job.
- **V2.** `.m01-chips { flex: 0 1 auto; min-inline-size: 0 }`, `.m01-applied { align-items: flex-start }`: Clear all follows the last chip.
- **V3.** `.m01-filter-fields > .service-field { inline-size: min-content }`: labels wrap to their control's width.
- **V4.** The jobs note already ends with a full stop in EN, KO and AR on this build (the QA check passed before and after; it could not be shown failing).
- **V5. Not done.** The ui-designer §4.2/§14 amendment text was not supplied to me (searched the scratchpad and task files), and `visual-spec.md` is outside my files. Root to paste the text.
- **T1.** Count fragments lose their end punctuation inside `updated`/`removed` (`bare`); unit test over 0/1/n rows and k = 0/1/2 in EN/KO/AR.
- **T2.** KO "{n}일 이후 종료".
- **T3.** Users-tab and detail sign-in setup: "Password set link issued, expires {time}" / "Password reset link issued, expires {time}" with KO and AR as given.
- **T4.** Glossary: the Create form keeps "Sign-in email" because new logins must be emails.
- **T5.** Method line: KO particles by the final sound of the word before any parenthesis (은/는, 와/과); AR agreement by item count (غير متصل, غير متصلَين, غير متصلة); EN sentence capitalised. Unit tests for 1, 2 and 3 items.
- **T6.** AR membership option "نشطة" (filter only); glossary updated.

Every new unit and QA check was run on the pre-fix build first (scratchpad `qa294-prove-*.out`, unit run before the source changes):
- unit: T1 (`List updated. No people shown.. Filters applied: 0.`), T2/T3/T5/T6 values, `selectableIds` (did not exist), the select-loaded render test, the N2 x-codes test — all failed;
- QA (EN/KO/AR × 1440 and 390, plus the functional run): no select-loaded control in flight, in the filtered empty state and in a zero-row stop — failed in every case; Clear all after the last chip — failed in every case; the action-result line and combined announcement — failed; field labels without one-word lines — failed only in AR at 390; the jobs-note full stop and the same-query reload keeping the control — passed before (the first because the stop was already there, the second because the control was always rendered before V1).

## Results on the round-2 build

| Check | Result |
| --- | --- |
| gofmt; go vet darwin, linux, windows | clean; all pass |
| `go test -short ./...` | 73 packages ok |
| `-race` (4 changed packages; latest-index tests ×20) | all pass |
| Coverage | server 100.0%, bulkrevoke 100.0%, identity 99.0% |
| Console | tsc clean; `npm test` 763/763; build passes |
| Browser QA (fresh API and harness) | harness 22/22; self-test 27/27; state matrix EN/KO/AR × 1440/1280/1024/390 1497/1497; functional 70/70 (2 page errors: the deliberately aborted next-page request) |
| Regressions | M01 `qa299` 784/784; `qa264on294` 52/52; M03 157/157; M02 191/191; HYB-293 134/134; P01–P05 828/828 |
| Scale test with 100 executed jobs × 5,000 targets | passes; first page with history 234 µs (10,000 people) and 370 µs (100,000) |

- Live-region text logged in the functional run: R-6 "The person you had open is not in the filtered list."; N1 "Account suspended. The person you had open is not in the filtered list." Captures: `r6-closed-en-1440.png`, `n1-action-result-en-1440.png`; 390 stacked rows without a membership in `filtered-*-390.png` and `departed-*-390.png`.
- After the first round-2 QA pass two layout problems showed and were fixed before the final run: with `flex: 0 1 auto` the membership chips wrapped below "Applied filters:" at 1024/1280 (EN), so Clear all is now the last item of the chip list and the applied row does not wrap; and the AR search label left one word on its last line, so field labels use `text-wrap: balance`.
- Field rows at 1280 (recorded): EN `1122233`, KO and AR `1112222` (labels now wrap to their controls, so AR fits more per row than in round 1).


# What was built

## modules/identity

- Sorted ID indexes inside the registry: `userIndex` and `memberIndex[workspace]`, each with a per-index version counter (`usersVersion`, `membersVersion[workspace]`). All inserts go through `putUser` (bootstrap, `CreateUser`, `CreateServiceUser`) and `putMembership` (bootstrap, `Accept`), which insert in place by binary search. A source test fails if `r.users` or `r.memberships` is written or deleted anywhere else (`registry.go`, `boundary.go`, `installation.go`, `workspace.go`).
- `Installation.Users` and `Workspace.Members` page by `sort.Search` on the index; `Members` skips removed memberships while filling the page. Results are unchanged (property test against the old sort-and-page code kept as the oracle in `people_test.go`).
- New `people.go`: `(*Workspace) People(actor, workspace, ids)` (administrator verifier, at most `PeopleLimit` = 256 IDs, unknown IDs absent, membership with `Effective` at the registry clock, `ErrInvalid` for a bad workspace, ID or too many IDs) and `(*Installation) User(id)`, the indexed point lookup.
- The identity part of `list_version` is `Subjects().Generation()` (unchanged).

## modules/bulkrevoke

- `LatestFor(workspace, ids)` returns the latest executed outcome for the page's people only; `Latest` now shares the same loop.

## apps/api/server

- `directory.go`: `byLogin`, `order` (user IDs in login order, user ID as tie-breaker), `departed` (user IDs ascending) and a directory `version`, all maintained in place by `add`, `remove` and `depart`. `invite`, `dropInvite`, `issued`, `dropLink` and `dropHolder` change invitations, links and holders and move the version only when something changed. `userOf` uses `byLogin`.
- `admin.go`: `findUser` uses `Installation.User`; `departUser` calls `dir.depart` regardless of `c.known` (N4) and drops the invitation, link and holder through the directory; invitation and link writes go through the directory helpers.
- New `userfilter.go`: parameter parsing (unknown, repeated or empty parameter, unknown value, `workspace` → `invalid_request`; `method` → `unsupported_capability`), cursors `a:<user>` / `d:<user>`, simple Unicode case folding for `q`, the membership partition (`active`, `suspended`, `ended`, `none`), sign-in setup, invitations and holder predicates.
- `users.go`: the read checks `answer.Workspace() == s.bootWorkspace` (L1), resumes from the cursor's segment, fetches candidates through `People` in chunks of at most 256 within the scan budget, evaluates every filter at the page's `server_time`, and answers `filters`, `scan_stopped`, `scan_budget`, `order`, `list_version` (`<identity generation>.<directory version>`), `auth_methods`, `warning_days` and `next_login`. A login-segment cursor whose person has departed answers `revision_conflict` (409, `inspect_current_state`, field `after`) through the new read-side writer `failRead`; other bad cursors answer `invalid_request` with field `after`. `roster()` and `everyone()` are unchanged in code and now page through the indexed listings.
- `settings.go`: `user_filter_scan` (people, default 1000, bounds 100–10,000) in its own `UserSettingList`, validated with the other settings; `cmd/hb-api` registers `-user-filter-scan`.
- `openapi.go`: the read is recorded under `x-reads["/identity/users"]` with its parameters, the refused parameters, the order and the 409/422/501 answers.

## apps/console

- `service/admin.ts`: `users(query)` sends only the set filters, `after` and `limit`; the page carries the new fields.
- New `filter-model.ts` (link parameters `q`, `st`, `ms`, `role`, `per`, `si`, `adm`; invalid values dropped with the bad-link notice; query building with the fixed `invitation=pending` on the Invitations tab; merge of pages by user ID), `filter-copy.ts` (EN/KO/AR drafts) and `filters.tsx` (scope line, filter band, chips with Clear all, the sign-in method line, the paging area with Show more, Continue searching, the next-page failure, the list-changed notice and Start again).
- `Live.tsx`: filters from the link, first page of 50 per filter change with rows cleared and "Loading users…", rows merged by user ID, the list-version notice, the restart answer, the filtered and stopped empty states, the jobs-tab note, the bad-link notice, chip-removal focus moves, announcements, closing the panel when the open person leaves the result (R-6), the departed sub-line, the `ended` membership word, and the selection disclosure kept from the last ready rows while a read is in flight. The client-side `rowsFor` is removed.
- `app/screen-groups/identity.tsx`: M01 receives `params` and `onParams`.
- `bulk.css`: band, chips, paging styles (neutral borders, radius 0), full-width grid placement for the scope line, conditions, band, bad-link notice and jobs note.

# Checks and observed results

| Check | Command | Result |
| --- | --- | --- |
| Format | `gofmt -l apps modules` | no files listed |
| Vet | `go vet ./...` on darwin, `GOOS=linux`, `GOOS=windows` | all three passed |
| Cross builds | `GOOS=linux GOARCH=386 go build ./...`, `GOOS=windows GOARCH=amd64 go build ./...` | both passed |
| Go tests | `go test -short -count=1 ./...` | 73 packages `ok`, none failing; non-short runs of `modules/identity`, `modules/bulkrevoke` and `apps/api/cmd/hb-api` pass, and the server scale test passed in its own run (below) |
| Race | `go test -race -count=40` on the new identity, bulkrevoke and server tests; `-race -count=3` on the brute-force property test; `-race -short` on the four changed packages (repeated after the last code change) | all passed |
| Coverage | `go test -short -cover` | `apps/api/server` 100.0%, `modules/bulkrevoke` 100.0%, `modules/identity` 99.0% (98.9% on `e360bf7`; every new function 100%), `apps/api/cmd/hb-api` 92.9% (same as on `e360bf7`) |
| Console | `npx tsc --noEmit`, `npm test`, `npm run build` | tsc clean; 720/720 tests pass; build passes |
| File size | changed source and test files | largest changed file `admin.ts` 1,083 lines; `Live.tsx` 776 |
| Source rules | no comments, no binary-fraction type in Go tests (server `TestSourceRules`) | pass |

## New tests

- `modules/identity/people_test.go`: index and version after bootstrap, create, service user and acceptance; versions unchanged by sessions; sorted insert; removed memberships skipped without shortening a page; `People` (unknown IDs, limit 256/257, bad workspace or ID, member, other-workspace administrator, no session, nil handle, period end, two workspaces); `User`; listings against the sorting oracle over 30 random registries; concurrent readers of a growing index under `-race`; the source test.
- `modules/bulkrevoke/latest_test.go`: `LatestFor` equals `Latest` restricted to the given people.
- `apps/api/server/users_filter_test.go`: each filter alone and combined; the membership partition; case folding with KO, AR, Greek final sigma and the Kelvin sign; 64/65 bytes; every refusal including `workspace` in three forms and `method`; no session, CSRF, member, the workspace-source check; cursor refusals (unknown, service, human without a directory entry, departed-segment misuse, past the last departed ID); the 409 restart; registry faults; the scan budget (exactly at the end, before the only match, resume, at the last login, limit at the end of the login segment with and without departed people); a person created between pages before and after the cursor; `list_version` unchanged by time and changed by a change; departures with and without a directory entry; directory index unit test; departure clears the invitation, link, holder and password; the directory equals the identity humans (A1); `findUser` against the full listing; the invitations filter against the old console rule; top-level key allowlist; the `x-reads` record; the setting bounds; two installations never mix.
- `apps/api/server/users_property_test.go` (added): random populations of 20 to 2,000 people with random account and membership states, periods, links, invitations, holders and logins; random filters, budgets (1–30, or 100–500 at 2,000) and limits; every walk equals a brute-force filter in the two-segment order, has no duplicates and has strictly increasing cursors per segment; each generated person's single-key filter results equal an independent model (with an `EqualFold`-based search); every person matches exactly one membership value. The test was shown to fail when the limit check, the login order or the case folding was broken.
- `apps/api/server/directory_source_test.go`: directory fields (`people`, `byLogin`, `order`, `departed`, `version`, `invites`, `links`, `holders`) are written only in `directory.go`, with no exceptions (D9), and the password-state call sites are pinned. Shown to fail on an added stray write.
- `apps/api/server/users_scale_test.go`: the scale run and the benchmarks (skipped with `-short`).
- `apps/api/cmd/hb-api/main_test.go`: `-user-filter-scan` bounds, form and usage text.

# Scale measurements (D4/D5, IAM-07, UI-04)

Recorded numbers, not thresholds. 100,000 people with memberships were built through the public identity API in invite-and-accept pairs (pending invitations stay at zero); building took 66.1 s on this build and 69.6 s on `e360bf7`, because `Invite` counts every stored invitation. No fallback to an engine-level test was needed.

**Before** (`e360bf7`, the same population, measured with a scratch test in a detached base worktree):

| People | First page (50) | Deep page (90%) | Whole-list calls per page | `roster()` | `everyone()` | `sweepRoles` |
| --- | --- | --- | --- | --- | --- | --- |
| 10,000 | 115.3 ms, 72.7 MB allocated | 110.0 ms | 101 roster pages + 1 users page | 116.8 ms | 197.9 ms | 95.7 µs |
| 100,000 | 15.94 s, 8.79 GB allocated | 15.94 s | 1,001 roster pages + 1 users page | 15.86 s | 28.67 s | 2.02 ms |

**After** (this build, `TestScaleUsersRead`):

| People | First page (50) | Deep page (90%) | Sparse filter exhausting the budget of 1,000 | `roster()` | `everyone()` | `sweepRoles` |
| --- | --- | --- | --- | --- | --- | --- |
| 10,000 | 237 µs, 3,019 allocations, 419 KiB | 213 µs | 292 µs, 623 KiB | 2.06 ms | 3.27 ms | 78.4 µs |
| 100,000 | 255 µs, 2,988 allocations, 416 KiB | 229 µs | 316 µs, 623 KiB | 49.2 ms | 66.4 ms | 2.26 ms |

- The page cost includes the HTTP handler, JSON encoding, the clock and settings rows. The assertion that a page never calls the whole-list path (`roster` or `users` fault hooks) held at both sizes; the page made only `People` calls.
- Identity listing at 100,000 (`TestScaleListingsOfOneHundredThousand`): indexed `Users` page at 90% 235 ns, indexed `Members` page 404 ns, `People` of 256 IDs 24.0 µs; the replaced sort-and-page path 13.0 ms (users) and 16.7 ms (members) per page.
- `sweepRoles` stays O(N) per command: 2.26 ms at 100,000 memberships (R7).
- Benchmarks (`go test -run '^$' -bench BenchmarkUsers -benchtime=50x`, after all other runs had finished):

| Benchmark | ns/op | B/op | allocs/op |
| --- | --- | --- | --- |
| `BenchmarkUsersPage` 10,000 | 231,268 | 427,598 | 2,992 |
| `BenchmarkUsersPage` 100,000 | 233,008 | 424,861 | 2,991 |
| `BenchmarkUsersFiltered` 10,000 (budget 1,000, nothing matches) | 317,708 | 639,427 | 3,379 |
| `BenchmarkUsersFiltered` 100,000 | 315,880 | 638,718 | 3,377 |

# Browser QA

Runner `qa294.mjs` in the scratchpad (helpers copied from `qa299.mjs`: overlap and clipping, page scroll, the computed-border sweep, header/cell parity, the text checks (a) repeated block, (b) AR date isolation, (c) word joins, hint proximity; plus the `qa300.mjs` layout check: cut cell, split date, broken word, escaping text, overlap; plus a cell-gap check and a neutral-border check for chips, toggle, selects, inputs and paging buttons). Live `hb-api` built from this worktree with `-user-filter-scan 100`, the sample seed, and a QA harness that adds 300 people through the API (10 suspended, 10 departed) — no harness code is in the worktree. Pointer moved off before every capture; fonts awaited.

- **Self-test (phase X): 16/16.** Each check was proven to fail on a deliberately broken state (coloured band border, coloured chip border, a removed cell, zero cell padding, a 4,000px element, a forced-wide table, a wrapped `time`, an unisolated AR date, joined inline words, a repeated block, a shifted chip, a total claim, a removed departed sub-line, a word broken in a cell) and the unbroken page passed the layout check.
- **Harness (phase S): 21/21.**
- **State matrix (phase F): 1071/1083** at EN, KO and AR × 1440, 1280, 1024 and 390: users (collapsed below 1280, then opened), status filter within the scan budget (rows, "more may exist", "Searched up to …", Continue searching), a search that stops with no rows ("No matches yet."), the filtered empty after continuing to the end with Clear all, departed rows after continuing (ID, sub-line, no login), invitations with the fixed chip, membership with the suffix chips, the jobs-tab note without a band, and the bad-link notice. Checked in every combination: count line wording, order line, scope line, method line as text, 44px controls, band width over the list, chip names, link kept, announcement, keyboard order through the band (1440), clear-all focus.
  - The 12 failures are all the inherited check (a) on the Invitations tab: the invitation expiry and the set-password link expiry are the same instant for a person created by `identity.create_user`, and both are now rendered as isolated date and time parts, so the identical stamp text appears in two columns of the row. Before this build the second date was part of a plain sentence and the check did not see it. Not changed; see residual R14.
- **Functional (phase G): 62/65** at EN 1440 (and 390 where noted): Show more appends 50 and focuses the first new row; the compact count at 390; an aborted next page keeps 100 rows, alerts and retries to 150; a create between pages shows the list-changed notice with Reload and keeps rows; departing the cursor person gives the restart alert with Start again focused and rows kept, then Start again reloads; a full walk with a departure while paging has no duplicate row, departed people last and "end of list"; a filter change keeps three selected people and lists them as not loaded; removing a chip moves focus to the next chip and announces "Filter removed."; the open person outside the new result closes the panel and is announced; filters survive the membership tab; a member session gets the denied state without band or scope line.
  - The 3 failures are the `qa300` broken-word check on a harness login of 32 characters without break opportunities (`aaa1794996594070@example.invalid`) in the user column at 1440. The same login breaks the same way on `e360bf7` (probe against a base build); the column CSS is in `live.css`, outside Part A. Residual R13.
  - The 2 page errors are the browser's `net::ERR_FAILED` for the deliberately aborted next-page request.
- Screenshots: 107 PNGs under `scratchpad/qa294shots` (`<state>-<lang>-<width>.png`).

**Final build re-run (after the two late server changes and the D9 switches, fresh API and harness):** harness 21/21, self-test 16/16, state matrix 1071/1083 (the same 12 Invitations repeated-date failures, R14), functional 64/65 (one long-login word break, R13; the 2 page errors are the aborted request). `go test` for `apps/api/server` (100.0%) and `modules/identity` (99.0%) passed after the D9 change. The regression suites above ran before the late changes and were not repeated.

## Defects found by QA and fixed in this build

1. Long chips at 390 overflowed the band (e.g. "Membership in this workspace: No membership"): chips, their list items and labels now shrink and wrap (`bulk.css`).
2. AR dates in the sign-in setup column, the detail, the Invitations columns and the link pickup panel were plain text inside a sentence (not isolated). Present on `e360bf7` (base probe). They now render through the existing `Stamp`/`At` parts (`Live.tsx`).
3. The QA layout check reported text in the visually hidden header label and in a closed disclosure; the runner now skips those (check fix, not product).

# Regressions

All runners ran against this worktree's console build and an `hb-api` built from this worktree (`hb-api294`), each on its own console and API ports (4481–4489, 8840–8849), through a scratch copy of the static proxy that forwards every `/api/` path to the chosen port (the vite preview proxy is fixed to 8820).

| Suite | Runner | Result |
| --- | --- | --- |
| M01 (HYB-299) | `qa299.mjs` phases A, C, B, D, E, L, P, N via `qa299g.sh` | A 334/334, C 3/3, B 5/5, D 5/5, E 53/53, L 352/352, P 1/1, N 31/31 |
| M01 (HB-264 runner) | `qa264on299.mjs` | stopped at its wait for the old "{n} invitations" count noun, which the plan replaces with the count line; `qa264on294.mjs` (the same runner with the two count expectations changed to "4 shown; end of list" and the fixed invitations chip): 52/52 |
| M03 | `qa298on297.mjs` phases A–E | A 67, B 21, C 19, D 30, E 20 pass, 0 fail |
| M02 | `qa297.mjs` phases A, B, C, D, E, I, K, L, F1, F, G, H | 17, 3, 3, 3, 2, 3, 130, 21, 2, 2, 3, 2 pass, 0 fail |
| HYB-293 | `qa293on297.mjs` groups B, C, D, A | 32, 27, 54, 21 pass, 0 fail |
| P01–P05 | `qa315bodyon297-full.mjs` groups A, B, C1, C2, C3, F, P, R (sample seed) and E (`--seed none`) | 17, 9, 2, 3, 30, 132, 12, 481 and 142 pass, 0 fail. Group E first ran with the sample seed by mistake (0/1, waiting for the bootstrap form) and passed when run with the seed it needs. |

# Mutation

Driver `mutate294.py` (scratchpad) on an rsync copy of the worktree, three parallel parts, each with its own `GOCACHE` removed after the batch (disk 63 GiB free afterwards). Mutants were generated on every changed or new line of `users.go`, `userfilter.go`, `directory.go`, `admin.go`, `settings.go`, `openapi.go`, `bulkread.go`, `cmd/hb-api/main.go`, identity `registry.go`, `installation.go`, `workspace.go`, `boundary.go`, `people.go` and bulkrevoke `job.go`: operator and comparison swaps, error-code swaps, boolean flips, `min`/`max`, `++`/`--`, removed negations, `if false &&`, removed statements and segment swaps. The first pass ran the server mutants against the HYB-294 and user-administration tests (`-run` subset) to keep it within time; every survivor was then re-run against the full package.

- First pass: 287 mutants, 246 killed by tests, 18 did not compile, 23 survived.
- After adding tests and two code simplifications, every survivor was re-run against the full `-short` package: 16 killed, 2 gone because the code changed, 5 survived. The 16 directory-version mutants (8 lines, `++`→`--` and removed) and 5 mutants on the rewritten `Members` loop were all killed.
- Killed by tests added for them: a `d:` cursor past the last departed ID; an `a:` cursor of a human without a directory entry; limit reached at the last login with nobody departed; version increments for `add`, `invite`, `issued` and the drops; departure clears the invitation, link and holder; the 422 answer has no `x-fields`; the bulk list accepts `limit=100`.
- Gone: `!ok ||` in the candidate loop was redundant (a missing person is a zero value whose kind is not human) and was removed; the `Members` loop now carries its bounds in the loop condition instead of a `break` whose `continue` mutant only cost time.
- Survivors, each equivalent:
  1. `users.go` `segment` returns `true, err` instead of `false, err` on a `People` failure: the caller returns the error and ignores the flag.
  2. `userfilter.go` `foldRune` with `max` instead of `min`: any fixed member of the case-folding orbit is a correct canonical form; equality and substring results are the same.
  3. `admin.go` `findUser` returns `ok = true` with `ErrUnavailable`: every caller checks the error first.
  4. `admin.go` `issueLink` failure path without `dropLink`: `passwords.Issue` drops the login's own link before the limit check, so a reissue for a login that has a live link record cannot hit the limit, and expired records are pruned at the start of every user command. The line only changed from `delete` to the directory helper.
  5. `bulkread.go` `pageLimit` with the error check disabled: on any `strconv.Atoi` error `n` is 0 or out of range, so the bounds still refuse it. The function was moved unchanged from `users.go`.

# Deviations from the plan

- **D1. No `dropUser`.** Nothing deletes a user; adding an unused helper would be dead code. The source test forbids `delete(r.users…)` outside the helpers, so a future delete must add one.
- **D2. `People` limit is a fixed 256** (`identity.PeopleLimit`). The plan says "PageSize×4 (256)", which is 400 at the installation's page size of 100.
- **D3. Per-index counters move only on index inserts.** In-place state changes are covered by `r.generation`, the identity part of `list_version` (round 2, N5); every state change a listing reports already moves it.
- **D4. The read is recorded under `x-reads`**, not `paths`: existing OpenAPI tests and consumers treat every `paths` entry as a command.
- **D5. `next_login` added to the read.** "Searched up to {login}." needs the login of the last examined person, who may not be a returned row. Departed cursors have none; the console then says "Searched up to user {id}.".
- **D6. `user_filter_scan` is not in the M01 settings table.** It is its own settings group, so the M01 identity settings rows (labelled in `bulk-copy.ts`, a Part B file) do not show an unlabelled row. The page carries `scan_budget`.
- **D7. Existing tests changed for the new contract or the write path:** `users_test.go` (login order; no empty `after`), `users_edges_test.go` (the `people` fault, directory entries for directly created people, the new `usersPage` signature), identity `workspace_test.go`, `periods_test.go`, `scale_test.go` (direct map writes replaced by the helpers).
- **D8. Two QA-found defects fixed in Part A files** (chip wrapping; AR date isolation, which predates this ticket).
- **D9. Resolved after root authorization (2026-11-18).** `self.go` `changePassword` and `signin.go` `setPassword` now drop the link through `dir.dropLink`, and `seed.go` `grantInstallation` records the holder through the new `dir.grant`; each moves the directory version. The source test now requires every directory write to be in `directory.go`, with no listed exceptions, and was shown to fail on an added stray write.
- **D10. `bulkimpact.go` not changed:** `everyone()` already pages through the now indexed `Users` and `Members`.
- **D11. Vertical spacing:** the scope line and the band take the existing 32px row gap of the shared `.live-users` grid (`live.css`, not changed) instead of the spec's 16px and 24px margins.

# Plan-text changes for root

1. D2: "at most 256 IDs" instead of "PageSize×4 (256)".
2. D4 and D5: the read's record is `x-reads`; add `next_login` to the D2 response list.
3. D9: `self.go`, `signin.go` and `seed.go` were added to the Part A allowed files by root for the three one-line switches.
4. D6: say where `user_filter_scan` is shown (today only as `scan_budget` on the page).
5. Visual spec §4.2/§14 widths and wrap results: membership 336px and period 242px after the technical-writing strings; 1280 rows are language-dependent (EN/AR `1122233`, KO `1112222`).
6. Plan copy table: the TW review rows replace `membershipEnded`, `setLink`/`resetLink`, `search`, `departedRow`, `jobsNote`, period options, the count line with zero rows, the announcement and the Invitations set-link cell; the method line is built from `auth_methods`.
7. L2 versus visual spec §9 ("keep the previous disclosure while a read is in flight"): root's L2 passes no loaded people during a read, so the not-loaded disclosure lists every selected person until the read completes. Spec text should follow L2. `Selection.tsx` (a Part B file) was not changed, so the bar still shows "Select the 0 loaded people" rather than hiding the control.

# Residuals

The plan's R1–R11 stand. Added:

- **R12.** Withdrawn: the D9 gap is closed.
- **R13–R14** are closed by R4 and TW row 10 (no failures on the fixed build).
- **R13.** A long login without break opportunities wraps inside the word in the user column at 1440 (pre-existing, `live.css`).
- **R14.** The inherited repeated-block check flags the Invitations tab when the invitation and the set-password link expire at the same instant; the two columns are separate facts. Root decides whether to keep the second date.
- **R15.** EN, KO and AR strings marked [TW] in the visual spec are implementer drafts; no technical-writing or native-language review has run. No ui-designer render review and no owner acceptance.
- **R16.** The browser harness has 304 people; 100,000 is measured in Go tests only.

# Planning cross-check

| ID | Source | Requirement here | Evidence in this build | Status after Part A |
| --- | --- | --- | --- | --- |
| M01 filters | `M01/spec.md:61` | Workspace, status, auth method, role, expiry | `membership`, `status`, `role`, `period`, `signin`, method as text and refused; tests and QA | covered; workspace limited to one (R1); method not connected (R2) |
| M01 search and sort | `M01/spec.md:63` | Server search, stable sort, selection by canonical ID | `q` over login, email, ID with case folding; login order with user ID tie-breaker; selection kept by user ID across filter changes (QA G) | covered |
| M01 tabs | `M01/spec.md:79`, `:81` | Filters and selection kept across tabs | Link parameters; QA G "filters survive the membership tab"; fixed invitations chip; jobs note | covered |
| M01 scope | `M01/spec.md:25` | Scope fixed and checked on the server | `workspace` refused; `answer.Workspace() == bootWorkspace`; scope line; two installations never mix | covered |
| M01 states | `M01/spec.md:89-98` | Empty, error and the rest distinct from zero | Filtered empty, stopped empty, unfiltered empty, next-page failure, list changed, restart, denied (QA F/G) | covered |
| M01 large data | `M01/spec.md:104` | Cursor; separate page and filter-wide selection; fixed manifest | Cursor, budget, restart and list-changed notice built; filter-wide selection is Part B | partly covered (read half); rest after Part B |
| retention-access | `retention-access.md:65` | Search, status, membership, role, method, sessions; 100,000 with search and cursor | Filters and search; 100,000 measured | covered except sessions (M05) and method (R2) |
| IAM-07 | `retention-access.md:116` | 100,000 isolated, resumable lookup | 100,000 pages at about 0.25 ms; isolation tests; cursors resume; restart answer | lookup covered; invitation, sync and bulk at 100,000 not in Part A |
| UI-04 | `ui-design.md:547`, `:412-424` | Filter, page and total bases agree; nothing loaded whole | Server pages of 50, budget, counts never claim totals, chips and Clear all, reference time, no client filtering | covered for M01 reads; selection totals after Part B |
| Profile minimisation | `retention-access.md:76` | No unnecessary profile data | Departed rows carry no login or email; login not kept for ordering | covered |
| HB-03 | `requirements.md:53` | Scope checked on the server | D1 tests | covered |
| W03 / W07 | `ui-design.md:270-292`, `:366-379` | Band with search and filters | Band built; "select all N" is Part B | partly covered |

## Root final verification (2026-11-18, rebased on main 8d673f1)

- Reviews: change review round 1 PASS (1 Medium, 4 Low) → round 2 PASS (lows fixed); render review round 1 FAIL → round 2 FAIL → round 3 PASS; technical-writing review round 1 FAIL → rounds 2–3 FAIL → round 4 PASS. Every finding fixed or recorded in this file.
- After rebasing onto HYB-303, HYB-295, HYB-325 and HYB-326, root re-ran: gofmt clean; `go vet ./...` clean; `go test -short ./...` all packages ok; `-race -count=2` identity and bulkrevoke ok; coverage server 100.0%, identity 99.0%, bulkrevoke 100.0%; console tsc clean, npm test 765/765, build ok; OKF status test ok after setting these three documents to `stable`.
- Browser QA and regressions are the implementer's round-3 runs (results above); the rebase brought no console change into these screens.
- Housekeeping: this change also deletes `tests/docs_format/__pycache__/test_okf_status.cpython-311.pyc`, a generated file committed by mistake in HYB-326 (PR #226) when the new test was run before staging.
