---
type: Design
title: Deployment-realistic onboarding workflow
description: Defines conditional setup tasks, role handoff, evidence transitions and recovery for provider-specific onboarding.
status: draft
---

# HB-52 planning boundary

The owner requires onboarding detailed enough to represent real CSP setup, including high complexity hidden by the earlier happy-path prototype. [HB-52](https://linear.app/hybrid-billing/issue/HB-52/document-deployment-realistic-provider-onboarding-plans) expands this same r2 draft. Root owns all writing; three independent read-only domain reviewers cover AWS/Azure, Google/OCI and Alibaba/IBM. Current provider facts are sourced in the guides. Proposed platform behavior is explicitly a design, and all real-account deployment/permission/collection acceptance is NOT_RUN.

The existing prototype shows only route selection and illustrative validation states. It is **not a complete operational setup wizard**. Its three visual stages remain a navigation grouping; actual conditional tasks are determined by provider, agreement, source state, installation identity and network policy. No universal action count or universal single-field configuration is promised. This document is the next-screen planning input, not proof those screens already exist.

# Provider guides

| Provider | Detailed setup contract |
| --- | --- |
| AWS | [Account/export/role/storage setup](guides/aws.md) |
| Azure | [Agreement scope, export identity and storage](guides/azure.md) |
| Google Cloud | [Billing account, linked dataset and query identity](guides/gcp.md) |
| OCI | [Tenancy authorization and Oracle-owned report source](guides/oci.md) |
| Alibaba Cloud | [Preview, OSS subscription and authoritative bills](guides/alibaba.md) |
| IBM Cloud | [Account/enterprise route and report API](guides/ibm.md) |
| VMware / files | [Metering/cost and supplied-evidence paths](guides/other-sources.md) |

# Conditional setup model

The operator chooses source and existing/new configuration, then the platform constructs only the unsatisfied tasks. It first checks current authorization, provider availability and billing scope; unsupported scope must fail before any storage/role creation. Discovery stays within an approved account/container budget and cannot broaden access. Existing compatible resources are reused only after read-back; discovery is not permission to edit them.

| Task | Owner | Required input | Completion evidence | What invalidates it |
| --- | --- | --- | --- | --- |
| Installation connectivity | Installation administrator | Deployment location, egress/proxy/DNS policy, clock, service identity reference, trusted endpoints | Time-bound reachability and identity/audience result | Endpoint/proxy/identity/CA policy change |
| Billing scope and custody | Billing-account owner plus authorized product administrator | Exact provider scope IDs, agreement type, organizational relations, expected accounts, raw-data custodian and customer attribution | Approved scope revision and evidence of provider entitlement | Scope, ownership, role or membership change |
| Resource plan | Provider setup administrator | New/reused resources, storage/query location, encryption, permissions, billing/cost impact | Versioned plan and explicit setup intent | Template/configuration/version or destination change |
| Source production | Provider setup administrator / export service | Provider-specific export/table/report, schema, grain, destination and schedule | Provider resource/configuration read-back, not an accepted create request | Source/query/schema/destination change |
| Collection access | Credential/security owner | Workload/service identity, narrow read/query grants, renewal and secret reference | Actual authorized object read or bounded query/API response | Grant revocation/expiry, secret rotation, policy drift |
| First receipt | Collection operator / service | Authorized scope, bounded period and pinned source profile | Complete immutable execution/files/pages/query snapshot and source watermark | Source revision or incomplete pagination/chunk set |
| Format and coverage | Validation service, operator for findings | Mapping/target version and independently expected domains/periods | Full-scope field/type/precision/unit and coverage outcomes | Schema/profile/mapping/scope/input changes |
| Financial evidence | Billing preparer / authorized input confirmer | Required basis-specific comparison evidence, official bill source where required | Independent totals/identity comparison with no unexplained differences | New bill/revision, unresolved reconciliation or missing policy |
| Eligible publication | Currently authorized input confirmer | Fixed candidate/scope revision and evidence digest, explicit intent/rationale | Existing input.confirm_publish atomic decision | Authority or evidence revision conflict |

Each diagnostic persists observed fact, check time, safe request ID, evidence reference, dependency revision/digest, expiry, impact, responsible role and one safe next action. Store secrets only in approved custody; ticket/handoff records contain references, never values. Names and account labels are display data; canonical IDs bind every task.

Separate human setup authorization, provider export-service writes and recurring collector reads. A broad billing administrator role documented for export creation is not automatically the collector role. If least-privilege read access has not been verified, mark that provider profile UNVERIFIED instead of silently accepting administrator credentials. One-time installation credential configuration is counted separately from per-source setup, not hidden behind a green prerequisite.

# Proposed state transitions

A setup attempt owns tasks with states not_required, pending, ready_to_act, awaiting_owner, executing, outcome_unknown, verified, blocked, stale and cancelled. These are design state names, not implemented API enums. A task becomes verified only from observed evidence. User confirmation that a provider-console action was done triggers read-back; it does not certify success.

Tasks form a dependency graph rather than an irreversible sequence. Independent non-sensitive setup can continue while one task awaits an administrator. Data access still waits for custody and collection authority; financial configuration continues in the prescribed OP05 order. A failed format check does not erase a verified role; changed source scope invalidates dependent reads and acceptance. Show latest valid point and invalidation reason rather than resetting the whole wizard.

Create/update intent binds installation, connection, provider scope, plan/configuration digest and expected revision under the existing idempotency contract. On timeout, retain outcome_unknown and inspect the exact named resource/operation before retry. Never duplicate an export, role or subscription to make the screen appear to progress. Source correction and backfill are new evidence revisions, not permission to sum repeated cumulative reports.

# Administrator handoff and resume

A handoff task contains only the needed action, affected source/scope, required role, safe provider-console entry, resource plan digest, validity period and verification method. It can be copied or assigned within the product; sending email or invitations is a separate configured operation, not implicit authorization from this plan. A handoff link grants no provider privileges or product data access. A recipient authenticates and proves the delegated scope; an external administrator may act in the provider console without receiving product raw-data access.

Show who is responsible, what remains, last activity and a return action. Recipient completion triggers a fresh diagnostic with current principal/role/configuration checks. Changed scope or expired handoff requires renewed intent. Do not require unrelated billing users to repeat technical configuration or paste a service-account JSON/key. Retries preserve completed valid tasks and do not reinvite everyone.

# Screen specification for the next visual revision

| Stable child ID | Screen and visible content | Primary action | Conditional branch |
| --- | --- | --- | --- |
| setup-connectors.environment | Installation and provider readiness; existing/new source choice; actual provider agreement/availability limits | Check prerequisites | Missing service identity or network policy → administrator task |
| setup-attribution.source-scope | Exact payer/billing scope, observed child accounts versus approved custody, effective customer assignments | Confirm authorized scope | Unassigned account, shared payer or insufficient authority → scope owner |
| setup-connectors.resource-plan | Existing resources first; proposed creates/changes, destination/residency, encryption, charges and setup rights | Review setup plan | Unsupported storage/KMS/cross-tenant profile → explicit supported alternative, no silent downgrade |
| setup-connectors.provider-setup | Provider-specific required fields with known values reused; responsible administrator and official entry point | Open setup instructions / approved automation | Missing authority → scoped handoff; no fictional common OAuth |
| setup-progress.handoff | Pending action, owner, expiry and completed task evidence | Resume verification | Wrong account, changed plan, expired link or revoked rights → new intent |
| setup-readtest.diagnostics | Reachability, token/audience, metadata permission and actual object/query/API read shown separately | Repair the identified blocker | List works but read fails, KMS/firewall/billing permissions → targeted guidance |
| setup-schedule.history | Provider refresh cadence versus collector polling, actual available history, bounded backfill and freshness expectations | Confirm collection scope | Unsupported historical period stays unavailable; do not fabricate earlier costs |
| setup-first.validation | Execution receipt, source/target versions, mapping, counts, expected account/period coverage, errors and reconciliation | Inspect affected evidence | Unsupported schema/partial delivery/source drift → protected hold |
| setup-readiness.acceptance | Separate technical readiness, data completeness and financial eligibility; current actor and immutable evidence | Confirm eligible input only when prerequisites pass | Missing official bill, pricing policy, independent comparison or stale revision → blocked |

Every screen distinguishes unknown, not_applicable, waiting, failed and evidence-backed zero. Technical detail is expandable but blockers, data scope and cost/permission consequences are not hidden. The default view asks only for missing values; it does not falsely claim all fields can always be auto-discovered. Exact input provenance is shown as discovered, inherited-approved, recommended-draft or user-supplied. A recommendation cannot activate rates, legal identity or access rights.

# Shared validation and recovery cases

All are NOT_RUN until the adapter/profile and corresponding screens exist.

| Case | Setup or fault | Required result |
| --- | --- | --- |
| R01 | New source, no setup rights | Scoped administrator task; no blanket admin request or secret paste |
| R02 | Existing export matches all pinned parameters | Reuse with read-back, no new source mutation |
| R03 | Existing export filtered/different version/grain | Explicit limited capability or reviewed new plan; no silent change |
| R04 | Resource create accepted then timeout | outcome_unknown; inspect same resource before retry; one economic/source identity |
| R05 | Token expires mid-page/file sequence | Renew under same approved identity; resume/restart consistent snapshot; no partial publish |
| R06 | IAM appears valid but object/query denied | Distinct metadata/read evidence; correct policy/network layer only |
| R07 | Proxy/DNS/region/firewall/KMS failure | Safe bounded endpoint diagnosis, no insecure fallback or credential logging |
| R08 | Late data or no data for expected account | Waiting/delayed/missing/true-zero evidence separated; no inferred 0 |
| R09 | Pagination loop/missing chunk/manifest changes | Bounded retry and immutable execution; incomplete candidate held |
| R10 | 1,000–5,000 customer assignments under shared payer | Incremental selection/validation with counts and scoped bulk plans; no one-customer-per-export assumption |
| R11 | New child account or customer reassignment | Hold unapproved attribution; no new tenant access or historical rewrite |
| R12 | Schema/enum/decimal/unit change | Quarantine with restricted functions and affected fields; no guess or LLM auto-approval |
| R13 | Separate official invoice authority required | Analysis may remain protected; billing eligibility blocked until independent required evidence passes |
| R14 | Scope or role revoked while task runs | Stop further access and reject stale confirmation; history follows retention policy |
| R15 | Disconnect/cancel with an existing shared export | Stop only owned recurring work; no export/bucket deletion or other consumer disruption |
| R16 | Rotation or reconnect | Revalidate new credential/grant; retain source identity/checkpoints; no duplicate billing |
| R17 | Different provider/account/destination selected mid-wizard | Invalidate affected task evidence, preserve unrelated valid tasks, resolve pending remote actions |
| R18 | Keyboard/mobile/error recovery | Visible current focus, no focus theft from polling, one next action with responsible role |

# Implementation order and acceptance boundary

1. Resolve each supported provider profile: deployment topology, exact source version/API, least-privilege policy, network/encryption variant and economic revision mode. Unknown combinations remain unsupported; do not broaden support with a generic adapter flag.
2. Implement common persisted task/handoff/diagnostic state and secure credential references, then provider setup/read-back adapters. Reuse canonical authorization, evidence and idempotency contracts; new APIs require separate reviewed definitions.
3. Implement source-specific receipt/checkpoint/validation and independent comparison evidence before any eligible publication automation.
4. Extend the visual prototype to the nine conditional screen states and administrator handoff paths; measure actual human work, external waiting and repair effort separately.
5. Verify each supported profile in an explicitly authorized disposable account/environment, including negative and revocation cases. Publish the tested support matrix; do not call documentation research a live integration test.

The owner authorizes this planning work, not cloud deployments, customer data access or provider billing charges. The existing short mockup and its 192 synthetic UI cases do not satisfy the new operational setup acceptance. Final-design cleanup applies only after actual replacement acceptance; these provider guides are distinct responsibilities within the same draft.

# Operable configuration prototype

The owner rejected summary-only screens and requested actual account/setup fields with automation where feasible. [connection-setup.html](connection-setup.html) is a standalone Korean prototype with seven navigable surfaces: provider selection, account/access, source settings, reviewed plan, connection diagnostics, first-receipt/schema validation and readiness. Provider-specific forms use the guide contracts. Examples are synthetic; no secrets, files, provider APIs or remote resource creation are consumed. In-memory drafts disappear on reload.

Existing/new is an operational branch. New roles and collectors may remain pending until administrator read-back; missing setup cannot be bypassed by the read-test button. IBM account manual CSV does not request API credentials or a recurring schedule; its unresolved API profile stays blocked. File intake distinguishes one-time from recurring storage. Unsupported Azure, OCI, Alibaba and prototype GCP location combinations stop progression. Additional provider profiles remain unverified rather than hidden behind generic success.

Automation is conditional on existing authority: discover authorized resources, fill only missing source values without changing selected account/identity, inspect profile/coverage, retry bounded operations and retain valid configuration during repairs. This prototype's discovery and checks explicitly simulate those operations. It does not pretend a free-text reference selects a real registered secret; protected credential management and its backend are separate implementation work. Future actual selectors must query the authorized registry and validate existence.

Input changes clear approvals and dependent evidence; visible checkboxes follow that invalidation. Scenario changes invalidate tests and return to diagnostics. Publication remains blocked because independent billing-basis evidence is absent. VMware uses approved internal-cost and independent metering/cost comparisons, not a fictional supplier invoice. Runtime implementation requires persisted task ownership, secure identity registry, actual provider discovery/setup/read adapters and canonical authorization/confirmation APIs.

The compact prototype.html remains a separate navigation study while this draft is reviewed; it is not the detailed operational configuration deliverable. No final replacement acceptance is inferred.
