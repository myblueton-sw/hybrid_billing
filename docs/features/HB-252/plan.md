---
type: ImplementationPlan
title: License mode register with explicit mode selection, dual-controlled ed25519 verification keys, signed evaluation grants on authoritative time, offline-verified purchased entitlements, nonblocking status and DB-14 rights checks
description: First increment of work item W-40. A register records the selected license mode per installation. Evaluation grants and purchased entitlements are verified offline against trusted public keys only, and an evaluation lasts 336 hours from an authoritative activation instant that repeating an activation never resets. Outcomes are kept apart (invalid, unverifiable, scope mismatch, expired and unverified), a valid registration is never replaced by an invalid one, and the status read gives a persistent nonblocking warning. Overlapping direct and marketplace rights, and billing metrics outside the contracted set, are refused. Nothing is blocked, locked, sent or deleted.
status: final
---

# Scope and ownership

[HB-252](https://linear.app/hybrid-billing/issue/HB-252) is the first increment of work item W-40 (requirement HB-40). It implements:
- the owner-accepted requirements in [usage-modes-requirements.md](../../licensing/usage-modes-requirements.md), acceptance scenarios LM-01 to LM-11;
- the entitlement part of HB-40-I: offline verification, and keeping access to past evidence;
- the DB-14 rights model checks behind AUD-DB-14.

Every rule of [product-policy.md](../../licensing/product-policy.md) is kept: there is no activation check that blocks anything, no automatic external reporting, no remote lock, no fee collection and no enforcement runtime.

It does not cover:
- key issuance, private-key custody and signing tools;
- online activation and revocation services;
- the OSS candidate register (pinned versions, LICENSE, SBOM), which is increment 2;
- legal suitability, which no analysis here settles;
- persistence and screens.

Base: main `d7a88cf`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the design and the change, because the requirements document asks for an independently reviewed design of authority, identity binding, replay, persistence, restore, clock anomalies and key rotation. Machine: mac_mini.

Allowed files: `modules/licensemode/*` and `docs/features/HB-252/*`.

# Observed baseline

- **Modes.** There are three modes, each with its own basis:
  - noncommercial, on the PolyForm permitted purposes;
  - commercial evaluation, on separate express permission for 14 days;
  - purchased commercial, on an actual agreement plus an authentic signed entitlement.

  Selecting a mode, downloading the software, a corporate email or an IP address never grants or infers commercial rights. Switching modes keeps records and never creates a fresh trial.
- **Evaluation.** An evaluation starts only after explicit permission and acceptance are recorded: the grant, the subject, the terms version, the acceptance, the activation, the authoritative UTC start and end, and the scope.
  - It lasts 336 hours, with an inclusive start and an exclusive end.
  - Reinstalling, restoring or rolling back the clock must not reset or extend it.
  - A purely local self-hosted check cannot resist an operator who controls both code and storage, and this must be stated.
  - When time or verification is unavailable, the state is explicitly unverified, never a new start or a silent extension.
- **Purchased.** The signature is verified against a trusted public key before the contents are trusted. Private signing keys are never shipped.
  - The claims bind the issuer, the legal entity, the agreement, the product, the scope, the validity (or an agreed perpetual term) and the key identifier.
  - Altered payloads are invalid, and unknown keys are unverifiable. A verified scope mismatch is shown apart from expiry.
  - A currently valid registration is never replaced by an invalid submission.
  - Registration is distinct from authentication, tenant authorisation and billing, and never changes invoice amounts.
- **Status.** The expiry warning is persistent and nonblocking, and offers register and contact actions.
  - There is no automatic email, lock, deletion or denial of historical access or export.
  - Network unavailability is shown apart from expiry, with the last verified evidence and its timestamp.
- **DB-14.** Product marketplace sales are a later scope. When introduced, direct licences and marketplace subscriptions are linked in one rights model, so that the same subject and installation never pay twice. Customer cloud billing amounts are never used as product licence usage; the contracted billing metric is. AUD-DB-14 requires tests before activation.

# Decision

A new package `modules/licensemode` follows the sibling conventions:
- one mutex and an injected clock that never goes backward;
- `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (with `Refusal{Reason}`) and `ErrLimit`;
- check order: form, lookup, repeat, rule, room;
- assistants refused as `ai_not_allowed` and system actors as `not_human` where a human is required;
- numbered proposals approved by a second human with a recheck, withdrawal, register-wide pending caps (1,000 in total, 250 per proposer) and 1,000 applied proposals kept.

It imports only the standard library, including `crypto/ed25519`.

`NewRegistry(Config{Authority}, clock)`:
- `Authority` is a port, `Now() (time.Time, bool)`, giving the separately established authoritative time. It returns `false` when that time is unavailable. It is called while the register's lock is held, so it must return quickly and must never call back into the register.
- The injected local clock stamps records and bounds how far ahead the Authority may be.

## Trusted keys

`key` proposals register `Key{ID, Issuer, Public, NotBefore, NotAfter, AcceptUntil, Proof}`:
- **`Public`** is exactly 32 bytes in lowercase hex: an ed25519 public key.
  - 64 bytes is refused with `private_key`, because that is the length of an ed25519 private key.
  - Any other length is `ErrInvalid`.
- **`Proof`** is an ed25519 signature under `Public` over the length-prefixed encoding of `hybrid-billing/license-key/v1`, the key ID and the issuer. A key without a valid proof is refused with `no_proof`.
  - A 32-byte private seed pasted as `Public` is therefore refused, because nobody holds the private key for it.
  - Small-order public keys cannot be fully excluded with the standard library. This is a residual, mitigated by dual control.
- **Two windows.** All bounds are whole seconds.
  - The signing window `[NotBefore, NotAfter)` bounds a document's `IssuedAt`.
  - The acceptance window `[NotBefore, AcceptUntil)`, with `AcceptUntil ≥ NotAfter`, bounds when new submissions signed by the key are accepted.
  - An offline or restored installation can therefore still import a long-term entitlement after its key stopped signing, until `AcceptUntil`. After that, the issuer re-signs live entitlements under the new key. This is the documented rotation procedure.
- **Rotation.** A `retire_key{Key, At}` proposal moves the signing end `NotAfter` earlier to `At`, and leaves `AcceptUntil`. Retiring again is refused with `retired`, and retiring a compromised key with `compromised`. `At` must not be earlier than the authoritative time when it is approved, otherwise `retire_past`, so a rotation cannot silently cut off documents already accepted.
- **Compromise.** A `compromise_key{Key}` proposal marks the key compromised, whether the key is active or already tombstoned. From then on, every document signed by that key verifies as `unverifiable`, including entitlements and activations already recorded.
  - `Status` re-checks compromise, and only compromise, at every read, so a leaked key cannot keep or create rights while normal rotation never invalidates accepted rights.
  - Compromised entitlements are left out of every computation: the current entitlement, effective ends, `superseded` and DB-14 overlaps. Because effective ends and the current entitlement are computed on read (see Commercial registration), a compromised entitlement that superseded a genuine one stops cutting it short the moment the compromise is marked. The genuine one is current again without being registered again.
- **IDs.** Key IDs are never reused. Retired and compromised keys move to a tombstone set that does not count toward the active key cap.

## Signed documents

A document is `Signed{Body, Signature}`:
- **`Body`** is the canonical encoding below of `Claims`.
- **`Signature`** is a 64-byte ed25519 signature over `hybrid-billing/license/v1` followed by `Body`.

`Claims{Kind, ID, Issuer, KeyID, IssuedAt, SubjectID, Subject, Agreement, Supersedes, Product, Installation, Scope, Channel, Metric, Start, End, Perpetual, Terms, ActivateBy}`:
- **`Kind`** is `evaluation` or `commercial`.
- **`SubjectID`** is an identifier for the contracted legal entity. `Subject` is its display name.
- **Fields by kind.** Fields that do not apply to the kind must be empty, or the body is `invalid`.
  - An evaluation needs `Terms` and may carry `ActivateBy`. It has no times, agreement, channel or metric, because its interval comes from activation.
  - A commercial entitlement needs `Agreement`, `Channel` (`direct` or `marketplace`), `Metric` and `Start`, plus either `End > Start` or `Perpetual` with an empty `End`. It may carry `Supersedes`, the agreement ID it renews.
- **`Scope`** lists 1–32 identifiers, strictly ascending, so there are no duplicates.
- **`Metric`** comes from the closed set of contracted product metrics: `installations`, `seats`, `tenants`, `documents`. Customer cloud spend is not a member and cannot be named.
- **Intervals.** Intervals are half-open: `[Start, End)`, and `[Start, ∞)` when perpetual.

Verification runs in this fixed order, so that an edited document is always reported as `invalid`:
1. **Decode.** The body must decode exactly, with every length within the remaining bytes and no trailing bytes. Otherwise it is `invalid`.
2. **Key lookup.** An unknown key ID is `unverifiable`.
3. **Signature.** The signature is verified, and a failure is `invalid`.
4. **Time.** Authoritative now is read here. Unavailable time gives `time_unavailable`, and an anomalous time gives `clock_anomaly`, never `unverifiable`.
5. **Issuer and validity.** The result is `unverifiable` if:
   - the key's issuer differs from the claimed issuer;
   - the key is compromised;
   - `IssuedAt` is outside `[NotBefore, NotAfter)`;
   - `IssuedAt` is later than authoritative now;
   - authoritative now is outside the acceptance window `[NotBefore, AcceptUntil)`.

   A document backdated into the signing window after a compromise is caught by the compromise flag. After normal rotation it is accepted only until `AcceptUntil`.
6. **Kind.** A document of the wrong kind for the call is `wrong_kind`.
7. **Scope.** An `Installation` or `Product` different from the installation's is `scope_mismatch`.

Contents are used only after step 3.

## Installations and modes

Every mutating call needs a human actor (`not_human`, `ai_not_allowed`):
- **`Install(installation, product, actor)`** registers an installation once.
- **`SelectMode(installation, mode, actor)`** records the declared mode (`noncommercial`, `evaluation` or `commercial`).
  - It never grants anything, and no other field (email, network, employer) is accepted.
  - Every selection is kept in the installation's history.
  - Selecting `evaluation` does not start an evaluation.

## Authoritative time and clock anomalies

- **Source.** Every call that needs time reads `Authority.Now()`.
- **High-water mark.** The register keeps the latest authoritative time it has seen as a high-water mark, never moving it back. These are clock anomalies, and neither moves the mark:
  - a value earlier than the mark (`clock_anomaly`);
  - a value more than 24 hours ahead of the injected local clock (`local_clock_behind`). The local clock is used for this sanity bound as well as for stamps.
- **Reset.** A dual-controlled `reset_high_water{At, Reason}` proposal sets the mark back to `At`, for recovery from a wrong future reading. It does not need authoritative time, since recovery is exactly when that time is wrong. It is kept in history.
  - `At` must not be earlier than the latest evaluation activation start, otherwise `reset_too_early`.
- **Latched expiry.** Once any call has observed an evaluation's own end, or an entitlement's own `End`, at or before authoritative now, that record is latched as expired, whatever the mark or the Authority later reports. A reset therefore cannot revive anything.
  - Latching never uses an effective end, so a cut made by a later-compromised entitlement is never latched.
  - `reset_too_early` checks only evaluation starts, because the latch already covers entitlements.

  Approvers who also control the Authority can still misreport time before expiry is first observed. This is a stated residual.
- **When time is unusable.** If time is unavailable or anomalous:
  - `Activate` and `Register` are refused with `time_unavailable`, `clock_anomaly` or `local_clock_behind`;
  - `Status` returns `unverified` with that reason.

  This way an expired evaluation can never be revived by rolling time back.
- **Local clocks.** An `Authority` backed by the installation's own clock does not satisfy LM-04. It must be a separately established source.

## Evaluation activation

`Activate(installation, Signed, Acceptance{Actor, Terms}, actor)`:
- **Preconditions.**
  - `Acceptance.Actor` must equal `actor`.
  - The document must be verified, with `Kind` `evaluation`.
  - The acceptance must name exactly the grant's `Terms`, otherwise `terms_mismatch`.
  - The installation's mode must be `evaluation`, otherwise `mode_mismatch`.
  - If `ActivateBy` is set and authoritative now is at or after it, the grant is `stale_grant`.
- **Interval.** Start is the authoritative instant truncated to whole seconds, so that it compares exactly with the high-water mark and resets. End is start plus 336 hours: active on `[start, end)`.
- **Once per grant, once per installation.**
  - A grant ID can be activated once in the whole register. A repeat returns the original record and never a new interval.
  - An installation can have one evaluation. A second grant for it is refused with `already_evaluated`, and the original stays.
  - An extension needs reconciliation under a different agreement. It is a residual.

  A reinstall that keeps the register, a restore of the installation or a client clock change therefore cannot reset an evaluation. Restoring or replacing the register itself, by an operator who controls code and storage, cannot be detected. This is a stated residual.

## Commercial registration

`Register(installation, Signed, actor)` records an outcome in the installation's submission history: the outcome, the time, and the digest of the length-prefixed body and signature.
- **Privacy.** The history never keeps claims from documents that did not verify.
- **Outcomes.** The full list is:
  - `bad_supersedes`;
  - `invalid`, `unverifiable`, `wrong_kind`, `scope_mismatch`;
  - `time_unavailable`, `clock_anomaly`, `local_clock_behind`;
  - `not_yet_valid` (`Start` after now), `expired` (`End` at or before now);
  - `superseded` (its `IssuedAt` is not later than the current entitlement's);
  - `duplicate_right`;
  - `accepted`, and `repeat` (the same document ID again, which is idempotent and changes nothing, since the current entitlement is computed).
- **Computed on read.** Each accepted, uncompromised entitlement has an effective end: its own end, cut to `max(own Start, S.Start)` by the earliest accepted, uncompromised entitlement S of the same subject and installation that supersedes its agreement and was accepted after it. The register keeps an acceptance sequence for this.
  - An older entitlement never cuts a newer one, so chains under the same agreement (A, A, A) and migrations back (A to B to A) keep the newest right effective and the DB-14 check intact.
  - The installation's current entitlement is computed on read: the uncompromised accepted one whose effective interval contains authoritative now, with the latest `IssuedAt`.
  - If none contains now, it is the latest one, which shows as expired.
- **Acceptance.** A document is accepted only when it verifies, is valid at authoritative now, and passes the rights check.
  - While the current entitlement is still active, the new one must also have a later `IssuedAt`; otherwise it is `superseded`. A replay of an older but still valid entitlement therefore cannot roll back the current one.
  - After the current entitlement has expired, any valid one is accepted.
- **Rights (DB-14).** The check covers every accepted, uncompromised entitlement in the register with the same `SubjectID` and `Installation`, on any channel. One whose effective interval overlaps the new one refuses it with `duplicate_right`.
  - An overlap is allowed only when the new document's `Supersedes` names that entitlement's agreement, for a renewal or a migration.
  - `Supersedes` must name an agreement already accepted for the same `SubjectID` and `Installation`. Otherwise it is refused with `bad_supersedes`, which includes a first-ever document that names its own agreement. A renewal under the same agreement ID sets `Supersedes` to its own `Agreement`.
  - Overlap checks use effective ends, so renewal chains work and two paid rights are never open at once.
  - `bad_supersedes`, `superseded` and `duplicate_right` are decided only after verification steps 1 to 7.
  - The same subject and installation therefore never hold two paid rights at once, whether direct, marketplace or mixed.
- **Separation.** Registration never touches authentication, tenant authorisation or billing amounts. This module has no path to them.

## Status

`Status(installation)` reads authoritative time and returns:

| State | When |
| --- | --- |
| `unselected` | no mode has been selected yet |
| `noncommercial` | mode is `noncommercial`. There is no countdown and no expiry warning |
| `evaluation_pending` | mode is `evaluation` with no activation |
| `evaluation_active` | the activation interval contains now |
| `evaluation_expired` | now is at or after the end |
| `commercial_pending` | mode is `commercial` with no current entitlement |
| `commercial_active` | the current entitlement's interval contains now |
| `commercial_expired` | now is at or after the current entitlement's effective end |
| `unverified` | a time-dependent mode, with authoritative time unavailable (`time_unavailable`), earlier than the high-water mark (`clock_anomaly`; the mark is never below any activation start, so a time before the start is also caught here), or ahead of the local clock (`local_clock_behind`), or with the governing key compromised (`key_compromised`). The last applies too when every entitlement that would be current is compromised |

The view also carries:
- the verified details: subject ID, subject, agreement, scope, term, channel and metric;
- the remaining time for an active evaluation;
- the last submission outcome;
- the last verified state with its timestamp, shown for `unverified` and never as fresh;
- `Warning` (true for the expired states and for `unverified`) and the fixed actions `register` and `contact`.

`Status` changes only the following, and `Activate` and `Register` change them in addition to their own writes: the authoritative-time high-water mark, the expiry latches, and the installation's last verified state.

No call in this module blocks, locks, sends, deletes or denies anything. Callers must not use `Status` to switch product features on or off. Historical access and export are outside this module and unaffected.

## Canonical encoding

- **Fields.** Each field is length-prefixed with an 8-byte big-endian length. A length must not exceed the remaining bytes.
- **Integers.** Times are canonical decimal UTC Unix seconds: no sign, no leading zeros, and from 0 to 253,402,300,799. Counts are canonical decimal too.
- **Booleans.** `1` or `0`.
- **Order.** `hybrid-billing/license-claims/v1`, then:
  - `Kind`, `ID`, `Issuer`, `KeyID`, `IssuedAt`;
  - `SubjectID`, `Subject`, `Agreement`, `Supersedes`;
  - `Product`, `Installation`;
  - the scope count, and each scope entry;
  - `Channel`, `Metric`, `Start`, `End`, `Perpetual`, `Terms`, `ActivateBy`.

  An empty time is the empty string.
- **Strictness.** A body that does not decode exactly is `invalid`.

Known-answer tests are cross-checked against an independent Python encoder. Signature tests use Go's `crypto/ed25519` with fixed seeds, and are cross-checked with Python's `cryptography` package where it is available.

## Bounds

| Bound | Limit |
| --- | --- |
| Active keys | 100, with tombstones kept separately |
| Installations | 10,000 |
| Mode and submission history kept per installation | the last 100 entries each |
| Activated grant IDs and accepted entitlements | 100,000 each (`ErrLimit`). Running out is a stated residual, with no eviction, so that a grant can never be activated twice |
| Body | 4,096 bytes |
| Identifiers | 1–128 bytes of `[A-Za-z0-9._:-]` |
| Subject and agreement text | 1–200 bytes of UTF-8, with no control characters |

## Residuals

- Restoring or replacing the register itself, by an operator who controls code and storage, cannot be detected.
- Key issuance, custody and signing, online verification and revocation lists. Small-order key rejection.
- Evaluation extensions, which need reconciliation.
- Checking actual usage against each entitlement's contracted metric (AUD-DB-14 usage half).
- Exhausting the grant and entitlement caps.
- Registering a renewal before its `Start`, as a pending successor. Until then, a renewal waits for its `Start`.
- After normal retirement, the old key can still backdate documents into its signing window until `AcceptUntil`. A leak is handled by `compromise_key`, and shortening `AcceptUntil` is a residual.
- Approvers who also control the Authority can misreport time before an expiry is first observed.
- The OSS candidate register (increment 2).
- Marketplace sales activation remains a later product scope. Only the DB-14 model checks are delivered.
- Persistence, screens, and wiring `Status` into the UI with the contact action.
- LM-01 to LM-11 at system level and AUD-DB-14 are NOT_RUN beyond these unit tests.
- Owner decisions not confirmed by the owner:
  - the actual evaluation terms;
  - the issuer;
  - the metric set;
  - the bounds;
  - the algorithm choice (ed25519).

# Verification plan

Literal tests:
- **Forms and bounds.** Every form rule and bound.
- **Keys.** The private-key length, missing or wrong proof, rotation, `retire_past`, compromise affecting recorded rights, and validity at the edges.
- **Verification.** Each step of the order:
  - an altered `IssuedAt` is `invalid`, not `unverifiable`;
  - trailing bytes, overlong lengths and non-canonical decimals;
  - an unknown key, the wrong issuer, a future `IssuedAt`, a backdated document signed after retirement, `wrong_kind` and `scope_mismatch`.
- **LM scenarios.** LM-01 to LM-11, each as a named test:
  - evaluation at `end − 1ns` and at `end`;
  - a repeated activation;
  - time unavailable;
  - a valid key not replaced by an invalid one;
  - expired against unavailable;
  - clock rollback after expiry, and now before start;
  - a compromise after a `Status` read restoring the genuine entitlement, and an only-compromised entitlement reading `unverified` with `key_compromised`;
  - `already_evaluated`, `stale_grant` and `superseded` replay;
  - a privacy-safe submission history.
- **DB-14.** Overlap across channels and on the same channel, `Supersedes` renewals, adjacent half-open intervals, and a perpetual term overlapping everything later.
- **Concurrency.** Concurrent calls under `-race`.

Property test: random keys, documents (valid, altered, unknown key, mismatched), activations and authority outages. An independent model of states and outcomes must match.

Checks: `-race -count=10`, the 386 build, gofmt, vet, `go test -race ./...`, 100% statement coverage and mutation testing.

# Review history

- Round 1 (independent reviewer, read-only): FAIL. Findings and what changed:
  - H1.
    - Keys must be current at authoritative now.
    - `IssuedAt` must not be in the future.
    - `retire_past` is refused.
    - `compromise_key` invalidates recorded rights on read.
  - H2. An authoritative-time high-water mark, `clock_anomaly`, and a stored last verified state.
  - M1. A fixed verification order and the full outcome list.
  - M2. One evaluation per installation (`already_evaluated`) and `ActivateBy`.
  - M3. Replacement only by a later `IssuedAt` (`superseded`), plus `not_yet_valid`, and idempotent repeats.
  - M4. A signed `SubjectID`, overlap refused on every channel unless `Supersedes` names the agreement, and half-open intervals. Metric usage is a residual.
  - M5. Proof of possession for keys. Small-order keys are a residual.
  - L1. `wrong_kind`.
  - L2. Encoding strictness.
  - L3. Submission history keeps only the outcome, time and digest.
  - L4. Key tombstones, documented caps, and human actors on every mutating call.
  - Residual note. `Status` must not switch features.
- Round 2 (same reviewer): FAIL. Findings and what changed:
  - M1. Compromised entitlements are excluded from comparisons, with a fallback.
  - M2. Separate signing and acceptance windows, and `Status` re-checks compromise only.
  - M3. A 24-hour ahead-of-clock anomaly, and a dual-controlled `reset_high_water`.
  - M4. Effective ends for superseded rights, same-agreement renewals, and `bad_supersedes`.
  - M5. `superseded` applies only while the current entitlement is active.
  - L1. The proof message is length-prefixed.
  - L2. Time is step 4 of the verification order.
  - L3. Recorded as a residual.
- Round 3 (same reviewer): FAIL. Findings and what changed:
  - N1. Effective ends and the current entitlement are computed on read from uncompromised entitlements, so a compromise immediately restores the genuine one.
  - N2. Latched expiry, and `reset_too_early`.
  - L1. The local clock's role is stated, with `local_clock_behind`.
  - L2. `max(own Start, S.Start)`, first-ever `bad_supersedes`, and rule checks only after verification.
  - L3. Recorded as a residual.
- Round 4 (same reviewer): FAIL. Findings and what changed:
  - P1. Latches use only the record's own end.
  - L1. Which calls latch is stated.
  - L2. `local_clock_behind` is listed.
  - L3. An only-compromised entitlement reads `unverified` with `key_compromised`.
  - L4. Effective end.
  - L5. The reset scope is stated.
- Round 5 (same reviewer): PASS. The optional wording fix is folded in.

# Change review resolution

The change review (independent reviewer, read-only) returned FAIL. Resolved:
- H1. Supersession now cuts only entitlements accepted earlier, by an acceptance sequence. The plan text is amended. Tests cover a same-agreement chain (A, A, A) and a migration back (A to B to A). Reverting the fix makes those tests fail.
- M1. The property model now uses an independent acceptance-order rule, reuses agreement IDs and random `Supersedes` targets, and must reach `bad_supersedes`. Activations, resets, retirement and `local_clock_behind` are covered by literal tests rather than the property model; this is stated.
- L1. Activation start is truncated to whole seconds.
- L2. `local_clock_behind` is listed as an outcome.
- L3. Retiring again is `retired`.
- L4. The `Authority` contract is stated: called under the lock, it must be fast and must not call back.
- L5. The submission digest is length-prefixed.
- Mutation survivors from the first run were closed with boundary tests that use literal values, so that the tests no longer simply repeat the constants.

The re-review (same reviewer, read-only) returned PASS with no regression.
