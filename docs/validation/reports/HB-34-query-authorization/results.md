---
type: VerificationReport
title: HB-34 internal query authorization verification
description: Records independent review and synthetic race-enabled Go verification of scoped query authorization without claiming external authentication or production disclosure integration.
---

# Scope and provenance

[HB-34](https://linear.app/hybrid-billing/issue/HB-34/implement-authenticated-customer-query-and-export-boundaries); machine mac_mini; accountable owner Seung Woo Park. Root implemented the Go module and tests. The separate read-only `query_auth_review` agent performed design review, actual source/test review and an independent race-test run. Base main: `341f93475cf4cdbcaa13b1cd54822f9892b01bfd`. Source and test version: the commit containing this report on `feat/HB-34-query-authorization`.

The [accepted design](../../../work/HB-34-customer-access/query-authorization.md) covers a process-local current-authority store and internal query service. Test data and identities are synthetic. No database, customer data, real account, public listener or authentication provider was used. No existing Python lifecycle code changed.

# Executed checks

Observed toolchain: `go version go1.25.7 darwin/arm64`. The module requires Go 1.25.0 and uses only the standard library. No third-party Go dependency was added.

| Check | Observed result |
| --- | --- |
| Root `go test -race -count=1 ./...` | PASS; package completed in 1.993s; no failing test or race diagnostic. |
| Independent reviewer `go test -race -count=1 ./...` | PASS; package completed in 1.264s; no failing test or race diagnostic. |
| `go vet ./...` | PASS after authorized retry for sandbox-denied Go build-cache access. The initial attempt did not complete and is not a pass. |
| Formatting and source structure | `gofmt` applied; seven Go source/test files, each below 1,500 physical lines; no explanatory source comments added. |
| Whitespace and documentation | `git diff --check` and frontmatter/relative-link inspection; final staged scope inspected before commit. |

The three test files contain 13 top-level tests plus table-driven cases. Tests enter through `Service.Query` with an installed synthetic verifier and reader; they do not just invoke a detached permission predicate.

| Scenario | Evidence and result |
| --- | --- |
| Allowed read and ownership | Exact long decimal string preserved; reader selection mutation cannot change final selection; returned maps do not alias source maps; trusted config deep-copies grants and catalog. PASS. |
| Scope and action denial | Unknown subject, installation/epoch mismatch, foreign/mixed scope, wrong company, tenant reassignment, inactive records and export-only grant deny before reader execution. PASS. |
| Field restrictions | Unknown/ungranted/duplicate fields and cross-scope field composition deny; same-scope grant union succeeds. PASS. |
| Reader failures | Foreign/extra/missing/substituted fields, duplicate/empty row identities, excess rows and partial rows with reader error return no data. Raw reader/verifier diagnostic text is suppressed. PASS. |
| Revocation and time | Paused reads are rejected after grant removal, revoke/regrant, epoch rotation, subject disablement, company deactivate/reactivate, tenant move/restore, equivalent relevant updates, exact-deadline expiry and outage/recovery. PASS. |
| Unrelated changes | Unrelated company/tenant/subject writes and rejected stale updates do not invalidate an authorized paused read. PASS. |
| Credential resurrection | Epoch 1→2→1 rejected; unchanged-epoch reactivation rejected; stale administrative snapshot cannot restore removed grants. Invalid writes preserve data/version. PASS. |
| Concurrency and cancellation | 64 concurrent A/B queries preserve scope and values; explicit MSP grant accesses both scopes; cancellation before/during read returns no rows. PASS under race detector. |

# Independent review

Design initially FAIL: credential epochs could theoretically be lowered after revocation. The design was corrected before source implementation to require monotonic positive epochs, a higher epoch on reactivation and expected-version writes. The separate reviewer reread it and returned PASS.

Actual-change review and scoped runtime QA: PASS, no blocking finding. The reviewer inspected all four implementation files and three test files and independently executed the race suite. Review confirms only the implemented internal boundary. It does not approve a future verifier, customer handler, field profile or data adapter.

# Remaining boundaries

HB-34 remains open. Authentication selection/integration is deferred by the owner. Actual customer entry points, production disclosure profiles, fixed data revision/period/currency/basis semantics, FOCUS/source integration, export/download, durable/distributed authority and rollback-safe restoration are not implemented by this increment. I18/I19 external end-to-end certification remains NOT_RUN.

The trusted verifier must establish identity independently of query input; the reader must honor the selection and transfer ownership of returned values. This boundary verifies field names and scopes, not the economic meaning or safe content of string values. Callers cannot mutate input concurrently during copying. The server clock must be concurrency-safe. Reader runtime and string payload bytes require future adapter budgets; only row/scope/field counts are bounded here. Revocation linearizes at internal result release under the authority lock and cannot recall already released bytes.
