---
type: ImplementationPlan
title: Exact-decimal primitives and largest_remainder_v1 calculator
description: Defines the first server-side numeric packages, an exact bounded decimal with explicit rounding and a whole-pool allocation calculator, implemented in Go against the proposed numeric and allocation contracts with a synthetic profile.
status: reviewed
---

# Scope and ownership

[HB-167](https://linear.app/hybrid-billing/issue/HB-167/exact-decimal-primitives-and-largest-remainder-v1-allocation) is the first increment of the remaining build. On 2026-11-12 the owner instructed that the remaining scope be built in delivery order without per-item approval. Base: main `a3f598a`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, as billing amounts require. Machine: mac_mini.

# Observed baseline

Facts from a read-only survey of the repository on 2026-11-12 (no test was run for the survey):

- None of the 43 planned work items is implemented beyond the FOCUS collection and read slice. There is no server-side money type, rounding or allocation. B03, B04 and B05 compute drafts in the browser with BigInt.
- [Current design decisions](../../architecture/current-design-decisions.md) set Go for the backend (APIs and business coordination) and Python for source postprocessing. `go.mod` declares Go 1.25 with the standard library only; the two Go modules (`modules/customeraccess`, `modules/customerusage`) keep tests beside the code.
- The proposed repository layout puts verified decimal, currency, unit, period and identifier values in `packages/primitives/` and allocation in `modules/allocation/`.
- The [Claim and numeric contract](../../contracts/billing/claim-and-numeric.md) specifies the bounded exact-decimal profile, explicit division and rounding, six rounding modes, quantum rounding and `largest_remainder_v1`, with cases NM-01 to NM-06, all NOT_RUN. The [allocation profile contract](../../contracts/billing/allocation-profile.md) adds the rounding component, canonical target order, the manifest check and calculator cases AP-T01 to AP-T05, and states that the first calculator may use an explicitly synthetic complete profile and activates nothing.
- The Python ingestion parsers accept exponent notation, which the numeric contract forbids for amounts; they are not reused.

# Decision

Two Go packages with no storage, no I/O and no dependency outside the standard library.

## `packages/primitives/decimal`

| Area | Decision |
| --- | --- |
| Value | Immutable `Decimal`: integer coefficient (`math/big`) and non-negative scale, canonical: trailing fractional zeros removed, negative zero normalized to zero. The zero value `Decimal{}` is not a number: every operation rejects it as a missing operand, so an unset field can never be read as zero. `String()` prints the canonical finite base-10 form without exponent. `Equal`, `Sign`, `Scale` and a copying `Coefficient()` are the read surface; no text or JSON unmarshalling is offered, because it could not take a profile |
| Parsing | `Parse(profile, text)` accepts the grammar `-?(0|[1-9][0-9]*)(\.[0-9]+)?` in ASCII and nothing else: no plus sign, whitespace, exponent, `NaN`, `Infinity`, grouping, bare or trailing point, or non-ASCII digit. `-0` and `-0.00` are accepted and are zero. A redundant leading zero (`00.1`) is rejected; this is stricter than the contract's wording and is a recorded choice, made so that one text has one reading and the integer-digit bound counts what was written. The input bounds are checked on the submitted digits before canonicalization. The original input bytes are evidence the caller keeps; the package does not carry them |
| Profile | An opaque validated value built by `NewProfile(spec)`; the zero `Profile{}` is rejected by every operation. The spec carries ID, revision, input integer digits and scale, result integer digits and scale, and an intermediate digit bound; any missing bound, a result bound below the input bound, or an intermediate bound below the result size is rejected. Effective scope, allowed signs per field, units and conversions, rounding stages and expected-case approval are activation and caller concerns outside a calculator. Bounds mean: integer digits of the absolute value at most the maximum, canonical scale at most the maximum |
| Exact operations | `Add`, `Sub`, `Mul`, `Neg`, `Cmp` check both operands against the passed profile, since a value may have been built under a wider one. `Add`, `Sub`, `Mul` return the exact result or `ErrResultBound`; `Mul` and the divisions first check digit counts against the intermediate bound and return `ErrIntermediateBound` without widening. `Quo` returns the exact quotient only when it terminates within the result scale; a recurring or over-long quotient is `ErrInexactDivision` in both cases; a zero divisor is `ErrDivisionByZero` |
| Explicit rounding | `DivideRound(n, d, scale, mode)`, `Round(x, scale, mode)`, `RoundToQuantum(x, quantum, mode)` and `Quanta` (the integer number of quanta) share one core: for an integer ratio N/D with D positive, take the truncated quotient q and remainder r, then `toward_zero` q; `away_from_zero` q plus the sign of N when r is not zero; `floor` q minus one when r is negative; `ceiling` q plus one when r is positive; `half_up` q plus the sign of N when twice the absolute remainder is at least D; `half_even` the same when it exceeds D, or equals D and q is odd. A scale below zero or above the result scale is rejected; a scale at or above the value's own is the identity. Result bounds apply to rounded results. The zero `Mode` is invalid; `ParseMode` accepts only the six contract names |
| Money | `Money` pairs a `Decimal` with a currency code of three uppercase ASCII letters; the shape check is not a currency list. `AddMoney` and `SubMoney` reject different currencies; there is no conversion |
| Errors | Sentinel errors with fixed text that never contains an input value |

## `modules/allocation`

`LargestRemainderV1(Request)` implements the contract's algorithm. The request carries the profile, `Limits{MaxTargets, MaxIntermediateDigits}`, the target-ID encoding, total, quantum, mode and targets; all are required.

1. Reject a missing profile, missing limits, an encoding other than `ascii_bytewise_v1`, no targets, or more targets than the limit.
2. For every target: the ID must be non-empty, at most 128 bytes, from `A-Z a-z 0-9 . _ : -`, and not repeated; the weight must be a present non-negative value within the profile. The exact decimal sum of the weights is the denominator and must be positive.
3. Quantize the total to the quantum with the mode. The quantized total and the rounding component (original minus quantized) must fit the result bounds.
4. Scale the weights losslessly to integers W with sum S, and reject with the allocation intermediate error when the digits of T plus the digits of S exceed the limit, T being the absolute number of quanta. Since no W exceeds S, this one check bounds every product.
5. Each target gets the integer quotient of `T × W ÷ S`; the remaining quanta go one each in descending order of the exact remainder, ties by ascending bytewise ID.
6. Restore the sign. Return algorithm name, encoding, profile ID and revision, mode, quantum, original, quantized, rounding component, the exact decimal denominator and the shares in ascending bytewise ID order.
7. Before returning, require the signed sum of the shares to equal the quantized total and original to equal that sum plus the rounding component; otherwise return an internal conservation error and no result.

`ascii_bytewise_v1` is a named, fail-closed encoding for this increment. It is synthetic: the production canonical ID encoding remains a product decision listed by the allocation contract, and a second encoding can be added beside it. Raw-input bounds such as excess scale are rejected by `Parse`; the calculator re-checks canonical operands only. Caps, minima and other algorithms are absent from the API.

## Values that are not decided here

No precision, scale, quantum, rounding stage, currency list, tolerance, target limit, intermediate limit or ID encoding of any customer or finance policy is chosen. Every bound used by the tests belongs to a synthetic profile or limit declared in the tests. Neither package can be used without caller-supplied validated values.

# Allowed files

`packages/primitives/decimal/*` and `modules/allocation/*` (new); `docs/features/HB-167/*`.

# Verification

- Contract cases as table tests whose expected values are literals taken from the contracts or worked by hand, never values produced by the code under test: NM-01, NM-02, NM-03 (overflow, excess scale, float and exponent text, mixed currency, missing profile, intermediate bound), NM-04, the calculator part of NM-05 (duplicate target, zero total weight, negative weight), AP-T01, AP-T02, AP-T03, AP-T04 and AP-T05. Of AP-T01, AP-T05 and NM-04 only the calculator part is exercised: exclusion reasons, filtered views, separately approved revisions and chunking are pool and view semantics. NM-06 and the pipeline, authority and UI cases stay NOT_RUN, and the contracts' NOT_RUN labels are not changed by this ticket.
- Rounding: every mode on positive and negative ties and non-ties against literal values, including `-0.5` half-up to `-1` and `-2.5` and `-1.5` half-even to `-2`; quantum rounding with quanta that are not powers of ten; totals that round to zero.
- Property tests with a fixed-seed generator and a stated number of cases. The oracle is built from the generator's own text with `math/big.Rat` and checks defining predicates instead of repeating the implementation's steps: exact operations equal the rational result, and a rejection is accepted only when the rational result is not representable; a rounded value is a multiple of the unit and satisfies its mode's inequality and tie rule; an allocation is characterized by: every share is the floor of its exact share or that plus one quantum, the shares sum to the quantized total, and no target that received an extra quantum has a smaller remainder than, or an equal remainder and a larger ID than, a target that did not. Also asserted: ascending ID order of the result, zero weight gives zero, permutation invariance, invariance under multiplying every weight by the same positive factor within bounds, and that negating the total with floor and ceiling swapped negates every share. Not asserted, because they do not hold: associativity under bounds, distributing rounding over sums, and splitting, merging or chunk invariance. A run whose tie, negative, zero-weight or rejection categories are empty fails.
- A Go fuzz target for `Parse` that decides acceptance with an independent regular expression and digit counts, compares the value with `big.Rat` and checks the canonical print. Its seed corpus runs in every test run; a timed `-fuzz` run is exploratory and not reproducible.
- Mutation check: temporary single-point changes to the tie order, remainder order, three rounding rules, the input scale check, the exactness check and sign restoration must each fail the tests.
- `go vet ./...`, `gofmt -l`, `go test ./...` for the whole module.
- Independent review of the actual change.

Not verified by this ticket: durability, concurrency, authority, approval, any database or driver behavior, performance at scale, behavior on other operating systems, and any real numeric policy.

# Deferred

The restricted formula AST; FX conversion and rounding stages (W-18); allocation pool snapshots, approval subjects and publication (W-14); Claim reservations (W-21); commands and API; replacing the browser drafts of B03 to B05 with server results.

# Design review resolution

The independent design review of 2026-11-12 (read-only, nothing executed) returned FAIL without a blocker: 4 HIGH, 6 MEDIUM, 5 LOW. Resolution:

| Finding | Resolution |
| --- | --- |
| F1 no intermediate bound or overflow test | Intermediate digit bound in the profile and in the allocation limits, checked before multiplication; overflow cases tested |
| F2 zero values undefined | Zero `Decimal`, `Profile` and `Mode` are invalid and rejected; profile is opaque; operands are checked against the passed profile |
| F3 sign-inversion property false for floor and ceiling | Property uses the mirrored mode; literal cases for totals that round to zero |
| F4 oracle independence and fuzz self-comparison | Oracle from generator text, predicates instead of procedure, independent grammar in the fuzz target |
| F5 target limit inside the decimal profile | Moved to allocation limits |
| F6 rounding core, domains and bounds | One core with the stated rule; scale domain and bounds pinned |
| F7 division error ambiguity and name | One inexact-division error; `DivideRound` |
| F8 unversioned ID encoding | Named `ascii_bytewise_v1`, required in the request, echoed in the result, labelled synthetic |
| F9 ambiguous denominator | Exact decimal sum of weights; both reconciliation identities checked |
| F10 public surface for allocation | `Sign`, `Cmp`, `Neg`, `Equal`, `Scale`, copying `Coefficient`, `New`, `Quanta` |
| F11 raw bounds and evidence bytes | Stated as `Parse`'s and the caller's responsibility |
| F12 leading-zero rejection | Kept and recorded as a choice stricter than the contract; a contract clarification would be its own ticket |
| F13 case scope | Calculator parts only; NOT_RUN labels untouched |
| F14 generator hygiene | Seeded source, deliberate tie classes, category counters that fail when empty |
| F15 money and error text | Minimal money; allocation sentinels; fixed error text |

The reviewer's hypothesis that the owner's earlier "Codex implements" split still applies was checked against the owner's words of 2026-11-12, which instruct this session to build the remaining scope.
