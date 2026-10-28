---
type: VerificationReport
title: HB-35 visual prototype v2 verification
description: Records financial-content, navigation and rendered-screen verification for the fifteen-state synthetic billing design.
status: draft
---

# Scope and ownership

2026-10-28, HB-35, machine mac_mini. Root wrote the [prototype](prototype/README.md). Separate csp_financial_research reviewed design and actual financial/disclosure content. Separate msp_workflow_research reviewed actual navigation semantics, desktop/mobile screenshots and rendered PDF defects. This is a design artifact with local navigation, not a monetary product implementation.

# Checks and findings

| Check | Observed result |
| --- | --- |
| Independent design review | PASS before authoring, with explicit original-versus-attributed cost, pool conservation, synthetic tax and staged-approval constraints. |
| Independent financial artifact review | Thirteen independent Decimal controls PASS for supplier originals, customer attribution, pool, provider bridges, sales/discount totals, duplicate fee and customer AWS details. Late-credit ambiguity found and corrected: September original unchanged; October credit ID/date and September usage retained; projected positions provisional pending attribution. Reviewer re-read corrected HTML: PASS. |
| Local navigation | Playwright executed worklist matching/no-match filter, operator case and correction-preview navigation, customer inquiry dialog opening/closing: PASS. No browser page errors. Waiting for rendered visibility replaced an initially premature navigation assertion. |
| Local links | Independent static review found all 161 link targets valid, including the inquiry alias. Aster/Atlas and late-credit links were clarified as different staged patterns, avoiding a false implication that another customer's record or the same correction case opened. |
| Desktop renders | All 15 screen states captured at 1440px viewport. Renderer reported no visible-overflow elements. Independent reviewer inspected all 15 PNGs: no observed overlap/truncation. |
| Mobile samples | Close, discrepancy and customer invoice captured at 390px; document width remained 390px. Independent visual review confirmed no outer horizontal overflow. Tables/navigation require horizontal scrolling; a table scroll hint was added. This is not all-state mobile or accessibility certification. |
| PDF | Chromium generated 15 pages at 1080 × 1050pt (1440 × 1400 CSS px). Poppler rendered all pages. Initial fixed height and print-responsive stacking caused clipping; screen-only responsive rules and final page sizing corrected it. Root inspected all 15 final pages as contact sheets and dense source/correction pages at full image size. Independent reviewer re-inspected final Poppler pages 2 and 11 and closed the clipping/overlap blocker: PASS. |
| Production behavior | NOT_RUN: source acquisition, financial engine, AI, approval authority, invoice issuance, delivery, ERP, payments, authentication and tenant isolation. All visible status transitions are authored synthetic examples. |

# Deliverables and limits

Canonical HTML/CSS/navigation plus a reproducible renderer are under `prototype/`. Generated PDF, PNGs and render-check output are under `output/pdf/` and `output/designs/HB-35-v2/`; they are not staged. No vendor images, real financial documents, credentials or customer data were used.

The visual set addresses the principal monthly MSP-to-customer story, provider differences, allocation/showback/conditional chargeback, repeated correction and later supplier credits. It does not certify all 49 baseline screens or every provider/account/tax variant. The separate empty installation is explicitly labelled. Customer HTML contains selling information without operator evidence payloads; this static separation is not an implemented security control.
