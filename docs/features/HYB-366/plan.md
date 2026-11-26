---
type: DesignPlan
title: P05 retire confirmation reads the replacement it shows
description: The confirmation of a retirement reads the replacement's requirement row itself when the page does not hold it, so a reload, a later page revision or a frozen proposal shows the replacement's state.
status: draft
---

# Scope and ownership

HYB-366, machine:mac_mini. Non-blocking observations of the HYB-359 reviews. Console only; no read, command or field of the service changes.

# Facts

- In HYB-359 the row of a replacement outside the page's requirement rows was read when the person chose it in the select, and its state was copied into the confirmation's context. The rows read were kept for the life of the form whatever the page revision; a proposal sent while the read was under way froze a context without a state; a frozen replacement restored after a reload was never read and said "Unknown".

# Built

- `useReplacementDetail(api, page, target)` in `release-evidence/LiveRetireForm.tsx`: the page's row when it holds the target; otherwise one read of the requirement by target (the existing requirements read), held under the page revision and the target, so a new revision reads again. `RetireFacts` uses it, so the editing confirmation and a frozen one show the same thing: the state, nothing while the read is under way, or the sentence that the details could not be read. The context no longer carries the state, and the select no longer reads on choice.
- The rule deciding whether a chosen value stays an option is `replacementChoices` in `live-model.ts`, with a unit test.
- The failure sentence has `margin-block: 2px 0`.

# Verification

Console tests 1426, 1423 passed, 3 skipped (one added). Rendered check `tools/replacement-search-qa.mjs` 55 passed: the confirmation with the read answered (English, Arabic) and failed (Korean), and the sentence's margins.

# Not verified

A frozen proposal after a reload and a change of page revision were not exercised in the browser; a synthetic replacement cannot be proposed to the real service, and the sample has no replacement outside the page's rows.
