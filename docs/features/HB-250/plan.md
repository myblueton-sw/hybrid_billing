---
type: ImplementationPlan
title: Staged dataset versions with bounded idempotent chunks, two-stage exact aggregation, atomic publication and snapshot-pinned cursor pagination
description: First increment of work item W-39. A register stages a dataset version against an expected row count and digest. Chunks are bounded and idempotent by index, so an interrupted load resumes without duplicates or gaps. Per-chunk partial totals are merged exactly at seal. A sealed version is published atomically while a bounded number of earlier versions stay readable. Rows and totals are read through snapshot-pinned, key-ordered cursors with bounded pages.
status: stable
---

# Scope and ownership

[HB-250](https://linear.app/hybrid-billing/issue/HB-250) is the first increment of work item W-39 (requirement HB-39). It covers these HB-39-I building blocks:
- bounded ingestion;
- snapshot pagination;
- two-stage aggregation;
- checkpoint resume;
- atomic publication;
- aggregate projection.

It applies the rule that no path may let memory, transactions or HTTP payloads grow with the total data size.

It does not cover:
- **Fair queues and leases with checkpoints.** These exist in `modules/jobs` (HB-180) and are not repeated.
- **Performance measurement, SLA proposals and the P0 scale gates.** These come in later increments.
- **Generators**, which are a W-38 residual.
- **Persistence and screens.**

Base: main `40afc7b`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the design and the change, because the register computes billing totals and holds tenant data. Machine: mac_mini.

Allowed files: `modules/publication/*` and `docs/features/HB-250/*`.

# Observed baseline

- **HB-39.** The design must assume sustained inflow above 1,000,000 rows a day and more than 30,000,000 rows a month, and 1,000,000 is not an upper bound. Individual bulk jobs keep 100,000 rows as the minimum test size. The proposed direction is bounded ingestion, snapshot pagination, two-stage aggregation and distribution, checkpoint resume, atomic publication, fair queues and aggregate projections. No path may let memory, transactions or HTTP payloads grow without bound with total size. Acceptance is these P0 gates:
  - end-to-end runs of at least 100,000 rows;
  - sustained inflow above 1,000,000 a day;
  - a monthly close over 30,000,000 rows;
  - interruption and resume;
  - catch-up.

  SLA numbers are proposals until approved together with hardware and real distributions, and NOT_RUN is never shown as passing.
- **`modules/jobs` (HB-180).** Jobs have leases with fencing, checkpoints, retries, and per-type and per-workspace caps. Selection picks the least-leased workspace, ties broken by the oldest grant.
- **Siblings.** `packages/primitives/decimal` gives exact decimal arithmetic under a profile, with overflow refused. `modules/overview` has versioned as-of snapshots with cursors for status projections, but not for dataset rows.
- **Open owner decisions.**
  - what a source row means;
  - the daily inflow ceiling and the peak arrival window;
  - retention;
  - concurrent closes and the close window;
  - RPO and RTO;
  - hardware.

# Decision

A new package `modules/publication` follows the sibling conventions:
- one mutex and an injected clock that never goes backward;
- `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (with `Refusal{Reason}`) and `ErrLimit`;
- check order: form, lookup, repeat, rule, room.

It imports only the standard library and `packages/primitives/decimal`. It is constructed with `NewRegistry(Config{Profile, Limits}, clock)`, where `Limits` sets the bounds below. Each limit has a default and a hard maximum.

The callers are system components such as importers, normalisation runs and projections. Each call names its tenant. Permission checks belong to the caller, and binding to `modules/auth` is a residual.

The register holds rows in memory. It is a reference implementation of the contracts and the bounds, not the storage engine for 30,000,000 rows. Persistence is a residual, and the scale gates are NOT_RUN.

## Datasets and versions

A dataset is `(Tenant, Dataset)`, and is created by its first `Begin`. Each version gets a sequence number in its dataset that only grows.

A dataset holds at most one unpublished version, whether staging or sealed. `Begin` is refused with `unpublished_open` while one exists.

`Begin(Spec{Tenant, Dataset, Version, Rows, Digest})` starts a staging version:
- **`Version`** is an identifier that must be bytewise greater than every version ID the dataset has used, for example a zero-padded sequence or a UTC timestamp.
  - The dataset remembers only its highest ID, at O(1) cost.
  - A version that is no longer live (aborted, expired or evicted) is forgotten.
  - Any ID at or below the highest that is not live reads as `stale`, and `Begin` refuses it with `stale`.

  An ID is therefore never reused, and remembering IDs can never run out. Loaders must use fixed-width or timestamp IDs. With an unpadded counter, `10` after `9` is refused loudly with `stale`.
  Folding expiry, abort and eviction into `stale` is accepted: a loader can only start again under a higher ID in every one of those cases.
- **`Rows`** is the expected row count, from 1 to `Limits.VersionRows`.
- **`Digest`** is the expected hex SHA-256 over the chunk digests (see Canonical encoding). It fixes how the rows are split into chunks, so a resume must send identical chunk boundaries.
- **Reservation.** `Begin` reserves `Rows` at once against `Limits.StagedRows` and `Limits.HeldRows` register-wide, and against `Limits.TenantStagedRows` and `Limits.TenantHeldRows` for the tenant.
  - If any of these is short, `Begin` is refused with `ErrLimit` before any row is accepted.
  - An admitted version can therefore always finish. Room never runs out halfway through a load, and a tenant cannot take the whole register.
  - The reservation becomes held rows at seal. It is released on abort, on expiry, and when a version is evicted or its dataset retired.
- **Deadline.** An unpublished version, whether staging or sealed, expires `Limits.StagingTTL` after its `Begin`, measured by the injected clock. An abandoned loader therefore cannot hold room or block its dataset forever.
  - **When expiry runs.** Expiry runs when the version's dataset is next used, and in a sweep at every `Begin`.
  - **Queue entries.** Each entry is small (dataset, sequence, deadline) and holds no rows. Popped slots are cleared. The queue is compacted at `Begin` when it grows past twice the live unpublished versions plus 64. Released versions drop their rows, chunks and totals at once.
  - **Sweep cost.** Unpublished versions are kept in a queue in `Begin` order. The clock never goes backward and the TTL is fixed, so this is also deadline order. A sweep pops only expired entries from the front, and costs O(expired). Entries for versions already published or aborted are skipped.
  - **Expiry and `Publish`.** Every call checks expiry at its lookup step, including `Publish`, so a version sealed close to its deadline must be published before that deadline. Otherwise it is refused with `stale`.
  - **After expiry.** An expired version frees its reservation and is forgotten. Reads of it, and a `Begin` repeat of its spec, are refused with `stale`. The loader starts again under a new, higher version ID.
- **Repeats.** Repeating the same spec while it is staging returns its view and does not reserve again. A repeat for a sealed, published or retained version is `ErrConflict`. Repeats of `Seal` and `Publish` return the view without the `Repeat` flag.

`State(tenant, dataset, version)` returns the version's view:
- `State` (`staging`, `sealed`, `published` or `retained`);
- `NextIndex`, `ReceivedRows`, `Rows` and `Digest`;
- `Deadline`, `Sequence` and the times.

A loader that has lost its checkpoint resumes from `NextIndex`.

`Abort(tenant, dataset, version)` discards a staging or sealed unpublished version and frees its reservation. Its ID stays used. A published or retained version is refused with `published`.

Freeing held room on demand:
- **`Evict(tenant, dataset, version)`** evicts one retained version at once and frees its rows. Later reads are `stale`. Other versions are refused:
  - the current version with `published` (`Retire` is used instead);
  - a sealed unpublished version with `sealed` (`Abort` is used instead);
  - a staging version with `not_sealed`.
- **`Retire(tenant, dataset)`** frees every row the dataset holds:
  - its current and retained versions are evicted, and later reads of them are `stale`;
  - its unpublished version is aborted.

  The dataset record and its highest version ID stay, so an ID is never reused, even after a retire. A reader pinned to an old version can therefore never be given different content under the same ID. The dataset may begin again under higher version IDs. Dataset records are never freed and count toward the dataset limits; this is a residual.

  `Retire` of an unknown dataset is `ErrNotFound`. A retire of a dataset that holds nothing is a no-op. `Evict` of an evicted version is `stale`.

These two calls are how a tenant or an operator frees room at the held limits. Without them, a tenant whose current versions fill `TenantHeldRows` could never begin again.

## Chunks (bounded ingestion and checkpoint resume)

`AddChunk(Chunk{Tenant, Dataset, Version, Index, Rows})`:
- **`Index`** counts from 0.
- **`Rows`** is a list of 1 to `Limits.ChunkRows` `Row{Key, Group, Currency, Amount}`:
  - `Key` and `Group` are identifiers;
  - `Currency` is three uppercase letters;
  - `Amount` is decimal text that must parse with the profile and already be in canonical form, otherwise `ErrInvalid`. `1.50` is refused and `1.5` is accepted, so the caller and the register hash the same text.
- **Cost.** The ascending-order check is O(chunk). Chunks are not sorted by the register.
- **Key order.** Within a chunk, keys are strictly ascending bytewise. Across chunks, keys must be unique within the version; this is checked at seal.
- **Idempotent by index.** A repeat of an index with the same chunk digest is a no-op that reports `Repeat: true`. A repeat with a different digest is refused with `ErrConflict`. A loader that resumes from its checkpoint or from `NextIndex` and resends a chunk therefore neither duplicates nor changes rows.
- **Contiguous indexes.** An index may be at most `NextIndex`. Otherwise it is refused with `gap`, so a missing chunk cannot be skipped silently.
- **Room.** Rows beyond the version's reserved `Rows` are refused with `ErrLimit`.
- **After seal.** A chunk for a version that is no longer staging is refused with `sealed`.
- **Stage 1.** Each chunk computes, per `(Group, Currency)`, an exact partial sum as a `big.Int` coefficient at the profile's result scale. No bound is applied to partials, and no running sum can overflow. A chunk's partials are bounded by `ChunkRows` times the input maximum.

## Seal (stage 2)

`Seal(tenant, dataset, version)` first checks:
- that the row count equals `Rows`, otherwise `count_mismatch`;
- that the digest over the chunk digests equals `Digest`, otherwise `digest_mismatch`.

It then k-way merges the chunks, which are already sorted, into one key order, refusing a repeated key with `duplicate_key`. This is O(v log k) for v rows in k chunks.

Next it merges the chunk partials per `(Group, Currency)` exactly (stage 2). More than `Limits.Groups` groups is `ErrLimit`.

Finally it builds each final total once with `decimal.New` under the profile. Any error from it fails closed as `overflow`. so the result bound applies only to final totals. A total outside the bound is refused with `overflow`, whether the decimal package reports `ErrResultBound` or `ErrIntermediateBound`. The outcome therefore never depends on how the rows were split or ordered.

Recovery and repeats:
- **After a refusal.** The version stays staging. A short count can be fixed by sending the missing chunks. `digest_mismatch`, `duplicate_key` and `overflow` cannot be fixed by resending, because a chunk can never change. They end in `Abort` and a new version ID.
- **Repeats.** Repeating `Seal` on a sealed version is a no-op that returns its view.

## Atomic publication

`Publish(tenant, dataset, version)` makes the sealed version current in one step:
- **Atomic.** A reader sees either the previous current version or the new one, never a mixture, because reads are pinned to a version (see Reads).
- **Order.** Only a sealed version whose sequence is greater than the current one's can be published. There is one unpublished version at most, so publication only moves forward.
  - Publishing the current version again is a no-op.
  - A retained version is refused with `retained`.
  - A staging version is refused with `not_sealed`.
  - An evicted version is refused with `stale`.
  - Rollback to a retained version is a residual.
- **Retention.** The previous current version becomes retained. A dataset keeps at most `Limits.Retained` retained versions. The oldest is evicted, and its rows and reservation are freed.

## Reads (snapshot pagination and aggregate projection)

- **`Current(tenant, dataset)`** returns the current version's view, or `ErrNotFound` before any publication.
- **`Rows(tenant, dataset, version, after, limit)`** returns at most `limit` rows with keys bytewise strictly greater than `after`, in key order, plus a `Next` cursor.
  - `Next` is the last key returned. It is empty when the page reaches the end, including a full page that ends on the final row.
  - `limit` is from 1 to `Limits.Page`.
  - A sealed, published or retained version can be read. A staging version is refused with `not_sealed`.
- **`Totals(tenant, dataset, version, after, limit)`** pages the merged totals the same way. Its cursor is a typed pair `TotalKey{Group, Currency}`, compared bytewise first by group and then by currency.
- **Snapshots.** A reader names the version, so it pages through one fixed snapshot while later versions are published. The snapshot stays readable for `Limits.Retained` further publications. After that the version is evicted, and reads are refused with `stale`; the reader starts again from `Current`.
- **Unknown versions.** An ID above the highest used is `ErrNotFound`. Aborted, expired and evicted versions are `stale`.
- **Tenants.** Every lookup is by tenant. Another tenant's dataset with the same name is a different dataset.
- **Copies.** Every read returns copies.

## Canonical encoding

Each field is length-prefixed with an 8-byte big-endian length. Integers are canonical decimal.
- **Chunk digest.** `hybrid-billing/publication-chunk/v1`, the index, the number of rows, then each row's `Key`, `Group`, `Currency` and `Amount`, with `Amount` as the canonical text.
- **Version digest.** `hybrid-billing/publication-version/v1`, the number of chunks, then each chunk digest in index order.

Known-answer tests are cross-checked against an independent Python encoder.

## Bounds

| Limit | Default | Hard maximum |
| --- | --- | --- |
| `ChunkRows` | 10,000 | 100,000 |
| `VersionRows` | 1,000,000 | 10,000,000 |
| `StagedRows` (register-wide) | 2,000,000 | 20,000,000 |
| `TenantStagedRows` | 1,000,000 | 20,000,000 |
| `HeldRows` (staging, sealed and retained, register-wide) | 10,000,000 | 100,000,000 |
| `TenantHeldRows` | 5,000,000 | 100,000,000 |
| `Groups` per version | 100,000 | 1,000,000 |
| `Retained` versions per dataset | 2 (minimum 1) | 16 |
| `Page` | 1,000 | 10,000 |
| `TenantDatasets` | 10,000 | 100,000 |
| `Datasets` (register-wide) | 100,000 | 1,000,000 |
| `StagingTTL` | 24 hours | 30 days |

Identifiers are 1–128 bytes of `[A-Za-z0-9._:-]`.

Every call does work bounded by these limits:
- `AddChunk` is O(chunk);
- `Seal` is O(v log k);
- `Publish` is O(1), apart from evicting one version;
- a page is O(page + log v), by binary search;
- a sweep at `Begin` is O(expired versions).

No call returns more than a page.

`Seal` holds the single mutex for its whole merge, and so blocks other calls for that time. Splitting the lock per dataset is a residual.

## Residuals

- Persistence and a storage engine for the monthly volume. The 100,000, 1,000,000 a day and 30,000,000 a month gates are NOT_RUN.
- **Later increments.** Performance measurement, SLA proposals and gate records.
- **Not in this increment.**
  - distribution as the second stage of allocation;
  - rollback to a retained version;
  - catch-up scheduling, which is left to `modules/jobs`.
- **Dataset records.** Dataset records are never freed. They are capped by `TenantDatasets` and `Datasets`.
- **Seal lock time.** `Seal` holds the single mutex for its whole merge.
- **Digest at `Begin`.** Requiring the version digest before loading makes streaming sources read their data twice. A digest supplied at `Seal` is left open as a design choice.
- **Wiring.**
  - binding callers to `modules/auth` and to the job checkpoints in `modules/jobs`;
  - importers and projections using this register.
- **Owner decisions not confirmed by the owner:** the open decisions listed above, and the limit defaults.

# Verification plan

Literal tests:
- **Forms and limits.** Every form rule and every limit, at the limit and one past it, including the configured hard maximums.
- **Staging and chunks.**
  - Begin repeats, `unpublished_open`, abort.
  - Chunk idempotency and conflicts, `gap`, a resend after an interruption.
  - Backpressure at `StagedRows`.
- **Reservation and expiry.**
  - Reservation at `Begin` against all four room limits.
  - Expiry and sweeps.
  - `unpublished_open`.
  - Expiry of a sealed version, its `stale` refusals, and a sweep that pops only expired entries.
  - Ascending version IDs, including an unpadded `10` after `9`.
  - `Evict` and `Retire` freeing room for a tenant at its held limit.
  - A reader paging through a retire, then a new `Begin` that tries the old ID.
  - `Publish` after the deadline.
  - `State` and `NextIndex` resume.
- **Seal.**
  - `count_mismatch`, fixed by sending the missing chunks.
  - `digest_mismatch` and `duplicate_key` across chunks, which end in abort.
  - Final-only `overflow`, with a split that a running sum would refuse.
  - The group limit.
  - Repeated seal.
- **Publication.**
  - Atomic publication, retention and `stale`.
  - Repeats, `not_sealed` and `retained`.
  - Abort on published versions.
- **Pagination.**
  - Pages across a publish: a reader pinned to version A keeps reading A after B is published.
  - Cursor edges: an empty `after`, the last key, a key between rows.
  - Totals pagination with typed pair cursors, including groups `a` and `a-`.
- **Tenants.** Isolation between tenants.

Property test:
- Random rows split into random chunks, resent in random order with duplicates.
- Seal totals must equal an independent recomputation over all rows using `math/big` rationals. The outcome must not depend on the split, including overflow near the bound.
- Paging with random limits must return exactly the sorted rows once.

Checks: `-race -count=10`, the 386 build, gofmt, vet, `go test -race ./...`, 100% statement coverage and mutation testing.

# Review history

- Round 1 (independent reviewer, read-only): FAIL. Findings and what changed:
  - H1. Exact unbounded partials, with the result bound applied only to final totals.
  - H2. Reservation at `Begin` against register-wide and per-tenant room, a staging deadline and sweeps.
  - M1. One unpublished version per dataset, plus `HeldRows` limits.
  - M2. Monotonic sequences and `retained`.
  - M3. Typed total cursors and bytewise order.
  - M4. Recovery text corrected, and repeat and abort rules defined.
  - M5. `State` with `NextIndex`, and fixed chunk boundaries.
  - M6. Per-tenant caps.
  - L1. Chunks must arrive sorted and are k-way merged at seal. The lock time is a residual.
  - L2. Canonical amount text is required.
  - L3. Snapshot lifetime and unknown versions are stated.
  - L4. The end cursor is stated.
- Round 2 (same reviewer): FAIL. Findings and what changed:
  - N1. `Evict` and `Retire` free held room on demand.
  - N2. The deadline applies to sealed unpublished versions too.
  - N3. A register-wide `Datasets` limit and `TotalVersionIDs`.
  - N4. A deadline-ordered queue makes a sweep O(expired). `expired` is reported.
  - N5. Both decimal bounds map to `overflow`.
  - N6. Text fixes. `Retained` has a minimum of 1.
- Round 3 (same reviewer): FAIL. Findings and what changed:
  - R1. `Retire` keeps the dataset record and its IDs, so IDs are never reused.
  - R2. Resolved by R1. Queue entries also compare version identity.
  - R3. `TenantVersionIDs`.
  - R4. `Publish` checks expiry first. The repeat cases are defined.
- Round 4 (same reviewer): FAIL. Findings and what changed:
  - S1. Version IDs must ascend bytewise, so a dataset remembers only its highest ID. The ID limits were removed, and `expired` is folded into `stale`.
  - S2. A retire of an empty dataset is a no-op.
- Round 5 (same reviewer): PASS. Low items folded in:
  - T1. The ID format is stated and tested.
  - T2. Accepted and stated.
  - T3. Expiry runs at the lookup step.

# Change review resolution

The change review (independent reviewer, read-only) returned FAIL. Resolved:
- H1. Queue entries hold no rows: they are (dataset, sequence, deadline) only. Popped slots are cleared, the queue is compacted, and released versions drop their rows, chunks and totals.
- M1. A literal test has partial sums crossing the result bound with a final total inside it. The property test mixes large signed amounts.
- M2. An `audit` helper recomputes every counter from live versions. It runs after sealed aborts, expiries, evictions, retires and publications, and at every property step.
- L1. Any `decimal.New` error fails closed as `overflow`.
- L2. A test covers an old queue entry not dropping a new version of the same dataset.
- L3. The undefined behaviours are now stated. `Evict` of a sealed version reports `sealed`.

The re-review (same reviewer, read-only) returned PASS. Two test gaps, N1 and N2, were then closed: a stale queue entry behind a live blocker, and the queue length after compaction. Both target mutants were checked by hand and are killed.
