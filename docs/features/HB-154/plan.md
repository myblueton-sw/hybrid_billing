---
type: ImplementationPlan
title: B08 source input evidence in the S04 source-field terms
description: Defines how B08 presents collected source row values with the field vocabulary and value states fixed for S04 and applied to B01, without changing its comparison arithmetic.
status: draft
---

# Scope and ownership

[HB-154](https://linear.app/hybrid-billing/issue/HB-154/align-b08-source-input-evidence-with-the-s04-source-field-terms-and) is a scoped child of B08 HB-109 and follows HB-151 and HB-153 from main `e7abbea`. The owner continued the screen-by-screen refinement on 2026-11-12. Root (Claude Code) owns planning, frontend implementation, terminology writing, QA and integration; the ui-designer role compares the final renders; a separate read-only reviewer checks money semantics, clearing behavior, the actual change and copy. Machine: mac_mini. No owner FIX is created.

B03 precedes B08 in the menu but is not changed here. Its Korean policy editor carries an owner FIX ([acceptance record](../../ui/acceptance/B03.md)), it reads scope aggregates and not rows, and its only pending difference from the current terms is wording (공급자, "rows without an amount") inside the accepted scope. Changing accepted copy needs the owner's explicit change request; the question is reported to the owner.

# Observed baseline

Facts read on 2026-11-12 from the canonical preview and the unchanged B08 browser suite, which passed on main `e7abbea` with 36 captures:

- The input table, mobile list, row detail and the trace view show row amounts through one component that also formats totals and differences. A null row amount reads "No observed amount"; a negative amount is written with a leading minus before the currency, where S04 and B01 write `USD -2.6137`; an exact zero has no text state.
- The row detail joins values in pairs ("Service / provider", "Billing account / sub-account", "Charge interval", "Billing interval", "Source batch / ordinal") with raw ISO strings, prints the sample annotation as the raw enum under "Provenance class", and does not show the derived charge-start month or the billing currency as fields. Null text reads "No source value", a null quantity "No observed quantity", a null unit "Unit unavailable".
- Korean copy says 청구 계정 and 공급자 where the glossary fixes 결제 계정 and 공급사; quantity labels differ from S04 (소비 수량, 과금 수량).
- The row detail does not show list or contracted cost or unit prices; S04 does.

# Decision

Frontend-only presentation change with no layout change: the table with a side detail at desktop width and inline detail on mobile stay as designed in HB-109. `/api/focus-overview`, B08's validator (`api.ts`), the baseline lifecycle and comparison arithmetic (`model.ts`), storage (none) and URLs are unchanged.

Alternatives considered: embedding the S04 row-detail component (rejected: it is another screen's review unit and carries S04-specific statements about scope versions and time zones); leaving the sign format of totals and differences as is while changing rows (rejected: one screen would then write negative amounts two ways).

# Presentation

Amounts are split by meaning into two components:

| Use | Null | Zero | Other |
| --- | --- | --- | --- |
| Row amount (table, mobile list, detail, trace view) | "Missing" alone, without currency, dash or digit | exact formatted zero; the "Exact zero" note is added in the detail and the trace view, not in the 25-row table or the mobile list | shared `money` format, sign kept |
| Total and difference (totals, comparison) | "No values present / 값 없음"; differences keep their existing "Unavailable: complete matching observations required" | formatted zero, never labeled Exact zero because a total can be a net and a zero difference is a result | shared `money` format; a negative total or difference now reads `USD -1.00` instead of a leading minus before the currency |

Present and missing counts beside totals are unchanged. No arithmetic moves: the difference is still computed by `compare` in `model.ts`.

The row detail lists one field per line with the shared labels, grouped by origin: from source columns (provider, service, billing account ID, subaccount ID, billing currency, charge period start and end, billing period start and end, source billed cost, source effective cost), derived by Hybrid Billing (charge-start month, row ID broken only at its midpoint), delivery lineage (delivery ID, row ordinal in delivery) and sample annotation (sample provenance through the glossary label). Instants use the shared UTC presentation followed by "UTC" on the same line; an instant the shared parser does not read as `+00:00` (B08's validator also admits `Z` and other offsets) is shown as collected with "Shown as collected; not converted to UTC". Null text reads Missing. The quantity disclosure uses the S04 labels, Missing, Unit missing and the Exact zero note, and states that list and contracted costs and unit prices are shown in S04. B08's basis labels "Source billed cost" and "Source effective cost" are kept because they mark the amounts as source costs on a billing screen.

The table's account column is headed "Billing account ID" with "Charge period start (UTC)" beneath and shows the start through the shared presentation; a null service reads Missing, with its field label on the mobile card. The trace view shows the same row with Missing for a null service and "Row ID" for the row identifier. The account list uses the shared labels and Missing for a null subaccount. Korean 공급자 becomes 공급사 in the toolbar and the coverage note.

Stable item IDs: the ten existing `B08:` items are preserved. Added: `B08:data:source-field-origin` (grouped detail) and `B08:state:value-missing-zero`.

# Structure

`SourceInputs.tsx` (494 lines) is split by responsibility: `Amounts.tsx` (total, row amount, amount with counts) and `SourceDetail.tsx` (row detail). Shared labels come from `sources/collected-focus/labels.tsx`. Allowed files: `apps/console/src/modules/calculation/pages/billing-runs/*`, `apps/console/tools/billing-runs-qa.mjs`, `apps/console/README.md` (B08 paragraph), `docs/features/HB-154/*`, `docs/ui/terminology/product-glossary.md` (B08 note).

# Review order

The value states, labels and period presentation are the decisions already reviewed for HB-151 and HB-153; this ticket applies them to a third screen. New here are the sign format of totals and differences and several strings: the detail introduction, the not-converted note, the reference to the S04 screen, the split table header and the mobile heading for a missing service. Root therefore implemented before review and asked one independent reviewer to judge this plan and the actual change together. That is a single combined review, not a separate design review followed by a change review, and it is recorded as such.

# Verification

- Console unit suite, type check, build and build verifier. No new pure logic is added; the value-state and label helpers are covered by the existing S04 and B01 unit tests.
- B08 browser suite on the retained sample, extended: an Oracle row detail end to end (four origin groups, `USD 0.24`, `USD 0.00` with Exact zero, UTC period lines, charge-start month, ordinal, Public sample, full row ID, no superseded labels); Korean and Arabic zero labels; an AWS row with `USD -2.6137` and `USD -3.00`; an Alibaba Cloud CNY row; a routed response with all billed amounts missing plus a row with missing service, subaccount, consumed quantity and unit, an exact zero pricing quantity and unclassified provenance, in three languages. The existing baseline, refresh, discard, scope, 403, 503, 409, late-response, paging, search and focus cases and the 36 view, language and width variants still run.
- S04 and B01 browser suites as regression; isolated `tests/focus_overview`; retained file hashes against the HB-150 manifest.
- Independent combined review; ui-designer render comparison; technical-writing review of the changed strings. AI review is not native-language, practitioner or owner acceptance.

# Review resolution

The combined independent review returned PASS with five medium and seven low findings; the ui-designer comparison returned conditional PASS with one blocking and five should-fix items. Applied: Arabic amounts share the edge of their labels (the amount wrapper, not the left-to-right isolate, is the block); the Arabic S04 screen name matches the menu; the two item IDs are registered in the screen; a routed lower total asserts a difference of `USD -1.00`; a missing service carries its field label on the mobile card and in the trace view, which also names the row ID; the unit state sits on its own line; `Z` instants are described as shown as collected; UTC moves onto the value line; group headings use body color with section rules; delivery ID and the trace row ID use the code style; the table columns are rebalanced; dead style rules are removed. These corrections were verified by root's rerun and renders, not by a further review.

# Deferred

Billing-run, policy, claim, approval and ERP producers; baseline persistence; list and contracted values in the B08 detail; read times in the comparison view still shown as raw ISO strings; B03 wording pending the owner's change request; U01, B04, B05, B06 and B07 terminology in their own tickets; HB-152.
