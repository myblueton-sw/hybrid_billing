---
type: VerificationRecord
title: HYB-293 verification of step-up reauthentication for mutating commands
description: Checks, browser QA, mutation testing, regression runs and review outcomes for the step-up rule, the holder-only password confirmation endpoint and the console password confirmation dialog, run on mac_mini on 2026-11-17 and 2026-11-18.
status: final
tags: [HYB-293, security, identity, step-up, verification]
sources:
  - resource: plan.md
  - resource: ../../reviews/HYB-291-planning-cross-check/report.md
---

# Scope

This record verifies [HYB-293](https://linear.app/hybrid-billing/issue/HYB-293) against its [plan](plan.md). A delegated agent wrote the plan and the code in the ticket worktree. Root re-ran the checks marked "root" on the final code. All runs used synthetic data, fresh API processes and new mode-0700 data directories that were deleted afterwards. Go commands used a `GOCACHE` in the session scratchpad.

# Results

| Check | Result | Run by |
| --- | --- | --- |
| `gofmt -l`, `go vet ./...` | nothing listed; clean | root |
| `go test ./...` | all packages ok, except one run in which `TestTimingIndistinguishable` (sign-in timing, existing test) failed while a race run was loading the machine; the test then passed 8 of 8 on both main and this branch at a load average of about 10 | root |
| Coverage | `apps/api/server` 100.0%, `modules/passwords` 100.0% | root |
| Console | `tsc` clean; `npm test` 555 pass, 0 fail; `npm run build` ok | root (554 at first check), implementer (final) |
| `go test -race -count=3 -timeout 90m ./apps/api/server ./modules/passwords` | ok (876 s and 246 s), 0 `DATA RACE` lines | root, final Go code |
| Cross builds | linux/amd64, windows/amd64, darwin/arm64, linux/386 | implementer |
| Mutation testing (`stepup.go`, `reauth.go`, `session.go`, and the changed lines of `commands.go`, `self.go`, `policy.go`, `signin.go`, `boundary.go`, `modules/passwords/registry.go`; run with `-short`) | 116 mutants, 112 killed. Of the four survivors, three are now killed by added tests, each shown by applying the single mutant (the test fails) and restoring the file byte-identical: `stepup.go:64` (`TestCandidateRuleOnlyAppliesToApprove`), `reauth.go:48` (`TestReauthenticationLogOutcome`) and `registry.go:133` (`TestVerifyHolderUnknownKeyWithTheDummyPassword`). Root had first judged `registry.go:133` equivalent; the implementer showed it is not (an unknown key with the dummy password matched), so the guard is load-bearing. `stepup.go:68` is equivalent: an error yields a zero view whose kind is not `revision`. | root (full run), implementer (single mutants) |

No Go test file contains the substring "flo" + "at" (spelled apart here on purpose). No changed file exceeds 1,500 lines.

# Browser QA

The QA was run by the implementer on the gated preview with the `qa293` runner. Final run per group:
- **A**, a real wait over 900 s: 21 pass, 0 fail. It covers the M01 suspend (a wrong password, then the right one, with a byte-identical resend), the recovery lock, and a fresh session that shows no dialog.
- **B**, the first refusal stubbed and then a live resend: 32 pass, 0 fail. Nine command families: M02, M03, M04, M05, O02, O03, installation, integrations and retention.
- **C**, cancel, Escape, a pending dialog, keyboard-only use, the fifth wrong password and neutral copy: 27 pass, 0 fail.
- **D**, EN, KO and AR at 1440 and 390: 54 pass, 0 fail. It covers direction, copy, the hint and its `aria-describedby`, and no horizontal scroll. Arabic renders Latin digits both in literals and in `Intl` durations.

The password appeared only in the request bodies for password confirmation, sign-in and link redemption. It never appeared in URLs, the console or browser storage.

# Regression

Earlier screen scripts were rerun on this branch:
- **M01:** qa264 51/0.
- **M02:** qa265r 38/0. This is a copy of qa265 whose expectations were changed only where this plan changed M02: the step-up labels, and the removed stale sign-out panel.
- **M03:** qa267 38/0.
- **M04:** qa269 49/0 in two of three runs.
- **M05:** qa271 62/0.
- **O01:** qa273 53/0.
- **O02:** qa275 77/0.
- **O03:** qa281 71/0.
- **P01:** qa283full 46/0.
- **P02:** A/B/C/D 162/32/10/16, all without failures.
- **P03:** R/A/B/C 288/23/12/24, all without failures.
- **P04:** R/A/B/C/D 559/28/20/46/11, all without failures.

The qa269 held-retry step (after a lost response, "Try again" sometimes sends no request) failed in 3 of 3 runs on main `3e36101` and in 1 of 3 on this branch. It predates this change and is ticketed as HYB-316. Its cause is not established.

# Reviews

These reviews were done by AI agents, not the owner.

- **Design review** (security focus): round 1 CHANGES REQUIRED, round 2 clarifications, round 3 PASS. See the plan's review history.
- **Change review** (security focus):
  - Round 1 PASS, with three Lows:
    - the shared copy claimed the password was not confirmed after a confirmed resend;
    - "signed out after wrong passwords" named a cause the client cannot know;
    - one assertion could never fail, and a memoization test was missing.
  - Round 2 PASS, after the fixes. One Nit (N4) was inferred: `reauthRefused` assumes the client is inside `ServiceContext`. Root checked this: `ServiceContext.Provider` exists only in `Service.tsx:429`, and live screens render inside it.
- **Terminology review:**
  - Round 1 CHANGES REQUIRED, with one High: `reauthChanged` claimed the password changed "in this session", but the counter is per login and also moves on reset-link redemption. It also asked for one user-facing term and the five-attempt hint.
  - Round 2 CHANGES REQUIRED: a KO timestamp label, the AR wording "saved", and the "consecutive" count.
  - Round 3 PASS.
  - Root decided the user-facing term is "password confirmation" (KO 비밀번호 재확인, AR تأكيد كلمة المرور). Reauthentication and step-up remain internal aliases.

No review is owner acceptance. KO and AR copy is an unreviewed AI draft.

# External references

Root rechecked the plan's references on 2026-11-18 using WebFetch summaries:
- **GitHub sudo mode:** matches the plan (two hours, reset by each sensitive action).
- **OWASP:** the plan's row was corrected. Token rotation is in the "Re-authentication After Risk Events" section, not in the sensitive-features section.

# Planning cross-check

The requirement trace is in the plan under "Requirement trace", with statuses as defined by HYB-291.

| Requirement | Status | What this ticket provides or lacks |
| --- | --- | --- |
| HYB-291 finding H2 | fixed for password sessions | Step-up on every registered mutating command and on M05 end sessions. |
| IAM-08 | covered for password sessions; MFA not connected | |
| HB-42 | covered for the session; MFA not connected | |
| M01 suspend with reauthentication | covered | |
| M05 password confirmation before sensitive changes; end other sessions | covered | Link, unlink and MFA management are not connected. |
| IAM-05, IAM-06, QA-04, HB-03 | covered as regressions | |
| M02 policy configuration of the sensitive-command rule | deferred | Owner question O-4. |

No acceptance test was run; planning acceptance stays NOT_RUN.

# Residuals

The plan's Residuals section stands. In short:
- A stolen session gets 5 holder-only guesses.
- Sign-in and password change can still be blocked by third parties.
- There is no cookie or CSRF rotation (owner question Q1).
- Refusals and confirmations are logged but not journaled (Q2, HYB-292).
- There is no M02 setting (O-4).
- The identity module's own clock is unreconciled.
- There is no MFA, SSO or linking.
- The M05 preview copy still says "reauthentication".
- Not done: axe, and the dialog over another modal (no command screen opens one).
