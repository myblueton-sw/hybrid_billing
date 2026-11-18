---
type: VerificationRecord
title: HYB-303 verification of IAM-01 concurrent first-administrator creation and acceptance from another workspace
description: Commands run and results observed for the HYB-303 Bootstrap guard and the IAM-01 concurrency, two-workspace, two-installation and restart tests, with per-mutation kill results, coverage before and after, and what was not run.
status: stable
tags: [HYB-303, security, identity, IAM-01, verification]
sources:
  - resource: plan.md
  - resource: docs/planning/v1.16/retention-access.md
---

# Scope

Verifies the [plan](plan.md) (approved after design review round 3) on branch `feat/HYB-303-iam01-bootstrap-race`, base main `bc0d03e`, machine mac_mini, Go 1.25.7 darwin/arm64.

**Change set:**
- `modules/identity/boundary.go`: `Bootstrap` refuses with `bootstrap_state` when any user exists or the triple is already in `r.identities`. Three lines, placed after the armed, consumed and locked check and before the validity and verifier checks.
- New tests:
  - `modules/identity/bootstrap_race_test.go` (299 lines): C3, C4, C4b, C6, C6b, C7, C8;
  - `apps/api/server/iam01_test.go` (275 lines): C1, C2, C5, C9, C10 with C10b, C11.
- Dated pointers in `HB-259/verification.md:109`, `HB-263/verification.md:204` and the HYB-301 report gap 15. The original text is unchanged.

Only the security change review of this diff is still to come. No console or copy change.

# Results

| Check | Command | Observed |
| --- | --- | --- |
| New tests on unchanged product code (the guard removed) | `go test -race -count=1 -run '<the 13 new tests>'` per package | Identity: 6 of 7 pass. `TestBootstrapRefusesAnExistingUserOrLinkedLogin` (C4b) fails as expected, because it needs the guard: "a bootstrap with a user present: error <nil>", "the linked login was taken over by root". Server: all 6 pass. No defect found in the existing behaviour |
| New tests with the guard | `go test -race -count=50` on the 7 identity tests and on the 6 server tests | `ok` for both packages |
| Whole repository | `go test ./...` | 73 packages `ok`; no failing package |
| Formatting | `gofmt -l modules/identity apps/api/server` | no output |
| Vet | `GOOS=linux\|darwin\|windows go vet ./...` | all three pass |
| Coverage, identity | `go test -coverprofile` before (base `boundary.go`, without the new test file) and after | Before 988/999 statements (98.9%). After 990/1001 (98.9%). The two guard statements are covered, and the 11 uncovered statements are the same pre-existing ones. Not at the requested 99.0%; see Deviations |
| Coverage, server | `go test -coverprofile ./apps/api/server` | 8338/8338 (100.0%), unchanged |
| File rules | `wc -l`; search for the forbidden substring; search for source comments | 299, 275 and 274 lines; 0 hits; no comments |
| Plan command | `go test -race -short ./...` | exit 0; 73 packages `ok` |
| Plan command | `go test -race -count=10 ./modules/identity ./apps/api/server` | First run, default 10-minute timeout: identity `ok`, server `FAIL` at 600.295 s (the default timeout expired; only the tail was kept, so no failing test was identified). Re-run as written with `-timeout 90m`: identity `ok` (21.881 s), server `ok` (4847.604 s), exit 0 |

# Change review round 1 (PASS with 5 Low) and fixes

| Finding | Fix |
| --- | --- |
| F1 | C2 waits with `select` on `entered` or an early reply from the first request, and fails if that reply arrives first. `release` is closed through `sync.Once`, also in `t.Cleanup` |
| F2 | C9 fails when the row is missing, or when `invitation_expires` is missing or empty |
| F3 | Three assertions added: C1 checks there is exactly one effective administrator membership (`admin`) in the boot workspace; C10b checks the only member at B is `admin`; C4 checks, through package-internal access, that no identity other than `triple("root")` belongs to `root` and that `root` has one linked identity. C2's limitation is recorded below |
| F4 | The plan's commands were run as written; see Results |
| F5 | The C10, C10b and C11 worlds close their servers in `t.Cleanup` |
| I2 | The HYB-301 gap 15 pointer also notes that `retention-access.md:49` is not fully covered (HYB-303 R1) |

After these fixes:
- the 13 new tests pass `-race -count=50` again (`ok` for both packages);
- the mutations of the changed tests (M1, M1′, M2′, M3, M6, M8, M9, M12) were re-run, with the same results as the table below (M1 and M3 survive, the rest are killed);
- the four product files hash the same as before the batch.

**C2 limitation.** Every activation runs as the user ID `admin`, and the installation, trust and bulk grants are keyed to that user ID, not to a login. C2 therefore cannot show that the losing login received no grant. It shows that the losing login has no identity, password or directory entry, and the single winner is the only login linked to `admin`.

# Mutation evidence

For each mutation:
1. A script backed up the touched files and applied the mutation.
2. It ran each named new test **alone** with `go test -race -count=20 -run '^Name$'`.
3. It restored the files and compared them byte for byte with the backup. All restored: `True`.
4. It deleted the session `GOCACHE` after each mutation.

After the batch, the MD5 hashes of `boundary.go`, `signin.go`, `workspace.go` and `installation.go` equal those taken before it.

| Mutation | Test run alone | Result | First failure line |
| --- | --- | --- | --- |
| M1: drop `r.boot.consumed \|\|` in `Bootstrap` | C3, C4 | survived (expected; the guard refuses) | none |
| M1′: M1 plus drop the guard | C3 | killed | "two bootstraps succeeded: u0 and u1" |
| M2′: M1′ plus drop the re-check at `signin.go:463` | C1 | killed | "two activations succeeded" |
| M2′ | C2 | killed | "got 204 want 422 link_invalid" |
| M3: drop only the re-check at `signin.go:463` | C1, C2 | survived (expected; the identity layer refuses) | none |
| M4: drop `found.workspace != workspace` in `Accept` | C7 | killed | "request is refused" (the w1 acceptance lost to a w2 acceptance) |
| M5: drop the invitee check | C8 | killed | "bob with the right verifier: error <nil>" |
| M6: drop `workspace == s.bootWorkspace &&` at `signin.go:268` | C9 | killed | the row shows an active administrator membership after the other-workspace sign-in |
| M7: drop the state check in `Accept` | C6b | killed | "replaying the accepted invitation: error <nil>" |
| M8: drop the consumed check in `CreateUser` | C4 | killed | "creating the bootstrap user ID before the bootstrap: error <nil>" |
| M9: drop `r.boot.armed \|\|` in `ArmBootstrap` | C4 | killed | "arming again before the bootstrap: error <nil>" |
| M10a: drop the user-exists half of the guard | C4b | killed | "a bootstrap with a user present: error <nil>" |
| M10b: drop the linked-login half of the guard | C4b | killed | "a bootstrap of a linked login: error <nil>" |
| M11: move the invitee check after the verifier check | C8 | killed | "bob with a wrong verifier 0: … verifier_mismatch" |
| M12: drop `s.boot.live = false` | C1 | killed | "peek after activation: got 200 {"purpose":"activate"}" |

The only survivors are M1 and M3, the two expected defence-in-depth layers. Neither is dead code: each is the sole gate whenever the other layer is removed (M1′, M2′).

# Requirement trace

| ID | Status | Reason |
| --- | --- | --- |
| IAM-01 (`retention-access.md:110`) | covered for the pre-launch scope: local password accounts, one workspace per installation | Concurrent first-administrator creation is refused at the API (C1, C2) and in the module (C3, C4, C4b). Invitation replay is refused, including after removal (C6, C6b). Acceptance naming another workspace (C7), by a user of another workspace (C8), through a sign-in to another workspace (C9) and across installations (C10, C10b) is refused |
| `retention-access.md:49` | **not fully covered** | One-time within a run, and an earlier run's link is refused (C11). Re-arming after a restart remains residual R1 of the plan |

# Deviations

- **Identity coverage is 98.9%, not the requested 99.0% or more.** The baseline at `bc0d03e` was already 98.9% (988/999), and the change did not lower it (990/1001). The 11 uncovered statements are pre-existing and outside this ticket. They are in `OpenSession`, `UnlinkIdentity`, `DepartUser`, `Users`, `New`, `RevokeInvitation`, `Accept` (`workspace.go:129`, the version overflow), `serviceState`, `AddCredential` and `RevokeCredential`. Raising the figure needs tests outside the IAM-01 scope; this is left for root to decide.
- **A test helper was renamed.** The plan named no helpers. The world helper planned as `activated` clashed with an existing method in `trust_restart_test.go`, so it is `firstAdmin`.
- **C10b creates the set link with `created`.** The administrator's session comes from `openFor` in the boot workspace, as in `TestSeedNoneActivationGrantsAdministration`.

# Not run

- The independent security change review of this diff.
- Windows file locking (R4): only `go vet` ran for windows; no Windows test run.
- The R4 behaviour of a second process deleting `activation.txt`: not tested, as the plan says.
- The optional sample+sample pair of C10b, which the plan left optional.

## Root final verification (2026-11-18, rebased on main 82c9272)

- Change review (independent, security-focused): PASS with five Low findings and one Info; all fixed (F1–F5, I2).
- Root re-ran after rebase: gofmt clean; go vet ok; targeted `-race -count=5` over the bootstrap, invitation, activation and restart tests ok; `go test -short ./...` all packages ok; coverage identity 98.9% (equal to the main baseline, the 11 uncovered statements predate this ticket and lie outside IAM-01 — accepted by root, not raised here), server 100.0%.
- The plan's `-race -count=10` regression command was amended to add `-timeout 90m`, because the server package needs about 81 minutes under race ×10.
