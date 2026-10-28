---
type: DesignReview
title: Billing-first screen review and assisted reconciliation story
description: Reviews the existing 49-screen baseline against later decisions and international workflow evidence, then separates the MSP correction loop from customer billing views.
status: draft
---

# Outcome and ownership

**Baseline sufficiency update, 2026-10-28:** [External assessment](baseline-sufficiency-review.md) finds the core story adequate but the full operational screen baseline incomplete. Prior visual QA is not a workflow-completeness certification.

**Visual delivery update:** the [v2 prototype](prototype/README.md) now provides 12 operator and 3 customer screen states, plus a generated fifteen-page review PDF. It applies the renewed research to the connected monthly billing story. See [visual verification](visual-verification.md). Earlier notes below about the two-frame limitation remain historical; the 49-screen baseline PDF is still unchanged.

**Research update, 2026-10-28:** the owner rejected the two narrow storyboards below as insufficient for full CSP billing. The [renewed multi-provider research and screen brief](multi-provider-research.md) now supplies the provider-specific evidence, international MSP comparison, portfolio-to-invoice story and showback/allocation/chargeback boundaries for the next design. The two SVGs remain historical arithmetic illustrations; they are not an accepted complete invoice design. The existing v1.16 PDF has not been updated.

Tracking: [HB-35](https://linear.app/hybrid-billing/issue/HB-35/review-planning-baseline-for-the-complete-product-screen-mockup). Accountable owner: Seung Woo Park. Machine: mac_mini. Baseline: main `4f990f168790ad15b10bf6359927fadeefb8e6ef`. Root owns PM/UI-UX synthesis and these artifacts; separate read-only `screen_source_review` reviews source trace, billing/permission design and the resulting artifacts. This is a planning review and static design study, not a product implementation or a full DOCX re-audit.

**Finding:** the existing inventory is reusable, but the old screens should not be reproduced unchanged. The visible generic tables and repeated explanation cards do not adequately tell the monthly billing story. The central MSP task is an iterative evidence-and-money reconciliation loop, assisted by AI, before separate approval and issuance. The customer task is understanding their own disclosed bill, usage and questions. These require distinct experiences.

The owner explicitly requested, during this review: existing planning review before the whole mockup; visible screens before extended description; international product comparison; a coherent first-use/core billing story; separate MSP billing-operator and customer views; repeated original-data/rate/discount/error correction through monthly invoice issuance; and AI assistance to reduce manual correction effort. OpenAI was mentioned as a candidate, not a selected provider/model or permission to transmit data. Authentication selection remains deferred.

# Source precedence and review limits

1. The [planning completion declaration](../../planning/planning-completion.md) supersedes historical development-prohibited text; it does not certify implemented screens.
2. [Current architecture decisions](../../architecture/current-design-decisions.md), [MIT policy](../../licensing/product-policy.md), [authentication deferral](../../work/HB-34-customer-access/brief.md), and the owner's current directions constrain older proposals.
3. The 43 source requirement groups and 49 screen IDs remain the inventory. Source requirement IDs such as HB-29 are not the same as Linear ticket HB-29. No menu redesign deletes baseline scope.
4. Existing screen specifications remain detailed source inputs, subject to the explicit unresolved [binding register](../../ui/screen-binding-register.md). A register entry is not proof that a local specification was changed.
5. [SaaS subscription planning](../../planning/saas-subscription-billing.md) extends existing screen families; it does not turn this self-hosted product into a hosted subscription service or alter MIT rights.

Reviewed sources include local v1.16 requirements/catalogue, provider profiles, domain contracts, UI navigation/flows, binding register, representative affected specifications, later owner decisions, HB-17 competitor research, and current implementation reports. The local registry/spec/design/SVG inventory was structurally checked across all 49 screens. This is not a clause-by-clause semantic audit of every screen or the original DOCX. Authorized local companions remain partly unpublished; their English publication/adoption belongs to HB-24. This report does not publish those files or claim a fresh clone contains them.

# International workflow evidence

Official pages were fetched on 2026-10-28. These are documented product patterns, not logged-in product tests, visual superiority scores or compatibility certification. Prior [HB-17 research](../HB-17-competitor-ux/comparison.md) already proposed connected journeys; the legacy visual examples did not sufficiently express them. No vendor images are redistributed.

| Reference | Observed pattern | Adaptation and limit |
| --- | --- | --- |
| [Stripe invoice management](https://docs.stripe.com/invoicing/dashboard/manage-invoices) | Filterable invoice list, detail and actions dependent on document status; operational badges add context | Document detail should state what can happen next. Do not import Stripe payment automation or financial authority. |
| [Flexera billing plans](https://docs.flexera.com/flexera-one/partners/cloud-cost-optimization/billing-plans) | Customer plans, ordered rules/adjustments and processing history; changes require reprocessing | Explain which rule affected which customer-period and compare revisions. Preserve our approved arithmetic order and immutable history. |
| [Finout MegaBill](https://docs.finout.io/user-guide/inform/megabill) | Persistent filters/saved views, cost-versus-usage selection, graph and table drilldown | Use for analysis and source investigation. Spend exploration cannot replace the billing operator's correction/approval workspace. Do not inherit public-sharing defaults. |
| [Lago draft invoices](https://getlago.com/docs/guide/invoicing/draft-invoices) | Explicit review window before finalization | Make draft, review and issued results visibly different. Do not adopt automatic finalization at expiry or direct fee overwrites. The linked screenshot fetch returned HTTP403; it was not visually inspected. |

“International standard” here means an evidence-based design benchmark, not a universal UI standard. Proposed evaluation criteria are clear entry task, persistent scope, meaningful amount basis, explainable differences, context-preserving drilldown, specific next action, separate consequential stages and accessible state presentation. Language, theme and country-specific tax choices are not determined by this comparison.

# Two different user stories

| | MSP billing operator | Customer |
| --- | --- | --- |
| First question | Which customer-period can close, what is wrong, and what must I do? | What am I being charged, why, and what document or help do I need? |
| Main object | Fixed customer-contract-period candidate, evidence and versions | Own disclosed invoice/statement and permitted usage |
| Main work | Investigate source/rate/discount differences; prepare correction; recalculate and independently reconcile repeatedly | Read charge composition; inspect allowed usage; obtain document; ask a line-specific question |
| Sensitive information | Scoped source costs, contract prices, margins where authorized, internal differences | Only allowed selling/allocation amounts and explanations; no internal cost/margin/cases/raw locators |
| Completion | Validated fixed amount, separate approval and issuance; delivery and ERP remain separately evidenced | Understand own bill and permitted status; download or inquiry does not imply payment or cancellation |

A customer view is not an operator payload with hidden columns. Access, aggregates, errors, exports, Assistant context and notifications must all follow current disclosure scope. A simulated persona is never authentication.

# First use, daily landing and the monthly correction loop

First use: choose the supported operating profile without granting authority → establish company/customer/contract and responsible roles → configure the source's actual capability → obtain and validate evidence → confirm eligible input → enter the billing workbench. An empty account gets guided setup and explicit unmet prerequisites, not invented dashboard totals. All provider families remain in scope, including AWS, Azure, GCP, OCI, Alibaba, VMware, HawkEye and TokenMeter; supported/limited/unsupported/unverified branches remain explicit.

Returning operator: U01/U02 presents the current period, fixed eligible portfolio denominator, deadlines, assigned work and next actions. Monetary summaries have separate basis/currency/as-of. One primary progress stage may summarize each candidate, but blockers, issuance, delivery and ERP confirmation remain independent badges. Issued plus delivery-failed, issued plus correction-pending, and unknown ERP must remain expressible. Counts with overlapping predicates are not summed as a completed total.

The central workbench joins S03/S04/S05/S06, B04/B05/B08 and embedded A01 without removing their detailed destinations:

1. **Pin scope and evidence.** Customer, issuer, contract, period/timezone, currency/basis, source/mapping/pricing/FX revisions and as-of remain visible or available in contextual detail.
2. **Check original input.** Compare matching source billing controls with admitted source rows; inspect missing/duplicate/corrected data. BilledCost, EffectiveCost, usage and original invoice controls have distinct purposes. Missing is not zero. Receipt or a zero residual alone does not establish completeness.
3. **Explain selling calculation.** Show quantity/target → base selling price → contract discount → fees → minimum/maximum → FX → selling tax/rounding under the approved version. Fixed fees, markup, discount and raw FX have their own bases/signs. Raw source cost need not equal the selling amount.
4. **Compare independently.** Calculated selling amounts are compared with independently derived expected selling amounts on the same basis/currency/coverage. Explain legitimate transformation components separately from unexplained residuals. A renamed production output is not an independent expectation.
5. **Investigate and correct.** Open the difference, evidence and affected rules; propose a source/mapping/policy/adjustment revision through its appropriate authority. Never overwrite originals or add an arbitrary balancing line to hide a defect.
6. **Recalculate and recheck.** Preserve each attempt and comparison. New relevant revisions stale prior verification, proposals and approvals. Continue this loop as business work; do not implement an unbounded autonomous AI loop. Stop automated assistance on missing evidence, conflict, no progress, authority loss or execution budget and leave an actionable case.
7. **Approve, then issue.** Required completeness, monetary controls and nonwaivable findings must pass; policy, input confirmation, adjustment decision, amount approval, ERP approval and issuance are separate where applicable. Issued amounts are corrected through linked documents, not an old draft edit. Delivery, ERP confirmation, monthly recognition and receipt confirmation remain distinct.

# AI assistance inside the loop

The owner wants assistance to reduce repeated manual work, not a chat page detached from the case. Preserve the [intelligence-layer contract](../../architecture/intelligence-layer.md) and existing A01 scope.

| Assistant output | Required visible evidence and action boundary |
| --- | --- |
| Investigation summary | Distinguish tool-observed fact, model hypothesis, proposed change and deterministic result. Cite exact permitted source/rule revisions. |
| Cause candidates | Missing rows, duplicate application, stale rate/discount, period/unit/FX mismatch are hypotheses until supported. Lack of evidence remains unknown. |
| Correction proposal | Show affected scope/revisions, reason, change and dependencies; authoritative before/after net/tax/gross come from the deterministic preview, or show not calculated. |
| Recheck support | Invoke allowlisted, authorized read/preview/recalculation commands with expected revision and execution limits. Ordinary job identity and unknown-result handling still apply. |
| Operator handoff | Present unresolved exceptions and approval needs. The model cannot waive blockers, approve money, issue, invent payment details or directly write a ledger/ERP. |

Application of a proposal is a normal governed draft/revision action, not implicit approval. Recheck authority and evidence at execution and prevent duplicate effects. Source text/tool output is untrusted; it cannot widen tools, scope or privileges. Restrict retrieval before and after model use. A model outage preserves manual investigation and deterministic verification; accepted business jobs retain their own outcome even if the response fails. Do not silently send customer financial evidence externally.

OpenAI is a possible adapter. [Official function-calling documentation](https://developers.openai.com/api/docs/guides/function-calling) describes model tool requests followed by application-side execution and returned results; this supports keeping authorization and deterministic tools in the application. It does not certify financial correctness, data policy or this integration. No model/provider/key/endpoint has been selected or called. No customer Assistant is added by implication; any future own-bill explanation requires its own scoped design.

# Whole-product scope retained

| Existing IDs | Design responsibility in the revised story |
| --- | --- |
| U01–U04 | Period landing, close worklist, tasks and exceptional billing findings |
| O01–O03 | Customer/legal/organization relationships and effective contract context |
| S01–S06 | Source readiness, onboarding, intake/completeness, detail/evidence and reconciliation |
| B01–B07 | On-prem cost, metering, allocation, rates, FX, commitments and Marketplace evidence feeding the workbench |
| B08–B11 | Calculation versions, distinct approvals, economic application history, credits and inquiries |
| I01–I04 | Issued documents, ERP, controlled file handoff and disclosed delivery packages |
| R01–R02 | Separate cost/usage exploration, reporting and auditable evidence |
| A01 | Embedded operator assistance plus permitted full case history; same command authority |
| C01–C03 | Own billing overview, document/usage explanation and inquiries |
| M01–M05, L01–L03 | Access/user lifecycle retained; authentication method-specific design deferred |
| P01–P07 | Installation, integrations, recovery, retention, support evidence, demo and MIT notices |

SaaS product/plan/subscription, seat/usage history, change preview and entitlement state remain proposed subflows of O03/B04/B08/B09/C01–C03/U/R rather than arbitrary extra screen IDs. Payment adapters/mandates are not selected. Payment Terms and separate monthly ERP recognition remain linked to their existing contracts. Linear HB-29 explicitly defers the MSP service stage after invoice issuance; it does not remove all invoice/ERP/terms/correction planning. The issue description was read back on 2026-10-28.

# Findings to resolve in the full mockup

| ID | Priority | Evidence and consequence | Required treatment |
| --- | --- | --- | --- |
| UX-01 | High | The shown S01/S03/C02 previews repeat generic examples and explanation cards; the next billing decision and loop are not their organizing structure | Use the two persona stories and case-centered reconciliation; avoid producing 49 near-identical tables. |
| UX-02 | High | P07 spec lines33–37,49–52 retains paid-commercial/offline entitlement gates, conflicting with current MIT policy | Replace base-software activation restrictions with MIT/third-party notices and optional service information. |
| UX-03 | High | B04 lines51,69–70 and B08 lines65–68,79 omit/cross adjustment approval and stale replacement-month paths | Adopt binding-register obligations; distinguish policy, adjustment and final amount decisions and dependency invalidation. |
| UX-04 | High | B05 line50 rejects zero raw FX, fee and markup together | Positive raw FX; separate approved zero-valid fee/markup semantics. |
| UX-05 | High | I02 lines49–56,72–76,96–101 lacks explicit route-lock/manual assignment/recovery and post-posting read-back fields | Display unknown outcomes, guarded routing, initial automatic-posting-off and independent external result confirmation. |
| UX-06 | Medium | S01 lines24–26,46–52 lacks fixed FOCUS source/target/mapping versions and distinct acquisition/billing readiness | Carry actual source versions and separate readiness axes. |
| UX-07 | High | C02 line44 and S05 lines50,71 lack explicit provisional/derived/original document and full-original-scope conditions | Distinguish provenance and disclosure; do not expose multi-customer originals. S05 already excludes customer raw locators. |
| UX-08 | Medium | P06 lines17,49–55 omits operating profile and demo purpose; local registry still says development unauthorized | Separate public-sample view, synthetic validation and failure reproduction; apply current planning completion without claiming all runtime tests pass. |

These findings include confirmed pre-existing binding residuals, not eight newly discovered product requirements. Historical specifications are not rewritten in this review. HB-24 adoption/publication remains open.

# Revised visual study and verification

[MSP reconciliation workbench](storyboards/operator-reconciliation.svg) shows a synthetic blocked r7: source billed control and rows both USD1,000.00; expected selling net USD1,100.00; candidate net USD1,120.00; residual USD20.00. Base sales1,200 minus120 discount plus20 expected fee equals1,100. Candidate duplicates the fee, giving1,120. Assistant suggests investigation, not an executed correction. Source confirmation and tax review remain pending.

[Customer bill](storyboards/customer-bill.svg) is a separately labelled later fictional state, after correction, verification, approval and issuance of r8: net1,100 plus illustrative10% tax110 equals gross1,210. Issuance/due dates and terms are illustrative. Payment is not confirmed. Supplier costs and internal findings are absent. These are distinct temporal story frames, not simultaneously exposed versions.

The SVGs are self-contained static design artifacts with precomputed synthetic figures. PNG previews were generated locally and inspected; initial missing minus glyphs and word wrapping were corrected. They do not execute actions or render live data. The study covers two core workspaces, not all49 redesigned screens or a working prototype. The full mockup still needs first-use/portfolio/loop-transition/approval/issuance/exception/mobile views and detailed persona-specific interaction testing.

Structural inventory, arithmetic and final independent artifact review results are recorded in [verification](verification.md). Runtime integration, actual AI, financial engine, browser interaction, accessibility compliance, provider support and real-user usability tests are NOT_RUN for this design work. Existing source processing and internal usage/export evidence remains bounded by [HB-34](../../work/HB-34-customer-access/brief.md); it is not a working monetary billing UI.
