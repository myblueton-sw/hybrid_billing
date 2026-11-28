---
type: ReviewRecord
title: HYB-399 — accessibility follow-up of the phase 2–5 screens
description: Results of the checks carried from HYB-313, step by step.
status: draft
---

# Scope and ownership

[HYB-399](https://linear.app/myblueton/issue/HYB-399), machine:mac_mini, root (Claude). Carried from [HYB-313](../HYB-313/findings.md) ("Residual verification"). Order of steps: A focus-outline contrast on opened states; B dialog containment, Escape and focus return; C colour-vision emulation; D post-submission states; E native zoom and a clipping detector.

# A. Focus-outline contrast on states opened on M, O and P01 screens (2026-11-28)

`tools/opened-states-qa.mjs` now focuses every focusable control (inputs, selects, text areas, buttons, links, summaries) inside each dialog, panel, form, section or aside that a button press newly opens on M01–M05, O01–O03 and P01, and measures its outline: at least 2 px and at least 3:1 against the composited background behind it. A control shown only by a box shadow is counted separately, because the shadow's contrast and size are not measured.

- English at 1440: 126 newly opened states, 579 focusable controls, every one with an outline of at least 2 px at 3:1 or more; none relied on a box shadow.
- Arabic at 320: 124 states, 583 controls, the same result.
- The detector was proven on a planted defect: with a 1 px light grey outline injected (`--plant 1`, M03 and O02), it failed 15 of 32 checks, naming the controls and their 1.36:1 ratio.

Limits: hover states are not measured; Chromium only; controls that are not `:focus-visible` after a programmatic focus are skipped (none expected after a keyboard press).

# B. Dialog containment, Escape and focus return (2026-11-28)

`tools/dialogs-qa.mjs` opens each modal dialog of the phase 2–5 screens that can be reached without sending a command, from the keyboard (focus the opener, Enter), with writes blocked, in English and Arabic at 1440 and English at 390. For each it checks that focus moves into the dialog, that Tab and Shift+Tab (the number of its focusable controls plus two) never reach page content outside it — focus may leave to the browser's own controls, which a native modal allows and which the page sees as no focused element — that Escape closes it, and that focus returns to the control that opened it.

- Screen review ("Confirm screen review", shared `ReviewPanel`, measured on M03) and "End other sessions" (M05 `SessionSelection`): 31 checks, none failed.
- Proven on a planted defect: turning each dialog into a non-modal one (`--plant 1`) failed 24 of 31 checks (Tab reached page buttons, Escape did not close).
- Not measured here, and why: M01 "Prepare revocation" opens an inline step, and its modal confirmation follows only after the preparation command (step D); the trust-change confirmation of M02 needs a proposal (covered by the HYB-313 M02 journey); `Reauthenticate` opens only when step-up is required (step D); "Background jobs" and the narrow menu belong to the planning shell, not to the service screens (opener not present in service mode).

# C. Text contrast as seen with colour-vision deficiencies (2026-11-28)

`tools/view-measure.mjs` can now compute the text contrast of every measured text element as seen with protanopia, deuteranopia or tritanopia: the composited text and background colours are converted to linear RGB, transformed by the Machado, Oliveira and Fernandes (2009) full-severity simulation matrix, clamped, and their relative luminance compared against the same thresholds (4.5:1, 3:1 for large text). `tools/views-qa.mjs --vision <kind>` runs it over the 62 views of M01–M05, O01–O03 and P01–P05.

- English at 1440 and Arabic at 320: protanopia 870 checks, deuteranopia 870, tritanopia 870, none failed.
- Achromatopsia keeps only luminance, so its result equals the ordinary contrast measurement, which passes (HYB-313).
- Proven on a planted defect: red text on black (5.25:1 with normal vision, 3.29:1 as seen with protanopia), injected into every view with `--plant 1`, failed on all 62 views with protanopia.
- With the earlier finding that colour appears only on primary buttons and one tag, each with its own words (HYB-313), no state is carried by colour alone in the default views.

Limits: computed from CSS colours, not from rendered pixels; images, SVG, gradients and partly transparent layers are counted as unsure and skipped, as in the ordinary measurement; states that appear only after a submission are step D.

# D. States after the final press on P02–P05 command forms (2026-11-28)

`tools/form-states-qa.mjs --confirm 1 --outcome <kind>` takes each of the 22 kinds of command form on P02–P05 to its confirmation, presses the confirmation's primary button from the keyboard, and answers the command with a routed response instead of the service: `refused` (422 `prerequisite_failed`), `conflict` (409 `revision_conflict`), `unavailable` (503 `dependency_unavailable`) or `unknown` (the request is failed at once with a network timeout error, so the client's own timeout is not exercised). No command reaches the service.

For each form it checks that:
- the press sends exactly one command;
- focus is on a visible element and not the page body;
- the focused element sits in a live region or alert (`role=status`, `role=alert` or `aria-live`) whose text was not on the page before the press;
- that text states the outcome. A refusal must say "Nothing changed"; a conflict must say "confirm again"; unavailable and unknown must say "may have been applied". Refusal and conflict must not use the "may have been applied" wording. The English and Arabic forms of each phrase are both accepted;
- the page does not overflow, cut off or lose text contrast.

A summary check requires every confirmation reached to have been pressed and answered.

| Answer | Language, width | Forms | Focused region | Checks |
| --- | --- | --- | --- | --- |
| refused | English 1440 | 22 of 22 | `role=status` | 293, none failed |
| unknown | English 1440 | 22 of 22 | `role=alert` | 293, none failed |
| conflict | Arabic 320 | 22 of 22 | `role=status` | 293, none failed |
| unavailable | English 320 | 22 of 22 | `role=alert` | 293, none failed |

- Proven on a planted mismatch: answering `unavailable` while expecting `refused` (`--answer unavailable --outcome refused`) failed the outcome-text check on all 22 forms.
- Unavailable and unknown show the same text and role ("may have been applied", `role=alert`): a 503 is treated like a lost response and offered for retry with the same key. The unavailable run therefore does not show a 503-specific statement. Whether that wording is right is a product question, recorded in HYB-400.

Detector corrections made on the way, recorded because they changed what a pass means:
- The first run pressed nothing, because the final button is not a submit button. The summary check was added.
- The first live-region test accepted any text, including the screen-review panel's status line. The test now reads only the focused region, requires it to be new, and requires outcome-specific text (independent method review, conditions taken).
- `input` and `textarea` elements are no longer reported as cut off when their typed value is longer than the box, since they scroll their content. `select` is still measured. The all-views check still passes 1,738 checks.

Not measured here:
- The same states on M, O and P01 forms. They have no shared confirmation step and need a per-form harness.
- The busy state during sending, which lasts only until the routed answer.
- A successful completion, which would need the service to accept a command.
- Screen-reader announcement.

# E. Vertical clipping, text spacing and zoom (2026-11-28)

`tools/view-measure.mjs` now also reports text that is cut off vertically: an element with text whose box hides its vertical overflow, or clamps its lines, while its content is taller than the box. `input`, `textarea` and `select` are excluded, as for horizontal cut-off. `tools/views-qa.mjs` asserts it on every view, and its planted-defect probe now includes a vertically cut box.

`--spacing 1` applies the WCAG 1.4.12 text-spacing override inside `main`: line height 1.5, letter spacing 0.12em, word spacing 0.16em, and paragraph spacing 2em. On each view a check confirms the override is in effect, so the run cannot pass without it.

| Run | Combinations | Checks |
| --- | --- | --- |
| Plain | 62 views in English 1440, Arabic 320 and English 640 | 1,490, none failed |
| With text spacing | the same | 1,676, none failed (including 186 override-in-effect checks) |

In both runs the page was not wider than the viewport, nothing was cut off horizontally or vertically, and text contrast held.

Zoom: browser zoom of 200% on a 1280-wide window lays out the page as a 640-wide viewport, and 400% as 320. Both widths are measured above and in HYB-313. Text-only zoom, where only the font size grows, is not measured, because headless Chromium offers no text-only zoom. The text-spacing run is the closest stress.

# F. States after a refused request on M01–M05, O01–O03 and P01 (2026-11-28)

These screens have no shared confirmation step, so `tools/other-outcomes-qa.mjs` explores them generically. On each view, for each distinct enabled button in `main`, it reloads the view and presses the button from the keyboard. It skips pager, filter and navigation buttons, and sign-out. It then repeats up to six rounds of: fill the empty visible fields (text, number, date, email, password, the first option of a select, the first radio of a group, and checkboxes inside a form, but not in lists); then press the newly shown submit or primary button, or another new button that is not cancel, close, reload or go-to. It stops when a request is sent. Every non-read request is answered with a routed 422 `prerequisite_failed`, so nothing reaches the service.

For each request reached, it checks that:
- only the final press sent requests;
- focus is on a visible element inside the page, not the body;
- the refusal is stated in a live region or alert that is new since the press. The region must either hold focus, or be the nearest new one above the focused title. Example: a dialog or detail heading that takes focus while the alert sits below it;
- the refusal does not say the request "may have been applied";
- the page does not overflow, cut off (horizontally or vertically) or lose text contrast.

| Run | Requests reached | Checks | Buttons that sent no request |
| --- | --- | --- | --- |
| English 1440 | 36 | 183, none failed | 157 |
| Arabic 320 | 41 | 208, none failed | 152 |

Reached in English:
- M01: create user and suspend user;
- M03: request revocation;
- M05: change password and end other sessions;
- O02: relationship proposal;
- O03: issuance setup reads opened from the contract rows;
- P01: plan proposal, approval and withdrawal.

Most buttons that sent no request open details, reload, search, page, confirm the screen review or go to another screen. None of these sends a command.

Not reached, because each needs valid domain input the explorer cannot produce: M02 "Propose trust change", M03 "Request access", M04 "Create service account" and O01 "New view group draft". They are moved to HYB-401 for per-form harnesses.

Method notes:
- In the development server, React runs effects twice, so a dialog that prepares on open may send the same request twice. Requests are counted once per distinct path.
- O03 sends two reads when a contract is opened, so up to three distinct requests per press are accepted.
- Only the refused answer is run here. Conflict, unavailable and unknown on these forms are left to HYB-401 and the per-screen checks.

# G. Other answers on M, O and P01, and native zoom (2026-11-28)

`tools/other-outcomes-qa.mjs --answer conflict|unavailable|unknown` reruns the step F exploration with a 409 `revision_conflict`, a 503 `dependency_unavailable`, or a request failed at once with a network timeout error.

| Answer | Language, width | Requests reached | Checks |
| --- | --- | --- | --- |
| conflict | Arabic 320 | 41 | 208, none failed |
| unavailable | English 1440 | 36 | 147, none failed |
| unknown | English 320 | 45 | 183, none failed |

In every case focus stayed on a visible element in the page and a new alert or status stated the answer.

A wording finding was recorded from the stated texts. When the response is lost (`unknown`), these screens state that nothing was applied:
- M01 create user and suspend user, and M03 request revocation: "Could not save this change. The user service is not responding."
- M01 bulk add: "Nothing has been revoked."

P01, O02 and O03 state the uncertainty ("may have been applied", "resend the same request"). Because the M01 and M03 statement may be false, it is moved to HYB-402.

Native zoom: `tools/views-qa.mjs --scale 2` lays the page out at device pixel ratio 2, as native browser zoom does, rather than with CSS zoom. Each view checks that the ratio is in effect. 640 CSS pixels at ratio 2 is a 1280-pixel window at 200%; 320 is the same window at 400%. All 62 views at English 640, Arabic 640 and English 320 passed: 1,676 checks, none failed.

# H. While a command is in flight (2026-11-28)

`tools/other-outcomes-qa.mjs --busy 1` holds each routed answer until the page has been measured, then lets it through.

While the request is in flight, it checks that:
- focus is on a visible element, not the page body;
- the page says it is working, through `aria-busy` or a new status;
- nothing overflows, is cut off or loses text contrast.

| Run | Requests measured | Checks |
| --- | --- | --- |
| Explorer, English 1440 | 36 | 291, 23 failed |
| Four recipes, English 1440 and Arabic 320 | 8 | 68, 6 failed |

What held:
- Every request announced that it was working.
- No layout failed.

The failures are all one kind: focus fell to the page body while the request was in flight.

| Where | Requests |
| --- | --- |
| M01 | 7 |
| M02 trust proposal | 2 |
| M03 | 2 (explorer) and 2 (recipe) |
| M04 account creation | 2 |
| M05 password | 1 |
| O02 relationship proposal | 10 |
| P01 plan proposal, approval and withdrawal | 3 |

O01 and O03 kept focus.

The pressed control is disabled while busy, and a focused control that becomes disabled drops focus. Focus returns to the outcome when the answer arrives (steps F and G, HYB-404), so the loss lasts only for the wait. The fix is moved to HYB-406.

Not measured: P02–P05, because `form-states-qa` has no busy mode yet (recorded in HYB-406).
