# Hybrid Billing

Hybrid Billing is a self-hosted, installation-based billing platform being developed for reconciling provider costs, calculating customer charges, allocating internal costs, and handing approved billing results to ERP systems or controlled document workflows.

It is intended for managed service providers, individual enterprises, and corporate groups across cloud, on-premises, data-center, and AI usage sources. Provider cost, customer selling price, and internal allocation remain separate business concepts.

## Installation model

Customers deploy and operate Hybrid Billing in their own infrastructure. This repository targets customer-operated installations, not a project-hosted SaaS offering. Collection, storage, queries, backup and recovery belong to the customer-controlled installation. Companies, workspaces and tenants describe authorization and data boundaries within an installation; a shared company name never grants cross-tenant access.

The current executable environment is a bounded local verification stack. A supported production installer, installation upgrade/rollback procedure, authentication integration and production recovery certification are still pending. References to SaaS subscription billing describe a potential billing source or business capability, not a change to this product's delivery model.

## Current stage

The owner declared [planning complete](docs/planning/planning-completion.md) on 2026-10-22. Local implementation has started through the ticket/review workflow. No provider/ERP compatibility, authentication integration, billing implementation, or production performance has been certified.

The v1.16 source plan covers 123 chapters and 102 tables. Current planning artifacts organize that scope into 43 requirement groups and 49 screen specifications. Additional requirements include more than one million incoming records per day, data retention/deletion, account lifecycle and authentication, contract Payment Terms, and monthly ERP recognition independent of customer billing cadence.

## Local executable FOCUS path

[Run the HB-27 local path](docs/ops/installation/local-focus-path.md) against the existing Docker verification stack: real PostgreSQL authority tables, ClickHouse facts and S3 originals/Parquet exports. Version-declared input selects a validated supported-version registry; the bundled entries are 1.2 and 1.3. Synthetic local personas/profile are for local verification only. Product authentication, live providers, pricing/issuance and production scale remain unimplemented.

The [tenant-aware official demo path](docs/ops/installation/tenant-focus-data.md) additionally downloads the pinned official FOCUS 1.0 sample, preserves and stores its original rows, and exposes tenant-scoped detail/aggregate queries. Two synthetic tenant profiles partition the actual 1,000 public rows (942/58) using explicit source account/period assignments; exact values and NULLs are retained. This is exploratory evidence ingestion with configurable tenant properties, separate from full FOCUS conformance or billing acceptance. [Execution evidence](docs/validation/reports/HB-27-tenant-focus/summary.md).

The [six-provider sample workflow](docs/ops/installation/six-provider-samples.md) adds all 115 observed GCP rows and eight explicitly synthetic VMware/Alibaba rows. It stores 1,123 sample records and provides an explicitly authorized 1,121-row company view across two tenants while excluding a separate company's two rows, even when company display labels match. Provider, currency, unit, tenant and origin boundaries remain visible. [Validation and limits](docs/validation/reports/HB-27-six-provider/summary.md).

The [recurring collection and lifecycle workflow](docs/ops/installation/focus-collection-lifecycle.md) accepts bounded CSV or exact Parquet input under an explicitly declared registry version, registers operator-declared complete daily or monthly delivery windows, preserves revisions, replaces cumulative monthly refreshes, and queries accumulated authorized periods. It implements verified tenant Parquet archive/recall, holds/dependencies, approved deletion with explicit shared/physical-copy residuals, origin-based backup horizons and latest-control restore gates. Use its periodic query command after activation; the earlier single-snapshot query then rejects rather than showing only the newest delivery. This is a bounded trusted local CLI; live cloud adapters, production authentication, multi-host recovery and capacity remain outside its verified scope.

The [native account hierarchy workflow](docs/ops/installation/native-account-hierarchy.md) normalizes six-provider IDs and typed billing/resource/organization/identity relationships, persists tenant-scoped immutable snapshots, and exposes hierarchy-filtered and account-grouped queries without multiplying costs. Approved supplemental metadata cannot broaden tenant access or rewrite typed historical snapshots. Public samples retain explicit unknown native ancestry; separately labelled synthetic metadata validates complete provider-specific shapes.

The [common-ingestion omission audit](docs/reviews/HB-30-focus-completeness/findings.md) records the original gaps. Its [remediation](docs/reviews/HB-30-focus-completeness/resolution.md) and [connection runbook](docs/ops/installation/common-focus-connections.md) add independent source/window identity, expected account/calendar coverage, pinned consumed-field semantics, guarded quarantine/replacement and scheduler fairness. These remain bounded installation workflows; full provider conformance, live connectors and installation production readiness require separate evidence.

## Planned capabilities

- Collect and validate provider evidence with explicit coverage, source authority, immutable revisions, and reconciliation.
- Maintain customer contracts, rates, discounts, allocation policies, exchange rates, and approved calculation snapshots.
- Prevent duplicate billing across retries, corrections, ERP submissions, and manual file handoffs.
- Separate approval, invoice issuance, document delivery, settlement, and dispute handling.
- Support administrator operations and customer access through scoped permissions and controlled disclosure.
- Design for sustained ingestion, monthly closing, recovery, and auditable evidence retention.

Technology choices and supported provider/ERP combinations remain subject to planning decisions and actual validation. Existing HawkEye and TokenMeter workflows inform collaboration practices; their product stacks are not assumed.

## Architecture preparation

Start with the [current owner decisions](docs/architecture/current-design-decisions.md): Go backend, Python processing/analysis, React frontend, cloud-independent operation, local Docker and a separate collection/lifecycle policy server. Earlier component bundles remain comparison proposals; integration does not declare planning complete.

The [backend logical architecture](docs/architecture/high-level-architecture.md) starts with business responsibilities, authoritative ownership and principal interactions. Context diagrams and twelve gated flow contracts provide supporting detail before configuration selection. SaaS subscription, payment and entitlement paths are explicitly proposed through the conditional PR14 proposal; its original stack and physical sizing hypotheses are superseded where the current owner decisions differ.

The [HB-21 logical data schema draft](docs/architecture/data-ownership/logical-data-schema.md) centers on [actual usage, MSP contract pricing, monthly closing and final charge adjustments](docs/architecture/data-ownership/monthly-settlement-schema.md), with separate evidence/detail and financial authority. Direct/resale responsibility, provisional versus final amounts and pre-issuance versus issued corrections are explicit. Its latest capacity premise is more than 100 million source records/day; earlier examples elsewhere remain historical planning inputs. Physical database design, policy adoption and runtime verification remain pending.

The owner has selected self-hosted delivery. The [architecture readiness plan](docs/architecture/architecture-readiness.md) organizes the operating prerequisites, development sequence, ClickHouse/Kafka/OpenTelemetry assessment, billing data model, collection methods and source grouping, tenant/customer/organization management, session and permission policies, admin/customer data views, and intelligence layer. PostgreSQL is the first transactional candidate; component adoption, sizing and implementation remain unverified.

The [external-source design review](docs/reviews/HB-8-external-design/findings.md) supports the conceptual direction and records six open contract refinements with closure criteria. Official sources and published expert research informed the review; external human consultation and runtime validation have not been performed.

The owner requires [complete authorized usage details and maintained storage](docs/architecture/usage-detail-lifecycle.md), including cost explanation, pricing/charge finalization, historical revisions, full scoped exports and archive/retention/recovery contracts. The complete product requirements remain broader than the bounded local collection/lifecycle implementation linked above.

The [recent owner requirement register](docs/planning/recent-owner-requirements.md) consolidates fourteen discussion bundles without inflating the original 43-group count. The [current unresolved review](docs/reviews/HB-10-unresolved/findings.md) confirms all thirteen earlier refinements remain open and defines evidence-backed closure order; document publication is not finding closure.

The [HB-11 supplementation](docs/reviews/HB-11-refinements/resolution.md) supplies reviewed security, financial and operational contracts for all thirteen refinements, with detail/retention decision packets and planned acceptance. Profile/policy choices and canonical trace integration remain pending; no runtime enforcement or planning completion is claimed.

The [installation assessment](docs/architecture/self-hosted-installation-assessment.md) conditionally favors an existing customer-operated Kubernetes cluster and retains a VM/container comparison profile. New-cluster operations, stateful placement and actual installation/failure validation remain unverified; VMware is not required.

The [full planning reconciliation](docs/planning/v1.16/planning-reconciliation.md) indexes all 43 requirement groups, existing work/screen/acceptance references and owner additions. The [decision/action register](docs/planning/v1.16/planning-decisions.md) separates established controls, integration gaps, product decisions, customer configuration and external evidence. The [payment and recognition contract](docs/contracts/billing/payment-and-recognition.md) carries contract Payment Terms and independent monthly ERP recognition into English. This is partial canonical integration, not completion of the detailed English baseline or planning acceptance.

## Planned SaaS subscription billing capability

The owner has requested future SaaS subscription and billing management. The [HB-18 planning supplement](docs/planning/saas-subscription-billing.md) proposes recurring, seat-based, metered and hybrid pricing, subscription changes, service entitlements, customer self-service and controlled payment integration. It extends existing contract, calculation, approval, invoice and ERP concepts; payment collection, service access and monthly cost recognition remain distinct. Exact policies, processor support and canonical screen/API integration are pending. This capability does not change self-hosted delivery or authorize product development.

## Working sequence

All project work follows a matching Hybrid Billing Linear ticket, assigned ownership and machine claim, ticket branch, planning and relevant review, scoped execution, verification and re-review, PR, authorized merge, and merge read-back. This README-only initial commit is the owner's authorized seed for an otherwise empty repository; subsequent changes follow the full PR sequence.

User conversation is in Korean. Git-bound documents, tool/agent task instructions, tickets, commits, and PRs are in English.

The planning baseline is maintained locally under `docs/planning/v1.16/`; screen designs are under `docs/ui/`. Those artifacts are being reviewed and will be added through tracked follow-up changes.

## License and commercial inquiries

The current project-authored original material is offered under the [PolyForm Noncommercial License 1.0.0](LICENSE), with the applicability described in [LICENSING.md](LICENSING.md). It is source-available, not an OSI open-source license. Commercial-purpose use outside the license's permitted purposes requires a separate agreement; contact [myblueton@gmail.com](mailto:myblueton@gmail.com).

The owner reports sole development and no external distribution or downloads; this has not been independently audited. Any rights already obtained for earlier MIT copies remain unaffected, with the [historical license and notice](LICENSES/MIT-legacy.txt) preserved for that scope. They do not offer the current revision under MIT. Third-party rights and external-contributor commercial permissions remain separate. See the [owner's transition decision](docs/licensing/source-available-adoption.md), [current policy](docs/licensing/product-policy.md) and [third-party notices](THIRD_PARTY_NOTICES.md).
