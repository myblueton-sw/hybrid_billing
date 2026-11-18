---
type: ImplementationPlan
title: M05 self-service security API with the signed-in person's own account, sessions and password change
description: Phase 3i of the pre-launch plan (HB-257). It adds to apps/api a read of the person's own account, sessions and policy summary, a password change that verifies the current password and ends the person's other sessions, and ending one or all other sessions. No module rules change.
status: stable
---

# Scope and ownership

[HB-270](https://linear.app/hybrid-billing/issue/HB-270) adds the API behind M05 ("my security"). Base: main `ec159fa`. The M05 screen follows in HB-271.

**Ownership**
- Root (Claude Code) owns the plan, the implementation and the tests.
- An independent read-only reviewer reviews the design and the change, with a security focus: credentials and sessions.
- Machine: mac_mini.

**Allowed files**
- `apps/api/server`:
  - a new `self.go` and its tests;
  - route registration in `boundary.go`;
  - the per-session failure count map in `server.go`;
  - the two new codes in `errors.go`.
- `docs/features/HB-270/*`.

# Observed baseline

- **Sessions:**
  - Sessions live in `apps/api` (`sessions`: verifier, workspace, CSRF token, created, login) and in `modules/identity`, which enforces 30 minutes idle and 12 hours absolute.
  - `byLogin(login)` lists a login's session verifiers.
  - `endSessions(login)` ends all of them; logout ends the current one.
- **Passwords:**
  - `modules/passwords` holds PBKDF2 records keyed by login.
  - `Verify` returns a result code; `Hash` applies the form rules (`CodePasswordForm` on refusal); `Put` stores a record.
  - Hashing runs outside the domain lock through `derive`, which uses a bounded slot pool.
- **Sign-in checks:**
  - Sign-in compares the per-login `generation` taken before hashing, so a password change made during a sign-in voids that sign-in.
  - The global limiter `s.limit` throttles unauthenticated password attempts.
- **What is missing:**
  - No endpoint lets a signed-in person read their own account or sessions, change their password, or end other sessions.
  - There is no MFA, no identity-linking UI path, and no notification-preference module.

# Design

## Admission

Every endpoint admits the caller the same way:
- a current session cookie;
- the session's CSRF token in `X-CSRF-Token`;
- `s.access(found)`, which includes the policy's maximum authentication age.

No gate action is required. These endpoints only ever act on the caller's own login and sessions. The target is never taken from the request; it is always the session's own login.

## `GET /api/v1/me`

Read under the domain read lock. It returns:
- `user_id`, `login` and `email` (from the directory);
- `workspace`, `role`, and `membership` (`state` and `version`);
- `identities`: one entry, `{method: "local-password", login}`, because password sign-in is the only connected method in pre-launch;
- `password_set`;
- `installation_administrator`;
- `sessions`: the caller's own sessions, current one first, then newest first, each with:
  - `handle`: 16 hex characters of SHA-256("hb-session-handle" ‖ verifier). It is not reversible to the cookie or the verifier, and is stable for the session's life.
  - `current` (boolean);
  - `signed_in` and `ends` (created + 12 hours absolute);
  - `idle_minutes` (30), the same for every session;
- `policy`: `max_age_seconds` from the authentication policy of the session's own workspace, or 0 when there is none;
- `signed_in_at`;
- `now`.

## `POST /api/v1/me/password`

**Body:** `{current, new}`, decoded with `decodeStrings` (exactly those two keys; a body-size cap).

**Steps**
1. Under the domain write lock:
   - admit the caller;
   - validate the body;
   - reserve an attempt for this session;
   - read the login and its `generation`.
2. **Attempt reservation**
   - The per-session count lives in a server map keyed by session verifier. Entries whose session no longer exists are pruned each time an attempt is reserved.
   - At 5 reserved attempts, the request ends the session (401 `unauthenticated`).
   - Because the attempt is reserved before hashing, concurrent requests cannot exceed the limit.
3. Take a global limiter token, like sign-in. A token is refunded on success. A limiter refusal or a `derive` failure gives the reserved attempt back, and the count never goes below zero, so only checked passwords use attempts.
4. Outside the lock, through `derive`, run `Verify(current)` and, if it succeeds, `Hash(new)`. A `derive` failure gives `dependency_unavailable` (503).
5. Under the domain write lock, check that the session still exists; otherwise return `unauthenticated` (401). Then handle the result:
   - **`ResultLocked`** (the per-login lockout in `modules/passwords`, shared with sign-in): 429 `password_locked`. The reserved attempt is released, so a locked result never counts against the session.
   - **Wrong current password:**
     - 422 `current_password_wrong`;
     - the 5th wrong attempt in one session ends that session (401);
     - wrong attempts through M05 count towards the same per-login lockout as sign-in, so they can lock sign-in for 15 minutes; this is intended.
   - **Form refusal**, or new equal to current: 422 `password_form`. The session's count is reset on a correct current password.
   - **Revalidate access** with `s.access(found)`, which covers suspension, membership and policy age. A failure returns that code.
   - **Generation changed:** 409 `revision_conflict`.
   - **Otherwise:**
     - `Put` the new record;
     - delete the directory's outstanding link entry;
     - bump the generation and `s.counter`;
     - end every other session of the login, dropping their pickups;
     - keep the current session.
     - Return 204 with `no-store`.
6. **Not changed:** the current session's CSRF token and cookie. The current session's `created` time is not reset either; changing a password is not a fresh sign-in under the maximum-age policy.

## `POST /api/v1/me/sessions/end`

**Body:** `{session}`, which is either a handle or `"others"`.

- `"others"` ends every session of the caller's login except the current one.
- A handle ends that one session, only if it belongs to the caller's login and is not the current session.
  - The current session is ended through the existing logout endpoint.
  - Handles are resolved only within the caller's own login.
  - An unknown handle, a foreign handle and the current handle all give the same `invalid_request` (422), so nothing is revealed about other people's sessions.
- The response is 200 `{ended: n}`.

## Audit

These endpoints do not go through the command framework, so they write no command journal records. Each success logs one structured line through the configured logger: the event (`password_changed`, `sessions_ended`), the user ID and the count. Secrets, tokens and handles are never logged. This is a residual, because the pre-launch journal is in memory anyway.

## Excluded, and marked unavailable on M05

- MFA and recovery methods.
- Linking or unlinking other sign-in methods.
- Changing the display name or email.
- Notification preferences.
- A recent access history beyond the session list.

# Review history

- **Design round 2 (FAIL):**
  - R1 (Medium), limited or unavailable calls used attempts: the attempt is given back on those paths, with a test.
  - R2: the count can no longer go below zero.
  - R3: the policy is read from the session's own workspace.
  - R4: an oversized end-sessions body now gives 413.
- **Design round 1 (FAIL):**
  - F1 (Medium), the locked result and the shared lockout: handled with 429 `password_locked`, which is not counted.
  - F2: attempts are reserved before hashing.
  - F3: the map is pruned.
  - F4: a residual.
  - F5: the session and access are rechecked under the lock.
  - F6: the link entry is deleted and the counter bumped.
  - F7: the code is renamed `current_password_wrong`, and the remaining codes are stated.

# Tests and verification

- **Admission:** no cookie gives 401; a bad CSRF token gives 403; a stale policy age gives 401.
- **Read:** the shape, a member versus an administrator, only the caller's own sessions, the handle format, and no verifier, token or CSRF token from other sessions.
- **Password change:**
  - success, after which the other sessions are ended, the current session still works, the old password fails and the new one works;
  - a wrong current password, and the 5th attempt ending the session;
  - a form refusal and new equal to current;
  - a generation race (`pause` hook) and a derive-unavailable fault;
  - body validation and the limiter.
- **Sessions:** "others", a single handle, the current handle refused, and unknown or foreign handles refused.
- **Checks:** race with count=40; linux/386 and Windows builds; 100% coverage of `self.go`; mutation testing of the changed lines; gofmt; vet.

# Residuals for the owner

- No command-journal audit for self-service changes; a log line is written instead.
- A password change does not refresh the maximum-age clock.
- MFA, identity linking, profile changes and notification preferences are unavailable.
- The session list comes from the apps/api session store, which is pruned only by the 12-hour absolute limit. A session that has passed the 30-minute idle limit is still listed, and counted when ended, until it is pruned. Identity is not queried for liveness, because `Access` extends the idle expiry.
- Wrong current passwords sent through M05 share the per-login lockout with sign-in.
- State resets on restart.
