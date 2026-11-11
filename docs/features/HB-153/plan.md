---
type: ImplementationPlan
title: B01 source-row identity and S04-aligned value states
description: Defines how B01 identifies each collected source row and presents its values with the meanings fixed for S04, using only the current authorized row contract.
status: draft
---

# Scope and ownership

[HB-153](https://linear.app/hybrid-billing/issue/HB-153/refine-b01-infrastructure-cost-composition-to-identify-source-rows-and) is a scoped child of B01 HB-101 and follows HB-149, HB-150 and HB-151 from main `0ffb362`. The owner continued the screen-by-screen refinement on 2026-11-12; B01 is the next entry of the Costs and allocation menu after S04. Root (Claude Code) owns planning, frontend implementation, terminology writing, QA and integration. The ui-designer role leads composition and final render comparison. A separate read-only reviewer checks money semantics, authorization and clearing behavior, the actual change and copy. Machine: mac_mini. Onboarding stays deferred. No owner FIX is created.

# Observed baseline

Facts read on 2026-11-12 from the canonical preview, the protected reader through the console proxy and the unchanged B01 browser suite, which passed on main `0ffb362`:

- The default VMware source returns three rows, all annotated `synthetic-contract-fixture`, USD, charge-start month `2024-09`, with identical service name, provider, charge period and billing period. They differ in billed cost (`0.75`, `0.25`, `1.50`), row ordinal in the delivery (`1000`, `1001`, `1002`), row ID, and subaccount (two rows `synthetic-a`, one row `synthetic-b`; two account scopes).
- The source-candidate and cost-item tables show service, provider, month, two raw ISO instants, the amount on the selected basis and the category. Row ID, account scope, annotation and both cost bases sit inside a collapsed disclosure. The three rows therefore read as the same row with different amounts.
- The inclusion-target selector labels a row by service name plus the first eight ID characters, so for this sample the options differ only in that prefix.
- In the baseline capture of the cost-items view at 1280px the table is scrolled horizontally and the source-row column is outside the viewport while category, inclusion and reference are edited.
- The declared-relationship list on the products tab identifies both ends by 64-character row ID only.
- Wording diverged from S04 after HB-151: "Charge interval / 과금 구간" where the glossary now fixes "Charge period / 과금 기간"; "No value / 값 없음" for a null row amount where the glossary fixes "Missing / 누락"; an exact zero row amount has no text state; the annotation is printed as the raw enum; instants are raw ISO strings; the derived charge-start month shares a cell with the source charge period under the charge-interval header; the toolbar says 공급자 where S04 says 공급사.
- Root's reads on 2026-11-12 found no missing billed or effective cost on the pages used here; this is an observation on those pages, not a recomputed whole-sample count. Exact zero (Oracle), negative (AWS, offset 450) and CNY (Alibaba Cloud) amounts occur in other providers' rows, which B01 can select through its source filter. The Oracle page spans two deliveries.

# Decision

Frontend-only presentation change. `/api/focus-overview`, its DTO and validator, permissions, the browser draft format and storage keys, draft validation and the subtotal arithmetic in `model.ts` are unchanged.

Alternatives considered:

1. Add a row deep link from B01 to the S04 row detail. Rejected for this ticket: S04 has no authorized row locator in its URL contract, and HB-151 deliberately added none.
2. Open the S04 row-detail component inside B01. Rejected: it duplicates a second screen's review unit inside B01 and widens the change beyond identity and wording.
3. Show the distinguishing fields the row already carries and reuse the S04 vocabulary. Chosen.

# Row identity

A source row is identified on B01 by values the row DTO already supplies:

| Part | Field | Origin class (HB-151) | Presentation |
| --- | --- | --- | --- |
| Service and provider | `service`, `provider` | From source columns | Unchanged; a null service keeps "Unspecified service" |
| Account | `billing_account`, `sub_account` | From source columns | Billing account ID and subaccount ID as labeled values; a null subaccount reads Missing |
| Position in delivery | `ordinal`, `batch` | Delivery lineage | "Row ordinal in delivery" value; the delivery ID is added to every row whenever the rows of the page come from more than one delivery |
| Row ID | `id` | Derived by Hybrid Billing | Full 64-character ID, left-to-right, breaking only at its midpoint |
| Sample provenance | `evidence_class` | Sample annotation | Glossary label through `provenanceClass`; unrecognized values read Unclassified |

All parts are visible without opening a disclosure in both tables. The row ID is the only unique key: the ordinal is unique only inside one delivery and tenant, and the screen says so above the table. The disclosure is removed. The cost cell shows the selected basis and the other basis as two lines, each with its own basis label, with no rule or total between them, so that billed and effective cost stay separate and are never summed or differenced. Sample provenance always appears with its field label because its Unclassified value would otherwise read like the Unclassified cost category.

A pure helper builds one short identity text per row: labeled ordinal, labeled subaccount (or labeled billing account when the subaccount is missing; account text longer than 28 characters is shortened to its first and last 12) and the labeled first eight ID characters. Each left-to-right token is wrapped in Unicode isolates because native option text and `aria-label`s cannot hold elements. It is used for the inclusion-target option text, the control `aria-label`s, the group label of the edit controls and the products-tab relationship list, which keeps the full IDs as well; a relationship end that is not in the current page data shows the full ID only. The helper performs no arithmetic and adds no stored field. The option text also carries the amount on the selected basis with its basis label; it branches on the value state before formatting, so a missing amount reads "Billed cost: Missing" and never reaches the money formatter, and an exact zero carries the Exact zero note.

The response-level join from `row.account` to `accounts[].tenant` is not used, for the reason recorded in HB-151.

# Value states and periods

Row amounts on the selected basis and on the other basis use the HB-151 states through `valueState`:

| State | Presentation |
| --- | --- |
| Missing | "Missing" alone, without currency, dash or 0 |
| Exact zero | The exact formatted zero with the "Exact zero" note |
| Value | Exact decimal with currency; negative values keep their sign and are not labeled credit or refund |

Subtotals keep their own wording: a subtotal without present values reads "No values present / 값 없음"; a subtotal that was not computed reads "Unavailable / 산정 불가", a defensive state the current screen flow does not reach and no test asserts; a zero subtotal is not labeled exact zero because it can be a net of offsetting rows; the count of rows without an amount reads "Rows with a missing amount". The arithmetic and its partial-state rules are untouched.

The charge period is shown as start and end through the shared `Instant` presentation (non-wrapping `YYYY-MM-DD HH:MM:SS`, UTC stated in the header; an instant that is not a `+00:00` string is shown raw with "Time format not interpreted"). The derived charge-start month becomes its own column labeled as a Hybrid Billing value. "Interval" becomes "period" in headers, the validation message and the inclusion option hint, which names the charge period in all three languages and is appended in parentheses so that no dash follows a Missing amount. The billing period is not added to B01; inclusion validation compares charge period and charge-start month only, as today.

# Screen behavior

- Summary view (source candidates) and cost-items view: the table gains visible identity as specified above. Column composition, widths and the card order follow the ui-designer specification.
- Inclusion-target select: option text from the identity helper; the full target row ID and the target's amount on the selected basis stay below the select because the closed select truncates long option text.
- Products view: each declared relationship shows the identity text of both rows above the full IDs.
- Wording aligned in the same screen: 공급사 in the toolbar and the source notice; 샘플 for 표본 in the notice heading; "matching charge period" in the notice; 누락 in the subtotal footnote; an unentered evidence reference reads "No reference entered" so that it does not collide with the Missing data state.
- Layout: the edit controls (category, included in, evidence reference, verification and approval) sit in a band under each row instead of further table columns, so identity stays in view while editing and the table no longer scrolls horizontally at 1280px. The table region no longer has its own vertical scroll. Below 1320px rows become cards with 44px controls.
- Unchanged: toolbar behavior, source notice facts, review amounts layout, draft form, 25-row bounded selection, save and restore, unsaved-input guard, page controls, evidence-still-required text, approvals view, route, four tab keys, review control.

Selection and clearing are untouched: the draft stays bound to the fresh authorized query, failed or denied reads remove source rows and draft, and no browser draft is used as a fallback.

Stable item IDs: the seven existing `B01:` items are preserved. Added: `B01:table:row-identity` (visible account, ordinal, row ID and sample provenance in both tables, the inclusion choice and the relationship list) and `B01:state:value-missing-zero` (Missing and Exact zero row amounts, subtotal wording).

# Structure

`sources/pages/collected-source-data/labels.tsx` moves to `sources/collected-focus/labels.tsx` because B01 becomes its second consumer and the following cost, allocation and billing screens will use the same vocabulary; S04 imports are updated and its behavior is unchanged. The `collected-token` style used by `Instant` is defined only under the S04 root class, so B01 styles the same element inside its own stylesheet instead of sharing S04 CSS. `CostItems.tsx` (403 lines) is split by responsibility: `identity.ts` (identity and option text), `RowFacts.tsx` (identity, period and cost cells), `Declaration.tsx` (edit controls) and `RelationEnd.tsx` (relationship end). No file approaches 1,500 lines.

Allowed files: `apps/console/src/modules/pricing/pages/infrastructure-cost/*`, `apps/console/src/modules/sources/collected-focus/labels.tsx` (moved), import lines in `apps/console/src/modules/sources/pages/collected-source-data/*`, `apps/console/tools/infrastructure-cost-qa.mjs`, `apps/console/package.json` (test list, for the new `identity.test.ts`), `apps/console/README.md` (B01 section if wording changes), `docs/features/HB-153/*`, `docs/ui/terminology/product-glossary.md` (affected-screens column and B01 note).

# Terminology

The glossary entries `missing-row-value`, `exact-zero-value`, `charge-period`, `derived-value`, `sample-annotation` and the provenance labels are applied to B01 without new terms; a short B01 note is added under the HB-151 terms. The same superseded wording remains in U01, B04, B05, B06, B07 and B08 source files. Those screens are separate review units with their own tickets in the refinement sequence and are not edited here; the verification record lists them.

# Verification

- Unit: identity helper (ordinal, subaccount present and missing, short ID), option text with missing, zero, negative and non-USD amounts; existing ten B01 model tests unchanged and passing; S04 field tests unchanged.
- Console unit suite, type check, build and build verifier.
- B01 browser suite on the retained sample, extended: the three VMware rows carry distinct visible identity text in the candidate table and the two selected rows in the cost-items table; inclusion options are pairwise distinct; an Oracle row shows Exact zero on the effective basis; an AWS row shows a negative amount; a CNY row shows CNY; a routed response with a null basis amount shows Missing without a currency and the subtotal wording; existing select, classify, inclusion, cycle, save, restore, storage failure, dirty-scope, 403 and stale-response cases still pass. EN/KO/AR at 1440 and 390: no page overflow, 16px body, neutral computed borders, no radius or shadow, left-to-right identifiers and money, row ID inside its cell.
- S04 browser suite rerun as regression for the moved labels file. Requests are sent one at a time as in HB-151; concurrency and latency are not asserted (HB-152).
- Retained source and control file hashes compared with the HB-150 preservation manifest after all reads. No ingestion, reset, benchmark or Graphify measurement.
- Independent review of this plan and of the actual change; ui-designer composition and final render comparison; technical-writing review of KO/EN/AR wording in rendered context. AI review is not native-language, practitioner or owner acceptance.

# Design review resolution

The independent design review of the first draft returned FAIL without blockers. Resolved in this revision: the row ID is stated as the only unique key and the delivery ID accompanies the ordinal when a page spans deliveries (1); option text branches on the value state, always carries the basis label and isolates left-to-right tokens (2); sample provenance keeps its field label (3); the remaining superseded B01 wording is aligned (4); not-computed and no-values subtotals read differently (5); both cost lines are labeled (6); the account token is labeled and bounded (7); uninterpreted instants carry the S04 note (8); storage, clearing, degraded and per-language cases are added to the verification (9); the test list change is unconditional (10); baseline overclaims are corrected (11). Composition follows the ui-designer specification, with two deviations decided by root after rendering: the 720px inner vertical scroll is removed because 240px rows left three rows in view, and identifiers longer than 16 characters take the full identity width because GUID and OCID accounts wrapped to six lines at 1280px.

# Change review resolution

The independent review of the actual change returned PASS with three medium and ten low findings; the ui-designer comparison returned conditional PASS with eight should-fix items. Applied: parenthesized option hint naming the charge period (F1, T6); this verification record (F2); plan wording (F3); group label by reference instead of a repeated label (F5); 결제 계정 in the source notice (T1); plainer English for the row-ID sentence (T3); 과금 기간 in the Korean validation message (T7); a 208px cost column; cards below 1320px with 44px controls; the 16-character rule and spacing for full-width identifiers; the target amount under the target row ID; no tint in the cost-items view, with item separators instead. The narrow re-review then found the separator cancelled by a shared table rule and a remaining hover tint; both were corrected and are asserted by computed style. Accepted without change and recorded in the verification record: the short identity carries no delivery token (F4), very long identifiers render in full (F7), the header is not sticky (F8), the uncomputed-subtotal wording is untested (F9), the month header height, the native checkbox corner and the list length on 25-row pages.

# Deferred

Row deep link to S04 and S05 evidence locator; authoritative infrastructure records, policy ownership, product catalog, approval workflow and server persistence (HB-101 remaining integration); terminology alignment of U01, B04, B05, B06, B07 and B08; HB-152.

Planning trace: HB-06 lineage and HB-07 exact values, NULL versus zero and period semantics, bounded to the retained sample; B01 spec table columns "구성요소 → 출처 → 대상 기간 → 금액·통화".
