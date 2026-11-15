---
type: ImplementationPlan
title: Webhook subscriptions and deliveries as per-subscription cursors over the event outbox, with confirmation before each send, signed attempts, throttling, budgets and dead letters
description: Defines the third increment of work item W-34. A register delivers outbox events to dual-controlled webhook subscriptions. Each subscription advances its own cursor, so receivers can never block publication, and a subscription whose cursor falls behind retention is suspended explicitly. Dispatch is fair and fenced. A confirmation just before each send rechecks the subscription, the destination and a fallible permission check, then signs the attempt with an injected signer. Delivered, rejected, throttled, failed and unknown outcomes drive backoff with jitter, attempt budgets and redeliverable dead letters, and delivery states map to the QA-05 job states.
status: draft
---

# Scope and ownership

[HB-240](https://linear.app/hybrid-billing/issue/HB-240) continues work item W-34 (requirement HB-34; the webhook parts of QA-05) after HB-239 (`modules/events`). Base: main `fe2fec4`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the plan and the change, because this register decides what leaves the installation, and when. Machine: mac_mini.

Allowed files: `modules/webhooks/*`, `modules/events/registry.go` and its tests (the `Bounds` method), `docs/features/HB-240/*`, and `docs/features/HB-239/plan.md` (deviation note).

Inputs: the HB-239 design review findings on deliveries (H2, H4, H5, M1, M2, M3, M5, M6, M7, M8, M9, M12, M14, L1, L3, L6). Each one is answered below.

Out of scope, recorded as residuals:

- the HTTP sender, including TLS and the dialer; senders must use `Redirect` or refuse redirects (H5);
- publishing receivers' public keys and the rotation ceremony (M6, follow-up ticket); the signer owns keys and versions;
- retention of delivery snapshots beyond the outbox retention (owner decision);
- persistence;
- business event types;
- screens.

# Observed baseline

- `modules/events` (HB-239) keeps a per-workspace outbox with sequences, retention, a floor and a head. Relays read it with `Read(workspace, after, limit, actor)`, which returns `Expired{Resync}` below the floor. `Body(event)` renders the CloudEvents JSON.
- `modules/trust` (HB-237) approves destinations and checks addresses and redirects, but `CheckRedirect` accepts any active destination of the workspace.
- QA-05 requires webhook signature, time, event identifier, key rotation and replay protection. It also requires subscription and send permissions to be rechecked, minimal payloads, approved destinations, 429 retry guidance, and the six job states.

# Decision

A new package `modules/webhooks` follows the sibling conventions:
- an injected clock that never goes backward and one mutex, which is never held while a callback runs;
- the standard errors plus `ErrStale`;
- dual control for configuration, human actors for configuration and system actors for `Dispatch`, `Confirm`, `Redirect` and `Report`.

`modules/events` gains `Bounds(workspace, actor) (floor, head uint64, error)`. It is for system actors only. It sweeps before answering and returns (0, 0) for an unknown workspace without creating it, so approval and resume read the head directly. The change review covers that file too.

The constructor is `NewRegistry(Config{Outbox, Destinations, Permits, Signer, Redirects}, Limits{...}, clock)`. The limits and their ranges:

| Limit | Meaning | Range |
| --- | --- | --- |
| `Workspaces` | workspaces held | 1 to 100,000 |
| `Subscriptions` | per workspace, pending ones included | 1 to 10,000 |
| `Types` | types per subscription | 1 to 32 |
| `Lease` | lease length | 1 second to 1 hour |
| `Backoff` | first backoff step | 1 second up to `MaxBackoff` |
| `MaxBackoff` | largest backoff step | up to 24 hours |
| `Attempts` | attempts per delivery | 1 to 100 |
| `MaxAge` | age limit per delivery | `Backoff` to 30 days |
| `Dead` | dead letters per subscription | 1 to 10,000 |
| `Batch` | events read per page | 1 to 10,000 |
| `Work` | pages read per `Dispatch` | 1 to 10,000 |

`MaxAge` must stay well below the outbox `Retention` (a configuration rule stated to the owner; the outbox is a separate register).

## Callbacks

- **`Outbox`.** `Read(workspace, after, limit)` and `Bounds(workspace)`, served by `modules/events` with a relay actor.
- **`Destinations`.** `State(workspace, destination) (active|inactive|revoked, error)`, served by an adapter over `trust.DestinationOf`. A destination that is not found maps to `revoked`.
- **`Permits`.**
  - `Subscribe(workspace, subscriber, types) (bool, error)`, checked at approval and at resume. This answers QA-05's subscription recheck.
  - `Allowed(workspace, subscriber, event) (bool, error)`, checked at each send. It sees the whole event.
- **`Signer`.**
  - `Current(workspace, subscription) (version, error)`.
  - `Sign(workspace, subscription, version, message) ([]byte, error)`.

  Keys are per subscription, so no receiver can forge requests to another subscription. The signer contract asks for asymmetric keys whose public halves are published.
- **`Redirects`.** `Check(workspace, destination, current, hop, location) (destination, url string, hop int, error)`, served by `trust.CheckRedirect`.

## Subscriptions

`ProposeSubscription(workspace, Subscription{ID, Subscriber, Destination, Types}, actor)` by one human and `ApproveSubscription` by another activate a subscription. Pending proposals count toward `Subscriptions`. At approval, the destination must be active and `Subscribe` must allow it. The cursor starts at the outbox head, and a subscription is immutable once approved.

Lifecycle:
- **Pause.** `PauseSubscription` by a human holds deliveries without cancelling anything.
- **Resume.** `RequestResume` by one human and `ApproveResume` by another resume after a pause, a `gap` or a `backlog`, and they recheck `Subscribe`.
- **Gap.** `ApproveResume` after a gap calls `Bounds` and continues from the current floor, the last swept sequence, so reading resumes at the oldest retained event. The register records the lost range, from the cursor (exclusive) to that floor (inclusive), on the subscription, so the approver sees exactly what was lost and nothing more is skipped.
- **Revoke.** `RevokeSubscription` by a human ends the subscription at once.
- **Revoked destination.** A destination that reports `revoked` ends the subscription in the same way, with reason `destination_revoked`, because trust destinations never come back.

## Deliveries

Each subscription has the following state:
- a cursor;
- at most one open delivery, holding a snapshot of the event it delivers: the stored event and its sequence;
- a redelivery queue served before the cursor;
- a dead-letter list.

Lists:
- The redelivery queue and the dead-letter list together hold at most `Dead` entries.
- When they are full, the delivery that should move there stays open and the subscription is suspended with `backlog`.
- `DismissDeadLetter` by one human, approved by another, removes a dead letter.

Snapshots are kept only while a delivery is open or dead. Retention of these snapshots beyond the outbox retention is an owner decision (residual).

`Dispatch(workspace, limit, actor)`, with `limit` up to `Batch`:

- **Order.** It visits subscriptions round-robin, starting after the one visited last, where reading a page and leasing both count as a visit, and gives each at most one lease. A call stopped by the `Work` bound therefore resumes with the next subscription, and none is starved.
- **Skips.** It skips:
  - subscriptions that are paused or suspended;
  - those inside their throttle or backoff time;
  - those whose cursor equals the head. `Dispatch` calls `Outbox.Bounds` once per call, outside the mutex, and compares cursors with that fresh head, so events published after catching up are always picked up. If `Bounds` fails, `Dispatch` returns the error, leases nothing and changes no cursor.
- **Reading.** For a subscription without an open delivery, it takes the next queued redelivery. Otherwise it reads one page of the outbox after the cursor.
  - Events of other types advance the cursor.
  - The first matching event becomes the open delivery.
  - At most `Work` pages are read per call.
  - An `Expired` answer suspends the subscription with `gap`.
- **Result.** It leases due deliveries for `Lease` with a fencing token and returns identifiers only (delivery, event, destination).
- **Locking.** Reads happen outside the mutex, and the register rechecks that the subscription and cursor are unchanged before committing.

**Expiry of a lease without confirmation.** Nothing was signed, so nothing was sent. The delivery returns to due, takes one backoff step, and does not count as an attempt.

`Confirm(workspace, delivery, token, actor)` is called immediately before the network write and is one-shot per token.

1. **Mark.** Under the mutex it marks the lease `confirming`, with a deadline of `Lease` from now. A second `Confirm`, or a `Report` on that token, gets `ErrStale` while confirmation is in progress. When the deadline passes, the delivery returns to due with one backoff step and no counted attempt, and a `Confirm` that returns late answers `ErrStale` without a signature.
2. **Rechecks, outside the mutex.** The destination's state, `Allowed`, and signing with the current version for this subscription.
3. **Recheck under the mutex.** It requires the same token, the subscription still active and neither paused nor suspended, and the delivery still `confirming`. Otherwise the lease returns to due and the call answers `ErrStale` without returning a signature.
4. **Success.** The lease becomes `confirmed`, with a fresh lease from now, and the call returns the body (`events.Body`), the headers and the signature.

Recheck outcomes:
- **Permission refused.** The delivery is cancelled (`permission_revoked`) and the cursor advances.
- **Destination inactive.** The subscription is suspended (`destination_inactive`).
- **Destination revoked.** The subscription is ended.
- **Callback error.** The delivery returns to due and takes one backoff step without counting an attempt.

**Signature.** It covers the length-prefixed tag `hybrid-billing/webhook/v1`, workspace, subscription, delivery identifier, event source and identifier, attempt number, key version, attempt time in Unix nanoseconds, and the SHA-256 of the body. A known-answer test pins it, and a reference verifier checks that changing any field breaks it. Each new attempt is signed with the current version, so a key change between attempts re-signs.

**Receiver contract**, published with the plan:
- Verify against the published versions that overlap.
- Accept attempt times within five minutes.
- Check that the signed subscription is the receiver's own.
- Deduplicate by event source and identifier.
- Answer a repeat of an already processed event with 2xx and no reprocessing.

`Redirect(workspace, delivery, token, current, hop, location, actor)` serves a confirmed lease. It calls `Redirects.Check` and allows only a result whose destination is the subscription's own destination (`unapproved_redirect`). Senders must use it, or refuse redirects. A refused redirect or any other 3xx is reported as `rejected`.

`Report(workspace, delivery, token, Outcome{Status, RetryAfter}, actor)` applies only to a confirmed lease. On other tokens it returns `ErrStale`.
- **`delivered`.** Advances the cursor, or removes the redelivery.
- **`rejected`.** Covers 4xx other than 408 and 429, 3xx, and refused redirects. The delivery becomes a dead letter with its reason.
- **`throttled` (429).** Sets the subscription's throttle time to `RetryAfter`, raised to at least `Backoff` and clamped to `MaxBackoff`. Without a positive `RetryAfter`, it uses the backoff. It does not count as an attempt, but it advances the backoff step.
- **`failed` and `unknown`.** Count as attempts. Backoff is exponential from `Backoff`, capped at `MaxBackoff`, minus a deterministic jitter of up to 20% derived from the delivery identifier and attempt number, so the result never exceeds `MaxBackoff`.

A confirmed lease that expires is booked as `unknown` when a later call observes the expiry. Booking invalidates the token, so a report that arrives late, after booking, gets `ErrStale` and cannot count twice. A report that arrives before booking is applied normally. If booking uses up the budget, a delivery that actually arrived can become a dead letter. Under at-least-once delivery this is accepted: redelivery is safe because receivers deduplicate.

A delivery reaches its dead letter when any of these happens:
- it has used up `Attempts`;
- it has been open for longer than `MaxAge`, in every state (throttled, confirming, paused or suspended). When the lists are full, the full-list rule wins: the delivery stays open and the subscription stays suspended with `backlog`;
- it was rejected.

`RequestRedeliver` by one human and `ApproveRedeliver` by another queue a dead letter again. It runs as a new delivery, with a new delivery identifier and a new budget.

## States and listing

Each delivery reports a QA-05 job state and a reason. Deliveries stay apart from `modules/jobs` because a delivery is a stream position rather than a task manifest, but they use the same state names.

| QA-05 state | Delivery status |
| --- | --- |
| `queued` | due |
| `running` | leased, confirming or confirmed |
| `waiting` | backoff, throttle, or held by a pause or suspension |
| `succeeded` | delivered |
| `failed` | dead-lettered |
| `cancelled` | permission revoked, or subscription or destination revoked; carries `outcome_unknown` when a confirmed attempt may already have been sent |

`Deliveries(workspace, subscription, after, limit, actor)` pages, for human or system actors:
- the open delivery;
- redeliveries;
- dead letters;
- the most recent cancellations, bounded by `Dead`.

## Deviation from HB-239

HB-239 expected per-consumer low watermarks on the outbox sweep. This plan chooses explicit gap suspension with a recorded lost range instead. Pinning would let a stuck receiver block publication, which H2 forbids. The HB-239 plan is annotated accordingly.

Key publication and rotation for receivers (M6) goes to a follow-up ticket, opened when this ticket closes.

# Verification plan

Literal tests:
- **Subscriptions.** Dual control, pending proposals that count, approval at the head, destination and subscribe checks, pause, dual-controlled resume with its subscribe recheck, revocation, and the end of a subscription whose destination was revoked.
- **Dispatch and gaps.**
  - Type filtering.
  - Gap suspension, and resume from the floor with the lost range recorded, including events published while suspended.
  - Round-robin fairness, the `Work` bound, skipping subscriptions that are at the head, and one open delivery per subscription.
  - An expired lease that was never confirmed returns to due without counting as an attempt.
- **Regressions from round 2.**
  - Catch up, publish, then expect a lease.
  - A signer that hangs past the confirming deadline, then a late `Confirm` that gets `ErrStale`.
  - `Work = 1` with one subscription whose types never match, without starving the others.
  - `MaxAge` reached while the lists are full.
- **Confirmation.**
  - Each recheck outcome and callback error.
  - Interleavings during callbacks: pause, revoke, a report, a second `Confirm`, a lease expiry and a concurrent dispatch.
  - A slow signer that does not block another workspace.
  - The signature as a known answer, with a reference verifier that rejects any changed field.
  - A key change between attempts.
- **Redirects.** Redirects to the own destination and to another one.
- **Reports.**
  - Each status, and stale tokens.
  - A 429 on the subscription: `RetryAfter` zero, negative, normal and above `MaxBackoff`.
  - Throttles that never stop, ended by `MaxAge`.
  - Backoff and jitter bounds, and the attempt budget.
  - Late reports before and after an expiry is booked.
- **Dead letters.** The dead-letter and queue bound with backlog suspension, redelivery and dismissal under dual control, and the new delivery identifier on redelivery.
- **States and listing.** The state mapping and pages, including cancellations with `outcome_unknown`.

Property test: random publications, dispatches, confirmations, reports, throttles, callback errors, pauses, resumes, gaps and redeliveries over several subscriptions, against an independent model. It checks two things:
- Every matching event outside recorded gaps is delivered, dead-lettered or cancelled exactly once, in order per subscription; redeliveries are excluded from the ordering check.
- Publication is never refused because of deliveries.

Checks:
- `-race -count=10`, a 386 build, gofmt and vet;
- the whole repository with `-race`;
- 100% coverage;
- mutation testing.

# Review history
- Design review round 1: FAIL (2 high, 11 medium, 10 low). Folded in:
  - resume after a gap from the floor, with the recorded lost range (F1);
  - `MaxAge`, revoked destinations as terminal, and the retention configuration rule (F2);
  - one-shot `Confirm` with a `confirming` mark and a full recheck under the mutex (F3);
  - unconfirmed expiries that do not count, and reports only on confirmed leases (F4);
  - expiry booking that invalidates the token (F5);
  - deduplication by source and identifier, with the source signed (F6);
  - the receiver contract for repeats and subscription checks, and new delivery identifiers on redelivery (F7);
  - keys per subscription, asymmetric (F8);
  - page, work and head bounds per dispatch (F9);
  - callback errors that advance the backoff (F10);
  - a subscription permission check at approval and resume (F11);
  - a `Redirect` helper and a follow-up ticket for keys (F12);
  - dual-controlled dismissal of dead letters (F13);
  - a redelivery queue, snapshots, the `RetryAfter` clamp, `Bounds`, destination states, the recorded deviation, limit ranges, system actors, 3xx mapping, jitter direction, listed cancellations, and identifiers only at dispatch (low items).
- Design review round 2: FAIL (1 high, 2 medium, 5 low). Folded in:
  - a fresh head from `Bounds` on every dispatch (R1);
  - a deadline for `confirming` and `MaxAge` in every state (R2);
  - round-robin advancing past every visit (R3);
  - gap resumes reading the current floor (R4);
  - a minimum `RetryAfter` (R5);
  - full lists keeping the delivery open (R6);
  - `Bounds` and adapter edge cases (R7);
  - the stated at-least-once consequence of booked expiries (R8).
- Design review round 3: PASS (3 low). Folded in: full lists win over `MaxAge` (N1); a failing `Bounds` leaves everything unchanged (N2); named regression tests (N3).

# Change review resolution

- Change review round 1: FAIL (1 high, 3 medium, 7 low). Resolved:
  - H1: `ApproveResume` requires a paused or suspended subscription before and after its callbacks, and revocation clears the resume request, so a revoked subscription can never come back. A regression test covers it.
  - M1: a subscription that needs a read when the `Work` bound is used up stops the visit without being marked visited, so the next call starts with it. A regression test with continuous traffic of other types covers it.
  - M2: revoked subscriptions leave the dispatch order and do not count toward `Subscriptions`. Revoking a proposal withdraws it, and revoked subscriptions without dead letters are purged once the map holds twice the bound.
  - M3:
    - a reference verifier rejects every changed header and a changed body, and the signature is pinned as a fixed hex value;
    - tests cover `MaxAge` with full lists, `outcome_unknown` after a confirmed attempt, and concurrent dispatches;
    - the property test adds pauses, resumes, permit outages and recorded gaps, and publication must never be refused.
  - L1: request maps are cleared together when a letter leaves the dead list.
  - L2: gap history is bounded by `Dead`.
  - L3: a delivery held by a full list keeps its pending reason; a dismissal frees room.
  - L5, L6: `Confirm` checks destination and permission before signing, takes the attempt time from the clamped clock, and orders outcomes so a revoked destination wins over later callback errors.
  - L7: `Dispatch` maps an outbox failure to `dependency_unavailable`.
- Residuals:
  - L4: a due delivery after a backoff still reports `waiting` until it is leased (the state carries no clock).
  - L7: index paging of `Deliveries` can skip or repeat entries when lists change between calls; `SubscriptionOf` takes no actor, like sibling reads.
- Change review round 2: FAIL (1 medium, 3 low, 1 informational). Resolved:
  - N1: after purging, a proposal is refused with `ErrLimit` while the map still holds twice the bound, and redelivery on a revoked subscription is refused, so revoked subscriptions with letters cannot grow the map. A regression test covers it.
  - N3: the attempt time is read again just before signing.
  - N4: when a dismissal frees room, a delivery held for a terminal reason (rejected, maximum age, attempts) moves straight to the dead letters and is not sent again.
- Residuals:
  - N2: purging frees revoked subscription identifiers, so identifiers must be unique per workspace; signer keys are per identifier.
  - N5: removing a revoked subscription from the dispatch order can skip one subscription for one round.
- Change review round 3: PASS (2 low), both recorded as residuals.
  - L-A: a delivery held for a terminal reason keeps that reason if the subscription is resumed without a dismissal and the delivery is sent again. A dismissal approved while it is in flight then buries it under the old reason. This stays within at-least-once delivery, because redelivery is safe, but the recorded reason can be wrong. A follow-up should bury held terminal deliveries at dispatch and check the phase before burying.
  - L-B: assertions for the `max_age` and `attempts_exhausted` branches of that path.
