---
type: ImplementationPlan
title: Contract lifecycle events, settlement state, period segments and proration
description: Defines the first model of work item W-15, an append-only dated lifecycle per contract with knowledge points, a settlement state stored independently of the contract state with guarded transitions, the split of a billing period by lifecycle events and the proration of a fixed periodic amount with exact totals.
status: draft
---

# Scope and ownership

[HB-178](https://linear.app/hybrid-billing/issue/HB-178/w-15-contract-lifecycle-events-independent-settlement-state-period) is the first increment of work item W-15 (requirement HB-15, with the lifecycle part of QA-02). It is the eighth increment built under the owner's instruction of 2026-11-12. Base: main `eccf3b6`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the model decides billing amounts and is keyed by tenant. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-13:

- Requirement HB-15: start, suspension, renewal, price change, termination and final settlement are distinguished, and so are usage period and billing period. Ending a contract does not cancel a provider subscription or delete a resource. Proposed direction: `ContractRevision`, `ServiceSubscription`, `BillingSchedule` and `ContractEvent` carry effective and recorded times; a period `[start, end)` is split by events; proration, included quantity, minimum and advance or arrears billing are fixed. Scale: interval operations run sorted per contract; a mid-month change references interval revisions instead of copying data.
- Acceptance of HB-15: 300,000 for 30 days of which 10 days is 100,000; an arrears addition of 50,000 gives 150,000; and a correction of an existing advance bill. Linked source cases, all NOT_RUN: REV01 (no missed or doubled billing on mid-month change, end, late usage and cycle change), GH05, GH06.
- Open decisions of HB-15, not made: the proration basis, what a period change or a termination leaves over, and the final grace period. Rounding is an open decision of HB-16.
- QA-02: contract state (active, suspended, ended) and settlement state (pending final, settled) are stored independently; the correction path for usage before termination and late credits stays open after the end; the final evidence is secured before source access is withdrawn, and when that is impossible the reason, the owner and the follow-up path are recorded; an emergency security withdrawal is not delayed for evidence and is recorded as incomplete. AUD-QA-02 (NOT_RUN): an end is not shown as settled, and a failed last collection, an immediate security withdrawal and a source received after the end all leave a recorded gap and responsibility.
- Existing code: `modules/organizations` (HB-168) holds `ContractRevision` as append-only numbered revisions without time, state or approval, and its plan left lifecycle and periods to W-15. `packages/primitives/decimal` gives exact decimal arithmetic under a profile with explicit rounding modes; its `Quo` refuses an inexact quotient, `DivideRound` rounds a quotient once, and every operand must fit the profile's result bounds. `modules/relationships` (HB-170) has its own `Day` and `Interval` with unexported fields. Everything is in memory; the business database is not selected.
- The repository structure names a `contracts` module for contract revisions, subscriptions, billing profiles, proration and cycles.
- The planning sources (`requirements.md`, `detailed-requirements.md`, `work-breakdown.md`) are untracked local files of the primary checkout and are not in the ticket worktree; they were read there.

# Decision

One Go package, `modules/contracts`, standard library and the decimal primitive only, with an in-memory registry and pure calculation functions. It records lifecycle and settlement facts and computes proration; it authorizes nothing, issues nothing, calls no provider and has no entry point wired in this increment. Ending a contract here has no outward effect by construction.

Not moved: `ContractRevision` stays in `modules/organizations`. This package refers to a revision by its number as an opaque reference and does not check that the contract, the revision or the workspace exists there; binding the two is part of the follow-up that introduces persistence or the commands (stated limit). The package has its own copy of the identifier rule, the third in the repository (stated; a shared primitive is deferred).

## Days and intervals

A `Day` is a civil date (year 1 to 9999) without a time or a zone; which zone a contract's days are read in is the caller's decision (the contract time zone is on screen O03 and not decided here). An `Interval` is `[start, end)` over days, bounded, with `start < end`. The number of days of an interval is the count of civil dates in it. The package defines its own `Day`, because the one in `modules/relationships` cannot be constructed arithmetically from outside and importing that package for a date would couple two unrelated modules; moving a shared date type under `packages/primitives` is deferred.

## One sequence per contract

Per (workspace, contract) the registry holds one append-only log and one sequence number, starting at 0 for a contract without records. Every successful write of any kind, a lifecycle batch or a settlement entry, takes the caller's expected sequence, conflicts on a mismatch with nothing written, and advances the sequence by exactly one. A sequence value is a knowledge point: what was known after that write. Lifecycle and settlement are checked and written under one lock, so a settlement step can never be applied to a lifecycle its caller did not see.

Every write operation (`Amend`, `RecordEvidence`, `Settle`, `Reopen`) takes the expected sequence and a caller-assigned write identifier. Write identifiers are one space per contract across all kinds of write. Repeating a write:

- the content of a write is its kind and every member the caller passed, with withdrawals and events in the order given; the expected sequence is not part of it;
- a write whose identifier is stored and whose content is identical writes nothing and returns the stored outcome: the sequence that write produced, marked as a replay. It reports what that write did, not the contract's present state, which the caller reads separately; a replayed end batch answers "written as sequence 4" even if the contract was un-ended at sequence 5;
- a stored identifier with other content, or of another kind, is a conflict;
- this is evaluated before the expected sequence, so a caller that lost a response can repeat the write and learn its outcome;
- a refused write stores nothing and does not consume its identifier.

Every record stores the registry's time, read once per write before anything is changed, so that a failing clock leaves nothing half written. Nothing is ever deleted or rewritten.

## Lifecycle events

| Kind | Fields | Meaning |
| --- | --- | --- |
| `start` | `From` day, revision reference | Active from that day under that revision |
| `change` | `From` day, revision reference | From that day another revision applies (price or terms change, renewal with new terms) |
| `renew` | `From` day | A renewal without new terms took effect that day. A recorded fact only: it changes neither state nor revision and cuts no segment |
| `suspend` | `From` day | Suspended from that day |
| `resume` | `From` day | Active again from that day |
| `end` | `From` day | Ended from that day. `From` is the first day that is not served; the last served day is the day before. An operator's "termination date" meaning the last served day must be converted by the caller, and the field is named so that the exclusive meaning is in the name |

Every event has a caller-assigned identifier under the identifier rule, unique within the contract and never reused, also after its withdrawal.

`Amend(workspace, contract, expected, write, withdrawals, events, cancelFinal)` writes one batch:

- the batch is applied as a whole or not at all; a batch without withdrawals and events is refused;
- a withdrawal names an event that holds before the batch. Naming an unknown identifier, an event already withdrawn, an event added in the same batch, or the same event twice refuses the batch. A withdrawal has no identifier of its own and cannot be withdrawn; a withdrawn event is restored by adding a new event;
- after the batch, the events that hold, ordered by `From`, must form a valid lifecycle, else the batch is refused with nothing written.

A valid lifecycle is either empty or: at most one event per day; the first event is `start` and there is exactly one; `suspend` only while active; `resume` only while suspended; `change` while active or suspended, with a revision reference different from the one in force; `renew` while active or suspended; `end` while active or suspended; nothing after `end`.

An empty lifecycle that holds after at least one batch is a voided contract: every event was withdrawn, which is how "cancelled before it started" is expressed, since an end cannot precede or share the day of the start. Its state is `not_started` on every day.

Provisional rules, fail-closed, each a known limit:

- one event per day. The reason is that the order of two events on one day is not defined by the owner; the rule also refuses combinations whose order would not matter, such as a resume and a change on a renewal day, and the caller cannot express them without moving one by a day;
- a contract without `end` continues: its state is `active` on every later day. Term expiry is not modelled, `active` does not mean "within an agreed term", and a consumer must not read it so. A fixed term that lapses without a renewal fact would bill less than this rule; which of the two holds is the owner's decision;
- a re-start after an end is a new contract identifier.

Backdating is allowed and is the correction path QA-02 asks for: an event may take effect before events recorded earlier, and a wrong event is corrected by withdrawing it and adding the right one in the same batch.

`Lifecycle(workspace, contract, at)` returns the events that held at knowledge point `at`; for a contract that has at least one write, `at` 0 is the empty lifecycle, while a contract without any write is not found at any point, 0 included, so that a read cannot tell an unused identifier from a contract of another workspace; `at` beyond the current sequence is refused, so the answer for a given `at` never changes. `Records` returns the raw log. `StateOn(workspace, contract, at, day)` is `not_started`, `active`, `suspended` or `ended`, with the revision reference in force (none before the start).

## Settlement: stored independently, transitions guarded

The settlement of a contract is its own stored state with its own entries in the log. It is never computed from the contract state when read; the transitions between its values are guarded by the lifecycle.

| State | Entered |
| --- | --- |
| none | Initially; and when a batch during `pending_final` makes the lifecycle neither ended nor voided |
| `pending_final` | In the same atomic step as the batch after which the lifecycle that holds is ended or voided; and by `Reopen` from `settled` |
| `settled` | By `Settle` from `pending_final` |

Invariant, checked on every write: the settlement state is none exactly when the lifecycle that holds is neither ended nor voided. An ended contract is therefore never without a settlement state, and never settled by being ended (AUD-QA-02).

Rules:

- a batch during `pending_final` that leaves the lifecycle ended or voided (for example moving the end day) is allowed; it invalidates secured evidence as described below. A batch that leaves it neither ended nor voided (the end was a mistake) returns the settlement to none with an entry that keeps a reason from a closed list (`recorded_in_error`, `contract_continues`), passed as `cancelFinal`;
- `cancelFinal` must be set exactly when the state before the batch is `pending_final` and the lifecycle after it is neither ended nor voided. Set in any other case (state none, the lifecycle stays ended or voided, the batch itself ends the contract) or missing in that case, the batch is refused with nothing written, so that no false cancellation is stored;
- any batch is refused while `settled`: a lifecycle correction after settlement requires `Reopen` first, so that a settled contract's basis cannot move silently;
- `RecordEvidence` appends one evidence record, on a contract that has at least one write (an unknown contract is not found, so a mistyped identifier creates nothing): secured, allowed only while `pending_final`, with an opaque reference to the stored evidence, the day it covers the source through (`CoveredUntil`, exclusive: every day before it is covered), and the write identifiers of the missing records it resolves, each of which must exist and be unresolved; or missing, with a reason from a closed list (`source_unavailable`, `access_withdrawn_emergency`, `provider_refused`, `not_attempted`, `late_source_received`), an owner identifier and a follow-up reference, allowed in every settlement state including none, so that an emergency withdrawal on a running contract and a source that arrives late leave a record wherever the contract stands;
- `Settle(asOf)` for an ended lifecycle is allowed only when all of these hold: the state is `pending_final`; the last evidence record is secured; that record was written after the most recent entry into `pending_final` and after the most recent lifecycle batch, so that evidence gathered for an earlier basis or before a reopening never settles; its `CoveredUntil` is not before the end's `From`, so that evidence secured while the contract was still served does not settle the days after it; every missing record of the contract, whenever written, is resolved by a secured record that names it, so that a later secured record cannot bury an open gap; and the end's `From` is not after `asOf`, the caller's current day, so that a future-dated end cannot be settled while the contract is still served;
- `Settle` for a voided contract needs the state `pending_final` and no unresolved missing record, and no secured evidence, because a contract that never started has no source to secure (provisional; without this rule a voided contract could never leave `pending_final` except with an invented reference);
- a missing record written while `settled` does not change the state by itself; `Reopen(reason)` with a reason from a closed list (`late_source`, `late_credit`, `correction`) moves `settled` to `pending_final`, and after it the earlier evidence is history only;
- there is no time rule: the final grace period is not decided and is not modelled.

Provisional and conservative, the owner's to decide: `Settle` is refused while any gap is unresolved, also one recorded long before the end, so a gap that can never be closed keeps the contract `pending_final` for good; whether a settlement may close with an accepted gap is not decided. `asOf` is supplied by the caller and not compared with any clock.

Reference, owner and follow-up are identifiers under the identifier rule and reasons are closed codes, so the record has no free-text member. The identifier rule still admits strings that resemble names; that no personal or customer text is put into an identifier is an obligation of the caller that the package cannot check, and it does not check that a reference or an owner belongs to the workspace (stated limits).

`Settlement(workspace, contract, at)` returns the state, the evidence records and the entries as of a knowledge point.

## Segments of a billing period

`Segments(events, period)` is a pure function over the events that hold at some knowledge point. The state and revision at `period.start` follow from all events with `From` on or before that day. The period is cut at every `start`, `change`, `suspend`, `resume` and `end` whose `From` is strictly inside the period; an event on `period.start` sets the first segment's state without a cut, and an event on `period.end` is outside. Days before the start are `not_started`, days from the end's `From` on are `ended`. The segments cover the period exactly, in order, without overlap, and adjacent segments differ in state or revision. The registry offers `SegmentsAt(workspace, contract, at, period)`.

This is the usage-period and billing-period distinction of the requirement at the level this increment reaches: the caller passes the billing period, and the lifecycle decides which of its days are served.

## Proration of a fixed periodic amount

`Prorate(profile, basis, amounts, segments)` computes the lines of one fixed periodic charge for one billing period.

`basis` is everything that must be equal for two results to be comparable, all required, none defaulted: workspace, contract and a charge identifier (opaque to the calculation, carried into the result); the billing period; the knowledge point the segments were read at; the day basis, with the single value `actual_days`; whether suspended days are billed (`bill_suspended` or `skip_suspended`); the result scale; the rounding mode. `amounts` gives, per revision reference, the price for exactly this billing period; a monthly price passed with a quarterly period is the caller's error and cannot be detected here. An amount for a revision that no billable segment uses is ignored, and a segment that is not billable needs no amount. The knowledge point may be any value including 0 and the scale may be 0; the members whose absence is detected are the identifiers, the period, the day basis, the suspended-day rule and the rounding mode, which have no valid zero value.

Refused, never repaired: segments that are empty, unordered, overlapping, with a gap, or that do not cover the period exactly, so that a caller cannot pass the billable part or one chunk and silently change the denominator; a billable segment whose revision has no amount; an amount with more decimals than the result scale, because a full period would then bill a rounded and not the agreed amount; billed amounts of both signs in one call (a positive and a negative amount among the revisions that are billed), because the bound on a line stated below holds only for one sign; a scale outside the profile; a missing or unknown member of the basis.

Calculation:

- billable days are those of `active` segments and, under `bill_suspended`, of `suspended` segments; `not_started` and `ended` days are never billed;
- with `D` the days of the period, the exact share of billable segment `i` is `amount(revision) × days(i) / D`. The exact running sum is kept as a numerator, `N(i) = Σ amount × days` over billable segments 1 to `i`, which the decimal primitive computes exactly, and rounded once per step: `R(i) = DivideRound(N(i), D, scale, mode)`, `R(0) = 0`. The line of segment `i` is `R(i) − R(i−1)`;
- the result has one line per billable segment (interval, revision reference, days, amount), per billed revision reference its input amount and the total of its lines, the total, the days of the period, the basis, and the profile's identifier and revision. A period without a billable day gives no lines and a total of zero. A billable segment under a zero amount gives a line whose amount is zero.

Properties that follow and are tested:

- the lines sum exactly to `R(n)`, the rounded exact total;
- every rounding mode of the primitive is monotone, so a line never has the sign opposite to its exact share, and with amounts of one sign it differs from its exact share by at most one unit of the result scale (exactly one is reached: amount 3 at scale 0 over 6 days in segments of 1, 2 and 3 days with `half_even` gives 0, 2, 1 for shares 0.5, 1, 1.5); a line can be zero for a non-zero share (amount 1 at scale 0 over three 10-day segments with `half_up`: 0, 1, 0);
- cutting any segment further at arbitrary days, within one call, never changes the total. This is the only chunk-independence claimed: the calculation is one call per period with the full segment list, never one call per chunk;
- a full billable period under one revision gives exactly the amount.

Stated consequences of the rounding rule, which is a remainder-allocation policy and provisional because rounding is an open decision of HB-16:

- a line depends on the segments before it: a backdated change to an early segment can move an unchanged later segment's line by one unit, and with several revisions in one period less than one unit can move between revisions, so the totals per revision are sums of lines and not independently rounded shares;
- `floor` and `ceiling` are not symmetric in sign: 100,000 over 31 days with 10 days served is 32,258 under `floor`, and the mirrored credit is −32,259; a charge and its reversing credit can differ by one unit under those modes. `half_up` in the primitive rounds halves away from zero;
- usable range: `|amount| × D` must fit the profile's result bounds, because the numerator is an operand of the primitive. An amount within about one and a half digits of the bound for a monthly period, or two and a half for a yearly one, is refused although its result would fit. The primitive can also refuse a numerator whose digits plus the scale exceed the profile's intermediate bound. Either refusal is the primitive's error, returned as it is.

`actual_days` is the only basis implemented and is provisional, not conservative: ten days of 300,000 are 107,143 in a 28-day period, 96,774 in a 31-day period and 100,000 in a 30-day period, and a 30-day convention would give 100,000 in all three. The acceptance example does not distinguish the two. The convention is not implemented because for a 31-day or 28-day period it is ambiguous whether a full period bills the amount or 31/30 of it; a basis without an explicit value is refused.

Negative amounts are accepted (a periodic credit). Every arithmetic error of the decimal primitive is returned, never replaced by a value.

## Correction against an earlier result

`Correction(profile, before, after)` compares two proration results and returns the difference of the totals, the difference per revision reference (a revision present in only one result counts as zero in the other), and both results. A result is a plain value that the caller stores and passes back; the function cannot know that a result came from `Prorate` and trusts its lines and amounts. It checks what it can: each result must list a revision at most once, with a price, and its totals per revision must sum to its total. Beyond that a result built by hand gives a difference of that hand's numbers, which is the caller's responsibility (stated limit). It is refused unless the two have the same workspace, contract, charge, period, day basis, suspended-day rule, scale and rounding mode, the profile identifier and revision of the profile passed, and the same input amount for every revision reference billed in both, all of which a result carries; the knowledge points may differ and are carried in the answer.

`before` must be the result the last issued document was built from, which the caller stores with its knowledge point; this package does not store results and has no lookup by time. With `before` from the lifecycle as known when an advance bill was issued and `after` from the lifecycle as known now, this is the correction of an existing advance bill: for 300,000 billed in advance for a 30-day period and an end later recorded with `From` on day 11, `after` is 100,000 and the correction is −200,000. Because every total is rounded on its own, corrections across successive knowledge points add up exactly to the correction across the whole span.

The per-revision differences are there because the net hides them: before, revision 1 for the full period, 300,000; after, revision 1 for 10 days (100,000) and revision 2 at 360,000 for 20 days (240,000); the net is +40,000, made of −200,000 and +240,000, and the two revisions can differ in transaction path or eligible components.

The package computes differences. Issuing a correcting document, and deciding whether a termination refunds the remainder at all, are not part of it: the remainder after a termination is the owner's open decision, so the function states the arithmetic consequence and nothing more.

The acceptance's middle case (an arrears addition of 50,000 giving 150,000) is a sum of a prorated line and another charge. It is checked in the tests with the decimal primitive's addition; combining charges into a statement belongs to rate calculation (W-16) and is not a function of this package.

## Tenant boundary and limits

Every registry operation takes the workspace identifier and addresses only that workspace's contracts; the same contract identifier in two workspaces is two contracts, and a contract of another workspace is not found. The pure functions hold no state; the basis carries workspace and contract so that `Correction` refuses to subtract results of two tenants or two contracts. The registry does not authorize: callers reach it only through commands behind the gate, which do not exist yet for this module, so the package must not be exposed directly (stated limit).

Required and positive at construction, without defaults: contracts per workspace; writes per contract; a reserve of further writes per contract; and withdrawals plus events per batch. The count is of writes, that is of sequence values: a batch is one write whatever it contains, and the entry into `pending_final` that a batch causes is part of that write. At the write limit ordinary writes are refused and nothing is evicted, but the reserve stays usable for the writes that stop billing or keep a gap visible: a batch without withdrawals whose events are only `end` or `suspend`, a missing-evidence record, and `Reopen`. A batch with a withdrawal is never admitted to the reserve, because withdrawing a suspension or moving an end later would extend billing. Not admitted either, and so refused at the limit: a cancellation of a mistaken end (the contract stays ended, which bills less), secured evidence and `Settle` (the contract stays `pending_final`). When the reserve is used up, a running contract can no longer be ended in this registry, which is the one refusal here that fails toward billing; a contract near its limit is an operational alarm condition that this package reports through the refusal and does not otherwise signal (stated limit, acceptable only while the store is in memory).

## What is not decided here

The proration basis; billing of suspended days; the rounding and remainder-allocation rule; whether a termination refunds an advance-billed remainder; the final grace period; settlement with an accepted evidence gap; continuation without an end versus a fixed term; re-subscription under the same contract identifier; several events on one day; the contract time zone; who may record events and settle. Each is either a required parameter, a labelled provisional rule above, or absent.

# Allowed files

`modules/contracts/*` (new); `docs/features/HB-178/*`.

# Verification

- Literal table tests with expected values written by hand: the three acceptance cases; the `half_even` case where a line is off by exactly one unit; billed amounts of both signs refused; an unused amount ignored; a price change in mid-period under two revisions with the per-revision totals; suspension billed and skipped; a period before the start, after the end and spanning both; an event exactly on the period start and on the period end; 28, 29, 30 and 31-day periods, a period over 29 February and one over a year boundary; amounts that do not divide (100,000 over 31 days in three segments) with every rounding mode, each line and the exact total; the zero line of a non-zero share; negative amounts and the `floor` and `ceiling` asymmetry; an amount at the profile's bound, refused; an amount with more decimals than the scale; a missing amount for a billable revision; segments that do not cover the period (billable part only, one chunk, a gap, an overlap, wrong order); every missing or invalid member of the basis.
- Correction: the advance-bill case; the two-revision netting case; every mismatch refused; corrections over three knowledge points adding up to the correction over the span.
- Lifecycle: `renew` leaving state, revision and segments unchanged and refused on a day that has another event; every invalid order refused with nothing written; each withdrawal error; a voided contract; backdated events; withdrawal with replacement; the lifecycle and the state at each earlier knowledge point unchanged by later writes; `at` 0 and beyond the current sequence; the expected-sequence conflict; a repeated write identifier with identical content (returning the stored sequence, also after the state has moved on), with different content and with another kind of write; a refused write not consuming its identifier; the same contract identifier in two workspaces; a batch that would pass a limit writing nothing; the reserve usable only by the terminating and gap-keeping writes, refused for a terminating batch that carries a withdrawal, and its exhaustion.
- Settlement: the invariant after every kind of write; an end opening `pending_final` in the same step; `Settle` refused with missing evidence, with evidence older than the last batch, with evidence older than a reopening, with evidence covering less than the served days, with a gap recorded before the end and not named by the secured record, and for a future-dated end; secured evidence refused in none and in `settled`, and naming an unknown or resolved gap; `Settle` and `Reopen` refused from every wrong state; evidence on an unknown contract; the mistaken end cancelled with the reason, and `cancelFinal` refused in each case where it is not due or is missing; a voided contract settled without evidence, and a voided contract given a new start; reopening, cancelling the end, and a new end needing new evidence; any batch refused while settled; AUD-QA-02's three situations (failed last collection, emergency withdrawal on a running contract, source after the end) each leaving a record with owner and follow-up.
- Properties with a fixed seed against a reference model written in the test: a day-by-day simulation that knows nothing of segments, with day counts from the standard library's time package and expected totals from `math/big.Rat` rounded by a rounding function written in the test, so that an error in the package's day arithmetic or in its use of the decimal primitive is not mirrored. Checked: segments cover the period exactly and agree with the per-day state; the lines sum exactly to the model's rounded total; each line has the sign of its exact share or is zero and differs from it by at most one unit; refining segments never changes the total; a full active period under one revision bills exactly the amount; the correction equals the difference of two independent model totals; random batch histories keep the settlement invariant and never change an earlier knowledge point's answers.
- Concurrency under the race detector: concurrent batches with the same expected sequence, of which exactly one is written; a batch and a `Settle` with the same expected sequence, of which exactly one is written.
- Mutation check by a script with per-row results appended to files that are never overwritten.
- `gofmt -l`, `go vet ./...`, `go test -race -count=1 ./...`.

Remaining NOT_RUN at system level after this ticket: REV01, GH05, GH06, AUD-AR-01, AUD-AR-10, AUD-QA-02, PT-01 to PT-08, ERPC-01 to ERPC-08. This ticket does not complete W-15.

# Deferred

AR-01 (transaction path and eligible components in selling calculation and approval) and AR-10 (flat rack, reserved base and usage, SLA credits), both mandatory scope of W-15 and not reached by this increment; payment terms and aging (PT); ERP recognition cadence (ERPC); included quantities, minimums and tiers (W-16), for which the result exposes the days of the period and of each line so that they can be prorated on the same basis; service subscriptions; cycle changes of a billing schedule; commands, permissions and screens O03, S01, S05; persistence and the binding to `modules/organizations`.

# Design review resolution

The independent design review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker, 4 HIGH, 8 MEDIUM and 4 LOW, and judged the settlement section as written to fail AUD-QA-02. All sixteen were adopted in the sections above.

| Finding | Resolution |
| --- | --- |
| D1 proration without a period | The period is in the basis; segments must cover it exactly; chunk-independence restated as refinement within one call |
| D2 settling again on evidence older than a reopening | `Settle` needs secured evidence written after the most recent entry into `pending_final` |
| D3 settling a lifecycle the settler did not see | One sequence per contract for lifecycle and settlement writes; evidence older than the last batch does not settle |
| D4 gaps lost outside `pending_final` | Ending opens `pending_final` atomically (invariant); missing-evidence records allowed in every state |
| D5 future-dated end settled | `Settle` takes the caller's current day and refuses an end after it |
| D6 no way back from a mistaken end | A batch that removes the end during `pending_final` needs a cancel reason and returns the state to none |
| D7 continuation and renewal silently decided | `renew` added as a recorded fact; continuation labelled provisional and not conservative; listed as undecided |
| D8 undefined cases | Withdrawal errors, the voided contract, `at` 0 and beyond, caller-assigned identifiers and repeatable writes defined |
| D9 exact sum not representable | Numerator form and usable range stated; boundary test added |
| D10 comparability and netting | The basis is part of the result; mismatches refused; differences per revision returned |
| D11 properties of the rounding rule | Stated; excess decimals refused; rule labelled provisional |
| D12 limits blocking the event that stops billing | A reserve for terminating and gap-keeping writes; alarm condition stated |
| D13 one event per day wider than its reason | Named as a limit |
| D14 naming and boundary wording | `From` with the exclusive meaning stated for an end; cut rule strictly inside |
| D15 identifiers and tenant | Closed reasons for every operation; obligations and unchecked references stated; basis carries workspace and contract; settlement entries in the log with time |
| D16 basis rationale | `actual_days` called provisional, with the three numbers; the amount must be the price of the period |

# Design re-review resolution

The narrow re-review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker, 1 HIGH, 6 MEDIUM and 5 LOW; eleven of the sixteen first findings resolved and five partly. All twelve new findings were adopted in the sections above; the text after them was not reviewed a third time.

| Finding | Resolution |
| --- | --- |
| E1 evidence secured while still served | A secured record states the day it covers through, and `Settle` needs it to reach the end |
| E2 a later secured record burying open gaps | A secured record names the gaps it resolves; `Settle` needs none unresolved |
| E3 false bound on a line | Bound restated as at most one unit for amounts of one sign, with the case that reaches it; amounts of both signs refused in one call |
| E4 conditions not checkable from a result | The result carries input amounts per billed revision and the profile's identifier and revision |
| E5 `cancelFinal` when not due | Required exactly in one case, refused in every other |
| E6 replay semantics | Content, outcome, kinds, the shared identifier space and refused writes specified; every write operation takes expected sequence and write identifier |
| E7 reserve ambiguous | Limits count writes; the reserve admits only batches without withdrawals; what is not admitted and the one refusal that fails toward billing stated |
| E8 voided contract never settles | Settled without evidence when no gap is open (provisional) |
| E9 zero values and unused amounts | Stated which members can be detected as missing; unused amounts ignored; zero-amount wording fixed |
| E10 second bound | Stated |
| E11 evidence on an unknown contract | Not found |
| E12 missing verification cases | Added |

# Change review resolution

The independent change review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker, no HIGH, 2 MEDIUM and 7 LOW. It found every rule of the plan implemented, no input sequence that lets the settlement state none coexist with an ended or voided lifecycle or the reverse, no path to settled for an ended contract without qualifying evidence, no change of a past knowledge point's answers, and the reference model's rounding correct for all six modes and independent of the primitive's.

Stated difference from the text above, confirmed by the review: the entry into `pending_final` that a batch causes is not tracked on its own. A batch that enters it is itself the most recent batch, so "after the most recent lifecycle batch" already covers it; only a reopening is noted separately.

| Finding | Resolution |
| --- | --- |
| C1 `Correction` trusts hand-built results | Repeated revisions, missing prices and totals per revision that do not sum to the total are refused; the remaining trust is a stated limit |
| C2 an end from the reserve never written in a test | Tests added: an end from the reserve opening the final settlement in the same write, and exhaustion after a suspension and an end |
| C3 `Day.Add` wrapped for very large moves | Bounded before any conversion by the day number of the last day; cases added |
| C4 the clock was read after the state changed | Read once per write before the first change |
| C5 no test that a refused secured record resolves nothing | Tests added for a rejected and for a limit-refused record |
| C6 the random-write property checked little of a settlement and shared logic | It now checks the whole precondition of `Settle` from the log, and finality through the state on the last representable day |
| C7 property lines bounded but not pinned | Each line is compared with the model's own cumulative rounding |
| C8 an unknown contract at point 0 | Stated above |
| C9 mutation evidence not in the repository | The run and its results are in the verification record; the per-row files stay in the session's scratch directory (stated limit there) |

Not changed: the number of workspaces is not bounded by this package; the registry is not exposed and a bound belongs with the commands (stated).
