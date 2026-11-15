---
type: VerificationRecord
title: HB-254 release trace verification
description: Actual checks run for W-41 increment 1 (modules/releasetrace), plus review outcomes and residuals.
status: final
---

# Scope

[HB-254](https://linear.app/hybrid-billing/issue/HB-254) adds `modules/releasetrace`, following [plan.md](plan.md):
- **Requirements.** Namespaced, immutable requirement records with test references and categories; retirement with a newer, unretired, equally strong replacement, or without one for non-common requirements only.
- **States.** Per-build computed states (designed, implemented, failing, tested, customer proven) from results recorded by bound, unrevoked harnesses, for builds registered after the latest implementation, with self-verification refused at `Record` and on read.
- **Support paths.** Activation under dual control after the two latest consecutive matched reconciliation periods (latest within 3 months), passing tagged evidence for every category and tested requirements, without harness-binder or reconciler self-verification.
- **Release gate.** Every problem reported and sorted, register-wide common and blocking checks, `path_evidence` on every gate, and templates shown as verified only for problem-free active paths.

Base: main `6de9b94`. Machine: mac_mini.

# Checks run (2026-11-16, this machine)

| Check | Result |
| --- | --- |
| `gofmt -l`, `go vet ./modules/releasetrace/` | clean |
| `go test -race -count=10 ./modules/releasetrace/` | pass, including concurrent records, gates and proposals |
| `GOOS=linux GOARCH=386 go test -c -o /dev/null ./modules/releasetrace/` | pass |
| `go test -race ./...` | every package passes |
| Statement coverage | 100.0% |
| Property tests | 60 seeds of random results, builds, implementations and revocations against an independent state model; 200 seeds of random registers and releases against an independent gate-problem model |

# Mutation testing

| Run | Mutants | Did not build | Survived |
| --- | --- | --- | --- |
| First | 499 | 27 | 36 |
| After review fixes and literal tests | 502 | 27 | 12 |

Literal-value tests (exact bounds, period parsing, implementation history, reconciliation ordering, problem ordering, category evidence, single-period mismatch, read-side self-verification) killed the first run's non-equivalent survivors. One remaining survivor (a length check that is always true) was removed by simplifying the code.

The remaining 11 are equivalent:
- the length prefix buffer `[8]byte` → `[9]byte` and two `boolText` changes, which only alter the internal proposal identity consistently;
- `Round(0)` → `Round(1)`;
- the year digit limit 4 → 5 on a 4-byte slice;
- two `return 0, false` values that callers ignore;
- three sequence or period ties that cannot occur (`seq` is unique, periods are unique per path);
- a problem-subject tie that cannot occur, because problems are unique.

# Reviews

- **Design review** (independent reviewer, read-only): round 1 FAIL, round 2 FAIL, round 3 PASS with lows folded in.
- **Change review** (a separate independent reviewer, read-only): round 1 FAIL (M1 template shown after evidence lapsed, M2 untested read-side self-verification, M3 narrow property test, lows L1–L5); round 2 PASS with lows folded in.

Details are in plan.md.

# Residuals and exposure blockers

- Running tests and reconciliations, production classification of proof environments, and wiring `modules/perfgate` and `modules/simulation` evidence.
- Commit ancestry between builds and implementations is a trust assumption.
- Persistence, consumer wiring and screens are not built here.
- AUD-DB-01 and the HB-41-V system-level cases are NOT_RUN.
- Owner decisions not confirmed by the owner: the category list, the 3-month staleness rule, the two-period activation rule and the first release's supported combinations.
