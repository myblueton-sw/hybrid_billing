---
type: ImplementationPlan
title: User administration API for M01 on the commands path, gated by the installation administration action
description: Phase 3b of the pre-launch plan (HB-257). apps/api gains a user list read and identity administration commands. User creation brings a workspace invitation, accepted at the first password set. Password-set and reset links are issued by administrators and picked up once outside command results.
status: stable
---

# Scope and ownership

[HB-263](https://linear.app/hybrid-billing/issue/HB-263) builds the API behind M01 (users, invitations and revocation). The M01 screen follows in HB-264. Base: main `1403f12`, after [HB-262](../HB-262/plan.md).

Root (Claude Code) owns the plan, implementation and tests. A separate read-only reviewer reviews the design before code and the change after it, with a security focus, because this changes permissions and identity state. Machine: mac_mini.

Allowed files:
- `apps/api/server/*`, `apps/api/cmd/hb-api/*`;
- `docs/features/HB-263/*`.

No module changes. If a module change proves necessary, it becomes a separate ticket.

# Observed baseline

**apps/api (HB-258–HB-262)**
- Session cookie; `POST /api/v1/commands/{command}` runs `commands.Execute` under the installation-wide domain write lock, and each applied mutating command increments the domain counter.
- No production command definitions are registered yet.
- Sign-in endpoints: `/api/v1/auth/{csrf,login,logout,link,redeem}`.
- `redeem` either activates the bootstrap administrator or sets a password from a `modules/passwords` link.
- The seed creates `sample-admin` (bootstrap administrator) and `sample-admin-2` (administrator).
- The installation workspace is set, and the sample seed grants `administer_installation` to `sample-admin` and the console client.

**modules/identity**
- `Installation`, keyed by a free-text `Operator`:
  - `CreateUser`, `SuspendUser`, `ReinstateUser` and `DepartUser`, the last three with an expected epoch or version;
  - `Users(after, limit)` returns `UserInfo` (ID, kind, state, epoch, workspace).
- `Workspace`, authorized by the caller's session verifier, which must be a workspace administrator:
  - `Invite`, `RevokeInvitation` and `Accept`;
  - `ChangeRole`, `SuspendMembership`, `ReinstateMembership` and `RemoveMembership`, each with the membership version;
  - `Members(actor, workspace, after, limit)` returns `MemberInfo` (user, role, state, version, user state).
- The last administrator is protected (`ErrLastAdministrator`).
- Identity has no list of invitations and does not return a user's login ID or email.

**modules/passwords**
- `Issue(key, purpose, validity)` replaces any earlier link for the key; purposes are `set` and `reset`.

**modules/commands**
- A definition carries one gate permission. `Resolve(workspace, target)` returns a revision, and a mutating call must send the matching expected revision.
- Results and outcome fields are returned to the caller and kept in the idempotency store for replay. The journal keeps no outcome fields.

# Decision

## Permission and grant holders

Every M01 command and read uses `administer_installation` with the kind `installation`. The gate therefore requires a confirmed human administrator Answer in the installation workspace, a direct grant to that user, and the console client's grant.

Workspace operations that identity authorizes by verifier are checked a second time by identity, with the caller's own session verifier. That is the same session that produced the Answer.

**Grant holders.** apps/api records the holders of the installation grant it created:
- With the sample seed, the holder is `sample-admin`, as in HB-262.
- With `--seed=none`, bootstrap activation now grants the action, with the HB-262 terms, to the bootstrap user and the console client. It does this in the same domain write lock as `Bootstrap`. This closes the hand-off in HB-262.

The grants are valid for 30 days from the start of the process. An expired grant does not count as a holder in protection rule 3.

No command in this ticket grants or revokes the action. That is a residual: an explicit, separately reviewed grant command.

There is one workspace in pre-launch, so the installation administrator and the workspace administrator coincide. A separate workspace-administration action is a residual for a multi-workspace build.

## Protecting administration

apps/api checks these rules under the domain write lock, before calling identity. A refusal is `prerequisite_failed`.
1. **No self-targeting.** A command whose target is the caller is refused. This covers suspend, depart, role change, membership suspension and removal, and reset links.
2. **Last active administrator.** A command is refused when it would leave the installation workspace without an active administrator: an active user with an active administrator membership. This covers `suspend_user`, `depart_user`, a demotion, `suspend_membership` and `remove_membership`. It closes the gap in `SuspendUser`, which identity does not check.
3. **Last grant holder.** A command is refused when it would leave no grant holder who is an active user with an active administrator membership.
4. **Reset links for administrators.** A reset link is refused for any user with an administrator membership.
   - `redeem` checks the rule again: a reset link whose user now has an administrator membership is refused as `link_invalid`.
   - `change_role` to administrator is refused while the target has an unredeemed link or an unused pickup.
   - `modules/passwords` cannot report live links. The directory therefore tracks, per login, the outstanding link's purpose and expiry. The entry is cleared on `Commit`, on a reissue that replaces the link, on depart, and on expiry.
   - Administrator password recovery is a residual and the owner's decision.

**Trust boundary.**

The installation administrator issues every password-set and reset link and hands it over out of band. They can therefore take over any account other than their own:
- every first password, including that of a user created or invited as an administrator;
- any existing administrator, in several steps:
  1. demote the administrator, or remove their membership;
  2. issue a reset link;
  3. redeem it with a password they choose;
  4. promote the user back, or re-invite them as an administrator.

The rules above only remove the one-step routes: a reset link for a current administrator, and promotion while a link is outstanding. The journal records every step with the target. As a consequence, two-person approval of the authentication policy does not hold against the installation administrator.

This is declared as a residual for the owner's decision. A delivery channel the administrator cannot read needs a message permission that pre-launch does not have.

## Caller verifier

Identity's workspace operations need the caller's session verifier:
- The command endpoint holds the domain write lock for the whole `Execute` call. Under that lock it sets `s.caller` to the request session's verifier, and clears it with `defer` before the lock is released.
- Handlers read `s.caller` only inside `Apply`, and never store it.
- The read runs under the read lock and passes the session's verifier directly. It never touches `s.caller`.
- Tests check that the field is empty outside a call, and run concurrent reads and commands under `-race`.

## Read: `GET /api/v1/identity/users?after=&limit=`

The read needs the session and the CSRF header, and runs under the domain read lock:
- **Admission.** It gets the Answer from `Access`, then `gate.Decide` for `administer_installation` in the installation workspace.
- **Membership map.** It walks every page of `Workspace.Members` with the session's verifier into a map.
- **Paging.** It pages through `Installation.Users`, skipping service users (M04 handles them), until `limit` human users are collected or the list ends. The limit is 1–100, default 50. `next` is the last user ID scanned; it is empty at the end.
- **Row fields.** Each row has:
  - the user ID, state and epoch;
  - the login ID and email, from the directory, or empty if unknown;
  - the membership role, state and version, or none;
  - whether a password is set;
  - the pending invitation's expiry, if the directory holds one that has not expired.
- **Revision.** The domain counter, read under the same lock. Sign-in, sign-out and password set also bump the counter, so M01 may see `revision_conflict` after another user signs in, and reloads. The HB-257 plan accepts this.
- **Excluded data.** No password data, token, verifier or session data. A test checks the JSON keys against an allowlist.

## Directory

apps/api keeps an in-memory directory, changed only under the domain write lock:
- `user ID → login ID, email`, filled by the seed, bootstrap activation and user creation;
- the holders of the installation grant;
- pending invitations, `login ID → invitation ID, invitation verifier, expiry`. The verifier is random, held only in memory, and never returned or logged.

## Commands

**Common rules**
- Every command uses `administer_installation`, with the kind `installation`.
- The target is `users` for creation, and a user ID otherwise.
- The revision is the domain counter.
- An unknown target is refused by the executor as `permission_denied`. That is the executor's fixed behaviour for an unresolved target, which this ticket keeps.
- Every command requires a rationale of at most 256 bytes. The handler checks the length and refuses a longer one with `invalid_request`.
- `Operator.Reference` is the Answer's user, and `Operator.Reason` is the rationale. Neither ever comes from body fields.

| ID | Fields | Effect |
| --- | --- | --- |
| `identity.create_user` | `login` (normalized like L02), `email`, `role` (`administrator` or `member`) | <ol><li>Checks first: the login is free (directory and `Boundary.Resolve` of the triple), and the pending invitations and live links are below their configured limits.</li><li>Then: `CreateUser` with a random user ID, `Invite` with a fresh verifier valid for 7 days, a password-set link valid for 7 days, the directory entry and a pickup.</li></ol> Outcome: `user_id`, `link_ref`. |
| `identity.invite` | `role` | Invites an existing, active human user who has no membership in any state (none, or removed) and no unexpired pending invitation. This recovers a lost invitation. A suspended membership is restored with `reinstate_membership` instead. |
| `identity.issue_set_link` | — | Only while the user has no password. It replaces any earlier link. Outcome: `link_ref`. |
| `identity.issue_reset_link` | — | Only when a password is set and the user has no administrator membership. It replaces any earlier link. Outcome: `link_ref`. |
| `identity.suspend_user`, `identity.reinstate_user` | — | `SuspendUser`/`ReinstateUser` with the current epoch. |
| `identity.depart_user` | — | `DepartUser`. The directory entry, any pending invitation, the password and any link are removed. A departed login ID can never be created again; a new login ID is needed. |
| `identity.change_role` | `role` | `ChangeRole` with the current membership version. |
| `identity.suspend_membership`, `identity.reinstate_membership`, `identity.remove_membership` | — | The membership operations with the current version. |
| `identity.revoke_invitation` | — | `RevokeInvitation` for the user's pending invitation. The pending-invitation entry is removed; the user's login ID and email are kept. |

**Failures after a mutation**

`create_user` makes several changes. If a step fails after the first change, the handler returns a plain error, not a `Failure`. The executor then stores `internal_error` with an operation ID, the domain counter is bumped, and the read shows the partial state. The administrator recovers with `identity.invite` or `identity.issue_set_link`. Tests inject a failure at each step.

**Error mapping**

Only the executor's handler codes are used:

| Source | Code |
| --- | --- |
| `ErrLastAdministrator` and the protection rules above | `prerequisite_failed` |
| `ErrRefused` with `ReasonState`, `ReasonMember`, `ReasonMembership` or `ReasonUser` | `invalid_stage` |
| `ErrConflict` (the module version is read under the lock, so this means a state conflict) | `invalid_stage` |
| `ErrRefused` with `ReasonSession`, `ReasonNotAdministrator` or `ReasonAnswer` | `permission_denied` |
| `ErrRefused` with any other reason | `invalid_stage` |
| `ErrLimit`, `ErrUnavailable` | `dependency_unavailable` |
| `ErrInvalid` | `invalid_request` |

Module versions are read under the same lock, as in the HB-257 plan. The domain counter is therefore the only guard against lost updates; the module's own version check is not a second guard.

## Links are never in results

Command results and outcome fields are kept for replay, so link tokens must never appear there. A command that issues a link stores a pickup instead:
- the pickup holds a random `link_ref`, the link URL, the issuing session, and an expiry 10 minutes ahead;
- the outcome carries only the `link_ref`.

`GET /api/v1/identity/links/{ref}`:
- needs the session and the CSRF header, and answers with `Cache-Control: no-store`;
- returns the URL once, only to the issuing session, before the expiry, and then deletes the pickup;
- gives `permission_denied` to any other caller, a second pickup or an expired pickup, with no difference between those cases.

Pickups are deleted at sign-out and when they expire. A replay of the command returns the same `link_ref`, but the pickup is already used, so the administrator reissues.

## Invitation acceptance

A pending invitation is accepted, under the domain write lock, at two points:
- when `redeem` sets a password with a `set` link;
- after a successful password sign-in by that login.

Both prove the invitee, and both use the user's own new session verifier with the stored invitation verifier. The `set` path opens a short internal session and ends it afterwards.

- **If acceptance fails** (expired, revoked, inviter changed, user suspended): the pending entry is removed, the password set or sign-in still succeeds, and the read shows the user without a membership. The administrator then uses `identity.invite`.
- **If no invitation is pending,** for example for seeded users, nothing happens.

**Order at sign-in.** Sign-in proceeds as follows:
1. The password is verified outside the lock.
2. Under the domain write lock:
   - the reset generation is compared first (below);
   - the session opens;
   - if the login has a pending invitation whose workspace is the installation workspace and the requested workspace, it is accepted;
   - then `Access` runs, then the authentication policy check.

The membership persists even if the policy then refuses the sign-in. With `--seed=none` and no authentication policy yet, a member is refused by the policy, because only administrators are exempt from the "no policy" refusal. The member's acceptance still persists. This is tested, and the M02 policy screen is the remedy.

**Reset during sign-in.**
- The directory keeps a reset generation per login, bumped under the domain write lock when a reset link is committed.
- `signIn` reads the generation under the read lock before verifying the password.
- Under the write lock, as its first step and before it opens the session or accepts any invitation, `signIn` compares the generation. If it changed, `signIn` refuses with the ordinary sign-in failure.
- Wall-clock times are not used.
- The generation map is kept apart from the directory entry. It is never deleted, even on depart, so a login's generation only grows for the life of the process.

A sign-in verified against the old password therefore cannot open a session after the reset.

## Sessions end on reset

When `redeem` consumes a `reset` link, every apps/api session of that user is ended in the same domain write lock:
- identity `EndSession` is called for each;
- each is removed from the session table;
- the user's pickups are deleted.

The journal records the reset command with the target user. A test checks that an existing session is refused afterwards.

## Seed

The directory is filled for `sample-admin` and `sample-admin-2`, and `sample-admin` is recorded as the grant holder.

One more synthetic user is added: `sample-member` (`member@example.invalid`, role member). It is created and invited through the same functions as `create_user`. Its invitation is accepted when its seeded password-set link, written to the 0600 file, is first used.

# Not in this ticket (residuals)

- **Bulk revocation (M01 tab 4).** It needs asynchronous jobs and manifests. The screen will show it as not available in pre-launch.
- **Delivering links by email or message.** Not permitted; the administrator hands the link over out of band.
- **Granting or revoking the installation action by command**, and a separate workspace-administration action for multi-workspace builds.
- **Administrator password recovery.** An owner decision; reset links are refused for administrators.
- **Trust in the installation administrator.** They can take over any other account (see the trust boundary), so two-person approval does not hold against them. This is an owner decision. It goes away with a delivery channel they cannot read.
- **Members with `--seed=none` and no authentication policy** cannot sign in until a policy exists (M02).
- **Renewing the 30-day installation grants.**
- **Unknown targets** are reported as `permission_denied`, because that is the executor's behaviour.
- **Login ID and email in identity.** They are kept in the apps/api directory and lost on restart, like all state.
- **Persistence.**
- **`installation_scope` in the journal leak check** in `modules/commands`, a separate module ticket.

# Verification plan

**Literal tests**
- **Permissions.** Every command and the read, with:
  - the grant holder: allowed;
  - `sample-admin-2`: `permission_denied` from `installation_scope`;
  - a member;
  - no session;
  - a bad CSRF token.
- **Protection.**
  - Self-targeting is refused.
  - The last active administrator cannot be suspended, departed, demoted, or have their membership suspended or removed.
  - The last grant holder is protected.
  - A reset link for an administrator is refused.
- **Commands.**
  - The happy path of each command.
  - A stale revision gives `revision_conflict`.
  - An unknown target gives `permission_denied`.
  - A taken or departed login is refused.
  - A rationale longer than 256 bytes is refused.
  - `Operator.Reference` is the Answer user.
  - Each row of the error mapping.
- **`create_user` partial failure.** A failure is injected after each step; the result is `internal_error` with an operation, the counter is bumped, and recovery works through `invite` or `issue_set_link`.
- **Links.**
  - The pickup works once, only for the issuing session, and needs the CSRF header.
  - It answers with `no-store`.
  - It expires after 10 minutes, and is deleted at sign-out.
  - A replay returns the `link_ref`, and the pickup stays used.
  - No result, journal record or log line contains a token, checked by scanning the bytes.
- **Invitations.**
  - create, pickup and redeem: the user is a member with the role and signs in;
  - acceptance at sign-in after `identity.invite` for a user who already has a password;
  - an inviter changed or an invitation revoked leaves the user without a membership, and `invite` recovers it;
  - a reissued link invalidates the old one;
  - redeem with no pending invitation.
- **Reset.**
  - Existing sessions and pickups of the user end, and the journal shows the target.
  - A sign-in verified before the reset commit is refused.
  - Promotion is refused while a link is outstanding, and the refusal ends when the link is committed, replaced, departed or expired.
  - A reset redeem is refused when the user is an administrator at redeem time.
  - A sign-in whose generation changed is refused before any session or acceptance, tested with a fixed clock.
- **Sign-in order.** Acceptance happens before `Access`. A member with `--seed=none` and no policy keeps the membership while the sign-in is refused.
- **`--seed=none`.** Activation grants the action, and M01 works for the bootstrap user.
- **The read.**
  - service users are skipped, with paging and cursor;
  - all member pages are walked;
  - the joined fields are right;
  - the revision equals the counter;
  - the key allowlist holds.
- **The caller verifier** is empty outside a call, and concurrent reads and commands pass under `-race`.
- **Concurrency.** A redeem concurrent with a revoke or reissue passes under `-race`.
- **OpenAPI** lists the new commands.

**Property test.** Random sequences of the commands are run against a model of users, memberships, invitations, grant holders and the protection rules. After each step it checks:
- the exact result code;
- the read result;
- that there is always at least one active administrator who holds the grant.

**Checks**
- `-race -count=40` on the new tests;
- 100% coverage of apps/api/server;
- mutation testing of the new files;
- the cross builds, vet and gofmt.

# Review history

**Round 1 design review: FAIL.**

| Finding | Severity | Resolution |
| --- | --- | --- |
| H1: no protection for the last administrator or the last grant holder, and no ban on self-targeting | High | The protection rules above. |
| M1: error codes that do not exist in the catalogue | Medium | Mapped onto handler codes; unknown targets are `permission_denied`. |
| M2: partial `create_user` | Medium | Preflight checks, then a plain error after any mutation. |
| M3: no way to recover an invitation | Medium | `identity.invite`, and acceptance at sign-in. |
| M4: a reset could take over another administrator | Medium | Refused for administrators. |
| M5: with `--seed=none`, nobody holds the grant | Medium | Activation grants it. |
| L1: rationale and link command shape | Low | A rationale on every command, at most 256 bytes; separate set and reset link commands. |
| L2: pickup hardening | Low | CSRF, `no-store`, and pruning at sign-out and expiry. |
| L3: the read and the caller verifier | Low | The read passes the session verifier directly; `defer` clears `s.caller`. |
| L4: read paging and revision noise | Low | Paging and counter noise are defined. |
| L5: seeded users have no pending invitation | Low | Redeem tolerates no pending invitation. |

**Round 2 design review: FAIL.**

| Finding | Severity | Resolution |
| --- | --- | --- |
| M1: acceptance at sign-in came after `Access` | Medium | The sign-in order is now explicit; the `--seed=none` member case is tested and a residual. |
| M2: reset takeover could be bypassed, and full trust in the installation administrator was undeclared | Medium | Redeem recheck and promotion block; the trust boundary is declared as an owner residual. |
| L1: a sign-in verified before a reset could still open a session | Low | Per-login reset time check under the lock. |
| L2: `invite` could create a dead invitation for a suspended member | Low | `invite` requires no membership in any state. |
| L3: wording of what `revoke_invitation` removes | Low | It removes the pending-invitation entry only. |
| L4: grant validity was unstated | Low | 30 days; an expired grant is not a holder. |

**Round 3 design review: FAIL.**

| Finding | Severity | Resolution |
| --- | --- | --- |
| M1: the plan wrongly claimed the multi-step takeover route was closed | Medium | The trust boundary is restated accurately, and the tests assert only what the rules block. |
| L1: wall-clock reset times could be fooled | Low | A per-login reset generation, compared first under the lock. |
| L2: `modules/passwords` cannot report live links | Low | The directory tracks outstanding links. |

**Round 4 design review: PASS.** Two Low wording items were fixed:
- L-a: the generation check is now the first step in the sign-in order;
- L-b: the generation map's lifecycle is stated.
