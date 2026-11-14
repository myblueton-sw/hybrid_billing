---
type: ImplementationPlan
title: Monthly ERP recognition batches independent of the billing cycle, reconciled against quarterly billing without double counting
description: Defines the ERP recognition cadence increment of work item W-23, a register of immutable monthly recognition batches with approval, purpose-bound ERP submissions and posting results, corrections that never overwrite a month, approved recognition adjustments, a quarterly reconciliation that counts every amount once and separates collecting, unposted, difference, pending adjustment, unposted adjustment and matched states, and a quarter clearance that refuses without a declared double-recognition method.
status: draft
---

# Scope and ownership

[HB-224](https://linear.app/hybrid-billing/issue/HB-224) continues work item W-23 after HB-223 (`modules/protected`). It builds the ERP recognition cadence (`docs/planning/v1.16/erp-recognition-cadence.md`, acceptance ERPC-01 to ERPC-08) in delivery order under the owner's instructions of 2026-11-12 and 2026-11-14. Base: main `e714b02`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan and the change, because the register decides billing-related amounts and their reconciliation. Machine: mac_mini.

Not in this increment: the ERP connectors and provider profiles (W-24, W-25), the common submission ledger of W-24 (this register keeps a recognition-purpose submission record that W-24 will absorb), FX conversion, screens O03, U02, I01, I02, B08, B09, I03 and R01, persistence and the integration layer.

# Observed baseline

- The cadence document requires the following:
  - Usage period, ERP recognition period, customer billing cycle and due date are separate axes. A monthly recognition submission never issues, delivers or starts payment terms of a customer invoice.
  - A monthly `RecognitionBatch` fixes workspace, entity, contract, recognition and attribution periods, source snapshot, line lineage, amount basis, currency, amounts, approval and hash.
  - The `ERPRecognitionSubmission` has purpose `monthly_recognition`, target, batch revision, idempotency key, external document reference, posting state and confirmation time, separate from `customer_invoice` submissions.
  - Flow: attribute → validate → fix batch → authorised approval → submit → look up result and reconcile. Accepted is not posted. A timeout or unknown result is looked up first and never resent under another key. A partial failure resumes only the unprocessed part.
  - Quarterly comparison base = current monthly recognised net + approved linked adjustments not yet included in that net. Net = initial + confirmed amendments − confirmed cancellations or reversals, each once. Provisional and final, different bases, currencies or tax bases are never mixed. A missing or unconfirmed month is never complete even when the difference is zero. Net, tax and gross reconcile separately.
  - Differences are explained only by a `RecognitionAdjustment` with cause, target month, before and after, approver and external state; no balancing rows.
  - States: collecting, monthly posting to confirm, amount difference, adjustment approval pending, adjustment posting to confirm, matched. A change of hash, months or external state requires a new check before approval.
  - A quarterly invoice must not re-recognise cost; without a verified offset, replacement, reversal or reference method the automatic quarterly submission is blocked. Closed periods are corrected through open-period adjustments that link the original month.
- No module models ERP recognition. `modules/approvals` binds approvals to digests but has no recognition stage; this register records its own approvals, like `modules/protected`.
- The newer registers import only `packages/primitives` in non-test sources.

# Decision

A new package `modules/recognition` holds one registry per process, with an injected clock (the highest reading, in UTC, rounded to zero), one mutex, workspace isolation (`ErrNotFound` across workspaces), errors `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (`Refusal{Reason}`) and `ErrLimit`, and the check order form, lookup, repeat or conflict, rule, room. An identical repeat of a write by the same human returns the current view. Every write needs a human (`ai_not_allowed`, `not_human`), except `Record`, which a system connector may call.

Amounts are decimal strings parsed with the registry's decimal profile and held as integer units at the profile scale. Every sum is exact and checked against the profile's result bounds.

## Profiles

`Define(workspace, Profile, actor)` records a recognition profile: identifier, entity, ledger, contract, amount basis (`sales_billing`, `provider_cost` or `internal_allocation`), tax basis (`tax_separate` or `tax_free`), currency, amount scale (0 to 8), whether provisional batches are allowed, and the double-recognition method (`offset`, `replace`, `reverse_then_confirm`, `reference_link` or `none`). A profile is immutable; a different profile needs a new identifier. Amounts never add across profiles. Amount strings are parsed with the registry's decimal profile and must fit the profile's amount scale; more digits are `ErrInvalid`.

`Close(workspace, profile, period, actor)` closes a posting month (`YYYY-MM`). Closing is permanent.

## Batches

`Open(workspace, Batch, actor)` records a batch: identifier, profile, attribution month, posting month, kind, the batch it reverses, provisional flag, source snapshot digest and lines (identifier, source lineage reference, net, tax; signed amounts on the profile scale). The kinds:

- `initial`: posts in its attribution month. A month's initial batch is live while it is not void and not reversed by a posted reversal. A new initial is allowed only when the month has no live initial; a provisional month becomes final by reversing the provisional initial and opening a final one.
- `amendment`: a signed delta to an attribution month whose live initial is posted and has no non-void, non-abandoned reversal (`initial_reversing`). Because amendments need a posted initial, an initial with amendments can never be abandoned.
- `reversal`: names a posted initial or amendment of the same profile and attribution month and has no lines; its amounts are the exact negation of that batch and its provisional flag must equal the target's. A batch is reversed at most once, and a reversal cannot be reversed. An initial can be reversed only when every non-void, non-abandoned amendment of the month has a posted reversal (`has_live_amendments`), so amendments never survive a replaced initial.

Refusals, in order: actor kind; `unknown_profile`; `provisional_not_allowed`; `month_cleared` when the attribution month belongs to a cleared quarter; `period_closed` when the posting month is closed; `posting_mismatch` when an initial posts outside its month or a correction posts before its attribution month; `initial_exists`; `no_initial` for an amendment without a live posted initial or a reversal without a live approved initial; `initial_reversing`; for a reversal `unknown_target`, `target_is_reversal`, `target_not_posted`, `already_reversed` (another non-void, non-abandoned reversal names it), `carries_adjustment` (the target carries a pending or approved adjustment) and `has_live_amendments`; `empty` when an initial or amendment has no lines, or an amendment's net and tax totals are both zero. A tax-only amendment is valid, and an initial may total zero, because an explicit zero month is recognised evidence while a missing month is not. The batch digest (label `recognition_batch_v1`) covers every field, line and total.

`Approve(workspace, batch, digest, actor)` approves the batch digest: `not_pending` unless open, `digest_changed`, `self_approval` for the author, `period_closed` for a closed posting month. `Void(workspace, batch, actor)` ends an open batch; any human may void; `carries_adjustment` when it carries a pending or approved adjustment. `Abandon(workspace, batch, actor)` retires an approved batch that never posted and has no `sent`, `accepted` or `unknown` submission (`outcome_unknown`), does not carry a pending or approved adjustment, and is abandoned by a human other than its author (`self_approval`). Void and abandoned batches count nowhere, and an abandoned initial is not live.

## Submissions and results

`Submit(workspace, batch, key, actor)` creates the batch's `monthly_recognition` submission with an idempotency key, single-use in the workspace. Refusals, in order: `not_approved`; `period_closed`; `already_posted` once a submission of the batch is posted; `outcome_unknown` while an earlier submission of the batch is `sent`, `accepted` or `unknown`. A failed or rejected submission allows a new key.

`Record(workspace, key, Result{State, External, Reason}, actor)` records what the ERP reported, by a human or a system actor. Allowed transitions: `sent` → `accepted`, `posted`, `failed`, `rejected` or `unknown`; `accepted` → `posted`, `failed`, `rejected` or `unknown`; `unknown` → `accepted`, `posted`, `failed` or `rejected`. An assistant is refused first; the same result repeated by a human or system then returns the view; any other report against a final state (`posted`, `failed`, `rejected`) or a disallowed transition is `ErrConflict`. `posted` needs an external document reference, unique in the workspace; failures need a reason. A posted result recorded after its posting month closed is accepted, because the submission predates the closure. Only a posted batch counts as recognised.

## Recognition adjustments

`Propose(workspace, Adjustment, actor)` records an adjustment: identifier, profile, target attribution month, cause (`tier_pricing`, `discount`, `commitment`, `late_source`, `credit`, `rounding` or `other`), the month's amounts before and after (net and tax), and optionally the carrying batch. The delta is after − before and must not be zero in both net and tax. `month_cleared` refuses a cleared target month.

`Decide(workspace, id, Decision{Verdict, Digest}, actor)` approves or rejects it; the proposer gets `self_approval`. `Withdraw(workspace, id, actor)` ends a pending or approved adjustment whose carrying batch is not posted. `Link(workspace, id, batch, actor)` attaches a carrying batch to an adjustment that has none. A carrying batch must be a non-void amendment with no reversal (pending or posted) of the same profile and target month whose net and tax equal the delta, and it carries at most one live adjustment (another adjustment is `ErrConflict`). Links never change and record who linked (the proposer when the carrier came with the proposal); a repeat by that human returns the view. A wrong link is corrected by withdrawing the adjustment and proposing a new one.

A batch carries an adjustment only while that adjustment is pending or approved; after a withdrawal or rejection it is an ordinary batch again. An adjustment is included in the monthly net exactly when its carrying batch is posted; a carrying batch cannot be reversed, voided or abandoned, so an included adjustment stays included.

## Reconciliation

`Reconcile(workspace, profile, Quarter{ID, Months, Currency, Basis, TaxBasis, Billed{Net, Tax}, Final})` computes, without recording anything. Months are 1 to 12 distinct `YYYY-MM` values in increasing order, so quarters need not be calendar quarters. A currency, basis or tax basis different from the profile is `basis_mismatch`. Batches group by attribution month.

- Per month: the posted net and tax (initial, amendments and reversals, posted only, each once), whether a live approved initial exists, and the counts of open, approved-unposted (excluding batches carrying a pending or approved adjustment) and provisional posted batches. A provisional batch neutralised by a posted reversal, and that reversal, are not counted as provisional.
- The adjustment term: approved adjustments targeting a quarter month whose carrying batch is absent or not posted.
- The comparison base (net, tax and gross = net + tax), the billed amounts and the three differences.
- The state, the first that applies:
  1. `collecting`: a month without a live approved initial, or with an open batch;
  2. `posting_unconfirmed`: an approved batch that is not posted and carries no adjustment;
  3. `mixed_basis`: a final quarter over a month with a posted provisional batch, or a provisional quarter over a final initial;
  4. `adjustment_pending`: a pending adjustment targeting a quarter month;
  5. `amount_difference`: any of the three differences is not zero;
  6. `adjustment_posting`: an approved adjustment of a quarter month whose carrying batch is absent or not posted;
  7. `matched`.
- The reconciliation digest (label `recognition_reconciliation_v1`) covers the profile and its method, the quarter, the counted batches with their external references, every adjustment of the months with its state and carrying batch, and the result.

`Clear(workspace, profile, Quarter, digest, actor)` records the clearance for the quarterly ERP submission. Refusals, in order: actor kind; `no_double_recognition_method` when the method is `none`; `month_already_cleared` when a month belongs to a cleared quarter of the profile with another identifier or months; `not_matched`; `digest_changed` unless the digest equals a fresh reconciliation. A repeat of the identical quarter and digest by the same human returns the clearance. The clearance carries the method the quarterly submission must use and issues nothing. After clearance, `Open` and `Propose` refuse the cleared months (`month_cleared`). A matched quarter has no open batch, no approved unposted batch (carrying or not) and no pending adjustment, so nothing in a cleared month is left to submit, and every counted submission is final; the cleared result cannot change. `Submit` therefore needs no cleared-month check.

## Reads and limits

`ProfileOf`, `BatchOf`, `SubmissionOf`, `AdjustmentOf` and `ClearanceOf` return copies. `Limits{Workspaces ≤ 100000, Profiles ≤ 100000, Batches ≤ 1000000, Lines ≤ 10000, Adjustments ≤ 1000000}` apply per workspace, except `Workspaces`, and `Lines` per batch.

# Tests

- ERPC-01: three monthly batches of a quarterly profile are approved, submitted and posted; nothing in the register issues, delivers or starts payment terms.
- ERPC-02: months 100 + 120 + 80 against billed 300 match; billed 290 needs a −10 adjustment that is approved, carried by a posted amendment and then counted once (net 290, term 0).
- ERPC-03: a missing month is `collecting`, an unposted month is `posting_unconfirmed` even with a zero difference; `sent`, `accepted` and `unknown` results block a new key; idempotent keys and results.
- ERPC-04: clearance is refused without a method, without a match, for a changed digest and for overlapping quarters.
- ERPC-05: closed posting months (open and submit), corrections in an open month linked to the original, reversals only of posted batches and once, final results, unique external references, cleared months frozen.
- ERPC-06: `basis_mismatch`, profiles separate entity and basis, provisional and final never mixed.
- ERPC-07: non-calendar quarters, approval separation, immutable batches and profiles.
- Counting edge cases: two adjustments on one batch, reversing or voiding a carrying batch, an adjustment outside the quarter, a tax-only amendment.
- A property test of reconciliation against an independent `big.Rat` model that encodes liveness, link uniqueness and reversal effects itself, over random batches, results and adjustments; race, 386 build, gofmt, vet, coverage and mutation testing. ERPC-08 scale and provider runs are not part of this increment.

# Provisional choices

- The kinds, causes, methods, states, ranges and limits above.
- Gross is net + tax; tax is reconciled as one amount per batch.
- Batch-level submissions; line-level partial resumption belongs to the connector increment.
- Late sources and manual changes in the ERP are reconciled through amendments and recorded results; detecting manual ERP edits belongs to the connector increment.
- `adjustment_posting` covers carrying batches that are approved but not posted, so `posting_unconfirmed` counts only batches that carry no adjustment.
- A reversal negates a whole approved batch; partial reversals are amendments.
- The quarter clearance names the method; whether a provider supports it is verified in W-25.

# Review history

- Design review 1 (independent reviewer, read-only): CONDITIONAL PASS with 6 high, 6 medium and 3 low findings. Folded in: no new key while a result is `sent`, `accepted` or `unknown`; `month_already_cleared` and frozen cleared months; one adjustment per carrying batch with the same checks on `Propose` and `Link`; carrying batches cannot be reversed or voided; reversals only of posted, non-reversal batches; adjustments selected by target month inside the quarter; currency, basis and tax basis on profile and quarter (`basis_mismatch`); live initial defined and provisional-to-final path; open batches keep a month `collecting`; `Submit` checks closed periods; digest covers the method and adjustment links; precedence between carrying and other batches; the `empty` rule; before and after amounts, `Decide` and `Withdraw` for adjustments; grouping by attribution month, scale, result transitions and `Void`.
- Design review 2 (same reviewer): CONDITIONAL PASS; all 15 findings resolved, one high and three medium new findings, folded in: carrying means a pending or approved adjustment, and `Submit` refuses cleared months; neutralised provisional batches and their reversals are not provisional for `mixed_basis`, and a reversal takes its target's flag; an initial is reversed only after its amendments (`has_live_amendments`); `Abandon` retires an approved batch that never posted.
- Design review 3 (same reviewer): CONDITIONAL PASS, to be treated as PASS once two medium findings are folded in with tests, which was done: amendments need a posted live initial without a pending reversal (`initial_reversing`), so an initial with amendments can be neither abandoned nor replaced while amendments survive.

# Change review resolution

Change review 1 (a separate independent reviewer, read-only): FAIL with 1 high, 2 medium and 3 low findings. All were addressed:

1. High: a reversed or reversing amendment could be linked as a carrier, so the reversal cancelled an approved adjustment that had left the term. `carrierFor` now refuses a batch with any non-void reversal (`carrier_mismatch`); `TestReversedAmendmentCannotCarry` covers both orders and both pending and posted reversals.
2. The property model copied production predicates. It now also checks an independent counted-once invariant: the base equals every approved adjustment of the months plus every posted batch that neither carries an approved adjustment nor reverses such a carrier. With the high finding reintroduced, the property test fails on this invariant. The model now generates provisional batches and `mixed_basis`, and biases operations toward carriers, reversals of amendments, submissions and decisions. Clearance, closing and multiple profiles remain covered by example tests.
3. `Link` repeats now require the human who linked; the view records the linker.
4. `Record` refuses an assistant before the repeat check.
5. `Clear` repeats require the identical quarter.
6. The reversal flag check runs after the target is resolved, so `unknown_target` comes first.

Change review 2 (same reviewer): PASS; all six findings resolved and no new defects.
