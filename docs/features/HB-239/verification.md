---
type: VerificationRecord
title: Event outbox and inbox verification
description: Records what was executed and reviewed for HB-239, the second increment of work item W-34. It covers registered event types, the CloudEvents outbox, relay reads with expiring cursors, source bindings, and the two-phase inbox. The inbox deduplicates by source and identifier, orders by source and subject revisions, and uses fencing reservations, a seen queue and bounds. The record lists results, the mutation check and what remains unverified.
status: draft
---

# Delivered behavior

[HB-239](https://linear.app/hybrid-billing/issue/HB-239) adds the Go package `modules/events` from main `6375ea4`. It uses only the standard library.

## Event types and outbox

- **Event types.** Types are registered under dual control by name and schema version, with at most 16 field names that events may carry as identifier values.
- **Publishing and storage.**
  - System actors publish CloudEvents 1.0 events with a canonical revision.
  - Repeats ignore time.
  - Time is monotonic.
  - Each workspace has its own sequence.
  - Events are retained for a set period, and expired events are swept before the capacity check, which refuses rather than evicts.
- **Body.** `Body` renders sorted structured JSON with the `schemaversion` and `objectrevision` extensions. A known-answer test pins it.
- **Reads.** Only relays may read. Pages are bounded and carry the head. A cursor below the floor returns `Expired{Resync{Floor, Head}}`, and one beyond the head is invalid.

## Inbox

- **Source bindings.** These are dual-controlled: one connection per source.
- **`Receive` checks**, in order:
  1. the verified identifiers;
  2. the binding;
  3. seen identifiers, giving duplicate or conflict by digest;
  4. committed revisions per source and subject, giving duplicate or stale, including after a seen identifier expires;
  5. pending reservations per subject and per key, giving in progress.
- **Reservations.** An accepted event gets a fencing reservation. Expired predecessors are released before the bound checks.
- **`Commit` and `Abandon`.** `Commit` marks the event as seen and records the revision with its identifier and digest. A defensive recheck stays in place. Late commits of unreplaced reservations are accepted, and replaced ones get `ErrStale`. `Abandon` frees the reservation.
- **Bounds.** Seen identifiers plus reservations stay within `Seen`. Revision records, including pending new subjects, stay within `Revisions`.
- **Queues.** Expiry-ordered queues for seen identifiers and reservations stay proportional to what is open.
- **Data handling.** Reservations keep no data.

# Commands and results

All runs were made in the worktree `/private/tmp/hb239` on 2026-11-15, using Go 1.25.7 on darwin/arm64.

| Check | Result |
| --- | --- |
| `gofmt -l modules/` | no output |
| `go vet ./modules/events` | pass |
| `go test -race -count=10 ./modules/events` | pass |
| `GOOS=linux GOARCH=386 go test -c ./modules/events` | builds |
| `go test -race -count=1 ./...` | 51 packages, all pass |
| Coverage | `modules/events` 100.0% |
| Mutation testing | 409 mutants of `registry.go` and `inbox.go`: 373 killed, 17 did not build, 19 survived |

The 19 survivors were read and judged equivalent:

- **Compaction (7).** Compaction guards that only copy an unchanged slice or compact earlier.
- **Map clean-up (3).** Pointer checks in map clean-up. Replacement always releases first, so the maps never point elsewhere.
- **Rebuild threshold (1).** The rebuild threshold boundary, which affects cost only.
- **Number formatting (1).** The token number base.
- **Repeat lookup (1).** The identifier and source comparison in repeat lookups, where both are equal by key.
- **Revision parsing (4).** Ignored values and digit checks that `ParseUint` already enforces.
- **Shared patterns (2).** A `[][3]` limits array and `Round(1)`.

## Property tests

Two property tests run, each with 100 seeds:

- **Inbox.** The test runs 120 steps of up to 20 minutes against an independent model, using two sources that share subjects. It covers:
  - conflicting content;
  - events without a subject or revision;
  - commits and abandons;
  - late commits of expired reservations;
  - seen identifiers expiring.

  It checks every decision, and that every ranked event is applied at most once.
- **Cursors.** Random publications, sweeps and reads are checked against the timestamps.

# Reviews

- **Design review.** Four rounds by a separate read-only reviewer:
  - round 1 FAIL, after which the ticket was split and deliveries moved to HB-240;
  - rounds 2 and 3 FAIL, on reservation ordering and replacement;
  - round 4 PASS ([plan](plan.md)).
- **Change review.** Three rounds by a fresh read-only reviewer:
  - round 1 FAIL: a subject blocked at the revision bound, and scans under the mutex;
  - round 2 FAIL: an unbounded expiry queue;
  - round 3 PASS.

# Provisional choices for the owner

- Up to 16 data fields.
- Sources limited to 256 printable ASCII bytes, without `@` or backslashes.
- Revisions as unsigned 64-bit decimals.
- A reservation duration from 1 second to 24 hours.
- Retention bounds from 1 hour to 10 years.
- Refusal instead of eviction at every bound.

# Not verified and residuals

- **HB-240.** Webhook subscriptions, deliveries, signing, key rotation, the send-time recheck, and the per-consumer low watermarks on the sweep.
- **Atomicity and lost events.** The outbox write is not atomic with the business change, so a publication refused after a commit loses the event. The business apply must be idempotent by source and identifier, or atomic with `Commit`.
- **Trust receipts.** Trust gives no receipt that binds identifiers to the signed body.
- **Reservation tokens.** Tokens are counters that system actors can guess.
- **Revision records.** There is no purge of revision records.
- **Deduplication after expiry.** Events without a subject or revision are deduplicated only within the inbox retention.
- **Not yet built.** Mapping the reasons to the commands error catalogue, cursor epochs, and persistence.
