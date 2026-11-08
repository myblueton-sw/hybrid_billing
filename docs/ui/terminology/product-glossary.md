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
| sales-pricing-policy / B04 | 판매 요금 정책 | Sales pricing policies | سياسات تسعير المبيعات | Manual independent unit-price review. Contract activation, eligibility, tax/FX, full tier/minimum pools and approval remain distinct unavailable integrations. | B04 |
| manual-selling-unit-price / unitPrice | 판매 단가 — 직접 입력 | Selling unit price — manual input | سعر وحدة البيع — إدخال يدوي | Explicit review proposal, never preloaded or inferred from provider ListUnitPrice/ContractedUnitPrice. | B04 |
| quantity-per-price / denominator | 단가 기준 수량 | Quantity per price | الكمية المقابلة للسعر | Positive denominator in the unchanged source PricingUnit. Must be entered/chosen; no consumed-unit substitution or implicit conversion. | B04 |
| pre-fx-tax-review / candidate | 환산·세금 적용 전 검토 금액 | Review amount before FX and tax | مبلغ المراجعة قبل تحويل العملة والضريبة | Provisional source-precision result, not final customer sales or a zero-tax claim. Source costs remain separate evidence. | B04 |
| target-review-amount / target | 목표 검토 금액 | Target review amount | مبلغ المراجعة المستهدف | Same-currency pre-FX/pre-tax target on a current session candidate. Not an approved invoice adjustment. | B04 |
| review-difference / adjustment delta | 검토 차액 | Review difference | فرق المراجعة | Target minus original candidate, displayed as a separate signed result. Never overwrites source or implies financial application. | B04 |
| source-precision-truncation / nonzero division remainder | 원천 정밀도에서 절삭 | Truncated at source precision | مقتطع بدقة المصدر | Explicit toward-zero source-scale arithmetic. Raw division remainder is not money or an activated currency quantum. | B04 |
| fx-rounding-policy / B05 | 환율·반올림 정책 | FX and rounding policies | سياسات تحويل العملة والتقريب | Manual source-cost conversion review; not an approved FX policy or B04 selling amount. | B05 |
| manual-fx-quote / rate | 수동 입력 환율 | Manually entered exchange rate | سعر الصرف المُدخل يدوياً | Target-currency units per explicit source quotation unit; manual reference is not verified published rate authority. | B05 |
| fx-quotation-unit / quoteUnit | 환율 기준 수량 | Exchange-rate quotation unit | كمية أساس تسعير الصرف | Positive source-currency denominator retained in the complete equation, including per 100 quotations. | B05 |
| amount-quantum / quantum | 금액 처리 단위 | Amount rounding increment | وحدة تقريب المبلغ | Explicit positive target-currency increment such as 0.05; never inferred decimal places or currency standard. | B05 |
| rounding-difference / difference | 금액 처리 차이 | Rounding difference | فرق التقريب | Rounded result minus exact pre-round rational amount; raw quotient remainder is not money. | B05 |
| unconverted-declaration / treatment | 미환산 원천 금액으로 검토 — 사용자 선언 | Treat as unconverted source amount — user declaration | اعتبار المبلغ أصلياً غير محوّل — إقرار المستخدم | Explicit review choice; supplier conversion history remains unknown and already-converted choices block this exercise. | B05 |
| commitment-source-group / B06 | 관측된 약정 ID 그룹 | Observed commitment ID groups | مجموعات معرّفات الالتزام المرصودة | Source ID scoped by authorized tenant/provider/account/subaccount/currency. Not a legal-contract count or ownership claim. | B06 |
| commitment-usage-class / source status | 약정 사용 분류 — 원천 값 | Commitment usage classification — source value | تصنيف استخدام الالتزام — قيمة المصدر | Charge-level mapped Used/Unused evidence; not lifecycle status, VM utilization, savings or contractual unused burden. | B06 |
| no-observed-classified-rows / empty bucket | 관측된 행 없음 | No observed rows | لا توجد صفوف مرصودة | No classified rows in this selection; never a measured zero-unused-cost claim. | B06 |
| version-mapping-evidence / mapping | 버전 매핑 행 | Version-mapped rows | صفوف مربوطة بالإصدار | Rows interpreted through the pinned version mapping, not provider conformance, source validation or financial eligibility. | B06 |
| source-interval-envelope / minmax | 원천 기간 범위 | Source interval envelope | نطاق الفترة من المصدر | Earliest start through latest end of observed records, not continuous coverage or commitment lifetime. | B06 |
| reconciliation-evidence-review / B06 | 대사 검토 | Reconciliation review | مراجعة المطابقة | Source completeness/version/field evidence inspection, not financial reconciliation completion or monthly Billed/Effective equality. | B06 |
| marketplace-candidate-source / B07 | Marketplace 후보 원천 행 | Marketplace candidate source rows | صفوف مصدر مرشّحة للسوق | Exact non-null mapped issuer/publisher name inequality is a reporting signal, not a confirmed transaction, legal role or customer invoice. | B07 |
| invoice-issuer / InvoiceIssuerName | 청구서 발행사 | Invoice issuer | جهة إصدار الفاتورة | Entity named in the source as issuing its invoice, not an inferred reseller or customer-credit authority. | B07 |
| publisher / PublisherName | 상품 제공사 | Publisher | ناشر المنتج | Source publisher identity; provider-label inequality does not establish Marketplace purchasing. | B07 |
| no-name-difference / no_difference | 이름 차이 없음 | No name difference observed | لم يُرصد اختلاف في الاسمين | Exact source-name equality; not proof of legal-party identity, direct purchase or absence of Marketplace activity. | B07 |
| insufficient-classification / unknown | 분류 근거 부족 | Insufficient classification evidence | أدلة غير كافية للتصنيف | Missing, blank, invalid or unmapped names remain unknown and are excluded from candidate amounts. | B07 |
| seller-withheld-amount / withholding | 보류액 | Withheld amount | المبلغ المحتجز | Separately sourced withheld balance; never assume this is withholding tax or calculate it from buyer source costs. | B07 |

# B01 labels

Keep the existing `summary/costs/products/approvals` tab keys, displaying 구성 요약 / 원가 항목 / 상품별 포함 항목 / 승인 이력. Name field: 원가 구성명. `원가 소유 법인` refers to verified legal ownership only; when unavailable, show it as unconnected rather than making the FOCUS account the owner. Browser-only review editing is `이 브라우저의 검토 초안`; saving is `검토 초안 임시 저장`, never server policy approval.

# Evidence

B01's authorized planning spec and v1.16 ui-design119/177/505 distinguish underlying infrastructure costs, included charges, approved internal costs and later allocation/sales rules. [FinOps for Data Center](https://www.finops.org/framework/technology-categories/data-center/) distinguishes asset cost from service chargeback; [FinOps Allocation](https://www.finops.org/framework/capabilities/allocation/) concerns assigning/shared costs to responsible groups. These support the conceptual boundary, not the specific translation.

FOCUS definitions: [Billed Cost](https://focus.finops.org/docs/specification/v1-0/columns/cost-and-usage/billed-cost/), [Effective Cost](https://focus.finops.org/docs/specification/v1-0/columns/cost-and-usage/effective-cost/), [Consumed Quantity](https://focus.finops.org/docs/specification/v1-0/columns/cost-and-usage/consumed-quantity/) and [Pricing Quantity](https://focus.finops.org/docs/specification/v1-0/columns/cost-and-usage/pricing-quantity/). Canonical mappings remain unchanged.

# B03 terminology evidence

Separate AI TW design review for HB-103 inspected the authorized B03 planning and numeric contract. Preferred tabs preserve `policies/pools/simulation/comparison` while displaying 배분 정책 / 배분 기준액 / 배분 미리보기 / 결과 비교. The full denominator is100 percent; display search/pages never normalize entered weights. Blank percentages are missing. Unknown idle cost stays separate from the provisional arithmetic equation. Native-language and human practitioner acceptance remain pending.

# B04 terminology evidence

Separate AI TW review of HB-104, B04, PC-01/SP-01 and numeric/H23 contracts conditionally approved the terminology. Prefer 요금 규칙 / 계산 단계 / 정책 상속 / 조정 검토 / 금액 비교 for the stable five tab keys. Entered unit prices, fees and target amounts are financial review parameters intentionally saved locally; never call that payload merely metadata-only. Signed source quantities retain their sign without automatically being classified as approved refunds. Native-language/human practitioner acceptance remains pending.

# B05 terminology evidence

Independent AI TW design review of B05 and the PC-01/AR-06/numeric contracts conditionally passed the displayed manual quote, quotation unit, markup, target fee, quantum and difference distinctions. Preserve policies/imports/comparison/approvals while displaying 환산 규칙 / 환율 근거 / 환산 비교 / 승인 이력. Dates and maximum age are declared UTC review evidence, not verified market freshness. Explicit browser storage retains entered financial parameters, source descriptions and references. Actual source and selected rendered TW reviews passed after the mobile source-card correction. Native-language/human practitioner acceptance remains separate.

# B06 terminology evidence

Independent AI TW design and actual-source reviews distinguish scoped source ID groups from legal contracts, charge-level Used/Unused mapping from lifecycle/utilization, observed zero from absent observations, and evidence review from completed financial reconciliation. The actual source counters are explicitly version-mapped/unmapped; min/max intervals are evidence envelopes. Preferred tabs preserve commitments/utilization/benefits/reconciliation while displaying 약정 명세 / 사용·미사용 근거 / 혜택 흐름 / 대사 검토. Current-source copy was corrected to usage classification and existing collected/query reference-ID terminology. Independent AI TW selected actual-render reviews and the final coverage-copy follow-up passed. Native-language/human practitioner and owner acceptance remain pending.

# B07 terminology evidence

Independent AI finance/TW design and actual-source reviews preserve candidate source-row counts, source Billed/Effective bases, lexical name comparison and the unconnected seller/benefit/legal boundaries. Canonical buyer/seller/credits/mapping keys remain; preferred display tabs are 구매 측 청구 근거 / 판매자 정산 / 혜택·크레딧 / 거래 연결 근거. Refunds and taxes remain separate from fees, withheld balances and disbursements. No settlement formula or invoice-to-collection inference is adopted. Selected actual rendered review and the final count/version/selection-copy follow-up passed, recorded in HB-107 verification; native-language/human practitioner and owner acceptance remain pending.
