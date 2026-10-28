---
type: PrototypeGuide
title: Customer, issuance and ERP preparation prototype v5
description: Explains the bilingual seven-screen customer and monthly issuance extension and how to reproduce its 42-page review books.
status: draft
---

# Scope

[HB-35](https://linear.app/hybrid-billing/issue/HB-35) extends the previous 35-state prototype with seven operator screens. All records, contacts and actions are synthetic. The original v4 page 18 monthly-close entry is preserved as page 25 in this edition; customer-name links now lead to the new customer detail. New pages 1–7 are the current review entry points.

| Page | Purpose |
| --- | --- |
| 1 | MSP → month → company issuance register, including unissued targets; company/month/status/actor filters, single-company view and optional actor columns |
| 2 | Customer identity, billing contacts, contract dates and source attribution |
| 3 | Contract version register and future renewal draft, without activating new rates |
| 4 | Invoice identity, designated owners, actual actors and retained action history |
| 5 | Independent optional/required issuer and reviewer assignment policies |
| 6 | Synthetic review/issuance with assignment, evidence and current-review gates |
| 7 | ERP preparation: destination owner, purpose, connector candidate and missing mapping evidence |

# Behavior and boundaries

The default is optional issuer assignment and required reviewer assignment, an adjustable design assumption. Optional assignment never skips review. An unassigned optional successful action records the simulated signed-in name, ID and UTC time. Required missing assignment or a designated-user mismatch blocks that action. Failed actions do not fabricate completed actors. The demo permits the same person to review and issue; it does not establish a production segregation-of-duties policy.

Open-draft assignment, evidence or policy changes invalidate its current review while preserving old events. Issuance requires current successful review and creates one fixed invoice snapshot. Changing the viewing user or policy after issuance does not rewrite it. The local state resets on a full reload; no persistence, authorization enforcement or real invoice issuance is claimed.

Northstar July/August reuse the v4 original invoice IDs and amounts. June and other-company rows are additional synthetic examples. Same-named Northstar records under different MSPs have separate IDs. Sidebar context follows the selected company/MSP rather than silently opening another example. Prior customer HTML retains its separate navigation and selling-only payload. The static PDF is a combined design review book, not a tenant portal.

Contract renewal is a draft-only interaction. No contract activation, signed-contract upload, production customer editing or ERP connector is implemented. ERP remains unspecified: Odoo is a candidate, UBL/Peppol is an invoice/credit-note profile candidate, and ledger recognition requires separate approved accounting mappings. Sending data is not proof of posting. The screen deliberately leaves readiness disabled.

License usage modes and commercial evaluation are tracked separately under HB-37; these screens do not enforce licensing or change authentication scope.

# Reproduction

First build/render the existing v4 bundle following its [guide](../prototype-v4/README.md). Its generated 35-state print document and synthetic supplier source image are inputs here.

```sh
python3 docs/reviews/HB-35-screen-planning/prototype-v5/build.py
HB_DESIGN_LOCALE=ko node docs/reviews/HB-35-screen-planning/prototype-v5/render.cjs
HB_DESIGN_LOCALE=en node docs/reviews/HB-35-screen-planning/prototype-v5/render.cjs
python3 docs/reviews/HB-35-screen-planning/prototype-v5/assemble.py
```

Set `HB_PLAYWRIGHT_MODULE` and `HB_CHROMIUM_PATH` for the local Playwright/Chromium installation when needed. Poppler supplies `pdfinfo`, `pdftoppm` and `pdfunite`. The renderer verifies browser behavior, writes seven one-page PDFs and regenerates the previous 35-page flow with updated navigation. Assembly rejects unexpected page counts and produces 42-page books.

Generated HTML, PDFs, screenshots and `render-check.json` live in `output/designs/HB-35-v5[-ko]` and `output/pdf/`; they are not maintained sources. See the [verification report](../v5-verification.md) for actual checks and limitations.
