---
type: DesignCoverageReview
title: Planning-to-design coverage audit
description: Separates missing and partial visual journeys from existing screens, deferred decisions and unverified implementation.
status: draft
---

# Scope and conclusion

Tracking: [HB-54](https://linear.app/hybrid-billing/issue/HB-54/document-design-coverage-gaps-against-full-planning). Root owns this report; two read-only reviewers inspected billing/customer coverage and access/onboarding/settings/VMware coverage. No screens, prices, permissions or frozen baseline artifacts are changed.

The design is broad but not yet a complete operational screen specification. The fixed baseline has 89 pages across 11 delivered modules; module 12 has no delivered dedicated visual pages. Current onboarding r2 adds operable synthetic forms but remains a held draft. Existing [56-item checklist](../reviews/HB-43-onboarding/checklist.md) explicitly distinguishes Shown, Specified and Partial from runtime acceptance. An unchecked runtime item does not prove a missing screen, and a page count does not prove an end-to-end usable workflow.

Evidence examined: [baseline registry](registry.json), HB-43 checklist/requirements/generators and actual billing PDF/HTML contents, [screen binding register](../ui/screen-binding-register.md), [payment and recognition contract](../contracts/billing/payment-and-recognition.md), [onboarding r2](../design-modules/03-onboarding/r2/README.md) and relevant authorized local planning companions. Original v1.16 source obligations use their existing canonical traces; this is not a new line-by-line reread of all 123 original chapters. Older local inventory fields saying development is unauthorized are historical and do not supersede [planning completion](../planning/planning-completion.md).

Classification: **missing visual journey** means a dedicated current-baseline flow is not represented; **partial** means screens/options exist but key decisions or recovery interactions are not demonstrated; **deferred** means the owner deliberately postponed detail. Earlier local specs/SVGs can exist even when their journey has not been carried into the current v10 book. No percentage-complete estimate is assigned.

# Module coverage

| Module / baseline pages | Coverage assessment |
| --- | --- |
| 01 Overview / 1–2 | Index exists. Current later module drafts must not be mistaken for an assembled, accepted replacement edition. |
| 02 Access / 3–10 | Entry/login/invitation/roles/recovery concepts exist; administrator invitation and recovery governance detail is partial. Authentication product selection is deferred. |
| 03 Onboarding / 11–18 plus r2 | First connection and validation are represented. Credential lifecycle, administrator task lifecycle, recurring source management and attribution transfer need detail. Visual refinement and actual-console experience are deferred. |
| 04 Work / 19–21 | Large customer portfolio and profile exist. Payment terms are displayed largely as options, not a full editing/operational flow. Issuance-delay wording needs clarification. |
| 05 Sources / 22–32 | Supplier PDF intake, extraction, differences, audit, pivot and raw exploration exist. AI failure/rejection/resume and complete workbench interaction need detail. |
| 06 Contracts / 33–57 | Supplier/customer contracts, pricing rules, matching, bulk preview, exceptions, Marketplace and allocation are well represented. Payment schedules/deposit handling/master-child conflict editing remain partial. |
| 07 Reconcile / 58–66 | Calculation trace, discount bridge, discrepancies, recalculation and independent approval exist. Iterative failure/no-progress handling remains partial. |
| 08 Issue / 67–71 | Assignment, issue review, document history and ERP mapping exist. Receivables and monthly-recognition/quarterly-billing recovery are incomplete visual journeys. |
| 09 History / 72–77 | Monthly lists, customer/month detail, variance and correction exist. Do not label monthly history missing. |
| 10 Customer / 78–86 | Product/instance discounts, invoices, usage, monthly history and inquiry/response exist. Open-book disclosure grant/revoke and confirmed receivable visibility need detail. |
| 11 Settings / 87–89 | Setup and license screens exist. Operational notifications, service-credential administration, recovery/retention and scenario governance are not represented as full current-book journeys. |
| 12 VMware/on-prem / no pages | Genuine dedicated visual gap. Generic VMware references do not replace metering/history/cost-pool/allocation/selling/reconciliation screens. |

# Evidence-backed gaps and completion criteria

Priorities below are proposed design sequencing, not new release commitments or changes to optional scope.

| ID / priority | Classification and existing evidence | Required screen-level completion |
| --- | --- | --- |
| G01 / P1 | **Partial contract/payment editing.** CT-01–05; customer-detail p21 explicitly presents options rather than active selections; customer-contract p42 provides context. | Operable advance/interim/balance stage editor and sum/currency validation; approved master-child inheritance/override conflict review; allowed/default payment methods; confirmed deposit allocation, shortage and reversal detail. Preserve existing customer/business-registration fields. |
| G02 / P1 | **Missing receivables lifecycle.** CT-06 and PT-02/03/04/06/07. p69–70 distinguish delivery/ERP/payment; p80 has due date with payment confirmation unavailable; p81–83 show documents, not confirmed debt. | Operator balance/receipt allocation detail and aging queue, partial receipt, reversal/overpayment, approved due-date extension, as-of evidence and unknown-versus-confirmed overdue. Customer view exposes only authorized balances with confirmation status and as-of evidence; a last-known balance remains clearly labeled as needing confirmation. Payment execution is not implied. |
| G03 / P1 | **Missing monthly ERP recognition to quarterly billing reconciliation journey.** CT-08 / ERPC-01–08. p71 shows purpose/destination/mapping and status taxonomy. | Monthly recognition batch → external result/read-back → quarterly month bridge; correction counted once, locked-period handling and unknown/partial posting recovery. Prevent double recognition; do not duplicate the existing ERP mapping page. |
| G04 / P1 | **Missing dedicated VMware visual module.** BL-07, CT-09, ON-02–05, SP-04/05 and OP-01/02. Registry marks module12 planned-gap with no pages. Generic connection p14, allocation p52 and usage p84 exist. | Endpoint/history capability → inventory/customer identity → rename/recreate/move/resize history → counter completeness → approved asset/lease/power/operations cost pools → shared/idle allocation → separate selling approval → reconciliation/customer detail. Preserve the local VMO-01–10 draft as planning evidence, not delivered visuals. |
| G05 / P1 | **Partial connection lifecycle.** ON-01–05 / OP-05/06. r2 forms demonstrate first connection; handoff is a text modal and protected credential registration is referenced but not rendered. | Credential register/reference/rotate/revoke, disconnect impact without deleting shared exports, administrator pending task/expiry/return/conflict views, recurring source health/history and effective-dated attribution transfer with recipient acceptance and prior access revocation. Full console interaction/visual style remains deferred by the owner. |
| G06 / P2 | **Partial AI-assisted work and exception loop.** BL-10; extraction p24, differences p25, evidence p26, hypothesis p62, recalculation p63 and approval p65 already exist. Local A01 specifies context, evidence, command plan and durable results. | Rejected suggestion, no-progress, new conflicting evidence, retry history and manual handoff; explicit context → evidence → proposed action → authorized execution → actual result. A conversational answer is not a completed billing job. AI itself is not absent, and provider choice remains open. |
| G07 / P2 | **Partial customer cost-disclosure administration.** PR-12 / BL-01. p78 gives customer-safe rates; p79 assumes an approved open-book example; p85–86 cover inquiry/response. | Select recipient/fields/source/version, review grant, revoke it and show resulting omitted/blocked export states. Existing discount and invoice explanation screens are retained; backend enforcement is a separate verification obligation. |
| G08 / P2 | **Operational administration not integrated into current v10 journeys.** Original local P01/P02/P03/P04/P05/P06/M04/U03/M05 and tracked binding register; v10 settings only p87–89. | Operational notification rules/channels versus personal preferences, task/alert ownership and escalation; service credentials; isolated restore/migration and opening balances; retention/legal hold/deletion state; supported-profile evidence and reproducible demo/failure scenarios. Earlier specs/static drawings are inputs, not proof these current-product journeys are complete. |
| G09 / P2, method-neutral | **Partial invitation/recovery governance.** AC-02–04, p5/7/9/10. Role ceilings and recovery rules are shown. | Administrator invitation queue, expiry/reissue/revoke, scoped role-change impact, recovery request/custodian review/limited restoration/revocation confirmation. This does not require choosing an authentication vendor now. |
| G10 / P1 wording correction | **Ambiguous issuance delay versus payment overdue.** BL-03 / CT-06 / PT-07. p19 operations combines No confirmed invoice with Due / overdue and 5 days overdue. | Label the monthly issuance queue explicitly as issuance deadline/days late. Actual payment aging belongs to confirmed receivables with evidence; unissued drafts must not appear as payment-overdue debt. Change through a future module revision, not by rewriting frozen v10. |

# Covered items that must not be rediscovered as missing

- BL-02/03: p19 represents 5,000 targets, 900 unissued and 4,100 issued, with post-issue cases as a subset, company/reason/owner filtering and next actions. Large-scale runtime performance remains unverified, but the portfolio screen exists.
- BL-06: monthly issuance/history and correction views are present at p72–77 and customer p81–83.
- BL-08 / EX-01/02: supplier PDF extraction/differences, pivot and raw-data exploration are present at p22–32.
- PR-01–11 / PR-06–15: rule matching, broad and SKU-specific rates, bulk changes, discount sequencing, per-instance explanation and fixed invoice totals appear in p33–60 and p78–79.
- EX-06/07/08 and CT-09: Marketplace/commitment and allocation have representative views; optional purchase forecasting is not a missing mandatory settlement prerequisite.
- AC-01–04 and LC-01/02: entry/access and licensing screens exist. Unknown authentication products and runtime license verification are not evidence of absent screens.

# Owner-deferred and intentionally optional scope

Current onboarding visual style is held, not accepted as final. [Recorded owner feedback](../design-modules/03-onboarding/r2/console-comparison.md#owner-feedback-keep-this-draft-and-defer-visual-refinement) requires product-wide look and feel without the perceived Claude style and actual customer-console guided setup; screenshots alone are insufficient. No redesign is performed by this audit.

Authentication method/product selection remains deferred. HawkEye/TokenMeter integration remains deferred under the collection architecture. EX-10 third-party API and EX-08 purchase planning remain optional/peripheral. Infrastructure control and a general optimization/BI product are not added to billing scope merely to fill screen counts.

# Proposed next design sequence

1. Connect the actual operator month-close story across customer terms, source/contract evidence, exceptions, approval and issuance; clarify issuance-delay wording. Existing pages are the base, not discarded work.
2. Complete payment/deposit and monthly ERP-recognition workflows (G01–03), including normal, blocked, partial, unknown and retry states.
3. Produce the dedicated VMware visual module if VMware remains in the first delivery scope (G04); the requirement remains regardless of delivery sequencing.
4. Complete AI exception/workbench and customer-disclosure administration (G06–07).
5. Complete credential/handoff/recurring-source and operational governance detail (G05/G08/G09), respecting the owner's current visual-detail deferral.

Before another full-book freeze, trace each original screen-family obligation and each owner checklist item to an active module/screen/state, explicit deferral or optional/excluded decision. Do not infer readiness from an assembled PDF alone. Runtime acceptance remains separate and unchanged.

# Verification and limitations

Two independent read-only reviewers supplied concrete source/screen findings; root integrates the report. Billing review inspected actual v10 PDF/HTML contents in addition to requirements. Access/onboarding/VMware review inspected registry, generators, canonical bindings and current prototype source. Root checked original local screen-family inventories and A01/P03/P04/U04 specifications. No new visual rendering, usability/scale test, live connector test or original-DOCX exhaustive audit was run for HB-54.

The local module12 draft and original planning companions are untracked inputs and were not staged, rewritten or certified as delivered. Historical local inventory status flags were not used as current implementation facts. The audit changes only this report and the baseline index link; all 89 baseline pages, 56 runtime checkboxes and current held UI remain unchanged.

Actual report re-review: PASS from both independent domain reviewers within their stated scopes. Root applied the suggested quarterly-billing terminology and last-known-balance clarification. Root QA: PASS for report/index OKF metadata, relative link targets, scope and whitespace. These checks validate the audit artifact, not the product workflows.
