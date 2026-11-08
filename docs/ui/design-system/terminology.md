---
type: Guideline
title: Regional IT terminology and localization rules
description: Source-backed rules for meaning-preserving English, Arabic and Korean interface terminology, with candidate terms and review criteria.
status: draft
---

# Regional IT terminology and localization

Owner direction, recorded 2026-10-31 under [HB-61](https://linear.app/hybrid-billing/issue/HB-61/import-owner-supplied-design-system-r1-package): English and Arabic design have highest priority, Korean is a first-class market, and terminology must reflect each market's IT usage rather than mechanical translation. This guide implements that content rule. Its seed labels remain candidates until contextual review; draft status does not permit bypassing the owner's research requirement.

Scope: navigation, buttons, forms, tables, validation, status messages, notifications, help, exports and design mockups. Technical API identifiers remain stable; localized display labels map to their concept rather than renaming public fields. This document does not change financial calculations, document classifications, authorization or tax obligations.

## Rules for choosing words

1. **Define the concept before translating.** Record the user's task, the object, the action/result, and the applicable domain contract. A label must describe what the actual product does. Do not derive a business definition from an English word or a vendor's translation.
2. **Research the target locale directly.** Use regional first-party IT products, technical documentation and terminology/style guidance. Prefer multiple independent sources for generic labels. A vendor-specific term can use that vendor as its authority within the integration context. A search snippet alone is a discovery lead, not final terminology approval.
3. **Check source quality.** A localized vendor page may itself be automatically translated or awkward. Record its publisher, locale, document section, access date and limits. Published usage is evidence of usage, not proof of native quality or market consensus.
4. **Choose by meaning and audience.** Resolve differences against product contracts first, then relevant regional/domain usage and local readability. Record the selected label, alternatives and rejected senses. Use a short explanation when a specialist term is necessary; do not replace a precise concept with a familiar but incorrect word.
5. **Keep concepts distinct and labels consistent.** One approved label per concept, locale and context. Permit documented technical/customer-facing variants, not uncontrolled synonyms. Reuse the same meaning in menus, help, statuses and exports. Search aliases may recognize familiar alternatives without changing the canonical display name.
6. **Keep provider names recognizable.** Preserve product names, resource identifiers, API keys, protocol names and established acronyms. Explain an unfamiliar acronym on first relevant use. Do not translate AWS account, Azure subscription and Microsoft Entra tenant into a single generic organization type.
7. **Machine translation is draft assistance only.** AI/MT output requires source checking, domain review and review by a person competent in the locale and relevant IT work before it is finalized. Back-translation or a second model alone does not establish regional usability. Never record native-speaker or practitioner approval that did not occur.
8. **Separate language from market context.** Track `en-US`, `ko-KR`, `ar-AE` and `ar-SA` separately. A shared Arabic term can be reused with evidence; UAE acceptance does not certify Saudi acceptance or every Arabic-speaking market. Language switching must not change currency, entity, tax profile, date semantics or permissions.
9. **Review copy in the screen.** Check labels in the actual task, including empty, pending, failed and completed states. An error names the problem and a supported next action; an action names the operation and object when needed. Translate complete messages, not concatenated fragments. Keep zero, missing and unknown distinct.
10. **Make changes traceable.** A terminology change lists affected screens/messages/help/exports, its source and rationale. Update those artifacts together. A wording change that implies a different financial effect or authority requires the existing independent domain-review workflow before adoption.

These rules are a product-specific synthesis. Google's global-writing guidance supports unambiguous wording and consistent terminology [S1]. UAE's content guidance supports audience-appropriate tone, understandable language and acronym explanation; its Arabic guidance specifically addresses Arabic phrasing [S2–S3]. None of these sources supplies a complete billing glossary for this product.

## Locale writing profiles

| Profile | Product writing rule | Evidence boundary |
| --- | --- | --- |
| US English | Use US English, concise task/action wording and established cloud/FinOps terms. Avoid idioms, marketing language and unexplained internal names. Distinguish billing operations from payment movement. | S1, S4 and S10 inform this profile; they do not prove every US user's preference. |
| UAE Arabic | Use clear professional Modern Standard Arabic as the product baseline. Explain specialist acronyms with Arabic wording and retain the Latin acronym where needed. Avoid literal English word order. Do not shorten critical distinctions to fit a menu. | Professional register and acronym handling draw on S2–S3; choosing Modern Standard Arabic is this product's baseline, not a claim of a prescribed billing vocabulary. |
| Saudi Arabic | Start from shared Arabic candidates, but keep country-specific validation separate. Retain recognizable provider terminology in integration/admin tasks. Do not assume country-specific legal/document terms are interchangeable. | S9 demonstrates published `ar-sa` IT terminology, not general Saudi practitioner acceptance or UAE equivalence. |
| Korean | Use concise natural Korean labels and consistent polite explanatory messages. Keep established IT loanwords where they are clearer, and explain them for nontechnical audiences. Preserve provider-specific names while distinguishing general business concepts. | Korean cloud usage in S5, S8 and S8a demonstrates real terminology. It is not a requirement to reproduce those providers' wording everywhere. |

RTL, punctuation, plural forms, font shaping and mixed Latin/Arabic identifiers require rendered checks. Do not alter IDs, customer-entered names or exported machine-readable keys to improve phrasing. Formatting preferences and country-specific document terminology require their own verified requirements.

## Initial term register: researched candidates

All rows are **OBSERVED/CANDIDATE**, not approved translations. English labels are concept anchors; Korean/Arabic entries are candidate UI labels extracted or normalized from the stated usage evidence. Normalization, such as choosing singular instead of plural, is a product proposal. Arabic candidates are not separately approved for either UAE or Saudi Arabia. A dash is an unresolved label, not permission to improvise one. Sources were accessed on 2026-10-31; domain and locale reviewer identities are not yet assigned.

| Concept key and meaning | English candidate | Korean candidate | Arabic candidate | Usage evidence and exclusions |
| --- | --- | --- | --- | --- |
| `billing.operations`: preparing and managing charges/invoices | Billing | 청구 | الفوترة | EN S4; KO S5; AR S6. Korean candidate deliberately distinguishes this operation from payment despite broader vendor product titles. Do not use it for all settlement/closing tasks. |
| `payment.records`: records of payments | Payments | 결제 내역 | المدفوعات | EN S4; KO S5; AR S6. Menu label does not imply the product can execute a payment. Receipt/collection and approval have separate meanings. |
| `invoice.document`: an invoice record | Invoice | 청구서 | فاتورة | EN S4; KO S5; AR S9b. Do not automatically call every invoice a tax invoice or every internal statement an invoice. |
| `usage.quantity`: measured consumption | Usage | 사용량 | الاستخدام | EN S4; KO S5; AR S6. Keep quantities/units separate from charges and utilization percentages. |
| `cost.analysis`: analysis of costs | Cost analysis | 비용 분석 | — | EN S4; KO S5. Exact Arabic task label needs contextual selection; S6 supports cost terminology, not approval of a full label. |
| `cloud.subscription`: a provider subscription | Subscription | 구독 | اشتراك | EN concept/KO S8; AR S9. Preserve provider context; not automatically a customer contract or tenant. |
| `identity.tenant`: an identity-provider tenant | Tenant | 테넌트 | مستأجر | KO S8; AR S9. This term is recognizable in Microsoft administration, but is not automatically the name of Hybrid Billing's organization/customer boundary. |
| `cost.allocation`: assigning costs to responsible units | Cost allocation | 비용 배분 | — | EN S10; KO choice is a product candidate; S5 uses 비용 할당. Resolve audience and contract before standardizing. Arabic equivalent remains unresolved. |
| `finops.showback`: visibility of attributed costs | Showback | — | — | S10 distinguishes visibility from charging departmental financial accounts. No reviewed local equivalent yet. |
| `finops.chargeback`: departmental financial charging | Chargeback | — | — | S10 supports the FinOps sense. Do not use an unqualified payment-dispute translation or imply money movement unsupported by the product. |

The register is intentionally incomplete. Country-specific tax documents, reconciliation, closing, settlement, credit notes, refunds, collections and permission-action terms require further context-specific research. No source consulted establishes a single universal translation for those concepts.

## Meaning traps and design examples

These are product copy constraints, not new financial or identity contracts.

| Trap | Required handling |
| --- | --- |
| Translating every billing-related screen as 결제 | Preserve the billing/payment distinction. 청구 and 결제 are the contextual candidates above, pending review; do not imply payment execution on a reporting-only screen. |
| Translating Korean 정산 to Settlement everywhere | Determine whether the screen calculates charges, allocates costs, closes a period, reconciles records or settles obligations. Research each concept separately. |
| Mixing Korean 결제 and 결재 | Payment and organizational approval are different actions. Match the actual operation before writing the label. |
| Translating Invoice as 세금계산서 everywhere | Use the actual document classification. Tax-specific labels require the applicable jurisdiction's authoritative source and domain review. |
| Replacing Tenant with 고객사 or 조직 for readability | Check the entity and security model; use contextual help rather than silently equating distinct objects. Conversely, do not expose every internal tenant identifier as a user-facing business concept. |
| Copying NAVER Cloud's 서브 계정 into every provider | S8a demonstrates a provider-specific service term. Preserve it in that integration; it is not a universal name for all users/service accounts. |
| Translating Chargeback without context | The FinOps sense in S10 concerns allocating financial responsibility. Distinguish it from payment disputes and refunds. |
| Using one generic success message | Name the completed stage accurately: a saved draft, submitted approval and issued invoice must not all appear as an issued invoice. |

## Required term record and review

For each label promoted from candidate to approved, record: concept key; definition and relevant contract/screen; audience; locale and jurisdiction where relevant; preferred label; contextual example; allowed aliases; rejected senses; primary source URL/section/date; whether text was observed or proposed; domain reviewer; locale reviewer; review date; decision and affected artifacts. Reviewers may share responsibilities only when competent for both; model review is recorded as model review.

States: `candidate` → `domain-reviewed` → `locale-reviewed` → `approved`; a superseded choice becomes `deprecated` with its replacement. Approval requires both semantic and contextual language checks; it does not require four different people. The initial rows above have no completed domain/locale approvals.

Before accepting a revised screen, check:

- A user can explain the action, object and outcome in each language without relying on the English screen.
- Menu, page title, field, action, status, help and export labels use the same concept consistently.
- Billing/payment/approval, organization/account/tenant and invoice/tax-document distinctions survive translation.
- No silent fallback or machine-generated wording is presented as approved Arabic or Korean copy; visibly flag unresolved copy in review artifacts.
- Arabic RTL and embedded IDs/currency codes, Korean wrapping, and longer English/Arabic labels remain readable without truncating meaning.
- Semantic and regional review results are recorded as PASS, FAIL or UNVERIFIED. A document-level source review is not a native-user usability test.

## Source register and limitations

All sources below were consulted on 2026-10-31. Official publication proves that a wording is used there, not that it was human-translated or accepted by all regional users. Summaries are limited to their stated scope.

| ID | Primary source | Relevant evidence |
| --- | --- | --- |
| S1 | [Google: Write for a global audience](https://developers.google.com/style/translation) | US-English global writing; clear sentences, consistent terms, localization beyond translation. |
| S2 | [UAE content guidance, English](https://designsystem.gov.ae/guidelines/content) | Human, understandable content and consistent audience-appropriate tone; government context. |
| S3 | [UAE content guidance, Arabic](https://designsystem.gov.ae/ar/guidelines/content) | Arabic tone, explanation of acronyms with the English acronym, and Arabic punctuation conventions. |
| S4 | [AWS Billing and Cost Management, English](https://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/billing-what-is.html) | Billing/payments, invoices, usage, and cost-analysis vocabulary. |
| S5 | [AWS Billing and Cost Management, Korean](https://docs.aws.amazon.com/ko_kr/awsaccountbilling/latest/aboutv2/billing-what-is.html) | Published use of 청구, 결제, 청구서/인보이스, 사용량, 비용 분석 and 비용 할당; wording varies by context. |
| S6 | [AWS Cloud Financial Management, Arabic](https://aws.amazon.com/ar/aws-cost-management/) | Published billing, payment, cost and usage terminology; no country-specific acceptance claim. |
| S8 | [Microsoft cloud subscriptions/accounts/tenants, Korean](https://learn.microsoft.com/ko-kr/microsoft-365/enterprise/subscriptions-licenses-accounts-and-tenants-for-microsoft-cloud-offerings?view=o365-worldwide) | Distinguishes provider subscription, account and tenant vocabulary. |
| S8a | [NAVER Cloud policy and role management](https://guide.ncloud-docs.com/docs/subaccount-policy-role) | Korean provider terminology for policies, roles, permissions and subaccounts; its permission rules are not imported. |
| S9 | [Microsoft cloud subscriptions/accounts/tenants, Arabic](https://learn.microsoft.com/ar-sa/microsoft-365/enterprise/subscriptions-licenses-accounts-and-tenants-for-microsoft-cloud-offerings?view=o365-worldwide) | Published `ar-sa` subscription and tenant terms. |
| S9b | [Azure invoice download, Arabic](https://learn.microsoft.com/ar-sa/azure/cost-management-billing/understand/download-azure-invoice) | Published Arabic invoice terminology. |
| S10 | [FinOps Foundation terminology](https://framework.finops.org/assets/terminology/) | Cost allocation and showback/chargeback distinctions; no approved Korean/Arabic equivalents supplied. |

Microsoft's [localization style-guide catalog](https://learn.microsoft.com/en-us/globalization/reference/microsoft-style-guides) lists Korean and Arabic resources, but the Korean PDF could not be inspected through the research tool. KRDS application-form guidance surfaced in search but its page retrieval failed. Neither unread document supplies a claimed rule here. A NAVER Cloud menu-migration notice appeared in Korean search results but direct retrieval did not expose that notice text; it was excluded from the evidence register. No regional practitioner interviews, native-language approval, production locale files or rendered multilingual QA were completed in this documentation task.
