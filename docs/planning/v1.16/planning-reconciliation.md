---
type: Planning Reconciliation
title: Full baseline requirement and addition reconciliation
description: Separates established controls from unresolved decisions and documents exact source work screen and acceptance references for the full planning baseline.
status: draft
---

# Full baseline reconciliation

Tracking: [HB-14](https://linear.app/hybrid-billing/issue/HB-14). Baseline c4dbb41; accountable owner Seung Woo Park; root sole writer and PM/planner; machine mac_mini. This is an English reconciliation index, not a replacement for the detailed source or final planning approval.

The [machine-readable register](planning-reconciliation.json) preserves all 43 source groups and exact existing work/detail/acceptance identifiers, source block references, and screen-design JSON hashes. Source-local HB IDs are requirement IDs, not proof of Linear tickets. Reading a source and indexing its IDs does not certify implementation or semantic completeness. All groups remain NOT_STARTED / NOT_RUN.

## How to use the classifications

HB-16 adds a [normative remediation trace](remediation-trace.json) for the specifically changed contract/action/work/acceptance fields and publishes ten complete English screen specifications plus the common domain contract. Apply those explicit revisions alongside unchanged source clauses. Historical HB-14 hashes remain baseline evidence; current screen hashes and predecessor hashes are recorded separately. This is not full English publication or closure of all 43 detailed groups.

- **Established:** source-required controls and explicit owner directions. Do not ask whether to include them again.
- **Integration needed:** a contract exists, but full source/requirement/work/API/screen/acceptance reconciliation is incomplete.
- **Decision/evidence needed:** a concrete value, product contract, customer configuration or compatibility proof remains missing. Use the [decision and action register](planning-decisions.md) to distinguish these.
- **Proposed:** architecture mechanisms, exact interactions and additional mappings are reviewed design proposals, not owner-approved policy.

Every row has an established control and remaining work; these are independent dimensions, not a percentage-complete score. The summaries deliberately do not reproduce every source clause. All 35 detail IDs and full acceptance semantics remain obligations. In particular, opening-balance deduplication (QA-09), cumulative-versus-interval event semantics (RT-01), approved-only raw retention (AR-05), optional CSP AP submission (DB-03), and later product-Marketplace distribution (DB-14) retain their original applicability.

## Requirement-by-requirement result

| Source ID / group | Established control summary | Actual remaining decision or evidence |
| --- | --- | --- |
| HB-01 Operating entities and profiles | One core supports MSP, enterprise and group operations; installation, workspace, legal entity, organization and purchasing/paying/issuing parties remain distinct. | Actual parties, payment responsibility, tenancy conditions and sale/internal-allocation configuration; do not reselect or remove operating profiles. |
| HB-02 Organization and account history | Organization trees, viewing groups, payer relationships and contract/allocation pools are distinct; effective-time changes preserve approved historical attribution. | Hierarchy limits, shared-account precedence and missing mid-period history policy. |
| HB-03 Actions and delegation | Read, raw-cost access, policy edit, approval, issuance, download and delivery have distinct authority; policy ownership and author/approver separation are required. | Exact scoped role/field/action matrix, ceilings, approved delegation and exceptional segregation policy. |
| HB-04 Provider onboarding readiness | Authentication, operational read, first data, full coverage, FOCUS conformity, reconciliation and activation are separate evidence states. | Full provider/contract/account-role evidence matrix, actual samples and source access owners; verification order does not narrow scope. |
| HB-05 Connectors and capabilities | AWS, Azure, GCP, OCI, Alibaba, VMware, HawkEye and TokenMeter remain scoped; distinguish supplier originals from reseller/pro-forma sales data. | Per-profile invoice/export/API paths, quotas, complete export contracts and capability evidence. |
| HB-06 Idempotent receipt and immutable originals | Same identity/content is a retry; same identity/different content is a conflict; corrections are new revisions; equal amounts alone do not imply duplicates. Preserve only permitted billing evidence; AR-05 excludes original AI prompt/response bodies from raw/staging/log/support storage. | Stable source identities, upload/archive limits, delivery mode and revision/precedence per profile; storage selection. |
| HB-07 Normalization and monetary meaning | Source and internal FOCUS versions are distinct; billed/effective/list/contracted costs and sales are not interchangeable; null, absent and zero differ. | Internal target version/datasets, mapping coverage and numerical limits; compatibility evidence. |
| HB-08 Evidence and reproducibility | Approved source, mapping, attribution, policy, FX and engine/module versions plus results remain accessible; URLs and hashes alone are insufficient. | Class retention durations, key custody and remote immutability guarantees. |
| HB-09 Supplier and internal-cost reconciliation | Reconcile matching scope/currency/basis with explained differences; do not force EffectiveCost to monthly invoice totals or invent internal-cost invoices. | Per-basis required evidence and approved rounding components; retain zero unexplained difference. |
| HB-10 HawkEye integration | Accept eligible GPU power allocation and immutable statements; selling is this platform's responsibility; mock/incomplete/unmeasurable evidence cannot finalize billing. | Actual version, supersession identity, full export and authentication contract evidence. |
| HB-11 TokenMeter integration | Preserve input/output/cache, request/attempt/event and evidence-grade meaning; organizational AI cost differs from TokenMeter SaaS charges. | Full export, evidence-grade eligibility and exact fixed-rate input precision. |
| HB-12 On-prem cost pools | Customer-approved depreciation/lease, power, facility, license and operations costs remain distinct from allocation and sales; no double-counted purchase and depreciation. | Cost-pool components/boundaries, idle responsibility and approved customer policies. |
| HB-13 VMware measurement history | Period charges use historical configuration and observations; current VM size/CPU utilization cannot fabricate past monthly charges. | vCenter/counter history evidence, metering units and allocated-versus-consumed policy. |
| HB-14 Organization allocation and shared pools | Per currency, allocated amounts plus operator responsibility plus explicit residual equal the pool; showback, simulation and final chargeback differ. | Full-pool denominators, residual recipient, boundaries and ordering policy. |
| HB-15 Contract periods and lifecycle | Start, pause, renewal, rate changes, termination and final settlement are distinct; contract end does not cancel supplier subscriptions or delete resources. | Proration, period transitions, termination remainder/grace and customer-specific terms/cadences. |
| HB-16 Selling rates | Pass-through, markup, target margin, unit/fixed/tier/allowance/overage/minimum/maximum modes differ; supplier commitment engines are not rebuilt. | Exact precision/rounding, discount stacking, component policies and approved sequence exceptions; default sequence already exists. |
| HB-17 Formula and policy changes | Settings and restricted formulas are supported without arbitrary code/network access; draft, static checks, full impact, approval and activation remain separate. | Allowed DSL operators/variables/aggregation and separately governed code-extension conditions. |
| HB-18 FX and rounding | Original, sales, display and ERP accounting FX remain distinct; fixed/date/month-average policies preserve quotation units and calendar meaning. | Actual provider/date/average method, currency scales and correction FX; source AR-06 precedence already exists. |
| HB-19 Supplier commitment results | Consume supplied commitment benefits/unused/offset results; do not add purchase optimization or recalculate supplier amortization. | Actual sample evidence, unused responsibility, benefit pass-through and negative-savings policy. |
| HB-20 Marketplace transaction roles | Buyer charges, seller fees, settlement receipts and commitment consumption are different transactions; do not repeat source Private Offer discounts. | Applicable role/channel evidence, offer/fee terms and settlement permissions. |
| HB-21 Billing Claims and uniqueness | Receipt identity, API idempotency and economic billing identity differ; new clients/keys/revisions cannot duplicate the same contract/period/component/scope. | Economic Claim keys, partial consumption units, reservation expiry and correction application semantics. |
| HB-22 Prepayments credits and disputes | Discounts, allowances, cash prepayments, credit documents and refunds differ; one cash authority; disputes do not automatically cancel issued obligations. | Cash/receivable authority, external reservation support, promotional expiry and dispute SLA/hold policy. |
| HB-23 Approval and execution | Billing approval, ERP final-amount approval, issuance and delivery differ; explicit human approval and segregation remain; affected revisions require renewed approval. | Actual approvers/ceilings and trusted external approval; additional input-confirmation role/interaction details. |
| HB-24 ERP submission ledger | ERP issuance mode assigns final numbering/tax/accounting documents to ERP; API and file paths share issuance responsibility and submission tracking. | External identity, draft/post guards, tax/master authority and recognition mapping/capabilities. |
| HB-25 ERP profile validation | Odoo vertical validation is a proposal; SAP product/version families have distinct adapters, not one generic API. | Product/version/deployment/module/sandbox evidence across intended profiles; safe automatic posting support. |
| HB-26 XLSX and manual handoff | Standard internal/customer XLSX and controlled result registration are common requirements; download does not prove external application. | Per-ERP upload formats, file limits and accountable manual application evidence. |
| HB-27 Disclosure and export | Raw MSP, internal and customer files differ; generate customer output from semantic allowlists and exclude unclassified new fields. | Approved fields/recipients, inference conflicts, link policy and per-profile detail resolution. |
| HB-28 Own issuance and internal documents | Own issuance is new product work; PDF generation is not legal-document support; ERP/own/internal numbering and responsibility differ. | Document/jurisdiction profiles, tax/number policy, templates and governed payment-detail approval. |
| HB-29 Corrections and delivery | Issued originals are immutable; late/retroactive changes use linked difference documents; issuance success and delivery failure differ. | Correction types, cumulative credit ceiling, channel lookup and deduplication capabilities. |
| HB-30 Admin overview and history | Show scope-specific stage/deadline/owner/next action and evidence; do not add costs, sales and allocations as one total. | Saved-filter sharing/search scope and presentation ordering; retain all required screens. |
| HB-31 Closing schedules and notifications | Collection, verification, approval, issuance, delivery and payment deadlines differ; due time is not an issuance command; read/accepted/resolved differ. | Calendars, business hours, channels, escalation and pre-send current-state checks. |
| HB-32 AI Assistant entry point | Use the same authorized business APIs for query/draft/simulation/approval requests and eligible execution; LLM is not calculator or approver. | Model connection/data egress/conversation retention/tool exposure; tenant training requires its own contract. |
| HB-33 Common commands and Open API | UI, CLI, AI, external clients and plugins share one command/permission registry; API availability is not anonymous Internet access. | Canonical command/permission IDs, compatibility/deprecation policy and SDK languages. |
| HB-34 Jobs and event delivery | Technical job states differ from business states; accepted/unknown is not complete; duplicate/out-of-order events are expected. | Leases, retry/retention/cancellation contracts and queue-lag targets. |
| HB-35 Plugins and extensions | Versioned extension contracts/examples are required; core owns originals/calculation approval/ledger; no direct external DB write or admin DOM execution. | Package signer/trust, remote endpoint approval and version compatibility. |
| HB-36 Self-hosted installation and support | Customer-operated independent installation is required; VM/container and conditional Kubernetes assessments do not certify either environment. | Resolve source VM-first validation wording versus conditional Kubernetes preference; actual OS/CPU/network/storage/runtime/support evidence. |
| HB-37 Recovery migration and end of retention | Recover database/evidence/keys/policies/modules/numbers/submissions/outbox coherently; fence old issuers and reconcile external uncertainty before writes. | RPO/RTO, key recovery, external lookup retention, migration rights and customer operating evidence. |
| HB-38 Simulator and independent oracles | Use the production core after development approval for synthetic cases and independent expected answers; partial public months are not invoice truth. | Pinned sample/license evidence, expected-answer approvers and performance environment. |
| HB-39 Scale and performance budgets | Design for more than one million incoming records/day and at least thirty million/month; one million is not a cap; individual bulk tests start at one hundred thousand. | Record definition, peaks, retention/skew/concurrency, closing windows and hardware/SLO/RPO/RTO. |
| HB-40 Reuse licensing and commercial rights | Compare a small core and OSS by accuracy/isolation/operations/rights; product source-available commercial terms cannot override third-party rights. Expired entitlement preserves retained evidence access; selling this product on Marketplace remains later/conditional under DB-14. | Language/components/pinned versions, final rights-holder/license terms and SDK license; legal approval remains separate. |
| HB-41 Release trace and activation | Connect requirements/source/owner/API/screens/tests/evidence/blocking status; design, implementation, tested and customer-validated are separate. | Named acceptance responsibilities, staffing/dates and actual activation evidence; no arbitrary design exclusions. |
| HB-42 Identity and access lifecycle | Manage creation/invitation/activation/grant/change/suspension/withdrawal/revocation; login users differ from collection accounts. | IdP/auth/LDAP modes, login ID scope, MFA, self-registration, recovery and external collaborator policy. |
| HB-43 Retention archive and deletion | Manage by data class/workspace/contract/evidence dependencies; retention, holds, physical deletion and backup expiry differ. | Basis dates/durations/hold owners, backup expiry/deletion targets, privacy minimization and key/shared scope. |

## Owner additions and overlap

RA-01–14 are existing discussion bundles, not fourteen independent new features. ADD references below are editorial aliases for previously recorded additions. These proposed overlaps do not replace source mappings or increase the 43-group count. The optional phrase has no inferred mapping.

| Addition | Proposed overlap with source groups | Current evidence |
| --- | --- | --- |
| RA-01 Self-hosted delivery | HB-01, HB-36, HB-37 | [Contract](../../../docs/architecture/self-hosted-installation-assessment.md) |
| RA-02 Technology suitability assessment | HB-07, HB-08, HB-34, HB-36, HB-39, HB-40 | [Contract](../../../docs/architecture/technology-assessment.md) |
| RA-03 Billing data modeling and storage | HB-06, HB-08, HB-21, HB-24, HB-37, HB-43 | [Contract](../../../docs/architecture/data-ownership/billing-data-model.md) |
| RA-04 Sessions tokens and authentication | HB-03, HB-42 | [Contract](../../../docs/architecture/security-boundaries/identity-session-permissions.md) |
| RA-05 Admin and customer information management | HB-01, HB-02, HB-03, HB-30, HB-42 | [Contract](../../../docs/architecture/admin-design.md) |
| RA-06 MSP customer tenant separation | HB-01, HB-02, HB-03, HB-27, HB-42 | [Contract](../../../docs/architecture/data-ownership/billing-data-model.md) |
| RA-07 Organization management | HB-02, HB-14, HB-15, HB-23 | [Contract](../../../docs/architecture/admin-design.md) |
| RA-08 Scoped permissions and data views | HB-03, HB-27, HB-30, HB-33, HB-42 | [Contract](../../../docs/contracts/security/access-and-trust.md) |
| RA-09 API upload and other collection methods | HB-04, HB-05, HB-06, HB-07 | [Contract](../../../docs/architecture/data-collection.md) |
| RA-10 Provider and other grouping axes | HB-01, HB-02, HB-05, HB-06, HB-07 | [Contract](../../../docs/architecture/data-collection.md) |
| RA-11 Validation correction adjustment and prediction intelligence | HB-09, HB-17, HB-22, HB-29, HB-30, HB-32, HB-35, HB-38 | [Contract](../../../docs/architecture/intelligence-layer.md) |
| RA-12 Complete authorized cost and usage detail | HB-07, HB-08, HB-09, HB-16, HB-27, HB-30 | [Contract](../../../docs/architecture/usage-detail-lifecycle.md) |
| RA-13 Durable information maintenance | HB-08, HB-27, HB-37, HB-43 | [Contract](../../../docs/architecture/usage-detail-lifecycle.md) |
| RA-14 Definitive pricing and final charges | HB-09, HB-16, HB-17, HB-18, HB-21, HB-23, HB-24 | [Contract](../../../docs/contracts/billing/inputs-pricing-and-erp.md) |
| ADD-PT Contract payment terms and overdue | HB-15, HB-22, HB-24, HB-27, HB-28, HB-29, HB-30, HB-31 | [Contract](../../../docs/contracts/billing/payment-and-recognition.md) |
| ADD-ERPC Monthly ERP recognition independent of billing | HB-15, HB-21, HB-23, HB-24, HB-25, HB-26, HB-28, HB-29, HB-30, HB-31 | [Contract](../../../docs/contracts/billing/payment-and-recognition.md) |
| ADD-SCALE Sustained scale and independent user axis | HB-30, HB-39, HB-42 | [Contract](../../../docs/planning/v1.16/scale-validation.md) |
| ADD-H13 Full scope accuracy cross-validation and operator confirmation | HB-04, HB-05, HB-06, HB-07, HB-09, HB-23, HB-38, HB-41 | [Contract](../../../docs/architecture/data-collection.md) |
| ADD-OPTIONAL Interrupted optional-capability statement |  | [Contract](../../../docs/planning/recent-owner-requirements.md) |

## Integration delivered and remaining

This ticket publishes the English [payment and monthly recognition contract](../../contracts/billing/payment-and-recognition.md), carrying the existing PT-01–08 and ERPC-01–08 acceptance obligations without choosing new accounting policy. Existing calculation order, FX precedence and common XLSX/manual result support are treated as already specified, not fresh owner questions.

The index reads existing 49 screen design summaries for references; detailed spec.md remains the field/action authority. It does not certify all screen details. Remaining synchronization includes S03/S06/B09 input confirmation; M02 trust-root epoch; S02 onboarding metrics; U01 favorites; full detail/disclosure/retention and financial profile choices. Source design.json links are explicitly local/unpublished, not portable published API contracts.

The installation source says VM/container validation first, while HB-12 conditionally prefers existing operated Kubernetes. Neither is a final adoption decision. The conflict is explicit in the decision register instead of silently rewriting the original.

JEV recommendation for this execution: no selected skill and uncertain review depth. Root selected planner/technical-writing and independent architecture/DBA review from the actual scope. No JEV response authorizes policy or execution. See [verification](../../reviews/HB-14-reconciliation/verification.md).
