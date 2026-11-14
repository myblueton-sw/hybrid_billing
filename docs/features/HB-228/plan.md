---
type: ImplementationPlan
title: Deterministic settlement and customer XLSX and CSV handoff artifacts with neutralised payloads, machine-value manifests, confirmed bytes and amount-safe result registration
description: Defines the first increment of work item W-26, a register that turns an approved submission specification into deterministic standard settlement and customer workbooks and CSV files, keeps long identifiers and decimals as text, neutralises formula, control and markup payloads in the viewing copy while a separately labelled machine manifest keeps raw values, records the serialisation contract and versions apart from volatile metadata, returns the same bytes for confirmed artifacts, compares regenerations on a normalised data digest, and registers ERP results that carry only references and states while refusing changed amounts.
status: draft
---

# Scope and ownership

[HB-228](https://linear.app/hybrid-billing/issue/HB-228) starts work item W-26 (requirement HB-26; detail contracts DB-01, DB-06, DB-07; acceptance AUD-DB-01, -06, -07 and HB-26-V). Base: main `3b80cab`. Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviews the plan and the change, because exported amounts and result registration touch billing data. Machine: mac_mini.

Out of scope, recorded as residuals: ERP-specific upload templates (they stay unverified until proven per product and version, HB-227), PDF and HTML previews and their defences (HB-27), streaming generation and file storage, opening the files in real spreadsheet products (the AUD-DB-06 real-spreadsheet check stays unverified), reading XLSX uploads, the release gate of AUD-DB-01, persistence and screens I03, I04 and P05.

# Observed baseline

- HB-26-D: standard internal settlement XLSX and customer files are provided; ERP upload formats are verified per product and version; a download is not external reflection.
- HB-26-I: header, detail, tax, reconciliation and result sheets from the approved specification; fixed values and long identifiers kept as text; result collection takes only ERP references and states and blocks amount overwrites; API and file use the same submission and row identifiers.
- DB-06 / AUD-DB-06: non-executable output; formula and HTML injection defended; no macros or external data connections; machine raw values distinguished from the safe viewing copy; long IDs, decimals and machine values preserved.
- DB-07 / AUD-DB-07: inputs, publication policy, generator and schema versions, stable sort key, date, time zone, scale, encoding, separator and sign contracts fixed; volatile metadata separate; confirmed artifacts re-downloaded as identical bytes; regeneration compared on a normalised data result; chunk, worker or retry order never changes the data; a generator change is a new version and never silently replaces a confirmed file.
- HB-26-V: more than 100,000 rows, interrupted and partial handoff, formula strings, long IDs and decimals, no leak of internal sheets and refusal of changed amounts.
- `modules/erpledger` (HB-226) holds submissions with keys and line identifiers; its result-file observations take references and states. No module produces files.

# Decision

A new package `modules/exports` uses only the standard library (`archive/zip`, `encoding/csv`, `encoding/xml`, `crypto/sha256`). It has the sibling conventions: injected clock, one mutex, workspace isolation, `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (`Refusal{Reason}`), `ErrLimit`, check order form, lookup, repeat, rule, room; assistants are refused; generating and registering results may be done by a human or a system job, confirming needs a human.

`NewRegistry(Config{Generator, Schema, Numbers}, Limits, clock)`, where `Numbers` is the ledger's `decimal.Profile`, fixes the generator and schema versions of this build; every artifact records them.

## Specification and request

`Generate(workspace, Request, actor)` creates an artifact from an approved specification:

- request: artifact identifier (at most 100 bytes, so observation identifiers stay within 128 bytes), kind (`settlement` or `customer`), time zone (`UTC` or `+HH:MM`/`-HH:MM`; recorded in the contract, it governs no value in this increment and is excluded from the data digest);
- specification: ledger connection, submission key, approval digest (64 hex), customer, period (`YYYY-MM`), currency, amount scale (0 to 8) and lines (row identifier, source line, product, tax code, quantity, net, tax). The connection, submission key, customer and row identifiers use the ledger identifier form (letters, digits, `.`, `_`, `:`, `-`, at most 128 bytes); source, product and tax code are free text of 1 to 512 bytes of valid UTF-8; row identifiers are unique.
- Quantity, net and tax are parsed with the same injected `decimal.Profile` as the ledger (`decimal.Parse`), must have at most `scale` fraction digits, and are written padded to exactly `scale` fraction digits (negative zero becomes zero). Tax and reconciliation totals are exact big-integer sums at that scale, checked against the profile's result bounds. The profile identifier and revision are recorded in the contract.

Lines are sorted by the stable key (row identifier) before anything is written, so input order, chunking and retries never change the result. A `settlement` artifact belongs to one ledger submission and holds at most 10,000 lines, the ledger's limit; a `customer` artifact may hold up to the configured line limit (at most 200,000 lines while generation is in memory).

## Files and columns

Per-kind allowlists define every sheet, header field and column:

- `settlement`: workbook sheets `header` (submission, connection, customer, period, currency, scale, approval, generator, schema), `detail` (row, source, product, tax_code, quantity, net, tax), `tax` (tax_code, net, tax), `reconciliation` (lines, net, tax, quantity) and `result` (row, submission, erp_reference, erp_state, net, tax, quantity); a machine CSV of the detail; a viewing CSV of the detail.
- `customer`: workbook sheets `header` (customer, period, currency, scale, generator, schema), `detail` (row, product, tax_code, quantity, net, tax) and `tax`; a viewing CSV of the detail. No submission key, connection, approval digest, source lineage, reconciliation or result sheet.
- Detail sheets split into `detail-2`, `detail-3` and so on at 1,048,575 data rows (an internal threshold that tests may lower).

Serialisation contract recorded on the artifact: UTF-8, comma separator, `\n` line ends, decimal point, leading minus sign, padded scale, `YYYY-MM` periods, no generation time in any file, viewing CSVs start with a UTF-8 byte order mark and are marked not precision-safe, machine CSVs have no byte order mark.

- Workbook cells are inline strings with the text number format, so no cell is a formula or a number that loses precision; cell values are the raw values with only XML-invalid and control characters replaced by U+FFFD. A value that would be risky in a spreadsheet gets the `quotePrefix` style instead of a changed value. Workbooks contain no macros, external links, data connections, defined names, formulas, hidden sheets, document properties or cached values.
- A value is risky unless it fully matches the decimal grammar; a risky value is one whose first character, or first character after leading spaces, is `=`, `+`, `-`, `@`, tab, carriage return, `'`, or a fullwidth `＝`, `＋`, `－`, `＠`.
- Viewing CSVs prefix every risky value with exactly one apostrophe; reading back removes exactly one leading apostrophe when the rest is risky, which reverses the prefix for every value.
- The machine CSV keeps raw values, starts with `#machine-values,<schema>` and is named `*.machine.csv`.
- Zip entries are stored without compression in a fixed order with a fixed timestamp, so identical data and versions give identical bytes independent of the compressor.
- HTML and PDF previews and their defences are HB-27; this increment only produces workbooks and CSV.

## Artifact and digests

The artifact view holds the request, the files (name, media type, bytes, byte digest), the contract, the generator and schema versions, the data digest, the inputs digest and the volatile metadata (actor, generation time) kept apart.

- The data digest (label `export_data_v1`, length-prefixed fields in a fixed order) covers the kind, the contract without the time zone and the raw values of every sheet except the generator and schema rows. Two different specifications never share a data digest because raw values are digested, not viewing values.
- The inputs digest (label `export_inputs_v1`) covers the identifier, kind, time zone and every specification field with the lines in sorted order.
- An identical repeat of `Generate` by the same actor returns the artifact; anything else under the identifier is `ErrConflict`.
- `Confirm(workspace, id, actor)` freezes an artifact by a human other than the generating actor (`self_approval`). A confirmed artifact is never regenerated under its identifier.
- `Download(workspace, id, name, actor)` returns the stored bytes of a confirmed artifact (`not_confirmed` otherwise); assistants are refused.
- `Compare(workspace, id, Request, actor)` (assistants refused) regenerates the given request in memory and reports whether its data digest equals the stored one and whether the generator or schema version differs; it never changes the stored artifact.

## Result registration

`RegisterResults(workspace, id, csv, actor)` reads a filled result sheet saved as CSV for a confirmed `settlement` artifact. A leading byte order mark and CRLF line ends are accepted; non-UTF-8 input is `not_utf8`; a header other than `row,submission,erp_reference,erp_state,net,tax,quantity` (for example a `;` separator) is `wrong_format`; files over 32 MiB or with more rows than the artifact are `ErrLimit`.

- Every row must be a row of the artifact and appear at most once (`unknown_row`, `ErrInvalid` for duplicates); the submission must equal the artifact's key (`submission_mismatch`); net, tax and quantity must equal the exported padded text exactly (`amount_changed`). Amounts are never taken from the file.
- Rows with an empty reference and state are not registered. A reference is an identifier and is set once; a different non-empty reference for a row is `ErrConflict`. States move forward only: none, then `draft`, then `posted` or `rejected`; `posted` may become `rejected`; anything else is `ErrConflict`. `draft` and `posted` need a reference.
- Within one registration all stated rows carry the same state, and across all registrations of an artifact every stated row carries the same reference (`mixed_results`), because a ledger submission has one external reference (an empty reference on a `rejected` row is exempt); split documents are reconciled through the ledger's document links instead.
- A registration that changes no row returns the previous hand-off without a new revision, so a retried upload creates no new observation.
- The hand-off state of an artifact moves forward like a row (none, `draft`, `posted`, `rejected`); a registration whose state would move it backward is `ErrConflict`. A `rejected` registration reports the whole submission as rejected, because the ledger holds one state per submission; partial row rejection is reconciled through document links.
- Row, submission and reference values are identifiers that cannot contain an apostrophe, so removing one leading apostrophe from a risky value is safe for both viewing CSVs and spreadsheet-saved CSVs.

A registration returns the accepted rows and a hand-off: connection, submission, reference, state and a revision equal to the artifact's registration count, with an observation identifier `<artifact>:<revision>`. The integration layer passes it to the ledger as a `result_file` observation after the submission has been marked sent through the file transport. File-transport submissions have result files as their only observation source in this flow, so the registration count is a valid ledger revision; mixing sources needs the integration layer to derive revisions from the ledger (residual). Binding the approval digest and line identifiers to the ledger submission is also an integration-layer check (residual).

## Limits

`Limits{Workspaces, Artifacts, Lines per artifact up to 200,000}`; settlement artifacts are further capped at 10,000 lines; artifacts are held in memory in this increment, and streaming generation for larger exports is a residual.

# Tests

- AUD-DB-06: formula (including `-2+3+cmd|' /C calc'!A0`, fullwidth and leading-space variants), HTML, control-character and apostrophe payloads in every free-text field; a round-trip property of the viewing prefix; long identifiers and 8-digit decimals preserved; the workbook parts are checked for the absence of formulas, macros, external links and connections; the machine CSV keeps raw values.
- AUD-DB-07: shuffled line order, repeated generation and a different generator version give the same data digest; the same versions give the same bytes; confirmed artifacts are never replaced and downloads return identical bytes.
- AUD-DB-01 within this module: a settlement artifact and result registration complete a first handoff without any API; the result feeds `modules/erpledger` as `result_file` observations in a test.
- Result registration: refusals, amount changes, forward transitions, mixed results, BOM and CRLF input; the hand-off drives `modules/erpledger` as a `result_file` observation after a file submission.
- Customer allowlist: every zip part, sheet, header field and CSV column compared exactly; detail sheets split at a lowered threshold.
- Forms, limits, repeats, isolation, a property test of the data digest and the CSV round trip against an independent model, race, 386 build, gofmt, vet, coverage and mutation testing.

# Provisional choices

- Sheet names, columns, the neutralisation rule, the apostrophe prefix, inline strings for every cell, the time zone forms and the result states.
- Row identifier as the stable sort key; row identifiers are shown to customers.
- Risk classification uses an 18-digit decimal grammar independent of the profile.
- Result registration through CSV of the result sheet; reading XLSX uploads is a later increment.

# Review history

- Design review 1 (independent reviewer, read-only): CONDITIONAL PASS with 4 high, 6 medium and 4 low findings. Folded in: risky values defined by the full decimal grammar with leading-space and fullwidth variants; a reversible apostrophe prefix in CSV and `quotePrefix` in workbooks; the ledger hand-off (one reference and state per registration, revision, observation identifier, connection in the specification, file-transport precondition); forward result transitions and set-once references; the decimal grammar and exact totals; raw-value, length-prefixed digests and a defined inputs digest; stored zip entries; per-kind allowlists and no hidden sheets or properties; `Download` with an actor and confirmed artifacts only, and a confirmer other than the generator; tolerant but strict CSV reading with limits and a viewing BOM; viewing CSVs marked not precision-safe; an internal split threshold; the time zone excluded from data; HTML previews deferred.
- Design review 2 (same reviewer): CONDITIONAL PASS; all high findings resolved, 3 medium and 5 low new findings folded in: one reference across registrations, retries without new revisions, settlement artifacts capped at the ledger's 10,000 lines, file results as the only observation source and short artifact identifiers, the ledger's decimal profile, identifier-only result fields, the approval binding as a residual, `Compare` with an actor, and a 200,000-line memory bound.
- Design review 3 (same reviewer): PASS with three low notes, folded in: identifier form stated for identifier fields and payload tests limited to free text, the decimal profile in the constructor, and the rejected-row exemption from the shared reference.

# Change review resolution

Change review 1 (a separate independent reviewer, read-only): FAIL with 3 medium and 5 low findings. All were addressed:

1. A rejected row inherits the shared reference before comparison, so a retried rejection returns the previous hand-off; tested.
2. The hand-off state moves forward only; a backward registration is `ErrConflict`, and the plan states that a rejection covers the whole submission; tested.
3. Line limits are checked before validation and building in both `Generate` and `Compare`; tested.
4. Per-tax-code net and tax sums are checked against the profile's result bounds; tested.
5. Empty references on rejected rows are exempt within a registration as well, and the hand-off takes the first non-empty reference.
6. The result header must have exactly seven fields.
7. Row and submission values in results must be identifiers before they reach an error.
8. Risk classification uses a fixed 18-digit decimal grammar independent of the profile; a longer canonical number is classed as risky, which is reversible and safe. This is recorded as a provisional choice.

Added tests also cover the time zone staying out of the data digest and stored zip entries with the fixed timestamp.

Change review 2 (same reviewer): PASS; all findings resolved. Accepted behaviour: re-uploading an older file whose rejected row predates the shared reference is `ErrConflict`, like any other stale file.
