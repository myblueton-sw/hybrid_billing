---
type: VerificationRecord
title: Formula engine and change path verification
description: Records what was executed and reviewed for the first model of work item W-17, with results, the corrections made on the way, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-206](https://linear.app/hybrid-billing/issue/HB-206/w-17-restricted-typed-formulas-with-a-static-check-exact-evaluation) adds the Go package `modules/formula` from main `52e2ce9`. Its pure part: a formula set is data, variables with one of four types (number, money in a currency, quantity in a unit, price) and formulas that are trees of ten closed kinds of node; `Compile` judges a set under a decimal profile and limits and returns either findings of ten codes or a compiled set; `Evaluate` computes the formula of a target with exact fractions, each formula rounded once to its quantum, and returns a value or one of three reasons for no value, never a zero in its place. Its registry: revisions of a set whose content never changes and has a SHA-256 digest of a fixed print; a path draft, checked or check failed, impact recorded, approval recorded, active, superseded, withdrawn; an approval bound to a subject digest over workspace, set, revision, content and the impact record in force; an epoch per set that rises whenever the active revision changes and makes every earlier impact and approval stop holding.

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan twice before implementation and the implemented change afterwards (see Independent review). Machine: mac_mini.

Nothing calls the package. Of the acceptance of HB-17, at the level of this package only: a revision evaluated again, also after it was superseded and withdrawn, gives the same result with the same digest, in memory and under one profile and limits; an approval stops holding when the active revision of its set changes, and not for any other change of policy; a tree, a set, inputs and whole numbers beyond the limits are refused before unbounded work; a margin of 100% or more in a `gross_up` is refused, as a finding when it is a constant and as a reason when it is computed, while the same written as a division by one less a rate is not a margin and is computed. Not delivered: general settings; a text syntax; conditionals; aggregation scopes; the computation of an impact over a manifest, of which only a caller's record is kept; code plugins; manual adjustments; the approval subject, eligible approvers and limits of HB-23; any check of permission or of the network. The registry authorizes nothing and must not be exposed directly.

# Executed checks

Executed on 2026-11-13 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain, with `GOCACHE` set to a directory of the session. No other operating system and no 32-bit platform was tested.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All twenty-five packages `ok`, including `modules/formula`; run on the final code |
| `go test -count=15 ./modules/formula` and `-count=10` | `ok`: fifteen consecutive runs before the first mutation run and ten on the corrected code before the last |

Literal tests with expected values worked out by hand (`compile_test.go`, `evaluate_test.go`, `registry_test.go`): invalid profiles and limits and limits at their bounds; counts judged before elements; trees at and beyond the limits of nodes and depth, a shared subtree, a tree that contains itself, a tree both too deep and too large; every malformed form of a set, a formula and a node; every semantic code alone and several in their order; the whole matrix of every operation over eight types with every candidate output; every kind of node, among them 100 times (1 + 0.2) = 120 and a gross-up of 100 at 0.2 = 125; quotients by negative values under the six modes; the five quanta; a referenced amount seen rounded (99.99, not 100) and a referenced share seen exact; an amount that rounds to zero as a divisor; a price shown as 0 and used exact; a margin shown as 1 with exact value below one, computed, and above one, refused; the order of reasons; an overflow before a reason behind a reference; thirty and thirty-one digits; values the profile cannot hold; inputs and targets; a chain of six formulas of depth four; the digest against bytes built in the test and against seventeen changed contents; every operation of the registry in each of the eight states; every refusal with its kind and their order; repeats and conflicts; the history and its limit; the epoch through activations and withdrawals, the case of no active revision twice included; pages; the tenant boundary; times; errors as fixed values.

Properties with fixed seeds (`property_test.go`): the package's rounding against the primitive's `DivideRound` over 20,000 quotients, a third with negative divisors, at five scales under six modes; 30,000 pairs of random formulas built to be well typed, evaluated and compared with a reference written with `math/big.Rat` and its own rounding, with thresholds for values, each reason, limit errors and both kinds of reference; 20,000 well-typed trees with one variable's type changed or its name removed, refused with exactly the code of that defect; 4,000 random histories of sixty steps against a model, with invariants on the one active revision, on what an active revision's impact and approval are, on the epoch and on digests, and a list of every outcome per operation and every state that must have been met. Concurrency under the race detector with gates: two activations at once, an approval beside an activation, evaluations beside changes. A scan of the seven source files that are not tests for words of floating point, rational types, running code, connections and comments; it is a scan of words.

# Corrections made on the way

- The first design review returned FAIL (content hashed before anything bounded it; an approval bound only to content that cannot change). The plan was rewritten from its decision on and the re-review returned PASS with findings; all are resolved in the plan.
- The change review's findings C2, C3, C4, C5, C7, C8 and C10 changed code or tests after the review: `min` and `max` measured, a size judged below a node with a stray field, the form of inputs judged before the lookup, `Draft` and `Content` working outside the lock, unused code removed, the bound of counts lowered to fit a 32-bit `int`, tests added. These corrections were not reviewed again by a separate reviewer.
- One test expectation of mine was wrong and was corrected before the mutation runs: an input of 64 digits divided into one exceeds the digits of the limit, so the test of the length of an input uses the variable alone.

# Mutation check

A script in Python applies each of 374 changes to the source in turn, runs the package's tests and restores the file; results are appended to files outside the repository. The table is the last complete run, on the final code. 372 mutants were killed. Two survive and cannot be told apart by any test; the change review agreed with both arguments: 155, because there is one finding per code and formula and nothing is compiled with findings; 158, because only a caller changing its set during the call could show it, against which the copy is kept.

What went wrong in the runs, stated: the first run was cut off by a time limit of the session after 173 mutants and left one mutant in the source, which was found by the failing tests and restored; a file of two results taken on that broken source was deleted as invalid, against the rule that result files are kept; in two runs a group of consecutive mutants of one file was reported as not building and all were killed when run again, the cause not determined. Of the first run's mutants, eight survived and were killed by added tests (currency letters, types with fields of another kind, the length of a number text under a wide profile, a malformed operation not followed, two formulas of one rank), and seventeen did not build as written and were rewritten.

| # | File | Mutant | Result |
| --- | --- | --- | --- |
| 1 | `types.go` | Identifier length unbounded | Killed |
| 2 | `types.go` | An empty identifier | Killed |
| 3 | `types.go` | The longest identifier refused | Killed |
| 4 | `types.go` | Short number texts refused | Killed |
| 5 | `types.go` | The highest rank refused | Killed |
| 6 | `types.go` | A rank of a thousand | Killed |
| 7 | `types.go` | The highest revision refused | Killed |
| 8 | `types.go` | A revision of a million | Killed |
| 9 | `types.go` | The highest count refused | Killed |
| 10 | `types.go` | A count beyond the bound | Killed |
| 11 | `types.go` | A slash in an identifier | Killed |
| 12 | `types.go` | A space in an identifier | Killed |
| 13 | `types.go` | Third currency letter unchecked | Killed |
| 14 | `types.go` | First currency letter unchecked | Killed |
| 15 | `types.go` | A longer digest | Killed |
| 16 | `types.go` | A shorter digest | Killed |
| 17 | `types.go` | Letters beyond f in a digest | Killed |
| 18 | `types.go` | Upper case in a digest | Killed |
| 19 | `types.go` | A number with a unit | Killed |
| 20 | `types.go` | A number with a currency | Killed |
| 21 | `types.go` | Money with a unit | Killed |
| 22 | `types.go` | Money in any currency | Killed |
| 23 | `types.go` | A quantity with a currency | Killed |
| 24 | `types.go` | A quantity without a unit | Killed |
| 25 | `types.go` | A price without a unit | Killed |
| 26 | `types.go` | A price without a currency | Killed |
| 27 | `types.go` | Limit not required: l.Variables > 0 | Killed |
| 28 | `types.go` | Limit not required: l.Formulas > 0 | Killed |
| 29 | `types.go` | Limit not required: l.Nodes > 0 | Killed |
| 30 | `types.go` | Limit not required: l.Depth > 0 | Killed |
| 31 | `types.go` | Limit not required: l.Digits > 0 | Killed |
| 32 | `types.go` | Variables without an upper bound | Killed |
| 33 | `types.go` | Formulas without an upper bound | Killed |
| 34 | `types.go` | Nodes without an upper bound | Killed |
| 35 | `types.go` | Depth without an upper bound | Killed |
| 36 | `types.go` | Digits without an upper bound | Killed |
| 37 | `types.go` | A bound of variables one lower | Killed |
| 38 | `types.go` | A bound of formulas one lower | Killed |
| 39 | `types.go` | A bound of nodes one lower | Killed |
| 40 | `types.go` | A bound of depth one lower | Killed |
| 41 | `types.go` | A bound of digits one lower | Killed |
| 42 | `types.go` | Registry without formula limits | Killed |
| 43 | `types.go` | Limit not required: l.Sets > 0 | Killed |
| 44 | `types.go` | Limit not required: l.Revisions > 0 | Killed |
| 45 | `types.go` | Limit not required: l.History > 0 | Killed |
| 46 | `types.go` | Limit not required: l.Page > 0 | Killed |
| 47 | `types.go` | Sets without an upper bound | Killed |
| 48 | `types.go` | Revisions without an upper bound | Killed |
| 49 | `types.go` | History without an upper bound | Killed |
| 50 | `types.go` | Page without an upper bound | Killed |
| 51 | `types.go` | A bound of sets one lower | Killed |
| 52 | `types.go` | A bound of revisions one lower | Killed |
| 53 | `types.go` | A bound of history one lower | Killed |
| 54 | `types.go` | A bound of pages one lower | Killed |
| 55 | `fraction.go` | A denominator below zero kept | Killed |
| 56 | `fraction.go` | Fractions not kept in lowest terms | Killed |
| 57 | `fraction.go` | Fractions added without a common denominator | Killed |
| 58 | `fraction.go` | A difference computed as a sum | Killed |
| 59 | `fraction.go` | A product without its denominator | Killed |
| 60 | `fraction.go` | A quotient computed as a product | Killed |
| 61 | `fraction.go` | Equal values called less | Killed |
| 62 | `fraction.go` | The digits of a denominator not measured | Killed |
| 63 | `fraction.go` | The digits of a numerator not measured | Killed |
| 64 | `fraction.go` | A sign counted as a digit | Killed |
| 65 | `fraction.go` | A whole number written as a fraction | Killed |
| 66 | `fraction.go` | Away from zero rounds toward zero | Killed |
| 67 | `fraction.go` | Floor rounds up | Killed |
| 68 | `fraction.go` | Ceiling rounds down | Killed |
| 69 | `fraction.go` | Half up rounds a tie toward zero | Killed |
| 70 | `fraction.go` | Half even rounds every tie toward zero | Killed |
| 71 | `fraction.go` | Half even rounds every tie away from zero | Killed |
| 72 | `fraction.go` | Values below zero rounded toward the larger value | Killed |
| 73 | `fraction.go` | A count of whole units whatever the quantum | Killed |
| 74 | `fraction.go` | A quantum of five decimals | Killed |
| 75 | `fraction.go` | Half up not a mode | Killed |
| 76 | `compile.go` | Findings ordered by formula alone | Killed |
| 77 | `compile.go` | Findings of a code in reverse | Killed |
| 78 | `compile.go` | Findings ordered by the spelling of codes | Killed |
| 79 | `compile.go` | A constant of any length | Killed |
| 80 | `compile.go` | A constant with a name | Killed |
| 81 | `compile.go` | A constant with a formula | Killed |
| 82 | `compile.go` | A constant of a malformed type | Killed |
| 83 | `compile.go` | A constant that is no number | Killed |
| 84 | `compile.go` | A variable node without a name | Killed |
| 85 | `compile.go` | A variable node with a number | Killed |
| 86 | `compile.go` | A variable node with a formula | Killed |
| 87 | `compile.go` | A variable node with a type | Killed |
| 88 | `compile.go` | A reference without an identifier | Killed |
| 89 | `compile.go` | A reference with a number | Killed |
| 90 | `compile.go` | A reference with a name | Killed |
| 91 | `compile.go` | A reference with a type | Killed |
| 92 | `compile.go` | An operation with any number of operands | Killed |
| 93 | `compile.go` | An operation with a number | Killed |
| 94 | `compile.go` | An operation with a name | Killed |
| 95 | `compile.go` | An operation with a formula | Killed |
| 96 | `compile.go` | An operation with a type | Killed |
| 97 | `compile.go` | The last node refused | Killed |
| 98 | `compile.go` | One node beyond the limit | Killed |
| 99 | `compile.go` | Trees of any size | Killed |
| 100 | `compile.go` | The last level refused | Killed |
| 101 | `compile.go` | One level beyond the limit | Killed |
| 102 | `compile.go` | Depth judged before the count | Killed |
| 103 | `compile.go` | Operands on a kind that has none | Killed |
| 104 | `compile.go` | A malformed operation still followed | Killed |
| 105 | `compile.go` | A size below a node with a stray field not judged | Killed |
| 106 | `compile.go` | Depth not counted | Killed |
| 107 | `compile.go` | A size met below not passed up | Killed |
| 108 | `compile.go` | Variables without a limit | Killed |
| 109 | `compile.go` | Formulas without a limit | Killed |
| 110 | `compile.go` | The last variable refused | Killed |
| 111 | `compile.go` | The last formula refused | Killed |
| 112 | `compile.go` | Elements read after too many | Killed |
| 113 | `compile.go` | A variable without a name | Killed |
| 114 | `compile.go` | A variable of a malformed type | Killed |
| 115 | `compile.go` | A variable name twice | Killed |
| 116 | `compile.go` | A formula with a malformed identifier | Killed |
| 117 | `compile.go` | A formula identifier twice | Killed |
| 118 | `compile.go` | A formula too large also judged for its form | Killed |
| 119 | `compile.go` | A quantum the profile cannot read | Killed |
| 120 | `compile.go` | Any quantum the profile reads | Killed |
| 121 | `compile.go` | Malformed nodes | Killed |
| 122 | `compile.go` | A formula without a target | Killed |
| 123 | `compile.go` | A rank of zero | Killed |
| 124 | `compile.go` | A rank without an upper bound | Killed |
| 125 | `compile.go` | A malformed output type | Killed |
| 126 | `compile.go` | Any rounding mode | Killed |
| 127 | `compile.go` | Structural findings not returned | Killed |
| 128 | `compile.go` | Variables in the order given | Killed |
| 129 | `compile.go` | Formulas in the order given | Killed |
| 130 | `compile.go` | Sums of other currencies and units | Killed |
| 131 | `compile.go` | A number times a value refused | Killed |
| 132 | `compile.go` | A value times a number refused | Killed |
| 133 | `compile.go` | A quantity times a price of another unit | Killed |
| 134 | `compile.go` | A price times a quantity of another unit | Killed |
| 135 | `compile.go` | A quantity times a price always in one currency | Killed |
| 136 | `compile.go` | A price times a quantity always in one currency | Killed |
| 137 | `compile.go` | A ratio of other currencies or units | Killed |
| 138 | `compile.go` | A ratio refused | Killed |
| 139 | `compile.go` | An amount by a quantity refused | Killed |
| 140 | `compile.go` | An amount by a price of another currency | Killed |
| 141 | `compile.go` | An amount by a price always in hours | Killed |
| 142 | `compile.go` | A margin of any type | Killed |
| 143 | `compile.go` | A value by a number refused | Killed |
| 144 | `compile.go` | An unknown variable typed as nothing | Killed |
| 145 | `compile.go` | An unknown formula typed as nothing | Killed |
| 146 | `compile.go` | A constant divisor of zero | Killed |
| 147 | `compile.go` | A constant of zero under any operation | Killed |
| 148 | `compile.go` | A constant margin below zero | Killed |
| 149 | `compile.go` | A constant margin of one or more | Killed |
| 150 | `compile.go` | A constant margin of exactly one | Killed |
| 151 | `compile.go` | A constant margin of zero refused | Killed |
| 152 | `compile.go` | A constant outside a margin's range under any operation | Killed |
| 153 | `compile.go` | The constant checks on the left operand | Killed |
| 154 | `compile.go` | A finding that follows from another | Killed |
| 155 | `compile.go` | A mismatch typed as nothing | Survived, cannot be told apart |
| 156 | `compile.go` | A root of any type | Killed |
| 157 | `compile.go` | A root mismatch after another finding | Killed |
| 158 | `compile.go` | The caller's set compiled, not the copy | Survived, cannot be told apart |
| 159 | `compile.go` | Compiling without a profile | Killed |
| 160 | `compile.go` | Compiling without limits | Killed |
| 161 | `compile.go` | The highest rank number applies | Killed |
| 162 | `compile.go` | The first formula of a target applies | Killed |
| 163 | `compile.go` | Cycles | Killed |
| 164 | `compile.go` | A formula ready after one of its references | Killed |
| 165 | `compile.go` | Two formulas of one rank | Killed |
| 166 | `compile.go` | Any two formulas of one target conflict | Killed |
| 167 | `compile.go` | Formulas of one rank conflict whatever the target | Killed |
| 168 | `compile.go` | A set with findings compiled | Killed |
| 169 | `compile.go` | References not followed | Killed |
| 170 | `evaluate.go` | The last digit refused | Killed |
| 171 | `evaluate.go` | One digit beyond the limit | Killed |
| 172 | `evaluate.go` | A variable not known read as zero | Killed |
| 173 | `evaluate.go` | A referenced amount seen exact | Killed |
| 174 | `evaluate.go` | A referenced reason dropped | Killed |
| 175 | `evaluate.go` | The right operand first | Killed |
| 176 | `evaluate.go` | A sum not measured | Killed |
| 177 | `evaluate.go` | A difference not measured | Killed |
| 178 | `evaluate.go` | A product not measured | Killed |
| 179 | `evaluate.go` | A quotient not measured | Killed |
| 180 | `evaluate.go` | A gross-up not measured | Killed |
| 181 | `evaluate.go` | A difference the other way | Killed |
| 182 | `evaluate.go` | A division by zero read as zero | Killed |
| 183 | `evaluate.go` | A quotient the other way | Killed |
| 184 | `evaluate.go` | A minimum that takes the larger | Killed |
| 185 | `evaluate.go` | A maximum that takes the smaller | Killed |
| 186 | `evaluate.go` | A margin below zero | Killed |
| 187 | `evaluate.go` | A margin of exactly one | Killed |
| 188 | `evaluate.go` | A margin of one or more | Killed |
| 189 | `evaluate.go` | A margin of zero refused | Killed |
| 190 | `evaluate.go` | A gross-up computed as a markup | Killed |
| 191 | `evaluate.go` | Inputs without a limit | Killed |
| 192 | `evaluate.go` | The last input refused | Killed |
| 193 | `evaluate.go` | An input the set does not declare | Killed |
| 194 | `evaluate.go` | Inputs of any form evaluated | Killed |
| 195 | `evaluate.go` | An input name of any form | Killed |
| 196 | `evaluate.go` | A minimum not measured | Killed |
| 197 | `evaluate.go` | A maximum not measured | Killed |
| 198 | `evaluate.go` | An input twice | Killed |
| 199 | `evaluate.go` | An input text of any length | Killed |
| 200 | `evaluate.go` | An input twice when once not known | Killed |
| 201 | `evaluate.go` | An input that is no number | Killed |
| 202 | `evaluate.go` | A malformed target not found | Killed |
| 203 | `evaluate.go` | Only direct references reached | Killed |
| 204 | `evaluate.go` | Every formula computed | Killed |
| 205 | `evaluate.go` | A rounded value the profile cannot hold | Killed |
| 206 | `evaluate.go` | Every referenced value seen rounded | Killed |
| 207 | `evaluate.go` | Formulas not reached listed | Killed |
| 208 | `evaluate.go` | The computed formula listed among the reached | Killed |
| 209 | `evaluate.go` | Reached formulas in reverse | Killed |
| 210 | `evaluate.go` | A result without the profile's revision | Killed |
| 211 | `evaluate.go` | A result without its label | Killed |
| 212 | `digest.go` | Fields without their length | Killed |
| 213 | `digest.go` | Currency and unit as one field | Killed |
| 214 | `digest.go` | A type without its kind | Killed |
| 215 | `digest.go` | A node without its kind | Killed |
| 216 | `digest.go` | A constant without its type | Killed |
| 217 | `digest.go` | A constant without its text | Killed |
| 218 | `digest.go` | A variable node without its name | Killed |
| 219 | `digest.go` | A reference without its identifier | Killed |
| 220 | `digest.go` | Operands printed the other way | Killed |
| 221 | `digest.go` | The right operand not printed | Killed |
| 222 | `digest.go` | Content without its label | Killed |
| 223 | `digest.go` | Variables without their count | Killed |
| 224 | `digest.go` | Formulas without their count | Killed |
| 225 | `digest.go` | A variable without its type | Killed |
| 226 | `digest.go` | A variable without its name | Killed |
| 227 | `digest.go` | A formula without its identifier | Killed |
| 228 | `digest.go` | A formula without its target | Killed |
| 229 | `digest.go` | A formula without its rank | Killed |
| 230 | `digest.go` | A formula without its output | Killed |
| 231 | `digest.go` | A formula without its quantum | Killed |
| 232 | `digest.go` | A formula without its mode | Killed |
| 233 | `digest.go` | A formula without its tree | Killed |
| 234 | `digest.go` | A subject without its label | Killed |
| 235 | `digest.go` | A subject without its workspace | Killed |
| 236 | `digest.go` | A subject without its set | Killed |
| 237 | `digest.go` | A subject without its revision | Killed |
| 238 | `digest.go` | A subject without the content | Killed |
| 239 | `digest.go` | A subject without the manifest | Killed |
| 240 | `digest.go` | A subject without the manifest's digest | Killed |
| 241 | `digest.go` | A subject without the epoch | Killed |
| 242 | `digest.go` | A subject without the coverage | Killed |
| 243 | `digest.go` | A subject without the number evaluated | Killed |
| 244 | `digest.go` | A subject without the number changed | Killed |
| 245 | `digest.go` | A subject without the number without a value | Killed |
| 246 | `digest.go` | A subject without the number failed | Killed |
| 247 | `digest.go` | Content read shares its operands | Killed |
| 248 | `digest.go` | Content read shares its trees | Killed |
| 249 | `digest.go` | Content read shares its variables | Killed |
| 250 | `registry.go` | A registry without a profile | Killed |
| 251 | `registry.go` | A registry without limits | Killed |
| 252 | `registry.go` | A registry without a clock | Killed |
| 253 | `registry.go` | Times not in UTC | Killed |
| 254 | `registry.go` | A clock that goes back followed | Killed |
| 255 | `registry.go` | A revision of zero | Killed |
| 256 | `registry.go` | A revision without an upper bound | Killed |
| 257 | `registry.go` | A view shares its approval | Killed |
| 258 | `registry.go` | Every impact shown as current | Killed |
| 259 | `registry.go` | A size reported as malformed | Killed |
| 260 | `registry.go` | Malformed content drafted | Killed |
| 261 | `registry.go` | Draft: any workspace | Killed |
| 262 | `registry.go` | Draft: any set | Killed |
| 263 | `registry.go` | Draft: any number | Killed |
| 264 | `registry.go` | Draft: any author | Killed |
| 265 | `registry.go` | Another content under a number | Killed |
| 266 | `registry.go` | Another author under a number | Killed |
| 267 | `registry.go` | One set beyond the limit | Killed |
| 268 | `registry.go` | A revision refused in a full workspace | Killed |
| 269 | `registry.go` | Sets without a limit | Killed |
| 270 | `registry.go` | One revision beyond the limit | Killed |
| 271 | `registry.go` | Revisions without a limit | Killed |
| 272 | `registry.go` | A draft that does not raise the sequence | Killed |
| 273 | `registry.go` | A draft that keeps the caller's set | Killed |
| 274 | `registry.go` | A check on a withdrawn revision | Killed |
| 275 | `registry.go` | A check run again | Killed |
| 276 | `registry.go` | A check without its time | Killed |
| 277 | `registry.go` | A check that does not raise the sequence | Killed |
| 278 | `registry.go` | A failed check called checked | Killed |
| 279 | `registry.go` | Counts below zero | Killed |
| 280 | `registry.go` | Counts without an upper bound | Killed |
| 281 | `registry.go` | An epoch outside its range | Killed |
| 282 | `registry.go` | A number failed below zero | Killed |
| 283 | `registry.go` | A number without a value below zero | Killed |
| 284 | `registry.go` | A number changed below zero | Killed |
| 285 | `registry.go` | A number evaluated outside its range | Killed |
| 286 | `registry.go` | An impact without a manifest | Killed |
| 287 | `registry.go` | An impact without a manifest digest | Killed |
| 288 | `registry.go` | Any coverage | Killed |
| 289 | `registry.go` | More changed than evaluated | Killed |
| 290 | `registry.go` | More without a value and failed than evaluated | Killed |
| 291 | `registry.go` | Failed not counted against evaluated | Killed |
| 292 | `registry.go` | Without a value not counted against evaluated | Killed |
| 293 | `registry.go` | All changed refused | Killed |
| 294 | `registry.go` | All without a value or failed refused | Killed |
| 295 | `registry.go` | Impact: any workspace | Killed |
| 296 | `registry.go` | Impact: any set | Killed |
| 297 | `registry.go` | Impact: any number | Killed |
| 298 | `registry.go` | An identical impact is no repeat | Killed |
| 299 | `registry.go` | Another impact under a current approval | Killed |
| 300 | `registry.go` | Another impact under a past approval is a conflict | Killed |
| 301 | `registry.go` | An impact cannot be replaced before an approval | Killed |
| 302 | `registry.go` | An impact on a checked revision refused | Killed |
| 303 | `registry.go` | An impact cannot be replaced | Killed |
| 304 | `registry.go` | An impact after a past approval refused | Killed |
| 305 | `registry.go` | An impact in any state | Killed |
| 306 | `registry.go` | An impact of any epoch | Killed |
| 307 | `registry.go` | One impact beyond the history | Killed |
| 308 | `registry.go` | A history without a limit | Killed |
| 309 | `registry.go` | A replaced impact not kept | Killed |
| 310 | `registry.go` | An impact that does not raise the sequence | Killed |
| 311 | `registry.go` | A new impact keeps the state of the approval | Killed |
| 312 | `registry.go` | An impact with another time | Killed |
| 313 | `registry.go` | Approval: any workspace | Killed |
| 314 | `registry.go` | Approval: any set | Killed |
| 315 | `registry.go` | Approval: any number | Killed |
| 316 | `registry.go` | Approval: any approver | Killed |
| 317 | `registry.go` | Approval: any reference | Killed |
| 318 | `registry.go` | Approval: any subject text | Killed |
| 319 | `registry.go` | Another approver is a repeat | Killed |
| 320 | `registry.go` | Another reference is a repeat | Killed |
| 321 | `registry.go` | Another subject is a repeat | Killed |
| 322 | `registry.go` | An approval in any state with an impact | Killed |
| 323 | `registry.go` | An approval on an impact of a past epoch | Killed |
| 324 | `registry.go` | An approval of a sample | Killed |
| 325 | `registry.go` | An approval with subjects without a value | Killed |
| 326 | `registry.go` | An approval with subjects that failed | Killed |
| 327 | `registry.go` | An approval of another subject | Killed |
| 328 | `registry.go` | An approval by the author | Killed |
| 329 | `registry.go` | An approval that does not raise the sequence | Killed |
| 330 | `registry.go` | An approval with another time | Killed |
| 331 | `registry.go` | Activation of the active revision refused | Killed |
| 332 | `registry.go` | Activation on an approval of a past epoch | Killed |
| 333 | `registry.go` | Activation without an approval | Killed |
| 334 | `registry.go` | Two active revisions | Killed |
| 335 | `registry.go` | A supersession without its time | Killed |
| 336 | `registry.go` | An activation that leaves the epoch | Killed |
| 337 | `registry.go` | An activation that does not raise the sequence | Killed |
| 338 | `registry.go` | An activation without its time | Killed |
| 339 | `registry.go` | Withdraw: any workspace | Killed |
| 340 | `registry.go` | Withdraw: any set | Killed |
| 341 | `registry.go` | Withdraw: any number | Killed |
| 342 | `registry.go` | Withdraw: any reason | Killed |
| 343 | `registry.go` | A withdrawal with another reason is a repeat | Killed |
| 344 | `registry.go` | A stop that leaves the epoch | Killed |
| 345 | `registry.go` | A stopped revision still returned as active | Killed |
| 346 | `registry.go` | Every withdrawal raises the epoch | Killed |
| 347 | `registry.go` | A withdrawal that does not raise the sequence | Killed |
| 348 | `registry.go` | A withdrawal with another time | Killed |
| 349 | `types.go` | A reason refused | Killed |
| 350 | `read.go` | Inputs counted after the form | Killed |
| 351 | `read.go` | Evaluate: any workspace | Killed |
| 352 | `read.go` | Evaluate: any set | Killed |
| 353 | `read.go` | Evaluate: any number | Killed |
| 354 | `read.go` | Evaluate: a target judged after the lookup | Killed |
| 355 | `read.go` | Evaluate: inputs judged after the lookup | Killed |
| 356 | `read.go` | Content of another set | Killed |
| 357 | `read.go` | A revision without a compiled set not found | Killed |
| 358 | `read.go` | A result without its digest | Killed |
| 359 | `read.go` | A result without its revision | Killed |
| 360 | `read.go` | A result without its set | Killed |
| 361 | `read.go` | Active: any identifiers | Killed |
| 362 | `read.go` | Active without a digest | Killed |
| 363 | `read.go` | Revision: any identifiers | Killed |
| 364 | `read.go` | Content: any identifiers | Killed |
| 365 | `read.go` | Content read is the stored set | Killed |
| 366 | `read.go` | Revisions: any workspace | Killed |
| 367 | `read.go` | Revisions: any set | Killed |
| 368 | `read.go` | Revisions: after below zero | Killed |
| 369 | `read.go` | Revisions: after without a bound | Killed |
| 370 | `read.go` | Revisions: a page of nothing | Killed |
| 371 | `read.go` | Revisions: a page of any size | Killed |
| 372 | `read.go` | A page that starts at its cursor | Killed |
| 373 | `read.go` | Revisions in reverse | Killed |
| 374 | `read.go` | A page longer than asked | Killed |

# Deviations from the plan

Recorded in the plan's change review resolution: the compiled order is first-in-first-out over formulas sorted by identifier; `Content` returns the order of the print; a printed value has no trailing zeros; a well-formed input name the set does not declare is judged after the lookup; counts of an impact are at most 1,000,000,000.

# Independent review

One read-only reviewer, separate from the writer, on 2026-11-13: design review FAIL without need for redesign (F1 to F21), design re-review PASS with findings (N1 to N12), change review PASS with findings and no blocker (C1 to C12). The reviewer read and did not run anything. W-17 asks for independent verification by a DBA and by QA; that was not done.

# Provisional choices awaiting the owner

The ten kinds of node and four types, without conditionals; one rounding per formula; a reference sees a rounded amount and an exact value of any other type; the lowest rank number applies and nothing falls back; an approver other than the author as the only separation of duties; an approval needs a full impact with no subject without a value or failed, a full impact over zero subjects being accepted; a quantum not tied to a currency. Not conservative, stated: an impact record is a caller's statement and only its contradictions are refused; nothing verifies a manifest's digest; author and approver are identifiers that nothing ties to a person; the active revision can be withdrawn without impact or approval, and no actor is recorded for an impact, an activation or a withdrawal. The open decisions of HB-17, which operators, variables and aggregation scopes exist at launch and when a code extension may be activated, are not made by this ticket.

# Not verified

System cases RATE01, SEC01, COM07 and AUD-AR-07 remain NOT_RUN. Not verified: any use of the package by another; behaviour at the upper bounds of the limits (1,000 formulas of 10,000 nodes, 10,000 digits) in time and memory; a build on a 32-bit platform; any operating system but the one named; persistence and reproduction across a restart. This ticket does not complete W-17.
