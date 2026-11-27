---
type: ValidationRecord
title: HYB-304 service query lifetime and boundary checks
description: Records real expiry and membership transitions, public-module source interleavings and authenticated query-boundary robustness without changing production behavior.
status: stable
sources:
  - resource: ./m04-invariant-cleanup.md
---

# Final status

Checkpoint record. The work it names as remaining or not yet built was completed later in HYB-304, which closed on 2026-11-27 ([closing audit](closing-audit.md)).

# Scope and fixture design

HYB-304 remains In Progress, machine mac_mini, following pushed/read-back checkpoint5cbf06d313420d18af381eeca3fbd61c20ce1cfd. Root owns new server tests and evidence; independent read-only /root/hyb310_review reviews their actual contracts. Current production Go and Console source are unchanged. The latest prior server profile is98.8% with140 uncovered weighted statements,58 in servicequery. Required100% is not waived or narrowed.

The prior independent reachability review established legitimate expiry/demotion and source-change boundaries, plus explicit signed-cursor/scalar/encoder boundaries. The new design review passed with corrections: the service-declaration seam cannot be described as occurring before an already completed admission-count lookup; advancing the world clock from inside Config.Clock deadlocks its mutex. The mid-page fixture instead wraps only the server observation-clock function after admission and advances before invoking its original function. Identity retains the original clock and sees the same advanced world time on its next authorization call. No private actor, owner, registry or catalog map is corrupted.

# Verification scope

- Actual HTTP requests advance fixture time at existing named module seams, returning false rather than fabricating a dependency error. Expired module access returns a failure with no rows/count/admission/query metadata, and the same expired session is rejected by a subsequent request. The known-service publication case expects401, rather than misclassifying it as an admission-count error.
- An admitted credential component walk uses PageSize1 and two actual issued credentials. First projection advances time; the next credential-page call rejects the actual expired actor. The accumulated row is discarded, matching-count progress proves a projection occurred, and failure mapping emits one JSON object without page metadata. This is an explicitly staged component test, not a full HTTP timing claim.
- Real emitted continuations retain a partially scanned last service under the valid100-unit scan minimum. After continuation admission, a public identity-module insertion sorts before its position. Updated total and remaining ordered entries disagree, so both account and credential walkers refuse rather than publish partial/exact data. Another public module mutation after projection is refused at publication while the caller remains authorized. These are controlled module interleavings, not evidence that HTTP command writes bypass the server domain lock.
- A real seeded administrator is demoted through the public membership API between admission and module reading; the read and subsequent access refuse without page data, while a peer administrator retains access. A workspace promotion alone was an invalid initial fixture because it did not supply the separate installation authorization required by the server.
- Actual collection, revocation and expiry are excluded by uncollected selection even with state=all. Excluded credential records remain independently readable; a still-active uncollected credential remains selected.
- Authenticated copies of real producer cursors exercise exhausted matching/completion/aggregate counters and malformed numeric history positions. Tests deliberately use the server signer; these are robustness fixtures, not ordinary emitted states or attacker-forgeable continuations. They allocate only small valid populations, never impossible uint64-sized registries.
- Directly setting the private scalar deliveryVersion to its maximum is an explicit exhaustion fixture. Credential list/detail reads refuse without wrapping or repairing the counter, while account metadata remains readable. This is not a naturally reached lifecycle or public-command transition. Installer setting rows are independently mutable copies with functional minimum/default setters.
- Publisher component inputs with unrepresentable timestamp fields refuse encoder failure without returning rows or a continuation. These are explicit local encoding-boundary inputs; no real year10000 HTTP execution is claimed.

Initial test attempts used invalid scan budgets2/4, below the public100 minimum, and incorrectly treated workspace promotion as installation authorization. Corrected fixtures use the declared minimum and existing authorized administrators. Those failures are setup errors, not product regressions or fabricated fail-before evidence. Checks for already-correct production paths are expected to pass unchanged. Actual final source review, suite, mutation, coverage and portability observations will be recorded separately; no result is implied by this description. No Graphify or per-query benchmark was restarted.

# Actual results and independent review

All fourteen new top-level groups passed against unchanged production. The final restored full server short suite passed in184.683 seconds at99.1% statement coverage. Its11,805 weighted statements include101 uncovered units,19 in servicequery. The prior frozen profile had140 uncovered units,58 in servicequery. Required package100% remains unmet; no exclusion, narrower denominator or waiver was introduced. The earlier186.459-second full run passed before the final two assertion corrections and is preserved as a distinct snapshot rather than substituted for the final run.

The independent source reviewer found two masking weaknesses: accepting either403 or503 after completed demotion, and checking collection exclusion only after that credential had also expired. Root changed the first to exact403 permission_denied and added an immediate pre-expiry collection selection plus canonical active/collected observation. The corrected targeted run passed; the actual-source re-review passed all five files. The reviewer executed no tests. Final targeted race covering all fourteen groups passed in8.232 seconds, and go vet exited zero. Linux amd64 and Windows amd64 test binaries compiled; their ELF/PE formats were inspected. Neither foreign runtime was executed.

Four semantic mutants independently disabled publication source-basis comparison, matching-counter overflow refusal, collected-credential exclusion and canonical history-position comparison. Each failed its intended new test assertion, rather than compilation. Production bytes and SHA256 were restored after every mutant, and the batch's disposable private Go build cache was removed. The final suite, race, vet and cross-compiles ran after restoration. Already-correct production tests passing before mutation are not product fail-before regressions.

The [query lifecycle manifest](query-lifecycle-evidence.json) binds fifteen source files and twenty-nine temporary artifacts, including initial fixture errors, corrected results, semantic failures, final profiles and binaries. Empty vet/compiler logs rely on observed exit-zero tool results, not their empty contents. Console source, its1,016-test checkpoint and prior rendered matrices are unchanged and were not rerun for this Go-test-only increment. Full M04 parity, remaining package coverage and final integration reviews precede readying draft PR251 and its already authorized merge/read-back. Parent M03/M02 remain afterward; no feature completion, owner acceptance, deployment, Claude visual leadership, native-language or assistive-technology acceptance is claimed.
