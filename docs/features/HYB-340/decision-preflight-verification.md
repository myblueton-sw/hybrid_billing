---
type: VerificationRecord
title: Current financial decision preflight verification
description: Fresh eligibility checks without decision mutation or historical replay shortcuts.
status: draft
---

# Implemented boundary

HYB-340, machine:mac_mini, following B09 projection commit 26838c7. Root implemented Registry.CheckDecision and factored fresh decision checks into the same lock-held helper used by Decide. Independent refusal_review passed the concrete plan before code and passed the resulting source/tests without running them. No blockers remained in that scoped review.

CheckDecision validates closed input, exclusion membership, human actor, pending state, original author/requester separation, exact current approval coverage and nonempty remaining items. An already-recorded decider is refused even while a multi-approver request remains pending. Reject retains its closed reasons and absence of amount-coverage requirement. It does not mutate decision, exclusion, approver, execution or request state; the existing monotonic clock watermark can advance. This check is neither a reservation nor authenticated authorization.

Decide retains historical same-actor/same-decision replay and conflicting replay semantics. Exclusion membership remains checked before replay. Fresh decisions use the shared helper, and all existing arithmetic is unchanged, including absolute item coverage and signed subject limits. The later authenticated preparation endpoint and screen controls are not implemented by this increment.

# Executed verification

All module tests passed in 0.661s. Module race suite passed in 1.837s. Affected server financial charge, mixed projection and approval-read tests passed in 2.595s. Module go vet passed. Evidence logs are /private/tmp/hyb340-decision-preflight-test.log, /private/tmp/hyb340-decision-preflight-race.log, /private/tmp/hyb340-decision-preflight-server.log and /private/tmp/hyb340-decision-preflight-vet.log. The earlier full 81-package and 930-console verification applies to projection commit 26838c7; those suites were not rerun for this module-only change.

New tests compare public request views and internal excluded/decider/approver collections before and after successful and refused checks. They cover exact positive/negative limit boundaries, original author/requester/system refusal, invalid rejection reasons, revoked/future/expired/currency/stage authority, monotonic time after rollback, preserved historical Decide replay, pending duplicate-actor refusal, terminal states, detached exclusions, absolute item coverage, all-items exclusion, missing item/workspace and malformed workspace. Server regressions exercise existing actual-session decision integration after the shared-helper extraction.

No new UI, permission or external effect was added. No owner screen acceptance, main merge, deployment, customer data or payment occurred. HYB-340 and full W-23 remain active; the next implementation is authenticated preparation with explicit action-specific current outcomes.
