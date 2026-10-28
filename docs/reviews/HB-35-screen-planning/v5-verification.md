---
type: VerificationReport
title: HB-35 customer and issuance prototype v5 verification
description: Records the seven-screen bilingual extension, independent review corrections and browser/PDF validation boundaries.
status: draft
---

# Scope and owner decisions

HB-35; owner Seung Woo Park; machine mac_mini; root writes and integrates, independent workflow reviewer reviews design and actual artifacts. The owner requested customer/contract detail from the prior page 18, MSP/month/company lists including nonissued targets, single-company operation, company lookup, optional actor columns, independently optional/required assignments and actual signed-in actor fallback when unassigned. ERP is unspecified; Odoo or document standards are candidates, not a selected integration.

Seven new operator screens precede the retained 35-state bilingual flow, producing 42 pages per locale. Existing page 18 is now page 25. [Prototype guide](prototype-v5/README.md) defines reproduction and boundaries. Licensing modes are a separate HB-37 requirement, not added runtime.

The owner authorized main integration and subsequently requested public upload after being told the current repository is public and cannot restrict just a docs folder. No visibility change, history purge or blanket staging of local companion files is included.

# Independent review

- PASS: design review covered stable billing-target identity before invoice creation, designated versus actual actor, mandatory review despite optional assignment, draft invalidation and fixed historical snapshots.
- The first actual source review found three blockers: generic navigation hardcoded another company/MSP, copied customer HTML received operator links, and Northstar July/August invoice IDs differed from v4 originals.
- Fixed and independently re-reviewed PASS: context-derived navigation, customer files excluded before link injection, and original invoice IDs preserved.
- PASS: independent inspection of all seven Korean desktop screenshots found no clipping or overlap. ERP destination/purpose/mapping and unverified connector boundaries were accepted. Minor English labels in Korean policy/history were subsequently localized and rerendered.

# Executed verification

Completed 2026-10-28:

- PASS: EN and KO Chromium runs, seven initial desktop states each, four mobile states each at width 390, and successful-issued-history screenshot. Zero page errors and zero HTTP(S) requests in both runs.
- PASS: MSP, company, month, issuance status and assigned/actual actor filters; three-month grouping; unissued navigation; single-company view; optional actor columns; same-named companies under distinct MSPs keep identity through customer/contract navigation.
- PASS: required missing reviewer/issuer blocks; optional unassigned successful actions snapshot actual user; assigned-user mismatch blocks; failed actions do not create completed actors; evidence and policy changes invalidate draft review; issue requires current successful review.
- PASS: duplicate issue leaves the record byte-identical; user and later policy changes do not mutate an issued snapshot; historical events remain; renewal draft saves without activation; ERP preparation identifies missing connector/destination/mapping and never submits.
- PASS: generated customer pages exclude new operator navigation; prior operator close pages have customer-detail entry links. These are prototype boundaries, not production authorization tests.
- PASS: seven one-page PDFs plus the retained 35-page flow assembled into each 42-page book. Poppler page-count check catches spill pages. Initial register spill was corrected by limiting responsive breakpoints to screen media. Final new PDF pages and updated close page visually inspected in both languages.
- PASS: source file lengths below 1,500 lines, scoped diff whitespace and document frontmatter/relative links checked before commit. Generated assets remain under output and are not staged.

# Evidence and limitations

Generated evidence: `output/designs/HB-35-v5[-ko]/render-check.json`, desktop/mobile PNGs, individual page PDFs and final-page PNGs; final books in `output/pdf/hybrid-billing-msp-design-v5[-ko].pdf`. Prior v4 finance/browser verification remains recorded in [v4 report](v4-verification.md); this extension does not recertify all product screens or implement production billing.

No backend, authentication, persistent audit store, customer CRUD, actual contract activation, invoice delivery/PDF issuing service, AI parsing or ERP submission is connected. New issued records exist only in browser memory and reset on reload. Same-person review/issue is allowed for demonstration; production segregation policy is not selected. Third-party ERP API/version/hosting and exact accounting mappings remain unverified.

Primary ERP references checked 2026-10-28: [Odoo 19 external API](https://www.odoo.com/documentation/19.0/developer/reference/external_api.html) and [Peppol post-award specifications](https://peppol.org/documentation/technical-documentation/post-award-documentation/). Invoice standards do not certify arbitrary ledger recognition support.
