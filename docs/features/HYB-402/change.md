---
title: HYB-402 lost command answers on M01, M02 and M03
status: stable
ticket: HYB-402
updated: 2026-11-28
---

# Problem

HYB-399 step G answered M01, M02 and M03 commands with a lost response: a network timeout error, so the outcome is unknown. These screens said "Could not save this change. The user service is not responding." That states that nothing was applied when the outcome is unknown.

Planning basis:
- [HB-34](../../planning/v1.16/requirements.md) says an accepted or unknown result must not be treated as settled.
- The domain contract keeps an unknown outcome distinct from a failure.

# Change

- `commandFailure` and `outcomeUnknown` (`users/live-model.ts`): an `unavailable` failure covers transport loss and any answer whose code the console does not map. Together with a non-`AdminError`, it now reads "This change may have been saved. No answer confirming the result was received, so it is not known whether this change was saved. Check the current details before sending it again." The text is in English, Korean and Arabic, and it promises no safe resend.
- Used by:
  - M01: the shared failure box, role period, bulk confirmation and match-and-add. The bulk preparation failure is titled "The preview may have been prepared"; "Nothing has been revoked" stays, because preparation revokes nothing.
  - M03: access requests, grants and revocations, through the new pure module `access/failure-model.ts`.
  - M02: policy and trust commands, through `policyFailure`.
- A `partial` failure, whose body already said the outcome may be partial, is no longer titled "Could not save this change".
- Read failures keep their own text.

# Verification

- `tsc` clean. Console tests: 1,514 total, 1,509 pass, 0 fail, 5 skipped. New tests are in `users/live-model.test.ts`, `authentication/live-model.test.ts` and `access/failure-describe.test.tsx`.
- Rendered, with `tools/other-outcomes-qa.mjs --answer unknown`:

| Screens | Language, width | Checks |
| --- | --- | --- |
| M01–M03, M05 | English 320 | 79, none failed |
| M01–M03, M05 | Arabic 1440 | 47, none failed |
| M01–M03 | Korean 1440 and Arabic 320 | 95, none failed |
| M01 | English 320 (reaches match-and-add) | 59, none failed |

  The stated texts were read back in each language.
- Existing checks pass: `m03-requests` 90, `m03-revoke-panel` 59, `m02-detail` 100, `m03-revocations` 81 and `authentication-loss`.
- Independent change review: conditions C1–C4, then N1 on re-review, all taken.

# Limits

- The Korean and Arabic wording has not been reviewed by native speakers.
- M05 password change uses the same "Could not save" title over an outcome-unknown body. That is moved to HYB-403.
- An unmapped 4xx answer is also treated as unknown. Giving it its own outcome needs a server contract review.
