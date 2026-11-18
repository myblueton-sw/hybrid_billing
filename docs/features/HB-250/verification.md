---
type: VerificationRecord
title: HB-250 publication register verification
description: Actual checks run for W-39 increment 1 (modules/publication), plus review outcomes and residuals.
status: stable
---

# Scope

[HB-250](https://linear.app/hybrid-billing/issue/HB-250) adds `modules/publication`, following [plan.md](plan.md):
- **Staging.** Each staged dataset version is bound to an expected row count and digest. Room is reserved at `Begin` against register-wide and per-tenant limits, and a deadline-ordered expiry queue holds no rows.
- **Chunks.** Chunks are bounded and idempotent by index, and must be contiguous. A resend never duplicates or drops rows.
- **Aggregation.** Stage-1 per-chunk partials are exact and unbounded. Stage 2 merges them exactly at seal, with the result bound applied only to final totals; any `decimal.New` error is `overflow`.
- **Seal.** A k-way merge refuses duplicate keys.
- **Publication.** Publication is atomic and only moves forward, keeping a bounded number of retained versions. `Evict` and `Retire` free room on demand. Ascending version IDs are never reused.
- **Reads.** Rows and totals are read through snapshot-pinned, bytewise key-ordered cursors with bounded pages, including a typed totals cursor.

Base: main `40afc7b`. Machine: mac_mini.

# Checks run (2026-11-16, this machine)

| Check | Result |
| --- | --- |
| `gofmt -l`, `go vet ./modules/publication/` | clean |
| `go test -race -count=10 ./modules/publication/` | pass, including concurrent loads, seals, publishes and reads |
| `GOOS=linux GOARCH=386 go test -c -o /dev/null ./modules/publication/` | pass |
| `go test -race -count=1 ./...` | every package passes |
| Statement coverage | 100.0% |
| Property test | 150 seeds |

The property test:
- splits random rows into random chunks, with resends, conflicts and gaps;
- compares seal totals with an independent `math/big` rational model, including overflow decided only on the final total, with mixed signs;
- pages rows and totals with random limits, and must return exactly the sorted model;
- audits every counter.

In one measured run it reached 142 seals, 8 overflows, 1,043 gaps and 6,523 resends and conflicts.

A literal test makes partial sums cross the result bound while the final total stays inside it.

The chunk and version digest known-answer values were cross-checked with an independent Python encoder in the root session. That encoder is not in the repository, so the reviewers could not check it.

# Mutation testing

| Run | Mutants | Killed | Did not build | Survived |
| --- | --- | --- | --- | --- |
| First (before the test gaps found in review were closed) | 310 | 275 | 14 | 21 |
| Final code | 304 | 282 | 14 | 8 |

After the final run, three survivors were killed by tighter tests and checked by hand:
- the identifier bound written as a literal;
- the per-`Begin` queue bound against two compaction-threshold mutants.

The 8 remaining are equivalent:
- `value >= 0` and two `StagingTTL` forms, where zero is already mapped to the default;
- `Round(0)` → `Round(1)`;
- compacting at `>=` instead of `>`, where the bound still holds;
- the capacity hint of the compacted slice;
- `scale >= p.scale` in the rescale, a multiply by 10⁰;
- `<=` in the heap order, where equal keys are refused as adjacent duplicates either way.

# Reviews

- **Design review** (independent reviewer, read-only):
  - round 1 FAIL: H1 running-sum overflow, H2 backpressure deadlock, and M1–M6;
  - round 2 FAIL: N1 held room never freed;
  - round 3 FAIL: R1 name reuse after retire;
  - round 4 FAIL: S1 ID exhaustion;
  - round 5 PASS.
- **Change review** (a separate independent reviewer, read-only):
  - round 1 FAIL: H1 the expiry queue kept evicted data in memory, and M1 and M2 were test gaps;
  - round 2 PASS, with two test gaps closed afterwards.

  Details are in plan.md.

# Residuals and exposure blockers

- Persistence and a storage engine for the monthly volume. The 100,000-row, 1,000,000-a-day and 30,000,000-a-month P0 gates, interruption and resume, and catch-up are NOT_RUN at system level.
- Performance measurement, SLA proposals and gate records (later increments).
- Distribution as the second stage of allocation, and rollback to a retained version.
- The single mutex held for the whole seal merge. Dataset records are never freed.
- A digest supplied at `Seal` instead of `Begin`, which is an open design choice.
- Binding callers to `modules/auth` and to job checkpoints in `modules/jobs`, and wiring importers and projections.
- Owner decisions not confirmed by the owner:
  - what a source row means;
  - the inflow ceiling and peak window;
  - retention;
  - concurrent closes;
  - RPO and RTO;
  - hardware;
  - the limit defaults.
