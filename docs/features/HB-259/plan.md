---
type: ImplementationPlan
title: Passwords module with PBKDF2 records, constant-time verification, per-account lockout and digest-only one-time links, and the pre-launch sign-in API for login, logout and link redemption
description: Phase 2a of the pre-launch plan (HB-257). A new standard-library module keeps local password records and one-time set and reset links. The API service adds a pre-session CSRF token, login through the password, the authentication policy and an identity session, logout, and redemption of password links and of the first-administrator activation link, with a global failure limiter. The sample seed adds a second administrator, the first policy revision and password-set links.
status: stable
---

# Scope and ownership

[HB-259](https://linear.app/hybrid-billing/issue/HB-259) is the first part of phase 2 of the [pre-launch plan](../HB-257/plan.md). Phase 2 is split into three tickets:
- HB-259: this backend;
- HB-260: the L01–L03 screens;
- HB-261: the session gate in front of the Python endpoints.

This ticket adds `modules/passwords` and the sign-in endpoints in `apps/api`. Administrator-issued resets need the installation administration action, so they come with M01 in phase 3. L03 here can only redeem a reset link.

Base: main `1fb3dbd`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the design and the change with a security focus. Machine: mac_mini.

Allowed files: `modules/passwords/*`, `apps/api/**` and `docs/features/HB-259/*`.

# Observed baseline

- `apps/api` (HB-258) provides:
  - the request boundary;
  - the domain lock and counter;
  - session records with a cookie and CSRF token: `Server.OpenSession(triple, workspace, secure)` reserves a slot, then opens the identity session;
  - the seed driver: with the sample seed it consumes the bootstrap for `sample-admin`, whose triple is `local-password` / `prelaunch` / `admin`.
- `identity`:
  - `Boundary().OpenSession(triple, verifier, idle, absolute)` requires a linked identity;
  - `EndSession(verifier)` ends a session;
  - `Installation().CreateUser(operator, user, triple, email)` creates a user with a linked identity;
  - `Bootstrap` creates the first administrator from the bootstrap verifier.
- `authpolicy.Evaluate(principal, authentication, action)` refuses with `no_policy` until the workspace has revision 1. Revisions are proposed and approved by two administrators of the workspace, each authenticated against the current policy, and before revision 1 by any two administrators.
- The Go standard library has `crypto/pbkdf2` (Go 1.24 and later). The repository builds with Go 1.25.

# Decision

## `modules/passwords`

Standard library only, one mutex, no imports of other modules. A record is keyed by a `Key{Issuer, Subject}`. The caller uses the installation as the issuer and the login ID as the subject, the same values as the identity triple with connection `local-password`.

**Parameters**, fixed for pre-launch and provisional. `New(clock, Params)` validates them:
- iterations: 600,000;
- salt: 16 bytes;
- derived key: 32 bytes;
- password length: 12 to 256 bytes;
- failed attempts before lockout: 5;
- lockout: 15 minutes;
- link validity: 1 hour to 7 days, set per link.

**Passwords.**
- A password must be valid UTF-8 within the length bounds. No composition rules apply, and it must not equal the subject.
- Passwords are compared as bytes; Unicode normalization is a residual.
- `Hash(key, password)` returns a `Record{Iterations, Salt, Hash}`. It is pure and takes no lock, so the costly derivation never runs under a lock.
- `Put(key, record)`:
  - stores the record, replacing any earlier one;
  - clears the failure count;
  - invalidates every outstanding link for the key.

**Verification.**
- `Verify(key, password)` reads the record, its version and the lock state under the mutex, releases it, derives the key, then re-locks to record the outcome.
- If the record's version changed meanwhile (a concurrent `Put`), the result is `wrong` and nothing is recorded.
- The lock state is checked again when recording.
- For an unknown key, or a key that is locked out, it derives against a fixed dummy record, so the time spent does not reveal the case.
- The comparison is constant-time (`subtle.ConstantTimeCompare`).
- A failure increments the key's count.
  - The window starts at the first counted failure and lasts 15 minutes. A failure after the window restarts the count at one.
  - The fifth failure within the window locks the key for 15 minutes.
  - Unknown keys keep no failure state, so memory is bounded by the records.
- Success resets the count.
- The result is `ok`, `wrong` (also used for unknown keys) or `locked`. A key's lock state is never revealed for an unknown key.

**One-time links.**
- `Issue(key, purpose, validity)` returns a 32-byte random token, encoded as unpadded base64url. The purpose is `set` or `reset`.
  - Only the SHA-256 of the token is stored, with the key, the purpose and the expiry.
  - Issuing a new link for a key invalidates the previous one.
- Redemption is two steps, so no derivation runs under any lock:
  - `Reserve(token)` finds the link by digest, refuses it if it is expired, used or reserved, marks it reserved, and returns the key and purpose. Two concurrent redemptions therefore cannot both proceed.
  - The caller then hashes with `Hash(key, password)` outside every lock. That call checks the password form, including `same_as_login`.
  - `Commit(token, record)` stores the record, marks the link used and invalidates the key's other links. If the key's link was replaced or removed in the meantime, the commit is refused.
  - `Release(token)` returns a reserved link to valid when hashing or committing fails, so a mistyped password does not burn the link. The endpoint releases in a deferred call, and a reservation also lapses on its own after 5 minutes, so a panic or a cancelled request never burns a link.
  - `Commit` has `Put` semantics: it bumps the record version and clears the failure count. A reserved link peeks as `link_invalid`.

  An invalid, used, reserved or expired token gives the same refusal, `link_invalid`.
- `Peek(token)` returns the key and purpose of a valid link without using it, so the activation screen can show which login it sets.

**Bounds.** 100,000 records and 10,000 outstanding links (`ErrLimit`).

**Errors.** `ErrInvalid` (form), `ErrLimit`, and `ErrRejected` wrapped in a refusal with one of the reasons `link_invalid`, `password_form` or `same_as_login`.

## Sign-in endpoints in `apps/api`

All of these are on `/api/v1/auth/` and pass the HB-258 boundary: Host, `Sec-Fetch-Site`, and Origin for every POST.

**`GET /api/v1/auth/csrf`.**
- It requires the header `X-HB-Client: console`.
- It sets a short-lived cookie `hb_presession` holding 32 random bytes (`HttpOnly`, `SameSite=Strict`, `Path=/api/v1/auth`, ten minutes, `Secure` over TLS) and returns the same value in the body.
- Every unauthenticated POST below must echo it in `X-CSRF-Token`, which is compared in constant time against the cookie (`403 csrf_refused`).

**`POST /api/v1/auth/login`** with `{workspace, login, password}`:
1. **Limits.**
   - A global limiter allows 30 unauthenticated attempts per minute, with a burst of 30. It covers login, link peek and redeem. A token is taken before any derivation and refunded on success.
   - When the limiter is empty, the call is `429 rate_limited` with `Retry-After`.
   - At most 4 derivations run at a time. Further calls wait for a slot while the request is still live (the wait selects on the request's context).
   - A correct password followed by a policy or membership refusal is a failure and does not refund the token.
   - Client addresses are not used, because through the proxy every request is loopback.
2. **Form.** The login must be valid UTF-8 of 3 to 200 bytes (the password subject limit) with no control characters. Only ASCII letters are lower-cased before use, in login, in activation, and everywhere a password key or identity triple subject is built. The workspace must be an identifier.
3. **Password check.** `passwords.Verify` runs without the domain lock.
4. **Session.** On `ok`, under the domain lock (a domain change), in this order:
   1. `OpenSession` for the triple, then `Access` for the workspace. Access gives the role.
   2. `authpolicy.Evaluate` with the principal from that answer (user, workspace, role, kind `human`) and the authentication `password` at the current time with no factors.
   3. The refusal `no_policy` is accepted only when all three hold:
      - the workspace is the bootstrap workspace;
      - the answer's role is `administrator`;
      - the reason is exactly `no_policy`.

      This matches authpolicy's rule that any two administrators may create revision 1, so a second administrator added later can sign in and approve it. The server records the bootstrap workspace at start.
   5. On any refusal after `OpenSession` (Access or policy), the identity session is ended and the server's record dropped, before the failure response.
   6. On success it sets `hb_session` (`Secure` when the request is TLS) and returns `{user, workspace, role, csrf_token}`.
4. **Failures.** `wrong`, `locked`, an unknown workspace or membership, and a policy refusal all return the same `401 unauthenticated`, with the same body, after the same derivation.
5. **Pruning.** Session records past their absolute lifetime are pruned on each login.

**`POST /api/v1/auth/logout`** needs the session cookie and its CSRF token. It ends the identity session, drops the record and clears the cookie.
- Without a session, it still returns 204.
- With a session cookie but a wrong or missing CSRF token, it returns `403 csrf_refused`.

**`POST /api/v1/auth/link`** with `{token}` in the body, so the token never appears in a URL. It needs the pre-session CSRF token. It returns the link's purpose and login for a valid password link, or `{purpose: "activate"}` for a valid first-administrator activation. Any other case returns `422 link_invalid` (the same code and status as redeem), never more detail.

**`POST /api/v1/auth/redeem`** with `{token, password}`, plus `{login, email}` for an activation:
It needs the pre-session CSRF token and takes a limiter token.

- **Password link.**
  - `Reserve` runs under the domain lock (read only).
  - `Hash` runs outside every lock.
  - `Commit` runs under the domain lock (a domain change).
  - `Release` runs on any failure.
- **Activation link** (`--seed=none` only).
  - The server keeps the bootstrap verifier and its expiry, one hour after start. Both peek and redeem compare SHA-256(token) with it in constant time first. They refuse, without hashing and without calling `identity.Bootstrap`, unless it matches and has not expired. Wrong tokens therefore never reach the identity attempt counter, so they cannot lock the bootstrap.
  - Only after a match is the password hashed outside the locks. Then, under the domain lock, `identity.Bootstrap` creates the user `admin` with the triple `local-password` / `prelaunch` / `<login>` and the given email, and the password is stored for the same key.
  - The activation file is deleted after success, and the server forgets the verifier.
- The response never opens a session. The user then signs in.
- Every failure is `422 link_invalid` or `422 password_form`.

## Seed

The sample seed adds these steps after the bootstrap:
The bootstrap administrator's triple subject (`AdminTriple`) becomes its login `admin@example.invalid` at bootstrap, replacing `admin`. Logins are e-mail-shaped in the sample.

1. `sample-admin-2`, a second administrator with login `admin2@example.invalid`, created with `CreateUser` and added to the workspace through an invitation that the seed accepts as that user. The invitation needs sessions, so the seed opens and ends internal sessions without cookies for these two calls.
2. The first `authpolicy` revision for `sample-workspace`:
   - methods `password`;
   - MFA `none`;
   - maximum age 12 hours;
   - no step-up.

   `sample-admin` proposes it and `sample-admin-2` approves it, each with a synthetic password authentication at the seed time.
3. A one-time `set` link, valid for 7 days, for each sample login: `admin@example.invalid` and `admin2@example.invalid`. The links file is not deleted, because each link is single use.

The links are written, one per line as `<login> <console origin>/activate#<token>`, to `activation.txt` with the exclusive 0600 create. Only the file's path is logged.

The links go through `/activate` (the L02 screen, HB-260). The token is in the fragment, so it reaches the server only in the bodies of the peek and redeem calls, never in a URL.

## Code layout

| Package | Files |
| --- | --- |
| `modules/passwords` | `types.go`, `registry.go`, `links.go` and tests |
| `apps/api/server` | `signin.go` (the endpoints and the limiter), changes to `seed.go` and `session.go` (pruning, created time), and tests |

# Residuals

- Administrator-issued resets (phase 3, M01). (2026-11-18: built by HB-263 and HB-264, PRs #183/#184.)
- MFA, SSO and LDAP.
- Password Unicode normalization and breached-password checks.
- The limiter and lockout values, which are provisional. A local process can lock a known login for 15-minute periods, or exhaust the global limiter, and so deny sign-in. This is accepted for a loopback pre-launch build; an unlock path comes with administrator resets in phase 3.
- A password reset through a link does not end the user's existing sessions. Ending them comes with phase 3.
- With `--seed=none`, the policy cannot be created until a second administrator exists (phase 3).

# Verification plan

**Literal tests for `modules/passwords`:**
- every parameter, form and bound;
- `Hash` and `Verify` with a known-answer value cross-checked with Python's `hashlib.pbkdf2_hmac`;
- the dummy path for unknown and locked keys;
- lockout at the fifth failure and expiry after 15 minutes;
- success resetting the count;
- links: issue, peek, redeem, reuse, expiry, invalidation by a new link or by `Put`, and concurrent redemption, where exactly one succeeds;
- `same_as_login`.

**Literal tests for the round 1 fixes:**
- six wrong activation tokens followed by the correct one still activate;
- activation expires after 1 hour;
- the `no_policy` exception is refused for a member and for a non-bootstrap workspace;
- an `Access` refusal cleans up the session;
- the limiter takes, refunds and returns 429, including for peek and redeem;
- the 4-slot semaphore, including a cancelled wait;
- a version change during `Verify` returns `wrong`;
- `Reserve`, `Release` and `Commit`, including a commit refused after the link was replaced, and a reservation lapsing;
- login form rules and lower-casing, also in activation;
- logout with a wrong CSRF token returns 403;
- the POST peek.

**Literal tests for the endpoints:**
- the pre-session CSRF cookie and header rules;
- login success with the cookie and body;
- every failure returning the identical body;
- the global limiter;
- the no-policy rule for the bootstrap administrator;
- a policy refusal ending the session;
- logout;
- link peek and redeem for both kinds;
- the activation file deleted after activation;
- the seed creating the second administrator, the policy and the links file.

**A timing check**, statistical and lenient: the median time of a login with an unknown login, with a wrong password and with a locked account stays within a factor of 1.5 of each other over 20 runs. It is skipped with `-short`.

**Property test:** random sequences of set, verify, lock and link operations against an independent model of counts, locks and link states.

**Checks:**
- `-race -count=10`;
- the 386 build;
- gofmt and vet;
- `go test -race ./...`;
- 100% coverage of `modules/passwords` and `apps/api/server`;
- mutation testing of both.

# Review history

- Round 1 (independent security reviewer, read-only): FAIL. Findings and what changed:
  - M1. The `no_policy` sign-in exception is limited to administrators of the bootstrap workspace and to that exact reason, so a second administrator can approve revision 1.
  - M2. The server keeps the activation verifier and expiry and checks a token in constant time before any hashing or `Bootstrap` call. Redeem and peek are limited.
  - L1. Every refusal after `OpenSession` ends the identity session and drops the record.
  - L2. A limiter token is taken before deriving; derivations are limited to 4 at a time; the lock is re-checked; the window is defined; the denial-of-service trade-off is a residual.
  - L3. Redemption is split into reserve, hash, commit and release.
  - L4. The link peek is a POST with the token in the body.
  - L5. Records are versioned for `Verify`; ending sessions on reset is a residual.
  - L6. `AdminTriple` uses the e-mail-shaped login.
  - L7. Login form rules, `Secure` on the pre-session cookie, 403 for logout with a wrong CSRF token.
- Round 2 (same reviewer): PASS. Low items folded in:
  - L-a. Tests named for each round 1 fix.
  - L-b. Login normalization everywhere.
  - L-c. Deferred release, a 5-minute reservation lapse, and `Commit` with `Put` semantics.
  - L-d. Text fixed; no refund after a correct password.
  - L-e. A context-aware semaphore wait.

# Change review resolution

- Change review round 1 (separate independent reviewer, security focus, read-only): FAIL. Findings and what changed:
  - M1. Tests now cover the other two `no_policy` conditions:
    - a non-administrator member of the bootstrap workspace without a policy is refused, with no session left;
    - an administrator refused by a real policy (MFA required) is refused, with no session left.
  - L1. The plan states 422 for peek, matching redeem.
  - L2. ASCII-only lower-casing; logins are bounded at 200 bytes to match the password subject limit.
  - L3. `seedLinks` evaluates `run` before returning the lines.
  - L4. `Release` takes the reservation's hold time, so a lapsed reservation cannot clear a newer one. Expired and used links are purged when the link cap is reached.
  - L5. Eviction at the per-user limit retries with successively older own sessions, up to 20.
  - L6. The timing check uses 20 runs and keeps the wrong-password series unlocked. The identical-failure check compares whole bodies and headers, including a locked account.
- Mutation survivors were killed with literal tests:
  - limiter refund and refill;
  - login and workspace boundaries;
  - TLS `Secure`;
  - a duplicate allowed key;
  - counter increments;
  - email bounds;
  - seed constants;
  - another user's session kept at eviction;
  - parameter and validity edges.

  Redundant record versioning and the reset after a lock were removed; pointer identity and the window already cover them.

