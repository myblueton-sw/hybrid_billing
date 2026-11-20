---
type: ImplementationPlan
title: Phase 2–5 accessibility verification
description: Rendered keyboard, contrast, reflow and colour-independent checks with corrective implementation and independent visual review.
status: draft
---

# Scope and ownership

HYB-313, accountable owner Seung Woo Park, machine mac_mini. Branch fix/HYB-313-screen-accessibility starts at main 0f9d9a2. Root owns this ticket's QA runner, evidence and narrowly justified accessibility corrections. Independent reviewer owns read-only source/QA review. Claude CLI ui-designer reviews actual renders and leads any material visual revision. One writer per file. Preserve other worktrees and the user's running API/readers/preview.

The HB-257 phase table and HYB-301 UI-09 finding define L01–L03, M01–M05, O01–O03 and P01–P07. Main still has preview-only P06/P07; record their mode explicitly. HYB-328's separate open PR has additional P07 verification but is not merged into this branch. Do not certify unavailable live contracts by testing a preview.

# Verification contract

Use the repository browser runner pattern and installed Playwright Chromium, isolated synthetic sessions and disposable API state. Check EN/KO/AR at desktop, tablet/mobile and 320 CSS-pixel reflow; test 200% enlargement separately and distinguish emulation from actual browser zoom. Run keyboard-only sign-in, filtering, row selection, command review/cancellation and error recovery, with visible focus and no trapped navigation. Do not submit actual financial or owner-review actions. Verify actual status/error announcements and accessible names; do not claim human screen-reader or native-language acceptance.

Measure rendered text/background contrast against the W3C minimum criteria (4.5:1 ordinary, 3:1 large text; exclude inactive/decorative elements), inspect ambiguous backgrounds manually, and inspect meaningful state/action markers with colour-vision emulation and non-colour labels. Record unsupported or untested cases explicitly. Automated measurements and screenshots are evidence, not blanket WCAG certification.

Reference evidence: W3C Understanding [contrast](https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum.html), [reflow](https://www.w3.org/WAI/WCAG22/Understanding/reflow.html), [use of color](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color.html) and [keyboard](https://www.w3.org/WAI/WCAG22/Understanding/keyboard.html), accessed 2026-11-20. Adopt measurable contrast/reflow and keyboard-equivalence criteria. Preserve full-width layouts, neutral borders, IBM Plex and explicit states; do not introduce decorative accent borders or external assets.

# Corrective loop

Maintain a finding ledger with route/state/locale/viewport, evidence, impact, correction and recheck. Review proposed shared/layout fixes before execution, implement only demonstrated defects, rerun affected journeys/renders and obtain independent source plus actual Claude visual re-review. Tests must verify outcomes rather than only selector existence. Reconcile evidence and a scoped PR after checks; merge and owner acceptance remain separate. Continue independent checks if a particular external producer is unavailable.

Independent design review passed for baseline collection. The final matrix must additionally identify subview and state; exercise modal entry/containment/Escape/focus return and isolated nonfinancial approval; include placeholder/control/focus contrast and inherited opacity/background handling; verify live-region updates rather than presence alone. Initial screenshot collection is only baseline discovery and does not satisfy these acceptance checks by itself.

# First reproduced correction

Baseline EN L01 at 320 CSS pixels has page overflow from the language selector. Claude CLI ui-designer inspected EN 320/1440, KO/AR 320 and current CSS, confirmed the composition defect and proposed service-entry-scoped wrapping, compact gaps/padding/type, hiding the redundant language code below 480px, and bounded selector text. Root adds only those rules to `apps/console/src/service/service.css`; shared authenticated headers remain outside this correction. Verify AR arrow bounds explicitly (designer observed possible clipping), dropdown geometry/keyboard/return focus and EN/KO/AR 320/640/1440 renders. This initial finding does not complete the 18-screen matrix.

# M03 reference and expanded rendered review

For the existing grant approval flow, compare [Carbon notification guidance](https://www.carbondesignsystem.com/building-blocks/core/components/notification/guidelines) and [Carbon text input guidance](https://www.carbondesignsystem.com/building-blocks/core/components/text-input/guidelines), accessed 2026-11-20. Adopt the evaluation criteria of contextual inline action feedback, explicit field labels and multi-line input for a reason. Reject copying Carbon's status-colour/border treatment: owner rules require neutral notice borders, explicit status text and transient keyboard focus. No asset, dependency or layout redesign is imported. Existing W3C references remain the accessibility criteria.

Following Claude's actual screenshot review, extend both approval error and applied-result captures to EN/KO/AR at 1440/640/390, record computed borders/focus outlines and include top-of-page viewport captures to distinguish sticky navigation from full-page scroll capture placement. Assert locale fonts before each state capture. The earlier six-render visual result does not cover this expanded matrix until re-reviewed.

# M02 confirmation validation focus correction

Root reproduced an empty-reason submission in the real policy proposal dialog: the error summary appears but focus remains on the submit button. No command was sent. Independent source/design review recommends replacing the manual summary with the established shared Summary using the existing incremented focusKey. Root owns TrustConfirm.tsx; validation, policy values and command guards remain unchanged. Verify first and repeated invalid submissions, error-link focus without navigation, modal keyboard containment and cancellation returning to the opener. Capture EN/KO/AR and obtain re-review.

Root subsequently traced a separate Escape-return failure: the original proposal-review button was disconnected after form remount, the current button differed, and no fallback focus call reached it. The existing fallback has no selector for a new proposal form. Preserve the connected opener first; for proposal confirmations add the current `.m02-form` submit button, with the top-level proposal button as the last fallback when no form remains. Preserve existing action/detail fallback behavior and all command guards. Verify an explicit responsive remount while the dialog is open, both Escape and cancel, summary-focus and reason-focus viewport captures, EN/KO/AR desktop/mobile including 844px mobile height.
