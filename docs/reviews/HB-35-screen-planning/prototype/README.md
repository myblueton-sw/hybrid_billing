---
type: DesignArtifact
title: Multi-provider MSP billing visual prototype v2
description: Provides fifteen linked synthetic screen states from supplier evidence through customer invoicing with separate operator and customer documents.
status: draft
---

# Open the design

Open [operator workspace](index.html) in a browser. Start with **Review Northstar**, or follow the footer links through the story. The [customer workspace](customer.html) is a separate HTML document with only disclosed selling information. HTML, CSS and navigation JavaScript are self-contained local assets; there are no external requests, analytics, APIs or financial operations.

The generated review PDF is `output/pdf/hybrid-billing-msp-design-v2.pdf` at the repository root. It contains all fifteen screen states, one per page. PDF/PNG files are generated local output and are not committed. This v2 artifact replaces the two narrow storyboards as the current core-flow visual proposal; the old v1.16 PDF has not been overwritten.

The [renewed research](../multi-provider-research.md) and [planning review](../review.md) supply the evidence and boundaries. Tracking: [HB-35](https://linear.app/hybrid-billing/issue/HB-35/review-planning-baseline-for-the-complete-product-screen-mockup). Root is the only artifact writer. Financial and visual reviewers are independent and read-only.

# Screen map

| PDF page | HTML state | Task | Existing families |
| --- | --- | --- | --- |
| 1 | index.html#close | Prioritize customer-period exceptions | U01–U04 |
| 2 | index.html#sources | Reconcile supplier documents and customer attribution | S03–S06 |
| 3 | index.html#aws | Trace shared AWS invoice components and source rows | S04–S06, B06 |
| 4 | index.html#variants | Explain Azure/GCP/OCI/Alibaba differences | S04–S06, B06–B07 |
| 5 | index.html#allocation | Explain cost pools, showback and conditional internal chargeback | B01–B03, R01 |
| 6 | index.html#pricing | Inspect effective selling rules, coverage and independent expectations | O03, B04–B08 |
| 7 | reconciliation.html#case | Investigate a discrepancy with simulated AI assistance | S06, B08, A01 |
| 8 | reconciliation.html#recalc | Compare original and corrected revisions | B08–B09 |
| 9 | reconciliation.html#approval | Review separate release gates and invoice grouping | B09, I01–I02 |
| 10 | reconciliation.html#issued | Distinguish issuance, delivery, ERP and payment evidence | I01–I04 |
| 11 | reconciliation.html#correction | Assess late supplier credit without editing an issued invoice | B10, I01 |
| 12 | reconciliation.html#setup | Guide a separate empty installation through prerequisites | O, S01–S03, P |
| 13 | customer.html#overview | Understand own issued charges and unavailable estimates | C01 |
| 14 | customer.html#invoice | Read itemized customer invoice and discounts | C02 |
| 15 | customer.html#usage | Inspect permitted detail and an invoice-linked inquiry example | C02–C03 |

These are connected representative states, not a claim that all 49 baseline screens, configuration editors or every provider variant have been redesigned. The full inventory and outstanding bindings remain in scope. Other portfolio rows link to explicitly labelled example patterns rather than their own records. The source-refresh, credit-correction and setup branches are illustrative states, not executed transitions. No new authentication method or post-invoice service implementation is selected.

# Synthetic financial story

Northstar is the focal September USD customer. Supplier originals total USD 126,000: AWS 64,000 plus Azure 32,000, GCP 18,000, OCI 8,000 and Alibaba 4,000. AWS is shared: Northstar 48,000 and another recipient 16,000. Therefore Northstar CSP attribution is 110,000, not 126,000.

The VMware pool is 20,000: Northstar 12,000, other customers 6,000 and retained idle cost 2,000. Northstar attributable cost including VMware is 122,000. This internal allocation is not added to its separate 15,000 contracted VMware selling charge.

Cloud selling bases sum to 134,400; customer discounts sum to 15,840; cloud selling net is 118,560. Add VMware 15,000 and one support fee 3,000 for expected selling net 136,560. Candidate r7 duplicates the support fee and reaches 139,560. Staged r8 removes the duplicate with a linked correction and retains the original revision. Worksheet W9 represents a separately authored expected control in the example; the prototype does not execute an independent financial engine.

TAX-DEMO-0 is an explicitly chosen synthetic zero-tax example, so the shown gross is 136,560. It is not a default for unknown tax or a valid statutory template. A later October supplier credit of 500 refers to September usage. The September original remains unchanged; projected supplier net positions are provisional while attribution is pending. Customer correction remains uncalculated and unpublished.

# Interaction and rendering

Working interactions: local screen navigation, customer worklist filtering (including no match), customer inquiry example dialog and focus transfer on screen changes. Buttons linking to approval or issued states are labelled as example navigation; they do not apply, approve, issue, send, download or post anything. Mobile tables and navigation scroll horizontally with a table hint. This is not tested tenant enforcement or production accessibility compliance.

The renderer requires Node.js, Playwright and a compatible Chromium installation. Use an existing trusted installation or provide `HB_PLAYWRIGHT_MODULE` and `HB_CHROMIUM_PATH`; neither variable carries credentials. Run from any directory:

```sh
node docs/reviews/HB-35-screen-planning/prototype/render.cjs
```

The command writes desktop/mobile screenshots, a render-check JSON, generated print HTML and the fifteen-page PDF under `output/`. It checks the specific local navigation/filter/dialog examples and records browser exceptions and layout observations. It does not install dependencies. Print assembly uses the desktop layout; responsive media rules are restricted to screens to prevent PDF stacking/clipping.

Render the PDF with Poppler for independent output inspection:

```sh
pdftoppm -scale-to 1440 -png output/pdf/hybrid-billing-msp-design-v2.pdf output/designs/HB-35-v2/pdf-page
```

See [visual verification](../visual-verification.md) for actual results and limits. A local file bundle is a public synthetic design artifact, not an authorization boundary; production customer endpoints must enforce access independently.
