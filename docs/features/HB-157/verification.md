---
type: VerificationRecord
title: B04 source evidence verification
description: Records what was executed and reviewed for the B04 source-field terms, value states and evidence layout on the retained sample, with results, limits and accepted residuals.
status: draft
---

# Delivered behavior

[HB-157](https://linear.app/hybrid-billing/issue/HB-157/align-b04-source-evidence-with-the-s04-source-field-terms-and-value) refines B04 from main `a33c444` using the retained public and synthetic sample verified in HB-149. Root (Claude Code) owns planning, implementation, terminology writing, QA and integration; a separate read-only reviewer reviewed the plan and the change; the ui-designer role compared renders and directed one layout change. Machine: mac_mini. See the [plan](plan.md).

B04's source row picker, evidence, source summary and source cost evidence present collected row values with the S04 source-field terms: Missing, Unit missing and Exact zero, the charge period in UTC with start and end labels, billing account ID and subaccount ID in the picker, and row ordinal, sample provenance and the full row ID in the evidence. A picker row's evidence opens in a full-width row under the record through a toggle button. Source amounts use the shared format. `/api/focus-overview`, the pricing arithmetic, validation and the browser draft format (`model.ts`, `store.ts`, `values.ts`) are unchanged, and B04's computed candidate, adjustment and comparison amounts keep their format.

# Executed checks

| Check | Actual outcome |
| --- | --- |
| Console unit suite (`npm test --prefix apps/console`) | PASS 296, unchanged count |
| TypeScript check, Vite build, `tools/verify-build.mjs` | PASS; 438 artifact hashes and licenses; existing large-chunk advisory remains |
| B04 browser suite `apps/console/tools/sales-pricing-qa.mjs` on the retained sample | PASS; 0 page errors; see below. The unchanged suite also passed on main `a33c444` before any edit |
| S04 browser suite | PASS as regression |
| Isolated backend regression (`tests/focus_overview`) | PASS 36; unchanged backend |
| Retained source, control and prior-result files | PASS 93 file hashes equal to the HB-150 preservation manifest after all reads |

Browser cases on real rows, English at 1440 unless stated: the VMware rows show the billing account and the subaccounts in the picker; the opened evidence shows ordinal `1000`, Synthetic test data, the full row ID, both period instants with UTC and `USD 0.75 / Hours`, and the table region does not scroll. The first Oracle row shows `USD 0.24`, `USD 0.00` with Exact zero and a Missing contracted unit price, with the Korean and Arabic labels asserted; after selecting it the summary shows the start and end lines, and the comparison view's cost evidence shows the two field labels with `USD 0.24` and `USD 0.00` with Exact zero. The AWS row at offset 450 shows `USD -2.6137` and an Alibaba Cloud row `CNY 8.80`.

A routed response derived from the real VMware response covers what the sample lacks, in EN, KO and AR: a missing billed cost shown as Missing without currency, dash or digit; a missing service; a missing pricing unit; a row with pricing quantity and unit both missing; a whitespace-only unit; a missing list unit price and subaccount; an exact zero consumed quantity without a unit; an unrecognized provenance class; a `Z` instant shown as collected; only one evidence row open at a time; and, after selecting the degraded row, Missing for its service in the summary. The not-converted note is asserted in English only.

Layout: Oracle rows with an evidence row open at 1280, 1024 and 768 assert no page overflow, no picker cell content wider or taller than its cell and a source-row column of at least 184px. The 30 view, language and width variants assert no page overflow and, in every view, no table cell content taller than its cell; in the rates view the same holds with an evidence row open. A Korean 390px pass with the degraded response asserts the same. The existing calculation, adjustment, baseline, save, restore, guard, denial and late-response cases still pass.

Host: macOS arm64, Node 26.7.0, Python 3.11.7 from the prepared environment, Playwright 1.56.1 with Chromium headless shell build 1243. Other operating systems, browsers and assistive technology are unverified. Captures stay local under `output/validation/HB-157` and are excluded from Git.

# Findings corrected during verification

- Root's first render showed the evidence column clipped inside a horizontally scrolled table; cells were allowed to wrap.
- Both reviews found overlapping text in the 390px picker cards: a shared rule gives table cells a fixed height, which becomes a hard limit once the cells are laid out as blocks on mobile, and the two added account lines overflowed it. The cell height is reset on this screen for mobile, and the suite now compares content height with cell height. The baseline already overflowed slightly in the same cells.
- Suite errors that were root's and not product results: a dialog handler left registered, a toggle step run in views without the picker, a reused URL that left the evidence closed at two widths, and a width floor set 4px above the 1440px region.

# Independent review

Reviewer in each round: the repository's read-only `independent-reviewer` agent role (AI), 2026-11-12. It read files and renders and executed nothing.

| Review | Result |
| --- | --- |
| Combined review of the plan and the actual change, after implementation | FAIL: one high finding (overlapping text in the 390px picker cards), three medium (a selected row with a missing service read as no selection; vacuous or absent assertions; no unit test) and smaller wording, style and document items |
| Narrow re-review of the fixes | FAIL: the earlier fixes and the evidence-row restructuring confirmed correct; one high finding inferred from the styles and not rendered (the fixed table layout squeezing the source-row column between the tested widths), the verification record missing, and remaining assertion gaps |
| Narrow check of the final corrections | PASS: no high or blocking finding; the table floor, the mobile cell-height reset in the other views, the arithmetic of the column widths and the added assertions confirmed by reading and by the 1280, 1024 and 768 renders. Two medium document items (the scroll range in the residuals understated; this row deferring its own result) and two low items (quantities can also break mid-digit; the mid-width captures show the region already scrolled) were then corrected in this record by root |

The plan was not reviewed before implementation; the plan states why.

# Design and wording review

The ui-designer role compared baseline and changed renders: FAIL on the mobile overlap, with should-fix items for the evidence placement, column widths, Arabic alignment of the period values and the period lines. All were applied; the evidence row, fixed widths and period alignment follow its specification. The corrected renders were checked by root and by the suite, not by a second designer pass. The designer saw no open evidence on mobile and no Arabic or Korean degraded state.

Open for a native or practitioner reviewer: Arabic labels that differ between B04 and the shared field labels for consumed quantity and unit prices, mixed diacritic spellings, and the Korean label 원천 근거 보기, which B08 and B07 pair with a different English label. AI design and wording review is not owner FIX, native-language acceptance or practitioner approval.

# Accepted residuals

- B04's computed amounts keep a leading minus and an explicit plus on differences while source amounts read `USD -2.61`; a formula line with a negative pricing quantity can show both.
- The picker table has a 760px floor and scrolls inside its region whenever the region is narrower: from 601px to about 807px and from 901px to about 1423px viewport width, so at 1280px either the Select column or part of the evidence column needs a horizontal scroll. Root measured the region on the unchanged build and on this build at 1440, 1280, 1024 and 768: before, the table was 815px wide at all four widths and scrolled even at 1440; now it is 776px at 1440 without scrolling and 760px at the other three. Whether scrolling at 1280 is acceptable is a design question for the owner; the designer did not see the corrected renders.
- A very long amount or quantity can break mid-digit inside its 132px column, so a trailing digit can sit beside the unit on the next line.
- The Escape key closes the summary disclosure but not an open evidence row.
- A `Z` instant is UTC but is shown as collected with the not-converted note, because the shared parser reads only `+00:00`.
- U01, B05, B06 and B07 keep older wording until their own tickets; HB-152 is unchanged.

# Acceptance

Parent HB-104 remains awaiting explicit owner screen acceptance. This change alters the B04 candidate revision; no acceptance exists for B04 and none is created. No ingestion, reset, reimport, schema or permission change, live account, customer data, deployment, payment or message occurred. Merge, read-back, ticket completion and machine release are recorded in the ticket only after they occur.
