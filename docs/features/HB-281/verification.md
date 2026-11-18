---
type: VerificationRecord
title: HB-281 verification of the live O03 contract lifecycle screen
description: Type check, unit tests, build, Playwright QA against the real API with the sample seed in EN, KO and AR at 1440 and 390 widths, regressions of O01, O02 and M01, and the outcomes of the design review, render comparison and change review. Run on mac_mini on 2026-11-17 with the real clock (UTC 2026-11-17).
status: stable
---

# Scope

This record verifies [HB-281](https://linear.app/hybrid-billing/issue/HB-281) against its [plan](plan.md). Every check ran on mac_mini (macOS) with Node 26.7.0, Chromium headless shell 1243 through Playwright, the console preview (`HB_GATED=1 vite preview`) and the `hb-api` binary built from main `4331d3e` with `--seed sample`. Only synthetic sample data was used.

# Results

| Check | Result |
| --- | --- |
| `npx tsc --noEmit` | clean |
| `npm test` | 402 pass, 0 fail (new: `service/contracts.test.ts`, `organizations/pages/contracts/live-model.test.ts`) |
| `npm run build` | builds |
| Playwright QA of O03 (71 checks) | 71 pass |
| O01 regression (qa273) | 53 pass |
| O02 regression (qa275) | 77 pass |
| M01 regression (qa264) | 51 pass |
| Console errors during QA | none |

**What the QA covers**
- **List:** five sample rows, "None" for zero gaps, filters by state, settlement and buyer with the "N of M" count, a contract ID that does not exist reads "This contract is not available".
- **Views:** parties (direct path reads "No seller (direct)"), products (components, "Not applicable (seller service)", rates not connected), schedule, termination, recognition and documents.
- **Commands through the real API:** suspend from the hinted earliest day (the date minimum equals the server's today), withdraw of that suspend, end from tomorrow with the confirm sentence about the first day not running, withdraw of the end with the required cancel choice (missing choice refused on the client, then "The contract continues" returns settlement to not applicable), missing evidence field errors and the emergency note, secured evidence with the cover-through hint and a resolved gap, settle with the minimum at the end day and the "final amount decision is not computed" sentence, the reopen and the resulting "evidence needed" hint, and the write log with actor login, reason and UTC times.
- **Refusals:** a rationale of 86 Korean characters (258 bytes) is refused on the client; a second administrator's write makes the first administrator's confirm return `revision_conflict`, the confirm step stays with the conflict note, and the next confirm succeeds.
- **Focus and announcement:** after success focus moves to the result status.
- **Layout:** no horizontal scroll on every view in EN and AR at 390 and in KO and AR at 1440, stacked cards at 390, `dir` set per locale, neutral tag and pair borders, table headers not sticky, keyboard order scope → reload → filters → list.

# Reviews

These reviews were done by AI agents, not the owner.

- **Design review** (independent reviewer, read-only): round 1 CHANGES REQUIRED (one High on attempt safety across remounts, six Mediums: not-found mapping, fieldless errors and the two `reason` fields with byte counting, exclusive-end wording, bounds from the server's today, sentinels, blocker sentences and fail-closed hints); round 2 CHANGES REQUIRED (superseded body text, pending-attempt status, null parsing); round 3 PASS, with its two wording items applied.
- **Visual specification and render comparison** (ui-designer, read-only): specification written before implementation; round 1 CHANGES REQUIRED (three Highs on table wrapping and an AR tag, Mediums on the filter disclosure and period labels, Lows applied); round 2 CHANGES REQUIRED (the AR header date order, the evidence column widths, a header overlap); all three were fixed. The header overlap came from a global sticky `th` rule; O03 headers are now static. The AR header date was checked in a fresh render.
- **Change review** (independent reviewer, read-only): round 1 CHANGES REQUIRED (four Mediums: a command could reach the previously shown contract, sign-out did not clear drafts and attempts, an invalid period left the panel loading, untested command bodies; ten Lows); round 2 PASS. Its remaining Low (per-event withdraw not gated by the revision check) was applied; its two Nits are recorded below.

No review is owner acceptance.

# Residuals

- Owner FIX of O03; KO and AR terminology review of the new copy.
- Single-administrator lifecycle and settlement commands (from HB-280).
- Money, proration, publication, ERP and the final amount decision are not computed; documents, payment terms and recognition are not connected.
- Contract creation, terms changes and server-side search beyond 200 contracts are out of scope.
- The mobile list is long (one card per contract with every column).
- Shell items outside this ticket: the "Payment Terms" tab is title case, the header shows "Work service not connected" while live data is shown, and the shell tabs do not support arrow keys.
- Change-review Nits left as they are: a form stays open after a refusal until closed (the server refuses again if confirmed), and the button row refreshes on the next render after an edit discards a pending attempt.
- External references in the plan were recalled, not opened.

# Planning cross-check (HYB-291)

Added by the [HYB-291 planning cross-check](../../reviews/HYB-291-planning-cross-check/report.md) on 2026-11-17. A record added by HYB-291; it changes no result above except where marked. The findings come from a read-only auditor; the HYB-291 writer agent (delegated by root) wrote this trace, the residual notes and the copy changes; the HYB-291 change review is their first independent check. Rows marked "(classified by HYB-301)" replace rows that HYB-291 left unclassified; the [HYB-301 trace classification](../../reviews/HYB-301-trace-classification/report.md) gives the evidence.

## Requirement trace

Source: `docs/ui/screens/O03/spec.md:121-155` (HB-280 is the API, HB-281 the screen). Status meanings are in the [cross-check report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). No acceptance test was run.

| ID | Status | Reason | Record |
| --- | --- | --- | --- |
| HB-01 | deferred | Covered (O03): contracts are linked to billing profiles and parties. Not built: the three operating profiles on one core, internal settlement without an external invoice, and blocking ineligible cross-entity or cross-currency sums. | HB-280/plan.md |
| HB-05 | not connected | No source connectors. | HB-280/plan.md:125-133 |
| HB-08 | deferred | Final evidence can be settled; durable evidence storage waits for the database decision. | HB-280/plan.md:125-133 |
| HB-10 | not connected | HawkEye results are not connected. | HB-280/plan.md:125-133 |
| HB-12 | not connected | On-premises cost pools are not connected. | HB-280/plan.md:125-133 |
| HB-15 | deferred | Lifecycle and state on a day are built; contract creation and change events are out of scope. | HB-280/plan.md:125-133 |
| HB-16 | not connected | Money is not computed. | HB-280/plan.md:125-133 |
| HB-18 | not connected | Exchange rates and rounding are not computed. | HB-280/plan.md:125-133 |
| HB-21 | not connected | `final_amount_decision: not_computed`. | HB-280/plan.md:125-133 |
| HB-22 | not connected | Prepaid credits are not connected. | HB-280/plan.md:125-133 |
| HB-23 | deferred | Lifecycle and settlement are single-administrator; two-person review is an owner residual. | HB-280/plan.md:125-133 |
| HB-24 | not connected | No ERP impact. | HB-280/plan.md:125-133 |
| HB-25 | not connected | No ERP adapters. | HB-280/plan.md:125-133 |
| HB-28 | not connected | Documents are not connected. | HB-281/verification.md Residuals |
| HB-32 | not connected | The assistant is not connected. | HB-280/plan.md:125-133 |
| HB-37 | deferred | Settle and reopen are built; access-removal plan and late-source window policy are not. | HB-280/plan.md:125-133 |
| AUD-AR-01 | not connected | No sale computation, so direct and resale paths are not exercised. | HB-280/plan.md:125-133 |
| AUD-AR-10 | not connected | No fixed-fee or SLA credit computation. | HB-280/plan.md:125-133 |
| AUD-DB-03 | not connected | No ERP submission. | HB-280/plan.md:125-133 |
| AUD-QA-01 | deferred | (classified by HYB-301) Built: the Documents and payment tab says it is not connected and names what it will hold (HB-281/plan.md:49). Not built: protected payment, recipient, tax identifier and template references, a separate change draft with verification, approval and a validity period, evidence of confirmation on the existing channel, and refusal of email-only or LLM-generated account changes (`O03/spec.md:56, 75, 151-153`). Confirmation on the existing channel depends on messaging, which pre-launch does not connect. | [HYB-301 report](../../reviews/HYB-301-trace-classification/report.md) |
| AUD-QA-02 | deferred | Covered: contract and settlement states are separate, ending is never shown as settled, pending_final with secured or missing evidence and reopening are built. Not connected: the emergency source-permission revocation path that revokes immediately and records the evidence gap (O03/spec.md:159-161). | HB-280/plan.md |
