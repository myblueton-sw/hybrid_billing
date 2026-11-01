---
type: ReviewReport
title: HB-63 baseline preview gap audit
description: Distinguishes route coverage from missing business journeys in the current multilingual preview.
status: draft
---

# Scope and verdict

This report records c8187fe before remediation. See [subsequent remediation](gap-remediation.md) for current changes; retain these findings as distinct review evidence.

Ticket: [HB-63](https://linear.app/hybrid-billing/issue/HB-63/implement-multilingual-whole-product-scalable-screen-redesign). Audited implementation: `c8187fe`. Review completed 2026-11-01, machine mac_mini. Root owns the report and runtime observations; research_review independently inspected source, earlier screen specifications and the built manifest without modifying files.

**PASS for route inventory; FAIL for complete business-preview coverage.** The catalog contains 112 routes and maps all 49 registry IDs. It has 4 overview, 19 form, 31 list, 18 workflow, 29 detail, 3 file and 8 report routes. These numbers do not establish business acceptance. The contribution page is a dedicated detail implementation; most other routes dispatch to shared generic renderers.

No interface implementation is changed by this audit. Existing onboarding and analysis modules are untracked work in progress, not imported by app.js or emitted by the build. Historical specifications provide comparison evidence, not automatic approval of every historical proposal or obsolete planning gate.

# Missing journeys

P1 means resolve before presenting the preview as a complete business journey; P2 means resolve before final design acceptance. Priorities describe design completeness, not a demonstrated production incident.

| Priority | Area and current state | Missing preview outcome | Evidence |
| --- | --- | --- | --- |
| P1 | Onboarding: eight setup routes exist as generic forms, lists and workflow checks. | Provider-specific existing/new setup, administrator handoff/resume, diagnostic repair, first receipt and separate technical/data/financial readiness. IBM Cloud and VMware are present in the list provider filter but absent from the current setup-connectors selector. NAVER/KT exist as unverified options, without dedicated operable setup. | app.js form/workflow/bindContent; catalog.js setup-connectors; ../../03-onboarding/r2/setup-workflow.md. |
| P1 | AI Assistant: two plain question/scope inputs. | Scope/period/currency, progress, partial/error recovery, facts versus interpretation, evidence/timestamp and review-task draft. Contextual analysis entry is absent. | app.js form; catalog.js assistant; ../../../ui/screens/A01/spec.md; tools/design/build-console.mjs. |
| P1 | Calculation and approval: generic checks and informational modal. | Before/after amounts by currency, fixed revision, blockers, rejection reason, distinct review/approval/issuance states. | app.js workflow/actionPreview; ../../../ui/screens/B08/spec.md and B09/spec.md. |
| P1 | Documents and ERP: generic details or draft-save form. | Invoice body and tax totals, approved statement versus ERP draft comparison, waiting/unknown outcome, issuance versus delivery failures and safe recovery. | catalog.js document/ERP routes; app.js detail/form; ../../../ui/screens/I01/spec.md and I02/spec.md. |
| P1 | Customer portal: generic overview and forms/details. | Customer-only summary and navigation, published document/correction relationships, permitted download state, document/line-linked inquiry, case number and conversation. Runtime customer overview currently displays global synthetic totals and operator-area links. This is a design context defect, not proof of a real data leak. | app.js overview; ../../../ui/screens/C01/spec.md, C02/spec.md and C03/spec.md. |
| P2 | Reports: all eight report routes fall through to list(). | Period/comparison period, amount basis, currency/grouping, report-specific summaries and scoped evidence/export. Declared report fields are not rendered as filters. | app.js content/list; catalog.js reports/cost-pivot; ../../../ui/screens/R01/spec.md. |
| P2 | Record navigation: generic modal details. | Retained record identity through related evidence, addressable full detail and Back preserving filters/page/selection. Drawer versus modal/page decision is not yet applied. | app.js resetQuery/route/rowDetail. |
| P2 | Large-file and business recovery: bounded metadata intake and timer-driven jobs. | Visible interrupted-transfer/resume, partial batch results, retryable subset, waiting/unknown result and linked actor/target/revision history. Real transfers are unnecessary for an honest simulated design. | widgets.js fileIntake/createJobs; data.js rowAt; ../../../ui/screens/U03/spec.md and R02/spec.md. |

# Cross-cutting acceptance still open

- The rail is a review-screen index, not a finalized role/task menu. Customer and operator entry/navigation require separate review.
- Brand SVGs, local fonts, semantic colors and logical RTL are integrated. Native Arabic/English/Korean practitioner terminology approval, full screen-reader verification and UAE conformity acceptance remain unverified; this is not a finding that all translated wording is wrong.
- Sponsorship benefits and individual/company recognition guidance are implemented. Actual roster, listing consent verification, live tier publication and a management/synchronization interface are not implemented. Work credit beyond the Git contributor link still needs evidence-backed entries.
- Common empty/loading/error/expired-selection states, bounded pagination and account search already exist. They must not be reported as entirely missing. Domain-specific blocked/partial/revoked/recovery scenarios remain incomplete.

# Actual verification

Root executed a Node catalog inventory and inspected runtime imports/build inclusion, shared renderers, data fixtures and prior verification. Independent reviewer reproduced inventory and confirmed build omissions. Earlier 98 passing generic checks are retained evidence, not new business acceptance.

Root ran `/private/tmp/hb63-gap-qa.cjs` against the built loopback preview with Playwright/Chromium. Observed:

1. setup-connectors contains provider, method and account only; provider options omit IBM/VMware.
2. assistant contains question and scope only.
3. cost-pivot shows generic search/provider/status/sort/page controls, without declared period/currency/grouping controls.
4. core-customer-overview shows 100,000 accounts, 1,000,000 files and operator-area links.
5. index-approval shows version/reason step buttons, without amount comparison or approval decision inputs.
6. Entering `REC-0000007` in index-intake, navigating to supplier-aws and returning with browser Back produces an empty search field.

Independent source/spec review agrees on the eight journey gaps above. It did not rerun the browser. Root spot-checked five routes and Back navigation, not every screen state. No live CSP, AI, ERP, financial operation, external message or customer data was used.

# Recommended completion order

First integrate onboarding and scoped AI scenarios, then correct customer navigation and record context. Next provide one end-to-end synthetic source → calculation comparison → human approval → document → customer inquiry journey. Complete reporting and partial/recovery variants against that shared context. Track each finding by its preview outcome rather than adding more menu entries.

Production authentication, durable storage, actual provider/model/ERP calls, monetary calculation and throughput validation remain separate backend delivery. Their absence alone is not a reason to reject a clearly labeled synthetic preview; the missing user-visible journeys are.
