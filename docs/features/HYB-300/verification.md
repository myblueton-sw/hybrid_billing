---
type: VerificationRecord
title: HYB-300 part A verification of O01 server filters, search and paging
description: Formatting, vet, Go tests with race and coverage, benchmarks, mutation, console type check, unit and render tests, build, browser QA in EN, KO and AR at 1440, 1280 and 390 for every state of the visual spec, and the P01 to P05 regressions for HYB-300 part A. Run on mac_mini on 2026-11-18 with synthetic sample data only.
status: stable
sources:
  - resource: plan.md
  - resource: visual-spec.md
  - resource: references.md
  - resource: ../HB-272/plan.md
  - resource: ../HB-273/plan.md
---

# Scope

This record verifies part A of [HYB-300](https://linear.app/hybrid-billing/issue/HYB-300) against its [plan](plan.md) ("Part A", "Evidence and decisions", the copy rows and "Verification common to all parts") and the final [visual spec](visual-spec.md). Parts B and C (HYB-318, HYB-319) are not built and nothing of them is rendered. Every check ran on mac_mini (macOS, Node 26, the repository Go toolchain, headless Chromium through Playwright). The API ran from this branch with the sample seed, or with `--seed none`. No real account, customer data or credential was used. Results are as observed; a check that was not run is listed under "Not done".

Implementer: Claude (implementer role). This record is not a review PASS; the design review, change review, ui-designer render review and technical-writing review are owned by root.

# Planning cross-check

| ID | Source | Requirement here | Where it is met | Check | Status |
| --- | --- | --- | --- | --- | --- |
| O01 filters | `O01/spec.md:61` | As-of date, legal entity, relationship kind, unattributed, provider | `on`, `entity`, `kind`, `unattributed` on the organizations read; the provider is refused `unsupported_capability` and reported `provider_filter: "not_connected"`; the console band shows the four controls and the provider as text | `TestOrganizationFiltersOneByOne`, `TestOrganizationFiltersCombined`, render tests, browser "provider line", "provider is not a control" | covered; provider not connected |
| O01 search | `O01/spec.md:63` | Server search, stable sort, selection by canonical ID | `q` (1-64 bytes, ID or label, case-insensitive); rows sorted by member ID with an `after` cursor on one pinned revision; selection kept by ID in the `sel` link parameter | `TestOrganizationFiltersOneByOne` (`q` rows), `TestFilteredPagesMatchABruteForceFilter`, browser "search by label", "selection in url" | covered |
| O01 tabs | `O01/spec.md:77, 79` | Filters and selection kept across tabs | Filters in `entity`, `rk`, `ua`, `q`; selection in `sel` | browser "filters survive the tab change", "selection kept across tabs, not on this page" | covered |
| O01 states | `O01/spec.md:89-96` | Loading, empty, partial, error, revoked | Per-view loading sentence; filtered empty distinct from unfiltered empty; next-page failure keeps rows; first-page error; denied block without the band | browser states in the table below | covered |
| O01 large data | `O01/spec.md:102` | Cursor reads, no full load | Accounts, groups and history read server pages (50, 10 for member search); scan budget with `scan_stopped`; joins by `member=` for at most 50 IDs | `TestScanBudget`, `TestPinnedPagesAcrossAnActivation`, browser B (136 accounts, budget 100) | covered for accounts, groups and history; the tree still loads on the client (owner question Q1) |
| UI-04 | `ui-design.md:547` | Filter, page and selection agree; no full load | As above; counts never claim a total | render test "paging controls…", browser "count reads all shown", "more may exist" | partly covered: tree deferred (Q1) |
| HB-01 | `requirements.md:19-35` | Workspace, legal entity and group distinct | `entity` must be a legal entity party of the caller's workspace; every read uses `answer.Workspace()` | `TestOrganizationEntityOfAnotherWorkspaceIsRefused`, `TestTargetsAcrossWorkspacesAndFailures` | covered for part A |
| HB-02 | `requirements.md:37-51` | View group separate from hierarchy and payer | View groups listed by the targets read; members by `target=`; the shared account appears in both groups | `TestOrganizationFiltersOneByOne` (`target` rows), `TestTargetsRead`, browser "member counts", "group members with the entity filter" | covered for part A |
| HB-03 | `requirements.md:53-65` | Scope checked on the server | All filtering on the server; reads admitted as before (installation administrator) | `TestTargetsRead` (member and no session refused), browser "member denied, no band" | partly covered: admin-only (residual) |
| D3 account set | plan D3 | Accounts without a timeline of the kind are not omitted | `subject=account` pages the union of non-party members across all kinds | `TestOrganizationFiltersOneByOne` (Accounts 01, 02, 05, 06), `TestSuspendedEntityAndAccountSet` (payer-only account) | covered |
| D3 scan budget | plan D3, Setting | `org_filter_scan` 1000, 100-10000, start flag | `OrgSettings`, `-org-filter-scan`, `scan_budget` in the read | `TestOrgSettings`, `TestOrgSettingFlags` | covered |
| D3 cost | plan D3, Module listings | Sorted per-revision index, `sort.Search` per page | `index` built once per revision under `sync.Once`; `Timelines`, `Targets`, `Members`, `IsTarget` read it | `TestIndexIsBuiltOncePerRevisionUnderConcurrentReaders`, `TestConcurrentFirstReadsOfANewRevision`, benchmarks | covered |

# Commands and results

GOCACHE pointed at the session scratchpad cache; mutation used its own cache, removed afterwards.

| Command | Result |
| --- | --- |
| `gofmt -l apps modules` | no files listed |
| `go vet` on `./apps/api/...` and `./modules/relationships/...` with GOOS darwin, linux and windows | clean |
| `GOOS=linux GOARCH=386 go build ./apps/api/... ./modules/...`; `GOOS=windows GOARCH=amd64 go build …` | both built |
| `go test -count=1 ./...` | all packages ok (round 1: 71 of 71; round 2 on 8d65a5f: no failures) |
| `go test -race -count=1 ./modules/relationships` | ok (15.8 s; round 2: 14.9 s) |
| `go test -race -count=1 ./apps/api/server` | ok (352.8 s; round 2 on 8d65a5f: 333.3 s) |
| `go test -race -count=40 -run 'Index|Members|Timelines|Concurrent|Organization|Scan|Targets|Pinned|Suspended|Brute|SeveralChunks|OrgSettings'` on both packages (includes concurrent first readers of a new revision's index and of the shared empty revision) | ok (relationships 465.3 s, server 165.0 s) |
| `go test -coverprofile … ./apps/api/server` | 100.0% of statements |
| `go test -coverprofile … ./modules/relationships` | 100.0% of statements |
| `go test -cover ./apps/api/cmd/hb-api` | 92.5% on 8d65a5f; every new function at 100%, `main` 0% (unchanged) |
| grep for the floating-point type word in the changed Go test files | none |
| `npx tsc --noEmit` (apps/console) | clean |
| `npm test` (apps/console) | 671 tests, 671 pass after the round-3 fixes on base 8d65a5f (new: `live-render.test.tsx`, registered in `package.json`) |
| `npm run build` | built; the existing chunk-size warning only |

## Benchmarks

`go test -run XXX -bench 'MembersPage|TargetsPage' -benchtime 20000x ./modules/relationships` (ns per page of 100 IDs after the first build of the index; not asserted):

| Benchmark | 10,000 members | 100,000 members |
| --- | --- | --- |
| `BenchmarkMembersPage` | 441.9 ns | 401.4 ns |
| `BenchmarkTargetsPage` (one target per 10 members) | 365.6 ns | 356.2 ns |

The page cost does not grow with the revision size, as `sort.Search` on the cached index intends. The one-time index build is excluded. Numbers are from an Apple M4 with no other load at the time.

# Browser QA

Runner: scratchpad `qa300.mjs` (extended from `qa273.mjs`), three runs against a live API:
- **A**: `hb-api` from this branch, sample seed, default settings. Functional checks plus every state at EN, KO, AR × 1440, 1280, 390.
- **B**: a QA harness in a scratch rsync copy (not in the worktree) that starts the same server with the sample seed plus 130 extra synthetic accounts (`sample-acct-x000`…`x129`, every third in the north view group) and `org_filter_scan` 100, and can activate one more plan on request. Used for Load more, the scan stop, the next-page failure and the newer-revision notice.
- **C**: `hb-api --seed none` with a bootstrap administrator, for the unfiltered empty states.

Each screenshot is preceded by a layout check on the rendered page: page `scrollWidth`; every visible text element inside the work area must stay within it (except inside a table scroller, whose own overflow is also flagged); text ranges must not escape their element; visible controls, labels, cells and paragraphs must not overlap. Elements hidden by clipping (the stacked-table header at 390) are excluded, and the `.live-open` hit-area padding (`padding-block: 8px; margin-block: -8px`, an HB-273 rule) is measured on its content box.

| Run | Checks | Screenshots |
| --- | --- | --- |
| A | 423 pass, 0 fail | 121 |
| B | 182 pass, 0 fail | 57 |
| C | 55 pass, 0 fail | 18 |

States captured at every locale and width (visual spec §7): initial loading (held response), filter changed / loading, accounts, accounts with four filters and chips, accounts with a selected row, filtered empty, unfiltered empty (C), groups with the member filter and a selected group, history with a chosen member, history member search, bad link parameter, first-page error, next-page failure (B), Load more appended (B), scan stopped with zero rows (B), scan stopped then continued (B), groups outside list with Load more (B), newer revision (B), tree with its note, denied.

Functional checks in run A include: the tree band is only the note; entity North gives Accounts 01-03 with a chip, the link parameter and the announcement "3 shown; end of list. Filters applied: 1."; on 2026-09-30 and 2026-10-01 Account 02 stays under North (operations, then data platform); search by label; chip removal moves focus to the next chip, then to the search field, and announces "Filter removed. …"; view group plus only unattributed gives Accounts 01, 02, 05, 06 with a View group column; Clear all; filtered empty and its Clear all; the selected row and chip borders are neutral; reading order date → Show → Today → Reload → search → entity → kind → unattributed → rows; filters and selection survive the tab change; group counts "All members on …" are 1 and 2; South's members with the North entity filter show only Account 03; the outside list is the account set; allocation pool; "View history" focuses the history title; member search and choosing a member; a bad link value is removed with the notice; at 390 the band is collapsed, opens, stays open after a change with focus on the control, and reads "Filters · 1 applied"; a member session sees the denied block and no band.

Run B: the first page has 50 rows and "50 shown; more may exist"; Load more appends 50 and focuses the first new row and announces "50 more shown. 100 shown; more may exist."; an aborted next page keeps 100 rows with the alert, Try again appends the rest to "136 shown; end of list" and no Load more remains; `q=x12` with budget 100 stops with "No matches yet." and "Searched up to sample-acct-x093." and no empty claim; Continue searching finds `x120`…`x129` and reads "10 shown; end of list"; the outside list pages 50 then 90; an activation during the session shows the newer-revision notice.

# Regressions

After the round-3 fixes on base 8d65a5f, with the HYB-297-updated runners (`qa285fullon297.mjs`, `qa287fullon297.mjs`, `qa289fullon297.mjs` through `qa315reg.sh`; P05 through a copy of `qa315on297.sh` with `qa315bodyon297`, pointed at this worktree and binary; ports 4475-4479):

| Screen | Groups | Result |
| --- | --- | --- |
| P01 (`qa283full.mjs`) | X | 0 failures of 46 |
| P02 | A, B, C, D | 162, 32, 10, 16 pass; 0 fail |
| P03 | R, A, B, C | 288, 23, 12, 24 pass; 0 fail |
| P04 | R, A, B, C, D | 559, 28, 20, 46, 11 pass; 0 fail |
| P05 | R, F, A, B, C1, C2, C3, P (sample); E (none) | 481, 132, 17, 9, 2, 3, 30, 12, 142 pass; 0 fail |
| O02 (`qa296.mjs`) | X | 209 pass, 0 fail |

The P02-P05 failures recorded in round 2 came from the pre-HYB-297 runners (they also failed on a clean export of main 8d65a5f); the updated runners pass. The last O01-only CSS change (one-line ID-fallback names) came after this run and does not touch those screens. The older O02 runner `qa275.mjs` is superseded by `qa296.mjs`. M01-M05 runners were not run.

# Mutation

`mutate300.py` (scratchpad) mutates every changed or new line of `listing.go`, `registry.go`, `organizations.go`, `orgfilter.go`, `orgtargets.go`, `orgsettings.go`, `server.go`, `boundary.go` and `main.go` on an rsync copy (operator, comparison, constant, condition and statement-removal swaps) and runs the package tests (the server package with the organization, scan, target, settings and property tests).

| Pass | Mutants | Killed | Did not compile | Survived |
| --- | --- | --- | --- | --- |
| Full pass | 354 | 291 | 50 | 13 |
| Survivors re-run with the full package suites after the added tests | 13 | 11 | 0 | 2 |

Tests added for the killed survivors: scans across several 100-ID chunks for members, accounts, a target's members and the targets read (`TestScansAcrossSeveralChunks`, targets across chunks); `has`, `stopped` and `ended` asserted false on every error return; a pinned targets read at the active revision; an exact account lookup returning the requested kind's targets. An earlier pass also showed two redundant branches (the empty-search shortcut and a party early return in the exact lookup, both re-checked by `match`); they were removed rather than tested.

Round 2: the 13 mutants on the lines changed by the review fixes (`member` with `limit`, the targets read's `provider` refusal) were all killed, and the earlier `Active`-error survivor in `readTargets` is now killed by a direct pinned-read test.

Remaining survivor, with the reason no input distinguishes it:
- `orgfilter.go` `if c.targets == nil` → `if true || …`: recomputes `MembersOn` for the same immutable pinned revision, target and day on every chunk instead of once per request. The rows, cursor and stop are identical; only the cost differs.

# Review fixes (round 2, 2026-11-18)

Root rebased the change onto main 8d65a5f (HYB-297). Every item of the round-2 fix list was applied:

| Item | Change | Evidence |
| --- | --- | --- |
| R-M1 dates and headers broken | `<time>` in tables `white-space:nowrap`; table headers `overflow-wrap:normal`, `word-break:normal` outside Korean (also against the shared `#content [data-remaining-screen] th` rule); tables `min-inline-size:40rem` inside their own `overflow-x:auto` wrapper; the list column and its sections `min-inline-size:0` so the table never pushes into the panel; header dates wrapped in `<time>` | New layout checks: a `<time>` on more than one line and a header word split across lines. Proof run (EN, AR × 1440, 1280; accounts, accounts with a kind column, groups): the round-1 build fails 6 of 12 screens (23 split dates, the header "Organization" broken); this build passes 36 of 36 |
| R-m1 provider spacing | `.o01-live .o01-provider` margin | renders |
| R-m2 search cancel glyph | suppressed on O01 search inputs | renders |
| R-m3 tree IDs wrapping | ID column `minmax(22ch, max-content)`, IDs `nowrap` at 768 and above | tree screenshots at 1280 |
| R-m4 control heights | search input and selects `block-size:44px`, same label line height | renders |
| Groups view unattributed | control and chip removed; `appliedKeys` groups `["q","entity"]`; a group with no matching members reads "No members of this group match the member filters on {date}.", without filters "No members in this group on {date}." | render test, browser "no unattributed control in the groups view", "group with no matching members" |
| Deviation 8 | parent fact shows muted "Loading…" while the exact lookup is in flight | code |
| Not-loaded line | plain first-line `p`, no box | renders |
| C-M1 false revision reset | `pinnedRead`: a refused read is retried once without the pin; only if that succeeds is the revision reset claimed; otherwise the link filter (`entity`) or `member` is removed with the bad-link notice. `sel` and `member` are checked against the ID pattern before any read | model test "a refused read claims a revision reset only when…"; model test for `member=bad id`; browser "a unit entity in the link is removed, not a revision reset", "a bad member in the link is removed" (both with `revision=1`) |
| C-M2 | as the groups-view row | as above |
| C-L1 | direct `readTargets("bad ws", pinned revision 1)` test | the survivor is now killed |
| C-L2 | `limit` with `member` refused `invalid_request` | `TestOrganizationFiltersCombined` |
| C-L3 | the targets read refuses `provider` with 501 | `TestTargetsRead` |
| C-L4 | a legacy history `kind=` becomes `rk` once and is removed from the link; "View history" keeps the current `rk` | model test `legacyKind`; browser "legacy history kind becomes rk", "view history keeps the relationship kind" |
| TW F1–F6 and the other rows | all copy rows adopted as proposed; loading sentences per list; zero rows show no count and announce "No items shown."; header "Organization since"; screens.json KO scope label 공급사; glossary O01 section updated | render and model tests; browser copy checks |
| Kind-only filter hypothesis | Confirmed: in the accounts view the relationship kind does not narrow the account set unless "Only unattributed" is on. `narrowing()` excludes a kind-only selection, so the filtered-empty sentence is not shown for it. The scan-stop ID is the last examined member (`TestScanBudget`: budget 8 stops at `sample-unit-north-ops`, the eighth candidate) | model test `narrowing` |

Column widths differ from the review's 24/20/18/22/16: after the header rename "Organization since" did not fit a 16% column of a 40rem table, so the five-column table uses 22/20/16/20/22 and the four-column table 28/24/24/24. The proof run above is with these widths.

# Review fixes (round 3, 2026-11-18)

| Item | Change | Evidence |
| --- | --- | --- |
| Render MAJOR: last column cut at 1280 | The fixed 40rem minimum and the percentage colgroups are gone from the accounts and groups tables. Tables use `table-layout:auto` with `inline-size` and `min-inline-size` at 100%. `<time>` stays on one line, so only the date column keeps a max-content minimum. At 768px and above, cells and their `<bdi>` wrap only at word boundaries (`overflow-wrap:break-word`, against the shared `anywhere`). Cell inline padding is 8px. The overflow wrapper stays only as a fallback | New QA check: a table cell extending past its wrapper's visible box at the default scroll position. Before the fix (EN/AR × 1440/1280/390; accounts, kind column, groups, selected panel) 8 screens failed, all at 1280 (`qa-P2-before-celltcut-fix.out`). The check for words broken inside body cells, extended to `td`, caught "Holdin/gs" on an intermediate build. After the fix 72 of 72 pass (`qa-P4-after-fix.out`, final build) |
| Render minor: account IDs broken at 1280 | ID lines in tables use `white-space:nowrap` at 768px and above | same runs |
| Names that fall back to an ID | Shown on one line at 768px and above, like the ID line, so `sample-acct-x001` never splits at its hyphens (found by run B at 1280) | run B 182 of 182 |
| N1 | `refusal()`: a refused read drops the linked `entity` or `member` only when one is applied; otherwise the normal error state shows | model test "a refused read drops a linked value only when one is applied" |

# Deviations

- **Selection in the link (`sel`).** The console remounts O01 on every tab change (`RemainingEntryShell` is keyed by the tab in `App.tsx`, outside the allowed files), so component state cannot carry the selection across tabs. The selected canonical ID is kept in a `sel` link parameter, read on mount. Not in the plan's parameter list (`entity`, `rk`, `ua`, `q`).
- **History kind is the shared `rk`.** The history view's relationship kind is the band's kind select and therefore the same `rk` parameter as the accounts view; "View history" keeps it. A legacy `kind` parameter is mapped to `rk` once.
- **Groups view search applies to the group list only**, as the visual spec labels it; legal entity applies to the member lists. "Only unattributed" is not offered there (root decision, round 2).
- **Setting rows.** `OrgSettings` mirrors `AccessSettings` (bounds, start-flag source in `Given`), but no read shows its rows in part A; the shared `settingRow` builder is left to part C, which adds the only reader (the drafts read).
- **`provider` refusal status.** `failCode` has no rule for `unsupported_capability` and `errors.go` is outside the allowed files; the handler answers through `failModule` with 501, retry false, next action none, which matches the command framework's rule for that code.
- **`targets` read refuses organization-only filters** (`subject`, `entity`, `unattributed`, `member`, `target`) with `invalid_request` instead of ignoring them.
- **CSS outside the band.** A shared rule `.work-domain table { min-width: 580px }` made the accounts and groups tables scroll inside the 568px list column at 1280; O01 now sets `min-width: 0` for its tables. The tree grid (unchanged since HB-273) overflowed its column at 1280; its name and ID columns now shrink (`minmax(16ch, 1fr) 12ch minmax(0, 22ch)`).
- **`provider` on the targets read** is refused the same way (round 2).

# Residuals and not done

- The client-built tree remains (owner question Q1); part A changes only its band.
- "Labels changing between pages while paging with `q`" needs part C's label map; not testable in part A.
- `target=` reads the whole group's members on the day (`MembersOn`) once per request before paging; large groups cost O(members) per request. A paged module read by target was not in the plan's module list.
- No external console was reopened beyond `references.md`.
- M01-M05 regressions and the O03 runner were not run; the shared files they could see changed are `admin.ts` (two additive methods and query fields) and O01's own CSS, which is scoped under `.o01-live`.
- Root still owns the design review, change review, ui-designer render review against these screenshots and the technical-writing review of the new EN, KO and AR copy.

## Root final verification (2026-11-18, on main 8d65a5f)

- Rebased by root from 867a3e9 onto 8d65a5f (HYB-297); conflicts only in `apps/console/package.json` (test list, both kept) and the glossary (both sections kept).
- Root re-ran: `gofmt` clean; `go vet` and `go test -count=1 ./apps/api/... ./modules/relationships` ok; earlier root run: coverage 100.0% on `apps/api/server` and `modules/relationships`; `tsc` clean; `npm test` 671/671; build ok; no changed file over 1,500 lines.
- Root re-ran the regressions with the HYB-297-updated runners: P02 A–D, P03 R/A–C, P04 R/A–D, 0 failures. The implementer's earlier P02–P05 failures came from runners that predate HYB-297 and failed identically on main.
- Reviews: design rounds 1–2; change review FAIL then PASS (M1 pinned-read refusal, M2, L1–L4); render review rounds 1–3 (dates split, then the last column cut at 1280, then PASS); technical-writing review FAIL then PASS. The provider line in the groups view is kept (visual spec §15 D11).
- Residuals: the tree is still built in the browser (HYB-320, owner acceptance of the deferral needed); labels between pages wait for Part C; `target=` costs O(group members) per request; the selected row's first cell shows the shared button hover tint after a click, identical on main (pre-existing); M01–M05 and O03 runners not run for this ticket; KO and AR copy remain drafts for native review.

