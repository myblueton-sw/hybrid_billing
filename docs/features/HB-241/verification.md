---
type: VerificationRecord
title: Webhook key publication and held-delivery cleanup verification
description: Records what was executed and reviewed for HB-241, a follow-up of HB-240 in work item W-34. It covers per-subscription published Ed25519 verification keys, with dual-controlled publication proven by a signer challenge, fast retirement and bounded names. Confirm signs only with a published version valid at the attempt time, and the signature is verified against the published key under the mutex. Deliveries held for terminal reasons are never leased again. The record lists results, the mutation check and the residuals.
status: draft
---

# Delivered behavior

[HB-241](https://linear.app/hybrid-billing/issue/HB-241) changes `modules/webhooks` from main `a0e68cd`.

## Keys

- **Publication.** Keys are proposed by one human and approved by another. A key must be 32 bytes, canonical and not of small order, with a window of at most `KeyLife` whose two ends are inclusive.
- **Approval challenge.** Approval needs the signer to sign a challenge that verifies against the proposed key. After that, approval re-locks and requires the same proposed key object, a subscription that is not revoked, and room within `Keys` (active, unexpired keys only).
- **Names.** Names of approved keys are never reused, up to `Keys × 64` names.
- **Proposal bound.** Outstanding proposals are bounded at `Keys`.
- **Retirement and withdrawal.** Retiring an approved key ends it at once. Withdrawing a proposal releases its name.
- **Listing.** `Keys` lists active keys, including future windows. It lists nothing for a revoked subscription and refuses assistants.

## Confirm

- After re-locking, Confirm requires the signer's version to be approved, not retired, and valid at the final attempt time.
- It verifies the signature with `ed25519.Verify` against the published key.
- If any check fails, the delivery returns to due with no attempt counted and no signature is released. The subscription is suspended with `key_unpublished` or `key_mismatch`.
- Every reason from the check is matched, and a default case refuses any unknown reason.
- The delivery keeps the version it was signed with.

## Redirect and held deliveries

- Redirect re-checks that version under the mutex. If the key was retired, it returns the delivery to due without counting an attempt.
- A delivery is marked held when it is kept open because the lists are full.
  - Expiry skips held deliveries.
  - Dispatch buries a held delivery again rather than leasing it.
  - A dismissal buries only held deliveries.

# Commands and results

All runs were made in the worktree `/private/tmp/hb241` on 2026-11-15 with Go 1.25.7 on darwin/arm64.

| Check | Result |
| --- | --- |
| `gofmt -l modules/` | no output |
| `go vet ./modules/webhooks` | pass |
| `go test -race -count=10 ./modules/webhooks` | pass (a test that depended on map order was fixed first) |
| `GOOS=linux GOARCH=386 go test -c ./modules/webhooks` | builds |
| `go test -race -count=1 ./...` | 52 packages, all pass |
| Coverage | `modules/webhooks` 100.0% |
| Mutation testing | 555 mutants of `subscriptions.go`, `deliveries.go` and `keys.go`: 490 killed, 17 did not build, 48 survived |

The 48 survivors were read and judged equivalent.

The six in `keys.go`:

- the field prime p − 1, whose encodings are both in the small-order list;
- the challenge buffer width, which the signer and the verifier share;
- `Round(1)`, twice;
- a redundant condition in the approval recheck;
- `<=` when sorting distinct versions.

The other 42 are in `deliveries.go` and `subscriptions.go`, of the kinds recorded for HB-240.

The weak-key test uses an independent literal list. A fake Ed25519 signer covers matching keys, mismatching keys, failures and hooks that run during the callbacks.

# Reviews

- **Design review:** three rounds by a separate read-only reviewer. Rounds 1 and 2 failed; round 3 passed ([plan](plan.md)).
  - Round 1 found retirement races and the key-to-signer binding.
  - Round 2 found a race during the approval challenge.
- **Change review:** two rounds by a fresh read-only reviewer. Round 1 failed because tests the plan required were missing; no defect was found in the production code. Round 2 passed.

# Residuals

- **Name and key limits.**
  - The hard name bound is `Keys × 65`.
  - Windows that have already expired are accepted and listed until retired.
- **Key views.** A key view carries proposer and approver identifiers, so a receiver endpoint must project them away.
- **Repeated calls after revocation.** Repeating a proposal or approval after revocation returns the stored view.
- **Not built here.**
  - Custody of private keys.
  - The HTTP endpoint that serves the keys.
  - Persistence.
  - Unique subscription identifiers after a purge (HB-240 N2).
