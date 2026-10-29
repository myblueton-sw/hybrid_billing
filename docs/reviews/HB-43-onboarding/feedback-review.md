---
type: RequirementsReview
title: Customer cost visibility feedback and duplication review
description: Reconciles new customer requests with existing billing requirements and defines only the missing design coverage.
status: draft
---

# Scope and duplication findings

HB-43 owner feedback is consolidated rather than creating parallel features for the same obligation. Existing contracts are requirements evidence, not proof of implemented or tested behavior. The owner treats a third-party cost API as peripheral, and asks whether ISV purchase forecasting fits the platform; the proposal keeps forecasting optional and distinct from actual billing.

| Request | Existing evidence / duplication | Result in v10 |
| --- | --- | --- |
| Product/contract discount and final price | HB-42 AUTO-01–05 and HB-43 PR-06–15 | Extend existing product/instance search and evidence; no duplicate rule engine |
| Raw versus console/Excel rounding discrepancy | Claim/numeric contract and existing reconciliation | Add named precision-stage case; no universal small-difference tolerance |
| Group-company allocation | SP-06/07, OP-02 and existing allocation screen | Extend existing screen with conservation/version/unallocated controls |
| AI version-specific amount mismatch | Existing versioned pricing/AI metering requirements | Add SKU/model/version/region/unit/effective-date discrepancy case |
| CSP support amount mismatch | Existing supplier PDF and late/missing-support examples | Add console/export/invoice comparison and scope/tier/minimum/as-of diagnosis |
| Project/service/SKU pivot | Collection architecture already permits provider/customer report axes | Add explicit pivot screen with day/month, project ID, service, SKU, charge type and region |
| Raw lookup/export | Existing usage-detail lifecycle and customer-safe export boundary | Add explicit raw explorer and filtered full-result export manifest |
| Marketplace ISV costs and contract links | SP-10 already requires transaction type, bill/contract evidence and discount exclusions | Add visible ISV register; do not invent missing source/private-contract metadata |
| Marketplace EA/EDP consumption | SP-10 and HB-38 distinguish commitment from discounts | Add actual purchase, assessed eligible contribution and confirmed contribution separately |
| Future ISV purchase contribution | Not demonstrated by existing actual billing | Optional scenario in the same commitment screen, using list-price assumptions and eligibility/limit uncertainty |
| Cost collection API | Existing intake and evidence boundaries; requested direction ambiguous | Peripheral adapter requirement; decide inbound source ingestion versus outbound customer cost access in its own contract, not a second billing core |

# Primary evidence checked 2026-10-29

Independent csp_financial_research reviewer inspected these official sources; no customer accounts, private contracts or real integrations were accessed.

- [Azure MACC FAQ](https://learn.microsoft.com/en-us/marketplace/macc-frequently-asked-questions): qualifying purchases depend on offer eligibility at purchase, channel and active commitment period; qualifying pretax amounts count at invoicing. Azure prepayment and MACC differ. Refunds can reverse consumption. EA is an agreement framework, not an interchangeable name for every commitment/prepayment. Existing SP-10 additionally preserves the Microsoft CSP private-offer exclusion.
- [AWS Marketplace Buyer Guide](https://docs.aws.amazon.com/pdfs/marketplace/latest/buyerguide/aws-marketplace-bug.pdf#page=222): eligible Multi-Product Solutions products can apply to EDP/PPA; eligibility is per product rather than guaranteed for a whole bundle. No universal EDP cap or percentage was verified; use the actual agreement.
- [Google Marketplace commitment policy](https://cloud.google.com/terms/marketplace/commit-policy): eligibility, commitment period, purchase/reseller relationship and applicable cumulative cap constrain drawdown. The currently inspected policy includes a 25% cap; it must not become a global cross-provider constant. The detailed linked rate/exclusion page could not be retrieved, so the full exclusions remain unverified. The [official partner announcement](https://cloud.google.com/blog/topics/partners/upgrades-to-google-cloud-marketplace-for-partners) describes qualifying MCPO purchases from June 9, 2025 based on final private-offer price; list-price plans are not confirmed drawdown.
- [AWS CUR billing columns](https://docs.aws.amazon.com/cur/latest/userguide/billing-columns.html) identify Marketplace billing entity and invoice/payer context, but do not establish commitment eligibility. [Subscription agreements](https://docs.aws.amazon.com/marketplace/latest/buyerguide/buyer-managing-subscriptions.html) provide a separate agreement/terms/PDF workflow.
- [Azure usage fields](https://learn.microsoft.com/en-us/azure/cost-management-billing/automate/understand-usage-details-fields) can supply publisher/product/order/plan attributes according to agreement/schema; availability and PublisherType semantics vary. Missing publisher metadata is not evidence of no Marketplace charge.
- [Google standard billing export](https://docs.cloud.google.com/billing/docs/how-to/export-data-bigquery-tables/standard-usage) includes service/SKU and transaction/seller fields under documented conditions; it is not a complete private-offer registry.

# Acceptance boundaries

Pivot and raw explorer operate on fixed permitted records. Distinguish usage date from invoice month, gross/discount/net semantics, source version, account/resource attribution and currency. No mixed-currency monetary total or mixed-unit usage sum; source-net costs cannot receive duplicate benefit subtraction. Filtering, pagination and exports cannot change pool membership or invoice economics. Preserve unknown/unallocated buckets and coverage limits.

The XPL-DEMO source has two illustrative rows: usage USD1,000 and discount minusUSD100 = USD900. The same two rows support pivot and raw detail. An export manifest binds requested and delivered scope, row count, decimal values, units, currency, filters, revision and checksum. Customer raw access is a permitted projection, never an automatic copy of shared-payer originals. Reauthorize delivery, protect CSV/XLSX formula injection, and distinguish partial/failed/truncated outputs. This mockup does not execute exports.

Discrepancy examples compare like scope before diagnosing: two KRW amounts1.6+1.6 give3.2; truncating the sum gives3 whereas truncating each gives2. The difference1 can be explained by stage only when policy and evidence match. Another synthetic support case compares console estimateUSD100 with export/invoiceUSD120; the difference20 remains unexplained, not automatically attributed to rounding. AI mismatches require exact SKU/model/version, region, metering unit, effective rate and source revision. Missing rows, late adjustments, tax/FX and account scope are distinct causes. Excel stored precision differs from its displayed format. Preserve originals and approved correction lineage.

Group allocation preserves allocated + unallocated = approved target per currency, with fixed weights, effective rule version, residual policy and closed/unmapped organization handling. Group visibility does not merge legal buyers or invoices. Marketplace license/usage, underlying infrastructure and MSP fees remain separate to prevent double billing. Link original procurement contract, ISV agreement/offer and customer selling terms by stable IDs rather than names.

PDF/AI may extract candidate publisher/product/term/discount clauses with source hash, page locator and confidence. Human review must confirm ambiguous or absent terms and executed/amended validity. Extraction alone never proves provider-recorded commitment consumption, signature, current eligibility or customer disclosure rights.

# Optional purchase forecast

A list-price ISV plan belongs in an optional commitment-planning extension adjacent to billing. Actual supplier cost, assessed eligible contribution and provider-confirmed consumption are separate measures, never an additive total. Customer selling price, cash discounts and prepaid balance consumption are distinct again.

Inputs: provider and specific commitment agreement/revision, total and confirmed used amount as of time, currency, eligible product/channel, planned purchase and qualifying invoice dates, list-price assumption, applicable contribution rate and remaining cap. Unknown required evidence yields confirmation-needed. A generic upper bound may use the minimum of eligible candidate amount, remaining applicable cap and remaining commitment only when the contract actually defines compatible caps; this is not a universal provider formula. Never treat a missing cap as unlimited or a missing eligibility rate as100%.

The illustration deliberately uses no real EA/EDP policy: totalUSD100,000, confirmed consumptionUSD40,000, planned list-priceUSD10,000, hypothetical100% eligibility and sufficient cap. Incremental scenario contributionUSD10,000 changes utilization40%→50% (+10 percentage points), leavesUSD50,000 and represents10% of original commitment or approximately16.67% of the prior remaining60,000. Name the denominator whenever displaying a percentage. Real negotiated prices, tax, dates, refunds, instalments and contract terms can change this result. Do not book the forecast as actual consumption, reserve the same balance twice across plans, or execute purchasing automatically. Scenario concurrency and overlapping plans require explicit exclusions/aggregation before production.

# Requirement links

- [Delivery checklist](checklist.md)
- [Source pricing, group billing and Marketplace obligations](../../contracts/billing/source-pricing-and-cost-obligations.md)
- [Numeric contract](../../contracts/billing/claim-and-numeric.md)
- [Allocation and onboarding obligations](../../contracts/operations/source-policy-onboarding-and-verification-obligations.md)
- [Usage-detail lifecycle](../../architecture/usage-detail-lifecycle.md)
- [Data collection architecture](../../architecture/data-collection.md)
- [Prior provider research](../HB-38-billing-operations/provider-pricing-research.md)
