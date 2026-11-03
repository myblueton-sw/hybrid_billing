---
type: IntegrationDesign
title: Collected FOCUS cost and usage source screen
description: Defines the protected collected-row drill-through from U01 to S04 with exact quantities, costs and source lineage.
status: draft
---

# Scope and implementation plan

Goal: Implement S04 consolidated cost/usage using the same collected source/identity as U01. Root sole writer; independent reviewers read only. Ticket HB-143 depends explicitly on HB-85/PR77 and HB-93 design. No merge/deployment.

Owned change: extend modules/focus_import/overview.py allowlisted row DTO, isolated tests/focus_overview; move reusable focus-api.ts with updated imports into modules/sources/collected-focus/api.ts; move exact decimal formatter into sources/collected-focus/values.ts (overview chartSeries/signedScale remain operations); create sources/pages/collected-source-data React screen/CSS/QA; scoped App.tsx/default S04 selection and U01 source-detail link; preserve legacy service=local inspector explicitly. Root does not edit collection auth/locks or reseed stores.

Fields: existing raw row has list_cost,contracted_cost,consumed_quantity,pricing_quantity,list_unit_price,contracted_unit_price,consumed_unit,pricing_unit,billing_start/end. Emit nullable exact coefficient strings using existing coefficient(), not number. Costs currency-qualified; quantities per row with native unit; no quantity sum, no derived rate or denominator, no pricing formula. Existing source_json never emitted. Increment projection identity from1to2 for the extended governed row content; schema remains1 for additive compatibility. Additional fields additive under schema1 to preserve existing U01 client; shared validator requires them for new S04. Row IDs/source snapshot/selection stay current. Existing bounded25row details/search/account list reused; provider/period/account/basis narrow actual authorized facts. New financial query filters service/currency deliberately deferred to avoid pretending filtering a page equals whole-selection filtering.

Source service supports normalized rows and existing mapping/account. S04 tabs normalized/sourcefields/usage/history preserve canonical keys. Sourcefields means verified mapped identifiers/billing-charge periods and pinned source lineage only; arbitrary original rows not exposed. History tab explicit unavailable, no inferred transformations. Usage shows consumed/pricing quantity and native unit plus sourceunitprices qualified by currency and pricing unit, not blended sums. Missing unit stays unavailable even when quantity is present. Canonical row detail focus/return, paged25 controls. Search account mappings may select account; arbitrary q is list-only and labelled. No generated fallback including legacy mode/scenario.

Scope handoff: U01 dedicated link to /source-data carrying focus-period/provider/account/basis/snapshot plus lang. Reset after/account_after/q for a fresh detail page, preserve source snapshot only when identical selection; if q cleared, snapshot must clear or keepq to avoid stale identity. Chosen preserve complete selection includingq with offsets reset, link pins same snapshot. S04 header back tooverview carries financialselection and snapshot, no row/detail state. Routing ID query values aren't authority.

Validation: independent design/code financialsecurity review; isolated monetary null/zero/negative/18precision and allowlist tests; same actual retained1121rows/78accounts; provider/period/account and snapshot scopes; costs equaloverview; usageunits/null preserved; async old-response/error clearing; boundedpages; detailfocusreturn; ownerreviewpending; six ENKOAR desktop/mobile renders, neutral computedborders and no page overflow; installed ClaudeCLI design+rendercritique with official references. Do not run tests/focus_import reset conftest. No ownerFIX, productionauthority, financialledger completion or crossOS certification.

# Design review

Separate financial design review: conditional PASS. Conditions include exact nullable quantities/prices, consumed versus pricing unit separation, currency/pricing-unit qualification for prices, projection revision2 and exact cross-screen selection including q. Architecture design review: conditional PASS for sources-owned shared DTO and bounded protected allowlist; no authentication or collection-lock changes. Actual-change review is required after implementation.

# Composition evidence

Root read the official [Cloudscape table view](https://cloudscape.design/patterns/resource-management/view/table-view/) on2026-11-04. Adopt full-page data priority, bounded pagination, scoped filters and explicit empty/error states; reject imported styling, optional sorting, preferences and bulk selection without server contracts. The installed Claude CLI ui-designer read the existing actual U01 renders and official frontend-design skill, then proposed scope/provenance, count/paging, normalized table and focused bottom detail. Root preserves natural-width controls and neutral separators/local IBM Plex16body/14metadata/20section headings. Mobile uses one keyboard-scrollable table, not a second generated card model.

The proposal's independent BillingAccount/SubAccount selects and row search are not implemented as misleading page-local filters. This scope uses a bounded account-mapping picker keyed by the complete observed identity and explicitly labelled list-only search. The proposal's account names, original source fields and row transformation/AI histories remain unavailable unless exposed by the current allowlist. Root displays source unit prices per pricing unit without asserting denominators/formulas. All 4 existing S04 tab keys remain.
