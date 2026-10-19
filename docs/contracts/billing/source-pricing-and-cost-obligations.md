---
type: Billing Contract
title: Source-derived pricing commitment cost-pool and lifecycle obligations
description: Restores omitted or weakened v1.16 source obligations for selling methods, commitments, FX versions, on-premises cost pools, contract pools and service lifecycle events.
status: draft
sources:
  - resource: ../../../통합_빌링_플랫폼_구현_기획안_v1.16.docx
  - resource: ../../reviews/HB-22-source-omission/findings.md
---

# Source pricing and cost obligations

Part of Linear ticket [HB-22](https://linear.app/hybrid-billing/issue/HB-22); requirement group HB-22 (prepaid credits and disputes) is a different identifier. Each section restores a source clause that the [source omission review](../../reviews/HB-22-source-omission/findings.md) found missing or weakened. `B` locators are zero-based DOCX body-child indexes from [source inventory](../../planning/v1.16/source-inventory.json); `.R` is a table row. This is a planning contract under the [planning gate](../../planning/v1.16/planning-gate.md): proposed names are not tables, APIs or implemented behavior. Customer, contract and finance values stay decisions; this document selects none.

It supplements, and does not replace, [inputs, pricing and ERP](inputs-pricing-and-erp.md), the [Claim and numeric contract](claim-and-numeric.md) and the [shared domain contract](../../planning/v1.16/domain-contracts.md). PC-01 calculation order, AR-06 FX precedence and complete-pool residual rules continue to apply.

## SP-01: selling methods and savings basis (P2-01, P2-10)

Source: B0545.R001–R004, B0548, B0549, B0566, B0569, B0571, B0572, B0574, B0576, B0581.

A selling policy revision declares exactly one selling method per component, with its required evidence:

| Method | Formula basis | Activation precondition |
| --- | --- | --- |
| Effective-cost pass-through | Allocated cost + contractual unused-commitment share + fee | Cost evidence and an unused-burden contract term |
| Savings share | Comparison basis − pass-through rate × approved savings + fee | Comparison unit price, scope and loss treatment defined |
| Independent selling price | Billed quantity × customer unit price + add-ons | Exact quantity and contract price |
| Prepay recovery plus usage service | Commitment purchase/prepay recovery + separate usage service fee | Recovered-cost ledger preventing repeat recovery |

Rules:

1. A savings figure uses the provider-supplied value and comparison basis first. A contractual difference is computed only from an evidenced basis amount and effective cost, and is labelled as an MSP contract calculation. The platform does not reproduce provider list prices or forecast savings. **When no evidenced basis exists, the savings-linked charge is held**, not estimated.
2. A savings-share policy states whether savings are usage-only or net of unused burden. The pass-through, MSP burden or floor for negative savings is a contract policy decision. Each rule fixes its input basis and stacking group so an already-discounted cost cannot receive the same discount again.
3. A loss result never raises a customer rate automatically. It raises a warning under the PC-02 acknowledgment policy and requires approval.
4. Whether a fee also applies to the unused burden is a separate, explicit setting. It is never inferred from the usage-fee setting.
5. A prepay already recovered under the selling contract is not billed again monthly. The platform does not recalculate provider amortization. Unrecovered balance and separate service fees follow the contract.
6. Unattributed amounts are never billed to a customer automatically. COM04 conserves received cost against allocation totals; it does not require same-month BilledCost and EffectiveCost equality.

## SP-02: commitment ingestion by provider (P2-03)

Source: B0523.R001–R006, B0526, B0527.

Provider profiles for commitments carry these ingestion guards. They are acceptance obligations; provider capability is verified per contract.

| Provider | Retained evidence | Guard |
| --- | --- | --- |
| AWS RI | Reservation identity, EffectiveCost, upfront allocation, recurring fee, unused cost | Do not add the usage row's effective cost and the commitment fee together again |
| AWS Savings Plans | CoveredUsage, Negation, RecurringFee, UpfrontFee, effective cost | Keep billing-row offsets separate from cost allocation |
| Azure | Actual and Amortized cost, reservation/savings-plan identity, unused | Never sum the two cost views; confirm available fields per agreement |
| Google Cloud CUD | Commitment fee, usage, credits, project attribution and attribution model | Preserve attribution model and its change time; never double-deduct credits |
| OCI, Alibaba Cloud | That contract's commitment, prepay, credit and usage detail | Detailed mapping is undecided; support only after per-product verification |
| VMware, on-premises | Customer license, lease, reserved-capacity contract and metering | Not treated as a cloud commitment; allocated through the internal cost pool (SP-04) |

Keep BilledCost and EffectiveCost separately. Validate BillingCurrency, charge and billing period bounds, ChargeCategory and CommitmentDiscountId, Type, Category and Status against the declared FOCUS version. Record provider source version and FOCUS target version. **Never change the version of existing data implicitly.** A monthly EffectiveCost total may differ from that month's invoice total.

MSP sale amount, unused-burden bearer, policy owner, allocation share, discount pass-through rate, calculation basis and evidence state live in a separate extension model. They never overwrite provider FOCUS amounts. **Unsupported values remain null with a reason; they are never filled with zero or an arbitrary commitment ID.**

## SP-03: FX versions and margin decomposition (P1-07, P2-02)

Source: B0460.R004, B0469, B0551, B0667.R009.

1. Cost reconciliation and allocation finish in provider currency. Conversion to billing currency happens at the contract-defined stage.
2. **The prepay purchase-date FX rate and the monthly billing FX rate are separate FX snapshot versions.** Neither substitutes for the other.
3. When the cost conversion basis differs from the selling conversion basis, the margin report shows the FX difference as a separate component. Margin-difference decomposition is a required operating report; the proposed components (cost FX, selling FX, markup, fees) are an HB-22 proposal, not source text.
4. **Cancellations and credits use the original invoice FX rate as the default proposal.** A contract or issuance-policy override records its reason and revision.
5. **A realized FX difference at payment is recorded separately and never modifies the original invoice.** It links to [payment and recognition](payment-and-recognition.md) records.
6. At close, the FX value, source response and policy are fixed as input evidence. An FX change after approval requires recalculation and reapproval. Issued documents never change to a later rate.
7. Tax follows the issuing entity's policy. Example rates are never used as statutory defaults.

## SP-04: on-premises cost pools and metering (P1-04, P1-05, P1-14)

Source: B0060, B0309, B0310, B0312, B0314, B0319.R001–R004, B0324, B0325, B0327, B0686.R006.

Cost pool inputs:

- The customer registers depreciation or lease, license, maintenance, power, floor space and operating cost into cost pools.
- Asset acquisition cost, residual value, useful life and effective period come from customer input or an approved asset system, never platform defaults.
- Block overlap between equipment purchase price and depreciation, and between metered power and facility charges that already include power.
- Pools may split into CPU, memory, storage, GPU and facility. **A shared server's cost is not applied in full to both CPU and memory**; it uses a customer-approved split ratio with revision and approver.
- The difference between accounting cost and selling price is explained as either margin or internal subsidy. Recording the classification per pool revision is an HB-22 proposal.
- The customer selects the allocation denominator: allocatable capacity, actual usage or reservation. Under capacity basis, unallocated cost may stay with the operations department; under usage basis, using organizations may bear the whole pool. The idle bearer and unattributed remainder are explicit, and conservation of the total is verified.
- **Internal allocation and an external sale of the related service are separate amounts and are never summed.**
- Metering APIs are not a substitute for internal cost or the customer selling contract. Cost is never fixed from utilization alone; external license purchases link to the customer cost policy.

Metering rules:

- Power separates cumulative meter differences from interval measurement. A value corrected for reset, replacement or clock error keeps its evidence and approval history. PUE, cooling and power-loss factors fix source, period and allocation basis. Facility cost already included is not added again.
- A VM identity combines the connected source with a stable identifier. It distinguishes rename, recreation and move, and records effective intervals for create, delete, resize and organization move. **The same cost is never charged on both the host and its VMs.** Metering interval, aggregation function, timezone and gap intervals are retained.
- The connector checks at connection time whether vCenter counters, collection level and history intervals provide the billing history needed. **The read connector never changes vCenter settings.**
- If source history is sufficient, query the period through the API. Otherwise collect change history and daily billing aggregates in the customer environment and fix only close inputs. **A period with neither source history nor permitted retention does not support exact usage billing**; an approved fixed-fee contract may apply separately.
- Each charging basis declares its customer policy: allocation basis (power-off, reservation hold, mid-month change); storage basis (allocated or used, thin provisioning, snapshot and backup inclusion); measured basis (unit, aggregation, gap treatment); reserved or fixed basis (included amount, overage, proration, minimum).

## SP-05: provisional estimated billing (P1-03)

Source: B0065.

Only usage that passed required-meter completeness, customer mapping and unit validation for the period is final. Missing usage is never zero. When the contract permits estimated billing, the estimate is issued as a **separate provisional statement** labelled as such, with its estimation basis and evidence. When final usage is confirmed, the difference is settled through a linked adjustment or correction document. HB-22 derived rules, not source text: a provisional statement never becomes final by elapsed time, and contracts without an estimate permission hold the period instead.

## SP-06: combined invoices and document kinds (P1-08, P1-09)

Source: B0034, B0129, B0130, B1219.R003, B1224.

- Cloud, data-center and AI selling lines may share one invoice **only when selling entity, customer, billing currency, period and tax conditions all match**, and, for group contracts, buying entity and ERP route also match (B1219.R003). A mismatch splits the invoice. Different issuing entities or payment responsibilities never merge into one legal invoice. A consolidated view is a separate non-billing report.
- A group relationship alone never merges invoices of different buying entities. One billing unit is formed only when buying and payment responsibility are contractually unified and the ERP and tax policy support it. Intercompany transactions differ from internal department charges; revenue recognition and consolidation elimination remain in the ERP.
- Document kind and title distinguish: provider-original invoice, **derived per-customer attribution statement** from a multi-customer supplier invoice, on-premises internal cost statement, and MSP sales invoice. A derived statement is never labelled as the provider's original, and the full multi-customer original is never disclosed to a customer automatically. Cost and margin information shown to end customers follows the contract and permissions.

## SP-07: contract pools and allowances (P3-02)

Source: B1218, B1219.R001–R005, B1222.

A standalone enterprise may use internal settlement only and activates contract functions when external selling is needed. Lookup groups, rate-sharing contract pools, cost-allocation pools and legal invoice bundles are configured separately. A contract pool revision fixes:

- Pool ID, participating entities, target SKUs, unit, currency and period, with simulation before and after join or leave.
- Shared or per-entity allowance and minimum, consumption order, remaining balance and carryover. **Moving an entity between groups never resets its allowance.**
- Payer, paying agent and document recipient. A paying agent is never interpreted as a change of buying entity.
- Start, renewal, end, price change and proration basis. New charges after the end date are separated from late corrections.

Overage attribution uses the approved policy, such as consumption order or proportional share. Proportional attribution uses the complete-pool residual algorithm of the [Claim and numeric contract](claim-and-numeric.md). Separating an entity in a lookup view keeps the contract pool until the pool itself changes. Mid-term exit applies contract rules for allowance reclaim, remainder and minimum. A group summary report may summarize several documents but is never issued as a new payment demand.

## SP-08: service lifecycle and proration (P3-01)

Source: B1469.R001–R006, B1472.

| Event | Required policy | Required check |
| --- | --- | --- |
| Start and first bill | Service start date, billing anchor date, first partial period | Distinguish no usage from not received |
| Quantity or price change | Immediate or next-period effect, period split, tier re-evaluation | Impact on existing approval and overlapping periods |
| Renewal and price transition | End and renewal dates, discount end, new contract basis | Explain price before and after renewal |
| Suspension and resumption | Whether fixed and reserved fees continue, usage treatment | **A collection outage is never treated as free usage** |
| Termination and final settlement | Charge end, late-source grace, balance handling | Accept late source data and pre-termination usage |
| Billing cycle change | Old and new periods, allowance, minimum, prepay handover | **Check for overlapping and unbilled intervals** |

Proration declares the contract method (actual calendar days, seconds, or fixed 30 days), timezone, start-inclusive and end-exclusive bounds, whether amount or quantity is prorated, and whether allowance, minimum and discount use the same ratio or separate policies. **Without a declared policy, proration is not finalized by any arbitrary method.** These fields are required policy structure; the chosen values are contract decisions.

## SP-09: AI profile view (P1-10)

Source: B0078.

The AI product view shows per-model usage, estimated and final cost, **remaining included quantity**, customer sale amount and invoice. Prompt and response bodies are excluded from billing storage by default, and user identifiers are pseudonymized when needed.

## SP-10: Marketplace and private offer policy (P2-09)

Source: B0686.R001–R006, B0697, B0699, B0700.

- Only meanings that have a FOCUS standard column map to that column; other Marketplace attributes live in a separate namespace.
- Every Marketplace or private-offer record requires a **transaction type**. For Microsoft: customer-direct, multiparty, resale-enabled or CSP private offer, each with its own billing and settlement path. The Microsoft CSP program is distinct from the generic cloud service provider term. A CSP private offer does not count toward customer MACC consumption.
- AWS private-offer pricing differs from public price, and the selling fee is never fixed as a single rate.
- Google Cloud product commitments and infrastructure CUDs are stored separately; VM and Kubernetes infrastructure cost is outside that private-offer discount. Usage period and billing period are separate.
- OCI eligibility and limits, and Alibaba Cloud coupon scope and actual deduction, are retained as contract-confirmed values. Alibaba official-site product coupons do not apply to Marketplace, and the global and China sites are not treated as identical. No purchase is assumed to consume a commitment unconditionally.
- Buyer credentials are never assumed to read seller settlement statements. Unauthorized fees, margins and private-offer terms are excluded from the customer portal and LLM input.
- Flow: register transaction type → identify original bill and contract → confirm whether the source already applied a discount → select MSP additional-policy targets → full simulation → reconciliation and approval → invoice for the matching payment responsibility.
- The contract template states **discount inheritance and Marketplace exclusion explicitly**. A blanket all-customer discount rule is never applied to Marketplace lines.
- Expiry, renewal, overage, instalments and refunds become work items. Possible return to public price or supply stop after renewal is shown as supplier contract state; **the platform never purchases automatically**. A late seller settlement never changes a customer invoice, and an unpaid settlement is never shown as collected.

## Synthetic acceptance cases

Numbers restate source examples for independent expectations. They are not chosen rates, tax or rounding policies. All cases are NOT_RUN.

| Case | Finding | Input and expected result |
| --- | --- | --- |
| H22-P01 | P2-01 | Savings-share contract with no evidenced comparison basis: the savings-linked line is held, with no list-price reconstruction |
| H22-P02 | P2-10 | B0566 commitment 100 (A usage 60, B 25, unused 15). Unused billed to A, fee 10% on usage only: A = 60 + 15 + 6 = 81, B = 27.50, total 108.50. Adding a fee on the unused burden requires the separate setting |
| H22-P03 | P2-10 | B0572 and B0574: unused 15 borne by the MSP, no fee, savings share 50%, at 1 USD = 1,350 KRW: A 135,000 − 27,000 = 108,000 KRW; B 67,500 − 16,875 = 50,625 KRW; total 158,625 KRW before tax |
| H22-P04 | P2-01 | B0576 prepay 1,200 already recovered and monthly effective cost 100: no monthly re-billing of the 1,200; no amortization recalculation |
| H22-P05 | P2-01 | Negative-savings loss: no automatic rate increase; warning and approval required |
| H22-P06 | P2-03 | AWS SP row with CoveredUsage and Negation: cost counted once. Azure Actual and Amortized: one view selected, never summed. GCP credit applied once |
| H22-P07 | P2-03 | Missing CommitmentDiscountId: null with reason, not zero or an invented ID; existing data's FOCUS version unchanged |
| H22-P08 | P2-02 | Prepay purchase-date FX and monthly billing FX stored as two versions; differing cost and selling bases show a separate FX margin component |
| H22-P09 | P1-07 | Payment received at a different FX rate: realized difference recorded separately; original invoice unchanged |
| H22-P10 | P1-05 | Shared server split ratios across CPU and memory pools sum to exactly 100% of the server cost, with idle or unattributed shares as explicit lines; a configuration totalling 200% or with an unexplained shortfall is rejected |
| H22-P11 | P1-14 | B0314 pool 1,000,000 KRW at 60/30/10: 600,000 / 300,000 / idle 100,000. External sale 750,000 is not added to the internal 600,000 |
| H22-P12 | P1-04 | Host and VM both metered for the same cost: charged once. VM recreated with the same name: new identity unless the stable identifier matches |
| H22-P13 | P1-04 | Period with no source history and no permitted retention: exact usage billing unsupported; fixed-fee contract applies only if approved |
| H22-P14 | P1-03 | Permitted estimate: provisional statement issued and labelled; final usage creates a linked difference adjustment |
| H22-P15 | P1-08 | Same customer, different tax condition: two invoices plus an optional consolidated report |
| H22-P16 | P1-09 | Derived per-customer split of a supplier invoice: document kind is derived statement, never provider original |
| H22-P17 | P3-02 | B1222 shared 1,000 units, A 700 and B 600: group overage 300. Under a consumption-order policy where A consumes first, A overage 0 and B overage 300. Under a proportional policy, overage splits 7:6 (about 161.54 and 138.46) with rounding and residual per the declared numeric policy. Separating B in a view keeps the pool |
| H22-P18 | P3-01 | Suspension during a collection outage: billed per suspension policy, never as zero usage. Cycle change produces no overlapping or unbilled interval |
| H22-P19 | P3-01 | Contract without a proration policy: period remains unfinalized with a blocking reason |
| H22-P20 | P1-10 | AI view displays remaining included quantity alongside estimated and final cost |
| H22-P21 | P2-09 | Marketplace record without transaction type: rejected. Blanket customer discount rule: not applied to Marketplace lines unless the template enables inheritance. CSP private offer: excluded from MACC consumption |
