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
