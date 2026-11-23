---
type: ImplementationPlan
title: Authenticated document and ERP preparation service
description: Connect governed packets to bounded public commands and a named synthetic runtime ledger.
status: draft
---

# Ownership and scope

HYB-340, machine:mac_mini, follows 959fec9. Root writes and integrates; refusal_review independently reviews financial/security design and source, platform_refusals reviews evidence and terms. Owned files are new document command/output and ERP seed implementations, narrow server/command/hold/impact/preparation integration, related tests and this ticket's documentation. Existing sample data stays synthetic. No external connector, posting, issuance or approval consumption is introduced.

# Public command contract

| Command | Permission | Input besides common envelope |
| --- | --- | --- |
| document_evidence.propose | ProposeDocumentEvidence | packet JSON; target is document ID |
| document_evidence.confirm | ConfirmDocumentEvidence | proposal digest |
| document_evidence.revoke | RevokeDocumentEvidence | proposal digest |
| document_evidence.inspect | ViewCost/document_evidence | proposal digest |
| erp_preparation.prepare | PrepareERPRecord | document ID and proposal digest; target is preparation key |
| erp_preparation.inspect | ViewCost/document_evidence | target is owned preparation key |

Resolve validates only workspace and target syntax. The service boundary checks actual human/session/console rights without delegation before cached replay. SeedNone and absent runtime refuse unsupported capability. Mutation retains existing CSRF, step-up, rationale, global revision and idempotency handling. Each successful fresh-key mutation returns the next revision and historical receipt; identical cached-key replay returns its recorded revision after current authority checks. Fresh preparation repeats current packet, financial and ERP checks. Hold refuses propose/confirm/prepare and allows revoke and historical reads.

# Bounded output before effects

Document mutation receipts contain mode, document, digest, recorded_state and receipt. Full inspection adds basis, basis_digest, current_eligibility, proposed_by, proposal_reason, proposed_at, confirmation_by/reason/at and revocation_by/reason/at: seventeen fields maximum. Basis is canonical complete JSON, bounded by min(4096, configured value limit). IDs are bounded by existing 128-byte syntax, digests 64, timestamps 35 and reasons 2000 bytes. Require at least seventeen fields and 2000-byte values. Before proposal retention, preflight the complete projection including worst-case future confirmation/revocation metadata. Recheck that complete bound before each decision; governance separately validates actual actor and rationale bounds. Never truncate or omit evidence to fit.

ERP historical inspection has eleven fields: mode, preparation, document_digest, recorded_receipt, observed_digest, ledger_observation, current_eligibility, submission, total, mappings and actor. JSON fields have the same 4096/configured limit. Before ownership insertion or Prepare, preflight this projection including bounded future digest fields. Preparation mutation returns only bounded historical identifiers/digests/state. Inspection requires current Cost rights, owned keys and exact observed submission/actor/totals/mappings/prepared state, with no external command/reference/divergence. It never calls reconcile or changes ownership. Observation is matching_prepared, missing, unavailable or conflict; none settles an unresolved recorded receipt or certifies absence of an earlier effect. Historical reads remain possible after document revocation and producer staleness.

# Runtime and impact

An ERP failure after invoking Prepare is an uncertain effect, never a clean dependency refusal. A plain non-command uncertainty error makes the command engine retain an operation reference and unknown result. Owned recovery lookup failures and contradictory post-effect receipts preserve the same uncertainty. Actual permission/current-evidence refusals still precede recovery. The same HTTP key replays its historical unknown result; a fresh command key with identical pinned document and preparation intent can reconcile the existing ledger record. Historical inspection never settles it. Private fault assertions and public operation/recovery tests must distinguish these cases.

A distinct sample startup step constructs the actual erpledger registry, named synthetic-document-erp connection, explicit customer/product/tax mappings and October 2026 synthetic survey. Organization attribution comes from sample-charge-01 solely as fixture data. Seed no packet, confirmation or document permission. Add a narrowly named decimal-import source allowance and update the expected sample revision from 24 to 25. A separate fallible document_evidence_proposals category counts pending documents by proposer; confirmation rechecks proposer authority. Tally errors fail closed. Screen labels and rendered multilingual review are a following required UI change, not implied by API completion.

# Verification

Exercise actual HTTP proposal/inspection/independent confirmation/ERP preparation/readback/revocation using real provisioned actors and ordinary startup ledger. Test permission-before-replay, Cost isolation, self-confirmation, wrong digests, missing/foreign records, current authority loss, exact retry, unknown effect ownership, hold/step-up/CSRF/revision, insufficient output bounds and oversized admission with no retained effect. Verify startup has no default packets/grants and SeedNone refuses. Update catalogue/hold/source invariants, run scoped race/vet and full short suite, and obtain source/evidence re-review. Keep public-service and UI completion claims separate.
