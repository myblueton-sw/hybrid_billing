---
type: VerificationRecord
title: Lists capped at 50 rows shown as complete, what was corrected without a contract change
description: Which cases of HYB-352 needed no change to a public read and were corrected, how they were checked, and what remains for a decision.
status: draft
---

# Outcome and ownership

HYB-352, machine:mac_mini, 2026-11-24. Two of the P05 cases are corrected in the console alone, because the lists already carry a total and a "more" flag and the screen already has a line for it. The other cases are unchanged and still need the decision named in the ticket.

# Corrected

| Place | Before | Now |
| --- | --- | --- |
| P05 path detail, test cases of each requirement (`release-evidence/LiveMatrix.tsx`) | the list of at most 50 tests had no limit line, so a requirement with more tests looked complete | the existing line follows each list: "Showing 50 of n." when the total is known, "There may be more." when it is not, nothing when the list is complete |
| P05 requirements, build comparison (`release-evidence/LiveRequirements.tsx`) | "No requirement changed state between these builds", or the difference table, was computed from at most 50 requirement rows and read as covering all | when the requirement list is not known to be complete, a line above the result says: "This comparison covers only the requirements listed on this screen, which may not be all of them." |

Checks: `tsc` clean; console tests 1364, pass 1361, fail 0, skipped 3. No new test and no rendered check: the sample data has fewer than 50 rows in every one of these lists, so neither line appears with it. Independent change review (independent-reviewer role, AI, by reading; nothing executed): pass, no blocking finding; its optional Korean wording was adopted afterwards. The reviewer that first found these cases could not do this review because of a usage limit, so another model did it.

# P05 decisions, corrected in a second step

When the list of proposals may be incomplete (the service says there are more, or does not say):

- the two headings mark their numbers as counts of what is shown: "Waiting for approval (n shown)" and "Closed (n shown)";
- a line under the first heading says "Older proposals waiting for approval may not be listed.";
- an empty list says "No proposals are shown." and not "No proposals are recorded.".

With a complete list the headings and the empty text are unchanged.

Checks: `tsc` clean; console tests 1364, pass 1361, fail 0, skipped 3. No new test and no rendered check, for the same reason as above. Copy and state review (ui-designer role, AI, by reading): fail with one finding (the empty text still claimed the whole list), corrected, and its recommendation for the line about older proposals adopted; re-review pass.

The reviewer's remark stands as part of the decision: marking the count removes the false total, but a person who must act on waiting proposals is only fully served when the service returns every waiting proposal whatever its age. That is a change to a public read.

# Still open on this ticket

Unchanged, from the ticket:

- P05: the pending-retirement link to a proposal outside the newest 50 (suspected).
- P01: the preview basis and the propose offer decided from the 50 newest proposals (suspected).
- P02: deliveries are the first 50 per subscription with no total, so dead letters can be absent from the screen (suspected).

Found by the change review in the same two files and not changed here:

- the build comparison's "Fewer than two builds are recorded", its build options and its note about the newest 10 builds are also derived from the capped requirement rows;
- the path detail's text for a requirement that is not among the loaded rows already says that its test cases "are not in this read", which is accurate; nothing to change there.

The decision named in the ticket still stands for all of these: either the server sends what the console needs to say more than "there may be more", which changes public reads and needs independent design and change review, or each such place states the limit.

# P05 matrix, read again: no change needed (2026-11-24)

The change review listed two matrix sentences as derived from the capped list of paths. Read at main ae40ae1 (`release-evidence/LiveMatrix.tsx`, `MatrixView`), neither states something about the paths that were not loaded:

- The sentence about missing execution evidence is shown when any loaded path is blocked. Its second half, that no support combination can be activated and that activation is not connected in this build, is true of the build whatever the paths are. When no loaded path is blocked the sentence is absent, which claims nothing.
- The intake note says that customer proof intake is not connected, so the count cannot rise in this build. It is shown when every loaded path reports intake as not connected, and it is a statement about the build. An earlier review of this screen found that an active path, a proven customer state and a ready gate cannot occur in this build.

Both would need another look when activation or customer proof intake is connected, because from then on they would describe the paths and not the build. Nothing was executed for this section and it has no independent review.
