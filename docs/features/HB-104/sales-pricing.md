---
type: ScreenDesign
title: Sales pricing policy review editor
description: Designs a source-backed B04 unit-price policy review with explicit quantity units and unconnected contract/activation boundaries.
status: draft
---

# Goal, ownership and evidence

HB-104 follows B03 in the approved service IA. Preserve B04 `/rates` and rates/formula/inheritance/adjustments/impact keys. The authorized B04 spec, PC-01 sequence, SP-01 selling-method evidence, exact numeric and H23 adjustment contracts govern. Root owns PM, model/storage, leaf integration, tests and documentation. Claude CLI ui-designer leads commercial composition; an isolated assigned implementer may own only the B04 UI files after independent finance/security/TW design reviews. No source or financial authority API/schema changes are planned.

The branch explicitly stacks on unmerged HB-103 PR80 including B03's scoped owner acceptance. Do not change B03 design/behavior or carry its FIX into new scopes. Commercial service hierarchy, full viewport, local Plex 16/14/20, neutral square controls, readable mobile actions and first-class KO/EN/AR are mandatory common rules, not optional feature criteria.

# Actual source and scope

Reuse the protected validated collected FOCUS query, server-side provider/month/account/search and bounded 25-row source page with its existing selection reference. Select exactly one canonical current-page source row as review evidence. The form is not a whole-contract/tier/allowance pool; changing source/page/row clears candidate, adjustment preview and session baseline after the unsaved-input guard. Actual Source `PricingQuantity` and `PricingUnit` are required for unit-price review; never substitute consumed quantity/unit or infer a unit conversion. Currency comes only from the selected source row. Missing quantity/unit stays unknown and blocks price calculation. Negative source quantities remain signed. Source service/provider are evidence, not a connected selling SKU or customer contract. No fabricated customer, contract or SKU identities.

Show actual collected ListUnitPrice/ContractedUnitPrice/price units and billed/effective row costs as source evidence, with null preserved. Do not preload them as customer prices or reproduce a provider catalog. Actual pricing authority, customer contract, eligibility and effective policy period remain unconnected. The review scope is the selected source charge interval; this is not a customer selling-policy activation interval. A specific charge-start month is required to save/run, without proration or recognition claims.

# Proposed executable review rule

This increment supports only an independently entered unit-price review. Name, unit price, positive price denominator, discount percentage and additional fee are explicit draft inputs. No default customer price/denominator/discount/fee is inferred; zero must be entered when intended. Unit price and additional fee are finite nonnegative decimal strings at scale<=18, bounded coefficient magnitude<10^38. Denominator is a positive finite decimal in the source pricing unit (same bounds), explicitly showing whether price is per 1/1000/1000000/custom units. Percentages use 0..100 with at most 4 fractional places. No arbitrary expression evaluator or user code.

Follow the supported PC-01 prefix: source quantity → base unit-price amount → review discount → additional fee. For integer source coefficients at scale 18: base = truncate_toward_zero(quantity_coefficient * price_coefficient / denominator_coefficient); discounted = truncate_toward_zero(base * (1000000-discount_coefficient)/1000000); candidate = discounted + fee_coefficient. Record quotient/remainder for every division, retain signed credits and reject output magnitude>=10^38. This is explicit `unit-price-source-precision-review-v1`, not an activated currency quantum/production rounding policy. Inexact division is visibly flagged as source-precision truncation, never silently labelled exact financial billing. Minimum/maximum, included allowances, tiers, FX, tax, commitment/savings share, target-margin and cost-plus policies remain unavailable without their full economic scope and approved contract. Never treat missing tax as zero or candidate as final sales.

Formula view presents the real ordered operands and results with units. Inheritance view shows one local review rule with organizational/customer precedence, overlapping periods and server revisions unconnected, not fabricated inheritance trees. Impact view compares immutable session previews only within identical company/source/selection/row/page/basis/currency/quantity-unit/arithmetic profile; source costs remain separate evidence, not approved margin or savings. Edits invalidate the current candidate; source/context failure clears candidate and baseline. Display/source filters never become a complete tier/allowance denominator.

# Adjustment review boundary

A candidate is required before target pre-tax amount, reason and evidence reference can produce a local target-minus-candidate delta. Show the original candidate, separate delta and requested target; never overwrite source/candidate or claim H23 financial application. Signed target amounts are allowed within the same exact bound. Changes to policy, target/reason/reference invalidate the adjustment preview. Issuance state, approved base revision, tax allocation and actor authority are unknown, so no apply/request/approve/issue command is offered. This is an arithmetic review on a session candidate, not an authoritative invoice adjustment.

# Draft, source and review state

Explicit browser save stores only bounded user-entered policy parameters, selected row hash and context hashes, never copied source amounts/quantities/account IDs/raw rows/search or credentials. User-entered draft prices/fees/target amounts are intentionally stored as review parameters and must be labelled as such. Current use is the explicit public/synthetic sample reader; warn not to enter real customer information. Browser storage is not authenticated private user storage or server policy.

Restore only after fresh validated authorization and identical source/selection/company/page/row/currency/unit/profile pins; reject unknown keys, duplicate/foreign references, stale/malformed/oversized envelopes. Write/readback failure stays visible and dirty. No offline fallback. Abort+generation+key masking covers source loading and asynchronous context hashing. Save/run/edit/tab/dirty paths expose exact screen-review state, including validation and storage failure. Owner FIX remains pending for B04; no assistant acknowledgement is created.

# Layout and references

Use existing compact title/review cluster and tab navigation. Rates: compact source selectors, bounded candidate table (actual service/quantity/unit/source costs), selected-row summary and first-viewport policy fields with Preview/Save; avoid tall caveat boxes. Formula: ordered calculation trace; inheritance: explicit missing contract/precedence state plus current local rule; adjustments: target/reason/reference form with independent delta; impact: baseline/current/delta table and source evidence disclosure. Mobile order is context → active fields → summary/actions → long source/details, with logical DOM order. Isolate money, percentages, dates, codes and units in RTL. Heading 20px must override the existing planning 18px cascade. Local per-screen review controls remain keyboard accessible.

Official external sources are consulted for unit-based pricing, explicit quantity/unit and form/table patterns. Adopt structure and semantic distinctions only, not vendor legal prices, assumed currency scales, dependencies, copied assets or decorative styling. Exact source URLs/adoption notes are completed after research. No unsupported provider capabilities are inferred.

# Validation plan

Independent design and actual-diff finance/security reviews, continuous TW source and rendered review, Claude CLI initial/final composition comparison, exact arithmetic/unit/signed/null/overflow and bounded generated tests, strict parameter storage and async/source failure tests. Actual browser tests cover source selection, policy editing, ordered preview, adjustment delta, session comparison, save/restore/readback failure, scope/dirty/late-response invalidation and all 5 views across KO/EN/AR desktop/mobile. Verify B03 accepted design has no semantic/layout change from leaf integration. Do not run retained-store reset fixtures or claim cross-OS/native/human financial approval.

# Independent design review and references

Financial and security design reviews conditionally passed the bounded review scope. Their requirements include fresh current-page row uniqueness/membership, source/row-aware async generation keys, all-stage and opposite-sign adjustment overflow checks, explicit signed quotient/remainders (never displayed as money), and separate policy/adjustment invalidation. TW approved proposed terms conditional on clearly labelled manual prices, same-currency pre-FX/pre-tax targets and local storage of entered financial parameters. Actual-diff and rendered review remain pending.

[Stripe pricing models](https://docs.stripe.com/products-prices/pricing-models) distinguishes product, price and quantity-based models. Adopt that structure; reject vendor defaults, actual prices, SDK/setup instructions or implicit currency quantum. [FOCUS Pricing Quantity](https://focus.finops.org/docs/specification/v1-0/columns/pricing-quantity/) distinguishes the price-related unit/quantity from consumption and permits signed corrections; adopt the distinction without certifying conformance or assuming every negative quantity is an approved refund. Existing Carbon form/table structural patterns inform bounded source selection and progressive detail; owner neutral/full-width rules prevail.

# B03 acceptance impact assessment

B03 remains byte-identical in its module source, labels, styles and calculations. The only shared App change adds the B04 leaf to the existing collected-source branch and new B04 tab labels; B01/B03 predicates, stable keys, controls and guards retain their behavior. Global console build fingerprints necessarily change when new B04 inputs are added; this does not rewrite B03’s immutable owner-confirmation record. The B03 accepted presentation must be checked by regression and any actual design effect treated as a new candidate rather than silently carrying acceptance.

# Stable implementation items

| ID | Location and behavior | Verification |
| --- | --- | --- |
| B04:scope:source-row | Source toolbar, canonical current-page selection and evidence identity; new source clears derived state. | Authorized reader and browser source/race/error scenarios. |
| B04:form:unit-price-rule | Explicit manual inputs, pricing unit and positive denominator; blank is not zero. | Numeric boundaries and editor invalidation. |
| B04:calculation:ordered-review | Readable captured operands followed by exact integer trace; no final billing claim. | Known and signed generated cases, rendered formula. |
| B04:adjustment:target-difference | Separate signed target-minus-candidate with required reason/reference. | Delta, stale candidate and overflow cases. |
| B04:comparison:session-baseline | Immutable same-context session baseline and parameters/amount differences. | Browser baseline and source-reset scenarios. |
| B04:action:browser-draft | Explicit entered-parameter save, fresh authorized restore, visible failure. | Strict schema/projection/readback and browser failure scenarios. |
| B04:state:source-unavailable | Loading, empty, denied and retry states without stale candidate or baseline. | Source failure and repeated-query regression. |

These IDs annotate design/review records; product tasks show business labels. All items remain pending owner B04 acceptance, regardless of implementation verification.
