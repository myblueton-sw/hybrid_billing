---
type: PrototypeGuide
title: Complete billing design review v8
description: Defines the integrated 54-page bilingual review book, full v5 coverage and customer-separated review entry for first design sign-off.
status: draft
---

# Deliverable and review entry

[HB-40](https://linear.app/hybrid-billing/issue/HB-40/integrate-whole-product-billing-mockup-for-first-design-sign-off) responds to the owner's rejection of partial v6/v7 deliverables. Root owns these integration sources. Independent workflow/coverage and financial reviewers inspect the design and actual artifacts. The book is ready for owner review only after the checks in [verification](verification.md); publication does not itself fix or approve the design.

The complete v8 book contains **54 pages: two review guides and 52 screen states**. Every one of v5's 42 states is retained or explicitly replaced. [Coverage](coverage.json) maps each original state to the integrated destination. The prior global register is replaced by the 5000-target monthly operations view; historical company-level records remain in the history chapter. The old monthly close is a selected-portfolio workbench, not a competing global landing page.

Generated deliverables:

- `output/pdf/hybrid-billing-full-design-v8-ko.pdf` and its English counterpart without `-ko`.
- `output/designs/HB-40/ko/index.html` and `en/index.html`: whole review entry with chapter links, full screen index and previous/next navigation.
- Each language's `customer/index.html`: separate seven-state customer review, without operator navigation or supplier/contract examples in its page payloads.

These are self-contained design-review snapshots, not a unified running application. Financial action controls are visibly read-only; book/page links work. Different synthetic Northstar, Aster, Juniper, Cedar, C00226 and provider-contract cases are identified at the top. They are not silently joined into one invoice, contract, approval or runtime state. Existing interactive source prototypes remain available in their original directories; their execution is not recertified by this integration.

# Whole-workflow order

| Pages | Review scope |
| --- | --- |
| 1–2 | Review guide and all-screen index |
| 3–5 | Monthly operations, selected close workbench and customer profile |
| 6–14 | Source completeness, supplier invoice intake, PDF extraction/AI review, differences, accepted evidence, source reconciliation and account attribution |
| 15–29 | Common supplier/customer contracts, five provider details, renewal, recurring/one-off sales policies, impact, allocation, calculation, scoped editing, seat proration, commitments, FX/tax |
| 30–34 | Difference investigation, revision comparison, recalculation jobs and human financial approval |
| 35–39 | Assignment policy, review/issue, document/actor history, issued state and ERP destination/mapping |
| 40–44 | Monthly history, closed-period evidence, variance bridge and linked post-issue correction |
| 45–51 | Customer overview, monthly/documents history, invoice, usage, inquiry and response |
| 52–54 | Initial readiness, declared license mode and registration/expiry states |

Common review navigation follows this order throughout. OCI stays inside supplier contract details. Customer selling and supplier purchase values remain separate. All added agreement IDs, dates, names, rates and state labels are fictional; they do not establish provider access, negotiated eligibility or commercial rights.

# Financial examples and authority

The OCI fixture has 1000 synthetic units × USD0.08 = USD80, included overage0, eligible commitment opening10000 − consumption80 = closing9920. It assumes all usage is covered and no other movements/expiry/adjustments. Incremental uncovered usage0 is not an invoice total, cash-paid fact or zero-dollar invoice; supplier invoice/settlement is unavailable. It does not determine customer selling price.

C00226 remains a separate sales-policy example: supplier net1000 → 10% markup1100 → 5% sales discount1045 → proposed one-off25 gives1020 pretax. Northstar historical amounts, Cedar extraction discrepancies and other v5 cases retain their original evidence and limitations. AI text proposes/explains; no activation, posting or issuance is performed.

The license screens visualize [existing requirements](../../licensing/usage-modes-requirements.md). Noncommercial permitted use has no trial timer; commercial evaluation requires separate permission; expired terms show a nonblocking notice. No key validation, automatic email, remote lock, telemetry or rights expansion is implemented.

# Reproduction and source provenance

Required inputs are the existing v5 generated EN/KO bundles, including `previous/print.html`, `sample-source.png` and the synthetic Cedar PDF. Rebuild those using the [v5 guide](../HB-35-screen-planning/prototype-v5/README.md) and its v4/v3 prerequisites if absent. The integration checks exactly 35 legacy sections and six retained v5 routes, with the seventh route explicitly replaced. It records SHA256 of each input print document in the generated language manifest. A missing baseline blocks generation; it is not silently substituted.

With Playwright/Chromium, Python BeautifulSoup and Poppler installed:

```sh
node docs/reviews/HB-40-full-mockup/capture.cjs
python3 docs/reviews/HB-40-full-mockup/build.py
node docs/reviews/HB-40-full-mockup/render.cjs
```

Set `HB_PLAYWRIGHT_MODULE` and `HB_CHROMIUM_PATH` for an external local runtime when necessary. Generated HTML, screenshots, manifests and PDFs stay under `output/`. The static coverage map and maintained source scripts are Git-bound. The renderer requires one page per state, readable full-size layout, loaded images, local book links, separate customer navigation, representative mobile widths and no external network requests.

Unmapped source-demo actions are recorded in generated manifests and converted to static labels rather than pointing at an unrelated customer. Book navigation is independent from those historical demo controls. The guide clearly identifies the read-only nature of this artifact.
