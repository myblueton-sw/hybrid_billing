---
type: ImplementationPlan
title: TokenMeter snapshot exports with usage events, cost classes and an eligibility verdict
description: Defines the first model of work item W-11, a registry of receptions of snapshot exports from TokenMeter in which usage events keep their identity, token kinds, outcome and a cost with grade, origin and original text, costs are summed apart by class, and an eligibility verdict keeps an export that is taken from a list interface, incomplete, inconsistent or conflicting out of a month close.
status: draft
---

# Scope and ownership

[HB-196](https://linear.app/hybrid-billing/issue/HB-196/w-11-tokenmeter-snapshot-exports-with-usage-events-cost-classes-and-an) is the first increment of work item W-11 (requirement HB-11, a conditional requirement that applies when the TokenMeter connection is opened). It is the seventeenth increment built under the owner's instruction of 2026-11-12. Base: main `5b29bdd`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the model decides which AI supply cost is kept apart as referenced by an invoice and holds usage per tenant. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-13:

- Requirement HB-11: the meaning of input, output and cache token kinds, of request, attempt and event, and of the evidence grade is preserved. The AI supply cost of an organisation and TokenMeter's own SaaS fee are separate transactions. Proposed direction: BILLED is not taken as an official invoice obtained; a provider-confirmed cost is linked with the attribution of consumption, and an estimated amount is not added as a separate cost; a float source keeps its original form and the limits of its conversion, and eligibility is judged under a decimal contract. Scale: a list API of the user interface is not reused as a month-close export; a stable snapshot export and a manifest of all rows, sums and quantity units are required.
- Acceptance of HB-11: a retried attempt, the inclusion relation of cache tokens, None, an observed estimate, and the duplication of the same GPU cost with HawkEye are verified. Linked source cases BILL01, COM01, SIM03 are NOT_RUN; their text is not in the planning files read.
- Open decisions of HB-11, not made: the full export, the contract per evidence grade, the input precision of a fixed unit price. No TokenMeter interface, field, grade or status name other than BILLED is in the planning files read.
- AR-04: an AI input keeps provider, model, version and endpoint and whether a call succeeded, failed, was cancelled or streamed to completion; whether the provider bills and whether a status may be sold under a customer contract are judged independently; rate denominators, inclusion of categories and the validity of model aliases and prices are stated. Its pricing parts belong to HB-16. AUD-AR-04 is NOT_RUN.
- AR-05: prompt and response bodies are excluded from billing source fields and a consumer is identified by a pseudonymous reference. AUD-AR-05 is NOT_RUN.
- RT-01: an event keeps source and event identifiers, schema version, period and source revision, a request adds request and attempt identifiers, a cost adds amount, currency, origin and status. AR-02 (authentication of the connection) names HB-11 and is not touched here.
- Existing code: `modules/hawkeye` (HB-194) receives power cost statements and has the patterns reused here: a reception under the caller's identifier, one that is not voided per source object, an eligibility verdict with ordered reasons, exclusion of overlapping periods, `Counting` with a sequence number.

# Decision

One Go package, `modules/tokenmeter`, standard library and the decimal primitive only, with an in-memory registry. It calls no TokenMeter and has no entry point wired in this increment: an adapter that fetched an export reports it, and the registry decides what it may be used for.

What this can and cannot show: that a reception is called eligible only when every listed condition holds as reported; that at any one reading at most one reception of a source is eligible for a moment of time and no event is held by two receptions that count; and that the costs of its events are summed apart by class. It cannot know that a report is what TokenMeter said, what TokenMeter's fields mean, that a grade, an origin or a document reference is true, or that an export was already used. The class of costs that name a provider document is called `invoice_referenced`, not confirmed: a provider's invoice is per account and month, a cost per event is TokenMeter's allocation of it, and nothing here compares the events that name a document with the document's amount or notices one document named in two receptions. `eligible` says nothing about any class: an eligible reception can have no invoice-referenced cost at all and token counts that are not known. A class sum may be used only from the reception that `Counting` returns (stated limits).

Not shown here, although the acceptance names it: that the same GPU cost is not counted both here and through HawkEye. Nothing links the two packages; this one only keeps a cost that the adapter marks as self-hosted apart from provider costs. The acceptance item is recorded as not met here.

Not violated here and open for later increments: the package computes no price and no selling charge, which a test checks by scanning the source for multiplication and division of the decimal primitive; TokenMeter's own SaaS fee is another transaction and has no place here.

Alternative considered: receiving events through `modules/intake` as rows of a document. Not taken: an intake row is a key, a digest and one amount, and the meaning that has to be preserved here, token kinds, attempts, grades, is exactly what it does not carry.

Every caller-supplied string is an identifier under the project's identifier rule (letters, digits, `. _ : -`, at most 128 bytes), of which the package has its own copy, a three-letter uppercase currency code, a whole number, the text of an amount as defined below, or a value of a closed list. So no prompt or response body can be reported: an event has no free text (AR-05), and a consumer is an identifier that the adapter must make a pseudonymous reference (stated obligation). Identifiers of TokenMeter that do not fit the rule are encoded by the adapter, always in the same way; two spellings are two different things, and one installation reported under two source identifiers escapes every rule between receptions (stated obligation). No floating point is used; all arithmetic is the package's own, exact, on integers of any size. All times are kept and returned in UTC.

## Amount texts

Every amount, of an event and of a declared total, is given as the text the source produced and is kept as that text. Its form is `-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]{1,2})?`, at most forty characters; anything else, `NaN`, `+1`, `.5`, `5.`, a leading zero, is malformed. Its exact value is computed from the text without floating point. A negative amount is accepted, as a credit, and lowers the sum it enters; `-0` is zero. The significant digits of a text are the digits of its coefficient after leading zeros and trailing zeros are removed, wherever the point stands; zero has one. A value is inside the profile when the decimal primitive accepts its reduced coefficient and scale as a new decimal.

A reception says how its source holds amounts, `decimal` or `float`. For a `float` source a text with more than fifteen significant digits is beyond what a float carries reliably and is not used (below). Whether a small tolerance should be allowed instead is not decided here.

## Exports and receptions

An export is what TokenMeter produced: it is identified in a workspace by its source, an identifier of the TokenMeter installation, and its snapshot identifier there. A reception is one report of an export by an adapter, under an identifier the caller chooses, unique in the workspace. At most one reception that is not voided exists per export; a second is a conflict that returns the identifier of the first, and a voided reception frees the export. So a wrong header and a cut-off transfer have the same exit: void the reception and report the export again under a new reception identifier. There is no supersession between exports and the source revision is not used for anything: a regenerated snapshot is another export, and which of two counts is settled by voiding one.

`Open(reception)` records its header, immutable:

- reception identifier; source and snapshot identifier; the schema version and the connection it was fetched through, recorded as first reported and not part of what a repeat is compared with;
- its period, a start and an end in whole seconds, the start before the end, not longer than the maximum given at construction; the time from its start up to and not including its end;
- its source revision, a whole number from 1 to 999999, recorded only;
- how it was produced: `snapshot_export`, a stable export made for closing a period, or `list_interface`, rows read from a list of the user interface;
- how the source holds amounts, `decimal` or `float`; and the unit of its quantities, which must be `token`: an export in another unit is converted by the adapter before it is reported;
- what it declares: the number of events, at least 1 and at most the limit; for each of the four token kinds the total over all events as a whole number of at most thirty digits, or not stated; the total cost per currency as amount texts, possibly none; and where the declaration comes from, `source_manifest` when TokenMeter stated it apart from the rows, or `derived` when the adapter computed it from the rows it received.

## Events

`Post(reception, events)` reports a batch of events of an `open` reception, at least one and at most the batch limit. A batch is applied whole or not at all. An event has:

- an event identifier unique in the reception, and the request and attempt it belongs to; the pair of request and attempt is unique in the reception too, so that one attempt reported under two event identifiers is refused, while a retried request has one event per attempt and each is kept;
- the time of the event, in whole seconds;
- a consumer, a provider and a model, identifiers; and a version and an endpoint, identifiers or not known;
- its outcome from a closed list (assumed names): `succeeded`, `failed`, `cancelled`, `stream_incomplete`;
- four token counts, input, output, cache read and cache write, each either a whole number from 0 up to eighteen digits or not known. Not known is not zero and stays apart everywhere;
- the evidence grade of those counts (assumed names): `provider_reported`, `estimated` or `not_known`;
- the relation of its cache-read count to its input count: `included`, when the input count contains it, `separate`, or `not_known`;
- optionally a cost, all of its parts or none: the amount text; a currency; an evidence grade from a closed list (assumed names) `invoiced`, `billed`, `computed`, `estimated`, `not_known`; an origin, `provider` or `self_hosted_gpu`; and, exactly when the grade is `invoiced`, a reference to the provider's document. An `invoiced` cost has the origin `provider`. An origin `tokenmeter_fee` is a known value that is always rejected.

An event reported again with identical content writes nothing; the same event identifier with other content, or a request and attempt already reported under another event identifier, is a conflict and the batch is refused. Events beyond the declared number are rejected, and so is an event that would bring the distinct currencies of a reception, declared ones and those of its events, or its distinct documents, above their limits.

Findings attach to an event, in this order; an event with a finding is kept and counted as in error, and its cost enters no sum:

- `outside_period`: its time is not inside the reception's period;
- `cache_relation_broken`: its relation is `included` and its cache-read count is known while its input count is not known or is smaller;
- `float_precision_exceeded`: the source holds amounts as floats and the text of its cost has more than fifteen significant digits;
- `cost_outside_profile`: the exact value of its cost is not inside the decimal profile.

## Closing and cost classes

`Close(reception)` is accepted for an `open` reception at any time, also with fewer events than declared, and makes the one pass over its events; nothing read later scans events again. It stores, as maps that are empty when there is nothing: the number of events and of events in error; the number of events per outcome; per usage grade and token kind the exact sum of the known counts over all events, with findings or without, and per token kind the number of events where the count is not known; the net input, the sum over events of the input count less the cache-read count where the relation is `included` and both are known, and of the input count where the relation is `separate` and it is known, with the number of events for which it cannot be said; per currency the exact sum of costs of each class over events without a finding; and per document the number of such events and their sum per currency. The class of a cost is decided in this order:

- `estimated`: the grade is `estimated`, whatever the origin;
- `self_hosted`: the origin is `self_hosted_gpu`;
- `invoice_referenced`: the grade is `invoiced`;
- `unconfirmed`: the grade is `billed`, `computed` or `not_known`. A status BILLED at TokenMeter maps to `billed` and is not an invoice.

The four sums are reported apart and are never added to each other by this package. That a self-hosted GPU cost is the one HawkEye's power statements carry, and so an attribution and not a further cost, is an assumption.

`Void(reception, reason)` ends an `open` or `closed` reception, with a reason from a closed list (`withdrawn_at_source`, `regenerated_at_source`, `wrong_report`, `operator`). Its events stay readable. States: `open`; `closed`; final: `voided`. A reception remembers whether it was closed.

## Eligibility

`Eligibility(reception)` reads, without writing and from stored figures only, `eligible` or `not_eligible` with every reason that applies, in this order. A voided reception has the single reason `voided`.

- `not_closed`: still open; the last five reasons of this list are not given for it;
- `list_interface`: it was read from a list of the user interface;
- `declaration_derived`: its declaration was computed from the rows received;
- `float_total_imprecise`: its source holds amounts as floats and a declared cost total has more than fifteen significant digits;
- `period_conflict`: another reception of the same source that is not voided has a period that overlaps this one's, whatever else is true of either, also one that can never be eligible itself, such as one read from a list for a single day; the others are returned by identifier in bytewise order up to the bound of lists, with their number;
- `event_shared`: an event identifier, or a pair of request and attempt, of this reception is also held by another reception of the same source that is not voided, as when a request around midnight is in the snapshots of two months; the others are returned in the same way. The relation is symmetric and its exit is a void;
- `events_in_error`: an event has a finding;
- `incomplete`: the number of events is not the declared number;
- `totals_not_stated`: the total of a token kind is not stated;
- `totals_differ`: for a token kind whose total is stated, the sum of known counts over all grades is not that total; or the currencies that have a cost without a finding are not exactly the declared ones; or the sum of all such costs of a currency, the four classes together, is not the declared total as a number;
- `sum_outside_profile`: a cost sum of a class, or the sum of a document in a currency, lies outside the decimal profile. It can occur alone, since a declared total is not bound by the profile, and for a document alone, since credits of one document can cancel the costs of another within a class.

`incomplete` and `totals_differ` show consistency with the declaration; that the declaration is TokenMeter's own is the adapter's statement in the header. That a manifest's total cost covers all four classes is an assumption, and no figure of a manifest checks the split into classes.

Every operation that writes, a first `Open`, a batch that adds an event, a first `Close`, a first `Void`, raises a sequence number of the source in the workspace by one. `Counting(workspace, source, start, end)` reads the eligible reception whose period is exactly the one given, with its snapshot identifier and the sequence number, also when nothing counts; for a source nothing was received from it is not found with the number zero. The number starts again with the process.

## Reading

`Reception(workspace, reception)` reads a reception in every state: header, state, whether it was closed, the number of events, and once closed everything `Close` stored; the reason of a void; the times it was opened and last changed, the latter moving at closing and voiding. `Events(reception, after, limit)` reads the events of a reception that is not open in the bytewise order of event identifiers, each with its findings in their order and, when it has a cost, the class of that cost whether or not a finding keeps it out of the sums, in pages, `after` being the number already read; for an open reception it is rejected. `Receptions(workspace, source, after, limit)` lists the receptions of a source in the bytewise order of their identifiers with snapshot, period and state, `after` being an exclusive lower bound that need not exist.

## Repeats, error kinds, tenant boundary, time and limits

Every operation is its own key and first tests for an identical repeat: a repeat writes nothing, succeeds in every state reached since and returns that state. A batch whose events were all reported identically before is such a repeat, in every state; a batch with one new event for a reception that is not open is rejected whole. Amount texts are compared as texts, in events and in headers, since they are a record of what the source said.

A request is judged in this order, after the reception it concerns was looked up, an unknown one or one of another workspace being not found: malformed, which is invalid and includes a value outside its closed list or range, a unit other than `token`, a period or event time that is not in whole seconds, an event without a time, a period that does not run forward or is too long, an amount text not of the stated form, a cost with a part missing, an `invoiced` cost without a document or with another origin, a document on another grade, an event identifier or a pair of request and attempt twice in one batch, and an empty batch; a declared number of events, a batch or a number of declared currencies above its limit, a limit error; an identifier that exists, which is a repeat when identical and a conflict otherwise, with a second reception for an export, a request and attempt under another event identifier, and a second `Void` with another reason being conflicts; what a stored state or rule refuses, rejected: an origin `tokenmeter_fee`, a new event for a reception that is not open, an event beyond the declared number or beyond the limit of currencies or documents, a first `Close` of a reception that is not open, reading the events of an open reception; no room, a limit error. A page limit above the page size is invalid; a position beyond the end gives an empty page; reads create nothing. No error repeats a string of the request.

Every operation takes the workspace and reaches only that workspace's receptions. The connection is recorded and not checked (AR-02; stated limit, an exposure blocker). The registry authorizes nothing and must not be exposed directly.

The clock is injected, read once per operation, and the registry keeps the highest reading it has seen. Required at construction, without defaults and positive: receptions per workspace, events per reception, events per batch, currencies per reception, documents per reception, the page size, the bound of lists, the longest period; and the decimal profile. Receptions are counted for good and keep their events, voided ones included; workspaces are created by first use and not limited in number; one lock serves every workspace, and the check for overlapping periods and `Counting` read every reception of a source while it is held; none has an exit here (stated limits, exposure blockers).

## What is not decided here

The full export of TokenMeter, the contract per evidence grade and the precision of fixed unit prices: these are the decisions the design part of W-11 asks for, they remain the owner's, and the design part stays open. Also: a tolerance for amounts held as floats; which outcomes a provider bills and which may be sold; whether an `unconfirmed` cost may ever be used; how an invoice-referenced cost is reconciled with its document and reaches a billing, and what keeps it from being used twice; how the `self_hosted` attribution meets the eligible power cost of `modules/hawkeye`; who may void.

Assumptions made here: the names of outcomes, grades, relations and origins and that these lists are complete, so that a rented GPU with an invoice can only be reported as `provider`; that BILLED maps to `billed`; that every event has a request and an attempt and that there is one event per attempt, without which an export cannot be received whole; that streaming is part of the outcome and every event has an outcome; that a cost attaches to an event; that a manifest's total cost covers every class and its token totals every usage grade; that fifteen significant digits is the limit of a float; that a self-hosted GPU cost is the one HawkEye carries.

Not the conservative choices, for the owner: `Close` accepts fewer events than declared; how an export was produced, how its source holds amounts, whether the declaration is the source's own, and for every event its outcome, grades, cache relation, origin and document are the adapter's statements and nothing checks them, so a cost is `invoice_referenced` on a reference alone and a self-hosted cost marked `provider` is taken as one; a `separate` relation that is wrong counts cached tokens twice in the net input; a negative amount is taken as a credit; a void works as a choice between two exports of a period; one document named in several receptions is not noticed.

Conservative choices that cost availability, none with an exit other than a new export from the source: one event with a finding makes its whole export not eligible; an amount text that is malformed refuses its batch, so an export with one such row cannot be received whole; a source that holds amounts as floats and adds or multiplies them will often produce texts of sixteen or seventeen digits and a total that differs from the exact sum of its rows, and is then never eligible; an export that carries rows of TokenMeter's own fee can only be reported without them and then differs from its manifest; an export whose source states no manifest, or no total for a token kind, is never eligible; any overlap of periods and any shared event makes every reception concerned not eligible.

# Allowed files

`modules/tokenmeter/*` (new); `docs/features/HB-196/*`.

# Verification

- The cases of the acceptance, as literal tests at the level of this package: a request retried three times with a failed, a cancelled and a succeeded attempt, each kept, and the same attempt under a second event identifier refused; cache `included`, `separate` and `not_known` with the net input of each, an unknown cache count, and a broken relation; counts that are not known beside zero, in sums and in the numbers of unknowns, and counts of each usage grade apart; an `estimated` cost that enters no other class; a `billed` cost that is not `invoice_referenced`; a self-hosted cost kept apart; a `tokenmeter_fee` refused. The item about HawkEye is recorded as not met.
- Literal table tests with a test clock: every refusal of a header and of an event; amount texts: every malformed form (`+1`, `.5`, `5.`, `1e`, an exponent of three digits, forty-one characters, `NaN`, a leading zero), exponents of either sign, negative amounts and negative zero, significant digits of `100000`, `1e16`, `0.00012`, `1.50` and zero, the fifteenth and sixteenth digit for a float and for a decimal source, values at and beyond the profile, a declared total in exponent form and one of sixteen digits; every class of cost with its precedence, estimated with self-hosted and `invoiced` with `self_hosted_gpu` among them; sums per document and a document shared by events; findings each alone and all four in their order; every reason of eligibility alone, `sum_outside_profile` included, and combined in order, while open and after closing; token totals stated, not stated, met and not met, a declared currency without a cost and a cost in an undeclared currency; one reception per export, the occupant returned, a voided one freeing it; periods that touch and overlap, a list reception for a day against a month's snapshot; an event identifier and a pair held by two receptions, symmetric, listed, removed by a void; `Counting` and the sequence number, not moving on repeats and refusals; batches applied whole, a batch reported before in every state, partly reported, and one that conflicts, the same pair twice in a batch, the limits of currencies and documents; `Void` with every reason from each state and events readable afterwards; a reception closed without events; pages of events and of receptions; the tenant boundary; limits; times in UTC; errors that are fixed values; a scan of the source for floating point and for multiplication and division of the decimal primitive.
- The parser of amount texts against an independent one, `math/big.Rat` in the test only, over generated texts that include the edges of the form: value, significant digits and whether the profile holds it.
- Properties with a fixed seed over random histories against a reference model written in the test with 64-bit integers, which decides the outcome of every operation and, from the events it was given, findings, classes, the stored sums and a verdict with reasons for every reception after every step; a second set of sums and a second verdict are recomputed from what `Reception`, `Events` and `Receptions` return, with class and findings derived by the test from grade, origin, text, time and counts and not taken from what `Events` says; both must equal the registry's. Read from the registry besides: no two eligible receptions of a source overlap or share an event; at most one reception that is not voided per export; `Counting` returns a reception exactly when it is eligible and has that period. The generator draws sources, periods and event identifiers from small sets, bounds its amounts so that nothing overflows, and the test fails unless every reason, every class, every finding and every outcome of an operation was reached a minimum number of times, a reason, class or finding being counted once per reception or event and not once per reading. No random number is drawn while ranging over a map.
- An export of 100,000 events in batches whose counts, token sums and cost sums per class equal the generator's.
- Concurrency under the race detector: the same batches reported by several goroutines and counted once while another closes; two receptions of one export, of which one succeeds; a void against reads of eligibility and of what counts.
- The package's tests run many times in a row before any mutation run; a mutation check by a script in Python with results appended to files that are never overwritten.
- `gofmt -l`, `go vet ./...`, `go test -race -count=1 ./...`.

Remaining NOT_RUN at system level after this ticket: BILL01, COM01, SIM03, AUD-AR-02, AUD-AR-04, AUD-AR-05, AUD-RT-01. This ticket does not complete W-11, and the TokenMeter connection is not opened.

# Deferred

An adapter and any call to TokenMeter, with the stable snapshot export; authentication and tenant binding of the connection (AR-02); rates, denominators, model aliases and which outcome may be sold (AR-04, HB-16); TokenMeter's own fee as a transaction; passing an invoice-referenced cost to a billing and keeping it from being used twice; meeting the self-hosted attribution with HawkEye's cost; the fields of RT-01 that are not kept: resource, meter, workload, quality, and a correction that names what it supersedes; reconciling an invoice-referenced cost with its document; commands, permissions and screens; persistence.

# Design review resolution

The independent design review of 2026-11-13 (read-only, nothing executed) returned FAIL without a blocker and without asking for a redesign: 5 HIGH, 14 MEDIUM, 7 LOW. It judged the structure taken over from HB-194 sound and found no path by which an estimated, billed, computed or self-hosted cost enters the sum of invoiced ones; its HIGH findings were about what `confirmed` and `eligible` may mean, a double count across exports, the contract for floats and where an evidence grade sits. All were adopted above.

| Finding | Resolution |
| --- | --- |
| F1 `confirmed` claiming too much | The class is `invoice_referenced`; sums per document; what is not reconciled stated |
| F2 an event in two exports | `event_shared`, symmetric, with the others listed |
| F3 the contract for floats | One form for every amount text, significant digits defined, the kind of numeric per reception, a reason for an imprecise declared total, the consequence and the open tolerance stated |
| F4 where a grade sits; None | A usage grade on every event with sums apart; a cost grade `not_known`; version and endpoint may be not known |
| F5 the duplication with HawkEye | Recorded as not met; the sentence is an assumption |
| F6 one event per attempt | Listed with its consequence |
| F7 the cache relation per export | Per event, with `not_known` |
| F8 fields of RT-01 | An event time with a finding; the rest listed as not kept |
| F9 net input, unknowns, a manifest without a total | Defined; `totals_not_stated` |
| F10, F11 negative and malformed amounts | Stated |
| F12, F13 the total over classes; `eligible` beside sums | Stated |
| F14 to F17 conflicts with receptions that never count, adapter flags, the unit, fee rows, spelling | Stated; the unit is a header field |
| F18 ambiguities | Settled in the text |
| F19 verification | Derivation by the test, a parser oracle, the literal cases |
| F20 to F26 | Assumptions listed, no exit stated, lookup first, empty maps, one pass at closing, wording |

# Design re-review resolution

The narrow re-review of 2026-11-13 (read-only, nothing executed) returned PASS with findings, no blocker and no HIGH: nineteen of the twenty-six first findings resolved, six partly and one it could not judge; fifteen new findings, seven MEDIUM and eight LOW, none of which yields a wrong `eligible`. It said implementation may start after text fixes. The rules below settle them and are part of the decision; where they are more precise than a sentence above, they govern. The change review reads them against the code.

- N1, the value of an amount text. It is an integer coefficient with the smallest scale that is not negative: a positive exponent is expanded into the coefficient, trailing zeros after the point are removed, zero is coefficient 0 at scale 0. It is inside the profile when the primitive's constructor of a new decimal accepts that coefficient and scale, which is the bound of results, not the narrower one of parsed inputs.
- N2, net input. It ranges over all events, with findings or without. An event with `cache_relation_broken` adds nothing and counts among those for which it cannot be said, so the net input is never negative. It is one figure over all usage grades, estimated counts included (stated limit).
- N3, how sharing is known. `Post` keeps, per source, an index of event identifiers and of pairs of request and attempt with the receptions that hold them, and per reception the number of identities it shares with each other one. `Eligibility` reads its own stored figures, the headers and states of the source's receptions and those numbers; voided receptions are left out at reading, so a void scans nothing. An open reception shares with what it has received so far, a reception that can never be eligible shares, and the reason is given to an open reception too. The index is kept for good, voided receptions included (with the stated limits of memory and the single lock).
- N4, an assumption without an exit: event identifiers and pairs of request and attempt are unique and stable across all exports of a source. If TokenMeter numbers rows anew in every export, every later reception shares with the first and only one reception per source can ever be eligible. When two exports report the same event with the same time and their periods do not overlap, one of them has `outside_period` already; `event_shared` adds something when the times differ or the periods overlap.
- N5. An `invoiced` cost with the origin `self_hosted_gpu` or `tokenmeter_fee` is malformed and never classified, so the classes `self_hosted` and `invoice_referenced` cannot meet and their order among each other cannot be observed.
- N6. Sums are returned as plain decimals without an exponent, zeros at the end of the fraction removed, zero as `0`, whether or not the profile holds them. Token sums are whole numbers.
- N7, the order of checks. For `Open` there is no reception to look up; an identifier that is new is the normal case. Within a batch a kind of refusal takes precedence over the position of the event that causes it: any malformed event makes the batch invalid before a conflict of another event is looked at. Also invalid: an identifier or currency that breaks its form, a page limit below one, and a `Counting` whose period does not run forward, is not in whole seconds or is too long. The declared costs are a set of currencies, so none occurs twice.
- N8. A whole number has the form `0|[1-9][0-9]*` and is compared as text in a repeat.
- N10. An entry of a stored map exists exactly when at least one event contributes to it, also when its sum is zero; a document that only events with findings name has no entry, though it counted towards the limit of documents when it was reported.
- N11. The five reasons not given while a reception is open are `events_in_error`, `incomplete`, `totals_not_stated`, `totals_differ` and `sum_outside_profile`; that `totals_not_stated`, a fact of the header, is among them is intended, so that the header facts shown early are those that no later report can cure.
- N12. The total of the four classes is formed only to compare it with the declared total and is never returned.
- N13. Token sums range over all events and cost sums over events without a finding, so a finding on an event that has a cost also makes the totals differ from a true manifest.
- N14, tests added: `1.50` against `1.5`, and `-0` against `0`, as a repeated event and a repeated header, each a conflict; `1e05` and `1E5` as valid; an event at the start of the period, one second before its end and at its end; `float_total_imprecise` while open; `totals_not_stated` with `totals_differ`; `Counting` with a malformed period.
- N15, further consequences that cost availability: an export without any event cannot be reported, so an empty month looks like a missing one; an export that names more documents or currencies than the limit cannot be received whole; one event whose input count is not known beside a known cache-read count under `included` makes its export not eligible. An eligible reception may consist of estimated usage only.

# Change review resolution

The independent change review of 2026-11-13 (read-only, nothing executed by the reviewer) returned PASS with findings, no blocker and no HIGH: two MEDIUM about verification the plan names and the tests did not contain, eight LOW. It found no path to a wrong `eligible`, to a cost that is not invoiced entering `invoice_referenced`, to a lossy amount or across a workspace, and it held the four mutants removed as equivalent to be so. All findings were resolved or stated; the corrections were not reviewed again by a separate reviewer.

| Finding | Resolution |
| --- | --- |
| C1 concurrency cases named and not tested | A second concurrency test: one batch reported by eight goroutines while another closes, counted once; eight receptions of one export, of which one succeeds and the others name it; a void against readers of eligibility and of what counts, who never see `eligible` after `voided`. The test fails unless the void was raced at least ten times |
| C2 the order of reasons pinned only inside groups | One literal verdict with the eleven reasons other than `voided` on one reception, open and closed |
| C3 the parser oracle did not compare the profile | Added, with a minimum of texts inside and outside |
| C4 the source scan claimed more than it checked | Any call of the decimal primitive other than its constructor fails; a multiplication or a division with remainder is allowed only by ten; a comment anywhere in a line fails |
| C5 a document sum outside the profile while its class sum is inside | Changed in the code and in the rule above: such a reception has `sum_outside_profile`. This is the conservative choice and costs availability |
| C6 one reception per export not read from the registry | Read from `Receptions` in the audit |
| C7 thresholds counted per audit | Counted once per reception, event or period |
| C8 assumptions the model shares with the code | Stated in the verification record; two literal cases added for a batch over the limit that is also malformed or conflicting |
| C9 discarded results in a test of the boundary | Asserted |
| C10 names and small deviations | `RelationNotKnown` and `ByOperator`; an event without a time stated as malformed; the stale sentence on thresholds corrected; a void with every reason from an open and from a closed reception; unused parameters removed. The mutation script and its result files stay outside the repository, as for the tickets before; the record says so |

