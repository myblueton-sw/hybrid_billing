---
type: DesignEvidence
title: S04 field-detail external patterns
description: Records the first-party design and specification sources consulted for the S04 source-field and row-detail composition and what was adopted or rejected.
status: draft
---

# External patterns consulted

Root read the following on 2026-11-11 and passed summaries to the ui-designer role, which did not browse them itself.

| Source | Observed | Adopted | Rejected |
| --- | --- | --- | --- |
| [Cloudscape split view](https://cloudscape.design/patterns/resource-management/view/split-view/) | Detail reflects only the current selection; bottom position for tables with more than five important columns; key-value pairs directly on the panel; concise content | Detail only for the selected row, removed on deselection, below the table, without containers | Tabs inside the detail, resizable panel chrome, preferences |
| [Cloudscape details page](https://cloudscape.design/patterns/resource-management/details/details-page/) | Content grouped into meaningful sections of key-value pairs | Sections are the origin classes | Separate details page; the row has no authorized standalone locator |
| [Carbon structured list](https://carbondesignsystem.com/components/structured-list/usage/) | Read-only grouped rows with short sentence-case headers; data table for complex content | Read-only field table and mapping reference with short headers | Using it for the paged row list, which stays a data table |
| [FOCUS 1.0 null handling](https://focus.finops.org/docs/specification/v1-0/attributes/null-handling/) | NULL is the marker for a missing value; empty strings and placeholder zeros are prohibited | Missing and Exact zero are distinct text states | Any claim that the provider conformed |
| [FOCUS 1.0 ChargePeriodEnd](https://focus.finops.org/docs/specification/v1-0/columns/cost-and-usage/charge-period-end/) | "The exclusive end date and time of a charge period." | End labeled exclusive "by FOCUS definition" | Stating that the product enforces the bound |
| [FOCUS 1.0 EffectiveCost](https://focus.finops.org/docs/specification/v1-0/columns/cost-and-usage/effective-cost/) | Amortized cost after discounts, denominated in the billing currency, can differ from BilledCost | Billed and effective costs as separate adjacent rows with the billing currency | Any difference, ratio or savings figure |

The Cloudscape key-value pairs usage page could not be retrieved beyond its introduction; no guidance is attributed to it. No vendor styling, asset or dependency was copied. Full viewport width, neutral separators, square geometry and local IBM Plex follow the repository rules.

# Design lead

The ui-designer role read the plan, the style rules, the current implementation, ten of the twelve baseline renders and the installed frontend-design skill, and returned the composition: origin as the structure in the table band, the reference and the detail; a closed-by-default reference; non-wrapping UTC timestamps; a four-column source table with unit lines; missing values without a currency; a not-carried list placed last. It did not read the English 390px baselines or the glossary. Its final render comparison is recorded in the [verification record](verification.md).
