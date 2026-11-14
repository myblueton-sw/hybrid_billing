---
type: VerificationRecord
title: Billing application ledger verification
description: Records what was executed and reviewed for the first model of work item W-21, with results, the mutation check, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-214](https://linear.app/hybrid-billing/issue/HB-214/w-21-billing-application-ledger-with-eligibility-reservations) adds the Go package `modules/claims` from main `a525099`. A registry keeps:

- **Eligibility revisions per contract**, written once and approved by another actor. Each has an effective interval, a `direct` or `resale` path and components. A component carries its kind, whether the MSP may bill it, its route (`platform`, `external_order` with a reference, or `unknown`) and its declared instances. Billable CSP usage under a direct path is invalid.
  - A later revision supersedes an earlier one from its from date.
  - It must not end before the revision it supersedes.
  - It must declare every component and instance with an active application in its interval.
- **Applications**, identified by contract, component and instance, in `interval` or `source` mode. They are reserved with a lifetime, replaced only while reserved, confirmed in batches all or none, and released.
  - A reservation is refused, without storing anything, with the first of eleven reasons: `no_eligibility` to `capacity_exceeded`.
  - Identifiers, actors and calculation revisions are recorded but are not part of the identity.
  - `Confirm` judges eligibility again.
- **Reversals**, recorded by one actor and made effective by another. In `interval` mode only a whole reversal is allowed; in `source` mode reversals may be partial. Only effective reversals free coverage.
  - `corrects` links a new application to a reversed or confirmed application, or to an effective reversal.
  - A source's capacity is fixed from its first confirmation.

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan three times and the change once (see Independent review). Machine: mac_mini. Nothing calls the package, and the registry authorizes nothing.

At the level of this package, these HB-21 acceptance cases hold:

- **AUD-AR-01:** under a direct contract only the service fee is reserved; under resale only the approved eligible components are.
- **AUD-DB-03:**
  - an `external_order` route is refused under new submission identifiers and calculation revisions;
  - an `unknown` route is refused.
- **Re-billing:** a confirmed month is refused under a new identifier, client and calculation revision. The next month and another declared instance are accepted.
- **Partial billing:** a source of 100 units / 50 is billed 60 / 30.00 and 40 / 20, and a further application is refused. A partial reversal of 10 / 5 frees exactly 10 / 5, which is billed again with `corrects`.
- **Late correction:** a whole reversal of a month frees it for a corrected charge of 120.

# Executed checks

Executed on 2026-11-14 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain and a `GOCACHE` of the session.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All thirty packages `ok` |
| `go test -race -count=3 ./modules/claims` | `ok` |
| `go test -count=10 ./modules/claims` | `ok` |
| `GOOS=linux GOARCH=386 go test -c ./modules/claims` | Builds; no test was run on that platform |
| Statement coverage | 99.4% |

Literal tests with values worked out by hand:

- **Acceptance cases:** the HB-21 cases above.
- **Refusal order:** every refusal reason, and their order in pairs.
- **Revisions:**
  - eligibility forms and conflicts, field by field;
  - supersession, adjacency in both orders, and a tail left uncovered;
  - renaming an instance or dropping a component with active applications;
  - an interval across two revisions.
- **Confirmation:**
  - `Confirm` refused after a superseding revision leaves a reservation across the boundary, returning that application with `no_eligibility`;
  - batches all or none, with expiry inside a batch, an identifier twice and repeats by another actor.
- **Reservations:**
  - replacement within a scope, across scopes, of replaced, expired and confirmed applications;
  - expiry at exactly the lifetime;
  - release in every state.
- **Reversals:**
  - a whole reversal, and a partial one rejected in `interval` mode;
  - two whole pending reversals, of which only the first approved takes effect;
  - a pending reversal freeing nothing;
  - approval by the author rejected, repeat and conflicting approvals;
  - fractions down to 0.0001 and 0.00000008 in views.
- **Capacity:** set again after a release or expiry, and fixed after a confirmation.
- **Coverage order:** by start, then by identifier.
- **Forms and errors:**
  - every refusal of every operation in the order of error kinds;
  - forms at and beyond their bounds (identifiers, currency, dates including leap days, a 64-byte number, a lifetime of a day, a thousand instances, revision 1,000, a batch at its limit);
  - a repeat of the last revision not stored twice.
- **Digests and boundaries:**
  - digests against bytes built in the test and against every printed field;
  - the lifetime not printed;
  - the tenant boundary;
  - no workspace created by refused requests;
  - a scan of every source file for words of floating point and comments.

A property test with a fixed seed runs 400 rounds of 120 random steps:

- **Steps:** reservations (with replacement), batch confirmations, releases, reversals with and without approval, and clock moves.
- **Setup:** one approved revision with platform, external, unknown and non-billable components.
- **Reference:** each step is compared with a model written in the test with `math/big.Rat` (reasons, error kinds and every application's state). The model follows this plan as the code does.
- **Invariants:**
  - no two active interval applications of a scope overlap;
  - no source is over capacity;
  - a confirmed application becomes inactive only through effective reversals.
- **Reach:** every refusal reason, errors, confirmations and reversals were reached at least five times.

The property test does not vary eligibility revisions, does not use `corrects`, and uses one workspace. Those are covered by the literal tests only. Its invariant on confirmed applications reads the reversals from the registry.

A concurrency test under the race detector runs eight goroutines, 50 times, against one interval and one source of 100 / 50. Exactly one interval application and exactly two source applications of 40 / 20 are reserved.

# Corrections made on the way

- A test that varied the path alone also changed a component, so a mutant comparing path and component count with `&&` survived. A test that changes the path alone was added.
- The first test of coverage order had three applications, so a non-strict comparison passed by chance. Nine applications in shuffled order are now used.
- The first fraction test needed more than the test limit of three reversals per application.

# Mutation check

A Python script (in the session's scratchpad, not committed) changes one line at a time in every source file of the package and runs the package's tests. It applies the same kinds of change as for HB-212, reads each restored file back, and empties the build cache after each batch.

Results:

- **First run:** of 631 mutants, 10 did not build, 543 were killed and 78 survived. Most survivors showed that no test used fractional amounts or exact bounds, so the tests in `bounds_test.go` were added.
- **Second run:** the 78 were re-run; 60 were killed and 18 survived.
- **Third run:** two of these were killed by the tests for the path and the order.

Final: 605 killed, 10 not built, 16 surviving that the author classifies as equivalent:

| Mutant | Why equivalent |
| --- | --- |
| `types.go:220` upper bound `2099-13-31` or `2099-12-32` (2 mutants) | A date above `2099-12-31` that passes the changed bound is not a calendar date and fails `time.Parse` |
| `types.go:224` `err == nil \|\| ...` | An invalid date fails to parse, gives the zero time, and that never formats back to the text |
| `registry.go:88` `Round(1)` | `Round` strips the monotonic reading for any duration |
| `registry.go:157`, `read.go:36` (identifier tie-break), `read.go:64` sorts with `<=` (3 mutants) | Keys are unique |
| `registry.go:263` `rev.From > other.effectiveTo` | At equality the earlier revision is "ended" at its own effective end, which changes nothing |
| `registry.go:271` `len(ended) >= 0` | Without a superseded revision no active application lies in the new revision's interval, since every application lies within an approved revision |
| `ledger.go:44`, `ledger.go:46` (2), `ledger.go:47` (4 mutants) | The capacity check in `Confirm` cannot fail: a reservation being confirmed is active, and an active application keeps its source's capacity fixed. It is kept as the plan's stated re-judgement |
| `digest.go:64` (2), `digest.go:83` | A fraction's text never starts with `/`; `fives >= scale` gives the same scale |

# Independent review

The design review returned FAIL (F1 to F13), a re-review FAIL (N1 to N7), and a second re-review PASS with four minor findings (M1 to M4). All resolutions are in the plan. The change review returned FAIL with one major finding (C1): a confirmed 0/0 application read as reversed, so its scope could be billed again. It was fixed with a test that fails without the fix, and C2 to C4 were resolved (see the plan's last section). The fixes were not reviewed again. After them, every check in the table was run again with the same results, and the 164 mutants of `reserve.go` were run again: 162 killed and 2 not built.

# Provisional choices awaiting the owner

- The scope as contract, component and declared instance, and the two modes; a scope keeps one mode while active.
- A source consumed under one scope at a time. Its identifier must be the provider's stable line key, and its capacity is fixed from its first confirmation. Provider restatements need a reversal and an application that `corrects` it.
- A reservation lifetime of at most one day.
- Reversals:
  - only whole reversals in `interval` mode;
  - two actors;
  - pending reversals are never cancelled.
- Eligibility revisions:
  - supersession only by a later from date;
  - no tail left uncovered;
  - a service interval within one revision.
- Payment responsibility reduced to whether the MSP may bill. An external order covers a whole component and instance, and an unknown route is a refusal that stores nothing.

# Not verified

- **System-level acceptance:** AUD-AR-01, AUD-DB-03, AUD-QA-09 and ERPC-01 to ERPC-08 stay NOT_RUN.
- **Not delivered:**
  - opening and prepaid balances (QA-09, W-22 and W-37);
  - reconciliation of purchase documents and the ERP routes (W-24, W-25);
  - the HTTP idempotency layer;
  - locks per contract;
  - persistence;
  - screens.
- **Inputs not seen:** no real contract, ERP order or provider line was seen.
- **Independent verification:** W-21 asks for independent verification by a DBA and by QA. This ticket had one independent reviewer of design and change.
- **Planning status:** the HB-21-D design task stays DRAFT with the owner.
