---
type: ContractDesign
title: Point-in-time setup action inspection
description: Caller-specific setup guidance without granting authority or reconciling retained operations.
status: implemented
---

# Scope

HYB-341, machine:mac_mini. Root implements; refusal_review independently reviews this new disclosure. Reuse earlier mutation/history/options reviews. The O03 client needs actual governed rights, independent actor, hold and step-up guidance rather than administrator-role guesses. This read never authorizes a later write.

# Request and disclosure

Add nonmutating issuance_setup.action targeting the selected contract. Required body: operation (propose_configuration or confirm_configuration), configuration (the actual client-generated or retained configuration ID). Proposal additionally requires on and scope_digest; confirmation requires digest. Reject irrelevant fields. Use the existing current protected Inspect, console human session and sample installation boundary; held reads are permitted.

Return schema hb_issuance_setup_action_v1, contract, configuration, operation, permission (allowed/denied/unavailable), step_up (satisfied/reauthentication_required/authentication_policy_unsatisfied/unchecked), action (recheck_required/blocked/recorded), code, current_eligibility unchecked. No ready/eligible claim. A write still checks all input, immutable rationale/settings, exact revision, source, ownership, module state and authority.

Check actual governor authority before disclosing action details. Check step-up against issuance_setup.<operation> and the configuration ID, not the read command or contract target. If held, report blocked/installation_held. Do not conflate unavailable authority with denied permission.

# Scoped facts

For a new proposal identity, re-resolve exact date/scope digest and validate the configuration index. Detect an existing configuration for the same contract revision/billing-profile version and report blocked/configuration_exists without disclosing another ID. Capacity and low-level mutation admission remain write-time checks.

For a retained identity, require exact workspace/contract correspondence and valid frozen/module observation before exposing action state. Legacy Setup-nil records are blocked/unsupported_capability. Proposal retry is available only to the original proposer and requires the retained date/scope digest; its complete settings/rationale still require write-time equality. Confirmation requires the exact configuration digest, a distinct actor from the proposer and the original confirmer if reserved. Validate current configuration and re-resolve its captured date/digest. Never promote uncertain states because module observation is active. Report recorded for already completed action, recheck_required for a valid pending/recovery path, and blocked for a proposal that has not yet reached the state required for confirmation.

Missing/foreign retained confirmation targets use the same non-disclosing refusal. Reads do not create IDs, mutate retained state, reconcile module evidence or advance revision. Inspection failure retains unchecked eligibility and gives a specific stable refusal code; transient dependency failure does not become permission denial.

# Closed response state machine

Current Inspect/session failure and malformed request return the normal HTTP refusal without action fields. After those guards, evaluate in this order: governor authority, installation hold, actual command/target step-up, retained identity/ownership, frozen observation and current source checks, then completed-action mapping. Stop at the first refusal; never override it with a historical recorded state.

Governor denial returns permission denied, step_up unchecked, action blocked, code permission_denied. Unknown/unavailable governor authority returns permission unavailable with unchecked/blocked/authority_unavailable. All later projections have permission allowed. Hold returns unchecked/blocked/installation_held. Step-up returns reauthentication_required/blocked/reauthentication_required or authentication_policy_unsatisfied/blocked/authentication_policy_unsatisfied; an unexpected session/step-up failure returns unchecked/blocked/authority_unavailable. After successful step-up, step_up is satisfied for all remaining states.

With satisfied step-up, blocked codes are the closed set permission_denied, self_approval_denied, evidence_stale, unsupported_capability, configuration_exists, configuration_pending and dependency_unavailable. Typed source permission/self/stale/unsupported failures retain those meanings; all unexpected observation/source failures map to dependency_unavailable. New occupied scopes use configuration_exists. Retained proposal_unknown cannot be confirmed and returns configuration_pending. Missing/foreign confirmation identity returns permission_denied, with no existence distinction.

After every required check succeeds, recheck_required and recorded both have permission allowed, step_up satisfied and an empty code. Proposal is recorded only for retained proposed, confirmation_unknown or confirmed states, whose completed proposal publication is validated; proposal_unknown remains recheck_required even if module evidence exists. Confirmation is recorded only for retained confirmed with validated publication; proposed and confirmation_unknown remain recheck_required. Active module observation alone never promotes an unknown state. Every response keeps current_eligibility unchecked.

# Verification

Actual HTTP tests cover governed permission versus Inspect-only, independent actors and reserved ownership, exact scope/digest/target, hold and step-up, stale source, legacy closure, occupied scope, unknown-state preservation, absent/foreign identity and no effects. Review only the new read boundary and projection. UI wiring, terminology review and owner screen acceptance remain subsequent work.

# Review and implementation evidence

Independent refusal_review identified one design blocker: unspecified response combinations and precedence. The closed state-machine section resolved that finding and received scoped design PASS. Source review then passed the action read and new context/options client transport without blocking findings. No earlier backend review was repeated. The reviewer executed no tests; controller-level late-response suppression and UI acceptance remain outside this slice.

Initial action HTTP tests passed in 0.768 seconds. The final setup and hold-completeness race suite passed in 11.979 seconds, including added stale-source, absent/foreign-target, legacy and malformed-input cases. Tests verify no unknown-state promotion, proposal/confirmation publication distinction, independent and pinned actors, unchanged state after reads, hold precedence and expired step-up. Console context/options client tests plus prior context decoder tests passed (7 total), and typecheck passed. The client checks actual session identity, request/target correlation, response bounds and exact selected context, preserving synthetic-only applicability. It is not yet wired to the documents view.
