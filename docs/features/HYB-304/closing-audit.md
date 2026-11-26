---
type: Review
title: HYB-304 closing audit against the plan and the planning list rules
description: What is met, what is not, and which ticket carries each open item, for server search, filters and paging on M02 to M05, read on 2026-11-27.
status: stable
---

# Scope

Read-only audit of [HYB-304](https://linear.app/myblueton/issue/HYB-304) before closing, on main `a1380542`, by a role-based AI reviewer separate from the writers of the M02 and M03 increments. It read the records and the code; it ran no test, build or browser. It did not read the bodies of the server cursor and parser code, the evidence manifests, or the local screen specifications. Root recorded this report and opened the tickets it names.

It compares the code with the [plan](plan.md) (the table of required query behaviour, the common contract items 1 to 8, increments A to D) and with the list rules of `docs/planning/v1.16/ui-design.md:412-421` and UI-04 (`:547`).

# Result

**HYB-304 does not yet meet every list rule of the planning documents.** Increments A to D are present: on all four screens the entry read carries no population and the screen reads bounded lists. Three rules are not met on some screens and are carried by tickets: applied filter chips (HYB-378), page size selection (HYB-376), and query state in the URL (HYB-379). The ticket stays open until those three are built or the owner decides otherwise; the owner's decision of 2026-11-27 covers only the five M02 areas without a producer (HYB-374).

# Per screen

| Rule | M05 | M04 | M03 | M02 |
| --- | --- | --- | --- | --- |
| Entry read carries no population; the screen reads bounded lists | met | met | met | met |
| Filters the plan names | met | met | met | met for the trust-change list; five areas HYB-374 |
| Server filter, order and permission projection (`ui-design.md:412`) | met | met | met | met |
| Stable order with a tie-breaker (`:416`) | stated | stated | stated | stated; the browser sorts an open group that is whole on one page |
| Applied filters as chips, cleared at once (`:417`) | met | not met | partly (a count and Reset) | not met |
| Reference time (`:418`) | met | met | met | met |
| Next and previous (`:419`) | met | met | met | met |
| Page size selection (`:419`) | met | met | not met | not met |
| Expired-cursor guidance (`:419`) | met | stated | partly (a refused query has the same wording) | met by reading, not tested |
| Page total against selection total (`:420`), fixed manifest for filter-wide selection (`:421`) | met | no bulk action | no bulk action | no bulk action |
| Query state in the URL (plan item 6) | session list yes; operations and items no | yes | permissions yes; requests, revocations, picker no | no |
| Installation defaults shown with their source (plan item 5) | met | partly | sends fixed limits | sends fixed limits |
| No whole population in browser memory (`:412`, `:547`) | met | met | met | met |

"Stated" means a record says so and the audit did not read the ordering code. UI-04 at its stated scale of 100,000 rows was not exercised by any record.

# Open items and the ticket that carries each

| Item | Ticket |
| --- | --- |
| Five M02 areas without a producer | HYB-374 |
| Applied filter chips on M04, M03, M02 | HYB-378 |
| Fixed page sizes against the installation's page limit; page size selection on M03 and M02 | HYB-376 |
| Query state in the URL for M02 lists, M03 requests, revocations and picker, M05 operations | HYB-379 |
| Whole-population server work: the access preview's roster, grant and delegation lookups, the history read's session copy, kept revocations per permissions page | HYB-380 |
| The policy read's holders summary | HYB-377 |
| Closed and expired records counting toward lookup limits | HYB-370 |
| M04 checks not rerun after integration; rotation under hold and step-up; the expiry index test; the rotate draft | HYB-381 |
| M02: a waiting activation can be pruned; two refusals differ for another workspace's session; no API description of the policy read; the accessibility runner not run; untested cursor cases on M02 and M03 | HYB-382 |
| The trust ledger discards the error of restoring earlier changes | HYB-375 |
| Server tests still asserting on rows built outside the reads | HYB-373 |
| An intermittent person-picker check failure | HYB-372 |
| Visual and wording polish on M03 and M02 | HYB-371 |

Not ticketed, by decision: the M03 filter-wide selection manifest does not apply while no bulk action exists.

# Record hygiene found

- Most HYB-304 records are `status: draft`, and several still say the work is unfinished where later records supersede them: `query-server-progress.md`, `selection-progress.md`, `session-ui-integration.md` (the `/me` read), `m04-completion.md` and `m04-coverage-residuals.md` (the 100 percent server coverage gate, accepted by the owner on 2026-11-26 as recorded only in `main-integration.md`), `m05-boundaries.md` (the same gate), `m03-query-design.md` (an owner question that was decided), and `plan.md`'s last sentence.
- These are corrected when the ticket closes, in the finalization change the working agreement requires.
- `docs/ui/screen-binding-fields/identity.md` outside its M02 and M03 sections still carries stale line references (the M01 rows citing the command sender, the route range in the preamble, three M03 references); they are noted on HYB-371.

# Never verified for any increment

Owner screen acceptance; a screen reader or other assistive technology; native-language review of Korean and Arabic; consumers outside the repository; UI-04 at its stated scale.
