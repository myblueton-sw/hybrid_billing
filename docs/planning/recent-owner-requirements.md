---
type: Requirement Register
title: Recent owner requirements and unresolved decisions
description: Consolidates discussion additions with existing design evidence without changing the original requirement count or adopting candidate technologies.
status: draft
---

# Recent owner requirements

Tracking: [HB-10](https://linear.app/hybrid-billing/issue/HB-10/review-unresolved-refinements-and-consolidate-recent-owner). Reviewed baseline: `6de63eed2d9c04dd3394b5584eaa0ceeabe826fa`, 2026-10-01. Accountable owner: Seung Woo Park. Root owns PM/planning and writing; independent architecture, DBA and QA reviewers assess the relevant contracts. Worked machine: `mac_mini`.

This register consolidates owner-confirmed requests from the recent discussion. It is an English supplement to the local v1.16 source and published HB-7/HB-9 designs; it does not rewrite the original source. Described mechanisms are proposed designs, not deployed capabilities. Named finance/security/data/operations decision owners remain to be obtained by the accountable owner.

## Requirement bundles

The fourteen RA references below are local editorial bundles, not Linear tickets, replacement requirement IDs or fourteen wholly new features. Several elaborate existing local scope. The original baseline remains **43 top-level requirement groups**; overlap and requirement/work/screen/acceptance integration must be reconciled under PP-08 before reporting a revised baseline count. UD-01–12 are acceptance-design cases, not twelve additional requirements. All runtime cases are NOT_RUN.

| Ref | Owner-confirmed request and intended outcome | Current design evidence | Still unresolved / planning completion criterion |
| --- | --- | --- | --- |
| RA-01 | Self-hosted delivery under customer-operated installation | [Readiness](../architecture/architecture-readiness.md), operating specification | First MSP/enterprise/group profile, VM/container/Kubernetes runtime, network/offline conditions, supported versions, operator responsibility, capacity/budget and recovery targets; no mandatory VMware |
| RA-02 | Assess ClickHouse, Kafka and OpenTelemetry suitability | [Technology assessment](../architecture/technology-assessment.md) | Decide actual components from workload and operating evidence; ClickHouse is an analytics candidate, Kafka conditional transport, OTel observability. None is selected billing authority |
| RA-03 | Define data modeling and appropriate billing storage | [Logical model](../architecture/data-ownership/billing-data-model.md) and [technology assessment](../architecture/technology-assessment.md) | Physical types/precision, source/Claim keys, concurrency, partition/index strategy and coordinated evidence/publication/restore; PostgreSQL is the first authority candidate, not an adoption decision |
| RA-04 | Define sessions, tokens and authentication policy | [Identity policy](../architecture/security-boundaries/identity-session-permissions.md) | Initial IdP/MFA/service-client profile, session/token lifetimes, revocation bounds, bootstrap/recovery custody and authentication-root change authority (ED-02) |
| RA-05 | Provide admin operations and customer/user data management | [Admin design](../architecture/admin-design.md) | Final role/action/field matrix, supported flows and screen contracts; distinguish installation operations, customer profiles and financial authority; runtime/visual verification remains later work |
| RA-06 | Define MSP/customer/tenant separation and customer information | [Model: installation, workspace, Party and relationship](../architecture/data-ownership/billing-data-model.md) | Select first operating/isolation pattern and permitted profile fields; explicit sharing/delegation rather than automatic access by MSP or common name |
| RA-07 | Define organization management | [Logical model](../architecture/data-ownership/billing-data-model.md) and [admin flows](../architecture/admin-design.md) | Hierarchy/group/assignment policy, effective-time overlaps and approval; retain historical attribution, prohibit double allocation and avoid grants inferred from org moves |
| RA-08 | Define access by permission and scoped data views | [Identity policy](../architecture/security-boundaries/identity-session-permissions.md) and [admin views](../architecture/admin-design.md) | Exact action/customer/resource/field ceilings, prohibited role combinations and revocation; enforce analytical query paths (ED-01), not only portal permissions |
| RA-09 | Define API, upload and other collection cases | [Collection methods](../architecture/data-collection.md) | First actual source/schema/version/method profiles, capabilities and coverage; common fetch policy for downstream references (ED-03); receipt alone cannot establish billing eligibility |
| RA-10 | Clarify provider grouping versus other organizing criteria | [Collection grouping](../architecture/data-collection.md) | Separate workspace ownership, customer/contract, provider/authority, source scope, period/revision and org/resource axes; choose source identity/precedence and snapshot/append/hybrid semantics (ED-05) |
| RA-11 | Add intelligence for validation, correction, adjustment and prediction | [Intelligence layer](../architecture/intelligence-layer.md) | First use cases, methods/models, permitted data/actions and evaluation thresholds; deterministic gates and ordinary financial approval retain authority. Tenant-data training is conditional and ED-06 unresolved |
| RA-12 | Costs include complete authorized usage details and explanations | [Usage detail lifecycle](../architecture/usage-detail-lifecycle.md), detail/lineage/query/export sections | Source-resolution and disclosure profiles, fixed revision/scope/currency/basis, component bridges, full exports and independent totals. No invented finer detail or hidden supplier/margin disclosure; ED-01/04/05 remain dependencies |
| RA-13 | Store, manage and maintain the information over its permitted lifetime | [Usage detail lifecycle](../architecture/usage-detail-lifecycle.md), storage/administration sections | Class-specific retention basis/durations, hold/reference protection, archive/recall availability, backup/deletion controls, capacity and RPO/RTO. Temporary exports/broker retention do not replace authoritative history |
| RA-14 | Define how definitive pricing logic and final charges are determined | [Usage detail lifecycle](../architecture/usage-detail-lifecycle.md), pricing section | Named commercial/finance/tax/security decisions; rate/unit/tier/modifier/FX/tax/rounding policy and complete order (PC-01), warning acknowledgment (PC-02), independent expected amounts and separate policy/amount/issuance/ERP states |

## Full-scope clarification after HB-11

The owner instructs planning against the whole existing requirement set, without arbitrarily choosing a first target to reduce scope. The collection set includes AWS, Azure, GCP, OCI, Alibaba and VMware; existing HawkEye/TokenMeter connections and all other baseline requirement groups remain included. Source preservation, normalization, missing/duplicate checks, reconciliation and correction revisions already exist in local v1.16; they are not new features or additions to the 43-group count.

Accuracy first, mandatory cross-validation and operator confirmation are confirmed directions. [Collection design](../architecture/data-collection.md#accuracy-cross-validation-and-operator-confirmation) supplies the proposed detailed gate and links it to financial publication. The interrupted phrase about an optional capability is unresolved; do not infer optional AI, automatic confirmation or another exception. Exact interaction, authority matrix and policy values remain distinct from the confirmed directions.

Earlier requests for first/initial profiles in this register and HB-11 decision packets concern evidence, capability and activation planning only. They do not authorize a reduced design baseline or require the owner to reselect already documented providers. Gather available evidence for the full matrix; ask only about actual missing information, conflicts or decisions requiring owner authority. JEV-originated advice must be labeled **JEV recommendation**, with adoption and evidence stated separately; it is never owner approval.

[HB-13 verification and trace](../reviews/HB-13-full-scope/verification.md) bounds this reconciliation. Full English canonical requirements/work/screens/acceptance integration and final planning acceptance remain pending under PP-08/09.

## Additions recorded under HB-9

RA-12, RA-13 and RA-14 are the three owner requirement bundles documented under HB-9. Complete authorized detail includes all eligible permitted rows at supported resolution, rather than a single page. Historical maintenance covers retained source/result dependencies, current permissions, archive/recall, integrity, deletion and restore. Pricing finalization separates policy approval, calculation/reconciliation and amount approval from issuance and external confirmation.

These procedures are documented. Exact prices, formulas/order, warning policy, permitted source/detail profiles, storage products, retention durations, export formats and service targets are still unresolved. HB-9 publication did not close PC-01/02 or ED-01/04/05. Use the [current unresolved review](../reviews/HB-10-unresolved/findings.md) for the thirteen finding statuses and ordered closure criteria.

## Governance and scope clarifications

English Git-bound artifacts and English tool/agent instructions, Korean minimal user conversation, and the mandatory ticket → owner/machine → branch → design/review → verification/QA → PR → authorized merge/read-back → Done/release sequence are owner rules recorded in [AGENTS.md](../../AGENTS.md). They are work governance, not product features to add to the 43-group count. Correct Hybrid Billing browser tracking works; the API credential/workspace mismatch remains separate HB-5 connection work.

The Kubernetes/VMware discussion remains a scope decision: infrastructure cost and workload allocation must have lineage and avoid duplicate charging; VMware is not an adopted prerequisite. The owner rejected Payout positioning. Current positioning remains billing/reconciliation, with no fund-movement capability added by the terminology discussion; see [closure-plan scope questions](v1.16/planning-closure-plan.md).

## Integration and acceptance

[HB-11 supplementation](../reviews/HB-11-refinements/resolution.md) now supplies concrete contracts for the thirteen findings and D-01–07 decision packets, including RA-12/13/14. Contract supply/review is progress; final profile choices, canonical trace and owner acceptance are separate remaining work.

PP-07 obtains named decision owners and first support profiles, including the three latest bundles. PP-08 maps this supplement to canonical requirements/work/contracts/screens and acceptance records, preserves source provenance and publishes a consistent English baseline. PP-09 independently reviews actual closure or explicit scoped deferral. Do not mechanically sum 43 + 14, count finding IDs as features, or claim that a register replaces detailed contracts.

Root checks coverage of these requests; relevant domain reviewers verify financial, authority and lifecycle consistency. Structural document checks and independent content review do not certify implementation. No runtime, provider/ERP compatibility, performance, retention execution or human external specialist sign-off is asserted. The owner has not declared planning complete; product development remains prohibited.
