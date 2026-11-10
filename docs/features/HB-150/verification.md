---
type: VerificationRecord
title: S03 current stored-data observations
description: Records scoped collection-status integration, receipt meaning, actual rendered checks and remaining acceptance boundaries.
status: draft
---

# Delivered behavior

[HB-150](https://linear.app/hybrid-billing/issue/HB-150/refine-s03-collection-status-using-verified-retained-sample-data) refines S03 from the HB-149 retained sample baseline. Root owns frontend, integration, terminology and QA; a separate backend writer owns the producer/HTTP tests, a separate read-only reviewer checks contract/source/TW/results, and the actual Claude CLI ui-designer leads composition and compares renders. Machine: mac_mini. Onboarding remains deferred; S04 field-detail refinement is next.

The default screen displays current permitted stored rows, contributing current deliveries, provider/source-window counts and public/synthetic provenance before independently counted whole-file receipt metadata. Actual protected and console-proxied reads return 1121 rows/3 contributing deliveries/6 providers versus 2 wholly attributable receipts. Source windows contribute 115, 1005 and 1 allowed rows; provenance is 1000 public samples, 115 public subset rows, 6 synthetic rows and 0 unclassified. No excluded whole-file identity, count or metadata is disclosed. No source count is hardcoded in the product.

Receipt-prefix search filters receipts only. Both observations retain their own timestamp/snapshot. The S04 link passes only language, not a receipt identity or misleading source-window-as-charge-month filter; the destination independently authorizes its existing query. Current read success is separate from upstream completeness and original-file availability, both explicitly unverified.

# Contract and async boundary

The unchanged v1 receipt producer remains separate from the new exact 19-key `/api/focus-collection-status` projection. Its fixed local reader accepts no query parameters or commands and retains existing HTTP host/origin/header/body, concurrency and response-bound guards. `collection.query` verifies permitted physical slices/hierarchy before the consumer emits bounded counts. Only actual allowed rows contribute delivery/version/period observations; zero-row foreign-only slices cannot expose presence. Periods merge by normalized source interval and all provider/period/provenance counts reconcile to row_count. The public snapshot hashes only the allowlisted projection and excludes the private whole-delivery snapshot, raw identities, source bodies and foreign metadata. No financial amount is projected.

Exact receipt auth_scope/permission_revision are reused inside the same guarded transaction. The frontend requires both observations' audience/member count/auth/permission pins to match. Every dependent read is request-bound, cancellable and generation-checked. Source availability failures remove stored-data values while preserving separately valid metadata; any 403 or authority mismatch removes receipt values, source values and selection together. Refresh is explicit; late results cannot restore older observations. Client parsing rejects unknown fields, inconsistent counts, unordered lists and equivalent UTC intervals spelled differently.

This is the existing trusted-local fixed-reader development boundary, not production multi-user authentication. No initializer, reset/reimport, migration, schema/permission change, source mutation, financial action, deployment or Graphify benchmark ran.

# Executed checks

| Check | Actual outcome |
| --- | --- |
| New isolated producer and actual TCP HTTP suite | PASS 84: 54 pure projection cases and 30 HTTP guard/dispatch cases |
| Existing receipt/connection/overview/commitment/Marketplace isolated suites | PASS 284; no retained mutation suite |
| Console unit suite | PASS 278, including 11 new status parser/transport cases |
| TypeScript/Vite build and final build verifier | PASS 438 artifact hashes and preserved licenses; existing large-chunk advisory remains |
| Final isolated browser suite | PASS 23 checks/10 captures, exact frontend revision recorded in evidence.json |
| Retained source/control/previous-result preservation | PASS 93 file hashes unchanged |
| Native owner-browser observation | PASS new KO S03 tab visibly shows 1121 rows/3 deliveries; existing S04 tab preserved |

Actual browser checks cover EN/KO/AR at 1440/390, scope-specific real counts, full viewport, no page overflow, neutral computed facts borders, at least 16px base text, review controls at least 16px/44px, physical LTR range glyph order, timestamp nowrap and logical numeric-header alignment. All six normal variants exercise detail-entry/return focus. Additional actual tab checks cover quarantine/completeness/extraction in EN390 and AR1440. The S04 handoff succeeds without injecting a period filter.

Isolated synthetic cases cover ordinary source failure/retry with metadata retained, closed-schema failure, valid and malformed403, mismatched permission pin with selected detail removed, exact0 versus unknown completeness, late old response after new receipt-filter results, mobile coverage zero-attestation meaning and metadata failure preventing the dependent read. These contexts create no owner acceptance. Browser page errors and non-GET requests were both 0. No customer browser profile is used by automated tests.

Host: macOS arm64, Node 26.7.0, prepared Python 3.11.7 and the installed Playwright/Chromium test environment. No application dependency/version upgrade occurred. Pinned cache-only Prettier 3.6.2 formatted the scoped source; it is not a product dependency. Other OS/browser combinations, full assistive-technology and native-language/practitioner acceptance remain unverified.

# Findings corrected and review boundaries

Preimplementation independent review blocked unbounded DTO details, private snapshot disclosure, zero-slice presence and authority-failure retention. The revised contract resolved those findings before implementation. Actual source review confirmed the producer/authority/async boundaries, then found missing client ordering and canonical UTC-alias checks; root added strict Unicode-codepoint/normalized-interval ordering and negative tests. Narrow re-review passed. The reviewer independently checked 93 preservation hashes and selected KO1440/AR390 wording against the earlier 23-check report. Child test tools could not bind sockets or start tsx IPC; root separately executed the actual 84/284/278 suites successfully. No failed child execution is reported as a pass.

The first rendered run reproduced global work-table min-width/row-height rules leaking into the new breakdowns. Leaf-only resets corrected actual overflow. Initial focus assertions were corrected to wait for observable focus, rather than DOM visibility alone. The initial baseline harness produced real design-reference images but failed its last canonical build-manifest read; it is not a passing suite. A later harness instrumentation omission produced an undefined measurement; it was fixed and rerun without weakening assertions.

Claude CLI actually read the official installed frontend-design skill, assigned baseline images and first-party reference evidence, and returned a concrete composition. Its first final-render comparison found orphan UTC text, ambiguous Arabic range presentation and numeric-header alignment. Root shortened visible timestamps while preserving full precision, isolated the range and aligned numeric headers; the final browser verifies actual glyph geometry and computed layout. The final CLI comparison passed for nine supplied frames, conditional on root computed neutral borders and physical range direction. Final browser range checks and a separate normal/detail/failure/authority rendered-edge, keyboard-focus and link/disclosure target check satisfied those conditions. The latter also captured a focused viewport. Non-blocking density, equivalent Arabic exclusive-bound wording and screenshot sticky-navigation observations remain candidate-review notes; no human language/layout acceptance is inferred. AI design/TW review is not owner FIX or native-language/practitioner approval.

The final independent scoped report review passed: 17 artifact hashes, 6 source/staged hashes, 93 preserved hashes and the final 23-check/10-capture report matched. The reviewer inspected the updated AR390 frame and confirmed prior source/TW boundaries. Root 84-suite execution is separately recorded in the curated JSON; the initial child socket-error log is not its source. Affected new links resolve. Four existing S03 references require authorized local companion inputs already documented by the repository; this change does not certify a standalone clone.

# Acceptance and operational handoff

S03 retains stable prior item IDs and adds stored rows, provenance, source windows, summary async and S04 navigation items. Changes are limited to this screen, its local transport exports, one new reader route/projection and scoped tests/docs. Existing sibling service routes retain their contracts; shared HTTP behavior was regression-checked. No accepted screen evidence was erased or carried to a changed revision. Parent HB-92 remains awaiting explicit owner screen acceptance.

Before removing the ticket worktree, restart its preview/reader processes from canonical main after authorized merge and confirm the same build/data. This avoids repeating HB-149's stale runtime references. Merge/read-back, ticket completion and machine release are recorded only after they occur.

This bounded current stored-data view does not complete HB-34 workers, HB-39 large-scale processing/performance budgets, live connectors, OCR, full upstream coverage, current original-byte verification or archived Parquet verification. Distinct planning/verification records are retained; no unrelated frozen design is declared superseded.
