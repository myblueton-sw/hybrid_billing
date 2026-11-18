---
type: ImplementationPlan
title: Authentication policy with allowed methods, MFA scope and step-up evaluated against the current revision, dual-controlled revisions that cannot lock out their approver, and block-first chunked bulk revocation
description: Increment of work item W-42 after HB-172 and HB-173. A per-workspace authentication policy names the allowed methods, the MFA scope and step-up rules with a maximum authentication age. An authentication event is evaluated against the current revision. Revisions are dual-controlled, and the approver must satisfy the new revision. Bulk revocation over a fixed scope blocks the whole scope before any chunk runs, and resumes by chunk.
status: stable
---

# Scope and ownership

[HB-255](https://linear.app/hybrid-billing/issue/HB-255) is an increment of work item W-42 (requirement HB-42). HB-172 built users, identities, invitations, memberships, sessions and service accounts, and HB-173 bound them to permission decisions. This increment covers the parts of HB-42-I and the W-42 scale rule that need no owner decision:
- **From HB-42-I.** A common authentication policy boundary that accepts password, SAML/OIDC SSO and LDAP/AD methods, MFA, and immediate blocking.
- **From the scale rule.** Bulk revocation uses a fixed scope, chunks and resume, and activates the security block first, so that a batch delay never keeps existing access for any caller that follows the recheck obligation below.
- **Negative property.** Nothing in this module re-enables a blocked user except a dual-controlled `Lift` by people outside the plan's scope, and no call path returns an allowed decision without the full evaluation.

It does not cover:
- protocol adapters, password storage and verifiers, MFA enrolment and factor verification. Callers pass an already verified authentication event.
- account recovery, the IdP product, the LDAP path, the login identifier scope, self-signup and external collaborator policy. These are owner decisions (HB-42-D).
- bulk invitation;
- wiring into `modules/identity` (`Access`, `Recheck`, session opening) and `modules/auth`. Blocking downloads, jobs and tokens in progress (HB-42-V) depends on that wiring and is NOT_RUN here.
- persistence and screens.

Base: main `9b95ea7`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the design and the change, because this is a permission and security boundary. Machine: mac_mini.

Allowed files: `modules/authpolicy/*` and `docs/features/HB-255/*`.

# Observed baseline

- `modules/identity` has users with an epoch for immediate blocking, identities as (issuer, subject, kind) triples never linked by email alone, sessions with a recent-authentication check (`session_too_old`), and the roles `member`, `administrator` and `service`. Its `Access` answer carries the user, workspace and role.
- `modules/identity` has no notion of authentication method, factor or policy. Its plan defers "authentication methods and protocol adapters; password verifiers, MFA, step-up, authentication strength and recovery" and "bulk invitation and revocation with chunks".
- Modules in this repository import only the standard library, so this module takes the principal as plain values and states the caller obligation.
- Other modules use a proposal/approval register with human-only actors, `ai_not_allowed` and `not_human` refusals and a single mutex. This module keeps that shape and adds workspace binding and authentication of every actor.

# Decision

## Caller obligations

Dangerous if a later layer forgets:
- **Principal.** Every `Principal{User, Workspace, Role, Kind}` passed in must be built from a current `identity.Access` answer for that workspace, never from client input. This module cannot check it.
- **Authentication.** Every `Authentication` must have been verified by the authentication boundary: its method, issuer, factors and `At` are trusted as given.
- **Reads.** `Plan`, `Chunk`, `Revision`, `Policy` and `Proposal` have no actor check. They are for trusted callers already bound to the workspace in the identifier; the batch worker reads `Chunk` only through the system path.
- **Recheck.** A caller must hold the decision's `Version` and call `Current(workspace, version)` on every request and at every `Recheck` checkpoint of a long download or job. When it returns false, the caller must evaluate again before continuing.

## Security version

Each workspace has a security version, starting at 0. It increases by one whenever a revision is applied, a revocation plan touching the workspace is applied, or a plan is lifted. Every decision carries the version it was made at. `Current(workspace, version)` is true only if the version is still the workspace's current one. A version change therefore forces every caller that follows the recheck obligation to evaluate again, and a block takes effect at the next request or checkpoint.

## Policy revisions

A workspace's policy is a sequence of immutable revisions, numbered from 1. Before revision 1, evaluation refuses everything with `no_policy`.

A revision holds:
- **`Methods`**: a non-empty, strictly ascending subset of `ldap`, `oidc`, `password`, `saml`.
- **`Issuers`**: optional, strictly ascending, at most 64 issuer IDs. When non-empty, an `oidc`, `saml` or `ldap` authentication must name a listed issuer. This binds the workspace to its own IdP connections.
- **`MFA`**: `none`, `administrators` or `everyone` (every human role).
- **`MaxAge`**: the maximum age of an authentication for ordinary access, 1 minute to 30 days.
- **`StepUp`**: up to 64 rules, strictly ascending by action ID. Each rule has an action ID, a maximum age (1 minute to `MaxAge`) and whether MFA is required for it regardless of `MFA`.

The action `policy.change` is used for every proposal and approval in this module. A workspace can give it a step-up rule.

## Actors

Every proposal, approval, withdrawal and lift is made by a `Principal` with an `Authentication`:
- Assistants are refused with `ai_not_allowed`, system actors with `not_human`.
- The principal's workspace must be the workspace the change is about (`other_workspace`), and its role must be `administrator` (`not_administrator`).
- The authentication must pass `Evaluate` for that principal against the current revision with action `policy.change`; its refusal reason is returned.
- The approver must be a different user from the proposer (`self_approval`).
- Every check is repeated at approval against the state at that moment, and the proposer must still not be blocked. The proposer's role is not rechecked, because the approval carries only the approver's principal; the approver's own administrator check covers the change.
- A proposal is bound to the workspace's security version when it is proposed. Whenever the version changes, every pending proposal of that workspace is dropped and its entries released in the same critical section as the version increase, so no proposal is applied over a revision, plan or lift applied after it was proposed. A later approval or withdrawal gets `ErrNotFound`, and the proposer must propose again.
- Only the proposer may withdraw a pending proposal (`not_proposer`), so a target of a pending plan or tightening cannot withdraw it.
- Until a workspace has revision 1, every action except proposing, approving and withdrawing a revision is refused with `no_policy`, and those revision actions skip the current-revision check.

**Revision approval, no lockout and no relaxation by a weaker sign-in.** The approver's authentication must also pass the new revision for the `administrator` role with action `policy.change` (`approver_locked_out`). So a relaxation can only be approved by someone who satisfies both the current and the new policy.

**Lift.** Neither the proposer nor the approver of a `Lift` may be in the plan's scope (`in_scope`). Because a blocked user fails `Evaluate`, a blocked administrator cannot act at all.

## Authentication events and evaluation

An `Authentication` is `Method`, `Issuer` (required for `oidc`, `saml` and `ldap`, empty for `password`), `Factors` (strictly ascending, each `otp`, `webauthn` or `idp_mfa`) and `At`.

`Evaluate(principal, authentication, action)` refuses an assistant principal with `ai_not_allowed` and a system principal with `not_human`. It then checks forms, all returning `ErrInvalid`: the principal, a role of `member` or `administrator` (`service` is a form error, because service subjects use credentials), the authentication form, `idp_mfa` with `password` or `ldap`, and the action ID (empty means ordinary access). Then it returns an allowed decision or one refusal reason, in this order:
1. `no_policy`: the workspace has no revision.
2. `blocked`: the user is in an unlifted plan for this workspace.
3. `method_not_allowed`: the method is not in `Methods`.
4. `issuer_not_allowed`: `Issuers` is non-empty and the method is not `password` and the issuer is not listed.
5. `future_authentication`: `At` is after the clock.
6. `stale`: the age is strictly over `MaxAge`. An age equal to `MaxAge` is allowed.
7. `mfa_required`: MFA is required for the role and there is no second factor.
8. For an action with a step-up rule: `step_up_required` if the age is strictly over the rule's maximum age, then `step_up_mfa_required` if the rule requires MFA and there is no second factor.

A second factor is `otp` or `webauthn` with any method, or `idp_mfa` with `oidc` or `saml`. The decision carries the revision number and the security version. An action with no step-up rule is ordinary access.

## Bulk revocation

A revocation plan has a workspace, a strictly ascending list of 1 to 100,000 user IDs, a chunk size of 1 to 1,000 and a reason. User IDs are not checked against `modules/identity`; unknown IDs are blocked harmlessly. It is proposed and approved as above.

When the plan is applied, it gets the ID `<workspace>/<n>`, where `n` is a per-workspace sequence that only increases, so IDs are never reused and need no tombstones. A plan ID has its own form: a valid workspace identifier, `/`, and a decimal `n` from 1 to 2^63−1 without leading zeros. The whole scope is blocked in the same critical section, and the security version increases. Like every version change, this drops the workspace's pending proposals, including those of proposers who are now blocked. Once apply returns, no `Evaluate` returns allowed for a user in scope.

For a scope of `s` users, the chunk count is `ceil(s / chunk)`. Chunk `i` covers users `i*chunk` to `min((i+1)*chunk, s) - 1`. `Chunk(id, i)` returns those users, with `ErrInvalid` for an index outside `[0, count)`; it also answers for a lifted plan, whose `Complete` is refused. A system actor (the batch worker) reports `Complete(plan, index)`:
- an index outside `[0, count)` is `ErrInvalid`;
- the next chunk advances the cursor; an already completed chunk is idempotent; a later chunk is refused with `out_of_order`;
- a lifted plan is refused with `lifted`, so a resuming worker never revokes access that was restored;
- assistants are refused with `ai_not_allowed`, humans with `not_system`.

`Progress(plan)` returns the scope size, chunk size, chunk count, completed chunks, the next chunk index and whether it is complete or lifted. A complete plan keeps its block. A block is lifted only by a `Lift` change for the whole plan. A user in several unlifted plans of the workspace stays blocked while any of them is unlifted.

## Reads

Every read returns a copy: `Revision(workspace, number)`, `Policy(workspace)` (the current revision), `Current(workspace, version)`, `Plan(id)` with progress, `Chunk(id, index)` (the users of one chunk, for the worker), `Proposal(number)`.

## Bounds

| Bound | Limit |
| --- | --- |
| Workspaces with a policy | 10,000 |
| Revisions kept for reads per workspace | the latest 100; numbering continues without limit |
| Step-up rules and issuers per revision | 64 each |
| Unlifted plans per workspace | 100 |
| User entries in pending and unlifted plans, per workspace | 100,000; a lift, a withdrawal or a drop on a version change releases them |
| User entries in pending and unlifted plans, all workspaces | a plan of `n` users is admitted if the global total plus `n` is at most 2,000,000, or the workspace's own total plus `n` is at most 1,000. So no group of workspaces can stop another's small emergency revocation, and the memory bound is 2,000,000 + 10,000 × 1,000 = 12,000,000 entries |
| Lifted plans kept for reads | the latest 1,000 |
| Users per plan, chunk size | 100,000; 1 to 1,000 |
| Pending proposals | 100 per workspace, 25 per proposer, 10,000 in all (covering workspaces that have no policy yet) |
| Applied proposals kept | 1,000 |
| Identifiers | 1–128 bytes of `[A-Za-z0-9._:-]` |
| Reason text | 1–200 bytes of UTF-8, with no control characters |

A full bound returns `ErrLimit`. Because pending caps and entry quotas are per workspace, one workspace cannot stop another from proposing; only the global entry bound is shared, and the 1,000-entry reservation keeps small revocations possible.

## Errors

`ErrInvalid` (form), `ErrNotFound` (including a proposal dropped as stale), `ErrLimit`, and `ErrRejected` wrapped in a `Refusal` with a reason. Lifting a plan that is already lifted is refused with `lifted`, at proposal and at approval.

## Residuals

- Protocol adapters, verifiers, password storage, MFA enrolment and factor verification.
- Recovery, including a workspace whose every administrator is blocked or cannot satisfy its policy. Nothing here can bypass the policy; recovery is an owner decision.
- Wiring into `modules/identity` and `modules/auth`, including the recheck obligation for downloads, jobs and tokens in progress.
- Bulk invitation; external collaborator policy.
- Pending proposals of a proposer who departs or loses the administrator role stay counted until recovery, because this module cannot see that change; a blocked proposer's proposals are dropped.
- Owner decisions, all provisional:
  - the method list, the MFA scopes and the age bounds;
  - trusting `idp_mfa` from `oidc` and `saml`;
  - issuer-level rather than connection-level binding, and an empty `Issuers` list allowing any issuer of an allowed method;
  - no grace on tightening;
  - dual control for bulk revocation, which is slower than the single-actor `identity.SuspendUser` for one user;
  - every security version change dropping all of the workspace's pending proposals, even unrelated ones, so a pending revocation may need to be proposed again, and two colluding administrators can delay one by applying an unrelated change;
  - blocks scoped to one workspace, lifted only per plan, never lifting themselves;
  - administrators as the only role that may change policy.
- Persistence, audit records and screens.
- Proposal numbers are installation-wide, so `Approve` or `Withdraw` of another workspace's number reveals that it exists (`ErrNotFound` versus `other_workspace`). A lift names its workspace in the plan ID and is checked against the actor before the plan is looked up.
- SEC01, AUD-AR-09, AUD-DB-08 and AUD-QA-04 are NOT_RUN at system level.

# Verification plan

Literal tests:
- **Forms and bounds.** Every form rule and bound, including `idp_mfa` with `password` and `ldap`.
- **Proposals.** Two-person rule, actor kinds, pending caps, repeat proposals, only the proposer withdrawing.
- **Actors.** Workspace binding, administrator role, authentication of proposer and approver against the current revision, `approver_locked_out` for a relaxation approved with a sign-in the current policy refuses, `in_scope` for lifts, and blocked administrators unable to act.
- **Evaluation.** Every form error before any refusal, every reason in order, at the exact age bounds; issuers; step-up age and MFA; the revision and version in the decision; `Current` false after each version change.
- **Revocation.** Block before any chunk; chunk ranges including a partial last chunk; in-order, idempotent, out-of-order, out-of-range and lifted chunks; resume from progress; completion keeps the block; lifting; overlapping plans; released entries.
- **Staleness.** A pending proposal dropped by each kind of version change (revision, plan, lift), with `ErrNotFound` at approval and released entries.
- **Plan IDs.** The form at its edges: leading zero, `2^63−1`, overflow, missing or misplaced `/`.
- **Concurrency.** Concurrent evaluations, chunks and proposals under `-race`.

Property test: random revisions, authentications, plans, chunk reports and lifts. An independent model of decisions and progress must match.

Checks: `-race -count=10`, the 386 build, gofmt, vet, `go test -race ./...`, 100% statement coverage and mutation testing.

# Review history

- Round 1 (independent reviewer, read-only): FAIL. Findings and what changed:
  - H1. A security version per workspace, carried in decisions, with `Current` and a stated recheck obligation; downloads, jobs and tokens in progress moved to residuals.
  - H2. Revision approvers must satisfy both the current and the new revision.
  - H3. Every actor is authenticated against the current revision, so blocked users cannot act; `in_scope` for lifts.
  - H4. Principals are bound to the workspace and must be administrators; the identity-answer obligation is stated.
  - M1. An optional issuer allow-list per revision.
  - M2. Per-workspace caps, released entries on lift, and kept-for-reads windows.
  - M3. `lifted` and out-of-range chunk reports, and the partial last chunk.
  - M4. More provisional decisions listed, and the negative property restated.
  - L1–L3. Form checks first, `service` as a form error, strict ages, `step_up_mfa_required`, strictly ascending scopes, and caller obligations.
- Round 2 (same reviewer): FAIL. Findings and what changed:
  - N1. Per-workspace entry quotas covering pending and unlifted plans, a shared global bound with a 1,000-entry reservation per workspace, and a stated memory bound.
  - N2. Only the proposer may withdraw.
  - N3. Plan IDs are never reusable.
  - N4. Issuer granularity and the permissive default are provisional.
  - N5. Every non-revision action is refused with `no_policy` before revision 1.
  - N6. `Evaluate` refuses assistant and system principals.
- Round 3 (same reviewer): FAIL. Findings and what changed:
  - P1. Plan IDs are generated per workspace at apply, so there is no global ID cap.
  - P2. A blocked proposer's pending proposals are dropped at plan apply; departure is a stated limit.
  - P3. The proposer's role is not rechecked at approval, stated.
  - P4. The reservation is written as a formula.
- Round 4 (same reviewer): FAIL. Findings and what changed:
  - Q1. Proposals are bound to the security version; a version change drops the workspace's pending proposals, so a stale proposal can never be approved.
  - Q2. Plan IDs have their own form.
- Round 5 (same reviewer): PASS. Low items folded in:
  - R1. The plan number is bounded by 2^63−1, and the scope size is named `s`.
  - R2. A caller obligation for reads, and `Chunk` ranges and lifted plans stated.
  - R3. Dropping unrelated pending proposals is listed as provisional.
  - R4. Literal tests named for staleness, plan IDs and chunk ranges.

# Change review resolution

- Change review round 1 (separate independent reviewer, read-only): PASS with lows, folded in:
  - L1. A global pending cap of 10,000.
  - L2. A lift's workspace comes from the plan ID, so the actor check runs before the plan lookup; proposal-number existence is a stated residual.
  - L3. The clock is read before the mutex is taken.
  - L4. Tautological and dead test lines removed, and exact entry counts asserted after drops.
  - L5. The property model keeps its own chunk size. Actors, staleness, quotas and cross-workspace cases are covered by literal tests.
  - L6. The unused `ErrConflict` was removed.

