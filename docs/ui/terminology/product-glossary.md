---
type: ProductGlossary
title: Service terminology and meaning guards
description: Defines preferred display terms while preserving canonical contracts and financial evidence boundaries.
status: draft
---

# Scope and provenance

The owner requested continuous TW review on 2026-11-04 and questioned `원가 풀`. A separate AI technical-writer role inspected B01 planning and current U01/S04 copy, then recommended the terms below. This is reviewed product wording, not an official standardized translation, human finance-practitioner approval or Arabic native-language acceptance. Apply the [ongoing rule](../../../.agents/rules/product-terminology.md) to each affected screen and record actual rendered review.

| Concept ID / canonical alias | Preferred KO | Preferred EN | AR candidate | Definition and forbidden implications | Current scope |
| --- | --- | --- | --- | --- | --- |
| infrastructure-composition / CostPool, B01 | 인프라 원가 구성 | Infrastructure cost composition | مكوّنات تكلفة البنية التحتية | Period-specific asset, lease/depreciation, license, power and facility cost composition and inclusion review. Not B03 allocation, sales pricing or a complete infrastructure ledger inferred from VMware rows. | B01 menu/title; AI TW design and scoped six-variant rendered review PASS; native-language acceptance pending |
| billed-cost / BilledCost | 청구 비용 | Billed cost | التكلفة المفوترة | Collected billed source cost, not customer sales, cash payment or approved internal allocation. Preserve FOCUS alias in mapped fields/help. | U01/S04/B01 |
| effective-cost / EffectiveCost | 실효 비용 | Effective cost | التكلفة الفعلية | Source cost with applicable amortization/discount treatment. Not merely discounted price, cash paid or financial approval. | U01/S04/B01 |
| selection-reference / snapshot | 조회 기준 ID | Selection reference ID | معرّف مرجع الاستعلام | Fixed selected source/version context. Not a financial close or a guarantee of continuous authorization. | U01/S04/B01 |
| source-reference / source_snapshot | 수집 데이터 기준 ID | Source data reference ID | معرّف مرجع بيانات المصدر | Pinned collected-source evidence identity. Not raw-object access or an approved ledger. | S04/B01 |
| account-scope / observed account combination | 계정 범위 수 | Account scopes | عدد نطاقات الحسابات | Distinct internal access/provider/BillingAccount/SubAccount identities. Not organization count or unique billing accounts. | U01/S04/B01 |
| charge-start-month / charge_start attribution | 과금 시작 월 | Charge-start month | شهر بدء الرسوم | Source charge-start month in tenant timezone, without proration. Not billing, settlement, recognition or depreciation month. | U01/S04/B01 |
| consumed-quantity / ConsumedQuantity, ConsumedUnit | 사용 수량 / 사용 단위 | Consumed quantity / Consumed unit | كمية الاستهلاك / وحدة الاستهلاك | Original consumed measurement/unit. Not pricing denominator or mixed-unit total. | S04/B01 |
| pricing-quantity / PricingQuantity, PricingUnit | 요금 산정 수량 / 요금 산정 단위 | Pricing quantity / Pricing unit | كمية التسعير / وحدة التسعير | Source pricing measurement/unit. Do not infer consumed-unit equivalence or an unprovided pricing formula. | S04/B01 |
| declared-inclusion / local review relation | 다른 원가 항목에 포함(검토 선언) | Included in another cost item (review declaration) | مشمول في بند تكلفة آخر (إقرار للمراجعة) | User-declared whole-row inclusion with a reference. Not verified product inclusion, certified duplication, savings or an approved cost denominator. | B01 temporary draft |
| allocation-policy / B03 | 배분 정책 | Allocation policies | سياسات التوزيع | Defines review proportions and result comparison. A local preview is not an approved policy revision or posted allocation. | B03 |
| collected-basis / full server amount | 수집 원천 기준액 | Collected-source basis amount | مبلغ الأساس من البيانات المجمّعة | Complete selected-currency source aggregate within the explicit authorized source query, not the visible page, sales amount or approved infrastructure cost. | B03 |
| review-recipient / tenant scope | 검토 대상 접근 범위 | Review recipient access scope | نطاق وصول مستلم المراجعة | Actual authorized sample tenant access scope. Does not establish legal entity, department or customer contract ownership. | B03 |
| recipient-sum / sum actual shares | 대상별 배분액 합계 | Total recipient allocations | مجموع المبالغ الموزّعة على المستلمين | Sum of actual provisional target shares after explicit source-precision truncation; not the intended amount before precision loss. | B03 |
| unallocated / source minus intended | 미배분 금액 | Unallocated amount | المبلغ غير الموزّع | Source amount minus the intended allocation at the entered percentage sum. Not unknown idle cost or precision remainder. | B03 |
| precision-remainder / intended minus shares | 계산 정밀도 잔액 | Calculation precision remainder | المتبقي الناتج عن دقة الحساب | Intended allocation minus actual recipient sum at the provisional source-precision arithmetic profile. Not automatically assigned or certified financial rounding. | B03 |
| session-baseline / local comparison result | 현재 세션의 비교 기준 | Comparison baseline for this session | أساس المقارنة للجلسة الحالية | Immutable by-value preview within identical source/recipient/profile context; not server history or prior financial approval. | B03 |

# B01 labels

Keep the existing `summary/costs/products/approvals` tab keys, displaying 구성 요약 / 원가 항목 / 상품별 포함 항목 / 승인 이력. Name field: 원가 구성명. `원가 소유 법인` refers to verified legal ownership only; when unavailable, show it as unconnected rather than making the FOCUS account the owner. Browser-only review editing is `이 브라우저의 검토 초안`; saving is `검토 초안 임시 저장`, never server policy approval.

# Evidence

B01's authorized planning spec and v1.16 ui-design119/177/505 distinguish underlying infrastructure costs, included charges, approved internal costs and later allocation/sales rules. [FinOps for Data Center](https://www.finops.org/framework/technology-categories/data-center/) distinguishes asset cost from service chargeback; [FinOps Allocation](https://www.finops.org/framework/capabilities/allocation/) concerns assigning/shared costs to responsible groups. These support the conceptual boundary, not the specific translation.

FOCUS definitions: [Billed Cost](https://focus.finops.org/docs/specification/v1-0/columns/cost-and-usage/billed-cost/), [Effective Cost](https://focus.finops.org/docs/specification/v1-0/columns/cost-and-usage/effective-cost/), [Consumed Quantity](https://focus.finops.org/docs/specification/v1-0/columns/cost-and-usage/consumed-quantity/) and [Pricing Quantity](https://focus.finops.org/docs/specification/v1-0/columns/cost-and-usage/pricing-quantity/). Canonical mappings remain unchanged.

# B03 terminology evidence

Separate AI TW design review for HB-103 inspected the authorized B03 planning and numeric contract. Preferred tabs preserve `policies/pools/simulation/comparison` while displaying 배분 정책 / 배분 기준액 / 배분 미리보기 / 결과 비교. The full denominator is100 percent; display search/pages never normalize entered weights. Blank percentages are missing. Unknown idle cost stays separate from the provisional arithmetic equation. Native-language and human practitioner acceptance remain pending.
