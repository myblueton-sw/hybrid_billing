---
type: VerificationRecord
title: Candidate-backed financial subject verification
description: Exact immutable-author subjects with current governed evidence and no financial command exposure.
status: draft
---

# Scope and independent review

HYB-340 continues after77ff07c, machine:mac_mini. Root implemented the internal lock-owned financial subject builder, exact shared rerating, typed current-input extraction, non-storing normalized input reconstruction and pure module subject validation. Independent refusal_review accepted the concrete design and scoped economic target before code. It later found an invalid direct primitive import in financialsubject.go; the correction moves shared exact composition into the already approved approvalproducer.go boundary without widening source rules. Corrective independent source review passed without remaining blockers. No test was executed by the reviewer.

The builder creates no request, candidate, grant or execution and does not authorize a caller. The supersession regression explicitly invokes the approval module only in a test after preparation; this is not HTTP request integration. Future session authorization, RequestApproval provisioning and actual financial decisions remain outstanding.

# Executed checks

Initial subject/module tests passed (server1.704s, approvals0.328s). Source/boundary/subject guards passed2.414s after the primitive-import correction. Final focused race passed approvals1.333s and server11.190s (/private/tmp/hyb340-subject-race.log), covering TestPrepareSubject, TestFinancialSubject, TestSourceRules, TestCandidateCapture, TestCandidateAuthor and TestIntegrationActorsAndEffects. The complete go test -short ./... run passed all80 packages, matched exactly against go list ./... (server131.721s; /private/tmp/hyb340-subject-all-short.log). During that run, separate vet found a mutex copied by a corruption-test fixture. The fixture now uses a pointer; product code was unchanged. Final focused subject/source race passed5.634s and final go vet ./... passed (/private/tmp/hyb340-subject-final-race.log and /private/tmp/hyb340-subject-vet-final.log). The full-run compiled test snapshot predates that test-only pointer correction; the corrected tests are covered by the final focused run. The initial vet failure is not a pass.

Actual governed capture fixtures produce an exact USD15 v7 candidate. Tests verify immutable original author, exact required digest set, defensive maps, unchanged stored candidate bytes/counts and no created request. Two separately captured candidates in one economic scope produce distinct result digests and the same target; explicit module requests then supersede the first while retaining its original author. Pure target tests distinguish economic scopes while excluding producer/currency/version changes. These are internal/module tests, not authenticated financial request or decision journeys.

Refusal tests cover v5/v6/invalid author, wrong candidate digest, corrupt result/input, revoked pricing policy, current billing target changes, document revision, voided reconciliation, corrupt source bytes, foreign review producer, an actual new normalization mapping revision and a separately confirmed newer input publication. Current policy ownership cannot be absent/corrupt; required FX/ERP evidence cannot be fabricated. Pure module preparation tests reject unsupported precision/oversized/nondecimal values, preserve signed exact amounts and defensive digests, and prove no request or clock mutation.

# Limits and remaining work

No screen or UI copy changed, so there is no new browser/visual acceptance claim. Existing capture v5/v6 bytes and v7 author behavior are covered by the affected regressions. The target is replacement within a provider/account billing scope, not global duplicate-obligation detection. Full W-23 remains active with request/decision transport and later financial effects still pending. No main merge, deployment, real-account/customer-data operation, payment or external message occurred.
