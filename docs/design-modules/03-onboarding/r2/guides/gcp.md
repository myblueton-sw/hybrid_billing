---
type: Design
title: Google Cloud billing and BigQuery setup
description: Defines required connection fields, setup ownership, provider constraints and verification gates.
status: draft
---

HB-52 detailed planning; proposed product behavior, not implemented support. See [shared workflow](../setup-workflow.md). Official provider evidence was researched on 2026-10-29; real integration tests are NOT_RUN.

# Form and ownership

| Field | Control and requirement | Owner / verification |
| --- | --- | --- |
| Billing account ID | Approved account selector or ID | Billing owner; whole source custody before query |
| Existing/new FOCUS export | Select route; native export is Preview | Billing administrator |
| Destination project and location | Required for new export; project must use the same billing account | Project owner; residency and history effect shown before save |
| Linked dataset and table | Discover after creation; existing path accepts exact IDs | Read-only inspection; do not invent IDs for users to enter |
| Query project | Required, can differ from source project subject permissions/location | Owner approves query cost and byte cap |
| Reader identity | Existing service identity/WIF reference | IAM owner; issuer/audience/subject restrictions for new federation |
| Requested history | Bounded interval with actual availability | Billing operator |

# Setup and permissions

Billing Account Costs Manager or Administrator plus Project IAM Admin and BigQuery Admin are documented setup roles. New project creation adds Project Creator. The selected project must have billing enabled against that same billing account. Location choices must be validated against the Cloud Billing FOCUS supported subset (US/EU and listed regions), not the complete BigQuery location list; unsupported locations block the plan before creation. Discover identifiers matching gcp_billing_immutable_<BILLING_ACCOUNT_ID>_<Location> and gcp_billing_export_focus_<BILLING_ACCOUNT_ID>, then verify actual resources rather than constructing names as proof. Console: Billing → Billing export → BigQuery export → enable FOCUS → project → BigQuery API → location → save. Google creates a linked immutable dataset/table. Preserve billing-export-bigquery@system.gserviceaccount.com.

Recurring reader is separate: BigQuery dataViewer on selected dataset/table and jobUser on execution project. WIF is installation setup, not a magic OAuth button. Direct access or impersonation needs verified issuer/audience/subject mapping; impersonation binds selected external subjects with workloadIdentityUser, not the whole pool by convenience. No pasted service-account private key in the mockup.

Existing route inspects dataset/table/location/schema; new route waits for Google-created identifiers. Verify identity → metadata → bounded actual query → complete interval/expected project coverage → independent totals → confirmation. Record job ID, location, scanned bytes, source/table/profile, row count, period and freshness. Source metadata success is not query permission.

# History and recovery

US/EU initial enablement backfills from the previous month's start; regional setup starts on enablement. Initial catch-up may take five days. Retention is two years. Disabling creates a gap that is not backfilled. Moving location/project requires re-enabling with a new dataset and does not transfer history: never use toggle-reset as generic repair. Queries cost money; linked source storage does not.

Token failure → identity owner; job denied vs table denied → respective project/dataset owner; location mismatch → correct query location without relocating source. Empty initial data → waiting. Source stopped → inspect export/service identity. Schema drift → quarantine adapter mapping. Use stable job/batch identity and replace repeated intervals without double billing.

Native Preview documents columns through FOCUS 1.2, not full conformance. Missing InvoiceId/commitment fields/Tags and ChargeCategory limitations require provider extension preservation and separate invoice evidence. Do not derive missing discounts from zero assumptions.

# Official evidence and acceptance

- [Setup, roles and limitations](https://docs.cloud.google.com/billing/docs/how-to/export-data-bigquery-focus-setup).
- [Schema/conformance](https://docs.cloud.google.com/billing/docs/how-to/export-data-bigquery-tables/focus-export), [release notes](https://docs.cloud.google.com/billing/docs/release-notes).
- [BigQuery IAM](https://docs.cloud.google.com/bigquery/docs/access-control), [WIF](https://docs.cloud.google.com/iam/docs/workload-identity-federation).

Live acceptance NOT_RUN: intended WIF identity with linked dataset, job/data permission separation, query budget, regional history, disabled gap, immutable query receipt, schema drift, replay and independent financial evidence.
