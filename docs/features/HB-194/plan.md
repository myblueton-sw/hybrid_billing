---
type: ImplementationPlan
title: HawkEye cost statements with an eligibility verdict and at most one eligible reception per period of a source
description: Defines the first model of work item W-10, a registry of receptions of cost statements from HawkEye in which each carries its period and what it supersedes, its team lines are compared with what it declares, and an eligibility verdict keeps a mock, unmeasured, provisional, incomplete, superseded or conflicting statement out of operational confirmation.
status: draft
---

# Scope and ownership

[HB-194](https://linear.app/hybrid-billing/issue/HB-194/w-10-hawkeye-statements-with-an-eligibility-verdict-and-one-statement) is the first increment of work item W-10 (requirement HB-10, a conditional requirement that applies when the HawkEye connection is opened). It is the sixteenth increment built under the owner's instruction of 2026-11-12. Base: main `f838a60`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the model decides which power cost may enter a confirmed billing and holds amounts per tenant. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-13:

- Requirement HB-10: the existing GPU power allocation and its immutable statements are taken as they are, and the selling charge is computed by the new platform. A statement that could not be measured, is a mock or is incomplete is not used for operational confirmation. Proposed direction: not only the statement identifier but period, supersession, measurement tier, coverage and totals per team are received and eligibility is judged from them; the connection by GET is separate from the POST that creates a statement; a cost is not multiplied by a power unit price again. Scale: the row limits of 500 and 2000 named by the source are a risk until verified again; without a full export, or verified pagination and a manifest, a bulk connection is not opened.
- Acceptance of HB-10: another statement_id in the same month, an unmeasured zero, a partial list and a mock are tested, and the same cost is not reflected twice. Linked source cases COM01, DC01, SIM06 are NOT_RUN; their text is not in the planning files read.
- Open decisions of HB-10, not made: the deployed version, the supersession relation of statements, the export API and authentication. No HawkEye interface, field name, tier name or status name is in the planning files read.
- RT-01: a source event keeps source_id, source_event_id, schema_version, period_start, period_end and source_revision; a cost adds amount, currency, cost_origin and cost_status; a correction names what it supersedes with a reason; a wrong correction reference is refused; the tenant of the connection is compared. AUD-RT-01 is NOT_RUN. Its other parts (resource and meter identifiers, quantities, the return of approval results, locks) concern other work.
- AR-02 (mutual authentication of an internal connector, push verification) and AR-10 (separate data-centre products, SLA credits) name HB-10 as well; neither is touched by this increment.
- Existing code: `modules/connectors` (HB-184) holds capabilities and a collection ledger; `modules/intake` (HB-186) holds artifacts and input manifests; `modules/reconcile` (HB-192) holds an `internal_cost` basis that a power cost would use. Nothing knows a statement.

# Decision

One Go package, `modules/hawkeye`, standard library and the decimal primitive only, with an in-memory registry. It calls no HawkEye and has no entry point wired in this increment: an adapter that fetched a statement reports it, and the registry decides whether it may be used.

What this can and cannot show: that a reception of a statement is called eligible only when every listed condition holds, and that at any one reading at most one reception of a source is eligible for a moment of time. It cannot know that a report is what HawkEye said, what HawkEye's fields mean, or that a statement was already used: what counts for a period can change after a consumer read it, and nothing here keeps the same cost from being reflected twice in a billing. `eligible` is necessary for using a power cost and not sufficient: whether the power is already inside a flat fee (AR-10), whether the connection belongs to the tenant (AR-02) and whether an estimated tier may ever be used are decided elsewhere (stated limits).

Three sentences of the requirement are not violated here and remain open for later increments: the package has no operation that asks HawkEye to create a statement, so the separation of reading from creating is the adapter's to keep; it has no unit price and derives no cost from energy, which a test checks by scanning the source for multiplication and division of the decimal primitive, but it keeps a line's energy as a record, and a later pricing step could still price it again; and it computes no selling charge.

Alternatives considered: the highest-revision rule of `modules/reconcile`, not taken because nothing says that HawkEye's revisions are comparable across statement identifiers; and receiving statements as document versions of `modules/intake`, not taken because eligibility here rests on header facts, tier, coverage and supersession, that an input manifest does not carry.

Every caller-supplied string is an identifier under the project's identifier rule, of which the package has its own copy, a three-letter uppercase currency code, a decimal that the profile given at construction accepts as an input, or a value of a closed list. An identifier of HawkEye that does not fit the rule, a team name in Korean for instance, has to be encoded by the adapter, always in the same way; two spellings of one source, statement or team are two different ones here, and one installation reported under two source identifiers escapes every rule below (stated obligation). All arithmetic is the package's own, exact, on integers of any size, and no floating point is used; sums are printed in the primitive's canonical form whether or not the profile holds them; a reception's own figures are kept and returned in canonical form. All times are kept and returned in UTC.

## Statements and receptions

A statement is what HawkEye issued: it is identified in a workspace by its source, an identifier of the HawkEye installation, and its statement identifier there. A reception is one report of a statement by an adapter, with an identifier the caller chooses, unique in the workspace. At most one reception that is not voided exists per statement; opening a second is a conflict, and a voided reception frees the statement for another. So a list that was cut off, a header reported wrongly in anything but its supersession reference, a reception voided by mistake and a statement that HawkEye reissued under the same identifier all have the same exit: void the reception and report the statement again under a new reception identifier. A wrong supersession reference has no such exit (see the re-review resolution).

`Open(reception)` records its header, immutable:

- reception identifier; source, statement identifier, and the origin of the cost as HawkEye names it, identifiers;
- the schema version and the connection it was fetched through, identifiers, recorded as they were at the first report and not part of what a repeat is compared with, so that the same reception reported again through a replaced connector is a repeat;
- its period, a start and an end in whole seconds, the start before the end, the length not above the maximum given at construction. A period is the time from its start up to and not including its end: two periods that only touch do not overlap;
- its source revision, a whole number from 1 to 999999, recorded and not compared with any other;
- optionally the statement identifier of the same source it supersedes, which is not its own, with a correction reason, an identifier, both or neither;
- whether it is a mock. A mock names no supersession;
- its cost status, `final` or `provisional`, and its measurement tier, `measured`, `estimated` or `unavailable` (assumed names);
- its coverage, in hundredths of a percent from 0 to 10000;
- what it declares about its content: the number of team lines, at least 1 and at most the limit; the total cost per currency, at least one currency and at most the limit of currencies, each not negative; and where that declaration comes from, `source_header` when HawkEye stated count and totals apart from the list of lines, or `derived` when the adapter computed them from the lines it received.

A supersession reference is right when a reception of the named statement was ever opened in the workspace under the same source, in whatever state, voided included; when the named statement is not already superseded by another statement; and when it would not close a circle of statements superseding each other. Otherwise the reception is rejected with nothing written; a predecessor that was never fetched has to be fetched first, and when the source no longer has it there is no exit (RT-01: a wrong correction reference; stated limit). The period of the named statement does not matter: a correction may move a period.

Being superseded is a fact about a statement, set when the reception that names it is opened, and never undone: it holds for every reception of that statement, present and later, also after the superseding reception is voided. A mock supersedes nothing.

`Post(reception, line)` reports one team line of an `open` reception whose statement is not superseded: a team identifier unique in the reception; its cost, not negative, with a currency; whether it was measured; and optionally its energy in kilowatt-hours, a decimal that is not negative, kept as a record. A line in a currency the header does not declare is accepted and shows later as a difference of totals. A line reported again with identical content writes nothing; with other content it is a conflict; a line beyond the declared number is rejected.

`Close(reception)` is accepted for an `open` reception whose statement is not superseded, at any time, also with fewer lines than declared, because a list cut off by a row limit has to be recorded as what it is. It stores the number of lines and the exact sum per currency, and makes the reception `closed`. `Void(reception, reason)` ends an `open` or `closed` reception with a reason from a closed list (`withdrawn_at_source`, `reissued_at_source`, `wrong_report`, `operator`). States: `open`; `closed`; final: `voided`. A reception remembers whether it was closed before it was voided. For a reception whose statement is superseded, a new line and a first `Close` are rejected, repeats succeed and `Void` is accepted.

## Eligibility

`Eligibility(reception)` reads, without writing, `eligible` or `not_eligible` with every reason that applies, in this order. One rule comes first: a voided reception has the single reason `voided`.

- `not_closed`: the reception is still open; the four reasons about its lines at the end of this list are not given for it;
- `mock`: it is a mock;
- `provisional`: its cost status is not `final`;
- `not_measured`: its tier is not `measured`;
- `coverage_below_required`: its coverage is below the required coverage given at construction;
- `declaration_derived`: its count and totals were computed from the lines received, so that comparing the lines with them shows nothing;
- `superseded`: its statement is superseded;
- `period_conflict`: another reception of the same source that is not voided, not a mock and whose statement is not superseded has a period that overlaps this one's, whatever else is true of either, open ones and ones that can never be eligible included. The receptions it conflicts with are returned by identifier, in bytewise order up to the bound of lists, with their number. Two statement identifiers for one month are therefore both not eligible until one supersedes the other or one reception is voided. A mock conflicts with nothing, since it is test data and can never be eligible;
- `unmeasured_lines`: a line is flagged as not measured, whatever its cost;
- `incomplete_list`: the number of lines is not the declared number;
- `totals_differ`: the currencies that have a line are not exactly the declared ones, or a sum differs as a number. A declared currency without a line differs also when its declared total is zero;
- `sum_outside_profile`: a sum of its lines lies outside the profile's result bound.

`incomplete_list` and `totals_differ` show consistency with the declaration in the header. That the declaration is HawkEye's own and not a count of what a truncated list returned is the adapter's statement in the header (`source_header`); nothing here can check it.

Every write to a statement of a source raises a sequence number of that source in the workspace. `Counting(workspace, source, start, end)` reads the eligible reception whose period is exactly the one given, with its statement identifier and the source's sequence number, and is not found when there is none; a consumer that kept the number can see that something changed since. It does not say why nothing counts; `Eligibility` and `Receptions` do.

## Reading

`Reception(workspace, reception)` reads a reception in every state: its header, state, whether it was closed, the number of lines, the sums per currency once closed, by which statement its statement is superseded, the reason of a void, and the times it was opened and last changed, the latter moving when it is closed and when it is voided. `Lines(reception, after, limit)` reads the lines of a reception that is not open, in the bytewise order of team identifiers, in pages, `after` being the number already read; for an open reception it is rejected. `Receptions(workspace, source, after, limit)` lists the receptions of a source in the bytewise order of their identifiers, each with its statement identifier, period and state, in pages, `after` being the last identifier read, so that a page does not shift when receptions are added.

## Repeats, error kinds, tenant boundary, time and limits

Every operation is its own key and first tests for an identical repeat, before state and limits: a repeat with the same content writes nothing and succeeds in every state reached since, and returns that state, which the caller must read. Amounts are compared as numbers. `Close` of a reception that was closed, also one voided afterwards, is such a repeat; `Close` of a reception voided while open is rejected.

A request is judged in this order: malformed, which is invalid and includes a value outside its closed list, a coverage or revision outside its range, a period that does not run forward, is not in whole seconds or is longer than the maximum, a supersession without a reason or a reason without one, a statement that names itself, a mock with a supersession, and a declared number of lines of zero; a declared number of lines or of currencies above its limit, a limit error; an identifier that exists, which is a repeat when identical and a conflict otherwise, with a second reception that is not voided for one statement and a second `Void` with another reason being conflicts; an unknown reception, not found, and so is one of another workspace; what a stored state or rule refuses, rejected: a supersession reference that is not right, an unknown named statement included, whether it is unknown here or exists in another workspace; a new line or a first `Close` for a reception that is not open or whose statement is superseded; a line beyond the declared number; reading the lines of an open reception; no room, a limit error. A page limit above the page size is invalid; a position beyond the end gives an empty page. No error repeats a string of the request.

Every operation takes the workspace and reaches only that workspace's receptions; the same identifiers in two workspaces are unrelated. The connection is recorded and nothing here checks that it belongs to the workspace, authenticated, or is the one a source is fetched through (AR-02; stated limit, an exposure blocker). The registry authorizes nothing and must not be exposed directly.

The clock is injected, read once per operation, and the registry keeps the highest reading it has seen. Required at construction, without defaults: receptions per workspace, lines per reception, currencies per reception, the page size, the bound of lists and the longest period, all positive; the required coverage, from 0 to 10000; and the decimal profile. Receptions are counted for good, voided ones included, so every correction uses a further one; workspaces are created by first use and not limited in number; one lock serves every workspace; the check for overlapping periods reads every reception of a source in the workspace while the lock is held; none has an exit here (stated limits, exposure blockers).

## What is not decided here

The deployed version of HawkEye, how its statements supersede one another, its export API and its authentication: these are the decisions the design part of W-10 asks for, they remain the owner's, and the design part stays open. Also: the required coverage; whether an `estimated` tier may ever be used; whether a statement with unmeasured teams may be used for the measured ones; how an eligible power cost reaches a billing, through the `internal_cost` basis of `modules/reconcile` or otherwise, and what keeps it from being reflected twice there; who may void.

Assumptions made here: the names `final`, `provisional`, `measured`, `estimated`, `unavailable`; that a statement supersedes at most one other statement; that coverage is one figure per statement; that a cost of a team is never negative, so that a credit of the source cannot be recorded; that energy is in kilowatt-hours.

Not the conservative choices, for the owner: `Close` accepts a reception with fewer lines than declared, which is then never eligible, rather than refusing it; the required coverage may be set to zero, which switches that check off; whether a line was measured, and whether the declaration is HawkEye's own, are the adapter's flags, so an unmeasured team that HawkEye reports as a plain zero passes when the adapter does not flag it; a void works as a choice between two conflicting receptions, also in favour of the one with the lower revision.

Conservative choices that cost availability: a superseded statement never counts again, also when the statement that superseded it was reported wrongly, is still open or was voided, in which case nothing counts for its period until a further statement is received; any overlap of periods makes every reception concerned not eligible, also when the other one can never be eligible itself.

# Allowed files

`modules/hawkeye/*` (new); `docs/features/HB-194/*`.

# Verification

- The cases of the acceptance, as literal tests at the level of this package: another statement identifier in the same month, with and without a supersession; a team flagged as unmeasured and reported as zero; a list cut off at 500 and at 2000 of more lines, with a declaration from the source and, as the stated limit, with a declaration derived from the lines; a mock. The sentence "the same cost is not reflected twice" is not shown by this package and is recorded as not met here.
- Literal table tests with a test clock: every refusal of a header; identical and differing headers field by field, and the same header through another connection and schema version; one reception per statement, a voided one freeing it, and each of the four exits (a cut list reported again, a wrong header, a void by mistake, a statement reissued under its identifier); every supersession reference that is not right (unknown, another source, another workspace, itself, already superseded, a circle) and the right ones, among them a named statement that is open, closed and voided, one of another period, a chain of three, and a predecessor fetched late; a superseded statement staying so after its successor is voided, for a later reception of it too, and the third statement that is the exit; what a superseded reception accepts and refuses; a mock that neither supersedes nor conflicts; lines with every refusal, repeats and conflicts, in an undeclared currency, with and without energy; closing with fewer lines than declared, with as many, and with none, more being impossible to reach; every reason of eligibility alone and all together in order, `not_closed` suppressing exactly the four reasons about lines; coverage at, below and above the requirement, and a requirement of zero; sums at mixed scales, a declared currency without a line and of zero, a sum outside the profile under a small profile; periods that touch, nest and coincide, at the longest length and beyond, not in whole seconds, across sources and workspaces; a conflict with an open reception and with one that can never be eligible, removed by a void, by a supersession, and by voiding the newer of two; the conflicting receptions listed up to the bound with their number; `Counting` for an exact period, for other bounds, after a void, and its sequence number moving with every write; `Void` from each state and with every reason; repeats of every operation after every state; pages of lines and of receptions, the latter while receptions are added; the tenant boundary; limits; errors that are fixed values; a scan of the source for floating point and for multiplication and division of the decimal primitive.
- Properties with a fixed seed over random histories against a reference model written in the test with 64-bit integers, whose generator draws sources and periods from a small set so that receptions collide, bounds its amounts so that no sum can overflow, and constructs eligible receptions and near misses of each reason; the test fails unless it produced enough conflicts, conflicts removed by a void, supersessions and supersessions followed by a void. Read from the registry after every step, independent of the model: no two eligible receptions of one source overlap; for every reception the reasons recomputed by the test from `Reception`, the pages of `Lines` and the list of `Receptions` equal those of `Eligibility`, in order; `Counting` of a period returns a reception exactly when it is eligible and has that period; being superseded is never undone; a voided reception is never eligible; at most one reception that is not voided exists per statement. No random number is drawn while ranging over a map.
- Concurrency under the race detector: the same lines reported by several goroutines and counted once; two receptions of one statement, and two statements superseding one, of which one succeeds each; closing against reporting; a void and a superseding reception against reads of eligibility and of what counts.
- The package's tests run many times in a row before any mutation run; a mutation check by a script in Python with results appended to files that are never overwritten.
- `gofmt -l`, `go vet ./...`, `go test -race -count=1 ./...`.

Remaining NOT_RUN at system level after this ticket: COM01, DC01, SIM06, AUD-AR-02, AUD-AR-10, AUD-RT-01. This ticket does not complete W-10, and the HawkEye connection is not opened.

# Deferred

An adapter and any call to HawkEye, with the separation of reading a statement from creating one; the selling charge and anything that prices energy; keeping a consumer from using a cost twice; the resource, meter and quantity fields of RT-01, the return of approval results and the locks; authentication and tenant binding of the connection (AR-02); verified pagination and a manifest for a bulk export, with `modules/connectors`; the products and SLA credits of AR-10; passing an eligible cost on to a billing; commands, permissions and screens; persistence.

# Design review resolution

The independent design review of 2026-11-13 (read-only, nothing executed) returned FAIL without a blocker and without asking for a redesign: 4 HIGH, 14 MEDIUM, 6 LOW. It judged the single predicate of eligibility, the exclusion of overlapping periods and a supersession that does not depend on the later statement's state sound, and found no path by which a flagged statement reads eligible. All findings were adopted above.

| Finding | Resolution |
| --- | --- |
| F1 no exit once a source identifier is used | A reception has the caller's identifier; one reception that is not voided per statement; being superseded belongs to the statement |
| F2 honest corrections refused | A reference is right when the named statement was ever received, is not superseded by another and closes no circle; period, state and revision of the named one do not matter |
| F3 completeness as a check | The header says where the declaration comes from; a derived one is a reason; the claim reworded |
| F4 one statement counting only at one instant | Claims narrowed; a sequence number with `Counting`; the acceptance sentence recorded as not met here; a void as a choice listed |
| F5 a statement that can never be eligible knocking out a good one | A mock neither supersedes nor conflicts; the other cases named with their exit |
| F6 a superseded statement | New lines and a first close rejected, repeats and void accepted |
| F7 the source as a spelling | Obligations stated; the title narrowed |
| F8 periods | Half-open, whole seconds, a longest length |
| F9 requirements met by absence | Reclassified as not violated and deferred; a scan for multiplication and division |
| F10 fetch provenance in the header | Connection and schema version recorded, not compared |
| F11 kinds of a wrong reference; the correction reason | One kind, rejected; an identifier |
| F12 flags and a coverage of zero | Listed as not the conservative choices |
| F13 undeclared currencies | Accepted and shown as a difference; a limit of currencies; zero totals and negative costs stated |
| F14, F15 finding the exit; pages | Conflicting receptions returned; `Receptions` paged by the last identifier |
| F16 fields of RT-01 | The origin of the cost recorded; energy in kilowatt-hours; the rest listed as deferred |
| F17, F18 tests | Adopted in the verification |
| F19 to F24 | The suppressed reasons, the times, alternatives, `eligible` as necessary only, revisions not compared, wording |

# Design re-review resolution

The narrow re-review of 2026-11-13 (read-only, nothing executed) returned PASS with findings, no blocker and no HIGH: nineteen of the twenty-four first findings resolved and five partly; twelve new findings, six MEDIUM and six LOW, none of which yields a wrong `eligible`. It said implementation may start after text fixes. The rules below settle them and are part of the decision; where they are more precise than a sentence above, they govern. The change review reads them against the code.

- N1, no exit for a wrong supersession reference. A reception that names the wrong statement supersedes it for good; voiding the reception and reporting again does not undo it, and nothing counts for the wrongly superseded statement's period until HawkEye issues a further statement. The same holds for a statement reissued under an identifier that is already superseded. There is no operator exit in this increment (stated limit, an exposure blocker).
- N2, what a statement supersedes is a fact about the statement too, set by its first reception that names a supersession and never changed. A later reception of that statement must name the same statement again, which writes no new fact; one that names another statement, names none, or is a mock is rejected. A statement that supersedes nothing yet may be received again with or without a reference. So a statement supersedes at most one other.
- N3. A reception of a statement that is superseded may be opened, and may name a supersession under the same rules, since a predecessor can be fetched late. The circle check walks the facts of statements, whatever the state of the receptions that set them.
- N4. `period_conflict` is given only to a reception that is itself not voided, not a mock and whose statement is not superseded, and counts only others of the same kind, so the relation is symmetric.
- N5. A mock occupies its statement like any reception. When `Open` is a conflict because the statement has a reception that is not voided, the identifier of that reception is returned with the error. The rows of `Receptions` show whether a reception is a mock and whether its statement is superseded.
- N6, the sequence number of a source. Each operation that writes, a first `Open`, a new line, a first `Close`, a first `Void`, raises it by exactly one; repeats and refusals do not. `Counting` returns it also when nothing counts. It starts again with the process, so a kept number is not comparable across restarts (stated limit).
- N7. A `Counting` whose period does not run forward, is not in whole seconds or is longer than the maximum is invalid. For a source nothing was ever received from, `Counting` is not found with the number zero and `Receptions` gives an empty page; reads create nothing. `after` in `Receptions` is an exclusive lower bound that need not exist; empty means from the start.
- N8. Two receptions that conflict and are both live can only come to a supersession by voiding one and reporting its statement again with the reference, which uses a further reception and, by N1, cannot be taken back.
- N9. The declared totals are a set of currencies, so none occurs twice. A line that would raise the number of distinct currencies of a reception, declared ones and those of its lines together, above the limit of currencies is rejected.
- N10. The lines of a reception whose statement was superseded while it was open cannot be read until it is voided; this is intended.
- N11, further consequences that cost availability: if HawkEye states no count and totals apart from the list, every reception is `derived` and none is ever eligible; a statement without any team line cannot be reported; a statement may name one whose only receptions are mocks.
- N12, tests added: an identical `Open` of a voided reception after another reception took its statement; the second reception of a superseding statement naming the same, another, none, and as a mock; a reception of a superseded statement that names a supersession; a mock occupying a statement and the occupant returned; `period_conflict` not given to a superseded or mock reception; the sequence number not moving on repeats and refusals and returned when nothing counts. The property test also requires a minimum number of eligible receptions, of `Counting` hits and of statements received again after a void. The reference model is compared with the outcome of every operation and with the reasons of every reception after every step; the invariants are read from the registry besides.

# Change review resolution

The independent change review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker or HIGH, 3 MEDIUM and 9 LOW. It found no path by which a voided, open, mock, provisional, unmeasured, derived, superseded or incomplete reception reads eligible, nor two eligible receptions of one source that overlap, and confirmed by reading the order of checks, the facts of supersession, the occupation of a statement, the comparison of totals and the tenant boundary. The three MEDIUM findings were about tests.

| Finding | Resolution |
| --- | --- |
| 1 MEDIUM the model computed no reasons | The model keeps the lines it was given and computes a verdict of its own, which is compared with `Eligibility` beside the verdict recomputed from the registry's reads |
| 2 MEDIUM no close raced a report; an assertion that could not fail | A close runs while lines are reported and the lines kept must be exactly the accepted ones; a first close races a superseding reception and must agree with what the reception then shows |
| 3 MEDIUM the model of supersession mirrors the code | Read from the registry after every step: no statement supersedes two or is superseded by two, what superseded a statement is the statement that names it, and no walk returns to its start |
| 4 a period checked on times that may carry a monotonic reading | Stripped before the check |
| 5 the circle of two not tested by itself | A test |
| 6 the place of two reasons in the order | Literal cases with `superseded` and `period_conflict` beside reasons about lines |
| 7 thresholds per reason | Each reason the generator can produce must be seen at least twenty times, and conflicts removed by a void are counted apart; the kinds of a rejected reception are not counted apart |
| 8 a weak case | It names a statement of the same source that nothing superseded |
| 9 cases of the plan missing | Added: the further statement that is the exit, a named statement whose reception was voided, repeats of a line and of a close on a superseded statement, a repeated open of a closed reception, a second reception while the first is closed |
| 10 the expectation while open derived by a filter | Kept; it states the rule |
| 11 a sign parameter and a negative branch never used | Kept as in the sibling package |
| 12 scans under the lock | Stated with the limits |

Stated with this review, as the code behaves:

- The correction reason belongs to a reception: a later reception of a superseding statement may give another reason than the one that set the fact.
- `sum_outside_profile` never appears without `totals_differ`, because a declared total is inside the input bound, which is never wider than the result bound.
- A reception closed without a line has sums that are empty; one voided while open has none.
- `Counting` answers not found both when nothing was ever received from a source and when nothing counts; the sequence number, zero in the first case, tells them apart.
- An identifier has at most 128 bytes.
- `Counting` judges every reception of the exact period and `Receptions` sorts the receptions of a source at each call, both while the single lock is held.
- The removal of a monotonic reading before a period is checked is kept as a defence; no test distinguishes it.
