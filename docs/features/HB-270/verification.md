---
type: VerificationRecord
title: HB-270 verification of the M05 self-service security API
description: Tests, coverage, race, cross builds, mutation results and review outcomes for the own-account read, the password change and ending one's own sessions. Run on mac_mini on 2026-11-16.
status: final
---

# Scope

This record verifies [HB-270](https://linear.app/hybrid-billing/issue/HB-270) against its [plan](plan.md). Every check was run on mac_mini (macOS) with Go 1.25.7, using the synthetic sample seed.

# Results

| Check | Result |
| --- | --- |
| `gofmt -l` and `go vet` (apps/api/server) | clean |
| `go test -race -count=1 ./server` | pass, 100.0% statement coverage |
| `go test -race -count=40` on the new tests | pass |
| `GOOS=linux GOARCH=386` and `GOOS=windows` builds | pass |
| Mutation testing of `self.go` (95 mutants) | 10 survived in the full run. Seven were then killed by added assertions, each re-run individually: current-first ordering (2), the revision counter at each change (4), and `retry` on the 429 (1). The other 3 are listed below. |

**Equivalent survivors**
- Two mutants change how an empty or invalid end-sessions body is refused. Both bodies still give the same 422.
- One flips the direction of the generation bump. Any change of the generation voids a concurrent sign-in, so either direction works.

**Tests cover:**
- **Admission:** no session, bad CSRF, and a policy age passed.
- **The read:** the shape, administrator versus member, own sessions only, current first, the handle format, no tokens or other people's data, and no-policy behaviour.
- **Password change:**
  - success: other sessions ended, the current one kept, the old password refused and the new one accepted, a log line without secrets;
  - form refusal, and new equal to current;
  - body validation and the size limit;
  - the fifth wrong attempt ending the session;
  - sign-in sharing the lockout;
  - a locked result not counted;
  - attempts given back on the limiter and derive-unavailable paths;
  - stale counts pruned;
  - the generation race, lost access, the session gone, and a refused store changing nothing.
- **Ending sessions:**
  - one, and "others";
  - current, foreign, unknown and empty handles refused alike;
  - an oversized body;
  - unrelated sessions kept;
  - an ended session's pickups dropped.

# Reviews

These reviews are by AI agents, not the owner:
- **Design and change review** (independent reviewer, read-only, security focus).
  - Round 1 (plan only): FAIL. F1 (Medium): the locked result was unhandled and the lockout is shared with sign-in.
  - Round 2 (plan and code): FAIL. R1 (Medium): a limited or unavailable call used an attempt.
  - Round 3 (plan and code): PASS.
  - Rounds 2 and 3 read the implementation and tests, so they also serve as the change review.

No review is owner acceptance.

# Residuals

These are listed in the plan:
- no command-journal audit, only a log line;
- a password change does not refresh the maximum-age clock;
- idle-expired sessions can still be listed until pruned;
- M05 failures share the sign-in lockout;
- MFA, identity linking, profile changes and notification preferences are unavailable.

# Planning cross-check (HYB-291)

Added by the [HYB-291 planning cross-check](../../reviews/HYB-291-planning-cross-check/report.md) on 2026-11-17. A record added by HYB-291; it changes no result above except where marked. The findings come from a read-only auditor; the HYB-291 writer agent (delegated by root) wrote this trace, the residual notes and the copy changes; the HYB-291 change review is their first independent check. Rows marked "(classified by HYB-301)" replace rows that HYB-291 left unclassified; the [HYB-301 trace classification](../../reviews/HYB-301-trace-classification/report.md) gives the evidence.

## Requirement trace

Source: `docs/ui/screens/M05/spec.md:18` (HB-270 is the API, HB-271 the screen). Status meanings are in the [cross-check report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). No acceptance test was run.

| ID | Status | Reason | Record |
| --- | --- | --- | --- |
| HB-03 | deferred | Addressed within pre-launch scope for own sessions and password change, limited to the signed-in account; registry-wide and leakage acceptance not built. | HB-270/plan.md |
| HB-31 | deferred | Notification preferences are not available. | HB-271/plan.md:109-111 |
| HB-42 | deferred | MFA, recovery, identity linking and profile changes are not available. | HB-271/plan.md:154-160 |
| HB-43 | deferred | (classified by HYB-301) Built: the session list keeps sessions only until the 12-hour absolute limit prunes them (HB-270/plan.md:161). Not built: the recent access history (`M05/spec.md:34`; recorded as not built, HB-270/plan.md:125) and its retention period, which is an open value (`M05/spec.md:119`). | [HYB-301 report](../../reviews/HYB-301-trace-classification/report.md) |
| IAM-01 | not applicable | (classified by HYB-301) IAM-01 is first-administrator creation, invitation replay and acceptance in another workspace (`retention-access.md:110`). The M05 action contract (`M05/spec.md:64-73`) acts only on the caller's own account and has no creation, invitation or acceptance action; those are L02 and M01. | [HYB-301 report](../../reviews/HYB-301-trace-classification/report.md) |
| IAM-04 | deferred | Federated token checks are not built. | HB-271/plan.md:154-160 |
| IAM-06 | deferred | MFA and recovery are not available. | HB-271/plan.md:109-111 |
| IAM-07 | deferred | (classified by HYB-301) M05's part is the bulk section (`M05/spec.md:93-95`). Built: reads and session ends are scoped to the caller's own login (`apps/api/server/self.go:74-89`); "end all other sessions" ends them in one call under the installation lock (`self.go:121-130`). Not built: cursor and period filters for many sessions, a fixed target manifest with a session-generation check for "all other sessions", and blocking separated from cleanup progress (`M05/spec.md:72`). The 100,000-user scale is not tested. | [HYB-301 report](../../reviews/HYB-301-trace-classification/report.md) |
| IAM-08 | deferred | A password change does not refresh the maximum-age clock; MFA evidence is not built. | HB-270/verification.md Residuals |
| AUD-QA-08 | deferred | Notifications are not available. | HB-271/plan.md:109-111 |
