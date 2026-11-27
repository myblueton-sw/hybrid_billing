---
type: AuditRecord
title: HYB-305 closing audit — server search, filters and paging on O02, O03 and P01 to P05
description: What the seven screens meet of the planning list rules at closing, which ticket carries each open item, and the owner's decisions of 2026-11-27.
status: stable
---

# Result

Every increment of the [plan](plan.md) is built (1a to 5e) and its routes are registered (`apps/api/server/boundary.go:66-104`). The paged reads end the silent cuts and give each growing list a further read with a count. Several rules the plan took on are not reached on every screen: the search box on most screens, the filters named in the specifications, applied-filter chips, three lists still returned whole, and two partial counts. On 2026-11-27 the owner chose to close HYB-305 with these carried by follow-up tickets, as HYB-304 was closed.

Audit: independent reviewer (AI, by reading main `984ae039`; nothing executed), 2026-11-27.

# Per screen

M met, N not met, NA not applicable, S stated that rows can move between pages.

| Rule | O02 | O03 | P01 | P02 | P03 | P04 | P05 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| No whole population in the browser | M (parties and profiles read whole up to 500) | N (at most 200, filtered in the browser) | M | partly (several small lists whole) | partly (lost events whole) | partly (a hold's scope whole) | M |
| Server search used by the screen | M closed plans; N members | N | N (server accepts `q`) | N | N | N (owner picker M) | M requirements; N others |
| Server filters from the specification | N | N | NA | N | N | N | partly |
| Cursor paging or further reads | M | NA | M | M | M | M | M |
| Count, exact or lower bound | M | M | M | M | M | partly (hold outcomes) | M (blocked-path sentence partial) |
| Stable order with a tie-breaker | M | M | M | M; S deliveries | M; S recovery rows | M | M; S results |
| Applied-filter chips with clear-all | N | N | N | N | N | N | N |
| Page size choice | N | N | N | N | N | N | N |
| Query state in the URL | N | M | N | N | N | N | N |

# Open items and the ticket that carries each

| Item | Ticket |
| --- | --- |
| Search box on P01 to P05 and O02 plan members | HYB-389 |
| Filters from the specifications | HYB-390 |
| Applied-filter chips with clear-all | HYB-391 |
| Page size choice within the installation's limit | HYB-392 |
| Query state in the URL | HYB-393 |
| Lists still returned whole | HYB-394 |
| Counts stated as partial | HYB-395 |
| Fixed snapshots, or acceptance of rows that can move | HYB-396 |
| Installation-wide deletion sequence numbers visible to a workspace (owner decision: hide them) | HYB-397 |

# Owner decisions, 2026-11-27

- Close HYB-305 with the follow-up tickets above.
- P04 deletion records must not expose installation-wide sequence numbers to a workspace (HYB-397).
- Still open as owner questions in the plan: the grouping of runs and holds, newest-first order with a 50-row cut on P02 and P03, the O03 200-contract limit, and the order of increments.

# Never verified

UI-04 at its stated scale (100,000 rows); owner screen acceptance; native-language review of Korean and Arabic; assistive technology.
