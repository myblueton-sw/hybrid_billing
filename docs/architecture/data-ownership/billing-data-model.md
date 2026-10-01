---
type: Logical Data Model
title: Billing tenancy organization and data ownership model
description: Defines tenant and customer relationships, canonical entities, revisions and transactional billing invariants before physical schema selection.
status: draft
---

# Logical data model

Part of [HB-7](../architecture-readiness.md). Proposed refinement of local v1.16 `domain-contracts.md`, `retention-access.md`, `payment-terms.md` and `erp-recognition-cadence.md`; these local files remain unpublished. Entity names, keys and relationships below are logical proposals. No DDL, migrations, index implementation or production data exist in this package.

## Installation tenant customer and organization

| Concept | Meaning and relationships | Integrity and access boundary |
| --- | --- | --- |
| Installation | One customer-operated product deployment; owns operational configuration and generation | Deployment operator is distinct from tenant financial authority |
| Workspace | Proposed tenant security and business-policy boundary inside an installation | Explicit memberships and scope; do not equate workspace with company/payer/customer |
| Party | Legal entity or contracting actor with separately governed legal/tax/payment references | A company may act as MSP, buyer, seller or group entity; role does not grant access |
| WorkspaceParty | Explicit association between a workspace and a Party/legal role | Party identity may be shared only by an explicit policy; private profile fields remain scoped |
| CustomerRelationship | MSP servicing/selling relationship, customer reference, roles, contract and effective/recorded time | Two customers may have similar names or the same legal actor without becoming the same tenant |
| OrganizationNode | Business hierarchy within a workspace: legal-unit reference, department, project or cost center | Parent-child structure is not implicit access inheritance; prohibit cycles and invalid parents |
| OrganizationGroupMembership | Effective-dated grouping for reporting/allocation | Groups may overlap; use canonical input IDs once; no double counting or new access grants |
| DelegationGrant | Explicit MSP/customer/group access: grantor, actor/client, allowed actions/fields, resource scope, expiry and chain | No blanket MSP/root-company access; no re-delegation unless allowed; current grants checked |
| SourceAccountAssignment | Source account to customer/org/contract mapping with effective interval and recorded revision | Provider payer is not customer identity; attribution history preserved after moves |

Proposed supported patterns: an MSP workspace manages customer records with customer/resource-scoped policy; a customer can have a separate workspace for independently governed operations through explicitly shared artifacts/relationships. A single enterprise uses one workspace with its own legal actors/org tree; a group can use scoped legal entities in one workspace or separate workspaces when isolation requires it. Choosing the initial pattern is an owner decision. Do not create every customer as a nested tenant automatically.

Cross-workspace sharing is a separately authorized view/export/reference contract with both source disclosure and recipient permissions; ordinary foreign keys cannot grant it. Customer end users see only authorized customer resources and permitted selling/allocated fields, never all MSP cost or margins. Hierarchy navigation, customer lists and relationship labels cannot reveal inaccessible records/counts.

Organization changes use draft impact preview → validation/approval → atomic activation of a complete revision. Keep business effective time separate from recorded time; forbid unsupported overlapping assignments and test retroactive corrections. Old calculations, approvals and issued documents retain fixed attribution snapshots. Large revisions activate only after complete validation; moving an org node does not transfer outstanding liabilities, grants or issued documents by implication.

## Core entity families and ownership

| Family | Canonical entities and relationships | Authority/storage role |
| --- | --- | --- |
| Identity/access | User → Identity(namespace, issuer, subject); User ↔ Workspace through Membership; RoleBinding/DelegationGrant; Session/Credential; RevocationDecision | Transactional identity/policy authority; secret references or verifier digests only |
| Customer/contracts | CustomerRelationship → ContractRevision → Rate/Formula/PaymentTerm revisions; seller/buyer Party references; contract and account effective history | Transactional business metadata; stable historical revisions |
| Sources | Connection → SourceAccount → SourceIdentity → SourceRevision → SourceArtifact; Coverage and CapabilityEvidence | Transactional identity/coverage registry; permitted source bytes in evidence store |
| Normalized inputs | SourceRevision + MappingRevision → NormalizedPartition; InputManifest contains partition/version/digest/count/unit/currency totals | Immutable verified detail and manifests; analytical copies are derived |
| Calculation | BillingRun binds InputManifest, contract/org/allocation/FX/engine revisions; CalculationPartition → ResultManifest → ReconciliationItem | Immutable detail plus authoritative completion/publication metadata |
| Claim and credit | BillingClaim → ClaimApplication; CreditBalance/Reservation/Application/Reversal; AdjustmentLink references original economic scope | Transactional business uniqueness/reservation authority |
| Approval/documents | ApprovalRequest → ApprovedSnapshot → IssuedDocument → DocumentLine; DocumentLine ↔ ClaimApplication through explicit links; NumberSeries | Atomic approval/version and numbering metadata; fixed document bytes |
| Receivables | ReceivableEvidence/Revision → ReceiptEvidence → ReceiptAllocation/Reversal; invoice PaymentTermSnapshot and original/effective due dates | External authority and freshness explicit; local metadata does not create a bank cash ledger |
| Recognition | RecognitionBatch/Line → purpose-specific Submission → ExternalPosting; RecognitionBillingLink/Reconciliation and approved adjustments | Monthly recognition separate from invoice issuance/payment; accounting mappings UNRESOLVED |
| External handoff | Submission → SubmissionAttempt → ExternalDocumentMapping; Outbox/Inbox records; DeliveryAttempt and manual result evidence | Transactional intent/result-unknown authority; external confirmations distinct |
| Retention/recovery | ArtifactDependency, PolicyRevision, LegalHold, DeletionRun/Tombstone, ControlWatermark, InstallationGeneration | Durable authoritative controls plus per-store deletion evidence |
| Intelligence | AnalysisRun/InputSnapshot → Finding/CorrectionProposal/AdjustmentProposal/Forecast; ProposalDecision and evaluation evidence | Derived analysis; accepted proposal links to ordinary authorized business command |

CustomerProfile contains scoped contact/address/legal reference and ERP customer mappings. Payment/tax/recipient changes use verified provenance, effective revisions and separate approval; models never invent them. Secret payloads and raw personal records do not belong in this Git model. Optional personal favorites reference canonical customer IDs and remain subject to current access; this proposal does not close HB-6 PC-07.

## Identity keys revisions and money

- Canonical IDs identify installation/workspace/customer/source/contract/run/document independently of display names, email, provider IDs or broker offsets. Tenant-owned references include matching ownership context; globally unique IDs alone do not enforce isolation.
- Source identity includes provider/authority, billing scope and stable source identity with explicitly versioned correction semantics. Missing stable identity is a reconciliation problem, not permission to infer uniqueness from an ingestion request.
- API idempotency is installation/client/command/key plus input digest. Business Claim uniqueness instead follows issuer/responsibility, customer/contract, economic service/component and covered source/period/quantity scope. The exact key and partial-coverage representation require domain approval; adding a new run, revision or request key cannot reopen consumed scope.
- Preserve event, received/recorded, business-effective, approved, issued and externally confirmed timestamps. Proposed effective periods use `[from,to)` with explicit timezone/calendar contracts. Null/missing coverage is unknown, not zero.
- Every quantity has unit and unit revision; every monetary amount has currency, amount basis and precision/rounding policy reference. Provider cost, selling price, allocated cost, tax and recognized amount remain distinct. Never sum currencies or mix net/gross bases without approved conversions.

## Atomic invariants and failure semantics

| Invariant | Proposed enforcement and failure behavior |
| --- | --- |
| Tenant integrity | Scoped authenticated commands and ownership-aware references; equivalent controls in workers/search/cache/export/analytics/models; negative cross-tenant tests |
| Complete input | Publish only fully verified manifests; quarantined/partial data cannot be final input; stored bytes and manifest registration are not assumed atomic |
| Approval integrity | Current authorization, expected revision and approved digest checked with the command transaction; changed inputs require new review/approval |
| Claim uniqueness | Durable economic identity and conditional reservations/applications; replay and partial reversal cannot create overlapping consumption |
| Credit/receipt conservation | Transactionally prevent overspend/over-allocation; currency-specific remaining balance and explicit reversal links; unallocated receipt is not invoice settlement |
| External uncertainty | Record intent/outbox before external dispatch; timeout means unknown until external lookup/reconciliation; no blind retransmission |
| Recognition conservation | Reconcile like-for-like monthly net with billed amount and explicit adjustments; already included adjustment cannot be deducted twice |
| Historical integrity | Original source and issued document revisions remain immutable; corrections create linked new revisions/documents, never hidden overwrite |
| Safe restoration | Validate latest deletion/revocation watermark and fence old writers; reconcile external outcomes before re-enabling actions |

If external ERP processes can concurrently consume the same credit or receivable balance, local locks cannot reserve that external value. Require a verified external reservation/conditional-application contract or an approved reconciled/manual path; unresolved external balance authority blocks automatic application. Confirm freshness and external consumption before claiming settlement or an available balance.

PostgreSQL constraints can enforce keys/references; cross-row balance sums require locked/conditional or serializable transactions rather than a row `CHECK`. Select and verify concurrency controls for each command; retry the whole failed transaction without repeating external effects. [Constraints](https://www.postgresql.org/docs/current/ddl-constraints.html), [Isolation](https://www.postgresql.org/docs/current/transaction-iso.html).

## Physical model and access paths to validate

Apply the owner-requested [usage detail lifecycle](../usage-detail-lifecycle.md): preserve source-to-measure-to-calculation-to-document contribution links, historical manifests and current-policy projections. Summary, complete permitted detail and exports agree only at the same scope/revision/currency/basis; maintained storage includes retained dependencies, verified archive and control-aware recovery.

Design indexes from actual reads: workspace + customer + status + stable cursor; source identity/revision; effective-dated assignment; contract/run/manifest; Claim scope; document/number; external key/purpose; job next-attempt/lease; session/revocation; retention due/hold/dependency. Compare detail partitioning by source period/workspace against query skew, cardinality and retention; do not create a physical partition for every customer without evidence.

Partitioned unique constraints must retain the real business uniqueness. PostgreSQL partitioned primary/unique keys must include partition columns; use a suitable unpartitioned registry or validated routing scheme when economic uniqueness spans periods. [Partitioning limitations](https://www.postgresql.org/docs/current/ddl-partitioning.html). Analytics denormalizes canonical customer/org/contract revisions, not mutable names as join keys. Projections are rebuildable from retained eligible evidence and respect tombstones.

Open decisions: exact types/precision/nullability; initial tenancy profile and lawful customer profile fields; source and Claim keys; financial authority/mappings; allowable effective-time overlaps; SQL isolation; schemas/partitioning/indexes; session store; historical/PII retention; driver/version and HA layout. Logical review is not executed schema or database validation.
