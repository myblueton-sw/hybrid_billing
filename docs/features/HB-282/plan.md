---
type: ImplementationPlan
title: P01 installation API over modules/installer with a synthetic sample site, release and plan
description: Phase 5a of the pre-launch plan (HB-257, HB-262 scope). Adds an installation read (site, installed state, releases with marks, proposals, install and upgrade previews, diagnostics destinations and reports with guidance) and administrator commands with a reason to approve or withdraw installer proposals and propose acceptance of the current install plan. Nothing changes a real host, downloads anything or holds a real key.
status: stable
---

# Scope and ownership

[HB-282](https://linear.app/hybrid-billing/issue/HB-282) adds the API behind P01 (installation and diagnostics). Base: main `1f656b0`. The P01 screen follows in a separate ticket. P01's support bundle and support access tabs are bound later with `modules/support` and are out of scope here.

**Ownership**
- Root (Claude Code) owns the plan and integration. Implementation may be delegated to one writer in this worktree.
- An independent read-only reviewer reviews the design and the change, with a security focus (keys, signatures, approvals, no host effects).
- Machine: mac_mini.

**Allowed files**
- `modules/installer/releases.go`: two exported wrappers, `Release.SigningMessage()` returning `encode()` and `KeyChallenge(version)` returning `challenge(version)`, plus their tests. No rule changes.
- `apps/api/server`: new `installation.go`, `installseed.go` and tests; registration in `admin.go`; routes in `boundary.go`; limits and the registry in `server.go`; the seed step in `seed.go`; the seed step count in tests.
- `docs/features/HB-282/*`.

# Observed baseline (`modules/installer`)

- Changes are proposals: `Propose(change, human)` then `Approve(number, other human)`. Self-approval is refused (`self_approval`), assistants and system actors cannot propose or approve, and `Withdraw` is limited to the proposer. Approval re-checks the change and applies it.
- Change kinds: `key` (ed25519 public key with a proof over `challenge(version)` and a bounded life), `release` (signed over `encode()` by an active key), `site`, `probe`, `config` (port and volume overrides of declared ones), `destination` (diagnostics allowlist: host, port, address ranges), `plan`, `upgrade`, `rollback` (acceptance of a preview hash).
- `Observe(site, observation, probe actor)` records the host inventory from the site's probe; `Preview(site, version)` returns the install plan (services, ports, volumes, data effects, permissions, conflicts, hash); `PreviewUpgrade` and `PreviewRollback` need an installed site.
- Runs (`StartRun`, `Complete`, `Interrupt`, `Resume`, `Cancel`, bootstrap codes) represent real host changes executed by the installer and its probe.
- Diagnostics: `CheckTarget(site, destination, actor)` checks an approved destination; `Report(site, destination, result, probe actor)` records a probe result and returns guidance; `Reports(site)` lists results.
- Errors: `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrLimit`, and `Refusal{Reason}` wrapping `ErrRejected` with a stable reason code.

# Design

## Registry and limits

`installer.NewRegistry(Config{Random: crypto/rand.Reader}, limits, clock)` with Keys 4, KeyLife 400 days, Releases 32, Sites 4, Destinations 16, Reports 200, Runs 16, ObservationLife 24 hours, BootstrapLife 1 hour, Proposals 64. All values are inside the module bounds.

## Sample seed (only with `--seed sample`; nothing is created with `--seed none`)

- Actors: the first sample administrator proposes and the second sample administrator approves, as in the authentication policy seed. The probe actor is `{ID: sample-probe, Kind: system}` and is built only inside the seed function.
- Release key `sample-key-1`: an ed25519 key pair generated at start from `crypto/rand`; the private key stays a local variable of the seed function (never stored on the server, logged or written). Proof over `installer.KeyChallenge`. Valid from the seed time for 180 days.
- Releases `1.0.0` and `1.1.0` (`1.1.0` upgrades from `1.0.0`), signed over `Release.SigningMessage()` by the generated key:
  - platforms in the module's order: darwin/arm64, linux/amd64, linux/arm64, windows/amd64;
  - artifacts `api` (container image) and `worker` (archive), digests are SHA-256 hex of `sample-api-<version>` and `sample-worker-<version>`;
  - services `api` (artifact `api`, port 8443/tcp, volumes `config`, `ledger`) and `worker` (artifact `worker`, port 9000/tcp, volume `ledger`);
  - volumes `config` (not data) and `ledger` (data).
- Site `sample-site` (tenant = the boot workspace, linux/amd64, probe `sample-probe`), then a complete empty observation from the probe.
- Destination `sample-erp-endpoint` (`erp.example.invalid:443`, range `192.0.2.0/24`) applied, and one probe report for it: kind `tls`, failure `unknown_authority`, address `192.0.2.10`; the returned guidance is stored with the report.
- Two pending proposals so the screen has work:
  - (a) plan acceptance of `1.0.0` for `sample-site` with the current preview hash, `Admin` = the second sample administrator, proposed by the second sample administrator (the first can approve);
  - (b) destination `sample-billing-export` (`export.example.invalid:443`, `198.51.100.0/24`) proposed by the first sample administrator (the second can approve). A destination does not clear the observation, so approving it keeps the plan valid.
- No run is started; the site reads not installed.

## Server record of proposals

The module has no list read, deletes withdrawn proposals and prunes applied ones. The server keeps its own store of installer proposal numbers it created or saw (number, kind, scope, created time, proposer, rationale, closing state), separate from the policy records. Commands accept only numbers in this store, and the read lists this store (newest first, at most 50), showing withdrawn ones as closed.

## Read (admin only, boot workspace only)

`GET /api/v1/installation` returns, or a `not_configured` shape when no site exists:
- `revision`, `now`;
- `site`: ID, platform, probe ID (an identifier, not a credential), config overrides;
- `observation`: the observed time and expiry from the server's record of the seed observation, and the state `fresh`, `stale` or `missing` derived from a dry-run `Preview` (`observation_stale`, `ErrNotFound`), so the read cannot drift from the module;
- `installed`: `not_installed`, or version, state, owned services and volumes, history and backups;
- `releases`: version, key version, upgrade-from, platforms, services with ports and volumes, volumes with the data flag, artifact names and kinds, and the mark (by, reason, confirmed by). No public key, proof, signature or digest;
- `preview`: the install plan for the version of the newest pending plan acceptance, otherwise the newest release not installed: kind, version, hash, services, ports, volumes, data effects, permissions, leftovers, conflicts; or `unavailable` with `not_observed`, `observation_stale` or the refusal reason;
- `upgrade` and `rollback`: previews when installed, otherwise `not_applicable`;
- `proposals`: number, kind, state (`pending`, `applied`, `withdrawn`), proposer and approver (ID and login), rationale, a summary (key version; release version; site ID; destination host, port and ranges; plan version and hash), and for the reader `can_approve`/`can_withdraw` with a blocker: `own_proposal`, `not_proposer`, `not_pending`, and for plan acceptances a dry-run check: `plan_changed` (with the current hash), `observation_stale`, `not_observed`, `conflicts`, or any other refusal reason passed through;
- `destinations`: ID, host, port, ranges from the server's record of applied destination proposals;
- `reports`: kind, failure, address, via, guidance (cause and steps) recorded at report time;
- `runs`: `none`.

## Commands (installation administration, reason required, actor = the reader as a human)

| Command | Target | Fields | Module call |
| --- | --- | --- | --- |
| `installation.approve` | `installation-proposal-<n>` | none | `Approve(n, reader)` |
| `installation.withdraw` | `installation-proposal-<n>` | none | `Withdraw(n, reader)` |
| `installation.propose_plan` | `installation` | `version`, `hash` | `Propose(plan acceptance for the site, Admin = reader)` |

- `Resolve` returns not found unless the workspace is the boot workspace and, for proposal targets, the number is in the server store, and for `installation`, the site exists.
- No command builds a system or assistant actor; a test enforces it. There is no diagnostics command: probes are not reachable from the console, and no endpoint signs anything.
- Before the module call the server checks that the release exists for `propose_plan` (`invalid_request` on `version`) and that the hash is 64 lowercase hex characters (`invalid_request` on `hash`).
- Error mapping: `Refusal` `self_approval` or `not_proposer` → `prerequisite_failed`; any other `Refusal` → `invalid_stage`; `ErrNotFound` after the prechecks (missing or stale observation, a vanished proposal) → `invalid_stage`; `ErrLimit` → `prerequisite_failed`; `ErrConflict` (a duplicate identifier) → `invalid_stage`; `ErrInvalid` → `internal_error` (the server built a bad input). The read's blockers explain the stage refusals, because these codes carry no fields.
- A duplicate `propose_plan` by the same administrator returns the existing proposal (module behaviour).
- Rationale is capped at 256 bytes and kept in the server store with the actor.

## Out of scope (residuals)

- Runs, bootstrap codes, interrupts and resumes (real host changes); key retirement, release revocation and compromise marks; probe observations and reports as commands. Approval does not install anything.
- No observe command: after 24 hours the observation is stale and plan approvals and new plan proposals are refused until restart; this is shown in the read.
- Probe IDs are identifiers, not credentials; a future probe API must authenticate probes. No endpoint may ever sign caller-supplied data.
- Support bundles and support access (`modules/support`), bound later.
- The P01 "diagnostic request" action is not bound; it needs an authenticated probe.
- Real keys, vendor releases, downloads, network diagnostics; persistence.

# Verification plan

- Module: tests for the two wrappers (signing the wrapper output verifies through `Propose`/`Approve`).
- API: read shape (and `--seed none`), each command's success and refusals (self-approval, not proposer, not pending, approve after withdraw, plan changed with the hint, stale observation via the clock, unknown number, unknown release, bad hash), duplicate plan proposal, non-admin and other-workspace refusals, idempotent replay, stale revision, a JSON leak test (no key bytes, proof, signature or digest), a test that no command builds a system actor, fault hooks and seed failure paths for every module call.
- `go test -race -count=40` on the new tests, cross builds, 100% coverage of the new files, mutation testing of `installation.go`.
- Independent design and change review; `verification.md`.
