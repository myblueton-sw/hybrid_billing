---
type: Design
title: AWS account, export and access setup
description: Defines required connection fields, setup ownership, provider constraints and verification gates.
status: draft
---

HB-52 detailed planning; proposed product behavior, not implemented support. See [shared workflow](../setup-workflow.md). Official provider evidence was researched on 2026-10-29; real integration tests are NOT_RUN.

# Form and ownership

| Field | Control and requirement | Owner / verification |
| --- | --- | --- |
| Source account ID | Required 12-digit input or authorized account selector | Billing custodian; management/member relationship independently checked |
| Export route | Existing export ARN or new reviewed setup | Setup administrator; read existing definition before reuse |
| Authorized linked accounts | Searchable approved IDs with coverage counts | Custodian; payer membership is not customer authorization |
| Reader role ARN | Required existing role or generated plan output | IAM administrator; exact trust principal and connection ExternalId |
| Workload identity | Installation-owned reference, not browser AWS credentials | Installation administrator; off-AWS federation is a separate prerequisite |
| Destination | Bucket, prefix, owning account and region; existing selection or reviewed new-resource proposal | Storage administrator; no guessed bucket or silent policy replacement |
| Source profile | FOCUS_1_2_AWS, HOURLY, Parquet, full columns, CREATE_NEW_REPORT | Proposed default; actual configuration must be read back |

# New and existing paths

New: verify entitlement/scope/installation identity → review the resource and charge plan → administrator creates approved resources → read stack/export configuration → assume the reader role → list prefix → read actual delivered objects. Cross-account destination policy precedes source export. The proposed commercial native CloudFormation profile uses us-east-1; alternate region/partition mechanisms need their own verified profile. Do not silently import custom-resource machinery. Existing: accept export ARN, inspect scope/query/version/destination, then fill verified values without modifying the export. Incompatible filters or grain require a reviewed replacement or explicit limited coverage.

The setup actor needs separate resource creation permissions. Export service writes through a bucket policy constrained by source account/ARN; collector reads through scoped ListBucket/GetObject and export metadata access. ListExports is optional broad discovery permission and cannot be represented as export-ARN-scoped. Caller AssumeRole and target trust/ExternalId must both pass. Actual least-privilege deployment remains UNVERIFIED.

Direct delivery uses SSE-S3. SSE-KMS requires post-delivery encryption; a KMS-only destination is not the baseline. A separate re-encryption/copy profile needs approved lineage and access. Existing-bucket console setup can overwrite bucket policy: show the impact before any change.

# Evidence and recovery

First delivery can take 24 hours. Show waiting, not connected-and-complete. Read the full stable manifest/chunk set, pin schema and source execution, compare expected accounts/windows and independent totals before publication. CREATE_NEW_REPORT executions revise the same billing period; never sum successive snapshots. FOCUS 1.0 migration is a replacement, not an in-place format toggle. Historical backfill up to 14 months is documented through source-account support cases, not guaranteed by a period dropdown.

Timeout after create → inspect the same stack/export before retry. Trust/SCP/boundary failure → IAM owner; list success/read denied → storage/object owner; no delivery → source execution inspection; partial manifest → hold candidate. Quota is two exports per FOCUS version: never delete an existing consumer's export to continue. Remove temporary setup grants only with verified ownership; preserve writer grants.

# Source evidence and acceptance

- [Data Exports setup](https://docs.aws.amazon.com/cur/latest/userguide/dataexports-create.html), [delivery](https://docs.aws.amazon.com/cur/latest/userguide/dataexports-export-delivery.html).
- [IAM actions and scoping](https://docs.aws.amazon.com/service-authorization/latest/reference/list_bcm-data-exports.html), [third-party role trust](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_common-scenarios_third-party.html).
- [Storage policy](https://docs.aws.amazon.com/cur/latest/userguide/dataexports-s3-bucket.html), [encryption](https://docs.aws.amazon.com/cur/latest/userguide/data-protection.html).
- [Region setup](https://docs.aws.amazon.com/guidance/latest/cloud-intelligence-dashboards/data-exports.html), [migration/history](https://docs.aws.amazon.com/guidance/latest/cloud-intelligence-dashboards/migration-to-focus-1-2.html).

Live acceptance NOT_RUN: same/cross-account role and storage, region/partition, policy preservation, full manifest, revoked trust, cumulative replay, independent totals, backfill and cleanup. Documentation does not prove a deployable template exists.

# Default screen path

[HB-53 Stack-first design](../console-comparison.md) makes the standard new route automatic: derive proposed resources, skip manual source-field entry and inspect generated outputs after customer approval. The input table above describes the full configuration contract, not mandatory freeform fields on the default first screen. Existing and advanced routes retain those controls. Narrow DescribeStacks permission or an explicit fallback must be verified before automatic output retrieval is claimed.
