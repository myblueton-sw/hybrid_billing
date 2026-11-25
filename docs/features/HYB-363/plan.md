---
type: DesignPlan
title: P04 load-more buttons are described by what they load
description: Each "Load more items" button of the retention screen is tied to the heading of its class, run, hold or section, so that buttons of the same name can be told apart without sight.
status: draft
---

# Scope and ownership

HYB-363, machine:mac_mini. Raised by the ui-designer check of HYB-305 increment 4h. Console only; no visible change.

# Facts

The retention screen's shared foot (`retention/live-format.tsx` `ItemsFoot`) renders a button "Load more items" under each class's items in the expiry view, under the required items in the holds view, inside each run card and inside each hold card. Several can be on one page with the same name and the same failure button "Try again", and nothing in the accessibility tree said which list a button belongs to.

# Built

`ItemsFoot` takes the identifier of a heading and sets it as the button's `aria-describedby`: the class heading (`p04-class-h-…`), the run's heading, the hold's heading, and the "Required items" heading. The name of the button is unchanged, so its visible label and its accessible name still match.

# Verification

Console tests 1425, 1422 passed, 3 skipped (one new test for the expiry view). The rendered checks `tools/items-qa.mjs` (70 passed) and `tools/members-qa.mjs` (73 passed) resolve each button's description to its heading's text in three languages. Not verified: an actual screen reader; the checks read the DOM relation, not speech.
