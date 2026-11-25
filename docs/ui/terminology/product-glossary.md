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
| retained-issuance-settings / O03 documents | 발행 설정 이력 | Recorded issuance settings | إعدادات الإصدار المسجلة | Historical proposal/confirmation evidence, including unknown outcomes; not current issuance eligibility or an editable confirmed configuration. | HYB-341 Claude source TW review PASS; owner screen/native-language acceptance pending |
| issuance-confirmation-attempt / O03 confirmer | 확인 시도자 | Confirmation attempted by | مَن حاول التأكيد | The actor of an unresolved confirmation attempt. Use Confirmed by only when retained confirmation publication completed; module observation alone is insufficient. | HYB-341 Claude source TW corrective review PASS; owner screen/native-language acceptance pending |
| synthetic-issuance-preset / synthetic_erp_sample_v1 | 합성 ERP 샘플 | Synthetic ERP sample | نموذج ERP اصطناعي | An explicit sample adapter option. Region and synthetic tax references do not establish statutory tax applicability or real issuance readiness. | HYB-341 Claude source TW review PASS; owner screen/native-language acceptance pending |
| decision-condition-check / B09 | 결정 조건 확인 | Check decision conditions | التحقق من شروط القرار | Explicit current observation for the selected financial request; not a decision, reservation or future authorization. | HYB-340 AI TW source review PASS; native-language acceptance pending |
| unknown-decision-outcome / B09 | 처리 결과 미확인 | Outcome unknown | نتيجة العملية غير معروفة | A submission may have recorded a decision; only the identical submission can be retried. A retry refusal does not establish failure of its earlier effect. | HYB-340 AI TW source review PASS; native-language acceptance pending |
| identical-decision-retry / B09 | 동일한 제출 다시 시도 | Retry the same submission | إعادة محاولة الإرسال نفسه | Resend the frozen action, target, evidence, rationale, reason and attempt identity; never create a fresh decision attempt automatically. | HYB-340 AI TW source review PASS; native-language acceptance pending |
| charge-approval-limit-proposals / financial_authority_proposals | 청구 승인 한도 제안 | Charge approval limit proposals | مقترحات حدود اعتماد الرسوم | Pending StageCharge proposals binding a human recipient, currency, exact maximum absolute amount and inclusive UTC dates. Counts proposals, not amounts or effective decision permission. Separate from policy proposals, gate grants, payments and cancellation. | HYB-340 M01; independent AI TW source and rendered context review PASS; native-language acceptance pending |
| charge-approval-policy-proposals / financial_policy_proposals | 청구 승인 정책 제안 | Charge approval policy proposals | مقترحات سياسة اعتماد الرسوم | Pending StageCharge policy proposals setting approval count, required evidence and validity. Not individual charge requests, pricing policies, permission grants, payments or automatic cancellation. | HYB-340 M01 bulk impact; independent AI TW source and rendered context review PASS; native-language acceptance pending |
| input-confirmation-permission-requests / input_authority_proposals | 입력 데이터 권한 요청 | Input data permission requests | طلبات صلاحية بيانات الإدخال | Pending proposals and unresolved applications for separate console/human permissions to confirm synthetic input data or inspect its cost-protected evidence. The impact count is not data approval, financial approval, cancellation or proof of effective permission. | M01 bulk impact; independent AI TW/domain wording review PASS; rendered verification PASS; native-language acceptance pending |
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
| source-interval-envelope / minmax | 원천 기간 범위 | Source period envelope | نطاق الفترة من المصدر | Earliest start through latest end of observed records, not continuous coverage or commitment lifetime. Replaces "source interval envelope" on B06 (HB-159). | B06 |
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
| password-confirmation / reauthentication, step-up (internal aliases), `POST /api/v1/me/reauthenticate` | 비밀번호 재확인 | Confirm your password | تأكيد كلمة المرور | Re-entering the current account's password inside an existing session so a sensitive command can run (HYB-293; sources: `docs/ui/screens/M05/spec.md:47` and `:89` (authorized local workspace, not yet in Git), `docs/planning/v1.16/retention-access.md:117` IAM-08 (authorized local workspace, not yet in Git)). It restarts the "Sign-in valid for" clock: the workspace sign-in age counts from the last sign-in or password confirmation. Five consecutive wrong passwords in one session, counted together with M05 password-change attempts, end that session without locking password sign-in. Not a new sign-in, does not change the password, grants no permission, is not MFA and does not reset a sign-in lockout. | Password confirmation dialog on every command-posting service screen, M02 (sign-in policy) and M05 (my security); KO and AR are AI drafts pending native-language review |
| access-change-request / access approval request, `access.request_grant`, `access.request_revoke` | 권한 변경 승인 요청 | Access change request (section heading "Access change requests") | طلب تغيير الصلاحيات (العنوان: طلبات تغيير الصلاحيات) | A request to add or remove one grant that another administrator approves against the fixed content and the request revision (HYB-298; `docs/ui/screens/M03/spec.md:83`, authorized local workspace). It is not access: nothing changes until it is approved, and an out-of-date request is never applied. The KO form and button for an addition read "권한 추가 요청". | M03 access change requests; KO and AR are AI drafts pending native-language review |
| request-revision / request `digest` | 요청 revision | Request revision | بصمة الطلب | The digest over a request's change and the facts it was judged against; the approver sends it back. Not a version of the person's access and not a preview result. | M03 approve panel and expert detail |
| revocation / revocation record, `revocation-<n>` | 회수 | Revocation | الإبطال (إبطال) | A grant or delegation revocation: the deny is applied first, then the server tries to record the decision in the decision ledger, then cleanup. "Received" and "Decision not confirmed" are not success. The deny revision is رقم إصدار الحجب in AR. | M03 revocations section |
| decision-not-confirmed / `decision_unconfirmed`, grant and delegation state `revocation_unconfirmed` | 결정 기록 미확정 (row: 회수 미확정) | Decision not confirmed (row: Revocation not confirmed) | القرار غير مؤكَّد (الصف: الإبطال غير مؤكَّد) | The deny applies in this server process, but the decision ledger has not confirmed that it recorded the decision. Means neither "access still allowed" nor "revoked"; it may not be re-applied after a restart. | M03 revocation state, grant and delegation rows |
| role-period / `role_from`, `role_until`, `role_period` | 역할 기간 | Role period | مدة الدور | The interval in which a workspace membership's role is in effect: a start set by the server and an optional end. "No end date" is a real value, not a missing one. Not the installation grant's 30-day term and not the session length. | M01 users and membership tables, detail, Change role period form (HYB-299) |
| ends-soon / `role_period=ends_soon` | {n}일 후 종료 | Ends in {n} days / Ends today at {time} | تنتهي بعد يوم واحد / بعد يومين / بعد {n} أيام (3–10) / بعد {n} يومًا (11–99) / بعد {n} يوم | The role period ends within the installation's "ends soon" window, measured from the server time at load. Not a countdown and not an expiry that already happened. | M01 role period column and detail (HYB-299) |
| role-period-ended / `role_period_ended` | 역할 기간 종료 | Role period ended | انتهت مدة الدور | The role period's end was reached; access stopped at that instant and the membership is suspended until a new period is given. Not "removed" and not reversible by Reinstate membership. | M01 rows, detail and refusals (HYB-299) |
| anchor-administrator / administrator without an end date | 종료일 없는 관리자 | Administrator without an end date | مسؤول بلا تاريخ انتهاء | An effective workspace administrator with no role end who still holds a live installation grant; at least one must remain after any period change, role change or bulk revocation. Not "the last administrator" in general and not the installation administrator alone. | M01 refusals, impact preview "Will remain afterwards" and replay notes (HYB-299) |
| bulk-revocation / `bulk-revocation-<n>` | 대량 회수 | Bulk revocation | الإبطال الجماعي | One fixed list of people whose access is revoked together, deny first, then the decision ledger, then cleanup. Not removal or departure, and not an approval request. | M01 Revocation jobs tab (HYB-299) |
| bulk-revocation-list / manifest, `bulk-draft-<n>` | 회수 대상 목록(고정) / 회수 대상 초안 | Prepared list / draft | قائمة الإبطال الثابتة / مسودة الإبطال | The draft collects selected user IDs; preparing freezes it into a list with a digest and an impact preview. Neither changes access. | M01 bulk flow and Revocation jobs (HYB-299) |
| revocation-target / bulk `targets` | 회수 대상 | Target / people to revoke | المستهدف (الجمع: المستهدفون) | A person on a prepared list whose access the bulk revocation suspends. AR uses المستهدفين in sentences about the list (plan wording); the M03 replay note "Target not present after restart" keeps الهدف. Not an excluded person. | M01 preview, refusals and job (HYB-299) |
| held / `held`, `changed_since_preview` | 보류 | Held: changed after preview | مُستبعَد مؤقتًا | A target who changed between preview and execution and was not processed; prepare again to include them. Not "failed" and not "revoked". AR avoids معلّق, which means Suspended on M01. | M01 job outcomes (HYB-299) |
| excluded / `excluded` | 제외 | Excluded | مستثنى | A selected person who is not a target (self, not found, no membership, already revoked). Not "protected"; excluded people are outside the job's denominator. | M01 preview and job (HYB-299) |
| effect-scope / `effect` | 효력 범위 | Effect | الأثر | Which suspension the bulk revocation applies: the membership in this workspace, or the account in every workspace. | M01 bulk flow (HYB-299) |
| bulk-revocation-right / `revoke_access_in_bulk` | 대량 회수 권한 | Bulk revocation right | صلاحية الإبطال الجماعي | An installation-kind right, separate from administer_installation; both are needed for bulk revocation. Granted only by the sample seed and bootstrap in pre-launch. | M01 bulk commands (HYB-299) |
| outside-controlled-scope / cleanup `outside_scope` | 통제 범위 밖 | Outside the controlled scope | خارج النطاق المتحكَّم فيه | A cleanup item the service cannot act on, such as copies already shown or downloaded. Not "cleaned". | M01 job cleanup groups (HYB-299) |
| decision-ledger / `decisions/ledger.jsonl` | 결정 원장 | Decision ledger | دفتر القرارات (قيد / قيود for its records) | The fsynced, hash-chained file in the installation's data directory that records revocation decisions. The chain detects damage; it is not protection, a backup or an off-host copy. | M03 ledger line |
| watermark / `<seq>:<sha256>` | 결정 워터마크 | Confirmed watermark | علامة التأكيد | The sequence and hash of the last record the ledger confirmed. Not a timestamp and not proof that the file was not replaced as a whole. | M03 expert detail |
| installation-setting / `-access-*`, `-decision-ledger-max-mib` flags | 설치 설정 | Installation setting | إعداد التثبيت | A value chosen when the server starts, with a validated range and a default. Not editable on screen and not a workspace policy. | M03 approval settings disclosure |
| outside-controlled-scope / cleanup `outside_scope` | 통제 범위 밖 | Outside the controlled scope | خارج النطاق المتحكَّم فيه | Cleanup items the product cannot reach, such as copies already shown or downloaded. Never counted as cleaned. | M03 revocation cleanup inventory |
| session-access-blocked / self-service session end, cleanup item C1, `POST /api/v1/me/sessions/end` | 접근 차단 | Access blocked | حجب الوصول | The ended session can no longer be used: it is removed from the session store and its sign-in record is ended (or refused through the denied-session guard) before the answer. Not the same as cleanup complete, and not a recall of pages already shown or requests already in progress (HYB-307; sources: `docs/ui/screens/M05/spec.md:72`, `:85`, `:95`). | M05 end other sessions result, "Access blocked" line; KO and AR are AI drafts pending native-language and TW review (AR حجب follows the revocation row) |
| session-cleanup / `cleanup_running`, `complete`, `cleanup_incomplete`; cleanup items C1–C8 | 세션 정리 | Session cleanup | تنظيف الجلسة | Removal of an ended session's leftover records after access is blocked: the sign-in record, secret pickups and the wrong-password count. "Already cleared" (`already_ended`) means the sign-in service had closed the record itself; a session that was already gone before the request is "Had already ended" (`already_gone`), a different fact. Never shown as done while unconfirmed. | M05 result "Cleanup" line, counts and per-session details |
| session-not-included / `not_included` | 확정 목록에 포함되지 않음 | Not included in the confirmed list | غير مشمولة في القائمة المؤكدة | A same-owner recorded session excluded from the confirmed target set and not ended by that operation. Legacy explicit manifests may omit an already-existing session, so exclusion alone does not prove later creation. It does not assert that the session is still active at a later read; known-ID admission rechecks before a new End action. | HYB-304 M05 result and bounded candidate row tag with known-operation recovery; independent source meaning review identified the temporal overclaim; AI wording/native review remains scoped and separate |
| session-list-time / list `now`, `sessions_generation` | 목록 기준 시각 | List taken at | وقت أخذ القائمة | The server reference time of the immutable confirmed selection, shown with seconds. HYB-304 submits its stored reference/digest/count; the opaque legacy `sessions_generation` token is not displayed or used to reconstruct the new selection. | M05 end other sessions dialog |
| session-effective-limits / session query and selection `settings[]` | 조회·목록 준비 유효 한도 | Effective query and preparation limits | الحدود الفعلية للاستعلام والإعداد | Read-only effective server values with units, allowed bounds, defaults and actual source. The installation default page size differs from the user's selected page size. Missing values or sources are unavailable; no local fallback or edit authority is implied. | HYB-304 M05 query and immutable preparation disclosures; AI TW review remains distinct from native/owner acceptance |
| session-record-capacity / `session_record_references` | 공유 기록 용량 | Shared record capacity | السعة المشتركة للسجلات | Shared capacity units for retained records, frozen targets, result and not-included rows, and retry references. The configured budget is not remaining free capacity or a session count. | HYB-304 M05 preparation settings |
| session-preparation-expiry / `expires_at` | 목록 준비 만료 시각 | Preparation expires at | تنتهي صلاحية الإعداد في | Exact server deadline for preparing/submitting an unaccepted list, displayed without losing timestamp precision. Accepted-result retention is a separate availability boundary. Browser time does not authorize admission or replacement after an uncertain submission. | HYB-304 M05 immutable confirmation; server admission remains authoritative |
| secret-pickup / pickup, `GET /api/v1/identity/links/{ref}`, cleanup item C4 | 비밀값 수령 링크 | Secret pickup | استلام السر | A one-time link, bound to the session that requested it, through which a set-password or activation link or a service credential secret is collected once. Ending the session makes its pickups unusable at once and the cleanup deletes them. Not a stored copy of the secret and not a download that can be recalled. | M05 cleanup details (C4); M01 and M04 secret delivery |
| ai-source-body / prompt and response text of AI usage | 프롬프트·응답 본문 | Prompt and response bodies | نصوص الموجّهات والردود | The text sent to and returned by an AI model. AI usage records have no field for it (`modules/tokenmeter` Event). Not "token counts", and not a claim that consumer identifiers are pseudonymous. | P04 deletion storage results (HYB-325); KO and AR are AI drafts pending native-language review |
| assistant-conversations / retention class `conversation` | Assistant 대화 | Assistant conversations | محادثات المساعد | The P04 data class for Assistant conversations, separate from billing records; a deletion run for it deletes only conversations, never the records that refer to them. Not billing evidence, an approval or an audit record; attachments and tool-run references are not modelled in it. | P04 data classes and deletion storage results (HYB-325) |
| plugin / `modules/plugins` | 플러그인 | Plugin | الإضافة | A separately published module that an installation registers, installs and enables under a signed manifest. Not a product component covered by the product's own license, and not an external app opened in its own origin (QA-06). | P02 (HYB-309); AR الإضافة confirmed for consistency with P02; native-language review pending |
| plugin-license / manifest `License` | 플러그인 라이선스 | Plugin license | ترخيص الإضافة | The license under which a plugin publisher offers its module, as one identifier in the signed manifest. Not the product's own license, an installation entitlement (P07) or a Marketplace right, and not a legal compliance finding. | P02 versions and proposals (HYB-309); KO and AR are AI drafts pending native-language review |
| integration-rights / `administer_api_clients`, `administer_modules`, `approve_modules`, `stop_modules`, `administer_channels`, `approve_channels`, `stop_channels`, `reprocess_events` | 연동 권한 | Integration rights | صلاحيات التكامل | Eight installation rights, separate from installation administration, the trust rights and the bulk revocation right; none implies another. Each P02 command and area needs its own right (HYB-310). Granted only by the sample seed and the bootstrap today; enrolment is not decided and M03 cannot grant them. Not a billing permission, not a role, and not confidentiality: a withheld P02 area may still be visible to installation administrators on M03 or M04. The right IDs are provisional (HB-33, `requirements.md:601`). | P02 rights line, withheld areas and reasons (HYB-310); KO and AR are AI drafts pending native-language review |
| right-administer-api-clients / `administer_api_clients` | API client 관리 | API client administration | إدارة عملاء API | Opens the P02 API clients area. No command needs it today; OAuth clients are not connected and service users are managed on M04. | P02 (HYB-310) |
| right-administer-modules / `administer_modules` | 모듈 관리 | Module administration | إدارة الوحدات | Proposing plugin changes. Not approval: someone holding Module approval who is not the proposer must approve. | P02 Modules (HYB-310) |
| right-approve-modules / `approve_modules` | 모듈 승인 | Module approval | اعتماد الوحدات | Approving another person's plugin proposal; the approval closes the proposal instead when its proposer no longer holds Module administration. | P02 Modules (HYB-310) |
| right-stop-modules / `stop_modules` | 모듈 중지 | Module stop | إيقاف الوحدات | Stopping or disabling an installed plugin. Cannot re-enable it; that is a proposal and an approval. | P02 Modules (HYB-310) |
| right-administer-channels / `administer_channels` | 구독·채널 관리 | Subscription and channel administration | إدارة الاشتراكات والقنوات | Proposing webhook subscriptions and requesting a resume. Notification channels for deadlines are not built; this covers webhook subscriptions only. | P02 Subscriptions and channels (HYB-310) |
| right-approve-channels / `approve_channels` | 구독·채널 승인 | Subscription and channel approval | اعتماد الاشتراكات والقنوات | Approving another person's subscription proposal or resume request. | P02 Subscriptions and channels (HYB-310) |
| right-stop-channels / `stop_channels` | 구독·채널 중지 | Subscription and channel stop | إيقاف الاشتراكات والقنوات | Pausing or revoking a subscription. Revoking cannot be undone, so one holder can end a subscription permanently (HYB-310 Q4). | P02 Subscriptions and channels (HYB-310) |
| right-reprocess-events / `reprocess_events` | 이벤트 재처리 | Event reprocessing | إعادة معالجة الأحداث | Requesting and approving redelivery or dismissal of dead letters and re-approving a held redelivery; a request and its approval need two different people. | P02 Events and redelivery (HYB-310) |
| your-pending-items / `own_pending` | 내 대기 항목 | Your pending items | عناصرك المعلّقة | The caller's own pending proposals and requests, which the author may withdraw without the area's right while still an administrator. Lists nothing of anyone else. | P02 withheld areas (HYB-310) |
| held-redelivery / `hold: approver_lost_right` | 보류됨 | Held | محتجَز | A queued redelivery whose approver no longer holds Event reprocessing; it is not dispatched until another holder approves it again. Not sent, not failed. | P02 Events (HYB-310) |
| closed-lost-right / `requester_lost_right` | 종료됨 | Closed | مُغلق | Pending work closed because the person who started it no longer holds the starting right, found when someone tried to approve it. Not withdrawn, not rejected, not applied. | P02 proposals, subscriptions, requests (HYB-310) |
| permitted-plugin-licenses / `-plugin-licenses` flag | 허용 플러그인 라이선스 | Permitted plugin licenses | تراخيص الإضافات المسموح بها | An installation setting: the license identifiers whose plugins this installation may install, upgrade to or enable. Not editable on screen, not a per-plugin exception and not a legal review. | P02 modules view (HYB-309); AR "installation default" is الإعداد الافتراضي للتثبيت (see installation-default); native-language review pending |
| installation-default / settings source `default` | 설치 기본값 | Installation default | الإعداد الافتراضي للتثبيت | The value an installation setting has when the server starts without that setting being given. Not a product recommendation, a legal position or a workspace policy; a value given at start is shown as "set when the server started" (source `start_flag`). | P02 permitted plugin licenses (HYB-309), M02 sign-in rules, M03 settings (aligned in HYB-329); native-language review pending |

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

B01 applies the HB-151 terms above without new concepts: Missing and Exact zero for row amounts, charge period, the derived charge-start month, delivery ID and row ordinal, and sample provenance, which always appears with its field label because its Unclassified value differs from the Unclassified cost category (미분류 / غير مصنّف). Subtotals keep 값 없음 / No values present and are never labeled Exact zero; a subtotal that could not be computed reads 산정 불가 / Unavailable / غير متاح. A short row identity for option text and control labels reads 순번 · 하위 계정 or 결제 계정 · 행 ID / Ordinal · Subaccount or Billing account · Row ID / الترتيب · الحساب الفرعي or حساب الفوترة · معرّف الصف; the row ID is the only unique key. An evidence reference the user has not entered reads 근거 참조 미입력 / No reference entered / لم يُدخل مرجع, not Missing. AI wording review only; Arabic and Korean practitioner review remains open. U01, B04, B05, B06, B07 and B08 were aligned in their own tickets (HB-154 through HB-161).

## B08 use of the source-field terms (HB-154)

B08 applies the HB-151 terms to its source input rows, row detail and account list: Missing, Unit missing and Exact zero for row values, charge period and billing period in UTC, the derived charge-start month, delivery ID and row ordinal, sample provenance, 결제 계정 ID and 공급사. The Exact zero note appears in the row detail and trace view, not in the 25-row table. Totals read 값 없음 / No values present when no value is present and keep 값 있음 / 누락 counts; totals and differences are never labeled Exact zero. Negative amounts, totals and differences use the shared format `USD -1.00`. B08 keeps its basis labels 원천 청구 비용 / 원천 실효 비용 because they mark the amounts as source costs on a billing screen. AI wording review only.

## B03 Korean wording alignment (HB-155)

At the owner's request B03 uses 공급사 for the provider and 금액 누락 행 for rows whose amount is missing, replacing 공급자 and 금액 없는 행. English and Arabic are unchanged. The earlier owner FIX of the Korean policy editor is retained for its source and the changed candidate awaits renewed confirmation ([acceptance record](../acceptance/B03.md)).

## B04 use of the source-field terms (HB-157)

B04 applies the HB-151 terms to its source row picker, evidence disclosure, source summary and source cost evidence: Missing, Unit missing and Exact zero, 원천 과금 기간 / Source charge period in UTC, 결제 계정 ID, 하위 계정 ID, 수집본 내 행 순번, 샘플 출처 and 공급사, replacing 원천 값 없음, 원천 과금 구간 and 공급자. The eligibility column keeps its specific 요금 산정 수량 누락 / 요금 산정 단위 누락. Source amounts use the shared format; B04's computed candidate, adjustment and comparison amounts keep their own reviewed format. AI wording review only. The evidence toggle reads 원천 근거 보기 / 원천 근거 닫기 (Show / Hide source evidence); an instant that is not read as UTC carries 수집된 그대로 표시하며 UTC로 변환하지 않았습니다 / Shown as collected; not converted to UTC, as on B08. The picker's first data column is headed 원천 행 / Source row / صف المصدر and the period lines are prefixed 시작 / 종료 (Start / End). Residual: 원천 근거 보기 is also the Korean label of B08's and B07's row-detail button, whose English reads Inspect source; the two pairings are not yet aligned.

## B05 use of the source-field terms (HB-158)

B05 applies the HB-151 terms to its source row picker, evidence row, source summary, rule editor source line and account scope: Missing and Exact zero, 원천 과금 기간 / Source charge period in UTC, 결제 계정 ID, 하위 계정 ID, 수집본 내 행 순번, 샘플 출처, 공급사 (including 공급사 환산 이력) and 금액 누락, replacing 원천 값 없음, 서비스명 없음, 금액 없음, 원천 과금 구간, 근거 분류, 계정 참조 and 공급자. The validation message for a null amount reads 선택한 원천 비용이 누락되었습니다. B05's 입력하지 않음 / Not entered is the state of a rule field the user has not filled; it is never used for a source value, and Missing is never used for an unfilled field. Source amounts use the shared format; converted, rounded and compared amounts keep B05's own reviewed format. AI wording review only. B05 reuses B04's toggle labels 원천 근거 보기 / 원천 근거 닫기; the residual recorded for B04 about B08 and B07 pairing 원천 근거 보기 with "Inspect source" now covers B05 as well.

## B06 use of the source-field terms (HB-159)

B06 keeps its own commitment field states (열 없음, 빈 원천 값, 원천 null 값, 버전 미매핑, 원천 형식 오류), which say more than Missing. For the charge rows behind a group it applies the HB-151 terms: Missing and Exact zero for the two costs, charge period and billing period start and end in UTC, 수집본 ID, 수집본 내 행 순번, 샘플 출처 and 행 ID, replacing 원천 값 없음, 서비스명 없음, 원천 행 참조, 근거 분류, 수집 배치 and 원천 행 순번; a row whose service is missing is headed 서비스: 누락. Scope identity uses 공급사, 결제 계정 ID, 하위 계정 ID and 계정 범위 ID (derived by Hybrid Billing, replacing 원천 계정 참조). Group envelopes read 원천 과금 기간 범위 / Source charge period envelope / نطاق فترة رسوم المصدر and the billing counterpart; a row has a period, a group has an envelope. An aggregate or metadata set without present values reads 값 없음 / No values present / لا قيم موجودة, counts read 값 있음 / 누락, and aggregates are never labeled Exact zero. B06's amounts, including aggregates, use the shared format `USD -1.00`, because its aggregates are sums of source costs as on B08. 구간 in B06's coverage notes means collection slices and stays. AI wording review only. After HB-161 no sample-backed screen carries the older source wording.

## B07 use of the source-field terms (HB-160)

B07 keeps its own states for the eight marketplace fields (관측 값, 원천 null, 필드 없음, 공백 값, 버전 미매핑, 원천 형식 오류), which say more than Missing. For the plain nullable row values it applies the HB-151 terms: Missing and Exact zero for the two costs, Missing for a null service or subaccount (a card without a service is headed 서비스: 누락), charge period and billing period start and end in UTC, 결제 계정 ID, 하위 계정 ID, 계정 범위 ID, 수집본 ID, 수집본 내 행 순번, 샘플 출처 and 행 ID. These replace 관측 금액 없음, 서비스 값 없음, 원천 값 없음, 청구 계정 / 하위 계정, 원천 계정 참조, 원천 배치 / 행 순번, 원천 출처 분류 and the unlabeled technical source reference; the row ID keeps the sentence that it is not a transaction ID. The account scope reads 계정 범위 1개 선택 / One account scope selected / نطاق حساب واحد محدد and states that the account scope ID is derived by Hybrid Billing. Aggregates without present values read 값 없음 / No values present / لا قيم موجودة, are never labeled Missing or Exact zero, and use the shared format `USD -1.00` because they are sums of source costs. A row whose issuer or publisher name is not an observed value is "counted as insufficient classification evidence", not "unclassified", which is a sample provenance value. Korean copy says 공급사. Residuals: the Arabic provider spelling differs between the shared label (المزود) and B07's toolbar (المزوّد); 원천 근거 보기 names both the row-detail button and the B04 and B05 evidence toggle. AI wording review only.

## U01 use of the source-field terms (HB-161)

U01 applies the HB-151 terms to its source rows: Missing for a null cost, service or subaccount, the shared headers 과금 시작 월, 공급사 / 서비스, 결제 계정 ID / 하위 계정 ID and 샘플 출처 with the provenance label instead of the raw class, and a row control 과금 기간·수집본 식별 정보 / Charge period and delivery lineage / فترة الرسوم وبيانات تتبّع التسليم that shows charge period start and end in UTC, 수집본 ID and 수집본 내 행 순번. These replace the dash, 제공 없음, 출처, 과금 시작, 수집 배치 / 순번 and the raw column names in headers. The 25-row table carries no Exact zero note; zero reads `USD 0.00`.

Aggregates (summary, provider, account, month) read 값 없음 / No values present / لا قيم موجودة without currency when no value is present and are never labeled Missing or Exact zero. A partial sum carries one count format everywhere, 누락 n개 / n missing / المفقود: n; the monthly state reads 부분 · 누락 n개 / Partial · n missing, 모든 값 누락 / All values missing / كل القيم مفقودة or 값 누락 없음 / All values present under the header 값 상태 / Value state / حالة القيم. 미확인 / Unknown is not a value state and is no longer used for amounts.

A month is distinguished from a value: 선택 범위에 행 없음 / No rows in selection / لا صفوف ضمن النطاق when the selection has no row in that month, and CUR 행 없음 / No CUR rows / لا صفوف CUR when it has rows but none in the chart currency. Neither is shown as zero. The counts card is titled 선택 범위 / Selected scope / النطاق المحدد as on S04; 수집 범위 stays the operator's coverage declaration. The provenance line lists each class by its glossary label with its count. Help prose keeps the FOCUS column aliases. AI wording review only; open for native review: 값 없음 beside 값 누락 없음, the Arabic "ضمن النطاق" without "المحدد", and "لا صفوف CUR".

## Still-working line on reader-backed screens (HB-164)

After about four seconds in the loading state, the screens that read from the collected reader add one line under their loading sentence: 아직 조회 중입니다. 먼저 진행 중인 조회가 있으면 차례를 기다린 뒤 조회합니다. / Still working on this query. Other queries may be running first, so this one may be waiting its turn. / ما زال الاستعلام جارياً. قد تكون استعلامات أخرى قيد التنفيذ قبله، فينتظر دوره. It uses the screens' action term 조회 / query / استعلام (as in 새 조회, Refresh query), not "read". It states no duration, because the screen does not know whether its query is running or waiting and the reader's wait is followed by an unbounded query. It is not an error and never appears with an error, an empty result or loaded content. The loading sentences above it keep their own verbs (조회, 불러오는, 확인); one verb across the twelve loading sentences (eleven screens and the import-review collection summary) is left to a writing review. On retained connections and import review the screens' own controls say 새로 고침 / Refresh rather than 조회 / query; the line keeps the shared term there. AI wording review only.

## O02 activation held (HYB-296)

O02 makes amount, publication and ERP impact evidence required before a relationship change can be activated. Until the billing modules produce that evidence, activation is held and the plan can only be discarded. EN is final after the HYB-296 technical-writing review; KO and AR are drafts pending native-language review.

| Concept | KO | EN | AR | Meaning and forbidden implication |
| --- | --- | --- | --- | --- |
| activation-held / impact_not_computed, impact_unavailable | 활성화 보류 | Activation held | التفعيل معلَّق | A plan with locked changes cannot be activated because the impact of the change on amounts, publication and ERP has not been calculated or could not be read. Not a rejection, not an approval wait, not zero impact and not a fault of the proposer. The step state reads 보류 / Held / معلَّقة. |
| changes-locked / sealed | 변경 고정됨 | Changes locked | التغييرات مقفلة | The plan's relationship changes and changed-member diff are locked at proposal. Does not mean its impact on amounts, publication or ERP is locked or calculated. Replaces 영향 고정됨 / Sealed / مثبّتة الأثر on O02. |
| impact-not-calculated / not_computed | 계산되지 않음 | Not calculated | لم يُحتسب | No impact has been calculated for this part (amounts, publication or ERP). Used for the impact row tag and the Impact step note. Never zero and never "no impact". Replaces 미계산 / Not computed / غير محسوب on O02. |
| impact-could-not-be-read / unknown, impact_unavailable | 확인하지 못함 | Could not be read | تعذّرت القراءة | The impact evidence for this plan could not be read, so whether it exists is unknown. Used for the impact row tag and the Impact step note. Not "not calculated" and not zero. Replaces 알 수 없음 / Unknown / غير معروف on O02 impact rows. |

## M02 trust change terms (HYB-297)

[HYB-297](https://linear.app/hybrid-billing/issue/HYB-297) replaces the M02 propose-then-approve flow with the trust change workflow. KO and AR are AI drafts awaiting the technical-writing review; the main KO and AR surface says 신뢰 버전 / إصدار الثقة; "epoch" is kept only in the version label and the technical details (review decision C16, 2026-11-18).

| Concept | KO | EN | AR | Meaning and forbidden implication |
| --- | --- | --- | --- | --- |
| trust-change / trust.* | 신뢰 변경 | Trust change | تغيير الثقة | A proposed sign-in rule configuration bound to an old trust version and digests. A draft, validation or approval never changes the active rules; only activation does. |
| trust-version / epoch | 신뢰 버전(epoch); 본문은 신뢰 버전 | Trust version (epoch) | إصدار الثقة (epoch)؛ في النص إصدار الثقة | Shown as "g.n" from the ledger generation and the number within it. Never formatted from the raw decimal value; a rollback gets a newer version, never an older one. |
| validate / trust.validate | 검증 | Validate | التحقق | Records evidence (test login, recovery path, not-applicable items) for the exact change. Not approval and not activation; a blocked result keeps the change a draft. |
| independent-approval / trust.approve | 독립 승인 | Independent approval | الاعتماد المستقل | Approval by a holder of the approval right who did not propose the change. Cleared when the change is revised or re-validated with different evidence; unusable for activation while the evidence is stale. Not activation. |
| activate / trust.activate | 활성화 | Activate | تفعيل | Commits the change to the decision ledger first, then applies it under a newer trust version and signs out everyone in the workspace. Done by a third person. |
| supersede / superseded | 대체됨 | Superseded | مُستبدَل | Another change was activated on the same old version, so this one can no longer be activated. Not a rejection. |
| invalidation-progress | 무효화 진행 상황 | Invalidation progress | تقدّم إلغاء الصلاحية | Per-item result of ending old sessions and evidence after activation (done, in progress, unknown, not applicable, outside scope). Unknown is never shown as done. |
| activation-unknown / activation_unknown | 활성화 확인 안 됨 | Activation not confirmed | لم يُؤكَّد التفعيل | The server could not confirm that the activation record was saved. Never labelled activated; resolved only at the next server start. |
| trust-hold | 신뢰 보류 | Trust hold | حجز الثقة | After an unconfirmed activation, only access reductions, stops of running operations, withdrawals, retention holds, ending sessions, password changes and already issued reset links work until restart. |
| trust-rights / trust_propose, trust_validate, trust_approve, trust_activate | 신뢰 권한 | Trust rights | صلاحيات الثقة | Four installation rights separate from installation administration; neither implies the other. Granted only by the sample seed and the bootstrap today; enrolment is not decided. |
| blocked (validation result) | 차단됨 | Blocked | يمنع المتابعة | A validation check that holds the change as a draft; it cannot be approved or activated until a validation passes. |
| activation-attempted-by | 활성화 시도자 | Activation attempted by | مَن حاول التفعيل | The administrator who ran an activation that could not be confirmed. Never "activated by". |

## O01 filters, search and paging (HYB-300 part A)

O01 reads its accounts, groups and history views as server pages filtered by legal entity, relationship kind, unattributed state and a name or ID search. The provider filter has no producer yet and is stated as text. EN follows the HYB-300 plan, visual spec and technical-writing review; KO and AR are drafts pending native-language review. Korean uses 공급사 for the provider, as on B03 (HB-155), including the O01 scope label.

| Concept | KO | EN | AR | Meaning and forbidden implication |
| --- | --- | --- | --- | --- |
| legal-entity-filter / entity | 법인 | Legal entity | الكيان القانوني | Keeps members whose dated organization hierarchy on the as-of date leads to this legal entity. Not the payer and not the registered parent. A suspended entity can still be chosen for a past date and reads "{name} · 정지됨 / Suspended". |
| relationship-kind / kind, rk | 관계 종류 | Relationship kind | نوع العلاقة | One of the five relationship timelines. In the accounts view it chooses the extra column and the timeline the unattributed filter reads; on its own it does not narrow the account list, so an empty list is never blamed on it. |
| only-unattributed / unattributed | 미귀속만 | Only unattributed | غير المُسندة فقط | Members with no target of the chosen relationship kind on the as-of date, including source accounts that have no timeline of that kind. Not "no relationship ever" and not zero. Not offered in the groups view, where the not-in-group list already shows these accounts. |
| organization-since | 조직 귀속 시작일 | Organization since | بداية الانتساب للمؤسسة | The start of the hierarchy entry that applies on the as-of date. Replaces 관계 시작일 / Entry from on O01. |
| provider-filter-not-connected / provider_filter not_connected | 공급사 — 아직 사용할 수 없습니다. 원천 계정과 공급사 연결이 연계되어 있지 않습니다. | Provider — Not available yet: source accounts are not linked to provider connections. | المزوّد — غير متاح بعد: حسابات المصدر غير مرتبطة باتصالات المزوّدين. | A text line, never a disabled control: no source-account-to-provider link exists. Both reads refuse a provider parameter. |
| applied-filter-chip | 적용한 필터 / 필터 제거: {chip} / 필터 모두 제거 | Applied filters / Remove filter {chip} / Clear all filters | عوامل التصفية المطبّقة / إزالة عامل التصفية {chip} / مسح كل عوامل التصفية | A removable chip for each non-default filter. The as-of date is scope, not a filter, and is never a chip. In the groups view chips read 구성원: / Members: for the legal entity and 그룹: / Groups: for the group search. |
| shown-count / next, scan_stopped | {n}개 표시, 더 있을 수 있음 / {n}개 표시, 목록 끝 / 그룹 {n}개 표시, … | {n} shown; more may exist / {n} shown; end of list / Groups shown: {n}; … | المعروض: {n}؛ قد يوجد المزيد / المعروض: {n}؛ نهاية القائمة / المجموعات المعروضة: {n}؛ … | Counts only the rows shown. Never a total, "of N" or a percentage. With no rows no count is shown and the live region says 표시된 항목이 없습니다 / No items shown / لا توجد عناصر معروضة. Korean counts accounts with 개, never 명. |
| searched-up-to / scan_stopped | {id}까지 검색했습니다. / 이어서 검색 | Searched up to {id}. / Continue searching | تم البحث حتى {id}. / متابعة البحث | The server stopped a filtered read after its scan budget; the ID is the last member examined. Not "no results": with zero rows the screen says 지금까지 일치하는 항목이 없습니다 / No matches yet / لا توجد نتائج مطابقة حتى الآن. |
| filtered-empty | {date} 기준 허용 범위에서 이 필터에 맞는 항목이 없습니다. | No items match these filters in your permitted scope on {date}. | لا عناصر تطابق عوامل التصفية هذه ضمن نطاقك المسموح في {date}. | The whole permitted set was read and nothing matches the narrowing filters. Distinct from the unfiltered empty sentences, from a group with no members (이 그룹에 구성원이 없습니다 / No members in this group) and from a stopped scan. |
| all-members-on / members_on | {date} 기준 전체 구성원 | All members on {date} | كل الأعضاء في {date} | A group's member count on the date, never reduced by the member filters. |
| bad-link-filter | 링크의 필터 중 올바르지 않은 값을 제거했습니다. | One or more filters in the link were not valid and were removed. | أُزيل من الرابط ما كان غير صالح من عوامل التصفية. | A link value the screen or the server refused (malformed, not a legal entity of this workspace, a malformed member ID). Never reported as a revision that could not be read. |

## O01 create change plan handoff (HYB-318, HYB-300 part B)

O01 hands a member, its relationship kind, the as-of date and the relationship revision it was read at to O02. O02 prefills the form, keeps the revision as the change's base and moves it only when the administrator chooses "Base on revision {a}". EN is final after the HYB-318 technical-writing review; KO uses 리비전 as elsewhere on O01 and O02 (root decision R11) and names the destination 조직 변경 계획; KO and AR are drafts pending native-language review. A revision number and its KO particle are never split across lines.

| Concept | KO | EN | AR | Meaning and forbidden implication |
| --- | --- | --- | --- | --- |
| create-change-plan | 변경 계획 만들기 | Create change plan | إنشاء خطة تغيير | Opens Organization changes (조직 변경 계획) with the selected member, its relationship, the relationship revision and, when it is today or later, the as-of date filled in. Navigation, not a command: no plan exists until Propose succeeds. Never shown on a legal entity, a group or a selection that is not loaded. |
| plan-base / base | 기준 관계 리비전 | Based on relationship revision | مراجعة العلاقات المتخذة أساسًا | The relationship revision the change is based on, sent with the proposal; the server refuses the proposal when it is no longer the active revision. Not the API revision of the screen and never moved silently. Not active: 활성 아님(활성 리비전 {a}). |
| base-changed / revision_conflict on base | 기준 변경됨 | Base changed | تغيّر الأساس | The active relationship revision differs from the one the change is based on. The change is proposed only after "Base on revision {a}". The entries are kept. |
| use-base | 리비전 {a} 기준으로 하기 | Base on revision {a} | اتخاذ المراجعة {a} أساسًا | The only way to move the base to the active revision. Does not change the form entries and does not count as unsaved input. Announced as "The change is now based on relationship revision {a}." |
| prefilled-from-organizations | 조직·관계 탐색에서 가져온 값입니다. 관계 리비전 {r} 기준입니다. 변경 방식과 연결 대상을 고르세요. | Prefilled from Organizations and relationships, based on relationship revision {r}. Choose the change and the target. | قيم مأخوذة من شاشة «المؤسسات والعلاقات»، استنادًا إلى مراجعة العلاقات {r}. اختر التغيير والجهة المستهدفة. | The kind, member and effective date came from the O01 link. The change and the target are never prefilled. |
| prefilled-without-base | 조직·관계 탐색에서 가져온 값입니다. 변경 방식과 연결 대상을 고르세요. | Prefilled from Organizations and relationships. Choose the change and the target. | قيم مأخوذة من شاشة «المؤسسات والعلاقات». اختر التغيير والجهة المستهدفة. | The link carried values but no valid revision, so no base is carried and the server applies its usual active-revision check. |
| link-effective-date-passed | 링크의 적용 시작일({d})이 지나 입력하지 않았습니다. 오늘(UTC) 이후 날짜를 고르세요. | The effective date in the link, {d}, has passed, so it was not filled in. Choose today or a later date (UTC). | تاريخ السريان الوارد في الرابط ({d}) قد مضى، لذا لم يُملأ. اختر تاريخ اليوم أو تاريخًا لاحقًا (UTC). | The date field is left empty; the date is never moved to today silently. |
| link-value-removed | 링크에서 올바르지 않은 값을 제거했습니다. | A value in the link was not valid and was removed. | أُزيلت من الرابط قيمة غير صالحة. | An unknown kind, a malformed member ID, a malformed date or a non-canonical revision in an O02 link. A removed base means no base is carried. |

## O01 view group drafts (HYB-319, HYB-300 part C)

A view group draft is a working list of source accounts for one view group. It grants nothing and changes no relationship; members join only when the change plan created from it is activated, and activation is held until impact evidence exists (HYB-296). KO and AR are drafts for the technical-writing review; KO counts accounts with 개.

| Concept | KO | EN | AR | Meaning and forbidden implication |
| --- | --- | --- | --- | --- |
| view-group-draft / view_group.draft | 조회 그룹 초안 | View group draft | مسودة مجموعة عرض | A server record of a group (new or existing) and the source accounts proposed to join it. Not a group, not membership and not access. Never say members "join" or "get access" on save. |
| draft-name | 초안 이름 | Draft name | اسم المسودة | The name proposed for a new group. It becomes the group's label only when the plan from that draft is activated; until then relationship views (relationship reads, targets, the O02 plan lead) show the group ID, while the drafts list and the draft panel title show the draft name. Never shown as a group name outside draft and plan contexts. A draft on an existing group has no name and cannot rename it. |
| draft-state | 작성 중 / 변경 계획 열림 / 활성화됨 / 폐기됨 | Open / Change plan open / Activated / Discarded | مفتوحة / خطة تغيير مفتوحة / مُفعّلة / مُتجاهَلة | Derived from the latest plan created from the draft. "Open" is editable and counts toward the open-drafts limit; the others are not editable. The sub-line reads "View group draft · {state}", so the state never repeats the noun. |
| remove-from-draft | 초안에서 빼기 | Remove from draft | إزالة من المسودة | Takes an account out of the draft only. Not removal from a group; leaving a group stays the O02 single-member "leave" change (Q3). |
| start-view-group-draft | 조회 그룹 초안 시작 | Start a view group draft | بدء مسودة مجموعة عرض | Moves the accounts checked on the loaded page into a new draft form. Selection is page-bounded; there is no "select all matching". |
| page-selection-count | 표시된 {k}개 중 {n}개 선택 | {n} selected of {k} shown | تم تحديد {n} من {k} معروضة | Counts loaded rows only; never a filter total. |
| view-group-not-ownership | 조회 그룹은 소유권이 아닙니다… | A view group is not ownership. Creating or editing a draft grants no access and changes no relationship. Members join only when a change plan is activated. | مجموعة العرض ليست ملكية… | Persistent in the draft form, the draft panel and the O02 draft handoff. Not dismissible. |
| group-draft-handoff | 조회 그룹 초안의 변경 계획 | Change plan for a view group draft | خطة تغيير لمسودة مجموعة عرض | The O02 confirmation before Propose. Effect sentence (EN final): "Proposing creates one change plan that, if activated, adds all {n} members to this group from the effective date. If the plan passes its checks, activation is held: the impact of this change on amounts, publication and ERP has not been calculated." Never says the plan is held when it may be rejected, and never implies that proposing adds members. |
| group-plan-lead | 조회 그룹 {group}에 추가: {date}부터 구성원 {n}개 | Join view group {group}: {n} members from {date} | الانضمام إلى مجموعة العرض {group}: عدد الأعضاء {n}، اعتبارًا من {date} | {group} is the group label, which is the ID for a new group before activation; the draft name follows only as "Draft name: {name}". |
| draft-member-counts | 저장된 구성원 {n}개 / 변경 후 {n}개 · 최대 {max}개 / 추가할 구성원 {n}개 · 최대 {max}개 | {n} saved in this draft / {n} after your changes · up to {max} / {n} to add · up to {max} | المحفوظ في هذه المسودة: {n} / بعد تغييراتك: {n} · الحد {max} / المراد إضافتهم: {n} · الحد {max} | The saved count, the count after unsaved changes (existing draft) and the count in the new-draft form. Never two different numbers under the same "in this draft" wording. |
| partial-save | 구성원 {total}개 중 {done}개를 초안에 저장했습니다… / 변경 {total}건 중 {done}건을 저장했습니다… | {done} of {total} members were saved to the draft. {rest} were not saved. / {done} of {total} changes were saved. {rest} were not saved. | الأعضاء المحفوظون في المسودة… / التغييرات المحفوظة… | Members for a new draft, changes for an edit. The reason line never says "nothing was saved". KO counts changes with 건 and members with 개. |
| view-group-plan-held | 활성화 보류 | Activation is held | التفعيل معلَّق | View group plans use the HYB-296 held wording ("the impact of this change on amounts, publication and ERP has not been calculated") and the destination name "Organization changes". |
| members-not-in-group | {d} 기준 조회 그룹에 없는 구성원 | Members not in a view group on {d} | الأعضاء خارج مجموعات العرض في {d} | List heading for accounts with no view group on the date (pool: "Members not in an allocation pool on {d}"). |

## M01 users search, filters and paging (HYB-294 part A)

M01 reads its users, membership and invitations tabs as server pages in login order, filtered by account status, membership in this workspace, workspace role, role period, sign-in setup and installation administrators, with a search over login, contact email and user ID. The sign-in method line is built from the read's `auth_methods`. EN follows the HYB-294 plan, the visual spec and the HYB-294 technical-writing review; rows marked draft in that review and all KO and AR wording are AI drafts pending native-language review. Korean counts people with 명 on M01.

| Concept | KO | EN | AR | Meaning and forbidden implication |
| --- | --- | --- | --- | --- |
| login / `login` | 로그인 ID | Login | اسم الدخول | The identifier a person signs in with; usually an email address, but the bootstrap login need not be one. Distinct from the contact email. Bare KO 로그인 means the act of signing in and is not used for the identifier. The Create user form deliberately keeps the narrower 로그인 이메일 / Sign-in email / البريد الإلكتروني لتسجيل الدخول, because a new login must be an email address. |
| workspace-scope-line | 업무 공간: {id} | Workspace: {id} | مساحة العمل: {id} | The caller's workspace, fixed by the server. Not a control and not a filter; the read refuses a workspace parameter. The ID is shown because the read returns no workspace name. |
| users-search | 로그인 ID·연락용 이메일·사용자 ID로 검색 | Search by login, contact email or user ID | البحث باسم الدخول أو البريد الإلكتروني للتواصل أو معرّف المستخدم | Case-insensitive search on the server over the three fields. A departed person can be found only by user ID. |
| membership-filter / membership, ms | 이 업무 공간 소속: 활성 / 정지됨 / 역할 기간 종료(정지 처리 대기) / 소속 없음 | Membership in this workspace: Active / Suspended / Role period ended, suspension pending / No membership | العضوية في مساحة العمل هذه: نشطة / معلّقة / انتهت مدة الدور (بانتظار التعليق) / بلا عضوية | The four values partition every person once. Short options in the filter; AR uses the feminine نشطة / معلّقة to agree with العضوية only in this filter, and table cells keep 소속 정지됨 / Membership suspended. |
| no-role-cell / Membership tab role cell | 역할 없음 (visually hidden, after "—") | No role (visually hidden, after "—") | بلا دور (visually hidden, after "—") | [TW] HYB-331. Screen-reader word for the Membership tab's role cell when the person has no membership in this workspace; the visible cell shows "—" because the membership status column beside it already says "No membership". Not a role value and not a filter option; other tabs and the row detail keep "No membership". |
| membership-ended / membership=ended | 역할 기간 종료(정지 처리 대기) | Role period ended, suspension pending | انتهت مدة الدور (بانتظار التعليق) | The role period's end has passed, so the membership is no longer effective; the server suspends it automatically. Not an action the administrator must take and not continuing access. Also the membership status word for such a row. |
| period-bounded / period=bounded | {n}일 이후 종료 | Ends in more than {n} day(s) | تنتهي بعد أكثر من {n} يومًا (AR plural forms) | An end date beyond the "ends soon" window. A person ending sooner matches "Ends within {n} days" instead. |
| period-ends-soon / period=ends_soon | {n}일 안에 종료 | Ends within {n} day(s) | تنتهي خلال {n} يومًا (AR plural forms) | The end falls inside the installation's warning window (`role_period_warning_days`). |
| sign-in-setup-cell | 비밀번호 설정 링크 발급됨 · {time} 만료 / 비밀번호 재설정 링크 발급됨 · {time} 만료 | Password set link issued, expires {time} / Password reset link issued, expires {time} | صدر رابط تعيين كلمة المرور، ينتهي في {time} / صدر رابط إعادة تعيين كلمة المرور، ينتهي في {time} | The Users tab's sign-in setup cell and the detail; the time is in its own date and time isolates. |
| sign-in-setup-filter / signin, si | 로그인 설정: 비밀번호 설정됨 / 비밀번호 설정 링크 발급됨 / 비밀번호 재설정 링크 발급됨 / 비밀번호·링크 없음 | Sign-in setup: Password set / Password set link issued / Password reset link issued / No password and no link | إعداد تسجيل الدخول: تم تعيين كلمة المرور / صدر رابط تعيين كلمة المرور / صدر رابط إعادة تعيين كلمة المرور / لا كلمة مرور ولا رابط | Describes setup, not the sign-in method. |
| sign-in-method-line / auth_methods | 로그인 방식 | Sign-in method | طريقة تسجيل الدخول | A text line built from the connected and not connected methods the server reports. Today: everyone signs in with an installation password; SSO (SAML, OIDC) and LDAP are not connected (AR الدخول الموحد). Never a disabled control. KO particles follow the final sound of the word before any parenthesis (은/는, 와/과); AR agreement follows the number of items (غير متصل, dual غير متصلَين, three or more غير متصلة). |
| installation-administrators-only / holder, adm | 설치 관리자만 | Installation administrators only | مسؤولو التثبيت فقط | People who hold a live installation grant. Not the workspace administrator role. |
| membership-chip-suffix | (소속이 있는 사람) | (people with a membership) | (الأشخاص الذين لديهم عضوية) | On the membership tab, a role or role period filter only matches people with a membership. |
| users-order-line / order | 로그인 ID순. 퇴사한 사람은 맨 뒤. | By login. Departed people last. | حسب اسم الدخول. يأتي المغادرون أخيرًا. | The only order; there are no sort controls. Departed people have no login and follow by user ID. |
| users-shown-count / next | {n}명 표시, 더 있을 수 있음 / {n}명 표시, 목록 끝 | {n} shown; more may exist / {n} shown; end of list | المعروض: {n}؛ قد يوجد المزيد / المعروض: {n}؛ نهاية القائمة | Counts only the rows shown. Never a total, "of N" or a percentage. With no rows no count is shown and the live region says 표시된 사람이 없습니다 / No people shown. / لا يوجد أشخاص معروضون. |
| users-list-updated | 목록을 새로 고쳤습니다. {count}. 적용된 필터 {k}개. | List updated. {count}. Filters applied: {k}. | تم تحديث القائمة. {count}. عوامل التصفية المطبّقة: {k}. | Live-region announcement after a filter change. |
| users-searched-up-to / scan_stopped, next_login | {login}까지 검색했습니다. / 사용자 {id}까지 검색했습니다. / 이어서 검색 | Searched up to {login}. / Searched up to user {id}. / Continue searching | تم البحث حتى {login}. / تم البحث حتى المستخدم {id}. / متابعة البحث | The read stopped at its scan budget; the login or ID is the last person examined. With no rows the screen says 지금까지 일치하는 사람이 없습니다 / No matches yet. / لا توجد نتائج مطابقة حتى الآن., never the filtered empty sentence. |
| users-filtered-empty | 허용 범위에서 이 필터에 맞는 사람이 없습니다. | No people match these filters in your permitted scope. | لا يوجد أشخاص يطابقون عوامل التصفية هذه ضمن نطاقك المسموح. | The whole permitted list was read and nobody matches. Distinct from 아직 사용자가 없습니다 / No users yet. and from a stopped scan. |
| users-list-changed / list_version | 첫 페이지 이후 목록이 바뀌었습니다. … | The list changed after the first page. … | تغيّرت القائمة بعد الصفحة الأولى. … | A later page was read at a different list version; the rows are kept and Reload gives a consistent list. Changes from time passing alone are not detected, and the copy says so. |
| users-start-again / revision_conflict on after | 페이지를 넘기는 동안 목록이 바뀌었습니다. 첫 페이지부터 다시 시작하세요. / 처음부터 다시 | The list changed while you were paging. Start again from the first page. / Start again | تغيّرت القائمة أثناء التنقل بين الصفحات. ابدأ من جديد من الصفحة الأولى. / البدء من جديد | The person the next page starts after has departed, so the cursor no longer exists in the login order. |
| departed-row | 퇴사(로그인 ID·이메일 삭제됨) | Departed (login and email removed) | مغادر (أُزيل اسم الدخول والبريد الإلكتروني) | A departed person's login and contact email were removed at departure; the row shows the user ID only. |
| invitation-set-link-cell | 발급됨 · {time} 만료 / 발급됨 · 초대와 함께 만료 | Issued, expires {time} / Issued, expires with the invitation | صادر، ينتهي في {time} / صادر، وينتهي مع الدعوة | The Invitations tab's set-link column. The second form is used when the link and the invitation expire at the same instant; the two are separate facts. |
| users-jobs-note | 필터는 '{사용자}', '{초대}', '{소속·역할}' 탭에 적용됩니다. | Filters apply to the {Users}, {Invitations} and {Membership} tabs. | تُطبَّق عوامل التصفية على علامات التبويب «…» و«…» و«…». | Shown on Revocation jobs when the link carries filters. The tab names are taken from the same source as the tab labels. |
| action-result-kept | {login}: {결과} | {login}: {result} | {login}: {النتيجة} | When an action's reload removes the open person from the list, the result stays as a status line naming the person (login in its own LTR isolate, or the user ID when there is no login), and the live region says "{login}: {result}." followed by the open-person sentence. Cleared on a tab change, a new filtered read or opening another row. |
| open-person-filtered | 열어 둔 사람이 필터 결과 목록에 없습니다. / 열어 둔 사람이 불러온 행에 없습니다. | The person you had open is not in the filtered list. / The person you had open is not in the loaded rows. | الشخص الذي فتحته ليس في القائمة المصفّاة. / الشخص الذي فتحته ليس ضمن الصفوف المحمَّلة. | Announced when a reload closes the detail of a person outside the new rows; the first form when filters are applied, the second otherwise. Not a claim that the person does not match. |

## M01 select everyone matching (HYB-323, HYB-294 part B)

M01 can fix everyone who matches the current filters into one revocation list on the server. EN follows the HYB-294 plan copy table, the HYB-323 plan and the HYB-323 visual addendum; every KO and AR string, and the EN strings the addendum marks [TW], are AI drafts pending the technical-writing and native-language review. Words about time say "at {time}" and never imply that execution re-applies the filters.

| Concept | KO | EN | AR | Meaning and forbidden implication |
| --- | --- | --- | --- | --- |
| select-matching / `bulk_revocation.add_matching` | 이 필터에 맞는 사람 모두 선택 / 목록의 모든 사람 선택 | Select everyone who matches these filters / Select everyone in the list | تحديد كل من يطابق عوامل التصفية هذه / تحديد كل من في القائمة | Fixes everyone who matches on the server into a new list. Separate from selecting loaded rows; the browser never carries the people. The second form is used when no filter is applied. |
| match-count / `GET /identity/users/count` | 조건에 맞는 {n}명 모두 선택 / 조건에 맞는 사람 모두 선택(최소 {n}명) | Select all {n} who match / Select all who match (at least {n}) | تحديد كل المطابقين ({n}) / تحديد كل المطابقين ({n} على الأقل) | A bounded count before fixing. "At least" when the count stopped at its budget. Never a total; no number while counting and none when too many match. |
| fixed-list / `selection: filter` | {time}에 이 필터로 {n}명을 고정했습니다 | {n} people fixed at {time} for these filters | تم تثبيت {n} في {time} لعوامل التصفية هذه | The set of people and the filters as they were at the fixed time. Changing the filters on the screen does not change it, and people who match later are not added. |
| fixed-when-counted | {time}에 {m}명을 고정했습니다(집계 당시 {n}명) / …(집계 당시 최소 {n}명) | {m} people fixed at {time} ({n} when counted) / (at least {n} when counted) | تم تثبيت {m} في {time} (كان العدد {n} عند العدّ) / (كان العدد {n} على الأقل عند العدّ) | The fixed number differs from an exact count just before, or is below a count that stopped at its budget ("or more"). |
| no-longer-matches / `no_longer_matches` | 더 이상 필터에 맞지 않음 / {k}명은 더 이상 필터에 맞지 않아 제외됩니다. | No longer matches the filters / {k} person no longer matches … and is excluded; {k} people no longer match … and are excluded. | لم يعد يطابق عوامل التصفية / (AR plural forms) | An exclusion reason found when the list is previewed. The person stays in the fixed list and is excluded; not a claim about the time of execution. |
| selection-basis | 직접 선택 / 필터에 맞는 모든 사람 / 선택 기준 기록 없음 | Chosen people / Everyone matching filters / Selection basis not recorded | أشخاص مختارون / كل المطابقين لعوامل التصفية / أساس التحديد غير مسجّل | How a revocation list was built. A job rebuilt from the decision ledger after a restart shows "not recorded", because the ledger keeps no basis. |
| search-applied / `search_masked` | 검색 조건 적용됨 | Search applied | تم تطبيق بحث | Shown to anyone but the preparer instead of the search text. A display rule, not secrecy; never described as private or protected. |
| anchor-dead-end | 이 목록으로는 접근을 관리할 관리자가 남지 않습니다. 필터를 좁히세요. 예: {업무 공간 역할: 구성원} | This list would leave no administrator who can manage access. Narrow the filters, for example {Workspace role: Member}. | لن يبقى بهذه القائمة أي مسؤول يمكنه إدارة الوصول. ضيّق عوامل التصفية، مثل: {…} | Since HYB-324 the owner can also remove one such administrator from the filter list; the screen adds a second sentence saying so (`match-copy.ts` `blockedRemove`) and a way back to the list. Narrower filters and discarding the list remain. The example uses the rendered role option label. |
| list-closed | 이 목록은 더 이상 준비할 수 없습니다. … | This list can no longer be prepared: it expired, or nobody in it can be revoked now. Discard it and select again. | لم يعد ممكنًا تحضير هذه القائمة … | The draft expired, or every person in it is excluded at preview. |
| discard-list | 목록 버리기 | Discard list | تجاهل القائمة | Drops the fixed list; nothing was revoked. KO uses 버리기, as the module's other discard actions do. |
| base-revision / `fixed_revision` | 기준 개정 번호 | Base revision | رقم التعديل الأساسي | The server revision when the list was fixed; part of the list's digest. |
| settings: max matching, retained people, scan budgets | 필터 전체 회수당 인원 / 메모리에 보관하는 회수 목록 인원 / 검색 페이지당 확인 인원 / 집계당 확인 인원 | People per filter-wide revocation / People kept in revocation lists in memory / People examined per search page / People examined per count | الأشخاص لكل إبطال حسب التصفية / الأشخاص المحفوظون في قوائم الإبطال في الذاكرة / الأشخاص المفحوصون لكل صفحة بحث / الأشخاص المفحوصون لكل عدّ | Changeable installation defaults (start flags). The retained-people hint says "lists waiting to run", never "pending". AR uses «القوائم بانتظار التنفيذ» for lists waiting to run in the setting name, the hint and the pending-limit refusal (R-C4); «قيد الانتظار» stays for pending work. |


## Personal access history (HYB-306)

Candidate labels from the stable plan and M05 implementation; combined technical-writing and native-language review remain pending.

| Concept | Korean candidate | English candidate | Arabic candidate | Meaning and exclusions |
| --- | --- | --- | --- | --- |
| recent-access-history | 최근 접근 기록 | Recent access history | سجل الوصول الأخير | Own recorded sign-in, password-confirmation and session events. Not an administrator audit of other people. |
| history-recording-stopped | 접근 기록 저장 중지 | Access event recording stopped | توقّف تسجيل أحداث الوصول | Later events may be missing. Never shown as zero events or a successful empty period. |
| history-deletion-deferred | 서버 시각 확인까지 삭제 보류 | Deletion paused until server time is confirmed | أُوقِف الحذف مؤقتًا حتى يُؤكَّد وقت الخادم | Old events remain hidden; this does not assert physical deletion. |
| history-held | 법적 보류 때문에 보관 | Kept because of a legal hold | محفوظ بسبب حجز قانوني | Own older retained events are not displayed; backup expiry is a separate boundary. |
| access-history-period | 최근 접근 기간 | Access history period | فترة سجل الوصول | Server-time period filter over the caller’s retained workspace events; not a retention-setting change. |
| access-history-note | 접근 기록 안내 | Access history notice | تنبيه سجل الوصول | Recording, loss, hold or deletion fact. Unknown counts remain unknown; a notice is not a confirmed event total. |

## Credential pickup and unknown outcomes (HYB-333)

Candidate copy from the stable HYB-333 plan and implemented M04 messages. Semantic and regional review is deferred to the combined review; no native-language approval is claimed.

| Concept | Korean candidate | English candidate | Arabic candidate | Meaning and excluded implications |
| --- | --- | --- | --- | --- |
| credential-secret-pickup-state / `open` | 비밀 값 미수령 · 수령 가능 | Secret not collected yet · pickup open | لم تُستلَم القيمة السرية بعد · الاستلام متاح | Pickup is still available to the issuing session at this read; its end is estimated and can become earlier. |
| credential-secret-pickup-state / `collected` | 비밀 값 반환됨 | Secret returned once | أُعيدت القيمة السرية مرة واحدة | Server returned the secret once. Does not prove delivery to the person or storage. This state adds no status sub-line. |
| credential-secret-pickup-state / `never_collected` | 비밀 값 수령 안 됨 | Secret never collected | لم تُستلَم القيمة السرية | Pickup ended without the server returning the secret. Credential remains active until separately revoked or expired. |
| issue-outcome-unknown | 발급 결과 확인 안 됨 | Issue outcome unknown | نتيجة الإصدار غير معروفة | Request may have committed. Same-request retry retains its original identity; never call it failed or definitely issued. |


## Diagnostic request identity (HYB-330)

| Concept | Korean candidate | English candidate | Arabic candidate | Meaning and exclusions |
| --- | --- | --- | --- | --- |
| server-error-code | 서버 오류 코드 | Server error code | رمز خطأ الخادم | Actual bounded response code, never a local failure category or a guessed HTTP-status mapping. |
| server-request-id | 서버 요청 ID | Server request ID | معرّف طلب الخادم | Validated `request_id` returned in the error JSON. Does not prove that an action completed. |
| client-request-id | 클라이언트 요청 ID | Client request ID | معرّف طلب العميل | Identifier attached to the browser's original attempt, separately labelled even when the response is missing. |
| diagnostic-action-id | 작업 ID | Action ID | معرّف الإجراء | Canonical command identifier used to distinguish retained references for multiple actions on the same target. Does not grant permission. |

Microsoft's [English notification-reference guidance](https://learn.microsoft.com/en-us/entra/identity/enterprise-apps/troubleshoot-password-based-sso#essential-notification-items) and [Korean version](https://learn.microsoft.com/ko-kr/entra/identity/enterprise-apps/troubleshoot-password-based-sso#중요-알림-항목), inspected 2026-11-20, explicitly distinguish a browser-made client request ID from other notification identifiers. Adopt that distinction; reject their decorative error-colour example and identity data collection, which are outside this product scope. The Arabic `ar-sa` page returned an English body at inspection and is not evidence of Arabic label approval. Arabic candidates reuse the P05 diagnostic concepts and remain pending native/practitioner review; model TW/render review is recorded separately.

## HYB-327 field-binding review

- **API clients / API 클라이언트 / عملاء API** identifies authentication/integration clients, not billable business customers. Denied and unconfigured states retain the API qualifier even without a section heading.
- **Failures for this build since budget approval** describes the server counter. The adjacent qualification states that budget approval is not connected and zero proves neither approval nor a passing run. Arabic uses the existing build term `البنية`.
- **Rules tightened by the restored/default comparison** names only the reported comparison fields; it does not certify stronger authentication overall. Unknown and explicit empty lists remain distinct.

Separate AI technical-writing review identified the API qualifier and build-term corrections. Native-language acceptance remains unverified. Executed screen checks are recorded in [HYB-327 verification](../../features/HYB-327/verification.md).

## HYB-225 candidate access terminology

M03 access management and P02 integration grants now include the synthetic candidate cost-read capability. The independent AI TW review accepted the distinction below as provisional product copy; scoped rendered review passed, while human native-language acceptance remains pending. Retain the existing View cost / 원가 조회 / عرض التكلفة action labels.

| Canonical key | EN | KO | AR candidate | Meaning guard |
| --- | --- | --- | --- | --- |
| charge_candidate | Charge calculation candidates | 과금 계산 후보 | نتائج احتساب الرسوم المبدئية | Immutable computed rating records awaiting further eligibility and approval integration; not approved invoices, issuance authority or verified source ownership. |
| charge_evidence | Source costs and calculation evidence | 원천 비용·계산 근거 | تكاليف المصدر وأدلة الاحتساب | Cost-protected source inputs and recorded calculation provenance; access requires ViewCost and does not approve the calculation. |

On 2026-11-20, [AWS Korean billing guidance](https://docs.aws.amazon.com/ko_kr/awsaccountbilling/latest/aboutv2/billing-what-is.html) was inspected for its distinction between costs, calculated charges and invoices; [AWS Arabic cloud financial management](https://aws.amazon.com/ar/aws-cost-management/) was inspected for cost, usage, charge and invoice vocabulary. Neither source establishes a standard translation for this product's candidate record. Independent AI TW re-review replaced the ambiguous Arabic candidate/filter wording with provisional calculation results; contextual review remains required; no regional consensus or native approval is claimed. No permission or financial semantics change with language.

## Approval read evidence (HYB-337)

| Concept | Korean | English | Arabic | Meaning and exclusions |
| --- | --- | --- | --- | --- |
| approval-review-capability | 승인 검토 · 조회 전용 | Approval review · read only | مراجعة الاعتمادات · للقراءة فقط | Read-only inspection capability, not a claim of connection, freshness, approval or execution. |
| approval-request-count | 전체 건수 | Exact request count | العدد الدقيق للطلبات | Exact number of matching requests, not an amount total across currencies or bases. |
| approval-target-period | 기간 | Period | الفترة | The producer's target period if supplied. Does not infer billing, attribution or posting period when unknown. |
| approval-producer-prerequisites | 원천 검증과 단계별 선행 조건이 연결되지 않았습니다. | Source validation and stage prerequisites are not connected. | التحقق من المصدر والمتطلبات المسبقة للمراحل غير مربوطين. | Known producer_unbound blocker. Does not relabel unrelated failure codes or assert that required checks passed. |

These candidates follow the actual B09 contract and module/service mapping. Independent AI technical-writing review passed; no human practitioner or native-language acceptance is claimed. Existing financial-domain terminology and the screen specification remain authoritative. Visual reviews and their inspected scope are recorded in [HYB-337 verification](../../features/HYB-337/verification.md).


## HYB-337 shared access capability terms

M03 and P02 identify the read capability as Approval review records / 승인 검토 기록 / سجلات مراجعة الاعتمادات. Its evidence field is Approval request and decision evidence / 승인 요청·결정 근거 / أدلة طلبات الاعتماد والقرارات. The request phrase means viewing recorded requests and decisions, not granting approval authority, validating source prerequisites or checking current financial eligibility. Independent AI technical-writing review passed these scoped distinctions; native-language acceptance remains pending. See [the composed plan](../../features/HYB-337/plan.md) and [verification](../../features/HYB-337/verification.md).

## Input confirmation permission request evidence

HYB-225 adds the M01 bulk-impact label for input_authority_proposals. Regional pages read on 2026-11-20: [Microsoft Korean access-request approval](https://learn.microsoft.com/ko-kr/entra/id-governance/entitlement-management-request-approve) and [Microsoft Arabic access-request management](https://support.microsoft.com/ar-sa/sharepoint/sharepoint-sharing-and-permissions/set-up-and-manage-access-requests). They support distinguishing a permission request from the approval decision and assigned access level. They do not prescribe the product-specific compound or certify Arabic UAE/Saudi practitioner usage. The Entra ar-sa counterpart rendered English and was not used as Arabic wording evidence.

Independent refusal_review performed the scoped AI TW/domain review against BulkPrepare and bulk-copy source, recommending explicit Korean data wording. Pending counts include unresolved grant applications; the existing recheck sentence describes current governance evaluation and does not promise cancellation or successful execution. Source research and AI review do not constitute human native-language acceptance. No layout, action or financial semantics changed.


M03 and P02 also display confirm_input as **Confirm input data / 입력 데이터 확인 / تأكيد بيانات الإدخال**, and synthetic_input_evidence as **Synthetic input evidence / 합성 입력 데이터 근거 / أدلة بيانات الإدخال الاصطناعية**. The synthetic qualifier remains visible; neither label implies financial approval or production evidence. Independent AI TW/domain re-review passed. These are display labels only: the generic grant-request capability phrase/catalog is not expanded. Real sample-session browser checks passed for all three affected screens in EN/KO/AR at 1440, 1024 and 390 pixels; human native-language acceptance remains pending.


## Separate input confirmation and evidence inspection

HYB-225 now requires an explicit confirm or inspect capability in synthetic authority proposals. M01 uses the broader **Input data permission requests / 입력 데이터 권한 요청 / طلبات صلاحية بيانات الإدخال** because the impact count covers both. M03 labels input_review_evidence as **Input comparison and reconciliation evidence / 입력 데이터 비교·대사 근거 / أدلة مقارنة بيانات الإدخال ومطابقتها**. The existing View cost action stays separate from Confirm input data. Independent AI TW/domain review accepted these candidates; actual Claude visual review passed nine scoped captures. Neither capability is added to generic grant offerings. Native-language acceptance remains pending.

The service field current_eligibility describes freshly checked synthetic input-evidence publication status only. It is not current financial authority, approved pricing, ERP readiness or downstream execution permission. Historical decision receipts remain explicitly historical after evidence changes.

## S06 synthetic input review

The authenticated S06 screen uses **Synthetic input review / 합성 입력 데이터 확인 / مراجعة بيانات الإدخال الاصطناعية**. The qualifier is mandatory: the screen publishes synthetic input eligibility internally and does not approve billing amounts or production use. **Confirm and publish / 확인·공개 / تأكيد ونشر** is explained at the point of decision; Korean 공개 does not mean public disclosure. **Reject input / 입력 거절 / رفض الإدخال** records rejection without publishing eligibility or deleting source data.

Use the existing **Workspace / 업무 공간 / مساحة العمل** label. Arabic revision labels use نسخة to distinguish document/scope versions from review activity. Period ID remains an opaque identifier; UTC timestamps, currency codes and exact amounts are directionally isolated, while free-text decision reasons retain automatic natural-language direction. Current evidence and eligibility are separate from recorded decisions. Unknown request outcomes and later refusals remain distinct; a fresh read neither cancels nor proves failure of an earlier request. Independent AI domain/TW review passed these candidates. Human native-language acceptance remains pending.

## Synthetic pricing permission provisioning

HYB-225 uses the following domain-reviewed display candidates in M01 impact and M03/P02 grants. They identify permissions for synthetic sample calculations; this increment does not expose pricing-policy execution or approve calculated amounts. Independent AI domain/TW review and scoped actual Claude render review passed. Human native-language acceptance remains pending.

| Concept / canonical alias | Preferred EN | Preferred KO | AR candidate | Definition and forbidden implication |
| --- | --- | --- | --- | --- |
| propose_pricing_policy | Propose pricing policy | 가격 정책 제안 | اقتراح سياسة التسعير | Propose terms for independent review; no effective-policy or financial approval implied. |
| approve_pricing_policy | Approve pricing policy | 가격 정책 승인 | اعتماد سياسة التسعير | Approve a policy proposal under separate authority; distinct from approval of billing amounts. |
| revoke_pricing_policy | Revoke pricing policy | 가격 정책 철회 | إلغاء سياسة التسعير | Stop future use while preserving history; do not imply deletion. |
| use_pricing_policy | Use pricing policy | 가격 정책 적용 | تطبيق سياسة التسعير | Select and apply approved terms; not permission to issue or approve a bill. |
| synthetic_pricing_policy | Synthetic pricing policy | 합성 시험용 가격 정책 | سياسة تسعير اصطناعية | Synthetic test policy, not a production commercial policy. |
| pricing_policy_evidence | Pricing policy evidence | 가격 정책 근거 | أدلة سياسة التسعير | Cost-protected policy evidence; permission to inspect does not approve the policy. |
| pricing_authority_proposals | Pricing policy permission requests | 가격 정책 권한 요청 | طلبات صلاحية سياسة التسعير | Pending/unresolved authority proposals for console ceilings and human grants; count is not a policy or amount approval. |

Policy revocation and grant revocation are intentionally separate: the former stops use of a policy; the latter withdraws a person's permission. Korean 철회 versus 회수 and Arabic إلغاء versus إبطال preserve this difference. English Revoke is shared but names the distinct object. Use is translated as applying selected terms (적용 / تطبيق), never financial approval. M03 presents the evidence field; P02 preserves its existing action/kind/scope presentation.

Regional wording evidence read on 2026-11-20: [Microsoft Korean policy configuration](https://learn.microsoft.com/ko-kr/azure/api-management/set-edit-policies) uses policy configuration/application vocabulary; [Microsoft Arabic API Management concepts](https://learn.microsoft.com/ar-sa/azure/api-management/api-management-key-concepts) uses policy terminology and distinguishes management configuration from gateway behavior. These support general IT vocabulary only, not an official translation of this product's pricing compounds. The Arabic policy-overview page appeared in search but direct retrieval failed, so it supplies no claimed inspected-page evidence. Existing access-request evidence above remains applicable to permission requests. No regional consensus or native approval is claimed.


## Synthetic pricing policy proposal impact

HYB-225 adds a distinct M01 impact category: `pricing_policy_proposals` → **Pricing policy proposals** / **가격 정책 제안** / **مقترحات سياسة التسعير**. This counts pending policy terms proposed by the selected users. It excludes approved/revoked policies and remains separate from permission requests for console ceilings and human grants. The recheck statement means the proposer’s current authority is checked at approval; it does not mean automatic cancellation or approval of billing amounts. Independent AI domain/TW review accepted these display candidates; native-language and owner acceptance remain pending.


## B09 governed synthetic charge records

HYB-340 distinguishes recorded calculation evidence from current financial eligibility. Independent AI terminology review accepted these candidates; native-language and owner acceptance remain pending.

| Concept | EN | KO | AR candidate | Meaning |
| --- | --- | --- | --- | --- |
| rated_candidate_total | Rated candidate total | 과금 계산 후보 합계 | إجمالي نتيجة احتساب الرسوم المبدئية | Exact rated result, not inferred tax-inclusive gross, final invoice or payment. |
| requester | Requester in source record | 요청자 (원천 기록) | مقدّم الطلب في سجل المصدر | Person who submitted the approval request, separate from original calculation author and decision actor. |
| policy_revision | Recorded policy revision | 기록된 정책 버전 | نسخة السياسة المسجّلة | Policy revision bound to the recorded request, not an assertion of current policy applicability. |
| synthetic row | Synthetic record | 합성 기록 | سجل اصطناعي | Actual record in a synthetic environment, not production billing. |

For current_checks_unverified, use: **Current prerequisites and permission were not rechecked for this historical read.** / **이 이력 조회에서는 현재 선행 조건과 권한을 다시 확인하지 않았습니다.** / **لم تُعَد مراجعة المتطلبات المسبقة والصلاحية الحالية عند عرض هذا السجل التاريخي.** This is an unchecked condition, not a failed validation or permission grant. Keep the neutral warning separate from failed-validation blockers.


## Synthetic document evidence and preparation permission

HYB-340 separates grant-governor approval from actual document evidence confirmation. Confirming evidence does not approve invoice gross or establish tax policy. Revoking evidence stops future use, whereas revoking a grant removes a user's permission. Preparing an ERP record is internal preparation, not external submission, posting or issuance.

| Identifier | English | Korean | Arabic |
| --- | --- | --- | --- |
| propose_document_evidence | Propose document evidence | 문서 근거 제안 | اقتراح أدلة المستند |
| confirm_document_evidence | Confirm document evidence | 문서 근거 확인 | تأكيد أدلة المستند |
| revoke_document_evidence | Revoke document evidence | 문서 근거 철회 | إلغاء أدلة المستند |
| prepare_erp_record | Prepare internal ERP record | 내부 ERP 기록 준비 | إعداد سجل ERP داخلي |
| synthetic_document_evidence | Synthetic document evidence | 합성 문서 근거 | أدلة مستند اصطناعية |
| document_evidence | Document evidence | 문서 근거 | أدلة المستند |
| document_authority_proposals | Document evidence and ERP preparation permission proposals | 문서 근거·ERP 준비 권한 제안 | مقترحات صلاحيات أدلة المستند وإعداد ERP |
| document_evidence_proposals | Document evidence proposals | 문서 근거 제안 | مقترحات أدلة المستند |
| issuance_protected_authority_proposals | Protected issuance settings permission proposals | 보호된 발행 설정 권한 제안 | مقترحات صلاحيات إعدادات الإصدار المحمية |
| issuance_term_authority_proposals | Payment terms permission proposals | 납부 조건 권한 제안 | مقترحات صلاحيات شروط الدفع |
| issuance_draft_authority_proposals | Issuance draft permission proposals | 발행 초안 권한 제안 | مقترحات صلاحيات مسودات الإصدار |

M01 distinguishes permission proposals across all five capabilities from pending document-evidence packets counted by their original proposer. The document count excludes confirmed and revoked packets and all permission proposals. Confirmation rechecks the original proposer's current proposal authority; a count does not imply automatic cancellation, financial approval or ERP preparation. The document row says "The proposer’s permission is rechecked when the document evidence is confirmed." Its false/unknown fallback does not assert this safeguard. M03/P02 show actual human grants and console ceilings. The evidence field is protected as Cost; its name is not translated as document cost. Independent AI technical-writing review selected these terms from the actual contract and existing confirmation/revocation vocabulary; native-language and owner acceptance remain unverified.

## Synthetic ERP result evidence permission

HYB-341 introduces a separate result-evidence permission namespace. Existing document preparation permissions do not grant result recording or confirmation. Apply names internal ledger evidence ingestion, never external ERP submission/posting, financial approval, issuance or delivery. Provisioned rights alone do not establish an implemented result-consumer workflow.

| Key | English | Korean | Arabic |
| --- | --- | --- | --- |
| propose_erp_result | Propose ERP result evidence | ERP 결과 근거 제안 | اقتراح أدلة نتائج ERP |
| confirm_erp_result | Confirm ERP result evidence | ERP 결과 근거 확인 | تأكيد أدلة نتائج ERP |
| revoke_erp_result | Revoke ERP result evidence | ERP 결과 근거 철회 | إلغاء أدلة نتائج ERP |
| apply_erp_result | Apply ERP result evidence internally | ERP 결과 근거 내부 반영 | تطبيق أدلة نتائج ERP داخلياً |
| synthetic_erp_result_evidence | Synthetic ERP result evidence | 합성 ERP 결과 근거 | أدلة نتائج ERP اصطناعية |
| erp_result_evidence | ERP result evidence | ERP 결과 근거 | أدلة نتائج ERP |
| result_authority_proposals | ERP result evidence permission proposals | ERP 결과 근거 권한 제안 | مقترحات صلاحيات أدلة نتائج ERP |

M01 counts pending/unresolved permission proposals, excluding completed grants. Its shared recheck note refers to the proposer's authority at grant approval. M03 and P02 show explicit action/kind labels for already provisioned rights; these labels are not generic grant offers. Source terminology and initial M01 rendered review passed independent AI technical-writing review. Independent AI technical-writing review also passed eight representative actual M03/P02 grant-row/context captures under HYB-341; P02 shows the action and kind, not a separate Cost-field label. Native-language and owner acceptance remain pending. These compounds reuse the document-evidence and access-governance vocabulary above; no external vendor uses of the complete compounds are claimed.

## Result evidence pending and unresolved impact

HYB-341 M01 distinguishes pending result evidence proposals counted by the original proposer from unknown internal applications counted by the applying actor. These are separate work categories, not a deduplicated total. Revoked or superseded evidence does not erase unresolved application ownership. A count does not imply automatic cancellation or external ERP execution.

| Key | English | Korean | Arabic |
| --- | --- | --- | --- |
| result_evidence_proposals | ERP result evidence proposals | ERP 결과 근거 제안 | مقترحات أدلة نتائج ERP |
| result_applications_unknown | ERP result evidence applications with unknown outcome | ERP 결과 근거 내부 반영 결과 미확정 | عمليات تطبيق أدلة نتائج ERP داخلياً ذات نتيجة غير مؤكدة |

The proposal note describes current proposer permission checks at evidence confirmation. The unknown-application note describes current applying-actor permission and evidence checks at recovery while preserving the possibility of earlier effects. False-recheck displays explicitly withhold assurance and retain that uncertainty. Separate read-only AI technical-writing review selected EN/KO/AR wording against the actual contract; native-language and owner acceptance are unverified. Existing layout and neutral separators remain unchanged. Rendered acceptance is recorded separately, not inferred from this glossary.

## Owned ERP result replacement

HYB-341 B09 distinguishes the reported result, its internal application and retained application history. A reserved latest application with an unknown outcome is not proof of current ledger application. A historical ancestor with an unresolved successor is not proof that replacement completed. Prospective targets describe the currently observed head; only an owned operation has a retained predecessor. Unknown-predecessor recovery requires the original actor and current permissions/evidence and may remain unavailable. These terms do not authorize ERP transmission, posting or final financial approval.

| Concept | EN | KO | AR |
| --- | --- | --- | --- |
| Initial operation | Initial internal application | 최초 내부 반영 | التطبيق الداخلي الأول |
| Replacement operation | Internal replacement | 내부 대체 반영 | الاستبدال الداخلي |
| Prospective predecessor | Proposed replacement target | 대체 예정 대상 | الهدف المقترح للاستبدال |
| Historical application | Historical record | 이전 처리 기록 | سجل سابق |
| Internal receipt evidence | Internal processing record hash | 내부 처리 기록 해시 | بصمة سجل المعالجة الداخلية |

Do not translate internal processing receipt as a payment receipt (영수증). Do not label an unknown reserved head as a completed application, and do not promise that retrying will recover a watermark-stranded predecessor. Independent technical-writing review inspected these strings in source and bounded actual EN/KO/AR renders; native-language/practitioner and owner acceptance remain unverified. See [replacement screen verification](../../features/HYB-341/result-replacement-screen-verification.md).

## ERP final-amount approval policy impact

HYB-341 M01 counts pending service-owned StageERPFinal approval-policy proposals by original proposer, separately from charge approval policies. Confirmation rechecks the proposer’s current governance authority. Confirming a policy neither approves a particular amount nor grants request/decision rights, consumes an approval, or transmits/posts to ERP.

| Key | English | Korean | Arabic |
| --- | --- | --- | --- |
| erp_final_policy_proposals | ERP final-amount approval policy proposals | ERP 최종 금액 승인 정책 제안 | مقترحات سياسة اعتماد المبلغ النهائي في ERP |

Independent read-only AI technical-writing review passed these source/domain labels and subsequently inspected six actual final/charge row crops in EN1440, KO390 and AR390. The wording and bounded rendered context passed; native-language and owner acceptance remain unverified. See the [ERP-final implementation plan](../../features/HYB-341/erp-final-approval-plan.md) for the exact verification boundary.

## ERP final-amount approval limit impact

HYB-341 M01 counts pending service-owned StageERPFinal amount-authority proposals binding a human recipient, currency, exact maximum absolute amount and inclusive UTC dates. Confirmation rechecks original proposer governance authority and recipient eligibility. These are distinct from policy proposals and request/decision gate permissions; confirmation does not approve a particular final amount or transmit/post to ERP.

| Key | English | Korean | Arabic |
| --- | --- | --- | --- |
| erp_final_authority_proposals | ERP final-amount approval limit proposals | ERP 최종 금액 승인 한도 제안 | مقترحات حدود اعتماد المبلغ النهائي في ERP |

Independent read-only source/domain technical-writing review passed these exact labels and subsequently passed wording in three actual EN1440/KO390/AR390 row crops. Native-language and owner acceptance remain unverified. The separate Claude CLI review covered those rows and two context captures; see the [ERP-final plan](../../features/HYB-341/erp-final-approval-plan.md) for bounded evidence.

## ERP final-amount action permissions

HYB-341 distinguishes approval-action permission from submission permission. The former is a direct human grant; the latter requires an independently governed console ceiling and human subgrant. Permission grants authorize the named actor action only. They do not supply amount authority, satisfy current evidence/policy checks, record a final decision, or authorize ERP transmission/posting. M01 counts pending/unresolved proposals; M03/P02 display provisioned action/resource labels.

| Concept | English | Korean | Arabic |
| --- | --- | --- | --- |
| erp_final_permission_proposals | ERP final-amount approval permission proposals | ERP 최종 금액 승인 권한 제안 | مقترحات صلاحية اعتماد المبلغ النهائي في ERP |
| erp_final_request_permission_proposals | ERP final-amount approval submission permission proposals | ERP 최종 금액 승인 요청 제출 권한 제안 | مقترحات صلاحية تقديم طلب اعتماد المبلغ النهائي في ERP |
| erp_final_approval | ERP final-amount approval | ERP 최종 금액 승인 | اعتماد المبلغ النهائي في ERP |
| approve | Approve ERP final amounts | ERP 최종 금액 승인 | اعتماد المبلغ النهائي في ERP |
| request_approval | Request approval for ERP final amounts | ERP 최종 금액 승인 요청 | طلب اعتماد المبلغ النهائي في ERP |

Independent source/domain technical-writing review passed these exact labels, then inspected six actual M01 crops and six M03/P02 crops with bounded rendered PASS. M03/P02 render action and kind separately. Native-language and owner acceptance remain unverified; see the [ERP-final plan](../../features/HYB-341/erp-final-approval-plan.md) for exact capture and test scope.

## Issuance permission areas and capabilities (M03)

HYB-341. Labels of the three issuance permission areas on M03 and of the capabilities provisioned in them. Source: `apps/console/src/modules/identity/pages/access/result-authority/areas.ts`. HYB-346 reuses the capability labels for the existing M03 and P02 grant rows.

| Key | English | Korean | Arabic |
| --- | --- | --- | --- |
| issuance_protected_authority area | Protected issuance settings permissions | 보호된 발행 설정 권한 | صلاحيات إعدادات الإصدار المحمية |
| issuance_term_authority area | Payment terms permissions | 납부 조건 권한 | صلاحيات شروط الدفع |
| issuance_draft_authority area | Issuance draft permissions | 발행 초안 권한 | صلاحيات مسودات الإصدار |
| protected `register_channel` | Register verification channel | 검증 채널 등록 | تسجيل قناة التحقق |
| protected `admit_channel` | Accept verification channel | 검증 채널 수락 | قبول قناة التحقق |
| protected `request_change` | Request protected settings change | 보호된 설정 변경 요청 | طلب تغيير الإعدادات المحمية |
| protected `verify_change` | Verify protected settings change | 보호된 설정 변경 검증 | التحقق من تغيير الإعدادات المحمية |
| protected `approve_change` | Approve protected settings change | 보호된 설정 변경 승인 | اعتماد تغيير الإعدادات المحمية |
| protected `inspect` | Inspect protected settings evidence | 보호된 설정 근거 조회 | عرض أدلة الإعدادات المحمية |
| term `record` | Record payment terms | 납부 조건 기록 | تسجيل شروط الدفع |
| term `approve` | Approve payment terms | 납부 조건 승인 | اعتماد شروط الدفع |
| term `bind` | Apply payment terms to a contract | 계약에 납부 조건 적용 | تطبيق شروط الدفع على عقد |
| term `approve_binding` | Approve applying payment terms to a contract | 계약에 납부 조건 적용 승인 | اعتماد تطبيق شروط الدفع على عقد |
| term `inspect` | Inspect payment terms evidence | 납부 조건 근거 조회 | عرض أدلة شروط الدفع |
| draft `submit` | Submit issuance draft for approval | 발행 초안 승인 요청 | إرسال مسودة الإصدار للاعتماد |
| draft `approve` | Approve issuance draft | 발행 초안 승인 | اعتماد مسودة الإصدار |
| draft `inspect` | Inspect issuance draft evidence | 발행 초안 근거 조회 | عرض أدلة مسودة الإصدار |
| verification channel | Verification channel | 검증 채널 | قناة التحقق |

- "Accept" for the admit step is not an approval; it is kept apart from 승인 / اعتماد so one panel does not use one verb for four different acts.
- Submitting a draft creates work awaiting approval and issues nothing.
- "Apply to a contract" is the internal bind and is not a contract amendment.
- A verification channel is distinct from the webhook subscriptions and notification channels named elsewhere in this glossary.
- Open: 납부 조건 is used here, in the M01 labels, the drafts approval inbox and the planning fields, while the contracts, review-and-issue and credit case screens use 지급 조건; the contracts screen words "verification" as 확인 where these labels use 검증. Both need one terminology decision.
- Review boundary: role-based technical-writing review only. Native-language and owner acceptance of the Korean and Arabic labels are pending.

### Issuance grants in the M03 grant rows and P02 ceiling rows

Once an issuance permission is granted it appears in the existing grant and ceiling rows by its action, resource kind and evidence field. The action labels repeat the capability names above. An inspect permission is stored as the `view_cost` action limited to one evidence field. Grant, ceiling and request rows on M03 show it as "View cost · <evidence field>"; P02 shows "View cost" with the resource kind. "View cost" is not an accurate name for reading settings, terms or draft evidence; the label predates these permissions and is left for a terminology decision. No request phrase is defined for these grants, so the generic access request form does not offer them (HYB-346).

| Identifier | English | Korean | Arabic |
| --- | --- | --- | --- |
| register_issuance_channel | Register verification channel | 검증 채널 등록 | تسجيل قناة التحقق |
| admit_issuance_channel | Accept verification channel | 검증 채널 수락 | قبول قناة التحقق |
| request_issuance_protected_change | Request protected settings change | 보호된 설정 변경 요청 | طلب تغيير الإعدادات المحمية |
| verify_issuance_protected_change | Verify protected settings change | 보호된 설정 변경 검증 | التحقق من تغيير الإعدادات المحمية |
| approve_issuance_protected_change | Approve protected settings change | 보호된 설정 변경 승인 | اعتماد تغيير الإعدادات المحمية |
| record_issuance_term | Record payment terms | 납부 조건 기록 | تسجيل شروط الدفع |
| approve_issuance_term | Approve payment terms | 납부 조건 승인 | اعتماد شروط الدفع |
| bind_issuance_term | Apply payment terms to a contract | 계약에 납부 조건 적용 | تطبيق شروط الدفع على عقد |
| approve_issuance_term_binding | Approve applying payment terms to a contract | 계약에 납부 조건 적용 승인 | اعتماد تطبيق شروط الدفع على عقد |
| submit_issuance_draft | Submit issuance draft for approval | 발행 초안 승인 요청 | إرسال مسودة الإصدار للاعتماد |
| approve_issuance_draft | Approve issuance draft | 발행 초안 승인 | اعتماد مسودة الإصدار |
| synthetic_issuance_protected | Synthetic protected issuance settings | 합성 보호된 발행 설정 | إعدادات إصدار محمية اصطناعية |
| synthetic_issuance_terms | Synthetic payment terms | 합성 납부 조건 | شروط دفع اصطناعية |
| synthetic_issuance_draft | Synthetic issuance drafts | 합성 발행 초안 | مسودات إصدار اصطناعية |
| issuance_protected | Protected issuance settings evidence | 보호된 발행 설정 근거 | أدلة إعدادات الإصدار المحمية |
| issuance_terms | Payment terms evidence | 납부 조건 근거 | أدلة شروط الدفع |
| issuance_draft | Issuance draft evidence | 발행 초안 근거 | أدلة مسودة الإصدار |

## Issuance pending work on M01 (HYB-343, HYB-344)

M01 lists these rows under pending work when a person is about to lose access. Each count is the number of open items that person started; it does not cancel them. The note beside a count says whether that person is checked again at the next step. If so, the item fails once their access is removed.

| Identifier | English | Korean | Arabic |
| --- | --- | --- | --- |
| issuance_setup_proposals | Issuance settings proposals | 발행 설정 제안 | مقترحات إعدادات الإصدار |
| issuance_draft_submissions | Issuance drafts awaiting approval | 승인 대기 중인 발행 초안 | مسودات الإصدار بانتظار الموافقة |
| issuance_protected_steps | Protected issuance settings steps awaiting another person | 다른 사람의 처리를 기다리는 보호된 발행 설정 단계 | خطوات إعدادات الإصدار المحمية بانتظار شخص آخر |
| issuance_term_steps | Payment terms steps awaiting approval | 승인 대기 중인 납부 조건 단계 | خطوات شروط الدفع بانتظار الموافقة |

The `issuance_setup_proposals` row counts issuance settings a person proposed that are not yet confirmed, including a proposal or a confirmation whose outcome is unknown. Confirmation rechecks that the proposer is still an installation administrator, so the row uses its own note: "The proposer’s installation administrator status is rechecked when the issuance settings are confirmed." / "발행 설정 확인 시 제안자의 설치 관리자 자격을 다시 확인합니다." / "تُعاد مراجعة صفة مقدّم المقترح كمسؤول تثبيت عند تأكيد إعدادات الإصدار." It replaces the general note "Rechecked when approved", because issuance settings are confirmed, not approved.

The `issuance_draft_submissions` row of M01 counts, for the person who submitted them, issuance drafts whose submission is recorded or has an unknown outcome and that have no approved review; a refused submission is not counted. A new review and a new approval attempt recheck that the submitter still holds the submission permission; recovering an approval the module already recorded does not. The row uses the general recheck note. A reviewer whose prepared review is the last one has no row of their own (HYB-344).

The `issuance_protected_steps` row of M01 counts, for the person who started each, a protected settings rules proposal not yet confirmed, a verification channel not yet accepted or whose acceptance has an unknown outcome, and a protected settings change requested or verified but not decided. A step whose own first request has an unknown outcome is counted too, although it waits for the same person's retry rather than for someone else. The next step rechecks that person's standing, so the row uses the note "Rechecked when the next step is sent". The `issuance_term_steps` row counts payment terms and their applications to a contract that are recorded but not approved, or whose recording or approval has an unknown outcome, for the person who recorded them; approval rechecks that person's permission. A change is counted by its recorded state, and ordinary reads do not write the lapse of its validity back, so a lapsed change can remain counted. Statements not yet used by a step and the people who must perform the next step have no row (HYB-344).

## Issuance setup sections on O03

HYB-341. Labels of the second-level navigation of the O03 documents view and of the selection of confirmed issuance settings that scopes protected settings and payment terms. Source: `apps/console/src/modules/organizations/pages/contracts/setup/part-copy.ts`. Composition and the remaining labels of these sections are in [the O03 screens plan](../../features/HYB-341/issuance-setup-screens-plan.md).

| Key | English | Korean | Arabic |
| --- | --- | --- | --- |
| navigation label | Issuance setup sections | 발행 설정 하위 메뉴 | أقسام إعداد الإصدار |
| part `settings` | Issuance settings | 발행 설정 | إعدادات الإصدار |
| part `rules` | Protected settings rules | 보호된 설정 규칙 | قواعد الإعدادات المحمية |
| part `channels` | Verification channels | 검증 채널 | قنوات التحقق |
| part `changes` | Protected settings changes | 보호된 설정 변경 | تغييرات الإعدادات المحمية |
| part `statements` | Statements | 진술 | الإفادات |
| scope action | Use these settings | 이 설정 사용 | استخدام هذه الإعدادات |
| chosen row | Selected | 선택됨 | محددة |
| scope bar | Selected issuance settings | 선택한 발행 설정 | إعدادات الإصدار المحددة |
| setup date | Setup date | 설정 기준일 | تاريخ الإعداد |

Labels of the protected settings rules part, from `setup/rules-copy.ts`:

| Key | English | Korean | Arabic |
| --- | --- | --- | --- |
| `rules_approvals` | Approvals required for a change | 변경에 필요한 승인 수 | عدد الاعتمادات المطلوبة للتغيير |
| `rules_validity_seconds` | Time allowed between verification and decision | 검증 후 결정까지 허용 시간 | المدة المسموح بها بين التحقق والقرار |
| `rules_max_lead_seconds` | Longest time from request to effective time | 요청부터 적용 시각까지 허용되는 최대 기간 | أطول مدة من الطلب إلى وقت السريان |
| `rules_channel_age_seconds` | Minimum time after channel acceptance before a change request | 채널 수락 후 변경 요청까지 필요한 최소 시간 | أدنى مدة بعد قبول القناة قبل طلب تغيير |
| retained state | Recorded state | 기록된 상태 | الحالة المسجَّلة |
| observation | Observed now | 현재 관찰된 상태 | المرصود حالياً |
| pinned confirmer | Confirmer pinned, not yet observed | 지정된 확인자, 아직 관찰되지 않음 | المؤكِّد المعيَّن، لم يُرصد بعد |
| observation `absent` | No rules observed | 관찰된 규칙 없음 | لا توجد قواعد مرصودة |
| observation `pending` | Proposal awaiting confirmation | 확인 대기 중인 제안 | اقتراح بانتظار التأكيد |
| observation `active` | In effect | 적용 중 | سارية |
| unconfirmed values | Proposed rule values — not confirmed | 제안된 규칙 값 — 확인 전 | قيم القواعد المقترحة — غير مؤكدة |

Rule steps (slice 8): "Propose protected settings rules / 보호된 설정 규칙 제안 / اقتراح قواعد الإعدادات المحمية" and "Confirm protected settings rules / 보호된 설정 규칙 확인 / تأكيد قواعد الإعدادات المحمية". Confirm (확인, تأكيد) is used for rules and issuance settings; verify (검증, تحقق) for changes and channels. Units are "Hours / 시간 / ساعات" and "Days / 일 / أيام".

"Recorded state" is what the service retained and may be an unknown outcome; "Observed now" is what is held at the time of the read; neither states that issuance is possible. The recorded states reuse the issuance settings labels (proposed, confirmed, proposal outcome unknown, confirmation outcome unknown).

Labels of the verification channels part, from `setup/channels-copy.ts`:

| Key | English | Korean | Arabic |
| --- | --- | --- | --- |
| state `registration_unknown` | Registration outcome unknown | 등록 처리 결과 미확인 | نتيجة التسجيل غير معروفة |
| state `registered` | Registered, awaiting acceptance | 등록됨, 수락 대기 | مسجَّلة، بانتظار القبول |
| state `admission_unknown` | Acceptance outcome unknown | 수락 처리 결과 미확인 | نتيجة القبول غير معروفة |
| state `admitted` | Accepted | 수락됨 | مقبولة |
| observation `absent` | No channel observed | 관찰된 채널 없음 | لا توجد قناة مرصودة |
| observation `retired` | Retired | 사용 종료됨 | أُوقف استخدامها |
| pinned admitter | Accepting person pinned, not yet observed | 지정된 수락자, 아직 관찰되지 않음 | المعيَّن للقبول، لم يُرصد بعد |
| `request_not_before` | Earliest change request | 가장 이른 변경 요청 시각 | أبكر وقت لطلب تغيير |
| rules `other` | Recorded under rules other than the current rules | 현재 규칙이 아닌 규칙에서 기록됨 | مسجلة وفق قواعد غير القواعد الحالية |
| subject kinds | Customer, Issuing entity, Document kind | 고객, 발행 법인, 문서 종류 | العميل، الجهة المصدرة، نوع المستند |
| channel kinds | Email, Phone, Portal, Postal mail | 이메일, 전화, 포털, 우편 | البريد الإلكتروني، الهاتف، البوابة، البريد العادي |

Channel and statement steps (slice 9): the step labels are the M03 capability labels "Register verification channel" and "Accept verification channel"; the row action is the short "Accept channel / 채널 수락 / قبول القناة". Statement steps are "Record statement / 진술 기록 / تسجيل الإفادة" and "Withdraw statement / 진술 철회 / سحب الإفادة". The review groups are "Statement you will record / 기록할 진술 / الإفادة التي ستسجّلها", "Statement you recorded in this session / 이 세션에서 기록한 진술 / الإفادة التي سجّلتها في هذه الجلسة" and "Action it supports / 진술이 뒷받침하는 단계 / الإجراء الذي تدعمه". A statement "supports" an action only in the sense that the action names it; it is never called evidence or proof.

"Accepted" is the admission step of a channel (수락, قبول) and never means valid or approved. The earliest change request time is shown only when the read states it and does not state that the user may request a change through the channel.

Labels of the protected settings changes part, from `setup/changes-copy.ts`:

| Key | English | Korean | Arabic |
| --- | --- | --- | --- |
| field `document_recipient` | Document recipient | 문서 수신자 | مستلم المستند |
| field `document_template` | Document template | 문서 템플릿 | قالب المستند |
| field `payment_account` | Payment account | 결제 계좌 | حساب الدفع |
| unknown phase label | Step outcome | 단계 처리 결과 | نتيجة الخطوة |
| phase `request_unknown` | Request outcome unknown | 요청 처리 결과 미확인 | نتيجة الطلب غير معروفة |
| phase `verification_unknown` | Verification outcome unknown | 검증 처리 결과 미확인 | نتيجة التحقق غير معروفة |
| phase `decision_unknown` | Decision outcome unknown | 결정 처리 결과 미확인 | نتيجة القرار غير معروفة |
| state `requested` | Requested | 요청됨 | تم طلبه |
| state `confirmed` | Verification recorded, awaiting decision | 검증 기록됨, 결정 대기 | سُجِّل التحقق، بانتظار القرار |
| state `scheduled` | Scheduled | 예정됨 | مجدول |
| state `rejected` | Rejected | 반려됨 | مرفوض |
| state `lapsed` | Lapsed | 기한 경과 | انقضت المهلة |
| observation `absent` | No change observed | 관찰된 변경 없음 | لا يوجد تغيير مرصود |
| pinned verifier | Verifier pinned, not yet observed | 지정된 검증자, 아직 관찰되지 않음 | المتحقِّق المعيَّن، لم يُرصد بعد |
| pinned decision actor | Deciding person pinned, not yet observed | 지정된 결정자, 아직 관찰되지 않음 | صاحب القرار المعيَّن، لم يُرصد بعد |
| `effective` | Takes effect | 적용 시각 | وقت السريان |
| provenance | Entered by staff, Customer document, Customer portal | 직원 입력, 고객 문서, 고객 포털 | إدخال من الموظف، مستند من العميل، بوابة العميل |

Change steps (slice 10): "Request protected settings change" and "Verify protected settings change" are the M03 capability labels. The decision step is "Decide protected settings change / 보호된 설정 변경 결정 / البتّ في تغيير الإعدادات المحمية", because it may approve or reject, while the capability is labelled "Approve protected settings change"; its choices are "Approve / 승인 / اعتماد" and "Reject / 반려 / رفض". Verification outcomes read "The instruction was confirmed through the channel" and "The instruction was denied". Rejection reasons: Incorrect details / 내용 오류, Missing supporting material / 뒷받침 자료 누락, Suspected fraud / 사기 의심, Other / 기타. The reason code `missing_evidence` is not shown as "evidence".

The change state `confirmed` is not shown as "confirmed" or "verified": it means that a verification was recorded and a decision is awaited. A verifier's outcome is shown as what the verifier stated. "Latest scheduled change" is recorded history and does not state that an invoice draft can use the change.

Labels of the statements part and of the sample references, from `setup/statements-copy.ts`:

| Key | English | Korean | Arabic |
| --- | --- | --- | --- |
| purpose `channel_registration` | Channel registration | 채널 등록 | تسجيل القناة |
| purpose `channel_admission` | Channel acceptance | 채널 수락 | قبول القناة |
| purpose `change_instruction` | Change instruction | 변경 지시 | تعليمات التغيير |
| purpose `change_verification` | Change verification | 변경 검증 | التحقق من التغيير |
| state `unconsumed` | Not used — supports no record | 미사용 — 어떤 기록도 뒷받침하지 않음 | غير مستخدمة — لا تدعم أي سجل |
| state `consumed` | Used | 사용됨 | مستخدمة |
| state `withdrawn` | Withdrawn — supports no record | 철회됨 — 어떤 기록도 뒷받침하지 않음 | مسحوبة — لا تدعم أي سجل |
| `locator` | Where the source is kept | 원본 보관 위치 | مكان حفظ المصدر |
| sample reference | Sample recipient A | 샘플 수신자 A | مستلم تجريبي A |
| sample channel | Sample issuing entity channel A | 샘플 발행 법인 채널 A | قناة جهة مصدرة تجريبية A |
| sample sender | Sample unregistered sender | 샘플 미등록 발신자 | مرسل تجريبي غير مسجل |

Sample labels are built from the use and the letter of the catalog code; the code itself is not shown.

Labels of the O03 payment terms view, from `setup/terms-copy.ts`. The Korean term is 납부 조건 on this view, as on M03, M01 and the draft inbox; the payer labels (지급 주체, 지급자) are a separate concept.

| Key | English | Korean | Arabic |
| --- | --- | --- | --- |
| view | Payment terms | 납부 조건 | شروط الدفع |
| `net_days` | Net days (calendar): 30 days | 납부 기한 일수(달력일): 30일 | أيام الاستحقاق (تقويمية): 30 يومًا |
| part | Payment terms of these settings | 이 설정의 납부 조건 | شروط الدفع لهذه الإعدادات |
| binding author | Applied by | 적용자 | طبّقه |
| state `registration_unknown` | Recording outcome unknown | 기록 처리 결과 미확인 | نتيجة التسجيل غير معروفة |
| state `pending` | Recorded, awaiting approval | 기록됨, 승인 대기 | مسجَّلة، بانتظار الاعتماد |
| state `approval_unknown` | Approval outcome unknown | 승인 처리 결과 미확인 | نتيجة الاعتماد غير معروفة |
| state `approved` | Approved | 승인됨 | معتمدة |
| pinned approver | Approver pinned, not yet observed | 지정된 승인자, 아직 관찰되지 않음 | المعتمِد المعيَّن، لم يُرصد بعد |
| bindings | Payment terms applied to this contract | 이 계약에 적용된 납부 조건 | شروط الدفع المطبَّقة على هذا العقد |
| open-ended | No end date | 종료일 없음 | بلا تاريخ انتهاء |
| reasons | Initial, Renegotiation, Correction | 최초 적용, 재협상, 정정 | أولي، إعادة تفاوض، تصحيح |
| origins | These issuance settings, Other issuance settings, Billing profile | 이 발행 설정, 다른 발행 설정, 청구 프로필 | إعدادات الإصدار هذه، إعدادات إصدار أخرى، ملف الفوترة |

Step and result labels of the payment terms view (slice 7): the four step labels reuse the M03 capability labels, with "this contract" for the two application steps (이 계약에 납부 조건 적용, 이 계약에 납부 조건 적용 승인; تطبيق شروط الدفع على هذا العقد, اعتماد تطبيق شروط الدفع على هذا العقد). The check button reads "Review before sending / 전송 전 검토 / المراجعة قبل الإرسال" because no permission is checked before sending. Results read "Refused. This request recorded nothing.", "Outcome unknown…", "Recorded. Current issuance eligibility was not checked." and "Settled by a matching recovery"; the retry and recovery labels are the merged "Retry exact request / 동일 요청 재시도" and "Recover recorded attempt / 기록된 시도 복구".

"Approved" is the approval of the recorded terms or of their application by a second person. The view calculates no due date and states nothing as payable, due or overdue.

A statement (진술, إفادة) is its author's accountable claim and is never labelled evidence, proof or verification. "Selected" states which confirmed settings the user chose as scope; it does not state that issuance is possible. Role-based AI review only; native-language and owner acceptance remain unverified.
