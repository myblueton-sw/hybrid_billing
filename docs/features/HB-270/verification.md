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

Added by the [HYB-291 planning cross-check](../../reviews/HYB-291-planning-cross-check/report.md) on 2026-11-17. A record added by HYB-291; it changes no result above except where marked. The findings come from a read-only auditor; the HYB-291 writer agent (delegated by root) wrote this trace, the residual notes and the copy changes; the HYB-291 change review is their first independent check.

## Requirement trace

Source: `docs/ui/screens/M05/spec.md:18` (HB-270 is the API, HB-271 the screen). Status meanings are in the [cross-check report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). No acceptance test was run.

| ID | Status | Reason | Record |
| --- | --- | --- | --- |
| HB-03 | deferred | Addressed within pre-launch scope for own sessions and password change, limited to the signed-in account; registry-wide and leakage acceptance not built. | HB-270/plan.md |
| HB-31 | deferred | Notification preferences are not available. | HB-271/plan.md:109-111 |
| HB-42 | deferred | MFA, recovery, identity linking and profile changes are not available. | HB-271/plan.md:154-160 |
| HB-43 | unclassified – owner review | No retention rule for self-service records was built or recorded. | this section |
| IAM-01 | unclassified – owner review | M05 has no creation or invitation path. | this section |
| IAM-04 | deferred | Federated token checks are not built. | HB-271/plan.md:154-160 |
| IAM-06 | deferred | MFA and recovery are not available. | HB-271/plan.md:109-111 |
| IAM-07 | unclassified – owner review | The scale criterion does not map to a single-account screen. | this section |
| IAM-08 | deferred | A password change does not refresh the maximum-age clock; MFA evidence is not built. | HB-270/verification.md Residuals |
| AUD-QA-08 | deferred | Notifications are not available. | HB-271/plan.md:109-111 |
