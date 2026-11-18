---
type: VerificationRecord
title: HB-259 passwords module and sign-in API verification
description: Actual checks run for phase 2a of the pre-launch plan (modules/passwords and the apps/api sign-in endpoints), plus review outcomes and residuals.
status: stable
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
- Administrator-issued resets and ending a user's sessions on reset (phase 3). (2026-11-18: both built by HB-263 and HB-264, PRs #183/#184; a redeemed reset ends the user's sessions, HB-264/plan.md:87.)
- MFA, SSO, LDAP; Unicode normalization of passwords and breached-password checks.
- The lockout and limiter values are provisional. A local process can lock a known login for 15-minute periods.
- With `--seed=none`, the policy needs a second administrator, who comes in phase 3.

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
| IAM-01 | deferred | Covered: activation and set links are one-time and invalidated on use. Not built or tested: concurrent first-administrator creation and acceptance from another workspace. 2026-11-18: see [HYB-303/verification.md](../HYB-303/verification.md); `retention-access.md:49` not fully covered (restart re-arming, HYB-303 R1). | HB-259/verification.md |
| IAM-02 | deferred | Only local password accounts exist; SAML, OIDC and LDAP sign-in are residuals (HB-257 out of scope). | HB-260/plan.md:132-146 |
| IAM-03 | deferred | Does not arise while only local accounts exist; applies when another identity provider is connected. First recorded here. | this section |
| IAM-04 | deferred | Local link expiry, one-time use and an identical 401 are built; issuer, audience, signature, nonce, SAML replay and LDAP checks wait for federated sign-in. | HB-260/plan.md:132-146 |
| IAM-05 | deferred | Suspend and depart belong to M01 (HB-263); effects on keys, jobs and files and the external detection delay are not measured. | HB-263/plan.md |
| IAM-06 | deferred | Recovery is administrator-issued only (HB-263) with last-administrator protection; MFA recovery and identity unlinking are not built. | HB-259/verification.md Residuals |
| IAM-07 | deferred | No 100,000-user test; M01 server search and filters are HYB-294. | HYB-291 report |
