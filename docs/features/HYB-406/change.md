---
title: HYB-406 keeping focus while a command is in flight
status: stable
ticket: HYB-406
updated: 2026-11-28
---

# Problem

HYB-399 step H found that while a command was in flight, keyboard focus fell to the page body on M01–M05, O02 and P01. The pressed control is disabled, removed or hidden while busy, and the browser's focus fix-up then moves focus to the body. About 50 controls across 20 files follow this pattern. Changing each one would risk double submission, so one shared keeper handles it.

# Change

`src/shared/focus-keeper.ts`, installed once in `src/main.tsx`:
- **What it remembers.** It records the last focused element and its ancestors. A pointer press clears that record.
- **When it checks.** A mutation observer reacts to changes near that element. It watches added and removed nodes, and the `disabled`, `hidden`, `inert`, `class` and `style` attributes. It checks only while focus is on the body or still on that element. Checks are coalesced: after a frame plus a zero timeout, then again 120 ms later. A change that arrives during that window triggers one more cycle.
- **When it acts.** Only when focus is on the body, and the remembered element is now removed, disabled, inert, not rendered or not visible.
- **Where focus goes.** To the first rendered, visible heading that is not `aria-hidden`, in the nearest container still on the page. The search stops at `main` or a dialog, and the keeper never acts outside them. If the heading has no `tabindex`, it gets `tabindex="-1"`; focus moves with `preventScroll`.

An outcome focus move made by the app still happens afterwards: the keeper only acts while focus is on the body.

# Verification

- `tsc` clean. Console tests: 1,517 total, 1,512 pass, 0 fail, 5 skipped. New: `src/shared/focus-keeper.test.ts`.
- In-flight checks (`tools/other-outcomes-qa.mjs --busy 1`):

| Run | Checks | Before |
| --- | --- | --- |
| Explorer, English 1440 | 291, none failed | 23 failed |
| Four recipes, English 1440 and Arabic 320 | 68, none failed | 6 failed |

  Focus now rests on the form heading, for example "Propose trust change", "Request access" or "Create service account".
- Regression with the keeper, all passing:

| Check | Checks |
| --- | --- |
| Refused explorer | 183 |
| P02–P05 refused | 293 |
| `m02-detail` | 100 |
| `m03-requests` | 90 |
| `m05-sessions` | 91 |
| `organization-changes` | 313 |
| `dialogs` | 31 |
| `opened-states` | 254 |

  `m04-errors` exits 0. An earlier pass also ran `m03-revoke-panel`, `m03-revocations`, `m05-operations`, `organizations` and `members`, all passing.
- Independent change review:
  - First pass: conditions C1 (act only inside main or a dialog) and C2 (coalesce and filter scheduling), plus advisories. All were taken.
  - Re-review: pass, with residuals N1 (a change during the pending window was dropped) and N2 (headings with `visibility: hidden`). Both were fixed after the re-review and verified by the in-flight runs above.

# Limits

- The chosen heading may be off-screen, because of `preventScroll`. Whether that is acceptable is for the owner's screen review.
- iOS Safari does not focus buttons on tap, so the keeper does nothing there. This is unchanged from before.
- P02–P05 have no in-flight measurement yet, because `form-states-qa` has no busy mode.
