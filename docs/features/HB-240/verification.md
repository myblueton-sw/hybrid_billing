---
type: VerificationRecord
title: Webhook subscriptions and deliveries verification
description: Records what was executed and reviewed for HB-240, the third increment of work item W-34. It covers dual-controlled webhook subscriptions, deliveries as per-subscription cursors over the event outbox with explicit gap suspension, fair and fenced dispatch, a one-shot confirmation that rechecks destination and permission before signing each attempt, throttling, backoff with jitter, budgets, an age limit, and dead letters with dual-controlled redelivery and dismissal. The record lists results, the mutation check and what remains unverified.
status: draft
---

# Delivered behavior

[HB-240](https://linear.app/hybrid-billing/issue/HB-240) adds the Go package `modules/webhooks` from main `fe2fec4`, and `events.Registry.Bounds`.

## Subscriptions

- Approval is dual-controlled and checks the destination and the subscription permission.
- The cursor starts at the outbox head.
- Pause holds deliveries without cancelling them. Resume is dual-controlled and rechecks both the state and the subscription permission.
- Resuming after a gap continues from the current floor and records the lost range.
- Revocation is terminal: it clears pending resumes and removes the subscription from the dispatch order.
- Only subscriptions that are not revoked count toward the limit. Revoked ones without letters are purged once the map reaches twice the limit, and proposals are refused while it stays there.

## Dispatch

- Each call reads a fresh head.
- Subscriptions are visited round-robin. A subscription that needed a read and was stopped by the `Work` bound is visited first on the next call.
- Each subscription has at most one open delivery, and outbox pages are read outside the mutex with generation and cursor rechecks.
- Events of other types advance the cursor.
- An expired cursor suspends the subscription as a gap.
- A lease that expires without confirmation returns the delivery to due without counting an attempt.

## Confirmation

- Confirmation is one-shot and has a deadline.
- It checks the destination state first, then the permission, then the signer version, and only then signs. All of these run outside the mutex.
- It takes the attempt time again just before signing.
- Under the mutex it rechecks the token, the phase, the deadline and the subscription state.
- The signature covers the subscription, the delivery, the event source and identifier, the attempt, the key version, the time and the body digest. A reference verifier and a pinned hex value pin it.
- Redirects are allowed only to the subscription's own destination.

## Reports

- A report applies only to a confirmed lease.
- `delivered` ends the delivery.
- `rejected`, including 3xx responses, moves it to the dead letters.
- `throttled` sets a throttle on the whole subscription. `RetryAfter` is clamped between `Backoff` and `MaxBackoff`, and the backoff step advances without counting an attempt.
- `failed` and `unknown` count as attempts. Backoff is exponential with up to 20% jitter subtracted, and capped.
- Booking an expiry invalidates the token.
- A delivery is dead-lettered after `Attempts` attempts or after `MaxAge` in any state.

## Dead letters and states

- Full lists hold the delivery open with a backlog suspension. A dismissal buries held deliveries that ended for a terminal reason.
- Redelivery is dual-controlled and creates a new delivery identifier. It is refused on revoked subscriptions.
- Delivery states use the QA-05 state names.
- Listings show open deliveries, redeliveries, dead letters and recent cancellations, marked `outcome_unknown` where an attempt was already confirmed.

# Commands and results

All runs were made in the worktree `/private/tmp/hb240` on 2026-11-15, with Go 1.25.7 on darwin/arm64.

| Check | Result |
| --- | --- |
| `gofmt -l modules/` | no output |
| `go vet ./modules/webhooks ./modules/events` | pass |
| `go test -race -count=10 ./modules/webhooks ./modules/events` | pass (a race in a test fake's read counter was fixed first) |
| `GOOS=linux GOARCH=386 go test -c ./modules/webhooks` | builds |
| `go test -race -count=1 ./...` | 52 packages, all pass |
| Coverage | `modules/webhooks` 100.0%, `modules/events` 100.0% |
| Mutation testing | 437 mutants of `subscriptions.go` and `deliveries.go`: 376 killed, 19 did not build, 42 survived |

The 42 survivors were read and judged equivalent:

- **Generation increments.** Dropped increments, where every guarded path also rechecks the subscription state or cursor.
- **Redundant conditions.** Altered combinations in rechecks whose other terms already decide the outcome.
- **Phase labels.** Phase labels on dead, new or delivered deliveries that are never read.
- **Backoff boundaries.** Backoff loop and cap boundaries, and jitter constants, within the tested range.
- **Ignored values.** Ignored return values, the identifier number base, `Round(1)`, and a `[][3]` limits array.
- **Redundant state checks.** Queued-redelivery state checks that the default already covers.
- **Equal results.** Gap-range boundaries and a dismissal-reason disjunct whose outcome is the same.

The property test runs 60 seeds of 150 steps. Each run covers publications, dispatches, unconfirmed leases, confirmations and reports of every status, permit outages, and pauses and resumes, followed by a drain. It checks two things:
- every matching event outside recorded gaps ends exactly once and in order for each subscription;
- publication is never refused.

Literal tests cover the regressions found in each review round. Removing the key mutex was not part of this package.

# Reviews

- **Design review:** three rounds by a separate read-only reviewer. Rounds 1 and 2 failed (gap loss, holds without limit, confirmation races, starvation); round 3 passed ([plan](plan.md)).
- **Change review:** three rounds by a fresh read-only reviewer. Round 1 failed (revival of a revoked subscription, starvation under `Work`, growth, tests); round 2 failed (map growth); round 3 passed.

# Provisional choices for the owner

- The jitter fraction (20%) and the receiver tolerance window (five minutes).
- The shared bound for dead letters and redeliveries.
- Purging at twice the subscription bound.
- Gap history bounded by `Dead`.
- Throttles that advance the backoff step without counting an attempt.
- Keeping event snapshots while deliveries are open or dead.

# Not verified and residuals

- **Follow-up ticket.** Key publication and rotation for receivers (M6).
- **Owner decisions and contracts.**
  - Retention of snapshots beyond the outbox retention.
  - Unique subscription identifiers after a purge (signer keys are per identifier).
- **Sender and production wiring.**
  - The HTTP sender must use `Redirect` or refuse redirects.
  - Production adapters for the outbox, trust destinations, permits and signer have no consumer yet.
- **Known inaccuracies, low severity.**
  - A held terminal delivery can keep a stale reason after a resume without dismissal (L-A).
  - A due delivery after backoff reports `waiting` until it is leased.
  - Index paging can skip or repeat entries when lists change.
  - Removing a revoked subscription from the order can skip another one for one round.
- **Not yet built.** Persistence and screens.
