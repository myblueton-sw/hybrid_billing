---
type: VerificationRecord
title: HB-246 recovery register verification
description: Actual checks run for W-37 increment 1 (modules/recovery), plus review outcomes and residuals.
status: final
---

# Scope

[HB-246](https://linear.app/hybrid-billing/issue/HB-246) adds `modules/recovery`, following [plan.md](plan.md):
- **Generations and fencing.** Recovery generations change on lock, begin and abandon. Permits are writer-bound and expire. `Call` is the single choke point, refusing past-generation tokens, fenced keys and numbers at or below the high-water mark. `Done` records outcomes, with reclaiming.
- **Stop records.** `StoppedAt`, in-flight deadlines and the longest `MaxCall` are captured when writes stop.
- **Restore points.** Restore points are dual-controlled and bound to a manifest digest. Chunks are verified per domain, with no sampling.
- **Reconciliation.** Pages are verifier-only, sequential and contiguous. The final page must meet the coverage rule (`From ≤ AsOf − Skew`, `Through ≥ StoppedAt + MaxCall + Skew` and the in-flight deadlines plus `Skew`) and attest `Rotated`. Caps are applied all or nothing.
- **Re-application.** An ordered log of protective events is re-applied in order, with `Unapplicable`, `must_apply` and a `Lost` digest.
- **Unlock.** Unlock is dual-controlled and checks every condition, including kind coverage over every writer ever registered. Abandon, prune and history are bounded.

Base: main `2988ecb`. Machine: mac_mini.

# Checks run (2026-11-15, this machine)

| Check | Result |
| --- | --- |
| `gofmt -l`, `go vet ./modules/recovery/` | clean |
| `go test -race -count=10 ./modules/recovery/` | pass, including a concurrency test where Permit and Call race begin |
| `GOOS=linux GOARCH=386 go test -c -o /dev/null ./modules/recovery/` | pass |
| `go test -race -count=1 ./...` | every package passes |
| Statement coverage | 100.0% |
| Property test | 150 seeds over two installations. It models generation, state, fences, auto-fencing at begin, coverage refusals, re-application, `Unapplicable` with `Lost`, unlock and abandon. In one measured run of an earlier version of the test, 937 begins, 113 re-applications and 132 unlocks succeeded, 28 of the unlocks after a full recovery |

The manifest digest and lost digest known-answer tests were cross-checked against independent Python computations.

# Mutation testing

| Run | Mutants | Killed | Did not build | Survived |
| --- | --- | --- | --- | --- |
| First | 593 | 510 | 40 | 43 |
| Final | 611 | 561 | 41 | 9 |

Intermediate runs closed the first run's survivors with boundary and cap tests, and with one simplification.

The 9 final survivors are equivalent:
- `Round(0)` → `Round(1)`, three times;
- a `stopCall` start of 1 ns, which any `MaxCall` of at least 1 s exceeds;
- `>` → `>=` in four maximum comparisons, which give the same result on equal values;
- `<` → `<=` in a sort over unique call IDs.

# Reviews

- **Design review** (independent reviewer, read-only):
  - round 1 FAIL, H1–H5: reconciliation coverage, in-flight calls, the register's own invariants, the choke point, protective events;
  - round 2 FAIL: abandon, the pending window, `Lock` during recovery;
  - round 3 FAIL: unlock after abandon, pruning fences;
  - round 4 PASS, with text fixes.
- **Change review** (a separate independent reviewer, read-only):
  - round 1 FAIL: M1 auto-fence timing, M2 removed writers, M3 race test, M4 property model;
  - round 2 FAIL: R1 writers removed before the stop;
  - round 3 PASS.

# Residuals and exposure blockers

- Persistence that keeps the register outside restored domains, with a generation that never decreases.
- Wiring `Call`, `Done` and `Record` into issuance, delivery, webhooks, erpadapters, jobs and the evidence and auth deletion sources.
- Storage and external adapters, checksum computation, and binding the manifest digest to installer checkpoint evidence.
- Migration import (increment 2).
- A consumer acknowledgement hook that would make fences prunable.
- `Done` after a reclaim keeps the outcome `unknown`.
- HB-37-V system-level cases are NOT_RUN.
- RPO and RTO targets, key recovery, and screens.
- Owner decisions not confirmed by the owner:
  - the closed domain, writer and event kinds;
  - the bounds;
  - the accepted permanent-block risks of full caps.
