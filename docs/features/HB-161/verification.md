---
type: VerificationRecord
title: U01 source rows and aggregates verification
description: Records what was executed and reviewed for the U01 source-row terms, aggregate states and monthly wording on the retained sample, with results, failed attempts, limits and accepted residuals.
status: draft
---

# Delivered behavior

[HB-161](https://linear.app/hybrid-billing/issue/HB-161/align-u01-source-rows-and-aggregates-with-the-s04-source-field-terms) refines U01 from main `9cb6f23` using the retained public and synthetic sample verified in HB-149. Root (Claude Code) owns planning, implementation, terminology writing, QA and integration; a separate read-only reviewer reviewed the plan before implementation and the change afterwards; the ui-designer role compared renders. Machine: mac_mini. See the [plan](plan.md).

Source rows show Missing for a null cost, service or subaccount, the shared headers, the provenance label instead of the raw class, and a text control "Charge period and delivery lineage" that opens a full-width row with the charge period in UTC, the delivery ID and the row ordinal. Every aggregate (summary, provider, account, month) goes through one sum component that reads "No values present" without currency when no value is present; partial sums carry one missing-count format, now also on the provider subtotal and the account amounts. The monthly table distinguishes a month without rows in the selection, a month without rows in the chart currency, all values missing, partial and complete; "Unknown" is no longer used for amounts. The provenance line lists each class with its count and the counts card is titled "Selected scope". `/api/focus-overview`, its contract and validator, selection, paging, permissions and URLs are unchanged.

# Retained sample facts used

Read on 2026-11-12 through the console proxy by root. The unfiltered selection has 1,121 rows in 78 account scopes over 2024-03, 2024-09 and 2024-10: 1,119 USD rows and 2 CNY rows, no missing billed cost, provenance 1,000 public sample, 115 public sample subset and 6 synthetic test data. The first row is an Oracle row with a long subaccount ID and UTC instants. Missing row values, an all-missing currency and a non-UTC instant are therefore covered by a routed response.

# Executed checks

| Check | Actual outcome |
| --- | --- |
| Console unit suite (`npm test --prefix apps/console`) | PASS 306 (302 before): a render test for the sum component, the chart states and the two absent-month cases |
| TypeScript check, Vite build, `tools/verify-build.mjs` | PASS; 438 artifact hashes and licenses; existing large-chunk advisory remains |
| U01 browser suite `apps/console/tools/overview-qa.mjs` on the retained sample | PASS on the final build with 0 page errors, after failed attempts described below. The unchanged suite passed on main `9cb6f23` before any edit, on its second attempt |
| S04 browser suite | PASS as regression on the final build |
| Isolated backend regressions (`tests/focus_overview`) | PASS 36; unchanged backend |
| Retained source, control and prior-result files | PASS 93 file hashes equal to the HB-150 preservation manifest after all reads |

Each added case starts from a fresh page load at its width with the monthly table opened. On real rows at 1440 in English: the row and account headers; the provenance line equal to the response's counts per class; the first row's charge-start month, billing account ID and provenance label; no in-cell disclosure; the control sets `aria-expanded`, leaves the row's column widths unchanged, shows the UTC instants, delivery ID and ordinal equal to the response, and closes when another row's control is used; no superseded word, raw class, raw column name or dash in the summary, provider, account and row regions or the monthly table body.

The routed response is patched from the live response of the same request and only for the unfiltered first page on the billed basis: the first row loses its billed cost, service and subaccount and gets a `Z` charge start, with the one-value difference applied to the totals, the row's provider, month and listed account; the first account loses its subaccount; CNY is emptied on the billed basis in the totals, every provider, every month and the listed accounts. In EN, KO and AR at 1440 and in KO and AR at 390, with CNY as the chart currency: the row values and the account subaccount read Missing; the summary has exactly one "No values present" and the missing counts expected per currency; the provider table has empty sums and counts; every CNY month reads "No values present" with "All values missing" and months without CNY rows carry the absent phrase; the reference row shows the instant as collected; no superseded word in three languages.

For every case, real and routed: no page overflow; no state, count, provenance, sum-phrase or month-phrase node inside any `bdi`; no row state attribute or class inside an aggregate region; no Exact zero note; no child of a table cell outside its cell; no instant or amount broken across lines; the provider and monthly tables neither clipped nor scrolling; each table header at the top of its region; the control at least 32px high, 44px at 767 and below; exactly one reference row, inside the viewport. Layout on real data, asserted as real by the first row's cost state and the absence of a raw instant, in EN, KO and AR at 1440, 1200, 1199, 1024, 768, 767 and 390. A failed read leaves no source value, sum, field or row node. The existing selection, paging, history, late-response, failure and foreign-scope cases and the six variants still pass.

Not asserted: the chart's decorative gap labels; the sticky reference list after scrolling the row table sideways (the list is checked against the viewport without scrolling); column-width stability outside English at 1440; the account amounts' counts when the first row's account is not on the first account page; a zero-row currency, which the sample does not contain. The row table scrolls horizontally at 768 and below, as before this change (up to 614px at 390 in English).

Host: macOS arm64, Node 26.7.0, Python 3.11.7 from the prepared environment, Playwright 1.56.1 with Chromium headless shell build 1243. Other operating systems, browsers and assistive technology are unverified. Captures stay local under `output/validation/HB-161` and are excluded from Git.

# Failed attempts and corrections during verification

- Reader contention. During this work the machine ran unrelated workloads (load average up to 35) and the local reader, which serves two requests at a time and answers `source_busy` instead of queueing, took up to 25 seconds per read. The suite failed more than ten times in its unchanged selection cases on `source_busy` or its 30-second wait, including on the baseline's first attempt. The reader code is unchanged and the screen issues the same single request per selection as before; the cause is the reader's slot limit under slow reads, not this change. The suite's wait helper now recovers from `source_busy` through the screen's own "Refresh query" action and reports the number of recoveries (2 in the final passing run); any other error still fails. A user who changes the selection quickly while reads are slow sees the same error and must refresh; that is existing behavior, recorded as a residual.
- A long subaccount ID overflowed its cell in the row table before this change; it now wraps.
- Opening the reference row moved the row's column widths, first through a changing control label and then through the spanning cell; the label is constant and the spanning content no longer contributes to column sizing.
- The first routed expectation read the first missing count instead of the count per currency. Suite error.
- The suite removed its route with a new function instead of the registered one, so the layout cases ran on the routed response. Found by both reviewers. The same reference is now used and the layout cases assert real data.
- Removing the provider table's scroll region to stop clipping detached its sticky headers from the table when the page scrolled. Found by root in a capture after the designer round. The region is kept and only its height cap is lifted; the suite asserts header position.
- A header-position assertion first flagged rows scrolled out of a capped table. Suite error.

# Independent review

Reviewer in each round: the repository's read-only `independent-reviewer` agent role (AI), 2026-11-12. It read files and executed nothing.

| Round | Result |
| --- | --- |
| Design review of the plan, before implementation | FAIL without blocker: 2 HIGH, 8 MEDIUM, 6 LOW. Resolved in the plan's "Design review resolution" table before code was written |
| Change review of the implemented diff | FAIL without blocker or HIGH: 3 MEDIUM. The no-op route removal, corrected as above; the missing glossary section, added; a wrap rule that also matched account sums, narrowed. LOW corrected: provenance classes listed separately; control named with glossary terms and described by its row's billing account ID; sum phrase sized to its secondary line; no empty count element; superseded-word and count assertions strengthened; a test title. LOW left: see residuals. No second change-review round ran on the corrections |
| ui-designer render comparison | PASS with conditions. Corrected: the monthly table's height cap that cut the table at 390; the provider table's clipping; the control's target size; the summary phrase at 20px; the chart's all-missing label in body colour and weight; a neutral surface for the reference row; reference-row column proportions; the control wraps below 1200. The captures the designer saw as real data were routed data, per the route error above; the corrected real-data captures were read by root, not by a second designer round |

# Accepted residuals

- AI wording review only. Open for native review: 값 없음 beside 값 누락 없음; Arabic "ضمن النطاق" without "المحدد"; "لا صفوف CUR"; the Korean subtitle mixing a sentence into a phrase list.
- Quick selection changes while reads are slow can surface `source_busy`, which the user clears with "Refresh query". Existing reader and screen behavior; a follow-up decision for the owner.
- The decorative chart marks a month without rows with a dash while the table gives the phrase; the chart is hidden from assistive technology.
- A currency with zero rows would read "No values present" beside "0 rows"; not emitted by the sample.
- All 25 row controls share one label and are distinguished by their description and table position.
- Help prose keeps the FOCUS column aliases; the company line renders empty for a null company; both predate this change.
- The demo components that the screen does not render are untouched.
- No owner FIX was created or carried forward by this change.
