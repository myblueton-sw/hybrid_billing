---
type: ImplementationPlan
title: Live M05 my-security screen in the console service mode, bound to the HB-270 API
description: |
  Phase 3j of the pre-launch plan (HB-257).
  In service mode M05 renders:
  - an account view (read-only identity, membership and sign-in method);
  - a methods view (password change);
  - a sessions view (own sessions, end one or all others, sign out here);
  - recovery and notifications views marked unavailable in pre-launch.
status: stable
---

# Scope and ownership

[HB-271](https://linear.app/hybrid-billing/issue/HB-271) binds M05 to the [HB-270](../HB-270/plan.md) API. Base: main `0818bb9`.

**Ownership**
- Root (Claude Code) owns the plan, implementation, tests and browser QA.
- The `ui-designer` role (AI, read-only) writes the visual specification and compares the final renders.
- A separate read-only reviewer reviews the design and the change, with a security focus, because the screen handles passwords and sessions.
- Machine: mac_mini.

**Allowed files**
- `apps/console/src/service/admin.ts` and its test.
- `apps/console/src/modules/identity/pages/my-security/{Live.tsx,live-copy.ts,live-model.ts,live-model.test.ts}` (new).
- `apps/console/src/modules/identity/pages/users/live.css` (M05-scoped rules only).
- `apps/console/src/app/screen-groups/identity.tsx`.
- `apps/console/package.json` (the test list).
- `docs/features/HB-271/*`.

# Design

Everyone signed in can use M05: it acts only on the caller's own account. It is not limited to administrators.

## account

Read-only facts:
- user ID;
- login;
- email;
- workspace and role;
- installation administrator, shown only when true;
- sign-in method ("Password", with the login);
- signed in at;
- the policy's maximum sign-in age, if any.

A note says that the name, email and other sign-in methods cannot be changed in pre-launch.

## methods

**Fields**
- The password change form has three fields:
  - current password, with `autocomplete="current-password"`;
  - new password, with `autocomplete="new-password"`;
  - confirmation, also with `autocomplete="new-password"`.
- A read-only `autocomplete="username"` field holds the login, so a password manager saves the new password against the right account.
- The confirmation is checked locally. "New equals current" is also checked locally, with its own message. Neither check sends a request.

**Rules hint and impact**
- The form shows the rules hint and an impact line saying the person's other sessions will be signed out.
- It also shows a fixed warning: repeated wrong current passwords sign you out and can lock sign-in for up to 15 minutes, across this screen and the sign-in page.
- The warning shows no remaining count. The API returns none, and the account lockout also counts sign-in failures made elsewhere.

**Handling the password values**
- The form is `method="post"` with no `action`, and the submit handler calls `preventDefault` first.
- The inputs have no `name`, are uncontrolled, and are read through refs at submit time.
- The values are cleared in a `finally` after every response, and again on unmount, view change or locale change.
- The values are never put in state, the URL, storage, notices, errors or the console.

**While a request is pending**
- Submit is disabled and re-entry is ignored. Every request reserves one of the session's attempts on the server, so a double click would otherwise spend two.

**Hidden case**
- When `password_set` is false, the form is replaced by a note that this account has no password to change.

**On success**
- A status message says the password was changed and other sessions were signed out. It gives no count, because the server returns none.
- The session list is refreshed. A failed refresh never turns the success into an error.
- The form is reset and focus returns to the heading.

**Failures**

| Code | Result |
| --- | --- |
| `wrong` | A field error on the current password, with `aria-invalid` and `aria-describedby`; focus moves to that field |
| `locked` | "Sign-in is locked after too many attempts. Try again in up to 15 minutes." |
| `limited` | The retry time |
| `form` | A rules error on the new password; focus moves to that field |
| `conflict` | "Your password changed elsewhere; sign in again" |
| `unauthenticated` after at least one `wrong` in this view | "Signed out after too many wrong attempts; sign-in may be locked for up to 15 minutes", then the sign-in-again path |
| other `unauthenticated` | The existing sign-in-again path |
| `busy` (returned before any write) | "Not responding; nothing was changed" |
| `unavailable`, `partial` or an unknown error (the outcome is unknown) | "We could not confirm whether your password changed", with guidance to reload and try the new password first |

## sessions

- A table of the person's own sessions: current first, then signed in and ends.
- Each other session has an "End" button. There is also "End all other sessions", plus "Sign out here", which uses the existing logout.
- A persistent note: a session can stay listed after 30 minutes of inactivity has already ended it, until 12 hours after sign-in.
- After ending sessions the list reloads, a `role=status` message reports the count returned by the API, and focus moves to the list heading.
- Each End button's accessible name includes that session's sign-in time. The current session is marked in text ("This session"). Header cells have `scope="col"`.
- **Failures when ending sessions:**
  - `invalid` means "That session has already ended"; the list reloads and focus moves to the list heading.
  - `unauthenticated` follows the sign-in-again path.
  - `limited` shows the retry time.
  - `busy`, `unavailable` and other errors mean the result could not be confirmed, with a reload.

## recovery and notifications

Both are unavailable views with "Go to My account". No API call is made.

## Copy

EN, KO and AR copy follows the designer specification. Failure copy is keyed by code.

## External references

These are recorded from the designer specification, and root checks them before the PR.

# Review history

- **Change review round 1 (FAIL):**
  - M1: a lost response was reported as "nothing changed". `unavailable` and unknown errors now say that the result is unconfirmed, with a test.
  - L1: inputs are cleared in a layout-effect cleanup on unmount.
  - L3: the pending flag lives in the parent, and the form does not remount while a request is pending.
  - L4: the retry time is shown on end-session limits.
  - L5: the conflict and load copy is corrected, and Done returns to the list when stepping.
  - L6: the plan's ordering test is dropped.
  - L2: a residual.
- **Design round 1 (FAIL):** M1 (Medium) was the numeric remaining-attempts warning and the unexplained lockout after a sign-out; the warning is now fixed text and the sign-out message names the lockout. L1–L6, H1 and H2 are addressed above.

# Tests and verification

**Unit (model)**
- Confirmation matching and the new-equals-current check.
- Failure-to-outcome mapping, including the lockout sign-out choice and an unknown outcome. Session order comes from the server and is not re-sorted in the client.

**Browser QA against a live `hb-api`**
- The account facts for an administrator and for a member. The member can also change their password and end sessions.
- Password change:
  - mismatch, wrong current password, form refusal;
  - success, after which a second browser context is signed out;
  - the fifth wrong attempt signs the person out with the lockout message, after which sign-in is locked;
  - the account locked by earlier sign-in failures gives the `locked` message;
  - a double click on submit sends one request.
- Fields empty after every response.
- Sessions: end one, end all others, sign out here, and end a session that has already ended.
- The unavailable views.
- EN, KO and AR at 1440 and 390: no horizontal scroll, focus returns, neutral computed borders.

**Regression:** rerun the M01–M04 QA.

# Residuals for the owner

- No MFA or recovery, no linking of other sign-in methods, no profile changes, no notification preferences.
- An idle-expired session can stay listed.
- The lockout sign-out message depends on a wrong attempt made in this view. After a reload or a screen change, the fifth wrong attempt shows the plain signed-out message.
- KO and AR copy is an unreviewed AI draft.
- Screen acceptance (FIX) is the owner's alone.
