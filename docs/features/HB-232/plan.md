---
type: ImplementationPlan
title: Delivery attempts over fixed disclosure packages with channel capability gating, per-delivery approval, deduplication, re-checked packages and honest outcome records
description: Defines the second increment of work item W-29, a delivery register whose channels open automatic delivery only after a channel-specific capability diagnosis proves recipient field and attachment control without automatically added attachments and a second human confirms it; other deliveries need a per-delivery approval. Each delivery targets one fixed disclosure package on one channel, is deduplicated by package and channel, never re-issues documents, re-checks the package before every attempt, and records accepted, rejected and unknown outcomes with possible-duplicate flags instead of claiming exactly-once delivery.
status: draft
---

# Scope and ownership

[HB-232](https://linear.app/hybrid-billing/issue/HB-232) continues work item W-29 (requirement HB-29; DB-05; HB-29-I and HB-29-V delivery parts) after HB-231 (`modules/corrections`). Base: main `2b77269`. Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviews the plan and the change, because the register decides what reaches customers and partners. Machine: mac_mini.

Out of scope, recorded as residuals: real mail, portal and ERP transports and their receipts, backoff schedules, binding the HB-227 ERP delivery gate (`modules/erpadapters`) to these channels, delivery of `modules/issuance` files outside disclosure packages, notification policy (U03), persistence and screens.

# Observed baseline

- HB-29-D: issuing successfully and delivery failing are different states; per-channel lookup and duplicate guarantees.
- HB-29-I: a DeliveryAttempt targets a fixed package, recipient and channel; an issued bill is never issued again.
- HB-29-V: delivery failure retries and policy changes are tested; exactly-once delivery is not claimed without a channel guarantee; mass delivery is controlled with a bounded queue and duplicate records.
- DB-05 and AUD-DB-05: ERP automatic attachment, mail and portal publication are diagnosed as connection capabilities; without proof of per-recipient field and attachment control, automatic delivery stays off and only approved paths are allowed; platform export verification alone never passes ERP sending; an ERP template that adds an internal cost file blocks opening automatic delivery regardless of issue or posting state.
- `modules/disclosure` holds packages bound to a recipient grant and policy revision and refuses access after supersession, suspension, grant change or revocation; it has no read-only validity call.

# Decision

A new package `modules/delivery` follows the sibling conventions: injected clock, one mutex, workspace isolation, `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (`Refusal{Reason}`), `ErrLimit`; check order form, lookup, repeat, rule, room; assistants refused everywhere. `NewRegistry(Config{Packages, Random}, Limits{Workspaces, Channels, Deliveries, Queue, Attempts, Lease}, clock)`:

- `Packages` has the signatures of two new additive read-only calls in `modules/disclosure`: `Validity(workspace, package)` returns, from that workspace only (`ErrNotFound` otherwise), the package's recipient, file digest, policy and revision, grant revision and whether `Access` would still serve it (the same `live` check, with the refusal reason when not); `Recipient(id)` reports whether an identifier was ever proposed as a recipient.
- `Random` is the source of attempt tokens (`crypto/rand` in production); a read failure is `random_unavailable`.

Every operation that takes an actor refuses recipient identifiers (`recipient_actor`), as disclosure does for staff operations, so a recipient can neither approve nor report outcomes or receipts for a delivery. Allowed files: `modules/delivery/**`, the additive `Validity` and `Recipient` with tests in `modules/disclosure`, `docs/features/HB-232/**`. The register never calls issuance and changes no issue date, due date or other payment-term anchor; delivery and resend leave them untouched (payment-terms.md).

## Channels and capability gating

`Register(workspace, Channel{ID, Kind, Recipient, Target}, actor)` by a human records a channel of kind `erp_attachment`, `mail` or `portal` for exactly one recipient with an opaque target reference; a channel never changes, so a new target or recipient is a new channel. `Diagnose(workspace, channel, Diagnosis{FieldControl, AttachmentControl, AddsAttachments, Evidence{Kind, Reference}}, actor)` by a human records a new diagnosis revision for that channel and recipient and closes automatic delivery. The evidence kind must be `channel_test`; `platform_export` and any other kind are refused (`export_evidence_insufficient`), so a platform export check cannot stand in for a channel test. The reference is declared, not verified, by this register (residual). `Open(workspace, channel, revision, actor)` checks, in this order, the ERP refusal below, the latest revision, the diagnoser and the capabilities; by a human other than that revision's diagnoser it opens automatic delivery only for the latest revision (`stale_diagnosis`), only when field and attachment control are both proven (`capability_unproven`) and no attachment is added automatically (`adds_attachments`). `erp_attachment` channels are never opened here (`erp_gate_unbound`): their automatic delivery belongs to the HB-227 gate in `modules/erpadapters`, and binding the two is a residual, so ERP attachments go only through approved manual deliveries in this increment. `Close(workspace, channel, actor)` by a human closes automatic delivery.

## Deliveries

`Request(workspace, Delivery{ID, Package, Channel}, actor)` by a human or system. The package must be live (`package_not_live`) and its recipient must be the channel's (`recipient_mismatch`); the delivery records the package, recipient, file digest, policy revision and grant revision. Deduplication: a request for a package and channel that already have a delivery that is not cancelled (waiting, in flight, accepted, confirmed, stopped or exhausted) returns that delivery (`Duplicate` on the returned view only); only after a human `Cancel` may a new `Request` create another delivery for the pair, and it inherits the pair's possible-duplicate flag. Any other second sending uses `Resend`. Deliveries waiting or in flight are bounded per workspace (`Limits.Queue`, `ErrLimit`).

The mode is decided at every `Begin`: a delivery may be attempted automatically only while its channel is open; otherwise it needs `Approve(workspace, id, actor)` by a human other than the requester, which re-checks the package and recipient match. So a delivery whose channel closes is never stuck: it waits for approval. `Cancel(workspace, id, actor)` by a human ends a delivery that is not accepted or confirmed.

`Resend(workspace, id, Delivery{ID}, reason, actor)` by a human creates a new delivery linked to an earlier accepted, confirmed, cancelled, stopped or exhausted one for the same package and channel; it always needs approval by a second human, whatever the channel, and is flagged as a possible duplicate.

Possible duplicates are tracked per package and channel across deliveries: once any attempt for that pair ended `unknown`, was superseded, or was accepted late while another attempt was running, every later delivery and attempt for the pair carries `PossibleDuplicate`.

## Attempts and outcomes

`Begin(workspace, id, actor)` by a human or system starts an attempt and returns a token, the package, recipient, file digest, channel, attempt number and the `PossibleDuplicate` flag; the bytes themselves are fetched by the transport through disclosure (residual). It requires a waiting delivery, approval unless the channel is open now, a channel recipient equal to the package recipient, no attempt younger than the lease (`attempt_running`) and a live package; a package that is no longer live stops the delivery (`stopped`, with the reason) and nothing is sent. A running attempt older than the lease is superseded (outcome `unknown`, possible duplicate) before anything else is checked. When the attempt limit is reached, the delivery becomes `exhausted` (visible, cancellable, resendable) and `Begin` returns `ErrLimit`. Package file digests never change in disclosure, so no digest-change branch exists.

`Report(workspace, id, token, Outcome{Result, Reference}, actor)` by a human or system records `accepted` (the channel took it; not proof of receipt), `rejected` (waits for a retry) or `unknown` (waits for a retry and marks the pair as possible duplicate). Reports are accepted for every attempt that was ever started, including superseded attempts and attempts of cancelled, stopped or exhausted deliveries; such reports are marked late. A late or current `accepted` always sets the delivery to `accepted` (never removing any record), and marks the pair possible duplicate when another attempt was started after it. A token that was never issued is `ErrNotFound`; the same report again returns the view; a different report for the same attempt is `ErrConflict`. `Confirm(workspace, id, receipt, actor)` records a channel receipt on an accepted delivery. No state claims exactly-once delivery.

# Residuals and coverage

DB-07 is delivered by HB-228 (`modules/exports`). Payment-term anchors (PT-01 to PT-08) are untouched by delivery; their receivables effects remain residuals of earlier increments. AUD-DB-05 for ERP channels is satisfied here only because ERP channels are never opened; the real ERP evidence is the HB-227 gate. The `adds_attachments` refusal is tested on mail and portal channels. Residuals: real transports and receipts, transport access to package bytes, verifying channel-test evidence, binding `erp_attachment` channels to the HB-227 gate, backoff schedules, notification policy, persistence and screens.

# Verification plan

Literal tests for every refusal and repeat; a package for one recipient refused on another recipient's channel; platform export evidence refused; ERP channels never opened; late reports after supersession, cancellation and stop; resend and inherited possible-duplicate flags; a closed channel turning delivery manual; AUD-DB-05 as the `adds_attachments` refusal on mail and portal channels with both controls proven, plus a test that ERP channels are never opened; a policy change and a grant revocation between attempts stopping delivery; retries after rejected and unknown outcomes with possible-duplicate flags; lease supersession; queue and attempt limits; deduplication; a property test of random operations against an independent model of delivery states; integration with real `modules/disclosure` packages; `-race -count=10`, a 386 build, gofmt, vet, the whole repository with `-race`, 100% coverage and mutation testing.

# Review history

- Design review round 1: FAIL (3 high, 4 medium, 5 low). Folded in: channels bound to one recipient and immutable (1); late reports always recorded and late acceptance kept (2); ERP channels manual-only here, HB-227 gate binding a residual (3); typed channel-test evidence (4); `Resend` with second approval and per-pair possible-duplicate tracking (5); mode decided at each `Begin` (6); recipient actors refused on staff operations (7); no digest-change branch (8); `Validity` scope and bound revisions, approval re-check (9); `exhausted` state (10); `Random` in config (11); byte transport a residual (12); DB-07 and PT coverage stated.
- Design review round 2: FAIL (1 medium, 2 low). Folded in: deduplication covers stopped and exhausted deliveries, new requests only after a human cancel (A); recipient actors refused on every operation (B); `Open` check order and the AUD-DB-05 test on non-ERP channels (C).
- Design review round 3: PASS; the verification wording for AUD-DB-05 was aligned with the residual note.

# Change review resolution

Change review round 1: FAIL (1 high, 2 medium, 5 low). Resolved:

- A late `accepted` on a delivery whose pair is free takes the pair back, so a new `Request` returns it as the existing delivery (1).
- Possible duplicates are tracked per pair: cancelling a delivery with an attempt in flight, a superseded attempt, an `unknown` outcome, or an acceptance while a later attempt or another delivery holds the pair marks the pair, and marking also flags the delivery that currently holds the pair; a resend's own attempts are flagged without marking the pair, as the plan states (2, 7).
- The property test now keeps an independent model of every delivery's state, attempts, approval, the pair holder and pair uncertainty over two pairs, and compares every view after each step (3).
- Added tests: late acceptance after cancel, a late report after a stop, grant revocation through real disclosure packages, token collision (4).
- An unreachable or failing package register is `packages_unavailable` and leaves the delivery waiting; only a package that is no longer live stops it (5).
- Confirmations record the confirmer and repeats compare it; a colliding token is refused as `random_unavailable` (6).
- Test hygiene fixed (8).

Change review round 2: PASS. Remaining non-blocking notes: the per-step model check does not compare possible-duplicate or late flags (they are checked at request and begin and in the scenario tests), and packages never lapse during random operations (stops are covered by scenario and real-disclosure tests).
