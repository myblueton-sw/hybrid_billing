---
type: ReviewReport
title: Contract navigation correction and requirement continuity
description: Records the common contract hierarchy, independent review and retained owner requirements across the focused billing mockups.
status: draft
---

# Problem and ownership

[HB-39](https://linear.app/hybrid-billing/issue/HB-39/correct-contract-navigation-with-independent-requirement-continuity), 2026-10-29, machine mac_mini, accountable assignee Seung Woo Park. Root writes the correction; independent read-only `msp_workflow_research` reviews design, actual changes and requirement continuity. The owner explicitly requested a reviewer alongside correction because earlier requirements were being forgotten.

HB-38 incorrectly promoted an OCI research example into a top-level product menu. It also left customer-policy and impact examples fixed to C00226 regardless of the selected customer. The correction uses a common contract hierarchy and refuses to substitute another customer's or month's financial example.

# Corrected navigation

- Monthly operations → company work → customer sales contract → impact preview → preserved operations results.
- Contracts and rates → supplier purchase contracts → AWS / Azure / GCP / OCI / Alibaba evidence.
- Contracts and rates → selected customer's sales contract; recurring defaults, explicit exceptions and one-off grants remain distinct.
- Company history retains selected customer identity and pending versus unavailable monthly evidence.

OCI-specific rate-card, Universal Credits, consumption and invoice distinctions remain inside the OCI supplier detail. None of the five provider entries claims a connected account or active private agreement. Evidence is a detail area, not a provider-specific main menu.

Stable routes use `contracts/supplier/<provider>` and `contracts/customer/<customer-id>[/preview]`. Legacy `oci`, `policies` and `preview` links resolve to the nested hierarchy; customer redirects retain the selected identity. Single-company context constrains the customer ID. September-only seeded policy/impact data is unavailable for other customers or months rather than reused as their pricing.

This is the navigation of a focused seven-view revision, not a declaration that the entire product has only three menus. The existing [v5 full review study](../HB-35-screen-planning/prototype-v5/README.md) remains the reference for customer portal, assignment, ERP and other unchanged flows. Its limitations remain in force.

# Requirement continuity checklist

Use this table during subsequent screen reviews. Retained references are not newly implemented or revalidated features.

| Owner requirement | Location after correction | Review boundary |
| --- | --- | --- |
| International multi-provider billing | [Provider research](../HB-38-billing-operations/provider-pricing-research.md); common supplier register with five details | Existing research reused; no new live-provider verification |
| Separate MSP operator and customer views | This study is operator-only; [v5](../HB-35-screen-planning/prototype-v5/README.md) retains customer disclosure boundaries | No customer portal expansion or supplier-margin disclosure |
| Source → rates/discounts → reconciliation → issuance | [Pricing design](../HB-38-billing-operations/pricing-design-review.md), customer impact and existing monthly work | No supplier-net discount reapplied; issuance not executed |
| 1000–5000 companies, simple filters | Existing monthly operations; 5000 local synthetic targets, search and collapsed optional controls | Local pagination only; no server-scale certification |
| Know why a bill is unissued | Reason, responsible person, due/overdue and next action retained | Unknown issuance inspected; delivery/ERP never triggers reissue |
| Monthly company lists and unissued history | Operations and company history retained; missing prior snapshots remain unavailable | No fabricated historical invoices |
| Customer detail and contract maintenance | Company ID/contact/contract reference links to the same customer's sales evidence; v5 retains renewal draft | Only C00226/September has seeded pricing, other evidence unavailable |
| Optional/required issuer and reviewer assignment | Existing detail explanation and v5 assignment workflow | No reassignment of historical actors to current viewer |
| Login user fallback when optional assignment is empty | Existing detail/history explanation and v5 behavior | Successful action's actor, not a later viewer; synthetic actors labelled |
| MSP-wide and customer-specific recurring rules | Customer sales view under common Contracts and rates | Explicit replace/add/exclude and eligibility requirements retained |
| One-off discounts and credits | Customer sales and impact view | Separate grant identity, no replay consumption; runtime concurrency unimplemented |
| EDP and peers including OCI | Provider-specific evidence nested under supplier purchase contracts | Private eligibility/terms remain unknown; no universal discount formula |
| AI-assisted PDF extraction and discrepancy lists | [v4 PDF workbench](../HB-35-screen-planning/prototype-v4/README.md), pricing proposal and impact explanation | Evidence-based proposals; no autonomous activation, posting or issuance |
| ERP destination unspecified; Odoo/standard options | v5 ERP preparation reference | Candidate connector/profile only; no assumed internal/customer ERP |
| Showback / chargeback applicability | [Multi-provider review](../HB-35-screen-planning/multi-provider-research.md) | No new allocation or chargeback engine in this correction |
| Korean and English; neutral visual treatment | Same bilingual prototype and regenerated PDFs; tables/dividers retained | No provider-themed or decorative highlight redesign |
| Authentication postponed | Existing scope retained | No authentication method chosen here |
| Noncommercial/commercial licensing and trial requirements | [License usage requirements](../../licensing/usage-modes-requirements.md) | No change to rights, keys, trial enforcement, telemetry or locks |

# Review and verification

Independent design review PASS: common hierarchy accepted with customer/month identity, preserved operations context, explicit unavailable evidence and retained prior-flow references required. Actual source/diff review PASS: reviewer inspected `app.js`, `contracts.js`, HTML/CSS and renderer, plus EN contract/policy/OCI/impact screenshots. Reviewer requested additional browser assertions for legacy customer links; these were added. No reviewer edits or external comments occurred.

Root executed the updated Chromium harness for both languages: PASS. Seven one-page captures were assembled into each seven-page PDF. The harness covers operations pagination/search/empty states, counts, missing month, unknown issuance, workspace identity, all five provider routes and return paths, legacy OCI/customer-policy/preview links, customer/month policy isolation, and operations page/focus after visiting contracts. At 390px all seven views stayed within the viewport; browser errors and HTTP requests were zero. Evidence: `output/designs/HB-39/verification.json`.

The first new browser run retained single-company state between supposedly independent scenarios and correctly refused C00001. The harness now loads a fresh document for independent cases while testing state preservation through actual in-app navigation. No product authorization bypass was added to satisfy the test.

Root inspected all seven KO PDF pages and the EN seven-page contact sheet; no observed clipping or overlap. Generated PDFs/screenshots stay outside Git. Final independent continuity/document review PASS. Reviewer inspected KO PDF pages 3–4 and contract-register/customer-policy mobile screenshots: no observed overlap or clipping beyond intentional table scrolling. Reviewer did not rerun the browser harness or inspect every PDF page. JavaScript syntax, changed-document local links/frontmatter, file-size limits and whitespace checks also passed. Root's checks are distinct from independent source/visual review. No production data, financial mutation or account access is authorized by this correction.
