---
type: ImplementationPlan
title: Avoid repeated administrator scans for membership transitions
description: Preserves the final-administrator guard while removing an unnecessary full membership scan on ordinary-member transitions.
status: draft
---

# Evidence and ownership

[HYB-338](https://linear.app/myblueton/issue/HYB-338) is owned by Seung Woo Park, In Progress, Bug, machine:mac_mini. Root owns PM/planning, implementation and integration; the independent security reviewer is read-only. Branch fix/HYB-338-membership-guard starts at origin/main 0f9d9a27d6202734e79d8f86ef6e65dfdfc5a752 in /private/tmp/hyb338. HYB-225/HYB-337 worktrees and full regression processes remain unchanged. Existing owner no-restoration authorization applies to missing companion files; no missing policy is reconstructed.

A noninvasive macOS process sample of the long-running HYB-337 server test, decoded with go tool addr2line, identifies TestScaleFilterListsOverHTTP line 232 calling ReinstateMembership, transition, strands and administrators. The fixture has 100,000 memberships and reinstates ordinary suspended members between 5,000/20,000/100,000 target scenarios. strands currently counts every active administrator across the entire workspace before checking whether this target could reduce that count. This is an avoidable O(N) scan per ordinary reinstatement, O(N squared) across N targets. The sample does not prove this is the only whole-suite bottleneck or predict completion time.

# Minimal security-preserving design

Keep registry locking and the caller's fixed now unchanged. In strands, first return false if the target is not currently an active administrator. In the old implementation, the administrator count is either zero (already false) or positive; the inactive target does not decrement it, so remaining stays positive (also false). Use activeAdministrator rather than the role alone, preserving effective membership expiry and active human state.

Also move the already-existing after-remains-effective-administrator check before the count. For a target that can remove an administrator seat, retain the exact count, decrement, remaining-seat check and empty-workspace/others rule. Do not alter installation.onlySeat, acting/authentication, dry runs, version comparison/bumping, generation, user state, timing, lock scope or any public contract. Do not add cached counts or counters requiring invalidation.

Allowed edits: modules/identity/registry.go; new guard equivalence, focused transition/concurrency and scale test files; docs/features/HYB-338. Root is the sole writer. Existing test fixture sizes and assertions remain intact. No server, seed, frontend, dependency or other-ticket changes.

# Verification

Use the repository's existing standard-library model/property testing pattern, without a new package. Compare the optimized predicate against a frozen pre-change reference across finite combinations of absent/removed/suspended/active membership, member/admin role, nil/permanent/expired/exact-expiry/future expiry, human/service/missing account, active/suspended/departed account, zero/one/multiple other active administrators, and other human/service presence including foreign workspace. Pin meaningful exact-boundary and last-seat cases. This is regression equivalence, not proof that the original policy was correct.

Run existing identity tests, model tests, final-administrator and concurrent transitions with race detection, source rules and vet. Add an actual 100,000 suspended-member reinstatement run using normal public transition calls and assert versions, states and unaffected workspace isolation. Measure old and new predicate costs separately using the same bounded workload; do not turn host timing into a flaky correctness threshold. Independent security re-review must inspect equivalence, final-seat protection and actual logs. A passing helper benchmark does not certify the full API scale suite; run that existing suite on the optimized branch without reducing its population. No merge, deployment, external effects or owner acceptance is granted by this plan.
