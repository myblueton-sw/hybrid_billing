---
type: Admin Design
title: Installation tenant customer and billing administration
description: Defines administrative responsibilities, organization workflows and evidence-based operational states for a self-hosted deployment.
status: draft
---

# Administration design

Part of [HB-7](architecture-readiness.md). An admin console is owner-requested and already represented by local v1.16 screen proposals. This document organizes those proposals; it does not replace all 49 screen specifications or claim UI implementation. Apply the [tenant model](data-ownership/billing-data-model.md) and [permission policy](security-boundaries/identity-session-permissions.md).

| Administrative area | Workflows | Authority and safeguards |
| --- | --- | --- |
| Installation operations | Preflight, runtime/version/storage/network status, backup/restore, upgrade/migration, secret/certificate expiry, entitlement | Installation operator; financial access requires separate grants. No secret viewer, silent TLS bypass or unapproved external diagnostics |
| Tenant/workspace | Create/configure workspace, allowed operating profile, isolation/disclosure/login policies and customer scope | Authorized bounded workspace owner; tenant lifecycle and retained financial evidence remain distinct |
| MSP/customer profiles | Customer directory, Party/legal/billing contacts, effective servicing relationship, contracts, provider accounts and ERP mappings | Explicit customer/resource grants; verify legal/payment/recipient changes; no automatic global MSP access |
| Organization | Legal-unit references, departments/projects/cost centers, reporting groups, account assignments, future/retroactive revision impact | Draft/preview/validate/approve/activate; cycles/overlaps/double allocation blocked; historic attribution preserved |
| Identity and permissions | Local/SSO/LDAP policy diagnosis, invite/JIT rules, memberships, roles/delegation, session revocation and service-account custody | Access administrator within a role ceiling; no grant beyond authority, email auto-link or silent escalation |
| Provider/data operations | Connection capabilities, coverage, manifests, quarantine, source corrections, replay and retention/holds | Scoped data operator; connectivity is not complete source coverage or billing readiness |
| Financial closing | Rate/FX/allocation changes, runs, reconciliation, warnings, approvals, Claims, documents, adjustments | Preparer/approver/issuer separation and fixed revisions; HB-6 calculation/warning gaps remain pending |
| ERP/receivables | Purpose-specific submissions, unknown results, monthly recognition, receipt evidence, due/overdue freshness and dispute holds | Finance-approved capabilities/authority; job success is not external posting or invoice completion |
| Analysis/intelligence | Data-quality/anomaly review, correction/adjustment proposal evidence, forecast/evaluation and module controls | Analyst/model has scoped input; acceptance uses ordinary approved commands, never direct ledger write |
| Audit/support | Actual actor, approvals, changes, errors, restricted support sessions and export evidence | Scoped auditor/support; redacted logs; completed external dispatch cannot be retrospectively undone |

## Principal user flows

Installation operator preflight → atomically create first administrator → test login/recovery policy → configure workspace/Party/customer/org → authorize source account assignments → receive/verify evidence → configure contract/rates or internal allocation → calculate/reconcile → approve snapshot → issue/hand off → confirm external result and reconcile. Customer user access begins only after explicit membership and disclosure policy. A successful setup diagnostic never grants financial authority.

MSP customer onboarding captures legal/billing identity, relationship effective dates, provider billing scope, contract/ERP mapping and designated approver. Customer organization is maintained independently of source/provider trees; a payer shared by several customers has an authorized attribution rule and protected shared evidence. Offboarding revokes future access/collection according to contract while preserving required Claims/documents/shared evidence and handling pending responsibilities; it does not delete issued liabilities.

Organization move preview lists affected accounts/contracts/allocation pools, inherited-policy proposals, explicit grants, open jobs and historical periods. Authority changes are reviewed separately from structural moves. Future activation affects new eligible runs; retroactive effects require explicit correction/recalculation and reapproval. Group membership does not duplicate usage and cross-workspace reporting needs authorized sharing.

## Data views and responsibility matrix

This is a proposed baseline role/action matrix. Every allowed cell requires current workspace/customer/resource/field scope; role labels do not create global privileges. Multiple roles do not waive prohibited self-approval. Exact approval ceilings and role combinations require owner/security/finance decisions.

| Responsibility | Permitted reads | Permitted changes | Restriction |
| --- | --- | --- | --- |
| Installation operator | Runtime, capacity and redacted diagnostics | Approved installation/backup/upgrade operations | No tenant finance/profile access from operator role alone |
| Workspace/access administrator | Scoped user/membership/login/session metadata | Bounded invite/role/delegation/revocation and tested IdP policy changes | Cannot reveal secrets or grant beyond delegated ceiling |
| MSP customer manager | Assigned customer/profile/org/contract/source relationships | Authorized contact/org/assignment drafts and bounded user management | Customer-specific grants; legal/tax/payment changes require separate approval |
| Data operator | Assigned manifests/coverage/mapping/quarantine | Diagnose/collect/upload/replay and propose corrections | No own billing approval; shared payer raw evidence needs explicit privilege |
| Billing preparer | Authorized costs/rates/runs/reconciliation/drafts | Calculate, propose adjustments and request approval | No protected self-approval or issuance without valid approval |
| Billing approver/issuer | Fixed evidence, approved snapshot and current financial scope | Approve/reject; separately authorized issue/submission | Actor/revision/amount checks; reject stale approval and prohibited combinations |
| Analyst | Permitted aggregate/detail features and findings | Analysis/forecast/correction proposals | Restricted margin/customer scope; no ledger/ERP direct writes |
| Auditor | Assigned evidence/change/approval history and permitted exports | No routine business mutation | Read-only is not all-tenant or unrestricted PII access |
| Customer organization administrator | Own customer membership/org and disclosed charges | Authorized own user/profile requests and bounded memberships | No MSP margins/shared raw costs/cross-customer access or payment self-approval |
| Customer user | Assigned org/contract/documents and disclosed figures | Own permitted preferences/profile/security actions and dispute requests | No authoritative charge/receipt/term change; preferences grant no access |

Separate profile ownership: a person manages allowed display/contact preferences and own security settings; an access administrator manages identity/membership under policy; a customer manager manages scoped customer records; finance owns verified legal/tax/payment/recipient revisions. Login subjects/internal IDs are not editable identity keys. Profile updates require validation, revision/audit and conflict handling; a submitted email cannot change trusted account linkage. Disclose minimal personal fields and protect audit values according to policy.

Admin query paths return authorized rows/fields, stable cursors and aggregates over the same allowed set. Customer views expose issued/allowed charges, due dates and confirmed receipt freshness; internal recognition, provider purchase rates, other-customer metadata and MSP margins remain restricted unless explicitly permitted. Caches/projections include ownership, policy and data revisions. Export/model retrieval inherit identical restrictions and check current authority before delivery; hiding front-end columns is insufficient.

## Screen and interaction contract

Map into existing local families: P01–P06 installation/modules; O01–O03 organization/contracts; S01–S06 source/data; B/I closing/document/submission; M01–M05 identity/access; U/R operational/audit; C customer views. This mapping is an integration reference, not a claim every family owns all workflows above. New intelligence screens/actions need detailed screen-spec and requirement-trace follow-up before implementation.

Every list has server search, stable cursor/filter and authorization-scoped counts. Bulk selection freezes an explicit target manifest and rechecks current permissions before work; one customer failure must not display another customer's data. Show loading, empty, denied, partial, failed, stale/unknown and completed states with retry/cancel limits. Monetary tables display currency, amount basis, input revision, coverage and as-of; missing/stale values never appear as zero or confirmed.

Before sensitive actions show affected scope, reason, evidence/revision, approval requirement and irreversible/external effects. Execution exposes job ID and actual business outcome; accepted, running, confirmed and unknown remain distinct. Keyboard focus, accessible errors and responsive layouts require real UI verification. No new executable mockup is produced here.

## Administrative acceptance plan

Negative cases include operator attempting tenant ledger access; MSP viewing an unassigned customer; org move implying extra grants; cross-tenant bulk search/export; credential leak in diagnostics; finance self-approval; uncertain ERP shown complete; expired support session; offboarding deleting shared evidence; restore reviving access; forecast displayed as issued charge. Functional/a11y/security/visual tests are **NOT_RUN**. Exact role/action matrix, bulk limits, final layouts and permitted customer-profile fields remain unresolved.
