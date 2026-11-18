---
type: Verification
title: HYB-312 P04 copies outside the controlled scope
description: The P04 deletion storage results now state that copies already sent to ERP or downloaded are outside the controlled scope, as the P04 spec requires; verified by tests, the P04 browser regression and a copy and render review.
status: stable
tags: [HYB-312, P04, retention, HB-27]
sources:
  - resource: ../HB-288/plan.md
---

# HYB-312 verification (root, 2026-11-18, main bc0d03e)

## Requirement

- HB-27 and P04 spec (local planning input `docs/ui/screens/P04/spec.md`) line 41, storage results: "외부 ERP/다운로드 사본 통제 밖 표시"; line ~115 `policy_complete`: "통제 밖 사본은 별도로 표시합니다".
- Recorded missing by the HYB-301 trace classification.

## Change

- `live-copy.ts` key `outsideNote`, using the glossary term outside-controlled-scope (glossary lines 82, 86):
  - EN "Copies already sent to ERP or downloaded are outside the controlled scope: deleting here does not delete them."
  - KO "ERP로 이미 보낸 사본과 다운로드한 사본은 통제 범위 밖입니다. 여기서 삭제해도 그 사본은 삭제되지 않습니다."
  - AR "النسخ المرسلة مسبقًا إلى ERP أو التي نُزِّلت خارج النطاق المتحكَّم فيه: الحذف هنا لا يحذفها."
  - KO and AR are drafts pending native review.
- `LiveDeletion.tsx` renders it under each run's per-store results, after the sum and backup note, in the default text colour.
- `policy_complete` cannot be reached in this build (decision record not connected); when it becomes reachable, its status must keep this note beside the completion wording.

## Checks

- `tsc` clean; `npm test` 748/748 (new test asserts placement and the three strings); build ok.
- P04 browser regression `qa289fullon297` on this build: R 559/0, A 28/0, B 20/0, C 46/0, D 11/0.
- Copy and render review (ui-designer role): FAIL on terminology (it first used "retention control" / "보존 통제", which collides with 보존 = hold on this screen) → the glossary term was adopted; placement and rendering PASS (1440 visible at 14px default colour; 390 wraps, RTL right-aligned).

## Residuals

- Spec line 41 also asks to separate AI source-text non-collection from Assistant conversation retention; not on the Deletion tab today — ticket HYB-325.
- 390 glyph-level clipping was checked only on downscaled full-page captures.
