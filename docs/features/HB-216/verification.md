---
type: VerificationRecord
title: Prepaid and credit balance ledger verification
description: Records what was executed and reviewed for the first model of work item W-22, with results, the mutation check, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-216](https://linear.app/hybrid-billing/issue/HB-216/w-22-prepaid-and-credit-balance-ledger-with-opening-references-atomic) adds the Go package `modules/balances` from main `e0790d1`. A registry keeps, per workspace:

- **Accounts**, one per customer, legal entity, currency and balance kind (`cash_prepayment` or `credit`), under a `platform` or `external` authority.
- **Opening balance references** (QA-09) with prior ledger, as-of date, amount, evidence digest, migration key, included first date and optional expiry. They are approved by another actor or withdrawn while pending.
  - An imported transaction on or before the as-of date of an opening of its ledger, or of its account's opening, is rejected.
  - An opening is rejected while the workspace holds an active import of its ledger on or before its as-of date.
  - A transaction is imported once per workspace.
- **Increases** of a kind matching the balance kind (`cash_receipt`; `credit_document`, `promotion`, `sla_credit`), unique by kind and reference. They are approved by another actor, become lots with an optional expiry, and can be withdrawn while pending.
- **Applications** to a target document line. They are reserved atomically for the least of the requested amount, the target's room and the available balance, or refused with the first of nine reasons, then confirmed or released.
  - Holds go to lots by earliest expiry, and a reservation never outlives its lots.
  - A target's customer, entity, currency and amount are fixed while it has an active or ever-confirmed application.
- **Reversals** of applications, increases and openings, recorded by one actor and made effective by another, limited at both steps. Application reversals return to the lots used, last first, and expire at once on an expired lot.
- **A statement** per platform account: opening + increases − applications − expired + application reversals − lot reversals = closing, with held and available amounts.
- **External accounts** hold every application (`external_authority`) and keep no balance.

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan twice and the change once (see Independent review). Machine: mac_mini. Nothing calls the package and the registry authorizes nothing.

At the level of this package:

- **HB-22-V:** 100 USD with partial applications of 60 and 50 applies 60 and 40 and leaves 10 unpaid, in both orders and with two goroutines, 20 rounds under the race detector. The same reversal is not applied twice; a second reversal beyond the remainder is rejected.
- **AUD-QA-09:** an opening as of 2026-09-30 refuses imports on or before that date, from its own account or another account of the same ledger, and accepts 2026-10-01. A second import of a transaction is a conflict in any account. An opening recorded after an import it would include is rejected. The identity holds after concurrent use, expiry and reversal.

# Executed checks

Executed on 2026-11-14 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain and a `GOCACHE` of the session.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All 31 packages `ok` |
| `go test -race -count=3 ./modules/balances` | `ok` |
| `go test -count=10 ./modules/balances` | `ok` |
| `GOOS=linux GOARCH=386 go test -c ./modules/balances` | Builds; no test was run on that platform |
| Statement coverage | 99.2% |

The last test (`TestOpeningOnTheDateOfAnImport`) was added after this table; the package tests were run again with it and passed, and it killed its two mutants.

Literal tests with values worked out by hand:

- **Acceptance cases:** the cases above.
- **Refusals:** every reason alone, the order of reasons, a target set again once nothing is active, and a target fixed after a confirmation even when wholly reversed.
- **Openings and imports:**
  - pending openings blocking increases and reservations;
  - approval by the author;
  - withdrawal and the keys and imports it frees;
  - a second opening;
  - a zero opening making no lot;
  - expiry dates at record and at approval, including the expiry date itself;
  - imports on the as-of date, on the first included date, before it, today and after the registry's date;
  - partial imports invalid;
  - imported applications reopening their import when released or expired.
- **Lots and expiry:**
  - allocation by earliest expiry and lot number;
  - a reservation capped at its lot's expiry;
  - a confirm at the exact instant of expiry rejected;
  - an expired lot reported with nothing remaining;
  - a used-up lot not capping a reservation;
  - an amount returned to an expired lot expiring at once.
- **Reversals:**
  - pending reversals freeing nothing;
  - two whole pending reversals, of which only the first approved takes effect;
  - repeat and conflicting approvals;
  - returns split across two holds, last first;
  - lot reversals limited by holds at record and approval;
  - opening reversals;
  - expired lots rejected.
- **Forms and errors:** every operation's error kinds in order, fields one at a time, a 64-byte number accepted and 65 rejected (with a wide profile), a lifetime of 1 second and a day, limits at bottom and top.
- **Digests and boundaries:**
  - digests of the five records against bytes built in the test;
  - the tenant boundary for every operation;
  - fractions down to 0.00000008;
  - a scan of every source file for words of floating point and comments.

A property test with fixed seeds runs 40 rounds of 250 random steps:

- **Steps:** increases (with imports, expiry and withdrawal), approvals, reservations, confirmations, releases, reversals of applications and increases, approvals and clock moves, over three accounts and four targets.
- **Checks after every step, against a model in `math/big.Rat`:**
  - the statement totals;
  - the identity;
  - the lots against the statement;
  - expiry from each lot's own expiry date;
  - held amounts against the account's reservations;
  - each target's active amount within its amount;
  - reversals within what they reverse.
- **Checks per step:** the expected refusal reason of every reservation, the expected error of every import, and whether every reversal is within its limit, from the registry's views.
- **Reach:** every refusal reason except `external_authority` and `opening_pending`, both import errors and every operation are reached at least five times.
- **Sensitivity:** breaking the opening exclusion makes it fail.
- **Limits:** the available balance is read from the registry and checked through the lots and the identity, not computed by the model.

# Corrections made on the way

- An expectation in the reversal test assumed the wrong lot order; the lots were allocated by number as designed and the test was corrected.
- The first property test picked applications from a Go map, which is not reproducible; it now picks from slices.
- The mutation script was first started with a shell loop that did not split its arguments, and ran nothing; it was run again correctly.

# Mutation check

A Python script (in the session's scratchpad, not committed) changes one line at a time in every source file of the package and runs the package's tests. It reads each restored file back and empties the build cache after each batch.

- **First run (before the change review):** of 681 mutants, 18 did not build, 615 were killed and 48 survived. `edges_test.go` was added, and the survivors were run twice more, leaving 14.
- **After the change review fixes, all 702 mutants were run again:** 19 did not build, 666 were killed and 17 survived. A test of an import on the as-of date killed two.
- **Final:** 668 killed, 19 not built, and 15 surviving that the author classifies as equivalent.

| Mutant | Why equivalent |
| --- | --- |
| `types.go:255` `&&` in the length test, upper bound `2099-13-31` or `2099-12-32` (3) | Every such text fails `time.Parse` or the round trip |
| `types.go:259` `err == nil \|\| ...` | An invalid date parses to the zero time, which never formats back to the text |
| `registry.go:136` `Round(1)` | `Round` strips the monotonic reading for any duration |
| `registry.go:276` `\|\|` in `prune` | Keeps more entries in the live list; `held` filters them again (performance only) |
| `lots.go:41` `len(acct.apps) > 1` | An account cannot have an application without an increase, which the same condition already rejects |
| `apply.go:71` `exact.Whole(1)` | The value returned with a refusal is not used |
| `apply.go:122` `<=` on lot numbers | Lot numbers are unique |
| `apply.go:127` `left.Sign() < 0` | Adds zero holds on later lots, whose expiry is no earlier than the ones used |
| `reverse.go:116` `left.Sign() >= 0`, `\|\|` (2) | Further iterations return zero; the reversal is limited to what the holds can take |
| `digest.go:74` (2), `digest.go:93` | A fraction's text never starts with `/`; `fives >= scale` gives the same scale |

# Independent review

- **Design review:** FAIL with F1 to F12, among them four major findings: customer boundary, duplicate increases, imports across accounts, and order. All were resolved in the plan.
- **Re-review:** PASS with five minor findings (N1 to N5), resolved in the plan.
- **Change review:** PASS with C1 (major, a gap in the approved design: an opening recorded after an import of its ledger) and C2 to C4 (minor). All were resolved, as described in the plan's last section. The test for C1 fails without the fix.
- **After the fixes:** every check in the table was run again and the whole mutation set was run again. The corrections were not reviewed again.

# Provisional choices awaiting the owner

- **Accounts:** the two balance kinds and four increase kinds; one account per customer, entity, currency and kind.
- **Expiry and allocation:**
  - expiry at 00:00 UTC after the expiry date, not in the contract's time zone;
  - allocation by earliest expiry;
  - a reservation capped at its lots' expiry;
  - an amount returned to an expired lot expiring at once.
- **Approvals:**
  - two actors for openings, increases and reversals, but not for applications;
  - withdrawal of pending openings and increases by any actor.
- **External accounts:** every application on an external account is refused with no hold record.
- **Imports:** an import whose ledger appears in no opening is accepted. A whole ledger is blocked across customers by any opening of that ledger.
- **Concurrency:** one registry lock, and a reservation lifetime of at most one day.

# Not verified

- **System-level acceptance:** AUD-QA-09, AUD-AR-08, AUD-AR-10, AUD-DB-10 and PT-01 to PT-08 stay NOT_RUN.
- **Not delivered (W-22 follow-ups):**
  - BillingFinding (AR-08);
  - BillingCase and disputes with inquiry permissions;
  - AR-10 components and SLA credit rules;
  - DB-10;
  - payment terms and aging;
  - refunds;
  - reservation, lock and staleness with an external authority;
  - included-pool balances;
  - per-account locks;
  - persistence;
  - screens.
- **Inputs not seen:** no real receipt, credit document or prior ledger was seen.
- **Independent verification:** W-22 asks for independent verification by a DBA and by QA. This ticket had one independent reviewer of design and change.
- **Planning status:** the HB-22-D design task stays DRAFT with the owner.
