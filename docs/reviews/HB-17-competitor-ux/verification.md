---
type: Research Verification
text_language: en
title: Competitor UX research scope and verification
description: Records the research ownership source limitations review results and remaining screen validation for HB-17.
status: draft
---

# Research verification

Tracking: [HB-17](https://linear.app/hybrid-billing/issue/ad8c4443-9c7f-479c-ab80-5c49a45e7066). Date: 2026-10-06. Owner: Seung Woo Park; active research machine `mac_mini`. Root applies PM/UI-UX/technical-writing skills and owns both report files. A separate read-only QA reviewer checks evidence and scope. No product code, canonical contract, financial policy or permission grant is changed.

## Boundary and baseline

Branch `docs/HB-17-competitor-ux` uses isolated worktree `/private/tmp/hb17-research`, created from fetched `origin/main` at `76102ddd2653e88c383fe8dcc6b9d6c19da71e90`. Authorized local v1.16 planning and HB-16 head `c161dc5a2b4b2a1321a36c2db673c074b0dd6bac` were read as planning evidence. HB-15 PR #11 and HB-16 PR #12 are separate pending work; neither is merged by this ticket. Local companion rules and unpublished inputs were available in the authorized root workspace; this isolated checkout is not represented as a standalone planning baseline.

## Actual work

- Searched official product documentation and articles for MSP billing, FinOps, multi-workspace operations and invoice workflows. Nine references are analyzed in [comparison](comparison.md).
- Read official pages and distinguished documentation from marketing and search-only evidence. IBM direct fetch failed; the indexed results are labeled S. No vendor support or sales messages were sent.
- Visually inspected Finout's published MegaBill example and Cloudforet's published Data Sources table in Chrome. These are documentation screenshots, not live product sessions. Lago image retrieval failed and is not counted as visual evidence.
- Read the original screen catalog, preserving 49 screen identities and the full provider concept in the proposal. This is not a full semantic audit of every detailed screen.
- Detected source inconsistencies: Stripe paid-state statements and Finout updated cap wording. Neither uncertain value/transition is adopted as a product rule.
- Root research proposals are explicitly marked P. JEV recommendation was UI/UX at 0.59 and standard depth at 0.47; it neither selected product behavior nor authorized execution.

## Verification status

Independent evidence/UX review PASS after two clarifications: S03 inspection hands off to S06 confirmation/rejection, and only static non-executing design artifacts are currently permitted. A01 remains in scope with optional user entry choice. The reviewer independently checked Flexera, Finout, Lago and AWS official sources; other references and root screenshot observations were not independently verified.

Root document QA PASS: both files are English with non-empty OKF type, local links resolve, and credential-pattern checks found no matches. Staged scope and whitespace are inspected before commit. Runtime, vendor sandbox execution, user task testing, accessibility, mobile layout, financial accuracy and connector compatibility are NOT_RUN. No competitive superiority or complete market/provider coverage is certified.

## Handoff

Prepare the scoped research PR after review and document checks. Merge/read-back/ticket completion remain separate owner-authorized steps. Next design work should apply accepted research to connected storyboards and failure states, retaining all original requirements; the research itself does not authorize development or automatic financial actions.
