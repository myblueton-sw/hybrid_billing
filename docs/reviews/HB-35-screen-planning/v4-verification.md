---
type: VerificationReport
title: Monthly history and PDF analysis prototype v4 verification
description: Records financial, revision-lifecycle, privacy and rendered-layout checks for the eight added bilingual design states.
status: draft
---

# Scope

[HB-35](https://linear.app/hybrid-billing/issue/HB-35/review-planning-baseline-for-the-complete-product-screen-mockup), 2026-10-28. Root owns all v4 writing and integration. Read-only independent reviewers are `csp_financial_research` for financial design/artifacts and `msp_workflow_research` for workflow/source/visual QA. Accountable owner Seung Woo Park; machine mac_mini.

The owner requested monthly history and supplier PDF parsing/difference analysis in the mockup. Eight new states and a synthetic source PDF extend the prior 27-state prototype to a 35-page EN/KO review bundle. This task does not implement a real OCR/model/parser, upload service or financial engine. The separate license-policy change belongs to HB-36.

# Independent financial review

Design review required a distinct Cedar document identity because the new three-line PDF differs from preserved Northstar supplier detail. It also required fixed historical inputs, original/adjusted basis, separate credit document month, no zero substitution for missing detail, and immutable extraction provenance.

Fourteen independent Decimal controls passed: monthly margins, original/adjusted percentage changes, PDF extraction discrepancy, collection gap and allocation. Generated EN/KO customer data each contain four monthly records and exclude source cost. Source artifact review found and corrected two continuity issues: historical source aliases were replaced with original s3/m2/cp6 references; acceptance lifecycle now visibly retains prior snapshots and updates current E2/E3 labels. Final scoped source re-review PASS, no remaining financial blocker. Browser execution is root evidence, not attributed to this reviewer.

# Root browser verification

Both EN/KO runs completed with eight new screens and combined 35-page PDFs, zero browser exceptions and zero HTTP(S) requests. Actual checks:

- Original/adjusted monthly totals, October-only filter, August's own document and absence of September-only links; historical edit protection; adjusted comparison basis.
- Metadata-only selection using an in-memory synthetic PDF-labelled fixture; duplicate, encrypted, unsupported and OCR-failure sample branches; the sample never analyzes the selected file.
- Acceptance blocked before checks; incorrect support extraction blocked; page confirmation and reason required for E2; extraction arithmetic passing does not bypass source reconciliation.
- Wrong-period detail blocked; corrected C2 reconciles; editing a verified field invalidates checks; acceptance blocked until renewed verification/reconciliation.
- E2 acceptance prevents duplicate acceptance and makes inputs read-only. E3 starts a new draft, updates scope immediately, blocks premature acceptance and preserves E2 history. After E3 recheck/acceptance, both accepted snapshots remain visible.
- Customer September export and October unavailability; three mobile states per language without page-width overflow.

Generated evidence is under `output/designs/HB-35-v4/` and `output/designs/HB-35-v4-ko/`: eight initial state images, verified extraction and multi-revision audit images, three mobile captures, source PDF/PNG, sample monthly CSV and render-check JSON. Print assembly verifies that new-page content stays above its footer and retains the prior reference pages.

# Handoff limits

Independent workflow/visual review inspected all eight new Korean desktop states, three mobile views, two result states and English extraction/audit views. It found stale current-verification labels after revision changes; synchronized current scopes, field/result states and audit history fixed the finding, with scoped re-review PASS. The final EN/KO browser reruns passed new E3 state assertions. Independent link inspection covered 348 links per language, with the existing usage-question navigation alias explicitly identified. Root inspected both-language PDF contact sheets and dense PDF/source-preview pages; Poppler confirmed 35 pages per document. Maintained source length, document frontmatter, relative links and staged scope are checked before commit. No claim of real invoice authenticity, extraction accuracy across providers, model confidence calibration, legal invoice correctness, complete 49-screen coverage or production authorization is made. Arbitrary historical periods and real uploaded documents are not simulated as successfully analyzed. Financial/runtime and actual-user tests remain unverified.
