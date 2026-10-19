---
type: Architecture Decision Register
title: Current owner decisions and initial design boundaries
description: Records the owner-confirmed implementation languages, deployment constraints, policy responsibility and data design boundaries before planning completion.
status: draft
---

# Current design decisions

Decision date: 2026-10-19. Tracking: [HB-19](https://linear.app/hybrid-billing/issue/HB-19). Accountable owner: Seung Woo Park. Machine during integration: `mac_mini`.

This register records explicit owner choices from the current design discussion. It takes precedence over conflicting technology-selection and topology hypotheses in earlier HB-19, HB-20 and HB-21 proposals. Historical diagrams, research and verification records remain evidence of their stated baselines. Integration of those proposals does not select their full technology bundles or close outstanding design findings. The owner authorized merging the pending planning/design work; planning completion and product implementation remain separate and have not been declared.

## Confirmed constraints

| Area | Current direction | Boundary |
| --- | --- | --- |
| Backend | Go | APIs, business coordination, collection control and the separate policy server. |
| Data processing and analysis | Python | Source-specific postprocessing, mart transformations, analysis and the proposed FOCUS verification path. Exact Go/Python job contracts remain design work. |
| Frontend | React | Operator/customer interfaces; permission and billing authority remain on the backend. |
| Deployment independence | Customer-operated, without mandatory cloud-provider services | Database, storage, identity, scheduling and model execution must have self-hosted paths. Provider collection connectors are external integrations, not a dependency on a cloud hosting account. Cloud independence does not by itself certify air-gapped support. |
| Local environment | Docker containers; Docker Compose is the proposed local orchestration | No cloud account required for the local fixture path. Production deployment and HA topology remain unselected. No Compose configuration or runnable product is delivered by this register. |
| Complexity | App and workers share a codebase within their language boundaries, with separate execution where needed | Prefer one initial bulk processing/query engine selected against actual work. Kafka, Iceberg, Spark together with ClickHouse, and additional orchestration services are not mandatory. Existing candidate research remains useful comparison evidence. |
| Data foundation | Object storage and Parquet | Original permitted source formats remain intact. An S3-compatible interface does not require AWS. Object service, table metadata implementation, business DB and processing engine remain selection decisions. MongoDB remains excluded. |
| File size | Directly maintained source and test files must not exceed 1,500 physical lines, including blanks | Applies to Go, Python and React sources/tests. Split by responsibility and reconsider logic before exceeding the limit; compression or arbitrary splitting is not compliance. Future review/CI must enforce this; no executable check is claimed here. |

## Processing and mart boundaries

Collection endpoints and pull adapters feed bounded preprocessing and durable collection. Source identity, receipt completion, postprocessing validation and business eligibility are separate states. No arrow from stored raw evidence to calculation permits unvalidated final billing input.

Postprocessing applies source-specific schema, identity, unit and attribution rules. A reusable canonical detail dataset may be materialized where replay, shared consumers or reproducibility justify it. There is no mandatory universal normalized table that every source or mart must pass through.

Each mart has explicit producers, input versions, grain, permitted measures and refresh/correction rules. Usage/cost views may consume validated source facts; allocation and calculated sales also require the applicable complete pools and approved contract/rule revisions. Issued/receivable reporting consumes authoritative workflow and external evidence. A generic normalization job does not establish an invoice, payment or approval state. This is a dependency model, not an approved list of physical marts.

The durable collection store, business records and derived marts have different roles. An analytical total is not the authority for approving or issuing a bill. "Billing transaction" means the local DB consistency boundary for related billing state changes; it does not add a payment processor or banking ledger.

## Separate collection and lifecycle policy server

The owner requested a separate policy server for collection, retention and data lifecycle. It owns typed policy versions, scope/effectivity, publication/activation and control evidence; collectors and workers execute and report outcomes. Bulk source data does not transit the policy server. A separate service does not require a separate database or generic policy-language platform.

Its policy domain covers source/schema admission rules, collection schedules/limits, retention origin/duration, archive/recall, deletion eligibility and retention holds. Pricing, monetary approval, economic coverage and document issuance remain owned by the billing domain. Access permissions remain an independently enforced boundary; a retention policy does not grant collection or disclosure rights. Financial dependency references and active holds must have an explicit authoritative owner and a coordinated conflict protocol, not competing copies of control state.

Before implementation, specify policy precedence, immutable revision/digest, activation and worker acknowledgment, job scope/input references and expected state, and an action-specific outage matrix. Cached processing configuration is not proof of current collection permission. Any bounded continuation must define staleness, expiry, source/tenant scope and volume; destructive work stops when current controls cannot be established. Strict shared collection quotas require coordinated accounting or bounded reservations; independent cached counters cannot establish a global cap.

Deletion must serialize against new holds, publication, readers and dependency pins at a defined irreversible boundary. A fresh check followed by an uncoordinated delete is insufficient. Fence stale executors, reconcile unknown outcomes and distinguish requested, logically denied, physically removed and residual-backup states.

## Retention and data lifecycle

Policies apply separately to quarantine, originals, optional reusable detail, rule/mapping history, calculated/issued evidence, marts, exports and backups. Durations and legal/customer requirements are not invented in this register. Retention origin survives copying, compaction and replay. Archive verification includes readability, digests and required keys/schema; recall does not change financial state.

Maintain references needed to explain or reproduce retained results. A mart cannot be advertised as rebuildable after its required original data or exact transformation/policy revisions expire. Mixed-tenant or mixed-retention objects require scoped logical denial and, where permitted, verified retained-row rewrites; a path or partition is not authorization. Restore must reconcile current deletion/revocation/hold controls before serving or dispatching, so old backups cannot resurrect access or consumed billing coverage.

## First validation basis and remaining design work

The owner selected FinOps FOCUS data as the first verification basis. Exact specification version, public/synthetic sample provenance, license, schema and completeness must be recorded before fixture adoption. FOCUS source cost is not automatically the MSP customer selling price. Tenant mappings, selling contracts and missing correction/lifecycle cases must be explicit synthetic inputs, never presented as supplied source evidence.

The first representative path will inform schema and engine decisions without reducing the existing full product/provider scope. Sample conformance is not proof of correctness for every provider or of the greater-than-100M-source-records/day design premise. No fixtures, executable tests, benchmark, code or infrastructure are introduced here.

Next design outputs are: component/authority ownership; source-to-output and mart dependency contracts; policy/publication/correction/lifecycle state and failure tables; and a small engine/local-deployment comparison grounded in actual workloads. Per dataset, define keys/grain, types/nullability, units/currency/basis and exact numeric bounds, timestamps, tenant scope, source/release versions, partition/sort hypotheses, refresh/replay, retention and publication rules. Partitioning and shared detail placement remain workload decisions.

Initial failure cases include duplicate versus corrected intake, partial scope, tenant mismatch, output persisted before publication failure, changed policies and stale workers, hold/delete races, missing archive dependencies, restored older controls, exact-decimal edge cases and external unknown outcomes. These are acceptance cases to specify, not tests already executed.

## Integration evidence and limits

This update reconciles current owner choices with retained historical proposals. The local workspace still lacks some companion rules/skills, the binding file and planning-gate document. The owner explicitly authorized continuing with available planning evidence; unavailable inputs are not reconstructed as verified policies. Correct Hybrid Billing workspace/project and the existing accountable owner/machine claim were verified through authenticated browser UI; the Linear connector returned a different workspace and was not used for updates.

Current independent integration review and document QA are recorded in the [HB-19 verification report](../reviews/HB-19-high-level-architecture/verification.md). Runtime, numeric enforcement, tenant isolation, failover, restore, provider compatibility and capacity remain NOT_RUN. A merged document is not planning completion, a functioning policy server or an implementation acceptance result.
