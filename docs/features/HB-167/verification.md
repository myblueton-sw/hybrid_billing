---
type: VerificationRecord
title: Exact-decimal primitives and allocation calculator verification
description: Records what was executed and reviewed for the first server-side numeric packages, with results, limits and the values that remain undecided.
status: draft
---

# Delivered behavior

[HB-167](https://linear.app/hybrid-billing/issue/HB-167/exact-decimal-primitives-and-largest-remainder-v1-allocation) adds two Go packages from main `a3f598a`, with no storage, I/O or dependency outside the standard library. Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan before implementation and the implemented change before merge. Machine: mac_mini. See the [plan](plan.md).

- `packages/primitives/decimal`: a canonical exact decimal, strict text parsing, an opaque validated numeric profile, exact addition, subtraction and multiplication within bounds, exact-only division, explicit `DivideRound`, `Round` and `RoundToQuantum` with the six contract rounding modes, and same-currency money.
- `modules/allocation`: `LargestRemainderV1`, the whole-pool allocation of the numeric contract with the rounding component, the exact denominator, a named target-ID encoding and the two reconciliation checks.

Nothing calls these packages yet. No customer value, numeric policy, profile or encoding is activated.

# Executed checks

| Check | Actual outcome |
| --- | --- |
| `go test ./...` | PASS: `packages/primitives/decimal`, `modules/allocation` and the two existing Go packages |
| `go vet ./...`, `gofmt -l` | No finding, no unformatted file |
| Contract cases (literal expected values) | PASS: NM-01, NM-02, NM-03, NM-04, calculator part of NM-05, AP-T01 to AP-T05 calculator parts. AP-T04's caps and minima are not exercised: the API has no such inputs |
| Mode and remainder cases added after the change review | PASS: each of ceiling, away from zero, floor, toward zero and half even on an allocation whose result differs from half up; a pool where the larger remainder belongs to the larger ID; unequal weights with a quantum of 0.03 |
| Rounding tables | PASS: 24 scale-zero cases over the six modes and both signs, 13 quantum cases including quanta 0.05, 0.25 and 3 |
| Decimal property tests, seed 20261112/167 | PASS: 4,000 operation cases (2,661 accepted, 1,339 rejected as not representable or division by zero) and 4,000 rounding cases (22 exact ties, 1,354 negative values, 1,289 rounded divisions) |
| Allocation property test, seed 20261112/1670 | PASS: 3,000 pools (2,964 allocated, 36 zero-weight pools rejected), 4,102 tie comparisons, 821 negative totals, 1,601 zero-weight targets; permutation, weight rescaling and mirrored sign inversion on every allocated pool |
| Parse fuzz target | Seed corpus PASS in every run; one exploratory 20 second run, about 1.06 million executions, no failure |
| Mutation check | Eight temporary single-point changes (tie order, remainder order, half-up tie, half-even parity, floor, input scale check, exactness check, sign restoration) each made the tests fail; all were reverted |

The contract cases NM-06 and the pipeline, authority, approval and UI cases of both contracts were not run and remain NOT_RUN. Only the calculator part of AP-T01, AP-T05 and NM-04 is covered.

Host: macOS arm64, Go 1.25.7. Other operating systems and architectures were not exercised; the code is pure computation over `math/big`.

# Values that remain undecided

Precision, scale, quantum, rounding stages and modes per stage, currency list, target and intermediate limits, and the production target-ID encoding are not chosen. The tests use synthetic profiles and limits. `ascii_bytewise_v1`, its alphabet and its 128-byte limit are a synthetic first encoding. The three-uppercase-letter currency shape is a format check, not a currency policy. Rejecting a redundant leading zero in input text is stricter than the contract's wording and is recorded as a choice.

# Limits

- No durability, concurrency, authority, approval, database or driver behavior is involved or verified.
- No performance or resource measurement was made; the intermediate bounds reject oversized work but their real values are a profile decision.
- The browser drafts of B03 to B05 were not compared with these packages; they may round or break ties differently.
- A profile with an input scale of zero (integers only) cannot be declared; it is rejected as incomplete.
- Malformed text longer than the profile's longest valid input is reported as an input-bound error rather than a syntax error; it is rejected either way.
- The digit pre-checks are upper bounds: a representable product can be refused when the intermediate bound has no headroom above the result size, and the allocation check charges a zero weight its scaled width.
- The fixed-seed sequences are assumed stable for a given Go version; this was not checked across platforms.

# Independent review

Reviewer in each round: the repository's read-only `independent-reviewer` agent role (AI), 2026-11-12. It read files and executed nothing.

| Round | Result |
| --- | --- |
| Design review of the plan, before implementation | FAIL without blocker: 4 HIGH, 6 MEDIUM, 5 LOW, all resolved as in the plan's resolution table before code was written |
| Change review of the implemented code | PASS without blocker or HIGH; no input was found that yields a wrong amount or a wrong acceptance. Two MEDIUM: this record claimed the change review before it existed, corrected; no allocation test fixed which rounding mode is applied, corrected with the mode cases above and confirmed by a mutation that forces half up. LOW corrected: a remainder case that discriminates, representability asserted on accepted results, a sum that overflows the bound. LOW recorded as limits above. No second round ran on these test and record corrections |
