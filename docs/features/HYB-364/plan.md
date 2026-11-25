---
type: DesignPlan
title: P04 hold card keeps its facts together beside a long outcomes list
description: Correction of the hold card of the retention screen, whose facts on the left were spread over the height of the outcomes list on the right when that list was long.
status: draft
---

# Scope and ownership

HYB-364, machine:mac_mini. Raised by the ui-designer check of HYB-305 increment 4i. One style rule; no data, command or contract change.

# Facts

Reproduced on main e993712 at 1440 with a hold given fifty outcomes by the check: the five facts of the card (owner, placed by, placed, hold reason, placing reason) were spread down the card, with up to 759 pixels between one fact and the next in English (691 in Korean, 750 in Arabic). At 1024 and 390 the card has one column and was not affected. The cause: on wide screens the card is a two-column grid, and the facts list on the left is itself a grid that stretched its rows to the height of the right column.

# Built

The facts list aligns to the top of the card and no longer stretches its rows (`.p04-hold-grid > .p04-hold-facts`).

# Verification

`tools/hold-card-qa.mjs`, new: with fifty outcomes, at 1440, 1024 and 390 in three languages, at most 40 pixels between one fact and the next, and the page not wider than the viewport. Three of ten checks failed before the change (the three at 1440) and all ten pass after it. My first version of the measurement did not detect the defect, because it measured the stretched boxes instead of their text; it was corrected before the fix was applied. I looked at the English render at 1440 before and after. A designer did not review the result; the owner has not reviewed the screen.
