---
type: Design
title: Alibaba preview, OSS and billing evidence setup
description: Defines required connection fields, setup ownership, provider constraints and verification gates.
status: draft
---

HB-52 detailed planning; proposed product behavior, not implemented support. See [shared workflow](../setup-workflow.md). Official provider evidence was researched on 2026-10-29; real integration tests are NOT_RUN.

# Form and ownership

| Field | Control and requirement | Owner / verification |
| --- | --- | --- |
| Account UID/site | Required exact billing source | Billing owner; source custody |
| Preview eligibility | Verified Bill Subscription availability, not assumed from logo | Billing administrator |
| Subscription | Existing or new Standard Bill FOCUS; start month | Billing administrator |
| OSS destination | Region, bucket, optional directory | Storage owner |
| Reader | Approved role ARN/credential reference | Security owner; temporary session renewal |
| Official billing source | Separate bill overview/details connection or evidence intake | Billing owner; required for settlement |

# Setup and receipt

FOCUS 1.0 is invitation Preview and explicitly analysis-only: it cannot support reconciliation. Existing subscription: inspect source and actual files without recreation. New: Billing → Bill Subscription → first-use Authorize/Agree → OSS settings → Standard Bill FOCUS → start month → bucket/directory. AliyunConsumeDump2OSSRole is the export writer authorization, not the recurring collector role.

Collector proposal scopes ListObjects to bucket/prefix and GetObject to objects; exact setup RAM actions and encryption/temporary-credential profile remain UNVERIFIED. Never equate billing read-only rights with subscription creation. Platform stores a protected credential reference, not AK values in documentation or normal configuration state.

Read complete CSV split sets or ZIP archives. Preserve actual names/timestamps rather than assuming a fixed delivery suffix. MTD daily outputs replace prior complete revisions, not additive charges. Historical floor depends on first billing after the upgraded preview launch; no generic history promise. Monthly documentation gives differing fourth-day times (after noon/by 18:00 UTC+8): actual delivery/finality evidence is required.

# Data limits and repair

Null ChargeDescription, ConsumedQuantity/Unit, ContractedCost/UnitPrice and intermittent price/unit gaps are documented. Preserve NULL and +08:00 offsets; never invent zero prices or discounts. Required reservation/commitment visibility may be incomplete.

No subscription menu → eligibility owner; no object → distinguish expected wait; list denied → prefix policy; object denied → read/decryption; partial ZIP/split → quarantine whole revision; expired session → renew then replay same immutable source; schema change → mapping review. Analysis-ready never becomes settlement-ready without authoritative bills and independent comparison.

# Official evidence and acceptance

- [FOCUS preview, timing and authority limits](https://www.alibabacloud.com/help/en/user-center/export-alibaba-cloud-standard-billing-focus).
- [First-use export authorization](https://www.alibabacloud.com/help/en/user-center/export-and-subscribe-bills/), [OSS policy scope](https://www.alibabacloud.com/help/en/oss/user-guide/access-control-base-on-ram-policy).
- [Official field differences](https://www.alibabacloud.com/help/zh/user-center/field-differences-of-alibaba-cloud-focus-1-0-preview).

Live acceptance NOT_RUN: preview entitlement, exact setup actions, STS renewal, bucket/decryption, all files/ZIP parts, null mapping, cumulative replay and authoritative bill reconciliation. Chinese official conformance page supplies evidence where English retrieval was unavailable.
