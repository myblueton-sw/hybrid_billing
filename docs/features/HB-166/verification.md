---
type: VerificationRecord
title: Receipt-reader browser suite correction
description: Records why the receipt-reader browser suite failed on main, what was corrected in the suite, and what was executed.
status: draft
---

# Finding

[HB-166](https://linear.app/hybrid-billing/issue/HB-166/receipt-reader-browser-suite-fails-on-main-four-receipt-controls-where) was opened during HB-164 because `apps/console/tools/receipt-reader-qa.mjs` failed on main `6374496` at its first receipt count (four controls where two records exist). Root (Claude Code) investigated and corrected it on 2026-11-12. Machine: mac_mini.

Cause, read from Git history: HB-150 (`27c5b8c`) changed the S03 screen and added `collection-status-qa.mjs` for it, but did not update this suite, written for HB-145. Three expectations no longer described the screen:

| Expectation in the suite | Screen since HB-150 | Correction |
| --- | --- | --- |
| One `.receipt-open` control per record | Each record is rendered twice, as a table row and as a card; CSS shows the table from 720px and the cards below | Locators count and click the visible control; in-page counts read the table |
| Every time reads `YYYY-MM-DD HH:MM:SS UTC` | A midnight UTC instant reads `YYYY-MM-DD UTC` in lists | The pattern accepts the date-only form |
| A refresh button inside `.receipt-refresh` on two views, and "Refresh evidence" inside the search form | One refresh button in `.receipt-scope-strip` on every view | The three selectors point at the scope strip |

Two further mismatches were found by the change review and corrected, both hidden because the suite never waited for the collection summary that HB-150 added:

| Hidden dependence | Correction |
| --- | --- |
| The typography rule (16px for every visible header cell) only held while the summary was still loading; the summary's header cells are 14px by HB-150's design | The live loop waits until the summary is no longer loading and applies the rule outside the summary |
| The isolated fixture did not route the summary's request, which went to the real reader with a scope that cannot match and would clear the screen if it answered in time | The fixture answers that request with a non-authority failure, which leaves the receipt screen in place |

Assertions tightened in the same pass: the cases that expect no receipt control count controls in the document, not only visible ones; where records exist the suite asserts that the table is the visible form from 720px and the cards below; the date-only time form is accepted only in the lists and only for a midnight UTC instant.

No screen code is changed. The screen was not found defective: the doubled control is the responsive pair introduced on purpose, with one of the two hidden at every width.

# Executed checks

| Check | Actual outcome |
| --- | --- |
| The suite on main `6374496` before the correction | FAIL at line 78, `4 !== 2`, twice against main's build and twice against the HB-164 build |
| The suite after the first three corrections | PASS twice: 45 checks, 43 captures |
| The suite after the review corrections, against the console built from this branch (no console source change) on the retained sample | PASS twice: 45 checks, 43 captures, no page error, no non-GET request. HB-145 recorded the same counts |

The suite ran against the reader restarted from main, read-only, while another browser suite was reading from the same reader. No reset, ingestion or benchmark. Host: macOS arm64, Node 26.7.0, Playwright 1.56.1 with Chromium headless shell build 1243. Captures stay local under `output/validation/HB-166`.

# Independent review

The repository's read-only `independent-reviewer` agent role (AI) reviewed the first version of the correction on 2026-11-12 and executed nothing. Result: the three corrections match the screen; one HIGH (the suite passed only while the summary was loading) and three smaller weakenings, all corrected as described above. No second round ran on the corrections.

# Limits

- Not corrected, recorded from the review: the list-text size rule measures the hidden table at 390 and nothing measures the cards' 14px labels; the focus-return check accepts any receipt control; times inside the detail panel are not checked; the suite's default output directory still points at HB-145's evidence folder.
- Whether the 14px card labels are intended is a design question that was not judged.
- This suite and `collection-status-qa.mjs` overlap little and both are kept.
