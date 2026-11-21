---
type: ImplementationPlan
title: Server-owned candidate-backed financial subjects
description: Exact StageCharge subjects from current governed v7 candidate evidence.
status: draft
---

# Scope and ownership

Continue HYB-340 after77ff07c, machine:mac_mini, with root as sole writer. Owned files are financialsubject.go and focused tests, typed input-freshness extraction in candidatecapture.go, a non-storing normalized-input builder extracted from approvalnormalized.go, shared exact rating composition in approvalproducer.go, pure approvals.PrepareSubject validation and these records. Independent refusal_review accepted the concrete design before implementation with the conditions below. Existing HTTP capture, historical v5/v6 bytes and grant governance remain unchanged.

The internal helper runs under the caller-owned server domain lock and accepts only workspace, request ID, candidate ID and expected candidate digest. It does not authorize a caller or expose an endpoint. Future financial transport must independently authorize the actual caller before lookup and perform freshness validation plus Request under the same domain lock; current evidence is never permission. No Request/Decide effect is added in this increment.

# Current evidence and exact subject

Require an immutable retained v7 candidate with valid original CaptureAuthor, matching workspace/digest, complete input/pricing provenance and a fully rated result with no held or excluded items. Recover the service-owned capture binding. Check current approved pricing evidence through CheckEvidence, current input publication/review/digest/revision and recomputed reconciliation, then rebuild the complete normalized input through the existing source/mapping/attribution/lifecycle/row pipeline without storing a candidate. Compare it to the retained input, recompute the exact rating result and full envelope digest, and refuse any mismatch.

Require a currently confirmed service-owned financial policy at the current StageCharge revision. Do not silently adopt the sample demonstration policy. Derive exact amount and currency from the verified result and Author from immutable CaptureAuthor, never from the requester. The initial subject is aggregate-only, without entry exclusions. Pure module PrepareSubject validation must reject values outside its numeric profile without mutating requests or advancing its clock.

Digest meanings are versioned: input binds current complete input provenance; rule binds the approved pricing record; mapping binds target/mapping keys, revisions and digests; attribution binds contract/profile/document/lifecycle and current pricing target; result is the full verified candidate envelope digest. Select exactly the current policy's required keys. Policies requiring fx or erp_draft refuse until real corresponding producers exist; empty or identity-rate placeholders are forbidden.

The economic target hashes workspace/customer/entity/contract/billing profile/provider/account/period/from/to. It excludes request/candidate IDs, source run, manifest, mapping/pricing versions and currency, so replacement candidates for the same economic scope supersede rather than coexist. This is replacement within one provider/account billing scope. Changing billing profile, provider/account or period creates a different target and will not supersede the earlier request; cross-scope obligation deduplication is not claimed.

# Acceptance

Verify the real governed USD15 v7 fixture, immutable original author preservation, v5/v6 and mismatched-envelope refusal, current source/mapping/input/pricing/target changes, distinct candidates sharing one economic target, current policy replacement and unavailable FX/ERP refusal, exact module numeric preflight, defensive outputs, and no request/candidate mutation on failed checks. Existing capture regression must preserve original bytes and behavior. Later request/decision/execution transport must repeat external freshness checks; approval-module CheckCurrent alone is insufficient.
