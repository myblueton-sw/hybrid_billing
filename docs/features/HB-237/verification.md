---
type: VerificationRecord
title: Connection trust verification
description: Records what was executed and reviewed for the first increment of work item W-33, a register of dual-controlled connections that pin certificate fingerprints to one workspace and its endpoints, Ed25519 push and batch verification with key versions, skew, replay store and quarantine, and outbound destinations checked on canonical URLs, every dialled address and every redirect. The record lists results, the mutation check, provisional choices and what remains unverified.
status: draft
---

# Delivered behavior

[HB-237](https://linear.app/hybrid-billing/issue/HB-237) adds the Go package `modules/trust` from main `d0c9684`. It uses only the standard library.

## Connections

- Connections hold one or two certificate fingerprints and registered endpoints.
- Every proposal gets a connection proposal number that is never reused. Pending proposals are immutable and can be withdrawn. Approval by another human must name the reviewed proposal number.
- A fingerprint belongs to one active connection in the installation. Revoked or replaced fingerprints become permanent tombstones.
- Approvals reserve a tombstone for every owned fingerprint, so revocation never fails.
- Admission refuses unknown, revoked or retired certificates, certificates of another workspace and unregistered endpoints, and counts each refusal per reason.

## Keys and pushes

- Dual-controlled Ed25519 public keys are held per version. Wrong lengths, non-canonical encodings and small-order encodings are refused (`weak_key`); Go's `ed25519.Verify` accepts an identity-key forgery, and the tests assert this. A key's validity window may not exceed the key life.
- Keys may overlap during rotation. At most `Keys` retired versions are kept.
- Pushes and batches are signed over a length-prefixed message, which a known-answer test pins. They are checked in this order:
  1. the connection is still at the admitted revision;
  2. the key is active;
  3. the signature verifies;
  4. the key's inclusive validity window;
  5. the inclusive skew;
  6. replay.
- Replays of accepted deliveries are `already_accepted` (counted). Conflicting identifiers are `duplicate`. The per-connection replay capacity is swept before a delivery is refused.
- Only refusals after a valid signature are quarantined, without bodies, and a dropped counter records overflow.

## Destinations

- Destinations are dual-controlled `https` origins. Hosts are canonicalised: ASCII LDH only, a last label that starts with a letter, strict IP literals, no mapped IPv4 literals, no `%`.
- An origin is unique per workspace. Private allowlists are validated.
- `CheckAddress` takes the socket address and port. It requires an IP-literal destination's own address, and classifies in this order:
  1. metadata and link-local;
  2. forbidden special-purpose blocks, including IPv4-embedding IPv6 and anything outside `2000::/3`;
  3. private ranges, unless allowlisted.
- `CheckRedirect` resolves against the current hop's URL. It allows only active destinations of the same workspace, counts hops, and returns the canonical URL to request.

# Commands and results

All runs were made in the worktree `/private/tmp/hb237` on 2026-11-15 with Go 1.25.7 on darwin/arm64.

| Check | Result |
| --- | --- |
| `gofmt -l modules/` | no output |
| `go vet ./modules/trust` | pass |
| `go test -race -count=10 ./modules/trust` | pass |
| `GOOS=linux GOARCH=386 go test -c ./modules/trust` | builds |
| `go test -race -count=1 ./...` | 50 packages, all pass |
| Coverage | `modules/trust` 100.0% |
| Mutation testing | 596 mutants of `address.go`, `registry.go`, `push.go` and `destination.go` on the final code: 541 killed, 13 did not build, 42 survived |

The 42 survivors were read and judged equivalent:

- **Address tables (27).** Each mutant changes host bits outside a prefix's mask, which `Contains` ignores, or widens a prefix to an equal one (for example `198.19.0.0/15` or `2001::/3`).
- **Rounding (5).** `Round(1)` instead of `Round(0)`.
- **Sort order (3).** `<=` in sorts and in pruning order, which only ever see distinct keys.
- **Limits array (1).** A `[][3]` limits array.
- **Field prime (1).** Replacing the field prime constant with p-1 changes only y = p-1, whose encodings are both in the small-order list.
- **URL parsing (2).** Opaque URLs and bracketed non-IPv6 hosts are already refused by `url.Parse` and the host rules.
- **Character limits (3).**
  - `>` 0x7f is already refused by `url.Parse`.
  - Non-ASCII host bytes are always multi-byte, so a check that skips the first byte, or treats 0x80 itself as ASCII, still refuses them through the remaining bytes or the label rules.

Property tests:

- 200,000 random addresses (20,000 with `-short`), biased to every table entry, with allowlists. Classifications and allowlist shapes are compared with an independent bit-level oracle of the plan's policy, and every class is seen at least 100 times.
- 100 seeds of random pushes and batches. Outcomes and quarantine counts are compared with an independent replay model.

# Reviews

- **Design review:** two rounds by a separate read-only reviewer: FAIL, PASS ([plan](plan.md)).
- **Change review:** four rounds by a fresh read-only reviewer.
  1. FAIL: revocation blocked by the tombstone bound.
  2. FAIL: replaceable pending proposals.
  3. FAIL: withdraw and propose again.
  4. PASS.

# Provisional choices for the owner

- Limits: the skew range (1 second to 1 hour), the key life range (1 hour to 10 years), at most two fingerprints per connection, 10 redirects and 16 allowlist prefixes.
- The fixed metadata address list.
- Refusing all of `2001::/23`.
- Revocation without dual control.
- Counting `already_accepted` instead of quarantining it.

# Not verified and residuals

- **TLS and certificates.** TLS termination, chain validation and proof of possession of the certificate key are not built here.
- **Dialer and screen obligations.**
  - The dialer must use no proxy and check the exact socket address in its control hook.
  - Pooled connections must not be reused across destinations or after a revocation.
  - The client must request only the returned URL and pass hop counts on.
  - Approving clients must send the proposal number the human reviewed.
- **Accepted residuals.**
  - Fingerprint squatting and probing.
  - A pruned key version may be reused.
  - `Admission` values are trusted in process.
  - Operator NAT64 prefixes inside `2000::/3`.
  - Reads take no actor.
- **Later work items.**
  - OAuth and PKCE (QA-04).
  - Webhook sending, signing and pre-send permission checks (W-34).
  - Inbox deduplication of business events.
  - The HB-177 journal retention blocker.
  - Missing business commands, SDKs, persistence and screens.
