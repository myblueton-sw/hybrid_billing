---
type: DesignSpecification
title: FX settings field validation supplement
description: Separates raw exchange rates quotation units selling markup and fees without changing approved financial semantics.
status: draft
---

# Scope

HB-59 DOC01; B05 settings and B09 approval handoff only. Inherits [shared rules](../../11-settings/r2/README.md). Evidence: v1.16 B0463/B0464/B0467/B0474/B0477.R002; [FX command validation](../../../contracts/commands/planning-actions.md#fx-component-validation), [numeric contract](../../../contracts/billing/claim-and-numeric.md) and [binding register](../../../ui/screen-binding-register.md). This draft corrects the blanket zero/negative rule in the local B05 companion; it does not edit/adopt that companion, replace all module 06 or recompute any amount.

# B05 fields and actions

Entry: Contract → Pricing → Exchange rates; scoped pricing readers/editors with separate approval authority. Fields: source rate/provider evidence, base/quote currency pair and direction, raw published decimal rate, quotation unit/denominator, effective time and timezone, cost-rate rule, selling-rate rule, markup, conversion fee, units/basis per component, sign-policy revision, rounding stage/mode, source freshness and expected revision. Pair/direction, scale/units and effective time must be explicit; no reciprocal conversion or default policy is inferred.

| Component | Validation and presentation |
| --- | --- |
| Raw exchange rate | Greater than zero; approved source and correct pair/direction. A rate already containing selling markup is a finding, not an eligible raw rate |
| Quotation unit / denominator | Greater than zero and in the stated units; zero is invalid |
| Selling markup | Zero valid; negative only when that component's approved sign policy permits it |
| Conversion fee | Zero valid; negative only when that component's approved sign policy permits it; distinguish fixed amount versus rate and its currency/basis |
| Cost versus selling | Separate rules/evidence; show FX difference explicitly, do not hide it inside margin or raw rate |

Flow: draft components → field validation → preview affected unissued scope and rounding/effective rules → independently approved exact revision → apply/read-back through the existing governed pricing flow. Approval is invalidated by component/source/scope changes. Published or externally pinned documents retain their snapshot; later changes follow correction policy. Existing B05 approval request is referenced, but exact draft/apply command bindings remain **unresolved**, not invented in this supplement. A policy approval is not `settlement.approve_final`.

Walkthrough F01: raw rate 1 with quotation unit 1 and zero markup/fee passes sign validation only; source, currency, effective period and independent approval still apply. Raw rate zero is blocked. A negative markup with no approved sign policy is blocked as missing policy, not globally forbidden; with an applicable permitting policy it proceeds to preview, never automatic approval. A stale source or changed rate after preview invalidates the decision and requires fresh evidence. Examples are synthetic validation cases, not selected commercial rates or calculated invoice results.
