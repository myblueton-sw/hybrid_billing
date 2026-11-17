---
type: VerificationRecord
title: HB-283 verification of the live P01 installation screen
description: Type check, unit tests, build, Playwright QA against the real API with the sample seed in EN, KO and AR at 1440 and 390 widths, regressions of O03, O02, O01 and M01, and the outcomes of the design review, render comparison and change review. Run on mac_mini on 2026-11-17 with the real clock.
status: final
---

# Scope

This record verifies [HB-283](https://linear.app/hybrid-billing/issue/HB-283) against its [plan](plan.md). Every check ran on mac_mini (macOS) with Node 26.7.0, Chromium headless shell 1243 through Playwright, the console preview (`HB_GATED=1 vite preview`) and the `hb-api` binary built from main `748d0c2` with `--seed sample`. Only synthetic sample data was used. The implementation was written by a delegated writer agent in the ticket worktree; root re-ran the checks below on a fresh API.

# Results

| Check | Result |
| --- | --- |
| `npx tsc --noEmit` | clean |
| `npm test` | 428 pass, 0 fail (new: `service/installation.test.ts`, `administration/pages/installation/live-model.test.ts`) |
| `npm run build` | builds |
| Playwright QA of P01 (46 checks) | 46 pass |
| O03 regression (qa281) | 71 pass |
| O02 regression (qa275) | 77 pass |
| O01 regression (qa273) | 53 pass |
| M01 regression (qa264) | 51 pass |
| Console errors during QA | none |

**What the QA covers**
- As the second sample administrator: the plan acceptance row reads "You proposed this"; the pending destination is approved through the confirm step and closes with the approver; diagnostics then lists two destinations.
- As the first sample administrator: plan 1.0.0 is approved; the accepted-plan line ("Plan 1.0.0 accepted by … Not installed: runs are not connected in this build.") appears and the previewed version states why (newest release not installed); "Propose this plan" for 1.1.0 goes through a confirm step that says approval installs nothing; the own pending proposal hides Propose and reads "You proposed this"; withdrawing it offers Propose again.
- The TLS report reads "Untrusted certificate issuer" with "Keep certificate verification on"; no copy suggests turning verification off.
- Plan, diagnostics, support and access views in KO and AR at 1440 and KO, AR and EN at 390 with no horizontal scroll, `dir` per locale and neutral tag borders.

# Reviews

These reviews were done by AI agents, not the owner.

- **Design review** (independent reviewer, read-only): round 1 CHANGES REQUIRED (eight Mediums: preview shape, propose gating for a stale own plan, diagnostics wording, refusals without a blocker, command denial, host and remount rules, the post-acceptance view, external references); round 2 PASS with its Low items applied.
- **Visual specification and render comparison** (ui-designer, read-only): specification before implementation; round 1 CHANGES REQUIRED (one High: the destination host and port order in Arabic; six Mediums on row height, Port alignment, rule width, column widths, tag wrapping and the mobile first cell); round 2 PASS. Its remaining Low (a short rule above the row note) was fixed afterwards and the QA re-run; the bold mono first cell at 390 renders at 16px but regular weight because no semibold mono face is loaded, which is accepted.
- **Change review** (independent reviewer, read-only): round 1 CHANGES REQUIRED (one Medium: an unavailable preview read "Conflicts: None"; Lows on the Arabic endpoint, upgrade wording when installed and the reason message); round 2 PASS.

No review is owner acceptance.

# Residuals

- Owner FIX of P01; KO and AR terminology review (including the Korean particle spacing on the accepted-plan line and the Arabic "Unknown" inside an LTR endpoint).
- The screen derives the preview basis, the accepted-version check and duplicate detection from the newest 50 proposals the API returns.
- Several behaviours (command denial and reload, the kept refusal sentence, the operations host branch, RTL output) are covered by model tests and browser QA, not by component render tests.
- Runs, install execution, key and release management, diagnostic requests, support bundles and support access are not connected.
- The accepted-plan line has no date and preview volume items have no data flag, because the API does not provide them.
- The external references were recalled, not opened.
