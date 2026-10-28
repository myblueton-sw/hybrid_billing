---
type: VerificationReport
title: HB-38 research and focused prototype verification
description: Records primary-source review, independent financial checks and bounded synthetic UI validation.
status: draft
---

# Scope and ownership

[HB-38](https://linear.app/hybrid-billing/issue/HB-38/redesign-monthly-billing-operations-for-1000-customer-msp-usability); root sole writer, machine mac_mini, 2026-10-28. Accountable ticket assignee Seung Woo Park. Existing public scoped publication/main integration authorization is retained. The ticket remains open until verified merge read-back and release.

Deliverables: [provider research](provider-pricing-research.md), [pricing/operations proposal](pricing-design-review.md), and [bilingual prototype](prototype/README.md). This work does not implement financial processing or connectors.

# Independent reviews

- `csp_financial_research`: PASS for actual AWS/Azure/OCI report claims and pricing proposal. Reopened Oracle cost-report, rate-card, computed-usage, statement and invoice primary documentation. Independently verified Decimal example 1000 → 1100 → 1045 → 1020 and target counts 900 + 4100 = 5000. Separately reviewed actual prototype pricing/OCI/impact content: PASS, with margin-display improvement applied.
- `msp_workflow_research`: PASS for actual GCP report and operations/pricing proposal. Applied requested Preview qualification, credit-memo evidence boundary and pooled one-off entitlement clarification. Initial prototype review FAIL: single-company workspace could retain another company's detail and active filters became hidden. Root corrected identity selection/routing and kept active advanced filters open. Final source re-review PASS: both blockers resolved; reviewer also inspected KO detail/history screenshots.
- Alibaba/IBM/Broadcom primary-source research came from separate `license_transition`; final scoped review of the actual synthesis PASS; settlement terminology and VMware design-inference qualification were clarified. No live account, private agreement or API response was checked by any reviewer.

# Checks actually performed

Root ran the Chromium renderer for EN/KO. It produced six pages per PDF and passed local checks for default 25 rows, 100-row selection, pagination, company search, whole-month totals remaining unchanged by search, empty search, preserved detail-return page/focus, unknown-outcome inspection-only messaging, issued exception subset, unavailable prior month, single-company mode, workspace/detail identity, unknown customer and 390px viewport width. Browser page errors and HTTP requests were zero. Result file: `output/designs/HB-38/verification.json`.

An initial print overflow and extra page were found and fixed using an explicitly labeled eight-row print excerpt and compact print spacing. Initial navigation assertions ran before hash rendering completed; the harness now waits for the destination content. These were test timing corrections, not financial-engine validation.

Final rerender PASS for both locales after all UI changes. Root visually inspected all six KO PDF pages and an EN six-page PDF contact sheet: no observed clipping or overlap. Independent reviewers inspected selected EN/KO screenshots, not every final PDF page. JavaScript syntax checks passed. Frontmatter, local links, file-size limits, JavaScript syntax and scoped whitespace checks PASS. Staged scope inspected: README plus eight HB-38 source/report files only; no generated PDF, screenshot, credential, customer data or unrelated companion assets staged. Remote CI availability and merge read-back are recorded in the PR and ticket.

# Explicit limits

All 5000 entries are local synthetic fixtures. Server query performance, consistent snapshots, tenant authorization, real-user usability, assistive-technology compliance, live exports, contract eligibility, tax, production rounding, atomic credit consumption, actual review/issuance/ERP and AI extraction remain UNVERIFIED or NOT_RUN. No real billing record, customer data, payment, email, cloud resource or license enforcement changed.
