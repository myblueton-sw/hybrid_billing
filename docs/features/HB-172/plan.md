---
type: ImplementationPlan
title: User identities, invitations, memberships, sessions and service accounts
description: Defines the first identity model of work item W-42, stable users with external identities, a one-time first administrator, single-use pinned invitations, workspace memberships with a last-administrator rule, sessions and access answers that end at once on suspension, and service accounts with expiring credential verifiers.
status: draft
---

# Scope and ownership

[HB-172](https://linear.app/hybrid-billing/issue/HB-172/w-42-user-identities-invitations-memberships-sessions-and-service) is work item W-42 (requirement HB-42) with the identity parts of AR-09 and QA-04. It is the fifth increment built under the owner's instruction of 2026-11-12. Base: main `4dfb5fd`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the model decides who may enter a workspace. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-13:

- Requirement HB-42: manage a user's access identity from creation through invitation, activation, permission grant, change, suspension, departure and revocation; user login accounts and CSP collection accounts are separate. Proposed direction: separate immutable UserID, external Identity, WorkspaceMembership, RoleBinding, Session, ServiceAccount and Credential; a common authentication boundary for password, SAML/OIDC and LDAP/AD adapters; a one-time first administrator; MFA, recovery, permission expiry and immediate blocking; no automatic linking of another identity because an email matches. Scale: users, memberships and audit are listed with server filters and cursors; bulk invitation and revocation use a fixed scope, chunks and resumption, and the blocking version is activated first so that a slow batch never leaves existing access open. Acceptance: invitation reuse, acceptance in another workspace, concurrent first administrator, email reassignment, download, job and token during suspension, removal of the last administrator, recovery bypass; a departed human's owned work is never promoted implicitly to a service account.
- Open decisions of HB-42: SSO/IdP product, allowed authentication methods per organization, LDAP direct or federated path, login ID scope, MFA scope, self-signup, account recovery and external collaborator policy. On 2026-10-28 the owner deferred the choice of authentication method.
- The identity boundary document (`docs/architecture/security-boundaries/identity-session-permissions.md`, draft): identity namespaces are provider connection and trusted issuer and subject; the same email, domain or name never links accounts or grants membership; linking needs existing reauthentication plus new identity proof; sessions keep a verifier, not the identifier, with authentication time, idle and absolute expiry and a revocation version; service credentials are separate from people with a named owner, expiry, rotation and revocation and never human approval authority by default; bootstrap, invitation and reset material is short-lived, single-purpose, target-bound, atomically consumed, with reissue invalidation and bounded attempts, stored as a verifier digest; the authoritative denial is committed before a revocation is reported effective; bootstrap creates one initial administrator through atomically consumed operator-held material; organization SSO enforcement must not be bypassed by another login path; last-administrator protection is tested. All durations, algorithms and products are unresolved.
- Existing code: `modules/auth` (HB-169) decides permissions for a subject with kind, state and epoch that it is given; it authenticates nobody. `modules/organizations` (HB-168) holds workspaces. `modules/customeraccess` has its own subjects for the customer usage path. All are in memory.

# Decision

One Go package, `modules/identity`, standard library only, with an in-memory registry. It records who exists, how they are recognised and where they belong, and answers whether a presented session or service credential is current. It verifies no password, speaks no protocol and stores no secret. No entry point is wired to it in this increment.

## Three handles

One constructor returns a registry from which three handles are taken, so that a layer can be given only the operations it may call:

| Handle | Holder | Operations |
| --- | --- | --- |
| Installation | the installation operator path | `ArmBootstrap`, `CreateUser`, `SuspendUser`, `ReinstateUser`, `DepartUser`, `Users`, `WithoutAdministrator` |
| Boundary | the authentication boundary | `Bootstrap`, `Resolve`, `OpenSession`, `EndSession`, `Access`, `AccessByCredential`, `Recheck`, `LinkIdentity`, `UnlinkIdentity` |
| Workspace | the workspace command layer | `Invite`, `RevokeInvitation`, `Accept`, `ChangeRole`, `SuspendMembership`, `ReinstateMembership`, `RemoveMembership`, `Members`, `CreateServiceUser`, `SuspendServiceUser`, `ReinstateServiceUser`, `RetireServiceUser`, `ServiceUsers`, `AddCredential`, `RevokeCredential` |

Workspace operations check the acting session inside the package, because being an administrator is this package's own fact. An acting session is named only by its 32-byte verifier and evaluated by the current-session predicate; a serial number or an access answer never names an acting session. Every Workspace operation other than `Accept` and self-removal requires a current session whose user holds an active administrator membership in the workspace the operation names, and a record of any other workspace is refused like an unknown one; this includes `RevokeInvitation`, `AddCredential`, `RevokeCredential`, `Members` and `ServiceUsers`. Installation and Boundary operations are trusted from their holder: the layer that exposes them must be authorized and must have authenticated the person. `CreateUser`, `SuspendUser`, `ReinstateUser` and `DepartUser` require a non-empty operator reference and reason, kept for the later audit record, and the three state changes require the user's expected epoch, so that a stale reinstatement cannot undo a later suspension and a repeated call is refused instead of increasing the epoch twice. An installation operator role is not modelled. Dangerous if a later layer forgets to authorize them, stated: `OpenSession` is login as anyone whose triple is known; `ReinstateUser` undoes a security block; `SuspendUser` by one workspace's staff can leave another workspace without an administrator; `ArmBootstrap` on a fresh installation is a takeover; `Users` lists across workspaces.

## Records

| Record | Key | Content | Life |
| --- | --- | --- | --- |
| User | user identifier, create-only | kind `human` or `service` (never changes), state, epoch; a service user also has its owning workspace, creator and named owner | `active` ↔ `suspended`; `active` or `suspended` → `departed`, terminal |
| Identity | (connection, issuer, subject), create-only forever | the user, an email attribute, its link order within the user | `linked` → `unlinked`, terminal |
| Bootstrap | one per installation | verifier, validity, attempts, the workspace it seeds | `armed` → `consumed` or `locked`; expiry by clock; never armed twice |
| Invitation | invitation identifier, create-only | workspace, role, verifier, validity, inviter with the inviter's epoch and membership version, the user it is for, attempts | `pending` → `accepted`, `revoked`, `locked`; expiry by clock |
| Membership | (workspace, user), kept forever | role `member` or `administrator`, state, version | `active` ↔ `suspended`; `active` or `suspended` → `removed`; `removed` → `active` only by a new invitation |
| Session | verifier of the session secret | serial number, user, epoch at opening, identity, opened, idle length, idle and absolute expiry | open → `ended`; current only by the predicate below |
| Credential | credential identifier, create-only | service user, verifier, validity | `active` → `revoked`; expiry by clock |

User, invitation and credential identifiers follow the identifier rule of `modules/organizations`, duplicated with a test, and are generated by the server layer, never chosen by a client (a caller obligation; they share installation-wide namespaces). Connection, issuer and subject are opaque strings with stated length bounds, compared bytewise. The email attribute is free text with a length bound; it identifies nobody, there is no lookup by email, and no listing returns it.

Every validity is the half-open interval `[notBefore, notAfter)` on the registry's injected clock, as in `modules/auth`. The version of a membership increases on every transition, including removal and re-creation, and is never reset. A removed membership is absent for every operation except `Accept`, and `Members` does not list it. Epochs and membership versions refuse at their maximum instead of wrapping wherever a refusal is possible; the version of a membership removed by a departure and the link counter simply stop at the maximum.

## Verifiers

Every secret reaches the package only as a 32-byte verifier computed by the authentication boundary. The package rejects the zero value and, as the one algorithm-specific check, the SHA-256 digest of the empty string, and otherwise depends only on the length. It compares bootstrap, invitation and credential verifiers through one helper built on `crypto/subtle`, and finds a session by using its verifier as a map key. No returned value, listing or error contains a verifier.

Caller obligations, stated because the package cannot check them: the secret behind a verifier is generated by the boundary with at least 128 random bits and is never chosen by a person; verifiers are derived with a distinct purpose label per record type; a verifier is never accepted from a client. At this boundary the verifier is the credential: whoever can read stored verifiers and call the package is authenticated, so the store's confidentiality matters as much as a password file's. SHA-256 over a high-entropy secret is the assumed derivation; the algorithm is not selected by this ticket.

## Users, epoch and immediate blocking

A user's epoch starts at 1 and increases on every suspension, reinstatement, departure, and every identity link and unlink. Every session records the epoch at which it was opened and every access answer carries it. Suspension therefore ends every session and every access answer of the user in one write, before any cleanup; this is the "blocking version first" rule of the requirement at the level of one user. Reinstatement does not revive old sessions. Suspension is installation-wide for the user (provisional). `SuspendUser` and `DepartUser` also accept a service user, so that a service account can be blocked when its workspace has no administrator; a service user suspended through the Installation handle carries an operator block that only `ReinstateUser` lifts, and `SuspendUser` also places the block, with a new epoch, on a service user that a workspace administrator has already suspended. The block binds the record: an administrator may still retire a blocked account and create another one (stated, not forbidden).

`CreateUser` creates a human user together with exactly one initial identity in the same write, and is refused until the bootstrap is consumed. Departure is terminal: the user's memberships are removed, identities unlinked and sessions dead in the same write, and the user's triples can never be linked again, so a returning person is a new user (provisional; re-hire and external-collaborator policy are the owner's). The kind of a user never changes and no operation transfers a human's memberships, sessions or anything else to a service user; whatever the departed user owned in other modules stays attributed to that user and must be reassigned explicitly there.

## Current session

One predicate is used by every operation that names a session: the session exists, is not ended, `opened <= now < idle expiry`, `now < absolute expiry`, its identity is linked, and its user is an active human at the epoch recorded at opening.

`OpenSession(connection, issuer, subject, verifier, idle, absolute)` resolves the identity, requires an active human user, and rejects (never clamps) a non-positive length, a length over its configured maximum, and an idle length above the absolute one. A verifier that names a session record is refused; once that record is deleted the verifier could be used again, which only a boundary that breaks its fresh-secret obligation would do. Only a successful `Access` moves the idle expiry, to the earlier of now plus the idle length and the absolute expiry, and never backwards. `EndSession` ends the session its verifier names by deleting its record; ending all of a user's sessions is the epoch, and the records of a user's sessions are deleted with every change of that user's epoch that kills them (suspension, departure, linking, unlinking). Records past their absolute expiry are deleted at the user's next login; those of a user who never logs in again stay (a stated limit). Rotating the session secret on a role change and recording authentication strength are deferred.

## Access answers

`Access(verifier, workspace)` answers only when the session is current and the user has an active membership in that workspace. `AccessByCredential(credential, verifier)` answers for the owning workspace only when the credential is active and inside its validity, the verifier matches, and its service user is active.

An answer is an opaque value with unexported fields: the registry instance (a random value created at construction), the installation, the workspace, the user, the kind, the epoch, the role (`member`, `administrator` or `service`), `notAfter`, and either the membership version, the session serial, the identity and its connection, or the credential identifier. Accessors expose what consumers need, including the connection, so that a later rule against bypassing an organization's enforced login path can be applied.

`Recheck(answer)` confirms an earlier answer without a session secret. It succeeds only when: the answer was issued by this registry instance; `now < notAfter`, where `notAfter` is the session's absolute expiry or the credential's expiry, and now is not before the session's opening or the credential's start; the user is active at that epoch; and then, for a session answer, all of: the membership is active with that role and version, the identity is linked, and the session is not ended; for a credential answer: the credential is not revoked. The zero answer and an answer of another registry instance fail. The idle expiry is not part of a recheck (a running job is not an idle user), and an explicit end of the session fails it (provisional: whether a job should survive logout is the owner's question).

`Recheck` is a state query, not proof of authentication. Answers are held by the server and are never accepted from a client or embedded unsigned in a token. Every interactive request calls `Access` with the session verifier. `Recheck` is only for work already started under an answer, such as a download in progress or a queued job, which keeps the answer it started with and rechecks it at its checkpoints; an answer is never cached to serve later requests, and issued tokens are outside this increment.

## Identities

`Resolve(connection, issuer, subject)` is the only way from an external login to a user. An unknown triple resolves to nothing, whatever its email.

`LinkIdentity` and `UnlinkIdentity` are Boundary operations, because the proof of an identity is the boundary's. They act on the session's own user and require: a current session opened no longer ago than the configured reauthentication window; for linking, a non-empty proof reference from the authentication boundary stating that the new triple was proven in the same flow (the proof itself is the boundary's duty); a human user. Both increase the user's epoch, which ends the session used and every other session. A triple that was ever linked can never be linked again, to anyone. Unlinking the last linked identity is always refused; only departure removes it. An identity cannot be unlinked through a session that was opened with an identity linked later than it, so that a newly added identity cannot be used to remove the older ones (provisional; migrating away from an old provider is deferred with recovery). Side effects of the epoch increase, stated: every answer of the user fails its next recheck, and every pending invitation the user issued dies. A stolen recent session together with a proof the boundary accepts can still add an identity; that risk is the boundary's to close with real reauthentication (a stated limit).

Email reassignment: when a provider gives an old address to a new person, that person arrives with a different subject, resolves to nothing and gains nothing. When a provider reuses a subject for a new person, this package cannot tell; the trust-root rules of the access contract apply (a stated limit).

## First administrator

`ArmBootstrap(verifier, validity, workspace)` succeeds once per installation and only while no user record of any state exists; like an invitation and a credential it refuses a validity that is already over. `Bootstrap(verifier, user, identity)` creates the first human user, links the identity and creates an active administrator membership in the armed workspace, all in one write, and consumes the material. A call with malformed arguments neither counts an attempt nor consumes. A wrong verifier counts an attempt; the attempt that reaches the limit locks the bootstrap, and a locked or expired bootstrap refuses the right verifier. Attempts never reset. Two concurrent calls with the right verifier: exactly one succeeds. A locked or expired bootstrap can only be cured by reinitialising the installation (provisional), so the entry point that reaches `Bootstrap` must not be publicly reachable (a caller obligation).

In this increment only the bootstrap workspace can have members: no operation creates a membership other than `Bootstrap` and `Accept`, and `Accept` needs an administrator of the workspace. Seeding an administrator for a further workspace is the same operation as recovering a workspace without one, and is deferred with recovery.

## Invitations

Only a current session of an active administrator of the workspace may invite. An invitation names the workspace, the role offered, a verifier, a validity not longer than the configured maximum, and the user it is for; that user's existence is not checked when inviting, so the operation cannot probe identifiers. Bearer invitations without a named user are not offered (provisional; they are the owner's decision).

`Accept(workspace, invitation, verifier, session)` evaluates in this order and refuses at the first failure: the session is current; the invitation belongs to that workspace, is pending and inside its validity; the session's user is the user it is for; the inviter is still an active administrator of the workspace at the epoch and membership version recorded when inviting, so an invitation never revives with a reinstated or re-added inviter; the user has no active or suspended membership there; then the verifier. Only a wrong verifier at that last step counts an attempt; the attempt that reaches the limit locks the invitation. It then creates or re-creates the membership and marks the invitation accepted in one write. A second acceptance is refused.

Every refusal of `Accept` is the same external error, with a reason kept for audit; nothing tells a caller that an invitation exists in another workspace. An invitation never links an identity and never creates a user. Reissue invalidation is deferred: a reissued invitation is a new record and the old one must be revoked by the caller.

## Memberships and the last administrator

`ChangeRole`, `SuspendMembership`, `ReinstateMembership` and `RemoveMembership` name the acting session, the workspace, the target user and the expected version. The actor must hold a current session and an active administrator membership in that workspace. An administrator may act on any membership of the workspace, including another administrator's and its own, and on a member whose user is suspended (provisional: no ceiling between administrators). A user may remove its own active membership whatever its role. A target without a membership in the workspace is refused like an unknown one. A change to the role already held is refused.

An active administrator of a workspace is an active administrator membership of an active human user. Rule: an operation is refused, with a dedicated error, when it would reduce the number of active administrators of a workspace from at least one to zero while, after the operation, the workspace would still have any active or suspended membership, the target's own included, or any service user that is not retired. It applies to role change, membership suspension and removal, self-removal, and every departure whoever calls it. Demoting or suspending the last administrator is therefore always refused; the last administrator may only be removed from an otherwise empty workspace; and a member may still leave a workspace that already has no administrator. In addition `DepartUser` is refused while the user holds an administrator membership, active or suspended, of a workspace in which no other user who has not departed holds an active administrator membership, and that workspace would keep any other membership or service user that is not retired. A suspended membership of another administrator is not a way back, because reinstating it needs an administrator; a security suspension followed by a departure therefore cannot destroy the only way back. Pending invitations are not counted: none can be accepted without its inviter being an active administrator. The actor check, the count and the write happen under one lock hold.

Exception, labelled: `SuspendUser` is a security block and always succeeds. `WithoutAdministrator()` reports every workspace that has an active or suspended membership or a service user that is not retired, and no active administrator. Putting an administrator back is an installation-level recovery that this increment does not implement; `ReinstateUser` of the suspended administrator is the only way back and is the trusted caller's responsibility.

The two roles are the whole role model of this package. What a member or an administrator may do with billing data is decided by grants in `modules/auth`; role bindings and their expiry live there. `administrator` here means only "may manage memberships, invitations and service accounts of this workspace".

## Service accounts

`CreateServiceUser` needs an administrator's current session and creates a service user for that workspace with its creator and a named human owner who must be an active member of the workspace at creation (provisional; the account outlives its creator and its owner, and the owner is a record for accountability, not an authority). A service user has no identity, session, invitation or membership; it is never an administrator and never counts as one. The owner must hold an active membership as an active human user when the account is created. Administrators of the owning workspace list, suspend, reinstate and retire it (retire is its departure); each state change requires the service user's expected epoch and increases it. `ReinstateServiceUser` is refused while the operator block is set. An operation on a service user or credential of another workspace is refused like an unknown one.

A credential has an identifier, a verifier and a mandatory validity bounded by the configured maximum. Rotation is a new credential followed by revocation of the old one. Sessions and credentials never lock on a wrong verifier; a wrong verifier is simply refused.

CSP collection accounts and other upstream secrets are not users and do not appear in this package.

## Listing

`Users(after, limit)` returns identifier, kind, state, epoch and, for a service user, its workspace. `Members(session, workspace, after, limit)` returns user, role, state, version and the user's state, and `ServiceUsers(session, workspace, after, limit)` returns identifier, state, creator and owner; both require an administrator's current session. Pages are in identifier order with a bounded page size. No listing returns an email or a verifier. Server-side filters beyond the workspace are deferred.

## Limits and errors

Limits are required and positive at construction with no default: maximum validity of the bootstrap, an invitation and a credential; maximum idle and absolute session length; the reauthentication window; wrong-verifier attempts; pending invitations per workspace; sessions per user; credentials per service user; page size. Each count limit counts only current records — sessions current by the predicate, invitations pending with now before their end and whose inviter check would still pass, credentials active with now before their end — and at the limit the new record is refused (provisional: refuse, never evict). Records that have not started yet are counted. Terminal invitation and credential records are kept, and their growth is bounded only by the caller's rate limiting (a stated limit).

Errors carry no identifiers. Creating a record under an identifier that is already used anywhere in the installation is a conflict, which tells a workspace administrator that the identifier exists; this is acceptable only because identifiers are generated by the server layer (a stated limit). The external errors are: malformed request, refused (one undifferentiated error for everything a caller must not be able to tell apart, with an audit reason retrievable from the error by the server), version or epoch conflict, last administrator, limit, and unavailable. A registry must be built by its constructor; an unconstructed one refuses every call without a panic. Workspace existence and state in `modules/organizations` are not checked (a stated limit).

## Relationship to `modules/auth`

`modules/auth` needs a subject's kind, state and epoch and is given them. This package is where they are true, and its epoch rules fit those of `modules/auth` (start at 1, never decrease, a higher epoch on reactivation, an immutable kind). In this increment the two are not wired: a suspension here does not by itself deny a decision there. Binding them is [HB-173](https://linear.app/hybrid-billing/issue/HB-173/bind-identity-state-sessions-and-memberships-to-permission-decisions), created before this ticket merges and a stated prerequisite of the W-33 command plan; it records the known gaps: no departed state in `modules/auth`, a removed membership that does not change the user epoch, the support-session field that is not a login session, and the separate subjects of `modules/customeraccess`, which a suspension here does not block. Until then neither package may be exposed alone.

## What is not decided here

SSO/IdP product; allowed authentication methods per organization; LDAP path; login identifier scope; MFA scope and step-up; self-signup; account recovery, including recovery of a workspace without an administrator, of a locked bootstrap and of a user without a usable identity; external collaborator and re-hire policy; every duration; the verifier algorithm; password storage; refresh tokens; the database.

Provisional choices, conservative and each refusing rather than guessing: two roles without a ceiling between administrators; invitations always for a named user and dying with any change of the inviter's authority; a triple that can never be linked twice, including after departure; linking and unlinking by a recent session plus a proof reference, each ending all sessions; a recheck that fails on an explicit session end and ignores idle time; sliding idle renewal; refusal at the session, invitation and credential limits; member self-removal; one workspace per service user with a recorded owner; bootstrap that locks for good; installation-wide user suspension overriding the last-administrator rule; an operator block on a service user that a workspace administrator cannot lift; no unlinking through a later identity; sessions and credentials that never lock; only the bootstrap workspace having members, so that when its last administrator leaves it empty the installation has no usable workspace.

This ticket does not complete W-42: authentication methods, MFA, recovery (the acceptance case "recovery bypass" is covered here only by negative tests that no call sequence re-arms the bootstrap, relinks a triple, reactivates a departed user or creates a membership outside `Bootstrap` and `Accept`), bulk invitation and revocation, audit records, the binding to `modules/auth`, persistence and the acceptance cases SEC01, EXT02, REV05, AUD-AR-09, AUD-DB-08 and AUD-QA-04 remain NOT_RUN at system level.

# Allowed files

`modules/identity/*` (new); `docs/features/HB-172/*`.

# Verification

Literal table tests are the primary evidence. Fixtures use obviously synthetic identifiers and verifiers, and error text is asserted free of identifiers.

- Records: identifier and string bounds; create-only identifiers; immutable kind; every state transition of the table and every other transition refused; malformed and empty-string verifiers.
- Bootstrap: before arming; malformed arguments not counting; wrong verifier, the right verifier on the last allowed attempt and after the lock; expiry on the exact instant; success; a second call with any arguments; arming when any user record exists; `CreateUser` before the bootstrap is consumed; two concurrent calls with the right verifier under the race detector.
- Users: suspend, reinstate and depart with a stale epoch and repeated; a missing operator reference or reason; departure ending memberships, identities and sessions in one write; identifiers not reusable.
- Identities: an unknown triple with a known email; email reassignment; linking with a recent session and proof, with an old session, without proof, by a service user, with a triple linked to another user, with a triple linked before; the epoch increase ending the session used; unlinking one of two, the last one, and an older identity through a session of a later one (refused).
- Invitations: reuse; the right verifier in another workspace refused indistinguishably; expiry on the exact instant; a wrong verifier counting only at the last step (not for another user, a dead session or another workspace); the lock and the right verifier afterwards; an invitation for another user; the inviter suspended, demoted, removed, suspended and reinstated, removed and re-added; a user who is already a member; a service user or a suspended user accepting; revocation; the pending limit counting only current invitations, including ones that have not started and excluding ones whose inviter changed; inviting by a member, by an administrator with a suspended membership, by a suspended user and by an administrator of another workspace.
- Memberships: each operation by an administrator, by a member, by an administrator of another workspace, with a stale version, on an absent target, to the same role; the version across removal and re-invitation; the last-administrator rule for demotion, suspension, removal, self-removal and departure, each also with a second administrator present (allowed) and with a second administrator suspended as a user or as a member (refused); the sole member-administrator demoting itself and suspending its own membership (refused); the last administrator leaving an empty workspace (allowed) and one with a service user (refused); a security suspension followed by a departure (refused); a member leaving a workspace without administrator (allowed); security suspension of the last administrator and the report.
- Sessions and access: an acting session named only by its verifier; every condition of the predicate broken in turn; the idle expiry moving forward on success, not on refusal, never beyond the absolute expiry and never backwards; a clock moving backwards; rejected lengths; the session limit after suspension and reinstatement.
- Recheck: after suspension, reinstatement, departure, role change, membership suspension, removal and re-invitation, link, unlink, explicit end, past `notAfter` and with the clock before the opening; the zero answer and an answer of another registry instance fail; a removed member whose identity is still linked fails; service answers after credential revocation, credential expiry, service suspension and retirement.
- Service accounts: creation and each operation by an administrator and the refusals above; an owner who is not a member; a service user that cannot be invited, hold a session, link an identity, be an administrator or count as the last one; credential expiry on the exact instant, rotation, a wrong verifier never locking, limits counting only current credentials; operations from another workspace; a stale expected epoch; an operator suspension that a workspace administrator cannot lift and that `ReinstateUser` lifts; service answers after the last administrator's security suspension and then after the operator suspends the service user; the creator's departure leaving the account in place with nothing of the human passing to it.
- Listing: fields, order, cursor, page bounds, the administrator requirement.
- Handles: each handle exposes only its operations (a compile-time fact stated in the record).
- An unconstructed registry refuses every call without a panic.
- Recovery-equivalent sequences: no sequence of calls re-arms the bootstrap, relinks a triple, reactivates a departed user, or creates a membership outside `Bootstrap` and `Accept`.
- Properties with a fixed-seed generator and a reference model written in the test that shares no helper with the package, holds plain records and answers by scanning; clock advances, including to exact boundaries, are generated operations. After every operation: `Access`, `AccessByCredential` and `Recheck` of every answer ever issued equal the model; the workspaces reported by `WithoutAdministrator` equal the model's and each lost its last administrator by a security suspension; no session opened before a suspension, departure, link, unlink, end or expiry is current afterwards. The test fails when a category of operation or refusal is rarely reached.
- Concurrency under the race detector: callers of `Access` during a suspension never get an answer after the suspension has returned.
- Mutation check by a script, results in the verification record, covering at least: each condition of the session predicate, `Access`, `Recheck`, `Accept` and `AccessByCredential`; each actor check; the last-administrator count and each of its callers; every epoch and version increment; every expiry comparison; the attempt rules; the workspace match of an invitation; the kind checks; each limit and what it counts; the instance check; and each copy.
- `gofmt -l`, `go vet ./...`, `go test -race ./...`.
- Independent review of the actual change.

# Deferred

Authentication methods and protocol adapters; password verifiers, MFA, step-up, authentication strength and recovery; refresh tokens; session rotation on role change; reissue invalidation; installation operator role; seeding and recovering a workspace administrator; bulk invitation and revocation with chunks; server-side filters; audit records and commands (W-33); binding to `modules/auth`, `modules/customeraccess` and `modules/organizations` (HB-173); persistence.

# Design review resolution

The independent design review of 2026-11-13 (read-only, nothing executed) returned FAIL (narrow) without a blocker: 8 HIGH, 11 MEDIUM, 4 LOW, all fixable by wording, and stated that no owner decision blocks implementation when the fail-closed forms are adopted. They were adopted as written in the sections above.

| Finding | Resolution |
| --- | --- |
| D1 recheck undefined for service access and unbounded in time | Answers carry `notAfter` and a session or credential reference; recheck conditions stated for both kinds |
| D2 forgeable answers | Opaque value with the registry instance; stated as a state query held by the server |
| D3 linking by any live session | Recent session, proof reference, epoch increase; the last identity never unlinked |
| D4 no first administrator for a further workspace | Stated: only the bootstrap workspace has members; deferred with recovery |
| D5 first identity of a created user | `CreateUser` links one initial triple; refused before bootstrap |
| D6 last-administrator rule | Restated as a reduction from at least one to zero with remaining records; every departure; the report widened |
| D7 bearer invitations | Every invitation names its user |
| D8 version of a re-created membership | Never reset |
| D9 recovery-equivalent paths | Connection on the answer; negative sequence tests; the acceptance case named |
| D10 invitations reviving with the inviter | Inviter epoch and membership version recorded and compared |
| D11 attempt counting and check order | Order fixed; only the last step counts; one external refusal |
| D12 what limits count | Current records only; refuse at the limit |
| D13 verifier handling | Caller obligations and package rules stated |
| D14 current session | One predicate; lengths rejected, not clamped; half-open |
| D15 identifiers chosen by clients | Server-generated, stated; cross-workspace operations refused as unknown |
| D16 trusted-caller surface | Three handles; operator reference, reason and expected epoch |
| D17 service account owner and listing | Creator, named owner, listing, suspend and retire |
| D18 relationship to `modules/auth` | HB-173 created and cited with the known gaps |
| D19 unlabelled decisions | Added to the provisional list |
| D20 to D23 | Smaller rules, deferred items and fixture rules stated; verification amended |

# Design re-review resolution

The narrow re-review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker, 2 HIGH, 8 MEDIUM and smaller items, all wording, and judged the document implementable once they were folded in; no owner decision blocks implementation. All were adopted as written in the sections above: service users reachable by the operator's suspension and departure with a block a workspace administrator cannot lift (E1); the last-administrator rule counting the target's own membership (E2); the recheck condition stated per kind (E3); the actor and workspace rule for every workspace operation (E4); acting sessions named only by verifier (E5); expected epochs on service-user state changes (E6); departure after a security suspension (E7); records that have not started counted against limits (E8); `Recheck` limited to work already started (E9); linking and unlinking moved to the Boundary handle with the later-identity rule and its residual risk stated (E10); stale records (E11); and the smaller rules of E12. The folded text was not reviewed a third time; the change review of the implementation is the check on it.

# Change review resolution

The independent change review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker, 1 HIGH, 4 MEDIUM and 9 LOW. It found no sequence that yields a forbidden access, answer or recheck and no missing or wrong-workspace actor check.

- C1 (HIGH): a security suspension followed by a departure could still strand a workspace when the only other administrator membership was suspended, because the seat rule counted a suspended membership as a way back. The code followed the plan's wording, and the reference model had copied the same reading. The rule now counts only another active administrator membership; the plan sentence, the model and a literal test were corrected.
- C2: the operator could not block a service user that an administrator had already suspended; `SuspendUser` now does.
- C3: conflicts on creation reveal that an identifier exists; stated as a limit.
- C4: the property test drew random numbers inside a map iteration and was not reproducible; it now iterates sorted keys and gives the same counts on every run.
- C5: the concurrency test could pass without concurrency; every reader now succeeds once before the suspension.
- C6 to C14: vacuous cases replaced by real ended, idle, suspended and reinstated sessions; session records deleted when they die; counters through the overflow guard; a paging bound that cannot overflow; an expired bootstrap validity refused; the missing literal cases added; the equivalence statement for the validity mutation corrected; deviations listed; helper names corrected.
