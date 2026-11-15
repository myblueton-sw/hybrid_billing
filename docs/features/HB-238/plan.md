---
type: ImplementationPlan
title: Mutation locks per workspace and target with pending idempotency reservations
description: Defines the third increment of work item W-33, which removes the last exposure blocker of the command executor by serializing mutations per workspace and target instead of per installation, reserving an idempotency key as a pending operation before the handler runs so that concurrent calls with the same key elsewhere are refused instead of executed, and releasing reservations and locks on every path.
status: draft
---

# Scope and ownership

[HB-238](https://linear.app/hybrid-billing/issue/HB-238) follows HB-177 as the third increment of work item W-33. Base: main `2b3b27e`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the plan and the change, because the change decides when a billing mutation may run twice. Machine: mac_mini.

Allowed files: `modules/commands/*` and `docs/features/HB-238/*`.

Out of scope, recorded as residuals:
- handler time limits and cancellation;
- locks across processes;
- handlers whose apply changes objects other than the resolved one; these must either omit `Key`, which falls back to the workspace lock, or serialize those writes inside their own registers.

# Observed baseline

- `modules/commands` holds one installation-wide mutation lock, taken after the shape and approval checks and before target resolution, and released after the journal record is written.
- Under that lock it does the following in order:
  1. resolves the target;
  2. decides at the gate;
  3. looks up the operation store, which covers replay, tombstone, conflict and capacity;
  4. checks the expected revision;
  5. rechecks the gate;
  6. applies the handler;
  7. stores the operation.
- One slow handler therefore stalls every mutation of every workspace. HB-176 (D5) and HB-177 state this as the remaining exposure blocker.
- The operation identity is installation, client, command and key; the stored scope is workspace, user and payload digest. The digest covers command, target, expected revision, rationale and body. A key used by another target or workspace is therefore always a conflict.
- An HB-176 test requires a replay's record to come after its original's record. This holds because the lock is released only after the record is written.

# Decision

## Canonical lock keys

`Target` gains `Key`, the canonical identity of the resolved object, as returned by `Handler.Resolve`. Handlers that set it must return the same key for every spelling or alias of one object, and a different key for different objects.

A mutating call takes two locks: a read-write lock per workspace, then a mutex per canonical key. It obtains them as follows:

1. After the shape and approval checks, it resolves the target once outside any lock. This is a read, and its result is used only to choose the lock.
2. If the target was found and carries a valid `Key`, the call takes the workspace lock shared, then the mutex for (workspace, `Key`).
3. If the target was found without a key, it takes the workspace lock exclusively.
4. If the target was not found, unavailable, or the read failed, it takes no lock at all. Such a call can never apply anything. It still runs the HB-176 sequence for its refusal, and it ends with `dependency_unavailable` if the second resolution finds the object.
5. Under the lock, the existing sequence runs unchanged, starting with resolution. If a call holding a key lock resolves a different key, no key, or nothing the second time, the object moved between the two reads, and the call returns `dependency_unavailable` (retryable) without applying anything.

So two calls that reach one object always share a lock, whether through one key or through the workspace fallback. A handler that never sets `Key` keeps the HB-176 guarantee within its workspace, and only other workspaces proceed in parallel. The HB-176 rule that steps 4 to 9 of mutating calls do not interleave becomes: they do not interleave for one object of one workspace.

Locks:

- **Lifetime.** Both kinds of lock live in a table of reference-counted entries under the registry mutex. An entry is created on first use, and its holder count is decremented only after the lock is released. It is deleted at zero, so no newcomer can create a second lock for the same key while one is held.
- **Lock order.** The order is always: workspace lock, then key mutex, then registry mutex. The registry mutex is never held while waiting for the other two.
- **Release.** Release is registered as its own deferred step before the epilogue, so it runs even if the epilogue panics. Locks are released after the journal record is written, as before, which keeps replay records after their originals.
- **Reading commands.** These take no lock, as today.
- **Concurrent resolution.** `Resolve` now runs outside any lock while `Apply` runs on the same object, so handlers must make `Resolve` safe to call at the same time as `Apply` (handler obligation).
- **Re-entry.** Handlers must not call `Execute`. This is stated as a handler obligation, because a re-entrant mutation on the same object or workspace deadlocks.

## Pending reservations

Everything below is decided under the registry mutex.

**Lookup.** The lookup returns a decision rather than a pointer: conflict, expired, replay (with a detached copy of the result), no room, or reserved.

**Reservation.** When the store has no live entry for the key and there is room, the lookup inserts a pending operation that carries the workspace, user and digest. It counts towards the shares at once.

- A pending operation is not in the creation order and never expires.

**Meeting a reservation.** Another call that meets a pending operation gets `idempotency_conflict` when the workspace, user or digest differs. With an identical scope, which can happen after an alias moved to another key, it gets `dependency_unavailable` (retryable) instead.

**Completion.** Completion happens exactly when HB-176 stores an operation: completed, accepted, or the unknown end of a mutating handler. It stores the result in place, clears the pending mark, sets the creation time (raised to at least the previous one, so the creation order stays monotonic, with the HB-177 fallbacks for failed clock reads) and appends the operation to the creation order. Only the owning call completes or abandons its reservation, and nothing else removes a pending operation, so the map still points at it. Completion never adds to the shares a second time.

**Abandonment.** The reservation is removed, and its share released, when the call ends any other way:

- a revision conflict;
- a recheck refusal;
- a random-source failure;
- a named handler error;
- a panic outside the handler, which `Execute` recovers;

Because abandoned reservations never enter the creation order, refused calls leave nothing behind.

**Journal ordering.** A conflict with a pending operation on another target can be recorded before the original's record, and a conflict with a reservation that is later abandoned depends on timing. Both are acceptable under the key contract (a key belongs to one request).

Replay, tombstone, retention, share and rate-limit rules are unchanged.

# Verification plan

Literal tests:

- **Concurrency by object.** A slow handler on one keyed target does not delay a mutation of another keyed target. It does delay a mutation of the same key, including one through another spelling that resolves to the same key.
- **Workspace fallback.** Targets without keys serialize with each other and with keyed targets of the same workspace.
- **Moved object.** A key that changes between the two resolutions returns `dependency_unavailable` without applying anything.
- **Same key, two targets.** Concurrent calls with the same idempotency key on two targets: exactly one executes, and the other gets `idempotency_conflict`.
- **Abandonment.** A reservation is freed after a revision conflict, a recheck refusal, a named handler error, a panic outside the handler and `runtime.Goexit`. After each, the same key executes, and the creation order does not grow.
- **Handler panic.** A panic inside the handler is stored as `internal_error`.
- **Pending state.** A pending reservation holds capacity, never expires under a forward clock step, and survives a sweep.
- **Monotonic creation.** Creation times stay monotonic.
- **Race detector.** A replay on one target is read while another target is stored, under `-race`.
- **Reading commands.** These take no lock.
- **Empty at the end.** The lock table and the shares are empty after every path.
- **Lock unit test.** The lock table, run for two workspaces with the same key.
- **Existing tests.** The HB-176 and HB-177 tests still pass, with resolve counts updated for the extra resolution.

A property test runs concurrent random mutations over aliased and distinct targets and a few keys, with random handler delays and failures. It checks:

- no two applies to one object overlap;
- every key executes at most once per retention;
- the lock table and capacity counts are empty at the end.

Checks: `-race -count=10`, a 386 build, gofmt and vet, the whole repository with `-race`, coverage of the new code and mutation testing.

# Review history
- Design review round 1: FAIL (1 high, 3 medium, 4 low). Folded in: canonical keys from the handler with a shared workspace lock, an exclusive workspace fallback and a second resolution that detects moved objects (H1); lookups that decide under the registry mutex and return copies (M1); pending operations outside the creation order until completion (M2); pending operations that never expire and are only completed or abandoned by their owner (M3); monotonic creation times (L1); journal ordering of conflicts (L2); a separate deferred release (L3); no re-entry into `Execute` (L4); the added tests.
- Design review round 2: PASS (2 medium, 2 low). Folded in: not-found, unavailable and failed first reads take no lock (N1); identical-scope calls that meet a reservation get `dependency_unavailable` (N2); creation times set at completion (N3); lost keys and objects count as moved, and `Resolve` must be safe alongside `Apply` (N4).

# Change review resolution

- Change review round 1: PASS (6 low). Resolved:
  - L1: the property test now sleeps inside the apply window, detects keyless and keyed applies overlapping in one workspace, checks that handlers receive the locked key, and requires successes. Removing the key mutex makes both the literal and the property test fail (checked on a scratch copy).
  - L3: a handler that leaves through `runtime.Goexit` once apply has started is completed as an unknown end (`internal_error` with the operation, next action `inspect_operation`) instead of being abandoned, so a retry replays and never applies again.
  - L4: `Context.Key` carries the locked canonical key to the handler.
- Recorded as residuals:
  - L2: random-source failures cannot be injected; their abandonment is verified by reading.
  - L5: within one workspace, lock waits can reveal by timing whether a target exists.
  - L6: clamped creation times lengthen replay windows after a forward clock jump.
