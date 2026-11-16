---
type: ImplementationPlan
title: Installation administration action in modules/auth, valid only in the recorded installation workspace, for human subjects, never delegable
description: Phase 3a of the pre-launch plan (HB-257). The permission registry gains an installation administration action whose grants and decisions are confined to one installation workspace recorded once, so that installation-level user operations can later be exposed only to administrators of the bootstrap workspace.
status: final
---

# Scope and ownership

[HB-262](https://linear.app/hybrid-billing/issue/HB-262) is the separately reviewed `modules/auth` change that the [pre-launch plan](../HB-257/plan.md) requires before phase 3:

> Phase 3 therefore starts with a separately reviewed change, with a security-focused review. It adds an installation administration action to `modules/auth` and the gate catalogue. It defines the installation administrator as an administrator of the bootstrap workspace, which in `modules/identity` is the only workspace with members. The change checks the bootstrap workspace ID explicitly rather than relying on that invariant, and a test fails if a second workspace ever gets an administrator.

This ticket exposes no endpoint. M01 (HB-263) is the first user of the action.

Base: main `4888504`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the design before code, and the change after it, with a security focus. Machine: mac_mini.

Allowed files:
- `modules/auth/*`, `modules/gate/*`;
- `apps/api/server/*` (start and seed only);
- `docs/features/HB-262/*`.

# Observed baseline

- `modules/auth` has a closed action set: `view`, `view_cost`, `change_customer_policy`, `change_selling_policy`, `request_approval`, `approve`, `issue`, `download`, `deliver` and `approve_support_session`.
  - Grants are per subject or per client, with terms: action, kind, workspace scope with customer, legal entity and contract selectors, a period, and fields.
  - `decide` needs both a subject grant and, for a client principal, a client grant.
  - Delegations reuse `validTerms` and refuse the approval actions.
  - Support sessions allow only `view`.
- `modules/gate` builds the auth registry from identity (`NewRegistryWithSubjects`) and exposes `Administration` (`PutClient`, `AddGrant`, `RevokeGrant`).
- `modules/identity` gives workspace members only through `Bootstrap` and `Accept`, and `Accept` requires an existing administrator of the target workspace. In practice, only the bootstrap workspace has members.
- `identity.Installation` operations (`CreateUser`, `SuspendUser`, `ReinstateUser`, `DepartUser`) act on users across the installation and take a free-text operator.

# Decision

## Action

`AdministerInstallation = "administer_installation"` joins the action set. It has its own predicate, `installationAction`; the existing `humanOnly` is not reused, because `humanOnly` refuses client grants and routes decisions to the approval check. The action:
- is not a field action: it takes no fields, and `fields` must be empty;
- is not an approval action: `checkApproval` accepts only `approve`, so it is unchanged;
- is never delegable: `AddDelegation` refuses terms with it;
- has one fixed kind, `installation`: a grant or request for it with any other kind is invalid.

The gate re-exports `AdministerInstallation`, the reason `ReasonInstallationScope` and `SetInstallationWorkspace`. `commands.validDefinition` will accept the action through `Describes`, which is intended for HB-263.

## Installation workspace

The registry records one installation workspace with `SetInstallationWorkspace(id)`, exposed on `gate.Administration`:
- the method reads and writes under the registry lock;
- the ID must be valid, or the result is `ErrInvalid`;
- setting it again to the same value is a no-op; setting it to a different value is `ErrConflict`;
- until it is set, the action is unusable.

Setting it changes no existing decision, because no grant for the action can exist before it is set. It therefore does not bump the stamp generation.

## Grants

A grant with the action, subject or client, is valid only when all of these hold:
- the registry has a subject source (a registry built with `NewRegistry` and no source refuses the action);
- the installation workspace is set, and the scope's workspace equals it;
- the kind is `installation`;
- the customer, legal entity and contract selectors are all `All`;
- the period is `AnyPeriod`;
- there are no fields;
- for a subject grant, the subject is an active human member of that workspace.

Otherwise `AddGrant` returns `ErrInvalid`.

The administrator role is not checked when a grant is added, because the subject source reports only membership. The role is enforced at decision time in the gate (see below), so a grant to a plain member is harmless: it can never be used.

## Decisions in `modules/auth`

In `decide`, a request for the action is checked right after `current()`, before the `humanOnly` and support-session branches. It is allowed only when all of these hold:
- the registry has a subject source;
- the installation workspace is set, and the request's resource workspace equals it;
- the principal has no support session, and its subject is an active human;
- the subject has a direct grant (`direct`, never through delegations); for a client principal, the client also has a direct grant.

The refusals come in this order:
1. A request for the action with another kind or with fields fails `validRequest` and is refused with `invalid_request`, as for any other action.
2. A failed source, workspace, session, subject or direct-grant condition is refused with `PermissionDenied` and the new reason `installation_scope`. The support-session refusal therefore reports `installation_scope`, not `support_scope`.
3. The client check reuses the existing path, so a client without a direct grant is refused with `client_not_allowed`.

## Administrator role in `modules/gate`

The installation administrator is an administrator of the installation workspace, so the gate checks the role:
- for the action, `Gate.Decide` and `Gate.Recheck` refuse unless `answer.Kind() == identity.Human` and `answer.Role() == identity.Administrator`, with `PermissionDenied` and `installation_scope`;
- `DecideInSupport` refuses the action through `auth`.

The role check stays current for three reasons:
- the identity Answer carries the role and the membership version;
- `identity.Boundary.Recheck` rejects an Answer whose role or version is stale;
- `run` confirms the Answer again after the decision, and `Recheck` confirms it on each use.

A demoted administrator therefore loses the action at the next Answer or recheck, without revoking the grant.

## `apps/api`

At start, `apps/api` calls `SetInstallationWorkspace` with the bootstrap workspace, before the seed runs.

The value comes from `s.Workspace()`, which is also the workspace the server passes to `ArmBootstrap`. A test asserts that the armed bootstrap workspace and the installation workspace are equal.

The sample seed grants the action, with the kind `installation` and the scope rules above, to:
- `sample-admin`, the bootstrap administrator;
- the console client.

`sample-admin-2` is a member of the workspace and gets no installation grant.

## Invariant

The explicit workspace check in `modules/auth` and the role check in the gate are the real guards. Identity has no list of workspaces, so no test can enumerate every workspace. The single-workspace property holds by construction:
- `Bootstrap` creates members only in the armed workspace;
- `Accept` needs an inviter who is an active administrator of the target workspace;
- `CreateServiceUser` needs an administrator.

The tests show that the action cannot be obtained or used outside the installation workspace:
- an invitation into another workspace fails at identity, because no administrator exists there;
- a grant of the action for another workspace returns `ErrInvalid`;
- a decision for another workspace is refused with `installation_scope`.

A randomized identity-operation property that asserts every membership is in the bootstrap workspace would need a membership enumeration in `modules/identity`, which is outside this ticket. It is recorded as a residual.

# Residuals

- No endpoint uses the action yet (HB-263).
- With `--seed=none`, no installation grant exists. HB-263 decides how the first administrator of a real installation obtains it (for example at bootstrap).
- The installation workspace cannot be removed or rotated; it is set once for the life of the process. Under persistence, it must be stored with the identity bootstrap record.
- A randomized identity property over all memberships needs a membership enumeration in `modules/identity`.
- More installation roles, for example a read-only installation auditor.
- The role is not checked when a grant is added.
- The journal leak check in `modules/commands/commands_test.go` must list `installation_scope` once HB-263 journals refusals of the action.

# Verification plan

**Literal tests in `modules/auth`:**
- the action is known; it is described only with the kind `installation` and no fields;
- grant validation:
  - valid in the installation workspace, with full scope and any period, for an active human member and for an active client;
  - refused:
    - before the workspace is set, or without a subject source;
    - in another workspace, with another kind, with narrowed selectors, with a bounded period, or with fields;
    - for a service subject, a suspended subject, or a non-member;
  - a delegation of the action is refused;
- `SetInstallationWorkspace`: set once, idempotent, a conflicting value, an invalid ID, and concurrent set and decide under `-race`;
- decisions:
  - allowed for a granted human through the console client, and without a client;
  - refused with `installation_scope`:
    - before the workspace is set, or without a source;
    - for another workspace;
    - in a support session;
    - for a service subject;
    - when the only path is a delegation;
  - refused with `principal_not_current` for a suspended subject, because `current()` runs first;
  - refused with `client_not_allowed` when the client has no grant;
  - refused with `invalid_request` for another kind or with fields;
- the existing tests still pass.

**Gate tests:**
- `SetInstallationWorkspace` passes through;
- with an identity Answer, the action is:
  - allowed for the bootstrap administrator;
  - refused for a member who holds a grant;
  - refused after the administrator is demoted: a fresh decision with `installation_scope`, and a recheck of the earlier stamp with `identity_not_current`;
- an invitation into another workspace fails.

**`apps/api`:**
- the start sets the installation workspace to the armed bootstrap workspace;
- the seed grants the action to `sample-admin` and the console client, and the decision through the gate is allowed for `sample-admin` and refused for `sample-admin-2`.

**Property test:** the existing auth property model leaves out human-only actions, and the gate property test uses a fixed action list, so neither is disturbed. A separate random test checks auth decisions for the action against an independent model of the rules above.

**Checks:**
- `-race -count=10`, the 386 build, gofmt, vet, and `go test -race ./...`;
- 100% coverage of the changed code;
- mutation testing of the new rules.

# Review history

**Round 1 design review: FAIL.**

| Finding | Severity | Resolution |
| --- | --- | --- |
| The administrator role was never checked | High | The gate now checks the role at decision and recheck time. |
| The predicate and the position of the check were undefined | Medium | `installationAction` is a new predicate, checked before the session branch, through direct grants only. |
| The invariant test could pass while the invariant was broken | Medium | Replaced by negative tests; the enumeration property is a residual. |
| L1–L6 | Low | Addressed in the Decision and Residuals sections. |

**Round 2 design review: PASS.**

| Finding | Severity | Resolution |
| --- | --- | --- |
| L-a: the client refusal reason contradicted the rule | Low | The refusal order is now stated. |
| L-b: the reason for another kind or fields was not stated | Low | `invalid_request`; covered by a test. |
| L-c: the demotion recheck needs an exact reason | Low | `identity_not_current`. |
| L-d: the journal leak check | Low | Recorded as a residual for HB-263. |
