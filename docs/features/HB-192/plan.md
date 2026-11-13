---
type: ImplementationPlan
title: Reconciliation runs with explained residuals and a billing readiness verdict per billing basis
description: Defines the first model of work item W-09, reconciliation runs per provider document in which the document total, the internal billed cost and typed explanation lines must leave an unexplained residual of exactly zero, and a readiness verdict per billing basis in which an on-premises basis carries no provider document and an independent price shows when its cost is not determined.
status: draft
---

# Scope and ownership

[HB-192](https://linear.app/hybrid-billing/issue/HB-192/w-09-reconciliation-runs-with-explained-residuals-and-billing) is the first increment of work item W-09 (requirement HB-09). It is the fifteenth increment built under the owner's instruction of 2026-11-12. Base: main `d4e9529`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the model decides whether a cost is called reconciled and ready for billing, and holds amounts per tenant. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-13:

- Requirement HB-09: the reconciliation residual of a document is explained for the same scope and currency. EffectiveCost is not matched to the invoice total of the same month, and an independent selling price or an internal cost follows its own evidence standard. Proposed direction: a ReconciliationRun and a table of explained differences per document issuer, account, document, revision and currency; tax, support fee, credit, commitment purchase, Marketplace and rounding items are linked without duplication; billing readiness is judged by a condition per billing_basis. Scale: aggregation by set operations or bounded external aggregation, split per document or shared pool; a sample check does not replace the full reconciliation.
- Acceptance of HB-09: an unexplained difference of zero; no fake provider invoice for on-premises; a billing at an independent unit price whose needed quantity is complete shows clearly that its cost is not determined. Linked source cases REC01, REC02, COM02, FD07 are NOT_RUN; their text is not in the planning files read.
- Open decisions of HB-09, not made: the reconciliation scope of a document, the allowed rounding policy, the evidence an independent unit price must have. The values of billing_basis are named nowhere in the planning files read.
- AR-03 names HB-09: an unverified extracted value is not an input of a confirmed calculation. `modules/intake` (HB-186) keeps such a value out of an input manifest. A document total posted here can be exactly such a value; this increment records where it was taken from and checks nothing (see the decision).
- Existing code: `modules/normalize` (HB-188) keeps the four cost kinds apart and gives exact sums per cost field and currency of a closed run; `modules/evidence` (HB-190) records what is held; `packages/primitives/decimal` gives exact decimals under a profile. Nothing reconciles a document.

# Decision

One Go package, `modules/reconcile`, standard library and the decimal primitive only, with an in-memory registry. It reads no document, sums no rows of another module and has no entry point wired in this increment: a caller posts the figures, and the registry decides whether they balance and what may be called ready.

What this can and cannot show: that a run is called `explained` only when its residual is exactly zero, that no item explains a difference twice, that what is called rounding stays inside a bound, and that a run of a basis without a provider document carries none. It cannot know that a posted figure is true, that the basis a caller declares is the real one, or that an explanation is the real cause of a difference: any single line of another category than rounding can balance a run, so the rounding bound limits what may be called rounding and not what may be left unexplained. Each figure and line carries references to its evidence, which nothing here opens. The verdict is therefore called `explained`, not reconciled, and every reading of a run shows the count and sum of lines per category beside it (stated limits; listed below as not the conservative choice).

Alternative considered: computing the internal side here from the rows of `modules/normalize`. It was not taken: which rows belong to a document is the open decision "reconciliation scope", and a caller that knows the scope posts the parts.

Every caller-supplied string is an identifier under the project's identifier rule, of which the package has its own copy, a three-letter uppercase currency code, a decimal that the profile given at construction accepts as an input, or a value of a closed list. A document number that contains other characters has to be encoded by the caller, always in the same way, and a period is an identifier the caller spells consistently; two spellings are two keys (stated obligation). A total, part or line beyond the profile's input bound cannot be posted. All arithmetic is the package's own, exact, on integer coefficient and scale, and no floating point is used; a sum or residual is printed in the primitive's canonical form whether or not the profile holds it. All times are kept and returned in UTC.

## Billing bases, subjects and the one run that counts

A run has one billing basis from a closed list. The three values are an assumption of this increment, since the planning files name none:

- `provider_document`: a cost billed on the basis of a provider's document, which has to balance;
- `independent_price`: a billing at a unit price of the seller's own, which follows the evidence of quantity and price;
- `internal_cost`: a cost from an internal pool, such as on-premises infrastructure.

Each basis has an exact set of fields, and a run that has a field of another basis, or lacks one of its own, is invalid. In particular a run carries a document exactly when its basis is `provider_document`. The basis is the caller's declaration: nothing here stops a cloud cost from being opened as `internal_cost`, or an internal one under an invented issuer (stated limit).

Every run has a subject key, and in a workspace at most one run that is not voided exists per subject key. Opening a second is a conflict; a voided run frees its key. This is the single rule for which run counts on every basis:

- `provider_document`: (issuer, account, document, revision, currency), the revision a number of at least 1;
- `independent_price`: (customer, period, charge), the charge an identifier of what is billed;
- `internal_cost`: (customer, period, pool).

Keys and every index below are per workspace; the same key in two workspaces is two unrelated runs. Nothing links a `provider_document` run to the customer and period it is billed to (stated limit: readiness here is of a run, not of a billing).

For a document, (issuer, account, document) without revision and currency, the registry keeps the highest revision for which a run was ever opened. It only rises; voiding a run does not lower it. A run of a lower revision is superseded from that moment, whatever becomes of the higher one, and opening a run for a revision below the highest is rejected. A later revision supersedes in whatever currency it is issued; within the highest revision there is one run per currency. So a reissued document takes the earlier one out of what counts as soon as it is known, also while its own run is still open, unexplained or voided, and nothing brings the earlier one back (the conservative choice; nothing here checks that a higher revision exists at the provider).

## Runs on a provider document

`Open(run)` creates such a run with an identifier unique in the workspace, the key, the document total as stated on the document, a reference to the eligible input the total and currency were taken from (an input manifest or artifact; recorded and not checked), the number of internal parts that will be posted, at least 1 and at most the limit of parts per run, and the allowed rounding, a decimal that is not negative and not above the maximum given at construction for the run's currency, which is zero for a currency that has none. The total may be negative, as for a credit note. That the total is an eligible value and not an unverified extraction is the caller's obligation (AR-03; stated limit, an exposure blocker).

Signs: an amount of a line is the amount by which the document total exceeds the internal billed sum for that cause. A tax of 10 on the document that the internal cost does not contain is a line of 10; a credit of 5 on the document that the internal cost does not contain is a line of -5.

`Post(run, part)` posts one part of the internal side: a part identifier unique in the run; an amount, which may be zero or negative; its currency, which must be the run's; the cost kind it is; and a reference to its source, unique in the run, so that one source is not counted under two part identifiers. The cost kind is one of `billed`, `effective`, `list`, `contracted` and `sale`, and only `billed` is accepted; the others are rejected with nothing written, because matching another kind to a document total is what the requirement forbids. A part beyond the declared number is rejected. Kind and source are the caller's tags and are not checked against `modules/normalize` (stated limit).

`Explain(run, line)` records one explained difference: a line identifier unique in the run; a category from a closed list (`tax`, `support`, `credit`, `commitment_purchase`, `marketplace`, `rounding`); an amount that is not zero, in the run's currency; a reference to its evidence, always; and, for every category but `rounding`, the item it is about, which has a type: `document_line`, a line of this run's document, or `internal_item`, an item of the internal side. No item is linked twice: a `document_line` is named by at most one line of its run, and an `internal_item` by at most one line among all runs of the workspace that are not voided, so that one credit does not explain two documents. A `rounding` line names no item, at most one exists in a run, and its amount is not larger in absolute value than the run's allowed rounding. Nothing ties a line to what the parts contain: a purchase that is inside a billed part and is also posted as a line is counted twice, and the caller must not do that (stated limit).

A part or a line posted again with identical content writes nothing; with other content under the same identifier it is a conflict. A wrong part or line cannot be taken back; the run is voided and another opened.

`Close(run)` is accepted when the posted parts are exactly as many as declared, and rejected otherwise with nothing written. This shows consistency with the caller's own declaration, not that nothing was sampled or left out: a caller that declares three parts of ten is not caught (stated limit). It computes, exactly:

residual = document total - sum of internal parts - sum of explanation lines

and stores the internal sum, the count and sum of lines per category, the residual and the verdict: `explained` when the residual is zero, `unexplained` otherwise. A closed run changes no more. An `unexplained` run stays so; the way on is to void it and open another under a new identifier.

`Void(run, reason)` ends an `open` or `closed` run, with a reason from a closed list (`superseded`, `wrong_figures`, `operator`). Its parts and lines stay readable as a record; its subject key and its internal items are free again. States: `open`; `closed`; final: `voided`. A run remembers whether it was closed before it was voided.

## Runs on the other bases

A run of `independent_price` or `internal_cost` has no document, no total, no parts, no lines and no verdict. `Open` creates it `closed`; `Post` and `Explain` are rejected for it and `Close` succeeds without writing. It carries:

- `independent_price`: customer, period and charge; whether the quantity needed for the billing is complete, with a reference to the evidence of the quantity; a reference to the evidence of the price; and, optionally, an internal cost with its currency and a reference to its evidence, all three or none;
- `internal_cost`: customer, period and pool; the cost with its currency; and references to the evidence of the pool and of the allocation.

## Readiness

`Readiness(run)` reads, without writing, a verdict `ready` or `not_ready` with every reason from a closed list, and separately the state of the cost, one of `determined` (a document's cost that balanced), `declared` (a figure posted with references that nothing here checked) and `undetermined`. One rule comes first: a voided run makes no positive claim; it is `not_ready` with the single reason `voided` and its cost is `undetermined`, on every basis. Otherwise:

- `provider_document`: `ready` when the run is `closed`, `explained`, not superseded, and none of its sums lies outside the profile's result bound; reasons otherwise: `not_closed`, `unexplained_residual`, `superseded`, `sum_outside_profile`. The cost is `determined` exactly when the run is `ready`, and `undetermined` otherwise;
- `independent_price`: `ready` when the quantity is complete; reason otherwise: `quantity_incomplete`. The cost is `declared` when an internal cost with evidence was given and `undetermined` otherwise, whatever the verdict. So a billing whose quantity is complete is ready while its cost is shown as not determined: the two are separate fields and neither is derived from the other;
- `internal_cost`: `ready`; the cost is `declared`.

That the two other bases are `ready` on references alone, while the evidence an independent price must have is undecided, and that the completeness of a quantity is a caller's flag, are not the conservative choices (listed below).

## Reading

`Run(workspace, run)` reads a run in every state: what was opened, state, whether it was closed, the number of parts posted, and once closed the internal sum, the count and sum per category, the residual and the verdict; whether it is superseded; the reason of a void; and times. `Parts(run, after, limit)` and `Lines(run, after, limit)` read the parts and lines of a run that is not open, in the bytewise order of their identifiers, in pages, `after` being the number already read; for an open run they are rejected, so that a page cannot shift. `Current(workspace, issuer, account, document)` reads the highest revision known for a document and the runs of that revision that are not voided, by currency, each with its state and verdict; it is not found for a document no run was ever opened for, and lists no run when all of that revision were voided.

## Repeats, error kinds, tenant boundary, time and limits

Every operation is its own key and first tests for an identical repeat, before state and limits: a repeat with the same content writes nothing and succeeds in every state reached since, and returns that state, which the caller must read. Amounts are compared as numbers, so `1.50` and `1.5` are the same content. `Close` of a run that was closed and then voided is such a repeat and returns `voided`; `Close` of a run voided while open is rejected.

A request is judged in this order: malformed, which is invalid and includes a field set that does not fit the basis, a cost kind or category that is not in its list and a line of zero; a declared number of parts above the limit, a limit error; an identifier that exists, which is a repeat when identical and a conflict otherwise, with a second run for a subject key that is taken, a part with a source already used and a second `Void` with another reason being conflicts; something unknown, not found, and so is anything of another workspace; what a stored state or rule refuses, rejected: a revision below the highest known, a part or line for a run that is not open or not of `provider_document`, a currency other than the run's, a cost kind other than `billed`, a part beyond the declared number, an item already linked, a second rounding line, a rounding amount over the allowed one or an allowed rounding over the maximum, `Close` with another number of parts, reading the parts or lines of an open run; no room, a limit error. No error repeats a string of the request.

Every operation takes the workspace and reaches only that workspace's runs. The registry authorizes nothing and must not be exposed directly (stated limit).

The clock is injected, read once per operation, and the registry keeps the highest reading it has seen. Required and positive at construction, without defaults: runs per workspace, parts per run, lines per run, the page size, the bound of lists. Also given at construction: the largest allowed rounding per currency, each a decimal that the profile accepts and that is not negative; and the decimal profile. Runs are counted for good, voided ones included, so replacing a run uses a further one and its parts are posted again; workspaces are created by first use and not limited in number; one lock serves every workspace, and every uniqueness rule uses an index, not a scan; none has an exit here (stated limits, exposure blockers).

## What is not decided here

The reconciliation scope of a document, the rounding policy and the evidence an independent unit price needs: these are the decisions the design part of W-09 asks for, they remain the owner's, and the design part stays open. Also: the values of billing_basis; whether one document may be balanced in several runs per shared pool; whether a tax or a credit may be explained without an item; whether an `unexplained` run may be completed instead of replaced; how a run is linked to the billing of a customer and period; who may void.

Assumptions made here: the three bases; that only a billed cost is balanced against a document; the six categories, taken from the requirement; that revisions are numbers ordered by size; that customer, period, charge and pool are the subject of the other bases.

Not the conservative choices, for the owner: a run is `explained`, and then `ready` with a `determined` cost, when lines of any category sum to the difference, without any check of a line against its evidence; the allowed rounding is chosen by the caller inside the maximum; the number of parts is the caller's own declaration; `independent_price` and `internal_cost` runs are `ready` on references alone.

# Allowed files

`modules/reconcile/*` (new); `docs/features/HB-192/*`.

# Verification

- Literal table tests with a test clock: every refusal of `Open` per basis, with each field of another basis and each missing field; one run per subject key on each basis, a voided run freeing it, the same key in two workspaces; revisions: a second revision superseding the first while it is open, unexplained and voided, a lower revision opened after a higher one, a revision in another currency, two currencies of one revision, and what `Current` and `Readiness` say in each case; parts with every other cost kind and an unknown one, another currency, a zero part, a negative total, a source used twice, a part beyond the declared number, a declared number over the limit, repeats and conflicts; lines of every category with both item types, an item linked twice in one category, in two, and an internal item in two runs and again after a void, a zero line, a line without evidence, rounding at, below and above the allowed amount with either sign, a second rounding line, a rounding line with an item, an allowed rounding over the maximum and for a currency without one; `Close` with fewer parts, repeated, after a void that followed closing and after a void while open; residuals of zero through cancelling signs, of the smallest unit, and at mixed scales; counts and sums per category; a sum outside the profile, printed and `not_ready`; every reason of `Readiness` per basis with the cost state beside it, the case of the acceptance among them, a complete quantity that is `ready` with an `undetermined` cost; a voided run of each basis; `Post`, `Explain` and `Close` on the bases closed at once; `Void` from each state and with another reason; parts and lines read in pages and refused while open; the tenant boundary; limits; errors that repeat nothing; a scan of the source for floating point.
- Properties with a fixed seed over random histories against a reference model written in the test that computes with 64-bit integers at one finest scale and renders the mixed-scale texts itself. The generator constructs balanced runs, whose last line is derived from the rest, and near misses of one smallest unit, and the test fails unless it drew enough of each. Read from the registry after every step, independent of the model: at most one run that is not voided per subject key; no run that is voided or superseded is `ready` or has a cost other than `undetermined`; for every closed run the residual recomputed by the test from the pages of `Parts` and `Lines` equals the stored one; no item is in two lines that count. Further: splitting a part in two keeps the verdict; a run voided and opened again with the same content gets the same verdict; the same parts and lines in any order give the same sums. No random number is drawn while ranging over a map.
- A run with 50,000 parts at mixed scales and signs whose sums equal the generator's.
- Concurrency under the race detector: the same parts posted by several goroutines and counted once; two runs opened for one subject key, of which one succeeds; closing against posting and against explaining; voiding against closing; a second revision opened against the readiness of the first.
- The package's tests run many times in a row before any mutation run; a mutation check by a script in Python with results appended to files that are never overwritten.
- `gofmt -l`, `go vet ./...`, `go test -race -count=1 ./...`.

Remaining NOT_RUN at system level after this ticket: REC01, REC02, COM02, FD07, AUD-AR-03. This ticket does not complete W-09.

# Deferred

Computing the internal side from normalized rows once the scope is decided; reading a document; a link to the evidence bundles of `modules/evidence`; shared pools and their allocation (W-12, W-14); tax and FX rules; commands, permissions and screens; persistence.

# Design review resolution

The independent design review of 2026-11-13 (read-only, nothing executed) returned FAIL without a blocker and without asking for a redesign: 5 HIGH, 11 MEDIUM, 5 LOW. It judged the exact residual, the rule that ties a document to its basis, one run per key and the separate fields for readiness and cost sound, and found that `reconciled`, `ready`, `determined` and "exactly one run counts" claimed more than the rules established. All were adopted above.

| Finding | Resolution |
| --- | --- |
| F1 a run balanced by any line | The verdict is `explained`; signs stated; counts and sums per category beside every verdict; listed as not the conservative choice |
| F2 no run that counts on the other bases | A subject key on every basis, one run that is not voided per key |
| F3 supersession depending on the state of the later run | The highest revision ever opened, never lowered; a lower revision is superseded from then on and cannot be opened afterwards |
| F4 the other bases claiming too much | The claim reworded; a third cost state, `declared`; listed |
| F5 a voided run asserting a cost | A voided run makes no positive claim |
| F6 an item linked in two runs; two namespaces | Typed items; an internal item unique among runs that are not voided; the limit about parts stated |
| F7 the declared number of parts | The claim reworded; sources unique; refusals at `Open` and `Post` |
| F8 the total as an extracted value | A reference to the input, the obligation and the baseline sentence |
| F9, F10 currencies | Parts and lines carry the run's currency; a revision supersedes across currencies |
| F11 sums outside the profile | A reason of `not_ready`; bounds and arithmetic stated |
| F12 one rounding maximum | A maximum per currency |
| F13 the scope of keys | Per workspace |
| F14 repeats against refusals | Stated, with the bases closed at once and exact field sets |
| F15 lists and mandatory fields | Enumerated |
| F16 choices and assumptions | Listed |
| F17 to F21 | Reading only what is not open, voided runs kept, `Current` defined, a lower revision refused, indexes |

# Design re-review resolution

The narrow re-review of 2026-11-13 (read-only, nothing executed) returned PASS with findings and no HIGH: sixteen of the twenty-one first findings resolved or resolved by disclosure, four partly and one disclosed without change; eleven new findings, at most MEDIUM. It said implementation may start after text fixes. The rules below settle them and are part of the decision; where they are more precise than a sentence above, they govern. The change review reads them against the code.

- N1 and N3, a superseded run. A run of a revision below the highest known for its document is superseded, and from then on `Post`, `Explain` and `Close` are rejected for it; an open one can only be voided, a closed one stays as a record. Its internal items and document lines no longer count for uniqueness: an item is linked at most once among the runs that are neither voided nor superseded. The state a repeat returns is `open`, `closed` or `voided`; whether a run is superseded is read through `Run`.
- N2, revisions. A revision is a whole number from 1 to 999999. Only an `Open` that created a run raises the highest revision of a document. A revision opened too high by mistake supersedes every real revision of that document for good, in every currency, also after its run is voided, and there is no exit (stated limit, an exposure blocker).
- N4, readiness of a document run. The sums tested against the profile's result bound are the internal sum, the sum of each category and the sum of all lines. A run can be `explained` and `not_ready` for `sum_outside_profile`, with an `undetermined` cost. An open run has the reason `not_closed`, and `superseded` when it is; no residual is computed for it. Reasons are listed in the order `not_closed`, `unexplained_residual`, `superseded`, `sum_outside_profile`.
- N5, finding a run. `Holder(workspace, subject)` reads which run that is not voided holds a subject key, for every basis, and is not found when none does. `Current` also returns the identifiers of the superseded runs of the document that are not voided, in the order of revision and identifier, up to the bound of lists with the total.
- N6, refusals. `Parts` and `Lines` of a run of the other bases are rejected. A page limit above the page size is invalid; a position beyond the end gives an empty page. A new part whose source is already used is a conflict and is judged before the state of the run, so it is a conflict also for a run that is closed. `Close` of a run voided while open is rejected. `Close` returns the state and, for a document run, the verdict.
- N7, further choices that are not the conservative ones, for the owner: the completeness of a quantity is a caller's flag; the basis is a caller's declaration, so a cost whose document run stays `unexplained` can be opened again as `internal_cost` and read `ready`; the document total is not checked to be an eligible value.
- N8. A `document_line` is named by at most one line among the runs of its document and revision that are neither voided nor superseded, whatever their currency.
- N9. The registry's arithmetic uses integers of any size. The reference model uses 64-bit integers, and its generator bounds its values so that no sum of a test, the one of 50,000 parts included, can overflow.
- N10. The cost of an `internal_cost` run and the optional cost of an `independent_price` run are decimals inside the input bound that are not negative; zero is a cost.
- N11. The reason `superseded` of `Void` is a label and is not checked against whether the run is superseded.

# Change review resolution

The independent change review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker, HIGH or MEDIUM, and 9 LOW. It found no defect in the verdict, the indexes of subject keys, revisions and items, the readiness table, the arithmetic or the tenant boundary.

| Finding | Resolution |
| --- | --- |
| 1 `Holder` accepted a currency on a subject without a document | Invalid |
| 2 identical repeats on a superseded run | Stated below |
| 3 the figures of a run returned as first written | A run's total, allowed rounding and cost are kept and returned in canonical form |
| 4 which operations move the time of last change | Stated below |
| 5 two reading claims wider than the code | Stated below |
| 6 the order of two reasons never asserted | A test with both on one run |
| 7 a test that paired reasons with runs in map order | It walks a fixed list |
| 8 literal cases missing | Rounding below the bound, an internal item in the other categories, lines in any order and of several categories, lines read in pages and beyond the end |
| 9 unasserted values in two tests | Removed, and the pages of lines are read to the end with their errors |

Stated with this review, as the code behaves:

- An identical repeat of `Post`, `Explain` or `Close` on a superseded run succeeds and writes nothing, like every repeat; only a new part or line and a first `Close` are rejected for it.
- The time of last change of a run moves when it is closed and when it is voided, not when a part or line is posted.
- The count and sum of lines per category are read through `Run`; `Close` and `Current` return the verdict alone. The runs `Current` lists for the highest revision are bounded by one per currency, not by the bound of lists.
- A check in the code was removed because it could change no result: whether a part or line may be written does not ask for the basis, since a run of another basis is never open.
