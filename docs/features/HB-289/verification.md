---
type: VerificationRecord
title: HB-289 verification of the live P04 retention screen
description: Type check, unit tests, build, Playwright QA (a render pass and four flow groups) against the real API with the sample seed in EN, KO and AR at 1440, 1280 and 390 widths, regressions of P03, P02, P01, O03, O02, O01 and M01, and the outcomes of the design review, render comparison and change review. Run on mac_mini on 2026-11-17.
status: stable
---

# Scope

This record verifies [HB-289](https://linear.app/hybrid-billing/issue/HB-289) against its [plan](plan.md). Checks ran on mac_mini (macOS) with Node 26.7.0, Chromium headless shell 1243 through Playwright, the console preview (`HB_GATED=1 vite preview`) and the `hb-api` binary built from main `d1ac7d8` with the sample seed. Only synthetic sample data was used. The plan was drafted and the screen and QA written by delegated agents in the ticket worktree; root re-ran every check below on the final build.

# Results

| Check | Result |
| --- | --- |
| `npx tsc --noEmit` | clean |
| `npm test` | 532 pass, 0 fail (new: `service/retention.test.ts`, `administration/pages/retention/live-model.test.ts`; shared fixture `service/retention.sample.ts`) |
| `npm run build` | builds |
| Playwright render pass R (six tabs × EN/KO/AR × 1440/390, plus 1280 for policies and expiry) | 559 pass |
| Group A (the first administrator) | 28 pass |
| Group B (the second administrator) | 20 pass |
| Group C (failure paths, focus, stale key) | 46 pass; the only console errors are the three deliberate aborts |
| Group D (real release flow) | 11 pass |
| P03 regression (render, A, B, C) | 288, 23, 12, 24 pass |
| P02 regression (A, B, C, D) | 162, 32, 10, 16 pass |
| P01 (qa283full), O03 (qa281), O02 (qa275), O01 (qa273), M01 (qa264) | 46, 71, 77, 53, 51 pass |

Each group starts on a fresh API with a new mode-0700 data directory. The measured `.p04-live` width is 1120px at 1440 (container L), 960px at 1280 (M) and 358px at 390 (S).

**What the QA covers**
- **Render pass:** `data-screen-id="P04"` and the live source state on every render. No horizontal scroll or table overflow. Neutral tag borders. Dates kept whole with the locale separator, and IDs unbroken at 1440. One confirm per tab that offers a command, with one primary button and the heading focused. At 390 the focus ring is unclipped, with a 3px inset.
- **Group A:**
  - Places item holds in this order: `sample-source-1` becomes `retention-hold-1`, then `sample-source-5` becomes `retention-hold-2`.
  - Checks the owner selection and the "Choose an owner." error.
  - Proposes a policy change, and approves with the frozen impact.
  - Stops `run/2`. The run then reads finished, not complete, and says that unclaimed items were released, not deleted.
- **Group B:** "Your … proposal" blockers, withdrawal, and approval of the other administrator's proposals.
- **Group C:**
  - A revision conflict, and a lost response resent with the frozen snapshot.
  - A stubbed refusal that focuses the row note, and a stubbed `internal_error`.
  - A stale key on place hold: it keeps the warning, focuses the stale panel and offers only "Place another hold".
  - Text validation, and the member denied path.
- **Group D:** the first administrator withdraws #6 and proposes release #8 on `sample-hold-temporary` with a reason in the body. The second administrator approves, and the hold reads Released.

# Reviews

These reviews were done by AI agents, not the owner.

- **Design review** (independent reviewer, read-only):
  - Round 1: CHANGES REQUIRED, with four Mediums: shared helpers breaking RTL and routing; run copy that was false after a stop; the release flow never exercised; a duplicate hold through the stale path.
  - Round 2: PASS, with its Lows made binding clarifications.
  - The "Finished, not complete" wording was accepted because `done` also covers runs without recorded backup expiry.
- **Visual specification and render comparison** (ui-designer, read-only):
  - The specification was written before implementation.
  - Round 1: CHANGES REQUIRED, with five Mediums: store IDs breaking at hyphens; openers wrapping; misaligned strip numbers; 5,000px stacks at 1280; table links sitting low.
  - Round 2: PASS.
- **Change review** (independent reviewer, read-only):
  - Round 1: CHANGES REQUIRED. The two Mediums were service tests registered twice through a fixture imported from a test file, and a rate-limited resend discarding a possibly applied place-hold attempt.
  - Round 2: PASS.
  - The implementer's deviations were accepted: the Group A hold order, a tab character standing in for line breaks in browser validation, local helper copies, and plural fragments for the partial-hold sentence.

No review is owner acceptance.

# External references

The five references were fetched as text through a summarising tool; no page was viewed as rendered, and no screenshot was inspected. The plan records what each showed and what was adopted or rejected: Google Vault retention and holds, Microsoft Purview retention, Microsoft Purview disposition review, Amazon S3 Object Lock legal hold, and the GOV.UK summary list.

# Residuals

- **Owner decisions:**
  - FIX of P04.
  - KO and AR terminology review of the AI-drafted copy, including the hold terms reused from P03 and the Arabic plural forms. Check against `.agents/rules/product-terminology.md`, the product glossary and `docs/ui/design-system/terminology.md`.
- **Copy for the tech-writer or owner to decide:**
  - the run-state wording "Finished, not complete";
  - the chunk sentence;
  - the visible "This run has finished, so it cannot be stopped." state note.
- **Other-scope and shared rendering is covered by model tests only.** This includes other-scope runs, shared items, null hold counts and other-tenant deletion records. The sample seed has one workspace.
- **Sign-out clears the attempt store.** After signing in again the "may have been applied" warning is gone; the holds list is the place to check.
- **The warning is hidden on a rate-limited resend.** The wait message replaces the pending "may have been applied" sentence. The attempt and its key are kept.
- **Layout leftovers:**
  - At container M the expiry item grid puts Run on its own row, leaving a gap before the action.
  - The native owner select truncates long options at 390.
  - Expiry at 390 is long.
- **Shell overrides:** P04 CSS overrides `.planning-main:has(.p04-live) .planning-tabs` below 768px, and the shell `th` rule within `.p04-live`.
- **The global `bdi { direction: ltr }` rule** in shared styles still affects P02 and P03 reasons. P04 uses `span[dir=auto]` for free text.
- **Not connected:** deletion plan proposals and approval, the deletion worker, retry, backup expiry, archive restore, usage and size figures, and a durable decision record. All commands use installation administration.
