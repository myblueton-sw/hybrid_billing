---
type: ImplementationPlan
title: Sign-in screens L01, L02 and L03 bound to the pre-launch sign-in API in a console service mode with a fixed pre-launch notice and a session strip
description: Phase 2b of the pre-launch plan (HB-257). The console gains a service mode in which it requires a session. L01 signs in, L02 sets a password or activates the first administrator from a one-time link whose token stays in the URL fragment, L03 gives administrator-assisted recovery guidance, and every service-mode screen shows that data resets on restart.
status: final
---

# Scope and ownership

[HB-260](https://linear.app/hybrid-billing/issue/HB-260) is the second part of phase 2 of the [pre-launch plan](../HB-257/plan.md). It binds the sign-in screens to the API from [HB-259](../HB-259/plan.md). The session gate in front of the Python endpoints is HB-261.

Base: main `8df51be`.

Roles:
- Root (Claude Code) owns planning, implementation, terminology writing, QA and integration.
- The Claude `ui-designer` agent role gave design input before implementation and compares the renders after it.
- A separate read-only `independent-reviewer` reviews the change.

Machine: mac_mini.

Allowed files:
- `apps/console/src/service/*`;
- `apps/console/src/main.tsx` (the mode switch only);
- `docs/features/HB-260/*`.

The existing review shell, its synthetic and local modes and the `entry-*` preview screens are not changed.

# Observed baseline

- The console routes every registered screen through a hash router. L01–L03 render as previews in `RemainingEntryShell`, with placeholder text such as "Browser-only preparation" and "Identity service not connected". Renders were taken at 1440 and 390 in EN, KO and AR.
- The Vite proxy sends `/api/v1` to `apps/api` on port 8820 (HB-258).
- The sign-in API:
  - `GET /api/v1/auth/csrf`, `POST /api/v1/auth/login`, `/logout`, `/link` and `/redeem`, and `GET /api/v1/session`;
  - every login failure is the same 401;
  - 429 carries `Retry-After`;
  - the password is 12 to 256 bytes and must not equal the login.
- The seed writes one-time links of the form `<console origin>/activate#<token>`.
- The L01–L03 specs (`docs/ui/screens/L0x/spec.md`, local) require:
  - no account enumeration;
  - tokens never in URLs;
  - field-linked error summaries with focus moved to them;
  - no double submission;
  - the keyboard order workspace, method, ID, password, submit;
  - one-column mobile layouts.

# External references (consulted 2026-11-16)

| Source | Observed | Adopted | Not adopted |
| --- | --- | --- | --- |
| GOV.UK Design System, Passwords pattern | Allow paste; do not say which of ID or password was wrong; do not keep the password after a failed attempt; explain rules identically wherever passwords are created; 5–10 attempts before lockout | All of these | Emailing on reset (no mail in pre-launch) |
| GOV.UK Design System, Password input component | Hidden by default; show/hide toggle with distinct accessible names; `autocomplete` `current-password` or `new-password`; `spellcheck=false`; `autocapitalize` off | All of these | — |
| W3C WCAG 2.2, Understanding SC 3.3.8 Accessible Authentication (Minimum) | No cognitive test without an alternative; never block paste or password managers; correct autocomplete markup | Paste and autofill allowed everywhere | — |
| GOV.UK error summary (by the designer's account) | Red bordered summary box | Summary at top, `role="alert"`, focused, with links to fields | The red border, which the owner's no-coloured-border rule forbids; the summary is a neutral 1px box with danger-coloured heading text |

# Decision

## Service mode

- **Entering the mode.** The console enters service mode when the path is `/activate`, or the query has `service=api`. Otherwise it renders the review shell exactly as before.
- **Language.** The locale comes from `lang` in the query (`ko` by default). The language select rewrites it in place.
- **Gate.** `ServiceRoot` calls `GET /api/v1/session`:
  - without a session it shows L01, or L03 when `view=help`;
  - with a session it renders the existing shell, under a session strip.
- **Errors.** A 503 or a network failure shows L01 with an unavailable message and a Try again button, which checks the session again.
- **Activation path.** `/activate` takes the token from the fragment once and removes it from the address bar in a layout effect, which runs before any request. The token is never put in a URL.
- **After sign-in.** The console keeps any requested route and query from the hash and sets the shell's language there to the one used on L01. The session strip follows the shell's language from then on.
- **Status.** A `status` query parameter is shown once and then removed from the URL.

## Client

`service/api.ts` wraps the six calls:
- it sends `X-HB-Client: console`, and the pre-session token in `X-CSRF-Token`;
- concurrent calls share one in-flight pre-session request, so their cookies cannot clobber each other;
- an expired pre-session (`403 csrf_refused`, after its 10-minute cookie) is refreshed and the call retried once;
- a pre-session response without a token is `unavailable`;
- it maps failures to stable kinds: unauthenticated, limited (with `Retry-After`), link, password, form and unavailable;
- a network failure resets the pre-session token.

## Screens

These follow the designer's input. Copy is in EN, KO and AR and is a candidate pending locale review.

**Shared layout.**
- The entry header (logo and language) is kept.
- The pre-launch notice band follows directly. It is static and not dismissible: a neutral surface with a bottom divider and no accent.
- Main content then fills the viewport width:
  - at 1024px and above: a task column of 320–440px, then a help column separated by a 1px neutral divider;
  - below that: one column, with the task first.
- Inputs are 44px tall with 16px text and at most 440px wide.

**L01.**
- Fields in this order: Workspace, Email address (`autocomplete=username`) and Password (`current-password`, with a text Show/Hide toggle).
- Then the Sign in button and a "Can't sign in?" link to L03.
- Errors:
  - Empty fields give a field-linked summary.
  - Every server failure gives one generic message and clears the password.
  - 429 says when to retry, as relative time formatted for the locale.
  - Busy submits use `aria-disabled` and are ignored.

**L02.**
- The link is peeked first, with only a "Checking your link…" heading on screen.
- Headings: set gives "Set your password", reset gives "Reset your password", activate gives "Activate the first administrator".
- Set and reset show the login read-only, plus a hidden `username` field so password managers pair the new password with it. Activation adds a sign-in email and a contact email.
- The password fields are New password (with the rule hint, identical wherever a password is created) and Confirm.
- Lengths are checked in UTF-8 bytes, to match the server.
- An invalid, expired or used link shows one generic message and a link to sign in.
- A link check that fails because the service is unavailable or rate limited shows that reason and a Try again button. The token is kept in memory, so the user does not lose the link.
- Activation checks the sign-in email (3 to 200 bytes, no control characters) and the contact email (`@`, at most 254 bytes) in the browser, matching the server. Each has its own message.
- Any other server refusal of the form shows "We could not complete this…", never a password message. Both passwords are cleared after a failure and after success.
- On success, a result heading receives focus, followed by a Sign in button. The redeem call opens no session.

**L03.** Guidance only:
- forgotten password: contact the administrator; administrator-issued reset links are not available yet;
- too many attempts;
- after a restart;
- a Back to sign in button.

There is no form.

**Session strip.**
- It sits above the existing shell, showing the notice, then the workspace, the user and a visible Sign out button.
- Sign out posts logout and returns to L01 with "You have signed out."

**Deviation from the design input.** The designer proposed putting the session controls inside the shell's header actions. They are in a strip above the shell instead, so the 1,400-line `App.tsx` is not modified in this ticket. The designer's render comparison judges whether this is acceptable.

## Constraints

- No coloured borders or edge strips: the notice and error summary use neutral 1px borders, checked in computed style.
- Visible focus is kept, and the layout uses the full viewport width.
- Placeholders are never used as labels, and no icon-only control is used.

## Recorded deviations from the specs

- **Error focus.** The L01 spec says focus moves to the first error. The screens follow the GOV.UK pattern instead: focus moves to the error summary, and each summary item links to its field.
- **Method field.** L01 has no sign-in method field, because only password sign-in exists in pre-launch.
- **Session controls.** The session controls sit in a strip above the existing shell (see above).

# Residuals

- The session strip is not yet integrated into the shell header.
- The shell's screens are not yet bound to the service (later phases).
- Administrator-issued resets (phase 3).
- MFA and SSO.
- Locale review of all new copy by competent reviewers.
- Owner screen-review acceptance (FIX) of the bound L01–L03.
- In Arabic, the logo sits about 20px inside the right gutter, because of the canonical logo SVG's internal margin. This affects every entry screen, and the logo geometry is not changed.

# Verification plan

- Unit tests for the client:
  - pre-session handling, including concurrent calls and retry after failure;
  - failure mapping;
  - token never in a URL;
  - fragment parsing.
- `npm run typecheck`, `npm test` and `npm run build`.
- Rendered end-to-end flow against the real `apps/api` and the production build (`vite preview`):
  - activation link;
  - set-password errors and success;
  - L01 empty, failed and successful sign-in;
  - session strip;
  - reload keeping the session;
  - sign out;
  - L03;
  - an invalid link;
  - EN, KO and AR at 1440, 768 and 390, with no horizontal scroll and 16px inputs;
  - computed borders of the notice and the summary;
  - console and network errors recorded.
- `ui-designer` render comparison, and an independent change review.

# Review history

- 2026-11-16, design input before implementation: `ui-designer` agent role (AI, read-only), from renders of the current previews, the specs, the constraints and the references above. Adopted except the session-strip placement noted above.
- 2026-11-16, render comparison: `ui-designer` agent role (AI, read-only).
  - L01–L03 PASS with minor fixes:
    - the signed-out status is now a neutral bordered line at 16px;
    - the L03 aside has a heading;
    - the mobile footer has two deliberate rows.
  - Session strip FAIL, fixed:
    - its language now follows the shell's;
    - the workspace value has the same weight as the user.
  - The Arabic logo inset is a residual.

- 2026-11-16, re-checks: `ui-designer` (AI, read-only).
  - The session strip, signed-out status, L03 and unavailable box PASS.
  - The mobile footer grouping and the AR truncation were fixed and measured.

# Change review resolution

- Change review round 1 (separate independent reviewer, read-only): FAIL. Findings and what changed:
  - M1. An expired pre-session is refreshed and the call retried once.
  - M2. Activation fields are checked in the browser with their own messages; server form refusals no longer read as a password rule.
  - M3. A link-check failure that is not an invalid link offers a retry and keeps the token.
  - M4. An unavailable state with Try again on L01.
  - M5. The sign-in language is carried into the shell's hash, and the strip follows the shell.
  - Lows folded in:
    - L1, fragment removal in a layout effect;
    - L2, passwords cleared on success;
    - L3, an L02-specific failure copy;
    - L4, a hint saying how Korean and Arabic letters are counted;
    - L5, ASCII lower-casing for the same-as-login check;
    - L6, status removed from the URL after display;
    - L7, deep links kept;
    - L8, distinct activation messages;
    - L9, deviations recorded above;
    - L10, field-named visibility status;
    - L11, helper tests;
    - L12, pre-session body validation.
- Round 2 (same reviewer): FAIL on N1. A language change re-ran the link check and replaced a success result; the check no longer depends on the language. Lows L-a to L-d were folded in.
- Round 3 (same reviewer): PASS. The retry alert re-mounts so it is announced again.

