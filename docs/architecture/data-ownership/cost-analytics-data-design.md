---
type: Data Architecture Proposal
title: Cost analysis data ownership and fact design
description: Proposes transactional, analytical and semi-structured data responsibilities for more than one hundred million incoming records per day.
status: draft
---

# Cost analysis data design

Tracking: [HB-20](https://linear.app/hybrid-billing/issue/c04558aa-62af-4914-a71e-785049be5fcb). Date: 2026-10-08. Accountable owner: Seung Woo Park. Root owns synthesis and documentation; a separate read-only DBA reviews design. Machine: `machine:mac_mini` during execution, released at handoff. Base: `e436a49`.

Owner direction: retain original Hybrid Billing scope, defer HawkEye/TokenMeter integration, require **more than 100 million incoming records/day**, and give structural layers explicit purposes. These instructions supersede earlier 1 million/day sizing premises for this review. They do not authorize implementation, establish an upper bound, or select an exact NoSQL/SQL product. The prior Kafka/Spark/Airflow discussion is a provisional architecture, not measured adoption evidence. This document supplements the [logical ownership model](billing-data-model.md); it does not finalize DDL or pricing/accounting policy.

## Review position and storage authority

Propose a hybrid data model: relational business authority, column-oriented analytical facts, and protected semi-structured evidence. ClickHouse is a SQL analytical database, not the proposed NoSQL half of a binary SQL/NoSQL split. Variable provider attributes alone do not justify another database.

| Store | Owns or serves | Exclusions and consistency |
| --- | --- | --- |
| PostgreSQL candidate | Customer relationships, effective policy/mapping revisions, manifest publication, calculation state, approvals, economic Claims, documents, submission/receipt metadata and durable intents | Not the default location for all incoming detail. Per-economic-scope uniqueness/reservation structures still need measured capacity; metadata can also become large. |
| Protected object storage | Permitted original bytes; immutable normalized/calculation partitions and manifest artifacts; issued document bytes | Durable evidence, not a query authorization system. Complete bytes must be verified before publication. File/columnar formats and snapshot protocol remain to be selected. |
| ClickHouse candidate | Rebuildable, version-addressed cost/usage/allocation/charge projections, curated dimensions and analytical aggregates | Not financial approval or uniqueness authority. No direct client access; current permission before aggregation and output release. |
| Bounded JSON/Map fields | Source-specific extensions, approved resource attributes and tags not yet promoted to canonical dimensions | Core identity, monetary values, units, basis and time are typed fields. No secrets/unbounded payload copies in analytical extensions. |
| Dedicated document/NoSQL database | Conditional only: independently justified document retrieval/update workload with its own owner, versioning and lifecycle | Not adopted. Require demonstrated access patterns or latency/scale needs unmet by objects, bounded relational metadata and analytical projections before adding synchronization/backup burden. |

Kafka is transport/replay within a configured horizon; Spark is computation; Airflow coordinates finite workflows. None is the business source of truth. Redis/search engines are not assumed prerequisites. Document database, table format, broker, compute and scheduler choices remain reviewed candidates, not a full mandatory product bundle.

## Facts: define grain before tables

Do not force every source row into a single universal cost record, and do not repeat full usage quantity on every monetary component. One usage event may have several charges, one charge may span many measurements, and a subscription/commitment charge may have no measured usage.

| Proposed logical fact | Grain and identity | Measures and relationship |
| --- | --- | --- |
| UsageFact | One canonical source measurement slice, revision, meter/unit and interval | Quantity only; lineage to original bytes and source coverage. Retain cumulative/event source semantics to avoid double counting. |
| SupplierCostFact | One canonical supplier economic charge component, source revision, currency and amount basis | Supplier charge/credit/fee/tax component; optional links to usage through a bounded contribution relation. Source identity is not a delivery/Kafka key. |
| AllocationFact | One approved allocation-run revision, source pool/component, target and allocation basis | Allocated amount/quantity and rule revision; targets plus explicit residual/unallocated amount reconcile to the eligible pool. |
| CustomerChargeFact | One calculation-run revision, customer/contract, charge component, coverage and currency | Selling components, discounts and taxes with policy/engine/input revisions; may link to multiple source contributions. Issued application remains separately authoritative. |
| RecognitionFact | One approved monthly recognition component, entity/books, period and accounting basis | Cost recognition, separate from selling amount, invoice cadence and cash. Policy mapping remains open. |
| ReceiptAllocationFact | One confirmed receipt allocation or linked reversal to a document, currency and as-of revision | Cash/receivable evidence; never joined to usage to sum cash at arbitrary resource grain without an explicit allocation policy. |
| PeriodicAggregate | One dataset release, period, permitted dimensional scope, currency, basis and measure definition | Derived exact sums/counts at declared grain. Never mixes publications or pretends stale/partial coverage is complete. |

Use contribution links for drill-through, not unconstrained fact-to-fact joins. Pre-aggregate compatible facts to a common declared grain before comparison; test one-to-many and many-to-many cases. Do not duplicate quantity when a provider emits charge, discount and tax lines for one measurement. Missing quantities/costs are unknown or not applicable, not zero.

## Canonical dimensions and temporal identity

Typed shared dimensions: installation/workspace, customer relationship, legal issuer/buyer, contract, provider/source account, service/SKU, resource, region, organization/project/cost center, meter/unit, currency, charge category, time and cost basis. Keep applicability and unknown/unmapped states explicit. Supplier account, customer, payer and security tenant are distinct.

Every owned reference carries validated scope. Workspace identity comes from verified source configuration or authenticated context, never the incoming customer ID alone. Bind canonical IDs to explicit provider namespace/account and effective mapping revision; names and email are display attributes.

Preserve source event/service interval, received/recorded time, business-effective interval, approval/issuance and external-confirmation time. Dimension/mapping history records both effective and recorded time. Bill calculations pin the exact mappings used. Retrospective remapping creates a new analyzed revision without changing issued history.

Expose two explicitly named analytical views: **as billed/as attributed then**, using fixed historical revisions, and **restated to a selected mapping**, using an explicit mapping release. A live dimension join must not silently rewrite last month's cost center. Current authorization applies even to historical facts; historical ownership is evidence, not permission.

Tags are multi-valued relationships. An expense tagged to two teams is not automatically two expenses. For ordinary filtering, use membership/existence semantics and include the canonical fact once. A tag breakdown is labeled non-additive across overlapping tags unless an approved exclusive grouping or weighted allocation rule supplies conserving shares. Frequently filtered/grouped keys become typed curated dimensions after semantic review; arbitrary new tags stay bounded extensions.

## Cost analysis contracts

| Question | Required basis and safeguards |
| --- | --- |
| Who spent what, where and when? | Fixed release + authorized customer/org/account/service/resource + usage period + supplier cost basis/currency. Preserve unmapped records as visible exceptions within scope. |
| Supplier cost versus customer charge/margin | Compatible customer, economic coverage, currency conversion revision and declared net/tax basis. Pre-aggregate both sides; unmatched portions remain explicit, and original/converted amounts cannot both contribute to a total. Margin excludes separately identified tax under the selected policy. |
| Allocation and shared infrastructure | Original supplier pool and allocated target view are alternative bases, not additive layers. Include unallocated/residual amounts and deterministic rounding reconciliation. |
| Cost changes | Compare equivalent coverage/basis/mapping and completeness; distinguish quantity, unit price, FX, allocation and late-correction effects. A rate average is quantity-weighted only for compatible units and meaning. |
| Credits, commitments and amortization | Separate billed/cash charge from effective/amortized analytical cost. Link benefits/adjustments to original scope; define recognition horizon and policy before calculating amortized views. No inferred universal amortization schedule. |
| Current versus historical reporting | Explicit dataset release, as-of, mapping revision, source completeness and provisional/final status. Late corrections create a new release; the prior issued release remains addressable. |

Measure metadata specifies grain, unit/currency, amount basis, eligible charge categories, tax treatment, aggregation and permitted dimensions. Define separately additive amounts, period balances, rates and distinct counts; do not sum balances across snapshots or average averages. Use bounded exact decimal rules for authoritative money, never floating-point accumulation. Select precision, intermediate bounds, rounding level and residual policy from the reviewed numeric contract; a universal Decimal precision/scale is not selected here.

## Source-to-analysis and financial publication

1. Register permitted receipt identity, account, source mode (event/cumulative/snapshot), interval and digest; preserve immutable bytes and quarantine contradictory identities.
2. Normalize/classify against pinned schema and mapping revisions. Establish complete source coverage and required independent/operator checks before publishing eligible input.
3. Write and verify detail partitions. Publish a manifest with partition versions, digests, counts and typed control totals by compatible currency/unit/basis; no mixed total.
4. Calculate with fixed input/policy/engine revisions. Publish result manifests only after coverage, conservation and duplicate-economic-scope checks.
5. Build ClickHouse facts and aggregates for a new immutable analytical release. Verify partition coverage/counts/control totals, correction handling and allowed field lineage. Mark the release ready only when all required projections are available; queries pin one release for their entire result/export. Failed loads remain non-current and retries cannot append a second visible contribution.
6. Financial commands revalidate current authority, approval digest and economic Claim coverage in the transactional authority. Publish durable external intent with the local financial transition. ClickHouse readiness or aggregate totals do not authorize issuance.

Release publication is a coordinated visibility protocol, not a transaction across PostgreSQL, object storage and ClickHouse. Stage first, validate, then switch the authoritative release pointer conditionally. Readers must check both readiness and required retained partitions; missing storage or incomplete replicas block complete/final results. Recovery never rolls back canonical source, calculation or financial eligibility. An older verified analytical release may be served only as explicitly stale/historical with its real as-of and completeness; it must not replace a corrected release as current/final. Otherwise fail closed until the eligible projection is rebuilt. Abandoned objects remain noneligible until policy-governed cleanup. A release is a scoped manifest referencing reusable immutable partitions; correcting one slice does not require copying all retained history. Cross-scope queries pin an explicit compatible release set.

A correction uses a reviewed canonical replacement or a linked reversal/delta convention, never both in one analytical measure. Each query/release declares its convention. Incremental aggregates must remove the superseded contribution or rebuild the affected slice before publication. Raw insertion-triggered sums and eventual background deduplication are insufficient proof of billing correctness.

## Physical design hypotheses for the new scale

At 100 million/day, 30 days contains 3 billion incoming records before normalization, components, allocation fan-out, corrections, projections and replicas. More than 100 million/day is a lower design condition, not an upper throughput promise. Average volume is not a burst budget.

Choose ClickHouse sorting keys from dominant scoped period/detail/drill-through queries. Compare coarse time partitions and scoped sort/shard strategies against hot tenants, skew, retention and replacement costs. Do not partition separately for every customer, resource or arbitrary tag. Keep very high-cardinality dimensions searchable without producing every possible dimensional aggregate.

Use bounded batch files/parts, partition manifests and independent ingestion, compute, interactive-query and backfill resource budgets. Avoid a PostgreSQL row/transaction for every raw event by default, while retaining exact identity/economic coverage validation in the detail pipeline and financial claim mechanism. The scalable identity/dedup/Claim index is an explicit unresolved design obligation, not assumed cheap metadata.

Model retained bytes as incoming records times measured bytes, then separately account for normalization/component expansion, indexes, compression, replication, temporary working space, retained revisions and backups. Select hot/warm/archive horizons by workload and retention obligations; no arbitrary duration is adopted. Object evidence and transactional control backups must restore consistently; latest deletion/revocation controls and writer fencing apply before replay or access resumes.

## Acceptance before physical selection

All runtime checks below are NOT_RUN. Independent design review is distinct from database/runtime proof.

- One usage measurement with multiple charge/tax/discount components retains its original quantity once; source-to-document drill-through reconciles with compatible scoped totals.
- Two overlapping tags do not double total cost. Weighted allocation conserves the original pool including rounding residual and explicitly unallocated amounts.
- Multiple invoices/receipts linked to one source do not multiply cost or cash during joins. Distinct quantities/rates/balances obey their own aggregation definitions.
- Concurrent/replayed jobs, changed delivery keys, cumulative plus event sources, late corrections and partial coverage cannot duplicate an economic charge or expose a half-published analytical release.
- Original-currency and converted views never double sum; incompatible currencies/units/bases cannot be silently combined. Decimal bounds and rounding use independent expected values.
- Moving an account/project later preserves the as-billed result, produces an explicitly different restated result and grants no historical access automatically.
- Cross-customer queries, arbitrary JSON/tag filters, counts, margins and exports cannot reveal denied fields or inferred totals; revoke permissions before an export completes.
- Sustain the measured peak and daily workload while scoped interactive queries and close/backfill run; test hot-tenant skew, high-cardinality tags, expanded allocation and correction fan-out.
- Crash each publication boundary, lose an analytical replica, replay after retention/revocation and restore old backups; preserve approved history, deletion controls and visible freshness. After a correction from 100 to 90, an older verified 100 release may appear only as explicitly historical/stale, never current/final; canonical eligibility remains at the corrected revision.

Synthetic expected examples (design oracles, not executed tests): cost 100 allocated 60/40 has source total 100 and allocated total 100, never combined 200; two descriptive tags on cost 100 still filter to total 100; quantity 10 with base 20, fee 3 and tax 2 still has quantity 10; full replacement 100 to 90 produces current 90 and historical 100, never 190. These examples select no commercial prices or tax policy.

Acceptance budgets still needed: actual peak and duration, row widths, provider profiles, fan-out, query shapes/concurrency/latency, close/catch-up windows, retention, RPO/RTO and operating capacity. No node count, sharding key or latency guarantee is selected without these.

## Review sources and limits

Repository evidence: [logical model](billing-data-model.md), [technology assessment](../technology-assessment.md), [detail lifecycle](../usage-detail-lifecycle.md), [pricing/ERP contract](../../contracts/billing/inputs-pricing-and-erp.md), [access contract](../../contracts/security/access-and-trust.md). Existing owner-directed local policies and skills were read in the authorized main workspace; they are not assumed present in the isolated checkout.

Product capability references, checked 2026-10-08: [PostgreSQL JSON types](https://www.postgresql.org/docs/17/datatype-json.html) supports bounded relational document extensions; [ClickHouse JSON design](https://clickhouse.com/blog/a-new-powerful-json-data-type-for-clickhouse) motivates typed and bounded dynamic attributes; [ClickHouse merge correctness](https://clickhouse.com/resources/engineering/clickhouse-optimize-table-final) distinguishes eventual replacement from immediate query correctness. Exact versions and compatibility remain unselected. These references support capabilities, not this proposal's benchmark or correctness certification.

Independent review and publication evidence: [HB-20 report](../../reviews/HB-20-cost-data/verification.md). No DDL, migration, product implementation or new accounting rule is delivered. Planning remains open.
