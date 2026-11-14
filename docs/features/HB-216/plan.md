---
type: ImplementationPlan
title: Prepaid and credit balance ledger with opening references, atomic applications, expiry and explicit reversals
description: Defines the first model of work item W-22, a balance ledger per customer, legal entity, currency and balance kind, with an approved opening balance reference that blocks re-imported past transactions, approved increases kept as lots with optional expiry, applications to a target document reserved and confirmed atomically so that they never exceed the balance or the target, explicit approved reversals, the per-currency balance identity, and accounts under an external cash authority that hold every application.
status: draft
---

# Scope and ownership

[HB-216](https://linear.app/hybrid-billing/issue/HB-216/w-22-prepaid-and-credit-balance-ledger-with-opening-references-atomic) is the first increment of work item W-22 (requirement HB-22 with the required contract QA-09 and the implementation deliverable "BalanceReference, CreditApplication and cancelling reversals"). It is built in delivery order under the owner's instructions of 2026-11-12 and 2026-11-14. Base: main `e0790d1`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the ledger decides how much of a customer's money or credit is applied. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-14:

- W-22 design deliverable (HB-22-D, DRAFT): discounts, included quantities, cash prepayments, credit documents and refunds are separate, and there is one authoritative cash ledger; a dispute does not cancel an issued document or the obligation to pay. Open: the cash authority system, external reservation contracts, promotion expiry and the dispute SLA.
- W-22 implementation deliverable (HB-22-I): balances are tracked with BalanceReference, CreditApplication and cancelling reversals; an external balance that cannot be reserved or locked, or that is stale, holds confirmed application; BillingCase separates document and line, owner, due time, public answer and internal note.
- W-22 verification (HB-22-V): applying 60 and 50 at once to 100 USD applies at most 100 in total and leaves 10 unpaid; the same reversal is not repeated; inquiry permissions. Scale: atomic reservation per customer, legal entity and currency, or per included pool and unit, recording applications without loading every charge.
- QA-09: an OpeningBalanceReference fixes the prior ledger, as-of date, currency, evidence, approval, migration key and the range of past transactions it includes; an opening balance and a re-import of past transactions are not both counted; per currency, opening + increase − application − expiry ± explicit reversal = closing; an external cash or receipt authority is not replaced by a platform balance; going live for contracts that use this is proven separately. AUD-QA-09 (NOT_RUN): opening import and the re-import of included past transactions do not increase twice, and the identity holds after concurrent use, expiry and reversal.
- The other detail contracts of W-22 (AR-08 BillingFinding, AR-10 fixed and usage components and SLA credits, DB-10 benefit eligibility and seller fees) and the payment terms of `payment-terms.md` (PT-01 to PT-08) are not in this ticket.
- Existing code: `modules/claims` (HB-214) records what was billed per scope and states that prepaid balances belong to W-22; `modules/marketplace` (HB-212) keeps marketplace charges apart. Nothing keeps balances.

# Decision

One Go package, `modules/balances`, standard library and the primitives `decimal` and `exact`. A registry with a clock keeps the accounts of a workspace with their opening reference, increases, applications and reversals. Nothing calls it and no entry point is wired.

What this can show: that an application never takes more than the available balance of its account or the stated room of its target, also when requests arrive at once; that a past transaction the opening reference already includes cannot be imported again; that every movement is approved where it adds money back or creates it; that a reversal is never applied beyond what it reverses; that the closing balance always equals the identity. What it cannot know: that a receipt, a credit document or an opening balance is true, that a target's amount is right, or that the external authority agrees. It does not issue documents, refund money, compute taxes or talk to an ERP.

Every string is an identifier under the project's rule (letters, digits, `. _ : -`, at most 128 bytes), a currency code of three capital letters, a date `YYYY-MM-DD` from 2000-01-01 to 2099-12-31 that exists in the Gregorian calendar, the text of a number of at most 64 bytes that the registry's decimal profile reads, a digest of 64 lowercase hexadecimal characters, or a value of a closed list. There is no free text; customer, entity and actor identifiers must be pseudonymous (stated obligation). No floating point is used. Amounts are stored as the decimal primitive's canonical print; every amount is zero or above, and an amount that must be positive is stated.

## Accounts

`OpenAccount(workspace, Account, author)` records an account, written once:

- an identifier unique in the workspace;
- the customer, the legal entity and the currency;
- a balance kind, `cash_prepayment` or `credit`;
- an authority, `platform` (this ledger keeps the balance) or `external` with the identifier of the external ledger (the balance is kept elsewhere).

At most one account per workspace has a given customer, entity, currency and kind, so there is one ledger for one balance; a second account with the same key under another identifier is rejected. An identical repeat returns the stored account; another request under its identifier is a conflict. Discounts and included quantities are not balances and are not recorded here.

## Opening balance reference (QA-09)

`RecordOpening(workspace, Opening, author)` records the opening of an account, at most one per account, written once: the prior ledger identifier, the as-of date, the amount (zero or above), the evidence digest, a migration key unique in the workspace, the first date of the past transactions it includes (on or before the as-of date) and an optional expiry date. Its currency is the account's. `ApproveOpening(workspace, account, approver)` by another actor makes it effective. The opening is rejected for an external account or for an account that already has an increase or an application recorded in any state. While an opening is recorded and not approved, increases and reservations on the account are refused (`opening_pending`). An approved opening with an amount above zero becomes the account's first lot.

A past transaction carries an import: the prior ledger identifier, the transaction identifier and the date it occurred (not after the registry's date). An increase or an application with an import is rejected when the account's approved or pending opening has an as-of date on or after the date it occurred, so a transaction the opening includes is never counted again; one occurring after the as-of date is accepted. The same prior ledger and transaction identifier may be imported once per account, among increases and applications together; a second import is a conflict. The included first date is recorded and printed as evidence; transactions before it are also refused, because they are on or before the as-of date.

## Increases and lots

`RecordIncrease(workspace, Increase, author)` records an increase, written once under its own identifier: the account, a kind, a positive amount, a reference identifier (the receipt or credit document), an optional expiry date and an optional import. Kinds by balance kind: `cash_receipt` for `cash_prepayment`; `credit_document`, `promotion` and `sla_credit` for `credit`. A `promotion` requires an expiry date. `ApproveIncrease(workspace, identifier, approver)` by another actor makes it a lot of the account. An increase is rejected on an external account; its approval is rejected when its expiry date is before the registry's date. A lot is usable through its expiry date, and expires at 00:00 UTC of the following day (provisional: UTC, not the contract's time zone). Lots are numbered in order of approval.

## Applications (CreditApplication)

An application takes money from an account for a target, the line of a document: the document and line identifiers and the target's amount. `Reserve(workspace, Application, actor)` judges an application, written once under its own identifier, with the account, the target, a positive requested amount, whether a partial application is accepted, a reservation lifetime in seconds from 1 to 86,400 and an optional import. It returns a decision with a refusal reason, storing nothing, or the application stored as `reserved` with the amount applied, in this order:

1. `external_authority`: the account's authority is external; the application is held, because there is no reservation or lock with the external ledger in this increment;
2. `opening_pending`: the account has a recorded opening not yet approved;
3. `currency_mismatch`: the target was recorded with another currency;
4. `target_mismatch`: the target was recorded with another amount;
5. `target_settled`: the target has no room left;
6. `exceeds_target`: the application is not partial and the requested amount exceeds the target's room;
7. `nothing_available`: the account has nothing available;
8. `insufficient`: the application is not partial and the requested amount exceeds what is available.

The applied amount is the least of the requested amount, the target's room and the available balance. A target's room is its amount less the applied amounts of its active applications, from all accounts with its currency, less nothing for released or expired ones and plus the effective reversals of its confirmed ones. A target's amount and currency are fixed by the first reserved application of the target and may be set again only while the target has no application that is active or was ever confirmed. The available balance is the sum over unexpired lots of their remainder less their holds.

The applied amount is allocated to lots in order of expiry instant, earliest first and lots without expiry last, then by lot number, as holds on those lots. The reservation expires at the earlier of the registry's time plus its lifetime and the earliest expiry of the lots it holds, so a reservation never outlives its lots. `Confirm(workspace, identifier, actor)` turns an unexpired reservation into `confirmed` and its holds into consumption; `Release(workspace, identifier, actor)` turns it into `released` and frees its holds. A reservation past its expiry reads as `expired` and holds nothing. Confirming an expired, released or confirmed application is rejected except an identical confirm of a confirmed one, which returns it; releasing a released one returns it and any other state is rejected.

Concurrent requests are judged one after another under the registry lock: 100 USD available, with applications of 60 and 50 accepted partially, applies 60 and 40 and leaves 10 of the second target unpaid, in whichever order they arrive.

## Explicit reversals

`Reverse(workspace, Reversal, author)` records a reversal as `pending`, written once under its own identifier, of one of three kinds with a positive amount: `application` (a confirmed application), `increase` (an approved increase) or `opening` (the approved opening of an account, named by the account). `ApproveReversal(workspace, identifier, approver)` by another actor makes it effective.

- An application reversal is limited, at record and at approval, to the application's applied amount less its effective reversals. When effective, it returns the amount to the lots the application consumed, last allocated first, and frees the same amount of the target's room. An amount returned to a lot that has expired is expired at once.
- An increase or opening reversal is limited, at record and at approval, to the lot's remainder less its holds, and the lot must be unexpired at approval. When effective, it lowers the lot's remainder.

Several pending reversals of one target may each reach the whole remaining amount; the first approved takes effect and a later approval that would exceed it is rejected. An identical repeat returns the stored reversal; another request under its identifier is a conflict; approval by the recording actor is rejected; a repeat approval by the same approver returns it and another approver is a conflict. A reversal approved in error cannot be undone.

## Balance identity

`Statement(workspace, account)` returns, for a platform account at the registry's time, in the account's currency: the opening amount, the approved increases, the confirmed applications, the expired amounts, the effective application reversals, the effective increase and opening reversals, the closing balance, the held amount and the available amount. The closing balance is the sum of the lots' remainders; the identity `closing = opening + increases − applications − expired + application reversals − increase and opening reversals` is checked by the tests and by the property test. `available = closing − held − expired holds` is not needed, because holds end at the expiry of their lots. A lot's expired amount is its remainder at its expiry instant, which no later movement except a returned reversal can change; amounts are read when asked and stored only by movements. `Statement` of an external account is rejected.

## Reads and digests

`Account`, `Opening`, `Increase`, `Application`, `Reversal` and `Lots(workspace, account)` return the stored records with their states, actors, times and digests; `Lots` returns each lot's number, source, amount, remainder, held amount, expired amount and expiry. A digest is the SHA-256 of a print in which every field is its length in eight bytes, big-endian, then its bytes, numbers as canonical decimal text and absent parts as empty fields, under the labels `balance_account_v1`, `balance_opening_v1`, `balance_increase_v1`, `balance_application_v1` and `balance_reversal_v1`, printing the workspace and every field of the record except an application's lifetime; states, actors, times and allocations are not printed.

## Error kinds, tenant boundary, time and limits

A request is judged: sizes, a limit error; form by the request alone, invalid; lookup, not found; a repeat when identical, a conflict otherwise; a stored state or rule, rejected; room, a limit error. Refusals of `Reserve` are results, not errors. Errors are fixed values. Every operation takes the workspace and reaches only its own records. All operations run under one registry lock (provisional: the deliverable asks for atomicity per customer, entity and currency; one lock is stricter). The registry's time is the highest reading of its clock, in UTC, read once per call. Limits, each required and bounded within a 32-bit `int`: accounts per workspace (100,000), lots per account (100,000), applications per workspace (1,000,000, released and expired ones included), targets per workspace (1,000,000), reversals per target (1,000).

## What is not decided here

Provisional defaults, each to be confirmed or changed by the owner: the balance kinds and increase kinds; one account per customer, entity, currency and kind; expiry at 00:00 UTC after the expiry date; allocation by earliest expiry; a reservation capped at its lots' expiry; an amount returned to an expired lot expiring at once; holding every application on an external account; approval by a second actor for openings, increases and reversals but not for applications; the lifetime bound of one day. Not conservative, stated: a past transaction imported without an import record is not detected as included in the opening; a target's amount is what the caller states; refunds, which lower a cash balance, are not delivered and cannot be expressed except as an increase reversal.

# Allowed files

`modules/balances/*` (new); `docs/features/HB-216/*`.

# Verification

- Literal tests: 60 and 50 partial on 100 USD in both orders, applying 100 and leaving 10 unpaid; the same with goroutines under the race detector; an opening as of 2026-09-30 refusing an import that occurred on 2026-09-30 and accepting one on 2026-10-01; a second import of the same transaction a conflict; expiry of a promotion between two applications with the expired amount in the statement; a reservation capped at its lot's expiry; a reversal approved twice and a second reversal beyond the remainder rejected; a reversal returned to an expired lot; an external account holding every application; every refusal reason alone and in order; every error with the order of error kinds; repeats and conflicts; approvals by the author; digests against bytes built in the test; the tenant boundary; limits; leap days; a scan for floating point and comments; a build for linux/386.
- Properties with fixed seeds: random sequences of every operation and clock moves, checked after every step against a reference model written in the test with `math/big.Rat`, with invariants read from the registry: the identity holds per account; available is never below zero; the applied amounts of a target never exceed its amount; effective reversals never exceed what they reverse.
- A mutation check by a script in Python with restores read back and the build cache emptied after every part; `gofmt -l`, `go vet ./...`, `go test -race -count=1 ./...`.

Remaining NOT_RUN at system level: AUD-QA-09, AUD-AR-08, AUD-AR-10, AUD-DB-10, PT-01 to PT-08. This ticket does not complete W-22.

# Deferred

BillingFinding (AR-08), BillingCase and disputes, AR-10, DB-10, payment terms and aging, refunds, reservation and lock with an external authority, balances of included pools, persistence, screens, independent DBA and QA verification.

# Design review resolution

The independent design review of 2026-11-14 returned FAIL with findings F1 to F12, none needing a redesign. The rules below are part of the design and prevail over the sections above where they differ.

- F1, customer boundary: a target belongs to the customer, the legal entity and the currency of the account of its first reserved application, with its amount, all fixed together and set again only under the same rule as the amount. A reservation from an account of another customer or entity is refused `customer_mismatch`, before `currency_mismatch`.
- F2, one increase per document: the pair of increase kind and reference is unique in the workspace among increases in any state; a second increase with the same pair under another identifier is a conflict.
- F3, imports across accounts: the pair of prior ledger and transaction identifier is unique in the workspace among increases and applications of all accounts; a second import is a conflict. An import is also rejected when any opening of the workspace, pending or approved and not withdrawn, has the same prior ledger and an as-of date on or after the date the transaction occurred, whatever its account. An import whose prior ledger appears in no opening is accepted (not conservative, stated).
- F4, total order. `Reserve`: size; form (an import with partial acceptance is invalid); lookup of the account; repeat when identical, conflict otherwise, on the application identifier; a conflict for an import already recorded; rejected for an import occurring after the registry's date or included by an opening; then the refusals as results in the order `external_authority`, `opening_pending`, `customer_mismatch`, `currency_mismatch`, `target_mismatch`, `target_settled`, `exceeds_target`, `nothing_available`, `insufficient`; then room. A refusal stores nothing, and a refused request may be sent again under the same identifier and is judged again. `RecordIncrease`: size; form; lookup of the account; repeat or conflict on the identifier; conflict for a kind and reference or an import already recorded; rejected for an external account, a kind not of the account's balance kind, a pending opening, an expiry date before the registry's date, or an import occurring after the registry's date or included by an opening; room. `RecordOpening`: size; form; lookup; repeat or conflict on its identifier; conflict for a migration key already recorded in the workspace; rejected for an external account, an account with a pending or approved opening, an account with an increase or application in any state, or an expiry date before the registry's date; room. Approvals: form; lookup; repeat or conflict; rejected for the author or a state or date rule.
- F5, closing: the closing balance is the sum of the remainders of unexpired lots. From its expiry on, a lot's remainder is its expired amount, and an amount returned to it later adds to its expired amount. `Lots` reports an expired lot with a remainder and available amount of zero and its expired amount.
- F6, half-open times: a lot and a reservation are in force while the registry's time is before their expiry and expired from their expiry on. A confirm at the exact instant of expiry is rejected; a test covers it.
- F7, expiry dates: an opening or an increase whose expiry date is before the registry's date is rejected when recorded and when approved. An increase or opening reversal of a lot that has expired is rejected when recorded and when approved.
- F8, withdrawal: an opening has its own identifier unique in the workspace. `WithdrawOpening(workspace, identifier, actor)` turns a pending opening into `withdrawn`, by any actor (provisional); a repeat returns it and an approved one is rejected. A withdrawn opening keeps its migration key and does not block a new opening or the account's increases and reservations.
- F9, names and limits: "reversals of one target" is renamed reversals of one reversed record (an application, an increase or an opening). Limits: accounts per workspace (100,000), openings per account (100, withdrawn included), increases per account (100,000, pending included), applications per workspace (1,000,000), targets per workspace (1,000,000), reversals per reversed record (1,000), reversals per workspace (1,000,000).
- F10, prints and repeats: printed fields in order, after the label and the workspace. Account: identifier, customer, entity, currency, balance kind, authority, external ledger. Opening: identifier, account, prior ledger, as-of date, amount, evidence, migration key, included first date, expiry date. Increase: identifier, account, kind, amount, reference, expiry date, import ledger, import transaction, import date. Application: identifier, account, document, line, target amount, requested amount, partial as `true` or `false`, import ledger, import transaction, import date. Reversal: identifier, kind, reversed record, amount. An identical repeat compares the printed fields and, for an application, the lifetime; the actor is not compared. An identical confirm is any `Confirm` of a confirmed application; the actor is not compared.
- F11, honest deferrals: a refusal `external_authority` stores no hold record; a hold record, reservation and lock with an external ledger and the staleness of an external balance are not delivered. Inquiry permissions of HB-22-V are deferred with BillingCase.
- F12: an application with an import must not accept a partial application; otherwise it is invalid.

# Design re-review resolution

The independent re-review of 2026-11-14 returned PASS with five minor findings, resolved as follows.

- N1: an expired lot's remainder is zero and its expired amount holds what remained at its expiry, plus any amount returned to it later. The closing balance is the sum of the remainders of unexpired lots.
- N2: `WithdrawIncrease(workspace, identifier, actor)` turns a pending increase into `withdrawn`, by any actor (provisional); a repeat returns it and an approved one is rejected. A withdrawn increase does not count in the uniqueness of kind and reference or of imports, so a corrected increase may be recorded again; it still counts toward the limit of increases per account.
- N3: openings are named by their own identifier in `ApproveOpening(workspace, identifier, approver)`, in `WithdrawOpening`, in an `opening` reversal and in its print.
- N4: a withdrawn opening does not count in the uniqueness of migration keys, so a corrected opening may reuse its key.
- N5, tests added to the literal tests: `customer_mismatch`; a duplicate kind and reference; an import blocked by another account's opening with the same prior ledger; withdrawal of an opening and of an increase and the keys and imports they free; rejection of an expired date at record and at approval; a confirm at the exact expiry instant of a reservation capped at its lot. The property test includes both withdrawals.

# Implementation notes

Two points the resolutions left open were settled during implementation, in the conservative direction, and are stated here for the change review.

- An imported application holds its import while it is reserved and unexpired or confirmed. A released or expired one frees it, as a withdrawn increase does (N2), so the corrected transaction may be imported again. A refusal stores nothing and holds no import.
- `Confirm` of an application that is already confirmed returns it, whoever calls (F10). `Release` of a released one returns it; of a confirmed or expired one it is rejected.

# Change review resolution

The independent change review of 2026-11-14 returned PASS with one major finding against the approved design and three minor ones, resolved as follows. The corrections were not reviewed again.

- C1 (major, a gap in the design): an opening recorded after an import of its prior ledger counted that transaction twice, because coverage was judged only when the import arrived. `RecordOpening` and `ApproveOpening` now reject an opening when the workspace holds an import of the same prior ledger occurring on or before its as-of date, from an increase that is not withdrawn or from an application that is reserved and unexpired or confirmed. A withdrawn increase or a released or expired application does not block it. Literal tests cover the order import then opening, a reserved, expired and confirmed imported application and a withdrawn increase; the test fails when the check is removed.
- C2: identifiers above 128 bytes and numbers above 64 bytes are invalid, as in `modules/claims`; "sizes, a limit error" in the error order means counts in a request or in the registry, not the length of a field.
- C3: the property test now judges independently of the code: the expected refusal reason of every reservation from the views of the target's applications and the available balance; the expected error of every imported increase (a conflict for an active transaction, rejected while an opening of the ledger covers its date); whether each reversal is within its limit, from the views; that every lot is expired exactly from the day after its expiry date with nothing remaining, available or held; and that the held amounts of the lots equal the applied amounts of the account's reservations. Every refusal reason except `external_authority` and `opening_pending`, both import errors and every operation are reached at least five times. Breaking the opening exclusion makes it fail. The available balance itself is still read from the registry and checked through the lots and the identity.
- C4 (scale): holds are now computed from the account's live reservations, pruned on every reservation, instead of every application the account ever had. One registry lock and a scan of lots per reservation remain (stated cost; per-account locks and stored held totals are deferred with persistence).
