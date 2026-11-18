---
type: Verification
title: HYB-292 verification
description: Checks, mutation results, browser regressions, reviews and the planning cross-check for the eight-hour command journal sizing.
status: final
tags: [HYB-292, HB-257, HB-258, HB-30, HB-39, verification]
sources:
  - resource: plan.md
---

# HYB-292 verification (root, 2026-11-18)

Base main d102fb4. Go 1.25.7, darwin/arm64. Every result below was run by root in /private/tmp/hb292 unless stated.

## Change

- `apps/api/server/server.go` `CommandLimits`: `Retention` 2 h → 8 h, `Refill` 250 ms → 1 s. Every other limit unchanged.
- `server_test.go` `limitsOK`: `UserJournal` and `UserOperations` ≥ `Retention/Refill + Burst` (28,860 ≤ 30,000), `Retention ≥ 8 h`, `Refill ≤ 1 s`, plus the existing nesting and `Retention > Replay`.
- `literals_test.go` pins the new values.
- New `session8h_test.go` `TestEightHourSession`.
- Documents: HB-257 residual, HB-258 plan amendment and the HB-30 and HB-39 rows in HB-258 verification.

## Checks

| Check | Result |
| --- | --- |
| `gofmt -l` | clean |
| `go vet ./apps/api/...` | ok |
| `go test ./...` | all packages ok |
| `go test -count=1 ./apps/api/server ./modules/commands` after review fixes | ok (100.4 s, 1.9 s) |
| `-race` on `TestEightHourSession`, `TestRateAndAudit`, `TestLiterals`, `TestCapRefuses` | ok |
| Coverage `apps/api/server` | 100.0% |
| `TestEightHourSession` runtime | 0.8–1.4 s for about 29,400 in-process calls, not skipped under `-short` |

## Mutation (root, rsync copy, full `apps/api/server` package per mutant)

| Mutant | Result |
| --- | --- |
| `Retention` 7 h | killed |
| `Retention` 9 h | killed |
| `Refill` 250 ms | killed |
| `Refill` 2 s | killed |
| `Burst` 1300 | killed |
| `UserJournal` 28,800 | killed |
| `UserOperations` 28,800 | killed |

The `limitsOK` direction fix (`Refill <= 1 s`) and the upper-edge expired-key check were added after this run; both only add assertions.

## Browser regressions with the 1 s refill

Production console build and an hb-api built from this tree; data directories created with mode 0700 and deleted after each group. Runners are outside the repository (session scratchpad).

| Runner | Result |
| --- | --- |
| M01 `qa264` | 51 passed, 0 failed |
| P01 `qa283full` | 0 failures of 46 |
| P04 `qa289full` | R 559/0, A 28/0, B 20/0, C 46/0, D 11/0 |

No run produced a `rate_limited` answer. Runners for M03, HYB-293, P02, P03 and P05 were not re-run for this ticket.

## Reviews

- Design review round 1: FAIL (2 high, 3 medium, 2 low). Folded in: see plan.md.
- Design review round 2: FAIL (1 high: step-up every 15 minutes; 1 low). Folded in.
- Change review: PASS. M1 `Refill >= 1 s` pointed the wrong way: fixed to `<= 1 s`. L1 plan item 2 now lists the floor terms. L2 HB-258 plan line 82 gets a pointer to the amendment. L4 this file. L5 the HB-30 and HB-39 rows say what is closed and what stays deferred. Suggested upper-edge expired-key probe added (`k-1`, 8 h less one second old). L3 (`status` values) left as repository practice.

## Planning cross-check

| Source | Requirement | Status |
| --- | --- | --- |
| HB-257 plan line 109 | journal sized for 8 h at 1/s, at least 28,800 records, retention keeps the newest, test to the bound, idempotency store and rate limits sized and tested the same way | Met: 8 h retention, 30,000 shares, 1/s refill; test drives 28,800 + 600 mutations, checks replay, expiry, swept tombstone and the exact retained range |
| HB-257 residual line 213 | gap recorded | Closed, line updated |
| HB-258 plan line 91 guarantee | runaway loop throttled, never locked out | Kept (`limitsOK`) |
| v1.16 HB-30 | administrator status and history | Command journal part closed; durable storage still deferred |
| v1.16 HB-39 | large-scale processing and performance budget | Command-limit sizing part closed; the rest deferred |

## Residuals

- Sustained rate per user is 1 command per second after a 60-call burst.
- `Users` (20) count spans 8 hours.
- Reused keys answer `idempotency_key_expired` from 1 h to 8 h.
- Values are code defaults, not installation configuration.
- The journal is in memory and resets with the process.
