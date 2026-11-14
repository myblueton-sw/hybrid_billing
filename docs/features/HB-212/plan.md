---
type: ImplementationPlan
title: Marketplace transactions, buyer charges, seller settlement and a customer charge kept apart
description: Defines the first model of work item W-20, marketplace transactions with their role, channel, offer type and collector, buyer charge statements kept as given with source discounts and drawdown recognitions in an applied, not applied or unknown state, seller settlement statements as separate facts, and a customer charge computed under an approved policy that never re-discounts a private offer without a contract basis, never discounts lines outside its scope, never deducts a recognition and never reads settlement data.
status: draft
---

# Scope and ownership

[HB-212](https://linear.app/hybrid-billing/issue/HB-212/w-20-marketplace-transactions-buyer-charges-seller-settlement-and) is the first increment of work item W-20 (requirement HB-20 with the required contracts DB-10 and DB-14). It is built in delivery order under the owner's instructions of 2026-11-12 and 2026-11-14. Base: main `9cac406`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the model computes customer charges and keeps seller data away from customers. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-14:

- Requirement HB-20 (work item W-20): a buyer's actual charge, a seller's fee, a settlement received and a commitment drawdown recognition are different transactions; a private offer's discounted result is not discounted again. Proposed direction: TransactionRole/Channel, OfferReference, EntitlementEvidence and SettlementReference linked to their sources; public and private terms and support by contract validity shown; the product's own marketplace sales kept as later scope. Scale: an aggregation key per provider contract; buyer and seller data are not joined automatically to widen what a customer sees; one source is not summed twice. Acceptance: an actual charge of 800, separate infrastructure of 200, a virtual fee of 24 and a settlement of 776 kept as different amounts, and a drawdown recognition of 800 not treated as a discount.
- DB-10: benefit and commitment eligibility keeps applied, not applied and unknown with its evidence; a seller's fee is not passed to a customer without a contract and approval; gross, fee, refund, tax, hold, net and payout documents are kept apart from the buyer's charge; renewal, end, excess, instalments and refunds are tasks; a late settlement does not change a customer document and an unpaid settlement is never shown as received. AUD-DB-10 (NOT_RUN): unknown eligibility never becomes 0, and automatic fee pass-through, changes of customer documents by late settlement and unpaid settlement shown as received are blocked.
- DB-14: the product's own marketplace sales stay later scope; when introduced, direct licences and marketplace subscriptions are linked in one entitlement model; customer cloud spend is not used as the product's licence metric. AUD-DB-14 (NOT_RUN) is a test to run before that later feature is enabled.
- Acceptance cases MP01 to MP04 (all NOT_RUN): MP01 an MSP policy after a private offer: no second discount of the source, an additional discount only on a separate contract basis; MP02 buyer charge and seller fee: charge, fee and net settlement apart, no customer billed twice; MP03 a drawdown recognition such as MACC: the recognised amount does not reduce the charge; MP04 software discount and infrastructure: no discount applied to infrastructure outside the product's scope.
- The source document, chapters 54 to 56, read from `통합_빌링_플랫폼_구현_기획안_v1.16.docx` (paragraphs B0676 to B0703): a software price of 1,000 is charged 800 under a private offer; the source is 800 and applying 20% again to reach 640 is an error; a recognition of 800 against a commitment leaves the charge at 800; separate infrastructure use of 200 is not discounted without a basis; a virtual fee of 3% gives 24 and an expected seller receipt of 776 while the buyer is charged 800. Fees differ by product, amount, renewal, channel and region and are never one fixed rate. Minimum data: role buyer/reseller/seller, channel direct/marketplace/distributor and type; participants including the document issuer, the collector and the settlement payee; product, offer, agreement, subscription and order identifiers; contract term; benefits with eligibility state and evidence; settlement gross, fee, refund, tax, hold, net and payout document with state; policies. Buyer credentials are not assumed to read seller settlement; fees, margins and private offer terms without permission are kept out of the customer portal and of model inputs. The order of work: register the transaction type, identify the source charge and contract, check whether the source applied a discount, name the MSP's additional policy, simulate, reconcile and approve, then invoice under the party that owes payment; discount inheritance and marketplace exclusion are stated in the contract template; a discount rule for every customer is never applied unconditionally. Refunds and settlement adjustments link to the original and are separate corrections; a receipt and an issue are separate.
- Existing code: `modules/commitment` (HB-210) shows the revision and seal pattern; `packages/primitives/exact` has exact fractions and rounding to a quantum; the decimal primitive reads amounts.

# Decision

One Go package, `modules/marketplace`, standard library, the decimal primitive and `packages/primitives/exact`. A registry with a clock keeps transactions, buyer charge statements, seller settlement statements and customer charge policies; one pure function computes a customer charge. Nothing calls it and no entry point is wired.

What this can and cannot show: that a transaction's role, channel, type and collector are recorded once and decide whether the MSP may bill a customer for it at all; that a buyer charge statement is kept as given, corrected only by a new revision, and usable only when its highest revision is sealed; that no source line is current in two statements of one provider; that a recognition and a source discount keep their state, `unknown` with no amount; that a customer charge under an approved policy discounts only lines in its scope, never a private offer's line without the policy's explicit contract basis, never deducts a recognition, never adds a seller fee, and cannot read settlement data because it is given none; that a settlement statement keeps gross, fee, refund, tax, hold and net apart, checks that they add up, and is `paid` only with a payout document and a received amount; that a customer view leaves out cost, margin, list amounts, source discounts, offer references and every settlement field. It cannot know that a provider's statement is complete or true, that an account belongs to a customer, that a policy is what a contract says, that a fee is the contracted fee, or that a result was invoiced. It does not compute fees, recognitions or eligibility, reconcile a settlement against buyer charges, or convert currencies.

Names. A statement revision is `sealed` when this package's checks passed, which does not say the provider's statement is complete. A settlement statement is `consistent` when its amounts add up, which does not say they are the contracted amounts. A customer charge is `priced` when no hold applies.

Not delivered, although HB-20, DB-10, DB-14 or MP01 to MP04 name it: the product's own marketplace sales and the entitlement model of DB-14 (AUD-DB-14 stays NOT_RUN; the follow-up is W-40); fee rate schedules and expected fees; refunds, renewals, ends, excess usage and instalments as their own facts and tasks; reconciliation of settlement with buyer charges or with provider invoices (REC01, REC02); FX; FOCUS column mapping; the work queue and notifications; a fee passed to a customer under a contract (blocked here, since no input carries a fee into a customer charge); invoices, documents, screens, commands and persistence. W-20 asks for independent verification by a DBA and by QA; this ticket has one independent reviewer of design and change.

Every string is an identifier under the project's rule (letters, digits, `. _ : -`, at most 128 bytes), a currency code, a month `YYYY-MM` from 2000-01 to 2099-12, the text of a number of at most 64 bytes that the registry's decimal profile reads, a digest of 64 lowercase hexadecimal characters, or a value of a closed list. There is no free text; actor identifiers must be pseudonymous (stated obligation). No floating point is used. Number texts are stored as the decimal primitive's canonical print. Every amount is zero or above.

## Transactions

A transaction has an identifier in a workspace and is written once; an identical repeat writes nothing, anything else is a conflict. Fields:

- provider and agreement, identifiers; the agreement is the provider contract and, with the provider, the aggregation key;
- role of the MSP: `buyer` (the purchase is billed by the provider to a payer account of the MSP on behalf of an end customer, who the MSP rebills), `reseller` (the MSP resold an offer) or `seller` (the MSP is the software seller);
- channel: `direct`, `marketplace` or `distributor`;
- type: `public_offer`, `private_offer` or `channel_private_offer`; `channel_private_offer` requires role `reseller` and channel `marketplace` or `distributor`;
- collector, who collects the buyer's charge: `msp`, `marketplace` or `distributor`; `distributor` requires channel `distributor`, `marketplace` requires channel `marketplace`;
- participants, identifiers: software provider, seller, document issuer and, for roles `reseller` and `seller`, the settlement payee; reseller, optional, required for role `reseller`;
- references, identifiers: product and offer, required; subscription and order, optional;
- term: accepted, start and end months with accepted ≤ start ≤ end, and a currency.

## Buyer charge statements

A statement is a transaction and a month within its term (start to end). Its facts are recorded under revisions numbered from 1 without a gap, each written once (identical repeats return the stored revision), at the scale the provider states:

- lines, each with a source identifier unique in the revision, a kind `software`, `infrastructure` or `other`, the provider account charged, the amount charged, the list amount (the public price equivalent, optional), and the source discount: a state `applied`, `not_applied` or `unknown`, its amount required for `applied` and absent otherwise, and an evidence digest required for `applied` and `not_applied` and absent for `unknown`;
- recognitions, each with a program identifier unique in the revision (a provider commitment drawn down, such as a cloud consumption commitment), a state `applied`, `not_applied` or `unknown`, an amount required for `applied` and absent otherwise, and an evidence digest required for `applied` and `not_applied` and absent for `unknown`.

A month outside the term, or a revision beyond the next, is rejected. `Seal(workspace, transaction, month, revision)` accepts only the highest written revision; when a higher revision is written, lower revisions not judged become `superseded`; sealing a lower revision is rejected. The outcome, `sealed` or `held` with every reason that applies, is stored; a repeat returns it. Reasons, in this order:

- `duplicate_source`: a line's source identifier is in the highest written revision or the last sealed revision of another statement of any transaction of the workspace with the same provider;
- `recognition_exceeds_charge`: the applied recognitions of one program exceed the sum of the amounts charged on the statement (a recognition counts against what was bought, never more).

All seals of a workspace run under one lock. `Current(workspace, transaction, month)` returns `sealed` with the revision and its digest when the highest revision is sealed, `correction_held` when it is held, `not_judged` when it is written, and not found without a revision. `Statement(workspace, transaction, month, revision)` returns the transaction and the facts as a value only this package constructs, and only for the highest revision when it is sealed; otherwise rejected.

The statement digest is the SHA-256 of a print in which every field is its length in eight bytes, big-endian, then its bytes, numbers as canonical decimal text, absent parts as empty fields and repeated parts preceded by their count: the label `marketplace_statement_v1`, the workspace, every field of the transaction, the month, the revision, the lines in byte order of source identifier with every field, and the recognitions in byte order of program with every field.

## Seller settlement statements

A settlement statement exists only for a transaction of role `reseller` or `seller`; for role `buyer` it is rejected. It has a settlement identifier unique within its transaction and revisions numbered from 1 without a gap, each written once, with:

- a state `expected` (computed by the seller from terms), `reported` (a settlement report of the marketplace) or `paid` (a payout received);
- gross, fee, refund, tax, hold and net amounts, and a currency equal to the transaction's;
- a report document identifier, required for `reported` and `paid`, absent for `expected`; a payout document identifier and a received amount, required for `paid` and absent otherwise.

A revision's state is never lower than the state of the revision before it (`expected` < `reported` < `paid`); otherwise rejected. Each revision is judged when written: `consistent`, or `held` with every reason, in this order: `net_not_preserved` when gross − fee − refund − tax − hold ≠ net (gross below the sum of the others is this reason); `received_differs` when, for `paid`, the received amount ≠ net. `Settlement(workspace, transaction, settlement)` returns the highest revision, its state, its judgement and its digest; `Received` is true only when that revision is `paid` and `consistent`. No operation marks a settlement received otherwise.

The settlement digest is printed like the statement's, under the label `marketplace_settlement_v1`, with the workspace, the transaction's identifier, provider and agreement, the settlement identifier, the revision and every field.

Settlement data is never an input of the customer charge, of the customer view or of a statement. Reading it is a separate operation, and the registry authorizes nothing; the caller decides who may see it.

## Customer charge policies

A policy has an identifier and a version, written once with an author and approved by another actor; an identical repeat writes nothing, anything else is a conflict, and an approval by the author is rejected. Fields: the customer and the transaction it applies to, identifiers; a discount rate from 0 below 1 (provisional bound); a scope, a non-empty set of line kinds the rate applies to; `private_offer_lines`, `excluded` (default) or `included`; an evidence digest of the contract basis, required when the rate is above 0 and absent otherwise; and a quantum (`1`, `0.1`, `0.01`, `0.001`, `0.0001`) at which a discount is stated. `included` requires a rate above 0. `Policy(workspace, id, version)` returns an approved policy as a value only this package constructs; an unapproved one is rejected.

## Customer charge

`Charge(statement, policy)` is pure. The statement must be a value from `Statement` and the policy from `Policy`, each otherwise invalid; a policy for another transaction is invalid. Holds, as results, in this order: `seller_transaction` when the transaction's role is `seller` (the marketplace bills the buyer); `collected_elsewhere` when its collector is not `msp` (the customer already pays the marketplace or distributor; MP02, no second bill).

Otherwise each line's customer amount is its charged amount less its discount. A line has a discount only when its kind is in the policy's scope and it may be discounted; it may be discounted unless the transaction's type is `private_offer` or `channel_private_offer` and `private_offer_lines` is `excluded` (MP01). The discount is charged amount × rate reduced toward zero to the policy's quantum, so it is never above the charged amount and a customer amount is never above it either; a line without a discount keeps its charged amount exactly, at the provider's scale. A recognition never changes any amount (MP03); recognitions appear in the result as given, `unknown` stays `unknown` with no amount (AUD-DB-10). Lines outside the scope keep their charged amount (MP04). The total is the sum of the customer amounts.

The result carries the label `marketplace_charge_v1`; the statement's transaction, month, revision and digest; the policy's identifier, version and content digest; the hold, or per line in byte order of source its kind, charged amount, the discount (zero without one) and the customer amount; the recognitions; the total; the cost, the sum of charged amounts; and the margin, total − cost. A content digest of the result is printed like the statement's.

`CustomerView(charge)` returns only what a customer may see: transaction, month, per line the source, kind and customer amount, the recognitions, the total and the result digest. It leaves out the cost, the margin, charged amounts, list amounts, source discounts, the offer and agreement references and the policy's rate and evidence (B0697). A view is for the policy's customer only; a caller must not show it to another.

## Aggregation per provider contract

`AgreementCharges(workspace, provider, agreement, month)` returns, for the transactions of that provider and agreement whose term includes the month, in byte order of transaction: the statement state from `Current`, and for `sealed` statements the digest and the charged sums per line kind; the sums over sealed statements; and `complete`, true only when every such transaction has a sealed statement. It reads buyer charge statements only and never settlement data. An agreement without such transactions is not found.

## Error kinds, tenant boundary, time and limits

A request is judged: sizes, a limit error (lines and recognitions before they are read); form by the request alone, invalid; lookup, not found; a repeat when identical, a conflict otherwise; a stored state or rule, rejected; room, a limit error. `Charge` returns invalid for an input not of the stated form and a limit error beyond digits. Errors are fixed values. Every operation takes the workspace and reaches only its own records. The registry's time is the highest reading of its clock, in UTC, recorded on writes, seals and approvals. Limits, each required and bounded: transactions per workspace (100,000), revisions per statement or settlement (1,000), lines per revision (100,000), recognitions per revision (1,000), settlements per transaction (10,000), policies per workspace (100,000), digits of a whole number (10,000).

## What is not decided here

Provisional defaults, each to be confirmed or changed by the owner: the closed lists of roles, channels, types and collectors; private offer lines excluded from a discount by default; a discount rate below 1 and only as a discount, no markup; a discount reduced toward zero per line, which favours the MSP by less than one quantum per line; holding a seller transaction and any collector other than the MSP; `recognition_exceeds_charge` per program against the statement's charged sum. Not conservative, stated: facts are the provider's statements; a policy's contract basis is a digest the package does not read; the customer view is a filter, not an authorization.

# Allowed files

`modules/marketplace/*` (new); `docs/features/HB-212/*`.

# Verification

- The source example as literal tests: a private offer software line charged 800 with a list amount of 1,000 and a source discount of 200 `applied`, an infrastructure line of 200 `not_applied`, and a recognition of 800 `applied`; under a policy of 20% on `software` with `source_discounted` `excluded` the customer total is 1,000 (800 + 200), not 840 or 640; with `included` and a `source_basis` the software line is 640 and infrastructure stays 200; a recognition of 800 changes nothing; a settlement of gross 800, fee 24, net 776 is `consistent`, `expected` and not received; `paid` with a payout document and 776 received is received; 775 received is held `received_differs`.
- Literal tests: a `buyer` transaction with a collector other than `msp` invalid; a `reseller` transaction collected by the marketplace held `collected_elsewhere`; a `seller` transaction held `seller_transaction`; `unknown` source discounts and recognitions kept without amount; every seal and settlement reason alone and together; held corrections and `Current`; superseded revisions; a line moved between statements; settlement states moving back rejected; a settlement for a `buyer` transaction rejected; policies approved by the author rejected and unapproved policies refused; the customer view leaving out every listed field; a settlement written or changed leaving a customer charge's digest unchanged; aggregation `complete` and incomplete; digests against bytes built in the test and against every changed field; every refusal with the order of error kinds; repeats and conflicts; the tenant boundary; limits; errors as fixed values; a scan for floating point and comments; a build for linux/386.
- Concurrency under the race detector: two statements with a shared source sealed at once, neither passing; after one is corrected without the source and sealed, the other seals.
- Properties with fixed seeds: random statements and policies compared with a reference written with `math/big.Rat`, which shares the author's reading of this plan, with invariants: a recognition never changes an amount; out-of-scope lines and `source_discounted` lines under `excluded` keep their charged amount; an in-scope `unknown` line that is not `source_discounted` holds under a rate above 0; the total is the sum of the lines and never above `charged_total`.
- A mutation check by a script in Python with restores read back and the build cache emptied after every part; `gofmt -l`, `go vet ./...`, `go test -race -count=1 ./...`.

Remaining NOT_RUN at system level: MP01 to MP04, AUD-DB-10, AUD-DB-14. This ticket does not complete W-20.

# Deferred

Everything listed as not delivered above.

# Design review resolution

The independent design review of 2026-11-14 returned FAIL without need for redesign, findings F1 to F13. The rules below are part of the design and prevail over the sections above where they differ.

- F1 and F12, which lines may be discounted: a line is `source_discounted` when its source discount state is `applied`, or when the transaction's type is `private_offer` or `channel_private_offer` and the line's kind is `software` (an infrastructure or other line of a private offer is not covered by the offer, B0686). A line in the policy's scope is discounted only when it is not `source_discounted` and its state is `not_applied`, or when it is `source_discounted` and the policy's `source_discounted` is `included`. A line in scope whose state is `unknown` and that is not `source_discounted` holds the charge `source_discount_unknown` (after `collected_elsewhere`); unknown is never read as not applied. The policy field `private_offer_lines` is renamed `source_discounted`, `excluded` or `included`.
- F11, separate basis: `included` requires a second digest, `source_basis`, the contract basis for discounting a source-discounted line, absent otherwise. A rate has at most 8 decimals. The policy digest label is `marketplace_policy_v1`. A charge records the decimal profile's identifier and revision. A held charge carries the label, the profile, the statement's and the policy's identity and digests, the customer and the hold, and no lines, amounts or recognitions. The result's `cost` is named `charged_total`, the sum of the amounts the provider charged, which is not a reseller's wholesale cost.
- F2, one policy per transaction and month: a policy has a first and a last month. Approving a policy whose months overlap those of another approved policy of the same transaction is rejected, so at most one approved policy covers a transaction in a month, whatever its customer. `PolicyFor(workspace, transaction, month)` returns that policy as the value `Charge` takes; `Policy(workspace, id, version)` is not offered. `Charge` refuses, as invalid, a policy whose months do not include the statement's month. A policy is not matched to provider accounts; a transaction is billed to one customer. There is no issuance ledger: preventing a second invoice of one result is not delivered.
- F3, workspace: the statement and the policy values carry their workspace; `Charge` refuses a mismatch as invalid.
- F4, the customer view leaves out recognitions too; whose commitment a recognition draws down is not recorded, and showing a customer its own recognitions is not delivered. A held charge has no customer view (rejected).
- F5, aggregation: sums are grouped by role and currency, in byte order of role then currency; entries carry the role and currency.
- F6, collector: the collector is the party that invoices the MSP's end customer for this charge, not the party the provider invoices. The source example is recorded as role `buyer`, channel `marketplace`, type `private_offer`, collector `msp`: the marketplace bills the MSP's payer account and the MSP rebills its customer. `seller_transaction` holds because the MSP's own sale is revenue settled with the marketplace and is not a cost this function rebills. The collector is a fact the caller records (not conservative).
- F7, seals: `Seal` of a held highest revision judges it again and replaces the outcome; a sealed one returns its outcome. Writes, seals and every other operation run under one registry lock. Cost, stated: a highest revision of another statement that is never judged keeps blocking a line it shares.
- F8, duplicates: the key is the provider and the source identifier across months, transactions and agreements; this assumes a provider does not reuse a line identifier (provisional, to be checked against real samples). Duplicates with other modules, such as the same infrastructure line imported as ordinary usage, are not detected (not delivered).
- F9, settlement: the formula net = gross − fee − refund − tax − hold is provisional; a negative net, reversals and clawbacks are not delivered and a settlement cannot move back from `paid`. A `paid` revision whose payout document is used by the highest revision of another settlement of the same provider and agreement is held `payout_reused` (after `received_differs`); payout totals are not checked against the payout document itself.
- F10, not delivered, added: an issuance ledger; account-to-customer mapping; rebilling or delta handling when a sealed statement is corrected after a charge was issued; rounding to a currency's minor unit (customer amounts stay at the provider's scale); markups and any margin model; duplicates across modules; payout aggregation, reversals and negative net; showing public and private terms and support by contract validity beyond the type and term; corrections of a transaction, which is write-once (a wrong collector or term is permanent until a new transaction is recorded); the HB-20-D design task, still DRAFT, and the pilot roles, offers, fees and permissions, which stay open with the owner.
- F13, contract: `ApprovePolicy(workspace, id, version, approver)`: a repeat by the same approver returns the stored state, another approver is a conflict, the author is rejected, an overlap is rejected. A statement value is a copy and does not say whether it is still current; a consumer compares the digest a result names with `Current`. Limits fit a 32-bit `int`. The list amount, the source discount amount and the charged amount are not checked against each other. Whether an `other` line holds a marketplace fee is the caller's responsibility; the package cannot tell.

# Design re-review resolution

The independent re-review of 2026-11-14 returned FAIL with findings N1 to N6, none needing a redesign. The rules below prevail over the sections above where they differ.

- N1, ending a policy: a policy's months must lie within its transaction's term (rejected otherwise, when recorded). Approving a policy is rejected when its months overlap an approved policy of the same transaction with another identifier. A new version of the same identifier whose first month is later than an approved version's first month supersedes that version from its first month: on approval, the older version's effective last month becomes the month before the new first month. A new version with the same or an earlier first month overlapping an approved version is rejected. `PolicyState` shows the effective last month; `PolicyFor` and `Charge` use it. Charges already computed under the older version keep its digest; there is no issuance ledger to rebill them (stated).
- N2, verification: the property invariants are restated with `source_discounted` and `charged_total`. Literal tests added: an `applied` line on a public offer kept undiscounted under `excluded`; an in-scope `unknown` line holding `source_discount_unknown` and an out-of-scope `unknown` line not holding; a private offer's infrastructure line in scope with `not_applied` discounted under `excluded`; `included` without `source_basis` invalid; an overlapping approval rejected, a superseding version, and `PolicyFor`; a month outside the policy and a workspace mismatch refused; a held highest revision judged again and sealed; `payout_reused`; aggregation split by role and currency; a held charge refused a view; `ApprovePolicy` repeat and conflict; the order of the three holds. In the source example the infrastructure line's source discount is `not_applied` with evidence, the software line's is `applied` 200, and the policy covers the statement's month.
- N3, payout reuse: a payout document is checked across every settlement of the same provider, all agreements. A held settlement is judged again only when a new revision is written. Payouts that legitimately aggregate several settlements hold `payout_reused` and never become received until payout aggregation is delivered (stated cost).
- N4, holds: a charge carries the first hold in the order `seller_transaction`, `collected_elsewhere`, `source_discount_unknown`. A policy with rate 0 discounts nothing and never holds `source_discount_unknown`.
- N5: the role definition in Transactions is amended in place.
- N6: the race test also checks that after one statement is corrected and sealed the other seals.

# Second re-review resolution

The second re-review of 2026-11-14 returned PASS with three minor findings, resolved as follows.

- M1: a charge records the policy's first and effective last month, in its fields and its digest. A policy value is a copy; a consumer compares a result's policy months with `PolicyState`, as it compares the statement digest with `Current`. A test computes a charge with a value fetched before a superseding version was approved.
- M2: a `buyer` transaction whose collector is not `msp` is invalid when recorded; `collected_elsewhere` applies to `reseller` transactions.
- M3: the Verification bullets use `source_discounted` and `source_basis`.

# Change review resolution

The independent change review of 2026-11-14 returned PASS with six minor findings, C1 to C6, resolved as follows. These corrections were not reviewed again.

- C1, participants: a reseller participant is allowed only on role `reseller`, and a settlement payee is not allowed on role `buyer`. The code enforces both, and this plan now states them.
- C2, digests of unsealed revisions: `Current` and `AgreementCharges` also give the digest of a revision that is `not_judged` or `correction_held`, so that it can be identified. Only a `sealed` state makes a revision usable, and `Statement` refuses any other.
- C3, groups: `AgreementCharges` creates a role and currency group only from sealed statements, so a sum of 0 is a sealed 0; `TestGroupsInOrder` leaves one transaction unsealed.
- C4, registries: values of `Statement` and `Policy` are compared by workspace and transaction, not by registry. One process is assumed to hold one registry per workspace (stated assumption).
- C5, digests: tests now vary every line, recognition, month, revision, workspace, role and collector field of a statement, every policy field, every settlement field, and the policy and customer of a charge.
- C6, the scan for floating point and comments reads every non-test Go file of the package.
