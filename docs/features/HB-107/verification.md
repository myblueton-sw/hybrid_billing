---
type: VerificationReport
title: Marketplace source evidence screen verification
description: Records B07 protected source reads, independent reviews, exact financial distinctions and rendered workflow checks.
status: draft
---

# Scope and source identity

HB-107 stacks on open B06 PR83 at `2d79d9bb5ac015f714f7f0e3c2b6f37d0bfe576c`, not fresh main. Root integrates the frontend and evidence; an isolated backend implementer wrote the new read-only projection, fields, HTTP dispatch and tests. Separate finance/TW and security/contract reviewers inspected design and actual code. Claude CLI ui-designer provided initial composition and final rendered comparison. B03 accepted source and B04/B05/B06 modules remain unchanged. Owner FIX, merge and deployment are not claimed.

Read-only discovery verified the public/synthetic six-provider source artifact SHA-256 `986a297e5476f4747a2921f690745309256ffc5d117cf487ff8a1c2d7286b9c2`. The 1123-row artifact includes 1121 rows within the north assignment. A fresh protected database read through `/api/focus-marketplace?provider=&period=all` confirmed 1121 authorized/selected rows: one candidate, 1005 with no issuer/publisher difference observed, and 115 with insufficient evidence. Unknown rows have source-null Google Cloud invoice issuer; no fake Marketplace records were added.

The sole candidate is AWS Red Hat software with observed BilledCost USD 0.342 and EffectiveCost USD 0, present count one and missing count zero for both. These are separate exact source costs, not a saving or marketplace fee. Source references and SKU do not establish a transaction, offer, legal role or customer invoice. Seller revenue, fees, refunds, taxes, withheld balance, net settlement, disbursement, benefit eligibility and original-credit mapping remain unconnected or unverified.

The prepared local authority/control-state and dependency bindings were reused without reading/logging credentials. No store reset, ingestion, schema initialization, financial command or owner acknowledgment occurred. The first frontend diagnostic hit an old preview proxy configuration and was retried after restarting the task-owned preview; it was not an API connection success. Fresh protected reads subsequently succeeded.

# Independent source review and tests

Finance/TW design conditions removed legal-party inference from equal names, added separate refund/tax evidence and preserved the canonical `credits` route key. Contract design required tenant/batch/version mapping evidence, source-only candidate semantics, pinned current-page selection, bounded allowlist and authorization-transaction delivery. Actual source reviews passed these scopes.

The backend writer initially found a delivery-version snapshot regression in the isolated suite and directly pinned sorted tenant/batch/version association in the derived snapshot. Final backend suite: 98 PASS (40 Marketplace plus 58 existing overview/B06 tests). The contract reviewer independently reran all 98; finance independently ran the 64 Marketplace/overview tests. No retained-state reset suite ran. Frontend suite: 155 PASS, including eight new API tests; the contract reviewer separately ran those eight. Tests cover exact known-zero/null/signed sums, classification and page invariants, version/null tokens, source allowlist, candidate and selected-detail membership, query bounds, provider/account/currency scope, real calendar/timezone timestamps and false financial authority.

Independent async/review-scope source review passed: exact request keys clear old data immediately; abort and monotonic sequence gate late success/error responses. Review state includes request, view, full evidence, selection, errors and attempts and directly uses the existing common digest/generation gates. Persisted review variants retain curated references, not raw source data. No B07 code writes owner acceptance or a financial revision.

# Actual browser and HTTP verification

Actual-data browser workflow PASS: candidate source selection; detail close/Escape and return focus; all four views; seven separately unconnected seller concepts; contractual benefit/credit boundaries; normalized repeated account search; unmatched account search preserving candidate totals; account pagination and canonical selection/clear; 403 removing source evidence and requiring a new query; 503 same-query retry; 409 stale selection removal/new query; late source-A response after source-B selection; and VMware source rows without fabricated marketplace candidates. The actual common owner-review storage key remained absent in the isolated browser.

The runner originally asserted immediately after a tab click; it was corrected to wait for the actual view before checking content. This was a runner timing error, not a failed product state. All 24 combinations of four views × KO/EN/AR × 1440/390 px passed computed checks for 16px body/control/table, 20px headings, local Plex family, RTL direction, neutral borders, zero radii/shadows, inline mobile selected detail, no document overflow and no page errors. Actual loaded FontFaceSet entries and font checks separately passed for Plex Sans, KR and Arabic.

Seven actual read-only HTTP boundary checks passed with sanitized error-only bodies: missing trusted header (403), unknown query (400), stale snapshot (409), foreign row (400), off-page row (400), selected row without snapshot (409) and hostile Origin (403). Existing B06 workflow and its 24 locale/layout/view captures passed through the shared app and updated reader. The B03/B04/B05/B06 leaf directories have no diff against the stack base.

# Render comparison and corrections

Initial Claude CLI composition inspected the generic B07 render and selected B06 precedents. The final comparison inspected KO desktop and EN/AR mobile across all four views. It passed the commercial layout and financial boundaries but found two blockers: selected mobile articles inherited bold text/gray fill into long detail content, and the Arabic source-status version run did not isolate `FOCUS` together with its version. Root limited selection styling, restored normal-weight page-background mobile detail, and split counts into separately labeled pairs with the entire version run isolated LTR. These are meaningful rendered findings, not a build-only acceptance.

Root also retained Korean money-header words, grouped the table date, spaced the contract-counterpart heading and named the close action explicitly. Source zero displays as USD 0.00 through the existing minimum-two-decimal exact formatter; this is presentation padding, not invented precision, missing data or monetary rounding. Final scoped Claude CLI re-review PASS: inspected five final captures (EN mobile credits/mapping, AR mobile buyer/mapping and KO desktop buyer) plus the relevant source. Both blockers and the scoped presentation corrections passed; prior unchanged financial/layout findings retain their recorded PASS. Finance/TW final follow-up independently passed AR mobile mapping, KO desktop buyer and EN mobile credits. Final complete 24-variant workflow/render run passed, including new detail paragraph/label weight-400 and whole-version LTR isolation regression checks.

# Final build identity

Production build PASS; manifest verification PASS with 438 artifact hashes and seven bundled license notices. Final frontend fingerprint: `b2479881c6977ab0fb81834b00b2ceeeb714ab031ac5ee4909c11904796e21a8`. Source/tests remain below 1500 physical lines, and the staged-scope whitespace check passed. Presentation-only corrections do not change API or financial arithmetic; their build and rendered regressions passed without repeating unrelated retained-store tests.

# Reproduction and limits

Run `npm test`, `npm run build` and `node tools/verify-build.mjs` in `apps/console`; run `python -m pytest -q tests/focus_marketplace/test_marketplace.py tests/focus_overview/test_projection.py tests/focus_overview/test_http.py tests/focus_commitments/test_commitments.py` from the repository root in the prepared backend test environment. Browser runner: `apps/console/tools/marketplace-qa.mjs`, accepting explicit URL, output, installed Playwright module and Chromium executable arguments. Existing reader startup and local authorization/configuration prerequisites still apply.

Actual host: macOS arm64, Node 26.7.0, npm 11.19.0 and Chromium headless build 1243. The existing large-chunk build advisory remains. Portable Python/Node/browser interfaces are design targets; Windows/Linux, other engines, production capacity, native-language/human finance, full screen-reader coverage and provider/legal/settlement/eligibility integrations remain unverified. Shared mobile tab clipping and long evidence pages remain nonblocking design follow-ups outside the leaf scope. Raw captures/CLI outputs are temporary local artifacts outside Git, not permanently published evidence. Owner acceptance is pending.
