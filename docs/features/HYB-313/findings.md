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

Further scoped evidence is recorded in [Control paint and enlarged-layout verification](presentation-verification.md), [User-list retry focus and refresh announcements](read-recovery.md) and [Administration read-recovery accessibility](admin-read-recovery.md). These records extend the historical collections above; they do not turn the baseline into whole-ticket acceptance.

| Additional finding | Correction | Recheck |
| --- | --- | --- |
| M01 unavailable-read Retry removes its focused control and successful recovery has no live completion message | Focus existing stable Reload before Retry starts; announce only the latest successful user-triggered read using established localized copy | Six EN/KO/AR desktop/mobile keyboard cases, repeated failure, new announcement transitions, later user focus movement and separate response-order/filter-supersession checks pass; scoped independent source and actual visual review recorded in read-recovery.md |
| P01–P05 Retry disappears on successful read, leaving BODY focused and no read-completion status | Transfer focus to existing reload control and publish a dedicated localized status only after latest/current-scope success | Thirty locale/width journeys, five ordering/terminal cases and five actual actor-replacement cases pass; scoped source/TW and sixty-render Claude review recorded in admin-read-recovery.md |

# Remaining verification

Modal containment and focus return beyond the tested M01/M02 flows, approval/error recovery outside the tested M02 policy and M03 grant flows, 200% enlargement, colour-independent state meaning and control/placeholder/focus contrast remain open. P06/P07 in main are preview-only and cannot certify live behavior. Native language and human assistive-technology acceptance remain unverified. Screenshot counts are not journey coverage.


The first M01 keyboard journey reached a real grant-scope refusal for the seed member's account suspension preview. The seed has no active workspace membership. This invalidates the assumed successful-preview fixture; it is not classified as a product defect. The failed preview did not execute revocation. The corrected fixture redeems the existing member link in a separate context and verifies active member membership before the journey. Passing baseline captures were not repeated for this fixture correction.

Additional entry checks: 12 header renders including 375px, three CSS zoom 200% captures at 640px (not native browser zoom), and twelve Chromium colour-vision emulation captures completed. Root inspected AR enlargement and EN achromatopsia; labels and underlines remain visible with no horizontal overflow. Other emulation captures are generated evidence, not a claim of human review of every image. Full build/typecheck passed with the existing bundle-size warning.


The corrected eligible-member M01 journey passed keyboard search, selection, membership preview, modal navigation, Escape/focus return and discard with no execute request. A bare native showModal comparison reproduced Chromium's temporary unfocused BODY when keyboard navigation enters browser chrome; the runner permits only that unfocused state, rejects any underlying page control, and verifies return into the modal. This is not treated as a product focus-escape defect. The final journey-only rerun passed explicit one-target/zero-exclusion response and rendered counts, member identity, discard live announcement and authenticated membership read-back unchanged. The added read initially omitted CSRF and correctly received 403; the harness now obtains the current session token without logging or storing it. Final result: five recorded checks, no page errors. No revocation execute request was sent.


# Additional L02 locale and enlargement checks

A separate valid-link capture completed EN/KO/AR at 1440, 640 and 320px plus CSS zoom 200% at 640px: 12 renders, no document overflow or page errors, expected locale fonts loaded and no direct-text contrast candidates. Root visually inspected the AR enlarged render. This is CSS zoom emulation, not native browser zoom or a complete contrast audit. The initial helper left CSS zoom active before the next same-document navigation; explicit reset fixed that harness state leak. No product correction was required. The local helper is `/private/tmp/hyb336-qa/hyb313-entry.mjs`, reusing the main runner's capture routine; generated evidence is `/private/tmp/hyb313-entry-final/evidence.json`.

# M03 isolated approval journey

The final grant approval runs passed EN at 1440px and KO/AR at 390px. A synthetic member accepted the seeded invitation, the first administrator requested read access, and a separate administrator approved it. Fixture setup uses API and locator actions; only the subsequent approval interaction is keyboard evidence. The proposer was not eligible to approve its own request.

Each run verified opening/cancelling with originating-row focus restored, required-reason validation with error-summary focus, its link returning focus to the invalid field without navigation, and successful approval. The scoped status region changed from empty to the exact locale-specific success message. The refreshed screen displayed the effective grant and the ended request's approved status and second administrator; authenticated API read-back confirmed the applied request and active effective grant. No page errors or document overflow were recorded. The first runs checked locale fonts before the approved screenshot only. The expanded capture routine now checks loaded locale fonts and failures before both error and approved captures.

Earlier harness runs incorrectly assumed the applied request's state string, seeded reviewer ID and uniqueness of the ended-details selector. Source inspection corrected those assumptions; they are harness defects, not product defects. Final evidence explicitly reports passed for all three runs. No M03 product code changed. The independent reviewer inspected runner source and recorded evidence, found no remaining behavioral blocker, and did not execute the browser tests or visually review screenshots. Claude inspected those six actual renders and found no visual blocker, then required broader locale/width coverage and computed-border evidence before any broader compliance claim. The expanded review and subsequent corrective re-review are recorded below.

# Reproduction and limits

Run `apps/console/tools/accessibility-qa.mjs` with explicit local `--url`, isolated `--api`, private disposable `--activation`, existing `--playwright-module`, `--chromium` and generated `--output` paths. The default run captures the baseline then runs the M01 journey; `--journeys 1` runs setup and that journey only. Activation links are consumed, so each run needs a fresh disposable API instance. Never point it at the user's active data directory. Credentials stay in process memory and are omitted from evidence. The runner does not check owner review controls.

Executed environment: macOS arm64, Node 25.9.0, Chromium 159.0.8154.6. Other OS/browser combinations are unverified. Independent reviewer inspected CSS/claims and runner logic; actual browser evidence belongs to root. Claude CLI reviewed the specified header, M03 and M02 renders with the distinct results recorded above. The corrected runtime has not replaced the user's existing local stack.

The M03 runner is `apps/console/tools/access-approval-qa.mjs`, with the same explicit environment arguments plus `--locale` and `--width`. Each run requires a separate fresh disposable API because seed activation links are consumed. Initial local evidence is in `/private/tmp/hyb313-approval-en`, `/private/tmp/hyb313-approval-ko` and `/private/tmp/hyb313-approval-ar`; these generated artifacts are not Git deliverables. Synthetic grant approval changes only those disposable instances. Human screen-reader announcements and native-language approval are unverified.

Expanded M03 runs in `/private/tmp/hyb313-matrix-{en,ko,ar}` passed 18 state/width combinations (two states, three locales, 1440/640/390px), with six additional desktop viewport-top screenshots. No document overflow, page errors or font failures occurred. This supersedes the first runs for font-checked error captures. The independent reviewer inspected runner source and JSON evidence, not screenshots or runtime execution. A follow-up corrects the computed-style selectors to include the actual visible `.service-status`, `.live-panel` and active `[aria-current="page"]` elements; the follow-up in `/private/tmp/hyb313-border-{en,ko,ar}` passed all 18 combinations. Root inspected computed values: error/success boxes use the neutral line colour, selected tabs use a dark underline, and the focused textarea/summary uses the transient blue outline. Claude subsequently identified the capture blocker described below; computed values alone did not establish visual acceptance. Computed style collection alone is not contrast compliance.

The main runner's explicit outcome change was exercised by a fresh isolated M01 journey: five checks passed with `outcome: passed` in `/private/tmp/hyb313-journey-outcome/evidence.json`.

The first expanded Claude review found one evidence blocker: EN 390 approval-error had repeated page segments and did not show the reason field/buttons. Root confirmed the broken image. The runner now waits for the actual responsive data-wide state after resizing, restores the error state if its form remounts, waits for font readiness, scrolls to the top and waits two animation frames before capture. The corrected `/private/tmp/hyb313-settled-{en,ko,ar}` runs passed all 18 combinations. Root inspected the corrected EN 390 image, which shows one continuous page with the error, focused reason field and actions. Actual Claude corrective re-review passed after inspecting five specified corrected renders, clearing the repeated-page and KO640 trailing-blank concerns. It did not re-inspect all 18 new renders; the prior 18-render scope and limitations remain explicit. Earlier screenshot failures are not product-design defects or accepted visual evidence.

# M02 confirmation validation and return focus

Root reproduced two separate defects. Empty-reason submission displayed an error but retained submit-button focus (`/private/tmp/hyb313-policy-en/evidence.json`). A later trace proved the original opener disconnected after form remount; closing the modal left BODY focused because the fallback could not find the current proposal-review button (`/private/tmp/hyb313-policy-trace-matrix/evidence.json`). The exact remount trigger in that initial capture is uncertain. Explicit responsive replacement subsequently reproduced the disconnected opener.

Following independent design and source review, TrustConfirm uses shared Summary with the existing focusKey and resolves the current proposal form submit button when its saved opener is disconnected. Connected openers retain priority; existing action/detail fallback behavior and all command/authorization guards remain unchanged. The top-level proposal button is the last proposal fallback when no form remains.

Final repository-runner evidence in `/private/tmp/hyb313-policy-review/evidence.json` passes all six EN/KO/AR journeys at 1440×1000 and 390×844: initial and repeated invalid submission focus, keyboard error link without URL change, 24 modal Tab/Shift+Tab steps, explicit responsive opener replacement before both Escape and cancel, and final return to the proposal action. No policy commands or page errors occurred. Twelve viewport-only summary-focus/field-focus captures passed loaded-font and horizontal-bound checks. Root inspected the AR mobile summary and the actual computed neutral borders and transient focus outlines. Full build/typecheck and 24 focused trust model/render tests passed; the existing bundle warning remains. Actual Claude final visual re-review inspected all twelve viewport captures and computed styles: scoped PASS, no blocker. This resolved the earlier missing summary-focus, AR mobile and modal-overlay evidence. Prior full-page captures were insufficient to establish modal viewport fit. The top-level locale/width in the JSON describes fixture setup; per-check/per-render fields identify the actual matrix. The recorded viewport heights come from the runner and screenshots, not JSON height fields.

The harness now awaits the actual policy response, responsive mode and scheduled initial focus, verifies the intended minute input, and uses a fresh page for each case. Earlier input-reset and same-document-response assumptions were harness failures, not additional product corrections. The separate detached-opener failure was retained and fixed rather than dismissed with those harness failures. Native Chromium browser-chrome focus behavior is handled with the same bounded exception as M01, requiring in-dialog focus before continuing.

Reproduce with `apps/console/tools/policy-accessibility-qa.mjs` and explicit `--url`, disposable `--api`, private `--activation`, `--playwright-module`, `--chromium` and `--output`. Setup uses synthetic administrator activation/sign-in by locator; only the policy journey is keyboard evidence. The runner chooses the platform select-all key, but only macOS Chromium was executed. Ordinary connected-opener cancel also passed in the preceding `/private/tmp/hyb313-policy-return-final` matrix. Form-removed fallback and reload=true handling remain unverified by browser tests; human assistive-technology and native-language acceptance are not claimed.

# Integration with main on 2026-11-25

The five corrective commits of this branch were merged with main `eb614b48` (520 commits later). Git reported one conflict, in `docs/ui/terminology/product-glossary.md`, where both sides added rows; both sets are kept. No source file conflicted.

Rerun on the merged tree: TypeScript check clean; console tests 1426, 1423 passed, 3 skipped. Repository runners against a disposable local service with the sample seed: `admin-read-recovery-qa.mjs` passed, 30 cases; `read-recovery-qa.mjs` passed, 6; `reflow-qa.mjs` passed, 63; `admin-read-order-qa.mjs` passed in both modes, 5 and 5; `policy-accessibility-qa.mjs` passed, 6; `accessibility-qa.mjs` passed, 5 checks and 162 captures; `access-approval-qa.mjs` passed, 3 checks at English 1440.

One runner expectation was stale and is corrected: on main a 401 answer to a screen's first read now returns the person to sign-in ("Sign in again."), where the runner waited for an in-screen error button. `admin-read-order-qa.mjs` now waits for the sign-in form. The product behaviour is main's and is not changed here.

Limits of this rerun: the last three runners were run against built files (`vite build`, `vite preview`), because the development server of a worktree whose dependencies are linked from another checkout refuses the font files and those runners assert that no font fails; with the development server they failed on that assertion alone. Captures were not visually reviewed again; the earlier visual reviews apply to the unmerged revisions. The remaining audit scope stated under Current status is unchanged, and this is not whole-ticket accessibility acceptance.


# Command forms on P02–P05 lose keyboard focus when opened (2026-11-25)

Reproduced with `apps/console/tools/command-focus-qa.mjs` on main `d25f1720`: on P02, P03, P04 and P05 every command form opened from the keyboard (44 opener kinds across the views, English, 1440 and 390) left `document.activeElement` on BODY. Opening removes the button that was pressed and nothing took focus, so a screen reader is told nothing about the form that appeared. Chromium kept its navigation position, so Tab still reached the form; Cancel already returned focus to the opening control in all 44 cases.

Correction: when a command form shows its editing step and focus is on BODY, focus moves to the form's title (`tabIndex=-1`), as the confirmation step already does with its own title. The untitled subscription form of P02 focuses its wrapper, which is a named group. Files: the four `LiveCommand.tsx` of `integrations`, `recovery`, `retention` and `release-evidence`. Returning from the confirmation to the editing step follows the same rule.

Recheck on the corrected tree: 130 checks pass — for each of the 44 openers at both widths, focus after opening is on the form title (or the named group), Tab reaches Cancel, Cancel returns focus to the opener; no view takes focus into a form on load; no command was sent. TypeScript clean; console tests 1426, 1423 passed, 3 skipped. Three existing form runners still pass (`replacement-search-qa.mjs` 55, `owner-search-qa.mjs` 82, `hold-card-qa.mjs` 10).

Limits: English only, since the rule does not depend on language; Chromium only; no screen reader was used, so what is announced is inferred from the focused element; forms on P01 and outside administration (M and O screens) do not use this opener pattern and are not covered by this runner; the confirmation-to-editing return was not exercised by the runner.

Follow-up HYB-367 (2026-11-25): the form title and the confirmation title of P02–P05 take the width of their text (`inline-size: fit-content`, at most the form's width), so the focus ring no longer spans the column. The global focus rule already keeps the ring 3px from the text. `command-focus-qa.mjs` now also measures the focused title: 172 checks pass. The confirmation title's ring was not captured; it has the same rule.


# Buttons on M, O and P01 screens that leave focus on the page body (2026-11-25)

`apps/console/tools/press-focus-qa.mjs` signs in to a disposable local service, blocks every request that is not a read, and on each view of M01–M05, O01–O03 and P01 presses up to ten differently named buttons with Enter, one fresh page load per button, recording where focus is afterwards. On main `59b01919` three kinds of control left `document.activeElement` on BODY:

| Finding | Correction |
| --- | --- |
| "Go to …" buttons in unavailable-view panels (M02 providers, M03 roles and support, M04 credentials, usage and transfer, M05 recovery) switch the view, which removes the button | `App.tsx`: after a view change, if focus is on BODY, focus the main region, as a screen change already does |
| M05 methods, Cancel of the password form: the form is remounted, which removes the button | Focus returns to the form's first field |
| M05 sessions, "Check again" of the ended-sessions list: the button disabled itself while reading | The button stays focusable and is marked `aria-disabled` while reading, with the look of a disabled button; a press while reading does nothing |

Recheck on the corrected tree: the whole run gives 170 cases with one failure, the "Check again" button, which was then corrected and rechecked with the M05 run alone (18 cases, none failing). TypeScript clean; console tests 1426, 1423 passed, 3 skipped.

Limits: English, 1440 wide, Chromium; at most ten buttons per view, chosen by distinct name in document order, so later buttons of long views and buttons inside forms opened by another button are not reached; because writes are blocked, a button that sends a request is observed only up to the refused request; "Sign out" leaves the page and is recorded as such; no screen reader was used. The first two runs are not evidence: one stopped at M05, and one pressed "Sign out" for real before writes were blocked, so the screens after it were read signed out.

# Every view at 320 and 1440: overflow, clipping and text contrast (2026-11-25)

The baseline measured only the first view of each screen. `apps/console/tools/views-qa.mjs` loads every listed view of M01–M05, O01–O03 and P01–P05 (62 views) in English, Korean and Arabic at 320 CSS pixels and in English at 1440, 248 loads, against a disposable local service with the sample seed, on main `73634526`. For each it first requires that the view finished loading (no busy region within eight seconds), that the session is still signed in and that the main region shows no alert or error panel; a view failing this fails the run. It then measures, inside the main region:

- whether the document is wider than the viewport;
- any visible element reaching past the viewport that has no ancestor with a scrolling or hiding horizontal overflow;
- any box that hides its horizontal overflow, has more content than room and does not use an ellipsis;
- for each element with its own text, the contrast of its colour against the background composed from its ancestors, against 4.5:1, or 3:1 for text of 24px or of 18.66px bold; disabled controls are left out, and an element under a background image or an opacity other than 1 is counted as not measurable;
- the control measurements of `presentation-measure.mjs`.

Before the views, the runner plants a grey text on white, a 900px box and a 100px box with longer text in a real view and requires each detector to report its probe; all three were reported.

Result: 1,738 checks pass, none fail. All 248 loads finished loading, signed in, without an error panel (28 to several hundred words in the main region, median 171; the shortest are views that say the service has nothing to offer there, such as O01 history and M05 notifications). No document overflow, no element past the viewport, no cut-off box, no text below its threshold among 23,535 measured texts (at least 14 per view, median 61; none unmeasurable), and no control candidate. Two earlier runs of this runner also reported nothing and are not counted as evidence: the first had no planted probes or count of measured texts, the second did not yet require that each view had finished loading.

Limits: the views as they load with the sample seed, with no form open, no dialog, no error state and no hover or focus state, so the contrast of those states and of placeholders is not covered here; CSS-pixel width, not browser zoom; Chromium; computed colours, not painted pixels, so text over images or partly transparent layers would be reported as unmeasurable rather than judged (none occurred); P06 and P07 are not included. Not detected by this runner: content clipped vertically (a fixed height or line clamping); text marked `aria-hidden` or `inert`, which is skipped although it is visible; text in pseudo-elements, the values of inputs, selects and textareas, and SVG text; a text colour not expressed as `rgb()`/`rgba()`, which would be skipped without being counted (the console's CSS has none); a background painted by an element that is not an ancestor; a hidden-overflow box that is the main region itself or above it. A view that draws more after a later request than the one awaited could be measured early; the settling check does not rule that out. Nor does it recognise a permission or unavailable panel that carries neither `role="alert"` nor `.live-error`, or a screen that never marks itself busy; the word counts make both unlikely in this run but do not exclude them. Screenshots were not taken because nothing failed, and no visual review was made.
