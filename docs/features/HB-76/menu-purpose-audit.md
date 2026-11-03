---
type: DesignReview
title: Menu purpose, naming and hierarchy audit
description: Audits the eight-group implementation for over-promoted capabilities, ambiguous labels and object-level tasks that should not become independent menu entries.
status: draft
---

# Scope and judgment

HB-76. Owner-requested review, 2026-11-03. Root owns the report. This examines the actual service-ia.ts mapping and authorized screen specifications, not hypothetical navigation. No product-code change is included. The earlier eight-menu implementation is the reviewed baseline; its earlier mechanical coverage and shell checks do not establish optimal menu depth or naming.

The substantive error is treating a requirement's independent financial or permission semantics as proof that it deserves a separate top-level menu. Monthly recognition, billing approval, delivery controls and source evidence can remain independently governed while appearing as object tabs, workbench views or contextual actions. Conversely, operational work such as recurring input exceptions cannot disappear into connector Settings simply to reduce menu count.

No evidence establishes that a particular owner request was unnecessary. The unsupported inference was the assistant's choice to promote it into navigation. Navigation must expose the intended task without claiming unsupported capabilities.

# Classification rule

A primary destination needs a distinct recurring user goal, recognizable business object or work queue, and a practical reason to enter independently. A secondary workbench can be entered directly across many objects. An object tab preserves the selected connection, contract, run or document. Actions change that object; status and filters describe it. Configuration controls future behavior. Placement does not waive any authorization, evidence or approval contract.

Canonical routes and screen IDs are not the same as visible menu items. Preserve deep links and scope checks even when an owning surface becomes a contextual entry. Multiple entry points may open one canonical object; they must not create duplicate state or approval evidence.

# Concrete findings

| Current destination | Evidence and issue | Recommended treatment |
| --- | --- | --- |
| Overview / Overall overview (U01) | A group and its first item repeat the same idea. | A direct Overview entry can own summary sections; avoid an unnecessary overview-of-overview level. |
| Closing calendar (U02) | Spec manages deadlines, blockers, exemptions and reopening, not just a calendar widget. | Name Closing management or Period close; calendar is a view. Locate with settlement work, while surfacing deadlines on overview. Preserve independent recognition/billing deadlines. |
| My work and notifications (U03), header Notifications and Jobs | Current overlapping names hide distinct assigned work, events and execution jobs. | One My work entry for assigned tasks/events, with meaningful tabs/filters; retain Background jobs only for execution progress. Do not equate read, accepted and resolved. |
| Items requiring attention (U04) | Too generic: it could describe every screen error. Spec covers cost changes, missing benefits, conflicts and missing measurements. | Name Cost and billing issues, with an overview summary and a dedicated scoped issue list. Do not conflate findings with system alerts. |
| Costs and usage + B01/B02 | Observed expenses and editable internal cost construction are mixed. | Keep consolidated detail prominent; distinguish Internal cost construction and Assets/measurement as secondary workbenches rather than presenting configured cost as observed supplier cost. |
| B06/B07 under costs | Commitment usage/offset and marketplace buyer/seller transactions are basis-specific evidence, not general optimization. | Preserve permitted independent workbenches or filtered detail views where contracts require them; no invented commitment optimizer or generic Marketplace shop. |
| Organizations and contracts | Legitimate business context, but organization changes and handoff actions are not separate peer objects. | Retain Organization directory and Contracts; move relationship-change plans/history into the organization workflow with an independent authorized review queue if needed. |
| Allocation and pricing | Internal burden and sales pricing have different owners and lifecycle. The conjunction alone does not create one workbench. | Cost allocation belongs with cost/organizational responsibility. Selling policies and effective versions belong with contracts/pricing management. Shared policy libraries may remain directly accessible to policy owners; do not bury them in a single-contract form. |
| Internal allocation policy (B03) | Spec explicitly allocates by organization/contract, including cost or sales amounts; current label narrows the contract. | Use Allocation policies; identify amount basis and ownership inside the workbench. Keep Showback, simulation and confirmed Chargeback distinct. |
| Exchange rates and rounding (B05) | A cross-contract governed policy library, also used by pricing and ERP. | Place under policy configuration with contextual links. Preserve source/display/sales/accounting FX distinctions and approval. A filter cannot replace the policy editor. |
| Settlement and billing | A valid recurring domain, but run, approval, application ledger, document and delivery are presented as a flat set. | Use workbenches by business object and stage; expose run comparison, validation and original-linked correction as object views/actions. Preserve independent input eligibility and approval. |
| Accounting and ERP | I02/I03 provide controlled submissions and external-state reconciliation, not a general ledger/accounting product. | Remove the unsupported broad Accounting claim. Prefer an ERP processing or External processing workbench within settlement, directly reachable for monthly-recognition roles independent of customer invoicing. Promote to its own primary only if observed workloads justify it. |
| Monthly recognition, ERP submission and reconciliation (I02) | Long list of capabilities; includes an owner-added requirement as part of a title rather than organizing one submission object. | A stable ERP processing destination; separate Monthly recognition and Billing submission views by purpose. Draft comparison, external events and line matching belong within a selected submission. Posting outcomes remain separate. |
| File handoff and confirmation (I03) | API and file are transports on the same submission ledger, not different financial workflows. | Offer file handoff as a supported submission path/view, retaining standard internal/customer-file purposes that are not ERP submissions. Keep recipient, package, result ingestion and safe retry discoverable. A download never becomes confirmation. |
| Data and connections | Combines low-frequency configuration with daily intake and cross-domain reconciliation. | Separate Source connection management from recurring Data intake/validation inside the cost-data area. The operational queue remains directly reachable; diagnostics and credentials are connection context. |
| Connection onboarding and owner handoff (S02) | These are a guided setup process and task-limited action for a connection. | Enter from Add connection or Resume setup; retain resumable canonical route. Do not force a permanent peer item for every setup step or handoff command. |
| Intake, errors and completeness (S03) | Object, exception and property are concatenated in the label. | Data intake as the workbench; jobs, quarantine/conflicts and completeness are its views. Maintain eligible-input publication prerequisites. |
| Source evidence (S05) | Evidence usually answers why a selected row/amount is valid. A global menu without an object loses context. | Make evidence a contextual viewer from row/run/reconciliation/document; preserve an authorized evidence search for auditors when required. Do not delete its route or disclosure checks. |
| Source reconciliation and input confirmation (S06) | Spec tabs include source, sales, allocation, ERP and cutover reconciliation. The current label and source-only placement underrepresent it. | Reconciliation workbench with explicit types, entered from the relevant cost/run/submission. Input confirmation remains a source-specific governed action, not an alias for all reconciliation. |
| Credit and billing inquiries (B11) | Balance ledger and case/dispute lifecycle differ; payment-term additions are hidden behind a credit label. | Provide explicit Balance/receivable and Inquiry/dispute views with object links. Preserve one authoritative cash ledger and separate approval actions. Do not invent a new payment-processing product or claim all these views already exist. |
| Disclosure policy and delivery packages (I04) | Governed policy configuration and day-to-day delivery execution share one title. | Put disclosure configuration in authorized policy settings and contextual delivery preparation/history with the document or submission. Keep a directly accessible delivery queue if operators need cross-object work. |
| Reports and audit (R01/R02) | Business reporting and actual action history have different goals and visibility. | Reports remains a business destination. Object history stays local; an authorized cross-object Activity log must remain a discoverable cross-cutting audit utility for business auditors as well as relevant administrators; no administrator-only placement or permission assumption. Do not reduce business/financial audit history to login logs. |
| Administration (M01–M04, P01–P07) | One flat drawer mixes access, platform operations, release evidence, demo and licensing. | Use Access, Integrations and Platform operations sections in management. Demo/release evidence are operational utilities, not business peers. Grouping never grants OP or IA financial access. |
| Assistant (A01) | Valid alternative execution entry, not a new business domain or approver. | Global entry plus contextual continuation of the same object/job. Preserve common commands and server gates. |

# Candidate hierarchy, not a new arbitrary menu count

A useful test candidate is Overview, Costs and usage, Organizations and contracts, Settlement and billing, and Reports, with administration and My work/Assistant as utilities. This is not a claim that five primary destinations are universally optimal or approved.

Within that candidate, cost-data work exposes Intake, Allocation and necessary cost/measurement workbenches; connection setup and evidence open in context. Contracts expose shared selling/FX policy management without hiding cross-contract scope. Settlement exposes runs, approvals, documents, ERP processing, relevant balances/cases and required queues. Monthly recognition can enter ERP processing directly without first creating a billing run or invoice. Cross-domain reconciliation is reachable from each relevant object and from a directly addressable authorized workbench.

The replacement must avoid two extremes: a flat list of every screen and a deeply nested catch-all Settlement section. Prototype object entry and recurring queues before deciding final depth; never optimize to a predetermined primary-menu count. An audited screen route may remain valid without being permanently displayed in the LNB.

# Destination, object and task matrix

| Durable entry or alternate entry | Business object / task | Proposed depth and retained identity |
| --- | --- | --- |
| Overview | Scoped business summary | Direct entry, U01; links to owned queues rather than cloned queue data |
| Closing management | Period/deadline and exemptions | Settlement workbench, U02; calendar is a view |
| My work | Assigned action and event | Global utility, U03; customer projection retained |
| Cost and billing issues | Investigated business finding | Scoped queue, U04; contextual overview entry |
| Connections | Connection inventory and capability | Cost-data secondary entry, S01; setup/resume action opens S02 |
| Received inputs | Input job/revision | Direct operational queue S03 plus connection-filtered entry; no mandatory connection click-through for every daily exception |
| Costs, measurement, pools and allocation | Observed source rows versus maintained cost/policy | S04/B01/B02/B03 workbenches; supplier benefits B06 and buyer/seller marketplace B07 remain scoped views or authorized inventories |
| Contracts and policies | Contract, shared sales/FX policy versions | O01–O03/B04/B05; relationship changes stay O02 workflows, not silently deleted |
| Settlement | Run, approval decision and applied coverage | B08/B09/B10 remain distinct inventories/queues; comparisons and corrections keep object context |
| Documents and external processing | Issued artifact, submission, delivery and related case/balance | I01–I04/B11 retain canonical paths; API/file is a supported path, recognition is an independently enterable purpose-filtered queue, not invoice-gated |
| Reconciliation and evidence | Difference case and evidence artifact | S06 typed workbench; S05 contextual viewer plus authorized search; both remain directly addressable |
| Reports and audit | Aggregate analysis versus actual action history | R01 business reports, R02 separate authorized audit utility plus object links |
| Platform and access | Identity, integration, recovery/retention, support/demo/license | M01–M04/P01–P07 management, M05 self-only account; no expanded rights |
| Assistant, portal and prelogin | Alternate command entry / separate audience | A01, C01–C03 and L01–L03 retain their distinct scopes |

This matrix covers the 49 existing canonical IDs as review subjects, not 49 required visible LNB entries. It changes no route, public API, screen ownership or acceptance unit.

# Validation needed before adoption

Compare current and candidate navigation with role-scoped task finding: resume a failed import; validate/publish eligible input; inspect original evidence; edit a shared allocation policy; approve a changed sales policy; process monthly recognition without an invoice; recover ambiguous ERP posting; investigate receivable freshness; audit a correction. Verify internal-only/MSP/group profiles, read-only users, direct authorized links and no hidden existence/count leakage.

This is an analytical audit, not measured user-test evidence. Preserve all 49 canonical screen contracts, rights and review scopes while evaluating visible placement. No navigation rewrite, screen deletion, owner FIX or production authority is granted by this report.

# Independent review

Two independent source reviews identified taxonomy flaws in the implementation despite its prior coverage/boundary passes. Independent final report review PASS: the matrix preserves all 49 canonical IDs and material financial/visibility controls. This is not approval of a replacement menu count, prototype usability or implementation.
