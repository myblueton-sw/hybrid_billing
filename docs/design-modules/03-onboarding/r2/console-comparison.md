---
type: DesignResearch
title: Stack-first connection and provider-console comparison
description: Defines automated AWS setup and the provenance and limits of official console images shown beside connection forms.
status: draft
---

# HB-53 decision and ownership

[HB-53](https://linear.app/hybrid-billing/issue/HB-53/document-stack-first-onboarding-and-console-comparison) follows the owner's requirement to make AWS easy through CloudFormation and compare connection steps with actual provider-console screens. Root owns writing; separate AWS/Azure, GCP/OCI and Alibaba/IBM reviewers inspect source evidence and actual changes. No real accounts, deployment or credentials are authorized by this design work.

# AWS default and exception paths

New AWS connection defaults to a reviewed standard Stack. Existing Export and advanced configuration remain alternatives. The standard form asks for the account, management/member context, whole-source custody and verified installation workload identity; it does not require names/ARNs for resources that do not exist yet. The proposed Stack derives names, scoped trust, connection ExternalId, dedicated bucket/prefix and FOCUS export configuration. The source-settings page is automatically represented in the review plan instead of requiring another manual input screen.

Default profile: commercial us-east-1, same-account dedicated storage, SSE-S3, FOCUS_1_2_AWS/HOURLY/Parquet/CREATE_NEW_REPORT. Cross-account storage, alternate partitions/regions, existing shared resources and post-delivery KMS require separately reviewed advanced profiles. Installation identity/federation is a one-time prerequisite, not an invented OAuth callback. Customer attribution remains separate from authorized custody over an entire payer export.

The plan must show template identity/version/digest, exact trusted installation principal, resources, permission impact and possible provider charges before launch. A production Quick create URL may prefill only nonsecret parameters and must reference the reviewed immutable hosted template. Current prototype has no deployable template or live launch link; its comparison and read-back actions are explicitly synthetic. The example requested month is not a promise of source history.

The customer checks account/region and proposed parameters in AWS, acknowledges required IAM creation, then chooses Create stack. Returning to the application resumes bounded inspection. Automatic Outputs read-back requires verified narrow DescribeStacks access; it must not silently widen the collector role. If unavailable, use the verified deterministic-role/export-metadata route or an explicit manual output fallback. Exact minimum policy and deployed template remain release gates. Unknown create outcome must be inspected before retrying the same operation.

CREATE_COMPLETE verifies infrastructure only. Role assumption, export metadata, actual object read, full receipt/schema/coverage, independent financial comparison and authorized publication remain separate gates. Rebuilding the plan clears generated Outputs, provisioning/read/collection evidence and launch intent together; unchanged custody is still checked. Returning through account setup cannot retain stale success with deleted Outputs.

Official mechanics: [Quick create links](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-console-create-stacks-quick-create-links.html), [stack data](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-console-view-stack-data-resources.html), [Events](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/view-stack-events.html).

# Comparison behavior

The prototype has an optional side-by-side comparison panel: application configuration on the left, public official console evidence and matching actions on the right. At narrower widths the panels stack without clipping. Users choose the relevant screenshot stage and can enlarge the original image or open its source. An unavailable image produces an explicit fallback, not a fabricated capture or success badge.

Remote images load only after comparison is opened, from a fixed public URL catalog with no-referrer. User inputs, credentials and connection values are never interpolated into those URLs. The prototype does not promise that no-referrer disables all browser cookie behavior. No third-party image copies are committed or rebranded as Hybrid Billing assets. Rights remain with the source publisher. These public examples are not authenticated observations of the user's current console.

# Visual evidence catalog

Research and pixel inspection were performed against public official material. Dates below describe image provenance, not a claim that the current console is identical.

| Provider | Verified public images | Exact use and limitations |
| --- | --- | --- |
| AWS | Quick create/IAM acknowledgement and Outputs from the [2020 AWS example](https://aws.amazon.com/blogs/machine-learning/building-a-visual-search-application-with-amazon-sagemaker-and-amazon-es/); Events from the [2019 AWS example](https://aws.amazon.com/blogs/mt/managing-multiple-stacks-and-overriding-parameters-in-aws-cloudformation/) | Historical real console screenshots, not current 2026 UI or Hybrid Billing resources. Example template URL, Stack names and Outputs must not be copied into this setup. Current procedure reference has no console screenshot. |
| Azure | Basics, Datasets context and Destination images from [official export tutorial](https://learn.microsoft.com/en-us/azure/cost-management-billing/costs/tutorial-improved-exports) | FOCUS templates are visible. The right-hand Datasets panel is Price sheet; its 2023-05-01 version is not FOCUS. Source text governs current scope/version eligibility. |
| Alibaba | Bill Subscription list, OSS settings form and FOCUS dropdown from [official subscription guide](https://www.alibabacloud.com/help/en/user-center/export-and-subscribe-bills/) | Image label FOCUS Billing Data differs from prose Standard Bill FOCUS. List timing/banner is generic, not authoritative FOCUS finality. RAM Authorize/Agree screenshot unavailable. |
| Google Cloud | Current native FOCUS screenshot unavailable in [official setup guide](https://docs.cloud.google.com/billing/docs/how-to/export-data-bigquery-focus-setup) | Show current procedure and missing-image state; do not substitute old SQL conversion screenshots for the native linked dataset. |
| OCI | Official [2024 announcement](https://blogs.oracle.com/cloud-infrastructure/announcing-focus-support-for-oci-cost-reports) links an image, but image retrieval did not produce inspectable pixels | No verified image displayed. Current [listing guide](https://docs.oracle.com/en-us/iaas/Content/Billing/Tasks/list-cost-usage-report.htm) takes precedence over the historical FOCUS/ path; use FOCUS Reports. |
| IBM | No matching screenshots found in official [Account](https://cloud.ibm.com/docs/account?topic=account-export-usage-focus) or [Enterprise](https://cloud.ibm.com/docs/enterprise-management?topic=enterprise-management-enterprise-export-focus) guides or their published source | Account CSV and Enterprise API instructions branch independently; neither is illustrated with an unrelated invoice screen. |
| VMware / files | No exact supported collector/vCenter deployment profile image verified | Do not use generic vSphere images as proof of this collector's setup. One-time file input has no provider console; recurring storage needs its own verified profile. |

Direct image URLs are maintained once in connection-setup.html's consoleEvidence catalog. Reviewers downloaded and inspected AWS/Azure/Alibaba images temporarily; no private console was opened. Their screenshots supply navigation comparison only, not IAM, FOCUS conformance or financial acceptance evidence.

# Remaining acceptance

Current actual-console capture coverage is incomplete for GCP, OCI, IBM, VMware and some permission steps. Acquiring current authorized sanitized screenshots is an explicit requirement before claiming complete visual coverage; image absence is not waived. Real template hosting/deployment, narrow permission testing, current console walkthroughs and Outputs read-back are NOT_RUN. Screenshots must be refreshed when the documented provider flow changes, with source/date and mismatches re-reviewed.

# Owner feedback: keep this draft and defer visual refinement

The owner explicitly asked to keep the current draft and record the following requirements for later detailed work. This is not final visual acceptance and does not authorize another redesign in this change.

- Remove the perceived Claude-style presentation and align onboarding with the product-wide look and feel. The current layout is a functional study, not an approved visual system.
- Design the experience around the customer viewing and working through the actual provider console while following setup. A panel of reference screenshots alone does not satisfy that intended experience.
- Define the later console-opening, side-by-side guidance, step synchronization, return/resume and verification interaction in detail. Do not infer permission to control a customer's live console or capture private account data.
- Retain the present implementation as a held draft. The owner's instruction to continue resumes documentation and integration of this work; it does not revoke the explicit deferral of detailed visual design.
