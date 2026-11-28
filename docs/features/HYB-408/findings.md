---
title: HYB-408 success states against the disposable sample service
status: stable
ticket: HYB-408
updated: 2026-11-28
---

# Method

`--pass <commands>` in `tools/other-outcomes-qa.mjs` and `tools/form-states-qa.mjs` lets the named commands through to the disposable local sample service: `hb-api --seed sample` with a throwaway data directory. Any command whose path contains `approve`, `activate`, `decide`, `issue`, `payment` or `invoice` is refused by name and never passed. All other commands stay routed.

Commands passed:
- `view_group.draft`
- `relationship.propose` and `relationship.discard`
- `service.create`
- `issuance_setup.context` and `issuance_setup.configurations`, which are reads sent as commands
- `retention.propose_policy`, `retention.place_hold` and `retention.propose_release`
- `release_evidence.retire_requirement` and `release_evidence.retire_profile`

No financial or approval command reached the service.

# Results

Checks: focus on a visible element, a new statement, and layout. All runs were in English at 1440; the recipes also ran in Arabic at 320.

| Run | Checks |
| --- | --- |
| O01 and M04 recipes, English 1440 and Arabic 320 | 20, none failed |
| O02 with propose and discard passed | 67, none failed |
| P04 and P05 | 137, none failed |
| O02 and O03, first run | 115, 6 failed |

The six failures in the first O02 and O03 run were plan discards, which were still routed with an empty success in that run. They pass in the O02 rerun above.

Statements seen after the real answers:

| Screen | Statement |
| --- | --- |
| P04 | "Policy proposal #8 is waiting for another administrator's approval." |
| P04 | "Hold retention-hold-1 is in force." |
| P05 | "Retirement proposal #20 … is waiting for another administrator's approval." |
| O02 | "Plan proposed. Its relationship changes are locked." |
| O02 | "Plan discarded." |
| O02 | "Plan rejected. See the reasons." (a real server rejection) |
| O03 setup read | "Issuance setup inspection was refused. Ask an administrator to check inspection rights…" (the sample user lacks the inspection right) |
| M04, Arabic run | "Command refused", because the English run had already created the same account name |

O01 and M04 announce success without visible text:
- O01 moves focus to the saved draft's title and announces "Draft saved with N members" through the hidden announcer.
- M04 returns to the account list with focus on its title.

Neither keeps a visible completion sentence on screen. Whether one is wanted is a screen-design question for the owner's review; it is not a defect against the checklist.

# Not measured

These commands stay routed, so the routed answer shows an unknown result. That is expected; they were not passed because they are approvals or operational stops:
- P04 approval, withdrawal and run stop
- P05 approval and withdrawal
- P01 plan proposal
