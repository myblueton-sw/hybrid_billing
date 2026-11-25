---
type: VerificationRecord
title: Pending invitation index security and scale verification
description: Records state-index invariants, quota equivalence and actual local performance checks without claiming full API completion prematurely.
status: draft
---

# Implemented boundary and review

The registry owns a workspace-local set of state-pending invitation IDs. InviteUntil scans only those canonical records through unchanged countable checks, then inserts a new ID after successful creation. Successful acceptance, revocation and verifier-attempt locking remove the ID; failed operations, expiry and ineffective authority do not. Canonical historical records and global ID conflicts remain. There is no cached effective count, expiry pruning or public contract change. HYB-338 guard code is unchanged; this branch explicitly depends on reviewed commit 87f45f3.

Independent /root/refusal_review inspected the plan, implementation and tests and returned scoped security design/change PASS. It confirmed index mutation ordering, version-overflow retention, concurrency and historical-ID semantics. It inspected the focused log but executed no tests. Root remains the sole writer. No merge, real accounts, deployment or external effects occurred.

# Executed results

Initial existing invitation/concurrency tests passed with race detection (1.361 seconds). Focused new tests passed with race detection (2.361 seconds), /private/tmp/hyb339-index-final.log. A reproducible sequence generated 82 accepted, 118 revoked and 100 locked terminal outcomes, checking exact pending index membership and countable counts against full canonical scans after each public operation. Quota/time/authority tests cover future NotBefore consuming the existing quota, early acceptance refusal, exact expiry, clock rollback without pruning, and inactive inviter exclusion while retaining state-pending entries. Version overflow on acceptance leaves its pending record indexed. Separate workspaces retain separate quotas; a foreign revocation cannot delete another workspace's entry; 16 concurrent creates at a one-invitation limit admit exactly one and refuse 15.

The 100,000-history test uses normal public Invite/Accept/RemoveMembership/Revoke operations on one synthetic recipient, alternating accepted and revoked records. Its loop measured 1.033 seconds in this run. Every historical record remains canonical, the pending index empties, an old ID still conflicts, and the current quota still refuses an additional pending invitation. This is terminal-history performance on a fixed local workload, not 100,000 distinct-user throughput or an API SLO. The unchanged API scale test supplies that separate workload.

Go vet and diff whitespace checks passed. Full identity race passed 48.924 seconds (/private/tmp/hyb339-identity-race.log). The original unchanged full-population TestScaleFilterListsOverHTTP passed 108.744 seconds overall, including full 5,000/20,000/100,000 target phases, final reinstatement and cleanup assertions (/private/tmp/hyb339-api-scale.log). This branch contains both HYB-338 and HYB-339. HYB-338 alone still scanned terminal invitation history during fixture population; root interrupted that separately running test after the combined run passed. The interrupted standalone run is not a pass.

# Limits

Expired and ineffective state-pending records still accumulate and are scanned, deliberately preserving current time/authority semantics. This change is not a registry-wide indexing redesign, persistence migration or history-retention policy. Runtime evidence is macOS arm64, Go 1.26.4. Other operating systems and production installations are unverified. Full consumer-branch integration, fresh regressions, PR verification, authorized merge and read-back remain open.
