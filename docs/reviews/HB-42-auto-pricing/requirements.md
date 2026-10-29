---
type: Requirements
title: Automatic contract matching and bulk customer pricing
description: Defines the owner-requested automatic pricing workflow, financial controls and twelve connected review screens.
status: draft
---

# Ownership and boundary

[HB-42](https://linear.app/hybrid-billing/issue/HB-42/specify-automatic-contract-pricing-and-integrate-full-v9-mockup), 2026-10-29, machine mac_mini. Root is sole writer/integrator; Seung Woo Park is accountable assignee. Independent read-only financial and workflow reviewers are csp_financial_research and msp_workflow_research. UI/UX, documentation and PDF workflows apply. Owned changes are this directory and the root README entry. Prior owner authorization permits scoped publication and main integration; no live financial or customer actions.

The owner requires approved contract types to automatically match customer usage, calculate unit prices and discounts, and display batch outcomes and individual evidence. This ticket records that product requirement and supplies a complete EN/KO v9 design review. It does not implement or certify the pricing engine. [HB-41 research](../HB-41-msp-pricing/research.md) is the evidence baseline; named MSP anecdotes are not universal vendor policies. CFRC remains unresolved and inactive until its meaning and terms are confirmed.

# Required product behavior and acceptance

| ID | Requirement | Acceptance / linked screen |
| --- | --- | --- |
| AUTO-01 | Automatically match accepted usage to an approved, effective customer contract and native billable identities. | Service/component, instance family/type, engine/license, region, deployment, purchase model, SKU/meter and account scope are available. Missing metadata is unknown, not wildcard. `auto-mapping`. |
| AUTO-02 | Support service-wide, type-specific, fixed-unit, tier, commitment/allowance, DTO, general sales, fee and credit rules. | Basis, formula, scope and duration are separate fields; public-list, already-net supplier basis and customer contractual price are distinguished. Not all providers share identical fields or benefits. `auto-rulebook`, `auto-editor`. |
| AUTO-03 | Apply explicit precedence and discount exclusions. | Product-discounted charge portions are excluded from general sales concessions under the illustrated contract. Record exclusion granularity, benefit identity and group. More specific does not automatically mean higher priority. Equal-priority unresolved overlaps block affected calculations. `auto-exclusions`. |
| AUTO-04 | Show bulk matching/calculation outcomes for 1,000–5,000 customers without forcing per-customer manual setup. | Search and primary status, versioned templates and explicit exceptions, staged rate-card import, changed/unchanged/blocked results, currency-separated totals. Selecting/filtering a page does not redefine the authorized target snapshot or tier/credit pools. `auto-portfolio`, `auto-bulk`. |
| AUTO-05 | Preserve precise calculation evidence. | Pin source, price basis, rule versions, quantity/unit, currency, rounding stage/mode and period. Show selected/replaced rules, intermediate values and excluded reasons. Missing historical prices or benefit provenance block the affected portion. `auto-trace`. |
| AUTO-06 | Trigger calculation on accepted source revisions and approved effective rule changes. | Jobs are idempotent for fixed inputs; changed inputs create a new revision. One-off grants use stable identity, reservation/consumption/reversal history and fixed pool membership. Reruns do not consume twice. `auto-job`. |
| AUTO-07 | Separate preview readiness, contract activation, calculation completion, financial approval and issuance. | A ready count grants no approval. Explicit target snapshot and policy required for partial activation; blocked targets cannot inherit a new version silently. Stale previews/affected approvals are invalidated on input changes. `auto-activation`, `auto-job`. |
| AUTO-08 | Resolve exceptions and failures without inventing costs. | Missing source means waiting, not zero; unmatched SKU, conflicting rules and uncertain outcomes remain visible. Failed jobs retry affected scope; uncertain results require readback first. AI proposes, humans approve rates and activation. `auto-exceptions`, `auto-job`. |
| AUTO-09 | Keep effective-dated contract history and issued evidence immutable. | Freeze period/timezone, mid-period split/proration and renewal policy. Any post-issue correction links to the original document; simulation comparisons use identical input revisions. `auto-history` and retained correction screens. |
| AUTO-10 | Explain customer selling prices without supplier or operator disclosure. | Customer-only pages carry contractual rates, applied/excluded discounts and estimate/issued state, without supplier net cost, MSP margin, other customers, internal approval details or operator navigation. `auto-customer`. |

A rule activation gate must require unresolved policy fields to be completed: eligible dimensions, date intervals, timezone, quantity conversion/rounding, tax/FX stages, tier method and pool/reset, commitment underuse/overage, benefit pass-through and exclusion granularity. This design does not select one global formula or legal policy for every agreement. Provider procurement discounts already present in source net cost must not be reapplied as procurement benefits. Customer resale rules may intentionally use another approved basis.

# Synthetic design cases

PORT-OCT-S1 is an illustrative disjoint 5,000-company portfolio: 4,800 preview-ready (4,100 changed and 700 unchanged), 150 mapping-blocked and 50 conflict-blocked. These are screen fixtures, not a load test or loaded production records. Each target has its own pinned inputs. Portfolio monetary totals are unavailable; no aggregate is extrapolated from a single customer.

SIM-NS-OCT is an isolated Northstar October USD pretax simulation, independent of the retained September examples. SRC-OCT-S1 and LIST-OCT-L1 stay identical across comparisons. DB-instance basis1,000 less10%=900; DTO200 less20%=160; other eligible500 less5%=475. Proposed PB-NS-R2 totals1,535; comparison PB-NS-R1 without the sales concession totals1,560; delta−25. Supplier cost and margin are unavailable. Line amounts use two-decimal half-up rounding in this example only. No commitment/credit balance is consumed in this case.

The activation page shows a proposed one-customer target NS-OCT-T1 with approval pending; it does not claim 4,800 companies were activated. The job page is a complete-calculation state example, not actual execution or issuance. The customer page is a separate illustrative view assuming its displayed selling terms have been approved; it does not turn the operator proposal into an actual approval. Customer presentation is an estimate, with tax/settlement outside the amount. All source/product IDs are synthetic.

# Complete book and completion criteria

Preserve all 54 v8 pages (52 screen states and 2 guides), integrate 12 new states in their workflow chapters, and produce 66-page EN/KO full books. Maintain explicit old-to-new page coverage and a customer-only entry with eight states. Every new financial state has case/period context. Links navigate review states; financial controls remain visibly read-only.

Completion requires independent design and actual artifact review; arithmetic checks; complete print rendering; local links, loaded images, representative mobile widths and customer payload checks; final PDF visual inspection; scoped PR, authorized merge readback and machine release. Neither publication nor render checks establish production automation, provider interoperability, authentication or owner design sign-off.
