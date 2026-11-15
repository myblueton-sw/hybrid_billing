---
type: ImplementationPlan
title: Retention, per-workspace capacity and rate limits for command operations and the audit journal
description: Defines the second increment of work item W-33, which lifts the exposure blocker of the command executor by giving stored operations a replay window, journal records a retention period, both stores per-workspace capacity on top of the installation bound with expired entries swept before any refusal, and every admitted caller a token bucket per workspace and user that returns a stable rate_limited error with retry guidance instead of filling either store.
status: draft
---

# Scope and ownership

[HB-177](https://linear.app/hybrid-billing/issue/HB-177) follows HB-176 (`modules/commands`) as the second increment of work item W-33, after HB-237. Base: main `953163b`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the plan and the change, because this changes the public command contract (a new error code) and the evidence kept for audit. Machine: mac_mini.

The ticket names owner decisions: retention of operations and audit records, rate limits, per-workspace capacity, and the persistence and custody of the digest key. This increment builds the mechanisms with configuration. Every value is a labelled provisional default for the owner.

Out of scope, recorded as residuals:

- durable storage of operations and records, and persistence and custody of the journal digest key (records stay in memory, so a restart loses them);
- export of records to long-term audit storage before eviction;
- rate limits per client or command, and distributed rate limiting across processes;
- an HTTP layer that maps `rate_limited` to `429` with a `Retry-After` header.

# Observed baseline

- `modules/commands` keeps every operation (for idempotent replay) and every journal record in memory and never evicts either.
- Capacities are installation-wide (`Limits.Operations`, `Limits.Journal`). When they are reached, every call of every workspace ends with `dependency_unavailable` until restart. HB-176 states this as an exposure blocker (D5).
- Execution order today:
  1. the answer is admitted (`Admit`);
  2. a journal slot is reserved;
  3. definition and shape are checked;
  4. the gate decides;
  5. for mutating calls, the operation store is looked up, then the revision checked and the handler applied;
  6. the record is written with a sequence equal to its index plus one.
- Callers that are not admitted never reach the journal.
- QA-05 requires stable codes and `429` retry guidance. QA-04 requires the idempotency key scope (installation, client, command) and `409` for a different payload under the same key.
- The only importer of `modules/commands` is `modules/assistant` (tests construct `commands.Limits`).

# Decision

## Allowed files

- `modules/commands/*`
- `modules/assistant/calls.go`, `modules/assistant/types.go` and the assistant tests, which build `commands.Limits` and now pass the retry interval on
- `docs/features/HB-177/*`

## Limits

`commands.Limits` gains these fields, all required:

| Field | Meaning | Validity |
| --- | --- | --- |
| `Workspaces` | workspaces that may hold records or operations at once | 1 or more |
| `WorkspaceJournal` | records per workspace | `Workspaces × WorkspaceJournal ≤ Journal` |
| `Users` | users per workspace that may hold records or operations at once | 1 or more |
| `UserJournal` | records per workspace user | `Users × UserJournal ≤ WorkspaceJournal` |
| `WorkspaceOperations` | operations per workspace | `Workspaces × WorkspaceOperations ≤ Operations` |
| `UserOperations` | operations and tombstones per workspace user | `Users × UserOperations ≤ WorkspaceOperations` |
| `Replay` | replay window of a stored operation | 1 minute to 30 days |
| `Retention` | minimum retention of records and expired-key tombstones | 1 hour to 3650 days, and at least `Replay` |
| `Burst` | token bucket capacity per workspace and user | 1 to 100,000 |
| `Refill` | interval for one token | 1 millisecond to 1 hour |

Because the per-workspace bounds multiplied by `Workspaces` fit within the installation bounds, the installation bound can never fill while a workspace is below its own bound. One workspace therefore cannot stop another; a new workspace beyond `Workspaces` is refused as capacity. In the same way, `Users ×` the user share fits within the workspace share, so one user (or a set of users smaller than the whole) cannot fill the workspace. A user beyond `Users` is refused as capacity for their own recorded calls and mutations only.

Tombstones count toward the operation shares for the whole retention, and records and tombstones keep a workspace in the `Workspaces` count for the whole retention. Shares must therefore be sized for `Retention`, not for `Replay`. A user who exceeds their share over a retention period, for example an automated loop, locks only their own mutations or recorded calls until their oldest entries expire. That is fail-closed, and the shares are deliberately not tied to the rate, so an owner can choose a large share for service users. Users likewise stay counted in `Users` while they hold records or tombstones, so a workspace that rotates more users than `Users` within the retention refuses its newest users' recorded calls. These behaviours are provisional limits for the owner.

Provisional defaults for the owner:

- replay 24 hours and retention 400 days;
- a burst of 60, with one token every second;
- 1,000 workspaces;
- per-workspace and per-user shares sized to the expected users.

## Rate limits

After `Admit` and before anything else, every call takes one token from the bucket of its workspace and user. Replays and refused calls count too.

Each bucket holds whole tokens and the time of its last refill:

- When now is at or after the last refill time, `n = floor((now − last) / Refill)` tokens are added, up to `Burst`, and the last refill time advances by `n × Refill`. A full bucket sets it to now.
- When now is before the last refill time by less than one interval (a reordered concurrent read or a small step back), the step is ignored. When it is earlier by one interval or more, the last refill time is set to now. Either way nothing is added, and a clock step never costs a user more than two intervals.
- When the clock fails to read, nothing is added and nothing changes.

Times are compared as the clock returns them, without stripping the monotonic reading. A forward step of a clock without a monotonic reading (a test clock) expires records and tombstones early, and a step past `Retention` also removes tombstones, so the key could execute again with only the revision check as a guard, and records are evicted early. A production clock must therefore keep the monotonic reading.

Without a token, the call returns the new code `rate_limited`:

- status 429, `Retry` true, next action `retry_later`;
- `RetryAfter`, the time until the next token: `Refill − (now − last)`, always in (0, `Refill`]. If the clock failed or stepped back by less than one interval, it is `Refill`, which by the caller's clock can be up to one interval short.

A rate-limited call leaves no record and no operation. `Throttled(workspace)` returns, for each user, the count and the first and last time of rate-limited calls. This counter replaces HB-176's one-record-per-admitted-call invariant for throttled calls.

Buckets exist only for admitted users. When the number of buckets has doubled since the last cleanup, full buckets are removed, so cleanup cost stays amortised. Floods of unauthenticated calls are refused at `Admit` and are left to the HTTP layer (residual).

## Journal: recorded and aggregated calls

The audit rule is "no record, no action". A call reserves a record slot before any handler is reached. A reservation needs room in three places:

- the user's share (`UserJournal`);
- the workspace's share (`WorkspaceJournal`);
- the number of workspaces holding records (`Workspaces`).

Expired records are swept first.

When there is no room, the call runs in unrecorded mode:

- Refusals decided before any handler call keep their own code: unsupported command, missing handler, invalid shape, and an approval through a client.
- Any call that would reach the handler (resolve or apply) ends with `dependency_unavailable` instead.
- Neither case produces a record. Both are added to an aggregated counter per workspace, user and code, with the first and last time, which `Unrecorded(workspace)` returns.

So refused floods never block anything, and a user who fills their share blocks only their own recorded calls until their records expire.

Retention:

- Records are kept for at least `Retention` after their time.
- A record whose clock read failed takes its predecessor's retention time, or, without one, the time read at its reservation. If both failed, it takes the time of the next record that has one; operations whose clock read failed are handled the same way. So anomalous records are never deleted first and never stop the sweep.
- Sweeps run from the oldest end on every reservation and stop at the first live record. Record times may not be monotonic, which only delays a sweep.
- The backing array is compacted when more than half of it is dead.

Sequences come from a counter and stay global and increasing. `Journal(after, limit)` returns retained records with a sequence above `after`, found by binary search. `JournalStart()` returns the oldest retained sequence, or the next sequence when empty, so a reader can tell that earlier records were removed by retention.

## Operations: replay window and tombstones

A stored operation keeps its creation time, its workspace, its user and an entry in a creation-order list that points at the operation itself.

| Age of the operation | Same key and same scope |
| --- | --- |
| Before `Replay` | Replays the stored result, as today. |
| From `Replay` to `Retention` | The operation is a tombstone that keeps identity, workspace, user and digest. The call returns the new code `idempotency_key_expired`: 409, not retryable, next action `inspect_current_state`. The key never executes again within that time, so a late retry cannot charge twice. Unknown-effect (`internal_error`) operations follow the same rule. |
| After `Retention` | The tombstone is removed and the key can execute again. Clients must not reuse keys older than the retention (stated contract). |

A different workspace, user or digest under a live operation or tombstone stays `idempotency_conflict`.

The sweep walks the creation-order list from the oldest end. It deletes a map entry only when the map still points at the list entry's own operation, and then decrements the counts of that operation's own workspace and user. A key whose own entry is past retention is deleted before the capacity check.

A new operation needs room in three places:

- the user's share (`UserOperations`);
- the workspace's share (`WorkspaceOperations`);
- the number of workspaces holding operations (`Workspaces`).

Otherwise the call returns `dependency_unavailable`. All of this happens under the mutation lock, which is already held from the store lookup to the store write.

## Contract

- `rate_limited` and `idempotency_key_expired` are added to the error catalogue, the error rules and the OpenAPI `code` enum. Adding values to a closed enum breaks strictly generated old clients; this is accepted because the document is provisional.
- `Error` gains `RetryAfter`, a duration that is non-zero only for `rate_limited`. On the wire it is `retry_after_ms`, a whole number of milliseconds rounded up with a minimum of 1, added to the OpenAPI error schema.
- `idempotency_key_expired` is listed only on mutating operations, like `idempotency_conflict`.
- Every operation of the OpenAPI document lists 429, including commands without a handler, because rate limiting comes before the handler check.
- The handler obligations gain one item: every completed or accepted mutation changes the target's revision, including the parent of a create, so the revision check guards anything a tombstone does not.
- The command contract document's idempotency section must state the replay window, `idempotency_key_expired` and key reuse after the retention. That document is owner-maintained and outside this change, so it is a residual.

## Assistant

`modules/assistant` still counts a rate-limited call against the job's call budget, because every attempt that reaches the executor counts. Its `Reply` gains `RetryAfter`, which carries the executor's value through.

## Residual exposure blocker

The installation-wide mutation lock (HB-176) remains: a slow handler can stall mutations in every workspace. Scoping the lock per workspace needs a pending reservation of keys in the store and is left to a follow-up. Handlers must stay bounded in time until then.

# Verification plan

Literal tests:

- every limit validity boundary, including the products with `Workspaces`;
- bucket burst, whole-interval refill, the exact interval boundary, a backward clock with an empty bucket, a failing clock, and `RetryAfter` within (0, `Refill`];
- buckets isolated between users and workspaces, replays counted, and the cleanup of full buckets;
- concurrent callers draining one bucket under `-race`, where exactly `Burst` calls pass;
- rate-limited and unrecorded calls leaving no record and no operation, with their counters;
- the unrecorded mode keeping early refusal codes and stopping before the handler, including replays and key conflicts, which become `dependency_unavailable`;
- the `Users` bound, where one user filling their share leaves the other users of the workspace working;
- tombstones holding operation capacity until the retention;
- the user, workspace and workspace-count bounds, where one workspace's full share leaves another working;
- retention at the boundary, a zero-time record keeping its predecessor's time, compaction, and `Journal` paging across evicted ranges with `JournalStart`;
- operation replay before `Replay`, `idempotency_key_expired` from `Replay` to `Retention` (including unknown-effect operations), and re-execution after `Retention`;
- a reused key after expiry followed by a sweep that keeps the new operation;
- operation capacity by user, workspace and workspace count;
- reservation counts returning to zero on every path, including handler panics;
- OpenAPI listing 429 on every operation, both new codes and `retry_after_ms`;
- the assistant passing `RetryAfter` on.

Existing HB-176 tests are updated, not removed. A property test runs random calls of several users and workspaces over time against an independent model of buckets, records, aggregation, operations and tombstones.

Checks:

- `-race -count=10`, a 386 build, gofmt and vet;
- the whole repository with `-race`;
- 100% coverage and mutation testing.

# Review history
- Design review round 1: FAIL (3 high, 7 medium, 7 low). Folded in: user shares, workspace count bounds and aggregated unrecorded refusals (H1); tombstones with `idempotency_key_expired` until the retention (H2); creation-order entries that point at their own operation (H3); zero-time records keeping their predecessor's time (M1); minimum retention with sweeps on every reservation (M2); the mutation lock as a stated residual exposure blocker (M3); bucket clock rules (M4); `retry_after_ms` and 429 on every operation (M5); allowed files and the assistant (M6); throttle counters per user with times (M7); own-entry deletion before the capacity check, compaction, amortised bucket cleanup, the sequence counter, enum compatibility and non-monotonic record times (low items). Not taken: retry guidance on `dependency_unavailable` (L2), left to a follow-up.
- Design review round 2: FAIL (1 high, 1 medium, 3 low). Folded in: the `Users` bound with `Users × UserX ≤ WorkspaceX` (N1); tombstones counting toward the shares, with sizing by retention and the workspace count over retention stated as provisional limits (N2); zero-time records without a predecessor, `idempotency_key_expired` only on mutating operations, forward clock steps and unrecorded replays in tests (N3).
- Design review round 3: PASS (1 medium, 2 low). Folded in: records and operations whose clock read failed take the next known time, so they never stop a sweep (P1); forward clock steps leave only the revision check (P2); users stay counted for the retention (P3).

# Change review resolution

- Change review round 1: FAIL (2 medium, 3 low). Resolved:
  - M1: bucket refill is computed in `time.Duration` and clamped before conversion, so a 32-bit build never wraps after long idle periods. Tested with a 1 ms refill and a 30-day gap.
  - M2: workspace isolation of buckets, journal shares, operation shares and the workspace count is tested on the store directly. The identity register currently holds one workspace per installation, so two workspaces cannot run end to end through `Execute` (residual for the first multi-workspace identity increment). The property test now simulates refills step by step instead of repeating the formula. It asserts replays and handler applies on every call, which detects re-execution, and compares the held user counts with the model at the end.
  - L1: a backward clock step smaller than one refill interval is ignored, so concurrent callers reading the clock in a different order cannot credit extra tokens.
  - L2: an operation whose clock read failed takes its reservation time, then the last known time.
  - L3: throttle and unrecorded counters are kept for the life of the process and grow only with admitted identities (residual: pruning with the retention).
- Change review round 2: PASS (1 low). Folded in: the plan text for small backward clock steps and their retry interval (N1).
