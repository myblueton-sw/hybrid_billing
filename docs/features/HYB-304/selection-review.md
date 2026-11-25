---
type: ReviewRecord
title: HYB-304 bounded selection and operation foundation review
description: Records independent source, contract and documentation findings for the unfinished M05 integration checkpoint.
status: stable
sources:
  - resource: ./selection-integration.md
  - resource: ./session-ui-integration.md
  - resource: ./plan.md
---

# Scope and provenance

Root is sole writer and test executor on machine mac_mini. Independent read-only reviewer `/root/hyb310_review` inspected the uncommitted selection/operation changes against foundation79856aed and the approved design. Independent read-only technical writer `/root/hyb310_copy_render` inspected the two integration documents against actual handlers and client contracts. Neither reviewer edited files, executed tests or started services. Their source verdicts do not certify runtime behavior or full M05 completion.

# Findings and correction read-back

| Finding | Correction and independent result |
| --- | --- |
| Legacy digest replay bypassed original record scope | Replay requires the canonical owner/epoch/workspace/caller binding; regression failed first, corrected source re-review PASS. |
| Expired prepared reservations could block legacy admission | Sweep expired unaccepted records before new record admission; regression failed first, re-review PASS. |
| Duplicate build continuation could return before source invalidation | Check current source/revision/expiry before replaying prior continuation; regression failed first, re-review PASS. |
| Expiry could pass during the application hook | Recheck unaccepted preparation expiry inside the session-mutex admission callback, before reservation/effects; fail-first hook regression, re-review PASS. |
| Bounded summaries traversed full retained result arrays | Maintain authoritative ended/gone/cleanup counters during construction and C3 transitions; summaries use constant work. Row-derived oracle and stored-row independence checks added; corrected source review PASS. |
| Overlapping OpenAPI oneOf alternatives | Require actual legacy response fields, close legacy result/reference properties, and distinguish result rows from reference rows. Actual HTTP payload shape tests failed first, correction source review PASS. The test matcher checks required/closed object shapes, not the complete JSON Schema specification. |
| New POST route missing explicit policy classification and removal primitive outside central helper | Preparation is explicitly a read-only no-step-up/hold-allowed snapshot; actual termination remains step-up protected. Move the unchanged removal primitive into sessionending.go, preserving the central source guard. Independent correction review PASS. |
| New UI model allegedly double-escaped Unicode controls | Reviewer rechecked raw bytes and withdrew the finding: source has single escapes and is correct. Added actual U+0000/U+0085/U+007F plus valid identifier/KO/AR cases; no implementation change. Corrected review verdict PASS. |
| Documents implied sealed-only count and unfinished operation handlers | Clarify building null vs sealed/applying/accepted immutable count; distinguish implemented server/client foundation from unfinished /me/UI. Independent TW re-review PASS. |

The reviewer separately passed typed client contract alignment: request IDs, CSRF, existing step-up transport, bounded pages/items, exact selection fingerprint and nullable outcomes match the actual handlers. Accepted operation recovery from another own session uses current canonical owner/epoch/workspace; preparation/preview/acceptance/replay remain original-caller bound. Lists explicitly count accepted records only.

# Remaining boundary

This is a foundation checkpoint. `/me` still emits full session/expanded-operation data, and rendered M05 consumers still use it. New state helpers are not yet wired to the screen. Full UI-04, date-input validation, stale responses, uncertain-result locking, browser history, final coverage, rendered EN/KO/AR/four-width checks and final increment review remain pending. No PR, merge, parent completion or owner FIX is recorded here. Native-language and full assistive-technology acceptance remain unverified.

The actual Claude CLI ui-designer invocation returned a weekly usage limit, reset November 24 at 07:00 Asia/Seoul. No same-environment review is relabelled as Claude work. Existing user merge authorization remains valid; it does not turn an unfinished increment into a completed one.
