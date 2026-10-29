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
