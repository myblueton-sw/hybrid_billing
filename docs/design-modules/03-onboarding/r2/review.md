---
type: Review
title: HB-49 AWS FOCUS onboarding design review
description: Records independent design findings and document checks without claiming deployed AWS or UI validation.
status: stable
---

# Ownership and result

Ticket: [HB-49](https://linear.app/hybrid-billing/issue/HB-49/document-minimal-action-aws-focus-onboarding). Root is sole writer for 03-onboarding/r2 and the baseline README link. The separate read-only aws_onboarding_review agent reviewed the design direction, actual README/research and scoped corrections against canonical onboarding, security and input-publication obligations. Final verdict: PASS for design; live implementation and visual prototype remain NOT_RUN. Machine: mac_mini.

# Findings resolved

| Finding | Resolution | Re-review |
| --- | --- | --- |
| A new export with no objects cannot prove a successful GetObject read | Initial status explicitly verifies identity/export metadata/list access only; operational object read stays pending until delivery | PASS |
| Setup permission cleanup needed explicit evidence | Verify removal/expiry of temporary grants and show pending when unverified; preserve export writer and customer-managed standing administrator rights | PASS |
| Eligibility table could imply required source reconciliation happens later | All source/billing-basis comparisons precede eligible publication; later customer pricing/amount/issuance approvals remain separate | PASS |

The reviewer also confirmed early connection is a draft rather than completion of OP05 financial onboarding, raw shared-payer custody must be authorized before download, discovered accounts do not grant access, confirmation is current-authority/revision bound, and the simple action budget does not invent AWS OAuth or omit workload-identity prerequisites.

# Verification

Root document QA: PASS. Relative source links resolve; Markdown documents have nonempty type metadata. git diff --check passes. Registry and historical HB43 design sources are unchanged. Original English and Korean v10 PDF SHA-256 values match the frozen registry. Actual scoped content was inspected for unrelated work and secrets; no real AWS accounts, credentials, customer data, deployable template or cloud mutation are included.

A01–A14 are acceptance scenarios, not executed tests. IAM policy deployment, source adapter, actual CloudFormation console interactions, AWS account/period evidence, independent monetary reconciliation and interactive visual accessibility remain NOT_RUN. Existing delivery-checklist boxes remain unchecked. Design review does not certify FOCUS conformance or billing readiness.

# HB-51 multi-provider interface review

Root expanded the same r2 draft with provider profiles and a Korean interactive prototype, preserving the AWS-specific implementation design. The owner requested a multi-provider view centered on actual data integration and schema validation. No final-version acceptance is inferred and no duplicate r3 copy is created.

Independent actual design/code-semantics review and scoped correction re-review by aws_onboarding_review: PASS. Findings resolved: Alibaba analysis-only FOCUS must use separate official bills for settlement; OCI reports reside in Oracle-owned storage; VMware schema errors use metering units rather than a misleading currency field; schema table includes transformations and profile context; existing/new route state is reset; explicit navigation moves focus to a visible native control. The reviewer independently corroborated IBM native FOCUS support and inspected a desktop screenshot, but did not independently rerun the browser suite.

Root browser QA: PASS for 192 provider/state/width/theme combinations (8 routes, 4 states, widths 1024/360/320, light/dark), with account input validation, existing-route/back/new-route regression, action feedback, persistent synthetic labeling, zero page errors and no outer horizontal overflow. State echo initially reset action feedback; it was corrected by suppressing identical host restoration. The test harness was adjusted to respect persisted state on reload. The final matrix passed after provider-boundary corrections. Root visually inspected desktop and mobile/dark captures; this is mockup verification, not accessibility certification.

Root document/source QA: PASS for relative links, OKF metadata, source/output byte equality and git diff --check. Prototype has no explanatory source comments or external-request APIs. Optional host calls only persist non-sensitive provider/screen/scenario choices. No live files, credentials, AWS deployment, OAuth, provider API or billing approval is executed.

The preview can be rendered from prototype.html using the visualize skill renderer; local review output is output/HB51/preview.html, screenshots desktop.png and mobile-dark.png, and summary ui-check.json. Generated output is not committed. Source digest for this review: `fecc8b4a5d4a9ea9d7d99d6fa67740266cd91f7077b4f622c348bb595783e6cd`. The final conversation preview uses the matching output/HB51/focus-onboarding.html copy.
