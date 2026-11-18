---
type: VerificationRecord
title: HB-286 verification of the P03 recovery API
description: Tests, coverage, race, cross builds, mutation results and review outcomes for the recovery read, the four approve, withdraw, lock and propose-unlock commands, the recovery module's read-only exports and the sample seed. Run on mac_mini on 2026-11-17; the test clock is fixed at 2026-11-16.
status: stable
---

# Scope

This record verifies [HB-286](https://linear.app/hybrid-billing/issue/HB-286) against its [plan](plan.md). Every check ran on mac_mini (macOS) with Go 1.25.7, using synthetic data only. The plan was drafted and the implementation written by delegated agents in the ticket worktree; root re-ran the checks below and applied the last change-review Low and one mutation test.

# Results

| Check | Result |
| --- | --- |
| `gofmt -l` and `go vet` | clean |
| `go test ./apps/api/...` | pass; `server` 100.0% statement coverage, every function in `recovery.go`, `recoverycmds.go` and `recoveryseed.go` 100% |
| `go test ./modules/recovery/` | pass, 100.0% statement coverage, including `exports_test.go` (`Check` compared with `Propose`/`Approve` across the refusals with no state change; `Status` transitions and errors) |
| `go test -race -count=40 -timeout 45m -run Recovery ./apps/api/server` | pass (1368 s) |
| Cross builds linux/amd64, windows/amd64, darwin/arm64 | build |
| Mutation testing of the three new server files (179 mutants) | 176 killed, 3 survived. One survivor (the tenant lookup for an installation proposal) was killed by an added test and re-run on its own. The other two are equivalent: a fault-hook error value, and the tamper condition in the sample manifest, which builds both the expected and observed inventories so the inverted condition still leaves only `outbox` mismatched. |

**What the tests cover**
- Read shape for sets, isolation, reconciliation, migration (not connected) and resume, with `--seed none`, another workspace, another tenant and a member; `Status` as the authority for state and generation after every transition; `other_scope_present` as a boolean; no chunk checksums, permit tokens or event keys (hex, uppercase, base64 and raw, plus the tampered checksum and its label).
- The four commands: approve only for unlock proposals (injected pending begin and prune are refused before any module call), withdraw by the proposer only and refused on non-pending entries, lock refused unless serving, one pending unlock per installation from any proposer, state checked before the lost digest, the lock-release lost digest, retired entries, replay after state changes, `internal_error` replayed with the revision unchanged, non-admin and other-workspace refusals.
- AST tests: no system or assistant actors outside the seed, no `RemoveWriter`, no restricted module calls from requests, no `recovery.Change` kind other than unlock outside the seed, the exact four command IDs.
- Fault hooks for reads and commands and seed failures at every module call; concurrent reads, commands and `Check` under the race detector.

# Reviews

These reviews were done by AI agents, not the owner.

- **Design review** (independent reviewer, read-only, DBA and security focus): round 1 CHANGES REQUIRED (five Mediums: approve could apply restore or delete proposals, a server copy of installation state, the duplicate unlock no-op, the dropped writer removal still described, the tenant check); round 2 PASS, with its three clarifications folded into the plan. Root decided to leave writer removal out (irreversible and single-administrator), to show other workspaces' calls as a boolean only, and to label a standby unlock as a lock release.
- **Change review** (independent reviewer, read-only, DBA and security focus): PASS with four Lows and three Nits. The implementer's deviations (a narrow widening of the HB-284 AST rule for the recovery seed's report, `open_calls_source`, the `none` recovery object, reconciliation configured only with an open recovery, `high_water` null without a series, the shared first `internal_error` revision move, the unreachable `lost` blocker in the read, locked treated as no open recovery) were accepted. L1 (withdraw of a non-pending entry answers `invalid_stage`) was applied with a test.

No review is owner acceptance.

# Residuals

- Begin, abandon and prune (restore, discard and delete), every writer and verifier path, registrations, and writer removal (single administrator and irreversible; needs a two-person design) are not commands.
- A lock release checks no decision watermark; open calls are listed only while a recovery is open.
- Migration is not connected; binding `modules/migration` read-only is a later ticket. All commands use `AdministerInstallation`, a recorded deviation from P03's five permissions; AU and BL read access is not provided.
- Change-review Lows left as they are: retiring an entry (test injection only) changes the read without moving the revision; the AST guards are syntactic; another workspace's administrator is simulated in tests. The 200-entry store bound and persistence are residuals.

# Planning cross-check (HYB-291)

Added by the [HYB-291 planning cross-check](../../reviews/HYB-291-planning-cross-check/report.md) on 2026-11-17. A record added by HYB-291; it changes no result above except where marked. The findings come from a read-only auditor; the HYB-291 writer agent (delegated by root) wrote this trace, the residual notes and the copy changes; the HYB-291 change review is their first independent check.

## Requirement trace

Source: `docs/ui/screens/P03/spec.md:120-130` (HB-286 is the API, HB-287 the screen). Status meanings are in the [cross-check report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). No acceptance test was run.

| ID | Status | Reason | Record |
| --- | --- | --- | --- |
| HB-08 | deferred | Restore sets and reconciliation are read; durable evidence storage waits for the database decision. | HB-286/plan.md:222-232 |
| HB-21 | not connected | No billing application ledger is bound. | HB-286/plan.md:222-232 |
| HB-22 | not connected | Opening balances are not bound. | HB-286/plan.md:227 |
| HB-34 | not connected | Writer and verifier paths are not commands. | HB-286/verification.md Residuals |
| HB-37 | deferred | Isolation and lock-release commands are built; begin, abandon, prune, writer removal and migration are not connected. | HB-286/verification.md Residuals |
| HB-42 | deferred | All commands use installation administration, a deviation from P03's five permissions; AU and BL read access is not provided. | HB-286/verification.md Residuals |
| HB-43 | deferred | A lock release checks no decision watermark; tombstone reapplication after restore is not connected. | HB-286/verification.md Residuals |
| AUD-QA-09 | not connected | Opening balances (`modules/balances`) are not bound. | HB-286/plan.md:227 |
| RET-01–RET-08 | not connected | System-level retention tests are NOT_RUN; P03 did not cite them before this section. | this section |
