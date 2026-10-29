---
type: PrototypeGuide
title: Complete billing design v9 with automatic contract pricing
description: Connects the automatic pricing requirements and twelve added states to the full bilingual review book.
status: draft
---

# Review entry

The complete v9 review contains 66 pages: all 54 v8 pages retained plus 12 new screen states. Start with the guide and all-screen index. [Requirements](requirements.md) define automatic matching, bulk pricing, discount exclusions and approval boundaries; [verification](verification.md) records actual checks. The source evidence is [HB-41](../HB-41-msp-pricing/research.md).

Generated files are `output/pdf/hybrid-billing-full-design-v9-ko.pdf` and its English counterpart without `-ko`. Navigable entries are `output/designs/HB-42/ko/index.html` and `en/index.html`; each locale's `customer/index.html` includes only customer review pages. These are static synthetic design snapshots, not deployed financial automation.

The added flow is contract portfolio → customer rulebook → rule editor → exclusions → native mapping → bulk preview → exceptions → activation review → calculation state → line evidence → version history → customer explanation. These pages are inserted into the existing contract, review, history and customer chapters. Existing intake, PDF/AI, supplier contracts, issuance, ERP, history, setup and license screens remain in the same complete book. [Coverage](coverage.json) maps retained and added pages.

# Reproduction

Rebuild the [v8 prerequisite](../HB-40-full-mockup/README.md) when its captures or v5 inputs are absent. The v9 builder reuses frozen v8 new-screen definitions and the original captured inputs; it does not mutate old source books. Python BeautifulSoup, Playwright/Chromium and Poppler are required.

```sh
python3 docs/reviews/HB-42-auto-pricing/build.py
node docs/reviews/HB-42-auto-pricing/render.cjs
```

For an external runtime, set `HB_PLAYWRIGHT_MODULE` and `HB_CHROMIUM_PATH`. Generated manifests, render reports and PDFs remain under `output/`. PDF images must also be inspected before delivery. Requirements and maintained generators are Git-bound; generated screenshots/PDFs are not. The owner has not yet fixed the design, and CFRC's meaning remains unresolved.
