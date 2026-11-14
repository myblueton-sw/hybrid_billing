---
type: VerificationRecord
title: Marketplace transactions verification
description: Records what was executed and reviewed for the first model of work item W-20, with results, the mutation check, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-212](https://linear.app/hybrid-billing/issue/HB-212/w-20-marketplace-transactions-buyer-charges-seller-settlement-and) adds the Go package `modules/marketplace` from main `9cac406`. A registry keeps:

- marketplace transactions, written once, with the MSP's role, the channel, the offer type, the collector, the participants, the references and the contract term;
- buyer charge statements per transaction and month in revisions written once, with lines (software, infrastructure, other) and recognitions; each source discount and each recognition is `applied`, `not_applied` or `unknown`, and `unknown` never carries an amount. Only the highest revision is judged. It is sealed, or held for `duplicate_source` or `recognition_exceeds_charge`, and a held revision is judged again on the next seal;
- seller settlement statements for reseller and seller transactions, with gross, fee, refund, tax, hold, net, report and payout documents, and states that never move back. Each revision is judged when it is written: `net_not_preserved`, `received_differs` or `payout_reused` (a payout document used by another settlement of the same provider). A settlement is received only when its highest revision is `paid` and consistent;
- customer charge policies, approved by someone other than the author. At most one approved policy covers a transaction in a month, and a later version of the same policy supersedes the older one from its first month.

`ComputeCharge` reads only a sealed statement and an approved policy value from the same workspace and transaction. It discounts a line only when the line is in the policy's scope and either is not source-discounted with `not_applied`, or is source-discounted under `included` with its own contract basis. The discount is reduced toward zero to the quantum. A charge is held, in this order, for `seller_transaction`, `collected_elsewhere` or `source_discount_unknown` (the last only when the rate is above 0). A recognition never changes an amount. `ViewForCustomer` shows only source, kind, customer amount, total and digest. `AgreementCharges` sums sealed buyer charge statements per provider contract, grouped by role and currency.

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan three times and the change once (see Independent review). Machine: mac_mini. Nothing calls the package, and the registry authorizes nothing.

At the level of this package, the W-20 acceptance case gives these results:

- **Private offer, discount excluded:** a software line of 800 (list 1,000, source discount 200 applied) and an infrastructure line of 200 under a 20% policy on software total 1,000, not 840 or 640.
- **Discount included:** with `included` and a source basis, the software line becomes 640; infrastructure stays 200 when it is outside the scope and becomes 160 when it is inside.
- **Recognition:** a recognition of 800 changes no amount.
- **Settlement:** gross 800, fee 24, net 776 is consistent. It is not received while `expected` or `reported`. It is received when `paid` with 776 received, and held as `received_differs` with 775 received.
- **Late settlement:** writing a late settlement leaves a customer charge's digest unchanged.

# Executed checks

Executed on 2026-11-14 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain and a `GOCACHE` of the session.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All twenty-nine packages `ok` |
| `go test -race -count=3 ./modules/marketplace` | `ok` |
| `go test -count=10 ./modules/marketplace` | `ok` |
| `GOOS=linux GOARCH=386 go test -c ./modules/marketplace` | Builds; no test was run on that platform |
| Statement coverage before the boundary tests | 98.8% |

Literal tests with values worked out by hand:

- **The source example:** in the three policy variants above.
- **Discount rules:** which lines are discounted, in eleven cases, including:
  - `applied` on a public offer;
  - `unknown` inside and outside the scope and at rate 0;
  - a private offer's infrastructure line;
  - a discount reduced toward zero (0.333 at 10% gives 0.303);
  - a discount below one quantum.
- **Holds:** every hold, in order, for five transaction shapes, and a held charge refused a customer view.
- **Customer view:** checked field by field.
- **Settlement and charge:** a settlement written leaves a charge unchanged.
- **Settlements:**
  - every reason, alone and together, including `payout_reused` across agreements and not across providers;
  - states moving back refused;
  - a buyer transaction refused;
  - a correction judged again.
- **Seals:**
  - both seal reasons;
  - a line sold in a sealed revision staying linked until its correction is sealed;
  - a held revision judged again;
  - `Current` in three states;
  - superseded revisions;
  - duplicates across months and providers.
- **Policies:**
  - approval by the author refused, repeat and conflict;
  - overlap refused at its first and last months;
  - supersession, including across a year;
  - a value fetched before supersession recording its months in the result;
  - months outside the term or the policy refused.
- **Aggregation:**
  - complete and incomplete;
  - grouped by role and currency, in order over 30 runs with random map order;
  - the last month of a term.
- **Forms and errors:**
  - every refusal of every operation in the order of error kinds;
  - forms at and beyond their bounds (identifiers, currency, months, digests, a 64-byte number, revision 1,000, rates of eight decimals, limits of one);
  - digests against bytes built in the test, and against every changed field of a transaction;
  - the tenant boundary;
  - a clock read backwards;
  - a scan of the source for words of floating point and comments.

A property test with a fixed seed covers 3,000 random transactions, statements and policies over every role, offer type, collector, line kind, discount state, rate and quantum:

- Holds and every line amount, total, `charged_total` and margin are compared with a reference written with `math/big.Rat`. The reference follows this plan as the code does.
- Each line's amount plus its discount equals its charged amount, and no amount is above it.
- The total is never above `charged_total`.
- A statement with a recognition gives the same amounts as the same statement without it.
- Each hold and the unheld case was reached at least 50 times.

A concurrency test under the race detector seals two statements that share a source at once: neither passes. After one is corrected and sealed, the other seals.

# Corrections made on the way

- The first seal test expected a source line in an unsealed lower revision to keep counting after a correction was written. Under the plan, only the highest written and the last sealed revision count, so the test was rewritten so that the line is sold in a sealed revision first.
- The first policy test expected 5 extra policies to reach a limit of 6; it needed 6.
- The first test for the 64-byte bound used a number the test profile cannot hold; it now uses a profile with 70 integer digits.

# Mutation check

A Python script (in the session's scratchpad, not committed) changes one line at a time in every source file of the package and runs the package's tests. Its changes are:

- comparison operators;
- `&&` and `||`;
- error kinds;
- `Plus` and `Minus`;
- removal of `!`;
- reason and hold names;
- integer literals increased by one.

It reads each restored file back and stops if it differs, and it empties the build cache after each batch.

Results:

- **First run:** of 771 mutants, 18 did not build, 665 were killed and 88 survived. The surviving mutants showed untested boundaries, and the tests in `bounds_test.go` were added for them.
- **Second run:** the 88 surviving mutants were re-run; 66 were killed and 22 survived.
- **Third run:** of the seven validMonth mutants, one more was killed by a test for month 13.

Final: 732 killed, 18 not built, 21 surviving that the author classifies as equivalent:

| Mutant | Why equivalent |
| --- | --- |
| `types.go:89` rank of `paid` 3→4 | Order of states unchanged |
| `types.go:327` digit positions 0, 1, 2 and 5 skipped (4 mutants) | A non-digit there also fails the range checks against `2000-01`, `2099-12` and `12` |
| `types.go:332` upper bound `2099-13` | A month above 12 fails `text[5:] <= "12"` |
| `registry.go:89` `Round(1)` | `Round` strips the monotonic reading for any duration, and a reading in nanoseconds is unchanged |
| `statement.go:52`, `statement.go:53`, `agreement.go:54`, `agreement.go:95` `<` → `<=` in sorts (4 mutants) | Keys are unique, so no two elements compare equal |
| `policy.go:77` `ID` and `Version` joined by `&&` | Both are always equal, being the map key |
| `charge.go:26` index of `/` `>= 0` → `> 0` or `>= 1` (2 mutants) | A fraction's text never starts with `/` |
| `charge.go:45` `fives > scale` → `>=` | The same scale either way |
| `charge.go:96` `!s.made && !policy.made` | A value not made here has an empty workspace and fails the workspace comparison |
| `charge.go:129`, `charge.go:135`, `charge.go:147` self-checks joined by `&&` (5 mutants) | Unreachable: formats of exact fractions with denominators of 2 and 5 always succeed, and the total is never above `charged_total` |

# Independent review

The design review returned FAIL (F1 to F13), a re-review FAIL (N1 to N6), and a second re-review PASS with three minor findings (M1 to M3). All resolutions are in the plan. The change review returned PASS with six minor findings, C1 to C6, resolved in the plan's last section. Those corrections (groups only from sealed statements, more digest tests, a scan of every source file) were not reviewed again and not re-run through the mutation script; after them, every check in the table above was run again with the same results.

# Provisional choices awaiting the owner

- The closed lists of roles, channels, offer types and collectors.
- Source-discounted lines: lines with an `applied` source discount, and software lines of a private offer. They are excluded from a discount unless a policy includes them with a separate contract basis.
- A discount rate below 1 with at most eight decimals, as a discount only, with no markup. The discount is reduced toward zero per line, which favours the MSP by less than one quantum per line.
- Holds:
  - a seller transaction;
  - a reseller transaction with any collector other than the MSP;
  - a buyer transaction must have the MSP as collector;
  - unknown source discounts in scope at rates above 0.
- `recognition_exceeds_charge` judged per program against the statement's charged sum.
- Duplicates keyed by provider and source identifier across months, which assumes providers do not reuse line identifiers.
- The settlement formula: net = gross − fee − refund − tax − hold.
- `payout_reused` across all agreements of one provider. Aggregated payouts stay held until payout aggregation is delivered.
- One approved policy per transaction and month, and supersession only by a later first month of the same policy.

# Not verified

- **System-level acceptance:** MP01 to MP04, AUD-DB-10 and AUD-DB-14 stay NOT_RUN.
- **Not delivered:** everything the plan lists as not delivered, including:
  - the product's own marketplace sales (DB-14, W-40);
  - fee schedules;
  - refunds, renewals and reversals;
  - an issuance ledger;
  - account-to-customer mapping;
  - rounding to currency minor units;
  - reconciliation with invoices or settlements;
  - screens and persistence.
- **Inputs not seen:** no real provider sample, offer, fee or settlement report was seen.
- **Independent verification:** W-20 asks for independent verification by a DBA and by QA. This ticket had one independent reviewer of design and change.
- **Planning status:** the HB-20-D design task stays DRAFT with the owner.
