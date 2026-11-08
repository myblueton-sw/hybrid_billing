---
type: ScreenDesign
title: Commitment costs and benefits evidence workspace
description: Designs B06 against actual collected commitment identifiers with separate cost bases, source classifications and blocked financial conclusions.
status: draft
---

# Scope, owners and verified discovery

HB-106 preserves B06 `/commitments` and commitments/utilization/benefits/reconciliation. It stacks on unmerged HB-105 PR82 at `3dca894`, not fresh main; remote main exists separately and was read back. Root owns PM, frontend API/model/integration/tests/evidence. An isolated backend writer owns the new read-only projection/HTTP route and unit tests, and an isolated UI writer owns only the new B06 leaf. Independent finance/security/TW reviews cover design and actual changes; Claude CLI ui-designer leads composition and actual-render comparison. B03 FIX and B04/B05 modules remain unchanged; continuing work is not their owner acceptance.

Read-only discovery of hash-verified public/synthetic artifacts and documented north assignments found 1121 rows, 4 commitment-ID rows, 2 scoped groups, all AWS Savings Plan / Spend / Used / Usage. No unused classification or commitment name is present. This is artifact evidence until a fresh protected database read confirms it. Literal `NULL` is a pinned 1.0 null token, not an identifier; counting it naively produces 993 false commitment rows. Current overview and local-release detail APIs do not project these fields, so B06 requires a dedicated protected read-only projection. No ingestion/schema/source rewrite or original-file mutation.

# Read-only API and identity

Add `/api/focus-commitments` behind the existing trusted-local origin/header/actor/semaphore and collected-query authorization/read transaction. Do not alter `/api/focus-overview` response or permission behavior. Accept its provider/period/account/q plus after/account_after/snapshot and new group/row_after; bound query syntax, all offsets and duplicate keys. `q` searches account lists only. Explicit canonical account selection narrows rows. Page 25 groups, 25 accounts, 25 detail rows. Any selected group or nonzero offset requires the current derived commitments snapshot; mismatched expected snapshot rejects. A selected group must belong to the returned current group page. Group/detail/account page changes clear incompatible selection in the client.

Projection profile `commitment-source-evidence-sums-v1`: all aggregates cover the complete authenticated provider/account/month selection before paging. Group key is company/tenant/provider/canonical account (including subaccount)/currency/nonblank source commitment ID. Keep identical provider IDs across tenants/accounts/currencies separate. Null/header-absent/unmapped IDs remain unidentified source evidence, never one fabricated commitment. VMware/on-premises references are not certified cloud commitments. The group digest is used in URLs, never raw source IDs or amounts.

Read only allowlisted raw fields from authorized source_json: CommitmentDiscountId/Name/Type/Category/Status and ChargeCategory/Class/Frequency. Per-row version comes from verified preflight delivery metadata, not a guessed global version. Use the existing pinned version registry's null tokens. Preserve absent/source-null/blank/unmapped-version/invalid-type distinctions. Map Used/Unused only for the explicitly reviewed FOCUS 1.0 status mapping; other source lexical values remain unknown and visible. Never upgrade stored versions. Cap distinct metadata arrays at 16 values and retain exact distinct count/truncation/conflict flags. Each raw field value is bounded at 4096 characters; invalid/missing data remains explicit or rejects safely.

Monetary input coefficients remain exact signed scale 18, abs<10^38; projection aggregation has a defensive abs<10^50 bound but does not expand the collected reader admission envelope, which already rejects additive totals at 10^38. The stricter upstream check remains effective; overflow rejects. Separate BilledCost and EffectiveCost, including present/missing counts. No rows or all-missing is unknown/null; signed cancellation is known zero. Status buckets Used/Unused/unknown partition identified group rows once, with per-basis costs. Absence of Unused rows is not a measured zero-unused burden. Source-row sums do not prove provider fee/negation/amortization correctness. Never compute savings,unused burden or customer discount from Billed−Effective, and never demand monthly equality.

The response includes source/selection snapshots, company, versions/registry digest, charge-start period basis, observed/coverage/provenance metadata, bounded account search/options; scope/identified/unidentified row counts and separate per-currency amounts; paged groups with metadata/status/category evidence; optional selected current-page group and paged canonical charge details with source-field states and full charge/billing intervals. No arbitrary raw JSON or other fields are exposed. Server snapshots bind profile/mapping/source/selection; cursor references cannot authorize another scope. Detailed response types are maintained in the B06 frontend API, and backend/client fixtures must agree before UI wiring.

# Four actual views and authority boundary

Commitments: selected source filters/account picker, observed scoped commitment count, separate Billed/Effective full-selection amounts, bounded real group table and selected evidence detail. Do not label generic source rows as commitment count. Default source AWS (actual commitment evidence), not fabricated VMware commitments.

Utilization: selected group's source Used/Unused/unknown classification rows and costs, charge periods and evidence fields. No VM utilization percentage, capacity denominator, savings forecast, assumed unused amount or lifecycle active/expired claim.

Benefits: actual provider metadata and source rows first; internal allocation and customer sales-discount/pass-through authority remain visibly unconnected. Unused-burden bearer, legal ownership and sales-contract prepayment recovery are unknown. No discount reapplication, purchase, optimization, repricing or policy-activation button.

Reconciliation: inspect coverage/version/field-state/conflicting metadata and source charge categories. Show incomplete evidence and unconnected authoritative counterpart instead of financial reconciliation complete. Whole commitment-term conservation remains unverified in a monthly/filter scope. Evidence inspection is the actual action; no fake approval or job.

This read-only increment needs no invented editable financial policy or browser draft. Selection/filter/history/retry and focus return are the real interactions. Source changes/failure remove selected details; stale responses cannot restore them. The common per-screen review control remains independent and never confirms on the owner's behalf. B06 FIX remains pending.

# Commercial design and validation

Use full viewport/canonical IA/logo, semantic token variables/local Plex 16px body/table/controls, 14px metadata and 20px section headings. Neutral square separators, no decorative colored borders/shadows. KO/EN/AR with IDs/currency/date exact values LTR. Desktop table/detail workspace; mobile cards with auto cell heights plus separate labelled full-column disclosure. Long source IDs and metadata wrap. Data absent, null, zero, unmapped, error/revoked and no matching accounts remain distinct. Stable IDs cover source scope, grouped costs, classifications, benefit authority, evidence review and async states. Current tabs use TW candidates 약정 명세 / 사용·미사용 근거 / 혜택 흐름 / 대사 검토.

Financial design review conditionally passed with the identity/version/null/complete-aggregate and no-inferred-savings conditions incorporated here. Independent security/public-contract design review conditionally passed with authorization-transaction delivery, route-specific limits, tenant+batch version association, snapshot/group membership and upstream-bound requirements incorporated. Unit oracles cover group collisions, pinned null tokens, unknown/conflicting metadata, status partition, signed/partial/all-null sums and overflow, page-invariant totals, and no savings from 100/80. Pure unit/isolated-reader tests only; never run retained-store reset suites. Actual protected reads confirm artifact observations without writes. Browser verification covers real rows/groups/filter/account/detail/tab/paging/retry/denied/stale cases and 24 locale/layout/view variants plus inherited-screen regression. Other OS/browser engines, native-language, human finance, provider-event support and lifetime reconciliation remain unverified.

The producer/client response contract is maintained at `apps/console/src/modules/pricing/pages/commitment-benefits/contract.ts`. It is not a claim that every optional source field or provider event is supported.

# External evidence and stable review items

[FOCUS 1.0 Discount Handling](https://focus.finops.org/docs/specification/v1-0/attributes/discount-handling/) separates purchase/usage/unused/credit records and describes commitment-duration conservation. Adopt source-field semantics only; reject monthly-equality or provider-conformance claims. [FOCUS 1.0 Commitment Discount Status](https://focus.finops.org/docs/specification/v1-0/columns/cost-and-usage/commitment-discount-status/) establishes charge-level Used/Unused meaning, not VM utilization or commitment lifecycle. The pinned 1.0 version remains unchanged. Carbon data-table/disclosure patterns inform bounded selection and progressive evidence; no third-party assets/dependency or decorative styles are imported. The current Carbon page fetch timed out; prior verified guidance and the official search excerpt, not a successful new page read, supply that structural reference.

| ID | Purpose/location | Required behavior and verification |
| --- | --- | --- |
| B06:scope:authorized-source | Source filters and account picker | Actual auth/query/snapshot; account search is not a row filter. |
| B06:data:commitment-groups | Group table and global cost summary | Real scoped IDs, complete per-basis aggregates, bounded pages. |
| B06:data:source-status | Used/Unused/unknown evidence | Partition once; zero observed rows is not zero unused burden. |
| B06:data:benefit-boundary | Provider metadata/internal allocation/customer discount | Only actual source stage populated; no invented authority. |
| B06:data:reconciliation-evidence | Field/coverage/version/category review | Inspect evidence without false financial completion. |
| B06:action:inspect-detail | Selected group's source records and metadata | Fresh bounded details, close/focus return, canonical identity. |
| B06:state:async-unavailable | Loading/empty/error/denied/retry | Clear stale details and reject late responses. |
