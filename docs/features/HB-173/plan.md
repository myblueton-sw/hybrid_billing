---
type: ImplementationPlan
title: Binding identity state, sessions and memberships to permission decisions
description: Defines how the permission registry takes subject and membership state from the identity registry, and a gate through which a permission decision can be reached only with a current identity access answer for the workspace of the request.
status: draft
---

# Scope and ownership

[HB-173](https://linear.app/hybrid-billing/issue/HB-173/bind-identity-state-sessions-and-memberships-to-permission-decisions) is the follow-up of HB-169 (`modules/auth`) and HB-172 (`modules/identity`) and a prerequisite of the W-33 command plan. It is the sixth increment built under the owner's instruction of 2026-11-12. Base: main `b86dcd3`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because it changes how permissions and tenant boundaries are decided. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-13:

- `modules/auth` keeps its own subject records (identifier, kind `human` or `service`, state `active` or `suspended`, epoch) written through `PutSubject`. A decision takes a `Principal` (installation, subject, epoch, optional client, optional support session) that any caller can construct. Subjects are read in six places: the principal check, the active-human test of delegations and support sessions, grant registration, the approval check's author test, and support-session registration. The stamp is the registry instance, a generation that increases on every write and availability change, and a request digest. Delegations and grants know nothing of memberships.
- `modules/identity` is where a user's kind, state and epoch are true. It answers `Access` and `AccessByCredential` with an opaque answer (installation, workspace, user, kind, epoch, role, expiry) and confirms an answer with `Recheck`, which fails after suspension, departure, any identity change, a membership change in that workspace, a session end or a credential revocation, and which ignores idle time. It has a third state, `departed`. A membership change does not change the user's epoch. A membership has a version that increases on every transition.
- Nothing connects the two: a suspension in `modules/identity` does not deny a decision in `modules/auth`, and a removed membership would not either.
- `modules/customeraccess` has its own subject records for the customer usage read and export path, behind its own store.
- The identity boundary document: effective authority is the intersection of current account and membership state, grants, client scopes and resource ownership; tenant context is derived from authenticated membership and validated resource scope; a submitted workspace identifier is not sufficient; a delegation cannot exceed the grantor's authority.

# Decision

Three changes, standard library only.

## `modules/auth` takes subjects and memberships from a source

A registry may be constructed with a source: `NewRegistryWithSubjects(installation, catalog, clock, source)`. A nil source is refused. The source answers:

- `Subject(id)`: kind, state and epoch, or unknown;
- `Member(workspace, id)`: whether the subject is an active member of that workspace, and the membership's `joined` value, which changes only when the membership is created again;
- `Generation()`: a counter that increases with every change of any subject's kind, state or epoch and with every membership change.

With a source:

- every place that read the registry's own subject records reads the source instead, through one internal accessor, and `PutSubject` returns the invalid-request error and changes neither records nor generation, so subject state is written in exactly one place;
- a subject grant is registered only for an active member of the grant's workspace, records that membership's `joined` value, and matches a request only while the subject is an active member with the same value. This applies wherever a subject's direct grants are consulted: the decision itself, the root of a delegation chain, the backing check at delegation registration, the rule that a delegate holds a grant in the workspace, and the direct grants required of an approver and of a support-session approver. A grant is therefore unusable while the membership is suspended, usable again when that same membership is reinstated, and dead for good once the membership is removed: it does not come back when the user is invited again (provisional and conservative: whether removal should revoke or preserve grants is the owner's decision; until then an administrator must grant again). Client grants are unaffected. The recorded `joined` value is kept in the registry's internal record, not in the exported grant, and is unused without a source;
- a delegation is registered only while delegator and delegate are active members of the delegation's workspace, records the `joined` values of both, and covers a request only while every delegator and delegate of the chain is an active member with the recorded value. A membership-suspended delegator therefore suspends the delegation and a removed one ends it for good, also if the user is invited and granted again;
- a support session is registered only for an actor who is an active member of the session's workspace, records that `joined` value, and is current only while the actor is an active member with it. Registration checks the approver's authority before the actor, so that a caller without the approval grant learns nothing about who is a member. The test that an approver is not a delegate of the actor stays blind to memberships, because applying them there would loosen the separation;
- consistency: the source generation is read before the first source read and after the last one of every decision and of every registration that reads the source. If the two differ, a decision is a denial (`permission_denied`, reason `stamp_not_current`), except that an unavailable registry stays `authority_unavailable`, and a registration fails with its own error, distinct from and checked before the conflict of an existing identifier, so that the caller knows it may simply try again. A registration that is refused for another reason returns that reason without the second read (fail-closed), and a client grant, which reads no source, is still subject to the comparison; under continuous changes of users or memberships decisions can be refused repeatedly (fail-closed; a stated liveness limit). The stamp carries the source generation in addition to the registry's own generation, so a recheck also fails after any such change, including one that was made and undone on a delegator, a support actor or an approver whose own epoch the request does not carry;
- the source generation is installation-wide: a change in any workspace fails the outstanding stamps of all workspaces, and a recheck that fails with `stamp_not_current` is not retried but replaced by a new decision (a stated limit; generations per workspace are deferred);
- the registry's lock is held while the source is called. The source must never call back into the permission registry; the identity registry does not. Lock order is always permission registry, then identity registry.

A registry without a source behaves exactly as before and exists for the package's own tests; it must never back an exposed path.

## `modules/identity` offers a read-only view

`Registry.Subjects()` returns a fourth handle with:

- `Subject(user)`: kind, state, epoch, found;
- `Member(workspace, user)`: for a human, active when the membership is active and the user is active, with the membership's `joined` value, a new field that is 1 at bootstrap and at the first acceptance and increases only when a removed membership is created again; for a service user, active when the workspace is its owning workspace and it is active, with the value 1;
- `Service(user)`: creator and owner of a service user;
- `Generation()`: increases on every creation of a user, human or service, every change of a user's state or epoch (suspension, reinstatement, departure, linking, unlinking, the service-user state changes) and every membership change (bootstrap, acceptance, role change, suspension, reinstatement, removal, and the removals a departure makes). An operation that fails does not increase it. Invitation, session and credential changes do not increase it; they are covered by rechecking the access answer.

`Registry.InstallationID()` and `Registry.Clock()` return the installation identifier and the injected clock. The generation is a 64-bit counter and is not guarded against overflow (a stated limit; it cannot be reached in practice). The source cannot report that it is unavailable, so unknown and unavailable are the same answer, which is acceptable in memory and a limit for persistence. None of this changes any existing behaviour of `modules/identity`.

## `modules/gate`: the way to a decision

A new package composes the two. `gate.New(identityRegistry, catalog)` refuses a nil or unconstructed identity registry and an empty catalogue, constructs the permission registry itself — bound to the identity registry's installation identifier, its clock and an adapter over its view — and returns:

- a `Gate` with the decision operations;
- an `Administration` with the write operations of the permission registry (clients, grants, delegations, their revocations, ending a support session, availability) and none of its decision operations.

The permission registry is not returned. There is one gate per identity registry; constructing a second one over the same identity registry would give two sets of grants and is a breach of the composing code (a caller obligation). Both constructors of `modules/auth` remain exported, so "the only way" is enforced by a test in `modules/gate` that walks the module from its root, parses the imports of every non-test Go file outside the two directories `modules/auth` and `modules/gate` themselves, and fails if any of them imports the path of `modules/auth` or of a package below it (stricter than looking for calls, and not defeated by an alias, a dot or a blank import). So that other packages can still call the gate, `modules/gate` re-exports by type alias the catalogue, field class, request, resource, selector, month, period, scope, validity, terms, grant, delegation, support-session, client, action, code and reason types with their constants (including the two client state constants), constructors and errors; it does not re-export the registry, the principal, the permission stamp, the subject source or either constructor.

`Administration` is trusted from its holder and is the root of all authority: it proves neither the actor nor a delegator's consent, and its results reveal whether an identifier is an active member. The layer that exposes it must authorize each call (W-33), and it is never given to a layer that only decides.

Adapter: kind `human` and `service` map one to one and any other kind is unknown; state `active` maps to active and every other state, including `departed`, to suspended; an unknown user is unknown.

Decision operations, each taking the access answer instead of a principal:

| Operation | Passes to the permission registry |
| --- | --- |
| `Decide(answer, client, request)` | `Decide` with the subject and epoch of the answer and the optional client |
| `DecideInSupport(answer, supportSession, request)` | `Decide` with the support session; an empty identifier and a service answer are denied with `support_session_not_current` |
| `Recheck(stamp, client, request)`, `RecheckInSupport(stamp, supportSession, request)` | `Recheck` |
| `CheckApproval(answer, author, request)`, `RecheckApproval(stamp, author, request)` | the approval check, with the answer's subject as approver |
| `AddSupportSession(session, approverAnswer)` | support-session registration with the approver built from the answer |

Every decision applies this sequence, in this order:

1. the answer is confirmed with `identity.Recheck`. A refusal is a denial with code `permission_denied` and the audit reason `identity_not_current`; an unavailable identity registry gives code and reason `authority_unavailable`;
2. the workspace of the request's resource must equal the workspace of the answer, else a denial with `permission_denied` and the reason `workspace_mismatch`. The workspace of a decision is thus always the one the session or credential was admitted to, never one a caller submits. A support actor must therefore be a member of the workspace of the support session (provisional and conservative; support access by non-members is the owner's decision);
3. a service answer that carries a client is denied with `principal_not_current` (a service acts as a subject, never through a client);
4. the permission registry decides;
5. if it allowed, the answer is confirmed again, and a refusal now turns the result into a denial with `identity_not_current`.

What this guarantees: the validity of an answer never comes back once lost, so an answer that passes step 5 was valid at the instant of the permission decision, where the generation rule makes all subject and membership reads one consistent snapshot. Every allow was therefore true at one instant inside the call, and a call that starts after a suspension, removal or revocation has returned never allows. A change that completes while a call is returning can be followed by that call's allow; nothing stronger is claimed.

A gate denial sets the actor only from an answer that passed step 1 and never carries a stamp. The gate adds no authority: it can only turn an allow into a denial. The two new reasons are added to the permission package's closed list of audit reasons.

The gate returns its own decision value with its own stamp, an opaque value that holds the permission stamp and the answer the decision was made with. The recheck operations take that stamp and no answer: they apply the same sequence to the answer inside the stamp. Work started under an answer can therefore only be rechecked under that answer, and an explicit end of the login session fails every recheck of it. The zero stamp is denied with `stamp_not_current`.

`AddSupportSession` is a write: it returns the denial error when the approver's answer is not current, is not a human's, or its workspace differs from the session's workspace, and the unavailable error when the identity registry is unavailable. If the confirmation after a successful registration fails, the gate ends the session it just registered (a new session has version 1) and returns the denial error whatever that ending returns; the identifier stays consumed. Between the registration and that end the session exists, but it can be used only through the gate by its actor, whose own decision passes its own steps 1 to 5 (stated; the window gives nothing to the invalidated approver). The actor's membership is checked at use, by step 2.

`CheckApproval` and `RecheckApproval` additionally turn an allow of the permission registry into `self_approval_denied` when the author is a service user whose creator or owner is the approver (provisional). The author identifier is supplied by the caller's server-side code from the stored record; one person holding two users is not detectable, the author is not required to be a member of the request's workspace, and the audit reasons distinguish an unknown author from a missing grant for a caller who is a current member, so audit reasons must not reach callers (stated limits).

The role in the answer (`member`, `administrator`, `service`) is not used for permission decisions: what a subject may do with billing data is decided by grants only.

The zero `Gate` and the zero `Administration` deny with `authority_unavailable` and the unavailable error without a panic.

## Caller obligations the gate cannot check

- The client identifier is taken by the caller's server-side code from the admitted credential, never from request content; omitting it for a request that arrived through a client is a breach of the caller (a stated limit until the identity registry records clients).
- For an interactive request the answer passed to the gate is obtained from `Access` within that request, because `identity.Recheck` ignores idle time; a retained stamp is used only for work already started.

## What this does not do

- `modules/customeraccess` keeps its own subjects. A suspension in `modules/identity` still does not block the customer usage path. Bringing that path under the gate is [HB-174](https://linear.app/hybrid-billing/issue/HB-174/bring-the-customer-usage-path-under-identity-state-and-answer-based), created before this ticket merges; until then that path must not be exposed. This is a stated limit, not a resolution.
- Nothing is persisted, no command or HTTP layer exists, and no audit record is written.

Owner questions touched, each with a labelled conservative default above: the fate of grants when a membership is removed; support access by non-members; approvals of work authored by a service account.

# Allowed files

`modules/auth/*`, `modules/identity/*`, `modules/gate/*` (new), `docs/features/HB-173/*`.

# Verification

- `modules/auth`: all existing tests unchanged and passing. New tests for a registry with a source: a nil source refused; every subject read going through the source (principal check, delegation chain, support session actor and approver, grant registration, approval author); `PutSubject` refused without any change; a grant refused for a non-member and for a member of another workspace, unusable during a membership suspension, usable after reinstatement of the same membership, dead after removal and not returning after re-creation; each place that consults direct grants applying the membership rule (delegation root, backing check, delegate's workspace grant, approver, support-session approver); a delegation refused and ended by a delegator or delegate who is removed, membership-suspended or departed, and not coming back after re-invitation and a new grant; a support session refused for a non-member actor and not coming back after the actor's removal and re-invitation; a stamp failing after a source generation change with unchanged values; a generation change during a decision and during each registration, produced by a source that changes its generation between reads.
- `modules/identity`: the view for every state and kind; `Member` for humans and service users in the right and in another workspace; `joined` across removal and re-invitation; the generation increasing on exactly the listed operations and not on failed ones or on invitation, session and credential operations; all existing tests unchanged and passing.
- `modules/gate`, literal tests through real registries: a decision allowed for a member with a grant; denied with `identity_not_current` after user suspension, reinstatement with the old answer, departure, membership suspension and removal, role change, session end, identity link and unlink, answer expiry, and for the zero answer and an answer of another identity registry; denied with `workspace_mismatch` for a request naming another workspace; an answer for one workspace against a delegation in another; a service credential's decision allowed, denied after revocation, expiry and service suspension, denied for human-only actions and denied with a client; a client principal and a suspended client; `DecideInSupport` with an empty identifier, another user's session and a service answer; `AddSupportSession` with an approver answer for another workspace, a service approver and an approver invalidated during the call; approval with author separation, a departed, unknown and service author, a service approver, and a service author created or owned by the approver; each recheck operation after each of those changes, with the zero stamp and with a stamp of another gate; delegated decisions with the delegator removed, membership-suspended, suspended and reinstated, and departed, and the delegate departed; a removed and re-invited member needing a new grant; the zero gate and administration; `gate.New` on a nil and on an unconstructed registry; the import scan.
- Step 5 deterministically: the permission registry reads the injected clock at the start of its decision. A test clock that, once and only on that read, ends the login session, revokes the credential or changes the actor's role produces the interleaving in which only the second confirmation denies, because the permission registry cannot see those changes. A suspension or a membership removal at that read is seen by step 4 and asserted as such. The same technique with an approver's session end reaches the undo of `AddSupportSession`. The test clock never acts on a read made by the identity registry, which reads it under its own lock.
- Properties with a fixed seed over generated histories of identity operations, grants, delegations, clients and approvals: every gate decision equals a reference model written in the test from its own bookkeeping (user state, membership and its re-creation, session or credential, grants, delegations, client), sharing no helper with the packages and never calling `Recheck`; the test fails when an allow, a delegated allow or any denial reason is rarely reached.
- Concurrency under the race detector: while the acting user is suspended, while the membership is removed, and while a delegator is suspended, decisions that start after that operation has returned never allow; each reader is allowed at least once before.
- Mutation check by a script with results in the record, covering each step of the gate's sequence, the workspace comparison, the stamp's answer, the adapter's mapping of each state and kind, each subject and membership read in `modules/auth` going through the accessor, the grant's `joined` comparison at each place, the delegation membership checks, the refusal of `PutSubject`, both generation reads and their comparison, and each generation increase in `modules/identity`.
- `gofmt -l`, `go vet ./...`, `go test -race ./...`.
- Independent review of the actual change.

# Deferred

The customer usage path under the gate (HB-174); clients recorded by the identity registry; audit records and commands (W-33); persistence.

# Design review resolution

The independent design review of 2026-11-13 (read-only, nothing executed) returned FAIL (narrow) without a blocker: 3 HIGH, 9 MEDIUM, 5 LOW, all fixable in the plan text, and stated that no owner decision blocks implementation when the fail-closed forms are adopted. It found that the first draft settled two owner questions in the permissive direction. Adopted as written above:

| Finding | Resolution |
| --- | --- |
| D1 delegations ignored membership | The source answers membership; a delegation needs every party to be an active member of its workspace |
| D2 grants survived removal and re-invitation, and could be registered for non-members | Grants are registered for active members only, record `joined` and die with the membership |
| D3 subject reads and generation were not one consistent read | Generation read before and after; a difference denies or fails the registration |
| D4 `AddSupportSession` under the decision sequence | Its own rule, including undoing a registration whose confirmation fails |
| D5 the old path and the exported constructors | Nil source refused; an import scan; the sourceless registry stated as test-only |
| D6 client identifier | Caller obligation stated; a service answer with a client denied |
| D7 Administration | Stated as the root of authority, trusted from its holder |
| D8 what the stamp covers | The gate's stamp holds the answer; rechecks take the stamp and no other answer |
| D9 `DecideInSupport` | Human answer, same workspace, empty identifier denied; labelled provisional |
| D10 codes, reasons, zero values | Stated |
| D11 stale answers and idle time | Caller obligation stated |
| D12, D13 verification | Reference model with its own bookkeeping; a deterministic step-5 test; the missing cases |
| D14 author independence | Service author created or owned by the approver denied, provisional; the limit stated |
| D15 follow-up ticket | HB-174 created and cited |
| D16, D17 | One gate per identity registry with its clock; generation limits stated |

# Design re-review resolution

The narrow re-review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker or high finding, 5 MEDIUM and 6 LOW, all wording, and judged the document implementable once the five were folded in; no owner decision blocks implementation. All were adopted in the sections above: delegations and support sessions record `joined` like grants (E1); the guarantee of step 5 stated as what actually holds (E2); the deterministic step-5 test built on changes the permission registry cannot see (E3); the gate re-exporting the permission types, and the scan's rules (E4); the installation-wide generation and non-retryable rechecks stated (E5); the stamp carrying both generations (E6); a distinct error for a registration overtaken by a change (E7); the reasons of the support and approval rules (E8); the undo returning the denial whatever happens (E9); a second gate as a caller breach (E10); `joined` kept internally (E11). An earlier attempt at this re-review was started against the previous text by mistake, because the rewrite had not been saved; it was stopped and discarded before it reported. The folded text was not reviewed a third time; the change review of the implementation is the check on it.

# Change review resolution

The independent change review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker or high finding, 4 MEDIUM and 12 LOW. It found no sequence in which a gate operation allows or a registration succeeds against this plan, and confirmed the membership rule at every use of direct grants, the two generation reads around every source read, the lock order and the unchanged sourceless mode.

- MEDIUM: two claims of the verification record contradicted the code or tests (the actor asserted in only one of the step-5 cases; the order of the changed-source error and the identifier conflict, which held for grants only). The assertions were added and the three registrations made consistent.
- MEDIUM: the mutation section of the record was still a placeholder, because the full result file of the first run had been overwritten; one complete run on the final code replaced it.
- MEDIUM: no test would have failed if a generation read were moved. A test now changes the source at every single source read of a decision, an approval check and each registration and expects each to be noticed.
- LOW: the support-session registration told a caller without the grant whether an actor was a member; the approver is now checked first. An unavailable result is no longer overridden by a changed source. A vacuous recheck case and a vacuous record check were replaced, the reference model gained revocations and checks the allow bit of every decision, and the missing cases were added (unlink, rechecks of every kind, a second gate, a chain of two delegations, the service-user reinstatement in the view test). Naming remarks were followed. Three checks are kept as a second line of defence although nothing can reach them today: the adapter refusing an unknown user before it looks at the kind, `gate.New` refusing an identity registry without a clock, and the service-author rule in `RecheckApproval`. The early returns before the second generation read and the author-existence limit are stated above.

The narrow re-review of the same day returned PASS with findings, six LOW, on tests and the record: two mutations recorded as equivalent were missing assertions and are now caught; a third had only been caught by a hang and has a case behind it; the gate's comparison of a support session's workspace with the approver's cannot be tested with one workspace and is recorded in [HB-175](https://linear.app/hybrid-billing/issue/HB-175/seed-and-recover-workspace-administrators-and-add-the-cross-workspace).
