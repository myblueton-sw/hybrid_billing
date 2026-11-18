---
type: ImplementationPlan
title: M02 authentication policy, API and live screen, limited to what pre-launch sign-in can satisfy
description: Phase 3d of the pre-launch plan (HB-257). apps/api gains a policy read and three policy commands over modules/authpolicy with two-person approval. The console gains a live M02 screen in service mode. Only revisions that password sign-in can satisfy can be proposed.
status: stable
---

# Scope and ownership

[HB-265](https://linear.app/hybrid-billing/issue/HB-265) binds M02 (authentication providers and sign-in policy) for pre-launch. Base: main `ef74d1a`, after [HB-264](../HB-264/plan.md).

**Ownership**
- Root (Claude Code) owns the plan, implementation, tests and browser QA.
- The `ui-designer` role (AI, read-only) leads the visual design.
- A separate read-only reviewer reviews the design first and then the change, with a security focus, because this changes how authentication policy is set.
- Machine: mac_mini.

**Allowed files**
- `apps/api/server/*`
- `apps/console/src/service/*`
- `apps/console/src/modules/identity/pages/authentication/*` (new live files beside the preview)
- `apps/console/src/app/screen-groups/identity.tsx`
- `apps/console/package.json` (test list only)
- `docs/features/HB-265/*`

No module changes.

# Observed baseline

**`modules/authpolicy`**
- One current revision per workspace: methods, issuers, an MFA scope, the maximum authentication age, and step-up rules (an action, a maximum age, and whether MFA is required).
- Changes are proposed (`Propose`) and applied when a different administrator approves (`Approve`). Self-approval is refused. Only the proposer may withdraw (`Withdraw`).
- Both proposer and approver are judged against the current revision for the action `policy.change`. The approver must also satisfy the new revision (`approver_locked_out`).
- `Proposal(number)` reads one proposal; there is no list. Withdrawn proposals are deleted. Applied proposals are kept, up to a limit.
- Plans and lifts (bulk revocation) exist, but stay out of scope.

**apps/api**
- `signIn` calls `Evaluate` with `password` at the current time, and exempts administrators when no policy exists.
- The sample seed creates the first revision: password only, MFA none, maximum age 12 h, no step-up. `sample-admin` proposes it and `sample-admin-2` approves it.
- Only `sample-admin` holds `administer_installation`.

**Pre-launch sign-in has no MFA factor and no federated method.** A revision that requires MFA, a step-up with MFA, or any method other than password could not be satisfied by anyone. If such a revision were applied, sign-in or later policy changes would be locked out for good, until restart.

# Decision

## Permission

The read and all three commands use `administer_installation` with the kind `installation`, the same as M01.

To make two-person approval possible, the sample seed also grants the action to `sample-admin-2`, with the HB-262 terms, and records it as a grant holder.

**Trust boundary.** This restates [HB-263](../HB-263/plan.md) for two holders.
- Two-person approval separates two grant-holding accounts, not two people.
- Any installation administrator can still take over another account in several steps, as HB-263 declared:
  1. demote or remove the account;
  2. issue a reset link and redeem it;
  3. promote or invite the account back.
- With two holders, each can therefore:
  - act as both proposer and approver, by taking over the other's account;
  - suspend or depart the other and remain the only holder. From then on no policy change can be approved until restart, because no grant command exists.
- Both capabilities are declared residuals for the owner.

**Tests** pin this behaviour; they do not test it as if it were blocked. They check that:
- `sample-admin-2` can suspend `sample-admin`;
- the takeover route remains possible.

**Affected HB-263 tests.** These tests call `grantInstallation(SampleAdmin2)` or assume a single holder:
- `users_test.go` (TestProtectionRules, TestInvitationRecovery);
- `users_edges_test.go` (TestOperatorAndContract, TestInviterChangeLosesInvitationAtSignIn and the reset tests);
- `users_property_test.go` (its model).

They are updated as follows:
- where a test needs "an administrator without a grant", it revokes the seeded grant with `Administration.RevokeGrant` and deletes the `dir.holders` entry;
- tests that granted the action themselves now rely on the seed;
- the property model has two holders.

## Who acts, and their sign-in time

The authpolicy principal is built only from the Answer: `User`, `Workspace`, `Role` (`administrator` or `member` from the membership role) and `Kind` `human`.

The authentication is `{Method: password, At: <sign-in time>}`. The sign-in time is the `created` time of the request's apps/api session, which is set when `signIn` opens the session. No client field supplies any part of the principal or authentication.

`commands.Context` carries no role, and the endpoint's `access` runs before the domain lock. Each policy handler therefore re-derives both inside `Apply`, under the lock:
- the role, with identity `Access(s.caller, installation workspace)`;
- the session, with `s.sessions.get(s.caller)`.

A missing session or a failed `Access` is refused with `permission_denied`; a zero `At` is never passed to authpolicy.

The exported `OpenSession`, which opens a session without a password, is made unexported, because only tests use it. A source test checks that only `signIn` and test code open sessions.

## Maximum authentication age is enforced on every request

Today `signIn` evaluates the policy at the current time, so the maximum age never refuses anything. HB-265 also evaluates the policy in `access`, for every authenticated read, command and reader request:
- it calls `Evaluate(principal, {password, At: session.created}, "")`;
- a stale result (`stale`) ends the session and answers `unauthenticated`. Ending the session means identity `EndSession` and removal from the session table; each has its own lock.
  - `access` runs without the domain write lock on four paths: `current`, readers and commands (no lock), and `listUsers` (read lock only).
  - It therefore never touches the directory. In particular, it does not delete pickups.
  - The pickups of a session ended this way can no longer be reached, because `collect` requires the issuing session and `lookup` no longer finds it. They are pruned by expiry under the write lock.
- "no policy" passes, because a session already exists, and so does an allowed decision;
- any other refusal also ends the session and answers `unauthenticated`.

Sign-in itself evaluates at the current time, so it normally always passes the age rule.

**Exception: the wall clock steps back.** The authpolicy clock never moves backwards. If the wall clock steps back by more than the maximum age, for example after an NTP correction or a VM resume:
- every session is ended at its next request;
- sign-in is refused as `stale`;
- this lasts until the wall clock passes the registry's latest time.

This is declared as a residual and pinned with a fixed-clock test.

Sessions also have an absolute lifetime of 12 hours, so `max_age_minutes` is capped at 720. The step-up rule for `policy.change` then gives a shorter freshness window for policy changes alone.

## Read: `GET /api/v1/identity/policy`

The read needs the session and the CSRF header, and runs under the read lock with the same admission as the M01 read. It returns:
- `revision`: the domain counter.
- `policy`: the current revision, or `null`. The revision has `number`, `methods`, `mfa`, `max_age_seconds`, `step_up` (each with `action`, `max_age_seconds` and `mfa`), `proposer`, `approver` and `at`.
- `proposals`: every live pending proposal, plus the last 10 applied ones, most recent first.
  - Each has `number`, `state` (`pending` or `applied`), `proposer`, `approver`, `reason` and the proposed revision in the same shape.
  - For the caller it also has `can_approve` and `can_withdraw`, and `act_until`: the time the caller's sign-in stops satisfying the current policy for `policy.change`.
  - apps/api records the number and rationale of every proposal made through the seed or a command.
  - The read is pure. It fetches each record with `Proposal` and skips the numbers that no longer exist.
  - Dead records are pruned, and the 200-record cap is enforced, only under the write lock, in `policy.*` `Apply` and in the seed.
  - Applying a proposal discards every other pending one in the module. The screen says so before Approve.
- `now`: the server time.
- `signed_in_at`: the caller's session sign-in time.
- `act_until`: the time the caller must sign in again by to change the policy.
- `you`: the caller's user ID.

## Commands

Every command:
- uses `administer_installation` with the kind `installation`;
- has a revision equal to the domain counter;
- requires a rationale of at most 256 bytes.

| ID | Target | Fields | Effect |
| --- | --- | --- | --- |
| `policy.propose` | `policy` | `max_age_minutes`, `step_up_minutes` (both required `Integer`) | `Propose` with `ChangeRevision`, built as: workspace = the installation workspace; methods = `[password]`; MFA = `none`; maximum age from `max_age_minutes`; and, when `step_up_minutes` is not 0, one step-up rule `{policy.change, step_up_minutes, MFA false}`. The proposal number is recorded. |
| `policy.approve` | `proposal-<n>` | — | `Approve(n)`. |
| `policy.withdraw` | `proposal-<n>` | — | `Withdraw(n)`. |

**Field validation** (`invalid_request` with the field name)
- `max_age_minutes` is an integer from 5 to 720.
- `step_up_minutes` is 0, or an integer from 5 to `max_age_minutes`.
- Parsing rejects signs, leading zeros and values longer than 4 digits, so nothing can overflow.

**The rationale is stored in apps/api.** The command journal keeps only a digest of the payload, and authpolicy has no reason field. apps/api therefore keeps the validated rationale in its proposal record, and the read returns it, so the approver sees the proposer's reason before approving.
- Clients cannot set methods, issuers, the MFA scope or step-up MFA. Revisions that pre-launch sign-in could not satisfy therefore cannot be proposed through the API.

**Resolution**
- `policy` resolves when the installation workspace is set.
- `proposal-<n>` resolves when `n` is a recorded number and `Proposal(n)` exists.
- Anything else gets the executor's `permission_denied`.

**Error mapping** (handler codes only)

| authpolicy result | Code |
| --- | --- |
| `self_approval`, `not_proposer`, `approver_locked_out`, `not_administrator`, `other_workspace` | `prerequisite_failed` |
| `step_up_required`, `stale`, `future_authentication` | `prerequisite_failed` (the read gives the client what it needs to explain re-sign-in) |
| `not_pending`, `ErrNotFound` | `invalid_stage` |
| `ErrLimit` | `dependency_unavailable` |
| `ErrInvalid` | `invalid_request` |
| Any other refusal | `prerequisite_failed` |

## Screen (console service mode)

M02 views map as follows:
- `policies`: the live policy screen.
  - The current revision, in words.
  - Pending proposals, with Approve shown to the other administrator and Withdraw shown to the proposer.
  - Applied history.
  - A "Propose a change" form with the two durations and a required reason.
  - When the server's `now` is past `act_until`, the actions are replaced by a "sign in again to change the policy" line with a sign-out link. The client never compares against its own clock. A `prerequisite_failed` from a stale sign-in or `approver_locked_out` shows the same re-sign-in guidance.
  - Before Approve, the screen warns that other pending proposals will be discarded.
- `providers`: password sign-in shown as the active method. SAML, OIDC and LDAP are shown as not available in pre-launch.
- `test`, `keys`, `sync` and `recovery`: fixed "not available in pre-launch" states. Recovery points to M01's administrator-issued reset links.

The interaction patterns follow M01 (HB-264):
- a confirmation step with a reason;
- failures reload the list once settled;
- attempts are renewed on edit or revision change;
- focus is restored by stable selectors;
- a polite live region for results;
- no coloured borders;
- EN, KO and AR.

The visual design comes from the `ui-designer` role.

# External references (consulted 2026-11-16)

| Source | Observed | Adopted | Not adopted |
| --- | --- | --- | --- |
| GitHub Docs, "About protected branches" (https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches, retrieved 2026-11-16) | Required approvals before a change applies. An option requires the latest change to be approved by someone other than its author. Approvals of a changed diff are dismissed. | A pending proposal is applied only after a different administrator approves it. (Discarding the other pending proposals on apply comes from modules/authpolicy, not from this source; the screen says so before Approve.) | Approval counts above one; a dedicated pending-change interface is not described there. |
| Microsoft Learn, "Conditional Access adaptive session lifetime policies" (https://learn.microsoft.com/en-us/entra/identity/conditional-access/concept-session-lifetime, retrieved 2026-11-16) | Sign-in frequency is the time before a user must sign in again, with a time-based setting. Frequent re-authentication causes fatigue. Fresh authentication can be required for sensitive actions. | "Sign-in valid for" in minutes, enforced on every request. A separate "recent sign-in for policy changes" window for the sensitive action. A 5-minute floor against excessive prompts. | Per-application or per-device rules; the "every time" mode. |

The visual composition follows the `ui-designer` specification and the M01 patterns.

# Residuals

- **Wall-clock regression.** A wall-clock step back larger than the maximum age refuses sign-in and ends sessions until the clock catches up.
- **Trust in installation administrators.** With two holders, one can act as both proposer and approver, or remove the other (see the trust boundary). This is an owner decision.

- **`--seed=none`.** Only the bootstrap administrator holds the grant, so no second person can approve, and members cannot sign in until a policy exists. This awaits a reviewed grant command, or the owner's decision on single-administrator bootstrap.
- **No MFA, federation or step-up MFA** until those factors and methods exist.
- **Bulk revocation plans and lifts.**
- **Persistence.**

# Verification plan

**API (Go)**
- The trust boundary is pinned: the second holder can suspend the first, and the takeover route remains possible.
- Maximum age in `access`, tested on each path: `current`, a reader, a command, `listUsers` and `collect`.
  - A session older than the maximum age is ended and answers `unauthenticated`.
  - A fresh sign-in passes; "no policy" passes.
  - Another refusal (`future_authentication`, with a fixed clock) also ends the session.
  - Concurrent stale requests on all paths, together with commands and `collect`, pass under `-race`.
  - A clock step back larger than the maximum age refuses sign-in.
- A missing session in `Apply` gives `permission_denied`.
- Field boundaries: -1, 0, 4, 5, 720, 721, an overflowing value, and `step_up_minutes` equal to `max_age_minutes`.
- The pending list has more than 10 entries; applying a proposal discards the others; the read returns `now` and `act_until`; the rationale is stored and returned.
- Concurrent policy reads, with dead records present, alongside policy commands pass under `-race`.
- The property model has two holders; OpenAPI lists `policy.*`; the source test for opening sessions.
- Permissions: both holders, a member, no session, bad CSRF.
- Propose, approve by the other holder, withdraw by the proposer.
- Self-approval and withdraw-by-another are refused.
- Validation boundaries for both fields.
- The step-up rule: a session older than the step-up age is refused, and a re-sign-in passes.
- `approver_locked_out` (an approver session older than the proposed maximum age).
- The read shape and the proposal list, including skipping withdrawn proposals.
- The principal and authentication come only from the session.
- The seed's second grant, and the updated HB-263 tests.
- 100% coverage of the changed code; mutation testing; `-race`.

**Console**
- Unit tests for the client and model.
- Browser QA on the production build with a live API, in EN, KO and AR at 1440 and 390: propose as `sample-admin`, approve as `sample-admin-2` in another session, withdraw, the self-approval refusal, the step-up re-sign-in state, the provider and not-available views, keyboard use and computed borders.
- Design comparison by `ui-designer`, then the independent change review.

# Review history

**Round 1 design review: FAIL.**

| Finding | Severity | Resolution |
| --- | --- | --- |
| M1: granting the second holder widened the trust boundary, and the plan misstated it | Medium | The trust boundary is restated, with residuals and pinning tests. |
| M2: the maximum age was never enforced | Medium | It is now enforced in `access` for every request, and capped at 12 hours. |
| L1: the rationale was not stored | Low | Stored and returned. |
| L2: the proposal list bounds | Low | All pending proposals plus the last 10 applied; the records are bounded; discard-on-apply is stated. |
| L3: the role and session inside `Apply` | Low | Both are re-derived under the lock. |
| L4: the floor and the client clock | Low | A 5-minute floor; the server sends `now` and `act_until`. |
| L5: the affected tests | Low | Listed. |
| L6: field types and overflow | Low | Declared as integers, with bounded parsing. |
| L7: the exported `OpenSession` | Low | Made unexported, with a source test. |

**Round 2 design review: FAIL.**

| Finding | Severity | Resolution |
| --- | --- | --- |
| R2-M1: tearing down pickups in `access` without the domain lock risked a map race | Medium | `access` only ends the identity session and the session-table entry. Pickups become unreachable and expire. |
| R2-L1: wall-clock regression | Low | Declared as a residual and tested. |
| R2-L2: stale "journaled" wording | Low | Removed. |
| R2-L3: test paths unnamed | Low | The paths and the concurrency, refusal and clock cases are listed. |

**Round 3 design review: FAIL.**

| Finding | Severity | Resolution |
| --- | --- | --- |
| R3-M1: the read deleted records under the read lock | Medium | The read is now pure, pruning happens only under the write lock, and a `-race` test covers it. |
