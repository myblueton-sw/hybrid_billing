---
type: VerificationRecord
title: HB-288 verification of the P04 retention API
description: Format, vet, unit and edge tests with coverage, race runs, cross-platform builds, mutation testing, a smoke run of the built binary and the outcomes of the design and change reviews for the P04 retention API over modules/retention. Run on mac_mini on 2026-11-17.
status: final
---

# Scope

This record verifies [HB-288](https://linear.app/hybrid-billing/issue/HB-288) against its [plan](plan.md). Checks ran on mac_mini (macOS, darwin/arm64) with the Go toolchain from `go.mod`, using only synthetic sample data. The plan was drafted and the code written by delegated agents in the ticket worktree; root re-ran the checks marked below on the final code.

# Results

| Check | Result | Run by |
| --- | --- | --- |
| `gofmt -l` on changed files | nothing listed | root |
| `go vet ./...` | clean | root |
| `go test ./...` | all packages ok | root |
| Coverage | `apps/api/server` 100.0%, `modules/retention` 100.0% | root |
| Cross builds of `./apps/api/cmd/hb-api` | linux/amd64, windows/amd64, darwin/arm64 build | root |
| Smoke run of the built binary (fresh 0700 data dir, sample seed) | read 200 at r18 with 7 policies, 2 holds, 2 runs, 7 proposals, watermark 2; run/1 stores database 4 = 2 deleted + 2 excluded, object store 2 = 1 deleted + 1 failed; `retention.stop_run` on `retention-run-2` 200 `done` with 2 stopped at r19; re-read run/2 object store 3 = 1 deleted + 2 excluded | root |
| `go test -race -count=40 -timeout 45m -run Retention ./apps/api/server` | ok (1548 s) | implementer, final code |
| `go test -race -count=3 -timeout 30m ./apps/api/server ./modules/retention` | ok | implementer, before the last test-only and hold-count changes |
| Mutation testing of `retention.go`, `retentioncmds.go`, `retentionseed.go` and `modules/retention/exports.go` | first run 423 mutants, 406 killed; after the review fixes and new tests a full re-run of 440 mutants, 438 killed; the two survivors and the ten mutants of the later hold-count branch were each applied and shown to fail tests | root (full runs), implementer (single mutants) |

No Go test file contains the forbidden substring, and no changed file exceeds 1,500 lines.

# Reviews

These reviews were done by AI agents, not the owner.

- **Design review** (independent reviewer, read-only): round 1 CHANGES REQUIRED (six Mediums: other-tenant run IDs, all-tenant held and referenced counts, per-store pending including excluded artifacts, deletion shown as confirmed without a durable decision record, the hold owner membership rule, an unreachable `retired` test); round 2 one Medium (the read used `s.caller`, which is zero outside commands) and one Low (eligible approvers by role instead of grant); round 3 PASS.
- **Change review** (independent reviewer, read-only): round 1 CHANGES REQUIRED (Medium: another tenant's plan proposal visible with its people and rationale; Lows: run counts over hidden artifacts, outcome paging before filtering, unbounded holds); round 2 PASS; round 3 PASS on the decision that hold counts are withheld (`counts: null`, `counts_scope: other_scope`) when a hold reaches artifacts or tenants the reader cannot see.

No review is owner acceptance.

# Residuals

- All six commands use the installation administration grant rather than P04's separate permissions.
- Not connected: deletion plan proposal and approval, the deletion worker (claim, proceed, report, retry), backup expiry, archive and restore, usage and byte counts, and a durable decision ledger; every run, tombstone and blocked or deleted artifact reads `decision: not_connected`.
- Policy proposals from the console use synthetic values; whether real retention periods stay read-only until decided is the owner's call.
- With two administrators, a hold placed from the console can never be released (the placer cannot propose or approve); the read shows `no_eligible_approver`. The sample member places the sample temporary-files hold in the seed only, a deviation from the HB-257 seed actor rule.
- Gaps in tombstone sequence numbers reveal how many other-tenant deletions happened.
- The watermark and impact count and digest are installation-wide and are shown only to the boot workspace administrator.
- Live holds are capped at 100 per workspace; released holds are kept.
- Injection-only paths: `retireRetention` retires on any not-found answer, including a missing hold, which cannot happen because holds are never removed; an approval made directly in the module by the same approver makes a later API approve answer `applied` with no module change; `-0` is accepted as grace 0; the ID-probe fault test depends on the hold count.
- The P04 screen binding follows in its own ticket; `docs/ui/screen-binding-register.md` is not updated, as in HB-286.
