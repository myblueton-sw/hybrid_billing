---
type: VerificationRecord
title: HB-245 installer upgrade verification
description: Actual checks run for W-36 increment 2b (installer upgrades and software rollback), plus review outcomes and residuals.
status: final
---

# Scope

[HB-245](https://linear.app/hybrid-billing/issue/HB-245) extends `modules/installer`, following [plan.md](plan.md):
- **Ownership.** Owned-resource records are derived from completed runs, and the site has states `installed` and `upgrade_failed`.
- **Releases.** Releases are signed under tag v2 and include `UpgradeFrom`.
- **Previews.** Upgrade previews give replace, create, adopt, remove, reuse, retain, reattach and detach actions, guard data continuity by host and logical name, and detect drift.
- **Upgrade runs.** Each upgrade run has quiesce and backup checkpoints with a recorded scope. A site-level `data_touched` flag tracks the latest touching version and `data_form`.
- **Failures and rollback.** A failure union is kept across failed runs. A dual-controlled, previewed software-only rollback is possible only before any new-release service has started.
- **Acceptances.** Every acceptance kind is rechecked and retired consistently.
- **Out of scope.** Data restore belongs to W-37.

Base: main `e07771c`. Machine: mac_mini.

# Checks run (2026-11-15, this machine)

| Check | Result |
| --- | --- |
| `gofmt -l`, `go vet ./modules/installer/` | clean |
| `go test -race -count=10 ./modules/installer/` | pass (28.8 s) |
| `GOOS=linux GOARCH=386 go test -c -o /dev/null ./modules/installer/` | pass |
| `go test -race -count=1 ./...` | every package passes |
| Statement coverage | 100.0% |
| Upgrade property test | 150 seeds: upgrades, cancels and rollbacks. One measured run gave 940 starts, 92 completed upgrades, 334 failures and 56 rollbacks. It checks these invariants: the version changes only on completed upgrades, the failed state matches the model, data volumes stay owned, no rollback is offered after a service step, and no run is rolled back twice |
| HB-243 install tests and property test | still pass |

The release encoding v2 known-answer tests were cross-checked against an independent Python computation, for both empty and non-empty `UpgradeFrom`.

# Mutation testing

| Run | Mutants | Killed | Did not build | Survived |
| --- | --- | --- | --- | --- |
| First | 974 | 913 | 28 | 33 |
| After fixes | 984 | 945 | 28 | 11 |

The first run's survivors were closed with tests, or by removing redundant code.

Of the 11 survivors in the second run:
- 3 (the `finish` volume rules) were killed by `TestFinishVolumeRules`, re-run by hand;
- 8 are equivalent:
  - `Round(0)` → `Round(1)`, three times;
  - `<` → `<=` in three sort comparators over unique keys;
  - a `[2]int` → `[3]int` pair array;
  - the install entry of the resume kind map, which falls through to the same install preview.

# Reviews

- Design review (independent reviewer, read-only):
  - round 1 FAIL: data restore belongs to W-37, and rollback had no preview;
  - round 2 FAIL: the `data_touched` timing and scope;
  - round 3 FAIL: `data_form` across runs, and the detached cap;
  - round 4 PASS.
- Change review (a separate independent reviewer, read-only): round 1 FAIL (M1: missing detached volumes were reattached without being created), round 2 PASS. The notes are resolved in the plan.

# Residuals

- W-37: data restore, the write lock and reconciliation, and recovering from detached data volumes that block upgrades.
- Host quiesce and backup tooling, and checking backup content.
- Ownership by register record and name.
- Removing leftovers and detached volumes.
- Downgrades, persistence and screens.
- The upgrade property test uses one site and observations derived from register state. Literal tests cover forged and missing resources.
- Owner decisions not confirmed by the owner:
  - software rollback only before services start;
  - the cap of 32 detached volumes;
  - the history bounds.
