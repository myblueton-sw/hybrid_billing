---
type: Reference
title: HYB-300 external references
description: External professional references for the O01 filter band, applied-filter chips and server paging, opened by root on 2026-11-18, with what was observed, adopted and rejected.
status: final
tags: [HYB-300, O01, references]
sources:
  - resource: plan.md
  - resource: visual-spec.md
---

# HYB-300 external references (opened by root, 2026-11-18)

## 1. MOJ Design Patterns, Filter a list

https://design-patterns.service.justice.gov.uk/patterns/filter-a-list/ (opened; status on the page: "To be reviewed")

**Observed:**
- The filter panel sits in a left column beside a scrollable table.
- A toggle shows or hides it, and the example starts hidden.
- Selected filters appear as removable tags grouped by category; each tag carries hidden text "Remove this filter".
- A "Clear filters" link sits beside the "Selected filters" heading.
- Filters apply with an "Apply filters" button, not on change.
- Fieldsets with legends group the checkboxes. Filters do not persist across navigation by default.

**Adopted:**
- Removable chips with an accessible "Remove filter …" name.
- "Clear all filters" next to the applied list.
- Collapsed by default below 1280px.
- Fieldset and legend for grouped member filters.

**Rejected:**
- The left-column panel; a full-width band keeps the 576px list column at 1280 usable.
- The Apply button. Selects and the checkbox apply on change; each change is one bounded server read announced in the live region (decision D1).

## 2. AWS Cloudscape, Filter patterns

https://cloudscape.design/patterns/general/filter-patterns/ (opened)

**Observed:**
- Text filter for simple resources when users know the term.
- Collection-select filter when users filter by one or two properties, single value each, combined with "and".
- Property filter for complex resources.
- Filters must operate on the full collection, not only the visible resources.
- The page gives no guidance on counts, clearing, empty states or apply timing.

**Adopted:**
- Text search plus single-value selects combined with "and".
- Filtering on the server over the full permitted set, never over the loaded page.

**Rejected:**
- The property filter with operators; O01 has four fixed filters.

## 3. AWS Cloudscape, Table view

https://cloudscape.design/patterns/resource-management/view/table-view/ (opened)

**Observed:**
- Pagination gives an accurate sense of size and lets users jump.
- Progressive loading suits cases where users need all data in one view; do not use both at once.
- Errors fetching data appear as an alert in the table's empty state.
- The page gives no guidance on counts when the total is unknown.

**Adopted:**
- Progressive loading ("Load more"), because the server cannot state a total under the scan budget and page jumps would need one.
- A first-page error as an alert in the list slot.

**Rejected:**
- Numbered pagination; it implies a known total.

Not opened: IBM Carbon data table (already recorded for HYB-299) and GitHub Primer.
