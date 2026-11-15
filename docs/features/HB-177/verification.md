---
type: VerificationRecord
title: Command retention, capacity and rate limits verification
description: Records what was executed and reviewed for HB-177, the second increment of work item W-33. It gives command operations a replay window with tombstones until the retention period ends and journal records a minimum retention. Both stores get user, workspace and workspace-count shares, with unrecorded refusals aggregated, and every admitted caller gets a token bucket with a stable rate_limited error. The record lists the results, the mutation check, the provisional choices and what remains unverified.
status: draft
---

# Delivered behavior

[HB-177](https://linear.app/hybrid-billing/issue/HB-177) changes `modules/commands` and `modules/assistant` from main `953163b`.

- **Rate limits.** Each workspace and user has a token bucket. Refill happens in whole intervals and is computed in durations, so 32-bit builds are safe.
  - Clock handling:
    - A backward step smaller than one interval is ignored.
    - A larger step restarts the interval.
    - A failing clock changes nothing.
  - A call without a token returns `rate_limited` (429, retryable) with `RetryAfter` in (0, `Refill`]. On the wire this is `retry_after_ms`, rounded up.
  - Rate-limited calls leave no record and no operation. They are counted per user, with first and last times (`Throttled`).
- **Journal.**
  - Shares and modes:
    - Records need room in the user, workspace and workspace-count shares. The limits require `Workspaces × WorkspaceJournal ≤ Journal` and `Users × UserJournal ≤ WorkspaceJournal`.
    - Without room, a call runs in unrecorded mode. Early refusals keep their own code, and anything that would reach a handler ends `dependency_unavailable`. Both are counted per user and code (`Unrecorded`), and nothing runs without a record.
  - Retention and reading:
    - Records are kept for at least `Retention`. They are swept from the oldest end on every reservation, and the store is compacted when half of it is dead.
    - Records whose clock read failed take the previous record's time, then the reservation time, then the next record's time.
    - `Journal` searches by sequence, and `JournalStart` gives the oldest retained sequence.
- **Operations.**
  - Lifecycle:
    - Within `Replay`, an operation replays its result.
    - From `Replay` until `Retention`, it is a tombstone, and the same key returns `idempotency_key_expired` (409, not retryable). Unknown effects follow the same rule.
    - After `Retention`, it is removed.
  - Capacity: operations need room in the user, workspace and workspace-count shares. Tombstones count towards these shares.
  - Sweeps delete an entry only when the map still points at that entry's own operation. A key whose own entry is past its retention is dropped before the capacity check.
- **Contract.**
  - Two new codes in the catalogue and in the OpenAPI enum: `rate_limited` and `idempotency_key_expired`.
  - `retry_after_ms` added to the error schema.
  - 429 is listed on every operation; `idempotency_key_expired` only on mutating ones.
  - The golden document is updated.
- **Assistant.** `Reply.RetryAfter` carries the executor's value. Rate-limited calls still count against the job budget.

# Commands and results

All runs were made in the worktree `/private/tmp/hb177` on 2026-11-15 with Go 1.25.7 on darwin/arm64.

| Check | Result |
| --- | --- |
| `gofmt -l modules/` | no output |
| `go vet ./modules/commands ./modules/assistant` | pass |
| `go test -race -count=10` on both packages | pass |
| `GOOS=linux GOARCH=386 go test -c ./modules/commands` | builds |
| `go test -race -count=1 ./...` | 50 packages, all pass |
| Coverage | `modules/commands` 99.5%, `modules/assistant` 100.0% |
| Mutation testing | 423 mutants of `store.go`, `execute.go`, `types.go` and `openapi.go`: 385 killed, 13 did not build, 25 survived |

The uncovered statements are not reachable from tests:
- two random-source failures;
- the default branch of `Admit`;
- the error text of `Failure` before this change.

The 25 survivors were read.

**Twelve are in new code and are equivalent:**
- **Redundant bounds (3).** The workspace and installation share checks can never be reached first, because the product rules ensure the user shares fill earlier.
- **Refill branches (3).** In each, the mutated branch computes the same tokens and times.
- **Sort order (3).** `<=` in sorts of distinct keys.
- **Compaction guards (2).** `>= 0` guards make compaction copy an unchanged slice.
- **RetryAfter (1).** `< 0` instead of `<= 0` in `RetryAfterMillis`.

**Thirteen are in HB-176 code that this ticket does not change:**
- the integer pattern;
- the rationale, value and outcome-field bounds;
- payload buffer sizes;
- the character ranges of `validID`;
- a random-source failure;
- the emptiness condition of a named handler error.

They are recorded for a follow-up rather than fixed here.

**Property test.** It runs 20 seeds of 150 calls by three users over time. Calls are rate limited, early refusals, reads and keyed mutations with repeated payloads. They are compared with an independent model of:
- buckets, simulated one interval at a time;
- record retention and shares;
- operations, tombstones and revisions.

On every call it checks the code, the replay flag and whether a handler ran. At the end it compares the held user counts with the model. Every outcome class is reached.

# Reviews

- Design review: three rounds by a separate read-only reviewer: FAIL, FAIL, then PASS ([plan](plan.md)).
- Change review: two rounds by a fresh read-only reviewer: FAIL (32-bit refill, isolation tests and property strength), then PASS.

# Provisional choices for the owner

- **Defaults:** replay 24 hours, retention 400 days, burst 60, one token per second, 1,000 workspaces, shares sized to the expected users.
- **Retention behaviour:** users and workspaces stay counted, and tombstones hold shares, for the whole retention. Shares are not tied to the rate.
- **Assistant budget:** rate-limited calls count against assistant job budgets.

# Not verified and residuals

- **Exposure blocker.** The installation-wide mutation lock remains: a slow handler stalls mutations in every workspace.
- **Isolation.** The identity register holds one workspace per installation, so workspace isolation is tested on the store, not end to end.
- **Storage and keys.**
  - Durable storage and export of records before eviction are not built.
  - Persistence and custody of the digest key are not built.
- **Rate limiting.**
  - Rate limits per client or command are not built.
  - Distributed rate limiting is not built.
  - The HTTP mapping of `rate_limited` to `429 Retry-After` is not built.
  - Unauthenticated floods are not limited here.
- **Counters and contract.**
  - Throttle and unrecorded counters are never pruned.
  - The command contract document still has to describe the replay window, `idempotency_key_expired` and key reuse after the retention.
  - Handlers must change the target revision on every completed or accepted mutation.
