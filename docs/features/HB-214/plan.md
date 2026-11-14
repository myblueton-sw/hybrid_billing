---
type: ImplementationPlan
title: Billing application ledger with eligibility, reservations, confirmation and reversal independent of request keys
description: Defines the first model of work item W-21, a ledger of billing applications per contract, component and instance, reserved, confirmed, released, replaced and reversed by service interval or by source quantity and amount, judged against approved eligibility revisions with a direct or resale path and a billing route, so that a confirmed scope is never billed again under a new request, client or calculation revision while normal recurring service is kept.
status: draft
---

# Scope and ownership

[HB-214](https://linear.app/hybrid-billing/issue/HB-214/w-21-billing-application-ledger-with-eligibility-reservations) is the first increment of work item W-21 (requirement HB-21 with the required contracts AR-01, DB-03 and QA-09). It is built in delivery order under the owner's instructions of 2026-11-12 and 2026-11-14. Base: main `a525099`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the ledger decides what may be billed. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-14:

- Requirement HB-21 (work item W-21): a receipt key, an API idempotency key and the identity of a billing task are different; changing the client, the request key or the calculation revision must not confirm the same contract, period, component and scope twice. Open: the billing task key, the unit of partial consumption, reservation expiry and what a correction consumes. Direction: BillingClaim and ClaimApplication record the source or contract component, the period, the partial quantity and amount applied and the correction reference, and are reserved, confirmed and released under constraints and explicit locks; the identification of real interval overlap and of recurring flat fees is fixed first. Acceptance: with new keys, another client, concurrent runs, partial billing and late corrections, a confirmed scope is not billed again and normal recurring service is kept. Scale: locks narrowed to a contract, shared pool or usage scope and taken in a fixed order; staging is checked first and confirmed in a short transaction.
- AR-01: a ContractRevision carries the direct or resale path, payment responsibility per item and the scope the MSP may bill; under a direct contract, CSP usage is a basis for reporting and allocation and is separate from the MSP's service fee; sales and approval block rebilling usage the MSP is not responsible for. AUD-AR-01 (NOT_RUN): the same CSP input under a direct contract shows only the service fee, and under resale only the approved eligible components.
- DB-03: before submission, existing ERP sales orders, contracts and prepaid references and direct invoice or order routes are fixed so that one scope is not billed by two routes; purchase documents are reconciled first. AUD-DB-03 (NOT_RUN): a scope already reflected by an existing ERP order or prepayment is not issued again under a new platform submission identifier; duplicate purchase documents and an unknown external ledger hold before submission.
- QA-09: opening balance references and the per-currency balance identity (opening + increase − application − expiry ± explicit reversal = closing). AUD-QA-09 (NOT_RUN).
- The source document, read from `통합_빌링_플랫폼_구현_기획안_v1.16.docx`: B1422 says change requests return the earlier result for the same Idempotency-Key and request hash, and a reused key with another request is a conflict (409). The key's scope is the installation, client and command. Separately, a BillingClaim per contract, billing cycle, source scope and component prevents the same item being issued again under another calculation revision, client or key, and a correction refers to the original. B1476 says a new unissued calculation may replace an earlier candidate, but an issued scope is not billed again under a new revision. B1226 says the same contract, period and component is never confirmed twice, recurring services are identified as separate items, and partial billing and credits are traced as the original line and its balance. B1244 (GH08) says concurrent closes, partial failures, retries and late corrections must not bill the same item twice. B0489 says prepaid balances and credits use the application ledger of chapter 119.
- Existing code: `modules/contracts` keeps contract lifecycles and proration; `modules/marketplace` (HB-212) keeps customer charges without an issuance ledger; nothing records what was billed.

# Decision

One Go package, `modules/claims`, standard library and the decimal primitive. A registry with a clock keeps approved eligibility revisions per contract and the applications of a workspace. Nothing calls it and no entry point is wired.

What this can and cannot show: that an application is accepted only for a component an approved eligibility revision makes billable by the MSP on the platform route for the whole service interval; that no two active applications of one scope overlap in time when the scope is billed by interval; that the active applications of one source never exceed its stated quantity and amount, and that one source is never consumed under two scopes; that a reservation expires, is confirmed or released, or is replaced by a new candidate only while unconfirmed; that a confirmed application frees its coverage only by a confirmed reversal referring to it; that request identifiers and calculation revisions are recorded but are not part of the identity, so a new request for a confirmed scope is refused. It cannot know that a contract revision or an external order is recorded truly, that the amounts are right, or that a document was issued; it does not compute amounts, issue documents, keep balances or talk to an ERP.

Names. A scope is `claimed` when an active application covers it. An application is `active` when it is reserved and unexpired, or confirmed and not wholly reversed.

Not delivered, although HB-21, AR-01, DB-03 or QA-09 name it: opening balance references, prepaid and credit balances and the per-currency balance identity (QA-09, with W-22); the ERP recognition cadence (ERPC-01 to ERPC-08, with W-24); reconciliation of purchase documents (DB-03, with W-24); the HTTP idempotency layer with keys per installation, client and command; shared pools as a scope; locks per contract (one registry lock here); persistence and database constraints; screens and commands. W-21 asks for independent verification by a DBA and by QA; this ticket has one independent reviewer of design and change.

Every string is an identifier under the project's rule (letters, digits, `. _ : -`, at most 128 bytes), a currency code, a date `YYYY-MM-DD` from 2000-01-01 to 2099-12-31 that exists in the Gregorian calendar, the text of a number of at most 64 bytes that the registry's decimal profile reads, a digest of 64 lowercase hexadecimal characters, or a value of a closed list. There is no free text; actor identifiers must be pseudonymous (stated obligation). No floating point is used. Number texts are stored as the decimal primitive's canonical print. Every amount and quantity is zero or above. A service interval is a start date and an end date with start < end, the end not included.

## Eligibility revisions (AR-01, DB-03)

An eligibility revision belongs to a contract and has a number from 1 without a gap, written once by an author and approved by another actor (an approval by the author is rejected; a repeat by the same approver returns the stored state, another approver is a conflict). Fields:

- an effective interval, from and to dates;
- a path, `direct` (the customer contracts with the CSP; CSP usage is reported and allocated, not resold) or `resale` (the MSP resells);
- components, each with an identifier unique in the revision, a kind `csp_usage`, `msp_service` or `other`, whether the MSP may bill it, and a route: `platform` (billed through this platform), `external_order` (already billed by an existing ERP order, contract or prepayment, with its reference identifier) or `unknown` (the external ledger is not known). A component of kind `csp_usage` under the path `direct` that the MSP may bill is invalid (AR-01). A reference is required for `external_order` and absent otherwise.

Approving a revision whose effective interval overlaps that of another approved revision of the same contract is rejected. Unapproved revisions are not used.

## Applications

An application has an identifier unique in the workspace (the request's identity for repeats; any caller key is mapped to it outside this package) and is written once. Fields:

- the scope: contract, component and instance identifiers; the instance tells two recurring services of one component apart;
- the service interval;
- a mode, `interval` or `source`; for `source`, the source identifier and its capacity, a quantity and an amount;
- the quantity and the amount applied, a currency, and the calculation revision it came from (an identifier, recorded only);
- `replaces`, optional, the identifier of a reserved application of the same scope it replaces;
- a reservation lifetime in seconds from 1 to 86,400.

`Reserve(workspace, application, actor)` judges the request, in this order, and either stores the application as `reserved` with its expiry (now + lifetime) or refuses it with the first reason, storing nothing:

1. `no_eligibility`: no approved revision of the contract covers the whole service interval (a service interval across two revisions is refused with this reason);
2. `not_eligible`: the component is absent from that revision or the MSP may not bill it;
3. `routed_externally`: its route is `external_order`;
4. `route_unknown`: its route is `unknown`;
5. `mode_mismatch`: an active application of the same scope has the other mode;
6. `already_claimed`: in `interval` mode, the service interval overlaps that of an active application of the same scope;
7. `capacity_mismatch`: in `source` mode, an application of that source was recorded with another capacity or currency;
8. `source_in_other_scope`: the source has an active application under another scope;
9. `capacity_exceeded`: the quantity or the amount of the active applications of the source with this one would exceed its capacity.

The application replaced by `replaces`, if given, must be `reserved` and unexpired with the same scope, else the request is rejected (an error); its coverage is not counted in the judgement and, when the new application is reserved, it becomes `replaced` in the same step. An identical repeat returns the stored application with its current state; a different request under a stored identifier is a conflict. A refused request may be sent again under the same identifier and is judged again.

`Confirm(workspace, identifiers, actor)` confirms a batch at once, all or none: each must be `reserved` and unexpired, otherwise the whole call is rejected and nothing changes; an identifier already `confirmed` in the batch is accepted as a repeat. `Release(workspace, identifier, actor)` turns a reserved application into `released`; a confirmed one is rejected; a repeat returns it. A reservation past its expiry reads as `expired` and is not active; confirming it is rejected.

## Reversals

`Reverse(workspace, reversal, actor)` records a reversal, written once under its own identifier, referring to a confirmed application, with a quantity and an amount. In `interval` mode only a whole reversal is accepted (quantity and amount equal to the application's); in `source` mode the reversals of an application may not exceed its quantity or amount. A reversal is confirmed when recorded (provisional: no reservation step). An application whose reversals equal it is `reversed` and no longer active; partially reversed, it stays active with its remaining quantity and amount counting against its source. A new application for the freed scope is judged as any other.

## Reads and digests

`Application(workspace, identifier)` returns the application, its state, its reversed quantity and amount, the actors and times of reservation, confirmation, release and replacement, and its digest. `Coverage(workspace, contract, component, instance)` returns the active applications of a scope in order of service interval start, then identifier. `Source(workspace, source)` returns the source's capacity and its active applications with the remaining quantity and amount. `Eligibility(workspace, contract, number)` returns a revision with its approval.

An application's digest is the SHA-256 of a print in which every field is its length in eight bytes, big-endian, then its bytes, numbers as canonical decimal text and absent parts as empty fields: the label `billing_application_v1`, the workspace and every field of the application except its lifetime. An eligibility revision's digest is printed under `billing_eligibility_v1` with its components in byte order of identifier.

## Error kinds, tenant boundary, time and limits

A request is judged: sizes, a limit error (components before they are read); form by the request alone, invalid; lookup, not found; a repeat when identical, a conflict otherwise; a stored state or rule, rejected; room, a limit error. Refusals of `Reserve` for reasons 1 to 9 are results, not errors. Errors are fixed values. Every operation takes the workspace and reaches only its own records. All operations run under one registry lock, so concurrent reservations of one scope are judged one after the other. The registry's time is the highest reading of its clock, in UTC, recorded on every write; expiry is judged against it. Limits, each required and bounded: contracts per workspace (100,000), revisions per contract (1,000), components per revision (1,000), applications per workspace (1,000,000), reversals per application (1,000), applications in one `Confirm` (10,000), all within a 32-bit `int`.

## What is not decided here

Provisional defaults, each to be confirmed or changed by the owner: the scope as contract, component and instance; the two modes and the rule that a scope keeps one mode while active; a source consumed under one scope only; the reservation lifetime bound of one day; a whole reversal only in `interval` mode; reversals confirmed when recorded; a service interval within one eligibility revision. Not conservative, stated: eligibility and routes are what the caller records; a released or expired reservation frees its scope without a trace in billing; an expired reservation that a caller had already turned into a document is not detected here.

# Allowed files

`modules/claims/*` (new); `docs/features/HB-214/*`.

# Verification

- Literal tests: a direct contract where CSP usage is not billable and only the service fee is reserved, and a resale contract where only approved eligible components are reserved (AUD-AR-01 at package level); an `external_order` route refused under a new application identifier and an `unknown` route held (AUD-DB-03 at package level); a confirmed monthly fee refused under a new identifier, another actor and another calculation revision, while the next month and another instance are accepted; a source of 100 units and 50.00 billed 60 / 30.00 and then 40 / 20.00, a third application refused `capacity_exceeded`; a partial reversal freeing 10 units for rebilling; a whole reversal of a monthly fee freeing the month; a partial reversal in `interval` mode rejected; replacement of a reserved candidate and refusal to replace a confirmed one; expiry; batch confirmation all or none; every refusal reason alone and the order of reasons; every error with the order of error kinds; repeats and conflicts; digests against bytes built in the test; the tenant boundary; limits; leap days and invalid dates; a scan for floating point and comments; a build for linux/386.
- Concurrency under the race detector: many goroutines reserving overlapping intervals and one source at once; at most one interval application active and the source never over capacity.
- Properties with fixed seeds: random sequences of reserve (with `replaces` and `corrects`), confirm, release, reverse, approve reversal, record, approve and supersede eligibility revisions, and clock moves, checked after every step against a reference model written in the test with `math/big.Rat`, which shares the author's reading of this plan, with invariants: no two active interval applications of one scope overlap; active consumption of a source never exceeds its capacity; a confirmed application never becomes inactive without effective reversals equal to it; a pending reversal frees nothing.
- A mutation check by a script in Python with restores read back and the build cache emptied after every part; `gofmt -l`, `go vet ./...`, `go test -race -count=1 ./...`.

Remaining NOT_RUN at system level: AUD-AR-01, AUD-DB-03, AUD-QA-09, ERPC-01 to ERPC-08. This ticket does not complete W-21.

# Deferred

Everything listed as not delivered above.

# Design review resolution

The independent design review of 2026-11-14 returned FAIL without need for redesign, findings F1 to F13. The rules below are part of the design and prevail over the sections above where they differ.

- F1, anchored identities: each component of an eligibility revision declares its instances, 1 to 1,000 identifiers. `Reserve` refuses an instance the covering revision does not declare for the component with `unknown_instance`, after `not_eligible`. A source identifier must be the provider's stable line or usage key, not one minted by a caller; the same usage presented under a new source identifier is not detected (not conservative, stated).
- F2, periods in `source` mode: two sources of one scope with overlapping service intervals are both accepted, because one month of usage normally has many sources; in `source` mode, period duplicates are prevented only by the source's identity. A literal test shows this.
- F3, corrections: an application has an optional `corrects`, the identifier of a reversal or of a confirmed or reversed application of the same scope; it must exist with that scope, otherwise the request is rejected; it is recorded and printed in the digest and does not change any judgement. A source's capacity (quantity, amount and currency) is recorded with the first application of the source that is reserved; later applications must state that capacity, otherwise `capacity_mismatch`, whatever the state of earlier applications. (A capacity revision was proposed here and withdrawn under N2.) In `interval` mode an additional charge for a billed interval needs a whole reversal and a new application that `corrects` it (provisional).
- F4, ending eligibility: approving revision n whose interval overlaps an approved revision m of the same contract is accepted only when m's from date is earlier than n's; m's effective to date becomes n's from date. An overlap with an equal or later from date is rejected. `Confirm` judges reasons 1 to 4 and `unknown_instance` again for every reservation of the batch against the revisions in force at that moment; any refusal rejects the whole call. Confirmed applications under a superseded revision stay as they are (stated).
- F5, order: a request is judged by size; form; lookup (the identifier named by `replaces` or `corrects`; a contract without revisions is not a lookup failure but `no_eligibility`); repeat or conflict; the state of the `replaces` target and the scope of `replaces` and `corrects` (rejected); then the refusals as results, in the order `no_eligibility`, `not_eligible`, `unknown_instance`, `routed_externally`, `route_unknown`, `mode_mismatch`, `currency_mismatch`, `already_claimed`, `capacity_mismatch`, `source_in_other_scope`, `capacity_exceeded`; then room. A repeat is judged before the `replaces` state, so an identical repeat returns the stored application even when its target is now `replaced`.
- F6, seams: a prepaid balance or credit is not a source here; W-22 (and W-37 for opening balances in migration) may add an application kind or revise this package, and QA-09 belongs to them. The ERP recognition cadence is carried by W-15, W-21 and W-23 to W-31; the part of W-21 is stated here: monthly recognition submissions are not applications and consume no coverage; a quarterly interval application over months recognized monthly is accepted as one application; double cost recognition is not detected by this package.
- F7, reversals: `Reverse` records a reversal as `pending` by one actor; `ApproveReversal(workspace, identifier, approver)` by another actor makes it effective, and only effective reversals free coverage. The target must be `confirmed` (possibly partly reversed) when recorded and when approved; a target that is `reversed`, reserved, released, replaced or expired is rejected; the limits on quantity and amount are judged at both steps over the effective reversals. An identical repeat returns the stored reversal; another request under its identifier is a conflict; approval by the recording actor is rejected; a repeat approval by the same approver returns it, another approver is a conflict. A reversal approved in error cannot be undone, and its scope may already be billed again (stated cost). `Reversals(workspace, application)` lists an application's reversals.
- F8, time and batches: the clock is read once per call. A reservation is active while the registry's time is before its expiry and expired from its expiry on. A batch naming an identifier twice is invalid. In a batch, an application already `confirmed` or `reversed` counts as a repeat; the actor is not compared. `Release` of a released application returns it; of a confirmed, reversed, replaced or expired one it is rejected.
- F9, prints: the application digest prints, in order: the label, the workspace, the identifier, contract, component, instance, start, end, mode, source, capacity quantity, capacity amount, quantity, amount, currency, calculation, replaces, corrects. State, expiry, lifetime, actors, times and reversal totals are not printed. An identical repeat compares the printed fields and the lifetime; the actor is not compared. The eligibility digest prints the label, the workspace, contract, number, from, to, path, the count of components and, per component in byte order, identifier, kind, billable, route, reference, the count of instances and the instances in byte order; the approval is not printed. A reversal digest prints `billing_reversal_v1`, the workspace, its identifier, the application, quantity and amount.
- F10, currencies: `currency_mismatch` refuses an application whose currency differs from that of an active application of the same scope, in either mode. A scope with an active application in one mode refuses the other mode until those applications are released, expired or wholly reversed (stated).
- F11, limits: applications per workspace (1,000,000) include released, replaced and expired ones (stated cost); sources per workspace (1,000,000); instances per component (1,000); reversals per application (1,000). `Coverage` and `Source` return at most the applications of one scope or source, bounded by those limits.
- F12, modelling: payment responsibility per item is reduced to whether the MSP may bill a component (provisional); an `external_order` covers a whole component and instance for the whole effective interval, so a prepayment of part of a quantity cannot be expressed (not delivered); `route_unknown` is a refusal result that stores nothing, and a hold record for AUD-DB-03 is not delivered.
- F13, tests added: expiry between reservation and a batch confirmation; replacing across scopes and an expired application rejected; reversing a reversed application rejected; a mode switch after a whole reversal; a capacity or currency mismatch after release; a revision boundary inside an interval; approvals by the author and conflicting approvers; overlapping and superseding revisions; `Confirm` refused after a revision is superseded with a route change; `corrects` recorded.

# Design re-review resolution

The independent re-review of 2026-11-14 returned FAIL with findings N1 to N7, none needing a redesign. The rules below prevail over the sections above where they differ.

- N1, renaming through supersession: approving a revision n that supersedes an approved revision m is rejected while an active application of the contract, recorded under any component and instance, has a service interval that overlaps n's interval (it ends after n's from date and starts before n's to date), unless n declares that component, billable on the platform route, with that instance. A confirmed scope therefore keeps its identity across revisions, and a renamed instance cannot bill a confirmed period again. A literal test covers the case.
- N2, capacity: `ReviseCapacity` is withdrawn. A source's capacity is fixed by its first reserved application and never changes; a provider's correction of a usage line arrives as a new source line with its own identifier and capacity (stated assumption, not conservative). `Confirm` judges again each reservation's stated capacity against the source's capacity, which cannot have changed.
- N3, gaps: a superseding revision n must end no earlier than the effective end of every revision it supersedes; otherwise its approval is rejected, so no tail is left without eligibility silently. `Eligibility` returns the effective end date; the digest prints the original end date.
- N4, pending reversals: pending reversals that are never approved count toward the limit of reversals per application, and nothing cancels them (stated cost). Several pending reversals may each reach the whole remaining amount; the first approved takes effect and a later approval that would exceed the application is rejected, so the order of approvals decides (stated). A literal test approves two whole pending reversals.
- N5, `corrects`: an application names `corrects_kind`, `application` or `reversal`, with `corrects`; both are empty or both are given. A reversal named must be effective and belong to an application of the same scope. Application and reversal identifiers are separate namespaces. Both fields are printed in the digest.
- N6, refusals in `Confirm`: `Confirm` is judged by size; form (an identifier twice is invalid); lookup of every identifier; the state of each in batch order (an application expired, released or replaced rejects the call); then each reservation is judged again in batch order for `no_eligibility`, `not_eligible`, `unknown_instance`, `routed_externally`, `route_unknown` and `capacity_mismatch`, in that order; the first refusal rejects the call. A rejected call returns the identifier of the first application in batch order that caused it and, for a judgement, its reason, with the error `ErrRejected`; nothing changes.
- N7, verification: amended in place.

# Second re-review resolution

The second re-review of 2026-11-14 returned PASS with four minor findings, resolved as follows.

- M1: the withdrawn capacity revision is struck from the F3 and F13 resolutions in place.
- M2: `Confirm` judges reasons in the order of `Reserve`, stated in place in N6.
- M3: a source's capacity may be set again by a new reservation while the source has no application that is active or was ever confirmed; from its first confirmation the capacity is fixed. A mistyped capacity in a reservation that was released or expired therefore does not block the source.
- M4, provider restatements: when a provider restates a usage line in full under a new source line, the caller reverses the original application (a reversal recorded by one actor and approved by another) and reserves the new line with `corrects` naming that reversal. Detecting that a new source line restates an earlier one is not delivered and is not conservative (stated); a restatement billed without that procedure is billed on top of the original.

# Change review resolution

The independent change review of 2026-11-14 returned FAIL with one major finding and three minor ones. They are resolved as follows. The corrections were not reviewed again.

- C1 (major): a confirmed application of quantity 0 and amount 0 read as `reversed` at once, freeing its scope to be billed again. An application is now `reversed` only when it has at least one effective reversal and its effective reversals equal it. The property model has the same rule. A literal test confirms a 0/0 monthly fee and a 0/0 source application and checks that the month is still claimed and the source still bound to its scope. The test and the property test both fail when the fix is removed.
- C2: the property invariant now reads the reversals from the registry. A confirmed application is inactive exactly when it has effective reversals equal to it.
- C3: the property test does not exercise `corrects`, eligibility revisions or supersession, and uses one workspace. These are covered by literal tests only, and the verification record says so.
- C4: `ErrSelfCheck`, which nothing returned, is removed from this package.
