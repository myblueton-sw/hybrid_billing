---
type: VerificationRecord
title: HB-268 verification of the M04 service-account API
description: Module and apps/api tests, coverage, race, cross builds, mutation results and review outcomes for the credential metadata listing and lookup, the bound on ended credentials, the credential serial, and the service-account read and commands with one-time secret pickups. Run on mac_mini on 2026-11-16.
status: final
---

# Scope

This record verifies [HB-268](https://linear.app/hybrid-billing/issue/HB-268) against its [plan](plan.md). Every check below was run on mac_mini (macOS) with Go 1.25.7 in the HB-268 worktree, using the synthetic sample seed only.

# Results

| Check | Result |
| --- | --- |
| `gofmt -l` and `go vet` (apps/api/server, modules/identity) | clean |
| `go test -race -count=1 ./server/` (apps/api) | pass, 100.0% statement coverage |
| `go test -race -count=1 .` (modules/identity) | pass; package coverage 98.6%. The new `Credentials`, `Credential` and `pruneCredentials` are 100% covered; the uncovered statements were already uncovered on main, plus the serial-overflow return in `AddCredential` |
| `go test -race -count=40` on the new service, credential and pickup tests in both packages | pass |
| `GOOS=linux GOARCH=386` and `GOOS=windows` builds | pass |
| Mutation testing of the changed lines in services.go, users.go, workspace.go and boundary.go | 192 mutants: 183 killed and 9 survived. One survivor (a nil check in `Recheck`) was then killed by an added test, verified by re-running that mutant. The other 8 are listed below. |

**Surviving mutants, accepted as equivalent or unreachable**
- `services.go`, `findService` and `findCredential`: a `found` value returned together with an error. `Resolve` checks the error first, so `found` is never read.
- `services.go`, the `Required` flags on `name`, `owner` and `days`: the handler validates each field again and returns the same `invalid_request` with the same field, so the response does not change.
- `workspace.go`, the serial overflow guard (two mutants): unreachable before 2⁶⁴ serials.
- `workspace.go`, `<` versus `<=` in the pruning tie-break: credential IDs are unique, so the comparison never sees equal IDs.

# Reviews

- **Design review** (independent reviewer, AI, read-only, security focus):
  - Round 1 FAIL: F1 (Medium), unbounded reads, plus eight Lows.
  - Round 2 PASS: two Lows, both fixed.
- **Change review** (independent reviewer, AI, read-only, security focus): PASS with five Lows.
  - L1, reuse of a pruned ID: fixed with a credential serial checked in `Recheck`.
  - L2, unread secrets kept past the pickup window: fixed by pruning pickups in every service command.
  - L4, test gaps: covered.
  - L3 and L5: recorded as residuals in the plan.

No review is owner acceptance.

# Residuals

These are listed in the plan:
- service credentials cannot authenticate in pre-launch;
- no ownership transfer and no use history;
- an owner leaving does not stop the service user;
- reinstating brings back unrevoked credentials;
- a credential whose secret was never collected stays live;
- the partial-retire timing window;
- the cost of a read at the cap.

# Planning cross-check (HYB-291)

Added by the [HYB-291 planning cross-check](../../reviews/HYB-291-planning-cross-check/report.md) on 2026-11-17. A record added by HYB-291; it changes no result above except where marked. The findings come from a read-only auditor; the HYB-291 writer agent (delegated by root) wrote this trace, the residual notes and the copy changes; the HYB-291 change review is their first independent check.

## Requirement trace

Source: `docs/ui/screens/M04/spec.md:118-128` (HB-268 is the API, HB-269 the screen). Status meanings are in the [cross-check report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). No acceptance test was run.

| ID | Status | Reason | Record |
| --- | --- | --- | --- |
| HB-03 | deferred | Addressed within pre-launch scope for the service-account lifecycle under installation administration; registry-wide and leakage acceptance not built. | HB-268/plan.md |
| HB-05 | not connected | Service credentials cannot authenticate in pre-launch, so no connector can use them. | HB-268/plan.md:159-168 |
| HB-24 | not connected | No ERP connection; credentials cannot authenticate. | HB-268/plan.md:159-168 |
| HB-33 | deferred | Commands use the common path; credentials cannot be used against the API. | HB-268/plan.md:159-168 |
| HB-42 | deferred | Ownership transfer and the owner-departure policy (`retention-access.md:24`) are residuals. | HB-268/plan.md:159-168 |
| AUD-QA-04 | deferred | OAuth service credentials are not built; credentials cannot authenticate. | HB-268/plan.md:159-168 |
| IAM-01–IAM-08 | deferred | The service-user lifecycle is built; authentication and use history are not. | HB-268/plan.md:159-168 |
