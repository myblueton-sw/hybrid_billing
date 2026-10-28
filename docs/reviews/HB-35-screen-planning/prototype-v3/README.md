---
type: DesignArtifact
title: Review-driven billing operations prototype v3
description: Adds twelve bilingual operating and customer states to the preserved fifteen-state core story with local simulated decisions and reproducible twenty-seven-page PDFs.
status: draft
---

# Open the revised design

Generate the bundle with the commands below. Open `output/designs/HB-35-v3-ko/index.html` for Korean operator flows or `output/designs/HB-35-v3/index.html` for English. Each bundle's `customer.html` is a separate customer-only document and loads a separate script. PDF outputs are `output/pdf/hybrid-billing-msp-design-v3-ko.pdf` and `output/pdf/hybrid-billing-msp-design-v3.pdf`.

This revision incorporates the [external sufficiency review](../baseline-sufficiency-review.md). It supplies twelve additional working design states, retains the fifteen earlier core states and keeps neutral typography, tables and dividers. There are no colored highlights, decorative AI motifs or rounded-card treatments. The 27-page count is not certification of all 49 baseline screens or production billing readiness.

# Cases and review coverage

| PDF pages | Case / design task | Review finding addressed |
| --- | --- | --- |
| 1–2, 7–8 | Aster missing fixed-fee intake → account assignment → job read-back/recalculation → return/reverify/reapprove → separate issuance | Missing source, ownership, repeated correction, uncertain job and approval paths |
| 3 | Juniper effective-dated pricing draft; conflict and Marketplace exclusion; selected discount updates result | Missing price editor, coverage and impact preview |
| 4 | Annual commitment, monthly frequency, September actual-day seat adjustment and renewal explanation | Missing subscription lifecycle/proration |
| 5 | Provider-supplied commitment allocation, unused burden, recovered prepayment and typed GCP offsets | Missing commitment/credit mechanics |
| 6 | Separately pinned p5-FX scenario; source/selling FX, illustrative tax, rounding and split documents | Missing non-USD/nonzero-tax and grouping cases |
| 9 | Northstar supplier credit → contract treatment → checked correction → approval → linked credit issuance | Previously uncalculated late-credit case |
| 10–21 | Preserved earlier twelve operator states | Full source-to-selling explanation and provider/VMware examples |
| 22–24 | Northstar issued document history, sample downloads, inquiry and response | Missing customer document/correction/inquiry paths |
| 25–27 | Preserved earlier three customer states | Own bill, itemization and usage explanation |

Start at Aster intake. Accept the September F1 evidence, assign Aster for the displayed interval, read back R-A1, queue R-A2 and advance through running to verified. Enter a return reason, attach evidence and reverify A3, approve, exercise the stale-input branch, restore/reverify/reapprove, then issue. Issued mapping cannot be replaced. Reloading resets the local simulation; no server state is changed.

The initial forms intentionally include blocked examples. They display validation and consequential results. Date fields are editable, but only the stated September fixture is supported; other intervals explicitly require a new impact preview rather than pretending to calculate arbitrary data. Price editing supports the supplied 0%/10% examples. Saving a draft does not execute processing or approval. The FX page is a separate fixed 10% snapshot `p5-FX`, not a live continuation of an edited draft.

The refreshed-source branch exposes a new input/version scope and pending acquisition because no new file is supplied. The existing-data path runs the full staged loop. This is an explicit pending state, not a successful live connector. The customer history is labelled as a later issued snapshot and can be inspected independently of the operator simulation.

# Financial and disclosure boundaries

Aster source 9,000 usage + 1,000 fixed fee = USD 10,000; the duplicate 100 stays quarantined. Synthetic PRICE-A1 is explicit 1:1 pass-through with no fees/discount and TAX-DEMO-0. The corrected candidate therefore has expected/candidate net and gross 10,000. No real tax policy is selected.

Juniper eligible usage 10,000 less 10% plus negotiated Marketplace 1,000 gives selling net 10,000. The 0% alternative gives 11,000. The separate p5-FX snapshot uses 10,000 × 0.90 = EUR 9,000 net, illustrative tax 900 and gross 9,900; supplier cost conversion is separately 8,000 × 0.92 = EUR 7,360. It is not added to customer gross. Different issuer/currency J-USD remains a separate USD 200 document; no cross-currency grand total is claimed.

Northstar preserves original `DEMO-2026-09104`, issued 2026-10-05, USD 136,560. The specific synthetic clause CREDIT-NS-1 passes the attributed service credit through at 100%. Separately approved and issued `CN-NS-1007` is -500 with tax zero, giving net document position 136,060. This is neither a confirmed payment nor an outstanding receivable. Supplier original September GCP 18,000 is unchanged. The original v2 unresolved-credit state is the earlier point in this same case.

The commitment allocation 100 is provider-supplied evidence, not platform-reconstructed amortization. Recovery already completed is not billed again. Proration, credit types, Marketplace and rounding are explicit applicable examples, not universal provider policies. AI remains evidence/proposal assistance; it does not approve money.

Customer sample TXT summaries and CSV selling detail are generated locally and labelled synthetic. They are not statutory invoices or proof of payment. Customer HTML/script contains no operator evidence payload. This is document separation in a public synthetic bundle, not tested tenant enforcement or authentication.

# Reproduce and verify

Use Python 3 plus an existing trusted Playwright/Chromium installation. No dependencies or network data are installed by these commands:

```sh
python3 docs/reviews/HB-35-screen-planning/prototype/localize.py
python3 docs/reviews/HB-35-screen-planning/prototype-v3/build.py
node docs/reviews/HB-35-screen-planning/prototype-v3/render.cjs
HB_DESIGN_LOCALE=ko node docs/reviews/HB-35-screen-planning/prototype-v3/render.cjs
```

If needed, set `HB_PLAYWRIGHT_MODULE` and `HB_CHROMIUM_PATH` to existing installations. The renderer checks local decision paths, rejects invalid transitions, checks downloads/inquiry context, inspects three mobile layouts and generates screenshots, PDFs and `render-check.json`. Its checks demonstrate the local synthetic state machine only.

Render final PDFs with Poppler for visual inspection. Generated HTML/PDF/PNG/download samples stay under `output/`, outside Git. Maintained sources are separated into bilingual content, forms, generator, operator/customer interaction scripts, styles and rendering checks. No financial API, external message, model call, real account or authentication method is connected. Authentication and separately deferred post-invoice service scope remain unchanged.

See [v3 verification](../v3-verification.md) for executed checks, independent review findings and limits. Tracking: [HB-35](https://linear.app/hybrid-billing/issue/HB-35/review-planning-baseline-for-the-complete-product-screen-mockup); existing PR32; machine mac_mini. Root owns writing/integration, with independent financial and workflow/visual review.
