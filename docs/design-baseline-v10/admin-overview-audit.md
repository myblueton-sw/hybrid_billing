---
type: DesignCoverageReview
title: Settings administration and overview screen gaps
description: Narrows the planning coverage review to missing or partial settings, administrator and overview views in the current design book.
status: draft
---

# Scope

[HB-55](https://linear.app/hybrid-billing/issue/HB-55/document-settings-administration-and-overview-screen-gaps). Owner requested only a check of settings, administrator and overview omissions, followed by a Git commit. Root owns this report and its index link; a separate read-only reviewer checks classifications. No screens or product behavior change. This refines [HB-54](coverage-audit.md), without repeating the billing/contract/VMware audit.

Compare the original authorized local U/M/P/R screen inventory and specifications with the [v10 registry](registry.json), actual generated v10 page contents, [screen bindings](../ui/screen-binding-register.md) and current onboarding r2. Original specification or old SVG existence is not the same as inclusion in the current complete design. Missing below means a dedicated current-book operational view is missing, not that the requirement has never been written or the backend is absent.

# Dedicated views missing from the current book

| Area | Missing current visual view | Existing requirement / important limit |
| --- | --- | --- |
| Service accounts and credentials | Service identity list; owner/purpose/allowed scope; credential expiry/rotation/revocation; use history and owner handoff | M04. Current onboarding accepts references and mentions protected registration but does not render its management screen. No secret values in ordinary lists. |
| Operational notification settings | Delivery channels, event/severity rules, escalation/alternate owner, failed-delivery status; personal preferences kept separate | P02, U02, M05 and FA-13 binding. This is not a new request for every external integration. |
| Installation diagnosis and support | Installed profile/version, environment checks, resource/connection diagnostics, support bundle preview and bounded support access | P01. p87 is initial billing readiness, not a running installation diagnosis/support workbench. Installation role does not automatically see customer financial data. |
| Recovery and migration | Restore-set verification, isolated recovery, external unknown-result reconciliation, opening-balance/customer migration and safe resumption | P03. Account/password recovery is a different journey already represented at p7/p10. |
| Retention and deletion | Retention rules, legal holds/dependencies, deletion preview/status, archive restore and backup-expiry distinction | P04. Invoice history does not substitute for retention administration. |
| Cross-domain audit search | Search by actor, action, target, UI/API/AI/service path and result; masked before/after changes, approval/evidence linkage and scoped export | R02. PDF audit p26 and invoice/correction histories exist; the missing item is a cross-domain administrator/auditor workbench. |
| Support-profile and scenario administration | Supported provider/product/version evidence, limitations; demo profile/purpose, scenario revision, expected results and failure reproduction | P05/P06 and SR-03. These can be grouped under advanced operations; not every item needs a new top-level menu. |

# Existing but partial administrator and overview views

| Area | Already represented | Remaining screen detail |
| --- | --- | --- |
| User/invitation administration | p5 invite acceptance; p9 roles and invite preview | M01 invitation/member list, expiry/reissue/revoke, batch removal and result review; M03 scoped grant editing and effective-permission/impact preview. Do not call roles or invitations wholly absent. |
| Security and recovery administration | p6 login, p7 verification/recovery, p10 session/revocation/trust rules | Actual session list and revoke result, last-admin/custodian request-review-return workflow. Authentication vendor/method selection remains owner-deferred. |
| Personal settings and work inbox | Assigned owner filters and worklists p19–20; role landing examples p8 | M05 personal notification/security preferences; U03 unified assigned jobs/alerts with waiting/partial/failed/unknown states and next actions. Existing billing worklists are not missing. |
| Operational overview | p19 5,000-company issuance overview; p20 selected portfolio close workbench; p80 customer overview | U01's broader per-permission cost/sales/allocation and receivable freshness view is only partial in this issuance-focused book; full standalone-enterprise/group profile variants are not demonstrated. Do not invent a second universal dashboard or expose financial data to an installation operator. |
| Overview-to-exception navigation | p19 reasons/owners and p20 blocked/ready/source/ERP examples | U04 anomaly review and U03 work ownership need dedicated current-book details, without conflating an anomaly with a proven reconciliation error. Issuance delay versus payment overdue also needs the HB-54 wording correction. |

# Not missing or intentionally deferred

Initial administrator registration, login/verification, role descriptions, workspace selection, first-connection onboarding, basic legal/currency/period setup, customer profile, ERP mapping and license modes/key status are represented. Existing source/PDF audit and financial history must be reused, not replaced because cross-domain audit is incomplete. R01 reporting is also partial rather than absent: pivot/raw/monthly views exist; saved views, export-job status and historical-versus-current attribution need further detail.

Authentication product/method decisions and current onboarding visual refinement are explicitly deferred. A general third-party API/plugin product remains optional; channel administration required by operational notifications does not make every integration mandatory. Do not add unrelated infrastructure management or optimization features to this audit.

# Review and verification

Root inspected actual v10 HTML contents for operations p19, close workbench p20, roles p9, sessions p10 and initial setup p87; reviewed the original screen registry and U01/R02/P01/M04 specifications, with P03/P04 and related binding evidence carried forward from HB-54. Original local companions remain outside this commit. This is a content coverage check, not new browser rendering, runtime verification or a new exhaustive original-DOCX review.

Only this report and the baseline index link are changed. Held visual drafts, original planning, all frozen pages and runtime acceptance remain unchanged.

Independent actual-report review: PASS for the scoped settings/admin/overview classifications. Root metadata/link/whitespace checks: PASS. No runtime or visual approval is inferred.
