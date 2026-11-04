---
type: ScreenDesign
title: FX and rounding policy review
description: Defines B05 source-cost conversion review with manual quote evidence and one explicit rational quantum operation.
status: draft
---

# Scope, owners and evidence

HB-105 preserves B05 `/exchange-rates` and policies/imports/comparison/approvals keys. The authorized B05 screen specification, PC-01/AR-06 in the billing inputs contract and six-mode numeric contract govern. Root owns PM, exact model/storage, App integration, tests and documentation; an isolated UI implementer owns the B05 leaf only. Independent finance/security and TW reviewers inspect design and actual changes. Claude CLI ui-designer leads composition and compares rendered output. This branch explicitly stacks on unmerged HB-104 PR81 at 211ab38, not fresh main. B03 accepted source and B04 implementation remain unchanged.

The actual protected bounded collected FOCUS reader supplies exactly one selected canonical current-page row, signed billed/effective source-cost amount, source currency and charge interval. It does not supply a customer selling amount, approved FX policy, official quote, prior-conversion history, contract precedence or approval authority. No fake quote, customer, feed, approval log or latest-rate fallback is introduced. Fresh authorization, row uniqueness/membership, async digest generation guards and strict restore precede use. Source/query/page/row/error clears candidate and immutable session baseline; edits invalidate candidate; same normalized query retries. Review identity includes source/query/snapshots before selection plus all inputs/results/errors/attempts. The actual API q parameter filters only the bounded account list, not source rows. Account search and canonical account selection are separate: choosing a returned account narrows rows; searching alone does not. Account pagination pins the same snapshot and clears row/detail-page state. No client-side whole-dataset search is substituted.

# Executable review profile

`manual-source-cost-fx-quantum-v1` is a deliberately local single-row cost-conversion exercise. Source amount/currency/basis are immutable evidence. Explicit user declaration of treating it as unconverted is mandatory; default unknown or already-converted blocks calculation. That declaration is not verified supplier conversion history. Target is selected explicitly from USD/EUR/KRW/JPY/AED/GBP, a bounded exercise allowlist, not a live supported-currency catalog. Source and target must differ. Source cost is not B04 selling price; customer-currency fixed fees and supplier-converted financial components are not imported into this rule.

Every parameter starts blank. Manual quote >0 and positive quotation unit retain direction: quotation-unit source-currency units = quote target-currency units. Never invert silently or normalize recurring rates into a truncated decimal. Manual provider/reference, real quote/review calendar dates on fixed UTC and integer maximum-age 0..36600 days are required. Quote must not be after declared review date or older than the entered limit. Dates need not be equal; these are user declarations, not certified market freshness, policy effective scope or holiday fallback. No host timezone/default current date, monthly average or automatic quote collection.

Manual markup 0..100 percent (at most 4 fractional places), nonnegative flat target-currency fee, positive target-currency quantum and one explicit six-mode choice are required; zero markup/fee must be entered. Scale 18 input coefficients have absolute magnitude<10^38. Source sign is preserved; positive markup increases negative magnitude and a positive fee can offset/sign-flip it. Negative markup/fees remain unsupported in this named exercise rather than being forbidden for every future financial policy.

Let c be source-cost coefficient, r quote coefficient, u positive quotation-unit coefficient, m markup coefficient in percent-millionths, f target-fee coefficient and q quantum coefficient. Exact target-coefficient rational is N/D with D=u*1000000 and N=c*r*(1000000+m)+f*D. No intermediate rounding: converted and marked-up stages remain exact rational trace operands. Bound every intermediate numerator/denominator to absolute magnitude<10^84, require D>0, then compute round(N/(D*q),mode)*q. Final coefficient absolute magnitude<10^38. The distinct display/comparison profile accepts rational numerator magnitude<10^130 and positive denominator<10^90, covering cross-products of two bounded candidates without expanding candidate financial limits. Formatting/subtraction reject malformed or out-of-display-bound values with a localized unavailable state; decimal expansion caps at 512 fractional digits and falls back to the complete exact fraction, never a truncated value. Modes: toward_zero, away_from_zero, floor, ceiling, half_up (ties away), half_even (ties to even quantum count), including signed ties. Preserve exact quotient/remainder and rational before-round amount; rounding difference = rounded coefficient minus exact pre-round rational, never raw remainder as money. No binary monetary arithmetic, assumed currency decimals or production rounding activation.

# Views, storage and interaction

Policies: compact source toolbar/selected amount identity, manual quote and rounding form, primary Preview/Save rail, bounded source selector after active tasks on mobile. Imports: actual entered manual evidence, dates and direction plus explicit unconnected feed/approval state; no pretend import command. Comparison: captured quote/markup/fee/quantum trace and immutable same-source-context, same-target-currency session baseline/current/change; incompatible targets suppress monetary subtraction. Approvals: explicit no connected authoritative history, no fabricated events or apply/request-approval action. Inheritance and financial activation remain unconnected.

Explicit browser save retains only bounded entered parameters, row/context hashes and named profile. It intentionally stores entered rates/fees, source descriptions and evidence references; label nonprivate public/synthetic exercise and do not enter real customer information. Never copy source amounts/rows/account IDs/raw search or credentials. Strict schema/enum/string bounds and arithmetic preflight; fresh authorized restore only, no offline fallback. Save/readback failure remains visible and dirty. No server API/schema/data writes or actual approval/financial command.

# Design and review constraints

Apply full viewport, existing canonical IA/logo, local Plex 16px body/form, 14px metadata, 20px section headings, neutral separators/square controls/no shadows or decorative colored edges. Scope every selector to B05, include sufficiently specific enabled/hover/disabled/textarea rules to avoid known B04 cascade issues. KO/EN/AR, numeric/ISO/date LTR inside RTL, keyboard focus/error links/region labels, mobile comparison cards and visible labelled horizontal scrolling for exact rational trace. Owner review control/stable IDs remain separate from product content; B05 FIX is pending and B04 is not implicitly accepted by continuing.

Independent design reviews conditionally passed with the explicit direction, declared dates/maximum age, intermediate bounds, single quantum operation, signed cases and same-target comparison requirements incorporated above. Proposed TW labels are 환율·반올림 정책 / FX and rounding policies; tabs 환산 규칙 / 환율 근거 / 환산 비교 / 승인 이력. Actual-diff and rendered reviews remain required.

# External evidence and validation

[ECB reference-rate framework](https://www.ecb.europa.eu/stats/pdf/exchange/Frameworkfortheeuroforeignexchangereferencerates.en.pdf) distinguishes informational reference quotes from transaction authority. Adopt the distinction only; no ECB rates, entitlement or feed is claimed. [Carbon data-table guidelines](https://www.carbondesignsystem.com/building-blocks/core/components/data-table/guidelines) supplies bounded selection/expandable evidence structure. Adopt progressive disclosure, not assets/dependencies/colors. Owner full-width/neutral rules prevail.

Verify all six signed modes/ties, per 100 quotation, nondecimal quantum 0.05, exact rational case where intermediate truncation changes outcome, null/zero/negative/stale/future/leap-day/sign/overflow and bounded generated quotient/error invariants. Browser QA covers actual source/guard/save/restore/error/race plus four views in three locales at desktop/mobile. B03/B04 regression and byte-identical module checks protect inherited scopes. No retained-store reset suite. Other OS/browser engines, native-language and human finance approval remain unverified.

# Stable implementation items

| ID | Location and behavior | Verification |
| --- | --- | --- |
| B05:scope:source-cost | Authorized selected row, signed basis/currency and source identity. | Membership, failure and stale-response cases. |
| B05:form:manual-quote | Explicit quote/unit/direction/evidence/date/declaration and markup/fee. | Validity, missing and age tests. |
| B05:calculation:quantum | Exact rational then one selected quantum operation. | Six-mode signed oracle and generated cases. |
| B05:comparison:session-baseline | Immutable same-context/target monetary comparison. | Changed-target and edit cases. |
| B05:action:browser-draft | Explicit bounded financial-parameter save and fresh restore. | Strict schema/enum/readback failure. |
| B05:state:authority-unconnected | Manual evidence versus official feed/history/approval. | Copy and rendered review. |
| B05:state:source-unavailable | Loading/empty/denied/retry without stale results. | Browser lifecycle regression. |

# Implementation review reconciliation

The standalone UI has replaced the generic fake B05 records and actions. Actual source/parameter projection, rational arithmetic and explicit unavailable-display bounds passed independent finance/security reviews and scoped tests. Account-list search semantics were confirmed from the actual protected reader and exposed separately from canonical account selection. Source-card cell auto-height, full selected declaration text, LTR exact-trace origin, changed/unchanged parameter labels and the visible full-width mode legend reconcile the actual rendered findings. Verification scopes and remaining limits are recorded in the companion verification report. These are implementation/review outcomes, not owner B05 FIX or a replacement of the deferred authoritative FX/contract/approval requirements.
