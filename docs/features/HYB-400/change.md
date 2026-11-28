---
title: HYB-400 P02–P05 503 wording and refusal causes
status: stable
ticket: HYB-400
updated: 2026-11-28
---

# Item 1: a 503 shown as "may have been applied"

Kept. The command executor (`modules/commands/execute.go`) answers `dependency_unavailable` in several situations:
- a required record or read is unavailable;
- the idempotency store is full;
- an earlier request with the same key is still in flight (`inflight`).

In the last case the earlier attempt may still be applied. The P02–P05 screens resend with the same key. So "The last request may have been applied. Check the list, or send it again" is true for every 503, where "nothing changed" would not be.

# Item 2: refusal copy that names a cause

- **P05 withdrawal** ("Only the proposer can withdraw it"): kept. `releasetrace` and `perfgate` refuse a withdrawal with `prerequisite_failed` only when the user is not the proposer.
- **P05 approval** ("You cannot approve your own proposal"): changed. `prerequisite_failed` on approval means either self-approval or a register limit reached while applying (`ErrLimit` from `check`). The copy is now: "The register refused this approval. Nothing changed. You may be its proposer, who cannot approve it, or a register limit may have been reached." It is provided in English, Korean and Arabic.
- **P04 hold** ("This workspace already has 100 holds…") and the **O01 draft**: settled by HYB-405 and by the server evidence. Each refusal there has a single possible cause, or the screen already chooses between the causes from data it has. The P04 hold sentence was not re-traced here.

# Verification

- `tsc` clean. Console tests: 1,515 total, 1,510 pass, 0 fail, 5 skipped.
- `tools/form-states-qa.mjs --confirm 1 --outcome refused` on P05 shows the new text, and every check passes:

| Language and width | Checks |
| --- | --- |
| English 1440 | 59 |
| Arabic 320 | 59 |
| Korean 1440 | 59 |

- The outcome-text check now also recognises the Korean phrases "바뀐 것은 없습니다", "변경된 내용이 없습니다", "반영되었을 수 있습니다", "적용되었을 수 있습니다" and "다시 확인". With them, P02–P05 in Korean 1440 pass with every answer type:

| Answer | Checks |
| --- | --- |
| refused | 293 |
| conflict | 293 |
| unknown | 293 |
