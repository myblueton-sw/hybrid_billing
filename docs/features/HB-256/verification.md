---
type: VerificationRecord
title: HB-256 retention register verification
description: Actual checks run for W-43 increment 1 (modules/retention), plus review outcomes and residuals.
status: stable
---

# Scope

[HB-256](https://linear.app/hybrid-billing/issue/HB-256) adds `modules/retention`, following [plan.md](plan.md):
- **Artifacts.** Registered with a class, tenants, an immutable anchor and stores. Re-registration of a tombstoned ID is refused.
- **References.** `requires` blocks deletion and `mentions` never does, so deleting an Assistant conversation never deletes the evidence that mentions it.
- **Policies.** Class revisions under dual control. A shortening carries a module-set impact, and the approver must bind an impact read within the last hour.
- **Holds.** Placed by one human and never refused, with per-artifact outcomes `not_preserved`, `outcome_pending` and `partially_deleted`. Released by two humans other than the placer.
- **Deletion runs.**
  - Approval re-checks every artifact and blocks the rest at once.
  - Claims are ordered and leased with fencing tokens; a reclaim after lease expiry re-checks nothing.
  - Retries re-check and clear failed results.
  - Per-store reports carry terminal deletions and a damage flag.
  - A stop is available to one human.
  - Backup expiry is recorded per store.
  - Run states, outcome counts and per-store counts are reported.
- **Tombstones.** A monotonic sequence with a watermark and `Restored`.

Base: main `f0c7f78`. Machine: mac_mini.

# Checks run (2026-11-16, this machine)

| Check | Result |
| --- | --- |
| `gofmt -l`, `go vet ./modules/retention/` | clean |
| `go test -race -count=10 ./modules/retention/` | pass (229 s, including the 10,000,000-entry bound tests and concurrent registration, holds, claims and reports) |
| `GOOS=linux GOARCH=386 go test -c -o /dev/null ./modules/retention/` | pass |
| `go test -race ./...` | every package passes |
| Statement coverage | 100.0% |
| Property test | 120 seeds of random holds, references, clock moves, plans, claims and per-store reports against an independent model of access, outcomes and tombstones |

The impact digest known-answer value for the IDs `b`, `c` was cross-checked in this session with an independent Python encoder (8-byte big-endian length prefixes, SHA-256). It matches: `1b3536bb5505545b611a9077654d308b0aac52db128afb031b5646c47b308076`.

# Mutation testing

| Run | Mutants | Survived |
| --- | --- | --- |
| Final code, first pass | 554 | 31 |
| The 31 survivors against the added literal tests | 31 | 4 |

The added tests kill the other 27 survivors. They cover:
- duplicate list entries;
- the 24-hour day at the due boundary;
- pending-list removal and the applied-proposal window;
- equal-total revisions that are not shortenings;
- deleted artifacts excluded from impact and preview, and owned artifacts excluded from preview;
- the exact one-hour impact window;
- generated run IDs past 9;
- page limits of 1 and 1,000;
- the owner-chunk finish condition;
- `Proceed` on a reported artifact;
- the store form of backup expiry;
- all eight outcome keys.

The remaining 4 are equivalent:
- the length-prefix buffer `[8]byte` → `[9]byte`;
- `Round(0)` → `Round(1)`;
- a tie in the tenant insertion, which cannot happen because duplicates return earlier;
- `after >= len` → `after > len` in tombstone paging, where both give an empty page.

# Reviews

- **Design review** (independent reviewer, DBA focus, read-only):
  - round 1 FAIL (H1–H2, M1–M8);
  - round 2 FAIL (N1–N3);
  - round 3 PASS, with lows folded in.
- **Change review** (a separate independent reviewer, DBA focus, read-only):
  - round 1 FAIL: M1, a stale cross-run report;
  - round 2 PASS, with lows recorded.

Details are in plan.md.

# Residuals and exposure blockers

- Retention periods, grace, hold owners, deletion targets, backup windows and tombstone expiry are owner and legal decisions. Every rule here is provisional.
- Not built here:
  - storage adapters and physical deletion;
  - archive verification and key destruction;
  - replaying tombstones into a restored database;
  - deterministic ingestion IDs;
  - wiring the Assistant, ingestion, evidence and claim modules;
  - tenant-scoped authorization, persistence and screens.
- Hold reads scan all artifacts under one lock.
- Per-artifact store results are shared across runs.
- RET-01 to RET-08, AUD-AR-05 and AUD-QA-10 are NOT_RUN at system level.
