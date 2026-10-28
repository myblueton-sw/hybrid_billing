---
type: ResearchReport
title: Multi-provider MSP billing evidence and screen requirements
description: Reassesses supplier evidence, customer pricing, allocation, reconciliation and invoice presentation across cloud providers and international MSP products.
status: draft
---

# Finding and scope

[HB-35](https://linear.app/hybrid-billing/issue/HB-35/review-planning-baseline-for-the-complete-product-screen-mockup), 2026-10-28. Root owns synthesis and document edits; separate read-only csp_financial_research and msp_workflow_research supplied provider and competitor evidence. The owner rejected the two earlier storyboards as insufficient for full CSP billing and requested renewed research. They also asked whether showback/chargeback belongs in this product. This report answers both without certifying provider integrations or changing accounting policy.

**The two frames are narrow arithmetic illustrations, not an adequate multi-provider invoice design.** The missing story is a portfolio of customers, supplier documents, commercial relationships and billing periods progressing through repeated source validation, allocation, pricing, reconciliation and approval. An invoice PDF is one output of that process. Adding provider logos to the existing table would not resolve these gaps.

The five cloud families researched are AWS, Azure, GCP, OCI and Alibaba; VMware/on-premises requires a separate cost-pool model. HawkEye and TokenMeter remain in the existing product scope, governed by local contracts; this research does not claim fresh external verification of them. Regional, sovereign, reseller and account-type variants are not universally verified. Existing 49 screen IDs and 43 source requirement groups remain the inventory; the workspaces below are design groupings, not new screen IDs or an approved scope reduction.

# Four financial objects, with separate controls

| Object | Question and financial control | Presentation consequence |
| --- | --- | --- |
| Supplier original documents | What does each supplier/seller bill the purchasing legal entity? Compare document-specific totals and admitted evidence on matching currency, period and charge coverage. | Preserve invoice/credit/debit identity, issuer, payer, dates, currency, revision and originals. A receipt, pro forma or usage export is not automatically a payable invoice. |
| Normalized and allocated costs | Which customer, department or service consumed or bears which cost? Preserve billed versus effective/amortized basis and allocation conservation. | Explain mappings, shared pools, unallocated/idle amounts and allocation rules. Do not require amortized totals to equal current invoice cash charges. |
| MSP selling calculation | What does the customer contract permit charging? Validate independently expected selling amounts under fixed inputs and rules. | Preserve purchase cost, selling basis, customer rates/discounts, commitments, fees, FX and tax as separate components. Purchase concessions do not imply customer pass-through. |
| Issued customer documents | What approved amount was issued by which seller to which customer? | Lock issued version and link corrections. Keep delivery, ERP acknowledgement and payment evidence separate. Customer packages disclose only permitted scope. |

These distinctions implement existing [source-pricing and cost obligations](../../contracts/billing/source-pricing-and-cost-obligations.md) SP-01–SP-10 and [billing/ERP controls](../../contracts/billing/inputs-pricing-and-erp.md). Much of the missing detail already exists in planning; its absence from the two frames was a presentation failure, not proof that the requirements were absent.

# Provider evidence and differences

Evidence below is official documentation inspected on 2026-10-28, not account/API testing. Descriptions are scoped to the linked products and documented account models.

| Provider | Documented distinction | Required screen evidence and failure state |
| --- | --- | --- |
| AWS | CUR is estimated during the month and can change after finalization for later credits/refunds/support. Invoice identity, payer and issuer are explicit billing fields. [CUR](https://docs.aws.amazon.com/cur/latest/userguide/what-is-cur.html), [billing fields](https://docs.aws.amazon.com/cur/latest/userguide/billing-columns.html) | Source version/as-of, invoice identity, payer/usage accounts and issuer; show new evidence against an already reconciled month. Do not silently replace its control. |
| AWS commercial variants | Billing Conductor pro forma is a separate domain from AWS payable charges. Commitment line types and offsets need typed treatment. [Pro forma](https://docs.aws.amazon.com/billingconductor/latest/userguide/understanding-proforma.html), [line items](https://docs.aws.amazon.com/cur/latest/userguide/Lineitem-columns.html) | Mark payable versus pro forma; preserve commitment/offset/unused components and seller identity. Do not sum covered usage and offsetting rows indiscriminately or use pro forma as supplier control. |
| Azure CSP | Billed reconciliation contains usage and nonusage charges; daily rated usage omits licenses/flat SaaS fees and can show zero charges for reservation-covered usage while the fixed purchase fee appears in invoice reconciliation. [Bill and reconciliation](https://learn.microsoft.com/en-us/partner-center/billing/read-your-bill) | Separate daily usage, unbilled estimates and billed reconciliation. Coverage checklist includes recurring seats, fixed fees, reservations, Marketplace, taxes and adjustments. A daily-usage difference is not automatically an error. |
| Azure arithmetic | Reconciliation documents distinguish usage/seat rounding and pricing-to-billing FX; post-billing CustomerCredit can override ordinary subtotal recomputation. [Schema](https://learn.microsoft.com/en-us/partner-center/billing/modern-invoice-reconciliation-file) | Preserve supplied subtotal/tax/total and explain formula/credit treatment. Provider-specific comparison rules precede residual classification; no universal round-at-end assumption. |
| Azure visibility | Partner cost and customer retail views can use different rates. [Partner Cost Management](https://learn.microsoft.com/en-us/azure/cost-management-billing/costs/get-started-partners) | Label acquisition, retail and contracted selling bases. A customer-visible provider estimate does not establish the MSP invoice price. |
| GCP | Invoice month can differ from usage month; corrections can negate and remonetize entries. [Export schema](https://docs.cloud.google.com/billing/docs/how-to/export-data-bigquery-tables/standard-usage) | Separate economic/usage period and invoice period; retain correction links and negative/replacement rows. Show late usage without rewriting historical evidence. |
| GCP documents and resale | Billing-period reports can span multiple invoices; Channel exports and rebilling add customer repricing. [Invoice reports](https://docs.cloud.google.com/billing/docs/how-to/reports/charges-on-invoices), [Channel export](https://docs.cloud.google.com/channel/docs/rebilling/export-data-to-bigquery), [rebilling](https://docs.cloud.google.com/channel/docs/rebilling/rebilling-overview) | Per-document control plus reseller/customer mapping. Identify already-repriced inputs before applying an additional MSP rule. Channel eligibility is not established for every GCP account. |
| OCI | Parent reporting includes child consumption; tenancy reassignment splits costs across subscriptions over time. [Organization reporting](https://docs.oracle.com/en-us/iaas/Content/General/organization/organization_cost_reporting.htm) | Preserve tenancy, subscription and effective assignment history. Parent access is not permission to disclose the full original to every customer. |
| OCI detail | Cost reports provide resource-level invoice-line detail in proprietary and FOCUS formats, with late-arriving data possible. [Cost reports](https://docs.oracle.com/en-us/iaas/Content/Billing/Concepts/costusagereportsoverview.htm) | Show source format/version, detail coverage and arrival revision. Report availability does not alone establish invoice completeness or customer selling terms. |
| Alibaba | Orders, monthly bills, spending detail and usage are distinct; subscription and PAYG differ. Coupons have eligibility/combination conditions; savings/resource plans have coverage limits. [Expenses and Costs](https://www.alibabacloud.com/help/en/user-center/product-overview/billings-and-costs-product-introduction/) | Preserve order/subscription lifecycle, billable item, usage/unit/rate/formula, benefit type and coverage. Track summary/detail readiness separately. Global documentation does not certify China-region or every reseller variant. |
| VMware/on-premises | Aria Operations 8.18.x Allocation/Reservation Pool pricing can exist only at OrgVDC level rather than VM/vApp level. [Broadcom KB439948](https://knowledge.broadcom.com/external/article/439948) | Actual policy/version determines available grain. Show unavailable VM price detail explicitly; do not change a customer's commercial policy merely to populate a UI. Metering alone is not cost allocation or an invoice. |

FOCUS is a versioned analytical representation, not a universal legal-invoice source. [AWS FOCUS dictionary](https://docs.aws.amazon.com/cur/latest/userguide/table-dictionary-focus-1-2-aws-columns.html) distinguishes BilledCost and EffectiveCost; preserve invoice and source extensions. [Azure exports](https://learn.microsoft.com/en-us/azure/cost-management-billing/costs/tutorial-improved-exports) have scope restrictions; [GCP exports](https://docs.cloud.google.com/billing/docs/how-to/export-data-bigquery) labels FOCUS export Preview. Keep source/target/mapping versions and alternate acquisition paths; do not select a new platform version from these examples.

# International MSP operational evidence

These are workflow references, not visual approval or feature parity. No logged-in vendor workflow was executed and no vendor screenshot was successfully inspected in this renewed research.

| Product and evidence strength | Documented behavior | Design implication, not copied policy |
| --- | --- | --- |
| Flexera, directly opened [billing plans](https://docs.flexera.com/flexera-one/partners/cloud-cost-optimization/billing-plans) | Ordered rules and adjustments; reprocessing existing data differs from unlock/reprocess with new supplier data. | Separate rule recalculation from source refresh plus recalculation; preview affected customer-periods and invalidate stale approvals. Do not inherit automatic lock dates. |
| Flexera, directly opened [Billing Explorer](https://docs.flexera.com/flexera-one/partners/cloud-cost-optimization/using-billing-explorer-to-analyze-cloud-spend) and [price books](https://docs.flexera.com/flexera-one/partners/cloud-cost-optimization/using-price-books-to-manage-discounts-and-adjustments/) | Provider/customer cost bases differ; locked periods can exclude late charges until reprocessing. Splitting precedes discounts/markups. | Validate attribution before pricing; display basis, snapshot, rules and late-source state. Supplier discounts and customer discounts require separate explanations. |
| IBM Cloudability MSP, official search-index text; direct editing-page retrieval failed: [create rules](https://www.ibm.com/docs/en/cloudability-commercial/cloudability-msp/saas?topic=pricing-creating-rules), [edit rules](https://www.ibm.com/docs/en/cloudability-commercial/cloudability-msp/saas?topic=pricing-cloudability-msp-editing-rules), [price books](https://www.ibm.com/docs/en/cloudability-commercial/cloudability-msp/saas?topic=pricing-editing-price-book) | First-match behavior, unmatched list-cost fallback and zero list-cost credit rows can materially affect results; assignments have effective dates. | Show rule coverage, unmatched monetary exposure, shadowing and effective timelines. Do not adopt list-cost fallback or erase credits by default. Evidence is indexed official text, not inspected UI. |
| CloudHealth, official [partner API](https://apidocs.cloudhealthtech.com/#partner-billing-rules), [GCP API](https://help.cloudhealthtech.com/graphql-api/) and directly opened [Azure flow](https://help.cloudhealthtech.com/partners/billing-and-invoicing) | Customer/global adjustments, explicit GCP client targets and customer-only reconciliation output; the documented Azure flow has restricted reprocessing. | Scope support/custom charges and affected customers explicitly. Do not assume every provider supports identical replay/correction operations. Azure evidence is flow-specific and dated. |
| Cloudmore, directly opened [monthly checks](https://kb.cloudmore.com/kb/monthly-billing-check-routine) and [billing report](https://kb.cloudmore.com/kb/general-billing-report) | Review drills from service reconciliation to customer margin, customer/service and lines; aggregate totals can conceal problems. Reports distinguish billing/provisioning dates and price levels. | Begin with portfolio exceptions, then drill to evidence. Margin alerts are not arithmetic residuals. Do not adopt current-rate historical export conversion as the financial FX policy. |

# Showback, allocation, chargeback and external invoicing

**Recommendation: retain the capabilities, with explicit purpose and applicability.** [FinOps Foundation](https://www.finops.org/framework/capabilities/invoicing-chargeback/) treats showback as foundational visibility and formal chargeback as dependent on accounting policy. Neither is a substitute for supplier reconciliation or MSP resale invoicing.

| Capability | Meaning here | Applicability and boundary |
| --- | --- | --- |
| Showback | Explain attributable cost/usage by customer, department, project, service and resource at available grain. Label actual/effective/allocated/estimated/selling basis. | Needed for understanding and investigating bills. A customer sees only authorized disclosed amounts; internal purchase cost is not automatically shown. A showback report is not a payment demand. |
| Allocation | Apply approved distribution rules to shared CSP costs, support, commitments or VMware/on-prem cost pools. | Needed wherever costs are shared. Preserve basis, effective revision, allocation denominator, idle-cost bearer and explicit unattributed remainder; allocated plus retained amounts conserve the pool. No invented VM detail or silent equal split. |
| Internal chargeback | Attribute approved expenses to organizational budgets/cost centers and, where required, accounting systems. | Applicable according to the deployment's accounting policy. This is not a newly deferred or deleted product capability. Preserve existing internal settlement, ERP recognition and correction obligations. Departmental charges differ from intercompany transactions. |
| External MSP invoicing | Apply customer commercial terms, approve the selling result and issue a customer document. | Central monthly story. Internally allocated cost and external selling price must not be added as two charges for the same economic item. |

Local SP-04 already separates internal allocation from external sale; SP-06 distinguishes provider originals, derived customer attribution, internal cost statements and MSP sales invoices. Avoid two mandatory top-level menus merely to display the words Chargeback and Showback. Proposed UI labels are Cost Breakdown, Allocation and Internal Charging where applicable, with Customer Invoices as a separate destination. This is a navigation proposal, not a change to existing IDs.

# Revised screen story and required drilldown

| Workspace / existing family | Essential information | Main action and completion evidence |
| --- | --- | --- |
| Setup / O, S, P | Operating profile, seller/customer, provider relationship, account coverage, contract periods, currency, responsible roles and acquisition capability | Resolve prerequisites; no fabricated zero totals for a new installation. |
| Monthly close portfolio / U | Customer × period × issuer/currency, expected/received sources, estimated/confirmed amounts, selling draft, rule coverage, exceptions, owner and next action | Open the highest-impact case. Keep readiness, financial approval, issuance, delivery and ERP outcomes independently visible. |
| Supplier documents and evidence / S | Original document list; payer/seller, invoice/usage periods, charge coverage, source format/revision/as-of | Reconcile one document's controls to admitted detail. Drill account/project/subscription → service/SKU/charge type → resource/time → original row, only to supported grain. |
| Allocation and showback / B, R | Effective customer mapping; shared pools, denominator, unused/unallocated amounts, department/resource split and basis | Preview distribution and validate conservation. Keep internal cost explanation separate from sales amounts. |
| Contracts and selling rules / O, B | Rate basis; rule priority and dates; covered/missing lines; provider versus customer concessions; fixed fees, commitments, credits and lifecycle/proration | Explain matched rule and preview affected periods. No silent fallback, double credit, duplicated prepayment recovery or loss-driven price increase. |
| Exception workbench / S, B, A | Missing/duplicate/late source, mapping, pricing, credit, FX, rounding, coverage and margin categories; affected value, evidence and responsibility | Investigate with AI assistance; propose authorized corrections. Margin investigation is distinct from a failed money control. |
| Recalculation comparison / B | Old/new input, mapping, pricing, FX revisions; affected customers and periods; before/after components; independent checks | Recalculate and reconcile repeatedly. Distinguish existing-data recalculation from new-source intake. Retain approval history while marking affected approvals stale and unusable for issuance; retain unknown job outcomes explicitly. |
| Customer package review / B, I | Customer selling lines, permitted supporting detail, net/tax/gross bridge, prior-period corrections, document split reasons | Independently approve the fixed candidate; evidence confirmation, adjustment approval and financial approval remain distinct. |
| Issue and correction / I, B | Approved version, issuer/recipient, immutable document identity, linked correction, delivery and ERP read-back | Issue only after required gates. New source evidence after issuance creates impact assessment, not direct modification of issued amounts. |
| Customer experience / C | Own estimate separately from issued invoice; provider/service breakdown at selling basis, disclosed discounts/credits, tax, corrections, downloads and line inquiry | Understand and question a bill without exposure to MSP purchase cost, margin, other customers or internal cases. |

AI belongs beside exception evidence and revision comparison: facts, hypotheses and proposed corrections are separate; deterministic tools produce monetary previews. No AI balancing entry, blocker waiver, financial approval or issuance authority. Manual operation remains possible when the model is unavailable. Provider/model and external-data policy remain unselected; the [existing intelligence contract](../../architecture/intelligence-layer.md) governs the proposal.

# Invoice package hierarchy

The next mockup must show a document package, not a single unexplained total:

1. **Document header:** kind, issuer and customer legal identity, invoice/correction reference, invoice date, service period, currency, approved payment terms and permitted status evidence.
2. **Summary:** net, approved tax components and gross; separately identified prior-period corrections, discounts and credits. No assumed universal tax rate or FX.
3. **Commercial detail:** cloud/provider/service or on-prem service groups; usage, seats, fixed subscriptions, managed-service/support fees, Marketplace and contract adjustments where applicable. Quantity/unit/unit-price may be unavailable for a valid fixed/aggregated charge; show its supported basis.
4. **Explanation:** account/project/subscription/resource drilldown where authorized, applicable selling rule/period, disclosed commitment or discount treatment, source-derived versus calculated provenance and inquiry reference.
5. **Supporting documents:** own permitted statement/detail export and correction history. A shared provider original requires authorization for its full scope; filtering a page is not sufficient.

Multiple CSPs do not automatically mean one invoice. Under SP-06, seller, recipient, currency, period, tax treatment and applicable purchasing/ERP-routing context determine combination or split. A consolidated overview may link several invoices without becoming an additional payment demand. Internal allocation reports remain separately identified.

# Acceptance cases for the next visual design

The revised mockup is adequate only if reviewers can follow these cases without explanatory prose substituting for visible evidence:

- A portfolio contains a ready customer, missing supplier detail, unallocated cost, unmatched pricing, an issued invoice with failed delivery and a revised supplier source awaiting impact assessment.
- One customer uses several providers; incompatible currencies or issuers split documents, while a permitted combined invoice retains service-level detail.
- AWS commitment offsets, Azure fixed subscription charges and a GCP prior-usage correction each reach the appropriate supplier control and customer selling treatment without double counting.
- OCI reassignment preserves historical tenancy/subscription mapping; Alibaba coupon/plan eligibility is distinguished from a generic MSP discount.
- A VMware cost pool shows allocation, retained idle cost and unsupported VM price detail; internal chargeback where applicable under deployment policy does not create a duplicate external charge.
- A pricing change shows affected lines/customer-periods, then revised calculation and stale approval; new source arrival is a separate cause of recomputation.
- AI offers a cited hypothesis and preview; missing evidence or no progress returns an actionable human case rather than an invented adjustment.
- The customer sees only their authorized document/detail and can reference a specific charge in an inquiry; internal economics and other tenants remain absent.

# Limits and next deliverable

This is a researched screen brief, not a completed visual redesign. Earlier SVGs remain historical narrow studies and the v1.16 PDF has not been regenerated. No live vendor UI, local provider connector, financial engine, tax compliance, account entitlement, AI endpoint or customer usability was tested. IBM indexed-text limitations and screenshot failures are explicit above. Country-specific invoice legality and actual commercial agreements still need their own verified inputs.

Next visual work must use the portfolio-to-document story and acceptance cases above, with representative provider differences and correction states. Preserve the existing inventory and binding obligations; authentication selection and the separately deferred post-invoice MSP service remain unchanged. Research does not authorize merge, deployment, real-account access or financial actions.
