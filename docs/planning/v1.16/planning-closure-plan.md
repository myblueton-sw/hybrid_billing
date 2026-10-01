---
type: Planning Plan
title: Plan to close the v1.16 content gaps
description: Ticketed planning work will close seven source-backed gaps, resolve necessary decisions and reconcile the English planning baseline before owner acceptance.
status: draft
---

# Planning closure plan

Goal: produce a coherent, source-traceable planning baseline that the owner can assess for completeness. This plan permits documentation and design review only. Product development remains prohibited until the owner explicitly declares planning complete.

## Ticket and execution prerequisites

Create or reuse a semantically matching Linear parent ticket in the verified Hybrid Billing workspace. Create separate child tickets for independently deliverable work packages below. Ticket identifiers must be returned by Linear; package IDs below are local planning references, not invented issue IDs.

Assign a real accountable person, supporting role, work type, domain, acceptance criteria, dependencies and actual review evidence. Claim `machine:mac_mini` only while this machine actively works on a ticket. One active machine and one ticket branch per ticket; use isolated worktrees for concurrent writing.

Run ticket → classification/assignment/machine claim → branch → plan/design → relevant independent review → scoped changes → verification → changed-artifact re-review → QA → PR → authorized merge → merge read-back → ticket completion/handoff. A drafting or structural PASS does not authorize product implementation or close a release gate.

Tracking for this plan: https://linear.app/hybrid-billing/issue/HB-6/record-the-v116-content-review-and-establish-the-planning-closure-plan

HB-5 has been created through the authenticated browser in the verified Hybrid Billing workspace/team/project, assigned to Seung Woo Park. The local API credential still identifies `myblueton`; this is an unresolved API connection issue, not absence of the Hybrid Billing workspace or ticket. The owner-authorized README-only seed is remote commit `a5f4869`. HB-5 workflow PR #1 merged as `a8832ca`; remote read-back found no outstanding branch commits and identical branch/main contents. Subsequent work follows the normal ticket-branch/PR sequence. HB-6 records this review and plan. Remediation child tickets below remain pending creation; their work has not started. Do not redirect tracking to another workspace.

## Work packages and order

| Package | Scope and ownership | Dependency | Acceptance and review |
| --- | --- | --- | --- |
| PP-01 | Root: establish correct ticket tracking and the mandatory workflow rule; repository bootstrap must preserve existing files | Verified workspace and ticket | Rule covers documentation as well as code, device handoff, review, PR, merge read-back and honest completion; independent workflow review |
| PP-02 | Root with billing/DBA review: PC-01 calculation sequence and PC-02 warning acceptance | PP-01 | Required sequence and warning policy/acknowledgment are in English authoritative contracts, requirements, D/I/V work and screen semantics; order-sensitive expected amounts and warning cases defined, not marked executed |
| PP-03 | Root with architect review: PC-03 conditional billing-engine execution boundary | PP-01 | Credential/network/service-identity controls and negative bypass cases traced to selected-engine applicability; existing platform authorization preserved |
| PP-04 | Root with ERP/DBA review: PC-04 ERP profiles | PP-01 | Odoo/SAP distinctions restored as source-dated conditions; request/billing/FI completion semantics and per-call atomicity contract defined; current vendor facts remain UNVERIFIED unless officially checked |
| PP-05 | Root with product/QA review: PC-05 independent dispute holds | PP-01; reconcile with PP-04 external capabilities | Hold/request/release state and role contracts, partial scope, external uncertainty, payment obligations and non-cancellation cases linked to B11, APIs and acceptance |
| PP-06 | Root with product/QA review: PC-06 onboarding metrics and PC-07 favorites | PP-01 | Cohorts/denominators/timing categories and favorites persistence/revocation behavior specified; metric and access-boundary examples defined |
| PP-07 | Root PM with domain owners: decision register | PP-02–06 inputs | Initial support profiles, numeric policies, identity/retention/SLO decisions and ERP accounting ownership each have evidence, accountable owner, blocking effect and decision status; values are not invented |
| PP-08 | Root tech-writer: English artifact and trace reconciliation | Content packages stabilized | English Git-bound artifacts, preserved original source/provenance and third-party notices, consistent JSON/Markdown/work/screens/test references and current-versus-historical summaries |
| PP-09 | Independent architect/DBA/QA and root integration: final content re-review | PP-07–08 | Each PC finding closed by cited content and scoped re-review; remaining decisions explicit; owner receives a concise reviewable planning baseline |

PP-02–06 can be independently reviewed in parallel. Writing shared requirements/work/trace files must be sequential or have one designated writer. Do not let multiple child-ticket branches introduce overlapping changes without dependency and integration reconciliation.

## Closure scenarios

- Calculation: default order, approved order change, priority collision and minimum/maximum interactions have independent expected amounts.
- Warning: allowed class with current policy and named acknowledgment can proceed; an unwaivable blocker or expired policy cannot.
- Engine: direct unauthorized client/engine execution fails while authorized platform execution follows the same approval snapshot.
- ERP: accepted request, created draft, final invoice and FI posting remain distinct; timeouts and cross-call races preserve uncertainty and require reconciliation.
- Dispute: billing, delivery and external collections requests can be independently requested/released; unknown ERP acknowledgment cannot appear confirmed; due dates and issued documents remain unaffected unless separately authorized.
- Onboarding: connected accounts and first verified bills have distinct cohort outcomes and categorized waits.
- Favorites: personal preferences survive ordinary navigation but never reveal customers after access revocation.

These are acceptance-design obligations, not product tests executed during planning.

## Planning acceptance

[HB-7 architecture preparation](../../architecture/architecture-readiness.md) records the owner's self-hosted requirement and added technology, data, identity, admin, tenancy/organization and intelligence design scope. It supplies inputs to PP-07 and the later English baseline reconciliation; it does not close PP-02–06 findings, replace the 43-group requirements/work/screen trace, or authorize development. Detailed trace/schema/screen reconciliation remains PP-08 work.

Completeness requires source/user-request coverage, coherent contracts and states, named ownership for unresolved decisions, acceptance scenarios, trace consistency and independent domain review. Structural validation alone is insufficient.

The owner then decides whether planning is complete. That declaration is separate from PR approval or merge. A planning PR can merge while the development gate remains closed.

No delivery dates, staffing estimates or infrastructure budgets are asserted without evidence. No task is reported Done before the required authorized merge and read-back, including a Git report for a read-only review deliverable.

## Scope questions raised during planning

Kubernetes cost allocation does not require VMware. OpenCost describes physical/virtual, cloud/on-premises nodes and on-premises custom pricing. Proposed planning layers are authoritative infrastructure cost → Kubernetes allocation → separately approved customer pricing. Preserve VM/node-to-workload lineage to avoid charging the same infrastructure cost twice. Allocation basis, idle/shared-cost responsibility, customer identity boundaries and supported data paths still need explicit contracts and review. This is exploratory direction, not an approved new product feature or tool adoption. Sources: https://opencost.io/docs/specification/ and https://opencost.io/docs/configuration/on-prem/ (checked during this planning discussion).

Current positioning is a billing and reconciliation platform. Payout terminology implies sending funds to recipients; adding that capability would require its own scoped requirements and review. The naming question does not authorize fund movement or add a payout engine. Source: https://stripe.com/payouts (checked during this discussion).
