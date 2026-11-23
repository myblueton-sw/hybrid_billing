---
type: VerificationRecord
title: Five server-backed browser tests that failed on main
description: Cause and correction of five failing browser tests of apps/api/server, the runs that show it, and what remains open.
status: stable
---

# Outcome and ownership

HYB-350, machine:mac_mini, 2026-11-24. Five browser tests of `apps/api/server` failed when run on main 8a4c5eb. None of the five was a product defect. Two needed a different way of serving the console, two used a check script that had fallen behind the screen, and one had a frontend address written into the test. The last three are corrected in this change. No product code changes.

# Causes

| Test | Cause | Kind | Correction |
| --- | --- | --- | --- |
| `TestApprovalProjectionRenderedBrowser` | the check script imports the console source module `/src/service/api.ts` in the page, which a built preview does not serve | requirement of the run | none: run against the dev server |
| `TestFinancialDecisionCoreBrowser` | the same | requirement of the run | none: run against the dev server |
| `TestResultActualScreenBrowser` | `tools/result-screen-live-qa.mjs` filled "the" text area of the result inspection; the final approval request section added later put a second text area there, so the locator matched two elements | drifted check script | the script fills the first text area, which is the action reason |
| `TestResultReplacementScreenBrowser` | the same script | drifted check script | the same |
| `TestDocumentActualScreenBrowser` | the test allowed the origin `http://127.0.0.1:8896` and no other, so with the console served on another port every command was refused with `origin_refused` | fixed address in the test | the test allows the origin given in `HB_PROJECTION_QA_URL`, as the other browser tests do |

The result script now also records the first line of the error in its evidence file (`detail`), as the other scripts do. Before, it recorded only the error's name, which is why the cause could not be read from the evidence.

`HB_DOCUMENT_SCREEN_QA_MODE` was not the cause: the document test passes with the variable unset.

# Runs

All runs: isolated in-process server started by the test, headless Chromium, synthetic data, no real account or customer data. They are not an owner acceptance of any screen.

1. Main 8a4c5eb, console served as a built preview: 12 passed, 1 skipped, 5 failed (the five above).
2. b950947 (before the HYB-314 fixes), built preview: the same five failed, so the HYB-314 fixes did not cause them.
3. This branch, console source served by the dev server, whole suite (`go test ./server/ -run 'Browser|GrantScreens'`): 16 passed, 1 skipped, 1 failed. The five above pass. The skipped test is `TestFinancialDraftLiveBrowserJourney`, which needs its own runner settings.

# What remains open

The one failure in run 3 is `TestIssuanceSetupActualScreenBrowser`, which passes against the built preview (run 1) and fails against the dev server at stage `change-request-2`, waiting for the field cards of the documents view. Its cause is not established. It is tracked in HYB-351.

Because two tests need the dev server and that one currently needs the built preview, no single way of serving the console passes the whole suite today. Until HYB-351 is settled, a complete run takes two passes: the dev server for everything except the issuance setup test, and the built preview for that test.

# How to run

Serve the console on a loopback port, then set `HB_PROJECTION_QA_URL` to it, `HB_PROJECTION_QA_PLAYWRIGHT` and `HB_PROJECTION_QA_CHROMIUM` to the Playwright module and the Chromium executable, and one output directory variable per test family (`HB_PROJECTION_QA_OUTPUT`, `HB_DOCUMENT_SCREEN_QA_OUTPUT`, `HB_DOCUMENT_IMPACT_QA_OUTPUT`, `HB_FINAL_PERMISSION_QA_OUTPUT`, `HB_ISSUANCE_SETUP_QA_OUTPUT`, `HB_RESULT_AUTHORITY_QA_OUTPUT`, `HB_RESULT_EVIDENCE_QA_OUTPUT`, `HB_RESULT_IMPACT_QA_OUTPUT`, `HB_RESULT_REPLACEMENT_BLOCKED_OUTPUT`, `HB_RESULT_REPLACEMENT_QA_OUTPUT`, `HB_RESULT_SCREEN_QA_OUTPUT`). A test whose output variable is unset is skipped.
