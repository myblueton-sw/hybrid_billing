---
type: Domain Design
title: Shared data ownership and business-state contracts
description: Defines canonical ownership identity state command consistency and failure contracts before implementation.
status: draft
---

# Shared domain contract

This logical design remains under the [planning gate](planning-gate.md). Proposed names are not actual tables, DDL or deployed APIs. HB-16 translates the pre-existing local contract without dropping its obligations and integrates the reviewed input/trust/operating and monetary mechanisms. The previous bytes are preserved in the authorized local source snapshot; provenance is recorded in the [remediation record](../../reviews/HB-16-remediation/verification.md). Field lengths and runtime representations require approved profiles; never guess money, identities or dates.

## Data ownership

| Domain | Main entities and facts | Required invariant |
| --- | --- | --- |
| Installation/workspace | Installation, Workspace, Party, BillingProfile: immutable IDs, operating type, issuing/buying parties, state | Workspace, legal entity, group and payer are distinct |
| User access | User, Identity, Membership, RoleBinding, Session, ServiceAccount, Credential | Connection namespace, immutable internal ID, effective scope and revocation revision |
| Organization/source account | OrganizationNode, SourceAccount, RelationshipRevision, GroupMembership | Effective time differs from recorded time; cycles/overlap constrained; approved historical attribution immutable |
| Provider connection | Connection, CapabilityEvidence, DiagnosticRun, Coverage | Secret references only; readiness by actual contract/role/version |
| Ingestion/normalization | SourceArtifact, SourceIdentity, SourceRevision, MappingRevision, NormalizedPartition | Cost differs from sales; identity, revision, completeness and conflicts tracked |
| Evidence | InputManifest, Artifact, Dependency, ContentDigest, InputConfirmation | Immutable inputs, counts/bytes/currency totals, required retained bytes, evidence-bound confirmation |
| Contract/rating | ContractRevision, RatePlanRevision, FormulaRevision, FXSnapshot | Effective dates, units/currencies, rounding/approval revision; no arbitrary execution |
| Allocation/calculation | AllocationPoolRevision, BillingRun, CalculationPartition, ResultManifest | Complete-pool rules, deterministic conservation and verified publication |
| Reconciliation | ReconciliationRun, ReconciliationItem, DifferenceExplanation | Difference classification by document/scope/currency; unexplained discrepancy is not rounding |
| Billing ledger | BillingClaim, ClaimApplication, CreditReservation, CreditApplication, AdjustmentLink | No double consumption/overspending; correction/cancellation history retained |
| Approval/issuance | ApprovalRequest, ApprovedSnapshot, NumberSeries, IssuedDocument | Only approved hash/revision; unique numbers; issued originals immutable |
| ERP/delivery | Submission, SubmissionAttempt, ExternalMapping, DeliveryAttempt, Outbox | External idempotency differs from unknown result; calls occur after authority commit |
| Operations | Job, JobAttempt, Checkpoint, Event, Projection, CloseCalendar, StageSchedule | Technical and business states differ; lease/fencing/resumption and completed scope explicit |
| Retention/deletion | RetentionPolicyRevision, LegalHold, DeletionPlan/Run, Tombstone, DecisionWatermark | Serialize reference/hold races; restored backups cannot revive deleted/revoked data |
| Extension/release | PluginManifest, CompatibilityEvidence, InstallationGeneration, MigrationRun | Fixed digest, no direct core-ledger writes, fence old installation's external writes |

Map ownership to ledger, bulk detail, objects and derived projections without imposing a separate database per domain. References use canonical IDs/revisions, never names, email or array order. Business events originate in the same authority transaction's outbox. Analytics is not the authority for economic uniqueness.

## Identity time and money

Server establishes installation/workspace/principal/current authority for every command. Caller-submitted scope and globally unique IDs do not confer access. Distinguish event time, effective business interval, received/recorded time and confirmation time. Proposed effective intervals are `[from,to)`; validate closing timezone, DST, leap year and delayed source arrival. Storage/display timezone policy is explicit per data class.

Money carries decimal value, currency and numeric-profile reference; quantity carries decimal value, unit and unit version. The [Claim/numeric contract](../../contracts/billing/claim-and-numeric.md) now specifies the proposed bounded exact-decimal representation, explicit rounding/division, typed DSL, complete-pool residual algorithm and partial-coverage transitions. Float round trips, unconverted currency sums and null-to-zero are forbidden. Excess input scale, overflow, NaN/Infinity, negative/refund semantics and unsupported currencies produce explicit errors or governed correction flows.

Select approved rounding stages/modes from source through contract, allocation and document totals. Chunk size and completion order cannot affect final amounts. Actual numeric limits, currency quantum, finance policy and runtime enforcement remain profile decisions; no numeric values are invented by this document.

## Business and technical states

| Subject | Proposed flow | Failure/concurrency behavior |
| --- | --- | --- |
| Collection | received → original stored → normalized_candidate → validating → blocked or awaiting_confirmation → eligible | Atomic evidence-bound confirm/publication after independent complete comparison; partial/unknown scope never eligible; distinguish duplicate recollection from correction |
| Calculation | inputs fixed → calculating → validating → reviewable → approval requested | Failed chunks/inconsistent manifest never published; correction creates new run/revision |
| Approval | pending → approved or rejected/withdrawn/expired | Changed input hash/policy revision invalidates reuse |
| Issuance | approved → submission reserved → dispatching → externally confirmed/issued | Unknown outcome distinct; lookup/reconcile before resend |
| Correction | original fixed → impact preview → approval → linked correction/reversal → reconciliation | Never delete issued original to correct; quantity reversal alone cannot authorize rebilling |
| Job | queued → running/waiting → succeeded/failed/cancelled | ERP uncertainty is waiting with reason; partial scope is business detail, not overall success; cancellation receipt differs from actual stop |
| Account | invited → active → suspended/terminated | Lock, membership end and identity retirement are separate |
| Trust change | draft → validated → independently_approved → activated | Changed proof/config invalidates approval; activation advances epoch and invalidates old authority; rollback requires a newer epoch |
| Deletion | evaluated → held/approved → access blocked → store deletion → backup expiry → policy-complete | Cancellation after deletion starts cannot promise restoration; serialize holds/references |
| Schedule | current → change requested → approved revision applied | Stage/schedule identity fixed; preserve original deadline; unrelated stage and issued payment dates unchanged |

Technical job success does not imply business completion: successful ERP lookup can still report unknown issuance. Apply business transitions atomically with permitted command, current actor and expected revision; audit the actual result. Input confirmation is a prerequisite, not amount approval or issuance authority.

## Shared API and command boundary

The [planning action registry](../../contracts/commands/planning-actions.md) is canonical for the remediated paths: typed input confirmation, independent trust replacement, holds, metrics, favorites and stage deadlines. It defines request/result/error/permission/state mappings used by affected screens and APIs. URLs/OpenAPI generation remain stack-dependent; AI gets no alternate write bypass. Original command families below remain required, including those not yet fully specified in that registry.

| Area | Reads | Planned mutations | Preconditions/results |
| --- | --- | --- | --- |
| Organizations/accounts | Tree, groups, relationship revisions, impact | Create/validate/apply change plan | Expected revision, fixed targets and pool impact |
| Users/authentication | Membership, roles, sessions, connection diagnostics | Invite/suspend/revoke; trust propose/validate/approve/activate | Current permission/MFA/role ceiling, durable revocation acknowledgment; no generic trust activation bypass |
| Provider | Connection, capability, coverage, candidate/comparison | Diagnose/schedule/retry collection; input confirm_publish/reject | Actual scope, original preservation, secret references, independent evidence and current authority |
| Contract/rating | Current/future/prior revision and preview | Draft/simulate/review/activate | Unit/currency/lifecycle and policy-change authority |
| Calculation/reconciliation | Fixed input/results/difference explanations | Schedule/validate/recalculate/reconcile | Input manifest and engine revision |
| Approval/billing | Approval inbox, document, Claim, balance | Approve/reject/reserve issuance/correct | Author/approver separation, valid coverage Claim and snapshot |
| ERP/XLSX | Submission, external state, handoff files | Submit/query/register manual result/reconcile | External key/file revision/unknown-outcome contract |
| Disclosure | Document/artifact/disclosure scope | Generate/publish/deliver/revoke access | Allowed fields/currency, current rights and immutable issued revision |
| Retention/deletion | Policy, holds, remaining evidence/impact/jobs | Revise policy/hold/plan/delete/retry | Approval, current references/holds and per-stage evidence |
| Operations/extensions | Jobs/alerts/install/plugin | Cancel/resume/upgrade plan/restore validation | Generation fencing, actual installation/compatibility evidence |
| Operating interactions | Holds, cohort outcomes, personal favorites, stage schedules | Typed hold request/release; favorites add/remove; schedule change request/application | Independent hold effects, current disclosure, fixed metric definitions, stage-specific approved changes |

Reads return allowed scope, stable sort key, snapshot/revision, cursor/has-next-page, freshness and completeness. Exclude unauthorized counts/totals. Filter-wide operations capture a fixed target manifest and recheck rights. Bulk details cannot be an unbounded JSON payload.

Mutations carry command ID, idempotency key, expected revision, trace/request ID and input digest. Same key and same permitted input returns the prior result subject to current access; changed input conflicts. Economic Claim identity remains separate; credential rotation/new request IDs cannot reopen billing. Durable asynchronous receipt returns job/status and permitted lookup/cancel capabilities, never false completion. User errors identify permitted fields, correction path and retryability without secrets/original customer data. Authentication failure, denial, revision conflict, semantic invalidity, quota/transient outage and external uncertainty remain distinct.

## Concurrency and recovery

Calculation stages chunk results then validates the full manifest. Approval, Claims, balance reservation, submission intent and outbox use the smallest sufficient authority transaction. DB and object writes are not presumed atomic; verify completed bytes before publishing references. Orphan cleanup still checks retention/holds/references.

Two attempts to apply 60 and 50 to credit 100 cannot commit 110. If ERP also consumes that balance, a platform lock alone cannot protect it: external reservation/conditional-application evidence is required. Source identity, Claim and document-number uniqueness can cross physical partitions; validate an identity registry or consistent routing with actual DB constraints/tests later.

An expired worker lease does not stop the old worker. Validate generation/fencing and retain pinned uncertain reservations. Fencing cannot recall an already transmitted external request. Restore/migration isolates old network/credentials/schedulers and reconciles ERP outcomes before reopening. Deletion/revocation ledgers require durable continuity through the latest confirmed watermark; unavailable latest controls keep affected data quarantined.

## Remaining specification obligations

The required contract families remain identity/tenant, provider capability/export, decimal/time/FX, contract lifecycle/formula, Claim/correction, approval/ERP, jobs/events/plugins and retention/recovery. Each design task must finish field types/nullability/validation, permissions, state transitions, error codes, examples, version compatibility and acceptance. This common contract and the remediated registry do not finish every family or select customer values.

## Source-audit boundaries retained

Apply the [35 detailed contracts](detailed-requirements.md) when this summary omits a clause. Required source semantics are not proof of actual vendor support or approved policy values.

| Area | Required detail references |
| --- | --- |
| Sales eligibility | AR-01 direct/resale and payment responsibility; AR-10 independent DC products/SLA approval; DB-10 Marketplace fee/settlement status |
| Input eligibility | AR-02 mutual authentication/Push; AR-03 OCR candidate validation; AR-04 AI result state/denominator; AR-05 no default prompt/body collection; RT-01 envelope/correction/approved-result return |
| Calculation/correction | AR-06 FX selection/markup/fallback; AR-07 explicit adjustment rows and revalidation; QA-09 per-currency opening-balance conservation |
| ERP | DB-03 existing orders/prepayment/AP; DB-04 owned-line reconciliation; DB-05 automated delivery gate; DB-11 no direct ERP DB writes; DB-12 periodic lookup |
| Access | AR-09 delegation chain/actual support actor; QA-04 external user/client intersection; QA-01 existing-channel payment-detail verification and no LLM invention |
| Closure/schedule | QA-02 separate contract and settlement status; QA-03 exemption/original deadline/denominator revision; QA-08 recurrence after resolution is a new event |
| Disclosure/extensions | DB-06 non-executable documents/external-resource blocking; DB-07 deterministic serialization; QA-05 CloudEvents/webhook/expired cursor; QA-06 external origin/vulnerable-version exclusion |

Source metering envelopes differ from external CloudEvents; never merge both into an unnamed generic payload. Asynchronous receipt maps to 202 plus job/lookup, key conflict to 409, quota to 429 with retry guidance. Job technical enums and business-detail mappings remain owned by the Job contract and HB-34.

## Payment terms and independent monthly recognition

Contract Payment Terms (Net 30/45 are examples), issued term/due-date snapshots and confirmed-receipt balance/freshness-driven overdue/aging remain required. See the [English financial contract](../../contracts/billing/payment-and-recognition.md) and preserved local [payment terms](payment-terms.md). O03/I01/I02/B11/U01/R01/U03/C01/C02 and HB-15/22/24/27/28/29/30/31 remain linked; PT-01–08 are NOT_RUN.

Separate customer invoice cadence from monthly ERP recognition; reconcile monthly posted net/approved adjustments with quarterly billing and prevent duplicate cost recognition. Preserve [recognition cadence](erp-recognition-cadence.md) and the English contract. HB-15/21/23/24/25/26/28/29/30/31 and ERPC-01–08 remain linked; actual journal support is UNVERIFIED. No product implementation has begun.
