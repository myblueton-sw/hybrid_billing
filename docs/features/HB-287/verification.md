---
type: VerificationRecord
title: HB-287 verification of the live P03 recovery screen
description: Type check, unit tests, build, Playwright QA (a render pass and three flow groups) against the real API with the sample seed in EN, KO and AR at 1440 and 390 widths, regressions of P02, P01, O03, O02, O01 and M01, and the outcomes of the design review, render comparison and change review. Run on mac_mini on 2026-11-17 with the real clock.
status: final
---

# Scope

This record verifies [HB-287](https://linear.app/hybrid-billing/issue/HB-287) against its [plan](plan.md). Every check ran on mac_mini (macOS) with Node 26.7.0, Chromium headless shell 1243 through Playwright, the console preview (`HB_GATED=1 vite preview`) and the `hb-api` binary built from main `411a758` with the sample seed. Only synthetic sample data was used. The plan was drafted and the screen and QA written by delegated agents in the ticket worktree; root re-ran every check below on the final build.

# Results

| Check | Result |
| --- | --- |
| `npx tsc --noEmit` | clean |
| `npm test` | 493 pass, 0 fail (new: `service/recovery.test.ts`, `administration/pages/recovery/live-model.test.ts`) |
| `npm run build` | builds |
| Playwright render pass R (every view in EN, KO and AR at 1440 and 390) | 288 pass |
| Playwright group A (the first administrator) | 23 pass |
| Playwright group B (the second administrator) | 12 pass |
| Playwright group C (failure paths, focus, drafts) | 24 pass; the only console errors are the two deliberate aborts |
| P02 regression (four groups) | 162, 32, 10 and 16 pass (run on the build before the last two Low fixes, which touched only P03 files) |
| P01 (qa283full), O03 (qa281), O02 (qa275), O01 (qa273), M01 (qa264) | 46, 71, 77, 53 and 51 pass (same build) |

Each group starts on a fresh API with a new data directory. The open call on the sample primary installation expires one minute after the seed; QA asserts it only when its deadline is at least 5 s ahead of the read's `now` (it was listed in the final run) and its absence only when `now` is at least 5 s past it, using the standby's stop time plus 60 s as the seed deadline once the call is gone.

**What the QA covers**
- Render pass: all five views in three languages at both widths with no horizontal scroll, the right `dir`, the statement, neutral tag borders, dates kept whole with the locale's separator, form edges aligned with a 3px focus-ring inset at 390, the reconciliation tables stacked at 1440, and mono IDs that do not break.
- Group A: as the first administrator, approving the standby unlock (#10) after a lock release confirm that states no decision watermark is checked, locking the standby with "Takes effect now; no second approval" and the generation moving from 2 to 3, proposing an unlock (#11) and the duplicate blocker.
- Group B: as the second administrator, "Your unlock proposal" on #10, withdrawal, and approval of the first administrator's proposal.
- Group C: a two-tab revision conflict, a lost response resent with the frozen snapshot, a stubbed refusal focusing the row note with the kept sentence, a stubbed `internal_error` showing the operation text after a reload, focus back to the opener after Cancel, a draft reason surviving a tab change, and the member denied path.

# Reviews

These reviews were done by AI agents, not the owner.

- **Design review** (independent reviewer, read-only): round 1 CHANGES REQUIRED (four Mediums: restore wording on applied registry records, "all reapplied" when sequences were declared lost, mismatches without chunk indices, resends against the approve narrowing); round 2 PASS, with its Lows folded into the plan.
- **Visual specification and render comparison** (ui-designer, read-only): specification before implementation; round 1 CHANGES REQUIRED (two Highs: overlapping reconciliation tables and an Arabic date broken inside the year; four Mediums: stray rules, the focus ring at 390, IDs breaking at hyphens, stacked confirm headings); round 2 PASS on the seven images reopened. Its two remaining Lows (the date–time separator and the form edge inset) were fixed and checked by the render pass.
- **Change review** (independent reviewer, read-only): PASS with five Lows and four Nits, all applied except an optional isolation nit. The implementer's deviations (the open-call fallback deadline, an `unreadable` propose-unlock blocker, the unknown-result sentence, a scoped override of a shell table-header rule, the restore-sets column widths) were accepted.

No review is owner acceptance.

# Residuals

- Owner FIX of P03; KO and AR terminology review of the AI-drafted copy (writer, fenced, decision watermark, lost sequences, generation — KO currently reads "Generation" — and the KO lock-release and unlock-proposal pair), against `.agents/rules/product-terminology.md`, the product glossary and `docs/ui/design-system/terminology.md`.
- A global `bdi { direction: ltr }` rule in `shared/styles.css` forces free-text reasons inside `<bdi>` to left-to-right in Arabic, on P03 and on P02; a separate fix is needed in the shared styles.
- P03 CSS overrides `.planning-main:has(.p03-live) .planning-tabs` below 768px to a two-column grid, and a shell `th` rule within `.p03-live`.
- Tab links do not keep `installation=`, and switching tabs gives no unsaved-draft warning (the tab bar belongs to the shell; drafts survive in the store).
- Begin, abandon, prune, writer removal, migration and AU/BL read access are not connected; all commands use installation administration.
- The external references were recalled, not opened, so they are unverified evidence.

# Planning cross-check (HYB-291)

Added by the [HYB-291 planning cross-check](../../reviews/HYB-291-planning-cross-check/report.md) on 2026-11-17. A record added by HYB-291; it changes no result above except where marked. The findings come from a read-only auditor; the HYB-291 writer agent (delegated by root) wrote this trace, the residual notes and the copy changes; the HYB-291 change review is their first independent check.

## Requirement trace

Source: `docs/ui/screens/P03/spec.md:120-130` (HB-286 is the API, HB-287 the screen). Status meanings are in the [cross-check report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). No acceptance test was run.

| ID | Status | Reason | Record |
| --- | --- | --- | --- |
| HB-08 | deferred | Restore sets and reconciliation are read; durable evidence storage waits for the database decision. | HB-286/plan.md:222-232 |
| HB-21 | not connected | No billing application ledger is bound. | HB-286/plan.md:222-232 |
| HB-22 | not connected | Opening balances are not bound. | HB-286/plan.md:227 |
| HB-34 | not connected | Writer and verifier paths are not commands. | HB-286/verification.md Residuals |
| HB-37 | deferred | Isolation and lock-release commands are built; begin, abandon, prune, writer removal and migration are not connected. | HB-286/verification.md Residuals |
| HB-42 | deferred | All commands use installation administration, a deviation from P03's five permissions; AU and BL read access is not provided. | HB-286/verification.md Residuals |
| HB-43 | deferred | A lock release checks no decision watermark; tombstone reapplication after restore is not connected. | HB-286/verification.md Residuals |
| AUD-QA-09 | not connected | Opening balances (`modules/balances`) are not bound. | HB-286/plan.md:227 |
| RET-01–RET-08 | not connected | System-level retention tests are NOT_RUN; P03 did not cite them before this section. | this section |
