---
type: VerificationReport
title: HB-143 collected cost and usage verification
description: Records source-precision tests, independent financial and security review and rendered evidence for the collected S04 workflow.
status: draft
---

# Scope

HB-143 continues the owner's request to build the real service using the same collected FOCUS data as U01. Root owns implementation and integration. Dependencies are HB-85/PR77 collected projection and HB-93 S04 planning, not a newly merged main baseline. This report does not establish full planning parity, customer/production authorization, native-language acceptance or owner FIX.

# Implemented behavior

The default S04 route `/source-data` now reads the protected collected source, independent of legacy synthetic mode/scenario controls. The prior `service=local` inspector remains explicitly selectable by its existing deep links; it uses a different source and is not presented as the collected overview's detail. Direct legacy object IDs fail closed on the collected path rather than being guessed as row IDs.

The source module owns reusable collected DTO validation, exact decimal formatting and selection/handoff helpers. U01 keeps chart-specific aggregation/presentation. Shared projection content identity advances from1to2 when adding nullable list/contracted cost, consumed/pricing quantity, list/contracted unit price, native units and billing intervals. Schema1 remains additive-compatible. All numeric fields use exact coefficient strings at scale18. No original JSON, broad source fields, computed prices, blended quantities, FX, financial status or export is generated.

S04 displays normalized rows, mapped allowlisted source fields, native quantity/price rows and an explicit unavailable transformation-history tab. Account mapping is bounded/searchable and labels search as list-only. Full-scope cost aggregates are server-calculated; table pages contain at most25rows. Detail focus returns to its originating row, including modal-aware Escape. Each cost has currency; unit prices use pricing units; missing units remain unavailable. Source versions describe the full authorized source scope, not an invented row-specific mapping.

U01→S04 and S04→U01 preserve period/provider/account/basis/q plus the validated loaded snapshot while clearing offsets. Changed source/projection identity rejects the old snapshot. Errors/obsolete responses/scope changes clear prior protected content and detail. Owner review remains separate and pending.

# Actual verification

Host: macOS arm64, Python3.11.7, Node26.7.0, npm11.19.0, Chromium headless shell153.0.8010.12. Portable Python/Node/browser design does not certify Windows/Linux or other engines.

- Isolated `python -m pytest tests/focus_overview -q`: **36 PASS**. Covers original scoped projection/HTTP/ports plus extended18-place quantity/price serialization, missing unit without invention and old projection identity rejection. No retained-store reset suite was run.
- Console `npm test --prefix apps/console`: **88 PASS**. Adds source selection roundtrip, direct/post-filter loaded-snapshot pinning and normalized account-search scope. Existing exact-money/DTO tests require the new fields.
- TypeScript/Vite build and `node apps/console/tools/verify-build.mjs`: PASS;438 generated artifact hashes and7license notices. Existing bundle-size advisory remains.
- Actual retained-source API read:1,121rows/78account combinations,sixproviders,USD/CNY; extended fields present as allowlisted nullable strings, with the same underlying source snapshot used by U01. No source identifiers or public sample files committed.

# Independent review

Separate financial design review conditionally approved exact quantities/unit prices, snapshot continuity and projection revision2. Actual financial and architecture/security reviews initially failed S04's unpinned return link after direct/filter entry. Root passed the validated loaded snapshot into the handoff helper and added direct/post-filter tests. Both independent re-reviews returned scoped PASS; the financial reviewer independently ran3helpertests. Version-scope and missing-subaccount labels were corrected. Escape respects open dialogs and previously handled events.

These verdicts cover reviewed source contracts, not human finance fitness, source conformance or owner screen acceptance. Operational ledgers and U01's earlier controller/billing findings remain open.

# Reproduction and remaining acceptance

Use [HB-85 collected runtime prerequisites](../HB-85/focus-verification.md), the existing retained store and the same trusted reader process. No initialization or reseeding is required for this screen. Build from `apps/console`, start with `npm start -- --port 8805` and open `http://127.0.0.1:8805/#/source-data?lang=ko`. The dedicated API remains on8806 with trustedUI8805 and existing configured loopback storage ports.

Run the scoped rendered suite:

```sh
node apps/console/tools/collected-source-qa.mjs --url http://127.0.0.1:8805/ --playwright-module <installed-module> --chromium <installed-browser> --output output/validation/HB-143
```

Generated images/reports remain local under `output/validation/HB-143`; they are excluded from Git and do not create owner acceptance. Root records actual rendered and final CLI comparison results below after successful execution. Full S04 planning requirements such as global service/model/quality/attribution filtering, original-object evidence access, transformation/AI histories, export authorization and large-volume performance remain deferred contracts, not fake controls.

# Completed rendered and visual checks — 2026-11-04

The complete scoped browser suite passed. Six EN/KO/AR1440/390 variants read the actual1,121rows, display bounded25-row pages and preserve exact totals. It verified provider/month/empty/account scopes, source/usage/history tabs, row detail focus entry/return, Back, direct stale-snapshot clearing/restart, malformed extended DTO rejection, foreign-account clearing, U01→S04→U01 pinned cost equality with nonemptyq and the appropriate EffectiveCost basis. Additional isolated actual-response replay verified a slow obsolete source result cannot replace a fast new scope. Escape closes detail and restores its row; an open review dialog owns the first Escape. No acknowledgement or review confirmation was saved.

The rendered report is PASS with zero page errors and no viewport overflow. Current final title/scroll-hint captures also passed all six locale/layout checks. Browser tests pin their assertions to settled query and tab identity; hash URL mutation alone is not treated as completed rendering.

Claude CLI final comparison initially failed a real long Oracle SubAccountId crossing into adjacent columns. Root added definite-width account content, normal whitespace, bounded identifier wrapping and metadata400, then captured actual table views at desktop/mobile. The separate CLI scoped re-review returned PASS for identifier containment/table readability across KO/EN/AR desktop and KO/AR390 tables plus EN usage. Minor count-formatting and row-hover polish remain nonblocking. The final scroll hint and page-title copy were checked by root in six final captures, not represented as separately inspected by the CLI. Native-language and assistive-technology acceptance remain pending.

This increment reuses U01's compact title/review cluster only for the collected S04 candidate; other screens' shell composition is not changed. U01's shared DTO import and explicit source link also change its candidate revision, so no previous owner acceptance is carried over. All reviews are source/visual scope verdicts, not professional human financial approval.

The retained local query is bounded to10,000authorized rows and two admitted HTTP requests, with protected source verification serialized by the existing coordinator lock. This1,121-row execution does not establish large-organization throughput, arbitrary-size search or backend cancellation. No protected-read/integrity checks were bypassed for speed.
