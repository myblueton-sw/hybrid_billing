---
type: VerificationReport
title: Billing preparation and source comparison verification
description: Records B08 actual input reads, independent authority and financial-display reviews, memory lifecycle and rendered checks.
status: draft
---

# Scope and capability evidence

HB-109 stacks on held B07 PR84 at `1a7912d9d2bcef56ef0251e7fbe743fba49df982`; separately read main was `36394edf179d544d7ec2756f5329eba60bcc2d4e`. Root owns PM/frontend/QA, an isolated implementer wrote B08-only API/model/tests, and independent finance/TW and security/contract reviewers inspected design and actual code. Claude CLI ui-designer supplied actual-render-based composition. No prior accepted/implemented leaf was edited; Marketplace CSP-specific follow-up remains held, not FIX.

Read-only discovery found no authoritative financial run/history/policy/claim/approval/ERP-recognition producer. Existing source ingestion/checkpoints, protected customer-usage exports and numeric probes are not billing calculation. B04/B05 browser drafts are not approved policy bindings. Therefore this increment implements actual source input review and a temporary comparison baseline, not the full original B08 billing/approval/recognition workflow. No demonstration run, result hash, zero run count or fabricated progress was substituted.

A fresh protected `/api/focus-overview?period=all&provider=` read confirmed 1121 authorized/selected source rows and 78 accounts. Whole-selection CNY has two rows, Billed/Effective 11; USD has 1119 rows, Billed 27.37875272899 and Effective 18.72651418586, each with present count equal to rows and missing zero. These are public/synthetic collected source amounts, not eligible customer charges or monetary settlement. Source/API/backend state was not changed; no import, reset, schema, financial command or owner acknowledgment occurred.

# Independent design and implementation review

Finance/TW design conditionally passed explicit source-only terminology, complete-measure same-currency exact differences, volatile baseline lifetime and the expense-recognition boundary. Contract review required exact protected source_snapshot equality in addition to company/mapping/member/timezone/version/provider/account/period identity, because no historical aggregate reauthorization API exists. Changed source references discard baseline amounts before any comparison render. The B08 validator strengthens the existing overview client without modifying shared consumers or adding a backend endpoint.

The API/model writer passed 14 isolated tests and TypeScript. Independent contract review also ran those 14 and passed scope, real timestamps, identity/page/amount guards, pre-render baseline compatibility, cancellation/publication guards and review identity. Its nonblocking direct-history lifecycle observation led root to track provider/account/period request transitions independently; the scoped source re-review passed. This clears a baseline even when an intermediate source request is canceled before returning.

Finance independently reproduced a display defect: a partial baseline subtotal looked unqualified although its delta was unavailable. Root reused amount-plus-present/missing presentation on both comparison sides and added an SSR regression requiring baseline 1/1 versus current 2/0 counters and an unavailable delta. Independent reproduction and scoped source re-review then passed; the final isolated model/API/render suite has 15 PASS. Full frontend suite has 170 PASS. Exact BigInt cases cover signed/large/zero differences, partial/all-missing/empty and missing-currency observations, incompatible source/scope identities, deep-copied full aggregates and display-only paging/search.

No B08 baseline is stored in localStorage, a server, a file, a download or the URL. It is one bounded in-memory copy of full-selection aggregate evidence. The complete current/baseline state and source references participate in the common scoped review digest. Financial result, cause attribution, policy equality, legal eligibility, complete validation and owner acceptance are not inferred.

# Browser verification

The browser runner uses actual protected public/synthetic data for normal scenarios and isolated intercepted responses for denied/busy/stale/changed-reference/partial/race states. It never changes the retained data or creates a financial command. Checks include temporary pin, no immediate self-comparison, fresh same-source read, full-scope totals across row/account pages and account search, canonical row detail/close/Escape/focus return, source-scope invalidation, failed/denied/stale removal, changed-source invalidation before old values render, late A after B and canceled intermediate scope/history navigation, unmount lifetime and empty localStorage. Synthetic partial comparison verifies both sides’ presence counters and unavailable differences in the actual browser.

An initial runner assumed viewport changes discarded the baseline. The screen correctly preserves it within the same scope; the runner now explicitly discards between capture cases. The complete actual-data/intercepted-state workflow and 36 variants then passed without page errors or non-GET API requests; localStorage stayed empty. Actual loaded font checks passed for all three Plex families. An extra focused desktop keyboard reproduction subsequently found Escape closed the full table while retaining selection. Root changed only the desktop full-table branch to clear selection while keeping rows visible. Before: table closed with selection retained; after: table open and selection empty. Independent scoped source review passed. The final render-only run passed all 36 variants with an explicit desktop row-trigger Escape/reopen regression in all three locales; this was a scoped rerun, not a claim of repeating unrelated backend tests.

# Render review and final artifact

Independent finance/TW actual-render review PASS: KO desktop all six views, EN mobile comparison/recognition, AR mobile runs/validation/comparison, and the isolated synthetic partial-comparison capture. Both observed source-cost bases, separate currencies, presence/missing counters, unverified eligibility and unconnected financial/ERP boundaries retained their meaning. Native-language/human finance and owner acceptance were not performed.

Production build PASS; manifest verification PASS with 438 artifact hashes and seven license notices. Final frontend fingerprint: `c4c4e77d5141741c8bfc08329058574c186dbd6d6fbd51adde012e6ab22997d8`. All changed source/test files remain below 1500 physical lines. Existing B07 actual workflow and all 24 locale/layout/view variants passed through the updated shared app. The first final designer comparison passed the financial/layout boundaries but found an EN-mobile account-count wrap. Root applied the existing no-wrap count class to the entire summary count isolate; final scoped designer re-review PASS on EN mobile runs/comparison/recognition and source no-wrap rule. A separate normal-size AR mobile detail viewport confirmed mirrored layout and LTR source values without relying on the unreadably tall full-page image. Final 36 render variants passed again with the computed no-wrap summary-count check and desktop Escape regressions.

# Reproduction and limits

From `apps/console` run `npm test`, `npm run build` and `node tools/verify-build.mjs`. Browser runner `tools/billing-runs-qa.mjs` runs workflow and renders by default; `--renders-only true` targets visual and desktop Escape regressions. It accepts explicit URL, local output, installed Playwright module and Chromium executable arguments against the existing prepared protected local reader. No backend test/reset suite is required by this frontend-only increment. Actual host: macOS arm64, Node 26.7.0, npm 11.19.0 and Chromium headless build 1243.

Portable Node/browser interfaces are design targets; other OS/browser engines, native-language/human finance, full screen-reader coverage and production capacity remain unverified. Financial run/policy/claim/approval producers, original AR acceptance, ERP monthly recognition/quarterly reconciliation and owner FIX remain pending. Existing large-chunk advisory and shared mobile-tab geometry remain documented limits. Captures/CLI logs are temporary local artifacts outside Git, not permanent published evidence.
