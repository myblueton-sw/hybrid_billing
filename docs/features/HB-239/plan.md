---
type: ImplementationPlan
title: Event outbox with registered minimal payloads and expiring cursors, and a two-phase deduplicating event inbox bound to connections
description: Defines the second increment of work item W-34. A register publishes CloudEvents 1.0 events into a per-workspace outbox. Event types and their allowed identifier fields are registered under dual control, events are retained for a bounded time, and system relays read with cursors that expire with a clear resynchronisation point. An inbox accepts events only from sources bound to the delivering connection, reserves them until the caller commits or abandons processing, deduplicates by source and identifier with content comparison, and ignores events older than the stored revision of the same source and subject.
status: draft
---

# Scope and ownership

[HB-239](https://linear.app/hybrid-billing/issue/HB-239) continues work item W-34 (requirement HB-34; HB-34-D/I/V; the event, cursor and inbox parts of QA-05) after HB-180 (`modules/jobs`). Webhook subscriptions and deliveries follow in HB-240. Base: main `6375ea4`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the plan and the change, because the change decides what leaves the installation and when an incoming event is applied. Machine: mac_mini.

Allowed files: `modules/events/*`, `docs/features/HB-239/*`.

Out of scope, recorded as residuals:
- webhook subscriptions, deliveries, signing, key rotation and send-time rechecks (HB-240);
- the mapping of `ErrExpired`, `inbox_capacity` and the other reasons to the commands error catalogue;
- cursor epochs after a store reset;
- the HTTP sender and receiver, TLS and the dialer, which call `modules/trust` for addresses and redirects;
- atomicity of an outbox write with the business change, which needs a shared store (the register exposes an append that a store transaction can wrap later);
- persistence;
- ERP polling and reconciliation (DB-12);
- source event semantics (RT-01);
- business event types;
- screens.

# Observed baseline

- **QA-05 contract.**
  - Job states are queued/running/waiting/succeeded/failed/cancelled.
  - Events preserve CloudEvents 1.0 `id`, `source`, `type`, `specversion`, `subject` and `time`, plus a schema version. They are deduplicated by source and id and checked against the object revision, assuming at-least-once, duplicate and out-of-order delivery.
  - Webhooks require signature, time, event id, key rotation and replay protection.
  - Subscription and send permissions are rechecked. Payloads identify things and carry nothing more.
  - Destinations are approved, with a private allowlist; metadata, arbitrary loopback, DNS rebinding and unapproved redirects are blocked.
  - An expired cursor gets a clear error and a path to resynchronise.
  - Errors carry a stable code, request id, field errors, retry flag, next action and 429 retry guidance.
- **AUD-QA-05.** It tests job enums, waiting when the result is unknown, duplicates, out-of-order events, expired cursors, signature forgery, replay, destination bypass, permission revocation just before sending, and 429.
- **Existing modules.**
  - `modules/jobs` (HB-180) holds jobs, tasks and fenced leases.
  - `modules/trust` (HB-237) holds approved destinations, address and redirect checks, and inbound signature verification for pushes.
  - `modules/commands` (HB-176/177/238) holds the error catalogue and rate limits.
- **Sibling conventions.** Registers use one mutex, an injected clock, the standard errors and dual control for configuration.

# Decision

A new package `modules/events` follows the sibling conventions:
- one mutex and an injected clock that never goes backward;
- `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (`Refusal{Reason}`), `ErrLimit`, and a new `ErrExpired`;
- check order: form, lookup, repeat, rule, room;
- human actors for configuration under dual control, with assistant (`ai_not_allowed`) and system (`not_human`) actors refused for configuration;
- system actors for relays.

The constructor is `NewRegistry(Limits{Workspaces, Types, Events, Retention, Page, Sources, Seen, InboxRetention, Revisions, Reservation}, clock)`. It uses the standard library only and calls no callbacks, so nothing runs under the mutex except the register's own code.

Webhook subscriptions and deliveries move to HB-240. The findings of this ticket's design review that concern them are listed there as inputs (see Review history).

## Event types

`ProposeType(workspace, Type{Name, SchemaVersion, Fields}, actor)` by one human and `ApproveType` by another register an event type version.

- `Fields` lists at most 16 identifier field names that events of this type may carry in `Data`. Field names must match `^[a-z][a-z0-9_]{0,63}$`.
- Registered versions are immutable.

Payload minimisation stays the publisher's responsibility: values are identifiers, never amounts, names or text. The registry makes it checkable by type.

## Events and the outbox

`Publish(workspace, Event{ID, Source, Type, SchemaVersion, Subject, ObjectRevision, Data}, actor)` by a system actor appends an event.

Form rules:
- **Identifiers.** `ID`, `Type` and `Subject` follow the identifier rule. `Subject` is optional.
- **Source.** `Source` is a URI reference of at most 256 printable ASCII bytes, without user information.
- **Type and data.** `Type` and `SchemaVersion` must name an approved type version. `Data` keys must be among its fields, and its values must be identifiers.
- **Object revision.** `ObjectRevision` is either empty or a canonical unsigned decimal: digits only, no leading zero, at most 20 digits, at most 2⁶⁴−1.

Stored values:
- `specversion` is `1.0`.
- `time` is the register clock, raised to at least the previous event's time.

Other rules:
- **Repeats.** A repeated `(source, id)` with the same content (time excluded) returns the stored event. Different content is `ErrConflict`. Once an event has been swept by retention, the same `(source, id)` is published as a new event, so consumers deduplicate by `(source, id)`.
- **Sweep before capacity.** Expired events are swept before the capacity check.
- **Sequence and retention.** Each workspace numbers its events in order and keeps them for `Retention`, swept from the oldest end.
- **Capacity.** A workspace holds at most `Events` events; beyond that it gets `ErrLimit`, never silent eviction. Until outbox writes are atomic with business changes, a publication refused after a business commit is a known loss path (residual).

`Body(event)` renders the CloudEvents structured JSON. The keys are sorted, and the extension attributes are `schemaversion` and `objectrevision` (lowercase alphanumerics, as CloudEvents requires). A known-answer test pins it.

`Read(workspace, after, limit, actor)` is for system relays only. External consumers receive events through HB-240 subscriptions, whose sends recheck permissions. It returns events with sequences above `after`, at most `min(limit, Page)`, together with the current head.

The register keeps a floor: the last removed sequence, which is 0 when nothing has been removed.
- If `after` is below the floor, events were removed, and the call returns `ErrExpired` with a `Resync{Floor, Head}` value. The relay records the head, rebuilds its state from current objects, then reads after the head.
- A cursor beyond the head is `ErrInvalid`.
- A fresh workspace has floor 0 and head 0. `Retention` must exceed a relay's rebuild time, or the relay meets `ErrExpired` again. HB-240 will add per-consumer low watermarks to this sweep, so retention can be pinned for open deliveries without changing the cursor rules.

## Inbox

`BindSource(workspace, source, connection, actor)` by one human and `ApproveSource` by another bind an inbound source to one `modules/trust` connection. A source belongs to at most one connection in the workspace. `UnbindSource` by a human ends the binding.

`Receive(workspace, connection, verified, Event, actor)` by a system actor reserves an incoming event. The caller must call it only for an event of a push that `modules/trust` accepted for that connection, and must pass that push's event identifiers as `verified`. Trust gives no receipt that binds identifiers to the signed body digest, so this remains a caller obligation (residual).

**Form.** Inbound events follow the same form rules as published events, with one exception: the type need not be registered here. The rules are identifiers, the source rule, the canonical revision, and at most 16 data fields with field-name keys and identifier values.

**Digest.** The content digest is SHA-256 over the length-prefixed identifier, source, type, schema version, subject and object revision, followed by the decimal count of data pairs and then the pairs sorted by name. Each part is preceded by an 8-byte big-endian length, and the sender's time is excluded. A known-answer test pins it.

**Checks.** These run in this order, and the first failure decides:

1. The identifier is among `verified` (`unverified`).
2. The source is bound, active, to this connection (`source_not_bound`).
3. The same `(source, id)` is already seen: with the same digest it is `duplicate`, with a different digest it is `conflict`.
4. The event has a subject (the revision rules need both a subject and a revision):
   - The stored revision for `(source, subject)` is higher; or it is equal and was committed by another identifier or digest. The event is `stale`.
   - The stored revision is equal and was committed by this same identifier and digest. The event is `duplicate`, so a redelivery after its seen identifier expired is still not applied twice.
   - Another reservation is pending for the same subject: `in_progress`, whatever the revisions. `stale` is decided only against committed revisions, because a pending reservation can still be abandoned, and a connector must retry rather than drop an event. At most one reservation is pending per `(source, subject)`.
5. Another reservation is pending for the same `(source, id)` (`in_progress`).
6. Otherwise the event is reserved for `Reservation` and returned with a fencing token.

An expired reservation no longer blocks its key or subject; the next `Receive` replaces it.

`Commit(workspace, token, actor)` records the identifier as seen and raises the stored revision of its subject, keeping the committing identifier and digest. `Abandon(workspace, token, actor)` drops the reservation.

**Late commits.** A commit on an expired token is still accepted if its reservation has not been replaced. A reservation is replaced by any later `Receive` for the same `(source, id)` or `(source, subject)` after it expired, even one that is then refused for capacity, or by its removal when the capacity sweep runs. A replaced reservation's commit returns `ErrStale`. `Commit` also rechecks that its revision is still higher than the committed one, and returns `ErrStale` otherwise. Commits and abandons of reservations made before an `UnbindSource` are still accepted. The business apply must be idempotent by `(source, id)`, or happen in the same transaction as `Commit` (atomicity residual). So an event is never marked seen before its caller has applied it, and two workers cannot both commit it.

**Seen identifiers.**

- They are kept for `InboxRetention`, in an expiry-ordered queue swept from the oldest end.
- Seen identifiers and open reservations together count toward `Seen` per workspace. When the store is still full after sweeping seen identifiers and expired reservations, `Receive` refuses with `inbox_capacity`.

**Revision records.**

- They have their own bound, `Revisions` per workspace, and never expire, so a late older event is always recognised.
- Every unreleased reservation for a subject without a record, expired or not, counts toward the bound, so late commits cannot exceed it.
- `verified` holds at most 10,000 identifiers.
- Sources are checked by their bytes, not parsed as URI references: printable ASCII without spaces, `@` or backslashes.
- When full, a new subject is refused with `revision_capacity`, while existing subjects keep working.
- A dual-controlled purge, for example on unbinding, is a residual.

Events without a subject or revision are only deduplicated. Accepted events' data are returned to the caller by `Receive` and never stored; `Commit` returns the identifiers only.

# Verification plan

Literal tests:
- **Types and publication.**
  - Every form rule of types and events, every repeat, and dual control.
  - Data restricted to registered fields; revision canonical form, including leading zeros, overflow and 20-digit limits.
  - Monotonic time, sequences, the retention sweep and the capacity bound.
  - The known-answer body.
- **Read.** Only system relays may read.
- **Cursors.**
  - Cursors at the floor, below it and beyond the head.
  - A fresh workspace, and after a full sweep.
  - The `Resync` values.
- **Source bindings.** Dual control and unbinding.
- **Inbox decisions.**
  - Unverified identifiers, and spoofed sources on another connection.
  - Duplicates and conflicts.
  - `in_progress`.
  - Stale, equal and newer revisions, with the same subject from two sources.
- **Inbox reservations.**
  - Commit, abandon, and reservation expiry followed by redelivery, which covers a crash before commit.
  - Overlapping reservations on one subject (higher in progress, lower stale).
  - A redelivery after the seen identifier expired (duplicate by revision record).
  - The reservation flood bound.
  - Late commits with and without a replacement.
  - The digest known-answer test.
  - The capacity sweep of seen identifiers, revision capacity, and events without a subject or revision.

Property tests:
- random publications and reads against an independent cursor model;
- random inbox sequences with duplicates, reordering, crashes and commits against an independent model.

Checks:
- `-race -count=10`;
- a 386 build;
- gofmt and vet;
- the whole repository with `-race`;
- 100% coverage;
- mutation testing.

# Review history
- Design review round 1: FAIL (6 high, 15 medium, 6 low). The ticket was split: webhook subscriptions and deliveries move to HB-240.
  - Folded in here:
    - two-phase inbox reservations (H1);
    - relay-only reads (H3);
    - revisions keyed by source and subject, connection-bound sources, verified identifiers and a separate revision bound (H6);
    - registered data fields per type (M3, M4);
    - extension attributes in the known-answer body (M15);
    - the cursor floor, the head and resynchronisation (M10);
    - content-compared duplicates (M11);
    - no callbacks under the mutex (M13, for this register);
    - an expiry-ordered seen queue (L2);
    - canonical revisions and repeats that ignore time (L4);
    - cursor epochs (L5) as a residual.
  - Inputs for HB-240:
    - deliveries as per-subscription cursors that never block publication, and retention pins or explicit expiry (H2, H4);
    - a redirect policy for senders (H5);
    - a send-time confirmation (M1);
    - fallible permits (M2);
    - signing of delivery id, key version and attempt time, the receiver contract and key rotation (M5, M6);
    - throttle, rejection, retry-after, jitter and budget semantics (M7);
    - pause and resume (M8);
    - fairness (M9);
    - destination validation (M12);
    - the mapping of delivery states to QA-05 (M14);
    - late reports, listing and redelivery control (L1, L3, L6).
- Design review round 2: FAIL (1 high, 4 medium, 6 low). Folded in:
  - one pending reservation per source and subject (R1);
  - revision records that keep the committing identifier and digest (R2);
  - reservations counted toward `Seen` (R3);
  - fencing tokens with defined late commits and the idempotent-apply contract (R4);
  - inbound form rules and the digest (R5);
  - repeats after retention and sweeps before capacity (L-a, L-b);
  - residuals for revision purge, trust receipts, rebuild time and the HB-240 sweep hook (L-c to L-f).
- Design review round 3: FAIL (1 high, 2 medium, 2 low). Folded in: `in_progress` for any pending subject, with `stale` only against committed revisions (N1); replacement by any later reservation of the same subject and a revision recheck at `Commit` (M1); pending new subjects counted toward `Revisions` at `Receive` (M2); revision rules requiring both subject and revision, and commits after unbinding (L1, L2).
- Design review round 4: PASS (1 medium, 4 low). Folded in:
  - expired but unreleased reservations count toward `Revisions` (M-a);
  - the digest text includes the data count (L-a);
  - `verified` is bounded (L-b);
  - a JSON-escaped source in the known-answer body (L-c);
  - byte-level source rules (L-d).

# Change review resolution

- Change review round 1: FAIL (2 medium, 5 low). Resolved:
  - M1: expired predecessors of the same key or subject are released before the capacity and revision checks, so a crashed reservation never blocks its own redelivery at the revision bound. A regression test covers it.
  - M2: pending new subjects are counted incrementally, and reservations sit in an expiry-ordered queue swept from the oldest end, so no `Receive` scans all reservations under the mutex.
  - L1: the inbox property test covers:
    - steps of up to 20 minutes, so seen identifiers expire;
    - two sources sharing subjects;
    - conflicting content;
    - events without a subject or revision;
    - commits of expired but unreplaced reservations.
  - L2: the `Commit` recheck is kept as a defensive guard that the public API cannot reach (one reservation per subject); its test sets the state directly.
  - L3: unused reason constants are removed.
  - L4: reservations keep identifiers and the digest, not the data.
  - L5: reservation tokens are counters. System actors in a workspace are trusted with each other's tokens (residual).
- Change review round 2: FAIL (1 high, 1 low). Resolved: the expiry queue drops released entries from its head and is rebuilt when it holds more than twice the open reservations plus one, so it stays proportional to them; a regression test runs 500 cycles with an old expired reservation at the head (H1). The plan now states that a refused `Receive` also replaces expired predecessors (LOW-A).
- Change review round 3: PASS. Notes: the `Commit` revision recheck is reachable only by setting state directly; `release` relies on callers never passing a reservation twice, which every current caller guarantees.
