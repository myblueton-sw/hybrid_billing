---
type: VerificationRecord
title: Excel manual handoff artifacts verification
description: Records what was executed and reviewed for the first increment of work item W-26, the deterministic settlement and customer XLSX and CSV artifacts with neutralised payloads, raw-value machine manifests, recorded serialisation contracts, confirmed bytes and amount-safe result registration with a ledger hand-off, with results, the mutation check, provisional choices and what remains unverified, including that no real spreadsheet product opened the files.
status: draft
---

# Delivered behavior

[HB-228](https://linear.app/hybrid-billing/issue/HB-228) adds the Go package `modules/exports` from main `3b80cab`. Its non-test sources use the standard library and the primitive `decimal` only; its tests also import `modules/erpledger`.

- **Generation.** An approved specification (connection, submission, approval digest, customer, period, currency, scale, lines) produces a settlement artifact or a customer artifact.
  - Amounts are parsed with the ledger's decimal profile, padded to the scale and totalled exactly within the profile's bounds.
  - Lines are sorted by row identifier, so input order, chunking and retries give the same data.
- **Files and policy.**
  - Settlement artifacts hold a workbook with the sheets header, detail, tax, reconciliation and result, plus a viewing CSV and a machine CSV.
  - Customer artifacts hold a workbook (header, detail, tax) and a viewing CSV, without submission, connection, approval, source, reconciliation or result data.
  - Detail sheets split at 1,048,575 rows.
- **Safety.**
  - Every workbook cell is an inline string in text format. Risky values get a `quotePrefix` style.
  - No workbook contains formulas, macros, links, connections, defined names, hidden sheets or properties.
  - Control and XML-invalid characters become U+FFFD in viewing copies.
  - Viewing CSVs prefix risky values with one reversible apostrophe. A value is risky when it is not a canonical decimal and starts with `=`, `+`, `-`, `@`, a tab, a carriage return, an apostrophe or a fullwidth variant, even after leading spaces.
  - Machine CSVs keep raw values under a `#machine-values` marker.
- **Determinism.**
  - Zip entries are stored uncompressed, in a fixed order, with a fixed timestamp.
  - The data digest covers raw values and excludes the time zone, the generator, the schema and volatile metadata. The inputs digest covers the sorted request.
  - Confirmed artifacts cannot be replaced. Downloads return the stored bytes. `Compare` reports whether the data, generator or schema changed.
- **Result registration.**
  - Accepted only for confirmed settlement artifacts.
  - The importer handles a byte order mark, CRLF line endings, UTF-8 and format checks, and a 32 MiB size limit.
  - Amounts must equal the exported text exactly.
  - Row states and the artifact's hand-off state move forward only, and every row uses one shared reference. A retry returns the previous hand-off.
  - The hand-off carries the observation identifier, connection, submission, reference, state and revision.

# Commands and results

All runs were made in the worktree `/private/tmp/hb228` on 2026-11-15 with Go 1.25.7 on darwin/arm64.

| Check | Result |
| --- | --- |
| `gofmt -l modules/exports` | no output |
| `go vet ./modules/exports` | pass |
| `go test -race -count=10 ./modules/exports/` | pass (86.2 s) |
| `GOOS=linux GOARCH=386 go test -c -o /dev/null` in the package | builds |
| `go test -race -count=1 ./...` | pass, 41 packages with tests |
| `go test -count=1 ./...` | pass, 41 packages |
| statement coverage of `modules/exports` | 100.0% |
| golden digests of the workbook, viewing CSV and machine CSV for a fixed input | match |
| 120,001-row customer artifact | generated, all rows present, tax total exact |
| mutation testing (`mutate228.py`, 553 mutants) | 493 killed, 46 did not build, 14 survived (all equivalent) |

One earlier non-race repository run counted 40 passing packages instead of 41. It ran while the mutation run was loading the machine, and no failing package was captured. The immediate rerun passed for all 41 packages. I record this as an unexplained count difference, not as a pass.

The first mutation runs left 124 survivors, mostly in XML literals and validators. I added three groups of tests:

- golden digests and a well-formedness parse of every workbook part;
- exact tax and reconciliation sheet content, and the exact contract;
- boundary tables for identifiers, digests, months, currencies, zones, decimals, cleaning and risk.

I also removed two unreachable conditions: the surrogate range in cleaning, and the row-level reference comparison that the shared-reference check already covers.

The 14 remaining survivors are equivalent:

- the input length guard (`64`/`65`, `>`/`>=`);
- extra short month keys, which the length check already refuses;
- `Round(1)` on the clock;
- the hash length-prefix array size;
- `<=` in a sort over unique row identifiers;
- `>=` in the last-chunk clamp;
- `-2` instead of `-1` as the CSV field-count setting (any negative value disables the check);
- `&&` in the header check, where a read error always yields an empty header that fails the name comparison.

The 46 mutants that did not build were rejected by the compiler or `go vet`.

# Acceptance

| Criterion | Evidence |
| --- | --- |
| AUD-DB-06: formula, HTML, control and DDE payloads; long IDs and decimals preserved; no macros or external links | `TestPayloadsNeutralisedAndValuesPreserved`, `TestPropertyNeutralRoundTripAndDigest`, `TestValidators`, `TestGoldenBytes`. The check in a real spreadsheet product is NOT RUN. |
| AUD-DB-07: order, chunk and retry independence; generator versions separated; confirmed files never replaced | `TestDeterministicAndConfirmedBytes`, `TestPropertyNeutralRoundTripAndDigest`, `TestChangeReviewRegressions` |
| AUD-DB-01: first handoff by file and result registration without an API | `TestHandoffDrivesLedger`: settlement artifact, then result CSV, then ledger `result_file` observation, then posted. The release gate is not built. |
| HB-26-I: results never overwrite amounts | `TestResultRegistrationRefusesAmountChanges`, `TestResultTransitions` |
| HB-26-V: more than 100,000 rows; formula strings; long IDs and decimals; no internal leak; changed amounts refused | `TestOverHundredThousandRows` (customer, in memory), the payload tests, `TestCustomerAllowlist`, the amount tests. Interrupted and partial handoff at scale, and streaming, are NOT RUN. |

# Reviews

- **Design review.** Three rounds by an independent read-only reviewer:
  1. CONDITIONAL PASS: 4 high, 6 medium and 4 low findings.
  2. CONDITIONAL PASS: 3 medium and 5 low findings.
  3. PASS, with 3 low notes.
- **Change review.** Two rounds by a separate independent read-only reviewer:
  1. FAIL: 3 medium and 5 low findings, all fixed.
  2. PASS. One informational behaviour was accepted and recorded in the plan.

# Provisional choices

These are mine, not the owner's, and need owner confirmation:

- the sheet names and columns, the neutralisation rule, the apostrophe and `quotePrefix` handling, and inline strings for every cell;
- the time-zone forms, the result states, and the rule that a rejection covers the whole submission;
- the row identifier as the sort key, with row identifiers shown to customers;
- the 18-digit risk grammar;
- the settlement cap of 10,000 lines and the in-memory cap of 200,000 lines;
- CSV result registration.

# Not verified or not built

- Opening the files in Excel, LibreOffice or other products: precision, safety and possible repair prompts from the minimal styles.
- ERP-specific upload templates (HB-227), PDF and HTML previews (HB-27), and streaming generation and storage.
- Reading XLSX uploads, the AUD-DB-01 release gate, and persistence.
- Screens I03, I04 and P05.
- Binding the approval digest, line identifiers and observation revisions to the ledger in the integration layer.
