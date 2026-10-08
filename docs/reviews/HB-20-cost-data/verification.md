---
type: Design Review Report
title: HB-20 cost data architecture review and verification
description: Records the design review scope, findings and verification limits for the updated cost analytics proposal.
status: draft
---

# HB-20 data review

Tracking: [HB-20](https://linear.app/hybrid-billing/issue/c04558aa-62af-4914-a71e-785049be5fcb). Accountable owner: Seung Woo Park. Root owns PM/planning, writing and QA; separate read-only DBA owns independent design review. Machine: `machine:mac_mini`. Branch: `docs/HB-20-cost-data-review`, isolated from `origin/main` at `e436a49`. The correct Hybrid Billing workspace/team/project, assignee, In Progress and machine label were verified in browser UI before repository work. The connector previously pointed to another workspace and was not used to update it.

## Superseding comprehensive review

The following narrow review describes commit `5952b91` only. It is not complete architecture acceptance. The owner subsequently excluded MongoDB, preferred object storage/Parquet and requested comprehensive review plus performance, maintainability and operations research. See the [full review](full-review.md), [revised architecture](../../architecture/data-ownership/object-storage-data-architecture.md) and [comparative research](../../architecture/performance-operations-comparison.md). Independent architect, DBA and QA passed the revised logical proposal; physical design and runtime remain OPEN / NOT_RUN.

## Historical narrow scope and findings

Scope: new [cost data proposal](../../architecture/data-ownership/cost-analytics-data-design.md), this report, and cross-reference/sizing updates to the existing logical model, readiness, detail lifecycle and payment/recognition contract. No product code or database changes.

Baseline independent review found two material gaps: the previous >1M/day premise conflicts with the new >100M/day owner condition; existing entity-family documentation does not fully specify analytical fact grains, tag multiplicity and additive measures. The updated proposal separates usage, supplier cost, allocation, customer charge, recognition and receipts; records exact grain/basis/temporal identities; requires fan-out-safe queries and versioned publication; and synchronizes the active architecture/lifecycle/payment sizing references. Historical review reports remain historical evidence.

Proposed conclusion: SQL transactional authority plus SQL columnar analytics and protected semi-structured evidence is appropriate for review. This historical suggestion of a conditional document database is superseded: MongoDB is excluded; the revised proposal uses object-backed semi-structured evidence. This is not product adoption or benchmark evidence. Original scope remains; HawkEye and TokenMeter integration are deferred, not erased.

## Historical narrow verification status

Independent current-document review: PASS. Separate read-only DBA reviewed the full draft and actual existing-document diff. A MEDIUM recovery issue was fixed and re-reviewed: an older verified analytical release cannot replace corrected canonical eligibility or be displayed as current/final. Scoped manifests reuse immutable partitions. Final review found no remaining blocking findings in scope. Grain, bases, temporal dimensions, tags, publication/access and the updated capacity premise passed design review; synthetic expected examples were reasoned about, not executed.
Root document QA: PASS for relative links in all six changed files, required frontmatter type, file sizes (each below 1,500 lines), credential-pattern scan and whitespace. The actual six-file staged diff was inspected for unrelated content and sensitive/generated assets; none found. These are document checks, not product tests.
Runtime, exact engine behavior, load/recovery, tenant enforcement, money/concurrency and provider compatibility: NOT_RUN.
Canonical local v1.16 requirement/work/acceptance integration remains follow-up; the isolated published baseline does not contain all local planning artifacts. No claim of planning closure or complete canonical synchronization is made.

## Resumed research verification

Separate architect and QA reviewed the complete performance/operations comparison: scoped research PASS, no blocking findings. The full revised logical proposal separately received architect, DBA and QA PASS. Physical readiness remains OPEN; all 17 benchmark profiles are NOT_RUN. Root checked relative links and nonempty frontmatter type for all eight changed Markdown files, `git diff --cached --check`, scoped staged filenames and sensitive-pattern scan (zero matches). Nine staged files are documentation or input fingerprints; no product code, customer records or generated binary assets. These checks are document hygiene, not runtime/security certification.

## Handoff

Commit `5952b91` was pushed, but PR creation failed with `must be a collaborator (createPullRequest)`. No PR exists from that attempt. Root will release the resumed machine claim at handoff. Authorized merge/read-back and ticket completion remain pending. The ticket must remain open until authorized integration. Planning completion is not declared.
