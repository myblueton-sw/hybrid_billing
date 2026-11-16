---
type: VerificationRecord
title: HB-269 verification of the live M04 service-account screen
description: Unit tests, build, browser QA (including failure paths for the one-time secret and the held issue retry) in EN, KO and AR at 1440 and 390, the M01–M03 regression reruns, and the review outcomes for the live M04 screen. Run on mac_mini on 2026-11-16.
status: final
---

# Scope

This record verifies [HB-269](https://linear.app/hybrid-billing/issue/HB-269) against its [plan](plan.md).

**Environment**
- mac_mini (macOS), Node 26.7.0, Playwright 1.65.1 with the Chromium headless shell.
- A locally built `hb-api` from this branch.
- `vite preview` with `HB_GATED=1`.
- The synthetic sample seed only; no real customer data was used.

# Results

| Check | Result |
| --- | --- |
| `npx tsc --noEmit` | pass |
| `npm test` (console) | 361 pass, 0 fail |
| `npm run build` | pass |
| M04 browser QA (script kept outside the repository) | 49 of 49 checks pass. The only console error is the request the test aborts on purpose. |
| M01, M02 and M03 QA reruns on this build | 51/51, 38/38 and 38/38 pass |

**What the M04 QA covers**
- Create: an empty name, then a valid name; the ID preview; the detail focus; a taken name.
- Issue: the end date; the Reveal with the secret format; the heading focus.
- Where the secret must not appear:
  - in live regions;
  - in the URL;
  - in the DOM after Done.
- No Copy button.
- The acknowledgement is required before Done.
- "I didn't store it: revoke": the reason is prefilled, and the selected row border is neutral.
- The ended disclosure.
- The unstored reminder after leaving the step, and Dismiss.
- Suspend: the "On hold" label and the issue block.
- Reinstate (with its warning) and retire, then the retired disclosure.
- The usage and transfer views; the member denied view.
- **Failure paths for the one-time secret and the held issue retry:**
  1. The issue response is lost (fetched, then aborted). "Try again" appears; the retry replays the same credential, with exactly one added.
  2. `pagehide` drops the secret and keeps the unstored mark.
  3. A pickup 503 gives the retry state, then the secret is shown and the heading focused.
  4. A pickup 403 gives the gone state with Revoke.
  5. A mismatched pickup credential is not shown and is absent from the DOM.
- EN, KO and AR at 1440 and 390:
  - text direction;
  - no horizontal scroll;
  - the stepped create form with heading focus, and "Back" restoring focus;
  - the Reveal at 390 without horizontal scroll.

# Reviews

These reviews are by AI agents, not the owner:
- **Design review** (independent reviewer, read-only, security focus): FAIL in rounds 1 and 2, PASS in round 3. The fixes are listed in the plan.
- **Design comparison** (`ui-designer`, read-only):
  - Round 1: FAIL. One High (the `svc-` prefix wrapping) and five Mediums (Arabic date bidi, ID isolation in copy, secret line breaks, missing manual-copy instruction, literal placeholders). H1 and the other Mediums are fixed. The manual-copy instruction was added and then removed again: the change review found that it instructs a clipboard copy, which is the owner's decision under the residual below. The hint stays neutral.
  - Round 2: FAIL. One Medium remained: Arabic dates were garbled in the Valid cell. It was fixed with plain "From" and "Until" lines.
  - Round 3: PASS, with Low and owner-routed residuals. Two were addressed afterwards: a lead line in the reminder box, and the service account shown beside each credential.
- **Change review** (independent reviewer, read-only, security focus):
  - Round 1: FAIL. One Medium: the held retry and the Reveal failure states had no tests. It was fixed with `keepsHold` and its unit test, plus the five browser cases above. Lows L1–L9 were fixed. Round 2 found N1 (the copy instruction, now removed) and N2 (a shared CSS rule that changed M03 sub-line weight, now scoped to M04).
  - Round 3: PASS. Its two informational notes were applied afterwards: an `<h3>` for the reminder heading, and the reminder row button spacing.

After those last changes, everything was rerun: `npm test` (361), the M04 QA (49/49) and the M01–M03 QA (51/38/38).

No review is owner acceptance.

# Residuals

- **One-time reveal and clipboard:** the owner decides whether a one-time raw reveal, a manual copy and a Copy button are allowed. The M04 design record bans clipboard exposure.
- **Terminology:** the KO tab label "자격 메타데이터" stays, while the body uses 자격 증명. KO and AR copy is an unreviewed AI draft. On 390, the KO tab row wraps in the shared tab component.
- **Unstored reminders:** they are held in page memory, so a reload loses them; a `beforeunload` prompt warns first.
- **Held attempt:** it is lost if the issue form remounts after an unknown outcome.
- **Retired account:** the "Revoke required" state needs a partial retire, so it is unit-tested, not rendered.
- **Pre-launch limits:** credentials cannot authenticate, and there is no use history and no owner transfer.
- **Owner acceptance:** screen acceptance (FIX) for M04 is the owner's alone.
