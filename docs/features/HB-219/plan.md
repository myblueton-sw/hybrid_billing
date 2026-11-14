---
type: ImplementationPlan
title: Data-centre components with inclusions, usage add-ons that never charge included quantities, and SLA credits that need a contract clause and operator approval
description: Defines the AR-10 increment of work item W-22, a register of contract components for fixed racks, space and lines, reserved GPU base fees, usage add-ons and operations services, with fixed components declaring what they include, a period statement that charges usage add-ons only beyond the included quantity, incidents kept as evidence only, and SLA credits proposed under a contract clause and approved by another actor before they become a credit line linked to its credit document and balance increase.
status: draft
---

# Scope and ownership

[HB-219](https://linear.app/hybrid-billing/issue/HB-219) is the follow-up of HB-218 for the rest of work item W-22. This increment covers AR-10 and AUD-AR-10 at package level. DB-10 was partly delivered by HB-212 (`modules/marketplace`), and its remainder, refunds, external authority locks, included pools, PT-05 and PT-08 stay in a follow-up. Base: main `dcff4ca`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the statement computes customer charges and credits. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-14:

- **AR-10.** Fixed rack, space and line charges, reserved GPU base fees, usage add-ons, operations services and SLA credits are independent components.
  - Power or services included in a fixed fee are never charged again as usage.
  - An incident is only review evidence: no SLA discount line is created without a contract clause and operator approval.
- **AUD-AR-10 (NOT_RUN).**
  - A separate power charge on top of a fixed fee that includes power is blocked.
  - An SLA discount created from an incident alone is blocked.
  - An approved credit is linked to its source and to the balance.
- **Existing code.**
  - `modules/rating` (HB-204) prices provider lines with included quantities, but it has no data-centre components.
  - `modules/balances` (HB-216) accepts `sla_credit` increases with a reference.
  - `modules/hawkeye` (HB-194) receives data-centre cost statements.
  - Nothing models data-centre products or SLA credits.

# Decision

One Go package, `modules/components`, with the standard library and the primitives `decimal` and `exact`. A registry with an injected clock and one mutex holds the data per workspace. Nothing calls it and no entry point is wired. It imports no other module; the link to a balance increase is a recorded reference.

Strings follow the same rules as HB-218:

- identifiers;
- three-letter currency codes;
- dates from 2000-01-01 to 2099-12-31;
- decimal texts that the registry's profile reads;
- 64-character lowercase hexadecimal digests;
- closed lists.

There is no free text and no floating point. Errors are `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` and `ErrLimit`; a named refusal carries its reason. Checks run in the order form, lookup, repeat, rule, room. An identical repeat by the same actor returns the stored record, and the same content from another actor is `ErrConflict`.

## Components

`Define(ws, Component, actor)` records one component, written once:

- `ID`, `Contract`, `Currency`.
- `Kind`: `rack`, `space`, `line`, `gpu_reserved`, `usage`, `service`.
- `From`, `To`: inclusive effective dates; `To` is optional.
- Fixed kinds (`rack`, `space`, `line`, `gpu_reserved`, `service`) have:
  - `Fee`: a fixed amount per statement period, zero or more;
  - `Includes`: zero or more inclusions `{Meter, Quantity}`, where the quantity is positive or the word `all`;
  - no `Meter` and no `Rate`.
- A `usage` component has:
  - `Meter`;
  - `Rate`: per unit, zero or more;
  - no fee and no inclusions.

`ApproveComponent(ws, id, actor)` needs another actor. Only approved components take part in a statement.

Two approved usage components of the same contract and meter must not have overlapping dates. This is refused at approval (`ErrRejected`, `meter_overlap`), so a meter is never charged twice. At most one approved fixed component of a contract may include a meter at any time (`inclusion_overlap`, at approval).

## Statements

`Statement(ws, Request{Contract, PeriodStart, PeriodEnd, Usage []Reading{Meter, Quantity}})` computes and returns a result. It records nothing. The request holds:

- the period, with start ≤ end;
- amounts are rounded per line with the contract's pricing (`SetPricing`);
- one reading per meter: the measured quantity for the period, zero or more.

A component takes part when it is approved and its dates cover the whole period. A component that covers only part of the period refuses the statement (`ErrRejected`, `partial_period`); proration is a later contract. When several currencies take part, the statement is refused (`currency_mismatch`).

The result has one line per participating component, plus a line per meter reading that no component covers:

- **Fixed component:** one line with the fee as its amount. Its inclusions are listed.
- **Usage component on a meter:**
  - included = the sum of the inclusions of that meter over the participating fixed components, or `all`;
  - billable = max(0, measured − included), or 0 when the meter is fully included;
  - the line amount is billable × rate, rounded with the contract's pricing;
  - the line shows measured, included, billable and the rate.
- **A reading whose meter no usage component covers:**
  - if the meter is included in a fixed component, it is shown as an included line with amount 0;
  - otherwise it is shown as an unpriced line with no amount (`unpriced`) and counts in no total.
- **A usage component with no reading:** refused (`missing_reading`).
- **Approved SLA credits** of the contract whose credit period lies in the statement period: an informational credit line with its amount, state and link, excluded from the net; the credit is settled only through its credit document and balance increase.
- **Totals:** charges (the sum of fixed and usage lines) and net = charges, exact, in the component currency.

## Incidents and SLA credits

- `RecordIncident(ws, Incident{ID, Contract, Component, Start, End, Evidence}, actor)` records evidence only: a start and end time with end > start, and an evidence digest. It never creates a line or a credit.
- `AddClause(ws, Clause{ID, Contract, Component, Rate, From, To}, actor)` records an SLA clause of the contract. It holds a credit rate greater than 0 and at most 1 of the component's fixed fee and needs approval by another actor (`ApproveClause`).
- `ProposeCredit(ws, Credit{ID, Clause, Incidents []string, PeriodStart, PeriodEnd}, actor)` proposes a credit, which starts pending. The incidents are evidence (1 to `Limits.Incidents`) and must belong to the clause's contract and component. Its amount is computed as the component fee × the clause rate, rounded with the contract's pricing. It needs:
  - an approved clause whose dates cover the period, and a fixed component;
  - no other non-withdrawn credit of the same clause whose period overlaps (`credit_overlap`), so one breach is credited once.
- `ApproveCredit(ws, id, actor)`: another actor than the proposer, who is the operator approval. It re-checks the clause and the overlap rules.
- `WithdrawCredit` works while the credit is pending.
- `LinkCredit(ws, id, Link{Document, Increase}, actor)` records, for an approved credit, the credit document identifier and the `modules/balances` increase identifier. Each link is written once. A credit with no link is shown as `unlinked`.

Without an approved clause, a credit can never be proposed. Without approval, no credit line appears. An incident alone changes nothing (AUD-AR-10).

## Limits

`Limits{Workspaces, Components, Inclusions, Readings, Incidents, Clauses, Credits}`. Each limit is positive and has a cap. `Components`, `Clauses` and `Credits` count per contract; `Inclusions` per component; `Readings` per request; `Incidents` per credit and per contract.

# Provisional choices awaiting the owner

- Statements need whole-period coverage, and there is no proration.
- One SLA credit formula: a clause rate × the fixed fee. Tiered SLA levels and availability computation are not built.
- At most one fixed component includes a meter at a time, which refuses stacked bundles that each include the same meter.
- An uncovered, non-included reading is shown as unpriced and never charged.
- SLA credits are settled only through the credit document and the balance, never as a statement deduction.

# Verification

- Literal tests for every rule and refusal, including:
  - a fixed fee including all power, with power usage that charges 0;
  - partial inclusion;
  - two fixed components including the same meter;
  - an incident with no clause that produces no line;
  - a credit approved by another actor, linked, and shown on the statement;
  - overlap refusals.
- A property test against a `math/big.Rat` model of the statement: random components, inclusions, readings and credits.
- Race detector, count=10, a 386 build, gofmt, vet, the whole repository test run, a mutation run and a source scan.
- Not covered: proration, availability computation, documents, persistence, screens, and system acceptance.

# Design review resolution

The first design review returned FAIL with findings 1 to 12 and two hypotheses. Where this section and the text above disagree, this section governs.

1. **One settlement channel.** An approved SLA credit is settled only through its credit document and a `modules/balances` `sla_credit` increase.
   - On a statement it appears as an informational credit line: its amount, its state (`linked` or `unlinked`) and its link. It is excluded from `Net`, so `Net` = `Charges`, and a credit lowers what is owed once, through the balance.
   - `credit_exceeds_charges` is removed.
   - A test shows that an approved, linked credit leaves the net unchanged and appears once.
2. **Cadence.** Every component is billed per calendar month, which is the only cadence in this increment.
   - A statement period must be exactly one calendar month, from its first to its last day. Otherwise the statement is `ErrInvalid` (`period_mismatch`).
   - A component takes part when it is approved and its dates cover the whole month. Partial coverage is still `partial_period`.
   - The statement is a preview and records nothing. Issuing a month once is the duty of the issuance and claims layer (`modules/claims`, HB-214), not of this package.
3. **Once per breach.**
   - Non-withdrawn credits of the same component must not have overlapping periods (`credit_overlap`), whatever their clause.
   - An incident may back at most one non-withdrawn credit (`incident_used`).
   - Approved clauses of the same component must not overlap in dates (`clause_overlap`, at `ApproveClause`).
   - All of these are re-checked at `ApproveCredit`.
4. **Incident timing.** The UTC date range of each incident must intersect the credit period and lie within the clause dates. Otherwise the credit is refused (`incident_outside_period`).
5. **The credited component must take part.** At proposal and at approval, the component must be approved, fixed, and cover the whole credit period.
6. **Credit period.** A credit period is exactly one calendar month (`period_mismatch` otherwise). It therefore lies inside one statement, and that statement includes the component.
7. **Credit amounts.**
   - A clause rate must be greater than 0 and at most 1.
   - A credit whose computed amount is 0 is refused (`zero_credit`).
8. **Links.**
   - Within a workspace, a credit document identifier and an increase identifier may each be linked to only one credit (`link_used`).
   - A link is a recorded reference only: nothing checks that the balance increase exists or has the same amount and currency. AUD-AR-10's "linked to the balance" therefore stays UNVERIFIED until an integration checks it.
   - Approved credits without a link are listed as `unlinked`.
9. **Excess beyond a partial inclusion.** When a reading's meter has no usage component, the included part is shown as included with amount 0. Any excess quantity is shown separately as `unpriced` with no amount and counts in no total. An excess is therefore never hidden.
10. **One inclusion per meter.** At any time, an approved fixed component of a contract is the only one that may include a given meter (`inclusion_overlap`, at `ApproveComponent`). A second fixed charge for the same power is therefore refused. The provisional rule "inclusions add up" is replaced by this one, and the `all` combination question no longer arises.
11. **One rounding rule per contract.** `SetPricing(ws, Pricing{Contract, Scale, Mode}, actor)` is written once per contract, with a scale from 0 to 8 and a primitive rounding mode.
    - `Define` refuses a component of a contract with no pricing (`no_pricing`), and a fee with more decimals than the scale (`ErrInvalid`).
    - Usage lines and credit amounts are rounded per line to that scale and mode.
    - Requests and credits carry no scale and no mode.
    - The property model applies the same per-line rounding.
12. **Gaps.**
    - Approved credits have no reversal path in this increment; a reversal is a balance matter later.
    - Duplicate meters in a request are `ErrInvalid`.
    - `meter_overlap` and `inclusion_overlap` are judged at `ApproveComponent` under the lock, whatever the currency. A test approves two pending components one after the other.
    - `Limits` has `IncidentsPerCredit` and `Incidents` (per contract) as separate fields.
    - Clause dates must lie within the component's dates (`clause_outside_component`).
    - Meter units are implied by the meter identifier (H2), and the plan states this as a caller obligation.

## Re-review resolution

The re-review returned PASS with three low findings. They are resolved as follows:

1. **Inclusion claim.** Item 10 covers declared inclusions only. A fixed component that charges for power without declaring the meter in `Includes` is not detected; this residual is recorded against AUD-AR-10. The one-inclusion-per-meter rule is a provisional choice, and it refuses stacked bundles.
2. **Superseded text.** The superseded text in the body has been replaced by the resolved rules.
3. **Error details.**
   - A credit period that is not one calendar month is `ErrInvalid` (`period_mismatch`), as for a statement.
   - `incident_outside_period`, `zero_credit` and `clause_outside_component` depend only on records that are written once, so they are judged when the credit or clause is proposed or added. `credit_overlap`, `incident_used` and the clause and component state are re-checked at `ApproveCredit`, and `clause_overlap` at `ApproveClause`.
   - Check order: `no_pricing` (at `Define`), `clause_overlap` and `link_used` are rule refusals. They come after form, lookup and the identical-repeat check, and before room. Repeating the same link by the same actor returns the stored credit.

# Change review resolution

The independent change review returned FAIL on three low findings. It confirmed the charging and credit rules against the plan.

1. **Credit room per contract.** The `Credits` room now counts per contract, not per clause, as the plan states. A test fills the room with two clauses on two components.
2. **Credits in the property test.** The property test now adds credits:
   - random clauses, incidents and proposals, which are then withdrawn, approved, or approved and linked;
   - a check of each amount against an independent rounding model;
   - a check that exactly the approved credits appear, and that `Net` = `Charges`.

   It varies the scale from 0 to 4 and covers all six rounding modes.
3. **An uncovered zero reading** is now shown as `unpriced`. A reading is shown as included only when a fixed component actually includes the meter.

Hypotheses:

- H1, accepted: an incident's end is exclusive, so its last date is computed from end − 1 ns. An incident ending at 00:00Z on the first of a month does not back a credit for that month. A test covers both sides.
- H2, kept: an approval repeated by a different actor is `ErrRejected`, as in HB-218.
- H3, kept: more readings or incidents than allowed is `ErrInvalid`, because these are request form bounds.
- H4: the re-checks at `ApproveCredit` are kept as guards, even where mutation cannot reach them while clauses and components cannot be revoked.

These fixes were not reviewed again.
