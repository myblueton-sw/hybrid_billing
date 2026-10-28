---
type: VerificationReport
title: HB-35 planning and storyboard review evidence
description: Distinguishes source inventory, static financial and visual checks from unexecuted product and AI validation.
status: draft
---

# Scope

HB-35; root writer; machine mac_mini; 2026-10-28. Deliverables: [planning review](review.md), [source inventory](source-inventory.json), [operator storyboard](storyboards/operator-reconciliation.svg) and [customer storyboard](storyboards/customer-bill.svg). Owner directions expanded the review to international workflow evidence, separate personas, iterative financial correction and embedded AI assistance. This is not the completed whole-product redesign.

# Checks actually performed

| Check | Result and limit |
| --- | --- |
| Canonical local inventory | PASS: 49 distinct registry IDs; every referenced spec/design JSON/SVG exists; JSON/XML parsed; registry union covers all43 source requirement groups. Structural coverage does not prove every clause is correctly represented. |
| Publication boundary | 10 of49 local screen specs are currently tracked. Per-file SHA256 and tracked status recorded; remaining companions were not staged or translated. HB-24 remains open. |
| Source review | Separate read-only screen_source_review confirmed legacy license, adjustment, FX, ERP, source readiness, disclosure and demo conflicts. FAIL for unchanged adoption; PASS as an inventory with explicit correction obligations. |
| Independent design review | PASS for first-use/period landing, iterative correction, separate customer experience and embedded A01, subject to constraints applied in review.md. Reviewer required independent status axes instead of treating ready/blocked/approved/issued as inherently exclusive. |
| Synthetic arithmetic | Python Decimal check PASS: 1200 -120 +20 =1100; duplicate20 produces1120 and residual20; illustrative tax110 produces1210 gross. This checks the pictured example only, not the product calculator or an independent monetary oracle. |
| SVG structure | Both XML documents parse. No script, embedded image, foreignObject, event attributes or external href. No font files/vendor images included. |
| Local visual inspection | Root viewed both PNG renders. Corrected an unsupported minus glyph and mid-word wrapping, regenerated and viewed corrected versions. No canvas overflow reported by renderer. Static desktop observations only. |
| Current web evidence | Stripe, Flexera, Finout, Lago workflow documentation and OpenAI function-calling guide fetched. No logged-in vendor workflow executed. Lago screenshot fetch403 recorded as failed. |

# Independent final artifact review

PASS: separate read-only screen_source_review reviewed the actual Markdown, inventory and both SVG/PNG artifacts with no blocking finding. The reviewer independently checked all147 inventory hashes and tracked flags,49 unique IDs/43 source requirement references/10 tracked specs, JSON/XML, relative Markdown links/frontmatter, inert SVG structure and Decimal example arithmetic. Both PNGs were visually inspected: no visible clipping/overlap; blocked r7 and later fictional issued r8 are distinct; payment remains unknown and internal financial data is absent from the customer view. External vendor sources were not independently re-fetched. Runtime, browser interaction, accessibility compliance and actual AI/financial execution remain NOT_RUN.

# Remaining work and limits

## Renewed multi-provider research, 2026-10-28

The owner rejected the two-frame study as insufficient for full CSP invoices. The [expanded research](multi-provider-research.md) supersedes it as the next visual brief and adds AWS/Azure/GCP/OCI/Alibaba/VMware distinctions, international MSP workflows, document hierarchy, allocation/showback/internal chargeback applicability and scenario acceptance criteria. It does not claim a redesigned PDF or implemented features.

Root checked frontmatter, relative links, named provider/product coverage and `git diff --check`: PASS for document structure only. Separate csp_financial_research inspected AWS/Azure/GCP claims and local SP-04/SP-06 semantics; it found and required correcting Azure zero-charge wording to avoid confusing usage quantity with charges. Separate msp_workflow_research reviewed competitor evidence qualifications, portfolio-to-package coverage, disclosure and showback/chargeback applicability: PASS, with wording clarifications applied. Approval history now explicitly remains historical while affected approvals are stale and unusable for issuance; internal chargeback applicability is not a scope deferral.

OCI/Alibaba/Broadcom evidence was directly inspected by root, not independently re-fetched by these reviewers. IBM evidence is official search-index text with direct-fetch limitations. No renewed vendor screenshot inspection succeeded. No account, API, monetary engine, actual UI or AI runtime validation occurred. Existing two-frame arithmetic/visual PASS results above apply only to those narrow artifacts and do not establish adequacy of the whole-product design.

The full mockup needs first-use, portfolio, correction proposal, recheck/stale, approval/issuance, unknown ERP, customer document/usage/inquiry, mobile and accessibility states. Two static screens demonstrate direction only. Existing49 IDs and SaaS subflows remain in scope; no additional screen count is certified. Authentication method remains deferred; external model/provider and data handling are not selected. The post-invoice MSP service stays deferred separately from existing invoice/ERP/terms planning.

No product code, DB, cloud account, customer data, pricing/tax policy, payment, issuance, notification, deployment or AI API execution occurred. Runtime, real-user usability, browser interactions, screen-reader/keyboard compliance, financial-engine correctness and AI safety/performance remain NOT_RUN. Git-bound artifacts are English; the owner conversation is Korean. PNG previews remain local generated output, not tracked source.
