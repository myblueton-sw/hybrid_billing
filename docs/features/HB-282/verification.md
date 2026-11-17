---
type: VerificationRecord
title: HB-282 verification of the P01 installation API
description: Tests, coverage, race, cross builds, mutation results and review outcomes for the installation read, the installer proposal commands and the sample seed with a key generated at start. Run on mac_mini on 2026-11-17; the test clock is fixed at 2026-11-16.
status: final
---

# Scope

This record verifies [HB-282](https://linear.app/hybrid-billing/issue/HB-282) against its [plan](plan.md). Every check ran on mac_mini (macOS) with Go 1.25.7, using synthetic data only. The implementation was written by a delegated writer agent in the ticket worktree; root re-ran the checks below.

# Results

| Check | Result |
| --- | --- |
| `gofmt -l` and `go vet ./...` (apps/api, modules) | clean |
| `go test ./...` (apps/api) | pass; `server` 100.0% statement coverage, `installation.go` and `installseed.go` 100% per function |
| `go test ./installer/` (modules) | pass, 100.0% statement coverage, including `signing_test.go` for the two exported wrappers |
| `go test -race -count=40 -run Install ./server` | pass |
| Cross builds linux/amd64, windows/amd64, darwin/arm64 (and linux/darwin/windows amd64 by the writer) | build |
| Mutation testing of `installation.go` and `installseed.go` (131 mutants) | 123 killed, 8 survived. Four survivors (two loop boundaries in the plan-version choice and two in canonical target parsing) were killed by an added test and re-run on their own. The other four are equivalent: two `Required` flags the handler validates again, a fault-hook error value, and an `Installed` error that the module cannot return here. |

**What the tests cover**
- Read shape with the sample seed and with `--seed none` (`not_configured`), observation states (fresh, stale after 24 hours on the test clock, missing, unknown), previews and their unavailable reasons, proposal rows with blockers (`own_proposal`, `not_proposer`, `not_pending`, `plan_changed` with the current hash, `observation_stale`, `not_observed`, `conflicts`), destinations and reports with guidance.
- Commands: approve by the other administrator, self-approval refused, withdraw by the proposer only, approve after withdraw, `propose_plan` with an unknown release or a malformed hash, duplicate plan proposal returning the same number, non-admin refusal, other-workspace resolution, idempotent replay and stale revision.
- A leak test: the read contains no public key, proof, signature or artifact digest in hex, base64 or raw form.
- An AST test: request paths never name system or assistant actors; only the seed builds the probe actor.
- Fault hooks for every module call and seed failure paths; concurrent reads and commands under the race detector.

# Reviews

These reviews were done by AI agents, not the owner.

- **Design review** (independent reviewer, read-only, security focus): round 1 CHANGES REQUIRED (one High: the diagnostics check needed the system probe and would have made an administrator impersonate it; five Mediums: ambiguous not-found mapping, a config proposal that would disable the plan path, the observation lifetime, approval by raw number, invisible refusal reasons); round 2 PASS, with its Low items folded into the plan.
- **Change review** (independent reviewer, read-only, security focus): PASS. Key handling, the absence of signing endpoints and host effects, human-only actors from requests, separation of duties, tenant scoping, leaks, the dry-run blockers and the error mapping were confirmed by reading. The implementer's deviations were accepted.

No review is owner acceptance.

# Residuals

- Runs, bootstrap codes, key retirement, release revocation and compromise marks, observations and reports are not commands; approval installs nothing.
- The seeded observation becomes stale after 24 hours and cannot be refreshed until restart.
- The P01 diagnostic request action and the support bundle and access tabs are not bound.
- Probe IDs are identifiers, not credentials; no endpoint may sign caller-supplied data.
- Change-review Lows left as they are: `unknown` observation when the install target is refused before the observation check; any non-refusal preview error reads `not_observed`; `config_overrides` is always empty because no config command exists; destination proposals have no dry-run blocker; a repeated approval by the same approver moves the server revision. The ignored `ed25519.GenerateKey` error with `crypto/rand` and the stored public verification material (never shown) are accepted.
