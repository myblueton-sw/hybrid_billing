---
type: ImplementationPlan
title: B05 source evidence in the S04 source-field terms
description: Defines how B05 presents collected source row values with the field vocabulary, value states and evidence layout already fixed for S04 and B04, without changing its conversion and rounding arithmetic.
status: draft
---

# Scope and ownership

[HB-158](https://linear.app/hybrid-billing/issue/HB-158/align-b05-source-evidence-with-the-s04-source-field-terms-and-value) is a scoped child of B05 HB-105 and follows HB-151, HB-153, HB-154 and HB-157 from main `b834de8`. The owner continued the sample-backed screens on 2026-11-12; B05 is next. Root (Claude Code) owns planning, frontend implementation, terminology writing, QA and integration. A separate read-only reviewer reviews this plan before implementation and the actual change afterwards; the ui-designer role compares the final renders. Machine: mac_mini. No owner FIX is created.

# Observed baseline

Facts read on 2026-11-12 from the code at `b834de8` and the unchanged B05 browser suite, which passed:

- The picker's source cost and the evidence disclosure print "Source value missing / 원천 값 없음" for a null amount; the full-columns table for mobile prints a dash for the same null; the source summary prints a dash both when no row is selected and when the selected row's amount is null (`Screen.tsx:553-560`). An exact zero has no text state.
- The evidence disclosure sits in the last table column and lists billed cost, effective cost, row ID, the raw annotation enum under "Evidence classification", the billing account under "Account reference" and "Source charge interval" as two raw ISO strings. The subaccount and the row ordinal are not shown anywhere in the picker.
- The full-columns table prints the service cell empty when the service is null.
- The summary shows the service of the selected row or "No row selected", so a selected row with a null service reads as no selection (the defect fixed on B04 in HB-157).
- The account scope lists accounts as an unlabeled joined string and omits a null subaccount.
- The rule editor's first step shows the selected row's source cost in B05's own money format and prints "Select a source row" when a row is selected but its amount is null (`RuleEditor.tsx:130-156`). Below 641px the summary strip is hidden, so this line is the only place the selected amount appears on mobile.
- A null service reads "Service unavailable / 서비스명 없음" in the picker; the validation message for a null amount reads "source cost is unavailable / 원천 비용이 없습니다" (`copy.ts:108-112`).
- Korean copy says 공급자 (provider label, and "공급자 환산 이력" in three notes) and 금액 없음; "interval" appears in labels and the source note.
- Source amounts use B05's own money format with a leading typographic minus, the same function as its computed results (`values.ts:5-8`).

# Decision

Frontend-only presentation change. `/api/focus-overview`, the validator, the conversion and rounding arithmetic, validation and the browser draft format (`model.ts`, `values.ts`, `store.ts`) are unchanged. B05's converted, rounded and compared amounts are its computed review values and keep their reviewed format; source row values move to the shared presentation. The resulting difference in sign format between source values and results is the residual already recorded for B04.

Two structural choices follow decisions already reviewed for B04 in HB-157:

1. **Shared source-value components.** `pricing/pages/sales-pricing/SourceValues.tsx` is deleted and its components move to `sources/collected-focus/SourceValues.tsx` with screen-neutral class and attribute names and the shared copy helper, so that B04 and B05 use one implementation of Missing, Unit missing, Exact zero, the UTC instant and the labeled period (B01 and B08 keep their own). The exported names and the "Shown as collected; not converted to UTC" strings stay identical. The folder has no stylesheet, so each screen styles the neutral classes under its own root; B04's rules are renamed without touching its `data-pricing-state` attribute or its `pricing-period` select id. B04's rendered behavior must not change: its suite is rerun and its rates view is compared with the unchanged build. A unit test renders the shared components for each value state.
2. **Evidence in a full-width row.** A picker row's evidence opens under the record through a text toggle with `aria-expanded`, one row at a time. The open state is local to the picker and keyed to snapshot and page offset; it is not part of the screen state, the review payload or the URL, and it closes on row selection, paging, basis or account change and any failed read, because the picker unmounts while the query reloads. The toggle sits in the availability cell, so the table has four columns: select, source row, source cost, availability. Fixed column widths (124 / rest / 192 / 208) replace the per-cell minimum. At 800px and below the picker is cards, a breakpoint the ui-designer set after the first comparison because the fixed columns squeezed the source-row column between 641px and 800px; the open record and its evidence row form one card whose cells reset the shared fixed height. Selection is shown on the pressed select button and no longer by bolding the service name, so that a bold Missing service cannot be read as a selected row. Escape still closes only disclosures, the residual accepted for B04.
3. **The full-columns table for mobile is removed.** Its columns become a strict subset of the card plus the evidence row, its title "All source columns" would be false, and it is the site that prints a dash for a null amount. No requirement for it was found in the B05 design record. This removal is a design change put to the ui-designer and reported to the owner.

Alternative considered: keep B05's disclosure in the last column and change only the words. Rejected because the same layout was judged unreadable on B04 once the evidence carries twelve fields.

# Presentation

| Site | Change |
| --- | --- |
| Picker, source-row column | Header "Source row"; service (Missing when null), provider, billing account ID, subaccount ID (Missing when null) |
| Picker, source cost | Amount on the selected basis through the shared row-amount component with its basis label; Missing without currency, dash or digit; no zero note in this column |
| Picker, availability | "Amount missing / 금액 누락" instead of 금액 없음; otherwise unchanged |
| Evidence row | Billed cost, effective cost (both with the Exact zero note), charge period start and end in UTC, billing account ID, row ordinal in delivery, sample provenance by its glossary label, subaccount ID and the full row ID, the last two on full-width lines |
| Rule editor, step 1 | Source cost through the shared amount component with its basis label on the line below in every state; Missing when a row is selected and its amount is null; "Select a source row" only when no row is selected. `sourceAmount` in `model.ts` stays the single reader of the amount and is not changed |
| Source summary | Service through the shared text component, so a selected row with a null service reads Missing; source cost reads Missing for a null amount and keeps the dash only when no row is selected; charge period as labeled start and end lines |
| Account scope | Each account as labeled billing account ID, subaccount ID (Missing when null) and provider; the summary line shows the selected account's billing account ID and subaccount ID when that account is on the current account page |
| Wording | 공급사, including 공급사 환산 이력; "charge period / 과금 기간" in labels and the source note; the validation message for a null amount reads "source cost is missing / 원천 비용이 누락되었습니다" |

Source amounts are rendered only through the shared amount component, never through a money formatter directly, because the shared formatter prints a dash for null. A dash remains only where no row is selected (source cost, currency and period placeholders) and in rule and result fields the user has not filled, which read "Not entered" or a dash by B05's own design; that user-input state is never used for a source value. The editor's source cost in the shared format sits in the same view as the rounded result in B05's format; that adjacency is part of the recorded sign-format residual.

Stable item IDs: existing `B05:` items are preserved. Added: `B05:data:source-row-identity` and `B05:state:value-missing-zero`, registered in the screen and recorded here; the HB-105 design record is left as written.

# Allowed files

`apps/console/src/modules/pricing/pages/fx-rounding/*` except `model.ts`, `values.ts`, `store.ts` and their tests; `apps/console/src/modules/sources/collected-focus/SourceValues.tsx` (moved); in `apps/console/src/modules/pricing/pages/sales-pricing/` only the import lines and `screen.css` (the renamed classes, and one rule so that the not-converted note under a raw instant in the period sits on its own line, a defect found while rendering B05 that B04 shared); `apps/console/tools/fx-rounding-qa.mjs`; selector renames in `apps/console/tools/sales-pricing-qa.mjs`; `apps/console/package.json` (test list); `apps/console/README.md`; `docs/features/HB-158/*`; `docs/ui/terminology/product-glossary.md`.

# Verification

- Unit: a render test of the shared components for missing, zero, negative, unit present, unit blank and non-UTC instant states. Console unit suite, type check, build and build verifier.
- B05 browser suite on the retained sample, extended. Assertions target value nodes by attribute and state their width, because the summary strip is hidden at 640px and below and text searches would also match separators and placeholders.
  - At 1440: account and subaccount in the picker; evidence fields including ordinal, provenance label, full row ID and UTC period; exactly one evidence row open; closure of the evidence row after selecting a row, changing the basis, paging on a provider with more than 25 rows and a 403, with no source-value node left in the error state; the rule editor's source cost for a value, an exact zero, a negative amount and Missing; an Oracle row with `USD 0.00` and Exact zero in three languages; an AWS negative amount; an Alibaba Cloud CNY amount.
  - Routed response, valid for the client validator, in three languages at 1440: a null amount on the selected basis and on the other basis, a null service, a null subaccount, an unrecognized provenance class and a `Z` instant, asserted in the picker, the evidence row and, after selecting the row, the summary and the rule editor.
  - Layout with evidence open at 1440, 1280, 1001, 1000, 800, 641, 640 and 390 in EN, KO and AR: no page overflow; no child of a picker cell extending outside its cell; the toggle no wider than its column. At 390 the degraded response in KO and AR with evidence open and the degraded row selected.
  - Unchanged arithmetic: the existing converted and rounded results and the check that no source value is stored in the browser stay asserted. Existing rule, preview, baseline, save, restore, guard, denial and late-response cases still run.
- B04: its browser suite, and a comparison of its rates view at 1440 and 390 between the unchanged build and this build. S04 browser suite as regression; isolated `tests/focus_overview`; retained file hashes against the HB-150 manifest.
- Independent review of the actual change; ui-designer render comparison including mobile open-evidence and degraded Korean and Arabic renders; technical-writing review of the changed strings. AI review is not native-language, practitioner or owner acceptance.

# Design review resolution

The independent design review of the first draft returned FAIL without blockers. Resolved in this revision: the rule editor's source-cost line added as a source-value site (1); assertions scoped to value nodes with stated widths (2); the rename hazards, shared copy helper, per-screen styles and B04 comparison stated for the shared move (3); B05's own layout rules, toggle column, evidence state and floor stated (4); the full-columns table removed with its reason (5); the remaining wording sites, the validation message and the user-input state decided (6); new item IDs recorded here (7); widths, cell checks, degraded mobile states, closure, arithmetic guard and a unit test added to the verification (8).

# Deferred

Rate feeds, contract precedence, server history and approval producers; one sign format for computed results and source values on B04 and B05; U01, B06 and B07 terminology in their own tickets; HB-152.
