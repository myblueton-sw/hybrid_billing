---
type: Design
title: Azure agreement, export and storage setup
description: Defines required connection fields, setup ownership, provider constraints and verification gates.
status: draft
---

HB-52 detailed planning; proposed product behavior, not implemented support. See [shared workflow](../setup-workflow.md). Official provider evidence was researched on 2026-10-29; real integration tests are NOT_RUN.

# Form and ownership

| Field | Control and requirement | Owner / verification |
| --- | --- | --- |
| Entra tenant ID | Required GUID; use approved installation identity when available | IAM administrator; token tenant/audience verification |
| Agreement type | EA / MCA / MPA; unsupported option explains blocker | Billing administrator |
| Billing scope | Exact ARM scope selector or full ID | Billing custodian; verify agreement and entitled coverage |
| Reader identity | Existing managed/federated identity reference and client ID if applicable | IAM administrator; separate from export identity |
| Export | Existing name/resource ID or reviewed new plan | Scope administrator; read definition and etag |
| Destination | Storage resource ID, container, root folder, location and firewall posture | Storage administrator |
| Dataset | FocusCost, available dataVersion, format/compression | Administrator; API version and data version are separate |

# Scope and setup branches

Documented FOCUS scopes: EA enrollment/department/account/subscription/resource group; MCA billing account/profile/invoice section/subscription/resource group; MPA customer/subscription/resource group. MOSP and management groups are unsupported. Lower scopes can omit nonusage purchases; prefer EA enrollment or MCA billing profile for invoice-complete comparison where applicable rather than assuming all scopes are equivalent.

Existing: inspect definition/etag, identity, dataVersion, scope, destination and executions; reuse without mutation. New: customer portal or reviewed ARM/REST plan; a documented pinned API is 2025-03-01, definition.type FocusCost, explicit dataSet.configuration.dataVersion, Daily/MonthToDate, bounded recurrence and destination. This is a planning profile, not a deployed ARM template. Run acceptance is not file delivery. Updates bind the current etag and explicit intent.

Azure RBAC Cost Management Contributor can manage exports; Reader cannot. Billing-scope EA/MCA roles have different rules, including EA Enterprise Read-Only and MCA Reader export capabilities. EA visibility policies still apply. Destination setup needs storage rights; firewalled storage needs role-assignment authority. Export system-assigned managed identity receives container Storage Blob Data Contributor. Collector separately needs approved Storage Blob Data Reader and necessary metadata reads; never inherit writer permissions.

Trusted-services exception helps the writer, not collector connectivity. Cross-tenant firewalled storage is unsupported. Partner SAS cross-tenant export is a narrow exception, not universal onboarding. Do not request generic admin consent as a substitute for billing authority.

# Receipt, history and failure

Pin schema: documented versions include 1.2-preview, 1.0 and 1.0-preview; availability and conformance vary. Some ContractedCost/ContractedUnitPrice values have provider gaps. Check actual export version and preserve provider values. Inspect execution history, manifest and all blobs; replace complete monthly revisions atomically, never add MTD snapshots. Portal historical scope is 13 months; API requests can cover up to seven years subject availability, one month per request. Display observed history, not a seven-year guarantee.

403 → distinguish billing scope, data read and storage assignment. Writer works/reader blocked → collector network path. Timeout → inspect execution before another run. Partial blobs or schema drift → hold candidate. Disconnection preserves shared export; owned writer grants may need explicit cleanup because export deletion can leave assignments. Invoice-basis comparison remains mandatory.

# Official evidence and acceptance

- [Export setup and limitations](https://learn.microsoft.com/en-us/azure/cost-management-billing/costs/tutorial-improved-exports), [scope and permission differences](https://learn.microsoft.com/en-us/azure/cost-management-billing/costs/understand-work-scopes).
- [Create/update API](https://learn.microsoft.com/en-us/rest/api/cost-management/exports/create-or-update?view=rest-cost-management-2025-03-01), [execute API](https://learn.microsoft.com/en-us/rest/api/cost-management/exports/execute?view=rest-cost-management-2025-03-01).
- [Source schema](https://learn.microsoft.com/en-us/azure/cost-management-billing/dataset-schema/cost-usage-details-focus), [conformance gaps](https://learn.microsoft.com/en-us/cloud-computing/finops/focus/conformance-summary).

Live acceptance NOT_RUN: each agreement/scope, reader/writer separation, firewall, cross-tenant rejection, etag conflict, complete partitions, cumulative replay, historical availability and invoice comparison.
