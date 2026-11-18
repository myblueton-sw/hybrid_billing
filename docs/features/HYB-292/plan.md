---
type: Plan
title: HYB-292 audit journal sized for an eight-hour review session
description: Raises command journal and idempotency retention from two to eight hours and the sustained rate limit refill from 250 ms to 1 s, so a reviewer at one command per second fits for eight hours and a runaway loop is still throttled rather than locked out, with a test that drives one user through 28,800 mutations and on past the retention.
status: final
tags: [HYB-292, HB-257, HB-258, HB-177, HB-30, HB-39, commands, audit]
sources:
  - resource: ../HB-257/plan.md
  - resource: ../HB-177/plan.md
  - resource: ../HB-258/plan.md
  - resource: ../HB-258/verification.md
---

# HYB-292 audit journal sized for an eight-hour review session

## Requirement

HB-257 plan line 109: the in-memory command audit journal, the idempotency store (`Operations`) and the rate limits (`Burst`, `Refill`) are sized for an eight-hour review session at 1 command per second (at least 28,800 records, retention keeping the newest). A test drives the journal to that bound and checks that commands still run. HB-257 residual line 213 and HB-258 verification.md lines 85-86 record the gap against requirements HB-30 (administrator status and history) and HB-39 (large-scale processing and performance budget) of the local planning input `docs/planning/v1.16/requirements.md` (untracked in the authorized workspace): `Retention` is 2 hours (`apps/api/server/server.go:132`) and the test drives 5,400 commands (`server_test.go:237-242`, `TestRateAndAudit`).

HB-258 plan line 91 adds a reviewed guarantee (R2-L5, R2-L7, line 247): the user share exceeds the sustained rate limit over the retention, so a runaway client loop is throttled by the rate limit and never locked out by the cap. `limitsOK` (`server_test.go:109-112`) enforces it.

## Current values (`CommandLimits`)

| Limit | Value | Covers 8 h at 1/s? |
| --- | --- | --- |
| `Retention` (journal and operation tombstones) | 2 h | No: records older than 2 h are swept |
| `Replay` (idempotent replay window) | 1 h | Not a retention limit; unchanged |
| `UserJournal`, `UserOperations` | 30,000 | Yes: 28,800 + `Burst` 60 = 28,860 |
| `WorkspaceJournal`, `WorkspaceOperations` | 600,000 (20 users × 30,000) | Yes |
| `Journal`, `Operations` | 1,200,000 (2 workspaces) | Yes |
| `Burst`, `Refill` | 60, 250 ms (4/s sustained) | Yes for the reviewer; but at 4/s over 8 h a loop would need a 115,260 share |

`Retention` is short. Raising it alone would break the HB-258 guarantee, because 4/s over 8 h exceeds the 30,000 share.

## Change

Base: main d102fb4. Owner and only writer: root. Allowed files: `apps/api/server/server.go`, `server_test.go`, `literals_test.go`, a new `apps/api/server/session8h_test.go`, this folder, `docs/features/HB-257/plan.md` (residual line 213), `docs/features/HB-258/verification.md` (rows HB-30 and HB-39) and `docs/features/HB-258/plan.md` (limits table, as a dated amendment).

1. `Retention` becomes 8 h and `Refill` becomes 1 s. `Burst` 60, `Replay` 1 h and every share keep their values. 1 s is HB-177's own provisional default (HB-177 plan line 68) and exactly the requirement's one command per second.
2. The HB-258 guarantee is kept and generalised: `limitsOK` requires `UserJournal >= Retention/Refill + Burst` and `UserOperations >= Retention/Refill + Burst` (28,800 + 60 = 28,860 <= 30,000), plus the existing nesting and `Retention > Replay`, and the requirement floor `Retention >= 8 h` and `Refill <= 1 s` (28,800 at one per second fits). A sustained loop is answered 429 with `Retry-After`, never locked out by the cap, and the newest records are always kept.
3. Memory: every cap is unchanged, so the worst-case bound is unchanged. Typical memory for the same activity grows up to 4x because records and operations are held 4x longer. No new estimate is claimed.
4. New test `TestEightHourSession` (own file, about 29,400 calls; it runs in the normal suite and under `-race`, not skipped under `-short`; runtime is recorded in verification):
   - the test uses `probeWorld` and `probeCall` (`stepup_test.go:58-72`) and reauthenticates with `f.reauth` (204) at least every 14 minutes of test clock, because every mutation needs a password confirmation within 15 minutes (`stepup.go:17`, `:85`) and a mutation does not renew it. Step-up refusals are decided before `commands.Execute`, take no token and write no record, and reauthentication uses the separate sign-in limiter, so this does not disturb the bound. A real reviewer making changes reauthenticates every 15 minutes too; that is existing behaviour;
   - one signed-in user sends 28,800 mutations through the `probe` command at one per second on the test clock, each with a unique idempotency key and request ID and the current expected revision; every call answers 200;
   - `Journal` then holds all 28,800 test requests in order; this is read before any further call, because the next reservation at T0+8 h sweeps the first record;
   - check calls use unique request IDs and are counted in the boundary assertions (each takes a token and writes a record); a replay resends that key's original body including its original expected revision;
   - a key from within the last hour replays (`replay` true, state unchanged); a key 2 h old and the second key (about 8 h less one second old, the upper edge) answer 409 `idempotency_key_expired`;
   - 600 more mutations at one per second answer 200; the oldest retained test record is exactly the one sent 8 h before the last sweep (boundary `!now.Before(kept+Retention)`), nothing newer is missing and the last record is the last call; the oldest key's tombstone is gone, so a reuse is treated as new;
   - `Unrecorded` and `Throttled` for the workspace are empty.
5. `TestRateAndAudit` keeps its burst check (429 with `Retry-After` "1"). Its 5,400-call loop at one per second stays valid at the 1 s refill.
6. `literals_test.go` pins `Retention` and `Refill`.
7. Documents: HB-257 residual line 213 and HB-258 verification rows HB-30 and HB-39 say what HYB-292 closes and what stays deferred; HB-258 plan gets a dated amendment for the limits table (retention 8 h, refill 1 s, guarantee unchanged).

## Residual effects (provisional owner defaults)

- A single user sustains at most 1 command per second after the 60-call burst; faster clients get 429 with `Retry-After`. The console maps 429 to its `limited` state (`apps/console/src/service/api.ts:30`, `admin.ts:453`). The live browser QA runners are outside the repository, in the session scratchpad (`qa264.mjs`, `qa298.mjs`, `qa293full.mjs`, `qa283full.mjs` to `qa289full.mjs`, `qa315body`); they are re-run against this change and the results recorded in verification.
- Users stay counted in `Users` (20) while they hold records, now for 8 h, so a workspace with more than 20 distinct recording users within 8 h refuses the newest users' recorded calls.
- A reused idempotency key now answers `idempotency_key_expired` from 1 h to 8 h instead of 1 h to 2 h.
- Values remain code defaults in `CommandLimits`, not installation configuration.

## Out of scope

Durable journal storage, configuration flags for command limits, changes to the limiter algorithm.

## Tests and verification

`gofmt`, `go vet`, `go test ./...`, `-race` on `apps/api/server` and `modules/commands`, coverage unchanged, mutation of the changed lines, browser regression runners for rate-limit effects, independent change review.

## Review rounds

- Design review round 1: FAIL (H1 HB-258 guarantee withdrawn, H2 idempotency store not driven, M1 weak journal assertions, M2 residual text, M3 cross-check and documents, L1 test name, L2 runtime). All folded in above: refill 1 s with the generalised guarantee (H1), mutations with unique keys and replay/expiry/tombstone checks (H2), exact journal contents and boundary (M1), residuals rewritten (M2), HB-30/HB-39 and HB-258 documents (M3), `TestRateAndAudit` (L1), runtime recorded (L2).
- Design review round 2: FAIL (N1 high: mutations need a password confirmation every 15 minutes, so the test must reauthenticate; N2 low: ordering, replay body and counting of check calls; low: planning-input link and runner location). Folded in above. Round 1 findings all resolved. Root approves the plan with these folds; the change review verifies them.
