---
type: WorkBrief
title: Allocation profile and approval workflow design
description: Defines the HB-57 design scope, ownership, acceptance and sequenced implementation boundary.
status: draft
---

# Goal and ownership

[HB-57](https://linear.app/hybrid-billing/issue/HB-57/define-evidence-bound-allocation-profiles-and-approval-workflow) follows the allocation readiness audit. Accountable owner: Seung Woo Park. Machine: mac_mini. Root owns PM/planner, documentation, workflow design and document QA. A separate read-only DBA reviews financial design and actual changes. Root is the sole writer; no concurrent file writers.

Owned files are `docs/contracts/billing/allocation-profile.md`, this directory and a discoverability link in `docs/contracts/billing/inputs-pricing-and-erp.md`. Branch: `docs/HB-57-allocation-profile`, created from fetched `origin/main` at `b48b23c`. Existing untracked planning and module drafts are preserved.

# Evidence and acceptance

Use [numeric/Claim](../../contracts/billing/claim-and-numeric.md), [financial gates](../../contracts/billing/inputs-pricing-and-erp.md), [cost obligations](../../contracts/billing/source-pricing-and-cost-obligations.md), [automatic pricing](../../reviews/HB-42-auto-pricing/requirements.md) and [shared command semantics](../../contracts/commands/planning-actions.md). The [HB-56 audit](https://github.com/myblueton-sw/hybrid_billing/blob/b3cd86a/docs/reviews/HB-56-allocation/report.md) is an unmerged reviewed input at this immutable revision; PR #52 is not merged by this ticket.

Deliver an implementable profile field contract, evidence/approval invalidation rules, operator review journey and independent expected cases. Protect whole-pool conservation, deterministic residuals, embedded-benefit identity, idle/unattributed amounts, current tenant authority, immutable history and finite-benefit coverage. Keep provider costs and selling prices separate. Missing commercial terms block activation, without preventing synthetic calculator development.

Plan review precedes contract writing. Actual contract review follows, with corrections re-reviewed. Root validates source consistency, relative links, OKF metadata, expected-case arithmetic, whitespace and staged scope. Documentation checks do not execute product acceptance or visual QA.

# Sequenced delivery

1. This ticket: profile and operator workflow contract, independent financial reviews and reviewable PR.
2. Next ticket: pure bounded exact calculator implementing the selected `largest_remainder_v1` contract, explicit profile input validation and independent fixtures/property tests. It returns a candidate, with no approval or persistent economic effects. Select implementation language from actual owning service requirements; this design does not introduce a stack.
3. Separate authority ticket: immutable manifests, evidence/approval revision guards, current authorization and durable Claim integration, with database race/replay/recovery tests. No in-memory test may substitute for those guarantees.
4. Bind the existing pricing review journey to real services and verify screen states, accessibility and customer disclosure.

No live financial profile, actual rate, tax, tolerance, ERP posting, customer data access, production deployment or merge permission is selected. Runtime and visual completion remain UNVERIFIED. The planning-completion decision permits ticketed development; historical pre-completion wording does not reopen that gate.
