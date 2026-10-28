---
type: PrototypeGuide
title: Monthly operations and contract pricing v7
description: Explains the bilingual seven-view revision for 5000 synthetic company-month targets and provider-aware contract pricing.
status: draft
---

# Review entry

HB-39 corrects the HB-38 navigation in this same maintained source directory. See the [independent continuity review](../../HB-39-contract-navigation/review.md). OCI is now a supplier detail under Contracts and rates; customer sales and impact retain customer/month context.

Open `index.html?lang=ko` for Korean or `index.html?lang=en` for English. Use a local browser URL or static server. This is an internal MSP design study with synthetic data, not a customer portal or billing implementation. Root owns these files under HB-39.

The seven-page focused PDFs are generated under `output/pdf/hybrid-billing-operations-contracts-v7[-ko].pdf`. They revise monthly operations and contract pricing; they do not replace every screen in the [previous v5 study](../../HB-35-screen-planning/prototype-v5/README.md). Customer portal, ERP preparation, assignment policy and other earlier flows retain their separately stated limitations.

| Page | View |
| --- | --- |
| 1 | Single-month operations, 5000 targets, 900 unissued, 4100 issued, 115 post-issue exceptions within issued |
| 2 | Company-specific work, contract/contact identity, next safe action and retained result navigation |
| 3 | Common contract register with supplier purchase/customer sales navigation |
| 4 | Selected customer MSP default, recurring exception and separate one-off grant |
| 5 | Nested OCI supplier evidence, consumption, commitment movement and settlement |
| 6 | Selected customer synthetic adjustment impact and margin basis |
| 7 | Company-month issuance history, including pending and unavailable evidence |

# Interaction and evidence limits

September has 5000 local fixtures, while July/August deliberately have no loaded snapshots. Missing evidence is not a zero-billing claim. Name/ID search, status, reason, owner, 25/50/100-row pagination, company detail and workspace switching operate locally. Only the selected result page enters the DOM. Print shows eight rows, explicitly labeled as an excerpt. This is not a production pagination, count-snapshot or performance test.

Detail uses a full-page view, not an adjacent panel, to keep the reading area and mobile return action consistent. Returning restores query/page and focus. Switching into single-company context resets selection to C00226 and constrains detail identity. This is a prototype consistency check, not server authorization.

Customer sales examples are available only for C00226 in September; other customer/month combinations show unavailable evidence. Legacy OCI and pricing links redirect within the common hierarchy. Pricing and contract views are read-only; no financial action is performed. The example uses an expressly assumed pre-tax goodwill adjustment with no fees/minimum/FX. Credit tax classification must be resolved in the real contract. September invoice/actor examples are seeded facts, not executed user actions. No real agreement, contact, account, PDF or credential is loaded. No external network requests are needed.

# Reproduction

With Playwright, Chromium and Poppler available, run `node docs/reviews/HB-38-billing-operations/prototype/render.cjs`. Set `HB_PLAYWRIGHT_MODULE` and `HB_CHROMIUM_PATH` if using installed external runtimes. The renderer writes seven-page EN/KO PDFs, screenshots and `output/designs/HB-39/verification.json`; it checks page bounds/counts and the core local interactions. Those artifacts are generated and excluded from Git.

Review the [provider evidence](../provider-pricing-research.md), [design proposal](../pricing-design-review.md) and [verification report](../verification.md) for factual sources, independent review and limitations.
