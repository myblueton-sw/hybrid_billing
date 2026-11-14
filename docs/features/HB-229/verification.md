---
type: VerificationRecord
title: Disclosure policy and export projection verification
description: Records what was executed and reviewed for the first increment of work item W-27. The increment covers the field catalogue, dual-controlled combinations, policies and recipient grants, staff-only operations, keyed and unlinkable public references, recorded projections, register-rendered neutralised packages, recipient-bound expiring links, access and preview re-checks, and an attribute-free escaped preview. The record lists results, the mutation check, provisional choices and what remains unverified.
status: draft
---

# Delivered behavior

[HB-229](https://linear.app/hybrid-billing/issue/HB-229) adds the Go package `modules/disclosure` from main `ef83738`. It uses only the standard library.

- **Governance.**
  - Fields are classified once, and a locator field may only be `public` or `customer`.
  - Forbidden combinations, policies and recipient grants are each proposed by one human and confirmed by another.
  - Classes are limited per role, and combination conflicts are checked when a policy is proposed and again when it is confirmed. Confirming a new combination suspends any active policy that violates it.
  - A grant binds a role, a customer scope and one policy. Grant revisions stop older packages, and revoking a recipient is terminal.
  - An identifier that was ever a recipient, in any workspace, cannot perform staff operations.
- **Projection.**
  - Only the active, granted policy applies. Only allowlisted, classified fields and in-scope rows are kept.
  - Source locators become `ref:` HMAC references. They are bound to the recipient, stay the same across policy revisions, are refused when they collide anywhere in the workspace, and can be resolved only by staff.
  - Output is sorted and recorded under a digest that covers the workspace, recipient, grant revision, policy revision, input, fields and rows.
  - Dropped counts are kept only in the internal audit record.
- **Delivery.**
  - Packages come only from recorded projections, with policy and grant re-checks.
  - Package identifiers are random. The file is a neutralised CSV viewing copy that the register renders itself.
  - Links carry 128-bit random tokens, and only token digests are stored. Every call issues a new link.
  - `Access` checks the recipient first, then link revocation and expiry (refused at or after expiry), then policy supersession or suspension, then grant change or revocation. It returns only the file.
- **Preview.**
  - The preview is an HTML fragment with no attributes, built from `section`, `p`, `table`, `tr`, `th` and `td` only.
  - Content is escaped. Control and bidirectional characters are replaced.
  - Recipients get the same re-checks as on `Access`. Dropped counts appear only in the staff variant.

# Commands and results

All runs were made in the worktree `/private/tmp/hb229` on 2026-11-15 with Go 1.25.7 on darwin/arm64.

| Check | Result |
| --- | --- |
| `gofmt -l modules/disclosure` | no output |
| `go vet ./modules/disclosure` | pass |
| `go test -race -count=10 ./modules/disclosure/` | pass (2.8 s) |
| `GOOS=linux GOARCH=386 go test -c -o /dev/null` in the package | builds |
| `go test -race -count=1 ./...` | pass, 42 packages with tests |
| `go test -count=1 ./...` | pass, 42 packages, no failures |
| statement coverage of `modules/disclosure` | 100.0% |
| property test against an independent projection model | 200 random trials pass |
| mutation testing (`mutate229.py`, 544 mutants) | 511 killed, 29 did not build, 4 survived (all equivalent) |

The full mutation run left 45 survivors. I then added tests for:

- the canonical decimal table and neutralisation;
- the grant and policy states;
- the boundaries for the 8-field combination, 1,000 customers, 10 rows, the 64-character token and the package identifier length;
- combination conflicts held by the partner and auditor roles;
- a digest known answer.

A targeted rerun killed 41 of the 45. The 4 that remain are equivalent:

- the size of the limits pair array;
- `Round(1)` on the clock;
- an extra empty key in the canonical row sort, which changes every input digest consistently;
- `<=` in a sort over rows whose encodings are unique digests.

The 29 mutants that did not build were rejected by the compiler or `go vet`.

# Acceptance

| Criterion | Evidence |
| --- | --- |
| HB-27-V: zero leaks of customer rows, fields, file names and caches | `TestNoLeaks`, `TestRolesSeeOnlyTheirAllowlist`, `TestPropertyProjectionModel`. Nothing is cached; HTTP cache headers are a residual. |
| HB-27-V: policy change, permission revocation, link expiry | `TestAccessStops`, `TestChangeReviewRegressions`, `TestInferenceCombinations` |
| HB-27-D: inference-combination conflicts and direct-link scope | `TestInferenceCombinations`, `TestAccessStops` (recipient-bound links) |
| HB-27-I: public references instead of locators; one projection for every channel | `TestReferences`, `TestChannelParityAndOrder` (parity is a gate: delivery goes through packages accepted by `Access`) |
| DB-06: HTML preview defence | `TestPreviewEscapes` |
| HB-27-V: byte originals, CSV-to-document amount reconciliation, scale, streaming | NOT RUN; integration with `modules/exports` and streaming are residuals |

# Reviews

- **Design review:** three rounds by an independent read-only security reviewer, all folded in:
  1. FAIL: 3 high, 6 medium and 3 low findings.
  2. CONDITIONAL PASS: 1 high, 2 medium and 5 low findings.
  3. CONDITIONAL PASS: 1 medium and 3 low findings.
- **Change review:** two rounds by a separate independent security reviewer:
  1. FAIL: 1 high, 2 medium and 6 low findings, all fixed.
  2. PASS.

# Provisional choices

These are mine, not the owner's, and need owner confirmation:

- the classes, roles, combination sizes, link lifetime range, reference length and preview page size;
- dual control for policies, combinations and grants;
- any human may revoke links and recipients;
- links bound to the recipient;
- a recipient identifier is barred from staff work in every workspace;
- staff may preview stale projections.

# Not verified or not built

- A staff and channel register in `modules/auth`. Until it exists, system channels are trusted by identifier.
- Key rotation.
- Recipients combining files from older revisions.
- HTTP cache and content security headers.
- Binding the HB-227 ERP delivery gate and the `modules/exports` customer artifact to disclosure packages.
- PDF, mail and portal delivery.
- Streaming and scale.
- Persistence and screens.
