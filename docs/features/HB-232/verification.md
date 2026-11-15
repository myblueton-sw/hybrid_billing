---
type: VerificationRecord
title: Delivery attempts and channel gating verification
description: Records what was executed and reviewed for the second increment of work item W-29, a delivery register over fixed disclosure packages with recipient-bound channels, DB-05 capability gating, per-delivery approval, pair deduplication, resend with second approval, leased attempt tokens, late outcomes that are never lost and per-pair possible-duplicate tracking. The record lists results, the mutation check, provisional choices and what remains unverified.
status: draft
---

# Delivered behavior

[HB-232](https://linear.app/hybrid-billing/issue/HB-232) adds the Go package `modules/delivery` from main `2b77269`, and additive read-only `disclosure.Registry.Validity` and `Recipient`.

- **Channels.** Each channel is bound to one recipient and never changes. A diagnosis needs channel-test evidence (`platform_export` is refused) and closes automatic delivery; a second human opens it for the latest diagnosis only when field and attachment control are proven and nothing is attached automatically. ERP attachment channels are never opened here; they go through approved manual deliveries until the HB-227 gate is bound.
- **Deliveries.** A delivery targets one live package whose recipient is the channel's; the package, recipient, file digest, policy and grant revisions are recorded. One delivery per package and channel unless a human cancels it; a second sending uses `Resend`, which always needs a second human. Deliveries are attempted automatically only while their channel is open, otherwise after approval.
- **Attempts.** Tokens come from an injected random source; attempts are leased, superseded after the lease, re-check the package (a lapsed package stops the delivery, an unreachable register refuses without stopping) and are bounded (`exhausted`). Every report for an attempt that was started is recorded, including late ones after supersession, cancellation or stop; an acceptance is never lost and takes the pair back when it is free. Unknown outcomes, supersession, cancelling in flight and acceptance while another attempt or delivery holds the pair mark the pair, and every later attempt carries the possible-duplicate flag. No state claims exactly-once delivery.
- **Actors.** Assistants and recipient identifiers are refused on every operation; channel and approval operations need humans.

# Commands and results

All runs were made in the worktree `/private/tmp/hb232` on 2026-11-15 with Go 1.25.7 on darwin/arm64.

| Check | Result |
| --- | --- |
| `gofmt -l modules/` | no output |
| `go vet ./modules/delivery ./modules/disclosure` | pass |
| `go test -race -count=10 ./modules/delivery ./modules/disclosure` | pass |
| `GOOS=linux GOARCH=386 go test -c ./modules/delivery` | builds |
| `go test -race -count=1 ./...` | 45 packages, all pass |
| Coverage | `modules/delivery` 100.0%, `modules/disclosure` 100.0% |
| Mutation testing | 591 mutants of `delivery/registry.go`, `deliveries.go`, `attempts.go` and `disclosure/project.go`: 546 killed, 38 did not build, 7 survived |

The seven survivors were read: two are equivalent (`Round(1)` on the clock, a `[][3]` limits array); three change the acceptance marking condition in `Report`, which is redundant because every attempt that can still be reported late was superseded or cancelled in flight, and both already mark the pair; two are in pre-existing `modules/disclosure/project.go` code outside this change (a slice capacity and a sort comparator over unique rows).

Tests include an independent model of every delivery's state, attempts, approval, pair holder and pair uncertainty over two pairs and 60 seeds of 300 random steps, compared with every view after each step and requiring each key path to be exercised; scenario tests for automatic delivery, retries, late acceptance after cancel, a late report after a stop, closed channels, ERP, recipient binding, AUD-DB-05 on a portal channel and resend; and an integration test with real `modules/disclosure` packages covering policy supersession and grant revocation.

# Reviews

- Design review: three rounds by a separate read-only reviewer; FAIL, FAIL, PASS ([plan](plan.md)).
- Change review: two rounds by a fresh read-only reviewer; FAIL (late acceptance after cancel freed the pair; per-pair duplicate tracking; model independence), PASS.

# Provisional choices for the owner

- Lease 1 minute to 24 hours, attempts per delivery up to 100, bounded queue and deliveries per workspace.
- Identifiers that are recipients in any workspace are refused on every delivery operation.
- Channel-test evidence is declared by reference and not verified by this register.

# Not verified and residuals

Real mail, portal and ERP transports and receipts, transport access to package bytes, binding ERP channels to the HB-227 gate, verifying channel-test evidence, backoff schedules, notification policy (U03), persistence and screens. Delivery never touches issuance, so issue dates and due dates are unaffected.
