---
type: ImplementationPlan
title: Nonmutating issuance admission before cross-registry execution
description: Add current draft admission checks without reserving a number, recording a document or consuming approval.
status: draft
---

# Purpose and ownership

HYB-341 follows the [issuance investigation](issuance-integration-investigation.md). Root is sole writer of modules/issuance/admission.go, issue.go, registry.go, admission_test.go and this plan. refusal_review independently reviews the design and change. The first dependency for the later final-approval execution transaction is complete module admission before reserving either registry effect. This does not replace service-owned issuance drafts, mode/profile/protected-field/term composition, public rights, recovery or delivery.

# Proposed contract

Add CheckIssue(workspace, id, digest, actor) error and CheckExternal(workspace, id, digest, external, actor) error. Each holds the existing registry mutex and current monotonic clock, validates input/actor and exact current draft digest, rejects already-issued documents even for the original issuer, and checks existing approval/rules/protected references/payment coverage at the relevant instant. External checks retain the existing explicit external number/time, ERP-only class, no-future/no-preapproval instant and duplicate external number checks. Internal checks retain approved series and capacity. Existing reservations do not need another number slot; new number allocation checks width and workspace number capacity without incrementing either.

Success is advisory at the observed instant, not a reservation, execution token, proof of external issuance, current caller grant or approved final amount. It never creates a draft, reserves a number, writes a document, consumes approval or calls Renderer. The clock's monotonic observation may advance. Caller-owned providers must be read-only as already required by existing Resolve/Cover/Detail use. The service must retain its own domain lock and repeat current session/final approval/issuance-right checks when connecting these modules.

Factor shared internal/external admission helpers and the number-capacity check from the existing Issue/RecordExternal/take paths so refusal behavior does not drift. Preserve existing historical Issue/RecordExternal replay behavior: their already-issued branch remains before current admission. CheckIssue/CheckExternal deliberately refuse historical documents; callers must use DocumentOf plus exact owned reconciliation for recovery. No existing mutation signature changes.

# Verification

Verify no document, number allocation, series increment or external-number claim occurs on passing checks, repeated checks or refusals. Cover changed draft digest, missing approval, changed protected references/terms, number and document capacity, exhausted width, existing reservations, external uniqueness/class/time and actual-module mutation after a passing check. Change a prerequisite between check and issue to prove the mutation still rechecks. Cover same-actor historical replay remaining available while current preflight refuses it; cross-workspace and assistant refusal. Run module tests/race, source rules and vet. This is a backend-only prerequisite; no new screen or external operation is claimed.

# Status

Independent concrete design review returned scoped PASS. It required shared fresh admission, historical replay preservation, reserved-number capacity parity, issuance-time reference resolution and explicit nonmutation tests. Root implemented CheckIssue/CheckExternal plus shared internal/external admission and number-capacity helpers. Independent source review returned scoped PASS without executing tests. The later service still must exclusively own registry access and hold its domain lock across preflight and effect: existing Issue/RecordExternal accept no expected draft digest.

The initial module race suite passed2.160s, including retained-state comparisons and changed reference/terms/digest, capacity, reserved-number, external-time/uniqueness, assistant and workspace cases. An initial test compile typo used nonexistent StatePending; it was corrected to the actual StateProposed without changing production behavior. Reviewer-requested external check-to-effect reference changes, missing approval, foreign/assistant paths and actual constructor-bounded number/document exhaustion were added; the final full issuance race suite passed2.312s and go vet passed. Dependency baseline suites passed for issuance0.441s, protected0.440s, receivables0.699s and delivery0.296s.

This implements only the nonmutating module prerequisite. No service-owned issuance draft, current permission adapter, cross-registry execution, external issuance or delivery is completed.
