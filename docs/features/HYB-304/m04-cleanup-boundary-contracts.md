---
type: ImplementationPlan
title: HYB-304 cleanup and admission boundary contracts
description: Verifies stale cleanup callbacks, pickup replacement indexes, revoked held-delivery actions and selection expiry after the existing admission hook.
status: draft
sources:
  - resource: ./m04-service-reference-contracts.md
---

# Reviewed scope and ownership

HYB-304 remains In Progress, Seung Woo Park, machine mac_mini, after pushed/read-back b3295903465e5a437f22df6fcf75922a03c7a101. Root is sole writer. Separate read-only /root/hyb310_review approved four focused test contracts, with production unchanged. No direct registry-map rewrites, new fault dependencies, guard deletion or coverage exception are planned.

Capture a genuine session-end retry produced through the supported unavailable fault, remove that fault and drain actual cleanup. Calling release again with the old pointer must preserve exact remaining units, zero retries, cleanup outcome, retained recovery and the surviving caller. This is helper idempotence, not evidence of an asynchronous callback race.

Install two credential pickups through directory methods, then replace the first credential's reference. The old reference must disappear, canonical lookup must select the replacement, the second pickup must survive and recovery source version must change. This is a directory input/index contract, not a reissue endpoint or permission to extend a real pickup.

Produce a held redelivery, then revoke its subscription through normal supported commands. The queued held redelivery remains observable after subscription revocation. An observer with current approval rights, distinct from requester and approver, must see reapproval blocked by the module ReasonRevoked value, revoked. Reads must not change canonical state or server revision.

Use the existing one-shot accessHook only to advance the actual fixture clock beyond the caller's absolute deadline and clear itself. The captured answer is now past NotAfter; final Peek fails, while current access retains that answer. Preparation must refuse the already-expired new selection with409, release its reservation and create no operation or target termination. The global jump also expires peers; assert stored presence and unchanged operation/accounting effects before probing the caller, then require caller401. Do not assert peerHTTP200 or perform HTTP/open-session calls from the domain-locked hook.

Required100% is unmet at the preceding99.6%/51-uncovered checkpoint. Bounded M04 aggregate reconciliation already passed; overall readiness, authorized merge/read-back and parent M03/M02 remain open. Damaged access-history state is classified ErrConfig/ErrLocked and cannot establish the generic ErrStart fallback. Distinct-verifier truncated-SHA handle collisions, missing owner-index contradictions and unestablished inter-call failures remain unverified; these guards are preserved.

# Fixture corrections before final verification

The first scoped run passed pickup replacement and the actual admission-hook expiry case but failed two fixture expectations. The callback fixture originally resolved the target verifier through the session registry after the actual end had already removed that entry; it now captures the genuine verifier before termination. The revoked-held observation actually retains kind redelivery, hold approver_lost_right and a false reapproval action blocked by revoked. RevokeSubscription cancels the current open delivery; the queued redelivery is a distinct retained object. The test now checks that actual queued-state/action contract rather than inventing synchronous cancellation of the queue. These were test setup/expectation corrections with no production change, not product fail-before regressions.

# Isolated entropy-collision extension

Independent design review corrected the earlier broad collision classification. Installed Go1.25.7 crypto/rand.Read honors its exported Reader override, so repeated entropy can deterministically test random selection-reference collision refusal without editing registry maps. This does not construct different verifiers with the same truncated SHA handle.

Run an exact-match child test process, establish genuine authentication and an unrelated retained record normally, then fully close the owned server to join workers/history. Only that quiesced ledger component briefly receives finite repeated entropy. Restore the original reader before assertions/cleanup. Refusal must preserve the first registered object, unrelated record and exact units. Require a static success marker as well as exit0; distinguish known assertion failure from crash or instrumentation failure and suppress raw child output. This is a closed-fixture ledger input contract, not a live HTTP collision or post-close authentication claim.

A native parent coverage profile does not automatically prove child coverage. During an instrumented run an explicit fresh owned test-only child-profile path may collect the child's native profile. Preserve both profiles and matching source hashes; any union requires identical block ranges/statement weights and must be labeled combined coverage, not substituted for native parent measurement. Required100% remains unwaived.

# Observed scoped and instrumentation verification

The final four corrected contracts passed in0.767 seconds; adding the isolated entropy test yielded five PASS groups in1.231 seconds. Fourteen new and related cleanup, recovery and reapproval groups passed under the race detector in7.073 seconds. Server-package vet passed. Independent actual-source review passed both the four-contract implementation and the isolated child extension; the reviewer ran no tests or services.

Three semantic mutations were assertion-killed for stale retry identity, post-admission expiry and revoked reapproval. A fourth mutation bypassing reference-collision protection failed the isolated child's static assertion, which the parent distinguishes from a process/instrumentation failure. Each production file was restored to its exact SHA, and all private caches were removed before final checks. The actual isolated-instrumentation probe collected separate native profiles: the parent-only launcher measured0% product statements, while the executing child covered1,836 of11,783 weighted statements. Their7,889 block ranges/weights match; this probe is not the full server package measurement. A snapshot binds107 current production sources across probe and full verification.

The terminal full server short suite passed in191.852 seconds with native parent coverage99.6%:11,783 weighted statements and45 uncovered. The isolated child's native profile and the explicit set union are preserved separately. All107 production hashes and7,889 block ranges/weights match. The union also has45 uncovered statements, adding zero statement-coverage units; the entropy test establishes a distinct collision contract rather than an inflated package score. Required100% remains unmet. No product source changed, and prior Console/UI/all-package/foreign-compilation checkpoints were not rerun. The two documentation status tests and separate root metadata/link checks do not constitute full semantic OKF certification. [The cleanup manifest](cleanup-boundary-evidence.json) binds the exact tests, source, profiles and logs. Bounded aggregate M04 review remains PASS; authorized merge/read-back and parent M03/M02 remain open.
