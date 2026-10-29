---
type: DesignResearch
title: Provider-specific FOCUS onboarding routes
description: Separates documented provider data availability from proposed setup automation and unverified platform support.
status: draft
---

# HB-51 source profiles

Official documentation checked on 2026-10-29. All Hybrid Billing live integrations below are NOT_RUN. Native provider availability is not a platform certification. The common UI must show actual source version and scope before accepted ingestion; no single FOCUS version is forced on every source.

| Source | Documented data route | Proposed minimal user action | Required distinctions |
| --- | --- | --- | --- |
| AWS | S3 Data Exports, native FOCUS 1.2 | Select authorized account; resolve missing identity/storage/export prerequisites, then reviewed setup | Detailed AWS profile in README and research; actual deployment and workload identity prerequisites remain |
| Azure | Cost Management FOCUS export to Storage; dataset versions selectable | Select authorized billing scope/export; supply only missing Storage/permission setup | Scope/agreement eligibility before resource creation; MOSP and management-group FOCUS limitations; export writer versus Blob reader |
| Google Cloud | Native FOCUS BigQuery linked dataset, Preview; columns up to 1.2 documented | Select approved billing account/project/dataset; enable eligible export if absent | Location, preview availability, billing permissions and query permissions are distinct; query costs; not a plain S3-style file manifest |
| OCI | Native FOCUS cost reports in Oracle-owned report storage alongside proprietary reports | Select registered tenancy/report connection and approve required read access | Report type, source-declared schema and time/account coverage; missing products cannot be invented from the FOCUS label |
| Alibaba Cloud | FOCUS 1.0 invitation-based Preview export to OSS | Check entitlement first; connect approved report location | Analysis-only export, not reconciliation/settlement evidence; official bill overview/details required separately; 1.0 profile pinned |
| IBM Cloud | Native FOCUS 1.2 CSV download and monthly account API | Select registered account/service identity and report period | CSV versus paginated results, billing permissions and exact endpoint/profile validation; no copied token field |
| VMware / on-prem | Existing project plan: resource inventory, metering and change history plus approved internal-cost policies | Select registered collector, then complete only missing metering/policy dependencies | Internally calculated provenance; usage is not cost and a vCenter connection is not a supplier invoice |
| Existing file / storage | Customer-provided original evidence | Choose registered source or upload through the future supported intake; supply missing version/scope only | Unknown schema held, native versus converted explicit, one-time import distinct from automatic recurring readiness |

# Official references and limits

- [Microsoft export tutorial](https://learn.microsoft.com/azure/cost-management-billing/costs/tutorial-export-acm-data): FOCUS export, storage, selectable schemas, scope limitations and partitioning. Proposed setup assistance is not proof of implemented automatic tenant discovery or ARM deployment.
- [Google FOCUS setup](https://docs.cloud.google.com/billing/docs/how-to/export-data-bigquery-focus-setup): Google-managed immutable linked dataset, Preview, project/billing permissions and location setup. [Billing release notes](https://docs.cloud.google.com/billing/docs/release-notes) identify FOCUS columns up to 1.2. Do not reuse the older assumption that all Google FOCUS requires a customer-built SQL conversion.
- [Oracle Cost Reports](https://docs.oracle.com/en-us/iaas/Content/Billing/Concepts/costusagereportsoverview.htm): distinguish FOCUS and proprietary reports. [Oracle Cost Analysis](https://docs.oracle.com/en-us/iaas/Content/Billing/Concepts/costanalysisoverview.htm) records report coverage limitations, including VM cluster pluggable database costs; required missing evidence must be sourced separately before the affected billing basis is eligible.
- [Alibaba FOCUS export](https://www.alibabacloud.com/help/en/user-center/export-alibaba-cloud-standard-billing-focus): invitation-based FOCUS 1.0 Preview through OSS. Actual account availability and delivery/finality are verified, not presumed. The official documentation explicitly excludes these FOCUS exports from reconciliation; official bill overview/details must supply settlement authority.
- [IBM FOCUS export](https://cloud.ibm.com/docs/account?locale=en&topic=account-export-usage-focus) and [Usage Reports API](https://cloud.ibm.com/docs/apis/metering-reporting?code=java): native 1.2 and monthly account FOCUS endpoint are documented. The guide's optional x-focus-version explanation is inconsistent; pin tested API behavior rather than copy it as a settled contract. Its compliance marketing does not replace this platform's validation.
- [AWS research](research.md) retains source evidence and profile limits for the detailed first provider design.

No provider credentials or live customer data were accessed. All prefilled account/path labels in the prototype are synthetic. Proposed cross-provider automation must use each provider's actual credential and permission mechanisms, scoped to the customer-operated installation; it must not depend on a fictional common OAuth flow or vendor-held credentials.

# Detailed configuration

[HB-52 workflow and provider guides](setup-workflow.md) define the actual fields and conditional tasks behind these summaries. Automatic discovery requires existing approved access; provider-side mutations require a reviewed plan. Direct AWS export encryption is SSE-S3; post-delivery KMS is a distinct profile. IBM account and enterprise reports require different scope and identity handling.
