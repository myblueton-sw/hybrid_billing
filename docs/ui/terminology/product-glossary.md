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
| temporary-source-baseline / B08 | 원천 비교 기준 잠시 보관 | Temporarily keep source baseline | الاحتفاظ المؤقت بمرجع المصدر | Volatile source aggregate observation on this screen only; never a frozen manifest, calculation lock, financial run, policy or approval. | B08 |
| source-input-comparison / comparison | 원천 입력 비교 | Source input comparison | مقارنة مدخلات المصدر | Same-source-reference authorized scope comparison, not prior-run billing variance. Changed opaque source references invalidate the baseline. | B08 |
| observed-measure-presence / present-missing | 값 있음 / 누락 | Present / missing | موجود / مفقود | Presence within observed source rows, not whole-source completeness or financial readiness. Show both sides in comparisons. | B08 |
| pre-billing-checks / validation | 청구 전 확인 | Pre-billing checks | فحوص ما قبل الفوترة | Source read is observed; contract eligibility, shared dependencies, policy/result checks and calculation/issuance locks remain separate. | B08 |
| monthly-expense-recognition / recognition | 월별 비용 인식 | Monthly expense recognition | الإثبات الشهري للمصروفات | Legal entity/book/basis and actual ERP posting are required; not revenue recognition, invoicing, receivables, collection or payment-term commencement. | B08 |
| collection-receipt / delivery | 수집 접수 | Collection delivery | تسليم التجميع | Retained metadata for an attributable received source revision; not a job success, amount or financial approval. | S03 |
| source-publication-record / sealed | 원천 공개 기록 | Source publication recorded | تم نشر مراجعة المصدر | Historical source publication; does not establish current physical presence, financial eligibility or worker confirmation. | S03 |
| registered-collection-expectations / completeness | 등록된 기대 수집 범위 | Registered collection expectations | توقعات التجميع المسجلة | Operator-registered calendars/domains, not whole-CSP upstream completeness. Unsupported count remains unknown. | S03 |
| declared-source-rows / declared_rows | 선언된 행 수 | Declared rows | الصفوف المصرّح بها | Source descriptor row declaration, not the financial result, validated billed count or inferred duplicate count. | S03 |
| explicit-zero-attestation / known_zero | 명시적 0 확인 근거 | Explicit zero-attestation evidence | دليل الإقرار الصريح بالصفر | True only when every registered domain has a current published explicit zero attestation. False does not prove nonzero; missing publication remains unknown. | S03 |

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

# B08 terminology evidence

Independent AI finance/TW design and actual-source reviews distinguish source observation, temporary in-memory baseline, full-scope amount presence and financial run authority. Canonical runs/inputs/trace/comparison/validation/recognition keys remain, with source input comparison and pre-billing checks as qualified labels. Both comparison sides show present/missing counts after an independently reproduced partial-subtotal finding. Monthly expense recognition remains distinct from customer billing and actual ERP posting. Selected actual rendered finance/TW review passed and is recorded in HB-109 verification; native-language/human practitioner and owner acceptance remain pending.

# B09 terminology evidence

Independent AI finance/TW design and actual-source reviews preserve five separate stages: 청구 계산 결과 / Billing result, 최종 ERP 금액 / Final ERP amount, 조정 / Adjustment, 정책 / Policy, 월별 비용 인식 / Monthly expense recognition. A source-recorded decision is not present authorization, issuance, posting or collection. `내 결정` / My decisions requires the supplied actual decision actor to match the read actor; author identity or a role-preview label cannot substitute. `위임·대리` / Delegation and substitution remains unconnected without a grant producer. Unknown total is not zero; absent ERP observations are not explicit ERP not-applicable. `원천 기록` / Source record and `원천에서 미확인` / Source reports unknown distinguish declared state from missing fields. Synthetic records are always labelled and live producers remain unbound. Selected render-review outcomes and native-language/human-practitioner limits are recorded in HB-110 verification.

# B10 terminology evidence

Independent AI finance/TW design and actual-source reviews prefer 청구 적용 이력 / Billing application history / سجل تطبيق الفوترة; 부분 청구 / Partial billing / الفوترة الجزئية; 적용 취소·정정 / Reversals and corrections / عمليات العكس والتصحيح; 외부 참조 / External references / المراجع الخارجية. Canonical ledger/partial/reversals/references keys remain unchanged. `적용 범위·보호` / Coverage and protection refers only to the supplied application/subset with purpose/universe/unit/version; scalar observations cannot certify interval availability or duplicate clearance. `남은 금액 (원천 기록)` / Remaining amount reported by source is neither available coverage nor receivable balance. Money-only correction does not reopen quantity; quantity reversal and replacement pending confer no rebilling authority. Pending/unknown same-application external evidence requires unknown available coverage, including where a contradictory known0 was supplied. ERP outcome declarations do not prove reconciliation or release authority. Native Arabic/human-practitioner and owner acceptance remain separate; actual render-review scope is recorded in HB-111 verification.

# B11 terminology evidence

Independent AI finance/TW design and actual-source/render review prefers 크레딧·청구 문의 / Credits and billing inquiries / الأرصدة الائتمانية واستفسارات الفوترة. Five tabs retain canonical keys while displaying 크레딧 잔액 / Credit balances / أرصدة الائتمان; 예약·적용 이력 / Reservations and applications / الحجوزات والتطبيقات; 청구 문의·이의 / Billing inquiries and disputes / استفسارات الفوترة والاعتراضات; 조정·취소 이력 / Adjustments and reversals / التسويات وعمليات العكس; 지급 기한·연체 / Due dates and overdue / مواعيد الاستحقاق والتأخر في السداد. Stale balances are 마지막 원천 관측값 · 현재 사용 가능 미확인, not usable cash/receivable. Source signs are preserved without adopting a vendor convention. Internal notes and recorded public replies remain separate; a correction link does not imply consent. Holds use Arabic إجراءات التعليق, not comment terminology. Inquiry/hold does not erase payment obligations or change due dates; stale receipts leave applicable overdue unknown. Monetary observations do not provide quantity-reversal or replacement evidence. Native-language/human and actual producer acceptance remain separate; exact rendered review scope is recorded in HB-112 verification.

# S03 receipt evidence wording

HB-145 independent AI finance/TW source review and review of eight actual captures preserve collection metadata versus financial authority, historical publication versus current availability, declared counts versus financial totals, registered expectation versus upstream completeness, and explicit all-domain zero attestation versus unknown/nonzero. Arabic numeric UTC timestamps remain LTR with source precision; interval starts are inclusive and ends exclusive. Review scope and actual limits are recorded in [HB-145 verification](../../features/HB-145/verification.md). AI review is not human finance-practitioner, Arabic native-language or owner FIX acceptance.

# S03 current stored observations

HB-150 adds current scoped stored-data observations alongside historical receipt metadata. Keep all existing S03 receipt/declared-count terms and canonical fields. Candidate terms below received separate AI contract/TW review; native-language and practitioner acceptance remain pending.

| Concept | KO | EN | AR | Meaning and forbidden implication |
| --- | --- | --- | --- | --- |
| queryable-authorized-rows / row_count | 허용 범위의 조회 가능한 행 | Queryable rows in your permitted scope | الصفوف القابلة للاستعلام ضمن نطاقك المسموح | Currently read and verified allowed rows, independent of receipt prefix filtering; not whole-provider inventory or approved billable rows. |
| contributing-deliveries / delivery_count | 조회 가능한 수집본 | Queryable deliveries | عمليات التسليم القابلة للاستعلام | Distinct current publications contributing at least one allowed row; excludes foreign-only zero slices and is not a connection or whole-file receipt count. |
| source-delivery-window | 원천 수집 구간 | Source delivery window | نطاق تسليم المصدر | UTC interval with inclusive start/exclusive end, grouped across allowed contributions; not a charge-start-month filter. |
| public-source-sample | 공개 샘플 / 공개 샘플 일부 | Public samples / Public sample subsets | عينات عامة / أجزاء من عينات عامة | Allowlisted sample provenance from source; not live provider certification. |
| synthetic-source-fixture | 합성 시험 자료 | Synthetic test data | بيانات اختبار اصطناعية | Deliberately synthetic source rows, separate from public samples and customer data. |
| upstream-completeness | 원천 전체 수집 완전성 · 미확인 | Upstream completeness · Unverified | اكتمال التجميع من المصدر · غير متحقق | No complete upstream inventory is established by a successful stored-row read. |
| original-file-availability | 원본 파일 가용성 · 미확인 | Original-file availability · Not checked | توفر الملف الأصلي · لم يُتحقّق منه | Current row verification does not itself re-read original file bytes. |

Regional sources and adopted/rejected layout patterns are recorded in [HB-150 references](../../features/HB-150/references.md). Receipt metadata may be unavailable independently of stored rows; authority denial/mismatch clears both observations. AI review does not authenticate an owner or finalize regional terminology.

## S04 source fields and row detail — HB-151

[HB-151](https://linear.app/hybrid-billing/issue/HB-151/refine-s04-source-fields-and-row-detail-to-separate-source-values-from) separates values read from mapped FOCUS columns from values the product derives. A separate AI technical-writing review read the KO/EN/AR source strings and its corrections are applied here; the rendered-context inspection is recorded in the [HB-151 verification record](../../features/HB-151/verification.md). Neither is native-language, practitioner or owner acceptance. "Billing" stays 결제 for the account and 청구 for currency and period, as in existing S04 copy.

| Concept | KO | EN | AR | Meaning and forbidden implication |
| --- | --- | --- | --- | --- |
| source-column-value / mapped field | 원천 열 값 | From source columns | من أعمدة المصدر | Value read from the named FOCUS column through the version mapping. Not the original text form, a verified value or provider conformance. |
| derived-value / charge_month, id, account | Hybrid Billing 산출 값 · 행 ID · 계정 범위 ID | Derived by Hybrid Billing · Row ID · Account scope ID | قيم يشتقها Hybrid Billing · معرّف الصف · معرّف نطاق الحساب | Computed by the product; no FOCUS column holds it. Not a provider identifier or a financial result. |
| delivery-lineage / batch, ordinal | 수집본 식별 정보 · 수집본 ID · 수집본 내 행 순번 | Delivery lineage · Delivery ID · Row ordinal in delivery | بيانات تتبّع التسليم · معرّف التسليم · ترتيب الصف في التسليم | Group label, then its two fields: delivery ID and row ordinal inside that delivery. Not a receipt, a source delivery window or original-file access. |
| sample-annotation / x_HB_EvidenceClass | 샘플 주석 · 샘플 출처 | Sample annotation · Sample provenance | وسم العينة · منشأ العينة | Group label, then its field. Annotation column carried in the collected sample row. Absent or unrecognized values read Unclassified; never a collector-assigned class or customer-data claim. |
| missing-row-value / null | 누락 | Missing | مفقود | The collected row has no value for the field; the reason is not carried by the row record. Never rendered as 0, a dash or with a currency. Replaces 미확인/Unavailable and "not provided" for null row and account-list values on S04. A missing unit reads 단위 누락 / Unit missing / الوحدة مفقودة. |
| exact-zero-value / zero coefficient | 정확한 0 | Exact zero | صفر دقيق | The collected numeric value is zero. Row fields only; not S03's explicit zero-attestation and not applied to scope totals. A scope total without present values reads 값 없음 / no values present / لا قيم موجودة. |
| charge-period / ChargePeriodStart, ChargePeriodEnd | 과금 기간 | Charge period | فترة الرسوم | Source interval, start inclusive and end exclusive by FOCUS definition, shown in UTC. Not the charge-start month. Replaces "charge interval" on S04. |
| billing-period / BillingPeriodStart, BillingPeriodEnd | 청구 기간 | Billing period | فترة الفوترة | Source billing interval, same bounds and UTC display. Not a settlement or recognition period. Replaces "billing interval" on S04. |
| not-carried-by-row-record | 행 기록에 없는 정보 | Not carried by the row record | غير متضمَّن في سجل الصف | Evidence the current row query does not return, so the screen neither shows nor infers it. Not a statement that the evidence does not exist. |

## B01 use of the source-field terms (HB-153)

B01 applies the HB-151 terms above without new concepts: Missing and Exact zero for row amounts, charge period, the derived charge-start month, delivery ID and row ordinal, and sample provenance, which always appears with its field label because its Unclassified value differs from the Unclassified cost category (미분류 / غير مصنّف). Subtotals keep 값 없음 / No values present and are never labeled Exact zero; a subtotal that could not be computed reads 산정 불가 / Unavailable / غير متاح. A short row identity for option text and control labels reads 순번 · 하위 계정 or 결제 계정 · 행 ID / Ordinal · Subaccount or Billing account · Row ID / الترتيب · الحساب الفرعي or حساب الفوترة · معرّف الصف; the row ID is the only unique key. An evidence reference the user has not entered reads 근거 참조 미입력 / No reference entered / لم يُدخل مرجع, not Missing. AI wording review only; Arabic and Korean practitioner review remains open. U01, B04, B05, B06, B07 and B08 still carry "charge interval" or "no value" wording and are aligned in their own tickets.

## B08 use of the source-field terms (HB-154)

B08 applies the HB-151 terms to its source input rows, row detail and account list: Missing, Unit missing and Exact zero for row values, charge period and billing period in UTC, the derived charge-start month, delivery ID and row ordinal, sample provenance, 결제 계정 ID and 공급사. The Exact zero note appears in the row detail and trace view, not in the 25-row table. Totals read 값 없음 / No values present when no value is present and keep 값 있음 / 누락 counts; totals and differences are never labeled Exact zero. Negative amounts, totals and differences use the shared format `USD -1.00`. B08 keeps its basis labels 원천 청구 비용 / 원천 실효 비용 because they mark the amounts as source costs on a billing screen. AI wording review only.
