---
type: VerificationReport
title: HB-85 commercial overview verification
description: Records actual rendered and independent-review evidence for the first commercial U01 body redesign, with integration and acceptance limits.
status: draft
---

# Scope and environment

HB-85 changes the U01 overview body, retains the pending HB-76 shell and does not complete other screen redesigns. Source revision shown by the final build is `e68833797c09` (short form). Host execution: macOS arm64, Node 26.7.0, npm 11.19.0 and Chromium headless shell 153.0.8010.12. OS-neutral Node/npm/browser implementation is retained; Windows/Linux and other browser engines are unverified.

# Executed verification

Run from `apps/console`:

| Check | Result |
| --- | --- |
| `npm run build` | PASS: TypeScript and Vite production build; existing large-chunk advisory remains |
| `node tools/verify-build.mjs` | PASS: 242 artifact hashes, HTML and seven license notices |
| `npm test` | PASS: 76 tests, zero failures |
| `npm run qa:overview -- --url <loopback-preview> --playwright-module <installed-module> --chromium <installed-browser>` | PASS: 12 small/enterprise renders and scoped journeys |
| `git diff --check` | PASS before commit |

Browser regression covers EN/KO/AR at 1440px and 390px, scoped exact USD/KRW totals, partial/all-missing/verified-zero/empty/unconnected states, source/organization/currency/month changes, 2,400 departments and 4,800 closing units, bounded pages, hierarchy selection, compact URLs, list-filter independence, attention-detail entry/close focus restoration and scope invalidation. Source/organization money stays within the compact viewport. The workboard precedes charts everywhere and precedes amounts in mobile visual and DOM order. Computed headings are 20px, metadata 14px, table/control/body text 16px. Actual selectors distinguish source-record coverage from the whole-unit closing denominator. A scoped CSS guard found no literal colors, rounded controls, shadows or gradients; the canonical asset hash checks ran during build.

Six final captures under `output/validation/HB-85` show local font families with loaded font status and zero external runtime requests. Full 12-view screenshots remain generated under `apps/console/qa-output/overview`. Generated evidence and dependencies are excluded from the commit. Final visible review status is pending; no confirmation was performed.

# Failures found and corrected

The initial source review found reversed baseline/override stylesheet imports. Browser checks then identified equal-specificity heading/control overrides from the shared planning shell; scoped specificity corrected the measured values. The first existing regression run referenced a removed fixture selector and obsolete pre-HB76 directory controls; explicit example configuration and current design-directory routing now exercise those boundaries. A prior panel-color assertion incorrectly treated a zero-width border as a painted border; the check now accepts absent edges and verifies the remaining neutral edges.

Claude CLI visual review found an unqualified partial chart headline. The selected headline now labels partial and missing counts directly. Compact source/organization tables now stack complete currency values instead of leaving money fragments at the initial scroll position. Single-page exception controls remain disabled and use muted ink. Attention actions have distinct accessible names.

# Independent review

The separate planning reviewer returned conditional PASS for presentation-only scope with distinct financial meanings, overlapping attention counts, whole-unit denominators, scoped fixture inspection and explicit unknowns. The separate implementation reviewer returned PASS after fixes, including the final partial-label, currency and compact-table changes. The final all-missing cost wording refinement also passed independent source review and the browser rerun. This was source review, not independently run browser QA.

Claude CLI ui-designer produced the composition and then reviewed the six first-pass rendered variants with the installed official frontend-design skill. The initial render verdict was FAIL on the partial headline, with typography unverified. Final Claude CLI re-review returned scoped PASS: partial headline and compact money clipping are fixed; typography PASS relies on root computed-browser evidence, not independent measurement. It inspected final viewport/analysis captures, full EN/AR compact renders and targeted source. Desktop lower-table sections were not independently inspected in that final visual pass. Nonblocking refinements remain: chart value scale, stage-link density and compact headline wrapping.

# Limits

This is a commercial screen-composition increment over synthetic costs and closing states. No production authentication, overview aggregate, sales/allocation ledger, authoritative ERP feed, receipt allocation, payment terms, deadlines or assignee directory is connected. No refresh, financial command, amount conversion or approval authority was added. Full assistive-technology/native-language review and non-Chromium table semantics remain unverified. Owner FIX, production readiness, merge and deployment are not established by these checks.
