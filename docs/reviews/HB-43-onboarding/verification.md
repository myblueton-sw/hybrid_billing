---
type: VerificationReport
title: HB-43 design and artifact verification
description: Records actual static design reviews, bilingual rendering and PDF inspection with runtime limitations.
status: draft
---

# Results — 2026-10-29

| Check | Result | Evidence |
| --- | --- | --- |
| Independent pre-edit security design | PASS | Separate query_auth_review reviewer required and rechecked seven gates: invitation/bootstrap consumption, delegated roles/SoD, recovery/trust, diagnostic revision, provider custody, historical transfer and connection/evidence boundaries |
| Independent pre-edit workflow design | PASS | Separate msp_workflow_research reviewer checked entry forks, OP05 sequence, restartability, collection versus billability and full-book preservation |
| Actual access/setup sources and generated EN pages | PASS | Security reviewer inspected all 16 entry/setup states, generator, plan, CSS and checks; zero enabled form controls or scripts in these states |
| Actual requirements/README consistency | PASS | Security reviewer confirmed no runtime claims or deployed customer authorization claim |
| Customer/payment/master expansion | PASS | Workflow reviewer inspected customer_profile.py, requirements and both full-size page 21 screenshots; prior fields preserved and payment axes separated |
| Representative new layouts | PASS | Workflow reviewer inspected EN pages 03, 09, 13, 15, 17 and 18; no observed clipping/overlap |
| Bilingual build | PASS | build.py produced 89 pages each; coverage.json maps every 66 v9 IDs plus 23 new IDs |
| Browser/print artifact checks | PASS | render.cjs completed 226 records: 178 single-page renders, 46 mobile width checks, two locale summaries |
| Navigation and customer fixture isolation | PASS | Eleven review chapters, 87 screen-index entries, nine separate customer pages; local HTML targets exist; selected operator fixture strings absent from customer entry |
| Browser requests/errors | PASS | Zero remote HTTP(S) requests and browser page errors during the rendering run |
| Final PDF layout overview | PASS | Initial 82-page PDF inspected in full via seven contact sheets per locale. Intermediate 84-page PDF changes inspected by root and independent reviewer. Final 89-page additions received independent full-size EN layout review; final PDF rendering is checked separately below |
| Runtime auth/tenant/provider/payment enforcement | NOT_RUN | Static artifact only; no authentication, cloud setup, payment or historical-transfer action performed |

Commands: `python3 docs/reviews/HB-43-onboarding/build.py`, `node docs/reviews/HB-43-onboarding/render.cjs` with external runtime paths, and `pdftoppm -scale-to 600 -png` for each final PDF. The renderer also invokes pdfinfo per page and pdfunite. Final evidence resides in `output/designs/HB-43/render-check.json`, locale manifests/render images and PDF-derived contact sheets. Generated files are excluded from Git.

The inherited 54 references without a known local route per locale are explicitly converted to static text, not presented as working links. Financial/configuration controls are disabled review states. Mobile checks establish root-width fit, not a production responsive/accessibility certification. Customer isolation checks concern a static review package, not server enforcement. Contact sheets establish layout overview, not line-by-line text proofreading. No remote CI success, owner design sign-off or real connector certification is implied.

Security and workflow reviewers were separate from root, the sole writer. Billing implementation, concrete authentication policy, finance/ERP profiles and owner design approval remain future work. The source files are below the 1,500-line limit.

The customer internal source-reference finding was corrected by replacing it with a distinct approved customer-scoped instance reference. Security re-review of the actual customer search/open-book pages, requirements and delivery checklist passed; all runtime checklist items remain unchecked.

## Later owner feedback and independent review

The separate checklist now contains 56 unchecked delivery items. [Feedback review](feedback-review.md) maps duplicates to existing obligations and documents official Marketplace evidence. Five cost visibility/Marketplace states were added and the existing allocation screen extended; optional cost API and purchase planning remain peripheral requirements, not implemented core capabilities.

Independent CSP financial research/review passed the actual Marketplace and forecast source, official citations and arithmetic, with GCP detailed exclusions explicitly unverified. Independent security review passed raw/export scope, optional API boundaries and operator-only placement of the five pages. Workflow reviewer inspected EN pages31/32/35/36/61 and allocation52: layouts and financial boundaries passed except a link implying unrelated raw rows were discrepancy evidence. The link was relabelled as a separate XPL-DEMO example; independent source and generated EN/KO re-review returned PASS. No provider eligibility or runtime forecast/export is certified.

Final artifact QA: the 89-page-per-language rerender completed all226 records with no failure. Root inspected PDF-derived KO pages31/32/35/36/52/61 after the continuity fix, supplementing the prior complete-book and independent EN reviews. No observed clipping/overlap. Staged scope, document links/frontmatter, source-size and sensitive-pattern checks passed; generated files and unrelated workspace changes remain excluded.
