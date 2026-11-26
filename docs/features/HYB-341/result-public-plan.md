---
type: ImplementationPlan
title: Public synthetic ERP result commands and protected read projections
description: Expose reviewed result helpers through current-session commands while preserving unknown effect ownership and historical evidence boundaries.
status: draft
---

# Ownership and dependency

HYB-341, machine:mac_mini. Root is the sole writer; refusal_review independently reviews the command/security design and code, platform_refusals reviews evidence and terminology. This increment depends on the reviewed private [ingestion core](result-ingestion-verification.md). Reuse the actual financialERP instance and commands engine. No new approvals registry, external connection or default result evidence. No new screen control is implemented before the separate screen design/render review.

# Command contract

Add result_evidence.propose, confirm, revoke, inspect and history, plus erp_result.apply and inspect. Proposal takes the original packet field and an actual command rationale; target is the immutable result ID. Decisions/application require the proposal digest. Mutation replies are small historical receipts. Cost-protected inspection returns original source, canonical packet, retained comparison/preparation identity, immutable proposal/confirmation/revocation events and application state/receipt separately from current eligibility. Never describe a historical receipt as current permission or external confirmation. History is scoped to an owned preparation and exact recorded receipt, sorted by result ID with the existing bounded cursor/snapshot convention. It includes retained revoked and mismatched proposals. Unknown application remains visible without exposing raw errors or synthesizing settlement.

All commands bind the actual session/workspace and dedicated result namespace before lookup; mutations require current human/console intersection, step-up, CSRF, revision, rationale and existing idempotency envelopes. SeedNone remains unsupported. Explicit hold rules refuse propose/confirm/apply and allow revoke/protected reads. Keep the exact hold inventory and source rules current.

The command engine can return cached outcomes without invoking the handler. Therefore add a result boundary before Execute: current dedicated rights are always rechecked; confirm rechecks original proposer standing and revoked state; apply rechecks confirmed/matched evidence, current document/financial facts, original proposer/confirmer standing, watermark and actor ownership before any cached success. For an already recorded application, also re-read the coherent ledger snapshot and verify the stored receipt digest without writes; a cached success must not mask later changed rows, links or revisions. Perform this read-only under the same domain lock, never through the mutating apply helper. Preserve the uncertainty sentinel as an unknown handler failure with an operation reference, not a clean commands.Fail refusal. Revocation/inspection must remain usable under stale financial producers. A cached unknown operation remains unknown; use a new HTTP command request with a new idempotency_key, new request_id and fresh expected_revision, retaining the same immutable result ID/digest/actor to invoke owned recovery. Never change the retained ledger operation identity or discard the old unknown command record.

# Bounded output and effect ordering

Derive configured field/value requirements from the actual projections and preflight them before every mutation. Bound each serialized packet/basis/source/event and each history row to the current command value limit, with exact JSON-escaped event maximum sizes, not merely unescaped rationale lengths. Do not mutate and then discover that inspection or receipt cannot be represented. Prefer separate fields over a large unbounded aggregate JSON value. Return no raw exception or credential data. Apply must preflight the eventual small public receipt before entering private ownership/write logic. The private application record remains authoritative when command journaling fails after a possible effect.

# Read and impact semantics

Dedicated Cost inspection grants do not imply document or financial view permissions. Expose only the retained result scope needed by the explicitly protected result command. Preparation history must validate owned immutable association, not current producer eligibility, so revoked/stranded evidence remains auditable. Inspection is read-only and never repairs ownership or records a receipt.

Add fail-closed result evidence impact enumeration, distinguishing pending proposals from unresolved applications. Count pending proposals by their original proposer and unresolved applications by their applying actor; these are distinct work categories, not a deduplicated total. Label recheck semantics from the actual result contract, not permission-proposal approval wording; do not silently drop them because the proposal was revoked or superseded. Avoid counting completed receipts as pending work. Enumerate the application map independently, validate reverse external/preparation ownership and reject orphaned or corrupt records. A missing registry or invalid retained record makes enumeration unavailable, not zero. Label each store accurately in EN/KO/AR; any user-visible impact label change triggers the existing rendered review loop.

# Acceptance and remaining goal

Exercise actual HTTP commands and actual registry effects: independent confirmation, mismatch retention, inspection isolation, malformed packet/bounds before mutation, current rights before exact cached replay, cached success after changed ledger evidence, same-key/new-request-ID remaining unknown, stale document/charge/mapping, self confirmation with both rights, hold/SeedNone, wrong workspace/digest/actor, unknown after Document and each Link, same-request cached unknown versus new-request owned recovery, stable history cursors, unresolved impact counts and fallible tally. Run focused/race/full regression as warranted and independent corrective review.

Replacement revisions, protected readiness/context for the complete screen journey, dedicated result/provisioning controls, actual Claude screen design/render review, StageERPFinal, issuance and delivery remain required under W-23. This plan does not authorize external ERP, owner FIX, merge or deployment.

# Independent design review

platform_refusals passed this bounded command/security design after reviewing actual command lock/replay/finish semantics and existing protected output/history patterns. Root adopted explicit new idempotency-key recovery, read-only completed-receipt validation, unknown-failure preservation, escaped-byte bounds and reverse-ownership impact validation. Its separate terminology pass clarified original proposer versus applying actor counts. These are design reviews, not implemented contract or UI validation.
