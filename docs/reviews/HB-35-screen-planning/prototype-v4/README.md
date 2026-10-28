---
type: DesignArtifact
title: Monthly billing history and supplier PDF analysis prototype v4
description: Adds eight bilingual history and document-review states with a synthetic PDF, independent reconciliation gates and preserved prior workflows.
status: draft
---

The [v5 extension](../prototype-v5/README.md) is the current customer, contract, issuance and ERP-preparation entry point. This v4 evidence workflow is preserved within it.


# Revised entry points

Open `output/designs/HB-35-v4-ko/index.html` for Korean operator history and PDF review, or `output/designs/HB-35-v4/index.html` for English. Each bundle has a separate selling-only `customer.html`. The generated combined PDFs are `output/pdf/hybrid-billing-msp-design-v4-ko.pdf` and `output/pdf/hybrid-billing-msp-design-v4.pdf`, with 35 pages each.

The first eight pages address the owner's missing monthly-history and supplier-PDF analysis requirements. The prior 27 pages remain a clearly linked reference under `previous/`. Neutral typography, tables and dividers are retained. The PDFs are design reviews, not 35 independently implemented production screens.

# New screen map

| Page | State | Purpose |
| --- | --- | --- |
| 1 | monthly | Northstar July–October history, original versus adjusted basis, period filter and own-month detail |
| 2 | month | Fixed historical inputs and issued documents; September supplier s3, mapping m2 and VMware cp6 remain pinned |
| 3 | variance | August-to-September selling-net bridge, with later credit separately attributed |
| 4 | inbox | Supplier intake identity, duplicate/encrypted/unsupported/OCR failure branches and metadata-only file selection |
| 5 | extract | Three panes: original synthetic PDF, editable extraction and AI hypothesis; field/page evidence and sum checks |
| 6 | differences | Verified PDF versus collected detail; missing record and amount difference resolved without altering the PDF |
| 7 | audit | Fixed evidence acceptance and preserved E2/E3 histories; changed fields invalidate downstream checks |
| 8 | customer-months | Own monthly selling history, document references and synthetic monthly downloads; October remains unavailable |

The monthly archive concerns Northstar. The PDF workflow is explicitly a separate Cedar case, `SAMPLE-AWS-CEDAR-09`; its three-line breakdown is not the preserved Northstar `INV-AWS-09`. Switching examples is labelled. Selecting July or August opens that month's own synthetic metadata and does not attach September documents. Detailed supplier PDFs for those historical examples are explicitly unavailable.

# Monetary and time bases

All monthly examples use USD. Northstar July modeled cost/selling net are 110,000/125,000; August 116,000/130,000; September original 122,000/136,560. Margin means selling net less attributed modeled cost, not realized profit or cash receipts. October is incomplete, never a fabricated zero.

The original September invoice remains `DEMO-2026-09104`, issued October 5. `CN-NS-1007`, issued October 7, credits September by 500. The adjusted attribution view therefore shows cost 121,500 and selling document position 136,060, preserving margin 14,560. Document issue month and service attribution month remain distinct. Historical values are not recomputed using today's pricing.

The illustrative selling bridge is 130,000 + usage 5,000 + seats/services 2,000 - discount 440 = 136,560, an increase of 6,560 or 5.05%. The separately adjusted view includes the later -500, giving 136,060, an increase of 6,060 or 4.66%. Bridge evidence IDs are explicitly synthetic examples, not actual provider-derived causal findings.

Cedar's original synthetic PDF is compute 40,000 + storage 20,000 + support 4,000 = 64,000, with explicit TAX-DEMO-0. The initial extraction deliberately proposes support 400, giving 60,400 and extraction discrepancy 3,600. Human page/field confirmation and a reason create a new extraction without changing the PDF.

Collected detail initially contains compute 40,000 and storage 19,800; support is missing, not zero. Known subtotal 59,800 differs from the corrected PDF by 4,200. Separate C2 fixtures supply late storage 200 and fixed support 4,000. The 3,600 extraction discrepancy and 4,200 collection gap must never be added: they compare different objects. Cedar attribution 48,000 plus other recipients 16,000 conserves the supplier total.

# Processing and authority boundaries

File selection reads only the browser-provided name and size. It does not read file contents, upload, persist the file or call an AI service. The separately labelled sample action always loads the known Cedar fixture. The original shown on screen is an actual generated synthetic PDF rendered to PNG, not a real vendor invoice.

Extraction completion, arithmetic verification, supplier/detail reconciliation, evidence acceptance and financial approval are distinct. Accepting P1/E2/C2 means reviewed billing input only. It does not approve a customer bill, issue a document or post ERP entries. Accepted inputs become read-only; a new extraction revision preserves the previous accepted snapshot and requires fresh checks. AI text remains a hypothesis, never financial authority. Document content is evidence, not executable instructions.

The customer document and its JSON/script exclude source costs and internal evidence. Monthly sample CSVs contain only disclosed selling documents and references. This public synthetic bundle does not establish production tenant isolation. No automatic reporting, license telemetry or remote-lock behavior is included; the separate licensing discussion is tracked in HB-36.

# Reproduction

First generate v3 using its [instructions](../prototype-v3/README.md), including both-language render outputs. Then run:

```sh
python3 docs/reviews/HB-35-screen-planning/prototype-v4/build.py
node docs/reviews/HB-35-screen-planning/prototype-v4/render.cjs
HB_DESIGN_LOCALE=ko node docs/reviews/HB-35-screen-planning/prototype-v4/render.cjs
```

Use an existing trusted Playwright/Chromium installation through optional `HB_PLAYWRIGHT_MODULE` and `HB_CHROMIUM_PATH`. Poppler `pdftoppm` must be installed. The renderer generates the sample PDF from `source.html`, renders its source preview, tests the new interactions, captures desktop/mobile states and appends the previously verified 27-page reference story. Outputs stay under `output/`, not Git.

[Verification](../v4-verification.md) records actual results and limitations. Tracking: [HB-35](https://linear.app/hybrid-billing/issue/HB-35/review-planning-baseline-for-the-complete-product-screen-mockup); root writer/integrator, independent financial and workflow/visual reviewers, machine mac_mini.
