---
type: Reference
title: HYB-319 external references
description: External professional references for O01 view group drafts, page multi-selection and the review-before-commit handoff, opened by root on 2026-11-18, with what was observed, adopted and rejected.
status: final
tags: [HYB-319, O01, references]
sources:
  - resource: visual-spec.md
  - resource: ../HYB-299/references.md
  - resource: ../HYB-300/references.md
---

# HYB-319 external references (opened by root, 2026-11-18)

## 1. Microsoft Learn, How to manage groups (Microsoft Entra)

https://learn.microsoft.com/en-us/entra/fundamentals/how-to-manage-groups (opened; page dated 2026-06-17)

**Observed:**
- Groups exist to give every member the same access to resources; the page states that a group "applies the special permissions to every member".
- Members are added by "+ Add", scrolling or searching, choosing several names, then "Select"; the overview then shows the new member count. Membership takes effect on Select.
- Members are removed by checking boxes in the member list and choosing "Remove". Bulk add has a separate import path.
- Nested groups do not gain the parent group's access to shared resources.

**Adopted:**
- Search → add several → visible selected list → one commit step.
- Removal by explicit per-row action.

**Rejected:**
- Immediate membership on save: a view group draft changes nothing; membership changes only through a held O02 change plan.
- Groups as an access mechanism: the O01 note states that a view group grants neither ownership nor access.
- A file-import path for bulk add.

## 2. Carbon Design System, Data table usage (batch actions and selection)

https://carbondesignsystem.com/components/data-table/usage/ (opened for HYB-299 on 2026-11-18; observations in `../HYB-299/references.md` §4)

**Adopted:** a row checkbox column, a header checkbox with checked, unchecked and indeterminate states, and a count with the batch action shown while rows are selected.

**Rejected:** the coloured batch toolbar over the header; the bar stays in O01's list-header end slot with neutral styling.

## 3. AWS Cloudscape, Table view

https://cloudscape.design/patterns/resource-management/view/table-view/ (opened for HYB-300; observations in `../HYB-300/references.md` §3)

**Adopted:** progressive loading with selection bounded to loaded rows and a "shown" count; no "select all matching".
