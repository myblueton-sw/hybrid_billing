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
