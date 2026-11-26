---
type: ImplementationPlan
title: Current decision preflight for financial approval
description: A non-mutating module check preceding separately reviewed authenticated preparation and screen controls.
status: draft
---

# Scope and ownership

HYB-340, machine:mac_mini. Root is sole writer. Independent refusal_review owns read-only financial design and code review. This increment adds Registry.CheckDecision and shared fresh-decision validation in modules/approvals only, with focused tests and verification. The subsequent financial_charge.prepare endpoint and B09 controls remain separate implementations under the active goal. No permission, policy, grant or execution effect is added.

# Behavior

CheckDecision accepts the same workspace, request ID, decision and actor inputs as Decide. It validates input, workspace existence, exclusion item membership, human actor, pending state, original author/requester separation and current exact amount/stage/currency/date coverage for approval. Rejection retains its existing closed reason validation and does not require amount authority. Approval with exclusions must leave at least one item. It does not store exclusions, deciders, decisions, approvals or request state; the registry's existing monotonic clock bookkeeping is allowed.

Factor the fresh-decision checks into one lock-held helper shared by Decide and CheckDecision. Preserve Decide's existing successful historical same-actor/same-decision replay and conflicting replay behavior. CheckDecision must never return eligibility from historical replay: an actor already recorded as a decider is refused even when a multi-approver request is still pending. Unknown exclusion items remain not-found before replay, preserving existing Decide ordering. The check is not a reservation; a later Decide revalidates current state and coverage under its own lock. Authentication and actual human permission remain server responsibilities.

# Acceptance and follow-on contract

Verify no request mutation on successful or refused checks, requester/author/system refusal, invalid reasons and exclusions, exact limit boundary, expired/revoked currency/stage/date authority, rejected/superseded/approved states and repeated-decider refusal on a still-pending multi-approver request. Prove Decide historical replay is preserved while preflight refuses it. Run module regression and race checks and obtain independent re-review.

The follow-on authenticated preparation endpoint will retain B09 read permission, independently check actual current human approval permission and module eligibility for each action, rebuild service-owned candidate/policy evidence, expose exact bounded historical facts and explicit current outcomes, and return the current revision without requiring administrative list access. Hold/step-up and ambiguous mutation recovery require explicit designs before screen controls. Preparation does not authorize a later effect or replace command-time checks.
