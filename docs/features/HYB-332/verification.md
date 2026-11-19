---
type: VerificationRecord
title: HYB-332 M01 Revocation jobs table at narrow widths, verification
description: The CSS change that stacks the M01 Revocation jobs table into labelled blocks below a 600px list container, the investigation of the li.m01-item text-height report, the checks run with their observed results and the residuals. Written by the implementer; no change, render or technical-writing review has seen this build.
status: stable
tags: [HYB-332, HYB-299, verification, M01, bulk-revocation, responsive]
sources:
  - resource: ../HYB-299/visual-spec.md
  - resource: ../HYB-323/verification.md
---

# Scope

- Branch `fix/HYB-332-m01-jobs-table-narrow`, worktree `/private/tmp/hb332`, on main `e5f7751`. Nothing is committed. Observed on 2026-11-19 on the Mac mini (darwin/arm64).
- Scratchpad (`/private/tmp/claude-501/-Users-home-project-hybrid-billing/ff7545a1-604d-4178-88ba-25b4b88aab02/scratchpad`, `$SP`) holds the runners, outputs and screenshots. The API used in every browser run is `$SP/hb-api332`, built from this worktree.

# Causes found

1. **AR 390 overflow by 72px.** `.security-table-scroll table` has `min-inline-size: 430px`, and the jobs list container at 390 is 358px wide. The jobs table had no narrow layout, and `overflow-wrap: anywhere` on its `th` let header words break letter by letter.
2. **Stacked cells cut to 44px.** The global rule `td { height: var(--ui-row-h) }` is a minimum for a table cell but a fixed height for a cell shown as a block. The first stacking attempt clipped every jobs cell after 44px (observed in QA, `$SP/qa332f`). The existing stacked `.m01-settings` table at AR 390 had the same defect: long setting names spilled out of their 44px cells. The HYB-323 runner did not show this before because it keeps only the first four issues of a check.
3. **`span.m01-opener` "text escapes".** An inline span wraps the 44px opener button (padding-block 10px). The span's own inline box is 20px, so the button's text measured outside it. Nothing was clipped, but the wrapper box did not hold its content.
4. **`li.m01-item` text height vs box (EN 1440): not a layout defect.** A probe phase (`$SP/qa332probe.mjs`, phase I, output `$SP/qa323-hyb332b-I.out`) measured each text node of the first two items on the executed job:
   - every visible text node lies inside the item box (item 1: box 2129–2254, lowest visible text 2242);
   - all text measured outside the box sits inside the closed `<details class="m01-cleanup">`, which the browser lays out in a skipped state;
   - with the details opened, the item box grows (2129–2663) and contains all its text (2142–2651).
   The report comes from the runner, which measures `selectNodeContents` of the whole item, including the closed details. The product is not changed for this.

# Change

- `bulk.css`:
  - The jobs list container is named `m01jobs`. Below 599px it stacks the table like `.m01-settings`: `thead` is visually hidden; table, rows and cells are blocks; rows have a row separator; cells after the Job cell start with `attr(data-label) ": "` at 14px muted; the table's 430px minimum is lifted.
  - `overflow-wrap: anywhere` is removed from the jobs table `th` and kept on `td`. The headers of the jobs table and the settings table also set `overflow-wrap: normal` under `#content [data-security-screen="M01"]`. Without that, they inherit `anywhere` from `.remaining-security` and from the shell rule `#content [data-remaining-screen] th`. Phase V found this first: at container 616 the headers State (EN), الحالة (AR) and Source / المصدر were split inside the word.
  - Stacked jobs cells and stacked `.m01-settings` cells use `block-size: auto`.
  - `span.m01-opener` is `display: block`.
  - The open-job reduced rows still hide every cell except the Job cell. The list column of the split layout with a job open is about 435px at 1440, so those rows are stacked too. Stacked rows have `padding-inline: 12px`, so their text stays 12px inside the open-row fill (render review F1, 2026-11-19).
- `RevocationJobs.tsx` (`JobTable`, now exported) and `RolePeriod.tsx` (`Settings`): `role="table"`, `"rowgroup"`, `"row"`, `"columnheader"` and `"cell"` on the table elements. Stacked tables use `display: block`, which can remove the implicit table roles; the explicit roles keep table semantics (render review L1).
- New `jobs-render.test.tsx` (added to the console `npm test` list): every `tr`, `th` and `td` of both tables carries its role, in EN, KO and AR.
- `docs/features/HYB-299/visual-spec.md` §7.2: dated HYB-332 amendment paragraph. Its first sentence was replaced with root's wording after the render review (F2).

# Checks

| Check | Result |
| --- | --- |
| Console `npm test` | 795/795 |
| `jobs-render.test.tsx` before the roles (with `JobTable` exported) | 2 of 2 failing: 0 `role="table"` found where 1 was expected |
| `jobs-render.test.tsx` after | 2/2 |
| `tsc --noEmit`; `npm run build` | clean; passes |
| `qa332probe.mjs` (`qa323.mjs` plus session logging and phase I) G, baseline build (main, unchanged console), phases S M G I | S 22/22, M 1/1, G 95/101 (`$SP/qa323-hyb332b-*.out`): AR 390 overflow by 72, headers الحالة and الأشخاص broken, When header and cell cut, opener and item escapes |
| `qa332probe.mjs` G, first fix without `block-size: auto` (`$SP/qa323-hyb332f-G.out`) | G 96/101: the 72px overflow is gone; jobs cells cut at 44px (en, ko, ar 1440 and ar 390) |
| `qa332.mjs` (amended text-escape measurement, below) G, baseline build | G 99/101: AR 390 overflow and header/cell failures still reported, so the amended check still fails on main |
| `qa332.mjs` G, build with jobs `block-size: auto` but settings not yet fixed | G 100/101: AR 390 stacked settings cells' text escapes (text 2790–2861 in a box 2784–2828) |
| `qa332.mjs` phase V, build before the header fix (output `$SP/qa323-hyb332v-V.out`) | 4 failures: `header word broken` for State and الحالة (jobs) and Source and المصدر (settings) at container 616 in EN and AR; the item step stopped on a runner selector (the closed ended-jobs table) |
| `qa332.mjs` G, build before the 12px inset (render-review build), phases S M G V (ports 4579/8909, `$SP/qa323-hyb332pre-V.out`) | G 101/101, V 161/164: the 3 new inset checks fail. Stacked rows at EN and KO 390 had insets [0,0,0], and the open row at EN 1440 had [0,0,0,0,0] |
| `qa332.mjs` G, final build, phases S M G V (ports 4578/8908, `$SP/qa323-hyb332fin-*.out`) | S 22/22, M 1/1, **G 101/101**, **V 164/164**; insets at 390 [12,12,12] (EN and KO), at 1440 with a job open [12,12,12,12,12] |
| `qa323.mjs` G, unchanged runner, build before F1 and L1, phases S M G (ports 4573/8903) | S 22/22, M 1/1, G 97/101; all 4 failures are only `li.m01-item` closed-details reports (16 item entries in total, no other issue) |
| `qa294.mjs` regression S, X, F, G, W, final build (ports 4576/8906, rerun after F1 and L1) | 22/22, 27/27, 1497/1497, 70/70 (2 page errors `net::ERR_FAILED`, the same as HYB-294 r6 and HYB-323 round 2), 33/33 |
| `qa299` copy (`qa299on323.mjs`) A, C, B, D, E, L, P, N, build before F1 and L1 (ports 4577/8907; not rerun) | 334, 3, 5, 5, 53, 352, 1, 31: 784/784, 0 failed |

- **Phase V** (new, `$SP/qa332.mjs`; screenshots in `$SP/qa332v2` and `$SP/qa332fin`, output `$SP/qa323-hyb332v2-V.out` and `$SP/qa323-hyb332fin-V.out`). It runs the full `states323` set (overlap and clipping, cut cell, split date, broken word, escaping text, borders, header and cell parity, repeated text, AR isolates, word joins) on each state below, and also records the shape:
  - Revocation jobs and open settings at EN and KO 390: list container 358; jobs stacked with the label ("State: ", "상태: "); thead hidden; jobs and settings tables with 0px scroll.
  - The same at a list container of 584 (viewport 616) and 616 (viewport 648) in EN, KO and AR: at 584, both tables stacked; at 616, both tables shown as tables (settings width equals the list width), with 0px scroll.
  - Inset (added for F1): the text of each stacked row at EN and KO 390 starts at least 12px inside the row. At EN 1440 with a job open, the opener and the compact lines start at least 12px inside the open-row fill.
  - EN 1440 executed job: the item list with cleanup details closed, then with all 10 cleanup details open. In the open state, every item's full text range lies inside its box (0 outside).
- The shared-runner decision (root, 2026-11-19): the amended text-escape measurement is adopted.
- **Runner amendment** (`$SP/qa332.mjs`, a copy of `qa323.mjs`): the text-escape check measures the union of the visible text nodes, skipping nodes the runner's own `hidden()` already treats as hidden (closed `<details>` outside the summary, `.visually-hidden`, clipped) and nodes whose parent has `display: none`. The rest of the runner is unchanged. It still caught the 44px cut cells and the settings spill shown above.
- **Phase order:** G depends on the `qm…` people that phase M creates. A run of S then G without M stops at "fix for execution" (nobody matches `QM1`); this is runner order, not a product failure.
- **Screenshots** are in `$SP/qa332s2`:
  - `viewport-settings-ar-390.png`: stacked jobs row with الحالة / الأشخاص / الوقت labels and no horizontal scroll;
  - `viewport-jobs-filter-preparer-en-1440.png`: open-job reduced row with all lines shown.

# Residuals

- The unchanged HYB-323 runner (`qa323.mjs`) still reports the closed-details `li.m01-item` measurement in 4 checks; the adopted measurement lives in `$SP/qa332.mjs`.
- The reduced open-job rows were rendered only at 1440 (phases G and V).
- L2 (render review, recorded, no change): in the stacked form the People line can read as a double label, because the cell label precedes the equation sentence, which starts with its own count noun. Seen at AR 390 as «الأشخاص: 10 أشخاص: نجح 10، …» (`$SP/qa332s2/viewport-settings-ar-390.png`).
- The `qa299` copy and the unchanged `qa323.mjs` were not rerun after F1 and L1.
- No change, render or technical-writing review has been run.

## Root final verification (2026-11-19)

- Render review round 1 FAIL (F1 open-row inset regression at 1440, F2 amendment wording, L1 roles, L2 residual) → round 2 PASS. Root adopted the amended text-escape measurement (closed details, visually hidden, clipped and display:none text are skipped) because it still catches the real AR 390 defects on main.
- Root re-ran after rebasing on current main (package.json test list conflict with HYB-331 resolved as a union): tsc clean, npm test 798/798, build ok, OKF status test ok.
