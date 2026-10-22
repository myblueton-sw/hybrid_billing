---
type: Architecture Design
title: Platform data build design for the first FOCUS path
description: Proposed build-level storage roles, intake, quarantine and re-request, release publication, ordering, tenant isolation, numeric and missing-value rules, lifecycle and local topology for the first FinOps FOCUS path using PostgreSQL, ClickHouse and S3-compatible object storage.
status: draft
---

# Platform data build design (revision 4)

Tracking: Linear ticket [HB-26](https://linear.app/hybrid-billing/issue/HB-26). Owner: Seung Woo Park. Machine: `mac_mini`. Date: 2026-10-22. Planning completion: [decision record](../planning/planning-completion.md). Binding constraints: [current design decisions](current-design-decisions.md). This draft reflects the owner's direction on 2026-10-22 and two external architecture reviews each of revisions 1 to 3. It is not yet an accepted decision.

## Owner direction captured

- ClickHouse is the analytical store and the single bulk processing and query engine. DuckDB is not used.
- The product is a FOCUS **consumer**. The official FOCUS validator is not embedded and the product does not certify provider conformance. It records the declared FOCUS version and checks only what it consumes.
- FOCUS 1.2 and 1.3 are supported from the start through one internal model based on the latest supported version; 1.4 is added when a provider exports it.
- Checks run in two stages: before collection (single file) and before publication (cross-file). There is no separate ClickHouse staging database.
- A file that is received but unusable is quarantined and re-requested from its source. Only content that must not be accepted at all is rejected without storage.
- Parquet remains the object-storage data foundation: originals stay in their received format, and every sealed release is also exported as Parquet to object storage for long-term retention and ClickHouse rebuild.
- OpenTelemetry carries platform observability. Metered usage may arrive over OTLP but follows the full metering contract below.

## Authority and storage roles

| Store | Holds | Authority |
| --- | --- | --- |
| PostgreSQL 18 | Tenants, customers, contracts, pricing policy, connections, receipts, quarantine and re-request state, release manifests and publication log, charge lines, approvals, credits, audit, job queue | Single authority for financial state, publication, ordering and scope |
| ClickHouse 26.8 LTS (26.3 LTS acceptable) | Cost and usage facts loaded per release, aggregates and read models; a separate database for platform telemetry | Reproducible evidence and projection only; never an approval or publication authority |
| S3-compatible object storage (local: SeaweedFS 4.42 or later; production: customer-operated S3-compatible service certified by our tests) | Accepted originals exactly as received, quarantined originals in a restricted prefix, release Parquet exports, ClickHouse backups, and the control journal | Immutable evidence; content-addressed, SHA-256 recorded in PostgreSQL |

MinIO community edition is not used (archived, no maintained binaries). The product uses only the generic S3 API. ClickHouse and SeaweedFS are Apache-2.0.

## Intake

### Pre-collection checks (Go intake gateway, single file)

| Outcome | Cases | Handling |
| --- | --- | --- |
| Reject, not stored | Authentication failure, unregistered or expired connection; data outside the connection's allowed accounts, periods or customers; prohibited content types | Rejection record only: receipt ID, connection, reason code, size, SHA-256, policy and profile revisions |
| Quarantine and re-request | Size, SHA-256 or provider manifest mismatch; interrupted transfer; a column that the declared FOCUS version requires is absent; missing or unsupported declared FOCUS version; same receipt key and same source revision with a different digest, or a different digest without revision metadata, unless the receipt is an authorized quarantine replacement (below) | Original stored under the restricted quarantine prefix with its own retention; never loaded for serving or billing |
| Configuration error | Money columns typed as binary float, or another format the connection profile does not accept | Quarantined once and reported to the connection owner to change the export format or profile (for example CSV); not re-requested, because re-fetching returns the same format |
| Correction candidate | Same receipt key with a newer source revision | Stored as a new revision linked to its predecessor; processed into a superseding release after the usual checks and confirmation |
| Duplicate | Same receipt key, same source revision and same digest | Not stored again; duplicate-receipt record |
| Accept | All checks pass | Original stored; receipt moves to `stored` |

Receipt key: connection, provider account scope, dataset, declared billing or charge period and provider file or export identity. Source revision: the provider's revision, execution or export version identity. Deduplication uses receipt key, source revision and digest, never digest alone. Excess source scale is detected when the Python worker computes manifest control sums exactly from the original; the receipt is quarantined with that reason.

### Quarantine and re-request

State: `quarantined → re_requesting → replacement_received → resolved`, or `→ escalated` when attempts or deadline are exhausted. Three paths stay distinct: a transport retry returns the same source revision and must match the expected digest; a correction carries a newer source revision; a quarantine replacement is authorized by the open re-request record, which names the quarantined receipt, its source revision and the predecessor digest; a receipt matching an open re-request is exempt from the same-revision conflict rule, closes that re-request, and supersedes the quarantined receipt. Each path repeats every other pre-collection check and confirmation.

- Pull connections: the collector re-fetches the same source object with bounded attempts and backoff, then escalates to the connection owner. Re-fetch follows the fetch policy ED-03.
- Push connections: the sender receives a re-send request with receipt ID, reason code and missing items, and a response deadline; then escalation. Requests go only to the connection's approved destination; this design grants no new messaging permission.
- The affected scope stays held while unresolved. A replacement records that it supersedes the quarantined receipt. Quarantined originals are deleted only through the same retention, hold, pin and fencing protocol as other evidence; reaching the end of retention makes them eligible, not deleted. Operator confirmation of scope and evidence follows the existing input confirmation contract (`input.confirm_publish`).

### FOCUS versions

One internal fact model follows the latest supported version. Each declared version has a mapping: columns absent in that version are `not provided` (NULL with reason), never zero; deprecated or renamed columns map explicitly (FOCUS 1.3 lists `ProviderName` and `PublisherName` as deprecated and adds `HostProviderName`; the exact mapping to `ServiceProviderName`, `HostProviderName` and `InvoiceIssuerName` is confirmed against the official changelogs before use). A column that the declared version does not define is `not provided`; only a column the declared version requires and the file lacks is quarantined. Provider support observed on 2026-10-22: AWS 1.2 generally available, Azure 1.2 preview, Google Cloud 1.2 as a BigQuery dataset in preview (needs a pull connector from BigQuery rather than files). Which providers export 1.3 is not yet sourced. Fixture planning is governed by the [HB-27 demo dataset baseline](../validation/plans/HB-27-focus-demo-baseline.md): first inspect the exact FOCUS demo artifact and its actual provider/service coverage. The previously identified CC BY 4.0 FOCUS 1.0 sample is a candidate requiring provenance, license and version-compatibility checks, not evidence of an inspected or supported fixture. Label synthetic 1.2/1.3 examples, tenant mappings and contracts separately from observed demo data; external-connector validation is a separate evidence class.

## Release publication

1. PostgreSQL creates release `R` in state `loading` with its manifest: source receipts and digests, mapping and loader revisions, ClickHouse and engine versions, expected row count and per-currency control sums computed by the Python worker from the originals, and the predecessor release it supersedes, if any.
2. The worker claims the load with an attempt number recorded in PostgreSQL and loads with `INSERT ... SELECT FROM s3(named_collection, ...)` into immutable MergeTree tables, tagging every row with `release_id` and `attempt`. Only rows of the attempt recorded as current in PostgreSQL count; rows of stale attempts are ignored by verification and readers and deleted later. Before a retry, rows of earlier attempts are deleted synchronously (`lightweight_deletes_sync`), and the worker verifies that none remain. Fact tables do not use projections.
3. The worker verifies ClickHouse counts and control sums for `R` against the manifest. Mismatch marks `R` `failed`.
4. Pre-publication checks run over `R`: period and account coverage, reconciliation with supplied invoice totals, unmapped accounts, missing-value counts and applicability per consumed column, and correction detection. A release covers one connection, dataset and period; for shared-payer files the customer attribution scopes inside it are held or published separately, and a failed check holds only the affected attribution scopes.
5. Operator confirmation follows `input.confirm_publish` and is recorded against the exact manifest digest.
6. Publication is one PostgreSQL transaction that first re-checks, inside the transaction, that the confirmation is valid for the manifest digest, the confirming actor still holds authority, dependency pins and policy revisions are current, and no rows of stale attempts of `R` remain in ClickHouse (checked immediately before the transaction and recorded as part of the decision); then compare-and-swap of the current-release pointer for the scope against the expected predecessor and the scope's revision counter, increment of that counter, the immutable publication decision, an outbox record for the Parquet export and the control journal, and `R` becomes `sealed`. No rows are appended to a sealed release.
7. The sealed release is exported to Parquet in object storage with its digest recorded. The export uses Parquet DECIMAL types that preserve declared precision, scale and nullability; it inherits the retention origin, tenant and attribution scope of its release and sits under a prefix restricted like the facts. Before use as a rebuild source its digest and manifest control sums are verified. After the originals expire it supports only a ClickHouse rebuild of the same release, not re-derivation under new mappings.
8. Orphaned releases (`loading` past timeout, or unknown outcome) are reconciled before cleanup; their rows are deleted only after reconciliation.

Readers resolve the current release set for their scope from PostgreSQL once per request or export, pin it, and pass the explicit `(release_id, attempt)` pairs into ClickHouse queries, so rows of stale attempts are never read even if deletion is still pending. No query uses "latest" in ClickHouse. Concurrent corrections for the same scope are serialized by the compare-and-swap. A rebase is allowed only when the losing release carries a newer source revision than the winner; otherwise it fails. Releases with incomparable revisions go to operator review.

## Ordering for replay and restore

- Receipt identity: PostgreSQL-generated record identifiers; they identify receipts but do not define commit order.
- Correction order: provider revision metadata and explicit supersedes links; arrival order never decides which revision is current.
- Publication order: the per-scope revision counter updated inside the compare-and-swap. Global PostgreSQL sequences can have gaps and do not follow commit order, so they serve only as record identifiers.
- Time: event time (FOCUS `ChargePeriodStart`/`ChargePeriodEnd`, OTLP event time) partitions billing periods; receipt and publication timestamps are records, not ordering authority. All times are UTC with the source timezone retained.

Control journal: every hold, deletion, revocation, retention decision and publication is also recorded in an append-only control journal in object storage, one object per entry, created with conditional no-overwrite writes and object versioning or lock, excluded from retention deletion and never overwritten by restore. Safety-relevant controls (holds, revocations, deletion denials, retention decisions) are acknowledged to the caller and take effect only after their journal entry is durable; the PostgreSQL commit records the journal entry key, and a follow-up commit marker closes the entry. On restore, an entry without a commit marker is re-applied only when it restricts access or deletion; an uncommitted entry that would permit removal goes to reconciliation instead. Publications reach the journal through the outbox, because a lost publication entry only makes a release reload from its predecessor state, which is then reconciled. A point-in-time recovery of PostgreSQL alone would roll controls back, so the journal is the independent source of the latest controls. Conditional-create support is part of the object-store certification tests.

Restore: fence all executors; recover PostgreSQL to a point in time; read the control journal beyond the restore point and re-apply its holds, deletions, revocations and retention decisions, failing closed if the journal cannot be read up to its latest watermark; drop ClickHouse releases absent from the publication log; reload missing published releases in publication order from their Parquet exports, or from originals with the pinned loader, mapping and engine revisions; mark releases whose sources have expired as not rebuildable; reconcile external effects after the restore point (ERP submissions, issuance, Claims) under the outcome-unknown rules; verify rebuilt releases against their manifests before reopening. Loads are idempotent per release.

## ClickHouse rules

- **Money.** Each money column has a declared `Decimal(P,S)` within Decimal64 or Decimal128 and a documented maximum row count per release so the widest possible sum stays within an independently checked bound. ClickHouse performs grouping and sums only. Division, `round` and casts of money columns are prohibited in ClickHouse queries. Every total used for pricing is recomputed and matched in Python `decimal` before approval. Totals are grouped by currency, unit and economic basis. Excess scale in the source and Float-typed money are quarantined as described under intake.
- **Missing values.** Consumed columns that can be absent are `Nullable`. The loader, worker and reader profiles fix, as server-side constraints, `input_format_null_as_default = 0`, `input_format_csv_empty_as_default = 0`, `input_format_parquet_allow_missing_columns = 0` and `join_use_nulls = 1`. Aggregates are accompanied by missing-value counts, so a sum alone never establishes completeness.
- **Tenant isolation.** The Go API is the access authority: it derives scope on the server and passes explicit scope and release filters. ClickHouse row policies are defense in depth: restrictive policies on scope columns, `throw_on_unmatched_row_policies` enabled through the server-level `access_control_improvements` configuration, `serialize_query_plan = 0`, scope settings locked as constants. Separate users for loader, maintenance and reader; readers cannot alter tables, copy partitions or call `s3()`, `url()` or remote functions. Shared-payer data is restricted before aggregation. Cross-tenant negative tests are mandatory.
- **Partitioning.** Partition by charge period (month), with scope where retention differs; `release_id` is in the order key, not the partition key. Retention deletion is ordered through the policy server, which owns retention, against holds, pins and readers; mixed-retention partitions use verified rewrites. Physical removal of dropped partitions is delayed and tracked until complete, including detached parts, backups and release Parquet exports.
- **Engines.** Immutable MergeTree. `ReplacingMergeTree` is never used for deduplication or publication.
- **Credentials.** S3 access uses named collections or server configuration; no inline credentials in SQL or logs.
- **Backup.** ClickHouse `BACKUP` to the object store, or rebuild from release Parquet exports or originals with pinned revisions; restore follows the procedure above.

## OTLP metered usage

The received OTLP protobuf bytes are the original and are stored as received. Billable usage is accepted only as integer data points or as log records whose quantity is a decimal string attribute, each with `source_event_id`, `source_revision`, resource identity, unit and explicit delta or cumulative temporality. Counter resets, overlap, gaps and late arrival follow coverage windows: before close they are included; after close they become corrections. The gateway repeats the customer collector's checks. The OpenTelemetry ClickHouse exporter (metrics alpha, logs and traces beta) is used only for platform telemetry, never for billable usage.

## Local Docker Compose

`postgres:18`, `clickhouse` 26.8 LTS, `seaweedfs` 4.42+ (S3 path-style, bucket init), `otel-collector` (contrib), `migrate` (one-shot), `api` (Go, includes intake gateway), `policy` (Go), `worker` (Python), `web` (React). Each service pins its image, declares volumes, health and readiness checks, resource limits and secrets through environment files excluded from Git. No Kafka, table catalog service or Spark. Production and HA topology remain unselected.

## Supersession and changed assignments

On acceptance this design supersedes the HB-20 preference for Iceberg with Spark in [object storage data architecture](data-ownership/object-storage-data-architecture.md) and the "engine unselected" statements in [current design decisions](current-design-decisions.md). It also moves single-file intake checks to the Go gateway and pre-publication checks to ClickHouse SQL driven by the Python worker, where the decision register assigned the verification path to Python. Those documents are updated only after acceptance.

## Detailed in the physical design step

The external reviews also asked for detail that belongs to the per-dataset physical design and procedures, not this overview: exact precision, scale and quantity bounds; field-level disclosure, metadata and error leakage and export re-authorization; OTLP deduplication scope, conflicting revisions, cumulative-to-delta conversion, reset identity and overlap resolution; the irreversible deletion boundary, writer fencing and unknown-outcome reconciliation; policy server acknowledgment and outage behavior; image pins and compatibility tests. Each is specified there and reviewed before implementation. Implementation of the affected paths does not start until the physical design review accepts, in particular, the irreversible deletion boundary, writer fencing (synchronous deletion of stale attempts does not stop an in-flight writer) and policy-server outage behavior required by the [current design decisions](current-design-decisions.md).

## Open items before implementation

- Physical schemas per dataset: exact precision and scale, order keys, nullable columns, and the consumed FOCUS 1.2 and 1.3 column list with per-function restrictions.
- Behavior of `sum` at the Decimal64 to Decimal128 boundary and text parsing with excess scale in ClickHouse 26.8: to be tested, not assumed.
- Re-request attempt limits, deadlines and quarantine retention: profile values.
- Google Cloud BigQuery pull connector design.
