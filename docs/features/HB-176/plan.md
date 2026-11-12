---
type: ImplementationPlan
title: Command registry and executor behind the gate
description: Defines the first command layer of work item W-33, a registry of command definitions and an executor that admits a call only through the gate with a server-resolved target and applies idempotency, the revision condition, a stable error catalogue, an audit journal and a generated provisional OpenAPI document.
status: draft
---

# Scope and ownership

[HB-176](https://linear.app/hybrid-billing/issue/HB-176/w-33-command-registry-and-executor-behind-the-gate-with-idempotency) is work item W-33 (requirement HB-33) with the command parts of QA-04. It is the seventh increment built under the owner's instruction of 2026-11-12 and depends on the gate of HB-173. Base: main `e2fb6a5`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because it defines a public contract and the path on which permissions are applied. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-13:

- Requirement HB-33: UI, CLI, AI, external consumers and plugins follow one business command and permission registry; publishing an API does not mean anonymous internet access. Proposed direction: a CommandRegistry manages command_id, scope, schema and audit and generates the OpenAPI, tool and SDK contracts; amounts and quantities are decimal strings. Bulk work returns 202 with a job identifier, reads use bounded pages with a snapshot cursor, changes use a revision condition and an idempotency hash, and a stored target manifest is passed instead of arrays of identifiers. Acceptance: one business cycle completed through the API alone with the same permissions, amounts and states as UI and AI; an unsupported capability is never returned as success. Open decisions: formal command and permission identifiers, compatible patch versioning, SDK languages, API deprecation period.
- QA-04: an Idempotency-Key is scoped to installation, client and command, and the same key with a different request hash is a 409; a change requires an expected-revision condition; an approval request and the human approval permission are separate; a permission missing on either the client or the user is refused.
- The command contract (`docs/contracts/commands/planning-actions.md`, draft): every mutation carries command identifier, idempotency key, expected revision, target identifier, a typed body and, where authority, eligibility or a deadline changes, a rationale. The server resolves installation, workspace, principal, grants and target scope; a caller-supplied workspace, actor or validation flag is never authoritative. Idempotency identity is (installation, client, command, key) plus the canonical payload digest and the initiating principal: reuse by another principal and the same key with another body conflict; the same principal, key and body returns the existing operation, with current access checked before the result is disclosed, and does not run the action again. A synchronous success returns operation identifier, target, result revision, business state, permitted evidence references and as-of; an asynchronous acceptance returns a job identifier, a technical state and status and cancellation capabilities, and is not business success. A failed authority check cannot reveal that a forbidden target exists. The contract lists error families with an HTTP mapping proposal and states that no deployed HTTP path follows from it.
- The numeric contract: an external decimal is a finite base-10 string with an optional minus sign, integer digits and optional fractional digits; exponent notation, NaN, infinity and locale forms are rejected; precision and scale limits are an open finance decision.
- Existing code: `modules/gate` (HB-173) decides only with a current identity access answer whose workspace is the request's; its approval check takes no client; its stamps fail after any change of any user or membership in the installation; no other package may import `modules/auth`. There is no command layer, no entry point and no HTTP server, and no domain module offers a business action yet.

# Decision

One Go package, `modules/commands`, standard library only, on top of `modules/gate`. It defines what a command is, admits and runs one call, and describes the catalogue. It contains no business behaviour: the module that owns an action supplies its handler. This ticket registers no real command; tests use synthetic ones. No HTTP server is added.

`modules/gate` gains `Admit(answer)`, which confirms an answer and nothing else and returns only a code (admitted, not current, or unavailable), never a decision that could be mistaken for a permission; an accessor for its clock, which is nil for an unconstructed gate; a type alias for the identity answer, so that this package imports only the gate; and `Describes`, which passes a question about the permission catalogue to the permission registry. `modules/auth` gains that one read-only method (added after the change review; it reads the catalogue fixed at construction and neither decides nor changes anything).

The package must not back an exposed path until retention and rate limits are decided ([HB-177](https://linear.app/hybrid-billing/issue/HB-177/retention-rate-limits-and-durable-storage-for-command-operations-and)); see Limits.

## Definitions

A definition is registered once, under an identifier that follows the project's identifier rule, and never changes or disappears while the process lives:

| Element | Rule |
| --- | --- |
| Permission | The action, the resource kind and, for a field action, the field names of the gate request. Exactly one permission per command (provisional; contract commands that need two are not expressible yet) |
| Approval | A flag. An approval command is decided by the gate's approval check with the author read from the stored target, never by the ordinary decision. Registration refuses the flag with an action other than `approve`, and `approve` without the flag |
| Mutation | A flag. A mutating command requires an idempotency key and an expected revision; a reading command accepts neither. An approval command is mutating |
| Rationale | A flag. When set, a non-empty rationale is required; when not set, a rationale is refused |
| Fields | The typed body: each field has a name, a type from a closed set (`id`, `decimal`, `integer`, `boolean`, `text`, `timestamp`, `enum`), a required flag, and for an enum its values. Unknown and repeated fields in a call are refused |
| Handler | Supplied by the owning module, or absent. A command without a handler is listed as unsupported and every call of it ends with `unsupported_capability`, never with success. A handler that is a nil pointer counts as absent |

Registration also asks the gate whether its permission catalogue describes the permission (`gate.Describes`: a known action, a known kind, fields exactly when the action is a field action, every field known and of the action's class) and refuses a definition it does not describe. The action `approve_support_session` is refused as well, because the gate decides it only inside its own support-session administration. This removes the known ways to register a command that every call would be denied; it is not a proof that none remains. Enum values are refused when empty, repeated or longer than the value limit, and values on a field that is not an enum are refused.

Every command has exactly one target identifier; a command over a collection targets its scope, and a creating command targets the parent scope and returns the created identifier as a result field.

Field values travel as strings and are checked for form only, purely syntactically:

- `decimal`: `^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`, the numeric contract's external form. It is not parsed into a number here; the handler applies the profile, because precision and scale are an open finance decision. `1.0` and `1.00` are different payloads for idempotency (fail-closed);
- `integer`: `^-?(0|[1-9][0-9]*)$`; `boolean`: `true` or `false`; `id`: the identifier rule; `timestamp`: RFC 3339 with an explicit offset in its strict form (two-digit fields, `T`, an optional fraction after a dot, `Z` or an offset `±hh:mm` within a day), checked by a pattern and by parsing, fail-closed like decimals: a lowercase `t` or `z` and a leap second `:60`, which RFC 3339 allows, are refused, as are a one-digit hour, a comma fraction, a fraction without digits and an offset without a colon. One instant still has several accepted spellings (`Z`, `+00:00`, `-00:00`, another offset, trailing zeros of the fraction), and each is its own idempotency payload; `enum`: one of its values; `text`: any string within the length limit.

One value-length limit covers every field type. Lengths of values and of the rationale are counted in bytes of the text as given, not in characters, so the bound on stored and digested data does not depend on the script; a Korean or Arabic text reaches the limit with fewer characters than an English one. The limits themselves are construction parameters, not product decisions.

## Handlers

- `Resolve(workspace, target)` is read-only and returns one of: found, with the target's resource dimensions (customer, legal entity, contract, month), its current revision and, for an approval command, the author of the work to be approved; not found; or unavailable. A panic of `Resolve` is `internal_error`, stores nothing and leaves the key free. A found result whose revision breaks the identifier rule is `internal_error` too, but only after the gate has allowed the call (step 5), so that a handler fault does not tell a caller without the permission that the target exists. A resolution outside the three values is treated as not found. The workspace is the one of the caller's answer; the executor, not the handler, sets the workspace of the permission request.
- `Apply(context)` receives workspace, user, client, operation identifier (mutating commands), request identifier, target, the resolved revision, the expected revision, rationale, fields and the executor's time, and returns a completed outcome (result revision, business state, result fields), an accepted outcome (a job identifier), or a named error.

A revision is an opaque string under the identifier rule, compared for equality.

Handler obligations, stated because the executor cannot check them:

- any change of a target's resource dimensions or author changes its revision;
- a handler compares the revision it was given when it reads or writes and returns `revision_conflict` if the target moved, for reading commands too; the executor's comparison is the early refusal, not the guarantee;
- a handler returns a named error only when it changed nothing; a handler that cannot tell whether it changed anything returns an error that is not named, never `dependency_unavailable`;
- `unavailable` from `Resolve` must not depend on whether the target exists;
- a target against which an operation is stored stays resolvable with its dimensions while the operation is stored, otherwise the replay of that operation is denied (a stated limit);
- a handler must not call the executor.

A named error is recognised only as the handler's own error value of the named type; a named error wrapped in another error is an unknown end. Named errors a handler may return: `unsupported_capability`, `invalid_stage`, `revision_conflict`, `prerequisite_failed`, `dependency_unavailable`, `permission_denied` (without detail, indistinguishable from the executor's own), and `invalid_request` with field names; names that are not fields of the definition are dropped, so a handler cannot return a value in their place.

What a handler returns is bounded like what a caller sends: result revision, business state and job identifier follow the identifier rule, and result fields are bounded by the field-count and value-length limits. A violation makes the outcome invalid.

## Executing one call

A call carries the answer, the client identifier taken by the caller's server-side code from the admitted credential (the empty client is its own scope value), a request identifier, the command identifier, and for the command: target, idempotency key, expected revision, rationale and body. The answer must have been obtained from the identity registry within the request, because a recheck ignores idle time (caller obligation, as in HB-173).

The steps, in this order, stopping at the first refusal:

1. Admission: `gate.Admit(answer)`. A refusal is `unauthenticated`; an unavailable authority is `authority_unavailable`. Nothing about the catalogue or any target is disclosed before this. Immediately afterwards one journal slot is reserved; if none is free, the call ends with `dependency_unavailable`, whatever kind of call it is.
2. The command exists, else `unsupported_command`. It has a handler, else `unsupported_capability`.
3. The shape of the call matches the definition, else `invalid_request` with the names of the offending fields and never their values. Request identifier, client (the empty client allowed), target, key and expected revision follow the identifier rule. A call with more fields than the definition has, or with a field name that breaks the identifier rule, is `invalid_request` naming a fixed marker instead of the name. When there are more fields than the definition has, the marker is the only field name returned: no caller-chosen name is echoed and no declared field is reported missing. The catalogue and its schemas are not confidential to admitted callers (provisional). An approval command whose call carries a client ends here with `permission_denied`, before the target is resolved and without calling the handler or the gate, because the approval check cannot consult a client (provisional; approvals through clients are the owner's decision).
4. From here a mutating call holds the executor's lock until it returns. The handler resolves the target in the answer's workspace. Unavailable is `dependency_unavailable`. For a target that is not found the executor makes the same gate call as step 5, with the command's action, kind and fields on a resource that names only the workspace (for an approval command the approval check with an empty author), maps the result as in step 5, and replaces only an allowed or `self_approval_denied` result by `permission_denied`. An unavailable authority or an answer that stopped being current therefore gives the same code whether the target exists or not (constant time is not claimed).
5. The gate decides: the ordinary decision with the command's permission and the resolved resource, or for an approval command the approval check with the resolved author. An approval target without an author is passed to the approval check with the empty author, which the gate refuses. Mapping, by code and reason, with no gate reason ever appearing in a result or in the journal: code `authority_unavailable` to `authority_unavailable`; reason `identity_not_current` to `unauthenticated`; reason `stamp_not_current` to `authority_unavailable`, because it means only that something in the installation changed during the decision and replaces allowed and denied decisions alike; code `self_approval_denied` to itself; every other refusal to `permission_denied`.
6. For a mutating command, idempotency. The identity is (installation, client, command, key). If an operation exists under it: another workspace, another initiating user or another payload digest is `idempotency_conflict`; the same workspace, user and digest returns the stored result marked as a replay, without calling the handler. Because step 5 has just run, a replay is disclosed only to a caller who is currently permitted. If no operation exists and the store has no free slot, the call ends with `dependency_unavailable`; existing keys still replay when the store is full. A stored failure of step 9 is compared exactly like a stored success.
7. For a mutating command, the expected revision equals the resolved revision, else `revision_conflict`.
8. The gate rechecks the stamp of step 5 with the recheck that matches it (ordinary or approval, same client, same author); refusals map as in step 5, so a stamp that failed because of an unrelated change is `authority_unavailable` and the caller may retry a bounded number of times. Nothing was applied and the key is free. Then a new operation identifier is drawn — 128 random bits from the system source in identifier form, never reused, and visible to the caller only in a stored operation; a failing random source ends the call with `dependency_unavailable`, nothing applied — the executor's time is read, and the handler's `Apply` runs. No slot of the operation store is reserved separately: the lock held since step 4 keeps the free-slot test of step 6 true until the operation is stored.
9. The end of `Apply`:
   - a completed or accepted outcome is stored under the idempotency identity with that operation identifier and returned;
   - a named error is returned, stores nothing and leaves the key free: a later call under that key with any body is a new call;
   - an error that is not named, a panic, an outcome together with an error, a completed outcome without a result revision or business state, an accepted outcome without a job identifier, an accepted outcome of a reading command, or an outcome that breaks the bounds above is `internal_error` without any text. For a mutating command the operation is stored as failed with an unknown effect: the same call replays `internal_error` with the operation identifier and without calling the handler, and another workspace, user or body conflicts, exactly as for a stored success (provisional, fail-closed: the executor cannot know whether the action took effect).

After `Apply` returns, the outcome is checked and the operation stored by code that calls nothing outside the executor. The journal record is written in a deferred step on every path after admission, including a panic outside the handler, which ends as `internal_error`; the lock is released in the same step after the record. A clock that fails while the record is written gives the record the zero time instead of losing it.

What the lock gives and does not give: two calls with the same key cannot both apply, and the steps 4 to 9 of mutating calls do not interleave. A change of identity or grants that completes after the recheck of step 8 has returned does not stop an `Apply` in progress; nothing stronger is claimed. A slow handler blocks every mutating call of the installation (a stated limit). Reading calls do not take the lock and store no operation.

The payload digest is SHA-256 over a version tag, the command, the target, the expected revision, the rationale and the fields in the byte order of their names, every part length-prefixed; an absent optional field contributes nothing and differs from a present empty one. The request identifier is not part of it. The initiating principal bound to an operation is the user identifier; the epoch, the session and the role are not, so the same person replays after a new login. Keys must be unpredictable per intent: another user who uses a key first makes it conflict for everyone, and a conflict reveals only that the key is in use (a stated limit that follows the contract).

## Results and errors

A result is either an outcome or exactly one error.

- Completed, mutating: operation identifier, target, result revision, business state, result fields, as-of, and whether it is a replay. As-of is the executor's time read after the recheck and immediately before `Apply`, the same value the handler receives, so that nothing that can fail lies between the end of `Apply` and storing the operation.
- Completed, reading: target, the resolved revision, result fields and as-of; no operation identifier.
- Accepted: operation identifier and job identifier, with a status that is explicitly not business success.
- An error has a stable code, the HTTP status the contract proposes, whether a retry of the same call can succeed, a next action from a closed list, the request identifier and, for `invalid_request`, field names.

| Code | Proposed status | Retry of the same call | Next action |
| --- | --- | --- | --- |
| `unauthenticated` | 401 | no | obtain a new access answer; the admitting layer decides between 401 and 403 from the identity registry's own answer |
| `permission_denied`, `self_approval_denied` | 403 | no | none |
| `revision_conflict`, `idempotency_conflict` | 409 | no; a reading command's `revision_conflict` may be retried | inspect the current state and form a new intent |
| `invalid_request`, `invalid_stage`, `prerequisite_failed` | 422 | no | correct the request |
| `unsupported_command` | 404 | no | none |
| `unsupported_capability` | 501 | no | none |
| `authority_unavailable`, `dependency_unavailable` | 503 | yes | retry later |
| `internal_error` | 500 | no | for a stored operation, inspect it by its operation identifier, which the error then carries; otherwise none |

`unsupported_command`, `unsupported_capability`, `internal_error` and `prerequisite_failed` (the contract's name is `confirmation_prerequisite_failed`, which is one prerequisite of one command) are added here (provisional).

## Audit journal

Every call that passes admission has exactly one record, reserved at step 1 and written when the call ends, whatever its end, and for a mutating call before the executor's lock is released, so that a replay is never recorded before its original; the time of a record is read just before it is appended, outside the registry's lock, so for calls that are not both mutating the times of neighbouring records can be out of order against their sequence numbers; the sequence is the order: sequence number, time, request identifier, command, user, client, workspace, target, the resulting code or outcome, the operation identifier if any, whether it was a replay, and a keyed digest of the payload. The request identifier, the client, the command and the target are recorded, and the request identifier echoed in a result, only when they satisfy the identifier rule, otherwise as a fixed marker, so that arbitrary caller text never enters the journal or a result. The digest is recorded only when step 3 passed, otherwise the marker. Sequence numbers are assigned when a record is written, not when its slot is reserved, so the journal has no gaps. It never holds a body, a field value, a rationale text, a verifier, a gate reason or an error text.

The keyed digest is HMAC-SHA-256 of the payload under a random key generated at construction and never exported, so that a reader of the journal cannot recover a low-entropy body by enumeration; the idempotency store keeps the plain digest in memory only. Equal digests within one registry let a reader of the journal link identical payloads across users (a stated limit). Construction has no fallback key: the system source does not fail in this toolchain. The key's persistence and custody are a later decision (HB-177).

The journal is append-only and in memory, with a capacity given at construction. No audit, no action: when no slot can be reserved every admitted call is refused, as step 1 says. Reading the journal is a trusted operation of the holder; exposing it is a later command with its own permission.

## Catalogue and OpenAPI

`Catalogue()` returns the definitions in identifier order with their permission, flags, fields and whether they are supported.

`OpenAPI(title, version)` renders them as an OpenAPI 3.1 document, deterministic byte for byte and valid for an empty catalogue. Everything about its transport shape is provisional and exists so that the catalogue can be read by OpenAPI tooling; no HTTP path is deployed and none follows from it:

- one concrete `POST` path per command under `/commands/`, without a path parameter, also for reading commands, with `operationId` equal to the command identifier (which may contain `.`, `:` and `-`);
- key, expected revision, target, rationale and body as members of the JSON request, not as headers;
- the member names of request, result and error; the statuses 200 for completed and 202 for accepted, with 201 not used; the error statuses of the table, except `unsupported_command`, which belongs to no operation and appears only among the shared components;
- decimals as strings with the pattern above, and integers and booleans as strings too, because every field value travels as a string; the extension members `x-permission` (action, kind, fields, approval flag), `x-supported`, `x-mutating`, `x-type` on each field, `x-errors` (the error table, with the reading-command exception of `revision_conflict` as its own member) and, for lengths, `x-max-bytes` and `x-max-value-bytes`, because JSON Schema's `maxLength` counts characters and the executor counts bytes; `maxLength` is not emitted;
- per operation only the codes that operation can return: no `idempotency_conflict` on a reading command, `self_approval_denied` only on an approval command, and for a command without a handler no success response at all, only `unsupported_capability`, the codes that can precede it and `internal_error`;
- `Result` and `Error` are open objects, so that a later member does not break a strict client, and `next_action` is a closed list;
- no security scheme, because the authentication method is the owner's open decision. Since tooling reads a missing scheme as "no authentication", the document carries `x-provisional: true` and `x-anonymous-access: false` at the top level and says in its description that it is not a published contract and that no operation is anonymous.

Tool and SDK contracts, versioning and deprecation rules are not generated.

## Limits and construction

Required and positive at construction, without defaults: fields per definition, length of a field value, length of a rationale, stored operations, journal capacity. Stored operations and journal records are never evicted, because an evicted key could run an action twice and an evicted record is a lost audit trail.

Consequence, stated as an exposure blocker: any admitted caller of any workspace can fill the journal with refused calls, and a permitted caller can fill the operation store with fresh keys; every further call in every workspace then ends with `dependency_unavailable` until restart. Retention, rate limits and capacities per workspace are owner decisions recorded in HB-177, and until they are made and built this package must not back an exposed path.

The executor takes its clock from the gate. An unconstructed registry refuses every call with `authority_unavailable` and does not panic.

## What is not decided here

Formal command and permission identifiers; compatible patch versioning; SDK languages; deprecation period; the authentication method and the transport shape of the HTTP surface; retention of operations and audit records and rate limits; decimal precision and scale; approvals through clients; whether the catalogue is confidential.

Not produced, and deferred with the commands that need them: the contract's families `authority_epoch_conflict`, `evidence_stale`, `approval_stale`, `independent_approval_required`, `identity_proof_invalid`, `policy_incomplete`, `rate_limited`, the destination errors and `outcome_unknown`; permitted evidence references in a result; the technical state and the status and cancellation capabilities of an accepted job; commands that need two permissions; calls inside a support session; the delegations a decision used.

No real command is registered, so the acceptance "one business cycle through the API alone" and the cases EXT01, EXT02, EXT07, REV05, AUD-AR-02, AUD-QA-04 and AUD-QA-05 remain NOT_RUN at system level; this ticket does not complete W-33.

Provisional choices, labelled above: the four added error codes and all statuses; `unauthenticated` for an answer that is not current; a missing target reported as a permission denial; the initiating principal being the user identifier; a named handler error not consuming the key and an unknown one consuming it; one lock for mutating calls; refusing instead of evicting; one permission per command; approvals refused through clients; the non-confidential catalogue; every element of the OpenAPI document's transport shape.

# Allowed files

`modules/commands/*` (new); `modules/gate/*` (the admission operation, the clock accessor, the answer alias and `Describes`); `modules/auth/registry.go` (the read-only `Describes` only, added after the change review); `docs/features/HB-176/*`.

# Verification

Literal table tests are the primary evidence, with synthetic commands and handlers.

- Registration: identifier rule, an identifier registered twice, an unknown field type, an enum without values, a field named twice, the approval flag against the action in both directions, limits; a definition never changing.
- Each step in order, with the earlier steps passing and the later ones set up to fail, so that the code returned shows which step refused: an answer that is not current against an unknown command; an unknown command against a malformed body; `unsupported_capability` before a malformed body; a malformed body against a forbidden target; a missing target and a forbidden target giving the same result and the same journal shape, with the gate called in both; a denial before any idempotency answer; an idempotency conflict before a stale revision; a replay before the revision comparison.
- Shape: every field type with valid and invalid values (decimal: exponent, NaN, a plus sign, a comma, a leading dot, a trailing dot, a bare minus, leading zeros, `-0`; integers; timestamps without offset; enums), required, unknown and repeated fields, key and revision present or absent against the flags, rationale present or absent against the flag, identifier-rule violations of key, target, revision and request identifier.
- A differential check of the decimal form against the decimal primitive's parser under a wide profile.
- Permissions through the gate: allowed; no grant; a suspended user and a removed membership between two calls; a client without the grant and with it; an approval command with an independent approver, the author, an approver without the grant, a call through a client and a target without author; a command with a field permission.
- Idempotency (AUD-QA-04): the same key, user and body returns the stored operation and the handler ran once; the same key with another body, target, expected revision or rationale conflicts; an absent optional field against a present empty one conflicts; the same key from another user conflicts; the same key through another client, with no client, and for another command does not conflict; a stored operation of another workspace conflicts (tested at the store, because a second workspace cannot be built); a replay after the revision moved still returns the stored operation; a replay by a caller who lost the permission is denied and discloses nothing; a replay after the target stopped resolving is denied; a named handler error leaves the key free; an unknown error, a panic and an invalid outcome consume the key and replay `internal_error`.
- Revision: stale, current, and a handler that reports its own conflict, for a mutating and for a reading command.
- Outcomes: completed and accepted; every named handler error mapped; a command without a handler; a panic in `Resolve` and in `Apply` leaving the lock free and the record written.
- Recheck before apply: a test clock that ends the session at the permission registry's read during the recheck, with the handler never called, for an ordinary and for an approval command; a recheck that fails only because an unrelated membership changed ends with `authority_unavailable`.
- Journal: exactly one record per admitted call for every end and none before admission; no body, value, rationale, gate reason or error text in any record (searched for marker strings); a command or target that breaks the identifier rule recorded as the marker; the keyed digest differing between two registries for the same payload and equal within one; a full journal refusing a reading call, a refused call and a replay right after admission; order and sequence.
- Limits: each at and above its value; the operation store full while existing keys still replay.
- Catalogue and OpenAPI: a golden document; equal bytes for registrations in another order; an empty catalogue; valid JSON; every command and error code present; 401 on every operation and 403 on every operation of a command with a handler; decimals as strings with the pattern; no security scheme, the provisional and anonymous-access markers.
- An unconstructed registry; the import scan of the gate still passing with this package present.
- Properties with a fixed seed over generated calls against a reference model written in the test, with its own tables of grants, targets, revisions and operations and no shared helper: the code of every call equals the model's; `Apply` calls equal stored operations plus calls that ended with a handler-origin code, and no `Apply` for a call refused at steps 1 to 7; the journal has exactly one record per admitted call; the digest is injective over generated payloads.
- Concurrency under the race detector: many goroutines sending the same mutating call give exactly one `Apply` and replays with the same operation identifier for all others; a reading call racing mutating calls for the last journal slot never lets an action go unrecorded.
- Mutation check by a script with results in the record, covering each step and its position in the order, each condition of the idempotency rule, the digest's parts, the revision comparison, both rechecks, each error mapping, each shape rule, the journal's reservation and its capacity rule, the store's reservation, and each limit.
- `gofmt -l`, `go vet ./...`, `go test -race ./...`.
- Independent review of the actual change.

# Deferred

The HTTP server and authentication adapters; real commands and the administration commands for grants, identities and relationships, which need permission identifiers that do not exist yet; jobs (W-34); pages with a snapshot cursor and stored target manifests, which come with the first reading and bulk commands; webhooks and events (QA-05); connector authentication (AR-02); rate limiting, retention and durable storage (HB-177); tool and SDK contracts.

# Design review resolution

The independent design review of 2026-11-13 (read-only, nothing executed) returned FAIL (narrow) without a blocker: 5 HIGH, 11 MEDIUM, 5 LOW, all fixable in the plan text, and stated that no owner decision blocks implementation when the fail-closed forms are adopted. It found no attack on the choice of workspace, actor, author or permission, on a replay by a caller who lost the permission, or on turning an unsupported command into success. Adopted as written above:

| Finding | Resolution |
| --- | --- |
| D1 a stored operation was not bound to a workspace | Another workspace under the same identity conflicts |
| D2 approval commands dropped the client | An approval through a client is denied |
| D3 capacity checked but not reserved | Journal slot reserved after admission, operation slot before apply |
| D4 an unknown error or a panic left the key free | Stored as failed with an unknown effect; named errors only when nothing changed |
| D5 one caller can disable mutations for the installation | Stated as an exposure blocker with HB-177 |
| D6 lock scope and the recheck window | Lock from step 4 to step 9; the window stated |
| D7 step-8 refusals | Matching recheck; a stale stamp is retryable |
| D8 handler error set | Handler denial and handler validation errors added |
| D9 contract families dropped silently | Named as not produced and deferred |
| D10 replay when the target stops resolving | Handler obligation and stated limit; creating commands defined |
| D11 check-then-use gap for reads | Revision passed to the handler; dimension changes change the revision |
| D12 resolve failure modes | Found, not found, unavailable; the gate called in both paths |
| D13 gate result mapping | By code and reason, no reason disclosed |
| D14 unbounded caller strings | Identifier rule for key, target, revision and request identifier; markers in the journal |
| D15 payload digest in the journal | Keyed digest |
| D16 the OpenAPI document | Every transport element listed as provisional; markers and 401, 403 on every operation |
| D17 journal-full behaviour | Every admitted call refused |
| D18 401 for a removed membership | Next action stated |
| D19 verification | Cases, the property and the differential check added |
| D20 flag and action mismatch | Refused at registration |
| D21 cross-user key blocking | Stated as a limit |

# Design re-review resolution

The narrow re-review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker, 1 HIGH, 8 MEDIUM and 4 LOW, each a one-sentence correction, and judged the document implementable once the first five were adopted; no owner decision blocks implementation. All thirteen were adopted in the sections above: the gate call for a missing target is the real call and its unavailable and not-current results are kept, so an outage is no existence oracle (E1); an approval through a client is refused before the target is resolved (E2); the operation slot is reserved just before `Apply` and every path stores or releases it (E3); a stamp that is not current is retryable at step 5 as at step 8 (E4); request identifier, client and field names never carry arbitrary text into the journal or a result (E5); a stored failure is bound and compared like a success and its error carries the operation identifier (E6); `Resolve` failures defined (E7); operation identifiers random (E8); handler-supplied strings bounded (E9); invalid outcomes and the reading command's retryable conflict (E10); journal details (E11); `Admit` returning only a code (E12); the remaining OpenAPI ambiguities (E13). The folded text was not reviewed a third time; the change review of the implementation is the check on it.

# Change review resolution

The independent change review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker, no HIGH, 6 MEDIUM and 12 LOW. It found no sequence in which a caller chooses the workspace, user, author or permission, replays across workspaces, users or payloads, applies one intent twice, overruns a capacity, loses or doubles a journal record, or receives a gate reason.

| Finding | Resolution |
| --- | --- |
| C1 no test of an unavailable authority through the executor | Test added for an existing and a missing target, ordinary and approval, with equal record shapes; mutation row added |
| C2 a returned stored failure aliased the stored operation | The operation is stored and replayed as a detached copy, error included; test added |
| C3 a handler's error differed from the executor's by an empty field list | Fixed; the test compares the lists exactly |
| C4 names echoed beyond the marker | Only the marker is returned when there are more fields than declared |
| C5 lax timestamp forms accepted | Strict pattern in addition to parsing; cases added |
| C6 byte limits rendered as character limits, value limit missing | Bytes stated as the unit; `x-max-bytes` and `x-max-value-bytes` emitted and `maxLength` not |
| C7 a replay could be recorded before its original | The record is written before the lock is released; test added. The record time stays the writing time, stated |
| C8 a failure after `Apply` could leave the key free | The clock is read before `Apply`; nothing fallible remains between `Apply` and the store. A handler that ends its goroutine instead of returning is not covered (limit) |
| C9 a broken revision answered before the gate decided | Checked after the decision; test added |
| C10 registration ignored the permission catalogue; nil pointer handler | `Describes` added to the permission registry and the gate, read-only; a nil pointer handler counts as absent; tests added |
| C11 wrapped named errors | Stated under Handlers |
| C12 `Admit` and `Clock` untested in the gate package | Test added there, including that admission does not consult permission availability |
| C13 deviations incomplete | Verification record extended |
| C14 undeclared decisions of the document | Listed under Catalogue and OpenAPI; per-operation codes, open result objects and the closed next-action list changed as stated there |
| C15 weak tests | Recheck tests assert the hook fired and the resolve count; journal test now includes gate refusals of four kinds; tests added for a service user, values at the limit, an unknown resolution and two mutating calls with different keys |
| C16 justification of row 99 | Replaced by a test with a failing clock; the row is caught. Result files stay outside the repository (limit) |
| C17 dead test code | Removed |
| C18 ignored `rand.Read` errors | Checked, as in the identity and permission registries. `NotCurrent` keeps its text; it is never emitted |

# Change re-review resolution

The narrow re-review of the corrections on 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker, no HIGH, 1 MEDIUM and 8 LOW, and found no new defect on the permission or tenant path. It confirmed the trace of nine clock reads against the code.

| Finding | Resolution |
| --- | --- |
| R1 the permission registry was changed outside the allowed files | Allowed files and the decision text now name the one read-only method; recorded as a deviation |
| R2 a clock failing while the record is written left the lock held and a slot reserved | The record's clock read recovers and records the zero time; test added |
| R3 the limit about a handler ending its goroutine was false | Restated: the lock is released and a record without an outcome is written, but no operation is stored, so the key is free after a possibly applied action |
| R4 `approve_support_session` could be registered and never allowed | Refused at registration; the claim narrowed |
| R5 a loop made unreachable by the catalogue check; a test refused for another reason than named | Loop removed; the case is now a field action with a repeated field |
| R6 record times not monotonic in sequence | Stated in the journal section and in the limits |
| R7 valid RFC 3339 forms refused; several spellings of one instant | The pattern is emitted in the schema; the plan says fail-closed and lists both; cases added |
| R8 four stale sentences | Corrected in steps 8 and 9, the document section and the verification list |
| R9 the lost-slot assertion could not fail | The test runs with a journal of three records and asserts the reservation count |

These corrections were not reviewed a third time.
