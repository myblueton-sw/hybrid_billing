---
type: VerificationReport
title: HYB-304 malformed reservation charge preservation
description: Verifies malformed internal charge inputs cannot evict genuine retained operations or corrupt exact session-record accounting.
status: draft
sources:
  - resource: ./m04-detail-clock-contracts.md
---

# Reviewed scope and ownership

HYB-304 remains In Progress, Seung Woo Park, machine mac_mini, after pushed/read-back fd4e0f69c2e360d539eb5a23dc6cbd15df1ffd3f. Root owns focused tests/evidence only. Separate read-only /root/hyb310_review approved the malformed component-charge resource-preservation contract; design review grants no package or merge approval.

Create two genuine accepted, settled, evictable operations and nonempty recovery through authenticated producers. Capture registered identities, operation IDs and exact per-record/global units. Under the normal domain lock, pass negative one, configured budget plus one and native signed-int maximum/minimum to the existing reservation helper. Each must return false before eviction or accounting writes. Preserve every recorded object and unit, independently recomputed conservation and nonnegative accounting; after unlocking, require original known-ID recovery and a usable caller. Use the normal unlocked-session helper variant; do not claim sessionsLocked behavior without owning that mutex.

This is malformed helper-input validation and signed boundary protection, not proof that ordinary HTTP counts produce these values. It establishes neither valid-charge addition overflow nor live undercharge-panic safety. Configuration, registered records/maps and production code remain unchanged. A meaningful semantic guard mutation must demonstrate that the invalid input can corrupt the retained resource contract when protection is bypassed. No nil no-op tests, fabricated live states, coverage exclusions or waiver are planned. The preceding native99.7%/36-uncovered checkpoint did not meet required100%; bounded M04 aggregate review passed, while authorized merge/read-back and parent M03/M02 remain open.


# Actual verification and limits

The scoped native contract passed0.802s. Both negative-charge and upper-charge semantic mutations accepted a malformed input and failed the intended assertion. Each restored the exact original production SHA; the disposable private cache was removed. No product implementation changed. Independent actual-source review by /root/hyb310_review passed; that reviewer did not execute tests.

Full native server short passed198.269s at99.7%,11,779 weighted statements/35 uncovered. Matching production blocks and weights show one newly covered unit at sessionrecords.go92–94, the malformed-charge refusal. The separate focused native profile also hits that block; profiles are not combined. Eight named race groups passed3.333s. Server vet and Linux/Windows amd64 test compilation passed; foreign execution remains unverified. The previous75-package and unchanged Console/browser evidence retains its original source/run boundaries and was not rerun.

The malformed helper arguments are deliberately outside ordinary HTTP-produced charge values. Tests preserve genuine settled recovery records under the normal domain lock, then read their original operation IDs and surviving caller through HTTP. This proves rejection before destructive eviction/accounting; it does not establish all valid-charge sum overflow cases, lock-state variants or undercharge panic safety. Required100% remains unmet; authorized merge/main read-back and parent M03/M02 remain open. The [source/artifact manifest](reservation-input-evidence.json) records exact bindings. Two documentation status checks plus manual metadata/local-link review are limited checks, not general semantic certification.
