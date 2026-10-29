---
type: Design
title: OCI report tenancy and Object Storage access
description: Defines required connection fields, setup ownership, provider constraints and verification gates.
status: draft
---

HB-52 detailed planning; proposed product behavior, not implemented support. See [shared workflow](../setup-workflow.md). Official provider evidence was researched on 2026-10-29; real integration tests are NOT_RUN.

# Form and ownership

| Field | Control and requirement | Owner / verification |
| --- | --- | --- |
| Report-owning tenancy OCID | Required; standalone/parent choice | Billing administrator; consolidated source custody |
| Home region and realm | Required approved region; commercial FOCUS availability checked | Tenancy administrator |
| Signing identity | Tenancy/user reference and approved secret-store reference | IAM owner; fingerprint/rotation and clock checks |
| Reader group/policy | Existing approved group or scoped administrator task | Tenancy administrator |
| Report source | Namespace bling, bucket report-owning customer tenancy OCID, prefix FOCUS Reports | Derived proposal then actual list/read verification |
| Expected children/subscriptions | Approved effective-dated IDs, searchable selection | Billing owner; no automatic customer authorization |
| Evidence type | FOCUS; additional proprietary evidence when needed | Billing specialist |

# Source and setup

OCI reports live in Oracle-owned Object Storage. New connection configures access; do not offer a customer bucket or export creation. Non-metered tenancies are ineligible. Report-owning tenancy and signing tenancy are separate fields; unreviewed cross-customer-tenancy signing profiles remain unsupported. The administrator action card links the exact official reporting policy in the source below, displays its literal Oracle source-tenancy value unchanged, and substitutes only the approved customer group. Required policy defines Oracle's documented reporting tenancy and endorses the selected group to read its objects. Preserve the exact official source-tenancy value in the administrator action card; do not invent an OCID. Dynamic-group alternatives require separate verification.

Administrator steps: select the report-owning home region → Billing & Cost Management → Cost and Usage Reports → FOCUS folders; apply the reviewed official policy; run paginated ListObjects with the literal prefix quoted as "FOCUS Reports" in generated CLI instructions; then GetObject using a complete listed name and verify the full body. List success alone leaves read verification pending. Use report-owning tenancy home region, namespace bling and literal prefix FOCUS Reports. Exhaust listing pagination and stream complete objects. Parent organization reports consolidate children; a child does not necessarily have an independent populated report bucket. Custody over the full parent raw source must precede download, not be retroactively inferred from later row filtering. Invited-child and reassigned-subscription history can be split across earlier sources.

# Delivery, evidence and recovery

Detailed documentation describes six-hour generation and possible 24-hour delay, while introductory cadence differs. Show observed freshness, not a guaranteed schedule. Retention is one year; large reports split into gzip chunks. Record bucket, home region, signer reference, complete object path and content identity, all pages/chunks, decompression, header/profile, periods and observed subscription IDs. A successful listing is not a successful read.

401 → signing reference/clock; 403/404 → policy, home region, parent bucket and eligibility diagnostics without broadening access. Empty prefix → waiting/eligibility/ownership checks, not zero cost. Missing page/chunk or truncated gzip → hold period and retry exact object. Re-list bounded correction windows and retain original lineage; do not copy proprietary correction semantics into native FOCUS without proof.

BilledCost excludes tax. VM-cluster pluggable-database attribution is unavailable in FOCUS and needs proprietary evidence. Parent cluster totals and child attribution are different measures, never additive by default. Invoice reconciliation and current authority gate publication.

# Official evidence and acceptance

- [Report types/policy](https://docs.oracle.com/en-us/iaas/Content/Billing/Concepts/costusagereportsoverview.htm).
- [List source/region/prefix](https://docs.oracle.com/en-us/iaas/Content/Billing/Tasks/list-cost-usage-report.htm), [download](https://docs.oracle.com/en-us/iaas/Content/Billing/Tasks/download-cost-usage-report.htm).
- [Organization reporting](https://docs.oracle.com/en-us/iaas/Content/General/organization/organization_cost_reporting.htm), [attribution limitation](https://docs.oracle.com/en-us/iaas/Content/Billing/Concepts/costanalysisoverview.htm).

Live acceptance NOT_RUN: actual schema/version, intended signer, parent/child access, realm/home region, pagination/gzip completeness, subscription moves, correction behavior and tax/attribution reconciliation.
