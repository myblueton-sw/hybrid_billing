---
type: ImplementationPlan
title: B04 source evidence in the S04 source-field terms
description: Defines how B04 presents collected source row values with the field vocabulary and value states fixed for S04, without changing its pricing arithmetic.
status: draft
---

# Scope and ownership

[HB-157](https://linear.app/hybrid-billing/issue/HB-157/align-b04-source-evidence-with-the-s04-source-field-terms-and-value) is a scoped child of B04 HB-104 and follows HB-151, HB-153 and HB-154 from main `a33c444`. On 2026-11-12 the owner chose to finish the screens that read the retained sample first; B04 is the next of them. Root (Claude Code) owns planning, frontend implementation, terminology writing, QA and integration; the ui-designer role compares the final renders; a separate read-only reviewer checks money semantics, the actual change and copy. Machine: mac_mini. No owner FIX is created.

# Observed baseline

Facts read on 2026-11-12 from the canonical preview and the unchanged B04 browser suite, which passed on main `a33c444`:

- A null source value reads "Source value missing / 원천 값 없음" everywhere, including a missing unit; an exact zero has no text state.
- The evidence disclosure and the source summary label the period "Source charge interval / 원천 과금 구간" and print the two raw ISO strings joined by a dash.
- The source cost evidence in the comparison view prints the raw column names `BilledCost:` and `EffectiveCost:` with a dash for a null amount.
- Source amounts are written with a leading typographic minus before the currency when negative, where S04, B01 and B08 write `USD -2.6137`.
- The picker identifies a row by service and provider; the three VMware rows differ only by quantity and cost columns. The evidence disclosure shows the row ID but no account, ordinal or sample provenance.
- Korean copy says 공급자 where the glossary fixes 공급사.

# Decision

Frontend-only presentation change. The picker table, the source summary strip and the comparison view stay as designed in HB-104, with one layout change made at the ui-designer's direction after the first render comparison: a picker row's evidence opens in a full-width row under the record through a toggle button, instead of a disclosure inside the narrow last column. `/api/focus-overview`, the validator, the pricing arithmetic, validation and the browser draft format (`model.ts`, `store.ts`, `values.ts`) are unchanged.

Source values and B04's own results are kept apart. Source row values move to the shared presentation. Candidate amounts, adjustment differences and comparison deltas are B04's computed review values; they keep their existing format, including the leading minus and the explicit plus sign on differences, because the B04 result format was reviewed with the pricing arithmetic in HB-104 and is asserted by its suite. The consequences are recorded as a residual for the owner and not changed here: a negative source amount reads `USD -2.61` beside a negative candidate with a leading minus, and the formula lines, which already combined the shared quantity format with the B04 amount format before this change, can show both signs in one line for a negative pricing quantity.

# Presentation

New `SourceValues.tsx` holds the source-value components; `SourcePicker.tsx`, `Screen.tsx` and `Results.tsx` use them.

| Value | Presentation |
| --- | --- |
| Null text, amount or quantity | "Missing" without currency, dash or digit. A missing quantity whose unit is also missing adds "Unit missing"; a missing unit price shows Missing alone because its unit is the row's pricing unit, shown in the picker row |
| Unit missing on a present quantity or unit price | The number, then "Unit missing" on its own line; the eligibility column keeps its specific "Pricing quantity missing / Pricing unit missing" text |
| Exact zero | The formatted zero with the "Exact zero" note in the evidence disclosure, the source summary and the cost evidence; not in the picker's cost column |
| Other amount | Shared `money` format with currency; unit prices followed by the pricing unit |
| Instant | Shared UTC presentation followed by UTC; an instant the shared parser does not read is shown as collected with a not-converted note |

The evidence disclosure lists source list unit price, source contracted unit price, billed cost, effective cost, consumed quantity (unused in the calculation), charge period start and end, billing account ID, subaccount ID, row ordinal in delivery, sample provenance and the row ID. Billed cost is added so that both cost bases sit side by side as evidence. The picker's first data column is headed "Source row" and adds the billing account ID and subaccount ID under service and provider. The summary strip shows Missing for a selected row whose service is missing; the dash there means only that no row is selected. The source summary shows the charge period as labeled start and end lines in UTC and the pricing quantity with the same states. The comparison view's cost evidence uses the field labels. "Interval" becomes "period" in labels and the source note; 공급자 becomes 공급사.

The picker table uses fixed column widths with the source-row column taking the rest and a 760px floor, so at 1440px it does not scroll and at narrower two-column widths the existing scroll region scrolls instead of squeezing the source-row column. Cells may wrap, and the open evidence row lays its fields out in a grid with the subaccount and row ID last on full-width lines. On mobile the cards no longer inherit a fixed cell height, which had made the added lines overlap the next cell. The Korean validation messages for a missing pricing quantity or unit use 누락.

Stable item IDs: existing `B04:` items are preserved. Added: `B04:data:source-row-identity` and `B04:state:value-missing-zero`.

# Review order

As in HB-154, the value states, labels and period presentation are decisions already reviewed for HB-151 and HB-153. Root implemented before review and asked one independent reviewer to judge this plan and the actual change together. That is a single combined review after implementation, recorded as such.

# Verification

- Console unit suite, type check, build and build verifier. No unit test is added: the new components branch on value, unit and zero states, and the browser suite asserts those branches on real and routed rows. Not asserted: a zero amount without the note in the picker cost column, the Korean and Arabic start and end labels and not-converted note, the Korean validation messages, and closure of the evidence row on paging.
- B04 browser suite on the retained sample, extended: VMware rows show account and subaccount in the picker and ordinal, provenance, full row ID, UTC period and a unit price with its unit in the disclosure, with no horizontal scroll; an Oracle row shows `USD 0.24`, `USD 0.00` with Exact zero and a Missing contracted unit price, with Korean and Arabic labels; an AWS row shows `USD -2.6137`; an Alibaba Cloud row shows `CNY 8.80`; a routed response covers missing billed cost, service, subaccount, pricing unit, list unit price and pricing quantity, an exact zero consumed quantity without a unit and an unrecognized provenance class in three languages. The existing calculation, adjustment, save, restore, guard, denial and late-response cases and the 30 view, language and width variants still run.
- S04 browser suite as regression; isolated `tests/focus_overview`; retained file hashes against the HB-150 manifest.
- Independent combined review; ui-designer render comparison. AI review is not native-language, practitioner or owner acceptance.

# Review resolution

The combined independent review returned FAIL with one high finding, overlapping text in the 390px picker cards, and three medium findings: a selected row with a missing service read as "No row selected", vacuous or absent assertions, and no unit test. The ui-designer comparison returned FAIL on the same overlap and asked for the evidence to leave the last column, stable column widths, and the Arabic period values on the label's edge. Applied: the fixed mobile cell height is reset; the summary uses the missing state for the service; the evidence opens in a full-width row with a text toggle; fixed column widths and wrapping identifiers; start and end labels on the period; a full-width period cell on mobile; Korean validation wording; one behavior stated for missing values; assertions for the service node, the summary strip, the cost evidence, the period end, a whitespace-only unit, a `Z` instant, and no cell content taller than its cell at 390px with an evidence row open. A narrow re-review of these corrections then returned FAIL on an inferred squeeze of the source-row column between the tested widths under the fixed table layout, the missing verification record and remaining assertion gaps. Applied: the table floor, wrapping service names, a wider cost column, the full-width evidence fields moved last, assertions at 1280, 1024 and 768 with an evidence row open (no cell content wider or taller than its cell, source-row column at least 184px), a row with quantity and unit both missing, the zero note on a quantity, cost-evidence labels, single open evidence row, a degraded mobile pass, and the cell-height check in all five views. The verification record states each review and its result.

# Deferred

Contract, SKU, price-list, approval and activation producers; server persistence; one sign format for B04 results and source values; U01, B05, B06 and B07 terminology in their own tickets; HB-152.
