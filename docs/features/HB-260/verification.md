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

# Planning cross-check (HYB-291)

Added by the [HYB-291 planning cross-check](../../reviews/HYB-291-planning-cross-check/report.md) on 2026-11-17. A record added by HYB-291; it changes no result above except where marked. The findings come from a read-only auditor; the HYB-291 writer agent (delegated by root) wrote this trace, the residual notes and the copy changes; the HYB-291 change review is their first independent check.

## Requirement trace

Source: `docs/ui/screens/L01/spec.md:18`, `L02/spec.md:18`, `L03/spec.md:18` (HB-259 is the API, HB-260 the screens). Status meanings are in the [cross-check report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). No acceptance test was run.

| ID | Status | Reason | Record |
| --- | --- | --- | --- |
| HB-01 | deferred | Covered: sign-in and activation are bound to one workspace (L02). Not covered: party, organization and billing-entity separation, which L01–L03 do not build. | HB-259/plan.md |
| HB-03 | deferred | Addressed within pre-launch scope for sign-in sessions evaluated by `authpolicy` (L01, L03); registry-wide and leakage acceptance not built. | HB-259/plan.md |
| HB-36 | deferred | Covered: first-administrator activation through a one-time link in a 0600 file (L02). Not covered: installation packaging and support paths. | HB-259/plan.md |
| HB-42 | deferred | Local accounts with set and reset links are built; MFA, SSO and LDAP are residuals. | HB-259/verification.md Residuals |
| IAM-01 | deferred | Covered: activation and set links are one-time and invalidated on use. Not built or tested: concurrent first-administrator creation and acceptance from another workspace. | HB-259/verification.md |
| IAM-02 | deferred | Only local password accounts exist; SAML, OIDC and LDAP sign-in are residuals (HB-257 out of scope). | HB-260/plan.md:132-146 |
| IAM-03 | deferred | Does not arise while only local accounts exist; applies when another identity provider is connected. First recorded here. | this section |
| IAM-04 | deferred | Local link expiry, one-time use and an identical 401 are built; issuer, audience, signature, nonce, SAML replay and LDAP checks wait for federated sign-in. | HB-260/plan.md:132-146 |
| IAM-05 | deferred | Suspend and depart belong to M01 (HB-263); effects on keys, jobs and files and the external detection delay are not measured. | HB-263/plan.md |
| IAM-06 | deferred | Recovery is administrator-issued only (HB-263) with last-administrator protection; MFA recovery and identity unlinking are not built. | HB-259/verification.md Residuals |
| IAM-07 | deferred | No 100,000-user test; M01 server search and filters are HYB-294. | HYB-291 report |

## Planning cross-check residuals (HYB-291)

- **Administrator reset copy.** L03 said administrator-issued reset links were not available, although HB-263/264 ship `identity.issue_reset_link`. Fixed in HYB-291 (`apps/console/src/service/copy.ts`).
- **Security-focused review not stated.** HB-257 (Verification for every phase) requires a security-focused review for phases 1–3. This record names design and change reviews but no security-focused review. None is claimed here; a security re-review or a recorded reason is open.
