---
title: HYB-404 M04 focus after a failed command
status: stable
ticket: HYB-404
updated: 2026-11-28
---

# Problem

HYB-401 found that after "Create service account" was answered as refused, conflict, unavailable or lost, keyboard focus fell to the page body. M04 keeps a sent command in a held notice (`IntentNotice`). That notice announced its result through a polite live region but never took focus, and the form's submit button was disabled at that moment.

# Change

`IntentNotice` (`service-accounts/command-ui.tsx`) moves focus to its title when the phase it has watched goes from `sending` to `refused`, `unknown` or `lapsed`. Its title is the one that says "Command refused" or "Command outcome unknown".
- A notice restored from an earlier session does not take focus, because the notice never watched it sending.
- A completed command keeps its existing focus return.
- `settledAfterSending` holds the rule and is unit-tested.

# Verification

- `tsc` clean. Console tests: 1,515 total, 1,510 pass, 0 fail, 5 skipped.
- `tools/other-outcomes-qa.mjs --recipes M04:accounts`, English 1440 and Arabic 320: refused, unknown, conflict and unavailable all pass, with focus on the notice title (H3) in each language.
- With the four recipes together: unavailable 36 checks and refused 44 checks, none failed.
- The existing M04 checks exit 0: `m04-errors-qa`, `service-accounts-qa`, `m04-consumer-qa` and `m04-completion-qa`.
- The M04 recipe now waits for each catalog selection list to load, after one run started before the owner list arrived.

# Limits

- Focus only; the wording and permission logic are unchanged, so no separate independent review was run.
- Rotation and account actions use the same notice, but only creation was measured.
