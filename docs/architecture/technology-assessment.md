---
type: Technology Assessment
title: Database streaming and telemetry candidates for self-hosted billing
description: Evaluates PostgreSQL, ClickHouse, Kafka and OpenTelemetry by authority, failure semantics and operating cost.
status: draft
---

# Technology assessment

Part of [HB-7 architecture readiness](architecture-readiness.md). Candidate suitability is a design recommendation, not adoption or runtime verification. Official documentation was checked during this review on 2026-10-01; exact deployable releases and compatibility remain unselected.

## Storage recommendation

**PostgreSQL is the first candidate for authoritative billing records.** Contracts, authorization/revocation, approved snapshots, Claims, credit reservations, document numbers, payment evidence metadata, ERP submissions and outbox need coordinated transactional changes. PostgreSQL provides relational constraints and transaction isolation; implementation must choose concurrency controls per invariant and retry failed serializable transactions safely. External calls stay outside transaction retry loops. [Constraints](https://www.postgresql.org/docs/current/ddl-constraints.html), [Isolation](https://www.postgresql.org/docs/current/transaction-iso.html).

Exact decimal storage does not choose product rounding policy. Validate finite values and supported precision before storage. Scale-constrained `numeric` can round inputs; reject unsupported input rather than relying on defaults. [Numeric types](https://www.postgresql.org/docs/current/datatype-numeric.html).

MySQL/InnoDB is a credible alternative if customer operator expertise or a supported environment favors it. Assess the same uniqueness, reservation concurrency, monetary correctness, migration and restore scenarios before choosing. [InnoDB transaction model](https://dev.mysql.com/doc/refman/8.4/en/innodb-transaction-model.html). Do not support multiple transactional engines in the first release merely because they are alternatives.

| Candidate | Proposed role | Conditions and operating burden | Assessment |
| --- | --- | --- | --- |
| PostgreSQL | Authoritative transactional metadata and business ledger | Managed schema, locks/concurrency, backup/restore, connection budgets and HA responsibility; bulk detail must not starve financial commands | First OLTP candidate; version/topology UNRESOLVED |
| ClickHouse | Versioned high-volume normalized detail and analytical projections | Compare real query patterns, tenant isolation, correct revisions/deduplication, merge/disk overhead, retention, replay and restore | Suitable analytical candidate; not selected as billing authority |
| Kafka | Decouple producers/consumers, buffer bursts and replay events | Justify consumers/replay demand; own broker/quorum/security/schema/retention/offset recovery and capacity | Conditional candidate; not mandatory from daily volume |
| OpenTelemetry | Operational traces, metrics and logs | Collector/exporter selection, redaction before emission, queue limits and observable loss; no financial authority | Appropriate observability candidate; not general billing collector |
| Immutable object/file storage | Original permitted billing evidence, manifests, calculation/detail snapshots and issued documents | Durable bytes, checksums, permissions, encryption, version retention, legal hold and tested retrieval/restore | Required logical role; product/filesystem unselected |

ClickHouse supports defined insert guarantees and documents experimental multi-statement transaction support with limitations. Those capabilities do not establish the coupled financial invariants required here. `ReplacingMergeTree` background deduplication is eventual; a plain read can include duplicates. Billing inputs require fixed manifests and explicit canonical revisions; query-time deduplication is not a substitute for a business Claim constraint. [Transactions](https://clickhouse.com/docs/concepts/features/operations/insert/transactions), [ReplacingMergeTree](https://clickhouse.com/docs/concepts/features/operations/update/replacing-merge-tree).

Kafka guarantees depend on processing boundaries. External database/ERP writes require coordination with consumed progress and application-level idempotency. A Kafka offset is not an invoice identity. Use durable outbox/inbox records, acknowledged output/checkpoint coupling, entity-scoped ordering and replay rules. [Kafka design](https://kafka.apache.org/41/design/design/). Large evidence stays in protected storage; messages carry canonical references and hashes rather than entire confidential files.

OpenTelemetry processes telemetry; Collector queues and persistent buffering reduce loss but have overflow, retry and storage failure limits. Sampled traces or unverified counters cannot become final billable usage. Approved metering requires independent stable identity, coverage, unit/time and reconciliation contracts even if OTLP is used for transport. [Collector resilience](https://opentelemetry.io/docs/collector/resiliency/), [Sampling](https://opentelemetry.io/docs/concepts/sampling/).

## Alternatives to compare after development authorization

| Profile | Composition | Question to resolve |
| --- | --- | --- |
| Lower operating burden | Transactional DB + evidence store + durable jobs/outbox; bounded batch detail queries | Can agreed arrival/query/closing/replay requirements pass without a separate broker or analytical cluster? |
| Analytical expansion | Same authority + ClickHouse projections/detail | Does scan/query isolation justify the extra storage, synchronization and operations? |
| Streaming expansion | Same authority + Kafka; ClickHouse when independently justified | Do multiple consumers, burst buffering and replay justify broker operations? |

These are comparison profiles, not production sizing or deployment support declarations. If detail lives in ClickHouse, reproducible billing uses a verified immutable detail/input revision with complete counts and totals; unsettled merges or stale replicas cannot silently enter calculation.

## Adoption evidence

Select exact versions, licenses, dependency notices, source interfaces and deployment configuration. Test duplicate delivery before/after crash, output-before-offset failure, delayed/changed revisions, replay outside the allowed horizon, schema evolution, partition skew, disk/quorum loss, backup restore and retained/deleted data. Test analytical query isolation and decimal round trips against independent expected totals. Measure operational setup/recovery effort as well as throughput. Record unsupported combinations explicitly.

Security/business audit is persisted independently of sampled telemetry. Do not emit credentials, billing bodies, original AI prompts/responses or unnecessary customer identifiers. No mandatory external exporter or model endpoint is implied by self-hosting. Customer policies govern permitted destinations. [OTel sensitive-data guidance](https://opentelemetry.io/docs/security/handling-sensitive-data/).

Current result: logical fit reviewed; benchmarks, operational exercises, database drivers, connector exactly-once behavior and production compatibility **NOT_RUN**. There is no end-to-end exactly-once billing guarantee from installing these products.
