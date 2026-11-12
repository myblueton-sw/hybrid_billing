---
type: ImplementationPlan
title: U01 source rows and aggregates in the S04 source-field terms
description: Defines how U01 presents its collected source rows, account mapping and aggregates with the shared source-field vocabulary and value states, without changing its projection, selection, paging or permissions.
status: reviewed
---

# Scope and ownership

[HB-161](https://linear.app/hybrid-billing/issue/HB-161/align-u01-source-rows-and-aggregates-with-the-s04-source-field-terms) is a scoped child of U01 HB-85 and follows HB-151 through HB-160 from main `9cb6f23`. The owner continued the sample-backed screens on 2026-11-12; U01 is the last one that carries the older source wording. Root (Claude Code) owns planning, frontend implementation, terminology writing, QA and integration. A separate read-only reviewer reviews this plan before implementation and the actual change afterwards; the ui-designer role compares the final renders. Machine: mac_mini. No owner FIX is created.

# Observed baseline

Facts read on 2026-11-12 from the code at `9cb6f23` and the unchanged U01 browser suite (`apps/console/tools/overview-qa.mjs`), which passed with six variants:

- U01 (`/home`) renders `operations/pages/overview/Screen.tsx` and `FocusChart.tsx` over `/api/focus-overview`, the same reader S04 uses. `AmountSummary.tsx`, `WorkSummary.tsx`, `CostTrend.tsx`, `OrgBreakdown.tsx`, `Lifecycle.tsx`, `ScopeFilters.tsx`, `OrgPicker.tsx` and their model and helpers are demo components that `Screen.tsx` does not import; `Paging.tsx` is used by another screen.
- One local `amount()` helper in `Screen.tsx` prints currency and `decimal()` for every amount: the per-currency summary, the provider subtotals and provider rows, the account mapping amounts and the two costs of each source row. `decimal(null)` returns a dash, so a null aggregate and a null row cost both read `USD —`.
- The source-row table prints a dash for a null service and a null subaccount, the raw provenance class (`official-public`) as the disclosure summary under the header "Source / 출처", and in the disclosure the raw ISO instants under "Charge start / 과금 시작" and "Charge end", and a joined "Batch / ordinal / 수집 배치 / 순번". Its first header reads "Charge month" in English and "شهر الرسوم" in Arabic while the filter reads "Charge-start month"; its account header is the raw column pair "BillingAccountId / SubAccountId".
- The account mapping table heads its columns with the raw column names and prints "Not provided / 제공 없음 / غير مقدم" for a null subaccount inside a `<bdi>`.
- The monthly chart's exact-amount table prints `money(null)` (`USD —`) for a month without a measure and for a month whose values are all missing; its state column reads "No collected value / 수집 값 없음", "Unknown · all values missing / 미확인 · 모든 값 누락", "Partial · n missing" or "All values present". The decorative chart (hidden from assistive technology) marks an absent month with a dash and an all-missing month with "Unknown / 미확인". The provider table marks a partial sum "Partial: n / 부분: n".
- Amounts already use the shared `decimal()`, so a negative reads `USD -2.00`.
- The screen's styles are `overview.css`, `commercial.css` (imported globally) and `focus.css`, breakpoints 1199 and 767. `.unified-overview small`, `.focus-table td`, the global `table { white-space: nowrap }` and the global `bdi { direction: ltr; unicode-bidi: isolate }` meet the shared value components: every bare `<bdi>` on U01 is a left-to-right isolate.
- The provenance line reads "Public / 공개" and "Synthetic / 합성" and omits an unclassified count; the summary's fourth card is titled "Collected scope / 수집 범위" while S04 titles the same counts "Selected scope"; the chart subtitle says gaps "remain unknown"; the monthly state column is headed "Coverage" in English and Arabic; the provider subtotal and the account amounts show no missing count.

# Decision

Frontend-only wording and value-state change in `Screen.tsx`, `FocusChart.tsx` and `focus.css`. `/api/focus-overview`, its contract and validator, selection, paging, permissions, URLs and the chart geometry are unchanged. No field is added to the screen.

| Area | Decision |
| --- | --- |
| Row costs | Through the shared row-amount component: Missing without currency, dash or digit. No Exact zero note in the 25-row table, as on B07 and B08; zero is conveyed by `USD 0.00` and the state attribute |
| Row identity | Null service and null subaccount read Missing through the shared text component, replacing the whole isolate; headers use the shared labels, the paired ones on two lines: "Charge-start month", "Provider" over "Service", "Billing account ID" over "Subaccount ID" |
| Row provenance | Header "Sample provenance / 샘플 출처"; the cell shows the glossary label of the class as plain text |
| Row references | The in-cell disclosure is replaced by a text toggle "Charge period and delivery lineage / 과금 기간·수집본 식별 정보 / فترة الرسوم وبيانات تتبّع التسليم", described by its row's billing account ID, with `aria-expanded` and `aria-controls`, one open at a time, state keyed by snapshot and page; its label does not change so that column widths do not move. It opens a full-width row with one field per line: charge period start and end as UTC instants (a non-UTC instant is shown as collected with the not-converted note), delivery ID, row ordinal in delivery. The row table keeps its existing horizontal scroll region; the reference list stays inside the visible region |
| Aggregates | A separate sum component in its own module for the summary, provider, account and monthly amounts: "No values present / 값 없음 / لا قيم موجودة" without currency when no value is present, at body size; never Missing, never Exact zero; it carries its own attribute and no row state class. The local `amount()` helper is removed |
| Missing counts | One format everywhere through the shared count label, "n missing / 누락 n개 / المفقود: n": summary card, provider rows, and newly the provider subtotal and both account amounts; the monthly state reads "Partial · n missing" |
| Account mapping | Headers "Billing account ID" and "Subaccount ID"; a null subaccount reads Missing outside any isolate |
| Monthly state | Column header "Value state / 값 상태 / حالة القيم". A month absent from the selection: "No rows in selection / 선택 범위에 행 없음 / لا صفوف ضمن النطاق"; a month that has rows but none in the chart currency: "No CUR rows / CUR 행 없음 / لا صفوف CUR"; either phrase spans the amount and state cells. All values missing: amount "No values present", state "All values missing / 모든 값 누락 / كل القيم مفقودة". The word Unknown / 미확인 is dropped; the subtitle says a month without rows is not shown as zero. The decorative chart marks an all-missing month with the full "No values present" term and keeps its mark for a month without rows, which is not a value. The internal `data-value-state` names stay |
| Provenance line | Each class present, by its glossary label with its count: "Public sample", "Public sample subset", "Synthetic test data", "Unclassified" |
| Scope card | "Selected scope / 선택 범위 / النطاق المحدد", as on S04 |
| Style | U01-scoped rules in `focus.css` for the shared classes, the sum, the two-line headers and the reference row; Missing inside a secondary line keeps that line's size at weight 600 in body colour; a long subaccount ID wraps inside its cell |

Unchanged: the unconnected-ledger panels, the scope filters, the collection basis disclosure (its help prose keeps the FOCUS column aliases BilledCost, EffectiveCost, BillingAccountId and SubAccountId, which the glossary preserves in help text; the query time and the error code are not row values), the "No values missing" phrase, and the demo components that the screen does not render.

Stable item IDs: existing items are preserved. Added: `U01:state:value-missing-zero`, registered in the screen and recorded here.

# Allowed files

`apps/console/src/modules/operations/pages/overview/{Screen.tsx,FocusChart.tsx,Sum.tsx,sum.test.tsx,focus-values.ts,focus-values.test.ts,focus.css,focus-chart.test.tsx}`; `apps/console/package.json` (test list); `apps/console/tools/overview-qa.mjs`; `apps/console/README.md`; `docs/features/HB-161/*`; `docs/ui/terminology/product-glossary.md`. The shared components in `sources/collected-focus/` are used, not changed. `Screen.tsx` is 880 lines; if the change pushes any touched file toward the 1,500-line limit the row table moves to its own file in a separate commit without content change.

# Verification

- Console unit suite (the chart render test is updated for the new state wording and gains assertions that a null monthly amount prints no dash and no bare currency), type check, build and build verifier.
- U01 browser suite on the retained sample, extended. Each state or layout case starts from a fresh load at its width; the reference row and the monthly amounts disclosure are opened before assertions; absence checks use text content; a non-zero count of value nodes is asserted.
  - At 1440 on real rows in English: headers; a row's charge-start month, billing account ID, provenance label by its exact text, UTC instants, delivery ID and ordinal equal to the response; no raw provenance class, column name or dash in the row table; the account mapping headers.
  - Routed responses patched from the live response of the same request and valid for the overview validator, in EN, KO and AR at 1440: a row with null billed cost, null service and null subaccount and a `Z` charge start (aggregates of that currency adjusted by difference); an account with a null subaccount; a currency whose summary measure has no present value; a month whose values are all missing. State, count, provenance and sum-phrase nodes have no `bdi` ancestor of any kind; no aggregate region contains a row state attribute or class. The all-missing month is produced by emptying the smallest currency (CNY) on the selected basis in the totals, every provider, every month and the listed accounts, and is read with CNY as the chart currency. The patch applies only to the unfiltered first page on the billed basis; every case waits for the loaded state before asserting.
  - Layout in EN, KO and AR at 1440, 1200, 1199, 1024, 768, 767 and 390 with a row disclosure open: no page overflow, no child of a table cell outside its cell, no row instant broken inside its date. The routed response at 390 in KO and AR.
  - A failed read leaves no amount, table or source value node. Existing selection, paging, history, late-response, failure and foreign-scope cases and the six variants still run.
- S04 browser suite as regression; isolated `tests/focus_overview` backend tests unchanged; retained file hashes against the HB-150 manifest.
- Independent review of the actual change; ui-designer render comparison; technical-writing review of the changed strings. AI review is not native-language, practitioner or owner acceptance.

# Deferred

Operational ledger connections; closing targets (HB-156); removal or alignment of the demo components the screen does not render; one Korean label for the row-detail button and the evidence toggle across B04, B05, B07 and B08; HB-152.

# Design review resolution

The independent design review of 2026-11-12 (read-only, nothing executed) returned FAIL without a blocker: 2 HIGH, 8 MEDIUM, 6 LOW. Resolution:

| Finding | Resolution |
| --- | --- |
| F1 absent month wording | Split into "No rows in selection" and "No CUR rows"; `chartSeries` reports whether the month is in the selection; both cases in the unit tests |
| F2 routed cases | Invariants adopted: per-measure counts and null rule, provider sums equal to totals per currency and basis, paging and selection echo untouched. One-row delta applied to totals, the row's provider, month and listed account; the all-missing case empties one currency everywhere |
| F3 bare isolates | Wrappers replaced; the suite asserts no `bdi` ancestor of any kind |
| F4 aggregate selector | The sum has its own attribute and no state class; aggregate regions are asserted free of row states, dashes and superseded words |
| F5 scroll region and widths | The row table keeps its scroll region; paired headers on two lines; the toggle does not change column widths (asserted); cell children inside their cells and unbroken instants asserted; horizontal scroll per width is recorded |
| F6 disclosure summary | Provenance as plain text; a labeled toggle and a full-width reference row |
| F7 missed sites | Provenance line, chart subtitle, state header and scope card changed; help-prose column aliases, query time and error code listed as unchanged |
| F8 unit tests | The sum has its own module and render test; chart assertions are scoped to the table body |
| F9 partial sums | Missing counts added to the provider subtotal and the account amounts. A zero-row currency is not emitted by the sample and is not covered |
| F10 terminology | One count format; "Value state"; "Selected scope"; the chart uses the full term; new phrases recorded in the glossary |
| F11 allowed files | Extended |
| L1 to L6 | Two-span amount kept; zero conveyed by the amount only; the monthly table keeps its class; routed assertions for the count label; the demo list corrected. The new item ID is recorded in this plan only |
