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

# Still open on this ticket

Unchanged, from the ticket:

- P05 decisions: the counts in "Waiting for approval (n)" and "Closed (n)" cover only the newest 50 entries.
- P05: the pending-retirement link to a proposal outside the newest 50 (suspected).
- P01: the preview basis and the propose offer decided from the 50 newest proposals (suspected).
- P02: deliveries are the first 50 per subscription with no total, so dead letters can be absent from the screen (suspected).

Found by the change review in the same two files and not changed here:

- the build comparison's "Fewer than two builds are recorded", its build options and its note about the newest 10 builds are also derived from the capped requirement rows;
- on the matrix, the count of blocked paths and the intake note are computed from the capped paths, while the limit line for paths appears only under the table;
- the path detail can say that a requirement's test cases are missing when the requirement is simply beyond the first 50.

The decision named in the ticket still stands for all of these: either the server sends what the console needs to say more than "there may be more", which changes public reads and needs independent design and change review, or each such place states the limit.
