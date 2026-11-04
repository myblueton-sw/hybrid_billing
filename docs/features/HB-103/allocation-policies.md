---
type: ScreenDesign
title: Source-backed allocation policy review
description: Designs B03 as a full-source allocation review editor with explicit sample and approval boundaries.
status: draft
---

# Goal and evidence

HB-103 follows B01 in the actual service IA. Read the authorized B03 spec, v1.16 ui-design121 and the complete-pool numeric contract in the companion workspace. Root owns PM, model/storage, integration, tests and documentation; visual composition uses the mandated Claude CLI ui-designer and implementation may be assigned only after design reviews. Independent finance/security and TW roles review design and actual changes. Branch stacks on unmerged HB-101 PR79, not current main. B03 and `/allocations` plus tab keys policies/pools/simulation/comparison remain stable.

Commercial service hierarchy, typography, whole-viewport layout, responsive editing, precise data labels and practical task completion are mandatory common requirements. This is an actual policy-review form and results view, not a screen checklist or generic example table. Owner review remains pending.

# Actual source and limits

Reuse the protected validated collected FOCUS reader without backend/schema/auth changes. The pool is all authorized rows for the explicitly selected provider (or the entire authorized cross-provider source scope) and charge-start month, grouped by actual source currency and billed/effective basis. Reject nonempty account/search or nonzero page offsets as pool definitions, with an explicit Open whole source scope action removing them and their selection reference. Never silently shrink or enlarge a shared denominator. Provider/month changes create a new source-review pool and reset drafts/results after unsaved-input protection. The full selected currency aggregate comes from server `amounts`, never the 25-row detail page or account page. Current source totals are sample evidence, not approved infrastructure cost or sales.

Use actual authorized `data.tenants` as synthetic review recipients, labelled tenant access scopes, never legal entities or verified organization/contract bindings. No invented organizations or financial ownership. Targets are hashed canonical tenant keys mapped back only through each fresh authorized response. The browser review editor accepts at most100 actual authorized targets, displayed25 per page. More than100 blocks review calculation rather than truncating the manifest; production-scale server allocation remains unconnected. Every target is included in calculation even if a local display search hides it. The source/selection reference and target-set hash pin every preview, together with company, provider, month, cost basis, currency and the source-precision-review-v1 arithmetic-profile identifier. Duplicate canonical tenant keys/hash identities reject the manifest. A real organizational directory/contract membership/approved pool manifest remains unconnected.

# Review calculation proposal

Only manually entered fixed percentages are executable in this increment. Use percentage strings with at most four fractional digits, 0 to100 inclusive; full denominator is always100 percent, never the sum of visible/selected targets. Blank weights are missing, not silently zero. Require all target weights explicitly entered and at least one positive; sum must not exceed100. Usage/reservation methods remain unavailable because there is no verified utilization/reservation denominator.

Source coefficients remain signed exact18 integers. Require one existing actual currency with present>0 (complete zero is valid), specific charge-start month and complete amounts (no missing rows) before simulation. Present collected partial/null subtotals honestly, but block allocation of unknown totals. For each target: truncate toward zero `source_coefficient * percent_coefficient / 1000000` at source scale18. Do not invent a production rounding/currency quantum. This is an explicit provisional source-precision arithmetic profile, not an activated accounting policy. Let intended assigned amount be truncate(source * total percentage /100); unassigned amount = source - intended assigned; precision remainder = intended assigned - sum(target amounts). Display all three separately with signed credits preserved. Their exact sum equals the selected source basis. Do not distribute rounding residue automatically. Idle cost is unknown, never zero; arithmetic conservation is not approved allocation-policy validation.

Explicit Run preview captures the current configuration; any edit invalidates that candidate result. An explicit Keep comparison baseline retains an immutable preview only in this browser session. Compare only within identical fresh source/company/provider/month/basis/currency/target manifest. Scope change or source failure clears candidate and baseline. No server result history, approval request, authoritative policy revision or financial-write action is fabricated.

# Browser draft boundary

An explicit browser-only save stores a strict versioned bounded envelope of name, canonical target hashes, percentage strings and context reference hashes, never source amounts, account IDs, raw query text or raw rows. Restore requires a fresh successful validated authorized query, identical source/selection/company/currency/target set and strict schema/unique target validation. Failed or obsolete source queries clear all protected draft/result/baseline content; there is no offline fallback. Browser storage is not authenticated per-user storage; this is limited to the existing explicit public/synthetic reader. Save/readback failure stays visible and dirty. Scope/refresh/navigation preserve the existing keep/discard guard. No owner FIX is created.

# Layout and references

The [FinOps shared costs guidance](https://www.finops.org/wg/identifying-shared-costs/) distinguishes fixed and metric-based proportions. Adopt explicit fixed percentages and identify missing metric evidence; do not infer legal ownership or weights from spend. [FinOps Allocation](https://www.finops.org/framework/capabilities/allocation/) separates grouping and accountable assignment; keep B01 infrastructure evidence separate from B03 assignment. [Carbon Data Table](https://carbondesignsystem.com/components/data-table/usage/) supports table toolbars, row identity and progressive detail. Adopt bounded searchable results and readable editing; reject decorative colored edges, compressed14px body and fixed-width page containers. No third-party assets or dependencies are copied.

Policies: compact source bar, full-pool amount context, policy name/method and target percentage editor with visible total100 denominator and unassigned state; primary Preview then browser save. Pools: source full-currency/provider summaries with precision/coverage/target evidence. Simulation: source basis, assigned/unassigned/precision remainder, complete recipient result table and comparison-baseline action. Comparison: previous session baseline/current preview, deltas and exact context; no fake server history. Advanced caveats live in contextual disclosures; avoid tall repeated disclaimer panels that push the task below the fold.

Use existing compact title/review cluster, local IBM Plex16body/14metadata/20sections, neutral borders/square/no shadows, logical layout with isolated LTR dates/IDs/money. At390px cells must size automatically rather than inherit fixed44px rows. Keep row identity and selected-target context readable during horizontal editing. KO/EN/AR remain first-class variants.

# Validation plan

Financial review covers full-pool denominator, null/credit/currency cases, exact conservation and explicit unapproved residue. Security review covers source scope/pins, no display-filter denominator mutation, metadata-only/fresh authorized restore, async clearing and dirty navigation. TW reviews concept names, source-versus-approved and recipient-versus-organization semantics before and after rendering. Tests include bounded randomized conservation/permutation properties and meaningful independent known cases; storage corruption/foreign scope; actual UI edit/preview/compare/save/restore/failure/slow obsolete source; six locale/layout captures, focus and computed styling. Do not run retained-store reset fixtures. Product completeness/OS/native-language/owner approval remain separate evidence.

# Independent design review

Separate financial and security design reviews conditionally passed the bounded non-authoritative review editor. Their required complete-manifest100/paged25 cap, restricted-entry block with explicit whole-source reset, source present/missing checks, full100 denominator, signed precision residue, exact pins/profile and fresh authorized restore are included above. TW design review passed the terminology with source/recipient/approval boundaries; preferred tabs are Allocation policy / Allocation basis / Allocation preview / Result comparison. Actual-diff and rendered review remain pending.

The successful Claude CLI ui-designer design inspected the current generic B03 capture, the actual B01 reference, authorized B03 spec, active design guides, mandatory service style and installed frontend-design skill. Its compact source strip, first-viewport editor, side summary, sticky row identity and responsive layout are adopted. Currency requires explicit user selection; there is no automatic USD or first-currency choice. TW labels govern over the designer’s provisional wording. Sticky mobile bottom actions are rejected to keep validation and amounts unobscured. Complete original B03 approval/idle/organization contracts remain pending integrations rather than rewritten as implemented.

# Stable implementation items

| Item ID | Location and behavior | Verification boundary |
| --- | --- | --- |
| B03:scope:whole-source | Provider/month/basis/currency, full server aggregate and explicit restricted-entry reset | Actual source queries, strict source-context tests and browser scope guards |
| B03:form:fixed-percentages | Named review with exact fixed100 denominator and missing/invalid input states | Percentage bounds, complete manifest and error focus |
| B03:calculation:source-precision-review | Signed source-precision shares, unallocated amount and precision remainder | Independent known cases,250 generated invariants and actual USD2.50 sample |
| B03:table:recipient-manifest | Actual synthetic access scopes, max100/paged25, display-only search | Duplicate/limit tests, display search does not alter results |
| B03:comparison:session-baseline | By-value same-context session baseline, explicit current and signed changes | Context rejection, candidate invalidation and browser comparison |
| B03:action:browser-draft | Strict projected metadata, fresh-authorized restore and visible readback failure | Model/store tests and browser failure paths |
| B03:state:source-unavailable | Loading/restriction/error replaces prior source, draft and results |403 and obsolete actual-response replay; no offline fallback |

The original full B03 approval/organization/idle/production-rounding requirements are retained as explicit integrations, not rewritten as complete. The later [owner confirmation](../../ui/acceptance/B03.md) fixes only its declared policy-editor scope; remaining items/variants stay unconfirmed.
