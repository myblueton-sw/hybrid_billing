---
type: ImplementationPlan
title: Mapping revisions and normalization runs with separate verdicts for format conformance and billing readiness
description: Defines the first model of work item W-07, mapping revisions from source fields to a declared target schema and normalization runs that check rows in repeatable chunks, keep null, not provided and zero apart, keep the cost kinds apart, preserve source decimals exactly and judge format conformance and billing readiness separately.
status: draft
---

# Scope and ownership

[HB-188](https://linear.app/hybrid-billing/issue/HB-188/w-07-mapping-revisions-and-normalization-runs-with-separate) is the first increment of work item W-07 (requirement HB-07). It is the thirteenth increment built under the owner's instruction of 2026-11-12. Base: main `e858f6b`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the model decides which cost amounts are called ready for billing and holds rows per tenant. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-13:

- Requirement HB-07: the received FOCUS version and the internal target version are separate. BilledCost, EffectiveCost, ListCost and ContractedCost and sale amounts are not mixed. Null, not provided and zero are distinguished. Proposed direction: ColumnMapping and MappingRevision manage source field, transformation, target field and reason for absence; a FOCUS validator or converter is a candidate to be verified at a fixed version and support of every dataset is not assumed; SaleCharge is a separate model. Scale: format checks per chunk are separate from cross checks over the whole run (duplicates, period, totals); error details are paged; the counts of normal, error and quarantined rows add up to the input.
- Acceptance of HB-07: partial support is not shown as full conformance; format conformance and billing readiness are judged separately; source decimals are preserved without passing through floating point. Linked source cases FD04, FD05, FD06 and SIM01 are named in the requirement; their text is not in the planning files read and they are NOT_RUN.
- Open decisions of HB-07, not made: the internal target FOCUS version and dataset, the mapping scope per source, the precision limits.
- Detail contract QA-07 (separate setup, export-writer and collector-reader permissions per provider, AUD-QA-07 NOT_RUN) concerns connection profiles and is not touched by this increment.
- Existing code: `modules/intake` (HB-186) publishes input manifests of row keys, digests and amounts and holds no other field of a row. `packages/primitives/decimal` gives exact decimal arithmetic under a profile. The Python modules under `modules/focus_import` hold a working local import path with its own column handling. Everything is in memory or local files.

# Decision

One Go package, `modules/normalize`, standard library and the decimal primitive only, with an in-memory registry. It parses no file, knows no FOCUS column by name and has no entry point wired in this increment. The target schema is an input, so that no target version is chosen here; a schema that copies a FOCUS version is something a caller declares, and nothing in this package makes it conformant to the FOCUS specification. "Format conformance" below means conformance to the declared target as far as the listed checks go, and the received version and the target are identifiers that nothing here verifies (stated limits).

Alternatives considered: extending the Python import path, which has its own column handling but no revisions, no separate verdicts and floats in places that were not audited, and is not the path the Go modules of HB-184 and HB-186 lead to; and a fixed FOCUS schema in code, which would make the owner's open decision here. Neither was taken. The Python path is not changed and not connected (stated).

What this can and cannot show: that every row was classified once, that nothing the mapping does not cover is called complete, that amounts were not altered and that the sums of what is called ready equal what the caller expected. It cannot know that a caller's declaration of what a source field means is true, nor that an amount is right; expected sums taken from the source are what stands against a wrong amount.

## Targets

A target is a named, immutable schema: an identifier and a list of fields. Declaring an identifier again with identical content writes nothing; with other content it is a conflict. A field has:

- a name, an identifier unique in the target;
- a kind: `cost`, `quantity`, `text`, `time` or `currency`;
- for a `cost` field, its cost kind: `billed`, `effective`, `list` or `contracted`;
- whether format conformance requires it, and whether billing readiness requires it. The two flags are independent.

A target that has a `cost` field names exactly one of its `currency` fields as the currency of its costs. A target names either no period or one `time` field as period start and another as period end. One period pair only: a source with a billing period and a charge period needs the owner's choice of which is checked (not decided here).

## Mapping revisions

A mapping is identified by (provider, source version, target) and has numbered revisions, from 1, each immutable. The source version is the received version; the target is the internal one; they never have to be equal.

`DeclareMapping(mapping, expected, entries)`: when the entries equal those of the latest revision, nothing is written and its number is returned, whatever `expected` says; otherwise `expected` must be the number of the latest revision, 0 for none, else it is a conflict; then the next revision is created. So a retry after a lost response is answered and two declarers cannot both append.

A revision has exactly one entry for every field of its target, and none for anything else:

- `from`: a source field name, an identifier; what the caller declares that source field to be, a kind and, for a `cost`, a cost kind from `billed`, `effective`, `list`, `contracted` or `sale`; and a transformation from a closed list: `copy`; and for `cost` and `quantity` only, `negate` and `scale` by a power of ten from -12 to 12;
- or `missing`, with a reason from a closed list: `not_provided_by_source`, `not_supported`, `not_applicable`.

One predicate decides whether a `from` entry is accepted: the declared kind of the source field equals the kind of the target field and, for a `cost`, the declared cost kind equals the target's. A source field has one declared kind and cost kind in a revision. It follows that an amount declared as a sale amount can be mapped nowhere, since no target field has that kind (SaleCharge is a separate model, deferred), that a quantity cannot feed a cost, and that one source field cannot feed two cost kinds. The declaration is the caller's and is not verified; it is bound into the immutable revision, where it can be audited (stated limit: a caller that declares a sale amount as a billed cost is not caught here).

Also refused: `not_applicable` for a field that format conformance or billing readiness requires; a revision that maps a `cost` field and not the currency of costs; a revision that maps one period field and not the other.

Every revision states, computed and never supplied:

- `complete` when no entry is `missing` except with the reason `not_applicable`; otherwise `partial`, with every other `missing` field listed. So absence is accounted once, at the revision, and a field that the source cannot deliver makes the mapping partial whether or not the target calls it required;
- separately, whether billing readiness is possible: the target has at least one `cost` field that billing readiness requires, and no field that billing readiness requires is `missing`; otherwise not, with the reasons.

## Rows and cells

An input row has a row key and cells by source field name. The row key is the key of a record in an input manifest of `modules/intake`, three fields compared field by field: the key kind (`event` or `locator`), the document identifier (empty for an event key) and the row key. A cell is either null or a text of bounded length. A source field without a cell in the row is not provided. So the three are distinct on input, and they stay distinct on output: a normalized cell is `value` with a canonical text, `null`, `not_provided`, or `unmapped` when its target field is `missing` in the revision. Zero is a value. `negate` and `scale` leave null and not provided as they are.

Cells of source fields that the revision does not use are ignored for the row; their names are collected for the run, and a closed run reports their number and the first of them in byte order up to a bound, so that the mapping scope is visible.

`text` cells are free text held per tenant, bounded in length. Prompt and response bodies must not be mapped (AR-05); the registry cannot tell and the obligation is the adapter's (stated limit).

## Runs

`Open(run)` creates a run in a workspace: an identifier unique in the workspace; the mapping and its revision number, which must be the latest revision at that moment; the input it normalizes, a source scope, period and manifest number of `modules/intake` under the mapping's provider, mandatory, recorded and not checked against intake; the number of chunks, at least 1; the number of rows, at least the number of chunks and at most chunks times the rows allowed in one; optionally a window [from, to) for the period; optionally expected sums, per `cost` field a sum for each currency. Refused: a window when the revision maps no period; expected sums for a field that is unknown, not a `cost`, or `missing` in the revision; an expected sum that the profile does not accept as an input (a total longer than the input bound cannot be expected, and such a run cannot be ready: stated limit). Identical again, expected sums compared as numbers, writes nothing; other content is a conflict. A run is bound to its revision at opening and records the identifier and revision of the decimal profile; a later revision does not change it.

`Check(run, number, rows)` checks one chunk of an `open` run: a number from 1 to the declared count, at least one row and at most the limit. A malformed chunk is refused whole with nothing written: a row key that is not well formed, a source field name that is not an identifier, a text over the length limit. Whether a chunk is the same as one sent before is decided on what was sent, not on its meaning: a digest over the rows in order, each as its key fields and its cells sorted by source field name, each cell tagged null or text, every field preceded by its length. The same number with the same digest writes nothing; with another the run becomes `inconsistent`, finally, and the call returns that state without an error. (This differs from `modules/intake`, where equal amounts written otherwise are the same content: a row that is in error has no canonical form.) Chunks beyond the declared number of rows are not refused when they arrive; `Close` is.

Format checks are made per row when its chunk arrives, and only for target fields that the revision maps. Each finding has a code from this closed list:

- `not_decimal`, `outside_profile`: a value of a `cost` or `quantity` field must be a decimal that the profile accepts as an input, and after `negate` or `scale`, which are exact, its canonical form must lie inside the profile's result bound. The value is kept, not its spelling: `1.50` becomes `1.5` and `-0` becomes `0`;
- `not_time`: a value of a `time` field must be RFC 3339; its canonical text is the instant in UTC with the fraction of a second shortened to what is needed, so that equal instants have equal text and the offset is not kept;
- `not_currency`: three uppercase letters;
- `empty`: an empty text is a value for a `text` field and a finding for every other kind;
- `required_null`, `required_not_provided`: a field that format conformance requires must have a value;
- `currency_missing`: a row in which a `cost` field has a value needs a value in the currency of costs;
- `period_missing`, `period_inverted`, `outside_window`: when the run has a window, a row needs both period values, start before end, start not before the window's start and end not after its end. Without a window a row that has both needs start before end.

A row with at least one finding is an error row. A finding is (chunk, position, row key, target field, code) and holds no value.

A row key that occurs more than once in a run, in one chunk or in two, quarantines every row that has it, whatever their content and whatever they were before; nothing here can say which of them is the row. A quarantined row has no findings. This is the cross check for duplicates; it is decided as chunks arrive, and the set of quarantined rows does not depend on the order of arrival.

So each checked row is exactly one of quarantined, error or normal, in that precedence, and the three counts add up to the rows checked. A normal row records which fields that billing readiness requires lack a value. That list is not a verdict: only the run's billing verdict says whether anything is ready.

`Close(run)` is accepted when every declared chunk was checked and the rows checked equal the declared number, and is refused otherwise with the reason and nothing written; a run that cannot be closed is left by `Abandon`. It makes the run `closed` and stores the three counts, the exact sum per mapped `cost` field and currency over the values of normal rows, and two verdicts, each with every reason that applies:

- format: `conformant` with no reason; `partial` with the single reason `mapping_partial`; `nonconformant` when `rows_in_error` or `rows_quarantined` applies, `mapping_partial` listed with them when it applies too. A partial mapping never yields `conformant`;
- billing: `ready` with no reason, else `not_ready`. Reasons: `mapping_gap` (billing readiness is not possible under the revision); `rows_in_error`; `rows_quarantined`; `rows_lacking` (a normal row lacks a required value); `totals_not_expected` (no expected sums were given for a mapped `cost` field that billing readiness requires); `totals_differ` (for a field with expected sums, the currencies that have a value among the normal rows are not exactly the expected ones, or a sum differs as a number); `window_not_declared` (the revision maps a period and the run has no window); `sum_outside_profile` (an exact sum lies outside the profile's result bound).

The two verdicts are computed from the same facts and neither is an input of the other: a run can be `partial` and `ready`, when what is missing is not needed for billing (allowed here, not the conservative choice, listed below), and `conformant` and `not_ready`. Sums are reported as text whether or not the profile holds them.

`Abandon(run, reason)` ends an `open` run, with a reason from `transfer_failed`, `superseded`, `operator`. `Discard(run, reason)` ends a `closed` run with a reason from `superseded`, `wrong_input`, `operator`: its rows and findings are dropped, its verdicts, counts and sums stay. States: `open`; final: `closed` until discarded, `discarded`, `inconsistent`, `abandoned`.

Reading, for a `closed` run only and paged, `after` being the number already read: `Rows` returns the normal rows in the order (chunk, position), each with its normalized cells and the fields it lacks, and every page carries both verdicts of the run; `Findings` the findings in the order (chunk, position, field order of the target), of which a closed run keeps the first up to a bound and reports the total; `Quarantined` the quarantined row keys with their positions.

Nothing here says which run counts. Two closed runs over the same input, or over two manifests of one subject, can both be `ready`, and a run does not become stale when its manifest is superseded; a consumer that reads both counts costs twice. The input is mandatory so that a later increment can define the one run that counts per subject and target as a whole replacement (stated limit, an exposure blocker).

## Repeats, tenant boundary, time and limits

Every operation is its own key: a repeat with the same content writes nothing and succeeds in every state the run has since reached; other content is a conflict or, for a chunk, makes an open run inconsistent. No error repeats a string of the refused request, and no finding holds a value.

Targets and mappings belong to the installation and hold no tenant data; a tenant whose export has other columns needs its own source version (stated). Runs belong to a workspace; every run operation takes the workspace, and a run of another workspace is not found in the same way as one that does not exist. The registry authorizes nothing and must not be exposed directly (stated limit).

The clock is injected, read once per operation and never goes backwards. Required and positive at construction, without defaults: targets, fields per target, mappings, revisions per mapping, runs per workspace, chunks per run, rows per chunk, rows per workspace, the length of a cell's text, findings kept per closed run, unused source field names reported per run, and the page size. The decimal profile is injected. Rows are counted when checked and given back when a run is abandoned, becomes inconsistent or is discarded; until a run is closed its findings are held for every error row, bounded by rows and fields. Runs are counted for good, whatever their end, and so are targets, mappings and revisions, which are installation-wide: at those limits nobody can open a run or correct a mapping. One lock serves every workspace. Neither has an exit here (stated limits, exposure blockers; retention is undecided).

The guarantee that no floating point is involved starts at the text this package receives; an adapter that reads a float column is outside it (stated).

## What is not decided here

The internal target version and dataset; which source fields map to which target fields for any real provider, and what each source field is; the precision limits (the profile is an input); whether a duplicate key should quarantine or keep one row; which period pair is checked; what a window means for rows that straddle it (they are in error here); which run counts for a subject, and what happens to a run when its input is superseded. Choices made here that are not the conservative ones: a run that is `partial` may be `ready`; installation-wide mappings rather than per tenant.

# Allowed files

`modules/normalize/*` (new); `docs/features/HB-188/*`.

# Verification

- Literal table tests with a test clock: every refusal of a target and of a mapping, among them a sale amount, a quantity into a cost, one source field with two kinds, `not_applicable` on a required field, a cost without its currency and half a period; the guard of `DeclareMapping` with a lost response, a stale and an early expected number; revisions and their computed support, with a field that is not required and `not_supported`, and a target without a required cost; a run opened on a revision that is not the latest; an open run unaffected by a later revision; every refusal of `Open`; null, not provided and zero through every kind and transformation; decimals at the input bound, with trailing zeros, negative zero accepted as zero, exponents refused, a scaled value at and beyond the result bound; times with offsets and fractions, equal instants with equal text; every finding code, with required null apart from required not provided; the period at each edge (start at the window's start, end at its end, start equal to end, straddling, one value lacking, no window); duplicates in one chunk, across chunks, three times, and of a row that was in error, whose findings go; counts adding up after every chunk; a malformed chunk refused whole; chunk repeats, a repeat that differs only in spelling, in an unused cell and in null against not provided; `Close` refused with chunks missing and with another row count; every reason of both verdicts, alone and together; each of the five combinations of the two verdicts that can occur, and `nonconformant` with `ready` shown not to occur; a partial mapping never conformant; expected sums met, not met, with an unexpected currency, with an expected currency that has no value, for a null cell, and not given; a sum outside the profile; rows, findings and quarantined keys not readable before closing or after discarding; both verdicts on every page; the bounds of findings and unused names with their totals and the order kept; paging; discarding and rows given back; the tenant boundary; limits; no error repeating the request and no finding holding a value.
- A test that the package's source contains no floating point type, conversion, parsing or formatting, `big.Float` and `big.Rat` included.
- Properties with a fixed seed over random runs against a reference model written in the test that decides from the raw rows each row's class, the verdicts with their reasons and the sums with 64-bit integers at mixed scales; the counts add up after every step; the same rows in any chunking and order of arrival give the same verdicts, counts, sums and, compared by key, the same normal rows.
- A run of 100,000 rows in chunks with errors and duplicates spread over it, checked against counts and sums of the generator.
- Concurrency under the race detector: the same chunks checked by several goroutines and counted once; several declarers of one mapping with one expected number, of which one appends; closing racing checking.
- The package's tests run many times in a row before any mutation run; a mutation check by a script in Python with results appended to files that are never overwritten.
- `gofmt -l`, `go vet ./...`, `go test -race -count=1 ./...`.

Remaining NOT_RUN at system level after this ticket: FD04, FD05, FD06, SIM01, AUD-QA-07. This ticket does not complete W-07.

# Deferred

A target that is a verified copy of a FOCUS version, and a validator at a fixed version; real mappings per provider; SaleCharge; parsing files; a checked link to an input manifest of `modules/intake`; QA-07; joining the Python import path; commands, permissions and screens; retention; persistence.

# Design review resolution

The independent design review of 2026-11-13 (read-only, nothing executed) returned FAIL without a blocker and without asking for a redesign: 4 HIGH, 7 MEDIUM, 3 LOW. It found no path that counts a row twice or drops one; the HIGH findings were places where `conformant`, `ready` or "cannot be mixed" claimed more than the rules established. All were adopted above.

| Finding | Resolution |
| --- | --- |
| F1 rows of a run that is not ready marked ready; nothing says which run counts | The row attribute is the list of fields lacking; every page carries both verdicts; the input is mandatory; which run counts stated as a limit and an exposure blocker |
| F2 completeness depending only on flags; reasons without effect; required and unmapped fields ambiguous | Row checks apply to mapped fields only; `complete` means nothing missing but `not_applicable`, which a required field cannot have; a cost needs its currency; readiness needs a required cost |
| F3 mixing possible through two source fields | Every source field has a declared kind and cost kind, `sale` among them; one predicate accepts an entry; the claim reworded |
| F4 period and totals optional and silent | `totals_not_expected` and `window_not_declared` are reasons for `not_ready`; refusals at `Open`; the rule for currencies and for rows lacking a period value |
| F5 the row key of the predecessor | The three-field key of an intake manifest |
| F6 what the same chunk means | A digest of what was sent |
| F7 canonical text and bounds | Stated for decimals, times, non-values, empty text and where the guarantee starts |
| F8 findings depending on arrival | Kept in order and cut at closing; a quarantined row has none; readable only when closed |
| F9 refuse or classify; lists not enumerated; unsatisfiable runs | A malformed chunk is refused whole; lists enumerated; both verdicts have reasons; `Open` and `Close` refuse counts that cannot hold |
| F10 exhaustion | `Discard`; the guard of `DeclareMapping`; the remaining limits stated |
| F11 free text | Stated; source field names are identifiers |
| F12 undecided and non-conservative choices | Listed |
| F13 tests | Added to the verification |
| F14 alternatives, profile identity, an older revision | Recorded; the profile identity stored; only the latest revision can be opened |

# Design re-review resolution

The narrow re-review of 2026-11-13 (read-only, nothing executed) returned PASS with findings, no blocker and no HIGH: thirteen of the fourteen first findings resolved or stated and two sentences of F4 still open; thirteen new findings, at most MEDIUM. It said implementation may start after text fixes and that no further design pass is needed. The rules below settle them and are part of the decision; where they are more precise than a sentence above, they govern. The change review reads them against the code.

- N1, sums. For a `cost` field, the currencies that count are those of normal rows in which that field has a value, a sum of zero included. Expected sums are given for a field when the field is named, also with no currency at all; they then match only when no normal row has a value in that field.
- N2, findings. A cell of a mapped field is not provided, null or a text. "Has a value" in the checks for required fields, for the currency of costs and for the period means that the cell is a text, valid or not, so that one fault gives one finding. Per row and field, in this order: at most one value code (`empty` before the code of the kind); `required_null` or `required_not_provided`; `currency_missing`, once, on the currency field, when some `cost` field of the row is a text and the currency is not; `period_missing` on each period field that is not a text, only when the run has a window; `period_inverted` and then `outside_window` on the start field, only when both period values are valid times. Start equal to end is inverted. `Open` refuses a window whose start is not before its end. A normal row lacks a field that billing readiness requires when its cell is not a text.
- N3. The code follows the primitive: its syntax error gives `not_decimal`, its input bound `outside_profile`, also for a text so long that the primitive reports the bound first.
- N4. `negate`, `scale` and all sums are arithmetic on the integer coefficient and scale; trailing zeros are removed first and the result is inside the result bound exactly when the primitive accepts that reduced coefficient and scale as a new decimal. Sums and expected sums are compared and printed from the integers.
- N5. A further required limit: cells per row. A row over it makes its chunk malformed.
- N6, repeats. Every operation first tests whether it is an identical repeat of what is stored, before any check of the latest revision, of state or of limits, so an identical `Open` succeeds after a newer revision exists and at the run limit. `Open` and `Check` return the run's state, which the caller must read. One digest per checked chunk is kept for as long as the run is, also after its rows were dropped. A chunk that is not an identical repeat, sent to a run that is not open, is rejected with nothing written; so is the contradicting chunk sent again.
- N7. Five combinations of the verdicts can occur; `nonconformant` with `ready` cannot, since both of its causes are also reasons for `not_ready`.
- N8. The input has no provider of its own; it is under the provider of the run's mapping.
- N9. A target has at least one field and a revision at least one `from` entry.
- N10. `not_applicable` is the caller's declaration and is not verified, like the kind of a source field; a revision's computed statement lists the fields declared so (stated limit).
- N11. Two conditions accept a `from` entry: the kind predicate, and that the transformation is allowed for the kind. A revision that declares one source field with two kinds or cost kinds is refused. A `sale` declaration can only ever be refused; it exists so that the refusal is of a stated thing.
- N12. A row key is well formed when its kind is `event` with an empty document or `locator` with a document that is an identifier, and its key is an identifier under the package's own copy of the rule. `Quarantined` is in the order (chunk, position). `Run(workspace, run)` reads a run in every state: its state and counts so far, and its sums and verdicts once closed, also after it was discarded (sums before closing would shift as duplicates quarantine rows). `Revision(mapping, number)` and `Latest(mapping)` read revisions. A time is accepted in the RFC 3339 form with capital `T`, a capital `Z` or a numeric offset, at most nine digits of a fraction and no leap second.
- N13, further choices that are not the conservative ones, for the owner: a mapped `cost` field that billing readiness does not require needs no expected sums and is not cross checked; a window turns rows outside it into error rows, so a check of the run's scope enters the format verdict; unused cells are ignored; rows are not reserved at `Open`, so two half-checked runs can stop each other at the workspace limit, and the exit is `Abandon`.

# Change review resolution

The independent change review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker or HIGH, 2 MEDIUM and 10 LOW. It found no path that puts a row in two classes or none, lets a duplicate key escape quarantine, makes a run `ready` on a wrong, missing or duplicated sum, or `conformant` under a partial mapping, and confirmed by reading the arithmetic, the mixing predicate, the comparison of expected sums, the accounting of rows and the tenant boundary.

| Finding | Resolution |
| --- | --- |
| F1 MEDIUM the property test drew random numbers while ranging over a map, so a seed did not reproduce a run | Fields and currencies are walked in a fixed order |
| F2 MEDIUM a run that was abandoned or became inconsistent kept the unused source names it had collected | Dropped with the rows |
| F3 `Run` gives no sums before closing | The text says so |
| F4 the window in a run's identity could wrap for instants far from the present | Written as UTC text |
| F5 a canonical time could leave the four-digit years | Refused as `not_time` |
| F6 offsets beyond a day accepted | The offset is at most 23:59 |
| F7 a malformed expected field name judged differently with and without a currency | Invalid in both cases |
| F8 `Open` accepted a run that declares more rows than the workspace may hold | A limit error |
| F9 the number of workspaces is unbounded | Stated below |
| F10 closed rows kept numbers nobody reads | Released at closing |
| F11 the profile identity read when viewed | Recorded at opening |
| F12 the fields a row lacks | Stated below |

Stated with this review, as the code behaves:

- The chunk digest also holds the number of cells of each row, so that a cell cannot be read as the key of the next row. A null cell that carries a text is malformed.
- `scale` by ten to the power zero is refused, and a `missing` entry that carries anything but its reason.
- Error kinds: a malformed request is invalid, rows over the capacity of the declared chunks included; more chunks than the limit, more rows than the workspace may hold and no room are limit errors; a declaration that contradicts a stored target, revision or state is rejected; an unknown mapping, revision or run is not found.
- The fields a normal row lacks are among the mapped ones; a field that billing readiness requires and the revision does not map shows only as `mapping_gap`.
- The order of fields is part of a target's identity.
- Workspaces are created by the first run opened in them and are not limited in number (stated limit, an exposure blocker with the others).
- Two checks in the code can change no result and are kept as statements of the rule: the key kind in the chunk digest, which the presence of a document already implies, and the refusal of a small `z`, which the time parser refuses as well.
