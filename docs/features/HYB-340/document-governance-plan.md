---
type: ImplementationPlan
title: Governed explicit document packets and actual ERP preparation
description: Replace ungoverned packet retention with atomic dual-control records before exposing authenticated preparation.
status: draft
---

# Goal and ownership

HYB-340, machine:mac_mini, follows e17984d. Root is sole writer; refusal_review provides independent financial/security design and source review, platform_refusals reviews evidence/terms. The completed permission provisioning does not confirm any document. The intended full connection is explicit packet proposal, independent confirmation, revocation, separate Cost inspection and actual internal ERP preparation under the dedicated rights. No external submission, posting, issuance or consumption is authorized.

# Single ownership and first execution boundary

Source investigation recommends extending financialERP.documents as one domain-locked record store; pricingpolicy interval/overlap and inputreview publication contracts do not fit this service-specific packet. Each record retains canonical basis bytes/basis digest, a distinct versioned proposal digest binding basis digest plus actual proposer/rationale, proposed timestamp, pending/confirmed/revoked state and immutable confirmation/revocation actor/reason/time. Existing owners remains only module-submission ownership. No second packet store or publish-then-govern sequence. Every new record validates permissions, complete packet/current evidence, representability and capacity before one map insertion; IDs never recycle.

First implement and verify this real internal service boundary with existing authenticated grant fixtures; remove retainDocumentLocked's ungoverned insertion path. Root owns new financialdocumentgovernance/authority files and tests plus narrow financialerpprepare changes. Existing internal preparation fixtures must obtain actual separately provisioned grants and independently confirm their packet. This checkpoint does not yet publish HTTP commands or construct an ERP runtime fixture; those remain required immediately following integration.

# Permissions and current/historical semantics

Proposal requires actual current direct-human ProposeDocumentEvidence plus console intersection before record lookup. Confirmation requires ConfirmDocumentEvidence, a different actor from the original proposer, current stored proposer's direct proposal authority checked as subject facts (never a fabricated session), exact proposal digest and unchanged candidate/organization/charge approval/ERP evidence. Preparation requires current PrepareERPRecord plus console, confirmed nonrevoked record and the existing complete currentFinancialChargeLocked/current ERP checks before any owned recovery. Preparation may be performed by proposer if separately granted. Later loss of the historic document confirmer's permission does not automatically revoke the packet; explicit revocation controls it. Current financial charge approver authority remains mandatory through the existing helper.

Revocation requires RevokeDocumentEvidence and exact proposal digest; it may close pending or confirmed records without current producer checks or proposer authority, including during hold. It retains all prior events and cannot delete or cancel a prepared ledger record. Identical same-actor/reason terminal revoke returns history; another terminal decision conflicts. Confirmation replay repeats current checks and never revives revoked state. Identical proposal rechecks current evidence and returns retained state without reopening it. Mutations require bounded nonblank UTF-8 rationale. Enforce monotonic event time and preflight before changes.

Historical packet inspection independently requires actual human, nondelegated ViewCost/document_evidence plus console before lookup. It returns a defensive typed copy of recorded basis/events with current eligibility explicitly unchecked. Revoked records and uncertain already-created ERP records stay historical; revocation cannot erase unknown ownership or imply absence/cancellation. A later public receipt replay is history under fresh permission, while fresh-key preparation/recovery repeats current evidence/state.

# Following public service contract

Strict bounded JSON packet input uses existing4096-byte command fields, rejects unknown/duplicate/case-alias/null/lossy Unicode and missing mandatory values. Caller supplies named synthetic profile, explicit evidence reference, owned charge request/digest, connection selector, independently expected net/tax and line source ID/product/quantity/unit/tax-code/tax/evidence. Server derives candidate identity/digest, original charge rule/kind, exact organization DocumentKey, current connection/mapping/survey records and net amounts. Caller cannot submit authoritative snapshots or default quantity/tax. Retain current documentSubmission validation, complete source bijection and lossless scale.

Runtime constructs one explicitly named synthetic ledger/connection/mapping/survey fixture through actual module APIs, but no packet, confirmation or default permission. SeedNone is unsupported. Authenticated document_evidence.propose/confirm/revoke/inspect and erp_preparation.prepare require their closed capabilities, current sessions, hold/step-up/revision/idempotency boundaries and pre-effect output bounds. Inspection must fit allowlisted canonical packet and event output before proposal admission. Historical module preparation inspection needs separate Cost permission; it never implies current eligibility. Pending document proposals get their own fallible impact category distinct from permission proposals. Review concrete wire/assembly/schema before public implementation.

# Acceptance and scope limits

Verify real separately provisioned actors, dual-control, proposer authority loss, immutable aliases and changed-ID reuse; pending/confirmed/revoked transitions, confirmation/revocation replay and clock rollback; missing/foreign/no-permission lookup; stale input/pricing/organization/charge approval/mapping/survey; exact approved packet preparation; revoked packet before prepare and after unknown module effect; unchanged financial revision/consumption inside private helpers; bounded capacity without orphan bytes. Run appropriate unit/race/vet and full regressions, independent review and correction. Public commands, runtime assembly and later UI remain explicit incomplete requirements until their own end-to-end evidence exists. Do not replace this full integration objective with completion of the private checkpoint.

# Independent internal design review

refusal_review passed the single-store private boundary before implementation, requiring permission before lookup, distinct digests, current proposer subject facts, hold-safe revocation, explicit clock refusal and preserved unknown-effect ownership. The public wire/runtime contract remains a separate required review; its overview above is not implementation approval.
