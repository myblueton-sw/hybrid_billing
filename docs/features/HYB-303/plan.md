---
type: ImplementationPlan
title: IAM-01 concurrent first-administrator creation and acceptance from another workspace, established from the code and fixed by tests
description: Establishes from the code at main bc0d03e what happens when two first-administrator activations race, when activation races the sample seed, and when an invitation or activation link is used from another workspace or installation; finds that both are already refused fail-closed, adds one small fail-closed guard to identity.Bootstrap (refuse when any user exists or the login is already linked), and specifies concurrency tests under -race with -count, two-workspace and two-installation fixtures, a restart test, mutation evidence and the record updates, with no console copy change. Revised after security design review round 1.
status: stable
tags: [HYB-303, security, identity, bootstrap, invitation, IAM-01, HB-42, L02, M01]
sources:
  - resource: docs/planning/v1.16/retention-access.md
  - resource: docs/planning/v1.16/acceptance-trace.json
  - resource: ../HB-259/plan.md
  - resource: ../HB-259/verification.md
  - resource: ../HB-263/plan.md
  - resource: ../HB-263/verification.md
  - resource: ../HYB-297/plan.md
  - resource: ../HYB-299/plan.md
  - resource: ../../reviews/HYB-301-trace-classification/report.md
---

# Scope and ownership

[HYB-303](https://linear.app/hybrid-billing/issue/HYB-303): IAM-01 (`retention-access.md:110`) requires concurrent first-administrator creation to be blocked and invitation acceptance from another workspace to be refused. Both were recorded as "not built or tested" (`HB-259/verification.md:109`, `HB-263/verification.md:204`) and listed as gap 15 in the HYB-301 report (`docs/reviews/HYB-301-trace-classification/report.md:330`). Owner direction of 2026-11-17: build faithfully to the planning.

- Base: main `bc0d03e`. Branch: `feat/HYB-303-iam01-bootstrap-race`. Worktree: `/private/tmp/hb303`. Machine: mac_mini.
- Every `file:line` for code refers to `bc0d03e`. The v1.16 planning files are in the authorized local workspace (`/Users/home/project/hybrid_billing/docs/planning/v1.16/`), not in this Git tree; their line numbers refer to that copy.
- Owners: Claude (root) writes the plan, tests and records. An independent security reviewer (`.claude/agents/independent-reviewer.md`, read-only) reviews this plan, then the actual diff. QA is the test run itself, as no screen changes.
- Allowed files:
  - `modules/identity/boundary.go`, for the one product change (see "Fix", root decision after review F2);
  - new `modules/identity/bootstrap_race_test.go` and new `apps/api/server/iam01_test.go`;
  - new `docs/features/HYB-303/verification.md`;
  - the IAM-01 rows of `docs/features/HB-259/verification.md:109` and `docs/features/HB-263/verification.md:204`, and gap 15 of `docs/reviews/HYB-301-trace-classification/report.md:330`, for dated pointers only (Q3).

  Any other product code (`signin.go`, `users.go`, other `modules/identity/*.go`) is touched only under the contingency in "Fix", and then only with a re-review.

## In scope

- Concurrent first-administrator creation: two or more activations of the same activation link, activation against the sample seed's bootstrap, and bootstrap against user creation and re-arming in the identity module.
- Acceptance from another workspace: an invitation of workspace A accepted naming workspace B, by a user of workspace B, through a sign-in to another workspace, and links of one installation used at another.
- Invitation replay, the third part of IAM-01, as regression evidence only (already built and tested, see the cross-check).

## Not in scope

- A product path to create a second workspace. None exists, and the planning does not ask for one in this ticket (see "How a second workspace can exist").
- Persistence of identity across restarts (an existing residual, `HB-263/plan.md:232-233`). Its effect on IAM-01 is recorded as a residual and an owner question.
- IAM-02 to IAM-08.

# Requirement cross-check

| Source | Text (verbatim) | Where it is met | Status after this ticket |
| --- | --- | --- | --- |
| `retention-access.md:110` | `\| IAM-01 \| 최초 관리자 동시 생성·초대 replay·다른 workspace 수락을 차단 \|` | See the three rows below | covered for the pre-launch scope (local password accounts, one workspace per installation), once the tests below pass |
| IAM-01, part 1 | 최초 관리자 동시 생성 … 차단 | Two layers: the server re-checks the activation under the domain write lock (`signin.go:461-465`) and clears it (`:483`); `identity.Bootstrap` refuses once consumed (`boundary.go:14`, `:25`). Seed and activation cannot overlap (`seed.go:99-102`, `:131-133`; `main.go:74-80`). New guard: `Bootstrap` also refuses when any user exists or the login is already linked | covered: the module race is already tested (`lifecycle_test.go:243-268`); new API concurrency tests (C1, C2), an extended module race (C3) and the guard |
| IAM-01, part 2 | 초대 replay … 차단 | An invitation is one-time (`workspace.go:103-105`, `:132`); tested ("accepting twice", `workspace_test.go:51`; `TestConcurrentRedeemAndRevoke`, `users_edges_test.go:291`) | covered (regression; one concurrent replay test added) |
| IAM-01, part 3 | 다른 workspace 수락을 차단 | `Accept` refuses an invitation of another workspace (`workspace.go:99-101`) and any user but the invitee (`:109-111`); the server accepts only into the boot workspace and only when the sign-in names it (`signin.go:268`, `users.go:255`) | covered with new two-workspace and two-installation tests |
| `retention-access.md:49` | 설치 운영자가 확보한 단기 일회성 bootstrap으로 최초 관리자 한 명을 원자 생성하고 재사용을 차단한다. | One-hour link in a 0600 file (`seed.go:132`, `writeExclusive` `:66`); atomic creation under the registry lock (`boundary.go:5-31`); an earlier run's link is refused in the next run (C11) | **not fully covered**: covered within one process run; re-arming after a restart is residual R1. Every record keeps this status |
| `retention-access.md:51` | 초대에는 조직·대상·허용 역할·만료·일회성 토큰을 묶고 저장 시 토큰 원문을 보관하지 않는다. | The invitation binds workspace, invitee, role, validity and a one-time verifier (`registry.go:48-60`; `admin.go:185-197`). There is no token held by the invitee. The stored SHA-256 digest is itself the acceptance credential, which the server keeps in memory (`directory.go:21-25`) and presents to `Accept` on the invitee's behalf; it is never returned or logged (`HB-263/plan.md:120`) | covered in substance; the "no raw token stored" wording does not map one to one, because the server-held digest is the credential (unchanged) |
| `acceptance-trace.json` (IAM-01, HB-42, `NOT_RUN`) | criterion as `:110` | The tests in this plan are the first run of IAM-01 | to be recorded in `HYB-303/verification.md` |
| `HB-259/verification.md:109`, `HB-263/verification.md:204`, `HYB-301 report:330` | "Not built or tested: concurrent first-administrator creation and acceptance from another workspace." | Original observations kept unchanged; a dated pointer "see HYB-303/verification.md" is added (Q3) | pointer added |

# Current behaviour (facts from code at `bc0d03e`)

## A. First-administrator creation

**Identity module.**
- `ArmBootstrap` (`installation.go:5-19`) refuses when a bootstrap is already armed or any user exists (`:14`). It fixes the bootstrap workspace at arming (`:17`).
- `Bootstrap` (`boundary.go:5-31`) runs under `r.mu`. It refuses unless armed, not consumed and not locked (`:14`, `refused(ReasonBootstrap)`), checks validity (`:17`) and the verifier (`:20`), then sets `consumed` (`:25`) and creates the user and its administrator membership in the armed workspace (`:26-28`) in the same critical section.
- `CreateUser` refuses until the bootstrap is consumed (`installation.go:37-39`), so no user can be created before the first administrator.
- Already tested:
  - sequentially: a second bootstrap and arming after consumption are refused (`lifecycle_test.go:213-214`); arming twice is refused (`:201`);
  - concurrently: 8 `Bootstrap` calls, of which exactly one succeeds and the others get `bootstrap_state`, with one user (`lifecycle_test.go:243-268`).
- Not tested: memberships, `WithoutAdministrator`, or contention with `CreateUser` and `ArmBootstrap`.

**Other bootstrap-named code, checked and found not to be a first-administrator path.** `installer.Registry.IssueBootstrap` and `RegisterAdmin` (`modules/installer/runs.go:349-415`):
- belong to the installation-run records of the installer module;
- are called by no product handler; the only caller is `installation_api_test.go:73-77`;
- do not import or write the identity registry.

They create no user, membership or grant.

**API service, `--seed=none`.**
1. `redeem` (`signin.go:350-379`) checks `activating(token)` under the read lock (`:356-359`). `activating` (`:311-318`) needs `s.boot.live`, an unexpired link and a constant-time verifier match.
2. `activate` (`:444-487`) hashes the password outside the locks (`:452`), then takes the domain write lock (`:461`) and **re-checks `activating`** (`:463`). A request that lost the race returns `link_invalid` here.
3. `identity.Bootstrap` is the second gate (`:469`). If it refuses, the request also returns `link_invalid` and nothing is stored.
4. Only the winner stores the password (`:472`), the directory entry (`:473`) and the installation, trust and bulk grants (`:474-482`, added by HB-263, HYB-297 and HYB-299). It then sets `s.boot.live = false` (`:483`) and deletes the activation file (`:484`).

Steps 2 to 4 all run under one domain write lock, so two activations serialize. The user ID is always `admin` (`s.bootUser`, `seed.go:89`), so a loser cannot create a second user ID either.

**Seed against activation.**
- With `--seed=sample` the bootstrap is consumed inside `Start` (`seed.go:99-102`), and `s.boot` is never made live (it is set only for `SeedNone`, `:131-133`). The activation file holds set links only.
- With `--seed=none` the link is made live only after every start step has run (`:122-133`).
- `main.go` calls `Start` (`:74`) before `Serve` (`:80`). No request can reach `redeem` while the seed or the start steps run.
- A redeem racing the seed is therefore impossible by construction, not by a lock. A test pins the observable part (no live activation after a sample start; arming refused).

**Peek** (`signin.go:328-348`) reads under the read lock and changes nothing.

**Trust hold.** While the installation is held, activation answers `503 dependency_unavailable` (`signin.go:358`, `:466-468`), already tested in `trust_hold_test.go`.

**Restart.** The identity registry is in memory (`server.go:244`). Each start removes the old activation file, draws a new secret and, with `--seed=none`, arms a new bootstrap (`seed.go:82-95`). After a restart a new first administrator can be activated. The previous run's link is not live in the new run, because its verifier is a fresh secret and `s.boot` is rebuilt; no test pins this yet (C11). This is not a concurrent creation but bears on "재사용을 차단" (`retention-access.md:49`). See residual R1 and question Q1.

**What `Bootstrap` does not check today.** It does not check whether a user already exists or whether the login's triple is already linked. It overwrites `r.users[userID]` and links the triple (`boundary.go:26-27`). This is unreachable today: `CreateUser` needs the bootstrap to be consumed first (`installation.go:37-39`), and arming needs an empty registry (`:14`). It is still a single-check dependency, so review F2 adds a guard (see "Fix").

**Conclusion A.** Reading the code finds no path that creates two first administrators within one run. Each layer alone is enough: the server re-check (`signin.go:463`) and the identity `consumed` flag (`boundary.go:14`). The identity layer is already tested concurrently (`lifecycle_test.go:243-268`). The API path (concurrent redeems of one activation link) has never run concurrently, so it is unverified until C1 and C2 pass.

## B. Invitation acceptance and workspaces

**How an invitation is bound.**
- `inviteUser` (`admin.go:185-197`) always invites into `s.bootWorkspace`. It stores the invitation ID and a random verifier in the in-memory directory, keyed by login (`:196`).
- The verifier never leaves the server. The invitee never holds an invitation token; acceptance happens server-side.

**Where acceptance happens.** Both paths call `acceptPending` (`users.go:251-262`), which always calls `Accept(s.bootWorkspace, …)` (`:255`) with a session of the invitee's own login.
- At the first password set (`signin.go:427-430`). The login comes from the set link (`link.Key.Subject`). The request has no workspace field.
- At sign-in (`signin.go:268`): **only when the requested workspace is the boot workspace** (`workspace == s.bootWorkspace && …`). A sign-in that names another workspace skips acceptance. It leaves the pending invitation untouched and then fails `Access` with `no_active_membership`. The client sees `401 unauthenticated` with the same body as every other sign-in failure (`TestSignInFlow`, `signin_test.go:119-140`), and the session it opened is ended (`signin.go:278-284`).

**Identity module `Accept`** (`workspace.go:85-136`) refuses, in this order:
- a dead session (`ReasonSession`);
- an invitation of another workspace or an unknown one (`:99-101`, `record_unknown_here`);
- a non-pending invitation (`:103-105`, `record_state`);
- outside validity (`ReasonExpired`);
- anyone but the invitee (`:109-111`, `not_the_invited_user`);
- a changed inviter (`inviter_authority_changed`);
- an existing member (`already_a_member`);
- then a wrong verifier, counted toward a lock.

The checks before the verifier do not count attempts. A refused cross-workspace or wrong-user attempt therefore cannot lock the real invitee out.

**How a second workspace can exist.**
1. **Not through the product.** Only `Bootstrap` creates a membership map (`boundary.go:28`; it is the only `memberships[…] =` assignment outside tests), and it runs once per registry. `Invite` needs an administrator of the target workspace. So the registry of a running installation has exactly one workspace with members, the boot workspace.
2. **A workspace ID that the client names.** The sign-in body carries `workspace` (`signin.go:208-217`). Any valid identifier is accepted as input and refused at `Access` unless the user is a member there. This is the reachable "other workspace" at the HTTP boundary.
3. **A second installation.** Another process with its own data directory, registry, password store and boot workspace (`sample-workspace` for the sample seed, `workspace` for none, `seed.go:41-47`). Its links and cookies are the other reachable "other workspace" artifacts.
4. **Inside `modules/identity` tests only.** The package-internal fixture `seed(workspace, administrator)` writes a second membership map (`workspace_test.go:481-487`). `TestWorkspacesAreSeparate` (`:489`) and `TestAUserOfTwoWorkspaces` (`:604`) use it. A legitimate user of two workspaces is allowed by design (`retention-access.md:57`: ending one organization's access does not remove another's).
5. The `Workspaces: 2` limit in `commands.Limits` (`server.go:157`, enforced at `modules/commands/store.go:52`) caps journal holders per workspace. It is not a provisioning path, and the identity module has no workspace limit.

**What is tested today.**
- Module: an invitation accepted naming another workspace (`workspace_test.go:39`, `:45`; with a real second workspace, `:502`).
- API: a sign-in to `other-workspace` by an administrator with no invitation (`signin_test.go:120`).

**Not tested:**
- a user with a pending invitation signing in to another workspace (that the invitation stays pending and is not accepted into the boot workspace);
- a user of workspace B trying workspace A's invitation;
- links and cookies of one installation used at another;
- any concurrent acceptance.

**Conclusion B.** No path accepts an invitation into a workspace other than the one it was issued for, or for a user other than its invitee. The boot-workspace guard at `signin.go:268` is the one line whose removal would let a sign-in into another workspace consume, and accept, the boot-workspace invitation, while the sign-in itself still fails. No test currently pins it.

# Fix

Both IAM-01 behaviours already fail closed. The work is tests that would catch a regression, mutation evidence that the tests have teeth, record updates, and one small guard.

**The one product change (root decision after review F2).** `identity.Bootstrap` (`boundary.go:5-31`) refuses with `refused(ReasonBootstrap)` (`bootstrap_state`) and changes nothing if either holds:
- any user exists (`len(r.users) != 0`);
- the given triple is already present in `r.identities`.

Both checks run under `r.mu`, after the existing armed, consumed and locked check and before the verifier comparison. A refused call therefore does not count an attempt. They replace the silent overwrite at `:26-27`.
- No API change: the server already maps any `Bootstrap` error to `422 link_invalid` (`signin.go:469-471`).
- Tested by C4b. Its mutation is M10.
- Nothing else in product code changes.

That nothing else needs to change is the expected result, not a verified one. **Contingency:** if any test below other than C4b (which needs the new guard) fails against unchanged code, the failure is a defect.
- Root-cause it with `systematic-debugging`. Fix it in the smallest scope to the contract below, and keep the failing test as the regression test.
- Record the before and after runs.
- Send the fix to the security reviewer as a new change.
- The plan status stays `draft` until then.

## Fail-closed contract (what the tests assert)

| # | Scenario | Expected result | Code |
| --- | --- | --- | --- |
| C1 | N ≥ 8 concurrent redeems of the one activation link, distinct logins | Exactly one `204`. Every other one is `422 link_invalid`. One user (`admin`), one administrator membership in the boot workspace, one directory login, one stored password, the activation file gone, peek afterwards `422 link_invalid`. Only the winner can sign in; each loser login gets `401 unauthenticated` | `link_invalid`, `unauthenticated` |
| C2 | One activation paused after hashing (it signals an `entered` channel before the second request is sent) while a second completes | The second is `204`. The first is `422 link_invalid` with no password, directory entry or grant for its login | `link_invalid` |
| C3 | Concurrent `Bootstrap` calls in the module | Exactly one `nil`. The rest are `ErrRefused` with `bootstrap_state` | `bootstrap_state` |
| C4 | `Bootstrap` concurrent with `CreateUser` and `ArmBootstrap`. The `CreateUser` calls use the **bootstrap user ID** with different logins, and other IDs | `ArmBootstrap` is always refused (`bootstrap_state`). Every `CreateUser` that runs before the bootstrap is consumed is refused (`bootstrap_state`); after it, the bootstrap user ID gets `ErrConflict`. Exactly one administrator, the bootstrap user, linked only to the bootstrap login. No `CreateUser` login resolves (`Resolve`) to a user who is an administrator anywhere | `bootstrap_state`, `ErrConflict` |
| C4b | New guard. (i) A registry where a user already exists, with the bootstrap still armed (built with the package-internal fixture). (ii) A triple already present in `r.identities` with `r.users` empty | `Bootstrap` refuses with `bootstrap_state` and leaves users, identities, memberships and attempt count unchanged | `bootstrap_state` |
| C5 | Sample start | No live activation. `ArmBootstrap` refused. A redeem of a random 32-byte token with `login` and `email` is `422 link_invalid` and creates no user | `link_invalid` |
| C6 | Concurrent `Accept` of one invitation by its invitee | Exactly one `nil`. The rest are `record_state` or `already_a_member`. One membership with version 1 | `record_state`, `already_a_member` |
| C6b | An accepted invitation replayed after the membership was removed (`RemoveMembership`), with the same session and the right verifier | `record_state`. No new membership, and the removed membership's state and version are unchanged | `record_state` |
| C7 | Invitation of w1 accepted naming w2, concurrently with the right acceptance in w1 | The w2 calls are always `record_unknown_here`. No membership in w2 for the invitee from that invitation | `record_unknown_here` |
| C8 | Users of w2 (a member, then the administrator) presenting w1's invitation, first with the right verifier, then with a wrong verifier at least `Attempts` times each | Always `not_the_invited_user`. The invitation stays pending and is not locked. The invitee then accepts in w1 with the right verifier | `not_the_invited_user` |
| C9 | Invited user with a password signs in naming another workspace | `401 unauthenticated`, identical body. No session left. The invitation is still pending (row `invitation_expires` set, `membership` null). A later sign-in to the boot workspace accepts it with the invited role | `unauthenticated` |
| C10 | Two installations A (sample) and B (none, activated) | A's set link at B and B's activation link at A: `422 link_invalid`. A's session cookie at B: `401 unauthenticated`. No user or membership created on either side | `link_invalid`, `unauthenticated` |
| C10b | Two installations A and B, both `--seed=none` and both activated, so they share the workspace ID `workspace` and the user ID `admin`. A's activation link (used), a set link issued at A and A's session cookie are presented at B. Optional: the same with two sample installations | `422 link_invalid` for both links; `401 unauthenticated` for the cookie. No user, password or membership changes at B | `link_invalid`, `unauthenticated` |
| C11 | Run 1 and run 2 of `--seed=none` on the same data directory (`Close`, then `New` and `Start` on that directory). Run 1's activation link, both after it was used and when it was never used, is presented in run 2 | `422 link_invalid` in run 2 for both. Run 2's own link still activates | `link_invalid` |

No new refusal codes. The planning asks that login and reset answers not reveal whether an account exists (`retention-access.md:41`), and HB-259 fixed one uniform `link_invalid` for every unusable link (`HB-259/plan.md:127`). An "already activated" code would tell a holder of a stale link that the installation has an administrator. It is kept out unless the owner decides otherwise (Q2).

# Tests

All tests use the existing fixtures:
- `modules/identity`: `setup`, `empty`, `seed`, `user`, `login`, `v`;
- `apps/api/server`: `setup(t, seed).ready()`, `pre`, `auth`, `links`, `as`, `created`, `redeem`, `signin`, `row`, `run`.

Concurrency uses a start barrier (a closed channel) so the goroutines contend, and `sync.WaitGroup`. Assertions that read server state take the domain read lock, so the tests themselves are race-free.

## `modules/identity/bootstrap_race_test.go` (new)

- `TestConcurrentBootstrapCreatesOneAdministrator` (C3). Extends `lifecycle_test.go:243-268` with 16 goroutines and checks memberships and `WithoutAdministrator`, which that test does not. An armed registry from `empty` and `ArmBootstrap`. 16 goroutines call `Bootstrap(v("boot"), "u<i>", triple("u<i>"), …)`. Assert:
  - one `nil` and 15 refusals with `ReasonOf(err) == ReasonBootstrap`;
  - `Users` returns one user;
  - `Members(operator, "w1")` returns one active administrator;
  - `WithoutAdministrator` is empty.
- `TestBootstrapRacesUserCreationAndArming` (C4). No ordering counter; only schedule-independent properties are asserted.
  - **Deterministic sequential part.**
    - `CreateUser` before `Bootstrap` is `bootstrap_state`, for the bootstrap user ID and for another ID.
    - `ArmBootstrap` then a second `ArmBootstrap` is refused (`bootstrap_state`).
  - **Concurrent part.** These start together on a freshly armed registry:
    - one `Bootstrap(v("boot"), "root", triple("root"), …)`;
    - 4 `CreateUser(operator, "root", triple("other<i>"), …)`, with the bootstrap user ID and different logins;
    - 4 `CreateUser` with other IDs;
    - 4 `ArmBootstrap(v("other"), …)`.
  - **Assertions:**
    - each `CreateUser("root", …)` returns `bootstrap_state` or `ErrConflict`;
    - each `CreateUser` with another ID returns `bootstrap_state` or `nil`;
    - each `ArmBootstrap` returns `bootstrap_state`;
    - there is exactly one administrator, `root`, and it is linked only to `triple("root")`;
    - no login created by `CreateUser` resolves (`Resolve`) to an administrator.
- `TestBootstrapRefusesAnExistingUserOrLinkedLogin` (C4b, new guard). Two cases, each with the bootstrap still armed and not consumed:
  - (i) a user inserted with the package-internal fixture;
  - (ii) a triple already in `r.identities` while `r.users` is empty, so only the login check can refuse.

  `Bootstrap` returns `bootstrap_state`. Users, identities, memberships and `boot.attempts` are compared before and after and are unchanged.
- `TestConcurrentAcceptanceOfOneInvitation` (C6). One invitation; 16 goroutines accept with the invitee's session and the right verifier.
- `TestAnAcceptedInvitationIsNotReplayedAfterRemoval` (C6b). Accept, then `RemoveMembership`. The test asserts that the invitee's session is still live (`Resolve` of the invitee's triple succeeds and `Access` on the session reports `no_active_membership`, not `session_not_current`), so the replay is refused by the invitation state and not by a dead session. Then the same `Accept` again. It must return `record_state`, with the removed membership's state and version unchanged.
- `TestAcceptanceNamingAnotherWorkspaceUnderContention` (C7). `seed("w2", "adm2")`; 8 goroutines `Accept("w2", id, …)` and one `Accept("w1", id, …)`, concurrently.
- `TestAUserOfAnotherWorkspaceCannotTakeAnInvitation` (C8). `seed("w2", "adm2")`, a member of w2, then `adm2` itself.
  - Each presents w1's invitation with the right verifier, then with a wrong verifier `limits.Attempts` + 1 times.
  - Then the invitee accepts in w1 with the right verifier on the first try. This proves the invitation was neither locked nor charged an attempt.

## `apps/api/server/iam01_test.go` (new)

- `TestConcurrentActivationCreatesOneAdministrator` (C1). `setup(t, SeedNone).ready()`; 8 goroutines `redeem` with `first<i>@example.invalid`.
  - The winner is identified from `s.dir.people` and must be the only taken login.
  - Each loser login is refused by `identity.Boundary().Resolve` and `passwords.Has`.
  - Afterwards peek on the activation token is `422 link_invalid`. This kills M12.
- `TestActivationThatLostWhileHashingIsRefused` (C2). Deterministic. `s.pause` is set once, before either request, and is never reassigned during the run, so concurrent reads stay race-free. On its first call (counted with an `atomic.Int32`) it sends on an `entered` channel and then blocks on a `release` channel; later calls return at once.
  1. Request 1 starts in a goroutine.
  2. The test waits on `entered`, so request 1 has passed the pre-check and finished hashing, before it sends request 2.
  3. Request 2 must be `204`.
  4. The test closes `release`; request 1 must be `422 link_invalid`, with no password, directory entry or grant for its login.
- `TestSampleStartLeavesNoActivation` (C5).
- `TestSignInToAnotherWorkspaceDoesNotAcceptTheInvitation` (C9). The flow of `TestInvitationRecovery` (`users_test.go:378-401`) for an **administrator** invitation: create, revoke, set the password, then `identity.invite` with role `administrator`. Then sign in naming `other-workspace`, then `sample-workspace`.
- `TestLinksAndSessionsDoNotCrossInstallations` (C10, C10b). Separate temporary data directories for each world.
  - Worlds `setup(t, SeedSample)` and `setup(t, SeedNone)`.
  - Then two `setup(t, SeedNone)` worlds, both activated, so both have workspace `workspace` and user `admin`. At B the test presents A's used activation link, a set link that A's administrator issued, and A's session cookie.
  - Optionally, two `SeedSample` worlds the same way.
- `TestAnEarlierRunsActivationLinkIsRefusedAfterRestart` (C11). Each pair of runs uses one data directory.
  - Run 1 on `--seed=none` reads its link and (case a) activates it or (case b) leaves it unused.
  - Run 2 is built with the existing restart fixture `world.restartWith` / `restart` (`ledger_test.go:25-47`). It closes run 1, which releases the ledger lock, and starts a new server with the same config and data directory.
  - Run 1's link is presented in run 2: peek and redeem are `422 link_invalid`. Run 2's own link activates.

## Mutation evidence (run locally, not committed)

Each mutation is applied in the worktree, the named new test is run **alone** with `-run '^TestName$' -race -count=20`, and the change is reverted. The `git diff --stat` after reverting is recorded as empty. The session `GOCACHE` is cleaned between batches (project lesson "mutation runs fill the disk").

Some mutations are already killed by existing tests:
- M9 by `lifecycle_test.go:201`;
- M4 by `workspace_test.go:39`;
- M5 by `workspace_test.go:42`;
- M7 by `workspace_test.go:51`.

The evidence that counts here is that the **new** test, run alone, also kills the mutation.

| Mutation | New test expected to fail when run alone |
| --- | --- |
| M1: drop the `consumed` check at `boundary.go:14` | None expected. Survivor: after the guard, a second `Bootstrap` is refused because a user exists. Defence in depth, like M3; `lifecycle_test.go:213` no longer kills it alone |
| M1′: drop the `consumed` check **and** the new guard | C3 |
| M2′: M1′ plus dropping the re-check at `signin.go:463` | C1, C2 |
| M3: drop only the re-check at `signin.go:463` | None expected. Survivor: the identity layer refuses. Recorded as defence in depth, not dead code (project lesson "surviving mutant is not dead code") |
| M4: drop `found.workspace != workspace` at `workspace.go:100` | C7 |
| M5: drop the invitee check at `workspace.go:109-111` | C8 |
| M6: drop `workspace == s.bootWorkspace &&` at `signin.go:268` | C9: the invitation is consumed and accepted although the sign-in failed |
| M7: drop the state check at `workspace.go:103-105` | C6b (the replay after removal gets a new membership) |
| M8: drop the "bootstrap consumed" check in `CreateUser` (`installation.go:37-39`) | C4 |
| M9: drop `r.boot.armed \|\|` at `installation.go:14` | C4 (a concurrent `ArmBootstrap` re-arms before consumption) |
| M10a: drop the user-exists check of the new guard | C4b case (i) |
| M10b: drop the linked-login check of the new guard | C4b case (ii) |
| M11: move the invitee check (`workspace.go:109-111`) after the verifier check | C8 (other-workspace users' wrong verifiers lock the invitation, so the invitee's acceptance fails) |
| M12: drop `s.boot.live = false` at `signin.go:483` | C1 (peek after activation answers `activate`, not `422`) |

Every mutation except the expected survivors M1 and M3 must be killed by its new test run alone. A survivor means that test is too weak, and it is strengthened before the change review. M9 is killed deterministically by C4's sequential re-arm step (and already by `lifecycle_test.go:201`).

## Commands

- `go test -race -count=50 -run 'Bootstrap|Accept|AnotherWorkspace|Invitation' ./modules/identity`
- `go test -race -count=50 -run 'Activation|AnotherWorkspace|SampleStart|CrossInstallations|LinksAndSessions|Restart' ./apps/api/server`
- `go test -race -count=10 -timeout 90m ./modules/identity ./apps/api/server` (the server package needs about 81 minutes under race ×10; amended during verification), for the existing bootstrap, invitation and trust-hold tests as regression.
- `go test -race -short ./...`, `go vet ./...`, and the repository's line-limit and no-comments checks on the changed files.

**Line limits.** Both new files are planned well under 1,500 lines. `boundary.go` grows by a few lines. `lifecycle_test.go` (711) and `workspace_test.go` (733) are not extended.

# Console and copy

None. The activation and set-password screens already map `422 link_invalid` to their invalid-link state (`apps/console/src/service/api.ts:31`; `Service.tsx:248`, `:317`). The sign-in failure copy is already uniform. The new guard surfaces as the same `link_invalid`. No screen, rendered check or terminology review is needed.

# Records

- `docs/features/HYB-303/verification.md`:
  - commands and observed results, and the `-count` used;
  - per-mutation results with the M1 and M3 survivors explained;
  - the reviewer and the verdict;
  - what was not run (Windows locking, see R4).

  It states that `retention-access.md:49` is **not fully covered** (R1).
- `HB-259/verification.md:109`, `HB-263/verification.md:204` and the HYB-301 report gap 15 (`report.md:330`):
  - the original observations stay unchanged;
  - a dated pointer is appended: "2026-11-xx: see [HYB-303/verification.md](…)", with the date of the change;
  - the trace rows also keep the note "`retention-access.md:49` not fully covered: restart re-arming (HYB-303 R1)".

# Residuals

- **R1. Re-bootstrap after restart (deferred; Q1 default stands).** The identity registry is in memory, so with `--seed=none` every start arms a fresh first-administrator activation, even when the decision ledger in the same data directory holds earlier history. Run 1's own link is refused in run 2 (C11). The re-activated administrator:
  - **gets the same user ID `admin`** (`seed.go:89`), so ledger records keyed to `admin` from earlier runs refer to it;
  - **does not have earlier suspend or role-end records applied.** Replay parks records for users that are not present in `s.awaiting` (`identityledger.go:138`), and `activate` does not call `applyAwaiting` (`signin.go:444-487`). Only invitation acceptance does (`users.go:255-258`). They stay parked and unapplied;
  - **runs under the previous trust policy.** `replayTrust` (`trustledger.go:195-225`) restores the latest trust activation from the ledger at start, so the new administrator works under a policy approved by the previous run's people.

  This is not a concurrent creation, and it needs the operator's 0600 file. It conflicts with "재사용을 차단" (`retention-access.md:49`), which therefore stays **not fully covered** in every record. Owner of the fix: the identity persistence work (`HB-263/plan.md:233`).
- **R2. No product second workspace.** Cross-workspace refusal is proven:
  - in the module, with a test-built second workspace;
  - at the API, with a named foreign workspace ID;
  - across two installations, including two with identical workspace and user IDs.

  When a multi-workspace build adds workspace creation (`HB-263/plan.md:226`), C7 to C9 must be re-run through that path.
- **R3. Federated acceptance.** SAML, OIDC, LDAP and JIT provisioning do not exist (IAM-02, IAM-04). Their acceptance and first-login paths need their own IAM-01 tests when built.
- **R4. Second process on one data directory (corrected after review F5).** The decision ledger takes an exclusive non-blocking lock on its file (`modules/decisionledger/open_unix.go:40`, `flock` `LOCK_EX|LOCK_NB`; `open_windows.go:19`, `LockFileEx`). A second process on the same data directory therefore fails `Start` at the `decision-ledger` step with `ErrStart` (`ledger.go:42`, `:48`) and never serves. Two remaining issues:
  - **Deleted activation file.** Before failing, that process has already deleted `activation.txt` (`seed.go:82-85`, before the ledger step at `:125`). The running process's link stays valid in its memory, but the operator's copy of the file is gone. This is not tested and has no fix in this ticket.
  - **Windows locking.** Locking on Windows is unverified; only the Unix path is exercised on this machine.

# Owner questions (with defaults)

- **Q1. Restart re-arming (R1).** Should `Start` refuse to arm a bootstrap when the data directory shows the installation was already activated, for example when there are decision-ledger records?
  - Default: **no change in this ticket.** Identity is not persisted, so refusing would leave a restarted `--seed=none` installation with no administrator and no way in. Kept as R1, with C11 pinning that an earlier run's link is refused.
- **Q2. Distinct code for a lost activation race.** Should the loser get, for example, `already_activated` instead of `link_invalid`?
  - Default: **no.** Keep the uniform `link_invalid` (`HB-259/plan.md:127`, `retention-access.md:41`).
- **Q3. HYB-301 gap 15. Resolved by root (review F9):** dated pointers to `HYB-303/verification.md` are added to the HYB-301 report and the HB-259 and HB-263 rows. The original observations are unchanged.

# Review and sequence

1. Independent security design review of this plan by the read-only reviewer. Round 1: CHANGES REQUIRED, no product defect; resolved below. A round-2 review of this revision follows before tests are written.
2. Write the tests. Run them against unchanged code, then add the `Bootstrap` guard, then run the mutations.
3. If any test other than C4b fails against unchanged code, apply the contingency in "Fix".
4. Independent security change review of the actual diff, then re-review of any fix.
5. PR, authorized merge (standing instruction), merge read-back, ticket completion and machine release.

# Review round 1

Security design review of the draft at `bc0d03e`. Verdict: CHANGES REQUIRED, covering test design, mutation claims and residuals; no product defect.

| Finding | Change in this revision |
| --- | --- |
| F1. The replay mutation was not pinned by a distinct scenario, and some mutations are already killed by existing tests | Added C6b (an accepted invitation replayed after removal gives `record_state` and no new membership), mapped to M7. Each new test is run alone with `-run`. Recorded that M4, M5 and M7 are already killed by `workspace_test.go:39`, `:42`, `:51`. The round-1 claim that `lifecycle_test.go:213` kills M1 is withdrawn in round 2 (R2-1) |
| F2. C4 did not exercise the bootstrap user ID; `Bootstrap` overwrites without checking existing users or links | C4 uses the bootstrap user ID with different logins. It asserts that a `CreateUser` before consumption is always `bootstrap_state` and that no `CreateUser` login resolves to an administrator. Added M8 (`installation.go:37-39`) and M9 (`r.boot.armed \|\|` at `:14`). Root decision: one fail-closed product change, `Bootstrap` refuses with `bootstrap_state` when any user exists or the login is already linked (`boundary.go:26-27`), tested by C4b and mutation M10 |
| F3. The two-installation test used distinct IDs only | Added C10b: a none+none pair, both activated, with the same workspace and user IDs. A's activation link, set link and session cookie are refused at B. Sample+sample is optional |
| F4. R1 understated what restart re-arming means | R1 kept deferred (Q1 default stands). Added C11: run 1's link, used or not, gives `422 link_invalid` in run 2 on the same data directory. R1 now states that the user ID is `admin`, that earlier suspend and role-end records stay parked in `s.awaiting` unapplied, and that `replayTrust` restores the previous trust policy. `retention-access.md:49` stays not fully covered in every record |
| F5. R4 was wrong | Corrected: the ledger `flock` makes a second process fail `Start` with `ErrStart`. Added the residual that the failing process deletes `activation.txt` before the ledger step (`seed.go:82-85` vs `:125`). Windows locking is unverified |
| F6. C2 had no handshake guaranteeing the order | The first pause caller signals an `entered` channel; request 2 is sent only after it |
| F7. Missing mutations and a weak C8 | Added M12 (drop `s.boot.live = false`, killed by the C1 peek `422`). C8 has other-workspace users present a wrong verifier at least `Attempts` times, then the invitee still accepts. Added M11 (move the invitee check after the verifier check) |
| F8. The `retention-access.md:51` row misdescribed what is stored | Reworded: the stored digest is the acceptance credential, held in server memory and presented by the server on the invitee's behalf |
| F9. Q3 | Resolved: dated pointer "see HYB-303/verification.md" added to the HYB-301 report and the HB-259 and HB-263 rows, with the original observations unchanged |

# Review round 2

Security re-review of the round-1 revision. Verdict: CHANGES REQUIRED, all on the mutation table and the facts. The `Bootstrap` guard itself passed its regression check.

| Finding | Change in this revision |
| --- | --- |
| R2-1. With the guard, M1 and M2 cannot be killed | M1 alone is recorded as an expected defence-in-depth survivor, like M3. Added M1′ (drop the `consumed` check and the guard; killed by C3) and M2′ (M1′ plus the re-check at `signin.go:463`; killed by C1 and C2). The claim that `lifecycle_test.go:213` kills M1 is removed. The rule now lists the expected survivors M1 and M3 |
| R2-2. The M1 row named C4 | C4 removed from that row (M1 now expects no kill) |
| R2-3. The cross-check and Conclusion A missed the existing concurrent test | Both now cite `lifecycle_test.go:243-268` (8 concurrent `Bootstrap` calls). C3 is described as an extension (16 goroutines, memberships, `WithoutAdministrator`). `lifecycle_test.go:201` is recorded as already killing M9 |
| R2-4. C4's sequence counter | Removed. C4 asserts only schedule-independent properties plus a deterministic sequential part: `CreateUser` before `Bootstrap` gives `bootstrap_state`, and arming then re-arming is refused |
| R2-5. Other bootstrap code not covered | `installer.Registry.IssueBootstrap` and `RegisterAdmin` (`modules/installer/runs.go:349-415`) are listed as checked and not a first-administrator path. No product handler calls them (only `installation_api_test.go:73-77`), and they do not write identity |
| R2-6. C6b could pass through a dead session | C6b asserts the invitee's session is still live before the replay |
| C11 fixture | C11 uses the restart fixture at `ledger_test.go:25-47` |

# Review round 3

Review round 3: APPROVE (N1, N2 folded in).

| Item | Change |
| --- | --- |
| N1 | M7 row: only C6b kills M7; "C6 also" removed |
| N2 | C4b case (ii) is built with an `r.identities` entry and empty `r.users`. M10 is split into M10a (user-exists check, killed by case i) and M10b (linked-login check, killed by case ii) |
