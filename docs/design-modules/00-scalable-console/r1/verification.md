---
type: VerificationReport
title: HB-63 scalable screen prototype verification
description: Scope, independent findings, executed browser checks and remaining acceptance limits for the multilingual screen draft.
status: draft
---

# HB-63 verification

Ticket: [HB-63](https://linear.app/hybrid-billing/issue/HB-63/implement-multilingual-whole-product-scalable-screen-redesign). Date: 2026-10-31. Machine: mac_mini. Root implemented shell, fixtures, widgets, styles, tests and documentation; screen_catalog supplied catalog.js from an isolated worktree. Base: origin/main b48b23ce5b880ac4307f1436ede4fd1d7a726bb0. No accepted baseline file was modified.

## Delivered scope

The runnable draft contains 111 routes, retaining all 89 v10 screen identities and mapping all 49 registry IDs. Inventory coverage is not completed business-workflow parity. Four overview, 19 form, 31 list, 18 workflow, 28 detail, three file and eight report routes share the interaction framework. Detail/report values without a data contract are unavailable; workflows demonstrate review progression, not operational approval. Form drafts are memory-only. An explicit banner identifies synthetic data, missing backend connection and pending language review.

Six primary domains use TNB: overview, billing operations, sources/evidence, pricing/calculation, customer portal and administration. The searchable secondary navigation retains the eleven catalog groups. Arabic reverses logical navigation/table layout; identifiers retain isolated direction. No third-party fonts, images or executable dependencies are bundled. System font fallback is intentional in this draft.

The scale model is 100,000 synthetic accounts and up to 1,000,000 synthetic file records. Account search renders 20 results; lists render 25, 50 or 100 rows. File intake holds at most 100 metadata descriptors and displays 25 at a time. Demo jobs are capped at 100 with at most two processing jobs. These are bounded frontend demonstrations, not measured production database, upload or scheduler capacity.

NAVER Cloud and KT Cloud appear as required but unverified provider profiles. Real provider adapters, credentials, uploads and production API contracts remain outside this change. No file content, financial operation, customer data, live CSP call or external message is used.

## Independent review

interface_audit reviewed the design and actual changes separately. Actual-change verdict: PASS for the declared synthetic prototype. It independently ran local module checks for disjoint pages, scope-mismatched cursor rejection, scoped cancellation, duplicate/capacity outcomes and inventory/reference coverage. Its three initial findings were corrected and re-reviewed:

1. Account/customer context now filters jobs and guards cancel/retry lookup.
2. Duplicate and capacity rejection now produce explicit feedback.
3. File descriptors link to job IDs so intake reflects actual simulated processing status.

research_review applied read-only QA to code, assertions and representative screenshots. Its initial FAIL findings were corrected and re-reviewed to PASS: known-zero selections cannot submit; cancelled/retried jobs and completed workflow steps retain meaningful keyboard focus. The stale-response test was strengthened to start the older request, complete a newer request first and inspect again after the old timer completes. The reviewer inspected the refreshed parent-generated report, not a separately executed browser suite. Root ran the browser suite.

## Executed verification

The dependency-free prototype was served on loopback with Python http.server. Browser tests use Playwright 1.56.1 and the installed Chromium headless shell build 1243. Initial default-browser resolution and sandbox launch failed; the installed executable and authorized unsandboxed browser run were then used successfully. These failures are not recorded as successful checks.

`verify.cjs` completed with 65 passing checks, including all 111 routes in English, Arabic and Korean (333 route renders). Runtime assertions cover bounded pages, previous/next cursors, exact/unknown counts, page/all-filtered selection, exclusions, account 100000 search, keyboard picker behavior, account/customer job separation, duplicate submission, known-zero selection, loading/error/empty/expiry states, out-of-order requests, local file metadata intake without content reads, queue limits, cancellation/retry focus, workflow focus and draft feedback. Responsive checks cover 360, 768 and 1440 CSS pixels in each locale with no document-level horizontal overflow. Table/navigation scrolling remains local and intentional. No browser runtime error was observed.

Root inspected rendered English desktop, Arabic desktop, Korean tablet and English mobile screenshots. The independent QA reviewer inspected English desktop, Arabic mobile and Korean tablet. Full screenshot inspection of every route was not performed. Generated screenshots and machine output live under `output/validation/HB-63/`, are excluded from Git and are not promised as permanent hosted assets.

To reproduce, serve this directory, install a compatible Playwright runtime outside the artifact, and run:

```sh
HB_PLAYWRIGHT_MODULE=/absolute/path/to/playwright \
HB_CHROMIUM=/absolute/path/to/chromium \
node docs/design-modules/00-scalable-console/r1/verify.cjs
```

`HB_PREVIEW_URL` and `HB_QA_OUTPUT` can override the loopback endpoint and generated output directory. Each directly maintained file remains below 1,500 physical lines; source syntax checks passed. No new explanatory source comments were added.

## Remaining acceptance

- Native Arabic/English/Korean practitioner terminology review remains pending; the locale selector is ar-AE, not Saudi certification. HB-61 terminology evidence remains a separate PR.
- UAE references inform layout and typography; this is not a UAE regulatory, accessibility or federal-brand conformity certification. Full screen-reader and assistive-technology validation is not complete.
- Field-by-field business workflows, backend auth/scopes, tenant isolation, money contracts, indexed server search, stable server snapshots and real cursor tokens require production implementation and independent review. Client-side scope demonstrations do not authorize access.
- Large-file resume, multipart transport, durable queues, real retry recovery and ingestion throughput are not implemented or benchmarked here.
- Owner acceptance must identify the actual replacement scope before removing accepted v10 artifacts. PR review/merge and ticket completion are separate pending stages.
