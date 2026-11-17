---
type: VerificationRecord
title: HB-284 verification of the P02 integrations API
description: Tests, coverage, race, cross builds, mutation results and review outcomes for the integrations read, the fifteen plugin, subscription and dead-letter commands, the plugins signing wrappers and dry run, and the sample seed with keys generated at start and a narrowly scoped seed signer. Run on mac_mini on 2026-11-17; the test clock is fixed at 2026-11-16.
status: final
---

# Scope

This record verifies [HB-284](https://linear.app/hybrid-billing/issue/HB-284) against its [plan](plan.md). Every check ran on mac_mini (macOS) with Go 1.25.7, using synthetic data only. The plan was drafted by a delegated agent and the implementation written by a delegated writer agent in the ticket worktree; root re-ran the checks below.

# Results

| Check | Result |
| --- | --- |
| `gofmt -l` and `go vet` | clean |
| `go test ./apps/api/...` | pass; `server` 100.0% statement coverage, every function in `integrations.go`, `integrationcmds.go`, `integrationseed.go` and `integrationdeps.go` 100% |
| `go test ./modules/plugins/` | pass, 100.0% statement coverage, including `signing_test.go` (wrappers and `Recheck` compared with `Approve` across fifteen cases, with no state change) |
| `go test -race -count=40 -timeout 30m -run Integration ./apps/api/server` | pass (872 s). A first run without `-timeout` stopped at the default 10-minute limit; it was not a race. |
| Cross builds linux/amd64, windows/amd64, darwin/arm64 | build |
| Mutation testing of the four new server files (339 mutants) | 328 killed, 11 survived. Three survivors (the type-set length check and two parts of the modules-section state condition) were killed by added tests and re-run on their own. The other eight are equivalent: a sort comparison over unique sequences, two `Required` flags the handler validates again, the explicit disarm before the last report (the deferred disarm follows and the report does not sign), a fault-hook error value, an error break in the service-user loop that cannot be reached with a non-empty list, a stand-in result returned together with an error that callers check first, and the duplicate rule for the original proposer of an active subscription (the module refuses the reused identifier with the same code). |

**What the tests cover**
- Read shape for clients (console client, hidden installation grants, service users pointing to M04), modules (publishers and keys without key material, versions with digests, installations, proposals with dry-run blockers), channels (subscriptions with cursor, lag, gaps, active key versions, history) and events (types, outbox bounds, deliveries, dead letters and pending requests), with `--seed none`, another workspace and a member.
- All fifteen commands: success, self-approval and not-proposer refusals, the five silent module acceptances refused by the server without moving the revision, pending-request refusals for resume, redelivery and dismissal (same administrator and second administrator), duplicate subscriptions (another proposer, an active original, the original proposer), the inactive destination, purged subscriptions, idempotent replay, stale revision, non-admin and other-workspace refusals.
- A leak test over the read and every command result (public keys, proof, signatures, the captured webhook private key, event bodies; hex, base64 and raw).
- AST tests: runtime, delivery, key and publish calls only in the seed; system and assistant actors only in the seed and the outbox reader; no `system`/`assistant` literals in request files; no network imports in the stand-ins; the exact set of fifteen command IDs.
- The seed signer: signs only for the one sample triple, is disarmed after the last confirm and after every seed failure, its held key slice is zero after start, and key approval afterwards answers `dependency_unavailable`. The webhook private key is also cleared on every exit path.

# Reviews

These reviews were done by AI agents, not the owner.

- **Design review** (independent reviewer, read-only, security focus): round 1 CHANGES REQUIRED (five Mediums: a fifth silent no-op, overwritten pending requests, actors for Resolve and reads, the P02 permission deviation, the signer scope); round 2 PASS, with three Low clarifications folded into the plan.
- **Change review** (independent reviewer, read-only, security focus): PASS with four Lows and four Nits. The implementer's seven deviations were accepted. L1 (clearing the webhook private key when the seed fails before arming) was applied.

No review is owner acceptance.

# Residuals

- All integrations commands use `AdministerInstallation`, a recorded deviation from the P02 permission split; the three permission groups make the change one line each.
- Dispatch, confirm, redirect, report, token issuance and checks, egress and endpoint checks, publisher, key, version and endpoint registration, marks, and webhook key management are not commands. Without marks, a compromised publisher is contained only by a single administrator using stop or disable.
- Destinations and permits are server stand-ins over a fixed sample allowlist; `modules/trust` is not bound.
- Change-review Lows left as they are: paused or suspended subscriptions do not count as duplicates; the read builds the outbox reader per request; the AST checks match package qualifiers by name; another workspace's administrator is simulated in tests. Nits accepted: arming replaces keys without clearing earlier ones (the seed arms once), the ignored `ed25519.GenerateKey` error with `crypto/rand`, the shared not-pending reason string, and the letter mirror's dependence on requests never dispatching.
- Zeroing keys is best effort; persistence is out of scope.

# Planning cross-check (HYB-291)

Added by the [HYB-291 planning cross-check](../../reviews/HYB-291-planning-cross-check/report.md) on 2026-11-17. A record added by HYB-291; it changes no result above except where marked. The findings come from a read-only auditor; the HYB-291 writer agent (delegated by root) wrote this trace, the residual notes and the copy changes; the HYB-291 change review is their first independent check. Rows marked "(classified by HYB-301)" replace rows that HYB-291 left unclassified; the [HYB-301 trace classification](../../reviews/HYB-301-trace-classification/report.md) gives the evidence.

## Requirement trace

Source: `docs/ui/screens/P02/spec.md:120-146` (HB-284 is the API, HB-285 the screen). Status meanings are in the [cross-check report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). No acceptance test was run.

| ID | Status | Reason | Record |
| --- | --- | --- | --- |
| HB-03 | deferred | Addressed within pre-launch scope for installation-administration commands, a recorded deviation from P02's permission split; registry-wide and leakage acceptance not built. | HB-284/plan.md:363-386 |
| HB-05 | not connected | No source connectors. | HB-284/plan.md:363-386 |
| HB-06 | not connected | Dispatch and receipt are not commands. | HB-284/plan.md:363-386 |
| HB-08 | not connected | No durable evidence storage. | HB-284/plan.md:363-386 |
| HB-10 | not connected | HawkEye results are not connected. | HB-284/plan.md:363-386 |
| HB-11 | not connected | TokenMeter results are not connected. | HB-284/plan.md:363-386 |
| HB-24 | not connected | No ERP connection. | HB-284/plan.md:363-386 |
| HB-31 | deferred | (classified by HYB-301) P02's part is the subscriptions and channels tab, with API, module and notification-channel permissions not merged (`P02/spec.md:33, 80`). Built: webhook subscriptions with approval, pause, resume and revoke, and delivery states (HB-284/plan.md:80-95). Not built: the separate API, module and channel permissions, an internal recorded deviation where every command uses `AdministerInstallation` (HB-284/plan.md:308-310, 376); notification channels and events, which need external messaging (out of pre-launch scope). | [HYB-301 report](../../reviews/HYB-301-trace-classification/report.md) |
| HB-33 | deferred | Covered: plugin and subscription commands use the common command path. Not built: CLI, AI and external consumer parity, 202 jobs for bulk work, and an API-only business cycle. | HB-284/plan.md |
| HB-34 | deferred | Deliveries are read; dispatch, confirm, redirect and report are not commands. | HB-284/verification.md Residuals |
| HB-35 | deferred | Plugin proposals are built; publisher, key, version registration and marks are not commands. | HB-284/verification.md Residuals |
| HB-37 | deferred | (classified by HYB-301) Built: each plugin version's storage contract (namespace, owner, delete on removal or keep for evidence for N days, migration kind) is shown, and install and upgrade are refused when it changes (HB-284/plan.md:47-55; HB-285/plan.md:306; `P02/spec.md:51`). Not built: holding webhook dispatch after a restore until external results are reconciled. The P03 recovery register has a writer lock (HB-286/plan.md:38, 49), but no apps/api integration file refers to it, and dispatch is not a command in pre-launch. | [HYB-301 report](../../reviews/HYB-301-trace-classification/report.md) |
| HB-40 | deferred | (classified by HYB-301) Built: plugin versions are signed by a registered publisher key, and a revoked publisher, a compromised key or a vulnerable version blocks install (HB-284/plan.md:47). Not built: the licence check that `P02/spec.md:50, 74` requires in the module manifest and before activation; `modules/plugins` has no licence field. | [HYB-301 report](../../reviews/HYB-301-trace-classification/report.md) |
| HB-42 | deferred | Separate integrations permissions are not built. | HB-284/plan.md:363-386 |
| AUD-AR-02 | not connected | Signing, egress and endpoint checks are not commands; destinations are stand-ins over a fixed allowlist. | HB-284/verification.md Residuals |
| AUD-QA-04 | deferred | Idempotency keys and expected revisions are enforced; token issuance is not built. | HB-284/verification.md Residuals |
| AUD-QA-05 | deferred | Event and delivery states are shown; dispatch, signing and replay checks are not connected. | HB-285/verification.md Residuals |
| AUD-QA-06 | deferred | External application origins are a later design. | HB-284/plan.md:385, HB-285/plan.md:688 |
