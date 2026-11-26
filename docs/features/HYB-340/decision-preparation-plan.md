---
type: ImplementationPlan
title: Authenticated financial decision preparation
description: Current action-specific checks for candidate-backed charge decisions.
status: draft
---

# Scope and ownership

HYB-340, machine:mac_mini, after b91cbea. Root owns server implementation, tests and evidence; refusal_review independently reviews design and code read-only. Owned files: financialcharge/commands, new financialprepare implementation/tests, trusthold read allowlist and affected catalogue tests, scoped contract/evidence. B09 controls follow this authenticated producer. Existing owner services remain unchanged.

# Read boundary and result

Register financial_charge.prepare as a non-mutating, non-Approval command requiring approval_review View approval_evidence for the actual human and ConsoleClient. Require the selected request digest as an input. Enforce this read boundary before binding lookup; do not extend the actual-human mutation exception or grant decision permission through read access. CSRF/session/origin and command audit/rate controls remain unchanged. Allow preparation during trust hold because it causes no financial effect.

Under the server domain lock validate the service-owned binding and supplied digest. Missing/corrupt/mismatched evidence produces an existing non-success refusal, without partial result. Return a fixed bounded map: mode, receipt=current_decision_preparation, request, digest, recorded_state, author, requester, amount, currency, amount_basis=rated_candidate_total, policy_revision, checked_at, revision, approve, reject. No source rows, raw supplier Cost, rationale or new authority grants. Revision is the current global revision; preparation creates no new revision, reservation or approval. checked_at identifies this read, not a guaranteed future validity interval.

Each action outcome is a closed string: eligible, permission_denied, self_approval_denied, authority_unavailable, evidence_stale, dependency_unavailable, unsupported_capability, prerequisite_failed, reauthentication_required, hold_active, amount_authority_missing or not_pending. Use actual gate.CheckApproval with the original author and charge_approval resource; independently exclude requester. No console decision grant is fabricated. If permission is refused, return its action outcomes without probing current producers. For authorized decision actors, require pending state and rebuild the exact service-owned subject/current single-approver policy using a shared helper also used by fresh mutation. Preserve unavailable versus stale distinctions. Then apply action-specific hold (approval blocked, rejection allowed), step-up, and Registry.CheckDecision. Approval requires current amount/currency/stage/date coverage; rejection checks a valid representative closed reason and requires no amount coverage. This only predicts action eligibility; the later rejection command still validates the user's actual selected reason and rationale.

The later mutation still repeats all permission, session, step-up, hold, revision, candidate/policy and module checks. Preparation is not a mutation receipt or execution authorization. No successful outcome is cached across requests.

# Acceptance

Actual authenticated HTTP tests must prove exact amount/author/requester/policy/revision; read permission separate from decision permission; original-author/requester exclusions; stale selected digest and terminal state; revoked/missing amount authority permits reject only; stale producer/policy; hold and step-up action outcomes; missing binding and denied read nondisclosure; unchanged request and financial revision after repeated prepare; permission/evidence change after prepare refuses mutation. Retain existing transport regression, module checks and command catalogue/hold contracts. Independent source and evidence review must pass before checkpoint. No screen, deployment or full W-23 completion is claimed.

# Independent design conditions

Independent refusal_review passed this design before code. Required implementation details: use future mutation names for step-up, preserve authentication refusals, fail closed on unexpected module outcomes, check output bounds, and document clock bookkeeping. The existing24-field/1024-value financial handler preflight bounds the fixed15-field preparation response. checked_at is a wall-clock observation taken after checks, not the module monotonic watermark or a validity lease. Module reads can advance watermarks; normal command audit/rate state and role expiry sweeps remain existing infrastructure behavior. Preparation itself creates no business revision or financial effect. Unsettled stranded effects produce dependency_unavailable. Policy replacement supersedes existing pending requests, so these report not_pending before producer checks.
