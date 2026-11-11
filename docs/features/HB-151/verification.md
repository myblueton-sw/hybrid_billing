---
type: VerificationRecord
title: S04 source-field and row-detail verification
description: Records what was executed and reviewed for the S04 source-field explanation on the retained sample, with results, limits and items that remain unavailable.
status: draft
---

# Delivered behavior

[HB-151](https://linear.app/hybrid-billing/issue/HB-151/refine-s04-source-fields-and-row-detail-to-separate-source-values-from) refines S04 from main `9712d7f` using the retained public and synthetic sample verified in HB-149. Root (Claude Code) owns planning, implementation, terminology writing, QA and integration; the ui-designer role led composition and compared the final renders; a separate read-only reviewer reviewed the design, the actual change and the fixes. Machine: mac_mini. See the [plan](plan.md) and [reference evidence](references.md).

The source-fields tab groups its columns by origin, counts missing values per row across the 12 nullable source fields and offers a closed-by-default field mapping reference. The row detail lists every mapped FOCUS column with its exact value, labels Missing and Exact zero as separate text states, pairs quantities and unit prices with their own units, and separates values derived by Hybrid Billing, delivery lineage and the sample annotation. Nulls on the screen read Missing instead of Unavailable, and intervals are named charge period and billing period. `/api/focus-overview`, its DTO, validator, projection identity, permissions and storage are unchanged. No request, storage key, URL parameter, link or log was added.

# Retained sample facts used

Read on 2026-11-11 through the console proxy: 1,121 authorized rows, 78 account scopes, FOCUS version list `1.0`, currencies USD and CNY, tenant time zones `Asia/Seoul` and `UTC`. `contracted_cost` and `contracted_unit_price` are missing on 7 rows; `consumed_quantity`, `consumed_unit` and `list_unit_price` on 1 row; 13 rows carry negative costs; 2 rows are CNY-denominated; consumed and pricing units differ on 116 rows; the charge-start month differs from the UTC month of the charge period start on 17 rows; one row's charge period starts outside its billing period. Counts come from one full paged read by root and were not independently recomputed.

# Executed checks

| Check | Actual outcome |
| --- | --- |
| Field-model unit tests (`fields.test.ts`) | PASS 10: mapping mirror equals the tracked contract for versions 1.0, 1.2 and 1.3; disjoint origin lists covering all 25 row keys; missing, zero (`"0"`, `"-0"`), negative and 33-digit coefficients; missing-field listing; unknown versions reported, not guessed; provenance allowlist; UTC instant parsing; UTC-month comparison; charge start against the billing period |
| Console unit suite (`npm test --prefix apps/console`) | PASS 288, previously 278 |
| TypeScript check, Vite build, `tools/verify-build.mjs` | PASS; 438 artifact hashes and licenses; existing large-chunk advisory remains |
| Isolated backend regression (`tests/focus_overview`) | PASS 36; unchanged backend, regression only; no retained-store reset suite |
| Browser suite `apps/console/tools/collected-source-qa.mjs` on the retained sample | PASS; 6 existing variants and 6 source-field variants (EN/KO/AR at 1440 and 390); 0 page errors; see below |
| Retained source, control and prior-result files | PASS 93 file hashes equal to the HB-150 preservation manifest after all reads |

Browser cases on real rows: the first Oracle row end to end (BilledCost `USD 0.24`, EffectiveCost `USD 0.00` with Exact zero, ContractedCost and ContractedUnitPrice Missing without a currency, charge-period sentence, time-zone statement, six not-carried items, no link in the detail); a Google Cloud row with `seconds` against `month` units; an AWS row with missing consumed quantity and unit, zero pricing quantity and negative costs `USD -2.6137` and `USD -3.00`; a CNY row; a row whose charge-start month `2024-10` differs from UTC month `2024-09`. Interaction cases: reference closed by default with 25 field rows in four groups; page, provider, refresh and tab changes close the detail; synthetic 403 and 503 responses remove rows, totals, reference and detail and recover through restart; Escape and both close buttons return focus to the originating row; the S03 link opens S04 without a period filter. A routed response with versions `1.0` and `9.9`, a single tenant time zone and a non-UTC instant verifies the degraded statements. All six source-field variants check no page overflow, 16px values, left-to-right money, exact Missing and Exact zero strings per language, the selected-row text marker and computed borders (no radius or shadow, channel spread at most 32); the detail table staying inside its container; the 390px variants check 44px close buttons. The suite asserts that the only non-200 responses are the expected stale-snapshot 409, foreign-account 400 and the two synthetic failures.

Host: macOS arm64, Node 26.7.0, Python 3.11.7 from the prepared environment, Playwright with Chromium headless shell build 1243. Other operating systems, browsers and assistive technology are unverified. Captures and `report.json` stay local under `output/validation/HB-151` and are excluded from Git.

# Findings corrected during verification

- Global table rules (nowrap, sticky headers, row backgrounds, fixed heights) leaked into the new tables and caused a 269px page overflow at 390px and overlapping columns at 1440px. Scoped resets fixed both before any review.
- The suite found the detail close button at 36px on mobile; it is now 44px.
- Root's assumption that a detail could stay open across a tab change was wrong: the application remounts the screen when the tab changes. The unreachable focus fallback and history-view selection code were removed and the suite asserts the actual behavior.
- Three suite failures were root's own selectors (two alert elements, a hidden navigation link) and are not product results.

# Reader admission limit observed

The unserialized suite failed intermittently at different existing steps. Recorded responses showed the cause: a query change produces one aborted and one completed read of the same path, the local reader keeps working on the aborted request, and it admits two concurrent requests, so a third receives the deliberate `source_busy` 503. The screen fails closed and offers restart. This predates HB-151; `App.tsx` and the fetch effect are unchanged. It is tracked as [HB-152](https://linear.app/hybrid-billing/issue/HB-152/investigate-duplicate-s04-reads-and-user-visible-source-busy-on-quick) and is not fixed here.

The suite now sends its browser's reader requests one at a time and records the slowest read. Consequences: it no longer exercises concurrent reads, it asserts no latency bound (waits are 120 to 150 seconds), and a real operator can still meet `source_busy` after two quick actions. Single reads took about 2 seconds when idle; the slowest read in passing runs ranged from 6.1 to 13.6 seconds. This is not a throughput or scalability result.

# Independent review

| Review | Result |
| --- | --- |
| Design review of the first plan | FAIL without blockers: provenance misclassified, tenant and version wording, overclaiming handling statements, terminology collisions, currency beside missing values, focus-return target. Plan revised |
| Actual change and source-string technical-writing review | FAIL without blockers: two document overclaims, selected state color-only in two views, unused tested function, missing degraded-path tests, terminology residue, Arabic misreadings, ambiguous scope wording. All addressed |
| Narrow re-review of the fixes | PASS with this record pending; confirmed money semantics, fail-closed clearing, no new exposure, and that removing the unreachable selection code is correct |

The reviewer read files only and executed nothing; all executed results above are root's. Not read by the reviewer: storage round trip, `intake.py`, `collection.py`. Whether a delivery pins the mapping content used at collection time remains unverified and is not claimed by the screen.

# Design and wording review

The ui-designer role compared 20 final renders with its checklist: conditional PASS, no blocking defect. It confirmed the six source columns fit at 1440px without scrolling, non-wrapping table timestamps, the Missing and Exact zero treatment, section order, neutral separators, Arabic mirroring with left-to-right codes and money, and stacking at 390px. Its conditions were applied afterwards: the 64-character row ID breaks only at its midpoint, mobile reference captions use the full width, and note and field columns keep their minimum widths. The first version of those corrections widened the source table past its container at 1440px; root saw it in the render, rebalanced the column widths, measured table and container widths in nine language and width combinations, and added a suite assertion. These corrections were verified by root's rerun and rendered inspection, not by a second designer pass. It did not see a selected-row render or the scrolled second and third column groups; the suite asserts the selected-row marker.

Rendered wording was inspected by the same role in EN, KO and AR. Open questions left for a native or practitioner reviewer: Arabic labels for unit prices and list and contracted cost, "التكلفة الفعلية" for effective cost, and "وسم العينة"; Korean 결제 for the account against 청구 for currency and period, which predates this change. AI design and wording review is not owner FIX, native-language acceptance or practitioner approval.

# Unavailable and deferred

Stated on the screen and not inferred: original column text, missing reason, row mapping version when the scope has several, row tenant and time zone when the scope has several, source delivery window, row-level transformation history, and a link to original evidence. The history tab stays unavailable and no S05 link exists because the query supplies no authorized locator. A row-level evidence contract (missing reason, tenant, version, delivery window) is a separate scope decision needing design plus tenant and financial review. Global service, model, quality and attribution filters, export, HB-34 workers and HB-39 large-volume budgets remain deferred.

# Acceptance

Parent HB-93 remains awaiting explicit owner screen acceptance. This change alters the S04 candidate revision, so no earlier acceptance carries forward. No test, build or review created owner acceptance. No ingestion, reset, reimport, schema or permission change, live account, customer data, deployment, payment, message or Graphify measurement occurred. Merge, read-back, ticket completion and machine release are recorded in the ticket only after they occur.
