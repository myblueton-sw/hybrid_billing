---
type: ReviewRecord
title: Accessibility findings and corrective verification
description: Evidence ledger for the ongoing phase 2–5 accessibility audit.
status: draft
---

# Current status

Audit remains in progress; no whole-ticket accessibility PASS, owner acceptance or merge is claimed. Initial collection covered 144 default-screen combinations (L01 plus M01–M05, O01–O03 and P01–P07; EN/KO/AR at 1440/640/320). It was exploratory: the reviewer found that absence of aria-busy could precede a loading effect and that fonts.ready alone does not prove font success. The runner now waits for the expected route response and rendered screen plus settling state, and records failed/loaded font faces. The corrected baseline completed 156 captures (adds EN L02 at three widths and EN/KO/AR L03 at all three widths), with zero document overflows and page errors, and expected locale-specific IBM Plex faces loaded. M02 providers and M03 roles are static unavailable guidance and intentionally make no read. Dynamic policy/grant views remain separate journey checks. Initial zero text-contrast candidates do not establish contrast compliance; controls, placeholders, ambiguous compositing and dynamic states require further checks.

# Findings

| Finding | Evidence | Correction | Recheck |
| --- | --- | --- | --- |
| L01 EN header overflows at 320px; AR arrow may be clipped | Actual initial screenshots and document width; Claude observed EN spill and suspected AR clipping | Scoped service-entry header wrapping, tighter gaps/padding/type and redundant language code hidden below 480px; preserve logo and full language name | Nine EN/KO/AR renders at 320/640/1440: no document overflow, arrow/menu bounds inside viewport, keyboard opening/Home/End/Escape and focus return; visible outline, no FontFace failures. Actual Claude final visual review inspected six renders: PASS, no blocker |
| Initial activation test assumed adjacent password fields | Source has a keyboard-accessible show/hide control between fields; first test did not submit matching fields | Test tabs to the actual target field rather than assuming one Tab | Keyboard activation and sign-in completed in subsequent isolated run; this was a test defect, not a product defect |
| Initial M01 readiness selector assumed data-screen-id everywhere | Identity screens use data-security-screen; actual M01 had loaded | Runner recognizes both actual screen markers | Initial 144 captures completed; later response/font safeguards still apply |

# Remaining verification

L02 KO/AR coverage, modal containment and focus return beyond the tested EN M01 flow, approval workflow, error recovery/status updates beyond that flow, 200% enlargement, colour-independent state meaning and control/placeholder/focus contrast remain open. P06/P07 in main are preview-only and cannot certify live behavior. Native language and human assistive-technology acceptance remain unverified. Screenshot counts are not journey coverage.


The first M01 keyboard journey reached a real grant-scope refusal for the seed member's account suspension preview. The seed has no active workspace membership. This invalidates the assumed successful-preview fixture; it is not classified as a product defect. The failed preview did not execute revocation. The corrected fixture redeems the existing member link in a separate context and verifies active member membership before the journey. Passing baseline captures were not repeated for this fixture correction.

Additional entry checks: 12 header renders including 375px, three CSS zoom 200% captures at 640px (not native browser zoom), and twelve Chromium colour-vision emulation captures completed. Root inspected AR enlargement and EN achromatopsia; labels and underlines remain visible with no horizontal overflow. Other emulation captures are generated evidence, not a claim of human review of every image. Full build/typecheck passed with the existing bundle-size warning.


The corrected eligible-member M01 journey passed keyboard search, selection, membership preview, modal navigation, Escape/focus return and discard with no execute request. A bare native showModal comparison reproduced Chromium's temporary unfocused BODY when keyboard navigation enters browser chrome; the runner permits only that unfocused state, rejects any underlying page control, and verifies return into the modal. This is not treated as a product focus-escape defect. The final journey-only rerun passed explicit one-target/zero-exclusion response and rendered counts, member identity, discard live announcement and authenticated membership read-back unchanged. The added read initially omitted CSRF and correctly received 403; the harness now obtains the current session token without logging or storing it. Final result: five recorded checks, no page errors. No revocation execute request was sent.


# Reproduction and limits

Run `apps/console/tools/accessibility-qa.mjs` with explicit local `--url`, isolated `--api`, private disposable `--activation`, existing `--playwright-module`, `--chromium` and generated `--output` paths. The default run captures the baseline then runs the M01 journey; `--journeys 1` runs setup and that journey only. Activation links are consumed, so each run needs a fresh disposable API instance. Never point it at the user's active data directory. Credentials stay in process memory and are omitted from evidence. The runner does not check owner review controls.

Executed environment: macOS arm64, Node 25.9.0, Chromium 159.0.8154.6. Other OS/browser combinations are unverified. Independent reviewer inspected CSS/claims and runner logic; actual browser evidence belongs to root. Claude CLI reviewed the actual specified header renders only. The corrected runtime has not replaced the user's existing local stack.
