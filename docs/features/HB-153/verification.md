---
type: VerificationRecord
title: B01 source-row identity verification
description: Records what was executed and reviewed for the B01 row identity and value-state refinement on the retained sample, with results, limits and accepted residuals.
status: draft
---

# Delivered behavior

[HB-153](https://linear.app/hybrid-billing/issue/HB-153/refine-b01-infrastructure-cost-composition-to-identify-source-rows-and) refines B01 from main `0ffb362` using the retained public and synthetic sample verified in HB-149. Root (Claude Code) owns planning, implementation, terminology writing, QA and integration; the ui-designer role led composition and compared the final renders; a separate read-only reviewer reviewed the design and the actual change. Machine: mac_mini. See the [plan](plan.md) and [reference evidence](references.md).

Each source row in the candidate and cost-item tables shows its billing account ID, subaccount ID, row ordinal in the delivery, sample provenance and full row ID without a disclosure; the delivery ID is added when the rows of a page come from more than one delivery. Billed and effective cost appear as two labeled lines; a null amount reads Missing without a currency and a zero amount carries Exact zero. The charge period is shown in UTC and the derived charge-start month has its own column. The edit controls sit in a band under each row, the inclusion choice names rows by a short labeled identity plus the labeled amount, and the relationship list shows that identity above the full IDs. `/api/focus-overview`, its DTO and validator, permissions, the browser draft format and keys, draft validation and the subtotal arithmetic (`model.ts`, `store.ts`) are unchanged. No request, storage key, URL parameter or link was added.

# Retained sample facts used

Read on 2026-11-12 through the console proxy by root; not independently recomputed. VMware: three rows annotated as synthetic test data, USD, charge-start month `2024-09`, ordinals `1000` to `1002`, subaccounts `synthetic-a` twice and `synthetic-b` once, billed costs `0.75`, `0.25`, `1.50`, one delivery. Oracle: seven rows from two deliveries; the first row carries billed cost `0.24` and effective cost exact `0`. AWS at offset 450: a row with billed cost `-2.6137` and effective cost `-3.00`. Alibaba Cloud: three rows, two of them CNY (`8.80`). No missing billed or effective cost was seen on these pages.

# Executed checks

| Check | Actual outcome |
| --- | --- |
| Identity unit tests (`identity.test.ts`) | PASS 8: labeled ordinal, subaccount and short row ID in EN and KO; labeled billing-account fallback; deterministic shortening above 28 characters; left-to-right isolation; amount text for value, exact zero, missing, negative and an 18-place CNY value; no currency, dash or zero beside Missing in three languages; option text; rows sharing an ordinal across deliveries stay distinct |
| Console unit suite (`npm test --prefix apps/console`) | PASS 296, previously 288; the ten existing B01 model and storage tests are unchanged |
| TypeScript check, Vite build, `tools/verify-build.mjs` | PASS; 438 artifact hashes and licenses; existing large-chunk advisory remains |
| B01 browser suite `apps/console/tools/infrastructure-cost-qa.mjs` on the retained sample | PASS; 0 page errors; reader requests sent one at a time, slowest about 1.7 seconds; see below. The unchanged suite also passed on main `0ffb362` before any edit |
| S04 browser suite `apps/console/tools/collected-source-qa.mjs` | PASS as regression for the moved labels file |
| Isolated backend regression (`tests/focus_overview`) | PASS 36; unchanged backend, regression only |
| Retained source, control and prior-result files | PASS 93 file hashes equal to the HB-150 preservation manifest after all reads |

Browser cases on real rows: the three VMware rows show ordinals, subaccounts, provenance, visible full row IDs and both labeled costs in the candidate table, and no disclosure remains; the cost-items view shows the same identity for the selected rows, does not scroll horizontally, names the inclusion option `Ordinal 1000 · Subaccount synthetic-a · Row ID … · Billed cost: USD 0.75`, and shows the target row ID and amount below the select; the saved browser draft contains no account, ordinal label or currency text; a synthetic 403 removes identity, cost lines and edit controls; the products view shows both relationship identities and the full target ID; Oracle shows the delivery ID on all seven rows and `USD 0.24` beside `USD 0.00` with Exact zero, with the Korean and Arabic zero labels asserted; AWS shows `USD -2.6137` and `USD -3.00` without a state label; Alibaba Cloud shows `CNY 8.80` and two separate currency subtotals when a USD and a CNY row are selected. The earlier cases still pass: required name, category edit, inclusion and cycle error, save and restore, dirty-scope keep and discard, failed storage, tab changes and replay of an obsolete response.

A routed response derived from the real VMware response covers what the sample lacks. Asserted in EN, KO and AR: a missing billed amount reads Missing with no currency, dash or digit; a missing effective amount beside an exact zero billed amount; a missing subaccount; an unrecognized provenance class under its field label; a non-UTC instant shown raw with the not-interpreted note; two rows sharing an ordinal across deliveries with the delivery ID shown; a 30-digit coefficient shown exactly and contained in its cell; a selection whose only amount is missing reads no values present without a currency; the first row's inclusion options are pairwise distinct and carry no dash. Asserted in EN only: the labeled billing-account fallback, the Exact zero note and the shortened 120-character account in option text, and a disabled option reading `Billed cost: Missing (Different currency or charge period)`; the Korean and Arabic Missing amount text is covered by the unit test.

Layout, EN, KO and AR at 1440 and 390: no page overflow in both views; in the candidate view 16px screen text, row-ID halves inside their cell and left-to-right identifiers and amounts; in the cost-items view no in-region horizontal scroll, zero radius, no shadow and neutral computed borders (channel spread at most 32) on row groups, rows, cells, controls and the band. At 1440 the cost-items view asserts a 1px separator under each item and no background on a hovered item; the 390 variants assert 44px checkbox targets and controls. Oracle rows in the candidate view at 1320 and 1024 assert no overflow and each amount inside its cell on one line; the cost-items view at 1024 asserts 44px controls.

Host: macOS arm64, Node 26.7.0, Python 3.11.7 from the prepared environment, Playwright 1.56.1 with Chromium headless shell build 1243. Other operating systems, browsers and assistive technology are unverified. Captures and `result.json` stay local under `output/validation/HB-153` and are excluded from Git.

# Findings corrected during verification

- The first run failed because a paged read requires its snapshot; the suite now reads the AWS snapshot first. A font-size assertion targeted the document body, which is 15px outside the screen root; it now targets the screen. Both were suite errors, not product results.
- Rendered inspection by root found three product defects before review: the 720px inner scroll left about three 240px rows in view, an uninterpreted instant overlapped the month column, and OCID subaccounts wrapped to six lines at 1280px. The inner scroll was removed, raw instants wrap, and long identifiers take the full identity width.
- The formatter wrote invisible isolate characters into source files; they were replaced by explicit escapes.

# Independent review

| Review | Result |
| --- | --- |
| Design review of the first plan | FAIL without blockers: ordinal not unique across deliveries, option text and the missing amount, Unclassified collision, remaining superseded wording, plus seven low items. Plan revised |
| Actual change and source-string technical-writing review | PASS with three medium and ten low findings: a dash after Missing in a disabled option, this record missing, plan overclaims, Korean 청구 계정 beside 결제 계정, and smaller wording and test items. Applied as listed in the plan |
| Narrow re-review of the fixes | FAIL without blockers: the wording, group label, target amount and Korean fixes confirmed correct; the new item separator was cancelled by a shared table rule and a hover tint remained; one assertion could not fail; this record overstated what four assertions cover. Corrected: separator on the row group, hover reset, cell-containment assertion, a long exact amount case, and the narrowed statements above |

The reviewer read files and renders only and executed nothing; all executed results above are root's. The corrections after the narrow re-review were verified by root's computed-style assertions and rerun, not by a further review.

# Design and wording review

The ui-designer role returned the composition (edit band instead of a sticky column, fixed columns, labeled cost lines, cards) and compared all 16 final renders of the first passing run with its proposal: conditional PASS, no blocking defect. Its should-fix items were applied afterwards: a wider cost column, cards below 1320px with 44px controls, a 16-character rule and spacing for full-width identifiers, the target amount under the target row ID, and no tint in the cost-items view. These corrections were verified by root's rerun, the added 1320, 1024 and computed-style assertions and rendered inspection, not by a second designer pass. It did not see focus or hover states, native select popups, the 768 to 1199 range or a 25-row page.

Wording was reviewed in EN, KO and AR by the change reviewer from source strings and four renders. Open for a native or practitioner reviewer: Arabic "الترتيب" for the ordinal and the literal "المفتاح الفريد الوحيد" in the row-ID sentence, mixed Arabic diacritic spellings that predate this change, English and Arabic Unclassified for both the cost category and the provenance value (kept apart by the field label), and "Unspecified service" for a null service beside Missing for other null values. AI design and wording review is not owner FIX, native-language acceptance or practitioner approval.

# Accepted residuals

- The short identity in option text and control labels carries no delivery token; the row-ID prefix keeps it distinct and the full target ID is shown.
- A very long account identifier renders in full in its cell; only option text is shortened. The suite covers 120 characters, not the 4,096-character validator bound.
- The table header is not sticky and a 25-row page is long; each cost line repeats its basis label.
- The uncomputed-subtotal wording is defensive and not reached or asserted.
- A disabled option for an exact zero amount shows two parenthesized notes in a row.
- An amount wider than the cost column wraps inside its cell; the currency code can then sit on its own line.
- The month header wraps to three or four lines; native checkbox corners were not measured; the sticky navigation rail appears mid-page in full-page captures, as it did on main.
- U01, B04, B05, B06, B07 and B08 still use "charge interval" or "no value" wording and are aligned in their own tickets.
- Requests are sent one at a time; concurrency and latency are not asserted (HB-152).

# Acceptance

Parent HB-101 remains awaiting explicit owner screen acceptance. This change alters the B01 candidate revision, and moving the shared labels file changes the build fingerprint of S04 without changing its rendered behavior, which the S04 suite confirms; no earlier acceptance carries forward. No test, build or review created owner acceptance. No ingestion, reset, reimport, schema or permission change, live account, customer data, deployment, payment or message occurred. Merge, read-back, ticket completion and machine release are recorded in the ticket only after they occur.
