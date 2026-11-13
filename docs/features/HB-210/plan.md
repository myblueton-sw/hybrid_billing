---
type: ImplementationPlan
title: Provider commitment facts, cost preservation, allocation of used and unused cost and sale policies
description: Defines the first model of work item W-19, provider commitment facts kept as given and sealed per period after checks of relations and cost preservation, an allocation of a sealed period's cost into customer used cost, unused cost borne under a stated policy and an explicit unattributed part, and customer sales under separate sale policies, reproducing the source's worked example.
status: draft
---

# Scope and ownership

[HB-210](https://linear.app/hybrid-billing/issue/HB-210/w-19-provider-commitment-facts-cost-preservation-allocation-of-used) is the first increment of work item W-19 (requirement HB-19 with the required contract DB-09). It is the twenty-fourth increment built under the owner's instruction of 2026-11-12. Base: main `93a9aaf`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the model allocates cost and computes customer charges. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-13:

- Requirement HB-19: accept the results a provider gives for RI, SP and CUD commitments (benefit, used and unused, offset relations); purchase recommendation, optimisation and recomputing the provider's amortisation are excluded. Proposed direction: keep provider facts as CommitmentVersion, SourceCharge, PeriodCost and BenefitAttribution and keep AllocationPolicy and SalePolicy apart; separate cost-linked and independent prices. Scale: an installation, a payer account, a shared pool and a period are the unit of computation; customer results are final only after the pool's snapshot is complete; hourly facts are not synthesised from monthly totals.
- Acceptance of HB-19: the source example's costs 60/25/15, a sale of 93.50 with a loss of −6.50 and the alternative 117.50; cost preservation, re-collection of a prepayment, double discounting and late corrections are tested. Open decisions, not made: real commitment samples, who bears unused cost, the rate of benefit passed on, the policy for negative savings.
- The source document, chapter 45, read from `통합_빌링_플랫폼_구현_기획안_v1.16.docx`: a single USD commitment's month cost of 100 is settled as used cost A 60, B 25 and unused 15; the same usage without the commitment would be A 100, B 50. Effective cost plus a 10% fee on used cost gives A 66, B 27.50, sales 93.50, and after the commitment cost 93.50 − 100 = −6.50; a loss does not raise customer rates automatically. Alternative 1: A bears the unused 15, giving A 81, B 27.50, total 108.50; whether a fee applies to unused cost is a separate setting. Alternative 2: the MSP bears unused cost and 50% of used savings is passed on, giving A 100 − 40 × 0.5 = 80, B 50 − 25 × 0.5 = 37.50, total 117.50 and 17.50 after the commitment cost, without the fee. A prepayment of 1,200 and a month's effective cost of 100 are different bases; the platform does not recompute amortisation, and a purchase price already recovered from a customer is not charged again monthly.
- DB-09: keep a test matrix per commitment profile of full, partial and no upfront payment, utilisation 0 and 100%, excess, concurrent commitments, start, end and move within a month, scope change, late credit, refund or exchange, correction after cancellation, negative savings, FX change and partial CSV errors; an event a source does not support is recorded with its reason and not passed as normal. AUD-DB-09 and COM01–COM09 are NOT_RUN.
- Existing code: `modules/allocation.LargestRemainderV1` splits an amount at a quantum by weights; `packages/primitives/exact` (HB-208) has exact fractions and rounding; `modules/fx` converts amounts from snapshots.

# Decision

One Go package, `modules/commitment`, standard library, the decimal primitive, `packages/primitives/exact` and `modules/allocation`. A registry with a clock keeps commitments and period facts and judges period revisions; two pure functions allocate a period and compute sales. Nothing calls it and no entry point is wired. This section is the design after the first design review, whose findings are resolved at the end of this document.

What this can and cannot show: that a period's facts are kept as given at their own scale, never changed, and corrected only by a new revision; that only the highest revision of a period is judged, and a period is usable only when that revision is sealed; that a sealed revision's lines refer to its commitment, no source line is current in two periods, and its lines sum exactly to the period cost the provider gave; that the amortized parts of a commitment's current periods never exceed its prepaid amount; that an allocation sums exactly to the period cost as customer used cost, unused cost borne under a stated policy, a rounding part and an explicit unattributed part; that a sale under stated policies is computed with one rounding per charge, never invents a charge, never charges an amortized part of a commitment whose purchase is stated as recovered, and reports a result below cost only when every customer is priced, without changing any rate; that every result names the digests of the facts and policies it came from. It cannot know that the provider's facts are complete or true (COM01), that an account belongs to a customer, that a policy is what a contract says, or that a result was invoiced. It does not recompute amortisation, recommend or optimise commitments, or convert currencies.

Names. A period revision is `sealed` when this package's checks passed, which does not say the provider's facts are complete. A sale result is `all_customers_priced` when no customer is held, and `below_cost` when its charges are below the cost it compares with.

Not delivered, although HB-19, DB-09 or COM01–COM09 name it: source completeness against a manifest (COM01); reconciliation with the provider invoice (COM02); usage that should refer to a commitment and does not, and a commitment benefit combined with another discount on one line (COM03, beyond lines current twice); the provider's statement of a prepaid balance and a check that amortization reaches the prepaid amount by the last month, and balances recovered per customer (COM05); stepwise recomputation across contract bases, overlays and discounts (COM07); rounding differences between lines and totals and FX versions (COM08, currencies being HB-208's); approved inputs, no duplicate issuance and per-customer exposure (COM09); commitment versions for scope changes, moves and exchanges; hourly facts; excess usage, concurrent commitments sharing a line, credits and refunds as their own facts; the pool and payer-account snapshot as a unit across commitments; the readiness of cost-linked against independent prices; invoices, screens, commands and persistence. The design part of HB-19 stays open with the owner. W-19 asks for independent verification by a DBA and by QA; this ticket has one independent reviewer of design and change.

Every string is an identifier under the project's rule (letters, digits, `. _ : -`, at most 128 bytes), a currency code, a month `YYYY-MM` from 2000-01 to 2099-12, the text of a number of at most 64 bytes that the registry's decimal profile reads, a digest of 64 lowercase hexadecimal characters, or a value of a closed list. There is no free text; actor identifiers must be pseudonymous (stated obligation). No floating point is used. Number texts are stored as the decimal primitive's canonical print.

## Commitments

A commitment has an identifier in a workspace, a provider and a payer account (identifiers), a kind `reserved_instance`, `savings_plan` or `committed_use_discount`, a currency, a first and a last month, a payment option `all_upfront`, `partial_upfront` or `no_upfront`, a prepaid amount (zero for `no_upfront`, above zero otherwise), and a quantum (`1`, `0.1`, `0.01`, `0.001`, `0.0001`) at which allocated parts and charges are stated. It is written once; an identical repeat writes nothing, anything else is a conflict. A commitment version for scope changes is not delivered.

## Periods

A period is a commitment and a month within its months. Its provider facts are recorded under revisions numbered from 1 without a gap, each written once, at the scale the provider states:

- the period cost, the effective cost of the commitment for the month, as a recurring and an amortized part, each zero or above, the amortized part zero for `no_upfront`;
- lines, each with a source identifier unique in the revision, a kind `used` or `unused`, the commitment it refers to, for `used` lines the provider's account that used it, a recurring and an amortized effective cost, each zero or above, and for `used` lines a list cost, the cost of the same usage without the commitment, zero or above.

A revision is `written`, then judged by `Seal(workspace, commitment, month, revision)`, which accepts only the highest written revision of the period; when a higher revision is written, every lower revision that was not judged becomes `superseded`. Sealing an earlier revision is rejected. The outcome, `sealed` or `held` with every reason that applies, is stored; a repeat returns it. Reasons, in this order:

- `foreign_commitment`: a line refers to another commitment;
- `duplicate_source`: a line's provider and source identifier is in the current revision of another period of any commitment of the workspace with that provider (a line linked twice); a line moved from one period to another is therefore sealed in its new period only after the old period's correction is sealed;
- `cost_not_preserved`: the recurring or the amortized parts of the lines do not sum exactly to those of the period cost;
- `prepaid_exceeded`: the amortized part of this revision and those of the current revisions of the commitment's other periods exceed its prepaid amount.

All seals of a workspace run under one lock, so two periods of one commitment are never judged at once.

`Current(workspace, commitment, month)` returns the period's state: `sealed` with the highest revision and its digest when that revision is sealed; `correction_held` when the highest revision is held, whatever lower revision was sealed; `not_judged` when it is written and not judged; no revision, not found. Only a sealed current revision may be read as an input with `Revision(workspace, commitment, month, revision)`, which returns the commitment, the facts and the digest. The pure functions cannot know whether an input is still current; a consumer compares the digest a result names with `Current`.

The revision digest is the SHA-256 of a print in which every field is its length in eight bytes, big-endian, then its bytes, numbers as canonical decimal text, absent parts as empty fields and repeated parts preceded by their count: the label `commitment_period_v1`, the workspace, every field of the commitment, the month, the revision, the period cost's two parts, and the lines in byte order of source identifier with every field.

Negative savings (a list cost below the effective cost of a used line) are kept as given.

## Allocation

`Allocate(period, attribution, policy)` is pure. `period` is a revision as `Revision` returns it; one that is not sealed is invalid. `attribution` maps the provider's accounts to customers, with an identifier and a version; a used line whose account it does not map holds the allocation `account_not_attributed`. `policy` has an identifier, a version, the way unused cost is borne, and the recovery of the purchase:

- unused cost borne by `msp`; by `customer`, a named customer, who may have no used line and then appears with only unused cost; by `pro_rata_used`, customers in proportion to their total used effective cost, for both parts; or `unattributed`, recorded as borne by nobody;
- `amortized`: `charged`, the amortized parts are cost to be charged to customers monthly, or `recovered`, the purchase price was recovered from customers under their contracts and the amortized parts are left out of every charge and of the cost a sale compares with (provisional; the fee on amortized parts is then dropped too).

Used cost per customer is exact, at the provider's scale. Unused cost borne by customers under `pro_rata_used` is split by `LargestRemainderV1` at the commitment's quantum, recurring and amortized separately with the same weights; ties go to the customer first in byte order, so that customer may receive up to two more quanta in a period (stated bias); what the split cannot express at the quantum is the rounding part, borne by the MSP. Unused cost of zero needs no split. Unused cost above zero with a total used effective cost of zero holds `no_basis` (with utilisation 0% this policy always holds; stated cost).

The result carries the label `commitment_allocation_v1`; the period's revision and digest; the attribution's and the policy's identifier, version and content digest; per customer in byte order the used recurring, used amortized, used list cost, savings, the count of used lines with negative savings, and the unused recurring and amortized borne; the MSP's borne parts and the rounding part; the unattributed parts; or a hold with its reason. Its parts sum exactly to the period cost, checked before it is returned (COM06). A content digest of the result is printed like the revision's.

## Sales

`Sell(allocation, policies)` is pure. The allocation must not be held. Each policy names a customer, has an identifier and a version, a rounding mode of the six names, and a method:

- `cost_plus`: a fee rate from 0 below 1 (provisional bound) and whether the fee applies to unused cost; the charge is the customer's used effective cost × (1 + fee) + its unused borne × (1 + fee if it applies, else 1);
- `savings_share`: a share from 0 to 1; the charge is list cost − savings × share + unused borne. A customer with any used line of negative savings is held `negative_savings` (provisional; the owner decides).

When the allocation's amortized parts are `recovered`, every term uses the recurring parts only. Each charge is rounded once to the quantum under the policy's mode. A customer of the allocation without a policy is held `no_sale_policy`; a policy for a customer the allocation does not have is invalid; two policies for one customer are invalid.

The result carries the label `commitment_sale_v1`, the allocation's content digest, per customer the policy's identifier, version and content digest and either the charge with its exact value or the hold; the total of charges; the cost compared with, which is the period cost, less its amortized part when recovered; the recovered amortized part as its own term; and, only when `all_customers_priced`, the MSP result (total − cost compared with) and `below_cost` when that is below zero. A result contains every customer's figures; a view for one customer must be filtered before it is shown to anyone (COM09).

## Error kinds, tenant boundary, time and limits

A request is judged: sizes, a limit error (lines before they are read); form by the request alone, invalid; lookup, not found; a repeat when identical, a conflict otherwise; a stored state or rule, rejected (a revision beyond the next, sealing a revision that is not the highest); room, a limit error. `Allocate` and `Sell` return invalid for an input not of the stated form, a limit error beyond digits or a split the allocation primitive refuses, and holds as results. Errors are fixed values. Every operation takes the workspace and reaches only its own commitments; the registry authorizes nothing and must not be exposed directly. Its time is the highest reading of its clock, in UTC, recorded on writes and seals. Limits, each required and bounded: commitments per workspace (100,000), revisions per period (1,000), lines per revision (100,000), customers per allocation (10,000), digits of a whole number (10,000), page size (1,000), all within a 32-bit `int`. Modes are the six names; for `LargestRemainderV1` the name is turned into the decimal primitive's mode.

## What is not decided here

Provisional defaults, each to be confirmed or changed by the owner: the four ways to bear unused cost and pro rata by used effective cost; the rounding part borne by the MSP; recovery of the purchase stated per allocation policy and dropping amortized parts and their fee; the two sale methods, the fee below 100% and the fee on unused cost as a setting; holding `savings_share` on negative savings; one rounding per charge. Not conservative, stated: facts are the provider's statements; nothing checks an account against a contract; `below_cost` only marks the result. Costs to availability: a held correction blocks the whole period; a moved line waits for its old period's correction; `pro_rata_used` holds at utilisation 0%; one customer without a policy keeps the MSP result back.

# Allowed files

`modules/commitment/*` (new); `docs/features/HB-210/*`.

# Verification

- The source example as literal tests: used A 60, B 25, unused 15, list A 100, B 50; `cost_plus` 10% with the MSP bearing unused cost gives 66, 27.50, total 93.50, MSP result −6.50 and `below_cost`; alternative 1, A bearing unused cost without a fee on it, gives 81, 27.50, total 108.50, MSP result 8.50, and with the fee on unused 82.50 and 110; alternative 2, `savings_share` 0.5, gives 80, 37.50, total 117.50 and 17.50. A prepayment of 1,200 with a month of 100 amortized and A using all of it: with `recovered`, A's charge is 0, the cost compared with is 0, the recovered term 100 and no `below_cost`; with `charged` and 10%, A's charge is 110.
- Literal tests: every seal reason alone and together; a held correction after a sealed revision and `Current`; sealing a lower revision; superseded revisions; a line moved between periods in both orders; preservation with a gap of the provider's smallest unit; prepayment exceeded across periods and with corrections; utilisation 0% and 100%; negative savings under both methods; every bearing policy, ties and the rounding part; unattributed accounts; a customer bearing unused cost without used lines; an MSP result withheld when a customer is held; the digests against bytes built in the test and against every changed field; every refusal of every operation with the order of error kinds; repeats and conflicts; the tenant boundary; limits; errors as fixed values; a scan for floating point and comments; a build for linux/386.
- Concurrency under the race detector with gates: two periods of one commitment sealed at once against its prepaid amount, at most one passing when both together would exceed it.
- Properties with fixed seeds: random sealed periods allocated under every policy, the parts summing to the period cost, and sales compared with a reference written with `math/big.Rat`, which shares the author's reading of this plan; every hold reason and refusal kind must be reached.
- The DB-09 matrix in the verification record, each event under all, partial and no upfront with one status: covered by a named test with synthetic data and its expected value; not built in this package, deferred to a named ticket; or not supported by a source, with evidence. No package gap is recorded as a source's lack of support.
- A mutation check by a script in Python with restores read back and the build cache emptied after every part; `gofmt -l`, `go vet ./...`, `go test -race -count=1 ./...`.

Remaining NOT_RUN at system level: COM01–COM09, AUD-DB-09. This ticket does not complete W-19.

# Deferred

Everything listed as not delivered above.

# Design review resolution

The independent design review of 2026-11-13 returned FAIL without need for redesign, findings F1 to F15. The sections above are the corrected design.

| Finding | Resolution |
| --- | --- |
| F1 a held correction left stale facts current | Only the highest revision is judged; `Current` returns `correction_held` or `not_judged`; lower unjudged revisions are superseded. |
| F2 a spurious loss with a recovered purchase | The cost compared with leaves out recovered amortized parts, shown as their own term. |
| F3 recovery per customer allowed double recovery | Recovery is stated once per allocation policy for every customer; COM05 balances listed as not delivered; the dropped fee stated. |
| F4 a quantum on provider facts | Facts at their own scale; the quantum applies to splits and charges; a rounding part. |
| F5 digests and policies not recorded | Revision, allocation and policy digests; full result fields. |
| F6 duplicate sources | Among current revisions, by provider and source identifier; moved lines; other gaps listed. |
| F7 seals of one commitment at once | One lock; a race test; COM05 limits stated. |
| F8 attribution stored with provider facts | Lines carry the provider's account; the attribution is a separate versioned input; no commitment versions, stated. |
| F9 an MSP result while customers are held | Withheld unless `all_customers_priced`. |
| F10 pro rata edge cases | Defined; bias and cost stated. |
| F11 negative savings | Counted per customer; `savings_share` holds. |
| F12 contract details | Reads, repeats, holds as results, limits, modes, exposure stated. |
| F13 not delivered | Listed. |
| F14 names | `below_cost`, `all_customers_priced`; `sealed` qualified. |
| F15 tests | Added, with the DB-09 statuses. |

# Design re-review resolution

The independent re-review of 2026-11-13 returned FAIL with two blockers (N1, N2) that need no redesign, and findings N1 to N10. The rules below are part of the design and prevail over the sections above where they differ.

- N1, what other periods contribute to the checks: for `duplicate_source` and `prepaid_exceeded`, every other period contributes both its last sealed revision and its highest written revision: the lines of either, and the larger of their amortized parts. So a line sold in a sealed revision stays linked until a correction without it is sealed, and an amortized part sold stays counted until a lower one is sealed. A line moved to another period is therefore sealed there only after the old period's correction is sealed.
- N2, savings: the savings of a used line are its list cost less its recurring and amortized effective cost. When the commitment's purchase is recovered, `savings_share` holds `recovered_with_savings_share` (provisional; the owner decides).
- N3, the split: unused cost under `pro_rata_used` is first reduced toward zero to the commitment's quantum, then split by `LargestRemainderV1` under `toward_zero`; the rounding part is the unused cost less its reduced value, at least zero, borne by the MSP. Used cost and unused cost under `customer`, `msp` and `unattributed` stay at the provider's scale; only charges and pro-rata parts are at the quantum.
- N4, recovery: recovery of the purchase, `charged` or `recovered`, is a field of the write-once commitment, not of an allocation policy. Recovery from only some customers is deferred.
- N5, inputs of the pure functions: `Revision` returns a period value whose fields only this package sets, and `Allocate` returns an allocation of the same kind; their contents are read through methods. `Allocate` refuses a period it did not receive from `Revision` and `Sell` refuses an allocation it did not receive from `Allocate` or that is held, as invalid. A period value is a copy; it does not say whether it is still current.
- N6, prints: the attribution, the allocation policy, each sale policy, the allocation and the sale each have a content digest printed like the revision's, with their own labels (`commitment_attribution_v1`, `commitment_allocation_policy_v1`, `commitment_sale_policy_v1`, `commitment_allocation_v1`, `commitment_sale_v1`) and their entries in byte order. Results record the decimal profile's identifier and revision.
- N7, order: `Seal` of a revision that is not the highest written one is rejected before anything else is judged, also when it was sealed before. A period cost or line with an amortized part above zero on a `no_upfront` commitment, a month outside the commitment's months, and `Revision` of a revision that is not the current sealed one are rejected. An allocation holds with `account_not_attributed` before `no_basis`. A sum that does not reproduce the period cost is a self-check error, not a limit error.
- N8, judging again: `Seal` of a held highest revision judges it again and replaces its outcome, since its reasons may depend on other periods. A period whose revisions reach the limit can no longer be corrected (cost, stated).
- N9: the MSP result counts unattributed cost as the MSP's cost.
- N10: the duplicate key is the commitment's provider and payer account and the line's source identifier. Any customer of a largest-remainder split may receive one quantum more for its remainder, byte order only breaking ties.
- Tests added to Verification: both sequences of N1; `savings_share` with a recovered purchase; recovery with unused cost borne by a customer, pro rata and with the fee on unused; a non-zero rounding part; `no_upfront` with an amortized part; a repeat seal of a lower revision; `Revision` refused for held, unjudged and superseded revisions; hand-built inputs to the pure functions; several holds in one allocation; a held allocation passed to `Sell`; a customer bearing only unused cost under `savings_share`; attribution entries for accounts the period does not have; a month outside the commitment; the bearing policy and sale method named in each source example (alternative 2 under `msp`, the prepayment case under `cost_plus`), alternative 1 with the fee on unused giving an MSP result of 10; DB-09 rows for start and end within a month, correction after cancellation and partial CSV errors.

# Second re-review resolution

The confirmation of the re-review resolution on 2026-11-13 returned PASS with four minor points and no blocker.

- The values `Period` and `Allocation` carry a field only this package sets; their zero value and any value built elsewhere are refused as invalid, and the tests give both.
- A commitment is write-once, so a wrong recovery setting makes it unusable for good (cost, stated). `recovered` on a `no_upfront` commitment is invalid.
- A highest written revision that is never judged keeps blocking other periods through its lines and its amortized part until a newer revision supersedes it (cost, stated).
- Tests added: a held revision judged again after the blocking period changed, sealing the second time; a commitment repeated with another recovery setting, a conflict; a rounding part in both the recurring and the amortized parts at once.

# Change review resolution

The independent change review of 2026-11-13 returned PASS with findings C1 to C13 and no blocker; it found no input that gives a wrong number where a hold or an error is required, and reproduced the source example by hand. The corrections below were made after it and were not reviewed again by a separate reviewer.

- C1, stated: one lock guards the whole registry, not one workspace, and a seal scans every period of the commitments with the same provider and payer account; a large seal delays every workspace, and nothing bounds the number of workspaces or the stored lines in total. An index and a lock per workspace are deferred.
- C2: writing a revision other than 1 for a period that has none is rejected, as any revision beyond the next.
- C3: the exact value of a charge is printed as a decimal.
- C4: a sale records the decimal profile's identifier and revision, also in its digest.
- C5: a held allocation carries no parts, only its hold and the identifiers and digests of its inputs.
- C6: under `customer` the named customer is added only when there is unused cost.
- C7, stated: fee rates and shares are texts of at most eight decimals read by the package itself, not by the profile; a text equal in value with more decimals is invalid.
- C8, stated: the number of attribution entries is limited by the customer limit (provisional).
- C9: the page limit was removed, being unused; the public reads `Revisions` and `Commitment`, the view `PeriodView` and the labels of results are part of the package.
- C10, stated: a commitment records no time.
- C11: the property test computes each customer's used and list cost from the generated lines, the expected bearer of unused cost, and bounds on pro-rata shares, and leaves an account unmapped in some rounds; refusal kinds are covered by literal tests only.
- C12, tests added: a customer bearing only unused cost under both methods, and absent when there is none; a recovered purchase with unused cost borne by a customer and pro rata, with the fee on unused cost; a line moved in the other order; superseded revisions; the tenant boundary for seals, reads and duplicates; limits of attribution, customers and digits; the period digest against every field and a byte moved; attribution entries for accounts the period lacks.
- C13: the assertion on a commitment read back compares every field; the concurrency test shows only that seals under the race detector stay consistent, both being held under N1.
