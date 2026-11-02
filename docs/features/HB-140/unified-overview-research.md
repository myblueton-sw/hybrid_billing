---
type: DesignResearch
title: Unified billing overview and monthly flow
description: Compares official billing dashboard evidence with the planned unified billing scope and proposes a monthly overview information architecture.
status: draft
---

# Decision and current implementation gap

HB-140 follow-up, requested 2026-11-02. The owner rejected PR66's four work-record counters as insufficient for unified billing. PR66 is Draft and not accepted for merge. Root owns this report; independent financial and architecture reviewers check the design, and an editorial reviewer checks clarity. No new runtime change or owner FIX is made by this report.

The original planning already requires U01 to show separate cost/sales/allocation meanings, closing denominator, blockers, unknown ERP outcomes and personal next actions. Planning ui-design.md lines 28–46, 70, 86, 101 and W01 define organization/contract/period scope and the month-specific closing matrix. MSP, standalone enterprise and group-company installations have different default subjects; a standalone enterprise must not be forced into selling contracts. The current fixture queue is a useful lower-page action list, not a suitable whole-service information model.

# External evidence

Evidence is from publicly accessible official documentation, checked on 2026-11-02; no vendor account or live tenant dashboard was inspected. Feature documentation does not establish our integration readiness. Examples are evidence of information design, not authorization to copy branding, scripts or commercial assets.

| Source | Information actually documented | Adopt / boundary |
| --- | --- | --- |
| [AWS Billing home](https://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/view-billing-dashboard.html) | Month-to-date cost, comparable prior-month interval, forecast, prior-month total; six-month breakdown by service/region/account/category; budgets, anomalies, allocation gaps and actions. | Put monthly costs and causes above action lists. Its analytical amounts can differ from invoices; do not relabel forecasts or analytical costs as final billings. |
| [Azure Cost Management reporting](https://learn.microsoft.com/en-us/azure/cost-management-billing/costs/reporting-get-started) and [cost views](https://learn.microsoft.com/en-us/azure/cost-management-billing/costs/customize-cost-analysis-views) | Organizational scope, monthly trends/budgets, actual vs amortized cost, distinct invoice/credit views. | Keep amount basis and time granularity visible. Cost analysis does not necessarily include all invoice taxes, credits or charges. |
| [Google Cloud billing reports](https://docs.cloud.google.com/billing/docs/reports) | Cost history/drivers/forecast, project/service/region breakdown; a separate cost table for invoices; monthly cost and savings breakdown including discounts/credits and invoice-level adjustments. | Link totals to their underlying composition and document. Cost reports and invoice facts are different views, not interchangeable totals. |
| [Google Cloud forecasts](https://docs.cloud.google.com/billing/docs/how-to/reports/forecasted-costs?hl=en) | Forecasts are estimates; charge-period views and billing-period views have different support. | A monthly axis must identify its time basis; do not add a forecast to an invoice-month series without a defined contract. |
| [CloudHealth Partner Reporting](https://help.cloudhealthtech.com/partners/reporting) | Monthly customer cost history; customer-billed Cost versus Partner Cost; monthly margin history; invoice report with amount, month and customer. Direct public document read by independent researcher. | MSP views need distinct purchasing and customer billing meanings. Partner cost can differ from final supplier invoices after adjustments. Do not import a margin formula without our own approved basis. |
| [Flexera Cost Planning](https://docs.flexera.com/flexera-one/cloud/using-cloud-cost-optimization/cost-planning/) | Official indexed documentation describes budget, forecast, current spend, variance, monthly/cumulative charts and dimensional tables. Direct page returned restricted/403. | Supporting evidence for hierarchy-based monthly planning, not verified live UI. |
| [IBM Cloudability Forecast](https://www.ibm.com/docs/en/cloudability-commercial/cloudability-premium/saas?topic=forecasts-forecast) and [Budgets](https://www.ibm.com/docs/en/cloudability-commercial/cloudability-essentials/saas?topic=plan-cloudability-budgets-forecasts) | Official indexed documentation describes scoped cost bases, six-month trends, plan versus actual and incomplete-source notices. Direct page returned restricted/403. | Qualify estimates with source completeness and cost basis; not proof of our forecast capability or a visually inspected page. |

CloudHealth includes public cost-history and invoice-report image links. Their availability was identified, but reviewers did not visually inspect the pixels; this is document research, not live-console or completed visual-comparison evidence. The provider sources above do not establish a complete MSP collection-to-customer-receipt workflow. Our lifecycle design must come from Hybrid Billing's planning and contracts.

# Proposed information hierarchy

This specifies information placement, not a new visual skin. Preserve the owner-approved shared frame and design constraints. A subsequent implementation must use the Claude CLI visual-design workflow and actual renders.

| Order | Region and user question | Content / destination |
| --- | --- | --- |
| 1 | Scope: whose month am I seeing? | Installation/workspace, legal entity, permitted organization/customer/contract, period basis, period, currency/basis and data as-of. Source freshness and overview refresh time remain distinct. |
| 2 | Financial summary: what changed? | Separate provider original cost (provisional/final), internal cost, planned/final sales and internal allocation where applicable and permitted. Prior-period comparable change; budget/forecast only when supported. No grand total across these meanings. |
| 3 | Monthly trend: how has it evolved? | Proposed 6/12-month range with one selected amount meaning and consistent currency/time basis; optional separate aligned panels for other meanings. Historical finalized, current provisional and forecast values visually/textually distinct. Selecting a month retains its displayed time basis; lifecycle drill-down uses an explicit linked settlement period. If the mapping is unavailable, show that limitation rather than assume the same calendar month. |
| 4 | Selected-month lifecycle: where is the close blocked? | A month-specific overview of collection, reconciliation, allocation, calculation, approval, issuance, accounting, delivery and receipts. Independent statuses per stage and eligible denominator; no universal linear completion percentage. |
| 5 | Breakdown: which source or organization explains it? | Provider/on-prem source breakdown and customer/department/affiliate breakdown with unallocated amounts and partial-source coverage called out. Same-scope details; bounded top items plus a full-list link. |
| 6 | Exceptions and next actions: what needs attention? | Missing source, unreconciled differences, approval wait, invoice/delivery failures, ERP unknown, due/overdue/receipt unknown, then personal tasks. Each action retains scope and snapshot/revision. |

Desktop places monthly trend and selected-month progress above detailed action lists. Mobile retains scope and critical deadlines/blockers first, then the trend summary and lifecycle rows; an accessible table accompanies any chart. Historical browsing is not an automatic change to the installation's operating close month.

# Monthly views are two separate requirements

1. **Monthly financial trend** compares months. The proposed range is six or twelve months (design proposal, not an existing product promise). A partial current month is compared with the same prior-month elapsed interval or explicitly marked incomparable; never silently compare it with a full prior month. A zero comparison baseline yields no percentage rather than division by zero. Missing/late data remains a gap, not a zero point or a line interpolated as fact.
2. **Selected-month processing status** explains the current snapshot for a chosen closing period. Rows are closing units such as customer/contract/legal entity, or department/affiliate units for internal allocation. Columns show independent applicable stages. Stage cells link to canonical work/evidence; completion is defined per contract path, not by visual position.

The matrix may include multiple closing periods when reviewing outstanding months; it must preserve each row's period, currency, revision and denominator. This is different from the financial time-series chart. U01 summarizes; U02 remains the full closing calendar/list/detail workspace.

# Financial and state safeguards for specialist review

- Usage/charge period, provider billing period, customer settlement period and accounting recognition period are different keys. Display the selected basis and reconcile cross-period late items; do not join on a bare YYYY-MM string alone.
- Provider original, adjusted internal cost, sales and allocated amounts are not additive components of one grand total. Allocated cost commonly redistributes the same cost and must not be double-counted.
- Preserve currency-specific totals. Any display conversion identifies its policy, rate date and unconverted source values; it does not mutate finalized documents. No margin formula is introduced in this pass.
- Version the eligible closing targets and stage-specific denominators. Exempt, not applicable, missing, blocked, failed, unknown and completed remain distinct. Do not count unsupported or unconnected stages as success.
- Issued, delivered, accounted and paid are separate facts. ERP timeout means outcome unknown until reconciliation. Partial payment and approved credit application affect remaining receivables; unknown receipts stay out of confirmed overdue totals.
- Internal showback/chargeback can terminate on its own approved path without customer invoice or collection. MSP-specific sales/receivable regions are not forced on standalone enterprise users.
- A source breakdown may include cloud, on-prem and independent meters only where planned/connected. Source completeness must qualify the whole monthly picture, not hide behind a plausible total.
- Authorize aggregate rows and fields before aggregation. Installation templates determine initial views, not permission. No hidden-customer counts in cards, charts, totals, exports or tooltips.
- Monthly recognized amounts and quarterly invoiced amounts require linked revisions and exactly-once economic effects; approved adjustments already included must not be added again. An arithmetic zero difference does not resolve unknown ERP posting.
- Receivables are an as-of balance; invoices and receipts are period movements. Do not stack these into one monthly amount. Include tax basis in each financial series definition.
- State whether organizational comparisons use historical attribution or an approved restatement. This policy remains unresolved, not inferred from current organization membership.
- Historical corrections preserve original and corrected revisions; choosing a prior month must not silently blend original invoices and later corrections.

# Proposed Korean labels for editorial review

Qualify the trend label with its financial meaning and time basis, for example 월별 금액 추이 · 원가 · 사용월 기준. Use 전체 현황, 월별 금액 추이, 선택한 월의 처리 현황, 공급자·원천별 구성, 조직·고객별 구성, 확인이 필요한 항목 and 내 작업. Name the period basis explicitly (사용 기간 / 공급자 청구월 / 고객 정산월 / 회계 인식월). Avoid a bare 완료율, 총비용 that mixes meanings, 실시간 without a freshness contract, or 매출 when the source is only a planned sale. Use 수납 확인 불가 rather than 미수 when receipt status is unknown.

# Verification boundary

This is a researched design correction, not a revised screen implementation. Current operational fixture counts remain unsuitable as financial or lifecycle evidence. Independent findings and their dispositions are recorded below. Rendering, backend contracts, forecast models, native Arabic wording and production correctness remain unverified.

# Independent review and disposition

- Financial reviewer (`monthly_financial_review`, read-only DBA): FAIL for the existing work-counter screen as unified overview; PASS WITH CONDITIONS for the proposed design. Required two monthly views, distinct time bases and stage axes, non-additive amount meanings, operating-profile applicability, true closing-target denominators, exactly-once monthly/quarterly effects, balance-versus-flow separation and scoped aggregate visibility. All requirements are reflected above; unresolved financial mappings, calendar, historical attribution, forecast eligibility and freshness policies remain explicit implementation dependencies.
- Architecture researcher (`unified_platform_research`, read-only): FAIL for the existing overview as a whole-service model; PASS WITH CONDITIONS for trend plus monthly lifecycle. Added the CloudHealth MSP and qualified Flexera/IBM evidence above. Provider cost analysis does not establish end-to-end lifecycle support. No code or vendor live session was reviewed by this researcher.
- Root's initial arrow shorthand was oversimplified. The final proposal uses separately inspectable allocation/calculation and independent accounting/delivery/receipt states. It is not a mandatory linear pipeline or universal percentage-complete funnel.
- Implementation disposition: retain PR66 as Draft. Its directory-alias fix may be reused, but its four-card primary hierarchy is rejected for the unified overview. Rework the screen from this information architecture through HB-85/HB-72 dependency coordination; do not mark a researched plan as a delivered monthly dashboard.

- Final financial re-review: PASS for researched information architecture only; no blocking contradiction remains. Runtime financial correctness remains UNVERIFIED.
- Editorial review identified a broken table separator and ambiguous month-to-settlement navigation. Root fixed both, made the Korean month heading more natural and required financial/time basis next to trend labels.
- Editorial re-review: PASS; table continuity, explicit linked-period navigation and Korean labels verified. This is not runtime, financial-calculation, accessibility or native-Arabic verification.
