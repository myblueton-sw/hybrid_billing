---
type: Logical Schema Draft
title: Source-derived typed relationship bindings
description: Binds HB-15 findings FA-06 to FA-12, SR-01 and SR-02 to typed source-cited entities, keys, states, guards, open decisions and synthetic acceptance cases for the logical schema.
status: draft
sources:
  - resource: ../../../통합_빌링_플랫폼_구현_기획안_v1.16.docx
  - resource: ../../reviews/HB-15-full-planning/re-audit.md
  - resource: ../../reviews/HB-15-full-planning/source-review.md
  - resource: logical-data-schema.md
---

# Source relationship bindings

Part of Linear ticket [HB-23](https://linear.app/hybrid-billing/issue/HB-23). Each section remediates one finding of the HB-15 [re-audit](../../reviews/HB-15-full-planning/re-audit.md) or [source review](../../reviews/HB-15-full-planning/source-review.md) by restoring the source relationships as typed planning contracts. FA-01 to FA-03 are amended in place in the [logical schema](logical-data-schema.md), [inputs, pricing and ERP](../../contracts/billing/inputs-pricing-and-erp.md) and [source pricing obligations](../../contracts/billing/source-pricing-and-cost-obligations.md).

`B` locators are zero-based body-child indexes of the original v1.16 DOCX and `.R` is a table row, as in the HB-15 reports. Local `detailed-requirements.md`, `requirements.md` and `retention-access.md` references are authorized local v1.16 companions that are not yet published in Git. This document is under the planning gate: entity names are logical proposals, not tables, APIs, engines or implemented behavior. Names marked **HB-23 derived proposal** are not source text. Customer values, durations and vendor capabilities stay decisions. All acceptance cases are synthetic and **NOT_RUN**.

## Common rules

- Every tenant-owned entity carries the logical schema `Scope`, server-derived from the authenticated context, and uses `ScopedRef` with exact revision when historical meaning matters ([common types](logical-data-schema.md#common-types-and-reference-rules)). A client-supplied workspace or tenant ID grants nothing (B0086).
- Secrets, tokens, verifiers and credentials appear only as protected references or one-way verifier digests. No package, link, payload or log carries secret material.
- Approved revisions are immutable; current state changes through owner commands with expected revision. Analytical projections never authorize a transition.
- Conditional scope stays conditional: Marketplace support, external plugin execution and later product Marketplace selling are not activated by this document.
- Naming provenance: entity names that appear in the source text (for example `BalanceReference`, `BillingCase`, `DeliveryPackage`, `ExternalDocumentLink`, `MasterMapping`, `PeriodCost`, `PluginManifest`, `SupplierInvoice`) or in a cited local companion are cited there. Names first given in the HB-15 findings are review names. Every other name, and every English state identifier translated from a Korean source sequence, is an HB-23 derived proposal even where not individually marked.
- Commands that act on these entities use the [planning action](../../contracts/commands/planning-actions.md) envelope. Registering new typed commands for them is not done here.

## FA-06: operational onboarding setup records

Source: B0935, B0936, B1059, B1061, B1145; local DB-08 and QA-07. Extends `Connection`/`SourceAccount` in the [evidence schema](evidence-detail-schema.md) and the [onboarding obligations](../../contracts/operations/source-policy-onboarding-and-verification-obligations.md) OP-05.

| Entity | Identity key | Required data and relationships |
| --- | --- | --- |
| `OnboardingProject` | `(Scope, project_id)` | Installation, issuing legal entity, contract refs, connection path and assigned owners (B0935) |
| `SetupStep` | `(project_id, step_id, step_revision)` | Purpose, performing role, required permission, created resources and added cost, official console link with last-verified date and supported version, expected result, upstream step dependencies (B1059, B1061). State `not_started`, `in_progress`, `waiting_owner_or_input`, `blocked`, `completed`, `revalidation_required` (B0936) |
| `ConfigurationRevision` | `(project_id, config_revision)` | Settings digest, product version, template revision and its last-verified date (B0936, B1061) |
| `DiagnosticRun` | `(Scope, diagnostic_run_id)` | Step, exact configuration revision, digest of only the settings that step depends on, product version, checks, observed facts, result code, validity expiry (B0935, B0936) |
| `ProviderConnection`, `BillingRelationship`, `CapabilityEvidence` | Existing `Connection`/`SourceAccount` identity | Source account relationship, validity interval and readable data kinds (B0935) |
| `StepHandoff` (HB-23 derived proposal name) | `(Scope, handoff_id)` | One step target, issuing principal, one assigned actor (a specific principal), permitted actions and resources no wider than the issuer's current authority, expiry, link token stored only as a one-way verifier digest, `issued`/`consumed`/`revoked` state. Carries no secret or full billing data (B1059) |
| `ProviderPermissionCustody` (HB-23 derived proposal name) | `(Scope, connection_id, custody_class, revision)` | `custody_class` is `setup_temporary`, `export_writer` or `collector_reader`; actor, action, scope, reason, validity, revocation condition and actual-read verification evidence (B1145, QA-07) |

Guards:

1. A step is `completed` only while its latest `DiagnosticRun` matches the current digest of the settings that step depends on and the product version, and is unexpired. A change to those settings, the product version or validity invalidates that step and its dependent downstream steps, which move to `revalidation_required`; a change to unrelated settings invalidates nothing (B0936).
2. Certificate, permission, export or ERP mapping changes require revalidation, owner notification and a link to close impact (B0936).
3. Revoking `setup_temporary` custody never revokes `export_writer`; `collector_reader` never holds setup or export write permission (B1145). Readiness requires that `setup_temporary` custody whose validity ended or whose step completed is revoked with confirmation, and that remaining custody is verified by an actual read operation, not a role name (B1145).
4. A handoff link is accepted only when it is unexpired, neither consumed nor revoked, presented by its assigned actor authenticated in the same Scope, for its target step, and with actions no wider than the issuer's authority rechecked at use. Every other presentation rejects. Only the verifier digest is stored, as for the bootstrap/invite class in [session and token classes](../security-boundaries/identity-session-permissions.md#session-and-token-classes).
5. Readiness is decided from valid completed steps and capability evidence. `OnboardingCohort` connected or verified metrics in the logical schema are outcome measurements and never readiness authority.
6. Incomplete onboarding shows the billing target as pre-activation or blocked (B0936).

Decisions: per-provider step catalog, diagnostic validity durations, handoff expiry and notification recipients.

| Case | Scenario and expected result |
| --- | --- |
| H23-S12 | Diagnostic passed with dependent-settings digest D1; those settings change to D2: step returns to `revalidation_required`; activation is refused until a new run on D2 passes. Unrelated upstream steps, and this step after a change to a setting it does not depend on, stay `completed` |
| H23-S13 | Handoff link for step 3 used for step 4, after expiry, after revocation, by a principal other than the assigned actor, or from another tenant's Scope: each rejected; existing inputs are preserved |
| H23-S14 | Setup owner's temporary permission is revoked after setup: export writer keeps running, collector reader still reads; a cohort showing 100% connected does not mark the project ready. Setup custody still active after its validity ended, or collector access confirmed only by role name: project not ready |

## FA-07: plugin lifecycle

Source: B1432, B1433.R001–R006, B1437, B1439, B1440, B1441, B1444, B1445.R002–R005, B1450, B1526, B1527, B1528; local QA-06. Refines the `Extension/release` row of the [domain contract](../../planning/v1.16/domain-contracts.md) and the plugin revision pinned by `CalculationRun`.

| Entity | Identity key | Required data and relationships |
| --- | --- | --- |
| `PluginManifest` / `PluginRevision` | `(plugin_id, publisher, version, artifact_digest)`, immutable | Contract and API support range, license, configuration schema, required scopes, customer scope, declared secret slots only (no secret references or values), permitted egress, timeout, concurrency, memory/CPU limits, retryable operations, health endpoint, declared adapter operations (B1433.R001–R006). Signed package or remote endpoint with server identity, contract version and deployment evidence (B1432, B1527) |
| `CompatibilityEvidence`, `PluginTrustDecision` | `(plugin_revision, core_contract_version, decision_revision)` | Source, signature, license and compatibility checks, test connection and sample reconciliation evidence, reviewer and decision (B1439) |
| `PluginInstallation` | `(installation_id, plugin_revision)` | Installation approval of requested permissions, egress, secret access and support party (B1527). Separate from customer use permission (B1439) |
| `PluginActivation` | `(Scope, plugin_id, activation_revision)` | Exact plugin revision, configuration revision, approved scopes and egress, Scope-owned secret references bound to the declared slots; state `registered`, `in_use`, `error`, `quarantined`, `stopped`; owner, last success and close impact (B1439) |
| `PluginRunPin` | Existing `CalculationRun`/`Job` reference | Exact plugin revision digest and configuration revision; in-progress work keeps its pinned version across updates (B1441) |
| `PluginStorageInventory` | `(Scope or explicit installation-only scope, plugin_id, storage_ref)` | Storage space, data owner, migration and deletion rules, retained code digest and execution dependencies for reproduction (B1528). Installation-only storage holds no customer-owned data |

Guards:

1. Plugin output is a candidate. The core validates currency, decimal, conservation, totals and allowed items and confirms only after approval; plugins never write the core ledger directly (B1437).
2. Plugin results and tool descriptions are not trusted commands and cannot expand AI permissions or tool lists (B1527). Only job-scoped limited tokens and required secret references are passed (B1440).
3. Updates switch after compatibility check and a partial customer trial; stop and rollback check running jobs and migrations (B1441). Activation, update and rollback each require a current, unrevoked `PluginTrustDecision` for the exact artifact digest; a quarantined or revoked revision cannot be activated by rollback. Calls to destinations outside the approved egress are blocked (B1433.R002, B1440).
4. Removal keeps confirmed evidence in the core. Viewing confirmed evidence never depends on running an old module; a vulnerable old version runs only in an isolated verification environment and is never reactivated to read history (B1441, B1528).
5. Release obligations stand independently of whether external plugin execution is adopted: plugin contracts, contract tests, SDK and example plugins are managed with the product, and a module marketplace is later scope (B1444, B1445.R002–R005).

Decisions: SDK language, signing trust roots, resource limits, compatibility policy values.

| Case | Scenario and expected result |
| --- | --- |
| H23-S15 | Plugin revision with a digest different from its approved manifest: activation refused. Install approval exists but customer use permission does not: calculation for that customer rejects |
| H23-S16 | Update to revision 2 while a run pinned to revision 1 is in progress: the run completes on revision 1; switch happens only after compatibility evidence and a partial trial |
| H23-S44 | Rollback from revision 2 to revision 1 after revision 1 was quarantined or its trust decision revoked: rejected; revision 2 stays or the plugin stops |
| H23-S45 | Activated plugin attempts a call to a destination outside its approved egress, or requests a secret reference owned by another Scope: blocked and recorded; no secret released |
| H23-S17 | Module removed after issuance: issued and allocation history remain viewable without executing the module (EXT08); the same fixture through the built-in collector and the example module yields equal evidence, amounts and allocation totals (EXT07; B1450) |

## FA-08: source-result return and external event delivery

Source: B0082.R003, B0082.R005, B0082.R006, B0085, B0086, B0088, B0092, B1424.R003–R006, B1427, B1428, B1429, B1524; local RT-01 and QA-05 remain the source contracts. Two families stay separate; neither is a generic event payload.

Source intake and result return:

| Entity | Identity key | Required data and relationships |
| --- | --- | --- |
| `SourceEventEnvelope` | Receipt key `(installation, provider, source_scope, source_event_id, source_revision)` (B0088) | `source_id`, `source_event_id`, `schema_version`, `resource_id`, `meter_id`, `event_time`, `period_start`, `period_end`, decimal-string `quantity` and `unit`, `quality`, `source_revision`; conditional `request_id`, `attempt_id`, `workload_id`; cost `amount`, `currency`, `cost_origin`, `cost_status` (B0085). Correction `supersedes_id` and `correction_reason` (B0082.R005). Maps to `SourceEconomicIdentity`/`SourceRevision` |
| `SourceResultReturn` (HB-23 derived proposal name) | `(Scope, source_connection_id, document_id, document_revision)` | Approved `invoice_id`, `status` and protected `download_ref` only (B0082.R006); recipient connection, disclosure policy revision, send-time authorization evidence. Fetching `download_ref` rechecks current authority ([ED-01](../../contracts/security/access-and-trust.md#ed-01-mandatory-query-boundary)) |

Guards: tenant scope equals the authenticated connection (B0086); same receipt key with different content is quarantined as conflict and corrections reference the prior revision, never overwrite (B0088, B0082.R005); start-inclusive, end-exclusive intervals and counter/interval/gauge semantics are explicit (B0092); a result return carries only approved documents under current disclosure rights, never unapproved documents or out-of-scope cost.

External event delivery:

| Entity | Identity key | Required data and relationships |
| --- | --- | --- |
| `ExternalEvent` | `(source, id)` per CloudEvents 1.0 (B1424.R003, B1524) | `type`, `specversion`, `subject`, `time`, schema version, object reference and object revision; minimal identifier payload (B1427). Types include `billing.completed`, `approval.required`, `invoice.issued`, `allocation.finalized`, `closing.overdue` (B1424.R004). Emitted from the owner's committed outbox |
| `EventSubscription` | `(Scope, subscription_id, revision)` | Subscriber principal and client with effective permission equal to the user and client intersection (B1415), event types, resource scope, operator-approved destination under egress policy (B1428), signing key version reference, state. A destination change is a new revision requiring re-approval |
| `EventDelivery`, `DeliveryAttemptRecord` | `(subscription_id, event source+id, attempt_no)` | Signature, timestamp, key version, result, exponential retry, failure archive and approved resend history (B1424.R006, B1427) |
| `EventCursor` | `(Scope, subscription_id or consumer, position)` | Retention horizon of replayable events; expired-cursor error and state resynchronization path (B1429) |

Guards: at-least-once, duplicates possible, no ordering guarantee; consumers deduplicate by source and id and confirm current state by object revision through the API (B1424.R005, B1427). Recipient permission is checked at subscription time and again at send time (B1427), including cursor replay and approved resend. Metadata endpoints, arbitrary loopback, DNS rebinding and unapproved redirects are blocked (B1428). Webhook receipt success is neither customer document delivery nor billing approval (B1429).

Decisions: event retention horizon, retry limits, signing algorithm and key rotation period.

| Case | Scenario and expected result |
| --- | --- |
| H23-S18 | Event envelope with tenant ID different from the authenticated connection, or a correction whose `supersedes_id` does not exist: rejected; nothing overwritten |
| H23-S19 | Result return for an unapproved draft or including another customer's cost: not sent; an approved invoice returns only `invoice_id`, `status` and `download_ref` |
| H23-S20 | Same event delivered twice out of order: consumer dedupes by source and id and reads the current object revision; an older revision never rolls state back |
| H23-S21 | Subscriber's permission revoked between subscription and send: delivery suppressed. Cursor older than retention: explicit expired-cursor error and resync path, not silent skip. Approved resend or cursor replay after the recipient's permission was revoked: suppressed |

## FA-09: enrollment and authentication policy

Source: B0919, B1414, B1415, B1515, B0389; local HB-42, AR-09 and `retention-access.md` (identity provider connection, authentication policy, bootstrap and invitation). Extends the identity rows of the [logical schema](logical-data-schema.md#identity-parties-and-effective-policy) and the [access and trust contract](../../contracts/security/access-and-trust.md).

| Entity | Identity key | Required data and relationships |
| --- | --- | --- |
| `EnrollmentVerifier` (HB-23 derived proposal umbrella) | `(installation_id or Scope, verifier_id)` | Purpose `installation_bootstrap`, `invitation` or another approved purpose; target installation/workspace, role ceiling, expiry, one-use flag; verifier digest only, never the raw token; state `issued`, `consumed`, `expired`, `revoked` with consuming identity and time; high-entropy verifier, bounded attempts, reissue invalidates the prior verifier, inviter's current authority rechecked at consumption (B0919; local `retention-access.md` bootstrap and invitation; [session and token classes](../security-boundaries/identity-session-permissions.md#session-and-token-classes)) |
| `LocalRealm`, `LocalLoginIdentity` | Unique `(Scope, realm_id, normalized_login_id)`; a local realm belongs to one workspace (local `retention-access.md` per-organization realm) | Internal UserID, credential reference held by the authentication component; no plaintext or reversible password storage |
| `IdentityProviderConnection` | `(Scope, idp_connection_id, revision)` | Protocol, trust metadata, certificate and secret references, identity namespace `(connection, issuer, subject)` |
| `AuthenticationPolicyRevision` | `(Scope, policy_revision)` | Allowed methods, SSO enforcement, local allowance, MFA and reauthentication, session lifetime, external collaborators, failure limits, approved emergency access; proposer and independent approver. Any weakening (SSO enforcement off, MFA reduced, local login allowed, emergency access enabled) follows [ED-02](../../contracts/security/access-and-trust.md#ed-02-authentication-root-change) proposer/approver separation; self-approval is prohibited |
| `SupportAccessGrant` | `(Scope, grant_id)` | Real support actor, target customer, reason, approved scope, expiry and revocation (B0389, AR-09) |
| `ClientDelegation` | `(Scope, client_id, user_id, grant_revision)` | Effective permission is the intersection of user and client permissions (B1415) |

Guards:

1. Bootstrap creates exactly one first administrator atomically; reuse is blocked and no common default password exists (B0919). Concurrent consumption has one winner.
2. Invitation consumption checks target workspace, expiry and role ceiling; acceptance in another workspace rejects. JIT provisioning never grants administrator roles automatically.
3. Under SSO enforcement, local login is not a bypass for ordinary users; an IdP outage never switches methods automatically except through an approved emergency policy. Emergency access requires named custody, reason, limited scope and time, independent audit and post-use revocation ([revocation and outage contract](../security-boundaries/identity-session-permissions.md#revocation-and-outage-contract)).
4. Changing an identity provider trust root is a `TrustChange` with epoch activation, not ordinary enrollment.
5. Approval-request permission is not approval permission (B1415). Support access never acts as or impersonates an approver (B0389).
6. B1515 states that application database accounts avoid owner, superuser and BYPASSRLS and that RLS and service permissions are tested together, citing PostgreSQL row security (RV4). PostgreSQL remains a candidate engine, so this applies when PostgreSQL RLS is adopted; database administrator bypass is distinct from product user isolation. Verification is a physical-design gate.

Decisions: identity provider products, allowed methods per organization, MFA scope, self-signup, recovery and collaborator policies (local HB-42).

| Case | Scenario and expected result |
| --- | --- |
| H23-S22 | Two concurrent bootstrap consumptions of one verifier: one administrator created, second rejects; replay after consumption rejects |
| H23-S23 | Invitation for workspace W1 with ceiling `viewer` accepted in W2 or after expiry, or used to obtain `approver`: rejected |
| H23-S24 | SSO-enforced organization: ordinary user's local password login rejects; support actor viewing a customer under a valid grant cannot approve and is recorded as the real actor. Support access with an expired or revoked grant, or to a customer other than the grant's target: rejected |
| H23-S46 | Security administrator proposes reducing MFA for an organization and approves the same revision: rejected as self-approval; an independent approver is required before activation |

## FA-10: scoped customer transfer and historical import

Source: B1506.R005, B1513, B1203, B1475. Distinct from `RecoveryManifest` in the [evidence schema](evidence-detail-schema.md#lifecycle-and-coherent-recovery-metadata).

| Entity | Identity key | Required data and relationships |
| --- | --- | --- |
| `CustomerTransferDecision` | `(source Scope, transfer_id, revision)` | Customer scope, source and target installation, authorized disclosure scope preserving data ownership, other-customer and MSP margin policies, approver and reason (B1203, B1513). Rechecked as current and unrevoked at package generation |
| `TransferManifest` | `(transfer_id, manifest_digest)` | Contracts and revisions, issued documents, source evidence, balances, external references and verification manifest (B1513). Lists secrets only as connections to re-establish, never secret material. Package artifact has an expiry and is downloadable only by the authorized target recipient under a current-authority check |
| `TransferAcknowledgement` (HB-23 derived proposal name) | `(transfer_id, target_installation_id)` | Received manifest digest verified, acknowledging actor and time (B1506.R005) |
| `HistoricalImport` | Unique `(target Scope, source_installation_id, source_object_id, source_revision)` | Read-only history record, legacy link to source identity, import batch (B1513) |
| `InstallationBinding` | `(target Scope, customer_relationship, binding_revision)` | New installation binding and newly established connections |
| `OpenFinalWorkItem` | `(transfer_id, item_id)` | Last settlement status, `pending_final` settlements, unknown external effects, open BillingCases; custody `source` or `target`, owner and due action (B1506.R005, B1475) |
| `SourceAccessRevocation` | `(transfer_id, credential_ref)` | Permission and key revocation at the source with evidence (B1506.R005) |

Guards: a coherent restore never proves an authorized transfer; an organization group move is not a transfer (B1203). Imported issued documents are read history and never replay as new issuance events, Claims or charges (B1513). Re-import of the same source identity and revision is idempotent. Transferred balances land at the target as `OpeningBalanceReference` records with the covered historical range (FA-11), never as new increases. Transfer completes only with acknowledgement, revocation evidence and every open final-work item assigned.

Decisions: package format, disclosure profile for transfers, who approves at source and target.

| Case | Scenario and expected result |
| --- | --- |
| H23-S25 | Import of an issued invoice from the source installation: stored as read-only history; no number reserved, no Claim consumed, no issuance event emitted; repeating the import creates nothing new |
| H23-S26 | Transfer package containing a credential value or another customer's records: generation rejected |
| H23-S47 | Transfer approval revoked after the decision but before package generation, or package fetched after expiry or by another recipient: generation or download rejected |
| H23-S27 | Transfer with a `pending_final` settlement and an unknown ERP result: completion blocked until both items have custody and owner; restoring a backup at the target does not satisfy acknowledgement |

## FA-11: commitment, opening balance, Marketplace, final work and inquiry

Source: B0533.R001–R007, B0583, B1475, B1481.R006, B1484, B1485, B1487, B1488, B0695.R001–R007; local DB-09, DB-10, QA-02, QA-09. Complements SP-01, SP-02 and SP-10 in [source pricing obligations](../../contracts/billing/source-pricing-and-cost-obligations.md) and the flags on `CostComponentFact`/`MarketplaceComponentFact`.

Commitments (provider-supplied results only; no CSP amortization engine):

| Entity | Identity key | Required data |
| --- | --- | --- |
| `Commitment` | `(Scope, provider, source_commitment_id)` | Installation, purchasing entity, owner, product type, contract, start/end, currency, payment method, committed quantity and unit (B0533.R001) |
| `CommitmentVersion` | `(commitment_id, source_revision)` | Retrieval time, effective time, sharing scope, state, original locator and digest (B0533.R002) |
| `CommitmentSourceCharge` | `(Scope, source row or file id, source_revision)` | Purchase/usage/offset/correction relation, account, SKU, period, quantity, original currency and amount (B0533.R003) |
| `CommitmentPeriodCost` | `(commitment_version, source_interval)` | Provider-supplied cost, used and unused, customer allocation, provider-supplied residual, source evidence (B0533.R004) |
| `BenefitAttribution` | `(source usage row, commitment_version, interval)` | Actual beneficiary account, effective cost, discount comparison basis, evidence grade (B0533.R005) |

`AllocationPolicy` and `SalePolicy` (B0533.R006, R007) are typed `PolicyRevision`s with owner Party per SR-01. Residual and savings bases come from provider evidence; unsupported values stay null with a reason. The DB-09 event matrix (B0583) is the acceptance matrix; unsupported events are recorded as unsupported.

Opening balances: `OpeningBalanceReference` (local QA-09 name; source `BalanceReference` B1484) keyed `(Scope, authority_system, authority_account, issuing_legal_entity, customer_relationship, currency_or_unit, credit_kind, migration_key)` with previous ledger, as-of, evidence, approval and covered historical transaction range (B1481.R006, QA-09). Per currency and issuer, opening + increases − applications − expiries ± explicit reversals = closing (B1485). Where an approved cross-issuer transfer rule exists (FA-02, HB-23 derived proposal), typed transfer-in and transfer-out movements enter this identity as increase and decrease types. Historical transactions inside the covered range cannot be re-imported as increases.

Marketplace (conditional on an adopted Marketplace profile; product Marketplace selling remains later scope, local DB-14):

| Entity | Required data |
| --- | --- |
| `MarketplaceTransaction` | Buyer/reseller/seller role, direct/marketplace/distributor channel, transaction type; product, offer, agreement, subscription and order IDs with original references; acceptance, service start/end, renewal, billing schedule, currency, price basis (B0695.R001, R003, R004) |
| `MarketplaceParticipant` | Software provider, seller, reseller, document issuer, collector and settlement payee (B0695.R002) |
| `MarketplaceBenefit` | Source-applied discount, coupon or credit and commitment-eligible amount; eligibility `applied`, `not_applied`, `unknown` with evidence (B0695.R005, DB-10) |
| `SellerSettlement` | Gross, fees, refunds, tax, holds, net settlement, payout document and status (B0695.R006) |
| `MarketplacePolicyRevision` | Cost basis, additional rate applicability, benefit attribution, fee pass-through contract and approval revision (B0695.R007) |

Contract end and final work: `ContractState` (`active`, `suspended`, `ended`) is independent of `SettlementState` (`pending_final`, `settled`) (B1475). `FinalEvidenceAcquisition` (HB-23 derived proposal name) records, before source permission revocation, each attempt and result; when acquisition is impossible an `EvidenceGap` (HB-23 derived proposal name) records reason, owner and follow-up path. Emergency security revocation is not delayed for evidence (QA-02). Ending a contract never sets `settled`; pre-termination usage and late credits remain correctable.

Inquiry: `BillingCase` keyed `(Scope, case_id)` with document, line, disputed Money, reason, evidence refs, customer-visible answer, internal memo, owner and response due (B1487). States (HB-23 derived identifiers translating the B1487 sequence) `received` → `investigating` → `adjustment_proposed` → `approved` → `correction_linked` → `resolved`. Customer consent is a separate field (B1488). Financial effects occur only through ordinary adjustment or correction commands linked to the case; a case alone cancels no issued document or payment obligation, and unissued, delivery and ERP recall holds stay distinct (B1488). Customers see only their own documents; MSP cost, other customers' data and internal memos are never visible (B1487).

Decisions: Marketplace adoption, provider commitment profiles, migration cutover date, inquiry response deadlines.

| Case | Scenario and expected result |
| --- | --- |
| H23-S28 | Provider supplies no unused residual for a commitment period: residual stays null with reason; no platform-computed amortization; savings-linked charge is held |
| H23-S29 | Opening balance 500 USD imported with covered range up to the cutover, then a historical credit inside that range is re-imported: balance does not increase; per-currency identity holds |
| H23-S30 | Marketplace benefit with unknown eligibility: stays `unknown`, never `0`; seller fee is not passed to the customer without an approved contract; without Marketplace adoption these records are not required |
| H23-S31 | Contract set to `ended` with a late credit arriving: settlement remains `pending_final` and the credit enters correction; failed final evidence acquisition records a gap with owner |
| H23-S32 | Customer opens a BillingCase: sees own document and public answer only, not internal memo or MSP cost; resolving the case without a linked correction changes no amount |

## FA-12: ERP, issuance, artifact and delivery relationships

Source: B0096.R007, B0155, B0843, B0849, B0850, B0852, B0858.R001–R006, B0866, B0999, B1001, B1493.R001–R006, B1496, B1497; local DB-01, DB-03 to DB-07, DB-12. Extends `Submission`, `NumberSeries`, `IssuedDocument` and `ExportArtifact` in the [logical schema](logical-data-schema.md) and the [ERP and issuance obligations](../../contracts/billing/source-erp-and-issuance-obligations.md).

| Entity | Identity key | Required data and relationships |
| --- | --- | --- |
| `IssuanceModeBinding` (HB-23 derived proposal name) | `(Scope, contract_item, issuing_legal_entity, binding_revision)` | Mode `self_issuance`, `erp_issuance`, `erp_accounting_reference` or `internal_chargeback_statement`, effective from (B0096.R007, B0155, B1497). Change applies only to unissued scope (EI-01) |
| `ERPConnection` | `(installation_id, issuing_legal_entity, erp_company, connection_id)` | Product, version, endpoint, secret reference, capability (B0858.R001); credentials limited to the legal entity (B0866) |
| `MasterMapping` | `(connection_id, mapping_kind, external_key, revision)` | Customer, product, unit, tax, account, organization and sales-order keys with owner, effective interval and version (B0858.R002) |
| `ExistingERPReferenceDisposition` | `(Scope, submission_scope, external_reference)` | Existing sales order, contract, prepaid billing or AP document and its fixed disposition before submission (B0843, DB-03) |
| `ERPFinalDraft` | `(submission_id, erp_draft_id, erp_draft_revision)` | ERP-applied tax, rounding and total, comparison to the approved `final_settlement` subject, recorded permitted difference amount, cause and rule; approved through `erp_final_draft` subject (B0849, B0850). The `final_settlement` approval authorizes the draft stage and `erp_final_draft` the posting stage of one `customer_invoice` submission |
| `ExternalDocumentLineLink` | `(erp_document_id, erp_line_id, revision, charge_line_ref)` | ERP document type, number, revision, related original; many-to-many to platform charge lines; `platform_owned` flag (B0858.R005, B0852, DB-04) |
| `ERPSyncRecord`, `PeriodicLookup` (HB-23 derived proposal name) / integration | `(connection_id, external_object_id, observed_revision)` | Feeds the logical `ExternalResult` and `ExternalMapping`; external status, received time, source update time, response digest, retry; last and next check, owner (B0858.R006, B0866, DB-12) |
| `IssuanceProfile` | `(Scope, issuing_legal_entity, profile_revision)` | Selling/buying entity, address, protected business identifiers, payment terms, document language, PO/contract reference (B1493.R001) |
| `NumberReservation` | `(series_id, number)` | Issuer, document type, series and period uniqueness; draft ID distinct from final number; reason on failure or cancel (B1493.R002, B1497) |
| `DocumentSnapshot` / finance | `(document_id, snapshot_revision)` | Specializes the immutable approved revision of the logical `IssuedDocument`. Body, amount, number, template version and issuance event fixed in one transaction; pre-tax amount, tax code, rate, basis, rounding and total (B1493.R003, R004, B1496) |
| `SerializationPolicyRevision` | `(policy_id, revision)` | Stable sort key, date and timezone, decimal places, encoding, delimiter and sign rules; generator and schema version (B1001, DB-07) |
| `ArtifactGeneration` / reporting | `(document_snapshot, serialization_revision, generation_no)` | Specializes the logical `ExportArtifact` for issued documents. State `requested`, `generating`, `validation_failed`, `ready`, `delivered`, `expired`, `discarded`; stored bytes digest; volatile metadata separate (B0999, B1496) |
| `DeliveryPackage`, `DeliveryAttempt` | `(package_id)` immutable; `(package_id, attempt_no)` | Verified document plus customer CSV/XLSX, recipients, policy and hashes fixed (B1493.R006); attempt channel, recipient and result |
| `ResultRegistration` / integration | `(submission_id, registration_id)` | Specializes `ExternalResult` and writes `ExternalMapping`; manual or automatic ERP result mapping, operator versus ERP confirmation (EI-06), immutable once registered |

Guards:

1. Mode-specific authority: self-issuance only for verified numbering, PDF, tax and correction capability; accounting-only ERP references the existing invoice and never issues a second document (B0155). Unsupported tax conditions block finalization (B1493.R003).
2. Two-stage approval: approved selling amount goes to ERP draft; the ERP result is approved separately as `erp_final_draft`; differing amounts are never auto-aligned (B0849). Without draft protection, automatic posting is unsupported and posted documents are read back (B0850, EI-02, EI-03).
3. A reserved number that fails or is cancelled keeps its reason and is never reused for another document; gapless numbering is not promised unconditionally; ERP mode applies the same boundary to ERP numbers; internal chargeback uses a separate statement series (B1497).
4. Reconciliation covers only platform-owned linked lines; unrelated ERP products on the same document are excluded from platform totals (B0852, DB-04). Existing ERP orders, prepaid billing and AP are dispositioned first (B0843).
5. Stale ERP events never roll state back; webhooks are complemented by periodic lookup (B0866, DB-12).
6. File generation failure regenerates the same document's artifact; it creates no new number or charge (B1496, B0999). Re-download returns stored bytes; generator change is a new serialization revision.
7. Delivery retries resend the fixed package; ERP automatic attachment or send stays disabled unless recipient field and attachment control is proven (DB-05). Standard XLSX remains a common release path (DB-01).

Decisions: per-jurisdiction numbering policy, supported ERP profiles, lookup periods and quotas, template fonts and layouts.

| Case | Scenario and expected result |
| --- | --- |
| H23-S33 | Number N reserved, PDF generation fails: regeneration reuses N for the same document; if the document is cancelled, N is retired with reason and never assigned elsewhere |
| H23-S34 | ERP returns a draft total 1 KRW above the approved amount with no recorded rule: `erp_final_draft` approval blocked; with a recorded permitted rounding rule, approval may proceed with the difference shown |
| H23-S35 | ERP invoice combines a platform line and an unrelated ERP product: reconciliation compares only the platform-owned line; the unrelated product is excluded from platform revenue |
| H23-S36 | Out-of-order webhook reports an older ERP status: state is not rolled back; periodic lookup restores the current status without recreating the invoice |
| H23-S37 | Delivery fails after issuance: only the fixed DeliveryPackage is retried; hashes unchanged; no new issuance |

## SR-01: policy-owning Party and delegation

Source: B0293, B0294.R001–R004, B0297, B0299, B0388; local `requirements.md` policy-owner separation. Refines `PolicyRevision` and `DelegationGrant` in the [logical schema](logical-data-schema.md#identity-parties-and-effective-policy).

| Entity | Identity key | Required data and relationships |
| --- | --- | --- |
| `PolicyRevision` (amended relationship) | Existing policy ID and revision | `owner_party_id` as ScopedRef to Party, `policy_type` (`cost`, `allocation`, `selling` and the existing typed domains), scope, version, `effective_from`/`effective_to`, proposer, approver and evidence (B0299) |
| `PolicyDelegation` / access | `(Scope, delegation_id, revision)` | Typed specialization of the logical `DelegationGrant` for policy tasks. Delegator Party and principal, delegate, permitted tasks, targets, expiry, redelegation permission and parent delegation (B0299, B0388) |

Guards:

1. Approval or change is effective only when, at effect time, the actor acts for the owner Party with the required role, or within a current unexpired delegation from that owner covering the task and target (B0299).
2. Cost, allocation and selling policies may have different owners and are never bound to one administrator permission (B0297). MSP membership alone cannot change a customer's asset cost or allocation policy (B0299); the platform supplier never decides customer policy (B0294.R004).
3. The software owning domain of a policy record is not its business policy owner.
4. Redelegation requires explicit permission and cannot widen the parent scope (B0388). Revoking or expiring a parent delegation ends every child delegation derived from it.
5. No approver, whether owner principal or delegate, can approve a policy revision that the same principal proposed.

Decisions: which persons and roles hold owner authority per customer; delegation templates.

| Case | Scenario and expected result |
| --- | --- |
| H23-S38 | MSP operator without delegation edits a customer-owned data-center allocation policy: rejected; with a valid delegation for that target and period: accepted with delegation recorded |
| H23-S39 | Delegation expires between proposal and approval: approval rejects at effect time |
| H23-S48 | Delegate without redelegation permission grants a sub-delegation: rejected |
| H23-S49 | Parent delegation revoked: a child delegation's approval attempted afterwards rejects; a delegate approving their own proposal rejects |
| H23-S40 | Selling policy approved by the MSP owner Party does not satisfy approval of the customer's cost policy for the same service |

## SR-02: supplier invoice identity and cost-document bridge

Source: B0096.R004, B0163; source tests COM02 (B0579.R002), REC01 and REC02 (B0743.R005, R006); local `requirements.md` document-scoped reconciliation. Conditional on a contract basis or reconciliation that uses supplier documents.

| Entity | Identity key | Required data and relationships |
| --- | --- | --- |
| `SupplierInvoice` | `(Scope, issuer_party, supplier_billing_account, supplier_invoice_id)` | Document identity and document currency, immutable across revisions; a currency change is treated as a different document, not a revision (B0096.R004) |
| `SupplierInvoiceRevision` | `(supplier_invoice, revision)` | Immutable original artifact reference, predecessor and correction links, currency, billing period, total by basis, tax |
| `SupplierInvoiceCostLink` | `(supplier_invoice_revision, cost_fact_ref)` | Many-to-many to `CostComponentFact`/`SourceRevision` with linked account, billing and usage periods, currency, basis and contributed amount (B0096.R004) |
| `SupplierReconciliationBridge` (HB-23 derived proposal name) | `(supplier_invoice_revision, reconciliation_revision)` | Billed cost over full document scope, explained differences by tax, credit, commitment purchase and late item, unexplained remainder; separate MSP selling difference attributed to quantity, base cost, rate, FX, tax and one-off adjustment (B0163) |

Guards: reconciliation is by issuer, billing account, invoice, period and currency (B0163); BilledCost is compared with the full document scope and EffectiveCost is never forced to the invoice total; unexplained difference blocks issuance where the contract requires the supplier document (REC02). A cost component split across documents carries explicit contributed amounts so it is not counted twice. Supplier originals remain immutable; corrections are new revisions (REC01). Links of superseded revisions remain as history; only the current effective revision counts toward bridge and totals. Independent-tariff or internal operation needs no fictitious supplier invoice.

Decisions: tolerance policies, which contracts require supplier-document reconciliation.

| Case | Scenario and expected result |
| --- | --- |
| H23-S41 | Supplier USD invoice for billing month M includes a late item for usage month M−1: the bridge explains it as a late item; completion requires unexplained remainder 0 for the same scope and currency (COM02) |
| H23-S42 | Supplier issues a corrected invoice revision: original revision and its links are retained as history; only the new effective revision's links count toward the bridge and totals, so a component linked under both revisions is counted once |
| H23-S43 | Internal-only enterprise with independent tariff: no SupplierInvoice required to finalize; profitability shows cost unknown |

## Disposition boundary

These bindings close the schema-relationship gaps named in each finding. They do not register typed commands, bind screens, publish the full English clause-to-acceptance trace (FA-14, deferred to [HB-24](https://linear.app/hybrid-billing/issue/HB-24)) or approve customer values. Runtime, database, provider, ERP and UI behavior are **NOT_RUN**.
