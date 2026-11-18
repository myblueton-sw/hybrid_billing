---
type: VerificationRecord
title: HB-263 verification of the user administration API for M01
description: Tests, coverage, mutation and build results for the identity read, the twelve identity commands, link pickup, invitation acceptance, reset handling and the seed changes in apps/api. Run on mac_mini on 2026-11-16.
status: final
---

# Scope

This record verifies [HB-263](https://linear.app/hybrid-billing/issue/HB-263) against its [plan](plan.md). The design review passed in round 4. Every result below was run on mac_mini on 2026-11-16 with Go 1.25.7.

# Changes

**New files in `apps/api/server`**
- `directory.go`: people, taken logins, grant holders, pending invitations, outstanding links, reset generations and pickups.
- `admin.go`: the twelve `identity.*` command definitions, the protection rules, the error mapping and the caller-verifier use.
- `users.go`: `GET /api/v1/identity/users`, `GET /api/v1/identity/links/{ref}`, session ending and invitation acceptance.

**Changes to existing files**
- `commands.go`: `s.caller` is set and cleared around `Execute`.
- `signin.go`:
  - sign-in compares the reset generation first, then accepts any pending invitation before `Access`;
  - `redeem` rechecks reset links, ends sessions on reset, and accepts invitations on a set link;
  - activation grants the installation action.
- `seed.go`: directory entries, `sample-member` with an invitation and a set link, and `grantInstallation`.
- `session.go`: `byLogin`.
- `server.go`: the directory, the caller field, and the registration of the commands.
- `boundary.go`: the two new routes.

## Departures from the plan

- **Pickups targeting the reset user are not deleted at reset.** Such a pickup can only exist for a link issued after the one being redeemed, and that newer link already invalidates the reset token. The reset still ends every session of the user and the pickups those sessions hold. Mutation testing showed the removed step was unreachable.
- **Membership checks in commands.** Commands take the membership version from the roster, and a missing membership is refused by identity (`record_unknown_here`, mapped to `invalid_stage`). There is no separate apps/api check for it.
- **Probe sessions.** The reset recheck and the acceptance at a set link both open a short probe session for the user.
  - When the user is at identity's session limit, the probe evicts the oldest apps/api sessions of that login, as sign-in does.
  - A reset redeem fails closed (`link_invalid`) only when no probe can be opened: the user was suspended after the link was issued, or the limit is filled by identity sessions outside apps/api.
  - Reset links are refused for users who are not active.
  - If the probe fails on the set path, the pending invitation is kept, and the next sign-in accepts it.
- **A failed link reissue.** If `passwords.Issue` fails, it has already dropped the old link. The handler then deletes the directory's link record and the user's pickups, and returns a plain error, so the executor stores `internal_error` and bumps the counter.
- **Grant failure at activation.** If the grant fails during `--seed=none` activation, the failure is logged without secrets, and activation still completes without a grant holder.

# Results

| Check | Result |
| --- | --- |
| `go test -race ./...` in `apps/api` | pass |
| `-race -count=40` on the HB-263 tests | pass |
| `go vet`, `gofmt -l` | clean |
| `GOOS=linux GOARCH=386` and `GOOS=windows` builds | pass |
| Coverage of `apps/api/server` | 100.0% of statements, re-run after the change-review fixes |
| Largest file | `users_test.go`, 667 lines |

## What the tests cover

Tests are in `users_test.go`, `users_edges_test.go` and `users_property_test.go`.

**Permissions** (read, pickup and every command)
- `sample-admin-2`, an administrator without a grant, is refused with `permission_denied`.
- A member is refused, a request with no session gets `unauthenticated`, and a bad CSRF token gets `csrf`.

**Protection**
- Self-targeting is refused for every guarded command, including while another grant holder exists.
- The `protect` unit cases cover a member, a suspended administrator membership and a suspended administrator user, none of which count as active administrators. When only non-holders remain, the result is `prerequisite_failed`.
- Reset links for administrators are refused, and so is promotion while a link is outstanding.
- A reset redeem is refused when the user has been promoted in the meantime.
- An expired grant holder is no longer shown as one.

**Commands**
- The happy paths, including `business_state` values and counter bumps.
- Request envelope:
  - a stale revision gives `revision_conflict`;
  - a rationale over 256 bytes is refused, including on a workspace command;
  - a missing rationale or role is refused;
  - an unknown target gives `permission_denied`.
- Login rules:
  - taken, seeded, departed, padded or invalid logins are refused;
  - a login that identity already linked outside the directory gives `prerequisite_failed`; if that user departed, `invalid_stage`.
- Each row of the error mapping, through real identity refusals.

**Partial `create_user`**
- An injected failure at the invite step and at the link step gives `internal_error` with an operation and a counter bump.
- The partial row is visible, and recovery works through `invite` and `issue_set_link`.
- The capacity preflight answers `dependency_unavailable`.
- The identity invitation limit on `invite` answers `dependency_unavailable`. The password-link limit on `issue_set_link` and `issue_reset_link` answers `internal_error` with an operation, because the old link is already gone (see the departures from the plan).

**Links and pickups**
- A pickup works once, only for the issuing session, and needs CSRF.
- It answers with `no-store`.
- It expires at 10 minutes, and is deleted at sign-out, by pruning and on reissue for the same user. Another user's pickup is kept.
- A replay returns the same `link_ref`, which is already used.
- The command result and the journal contain no `activate#` string.

**Invitations**
- Acceptance on a first password set.
- Acceptance at sign-in after `identity.invite`.
- A revoked invitation, or one whose inviter changed, leaves the user without a membership.
- With no policy, the member's acceptance persists while the sign-in is refused, and the counter is bumped.
- Invitations to members, suspended members, departed users, or users with a pending invitation are refused.

**Reset**
- The user's sessions end, and their pickups are deleted; other pickups are kept.
- The reset generation is bumped, and a sign-in captured before the reset is refused.
- At the per-user session limit, a reset evicts the user's apps/api sessions and succeeds. It fails closed only when the limit is filled by identity sessions outside apps/api, or when the user was suspended after the link was issued.

**`--seed=none`**
- Activation grants the action, and the bootstrap user can read the user list.

**The read**
- The key allowlist holds, service users are skipped, and the cursor pages through 108 users at limits of 1 and 100.
- Rows beyond the first identity page are found, and commands work on them.
- Expired invitations and links are hidden.
- The read answers `authority_unavailable` and `unauthenticated` when admission fails, and `dependency_unavailable` when identity fails (through a fault hook).

**Concurrency**
- Concurrent reads and commands pass under `-race`.
- The caller verifier is empty after every call.

**OpenAPI** lists all twelve commands.

**Model test.** `TestUserAdministrationMatchesAModel` runs 4 seeds × 120 random steps. Each step is one of suspend, reinstate, depart, role change, membership suspend, reinstate or remove, or invite, against five users including the caller. It checks:
- the exact result code;
- each user's row (state, membership, role, pending invitation, login presence);
- that the grant holder stays an active holder after every step.

Every outcome class is reached at least 5 times per seed.

# Mutation testing

The changed lines of the seven touched files were mutated: comparison and boolean swaps, error-code swaps, purpose and role swaps, `++`/`--`, `true`/`false`, removed negations and disabled `if` conditions. The script is `scratchpad/mutate263.py`, run in four copies.

**First run: 307 of 341 killed.** The 34 survivors were resolved in two ways.
- **Code removed as redundant:**
  - the self-check in reset links (the actor is always an administrator);
  - the self-check in promotion (identity refuses the same role);
  - the membership helper (identity refuses an unknown membership);
  - `!c.known` in invitation revocation;
  - the `!known`/departed checks for reset links (a departed or unknown user has no password);
  - the reset-time pickup drop for the target (unreachable, see the departures from the plan);
  - `directory.pending` (a pickup implies a live link).
- **Tests added:**
  - a long rationale on a workspace command;
  - membership beyond the first identity page;
  - the `protect` unit cases with the exact code;
  - self-suspension with a second holder;
  - other users' pickups surviving a reissue or reset;
  - set and reset links outside the directory;
  - the purpose of a set link;
  - a missing role;
  - expired invitations and links;
  - explicit capacity limits;
  - an expired holder.

**Second run: 314 of 321 killed.** Then the reset-link precondition was simplified, which removed two more mutants. The 5 survivors that remain are equivalent:

| Location | Mutant | Why it is equivalent |
| --- | --- | --- |
| `admin.go`, `Apply` | `!h.create && err == nil` becomes `\|\|` | For creation, looking up the target `users` finds nothing. For other commands, the mutant would let `findUser` overwrite an error from `roster`. That error cannot occur, because `roster` runs under the same lock right after the gate admitted the caller as an active administrator. |
| `admin.go`, `findUser` | Returns `true` alongside an error | Every caller checks the error first. |
| `users.go`, `pageLimit` | The `Atoi` error check disabled | A parse error yields 0, which fails `n < 1`. |
| `users.go`, `probe` | The eviction bound `<` becomes `<=` or `--`, or the `!ok` break is disabled | Each iteration evicts one apps/api session of the login, and the loop stops when none is left or when a session opens. `maxEvictions` is a defensive bound, as in sign-in. Without the break, the result is still `ErrLimit` after the bounded retries. |

**Third run, after the change-review fixes: 316 of 324 killed.**
- Two survivors were resolved:
  - The probe's pickup drop was removed: a probe evicts only sessions of a non-administrator, which never hold pickups.
  - A test now kills the error-path pickup drop: another user's pickup is kept and the target's is dropped.
- The other survivors are the equivalent mutants in the table above.

# Change review

**Round 2: PASS.** Two Low items were fixed:
- L-a: two stale statements in this record are corrected;
- L-b: the concurrent test no longer calls helpers that can call `Fatal` from goroutines.

**Round 1: FAIL.**

| Finding | Severity | Resolution |
| --- | --- | --- |
| M1: a reset could be blocked by keeping the user's sessions at the limit | Medium | The probe now evicts like sign-in. The test shows a reset succeeding at the limit and ending every session. |
| L1: a failed reissue left a stale link record and no counter bump | Low | The record is cleared, and a plain error bumps the counter. |
| L2: the self-promotion code | Low | `protect` runs first for every role change; tested after the caller's password was set. |
| L3: two equivalence claims were wrong | Low | The `Apply` claim is restated. The `Required` mutants are now killed: `TestOperatorAndContract` checks the OpenAPI `required` list. |
| L4: missing tests | Low | Added: the operator mapping; concurrent redeem, revoke, reissue and read over HTTP; concurrent `listUsers` under `-race`; a stale sign-in opens no session and accepts no invitation; a raw-token scan of the result, journal, read and logs. |
| L5: smaller gaps | Low | Probe and activation behaviour are stated above; a departed holder is removed; `boundary.go` is listed. |

# Not verified

- The browser screen: HB-264.
- Real email or message delivery: not permitted.
- Persistence: all state resets on restart, as declared.

# Planning cross-check (HYB-291)

Added by the [HYB-291 planning cross-check](../../reviews/HYB-291-planning-cross-check/report.md) on 2026-11-17. A record added by HYB-291; it changes no result above except where marked. The findings come from a read-only auditor; the HYB-291 writer agent (delegated by root) wrote this trace, the residual notes and the copy changes; the HYB-291 change review is their first independent check.

## Requirement trace

Source: `docs/ui/screens/M01/spec.md:118-120` (HB-263 is the API, HB-264 the screen). Status meanings are in the [cross-check report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). No acceptance test was run.

| ID | Status | Reason | Record |
| --- | --- | --- | --- |
| HB-03 | deferred | Addressed within pre-launch scope for twelve identity commands under installation administration with last-administrator protection; registry-wide and leakage acceptance not built. | HB-263/plan.md |
| HB-34 | deferred | Bulk revocation needs asynchronous jobs and manifests and is not built. | HB-263/plan.md:224, HB-264/plan.md:61 |
| HB-42 | deferred | Invitation, set and reset links, suspend, depart and role change are built; role validity periods and bulk revocation are not. | Planning cross-check residuals below |
| IAM-01 | deferred | Covered: invitations and set links are one-time. Not built or tested: concurrent first-administrator creation and invitation acceptance from another workspace. 2026-11-18: see [HYB-303/verification.md](../HYB-303/verification.md); `retention-access.md:49` not fully covered (restart re-arming, HYB-303 R1). | HB-263/verification.md |
| IAM-02 | deferred | Only local password accounts exist; SAML, OIDC and LDAP sign-in are residuals (HB-257 out of scope). | HB-265/plan.md:177 |
| IAM-03 | deferred | Does not arise while only local accounts exist. | this section |
| IAM-04 | deferred | Federated token checks are not built. | HB-265/plan.md:177 |
| IAM-05 | deferred | Suspend and depart are built in the identity module; effects on keys, jobs and files and detection delay are not measured. | HB-263/plan.md |
| IAM-06 | deferred | Reset links cannot be issued for administrators and the last administrator is protected; MFA recovery and identity unlinking are not built. | HB-263/plan.md |
| IAM-07 | deferred | The read only pages (`HB-263/plan.md:105`); server search and the workspace, status, auth method, role and expiry filters (`M01/spec.md:61-63`) are missing; fix ticket HYB-294. | HYB-291 report |
| IAM-08 | deferred | No reauthentication before suspending a user (`M01/spec.md:73`); step-up applies only to proposing and approving authentication policy changes; fix ticket HYB-293. | HYB-291 report |

## Planning cross-check residuals (HYB-291)

- **Role validity periods.** `M01/spec.md:52` gives each role binding a validity interval; roles have no validity period.
- **Bulk revocation.** Not built (`HB-263/plan.md:224`); it needs asynchronous jobs and manifests.
- **Search and filters.** Missing; fix ticket HYB-294. **Reauthentication before suspend.** Missing; fix ticket HYB-293.
