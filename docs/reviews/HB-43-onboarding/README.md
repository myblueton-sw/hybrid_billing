---
type: PrototypeGuide
title: Complete bilingual billing design v10
description: Adds entry, permissions and first collection to the full review and expands customer payment and master agreement fields.
status: draft
---

# Review entry

[HB-43](https://linear.app/hybrid-billing/issue/HB-43) contains 89 pages: all 66 v9 pages plus sixteen connected entry/onboarding states two AI pricing/invoice states and five cost exploration/Marketplace states. The retained customer detail is expanded with business registration, composable payment conditions, payment methods and master/child agreements. It was v9 page 5 and is now v10 page 21. The new access/onboarding pages are 3–18. Instance discount search is p78; the AI stacked-discount and customer invoice states are p60 and p79. The [duplication review](feedback-review.md) links new requests to existing requirements and official Marketplace evidence. Start with the guide and all-screen index.

The separate [delivery checklist](checklist.md) tracks every owner-requested workstream with design status and unchecked runtime acceptance. [Requirements](requirements.md) define scope and unresolved decisions; [coverage](coverage.json) maps every retained and added page; [verification](verification.md) records actual checks. Authentication method/product remains unselected. These are synthetic read-only design snapshots, not runtime authentication, cloud connectivity or financial execution.

Generated review entries are `output/designs/HB-43/ko/index.html` and `en/index.html`. Each locale's `customer/index.html` includes only customer review pages; the full book is an operator/designer artifact, not a deployable customer portal or authentication boundary. Generated PDFs are `output/pdf/hybrid-billing-full-design-v10-ko.pdf` and `output/pdf/hybrid-billing-full-design-v10.pdf`.

# Reproduction

Rebuild the [v8 prerequisite](../HB-40-full-mockup/README.md) if its captures or original v5 inputs are absent. This builder imports retained v9/v8 screen definitions and captured baseline inputs without changing prior source books. Python BeautifulSoup, Playwright/Chromium and Poppler are required.

```sh
python3 docs/reviews/HB-43-onboarding/build.py
node docs/reviews/HB-43-onboarding/render.cjs
```

Set `HB_PLAYWRIGHT_MODULE` and `HB_CHROMIUM_PATH` for an external browser runtime. Generated manifests, screenshots, render reports and PDFs remain under `output/`, excluded from Git. Inspect rendered PDFs as well as automated bounds. Maintained generators and English requirements are Git-bound. The owner fixed v10 as the base design on 2026-10-29; [baseline decision and module management](../../design-baseline-v10/README.md) govern subsequent changes. Live integrations, runtime security/financial tests, detailed residual decisions and the earlier unresolved CFRC definition remain pending.
