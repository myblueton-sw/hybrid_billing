---
type: DesignEvidence
title: S03 external patterns and terminology evidence
description: Records first-party patterns and candidate wording without transferring provider product behavior or locale approval.
status: draft
---

# External patterns consulted

On 2026-11-10 root read [Cloudscape table view](https://cloudscape.design/patterns/resource-management/view/table-view/). Adopt its task-oriented collection heading, result count beside its own collection, bounded rows, filter-local empty results and inline read-error recovery. Reject destructive/create actions, copied vendor assets, decorative styling and any assumption of implemented sort/preferences. Use full available width, neutral separators, square controls and existing shared navigation.

Root read Microsoft's [Korean Cost Management export guide](https://learn.microsoft.com/ko-kr/azure/cost-management-billing/costs/tutorial-improved-exports) and [Arabic guide](https://learn.microsoft.com/ar-sa/azure/cost-management-billing/costs/tutorial-improved-exports). The guides distinguish export run history, refreshing observations and manifest/partition counts; the Korean guide explains that each file partition is only a subset. Adopt separate current stored-row observations versus historical receipt/run metadata and explicit source periods/observation time. Reject provider scheduling SLAs, completion guarantees, create/replay/delete actions and any inference that one receipt proves full source coverage. Our read-only status does not implement Azure exports.

# Candidate terminology

Use 조회 가능한 행 / Queryable rows / الصفوف القابلة للاستعلام for currently verified rows within authorized scope. Use 조회 가능한 수집본 / Queryable deliveries / عمليات التسليم القابلة للاستعلام for distinct current authorized delivery publications, not registered connections or whole-file receipt count. Use 자료 기간 / Source window / فترة المصدر for inclusive-start/exclusive-end delivery intervals; do not pass these as a charge-month filter. Keep 접수 메타데이터 / Receipt metadata / بيانات الإيصالات الوصفية distinct from normalized rows and 원본 / Original / الأصل. Public sample and synthetic fixture evidence are separate; unknown completeness is not0 or100 percent.

These are product-contract choices informed by regional first-party usage, not copied vendor terminology guarantees. The Arabic source uses ar-sa and does not establish UAE-native acceptance or jurisdiction. Existing brand/provider names and FOCUS identifiers remain unchanged; language never changes currency or authority. Separate AI TW review and actual rendered context are required; native-language/human practitioner acceptance remains unverified.
