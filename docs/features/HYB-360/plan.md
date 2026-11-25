---
type: DesignPlan
title: P02 subscription folds stay inside the page
description: Correction of the history and signing key tables inside a subscription's fold on the channels view of P02, which pushed the page wider than the viewport and overlapped their own columns between 768 and 1440 pixels.
status: draft
---

# Scope and ownership

HYB-360, machine:mac_mini. Found by the rendered check of HYB-305 increment 3b. Console styles and one column list; no data, command or contract change.

# Facts

Reproduced on main be6a327 with the sample data, every "History and signing keys" fold open, in English, Korean and Arabic:

| Viewport | Page width before | Element past the edge |
|---|---|---|
| 1440 | 1444 | a sign-in name in a fold table |
| 1280 | 1308 | the same |
| 1024 | 1052 | the same |
| 390 | 390 | none (rows are stacked) |

The issue named 4 pixels at 1440; at 1280 and 1024 it was 28. The cause was a rule that keeps sign-in names and identifiers on one line in every table cell from 768 pixels up, inside tables of fixed column widths. A screenshot at 1024 after the first correction showed two more defects of the signing keys table: the key identifier ran over the next column, and the state tag was broken across two lines.

# Built

- Inside a fold, sign-in names and identifiers wrap at any character.
- The signing keys table gives the key 20% and the state 14% (they had 16% and 12%); the state tag is kept whole.

# Verification

`tools/channel-folds-qa.mjs`, new: with every fold open, at 1440, 1280, 1024, 768 and 390 in three languages, the page is not wider than the viewport, nothing spills out of a table cell, and no state tag is broken; 16 passed. The earlier check `tools/subscription-history-qa.mjs` was run again. Console tests unchanged in number. I looked at the English render at 1024 before and after. A designer did not review this; the change follows the wrapping the same screen already uses for cells marked to wrap. The owner has not reviewed the screen.
