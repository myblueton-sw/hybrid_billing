---
type: VerificationRecord
title: HB-251 scale gate register verification
description: Actual checks run for W-39 increment 2 (modules/perfgate), plus review outcomes and residuals.
status: final
---

# Scope

[HB-251](https://linear.app/hybrid-billing/issue/HB-251) adds `modules/perfgate`, following [plan.md](plan.md):
- **Profiles.** The five P0 gates (`e2e`, `daily_inflow`, `monthly_close`, `interrupt_resume`, `catch_up`) are dual-controlled profiles with independent axes, P0 floors, and a fixed dataset and expected-result digest. Profiles can be retired under dual control.
- **Hardware and budgets.** Hardware records are immutable. Budgets have required metrics and a per-profile rate floor. They stay proposed until an approval with a distribution digest, by a third person.
- **Runs.** Runs are recorded by bound harnesses, with self-verification refused at `Record` and on read. Verdicts are computed on read against the budget that is active now, including `predates_budget` by a register-wide event sequence and an overflow-safe per-day rate check.
- **Readiness.** Gate status and readiness are per build. `not_run`, `provisional`, `predates_budget` and `unbudgeted` are never counted as passing.

Base: main `9a04497`. Machine: mac_mini.

# Checks run (2026-11-16, this machine)

| Check | Result |
| --- | --- |
| `gofmt -l`, `go vet ./modules/perfgate/` | clean |
| `go test -race -count=10 ./modules/perfgate/` | pass, including concurrent records, proposals and readiness reads |
| `GOOS=linux GOARCH=386 go test -c -o /dev/null ./modules/perfgate/` | pass |
| `go test -race -count=1 ./...` | every package passes |
| Statement coverage | 100.0% |
| Property test | 150 seeds against an independent model, every verdict reached |

The property test:
- covers all five gates and a `math/big` rate model;
- perturbs data, output, rows, duration, metrics, hardware and interruption;
- includes a second harness whose people overlap an approval;
- covers budget approvals, replacement, retirement and revocation;
- checks every run's verdict, every gate status and readiness for two builds after every step;
- ends each seed with clean runs that must make readiness true.

# Mutation testing

| Run | Mutants | Killed | Did not build | Survived |
| --- | --- | --- | --- | --- |
| First (before the change review fixes) | 419 | 366 | 39 | 14 |
| Final code | 426 | 382 | 39 | 5 |

The 5 remaining are equivalent:
- the encoder length field of 9 bytes, which only changes the internal proposal identity consistently;
- `Round(0)` → `Round(1)`;
- `<` → `<=` and `>` → `>=` against the approval event, because a run and an approval never share an event number;
- `<` → `<=` in the profile sort, because IDs are unique.

# Reviews

- **Design review** (independent reviewer, read-only):
  - round 1 FAIL: H1 harness-attested correctness, H2 no self-verification separation, and M1–M3;
  - round 2 FAIL: N1 per-day rate;
  - round 3 PASS, with lows folded in.
- **Change review** (a separate independent reviewer, read-only):
  - round 1 FAIL: H1 int64 overflow in the rate check that could give a false pass, M1 the total cap, M2–M4 test gaps;
  - round 2 PASS.

  Details are in plan.md.

# Residuals and exposure blockers

- Running the measurements: the harness, generation at scale and the P0 gates themselves. These stay NOT_RUN at system level, and nothing here has passed a real gate.
- p50 and warm or cold conditions as separate fields.
- Per-gate minimums for the other axes, `catch_up` lag rules, and fault injection.
- Retry counts are shown, not limited.
- Whether retiring a profile should free its runs.
- Persistence and screens.
- Owner decisions not confirmed by the owner:
  - every SLA number;
  - what a source row means;
  - the inflow ceiling and peak window;
  - RPO and RTO;
  - hardware;
  - the closed metric set.
