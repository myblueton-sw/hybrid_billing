---
type: ImplementationPlan
title: Governed financial authority and candidate-backed charge approval
description: Child integration of W-23 using the existing approval registry and current governed candidate evidence.
status: draft
---

# Scope and ownership

[HYB-340](https://linear.app/myblueton/issue/HYB-340) is a child of HYB-225, preserving all original W-23 scope. Linear read-back confirms the Hybrid Billing team/project, accountable owner Seung Woo Park, In Progress and machine:mac_mini. Root owns PM/planning/implementation/integration and is the sole writer in /private/tmp/hyb340, branch feat/HYB-340-financial-integration from reviewed HYB-337 eaa8663. The future draft PR is based on HYB-337; the acyclic stack is HYB-225 → HYB-337 → HYB-340. Independent financial/security review and QA are read-only. Actual Claude CLI ui-designer reviews any changed rendered screens. Missing companion inputs retain the existing owner-authorized no-restoration boundary.

Reuse the single approvals registry; implement explicit synthetic financial governance, stage/currency/amount authority and v7-backed requests/decisions with actual session authorization. Never adopt demonstration policy values, invent old candidate authors, or substitute pricing approval for financial approval. Durable storage, later stages, protected changes, recognition and remaining W-23 screens remain under HYB-225. No production policy, main merge, deployment, real-account/customer-data use, payment, external ERP effect or message permission is granted.

# Reviewed governance boundary

Independent refusal_review inspected the existing module/session/read contracts before execution. The first connected boundary is StageCharge financial policy propose/confirm/inspect/current against s.approvals. Current installation-administrator membership, active human identity and actual user/console AdministerInstallation authorization are mandatory in addition to module governance membership. Confirm rechecks the stored proposer under the actual caller's authority. Historical module replay cannot bypass these checks. No financial decision grant or amount ceiling is created by policy confirmation.

Policy fields are explicit: expected next revision, approval count, validity hours and exact required digest kinds. Unsupported stages and malformed/noncanonical fields refuse. Module rules remain authoritative for required result evidence and count/validity limits. Record immutable rationale and decision evidence in a bounded service binding. Refuse unknown/internal demonstration proposals for service mutations. Existing command CSRF/session/step-up/global revision/idempotency/hold boundaries remain. Preflight bounded response representation before mutation. Pending policy proposals have a separate fallible M01 impact tally; unavailable is never zero.

Root may edit narrow module read/history APIs and tests, a new server financial-policy adapter/commands/tests, assembly/hold/step-up/effect inventories, pending-impact integration, B09 historical-policy projection/tests, scoped existing EN/KO/AR impact copy/QA and these records. One writer per file. Independent reviewers cannot edit. Do not create a second registry or silently change generic financial approval dispatch.

# Historical policy and read consistency

Legitimate policy replacement supersedes earlier requests. The current B09 reader incorrectly requires each historical request's revision to equal the current policy. Add defensive ProposalOf/Proposals and confirmed PolicyAt reads from the existing bounded proposal history. Do not remove revision validation or construct historical policies from caller input. B09 validates the exact recorded revision and its supported one-approver shape; new unsupported multi-approval requests still refuse. A superseded formerly-approved request retains its historical single approval actor/time, with superseded status and end timestamp, while authority/prerequisites remain unknown. CheckCurrent and Execute still evaluate current policy and are unchanged.

# Acceptance and remaining boundaries

Verify two actual governors; self-confirmation; revoked/suspended/removed proposer or confirmer; ended sessions and missing console authority; malformed fields, stale digest/revision and conflicting confirmations; historical replay versus fresh refusal; defensive copies/bounds; no automatic financial grant; SeedNone; truthful B09 history after policy replacement; pending-impact failures and hold/step-up/source guards. Run appropriate module/server race, full relevant regression, console checks and actual synthetic browser paths with scoped independent code/TW/visual re-review.

Subsequent reviewed increments now provide separately governed financial Approve grants and module stage/currency/amount limits, current pricing evidence and internal server-derived v7 StageCharge subjects (see financial-subject-plan.md). The subsequent transport increment now connects a narrow actual-human CheckApproval path and repeats prerequisite revalidation before Request/Decide. See financial-transport-plan.md and financial-transport-verification.md. Downstream consumption revalidation remains outstanding. Subject author is the original capture author even when another user requests approval. Required FX/ERP evidence cannot be fabricated. Financial requests/decisions are implemented in synthetic HTTP commands; B09 listing and explicit current-checked decision controls are now integrated (decision-screen-verification.md); downstream effects remain outstanding. This plan is not full W-23 completion evidence.
