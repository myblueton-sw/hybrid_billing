---
type: Comparative Research
title: Competitor workflow evidence for Hybrid Billing screen planning
description: Compares nine official product references and translates observed patterns into proposed Hybrid Billing user journeys without changing financial or permission contracts.
status: draft
---

# Screen planning comparison

Research date: 2026-10-06. Tracking: [HB-17](https://linear.app/hybrid-billing/issue/ad8c4443-9c7f-479c-ab80-5c49a45e7066). Accountable owner: Seung Woo Park. Root owns research and writing; independent evidence/UX review is recorded in [verification](verification.md). The user prioritizes screen planning. This is comparative research and a proposal, not acceptance of a new product scope, implementation or planning completion.

## Conclusion and scope

**Research inference:** combine the operational customer/rate/reprocessing journeys of MSP billing products, the explorable cost views of FinOps products, and the explicit document lifecycle of invoicing products. Hybrid Billing should make the operator's next justified action visible throughout collection, validation, reconciliation, confirmation, billing and delivery. A spend chart alone cannot express that task.

All AWS, Azure, GCP, OCI, Alibaba, VMware, HawkEye and TokenMeter requirements remain. The nine references below are a research sample, not a reduced provider scope or exhaustive market ranking. No single reviewed source establishes complete coverage of our original preservation, missing/duplicate inspection, mandatory cross-check, human confirmation, revision, ERP and self-hosted requirements.

The local authorized v1.16 `ui-design.md` contains 49 canonical screen IDs. HB-16 at `c161dc5a2b4b2a1321a36c2db673c074b0dd6bac` supplies ten proposed English screen pairs and guarded interactions in [PR #12](https://github.com/myblueton-sw/hybrid_billing/pull/12). It remains unmerged at research time. This report does not publish or replace that baseline. The retained 43-group source index is [planning reconciliation](../../planning/v1.16/planning-reconciliation.md).

## Evidence method

- **D:** official documentation read; describes behavior but does not prove current deployed execution.
- **V:** official published product screenshot visually inspected; static example, not a hands-on product test.
- **M:** vendor article/marketing claim; useful direction, requires demonstration.
- **S:** official search-index content; direct fetch failed, so weaker than D.
- **P:** our proposed adaptation/inference; not a competitor claim or accepted decision.

No logged-in vendor sandbox, customer data, sales contact, purchase, benchmark, accessibility test or end-to-end transaction was used. Vendor images were viewed, not copied into Git. Research paraphrases are bounded; external rights are unchanged. Dates below distinguish a dated announcement from an undated page accessed today.

## Product comparison

| Reference and relationship | Official evidence and screen/workflow pattern | Proposed use in Hybrid Billing | Boundary or mismatch |
| --- | --- | --- | --- |
| IBM Cloudability MSP / Commercial Billing — close MSP reference | S: price-book rules have order and first-match behavior. S: June 30, 2026 release describes processing/freshness tracking; August 10 describes management dimensions in native reporting. [Rules](https://www.ibm.com/docs/en/cloudability-commercial/cloudability-msp/saas?topic=pricing-cloudability-msp-editing-rules), [release notes](https://www.ibm.com/docs/en/cloudability-commercial/cloudability-msp/saas?topic=billing-whats-new-in-cloudability-msp) | B04/B08: show rule order, matched rule and before/after effect. S03: expose processing timestamps and freshness beside customer scope | IBM direct documentation fetch failed, including 403; indexed evidence only. Do not adopt first-match arithmetic or claim current UI layout. Independent bill confirmation was not established |
| Flexera One CCO — close MSP reference | D: Billing Plans organize customer adjustments; placement before/after billing rules matters. Billing History separates recalculation with existing data from unlocking to ingest new data. [Billing plans](https://docs.flexera.com/flexera-one/partners/cloud-cost-optimization/billing-plans) | B04/B08: rule order and impact preview; separate “Recalculate current input” from “Collect updated input and review” | Our immutable originals, new revisions and approvals still apply; a vendor unlock action is not permission to mutate an approved result |
| CloudHealth — partner billing reference | D/API: customer-specific or global AWS/Azure adjustments, with documented Partner → Partner Billing → Billing Rules navigation. [Partner billing reference](https://apidocs.cloudhealthtech.com/#partner-billing-rules) | O03/B04: customer portfolio → contract → applicable pricing; show inherited versus overridden rules | API navigation evidence only, not visual UX or full provider coverage. Do not expose API credentials as customer selectors |
| Finout — cost exploration reference | D+V: saved views, compact filters/time/cost-type controls, graph and table, ordered grouping. Documentation identifies top-value aggregation into Others and bounded CSV detail. [MegaBill](https://docs.finout.io/user-guide/inform/megabill) | R01/S04: one persistent query context, chart-to-table drilldown and saved views; distinguish summary export from complete evidence export | Analysis aggregation is not full billing evidence. Do not inherit numeric display/export caps, public-sharing defaults, or an AI filter as an approval mechanism |
| Cloudforet / SpaceONE — multi-workspace operations reference | D+V: admin versus workspace context, data-source/account mapping, recent collection results. Published Data Sources list shows linked-account and workspace counts. [Admin guide](https://cloudforet.io/docs/guides/admin-mode/_print/) | S01/S02/S03/O01: source list → collection evidence → account/ownership mapping; keep active workspace conspicuous | Asset-account support is not proof of bill completeness or invoice support. Our separate sensitive command rights remain required |
| OpsNow FinOps Plus — regional FinOps reference | M: Analytics offers period/tag analysis; Invoice tab explains usage charges, discounts, adjustments and final billed amount. Article dated 2025-11-12. [Usage-charge analysis](https://www.opsnow.com/en/post/best-way-analyze-usage-charges) | S06/B08: explain the monetary bridge from usage to adjustments and invoice amount, then let users inspect source rows | No live UI or reconciliation/approval enforcement verified; “accurate” marketing language is not an independent accuracy result |
| Stripe Invoicing — invoice operations analog | D: invoice list uses tabs/chips, configurable columns and exports; detail actions depend on status. Status and operational badges can differ. [Manage invoices](https://docs.stripe.com/invoicing/dashboard/manage-invoices), [lifecycle](https://docs.stripe.com/invoicing/overview) | I01/B11/C02: searchable document register, status-specific actions, explicit pending/overdue explanations and original/correction links | Not a source-cost collection system. Do not inherit automatic collection, legal invoice treatment or payment-authority assumptions |
| Lago — metered-billing analog | D: draft invoices provide a review window; fee amount/units can be adjusted and finalization is manual or automatic at grace expiry. [Draft invoices](https://getlago.com/docs/guide/invoicing/draft-invoices). Historical article distinguishes invoice and payment grace periods. [2023 explanation](https://getlago.com/blog/grace-period-to-adjust-invoice-usage) | B08/B09/U02: visibly separate draft, reviewed and issued; show which late inputs changed the draft and why | Automatic expiry cannot replace our mandatory human confirmation. Direct amount overwrite is not our immutable-evidence correction model. A review deadline is not an invoice payment extension |
| AWS Billing Conductor — native pricing/showback analog | D: alternate pro forma billing is separated from the standard payable bill. Dashboard exposes the impact of custom pricing. [Pro forma meaning](https://docs.aws.amazon.com/billingconductor/latest/userguide/understanding-proforma.html), [dashboard](https://docs.aws.amazon.com/billingconductor/latest/userguide/understanding-abc.html) | U01/B08/R01: give source cost, internal cost, selling amount and allocation explicit labels and scope; explain differences | AWS-specific reference; pro forma amounts are not proof of payable invoice or ERP posting. Do not translate all monetary concepts into a single “cost” card |

## Provider and evidence limits

Cloudability's indexed organization guide says customer assignment covers AWS/Azure/GCP/OCI, while other listed costs remain management-level. This is a limit on that documented path, not a claim about every IBM product. [Organization key](https://www.ibm.com/docs/en/cloudability-commercial/cloudability-msp/saas?topic=billing-setting-up-org-structure-key)

Cloudforet's getting-started guide links service-account setup for AWS/Azure/GCP/OCI/Alibaba. That establishes documented onboarding entry points, not invoice completeness, actual adapter compatibility or identical workflows. [Getting started](https://cloudforet.io/docs/guides/getting-started/)

| Our input family | Evidence relevance in this sample | Remaining screen-design evidence |
| --- | --- | --- |
| AWS / Azure / GCP | Several MSP/FinOps references | Verify each account/contract mode and export/late-data failure path, not merely provider logos |
| OCI | Cloudability indexed account assignment; Cloudforet setup links | Exact bill-detail, unsupported fields, invoice availability and refresh behavior |
| Alibaba | Cloudforet setup link | Financial collection, invoice/reconciliation and permission failures remain unverified |
| VMware / on-premises | No adequate rate-card/cost-driver screen evidence collected in this pass; focused Broadcom search returned no results | B01/B02/B03 must preserve depreciation/license/facility, time-varying assets, missing measurement and allocation evidence; obtain an on-prem billing demo in follow-up |
| HawkEye / TokenMeter | Existing internal planning, not external competitor certification | Retain GPU/time and token/model/unit semantics, absent measurements and precise billing provenance; test with original synthetic scenarios later |

## What was actually seen

| Visual observation | Concrete visible arrangement | Limit |
| --- | --- | --- |
| Finout MegaBill documentation, “Using MegaBill” image | Saved-view actions above a compact filter/time/data-type/cost-type row; summary measures, grouping control and stacked chart underneath | Official static screenshot viewed in browser. No responsive behavior, interaction correctness or latest deployed version tested |
| Cloudforet Data Sources example | Searchable table with Name, ID, Linked Account, Workspace and Registered Time; a partial account count carries an alert marker | [Published image](https://cloudforet.io/guides/admin/data-sources/datasources-details-01-en.png) viewed in browser. It is a list, not proof of the unseen collection-detail workflow |

Lago's documentation image fetch failed; its workflow is D only. IBM direct fetch failed; it is S only. These failures are not successful visual inspections. Finout live documentation contained updated table-cap wording beyond the search extract; recommendations intentionally do not pin vendor-specific caps. Stripe's lifecycle overview contains inconsistent statements about whether paid is terminal; that transition is not used as a design rule here.

## Screen planning proposals

Every item below is **P: root research synthesis**, not a JEV product recommendation. JEV recommended considering the UI/UX skill (confidence 0.59) and standard review (0.47); root chose evidence-based document review. No design recommendation below is approved by JEV.

### Navigation and landing

Retain all 49 canonical screens but organize discovery around a role's job. Proposed menu groups: **Work overview; Customers and contracts; Sources and verification; Pricing and billing; Documents and delivery; Analysis and audit; Administration**. This is a navigation hypothesis, not seven replacement screens. A customer portal is a separate permitted experience. Provider is a filter and source-setup choice, not a competing full navigation tree for every workflow.

U01 should lead with work requiring a decision: blocked collection, unexplained differences, awaiting confirmation, approval requests, external result unknown and deadlines. Monetary cards follow with explicit meaning/currency/as-of. Each count links to the same authorized scope in a work list. Saved views and favorites help return to work; neither grants access. The hypothesis must be tested against both daily operations and executive analysis needs.

### Six connected workspaces to storyboard

| Proposed workspace and preserved screens | User, trigger and task | Layout and primary interaction | Success and failure to demonstrate |
| --- | --- | --- | --- |
| Closing work list: U01/U02/U03/U04 | Billing operator starts a closing cycle | Scope/date strip; step counts; rows by customer/contract/currency with next action, owner and deadline. Open a detail drawer, keep list filters | User identifies the actual blocker and responsible role. An unknown ERP result points to status verification, never blind reissue |
| Collection and input review: S01/S02/S03/S04/S05, handoff to S06 | Operator sees incomplete or changed data | Source/account/period list; completeness panel; original versus normalized detail; linked missing/duplicate cases. Inspect in S03 and preserve history; hand off to S06 for explicit confirmation or rejection | Connected, collected and eligible are distinct. Changed evidence invalidates pending confirmation; normal zero is distinct from missing |
| Reconciliation desk: S06 with B06/B07/B11 | Billing analyst investigates a difference | Expected/actual/difference by same basis and currency; side-by-side source and calculated lines; explanation/evidence panel; owner and next action | Every difference has evidence or remains blocked. A cost anomaly does not itself authorize a financial adjustment |
| Pricing and comparison: O03/B01–B08 | Pricing owner changes a contract rule or input revision | Version/effective-date header; ordered rule list; affected customers and line-level reason; previous/proposed comparison. Separate data refresh from recalculation | User can explain which rule changed which amount. Simulation is not approval; historical confirmed inputs remain immutable |
| Review, issue and handoff: B09/B10/I01–I04 | Approver or billing operator is ready to act | Exact target/version, evidence and before/after figures; explicit review action; downstream issue/ERP/delivery steps shown independently | Input confirmation, monetary approval, issuance, ERP posting and delivery are separate. Partial results retain per-target state and safe resume |
| Customer view: C01/C02/C03, linked B11 | Customer opens own bill or raises a question | Period/amount meaning, available documents and usage; correction relationship; question attached to a document/line with public response history | Customer understands the charge without internal cost leakage. A question does not cancel a bill or expose internal notes |

### Concrete interaction changes to explore

1. **Persistent context strip:** customer/legal entity/contract, period, currency, amount basis, input revision and as-of. Carry context from list to detail; visibly revalidate when it becomes stale.
2. **Progress as separate steps:** collection, verification, confirmation, calculation, approval, issue, ERP and delivery. A short summary can identify the main blocker without hiding other states. Payment/recognition remain independent where applicable.
3. **Evidence beside the decision:** expandable raw reference and normalized interpretation; mandatory cross-check result; who confirmed which revision. Keep technical hashes in details, with human-readable version and timestamp on the main surface.
4. **Exception-centered tables:** reason, affected scope, owner, latest evidence, last attempt and next allowed action. Retain keyboard operation and text statuses. Do not reduce exceptions to a colored badge.
5. **Different verbs for different effects:** collect updated input, recalculate, request approval, issue, recheck ERP status, resend delivery. Show affected scope and reversibility before the consequential action.
6. **Version comparison before replacement:** affected rows/rules, totals by meaning/currency and upstream change reason. Old confirmed results remain accessible; new results do not silently overwrite them.
7. **Export purpose first:** distinguish analysis summary, complete authorized detail, customer disclosure package and ERP import file. Show snapshot/coverage and exclusion rules; an on-screen top-N table is not the complete export.
8. **Customer preview in the disclosure workflow:** explicitly display the permitted customer package and omitted internal-only fields before delivery. This is not impersonation or a permission bypass.

## Remaining screen coverage

The six storyboards are cross-screen journeys, not a new first-release scope. O01/O02 organization and effective-dated relationship changes feed their context. R01/R02 remain analysis/audit destinations. A01 remains the required planned assisted-entry capability; users may choose it to access the same authorized actions, never a separate financial authority. L01–L03, M01–M05 and P01–P07 retain authentication, recovery, role preview, installation, plugins, retention, release evidence, demo and licensing purposes. All original detailed clauses remain authoritative; this report does not claim a 49-screen semantic re-audit.

The next design artifact should be a linked storyboard with normal, missing, partial, blocked, stale and external-unknown states for each journey, using linked static storyboards and non-executing design wireframes. Executable prototypes and HTML applications are deferred until explicit development authorization. Validate that users can locate a blocker, explain a charge, compare revisions, confirm the correct input, handle uncertain ERP outcomes and see exactly what a customer receives. Measure task success, mistaken actions and missing context; do not invent thresholds or claim a test that has not run.

Before final design acceptance, obtain stronger visual evidence for on-premises cost drivers, MSP customer portal disclosure, independent approval, and ERP uncertainty/correction. Public marketing absence is not evidence that competitors lack those functions. The output of this pass is a grounded design direction and comparison, not a declaration that Hybrid Billing is superior.
