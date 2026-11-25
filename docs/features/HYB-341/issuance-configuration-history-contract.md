---
type: ContractDesign
title: Contract-scoped retained issuance configurations
description: Pure historical discovery of reusable configuration and interrupted approval work.
status: implemented
---

# Outcome

HYB-341, machine:mac_mini. An operator selecting a contract must see its existing issuance configurations, including pending and uncertain work, instead of recreating setup. Root owns implementation; independent review covers this new disclosure. Existing context and authority reviews are reused.

# Read contract

`issuance_setup.configurations` targets the selected contract ID with optional `after` and `snapshot` strings. It uses the same current protected Inspect authority, exact Cost field, session/CSRF and synthetic boundary as context. It is a pure read during hold. Historical disclosure does not require current configuration eligibility, unchanged profile versions or continued standing of its original governors. Current reader authority remains mandatory.

Return at most 10 sorted configuration rows per page from the bounded 100-record adapter. Scope is exact workspace and selected contract across retained revisions. Each row contains configuration ID/digest/state, configuration and source revisions, document mode/profile/region/tax/zone/languages, original proposer/confirmer IDs and observed timestamps. No protected-reference contents, raw party data, recovery execution or current action permission is returned. Report current eligibility as unchecked. A missing current metadata record does not erase retained history.

Validate retained schema, identity, digest, derived kind/languages, frozen scope correspondence and unique scope-index ownership before projection. Observe the actual issuance kind without mutation. Preserve proposal_unknown and confirmation_unknown states regardless of a matching module observation; only their original mutation path may reconcile them. Absent module evidence is valid only for a retained proposal_unknown with no observed proposal. Proposed/active observations must exactly match their retained snapshots and valid state relationships. Do not adopt unrelated module evidence or discard pending work.

Snapshot identity binds reader/workspace/contract and all matching retained observations. Continuation requires a matching snapshot and an existing member cursor. A fresh read starts without both. Refuse corruption, ambiguous ownership, excessive record count, malformed cursor or output overflow without partial rows. Reuse established 4096-byte row and 65536-byte serialized response bounds.

# Verification

Cover proposed and confirmed records, both interrupted stages before/after module effects, historical profile/actor changes, unchanged retained state and server revision, corrupt digest/index/module/scope, exact foreign-contract exclusion, pagination and changed snapshot, reader revocation, hold and response bounds. Public writes and full operator completion remain separate pending work.

# Frozen validation implementation

Frozen scope validation creates a disposable organization registry solely from captured workspace, party and profile facts. Original captured versions must be nonzero and match the retained specification. Party ordering, exact referenced-party coverage and legal-parent consistency are checked. The contract copy uses revision 1 only for isolated shape validation; its actual retained revision remains unchanged and must match the specification. The derived document key must equal the captured key. This validates historical structure without reading or changing today's organization registry, and does not reconstruct missing revision history.

# Review and initial verification

The independent refusal_review reviewer returned scoped design PASS and then source PASS for the retained-history implementation. The review specifically covered frozen validation, disposable revision normalization, reverse ownership, interrupted-state observation and snapshot pagination. Prior context/authority reviews were not repeated. Root added the review's focused test gaps to the same correction pass: another authorized viewer's cursor, projection-budget refusal and an observed kind changed to a different nonmissing snapshot.

Initial `go test ./apps/api/server -run '^TestFinancialSetupConfigurations' -count=1` passed in 1.812 seconds before those additional cases. Its real private-write fixtures exercise proposed/confirmed and four interrupted stages, profile suspension and governor-right revocation after capture, metadata absence during held history reads, 11 distinct retained contract revisions, changed later-page approval, rehashed frozen-scope corruption, orphan/duplicate reverse ownership, missing/unobserved module and lookup failure. Reads preserve retained state and server revision. No real issuance, browser verification or operator-completion claim is made.

Final `go test -race ./apps/api/server -run '^(TestFinancialSetup(Context|Configurations)|TestHoldTableIsExplicitAndComplete)' -count=1` passed in 7.564 seconds, including the added focused cases, the existing context endpoint after shared-handler integration and hold-table completeness. This is the final tested code scope; no repeated independent source review was needed because the additions were test coverage only.
