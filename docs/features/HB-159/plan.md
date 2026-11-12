---
type: ImplementationPlan
title: B06 charge-row evidence in the S04 source-field terms
description: Defines how B06 presents the collected charge rows and scope identity behind its commitment groups with the shared source-field vocabulary and value states, without changing its projection, field states or aggregates.
status: draft
---

# Scope and ownership

[HB-159](https://linear.app/hybrid-billing/issue/HB-159/align-b06-charge-row-evidence-with-the-s04-source-field-terms-and) is a scoped child of B06 HB-106 and follows HB-151, HB-153, HB-154, HB-157 and HB-158 from main `796b5f9`. The owner continued the sample-backed screens on 2026-11-12; B06 is next. Root (Claude Code) owns planning, frontend implementation, terminology writing, QA and integration. A separate read-only reviewer reviews this plan before implementation and the actual change afterwards; the ui-designer role compares the final renders. Machine: mac_mini. No owner FIX is created.

# Observed baseline

Facts read on 2026-11-12 from the code at `796b5f9` and the unchanged B06 browser suite, which passed:

- B06 reads `/api/focus-commitments`, a server projection with its own contract (`contract.ts`) and validator (`api.ts`). Commitment fields carry a state: value, source null token, column absent, blank, unmapped version or invalid type (`copy.ts:24-36`). These are richer than the Missing state of the other screens and are B06's own glossary terms.
- Charge rows in the group detail (`GroupDetail.tsx:457-547`): a null billed or effective cost reads "No source value / 원천 값 없음"; an exact zero has no text state; amounts use B06's formatter with a leading typographic minus (`values.ts:4-7`); a null service reads "Service unavailable / 서비스명 없음"; lineage is labeled "Canonical row reference / 원천 행 참조", "Evidence classification / 근거 분류" with the raw enum, "Delivery batch / 수집 배치" and "Source row ordinal / 원천 행 순번"; the row's charge and billing periods are two raw ISO strings joined by a dash under "Source charge interval envelope" and "Source billing interval envelope".
- The same `Intervals` component renders the group-level envelope, which is the earliest start through the latest end of the group's rows (glossary "source interval envelope / 원천 기간 범위").
- Scope identity in the group table, the group detail and the account scope uses 공급자, 청구 계정 and "None in source / 원천 값 없음" for a null subaccount (`GroupTable.tsx:110-124`, `GroupDetail.tsx:332-340`, `AccountScope.tsx:93-109`).
- Group and scope aggregates (`GroupTable.tsx:12-50`) show "No observed rows" for an empty bucket, "No source value" when no value is present, and present and missing counts.
- Korean copy says 공급자 in the toolbar and three notes (`Screen.tsx:151`, `:452`, `:798`, `GroupDetail.tsx:259`).

# Decision

Frontend-only wording and value-state change. The structure of the screen is unchanged; inside the detail, periods become labeled start and end lines, costs can carry a note line, and the shared components need B06-scoped styles. `/api/focus-commitments`, `contract.ts`, `api.ts`, grouping, paging, permissions and URLs are unchanged.

What changes and what deliberately does not:

| Area | Decision |
| --- | --- |
| Charge-row costs | Through the shared row-amount component: Missing without currency, dash or digit; Exact zero note; shared signed format |
| Charge-row service, lineage, provenance | A null service is headed "Service: Missing" so that the heading names what is missing; "Row ID", "Delivery ID", "Row ordinal in delivery" and "Sample provenance" by the shared labels, provenance through its glossary label; the row ID broken only at its midpoint |
| Charge-row periods | "Charge period" and "Billing period" start and end through the shared UTC instant presentation; a row has one period, so the word envelope is not used for rows |
| Group-level envelopes | Stay envelopes, with English "Source charge period envelope" and "Source billing period envelope" instead of "interval envelope", and both ends through the shared UTC instant presentation. Korean 원천 과금 기간 범위 and 원천 청구 기간 범위 already say period and are unchanged |
| Scope identity | 공급사, 결제 계정 ID, 하위 계정 ID through the shared field labels; Missing for a null subaccount |
| Commitment field states | Unchanged: column absent, blank, source null token, unmapped version and invalid type say more than Missing and stay B06's terms |
| Aggregates and metadata sets | Unchanged in meaning: "No observed rows" for an empty bucket; present and missing counts, now worded "Present / missing / 값 있음 / 누락" as in the glossary. An aggregate without present values, and a metadata set in which no row carried a value, read "No values present / 값 없음 / لا قيم موجودة" instead of "No source value / 원천 값 없음"; the helper is renamed so it cannot be reused for a row value. Aggregates are never labeled Exact zero |
| Amount format | B06's formatter delegates to the shared one, so rows and aggregates both read `USD -2.00`. B06's aggregates are sums of source costs, the case B08 already moved to the shared format; they are not computed review amounts as on B04 and B05 |
| Account scope ID | "Canonical account reference / 원천 계정 참조" becomes the shared "Account scope ID / 계정 범위 ID", because the value is derived by Hybrid Billing and is not a source value; the account-scope summary and its note are reworded accordingly. "Authorized tenant scope reference", "Group reference" and the group "Currency" have no shared term and stay |
| Style | B06-scoped rules for the shared classes (state, state note, value, instant, time zone, period and its first label) and a code class for the row ID with nowrap halves; left-to-right values stay inline inside their cells so that they sit on their label's edge in Arabic; localized nodes (Missing, the provenance label) are never nested in a left-to-right isolate |
| Wording | 공급사 in the toolbar and notes |

The prose that referred to "missing-field states" now names the field states, because Missing now also appears in the same list for plain nulls. Collection-slice wording (구간), "열 없음", "관측된 행 없음" and the error copy are B06's own terms and stay. The observed-at and coverage timestamps of the query stay as collected; they are not row values.

Stable item IDs: existing `B06:` items are preserved. Added: `B06:state:value-missing-zero`, registered in the screen and recorded here.

# Allowed files

`apps/console/src/modules/pricing/pages/commitment-benefits/*` except `api.ts`, `contract.ts` and `api.test.ts` (`values.ts` only for the formatter delegation); `apps/console/tools/commitment-benefits-qa.mjs`; `apps/console/README.md`; `docs/features/HB-159/*`; `docs/ui/terminology/product-glossary.md`. The shared components in `sources/collected-focus/` are used, not changed.

# Verification

- Console unit suite, type check, build and build verifier. No new logic; the shared components are covered by their unit test from HB-158.
- B06 browser suite on the retained sample, extended. Every row disclosure, the identity disclosure and the account scope are opened before assertions, absence checks use text content, and a non-zero count of value nodes is asserted, because most changed nodes sit inside disclosures.
  - At 1440 on the real AWS group: three charge rows with `USD 0.00` and Exact zero for both costs, the full row ID, ordinal, delivery ID, provenance label and UTC instants equal to the response; the group envelope as labeled start and end lines; identity labels in the detail, the group table and the account scope; no superseded label in the detail text; the present and missing wording.
  - Routed responses derived from the real response per request and kept consistent with B06's validator and with their own aggregates, in three languages at 1440: a null billed cost with the group, status bucket and currency total adjusted; a negative effective cost shown in the row and in the group aggregate in the shared format; a null service; a null subaccount on the rows, the group and the account; a `Z` instant; then every row null on the effective basis with an aggregate reading "No values present". Missing nodes have no left-to-right isolate ancestor, and in Arabic the row ID ends on its label's edge.
  - Layout with the detail and all disclosures open in EN, KO and AR at 1440, 1280, 1279, 1024, 761, 760, 641, 640 and 390: no page overflow and no child of a detail, identity or account box wider than its box. The degraded response at 390 in KO and AR.
  - The denial case also asserts that no source value, row cost or identity node remains. Existing group, detail, paging, account, stale and failure cases and the 24 view, language and width variants still run.
- S04 browser suite as regression; isolated `tests/focus_overview` and the commitments backend tests unchanged; retained file hashes against the HB-150 manifest.
- Independent review of the actual change; ui-designer render comparison including the utilization view, where Exact zero rows sit under the note that no unused rows were observed; technical-writing review of the changed strings. AI review is not native-language, practitioner or owner acceptance.

# Design review resolution

The independent design review of the first draft returned FAIL without blockers. Resolved in this revision: B06-scoped styles and the real extent of the visual change stated (H1); disclosures opened before assertions (H2); the four uses of the old no-value helper decided one by one (M1); localized nodes kept out of left-to-right isolates (M2) and left-to-right values kept inline (M3); the glossary entry and the explanatory note updated with the period wording (M4); Arabic envelope labels given the envelope word (M5); the account scope ID label and its summary and note (M6); one amount format for rows and aggregates, with the reason (M7); B06's own breakpoints added to the widths (M8); non-table containers named in the layout check (M9); routed responses kept valid and self-consistent (M10); the smaller wording decisions recorded.

# Change review resolution

The independent review of the actual change returned PASS without high or blocking findings, with this verification record as a merge precondition; the ui-designer comparison returned conditional PASS without blocking findings. Applied from the review: the routed responses now derive every coefficient and count by difference from the fetched response instead of relying on the sample's zero costs; the layout check also measures period lines and value boxes; an unused import removed; one wording for "does not establish legal ownership"; the field-states sentence quotes the on-screen state labels and reads as examples; assertions for the partial count, an empty metadata set and superseded Korean labels. Applied from the designer: the evidence and identity lists use an auto-filling grid so that the stacked full-width panel no longer shows sixteen full-width lines; the commitment ID and row ID span the full width; periods and delivery lineage start their own groups; charge records are separated by a section rule; the label part of "Service: Missing" is muted; period labels share a minimum width; counts do not wrap; account tiles no longer break before a colon. Not applied: lowering the weight of the Exact zero note, which is the shared presentation on five screens.

# Deferred

Commitment lifecycle, utilization, allocation and sales-policy producers; U01 and B07 terminology in their own tickets; HB-152.
