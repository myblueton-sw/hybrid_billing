---
type: ImplementationPlan
title: HYB-304 detail read authority at clock boundaries
description: Sweeps instance-owned wall-clock observation boundaries in two quiesced detail handlers while verifying fail-closed authority and unchanged service data.
status: stable
sources:
  - resource: ./m04-lookup-parser-contracts.md
---

# Final status

Checkpoint record. The work it names as remaining or not yet built was completed later in HYB-304, which closed on 2026-11-27 ([closing audit](closing-audit.md)).

# Reviewed scope and ownership

HYB-304 remains In Progress, Seung Woo Park, machine mac_mini, after pushed/read-back 9e44b127707e5e9311f56500554686175f3fc062. Root owns focused tests and evidence only. Separate read-only /root/hyb310_review approved the bounded design and its corrected native-process ownership model. No production change, registry-map rewrite, runtime stack inspection or new dependency interface is planned.

For each actual account-detail and credential-detail handler, create real service/declaration/credential data and authenticate normally. Complete healthy controls and full Close before arming that fixture's instance-owned, mutex-protected Config.Clock. Config.Elapsed uses an independent monotonic origin. Unrelated fixture callbacks cannot access this clock; native nonparallel execution is sufficient because no process-global input is replaced. Use a fresh fixture for every cutoff; an adopted clock offset is never rolled back.

A healthy quiesced control supplies a finite set of observation boundaries. At a selected observation, the wall value permanently advances past the genuine caller's absolute expiry while remaining below the configured wall-lead tolerance. Counts select input cases and are not query benchmarks or evidence of a guessed internal failure stage. Capture whether a jump occurred immediately when the handler returns, before observer work consumes clock observations. Unobserved jumps must preserve healthy200 with the actual row. Observed jumps must produce an appropriate401 Unauthenticated,403 PermissionDenied or503 DependencyUnavailable, one JSON failure and no projected metadata.

This oracle is specific to these two direct handlers. serviceDetailRead.write reauthorizes and calls serviceBasis, whose identity ServiceQueryBasis performs a final timed administrator check. Remaining basis construction and writeJSON make no later configured-clock observation. Recheck that premise if those functions change; do not generalize it to arbitrary handlers. These are quiesced handler-component inputs, not live-server authentication, operational scheduling or live-history acceptance.

Compare actual service, declaration and credential state plus pickup identity/deadline and server revision before/after. Establish a fresh supported observer at the stable retained clock value without resetting it. Authentication cleanup and offers rejected by the already-closed history are separate allowed effects; changing time-derived open/expiry annotations is not a stored-data mutation. Derive exercised code arms from actual coverage, not cutoff numbers. Required100% remains unmet at native99.7%/40 uncovered; bounded M04 aggregate review passed, while authorized merge/read-back and parent M03/M02 remain open.

# Observed scoped verification

The initial two sweeps passed in7.257 seconds. After tightening mutex protection and capturing jump state immediately after handler return, the final two groups passed in6.848 seconds. Nine new and related authority/eligibility groups passed under the race detector in17.037 seconds. Server-package vet and Linux/Windows amd64 test-binary compilation passed; no foreign execution is claimed. Independent actual-source review passed the native fixture ownership, copied state, observer separation and oracle; the reviewer ran no tests or mutations.

Two semantic mutations were assertion-killed. Removing the complete final publication fence allowed projected data after an observed expiry; appending another JSON object after a dependency refusal failed strict response decoding. This demonstrates the combined publication fence and single-response contract, not separate necessity of every redundant check. Original production bytes were restored to exact recorded SHAs, and the private cache was removed before final checks. All107 production hashes remain unchanged. Clock observation counts remain test input selectors and are not recorded as per-query metrics.

# Native coverage attribution

The full server short suite passed in197.790 seconds with99.7% native statement coverage:11,779 weighted statements and36 uncovered. Required100% remains unmet. A separate instrumented execution of only the two sweeps passed in7.127 seconds. Its native counters and the full profile have exactly the same production blocks/weights as the preceding9e44b12 server profile. Both newly covered blocks are also hit by the focused sweeps: the account handler's service-admission failure after actual ServiceUser lookup and the credential handler's ServiceUser failure after actual Credential lookup. They total four weighted statements. This is observed native coverage; no cutoff is asserted to identify a named internal stage and no profile union is used.

The preceding75-package all-short checkpoint remains separately attributed; only the server suite, focused race/vet and portable compilation were rerun for this test-only increment. Frontend/browser evidence is unchanged. Two documentation status checks and root metadata/link checks do not establish general semantic OKF certification. [The detail-clock manifest](detail-clock-evidence.json) binds actual source, logs, profiles and restoration evidence. Bounded aggregate M04 review remains PASS; required100%, authorized merge/read-back and parent M03/M02 remain open.
