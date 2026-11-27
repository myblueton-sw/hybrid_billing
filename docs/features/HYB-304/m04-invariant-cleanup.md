---
type: ImplementationPlan
title: HYB-304 proven query invariants and remaining coverage
description: Records independent reachability review and three narrowly approved behavior-preserving simplifications without waiving package coverage.
status: stable
sources:
  - resource: ./m05-boundaries.md
  - resource: ./m04-boundaries.md
---

# Final status

Checkpoint record. The work it names as remaining or not yet built was completed later in HYB-304, which closed on 2026-11-27 ([closing audit](closing-audit.md)).

# Scope and reviewed decision

HYB-304 remains In Progress on machine mac_mini after checkpoint ed63b97a2465a7b33a34db691c4ed450aa7290a0. Root is sole writer. Separate read-only /root/hyb310_review inspected six servicequery source files and direct identity/organization/catalog/lifecycle dependencies against the frozen98.8% server profile. This is ordinary coverage/reachability review, not a restarted Graphify or per-query benchmark.

The independent design approval permits exactly three simplifications: remove the owner-list More-and-zero-scanned branch because positive bounded MemberPositions arithmetic makes it impossible; ignore the infallible json.Marshal error for the source-basis struct containing only booleans/strings/unsigned integers; remove the credential-revocation trust-hold denial branch contradicted by the explicit holdAllow policy. Strengthen actual held credential revocation verification before that last removal. Preserve result encoding, admission, query signatures and public behavior. If basis field types or hold policy change, their proofs must be re-reviewed. This is not permission to remove tenant, authorization, overflow, source-change or fallible cursor-encoding defenses.

# Remaining reviewed reachability

Separate module calls reauthorize and can legitimately fail after actor expiry/demotion; these arms remain and require actual lifecycle tests. Source growth before a retained ordering position can invalidate completion accounting. Uncollected filters legitimately exclude collected/revoked/expired credentials. Publication can observe a supported dependency change. Authenticated malformed/inconsistent signed cursors and exhausted scalar accumulators are explicit boundary fixtures, not attacker-forgeable continuations or naturally allocated populations. Constructor-bound immutable organization/catalog error arms need a separately reviewed narrowed API/refactor rather than ignored errors or fabricated map corruption.

The review covers60 uncovered weighted statements in accounts/credentials/choices/cursor/detail/history plus the audience unavailable arm. It does not satisfy or waive required package100%. No new test, mutation, runtime or removal PASS is claimed by this plan. Root will run the actual held case before/after, full server suite, scoped race/vet, inspect source diff and obtain actual-change review. UI field-error work is separate and is not evidence for server coverage.

# Actual cleanup verification

Root applied exactly the three independently approved removals. Before removal, the strengthened existing held-credential test passed: current projection reports execution ready, an actual held Revoke returns200, canonical detail reports revoked and disables another revocation. The same test passed after removal under the race detector; go vet passed. Separate actual-source re-review by /root/hyb310_review passed the narrow diff and confirmed that tenant/authentication, overflow, source-change and fallible cursor-encoding guards remain. The reviewer did not execute tests.

The full current server short suite passed in176.146 seconds at98.8% statement coverage. Its profile contains140 uncovered weighted statements,58 in servicequery files. Four impossible statement units were removed from the previous profile; the rounded percentage remains unchanged. Required package100% remains unmet without narrowing or waiver. Linux amd64 and Windows amd64 server test binaries compiled, and their ELF/PE headers were inspected; neither foreign runtime was executed. No mutation result or additional lifecycle test is claimed for invariant-only branches; strengthening an already-correct held reduction is not fabricated product fail-before evidence.

The [invariant manifest](invariant-evidence.json) binds source/dependency proofs and exact profiles/logs/binaries. Other constructor-only module error arms remain and need separately reviewed narrowing/refactoring; legitimate expiry/demotion, query-order changes, authenticated malformed cursor and counter/error propagation cases remain candidates for meaningful tests. This checkpoint is not the package gate, full M04 acceptance, PR readiness or merge. Concurrent UI validation verification is separately recorded and does not establish Go coverage.
