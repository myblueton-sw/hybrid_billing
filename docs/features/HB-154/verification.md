---
type: VerificationRecord
title: B08 source input evidence verification
description: Records what was executed and reviewed for the B08 source-field terms and value states on the retained sample, with results, limits and accepted residuals.
status: draft
---

# Delivered behavior

[HB-154](https://linear.app/hybrid-billing/issue/HB-154/align-b08-source-input-evidence-with-the-s04-source-field-terms-and) refines B08 from main `e7abbea` using the retained public and synthetic sample verified in HB-149. Root (Claude Code) owns planning, implementation, terminology writing, QA and integration; a separate read-only reviewer reviewed the plan and the actual change together after implementation; the ui-designer role compared renders. Machine: mac_mini. See the [plan](plan.md).

B08's source input table, mobile list, row detail, trace view and account list present collected row values with the S04 source-field terms: Missing, Unit missing and Exact zero, charge and billing period in UTC, the derived charge-start month, delivery ID and row ordinal, sample provenance and the full row ID, grouped by origin in the detail. Totals read "No values present" when no value is present. Negative amounts, totals and differences read `USD -1.00`. `/api/focus-overview`, B08's validator, the baseline lifecycle, the comparison arithmetic, storage and URLs are unchanged. No layout was redesigned.

# Executed checks

| Check | Actual outcome |
| --- | --- |
| Console unit suite (`npm test --prefix apps/console`) | PASS 296, unchanged count; no new pure logic was added |
| TypeScript check, Vite build, `tools/verify-build.mjs` | PASS; 438 artifact hashes and licenses; existing large-chunk advisory remains |
| B08 browser suite `apps/console/tools/billing-runs-qa.mjs` on the retained sample | PASS; 36 view, language and width captures; 0 page errors; no non-GET request; see below. The unchanged suite also passed on main `e7abbea` before any edit |
| S04 and B01 browser suites | PASS as regression |
| Isolated backend regression (`tests/focus_overview`) | PASS 36; unchanged backend |
| Retained source, control and prior-result files | PASS 93 file hashes equal to the HB-150 preservation manifest after all reads |

Browser cases on real rows, desktop detail, English unless stated: the first Oracle row shows the four origin groups in order, `USD 0.24`, `USD 0.00` with Exact zero, `2024-09-30 22:00:00 UTC`, billing period end, charge-start month `2024-09`, ordinal `1`, Public sample, USD and the full row ID, with no "interval", "batch", "Provenance class" or "sub-account" label left in the detail and no zero note in the table; the Korean and Arabic zero labels are asserted; the AWS row at offset 450 shows `USD -2.6137` and `USD -3.00` in the table and the detail; an Alibaba Cloud row shows `CNY 8.80`.

Routed responses cover what the sample lacks. A current read whose USD billed total is lower by one unit than the kept baseline shows the difference `USD -1.00` and no typographic minus. A response with every billed amount missing and one row with a missing service, subaccount, consumed quantity and unit, an exact zero pricing quantity, unclassified provenance and a `Z` instant is asserted in EN, KO and AR: Missing in the table without currency, dash or digit; "No values present" in the totals; Missing for the detail's billed cost, subaccount and service; the provenance label; the raw instant; Missing with Unit missing and no digit for the consumed quantity; the Exact zero note for the pricing quantity. In English only: the not-converted note, the account-list label, and the trace view's "Service: Missing" and "Row ID".

The existing cases still pass: baseline keep, refresh, discard and scope invalidation; 403, 503 and 409 responses removing totals, detail and comparison; changed source reference; late and aborted responses; paging, account search and selection; detail focus return and Escape; no storage write; computed 16px controls, 20px headings, zero radius, no shadow and neutral borders across the 36 variants. Those assertions do not cover the new group headings, state labels or 14px metadata, which were inspected in renders only. The mobile list and mobile inline detail are covered for visibility, not content.

Host: macOS arm64, Node 26.7.0, Python 3.11.7 from the prepared environment, Playwright 1.56.1 with Chromium headless shell build 1243. Other operating systems, browsers and assistive technology are unverified. Captures stay local under `output/validation/HB-154` and are excluded from Git.

# Findings corrected during verification

Three suite errors were root's and not product results: a locator matching two cells, a lowered total that the screen's validator correctly rejected because provider and month amounts no longer agreed, and a routed case that reused the cached query instead of reading again.

# Independent review

| Review | Result |
| --- | --- |
| Combined review of the plan and the actual change, with source-string technical writing | PASS with five medium and seven low findings: Arabic S04 screen name not matching the menu, Arabic amount alignment, two item IDs claimed but not registered, the new sign format untested, an unlabeled Missing heading on mobile, and smaller trace, unit, instant, wording and style items. Applied as listed in the plan |

This was one review after implementation, not a design review followed by a change review; the plan states why. The reviewer read files and renders and executed nothing. The corrections were verified by root's rerun and renders only. The reviewer did not read B03's source, so the statement that B03's pending differences lie inside its accepted scope is root's.

# Design and wording review

The ui-designer role compared the baseline and changed renders: conditional PASS with one blocking item, Arabic amounts on the opposite edge from their labels, and should-fix items for group boundaries, the repeated UTC line, adjacent state labels, the mobile heading and a wrapping amount. All were applied; the Arabic alignment was confirmed by root in the rerun render. The designer could not read the 390px renders at text level, saw no negative amount and no Arabic degraded state; mobile composition is therefore unreviewed by the designer.

Open for a native or practitioner reviewer: Arabic "لا قيم موجودة", the not-converted note, "التكلفة الفعلية" for effective cost and mixed diacritic spellings, which predate this change; Korean "정가·계약 비용과 단가". AI design and wording review is not owner FIX, native-language acceptance or practitioner approval.

# Accepted residuals

- The comparison view still shows its two read times as raw ISO strings in two formats.
- The S04 screen name in the detail note is a literal that mirrors the menu title, not a shared reference.
- A very long amount can wrap inside its table cell.
- U01, B04, B05, B06 and B07 keep older wording until their own tickets; B03 awaits the owner's change request.
- Concurrency and latency are not asserted (HB-152).

# Acceptance

Parent HB-109 remains awaiting explicit owner screen acceptance. This change alters the B08 candidate revision; no earlier acceptance exists for B08 and none is created. No ingestion, reset, reimport, schema or permission change, live account, customer data, deployment, payment or message occurred. Merge, read-back, ticket completion and machine release are recorded in the ticket only after they occur.
