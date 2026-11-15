---
type: VerificationRecord
title: HB-244 support bundle verification
description: Actual checks run for W-36 increment 2a (modules/support), plus review outcomes and residuals.
status: final
---

# Scope

[HB-244](https://linear.app/hybrid-billing/issue/HB-244) adds `modules/support`, following [plan.md](plan.md). Support bundles are built only from typed fields that a dual-controlled schema allowlists:
- free text is never accepted, and keys that look secret cannot be allowlisted;
- identifying values become pseudonyms per class and per bundle;
- the bytes carry an opaque site handle and per-site bundle numbers;
- a human must preview before export or upload, and the digest is the SHA-256 of the exact exported bytes;
- export is local by default, and upload works only through a dual-controlled enablement to an approved, non-disabled destination;
- schema changes purge affected bundles;
- byte budgets apply per site and register-wide, and bundles expire.

Base: main `6936e6b`. Machine: mac_mini.

# Checks run (2026-11-15, this machine)

| Check | Result |
| --- | --- |
| `gofmt -l`, `go vet ./modules/support/` | clean |
| `go test -race -count=10 ./modules/support/` | pass (3.0 s), including a concurrent-use test |
| `GOOS=linux GOARCH=386 go test -c -o /dev/null ./modules/support/` | pass, after moving the byte budgets to `int64` |
| `go test -race -count=1 ./...` | every package passes |
| Statement coverage | 100.0% |
| Property test | 300 seeds with marked secret, e-mail, address, hostname, token, key, invisible-character and non-ASCII values, mixed across declared, undeclared and enum fields. No marker appears in the bytes. Rendered fields match an independent model, and the digest equals the SHA-256 of the exported bytes |

The bundle encoding known-answer test was cross-checked against an independent Python computation.

# Mutation testing

| Run | Mutants | Killed | Did not build | Survived |
| --- | --- | --- | --- | --- |
| First | 462 | 414 | 33 | 15 |
| After fixes | 465 | 425 | 33 | 7 |

The first run's survivors were closed with boundary tests. Of the 7 survivors in the second run:
- 1 (the handle number base) was then killed by `TestHandleNumbering`.
- 5 are equivalent:
  - `<` → `<=` in two sort comparators over unique keys;
  - a `[2]int64` → `[3]int64` pair array;
  - `Round(0)` → `Round(1)`;
  - the first register-budget comparison, which only decides whether to sweep before the authoritative second check.
- 1 is untested: the exact 4 MiB output boundary (`>` → `>=`), recorded as a residual.

# Reviews

- Design review (independent reviewer, read-only): round 1 FAIL (H1–H4: pattern-based removal leaks), then a redesign to allowlisted typed fields, then round 2 PASS. Its items N1–N9 were folded into the plan.
- Change review (a separate independent reviewer, read-only): PASS with L1–L9. The fixes were re-reviewed and returned PASS.

# Residuals

- Not built:
  - collectors that map host and log events to schema fields;
  - default schemas shipped with the product;
  - the upload transport, which must reach only the approved host and port.
- Authorization and integration:
  - authorizing actors against the tenant, and AR-09 support delegation;
  - linking support sites to installer sites.
- Data that schema approval alone controls:
  - numeric fields that hold secret or identifying numbers;
  - enum literals that identify someone;
  - site handles that reveal registration order.
- Testing gaps: the exact 4 MiB output boundary is untested.
- Other work: upgrade and restore (W-36 increment 2b); persistence; screens.
- Owner decisions not confirmed by the owner:
  - the closed sources and classes;
  - the refused key fragments;
  - the numeric bounds.
