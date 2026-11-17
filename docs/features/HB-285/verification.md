---
type: VerificationRecord
title: HB-285 verification of the live P02 integrations screen
description: Type check, unit tests, build, Playwright QA in four groups against the real API with the sample seed in EN, KO and AR at 1440 and 390 widths, regressions of P01, O03, O02, O01 and M01, and the outcomes of the design review, render comparison and change review. Run on mac_mini on 2026-11-17 with the real clock.
status: final
---

# Scope

This record verifies [HB-285](https://linear.app/hybrid-billing/issue/HB-285) against its [plan](plan.md). Every check ran on mac_mini (macOS) with Node 26.7.0, Chromium headless shell 1243 through Playwright, the console preview (`HB_GATED=1 vite preview`) and the `hb-api` binary built from main `e47fd6f` with the sample seed. Only synthetic sample data was used. The plan was drafted and the screen and QA written by delegated agents in the ticket worktree; root re-ran every check below on the final build.

# Results

| Check | Result |
| --- | --- |
| `npx tsc --noEmit` | clean |
| `npm test` | 455 pass, 0 fail (new: `service/integrations.test.ts`, `administration/pages/integrations/live-model.test.ts`) |
| `npm run build` | builds |
| Playwright QA group A (renders and the second administrator) | 162 pass |
| Playwright QA group B (the first administrator's flows) | 32 pass |
| Playwright QA group C (revoke) | 10 pass |
| Playwright QA group D (conflicts, lost responses, client checks, URL state, member) | 16 pass; the only console errors are the two deliberate aborts |
| P01 regression (qa283full) | 46 pass |
| O03 regression (qa281) | 71 pass |
| O02 regression (qa275) | 77 pass |
| O01 regression (qa273) | 53 pass |
| M01 regression (qa264) | 51 pass |

Each QA group starts on a fresh API with a new data directory, so its starting state does not depend on order.

**What the QA covers**
- Group A: all four views in EN, KO and AR at 1440 and 390 with no horizontal scroll, the right `dir`, the statement and leads, neutral tag borders, choice rows inline with inputs no wider than 21px, form and confirm buttons full width at 390, and no rule above the subscriber fact; as the second administrator, approving the endpoint proposal (#9), the payments subscription reading "You proposed this", and approving the redelivery of `delivery-3` (a new queued delivery named from the notice, "Nothing is sent"); an open proposal has exactly one primary button.
- Group B: as the first administrator, approving the upgrade (#8), stop and disable with "Takes effect now; no second approval", proposing enable, proposing and withdrawing a rollback, approving the payments subscription, pause and resume across both administrators, the dead-letter request rules, proposing a subscription (inactive destinations listed but not selectable); the second administrator's identical choice reads "already covers…" once after Review with no confirm; an open form hides its opener and siblings and has an action heading; Cancel restores them and focus.
- Group C: revoking `sample-erp-invoices` states that queued redeliveries will not be sent; `delivery-5` reads cancelled with "Outcome unknown"; as the second administrator the revoked dead letter can still be dismissed after a request, and the dismissal removes it.
- Group D: a two-tab revision conflict keeps the reason and confirms again; a lost response on `subscription_pause` shows the pending-attempt sentence and the resend returns the stored result; a lost response on `letter_approve_redeliver` resends to `denied` and reads "Your earlier approval may have been applied. The dead letter is gone." with the deliveries hint; reason length and whitespace checks; invalid URL selections are ignored; a member sees the unavailable block.

# Reviews

These reviews were done by AI agents, not the owner.

- **Design review** (independent reviewer, read-only): round 1 CHANGES REQUIRED (four Mediums: resending a letter approval answered by denial, two QA flows that could not pass, the external references gate); round 2 CHANGES REQUIRED (two QA steps that could not pass against the real API); round 3 PASS, with its nit applied.
- **Visual specification and render comparison** (ui-designer, read-only): specification before implementation; round 1 CHANGES REQUIRED (two Highs: broken choice rows and mid-word breaks in tables; six Mediums); round 2 CHANGES REQUIRED (four Mediums: a rule above the subscriber fact, two primaries during confirm, split IDs in the verification column, form buttons at 390); round 3 PASS on the five images reopened. The other renders were checked by the QA assertions listed above, not reopened by the designer.
- **Change review** (independent reviewer, read-only): round 1 CHANGES REQUIRED (two Mediums: a missing installations list read as empty, an empty revision still offering commands; six Lows); round 2 PASS. The implementer's deviations (an extra proposals file, the kept-refusal and vanished-proposal wording, a dismiss button on the vanished notice, an "untrusted" propose blocker, fail-closed handling of unknown values) were accepted.

No review is owner acceptance.

# Residuals

- Owner FIX of P02; KO and AR terminology review, including the proposed terms in the plan (delivery, dead letter, plugin publisher, tab labels) and the AI-drafted Arabic copy.
- All integrations commands use installation administration (the recorded P02 permission deviation from HB-284).
- Dispatch, sending, signing, token issuance, egress checks, key management, marks, filters, search, row drawers, before/after comparison and cursor resync are not connected.
- The screen sees at most 50 proposals and 50 deliveries per subscription; duplicates and availability are derived from those.
- KO text was checked for layout only; the 768–1023 px range was not checked visually.
- The external references were recalled, not opened.

# Planning cross-check (HYB-291)

Added by the [HYB-291 planning cross-check](../../reviews/HYB-291-planning-cross-check/report.md) on 2026-11-17. A record added by HYB-291; it changes no result above except where marked. The findings come from a read-only auditor; the HYB-291 writer agent (delegated by root) wrote this trace, the residual notes and the copy changes; the HYB-291 change review is their first independent check. Rows marked "(classified by HYB-301)" replace rows that HYB-291 left unclassified; the [HYB-301 trace classification](../../reviews/HYB-301-trace-classification/report.md) gives the evidence.

## Requirement trace

Source: `docs/ui/screens/P02/spec.md:120-146` (HB-284 is the API, HB-285 the screen). Status meanings are in the [cross-check report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). No acceptance test was run.

| ID | Status | Reason | Record |
| --- | --- | --- | --- |
| HB-03 | deferred | Addressed within pre-launch scope for installation-administration commands, a recorded deviation from P02's permission split; registry-wide and leakage acceptance not built. | HB-284/plan.md:363-386 |
| HB-05 | not connected | No source connectors. | HB-284/plan.md:363-386 |
| HB-06 | not connected | Dispatch and receipt are not commands. | HB-284/plan.md:363-386 |
| HB-08 | not connected | No durable evidence storage. | HB-284/plan.md:363-386 |
| HB-10 | not connected | HawkEye results are not connected. | HB-284/plan.md:363-386 |
| HB-11 | not connected | TokenMeter results are not connected. | HB-284/plan.md:363-386 |
| HB-24 | not connected | No ERP connection. | HB-284/plan.md:363-386 |
| HB-31 | deferred | (classified by HYB-301) P02's part is the subscriptions and channels tab, with API, module and notification-channel permissions not merged (`P02/spec.md:33, 80`). Built: webhook subscriptions with approval, pause, resume and revoke, and delivery states (HB-284/plan.md:80-95). Not built: the separate API, module and channel permissions, an internal recorded deviation where every command uses `AdministerInstallation` (HB-284/plan.md:308-310, 376); notification channels and events, which need external messaging (out of pre-launch scope). | [HYB-301 report](../../reviews/HYB-301-trace-classification/report.md) |
| HB-33 | deferred | Covered: plugin and subscription commands use the common command path. Not built: CLI, AI and external consumer parity, 202 jobs for bulk work, and an API-only business cycle. | HB-284/plan.md |
| HB-34 | deferred | Deliveries are read; dispatch, confirm, redirect and report are not commands. | HB-284/verification.md Residuals |
| HB-35 | deferred | Plugin proposals are built; publisher, key, version registration and marks are not commands. | HB-284/verification.md Residuals |
| HB-37 | deferred | (classified by HYB-301) Built: each plugin version's storage contract (namespace, owner, delete on removal or keep for evidence for N days, migration kind) is shown, and install and upgrade are refused when it changes (HB-284/plan.md:47-55; HB-285/plan.md:306; `P02/spec.md:51`). Not built: holding webhook dispatch after a restore until external results are reconciled. The P03 recovery register has a writer lock (HB-286/plan.md:38, 49), but no apps/api integration file refers to it, and dispatch is not a command in pre-launch. | [HYB-301 report](../../reviews/HYB-301-trace-classification/report.md) |
| HB-40 | deferred | (classified by HYB-301) Built: plugin versions are signed by a registered publisher key, and a revoked publisher, a compromised key or a vulnerable version blocks install (HB-284/plan.md:47). Not built: the licence check that `P02/spec.md:50, 74` requires in the module manifest and before activation; `modules/plugins` has no licence field. | [HYB-301 report](../../reviews/HYB-301-trace-classification/report.md) |
| HB-42 | deferred | Separate integrations permissions are not built. | HB-284/plan.md:363-386 |
| AUD-AR-02 | not connected | Signing, egress and endpoint checks are not commands; destinations are stand-ins over a fixed allowlist. | HB-284/verification.md Residuals |
| AUD-QA-04 | deferred | Idempotency keys and expected revisions are enforced; token issuance is not built. | HB-284/verification.md Residuals |
| AUD-QA-05 | deferred | Event and delivery states are shown; dispatch, signing and replay checks are not connected. | HB-285/verification.md Residuals |
| AUD-QA-06 | deferred | External application origins are a later design. | HB-284/plan.md:385, HB-285/plan.md:688 |
