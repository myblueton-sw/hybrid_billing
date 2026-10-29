---
type: ResearchReport
title: MSP contract pricing patterns and discount exclusions
description: Separates documented pricing capabilities from private agreements and proposes evidence-backed contract controls for the next billing design revision.
status: draft
---

# Scope and finding

[HB-41](https://linear.app/hybrid-billing/issue/HB-41/research-msp-pricing-rule-types-and-contract-application), researched 2026-10-29, machine mac_mini. Root owns research and writing; csp_financial_research independently reviews financial boundaries. This report supplements [v8](../HB-40-full-mockup/README.md); it does not change its PDF or implement a pricing engine. Existing scoped publication and main integration authorization applies.

A single customer discount field cannot express the requested business. The owner describes both service-wide DB-instance discounts and instance-type-specific discounts, then adds commitment, Data Transfer Out, sales discounts excluding individually discounted products, and CFRC. These are requirement examples, not verified public policies of any named MSP. Public material reviewed does not establish the precise private agreements described. CFRC's expansion and conditions remain UNVERIFIED; a clarification is pending. Do not equate it with CloudFront bundles, flat-rate plans or any rate card by inference.

# Primary evidence

| Source | Documented capability and boundary |
| --- | --- |
| [AWS Billing Conductor pricing rules](https://docs.aws.amazon.com/billingconductor/latest/userguide/create-pricingrule.html) and [FAQ](https://aws.amazon.com/aws-cost-management/aws-billing-conductor/faqs/) | Global, service, billing-entity and SKU scopes; percentage adjustment and SKU custom rates/usage tiers. More granular rules take precedence in this product. Custom SKU rates replace public On-Demand rates and do not affect RI/SP purchases. The result is pro forma pricing, not the supplier payable or a universal contract ordering standard. |
| [CloudHealth partner price-book and billing APIs](https://apidocs.cloudhealthtech.com/) | AWS custom price books support percentage decrease/increase and fixed rates, product/region/usage/operation/record-type filters and instance family/size constraints. Rules use first matching entry in top-down order, with dated groups; unmatched entries receive no price adjustment. Separate partner billing rules support recurring or one-time charges/credits, spend percentages and support minimum/tier arrangements. Product-specific API scope matters; AWS price books are not proof of identical Azure support. |
| [Google partner repricing](https://docs.cloud.google.com/channel/docs/rebilling/manage-repricing-configurations) | A base price can be Direct Customer Cost or List Price, with SKU-group overrides. Credit presentation differs by basis; reseller margin is hidden. Repricing exports and the customer's Cloud console are different views. Direct Customer Cost must not be renamed MSP net purchase cost. |
| [Azure partner pricing](https://learn.microsoft.com/partner-center/pricing/pricing-and-offers) and [PEC FAQ](https://learn.microsoft.com/en-us/partner-center/billing/partner-earned-credit-faq) | Usage pricing is meter-oriented; price lists depend on market and offer. Partner earned credit has eligibility conditions. Neither a retail price list nor a partner benefit establishes the MSP's customer resale discount. |
| [OCI cost reports](https://docs.oracle.com/en-us/iaas/Content/Billing/Concepts/costusagereportsoverview.htm) | Resource-level consumption evidence includes unit price, subscription and overage information. Match the applicable report schema and purchase agreement; consumption/commitment evidence alone does not establish customer selling rates or cash settlement. |
| [Alibaba billing models](https://help.aliyun.com/en/user-center/product-overview/quickly-understand-the-billing-modes-of-alibaba-cloud-products) and [CDN tier example](https://www.alibabacloud.com/help/en/cdn/product-overview/billing-rules-of-basic-services) | Subscription, pay-as-you-go, savings and resource-plan structures differ. CDN illustrates geographically differentiated usage tiers and cumulative monthly consumption. Supplier tier behavior is not automatically an MSP resale policy. |
| [Amazon RDS pricing](https://aws.amazon.com/rds/pricing/) | Instance usage, storage, IOPS and extended support are distinct charge components; engines and purchase arrangements differ. An agreement covering DB instances must not silently discount every RDS-labelled charge. |
| [AWS CUR line items](https://docs.aws.amazon.com/cur/latest/userguide/Lineitem-columns.html) and [CUR 2.0 discount columns](https://docs.aws.amazon.com/cur/latest/userguide/table-dictionary-cur2-discount.html) | NetUnblendedCost is an after-discount amount when present. Discount representation depends on export configuration: manual-discount compatibility uses separate line items rather than the total-discount column. Preserve schema/configuration and avoid subtracting the same discount twice. These fields do not fully encode private sales-contract exclusions. |
| [CloudFront pricing](https://aws.amazon.com/cloudfront/pricing/) and [FAQ](https://aws.amazon.com/cloudfront/faqs/) | Public flat-rate plans and the Security Savings Bundle are distinct. The pricing page excludes combining flat-rate plans with other offers/promotions/discounts; the FAQ excludes flat-rate plans from bundle savings. Transfer destination and origin matter: AWS-origin-to-CloudFront transfer is distinct from delivery to viewers. This does not identify CFRC. |
| [Megazone Hyper Billing](https://www.megazone.com/product/hyper-billing), [Bespin public support](https://support.bespinglobal.com/ko/support/home) | Public product/support evidence does not verify the owner's exact flat-versus-instance-type private contract examples. No vendor-wide discount percentages are inferred. |

# Proposed contract model

The following is a Hybrid Billing design proposal inferred from the evidence and owner requirements, not a claim that all cited platforms implement it. Separate six dimensions rather than creating one fixed list of mutually exclusive discount names.

| Dimension | Required expression |
| --- | --- |
| Authority and customer scope | Supplier purchase benefit versus customer resale rule; MSP, legal customer, contract, account/subscription and approved membership revision. |
| Eligible charges | Provider, service, native SKU/meter, charge component, instance family/type/size, engine/license, deployment, region, usage type/operation, purchase model; exclusions and explicit wildcard versus unknown. |
| Calculation basis | Dated public rate, already-net supplier cost, explicitly defined amortized allocation or contractual unit price; currency, unit, quantity and immutable evidence revision. |
| Formula | Percentage discount, cost markup, target gross margin, fixed unit price, tier schedule, fixed fee/credit or allowance. Markup and margin have different denominators. |
| Applicability and obligations | Effective interval/timezone, recurring or one-time applicability, commitment pool and period, minimum spend, expiry/carryover, underuse/overage and mid-period treatment. Duration is independent of formula. |
| Interaction | Explicit selection priority, replacement, permitted stacking, exclusion group, affected charge portion and evidence. Equal-priority unresolved overlaps block affected calculation; specificity alone does not authorize replacement. |

# Contract patterns to represent

All percentages and amounts below are synthetic examples, not vendor offers.

| Pattern | Example | Critical condition |
| --- | --- | --- |
| Service/component-wide | Eligible DB-instance compute charges 10% off list | Explicitly decide storage, backup, I/O, support and RI-covered usage inclusion. |
| Family/type-specific | db.m family 8%; db.r family 12%; exact approved SKU 15% | Resolve native identities with engine, region, deployment and purchase model; do not use instance display text alone. |
| Fixed negotiated unit | USD 0.12 per eligible instance-hour | Quantity precision, billing increments and unit conversion; alternative to a percentage, not automatically stacked. |
| Spend or quantity tiers | Two volume bands with different rates | Graduated versus all-units method, pool membership, reset window and boundary treatment. Never aggregate different currencies or units blindly. |
| Commitment benefit | Annual spend promise or capacity/usage commitment | Provider RI/SP/CUD, enterprise spend agreement and MSP-customer minimum spend are distinct contracts; record unused and excess treatment. |
| DTO-specific | Eligible internet egress 20% off | Identify service, traffic direction/destination, geography, tier and excluded transfer types. Not every network charge is DTO. |
| General sales concession | 5% off eligible residual charges | Owner requirement: exclude individually discounted products. Define which granted benefits count and whether exclusion is entire product, SKU, line or covered quantity. |
| Cost-plus / margin | Purchase basis plus 10%, or target gross margin 10% | For cost 100, markup gives 110; gross-margin target gives 100/(1−0.10), before specified rounding. |
| Managed-service fee | Fixed monthly fee or percentage with a minimum | Specify basis exclusions, support/Marketplace treatment, and tier method. |
| Credit / rebate | One-time25 credit or recurring monthly allowance | Grant identity, eligible balance, consumption and reversal history; no duplicate application on reruns. |
| CFRC | Owner-supplied label retained | Meaning, basis, scope, evidence and interaction unresolved. Keep inactive pending clarification. |

# Sales-discount exclusion example

For this synthetic customer contract, product-specific DB and DTO concessions belong to exclusion group SPECIAL_PRODUCT_PRICE. The general sales rule excludes charges with an actually applied benefit in that group. This is a proposed explicit interpretation of the owner requirement, not a universal rule that every provider benefit disqualifies sales discounts.

| Charge | Selected pretax basis | Special rule | General sales 5% | Final |
| --- | ---: | ---: | --- | ---: |
| Eligible DB-instance compute | 1,000 | 10% = −100 | Excluded: DB special price | 900 |
| Eligible DTO | 200 | 20% = −40 | Excluded: DTO special price | 160 |
| Other eligible service | 500 | None | −25 | 475 |
| Total, same currency | 1,700 | −140 | −25 | 1,535 |

Applying 5% to the post-special subtotal 1,560 would produce 1,482 and an unauthorized additional 53 discount on already-special-priced charges. If only part of a charge is eligible for a benefit, split its amount/quantity with traceable allocation before exclusion. A provider-only procurement discount must not accidentally count as a customer product concession; classify benefit authority and agreed exclusion membership. Unknown or missing benefit provenance is not equivalent to no benefit. Conflicting rules or unresolved exclusion granularity block the affected rows, not silently use the lowest price.

# Application and audit controls

1. Keep source facts immutable and normalize known native identities. Preserve missing/unknown fields; do not map them to a wildcard. A DB type may expand to several billable SKUs and a DTO scope to several services.
2. Resolve approved customer contract version and dated price basis. Existing net amounts are not discounted again to recreate provider purchase benefits. Supplier purchase reconciliation and customer price calculation have separate totals.
3. Resolve eligible price rules and exclusions explicitly. Record winning rule, replaced candidates, stacking sequence and excluded reason per line/portion. A 10% then 5% sequence yields 85.5 from 100; a 15% additive concession yields 85. Neither is implicit.
4. Apply allowed credits/fees using their own grant and base contracts. Keep taxes, currency conversion and rounding stages explicit and versioned; no unspecified universal stage order is adopted here.
5. Preview affected companies, line counts and monetary deltas. Missing source coverage, unmatched new SKUs, missing historical price evidence and conflicts are review queues, not zero-cost outcomes. A permitted fallback requires an explicit contract rule.
6. Separate draft, review, approved, scheduled, active and superseded versions. Recalculation invalidates affected approvals. Issued documents remain immutable; corrections reference original documents and rule versions.

# Next mockup implications

Expand the contract workspace around customer price books. Show a concise default rule, exceptions table and affected-cost preview; advanced matching appears only when needed. Each row needs target, basis, adjustment, period and interaction. A product selector must let operators distinguish DB-instance compute from all DB-service charges. Bulk rate-card import needs preview, unknown-SKU and duplicate/conflict validation before publication.

Add a discount-interaction view for special product prices, commitment benefits, DTO concessions, general sales discounts and credits. Show Applied / Excluded / Replaced / Needs evidence, amounts and reasons. Customer screens expose their contractual rates and explanation, while supplier net cost and MSP margin remain permission-controlled.

For 1,000–5,000 customers, use versioned templates with explicit customer overrides and impact previews. Do not silently rewrite historical customer contracts when a template changes. AI can propose mappings and identify ambiguous clauses; a human confirms rates, exclusions and activation. CFRC remains a visible unresolved requirement until its meaning is confirmed. No screen or financial engine change is claimed by this research ticket.

# Verification

Primary-source searches and page reads completed. Arithmetic examples are synthetic and checked independently from provider data. Initial independent design review identified duration/formula conflation and implicit specificity precedence; both are corrected above. Final independent report review PASS: duration/formula and precedence fixes confirmed; sales exclusion, procurement/resale separation and CFRC uncertainty accepted. Both root and reviewer independently checked 1,700 → 1,535, erroneous 1,482 and excess discount 53 using Decimal. Reviewer freshly read AWS pricing-rule and CloudHealth API evidence; other provider pages were researched by root, not all independently revalidated. OKF frontmatter and report scope checks passed. No live account access, vendor contract inspection, financial execution or runtime tests were performed.
