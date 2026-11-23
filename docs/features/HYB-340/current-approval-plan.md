---
type: ImplementationPlan
title: Current charge approval prerequisite inspection
description: Revalidate recorded approving authority and owned producer facts without impersonation or consuming an approval.
status: draft
---

# Scope and evidence

HYB-340, machine:mac_mini. Root owns implementation and integration; refusal_review independently reviews financial/security behavior; platform_refusals reviews evidence and terminology. The B09 decision screen checkpoint is a7b18a1. The next bounded outcome is an internal current-charge prerequisite adapter, not ERP readiness, invoice issuance or an execution endpoint. No screen changes or new external connector are included.

Independent discovery found that approvals.Registry.CheckCurrent rechecks module policy, amount authority and exact subject, but not the recorded approver's current gate Approve grant. gate.CheckApproval requires the actual approver session and cannot be called using an invented answer. Ordinary auth.Explain intentionally rejects human-only actions. A historical approved badge or successful Execute replay cannot replace present authority. No assembled ERP/issuance producer was found in the server; their document, tax, quantity, protected-field, numbering and payment-term inputs remain distinct from a rated candidate total.

# Narrow subject inspection

Add auth.Registry.InspectApprovalAuthority(subjectID, authorID, request) returning the existing Explanation type, never Decision, Stamp, session or bearer authorization. Under its read lock, derive the current subject epoch and reuse the existing consistent/checkApproval path. This preserves exact Approve-only request validation, active human status, current direct grant, membership/scoped grant matching, author existence and self-separation. Return only explanatory allowed/code/reason facts; ordinary Explain and authenticated CheckApproval semantics remain unchanged. The method observes current authority without authenticating the named person or authorizing a caller.

Expose a narrow gate.Gate.InspectApprovalAuthority wrapper with availability refusal and the same service-owner/creator author-separation rule as authenticated CheckApproval. Read the identity subject generation before and after the combined inspection/ownership checks; refuse a changed generation. No identity Answer is constructed and no session is opened. The wrapper returns Explanation with no usable stamp. This is internal service inspection, not a new unauthenticated HTTP route or permission bypass.

# Service prerequisite adapter

Add a private helper in financialcurrent.go, invoked only while the existing server domain lock is held by a separately authorized consumer. Inputs are workspace, owned request ID, selected request digest and the actual consumer actor identity; never caller-supplied amounts, policy, author, currency or digest maps. Validate the authenticated caller/current session and workspace, then resolve financialChargeRecord and the selected digest. Require the supported single-approval recorded state and actual approving decision actor, distinct from original author and requester. Check that actor's current direct gate approval authority before current producer reads. Rebuild owned candidate/policy facts through checkFinancialChargeEvidence, refuse stranded state, construct Execution from the retained subject and call CheckCurrent. Its ID is only a validated check identifier, never a reservation or effect ID.

Return a defensive current subject/prerequisite result only after all checks; do not call Execute or Report, modify the binding, create a financial revision, reserve a target or grant issuer authority. The future caller must independently enforce its actual action's console/user permission, step-up and hold, keep the domain lock through dependent mutation, repeat current checks at the final effect boundary, and separately reconcile uncertain execution outcomes. A successful prerequisite check is not transferable or durable approval proof. No new HTTP command is proposed in this increment.

# Owned files and verification

Root may edit new auth/gate inspection files and focused tests, new server financialcurrent.go and tests, and this ticket's contract/verification records. Independent reviewers remain read-only. Preserve one registry, existing permission dispatch and all historical receipt behavior. No explanatory source comments or external dependencies.

Test active scoped direct authority, wrong action/kind/scope, unknown/service/inactive subject, missing/revoked/expired grants, membership changes, author/self/service-owner separation, unavailable authority and generation changes. Prove no stamp/session can be obtained and ordinary Explain still refuses Approve. At the service boundary cover actual approved candidate success without consumption; pending/rejected/foreign/corrupt bindings; gate grant revoked while module limit remains valid; module limit revoked with gate grant valid; changed producer/policy; inactive approver; and successful historical execution followed by current revocation. Concurrent state changes must be ordered by the existing domain lock rather than a cached result. Run scoped unit/race/vet and affected server regression, then independent re-review. Explicitly report unexecuted external/persistence/downstream paths.

# Review gate

This plan records the proposed next increment and independent discovery, not implementation or design acceptance. Concrete design review must resolve the helper's consumer authorization contract and subject-inspection generation semantics before code changes.

Independent refusal_review passed this design with explicit implementation conditions: validate the real current s.answer through gate.Admit and match workspace/consumer ID; do not require the consumer to possess Approve for this inspection. The private helper does not authorize a downstream action or expose HTTP. Bracket complete gate inspection including owner/creator checks by identity generation, preserve unavailable errors and discard internal stamps. Test mismatched consumer identity and a genuine non-approver consumer without implying issuance permission. The reviewer inspected source and plan, not executed checks.
