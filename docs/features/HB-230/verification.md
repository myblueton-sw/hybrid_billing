---
type: VerificationRecord
title: Self-issuance and internal documents verification
description: Records what was executed and reviewed for the first increment of work item W-28. The increment covers dual-controlled document kinds and number series, separated ERP, self-issued and internal-settlement numbering, reservations and recorded cancellations without reuse, digest-bound approval that records protected references and term coverage, issue-time snapshots with calendar due dates from the contract binding, partial credits, a summary linked to a verified detail attachment, and safety-checked rendering with recovery, digest-checked downloads and an escaped preview. The record lists results, the mutation check, provisional choices and what remains unverified.
status: draft
---

# Delivered behavior

[HB-230](https://linear.app/hybrid-billing/issue/HB-230) adds the Go package `modules/issuance` from main `0030cdc`, and an additive `Cover` method with a `Coverage` type in `modules/receivables`.

- **Kinds and tax review.** Kinds (`self_issued`, `internal_settlement`, `erp_issued`; region, tax regime, statutory flag, credit flag, `Sequential`, zone, languages `en`/`ko`/`ar`) are defined by one human and approved by another. Statutory kinds issue only after a human other than the definer records tax-review evidence; snapshots carry the statutory flag, evidence and reviewer. This is evidence of review, not legal certification.
- **Numbering.** Documents are identified by draft. ERP kinds never get a number here; `RecordExternal` records the ERP number (unique per kind) and issue instant. Each self-issued or internal kind has one dual-controlled series; prefixes are unique and end in a non-digit, so two series cannot produce the same text. Reservations, recorded cancellations (closed reason set) and issues share one counter under one mutex; values are never reused, every value is listed with reserver, last actor, time, state and reason, a refused issue never takes a value, and `Sequential` kinds refuse reservations.
- **Drafts.** Canonical positive decimals at a per-draft scale, totals equal to the exact line sum, at most 200 summary lines with text up to 4,000 bytes (control and bidirectional override characters refused), an optional detail attachment whose digest, rows, total, currency and scale must match the injected attachment register, and partial credits limited to the original's total across all issued and ERP-recorded credits (checked at submit and at issue).
- **Approval.** A human other than the submitter approves a revision digest; approval records the protected references (`entity:`/`customer:`/`kind:` subjects in `modules/protected`) and the contract term coverage. `Issue` and `RecordExternal` refuse `reference_changed` if either differs at the issue instant; the same approver can refresh.
- **Issue.** One critical section fixes the snapshot (draft revision and digest, people, issuer kind, number, instant, kind facts, references, due facts) under a digest. Payable documents take the due date from the contract's approved binding (`receivables.Cover`), day 0 being the issue date in the contract zone, and refuse a coverage whose dates disagree; internal and credit documents record `not_applicable` with the kind zone.
- **Rendering.** An injected renderer receives a deep copy of the snapshot outside the mutex, under an attempt token with a lease, panic recovery and dropping of superseded completions. The output is copied, then gated: PDF 1.x or 2.x header; a single classic xref table verified against every parsed object; one trailer with `/Size` and `/Root` to a catalog with `/Pages`; every reference resolving to a listed object; `startxref` followed only by `%%EOF`; stream bodies skipped by one direct `/Length` with `endstream` refused inside; a filter allow-list; and refusal of actions, annotations, forms, name trees, media, embedded and external files, encryption, object and cross-reference streams and `#` name escapes. The first accepted output fixes the file digest; later renders must reproduce it. Downloads re-check the digest and are recorded with actor, kind and time; the preview is escaped attribute-free HTML in `bdi` elements and labelled as a viewing copy.

# Commands and results

All runs were made in the worktree `/private/tmp/hb230` on 2026-11-15 with Go 1.25.7 on darwin/arm64.

| Check | Result |
| --- | --- |
| `gofmt -l modules/issuance modules/receivables` | no output |
| `go vet ./modules/issuance ./modules/receivables` | pass |
| `go test -race -count=10 ./modules/issuance ./modules/receivables` | pass |
| `GOOS=linux GOARCH=386 go test -c` for both packages | builds |
| `go test -race -count=1 ./...` | 43 packages, all pass |
| `go test -cover ./modules/issuance` | 100.0% of statements |
| `go test -cover ./modules/receivables` | 99.3%, unchanged from main; `Cover` fully covered |
| Mutation testing | 1,308 mutants of `modules/issuance` (`registry.go`, `drafts.go`, `issue.go`, `render.go`, `pdf.go`) and `modules/receivables/terms.go`: 1,205 killed, 51 did not build, 52 survived (see below) |

Tests include literal gate cases for every refusal, a property test of numbering against an independent model over 40 seeds, a concurrency test of 64 parallel issues, a property test of credit room against an integer model over 30 seeds, due-date tables (zone day boundary, month end, leap day, DST, net 0 and 366), an integration test with real `receivables` terms, PDF gate cases (every refused name, hidden xref and object tricks, filters, lengths, every truncated prefix and single-byte corruption without a panic, exactly the 20 MiB limit), and pinned draft and snapshot digests; the draft digest was recomputed independently in Python from the encoding rule and matches.

# Mutation survivors

A first full run left 116 survivors; tests were added (exact-offset PDF cases, every truncated prefix and single-byte corruption, boundaries, credit mismatches, pinned digests) and a full rerun left 66, of which a targeted rerun killed 14 more. The remaining 52 were read and judged as follows; none was reproduced as a behaviour difference:

- Equivalent, values ignored on refusal (15): `begin` returning token 1 with an error; `key`, `value`, `target`, `integer`, `number` and `length` returning 1 or starting from 1 on a path that also returns `false` or is overwritten.
- Equivalent, no effect on observable state (6): `Round(1)` instead of `Round(0)` on clocks and ERP instants (both strip the monotonic reading at nanosecond precision); `[][3]` instead of `[][2]` pair arrays; initial `xref`/`startxref` sentinels that are always overwritten before use.
- Equivalent, redundant defensive guards (28): bounds such as `at+2`, `at+4` or `size >=` that cannot differ because every dictionary ends with `>>` and every file ends with `%%EOF`; empty-input and empty-number checks already refused by the header or by later structure checks; xref count and remaining-length bounds (20-byte entries are always followed by `trailer`); `begin`, `table` and `verify` conditions whose other branches refuse the same inputs; zone names of 128 bytes, which no loader resolves; `key` stopping at any depth-0 lexeme, since the first one is always the closing `>>`.
- Outside this change (3): `modules/receivables/terms.go` lines 86 (the `9999-12-31` sentinel) and 187 (`lastIssue`) predate HB-230; only `Cover` was added there and all of its mutants were killed.

# Reviews

- Design review: four rounds by a separate read-only reviewer; FAIL, FAIL, FAIL, PASS. Findings and resolutions are in [the plan](plan.md).
- Change review: three rounds by a fresh read-only reviewer; FAIL (3 major PDF gate bypasses), FAIL (shared download records), PASS. The round-3 minor follow-up was applied.

# Provisional choices for the owner

These were chosen by the implementer and are not confirmed:

- Limits: 200 summary lines, 4,000-byte line text, 10,000,000 detail rows, net days 0 to 366, series width 1 to 12, 20 MiB files, render lease 1 minute to 1 hour, per-document attempt and download caps (downloads are refused once the cap is reached).
- The PDF gate refuses document outlines (`/Prev`), all annotations and links, `/F` keys and filters other than Flate and DCT.
- Statutory credit kinds need their own tax review; internal kinds cannot be statutory.
- Reads (`KindOf`, `DraftOf`, `Numbers`, `DocumentOf`) take no actor, as in sibling registers.
- Protected subjects use the `entity:`, `customer:` and `kind:` prefixes; `modules/protected` registrations must follow this.

# Not verified and residuals

- No real PDF renderer exists; the gate was tested with constructed files only, and how real readers treat files the gate accepts (including repair mode) is inferred, not tested. Renderer sandboxing and cancellation are residuals.
- No legal or tax validity of any document kind is claimed; tax amounts are not checked against rates.
- The `modules/exports` adapter for detail facts, the HB-28-V row trace beyond the attachment's own, currency scale profiles, yearly series resets, business-day terms, ERP due-date reconciliation, handing documents to `modules/receivables`, `modules/disclosure` and `modules/erpledger`, correction and replacement (W-29), delivery, persistence, an append-only download audit and screens are not built.
