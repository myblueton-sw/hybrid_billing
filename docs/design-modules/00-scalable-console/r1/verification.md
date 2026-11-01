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

Six primary domains use TNB: overview, billing operations, sources/evidence, pricing/calculation, customer portal and administration. The searchable secondary navigation retains the eleven catalog groups. Arabic reverses logical navigation/table layout; identifiers retain isolated direction. The initial draft used system font fallback. The brand follow-up below packages the supplied SVG assets and local IBM Plex fonts; the runtime remains native JavaScript.

The scale model is 100,000 synthetic accounts and up to 1,000,000 synthetic file records. Account search renders 20 results; lists render 25, 50 or 100 rows. File intake holds at most 100 metadata descriptors and displays 25 at a time. Demo jobs are capped at 100 with at most two processing jobs. These are bounded frontend demonstrations, not measured production database, upload or scheduler capacity.

NAVER Cloud and KT Cloud appear as required but unverified provider profiles. Real provider adapters, credentials, uploads and production API contracts remain outside this change. No file content, financial operation, customer data, live CSP call or external message is used.

## Independent review

interface_audit reviewed the design and actual changes separately. Actual-change verdict: PASS for the declared synthetic prototype. It independently ran local module checks for disjoint pages, scope-mismatched cursor rejection, scoped cancellation, duplicate/capacity outcomes and inventory/reference coverage. Its three initial findings were corrected and re-reviewed:

1. Account/customer context now filters jobs and guards cancel/retry lookup.
2. Duplicate and capacity rejection now produce explicit feedback.
3. File descriptors link to job IDs so intake reflects actual simulated processing status.

research_review applied read-only QA to code, assertions and representative screenshots. Its initial FAIL findings were corrected and re-reviewed to PASS: known-zero selections cannot submit; cancelled/retried jobs and completed workflow steps retain meaningful keyboard focus. The stale-response test was strengthened to start the older request, complete a newer request first and inspect again after the old timer completes. The reviewer inspected the refreshed parent-generated report, not a separately executed browser suite. Root ran the browser suite.

## Executed verification

The initial dependency-free prototype was served on loopback with Python http.server. The brand follow-up serves the built artifact instead of the source directory. Browser tests use Playwright 1.56.1 and the installed Chromium headless shell build 1243. Initial default-browser resolution and sandbox launch failed; the installed executable and authorized unsandboxed browser run were then used successfully. These failures are not recorded as successful checks.

`verify.cjs` completed with 65 passing checks, including all 111 routes in English, Arabic and Korean (333 route renders). Runtime assertions cover bounded pages, previous/next cursors, exact/unknown counts, page/all-filtered selection, exclusions, account 100000 search, keyboard picker behavior, account/customer job separation, duplicate submission, known-zero selection, loading/error/empty/expiry states, out-of-order requests, local file metadata intake without content reads, queue limits, cancellation/retry focus, workflow focus and draft feedback. Responsive checks cover 360, 768 and 1440 CSS pixels in each locale with no document-level horizontal overflow. Table/navigation scrolling remains local and intentional. No browser runtime error was observed.

Root inspected rendered English desktop, Arabic desktop, Korean tablet and English mobile screenshots. The independent QA reviewer inspected English desktop, Arabic mobile and Korean tablet. Full screenshot inspection of every route was not performed. Generated screenshots and machine output live under `output/validation/HB-63/`, are excluded from Git and are not promised as permanent hosted assets.

To reproduce, follow [the build guide](../../../design-build-guide.md), serve `output/designs/HB-63-console`, install a compatible Playwright runtime outside the artifact, and run:

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

## Brand and build follow-up

The owner identified missing logo/design-rule adoption after the initial prototype. The follow-up replaces the H placeholder with the unchanged Fusion AI lockup and mark favicon, imports supplied close/chevron icons, applies semantic tokens, removes rounded geometry/shadows and packages IBM Plex Sans/Sans Arabic/Sans KR/Mono locally. The base token file and seven SVGs match the supplied originals byte-for-byte; provenance preserves their hashes and metadata. React components remain unadopted. Body/table 16px, Arabic line-height 1.8 and responsive layout are explicit overrides in the active guide.

Active work entry points now include root AGENTS.md, .agents/rules/design-system.md, docs/design-system.md, docs/design-build-guide.md, the foundation entry page and module/tooling READMEs. Fontsource 5.3.0 dependencies are locked; their OFL-1.1 license files are copied into generated output and the root third-party notice identifies this scope.

Executed: locked npm preparation, source/asset checks and two clean builds produced 874 emitted files with byte-identical manifests. A synthetic unknown-file probe caused the build to fail while preserving that file, confirming the output ownership guard. The build rejects unknown files/symlinks and clears only its manifest-owned generated tree. No generated font, screenshot, node_modules directory or output artifact is staged.

The initial token migration introduced malformed white-space declarations; independent screenshot review caught vertical-character mobile navigation that the earlier overflow-only assertion missed. The declarations and malformed state token were corrected, the static guard strengthened, and computed nowrap checks added for TNB/table and mobile secondary navigation. Desktop secondary navigation intentionally supports wrapped titles.

Final browser run: PASS 98 checks, including all 333 localized route renders, correct unchanged-logo loading/LTR orientation, square controls/panels, restored navigation/table readability, actual loading of all four local IBM Plex families, zero external runtime requests, zero missing built assets and the prior scale/focus/error scenarios. Root inspected refreshed Arabic desktop/mobile and Korean mobile screenshots. Independent review records below distinguish source re-review from rendered evidence inspection. No native-language or accessibility certification is implied.

Independent follow-up verdict: PASS after source re-review and inspection of the refreshed 98-check report and Arabic mobile screenshot. The reviewer did not independently execute the browser suite. Its final reproduction-path documentation note was corrected to use the built output.

## HB-63 contribution follow-up — 2026-10-31

Added five owner-approved monthly tiers and separate individual/company/organization recognition guidance at `#community`. No roster or authenticated sponsor data is loaded. Source: [sponsorship decision and manual workflow](../../../community/sponsorship.md).

- `npm run build --prefix tools/design`: PASS, 8 immutable foundations, 875 emitted files.
- `node /private/tmp/hb63-preview.cjs`: PASS, 98 checks, 112 routes across 3 locales.
- `node /private/tmp/hb63-community-qa.cjs`: PASS, header entry, five tiers, contact link, EN/KO/AR at 360/1440px, no horizontal overflow or runtime errors; six screenshots in local validation output. Korean desktop screenshot visually inspected.
- Independent architect plan and actual-change review: PASS. Tier/identity separation, consent, private payment/contact exclusion, public branch and license boundaries, route/build integration verified. Reviewer also compared built module/hash and rendered localized markup.

Live tier publication, active sponsor verification, consent collection, automatic synchronization, native Arabic acceptance and benefit fulfillment remain UNVERIFIED. Existing onboarding/AI follow-up is not completed by this scoped change.


## Six-observation remediation (2026-11-01)

The [baseline audit](gap-audit.md) is retained for c8187fe. [Remediation](gap-remediation.md) connects the owner continuation, design constraints, changes and remaining business gaps.

- Build: PASS; 882 emitted files, eight unchanged canonical foundations. Onboarding, AI and journey modules are included in the reproducible build.
- Existing `verify.cjs`: PASS, 98 checks, all 112 routes in EN/AR/KO. The generic workflow test selects a remaining generic workflow because setup now has a dedicated implementation.
- New `verify-journeys.cjs`: PASS, 71 checks. Includes provider prerequisites/unsupported paths, new-setup acknowledgment and administrator read-back, denied/unknown diagnostics, partial receipt, financial publication block, implicit-submit prevention, revoked-acknowledgment/rerun invalidation, AI stale/error/partial/cancel/context behavior, running account/language/navigation changes, same-record evidence, Back search/selection/cursor/expiry and cross-account isolation, report descriptors, approval rejection and revision/currency reset, customer-only navigation and thirty EN/AR/KO 360/1440px layout checks. No browser error or external request was observed.
- Independent gap_review: design PASS with constraints; initial actual-change FAIL for stale onboarding prerequisites and implicit connector progression; scoped source re-review PASS after correction. Reviewer did not independently run browser tests.
- Root visually inspected Korean mobile onboarding, Arabic desktop approval and English desktop report captures. Generated evidence: `output/validation/HB-63-journeys/results.json` and PNGs. No claim of exhaustive visual, screen-reader or native-language approval.

The preview still does not calculate amounts, aggregate reports, authorize users, approve/issue invoices or call providers/models. Missing values remain unavailable. Document/ERP/customer-inquiry and complete recovery journeys from the baseline audit are not declared resolved.
