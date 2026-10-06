---
type: Financial Contract
title: Economic coverage Claims and bounded exact arithmetic
description: Specifies proposed partial-consumption correction and numeric mechanisms with independent synthetic expected cases.
status: draft
---

# Economic coverage and arithmetic

Tracking: [HB-16](https://linear.app/hybrid-billing/issue/HB-16). This is a concrete proposed product contract for D-03 and HB-15 FR-04, independently reviewed as a design. It does not approve a customer price, tax, precision value, software engine or live accounting profile. Apply the [existing financial gates](inputs-pricing-and-erp.md) and [payment/recognition separation](payment-and-recognition.md). Product development remains gated.

## Obligation identity and coverage

Define `obligation_id` from a stable registered tuple: installation/workspace, issuer/responsibility, customer relationship, contract ID, purpose namespace, economic component and occurrence/coverage-universe ID with unit/version. Different parties, units and purposes never share an accidental obligation. The registry enforces the complete tuple's uniqueness; hashing alone is insufficient without collision detection and comparison of the canonical tuple. Monthly recurring fees have distinct occurrence IDs. A correction to one occurrence retains its obligation.

Contract/source revision, run, request key, file transport and database partition are evidence references, never identity-reset dimensions. Source-to-coverage mapping records stable source economic IDs, scope, unit/version, canonical coordinates and replacement/overlap relationships. Corrected totals or finer source decomposition preserve already reserved/consumed coordinates; uncertain mapping blocks affected eligibility instead of generating a new unused universe.

Each application identifies one or more typed disjoint half-open coverage intervals `[start,end)` in a named universe and unit version. Coordinates use the exact numeric profile. Discrete records/fixed fee occurrences are indivisible registered atoms; their identity is not a row's display order. Quantity coordinates require explicit allocation meaning; elapsed time does not imply proportional billability. Partial application of an indivisible atom is invalid. Reject empty, reversed, duplicate or overlapping intervals within a request and incompatible universes/units. Canonical sort order is stable identity then interval start, never arrival order.

## Reservation and application transitions

Immutable events record obligation, scope, application/reservation ID, quantity, money separately, source/policy/evidence revisions, actor, reason, expected revision and idempotency input digest. An authoritative current projection answers overlap/availability; an analytical projection cannot grant a Claim.

| Command / transition | Atomic preconditions | Durable result |
| --- | --- | --- |
| `claim.reserve` eligible → reserved | Current authority, approved fixed amount/coverage, expected obligation revision, eligible uncovered coordinates, complete overlap-domain exclusion | Reservation plus all linked obligations and approved snapshot; all-or-nothing within the declared application boundary |
| `claim.pin_dispatch` reserved → dispatch_pinned | Reservation still current; valid submission/approval; all independent holds checked | Submission intent/outbox and pinned reservation in one authority transaction before external dispatch |
| `claim.consume` reserved/pinned → consumed | Authoritative corresponding economic-effect evidence; matching purpose/scope/application; no conflicting newer disposition | Unique immutable application; identical replay returns that application after current read authorization |
| `claim.release` reserved/pinned/outcome_unknown → released | No dispatch intent/effect, or authoritative no-effect/reconciled cancellation evidence for the pinned operation; current expected revision | Only the named reservation is released after the required no-effect/cancellation proof; confirmed effects require consumption and governed reversal, never release-as-erasure |
| `claim.reverse` consumed → linked reversed coverage | Named original application, approved correction and external reconciliation, exact subset, cumulative reversal not exceeding application | Immutable reversal and coverage disposition: cancelled, superseded, or replacement-pending; not automatically available |
| `claim.authorize_replacement` replacement-pending → replacement-eligible | Explicit approved replacement linkage, reconciled external outcome, defined same economic coverage and purpose | Eligibility only for the linked replacement; ordinary approval/reservation still required |

An obligation may contain both consumed and available intervals; status is per interval/application, not a single boolean for the service. Reversal events do not delete consumption history. Remaining eligible coverage is the eligible universe minus active reservations, pinned/consumed coverage and cancelled/superseded/replacement-pending coverage, with effective reversals interpreted through the approved disposition. Never calculate availability by simply subtracting monetary credits.

Multi-obligation operations acquire their full overlap domains in canonical obligation-ID order and commit all reservations or none for the declared boundary. A row-level compare-and-swap protecting only one application is insufficient. Physical locking/constraints remain implementation choices to prove later; no partition may bypass global overlap protection. Conflict responses identify only authorized affected scope and require reinspection, not silent retargeting.

Worker lease and economic reservation lifetimes are distinct. Expiry, worker cancellation, connection timeout or restart never releases a dispatch-pinned/externally uncertain reservation. Mark `outcome_unknown` with last evidence/as-of and reconcile the same external identity. Automatic pre-dispatch expiry is permitted only when an atomic check proves no dispatch intent and an approved expiry policy applies. Losing a worker lease cannot create a second economic dispatcher; fencing and endpoint uncertainty still apply.

Monetary-only credit, refund, service cancellation, source correction, invoice cancellation and quantity reversal are separate governed purposes. A monetary credit changes money but not consumed quantity. Every reversal subset must lie within the original effective application minus all previously effective reversed coverage, checked atomically under the expected application revision across the entire reversal-overlap domain. A same-intent replay returns the prior reversal; a distinct partly or wholly overlapping reversal conflicts even when the scalar cumulative quantity remains below the original total. Repeated reversal cannot free coverage twice. A newly computed source version cannot reset old Claim coordinates. Customer invoicing and monthly recognition use distinct logical purpose scopes/keys within the existing authority/submission model, not new competing financial authorities, and retain the existing anti-double-recognition linkage; this separation never permits duplicate recognition of the same economic cost.

## Versioned bounded exact-decimal profile

Represent an external decimal as a finite base-10 string with optional minus sign, integer digits and optional fractional digits; reject exponent notation, NaN/Infinity, binary float and implicit locale parsing. Canonical value uses an integer coefficient and nonnegative scale, removes redundant trailing zeroes and normalizes negative zero. Retain original input bytes separately as evidence. Validate submitted scale/magnitude before canonicalization so trimming zeroes cannot bypass declared input restrictions.

Every activated numeric profile must provide: ID/revision/effective scope; input and per-operation/result magnitude/scale bounds; aggregate growth bounds; allowed signs/units/currency conversions; division/rounding operations; currency quantum and rounding stages; residual algorithm/version; overflow/rejection behavior; independent expected-case approval. Missing values block activation. This schema deliberately supplies no universal maximum precision, tax rate, tolerance or currency scale.

Addition, subtraction and multiplication are exact within the approved bounds; unsupported magnitude/scale is an error, not rounding. Division succeeds without rounding only when its exact terminating decimal is within bounds. A recurring or excessive-scale result requires an explicit approved `divide_round(numerator, denominator, result_scale, mode)` operation. Zero denominator rejects. No DB storage scale, language context, cast or aggregate function may silently select a rounding point.

The supported proposed rounding-mode vocabulary is `toward_zero`, `away_from_zero`, `floor`, `ceiling`, `half_up`, `half_even`: nearest ties for half_up go away from zero; half_even selects the even retained digit. Each use supplies scale/mode from the approved policy. Rounding to a currency quantum uses exact integer quotient/remainder against that quantum, not an assumed power-of-ten currency. Unlisted modes reject. Signed values follow field/component-specific approved semantics, including ordinary discounts, credits, adjustments, negative margin/savings and correction/refund where applicable. A nonnegative-only field rejects negatives; no blanket negative prohibition removes a legitimate signed component. Mixed currencies or dimensions require an explicit approved conversion; null is unknown, never zero.

Policy approval fixes stage ordering from the [financial contract](inputs-pricing-and-erp.md), intermediate bounds and explicit rounding. Changing arithmetic, operation ordering, quantum or residual behavior creates a policy revision and invalidates affected draft approvals. No hidden tolerance replaces zero unexplained difference.

## Deterministic whole-pool allocation

Proposed `largest_remainder_v1` applies only when the approved allocation profile selects it. Quantize the approved total using its explicit quantum/mode; record any rounding component. For a whole-pool absolute integer target T and exact nonnegative weights w, scale weights losslessly to common integers and calculate each share's integer quotient/remainder of `T * w / sum(w)`. Assign remaining quanta by descending exact remainder; ties use immutable canonical allocation-target ID. Restore the total's sign. Individual zero weights receive zero and are allowed when total weight is positive. Reject duplicate target IDs, negative weights, zero total weight and unsupported caps/minima. Other algorithms require separate explicit contracts, not silent fallback.

Store pool scope/digest, all target IDs/weights, denominator, algorithm/policy revision and final per-target result. Filter/page/chunk/worker order never changes the denominator or tie order. A full result manifest verifies signed sum equals the quantized total before publication. Changing a weight, membership or denominator creates a new pool revision and review.

## Bounded formula and analytical contract

The proposed typed AST supports decimal constants; declared quantity/money variables; exact add/subtract/multiply/divide; explicit divide_round/round; dimension-compatible comparisons; conditional selection; min/max and approved full-pool sums. Unit checking rejects illegal additions/comparisons and unapproved dimensional conversions. Tax/discount/tier behavior uses versioned approved policy inputs; operators alone do not define those policies.

Every formula profile fixes allowed variables/operators/functions, input scope, AST depth/node/operation limits, aggregation membership, byte/time/resource budget and deterministic error behavior. Missing budget blocks activation; partial execution never produces an approvable amount. No arbitrary code, network/filesystem call, clock/random value, dynamic evaluation or implicit float conversion. Evidence records formula hash/version, inputs, stages, operations and policy references. Rate/tier/allowance pools remain complete economic scopes.

Proposed analytical mode is authoritative precomputed monetary components with exact checked grouping/sums over one eligible revision/currency/basis. Reject duplicate versions, join fan-out, unsupported casts and aggregate overflow before output reaches approval. Engine/version-specific expressions require separate review. This selects no DB/analytics product and does not certify a driver.

## Independent expected cases

All cases are NOT_RUN product acceptance scenarios. Synthetic constants demonstrate behavior only.

| ID | Given/action | Expected |
| --- | --- | --- |
| CL-01 | Eligible `[0,100)`, consume `[0,60)`; retry under a new run/source revision | Consumed 60, available 40; overlapping retry rejected; `[60,100)` can follow normal approval |
| CL-02 | Two concurrent reservations `[0,60)` and `[40,100)` | At most one commits; no overlap, no partial multi-obligation residue |
| CL-03 | Dispatch `[0,60)`, timeout then worker lease expires | 60 remains pinned/unknown; no automatic release or second dispatch |
| CL-04 | Monetary credit for first application | Consumed quantity unchanged; no new eligible coverage |
| CL-05 | Reverse `[40,60)` from first application twice | Exactly one 20-unit reversal; replacement-pending until explicit approved replacement; no extra 20 from replay |
| CL-06 | Correct source total/decomposition, or cancel service | No coordinate reset; ambiguous overlap blocks; cancellation does not authorize replacement |
| CL-07 | API/file or partition race; same key with changed scope | Global economic protection still holds; changed payload conflicts |
| CL-08 | Original `[0,60)`, reverse `[20,40)` then distinct `[30,50)` | Second reversal conflicts because `[30,40)` overlaps; scalar quantity below 60 is insufficient |
| NM-01 | Exact `0.1+0.2`; `100/(1-0.20)` | `0.3`; `125` |
| NM-02 | `1/3` without round; synthetic 4-place half_up `1/3`, `2/3` | Reject first; `0.3333`, `0.6667` for explicitly rounded cases |
| NM-03 | Overflow, scale excess, float, mixed currency, missing profile | Reject before approval; no truncation/default/partial success |
| NM-04 | Synthetic total `0.05`, quantum `0.01`, equal A/B/C | `0.02/0.02/0.01`; negative total gives sign inverse; permutation/chunking unchanged |
| NM-05 | Filtered pool, duplicated target, zero total weight/negative weights, missing formula budget | Reject/ineligible; never recompute a smaller denominator silently |
| NM-06 | Aggregation fans out a source component or changes eligible revision | Reject or use verified unique authoritative components; no double sum |

Remaining customer values, named finance approval and physical enforcement evidence are activation gates. A reviewed proposal is not a live policy or final planning acceptance.
