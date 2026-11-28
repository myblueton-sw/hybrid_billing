---
title: HYB-407 field refusals on M01, M02 and M05
status: stable
ticket: HYB-407
updated: 2026-11-28
---

# Problem

HYB-399 step J answered commands with `422 invalid_request` and field names that some forms do not show.
- M01 suspend user and M02 trust proposal then showed the summary title "Check the following" with nothing under it.
- M05 password change classified the answer as unknown ("Your password may have changed").

# Change

- **Commands (`commandFailure` in M01 and M03, `policyFailure` in M02).** A server `invalid_request` reads: "The service did not accept a value in this form. Nothing was saved. Check the details and try again."
- **Reads (`failureText`).** A server `invalid_request` reads: "The service did not accept this request. Reload the list and try again." Both sentences are provided in English, Korean and Arabic.
- **Malformed answers.** An `invalid` failure without the server code is raised by the console itself, when a server answer is malformed. The console checks for this only after a response arrives; its own pre-send validation never raises it. It is classified as an unknown outcome on commands.
- **M05 password change.** `invalid_request` is treated as the form-rule outcome. An `invalid` failure without the server code stays unknown.

# Verification

- `tsc` clean. Console tests: 1,517 total, 1,512 pass, 0 fail, 5 skipped. Tests were updated for both texts and the classification.
- `tools/other-outcomes-qa.mjs`:

| Answer | Scope | Checks | Result |
| --- | --- | --- | --- |
| invalid | M01–M05, English 1440 | 58 | None failed; M01 suspend shows the fallback |
| invalid | M02 and M03 recipes, English 1440, Korean 1440 and Arabic 320 | 34 | None failed; M02 shows the fallback in each language |
| unknown | M01–M05, English 1440 | 47 | None failed |
| invalid | M01 and M03, after separating the read text | 48 | None failed |

- Independent change review: conditions.
  - It confirmed that every `invalid` raised by the console comes from a malformed answer.
  - It confirmed that commands do not apply anything before answering `invalid_request`. This was read for the command executor and the trust and password handlers; partial writes in every other handler are inferred from the contract.
  - Its one condition was that read screens should not say "this form… Nothing was saved". This was taken: reads now use their own sentence.

# Limits

- No rendered check exercised a read with a server `invalid_request`; the read text is covered by the unit test.
- The Korean and Arabic text has not been reviewed against the glossary.
- TrustConfirm still closes for good when an invalid answer names a field other than the rationale. This is existing behaviour, now visible with the fallback.
