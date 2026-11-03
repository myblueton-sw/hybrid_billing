---
type: ScreenDesign
title: U01 collected FOCUS overview composition
description: Defines the commercial viewport candidate with collected cost, account mapping and explicit operational integration gaps.
status: draft
---

# Decision and ownership

The owner rejected the first PR77 candidate for typography, layout and usability, then required collected FOCUS data and account-grounded organization samples. Root owns PM, implementation and QA. Claude CLI ui-designer supplies composition and render critique; separate financial and architecture reviewers inspect the actual changes. No owner FIX exists. HB-85 retains the pending HB-76 shell dependency; IA, screen ID and canonical brand assets remain unchanged.

The [collected integration design](collected-focus.md) replaces the previous generated U01 cost/closing body. The prior implementation and its evidence remain identifiable at Git revision `0b86011b7290c8f4a85a8f3401ff0ea687601b86`. No second fixture-backed overview is selected when collection fails.

# Planning boundary

The [U01 specification](../../ui/screens/U01/spec.md), v1.16 UI design and HB-01/23/30/31/39 require financial meaning, scope, closing denominators, blockers and scoped next actions. Collected FOCUS supplies cost and observed billing/subaccount scope. It does not supply authoritative customer sales, organizational allocation execution, closing, approval, ERP or receivables. Those lanes and the existing operational tabs remain explicitly unconnected; the [finance/billing role-fitness findings](finance-billing-review.md) remain open. This increment does not redefine the planning contract as a cost-only product.

# External evidence and design lead

Root consulted official [Carbon typography](https://www.carbondesignsystem.com/building-blocks/foundations/typography/type-sets), [Cloudscape dashboards](https://cloudscape.design/patterns/general/service-dashboard/) and [CloudZero navigation](https://docs.cloudzero.com/docs/navigating-cloudzero) on 2026-11-03. Adopt stable type roles, trends followed by scoped investigation and normal service navigation. Cloudscape was read again for this collected-data revision. Retain predefined reading priority; do not import configurable widgets, third-party assets, their palette, rounded controls or decorative colored edges.

Claude CLI ui-designer actually read earlier rendered screens, the integration design and the installed official frontend-design skill before proposing this composition. Root adopted the quiet amount strip, trend/provider band and full-width account/detail tables. The proposed tax-exclusion claim was rejected: this projection establishes no tax policy. Account/detail pages remain 25 rows, matching the reviewed API. Tables use bounded keyboard-accessible horizontal scrolling instead of a second mobile renderer. These are explicit adjustments, not claims that the CLI tested them.

# Active composition

Use the full viewport with the existing 256px desktop rail and responsive shell. Keep title, exact source revision, review status and confirmation visible. Scope controls retain natural widths and use actual collected months/providers; the explicit default is all collected periods because providers cover different months. BilledCost and EffectiveCost use a radio group. Refresh starts a new protected query; reset clears narrowing filters.

A compact provenance line distinguishes public versus synthetic rows, FOCUS version, query-only authority and no FX conversion. Its disclosure explains tenant-timezone charge-start attribution without proration, synthetic internal membership versus observed account IDs, full-authorized-source coverage and query observation time. Source finalization never means financial closing.

The neutral, unfilled amount strip shows each currency independently at exact precision, selected/authorized row counts, observed account combinations and unconnected operational lanes. The desktop grid gives cost cells more width than metadata cells. Amounts wrap rather than shrink or round.

The signed monthly trend and provider table share a 7:5 desktop band, stacking on compact layouts. Calendar gaps remain unknown, recorded zero remains zero, partial-known bars include text, and all-missing values remain unknown. Exact monthly amounts have a table equivalent. No signed percentage shares or cross-currency totals are invented.

Organization/account mapping follows across the full width: internal sample tenant, provider, BillingAccountId, nullable SubAccountId, BilledCost/EffectiveCost and row count. Search narrows this bounded list only. Selecting a complete account identity narrows financial/source rows; labels never form identity. No missing subaccount, department, legal entity or native ancestry is fabricated. Source details expose 25 normalized rows and allowlisted collection evidence, not raw source JSON or authority graphs.

# Type and shared impact

Keep local IBM Plex Sans, Sans KR and Sans Arabic at real 400/600. U01 uses 24px page titles, 20px section headings, 16px body/table/control text, 14px metadata and 28px key amounts. Financial digits remain exact and LTR; Arabic money uses consistent Latin numeric notation. Use neutral separators, square controls and visible keyboard focus. No centered maximum-width page, decorative colored border, radius, gradient or shadow is introduced.

The earlier shared KR/Arabic 600 font availability remains part of this branch; other screens' visual acceptance is not implied. This revision changes U01 content and source semantics. Review context includes source/selection snapshot, offsets, resolved chart currency and operational tab; old acceptance must not transfer.

# Stable implementation items

| Item | Current behavior | Boundary |
| --- | --- | --- |
| U01:scope:collected-focus | Actual collected query, source revisions and no fallback | Public/synthetic samples only |
| U01:scope:provenance | Version, evidence class, membership and source coverage | No production conformance or financial authority |
| U01:summary:amount-meanings | Exact BilledCost/EffectiveCost and separate currencies | Sales/allocation/AR/closing unconnected |
| overview:field:cost-basis | Explicit cost-basis radio group | Snapshot-bound selection |
| overview:field:organization-selection | Complete account identity narrows rows | Observed account scopes, not a native organization tree |
| overview:field:organization-search | Bounded mapping search | Does not alter financial totals |
| overview:data:monthly-cost | Signed calendar series and exact table | Gaps, nulls and zero remain distinct |
| overview:data:source-breakdown | Complete filtered provider totals | Never summed from browser pages |
| overview:data:organization-breakdown | Observed account combinations | No invented missing ancestry |
| overview:section:detail | Bounded normalized rows and evidence | Allowlisted fields only |
| overview:action:page | Snapshot-pinned account/detail pages | Changed source/filter rejects continuation |
| overview:state:unconnected | Explicit unavailable operational lanes | No generated closing checklist |

The previous generated example-profile, work-attention and closing-matrix items are not implemented in this collected candidate; they are not reassigned to unrelated features. Their planning gaps remain deferred rather than silently accepted.

# Acceptance and verification

See [collected-data verification](focus-verification.md) for current evidence and [earlier verification](verification.md) for the separate prior revision. Neither a build nor a specialist source review fixes the screen. Native-language, assistive-technology, production identity, complete U01 business fitness and owner acceptance remain separate.
