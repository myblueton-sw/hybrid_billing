---
type: VerificationRecord
title: B06 charge-row evidence verification
description: Records what was executed and reviewed for the B06 charge-row terms, value states and scope identity on the retained sample, with results, limits and accepted residuals.
status: draft
---

# Delivered behavior

[HB-159](https://linear.app/hybrid-billing/issue/HB-159/align-b06-charge-row-evidence-with-the-s04-source-field-terms-and) refines B06 from main `796b5f9` using the retained public and synthetic sample verified in HB-149. Root (Claude Code) owns planning, implementation, terminology writing, QA and integration; a separate read-only reviewer reviewed the plan before implementation and the change afterwards; the ui-designer role compared renders. Machine: mac_mini. See the [plan](plan.md).

In the selected-group detail, each charge row shows its two costs through the shared components (Missing without currency, Exact zero as a note), a "Service: Missing" heading when the service is null, charge and billing period instants in UTC, delivery ID, row ordinal, sample provenance and the full row ID. Scope identity in the detail, the group table and the account scope uses 공급사, 결제 계정 ID, 하위 계정 ID and 계정 범위 ID. Group envelopes are labeled start and end lines. Aggregates and metadata sets without present values read "No values present", counts read "Present / missing", and every amount uses the shared format. B06's own commitment field states are unchanged. `/api/focus-commitments`, `contract.ts`, `api.ts`, grouping, paging and permissions are unchanged.

# Retained sample facts used

Read on 2026-11-12 through the console proxy by root. The AWS scope has two commitment ID groups and four identified rows out of 942; the first group has three rows classified Used, all with billed and effective cost exactly zero, one delivery, annotated as a public sample. The sample therefore has no missing, negative or non-zero commitment row cost; those states are covered by routed responses.

# Executed checks

| Check | Actual outcome |
| --- | --- |
| Console unit suite (`npm test --prefix apps/console`) | PASS 302, unchanged count; the shared components keep their unit test from HB-158 |
| TypeScript check, Vite build, `tools/verify-build.mjs` | PASS; 438 artifact hashes and licenses; existing large-chunk advisory remains |
| B06 browser suite `apps/console/tools/commitment-benefits-qa.mjs` on the retained sample | PASS; 0 page errors; see below. The unchanged suite also passed on main `796b5f9` before any edit |
| S04 browser suite | PASS as regression |
| Isolated backend regressions (`tests/focus_overview`, `tests/focus_commitments`) | PASS 70; unchanged backend |
| Retained source, control and prior-result files | PASS 93 file hashes equal to the HB-150 preservation manifest after all reads |

Every row disclosure, the identity disclosure and the account scope are opened before assertions. On the real group at 1440 in English: three charge rows, each cost reading `USD 0.00` with Exact zero; the row ID, ordinal and delivery ID equal to the response; Public sample; `2024-09-20 20:00:00 UTC` and the billing period end; the group envelope as labeled start and end lines; identity labels Provider, Billing account ID, Subaccount ID and Account scope ID; no superseded label in the detail text; the shared labels in the group table and the account list; the "Present / missing" wording.

Routed responses are built from the fetched response by difference, so counts and coefficients of the group, its status bucket and the currency total stay consistent with the rows and with B06's validator. In EN, KO and AR at 1440: a null billed cost reads Missing and the group reads two present and one missing; a negative effective cost reads `USD -2.00` in the row and in the group aggregate; a null service heading; a null subaccount reads Missing in the detail identity, outside any left-to-right isolate, and in the group table; a `Z` instant is shown as collected; an empty metadata set reads "No values present"; no superseded Korean label remains in the detail text; in Arabic the row ID ends on its label's edge. With every row null on the effective basis the aggregate reads "No values present" and the row reads Missing.

Layout with the detail and all disclosures open in EN, KO and AR at 1440, 1280, 1279, 1024, 761, 760, 641, 640 and 390: no page overflow, no child of a detail, identity, period line, value or account box wider than its box, and at least one value node present. The degraded response at 390 in Korean and Arabic passes the same checks. A denied read leaves no detail, group, source value, row cost or identity node. The existing group, detail, paging, account, stale and failure cases and the 24 view, language and width variants still pass.

Not asserted: left-to-right isolation of Missing outside the detail identity (confirmed by reading only), and the identity labels of the group table and account list in Korean and Arabic.

Host: macOS arm64, Node 26.7.0, Python 3.11.7 from the prepared environment, Playwright 1.56.1 with Chromium headless shell build 1243. Other operating systems, browsers and assistive technology are unverified. Captures stay local under `output/validation/HB-159` and are excluded from Git.

# Findings corrected during verification

The first version of the suite addressed a group by URL without its snapshot, which the screen correctly rejects; the suite now selects the group after a fresh load. This was a suite error, not a product result.

# Independent review

Reviewer in each round: the repository's read-only `independent-reviewer` agent role (AI), 2026-11-12. It read files and renders and executed nothing.

| Review | Result |
| --- | --- |
| Design review of the plan, before implementation | FAIL without blockers: B06-scoped styles for the shared components missing while the plan claimed no layout change; assertions vacuous inside closed disclosures; the four uses of the old no-value helper; localized nodes inside left-to-right isolates; glossary and note wording for the period terms; Arabic envelope labels; the account scope ID label; one amount format; B06's own breakpoints; routed response validity. The plan was revised before implementation |
| Review of the actual change | PASS, no high or blocking finding; money semantics, the renamed helper, the detail rework and the styles confirmed by reading. Medium: this record missing before merge. Low: routed responses consistent only because sample costs are zero; the layout check measuring direct children only; an unused import; plan claims broader than the assertions; wording of two sentences; Arabic items for native review. Applied as listed in the plan; the corrections were verified by root's rerun and renders, not by a further review |

# Design and wording review

The ui-designer role compared baseline and changed renders: conditional PASS, no blocking defect, with the limits that it read the mobile renders only at block level and did not open the 1024, 761, 760, 641 and 640 renders. Should-fix items: the stacked full-width panel showing about sixteen full-width evidence lines, no grouping inside the evidence list, and record boundaries looking like field separators. All were applied, together with the polish items for the heading label, period label alignment, non-wrapping counts and account tiles. Not applied: a lighter Exact zero note, because the note's weight is the shared presentation; the designer observed that six bold notes outweigh the caveat that no unused rows were observed. The corrected renders were checked by root and the suite, not by a second designer pass.

Open for a native or practitioner reviewer: Arabic "الخدمة: مفقود" (state not agreeing in gender with the noun), "لا قيم موجودة", mixed spellings of the word for provider; Korean 값 없음 for an aggregate without present values, which sits close to the retired 원천 값 없음. AI design and wording review is not owner FIX, native-language acceptance or practitioner approval.

# Accepted residuals

- A `Z` instant is UTC but is shown as collected with the not-converted note, because the shared parser reads only `+00:00`; real rows carry `+00:00`.
- The query's observed-at and coverage timestamps stay as collected; they are not row values.
- The Exact zero note appears under both costs of every real commitment row, because every retained commitment row costs zero.
- U01 and B07 keep older wording until their own tickets; HB-152 is unchanged.

# Acceptance

Parent HB-106 remains awaiting explicit owner screen acceptance. This change alters the B06 candidate revision; no acceptance exists for B06 and none is created. No ingestion, reset, reimport, schema or permission change, live account, customer data, deployment, payment or message occurred. Merge, read-back, ticket completion and machine release are recorded in the ticket only after they occur.
