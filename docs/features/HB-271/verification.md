---
type: VerificationRecord
title: HB-271 verification of the live M05 my-security screen
description: Unit tests, build, browser QA (password change, lockout paths, sessions) in EN, KO and AR at 1440 and 390, the M01–M04 regression reruns, and the review outcomes for the live M05 screen. Run on mac_mini on 2026-11-16.
status: final
---

# Scope

This record verifies [HB-271](https://linear.app/hybrid-billing/issue/HB-271) against its [plan](plan.md).

Environment:
- mac_mini (macOS)
- Node 26.7.0
- Playwright 1.65.1 with the Chromium headless shell
- a locally built `hb-api` from this branch
- `vite preview` with `HB_GATED=1`
- the synthetic sample seed only

# Results

| Check | Result |
| --- | --- |
| `npx tsc --noEmit` and `npm run build` | pass |
| `npm test` (console) | 366 pass, 0 fail |
| M05 browser QA (script kept outside the repository) | 62/62 pass, no console errors |
| M01–M04 QA reruns on this build | 51/51, 38/38, 38/38 and 49/49 pass. The one console error in M04 is the request that test deliberately aborts. |

The M05 QA covers the following.

**Account facts**
- For an administrator and for a member.

**Password change**
- Autocomplete attributes, including the username field.
- Checks that run locally before any request:
  - a mismatched confirmation;
  - a new password equal to the current one;
  - a new password that breaks the rules.
- A double click sends a single request.
- After a wrong current password, focus moves to the current-password field.
- All fields are empty after every response.
- After success:
  - the other session is signed out;
  - the old password is refused and the new one accepted;
  - no password appears in the URL.
- The fifth wrong attempt signs the person out with the lockout message, and sign-in is then locked.
- An account already locked by earlier sign-in failures shows the locked message.

**Sessions**
- The current session is listed first.
- Ending one session, with focus on the step and a neutral border on the selected row.
- Ending all other sessions.
- Ending a session that has already gone.
- Signing out here.

**Other views and layout**
- The recovery and notifications views.
- EN, KO and AR at 1440 and 390, on the account, methods and sessions views:
  - text direction;
  - no horizontal scroll;
  - at 390, the stepped password form with focus on its heading, and Back returning focus to the opener.

# Reviews

These reviews are by AI agents, not the owner.

**Design review** (independent reviewer, read-only, security focus)
- Round 1: FAIL on M1, the remaining-attempts count and the unexplained lockout.
- Round 2: PASS.

**Design comparison** (`ui-designer`, read-only)
- Round 1: FAIL on three Mediums: the fact sub-line, the AR action-column overflow, and End buttons with identical accessible names. All were fixed, along with three Lows.
- Round 2: PASS. Its one Low, the session title weight, was applied afterwards.

**Change review** (independent reviewer, read-only, security focus)
- Round 1: FAIL on M1: a lost response was reported as "nothing changed". It was fixed with an unconfirmed-result outcome and a test. L1 and L3–L6 were fixed.
- Round 2: PASS, with three informational notes.

No review is owner acceptance.

# Residuals

- No MFA or recovery, no linking of other sign-in methods, no profile changes, and no notification preferences.
- Password changes write a log line, not a command-journal record.
- A session that expired from inactivity can stay listed.
- M05 failures share the sign-in lockout.
- The lockout sign-out message appears only when there was a wrong attempt in the current view.
- If the person leaves the methods view during an in-flight change and returns before it settles, the new form ignores submits until the request settles, while its button looks enabled.
- KO and AR copy is an unreviewed AI draft, including the Arabic plural for hours.
- Screen acceptance (FIX) for M05 is the owner's alone.

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
