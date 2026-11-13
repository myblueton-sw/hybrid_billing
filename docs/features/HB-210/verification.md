---
type: VerificationRecord
title: Commitment results verification
description: Records what was executed and reviewed for the first model of work item W-19, with results, the DB-09 event matrix, corrections made on the way, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-210](https://linear.app/hybrid-billing/issue/HB-210/w-19-provider-commitment-facts-cost-preservation-allocation-of-used) adds the Go package `modules/commitment` from main `93a9aaf`. A registry keeps commitments (provider, payer account, kind, currency, months, payment option, prepaid amount, quantum, recovery of the purchase) and period facts as the provider states them, in revisions written once; only the highest revision of a period is judged, sealed when its lines refer to its commitment, no source line is linked in another period, its parts sum exactly to the period cost and the amortized parts of the commitment do not exceed its prepaid amount, or held with those reasons; `Current` tells whether a period is usable. `Allocate` splits a sealed period into customer used cost, unused cost borne by the MSP, a named customer, customers pro rata or nobody, a rounding part and the unattributed part, summing exactly to the period cost. `Sell` computes charges under `cost_plus` or `savings_share`, rounded once, leaves recovered amortized parts out, holds a customer without a policy, with negative savings under `savings_share` or with a recovered purchase under `savings_share`, and gives the MSP result and `below_cost` only when every customer is priced. Results carry the digests of the facts and policies they came from.

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan three times and the change once (see Independent review). Machine: mac_mini.

Nothing calls the package. Of the acceptance of HB-19, at the level of this package: the source example of chapter 45 gives 66, 27.50, total 93.50 and −6.50 below cost; alternative 1 gives 81, 27.50, 108.50 (110 with the fee on unused cost); alternative 2 gives 80, 37.50, 117.50 and 17.50; a purchase of 1,200 recovered is not charged again monthly, and charged it gives 110 with a 10% fee. Cost preservation, re-collection of a prepayment, a line linked twice and late corrections are tested in this package; what is not delivered is listed in the plan and in the matrix below. The registry authorizes nothing and must not be exposed directly.

# Executed checks

Executed on 2026-11-13 and 2026-11-14 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain and a `GOCACHE` of the session.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All twenty-eight packages `ok`; run on the final code |
| `go test -count=10 ./modules/commitment` | `ok` on the final code |
| `GOOS=linux GOARCH=386 go test -c ./modules/commitment` | Builds; no test was run on that platform |

Literal tests with values worked out by hand: the source example in every variant named above, each naming its bearing policy and sale method; the prepayment case recovered and charged, and `savings_share` refused with a recovered purchase; every seal reason alone and together; a held correction after a sealed revision, `Current` in its three states, superseded revisions, a lower revision refused; a line moved between periods in both orders and a correction lowering an amortized part; duplicates across payer accounts and commitments; pro rata with a rounding part in both parts (15.005 and 0.007 at 0.01 giving 10.59 and 4.41 with rounding parts 0.005 and 0.007); a recovered purchase with unused cost borne by a customer (8.80 and 2.20) and pro rata (7.83 and 3.17, exact 7.832); a customer bearing only unused cost under both methods; negative savings under both methods and with an amortized part; every quantum on a charge; holds of allocations and their order; refusals of every operation in the order of error kinds; repeats and conflicts; digests against bytes built in the test and against every changed field and a byte moved; limits at and beyond their bounds; the tenant boundary; inputs built outside the package and zero values refused; a scan of the source for words of floating point and comments, which is a scan of words.

A property test with a fixed seed: 1,500 random periods of the three payment options and quanta, each sealed, allocated under a random bearing policy and sold under random policies and modes; the parts are compared with each customer's used and list cost computed from the generated lines, the expected bearer of unused cost, bounds on pro-rata shares and the period cost, and each charge, total, cost compared with and MSP result with a reference written with `math/big.Rat`; some rounds leave an account unmapped. The reference's sale formulas follow this plan as the code does. A concurrency test under the race detector seals two periods of one commitment at once against its prepaid amount.

# DB-09 event matrix

Each event under the three payment options, with one status: covered by a named test with synthetic data or the source's figures; or not built in this package and deferred to a named ticket. No event is recorded as unsupported by a source, since no source evidence was read for any.

| Event | All upfront | Partial upfront | No upfront |
| --- | --- | --- | --- |
| Payment all upfront | Covered: `TestAPrepaymentIsNotChargedAgain`, `TestCorrectionsAndCurrent` (synthetic; 1,200 prepaid, 100 amortized; 0 recovered, 110 charged) | — | — |
| Payment partial upfront | — | Covered: `TestBearingPoliciesAndTheRoundingPart`, `TestAmortizedPartsInSavingsAndBasis` (synthetic) | — |
| Payment no upfront | — | — | Covered: `TestTheSourceExample` (the source's figures) |
| Utilisation 0% | Covered under pro rata: `TestAllocationHolds` (`no_basis`) | Covered: `TestAmortizedPartsInSavingsAndBasis` | Covered: `TestAllocationHolds` |
| Utilisation 100% | Covered: `TestAPrepaymentIsNotChargedAgain` | Covered by the property test | Covered: `TestCustomerBearingOnlyUnusedCost` (no unused cost) |
| Excess usage beyond the commitment | Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) | Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) | Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) |
| Concurrent commitments | A line in two commitments of one payer is held `duplicate_source`: `TestDuplicatesAcrossPayersAndProviders`; sharing usage is Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) | Same | Same |
| Start or end within a month | Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) | Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) | Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) |
| Move within a month (a line moved between periods) | Covered: `TestCorrectionsAndCurrent`, `TestMovedLineInTheOtherOrderAndSupersededRevisions` | Same | Same |
| Change of shared scope | Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) (no commitment versions) | Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) | Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) |
| Late credit | Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) (a correction revision is covered: `TestCorrectionsAndCurrent`) | Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) | Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) |
| Refund or exchange | Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) | Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) | Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) |
| Correction after cancellation | Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) | Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) | Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) |
| Negative savings | Covered: `TestAmortizedPartsInSavingsAndBasis` | Covered: same | Covered: `TestNegativeSavings` |
| FX change | Not built here: FX is HB-208's and applies after this package; combining both is Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) | Same | Same |
| Partial CSV errors | Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) (no source completeness) | Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) | Not built here; [HB-211](https://linear.app/hybrid-billing/issue/HB-211/w-19-follow-up-completeness-reconciliation-prepaid-balances-commitment) |

# Corrections made on the way

- The first design review returned FAIL (a held correction left earlier facts current; a false loss and double recovery with recovered purchases; provider facts forced to a quantum; results that did not record their inputs). The re-review returned FAIL on two points (other periods' sealed revisions dropping out of the checks; savings undefined with a recovered purchase), resolved by rules; the confirmation returned PASS with minor points. All are resolved in the plan.
- The change review's findings C1 to C13 led to code changes (the refusal for a new period, the exact value as a decimal, the profile in a sale, held allocations without parts, the named customer only with unused cost, the page limit removed) and to tests; these were not reviewed again by a separate reviewer.
- My own expectations that were wrong and corrected: after a correction lowered an amortized part, a period I expected to stay held is sealed under the rule; a test case was mislabelled; a quantum of 0.001 rounds 67.86549 to 67.865, not 67.866.
- Code errors found by my tests: formatting fractions failed for denominators not a power of ten, and texts of fee rates were reduced to fractions; both were corrected before any review.

# Mutation check

A script in Python generated 602 mutants from the package's source by fixed rules and ran the package's tests for each, reading back every restored file and emptying the build cache after every part. The source did not change during the runs; tests were added between parts, and every mutant that had survived was run again on the final tests. 555 were killed, 24 did not build, and 23 survive and cannot be told apart, each with its reason in the table: checks already implied by others, values never read, ties that cannot occur, or conversions that cannot fail for amounts the profile read.

| # | Place | Mutant | Result |
| --- | --- | --- | --- |
| 1 | `types.go:18` | `maxIDBytes     = 129` | Killed |
| 2 | `types.go:19` | `maxNumberBytes = 65` | Killed |
| 3 | `types.go:78` | `quanta     = map[string]int{"2": 0, "0.1": 1, "0.01": 2, "0.001": 3, "0.0001": 4}` | Killed |
| 4 | `types.go:78` | `quanta     = map[string]int{"1": 1, "0.1": 1, "0.01": 2, "0.001": 3, "0.0001": 4}` | Killed |
| 5 | `types.go:78` | `quanta     = map[string]int{"1": 0, "1.1": 1, "0.01": 2, "0.001": 3, "0.0001": 4}` | Killed |
| 6 | `types.go:78` | `quanta     = map[string]int{"1": 0, "0.2": 1, "0.01": 2, "0.001": 3, "0.0001": 4}` | Killed |
| 7 | `types.go:78` | `quanta     = map[string]int{"1": 0, "0.1": 2, "0.01": 2, "0.001": 3, "0.0001": 4}` | Killed |
| 8 | `types.go:78` | `quanta     = map[string]int{"1": 0, "0.1": 1, "1.01": 2, "0.001": 3, "0.0001": 4}` | Killed |
| 9 | `types.go:78` | `quanta     = map[string]int{"1": 0, "0.1": 1, "0.2": 2, "0.001": 3, "0.0001": 4}` | Killed |
| 10 | `types.go:78` | `quanta     = map[string]int{"1": 0, "0.1": 1, "0.01": 3, "0.001": 3, "0.0001": 4}` | Killed |
| 11 | `types.go:78` | `quanta     = map[string]int{"1": 0, "0.1": 1, "0.01": 2, "1.001": 3, "0.0001": 4}` | Killed |
| 12 | `types.go:78` | `quanta     = map[string]int{"1": 0, "0.1": 1, "0.01": 2, "0.2": 3, "0.0001": 4}` | Killed |
| 13 | `types.go:78` | `quanta     = map[string]int{"1": 0, "0.1": 1, "0.01": 2, "0.001": 4, "0.0001": 4}` | Killed |
| 14 | `types.go:78` | `quanta     = map[string]int{"1": 0, "0.1": 1, "0.01": 2, "0.001": 3, "1.0001": 4}` | Killed |
| 15 | `types.go:78` | `quanta     = map[string]int{"1": 0, "0.1": 1, "0.01": 2, "0.001": 3, "0.2": 4}` | Killed |
| 16 | `types.go:78` | `quanta     = map[string]int{"1": 0, "0.1": 1, "0.01": 2, "0.001": 3, "0.0001": 5}` | Killed |
| 17 | `types.go:227` | `return l.Commitments > 0 && l.Commitments < 100000 && l.Revisions > 0 && l.Revisions <= 10` | Killed |
| 18 | `types.go:227` | `return l.Commitments > 0 && l.Commitments <= 100000 && l.Revisions > 0 && l.Revisions < 10` | Killed |
| 19 | `types.go:227` | `return l.Commitments > 0 && l.Commitments <= 100000 && l.Revisions > 0 && l.Revisions <= 1` | Killed |
| 20 | `types.go:227` | `return l.Commitments >= 0 && l.Commitments <= 100000 && l.Revisions > 0 && l.Revisions <= ` | Killed |
| 21 | `types.go:227` | `return l.Commitments > 0 && l.Commitments <= 100000 && l.Revisions >= 0 && l.Revisions <= ` | Killed |
| 22 | `types.go:227` | `return l.Commitments > 0 && l.Commitments <= 100000 && l.Revisions > 0 && l.Revisions <= 1` | Killed |
| 23 | `types.go:227` | `return l.Commitments > 0 \|\| l.Commitments <= 100000 && l.Revisions > 0 && l.Revisions <=` | Killed |
| 24 | `types.go:227` | `return l.Commitments > 0 && l.Commitments <= 100000 \|\| l.Revisions > 0 && l.Revisions <=` | Killed |
| 25 | `types.go:227` | `return l.Commitments > 0 && l.Commitments <= 100000 && l.Revisions > 0 \|\| l.Revisions <=` | Killed |
| 26 | `types.go:227` | `return l.Commitments > 0 && l.Commitments <= 100000 && l.Revisions > 0 && l.Revisions <= 1` | Killed |
| 27 | `types.go:227` | `return l.Commitments > 0 && l.Commitments <= 100000 && l.Revisions > 0 && l.Revisions <= 1` | Killed |
| 28 | `types.go:227` | `return l.Commitments > 1 && l.Commitments <= 100000 && l.Revisions > 0 && l.Revisions <= 1` | Killed |
| 29 | `types.go:227` | `return l.Commitments > 0 && l.Commitments <= 100001 && l.Revisions > 0 && l.Revisions <= 1` | Killed |
| 30 | `types.go:227` | `return l.Commitments > 0 && l.Commitments <= 100000 && l.Revisions > 1 && l.Revisions <= 1` | Killed |
| 31 | `types.go:227` | `return l.Commitments > 0 && l.Commitments <= 100000 && l.Revisions > 0 && l.Revisions <= 1` | Killed |
| 32 | `types.go:227` | `return l.Commitments > 0 && l.Commitments <= 100000 && l.Revisions > 0 && l.Revisions <= 1` | Killed |
| 33 | `types.go:227` | `return l.Commitments > 0 && l.Commitments <= 100000 && l.Revisions > 0 && l.Revisions <= 1` | Killed |
| 34 | `types.go:228` | `l.Customers > 0 && l.Customers < 10000 && l.Digits > 0 && l.Digits <= 10000` | Killed |
| 35 | `types.go:228` | `l.Customers > 0 && l.Customers <= 10000 && l.Digits > 0 && l.Digits < 10000` | Killed |
| 36 | `types.go:228` | `l.Customers >= 0 && l.Customers <= 10000 && l.Digits > 0 && l.Digits <= 10000` | Killed |
| 37 | `types.go:228` | `l.Customers > 0 && l.Customers <= 10000 && l.Digits >= 0 && l.Digits <= 10000` | Killed |
| 38 | `types.go:228` | `l.Customers > 0 \|\| l.Customers <= 10000 && l.Digits > 0 && l.Digits <= 10000` | Killed |
| 39 | `types.go:228` | `l.Customers > 0 && l.Customers <= 10000 \|\| l.Digits > 0 && l.Digits <= 10000` | Killed |
| 40 | `types.go:228` | `l.Customers > 0 && l.Customers <= 10000 && l.Digits > 0 \|\| l.Digits <= 10000` | Killed |
| 41 | `types.go:228` | `l.Customers > 1 && l.Customers <= 10000 && l.Digits > 0 && l.Digits <= 10000` | Killed |
| 42 | `types.go:228` | `l.Customers > 0 && l.Customers <= 10001 && l.Digits > 0 && l.Digits <= 10000` | Killed |
| 43 | `types.go:228` | `l.Customers > 0 && l.Customers <= 10000 && l.Digits > 1 && l.Digits <= 10000` | Killed |
| 44 | `types.go:228` | `l.Customers > 0 && l.Customers <= 10000 && l.Digits > 0 && l.Digits <= 10001` | Killed |
| 45 | `types.go:232` | `if id != "" \|\| len(id) > maxIDBytes {` | Killed |
| 46 | `types.go:232` | `if id == "" \|\| len(id) >= maxIDBytes {` | Killed |
| 47 | `types.go:232` | `if id == "" && len(id) > maxIDBytes {` | Killed |
| 48 | `types.go:235` | `for index := 0; index <= len(id); index++ {` | Killed |
| 49 | `types.go:235` | `for index := 1; index < len(id); index++ {` | Killed |
| 50 | `types.go:237` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c != '.` | Killed |
| 51 | `types.go:237` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.` | Killed |
| 52 | `types.go:237` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.` | Killed |
| 53 | `types.go:237` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.` | Killed |
| 54 | `types.go:237` | `if !(c >= 'A' && c < 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.'` | Killed |
| 55 | `types.go:237` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c < 'z' \|\| c >= '0' && c <= '9' \|\| c == '.'` | Killed |
| 56 | `types.go:237` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c < '9' \|\| c == '.'` | Killed |
| 57 | `types.go:237` | `if !(c > 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.'` | Killed |
| 58 | `types.go:237` | `if !(c >= 'A' && c <= 'Z' \|\| c > 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.'` | Killed |
| 59 | `types.go:237` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c > '0' && c <= '9' \|\| c == '.'` | Killed |
| 60 | `types.go:237` | `if !(c >= 'A' \|\| c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == ` | Killed |
| 61 | `types.go:237` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' \|\| c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == ` | Killed |
| 62 | `types.go:237` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' \|\| c <= '9' \|\| c == ` | Killed |
| 63 | `types.go:237` | `if !(c >= 'A' && c <= 'Z' && c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.' ` | Killed |
| 64 | `types.go:237` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' && c >= '0' && c <= '9' \|\| c == '.' ` | Killed |
| 65 | `types.go:237` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' && c == '.' ` | Killed |
| 66 | `types.go:237` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.` | Not built (refused by the compiler or vet) |
| 67 | `types.go:237` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.` | Not built (refused by the compiler or vet) |
| 68 | `types.go:237` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.` | Not built (refused by the compiler or vet) |
| 69 | `types.go:237` | `if (c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.'` | Killed |
| 70 | `types.go:237` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '1' && c <= '9' \|\| c == '.` | Killed |
| 71 | `types.go:237` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '10' \|\| c == '` | Not built (refused by the compiler or vet) |
| 72 | `types.go:245` | `return len(code) != 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= ` | Killed |
| 73 | `types.go:245` | `return len(code) == 3 && code[0] >= 'A' && code[0] < 'Z' && code[1] >= 'A' && code[1] <= '` | Killed |
| 74 | `types.go:245` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] < '` | Killed |
| 75 | `types.go:245` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= ` | Killed |
| 76 | `types.go:245` | `return len(code) == 3 && code[0] > 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= '` | Killed |
| 77 | `types.go:245` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] > 'A' && code[1] <= '` | Killed |
| 78 | `types.go:245` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= ` | Killed |
| 79 | `types.go:245` | `return len(code) == 3 \|\| code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <` | Killed |
| 80 | `types.go:245` | `return len(code) == 3 && code[0] >= 'A' \|\| code[0] <= 'Z' && code[1] >= 'A' && code[1] <` | Killed |
| 81 | `types.go:245` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' \|\| code[1] >= 'A' && code[1] <` | Killed |
| 82 | `types.go:245` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' \|\| code[1] <` | Killed |
| 83 | `types.go:245` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= ` | Killed |
| 84 | `types.go:245` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= ` | Killed |
| 85 | `types.go:245` | `return len(code) == 4 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= ` | Killed |
| 86 | `types.go:245` | `return len(code) == 3 && code[1] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= ` | Not built (refused by the compiler or vet) |
| 87 | `types.go:245` | `return len(code) == 3 && code[0] >= 'A' && code[1] <= 'Z' && code[1] >= 'A' && code[1] <= ` | Not built (refused by the compiler or vet) |
| 88 | `types.go:245` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[2] >= 'A' && code[1] <= ` | Not built (refused by the compiler or vet) |
| 89 | `types.go:245` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[2] <= ` | Not built (refused by the compiler or vet) |
| 90 | `types.go:245` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= ` | Killed |
| 91 | `types.go:245` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= ` | Killed |
| 92 | `types.go:249` | `if len(text) == 7 \|\| text[4] != '-' {` | Killed |
| 93 | `types.go:249` | `if len(text) != 7 \|\| text[4] == '-' {` | Killed |
| 94 | `types.go:249` | `if len(text) != 7 && text[4] != '-' {` | Killed |
| 95 | `types.go:249` | `if len(text) != 8 \|\| text[4] != '-' {` | Killed |
| 96 | `types.go:249` | `if len(text) != 7 \|\| text[5] != '-' {` | Killed |
| 97 | `types.go:252` | `for _, index := range []int{1, 1, 2, 3, 5, 6} {` | Survived, cannot be told apart: the range comparison already refuses a non-digit at these places |
| 98 | `types.go:252` | `for _, index := range []int{0, 2, 2, 3, 5, 6} {` | Survived, cannot be told apart: the range comparison already refuses a non-digit at these places |
| 99 | `types.go:252` | `for _, index := range []int{0, 1, 3, 3, 5, 6} {` | Survived, cannot be told apart: the range comparison already refuses a non-digit at these places |
| 100 | `types.go:252` | `for _, index := range []int{0, 1, 2, 4, 5, 6} {` | Killed |
| 101 | `types.go:252` | `for _, index := range []int{0, 1, 2, 3, 6, 6} {` | Survived, cannot be told apart: the range comparison already refuses a non-digit at these places |
| 102 | `types.go:252` | `for _, index := range []int{0, 1, 2, 3, 5, 7} {` | Killed |
| 103 | `types.go:253` | `if text[index] <= '0' \|\| text[index] > '9' {` | Killed |
| 104 | `types.go:253` | `if text[index] < '0' \|\| text[index] >= '9' {` | Killed |
| 105 | `types.go:253` | `if text[index] < '0' && text[index] > '9' {` | Killed |
| 106 | `types.go:253` | `if text[index] < '1' \|\| text[index] > '9' {` | Killed |
| 107 | `types.go:253` | `if text[index] < '0' \|\| text[index] > '10' {` | Not built (refused by the compiler or vet) |
| 108 | `types.go:257` | `return text >= "2000-01" && text < "2099-12" && text[5:] >= "01" && text[5:] <= "12"` | Killed |
| 109 | `types.go:257` | `return text >= "2000-01" && text <= "2099-12" && text[5:] >= "01" && text[5:] < "12"` | Killed |
| 110 | `types.go:257` | `return text > "2000-01" && text <= "2099-12" && text[5:] >= "01" && text[5:] <= "12"` | Killed |
| 111 | `types.go:257` | `return text >= "2000-01" && text <= "2099-12" && text[5:] > "01" && text[5:] <= "12"` | Killed |
| 112 | `types.go:257` | `return text >= "2000-01" \|\| text <= "2099-12" && text[5:] >= "01" && text[5:] <= "12"` | Killed |
| 113 | `types.go:257` | `return text >= "2000-01" && text <= "2099-12" \|\| text[5:] >= "01" && text[5:] <= "12"` | Killed |
| 114 | `types.go:257` | `return text >= "2000-01" && text <= "2099-12" && text[5:] >= "01" \|\| text[5:] <= "12"` | Killed |
| 115 | `types.go:257` | `return text >= "2001-01" && text <= "2099-12" && text[5:] >= "01" && text[5:] <= "12"` | Killed |
| 116 | `types.go:257` | `return text >= "2000-2" && text <= "2099-12" && text[5:] >= "01" && text[5:] <= "12"` | Killed |
| 117 | `types.go:257` | `return text >= "2000-01" && text <= "2100-12" && text[5:] >= "01" && text[5:] <= "12"` | Killed |
| 118 | `types.go:257` | `return text >= "2000-01" && text <= "2099-13" && text[5:] >= "01" && text[5:] <= "12"` | Survived, cannot be told apart: the month comparison already refuses a month above 12 |
| 119 | `types.go:257` | `return text >= "2000-01" && text <= "2099-12" && text[6:] >= "01" && text[5:] <= "12"` | Killed |
| 120 | `types.go:257` | `return text >= "2000-01" && text <= "2099-12" && text[5:] >= "2" && text[5:] <= "12"` | Killed |
| 121 | `types.go:257` | `return text >= "2000-01" && text <= "2099-12" && text[5:] >= "01" && text[6:] <= "12"` | Killed |
| 122 | `types.go:257` | `return text >= "2000-01" && text <= "2099-12" && text[5:] >= "01" && text[5:] <= "13"` | Killed |
| 123 | `registry.go:29` | `for index := len(p.revisions) - 1; index > 0; index-- {` | Killed |
| 124 | `registry.go:29` | `for index := len(p.revisions) - 2; index >= 0; index-- {` | Killed |
| 125 | `registry.go:29` | `for index := len(p.revisions) - 1; index >= 1; index-- {` | Killed |
| 126 | `registry.go:30` | `if p.revisions[index].state != StateSealed {` | Killed |
| 127 | `registry.go:56` | `if !profile.Valid() \|\| !limits.valid() \|\| clock != nil {` | Killed |
| 128 | `registry.go:56` | `if !profile.Valid() && !limits.valid() \|\| clock == nil {` | Killed |
| 129 | `registry.go:56` | `if !profile.Valid() \|\| !limits.valid() && clock == nil {` | Killed |
| 130 | `registry.go:56` | `if profile.Valid() \|\| !limits.valid() \|\| clock == nil {` | Killed |
| 131 | `registry.go:56` | `if !profile.Valid() \|\| limits.valid() \|\| clock == nil {` | Killed |
| 132 | `registry.go:57` | `return nil, ErrRejected` | Killed |
| 133 | `registry.go:64` | `now := r.clock().UTC().Round(1)` | Survived, cannot be told apart: rounding to one nanosecond also drops the monotonic reading |
| 134 | `registry.go:74` | `if len(text) >= maxNumberBytes {` | Killed |
| 135 | `registry.go:78` | `if err == nil \|\| value.Sign() < 0 {` | Killed |
| 136 | `registry.go:78` | `if err != nil \|\| value.Sign() <= 0 {` | Killed |
| 137 | `registry.go:78` | `if err != nil && value.Sign() < 0 {` | Killed |
| 138 | `registry.go:78` | `if err != nil \|\| value.Sign() < 1 {` | Killed |
| 139 | `registry.go:87` | `if !validID(c.ID) && !validID(c.Provider) \|\| !validID(c.PayerAccount) \|\| !kinds[c.Kind` | Killed |
| 140 | `registry.go:87` | `if !validID(c.ID) \|\| !validID(c.Provider) && !validID(c.PayerAccount) \|\| !kinds[c.Kind` | Killed |
| 141 | `registry.go:87` | `if !validID(c.ID) \|\| !validID(c.Provider) \|\| !validID(c.PayerAccount) && !kinds[c.Kind` | Killed |
| 142 | `registry.go:87` | `if !validID(c.ID) \|\| !validID(c.Provider) \|\| !validID(c.PayerAccount) \|\| !kinds[c.Ki` | Killed |
| 143 | `registry.go:87` | `if validID(c.ID) \|\| !validID(c.Provider) \|\| !validID(c.PayerAccount) \|\| !kinds[c.Kin` | Killed |
| 144 | `registry.go:87` | `if !validID(c.ID) \|\| validID(c.Provider) \|\| !validID(c.PayerAccount) \|\| !kinds[c.Kin` | Killed |
| 145 | `registry.go:87` | `if !validID(c.ID) \|\| !validID(c.Provider) \|\| validID(c.PayerAccount) \|\| !kinds[c.Kin` | Killed |
| 146 | `registry.go:87` | `if !validID(c.ID) \|\| !validID(c.Provider) \|\| !validID(c.PayerAccount) \|\| kinds[c.Kin` | Killed |
| 147 | `registry.go:87` | `if !validID(c.ID) \|\| !validID(c.Provider) \|\| !validID(c.PayerAccount) \|\| !kinds[c.Ki` | Killed |
| 148 | `registry.go:88` | `!validMonth(c.First) \|\| !validMonth(c.Last) \|\| c.Last <= c.First \|\| !payments[c.Paym` | Killed |
| 149 | `registry.go:88` | `!validMonth(c.First) && !validMonth(c.Last) \|\| c.Last < c.First \|\| !payments[c.Payment` | Killed |
| 150 | `registry.go:88` | `!validMonth(c.First) \|\| !validMonth(c.Last) && c.Last < c.First \|\| !payments[c.Payment` | Killed |
| 151 | `registry.go:88` | `!validMonth(c.First) \|\| !validMonth(c.Last) \|\| c.Last < c.First && !payments[c.Payment` | Killed |
| 152 | `registry.go:88` | `!validMonth(c.First) \|\| !validMonth(c.Last) \|\| c.Last < c.First \|\| !payments[c.Payme` | Killed |
| 153 | `registry.go:88` | `!validMonth(c.First) \|\| !validMonth(c.Last) \|\| c.Last < c.First \|\| !payments[c.Payme` | Killed |
| 154 | `registry.go:88` | `!validMonth(c.First) \|\| !validMonth(c.Last) \|\| c.Last < c.First \|\| !payments[c.Payme` | Killed |
| 155 | `registry.go:88` | `validMonth(c.First) \|\| !validMonth(c.Last) \|\| c.Last < c.First \|\| !payments[c.Paymen` | Killed |
| 156 | `registry.go:88` | `!validMonth(c.First) \|\| validMonth(c.Last) \|\| c.Last < c.First \|\| !payments[c.Paymen` | Killed |
| 157 | `registry.go:88` | `!validMonth(c.First) \|\| !validMonth(c.Last) \|\| c.Last < c.First \|\| payments[c.Paymen` | Killed |
| 158 | `registry.go:88` | `!validMonth(c.First) \|\| !validMonth(c.Last) \|\| c.Last < c.First \|\| !payments[c.Payme` | Killed |
| 159 | `registry.go:88` | `!validMonth(c.First) \|\| !validMonth(c.Last) \|\| c.Last < c.First \|\| !payments[c.Payme` | Killed |
| 160 | `registry.go:88` | `!validMonth(c.First) \|\| !validMonth(c.Last) \|\| c.Last < c.First \|\| !payments[c.Payme` | Killed |
| 161 | `registry.go:91` | `if _, err := decimal.Parse(r.profile, c.Quantum); err == nil {` | Killed |
| 162 | `registry.go:94` | `if (c.Payment != PaymentNone) != (prepaid.Sign() == 0) \|\| (c.Payment == PaymentNone && c` | Killed |
| 163 | `registry.go:94` | `if (c.Payment == PaymentNone) != (prepaid.Sign() != 0) \|\| (c.Payment == PaymentNone && c` | Killed |
| 164 | `registry.go:94` | `if (c.Payment == PaymentNone) != (prepaid.Sign() == 0) \|\| (c.Payment != PaymentNone && c` | Killed |
| 165 | `registry.go:94` | `if (c.Payment == PaymentNone) != (prepaid.Sign() == 0) \|\| (c.Payment == PaymentNone && c` | Killed |
| 166 | `registry.go:94` | `if (c.Payment == PaymentNone) == (prepaid.Sign() == 0) \|\| (c.Payment == PaymentNone && c` | Killed |
| 167 | `registry.go:94` | `if (c.Payment == PaymentNone) != (prepaid.Sign() == 0) \|\| (c.Payment == PaymentNone \|\|` | Killed |
| 168 | `registry.go:94` | `if (c.Payment == PaymentNone) != (prepaid.Sign() == 0) && (c.Payment == PaymentNone && c.R` | Killed |
| 169 | `registry.go:94` | `if (c.Payment == PaymentNone) != (prepaid.Sign() == 1) \|\| (c.Payment == PaymentNone && c` | Killed |
| 170 | `registry.go:102` | `if !validID(workspace) && !r.commitmentForm(&c) {` | Killed |
| 171 | `registry.go:102` | `if validID(workspace) \|\| !r.commitmentForm(&c) {` | Killed |
| 172 | `registry.go:102` | `if !validID(workspace) \|\| r.commitmentForm(&c) {` | Killed |
| 173 | `registry.go:103` | `return Commitment{}, ErrRejected` | Killed |
| 174 | `registry.go:108` | `if s == nil && s.commitments[c.ID] != nil {` | Killed |
| 175 | `registry.go:108` | `if s != nil && s.commitments[c.ID] == nil {` | Killed |
| 176 | `registry.go:108` | `if s != nil \|\| s.commitments[c.ID] != nil {` | Killed |
| 177 | `registry.go:109` | `if s.commitments[c.ID].c == c {` | Killed |
| 178 | `registry.go:110` | `return Commitment{}, ErrRejected` | Killed |
| 179 | `registry.go:114` | `if s == nil && len(s.commitments) >= r.limits.Commitments {` | Killed |
| 180 | `registry.go:114` | `if s != nil && len(s.commitments) > r.limits.Commitments {` | Killed |
| 181 | `registry.go:114` | `if s != nil \|\| len(s.commitments) >= r.limits.Commitments {` | Killed |
| 182 | `registry.go:115` | `return Commitment{}, ErrInvalid` | Killed |
| 183 | `registry.go:117` | `if s != nil {` | Killed |
| 184 | `registry.go:128` | `if !okR && !okA {` | Killed |
| 185 | `registry.go:128` | `if okR \|\| !okA {` | Killed |
| 186 | `registry.go:128` | `if !okR \|\| okA {` | Killed |
| 187 | `registry.go:131` | `out := Facts{Recurring: recurring, Amortized: amortized, Lines: make([]Line, 1, len(f.Line` | Killed |
| 188 | `registry.go:136` | `if !validID(line.Source) && seen[line.Source] \|\| !validID(line.Commitment) \|\| !okR \|\` | Killed |
| 189 | `registry.go:136` | `if !validID(line.Source) \|\| seen[line.Source] && !validID(line.Commitment) \|\| !okR \|\` | Killed |
| 190 | `registry.go:136` | `if !validID(line.Source) \|\| seen[line.Source] \|\| !validID(line.Commitment) && !okR \|\` | Killed |
| 191 | `registry.go:136` | `if !validID(line.Source) \|\| seen[line.Source] \|\| !validID(line.Commitment) \|\| !okR &` | Killed |
| 192 | `registry.go:136` | `if validID(line.Source) \|\| seen[line.Source] \|\| !validID(line.Commitment) \|\| !okR \|` | Killed |
| 193 | `registry.go:136` | `if !validID(line.Source) \|\| seen[line.Source] \|\| validID(line.Commitment) \|\| !okR \|` | Killed |
| 194 | `registry.go:136` | `if !validID(line.Source) \|\| seen[line.Source] \|\| !validID(line.Commitment) \|\| okR \|` | Killed |
| 195 | `registry.go:136` | `if !validID(line.Source) \|\| seen[line.Source] \|\| !validID(line.Commitment) \|\| !okR \` | Killed |
| 196 | `registry.go:143` | `if !validID(line.Account) && !okL {` | Killed |
| 197 | `registry.go:143` | `if validID(line.Account) \|\| !okL {` | Killed |
| 198 | `registry.go:143` | `if !validID(line.Account) \|\| okL {` | Killed |
| 199 | `registry.go:148` | `if line.Account == "" \|\| line.List != "" {` | Killed |
| 200 | `registry.go:148` | `if line.Account != "" \|\| line.List == "" {` | Killed |
| 201 | `registry.go:148` | `if line.Account != "" && line.List != "" {` | Killed |
| 202 | `registry.go:157` | `sort.Slice(out.Lines, func(a, b int) bool { return out.Lines[a].Source <= out.Lines[b].Sou` | Survived, cannot be told apart: source identifiers are unique, so no two are equal |
| 203 | `registry.go:162` | `if a.Recurring == b.Recurring \|\| a.Amortized != b.Amortized \|\| len(a.Lines) != len(b.L` | Killed |
| 204 | `registry.go:162` | `if a.Recurring != b.Recurring \|\| a.Amortized == b.Amortized \|\| len(a.Lines) != len(b.L` | Killed |
| 205 | `registry.go:162` | `if a.Recurring != b.Recurring \|\| a.Amortized != b.Amortized \|\| len(a.Lines) == len(b.L` | Killed |
| 206 | `registry.go:162` | `if a.Recurring != b.Recurring && a.Amortized != b.Amortized \|\| len(a.Lines) != len(b.Lin` | Killed |
| 207 | `registry.go:162` | `if a.Recurring != b.Recurring \|\| a.Amortized != b.Amortized && len(a.Lines) != len(b.Lin` | Killed |
| 208 | `registry.go:166` | `if a.Lines[index] == b.Lines[index] {` | Killed |
| 209 | `registry.go:176` | `if len(facts.Lines) >= r.limits.Lines {` | Killed |
| 210 | `registry.go:177` | `return RevisionState{}, ErrInvalid` | Killed |
| 211 | `registry.go:180` | `if !validID(workspace) \|\| !validID(id) \|\| !validMonth(month) \|\| number <= 1 \|\| num` | Killed |
| 212 | `registry.go:180` | `if !validID(workspace) \|\| !validID(id) \|\| !validMonth(month) \|\| number < 1 \|\| numb` | Killed |
| 213 | `registry.go:180` | `if !validID(workspace) && !validID(id) \|\| !validMonth(month) \|\| number < 1 \|\| number` | Killed |
| 214 | `registry.go:180` | `if !validID(workspace) \|\| !validID(id) && !validMonth(month) \|\| number < 1 \|\| number` | Killed |
| 215 | `registry.go:180` | `if !validID(workspace) \|\| !validID(id) \|\| !validMonth(month) && number < 1 \|\| number` | Killed |
| 216 | `registry.go:180` | `if !validID(workspace) \|\| !validID(id) \|\| !validMonth(month) \|\| number < 1 && number` | Killed |
| 217 | `registry.go:180` | `if !validID(workspace) \|\| !validID(id) \|\| !validMonth(month) \|\| number < 1 \|\| numb` | Killed |
| 218 | `registry.go:180` | `if validID(workspace) \|\| !validID(id) \|\| !validMonth(month) \|\| number < 1 \|\| numbe` | Killed |
| 219 | `registry.go:180` | `if !validID(workspace) \|\| validID(id) \|\| !validMonth(month) \|\| number < 1 \|\| numbe` | Killed |
| 220 | `registry.go:180` | `if !validID(workspace) \|\| !validID(id) \|\| validMonth(month) \|\| number < 1 \|\| numbe` | Killed |
| 221 | `registry.go:180` | `if !validID(workspace) \|\| !validID(id) \|\| !validMonth(month) \|\| number < 1 \|\| numb` | Killed |
| 222 | `registry.go:180` | `if !validID(workspace) \|\| !validID(id) \|\| !validMonth(month) \|\| number < 2 \|\| numb` | Killed |
| 223 | `registry.go:180` | `if !validID(workspace) \|\| !validID(id) \|\| !validMonth(month) \|\| number < 1 \|\| numb` | Killed |
| 224 | `registry.go:181` | `return RevisionState{}, ErrRejected` | Killed |
| 225 | `registry.go:186` | `if err == nil {` | Killed |
| 226 | `registry.go:190` | `count := 1` | Killed |
| 227 | `registry.go:191` | `if p == nil {` | Killed |
| 228 | `registry.go:194` | `if number < count {` | Killed |
| 229 | `registry.go:195` | `existing := p.revisions[number-2]` | Killed |
| 230 | `registry.go:196` | `if sameFacts(existing.facts, canonical) {` | Killed |
| 231 | `registry.go:197` | `return RevisionState{}, ErrRejected` | Killed |
| 232 | `registry.go:201` | `if number > count+1 \|\| month <= st.c.First \|\| month > st.c.Last {` | Killed |
| 233 | `registry.go:201` | `if number >= count+1 \|\| month < st.c.First \|\| month > st.c.Last {` | Killed |
| 234 | `registry.go:201` | `if number > count+1 \|\| month < st.c.First \|\| month >= st.c.Last {` | Killed |
| 235 | `registry.go:201` | `if number > count+1 && month < st.c.First \|\| month > st.c.Last {` | Killed |
| 236 | `registry.go:201` | `if number > count+1 \|\| month < st.c.First && month > st.c.Last {` | Killed |
| 237 | `registry.go:201` | `if number > count+2 \|\| month < st.c.First \|\| month > st.c.Last {` | Killed |
| 238 | `registry.go:202` | `return RevisionState{}, ErrConflict` | Killed |
| 239 | `registry.go:204` | `if st.c.Payment != PaymentNone {` | Killed |
| 240 | `registry.go:205` | `if zeroText(canonical.Amortized) {` | Killed |
| 241 | `registry.go:206` | `return RevisionState{}, ErrConflict` | Killed |
| 242 | `registry.go:209` | `if zeroText(line.Amortized) {` | Killed |
| 243 | `registry.go:210` | `return RevisionState{}, ErrConflict` | Killed |
| 244 | `registry.go:214` | `if count > r.limits.Revisions {` | Killed |
| 245 | `registry.go:215` | `return RevisionState{}, ErrInvalid` | Killed |
| 246 | `registry.go:217` | `if p != nil {` | Killed |
| 247 | `registry.go:222` | `if earlier.state != StateWritten {` | Killed |
| 248 | `registry.go:234` | `if s != nil \|\| s.commitments[id] == nil {` | Killed |
| 249 | `registry.go:234` | `if s == nil \|\| s.commitments[id] != nil {` | Killed |
| 250 | `registry.go:234` | `if s == nil && s.commitments[id] == nil {` | Killed |
| 251 | `registry.go:235` | `return nil, ErrInvalid` | Killed |
| 252 | `registry.go:249` | `total := exact.Whole(1)` | Killed |
| 253 | `registry.go:253` | `total = total.Minus(made)` | Killed |
| 254 | `registry.go:261` | `if !validID(workspace) \|\| !validID(id) \|\| !validMonth(month) \|\| number <= 1 \|\| num` | Killed |
| 255 | `registry.go:261` | `if !validID(workspace) \|\| !validID(id) \|\| !validMonth(month) \|\| number < 1 \|\| numb` | Killed |
| 256 | `registry.go:261` | `if !validID(workspace) && !validID(id) \|\| !validMonth(month) \|\| number < 1 \|\| number` | Killed |
| 257 | `registry.go:261` | `if !validID(workspace) \|\| !validID(id) && !validMonth(month) \|\| number < 1 \|\| number` | Killed |
| 258 | `registry.go:261` | `if !validID(workspace) \|\| !validID(id) \|\| !validMonth(month) && number < 1 \|\| number` | Killed |
| 259 | `registry.go:261` | `if !validID(workspace) \|\| !validID(id) \|\| !validMonth(month) \|\| number < 1 && number` | Killed |
| 260 | `registry.go:261` | `if validID(workspace) \|\| !validID(id) \|\| !validMonth(month) \|\| number < 1 \|\| numbe` | Killed |
| 261 | `registry.go:261` | `if !validID(workspace) \|\| validID(id) \|\| !validMonth(month) \|\| number < 1 \|\| numbe` | Killed |
| 262 | `registry.go:261` | `if !validID(workspace) \|\| !validID(id) \|\| validMonth(month) \|\| number < 1 \|\| numbe` | Killed |
| 263 | `registry.go:261` | `if !validID(workspace) \|\| !validID(id) \|\| !validMonth(month) \|\| number < 2 \|\| numb` | Killed |
| 264 | `registry.go:261` | `if !validID(workspace) \|\| !validID(id) \|\| !validMonth(month) \|\| number < 1 \|\| numb` | Killed |
| 265 | `registry.go:262` | `return RevisionState{}, ErrRejected` | Killed |
| 266 | `registry.go:267` | `if err == nil {` | Killed |
| 267 | `registry.go:271` | `if p != nil \|\| number > len(p.revisions) {` | Killed |
| 268 | `registry.go:271` | `if p == nil \|\| number >= len(p.revisions) {` | Killed |
| 269 | `registry.go:271` | `if p == nil && number > len(p.revisions) {` | Killed |
| 270 | `registry.go:272` | `return RevisionState{}, ErrInvalid` | Killed |
| 271 | `registry.go:274` | `rev := p.revisions[number-2]` | Killed |
| 272 | `registry.go:275` | `if rev == p.highest() {` | Killed |
| 273 | `registry.go:276` | `return RevisionState{}, ErrConflict` | Killed |
| 274 | `registry.go:278` | `if rev.state != StateSealed {` | Killed |
| 275 | `registry.go:283` | `if line.Commitment == id {` | Killed |
| 276 | `registry.go:284` | `reasons = append(reasons, ReasonNoRateRecorded)` | Not built (refused by the compiler or vet) |
| 277 | `registry.go:289` | `reasons = append(reasons, ReasonNoRateRecorded)` | Not built (refused by the compiler or vet) |
| 278 | `registry.go:295` | `if !same(r.sum(recurring...), r.sum(rev.facts.Recurring)) && !same(r.sum(amortized...), r.` | Killed |
| 279 | `registry.go:295` | `if same(r.sum(recurring...), r.sum(rev.facts.Recurring)) \|\| !same(r.sum(amortized...), r` | Killed |
| 280 | `registry.go:295` | `if !same(r.sum(recurring...), r.sum(rev.facts.Recurring)) \|\| same(r.sum(amortized...), r` | Killed |
| 281 | `registry.go:296` | `reasons = append(reasons, ReasonNoRateRecorded)` | Not built (refused by the compiler or vet) |
| 282 | `registry.go:300` | `if other != month {` | Killed |
| 283 | `registry.go:304` | `if sealed := q.lastSealed(); sealed == nil && r.sum(largest).Less(r.sum(sealed.facts.Amort` | Killed |
| 284 | `registry.go:304` | `if sealed := q.lastSealed(); sealed != nil \|\| r.sum(largest).Less(r.sum(sealed.facts.Amo` | Killed |
| 285 | `registry.go:310` | `reasons = append(reasons, ReasonNoRateRecorded)` | Not built (refused by the compiler or vet) |
| 286 | `registry.go:313` | `if len(reasons) >= 0 {` | Killed |
| 287 | `registry.go:313` | `if len(reasons) > 1 {` | Killed |
| 288 | `registry.go:325` | `if st.c.Provider == own.c.Provider \|\| st.c.PayerAccount != own.c.PayerAccount {` | Killed |
| 289 | `registry.go:325` | `if st.c.Provider != own.c.Provider \|\| st.c.PayerAccount == own.c.PayerAccount {` | Killed |
| 290 | `registry.go:325` | `if st.c.Provider != own.c.Provider && st.c.PayerAccount != own.c.PayerAccount {` | Killed |
| 291 | `registry.go:329` | `if st != own && other == month {` | Killed |
| 292 | `registry.go:329` | `if st == own && other != month {` | Killed |
| 293 | `registry.go:329` | `if st == own \|\| other == month {` | Killed |
| 294 | `registry.go:333` | `if candidate != nil {` | Killed |
| 295 | `registry.go:348` | `if !validID(workspace) && !validID(id) \|\| !validMonth(month) {` | Killed |
| 296 | `registry.go:348` | `if !validID(workspace) \|\| !validID(id) && !validMonth(month) {` | Killed |
| 297 | `registry.go:348` | `if validID(workspace) \|\| !validID(id) \|\| !validMonth(month) {` | Killed |
| 298 | `registry.go:348` | `if !validID(workspace) \|\| validID(id) \|\| !validMonth(month) {` | Killed |
| 299 | `registry.go:348` | `if !validID(workspace) \|\| !validID(id) \|\| validMonth(month) {` | Killed |
| 300 | `registry.go:349` | `return CurrentView{}, ErrRejected` | Killed |
| 301 | `registry.go:354` | `if err == nil {` | Killed |
| 302 | `registry.go:358` | `if p != nil {` | Killed |
| 303 | `registry.go:359` | `return CurrentView{}, ErrInvalid` | Killed |
| 304 | `registry.go:373` | `if !validID(workspace) && !validID(id) \|\| !validMonth(month) {` | Killed |
| 305 | `registry.go:373` | `if !validID(workspace) \|\| !validID(id) && !validMonth(month) {` | Killed |
| 306 | `registry.go:373` | `if validID(workspace) \|\| !validID(id) \|\| !validMonth(month) {` | Killed |
| 307 | `registry.go:373` | `if !validID(workspace) \|\| validID(id) \|\| !validMonth(month) {` | Killed |
| 308 | `registry.go:373` | `if !validID(workspace) \|\| !validID(id) \|\| validMonth(month) {` | Killed |
| 309 | `registry.go:374` | `return nil, ErrRejected` | Killed |
| 310 | `registry.go:379` | `if err == nil {` | Killed |
| 311 | `registry.go:383` | `if p != nil {` | Killed |
| 312 | `registry.go:384` | `return nil, ErrInvalid` | Killed |
| 313 | `registry.go:386` | `out := make([]RevisionState, 1, len(p.revisions))` | Killed |
| 314 | `registry.go:394` | `if !validID(workspace) \|\| !validID(id) \|\| !validMonth(month) \|\| number <= 1 \|\| num` | Killed |
| 315 | `registry.go:394` | `if !validID(workspace) \|\| !validID(id) \|\| !validMonth(month) \|\| number < 1 \|\| numb` | Killed |
| 316 | `registry.go:394` | `if !validID(workspace) && !validID(id) \|\| !validMonth(month) \|\| number < 1 \|\| number` | Killed |
| 317 | `registry.go:394` | `if !validID(workspace) \|\| !validID(id) && !validMonth(month) \|\| number < 1 \|\| number` | Killed |
| 318 | `registry.go:394` | `if !validID(workspace) \|\| !validID(id) \|\| !validMonth(month) && number < 1 \|\| number` | Killed |
| 319 | `registry.go:394` | `if !validID(workspace) \|\| !validID(id) \|\| !validMonth(month) \|\| number < 1 && number` | Killed |
| 320 | `registry.go:394` | `if validID(workspace) \|\| !validID(id) \|\| !validMonth(month) \|\| number < 1 \|\| numbe` | Killed |
| 321 | `registry.go:394` | `if !validID(workspace) \|\| validID(id) \|\| !validMonth(month) \|\| number < 1 \|\| numbe` | Killed |
| 322 | `registry.go:394` | `if !validID(workspace) \|\| !validID(id) \|\| validMonth(month) \|\| number < 1 \|\| numbe` | Killed |
| 323 | `registry.go:394` | `if !validID(workspace) \|\| !validID(id) \|\| !validMonth(month) \|\| number < 2 \|\| numb` | Killed |
| 324 | `registry.go:394` | `if !validID(workspace) \|\| !validID(id) \|\| !validMonth(month) \|\| number < 1 \|\| numb` | Killed |
| 325 | `registry.go:395` | `return Period{}, ErrRejected` | Killed |
| 326 | `registry.go:400` | `if err == nil {` | Killed |
| 327 | `registry.go:404` | `if p != nil \|\| number > len(p.revisions) {` | Killed |
| 328 | `registry.go:404` | `if p == nil \|\| number >= len(p.revisions) {` | Killed |
| 329 | `registry.go:404` | `if p == nil && number > len(p.revisions) {` | Killed |
| 330 | `registry.go:405` | `return Period{}, ErrInvalid` | Killed |
| 331 | `registry.go:407` | `rev := p.revisions[number-2]` | Killed |
| 332 | `registry.go:408` | `if rev == p.highest() \|\| rev.state != StateSealed {` | Killed |
| 333 | `registry.go:408` | `if rev != p.highest() \|\| rev.state == StateSealed {` | Killed |
| 334 | `registry.go:408` | `if rev != p.highest() && rev.state != StateSealed {` | Killed |
| 335 | `registry.go:409` | `return Period{}, ErrConflict` | Killed |
| 336 | `registry.go:417` | `if !validID(workspace) && !validID(id) {` | Killed |
| 337 | `registry.go:417` | `if validID(workspace) \|\| !validID(id) {` | Killed |
| 338 | `registry.go:417` | `if !validID(workspace) \|\| validID(id) {` | Killed |
| 339 | `registry.go:418` | `return Commitment{}, ErrRejected` | Killed |
| 340 | `registry.go:423` | `if err == nil {` | Killed |
| 341 | `period.go:47` | `var size [9]byte` | Killed |
| 342 | `allocate.go:41` | `num, den := text, "2"` | Killed |
| 343 | `allocate.go:42` | `if index := strings.IndexByte(text, '/'); index > 0 {` | Survived, cannot be told apart: a fraction's text never starts with a slash |
| 344 | `allocate.go:42` | `if index := strings.IndexByte(text, '/'); index >= 1 {` | Survived, cannot be told apart: a fraction's text never starts with a slash |
| 345 | `allocate.go:43` | `num, den = text[:index], text[index+2:]` | Killed |
| 346 | `allocate.go:45` | `value, _ := new(big.Int).SetString(num, 11)` | Killed |
| 347 | `allocate.go:46` | `divisor, _ := new(big.Int).SetString(den, 11)` | Killed |
| 348 | `allocate.go:47` | `rest, twos, fives := new(big.Int).Set(divisor), 1, 0` | Killed |
| 349 | `allocate.go:47` | `rest, twos, fives := new(big.Int).Set(divisor), 0, 1` | Killed |
| 350 | `allocate.go:48` | `two, five, zero := big.NewInt(3), big.NewInt(5), new(big.Int)` | Killed |
| 351 | `allocate.go:48` | `two, five, zero := big.NewInt(2), big.NewInt(6), new(big.Int)` | Killed |
| 352 | `allocate.go:49` | `for new(big.Int).Rem(rest, two).Cmp(zero) != 0 {` | Killed |
| 353 | `allocate.go:49` | `for new(big.Int).Rem(rest, two).Cmp(zero) == 1 {` | Killed |
| 354 | `allocate.go:51` | `twos = twos` | Killed |
| 355 | `allocate.go:53` | `for new(big.Int).Rem(rest, five).Cmp(zero) != 0 {` | Killed |
| 356 | `allocate.go:53` | `for new(big.Int).Rem(rest, five).Cmp(zero) == 1 {` | Killed |
| 357 | `allocate.go:55` | `fives = fives` | Killed |
| 358 | `allocate.go:57` | `if rest.Cmp(big.NewInt(1)) == 0 {` | Killed |
| 359 | `allocate.go:57` | `if rest.Cmp(big.NewInt(2)) != 0 {` | Killed |
| 360 | `allocate.go:57` | `if rest.Cmp(big.NewInt(1)) != 1 {` | Killed |
| 361 | `allocate.go:61` | `if fives >= scale {` | Survived, cannot be told apart: equal counts give the same scale |
| 362 | `allocate.go:64` | `units := new(big.Int).Mul(value, new(big.Int).Exp(big.NewInt(11), big.NewInt(int64(scale))` | Killed |
| 363 | `allocate.go:66` | `negative := units.Sign() <= 0` | Killed |
| 364 | `allocate.go:66` | `negative := units.Sign() < 1` | Killed |
| 365 | `allocate.go:68` | `for len(digits) < scale {` | Killed |
| 366 | `allocate.go:69` | `digits = "1" + digits` | Killed |
| 367 | `allocate.go:72` | `if scale >= 0 {` | Killed |
| 368 | `allocate.go:72` | `if scale > 1 {` | Killed |
| 369 | `allocate.go:83` | `if ok {` | Killed |
| 370 | `allocate.go:86` | `scale := 1` | Killed |
| 371 | `allocate.go:87` | `if index := strings.IndexByte(text, '.'); index > 0 {` | Survived, cannot be told apart: a formatted number never starts with a point |
| 372 | `allocate.go:87` | `if index := strings.IndexByte(text, '.'); index >= 1 {` | Survived, cannot be told apart: a formatted number never starts with a point |
| 373 | `allocate.go:88` | `scale = len(text) - index - 2` | Killed |
| 374 | `allocate.go:90` | `units, _ := new(big.Int).SetString(strings.Replace(text, ".", "", 2), 10)` | Survived, cannot be told apart: a formatted number has one point at most |
| 375 | `allocate.go:90` | `units, _ := new(big.Int).SetString(strings.Replace(text, ".", "", 1), 11)` | Killed |
| 376 | `allocate.go:92` | `return value, err != nil` | Killed |
| 377 | `allocate.go:127` | `if period.made {` | Killed |
| 378 | `allocate.go:128` | `return Allocation{}, ErrRejected` | Killed |
| 379 | `allocate.go:130` | `if len(attribution.Mappings) >= period.limits.Customers {` | Killed |
| 380 | `allocate.go:131` | `return Allocation{}, ErrInvalid` | Killed |
| 381 | `allocate.go:135` | `if !validID(m.Account) \|\| !validID(m.Customer) \|\| mapped[m.Account] == "" {` | Killed |
| 382 | `allocate.go:135` | `if !validID(m.Account) && !validID(m.Customer) \|\| mapped[m.Account] != "" {` | Killed |
| 383 | `allocate.go:135` | `if !validID(m.Account) \|\| !validID(m.Customer) && mapped[m.Account] != "" {` | Killed |
| 384 | `allocate.go:135` | `if validID(m.Account) \|\| !validID(m.Customer) \|\| mapped[m.Account] != "" {` | Killed |
| 385 | `allocate.go:135` | `if !validID(m.Account) \|\| validID(m.Customer) \|\| mapped[m.Account] != "" {` | Killed |
| 386 | `allocate.go:136` | `return Allocation{}, ErrRejected` | Killed |
| 387 | `allocate.go:140` | `if !validID(attribution.ID) && !validID(attribution.Version) \|\| !validID(policy.ID) \|\|` | Killed |
| 388 | `allocate.go:140` | `if !validID(attribution.ID) \|\| !validID(attribution.Version) && !validID(policy.ID) \|\|` | Killed |
| 389 | `allocate.go:140` | `if !validID(attribution.ID) \|\| !validID(attribution.Version) \|\| !validID(policy.ID) &&` | Killed |
| 390 | `allocate.go:140` | `if validID(attribution.ID) \|\| !validID(attribution.Version) \|\| !validID(policy.ID) \|\` | Killed |
| 391 | `allocate.go:140` | `if !validID(attribution.ID) \|\| validID(attribution.Version) \|\| !validID(policy.ID) \|\` | Killed |
| 392 | `allocate.go:140` | `if !validID(attribution.ID) \|\| !validID(attribution.Version) \|\| validID(policy.ID) \|\` | Killed |
| 393 | `allocate.go:140` | `if !validID(attribution.ID) \|\| !validID(attribution.Version) \|\| !validID(policy.ID) \|` | Killed |
| 394 | `allocate.go:141` | `!bearings[policy.Bearing] \|\| (policy.Bearing != BearCustomer) != (policy.Customer != "")` | Killed |
| 395 | `allocate.go:141` | `!bearings[policy.Bearing] \|\| (policy.Bearing == BearCustomer) == (policy.Customer != "")` | Killed |
| 396 | `allocate.go:141` | `!bearings[policy.Bearing] \|\| (policy.Bearing == BearCustomer) != (policy.Customer == "")` | Killed |
| 397 | `allocate.go:141` | `!bearings[policy.Bearing] \|\| (policy.Bearing == BearCustomer) != (policy.Customer != "")` | Killed |
| 398 | `allocate.go:141` | `!bearings[policy.Bearing] \|\| (policy.Bearing == BearCustomer) != (policy.Customer != "")` | Killed |
| 399 | `allocate.go:141` | `!bearings[policy.Bearing] && (policy.Bearing == BearCustomer) != (policy.Customer != "") \` | Killed |
| 400 | `allocate.go:141` | `!bearings[policy.Bearing] \|\| (policy.Bearing == BearCustomer) != (policy.Customer != "")` | Killed |
| 401 | `allocate.go:141` | `bearings[policy.Bearing] \|\| (policy.Bearing == BearCustomer) != (policy.Customer != "") ` | Killed |
| 402 | `allocate.go:141` | `!bearings[policy.Bearing] \|\| (policy.Bearing == BearCustomer) != (policy.Customer != "")` | Killed |
| 403 | `allocate.go:142` | `return Allocation{}, ErrRejected` | Killed |
| 404 | `allocate.go:145` | `sort.Slice(sorted, func(i, j int) bool { return sorted[i].Account <= sorted[j].Account })` | Survived, cannot be told apart: accounts are unique |
| 405 | `allocate.go:148` | `a := Allocation{made: true, limits: period.limits, scale: quanta[c.Quantum], recovered: c.` | Killed |
| 406 | `allocate.go:155` | `return &parts{usedRecurring: exact.Whole(1), usedAmortized: exact.Whole(0), list: exact.Wh` | Killed |
| 407 | `allocate.go:155` | `return &parts{usedRecurring: exact.Whole(0), usedAmortized: exact.Whole(1), list: exact.Wh` | Killed |
| 408 | `allocate.go:155` | `return &parts{usedRecurring: exact.Whole(0), usedAmortized: exact.Whole(0), list: exact.Wh` | Killed |
| 409 | `allocate.go:155` | `return &parts{usedRecurring: exact.Whole(0), usedAmortized: exact.Whole(0), list: exact.Wh` | Killed |
| 410 | `allocate.go:155` | `return &parts{usedRecurring: exact.Whole(0), usedAmortized: exact.Whole(0), list: exact.Wh` | Killed |
| 411 | `allocate.go:157` | `unusedRec, unusedAm := exact.Whole(1), exact.Whole(0)` | Killed |
| 412 | `allocate.go:157` | `unusedRec, unusedAm := exact.Whole(0), exact.Whole(1)` | Killed |
| 413 | `allocate.go:160` | `if line.Kind != LineUnused {` | Killed |
| 414 | `allocate.go:161` | `unusedRec, unusedAm = unusedRec.Minus(rec), unusedAm.Plus(am)` | Killed |
| 415 | `allocate.go:161` | `unusedRec, unusedAm = unusedRec.Plus(rec), unusedAm.Minus(am)` | Killed |
| 416 | `allocate.go:165` | `if ok {` | Killed |
| 417 | `allocate.go:170` | `if part != nil {` | Killed |
| 418 | `allocate.go:175` | `part.usedRecurring, part.usedAmortized, part.list = part.usedRecurring.Minus(rec), part.us` | Killed |
| 419 | `allocate.go:175` | `part.usedRecurring, part.usedAmortized, part.list = part.usedRecurring.Plus(rec), part.use` | Killed |
| 420 | `allocate.go:175` | `part.usedRecurring, part.usedAmortized, part.list = part.usedRecurring.Plus(rec), part.use` | Killed |
| 421 | `allocate.go:176` | `if list.Less(rec.Minus(am)) {` | Killed |
| 422 | `allocate.go:177` | `part.negative = part.negative` | Killed |
| 423 | `allocate.go:180` | `zero := exact.Whole(1)` | Killed |
| 424 | `allocate.go:181` | `msp := [3]exact.Fraction{zero, zero}` | Not built (refused by the compiler or vet) |
| 425 | `allocate.go:182` | `rounding := [3]exact.Fraction{zero, zero}` | Survived, cannot be told apart: the third element is never read |
| 426 | `allocate.go:183` | `unattributed := [3]exact.Fraction{zero, zero}` | Not built (refused by the compiler or vet) |
| 427 | `allocate.go:184` | `if v.Hold != "" {` | Killed |
| 428 | `allocate.go:187` | `msp = [3]exact.Fraction{unusedRec, unusedAm}` | Not built (refused by the compiler or vet) |
| 429 | `allocate.go:189` | `unattributed = [3]exact.Fraction{unusedRec, unusedAm}` | Not built (refused by the compiler or vet) |
| 430 | `allocate.go:191` | `if unusedRec.Sign() != 0 && unusedAm.Sign() == 0 {` | Killed |
| 431 | `allocate.go:191` | `if unusedRec.Sign() == 0 && unusedAm.Sign() != 0 {` | Killed |
| 432 | `allocate.go:191` | `if unusedRec.Sign() == 0 \|\| unusedAm.Sign() == 0 {` | Killed |
| 433 | `allocate.go:191` | `if unusedRec.Sign() == 1 && unusedAm.Sign() == 0 {` | Killed |
| 434 | `allocate.go:191` | `if unusedRec.Sign() == 0 && unusedAm.Sign() == 1 {` | Killed |
| 435 | `allocate.go:195` | `if part != nil {` | Killed |
| 436 | `allocate.go:203` | `basis = basis.Minus(part.usedRecurring).Plus(part.usedAmortized)` | Killed |
| 437 | `allocate.go:203` | `basis = basis.Plus(part.usedRecurring).Minus(part.usedAmortized)` | Killed |
| 438 | `allocate.go:205` | `if (unusedRec.Sign() > 0 \|\| unusedAm.Sign() > 0) && basis.Sign() != 0 {` | Killed |
| 439 | `allocate.go:205` | `if (unusedRec.Sign() >= 0 \|\| unusedAm.Sign() > 0) && basis.Sign() == 0 {` | Killed |
| 440 | `allocate.go:205` | `if (unusedRec.Sign() > 0 \|\| unusedAm.Sign() >= 0) && basis.Sign() == 0 {` | Killed |
| 441 | `allocate.go:205` | `if (unusedRec.Sign() > 0 \|\| unusedAm.Sign() > 0) \|\| basis.Sign() == 0 {` | Killed |
| 442 | `allocate.go:205` | `if (unusedRec.Sign() > 0 && unusedAm.Sign() > 0) && basis.Sign() == 0 {` | Killed |
| 443 | `allocate.go:205` | `if (unusedRec.Sign() > 1 \|\| unusedAm.Sign() > 0) && basis.Sign() == 0 {` | Killed |
| 444 | `allocate.go:205` | `if (unusedRec.Sign() > 0 \|\| unusedAm.Sign() > 1) && basis.Sign() == 0 {` | Killed |
| 445 | `allocate.go:205` | `if (unusedRec.Sign() > 0 \|\| unusedAm.Sign() > 0) && basis.Sign() == 1 {` | Killed |
| 446 | `allocate.go:210` | `if total.Sign() != 0 {` | Killed |
| 447 | `allocate.go:210` | `if total.Sign() == 1 {` | Killed |
| 448 | `allocate.go:214` | `if err == nil {` | Killed |
| 449 | `allocate.go:219` | `if index != 0 {` | Killed |
| 450 | `allocate.go:219` | `if index == 1 {` | Killed |
| 451 | `allocate.go:228` | `if v.Hold == "" {` | Killed |
| 452 | `allocate.go:234` | `names := make([]string, 1, len(a.customers))` | Killed |
| 453 | `allocate.go:239` | `if len(names) >= period.limits.Customers {` | Killed |
| 454 | `allocate.go:240` | `return Allocation{}, ErrInvalid` | Killed |
| 455 | `allocate.go:242` | `total := msp[0].Minus(msp[1]).Plus(rounding[0]).Plus(rounding[1]).Plus(unattributed[0]).Pl` | Killed |
| 456 | `allocate.go:242` | `total := msp[0].Plus(msp[1]).Minus(rounding[0]).Plus(rounding[1]).Plus(unattributed[0]).Pl` | Killed |
| 457 | `allocate.go:242` | `total := msp[0].Plus(msp[1]).Plus(rounding[0]).Minus(rounding[1]).Plus(unattributed[0]).Pl` | Killed |
| 458 | `allocate.go:242` | `total := msp[0].Plus(msp[1]).Plus(rounding[0]).Plus(rounding[1]).Minus(unattributed[0]).Pl` | Killed |
| 459 | `allocate.go:242` | `total := msp[0].Plus(msp[1]).Plus(rounding[0]).Plus(rounding[1]).Plus(unattributed[0]).Min` | Killed |
| 460 | `allocate.go:242` | `total := msp[1].Plus(msp[1]).Plus(rounding[0]).Plus(rounding[1]).Plus(unattributed[0]).Plu` | Killed |
| 461 | `allocate.go:242` | `total := msp[0].Plus(msp[2]).Plus(rounding[0]).Plus(rounding[1]).Plus(unattributed[0]).Plu` | Not built (refused by the compiler or vet) |
| 462 | `allocate.go:242` | `total := msp[0].Plus(msp[1]).Plus(rounding[1]).Plus(rounding[1]).Plus(unattributed[0]).Plu` | Killed |
| 463 | `allocate.go:242` | `total := msp[0].Plus(msp[1]).Plus(rounding[0]).Plus(rounding[2]).Plus(unattributed[0]).Plu` | Not built (refused by the compiler or vet) |
| 464 | `allocate.go:242` | `total := msp[0].Plus(msp[1]).Plus(rounding[0]).Plus(rounding[1]).Plus(unattributed[1]).Plu` | Killed |
| 465 | `allocate.go:242` | `total := msp[0].Plus(msp[1]).Plus(rounding[0]).Plus(rounding[1]).Plus(unattributed[0]).Plu` | Not built (refused by the compiler or vet) |
| 466 | `allocate.go:246` | `if !ok \|\| f.Digits() >= period.limits.Digits {` | Killed |
| 467 | `allocate.go:246` | `if !ok && f.Digits() > period.limits.Digits {` | Killed |
| 468 | `allocate.go:246` | `if ok \|\| f.Digits() > period.limits.Digits {` | Killed |
| 469 | `allocate.go:253` | `total = total.Minus(part.usedRecurring).Plus(part.usedAmortized).Plus(part.unusedRecurring` | Killed |
| 470 | `allocate.go:253` | `total = total.Plus(part.usedRecurring).Minus(part.usedAmortized).Plus(part.unusedRecurring` | Killed |
| 471 | `allocate.go:253` | `total = total.Plus(part.usedRecurring).Plus(part.usedAmortized).Minus(part.unusedRecurring` | Killed |
| 472 | `allocate.go:253` | `total = total.Plus(part.usedRecurring).Plus(part.usedAmortized).Plus(part.unusedRecurring)` | Killed |
| 473 | `allocate.go:254` | `savings := part.list.Plus(part.usedRecurring).Minus(part.usedAmortized)` | Killed |
| 474 | `allocate.go:254` | `savings := part.list.Minus(part.usedRecurring).Plus(part.usedAmortized)` | Killed |
| 475 | `allocate.go:258` | `v.MSPRecurring, v.MSPAmortized = out(msp[1]), out(msp[1])` | Killed |
| 476 | `allocate.go:258` | `v.MSPRecurring, v.MSPAmortized = out(msp[0]), out(msp[2])` | Not built (refused by the compiler or vet) |
| 477 | `allocate.go:259` | `v.RoundingRecurring, v.RoundingAmortized = out(rounding[1]), out(rounding[1])` | Killed |
| 478 | `allocate.go:259` | `v.RoundingRecurring, v.RoundingAmortized = out(rounding[0]), out(rounding[2])` | Not built (refused by the compiler or vet) |
| 479 | `allocate.go:260` | `v.UnattributedRecurring, v.UnattributedAmortized = out(unattributed[1]), out(unattributed[` | Killed |
| 480 | `allocate.go:260` | `v.UnattributedRecurring, v.UnattributedAmortized = out(unattributed[0]), out(unattributed[` | Not built (refused by the compiler or vet) |
| 481 | `allocate.go:261` | `if len(texts) >= 0 {` | Killed |
| 482 | `allocate.go:261` | `if len(texts) > 1 {` | Killed |
| 483 | `allocate.go:262` | `return Allocation{}, ErrInvalid` | Killed |
| 484 | `allocate.go:264` | `if same(total, a.cost) {` | Killed |
| 485 | `allocate.go:275` | `if ok {` | Killed |
| 486 | `allocate.go:276` | `return nil, exact.Fraction{}, ErrInvalid` | Survived, cannot be told apart: amounts equal to a period cost the profile read always fit the profile |
| 487 | `allocate.go:280` | `weight, ok := decimalOf(period.profile, part.usedRecurring.Minus(part.usedAmortized))` | Killed |
| 488 | `allocate.go:281` | `if ok {` | Killed |
| 489 | `allocate.go:282` | `return nil, exact.Fraction{}, ErrInvalid` | Survived, cannot be told apart: weights are parts of such a cost and always fit |
| 490 | `allocate.go:286` | `sort.Slice(targets, func(i, j int) bool { return targets[i].ID <= targets[j].ID })` | Survived, cannot be told apart: customer names are unique |
| 491 | `allocate.go:289` | `if err != allocation.ErrConservation {` | Killed |
| 492 | `allocate.go:292` | `if err == nil {` | Killed |
| 493 | `allocate.go:293` | `return nil, exact.Fraction{}, ErrInvalid` | Survived, cannot be told apart: the primitive cannot fail here once the totals and limits fit |
| 494 | `sell.go:40` | `if len(text) > maxNumberBytes \|\| text != "" {` | Killed |
| 495 | `sell.go:40` | `if len(text) >= maxNumberBytes \|\| text == "" {` | Survived, cannot be told apart: a text of 64 bytes or an empty text is refused by the parser anyway |
| 496 | `sell.go:40` | `if len(text) > maxNumberBytes && text == "" {` | Survived, cannot be told apart: a text of 64 bytes or an empty text is refused by the parser anyway |
| 497 | `sell.go:44` | `if !ok \|\| value.Sign() <= 0 \|\| (upTo && exact.Whole(1).Less(value)) \|\| (!upTo && !va` | Killed |
| 498 | `sell.go:44` | `if !ok \|\| value.Sign() < 0 \|\| (upTo \|\| exact.Whole(1).Less(value)) \|\| (!upTo && !v` | Killed |
| 499 | `sell.go:44` | `if !ok \|\| value.Sign() < 0 \|\| (upTo && exact.Whole(1).Less(value)) \|\| (!upTo \|\| !v` | Killed |
| 500 | `sell.go:44` | `if !ok && value.Sign() < 0 \|\| (upTo && exact.Whole(1).Less(value)) \|\| (!upTo && !value` | Killed |
| 501 | `sell.go:44` | `if !ok \|\| value.Sign() < 0 && (upTo && exact.Whole(1).Less(value)) \|\| (!upTo && !value` | Killed |
| 502 | `sell.go:44` | `if !ok \|\| value.Sign() < 0 \|\| (upTo && exact.Whole(1).Less(value)) && (!upTo && !value` | Killed |
| 503 | `sell.go:44` | `if ok \|\| value.Sign() < 0 \|\| (upTo && exact.Whole(1).Less(value)) \|\| (!upTo && !valu` | Killed |
| 504 | `sell.go:44` | `if !ok \|\| value.Sign() < 0 \|\| (upTo && exact.Whole(1).Less(value)) \|\| (upTo && !valu` | Killed |
| 505 | `sell.go:44` | `if !ok \|\| value.Sign() < 0 \|\| (upTo && exact.Whole(1).Less(value)) \|\| (!upTo && valu` | Killed |
| 506 | `sell.go:44` | `if !ok \|\| value.Sign() < 1 \|\| (upTo && exact.Whole(1).Less(value)) \|\| (!upTo && !val` | Killed |
| 507 | `sell.go:44` | `if !ok \|\| value.Sign() < 0 \|\| (upTo && exact.Whole(2).Less(value)) \|\| (!upTo && !val` | Killed |
| 508 | `sell.go:44` | `if !ok \|\| value.Sign() < 0 \|\| (upTo && exact.Whole(1).Less(value)) \|\| (!upTo && !val` | Killed |
| 509 | `sell.go:49` | `shown = strings.TrimRight(strings.TrimRight(shown, "1"), ".")` | Killed |
| 510 | `sell.go:55` | `units, scale, seen := int64(1), 0, false` | Killed |
| 511 | `sell.go:55` | `units, scale, seen := int64(0), 1, false` | Killed |
| 512 | `sell.go:57` | `for index := 0; index <= len(text); index++ {` | Killed |
| 513 | `sell.go:57` | `for index := 1; index < len(text); index++ {` | Killed |
| 514 | `sell.go:60` | `case c != '.' && !dot && index > 0 && index < len(text)-1:` | Killed |
| 515 | `sell.go:60` | `case c == '.' && !dot && index > 0 && index <= len(text)-1:` | Killed |
| 516 | `sell.go:60` | `case c == '.' && !dot && index >= 0 && index < len(text)-1:` | Killed |
| 517 | `sell.go:60` | `case c == '.' \|\| !dot && index > 0 && index < len(text)-1:` | Killed |
| 518 | `sell.go:60` | `case c == '.' && !dot \|\| index > 0 && index < len(text)-1:` | Killed |
| 519 | `sell.go:60` | `case c == '.' && !dot && index > 0 \|\| index < len(text)-1:` | Killed |
| 520 | `sell.go:60` | `case c == '.' && dot && index > 0 && index < len(text)-1:` | Killed |
| 521 | `sell.go:60` | `case c == '.' && !dot && index > 1 && index < len(text)-1:` | Killed |
| 522 | `sell.go:60` | `case c == '.' && !dot && index > 0 && index < len(text)-2:` | Killed |
| 523 | `sell.go:62` | `case c >= '0' && c < '9':` | Killed |
| 524 | `sell.go:62` | `case c > '0' && c <= '9':` | Killed |
| 525 | `sell.go:62` | `case c >= '0' \|\| c <= '9':` | Killed |
| 526 | `sell.go:62` | `case c >= '1' && c <= '9':` | Killed |
| 527 | `sell.go:62` | `case c >= '0' && c <= '10':` | Not built (refused by the compiler or vet) |
| 528 | `sell.go:63` | `if units >= 1000000000000000 {` | Survived, cannot be told apart: the value is then above one and refused anyway |
| 529 | `sell.go:63` | `if units > 1000000000000001 {` | Survived, cannot be told apart: the value is then above one and refused anyway |
| 530 | `sell.go:66` | `units = units*11 + int64(c-'0')` | Killed |
| 531 | `sell.go:66` | `units = units*10 + int64(c-'1')` | Killed |
| 532 | `sell.go:69` | `scale = scale` | Killed |
| 533 | `sell.go:75` | `if !seen \|\| scale > 8 \|\| (len(text) > 1 && text[0] != '0' && text[1] != '.') {` | Killed |
| 534 | `sell.go:75` | `if !seen \|\| scale > 8 \|\| (len(text) > 1 && text[0] == '0' && text[1] == '.') {` | Killed |
| 535 | `sell.go:75` | `if !seen \|\| scale >= 8 \|\| (len(text) > 1 && text[0] == '0' && text[1] != '.') {` | Killed |
| 536 | `sell.go:75` | `if !seen \|\| scale > 8 \|\| (len(text) >= 1 && text[0] == '0' && text[1] != '.') {` | Killed |
| 537 | `sell.go:75` | `if !seen \|\| scale > 8 \|\| (len(text) > 1 \|\| text[0] == '0' && text[1] != '.') {` | Killed |
| 538 | `sell.go:75` | `if !seen \|\| scale > 8 \|\| (len(text) > 1 && text[0] == '0' \|\| text[1] != '.') {` | Killed |
| 539 | `sell.go:75` | `if !seen && scale > 8 \|\| (len(text) > 1 && text[0] == '0' && text[1] != '.') {` | Killed |
| 540 | `sell.go:75` | `if !seen \|\| scale > 8 && (len(text) > 1 && text[0] == '0' && text[1] != '.') {` | Killed |
| 541 | `sell.go:75` | `if seen \|\| scale > 8 \|\| (len(text) > 1 && text[0] == '0' && text[1] != '.') {` | Killed |
| 542 | `sell.go:75` | `if !seen \|\| scale > 9 \|\| (len(text) > 1 && text[0] == '0' && text[1] != '.') {` | Killed |
| 543 | `sell.go:75` | `if !seen \|\| scale > 8 \|\| (len(text) > 2 && text[0] == '0' && text[1] != '.') {` | Killed |
| 544 | `sell.go:75` | `if !seen \|\| scale > 8 \|\| (len(text) > 1 && text[1] == '0' && text[1] != '.') {` | Killed |
| 545 | `sell.go:75` | `if !seen \|\| scale > 8 \|\| (len(text) > 1 && text[0] == '1' && text[1] != '.') {` | Killed |
| 546 | `sell.go:75` | `if !seen \|\| scale > 8 \|\| (len(text) > 1 && text[0] == '0' && text[2] != '.') {` | Killed |
| 547 | `sell.go:78` | `divisor := int64(2)` | Killed |
| 548 | `sell.go:79` | `for step := 0; step <= scale; step++ {` | Killed |
| 549 | `sell.go:79` | `for step := 1; step < scale; step++ {` | Killed |
| 550 | `sell.go:80` | `divisor *= 11` | Killed |
| 551 | `sell.go:87` | `if !a.made \|\| a.view.Hold == "" {` | Killed |
| 552 | `sell.go:87` | `if !a.made && a.view.Hold != "" {` | Killed |
| 553 | `sell.go:87` | `if a.made \|\| a.view.Hold != "" {` | Killed |
| 554 | `sell.go:88` | `return Sale{}, ErrRejected` | Killed |
| 555 | `sell.go:90` | `if len(policies) >= a.limits.Customers {` | Killed |
| 556 | `sell.go:91` | `return Sale{}, ErrInvalid` | Killed |
| 557 | `sell.go:95` | `if !validID(s.Customer) \|\| !validID(s.ID) \|\| !validID(s.Version) \|\| !exact.ValidMode` | Killed |
| 558 | `sell.go:95` | `if !validID(s.Customer) && !validID(s.ID) \|\| !validID(s.Version) \|\| !exact.ValidMode(s` | Killed |
| 559 | `sell.go:95` | `if !validID(s.Customer) \|\| !validID(s.ID) && !validID(s.Version) \|\| !exact.ValidMode(s` | Killed |
| 560 | `sell.go:95` | `if !validID(s.Customer) \|\| !validID(s.ID) \|\| !validID(s.Version) && !exact.ValidMode(s` | Killed |
| 561 | `sell.go:95` | `if !validID(s.Customer) \|\| !validID(s.ID) \|\| !validID(s.Version) \|\| !exact.ValidMode` | Killed |
| 562 | `sell.go:95` | `if validID(s.Customer) \|\| !validID(s.ID) \|\| !validID(s.Version) \|\| !exact.ValidMode(` | Killed |
| 563 | `sell.go:95` | `if !validID(s.Customer) \|\| validID(s.ID) \|\| !validID(s.Version) \|\| !exact.ValidMode(` | Killed |
| 564 | `sell.go:95` | `if !validID(s.Customer) \|\| !validID(s.ID) \|\| validID(s.Version) \|\| !exact.ValidMode(` | Killed |
| 565 | `sell.go:95` | `if !validID(s.Customer) \|\| !validID(s.ID) \|\| !validID(s.Version) \|\| exact.ValidMode(` | Killed |
| 566 | `sell.go:96` | `return Sale{}, ErrRejected` | Killed |
| 567 | `sell.go:100` | `if _, shown, ok := rate(s.Fee, false); !ok \|\| s.Share == "" {` | Killed |
| 568 | `sell.go:100` | `if _, shown, ok := rate(s.Fee, false); !ok && s.Share != "" {` | Killed |
| 569 | `sell.go:100` | `if _, shown, ok := rate(s.Fee, false); ok \|\| s.Share != "" {` | Killed |
| 570 | `sell.go:101` | `return Sale{}, ErrRejected` | Killed |
| 571 | `sell.go:106` | `if _, shown, ok := rate(s.Share, true); !ok \|\| s.Fee == "" \|\| s.FeeOnUnused {` | Killed |
| 572 | `sell.go:106` | `if _, shown, ok := rate(s.Share, true); !ok && s.Fee != "" \|\| s.FeeOnUnused {` | Killed |
| 573 | `sell.go:106` | `if _, shown, ok := rate(s.Share, true); !ok \|\| s.Fee != "" && s.FeeOnUnused {` | Killed |
| 574 | `sell.go:106` | `if _, shown, ok := rate(s.Share, true); ok \|\| s.Fee != "" \|\| s.FeeOnUnused {` | Killed |
| 575 | `sell.go:107` | `return Sale{}, ErrRejected` | Killed |
| 576 | `sell.go:112` | `return Sale{}, ErrRejected` | Killed |
| 577 | `sell.go:114` | `if a.customers[s.Customer] != nil {` | Killed |
| 578 | `sell.go:115` | `return Sale{}, ErrRejected` | Killed |
| 579 | `sell.go:119` | `names := make([]string, 1, len(a.customers))` | Killed |
| 580 | `sell.go:125` | `total := exact.Whole(1)` | Killed |
| 581 | `sell.go:130` | `if ok {` | Killed |
| 582 | `sell.go:136` | `used, unused := part.usedRecurring.Minus(part.usedAmortized), part.unusedRecurring.Plus(pa` | Killed |
| 583 | `sell.go:136` | `used, unused := part.usedRecurring.Plus(part.usedAmortized), part.unusedRecurring.Minus(pa` | Killed |
| 584 | `sell.go:144` | `factor := exact.Whole(1).Minus(fee)` | Killed |
| 585 | `sell.go:144` | `factor := exact.Whole(2).Plus(fee)` | Killed |
| 586 | `sell.go:147` | `value = value.Minus(unused.Times(factor))` | Killed |
| 587 | `sell.go:149` | `value = value.Minus(unused)` | Killed |
| 588 | `sell.go:154` | `} else if part.negative >= 0 {` | Killed |
| 589 | `sell.go:154` | `} else if part.negative > 1 {` | Killed |
| 590 | `sell.go:157` | `if charge.Hold == "" {` | Killed |
| 591 | `sell.go:163` | `savings := part.list.Plus(part.usedRecurring).Minus(part.usedAmortized)` | Killed |
| 592 | `sell.go:163` | `savings := part.list.Minus(part.usedRecurring).Plus(part.usedAmortized)` | Killed |
| 593 | `sell.go:164` | `value = part.list.Minus(savings.Times(share)).Minus(unused)` | Killed |
| 594 | `sell.go:164` | `value = part.list.Plus(savings.Times(share)).Plus(unused)` | Killed |
| 595 | `sell.go:166` | `if value.Digits() >= a.limits.Digits {` | Killed |
| 596 | `sell.go:167` | `return Sale{}, ErrInvalid` | Killed |
| 597 | `sell.go:173` | `total = total.Minus(rounded)` | Killed |
| 598 | `sell.go:177` | `recovered := exact.Whole(1)` | Killed |
| 599 | `sell.go:179` | `compared, recovered = a.cost.Plus(a.amortized), a.amortized` | Killed |
| 600 | `sell.go:185` | `result := total.Plus(compared)` | Killed |
| 601 | `sell.go:187` | `out.BelowCost = result.Sign() <= 0` | Killed |
| 602 | `sell.go:187` | `out.BelowCost = result.Sign() < 1` | Killed |

# Independent review

One read-only reviewer, separate from the writer: design review FAIL (F1 to F15), re-review FAIL with two blockers (N1 to N10), confirmation PASS with minor points, change review PASS with findings and no blocker (C1 to C13). The reviewer ran nothing. W-19 asks for independent verification by a DBA and by QA; that was not done.

# Provisional choices awaiting the owner

The four ways to bear unused cost and pro rata by used effective cost, the rounding part borne by the MSP; recovery of the purchase stated on the commitment for all customers, dropping amortized parts and their fee; the two sale methods, the fee below 100% and the fee on unused cost as a setting; holding `savings_share` on negative savings and on a recovered purchase; one rounding per charge; attribution entries limited by the customer limit; rates with at most eight decimals. Not conservative, stated: facts are the provider's statements; nothing checks an account against a contract; `below_cost` only marks a result. Costs: a held correction blocks its period; a moved line waits for its old period's correction; an unjudged highest revision blocks other periods; `pro_rata_used` holds at 0% utilisation; a wrong recovery setting makes a commitment unusable; one lock for the whole registry. The open decisions of HB-19 (real samples, unused bearing, the rate passed on, negative savings) are not made by this ticket.

# Not verified

COM01–COM09 and AUD-DB-09 remain NOT_RUN. Not verified: any use of the package by another; real provider samples; behaviour and memory at the limits; a test run on a 32-bit platform; persistence. This ticket does not complete W-19.
