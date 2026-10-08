---
type: Architecture Research
title: Performance, maintainability and operations comparison
description: Compares object-based cost analytics options and defines evidence required before selecting a production topology.
status: draft
---

# Performance, maintainability and operations

Tracking: [HB-20](https://linear.app/hybrid-billing/issue/c04558aa-62af-4914-a71e-785049be5fcb). Research date: 2026-10-08. This supplements the [object storage design](data-ownership/object-storage-data-architecture.md). Original Hybrid Billing scope applies; HawkEye and TokenMeter integration is deferred. MongoDB is excluded. More than 100 million incoming source records per day is a requirement, not a measured result. No installation, engine benchmark or production test was performed.

## Recommendation and decision boundary

Evaluate object storage, Parquet, Iceberg/catalog, bounded Spark batch processing and governed lake SQL as the common foundation. PostgreSQL owns transactional business state and financial decisions. Iceberg is the preferred evaluation hypothesis, not an adopted or benchmark-proven winner. Add selective ClickHouse serving only if representative authorized queries miss agreed latency/concurrency budgets and the measured benefit pays for its extra lifecycle. Do not require every named product simultaneously.

| Topology | Performance opportunity | Maintenance and operational cost | Selection condition |
|---|---|---|---|
| A: lake foundation plus Trino as reference SQL reader | Column/partition/file pruning; shared historical data without another complete copy | Catalog, files, snapshots, SQL capacity, compaction, writer/reader compatibility | Historical and batch workloads dominate and interactive budgets are met |
| B: A plus selective native ClickHouse serving | Sorted native parts and curated aggregates for repeated hot queries | Additional writes, merges, replicas, projection lag, correction refresh, parity and rebuild | Same queries show sufficient measured benefit at acceptable whole-system cost |
| C: lake foundation with ClickHouse lake reading and native serving | Potentially one analytical reader family for lake and hot data | Connector/version limitations, metadata/delete support, exact-decimal and authorization compatibility remain gates | Can satisfy the same historical, snapshot, precision and access contracts; compare before removing a separate lake reader |

These are engineering inferences from product capabilities. No relative latency or throughput has been demonstrated. A serving outage must permit an explicitly degraded lake route where budgets allow; a lake outage may permit a previously verified serving release with honest freshness and current authorization. Neither route may turn stale evidence into current financial eligibility.

## File and table format choice

Parquet is a columnar file format, not an application transaction, catalog or secondary-index service. Its capabilities also depend on the chosen reader. [Parquet overview](https://parquet.apache.org/docs/overview/).

| Option | Why consider it | What the team must maintain |
|---|---|---|
| Plain Parquet and custom manifests | Transparent immutable batch artifacts, fewer product dependencies | Commit protocol, discovery, concurrent writers, schema evolution, retained references, cleanup and recovery; fewer products can mean more proprietary infrastructure code |
| Iceberg | Snapshots and table metadata for independently operated processing/query engines | Catalog, metadata/file growth, compaction, retained snapshot protection and a tested feature compatibility profile |
| Delta | Credible alternative for a Spark-centered ecosystem | Protocol compatibility, transaction-log/checkpoint maintenance, optimization and safe vacuum across actual readers |
| Hudi | Worth evaluating for dominant keyed-update and incremental-consumption workloads | Indexes, locking, table services and compaction; copy-on-write rewrites base files, merge-on-read transfers work to reads/compaction |

Sources: [Iceberg reliability](https://iceberg.apache.org/docs/latest/reliability/), [Iceberg maintenance](https://iceberg.apache.org/docs/latest/maintenance/), [Delta FAQ](https://docs.delta.io/delta-faq/), [Hudi table types](https://hudi.apache.org/docs/table_types/). These explain mechanisms, not workload winners. Start correction experiments with bounded copy-on-write slices; compare merge-on-read only with a supported engine matrix. Record rewritten bytes and query read amplification. Table commits do not atomically publish PostgreSQL business state.

## Catalog and object storage operations

Two catalog candidates are sufficient for the initial comparison:

| Candidate | Fit and benefit, as inference | Continuing duties |
|---|---|---|
| Iceberg JDBC catalog with isolated PostgreSQL | Smaller service footprint for trusted internal engines | Atomic database transactions, isolation, drivers, credentials, connection budget, HA, schema upgrades and catalog/object restore |
| Apache Polaris REST catalog with PostgreSQL persistence | Central catalog interface and governance across multiple engines | REST replicas, TLS, shared signing-key lifecycle, principals/realms, storage delegation, durable database persistence, HA and upgrade/restore |

[JDBC catalog documentation](https://iceberg.apache.org/docs/latest/jdbc/) specifies underlying database transaction requirements. [Polaris production configuration](https://polaris.apache.org/releases/1.7.0/configuration/configuring-polaris-for-production/) documents persistent storage and authentication configuration: per-replica generated signing keys and in-memory persistence are inappropriate assumptions for a replicated production deployment. The linked Polaris version is a research reference, not a selected release. Catalog authorization does not substitute for customer row/field authorization. Sharing PostgreSQL technology does not make catalog and financial commits atomic.

Prefer testing Polaris for the multi-engine governance requirement and retain JDBC as the simpler trusted-engine comparison. Add Nessie only if a concrete catalog branching requirement justifies it. Prefer an existing customer-operated, supported object service; introducing a new distributed storage cluster solely for billing adds a separate storage-operating responsibility. Managed object services are candidates only where delivery/data-residency policy permits. S3 API compatibility alone does not establish consistency, lifecycle, security or recovery compatibility.

## Performance design that matters more than product names

- Preserve raw source objects once; convert accepted facts into typed, denormalized Parquet datasets with explicit grain, amounts, currencies, cost basis, time and bounded extensions. Promote frequently filtered fields consistently; arbitrary JSON/tag keys are not automatically indexed.
- Select partitions, sort order, row groups and file size from measured query predicates, tenant skew and correction windows. Excessively fine tenant/time partitions create small-file and metadata overhead. Do not prescribe a universal partition count or megabyte target before measurement.
- Prune columns and files for lake queries; keep lineage lookups separately addressable and bounded. Large historical exports run asynchronously with authorization, exact values and progress checkpoints.
- Precompute recurring aggregates only with a named semantic grain and reproducible release. Fan-out joins must not multiply quantities or money. Do not preaggregate away evidence needed for corrections or audit.
- Financial claims, source identities and calculation pools retain their exact domains. Parallel chunks cannot independently consume one allowance or approve overlapping economic claims. PostgreSQL capacity must include claim/index growth, locks, WAL and correction contention; calling it metadata does not make it small.
- Batch files enter durable object storage and a receipt/manifest queue. Kafka is justified by a streaming/replay requirement, not by the daily row count alone. Spark handles normalization and full-domain calculation work; an orchestrator coordinates bounded jobs. Evaluate streaming engines only when a defined latency requirement needs them.

[Trino Iceberg connector](https://trino.io/docs/current/connector/iceberg.html) describes supported lake operations. [Trino caching](https://trino.io/docs/current/object-storage/file-system-cache.html) and [dynamic filtering](https://trino.io/docs/current/admin/dynamic-filtering.html) document mechanisms whose benefit depends on plans, connectors and workloads. [ClickHouse lake practices](https://clickhouse.com/docs/guides/use-cases/data-warehousing/getting-started/best-practices) distinguish direct lake access and metadata/file considerations. Direct lake access does not itself require a native full copy.

## Named operating responsibilities

These are required responsibility boundaries, not separate headcount commitments.

| Responsibility | Signals and routine work | Recovery/maintenance requirement |
|---|---|---|
| Storage operator | Requests, bytes, failures, lifecycle rules, capacity, credentials | Coherent metadata/data retention and restore; no independent TTL that removes pinned evidence |
| Data platform operator | Small files, manifests, snapshot pins, commit conflicts, failed/unknown jobs, compaction backlog | Fenced writers, bounded maintenance, reconcile unknown commits before orphan cleanup |
| Catalog/database operator | Connections, latency, locks, WAL, replication, certificates and signing keys | HA and restore catalog, business references and independent control state coherently |
| Processing operator | Arrival lag, skew, shuffle/spill, retry amplification, pool completion | Checkpoint/replay and correction runbooks; ingestion, closing and backfill resource isolation |
| Query/serving operator | Queue delay, planning time, scanned bytes, p95/p99, native merge backlog and release lag | Admission control, parity, rebuild and explicit degraded routes |
| Security/release owner | Current revocations, row/field filters, query functions, cached results, compatibility profile | Revocation and restore fences; upgrade rehearsal against retained snapshots |

One binary or service fewer is useful only if it does not move correctness and operations into custom code. Keep a pinned writer/catalog/reader/format feature matrix. Rollback must respect metadata/protocol changes; reinstalling an old binary is not sufficient evidence of safe rollback.

## Comparative acceptance profiles

All profiles below are NOT_RUN. Specify volume ceiling, compressed arrival windows, row width, cardinality/skew, corrections, retention, concurrent users, freshness, close deadlines, catch-up time, RPO/RTO and operator/cost budgets before assigning PASS. A measured result with undefined acceptance thresholds remains decision-pending.

100 million/day averages about 1,157 records/s; the same volume arriving in one hour requires about 27,778/s, or 166,667/s in ten minutes before retries and derived facts. Thirty such days contain 3 billion source records. These are arithmetic scenarios, not a sufficient capacity plan; the actual condition is above that daily minimum.

| Profile | Required observation |
|---|---|
| BM01 Semantic parity | Same cost bases, tags, M:N lineage and independently computed exact answers across all routes |
| BM02 Sustained scale | Proposed starting scenario: 120M/day for 72h and >3B retained rows; realistic widths, independent IDs, skew, all source dispositions and processing watermarks |
| BM03 Arrival bursts | One-hour and ten-minute arrivals, backpressure, durable receipt, fairness and catch-up |
| BM04 Interactive mix | Day/month aggregates, top resources/services, tag filters and joins under concurrency; p50/p95/p99, queue and planning time, scanned files/bytes |
| BM05 Cache fairness | Cold/cold and warm/warm; disclose result, engine, OS and object caches and any layer that cannot be cleared |
| BM06 Historical evidence | Sparse old lineage lookups and bounded unknown-ID fallback with unchanged authorization |
| BM07 Skew and full pools | Large pools and hot tenants coexist with small tenants; allowance/tier/rounding denominator remains complete |
| BM08 Contention | Ingestion, monthly close, backfill, exports, compaction and cleanup together |
| BM09 Corrections | Late replacement/delta, replay, schema changes and projection lag; absent-row deletion only for proven complete replacement scope |
| BM10 Publication failures | Crash at object, table, PostgreSQL and serving boundaries; unknown commits reconciled without double append or partial publication |
| BM11 Snapshot lifecycle | Active reader, legal hold, writer and cleanup races; pinned references survive |
| BM12 Access boundaries | Forbidden customers/fields, metadata/catalog/functions/object paths, revocation and cached-result leakage |
| BM13 Exports | Long XLSX/CSV output, splitting, exact values/IDs, progress, cold retrieval and mid-job revocation |
| BM14 Compatibility | Retained snapshots, field add/rename/widen, deletes, partition evolution and failed upgrades; rollback limitations |
| BM15 Recovery | Mismatched catalog/objects/PG restore, latest deny/tombstone fences, external ERP uncertainty and serving rebuild; measured RPO/RTO |
| BM16 Day-two effort | Catalog outage, credential/certificate rotation, full spill disk, poison file, stuck job and failed cleanup; operator time and escalations |
| BM17 Whole-system cost | CPU/RAM, replicas, history/backups, storage and requests, network/egress, shuffle/spill, compaction, projections, catalog and operator hours |

Use both fixed-resource comparisons and SLO-sized total-cost comparisons. A preaggregated ClickHouse result versus an unoptimized full scan is not equivalent unless supported query restrictions and aggregate build/refresh costs are included. Synthetic load must not merely repeat identical compressible rows or replace raw events with monthly summaries. Independently verify numerical answers and account for accepted, duplicate, quarantined and rejected records. No vendor benchmark certifies this installation.

## Decision gates

The current recommendation is lake-first, with conditional hot serving. Engine/version selection, physical schemas, partitions/indexes, exact source-identity strategy, claim-authority capacity and budgets remain open. Advance only with relevant independent design review and the owner's planning gate. This research does not authorize implementation or represent performance acceptance.
