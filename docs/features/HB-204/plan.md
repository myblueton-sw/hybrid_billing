---
type: ImplementationPlan
title: A deterministic rating engine with charge types, rule precedence, pool-wide tiers and bounds, and a trace of every step
description: Defines the first model of work item W-16, a pure rating engine that compiles a policy once and turns the lines of one contract pool into charge lines with a trace, telling seven charge types apart, with an included quantity, discounts and bounds, choosing the rule of a line by a fixed precedence, computing on the whole pool before distributing, and keeping a line with an unknown input out of the result instead of rating it as zero.
status: draft
---

# Scope and ownership

[HB-204](https://linear.app/hybrid-billing/issue/HB-204/w-16-a-deterministic-rating-engine-with-charge-types-rule-precedence) is the first increment of work item W-16 (requirement HB-16). It is the twenty-first increment built under the owner's instruction of 2026-11-12. Base: main `b6a7ef3`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the engine computes billing amounts. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-13:

- Requirement HB-16: cost pass-through, markup, target margin, a fixed unit price, a flat fee, graduated and whole-range tiers, an included quantity, overage and a minimum and maximum are told apart. The CSP's own usage or commitment engine is not rebuilt. Proposed direction: a pure deterministic engine that makes ChargeLine and RuleTrace from input, policy, aggregation scope and version; the precedence SKU exception, then service or contract, then an explicit group template, then the default, fixed as a contract, with the input basis of every step kept. Scale: validate and compile a policy once and avoid a lookup per row; compute included quantities, tiers and minimums on the whole contract pool and then distribute, so that a chunk boundary does not change a price.
- Acceptance of HB-16: 100 with a markup of 20% is 120 and with a target margin of 20% is 125; the source's examples of 1,160,000 won, 850,000 won and 14,000 won, and changes, negatives and stacking, are verified against independent expected values. The inputs of those three examples are not in the planning files read. Linked source cases RATE01 (the same result for the same input under versions of rate, period and exchange rate), COM07 and SIM03 are NOT_RUN.
- Open decisions of HB-16, not made: the charge types at launch, how discounts stack, the digits of amounts and quantities and the rounding mode.
- AR-01: under a direct contract the CSP's usage is a basis for viewing and allocation and is not charged again by the MSP; under resale only approved, eligible components are sold. AR-04: the denominator of a token rate (per thousand, per million) is explicit. AR-07 (manual adjustments as drafts) and AR-10 (what a flat fee includes) name HB-16 as well.
- Existing code: `packages/primitives/decimal` multiplies exactly, divides exactly or refuses, and rounds a ratio to a count of quanta under a named mode. `modules/allocation` distributes a quantized total over weighted targets with ties broken by identifier. `modules/contracts` (HB-178) prorates a fixed periodic amount; nothing computes a selling amount from cost or quantity.

# Decision

One Go package, `modules/rating`, standard library, the decimal primitive and `modules/allocation` only. It is pure: `Compile(profile, limits, policy)` validates a policy once and returns a compiled policy that holds its own copy and cannot change; `Rate(lines)` on it turns the lines of one contract pool into a result. Nothing is stored, nothing calls it, and no clock is read.

What this can and cannot show: that a result is a function of the policy and the set of lines, equal in every field and order for any order of the lines; that every charge is a whole multiple of the policy's quantum; that the charges of a rule equal, kind by kind, the steps of its trace; that a line is rated by at most one rule, the most specific that names it; that every line given appears exactly once, with a charge, as covered by a flat fee, or excluded with a reason; that a rule one of whose lines lacks a needed input is held as a whole and charges nothing; and that a rule's amount is computed on the sums over its lines, so that cutting one usage into several lines cannot change it. It cannot know that a cost, a quantity, a class or a policy is true or approved, that the lines given are the whole pool, that a cost is of the basis the policy names, or that a result was already billed. A caller who rates a pool in parts gets other prices for tiers, included quantities and bounds; passing the whole pool is the caller's obligation, and the result says how many lines it was given.

The engine does not price cloud usage: a cost is an input, and nothing here knows a CSP's rate card, commitment or discount. It reads no cost pool, allocation or contract; relating its input to `modules/normalize`, `modules/reconcile`, `modules/contracts` and its output to a billing is deferred. `modules/contracts` prorates a fixed periodic amount with a scale of digits; a flat fee here is not prorated and is stated in quanta, and nothing keeps one fee from being charged by both (stated).

Not delivered, although the requirement or its details name it: a rule of the contract as a level of its own, and precedence across several sources of policy, the requirement's second level being "service or contract" and its third an explicit group template, which is read here, as an assumption, as a group that a line names; a fee as a rate of a basis that is not itself charged, which a direct contract's management fee on CSP usage would need; the source's three examples, whose inputs are not in the planning files; manual adjustments as drafts (AR-07); what a flat fee includes (AR-10); model aliases and validity periods of prices (AR-04); versions of period and exchange rate (RATE01, W-18); proration; taxes. The design part of HB-16 stays open with the owner. W-16 asks for independent verification by a DBA and by QA; this ticket has one independent reviewer of design and change, and the rest remains.

Alternatives considered. Rounding each line and adding up: not taken, the split of a usage into lines would change the price. A registry of policies with versions: not taken in this increment; a policy carries an identifier and a version as labels, and storing and approving policies belongs with W-17 and W-23. Carrying intermediate values in the decimal primitive: not taken, since a product of a cost and a rate, or a quotient by a denominator of a million, exceeds the scale that the profiles in use can hold, and valid policies would end in bound errors or be rounded step by step; intermediate values are exact fractions of whole numbers of any size inside the package, and only inputs and final amounts pass through the primitive.

Every string is an identifier under the project's identifier rule (letters, digits, `. _ : -`, at most 128 bytes), of which the package has its own copy that a test compares with the allocation's, a three-letter uppercase currency code, the text of a number, or a value of a closed list. There is no free text. Keys of SKUs and services that do not fit the rule are mapped by the caller, always in the same way (stated obligation). No floating point and no floating or rational type of the standard library is used in the source.

## Forms and arithmetic

A number is a text the decimal primitive parses with the profile; a text it refuses is malformed. A cost may be negative. Every other number may not, and a text that starts with a minus sign is malformed for it. A rate of a markup is at least 0; a target margin and a percentage discount are at least 0 and below 1. The empty text means not known for a line's cost and quantity and is malformed elsewhere. A quantum is one of `1`, `0.1`, `0.01`, `0.001`, `0.0001`, and one the profile cannot parse is malformed. An amount of a flat fee, a fixed discount, a minimum and a maximum must be a whole multiple of the quantum, which for canonical values is that its scale is not greater than the quantum's; otherwise the policy is malformed. Amounts are returned as the primitive prints them.

Inside the package a value is an exact fraction, a whole numerator over a whole denominator above zero. A value is rounded once, to a whole count of quanta, under the policy's mode; the amount is that count times the quantum and must be a value the profile holds, else a limit error. The modes, by name as the primitive has them and with the same meaning for both signs: `toward_zero`; `away_from_zero`; `floor`, toward the smaller value; `ceiling`, toward the larger; `half_up`, to the nearest and a tie away from zero; `half_even`, to the nearest and a tie to the even count. So `ceiling` makes a credit smaller and `floor` larger. A test compares the package's rounding with the primitive's on values both can hold.

## Policy

A policy has an identifier and a version, labels that the result repeats; a currency; a cost basis of a closed list (`billed`, `effective`, `list`, `contracted`), a label that the result repeats and that says which cost the caller states it passes; a quantum and a mode, required and without default; a transaction path, `direct` or `resale`; for resale the classes of lines that may be charged, which may be none; and at least one rule. A direct policy carries no list of classes: under it only the class `service_fee` is charged (AUD-AR-01 at the level of this package).

A rule has an identifier unique in the policy, a level and a key, a charge type with its parameters, and optionally discounts, a minimum and a maximum. Levels, from the most to the least specific: `sku`, `service`, `group`, each with a key, and `default`, without one. At most one rule per level and key; a policy without a default rule is valid. A line is rated by the first of: the rule of its SKU, of its service, of its group, the default; rules are not combined.

Charge types. For each, the base is a fraction formed from sums over the lines of the rule that are rated, and is never below zero:

- `pass_through`: the sum of the costs above zero.
- `markup`: that sum times one plus the rate.
- `target_margin`: that sum divided by one less the margin.
- `unit_price`: the quantity above the included quantity, times the price, divided by the price's denominator, which is 1, 1,000 or 1,000,000.
- `graduated`: the quantity above the included quantity is cut along the tiers, each part priced at its tier's price and denominator, and the parts added as fractions before anything is rounded.
- `volume`: the quantity above the included quantity is priced at the price and denominator of the one tier that this quantity falls in.
- `flat`: the amount. A flat rule is not allowed at the level `default`.

For the three cost-based types the costs below zero are not part of the base: their sum is passed at cost, rounded once on its own (provisional: a credit is not marked up). Tiers are given in rising order of their upper bound, each above zero and above the one before, the last without a bound; a quantity exactly on a bound belongs to the tier below it. An included quantity may be given for the three quantity-based types.

The base is rounded once to the base amount. Then, in this order (provisional): discounts in the order given, each a percentage of what is left after the discounts before it, rounded once, or a fixed amount, and never more than what is left; a minimum, which adds what the amount lacks; a maximum, which takes off what it exceeds. A minimum above the maximum is malformed. Discounts and bounds see the base amount only, which is zero or more; what is passed at cost is outside them.

## Lines, exclusions and held rules

A line has an identifier unique in the call, a class (`csp_usage`, `service_fee`, `licence`, `other`), a SKU, a service and a group that may each be none, a cost and a quantity that may each be not known, and a currency, which must be the policy's. A malformed line, a second line with the same identifier among them, fails the whole call.

A line is excluded, with one reason, in this order of tests: `class_not_eligible`, a class the path or the policy does not charge; `no_rule`; `cost_not_known` or `quantity_not_known`, when the type of its rule needs that input and it is not known. A flat rule needs neither input. A rule with a line excluded for an unknown input is held: it has a trace that says so with the number of such lines, it charges nothing, and its other lines are excluded with the reason `rule_held`. So an unknown input is not rated as zero, neither for its line nor in a sum that decides a tier, an included quantity, a discount or a bound.

## Result

`Rate` returns: labels, the policy's identifier, version, currency, cost basis, path, quantum and mode, the profile's identifier and revision and the algorithm `rating_v1`; the number of lines given; charges; lines covered by a flat fee; excluded lines; a trace and a total per rule that had a line that was not excluded for class or lack of a rule; the grand total; whether every input was known, which is that no rule is held; and the number of excluded lines per reason. `Rate` of no lines is a result without charges.

Every line given appears exactly once: with a charge, as covered by the flat rule that names it, or excluded. A charge has a rule, a kind, an amount, and a line or none. Kinds and what they must equal: `rated`, a line's share of the base amount, zero included, for a line whose cost is zero or more under a cost-based type and for every line under a quantity-based type; `passed_at_cost`, a line's share of what is passed at cost, for a line whose cost is below zero; and, belonging to the rule and to no line, `flat`, `discount` once per discount, `minimum_top_up` and `maximum_cap`, a discount and a cap being negative. A charge of a rule without a line is returned only when it is not zero; its step in the trace is always there.

Shares are made by `LargestRemainderV1` with the quantum and the mode `half_even`, which has no effect on a whole count of quanta: the base amount over the lines that have a `rated` charge, weighted by cost for the cost-based types and by quantity for the quantity-based ones; what is passed at cost over the lines whose cost is below zero, weighted by its size. When every weight is zero the amount is zero and every share is zero without a call; an amount other than zero with weights of zero cannot arise and is a self-check error.

Order: traces and totals by rule identifier; charges by rule identifier, then by kind in the order named above, discounts in their order, then by line identifier; covered and excluded lines by line identifier. Nothing in a result follows the order of a map.

A trace is a closed list of steps for its type, each with a name, a basis, a parameter and an outcome as texts, a fraction being written as numerator and denominator: the sums of costs above and below zero, or the sum of quantities, the included quantity and what is above it; each tier's part and price; the base as a fraction; the base amount; what is passed at cost as a fraction and as an amount; each discount with what was left before it; the minimum; the maximum; and the total. The total of a rule is the sum of its charges, and the engine checks before it returns that the `rated` charges add up to the base amount, the `passed_at_cost` charges to their step, and all charges to the total.

## Errors and limits

Errors are fixed values and repeat no string of the input: invalid, for a malformed policy or line, a text the primitive refuses for syntax or input bound among it; limit, for more rules, tiers, discounts or lines than the limits allow, an amount the profile cannot hold (`decimal.ErrResultBound`, `decimal.ErrIntermediateBound`) or the allocation's bound (`allocation.ErrIntermediateBound`); and a self-check error for every other failure of the primitive or the allocation and for a total that does not equal its trace. No foreign error value is returned, and an error returns no partial result. Limits, each required and greater than zero: rules, tiers per rule, discounts per rule, lines per call, and the digits of an intermediate in the allocation. A compiled policy is safe for use by several callers at once.

## What is not decided here

Provisional defaults, each to be confirmed or changed by the owner: the seven charge types with an included quantity, discounts and bounds; one rounding per amount, to the policy's quantum and mode; discounts stack one after the other on what is left, before the bounds; a credit is passed at cost and takes no markup, margin, discount or bound; the precedence stops at the first rule that names a line; a quantity on a tier's bound belongs to the lower tier; a flat fee and a minimum are charged when the rule has a rated line, also one without usage, and not when it has none, so a month without a line charges neither; a rule with one unknown input charges nothing. Not conservative, stated: a caller who rates part of a pool gets a result that looks valid; classes, eligibility and the cost basis are the caller's statements; nothing ties a policy to an approval; a correction that arrives as a cost of 100 and a cost of minus 100 for the same usage gives 20 under a markup of 20% and 25 under a target margin of 20% on a net cost of zero, so corrections of one usage must be netted by the caller before rating (stated obligation); a flat rule covers every line it names without a further charge, which says what the fee includes without AR-10 being delivered. Assumptions: one currency per policy; quantities are never negative; the reading of the precedence.

# Allowed files

`modules/rating/*` (new); `docs/features/HB-204/*`.

# Verification

- The cases of the acceptance at the level of this package, as literal tests with expected values worked out apart from the code: 100 with a markup of 20% is 120; 100 with a target margin of 20% is 125; a change of the policy's version changes nothing but the label; costs below zero under each cost-based type, a sum below zero and the pair of 100 and minus 100; discounts that stack; the same lines under a direct contract, where only service fees are charged, and under resale. The three examples of the source are recorded as not verified.
- Literal table tests: every refusal of a policy and of a line, amounts that are not multiples of the quantum, a flat rule at the default level, a direct policy with classes, empty and duplicate lines; every charge type with its edges, under a profile whose scale is four with inputs at full scale; a quantity on a tier's bound, one tier, an included quantity above the quantity, denominators of 1, 1,000 and 1,000,000, a graduated rule with mixed denominators where rounding per tier would differ; each mode with ties above and below zero, and the package's rounding against the primitive's; each quantum; a target margin whose ratio does not end; discounts of both kinds and one larger than what is left; a minimum and a maximum, alone and together; a rule with lines and no usage against a rule without a line; every reason of exclusion and their order; a rule held by one unknown input beside a minimum, and one whose every line is unknown; the precedence with a line that four rules name; shares with ties and with weights of zero; charges against the steps of their trace; the order of every list; labels; limits and bounds of the profile as limit errors; identifiers compared with the allocation's; errors that are fixed values; a scan of the source for floating point, for a floating or rational type and for comments. The self-check error is produced by no test.
- Properties with fixed seeds over random policies and pools against a reference model written in the test with `math/big.Rat`, which computes every base, discount and bound exactly and rounds with its own implementation of the modes as this plan defines them: the amounts per rule, step by step, the total, the exclusions and held rules equal the model's; every charge is a multiple of the quantum; the shares of a rule add up to their amount and each differs from its exact proportion by less than one quantum; the whole result, in order, is equal for the lines in another order; splitting a line into two with the same keys whose cost or quantity add up exactly to the original leaves every amount of the rule unchanged, the shares apart; raising a cost above zero or a quantity never lowers a rule's base, for every type but `volume`. The model shares the author's reading of the plan; the shares are not in it. The test fails unless every charge type, every kind of charge, every reason of exclusion and a held rule were reached. No random number is drawn while ranging over a map.
- One compiled policy rated by several goroutines under the race detector with equal results; a policy changed by the caller after `Compile` changes nothing.
- A pool of the configured maximum of lines is rated.
- The package's tests run many times in a row before any mutation run; a mutation check by a script in Python with results appended to files that are never overwritten, kept outside the repository.
- `gofmt -l`, `go vet ./...`, `go test -race -count=1 ./...`.

Remaining NOT_RUN at system level after this ticket: RATE01, COM07, SIM03, AUD-AR-01, AUD-AR-04, AUD-AR-07, AUD-AR-10. This ticket does not complete W-16.

# Deferred

Stored, versioned and approved policies; free formulas (W-17); exchange rates and versions of rounding (W-18); manual adjustments as drafts; inclusion of power or services in a flat fee; model aliases and validity of prices; proration and period splits from `modules/contracts`; taking input from normalised and reconciled cost; passing charges to a billing ledger; taxes; commands, permissions and screens.

# Design review resolution

The independent design review of 2026-11-13 (read-only, nothing executed) returned FAIL without asking for a redesign: 3 HIGH, 11 MEDIUM, 7 LOW. It found the structure sound and the two acceptance figures exact. Its HIGH findings were that the exact intermediate values could not be carried by the decimal primitive under the profiles in use, that a line excluded for an unknown input still let its rule be rated as if the input were zero, and that `pass_through` with costs below zero was undefined. All findings were adopted above.

| Finding | Resolution |
| --- | --- |
| F1 intermediates the primitive cannot hold | Values are exact fractions of whole numbers inside the package, rounded once by the package's own rounding of a ratio, compared with the primitive's by a test; only inputs and final amounts pass through the primitive |
| F2 an unknown input rated as zero in a pool | A rule with such a line is held, charges nothing, and its other lines are excluded with `rule_held` |
| F3 `pass_through` with costs below zero | The same split as the other cost-based types; the base is never below zero |
| F4 a correction read as a credit | Stated with the example, and netting as the caller's obligation |
| F5 amounts that are not multiples of the quantum | Refused when the policy is compiled; a minus sign refused where a number may not be negative |
| F6 modes for values below zero | Defined for both signs |
| F7, F8 which lines appear and how | Every line exactly once; kinds and when a rule's charge is returned; the allocation's mode; weights of zero |
| F9, F10 order, empty and degenerate input, purity | Orders fixed; each case stated; the policy is copied |
| F11 the precedence | The reading recorded as an assumption; a contract level and several sources listed as not delivered |
| F12 a direct contract | Only service fees are charged; a fee on a basis that is not charged listed as not delivered |
| F13 flat fees | Not at the default level; when a flat fee and a minimum are charged, the overlap with proration and what a flat rule covers stated |
| F14 labels | Profile, quantum, mode, path, cost basis and an algorithm label in the result |
| F15 `complete` | Whether every input was known, with numbers of exclusions per reason |
| F16 to F18 names, error mapping, the trace | Seven types; at most one rule; a kind for what is passed at cost; errors enumerated; a closed list of steps with a total |
| F19, F20 properties and tests | The property about raising an input excludes `volume`; the split property restated; tests listed |
| F21 the basis of a cost | A label on the policy that the result repeats |


# Design re-review resolution

The narrow re-review of 2026-11-13 (read-only, nothing executed) returned PASS with findings and no blocker: of the first findings it held thirteen resolved and the rest partly; twelve new findings, four MEDIUM and eight LOW. It confirmed by reading that every base is a well-defined exact fraction, that an amount always goes back into the primitive or ends in a bound error, that the modes as defined agree with the primitive's for both signs, and that a percentage discount cannot exceed what is left under any mode. It said implementation may start after four text fixes. The rules below settle all twelve and are part of the decision; where they differ from a sentence above, they govern. The change review reads them against the code.

- N1, when a rule charges. A rule is active when it has at least one line that is not excluded. A flat fee and a minimum are charged for an active rule and for no other. So a cost-based rule whose every line is a credit is active: its base is zero, its credits are passed at cost, and its minimum, if it has one, is charged beside them (provisional, stated).
- N2. The size of a percentage discount is rounded as a value above zero and charged as its negative.
- N3. A flat rule has no `rated` charge: its `flat` charge equals its base amount. For every other type the `rated` charges add up to the base amount.
- N4. Cutting a usage into lines leaves a rule's amounts unchanged when the parts have the sign of the whole. Parts of different sign are a cost and a credit and are rated as such; the split property of the tests is restricted in the same way.
- N5. The allocation adds its weights through the primitive, so the sum of the quantities, or of the costs, of a rule is bound by the profile as well. Either bound error of the primitive, from the package's own use or from the allocation, is the limit error.
- N6, the steps of a trace, by name and in this order, each present exactly when it applies: for a cost-based type `costs_above_zero` and `costs_below_zero`; for a quantity-based type `quantity`, `included` and `above_included`; for `graduated` and `volume` one `tier` per tier that takes part, with its part and price; `base`, the fraction; `base_amount`; for a cost-based type `passed`, the fraction, and `passed_amount`; one `discount` per configured discount, with what was left before it; `minimum` and `maximum` when configured; `total`. A flat rule has `base_amount`, its discounts and bounds, and `total`. A held rule's trace has one step, `held`, with the number of lines whose input is not known, and its total is zero.
- N7. A fraction is written in lowest terms with a denominator above zero. Identifiers are ordered bytewise. The numbers of exclusions are given per reason in the order of the tests, `rule_held` last.
- N8. The bounds of tiers are measured on the quantity above the included quantity. A quantum is one of the five texts as written. A rule above the level of a SKU adds the quantities of its lines whatever their unit; giving such a rule only lines of one unit is the caller's obligation (stated).
- N9. A `passed_at_cost` charge is returned for its line also when it is zero.
- N10, errors. An invalid profile or invalid limits make `Compile` return invalid. In `Rate` the number of lines is tested before any line, so a call with too many lines is a limit error whatever the lines hold. Every bound, the grand total's included, is the limit error.
- N11. The mark of the result is that no rule is held, and it is named so: inputs that are not known on lines excluded for their class, without a rule, covered by a flat fee, or not needed by their rule's type do not change it.
- N12, not conservative, stated: a line with a cost below zero under a flat rule is covered, and its credit is not passed.

# Change review resolution

The independent change review of 2026-11-13 (read-only, nothing executed by the reviewer) returned PASS with findings and no blocker on amounts: three MEDIUM and seven LOW. It found no input for which a charge, a share or a total differs from the plan, recomputed the main expected values by hand and found them right, and held the compiled policy free of the caller's slices and safe for concurrent use. Its first finding was an error the author had introduced after the first mutation run: the running amount was no longer updated after a minimum, on the wrong ground that nothing read it, so the step of a maximum showed the amount before the minimum. All findings were resolved or stated; the corrections were not reviewed again by a separate reviewer.

| Finding | Resolution |
| --- | --- |
| C1 the maximum's step showed a stale amount | The running amount is updated again; literal traces for a maximum after a minimum, and the model now compares the basis of every discount and bound |
| C2 the check of the total against the trace was a tautology | The engine compares the sum of the charges with the running amount plus what is passed at cost, two computations, and returns the self-check error when they differ. That the `rated` and `passed_at_cost` charges add up to their amounts rests on the allocation's own check and on the checks of its result |
| C3 the model compared charges, not steps | It compares the names of the steps in order and the outcomes and bases of the base amount, what is passed, each discount and each bound |
| C4 a test that did not reach the allocation's sum of weights | A quantity-based case whose amount the profile holds and whose weights it does not |
| C5 a property asserted on the model only | What the engine rates per rule before and after raising an input is compared as well |
| C6 a mode the package does not round by | The package accepts its own closed list of six names |
| C7 unused code | Removed |
| C8 the source scan | Stated: it is a scan of words; that the source has no floating literal and no rational type rests on reading as well |
| C9 thresholds and the generator | The quantum `0.001` is drawn; the count of splits includes lines that are excluded or covered, stated |
| C10 the plan's wording on copies | Corrected here: a compiled policy keeps parsed values and none of the caller's slices, which has the effect of a copy |

Deviations from the plan, recorded here: code that tests showed could not change a result was removed after the first mutation run, an early return in the rounding for a ratio without remainder and a check of the profile at `Compile`, which parsing the quantum makes in any case.
