---
type: VerificationRecord
title: Post-issue corrections ledger verification
description: Records what was executed and reviewed for the first increment of work item W-29, a ledger of decreases to issued originals whose effective amount always comes from issuance credits, with source-key deduplication, digest- and base-bound approval, exact credit matching, explanations for outside credits and ERP final comparisons. The record lists results, the mutation check, provisional choices and what remains unverified.
status: draft
---

# Delivered behavior

[HB-231](https://linear.app/hybrid-billing/issue/HB-231) adds the Go package `modules/corrections` from main `0ab3797`, and an additive read-only `issuance.Registry.CreditsOf`.

- The effective amount of an original is its total minus every credit issuance lists for it, including ERP-recorded credits, so no credit is hidden.
- A correction states a target below the effective amount; increases are refused (`debit_unsupported`). Requests carry a source key, a closed reason, evidence and the affected period; reprocessing the same key returns the same correction, also after it is applied, and any differing content is `key_reused`.
- One open correction per original. Approval by another human is bound to the request digest and a base made of the effective amount and the exact credit identifiers; any new credit makes it `base_changed`.
- Applying links exactly one new credit of this original whose class, region, tax, statutory and external flags, contract, parties, currency, scale and amount match.
- Credits issued outside the ledger are listed as unlinked and block requests until a human other than their issuer explains them; requested corrections can always be withdrawn and approved ones once no unlinked credit remains, so no original gets stuck.
- ERP final amounts are recorded by reference and judged against the live effective amount on every read; nothing overwrites billing.

# Commands and results

All runs were made in the worktree `/private/tmp/hb231` on 2026-11-15 with Go 1.25.7 on darwin/arm64.

| Check | Result |
| --- | --- |
| `gofmt -l modules/` | no output |
| `go vet ./modules/corrections ./modules/issuance` | pass |
| `go test -race -count=10 ./modules/corrections ./modules/issuance` | pass |
| `GOOS=linux GOARCH=386 go test -c ./modules/corrections` | builds |
| `go test -race -count=1 ./...` | 44 packages, all pass |
| Coverage | `modules/corrections` 100.0%, `modules/issuance` 100.0% |
| Mutation testing | 319 mutants of `registry.go` and `finals.go`: 294 killed, 19 did not build, 6 survived |

The six survivors were read and judged equivalent: `Round(1)` instead of `Round(0)` on the clock and on ERP observation times (both strip the monotonic reading at nanosecond precision); a `[][3]` limits pair array; `validDate` with `||` (a failed parse formats as `0001-01-01`, which never equals the refused text); `>=` after the equal case is already handled; and a `<=` sort comparator over unique credit identifiers.

Tests use the real `modules/issuance` register as the document source, with a wrapper that injects lookup failures and altered snapshots. They include a property test over 40 seeds and 150 random steps of requests, approvals, withdrawals, applications, outside credits and explanations against an independent integer-cents model; each seed ends by explaining everything, withdrawing, and requesting a new correction, and cross-checks the issued credits.

# Reviews

- Design review: four rounds by a separate read-only reviewer; FAIL, FAIL, FAIL, PASS ([plan](plan.md)).
- Change review: one round by a fresh read-only reviewer; PASS with low findings, all applied.

# Provisional choices for the owner

- Only decreases are supported; increases need a debit document that references its original.
- Explanations need one human who is not the credit's issuer; no second approver.
- ERP final records may be made by humans or systems; bounds are per workspace.

# Not verified and residuals

Debit documents, line, tax and allocation breakdowns and revalidation (AR-07), RecognitionAdjustment and the ERP recognition cadence, receivables effects of applied credits (PT-03, PT-07), shared-pool propagation, generating difference drafts automatically, delivery attempts and channel capability gating (the next W-29 increment), persistence and screens.
