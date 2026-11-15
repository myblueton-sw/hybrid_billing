---
type: VerificationRecord
title: HB-259 passwords module and sign-in API verification
description: Actual checks run for phase 2a of the pre-launch plan (modules/passwords and the apps/api sign-in endpoints), plus review outcomes and residuals.
status: final
---

# Scope

[HB-259](https://linear.app/hybrid-billing/issue/HB-259) adds two parts, following [plan.md](plan.md).

**`modules/passwords`** keeps local password records:
- PBKDF2-HMAC-SHA256 with 600,000 iterations, a 16-byte salt and a 32-byte key;
- derivation that never runs under a lock;
- constant-time comparison, with a dummy record for unknown keys;
- a per-account lockout of 5 failures in 15 minutes;
- one-time set and reset links stored only as digests, redeemed in reserve, hash, commit and release steps, with holds and a 5-minute reservation lapse.

**`apps/api`** adds the sign-in endpoints and seed:
- a pre-session CSRF cookie;
- login in the order password, `OpenSession`, `Access`, `authpolicy.Evaluate`. The `no_policy` exception covers only administrators of the bootstrap workspace. Every refusal ends the session and returns an identical 401.
- a global limiter that takes a token before derivation and refunds it on success, and at most 4 concurrent derivations with a context-aware wait;
- eviction of the user's oldest own sessions at the per-user limit;
- logout;
- link peek and redeem by POST body, and first-administrator activation checked against a server-held verifier with a one-hour expiry;
- a seed with a second sample administrator, the first authentication policy and password-set links in the 0600 file.

Base: main `1fb3dbd`. Machine: mac_mini.

# Checks run (2026-11-16, this machine)

| Check | Result |
| --- | --- |
| `gofmt -l`, `go vet` | clean |
| `go test -race -count=10 -short ./apps/api/... ./modules/passwords` | pass |
| `go test -count=40 -short ./apps/api/server` | pass. A nondeterministic `select` on cancellation was found by this repeat run and fixed |
| `GOOS=linux GOARCH=386 go build` | pass |
| `go test -race -short ./...` | every package passes |
| Statement coverage | `modules/passwords` 100.0%, `apps/api/server` 100.0% |
| PBKDF2 known answer | `02169e67…ffdf08d` for 1,000 iterations, cross-checked with Python's `hashlib.pbkdf2_hmac` |
| Property test (`modules/passwords`) | 80 seeds of puts, verifies, clock moves, issues, reserves, commits and releases against an independent model of failures, locks and link states |
| Timing check (not `-short`) | median login time for an unknown login, a wrong password and a locked account within a factor of 1.5, over 20 runs, at 600,000 iterations |
| Manual start | link peek 200; redeem 204; login 200 in 64 ms at 600,000 iterations; `GET /session` 200 |

# Bugs found and fixed while testing

- Logging in more than 10 times without logging out hit identity's per-user session limit and refused every later login. The user's oldest own sessions are now evicted, up to 20.
- `failCode` had no rule for `dependency_unavailable`. A cancelled request wrote status 0 and became 500. The rule was added, and a test checks that every code `failCode` uses has a rule.
- The cancellation `select` chose at random when both cases were ready. Cancellation is now checked first.

# Mutation testing

| Run | Mutants | Did not build | Survived |
| --- | --- | --- | --- |
| First | 303 | — | 34 |
| Final code | 309 | 11 | 10 |

**Killing the first run's survivors.**
- Literal tests now cover:
  - limiter refund and refill;
  - login and workspace boundaries;
  - TLS `Secure`;
  - duplicate allowed keys;
  - counter increments;
  - email bounds;
  - seed constants;
  - parameter and validity edges.
- Redundant versioning and a lock-time reset were removed.

**After the final run.**
- New tests killed 6 of the 10 survivors, each checked by hand:
  - the retry-millisecond formula;
  - eviction bounded at exactly 20 and choosing the oldest;
  - another user's session never evicted.
- The remaining 4 are equivalent:
  - `MaxAge: -2` on the logout cookie (any negative value deletes it);
  - two length checks after a base64 decode error, which never yields a valid 32-byte value;
  - a login-or-email form branch that ends in the same `password_form` code.

# Reviews

- **Design review** (independent reviewer, security focus, read-only): round 1 FAIL (M1 no-policy deadlock, M2 activation denial of service), round 2 PASS with lows folded in.
- **Change review** (a separate independent reviewer, security focus, read-only): round 1 FAIL (M1 missing no-policy condition tests), round 2 PASS.

Details are in plan.md.

# Residuals

- The sign-in screens (HB-260) and the session gate in front of the Python endpoints (HB-261).
- Administrator-issued resets and ending a user's sessions on reset (phase 3).
- MFA, SSO, LDAP; Unicode normalization of passwords and breached-password checks.
- The lockout and limiter values are provisional. A local process can lock a known login for 15-minute periods.
- With `--seed=none`, the policy needs a second administrator, who comes in phase 3.
