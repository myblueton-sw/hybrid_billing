---
type: VerificationRecord
title: Mutation locks per object verification
description: Records what was executed and reviewed for HB-238, the third increment of work item W-33. It replaces the installation-wide mutation lock with a shared workspace lock plus a mutex per canonical object key, an exclusive workspace fallback for objects without a key, and no lock for calls that cannot apply. Idempotency keys are reserved as pending operations, and each reservation is either completed or abandoned on every exit path. The record lists results, the mutation check and what remains unverified.
status: draft
---

# Delivered behavior

[HB-238](https://linear.app/hybrid-billing/issue/HB-238) changes `modules/commands` from main `2b3b27e`.

**Locks**

- **Choosing the lock.** A mutating call first resolves its target outside any lock. Then:
  - a found object with a canonical `Target.Key` takes the workspace lock shared, plus the mutex for that key;
  - a found object without a key takes the workspace lock exclusively;
  - an object that was not found, was unavailable, or failed to read takes no lock and cannot apply.
- **Second resolution.** Under the lock the call resolves again. If the object moved, lost its key, disappeared, or appeared without a lock, the call returns `dependency_unavailable` and applies nothing.
- **Lock table.** The table counts references and deletes an entry only after its lock is released. The release runs as its own deferred step, after the journal record is written.
- **Context.** `Context.Key` carries the locked key to the handler.

**Reservations**

- **Reserving.** The operation lookup decides under the registry mutex and returns copies. A free key with room becomes a pending operation, which:
  - counts towards the shares;
  - stays out of the creation order;
  - never expires.
- **Meeting a reservation.**
  - A different scope gets `idempotency_conflict`.
  - An identical scope gets `dependency_unavailable`.
- **Completion and abandonment.**
  - **Completed:** when HB-176 stores an operation. The creation time is set at completion and kept monotonic.
  - **Abandoned:** on every other exit, including a revision conflict, a recheck refusal, a named handler error and a panic outside the handler.
  - **Goexit:** a handler that leaves through `runtime.Goexit` after apply began is completed as an unknown end (`internal_error`, `inspect_operation`), so a retry replays instead of applying again.

# Commands and results

All runs were made in the worktree `/private/tmp/hb238` on 2026-11-15 with Go 1.25.7 on darwin/arm64.

| Check | Result |
| --- | --- |
| `gofmt -l modules/` | no output |
| `go vet ./modules/commands` | pass |
| `go test -race -count=10 ./modules/commands` | pass |
| `GOOS=linux GOARCH=386 go test -c ./modules/commands` | builds |
| `go test -race -count=1 ./...` | 50 packages, all pass |
| Coverage | `modules/commands` 99.5%; the uncovered statements are the two random-source failures and the default branch of `Admit`, which existed before |
| Mutation testing | 328 mutants of `store.go` and `execute.go`: 306 killed, 1 did not build, 21 survived |

The 21 survivors were read:

- **11 in the HB-177 store code, equivalent.** These are the same kinds recorded for HB-177: redundant bounds, refill branches, sorts of distinct keys and compaction guards.
- **10 in HB-176 code that this ticket does not change.** These are the integer pattern, byte bounds, buffer sizes, a random-source failure and the emptiness condition of a named error, recorded for a follow-up.

The new tests cover:

- **Concurrency by object.**
  - Another object proceeds during a slow handler.
  - A missing object and a reading command do not wait.
  - An alias of the same object waits and then meets a revision conflict.
  - A keyless object waits, and it excludes keyed objects while it runs.
- **Moved objects, in five forms.**
- **Pending reservations.**
  - The same key on another object conflicts.
  - Pending operations hold capacity.
  - An identical retry after an alias moved gets `dependency_unavailable`.
  - A pending operation survives a forward clock step.
- **Abandonment and Goexit.**
- **The lock table** across two workspaces, and the shared/exclusive interaction.

A concurrent property test runs 10 seeds with 8 workers, over aliased, distinct and keyless targets, with random delays and failures. It checks:

- no overlapping applies per object or across a keyless one;
- handlers receive the locked key;
- at most one successful apply per idempotency key;
- shares, order and lock table consistent and empty at the end.

Removing the key mutex on a scratch copy makes both the literal and the property test fail.

# Reviews

- Design review: two rounds by a separate read-only reviewer: FAIL (aliases, races, order growth, pending expiry), then PASS ([plan](plan.md)).
- Change review: one round by a fresh read-only reviewer, PASS with low findings, which were folded in (property strength, Goexit after apply, `Context.Key`).

# Not verified and residuals

- **Untested paths.** Random-source failures cannot be injected; their abandonment is checked by reading only.
- **Handler obligations.**
  - Handlers must return stable canonical keys.
  - `Resolve` must be safe alongside `Apply`.
  - Handlers must not re-enter `Execute`.
  - Writes to objects other than the resolved one must be serialized by the handler, or the handler must omit `Key`.
- **Timing.** Within one workspace, lock waits can reveal by timing whether a target exists. Clamped creation times lengthen replay windows after a forward clock jump.
- **Not built here.** Handler time limits, locks across processes, and end-to-end tests across two workspaces, which must wait for multi-workspace identity.
