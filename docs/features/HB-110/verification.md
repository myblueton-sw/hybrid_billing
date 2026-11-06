---
type: VerificationReport
title: Approval inbox and result evidence verification
description: Records the unbound approval read consumer, independent finance and security reviews, isolated synthetic browser checks and integration limits.
status: draft
---

# Scope and actual connection

HB-110 implements B09 on the ticket branch stacked on open B08 PR85 at `8f9a902e4d262d48244e654f25b2e545a60e3913`. Root owns PM/frontend integration and QA. A separate implementer owned the strict read contract/client/API tests; independent finance/TW and security/contract reviewers inspected design and actual changes. The owner-required Claude CLI ui-designer supplied composition and compared actual final renders. This is a consumer-only increment, not completion of the original financial approval workflow.

Capability discovery found no authoritative financial approval/ERP producer. The proposed same-origin GET `/api/approval-review` returned actual404 from the trusted local reader and the proxied browser rendered unconnected. Unknown count is not zero. No backend endpoint, financial command, source import, reset, schema or retained-data change was added. Live payloads are rejected as producer_unbound; synthetic payload acceptance exists solely to exercise isolated intercepted QA, with unconditional global/row/detail labels and no URL/window/storage fixture switch. Source metadata and shape checks do not establish deployed server authorization.

# Independent reviews and corrections

Finance/TW design conditionally passed five distinct stages, decimal strings without a universal currency scale, actual actor versus author, explicit ERP applicability and the monthly-recognition boundary. Actual contract/API/source review PASS; the reviewer independently ran18 API tests. Source-reported unknown enums are now distinct from absent source values. No source cost, role preview, result digest, declared limit or status grants present approval authority.

Security/contract design conditionally passed only the unbound, live-fail-closed read prototype. Actual contract/API and UI/App review PASS; independently ran18 API tests. Abort and sequence publication guards, query transitions, current-page canonical selection, strict exact nested validation, snapshot/permission cursor pins, streamed byte cap, redirects and review-state identity were inspected. A minor focus-return RAF null dereference was corrected with optional root lookup; independent scoped review closed it. Future producer binding remains a separate independently reviewed change.

Independent QA-script audit identified hidden full-table buttons being counted twice, a repeated-same-URL reread gap and indistinguishable/outstanding late-response assertions. Root corrected visible row counts, explicit fresh-document navigation, bounded interception waits, a distinct old sentinel record and old-response release/drain after the new response. Scoped source re-review and independent runner syntax check PASS. Browser abort/interception is not proof of every ignored-AbortSignal transport; production source separately checks abort and monotonically increasing request sequence.

# Executed checks

Full frontend suite:188 PASS, including18 B09 API tests and170 inherited cases. Production TypeScript/build PASS. Manifest verification:438 artifact hashes and seven retained license notices. All maintained changed source/test files remain below1500 physical lines; App.tsx remains1447. Existing large-chunk advisory remains.

The B09 browser runner checks actual default unconnected views separately from isolated intercepted synthetic views. It covers60 locale/layout/view captures (five views × KO/EN/AR ×1440/390 × two modes), request selection and Escape/focus return, exact source decimal strings, zero versus missing, explicit ERP N/A versus missing observations, actual-actor My decisions, monthly-only recognition, bounded25-row pinned pagination, browser-history reread, normalized/repeated search, exact empty versus unknown total, denied/stale/unavailable/retry/malformed/live responses, and late success/error after a newer response. No non-GET API request, page exception or owner-review acceptance was observed; browser storage remained without an owner review record. Synthetic cards are not actual approval requests.

Full B08 workflow and all36 existing locale/layout/view variants passed through the changed shared App integration, without page exceptions or API mutations. No prior leaf module was edited. B07 Marketplace remains held for CSP-specific follow-up; prior owner-FIX scopes are not changed or carried into B09.

# Render review and corrections

Root actual-image inspection found inherited flex alignment centering list content; B09 now explicitly stretches row text for logical-start alignment. The runner checks it. First Claude CLI actual-render comparison inspected12 requested default/synthetic/error captures and failed two visible items: mobile active tabs could remain outside the scroll strip, and unisolated English source blocker punctuation reordered in Arabic. Root replaced only B09 mobile tabs with a wrapping two-column grid and a full-width final tab; source titles/blockers/plain table actor names use auto-direction isolates. Canonical IDs/dates/money retain LTR presentation. The runner requires the selected tab bounds to remain visible, as well as16px controls/body,20px h2, local Plex family, Arabic1.8, neutral token borders, square/no-shadow controls and no document overflow.

Normal-size390px identity and money/ERP viewport captures supplement tall full-page images; tall auto-downsampled images alone do not certify mobile microtext. Recognition type is intentionally a read-only fixed monthly request value, not a broken select. The general history example includes pending/blocked source records; the My decisions example separately demonstrates a recorded policy decision and actual actor/time. No synthetic historical chronology or financial audit completeness is claimed.

# Reproduction and limits

From apps/console run `npm test`, `npm run build`, `node tools/verify-build.mjs` and `node tools/approval-inbox-qa.mjs --url <prepared-local-frontend> --output <local-evidence-directory> --playwright-module <installed-module> --chromium <browser-executable>`. The default checks require the prepared existing reader to return404 for the proposed unimplemented route. QA uses isolated temporary browser storage and intercepted synthetic responses; no credential profile, financial write or owner confirmation is used. Generated images/reports/CLI logs are temporary local evidence outside Git, not promised permanent published artifacts.

Actual host: macOS arm64, Node26.7.0, Chromium153.0.8010.12 headless build1243. Portable Node/browser interfaces and locally bundled fonts/assets are used; Windows/Linux installations, other engines, native-language/human finance review, full screen-reader coverage, real server permissions/SoD/ceilings/delegation, producer capacity, approval commands, ERP posting/reconciliation and owner FIX remain UNVERIFIED/deferred. Draft PR integration is not merge or deployment.

# Final review outcome and artifact

Final B09 runner PASS:60 main variants plus12 normal-size mobile viewport supplements,10 workflow/assertion groups, no page errors or non-GET requests. The final artifact fingerprint is `c8e6e02b3c768d78b96ed1c7c4074c856ba285417f7aa764bd6380be4456ff66`;438 manifest hashes and all188 frontend tests passed after the final UI correction.

Independent finance/TW rendered review PASS on12 requested default/synthetic/error captures and eight real-size KO-history, EN-recognition, AR-history and AR-recognition identity/amount crops. It rechecked refreshed EN/AR mobile defaults. Rendered detailed cards covered billing result, policy and recognition; adjustment/final-ERP details were additionally covered by source/browser assertions rather than separately certified rendered finance review. No native Arabic or human-practitioner acceptance was claimed.

Claude CLI final scoped re-review PASS on both material findings and mobile legibility. It inspected five refreshed mobile fullpages and seven real-size crops, plus scoped CSS/direction source. It confirmed visible active tabs, corrected English punctuation, readable amounts/IDs and intentional fixed recognition type. Its remaining minor observations concern count-line grouping, synthetic selected-status wording and the fixed-LTR synthetic reader-label isolate; real multilingual producer names require review when producer binding is implemented. Keyboard/engine/font-load execution was root's responsibility, not claimed as independent designer testing. Independent security source re-review of the final RTL/grid/QA/document scope PASS. All screen variants remain Ready for owner review, not FIX.
