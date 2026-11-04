---
type: VerificationReport
title: Allocation policy review verification
description: Records B03 commercial screen behavior, source arithmetic and independent review boundaries.
status: draft
---

# Scope and dependency

HB-103 replaces the generic B03 example renderer at `/allocations` with an actual collected-source allocation policy review editor. The stable four tab keys now display Allocation policy, Allocation basis, Allocation preview and Result comparison. It depends on unmerged HB-101/PR79; neither current main integration nor full B03 operational completion is claimed.

The full server aggregate supplies the source basis, never a detail/account page. Fixed manually entered percentages use the complete actual synthetic tenant-access recipient manifest and a fixed100-percent denominator. Tenant access is not legal organization/department/customer-contract ownership. Source costs are not approved infrastructure cost or sales. Usage/reservation metrics, idle responsibility, production precision/rounding policy, server policy approval and result history remain unconnected.

# Arithmetic and state boundaries

The provisional `source-precision-review-v1` profile uses signed scale18 integer coefficients and strict percentage strings with four fractional places at most. It does not activate `largest_remainder_v1` or a currency quantum. Actual recipient sum + unallocated + precision remainder equals the collected source basis. Missing values or no specific month/currency block the calculation. Complete zero is valid. A negative source credit remains signed. No automatic residue redistribution or unknown-idle-as-zero is permitted.

More than100 recipients blocks browser review instead of truncating the manifest; 25-row display pages/search cannot alter membership, weights or denominator. This is an explicit browser-review boundary, not a production organization-size limit. Restricted account/search/offset entries block until the user explicitly opens the whole-source scope; old selection references are removed with the restriction. Unrestricted pins remain validated.

Fresh validated authorization precedes context preparation and metadata-only restore. Source/company/provider/month/basis/currency/target/profile pins bind drafts and results. Edits invalidate candidate results. A baseline is retained by value within the current session only; source/context failure clears both results and draft. Browser storage is not authenticated per-user storage; no amounts, account IDs, raw source/query or real customer data are persisted. Owner review remains pending.

# Independent design and source review

Separate financial and security design reviews conditionally passed these bounded review semantics. Both later independently executed all11 model/storage tests and returned scoped PASS for full denominator, signed conservation, source/target restrictions and strict metadata storage. Tests include250 deterministic generated signed/permutation cases plus independent ±1, less-than100, zero, partial/null, wrong currency, duplicate/over100 target and malformed/stale storage cases.

TW design, actual-copy and rendered re-reviews passed the preferred labels and financial/source/recipient distinctions. Actual reviewed captures include policy editors in all three locales at1440/390, EN preview/comparison and KO/AR preview/comparison at390 plus source basis at1440. This is selected-copy/readability review, not native-language acceptance. Claude CLI ui-designer successfully inspected the current generic B03 render, B01 visual reference, authorized B03 spec, active design guides and installed frontend-design skill; its compact source strip and first-viewport editor layout are implemented subject to actual render comparison. This is AI specialist review, not human practitioner/native-language/owner approval.

# Verification status

- `npm test --prefix apps/console`: **110 PASS**, including11 model/storage cases (250 deterministic generated conservation/permutation inputs) and the exact signed-format regression. The financial reviewer independently ran the signed-format case as well.
- Production TypeScript/Vite build and `node tools/verify-build.mjs`: PASS;438 final artifact hashes and seven license notices. Existing chunk-size advisory remains.
- `tools/allocation-policies-qa.mjs`: PASS against actual retained protected source. Tests cover fixed50/25 and50/50 results from USD2.50 full aggregate, immutable comparison baseline, hidden display recipients preserving denominator/result, browser metadata save/restore, failure visibility, keep/discard scope guard, restricted-entry reset and slow obsolete actual-response replay. No owner acknowledgement was saved.
- Four views × EN/KO/AR ×1440/390:24 captures and document overflow checks. Computed radio alignment/16px, source styling and local fonts are inspected separately; final h2 sizing is20px under an explicit scoped override of the old planning18px rule.
- Existing B01 `tools/infrastructure-cost-qa.mjs`: PASS after the shared leaf integration change, including six variants and original source/storage/guard cases. No backend reset or schema/data mutation tests were run.

# Visual findings and resolution

The mandated Claude CLI first final comparison failed four observed issues: stacked radio/label, mobile summary after the entire recipient table, inconsistent delta signs/current labels, and unisolated Arabic100% prose. Root used a16px row control with14px/600 legend, moved policy fields → summary/actions → target editor into matching DOM/grid order at smaller widths, added exact signed formatting with explicit increase/decrease/no-change and paired baseline/current labels, and isolated percentage fragments. The separate CLI re-review inspected all six policy variants plus EN preview/comparison and returned scoped PASS for all four blockers with no new blocking regression. Finance and TW separately re-reviewed the precision/sign and wording changes.

Root’s computed check found legacy planning h2 styling overriding the nominal20px screen declaration. The final scoped CSS overrides that rule; the browser suite checks actual20px headings. The final screen also supports Escape closing its focused open disclosure and error-summary focus into the invalid recipient input. These final small changes are root-verified, not represented as a separate CLI inspection.

Nonblocking polish remains: repeated mobile row metadata, scrollable tab discoverability, coverage count unit, explicit operators in the preview summary, and a few redundant preview/status lines. These do not imply the full commercial service is production-ready. All owner FIX scopes remain pending.

Host observed: macOS arm64, Node26.7.0. Portable Node/browser/local-asset design does not certify Windows/Linux or other browser engines. Source/visual test evidence remains local and excluded from Git. No FIX, merge or deployment is inferred.

# Final read-back

Final build fingerprint: `c7565d15d374`. The final complete browser run passed all24 captures and the focused disclosure Escape/error-summary-to-recipient input checks. Six-variant computed inspection confirms16px body,20px headings, local same-origin font requests, neutral pool-strip border `rgb(220,225,229)`, radius0, shadow none and no document overflow. Final review-state metadata includes attempted validation, visible validation codes and notice state so error/save-failure views cannot inherit a normal-view confirmation scope. A separate source re-review passed this metadata/keyboard change.

The CLI's four-blocker visual PASS preceded only the final scoped20px CSS specificity and review/keyboard metadata correction. These final changes are root-verified with the complete browser run and computed checks, not claimed as a further CLI approval. B01 retains its layout/copy/arithmetic; its shared App leaf integration and changed candidate source fingerprint were assessed with the B01 regression. Other leaf screens were not redesigned or comprehensively re-audited.

No owner acknowledgement, authoritative policy write, backend mutation, merge or deployment was performed. Browser foreground handoff was not completed; the local direct screen URL remains available for owner review.
