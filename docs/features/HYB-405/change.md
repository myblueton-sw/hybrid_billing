---
title: HYB-405 O01 view group draft answers
status: stable
ticket: HYB-405
updated: 2026-11-28
---

# Findings from HYB-401 and what was decided

| Answer | Shown before | Decision |
| --- | --- | --- |
| Lost or unrecognised answer | "The draft could not be saved." | Changed. The outcome is unknown, so stating it as unsaved breaks [HB-34](../../planning/v1.16/requirements.md). |
| Generic 422 `prerequisite_failed` | "A draft can hold up to 1000 members." | Kept. In `apps/api/server/groupdraftcmds.go`, draft creation refuses with `prerequisite_failed` only when the members exceed the per-draft cap, or when the open-draft limit is reached. The screen already chooses between those two from the list state. The routed check sent a refusal the real server would not send for one member. |
| 503 `dependency_unavailable` | "The relationships could not be read. Nothing was saved." | Kept. The command returns this before applying anything, when it cannot read parties, revisions or open drafts. |

# Change

- `refusalOf` (`drafts-model.ts`) returns a new class, `unknown`, for these failures: a non-`AdminError`, `unavailable` without `unsupported_command`, and `partial`.
- For that class, the draft notice says: "This change may have been applied. No answer confirming the result was received. Check the drafts before trying again; your input is kept." This covers creation, the first batch of an edit, and discard.
- When a later batch is lost, the partial notice says the rest may or may not have been saved, instead of "The rest could not be saved".
- English, Korean and Arabic.

# Verification

- `tsc` clean. Console tests: 1,515 total, 1,510 pass, 0 fail, 5 skipped. The classification test covers `partial`, `unavailable`, no failure, and an unclassified failure.
- `tools/other-outcomes-qa.mjs --recipes O01:groups` in English 1440, Korean 1440 and Arabic 320, with the texts read back:

| Answer | Checks |
| --- | --- |
| unknown | 16, none failed |
| refused | 19, none failed |
| unavailable | 16, none failed |

- `organizations-qa` 278 and `members-qa` 73 pass.

# Limits

- The partial notice for a lost later batch was not rendered by a check.
- The Korean and Arabic wording has not been reviewed by native speakers.
