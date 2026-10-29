---
type: Design
title: VMware and supplied evidence connection planning
description: Defines required connection fields, setup ownership, provider constraints and verification gates.
status: draft
---

HB-52 detailed planning; proposed product behavior, not implemented support. See [shared workflow](../setup-workflow.md). Official provider evidence was researched on 2026-10-29; real integration tests are NOT_RUN.

# VMware / on-premises form

This is a product proposal grounded in the existing project metering/cost distinction, not a verified vendor connector profile. Required controls: registered collector or new collector request; endpoint; approved identity reference; trusted certificate/CA reference; permitted vCenter/resource scope; usage interval; timezone; historical availability; internal-cost policy and effective date. Setup administrator owns connectivity and identity; infrastructure owner approves inventory scope; billing owner approves cost/allocation policy.

Do not assert a generic VMware read-only role covers all APIs: exact vCenter version, inventory, performance counters, historical retention and recurring permissions must be verified per supported adapter. Never disable certificate checking as an onboarding shortcut. Existing collector → inspect heartbeat/version/scope → bounded inventory/metering sample → complete time/resource coverage. New collector → administrator deployment task with explicit requirements and return verification; no fictional SaaS login creates a collector.

Inventory is not usage; usage is not a supplier bill. Missing counters or historical intervals remain missing. vCenter names do not automatically assign customer ownership. Converted FOCUS provenance must preserve the approved internal-cost policy revision, input units, time intervals and allocation basis. Acceptance requires independent source totals/coverage and policy evidence. No CSP invoice is fabricated from resource metadata.

# File / storage form

Choose one-time upload or recurring registered storage. Capture source custodian, provider/report type, declared schema/version, period, currency/timezone where source requires clarification, original filename/object identity and profile. File picker belongs to an authorized protected intake; the mockup must not read private files or credentials. Existing storage requires approved identity and bucket/container/prefix, network and actual read checks specific to its provider.

Unsupported schema leads to a mapping review task, not automatic guessed conversions. Preview masks sensitive rows and distinguishes NULL from zero; full validation covers all rows/files. Show archive/size/type constraints before intake based on implemented limits; do not invent limits now. Reject malformed/archive traversal/oversized content in the future intake implementation. A single uploaded file can never prove recurring freshness.

# Recovery and acceptance

Certificate failure → administrator trust repair; collector offline → heartbeat/network diagnosis; permission change → revalidate scope; partial metering → hold affected period; duplicate file → content/source identity inspection; changed same-path file → new immutable revision, not additive billing. Failed schema/policy comparisons cannot be waived by a green connection check.

Live acceptance NOT_RUN: exact VMware adapter/version/permissions, TLS, collector deployment/heartbeat, history and meter completeness, internal-cost calculation, protected upload parsing, recurring reads, duplicate/correction handling and customer isolation. [Shared workflow](../setup-workflow.md) defines authority and publication gates.
