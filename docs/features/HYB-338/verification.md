---
type: VerificationRecord
title: Membership guard equivalence and scale verification
description: Records actual security equivalence, public transition and scale results for the early-return optimization.
status: draft
---

# Implemented scope

The strands guard now returns before counting workspace administrators when the target is not currently an active administrator or the proposed membership preserves effective administrator status. The existing count/decrement/final-seat/others body remains unchanged. No cached count, caller, lock, authentication, dry-run, generation, version, installation onlySeat or public contract changed. Original policy semantics are preserved; their correctness is not independently proved by equivalence.

Root is the sole writer in /private/tmp/hyb338 on fix/HYB-338-membership-guard. Independent /root/refusal_review inspected the plan, both callers, onlySeat, implementation and tests and returned scoped design and change PASS. It ran no tests. Existing HYB-225/HYB-337 full runs were later interrupted with the identified old-path bottleneck recorded as INTERRUPTED, not PASS. Their source was not modified while running; they require integration and fresh full regression.

# Executed verification

- Enumerated predicate equivalence with race detection: PASS 1.528 seconds. The frozen reference is the pre-change implementation. 172,494 enumerated combinations include 316 refusal outcomes and cover target/account/membership/expiry/other-administrator/service-user states. Separate temporal cases preserve existing start/expiry-boundary behavior. Exhaustive means this explicit finite matrix, not every possible registry state. Log: /private/tmp/hyb338-equivalence.log.
- Full identity module race suite: PASS 54.453 seconds, including existing model, final-administrator, lifecycle, concurrency, source and scale tests plus new dry-run/version/generation and concurrent final-seat removal checks. Log: /private/tmp/hyb338-identity-race.log.
- Focused public scale measurement: PASS 1.976 seconds overall; 100,000 ordinary suspended memberships reinstated through normal public calls in 404.085 ms in this run. Fixture population uses private registry helpers, as in existing scale tests. Every target is checked for state/version, the generation increases exactly once per transition, and the other workspace stays unchanged. Log: /private/tmp/hyb338-public-scale.log.
- Additional last-administrator refusal no-write check: PASS 1.281 seconds, confirming demotion/suspension/removal refusals preserve membership and generation. Log: /private/tmp/hyb338-last-admin-write.log.
- Isolated guard benchmark: PASS, identical 100/10,000/100,000-member fixtures, 10 iterations repeated three times per original/optimized path. At 100,000 members the original path measured 6.69–14.69 ms per call and optimized path 258–317 ns per call, with zero allocations. This low-iteration exploratory measurement is neither an SLO nor end-to-end API throughput; concurrent work and the host affect timings. Log: /private/tmp/hyb338-guard-bench.log.
- go vet ./modules/identity and git diff --check: PASS.

The original, unchanged TestScaleFilterListsOverHTTP is running with race detection and its full 100,000-person population. /private/tmp/hyb338-api-scale.log is not a passing result until the process completes. Full consumer branch tests, remote PR, authorized merge and read-back remain outstanding. No fixture population was reduced and no existing assertion was skipped to speed verification.

# Environment and limits

Execution used macOS arm64 and Go 1.26.4. No Windows/Linux execution, real account data, deployment or owner approval is claimed. The optimization targets the sampled reinstatement scan; other expensive fixture or service paths may remain. The exact runtime sample and standalone guard benchmark do not establish that all pre-launch scale requirements are met.
