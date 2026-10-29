---
type: Research
title: AWS FOCUS onboarding evidence and unresolved implementation checks
description: Separates official AWS documentation from proposed onboarding automation and existing local capabilities.
status: draft
---

# Official evidence reviewed for HB-49

Sources accessed on 2026-10-29. These support feasibility and constraints; none is evidence of a successful Hybrid Billing AWS deployment.

| Source | Observed fact | Design implication |
| --- | --- | --- |
| [AWS Quick create links](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-console-create-stacks-quick-create-links.html) | Console links prepopulate a single stack creation page with an S3 template URL and non-NoEcho parameters. Users still review and submit. | Reduce forms, but do not promise OAuth, automatic return or one literal click. Template hosting is an installation/release prerequisite. |
| [AWS CloudFormation Export resource](https://docs.aws.amazon.com/AWSCloudFormation/latest/TemplateReference/aws-resource-bcmdataexports-export.html) | AWS::BCMDataExports::Export specifies an export query and delivery configuration. | Provisioning can be packaged in a reviewed template; this project has not built or tested that template. |
| [Creating AWS exports](https://docs.aws.amazon.com/cur/latest/userguide/dataexports-create.html) | S3 policy must exist for SDK/CLI creation; initial delivery can take up to 24 hours. Billing export refresh occurs at least daily once delivery begins. | Show asynchronous waiting and actual evidence, rather than a synchronous completion spinner. |
| [AWS FOCUS migration](https://docs.aws.amazon.com/cur/latest/userguide/table-dictionary-focus-1-2-migrating.html) | FOCUS 1.2 uses FOCUS_1_2_AWS, manifest FocusVersion 1.2 and configurable hourly/daily/monthly granularity. Migration changes rows and field semantics, including unused commitment quantities. | Pin 1.2 as the proposed new-export profile, preserve null semantics, and do not relabel old 1.0 files or compare row counts across versions without explanation. |
| [AWS FOCUS 1.2 table dictionary](https://docs.aws.amazon.com/cur/latest/userguide/table-dictionary-focus-1-2-aws.html) | Management-account exports cover management/member usage; member exports cover only that member. Organization membership changes limit historical coverage. Its introductory prose inconsistently mentions 1.0 despite the 1.2 title/table. | Preserve scope/period limits. Use explicit table, manifest and tested schema together; document inconsistency instead of relying on a page title as conformance proof. |
| [AWS export quotas](https://docs.aws.amazon.com/cur/latest/userguide/dataexports-quotas.html) | Two exports per FOCUS 1.2 table and two per FOCUS 1.0 table are documented. | Existing-export reuse and quota failure are first-class paths; never automatically delete another export. |
| [AWS export delivery](https://docs.aws.amazon.com/cur/latest/userguide/dataexports-export-delivery.html) | Export executions contain manifests and possibly multiple chunks; create-new and overwrite have different revision paths. | Bind complete execution snapshots; never sum repeated cumulative deliveries. Proposed create-new mode supports retaining upstream execution evidence at additional storage cost. |
| [AWS third-party role guidance](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_common-scenarios_third-party.html) | Roles delegate scoped access without sharing security credentials. External IDs address confused-deputy risk and are not secrets. | Use a customer-controlled trusted workload principal, scoped role and per-connection binding. ExternalId alone is not authentication or tenant authorization. |

# Repository evidence and continuity

- [v10 requirements](../../../reviews/HB-43-onboarding/requirements.md): separate custody classes, operational reads, resumable setup and independent readiness states.
- [OP-05/06 obligations](../../../contracts/operations/source-policy-onboarding-and-verification-obligations.md): financial setup order, no guessed money/legal defaults, actual reads and version-specific FOCUS readiness.
- [Current local connection runbook](../../../ops/installation/common-focus-connections.md): operator-declared source calendars, immutable connections, approved domains and consumed-field profiles; explicitly no live provider adapter.
- [Collection architecture](../../../architecture/data-collection.md): complete independent cross-validation, required comparison evidence and operator confirmation before eligible publication.
- [Planning commands](../../../contracts/commands/planning-actions.md): input.inspect and input.confirm_publish remain distinct, revision-bound operations.

The proposed 1.2 AWS export is not certified merely because the local registry lists 1.2/1.3. Native provider fields, delivery manifests, account coverage and permission controls require a verified adapter/profile. No assumption of a supported 1.0 production adapter follows from the earlier exploratory sample.

# Open implementation checks

- Release-tested AWS partition/region, complete table schema and conformance-gap profile; documentation inconsistencies must be resolved against actual supported source evidence.
- Pinned CloudFormation artifact distribution, reviewable template and IAM policy minimums; setup account checks and resource rollback/retention behavior.
- Customer-controlled workload identity for AWS-hosted and off-AWS installations; no central vendor service assumed.
- Existing-export metadata discovery permission and destination authorization; supported KMS and cross-account storage profiles.
- Delivery polling/backoff, history availability, source-version retention, action counts and initial-delivery timing under actual deployment.
- Independent verification method and required billing-basis comparisons; no global numeric tolerance or inferred zero.

All live-provider and UI acceptance is NOT_RUN. These are explicit implementation prerequisites, not reasons to add configuration fields to the default user flow.
