---
type: Research
title: External references for the console's data list pattern
description: What established enterprise design systems document for data tables, filters, pagination, status, master-detail and list state, with exact sources, gathered for one shared list pattern.
status: stable
---

# Scope

Gathered on 2026-11-27 for [HYB-376](https://linear.app/myblueton/issue/HYB-376), [HYB-378](https://linear.app/myblueton/issue/HYB-378) and [HYB-379](https://linear.app/myblueton/issue/HYB-379), as `.agents/rules/service-screen-style.md` requires before a screen layout is revised. A research agent fetched public official documentation pages; the fetch tool summarises pages, so short quotations are the phrases it returned and should be checked on the live page before they are reused as copy. Pages that could not be fetched are listed and not cited. These are references, not decisions: the pattern decisions, what is adopted and what is rejected, are recorded with the composition.

The project already uses IBM Plex, square geometry and hairline borders ([design system](../../design-system.md)), so IBM Carbon is the closest reference; the others are used where Carbon is silent or where systems disagree.

# Data table

| Source | What it prescribes | Numbers |
| --- | --- | --- |
| IBM Carbon, [Data table usage](https://carbondesignsystem.com/components/data-table/usage/) (updated 2026-09-30) | A toolbar at the top holds primary buttons, search, filtering and display settings; its height matches the row height. Header row matches the row size. Zebra striping optional. Multi-select by checkbox, single by radio; a batch bar replaces the toolbar on selection. Fewer than three row actions stay inline, otherwise an overflow menu. Hover state always on. Pagination always at the bottom. Loading uses skeleton rows, not spinners. Long headers wrap to two lines, then truncate with a tooltip. | Rows 24, 32, 40 (default), 48, 64 px. Up to five toolbar actions. |
| IBM Carbon, [Data table style](https://carbondesignsystem.com/components/data-table/style/) | At least 16 px between columns; header 14 px semibold, body 14 px regular. | 16 px cell padding |
| IBM Carbon, [Data table accessibility](https://carbondesignsystem.com/components/data-table/accessibility/) | Sort headers reachable by Tab, toggled with Space or Enter, state in `aria-sort` and visible. The table has an accessible name. | — |
| IBM Carbon, [Empty states](https://carbondesignsystem.com/patterns/empty-states-pattern/) | The empty state replaces the table, headers included; it says why it is empty and gives one next step, such as adjusting filters. | — |
| Red Hat PatternFly 6.7, [Table design guidelines](https://www.patternfly.org/components/table/design-guidelines) | Toolbar above the table. Default and compact density. Cells align to their content type; numbers use tabular figures. Row actions in a final-column menu. Bulk select covers the current page only. At narrow widths rows stack as attribute and value pairs. | — |
| PatternFly, [Empty state](https://www.patternfly.org/components/empty-state/design-guidelines) | In a table: "No results found", "Adjust your filters and try again." An error state names the problem plainly and how to fix it. | — |
| PatternFly, [Toolbar](https://www.patternfly.org/components/toolbar/design-guidelines) | Bulk selector first, then filters and actions; pagination is the last toolbar item. Count "17 items", or "10 of 17 items" when filtered. | At most two exposed buttons |
| USWDS 3.14, [Table](https://designsystem.digital.gov/components/table/) | Borderless, striped, compact, scrollable (focusable), sticky header and stacked variants. Right-align numbers in monospace, but not phone numbers, codes or dates. | — |
| GOV.UK, [Table](https://design-system.service.gov.uk/components/table/) | Right-align numbers; a caption names the table; headers carry `scope`. | — |
| Atlassian, [Dynamic table](https://atlassian.design/components/dynamic-table/usage) | Never rely on a highlighted row alone to show selection; actions carry row-specific labels. | — |
| Ant Design, [Data list](https://ant.design/docs/spec/data-list) | An empty state for no data and a separate one for no search results. | — |

# Filters

| Source | What it prescribes |
| --- | --- |
| IBM Carbon, [Filtering](https://carbondesignsystem.com/patterns/filtering/) | Several filter categories sit on the left or across the top, never inside a menu. Apply in a batch ("Apply filters") when there are several categories or the data is slow; instantly for a single selection. A closed filter area shows the applied count and a clear option. Per-category clear and clear-all. |
| IBM Carbon, [Search](https://carbondesignsystem.com/patterns/search-pattern/) | Always show the number of results, also when there are none. Submit-based search when searching is costly. |
| PatternFly, [Filters](https://www.patternfly.org/patterns/filters/design-guidelines) | Filters sit left in the toolbar. Applied filters appear as removable labels, grouped by category, with a clear-all control after the last label, worded "Clear filters". Categories combine with AND, values within a category with OR. |
| Shopify Polaris, [Index filters migration](https://shopify.dev/docs/apps/build/app-home/migrate-from-polaris-react/index-filters) | Chips are derived from the active query; each has a label like "Remove status filter"; a "Clear all" control keeps the sort; the count is announced in a status region. |

# Pagination

| Source | What it prescribes | Numbers |
| --- | --- | --- |
| IBM Carbon, [Pagination](https://carbondesignsystem.com/components/pagination/usage/) | Items per page, the range with the total, page of total, previous and next. Directly below the table with no gap, never above. Height matches the row height. Small screens drop the selects. | — |
| PatternFly, [Pagination](https://www.patternfly.org/components/pagination/design-guidelines) | Full, compact and indeterminate variants; the indeterminate one shows "many" instead of a total. On mobile a summary on top and a bottom bar, never two full bars. | — |
| USWDS, [Pagination](https://designsystem.digital.gov/components/pagination/) | A `nav` with a unique label and `aria-current`; an unbounded set shows no last page. Hide Previous on the first page and Next on the last. | Seven slots |
| GOV.UK, [Pagination](https://design-system.service.gov.uk/components/pagination/) | `nav` labelled "Pagination", after the content, hidden when there is one page; the page in the document title. | — |
| Atlassian, [Pagination](https://atlassian.design/components/pagination/usage) | At the bottom; labels must be localised. | — |
| Ant Design 6.7, [Pagination](https://ant.design/components/pagination) | Total as "1-20 of 85 items". | Sizes 10, 20, 50, 100 |
| Shopify, [Index table composition](https://shopify.dev/docs/api/app-home/patterns/compositions/index-table) | Previous and next only (cursor paging); reset the cursor and selection when the query changes. | — |

No fetched page says where focus goes after a page change.

# Status, master-detail and list state

| Source | What it prescribes |
| --- | --- |
| IBM Carbon, [Status indicators](https://carbondesignsystem.com/patterns/status-indicator-pattern/) | Colour alone never conveys status; an icon or shape goes with a text label, left-aligned in tables; more than five or six kinds overwhelm. |
| Atlassian, [Lozenge](https://atlassian.design/components/lozenge/usage) | Lozenges show status and are not interactive; tags describe; sentence case; never colour alone. |
| PatternFly, [Primary-detail](https://www.patternfly.org/patterns/primary-detail/design-guidelines) | Panes are halves, thirds or quarters; the selected item shows a selected state; at narrow widths the detail covers the list with a back link; a closable pane has a close control at its top. |
| Shopify Polaris, [Index filters migration](https://shopify.dev/docs/apps/build/app-home/migrate-from-polaris-react/index-filters) | The address owns the query, filters and sort; changing the query removes the cursor using replace navigation. |
| GOV.UK, [Pagination](https://design-system.service.gov.uk/components/pagination/) | A page address that no longer exists goes back to page 1. |

No fetched page gives guidance on telling users about a system limit near a list, or right-to-left rules for tables and chips beyond localised labels and a mirrored search box.

# Not fetched

SAP Fiori (403), Microsoft Fluent 2 table usage (404), Material Design 3 and Salesforce Lightning (pages render by script; only titles returned), PatternFly table accessibility (404), GOV.UK filter-a-list pattern (404). None of these is cited.
