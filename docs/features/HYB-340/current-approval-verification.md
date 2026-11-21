---
type: VerificationRecord
title: Current charge prerequisite inspection
description: Non-consuming current authority and owned evidence checks without impersonating a recorded approver.
status: draft
---

# Scope and result

HYB-340, machine:mac_mini, follows B09 checkpoint a7b18a1. A new internal prerequisite adapter checks that a recorded charge approval still has current direct gate authority, current module amount authority and unchanged owned producer/policy evidence. It creates no execution, reservation, decision, financial revision or issuer permission. No HTTP route or screen is added, and no ERP/issuance consumer is assembled by this increment.

Auth InspectApprovalAuthority derives the current named subject epoch and reuses the existing consistent direct-human approval checks under the registry lock. Its Explanation contains no stamp or authenticated answer. Ordinary Explain still refuses Approve; authenticated CheckApproval remains unchanged. Gate inspection preserves service-author creator/owner separation and brackets the complete inspection by identity-generation reads. Changed identity generations refuse, while authority-unavailable remains distinct. No session is opened for a historical approver.

The private currentFinancialChargeLocked helper requires the real current s.answer, a matching actual session/consumer/workspace and gate.Admit. It validates only the owned binding and supported approved single-decision shape, excludes original author/requester, inspects the recorded approver before producer reads, rebuilds current candidate/policy evidence and calls module CheckCurrent with an entirely server-derived Execution shape. That shape is a non-consuming check, not an effect identity. A genuine authenticated non-approver consumer can inspect these prerequisites without gaining issuance authorization. A future protected consumer must enforce its own user/console permission before lookup and hold the domain lock through its dependent mutation.

# Independent review and corrections

Independent refusal_review identified the gap before implementation: module CheckCurrent does not inspect gate grants, ordinary Explain refuses human-only actions, and a historical approver must not be impersonated. It passed the concrete plan with current-answer, generation, no-stamp and separate consumer-authorization conditions. Source re-review passed, requesting exact refusal codes and a revoked-grant plus unavailable-producer regression. Root added those and the final test review passed. The reviewer inspected source/tests without executing them. Independent platform_refusals also passed the contract, plan, implementation and race-evidence review, preserving the non-consuming and no-impersonation boundaries; that review preceded the final full-suite result.

The first auth test compile used RevokeGrant without its expected version argument; the fixture call was corrected. An initial server test accidentally referred to the operator function rather than seedOperator; it now uses the actual inspected user epoch and seedOperator. These were test compilation failures, not passing policy checks. No production behavior was weakened to make tests pass.

# Executed validation

Initial auth/gate suites passed 0.402s/0.494s and initial current-service tests passed 1.134s. Expanded targeted tests passed 0.096s/0.096s/1.570s before the final domain-lock concurrency test. Final full auth/gate race suites passed 2.881s/3.843s. The focused server race run covering CurrentFinancialCharge, FinancialCharge, FinancialPreparation and ApprovalProjection passed 12.466s, including the concurrency case. Vet passed for auth, gate and server. Logs: /private/tmp/hyb340-current-modules-race.log, /private/tmp/hyb340-current-server-race.log and /private/tmp/hyb340-current-vet.log. The full go test ./... -short suite subsequently passed all 81 packages, including server 134.711s (/private/tmp/hyb340-current-full.log).

Tests cover exact subject/defensive digest copies without consumption, active scoped direct authority, wrong action/customer, unknown/service/inactive subjects, author/self/service-owner exclusion, removed membership, revoked/expired grants, unavailable authority, generation changes, real consumer mismatch, missing actual caller, pending/rejected/missing/corrupt/foreign requests, current policy replacement, missing producer, separate gate and amount-limit revocation, refusal precedence and historical successful module execution whose current gate grant is later revoked. The synthetic historical execution test explicitly records a fixture outcome to demonstrate historical replay; it is not actual ERP, issuance or payment evidence.

The concurrency test orders an inspection and direct test-fixture revocation through the shared domain lock. It proves that contract's ordering, not protection against arbitrary module writes outside the lock or a fully assembled downstream effect. Candidate/financial module clock and session freshness bookkeeping may advance; no financial revision is produced.

# Remaining boundary

W-23 remains active. ERP and issuance need independently governed document, line, tax, quantity, external capability, protected reference, payment-term and numbering producers. A rated USD15 candidate cannot be relabeled gross, zero tax or ERP not applicable. Eventual effects must recheck prerequisites, separately authorize the actual action, invoke Execute only at a real effect boundary and reconcile uncertain outcomes before Report(Succeeded). Durable persistence, vendor compatibility, real accounts, owner acceptance, merge and deployment are not established here. No new UI render is required for this private backend-only increment.
