---
type: ImplementationPlan
title: Connection trust for internal connectors and push sources with pinned certificates bound to one workspace, signed pushes checked for key version, time and replay, and outbound destinations checked on resolved addresses and redirects
description: Defines the first increment of work item W-33, a register in which dual-controlled connections pin client certificate fingerprints to one workspace and registered endpoints, pushes and signed batches are verified with Ed25519 public keys of a version, an allowed clock skew and an event identifier store before acceptance and refused deliveries are quarantined without their bodies, and dual-controlled outbound destinations are checked on every resolved address and redirect so that loopback, link-local, metadata, unapproved private and mapped addresses are never reached.
status: draft
---

# Scope and ownership

[HB-237](https://linear.app/hybrid-billing/issue/HB-237) starts work item W-33 (requirement HB-33; HB-33-D/I/V; AR-02 and the inbound signature and outbound destination parts of QA-05) after W-32. Base: main `d0c9684`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the plan and the change, because this register decides which connections and addresses the installation trusts. Machine: mac_mini.

The command registry, idempotency scopes, expected revisions, stable error codes and the OpenAPI document already exist in `modules/commands` (HB-176). This increment adds what HB-176 lacks: the trust decisions for internal connections, pushes and outbound destinations.

The following are out of scope and recorded as residuals:

- TLS termination, certificate chain validation and the socket dialer (this register is called with the presented certificate's fingerprint and with each resolved address);
- OAuth and PKCE for external servers (QA-04);
- webhook subscriptions and sending (W-34 outbox);
- CloudEvents envelopes and source plus identifier deduplication of business events (W-34 inbox);
- cursor expiry, the journal retention blocker HB-177, registering the missing business commands, SDKs, persistence and screens (S01, S03, P02).

# Observed baseline

- AR-02:
  - Internal connectors support certificate-based mutual authentication, tenant binding per connection and restriction to registered endpoints.
  - Allowed destinations are checked on their actual resolved addresses and on redirects, so that forbidden and metadata addresses are never reached.
  - Pushes and signed batches are checked for signature, key version, time and event identifier, and expired, forged and replayed deliveries are quarantined.
  - The allowed skew and the key rotation period are configuration.
- AUD-AR-02 requires refusing another certificate or tenant, forged, expired and duplicate pushes, DNS destination changes, forbidden redirects and metadata addresses, and accepting only approved internal connections.
- QA-05: webhook signature, time, event identifier, key rotation and replay protection; approved destinations with a private allowlist; metadata, arbitrary loopback, DNS rebinding and unapproved redirects blocked.
- Sibling registers use one mutex, an injected clock, workspace isolation, dual control for security configuration, and the errors `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (`Refusal{Reason}`) and `ErrLimit`.

# Decision

A new package `modules/trust` follows the sibling conventions:

- an injected clock that never goes backward and one mutex;
- check order form, lookup, repeat, rule, room;
- human actors for configuration, an assistant actor refused (`ai_not_allowed`), and a system actor refused (`not_human`);
- `NewRegistry(Limits{Workspaces, Connections, Retired, Endpoints, Keys, Destinations, Events, Items, Quarantine, Allowlist, Redirects, Body, Skew, KeyLife}, clock)`. Every count is positive and bounded, `Items` is at most `Events`, and `Skew` and `KeyLife` are within fixed ranges.

It uses only the standard library (`crypto/ed25519`, `crypto/sha256`, `math/big`, `net/netip`, `net/url`), so it is independent of the operating system. Within `Verify`, the repeat check comes after the signature and time rules on purpose: only authenticated deliveries may touch the replay store.

## Connections

`ProposeConnection(workspace, Connection{ID, Fingerprints, Endpoints}, actor)` by one human and `ApproveConnection(workspace, id, proposal, actor)` by another make a connection active. Every proposal gets the next proposal number of its connection, never reused, and the approval must name the number it reviewed (`proposal_changed`):

- `Fingerprints` holds one or two SHA-256 digests of client certificates. Two allow a certificate rotation with overlap.
- `Endpoints` holds one or more registered endpoint identifiers.
- A fingerprint can belong to only one active connection in the whole installation, and a fingerprint that was ever revoked or replaced is kept as an installation-wide tombstone that can never be registered again (`fingerprint_in_use`, `fingerprint_retired`).
- Retired fingerprints are bounded by `Limits.Retired` for the installation (`ErrLimit`). A replacement proposal may keep fingerprints of its own active revision, and revoked connection identifiers are never reused (`revoked`).
- Uniqueness is checked at proposal and again at approval. Pending proposals do not reserve fingerprints: the first approval wins, and a later approval of another proposal with the same fingerprint is refused.
- A pending proposal never changes: it is approved as proposed or withdrawn by a human with `WithdrawConnection`, after which a new proposal starts a new approval round.
- A proposal for an active connection replaces it only on approval, and that approval retires the fingerprints that are dropped. Until then the active revision keeps working.
- `RevokeConnection` by a human ends it at once and retires its fingerprints. Revocation is not dual-controlled, because ending trust must be fast.
- Proof of possession of the certificate key belongs to the TLS layer (residual).

`Admit(Presented{Fingerprint, Workspace, Endpoint})` is called after TLS has verified the certificate chain. `Workspace` and `Endpoint` come from the request route. Admit returns an `Admission{Workspace, Connection, Revision}`, or a refusal:

- `unknown_certificate`, which covers revoked, retired and never-approved fingerprints;
- `workspace_mismatch`, when the certificate belongs to another workspace's connection;
- `endpoint_not_registered`.

Admission refusals are counted per reason for the installation, because an unknown certificate names no workspace. Nothing about the presented values is stored.

## Keys and pushes

`AddKey(workspace, connection, Key{Version, Public, NotBefore, NotAfter}, actor)` by one human and `ApproveKey` by another register an Ed25519 public key under a version identifier:

- Only public keys are held; signing keys never enter the installation.
- A public key must be 32 bytes, a canonical encoding (y below the field prime) and not one of the small-order points, including the encodings with the sign bit set and x = 0. Such keys would accept forged signatures (`weak_key`).
- `NotAfter - NotBefore` must be at most `Limits.KeyLife`, the provisional rotation period.
- Several versions may be valid at once, which gives overlap during rotation, up to `Limits.Keys` per connection.
- `RetireKey` by a human ends a version at once.

`Verify(admission, Push{EventIDs, KeyVersion, Signed, Body, Signature})` checks a single push (one event identifier) or a signed batch (up to `Limits.Items` identifiers, all or nothing; repeated identifiers inside one delivery are `ErrInvalid`) from an admitted connection. The body may be at most `Limits.Body` bytes.

The signed message is the concatenation of parts, each preceded by its length as an 8-byte big-endian integer:

- the domain tag `trust_push_v1`;
- the workspace, connection and key version;
- the decimal count of event identifiers, then each identifier in order;
- the signing time as decimal Unix nanoseconds;
- the 32-byte SHA-256 of the body.

A known-answer test pins this encoding. The checks run in this order, and the first failure decides the outcome:

1. The admission's connection is still active at the same revision (`connection_changed`).
2. The key version is known and approved (`unknown_key`).
3. The signature verifies (`forged`).
4. The key was valid at the signing time, with `NotBefore` and `NotAfter` both inclusive, and is not retired (`key_expired`).
5. The signing time is within `Limits.Skew` of now, inclusive, in either direction (`stale`).
6. The replay rules below.

Replay rules:

- If every identifier was already accepted for this connection with the same body hash, the result is `already_accepted`. This is not a success; the caller can acknowledge it so at-least-once senders stop retrying. It is counted per workspace but not quarantined, because an identical, authentic re-delivery carries no new evidence and quarantining every retry would crowd out real refusals. The replay store protects only within the time window: a sender that signs again after it is accepted again, so business deduplication by source and identifier stays with the W-34 inbox.
- Any other overlap with accepted identifiers is `duplicate`.
- Otherwise all identifiers are recorded with the body hash and the delivery is accepted.

Each accepted identifier is kept while now is at or before its signing time plus the skew; after that the time check alone refuses a replay. The store holds at most `Limits.Events` identifiers per connection. When it is full, expired identifiers are swept first; if it is still full, the delivery is refused with `replay_capacity` rather than accepted unchecked.

Refusals before the signature verifies (`connection_changed`, `unknown_key`, `forged`) carry values the sender chose freely. They are only counted per workspace and reason. Refusals after the signature verifies (`key_expired`, `stale`, `duplicate`, `replay_capacity`) are quarantined per workspace. A quarantine entry records the connection, event identifiers, key version, reason, signing time, receipt time and the SHA-256 of the body, but never the body. When the quarantine holds `Limits.Quarantine` entries, further refusals increment a dropped counter, so a loss is still recorded. A human reads it with `Quarantine(workspace, after, limit, actor)`.

## Destinations

`ProposeDestination(workspace, Destination{ID, URL, Private}, actor)` by one human and `ApproveDestination` by another approve an outbound destination. The URL rules:

- `https`, with no user information, query, fragment, backslash or control characters, and a host with an optional port. Port 443 is the default and is written explicitly in the canonical form.
- A DNS host is lowercased with any trailing dot removed. It must consist of ASCII letter, digit and hyphen labels (punycode `xn--` labels are allowed as given, Unicode is not), and its last label must start with a letter. This refuses numeric and hexadecimal forms such as `2130706433` or `0x7f.1`.
- An IP literal host must parse as a canonical address (bracketed for IPv6) and pass the address rules.
- The canonical origin (host and port) is unique among the workspace's proposed and active destinations (`ErrConflict`).

The `Private` allowlist holds at most `Limits.Allowlist` masked prefixes, each inside one of the private ranges below, none of them mapped IPv4 forms. `RevokeDestination` by a human ends a destination at once.

`CheckAddress(workspace, destination, netip.AddrPort)` is called by the dialer for the exact socket address of every connection it opens. The port must equal the destination's port (`port_mismatch`). Because each socket address is checked at dial time, a DNS answer that changes between resolutions (rebinding) cannot bypass the check. Mapped IPv4 (`::ffff:0:0/96`) is unmapped first, and zones are refused.

The address is then classified in this order:

1. **Metadata (`metadata_address`).** All link-local addresses (`169.254.0.0/16`, `fe80::/10`) and the known cloud metadata addresses (`169.254.169.254`, `192.0.0.192`, `100.100.100.200`, `fd00:ec2::254`, `fd20:ce::254`). These are refused even under an allowlist.
2. **Forbidden (`forbidden_address`).** Every IANA special-purpose block that is not globally reachable or that embeds IPv4:
   - IPv4: `0/8`, `127/8`, `192.0.0.0/24`, `192.0.2.0/24`, `192.88.99.0/24`, `198.18.0.0/15`, `198.51.100.0/24`, `203.0.113.0/24`, multicast `224/4`, `240/4` and the broadcast address;
   - IPv6: unspecified, loopback, IPv4-compatible `::/96`, SIIT `::ffff:0:0:0/96`, NAT64 `64:ff9b::/96` and `64:ff9b:1::/48`, discard `100::/64`, `2001::/23` (including Teredo), documentation `2001:db8::/32` and `3fff::/20`, 6to4 `2002::/16`, SRv6 `5f00::/16`, site-local `fec0::/10` and multicast `ff00::/8`;
   - any IPv6 address outside global unicast `2000::/3` that is not in a private range.
3. **Private (`private_address`).** Private ranges, refused unless a prefix of the destination's allowlist covers them: `10/8`, `172.16/12`, `192.168/16`, `100.64/10` and `fc00::/7`.
4. Everything else is allowed.

`CheckRedirect(workspace, destination, current, hop, location)` is called for every redirect. `current` is the canonical URL requested at this hop (the destination URL, or the URL returned by the previous `CheckRedirect`), and it must belong to the named destination. Redirect scope is per origin, so any path of an approved origin is allowed.

- `hop` is the number of redirects already followed, and the result carries `hop + 1`. More than `Limits.Redirects` redirects are refused (`redirect_limit`). The caller supplies the count, so the limit holds only if the caller passes it on (residual).
- The location is resolved against `current`. The resolved URL must use `https`, carry no user information, backslash or control characters, and its canonical origin must be that of an approved destination of the same workspace (`unapproved_redirect`).
- The result is that destination and the canonical absolute URL. The client must request exactly that URL and check its addresses in turn.

The dialer has these obligations, which are residuals of this register:

- use no proxy;
- check the exact socket address in the dial control hook;
- never reuse pooled connections across destinations or after a revocation;
- request only the URL that `CheckRedirect` returned.

Accepted residuals: a workspace that approves another workspace's certificate fingerprint first blocks it (and a revocation then tombstones it), and `fingerprint_in_use` tells a proposer that a certificate is registered somewhere. Neither exposes data, because `Admit` refuses the other workspace, and the owner recovers with a new certificate.

Outbound webhook signing, its key rotation and re-checking permissions just before sending belong to W-34.

# Verification plan

Literal tests cover every refusal reason and every repeat, plus the following groups.

Connections:

- a certificate of another workspace;
- the same fingerprint in two pending proposals, where the first approval wins and the second is refused;
- re-registration of a revoked or replaced fingerprint;
- certificate rotation with two fingerprints;
- an unregistered endpoint;
- revoked connections, keys and destinations;
- an admission whose connection changed before `Verify`.

Pushes:

- forged, tampered-body, wrong-version, expired-key, early, late, duplicate, same-identifier-different-body, already-accepted and replay-after-window pushes, with exact skew boundaries (signing time plus and minus the skew, and one nanosecond beyond);
- signed batches, all or nothing;
- key rotation overlap;
- small-order and non-canonical public keys, including a forgery attempt with the identity key;
- a known-answer vector of the signed message;
- quarantine without bodies, and its dropped counter;
- forged floods that do not displace genuine quarantine entries;
- the replay store sweep and its per-connection capacity, isolated between workspaces.

Addresses:

- each address class, including `64:ff9b:1::/48`, `::7f00:1`, `::ffff:0:7f00:1`, Teredo, site-local, documentation and benchmark ranges;
- metadata addresses under an allowlist, and metadata before link-local precedence;
- allowlist validation;
- duplicate origins;
- a port mismatch;
- a rebinding sequence where one name gives a public and then a metadata address;
- revocation between `CheckRedirect` and `CheckAddress`.

URLs and redirects:

- canonical host forms: case, trailing dot, Unicode, numeric and hexadecimal hosts, bracketed IPv6 and an explicit `:443`;
- redirects to other hosts, other ports, `http`, user information, backslashes, another workspace's destination and relative locations;
- the redirect limit.

Actors: an assistant or system actor on every configuration call, and the same human approving their own proposal.

Property tests:

- random addresses against an independent classifier of this plan's conservative policy (for example the whole of `2001::/23` refused), written from the IANA tables;
- random push sequences against an independent replay model.

Checks:

- `-race -count=10`, a 386 build, gofmt and vet;
- the whole repository with `-race`;
- 100% coverage and mutation testing.

# Review history
- Design review round 1: FAIL (1 high, 8 medium, 7 low). Folded in: refusal of all non-global and IPv4-embedding special-purpose blocks with metadata, forbidden and private precedence (H1); fingerprint tombstones and approval-time uniqueness (M1); quarantine only after a valid signature (M2); per-connection replay capacity (M3); already-accepted retries apart from conflicting duplicates (M4); signed batches all or nothing (M5); unique origins (M6); allowlist validation (M7); canonical hosts, ports in address checks, canonical redirect URLs and dialer obligations (M8); inclusive skew boundaries (L1); the signed message encoding with a known-answer test (L2); weak public keys (L3); `Verify` bound to an admission (L4); hop counts returned and per-origin scope (L5); quarantine readers and body bound (L6); scope wording and the order inside `Verify` (L7).
- Design review round 2: PASS (8 low). Folded in: redirects resolved against the URL of the current hop (1); a bound for retired fingerprints (2); identical re-deliveries counted, not quarantined, and inbox deduplication kept for W-34 (3); fingerprint squatting and probing as accepted residuals (4); limit rules (5); sign-bit small-order keys, inclusive key validity and repeated identifiers (6); replacements keeping their own fingerprints and revoked identifiers never reused (7); the property oracle encoding this policy (8).

# Change review resolution

- Change review round 1: FAIL (1 high, 6 low and hypotheses). Resolved:
  - H1: every approval reserves a tombstone for each fingerprint it makes owned. The installation-wide sum of retired and owned fingerprints stays within `Limits.Retired`, so revocation never fails on capacity, and a rotation in one workspace cannot block another workspace's revocation. The bound now refuses approvals instead. The test asserts that revocation succeeds at the bound.
  - L1: hosts with non-ASCII bytes (after percent decoding) or `%` are refused.
  - L2: bracketed mapped IPv4 literals are refused.
  - L3: a retired key is `unknown_key`: it is counted, not quarantined, before the signature check. `key_expired` covers only the validity window.
  - L4: the test asserts that Go's `ed25519.Verify` accepts the identity-key forgery, the reason for the weak-key rule.
  - L5: the address property test checks allowlist shapes against an independent bit oracle and feeds unmapped inputs half the time.
  - L6: at most `Limits.Keys` retired versions are kept per connection. The oldest retired version is removed, after which its identifier may be used again.
  - L7: for an IP-literal destination, `CheckAddress` also requires the dialled address to be that literal (`address_mismatch`). Azure's `168.63.129.16` is added to the metadata addresses. Pending proposals are immutable and can be withdrawn with `WithdrawConnection`, so a pending connection change never blocks.
- Residuals:
  - `Admission` values are trusted in process.
  - Operator NAT64 prefixes inside `2000::/3` need configuration.
  - Configuration reads take no actor, like the sibling registers.
- Change review round 2: FAIL (1 medium, 1 low). Resolved: a proposer could replace a pending proposal after the second human reviewed it (N1), and could not cancel a pending replacement (N2). Pending proposals are immutable again, and `WithdrawConnection` clears them so that a replacement needs a new approval round. Accepted note: after a retired key version is pruned its identifier may be reused, so older quarantine entries name the version only.
- Change review round 3: FAIL (1 medium). Resolved: after a withdrawal the proposer could propose new content that the approver had not reviewed (N1). Each proposal now carries a proposal number that is never reused, and `ApproveConnection` must name it. Keys and destinations cannot be re-proposed under an identifier that is still known, so their approvals are not exposed to this.
- Change review round 4: PASS. Residual: the binding protects dual control only if the approving client sends the proposal number from the view the human reviewed, never one fetched and approved in a single automated step (screen obligation).
