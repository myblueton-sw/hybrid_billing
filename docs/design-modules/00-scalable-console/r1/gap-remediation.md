---
type: ImplementationPlan
title: HB-63 preview gap remediation
description: Records the bounded synthetic remediation of six reproduced preview gaps and its verification boundary.
status: draft
---

# Scope and ownership

[HB-63](https://linear.app/hybrid-billing/issue/HB-63/implement-multilingual-whole-product-scalable-screen-redesign), `feat/HB-63-scalable-screens`, machine mac_mini. The owner requested continuation after runtime gap findings on 2026-11-01. Root owns source, build, regression tests and documentation. Independent gap_review reviewed the design; actual change review and browser verification are required before acceptance.

The [gap audit](gap-audit.md) is retained as historical baseline evidence. This change addresses its six reproduced runtime observations. It does not claim to finish the other document/ERP, inquiry, full record-detail or large-file recovery journeys.

# Design and acceptance

- Integrate the existing onboarding drafts and provider profiles into the eight setup routes and reproducible build. Require prerequisite completion, explicit plan acknowledgment and simulated administrator read-back for new setup. Keep technical, coverage and financial readiness separate. Unsupported profiles and financial publication remain blocked.
- Integrate scoped AI illustration with period, currency, source record, progress, cancellation, partial/error recovery and a local review-task draft. Never call a model or send source data. Input changes remove obsolete results; navigation/language changes cancel pending work. Evidence returns to the same synthetic source record.
- Render report period, comparison period, amount basis, currency and grouping. Display the requested descriptor; aggregate values and export remain explicitly unavailable. No pretend query or monetary conversion.
- Replace the customer overview's global statistics with a customer-context summary. Restrict all visible navigation and disable the global account picker inside customer screens. This is a synthetic presentation boundary, not authentication or authorization enforcement.
- Pin approval preview to target/revision/currency. Missing comparison amounts block approval and issuance. Keep review confirmation and local rejection with a required reason separate; changing revision/currency resets decisions.
- Restore list query, page, view and selection on route return in the current account/persona. Preserve expiration and discard navigation caches on account changes. Refresh/filter changes invalidate selection and analysis context.

# Validation plan

Use a real local browser against the built artifact. Exercise onboarding normal/denied/unknown/partial and new-provider handoff; AI result/input invalidation, error/retry/cancel, language/scope changes and evidence identity; customer navigation; report descriptors; approval rejection/blocking; Back with filter/page/selection and expired snapshot. Run existing generic regression checks. Inspect EN/AR/KO desktop/mobile screenshots and asset/runtime errors.

# Results

PASS for this scoped synthetic remediation. `npm run build --prefix tools/design` emitted 882 files and passed the eight immutable-foundation checks. Root ran `verify.cjs`: 98 checks, 112 routes in three locales. Root ran `verify-journeys.cjs`: 71 checks, including thirty desktop/mobile localized layout cases, account-switch invalidation, Back-state isolation, report descriptor invalidation and currency/revision decision resets. Reports and screenshots are local generated evidence in `output/validation/HB-63` and `output/validation/HB-63-journeys`, not committed assets.

Independent gap_review actual-change review initially failed acknowledgment/downstream invalidation and implicit connector submission. Root fixed both and added browser regression assertions; source re-review passed. Tests initially exposed missing navigation waits, a same-document navigation incorrectly used as a fresh-session setup, and a customer-link assertion that included the skip-to-content anchor. These test issues were corrected; final assertions passed without bypassing customer routes or production boundaries.

Root visually inspected Korean mobile onboarding, Arabic desktop approval and English desktop report screenshots. No clipping or page overflow was observed in those samples. All thirty locale/size cases passed automated overflow checks. Native-language practitioner review, real provider/model/ERP calls, production permissions and financial operations remain unverified and outside this change. Report aggregation, positive approval/issuance, document/ERP journeys, full record detail and customer inquiries remain incomplete; unavailable evidence is not successful validation.
