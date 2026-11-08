---
type: AllocationReadinessReview
title: EDP discount and shared-cost allocation readiness
description: Separates existing allocation contracts and screen examples from executable implementation and identifies the gates required before financial use.
status: draft
---

# Scope and conclusion

[HB-56](https://linear.app/hybrid-billing/issue/HB-56/audit-edp-and-shared-cost-allocation-correctness-and-implementation). The owner prioritized allocation correctness while resuming missing-screen work. This review concerns product EDP, procurement benefits, customer pricing and shared-cost allocation, not contributor sponsorship distributions. Root owns this report and its baseline index link; Seung Woo Park is accountable owner; machine is mac_mini. A separate read-only DBA reviews the evidence and actual report.

**Specified does not mean implemented.** Existing contracts define substantive allocation safeguards, including a proposed deterministic residual algorithm. Current pricing screens illustrate those rules. Inspection of maintained `modules/`, `packages/`, `migrations/`, `tests/` and `tools/` did not locate an executable EDP/customer-pricing/whole-pool allocation engine or its acceptance tests. Existing ingestion, exact source totals, account attribution and authorization code are useful prerequisites, not proof of financial allocation. Financial runtime readiness is **UNVERIFIED**.

This is a scoped readiness audit and next-step proposal. It changes no financial policy, source data, runtime code or frozen screen. Historical planning-gate language in older contracts does not override the [planning-completion decision](../../planning/planning-completion.md); actual financial policy values and implementation acceptance remain separate gates.

# Existing evidence and its limit

| Concern | Existing contract or implementation evidence | What it establishes / does not establish |
| --- | --- | --- |
| Supplier versus customer amount | [HB-38 pricing design](../HB-38-billing-operations/pricing-design-review.md), four separate records and calculation sections | Provider observation, purchase agreement, customer selling policy and benefit ledger are separate. Already-net procurement cost must not receive the same procurement discount again. Customer resale may use another approved basis. This is design evidence. |
| EDP attribution and eligibility | [HB-38 provider research](../HB-38-billing-operations/provider-pricing-research.md), AWS enterprise discounts section | Actual agreement, eligible spend, period, exclusions and export representation are necessary; list-minus-net is not evidence of EDP cause. This audit does not freshly verify provider documentation or private terms. |
| Exact arithmetic and residuals | [Claim and numeric contract](../../contracts/billing/claim-and-numeric.md), numeric profile and whole-pool allocation sections | Proposed exact bounds, quantum/mode, selected algorithm, stable target-ID ties, signed conservation and immutable result manifest. Actual profiles are not activated by this audit. |
| Shared costs and unused burden | [SP-01/02/04/07](../../contracts/billing/source-pricing-and-cost-obligations.md) | Contract selects pass-through, savings share or independent sale; idle/unattributed burden is explicit. Whole-pool allowances survive view filters and entity moves. Accounting cost and external sale are not summed. |
| Calculation order and approval | [PC-01](../../contracts/billing/inputs-pricing-and-erp.md), [AUTO-03–09](../HB-42-auto-pricing/requirements.md) | Ordered stages, explicit exclusion/override semantics, fixed input revisions, stale approval invalidation, one-off grant identity and linked corrections. AUTO requirements expressly do not implement the pricing engine. |
| Source numeric evidence | [FOCUS model](../../../modules/focus_import/model.py), [semantic validation](../../../modules/focus_import/semantics.py), [consumed registry](../../../packages/contracts/focus_import/consumed.json) | Exact source numeric validation and cost/quantity fields with commitment status. The source grammar/storage profile is not the proposed pricing numeric profile; do not reuse its exponent handling or fixed scale as implicit pricing policy. |
| Source grouping and customer view | [Company views](../../../modules/focus_import/views.py), [usage profile](../../../modules/customerusage/profile.go) | Authorized source grouping and quantity-only customer projection. Filtering these views is not an allocation denominator. Neither path calculates customer selling amounts or grant consumption. |
| Existing verification | [NM-04/05 and CL cases](../../contracts/billing/claim-and-numeric.md), [pricing cases](../HB-42-auto-pricing/requirements.md), maintained tests | Contracts mark product cases NOT_RUN; pricing numbers are synthetic screen examples. Ingestion/access/engine-probe tests do not establish allocation acceptance and were not rerun as substitute evidence. |

# Required calculation boundary

The following sequence consolidates existing obligations for the next design/implementation work; it does not select an EDP rate or new global commercial policy.

1. Pin the eligible source revision and its amount basis, currency, period, supplier account and economic identities. Reconcile supplier evidence independently of customer selling prices. Keep billed, effective and pro-forma views distinct.
2. Establish each benefit's identity, origin, eligible scope and whether it is already embedded in the chosen cost. Preserve authoritative source attribution when available; require an approved policy for a genuinely pooled amount. A general provider discount is not automatically an additional customer sales concession.
3. Freeze the complete pool and recipient identities, weights, unit, denominator and effective membership. Record non-customer idle/unattributed components separately. No arbitrary customer is used to absorb missing mapping, a zero denominator or unknown cost.
4. Calculate using the approved numeric/algorithm revision. Where selected, `largest_remainder_v1` quantizes the total, allocates exact integer quanta by nonnegative weights, resolves equal remainders by canonical target ID and restores the total sign. It rejects unsupported caps/minima rather than silently solving a different problem.
5. Reconcile the signed recipient sum against the quantized pool target, with the original-to-quantized rounding component separately recorded. Preserve idle/unattributed cost within the cost reconciliation; its existence does not make it customer-billable. Never reconcile by adding customer sales and internal cost allocations together.
6. Apply the separately approved customer selling policy and its stage order, benefit pass-through, exclusions, fees, FX and tax treatment. Display the origin of each charge and adjustment. Missing provenance, conflicting rules or unapproved policies block the affected calculation.
7. Publish an immutable complete result manifest with source, pool, policy and target revisions. Preview does not consume a finite benefit. Reservation/application/reversal requires the authoritative economic-coverage transitions; retries and unknown external outcomes cannot make the same grant available twice.

# Gaps that block a correctness claim

| Priority / gap | Required resolution before financial acceptance |
| --- | --- |
| P1: no located allocation execution path | Implement a reviewed pure calculator and then its authoritative orchestration. Identify actual entry points and tests; a static amount or UI button is insufficient. |
| P1: policy activation is not concrete | For each applicable agreement, resolve recipient eligibility, amount basis, weights, exclusions, pass-through, idle bearer, period/timezone, numeric bounds, quantum, stage order and residual policy. Unknown fields block activation; no universal default is inferred. |
| P1: benefit evidence and double counting | Carry benefit/source identity and representation through the pipeline. Verify embedded-net versus separate-benefit alternatives and partial eligibility. Reconcile purchase evidence separately from customer sales and commitment utilization. |
| P1: durable allocation and finite-benefit state | Implement whole-pool manifest publication, expected-revision checks, economic identity, all-or-nothing reservation boundaries, replay conflict behavior and reversal lineage. A run ID or retry key alone is insufficient. |
| P1: independent runtime expectations | Execute signed conservation, deterministic residuals, full-pool stability, scope isolation and concurrency cases against the actual implementation. No current runtime PASS is supplied here. |
| P2: operator review detail | Add a connected review of pool total, eligible/excluded targets and reasons, basis/weights, idle remainder, rounding recipient, old/new amount and policy approval. Customer output exposes only authorized selling explanations, not other recipients or supplier margin. |

The older HB-38 summary says cancellation releases a reservation and reversal restores an event. Implementers must use the more precise [Claim transitions](../../contracts/billing/claim-and-numeric.md): pinned/unknown work requires authoritative no-effect or reconciled cancellation evidence; confirmed effects require governed reversal; replacement eligibility is explicit. Cancellation or worker expiry must never be treated as unconditional release.

# Proposed acceptance cases

All product outcomes below are **NOT_RUN**. Constants are synthetic, not approved customer policy. Assertions belong to the future actual calculator and authority tests, with independent expected fixtures.

| Case | Input / action | Required result |
| --- | --- | --- |
| AL-01 | Approved pooled benefit USD100; eligible weights A=600, B=400; C explicitly excluded; full pass-through selected only for this case | A=60, B=40, C=0. Store eligibility reasons and basis. Changing the UI filter to A must not change A to100. |
| AL-02 | Reported supplier net900 already includes a100 procurement benefit | Preserve900 as that net basis; applying the same procurement benefit again to obtain800 is rejected. Customer resale remains a separate contractual calculation. |
| AL-03 | Total0.05, quantum0.01, equal A/B/C weights, canonical order A<B<C, selected largest remainder | A0.02, B0.02, C0.01; sum0.05. A negative total gives the sign inverse. Permuting input or worker order preserves the result. |
| AL-04 | Zero total with valid positive total weight; individual zero weight; all-zero weights; duplicates; negative weights | Zero target gives zero results; zero-weight target gets zero when another weight is positive. All-zero weights, duplicate identities and negative weights reject. Missing weights are not zero. |
| AL-05 | Capacity pool1000, approved weights A30/B50/operations-idle20 | A300, B500, idle200; exact total1000. A separately approved usage-basis revision could yield375/625; selecting that basis is a policy change, never a silent fallback. |
| AL-06 | Same amount under mixed currencies, missing numeric policy, caps/minima unsupported by selected algorithm, incomplete pool or unknown attribution | Block the affected final calculation. Do not add currencies, infer precision or silently redistribute missing scope. Unattributed cost remains visible and not automatically billed. |
| AL-07 | One shared grant100, concurrent invoice attempts consume80 each; rerun with another job ID; same request identity with changed inputs; timeout after dispatch | At most one conflicting80 application succeeds; replay does not consume again and changed-input replay conflicts. Unknown dispatch remains pinned until reconciled. An in-memory calculator test cannot prove this database behavior. |
| AL-08 | Corrected source/weights after approval or issuance; group move mid-period; repeated or overlapping reversals | New draft revision and reapproval where affected; issued history unchanged with linked correction. Partial reversal references original per-recipient applications and coverage, not today's weights. Replacement eligibility requires explicit authorization, never automatic freeing. No reset of grant/allowance identity or double release. Apply effective-dated membership and explicit proration. |
| AL-09 | Partial product concession plus general sales discount, or missing benefit provenance | Exclude from the general sales concession only the portion already carrying a benefit in the contract-defined exclusion group. Procurement EDP alone does not establish membership in that group; missing provenance blocks instead of assuming no discount. HB-41/42 example remains1535, not the unauthorized1482. |
| AL-10 | Wrong MSP/customer recipient, revoked actor, result viewed from a customer session | Deny unauthorized calculation/publication/read, with no other-customer membership or procurement-margin disclosure. Review authorization independently of mathematical conservation. |

Property-based tests should cover bounded approved profiles: signed conservation, stable-ID permutation invariance, zero-weight behavior, exact positive rescaling of weights, and reproducible fixed-input output. Do not assert arbitrary split/merge invariance or independent chunk allocation equivalence: rounding and pool policy can invalidate those properties. Exercise rejected inputs as well as valid cases. Durable replay, concurrency, publication and revocation need separate integration tests.

# Next work and verification record

Recommended sequence: evidence-bound allocation profile and connected operator review → independent financial design review → exact calculator with independent cases/properties → authoritative result/grant state and integration tests → screen binding and QA. Existing administrator-screen gaps remain open; this audit does not complete them. Provider-specific private terms remain unknown until supplied through the authorized process.

Root inspected the linked financial contracts, pricing requirements and implementation files, and searched the maintained implementation/test directories for EDP, allocation, discount, rounding and Claim mechanisms. This is a scoped local source inspection, not an exhaustive proof about untracked tools or external systems. No customer account, provider API, real invoice or production database was accessed. No runtime allocation test or visual QA was performed.

Independent DBA design/evidence review: PASS for the documented safeguards and implementation-coverage classification; financial runtime readiness remains UNVERIFIED. Initial actual-report review found no blocking issue and recommended more precise reversal and sales-exclusion cases; those clarifications are incorporated above. Final changed-case re-review: PASS for replay, reversal and exclusion cases and the verification scope. Root document metadata, local relative-link targets and whitespace checks: PASS. No runtime tests were run for this documentation-only change.
