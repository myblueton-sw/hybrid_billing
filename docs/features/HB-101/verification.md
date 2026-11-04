---
type: VerificationReport
title: B01 infrastructure cost composition verification
description: Records actual source, financial and terminology boundaries for the browser review-draft increment.
status: draft
---

# Delivered scope

HB-101 replaces B01's generic example renderer with an editable infrastructure cost composition candidate, preserving B01, `/cost-pools`, four tab keys and owner-only screen review. Menu/title is now `Infrastructure cost composition` / `인프라 원가 구성` with an Arabic candidate. Source/month/basis selectors, bounded source selection, categories, whole-row inclusion references, exact currency-separated previews and explicit browser-only save/restore are connected. The candidate depends on unmerged HB-143/PR78; this is not a fresh-main or merged implementation.

The ongoing [terminology rule](../../../.agents/rules/product-terminology.md) and [product glossary](../../ui/terminology/product-glossary.md) require planning/data-contract checks and a separate TW review of new/renamed financial screens, before and after rendering. U01/S04 display terminology now distinguishes charge-start month, charge interval, account scopes, billed/effective costs and source/selection references. Canonical FOCUS names remain in mapped fields/help.

# Data and financial limits

The actual protected collected reader returns three synthetic VMware rows, two account scopes and USD 2.50 billed/effective source cost for 2024-09. It is collected FOCUS sample evidence, not a complete infrastructure cost ledger or customer dataset. The default source is VMware; other authorized sources remain selectable. No values are fabricated for legal owner, assets, lease, depreciation, facilities, power, licenses, products or approvals.

A draft selects at most the current 25-row page. Named save requires a specific charge-start month and valid selected identities. Whole-row declared inclusion requires a reference and equal source intervals; cycle/self/foreign/duplicate/mixed-currency cases are rejected. Gross, declared-included reference subtotal and remaining review subtotal are separate. Signed credits, exact18 precision, null and partial states are preserved; currencies never combine. Charge-start attribution without proration does not establish monthly accrual, depreciation or approved cost eligibility. Products/approvals explicitly remain unconnected.

Storage contains projected draft metadata and hashed row identities/context, not amounts, account IDs, raw search or source JSON. Fresh validated authorization precedes every restore; obsolete or failed queries expose no cached draft fallback. Browser storage is not private per-user storage, a server policy or financial approval. No backend/API/authentication/database schema or retained-store mutation was introduced.

# Actual verification

Host: macOS arm64, Node26.7.0, npm11.19.0, Chromium headless shell153.0.8010.12. Installation/operation use portable Node/browser interfaces and local fonts; Windows/Linux and other engines remain UNVERIFIED.

- `npm test` in `apps/console`: **98 PASS**, including ten B01 exact-money, graph, malformed storage, stale/foreign context, metadata projection and failed readback tests.
- `npm run build` and `node tools/verify-build.mjs`: PASS; 438 final artifact hashes and seven preserved license notices. Existing chunk-size advisory remains.
- `tools/infrastructure-cost-qa.mjs` against the retained protected source: source selection, category editing, inclusion preview/cycle error, browser save/restore, failed storage, dirty scope keep/discard, unavailable source clearing, all four views, obsolete actual-response replay and six EN/KO/AR1440/390 variants. No owner review acknowledgement was saved.
- `tools/collected-source-qa.mjs`: PASS against the same server; six locale/layout variants, retained1121rows/78account scopes, source detail/paging/usage/history, pinned U01 roundtrip and malformed/stale/foreign clearing. No backend reset fixtures were run.
- Computed six-variant B01 inspection: body16px, neutral `--hb-line-control` source-notice border, radius0, shadow none, local same-origin font resources and no document horizontal overflow. This is scoped inspection, not an audit of every legacy screen.

The browser suite accepts external Playwright-module/Chromium options; these are development-host paths, not product installation requirements. Generated screenshots/results remain local and are not source deliverables. Only disposable synthetic browser test metadata was saved.

# Independent review and corrections

Separate finance and security design reviews were conditional, then actual-diff reviews found mixed-currency gross aggregation and whole-object save serialization. Root suppressed mixed-currency aggregates, projected allowed save fields, enforced context/data consistency and added regression tests. Both independent re-reviews returned scoped PASS; each independently ran the ten model tests. Financial approval and human practitioner fitness are not inferred.

TW actual-copy review caught an Arabic charge-interval/month meaning regression and stale snapshot wording. Root corrected both and added in-context identity/reference help. TW inspected six rendered variants, initially failed overlapping mobile table cells, then returned scoped PASS after root removed inherited fixed44px mobile cell height. Arabic native-language acceptance remains pending.

Claude CLI ui-designer supplied the composition design and inspected six actual captures. The first final visual comparison passed EN/KO desktop/mobile and AR desktop but failed AR mobile label/value alignment. Root aligned isolated LTR grid values through logical placement and narrowed the desktop edit button. The scoped B-1 re-review passed. That review additionally found the included-target identifier truncated in its select; root exposed the full canonical target ID below the select. The separate CLI B-2 image/code re-review returned PASS, with horizontal editor-scroll identity and numeric header alignment remaining nonblocking polish. Root's native Chrome read-back confirmed the actual B01 collected-source screen; no FIX was applied.

Security integration review additionally caught clearing the navigation guard when clicking an already-active B01 tab. Root now returns before changing that guard and added a dirty active-tab-to-source navigation regression. Independent source re-review returned scoped PASS for this guard fix.

# Remaining integration

Authoritative infrastructure records, policy ownership, included-product contracts, approval workflow, server persistence and production identity remain unavailable. This increment supplies a functional source-backed review editor for integration; it does not complete the full B01 operational service. Owner screen acceptance, merge and deployment remain pending. Earlier U01 controller/billing fitness findings are not closed by this increment.

Final build fingerprint: `fe9af7c5107e`. The reviewed visual captures preceded only formatting of the two B01 label tuples; those final formatting changes have no rendering or behavior effect. Financial/source reviews do not certify native-language, human practitioner or owner acceptance. The repository staging inspection excluded node_modules, generated builds/captures, local settings, retained data and unrelated files.
