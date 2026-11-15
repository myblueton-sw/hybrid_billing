---
type: ImplementationPlan
title: Published webhook verification keys per subscription with dual-controlled rotation, and cleanup of deliveries held for terminal reasons
description: Defines a follow-up increment of work item W-34. Each webhook subscription gets a registry of published Ed25519 verification keys, with versions, validity windows and dual-controlled publication and retirement. Receivers can fetch the published keys, and Confirm refuses to sign with a version that is not published and valid at the attempt time. The increment also moves deliveries held for a terminal reason to the dead letters at dispatch, without sending them again, and checks the phase before a dismissal buries a held delivery.
status: draft
---

# Scope and ownership

[HB-241](https://linear.app/hybrid-billing/issue/HB-241) follows HB-240 (`modules/webhooks`). It covers the residual M6 (receiver key publication and rotation, QA-05 key rotation) and the residuals L-A and L-B. Base: main `a0e68cd`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the plan and the change, because the change decides which keys receivers trust. Machine: mac_mini.

Allowed files: `modules/webhooks/*`, `docs/features/HB-241/*`.

Out of scope (residuals):

- the signer's private key custody;
- an HTTP endpoint that serves the keys;
- persistence.

# Observed baseline

- HB-240 signs each attempt with `Signer.Current(workspace, subscription)` and `Signer.Sign`. The register never sees the public keys. Receivers are told to verify against the "published versions that overlap", but nothing in the installation publishes them.
- `modules/trust` (HB-237) already has the pattern needed here, for inbound keys: version, public key, validity window, weak-key refusal and dual control.
- Residual L-A: a delivery that is held because the lists are full, for a terminal reason (rejected, max age, attempts used up), is sent again after a resume without a dismissal. A dismissal can also bury such a delivery while it is in flight.

# Decision

## Keys

`ProposeKey(workspace, subscription, Key{Version, Public, NotBefore, NotAfter}, actor)` by one human and `ApproveKey` by another publish an Ed25519 public key for one subscription that is not revoked.

**Form.**
- The version is an identifier.
- The key is 32 bytes, canonical, and not of small order (the rule copied from `modules/trust`).
- `NotBefore` is non-zero and before `NotAfter`.
- The window is at most `KeyLife`, and both ends are inclusive.

**Approval challenge.** Approval first asks the signer, outside the mutex, to sign a fixed challenge for that version: the length-prefixed tag `hybrid-billing/webhook-key/v1`, workspace, subscription and version. Approval is refused (`key_mismatch`) unless the challenge verifies against the proposed public key, so a published key is proven to be held by the signer. After the challenge, approval re-locks and requires that the key is still proposed and the subscription is not revoked. The `Keys` limit is also enforced there, under the mutex. Otherwise approval answers `ErrStale`, so a withdrawal, a revocation or a concurrent approval during the challenge can never publish a key.

**Names, limits and lifetime.**
- Approved keys are immutable.
- Version names of published keys are never reused on a subscription. Every name that was ever approved is kept, up to `Keys × 64` names per subscription; beyond that, proposals are refused with `ErrLimit`. A proposal withdrawn before approval releases its name, because it was never published.
- Only approved keys that are not retired and not yet expired count toward `Keys`, the limit of overlapping published keys (1 to 16). Outstanding proposals are bounded separately, at `Keys` per subscription.
- `RetireKey` by one human ends a version at once, and also withdraws a key that is only proposed.
- The key registry belongs to the subscription record, so purging a revoked subscription deletes its keys. Identifiers must stay unique: the signer keys by identifier, so reusing an identifier after a purge is an owner-contract residual (HB-240 N2).

**Reading.** `Keys(workspace, subscription, actor)` is for human or system actors. It lists the approved keys that are not retired, with their windows, including future windows so receivers can prepare for a rotation. It lists nothing for a revoked subscription.

`Limits` gains `Keys` (1 to 16) and `KeyLife` (1 hour to 3650 days).

## Signing check

`Confirm` asks `Signer.Current` for the version and signs at the final attempt time. After it takes the mutex again, it checks three things:

1. The version is approved for the subscription.
2. The version is not retired.
3. The window contains the final attempt time.

If any check fails, the delivery returns to due with no attempt counted. The subscription is suspended with `key_unpublished`, and a human publishes a key and resumes the subscription, so the cause is never hidden behind `max_age`.

Under the same mutex, the signature is verified with `ed25519.Verify` against the published key. A mismatch is refused with `key_mismatch` in the same way. The signature is never released.

Every reason that `check` returns is matched explicitly in `Confirm`, and a default case refuses any other non-empty reason. A refused attempt therefore never returns a signature.

**Contract.**
- Retirement stops new signatures.
- Confirmed messages already handed out keep their signature. The delivery stores the version it was signed with. `Redirect` checks that version again under the mutex after the trust check. When it was retired, `Redirect` refuses and returns the delivery to due without counting an attempt, so the sender starts a new attempt.
- Resuming a subscription suspended with `key_unpublished` before a key is published suspends it again at the next confirmation.
- Receivers must refresh `Keys`, reject retired versions, and check `webhook-time` against the key's window within their tolerance.

## Held deliveries

A delivery that `bury` keeps open because the lists are full is marked `held`, together with its terminal reason.

- **Dispatch.** `Dispatch` never leases a held delivery. When the subscription is active again, it calls `bury` once more: the delivery moves to the dead letters when there is room, and otherwise the subscription is suspended with `backlog` again.
- **Expiry.** `expire` skips held deliveries, so their reason is never overwritten by `max_age`.
- **Dismissal.** `ApproveDismiss` buries a held delivery after it frees room. Only held deliveries are ever buried there, and they are never in flight.
- **Leased deliveries.** A leased delivery is never held and carries no reason.

# Verification plan

Literal tests:
- **Key form and publication.**
  - Every key form rule, including weak keys, the window bounds at both ends and their exact boundaries.
  - The approval challenge with a matching and a mismatching signer.
  - Dual control; refusal of assistant and system actors.
  - Repeats and conflicts; version names refused after retirement and accepted again after withdrawal.
  - The bound on names.
  - Withdrawal of proposed keys by retirement; repeated retirement.
  - The bound on overlapping keys, counting only approved keys.
- **Listing and isolation.**
  - `Keys` lists future windows and excludes proposed and retired keys. It lists nothing for a revoked subscription.
  - Keys are isolated between workspaces.
  - Keys are deleted by a purge.
- **Confirm.**
  - Refused with suspension for a version that is unpublished, retired, or outside its window before and after.
  - Refused for an attempt time moved past the window's end.
  - Refused for a retirement during signing.
  - Refused for a signer whose actual key differs from the published key.
  - Under all of these, no signature is released and no attempt is counted.
  - Concurrent retirement and confirmation under `-race`.
  - Rotation with overlapping versions.
  - `Redirect` refuses a confirmed delivery after its version is retired.
- **Held deliveries.**
  - Resume with the lists still full: not leased, and suspended again with `backlog`.
  - Resume after room is freed: buried.
  - Reasons kept while suspended; assertions for `rejected`, `max_age` and `attempts_exhausted`.
  - A leased delivery carries no reason.
- **Fixtures.** Updated `Limits` fixtures, with each new bound tested.

Checks:
- `-race -count=10`, a 386 build, gofmt and vet;
- the whole repository with `-race`;
- 100% coverage and mutation testing.

# Review history
- Design review round 1: FAIL (2 high, 4 medium, 5 low). Folded in:
  - key checks under the mutex at the final attempt time (H1);
  - verification of the signature against the published key, and an approval challenge (H2);
  - version names never reused, keys deleted with purged subscriptions, and nothing listed for revoked subscriptions (M1);
  - held deliveries never leased and buried again at dispatch (M2);
  - explicit and default refusals, with `key_unpublished` suspending the subscription (M3);
  - the retirement contract, and redirects refused after retirement (M4);
  - held deliveries skipped by expiry (L1);
  - the impossible re-leasing case replaced by an assertion (L2);
  - only approved keys counted, and proposals withdrawn by retirement (L3);
  - listing, actor and window rules (L4);
  - updated fixtures (L5).
- Design review round 2: FAIL (1 medium, 3 low). Folded in:
  - the approval recheck after the challenge, with the `Keys` limit under the mutex (N1);
  - a redirect refused after retirement returns the delivery to due without counting an attempt, rechecked under the mutex (N2);
  - withdrawn proposals release their names, and only approved names count (N3);
  - resuming without a key suspends again (N4).
- Design review round 3: PASS (3 low). Folded in: the test of names accepted again after withdrawal (P1); the bound on proposals (P2); expired keys not counted (P3); approval rechecking that the map still holds the same key object.

# Change review resolution

- Change review round 1: FAIL (2 medium, 5 low; no defect in production code). Resolved:
  - M1: tests for the `Keys` and `KeyLife` limit bounds.
  - M2: tests for an attempt time moved past the window's end during the callbacks, concurrent retirement and confirmation under `-race`, isolation between workspaces, a purged and re-created subscription without inherited keys, a window of exactly `KeyLife`, and signing at exactly `NotAfter`.
  - L1: the name bound pinned at its boundary.
- Residuals:
  - L2: approvals can add up to `Keys − 1` names beyond the bound, so the hard bound is `Keys × 65`.
  - L3: windows that are already expired are accepted and listed until retired.
  - L4: `KeyView` carries proposer and approver identifiers; a receiver-facing endpoint must project them away.
  - L5: repeated proposals and approvals return the stored view after revocation, without changing state.
- Change review round 2: PASS. Optional note: also assert the signed version in the concurrent retirement test.
