---
type: VerificationRecord
title: HB-275 verification of the live O02 relationship change screen
description: Type checks, unit tests, build, browser QA in EN, KO and AR at 1440 and 390, regressions and review outcomes for the O02 screen bound to the HB-274 change API. Run on mac_mini on 2026-11-17 with synthetic sample data only.
status: final
---

# Scope

This record verifies [HB-275](https://linear.app/hybrid-billing/issue/HB-275) against its [plan](plan.md). Every check was run on mac_mini (macOS) with Node 26, Playwright 1.56.1 and headless Chromium. The API binary was built from this branch, with the sample seed.

# Results

| Check | Result |
| --- | --- |
| `npx tsc --noEmit` | clean |
| `npm test` | 382 pass, 0 fail. New: 9 model tests and the `plans()` client test |
| `npm run build` | OK |
| O02 browser QA (`qa275`) | 77 of 77 pass, with no page errors |
| O01 regression (`qa273`) | 53 of 53 |
| M01 regression (`qa264`) | 51 of 51 |

**The O02 QA covers:**
- **Propose.**
  - Client validation and the error summary with focus.
  - A past date refused on the field, and the error clearing on edit.
  - The current relationship from the listing.
  - A sealed proposal opening the impact view with focus on the plan title and an announcement.
  - "Not computed" three times, with "This is not zero".
  - The dated change lead and the step indicator.
- **Refusals.**
  - A member refused by the server, on its field.
  - A rejected no-change proposal, with its violation and the "not kept" note.
- **Approval and results.**
  - The own-proposal block, with no button.
  - Activation by a second administrator, with confirm focus and the reason required.
  - The result view with the announcement.
  - The O01 link, and O01 showing the move on the effective date.
  - A stale plan blocked after another activation, then discarded.
- **390 flows.** Propose, a rejected proposal reachable, continue to approval, and activate to the result detail.
- **Access.** A member is denied.
- **Locale and layout checks.**
  - The `dir` attribute in every view in EN, KO and AR at 1440 and 390.
  - No horizontal scroll at 390.
  - Back focus at 390.
  - Neutral computed borders on `.o02-tag` and the selected `.o02-plan`.

# Reviews

These reviews were done by AI agents, not the owner.

- **Design review** (independent reviewer, read-only):
  - Round 1: FAIL, with five Mediums: attempts, before and after, closed plans, choices, and refusals.
  - Round 2: PASS, with six Lows, all applied.
- **Visual specification and render comparison** (ui-designer, read-only):
  - Round 1: FAIL, with H1 (results table clipped in the list rail), three Mediums and five Lows.
  - Round 2: PASS, with four Lows. Three were applied: the double rule, the gap before Impact, and the inline Results link. The fourth was a hover state in a capture.
- **Change review** (independent reviewer, read-only):
  - Round 1: FAIL, with M-1: below 1280px, a view change lost the plan detail.
  - Round 2: PASS. N-1 (Show open plans keeps the stepped state) is fixed, and N-2 is recorded in the plan.

No review is owner acceptance. The FIX for O02 remains the owner's.

# Not verified

- No capture covers confirm, the own-proposal block, errors or rejected in KO and AR, or the impact and approval views with a plan in KO and AR at 390.
- The external references in the plan were recalled, not opened.
- No screen reader was used; announcements were checked through the live-region text only.

# Residuals

- The FIX for O02.
- KO and AR terminology:
  - 활성화 versus 적용, 폐기, and the tab labels 승인·예약 and 적용 결과, which promise scheduling;
  - the AR sentence "تعيين {target} لـ{member}".
- Money, publication and ERP impact are not computed.
- Two-person approval is between accounts.
- There is no scheduling or resume.
- Admin-only.
- A pruned plan answers activate with the whole-screen unavailable block, because the command layer maps NotFound to `permission_denied`.
- Shell items: "Work service not connected", the review strip, and the sidebar offset on long pages.
