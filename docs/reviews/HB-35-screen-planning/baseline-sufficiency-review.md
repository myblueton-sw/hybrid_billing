---
type: DesignReview
title: External baseline sufficiency assessment of the billing prototype
description: Compares the fifteen-state English and Korean prototype with official MSP and provider evidence and identifies required operational design coverage.
status: draft
---

# Verdict and scope

**Subsequent design revision:** [v3](prototype-v3/README.md) now illustrates the representative remediation cases. This report preserves the pre-revision assessment; [v3 verification](v3-verification.md) records the new checks and remaining limits.

Reviewed on 2026-10-28 for [HB-35](https://linear.app/hybrid-billing/issue/HB-35/review-planning-baseline-for-the-complete-product-screen-mockup). Root owns synthesis and document edits; independent read-only reviewers `msp_workflow_research` and `csp_financial_research` assessed the actual prototype and official external evidence. Accountable owner: Seung Woo Park; working machine: mac_mini.

**PASS as a representative core billing story; FAIL as a complete baseline MSP operating-workflow design. Runtime capability is UNVERIFIED by this assessment.** This verdict applies equally to the neutral English and Korean editions. Earlier arithmetic, translation and visual QA passes remain valid within their stated scope; they did not certify operational completeness or all 49 baseline screens.

The current fifteen states correctly separate supplier originals, customer attribution and selling amounts; show a duplicate-fee investigation and revision comparison; distinguish approval, issuance, delivery and ERP evidence; and present a separate customer view. These are useful foundations. The missing part is the operator's repeated work of changing inputs, investigating failures, checking impacts and reaching an approved result.

“Baseline” means the owner's agreed monthly MSP billing scope, assessed against documented international operating patterns. It is not a universal certification, a requirement to copy every competitor feature, or a claim about statutory invoice validity. A static prototype can satisfy design coverage through coherent simulated forms and states; live integrations are not required for that judgment.

# Evidence and source limits

Official documentation was freshly retrieved by root or the independent reviewers on the review date. Vendor accounts and live financial operations were not tested. Publication dates were unavailable for the cited MSP knowledge pages. IBM direct retrieval returned 403; refreshed official indexed text was available and is supporting evidence only.

| Source | Relevant evidence and boundary |
| --- | --- |
| [Flexera Billing Plans](https://docs.flexera.com/flexera-one/partners/cloud-cost-optimization/billing-plans) | Customer assignments, ordered rules, adjustments, processing history and distinct handling of existing versus refreshed data. These support operator configuration and run-history views, not automatic adoption of vendor locking policy. |
| [Flexera Bill Splitting](https://docs.flexera.com/flexera-one/partners/cloud-cost-optimization/using-bill-splitting-to-assign-costs-to-child-organizations/) | Rules attribute supported supplier costs to child organizations. Its documented provider coverage does not establish universal support. |
| [Cloudmore Monthly Billing Check Routine](https://kb.cloudmore.com/kb/monthly-billing-check-routine) | Service reconciliation and customer/service/product margin investigation precede invoicing. Arithmetic differences and low-margin exceptions need distinguishable investigation. |
| [Cloudmore General Billing Report](https://kb.cloudmore.com/kb/general-billing-report) | Period, organization and service filters, hierarchical detail and export fields support scoped work. Do not copy current-rate export conversion into historical invoice policy. |
| [IBM Editing a Price Book](https://www.ibm.com/docs/en/cloudability-commercial/cloudability-msp/saas?topic=pricing-editing-price-book) and [Pricing Rules](https://www.ibm.com/docs/en/cloudability-commercial/cloudability-msp/saas?topic=pricing-cloudability-msp-editing-rules) | Indexed official evidence describes effective assignments, overlap prevention and rule ordering/shadowing. These are corroborating patterns, not selected product policies. |
| [AWS Savings Plans in CUR](https://docs.aws.amazon.com/cur/latest/userguide/cur-sp.html) | Upfront, recurring and unused-related records extend beyond the currently illustrated covered-usage offset pair. |
| [Microsoft Read Your Bill](https://learn.microsoft.com/en-us/partner-center/billing/read-your-bill) | Billed/unbilled and usage/nonusage distinctions mean daily usage reconciliation alone cannot prove invoice completeness. |
| [Microsoft Reconciliation Schema](https://learn.microsoft.com/en-us/partner-center/billing/modern-invoice-reconciliation-file) | Quantity, currency, dated conversion, rounding and tax fields require traceable provider-specific treatment. |
| [Microsoft Billing Scenarios](https://learn.microsoft.com/en-us/partner-center/billing/common-billing-scenarios-onetime-recurring) | Commitment term, billing frequency and charge interval differ; quantity changes need explicit proration treatment. |
| [GCP Standard Export Schema](https://docs.cloud.google.com/billing/docs/how-to/export-data-bigquery-tables/standard-usage) | Commitment-fee offsets, legacy CUD credits, promotional credits and reseller margin are distinct types. A generic discount must not collapse them or duplicate benefits. |
| [FinOps Invoicing and Chargeback](https://www.finops.org/framework/capabilities/invoicing-chargeback/) and [Allocation](https://www.finops.org/framework/capabilities/allocation/) | Invoice reconciliation, late adjustments, allocation and shared-cost responsibility support the monthly loop. Showback is foundational; formal internal chargeback depends on accounting policy. |
| [FOCUS 1.3 Invoice Handling](https://focus.finops.org/docs/specification/v1-3/attributes/invoice-handling/) | This versioned specification addresses monetary invoice components beyond usage, including taxes, credits and other charges. It is contextual evidence, not adoption of this version or certification of our exports. |

# Current design gaps and acceptance cases

The table is an assessment and proposed design completion criterion, not a change to financial policy. Existing [source/pricing obligations](../../contracts/billing/source-pricing-and-cost-obligations.md), [screen bindings](../../ui/screen-binding-register.md) and [screen map](prototype/README.md) remain authoritative inputs. Most findings concern absent visual coverage of already planned work.

| Area | Current visible coverage | Minimum next design case | Applicability / trace |
| --- | --- | --- | --- |
| Source completeness and intake | Five supplier controls pass; setup is a checklist | Show intake/version review and a missing nonusage charge despite internally balanced usage; investigate duplicate or quarantined evidence before clearing the hold | Core; S01–S06 and source completeness obligations. Authentication method remains deferred |
| Customer attribution and portfolio | Six-row search; another customer's issue opens an explicitly labelled Northstar pattern | Resolve an unassigned/ambiguous account, set effective ownership, preview affected costs and verify conservation; provide period/provider/currency scope and own-case destinations | Core; O/S/B allocation families |
| Price books and contracts | Seven already-matched rules and fixed calculation | Edit basis, conditions, priority, credit exclusions and effective assignment; expose unmatched/shadowed/conflicting lines and historical impact | Core; O03/B04–B08, SP-01/02/10 |
| Subscription lifecycle | Aggregate Azure seat charge | Show mid-cycle quantity change with subscription, term, frequency, charge interval, quantity, price and checked proration; include renewal or cancellation-credit branch | Required for seat/subscription selling; SP-08. No supplier purchase/cancellation execution implied |
| Commitments and Marketplace | Selected AWS offsets; no full recovery or Marketplace example | Show consumed/unused commitment responsibility, upfront recovery without rebilling, and distinct GCP credit mechanics; show a negotiated Marketplace concession excluded from blanket discounts | Required for applicable transactions; SP-01/02/10. No automatic commitment purchasing |
| Currency, tax and rounding | USD-only focal story, explicit synthetic zero tax; transformation order is text | Show source-to-sale FX pair/direction/date/version, positive-rate validation, distinct conversion bases, illustrative nonzero tax and provider rounding; changing pinned inputs invalidates approval | SP-03; FX only when currencies differ. No universal tax or rounding rule |
| Reprocessing and approval | Authored r7-to-r8 comparison and gate table; next-state links | Select existing-data recalculation versus refreshed-source processing; preview scope and dependencies; inspect queued/failed/unknown history; reject/request correction and reapprove a fixed revision after changed-input conflict | Core; B08–B09/I01. Simulated decision states suffice for design |
| Issued correction and invoice package | Late credit assessment stops at uncalculated customer impact; one customer invoice | Follow a linked credit through contract treatment, independently checked customer correction, approval and issued correction document; show a split-document case and original-preserving history | Core correction and applicable split rules; SP-03/05/06, B10/I/C families |
| Customer explanation and inquiry | Selected AWS detail; fixed inquiry example dialog | Show own invoice history, document/detail download states, correction history, line-specific inquiry draft/confirmation/tracking and an operator response visible to that customer | C01–C03. No live message, payment collection or broader support-service implementation |

The inspected HTML contains a worklist search and an example-dialog close control; navigation supplies the remaining interactions. This observation does not require a working backend in a mockup. It confirms that the configuration, failure and consequential decision paths above have not yet been illustrated. The AI panel should continue to support evidence, hypotheses and proposed changes; it cannot substitute for those paths or final human financial approval.

# Boundaries and recommended next design sequence

Retain internal allocation and showback for shared cloud/VMware costs. Retain internal chargeback as an applicable capability; illustrate it where an explicit accounting policy requires it; do not confuse it with the MSP's external selling invoice. Keep customer views free of undisclosed supplier costs and margins, and internal raw evidence locators. Margin review belongs to authorized operators.

Recommended design packages, using the existing screen families rather than inventing a replacement inventory:

1. Source intake, completeness failure and customer attribution resolution.
2. Effective contract/price-book editing, conflict coverage and impact preview.
3. Subscription proration, commitment recovery and applicable Marketplace/credit cases.
4. FX, nonzero illustrative tax, rounding and split-invoice explanation.
5. Reprocessing failures, approval rejection/staleness and a fully resolved issued correction.
6. Customer invoice history, document states and line-specific inquiry lifecycle.

The completion test is a connected scenario: an operator finds an incomplete or misassigned charge, corrects the scoped input, examines affected periods, reruns processing, independently reconciles, resolves review feedback and issues the fixed candidate; a later credit produces a linked correction that the customer can understand. Include relevant provider transaction variants without pretending every account uses every variant.

This assessment does not choose authentication, an AI vendor/model, payment adapters or new accounting policies. It does not reopen the separately deferred post-invoice MSP service scope. Existing invoice/ERP/terms/correction planning remains in scope. No UI, PDF or billing implementation changed during this research follow-up; the next mockup remains work to do.

# Verification and handoff

Actual checks: current prototype and local SP obligations inspected; external source claims reviewed separately by the MSP workflow and provider-financial reviewers; root reconciled findings with existing scope. Source retrieval limitations are recorded above. No real invoice, customer data, tax-law assessment, provider connection, tenant enforcement or financial runtime was validated.

Independent artifact review: provider-financial reviewer PASS and MSP-workflow reviewer PASS, with no blocking findings; their scope-precision wording recommendations were incorporated. Root checked frontmatter, local evidence links and whitespace. This document-only follow-up did not rerun visual or runtime tests. Git handoff is recorded in HB-35 and PR32. PR32 remains subject to its existing authorization boundary; this research grants no merge permission.
