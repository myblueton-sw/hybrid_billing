---
type: Verification
title: HYB-325 P04 AI source-body and Assistant conversation statement
description: The P04 deletion storage results now state that AI usage records hold no prompt or response bodies, that AI usage intake and the Assistant are not connected, and that Assistant conversations are a separate data class whose deletion runs delete only conversations; verified by tests, the P04 browser regression and cropped renders.
status: stable
tags: [HYB-325, P04, retention, HB-27]
sources:
  - resource: plan.md
  - resource: ../HYB-312/verification.md
---

# HYB-325 verification (implementer, 2026-11-18, branch feat/HYB-325-p04-ai-assistant-statement on main 4471cfd)

## Change

- `live-copy.ts` key `aiNote` (EN/KO/AR as in [plan](plan.md#statement)); KO and AR are AI drafts pending native review.
- `LiveDeletion.tsx` `Stores` renders it as `<p className="p04-small">` directly after `outsideNote`.
- `live-model.test.ts` asserts placement and key phrases in the three languages.
- Glossary rows `ai-source-body` and `assistant-conversations`.

## Checks (observed)

- `tsc --noEmit`: clean.
- `npm test`: 749 tests, 749 pass, 0 fail (HYB-312 recorded 748; +1 new test).
- `npm run build`: built.
- `go build ./apps/api/cmd/hb-api`: built.
- P04 browser regression `qa289fullon297` on this build: R 559/0, A 28/0, B 20/0, C 46/0, D 11/0.
- Cropped render probe (390 and 1440, EN/KO/AR, runs 1 and 2): the note is present once per run, directly after the outside-scope note, 14px, same colour as that note (rgb(86, 97, 108)), `dir` rtl for AR, no horizontal page scroll in any of the 12 cases. Height at 390: EN 157px, KO 134px, AR 151px; at 1440: EN 112px, KO 90px, AR 101px. Crops inspected visually for EN 390, KO 1440 and AR 390: wrapped without clipping, AR right-aligned.

## Not done

- Copy, terminology and render review (ui-designer role, 2026-11-18): FAIL on one overclaim ("only token counts, costs and identifiers" — the event also holds time, outcome, usage, cache relation and cost grade/origin/document) and length; root adopted the reviewer's shorter strings and the glossary fix; placement and render PASS. After the fix: console 749/749, build ok, P04 render group R 559/0.
- Not committed.

## Residuals

- Pseudonymous consumer references are not enforced; the note does not claim them.
- Rejection of source input carrying unexpected bodies is not implemented (no AI usage intake).
- Attachments and tool-run references are not a modelled retention class.
- Parser staging, error logs and support bundles are not verified against the same classification.
- KO and AR native-language review pending.
