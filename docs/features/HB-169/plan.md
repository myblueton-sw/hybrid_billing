---
type: ImplementationPlan
title: Permission registry with scoped grants, delegation and support sessions
description: Defines the first authorization model of work item W-03, a single registry of scoped, conditional grants with delegation chains, support sessions, client intersection and approver independence, evaluated by pure rules and held in memory with expected versions.
status: draft
---

# Scope and ownership

[HB-169](https://linear.app/hybrid-billing/issue/HB-169/w-03-permission-registry-with-scoped-grants-delegation-and-support) is work item W-03 (requirement HB-03) with detailed contract AR-09 and the permission-intersection rule of QA-04. It is the third increment built under the owner's instruction of 2026-11-12. Base: main `8dccf40`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, as permissions and tenant boundaries require. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-12:

- Requirement HB-03: viewing, cost viewing, policy change, approval, issuance, download and delivery are separate. The customer policy owner and the MSP selling policy owner, and the author and the approver, are distinct. One PermissionRegistry connects scope with customer, legal entity, contract, field and period conditions; API, workers, caches, files, search and AI use the same policy. Current permission is rechecked at job start, chunk boundaries, external submission and download resume; a permission version change invalidates waiting jobs and caches. Acceptance: no row, field, count or file leaks under ID tampering, delegation expiry, permission revocation, connection pool reuse or group move.
- AR-09: a delegation links the original delegation, whether re-delegation is allowed, a scope ceiling and a period. Support access records the real actor, target customer, reason, approved scope and expiry separately and is revoked at the end. Customer account sharing and approver impersonation are not used, and support permission is never promoted to billing approval. Acceptance AUD-AR-09: re-delegation that is not allowed, a widened scope and an expired parent are blocked; the support audit names the real actor; a support session's impersonation of an approver is refused; a substitute with explicit approval permission approves only under their own identity.
- QA-04: a delegated client's effective permission is the intersection of the current user's and the client's permissions; approval request and human approval permission are separate.
- The command contract names the errors `permission_denied` and `self_approval_denied`, and requires failing closed with `authority_unavailable`.
- The access contract: caller-submitted scope is never authority; the server binds current actor and client, workspace, customer, resource and field grants, policy revision and period; jobs recheck authority before work, at checkpoints and before output release.
- Existing code: `modules/customeraccess` authorizes customer usage reads and exports with subjects, epoch, expiring field-level grants per company and tenant, and an authority stamp that is compared again before results are released. `modules/organizations` (HB-168) holds workspaces and parties and states that it is not an authorization source. There is no general permission model, no delegation and no support session.
- Open decisions of HB-03: SSO provider, approval amount ceilings and separation-of-duties exceptions, cross-group viewing delegation policy. Identities and sessions are W-42; commands and API are W-33.

# Decision

One Go package, `modules/auth`, pure decision logic over the standard library with an in-memory registry. It decides; it does not authenticate. A caller presents an already authenticated principal, an action and a request, and receives a decision with an external code, an audit reason and an authority stamp. No entry point is wired to it in this increment.

## Principals

| Element | Rule |
| --- | --- |
| Subject | Installation-wide opaque ID with kind `human` or `service`, state and epoch. The kind never changes, the epoch never decreases, and a suspended subject becomes active again only with a higher epoch. A delegation or session survives an epoch change of its subject (undecided; flagged to the owner). A suspended subject, an unknown subject, or a principal whose epoch is not the subject's current epoch is denied everything |
| Client | A separate ID namespace with state, version and its own grants. A client is never a subject, delegate, delegator, session actor or approver. A service acting by itself is a subject of kind `service`, never a client without a user |
| Principal | Installation, subject ID and epoch, optionally a client ID, optionally a support session ID. The installation must be the registry's. A principal that names both a client and a support session is denied. There is no on-behalf-of parameter anywhere in the API |

## Time

The registry owns the clock, injected at construction as in `modules/customeraccess`. `Decide`, `Recheck`, `CheckApproval` and `RecheckApproval` take no time. Every validity is the half-open interval `[notBefore, notAfter)`; a record without a start, without an expiry, or with `notAfter` not after `notBefore`, is rejected at registration.

## Catalogue

Actions are a closed set: `view`, `view_cost`, `change_customer_policy`, `change_selling_policy`, `request_approval`, `approve`, `issue`, `download`, `deliver`, `approve_support_session`. An unknown action is rejected. No action implies another. `approve` and `approve_support_session` are human-only: a grant for either may belong only to a human subject, never to a service subject or a client.

Resource kinds are a closed set supplied at construction (tests use a synthetic set such as `usage_statement`, `allocation_pool`, `selling_amount`, `adjustment`, `trust_policy`). Every grant and every request names exactly one kind; there is no "any kind". An `approve` grant for one kind therefore approves nothing of another kind.

Fields are a catalogue supplied at construction, each with a class, `cost` or `general`. Registration rejects an unknown, repeated or wildcard field. `view` grants may list only general fields and `view_cost` grants only cost fields. A `download` grant may list either; downloading a cost field additionally requires `view_cost` of that field, and downloading a general field `view` of it (a conjunction, provisional). The field catalogue is not keyed by resource kind, which is a limit while the catalogues are synthetic. The actions `view`, `view_cost` and `download` are field actions: a request must name at least one field. Any other action must name none.

## Grants and requests

A grant has an ID, the subject or client it belongs to (exactly one), an action, a resource kind, a scope, for field actions a non-empty field set, a business period, and a validity. The scope is a workspace and three selectors, customer, legal entity and contract, each explicitly `all` or `one(id)`. The period is explicitly `any` or `[fromMonth, toMonth)`. The zero value of a selector or of a period is invalid and rejected, so an omitted customer can never mean every customer and an omitted period can never mean every month.

Record identity: subjects, clients, grants, delegations and sessions have separate namespaces. Grants, delegations and sessions are create-only; an existing ID is a conflict, also after revocation. Their only transition is one revoke or end with an expected version; closing a closed record is a conflict. Records are never deleted. A grant's subject or client must exist and be active at registration.

A request is one action, one resource kind, one resource (a workspace, and optionally customer, legal entity, contract and a business month) and the field names. A request without a workspace is denied; installation-scope grants are deferred. A selector `one(id)` matches only a resource that has that dimension with that ID; `all` matches any, including absent. A bounded period matches only a resource with a month inside it.

The resource must be built by a trusted resolver from stored records, with the workspace taken from the authenticated context; per-dimension matching does not verify that a contract belongs to the named customer. `Decide` takes the resolved resource from its caller in this increment, and no entry point exists; binding to `modules/organizations` is a stated limit.

Allow for a subject is defined per single-field request. A non-field action is one such request without a field. A field request is allowed if and only if every one of its fields is allowed; each field may be allowed by a different grant or a different delegation. One single-field request (action, field) is allowed directly by one grant valid now with the same action and kind, a matching scope and period, that lists the field. For `download` of a field, both `download` of the field and the class view of the field (`view` or `view_cost`) must be allowed, each evaluated on its own, directly or through a delegation. Coverage is never combined across resources, kinds or periods: a request for several resources or months is several decisions that must all allow.

`Decide` never allows `approve` or `approve_support_session`; it denies them with the audit reason `approval_check_required`. `CheckApproval` is the only path that can allow `approve`, and support-session registration the only consumer of `approve_support_session`.

## Delegation (AR-09)

A delegation has an ID, delegator and delegate (two different active human subjects; human-only is provisional), terms (one action, one kind, a scope, fields, a period), a validity, a flag allowing re-delegation, and optionally a parent delegation. It is immutable except for revocation. `approve` and `approve_support_session` are not delegable (provisional and conservative; AR-09's substitute approver therefore needs an own direct `approve` grant, and whether approval may ever be delegated is the owner's decision). The server derives the backing; the caller cannot choose it.

Registration is a conservative hygiene check, not the safeguard:

- without a parent, for each field of the terms (or once for a non-field action) one direct grant of the delegator, valid now, with the same action and kind, the same workspace, each selector containing the terms' selector, a period containing the terms' period, and listing the field. Selector containment is `one(x)` within `one(x)`, `one(x)` within `all`, `all` within `all` only. A bounded period lies within `any`; `any` lies only within `any`. No union across grants;
- with a parent, the parent is valid now and not revoked, its delegate is this delegator, it allows re-delegation, the child's action and kind equal the parent's, and its scope, fields, period and validity lie within the parent's;
- a delegator who holds something only through a delegation therefore cannot register a delegation without naming that parent; authority through a client or a support session is not delegable by construction, because backing is direct grants or a parent;
- no subject appears twice in a chain, and a chain is at most four delegations long;
- provisional conservative default for the open cross-group question: the delegate must hold at least one valid direct grant in the same workspace, at registration and at every decision.

Nothing is materialized. At decision time a delegate is allowed a single-field request through a delegation only if every delegation from it up to the root is valid now, not revoked and covers the request, every delegation above the first allows re-delegation, every delegator and delegate in the chain is an active human whose delegate still holds a valid direct grant in the workspace, and the root delegator is allowed the same request now by direct grants. For a `download` delegation the root must directly hold both conjuncts for the field. A parent's expiry or revocation, a suspended subject in the chain, or the loss of the root's grant therefore ends every descendant at the next decision without cleanup. A delegation is inert whenever its root's direct grants do not allow the request, and it is live again when the root is granted again (labelled consequence). The decision returns the sorted IDs of the delegations used; the actor is always the principal's own subject.

## Support sessions (AR-09)

A support session has an ID, the real actor (an active human subject), a workspace and one customer (never `all`), a reason, approved terms limited to the action `view` with a kind and general fields (provisional; cost viewing and download in support are owner decisions), a validity and the approving subject. Registration rejects a cost field, an empty field set, an unknown kind, a missing customer or reason and a service or suspended actor.

The approver is passed as a principal: the registry's installation, an active human, the current epoch, no client and no session. The approver is not the actor and is not a delegate of the actor, meaning that no unrevoked, unexpired delegation chain of any action has the actor as delegator at any level and the approver as delegate. The approver must hold a direct `approve_support_session` grant that matches the request (kind of the session, resource of workspace and customer without legal entity, contract or month); a grant narrowed to one legal entity, one contract or a bounded period therefore does not qualify. Approval is point-in-time (provisional): the later revocation of the approver's grant or the approver's suspension does not end the session; ending is explicit or by expiry.

A decision whose principal names a session uses the session's terms only: the actor's own grants neither extend it nor are required. It is allowed only for `view` of the session's kind, in the session's workspace and target customer (a resource without a customer is refused), for fields in the approved set, while the session is valid, not ended, and its actor is the principal's active subject. Every other action is refused in a session with the reason `support_scope`, except the human-only actions, which `Decide` refuses everywhere with `approval_check_required`. The same subject outside a session acts under its own grants, including approval through `CheckApproval`. Ending a session takes effect at once. Allow and deny both return the actor and the session ID for the audit record a later layer writes.

## Client intersection (QA-04)

When a principal carries a client, the decision is allow only if the subject is allowed (directly or by delegation) and the client is active and allowed by its own direct grants for the identical request, including both conjuncts of a download. A client never holds or satisfies `approve` or `approve_support_session`. Clients may otherwise be granted any action, including `issue`, `deliver` and the policy actions (provisional; a policy question for the owner).

## Approver independence

`CheckApproval(approver principal, author subject ID, request)` supplies only the independence primitive; the approvals module will own approval subjects, ceilings and target digests. Because `approve` is not delegable, it allows when the request's action is `approve`, the approver is a current active human without client or session, the author is a known subject, the approver holds a direct `approve` grant that matches the request, and the author is not the approver. Codes: a non-`approve` action, an approver who is not current or carries a client or session (one shared reason), a service approver, an unknown or empty author and a missing grant are `permission_denied` with their audit reasons; `self_approval_denied` is returned only when the approver holds the grant and is the author. The author must be read from the stored record by the caller's server-side code. `RecheckApproval(stamp, approver, author, request)` re-evaluates and compares the stamp, whose digest binds the author. Limits until identities exist (W-42): one subject per human is assumed; the check is necessary, not sufficient; no exception path exists.

## Decision result, codes and stamp

A decision carries: allowed or not; the external code, `allowed` or one of the three refusals `permission_denied`, `self_approval_denied`, `authority_unavailable` (the only values a caller-facing layer may show); a detailed reason from a closed list for audit only; the actor subject, client, session and delegation IDs; and on allow a stamp.

The stamp is the registry instance (a random nonce created at construction), the registry generation, a flag for approval stamps, and a digest of the principal, action, kind, resource, sorted fields and, for approvals, the author, computed with the decision under one lock hold. Record IDs are audit basis only and are not compared. The generation increases on every write and on every availability change. `Recheck(stamp, principal, request)` re-evaluates now, which is what detects expiry by time, and additionally requires an equal stamp, which detects a change that was made and undone and any permission version change; it never accepts an approval stamp. A stamp presented with a different request or to another registry instance fails. A decision that would be allowed but whose stamp is not the current one is refused with the audit reason `stamp_not_current`, so a caller can tell "decide again" from a refused principal. When the registry is marked unavailable every decision is denied with `authority_unavailable` and session registration fails.

Labelled consequence (provisional): any write anywhere in the registry invalidates every stamp in flight. This is fail-closed and matches HB-03's "a permission version change invalidates", but it is a liveness risk under frequent writes; generations per workspace or per subject are a follow-up. On this point the model differs from `modules/customeraccess`, whose generation changes only with availability and whose stamp carries record versions.

## Registry

In memory, one installation, one mutex, a version per record, expected-version writes. Its mutation methods are not an authorization boundary in this increment: whoever holds the registry can write, and the layer that exposes writes must itself be authorized, which is a later work item. Stored terms are copies of the caller's values, taken before validation. The package has no read accessor in this increment: record versions and the stored session reason, approver and expiry cannot be read back, which the audit layer of a later work item needs (a stated limit). A registry must be built with `NewRegistry`; an unconstructed one refuses every decision with `authority_unavailable`, refuses subject, client and session writes with an unavailable error, accepts no grant or delegation, and cannot be made available. Error and reason text carries no identifiers.

## Relationship to `modules/customeraccess`

`modules/customeraccess` remains the sole authority for the customer usage read and export path. No code path may combine the two models with "or". The semantics are aligned on purpose in mandatory expiry, epoch, per-scope field coverage, fail-closed unavailability and recheck; the stamp differs as stated above. Its actions map to `view` and `download` of a usage kind. Folding the two together is a follow-up ticket.

## What is not decided here

SSO provider, approval amount ceilings, separation-of-duties exceptions, cross-group viewing delegation, who may approve support sessions and whether support may view cost or download, whether an author's delegate may ever approve, maximum validity durations, the real field and resource-kind catalogues, roles as named bundles, membership state, and the database. Installation-scope grants and whether clients may issue, deliver or change policy are also undecided. Provisional choices, each labelled above: the action names, the two policy actions, the download conjunction and its root rule under delegation, non-delegable `approve`, human-only delegation, the suspended-delegator rule, the same-workspace delegate rule, clients never approving, support limited to `view`, point-in-time support approval, the explicit period with month granularity, the chain length, and the registry-wide generation in the stamp.

This ticket does not complete W-03: the query boundary for every read path, database roles and row-level context, connection-pool hygiene, the effect of workspace, party or group changes on authority, and the acceptance cases SEC01, GH04, EX02, EXT02, REV05, AUD-AR-09, AUD-QA-04 and AUD-QA-08 remain NOT_RUN at system level.

# Allowed files

`modules/auth/*` (new); `docs/features/HB-169/*`.

# Verification

Literal table tests are the primary evidence; they share no helper with the implementation's decision code.

- Catalogue and registration: unknown action, kind or field; a cost field on a `view` grant and a general field on a `view_cost` grant; zero-value selectors; missing or inverted validity; field actions without fields and other actions with fields.
- Direct decisions: each action allowed only by its own grant; kind mismatch (an `approve` grant for one kind against another); workspace, customer, legal entity and contract mismatch; `one(c)` against a resource without a customer; `all` selectors; field coverage by one grant and by two grants of the same scope; a field covered only for another customer or another month; period bounds; validity bounds with the injected clock moved; the download conjunction for cost and general fields; the same party ID in two workspaces.
- Subjects and clients: suspended subject, wrong epoch, wrong installation, suspended client, a service subject requesting `approve`.
- Delegation, AUD-AR-09: a valid delegation; registration without a parent by a delegator who holds only through a non-re-delegable delegation; re-delegation not allowed; widened scope, field set, period and validity; a different action or kind than the parent; expired and revoked parent; suspended delegator; root delegator losing the direct grant after registration; chain length; a subject twice in a chain; delegator equal to delegate; re-delegating `approve`; a delegate without a direct grant in the workspace.
- Support: view inside the approved terms; refusal outside the customer, for a resource without customer, outside the fields, after expiry by clock, after ending, with another subject as actor; every other action refused although the actor holds it directly; the same actor approving outside the session; registration with the actor as approver, with an approver who is the actor's delegate, with an approver lacking `approve_support_session`, with a service approver, with customer `all`; a session combined with a client; actor and session returned on allow and on deny.
- Client intersection: both hold, only the user, only the client, differing field sets, `approve` through a client.
- Approver independence: the same subject with both permissions (`self_approval_denied`); a different independent approver; an approver without `approve` (`permission_denied`); unknown or empty author; a service approver; an approval recheck.
- Stamp: equal on unchanged state; fails after each of a grant revocation, a delegation revocation, a parent's expiry by clock, a subject suspension, an epoch change, a session end, an unrelated write (generation), a change that is made and undone, reuse for another request, and presentation to another registry instance.
- Properties with a fixed-seed generator that fail when a category is rarely reached: every decision equals an independent reference model written in the test (direct grants, per-field paths, the download conjunction and the root rule); allow for a field set implies allow for each single field; revoking a grant or a delegation or suspending a subject never turns deny into allow; grants written in another workspace never change the allow bit; a client principal's decision equals the conjunction of the subject's reference decision and the client's own direct grants. A fuzz target checks that a registry without grants never allows.
- Concurrency under the race detector: readers never observe an allow for a grant after its revocation has returned, and stored terms do not alias the caller's slices.
- Mutation check by hand, listing each mutated line and the outcome in the verification record: the selector match, the period and validity comparisons, the field loop, the kind check, the parent and re-delegation checks, the root-delegator check, the session action restriction, the client conjunction, the download conjunction, author and approver equality, the human-kind checks, the epoch comparison, the active-chain check, the delegate-workspace check, the session customer check, and the stamp comparison.
- `go vet ./...`, `gofmt -l`, `go test -race ./...`.
- Independent review of the actual change.

# Deferred

Identities, sessions, SSO, LDAP and service credentials (W-42); commands, HTTP and the audit record of decisions (W-33); roles as named bundles; amount ceilings; binding to `modules/organizations`; replacing or folding in `modules/customeraccess`; database roles and row-level security; notification permissions (QA-08).

# Design review resolution

The independent design review of 2026-11-12 (read-only, nothing executed) returned FAIL without a blocker: 7 HIGH, 7 MEDIUM, 4 LOW. Resolution in the sections above:

| Finding | Resolution |
| --- | --- |
| F1 re-delegation bypass | The server derives the backing; a parentless delegation needs direct grants only; delegated authority requires its parent |
| F2 caller-supplied time | The registry owns the clock; half-open validities |
| F3 one `approve` for everything | A mandatory closed resource kind on every grant and request |
| F4 nominal cost separation | A field catalogue with classes; class rules per action; the download conjunction |
| F5 no principal kind | Human and service subjects; clients in their own namespace; human-only approval |
| F6 unset means any | Explicit `all` or `one` selectors with an invalid zero value |
| F7 independence defeated by delegation | An authority path without the author as delegator is required |
| F8 materialized delegated grants | Computed pointwise at decision time; immutable records |
| F9 support authority source | Session terms only; approver qualification; audit data on deny |
| F10 cross-workspace delegation | Provisional same-workspace rule, labelled |
| F11 stamp definition | Instance nonce, generation, request digest, one lock hold (record versions dropped in the re-review, N5) |
| F12 `administer_access` | Replaced by `approve_support_session`; registry writes stated as ungated |
| F13 reasons and codes | Three external codes; detailed reasons for audit only |
| F14 resource binding | Trusted-resolver requirement stated as a limit; installation on the principal |
| F15 empty field requests | Invalid |
| F16 support actions | `view` only, provisional |
| F17 overlap with approvals | `auth` supplies the independence primitive only |
| F18 mutation check | Named list, recorded per line |

# Design re-review resolution

The independent re-review of 2026-11-12 (read-only, nothing executed) found F1 to F18 resolved and returned FAIL (narrow) without a blocker: 1 HIGH, 9 MEDIUM, 6 LOW, all rule gaps fixable by wording. Its corrected rules were adopted as written, with one deliberate difference: `approve` was made non-delegable, the more conservative of the two options the reviewer named. That choice is provisional and is the owner's to confirm or change. The sections above describe the adopted rules; the verification list above was amended with the reviewer's missing cases and mutation additions.

| Finding | Resolution |
| --- | --- |
| N1 `Decide` could allow `approve` | `Decide` always denies human-only actions; approval grants belong to human subjects only; `CheckApproval` binds the author in its stamp; `RecheckApproval` added and `Recheck` rejects approval stamps |
| N2 fields across paths, download under delegation | Allow is defined per single field and per conjunct; the root holds both conjuncts for a delegated download; clients hold both directly |
| N3 registration containment | A per-field single-grant containment check, stated as hygiene; the decision-time root check is the safeguard |
| N4 kind and resource of `approve_support_session` | The session's kind with workspace and customer; the approver is a principal |
| N5 stamp | Instance, generation and request digest; the registry-wide invalidation is labelled with a follow-up |
| N6 record identity | Separate namespaces, create-only records, revoke or end with an expected version |
| N7 omitted period | Explicit `any` or bounded, invalid zero value |
| N8 absent workspace | Denied; installation scope deferred |
| N9 timing of the delegate's workspace grant | Checked at registration and at every decision |
| N10 support-session details | Delegate-of-actor defined over live chains of any action; point-in-time approval; registration rejections listed |
| N11 `approve` delegation and re-delegation | Moot: `approve` is not delegable |
| N12 `CheckApproval` edge codes | `permission_denied` with distinct audit reasons; `self_approval_denied` only for a grant holder who is the author |
| N13 clients and non-approval actions | Listed as provisional |
| N14 redundant wording | Restated as consequences |
| N15 field catalogue not keyed by kind | Listed as a limit |
| N16 missing provisional labels | Added to the list |

The rewritten rules were not sent for a third design review before implementation; the independent change review of the implemented code and of this document is the check on them, and its result is in the [verification record](verification.md).

# Change review resolution

The independent change review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker or high finding, 5 MEDIUM and 9 LOW. It found no sequence of calls that yields a forbidden allow, a wrongly accepted stamp or a leak across workspaces or customers. Code corrections: delegation IDs sorted for every action and absent for a direct allow (C1); the reason `stamp_not_current` (C6); a closed record cannot be closed again (C7); copies taken before validation (C9); an unconstructed registry fails closed (C10); the unused grant ID computation removed (C11); availability set to its current value no longer changes the generation (C12). Document corrections are in the sections above (C8, C12). Test and record corrections (C2, C3, C4, C5, C13, C14) are in the verification record. The narrow re-review of the same day returned PASS with findings, seven LOW (R1 to R7), all corrected: writes on an unconstructed registry no longer panic (R1); a test for repeating the current availability (R2); record wording and mutation history (R3, R4); two test names made to match what they isolate, one with a strengthened assertion (R5, R6); setup errors checked (R7).
