---
type: ContractDesign
title: Selected-contract context for issuance setup
description: Initial authorized context read resolving server-owned contract and billing-profile versions.
status: implemented
---

# Boundary

HYB-341, machine:mac_mini. Root owns implementation. This read is one prerequisite of the complete O03 setup journey; it neither configures issuance nor exposes protected-reference values, payment terms or mutation readiness. Existing inspection-capability review remains valid; this new public disclosure requires its own independent review.

`issuance_setup.context` is a nonmutating command. Target is the selected contract ID; body contains only required `on`, an exact ISO calendar date. Use the normal session, CSRF and command envelope. Require the protected domain's exact ViewCost evidence permission and current governed Inspect ownership before command execution and again in Apply. Administrator status, contract viewing and mutation rights are insufficient. The read is permitted during installation hold but never changes or reconciles retained work. Only the synthetic sample boundary is supported.

# Source resolution and response

Under the server domain lock, resolve exact workspace-scoped metadata and its linked profile, then the lifecycle position on `on`. Require an active position and nonzero contract terms revision. Load that exact organization revision and the current billing-profile version. Validate their full correspondence with configurationScope, including active workspace/profile/parties, external invoice path and document key. Do not select the greatest revision, accept client profile IDs, repair mismatches or substitute today.

Return string fields: schema, contract, contract_name, on, lifecycle_sequence, contract_revision, billing_profile, billing_profile_name, billing_revision, buyer, payer, issuer, currency, scope_digest, profile_basis=current, current_eligibility=unchecked. The profile reflects current authoritative state, not a reconstructed historical profile at `on`. IDs are derived selection context, not operator input fields. The digest binds the date, lifecycle sequence and position, metadata contract/profile identity and the full authoritative configuration scope. Later write commands must re-resolve and compare it before effects.

The response contains exactly 16 fields. IDs retain the existing validRequestID constraints; every value, including names, is at most 4096 bytes and the serialized field map is at most 65536 bytes. A configured transport below the established 64-field/4096-byte minimum refuses with UnsupportedCapability; output overflow refuses with PrerequisiteFailed. Do not return raw party attributes, bank/email contents, protected evidence, module snapshots or a claim that issuance is ready. Missing metadata returns PermissionDenied after authorization, preserving the command framework's non-disclosing absent-target behavior; inactive lifecycle is a prerequisite refusal; malformed dates are invalid requests; missing or mismatched organization evidence is stale evidence; source read faults are unavailable dependencies.

# Verification

Use actual command HTTP handling with independently approved human/console Inspect grants. Check before/after lifecycle change dates, current profile-version changes, digest stability and changes, foreign/mismatched metadata, wrong profile correspondence, inactive/not-started/ended states, malformed dates, missing source, output bounds, explicit field grants, unauthorized existence probing, grant revocation, held reads, unchanged source and command revision, and non-sample refusal. No browser test or screen acceptance is claimed. Design/source review is scoped to this new boundary.

# Scoped review

The independent refusal_review reviewer returned design PASS with explicit conditions, then source PASS for the new endpoint and its command/hold integration. The resulting code checks dependencies before contractRead, forwards exact protected evidence authority, binds the complete scope and requested-date lifecycle position, and returns only the 16 bounded fields. The only review correction was the test's expected HTTP status for prerequisite failures: the established command contract uses 422. No production change or repeat review was required for that assertion correction.

Root's first test run additionally exposed two invalid fixture assumptions: existing billing profiles freeze currency, and a fresh non-sample installation has no sample administrator login. The revised tests exercise real profile suspension/reactivation and the non-sample command boundary using an already authenticated fixture. These corrections do not relax production validation. Full operator setup, mutation-time context comparison, retained setup discovery and O03 screen integration remain pending.

# Executed logic verification

`go test -race ./apps/api/server -run '^TestFinancialSetupContext' -count=1` passed in 5.693 seconds after fixture corrections. Actual HTTP cases verify 16-field disclosure, date-selected revisions 1 and 2, repeat-read digest stability, current profile suspension refusal and reactivation version 3, no configuration reservation or server revision change, unauthorized administrator/member probing, held authorized reads, grant revocation and ended-session refusal. Refusal cases cover malformed dates, missing/foreign metadata, mismatched profile, suspended party, nil dependencies, injected contract-read failure, inactive lifecycle, output-name overflow and reduced transport budget. A non-sample boundary refusal was tested by switching fixture configuration after authentication; this is not a fresh non-sample installation journey.

No browser verification or design-screen acceptance was performed. Existing inspection caller tests remain evidence for delegated and ineffective-grant refusal; this endpoint run does not claim a second execution of those tests.

The adjacent hold-table completeness check found three previously added draft mutation commands missing from the explicit table (159 registered mutations versus 156 entries). The default fallback already refused those commands. The same increment adds explicit holdRefuse/reasonChange entries for submit, prepare_approval and approve and updates the completeness expectation, preserving denial while restoring the required exhaustive inventory. This newly evidenced delta receives a bounded review; the passed context source is not re-reviewed.

The reviewer returned scoped PASS for the three explicit refusal entries without repeating context review. `go test -race ./apps/api/server -run '^(TestHoldTableIsExplicitAndComplete|TestHoldAllowsDecreasesOnly)$' -count=1` passed in 2.568 seconds after that correction, covering the exhaustive table and actual held mutation refusals.
