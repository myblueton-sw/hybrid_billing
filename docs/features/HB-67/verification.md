---
type: Verification
title: HB-67 React console verification
description: Records executed checks and remaining limitations for the first React frontend increment.
---

# Implemented correction and current boundary

The first candidate's design-fidelity failure below remains part of the record. Root has now corrected the six-domain shell, all 112 navigation identities, searchable rail, directory and intake layout in React. Independent source re-review passed after fixing scope, selection, expiration/retry, shared dependency direction and style findings. This is a scoped correction, not completion of all 112 screen implementations or owner acceptance. PR 63 remains draft for the remaining screen-porting scope.

Actual correction checks on the same macOS/Node/Chromium environment:

- Strict TypeScript/Vite build: PASS. Final artifact verification: 242 files PASS.
- Unit tests: 23 PASS, including provider count, descending order and last-page boundaries.
- Browser runner: 34 PASS. Covers six-domain order; 24-card directory/search; default unconnected state; review-before-content; original table columns; 25/50 paging; bulk selection/review/exclusions; account invalidation; provider filter; browser history; error retry; expiration recovery; empty results; cards/columns; extensions and canonical IDs; scoped FIX; same-tab directory refresh; shell account invalidation; revoked acknowledgment; six KO/EN/AR desktop/mobile combinations; customer navigation boundary; invalid profile; failed review save; no runtime errors/external requests.
- Root compared the corrected Korean directory render with the existing HB-63 Korean desktop capture. Header, top domains, rail, heading, review-before-content, statistics, filters, cards and related links retain the baseline hierarchy. Review wording, synthetic snapshot/data and separated review tools have explicit implementation differences; no pixel-exact parity assertion.
- Root inspected Korean intake and Arabic mobile output. A full-page Chromium capture showed a blank region in the scrollable table; a separate in-viewport capture and measured table geometry confirmed the header and rows render in the visible table. Raw images remain ignored QA artifacts.

The reviewer independently inspected source corrections; the 34 browser checks and rendered comparisons were executed by root. Other destinations remain explicitly pending; full account-picker scale, background jobs, all screen bodies and cross-OS/native-language/accessibility acceptance remain unverified. No real owner acceptance or merge occurred.

# Historical rejected candidate

# Corrected outcome after owner review

Design fidelity: FAIL. The owner identified that acceptance of the internal module/file structure did not authorize replacing the existing top-level IA or screen composition. PR 63 is a draft requiring correction. The previous completion statement is withdrawn. The technical test results below remain historical facts; they do not establish design compliance.

Compared with HB-63 baseline commit 4d51763458b314fd766998c05fb32a1c40c99e72, the React candidate omitted the six-domain top navigation, searchable full secondary navigation, account/jobs header controls and the established screen directory. It substituted an implementation workspace and a reduced two-route rail. Reusing tokens and logos was insufficient. Baseline-to-render comparison was missing from the acceptance checks.

Corrective work must trace each changed screen to the baseline app.js, catalog.js, design-directory.js and screen-specific designs before changing JSX. Preserve existing domain order, menu identities, labels, header hierarchy, screen composition and item meanings. Code-folder composition is an internal implementation choice; it cannot redefine product IA. Any unimplemented interaction must be identified explicitly without silently deleting the planned UI. Verify baseline and React renders side by side in KO/EN/AR at desktop/mobile sizes, including navigation and review placement, before another completion claim.

Independent fidelity audit also identified the review strip moved below content, changed intake controls/columns and replacement semantic item IDs. Intake must retain source-reference/amount/status/evidence-reference columns, its existing filter/action/pagination controls, and unavailable amounts where no contract exists. Preserve canonical IDs such as index-intake:field:record-search, index-intake:field:provider-filter and index-intake:action:record-detail. The directory must retain 24-item paging and its search/area/review-status filters. Baseline responsive rules include a 246px desktop rail and a horizontal rail below 1024px. These are correction requirements, not completed fixes.

No corrected React screen is claimed in this record. Existing HB-63 designs remain the visual reference. No owner FIX or acceptance has occurred.

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
