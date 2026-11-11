---
type: DesignEvidence
title: B01 row-identity external patterns
description: Records the first-party design and specification sources consulted for identifying source rows in the B01 tables and what was adopted or rejected.
status: draft
---

# External patterns consulted

Root read the following on 2026-11-12 and passed summaries to the ui-designer role, which did not browse them itself.

| Source | Observed | Adopted | Rejected |
| --- | --- | --- | --- |
| [Cloudscape table view](https://cloudscape.design/patterns/resource-management/view/table-view/) | "Order the columns by importance from left to right"; every cell needs a logical column or row header; checkbox selection applies to rows visible on the page and is cleared by pagination; sticky columns are a user preference | Identity before period and amount; selection bounded to the current page, as today | User preferences, sticky-column settings, header select-all |
| [Carbon data table](https://carbondesignsystem.com/components/data-table/usage/) | The expanded section is for "supplementary information or data that needs additional query time" | Identity is not supplementary, so it leaves the disclosure and becomes visible cell content | Batch action bar, expandable rows for the remaining fields |
| [FOCUS 1.0 SubAccountId](https://focus.finops.org/docs/specification/v1-0/columns/cost-and-usage/sub-account-id/) | "A Sub Account ID is a provider-assigned identifier assigned to a sub account."; null when a charge does not apply to a subaccount | A null subaccount reads Missing and the billing account ID carries the identity | Any statement about why a subaccount is missing |

The sources named in the [HB-151 reference record](../HB-151/references.md) for null handling, charge period bounds and effective cost continue to apply to the reused vocabulary and were not fetched again. No vendor styling, asset or dependency was copied. Full viewport width, neutral separators, square geometry and local IBM Plex follow the repository rules.
