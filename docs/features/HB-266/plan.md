---
type: ImplementationPlan
title: M03 access API, with read-only grant and delegation listing, an effect preview, and grant management outside installation administration
description: Phase 3e of the pre-launch plan (HB-257). modules/auth and modules/gate gain read-only listing and an explain call. apps/api gains an access read, an effect preview and commands to add and revoke grants and revoke delegations, all under the installation administration action.
status: stable
---

# Scope and ownership

[HB-266](https://linear.app/hybrid-billing/issue/HB-266) provides the API behind M03 (roles, delegation and effect preview). The M03 screen follows in HB-267. Base: main `e70bcc7`.

**Ownership**
- Root (Claude Code) owns the plan, implementation and tests.
- A separate read-only reviewer reviews the design and the change, with a security focus, because this changes how permissions are granted.
- Machine: mac_mini.

**Allowed files**
- `modules/auth/*` and `modules/gate/*`: read-only additions only, with no change to any decision.
- `apps/api/server/*`
- `docs/features/HB-266/*`

# Observed baseline

**`modules/auth`**
- Grants are held per subject or per client. Each has terms: action, kind, workspace scope with customer, legal-entity and contract selectors, fields, and period. Each also has a validity window and a version, and a revoked grant is marked `closed`.
- Delegations chain from a delegator who holds a backing grant.
- `Decide` needs a principal whose installation, subject and epoch are current. A client principal also needs the client's own grant.
- There is no listing and no way to evaluate a subject without its session.

**`modules/gate`**
- `Administration` exposes `PutClient`, `AddGrant`, `RevokeGrant`, `AddDelegation`, `RevokeDelegation` and `EndSupportSession`.
- `Decide` needs an identity Answer.

**apps/api**
- The catalogue has the kinds `installation` and `focus_data`, the field `rows`, and any kinds and fields the configuration adds.
- The sample seed grants `view focus_data rows` to both administrators and to the console client, and `administer_installation` to both administrators and the console client.
- Members get no grants, so they cannot use the collected-data screens.

# Decision

## Module additions (read-only)

**`auth.Registry.Grants() []GrantView`**
- `GrantView` has: `ID`, `SubjectID`, `ClientID`, the terms, the validity, `Version`, `Closed`, `Active` and `Effective`.
  - `Active` means not closed and valid now.
  - `Effective` means active and, for a subject grant, the membership still matches the one the grant was made under (`memberWith`), so the grant can apply.
- The views are deep copies (`copyTerms`), sorted by ID, taken under `r.mu.RLock`.

**`auth.Registry.Delegations() []DelegationView`**
- The same shape, plus `DelegatorID`, `DelegateID`, `ParentID` and `AllowRedelegation`.

**Accessors**
- On `Selector`: `Any() bool` and `ID() string`.
- On `Period`: `Any() bool`, `From() Month` and `To() Month`.
- These let callers show terms without changing them.

**`auth.Registry.Explain(subjectID string, request Request) Explanation`**
- `Explanation` has `Allowed`, `Code`, `Reason` and `Delegations`, and no `Stamp`. The type system therefore stops an explanation from being used where a decision is needed.
- It takes `r.mu.RLock`, builds a principal with the registry's installation, the subject and the subject's current epoch, and no client or session, and runs the same `decide` path under `consistent`.
- An unknown subject gives `principal_not_current`.
- Installation and human-only actions (`administer_installation`, `approve`, `approve_support_session`) are refused with `invalid_request`. Their real decisions depend on gate role checks or approval flows that `Explain` does not run.
- It changes no state.

**`gate.Administration`** exposes `Grants`, `Delegations` and `Explain`, together with the view types.

The source test that no other package imports `modules/auth` still holds, because apps/api only uses gate.

## apps/api

All endpoints need `administer_installation`, as in M01 and M02.

**`GET /api/v1/identity/access`** returns:
- `revision`;
- `grants`: every grant except `administer_installation` grants, with:
  - `id`;
  - `subject` or `client`;
  - `action`, `kind`, `fields`;
  - `scope` (customer, legal entity and contract, each `all` or an ID);
  - `period` (`any` or from/to months);
  - `not_before`, `not_after`;
  - `state`: `active`, `revoked`, `expired` or `pending` (not yet valid), and `effective`, which is false when the grant no longer applies because the membership changed;
  - `version`;
- `delegations`, in the same style;
- `catalog`: the grantable pairs (see the console ceiling below), each with its action, kind, fields and the latest date a grant may run to;
- `subjects`: the active human members of the installation workspace, with their user ID and login.

Service users are left out of `subjects`, because the console refuses service subjects with a client and no service route exists yet.

Installation administration grants are left out, because they are managed only by the seed and by activation.

**`GET /api/v1/identity/access/preview?subject=&kind=&customer=`**
- `subject` must be one of the returned `subjects`; anything else gives `invalid_request`. `kind` must be in the catalogue, and `customer` must pass the module's ID rules.
- The server sets the resource's workspace to the installation workspace; it never comes from the query.
- For that subject, kind and resource, it returns one decision for each action other than `administer_installation`, `approve` and `approve_support_session`: allowed, or refused with a reason.
- It calls `Explain`, so a delegation-derived allow is shown with its chain IDs.
- Field actions (`view`, `view_cost`, `download`) are evaluated with every catalogue field of the matching class.
- This is advisory: a real request also needs the client grant and a current session.

## Commands

All commands:
- use `administer_installation`;
- require a rationale of at most 256 bytes;
- use the domain counter as the revision.

**Console ceiling.** apps/api never widens the console client's grants from a command. A pair (action, kind, fields) is grantable only if an active, non-installation console client grant already covers it with full scope. In pre-launch that is the seeded `view focus_data rows`, the only data route that exists. Each later phase seeds its own console ceiling, with its own review.

A subject grant may not run past the covering client grant's `NotAfter`. Otherwise the listing would show a grant that the console refuses.

**`access.grant`** (target `grants`)
- Fields:
  - `subject`: an active human member of the installation workspace, other than the caller;
  - `action`, `kind`, `fields` (comma-separated): together, a grantable pair;
  - `customer`: `all` or an ID. For `focus_data` it must be `all`, because the reader routes carry no customer.
  - `days`: from 1 up to the whole days left on the covering client grant, rounded down, so `now + days` never passes its `NotAfter`.
- Refusals:
  - `prerequisite_failed` for a grant to the caller;
  - `invalid_request` with the field name for everything else, which the handler checks against the roster and the catalogue before calling the module.
- Effect: adds a subject grant with a random ID, with these terms: the installation workspace, the given customer selector, all legal entities and contracts, any period, and validity from now for `days`.

**Trust boundary.**
- An administrator cannot grant themselves anything.
- One installation administrator can still grant another member or administrator, in one step.
- Within the declared M01 and M02 trust boundary, a holder could make an account they control a grantee. This is recorded as an owner residual.

**`access.revoke_grant`** (target: a grant ID)
- Revokes a subject grant made through `access.grant` or the seed.
- Refused with `prerequisite_failed` for: `administer_installation` grants, client grants, and grants of the caller.
- The caller exclusion stops an administrator from removing their own reader access by mistake. It adds no security.

**`access.revoke_delegation`** (target: a delegation ID)
- Revokes the delegation; its descendants stop applying, through the module's chain rules.

**Delegation creation** is out of scope. It belongs to the delegator's own flow, which is a residual.

**Error mapping**

| Source | Code |
| --- | --- |
| An unknown grant or delegation ID | `Resolve` answers not found, so the executor gives `permission_denied` |
| `ErrInvalid` after the handler's checks | `invalid_request` |
| `ErrConflict`, an already revoked target | `invalid_stage` |
| `ErrChanged`, `ErrNotBacked` | `prerequisite_failed` |
| `ErrUnavailable` | `dependency_unavailable` |

**Lock discipline**
- The read and the preview hold the domain read lock across admission, the listing and every `Explain` call, so one response is self-consistent.
- Commands run under the write lock in `execute`.
- The module calls take `r.mu.RLock` themselves.

The commands reuse the M01 command envelope and patterns: the caller verifier is not needed, the actor comes from the Answer, and rationale limits are enforced.

# Residuals

- Delegation creation by delegators, and support sessions (M03 support view).
- Narrower scopes (legal entity, contract, months) in the grant command; pre-launch exposes the customer only.
- One installation administrator can grant another account (see the trust boundary).
- The console ceiling only grows through later reviewed phases; there are no ceiling commands.
- Service users cannot receive grants here until a service route exists.
- The preview is advisory and does not include the client or session checks.
- Persistence.

# Verification plan

**Module tests**
- Listing: active, revoked, expired and pending grants; the copies do not alias.
- The accessors.
- The `Effective` flag after a membership is removed and recreated.
- `Explain`:
  - refuses installation and human-only actions;
  - races `AddGrant` and `RevokeGrant` under `-race`, and a generation change gives `stamp_not_current`;
  - matches `Decide` for the same subject;
  - refuses an unknown subject;
  - its stamp cannot be reused;
  - it changes no state;
  - the source test for module imports still passes.

**apps/api tests**
- Permissions, as in M01.
- The read hides installation grants.
- Grant:
  - the happy path, after which a member can read the collected-data endpoint through the console;
  - each validation boundary, including `days` past the client grant;
  - pairs outside the ceiling, including `kind=installation`;
  - a customer-scoped `focus_data` grant;
  - a non-member subject;
  - a self-grant, refused;
  - a cross-grant to another administrator, allowed, which is the declared route.
- No client grant is ever added by a command.
- An unknown target ID gives `permission_denied`.
- Preview: a subject outside the set, and a bad customer.
- OpenAPI and route registration.
- Revoke: refused for an installation grant, a client grant, the caller's own grant, and a revoked grant.
- Delegation revoke.
- The preview shape and its reasons.
- Concurrency under `-race`.

**Checks:** 100% coverage of the changed code, mutation testing, and the cross builds.

# Review history

**Round 1 design review: FAIL.**

| Finding | Severity | Resolution |
| --- | --- | --- |
| M1: an automatic client grant widened the console ceiling | Medium | Commands no longer widen it. The ceiling is the existing seeded client grants, and subject validity is capped by them. |
| M2: an administrator could grant themselves | Medium | Self-grant is refused; the cross-grant route is declared as a residual. |
| L1: Explain's type, installation path and lock | Low | A separate `Explanation` type; installation and human-only actions refused; an explicit RLock. |
| L2: preview exposure | Low | Subjects restricted to the returned set; the workspace set by the server; the customer validated. |
| L3: grants that could never be used | Low | The ceiling restricts kinds; `focus_data` must have customer `all`; service users excluded. |
| L4: misleading `active` state | Low | The `Effective` flag. |
| L5: error mapping | Low | Unknown targets resolve as not found; handler pre-checks; no partial client step remains. |
| L6: lock discipline | Low | Stated. |

**Round 2 design review: PASS.** Two Low items were addressed:
- N1: `effective` is now exposed in the read.
- N2: the remaining days are rounded down, with boundary tests.
