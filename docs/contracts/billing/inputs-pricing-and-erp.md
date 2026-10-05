---
type: Billing Contract
title: Eligible source revisions pricing warnings arithmetic and ERP profiles
description: Specifies proposed delivery semantics and deterministic financial procedures with independent synthetic expectations and unresolved policy decisions.
status: draft
---

# Input and financial contracts

The [payment and recognition supplement](payment-and-recognition.md) preserves contract Payment Terms, overdue authority and monthly ERP cost recognition independent of customer billing cadence, including PT-01–08 and ERPC-01–08. It does not select actual accounting/tax policies or certify ERP capabilities.

Part of [HB-11](../../reviews/HB-11-refinements/resolution.md); supplements PC-01/02/04 and ED-04/05. Source basis is HB-6/HB-8/HB-10, B0110/B0502/B0836–B0842 and local AR-06/DB contracts. These are draft planning contracts, not selected rates, accounting policy, technology versions or certified adapters. [Usage detail](../../architecture/usage-detail-lifecycle.md), Claim uniqueness and purpose-specific monthly recognition remain required.

## ED-05: eligible dataset publication

Every eligible publication mode below requires the [cross-validation and operator-confirmation gate](../../architecture/data-collection.md#accuracy-cross-validation-and-operator-confirmation). This proposed HB-13 refinement binds confirmation to the exact candidate revisions and independent comparison evidence, rechecks current scoped authority at publication and invalidates stale decisions. Complete snapshot, append and hybrid paths cannot bypass it. Confirmation grants no amount-approval or issuance authority and cannot waive missing required evidence or unresolved conflicts.

A source profile must declare provider/authority/account scope, economic record identity, delivery scope, completeness criteria, mode, revision order and cross-transport precedence. Receipt ID, source economic identity, run ID and Claim are distinct. Missing/incomparable identity or conflicting versions block publication; request deduplication alone cannot prove economic uniqueness.

| Mode | Eligible revision contract |
| --- | --- |
| Complete snapshot | Verify all required pages/files/coverage and register immutable partitions; publish the complete scope revision atomically in authoritative manifest metadata. It supersedes that scope's prior eligible snapshot; omitted records cease to be eligible in the new revision. Never rewrite previous approved manifests |
| Incomplete snapshot | Keep protected received evidence and incomplete status; do not publish complete replacement or delete omitted rows. Previous complete history can remain readable, but cannot silently substitute for a required current revision |
| Append | Admit uniquely identified economic records once, including explicitly linked signed correction/reversal net effects. A replacement total is not a delta; repeated correction delivery is not an additional adjustment |
| Hybrid | Declare scope/period mode, transitions and correlation of closed-period corrections. Mode cannot switch because a file happens to arrive later |

Out-of-order revisions cannot roll eligibility backward. Identical API/upload evidence maps to one economic dataset; different payloads under one identity require conflict/correction handling. Stage bytes and verify them before manifest publication; cross-store writes are not assumed atomic. Concurrent publishers use expected current scope revision and reconcile losers rather than overwriting. Late eligible revisions create new calculation/impact review; approved/issued history remains linked and immutable.

## PC-01: default calculation and approved exceptions

Preserve the source default: **quantity/target determination → base selling price → contract discount → additional fees → minimum/maximum → currency conversion → selling tax/rounding**. Included quantity is applied during quantity determination using the approved contract accumulation scope. Tier/allowance/minimum pools use complete economic coverage, not the user's filter or processing chunk. Pricing modes/components and actual policies remain to be selected.

Each stage records fixed policy revision, operands, units/currencies, component basis, intermediate/result and applicable rule trace. Explicit profile/contract exceptions require an approved sequence revision with effective dates and customer-impact evidence. Changing sequence is a price change. Equal-priority competing rules at the same stage block activation; insertion order cannot select a winner. Changed affected policy/input/result invalidates draft approval; issued periods require linked corrections.

Use existing AR-06 FX precedence, permitted fallback and source/customer currency distinctions. Keep published FX, sales markup and conversion fees separate. Already converted supplier amounts and customer-currency fixed fees cannot receive unintended second conversion. Choose precision/magnitude/scales, rounding points/mode, tax basis and residual allocation in a named finance/tax decision packet; no defaults are inferred from a database type.

## PC-02: evidence-bound warning acknowledgment

A versioned warning policy declares category, eligibility predicates, scope, effective dates, acknowledgment role/ceiling and required explanation. Undefined warnings are not automatically waivable. Unwaivable blockers remain blockers despite global approval, model recommendation or acknowledgment.

An acknowledgment binds warning identity/category, policy revision, affected scope, input/result evidence digests, actor/time and reason. It accepts specified evidence, changes no amount and grants no issuance authority. Recheck policy/current actor authority/evidence before approval and execution. Changed affected evidence or policy requires renewed acknowledgment; an irrelevant change exception needs explicit canonical proof. Expired/revoked/stale acknowledgments cannot satisfy approval. Preserve prior records as history rather than mutating their evidence binding.

Policy approval → eligible inputs/calculation/reconciliation → warning resolution → fixed amount approval → separately authorized issuance/submission → authoritative external confirmation are distinct states. Undefined deterministic semantics, failing monetary oracles or unresolved completeness cannot be waived through this sequence.

## ED-04: analytical monetary processing

Select one reviewed profile before financial use:

| Profile | Permitted monetary work and failure boundary |
| --- | --- |
| Precomputed authority | Approved authoritative calculation creates monetary components; analytics performs reviewed exact grouping/sums only over matching eligible revision/currency/basis. Prevent duplicate versions and join fan-out; reject unsupported casts or conversions |
| Analytical expressions | Enumerate allowed expressions, operand/result types, intermediate magnitude/scales, rounding points and aggregate result types for the exact engine/version. Reject unlisted operations, overflow or incompatible truncation/scales |

Storage round trips or wider decimals cannot establish arithmetic compatibility. Float/statistical results cannot feed financial approval/reconciliation without a separately approved compatible monetary contract; forecasts remain distinct. Independently derived datasets/amounts must agree with authoritative components. The [access contract](../security/access-and-trust.md) still applies to precomputed amounts and aggregates.

## PC-04: source-dated ERP profile and current support

Record product/version/deployment, activated modules, actual methods/schema/permissions/plan conditions, company/customer/product/tax mappings, price authority, call boundaries, external identities and uncertainty lookup in each adapter profile. Source-era conditions, current official documentation and actual supported-environment evidence are separate columns; a historic source statement or successful mock cannot certify support.

Historical B0836–B0842 inputs include Odoo 19 JSON-2, actual database `/doc` and permissions, online Custom-plan conditions and separate self-hosted checks. Restore those as dated requirements to verify; no current Odoo certification is asserted. Each API call has its own boundary; multiple calls are not assumed atomic. The selected supported method/extension must bind the final approved amount/version to guarded posting or use an approved reconciled/manual alternative. If intervening changes/unknown results cannot be controlled, automatic posting is unsupported, not silently approved.

SAP request/EBDR, billing document and FI posting have separate external identities and statuses. Request acceptance or Journal Entry success alone cannot establish final customer invoice issuance. Actual price-condition reapplication and final monetary results require reconciliation; mismatches invalidate affected approval before posting. S/4HANA variants, ECC and Business One Service Layer are distinct profiles/adapters. Preserve recognition-versus-customer-invoice purposes and approved corrections; timeout is unknown pending lookup/reconciliation, never a blind resend or inferred failure/success.

Current vendor/deployment/method compatibility remains UNVERIFIED. First actual profiles and sandbox evidence are decisions; unsupported adapters must be explicitly classified. This supplement issues no ERP command.

## Independent synthetic expectations and acceptance

Numbers are synthetic reasoning examples, not actual rate/tax/rounding choices or executed product tests. All cases are NOT_RUN.

| Case | Input / independent expected result |
| --- | --- |
| H11-S01 | Complete `[A=10]` → complete `[A=10,B=5]`: eligible total 15, not 25. Then complete same-scope `[B=5]`: 5; old approved history preserved |
| H11-S02 | Partial `[B=5]` missing required coverage publishes no complete replacement or inferred omission deletion. Older complete revision cannot roll a newer eligible total back |
| H11-S03 | Complete total 15 plus one unique linked append correction −2: 13; identical retry remains 13; identical API/upload source cannot yield 30 |
| H11-S04 | Conflicting overlap or unspecified hybrid transition blocks publication until approved precedence; no arrival-order winner |
| H11-M01 | Base 100, discount 10%, fee 20, minimum 120: `max(100 × 0.90 + 20,120)=120`. Moving minimum first gives 128 and requires a sequence revision; same-stage priority conflict blocks activation |
| H11-M02 | Quantity 120 units, included quantity 20, synthetic rate 2: `(120−20) × 2=200`; allowance not reapplied by chunks, views or price stage |
| H11-M03 | Future exception and retroactive correction preserve effective policy/history; changed affected input invalidates draft amount approval; no second FX on already converted component |
| H11-W01 | Permitted current evidence-bound acknowledgment may satisfy only its approved warning; unwaivable/undefined blocker, revoked actor or global-approval bypass cannot |
| H11-W02 | Changed warning/input/result/policy requires fresh acknowledgment; prior actor/time/reason remains auditable |
| H11-A01 | Under an illustrative four-decimal nearest rule, `2/3=0.6667`; an engine returning truncated `0.6666` fails that hypothetical profile. No production rule selected |
| H11-A02 | Exact `100.25+0.75−1.00=100.00`; near-bound multiplication/sums/cancellation/conversion and Decimal-to-Float paths must match independently approved expectations or reject before financial use |
| H11-A03 | Duplicate joined components, mixed currencies/bases/revisions and unsupported types cannot silently produce a valid monetary aggregate |
| H11-E01 | Historical Odoo/SAP conditions remain dated/unverified until profile evidence; mock success cannot switch support state to certified |
| H11-E02 | Odoo multi-call intervening change, partial call failure and timeout preserve approval/version guard or explicitly unsupported automatic path; unknown result reconciled without duplicate effect |
| H11-E03 | SAP request, billing and FI outcomes displayed independently; price change requires reconciliation/reapproval, not assumed final invoice |

See [HB-11 decision packet](../../reviews/HB-11-refinements/resolution.md) for actual numeric, warning, source, ERP and analytical choices. Preserve source sequence while separately approving actual profile exceptions; draft publication alone activates none of these policies.
