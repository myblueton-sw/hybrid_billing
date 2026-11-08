---
type: VerificationReport
title: Commitment costs and benefits evidence verification
description: Records B06 protected source projection, exact cost distinctions and actual multilingual workflow verification.
status: draft
---

# Scope and ownership

HB-106 stacks on unmerged HB-105 PR82 at `3dca894`; remote main was read separately, with no merge or fresh-main claim. Root integrates frontend API/types/App/tests/evidence, isolated implementer wrote the new read-only projection/HTTP route/tests, a separate implementer wrote B06 UI. Independent finance/security/TW reviews inspect their actual scopes, and Claude CLI ui-designer provides composition and rendered comparison. B03 accepted source and B04/B05 modules remain unchanged. B06 owner FIX is pending.

# Source discovery and actual protected read

Read-only discovery verified pinned public/synthetic source-artifact hashes and north assignments: 1121 rows, 4 nonnull commitment-ID rows, 2 scoped AWS groups, Used classification only. Literal `NULL` is a pinned 1.0 null token, not an ID; a naive count would produce 993 false rows. The existing overview/local-release endpoints did not expose the needed fields, motivating a dedicated protected read-only endpoint rather than fake UI records or ingestion/schema changes.

The fresh `/api/focus-commitments?provider=AWS&period=all` protected database read then confirmed authorized 1121 / scope 942 / identified 4 / unidentified 938 rows and 2 groups (3+1 rows). All 4 actual mapped source rows are Used; both Billed/Effective group coefficients are 0 with 3/1 present and 0 missing. These are genuine observed zeros, not invented sample amounts. No Unused rows were observed; empty-bucket cost remains unknown/null. Client validation accepted the actual protected response. Source-company/account references are authorization scope, not legal ownership.

The first actual read returned 503 because this fresh worktree lacked the authorized local authority/control-state binding. Root traced the safe FileNotFound path, reused the existing verified local environment/state bindings and common coordinator lock, and retried successfully. No source-data reset, import, schema initialization or financial operation occurred. Generated evidence, runtime links, credentials and source snapshots remain outside Git.

# Independent backend and client checks

Independent design reviews conditionally passed the dedicated endpoint, authorization-transaction delivery, route-specific guards, pinned mapping, complete-before-page aggregation and no inferred financial conclusions. Actual finance/security reviews then passed after root fixed null-token precedence: pinned empty-string token is source_null; non-token whitespace is blank. Counters were renamed version_mapped_rows/version_unmapped_rows so mapping does not imply source/provider verification. Both reviewers independently executed the three exact isolated backend test files: 58 PASS. Existing overview projection/HTTP regression tests are included; no retained-store fixture suite ran.

Root frontend tests cover bounded query/schema/pages/group-detail membership, explicit provider/account/version constraints, page-to-full-total consistency with signed sums, known-zero/null, canonical group collisions, metadata/bucket invariants and timezone-explicit real-calendar timestamps. Full frontend: 147 tests PASS. Financial display source review confirms full-selection versus paged values, no-row/null/known-zero distinction, no inferred savings/utilization or completed reconciliation, unknown ownership/burden/prepay and qualified version/interval evidence.

Independent review found a separate asynchronous review-hash scope defect: generic pending/error scopes could repeat while loaded data differed. Root removed that extra hash stage and forwards complete request/evidence/error state to the existing common ReviewPanel digest/gate. Exact request/snapshot/group/page/data identity is retained; persisted variants contain only curated hash references and summary state, not raw data/query. Independent scoped re-review PASS. No owner acknowledgment was created.

# Actual browser and rendered verification

Core actual-data scenarios passed: canonical group selection, separate observed-zero costs, Used/Unused evidence, benefits/reconciliation authority boundaries, close/Escape focus return, repeated normalized account query, unmatched account search without group filtering, canonical account selection/clear,403 removal/new authorized query,503 same-query retry, late group-A digest after group-B selection, and VMware source rows without fabricated cloud commitments. The first two runners used a wrong account-disclosure selector and expected a retry action for denied access; selectors were corrected to actual controls, and denial requires a fresh query rather than a same-query retry.

Rendered computed checks then caught 14px group/pager controls and inherited nowrap cells despite the common 16px control requirement. Root corrected B06-only font sizes and table-cell specificity/normal wrapping. Final capture passed all 24 combinations: four B06 views × KO/EN/AR × 1440/390 px. Checks cover 16px body/controls, 20px section headings, local Plex font families, RTL direction, square neutral surfaces, mobile cell wrapping, no document overflow and no page errors. Actual loaded font faces were separately confirmed for all three locales. Existing B05 workflow and its 24 rendered variants also passed; B03/B04/B05 module directories have no diff against the recorded stack base.

# Verification limits

The new route retains exact trusted-local origin/header/fixed-actor/semaphore/no-store/response-size boundaries and collected-query authorization transaction. It exposes only eight declared raw fields; no arbitrary raw JSON. Projection bounds do not expand the stricter existing reader admission envelope. This local reader is not a production throughput or provider-event/lifetime conformance certification. Other OS/browser engines, native language, full screen-reader coverage, human finance, customer pass-through/unused responsibility/prepay recovery, purchase/optimization/financial activation remain unverified or unconnected. Installation/operation design uses existing portable Python/Node/browser interfaces; the actual tested host is macOS arm64.

# Final independent reviews and artifact identity

Claude CLI ui-designer inspected ten initial final captures across desktop/mobile and languages and returned PASS without blockers. Root then clarified that coverage declarations concern all collected sources, localized the unverified operator-declaration label, collapsed secondary reconciliation metadata and aligned monetary headers. The scoped final comparison inspected KO desktop and EN/AR mobile reconciliation captures and returned PASS. Independent AI TW actual-render review passed its selected cross-language captures and the final coverage-copy follow-up. Finance review passed the source semantics, including that global coverage does not certify the selected provider/month. These are independent AI reviews, not human finance approval, native-language acceptance or owner FIX.

Final production build PASS. Build verification PASS: 438 artifact hashes and seven license notices; frontend fingerprint `26ba1966ceefbfd1b0ad43afceaabb20bb9ef5332ce49cadf87960c92ff0b60c`. The existing large-chunk advisory remains. Actual tested environment: macOS arm64, Node 26.7.0, npm 11.19.0 and Chromium headless build 1243.

Six actual read-only HTTP boundary checks passed: missing trusted header (403), unknown query (400), stale snapshot (409), foreign group (400), off-page group (400), and row offset without group (400). Denials exposed no source groups or rows. Full frontend suite: 147 PASS. Isolated backend suite: 58 PASS, including existing overview projection/HTTP regression. Reproduction uses `npm test` and `npm run build` in `apps/console`, `node tools/verify-build.mjs` there, and `pytest -q tests/focus_commitments/test_commitments.py tests/focus_overview/test_projection.py tests/focus_overview/test_http.py` from the repository root with the verified Python environment. Browser reproduction uses `apps/console/tools/commitment-benefits-qa.mjs` with explicit local URL, output directory, Playwright module and Chromium executable arguments.

Raw local captures and CLI review output are temporary verification artifacts outside Git, not permanent published evidence. This report preserves the non-sensitive results. Purchase, approval, financial reconciliation completion, production capacity and owner acceptance remain pending or unconnected.
