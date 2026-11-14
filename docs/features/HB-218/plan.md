---
type: ImplementationPlan
title: Contract payment terms, issue-date due snapshots, extensions, outstanding amounts, overdue status and aging
description: Defines the payment-term increment of work item W-22, a receivable register that binds approved payment terms to contracts in a contract time zone, fixes each issued document's term and due dates at issuance, applies approved due-date extensions, keeps allocations and their reversals from the authoritative ledger, and derives due, overdue, settled, not applicable and unknown status and per-currency aging with dispute and collection holds on a separate axis.
status: draft
---

# Scope and ownership

[HB-218](https://linear.app/hybrid-billing/issue/HB-218) is the follow-up of HB-217 for the rest of work item W-22. This increment covers the payment-term addition of `docs/planning/v1.16/payment-terms.md` and its English contract `docs/contracts/billing/payment-and-recognition.md` at package level (PT-01, PT-02, PT-03, PT-04, PT-06, PT-07 and part of PT-08). PT-05 (portal and permission boundary) and the screens are later. AR-10, DB-10, refunds and external authority locks stay in HB-218 follow-ups. Base: main `5dcffc2`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the register decides amounts owed and overdue status. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-14 from the two documents above:

- A contract revision carries a term ID, revision and name, the issue-date basis, nonnegative integer net days, calendar or business-day mode, the contract time zone, the holiday calendar revision, effective dates, a reason and an approval. Net 30 and 45 are examples, not defaults. Calendar days are the proposal. Business days need an approved calendar and are not enabled without one. Month-end terms, installments and early-payment discounts are out of scope.
- At issuance the document keeps the confirmed issue date, the contract and term revision, the original and the current effective due date, and the zone and calendar. Changed terms never recalculate issued documents. A draft has an estimated due date and never a confirmed overdue status. An internal allocation with no payment obligation is not applicable. Missing or contradictory terms block issuance.
- Extensions need separate authority, approval, a reason and an effective time, and keep the original due date and past overdue evidence. Redelivery does not restart the clock. Replacements and corrections are linked, and the due-date inheritance is an explicit choice.
- The calendar computation is due = issue date in the contract zone + net days, with the issue date as day 0. Payment on the due date is on time until that date ends; the next local date starts overdue. Examples: 2026-10-01 + 30 = 2026-10-31; + 45 = 2026-11-15.
- Outstanding is the confirmed receivable less confirmed allocated receipts, credits, cancellations and corrections, with linked reversals counted once, exact per currency. Unallocated cash, promises and deliveries are not settlement. Overpayment is a separate credit, never a negative overdue.
- Overdue needs a passed effective due date, a positive outstanding and current authoritative receipt evidence. Partial payment leaves only the rest overdue. The states are not due, due today, overdue, settled, not applicable and unknown. Days are calendar days in the contract zone. Disputes and collection holds never remove the obligation or the overdue fact. No late fees, interest or debit.
- Missing or stale receipt state is unknown (payment confirmation needed). The last known balance, elapsed days and confirmation time are shown apart from confirmed overdue totals. Economic receipt time and arrival time are both kept. A reversed receipt is reevaluated against the current effective due date.
- Aging buckets: not due, due today, 1-30, 31-60, 61-90, 91 and more, and unknown kept apart. Totals match the same snapshot per currency.
- Open items: the calendar and holiday handling, the maximum net days, the receipt authority and freshness limit, the extension approval role, notifications and the refresh SLO.
- Existing code: `modules/cases` (HB-217) gives the dispute axis (`Disputes`); `modules/balances` (HB-216) keeps prepaid and credit balances. Nothing keeps receivables.

# Decision

One Go package, `modules/receivables`, with the standard library, `time/tzdata` (embedded zone data, so zone lookup does not depend on the host), and the primitives `decimal` and `exact`. It has a registry with an injected clock (UTC, never earlier than its highest reading) and one mutex. Nothing calls it, and no entry point is wired. It imports no other module than the primitives. Disputes reach it only as holds recorded by the caller.

Every string must be one of the following:

- an identifier under the project's rule (letters, digits, `. _ : -`, at most 128 bytes);
- a currency code of three capital letters;
- a date `YYYY-MM-DD` from 2000-01-01 to 2099-12-31 that exists;
- a decimal text of at most 64 bytes that the registry's decimal profile reads;
- a digest of 64 lowercase hexadecimal characters;
- an IANA zone name that `time.LoadLocation` resolves (not `Local`, and at most 64 bytes);
- a value of a closed list.

There is no free text and no floating point. Actors are identifiers trusted as stated.

Errors are `ErrInvalid` (form or size), `ErrNotFound`, `ErrConflict` (other content under an identifier), `ErrRejected` (state or rule; a refusal that the plan names carries a reason code in the returned view where one is listed) and `ErrLimit`. Checks run in this order: form, lookup, identical repeat or conflict, rule, room. An identical repeat by the same actor returns the stored record.

## Terms

`RecordTerm(ws, Term{ID, Revision, NetDays, Mode}, actor)`:

- the revision is an integer from 1;
- net days run from 0 to `Limits.NetDays`;
- the mode is `calendar` or `business_day`. `business_day` is recorded but can never be approved while no calendar is supported: `ErrRejected` at approval.

The pair (ID, revision) is written once. `ApproveTerm(ws, id, revision, actor)` needs an actor other than the recorder.

## Contract bindings

`Bind(ws, Binding{ID, Contract, Term, Revision, Zone, From, To, Reason}, actor)` binds a term revision to a contract for the effective dates From to To. To is inclusive and optional (open-ended). The reason is a code: `initial`, `renegotiation`, `correction`. The binding starts pending.

`ApproveBinding(ws, id, actor)` needs:

- another actor than the recorder;
- an approved term;
- no other pending or approved binding of the contract whose dates overlap.

`WithdrawBinding(ws, id, actor)` works while pending. An approved binding is never changed: a later revision ends a period only through a new binding. Closing an open-ended binding is `EndBinding(ws, id, to, actor)`, recorded by one actor and approved by another (`ApproveEnd`). The new end must not be before any document already issued under it.

## Issuance snapshot

`Issue(ws, Receivable{Document, Customer, Entity, Currency, Contract, Amount, IssuedAt, Obligation, Replaces, DueBasis}, actor)`:

- `Obligation` is `payable` or `internal`. An internal receivable has no term, no due date and is always `not_applicable`. A payable amount is positive.
- `IssuedAt` is the confirmed issue time. The issue date is its calendar date in the zone of the binding whose dates cover that date. Each candidate binding's zone is used to compute the date it would cover.
- Exactly one approved binding of the contract must cover the issue date:
  - none gives reason `term_missing`;
  - a pending binding covering it gives `term_pending`;
  - more than one gives `term_conflict`.

  Each of these is `ErrRejected`, and nothing is recorded.
- The snapshot keeps:
  - the binding, the term ID and revision, the net days and the mode;
  - the zone name, the issue date and the issued-at time;
  - the original due date (issue date + net days, calendar);
  - the effective due date (equal at issuance).
- `Replaces` names an issued document of the same customer, entity, currency and contract. `DueBasis` must then be `inherit` (copy the replaced document's original and effective due dates) or `reissue` (compute from the new issue date). Without `Replaces`, `DueBasis` is empty. The replaced document stays as it is. Its cancellation is an allocation of kind `cancellation`, recorded separately.
- The document identifier is written once per workspace. A second issue of the same content by the same actor returns it (redelivery does not restart the clock); other content is a conflict.

`Estimate(ws, contract, date)` returns the estimated due date of an approved binding covering that local date, or the same refusal reasons. Nothing is recorded, and the result is marked as an estimate.

## Extensions

`Extend(ws, Extension{ID, Document, Due, Reason}, actor)` records a pending extension of a payable document:

- the new due date must be after the current effective due date;
- reason codes are `customer_request`, `dispute_resolution`, `billing_error`, `commercial_decision`.

`ApproveExtension(ws, id, actor)` needs another actor. It re-checks that the new date is still after the current effective due date, then sets the effective due date, with history. `WithdrawExtension` works while pending. The original due date never changes. There is no shortening and no cancellation of an approved extension in this increment, so a later revision needs a new extension (provisional).

## Allocations and reversals

`Allocate(ws, Allocation{ID, Document, Kind, Amount, Effective, Source, Ledger, Revision, Evidence}, actor)`:

- kinds are `receipt`, `credit`, `cancellation`, `correction`; the amount is positive;
- `Effective` is the economic date;
- `Source` is `authority` (synchronized from the authoritative ledger, with ledger and revision identifiers) or `manual` (needs evidence and approval by another actor through `ApproveAllocation`, and stays pending until then);
- the registry keeps the arrival time.

An allocation that would make the outstanding amount negative is `ErrRejected` (`exceeds_outstanding`). It is judged at recording for authority allocations and at approval for manual ones. Overpayment belongs to the cash ledger, not here. An internal document refuses allocations.

`Reverse(ws, Reversal{ID, Allocation, Effective, Ledger, Revision}, actor)` reverses an effective allocation in full, once (a chargeback or a returned payment). A second reversal of the same allocation is `ErrRejected`.

Outstanding = amount − effective allocations + their reversals. It is exact, and its print uses the decimal profile's digits.

## Authority freshness

`Confirm(ws, entity, currency, asOf, actor)` records that the authoritative ledger was read completely as of `asOf` for that entity and currency. `asOf` must not be after now, and confirmations never move backwards (`ErrRejected`). `Limits.Freshness` (a duration from 1 minute to 31 days) is the provisional freshness limit.

## Status

`Status(ws, document)` evaluates at the registry's now:

- `not_applicable` for internal documents;
- else `settled` when the outstanding amount is zero;
- else `unknown` when no confirmation exists for the document's entity and currency, or `now − asOf > Freshness`;
- else, with `today` the date of now in the snapshot zone: `not_due` before the effective due date, `due_today` on it, and `overdue` after it, with days = today − effective due date, computed from calendar dates and immune to DST.

The view also carries:

- the outstanding amount, the last confirmation time and the elapsed days since the effective due date (also when unknown);
- the snapshot;
- the extension, allocation and reversal histories;
- the holds.

`Hold(ws, Hold{ID, Document, Kind, Reference}, actor)` records a hold of kind `dispute` (the reference is the case identifier) or `collection_hold`. `ReleaseHold(ws, id, actor)` releases it. Holds never change status, outstanding or due dates; they are shown beside them.

## Aging

`Aging(ws, Filter{Customer, Entity, Currency})` evaluates every payable, unsettled document matching the filter at one now. It returns per currency, for each bucket (`not_due`, `due_today`, `1_30`, `31_60`, `61_90`, `91_plus`, `unknown`), the count and the sum of outstanding amounts, plus the evaluation time. Bucket bounds: 1 to 30 days, 31 to 60, 61 to 90, and 91 or more. Settled and internal documents are excluded. Unknown is never added to any overdue bucket. The sum over buckets equals the sum of outstanding amounts of the included documents. This increment scans the workspace (bounded by `Limits.Receivables`); an indexed projection is later (PT-08 is partial).

## Limits

`Limits{Workspaces, Terms, Bindings, Receivables, Allocations, Extensions, Holds, NetDays, Freshness}`, each positive with a cap:

- `Bindings` counts per contract;
- `Allocations`, `Extensions` and `Holds` count per document; reversals count as allocations;
- `NetDays` caps at 3650.

# Provisional choices awaiting the owner

- Calendar days only. Business-day terms can be recorded but are never approved.
- The maximum net days is a configured limit.
- Freshness is one duration per registry, judged per entity and currency.
- A settled document is shown settled even when confirmation is stale.
- Extensions only lengthen. An approved extension cannot be cancelled in this increment.
- Replacement due basis `inherit` or `reissue` is chosen per document by the caller.
- Overpayment is refused here and belongs to the cash ledger.
- The issue date is taken in the zone of the covering binding.

# Verification

- Literal tests:
  - PT-01 dates: day 0, Net 30 and 45, midnight either side in the contract zone, month ends, 2028-02-29, and DST changes in America/New_York and Europe/Berlin;
  - PT-02 snapshot immutability after a new binding, and extensions;
  - PT-03 partial, credit, cancellation, reversal and overpayment;
  - PT-04 stale and missing confirmation, and late receipts;
  - PT-06 bucket bounds 0/1/30/31/60/61/90/91;
  - PT-07 internal, holds and replacements.
- A property test against a `math/big.Rat` model of outstanding and aging over random allocations, reversals, extensions and clock moves.
- Race detector, count=10, a 386 build, gofmt, vet, the whole repository test run, and a mutation run.
- Source scan: no floating point, no comments, imports limited to the standard library and the primitives.
- Not covered: PT-05, notifications, persistence, screens, ERP synchronization and the indexed projection.

# Design review resolution

The first design review returned FAIL with findings 1 to 11 and three hypotheses. Where this section and the text above disagree, this section governs.

1. **One zone per contract.** `Bind` rejects (`ErrRejected`, reason `zone_mismatch`) a zone that differs from the zone of any non-withdrawn binding of the same contract. The issue date is therefore computed in a single zone and the gap and false-conflict cases cannot arise. A zone change between bindings is a later contract with its own boundary rule. A test binds different zones and is refused, and it issues on both sides of midnight in the contract zone.
2. **Zone source.** `time.LoadLocation` prefers the host's zone files, and the embedded `time/tzdata` is only a fallback. The registry therefore takes an injected zone resolver `func(name string) (*time.Location, error)`, which is required. Pinning the zone data version is a stated deployment obligation (UNVERIFIED here). The tests use `time.LoadLocation` with `time/tzdata` imported in a test file only. The production package does not import it. Zone names must not be empty, `Local`, or longer than 64 bytes. They are resolved at `Bind`, and an error is `ErrInvalid`.
3. **Reversals count against no room.** Each allocation is reversible exactly once, so the per-document count stays at most twice `Limits.Allocations`.
4. **A reversal follows the source of its allocation.**
   - Only an effective allocation (approved, not reversed) can be reversed. A pending manual allocation is withdrawn with `WithdrawAllocation`.
   - An authority reversal carries ledger and revision and is effective at recording.
   - A manual reversal needs evidence and becomes effective only on `ApproveReversal` by another actor than its recorder. While it is pending, the allocation still counts.
   - `Reversal.Effective` must not be before the allocation's `Effective`.
   - Only one non-withdrawn reversal per allocation.
5. **Replacement.**
   - A document may be replaced at most once, by one live replacement document.
   - The replaced document must be payable.
   - Status shows `ReplacedBy` beside the normal state.
   - Aging puts a replaced document that still has an outstanding amount in a separate `replaced` bucket, never in the overdue or not-due buckets. The double count is visible, not hidden, until the cancellation allocation is recorded. A test covers that window.
6. **Dates.**
   - `IssuedAt` must be non-zero and not after now (`ErrRejected`), and its date in the contract zone must lie in the date range (`ErrInvalid`).
   - `Allocation.Effective` and `Reversal.Effective` must not be after today in the snapshot zone (`ErrRejected`, `future_effective`).
   - An effective allocation is an approved, unreversed one. Outstanding = amount − Σ effective allocations.
7. **Refusal precedence at issue.** With A the approved and P the pending bindings covering the issue date:
   - A + P ≥ 2 with A ≥ 1 gives `term_conflict`;
   - A = 0 and P ≥ 1 gives `term_pending`;
   - A = P = 0 gives `term_missing`.

   `Bind` does not check overlap; approval does. Each combination is tested.
8. **Extension history.** An extension is effective at approval (provisional). Each entry records:
   - the requested due date and the reason;
   - the recorder and the approver;
   - the approval time;
   - the previous effective due date;
   - the days overdue at approval (today − previous due, at least 0).

   A PT-02 test extends a document that is already overdue and checks that the history keeps the overdue days.
9. **Staleness applies only after the effective due date.** Before or on the due date, the status is `not_due` or `due_today` with `Stale` true when confirmation is missing or stale. After it, missing or stale confirmation gives `unknown`. This is added to the provisional list.
10. **Idempotency and scopes.**
    - An identical repeat by the same actor returns the stored record. Identical content by another actor is `ErrConflict`, because the recorder is part of the record.
    - Repeating an approval, withdrawal or release by the same actor returns the stored record. The same transition by another actor after it happened is `ErrRejected`.
    - `ApproveEnd` re-checks that no issued document under the binding has an issue date after the new end. To must be on or after From.
    - `Terms` and `Receivables` count per workspace, `Bindings` per contract, and `Allocations`, `Extensions` and `Holds` per document.
11. **Known gaps against the contract.**
    - The term has no display name (no free text).
    - The basis is implicitly the invoice issue date.
    - The holiday calendar revision is not applicable (calendar mode only).
    - The binding reason is a code.

    `Aging` adds a `Contract` filter, and empty filter fields mean any. `Status.Days` is signed: today − effective due date, negative before the due date.

Hypotheses:

- A refused allocation (`exceeds_outstanding`, `future_effective`) records nothing and can be retried, for example after a reversal arrives.
- The ledger revision is recorded and not used for ordering (provisional).
- A confirmation older than a document's issue time still counts by its age alone. A test shows this.

## Re-review resolution

The re-review returned PASS with three low findings, resolved as follows:

- A. Withdrawn manual allocations count against `Limits.Allocations`. All reversals of a document, withdrawn included, are capped at `Limits.Allocations`. A bounds test covers both.
- B. Aging reports, per bucket, the count and the sum of stale entries next to the totals (`Bucket.StaleCount`, `Bucket.StaleAmount`).
- C. A resolver that returns a nil location or an error makes `Bind` fail with `ErrInvalid`. The zone is resolved once at `Bind`, and the resolved location is kept with the binding and every snapshot.

# Change review resolution

The independent change review returned FAIL on one medium finding. It confirmed the core arithmetic for amounts, due dates and status against the plan.

1. **A reversal now needs the same source as its allocation** (`ErrRejected` otherwise). A manual allocation can be undone only by a manual reversal approved by a second actor. An allocation from the authority can be undone only by an authority reversal. Literal tests cover both mismatches, and the property test tries mismatched reversals.
2. **The bucket list is unexported.** `BucketNames()` returns a copy, and a test shows that changing it has no effect.
3. **Repeats and what digests cover.**
   - A repeated `Bind` is compared against the binding as first recorded, so an identical retry after an approved end returns the stored record.
   - Digests cover a record as recorded: the binding as bound and the document with its issuance snapshot.
   - Later ends and extensions are kept in their own histories (`Ends`, `Extensions`) and do not change those digests.
4. **The issue-date range check now runs before the future-time rule.**
   - A payable document is checked in the contract zone.
   - An internal document, which has no zone, is checked as a UTC date.
   - Literal tests cover both.
5. **Outstanding amounts and totals print the exact value with the fewest digits** (`60`, `1.5`), the same canonical form the decimal primitive uses for stored inputs. They are not padded to the profile scale.
   - Sums of terminating decimals always terminate, so formatting cannot fail.
   - A workspace total may exceed the profile's result digits, and it is still printed exactly.
6. **`EndBinding` may shorten any approved binding, not only an open-ended one.** The new end must still be before the current end, on or after the start, and on or after every issued document's issue date.

The test gaps listed by the review are closed:

- a future-dated reversal;
- source mismatches;
- `Estimate` refusing with `term_pending` and `term_conflict`;
- repeat and cross-actor approvals and withdrawals of every kind;
- withdrawn manual allocations counting against the room;
- status day counts across the Berlin daylight-saving changes;
- the property test run in Seoul, New York and Berlin;
- views that share no data with the registry.

New form tests check every argument of every call and the validator bounds.

Hypothesis H1 is recorded: `Aging` scans the workspace under the lock, which is a known limit of this increment (PT-08 is partial). These fixes were not reviewed again.
