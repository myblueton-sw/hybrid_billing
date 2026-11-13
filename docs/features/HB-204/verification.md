---
type: VerificationRecord
title: Rating engine verification
description: Records what was executed and reviewed for the first model of work item W-16, with results, the corrections made on the way, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-204](https://linear.app/hybrid-billing/issue/HB-204/w-16-a-deterministic-rating-engine-with-charge-types-rule-precedence) adds the Go package `modules/rating` from main `b6a7ef3`: a pure engine without state. `Compile` validates a policy once: a currency, a cost basis as a label, a quantum and a rounding mode without defaults, a transaction path, and rules of four levels (SKU, service, group, default) with one of seven charge types (pass-through, markup, target margin, unit price with a denominator of 1, 1,000 or 1,000,000, graduated tiers, volume tiers, flat fee), an included quantity, discounts, a minimum and a maximum. `Rate` turns the lines of one contract pool into charges with a trace per rule: values are exact fractions of whole numbers inside the package and are rounded once per amount; a rule's amount is computed on the sums over its lines and shared out by the existing largest-remainder allocation; costs below zero are passed at cost; a rule with a line whose needed input is not known is held and charges nothing; under a direct contract only service fees are charged; every line appears exactly once, with a charge, as covered by a flat fee, or excluded with a reason.

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan twice before implementation and the implemented change afterwards (see Independent review). Machine: mac_mini.

Nothing calls the package, and it prices no cloud usage: a cost is an input. Of the acceptance, 100 with a markup of 20% is 120 and with a target margin of 20% is 125. The source's examples of 1,160,000, 850,000 and 14,000 won are not verified, their inputs not being in the planning files read. Not delivered: a rule of the contract as a level of its own and precedence across sources of policy; a fee as a rate of a basis that is not itself charged; manual adjustments as drafts (AR-07); what a flat fee includes (AR-10); model aliases and validity of prices (AR-04); versions of period and exchange rate; proration; taxes. RATE01, COM07, SIM03, AUD-AR-01, AUD-AR-04, AUD-AR-07 and AUD-AR-10 are NOT_RUN. W-16 is not complete, its design part stays open with the owner, and the independent verification by a DBA and by QA that W-16 asks for remains.

# Executed checks

Executed on 2026-11-13 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain, with `GOCACHE` set to a directory of the session. No other operating system was tested.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All twenty-four packages `ok`, including `modules/rating`; run on the final code |
| `go test -count=20 ./modules/rating` | `ok`: twenty consecutive runs without a failure, before each mutation run; the last on the final code |

The cases of the acceptance at the level of this package (`rate_test.go`), with expected values worked out by hand: the two figures above with their traces; a change of the policy's version changing nothing but the label; costs below zero under each cost-based type, outside discounts and bounds; a cost of 100 and its reversal of minus 100 giving 20 under a markup and 25 under a target margin, which is the stated consequence of rating a correction as a credit; a rule of credits only that charges its minimum; three discounts stacking from 1,000 to 425; the same four lines under a direct contract, where only the service fee is charged, and under resale with all, some and no classes eligible.

Literal table tests (`rate_test.go`, `compile_test.go`): every refusal of a policy, of limits and of a line; a policy at its limits; every charge type with its trace; denominators; an included quantity below and above the quantity; a quantity on, just above and at the last bound of the tiers; graduated tiers with three denominators whose parts round to zero each and to one together; a minimum and a maximum, met, applied, equal, after a discount and one after the other with the basis of each step; a fixed discount larger than what is left; a flat fee with discounts and a cap, covering lines whose inputs are not known, and not charged for a rule without a line; each of six modes at ties above and below zero, at an odd count, below a tie and on a whole count; five quanta of a ratio that does not end; a discount of half a quantum under four modes; a product whose scale the profile cannot hold; shares on ties, of credits and of zero; cutting a usage into lines; the precedence with a line that four rules name; exclusions in their order; held rules beside a minimum, beside another rule and with a line that is not eligible; labels; no lines; an amount, a sum of weights and a grand total the profile cannot hold and the allocation's bound, each a limit error; identifiers compared with the allocation's; a policy changed by the caller after it was compiled; the package's rounding against the primitive's over 20,000 ratios at five scales under six modes, with a minimum of ties.

Property test (`property_test.go`, fixed seeds): 4,000 random policies and pools. A reference model written in the test with `math/big.Rat`, with its own rounding, computes per rule the base amount, what is passed at cost, each discount with what was left before it, the minimum, the maximum and the total, the exclusions, the covered lines and the held rules; the engine's charges per kind, the names of its steps in order, and the outcomes and bases of its steps must equal the model's. Besides: every charge is a multiple of the quantum; every line appears exactly once; the shares of a rule differ from their exact proportion by less than one quantum and a weight of zero has a share of zero; the whole result is equal for the lines in another order; a line cut into two parts of the same sign that add up exactly leaves the total and every step unchanged; raising a cost above zero or a quantity lowers neither the model's base nor what the engine rates, for every type but volume. The test fails unless every charge type, every kind of charge, every reason of exclusion, a held rule, 500 cuts and 500 raises were reached.

One compiled policy is rated by eight goroutines under the race detector with the lines in eight orders and equal results; a pool of 400 lines, the limit of that test, is rated.

Limits of these tests: the model is the same rules written a second time by the same author. As the change review noted, it shares with the engine the reading of the precedence, of eligibility, of when a rule is active, of the split of costs and credits, of a quantity on a bound, of how a discount is rounded and clamped, of the order of the bounds and of held rules; a misreading of the plan common to both would pass. Its rounding is written independently, and the package's rounding is compared with the primitive's apart from it. Shares are not modelled. The count of cuts includes lines that are excluded or covered by a flat fee. The scan of the source is a scan of words.

# Corrections made on the way

- The literal tests and the property test agreed with the engine on their first runs. Three expected values worked out by hand were wrong (1,535 for 1,520, a split of credits, 4,194 for 4,191) and were corrected against the arithmetic, which the change review recomputed.
- The first mutation run (160 rows) caught 143. Tests were added for two of the others; nine did not compile and were rewritten; six showed code that could not change a result. Five of those removals were right. The sixth was wrong: the running amount after a minimum was removed on the ground that nothing read it, while the step of the maximum did; no test saw it, since amounts were unaffected. The change review found it. The update was restored, with literal traces, a comparison of step bases in the model and a self-check of the total.
- The change review also led to a closed list of modes in the package, a test that reaches the allocation's sum of weights, and smaller corrections; see the plan's change review resolution.

# Mutation check

Single mutations of the four source files of `modules/rating` were applied one at a time by a Python script, and the package's tests were run after each. The per-row results were appended to files that were never overwritten. The script and the result files are kept outside the repository, so the table below cannot be audited from the change alone. Not among the rows, because no test can produce them: the self-check of the total and the checks of the allocation's result.

- First run, 160 rows, before the change review: 143 caught; handled as described above.
- Second run, 156 rows, on the final code: 155 caught and one row that did not compile; that row was rewritten and run again and caught. The table reports the second run with that row's second result.

| No. | Fault introduced | Result |
| --- | --- | --- |
| 1 | Identifier length unbounded | Caught |
| 2 | An empty identifier | Caught |
| 3 | The longest identifier refused | Caught |
| 4 | A slash in an identifier | Caught |
| 5 | A space in an identifier | Caught |
| 6 | An at sign in an identifier | Caught |
| 7 | Third currency letter unchecked | Caught |
| 8 | First currency letter unchecked | Caught |
| 9 | Second currency letter unchecked | Caught |
| 10 | A quantum of two | Caught |
| 11 | The quantum 0.1 refused | Caught |
| 12 | The quantum 0.01 refused | Caught |
| 13 | The quantum 0.001 refused | Caught |
| 14 | The quantum 0.0001 refused | Caught |
| 15 | The quantum 1 refused | Caught |
| 16 | Any class | Caught |
| 17 | A target margin that needs no cost | Caught |
| 18 | A pass-through that needs no cost | Caught |
| 19 | A markup that needs no cost | Caught |
| 20 | Volume tiers that need no quantity | Caught |
| 21 | A unit price that needs no quantity | Caught |
| 22 | Graduated tiers that need no quantity | Caught |
| 23 | Limit not required: l.Rules > 0 | Caught |
| 24 | Limit not required: l.Tiers > 0 | Caught |
| 25 | Limit not required: l.Discounts > 0 | Caught |
| 26 | Limit not required: l.Lines > 0 | Caught |
| 27 | Limit not required: l.Intermediate > 0 | Caught |
| 28 | Fractions added without a common denominator | Caught |
| 29 | A difference computed as a sum | Caught |
| 30 | A product without its denominator | Caught |
| 31 | A quotient computed as a product | Caught |
| 32 | Equal values called less | Caught |
| 33 | Fractions not written in lowest terms | Caught |
| 34 | A whole number written as a fraction | Caught |
| 35 | Away from zero rounds toward zero | Caught |
| 36 | Floor rounds up | Caught |
| 37 | Ceiling rounds down | Caught |
| 38 | Half up rounds a tie toward zero | Caught |
| 39 | Half even rounds every tie toward zero | Caught |
| 40 | Half even rounds every tie away from zero | Caught |
| 41 | Values below zero rounded toward the larger value | Caught |
| 42 | A count of whole units whatever the quantum | Caught |
| 43 | Negative numbers in a policy | Caught |
| 44 | An amount that is no multiple of the quantum | Caught |
| 45 | Any denominator | Caught |
| 46 | A denominator of a million refused | Caught |
| 47 | A denominator of a thousand refused | Caught |
| 48 | A rule without an identifier | Caught |
| 49 | A rule of any level | Caught |
| 50 | A key that does not fit its level | Caught |
| 51 | A malformed key | Caught |
| 52 | A rate that does not fit its type | Caught |
| 53 | A margin that does not fit its type | Caught |
| 54 | An amount that does not fit its type | Caught |
| 55 | A price that does not fit its type | Caught |
| 56 | Tiers that do not fit their type | Caught |
| 57 | An included quantity on another type | Caught |
| 58 | A markup that charges the rate alone | Caught |
| 59 | A margin of one or more | Caught |
| 60 | A target margin computed as a markup | Caught |
| 61 | Bounds that do not rise | Caught |
| 62 | Bounds compared with zero only | Caught |
| 63 | A last tier with a bound | Caught |
| 64 | A flat fee at the default level | Caught |
| 65 | A percentage of one or more | Caught |
| 66 | A discount of both kinds | Caught |
| 67 | A minimum above the maximum | Caught |
| 68 | A minimum equal to the maximum refused | Caught |
| 69 | Compiling without limits | Caught |
| 70 | A policy without an identifier | Caught |
| 71 | A policy without a version | Caught |
| 72 | A policy in a malformed currency | Caught |
| 73 | Any quantum the profile reads | Caught |
| 74 | A policy without a rule | Caught |
| 75 | Any cost basis | Caught |
| 76 | Any rounding mode | Caught |
| 77 | A quantum the profile cannot read | Caught |
| 78 | A direct policy with classes | Caught |
| 79 | A licence charged under a direct contract | Caught |
| 80 | CSP usage charged under a direct contract | Caught |
| 81 | An unknown class in a policy | Caught |
| 82 | A class twice | Caught |
| 83 | A class the policy does not list charged | Caught |
| 84 | Any path | Caught |
| 85 | Rules without limit | Caught |
| 86 | The limit of rules refused | Caught |
| 87 | Tiers without limit | Caught |
| 88 | Discounts without limit | Caught |
| 89 | The limit of tiers refused | Caught |
| 90 | The limit of discounts refused | Caught |
| 91 | A rule twice | Caught |
| 92 | Two rules of one level and key | Caught |
| 93 | Rules in the order given | Caught |
| 94 | A service rule before the rule of the SKU | Caught |
| 95 | A group rule before the rule of the service | Caught |
| 96 | Group rules ignored | Caught |
| 97 | An amount the profile cannot hold reported as a failed self-check | Caught |
| 98 | The allocation's bound reported as a failed self-check | Caught |
| 99 | Weights of zero given to the allocation | Caught |
| 100 | Shares joined to the wrong lines | Caught |
| 101 | Charges of zero returned for a rule | Caught |
| 102 | A share of zero not returned for its line | Caught |
| 103 | Discounts and bounds of a flat fee computed on zero | Caught |
| 104 | A cost of zero treated as a credit | Caught |
| 105 | Credits marked up with the costs | Caught |
| 106 | A markup that adds nothing | Caught |
| 107 | A target margin that lowers the price | Caught |
| 108 | A cost-based amount rounded toward zero whatever the mode | Caught |
| 109 | Credits rounded toward zero whatever the mode | Caught |
| 110 | Credits inside discounts and bounds | Caught |
| 111 | A negative quantity above the included one | Caught |
| 112 | The included quantity charged | Caught |
| 113 | A price charged per unit whatever its denominator | Caught |
| 114 | The first tier takes the whole quantity | Caught |
| 115 | Tiers without a part listed | Caught |
| 116 | Graduated tiers charge the last part only | Caught |
| 117 | Every tier priced from zero | Caught |
| 118 | A tier step for a quantity of zero | Caught |
| 119 | A quantity on a bound in the upper tier | Caught |
| 120 | Volume tiers priced at the last tier | Caught |
| 121 | A quantity-based amount rounded toward zero whatever the mode | Caught |
| 122 | A discount rounded toward zero whatever the mode | Caught |
| 123 | A discount rounded as a value below zero | Caught |
| 124 | A discount larger than what is left | Caught |
| 125 | Discounts that do not stack | Caught |
| 126 | A minimum that adds nothing | Caught |
| 127 | A maximum that takes nothing off | Caught |
| 128 | The maximum judged on the amount before the minimum | Caught |
| 129 | A mode the package does not round by | Caught |
| 130 | Steps for bounds that are not configured | Caught |
| 131 | A line without an identifier | Caught |
| 132 | A line of any class | Caught |
| 133 | A line in another currency | Caught |
| 134 | Malformed keys on a line | Caught |
| 135 | A negative quantity | Caught |
| 136 | Lines without limit | Caught |
| 137 | The limit of lines refused | Caught |
| 138 | A line twice | Caught |
| 139 | Lines in the order given | Caught |
| 140 | A result without the profile's revision | Caught |
| 141 | A result without the profile's identifier | Caught |
| 142 | A result without the algorithm | Caught |
| 143 | A result without the number of lines | Caught |
| 144 | A result without the cost basis | Caught |
| 145 | A result without the path | Caught |
| 146 | A result without the quantum | Caught |
| 147 | A result without the mode | Caught |
| 148 | The rule judged before the class | Caught |
| 149 | Classes that may not be charged rated | Caught |
| 150 | A held rule not marked in the result | Caught |
| 151 | The other lines of a held rule called unknown | Caught |
| 152 | Lines under a flat fee listed nowhere | Caught |
| 153 | The grand total is the last rule's | Caught |
| 154 | Excluded lines in the order of rules | Caught |
| 155 | Numbers of exclusions in another order | Caught |
| 156 | Numbers of zero listed | Caught |

# Deviations from the plan

Recorded in the plan's change review resolution: a compiled policy keeps parsed values and none of the caller's slices, which has the effect of a copy; two pieces of code that could not change a result were removed.

# Independent review

- Design: two read-only passes on 2026-11-13 (FAIL with 3 HIGH and no redesign asked; then PASS with findings, no blocker). See the plan's two resolution sections.
- Change: one read-only pass on 2026-11-13, PASS with findings (3 MEDIUM, 7 LOW), nothing executed by the reviewer. All findings were resolved or stated. The corrections made after it, among them the restored update and the new self-check, were not reviewed again by a separate reviewer; they are covered by the tests and the second mutation run.
- Not done: the independent verification by a DBA and by QA that W-16 asks for.

# Provisional choices awaiting the owner

Provisional defaults: the seven charge types with an included quantity, discounts and bounds; one rounding per amount, to the policy's quantum and mode; discounts stack one after the other on what is left, before the bounds; a credit is passed at cost and takes no markup, margin, discount or bound; the precedence stops at the first rule that names a line; a quantity on a tier's bound belongs to the lower tier; a flat fee and a minimum are charged for a rule with a line that is not excluded, also one of credits only or without usage, and not for a rule without a line; a rule with one unknown input charges nothing; under a direct contract only service fees are charged. Not conservative, stated: a caller who rates part of a pool gets a result that looks valid; classes, eligibility and the cost basis are the caller's statements; nothing ties a policy to an approval; a correction that arrives as a cost and its reversal is rated as a cost and a credit, so corrections of one usage must be netted before rating; a flat rule covers every line it names without a further charge, a credit among them; a rule above the level of a SKU adds quantities whatever their unit; a flat fee here and a prorated amount of `modules/contracts` can both be charged for one fee. Assumptions: one currency per policy; quantities are never negative; the reading of the requirement's precedence, a group being one that a line names. Undecided and not chosen here: the charge types at launch, how discounts stack, digits and rounding mode, who approves a policy.

# Not verified

The three examples of the source; a failed self-check, which no test produces; any real cost, contract or policy; the time a large pool takes; any operating system other than the one named; and everything listed as deferred in the plan. No deployment, payment, message, real account or customer data was involved.
