---
type: ImplementationPlan
title: B07 source-row evidence in the S04 source-field terms
description: Defines how B07 presents its collected marketplace source rows and scope identity with the shared source-field vocabulary and value states, without changing its projection, field states, classification or aggregates.
status: reviewed
---

# Scope and ownership

[HB-160](https://linear.app/hybrid-billing/issue/HB-160/align-b07-source-row-evidence-with-the-s04-source-field-terms-and) is a scoped child of B07 HB-107 and follows HB-151 through HB-159 from main `554fccb`. The owner continued the sample-backed screens on 2026-11-12; B07 is next. Root (Claude Code) owns planning, frontend implementation, terminology writing, QA and integration. A separate read-only reviewer reviews this plan before implementation and the actual change afterwards; the ui-designer role compares the final renders. Machine: mac_mini. No owner FIX is created.

# Observed baseline

Facts read on 2026-11-12 from the code at `554fccb` and the unchanged B07 browser suite, which passed with 24 captures:

- B07 reads `/api/focus-marketplace`, a server projection with its own contract and validator. Eight marketplace fields (publisher, issuer, description, category, charge class, frequency, SKU, commitment reference) carry a state: observed value, source null, field absent, blank, unmapped version or invalid type (`copy.ts:20-39`). Service, subaccount and the two costs are plain nullable (`contract.ts:35-55`).
- One `Money` component (`SourceViews.tsx:23-37`) formats row costs in the source table (`:570`, `:575`), in the row detail on the buyer view (`:183`), in a third site (`:660`) and, through `MeasureValue`, the candidate aggregates (`:434`, `:479`). A null reads "No observed amount / 관측 금액 없음" in every case; a negative amount has a leading typographic minus before the currency (`:19-22`); an exact zero has no text state.
- The row detail's "Period and source references" disclosure (`:248-311`) joins values in pairs: "Charge interval" and "Billing interval" as raw ISO strings, "Billing account / sub-account" with "None in source / 원천 값 없음", "Source batch / ordinal", a raw "Source provenance class" enum, "Canonical account reference / 원천 계정 참조" for the product-derived account hash, and a tenant reference. Every value sits inside a left-to-right isolate. The row ID is shown above the lists as a "technical source reference" (`:171-178`).
- The source table shows "Service unavailable / 서비스 값 없음" for a null service and a column "Account / charge start" with the billing account and the raw instant split by `slice` (`:526-565`).
- The account scope lists accounts with 청구 계정, 공급자 and "None in source" (`AccountScope.tsx:93-110`).
- Korean copy says 공급자 in the toolbar and the coverage note (`Screen.tsx:217`, `:435`).
- The row button reads "Inspect source / 원천 근거 보기"; B04 and B05 use the same Korean for their evidence toggle with a different English label (residual recorded in HB-157 and HB-158).

# Decision

Frontend-only wording and value-state change. The structure of the screen is unchanged; inside the row detail, joined pairs become one field per line, periods become UTC instants and costs can carry a note line, and the shared components need B07-scoped styles. `/api/focus-marketplace`, `contract.ts`, `api.ts`, classification, paging, permissions and URLs are unchanged.

| Area | Decision |
| --- | --- |
| Row costs | Through the shared row-amount component: Missing without currency, dash or digit. The Exact zero note appears in the row detail only, not in the 25-row table or the mobile card, as on B08 |
| Aggregates | A renamed sum component serves the candidate and the insufficient-classification aggregates only: "No values present / 값 없음 / لا قيم موجودة" when no value is present; present and missing counts unchanged; never Exact zero |
| Amount format | B07's formatter delegates to the shared one, so rows and aggregates read `USD -2.00`. B07's candidate amounts are sums of source costs, the case B08 and B06 moved to the shared format |
| Row detail references | One field per line with the shared labels: charge period start and end, billing period start and end as UTC instants; billing account ID; subaccount ID (Missing when null); account scope ID in place of "Canonical account reference"; delivery ID; row ordinal in delivery; sample provenance by its glossary label; FOCUS version and tenant reference keep their labels. The disclosure is titled "Period, account and delivery references". The row ID stays always visible above it as a labeled "Row ID" field in 14px monospace, broken only at its midpoint, with the sentence that it is not a transaction ID |
| Source table | Null service reads Missing under its field label on the mobile card and plain Missing in the desktop cell under the "Source charge" header; the second column is headed "Billing account ID / Charge period start" and shows the instant through the shared presentation, which carries its own UTC mark or not-converted note; date and time may wrap at the space in the cell. The credits view's second column is unchanged. The mobile card shows the description through B07's field states. Publisher and issuer are printed directly because the validator rejects a row where either is not an observed value |
| Account scope | Shared labels for billing account ID, subaccount ID (Missing when null) and provider; the summary, note and selected-scope line use B06's merged wording ("One account scope selected", account scope ID derived by Hybrid Billing). At 767px and below an account button stacks its four lines |
| Field states | Unchanged: B07's observed value, source null, field absent, blank, unmapped version and invalid type stay its own terms |
| Wording | 공급사 in the toolbar and the coverage note. "Inspect source / 원천 근거 보기" stays; the label collision with the B04 and B05 toggle remains the recorded residual |
| Style | B07-scoped rules for the shared classes (state, state note, value, instant, time zone) and a code class for the row ID with nowrap halves. Left-to-right values stay inline inside their cells so that they sit on their label's edge in Arabic. Localized nodes (Missing, the provenance label) are never nested in a left-to-right isolate, so the detail list carries rendered nodes instead of strings |

Collection-slice wording (구간), "선언 없음", "필드 없음" and the error copy are B07's own terms and stay. The query's observed-at and coverage timestamps stay as collected; they are not row values. A `Z` instant is shown as collected with the not-converted note, the residual accepted since HB-158.

Stable item IDs: existing `B07:` items are preserved. Added: `B07:state:value-missing-zero`, registered in the screen and recorded here.

# Allowed files

`apps/console/src/modules/pricing/pages/marketplace/*` except `api.ts`, `contract.ts` and `api.test.ts`; `apps/console/tools/marketplace-qa.mjs`; `apps/console/README.md`; `docs/features/HB-160/*`; `docs/ui/terminology/product-glossary.md`. The shared components in `sources/collected-focus/` are used, not changed. `SourceViews.tsx` stays one file (under the 1,500-line limit); no move is mixed with the wording change.

# Verification

- Console unit suite, type check, build and build verifier. No new logic; the shared components keep their unit test from HB-158.
- B07 browser suite on the retained sample, extended. The row detail's disclosure and the account scope are opened before assertions, absence checks use text content, and a non-zero count of value nodes is asserted.
  - At 1440 on a real row: detail fields by attribute (UTC instants, billing account ID, account scope ID, tenant reference, delivery ID, ordinal, provenance label, full row ID) equal to the response; no superseded label in the detail text; the table's account column and header; the account list labels.
  - Routed responses built from the fetched response by difference, valid for B07's validator and consistent with their own aggregates, in three languages at 1440: a null billed cost (Missing in the table and the detail, aggregate counts adjusted), the real row's exact-zero effective cost (note in the detail only) and a negative billed cost (shared format in the row and the aggregate), a null service, a null subaccount, a `Z` instant; then a selection whose rows are all null on one basis so that the aggregate reads "No values present". Missing nodes have no left-to-right isolate ancestor; in Arabic the row ID ends on its label's edge.
  - Layout with a row detail and its disclosure open in EN, KO and AR at 1440, 1251, 1250, 1024, 768, 767 and 390 (B07's breakpoints are 1250 and 767): no page overflow and no child of a detail, list, cell or account box wider than its box. The degraded response at 390 in KO and AR.
  - A denied read leaves no detail, row or source value node. Existing view, detail, paging, account, stale and failure cases and the 24 view, language and width variants still run.
- S04 browser suite as regression; isolated `tests/focus_overview` and the marketplace backend tests unchanged; retained file hashes against the HB-150 manifest.
- Independent review of the actual change; ui-designer render comparison; technical-writing review of the changed strings. AI review is not native-language, practitioner or owner acceptance.

# Deferred

Provider, legal, settlement and eligibility integrations; one Korean label for the row-detail button and the evidence toggle across B04, B05, B07 and B08; U01 terminology in its own ticket; HB-152.

# Design review resolution

The independent design review of 2026-11-12 (read-only, nothing executed) returned FAIL without a blocker. Resolution:

| Finding | Resolution |
| --- | --- |
| F1 routed cases | Restated: with the sample's single candidate row a null billed cost is necessarily the all-null aggregate (Missing with "No values present", 0 / 1); a synthesized second row gives Missing beside a present sum (1 / 1) with candidate count 2 compensated in the no-difference count; a negative billed cost equals its aggregate. Exact zero uses the real row's effective cost. Every routed response is patched from the live response of the same request, identically for the row and the selected row |
| F2 account-scope strings | Adopted with B06's wording |
| F3 mobile description and heading | Adopted |
| F4 aggregates | Both named; the insufficient-classification disclosure is opened and routed to a null basis; the mobile card follows the table rule |
| F5 row ID visibility | Row ID stays always visible |
| F6, F7 widths | Table instant wraps at the space; whole-instant nowrap only in the detail; row ID in 14px monospace |
| F8 header | Shared instant presentation in the cell, header without "(UTC)" |
| F9 wording | Sum component renamed; the unclassified note says "not an observed value" in B07's own state term; disclosure retitled; basis labels and B07's reference labels stay |
| F10 Arabic provider spelling | Residual: the shared label reads المزود, B07's toolbar and coverage note read المزوّد; left for native-language review |
| F11 verification | Visible-scoped assertions, a fresh load per width, disclosures opened after selection, denied read asserts no value or account node |
| F12 line references | The baseline line numbers above are approximate; no conclusion depends on them |
| F13 split | Dropped |
