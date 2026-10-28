---
type: VerificationReport
title: Review-driven bilingual prototype v3 verification
description: Records financial, workflow and visual checks for the twelve added design states and preserved fifteen-state billing story.
status: draft
---

# Scope and result

[HB-35](https://linear.app/hybrid-billing/issue/HB-35/review-planning-baseline-for-the-complete-product-screen-mockup), reviewed 2026-10-28. Root is the sole writer; `csp_financial_research` provides independent financial design/artifact review and `msp_workflow_research` provides independent workflow/source/visual review. Accountable owner Seung Woo Park; machine mac_mini. [Artifact and reproduction](prototype-v3/README.md).

The owner requested incorporating the external sufficiency findings into revised mockups. v3 adds nine operator and three customer states to the preserved fifteen-state story, yielding 27 pages in each language. The static PDF shows evidence, forms, failure histories and expected outcomes; local HTML also exercises staged decisions. This is representative workflow coverage, not all 49 screen specifications, live financial processing, statutory invoice validation or production readiness.

# Independent review

| Review | Actual evidence / result |
| --- | --- |
| Financial design | Required provider-supplied effective allocation rather than platform amortization; annual commitment separated from monthly frequency and charge interval. Incorporated before artifact review. |
| Financial artifact | 13 independent Decimal controls passed across intake, pricing, proration/renewal, commitment allocation, CUD, FX/tax/rounding and correction. 95 corresponding EN/KO monetary/percentage tokens agreed. |
| Financial fix re-review | Restored preserved Northstar original DEMO-2026-09104 and October 5 issuance after an initial continuity defect; made Aster October 1 issuance explicit for October 31 NET30. Scoped re-review PASS, no remaining financial blocker. |
| Workflow design | Required connected case identity, input-edit validation/impact, distinct job/approval states, complete credit path, separate customer payload and honest local downloads/inquiry. |
| Workflow artifact and visual | Inspected all twelve added Korean desktop states, three mobile states, two issued-state screenshots, English issued approval and four PDF samples. Validated 276 local links. No observed overlap/clipping; customer document loads only customer script and disclosed content. |
| Workflow fix re-review | Added issued-mapping replacement guard; pricing selection updates table/margin/outcome; FX identified as separate fixed p5-FX snapshot; translated remaining human-readable Korean prose. Scoped PASS, no remaining blocker. |

The independent reviewers did not execute a real monetary engine or real accounts. Root's browser results are separate evidence, not attributed to them.

# Root verification

Browser rendering uses the existing trusted Playwright/Chromium installation. Initial sandbox launch was blocked; authorized execution succeeded. Initial mobile page overflow was reproduced and fixed by removing the grid child's intrinsic minimum width. Subsequent runs check that page width stays within the mobile viewport.

The final renderer checks both languages:

- Aster: wrong-period rejection → accepted source → invalid/valid mapping → unknown-run read-back → refreshed-input pending scope → queued/running/verified pinned run.
- Duplicate queue prevention; approval before recheck blocked; return reason → attached evidence → recheck → approval → stale-input issuance blocked → recheck/reapproval → separate issuance; issued mapping replacement blocked.
- Price-rule conflict and selected 10%/0% results; missing/valid proration policy; generic versus typed credit mapping; invalid FX and changed-rate approval impact.
- Fixed customer credit verification/approval/issuance; original identity and amount retained; separate customer sample TXT/CSV downloads.
- Empty inquiry rejected; local reference and original question preserved into the customer response; local acknowledgement.
- Screenshots of all 27 states; three mobile views per language; print content checked against footer bounds; unique print anchors; browser exceptions recorded.

Generated evidence lives in `output/designs/HB-35-v3/` and `output/designs/HB-35-v3-ko/`. Both final language runs passed with 27 states, zero browser exceptions and no visible overflow reported. Poppler confirmed 27 text-bearing pages per PDF. Root inspected all six PDF contact sheets and dense Korean pages 6 and 9, and verified regenerated original/correction download references and amounts. A PDF navigation-selection mismatch was corrected in print assembly so each page highlights its own section. Source files remain below the repository size limit; document frontmatter, local links and staged scope are checked before handoff. No external messages or financial actions are executed by these checks.

# Remaining limits

Only the stated synthetic fixture dates, choices and amounts are supported. Reload resets local state. Refreshed-source acquisition is pending without new evidence; it does not invent a successful connector. Customer issued-history is an explicitly later snapshot. Arbitrary rule authoring, complete provider coverage, bulk operations, all 49 screens, real-user testing, accessibility certification, authorization/tenant enforcement, tax-law correctness and runtime financial integrations remain unverified.

The original external assessment is retained as the before-revision evidence; its gaps are addressed through the representative cases mapped in the v3 README, not silently marked as a production acceptance pass. No merge or deployment is included in this task.
