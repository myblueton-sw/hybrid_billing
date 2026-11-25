---
type: VerificationRecord
title: Diagnostics rendered check and an ended session
description: Why command-diagnostics-qa.mjs stopped at the ended-session case of P01, the correction of the script, and the run that passes.
status: stable
---

# Outcome and ownership

HYB-355, machine:mac_mini, 2026-11-24. Not a product defect. The script expected how a screen treated an ended session before HYB-304. The script is corrected and runs to the end.

# Cause

`apps/console/tools/command-diagnostics-qa.mjs` answers the read of each P screen with 401 `unauthenticated` and waited for the screen's own panel with the refusal references. Since HYB-304 (commit ebec24c, 2026-11-20) a 401 ends the session for the whole console (`apps/console/src/service/Service.tsx`, `retireAuthentication`), and the sign-in page says "Sign in again.". The script was last changed under HYB-330 and was not updated with that change; HYB-304 added its own check, `authentication-loss-qa.mjs`.

Run before the correction, main 1b09c91: 257 checks passed, then a timeout on `.p01-live .command-diagnostics-details`; the screenshot showed the sign-in page with "Sign in again.".

# Correction

For the 401 case of the five screens the script now checks that the console is on the sign-in page, says "Sign in again.", shows neither the screen nor a session, and that after signing in again the screen reads without a refusal panel. Sign-in is one function used for both sign-ins. No product file changed.

# Run after the correction

Console dev server from main 1b09c91 and the API binary built from the same commit, on the loopback address with the sample seed and a new data directory; started for this run and stopped afterwards; synthetic sample data only. Exit code 0; 409 checks passed, none failed, no page error.

# Observation, no change

The "session ended" panels inside the P screens (`signedOut` in each `Live.tsx`) can no longer be reached by a 401 on a read, because the console leaves the screen first. Whether they are still reachable another way was not traced.

# Limits

One run. A fresh data directory is needed per run because the activation link is used once; the service refuses a data directory that was created beforehand with open permissions ("configuration is invalid"), so it must create the directory itself. No independent review; the change is a test script. Not an owner acceptance of any screen.
