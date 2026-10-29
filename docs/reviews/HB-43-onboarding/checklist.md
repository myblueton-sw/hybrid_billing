---
type: AcceptanceChecklist
title: Owner requirements and delivery acceptance checklist
description: Tracks owner-requested billing capabilities independently from their appearance in the design book.
status: draft
---

# Status rules

Tracking: [HB-43](https://linear.app/hybrid-billing/issue/HB-43). This consolidated index carries the owner's earlier and current requests forward; detailed requirements and contracts remain authoritative through links below. Root owns integration; individual implementation owners and tickets must be assigned before execution. A visible mockup, design review or merged document never completes runtime acceptance. Every unchecked item below requires implementation evidence and appropriate independent verification. Existing runtime work is not assumed absent, but has not been reverified by this design ticket. These are acceptance workstreams, not invented completed implementation tickets.

Design status: **Shown** means a representative screen exists; **Specified** means required behavior is documented but no complete interactive demonstration exists; **Partial** means only part of the scope is represented. Runtime status for every item is **OPEN / NOT VERIFIED IN HB-43**. Check an item only after linking implementation revision, test evidence, independent review and QA result. Unknowns and owner decisions remain visible.

# Billing operations and evidence

| Acceptance | Design status / review location |
| --- | --- |
| [ ] BL-01 Separate MSP staff and customer experiences, with current tenant/resource/field enforcement in APIs, caches, searches and exports | Shown: access, operations, isolated customer chapter |
| [ ] BL-02 Support 1,000–5,000 customer operations with server-side search/paging, useful defaults, exception queues and company/month/status summaries | Shown: monthly operations; scale runtime unverified |
| [ ] BL-03 Retain every monthly billing target including unissued customers, reasons, owner and next action; reconcile counts | Shown: operations and monthly history |
| [ ] BL-04 Customer profile contains legal identity, business registration jurisdiction/type/number, tax ID, address, contact and renewal/notice dates | Shown: customer detail p21 |
| [ ] BL-05 Named issuer/reviewer can be required or optional by policy; actual logged-in actor recorded if no assignee; permissions/SoD cannot be bypassed | Shown: assignment and access roles |
| [ ] BL-06 Preserve monthly issued lists by MSP/customer and fixed historical document revisions, corrections, actor and timestamps | Shown: history and issued detail |
| [ ] BL-07 Collect and reconcile provider source data across AWS, Azure, GCP, OCI and other supported CSP/VMware profiles; distinguish support candidates from verified adapters | Partial: source/setup chapters; live coverage unverified |
| [ ] BL-08 Preserve original supplier invoices/PDFs and lineage; AI-assisted parsing lists extracted items, differences, confidence and review corrections | Shown: PDF intake/extraction/difference/audit |
| [ ] BL-09 Exact per-currency financial validation, missing versus zero, duplicate/revision handling and source/invoice reconciliation gate issuance | Shown/Specified: reconciliation and readiness |
| [ ] BL-10 AI assistant proposes mappings/corrections and supports repeated review without silently approving prices or exposing credentials/customer evidence | Partial: AI proposals and PDF review; provider choice unresolved |

# Contracts, payment and ERP

| Acceptance | Design status / review location |
| --- | --- |
| [ ] CT-01 Master agreement none/present/unverified, parties/signature/version/dates, linked child agreements, named inheritance and approved overrides | Shown: customer detail; cycle/cross-scope/conflict cases Specified |
| [ ] CT-02 Composable fixed/usage/hybrid pricing, monthly/quarterly/semiannual/custom cadence, prepaid/postpaid timing | Shown: customer detail; calendar/period implementation required |
| [ ] CT-03 Advance/interim/balance schedules with trigger, amount/percentage, due dates, total/currency/rounding validation | Shown/Specified: customer detail and requirements |
| [ ] CT-04 Confirmed deposit balance and authorized deduction allocation, shortage policy, reversals/idempotency; never treat deposit as discount | Shown/Specified: customer detail and requirements |
| [ ] CT-05 Bank transfer/card/electronic promissory note/other allowed methods, default and settlement evidence; no payment execution merely from selection | Shown/Specified: customer detail |
| [ ] CT-06 Due dates, overdue, unknown receipt state and correction history use approved terms and authoritative settlement evidence | Specified: payment/recognition contract; complete runtime workflow pending |
| [ ] CT-07 ERP connection identifies MSP/internal versus customer books and purpose; standard adapter contract with Odoo candidate; no assumed live compatibility | Shown: ERP mapping and onboarding |
| [ ] CT-08 Monthly recognition versus quarterly/customer billing cadence remains independent and reconciled; unknown posting reads back before retry | Specified: payment/recognition contract |
| [ ] CT-09 Chargeback/showback modes are explicit where applicable; allocation views do not imply external invoicing/payment | Shown: allocation and customer views; policy per installation |

# Automatic pricing and customer explanation

| Acceptance | Design status / review location |
| --- | --- |
| [ ] PR-01 Apply broad service/component discounts and type/SKU-specific rules, e.g. all DB compute versus DB-instance-type rates | Shown: rulebook/editor/mapping |
| [ ] PR-02 Combine common terms, standing customer exceptions, time-limited and one-off credits with explicit eligibility and precedence | Shown: recurring/one-off rules and bulk preview |
| [ ] PR-03 Provider EDP/commitment/DTO/additional benefits carry native evidence, allocation and cost-versus-selling treatment; OCI uses common contract navigation | Shown/Specified: provider details and pricing |
| [ ] PR-04 CFRC remains inactive until definition, contractual evidence and eligibility are confirmed; sales discounts exclude designated product discounts | Shown: exclusions; CFRC definition OPEN |
| [ ] PR-05 Bulk automatic matching/calculation shows changed, unchanged, missing, conflicting and blocked outcomes; idempotent retries and approved activation | Shown: portfolio, bulk, exceptions, activation and job |
| [ ] PR-06 Customer product overview directly shows contractual percentage and applied outcome without reverse calculation or asking sales | Shown: customer pricing explanation |
| [ ] PR-07 Search authorized collected product/instance/account/SKU/period data with stable IDs; show quantity, unit basis, discount amount, net rate and charge | Shown: instance search; server pagination/export enforcement Specified |
| [ ] PR-08 Explain no discount/excluded/nonstacking/pending/blocked cases; retain unallocated/shared charges and unknown coverage | Shown/Specified: pricing evidence/customer detail |
| [ ] PR-09 CSP commitment → supplier additional → MSP special discount bridge shows every basis, rate, amount, sequence and resulting price; avoid double counting source-net benefits | Shown: AI cost waterfall and customer invoice detail |
| [ ] PR-10 Distinguish contractual rate from effective discount; zero/incomparable bases N/A, fixed/tier/credit formulas explicit, tax/FX not disguised as discount | Shown/Specified: pricing and requirements |
| [ ] PR-11 Distinguish list/selling basis from actual supplier acquisition cost; cost needs independent frozen evidence and allocated scope | Shown: AI cost waterfall |
| [ ] PR-12 Customer open-book cost disclosure requires explicit recipient/field/source/version approval; defaults restricted; source contracts/margins not automatically exposed | Shown/Specified: AI invoice example and access requirements |
| [ ] PR-13 AI GPU-hour/token/request/storage charges retain appropriate units and SKU attribution; do not sum incompatible quantities or invent instances | Shown/Specified: AI cost case; non-GPU runtime adapters pending |
| [ ] PR-14 Provide invoice PDF plus detailed CSV/XLSX with each resource/charge and discount; exports reconcile to the same fixed invoice and customer scope | Shown: AI invoice detail; download generation NOT IMPLEMENTED here |
| [ ] PR-15 Search/paging never reprices pools or thresholds; filtered subtotal differs from full-period total; effective segments and historical versions remain fixed | Shown/Specified: customer search and pricing requirements |

# Cost exploration and Marketplace additions

These extend existing items where noted; [duplication review and official evidence](feedback-review.md) explain the mapping.

| Acceptance | Design status / review location |
| --- | --- |
| [ ] EX-01 Project/service/SKU pivot supports day/month basis, project ID, service, SKU, usage/discount type and region; fixed currency/version and unknown buckets | Shown: cost-pivot; extends BL-09/PR-07 |
| [ ] EX-02 Raw lookup/export preserves source lineage and allowed fields, complete filtered scope, row counts, decimal precision and totals; no silent truncation or unsafe spreadsheet formulas | Shown: raw-explorer; extends PR-14 |
| [ ] EX-03 Diagnose raw/console/Excel differences by matched scope and stored/calculation/display rounding stages, not blanket tolerance | Shown: cost-differences; extends BL-09 |
| [ ] EX-04 Diagnose AI version prices by exact SKU/model/version/region/unit/effective date; unknown inputs remain blocked | Shown: cost-differences; extends PR-13 |
| [ ] EX-05 Reconcile CSP Support using covered accounts, fee basis/window/tier/minimum, credits and finalization evidence; preserve unexplained differences | Shown: cost-differences; extends BL-08/09 |
| [ ] EX-06 Marketplace register distinguishes ISV license/usage, infrastructure and MSP fees; links actual publisher/product/offer/order and procurement/customer contracts | Shown: marketplace-register; extends PR-03 |
| [ ] EX-07 Track actual purchase, eligible contribution and confirmed commitment consumption/reversal separately with provider/contract/channel/date/cap evidence | Shown: marketplace-commitment; extends PR-03 |
| [ ] EX-08 Optional ISV list-price purchase planning shows assumptions, unknown eligibility, before/after utilization and remaining commitment; no actual posting or automatic purchase | Shown: marketplace-commitment; optional expansion, not core settlement |
| [ ] EX-09 PDF/AI contract extraction retains page/hash/revision and human confirmation; no inference of signature, amended terms or provider-confirmed consumption | Shown/Specified: marketplace-register; extends BL-08/10 |
| [ ] EX-10 Optional third-party API layer defines inbound versus outbound direction, approved schema/auth/scope/limits/idempotency/coverage separately | Specified: setup-connectors; optional peripheral scope |

Group allocation remains CT-09 rather than creating a duplicate workstream: the existing allocation screen now explicitly shows rule version, weights, residual conservation, unallocated cost and separate legal buyers.

# Entry, onboarding and lifecycle

| Acceptance | Design status / review location |
| --- | --- |
| [ ] AC-01 Separate first installation administrator, invited MSP member, invited customer and existing-user paths | Shown: p3–6 |
| [ ] AC-02 Atomic bounded bootstrap/invitations with expiry, revoke, recipient/scope, reissue and concurrency handling | Shown/Specified: p4–5 |
| [ ] AC-03 Configured authentication, step-up, recovery, last-admin and session/trust revocation; no implicit outage bypass | Shown: p6–10; method/product/timeouts OPEN |
| [ ] AC-04 Delegated role ceilings and financial SoD independent of named assignment policy | Shown: p9; runtime security verification required |
| [ ] ON-01 Restartable OP05 ordering, assigned handoff, revision/expiry-bound diagnostics and selective downstream invalidation | Shown: p11–18 |
| [ ] ON-02 Separate temporary setup, export writer and collector reader custody; verify actual scoped read and setup privilege removal | Shown: p14–15 |
| [ ] ON-03 Distinguish credential storage, connectivity, read success, arrival, parsing, coverage, reconciliation, acceptance and billability | Shown: p15–18 |
| [ ] ON-04 Scope/schedule/backfill, duplicate inputs, unknown outcome read-back, quarantine and nonzero missing-data handling | Shown/Specified: p16–17 |
| [ ] ON-05 Attribution and historical customer transfer have separate authorization and immutable history handling | Shown/Specified: p13 and source relationship contract |
| [ ] LC-01 Preserve noncommercial versus separately contracted commercial licensing, official key issuance, 14-day trial notice and contact path | Shown: license chapter; actual entitlement enforcement separately verified |
| [ ] LC-02 Sponsorship must not imply a commercial license or bypass authorization; preserve adopted license policy | Specified: existing license policy; no policy change in HB-43 |
| [ ] UX-01 Full bilingual book, neutral visual treatment, all prior coverage, understandable states and separate customer review | Shown: v10; static artifact QA separate from runtime accessibility |

# Evidence and update rule

Use [requirements](requirements.md), [verification](verification.md), [prior pricing requirements](../HB-42-auto-pricing/requirements.md), [payment/recognition contract](../../contracts/billing/payment-and-recognition.md), [identity boundaries](../../architecture/security-boundaries/identity-session-permissions.md) and [onboarding obligations](../../contracts/operations/source-policy-onboarding-and-verification-obligations.md). Licensing remains governed by [product policy](../../licensing/product-policy.md), not this checklist. Add a ticket/revision and independent evidence beside an item before changing its runtime status. All subsequent owner feedback must update both the relevant design and this checklist.
