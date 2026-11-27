---
type: ReviewRecord
title: User-list retry focus and refresh announcements
description: Reproduced keyboard continuity failure and verification of scoped M01 read recovery.
status: stable
---

# Final status

Record of a step of HYB-313, which closed on 2026-11-27; the remaining verification is listed in [findings](findings.md) and carried by HYB-399.

# Finding and correction

HYB-313 remains in progress. On M01 users, activating Retry after an unavailable read removed the focused button. All six EN/KO/AR desktop 1440/mobile 390 reproductions left BODY active during loading and after success, with no completion message in the live region. The initial harness did not match queryless request URLs and then assumed one initial request despite development StrictMode; those failed harness runs are not product evidence. The corrected discovery collection is `/private/tmp/hyb313-read-recovery-before3/evidence.json`.

Independent design review approved transferring focus synchronously to the existing toolbar Reload button before Retry removes its own button, and reusing the established localized list-update announcement. The reviewer required request-lifecycle scoping and preservation of later user focus movement. Root added stable `m01-reload`, a shared explicit refresh handler and a user-refresh flag on the existing load operation. Only the latest successful response sets pending reload intent, and existing filter/removal intent takes precedence. A subsequent load clears unconsumed reload intent. Failed or stale responses never set it. Automatic reads retain their existing behavior. Completion does not explicitly move focus.

No new translation, visual composition, dependency or external asset was introduced. Authorization, data scope, failed-read handling and latest-response suppression are unchanged. The existing unauthenticated branch still offers sign-in recovery.

# Behavioral evidence

The six final keyboard journeys in `/private/tmp/hyb313-read-final/evidence.json` pass unavailable→Retry→repeated failure→Retry→success, normal toolbar Reload and moving focus while a response is held. Retry puts focus on Reload before the response returns; repeated failure preserves that stable control; successful recovery keeps it focused. Normal reload announces an actual fresh empty→localized-message transition. When the user tabs elsewhere while waiting, completion preserves that later focus. All six cases have zero page errors and document overflow. No account command is submitted.

The final lifecycle run in `/private/tmp/hyb313-read-lifecycle-final/evidence.json` passes silent initial background read, repeated reload with the latest response resolving first, a filter change superseding an outstanding refresh, real empty search results and a late earlier response. Filter completion announces exactly “List updated. No people shown. Filters applied: 1.” The late earlier responses do not replace the rows or mutate the status region. The mutation observer is sampled before releasing each old response and remains active through a 150ms observation window, covering the existing 50ms status-publication timer. This bounded timer check tests delayed publication; it is not a substitute for readiness signals.

Independent source review found that the first normal-reload assertion could match an already-present message and that two animation frames did not cover the status timer. Root strengthened both checks and reran primary/lifecycle scenarios. Earlier passes are not the final evidence for these two properties. Source review distinguishes inspection of stored evidence from test execution.

# Rendered review and limits

Actual Claude CLI ui-designer reviewed twelve error/recovered viewport captures in `/private/tmp/hyb313-read-recovery-after/` and returned scoped visual PASS without blockers. It observed neutral borders, readable EN/KO/AR copy and visible focused Retry/Reload controls. Focus behavior and announcements are established by root's browser journeys, not still images. Root inspected the Arabic mobile recovered render. The reviewer noted that Retry outlines were near the screenshot edge; a follow-up capture centers the focused error control and records computed borders without altering product layout.

The apparent four-row/three-selectable-person difference is expected: the current actor is shown in the list but excluded from bulk selection. This correction does not alter selection semantics. Human screen-reader output, native-language acceptance, native browser zoom and additional browsers/operating systems are unverified. The executed environment is macOS arm64, Node 25.9.0 and Chromium 159.0.8154.6. The user's running stack is preserved; no merge or deployment is claimed.

# Reproduction

Run `node apps/console/tools/read-recovery-qa.mjs` with explicit `--url`, isolated `--api`, disposable `--activation`, `--playwright-module`, `--chromium` and `--output` paths. The default runs the six locale/width journeys; `--lifecycle 1` runs the English desktop ordering cases. Each actor activation is one-use; use a fresh disposable API or a distinct unused fixture actor via `--user`. Credentials stay in memory. HTTP 503 injection is browser-local; successful reads come from the disposable API. Generated screenshots and JSON remain outside Git.

The follow-up `/private/tmp/hyb313-read-visual-final/evidence.json` reran all six strengthened journeys successfully and captured centered error controls. Root checked all six computed results: buttons use four neutral 1px borders (RGB 141,148,156), errors use only a neutral top separator (RGB 220,225,229), and focused Retry uses the transient 2px outline (RGB 35,91,148). Actual Claude follow-up visual re-review opened all six centered error captures and returned scoped PASS, confirming the entire Retry outline and readable content. The report is `/private/tmp/hyb313-read-visual-rereview.txt`; it does not establish focus behavior or screen-reader output.

Full TypeScript/build passed with the existing bundle-size warning. Eighteen existing filter-model, filter-render and table-render tests passed. Final independent source and technical-writing review returned scoped PASS after inspecting the strengthened runner, stored evidence and logs; the reviewer did not execute checks or inspect screenshots. This does not complete the wider accessibility audit.
