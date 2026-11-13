---
type: ImplementationPlan
title: On-premises cost pool versions with components, an explicit allocation basis and a conserving approval
description: Defines the first model of work item W-12, a registry of versions of an on-premises cost pool in which asset inputs keep a purchase price apart from the depreciation or lease that is summed, components state what they are included in, power components carry the source, period and grade of their model, an allocation basis names every bearer and idle capacity explicitly, and an approval sums the pool exactly, allocates it and checks conservation.
status: draft
---

# Scope and ownership

[HB-198](https://linear.app/hybrid-billing/issue/HB-198/w-12-on-premises-cost-pool-versions-with-components-an-explicit) is the first increment of work item W-12 (requirement HB-12, a conditional requirement that applies when an on-premises internal cost pool is used). It is the eighteenth increment built under the owner's instruction of 2026-11-12. Base: main `f5cf009`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the model sums cost amounts and decides what each organisation bears. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-13:

- Requirement HB-12: an internal cost is made from depreciation or lease, power, facility, licence and operating cost that the customer approved. Internal cost, what an organisation bears and an external sale are separate, and a purchase price and its depreciation are not both added. Proposed direction: CostPoolVersion, AssetCostInput, CostComponent, AllocationBasis, with inclusion relations, period, owner and approval stated; PUE and a power model have a source, a period and a measurement grade. Scale: aggregate per asset and period first, check conservation per shared pool, then allocate; never create every asset times every organisation.
- Acceptance of HB-12: a synthetic pool of 1,200,000 won becomes 720,000, 240,000 and 240,000 and is conserved; a separate sale of 1,000,000 won does not have the internal allocation added to it a second time. Linked source cases DC01 (a missing measurement is not treated as zero), COM06 (cost to allocate = what customers or organisations bear + what the MSP bears + what is explicitly unattributed) and SIM06 (the pool total is conserved) are NOT_RUN.
- Open decisions of HB-12, not made: how a pool is split, who bears idle capacity, the customer's approval policy.
- AR-10: rack, space and line flat fees, reserved GPU base fees, usage surcharges, operating services and SLA credits are independent components; power or a service included in a flat fee is not applied again as a usage charge. Its SLA part belongs to other work items. AUD-AR-10 is NOT_RUN.
- Existing code: `modules/allocation` has `LargestRemainderV1`, which quantizes a total, distributes it over weighted targets, breaks ties by target identifier and verifies conservation itself. `packages/primitives/decimal` parses plain decimal texts within a profile and adds exactly. `modules/hawkeye` and `modules/tokenmeter` have the patterns reused here: an immutable header, write-once parts, a verdict with ordered reasons, exclusion of overlapping periods, `Counting` with a sequence number.
- No inventory of assets, no depreciation schedule, no PUE source and no approval record format is in the planning files read.

# Decision

One Go package, `modules/costpool`, standard library, the decimal primitive and `modules/allocation` only, with an in-memory registry. It reads no inventory and no meter and has no entry point wired in this increment: a caller reports a version of a pool, its assets, components and basis, and a recorded approval makes the registry compute the pool and its shares once.

What this can and cannot show: that the total of a version with a recorded approval is the exact sum of the components that are summed, each once; that the field for an acquisition price enters no sum; that a component stated as included in a flat fee adds nothing; that the shares add up to the total exactly, with nothing left to rounding; that every share has a bearer of one of three classes; that the weights agree with the capacity the caller stated; and that at any one reading at most one version of a pool is called `current` for a moment of time, which is a stored invariant here. It cannot know that an amount, a depreciation, a PUE, a weight, a capacity or an approval is true, that the components reported are all there are, that a purchase was not reported as a component of another kind, that a component not stated as included is in fact not included in another, or that a version was already used.

Names: a recorded approval is called `approval_recorded`, not approved. HB-12 makes the customer's approval the precondition of an internal cost; here any caller states an approver and a reference, the same caller may open, fill and record, and nothing ties the approver to the customer. That the customer approved is not met by this package. The verdict is `current` or `not_current`: it says which version of a pool stands for a period, is necessary for using its shares and not sufficient. A power grade is the grade the caller reported.

Not violated here and open for later increments: the package has no input and no output for a sale, so no figure of it adds an internal share to a sale amount; the acceptance sentence about a separate sale of 1,000,000 won is neither met nor testable inside this package. The package computes no rate and no unit price: its own arithmetic is addition and comparison of the decimal primitive and the call of the existing allocation, which a test checks by scanning the source.

Relation to AR-10: AR-10 concerns products sold and power billed again to a customer. `IncludedIn` between cost components is an analogue on the cost side and does not address it; AUD-AR-10 is untouched. Of SIM06 only the conservation of the pool total is addressed at the level of this package, not units of usage, assignment or power; of DC01 only that a missing amount is not zero, not attribution per period. The design part of HB-12 (how a pool is split, idle capacity, the approval policy) stays open with the owner. W-12 asks for independent verification by a DBA and by QA; this ticket has one independent reviewer of design and change, and the rest remains.

Alternatives considered. A period conflict judged at reading, as in the sibling packages, under which a later approval silently makes an earlier version not current: not taken; here recording an approval is refused while another current version overlaps, so a version that may be in use is never knocked out by a stranger. Summing acquisition prices for display: not taken, a display sum could block an approval by a bound. A rounding component borne by nobody: not taken, see Approval.

Every caller-supplied string is an identifier under the project's identifier rule (letters, digits, `. _ : -`, at most 128 bytes), a three-letter uppercase currency code, the text of an amount, or a value of a closed list. The package has its own copy of the identifier rule, which a test compares with the allocation's at the edges, so that a target accepted here is not refused there. There is no free text. Owner and approver identifiers denote people or roles; the caller must make them pseudonymous references (stated obligation). Target, evidence, source and reference identifiers are recorded and resolved nowhere. No floating point is used. All times are kept and returned in UTC; periods are half-open, the start included and the end not.

What a workspace is, the MSP's scope or one customer's, is not decided here. The shares of all targets of a version are readable by whoever may call the registry with its workspace; it must not be exposed to a customer-facing read (exposure blocker).

## Amount texts

An amount, a weight, a capacity and a PUE are texts in the form the decimal primitive parses with the registry's profile: digits, an optional point with digits, no exponent, within the profile's input bounds. A text the primitive refuses is malformed, and so is any text that starts with a minus sign, `-0` included. A capacity must be greater than zero; a PUE at least 1. The empty text means not stated where a field allows it and is malformed elsewhere. Texts are kept as written and compared as texts in a repeat, the empty text like any other, so `1.50` against `1.5` is a conflict; sums compare numbers, so weights `60.0`, `20` and `20` meet a capacity `100`. Sums are returned as the primitive prints them.

A quantum is one of `1`, `0.1`, `0.01`, `0.001`, `0.0001`, limited further by the profile's input scale. Whether it fits the currency is not checked.

## Versions

`Open(spec)` creates a version in the state `draft`. A header has: workspace, pool and version number (1 to 999,999), which together are its identity; a currency; a period of whole seconds that runs forward and is not longer than the configured maximum; an owner identifier; a quantum; and `Supersedes`, zero or a lower version number of the same pool. A version number need not follow the last one. The header is immutable. States are `draft`, `approval_recorded` and `withdrawn`; being superseded is a mark beside the state.

A version that names another is rejected unless that one exists in the pool, had an approval recorded at some time, is not superseded, and has the same period and the same currency. A correction that changes the period is made by withdrawing and opening a version that names nothing. Naming changes nothing until the approval is recorded.

## Assets and components

`PutAsset` and `PutComponent` add to a draft. Each is written once under its identifier: an identical repeat writes nothing and succeeds in every state; a differing one is a conflict; a new one for a version that is not a draft is rejected; there is no deletion. A wrong entry is left behind by withdrawing the version and opening another.

An asset has an identifier, a kind (`server`, `gpu`, `storage`, `network`, `facility`, `other`), a method (`depreciation` or `lease`) and an acquisition price that may be not stated. The price is kept and shown per asset and enters no sum of any kind; only the numbers of assets with and without a stated price are stored.

A component has an identifier, a kind (`depreciation`, `lease`, `power`, `facility`, `licence`, `operations`), an amount that may be not known, an asset that may be none, an evidence identifier, `IncludedIn` that may be none, and for power a model. What the request alone shows is malformed when broken:

- `depreciation` and `lease` require an asset;
- `power` requires a model and no other kind may have one. A model has a grade (`measured`, `estimated`, `not_known`), a source identifier, a PUE and a period of whole seconds that runs forward. For `measured` and `estimated` the source and the PUE are required; for `not_known` they may be not stated, so that nobody has to invent a value;
- a component with `IncludedIn` is of kind `power`, `licence` or `operations` and does not name itself.

What needs the stored version is rejected when broken:

- an asset named must exist in the version, and for `depreciation` or `lease` the kind must be the asset's method, so one asset cannot carry both;
- at most one component of kind `depreciation` or `lease` per asset in a version;
- a model's period lies within the version's period;
- `IncludedIn` names a component that exists already and is of kind `facility` or `lease`. Since those kinds cannot be included themselves, inclusion has one level.

An included component adds nothing to any sum; its amount is kept and shown, and only the numbers of included components with and without a known amount are stored. Stating an inclusion that is not true removes a real cost from the total (not conservative, stated).

## Basis

`SetBasis` is written once per version, to a draft, with the same rule of repeats; the order of targets is not part of a repeat. A basis has an identifier, a measure (`reserved_capacity`, `fixed_agreement`), a capacity and at least one target. A target has an identifier, unique in the basis, a bearer class (`organization`, `msp`, `unattributed`), a weight and whether it is idle capacity. The measure is a label. Weights are the caller's statements; a basis from usage that was measured, where a weight can be not known, is deferred, and a caller who lacks a weight must not set a basis with zero in its place (stated obligation, DC01).

Malformed: weights that do not add up to the capacity exactly, or whose sum the profile cannot hold; an idle target whose bearer is `organization`. The first rule establishes only that weights and stated capacity agree, so that a caller who states the real capacity has to name what is unused as a target; a caller who states the used sum as the capacity has normalised and nothing notices. The second is a provisional default for an undecided question; it is a refusal that can be relaxed later without touching anything stored.

## Approval

`RecordApproval(workspace, pool, version, approval)` takes an approver identifier and a reference identifier. It is rejected unless: the version is a draft; it has at least one summed component; every summed component has a known amount; every asset has the component of its method, a fully depreciated asset reporting `0`; a basis is set; a named superseded version is still not superseded; no other version of the pool that is current has a period overlapping this one's, the named one apart; and the total is a whole multiple of the quantum. A missing amount is never read as zero.

The last condition means nothing is left to rounding: with amounts of at most the profile's input scale a finer quantum always exists, the shares are distributed in quanta by largest remainder, and the identity is total = `organization` + `msp` + `unattributed` with no fourth term (COM06 at the level of this package). The allocation's rounding mode has no effect and is not a header field.

It then computes once and stores, all sums ranging over summed components only: the total; sums per kind, which add up to the total; sums per asset and one for pool-level components, which add up to the total; power sums per reported grade, which add up to the power sum; the numbers named above; a share per target with its bearer class, weight and idle mark, joined by target identifier; sums per bearer class and for idle targets. An entry of a sum exists exactly when a component or target contributes to it, also when it is zero. A total of zero is allowed and gives shares of zero.

After the allocation the registry checks on its own that the shares add up to the total and that the class sums do. While the allocation is correct that check cannot fail; if it does, or the allocation reports that its result does not reconcile, the operation returns a sixth fixed error for a failed self-check, which is a defect and not a refusal. A bound of the profile or of the allocation's intermediates that a sum or the allocation exceeds is a limit error. In each case nothing is written; the exit for a version that cannot be recorded is to withdraw it and open another.

With the record, a named version becomes superseded. That is never undone, also when the superseding version is withdrawn later; the exit is then a version that names nothing, for which the period is free again. Of two drafts naming the same version, the second to record is rejected and can only be withdrawn. Allocation is per pool version over its targets; nothing is created per asset and target.

`Withdraw(workspace, pool, version, reason)` takes any version that is not withdrawn out, superseded ones included, with a reason of a closed list (`entered_in_error`, `operator`). Figures stay readable. A repeated approval after a withdrawal is the identical repeat and returns `withdrawn`.

## Standing

`Standing` returns `current` or `not_current` with reasons in this order: `withdrawn` (alone); `no_approval_recorded`; `superseded`. So among the versions of a pool, the current ones have periods that do not overlap, by what `RecordApproval` refuses. A draft is told nothing about what would refuse its approval here; see Reading.

Every operation that writes raises a sequence number of the pool by one. `Counting(workspace, pool, start, end)` reads the current version whose period is exactly the one given, with the sequence number, also when nothing counts; for a pool that does not exist it is not found with the number zero.

## Reading

`Version` returns header, state, the superseded mark and by which version, the approval, the withdrawal reason, times, the numbers of assets, components and targets, and for a draft what stands against an approval: the number of summed components without an amount, the number of assets without the component of their method, whether a basis is set, and the current versions that overlap, up to the bound of lists with their number. After an approval it returns the stored totals and sums other than per asset and per target. `Assets`, `Components`, `AssetSums` and `Shares` return pages in the order of identifiers; `Versions` returns the versions of a pool in the order of numbers.

## Repeats, error kinds, tenant boundary, time and limits

Every operation first tests for an identical repeat, which writes nothing, succeeds in every state reached since and returns that state. A second approval with another approver or reference and a second withdrawal with another reason are conflicts.

A request is judged in this order: malformed by what the request alone shows, which is invalid; the version it concerns looked up, an unknown one or one of another workspace being not found (for `Open`, a new identity is the normal case); an identifier that exists, a repeat when identical and a conflict otherwise; what a stored state or rule refuses, rejected; no room, a limit error. Errors are fixed values and repeat no string of the request.

Every operation takes the workspace and reaches only that workspace's pools. The registry authorizes nothing and must not be exposed directly. It is built with limits, a valid decimal profile and a clock, all required; its time is the highest reading of the clock. Limits, each greater than zero: pools per workspace, versions per pool, assets, components and targets per version (the last is also the allocation's), the page size, the bound of lists, the longest period, and the digits of an intermediate in the allocation. Versions and pools are counted for good, withdrawn ones included, so an exhausted pool has no exit but another pool identifier; workspaces are not limited (stated). One lock guards the registry; `Counting`, `RecordApproval` and `Version` of a draft scan the versions of the pool while it is held.

## What is not decided here

Provisional defaults, each to be confirmed or changed by the owner: idle capacity is not borne by an organisation; one pool is split by one basis of weights the caller states; an approval is a recorded approver and reference without a check of who or what; a total must be a whole multiple of the quantum; a version with an estimated or unknown power grade can have an approval recorded, the grade being shown in the sums. Not conservative, stated: nothing checks that depreciation over the versions of a pool does not exceed an acquisition price; a purchase or a perpetual licence reported as a component beside its amortisation is summed twice; the same asset identifier may appear in two pools; an included component's amount is not compared with its container; a component whose inclusion is not reported is summed, and one wrongly stated as included is not. Costs to availability: one unknown amount, one asset without its component or one overlapping current version blocks the record. Assumptions: costs are never negative; one currency per version; one depreciation or lease per asset and period; a correction keeps the period.

# Allowed files

`modules/costpool/*` (new); `docs/features/HB-198/*`.

# Verification

- The cases of the acceptance at the level of this package, as literal tests: a pool of 1,200,000 won with weights 60, 20 and 20 gives 720,000, 240,000 and 240,000 (the weights are inferred from the amounts; the source chapter is not in the planning files read); an asset with a purchase price and a depreciation is summed once; power included in a flat facility fee adds nothing; a component whose amount is not known, and an asset without its component, block the record. The item about a separate sale is recorded as not violated and not met, the customer's approval as not met.
- Literal table tests with a test clock: every refusal of a header, an asset, a component, a basis, an approval and a withdrawal, with the order of error kinds and which rules are malformed and which rejected; repeats and conflicts field by field, texts compared as texts, `-0`, the empty text, the order of targets; identifiers of 128 and 129 bytes here and in the allocation; every rule of kinds, methods, inclusion and power models; weights against the capacity, a weight sum beyond the profile, idle targets and their bearers; the figures of a version with every kind and their identities; every quantum, a total that is not a multiple of it, a total of zero, a leftover unit decided by the order of identifiers, a sum at the bound of a small profile and the allocation's intermediate bound as limit errors; an included component without an amount; supersession, a named version superseded meanwhile, another period or currency, permanence and the exit; an overlapping current version refused and allowed again after a withdrawal; periods that touch; each reason of standing, alone; what stands against an approval, in `Version`; `Counting` and the sequence number; pages; the tenant boundary; limits; times in UTC; errors that are fixed values; a scan of the source for floating point, for any call of the primitive other than parsing, addition and comparison, and for comments; no input or output named for a sale.
- Properties with fixed seeds over random histories against a reference model written in the test with `math/big.Rat`, which decides the outcome of every operation, including whether a total is a multiple of the quantum, and computes totals and sums independently. Shares are checked against properties and not a second allocation, the algorithm being owned and tested by `modules/allocation`: they add up to the model's total; each differs from its exact proportion of the total by less than one quantum; a larger weight never has a smaller share; a zero weight has a zero share; equal weights differ by at most one quantum; bearer class, weight and idle mark of each share are the basis's. A second standing is recomputed from what `Version` and `Versions` return. Read from the registry besides: no two current versions of a pool overlap; `Counting` returns a version exactly when it is current and has that period; being superseded is never undone; no version is superseded by two. The test fails unless every reason, the refusal kinds of each operation that the generator can cause, each counted once per version, and a minimum of recorded approvals and supersessions were reached; a limit error and a failed self-check at the record are outside the model. No random number is drawn while ranging over a map.
- Concurrency under the race detector: the same component reported by several goroutines and counted once while another records the approval; two versions that name the same one, recorded at once, of which one succeeds; two overlapping versions recorded at once, of which one succeeds; a withdrawal against readers of standing and of what counts.
- A version with the configured maximum of assets, components and targets has its approval recorded and is read.
- The package's tests run many times in a row before any mutation run; a mutation check by a script in Python with results appended to files that are never overwritten, kept outside the repository. The registry's own check of the shares cannot fail while the allocation is correct, so mutants of it are expected to be equivalent and are reported as such.
- `gofmt -l`, `go vet ./...`, `go test -race -count=1 ./...`.

Remaining NOT_RUN at system level after this ticket: DC01, COM06, SIM06, AUD-AR-10. This ticket does not complete W-12; the release gate of HB-12 stays not implemented.

# Deferred

An inventory of assets and depreciation schedules; a PUE and power model fed by `modules/hawkeye`; splitting a pool by measured usage; usage-based bases computed from records; the customer's approval as a document with `modules/evidence`; who may approve and withdraw; passing shares to a billing and keeping a version from being used twice; sales and margins; SLA credits and the product components of AR-10; commands, permissions and screens; persistence.

# Design review resolution

The independent design review of 2026-11-13 (read-only, nothing executed) returned FAIL without a blocker and without asking for a redesign: 3 HIGH, 13 MEDIUM, 9 LOW. It found the structure sound and no path to two overlapping versions that count. All findings were adopted above.

| Finding | Resolution |
| --- | --- |
| F1 a rounding component borne by nobody, of any size and sign | Quanta are a closed list of powers of ten not above 1; a total that is not a whole multiple is refused; no mode; the identity has three terms |
| F2 an unknown read as zero in three places | Every asset needs the component of its method; counts instead of sums for included components and prices; no measured basis in this increment |
| F3 `approved` and `usable` | `approval_recorded`, `RecordApproval`, `current`; the customer's approval recorded as not met; `customer_rejected` removed |
| F4 the sum of prices and the claim about them | No sum of prices; the claim narrowed; the licence case listed |
| F5 the scope of partial sums | Summed components only, with identities and the rule for entries |
| F6 bounds found only at approval | The weight sum is checked at the basis; bounds are limit errors, a failed self-check its own error; the exit stated |
| F7 the claim about idle capacity | Reworded; the design part of HB-12 stated as open |
| F8 supersession | Not superseded, same period and currency, at `Open` and again at the record; the loser of a race and withdrawal of a superseded version stated |
| F9 a conflict that was not symmetric | No conflict at reading: an overlapping current version refuses the record |
| F10 measures that claim a measurement | Cut to two labels |
| F11 the relation to AR-10, SIM06 and DC01 | Stated; AUD-AR-10 untouched |
| F12 a PUE forced for an unknown grade | Not required for `not_known`; the grade is the reported one |
| F13 rules that need stored state judged as malformed | Classified as rejected |
| F14 repeats | Order of targets, `-0`, the empty text and number against text settled |
| F15 opaque refusals of an approval | `Version` of a draft shows what stands against it |
| F16 limits and the lock | Per-asset sums and shares are paged; construction, counting for good and scans stated; identifier parity tested |
| F17, F18 people and the tenant model | Pseudonymous references as an obligation; the exposure blocker stated |
| F19 to F21 the model, missing tests, scans | Adopted in Verification |
| F22 no alternative recorded | Recorded, and the one the review named was taken |
| F23, F25 a redundant rule and `replaced` | Removed |
| F24 independent verification asked by W-12 | Stated as remaining |

# Design re-review resolution

The narrow re-review of 2026-11-13 (read-only, nothing executed) returned PASS with findings, no blocker and no redesign: of the first findings it held nineteen resolved, four partly and one it could not identify; thirteen new findings, six MEDIUM and seven LOW. It confirmed by reading that an exact multiple leaves no rounding component in the allocation, that no path leads to two overlapping current versions and that no state lacks an exit other than the exhausted pool. It said implementation may start after text fixes. The rules below settle them and are part of the decision; where they are more precise than a sentence above, they govern. The change review reads them against the code.

- N1, the multiple and the scan. Values of the primitive are canonical and a quantum is a power of ten, so a total is a whole multiple of the quantum exactly when its scale is not greater than the quantum's, zero included. The source scan over files that are not tests allows of the primitive: the functions `Parse`, `Add` and `Cmp`, the methods `Scale`, `Sign` and `String`, the types `Decimal` and `Profile` with `Valid`, and the one mode constant of N8; anything else of it fails. Test files are exempt.
- N2, the sequence of `RecordApproval`, after malformed, lookup and repeat or conflict: the rejections that need no sum, in this order of reading though all give the same error (not a draft; no summed component; a summed component without an amount; an asset without its component; no basis; the named version superseded; an overlapping current version); the sums, a limit error on a bound of the profile; the multiple test, rejected; the allocation, a limit error on a bound; the self-check, its own error, which only `RecordApproval` returns.
- N3. A period longer than the configured maximum is malformed, for a version and for `Counting`; a model's period is bounded by the version's.
- N4, errors of the allocation and the primitive at the record. `decimal.ErrResultBound`, `decimal.ErrIntermediateBound` and `allocation.ErrIntermediateBound` are the limit error. Every other error, which the rules make unreachable, is the self-check error. Nothing is written in either case and no foreign error value is returned.
- N5, `Version` of a draft also returns: the number of summed components; whether the version it names is still not superseded, when it names one; and whether the total is a multiple of the quantum, as `yes`, `no` or `not_determined`, the last when a summed amount is not known or a sum exceeds a bound. The named version is left out of the overlapping current versions and their number, as at the record.
- N6. A component's amount with more than four digits after the point is malformed, whatever the profile's input scale, so that the finest quantum divides every total. Weights, capacities and prices are bound by the profile only.
- N7. In `SetBasis` the number of targets above the limit is a limit error judged before the weights are added, after the forms of the basis's own fields; the same holds for nothing else.
- N8. The allocation is called with the mode `decimal.HalfEven`, which has no effect on an exact multiple, is not stored and not returned.
- N9. Each reason of standing occurs alone: `withdrawn` hides the others, and a version without a recorded approval cannot be superseded.
- N10. After a superseding version was withdrawn, the period is free for a version that names nothing, and a version may also name the withdrawn one, which had an approval recorded and is not superseded.
- N11. Weights, the capacity, prices and components' amounts are returned as written; sums and shares as the primitive prints them.
- N12. The evidence identifier is required for every component, also one whose amount is not known.
- N13, further costs to availability: a correction that changes the period, and any correction of version 999,999, go by withdrawing and opening, which leaves an interval without a current version. In the Approval section "a named superseded version is still not superseded" means that the version it names is still not superseded.

# Change review resolution

The independent change review of 2026-11-13 (read-only, nothing executed by the reviewer) returned PASS with findings, no blocker and no HIGH: three MEDIUM, all about what the verification claims, and eleven LOW. It found no sequence of operations that gives a wrong total, wrong shares, a write on a failed record, two overlapping current versions or a reach across workspaces, and it held the removed checks and dropped mutants to be sound. All findings were resolved or stated; the corrections were not reviewed again by a separate reviewer.

| Finding | Resolution |
| --- | --- |
| C1 refusals counted per call, two kinds not required | Counted once per version; a basis for a pool that does not exist added and required; the sentence in Verification says what is outside the model |
| C2 the primitive's bound of intermediates at the record had no test | A literal case under a tight profile, with a mutant for it. `decimal.ErrResultBound` from the allocation is unreachable, shares never exceeding the total, and is mapped all the same |
| C3 the source scan claimed more than it checks | N1 is corrected here: the scan allows a fixed list of references to the primitive's package, the two bound errors of N4 among them, refuses a list of method names and a renamed import, and does not look at operators. That the source has no arithmetic on decimals other than addition and comparison rests on reading as well |
| C4 the time of change of a superseded version | Being superseded is a change and sets it |
| C5 a mark that read as superseded when nothing is named | The field says whether the named version is superseded; false when nothing is named |
| C6 a draft without a summed component is told its total is a multiple | Stated: zero is a multiple; the number of summed components is beside it |
| C7 reads the plan did not specify | Stated: `Versions` of a pool that does not exist returns nothing without an error, as the sibling packages do; `Targets` before a basis and `Shares` and `AssetSums` before figures are rejected |
| C8 the audit did not read targets | It reads the basis and its pages of targets against the model |
| C9 what the gated concurrency test shows | Stated in the verification record: the order of the record and its components is fixed in two of three rounds and free in the third, whose outcome is not required |
| C10 assertions weaker than their text | The number zero for a pool that does not exist; every share and their sum at the limits; negative zero for a weight and a PUE; one message corrected |
| C11 any failed addition is a limit error | Stated: by reading, an addition of valid operands fails only on the result bound |
| C12 unused test code and an unused receiver | Removed |
| C13 the self-check error is produced by no test | Listed as not verified |
| C14 the plan against the code | Deviations, recorded here: a read `Targets` returns pages of a basis's targets and `Version` returns the basis without them; three checks were removed as unreachable, a version number below one (refused by the rule on `Supersedes`), a basis without targets (its weights cannot add up to a capacity above zero) and a version overlapping itself (only a draft is recorded and a draft is not current) |

