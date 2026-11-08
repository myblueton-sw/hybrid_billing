---
type: PlanningReview
title: Planning-led product purpose and navigation review
description: Reassesses the proposed billing navigation against Hybrid Billing's own planning obligations and corrects a competitor-led reading of the product.
status: draft
---

# Review scope and source authority

HB-76, 2026-11-03. Root owns PM and this report. The owner requested review of our own planning because its intended purpose is more specific than the external comparison suggested. This review reads the authorized local v1.16 overview, requirements, UI design, payment terms and independent ERP-recognition documents, together with the planning-completion decision. It does not claim a fresh audit of all 123 original DOCX chapters, implementation or customer validation.

The planning-completion decision authorizes product development; older analysis paragraphs saying development has not begun are historical state, not a new development prohibition. Sources below are authorized local companions and may not all be available from a fresh clone. This report does not publish those companion files or their private workspace settings.

Planning defines required product behavior. Competitor documentation informs interaction patterns and is not authority to delete or replace that behavior. No objective claim that competitors lack these capabilities, or that our implementation is superior, follows from this review.

# Product purpose supported by the plan

Product-design synthesis: explain and control how heterogeneous supplier and internal costs become organizational responsibility or customer charges, with reproducible evidence and separately governed calculation, approval, issuance and external reflection.

This is supported by the overview's explicit separation of supplier cost collection, customer sales billing and internal allocation, and by UI design's success criterion: complete a first verified settlement, resolve failures, and trace issuance, handoff, correction and retention. It is not merely a spending dashboard or a subscription checkout flow.

| Planning evidence | Required distinction | Consequence for screens and navigation |
| --- | --- | --- |
| requirements.md HB-01–03; ui-design.md operating profiles | MSP, standalone enterprise and corporate group share a core; workspace, legal entity, organizational relationship and trading responsibility remain distinct. | Profile-specific landing scope: customer portfolio, departments/projects or affiliates. No mandatory fictional MSP/customer-sales path for internal settlement. Role/current authority still governs access. |
| HB-05, HB-07, HB-10–13 | Cloud, on-premise, GPU and AI inputs have different meanings and evidence; supplier cost, internal cost, allocated burden and sales amount differ. | Integrated views need explicit amount basis and source lineage. Source count is not the product story, and combining currencies or cost/sales totals is not integration. |
| HB-14, HB-16–20 | Shared-cost allocation and selling policy are distinct; supplied CSP commitment/discount outcomes are accepted rather than recreating CSP rating or optimization. | Preserve source-result acceptance: do not multiply accepted HawkEye electricity cost by an electricity price again, or introduce TokenMeter estimates as additional supplier costs. Separate observed cost exploration from allocation-policy approval and sales-rate changes. The previous broad Costs and usage group is too ambiguous unless these subdomains are explicit. |
| HB-08–09, HB-21, HB-23 | Fixed evidence, eligible inputs, duplicate-charge protection, amount approval and execution have separate contracts. | Every amount must lead to source/policy/revision evidence. A receipt, successful comparison or approval cannot stand in for another required gate. |
| HB-24–29 | ERP issuance, self issuance and internal statements have distinct authority; API/file are delivery methods. | Do not design a compulsory platform-invoice-then-ERP pipeline. Show the contract's issuing system, external confirmation and safe unknown-outcome recovery. Download is not external completion. |
| erp-recognition-cadence.md | Monthly cost recognition proceeds independently of quarterly customer billing; comparable monthly posted net amounts reconcile to quarterly totals without double recognition. | Parallel recognition and billing tracks, shared period/amount evidence and explicit reconciliation. A single left-to-right completion bar cannot represent this contract. |
| payment-terms.md | Due dates attach to applicable issued documents; overdue depends on authoritative, sufficiently current allocated receipt balances. | Show due, outstanding, overdue and unknown separately. Internal allocations without payment obligations are not overdue. Submission, delivery or a payment claim is not collection. |
| HB-30–34 | Home exposes stage, due date, owner and next action; UI, API, CLI and AI share commands and authority. | Overview is a scoped action center. AI can continue the same persisted work through authorized commands; it is not merely a help widget or a new approver. |

Source locations are the named sections of /docs/planning/v1.16/requirements.md, /docs/planning/v1.16/ui-design.md, /docs/planning/v1.16/overview.md, /docs/planning/v1.16/payment-terms.md and /docs/planning/v1.16/erp-recognition-cadence.md in the authorized workspace. Current planning authorization is docs/planning/planning-completion.md.

# Corrections to the earlier IA proposal

The earlier seven-destination assessment is a candidate, not the conclusion of this planning review. Existing IA already identifies many of the right business domains. The principal failure is that the implementation presents a screen catalog and metadata descriptions instead of connecting those domains to concrete decisions, inputs, commands and results.

- Preserve organization/contract context as a business control, not just a low-frequency setup page. Historical attribution and issuer responsibility govern every downstream result.
- Keep actual cost exploration, allocation policy and selling policy distinguishable even if navigation groups them. A neutral label such as Costs and usage must not hide pricing or ownership decisions.
- Keep source completeness, provider-document reconciliation and internal-cost validation visible. Moving every data function under Settings would undermine recurring close work.
- Expose monthly recognition and customer billing side by side. ERP belongs to more than the final document-delivery step.
- Preserve governed policy editing, impact simulation and approval as first-class tasks. Do not collapse them into an analytics filter or a generic assistant command.
- Preserve AI's common-command execution role and persistent job continuity. Global placement is acceptable only if it retains the current object/scope and applicable approvals.

Thus menu count and renaming should follow task validation. The commercial redesign should not implement the earlier proposed regrouping merely because every screen was mapped once. No IA change is applied by this report.

# Purpose-led story for the product

The recurring question is: for this organization or customer, period, contract and currency, what costs are supported, who bears or owes which amount, which obligations are pending, and what action may I take next?

1. The operator opens a profile-appropriate overview across permitted sources. It separately shows supplier/internal cost, allocated burden or customer charges as applicable, input coverage, upcoming deadlines and blockers with owners. Unknown is not zero and stages are not merged into a completion percentage.
2. The operator investigates an exception in the same period/object context, corrects or completes input and obtains the required evidence-bound input eligibility. From any proposed charge or allocation, the operator can inspect its source, relationship revision, policy and calculation basis.
3. Policy changes use draft, validation, scope-wide impact and authorized approval before activation. A calculation compares fixed inputs and revisions; a later material change requires renewed validation/approval. Financial approval and input eligibility cannot waive each other.
4. Work then follows the configured obligations, which may run in parallel: monthly cost recognition to the authorized ERP, periodic internal chargeback, or external customer billing. Neither a profile label nor a UI setting creates document authority. Quarterly billing reconciles only comparable monthly recognition amounts and handles differences by approved adjustment rather than double posting.
5. The result remains traceable through applicable document issuance, external acceptance, delivery, receipt allocation and correction. Each has its own evidence and status. The same authorized command/job may be continued from a screen, API or AI conversation without creating a second financial execution.

# Required acceptance journeys

- Standalone enterprise: cloud/on-premise/GPU or AI cost evidence through shared allocation and approved internal chargeback, without a compulsory sales invoice or receivable.
- MSP: supplier evidence through independently governed customer sales policy, approval and configured issuance/delivery, with supplier cost and margin protected from customer views.
- Corporate group: affiliate/legal-entity scope and monthly ERP recognition alongside a different billing cadence, comparable totals, missing-month handling and approved adjustments without duplicate recognition.
- Recovery across the above: late input, policy revision change, rejected approval, ambiguous ERP response, stale receipt state and original-linked corrections. Safe recheck/recovery differs from blind replay.

These are acceptance targets, not executed tests. A commercial IA needs observed task outcomes against these journeys; neither the clarity of planning nor a competitor comparison certifies usability or feature readiness.

# Independent review

Independent planning/billing-boundary review identified six omissions in the earlier comparison and reviewed this corrected synthesis: PASS. The verdict covers this report, not a fresh audit of all planning, IA adoption, implementation, usability or owner acceptance. Source availability and purpose coverage checks PASS. No baseline change, owner FIX, main merge or deployment is implied.
