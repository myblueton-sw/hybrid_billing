---
type: DesignAssessment
title: Purpose-led billing navigation and task narrative
description: Compares official billing-platform workflows with Hybrid Billing and proposes navigation around monthly consolidated cost, settlement and delivery.
status: draft
---

# Assessment and evidence boundary

HB-76, 2026-11-03. Root owns this assessment. The owner requested external billing-platform comparison and purpose-driven storytelling. The current menu covers technical domains but does not adequately communicate the core product outcome: consolidate multiple cost sources and carry explainable amounts through organizational allocation, settlement, approval and document/ERP delivery.

This is a proposed IA revision, not an approved replacement or a code change. Existing canonical screen IDs and contracts remain the baseline. Official documentation was read; authenticated competitor applications and usability studies were not executed. Documentation navigation is not evidence of a product's exact LNB. Only CloudZero's navigation guide explicitly describes its sidebar. Financial or permission behavior is not inferred from competitors.

# External comparison

| Official evidence | Observed purpose and flow | Adopt for Hybrid Billing | Boundary |
| --- | --- | --- | --- |
| [CloudZero navigation](https://docs.cloudzero.com/docs/navigating-cloudzero) | Daily tools surface consolidated cost exploration, analysis and allocation dimensions; connections/users/integrations sit in settings. | Start from consolidated cost and drill into organization/provider/time; keep administrative controls distinct. | Cost intelligence alone does not cover our settlement, approval or ERP obligations. |
| [CloudBlue usage flow](https://connect.cloudblue.com/community/modules/usage_module/) and [distributor/reseller usage interface](https://connect.cloudblue.com/community/modules/usage_module/distributor-portal/) | Vendor usage reporting feeds partner review/acceptance, billing export and reconciliation data. | Treat intake, review, downstream billing and returned reconciliation as connected work, with meaningful pending/rejected states. | A reseller flow is relevant to external-selling installations; it is not a required model for internal-only customers. |
| [Orb billing flow](https://docs.withorb.com/how-orb-works) and [invoicing](https://docs.withorb.com/invoicing/introduction) | Activity, billable measurement, prices and customer terms lead to invoices whose lines explain their basis. | Preserve drill-through from final amount to source quantity and applied policy. Keep price configuration separate from daily billing execution. | Product usage events and supplier cost imports are different inputs; Orb's subscription model is not our organization or tenant model. |
| [Stripe documented metered-billing flow](https://docs.stripe.com/billing/subscriptions/usage-based/implementation-guide) | A documented Meters flow connects measurement, price, customer/subscription and upcoming invoice preview. The current guide explicitly directs new integrations toward Metronome. | Preview the amount and its basis before an issuance action; make customer/contract context persistent. | This is workflow evidence, not a recommendation to adopt Billing Meters or duplicate Stripe's menu. Payment collection capabilities must follow our actual contracts. |

No third-party code or assets are imported. These sources support patterns; the following grouping is root's product-design inference.

# Problems in the current LNB

- Organization and technical-source groups appear before the daily cost/settlement task, obscuring the product's integrated view.
- Source rows and usage are buried among connection and evidence operations, although users need them for everyday cost investigation.
- Allocation, rate configuration and cost exploration are grouped as implementation domains rather than clearly distinguished tasks.
- The broad settlement group lacks an explicit sequence from preparation through approval, issuance and external reflection.
- Installation, access administration and Assistant compete with recurring business work as peer destinations.
- Removing the screen directory does not solve these problems. A menu also needs period/organization context, task status, evidence and a next permitted action across screens.

# Proposed navigation and complete screen mapping

Order the business menu around recurring work. Initial setup has its own guided entry, rather than forcing every returning user to start with connectors.

| Product destination | User question | Existing screens |
| --- | --- | --- |
| Consolidated overview | What happened this month, and what needs my attention? | U01, U03, U04 |
| Costs and usage | What costs were received, from whom, and how are they attributed? | S04, B01–B07; distinguish exploration, allocation and pricing subgroups |
| Settlement and billing | What can be calculated, approved, issued and handed off? | U02, B08–B11, I01–I04; separate runs/approval, documents/delivery and corrections |
| Organizations and contracts | Which organization/customer and terms govern this amount? | O01–O03 |
| Data collection and validation | Is input available, complete enough for the intended task and reconciled? | S01–S03, S05–S06 |
| Reports and audit | Can I explain and export the result and its history? | R01–R02 |
| Administration, in a separate lower utility area | Who can operate this installation and how is it maintained? | M01–M04, P01–P07; preserve distinct permission scopes |

A01 remains a globally accessible, permission-gated assistant entry; it does not become a required workflow stage. M05 stays in the account menu. C01–C03 remain a separate customer portal. L01–L03 remain pre-login. These placements cover all 49 canonical screens exactly once. They do not authorize merging screens, dropping fields, granting access or adopting new command semantics. Membership in an admin section never makes all administration visible to one role.

Internal installations use organization/department and internal statements; external-selling installations add permitted customer/contract/invoice context. Neither operating profile creates a legal entity, customer, contract or permission, or determines a legally valid document type by itself. U03 retains its separately authorized customer projection and M05 its self-only projection despite each having one canonical placement. Actual current authority determines visibility and actions. Role-specific landing views may prioritize exceptions and approvals while retaining the same canonical navigation.

# The product story in concrete work

A billing operator opens the October overview across cloud and on-premise sources. The page shows separately: confirmed costs by currency, missing or provisional inputs, settlement status and tasks requiring attention. It does not add incomparable currencies or treat a successful import as billing completion.

The operator follows a missing-data item to the relevant collection and validation detail, resolves the input problem and returns to the same October/organization scope. Successful receipt does not silently establish completeness. The operator opens costs and usage, checks the organizational allocation and follows an amount to its source evidence and policy revision.

Before downstream billing use, the applicable independent input-validation evidence and the exact revision-bound S06 input confirmation/publication eligibility must be satisfied. Successful receipt or source/storage reconciliation alone does not authorize billing use; B09 amount approval cannot waive input eligibility.

In settlement and billing, the operator prepares a run, examines exceptions and the previous/result difference, and requests the required approval. The approver sees the exact target, period, amounts and input/policy revisions; source validation or a role label cannot replace approval. A changed revision returns the work for revalidation.

After approval, an authorized operator issues the applicable internal statement or external invoice and submits the permitted ERP/file package. The overview separately shows document creation, ERP acceptance, delivery and collection outcomes. A generated document or successful submission is not proof of external acceptance or payment. Unsupported integrations remain explicit. Corrections link to the original document/run rather than inventing a second unrelated workflow.

# Storytelling acceptance criteria

Each screen must explain the current business target and scope, observed facts and blockers, the permitted action and expected result, and the evidence and status that support the next step. Back navigation preserves allowed period/organization filters and rechecks authority. An exception opens its exact business object. Context changes invalidate obsolete selections, previews and links.

Validate at least three complete journeys: normal monthly close; missing/late input and recalculation; approval rejection or ERP failure and recovery. Test internal-only and external-selling profiles separately. Navigation labels alone, a linear progress bar or a screenshot count cannot establish these outcomes. The chronological narrative is not a new global state machine: billing, issuance, posting, delivery and collection can follow distinct configured cadences and retain separate states.

# Review and decision status

Independent product/billing-boundary review initially required an explicit S06 input-eligibility prerequisite and projection/document-authority clarifications. Those corrections received targeted re-review PASS. A registry check confirms all 49 canonical IDs occur exactly once in the proposed mapping. No authenticated competitor UI or end-to-end product journey was tested. No owner approval, implementation change or commercial-readiness claim is made by this assessment.
