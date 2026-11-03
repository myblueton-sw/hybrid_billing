---
type: DomainReview
title: U01 finance and billing-operator fitness review
description: Records independent role-based findings on the current overview and confirms that its figures are generated fixtures rather than FOCUS-derived sample data.
status: draft
---

# Target and method

The owner requested independent finance and billing-operator review of the actual current screen, then asked whether its numbers came from FOCUS samples and absent fields were dummy values. The reviewed source is `13e3244ccbaf5008e4b785e71ff701a4170dd52f`, built revision `008bd182b66a`. Root integrated the report; two separate read-only AI reviewers supplied financial-controller and enterprise billing-operator assessments. Neither is a human practitioner or an owner acceptance record.

Both reviewers directly inspected current KO desktop/mobile captures under `output/validation/HB-85-revision`, including the first viewport, lifecycle matrix and selected detail. Finance also reviewed source/organization composition. They compared the implementation with the authorized U01 contract and relevant payment, ERP-cadence and operational planning. They edited nothing and ran no new runtime tests in this review. The application was kept unchanged while the role review ran. Previous visual/source/test outcomes are separate evidence.

# Verdict

Both independent role reviews returned **FAIL for the default enterprise integrated-billing home**. The screen is a useful read-only prototype with honest financial-state presentation, but it mainly answers which source costs and statuses exist. It does not yet organize the information needed to decide whether a period can close, what needs reconciliation or who must act next.

This is a product/role-fitness finding, not a claim that the existing numeric or layout checks failed. It does not supersede the owner's earlier rejection with a technical PASS, authorize a merge, or establish FIX.

# Data provenance: answer to the owner

| Displayed area | Actual current source | What is not established |
| --- | --- | --- |
| Source cost, currency totals and monthly bars | `fixtureCosts` generates deterministic example values in `model.ts:45–68`; the view aggregates these rows | No FOCUS sample file ingestion, selected version/dataset conformance or BilledCost/EffectiveCost mapping in this view |
| Closing-unit counts, blocked/pending/ERP states | `fixtureClosings` generates independent nine-stage example states in `model.ts:110–149` | No real closing register, assignment, approval or ERP result |
| Sales and organizational allocation | Explicit unavailable labels | No dummy monetary values or implied zeros |
| Receivables and overdue | Explicit unknown/unconnected state | No calculated balance, overdue amount or receipt authority |

`Screen.tsx:76–95` calls these fixture generators in synthetic mode; disconnected mode does not expose fixture financial results. The zero in organization C's on-premises example is deliberately recorded as zero, distinct from a missing record. It is not proof of an actual customer's zero charge.

The FOCUS question applies to this overview. This report does not claim every other repository component lacks FOCUS functionality. The planning HB-07 contract distinguishes incoming versus internal-target versions, cost meanings, mapping revisions, null/absent/zero and a separate SaleCharge model. Those mappings are not established by this generic fixture amount.

# Financial-controller review

| Priority | Finding | Required decision support |
| --- | --- | --- |
| P1 | Closing readiness is not the organizing information. Six units and overlapping attention counts do not show completed, exempt, held and unresolved targets. The matrix's `6 / 6` is filtered-list coverage, not closing completion. | Put an authoritative closing-control summary before analytical cost trends. Keep the fixed target denominator and distinct readiness states; do not manufacture overall completion from generic stage flags. Label list coverage explicitly. |
| P1 | Legal-entity/book, customer/contract and reporting/recognition/billing-period meaning are not visible. A synthetic snapshot cannot establish financial as-of authority. | Reserve a clear business-context strip and authoritative time/completeness basis. Missing bindings remain unknown instead of appearing as a verified enterprise aggregate. |
| P1 | A source-cost bar chart does not show the planned monthly-recognition versus customer-billing reconciliation flow. | Present independent recognition/posting, customer-billing and receipt lanes with relevant period linkage and reconciliation exceptions. Do not gate recognition on invoice issuance or invent accounting mappings. |
| P2 | The queue shows statuses but not the reasons, financial exposure, related closing-unit issues, owner, deadline or last confirmation needed for escalation. | Relate cases to their closing unit and show authoritative reason, affected meaning/currency and responsibility. Unknown remains unknown; no invented urgency thresholds. |
| P2 | The combined unavailable receivables cell does not reserve separate outstanding, due-today, overdue and receipt-unconfirmed decisions. | Use a distinct currency-scoped receivables region with confirmed/unknown classifications, payment-term basis and receipt freshness. Internal allocation must not acquire an external-payment obligation by implication. |

Evidence: U01 specification entry/completion, scope/field contracts and fixed-denominator acceptance; ERP-recognition-cadence and payment-terms planning; `WorkSummary.tsx`, `AmountSummary.tsx`, `Lifecycle.tsx`, the model and reviewed captures. The controller recommendation is **not to fix this composition as the final overview** yet.

# Billing-operator review

| Priority | Finding | Required operational support |
| --- | --- | --- |
| P1 | The first screen cannot answer whether this period can close. Readiness is subordinated to source-cost analysis. | Promote closing targets and distinct readiness/hold/exemption information, with independent financial outcomes. Unavailable counts must stay unavailable. |
| P1 | Inspect leads to fixture evidence but no exact permitted next business destination; the blocked row does not state its blocking reason. | Put the reason on the row and route to scoped closing/reconciliation/approval work where its contract exists. A missing integration must not become a fake resolution button. |
| P1 | Source/organization counts do not identify the complete customer/contract/legal-entity business scope. | Surface the business context and meaningful target/path identifiers; preserve canonical scope and destination permission checks. |
| P1 at enterprise scale | The category-ordered six-row queue has no work-type, responsibility or customer/contract search. An approver cannot efficiently isolate relevant work among thousands of other rows. | Provide queue-specific My work, Blocked, Approval and ERP views plus target search. Preserve aggregate financial scope and the closing denominator. Do not fabricate assignment. |
| P2 | A single blocked-unit row selects the first blocked stage, although the example has both collection and reconciliation blocked. | Group related issues by closing unit/case and expose all blocking reasons or an explicit multiple-stage expansion. Category order is a presentation order, not actual business priority. |

Evidence: U01/U02/S06 planning, `ScopeFilters.tsx`, `WorkSummary.tsx`, `Screen.tsx`, `model.ts` and reviewed first-screen/lifecycle/detail captures. The operator recommendation is **not to approve this as the default billing home** yet.

# What should be retained

Both roles supported separate financial meanings and currencies, labelled partial amounts, true-zero versus missing distinctions, honest unavailable states, independent lifecycle outcomes, readable target/path/stage rows and mobile work-first ordering. These are sound foundations, not evidence that the whole role workflow is complete. Do not repair the missing decision support by inventing finance values or real-user authority.

# Integrated design recommendation

The shared priority is business context and closing control before cost analysis. Proposed reading order:

1. Business scope, period meaning and authoritative as-of/completeness.
2. Fixed closing denominator and distinct readiness states, with independent recognition, billing and receipt lanes.
3. Separate cost, sales, allocation and receivable meanings with explicit availability.
4. Primary exception/assigned-work queue: reason, related target, stage, responsibility, deadline/freshness and a specific permitted next destination.
5. Monthly reconciliation and analytical trends, followed by detailed matrices and breakdowns.

This is a design recommendation, not an implemented screen revision. Keep the menu IA. Existing U01 item IDs remain; additions such as `U01:scope:business-context`, `U01:closing:readiness` and `U01:finance:independent-lanes` would be proposed items pending their contracts and design review. Extend existing `U01:work:attention` and `U01:state:receivables` rather than silently reusing them for different meanings.

# Official external cross-check

- [AWS Understanding your bill](https://docs.aws.amazon.com/us_en/awsaccountbilling/latest/aboutv2/getting-viewing-bill.html), consulted 2026-11-03: monthly scope, estimated versus final charges, consolidated account detail and invoices are distinct. This supports explicit basis and drilldown; AWS's provider-bill view is not a substitute for our complete operating model.
- [Salesforce Billing Analytics Dashboards](https://help.salesforce.com/s/articleView?id=ind.revenue_intelligence_billing_analytics_dashboards.htm&language=en_US&type=5), consulted 2026-11-03: overview and invoice operations include receivables, invoice health and bottlenecks. This supports role-oriented information, not copying every Salesforce metric.
- [FOCUS Billed Cost](https://focus.finops.org/docs/specification/v1-3/columns/billed-cost/) and [Effective Cost](https://focus.finops.org/docs/specification/v1-3/columns/effective-cost/), checked for the version-specific cost-basis distinction. The current generic fixture is not mapped to either.
- [Official FOCUS 1.4 release explanation](https://www.finops.org/insights/introducing-focus-1-4/), consulted 2026-11-03: Invoice Detail and Billing Period support invoice reconciliation and financial context. Do not claim FOCUS categorically excludes invoices or payment terms. Equally, do not infer this project's chosen dataset/version, mapping or operational ledger authority from the standard's existence.

Own planning remains authoritative. A future representative-data review must identify dataset version, source role, field lineage, amount meaning and period/currency basis. Provider invoice data cannot automatically stand in for this service's customer sales, receivables or approval/closing decisions. Missing non-source fields need clearly labelled synthetic operational scenarios or unavailable states; sample data is never live authority.

# Limits and next gate

No actual finance/controller or billing employee participated, and no satisfaction, legal/accounting policy or human acceptance is claimed. Human role priorities, legal-entity/book meanings, recognition/billing policies, receipt authority and real operational destinations remain to be validated. The next design iteration should address these ranked findings with traceable representative data. This review does not implement that iteration, connect FOCUS, alter the visible screen, acknowledge review, merge or deploy.

# Report verification

The billing reviewer read back this integrated report and returned PASS for reporting accuracy: its role-fitness FAIL was preserved, fixture provenance was correct, recommendations were not presented as implemented and no human acceptance was fabricated. It did not independently verify the external references or the finance reviewer's execution history. Root checked the unchanged source/build target, English document metadata and local links; no application regression run was needed for this documentation-only step.
