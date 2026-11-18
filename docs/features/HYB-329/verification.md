---
type: Verification
title: HYB-329 AR term for installation default in M03 settings
description: The M03 settings source label in Arabic now uses the glossary term الإعداد الافتراضي للتثبيت, matching M02 and P02; a unit test pins the term in every locale and was shown to fail before the change.
status: stable
tags: [HYB-329, M03, terminology, AR]
sources:
  - resource: ../../ui/terminology/product-glossary.md
  - resource: ../HYB-309/verification.md
---

# HYB-329 verification (root, 2026-11-19, main 8a0c8d7)

## Finding

The HYB-309 terminology review found that M03 settings (`request-copy.ts:395`) used القيمة الافتراضية للتثبيت while M02 (`trust-copy.ts:143`), P02 (`live-codes.ts:273`) and the glossary row `installation-default` use الإعداد الافتراضي للتثبيت. The M03 label names a settings source in a Value/Source table, so the setting form fits as well as the value form; the glossary term wins.

## Change

- `apps/console/src/modules/identity/pages/access/request-copy.ts`: `s.source.default` AR → الإعداد الافتراضي للتثبيت. EN and KO unchanged.
- `request-model.test.ts`: new test pinning `settingSource(…, default)` in EN, KO and AR.
- Glossary row `installation-default`: the follow-up note now reads "M03 settings (aligned in HYB-329)".
- Other uses of القيمة الافتراضية in M02 (`live-copy.ts:40`, `:47`) name a numeric default ("the 15-minute default"), a different concept, and are unchanged.

## Checks

- The new test fails with the old string (9 pass, 1 fail) and passes with the change (10 pass).
- `tsc` clean; `npm test` 793/793; build ok.

## Residuals

- KO and AR native-language review pending, as for all AR strings.
