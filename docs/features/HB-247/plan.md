---
type: ImplementationPlan
title: Migration import of read-only historical documents with manifest-verified batches, idempotent re-import, quarantined conflicts, numbering and delivery fences, and no secret key material
description: Second increment of work item W-37. A register imports past documents from dual-controlled source systems in batches. Each batch is bound to the exported manifest's item count and digest, sealed, verified and accepted under dual control. Accepted documents become immutable read-only history that can never be issued, delivered, submitted or changed. Re-importing is idempotent by source identity, and changed content is quarantined, never overwritten. Imported numbers and deliveries become high-water marks and delivery fences. Only parsed public keys and certificates are accepted, never secret key material.
status: stable
---

# Scope and ownership

[HB-247](https://linear.app/hybrid-billing/issue/HB-247) continues work item W-37 after HB-246. It covers these parts of HB-37-I and HB-37-V:
- "migration imports past documents as read-only history and does not move secret keys";
- the HB-37-V case "migration re-import": detecting and blocking external re-issue, double delivery and evidence loss.

Opening balances (QA-09) are covered by HB-216, and the recovery core by HB-246.

Base: main `e360c86`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the design and the change, because imported history sets numbering and delivery fences and crosses tenant boundaries. Machine: mac_mini.

Allowed files: `modules/migration/*` and `docs/features/HB-247/*`.

# Observed baseline

- **HB-37-I.** "Migration brings past documents in as read history and does not move secret keys."
- **HB-37-V.** Migration re-import must not cause external re-issue, double delivery or evidence loss.
- **Recovery fences.** `modules/recovery` (HB-246) provides numbering and delivery fences for recovery. Migration needs the same kind of fence for imported history. Feeding these fences into `modules/recovery`, or into the consumers, is a residual. This module exposes its own reads.
- **Issuance.** Series in `modules/issuance` are per workspace, and numbers are `int64`.
- **Sibling conventions.** The sibling conventions apply: proposals, dual control, actors, bounds.

# Decision

A new package `modules/migration` follows the sibling conventions:
- one mutex and an injected clock that never goes backward;
- `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (with `Refusal{Reason}`) and `ErrLimit`;
- check order: form, lookup, repeat, rule, room;
- assistants refused as `ai_not_allowed` and system actors as `not_human` where a human is required;
- numbered proposals approved by a second human with a recheck, withdrawal by the proposer, caps per scope and per proposer, and bounded history.

It imports only the standard library, including `crypto/x509` for key references. It is constructed with `NewRegistry(Config{Evidence}, clock)`. `Evidence` is a port that answers `Present(tenant, digest) (bool, error)` against `modules/evidence`.

Every fence, history entry and natural key is keyed by tenant.

## Sources, importers and series mappings

| Proposal kind | Payload | Effect |
| --- | --- | --- |
| `source` | `Source{ID, Tenant, Importer, Workspaces}` | Registers a source. `Workspaces` is a strictly ascending list of 1–64 workspaces. The first source to list a workspace claims it for its tenant. A source of another tenant that lists a claimed workspace is refused (`workspace_claimed`). Binding workspaces in `modules/auth` remains a hard exposure blocker |
| `importer` | `Source{ID, Importer}` | Rotates the importer identity. Old identities are tombstoned and never reassigned |
| `mapping` | `Mapping{Source, SourceSeries, Workspace, Series}` | Maps a source series to the issuance series of a workspace, which must be one of the source's workspaces. A mapping exists at most once per `(source, SourceSeries, Workspace)` and is never replaced (`ErrConflict`). Every document's source series must be mapped before its batch can be accepted (`unmapped_series`). Checking that the issuance series belongs to the tenant is part of the `modules/auth` binding residual |
| `batch` | `BatchSpec{Source, ID, Count, Digest}` | Opens a batch |
| `accept` | `Accept{Source, Batch, Effect}` | Accepts a sealed batch |

## Batches

- **Opening.** A batch ID is never reused. At most one batch per source may be unfinished, meaning open or sealed but not yet accepted or rejected (`batch_open`).
  - `Count` is 1–100,000.
  - `Digest` comes from the source system's export manifest.
  - At most 1,000 batch records are kept per source; at that point new batches are refused (`ErrLimit`), never evicted.
  - Batch records keep summaries only, never their items, once they are finished.
- **Adding items.** `Add(source, batch, Item, actor)` by the source's importer, only while the batch is open.
  - An item is a `Document` or a `KeyReference`.
  - An identical repeat is idempotent.
  - A different item with the same source ID is `ErrConflict`.
  - At most `Count` items.
- **Sealing.** `Seal(source, batch, actor)` by the importer, in two steps:
  1. Under the lock, it computes the batch digest (below). A batch whose count or digest differs from the manifest becomes `mismatch`, which is final.
  2. Otherwise, it releases the lock and asks `Evidence.Present` for every document's `Content`. It then re-enters the lock and requires the batch to be unchanged.
     - A missing digest makes the batch `evidence_missing`, which is final, so history without evidence is never accepted.
     - A dependency failure refuses the seal (`dependency_unavailable`) and keeps the batch open.

     Evidence is checked once, at seal. `modules/evidence` retention and legal holds protect it afterwards; deleting it between seal and accept is outside this register.

  The batch is otherwise `verified`.
- **Rejecting.** `Reject(source, batch, reason)` by one human finishes an unfinished batch as `rejected` and retires its pending accept proposals. Rejected and mismatched records keep their digests and counts.

## Classification and acceptance

- **Classification.** Classification runs against current history whenever an accept is proposed or approved. Each item is classified as:
  - `new`;
  - `duplicate`: the same tenant, source and source ID, with identical canonical bytes;
  - `conflict`: the same tenant, source and source ID, with different bytes;
  - `number_conflict`: another history document with the same tenant, workspace and either `Display` or mapped `(series, number)`;
  - `external_conflict`: another history document with the same tenant, workspace and `ExternalKey`.

  Natural-key conflicts catch re-keyed exports and the same document exported from two sources. Items are classified in source-ID order against history plus the earlier items of the same batch, so the first of two colliding items in a batch is `new` and later ones are quarantined.
- **Effect.** The acceptance effect is the SHA-256 over `hybrid-billing/migration-effect/v1` and, length-prefixed:
  - each item's source ID, class, incoming digest and stored digest (an empty value when there is none);
  - the resulting fence changes:
    - the new maximum for each touched `(workspace, series)`;
    - the number of new display fences;
    - the number of new delivery fences.

  `Accept.Effect` must equal the recomputed effect at both proposal and approval (`effect_changed`), so the approvers sign off the exact history, quarantine and fence changes.
- **Atomic acceptance.** Acceptance first checks room for history, quarantine, display fences, series fences and delivery fences, per tenant and in total. It refuses with `ErrLimit` and changes nothing if any would overflow. Otherwise it:
  - appends the `new` documents to history;
  - quarantines every conflict of any kind, with both digests, never overwriting. An exact repeat of an existing quarantine entry (same source ID, class, incoming and stored digest) adds nothing and needs no room;
  - applies fences from every item in the batch, whatever its class.

## Fences

For every document in an accepted batch, including duplicates and conflicts:

- **Numbering.**
  - Raises the high-water mark of `(tenant, workspace, mapped series)` to at least `Number`.
  - Fences `(tenant, workspace, Display)` as a printed number that issuance must never print again.
- **Delivery.** If the stored or incoming item is delivered or submitted, it fences `(tenant, workspace, "source:" + source + ":" + SourceID)` and, when an external key exists on either item, `(tenant, workspace, "external:" + ExternalKey)`.

Fences only grow.

- **Internal keys.** Delivery and display fences are stored under bounded keys made of recovery-valid identifier characters:
  - `src-` plus the first 32 hex characters of SHA-256(tenant, source, SourceID);
  - `ext-` plus the first 32 hex characters of SHA-256(tenant, ExternalKey);
  - `dsp-` plus the first 32 hex characters of SHA-256(tenant, Display).

  Every input is length-prefixed.
- **Reads.**
  - `HighWater(tenant, workspace, series)`;
  - `PrintedTaken(tenant, workspace, display)`;
  - `DeliveryFenced(tenant, workspace, source, sourceID)`;
  - `ExternalFenced(tenant, workspace, externalKey)`.
- **Export.** `Fences(tenant, workspace)` exports `Export{HighWater []{Series, Number}, Keys []string, Printed []string}` using those keys. A test checks every exported key against a copy of the `modules/recovery` identifier rule, so the recovery register and the consumers can take them over without loss. The module does not import `modules/recovery`.

These remain exposure blockers for serving imported tenants:
- issuance consulting high-water marks and printed numbers, and refusing a series whose prefix and number space overlap imported printed numbers below the high-water mark;
- delivery and erpadapters consulting the delivery fences;
- the migration register being included in recovery manifests and reconciliation.

## Documents

| Field | Rule |
| --- | --- |
| `SourceID` | Identifier, unique per source |
| `Kind` | `invoice`, `credit_note`, `debit_note` or `receipt` |
| `Workspace` | One of the source's workspaces (`workspace_not_allowed`) |
| `SourceSeries`, `Number` | Identifier, and an integer from 1 to 10^15 |
| `Display` | The printed number, 1–64 bytes of printable ASCII |
| `IssuedAt` | From 1970 up to now inclusive |
| `Currency` | Three uppercase letters |
| `Total` | `int64` in minor units, from 0 to 10^15. The sign follows `Kind` |
| `Content` | SHA-256 hex of the source rendering, which must be present in evidence |
| `Delivered`, `Submitted` | Booleans |
| `ExternalKey` | Identifier, required when `Submitted` |

- **No mutation path.** Accepted documents are `historical`. No operation changes, issues, delivers or submits them, and reads return copies.
- **Secret material heuristic.** Text fields containing `PRIVATE` or `BEGIN` in any letter case are refused (`secret_material`). This is a heuristic only. Text fields are restricted to identifier characters or printable ASCII without line breaks.
- **Content.** A 64-hex `Content` cannot be told apart from a raw secret seed. The evidence-presence check at seal ensures it names stored evidence.

## Key references

- **Form.** `KeyReference{SourceID, Kind, DER}` carries a public key as an X.509 certificate (`certificate`) or a PKIX SubjectPublicKeyInfo (`public_key`). Both are at most 16 KiB.
- **Parsing.** The register parses the DER with `x509.ParseCertificate` or `x509.ParsePKIXPublicKey`. Bytes that parse as a PKCS#8, PKCS#1 or EC private key are refused (`secret_material`), as is anything that does not parse (`ErrInvalid`).
- **Stored form.** It stores the public bytes and its own SHA-256 fingerprint. No raw key bytes or bare digests are accepted.

## Canonical encoding

Every value is a length-prefixed field: an 8-byte big-endian length, then the bytes. Integers are canonical decimal, with no leading zeros, no plus sign, and `0` for zero. Booleans are `1` or `0`. Times are UTC Unix nanoseconds in canonical decimal.

- **Batch digest.** It covers `hybrid-billing/migration-batch/v1`, the tenant, source, batch ID and count, then the items sorted bytewise by source ID.
- **Document item.** `document`, then `SourceID`, `Kind`, `Workspace`, `SourceSeries`, `Number`, `Display`, `IssuedAt`, `Currency`, `Total`, `Content`, `Delivered`, `Submitted`, `ExternalKey`.
- **Key item.** `key`, then `SourceID`, `Kind`, the fingerprint.

Known-answer tests pin a mixed batch, an effect and each fence-key prefix (`src-`, `ext-`, `dsp-`), each cross-checked independently.

## Reads

Every read takes the tenant, and an unknown or foreign source is `ErrNotFound`:
- `Document(tenant, source, sourceID)`;
- `Key(tenant, source, sourceID)`;
- `Batch(tenant, source, batch)`, which returns the state, counts per class and digests, and for a verified batch the current effect digest and fence changes, so approvers can see what they approve;
- `Quarantine(tenant, source)`.

## Bounds

| Limit | Maximum |
| --- | --- |
| Sources | 10,000 |
| Workspaces per source | 64 |
| Items per batch | 100,000 |
| Unfinished batches per source | 1 |
| Batch records per source | 1,000 (summaries) |
| History documents | 1,000,000 per tenant and 10,000,000 in total |
| Quarantine entries | 100,000 per source |
| Series fences | 10,000 per tenant and 100,000 in total |
| Display fences | 1,000,000 per tenant and 10,000,000 in total |
| Delivery fences | 2,000,000 per tenant and 20,000,000 in total |
| Key references | 10,000 per tenant |
| Pending proposals | 1,000 per scope |

## Residuals

- Source adapters and exporters that implement the canonical encoding.
- Evidence artifact transfer into `modules/evidence`.
- Binding workspaces to tenants in `modules/auth`.
- Consumers and the recovery register taking over the fences (exposure blockers, above).
- Whether migration operations should pause while recovery is locked.
- Workspace claims are first come, first served: one tenant can pre-claim another tenant's workspace ID, which blocks only migration for it until the auth binding exists. This is known and recorded.
- Persistence.
- Screens.

# Verification plan

Literal tests:
- **Form.**
  - Every form rule, including document kinds, currency, display text, `ExternalKey` when submitted, number and total caps, and `IssuedAt` bounds.
  - Key references: a certificate, a public key, a private key refused, and garbage refused.
  - Secret material in any letter case.
- **Proposals and roles.** Proposal rules, importer binding, and workspace allow-lists.
- **Batch lifecycle.** Open, add, seal, verified and mismatch, accept and reject, one open batch per source, and immutability after sealing.
- **Within a batch.** Idempotent repeats and conflicts.
- **Across batches.**
  - Re-importing the same export gives only duplicates and no history change.
  - Changed content gives a conflict, quarantined with both digests.
  - Acceptance requires the effect digest.
- **Fences.**
  - Fences from every class, including a conflict that differs only in `Delivered`, `Submitted` or `ExternalKey`.
  - Display fences, and the issuance prefix collision case.
  - High-water marks through series mappings.
  - Unmapped series.
  - Growth only.
- **Tenants.** Workspaces claimed by one tenant, reads across tenants, and `Number` at its cap.
- **Natural keys and approval.** Natural-key conflicts across source IDs and sources. Atomic acceptance at every cap. A stale effect after history changes.
- **Round-2 additions.**
  - A remap is refused.
  - A natural-key collision inside one batch.
  - Every exported key passes the recovery identifier rule.
  - A non-canonical decimal changes the digest.
  - The `Batch` read shows the effect.
- **Evidence and records.** Evidence missing and dependency failure at seal. Batch records retained, IDs never reused.
- **Read-only history.** No mutation path exists, and reads return copies.
- **Bounds and isolation.** Every bound, and isolation between sources and tenants.

A known-answer test pins the batch digest, cross-checked independently.

Property test: random sequences of batches over a pool of documents with random changes, run against an independent model of history, quarantine and fences. Invariants:
- history entries never change once accepted;
- fences only grow;
- re-importing an accepted export never changes history;
- every delivered or submitted item seen in an accepted batch is fenced;
- no two history documents share a natural key.

Checks:
- `-race -count=10`, a 386 build, gofmt and vet;
- the whole repository with `-race`;
- 100% coverage and mutation testing.

# Review history
- Round 1 (independent reviewer, read-only): FAIL. Findings and what changed:
  - H1. Fences come from every class.
  - H2. Printed-number fences and series mappings.
  - H3. Tenant-keyed fences, workspace claims, a number cap, and an effect digest.
  - M1. Natural-key conflicts.
  - M2. Atomic acceptance with per-tenant caps.
  - M3. DER key references parsed by the register.
  - M4. The canonical encoding is specified.
  - M5. Reclassification with the effect digest, and one unfinished batch.
  - M6. An evidence-presence port.
  - M7. A fence export, with exposure blockers.
  - L1. Namespaced keys.
  - L2. Batch records kept.
  - L3. Tenant-scoped reads.
  - L4. Bounds, and importer rotation.
- Round 2 (same reviewer): FAIL. Findings and what changed:
  - N1. Mappings are immutable and checked against the source's workspaces.
  - N2. Classification also checks earlier items in the batch.
  - N3. Bounded hashed keys, a separate printed-number list, and a validity test against a copy of the recovery rule.
  - L-a. Canonical decimals.
  - L-b. Evidence retention stated.
  - L-c. The `Batch` read shows the effect.
  - L-d. Recorded.
  - L-e. Wording fixed. The issuance prefix case is tested through `PrintedTaken`.
- Round 3 (same reviewer): PASS. Low items folded in:
  - R3-1. Quarantine entries are deduplicated.
  - R3-2. Known-answer tests for the fence keys.

# Change review resolution

The change review (independent reviewer, read-only) found no path to re-issue, double delivery, evidence loss or a cross-tenant leak. It returned FAIL with M1–M2 and lows. Resolved:
- M1. Mismatched and evidence-missing batches drop their items, like rejected and accepted ones.
- M2. The property model now covers:
  - two sources of one tenant, so natural-key conflicts cross sources;
  - printed numbers varied independently of numbers;
  - rejects;
  - a quarantine model with deduplication, compared by size;
  - exact equality of the exported delivery and printed fence sets and the high-water mark.
- L1. The secret heuristic also checks `Workspace` and `SourceSeries`.
- L2. `Proposal(number)` and the fence reads reveal tenant existence to any caller. This is recorded against the `modules/auth` binding residual.
- L3. Acceptance retires sibling accept proposals. Retired proposals are deleted and read as `ErrNotFound`, as in the sibling modules.
- L4. Rejection records the actor and the reason in the batch view.
- L5. `Add` runs the form check, then the lookup, then the secret and workspace rules.
- L6. Plan names aligned with the code: `Key` and `BatchSpec`.
- L7. Tests added for fence state after an atomic refusal, the issuance-style printed-number collision, and exact-fit caps for every bound.
- L8. The known-answer values were cross-checked with an independent Python encoder (recorded in verification.md).
- L9. Seal rechecks the importer after the evidence call.

The re-review (same reviewer, read-only) found one regression from the L3 fix: the applied accept proposal was itself retired, because it was still pending during `commit`. `Approve` now drops the proposal from pending before applying it, and retirement matches only `accept` kinds. A test confirms that the applied record and an idempotent repeat approval survive.
