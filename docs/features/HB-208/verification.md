---
type: VerificationRecord
title: Exchange-rate engine verification
description: Records what was executed and reviewed for the first model of work item W-18 and the exact-fraction primitive, with results, the corrections made on the way, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-208](https://linear.app/hybrid-billing/issue/HB-208/w-18-exchange-rate-observations-policy-precedence-snapshots-before-a) adds two Go packages from main `ae900c2`. `packages/primitives/exact` makes the exact fractions of `modules/formula` public: a positive denominator in lowest terms, operations that change no operand, a refusal of a zero divisor, and one rounding under six modes. `modules/fx` is an in-memory registry: rate observations kept as quoted, with revisions of which only the highest decides, manual and file entries and voids that need an approval by another actor; policy versions at the levels customer, contract and MSP with the methods fixed (with evidence and an approval), dated and monthly average, `ended` and `default_allowed`, withdrawals that hold their dates; snapshots prepared once for a list of needs, each entry selected by the precedence customer exception, contract, MSP default allowed by the contract, with a rate on one date for every leg or an average of normalised values, a sales quote rounded once in its quoted orientation and refused beyond a tolerance, or a hold with one of fifteen reasons; two digests; and a conversion that reads only a stored snapshot, returns a same-currency amount and an amount stated to be in the target currency unchanged, and rounds the converted amount and the fee once.

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan three times before implementation and the change afterwards (see Independent review). Machine: mac_mini.

Nothing calls either package. Of the acceptance of HB-18, at the level of this package: USD 900 at 1350 is 1,215,000 won; after a revision to 1360 a new snapshot gives 1,224,000, a difference of 9,000, while the earlier snapshot still gives 1,215,000 and the content digests differ; a quotation per 100 JPY, an inverse rate and rounding are covered by literal tests. Re-approval is not built: a changed rate gives another content digest, which is all this package contributes. Not delivered: fetching rates, a business-day calendar, approvals of other policy versions and of snapshots, the rate for corrections, rounding differences between lines and totals, the ERP posting, rates per line item, the purposes `display` and `accounting` beyond labels. The registry authorizes nothing and must not be exposed directly.

# Executed checks

Executed on 2026-11-13 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain, with `GOCACHE` set to a directory of the session. A test build for linux/386 was made; no test was run on another platform.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All twenty-seven packages `ok`, including `modules/fx` and `packages/primitives/exact`; run on the final code |
| `go test -count=15` and `-count=10` on the two packages | `ok`: fifteen consecutive runs before the mutation runs and ten on the final code |
| `GOOS=linux GOARCH=386 go test -c` on the two packages | Builds |

Literal tests with expected values worked out by hand (`acceptance_test.go`, `resolve_test.go`, `registry_test.go`, `bounds_test.go`, `review_test.go`, `exact_test.go`): the acceptance figures; per 100 JPY with a large amount; an inverse of 1,215,000,000 won giving 900,000 dollars; a spread on direct and inverse quotes; the fee from the exact converted amount (1 dollar at 1004.5 with a fee rate of 0.1 gives 1005 and a fee of 100); credits; negative amounts under `floor` and `ceiling`; the precedence and every hold reason; withdrawals and the version that closes a hole; fixed rates in both orientations before and after approval; the dated search at its bounds, legs on different dates, today with and without a value, offsets ahead of and behind UTC, a clock that went back; manual entries, corrections and voids in both search and average; averages with mixed units, inverse averages, months of 28, 29, 30 and 31 days and the turn of a year; a quote that rounds to zero, a cross refused for precision, a difference equal to and above the tolerance and the inverse case; every refusal of every operation in the order of error kinds; the conversion table cell by cell; the content digest against bytes built in the test and against changes of every field and a byte moved between neighbours; equal content digests across preparation times; limits at their bounds, digits exactly at and beyond the limit; pages; copies returned by reads; the tenant boundary; errors as fixed values; a scan of the source for words of floating point, connections and comments, which is a scan of words.

Properties with fixed seeds: the primitive's arithmetic against `math/big.Rat` over 20,000 cases and its rounding against the decimal primitive's `DivideRound` over 20,000 quotients, a third with negative divisors, at five scales under six modes; 400 rounds of random observations (revisions, manual entries, approvals, voids), random policies at three levels and thirty random needs, resolved by the registry and by a reference written with `math/big.Rat`, comparing kind, reason and sales rate of every entry and the converted amount and fee of every line, with invariants that every observation an entry names is the highest revision of its key, that no entry uses more days than its policy allows, that both legs of a dated cross share a date and that a total is amount plus fee, and with a list of every selection reason, hold reason, method and construction that must have been met. The reference's selection, search and averaging follow this plan as the code does; its arithmetic is its own. It does not track refusal kinds per operation; the literal tests do. Concurrency under the race detector with gates: a snapshot prepared beside a revision holds one value wholly; conversions beside a revision and another preparation give equal results.

# Corrections made on the way

- The first design review returned FAIL (a rounded rate that lost precision silently; a void that chose rates without approval). The plan was rewritten; the re-review returned FAIL on one point (averages of observations in different units), resolved by a rule, and its confirmation returned PASS with findings, all resolved in the plan.
- Code errors found by my own tests: `Convert` judged a line's need and its table cell in one pass, so a refusal could come before a missing need; corrected to judge every need first. A redundant sorted copy of policy versions, a condition on the search window and an unreachable return were removed after the mutation runs showed them unobservable; the argument for each is in the plan and below.
- Expectations of mine that were wrong and corrected before the mutation runs: the inverse sales rate with a spread (250/247401, not 625/618502) and the converted amount from it (1010.51, not 1010.5); the fields of the hand-built content digest for a same-currency entry.
- During a mutation run the disk filled because the session's build cache had grown to 16 GB; the restore of one source file failed and left a mutant (the spread subtracted instead of added) in `resolve.go`. My tests failed on it, it was restored, and the runner was changed to read every restored file back and stop when it differs, and the cache is emptied after every part. The change review then confirmed the spread.
- The change review's findings C1 to C7 led to added tests, the removal of a dead field and constant, held entries without observations, and renamed identifiers; these corrections were not reviewed again by a separate reviewer.

# Mutation check

A script in Python generates mutants from the source of the two packages by fixed rules (each comparison and boolean operator swapped, each negation removed, each error and hold reason replaced, each number raised by one, each lone increment dropped) and runs the package's tests for each, restoring and reading back the file after every mutant; results are appended to files outside the repository. The table covers the 871 mutants of the final code: the complete run of every file but `resolve.go`, whose source did not change afterwards, with the mutants that survived it run again on the final tests; every mutant of `resolve.go` run on the final code; and three survivors of that run run again after the last tests were added.

833 were killed. 14 did not build, the compiler or `go vet` refusing them (contradictory comparisons and character literals of two digits). 24 survive and cannot be told apart by any test, each for the reason given: values that are never read when a flag or a hold says so, a zero that changes nothing, or a bound that gives the same result. Earlier runs of these mutants are not reported, the source having changed since.

| # | Place | Mutant | Result |
| --- | --- | --- | --- |
| 1 | `exact.go:10` | `one = big.NewInt(2)` | Killed |
| 2 | `exact.go:11` | `two = big.NewInt(3)` | Killed |
| 3 | `exact.go:12` | `ten = big.NewInt(11)` | Killed |
| 4 | `exact.go:29` | `if den.Sign() <= 0 {` | Survived, cannot be told apart: the denominator is never zero |
| 5 | `exact.go:29` | `if den.Sign() < 1 {` | Survived, cannot be told apart: the denominator is never zero |
| 6 | `exact.go:37` | `return Fraction{num: big.NewInt(value), den: big.NewInt(2)}` | Killed |
| 7 | `exact.go:41` | `if value.Valid() {` | Killed |
| 8 | `exact.go:48` | `if units != nil \|\| scale < 0 {` | Killed |
| 9 | `exact.go:48` | `if units == nil \|\| scale <= 0 {` | Killed |
| 10 | `exact.go:48` | `if units == nil && scale < 0 {` | Killed |
| 11 | `exact.go:48` | `if units == nil \|\| scale < 1 {` | Killed |
| 12 | `exact.go:63` | `return f.Minus(Fraction{num: new(big.Int).Neg(other.num), den: other.den})` | Killed |
| 13 | `exact.go:71` | `if other.num.Sign() != 0 {` | Killed |
| 14 | `exact.go:71` | `if other.num.Sign() == 1 {` | Killed |
| 15 | `exact.go:80` | `return new(big.Int).Mul(f.num, other.den).Cmp(new(big.Int).Mul(other.num, f.den)) <= 0` | Killed |
| 16 | `exact.go:80` | `return new(big.Int).Mul(f.num, other.den).Cmp(new(big.Int).Mul(other.num, f.den)) < 1` | Killed |
| 17 | `exact.go:85` | `if den >= num {` | Survived, cannot be told apart: equal counts give the same number |
| 18 | `exact.go:92` | `if f.den.Cmp(one) != 0 {` | Killed |
| 19 | `exact.go:92` | `if f.den.Cmp(one) == 1 {` | Killed |
| 20 | `exact.go:99` | `if scale <= 0 \|\| !modes[mode] {` | Killed |
| 21 | `exact.go:99` | `if scale < 0 && !modes[mode] {` | Killed |
| 22 | `exact.go:99` | `if scale < 0 \|\| modes[mode] {` | Killed |
| 23 | `exact.go:99` | `if scale < 1 \|\| !modes[mode] {` | Killed |
| 24 | `exact.go:110` | `up = remainder.Sign() <= 0` | Survived, cannot be told apart: a remainder of zero adds nothing |
| 25 | `exact.go:110` | `up = remainder.Sign() < 1` | Survived, cannot be told apart: a remainder of zero adds nothing |
| 26 | `exact.go:112` | `up = remainder.Sign() >= 0` | Survived, cannot be told apart: a remainder of zero adds nothing |
| 27 | `exact.go:112` | `up = remainder.Sign() > 1` | Killed |
| 28 | `exact.go:114` | `up = half > 0` | Killed |
| 29 | `exact.go:114` | `up = half >= 1` | Killed |
| 30 | `exact.go:116` | `up = half > 0 \|\| (half != 0 && quotient.Bit(0) == 1)` | Killed |
| 31 | `exact.go:116` | `up = half > 0 \|\| (half == 0 && quotient.Bit(0) != 1)` | Killed |
| 32 | `exact.go:116` | `up = half >= 0 \|\| (half == 0 && quotient.Bit(0) == 1)` | Killed |
| 33 | `exact.go:116` | `up = half > 0 \|\| (half == 0 \|\| quotient.Bit(0) == 1)` | Killed |
| 34 | `exact.go:116` | `up = half > 0 && (half == 0 && quotient.Bit(0) == 1)` | Killed |
| 35 | `exact.go:116` | `up = half > 1 \|\| (half == 0 && quotient.Bit(0) == 1)` | Killed |
| 36 | `exact.go:116` | `up = half > 0 \|\| (half == 1 && quotient.Bit(0) == 1)` | Killed |
| 37 | `exact.go:116` | `up = half > 0 \|\| (half == 0 && quotient.Bit(1) == 1)` | Killed |
| 38 | `exact.go:116` | `up = half > 0 \|\| (half == 0 && quotient.Bit(0) == 2)` | Killed |
| 39 | `types.go:20` | `maxIDBytes     = 129` | Killed |
| 40 | `types.go:21` | `maxNumberBytes = 65` | Killed |
| 41 | `types.go:22` | `maxQuoteScale  = 9` | Killed |
| 42 | `types.go:23` | `minOffset      = -721` | Killed |
| 43 | `types.go:24` | `maxOffset      = 841` | Killed |
| 44 | `types.go:25` | `maxDays        = 32` | Killed |
| 45 | `types.go:26` | `maxRevision    = 1001` | Killed |
| 46 | `types.go:27` | `maxVersion     = 10001` | Killed |
| 47 | `types.go:28` | `firstYear      = 2001` | Killed |
| 48 | `types.go:29` | `lastYear       = 2100` | Killed |
| 49 | `types.go:88` | `units          = map[int]bool{2: true, 10: true, 100: true, 1000: true, 10000: true}` | Killed |
| 50 | `types.go:88` | `units          = map[int]bool{1: true, 11: true, 100: true, 1000: true, 10000: true}` | Killed |
| 51 | `types.go:88` | `units          = map[int]bool{1: true, 10: true, 101: true, 1000: true, 10000: true}` | Killed |
| 52 | `types.go:88` | `units          = map[int]bool{1: true, 10: true, 100: true, 1001: true, 10000: true}` | Killed |
| 53 | `types.go:88` | `units          = map[int]bool{1: true, 10: true, 100: true, 1000: true, 10001: true}` | Killed |
| 54 | `types.go:94` | `quanta         = map[string]int{"2": 0, "0.1": 1, "0.01": 2, "0.001": 3, "0.0001": 4}` | Killed |
| 55 | `types.go:94` | `quanta         = map[string]int{"1": 1, "0.1": 1, "0.01": 2, "0.001": 3, "0.0001": 4}` | Killed |
| 56 | `types.go:94` | `quanta         = map[string]int{"1": 0, "1.1": 1, "0.01": 2, "0.001": 3, "0.0001": 4}` | Killed |
| 57 | `types.go:94` | `quanta         = map[string]int{"1": 0, "0.2": 1, "0.01": 2, "0.001": 3, "0.0001": 4}` | Killed |
| 58 | `types.go:94` | `quanta         = map[string]int{"1": 0, "0.1": 2, "0.01": 2, "0.001": 3, "0.0001": 4}` | Killed |
| 59 | `types.go:94` | `quanta         = map[string]int{"1": 0, "0.1": 1, "1.01": 2, "0.001": 3, "0.0001": 4}` | Killed |
| 60 | `types.go:94` | `quanta         = map[string]int{"1": 0, "0.1": 1, "0.2": 2, "0.001": 3, "0.0001": 4}` | Killed |
| 61 | `types.go:94` | `quanta         = map[string]int{"1": 0, "0.1": 1, "0.01": 3, "0.001": 3, "0.0001": 4}` | Killed |
| 62 | `types.go:94` | `quanta         = map[string]int{"1": 0, "0.1": 1, "0.01": 2, "1.001": 3, "0.0001": 4}` | Killed |
| 63 | `types.go:94` | `quanta         = map[string]int{"1": 0, "0.1": 1, "0.01": 2, "0.2": 3, "0.0001": 4}` | Killed |
| 64 | `types.go:94` | `quanta         = map[string]int{"1": 0, "0.1": 1, "0.01": 2, "0.001": 4, "0.0001": 4}` | Killed |
| 65 | `types.go:94` | `quanta         = map[string]int{"1": 0, "0.1": 1, "0.01": 2, "0.001": 3, "1.0001": 4}` | Killed |
| 66 | `types.go:94` | `quanta         = map[string]int{"1": 0, "0.1": 1, "0.01": 2, "0.001": 3, "0.2": 4}` | Killed |
| 67 | `types.go:94` | `quanta         = map[string]int{"1": 0, "0.1": 1, "0.01": 2, "0.001": 3, "0.0001": 5}` | Killed |
| 68 | `types.go:317` | `return l.Keys > 0 && l.Keys < 1000000 && l.Revisions > 0 && l.Revisions <= 1000 &&` | Killed |
| 69 | `types.go:317` | `return l.Keys > 0 && l.Keys <= 1000000 && l.Revisions > 0 && l.Revisions < 1000 &&` | Killed |
| 70 | `types.go:317` | `return l.Keys >= 0 && l.Keys <= 1000000 && l.Revisions > 0 && l.Revisions <= 1000 &&` | Killed |
| 71 | `types.go:317` | `return l.Keys > 0 && l.Keys <= 1000000 && l.Revisions >= 0 && l.Revisions <= 1000 &&` | Killed |
| 72 | `types.go:317` | `return l.Keys > 0 \|\| l.Keys <= 1000000 && l.Revisions > 0 && l.Revisions <= 1000 &&` | Killed |
| 73 | `types.go:317` | `return l.Keys > 0 && l.Keys <= 1000000 \|\| l.Revisions > 0 && l.Revisions <= 1000 &&` | Killed |
| 74 | `types.go:317` | `return l.Keys > 0 && l.Keys <= 1000000 && l.Revisions > 0 \|\| l.Revisions <= 1000 &&` | Killed |
| 75 | `types.go:317` | `return l.Keys > 1 && l.Keys <= 1000000 && l.Revisions > 0 && l.Revisions <= 1000 &&` | Killed |
| 76 | `types.go:317` | `return l.Keys > 0 && l.Keys <= 1000001 && l.Revisions > 0 && l.Revisions <= 1000 &&` | Killed |
| 77 | `types.go:317` | `return l.Keys > 0 && l.Keys <= 1000000 && l.Revisions > 1 && l.Revisions <= 1000 &&` | Killed |
| 78 | `types.go:317` | `return l.Keys > 0 && l.Keys <= 1000000 && l.Revisions > 0 && l.Revisions <= 1001 &&` | Killed |
| 79 | `types.go:318` | `l.Policies > 0 && l.Policies < 100000 && l.Versions > 0 && l.Versions <= 10000 &&` | Killed |
| 80 | `types.go:318` | `l.Policies > 0 && l.Policies <= 100000 && l.Versions > 0 && l.Versions < 10000 &&` | Killed |
| 81 | `types.go:318` | `l.Policies >= 0 && l.Policies <= 100000 && l.Versions > 0 && l.Versions <= 10000 &&` | Killed |
| 82 | `types.go:318` | `l.Policies > 0 && l.Policies <= 100000 && l.Versions >= 0 && l.Versions <= 10000 &&` | Killed |
| 83 | `types.go:318` | `l.Policies > 0 \|\| l.Policies <= 100000 && l.Versions > 0 && l.Versions <= 10000 &&` | Killed |
| 84 | `types.go:318` | `l.Policies > 0 && l.Policies <= 100000 \|\| l.Versions > 0 && l.Versions <= 10000 &&` | Killed |
| 85 | `types.go:318` | `l.Policies > 0 && l.Policies <= 100000 && l.Versions > 0 \|\| l.Versions <= 10000 &&` | Killed |
| 86 | `types.go:318` | `l.Policies > 1 && l.Policies <= 100000 && l.Versions > 0 && l.Versions <= 10000 &&` | Killed |
| 87 | `types.go:318` | `l.Policies > 0 && l.Policies <= 100001 && l.Versions > 0 && l.Versions <= 10000 &&` | Killed |
| 88 | `types.go:318` | `l.Policies > 0 && l.Policies <= 100000 && l.Versions > 1 && l.Versions <= 10000 &&` | Killed |
| 89 | `types.go:318` | `l.Policies > 0 && l.Policies <= 100000 && l.Versions > 0 && l.Versions <= 10001 &&` | Killed |
| 90 | `types.go:319` | `l.Needs > 0 && l.Needs < 10000 && l.Snapshots > 0 && l.Snapshots <= 100000 &&` | Killed |
| 91 | `types.go:319` | `l.Needs > 0 && l.Needs <= 10000 && l.Snapshots > 0 && l.Snapshots < 100000 &&` | Killed |
| 92 | `types.go:319` | `l.Needs >= 0 && l.Needs <= 10000 && l.Snapshots > 0 && l.Snapshots <= 100000 &&` | Killed |
| 93 | `types.go:319` | `l.Needs > 0 && l.Needs <= 10000 && l.Snapshots >= 0 && l.Snapshots <= 100000 &&` | Killed |
| 94 | `types.go:319` | `l.Needs > 0 \|\| l.Needs <= 10000 && l.Snapshots > 0 && l.Snapshots <= 100000 &&` | Killed |
| 95 | `types.go:319` | `l.Needs > 0 && l.Needs <= 10000 \|\| l.Snapshots > 0 && l.Snapshots <= 100000 &&` | Killed |
| 96 | `types.go:319` | `l.Needs > 0 && l.Needs <= 10000 && l.Snapshots > 0 \|\| l.Snapshots <= 100000 &&` | Killed |
| 97 | `types.go:319` | `l.Needs > 1 && l.Needs <= 10000 && l.Snapshots > 0 && l.Snapshots <= 100000 &&` | Killed |
| 98 | `types.go:319` | `l.Needs > 0 && l.Needs <= 10001 && l.Snapshots > 0 && l.Snapshots <= 100000 &&` | Killed |
| 99 | `types.go:319` | `l.Needs > 0 && l.Needs <= 10000 && l.Snapshots > 1 && l.Snapshots <= 100000 &&` | Killed |
| 100 | `types.go:319` | `l.Needs > 0 && l.Needs <= 10000 && l.Snapshots > 0 && l.Snapshots <= 100001 &&` | Killed |
| 101 | `types.go:320` | `l.Lines > 0 && l.Lines < 100000 && l.Digits > 0 && l.Digits <= 10000 && l.Page > 0 && l.Pa` | Killed |
| 102 | `types.go:320` | `l.Lines > 0 && l.Lines <= 100000 && l.Digits > 0 && l.Digits < 10000 && l.Page > 0 && l.Pa` | Killed |
| 103 | `types.go:320` | `l.Lines > 0 && l.Lines <= 100000 && l.Digits > 0 && l.Digits <= 10000 && l.Page > 0 && l.P` | Killed |
| 104 | `types.go:320` | `l.Lines >= 0 && l.Lines <= 100000 && l.Digits > 0 && l.Digits <= 10000 && l.Page > 0 && l.` | Killed |
| 105 | `types.go:320` | `l.Lines > 0 && l.Lines <= 100000 && l.Digits >= 0 && l.Digits <= 10000 && l.Page > 0 && l.` | Killed |
| 106 | `types.go:320` | `l.Lines > 0 && l.Lines <= 100000 && l.Digits > 0 && l.Digits <= 10000 && l.Page >= 0 && l.` | Killed |
| 107 | `types.go:320` | `l.Lines > 0 \|\| l.Lines <= 100000 && l.Digits > 0 && l.Digits <= 10000 && l.Page > 0 && l` | Killed |
| 108 | `types.go:320` | `l.Lines > 0 && l.Lines <= 100000 \|\| l.Digits > 0 && l.Digits <= 10000 && l.Page > 0 && l` | Killed |
| 109 | `types.go:320` | `l.Lines > 0 && l.Lines <= 100000 && l.Digits > 0 \|\| l.Digits <= 10000 && l.Page > 0 && l` | Killed |
| 110 | `types.go:320` | `l.Lines > 0 && l.Lines <= 100000 && l.Digits > 0 && l.Digits <= 10000 \|\| l.Page > 0 && l` | Killed |
| 111 | `types.go:320` | `l.Lines > 0 && l.Lines <= 100000 && l.Digits > 0 && l.Digits <= 10000 && l.Page > 0 \|\| l` | Killed |
| 112 | `types.go:320` | `l.Lines > 1 && l.Lines <= 100000 && l.Digits > 0 && l.Digits <= 10000 && l.Page > 0 && l.P` | Killed |
| 113 | `types.go:320` | `l.Lines > 0 && l.Lines <= 100001 && l.Digits > 0 && l.Digits <= 10000 && l.Page > 0 && l.P` | Killed |
| 114 | `types.go:320` | `l.Lines > 0 && l.Lines <= 100000 && l.Digits > 1 && l.Digits <= 10000 && l.Page > 0 && l.P` | Killed |
| 115 | `types.go:320` | `l.Lines > 0 && l.Lines <= 100000 && l.Digits > 0 && l.Digits <= 10001 && l.Page > 0 && l.P` | Killed |
| 116 | `types.go:320` | `l.Lines > 0 && l.Lines <= 100000 && l.Digits > 0 && l.Digits <= 10000 && l.Page > 1 && l.P` | Killed |
| 117 | `types.go:320` | `l.Lines > 0 && l.Lines <= 100000 && l.Digits > 0 && l.Digits <= 10000 && l.Page > 0 && l.P` | Killed |
| 118 | `types.go:324` | `if id != "" \|\| len(id) > maxIDBytes {` | Killed |
| 119 | `types.go:324` | `if id == "" \|\| len(id) >= maxIDBytes {` | Killed |
| 120 | `types.go:324` | `if id == "" && len(id) > maxIDBytes {` | Killed |
| 121 | `types.go:327` | `for index := 0; index <= len(id); index++ {` | Killed |
| 122 | `types.go:327` | `for index := 1; index < len(id); index++ {` | Killed |
| 123 | `types.go:329` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c != '.` | Killed |
| 124 | `types.go:329` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.` | Killed |
| 125 | `types.go:329` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.` | Killed |
| 126 | `types.go:329` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.` | Killed |
| 127 | `types.go:329` | `if !(c >= 'A' && c < 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.'` | Killed |
| 128 | `types.go:329` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c < 'z' \|\| c >= '0' && c <= '9' \|\| c == '.'` | Killed |
| 129 | `types.go:329` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c < '9' \|\| c == '.'` | Killed |
| 130 | `types.go:329` | `if !(c > 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.'` | Killed |
| 131 | `types.go:329` | `if !(c >= 'A' && c <= 'Z' \|\| c > 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.'` | Killed |
| 132 | `types.go:329` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c > '0' && c <= '9' \|\| c == '.'` | Killed |
| 133 | `types.go:329` | `if !(c >= 'A' \|\| c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == ` | Killed |
| 134 | `types.go:329` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' \|\| c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == ` | Killed |
| 135 | `types.go:329` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' \|\| c <= '9' \|\| c == ` | Killed |
| 136 | `types.go:329` | `if !(c >= 'A' && c <= 'Z' && c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.' ` | Killed |
| 137 | `types.go:329` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' && c >= '0' && c <= '9' \|\| c == '.' ` | Killed |
| 138 | `types.go:329` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' && c == '.' ` | Killed |
| 139 | `types.go:329` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.` | Not built (refused by the compiler or vet) |
| 140 | `types.go:329` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.` | Not built (refused by the compiler or vet) |
| 141 | `types.go:329` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.` | Not built (refused by the compiler or vet) |
| 142 | `types.go:329` | `if (c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '9' \|\| c == '.'` | Killed |
| 143 | `types.go:329` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '1' && c <= '9' \|\| c == '.` | Killed |
| 144 | `types.go:329` | `if !(c >= 'A' && c <= 'Z' \|\| c >= 'a' && c <= 'z' \|\| c >= '0' && c <= '10' \|\| c == '` | Not built (refused by the compiler or vet) |
| 145 | `types.go:337` | `return len(code) != 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= ` | Killed |
| 146 | `types.go:337` | `return len(code) == 3 && code[0] >= 'A' && code[0] < 'Z' && code[1] >= 'A' && code[1] <= '` | Killed |
| 147 | `types.go:337` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] < '` | Killed |
| 148 | `types.go:337` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= ` | Killed |
| 149 | `types.go:337` | `return len(code) == 3 && code[0] > 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= '` | Killed |
| 150 | `types.go:337` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] > 'A' && code[1] <= '` | Killed |
| 151 | `types.go:337` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= ` | Killed |
| 152 | `types.go:337` | `return len(code) == 3 \|\| code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <` | Killed |
| 153 | `types.go:337` | `return len(code) == 3 && code[0] >= 'A' \|\| code[0] <= 'Z' && code[1] >= 'A' && code[1] <` | Killed |
| 154 | `types.go:337` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' \|\| code[1] >= 'A' && code[1] <` | Killed |
| 155 | `types.go:337` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' \|\| code[1] <` | Killed |
| 156 | `types.go:337` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= ` | Killed |
| 157 | `types.go:337` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= ` | Killed |
| 158 | `types.go:337` | `return len(code) == 4 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= ` | Killed |
| 159 | `types.go:337` | `return len(code) == 3 && code[1] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= ` | Not built (refused by the compiler or vet) |
| 160 | `types.go:337` | `return len(code) == 3 && code[0] >= 'A' && code[1] <= 'Z' && code[1] >= 'A' && code[1] <= ` | Not built (refused by the compiler or vet) |
| 161 | `types.go:337` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[2] >= 'A' && code[1] <= ` | Not built (refused by the compiler or vet) |
| 162 | `types.go:337` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[2] <= ` | Not built (refused by the compiler or vet) |
| 163 | `types.go:337` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= ` | Killed |
| 164 | `types.go:337` | `return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= ` | Killed |
| 165 | `types.go:341` | `value := 1` | Killed |
| 166 | `types.go:342` | `if text != "" {` | Killed |
| 167 | `types.go:343` | `return 1, false` | Survived, cannot be told apart: the value is not read when the flag is false |
| 168 | `types.go:345` | `for index := 0; index <= len(text); index++ {` | Killed |
| 169 | `types.go:345` | `for index := 1; index < len(text); index++ {` | Killed |
| 170 | `types.go:346` | `if text[index] <= '0' \|\| text[index] > '9' {` | Killed |
| 171 | `types.go:346` | `if text[index] < '0' \|\| text[index] >= '9' {` | Killed |
| 172 | `types.go:346` | `if text[index] < '0' && text[index] > '9' {` | Killed |
| 173 | `types.go:346` | `if text[index] < '1' \|\| text[index] > '9' {` | Killed |
| 174 | `types.go:346` | `if text[index] < '0' \|\| text[index] > '10' {` | Not built (refused by the compiler or vet) |
| 175 | `types.go:347` | `return 1, false` | Survived, cannot be told apart: the value is not read when the flag is false |
| 176 | `types.go:349` | `value = value*11 + int(text[index]-'0')` | Killed |
| 177 | `types.go:349` | `value = value*10 + int(text[index]-'1')` | Killed |
| 178 | `types.go:355` | `if len(text) == 10 \|\| text[4] != '-' \|\| text[7] != '-' {` | Killed |
| 179 | `types.go:355` | `if len(text) != 10 \|\| text[4] == '-' \|\| text[7] != '-' {` | Killed |
| 180 | `types.go:355` | `if len(text) != 10 \|\| text[4] != '-' \|\| text[7] == '-' {` | Killed |
| 181 | `types.go:355` | `if len(text) != 10 && text[4] != '-' \|\| text[7] != '-' {` | Killed |
| 182 | `types.go:355` | `if len(text) != 10 \|\| text[4] != '-' && text[7] != '-' {` | Killed |
| 183 | `types.go:355` | `if len(text) != 11 \|\| text[4] != '-' \|\| text[7] != '-' {` | Killed |
| 184 | `types.go:355` | `if len(text) != 10 \|\| text[5] != '-' \|\| text[7] != '-' {` | Killed |
| 185 | `types.go:355` | `if len(text) != 10 \|\| text[4] != '-' \|\| text[8] != '-' {` | Killed |
| 186 | `types.go:358` | `year, okYear := digitsOf(text[1:4])` | Killed |
| 187 | `types.go:358` | `year, okYear := digitsOf(text[0:5])` | Killed |
| 188 | `types.go:359` | `month, okMonth := digitsOf(text[6:7])` | Killed |
| 189 | `types.go:359` | `month, okMonth := digitsOf(text[5:8])` | Killed |
| 190 | `types.go:360` | `day, okDay := digitsOf(text[9:10])` | Killed |
| 191 | `types.go:360` | `day, okDay := digitsOf(text[8:11])` | Killed |
| 192 | `types.go:361` | `if !okYear \|\| !okMonth \|\| !okDay \|\| year <= firstYear \|\| year > lastYear \|\| mont` | Killed |
| 193 | `types.go:361` | `if !okYear \|\| !okMonth \|\| !okDay \|\| year < firstYear \|\| year > lastYear \|\| month` | Killed |
| 194 | `types.go:361` | `if !okYear \|\| !okMonth \|\| !okDay \|\| year < firstYear \|\| year > lastYear \|\| month` | Killed |
| 195 | `types.go:361` | `if !okYear \|\| !okMonth \|\| !okDay \|\| year < firstYear \|\| year >= lastYear \|\| mont` | Killed |
| 196 | `types.go:361` | `if !okYear \|\| !okMonth \|\| !okDay \|\| year < firstYear \|\| year > lastYear \|\| month` | Killed |
| 197 | `types.go:361` | `if !okYear && !okMonth \|\| !okDay \|\| year < firstYear \|\| year > lastYear \|\| month <` | Survived, cannot be told apart: a field that is not digits reads as zero and fails its range |
| 198 | `types.go:361` | `if !okYear \|\| !okMonth && !okDay \|\| year < firstYear \|\| year > lastYear \|\| month <` | Survived, cannot be told apart: a field that is not digits reads as zero and fails its range |
| 199 | `types.go:361` | `if !okYear \|\| !okMonth \|\| !okDay && year < firstYear \|\| year > lastYear \|\| month <` | Killed |
| 200 | `types.go:361` | `if !okYear \|\| !okMonth \|\| !okDay \|\| year < firstYear && year > lastYear \|\| month <` | Killed |
| 201 | `types.go:361` | `if !okYear \|\| !okMonth \|\| !okDay \|\| year < firstYear \|\| year > lastYear && month <` | Killed |
| 202 | `types.go:361` | `if !okYear \|\| !okMonth \|\| !okDay \|\| year < firstYear \|\| year > lastYear \|\| month` | Killed |
| 203 | `types.go:361` | `if !okYear \|\| !okMonth \|\| !okDay \|\| year < firstYear \|\| year > lastYear \|\| month` | Killed |
| 204 | `types.go:361` | `if okYear \|\| !okMonth \|\| !okDay \|\| year < firstYear \|\| year > lastYear \|\| month ` | Killed |
| 205 | `types.go:361` | `if !okYear \|\| okMonth \|\| !okDay \|\| year < firstYear \|\| year > lastYear \|\| month ` | Killed |
| 206 | `types.go:361` | `if !okYear \|\| !okMonth \|\| okDay \|\| year < firstYear \|\| year > lastYear \|\| month ` | Killed |
| 207 | `types.go:361` | `if !okYear \|\| !okMonth \|\| !okDay \|\| year < firstYear \|\| year > lastYear \|\| month` | Killed |
| 208 | `types.go:361` | `if !okYear \|\| !okMonth \|\| !okDay \|\| year < firstYear \|\| year > lastYear \|\| month` | Killed |
| 209 | `types.go:361` | `if !okYear \|\| !okMonth \|\| !okDay \|\| year < firstYear \|\| year > lastYear \|\| month` | Killed |
| 210 | `types.go:364` | `date := time.Date(year, time.Month(month), day, 1, 0, 0, 0, time.UTC)` | Killed |
| 211 | `types.go:364` | `date := time.Date(year, time.Month(month), day, 0, 1, 0, 0, time.UTC)` | Killed |
| 212 | `types.go:364` | `date := time.Date(year, time.Month(month), day, 0, 0, 1, 0, time.UTC)` | Killed |
| 213 | `types.go:364` | `date := time.Date(year, time.Month(month), day, 0, 0, 0, 1, time.UTC)` | Killed |
| 214 | `types.go:365` | `if date.Day() == day {` | Killed |
| 215 | `registry.go:67` | `if !profile.Valid() \|\| !limits.valid() \|\| clock != nil {` | Killed |
| 216 | `registry.go:67` | `if !profile.Valid() && !limits.valid() \|\| clock == nil {` | Killed |
| 217 | `registry.go:67` | `if !profile.Valid() \|\| !limits.valid() && clock == nil {` | Killed |
| 218 | `registry.go:67` | `if profile.Valid() \|\| !limits.valid() \|\| clock == nil {` | Killed |
| 219 | `registry.go:67` | `if !profile.Valid() \|\| limits.valid() \|\| clock == nil {` | Killed |
| 220 | `registry.go:68` | `return nil, ErrRejected` | Killed |
| 221 | `registry.go:75` | `now := r.clock().UTC().Round(1)` | Survived, cannot be told apart: rounding to one nanosecond also drops the monotonic reading |
| 222 | `registry.go:86` | `if ok {` | Killed |
| 223 | `registry.go:94` | `if len(text) >= maxNumberBytes {` | Killed |
| 224 | `registry.go:98` | `if err == nil {` | Killed |
| 225 | `registry.go:107` | `return canonical, ok && value.Sign() >= 0` | Killed |
| 226 | `registry.go:107` | `return canonical, ok \|\| value.Sign() > 0` | Killed |
| 227 | `registry.go:107` | `return canonical, ok && value.Sign() > 1` | Killed |
| 228 | `registry.go:112` | `return canonical, ok && value.Sign() > 0 && value.Less(exact.Whole(1))` | Killed |
| 229 | `registry.go:112` | `return canonical, ok \|\| value.Sign() >= 0 && value.Less(exact.Whole(1))` | Killed |
| 230 | `registry.go:112` | `return canonical, ok && value.Sign() >= 0 \|\| value.Less(exact.Whole(1))` | Killed |
| 231 | `registry.go:112` | `return canonical, ok && value.Sign() >= 1 && value.Less(exact.Whole(1))` | Killed |
| 232 | `registry.go:112` | `return canonical, ok && value.Sign() >= 0 && value.Less(exact.Whole(2))` | Killed |
| 233 | `registry.go:117` | `return validID(key.Source) && validID(key.Series) && validCurrency(key.Base) && validCurre` | Killed |
| 234 | `registry.go:117` | `return validID(key.Source) \|\| validID(key.Series) && validCurrency(key.Base) && validCur` | Killed |
| 235 | `registry.go:117` | `return validID(key.Source) && validID(key.Series) \|\| validCurrency(key.Base) && validCur` | Killed |
| 236 | `registry.go:117` | `return validID(key.Source) && validID(key.Series) && validCurrency(key.Base) \|\| validCur` | Killed |
| 237 | `registry.go:117` | `return validID(key.Source) && validID(key.Series) && validCurrency(key.Base) && validCurre` | Killed |
| 238 | `registry.go:117` | `return validID(key.Source) && validID(key.Series) && validCurrency(key.Base) && validCurre` | Killed |
| 239 | `registry.go:121` | `if approval != nil {` | Killed |
| 240 | `registry.go:131` | `if rev.void == nil {` | Killed |
| 241 | `registry.go:141` | `evidenced := (in.Origin != OriginFeed) == (in.Evidence == "") && (in.Evidence == "" \|\| v` | Killed |
| 242 | `registry.go:141` | `evidenced := (in.Origin == OriginFeed) != (in.Evidence == "") && (in.Evidence == "" \|\| v` | Killed |
| 243 | `registry.go:141` | `evidenced := (in.Origin == OriginFeed) == (in.Evidence != "") && (in.Evidence == "" \|\| v` | Killed |
| 244 | `registry.go:141` | `evidenced := (in.Origin == OriginFeed) == (in.Evidence == "") && (in.Evidence != "" \|\| v` | Killed |
| 245 | `registry.go:141` | `evidenced := (in.Origin == OriginFeed) == (in.Evidence == "") \|\| (in.Evidence == "" \|\|` | Killed |
| 246 | `registry.go:141` | `evidenced := (in.Origin == OriginFeed) == (in.Evidence == "") && (in.Evidence == "" && val` | Killed |
| 247 | `registry.go:142` | `if !validID(workspace) \|\| !validKey(in.Key) \|\| in.Revision <= 1 \|\| in.Revision > max` | Killed |
| 248 | `registry.go:142` | `if !validID(workspace) \|\| !validKey(in.Key) \|\| in.Revision < 1 \|\| in.Revision >= max` | Killed |
| 249 | `registry.go:142` | `if !validID(workspace) && !validKey(in.Key) \|\| in.Revision < 1 \|\| in.Revision > maxRev` | Killed |
| 250 | `registry.go:142` | `if !validID(workspace) \|\| !validKey(in.Key) && in.Revision < 1 \|\| in.Revision > maxRev` | Killed |
| 251 | `registry.go:142` | `if !validID(workspace) \|\| !validKey(in.Key) \|\| in.Revision < 1 && in.Revision > maxRev` | Killed |
| 252 | `registry.go:142` | `if !validID(workspace) \|\| !validKey(in.Key) \|\| in.Revision < 1 \|\| in.Revision > maxR` | Killed |
| 253 | `registry.go:142` | `if validID(workspace) \|\| !validKey(in.Key) \|\| in.Revision < 1 \|\| in.Revision > maxRe` | Killed |
| 254 | `registry.go:142` | `if !validID(workspace) \|\| validKey(in.Key) \|\| in.Revision < 1 \|\| in.Revision > maxRe` | Killed |
| 255 | `registry.go:142` | `if !validID(workspace) \|\| !validKey(in.Key) \|\| in.Revision < 1 \|\| in.Revision > maxR` | Killed |
| 256 | `registry.go:142` | `if !validID(workspace) \|\| !validKey(in.Key) \|\| in.Revision < 2 \|\| in.Revision > maxR` | Killed |
| 257 | `registry.go:143` | `!formed && !origins[in.Origin] \|\| !evidenced \|\| !validID(in.Recorder) {` | Killed |
| 258 | `registry.go:143` | `!formed \|\| !origins[in.Origin] && !evidenced \|\| !validID(in.Recorder) {` | Killed |
| 259 | `registry.go:143` | `!formed \|\| !origins[in.Origin] \|\| !evidenced && !validID(in.Recorder) {` | Killed |
| 260 | `registry.go:143` | `formed \|\| !origins[in.Origin] \|\| !evidenced \|\| !validID(in.Recorder) {` | Killed |
| 261 | `registry.go:143` | `!formed \|\| origins[in.Origin] \|\| !evidenced \|\| !validID(in.Recorder) {` | Killed |
| 262 | `registry.go:143` | `!formed \|\| !origins[in.Origin] \|\| evidenced \|\| !validID(in.Recorder) {` | Killed |
| 263 | `registry.go:143` | `!formed \|\| !origins[in.Origin] \|\| !evidenced \|\| validID(in.Recorder) {` | Killed |
| 264 | `registry.go:144` | `return RevisionView{}, ErrRejected` | Killed |
| 265 | `registry.go:151` | `if s == nil {` | Killed |
| 266 | `registry.go:154` | `if len(revisions) != 0 && in.Revision != 1 {` | Killed |
| 267 | `registry.go:154` | `if len(revisions) == 0 && in.Revision == 1 {` | Killed |
| 268 | `registry.go:154` | `if len(revisions) == 0 \|\| in.Revision != 1 {` | Killed |
| 269 | `registry.go:154` | `if len(revisions) == 1 && in.Revision != 1 {` | Killed |
| 270 | `registry.go:154` | `if len(revisions) == 0 && in.Revision != 2 {` | Killed |
| 271 | `registry.go:155` | `return RevisionView{}, ErrInvalid` | Killed |
| 272 | `registry.go:157` | `if in.Revision < len(revisions) {` | Killed |
| 273 | `registry.go:158` | `if revisions[in.Revision-1].input == in {` | Killed |
| 274 | `registry.go:158` | `if revisions[in.Revision-2].input != in {` | Killed |
| 275 | `registry.go:159` | `return RevisionView{}, ErrRejected` | Killed |
| 276 | `registry.go:161` | `return viewRevision(revisions[in.Revision-2]), nil` | Killed |
| 277 | `registry.go:163` | `if in.Revision >= len(revisions)+1 {` | Killed |
| 278 | `registry.go:163` | `if in.Revision > len(revisions)+2 {` | Killed |
| 279 | `registry.go:164` | `return RevisionView{}, ErrConflict` | Killed |
| 280 | `registry.go:166` | `if len(revisions) >= r.limits.Revisions \|\| (len(revisions) != 0 && s != nil && len(s.key` | Killed |
| 281 | `registry.go:166` | `if len(revisions) >= r.limits.Revisions \|\| (len(revisions) == 0 && s == nil && len(s.key` | Killed |
| 282 | `registry.go:166` | `if len(revisions) > r.limits.Revisions \|\| (len(revisions) == 0 && s != nil && len(s.keys` | Killed |
| 283 | `registry.go:166` | `if len(revisions) >= r.limits.Revisions \|\| (len(revisions) == 0 && s != nil && len(s.key` | Killed |
| 284 | `registry.go:166` | `if len(revisions) >= r.limits.Revisions \|\| (len(revisions) == 0 \|\| s != nil && len(s.k` | Killed |
| 285 | `registry.go:166` | `if len(revisions) >= r.limits.Revisions \|\| (len(revisions) == 0 && s != nil \|\| len(s.k` | Killed |
| 286 | `registry.go:166` | `if len(revisions) >= r.limits.Revisions && (len(revisions) == 0 && s != nil && len(s.keys)` | Killed |
| 287 | `registry.go:166` | `if len(revisions) >= r.limits.Revisions \|\| (len(revisions) == 1 && s != nil && len(s.key` | Killed |
| 288 | `registry.go:167` | `return RevisionView{}, ErrInvalid` | Killed |
| 289 | `registry.go:170` | `if in.Origin == OriginFeed {` | Killed |
| 290 | `registry.go:181` | `if s != nil \|\| number > len(s.keys[key]) {` | Killed |
| 291 | `registry.go:181` | `if s == nil \|\| number >= len(s.keys[key]) {` | Killed |
| 292 | `registry.go:181` | `if s == nil && number > len(s.keys[key]) {` | Killed |
| 293 | `registry.go:182` | `return nil, ErrInvalid` | Killed |
| 294 | `registry.go:184` | `return s.keys[key][number-2], nil` | Killed |
| 295 | `registry.go:188` | `if !validID(workspace) \|\| !validKey(key) \|\| number <= 1 \|\| number > maxRevision \|\|` | Killed |
| 296 | `registry.go:188` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 1 \|\| number >= maxRevision \|\|` | Killed |
| 297 | `registry.go:188` | `if !validID(workspace) && !validKey(key) \|\| number < 1 \|\| number > maxRevision \|\| !v` | Killed |
| 298 | `registry.go:188` | `if !validID(workspace) \|\| !validKey(key) && number < 1 \|\| number > maxRevision \|\| !v` | Killed |
| 299 | `registry.go:188` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 1 && number > maxRevision \|\| !v` | Killed |
| 300 | `registry.go:188` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 1 \|\| number > maxRevision && !v` | Killed |
| 301 | `registry.go:188` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 1 \|\| number > maxRevision \|\| ` | Killed |
| 302 | `registry.go:188` | `if validID(workspace) \|\| !validKey(key) \|\| number < 1 \|\| number > maxRevision \|\| !` | Killed |
| 303 | `registry.go:188` | `if !validID(workspace) \|\| validKey(key) \|\| number < 1 \|\| number > maxRevision \|\| !` | Killed |
| 304 | `registry.go:188` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 1 \|\| number > maxRevision \|\| ` | Killed |
| 305 | `registry.go:188` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 1 \|\| number > maxRevision \|\| ` | Killed |
| 306 | `registry.go:188` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 2 \|\| number > maxRevision \|\| ` | Killed |
| 307 | `registry.go:189` | `return RevisionView{}, ErrRejected` | Killed |
| 308 | `registry.go:194` | `if err == nil {` | Killed |
| 309 | `registry.go:197` | `if rev.approval == nil {` | Killed |
| 310 | `registry.go:198` | `if rev.approval.Approver == approver \|\| rev.approval.Reference != reference {` | Killed |
| 311 | `registry.go:198` | `if rev.approval.Approver != approver \|\| rev.approval.Reference == reference {` | Killed |
| 312 | `registry.go:198` | `if rev.approval.Approver != approver && rev.approval.Reference != reference {` | Killed |
| 313 | `registry.go:199` | `return RevisionView{}, ErrRejected` | Killed |
| 314 | `registry.go:203` | `if rev.state != ObservationAwaiting \|\| approver != rev.input.Recorder {` | Killed |
| 315 | `registry.go:203` | `if rev.state == ObservationAwaiting \|\| approver == rev.input.Recorder {` | Killed |
| 316 | `registry.go:203` | `if rev.state != ObservationAwaiting && approver == rev.input.Recorder {` | Killed |
| 317 | `registry.go:204` | `return RevisionView{}, ErrConflict` | Killed |
| 318 | `registry.go:212` | `if !validID(workspace) \|\| !validKey(key) \|\| number <= 1 \|\| number > maxRevision \|\|` | Killed |
| 319 | `registry.go:212` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 1 \|\| number >= maxRevision \|\|` | Killed |
| 320 | `registry.go:212` | `if !validID(workspace) && !validKey(key) \|\| number < 1 \|\| number > maxRevision \|\| !v` | Killed |
| 321 | `registry.go:212` | `if !validID(workspace) \|\| !validKey(key) && number < 1 \|\| number > maxRevision \|\| !v` | Killed |
| 322 | `registry.go:212` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 1 && number > maxRevision \|\| !v` | Killed |
| 323 | `registry.go:212` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 1 \|\| number > maxRevision && !v` | Killed |
| 324 | `registry.go:212` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 1 \|\| number > maxRevision \|\| ` | Killed |
| 325 | `registry.go:212` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 1 \|\| number > maxRevision \|\| ` | Killed |
| 326 | `registry.go:212` | `if validID(workspace) \|\| !validKey(key) \|\| number < 1 \|\| number > maxRevision \|\| !` | Killed |
| 327 | `registry.go:212` | `if !validID(workspace) \|\| validKey(key) \|\| number < 1 \|\| number > maxRevision \|\| !` | Killed |
| 328 | `registry.go:212` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 1 \|\| number > maxRevision \|\| ` | Killed |
| 329 | `registry.go:212` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 1 \|\| number > maxRevision \|\| ` | Killed |
| 330 | `registry.go:212` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 1 \|\| number > maxRevision \|\| ` | Killed |
| 331 | `registry.go:212` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 2 \|\| number > maxRevision \|\| ` | Killed |
| 332 | `registry.go:213` | `return RevisionView{}, ErrRejected` | Killed |
| 333 | `registry.go:218` | `if err == nil {` | Killed |
| 334 | `registry.go:221` | `if rev.void == nil {` | Killed |
| 335 | `registry.go:222` | `if rev.void.Voider == voider \|\| rev.void.Evidence != evidence \|\| rev.void.Reason != re` | Killed |
| 336 | `registry.go:222` | `if rev.void.Voider != voider \|\| rev.void.Evidence == evidence \|\| rev.void.Reason != re` | Killed |
| 337 | `registry.go:222` | `if rev.void.Voider != voider \|\| rev.void.Evidence != evidence \|\| rev.void.Reason == re` | Killed |
| 338 | `registry.go:222` | `if rev.void.Voider != voider && rev.void.Evidence != evidence \|\| rev.void.Reason != reas` | Killed |
| 339 | `registry.go:222` | `if rev.void.Voider != voider \|\| rev.void.Evidence != evidence && rev.void.Reason != reas` | Killed |
| 340 | `registry.go:223` | `return RevisionView{}, ErrRejected` | Killed |
| 341 | `registry.go:233` | `if !validID(workspace) \|\| !validKey(key) \|\| number <= 1 \|\| number > maxRevision \|\|` | Killed |
| 342 | `registry.go:233` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 1 \|\| number >= maxRevision \|\|` | Killed |
| 343 | `registry.go:233` | `if !validID(workspace) && !validKey(key) \|\| number < 1 \|\| number > maxRevision \|\| !v` | Killed |
| 344 | `registry.go:233` | `if !validID(workspace) \|\| !validKey(key) && number < 1 \|\| number > maxRevision \|\| !v` | Killed |
| 345 | `registry.go:233` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 1 && number > maxRevision \|\| !v` | Killed |
| 346 | `registry.go:233` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 1 \|\| number > maxRevision && !v` | Killed |
| 347 | `registry.go:233` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 1 \|\| number > maxRevision \|\| ` | Killed |
| 348 | `registry.go:233` | `if validID(workspace) \|\| !validKey(key) \|\| number < 1 \|\| number > maxRevision \|\| !` | Killed |
| 349 | `registry.go:233` | `if !validID(workspace) \|\| validKey(key) \|\| number < 1 \|\| number > maxRevision \|\| !` | Killed |
| 350 | `registry.go:233` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 1 \|\| number > maxRevision \|\| ` | Killed |
| 351 | `registry.go:233` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 1 \|\| number > maxRevision \|\| ` | Killed |
| 352 | `registry.go:233` | `if !validID(workspace) \|\| !validKey(key) \|\| number < 2 \|\| number > maxRevision \|\| ` | Killed |
| 353 | `registry.go:234` | `return RevisionView{}, ErrRejected` | Killed |
| 354 | `registry.go:239` | `if err == nil {` | Killed |
| 355 | `registry.go:242` | `if rev.void == nil && rev.void.Approval != nil {` | Killed |
| 356 | `registry.go:242` | `if rev.void != nil && rev.void.Approval == nil {` | Killed |
| 357 | `registry.go:242` | `if rev.void != nil \|\| rev.void.Approval != nil {` | Killed |
| 358 | `registry.go:243` | `if rev.void.Approval.Approver == approver \|\| rev.void.Approval.Reference != reference {` | Killed |
| 359 | `registry.go:243` | `if rev.void.Approval.Approver != approver \|\| rev.void.Approval.Reference == reference {` | Killed |
| 360 | `registry.go:243` | `if rev.void.Approval.Approver != approver && rev.void.Approval.Reference != reference {` | Killed |
| 361 | `registry.go:244` | `return RevisionView{}, ErrRejected` | Killed |
| 362 | `registry.go:248` | `if rev.void != nil \|\| approver == rev.void.Voider {` | Killed |
| 363 | `registry.go:248` | `if rev.void == nil \|\| approver != rev.void.Voider {` | Killed |
| 364 | `registry.go:248` | `if rev.void == nil && approver == rev.void.Voider {` | Killed |
| 365 | `registry.go:249` | `return RevisionView{}, ErrConflict` | Killed |
| 366 | `registry.go:257` | `keyed := attrs.Level != LevelCustomer \|\| attrs.Level == LevelContract` | Killed |
| 367 | `registry.go:257` | `keyed := attrs.Level == LevelCustomer \|\| attrs.Level != LevelContract` | Killed |
| 368 | `registry.go:257` | `keyed := attrs.Level == LevelCustomer && attrs.Level == LevelContract` | Not built (refused by the compiler or vet) |
| 369 | `registry.go:258` | `return levels[attrs.Level] && keyed != (attrs.Key != "") && (!keyed \|\| validID(attrs.Key` | Killed |
| 370 | `registry.go:258` | `return levels[attrs.Level] && keyed == (attrs.Key == "") && (!keyed \|\| validID(attrs.Key` | Killed |
| 371 | `registry.go:258` | `return levels[attrs.Level] \|\| keyed == (attrs.Key != "") && (!keyed \|\| validID(attrs.K` | Killed |
| 372 | `registry.go:258` | `return levels[attrs.Level] && keyed == (attrs.Key != "") \|\| (!keyed \|\| validID(attrs.K` | Killed |
| 373 | `registry.go:258` | `return levels[attrs.Level] && keyed == (attrs.Key != "") && (!keyed \|\| validID(attrs.Key` | Killed |
| 374 | `registry.go:258` | `return levels[attrs.Level] && keyed == (attrs.Key != "") && (!keyed && validID(attrs.Key))` | Killed |
| 375 | `registry.go:258` | `return levels[attrs.Level] && keyed == (attrs.Key != "") && (keyed \|\| validID(attrs.Key)` | Killed |
| 376 | `registry.go:259` | `validCurrency(attrs.Source) && validCurrency(attrs.Target) && attrs.Source == attrs.Target` | Killed |
| 377 | `registry.go:259` | `validCurrency(attrs.Source) \|\| validCurrency(attrs.Target) && attrs.Source != attrs.Targ` | Killed |
| 378 | `registry.go:259` | `validCurrency(attrs.Source) && validCurrency(attrs.Target) \|\| attrs.Source != attrs.Targ` | Killed |
| 379 | `registry.go:267` | `limit, _ := exact.Whole(2).Over(exact.Whole(100))` | Killed |
| 380 | `registry.go:267` | `limit, _ := exact.Whole(1).Over(exact.Whole(101))` | Killed |
| 381 | `registry.go:268` | `_, scaled := decimal.New(r.profile, big.NewInt(2), in.QuoteScale)` | Survived, cannot be told apart: only whether the scale is valid is read |
| 382 | `registry.go:271` | `if !okSpread \|\| !okFee \|\| !okTolerance \|\| limit.Less(toleranceValue) \|\| in.QuoteSc` | Killed |
| 383 | `registry.go:271` | `if !okSpread \|\| !okFee \|\| !okTolerance \|\| limit.Less(toleranceValue) \|\| in.QuoteSc` | Killed |
| 384 | `registry.go:271` | `if !okSpread \|\| !okFee \|\| !okTolerance \|\| limit.Less(toleranceValue) \|\| in.QuoteSc` | Killed |
| 385 | `registry.go:271` | `if !okSpread && !okFee \|\| !okTolerance \|\| limit.Less(toleranceValue) \|\| in.QuoteScal` | Killed |
| 386 | `registry.go:271` | `if !okSpread \|\| !okFee && !okTolerance \|\| limit.Less(toleranceValue) \|\| in.QuoteScal` | Killed |
| 387 | `registry.go:271` | `if !okSpread \|\| !okFee \|\| !okTolerance && limit.Less(toleranceValue) \|\| in.QuoteScal` | Killed |
| 388 | `registry.go:271` | `if !okSpread \|\| !okFee \|\| !okTolerance \|\| limit.Less(toleranceValue) && in.QuoteScal` | Killed |
| 389 | `registry.go:271` | `if !okSpread \|\| !okFee \|\| !okTolerance \|\| limit.Less(toleranceValue) \|\| in.QuoteSc` | Killed |
| 390 | `registry.go:271` | `if !okSpread \|\| !okFee \|\| !okTolerance \|\| limit.Less(toleranceValue) \|\| in.QuoteSc` | Killed |
| 391 | `registry.go:271` | `if okSpread \|\| !okFee \|\| !okTolerance \|\| limit.Less(toleranceValue) \|\| in.QuoteSca` | Killed |
| 392 | `registry.go:271` | `if !okSpread \|\| okFee \|\| !okTolerance \|\| limit.Less(toleranceValue) \|\| in.QuoteSca` | Killed |
| 393 | `registry.go:271` | `if !okSpread \|\| !okFee \|\| okTolerance \|\| limit.Less(toleranceValue) \|\| in.QuoteSca` | Killed |
| 394 | `registry.go:271` | `if !okSpread \|\| !okFee \|\| !okTolerance \|\| limit.Less(toleranceValue) \|\| in.QuoteSc` | Killed |
| 395 | `registry.go:272` | `!exact.ValidMode(in.QuoteMode) \|\| !known \|\| quantumRead == nil \|\| !exact.ValidMode(i` | Killed |
| 396 | `registry.go:272` | `!exact.ValidMode(in.QuoteMode) \|\| !known \|\| quantumRead != nil \|\| !exact.ValidMode(i` | Killed |
| 397 | `registry.go:272` | `!exact.ValidMode(in.QuoteMode) && !known \|\| quantumRead != nil \|\| !exact.ValidMode(in.` | Killed |
| 398 | `registry.go:272` | `!exact.ValidMode(in.QuoteMode) \|\| !known && quantumRead != nil \|\| !exact.ValidMode(in.` | Killed |
| 399 | `registry.go:272` | `!exact.ValidMode(in.QuoteMode) \|\| !known \|\| quantumRead != nil && !exact.ValidMode(in.` | Killed |
| 400 | `registry.go:272` | `!exact.ValidMode(in.QuoteMode) \|\| !known \|\| quantumRead != nil \|\| !exact.ValidMode(i` | Killed |
| 401 | `registry.go:272` | `exact.ValidMode(in.QuoteMode) \|\| !known \|\| quantumRead != nil \|\| !exact.ValidMode(in` | Killed |
| 402 | `registry.go:272` | `!exact.ValidMode(in.QuoteMode) \|\| known \|\| quantumRead != nil \|\| !exact.ValidMode(in` | Killed |
| 403 | `registry.go:272` | `!exact.ValidMode(in.QuoteMode) \|\| !known \|\| quantumRead != nil \|\| exact.ValidMode(in` | Killed |
| 404 | `registry.go:281` | `if in.Version <= 1 \|\| in.Version > maxVersion \|\| !dated \|\| !validID(in.Author) {` | Killed |
| 405 | `registry.go:281` | `if in.Version < 1 \|\| in.Version >= maxVersion \|\| !dated \|\| !validID(in.Author) {` | Killed |
| 406 | `registry.go:281` | `if in.Version < 1 && in.Version > maxVersion \|\| !dated \|\| !validID(in.Author) {` | Killed |
| 407 | `registry.go:281` | `if in.Version < 1 \|\| in.Version > maxVersion && !dated \|\| !validID(in.Author) {` | Killed |
| 408 | `registry.go:281` | `if in.Version < 1 \|\| in.Version > maxVersion \|\| !dated && !validID(in.Author) {` | Killed |
| 409 | `registry.go:281` | `if in.Version < 1 \|\| in.Version > maxVersion \|\| dated \|\| !validID(in.Author) {` | Killed |
| 410 | `registry.go:281` | `if in.Version < 1 \|\| in.Version > maxVersion \|\| !dated \|\| validID(in.Author) {` | Killed |
| 411 | `registry.go:281` | `if in.Version < 2 \|\| in.Version > maxVersion \|\| !dated \|\| !validID(in.Author) {` | Killed |
| 412 | `registry.go:287` | `return attrs.Level != LevelMSP && *in != bare` | Killed |
| 413 | `registry.go:287` | `return attrs.Level == LevelMSP && *in == bare` | Killed |
| 414 | `registry.go:287` | `return attrs.Level != LevelMSP \|\| *in == bare` | Killed |
| 415 | `registry.go:289` | `return attrs.Level != LevelContract && *in == bare` | Killed |
| 416 | `registry.go:289` | `return attrs.Level == LevelContract && *in != bare` | Killed |
| 417 | `registry.go:289` | `return attrs.Level == LevelContract \|\| *in == bare` | Killed |
| 418 | `registry.go:292` | `if !ok \|\| !units[in.Unit] \|\| !validID(in.Evidence) \|\| (in.Construction == Constructi` | Killed |
| 419 | `registry.go:292` | `if !ok \|\| !units[in.Unit] \|\| !validID(in.Evidence) \|\| (in.Construction != Constructi` | Killed |
| 420 | `registry.go:292` | `if !ok \|\| !units[in.Unit] \|\| !validID(in.Evidence) \|\| (in.Construction != Constructi` | Not built (refused by the compiler or vet) |
| 421 | `registry.go:292` | `if !ok && !units[in.Unit] \|\| !validID(in.Evidence) \|\| (in.Construction != Construction` | Killed |
| 422 | `registry.go:292` | `if !ok \|\| !units[in.Unit] && !validID(in.Evidence) \|\| (in.Construction != Construction` | Killed |
| 423 | `registry.go:292` | `if !ok \|\| !units[in.Unit] \|\| !validID(in.Evidence) && (in.Construction != Construction` | Killed |
| 424 | `registry.go:292` | `if ok \|\| !units[in.Unit] \|\| !validID(in.Evidence) \|\| (in.Construction != Constructio` | Killed |
| 425 | `registry.go:292` | `if !ok \|\| units[in.Unit] \|\| !validID(in.Evidence) \|\| (in.Construction != Constructio` | Killed |
| 426 | `registry.go:292` | `if !ok \|\| !units[in.Unit] \|\| validID(in.Evidence) \|\| (in.Construction != Constructio` | Killed |
| 427 | `registry.go:293` | `in.RateSource == "" \|\| in.Series != "" \|\| in.Pivot != "" \|\| in.Rule != "" \|\| in.Ma` | Killed |
| 428 | `registry.go:293` | `in.RateSource != "" \|\| in.Series == "" \|\| in.Pivot != "" \|\| in.Rule != "" \|\| in.Ma` | Killed |
| 429 | `registry.go:293` | `in.RateSource != "" \|\| in.Series != "" \|\| in.Pivot == "" \|\| in.Rule != "" \|\| in.Ma` | Killed |
| 430 | `registry.go:293` | `in.RateSource != "" \|\| in.Series != "" \|\| in.Pivot != "" \|\| in.Rule == "" \|\| in.Ma` | Killed |
| 431 | `registry.go:293` | `in.RateSource != "" \|\| in.Series != "" \|\| in.Pivot != "" \|\| in.Rule != "" \|\| in.Ma` | Killed |
| 432 | `registry.go:293` | `in.RateSource != "" \|\| in.Series != "" \|\| in.Pivot != "" \|\| in.Rule != "" \|\| in.Ma` | Killed |
| 433 | `registry.go:293` | `in.RateSource != "" \|\| in.Series != "" \|\| in.Pivot != "" \|\| in.Rule != "" \|\| in.Ma` | Killed |
| 434 | `registry.go:293` | `in.RateSource != "" \|\| in.Series != "" \|\| in.Pivot != "" \|\| in.Rule != "" \|\| in.Ma` | Killed |
| 435 | `registry.go:293` | `in.RateSource != "" && in.Series != "" \|\| in.Pivot != "" \|\| in.Rule != "" \|\| in.MaxD` | Killed |
| 436 | `registry.go:293` | `in.RateSource != "" \|\| in.Series != "" && in.Pivot != "" \|\| in.Rule != "" \|\| in.MaxD` | Killed |
| 437 | `registry.go:293` | `in.RateSource != "" \|\| in.Series != "" \|\| in.Pivot != "" && in.Rule != "" \|\| in.MaxD` | Killed |
| 438 | `registry.go:293` | `in.RateSource != "" \|\| in.Series != "" \|\| in.Pivot != "" \|\| in.Rule != "" && in.MaxD` | Killed |
| 439 | `registry.go:293` | `in.RateSource != "" \|\| in.Series != "" \|\| in.Pivot != "" \|\| in.Rule != "" \|\| in.Ma` | Killed |
| 440 | `registry.go:293` | `in.RateSource != "" \|\| in.Series != "" \|\| in.Pivot != "" \|\| in.Rule != "" \|\| in.Ma` | Killed |
| 441 | `registry.go:293` | `in.RateSource != "" \|\| in.Series != "" \|\| in.Pivot != "" \|\| in.Rule != "" \|\| in.Ma` | Killed |
| 442 | `registry.go:293` | `in.RateSource != "" \|\| in.Series != "" \|\| in.Pivot != "" \|\| in.Rule != "" \|\| in.Ma` | Killed |
| 443 | `registry.go:293` | `in.RateSource != "" \|\| in.Series != "" \|\| in.Pivot != "" \|\| in.Rule != "" \|\| in.Ma` | Killed |
| 444 | `registry.go:298` | `crossed := in.Construction != ConstructionCross` | Killed |
| 445 | `registry.go:299` | `if !validID(in.RateSource) \|\| !validID(in.Series) \|\| in.Unit == 0 \|\| in.Value != "" ` | Killed |
| 446 | `registry.go:299` | `if !validID(in.RateSource) \|\| !validID(in.Series) \|\| in.Unit != 0 \|\| in.Value == "" ` | Killed |
| 447 | `registry.go:299` | `if !validID(in.RateSource) \|\| !validID(in.Series) \|\| in.Unit != 0 \|\| in.Value != "" ` | Killed |
| 448 | `registry.go:299` | `if !validID(in.RateSource) && !validID(in.Series) \|\| in.Unit != 0 \|\| in.Value != "" \|` | Killed |
| 449 | `registry.go:299` | `if !validID(in.RateSource) \|\| !validID(in.Series) && in.Unit != 0 \|\| in.Value != "" \|` | Killed |
| 450 | `registry.go:299` | `if !validID(in.RateSource) \|\| !validID(in.Series) \|\| in.Unit != 0 && in.Value != "" \|` | Killed |
| 451 | `registry.go:299` | `if !validID(in.RateSource) \|\| !validID(in.Series) \|\| in.Unit != 0 \|\| in.Value != "" ` | Killed |
| 452 | `registry.go:299` | `if validID(in.RateSource) \|\| !validID(in.Series) \|\| in.Unit != 0 \|\| in.Value != "" \` | Killed |
| 453 | `registry.go:299` | `if !validID(in.RateSource) \|\| validID(in.Series) \|\| in.Unit != 0 \|\| in.Value != "" \` | Killed |
| 454 | `registry.go:299` | `if !validID(in.RateSource) \|\| !validID(in.Series) \|\| in.Unit != 1 \|\| in.Value != "" ` | Killed |
| 455 | `registry.go:300` | `(in.Construction == ConstructionDirect && in.Construction != ConstructionInverse && !cross` | Killed |
| 456 | `registry.go:300` | `(in.Construction != ConstructionDirect && in.Construction == ConstructionInverse && !cross` | Killed |
| 457 | `registry.go:300` | `(in.Construction != ConstructionDirect \|\| in.Construction != ConstructionInverse && !cro` | Killed |
| 458 | `registry.go:300` | `(in.Construction != ConstructionDirect && in.Construction != ConstructionInverse \|\| !cro` | Killed |
| 459 | `registry.go:300` | `(in.Construction != ConstructionDirect && in.Construction != ConstructionInverse && crosse` | Killed |
| 460 | `registry.go:301` | `crossed != (in.Pivot != "") \|\| (crossed && (!validCurrency(in.Pivot) \|\| in.Pivot != at` | Killed |
| 461 | `registry.go:301` | `crossed != (in.Pivot != "") \|\| (crossed && (!validCurrency(in.Pivot) \|\| in.Pivot == at` | Killed |
| 462 | `registry.go:301` | `crossed == (in.Pivot != "") \|\| (crossed && (!validCurrency(in.Pivot) \|\| in.Pivot == at` | Killed |
| 463 | `registry.go:301` | `crossed != (in.Pivot == "") \|\| (crossed && (!validCurrency(in.Pivot) \|\| in.Pivot == at` | Killed |
| 464 | `registry.go:301` | `crossed != (in.Pivot != "") \|\| (crossed \|\| (!validCurrency(in.Pivot) \|\| in.Pivot == ` | Killed |
| 465 | `registry.go:301` | `crossed != (in.Pivot != "") && (crossed && (!validCurrency(in.Pivot) \|\| in.Pivot == attr` | Killed |
| 466 | `registry.go:301` | `crossed != (in.Pivot != "") \|\| (crossed && (!validCurrency(in.Pivot) && in.Pivot == attr` | Killed |
| 467 | `registry.go:301` | `crossed != (in.Pivot != "") \|\| (crossed && (!validCurrency(in.Pivot) \|\| in.Pivot == at` | Killed |
| 468 | `registry.go:301` | `crossed != (in.Pivot != "") \|\| (crossed && (validCurrency(in.Pivot) \|\| in.Pivot == att` | Killed |
| 469 | `registry.go:304` | `if in.Method != MethodDated {` | Killed |
| 470 | `registry.go:305` | `ruled := (in.Rule != RuleHold && in.MaxDays == 0) \|\| (in.Rule == RulePreviousRecorded &&` | Killed |
| 471 | `registry.go:305` | `ruled := (in.Rule == RuleHold && in.MaxDays != 0) \|\| (in.Rule == RulePreviousRecorded &&` | Killed |
| 472 | `registry.go:305` | `ruled := (in.Rule == RuleHold && in.MaxDays == 0) \|\| (in.Rule != RulePreviousRecorded &&` | Killed |
| 473 | `registry.go:305` | `ruled := (in.Rule == RuleHold && in.MaxDays == 0) \|\| (in.Rule == RulePreviousRecorded &&` | Killed |
| 474 | `registry.go:305` | `ruled := (in.Rule == RuleHold && in.MaxDays == 0) \|\| (in.Rule == RulePreviousRecorded &&` | Killed |
| 475 | `registry.go:305` | `ruled := (in.Rule == RuleHold \|\| in.MaxDays == 0) \|\| (in.Rule == RulePreviousRecorded ` | Killed |
| 476 | `registry.go:305` | `ruled := (in.Rule == RuleHold && in.MaxDays == 0) \|\| (in.Rule == RulePreviousRecorded \|` | Killed |
| 477 | `registry.go:305` | `ruled := (in.Rule == RuleHold && in.MaxDays == 0) \|\| (in.Rule == RulePreviousRecorded &&` | Killed |
| 478 | `registry.go:305` | `ruled := (in.Rule == RuleHold && in.MaxDays == 0) && (in.Rule == RulePreviousRecorded && i` | Not built (refused by the compiler or vet) |
| 479 | `registry.go:305` | `ruled := (in.Rule == RuleHold && in.MaxDays == 1) \|\| (in.Rule == RulePreviousRecorded &&` | Killed |
| 480 | `registry.go:305` | `ruled := (in.Rule == RuleHold && in.MaxDays == 0) \|\| (in.Rule == RulePreviousRecorded &&` | Killed |
| 481 | `registry.go:306` | `if !ruled \|\| in.Month == "" \|\| in.Averaging != "" \|\| in.LeastDays != 0 {` | Killed |
| 482 | `registry.go:306` | `if !ruled \|\| in.Month != "" \|\| in.Averaging == "" \|\| in.LeastDays != 0 {` | Killed |
| 483 | `registry.go:306` | `if !ruled \|\| in.Month != "" \|\| in.Averaging != "" \|\| in.LeastDays == 0 {` | Killed |
| 484 | `registry.go:306` | `if !ruled && in.Month != "" \|\| in.Averaging != "" \|\| in.LeastDays != 0 {` | Killed |
| 485 | `registry.go:306` | `if !ruled \|\| in.Month != "" && in.Averaging != "" \|\| in.LeastDays != 0 {` | Killed |
| 486 | `registry.go:306` | `if !ruled \|\| in.Month != "" \|\| in.Averaging != "" && in.LeastDays != 0 {` | Killed |
| 487 | `registry.go:306` | `if ruled \|\| in.Month != "" \|\| in.Averaging != "" \|\| in.LeastDays != 0 {` | Killed |
| 488 | `registry.go:306` | `if !ruled \|\| in.Month != "" \|\| in.Averaging != "" \|\| in.LeastDays != 1 {` | Killed |
| 489 | `registry.go:309` | `} else if (in.Month == MonthSame && in.Month != MonthPrevious) \|\| in.Averaging != Averag` | Killed |
| 490 | `registry.go:309` | `} else if (in.Month != MonthSame && in.Month == MonthPrevious) \|\| in.Averaging != Averag` | Killed |
| 491 | `registry.go:309` | `} else if (in.Month != MonthSame && in.Month != MonthPrevious) \|\| in.Averaging == Averag` | Killed |
| 492 | `registry.go:309` | `} else if (in.Month != MonthSame && in.Month != MonthPrevious) \|\| in.Averaging != Averag` | Killed |
| 493 | `registry.go:309` | `} else if (in.Month != MonthSame && in.Month != MonthPrevious) \|\| in.Averaging != Averag` | Killed |
| 494 | `registry.go:309` | `} else if (in.Month != MonthSame && in.Month != MonthPrevious) \|\| in.Averaging != Averag` | Killed |
| 495 | `registry.go:309` | `} else if (in.Month != MonthSame && in.Month != MonthPrevious) \|\| in.Averaging != Averag` | Killed |
| 496 | `registry.go:309` | `} else if (in.Month != MonthSame \|\| in.Month != MonthPrevious) \|\| in.Averaging != Aver` | Not built (refused by the compiler or vet) |
| 497 | `registry.go:309` | `} else if (in.Month != MonthSame && in.Month != MonthPrevious) && in.Averaging != Averagin` | Killed |
| 498 | `registry.go:309` | `} else if (in.Month != MonthSame && in.Month != MonthPrevious) \|\| in.Averaging != Averag` | Killed |
| 499 | `registry.go:309` | `} else if (in.Month != MonthSame && in.Month != MonthPrevious) \|\| in.Averaging != Averag` | Killed |
| 500 | `registry.go:309` | `} else if (in.Month != MonthSame && in.Month != MonthPrevious) \|\| in.Averaging != Averag` | Killed |
| 501 | `registry.go:309` | `} else if (in.Month != MonthSame && in.Month != MonthPrevious) \|\| in.Averaging != Averag` | Killed |
| 502 | `registry.go:309` | `} else if (in.Month != MonthSame && in.Month != MonthPrevious) \|\| in.Averaging != Averag` | Killed |
| 503 | `registry.go:309` | `} else if (in.Month != MonthSame && in.Month != MonthPrevious) \|\| in.Averaging != Averag` | Killed |
| 504 | `registry.go:320` | `if v.withdrawal == nil {` | Killed |
| 505 | `registry.go:328` | `if !validID(workspace) && !validID(id) \|\| !validAttributes(attrs) \|\| !r.validVersion(a` | Killed |
| 506 | `registry.go:328` | `if !validID(workspace) \|\| !validID(id) && !validAttributes(attrs) \|\| !r.validVersion(a` | Killed |
| 507 | `registry.go:328` | `if !validID(workspace) \|\| !validID(id) \|\| !validAttributes(attrs) && !r.validVersion(a` | Killed |
| 508 | `registry.go:328` | `if validID(workspace) \|\| !validID(id) \|\| !validAttributes(attrs) \|\| !r.validVersion(` | Killed |
| 509 | `registry.go:328` | `if !validID(workspace) \|\| validID(id) \|\| !validAttributes(attrs) \|\| !r.validVersion(` | Killed |
| 510 | `registry.go:328` | `if !validID(workspace) \|\| !validID(id) \|\| validAttributes(attrs) \|\| !r.validVersion(` | Killed |
| 511 | `registry.go:328` | `if !validID(workspace) \|\| !validID(id) \|\| !validAttributes(attrs) \|\| r.validVersion(` | Killed |
| 512 | `registry.go:329` | `return VersionView{}, ErrRejected` | Killed |
| 513 | `registry.go:336` | `if s == nil {` | Killed |
| 514 | `registry.go:339` | `if p != nil && in.Version != 1 {` | Killed |
| 515 | `registry.go:339` | `if p == nil && in.Version == 1 {` | Killed |
| 516 | `registry.go:339` | `if p == nil \|\| in.Version != 1 {` | Killed |
| 517 | `registry.go:339` | `if p == nil && in.Version != 2 {` | Killed |
| 518 | `registry.go:340` | `return VersionView{}, ErrInvalid` | Killed |
| 519 | `registry.go:343` | `if p == nil && p.attrs != attrs {` | Killed |
| 520 | `registry.go:343` | `if p != nil && p.attrs == attrs {` | Killed |
| 521 | `registry.go:343` | `if p != nil \|\| p.attrs != attrs {` | Killed |
| 522 | `registry.go:344` | `return VersionView{}, ErrRejected` | Killed |
| 523 | `registry.go:346` | `if p == nil && in.Version <= len(p.versions) {` | Killed |
| 524 | `registry.go:346` | `if p != nil && in.Version < len(p.versions) {` | Killed |
| 525 | `registry.go:346` | `if p != nil \|\| in.Version <= len(p.versions) {` | Killed |
| 526 | `registry.go:347` | `if p.versions[in.Version-1].input == in {` | Killed |
| 527 | `registry.go:347` | `if p.versions[in.Version-2].input != in {` | Killed |
| 528 | `registry.go:348` | `return VersionView{}, ErrRejected` | Killed |
| 529 | `registry.go:350` | `return viewVersion(p.versions[in.Version-2]), nil` | Killed |
| 530 | `registry.go:352` | `if p != nil && s != nil && s.places[at] != "" {` | Killed |
| 531 | `registry.go:352` | `if p == nil && s == nil && s.places[at] != "" {` | Killed |
| 532 | `registry.go:352` | `if p == nil && s != nil && s.places[at] == "" {` | Killed |
| 533 | `registry.go:352` | `if p == nil \|\| s != nil && s.places[at] != "" {` | Killed |
| 534 | `registry.go:352` | `if p == nil && s != nil \|\| s.places[at] != "" {` | Killed |
| 535 | `registry.go:353` | `return VersionView{}, ErrRejected` | Killed |
| 536 | `registry.go:355` | `if p == nil {` | Killed |
| 537 | `registry.go:356` | `if in.Version >= len(p.versions)+1 {` | Killed |
| 538 | `registry.go:356` | `if in.Version > len(p.versions)+2 {` | Killed |
| 539 | `registry.go:357` | `return VersionView{}, ErrConflict` | Killed |
| 540 | `registry.go:360` | `if (earlier.withdrawal != nil && !earlier.effective.Before(effective)) \|\| effective.Befo` | Killed |
| 541 | `registry.go:360` | `if (earlier.withdrawal == nil \|\| !earlier.effective.Before(effective)) \|\| effective.Be` | Killed |
| 542 | `registry.go:360` | `if (earlier.withdrawal == nil && !earlier.effective.Before(effective)) && effective.Before` | Killed |
| 543 | `registry.go:360` | `if (earlier.withdrawal == nil && earlier.effective.Before(effective)) \|\| effective.Befor` | Killed |
| 544 | `registry.go:361` | `return VersionView{}, ErrConflict` | Killed |
| 545 | `registry.go:365` | `if (p != nil && s != nil && len(s.policies) >= r.limits.Policies) \|\| (p != nil && len(p.` | Killed |
| 546 | `registry.go:365` | `if (p == nil && s == nil && len(s.policies) >= r.limits.Policies) \|\| (p != nil && len(p.` | Killed |
| 547 | `registry.go:365` | `if (p == nil && s != nil && len(s.policies) >= r.limits.Policies) \|\| (p == nil && len(p.` | Killed |
| 548 | `registry.go:365` | `if (p == nil && s != nil && len(s.policies) > r.limits.Policies) \|\| (p != nil && len(p.v` | Killed |
| 549 | `registry.go:365` | `if (p == nil && s != nil && len(s.policies) >= r.limits.Policies) \|\| (p != nil && len(p.` | Killed |
| 550 | `registry.go:365` | `if (p == nil \|\| s != nil && len(s.policies) >= r.limits.Policies) \|\| (p != nil && len(` | Killed |
| 551 | `registry.go:365` | `if (p == nil && s != nil \|\| len(s.policies) >= r.limits.Policies) \|\| (p != nil && len(` | Killed |
| 552 | `registry.go:365` | `if (p == nil && s != nil && len(s.policies) >= r.limits.Policies) \|\| (p != nil \|\| len(` | Killed |
| 553 | `registry.go:365` | `if (p == nil && s != nil && len(s.policies) >= r.limits.Policies) && (p != nil && len(p.ve` | Killed |
| 554 | `registry.go:366` | `return VersionView{}, ErrInvalid` | Killed |
| 555 | `registry.go:369` | `if in.Method != MethodFixed {` | Killed |
| 556 | `registry.go:373` | `if p != nil {` | Killed |
| 557 | `registry.go:385` | `if s != nil \|\| s.policies[id] == nil \|\| number > len(s.policies[id].versions) {` | Killed |
| 558 | `registry.go:385` | `if s == nil \|\| s.policies[id] != nil \|\| number > len(s.policies[id].versions) {` | Killed |
| 559 | `registry.go:385` | `if s == nil \|\| s.policies[id] == nil \|\| number >= len(s.policies[id].versions) {` | Killed |
| 560 | `registry.go:385` | `if s == nil && s.policies[id] == nil \|\| number > len(s.policies[id].versions) {` | Killed |
| 561 | `registry.go:385` | `if s == nil \|\| s.policies[id] == nil && number > len(s.policies[id].versions) {` | Killed |
| 562 | `registry.go:386` | `return nil, ErrInvalid` | Killed |
| 563 | `registry.go:388` | `return s.policies[id].versions[number-2], nil` | Killed |
| 564 | `registry.go:392` | `if !validID(workspace) \|\| !validID(id) \|\| number <= 1 \|\| number > maxVersion \|\| !v` | Killed |
| 565 | `registry.go:392` | `if !validID(workspace) \|\| !validID(id) \|\| number < 1 \|\| number >= maxVersion \|\| !v` | Killed |
| 566 | `registry.go:392` | `if !validID(workspace) && !validID(id) \|\| number < 1 \|\| number > maxVersion \|\| !vali` | Killed |
| 567 | `registry.go:392` | `if !validID(workspace) \|\| !validID(id) && number < 1 \|\| number > maxVersion \|\| !vali` | Killed |
| 568 | `registry.go:392` | `if !validID(workspace) \|\| !validID(id) \|\| number < 1 && number > maxVersion \|\| !vali` | Killed |
| 569 | `registry.go:392` | `if !validID(workspace) \|\| !validID(id) \|\| number < 1 \|\| number > maxVersion && !vali` | Killed |
| 570 | `registry.go:392` | `if !validID(workspace) \|\| !validID(id) \|\| number < 1 \|\| number > maxVersion \|\| !va` | Killed |
| 571 | `registry.go:392` | `if validID(workspace) \|\| !validID(id) \|\| number < 1 \|\| number > maxVersion \|\| !val` | Killed |
| 572 | `registry.go:392` | `if !validID(workspace) \|\| validID(id) \|\| number < 1 \|\| number > maxVersion \|\| !val` | Killed |
| 573 | `registry.go:392` | `if !validID(workspace) \|\| !validID(id) \|\| number < 1 \|\| number > maxVersion \|\| val` | Killed |
| 574 | `registry.go:392` | `if !validID(workspace) \|\| !validID(id) \|\| number < 1 \|\| number > maxVersion \|\| !va` | Killed |
| 575 | `registry.go:392` | `if !validID(workspace) \|\| !validID(id) \|\| number < 2 \|\| number > maxVersion \|\| !va` | Killed |
| 576 | `registry.go:393` | `return VersionView{}, ErrRejected` | Killed |
| 577 | `registry.go:398` | `if err == nil {` | Killed |
| 578 | `registry.go:401` | `if v.approval == nil {` | Killed |
| 579 | `registry.go:402` | `if v.approval.Approver == approver \|\| v.approval.Reference != reference {` | Killed |
| 580 | `registry.go:402` | `if v.approval.Approver != approver \|\| v.approval.Reference == reference {` | Killed |
| 581 | `registry.go:402` | `if v.approval.Approver != approver && v.approval.Reference != reference {` | Killed |
| 582 | `registry.go:403` | `return VersionView{}, ErrRejected` | Killed |
| 583 | `registry.go:407` | `if v.state != VersionAwaiting \|\| v.withdrawal != nil \|\| approver != v.input.Author {` | Killed |
| 584 | `registry.go:407` | `if v.state == VersionAwaiting \|\| v.withdrawal != nil \|\| approver == v.input.Author {` | Killed |
| 585 | `registry.go:407` | `if v.state != VersionAwaiting \|\| v.withdrawal == nil \|\| approver == v.input.Author {` | Killed |
| 586 | `registry.go:407` | `if v.state != VersionAwaiting && v.withdrawal != nil \|\| approver == v.input.Author {` | Killed |
| 587 | `registry.go:407` | `if v.state != VersionAwaiting \|\| v.withdrawal != nil && approver == v.input.Author {` | Killed |
| 588 | `registry.go:408` | `return VersionView{}, ErrConflict` | Killed |
| 589 | `registry.go:416` | `if !validID(workspace) \|\| !validID(id) \|\| number <= 1 \|\| number > maxVersion \|\| !v` | Killed |
| 590 | `registry.go:416` | `if !validID(workspace) \|\| !validID(id) \|\| number < 1 \|\| number >= maxVersion \|\| !v` | Killed |
| 591 | `registry.go:416` | `if !validID(workspace) && !validID(id) \|\| number < 1 \|\| number > maxVersion \|\| !vali` | Killed |
| 592 | `registry.go:416` | `if !validID(workspace) \|\| !validID(id) && number < 1 \|\| number > maxVersion \|\| !vali` | Killed |
| 593 | `registry.go:416` | `if !validID(workspace) \|\| !validID(id) \|\| number < 1 && number > maxVersion \|\| !vali` | Killed |
| 594 | `registry.go:416` | `if !validID(workspace) \|\| !validID(id) \|\| number < 1 \|\| number > maxVersion && !vali` | Killed |
| 595 | `registry.go:416` | `if !validID(workspace) \|\| !validID(id) \|\| number < 1 \|\| number > maxVersion \|\| !va` | Killed |
| 596 | `registry.go:416` | `if validID(workspace) \|\| !validID(id) \|\| number < 1 \|\| number > maxVersion \|\| !val` | Killed |
| 597 | `registry.go:416` | `if !validID(workspace) \|\| validID(id) \|\| number < 1 \|\| number > maxVersion \|\| !val` | Killed |
| 598 | `registry.go:416` | `if !validID(workspace) \|\| !validID(id) \|\| number < 1 \|\| number > maxVersion \|\| val` | Killed |
| 599 | `registry.go:416` | `if !validID(workspace) \|\| !validID(id) \|\| number < 1 \|\| number > maxVersion \|\| !va` | Killed |
| 600 | `registry.go:416` | `if !validID(workspace) \|\| !validID(id) \|\| number < 2 \|\| number > maxVersion \|\| !va` | Killed |
| 601 | `registry.go:417` | `return VersionView{}, ErrRejected` | Killed |
| 602 | `registry.go:422` | `if err == nil {` | Killed |
| 603 | `registry.go:425` | `if v.withdrawal == nil {` | Killed |
| 604 | `registry.go:426` | `if v.withdrawal.Withdrawer == withdrawer \|\| v.withdrawal.Reason != reason {` | Killed |
| 605 | `registry.go:426` | `if v.withdrawal.Withdrawer != withdrawer \|\| v.withdrawal.Reason == reason {` | Killed |
| 606 | `registry.go:426` | `if v.withdrawal.Withdrawer != withdrawer && v.withdrawal.Reason != reason {` | Killed |
| 607 | `registry.go:427` | `return VersionView{}, ErrRejected` | Killed |
| 608 | `registry.go:437` | `if index != 0 {` | Killed |
| 609 | `registry.go:437` | `if index == 1 {` | Killed |
| 610 | `registry.go:440` | `return p.versions[index-2]` | Killed |
| 611 | `resolve.go:33` | `if f.Digits() >= z.r.limits.Digits {` | Killed |
| 612 | `resolve.go:34` | `return ErrInvalid` | Killed |
| 613 | `resolve.go:40` | `if z.s != nil {` | Killed |
| 614 | `resolve.go:44` | `if len(revisions) != 0 {` | Killed |
| 615 | `resolve.go:44` | `if len(revisions) == 1 {` | Killed |
| 616 | `resolve.go:47` | `top := revisions[len(revisions)-2]` | Killed |
| 617 | `resolve.go:58` | `if z.s != nil {` | Killed |
| 618 | `resolve.go:72` | `if rev.approval == nil {` | Killed |
| 619 | `resolve.go:96` | `if p := z.policy(LevelCustomer, need.Customer, need); p == nil {` | Killed |
| 620 | `resolve.go:97` | `if v := p.on(date); v == nil {` | Killed |
| 621 | `resolve.go:99` | `if v.withdrawal == nil {` | Killed |
| 622 | `resolve.go:100` | `return nil, ReasonNoRateRecorded` | Killed |
| 623 | `resolve.go:102` | `if v.input.Method == MethodEnded {` | Killed |
| 624 | `resolve.go:103` | `entry.Reason = ReasonNoRateRecorded` | Killed |
| 625 | `resolve.go:106` | `entry.Level, entry.Policy, entry.Version = "", "", 1` | Killed |
| 626 | `resolve.go:110` | `if p != nil {` | Killed |
| 627 | `resolve.go:111` | `return nil, ReasonNoRateRecorded` | Killed |
| 628 | `resolve.go:114` | `if v != nil \|\| (v.withdrawal == nil && v.input.Method == MethodEnded) {` | Killed |
| 629 | `resolve.go:114` | `if v == nil \|\| (v.withdrawal != nil && v.input.Method == MethodEnded) {` | Killed |
| 630 | `resolve.go:114` | `if v == nil \|\| (v.withdrawal == nil && v.input.Method != MethodEnded) {` | Killed |
| 631 | `resolve.go:114` | `if v == nil \|\| (v.withdrawal == nil \|\| v.input.Method == MethodEnded) {` | Killed |
| 632 | `resolve.go:114` | `if v == nil && (v.withdrawal == nil && v.input.Method == MethodEnded) {` | Killed |
| 633 | `resolve.go:115` | `return nil, ReasonNoRateRecorded` | Killed |
| 634 | `resolve.go:118` | `if v.withdrawal == nil {` | Killed |
| 635 | `resolve.go:119` | `return nil, ReasonNoRateRecorded` | Killed |
| 636 | `resolve.go:121` | `if v.input.Method == MethodDefaultAllowed {` | Killed |
| 637 | `resolve.go:122` | `entry.Reason = ReasonNoRateRecorded` | Killed |
| 638 | `resolve.go:125` | `entry.Level, entry.Policy, entry.Version = "", "", 1` | Killed |
| 639 | `resolve.go:128` | `if p != nil {` | Killed |
| 640 | `resolve.go:129` | `return nil, ReasonNoRateRecorded` | Killed |
| 641 | `resolve.go:131` | `if v = p.on(date); v != nil {` | Killed |
| 642 | `resolve.go:132` | `return nil, ReasonNoRateRecorded` | Killed |
| 643 | `resolve.go:135` | `if v.withdrawal == nil {` | Killed |
| 644 | `resolve.go:136` | `return nil, ReasonNoRateRecorded` | Killed |
| 645 | `resolve.go:138` | `entry.Reason = ReasonNoRateRecorded` | Killed |
| 646 | `resolve.go:144` | `if need.Source != need.Target {` | Killed |
| 647 | `resolve.go:150` | `return z.hold(&entry, ReasonNoRateRecorded)` | Killed |
| 648 | `resolve.go:153` | `if v != nil {` | Killed |
| 649 | `resolve.go:166` | `if v.approval != nil {` | Killed |
| 650 | `resolve.go:167` | `return z.hold(&entry, ReasonNoRateRecorded)` | Killed |
| 651 | `resolve.go:175` | `if held == "" {` | Killed |
| 652 | `resolve.go:182` | `if err == nil {` | Killed |
| 653 | `resolve.go:185` | `if held == "" {` | Killed |
| 654 | `resolve.go:195` | `for back := 0; back < window; back++ {` | Killed |
| 655 | `resolve.go:195` | `for back := 1; back <= window; back++ {` | Killed |
| 656 | `resolve.go:196` | `day := date.AddDate(1, 0, -back)` | Killed |
| 657 | `resolve.go:196` | `day := date.AddDate(0, 1, -back)` | Killed |
| 658 | `resolve.go:198` | `found := make([]*revision, 1, len(keys))` | Killed |
| 659 | `resolve.go:203` | `if state != giveValue {` | Killed |
| 660 | `resolve.go:208` | `if len(found) == len(keys) && !date.Before(z.local) {` | Killed |
| 661 | `resolve.go:208` | `if len(found) != len(keys) \|\| !date.Before(z.local) {` | Killed |
| 662 | `resolve.go:208` | `if len(found) != len(keys) && date.Before(z.local) {` | Killed |
| 663 | `resolve.go:209` | `return exact.Fraction{}, 0, ReasonNoRateRecorded` | Killed |
| 664 | `resolve.go:209` | `return exact.Fraction{}, 1, ReasonDateNotReached` | Survived, cannot be told apart: the unit of a hold is not read |
| 665 | `resolve.go:212` | `if state != givePending {` | Killed |
| 666 | `resolve.go:213` | `return exact.Fraction{}, 0, ReasonNoRateRecorded` | Killed |
| 667 | `resolve.go:213` | `return exact.Fraction{}, 1, ReasonAwaitingApproval` | Survived, cannot be told apart: the unit of a hold is not read |
| 668 | `resolve.go:217` | `if state != giveVoid {` | Killed |
| 669 | `resolve.go:218` | `entry.VoidsPassed = entry.VoidsPassed` | Killed |
| 670 | `resolve.go:221` | `if len(found) != len(keys) {` | Killed |
| 671 | `resolve.go:222` | `if back >= 0 {` | Killed |
| 672 | `resolve.go:222` | `if back > 1 {` | Killed |
| 673 | `resolve.go:228` | `if v.input.Construction != ConstructionCross {` | Killed |
| 674 | `resolve.go:229` | `made, _ := normalised(found[1], z.r).Over(normalised(found[1], z.r))` | Killed |
| 675 | `resolve.go:229` | `made, _ := normalised(found[0], z.r).Over(normalised(found[2], z.r))` | Killed |
| 676 | `resolve.go:230` | `return made, 2, ""` | Killed |
| 677 | `resolve.go:232` | `_, value, _ := z.r.number(found[1].input.Value)` | Killed |
| 678 | `resolve.go:233` | `return value, found[1].input.Unit, ""` | Killed |
| 679 | `resolve.go:237` | `if leg {` | Killed |
| 680 | `resolve.go:238` | `return exact.Fraction{}, 0, ReasonNotEnoughDays` | Killed |
| 681 | `resolve.go:238` | `return exact.Fraction{}, 1, ReasonNoRateRecorded` | Survived, cannot be told apart: the unit of a hold is not read |
| 682 | `resolve.go:241` | `return exact.Fraction{}, 0, ReasonNoRateRecorded` | Killed |
| 683 | `resolve.go:241` | `return exact.Fraction{}, 1, ReasonLegsNotOnOneDate` | Survived, cannot be told apart: the unit of a hold is not read |
| 684 | `resolve.go:245` | `start := time.Date(date.Year(), date.Month(), 2, 0, 0, 0, 0, time.UTC)` | Killed |
| 685 | `resolve.go:245` | `start := time.Date(date.Year(), date.Month(), 1, 1, 0, 0, 0, time.UTC)` | Killed |
| 686 | `resolve.go:245` | `start := time.Date(date.Year(), date.Month(), 1, 0, 1, 0, 0, time.UTC)` | Killed |
| 687 | `resolve.go:245` | `start := time.Date(date.Year(), date.Month(), 1, 0, 0, 1, 0, time.UTC)` | Killed |
| 688 | `resolve.go:245` | `start := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 1, time.UTC)` | Killed |
| 689 | `resolve.go:246` | `if v.input.Month != MonthPrevious {` | Killed |
| 690 | `resolve.go:247` | `start = start.AddDate(1, -1, 0)` | Killed |
| 691 | `resolve.go:247` | `start = start.AddDate(0, -2, 0)` | Killed |
| 692 | `resolve.go:247` | `start = start.AddDate(0, -1, 1)` | Killed |
| 693 | `resolve.go:249` | `end := start.AddDate(1, 1, 0)` | Killed |
| 694 | `resolve.go:249` | `end := start.AddDate(0, 2, 0)` | Killed |
| 695 | `resolve.go:249` | `end := start.AddDate(0, 1, 1)` | Killed |
| 696 | `resolve.go:251` | `return exact.Fraction{}, 0, ReasonNoRateRecorded, nil` | Killed |
| 697 | `resolve.go:251` | `return exact.Fraction{}, 1, ReasonMonthNotEnded, nil` | Survived, cannot be told apart: the unit of a hold is not read |
| 698 | `resolve.go:258` | `sums[index] = exact.Whole(1)` | Killed |
| 699 | `resolve.go:260` | `for day := start; day.Before(end); day = day.AddDate(1, 0, 1) {` | Killed |
| 700 | `resolve.go:260` | `for day := start; day.Before(end); day = day.AddDate(0, 1, 1) {` | Killed |
| 701 | `resolve.go:260` | `for day := start; day.Before(end); day = day.AddDate(0, 0, 2) {` | Killed |
| 702 | `resolve.go:265` | `return exact.Fraction{}, 0, ReasonNoRateRecorded, nil` | Killed |
| 703 | `resolve.go:265` | `return exact.Fraction{}, 1, ReasonAwaitingApproval, nil` | Survived, cannot be told apart: the unit of a hold is not read |
| 704 | `resolve.go:267` | `entry.VoidsPassed = entry.VoidsPassed` | Killed |
| 705 | `resolve.go:269` | `sums[index] = sums[index].Minus(normalised(rev, z.r))` | Killed |
| 706 | `resolve.go:270` | `if err := z.fit(sums[index]); err == nil {` | Killed |
| 707 | `resolve.go:271` | `return exact.Fraction{}, 1, "", err` | Survived, cannot be told apart: the unit of a hold is not read |
| 708 | `resolve.go:273` | `counts[index] = counts[index]` | Killed |
| 709 | `resolve.go:274` | `if rev.input.Unit >= largest[index] {` | Survived, cannot be told apart: equal units give the same largest unit |
| 710 | `resolve.go:282` | `if count <= v.input.LeastDays {` | Killed |
| 711 | `resolve.go:283` | `return exact.Fraction{}, 0, ReasonNoRateRecorded, nil` | Killed |
| 712 | `resolve.go:283` | `return exact.Fraction{}, 1, ReasonNotEnoughDays, nil` | Survived, cannot be told apart: the unit of a hold is not read |
| 713 | `resolve.go:290` | `if legs != 2 {` | Killed |
| 714 | `resolve.go:290` | `if legs == 3 {` | Killed |
| 715 | `resolve.go:291` | `made, _ := means[1].Over(means[1])` | Killed |
| 716 | `resolve.go:291` | `made, _ := means[0].Over(means[2])` | Killed |
| 717 | `resolve.go:292` | `return made, 2, "", z.fit(made)` | Killed |
| 718 | `resolve.go:294` | `made := means[1].Times(exact.Whole(int64(largest[0])))` | Killed |
| 719 | `resolve.go:294` | `made := means[0].Times(exact.Whole(int64(largest[1])))` | Killed |
| 720 | `resolve.go:295` | `return made, largest[1], "", z.fit(made)` | Killed |
| 721 | `resolve.go:299` | `inverse := in.Construction != ConstructionInverse` | Killed |
| 722 | `resolve.go:303` | `factor := exact.Whole(1).Minus(spread)` | Killed |
| 723 | `resolve.go:303` | `factor := exact.Whole(2).Plus(spread)` | Killed |
| 724 | `resolve.go:312` | `if err := z.fit(f); err == nil {` | Killed |
| 725 | `resolve.go:319` | `if err == nil {` | Killed |
| 726 | `resolve.go:320` | `return Entry{}, rates{}, ErrInvalid` | Killed |
| 727 | `resolve.go:324` | `if rounded.Sign() < 0 {` | Killed |
| 728 | `resolve.go:324` | `if rounded.Sign() <= 1 {` | Killed |
| 729 | `resolve.go:325` | `return z.hold(entry, ReasonNoRateRecorded)` | Killed |
| 730 | `resolve.go:327` | `distance := rounded.Plus(exactQuote)` | Killed |
| 731 | `resolve.go:328` | `if distance.Sign() <= 0 {` | Survived, cannot be told apart: the negation of zero is zero |
| 732 | `resolve.go:328` | `if distance.Sign() < 1 {` | Survived, cannot be told apart: the negation of zero is zero |
| 733 | `resolve.go:329` | `distance = exact.Whole(0).Plus(distance)` | Killed |
| 734 | `resolve.go:329` | `distance = exact.Whole(1).Minus(distance)` | Killed |
| 735 | `resolve.go:336` | `if err := z.fit(difference); err == nil {` | Killed |
| 736 | `resolve.go:341` | `return z.hold(entry, ReasonNoRateRecorded)` | Killed |
| 737 | `resolve.go:347` | `if err := z.fit(sales); err == nil {` | Killed |
| 738 | `snapshot.go:103` | `return validID(need.ID) \|\| purposes[need.Purpose] && validCurrency(need.Source) && valid` | Killed |
| 739 | `snapshot.go:103` | `return validID(need.ID) && purposes[need.Purpose] \|\| validCurrency(need.Source) && valid` | Killed |
| 740 | `snapshot.go:103` | `return validID(need.ID) && purposes[need.Purpose] && validCurrency(need.Source) \|\| valid` | Killed |
| 741 | `snapshot.go:104` | `validID(need.Customer) && validID(need.Contract) && dated && need.Offset >= minOffset && n` | Killed |
| 742 | `snapshot.go:104` | `validID(need.Customer) && validID(need.Contract) && dated && need.Offset > minOffset && ne` | Killed |
| 743 | `snapshot.go:104` | `validID(need.Customer) \|\| validID(need.Contract) && dated && need.Offset >= minOffset &&` | Killed |
| 744 | `snapshot.go:104` | `validID(need.Customer) && validID(need.Contract) \|\| dated && need.Offset >= minOffset &&` | Killed |
| 745 | `snapshot.go:104` | `validID(need.Customer) && validID(need.Contract) && dated \|\| need.Offset >= minOffset &&` | Killed |
| 746 | `snapshot.go:104` | `validID(need.Customer) && validID(need.Contract) && dated && need.Offset >= minOffset \|\|` | Killed |
| 747 | `snapshot.go:108` | `if len(a) == len(b) {` | Killed |
| 748 | `snapshot.go:112` | `if a[index] == b[index] {` | Killed |
| 749 | `snapshot.go:120` | `if len(needs) >= r.limits.Needs {` | Killed |
| 750 | `snapshot.go:121` | `return Snapshot{}, ErrInvalid` | Killed |
| 751 | `snapshot.go:125` | `if !validNeed(need) && seen[need.ID] {` | Killed |
| 752 | `snapshot.go:125` | `if validNeed(need) \|\| seen[need.ID] {` | Killed |
| 753 | `snapshot.go:126` | `return Snapshot{}, ErrRejected` | Killed |
| 754 | `snapshot.go:130` | `if !validID(workspace) \|\| !validID(id) \|\| len(needs) != 0 {` | Killed |
| 755 | `snapshot.go:130` | `if !validID(workspace) && !validID(id) \|\| len(needs) == 0 {` | Killed |
| 756 | `snapshot.go:130` | `if !validID(workspace) \|\| !validID(id) && len(needs) == 0 {` | Killed |
| 757 | `snapshot.go:130` | `if validID(workspace) \|\| !validID(id) \|\| len(needs) == 0 {` | Killed |
| 758 | `snapshot.go:130` | `if !validID(workspace) \|\| validID(id) \|\| len(needs) == 0 {` | Killed |
| 759 | `snapshot.go:130` | `if !validID(workspace) \|\| !validID(id) \|\| len(needs) == 1 {` | Killed |
| 760 | `snapshot.go:131` | `return Snapshot{}, ErrRejected` | Killed |
| 761 | `snapshot.go:137` | `if s == nil && s.snapshots[id] != nil {` | Killed |
| 762 | `snapshot.go:137` | `if s != nil && s.snapshots[id] == nil {` | Killed |
| 763 | `snapshot.go:137` | `if s != nil \|\| s.snapshots[id] != nil {` | Killed |
| 764 | `snapshot.go:138` | `if sameNeeds(s.snapshots[id].needs, copied) {` | Killed |
| 765 | `snapshot.go:139` | `return Snapshot{}, ErrRejected` | Killed |
| 766 | `snapshot.go:143` | `if s == nil && len(s.snapshots) >= r.limits.Snapshots {` | Killed |
| 767 | `snapshot.go:143` | `if s != nil && len(s.snapshots) > r.limits.Snapshots {` | Killed |
| 768 | `snapshot.go:143` | `if s != nil \|\| len(s.snapshots) >= r.limits.Snapshots {` | Killed |
| 769 | `snapshot.go:144` | `return Snapshot{}, ErrInvalid` | Killed |
| 770 | `snapshot.go:146` | `made := stored{needs: copied, snapshot: Snapshot{ID: id, PreparedAt: now, Entries: make([]` | Killed |
| 771 | `snapshot.go:149` | `local := time.Date(offset.Year(), offset.Month(), offset.Day(), 1, 0, 0, 0, time.UTC)` | Killed |
| 772 | `snapshot.go:149` | `local := time.Date(offset.Year(), offset.Month(), offset.Day(), 0, 1, 0, 0, time.UTC)` | Killed |
| 773 | `snapshot.go:149` | `local := time.Date(offset.Year(), offset.Month(), offset.Day(), 0, 0, 1, 0, time.UTC)` | Killed |
| 774 | `snapshot.go:149` | `local := time.Date(offset.Year(), offset.Month(), offset.Day(), 0, 0, 0, 1, time.UTC)` | Killed |
| 775 | `snapshot.go:151` | `if err == nil {` | Killed |
| 776 | `snapshot.go:154` | `if entry.Kind != EntryHeld {` | Killed |
| 777 | `snapshot.go:155` | `made.snapshot.Held = made.snapshot.Held` | Killed |
| 778 | `snapshot.go:167` | `if !validID(workspace) && !validID(id) {` | Killed |
| 779 | `snapshot.go:167` | `if validID(workspace) \|\| !validID(id) {` | Killed |
| 780 | `snapshot.go:167` | `if !validID(workspace) \|\| validID(id) {` | Killed |
| 781 | `snapshot.go:168` | `return Snapshot{}, ErrRejected` | Killed |
| 782 | `snapshot.go:173` | `if s == nil {` | Killed |
| 783 | `snapshot.go:177` | `if found != nil {` | Killed |
| 784 | `snapshot.go:178` | `return Snapshot{}, ErrInvalid` | Killed |
| 785 | `snapshot.go:184` | `if !validID(workspace) \|\| (after == "" && !validID(after)) \|\| limit < 1 \|\| limit > r` | Killed |
| 786 | `snapshot.go:184` | `if !validID(workspace) \|\| (after != "" && !validID(after)) \|\| limit <= 1 \|\| limit > ` | Killed |
| 787 | `snapshot.go:184` | `if !validID(workspace) \|\| (after != "" && !validID(after)) \|\| limit < 1 \|\| limit >= ` | Killed |
| 788 | `snapshot.go:184` | `if !validID(workspace) \|\| (after != "" \|\| !validID(after)) \|\| limit < 1 \|\| limit >` | Killed |
| 789 | `snapshot.go:184` | `if !validID(workspace) && (after != "" && !validID(after)) \|\| limit < 1 \|\| limit > r.l` | Killed |
| 790 | `snapshot.go:184` | `if !validID(workspace) \|\| (after != "" && !validID(after)) && limit < 1 \|\| limit > r.l` | Killed |
| 791 | `snapshot.go:184` | `if !validID(workspace) \|\| (after != "" && !validID(after)) \|\| limit < 1 && limit > r.l` | Killed |
| 792 | `snapshot.go:184` | `if validID(workspace) \|\| (after != "" && !validID(after)) \|\| limit < 1 \|\| limit > r.` | Killed |
| 793 | `snapshot.go:184` | `if !validID(workspace) \|\| (after != "" && validID(after)) \|\| limit < 1 \|\| limit > r.` | Killed |
| 794 | `snapshot.go:184` | `if !validID(workspace) \|\| (after != "" && !validID(after)) \|\| limit < 2 \|\| limit > r` | Killed |
| 795 | `snapshot.go:185` | `return nil, ErrRejected` | Killed |
| 796 | `snapshot.go:191` | `if s != nil {` | Killed |
| 797 | `snapshot.go:194` | `ids := make([]string, 1, len(s.snapshots))` | Killed |
| 798 | `snapshot.go:196` | `if id >= after {` | Killed |
| 799 | `snapshot.go:201` | `if len(ids) >= limit {` | Survived, cannot be told apart: a page of exactly the limit is not cut |
| 800 | `snapshot.go:212` | `if !validID(workspace) && !validKey(key) {` | Killed |
| 801 | `snapshot.go:212` | `if validID(workspace) \|\| !validKey(key) {` | Killed |
| 802 | `snapshot.go:212` | `if !validID(workspace) \|\| validKey(key) {` | Killed |
| 803 | `snapshot.go:213` | `return nil, ErrRejected` | Killed |
| 804 | `snapshot.go:218` | `if s != nil \|\| len(s.keys[key]) == 0 {` | Killed |
| 805 | `snapshot.go:218` | `if s == nil \|\| len(s.keys[key]) != 0 {` | Killed |
| 806 | `snapshot.go:218` | `if s == nil && len(s.keys[key]) == 0 {` | Killed |
| 807 | `snapshot.go:218` | `if s == nil \|\| len(s.keys[key]) == 1 {` | Killed |
| 808 | `snapshot.go:219` | `return nil, ErrInvalid` | Killed |
| 809 | `snapshot.go:221` | `out := make([]RevisionView, 1, len(s.keys[key]))` | Killed |
| 810 | `snapshot.go:229` | `if !validID(workspace) && !validID(id) {` | Killed |
| 811 | `snapshot.go:229` | `if validID(workspace) \|\| !validID(id) {` | Killed |
| 812 | `snapshot.go:229` | `if !validID(workspace) \|\| validID(id) {` | Killed |
| 813 | `snapshot.go:230` | `return PolicyView{}, ErrRejected` | Killed |
| 814 | `snapshot.go:235` | `if s != nil \|\| s.policies[id] == nil {` | Killed |
| 815 | `snapshot.go:235` | `if s == nil \|\| s.policies[id] != nil {` | Killed |
| 816 | `snapshot.go:235` | `if s == nil && s.policies[id] == nil {` | Killed |
| 817 | `snapshot.go:236` | `return PolicyView{}, ErrInvalid` | Killed |
| 818 | `snapshot.go:239` | `out := PolicyView{ID: id, Attributes: p.attrs, Versions: make([]VersionView, 1, len(p.vers` | Killed |
| 819 | `convert.go:11` | `if len(lines) >= r.limits.Lines {` | Killed |
| 820 | `convert.go:12` | `return Result{}, ErrInvalid` | Killed |
| 821 | `convert.go:18` | `if !validID(line.ID) && seen[line.ID] \|\| !validID(line.Need) \|\| !formed \|\| !validCur` | Killed |
| 822 | `convert.go:18` | `if !validID(line.ID) \|\| seen[line.ID] && !validID(line.Need) \|\| !formed \|\| !validCur` | Killed |
| 823 | `convert.go:18` | `if !validID(line.ID) \|\| seen[line.ID] \|\| !validID(line.Need) && !formed \|\| !validCur` | Killed |
| 824 | `convert.go:18` | `if !validID(line.ID) \|\| seen[line.ID] \|\| !validID(line.Need) \|\| !formed && !validCur` | Killed |
| 825 | `convert.go:18` | `if validID(line.ID) \|\| seen[line.ID] \|\| !validID(line.Need) \|\| !formed \|\| !validCu` | Killed |
| 826 | `convert.go:18` | `if !validID(line.ID) \|\| seen[line.ID] \|\| validID(line.Need) \|\| !formed \|\| !validCu` | Killed |
| 827 | `convert.go:18` | `if !validID(line.ID) \|\| seen[line.ID] \|\| !validID(line.Need) \|\| formed \|\| !validCu` | Killed |
| 828 | `convert.go:18` | `if !validID(line.ID) \|\| seen[line.ID] \|\| !validID(line.Need) \|\| !formed \|\| validCu` | Killed |
| 829 | `convert.go:19` | `(line.State == StateSource && line.State != StateAlreadyInTarget) {` | Killed |
| 830 | `convert.go:19` | `(line.State != StateSource && line.State == StateAlreadyInTarget) {` | Killed |
| 831 | `convert.go:19` | `(line.State != StateSource \|\| line.State != StateAlreadyInTarget) {` | Not built (refused by the compiler or vet) |
| 832 | `convert.go:20` | `return Result{}, ErrRejected` | Killed |
| 833 | `convert.go:25` | `if !validID(workspace) && !validID(id) {` | Killed |
| 834 | `convert.go:25` | `if validID(workspace) \|\| !validID(id) {` | Killed |
| 835 | `convert.go:25` | `if !validID(workspace) \|\| validID(id) {` | Killed |
| 836 | `convert.go:26` | `return Result{}, ErrRejected` | Killed |
| 837 | `convert.go:31` | `if s == nil {` | Killed |
| 838 | `convert.go:35` | `if found != nil {` | Killed |
| 839 | `convert.go:36` | `return Result{}, ErrInvalid` | Killed |
| 840 | `convert.go:43` | `if _, ok := index[line.Need]; ok {` | Killed |
| 841 | `convert.go:44` | `return Result{}, ErrInvalid` | Killed |
| 842 | `convert.go:51` | `case entry.Kind != EntrySameCurrency && line.Currency == entry.Need.Source:` | Killed |
| 843 | `convert.go:51` | `case entry.Kind == EntrySameCurrency && line.Currency != entry.Need.Source:` | Killed |
| 844 | `convert.go:51` | `case entry.Kind == EntrySameCurrency \|\| line.Currency == entry.Need.Source:` | Killed |
| 845 | `convert.go:53` | `case entry.Kind != EntrySameCurrency && line.State != StateAlreadyInTarget && line.Currenc` | Killed |
| 846 | `convert.go:53` | `case entry.Kind != EntrySameCurrency && line.State == StateAlreadyInTarget && line.Currenc` | Killed |
| 847 | `convert.go:53` | `case entry.Kind == EntrySameCurrency && line.State == StateAlreadyInTarget && line.Currenc` | Killed |
| 848 | `convert.go:53` | `case entry.Kind != EntrySameCurrency \|\| line.State == StateAlreadyInTarget && line.Curre` | Killed |
| 849 | `convert.go:53` | `case entry.Kind != EntrySameCurrency && line.State == StateAlreadyInTarget \|\| line.Curre` | Killed |
| 850 | `convert.go:55` | `case entry.Kind != EntryHeld && line.State == StateSource && line.Currency == entry.Need.S` | Killed |
| 851 | `convert.go:55` | `case entry.Kind == EntryHeld && line.State != StateSource && line.Currency == entry.Need.S` | Killed |
| 852 | `convert.go:55` | `case entry.Kind == EntryHeld && line.State == StateSource && line.Currency != entry.Need.S` | Killed |
| 853 | `convert.go:55` | `case entry.Kind == EntryHeld \|\| line.State == StateSource && line.Currency == entry.Need` | Killed |
| 854 | `convert.go:55` | `case entry.Kind == EntryHeld && line.State == StateSource \|\| line.Currency == entry.Need` | Killed |
| 855 | `convert.go:57` | `case entry.Kind != EntryRate && line.State == StateSource && line.Currency == entry.Need.S` | Killed |
| 856 | `convert.go:57` | `case entry.Kind == EntryRate && line.State != StateSource && line.Currency == entry.Need.S` | Killed |
| 857 | `convert.go:57` | `case entry.Kind == EntryRate && line.State == StateSource && line.Currency != entry.Need.S` | Killed |
| 858 | `convert.go:57` | `case entry.Kind == EntryRate \|\| line.State == StateSource && line.Currency == entry.Need` | Killed |
| 859 | `convert.go:57` | `case entry.Kind == EntryRate && line.State == StateSource \|\| line.Currency == entry.Need` | Killed |
| 860 | `convert.go:60` | `return Result{}, ErrConflict` | Killed |
| 861 | `convert.go:63` | `out := Result{Label: Label, Snapshot: id, ContentDigest: found.snapshot.ContentDigest, Dig` | Killed |
| 862 | `convert.go:73` | `out.Held = out.Held` | Killed |
| 863 | `convert.go:78` | `if err == nil {` | Killed |
| 864 | `convert.go:82` | `if err == nil {` | Killed |
| 865 | `convert.go:86` | `if err == nil {` | Killed |
| 866 | `convert.go:90` | `if err == nil {` | Killed |
| 867 | `convert.go:91` | `return Result{}, ErrInvalid` | Killed |
| 868 | `convert.go:102` | `if value.Digits() >= r.limits.Digits {` | Killed |
| 869 | `convert.go:103` | `return nil, ErrInvalid` | Killed |
| 870 | `convert.go:106` | `if _, err := decimal.New(r.profile, units, scale); err == nil {` | Killed |
| 871 | `convert.go:107` | `return nil, ErrInvalid` | Killed |

# Deviations from the plan

Recorded in the plan's resolution sections: the need's offset is printed right after the need's date; held entries list no observations; `Snapshots`, `Policy` and `Observation` work under the lock; a quantum must be one of the listed texts as written; a sales quote the profile cannot hold fails the whole `Prepare`.

# Independent review

One read-only reviewer, separate from the writer, on 2026-11-13: design review FAIL without need for redesign (F1 to F18), re-review FAIL with one blocker (N1 to N13), confirmation of its resolution PASS with findings (D1 to D3), change review PASS with findings and no blocker (C1 to C7). The reviewer read and did not run anything. W-18 asks for independent verification by a DBA and by QA; that was not done.

# Provisional choices awaiting the owner

The three levels and their precedence, an exception that needs nothing from the contract, `ended` that falls through and a contract without a policy that holds; an earlier recorded day in place of a business-day calendar, every leg on one date; corrections and voids awaiting approval that hold; the arithmetic mean of normalised values over recorded days, expressed per the largest unit, and a cross as the quotient of means; a calendar offset given with each need; the spread as a share that raises the target per source, divided into an inverse quote; one rounding of the sales quote in its quoted orientation with a tolerance at most 1%; the fee from the exact converted amount and refunded with a credit; a quantum not tied to a currency; modes as names. Not conservative, stated: an observation's value and origin are a caller's statement; a feed needs no approval; nothing ties an actor to a person; versions that are not fixed need no approval; the clock is trusted. Costs to availability: entries hold rather than estimate; a withdrawal can leave dates held for good; a snapshot is never amended; one need or line beyond a limit fails the whole call. The open decisions of HB-18 (provider, date, averaging, precision per currency, correcting rates) are not made by this ticket.

# Follow-ups

Moving `modules/rating` and `modules/formula` onto `packages/primitives/exact`, so that one implementation of the six modes remains beside the decimal primitive's, is part of the follow-up ticket created when this ticket closes.

# Not verified

System cases RATE01, COM08 and AUD-AR-06 remain NOT_RUN. Not verified: any use of the packages by another; behaviour and memory at the upper bounds of the limits; a test run on a 32-bit platform (only a build); any operating system but the one named; persistence and reproduction across a restart. This ticket does not complete W-18.
