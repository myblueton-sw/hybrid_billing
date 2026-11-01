---
type: Verification
title: HB-67 React console verification
description: Records executed checks and remaining limitations for the first React frontend increment.
---

# Executed checks

Environment: macOS arm64, Node 26.7.0, npm 11.19.0, Playwright 1.56.1 with installed Chromium 153.0.8010.12. Browser storage is isolated synthetic QA storage. No real owner acceptance was created.

- Clean `npm ci --ignore-scripts --no-audit --no-fund`: PASS.
- `npm run build`: PASS, TypeScript strict check, canonical asset hash verification and Vite bundle with local fonts/licenses.
- `node tools/verify-build.mjs`: PASS, exact coverage and final-byte hashes of 241 output files, including HTML and seven licenses.
- `npm test`: PASS, 22 tests for valid/invalid assembly, slot ownership, bounded pages, known missing count, abort/late result eligibility and review persistence failures.
- `npm run qa` with explicit installed Playwright/browser paths: PASS, 22 checks. Default unconnected mode; ten-row synthetic result; selection invalidation; late old scope; back/search restoration; error/retry; empty state; extension renderer; isolated FIX; unsubmitted input invalidation; same-scope reload; exported variants; acknowledgment revoked during delayed digest; KO/EN/AR at 1440 and 390 pixels; unknown profile; failed storage; no page errors or external requests.
- Rendered Korean desktop and Arabic mobile inspected: bounded table scrolling, local logo, responsive shell and consistently positioned review block. Automated direction/overflow checks cover all six language/width combinations. Native terminology and full accessibility acceptance remain unverified.

Raw reports and six screenshots are reproducible through apps/console/tools/browser-qa.mjs and remain ignored under apps/console/qa-output; they are local working evidence, not permanently hosted artifacts.

# Independent review

Design review passed with a mandatory synthetic-only URL search boundary. Initial actual-change review found six issues: asynchronous acknowledgment revocation, unsubmitted input scope, missing exported variants, slot ownership, hardcoded contribution rendering and incomplete item coverage. Root corrected each; the browser and unit checks above cover the critical regressions. The next review found an incomplete manifest and a CSS hash recorded before final output transformation. Root moved manifest generation after finalized output. Final independent re-review: PASS; reviewer independently verified all 241 artifact hashes and reviewed the added documents without blocking findings. The reviewer independently ran the 22 unit tests; browser and clean-install results are root-executed evidence.

# Limits

Two React screens only. No production APIs, backend authorization, financial transactions, real owner FIX, remote deployment or merge. Windows/Linux, other browsers, Node minimum-version execution, screen-reader and native-language review are UNVERIFIED. Standard/Evidence are public examples, not secure per-customer isolation. Local evidence export requires canonical ticket/Git reconciliation before project acceptance.
