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

# Step 2, the console (2026-11-25)

- `service/admin.ts` `tree()` and the strict parser `organizationTreeOf`: an answer with an unknown kind, source or `children` value, or a missing field, is refused as invalid.
- `organizations/tree.tsx`: each list (the roots, what is under one node, the legal entities on a cycle, the members without an organization) reads its own pages and has its own loading line, failure with "Try again", empty sentence, "Load more", stopped-search line and failed-next-page message. A node marked `none`, or already on the path above it, has no toggle; the second case says so. Closing a node drops what was loaded under it; opening it reads again.
- `organizations/Live.tsx`: the hierarchy cache, the ten-page follow and the client tree builder are removed (`live-model.ts` `tree`, `loaded`). The heading shows the revision and the day. The revision is pinned by a first read of the existing organizations read with a page size of one, which also supplies the workspace facts and the party list used by the detail. The selected node's detail uses the exact member read that the accounts view uses.
- "Expand all" is removed; "Collapse all" stays. A single root is opened at the start; several roots start closed.
- The paging hook moved to `organizations/use-pages.ts` unchanged.

Differences from the design: the header reads `parties` from the existing read, so `registry_empty` is not used yet. Members without an organization are shown as openable (`unchecked`), because the read that lists them does not say whether anything is under them; opening one that has nothing says so.

Checks on the development machine:

- `tsc` clean. Console tests: 1378, pass 1375, fail 0, skipped 3 (three tests of the removed client tree deleted, eleven added in `tree.test.tsx`: the client call and ten refused answers, every state of a list in three languages, the path rule, the plan rule, the new sentences).
- Rendered check against a real service with the sample data (`tools/organizations-qa.mjs`, headless Chromium, English, Korean and Arabic at 1440, 1024 and 390): 278 passed, 0 failed. For the tree it checks the closed roots in order, a root without a toggle, that only the tree read, the page-size-one read, the unattributed read and exact member reads are sent, opening a root and a unit, an account without a toggle, an injected failure under one node with its retry, "Collapse all", the detail and plan link of a selected unit, and a day before any relationship (nine members without an organization, one opened and found empty). Not an owner acceptance.

Not covered by the rendered check: a cycle (the sample has none), a further page, a stopped search, a node that repeats on its path; these are covered by the static render tests only.

Reviews of step 2 (AI, Claude Opus 5.5, by reading; neither ran anything), 2026-11-25:

- Independent change review: approve with conditions, then approve after a bounded re-review. Three blocking findings corrected: the tree could stay blank when another view had already set the screen to ready without the base facts (the first read now starts whenever the base is missing and no error is shown); the "nothing recorded" sentences could appear above a non-empty cycle list or no-organization list (they now need all three lists empty, and the roots otherwise say that nothing is placed at the top of the tree); a binding row described removed behaviour. Also taken: the detail shows the row's name and kind while its facts are read; unused imports removed. Left open: when the date changes, each open list sends one read that is discarded; a read without a pinned revision that races an activation is reported by the existing inconsistency notice, as in the accounts view. No test covers the two corrected effects in `Live.tsx`; the rendered check does not reach them.
- ui-designer review: FAIL with three blocking findings, corrected, re-review PASS. The cycle heading and note were inaccurate for a cycle through more than two legal entities; the stopped-search sentence was unnatural and misleading with no rows; a node found empty was not announced. Its strings were adopted, with its suggestions for six further strings, names isolated inside sentences, and "Collapse all" moved into the heading row. Not judged from a rendering by the reviewer: loading under a node, a further page, a stopped search, a repeated node, the cycle list, the single open root, the new roots sentence. Left for the owner: sibling order by identifier; toggles on accounts in the no-organization list; that list's order differs from the tree's; errors show no code or request identifier (as elsewhere on this screen).

Final checks after the corrections: `tsc` clean; console tests 1378, pass 1375, fail 0, skipped 3; rendered check 278 passed, 0 failed. The Arabic wording of one sentence was changed after that run to the reviewer's suggestion; the tests were run again, the rendered check was not.
