---
type: ReviewRecord
title: Control paint and enlarged-layout verification
description: Provisional CSS contrast observations, rendered color checks and independently identified reflow defects.
status: stable
---

# Final status

Record of a step of HYB-313, which closed on 2026-11-27; the remaining verification is listed in [findings](findings.md) and carried by HYB-399.

# Scope and outcome

HYB-313 remains in progress. The expanded baseline collected 162 normal captures and 54 CSS-zoom captures: all 18 phase 2–5 screens in EN/KO/AR at 1440/640/320, with 200% CSS zoom at 640. There were no document horizontal overflows or page errors, and expected local fonts loaded. These checks did not detect clipping inside individual controls or panels. Actual parallel Claude reviews identified authenticated-strip clipping and P06/P07 preview grid defects. Zero document overflow is not visual acceptance.

M02 adds twelve summary-focus/field-focus captures, and M03 adds eighteen error/applied-result captures with their previously documented real synthetic keyboard journeys. All runs completed. This expands observed states; it does not certify every dynamic subview or all accessibility criteria.

# Contrast measurement boundaries

The optional `--presentation 1` runner flag records control text, actual visible placeholders, field-boundary candidates and existing keyboard-focus observations. It never records input values. Alpha colors use paint-order compositing; group opacity, filters, blending, shadow and unsupported/native painting remain uncertain. Native internals are excluded. Clipping observations are axis-specific geometry candidates, not proof of occlusion. Button boundaries are not automatically treated as required field boundaries.

Independent source review found and corrected alpha-border underlay, focused-element uncertainty, native appearance, clipping-axis, placeholder typography, field-border opacity and nonmodal-exclusion issues. Eight actual Chromium fixtures now pass, including value omission and native modal underlay handling. The reviewer inspected source and the test log, not runtime execution.

The production-screen collections imported the helper before its final two classification corrections. They are not represented as a rerun of the final helper. Their stored observations have no opacity/filter/blend uncertainty, only native appearance; M02 used real native modal dialogs, while those baseline and M03 captures opened no nonmodal dialog. The final focused fixtures cover the corrected cases. A prior exploratory 156-capture collection is not accepted final measurement evidence.

The current observations contain 2,904 baseline, 132 M02 and 285 M03 CSS ratios. None is numerically below its applicable threshold. All baseline ratios and most dynamic ratios remain provisional because their controls retain native appearance. There are no visible-placeholder observations in the baseline; the placeholder fixture tests the measurement method, not product placeholder coverage. Do not report these counts as a contrast PASS.

# Rendered support for specific L01 instances

The sign-in page was separately captured in EN/KO/AR at 1440×1000, device scale 1 and CSS zoom 1, in normal, keyboard-focused and invalid states. No authentication mutation was sent. Existing authored foreground/background/border rules are in `shared/styles.css` and `service/service.css`; no appearance reset or product styling change was made to satisfy the measurement tool.

Read-only PNG analysis samples control interiors and border midpoints away from corners. Exact rendered RGB agrees with the computed foreground for 24 nonempty field/button observations, with opaque backgrounds in 36 observations, and with all four sampled border points in 39 observations. The three keyboard-focused email fields also match all four sampled focus-outline points. These are supporting observations for these instances only. Empty password fields do not establish masking-glyph paint; antialiased edge pixels are not used to derive text contrast. Transparent summaries, mouse-triggered summary focus and native internal controls are not certified by these checks.

Supported colors in this sampled profile yield ink/white 15.4307:1, white/primary fill 11.6256:1, control border/white 3.0663:1 and focus outline/white 7.0105:1. Displayed ratios are rounded here for readability; threshold decisions use unrounded values. This is not acceptance of every control, state, locale/font combination, browser, zoom level or the remaining 2,904 baseline observations.

# Enlarged-layout review findings

All three actual Claude locale reviews found the same shared session-strip and P06/P07 blockers. L01–L03 enlarged renders were visually acceptable. Several long O03/P01/P03–P05 images were downscaled too much for reliable small-text inspection; they were initially unverified and are rechecked with the viewport evidence below. Root confirmed the enlarged KO P06 clipping and narrow grids visually. Claude's corrective design preserves existing content and preview-only status, makes the session strip wrap and gives only P06/P07 intrinsic field/aside grids. Root implemented the scoped correction; actual Claude visual re-review passed for the specified first-tab views in all three locales.

# Scoped reflow implementation and checks

The mobile session strip now wraps without shrinking the sign-out control; workspace and user identifiers retain full DOM text with bounded visual ellipsis. Only P06/P07 opt into intrinsic field and aside grids. Existing layout breakpoints, neutral borders and preview-only semantics remain. The independent reviewer approved the scoped CSS/source; rendered review is tracked separately.

Before the correction, 36 measurements found 18 out-of-viewport session strips, four heading-bound failures, three narrow two-column forms, six half-width P06 result cards and six narrow P07 aside layouts. These categories overlap. After correction, 63 measurements across EN/KO/AR at native widths 320/640/768/1024/1199/1440 and CSS zoom 2 at 640 found zero instances in those categories, zero document overflows and zero page errors. Populated date/time controls were captured separately. This geometry check does not prove every pixel is readable.

Eighteen actual keyboard header cases passed, including synthetic long identifiers at native 320 and enlarged 640: sign-out remained reachable with a visible outline, all identifier/control bounds stayed within the viewport and the row did not overflow. Long text was changed only in the presentation DOM, not in session or authority records. The nested grid expression was accepted by the tested Chromium engine; other browsers remain unverified.

The portable `tools/reflow-qa.mjs` runner captures the default three-screen matrix or, with `--alternate 1`, P06 failures and P07 offline preview views. Its alternate run passed eighteen EN/KO/AR captures at 320/1440 and CSS zoom 2 at 640, asserting document/header/heading bounds, narrow single-column forms and full-width/stacked aside panels. No live entitlement application or demo-engine execution is asserted.

The long-screen review evidence now contains 203 viewport-sized tiles covering O03/P01/P03/P04/P05 in all three locales at 640×1000 with CSS zoom 2. Each route starts at scroll 0, adjacent tiles overlap and the final tile reaches the document end. No document overflow was measured. Actual parallel Claude locale reviews opened the listed tile ranges and found no blocking clipping, overlap or unreadable controls. Their headline image totals are arithmetically inconsistent; the manifest and explicit inspected ranges establish 68 EN, 66 KO and 69 AR tiles (203 total). Root does not adopt the incorrect reported totals. Continuous capture coverage alone is not acceptance.

# Reproduction and evidence

Use the existing baseline, policy and access-approval runners with `--presentation 1`. The baseline additionally accepts `--zoom 1`; this is CSS enlargement, not native browser zoom. Execute the helper fixtures with `node apps/console/tools/presentation-measure.test.mjs`, explicit `--playwright-module` and `--chromium` arguments. Only macOS arm64, Node 25.9.0 and Chromium 159.0.8154.6 were executed.

Local evidence is under `/private/tmp/hyb313-presentation-{baseline,policy,grant-en,grant-ko,grant-ar}`; fixture results are `/private/tmp/hyb313-presentation-final-tests.log`. L01 paint capture/analysis is `/private/tmp/hyb313-paint/{evidence,pixel-analysis}.json`, produced by local `hyb313-paint.mjs` and `hyb313-paint-read.py` helpers in `/private/tmp/hyb336-qa`. Actual locale reviews are `/private/tmp/hyb313-zoom-{en,ko,ar}-review.txt`; the corrective design is `/private/tmp/hyb313-reflow-design.txt`. Generated images stay outside Git. No owner review action, merge or user-runtime replacement is claimed.

Reflow evidence is `/private/tmp/hyb313-reflow-{before,after,alternate}/evidence.json`; keyboard header stress is `/private/tmp/hyb313-header-reflow/evidence.json`; long-screen tiles are `/private/tmp/hyb313-viewport-tiles/evidence.json`. The default before/after geometry was collected by a local helper; the alternate run used the committed-runner candidate.

# Independent rendered review and residuals

The three actual Claude CLI ui-designer reviews inspected P06/P07 first tabs at 320/640/1440 and enlarged 640, populated date-control crops and normal/long header crops. Each locale returned scoped PASS without blockers. Root also inspected the Arabic focused-header crop and Korean enlarged P07 render. The reviewer observed neutral borders, complete identifiers at normal lengths, visible sign-out outline, readable date values and corrected form/card widths. Static renders establish outline appearance; root's separate keyboard journey establishes how focus was reached. No native-language, assistive-technology or owner acceptance is inferred.

The exact long-screen ranges are O03EN/KO/AR00–10; P01EN00–16, KO00–15, AR00–16; P03EN/KO00–12, AR00–13; P04EN/KO00–14, AR00–15; P05EN00–11, KO/AR00–10. All are in the viewport-tile directory. Nonblocking observations remain: P05 metadata wraps identifiers tightly, P03 disclosure text and its reference are visually separated, some two-column P06 input tops differ when labels wrap, and P01 has uneven divider spacing. These are not silently presented as resolved. The existing layout scope is retained; the findings remain available for subsequent screen refinement.

Review outputs are `/private/tmp/hyb313-final-{en,ko,ar}-visual.txt`. The subsequent actual Claude review opened all 18 alternate-tab renders and all 18 P06/P07 first-tab renders at 768/1024/1199; it returned scoped reflow PASS with no blockers. Its output is `/private/tmp/hyb313-reflow-extra-visual.txt`. The exact 1200px transition and Arabic glyph-level evaluation in downscaled images remain unverified. Native browser zoom, additional browsers, human screen-reader checks, full product placeholder/selected-native-control contrast and all dynamic subviews remain unverified. This increment does not complete HYB-313 or the broader product goal.

The portable default runner subsequently passed all 63 live captures in `/private/tmp/hyb313-reflow-final/evidence.json`. Independent source review requested explicit nonempty selector checks: the final runner requires three header elements, the expected P07 aside cards, present panel headings and expected form grids. Its `--evidence <path>` mode rechecks recorded geometry without a browser: the 63 default and 18 alternate captures passed those strengthened checks, while the before-change collection correctly failed the reproduced header-bound check. An initial refactoring error shadowed the viewport record with a child rectangle and falsely failed stored passing data; root corrected it before the successful rechecks. This was a runner defect, not a product regression.

The final guarded runner was also executed live for the eighteen alternate cases in `/private/tmp/hyb313-reflow-guarded/evidence.json`; all passed. The final visual reviewer noted a nonblocking difference between Arabic buttons at native 320 and enlarged 640. Source confirms the existing max 639px `.ops-actions > button { flex: 1 1 auto; }` rule; this increment did not change it. The original design's natural-width preference is therefore not fully realized at the native mobile breakpoint. The review explicitly accepts the reflow with this deviation; no claim of full specification identity is made. Button sizing and uneven label-row alignment remain recorded refinements, rather than additional unreviewed product changes.

Final independent source and technical-writing re-review returned scoped PASS after these guards and provenance distinctions. The reviewer inspected source, documentation and evidence; they did not execute checks.
