---
type: Verification
title: HB-61 interface research verification
description: Evidence checks and remaining limits for the research-only billing interface report.
status: draft
---

# Research verification

Date: 2026-10-31. Ticket: [HB-61](https://linear.app/hybrid-billing/issue/HB-61/import-owner-supplied-design-system-r1-package). Branch: `feat/HB-61-design-system`. Machine: `mac_mini`. Report: [market-research.md](market-research.md).

## Scope and ownership

Root owns research synthesis and both files in this directory. The independent `research_review` agent has read-only evidence-review responsibility. This phase covers desk research only; design adoption and runtime import remain pending under the owner's research-first correction.

## Executed checks

| Check | Result | Evidence and limits |
| --- | --- | --- |
| Ticket binding and ownership | PASS | Correct Hybrid Billing team/project and a single mac_mini claim were verified through the workspace-specific API before work. The unrelated connected app workspace was not used for mutations. |
| Screen inventory coverage | PASS | Python compared registry IDs with the report's inventory section: 49 expected IDs, no missing or extra IDs. This checks mapping, not screen behavior. |
| Local style evidence | PASS | Read source tokens and responsive/font CSS; matched reported palette, type sizes, dimensions, font families, and breakpoints. No component build or browser runtime test was run. |
| Primary-source research | PASS with limits | Official sources are linked at claims. Kill Bill's published homepage and account-menu screenshots were visually inspected in Chrome. ERPNext, billing workflow, shell, regional-design, and W3C claims were inspected in official written material. |
| Evidence classification | PASS, root check | Report distinguishes observed screenshots, documented guidance, local code, and unknowns; it does not rank regional preferences or adopt a shell/style. |
| OKF frontmatter | PASS, root check | Both documents contain nonempty type, title, description, and draft status. |
| Independent evidence review | PASS with limits | Separate read-only research_review agent verified all 49 IDs, local CSS, and sampled Carbon, ERPNext, Lago analytics, and UAE sources. No blocking findings. Its minor wording suggestions were applied: qualify larger-screen gutters and avoid imposing a new user-study gate. Reviewer did not independently reinspect Kaui pixels, all links, or product runtime. |

## Verification limits

No live competitor account, payment, customer data, usability participant, or device-usage dataset was used. No product screenshot proves the current interface in every edition. Saudi DGA design-system details were inaccessible; Zuora and Odoo retrieval also did not support a reliable current-layout comparison and were excluded from the comparison table. The source package, screen registry, and old foundation are local workspace inputs not included in this two-file change; a fresh clone may lack them.

No accessibility conformance, translation quality, RTL interaction, responsive runtime, financial formatting, permissions, or full-screen functional QA is certified. No implementation tests were required for this document-only change. The 49-screen map is research coverage, not 49 completed visual reviews.

## Handoff boundary

This report is a reviewable research artifact. No merge, style adoption, menu replacement, or runtime import is claimed. HB-61's broader import scope remains open. Research publication and remaining tracking status are reported in the PR and ticket handoff.

## Terminology extension, 2026-10-31

The owner subsequently requested externally researched regional IT terminology rules for English, Arabic and Korean. Root owns the new [terminology guide](../../ui/design-system/terminology.md), the AGENTS.md entry link, and updates to these research records. This extension establishes a content workflow; the original visual research remains unchanged in meaning. No runtime translation, screen redesign or financial/security contract change is included.

Primary evidence covers Google global writing, UAE English/Arabic content guidance, AWS English/Korean/Arabic terminology, Microsoft Korean/Arabic cloud vocabulary, NAVER Cloud Korean administration terminology and FinOps definitions. A localized official page is treated as usage evidence, not proof of human translation or native-market acceptance. The initial ten concept rows are candidates with explicit unresolved labels; no native-language/domain approval is claimed.

Root checks completed: nonempty OKF type fields, local document links, and source-ID references. Unsupported discovery leads were excluded from term evidence: the Korean Microsoft PDF was not readable through the research tool, KRDS retrieval failed, and a NAVER notice's Korean search excerpt was not exposed on direct retrieval. These failures are not recorded as successful source review.

Independent documentation review by the separate read-only research_review agent: PASS; sampled primary-source usage and final four-file staged diff checked. No blocking findings. Root inspected the staged diff for scope and sensitive data; staged whitespace checks passed. Native practitioner review, regional term acceptance and rendered multilingual QA are UNVERIFIED; those limits do not prevent recording the requested source-checking rules.
