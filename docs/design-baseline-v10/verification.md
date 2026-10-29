---
type: VerificationReport
title: HB-44 modular baseline verification
description: Records immutable baseline checks and chapter package continuity without claiming new runtime functionality.
status: stable
---

# Actual results

`python3 docs/design-baseline-v10/build.py` completed successfully:11 existing modules,22 PDFs,89 pages per locale. Both original PDFs match their frozen SHA-256 values before and after generation. Every original page number is assigned exactly once and every module's full extracted text equals the concatenated original page text. First-page Poppler render PNGs match byte-for-byte for all22 module PDFs, preserving the previously reviewed visual content. The guide occupies its own two-page module; planned VMware/on-prem adds no pages.

The local HTML package index links to generated module files. Original embedded PDF links retain their previous targets; they are not represented as repaired cross-module links. Source pages and amounts are unchanged. The ZIP contains the HTML entrypoints,22 PDFs and verification JSON; PNG comparisons remain local QA artifacts.

Independent workflow design review passed the immutable baseline, stable-ID mapping, modular revision and gap/acceptance boundaries. Actual-change independent review by the separate workflow reviewer passed stable IDs against both locale manifests, all checklist references, module/gap status, baseline hashes and recorded22 results. Reviewer did not rerun splitting. Root also inspected the Korean module-first-page contact sheet and verified local index targets, Markdown links/frontmatter and stable-ID coverage. No runtime tests were applicable or performed. The owner baseline decision does not complete the56 existing delivery checks.
