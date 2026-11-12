---
type: VerificationRecord
title: Command registry and executor verification
description: Records what was executed and reviewed for the first command layer of work item W-33, with results, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-176](https://linear.app/hybrid-billing/issue/HB-176/w-33-command-registry-and-executor-behind-the-gate-with-idempotency) adds the Go package `modules/commands` from main `e2fb6a5`: a registry of command definitions (permission, approval, mutation and rationale flags, a typed body), an executor that admits one call through the gate in nine ordered steps (admission, command, shape, target resolution by the handler, permission decision, idempotency, expected revision, recheck, apply), a closed error catalogue with the contract's proposed statuses, an append-only audit journal without bodies, and a deterministic, provisional OpenAPI rendering of the catalogue. `modules/gate` gains `Admit`, a clock accessor, an alias for the access answer and `Describes`, which asks the permission registry (`modules/auth`, one new read-only method) whether its catalogue describes an action, kind and field set; it grants and decides nothing.

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan twice before implementation and the implemented change afterwards (see Independent review, which records its result). Machine: mac_mini. See the [plan](plan.md).

No real command is registered and no HTTP server exists: every test uses synthetic commands and handlers. The acceptance "one business cycle through the API alone" and the cases EXT01, EXT02, EXT07, REV05, AUD-AR-02, AUD-QA-04 and AUD-QA-05 remain NOT_RUN at system level. The package must not back an exposed path until retention and rate limits are decided ([HB-177](https://linear.app/hybrid-billing/issue/HB-177/retention-rate-limits-and-durable-storage-for-command-operations-and)).

# Executed checks

Executed on 2026-11-13 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain. No other operating system was tested.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All ten packages `ok`, including `modules/commands`, `modules/gate` and `modules/auth`; run again after the last correction |

Literal table tests (`commands_test.go`), through a real identity registry and gate:

- registration: fourteen malformed definitions, an identifier registered twice, stored definitions and the catalogue not aliasing the caller's, required limits, a nil and an unconstructed gate;
- a completed mutation and a reading call with every member of their results and of the journal record;
- the order of refusals: an answer that is not current and the zero answer before anything else and without a journal record or a handler call; an unknown command and a command without a handler before a malformed body; a malformed body before the target is resolved; a forbidden and a missing target giving equal results and equal record shapes; a denial before any idempotency answer; an idempotency conflict before a stale revision; a replay after the revision moved; a new key with the old revision;
- shape: seven valid bodies; thirty-four invalid values (decimal: exponent, NaN, infinity, plus sign, comma, leading dot, trailing dot, bare minus, leading zeros, empty, space, grouping; integers, booleans, timestamps, enums, length); fifteen structural errors with the exact field names returned; reading commands with a key, a revision or a rationale; free text in the request identifier, client, target and command never echoed or recorded;
- permissions: no grant, a field permission, an unknown client, a client without and with the grant, the same key with and without a client as two scopes, a suspended membership and a suspended user;
- approvals: an independent approver, the author, a missing target, an approver without the grant, an approval through a client refused before the target is resolved, a target without an author, and ordinary and approval grants not substituting for each other;
- idempotency: replay with the same operation and one apply; six changed payloads conflicting, among them the same amount in another form and an added empty field; another user; field order not mattering; another command and another client scope not conflicting; a replay by a caller who left the workspace, by one who lost the grant (disclosing nothing), and after the target stopped resolving; an operation stored for another workspace (set directly in the store, because a second workspace cannot be built);
- handler outcomes: each of the seven named errors with its code, retry flag and filtered field names, and the key free afterwards; twelve unknown ends (an unnamed error, a named error outside the handler set, a panic, an outcome with an error, missing revision, state or job, a job on a completed outcome, text as a revision or a field name, a value and a field count above the limits) each giving `internal_error` with an operation identifier, replaying it, conflicting for another body, applying once and leaving the lock free; accepted outcomes and their replay; three invalid ends of a reading command; the retry flag of a reading and of a mutating conflict;
- resolve failures: unavailable, a panic, a revision that is text, and the key free afterwards;
- the recheck before apply, with the test clock acting on the permission registry's clock read inside the recheck: a session end for an ordinary and for an approval command (refused, handler not called) and an unrelated membership change (retryable `authority_unavailable`, then success);
- the journal: sequence, paging, no caller text, rationale or gate reason in any record (searched for marker strings), copies, the keyed digest differing between two registries for one payload, equal within one, and never the plain digest;
- capacity: a full journal refusing a replay, a mutating, a reading and a refused call right after admission with no handler call; a slot reserved by a running mutation not taken by a reading call made during its apply; a full operation store refusing new keys before the revision comparison while existing keys replay;
- a missing target denied also under a grant for every customer;
- mutating calls not interleaving: a second identical call sent while the first is applying waits and replays;
- an unconstructed registry; results not aliasing stored operations.

`document_test.go`: the OpenAPI document against a golden file, equal for two renderings and for registrations in reverse order, valid for an empty catalogue, with the provisional and anonymous-access markers, no security scheme, the exact set of statuses per operation (no 404, no 201, 202 only for mutating commands with a handler, only 401, 500, 501 and 503 for a command without a handler), the codes behind 403 and 409 per kind of command, open result objects, the closed next-action list, byte limits present and `maxLength` absent, the reading exception of `revision_conflict` in the error list, request members per flag, the decimal pattern, and the full error list; the error catalogue against a literal table; the decimal form against the decimal primitive's parser under a wide profile on 35 literal and 20,000 generated strings; the payload encoding injective over 20,000 generated payloads and distinguishing an absent field from an empty one.

Added after the change review (`review_test.go`, and cases in the files above): an unavailable authority giving `authority_unavailable`, retryable, with equal record shapes for an existing and a missing target, ordinary and approval, and the key free afterwards; a stored failure not aliased by the result that was returned or replayed; a handler's named error equal to the executor's, compared as exact field lists; only the marker returned for more fields than declared; eleven lax or wrong timestamp forms refused (one-digit hour, comma fraction, an offset of a day, a space separator, month 13, an offset without a colon, lowercase separator, lowercase zone, leap second, a fraction without digits, 30 February) and a fraction of a second with an offset accepted; values exactly at the limit accepted, and Korean text counted in bytes (13 characters of 39 bytes accepted, 14 of 42 refused; a rationale of 21 characters and 63 bytes refused); a broken revision giving a denial, not `internal_error`, to a caller without the permission; registration refusing five permissions the gate's catalogue does not describe and the support-session approval action, and a nil pointer handler listed as unsupported; a service user through the same steps, refused for an approval and after its credential is revoked; a resolution outside the list treated as a missing target; a clock that fails after the recheck and before the apply leaving no operation, no held lock, no lost journal slot and a free key, and a clock that fails while the record of an applied action is written still giving the result, the record (with the zero time) and a reservation count of zero, shown with a journal of three records that is then exactly full; two mutating calls with different keys not overlapping from resolve to apply; a replay recorded after its original. The recheck tests now assert that the hook fired and how often the target was resolved; the journal search now runs over records of four kinds of gate refusal. `modules/gate` has its own test of `Admit` (current, empty and ended answers; not dependent on permission availability), `Clock` and `Describes` (seven cases) and of all three on a gate that is not built.

The golden file was produced by the generator under test and then read; it pins stability, not correctness. The structural assertions are the evidence for its content.

Property test (`property_test.go`, fixed seed): 60 histories of 150 steps with three users, three targets and one missing target, grants given and revoked, mutating and reading calls, exact retries of stored operations, and handlers that succeed, return a named error or return an unknown error. A reference model written in the test keeps its own tables of grants, revisions and operations. After every call the code, the replay flag, the replayed operation identifier, the revision, the number of handler applies, the number of journal records and the number of stored operations equal the model's. Counts of the run: 762 mutations applied and 928 replayed, 339 idempotency conflicts, 222 revision conflicts, 207 named and 177 unknown handler errors with 213 replays of an unknown one, 2,893 denials; 419 reads answered and 815 denied; 159 grants revoked. The test fails when any of 13 categories is reached fewer than 20 times.

Limits of the reference model: answers are always fresh, so identity changes are covered only by the literal tests; one client scope; no approval commands; no capacity limits; it shares the formula of the synthetic handler's revision with the handler.

Concurrency under the race detector: eight goroutines sending the same mutating call ten times each, interleaved with reading calls: one apply, one fresh result, every other result a replay of the same operation, and one journal record per call.

# Mutation check

144 single-line mutations of `modules/commands`, of the gate's additions and of the permission registry's `Describes` were applied one at a time by a script, and the tests of the three packages were run after each. The per-row results were appended to files that were never overwritten.

History, because the list and the code changed twice:

- Before the change review the list had 122 rows. First run: 97 caught, 9 not caught, 16 did not compile because removing a check left a variable unused. Four of the 9 showed missing tests, which were added (the journal reservation count; serialization of mutating calls; a missing target under a grant for every customer; the keyed journal digest), and a fifth showed a returned result not checked for aliasing. The 16 were rewritten so that they compile. Result then: 117 of 122 caught.
- After the change review's corrections, 13 rows were rewritten for the changed code and 20 rows were added for the corrections (rows 123 to 142; row 133 repeats row 39 in another form). All 142 were run again from the start: 137 caught, 5 not caught. The fifth, row 130 (a replay recorded before its original), showed that the new test of that rule did not test it: its clock hook was armed one read too early and fired while the lock was still held in both variants. Tracing the clock reads of one successful mutating call gave nine (admission 1; decision 2 to 4; recheck: identity 5, permission registry 6, identity again 7; the executor's time before the apply 8; the journal record 9), not the eight assumed. The same miscount had put the failing-clock test inside the identity recheck instead of at the executor's own read. Both tests were corrected to reads 9 and 8, the journal record's clock read was moved out of the registry's lock so that the hook can act there, and rows 5 to 8 and 129 to 132 were run again: all caught.
- After the re-review's corrections two rows were added (143: a failing clock losing the record of an applied action; 144: registering the support-session approval action), one row was rewritten, and the 29 rows of the changed files were run again: all caught.
- Result: 140 of 144 caught (3 of them because the tests did not finish), 4 not caught. Row 99 (a panic escaping the executor), not caught before, is now caught by the failing-clock test.

The 4 that are not caught:

| Mutation | Why no test distinguishes it |
| --- | --- |
| 2, an unavailable identity registry reported as "not current" by `Admit` | The identity registry is unavailable only when unconstructed, and the gate cannot be built over an unconstructed one; the branch is a second line of defence |
| 41, the data of a missing target used for the decision | Whatever the decision, a missing target is then turned into a denial by the next check |
| 48, a missing target of an approval command reported as self-approval | The approval check with an empty author never returns self-approval |
| 63, the payload digest without its version tag | The tag is a constant; it matters only when a second version exists |

| No. | Fault introduced | Result |
| --- | --- | --- |
| 1 | Every answer admitted | Caught |
| 2 | Unavailable identity reported as not current | Not caught |
| 3 | A call that is not admitted proceeds | Caught |
| 4 | A call proceeds without a journal slot | Caught |
| 5 | Reserved journal slots not counted | Caught |
| 6 | Journal one record over its capacity | Caught |
| 7 | Journal reservations never released | Caught |
| 8 | Records without a sequence | Caught |
| 9 | An unknown command proceeds | Caught |
| 10 | A command without a handler reported as completed | Caught |
| 11 | A malformed call proceeds | Caught |
| 12 | Invalid request without field names | Caught |
| 13 | Request identifier unchecked | Caught |
| 14 | Client identifier unchecked | Caught |
| 15 | Target identifier unchecked | Caught |
| 16 | A key accepted on a reading command | Caught |
| 17 | A mutating command without a key | Caught |
| 18 | A revision accepted on a reading command | Caught |
| 19 | A mutating command without an expected revision | Caught |
| 20 | A rationale accepted where none is declared | Caught |
| 21 | A required rationale missing | Caught |
| 22 | Rationale length unbounded | Caught |
| 23 | More fields than declared accepted | Caught |
| 24 | A field name that is text echoed | Caught |
| 25 | An unknown field accepted | Caught |
| 26 | A repeated field accepted | Caught |
| 27 | Value length unbounded | Caught |
| 28 | Field types unchecked | Caught |
| 29 | A required field missing accepted | Caught |
| 30 | Identifier fields unchecked | Caught |
| 31 | An exponent accepted as a decimal | Caught |
| 32 | A fraction accepted as an integer | Caught |
| 33 | A numeric boolean accepted | Caught |
| 34 | A timestamp without offset accepted | Caught |
| 35 | Any enum value accepted | Caught |
| 36 | Leading zeros accepted in a decimal | Caught |
| 37 | An approval through a client | Caught |
| 38 | Mutating calls not serialized | Caught |
| 39 | A resolved revision that is text accepted | Caught |
| 40 | An unavailable target treated as missing | Caught |
| 41 | Data of a missing target used | Not caught |
| 42 | The target's customer left out of the decision | Caught |
| 43 | Permission fields left out of the decision | Caught |
| 44 | Approval commands decided by the ordinary decision | Caught (the tests did not finish) |
| 45 | The approval author not taken from the target | Caught |
| 46 | The client left out of the decision | Caught |
| 47 | A missing target proceeds when the gate allows | Caught |
| 48 | A missing target reported as self approval | Not caught |
| 49 | A denial ignored | Caught |
| 50 | A stale stamp reported as a denial | Caught |
| 51 | An answer that is not current reported as a denial | Caught |
| 52 | Self approval reported as a denial | Caught |
| 53 | An operation of another workspace replayed | Caught |
| 54 | An operation of another user replayed | Caught |
| 55 | Another payload replayed | Caught |
| 56 | A stored operation applied again | Caught |
| 57 | The operation store unbounded | Caught |
| 58 | One operation over the store's capacity | Caught |
| 59 | The expected revision not compared | Caught |
| 60 | The client left out of the idempotency identity | Caught |
| 61 | The command left out of the idempotency identity | Caught |
| 62 | The recheck's refusal ignored | Caught (the tests did not finish) |
| 63 | Digest without its tag | Not caught |
| 64 | Digest without the command | Caught |
| 65 | Digest without the target | Caught |
| 66 | Digest without the expected revision | Caught |
| 67 | Digest without the rationale | Caught |
| 68 | Digest without field names | Caught |
| 69 | Digest without field values | Caught |
| 70 | Digest parts without length prefix | Caught |
| 71 | Digest depends on field order | Caught |
| 72 | Journal digest without the secret key | Caught |
| 73 | Journal digest never recorded | Caught |
| 74 | Journal key not random | Caught |
| 75 | Operation identifiers not random | Caught |
| 76 | An outcome together with a named error accepted | Caught |
| 77 | Any named code accepted from a handler | Caught |
| 78 | Undeclared field names returned from a handler | Caught |
| 79 | A reading conflict not retryable | Caught |
| 80 | A mutating conflict retryable | Caught |
| 81 | A named error consumes the key | Caught |
| 82 | Invalid outcomes accepted | Caught |
| 83 | An unnamed error treated as success | Caught |
| 84 | An unknown failure leaves the key free | Caught |
| 85 | An unknown failure without its operation identifier | Caught |
| 86 | Result fields unbounded | Caught |
| 87 | Result field names that are text | Caught |
| 88 | Result values unbounded | Caught |
| 89 | An accepted outcome of a reading command | Caught |
| 90 | An accepted outcome without a job | Caught |
| 91 | A job on a completed outcome | Caught |
| 92 | A revision from a reading command | Caught |
| 93 | A completed outcome without a revision | Caught |
| 94 | A completed outcome without a state | Caught |
| 95 | Operations never stored | Caught |
| 96 | A stored or replayed result aliases the fields | Caught |
| 97 | A replay not marked | Caught |
| 98 | A fresh result aliases the stored operation | Caught |
| 99 | A panic escapes the executor | Caught |
| 100 | The journal record never written | Caught |
| 101 | Replays not marked in the journal | Caught |
| 102 | A failed operation's identifier missing from the journal | Caught |
| 103 | No client recorded as the marker | Caught |
| 104 | Free text of a target recorded | Caught |
| 105 | Free text of a command recorded | Caught |
| 106 | Free text of a client recorded | Caught |
| 107 | Free text of a request identifier echoed | Caught |
| 108 | A reading result without the resolved revision | Caught |
| 109 | An accepted outcome reported as completed | Caught |
| 110 | Approval flag and action may disagree | Caught |
| 111 | An approval that does not mutate registered | Caught |
| 112 | Fields per definition unbounded | Caught |
| 113 | A field named twice registered | Caught |
| 114 | An unknown field type registered | Caught |
| 115 | An enum without values registered | Caught |
| 116 | A definition replaced | Caught |
| 117 | A registered definition aliases the caller's | Caught |
| 118 | The catalogue aliases the registry | Caught |
| 119 | The journal aliases what it returns | Caught |
| 120 | An internal error retryable | Caught |
| 121 | An unavailable authority not retryable | Caught |
| 122 | The marker never used | Caught |
| 123 | An outage of the authority reported as a denial | Caught |
| 124 | A stored or replayed error aliases what is returned | Caught |
| 125 | A handler's error tells itself from the executor's by an empty list | Caught |
| 126 | Names echoed although more fields than declared | Caught |
| 127 | A missing name echoed although more fields than declared | Caught |
| 128 | Lax timestamp forms accepted | Caught |
| 129 | An offset of a day accepted by the pattern alone | Caught |
| 130 | A replay may be recorded before its original | Caught |
| 131 | The lock kept after a call | Caught (the tests did not finish) |
| 132 | A failure outside the handler not recovered | Caught |
| 133 | (same as 39, kept for the count) a broken revision passes | Caught |
| 134 | Registration ignores the permission catalogue | Caught |
| 135 | A nil pointer handler listed as supported | Caught |
| 136 | The catalogue describes any kind | Caught |
| 137 | The catalogue describes any action | Caught |
| 138 | The catalogue describes any fields | Caught |
| 139 | The gate describes everything | Caught |
| 140 | A key conflict listed for a reading command | Caught |
| 141 | A self approval listed for an ordinary command | Caught |
| 142 | Success listed for a command without a handler | Caught |
| 143 | A failing clock loses the record of an applied action | Caught |
| 144 | A command no call of which can be allowed registered | Caught |

# Deviations from the plan

- The operation store has no separate reservation: because mutating calls hold the executor's lock from step 4 to step 9, the free-slot test of step 6 holds until the operation is stored, and nothing needs to be released. The journal has a real reservation, because reading calls run outside the lock. There is accordingly no deferred store-or-release step; the plan's sentence about one is not implemented as written.
- A named handler error is recognised only when it comes without any outcome; a named error together with an outcome is an unknown end.
- The result of a reading command carries the resolved revision and no business state; a handler that returns a revision or state for a reading command produced an invalid outcome.
- An accepted result also carries the target and the result fields.
- The next action `inspect_operation` is returned for a stored failure although no command to inspect an operation exists yet.
- `client` can be returned as a field name of `invalid_request` although it is not a member of the request body.
- `OpenAPI` refuses an empty title or version.
- The golden OpenAPI document is under `modules/commands/testdata/`.
- `modules/auth/registry.go` was changed although the plan's allowed files did not name it: one read-only method, `Describes`, needed for the catalogue check at registration. The plan now names it.
- After the change review the plan itself was changed where the implementation now differs from the reviewed design (strict timestamps, bytes as the length unit, the marker alone for excess fields, the catalogue check at registration, as-of read before `Apply`, the record written before the lock is released, the per-operation codes of the document); see the plan's change review resolution.

# Provisional choices and undecided matters

The owner has not decided these; the implementation uses the labelled defaults of the plan: the four added error codes and every status; `unauthenticated` for an answer that is not current; a missing target reported as a permission denial; the user identifier as the initiating principal; a named handler error leaving the key free and an unknown end consuming it; one lock for mutating calls; refusal instead of eviction at the limits; one permission per command; approvals refused through clients; a catalogue that is not confidential to admitted callers; every element of the OpenAPI document's transport shape. Open decisions of HB-33 are unchanged: formal command and permission identifiers, compatible patch versioning, SDK languages, deprecation period.

# Limits

- Exposure blocker (HB-177): operations and journal records are never evicted; any admitted caller can fill the journal with refused calls and stop every call of every workspace until restart.
- No real command, no HTTP server, no authentication adapter.
- A change of identity or grants that completes after the recheck of step 8 does not stop an apply in progress.
- A slow handler blocks every mutating call of the installation.
- The handler obligations of the plan cannot be checked by the executor: comparing the revision when writing, returning a named error only when nothing changed, keeping a target with a stored operation resolvable.
- Keys must be unpredictable per intent; another user who uses a key first makes it conflict.
- Equal journal digests within one registry link identical payloads across users.
- State is in memory and lost on restart, including the journal key.
- A handler that ends its goroutine instead of returning (inferred from the language's rules, not tested): the lock is released and a record without an outcome is written, but no operation is stored, so the key is free after an action that may have been applied.
- Record times can be out of order against sequence numbers for calls that are not both mutating; a clock that fails while a record is written gives it the zero time.
- Timestamps are stricter than RFC 3339 (no lowercase `t` or `z`, no leap second), and one instant has several accepted spellings, each its own idempotency payload.
- Registration refuses the known kinds of permission no call could satisfy; that none remains is not proven.
- The time of a journal record is read just before it is appended, not when the action was applied.
- The test of two mutating calls with different keys shows overlap only if the second call starts within 100 ms; it is evidence against a missing lock, not a proof of the lock's extent.
- The reference model cannot see identity changes, clients, approvals, capacity, shape, resolve failures, journal content, or named codes other than `invalid_stage`; those are covered by literal tests only.
- The per-row mutation results are in the session's scratch files, not in the repository.
- The OpenAPI document was not checked with a validator; its 3.1 validity rests on structural assertions and reading.

# Independent review

The design review of 2026-11-13 returned FAIL (narrow: 5 HIGH, 11 MEDIUM, 5 LOW) and the plan was rewritten. The re-review of the same day returned PASS with findings (1 HIGH, 8 MEDIUM, 4 LOW, each a one-sentence correction), which were folded in; the folded text was not reviewed a third time. Both were read-only and executed nothing.


