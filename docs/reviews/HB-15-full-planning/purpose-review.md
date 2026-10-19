---
type: Product Purpose Review
title: Original purpose alignment across all operating profiles
description: Compares the original product mission with current planning and proposed schemas and identifies purpose-specific dependency conflicts and incomplete completion routes.
status: draft
---

# Original purpose alignment

Tracking: [HB-15](https://linear.app/hybrid-billing/issue/HB-15). Accountable owner: Seung Woo Park. Root owns PM/planning, integration and this report; separate read-only architect, DBA and QA reviewers inspect business boundaries, money semantics and completion journeys. Machine: `mac_mini`. Date: 2026-10-14. Documentation review only; original planning, architecture, schemas and screens are unchanged.

**Verdict: PARTIAL alignment; purpose-complete planning is not demonstrated.** The original mission remains in the requirements and architectural responsibilities. A current common-schema dependency conflict can force internal settlement into selling objects. Other concerns are incomplete profile-specific linkage and uneven presentation, not proof that the five profiles were deleted or that the product has become a different platform. Implementation remains prohibited until explicit owner planning completion.

## Evidence and scope

The original v1.16 purpose is a customer-operated platform connecting multi-cloud, datacenter and AI costs to organization attribution, contractual pricing and settlement. It serves MSPs, standalone enterprises and corporate groups. It preserves supplier originals, separates selling charges, and completes external work through ERP or standard XLSX handoff or qualified self-issuance, while internal Chargeback completes as organization settlement. Admin and Assistant use the same validated authority paths. Original body locators: B0004–B0005, B0010, B0016.R001–R005, B0019–B0026, B1186–B1192 and B1196–B1246.

Use the verified original hash, complete source extraction and collective 123-chapter review in [the preceding re-audit](re-audit.md) and its [input inventory](re-audit-inputs.json). This follow-up freshly inspects purpose passages, operating-model clauses and the relevant current/proposed contracts; it does not claim another verbatim reread of all 123 chapters or all screen prose. The preceding fourteen findings remain open. This report adds a purpose-level interpretation and refines FA-01's impact, rather than replacing that omission audit.

Reviewed bases: merged main `b5c65cf66992f703247b8e4713f86cef7835b767`; HB-18 SaaS proposal `6acb38c68117ff1f1865776aca00f87c61956196`; HB-19 architecture `75c4c188e5d42d682d286cebe9900b2bc0bc660c`; HB-20 capacity/storage `62f7df1`; HB-21 schema `129c525ba3e942a086a4cbb71b18dc7f5795cf4c`. HB-18–21 remain separate proposed inputs, not merged or approved choices. Remote main, HB-15 and HB-21 heads were read back with `git ls-remote` and matched the recorded bases. Authorized local companion planning remains necessary.

The schema citations below refer to HB-21 at that fixed revision: [logical catalog](https://github.com/myblueton-sw/hybrid_billing/blob/129c525ba3e942a086a4cbb71b18dc7f5795cf4c/docs/architecture/data-ownership/logical-data-schema.md), [monthly schema](https://github.com/myblueton-sw/hybrid_billing/blob/129c525ba3e942a086a4cbb71b18dc7f5795cf4c/docs/architecture/data-ownership/monthly-settlement-schema.md) and [evidence schema](https://github.com/myblueton-sw/hybrid_billing/blob/129c525ba3e942a086a4cbb71b18dc7f5795cf4c/docs/architecture/data-ownership/evidence-detail-schema.md). Original B/R locators address extracted document body blocks/table rows, not page numbers.

## Five original profiles and their completion

These are business profiles in one shared core, not five proposed products. Operating type, source authority and legal output are independent dimensions.

| Profile / original purpose | Required economic route and actors | Required completion | Current assessment |
| --- | --- | --- | --- |
| MSP Cloud Billing — B0016.R001, B0033 | Preserve/reconcile supplier cost; seller selects effective customer contract rates and eligible components; the relevant infrastructure/cost owner approves cost/allocation policy; the MSP approves its selling policy and manages customer-owned policies only within explicit delegation; independent amount approval | Designated ERP or qualified platform-issued customer document, or standard XLSX handoff with subsequent result registration. Direct CSP contracts yield only eligible MSP service fees, not duplicate CSP sales | Direction aligned; end-to-end closure PARTIAL. Monthly:49,52–53,66–69 separates passthrough/markup/margin and independent tariffs. FA-04/12 leave adjustment commands and output-mode subjects incomplete |
| Datacenter Billing — B0016.R002, B0058–B0065 | Reuse eligible native cost/Chargeback without repricing its cost; separately price validated quantity/capacity when the approved customer contract selects that route. Separate rack/network fixed charge, reserved GPU base, overage, included power and governed SLA credit | Contract-selected external charge or approved internal settlement, with evidence, exact approval and designated output | PARTIAL. Monthly:50,59 preserves charge distinctions, but explicit native-result versus raw-quantity authority selection and complete adjustment/output journey remain incomplete |
| Private Cloud Billing — B0016.R003, B0026/B0029 | Preserve VMware/HawkEye result authority; use approved internal cost and conserving organization allocation where appropriate; no fictitious supplier invoice or MSP | Authorized Showback view or independently approved internal Chargeback statement; external selling only under an additional real contract and authority | Shared dependency CONFLICT. Allocation intent exists in monthly:41, but generic calculation/fact/approval fields require selling dependencies described below |
| AI Billing — B0016.R004, B0068–B0076 | Preserve TokenMeter native results and evidence semantics; select one CSP/internal cost authority. Independently price eligible token/request/GPU quantities when that is the contract route; customer price differs from supplier cost | Approved customer charge or internal AI allocation, depending on profile; exact model/category/status/denominator and evidence lineage retained | PARTIAL. Monthly:51,59,68–69 preserves tariffs and cost separation; native priced result reuse and cross-source GPU authority need explicit route binding. No actual duplicate cost is demonstrated |
| Enterprise/group FinOps — B0016.R005, B0010/B0026, B1218/B1224 | Customer approves cost/allocation policy; complete pools retain idle/unallocated residual; groups preserve legal entities and current versus historical organization meaning | Showback analysis or approved internal Chargeback/XLSX. Group summary creates no new legal invoice or receivable; intercompany sales require separate contract/ERP treatment | Shared dependency CONFLICT, despite aligned prose. Internal completion must not require seller, selling RateBinding, external invoice or Payment Terms |

## Purpose-level findings

### PA-01 — Actual conflict: common dependencies assume a selling purpose

Logical catalog:51 says fields are required unless conditional. `CalculationRun` at :110 requires monthly-close/item/binding; common `ApprovalRequest/ApprovedSnapshot` at :113 requires a final settlement snapshot, service/rate/adjustment references and selling eligibility. Evidence `CalculatedComponentFact` at :49 requires commercial service-item/source-mapping/rate-binding even for an allocation component. These dependencies conflict with allocation-only operation in monthly:41 and original B0010/B0026/B0029. Independent cost recognition also has its own approval purpose in logical:141 and the merged payment/recognition contract.

This refines existing FA-01 beyond the approval row: calculation, result and approval dependency types also need purpose-specific applicability. It is a documentary contradiction and implementation risk, not an observed financial failure. A purpose label alone does not make universally required selling fields conditional.

**Required correction:** define purpose-bound input/result/dependency and approval subjects for customer selling settlement, internal allocation/Chargeback, independent cost recognition and ERP-final draft. Keep shared current permissions, exact revisions, reconciliation, immutable evidence and applicable economic uniqueness. Commercial seller/rate/issuance/receivable dependencies apply only to their actual purpose. Showback is an authorized view and need not manufacture financial issuance.

### PA-02 — Incomplete linkage: already-priced results versus new pricing

Original B0058 requires reuse of selected HawkEye snapshots without applying power rates again. B0068 says TokenMeter is not merely a meter adapter; B0073–B0076 preserve supplied evidence classes and distinguish CSP AI cost, token attribution and shared GPU observations. Current cost facts and monthly pricing clauses preserve these distinctions, but lack a complete explicit selection contract between reuse of native priced results, allocation of authoritative cost and independent pricing of verified quantity.

**Required correction:** bind every received amount to source authority, monetary role, stage, exact native result revision and permitted transformation. Select the permitted transformation per monetary role/stage, not exclusively for a whole business profile: native cost reuse can coexist with independent customer pricing. Select one authoritative economic cost where HawkEye/TokenMeter/CSP observations overlap. Preserve TokenMeter's MEASURED/BILLED/OBSERVED/estimated/PRESENCE_ONLY/None semantics rather than reducing them to one generic quality field. No claim is made that current text universally multiplies cost by quantity or already double charges; source protection and independent tariff rules are explicit in monthly:52–53,66–69.

### PA-03 — Incomplete completion: final adjustment and output paths

The redesigned schema correctly supplies provisional/final month close, explicit target-current deltas, once-only adjustment application, fresh final amount approval, late-month invalidation and immutable issued corrections. It restores an important initial purpose: calculation and final customer charge adjustment, not merely processing pre-finalized bills. However, FA-04/12 show that this lifecycle has not been fully connected to typed commands, existing screens, work and mode-specific ERP/XLSX/internal output records.

**Required correction:** trace each profile from input confirmation through calculation/allocation, reconciliation, adjustment where applicable, exact purpose approval and confirmed selected output. Download or transport success is not reflected ERP/file business success. Output alternatives share the authorized economic intent; they must not issue three charges for one result. Internal settlement needs its own completion artifact.

### PA-04 — Presentation imbalance, with preserved business boundaries

HB-21 opens with actual usage to selling approval/Claim/issuance. Its prose permits internal-only operation, but the primary view lacks explicit internal and native-result exits. HB-19 preserves cost/allocation/selling responsibilities; its opening solution SVG emphasizes messaging/processing/stores/RAG and does not clearly show all five profiles, outbound standard XLSX/manual registration or the Assistant ordinary-command path. The encoded SVG labels/structure were inspected; newly rendered visual QA was not performed.

**Required correction:** lead the architecture and schema with the mission/profile-to-input-policy-approval-output map; show internal versus external completion and ERP/manual XLSX explicitly; distinguish advisory analysis/RAG from human-directed Assistant commands through ordinary validation/approval. Present storage/HA technology candidates as supporting views. This does not establish that the architecture has drifted into an unrelated AI/data product.

## Preserved boundaries and later additions

| Boundary | Verdict and limits |
| --- | --- |
| Customer-operated installation and disconnected work | Aligned direction: original B0004/B0063 and installation assessment preserve local computation without cloud LLM dependency. Offline packaging, entitlement and supported deployment profiles remain unverified; HA candidates are not universal installation prerequisites |
| ERP and standard XLSX as core | Preserved in original B0019–B0022 and current release gates. Their detailed result/artifact relationships and overview visibility remain incomplete; standard XLSX is not merely an optional report export |
| Supplier originals and distinct money meanings | Preserved: B0005/B0099–B0106 and current schemas separate provider-reported cost, internal cost, allocation and customer sales. FOCUS cost columns are neither one additive total nor the customer's selling rate |
| AI under ordinary authority | Preserved in HB-19: analysis is advisory; accepted or human-directed action re-enters the ordinary authenticated command and domain approval path. AI availability cannot become a requirement for deterministic financial closing |
| No infrastructure provisioning/accounting engine | Preserved: B0024/B1192/B1462 exclude resource control, CSP optimization and a new accounting/payment gateway. Own-product recovery is distinct. Proposed SaaS entitlement delivery grants no infrastructure-provisioning authority |
| Later SaaS extension | Separately proposed owner addition, with shared identities and distinct service/payment states. Base contract service subscriptions were already original scope. TokenMeter service fees differ from AI supplier cost; SaaS extension does not replace customer-operated delivery |
| Capacity/object storage/HA | Owner additions support billing workload. >100M source records/day is a premise, not measured performance. Parquet/object foundation and MongoDB exclusion are recorded; engine/topology choices and operational limits remain design decisions |

## Purpose closure and next review order

1. Root integrates the purpose-route matrix; architect/DBA independently review typed applicability for PA-01 before physical schema or implementation.
2. Bind native-result reuse, authoritative-cost allocation and independent quantity pricing to fixed source profiles and lineage (PA-02).
3. Connect the existing adjustment lifecycle and all selected completion routes to commands, permissions, existing screens, tasks and acceptance (PA-03); update business overview presentation (PA-04).
4. QA reviews five complete documentary walkthroughs: MSP direct/resale; DC fixed/reserved/usage/included power; VMware/HawkEye internal Chargeback; CSP+TokenMeter AI cost and customer tariff; enterprise/group allocation without external invoice. Use preserved original fixed examples B0121–B0137 and GH01–GH08, not new unapproved production tariffs.
5. Each walkthrough must retain current actor authority, input versus money approval, exact decimal/currency/basis, whole-pool conservation, once-only effects, target-total adjustment/current revision, late-month versus issued/unknown correction, customer-safe disclosure and independent cash/recognition. Complete the canonical English trace and portable reviewed package before purpose-complete acceptance.

This is a recommended planning correction sequence, not authorization to implement or silently modify the baseline. Customer policy values, supported source/ERP combinations, legal issuance, runtime accuracy, performance, recovery and offline operation remain UNVERIFIED or NOT_RUN.

## Actual verification and handoff

Architect, DBA and QA supplied separate purpose reviews. Root checked the cited required calculation/approval/fact rows against the original internal-operation clauses, integrated disagreements by distinguishing conflict from incomplete linkage, and retained the earlier omission report unchanged. Important integrated-report re-review and scoped publication checks are recorded in [purpose verification](purpose-verification.md). A report-correctness PASS cannot change the product completeness verdict or authorize planning completion.
