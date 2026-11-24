---
type: Record
title: Implementation record of the lazy O01 organization tree
description: What each step of HYB-320 built, how it was checked, what the reviews found and what is left.
status: draft
---

# Step 1, the server read (2026-11-25)

Built against [the design](design.md), rules R1 to R9.

- `modules/relationships`: `MembersOnAfter` walks the members one target ever had, in identifier order from a cursor, and returns those covering the day, the last identifier examined, the number examined and whether the walk reached the end. It stops at a limit of members or at a budget of candidates.
- `apps/api/server/orgtree.go`: `GET /api/v1/organizations/tree` with `on`, `revision`, `parent`, `group=detached`, `after` and `limit`. Any other parameter, a repeated parameter, `parent` with `group`, a malformed cursor and a cursor naming a party that does not exist are `invalid_request`. Admission, the `provider` refusal and the failure codes are those of the organizations read.
- The cursor is `p:<identifier>` while parties are being returned and `m:<identifier>` afterwards; `m:` alone starts the second phase.
- The budget (`org_filter_scan`) is counted in candidates examined: members of the parent walked, hierarchy targets tested as unknown roots, and members examined to decide `children`. The party part is bounded by the party cap.

Differences from the design, for the change review:

- The public path is `/api/v1/organizations/tree`, beside the existing `/api/v1/organizations` reads. The design asked for an OpenAPI entry; the existing organization reads have none, so none was added here and HYB-357 covers all of them.
- A page reports `scan_stopped` when the walk for further nodes was ended by the budget, which can also happen on a page that is full; `next` is then always set. When the budget runs out while `children` is being decided, the remaining nodes answer `unchecked` and the page is otherwise complete.
- R2(b), progress: when a hierarchy target that is not a party and has no timeline cannot be decided with a whole untouched budget (it had more former members than the budget, none found covering the day so far), it is returned as a root with `children: unchecked` so that the next page can move past it. Such a root may turn out to have nothing under it on the day. The design now states this exception in R2.

Checks run on the development machine (macOS arm64):

- `go vet` clean for the module and `apps/api`; the module tests and the whole `apps/api/server` test package pass (results below).
- `TestTreeReadOfTheSample`: the sample data through HTTP, including revision 0 (registry parents only), a day before any relationship, cut pages in both phases, 19 refused queries, a member (403), no session (401), a fault (503).
- `TestTreePagesMatchTheTreeBuiltFromEveryMember`: for generated revisions on four days, every list (roots, the children of every node and of an unknown identifier, the detached group) read in pages of one to four equals a Go port of the tree the console builds today from every member; kinds, states and `children` are compared; every node the present tree lists under "cycles" is found under a detached row and nothing else is. The test requires at least 100 accepted revisions, 20 days with a cycle and 10 with nodes hanging below one.
- `TestTreeCycleThroughSeveralLegalEntities`: a cycle through six legal entities, each stretch 31 hierarchy steps, longer in total than the depth limit; all six are listed.
- `TestTreeStaysWithinItsScanBudget`: budgets of 1, 2, 4, 5 and 13 give the same lists as the whole budget, in more pages, with `scan_stopped` and `unchecked`, except for one target with twenty former members and none on the day, which every small budget returns as an undecided root and the whole budget does not.
- `TestTreeAcrossWorkspacesAndFailures`: another workspace answers nothing for the sample's identifiers; the same identifier registered there is unrelated to the sample's relationships.
- `TestMembersOnAfterWalksOneTargetInOrder` in the module.
- Each rule removed in turn (14 changes): 12 made a test fail. One survivor was a condition that could not differ (a held party has no parent) and was deleted. The other changes only how much work the detached walk does (nodes marked done are not walked again); no test measures that, and the walk's hard bound was not reached without it.

Not checked: any measurement with a workspace near the party cap or a target with a long history; other operating systems.

Independent change review (AI, Claude Opus 5.5, by reading; it ran nothing), 2026-11-25: approve with conditions, no blocking finding and no path to another workspace found. Four conditions, all done: the missing OpenAPI entry recorded as HYB-357; the design amended for the undecided root; a budget case added that generates such a root; the paging invariant of the test helper corrected to what the read does (a full page can also be marked stopped) with a budget that reaches that case. Observations left open: a party whose registry parent is not a party is shown by the present console under an unknown root and by this read nowhere, which cannot arise unless a party can be deleted (not verified by anyone); when a page is exactly filled by parties, the look-ahead for further nodes can spend the budget and leave every `children` value of that page unchecked. Not verified by the reviewer: that a depth violation blocks activation.
