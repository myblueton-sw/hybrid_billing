---
title: HYB-403 M05 password change with an unknown outcome
status: stable
ticket: HYB-403
updated: 2026-11-28
---

# Problem

When the answer to a password change was lost or not recognised, M05 showed the title "Could not save this change" above a body saying the change could not be confirmed and may already have happened. The title contradicted the body. [HB-34](../../planning/v1.16/requirements.md) keeps an unknown result distinct from a failure.

# Change

- `changeUnsettled` (`my-security/live-model.ts`) is true when the password outcome is `unknown`. An `unknown` outcome comes from a non-`AdminError`, an `unavailable` failure, or any other failure the screen does not classify.
- For an unknown outcome, M05 now uses the title "Your password may have changed" (Korean and Arabic included) and keeps the existing body.
- The definite outcomes keep "Could not save this change": wrong password, locked, rate-limited, rule failure, conflict and busy.

# Verification

- `tsc` clean. Console tests: 1,514 total, 1,509 pass, 0 fail, 5 skipped. New assertions in `my-security/live-model.test.ts`.
- `tools/other-outcomes-qa.mjs --screens M05`:

| Answer | Language, width | Checks | Title shown |
| --- | --- | --- | --- |
| Lost | English 1440 | 11, none failed | "Your password may have changed" |
| Refused | Arabic 1440 | 13, none failed | "ربما تغيّرت كلمة المرور" |
| Lost | Korean 320 | 11, none failed | — (the explorer did not reach the password form at this width) |

- An unmapped refusal is treated as unknown, as in HYB-402. That is cautious, not false.
- Independent review was not repeated. The change applies the reviewed HYB-402 rule to the one path that review named.
