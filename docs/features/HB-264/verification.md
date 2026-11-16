---
type: VerificationRecord
title: HB-264 verification of the live M01 user administration screen
description: Unit tests, browser QA on the production build against a live apps/api with the sample seed, design comparison by the ui-designer role and the change review, run on mac_mini on 2026-11-16.
status: final
---

# Scope

This record verifies [HB-264](https://linear.app/hybrid-billing/issue/HB-264) against its [plan](plan.md).

# Environment

| Item | Value |
| --- | --- |
| Machine | mac_mini, macOS 26 (Darwin 25.2.0) arm64 |
| Node | `npm run build` (tsc and Vite); `npm test` (`tsx --test`) |
| Browser | Playwright with chrome-headless-shell 1243 |
| Console | Vite preview of the production build, port 4174, proxying `/api/v1` |
| API | `apps/api` built from main `95eb7da`, with `--seed sample` and `--console-origin http://127.0.0.1:4174`, using a fresh data directory for each QA run |

# Results

| Check | Result |
| --- | --- |
| `tsc --noEmit`, `npm run build` | pass |
| `npm test` | 341 pass, 0 fail |
| Browser QA (`scratchpad/qa264.mjs`) | 51 PASS, 0 FAIL, no console errors |
| Review shell outside service mode | M01 still renders `data-security-phase="unconnected"` |

## Unit tests

**`src/service/admin.test.ts`** covers:
- the failure-code mapping, including `busy` and `stale`;
- the command envelope, with CSRF and the request ID;
- fresh attempts;
- the read with its cursor;
- rate-limit waits;
- network and unreadable failures;
- activation-link validation on collect;
- the latest-request guard.

**`live-model.test.ts`** covers:
- the offered actions for every state: self, departed, account, membership, sign-in, an outstanding link, a suspended user and a missing login;
- the setup text;
- names and dates;
- distinct failure text and titles.

## Browser QA cases

All cases ran on the live API.

**Users, details and confirmations**
- Live phase; three seeded users; the count text.
- Self is blocked.
- The selected row has `aria-current`.
- Cancel returns focus to the action button that opened the step.
- Confirmation focus; the empty-reason summary.
- Suspend and reinstate membership.
- A reset link for an administrator is hidden, with its reason shown.

**Conflict**
- A second session changes a user, so the first session's submission conflicts.
- The conflict shows the "Could not save this change" title, the list reloads automatically, and the retry succeeds.

**Create user and link pickup**
- Create user, then link pickup.
- The URL has the activation-link form and never appears in the address bar.
- Copy to the clipboard works.
- The link is not in local or session storage.
- The link is discarded on a tab change.
- The created user sets a password with the link and becomes a member.

**Other tabs**
- The revocation tab and "Go to Users".
- The invitations count uses the invitation noun.
- The membership tab.

**Languages and widths** (EN, KO and AR at 1440 and 390)
- `dir` is set correctly, with no horizontal scroll.
- At 390: the detail step moves focus to its heading, Back returns focus to a row button, and the visible field labels render.

**Access**
- A member sees only the permission state, with no buttons.
- The keyboard reaches a row and opens it; the focus outline is visible.

**Computed borders**
- The selected row has no border.
- The panel has only a 1px `--border-row` rule on its inline start.
- The summary and status boxes have a uniform 1px neutral `rgb(141, 148, 156)` border.

Renders are in `scratchpad/hb264qa/`.

# Design and review rounds

**Design input.** The `ui-designer` role (AI, read-only) wrote the specification. Root's departures from it are recorded in the plan.

**Design comparison, round 1: does not match yet** (4 Medium, 6 Low). All Medium items were fixed:

| Finding | Fix |
| --- | --- |
| M1: the detail panel scrolled internally | The panel now grows with the page. |
| M2: non-validation errors were titled "Check the following" | They are titled "Could not save this change". "Check the following" is kept for field validation only. |
| M3: the 403 state still offered actions | It now shows a neutral title and the permission sentence only, with no buttons. |
| M4: the mobile list had no hierarchy | The stacked rows have visible muted labels from `data-label`, a muted sign-in line, and explicit table roles. |

Low items:
- L1: the invitation count noun is fixed.
- L2: the table uses a fixed layout, with the user column at 36%.
- L3: the copy status has spacing.
- L4: the link is discarded on a tab change.
- L5: the row button has a larger hit area.
- L6 (outside this screen) remains: the shell header still says the work service is not connected, the workspace name is truncated at 390, and the EN tab wraps at 390.

**Change review, round 1: FAIL** (3 Medium, 8 Low). All Medium items were fixed:

| Finding | Fix |
| --- | --- |
| M1: Change role was offered when it could only fail | It is no longer offered to a member with an outstanding link, and the role defaults to an allowed value. Unit test added. |
| M2: focus was lost when leaving a step | Focus is now restored by a stable selector after rendering: the row's `data-user-id`, the action's `data-action`, the detail heading, or the Create button. QA checks Cancel at 1440 and Back at 390. |
| M3: Create was stuck after a conflict | Every command failure reloads the list, as the plan says. Create renews its idempotency key when the revision changes, as Confirm does. |

Low items:
- L1: the reason limit is described in bytes, with character estimates for each script.
- L2: `dependency_unavailable`, `idempotency_conflict` and `idempotency_key_expired` now map to their own messages.
- L3: a late success is ignored if the selection changed.
- L4: Invite is hidden for users who are not active, and a set link is hidden when there is no login.
- L5: the conflict copy is neutral ("the user list changed").
- L6: a persistent polite live region announces success.
- L7: see "Not verified" below.
- L8: the duplicate plan heading and the selection wording are fixed.

**Design comparison, round 2: matches the spec.** Two Low copy items were applied:
- the English reason hint is shorter;
- the counter shows its unit ("{n}/256 bytes").

**Change review, round 2: FAIL**, with one new Medium finding.

| Finding | Severity | Resolution |
| --- | --- | --- |
| N1: the error box took focus again on every re-render, so typing in the form broke | Medium | The box is now a component that focuses itself only when a new error arrives, through an effect with a stable ref. QA now types into the reason field after a conflict and checks that focus stays there. |
| N2: a retry after a network failure could become a new request | Low | The list is no longer reloaded after `unavailable`, `limited` or `unauthenticated`, so the attempt is kept. |
| N3: a pending focus target could wait and grab focus later | Low | The focus target is tried once after the render that sets it, then cleared. |
| N4: a repeated success message was not announced again | Low | The live region is cleared, then set again. |
| N5: a link was lost if the selection changed during the request | Low | Accepted: the administrator issues the link again. |
| N6: a `stale` error could repeat | Low | A `stale` error now renews the attempt. |

**Change review, round 3: PASS.**

# Not verified

- **Component tests.** The focus restoration, key renewal and pickup lifecycle are covered only by the browser QA script; there are no `Live.tsx` component tests.
- **Copy.** All copy is a draft that needs the owner's language and terminology review.
- **Owner FIX** of M01 is still open.
- **Shell issues** outside M01 (design L6) remain open.
