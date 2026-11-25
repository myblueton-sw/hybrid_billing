---
type: DesignPlan
title: O02 plan detail says what was cut and names a changed member's kind
description: Correction of a sentence on the plan detail of O02 that spoke of members while its flag is about rejection reasons, with the relationship kind shown for changed members of a plan of several kinds and a denied further read handled as the page handles it.
status: draft
---

# Scope and ownership

HYB-365, machine:mac_mini. Found while building HYB-305 increment 5c and confirmed by its independent review. Console only; no server, command, permission or contract change.

# Facts

- The plans reads give each plan `truncated`. The relationships module sets it when the list of violations of a rejected plan was cut at its limit of 100 (`modules/relationships/validate.go`, `plan.go` `reject`). It says nothing about changed members.
- The plan detail (`organizations/pages/changes/Live.tsx` `Detail`) showed, for a truncated plan, "Not all members shown. More members changed than are listed here." in three languages. That sentence was untrue for what the flag means.
- Since HYB-305 increment 5c the changed members have their own count and "load more", and each row carries its relationship kind. The table listed every kind in one list without saying which kind a row belongs to, so a member changed in two kinds appeared twice with nothing telling the rows apart.
- A denied or signed-out answer of the further members read was shown as the generic failure of that list.

# Built

- The sentence now says that not all reasons are shown and that the plan was rejected for more reasons than are listed, in English, Korean and Arabic. It moved from a note near the top of the detail, where it also showed when the reasons were not on screen, to the foot of the list of reasons.
- When a plan changes more than one relationship kind, each changed member shows the kind under its name, with the word the "Changes by relationship" table uses. A plan of one kind is shown as before.
- A denied answer of the further members read sets the page's denied state, and a signed-out answer reads the plans again, which shows the sign-in state; neither is shown as a list failure.

# Review

Copy and composition check (ui-designer, from screenshots, 2026-11-25): pass. It asked for the sentence to sit with the list of reasons, which was done, and preferred the kind as a line under the member's name over a column of its own. It did not see the list of reasons or the sentence in Korean and Arabic renders. The fact that the flag is about violations was confirmed by the independent review of HYB-305 increment 5c; this change was not reviewed again by that reviewer, being copy and a display line in the console.

# Verification

Console tests 1424, 1421 passed, 3 skipped (`live-render.test.tsx`: the sentence in three languages and the kind line). Rendered check `tools/plan-detail-qa.mjs`, 25 passed, in three languages at 1440 and 390, with plans written by the check, since the sample has no plan. The owner has not reviewed the screen.

# Not verified

No plan of more than 100 violations exists to show the flag from a real service; the rendered check sets it on a synthetic rejected plan.
