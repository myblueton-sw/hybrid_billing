---
type: ContractDesign
title: Governed issuance configuration proposal and confirmation
description: Public setup writes tied to exact selected-contract context and original actors.
status: implemented
---

# Behavior

HYB-341, machine:mac_mini. Root owns implementation. A governor selects a contract/date and explicit supported document settings, proposes one reusable configuration, and a different governor confirms it. The owner generates no internal identifiers manually; clients generate new request/configuration identifiers and retain them for retry. This initial-revision workflow neither edits existing configuration nor issues a document.

`issuance_setup.propose_configuration` targets a new configuration ID. Body: contract, on, scope_digest, mode, profile, region, tax, zone and comma-separated languages. Normal command rationale and mutation envelope are required. `issuance_setup.confirm_configuration` targets that configuration with its digest. Both require current protected Inspect plus existing independent governor authority, command step-up and explicit hold refusal. Administrator role alone must not replace governed inspection.

Before effects, resolve the requested contract/date with the existing context path and compare scope_digest. Derive the profile and source revisions server-side. Retain the original on/scope_digest in an optional setup context on the configuration; include it in the immutable configuration digest. Omit nil legacy setup context from serialized records so prior private-record digest input remains unchanged. Deep-copy the optional context. Legacy configurations remain readable but are not silently adopted by public writes.

Confirmation and original-actor recovery re-resolve the retained setup context. Pre-cache checks preserve current rights, immutable proposal arguments, original proposer/confirmer and independent confirmation. Another actor cannot take over an interrupted write. Later source changes refuse further mutation without rewriting retained history.

# Effects and receipts

Track actual retained changes, scope ownership writes and module attempts per invocation. A typed failure before effects remains a clean refusal. An error after a possible effect returns an uncertain operation; retained evidence supports exact original-actor recovery. A read never reconciles work. Output admission precedes the first business mutation, covering the bounded receipt for each possible success state. A post-effect projection failure remains uncertain.

Receipts retain configuration identity/digest/state, exact document/source settings and original setup context. Historical unknown states are never relabelled successful merely because the module contains matching evidence. Existing private proposal/confirmation wrappers retain their behavior when no public setup context or tracker is supplied.

# Verification scope

Actual HTTP tests must cover independently governed Inspect for both governors, self-confirmation, wrong actor, changed context/arguments, stale revision, hold/step-up and revoked rights before replay. Inject failures before/after retained intent, module effect and publication; distinguish clean no-effect refusal from uncertainty, preserve exact-envelope behavior and fresh owned recovery. Check receipt admission before effects, optional-context deep copies and legacy digest serialization compatibility. Source review covers only this new mutation boundary and necessary private instrumentation; passed read implementations are reused.

# Review and implementation evidence

The independent refusal_review reviewer returned scoped design PASS with conditions, then source PASS for the mutation boundary and required private instrumentation. The review covered pre-cache authority/identity, captured-context revalidation, legacy closure, hash-bound copied Setup, invocation-local effects and pre-effect receipt/history admission. Existing read/capability reviews were reused. No UI or complete operator workflow approval was implied.

The implemented preflight reserves space for the longest retained state, maximum actor ID and all observed timestamps, including unknown history rows. Public history includes original setup date/digest and proposal rationale for exact proposal recovery. Nil Setup omits the new JSON field; an explicit old-structure byte/digest comparison checks legacy compatibility. Malformed partial Setup is refused before writes.

Initial `go test ./apps/api/server -run '^(TestFinancialSetupMutation|TestFinancialConfiguration)' -count=1` passed in 3.304 seconds after correcting a fixture that attempted to propose a grant to its own proposer. The fixture now uses distinct grant proposer, approver and recipient for each governor, preserving actual independent grant governance. The run covers private configuration regressions, public independent proposal/confirmation, pinned actors, exact and fresh-envelope recovery at all configured interruption points, legacy-write refusal, context copying/hash binding and receipt/history admission.

Additional edge tests cover changed proposal arguments/rationale, source suspension during an uncertain proposal, preserved unknown history, and revoked inspection/proposer authority, step-up expiry and hold before cached confirmation. Final `go test -race ./apps/api/server -run '^(TestFinancial(Configuration|Setup)|TestHoldTableIsExplicitAndComplete|TestHoldAllowsDecreasesOnly)' -count=1` passed in 14.051 seconds, including the additional edges and explicit refusal-table coverage for both new mutation commands. No deployment, merge, real issuance or browser verification is authorized or claimed by this increment.
