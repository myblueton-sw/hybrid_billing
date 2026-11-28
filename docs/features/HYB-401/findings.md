---
title: HYB-401 post-submission states of four forms with domain input
status: stable
ticket: HYB-401
updated: 2026-11-28
---

# Method

The generic explorer of HYB-399 step F could not reach four forms, because each needs valid domain input. `tools/other-outcomes-qa.mjs --recipes <screen:view,…>` adds a recipe per form. Each recipe opens the form from its localized opener, fills it, and marks its final button. The existing loop then presses it from the keyboard, carries on through any confirmation, and measures what happens after the answer.

| Recipe | Steps |
| --- | --- |
| `M02:policies` | Propose trust change, then a shorter sign-in time of 10 minutes, then Review proposal |
| `M03:grants` | Request access, then person admin2, then the first access option, then 7 days and a reason |
| `O01:groups` | New view group draft, then new group, then a name, then search source accounts, add one, and give a reason |
| `M04:accounts` | Create service account, filled as in `m04-errors-qa`: owner, purpose, organization, audience, the read scope and a reason. Needs `--service-catalog tools/fixtures/m04-service-catalog.json` |

A summary check requires every recipe to reach its command in every combination.

# Results

Each answer was run in English at 1440 and Arabic at 320, routed so that nothing reached the service.

| Answer | Checks | Failures |
| --- | --- | --- |
| refused (422) | 44 | 4, all M04 |
| unknown (lost) | 36 | 4, all M04 |
| conflict (409) | 44 | 4, all M04 |
| unavailable (503) | 36 | 4, all M04 |

M02, M03 and O01 passed every check in every answer:
- focus stays on a visible element in the page;
- a new alert or status states the answer;
- the layout and contrast hold.

The stated texts were read back.

# Defects found

- **M04:** after any failed creation, the failure alert is shown but keyboard focus falls to the page body. `CommandFailure` renders its box with `focusKey={0}`. Moved to HYB-404.
- **O01, lost answer:** "The draft could not be saved" is shown although the outcome is unknown (the HB-34 pattern fixed for M01–M03 in HYB-402).
- **O01, generic refusal:** the message names the 1,000-member limit, a cause the server did not give.
- **O01, 503:** a 503 is stated as "Nothing was saved".

The three O01 defects are moved to HYB-405.

# Limits

- The busy state, a successful completion and screen-reader announcement are not measured.
- Korean was not run for these recipes.
