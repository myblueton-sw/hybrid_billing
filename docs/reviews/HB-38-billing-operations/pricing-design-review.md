---
type: DesignReview
title: Contract pricing and monthly operations design proposal
description: Applies provider research to recurring and one-off MSP pricing with explicit evidence, calculation basis and issuance controls.
status: draft
---

# Ownership and boundary

HB-38; root owns this proposal and prototype files. Independent financial/domain reviewers must review the actual artifacts. This is a design proposal based on [provider research](provider-pricing-research.md), not implemented billing behavior. Existing [PC-01](../../contracts/billing/inputs-pricing-and-erp.md) and [source-cost obligations](../../contracts/billing/source-pricing-and-cost-obligations.md) remain authoritative.

# Four separate records

| Record | Purpose | Evidence and restrictions |
| --- | --- | --- |
| Provider observation | What the provider reported | Immutable source, object/row identity, schema/configuration, revision, usage/billing period, currency, signed amount and native type |
| Supplier agreement | What the MSP should pay or consume | Rate-card/contract version, eligible SKUs/accounts, term, fixed/tiered prices, commitment, exclusions and source evidence |
| Customer selling policy | What the customer has agreed to pay | Explicit list/net/effective/custom price basis; markup or margin; customer discount; fees/minimums; effective dates and accepted version |
| Credit/settlement ledger | How a particular benefit or settlement applies | Funder, beneficiary, layer, currency, eligibility, expiry, remaining balance, reservations, applications and reversals |

Customer identity, cloud account, payer account, commitment pool and invoice legal entity are different dimensions. Scope every relation to the authorized MSP and effective period; a search filter does not redefine the financial pool. Customer views expose their approved selling calculation, never another customer's terms or confidential MSP purchase margin.

# Policy configuration

Use independent axes instead of a single free-form discount percentage:

- **Scope:** approved MSP customer pool, selected contract/customer, and eligible provider/service/SKU/accounts. Marketplace, support, tax, commitment purchase and credits require explicit inclusion or exclusion.
- **Duration:** recurring with effective start/end, or one-off with one eligible occurrence and an immutable grant identity. A one-off policy can target a pool; a recurring policy can target one customer. A pooled one-off benefit must explicitly choose one shared capped grant or separately capped grants per eligible beneficiary; pool membership alone does not multiply entitlement.
- **Basis:** provider billed net, approved effective allocation, applicable list price, custom rate card or another evidenced contract base. Show source period, currency, unit and whether discounts are already embedded.
- **Effect:** fixed unit price, tier, markup, target margin, percentage discount, fee, minimum/maximum or a separately typed credit. Markup and margin are different formulas.
- **Composition:** explicit replace, additional adjustment or exclusion. Specificity alone must not silently select a winner. Equal-priority conflicts block affected calculations.
- **Governance:** draft/review/approved/effective/expired revision, approver, reason and evidence. Changes preview affected companies, periods, invoices and approval invalidations before activation.

An MSP-wide default plus a customer's permanent exception must visibly show the inherited rule and the exception's replace/add behavior. A one-time goodwill benefit must retain its own identity and reason; it must not alter the recurring contract rate.

# Calculation and reconciliation

Retain PC-01's sequence: quantity/target → base selling price → contract discount → additional fees → minimum/maximum → FX → selling tax/rounding. A special sequence requires an explicitly approved exception. A credit is not automatically inserted at a universal stage: classify whether it changes taxable price, settles a receivable, or is a provider-side purchase benefit, and apply the approved contract/tax treatment.

Every result records operands, units, currency, effective time, source snapshot, rate/policy revision and rounding. Evaluate tiers, allowances and minimums over the complete contractual pool before customer display or pagination. No summing across currencies without an approved FX basis. A missing contractual rate or unresolved conflict is a blocking reason, not zero.

Synthetic pre-tax illustration, with no fees/minimum/FX and an explicitly agreed pre-tax goodwill adjustment: supplier net USD1,000; customer base uses 10% markup → USD1,100; customer 5% sales discount → USD1,045; one-off USD25 adjustment → USD1,020. This does not reapply the supplier's enterprise discount. Re-preview remains USD1,020; it does not consume another USD25. A 10% target margin would instead start at 1,000/(1−0.10); subsequent discounts reduce the achieved margin. Rounding follows the actual selected policy, not this example by implication.

Provider credits require signed source treatment and reconciliation; customer pass-through is a separate contractual decision. Credit reservation/application must be atomic and idempotent across retries and concurrent invoices. Cancellation releases a reservation; reversal creates an auditable restoring event. No silent reuse, cross-currency consumption or unlimited negative balance. A retry key alone does not replace concurrency protection.

Draft changes invalidate affected approvals. Issued invoices remain immutable and use linked corrections/credit documents. Unknown issuance outcome is inspected before retry; a delivery or ERP failure references the original invoice and never authorizes reissuance.

# Operator story for 1,000–5,000 companies

Start with one billing month and the selected MSP workspace. Default to unissued companies, ordered by overdue/due status with stable target identity. Keep company name/ID search visible; collapse less frequent owner/currency filters. Single-company mode presents the same company's history without artificial MSP filters.

The month overview separates disjoint all/unissued/issued totals from post-issuance exceptions, which are a subset of issued. For the planned synthetic 5,000-target month: 900 unissued + 4,100 issued = 5,000. Delivery/ERP exceptions are 115 of the issued targets, not an additional 115 targets. Whole-month counts and filtered results have separate labels.

Show a primary blocking reason, responsible person, due date/overdue status and next safe action on each company row. The synthetic 900 unissued targets partition into missing source125, mismatch100, contract75, review350, ready125, hold75, confirmed issuance failure35 and unknown outcome15. Additional reasons remain in detail; counts are not independent overlapping badges presented as additive. Missing due dates are not overdue.

Use 25/50/100-row pages and preserve month/search/filter/page on detail return. Customer and invoice IDs link detail, never display names or arbitrary fixed example IDs. A 5,000-row local prototype demonstrates interaction only. Production requires authorized server-side queries, stable ordering/cursors, consistent count/result snapshots and invalidation when the billing run changes; no production performance claim follows from local pagination.

# Required pricing and contract views

1. **Contract register:** customer, supplier agreement versus customer sales agreement, provider, validity, coverage, rule version, evidence freshness and conflict state.
2. **Rate evidence:** reported net cost and embedded discount, contract expected rate, difference and source link. OCI additionally shows subscription, rate-card term and commitment balance; GCP identifies consumption model; Azure agreement type; AWS export representation.
3. **Customer policy:** inherited MSP default, permanent exception and one-off entries with scope, duration and replace/add/exclude behavior.
4. **Impact preview:** affected companies/months, old/new customer amount and margin, excluded SKUs, unresolved evidence, approvals becoming stale and already-issued documents needing corrections.
5. **Calculation detail:** provider amount → selected customer base → sales adjustments → fees/minimums/FX/tax → issued amount, with independent supplier-invoice reconciliation and credit balance/application evidence.

AI may propose PDF/contract fields, match supplier invoice lines and explain differences with page/row evidence and uncertainty. A deterministic calculator verifies totals and rules. A person confirms ambiguous mappings and contract changes; AI does not activate rates, consume credits or issue invoices by itself. Re-imported PDFs retain document hash/version and do not duplicate financial events.

# Acceptance cases and remaining verification

Required cases: already-discounted source; customer replace versus additive exception; conflicting rules; one-off retry/concurrency; partial and expired credit; excluded Marketplace SKU; mid-month contract change; late correction after issuance; incomplete source; multi-currency view; OCI overage already included; GCP reseller margin separation; Azure Actual/Amortized separation; AWS discount-row/map alternative representation; Alibaba funded coupon settlement layers.

Public-source evidence is available. Real export validation, private agreement interpretation, billing-engine execution, production scalability, PDF extraction accuracy and runtime authorization are UNVERIFIED. No revised screen or PDF is certified by this proposal alone.
