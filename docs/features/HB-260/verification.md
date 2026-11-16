---
type: VerificationRecord
title: HB-260 sign-in screens verification
description: Actual checks run for phase 2b of the pre-launch plan (L01, L02 and L03 bound to the sign-in API in console service mode), plus design and review outcomes and residuals.
status: final
---

# Scope

[HB-260](https://linear.app/hybrid-billing/issue/HB-260) adds `apps/console/src/service` and the mode switch in `main.tsx`, following [plan.md](plan.md).

- **Entry.** Service mode is entered with `?service=api` or `/activate`. Every other URL renders the existing review shell, unchanged.
- **Gate and screens.** The gate checks `GET /api/v1/session` and offers:
  - L01, sign in;
  - L02, set or reset a password, or activate the first administrator, from a one-time link. The token comes from the fragment and is removed before any request.
  - L03, recovery guidance.
- **Signed-in chrome.**
  - A fixed pre-launch notice that data resets on restart.
  - A session strip above the shell, with workspace, user and sign out, that follows the shell's language.
- **Client.** It handles the pre-session token (shared, refreshed once when expired) and stable failure kinds.

Copy is in EN, KO and AR, as candidates pending locale review.

Base: main `8df51be`. Machine: mac_mini.

# Checks run (2026-11-16, this machine)

| Check | Result |
| --- | --- |
| `npm run typecheck` | pass |
| `npm test` | 326 of 326 pass, including 17 new client and helper tests |
| `npm run build` | pass |
| Browser flow against the real `apps/api` and the production build (`vite preview`) | 48 of 48 checks pass (list below) |
| API down, then restarted | the unavailable message shows, and Try again clears it once the API is back |

**Browser flow coverage:**
- **Activation.**
  - The activation link loses its fragment from the address bar.
  - A too-short password gives a focused, neutral error summary (computed 1px uniform border).
  - Setting the password succeeds, and the result survives a language change.
- **L01.**
  - The notice is in flow, with only a bottom divider.
  - Empty fields give field-linked errors.
  - A wrong password gives the generic message and clears the password.
  - The show/hide toggle works.
- **Session.**
  - Signing in shows the session strip, which is in the sign-in language at KO, AR and EN 1440 and KO and AR 390.
  - A reload keeps the session.
  - Signing out shows the signed-out status once, then removes it from the URL.
- **Other screens.** L03; an invalid link.
- **Layout.** No horizontal scroll and 16px inputs on L01 and L03 at KO, AR and EN, 1440, 768 and 390.

**Console and network errors recorded during the flow:**
- 401 on `/api/v1/session` while signed out (expected);
- 422 on the invalid-link check (expected);
- 503 from the existing shell's calls to the Python readers, which were not running. That path is HB-261.

**A QA environment limit.** Running the dev server from the ticket worktree blocked font files behind a symbolic link (403). Final renders therefore use the production build, which bundles the fonts.

# Bugs found and fixed

- Two concurrent calls each fetched a pre-session token, and the second cookie overwrote the first (403). The client now shares one in-flight request, found by the browser run.
- The reviewers found the rest:
  - an expired pre-session read as "service not responding";
  - activation form errors read as a password rule;
  - every failed link check read as an invalid link;
  - L01 showed no unavailable state;
  - the sign-in language was lost in the shell;
  - changing the language on the result page re-checked the link and replaced the success with "cannot be used".

  All are fixed. See plan.md.

# Design and reviews

- **`ui-designer` agent role (AI, read-only).**
  - Design input before implementation.
  - Render comparison: L01–L03 PASS with minor fixes, and the session strip FAIL on language and weight, both fixed.
  - Re-check: the strip, the signed-out status and L03 PASS. The mobile footer grouping and the unavailable box got minor fixes.
  - Final check:
    - the unavailable box and the KO strip PASS;
    - the mobile footer meta row and the AR value truncation got one more fix each. Both were then measured by root: both footer groups render as one line (18px tall each), and the AR workspace value is a left-to-right element truncated at its end. Root also inspected the render.
- **`independent-reviewer` (AI, read-only), change review.**
  - Round 1 FAIL (M1–M5).
  - Round 2 FAIL (N1, re-check on language change), fixed.
  - Round 3 PASS, with one Low folded in: the unavailable alert re-mounts on each retry, so screen readers announce it again.
- The external references and the recorded deviations are in plan.md.

# Residuals

- The session strip sits above the shell header, not inside it.
- In Arabic, the logo sits about 20px inside the gutter (canonical logo SVG margin, geometry unchanged).
- The shell's screens are not yet bound to the service (later phases). The Python readers behind the Go session come in HB-261.
- Administrator-issued resets, MFA and SSO.
- Locale review of the new copy.
- Owner screen-review acceptance (FIX) of the bound L01–L03.
