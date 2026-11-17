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
