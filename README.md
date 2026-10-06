# Hybrid Billing

Hybrid Billing is a planned platform for reconciling provider costs, calculating customer charges, allocating internal costs, and handing approved billing results to ERP systems or controlled document workflows.

It is intended for managed service providers, individual enterprises, and corporate groups across cloud, on-premises, data-center, and AI usage sources. Provider cost, customer selling price, and internal allocation remain separate business concepts.

## Current stage

The project is in planning and design review. **Product development must not begin until the project owner explicitly declares planning complete.** No provider/ERP compatibility, authentication integration, billing implementation, or production performance has been certified.

The v1.16 source plan covers 123 chapters and 102 tables. Current planning artifacts organize that scope into 43 requirement groups and 49 screen specifications. Additional requirements include more than one million incoming records per day, data retention/deletion, account lifecycle and authentication, contract Payment Terms, and monthly ERP recognition independent of customer billing cadence.

## Planned capabilities

- Collect and validate provider evidence with explicit coverage, source authority, immutable revisions, and reconciliation.
- Maintain customer contracts, rates, discounts, allocation policies, exchange rates, and approved calculation snapshots.
- Prevent duplicate billing across retries, corrections, ERP submissions, and manual file handoffs.
- Separate approval, invoice issuance, document delivery, settlement, and dispute handling.
- Support administrator operations and customer access through scoped permissions and controlled disclosure.
- Design for sustained ingestion, monthly closing, recovery, and auditable evidence retention.

Technology choices and supported provider/ERP combinations remain subject to planning decisions and actual validation. Existing HawkEye and TokenMeter workflows inform collaboration practices; their product stacks are not assumed.

## Architecture preparation

The owner has selected self-hosted delivery. The [architecture readiness plan](docs/architecture/architecture-readiness.md) organizes the operating prerequisites, development sequence, ClickHouse/Kafka/OpenTelemetry assessment, billing data model, collection methods and source grouping, tenant/customer/organization management, session and permission policies, admin/customer data views, and intelligence layer. PostgreSQL is the first transactional candidate; component adoption, sizing and implementation remain unverified.

The [external-source design review](docs/reviews/HB-8-external-design/findings.md) supports the conceptual direction and records six open contract refinements with closure criteria. Official sources and published expert research informed the review; external human consultation and runtime validation have not been performed.

The owner requires [complete authorized usage details and maintained storage](docs/architecture/usage-detail-lifecycle.md), including cost explanation, pricing/charge finalization, historical revisions, full scoped exports and archive/retention/recovery contracts. These are planned requirements, not implemented capabilities.

The [recent owner requirement register](docs/planning/recent-owner-requirements.md) consolidates fourteen discussion bundles without inflating the original 43-group count. The [current unresolved review](docs/reviews/HB-10-unresolved/findings.md) confirms all thirteen earlier refinements remain open and defines evidence-backed closure order; document publication is not finding closure.

The [HB-11 supplementation](docs/reviews/HB-11-refinements/resolution.md) supplies reviewed security, financial and operational contracts for all thirteen refinements, with detail/retention decision packets and planned acceptance. Profile/policy choices and canonical trace integration remain pending; no runtime enforcement or planning completion is claimed.

The [installation assessment](docs/architecture/self-hosted-installation-assessment.md) conditionally favors an existing customer-operated Kubernetes cluster and retains a VM/container comparison profile. New-cluster operations, stateful placement and actual installation/failure validation remain unverified; VMware is not required.

The [full planning reconciliation](docs/planning/v1.16/planning-reconciliation.md) indexes all 43 requirement groups, existing work/screen/acceptance references and owner additions. The [decision/action register](docs/planning/v1.16/planning-decisions.md) separates established controls, integration gaps, product decisions, customer configuration and external evidence. The [payment and recognition contract](docs/contracts/billing/payment-and-recognition.md) carries contract Payment Terms and independent monthly ERP recognition into English. This is partial canonical integration, not completion of the detailed English baseline or planning acceptance.

## SaaS subscription planning

The owner has requested future SaaS subscription and billing management. The [HB-18 planning supplement](docs/planning/saas-subscription-billing.md) proposes recurring, seat-based, metered and hybrid pricing, subscription changes, service entitlements, customer self-service and controlled payment integration. It extends existing contract, calculation, approval, invoice and ERP concepts; payment collection, service access and monthly cost recognition remain distinct. Exact policies, processor support and canonical screen/API integration are pending. This capability does not change self-hosted delivery or authorize product development.

## Working sequence

All project work follows a matching Hybrid Billing Linear ticket, assigned ownership and machine claim, ticket branch, planning and relevant review, scoped execution, verification and re-review, PR, authorized merge, and merge read-back. This README-only initial commit is the owner's authorized seed for an otherwise empty repository; subsequent changes follow the full PR sequence.

User conversation is in Korean. Git-bound documents, tool/agent task instructions, tickets, commits, and PRs are in English.

The planning baseline is maintained locally under `docs/planning/v1.16/`; screen designs are under `docs/ui/`. Those artifacts are being reviewed and will be added through tracked follow-up changes.

## Rights and status

The intended product policy is source-available commercial software. Final product license terms, rights-holder details, and third-party distribution rights remain under review. This overview does not grant a product license or certify release readiness.
