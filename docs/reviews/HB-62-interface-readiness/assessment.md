---
type: ArchitectureAssessment
title: Collector interface readiness and frontend backend design handoff
description: Separates implemented local collection and usage bridges from unimplemented production collectors and frontend APIs, and defines the next design scope.
status: draft
---

# Collector interface readiness

Ticket: [HB-62](https://linear.app/hybrid-billing/issue/HB-62/assess-collector-interface-readiness-and-define-frontend-backend). Date: 2026-10-31. Accountable owner: Seung Woo Park. Root owns this report; machine: mac_mini. Reviewed baseline: `b48b23ce5b880ac4307f1436ede4fd1d7a726bb0` (`origin/main` when fetched). Independent read-only reviewer: interface_audit. HB-61's visual research and terminology additions remain on separate open PR57; this assessment does not merge them.

## Answer

The intermediate layer exists **partially**. There is real bounded local FOCUS intake, validation, confirmation, publication, query and lifecycle code, plus a Go-to-Python customer-usage projection bridge. This is not a complete production collector integration layer or a frontend-ready backend API.

The owner's statement that collection is configured is consistent with having a local data foundation and onboarding designs. Repository evidence does not establish that every CSP has a working live collector. A collector running outside this repository was not inspected and is UNVERIFIED, not declared nonexistent.

## Evidence by boundary

| Boundary | Current evidence | Readiness limitation |
| --- | --- | --- |
| CSP → acquisition adapter | Multi-provider setup forms and provider guides exist in `docs/design-modules/03-onboarding/r2/`. Its README and review explicitly identify synthetic previews and runtime NOT_RUN. | Live discovery/setup/read adapters, source credentials, provider pagination and complete real-export retrieval are not certified. A provider logo or completed design ticket is not a collector. |
| Local file → common intake | `tools/focus_data.py` supplies bounded file ingestion. `tools/focus_lifecycle.py:21` scans a local inbox for registered deliveries. `modules/focus_import/intake.py` parses CSV/Parquet. | Local inbox scanning is not a cloud polling connector. CLI actors are trusted scenario identities, not production authentication. |
| Intake → validated and published source | `collection.py:40` registers connections; `register`, `collect`, `confirm`, `publish` implement separate stages. `connections.py` checks domains, calendars, source identity and zero evidence. `service.py` coordinates metadata/storage and publication. | Consumed-field validation and operator confirmation do not certify upstream completeness, supplier reconciliation or financial billability. Source coverage explicitly reports `upstream_verification: False`. |
| Published data → internal query | `collection.py:202` exposes scoped temporal queries; `service.py` and `warehouse.py` use PostgreSQL/ClickHouse and retained source evidence. | Internal results include supplier/source detail. They must not be exposed directly to customer browsers. Current source model has 16 MiB input and 10,000-row bounds (`model.py:14–15`), not production capacity certification. |
| Python query → Go usage service | `modules/customerusage/gateway.go:84` executes a configured Python subprocess. `modules/focus_import/customer_usage.py` projects approved usage fields; `tools/customer_usage.py` is the bridge entrypoint. | Fixed-profile, bounded usage projection with a held-source handshake; not a general remote RPC/data API. No customer selling/allocation amounts or invoice totals are inferred. |
| Go usage → protected query/export | `modules/customeraccess/` provides scope/current-authority checks; `modules/customerusage/exports.go` provides explicit export job operations. | Authority, jobs and artifact bytes are process-local; public authentication, durable distributed administration/job scheduling and production download handlers remain incomplete. |
| Backend → React frontend | Existing documents specify React frontend and Go backend; UI prototypes are available. | No product HTTP server/handlers or OpenAPI definition were found in the inspected `modules/`, `tools/`, `packages/`, `deploy/` paths. Existing Go gateway construction is test-driven, not wired to a deployed application server. |

Supporting maintained runbooks explicitly state these boundaries: [common FOCUS connections](../../ops/installation/common-focus-connections.md), [usage bridge design](../../work/HB-34-customer-access/usage-projection.md), and [protected-export verification](../../validation/reports/HB-34-query-authorization/protected-export.md). Older documents record their own baseline and must not be interpreted as current implementation certification.

## Existing flow and missing links

```text
Provider source / export / API
  -> live acquisition adapters                 not established here
  -> locally supplied CSV / Parquet             implemented entry
  -> registered connection + delivery
  -> validation / quarantine
  -> confirmation + publication                 local implementation
  -> scoped internal query
  -> safe customer usage projection
  -> Go authorization + Python bridge           bounded local implementation
  -> authenticated application HTTP API         not wired
  -> multilingual React screens                 design/integration work
```

The mapping registry requires `exploratory-query-only` (`modules/focus_import/model.py:39`), and the Go response identifies usage-only, exploratory, bounded-page results (`modules/customerusage/gateway.go:163`). A sealed/publication state here must not be presented as production billing eligibility. The source-to-Go bridge covers a narrow usage path. It must not be treated as the universal billing, pricing, organization or administration API. Adding another abstract middle layer before naming its consumers would duplicate existing functions; first define and expose the required boundaries around them.

## NAVER Cloud and KT Cloud requirement

The owner explicitly added both NAVER Cloud and KT Cloud billing support. They are required design targets, not optional omissions. No provider-specific implementation/profile evidence was found for either in the inspected modules, contracts, architecture and design-module paths. The native hierarchy registry currently permits AWS, Azure, GCP, OCI, VMware and Alibaba (`modules/focus_import/hierarchy_model.py:13`); adding a menu label alone cannot add a supported provider. This report records the requirement without inventing an API or claiming provider support.

Each needs a source profile covering commercial/public-sector or other relevant product scope, account/billing authority, acquisition route, cost/usage/document availability, schema/version, units/currency, correction timing, pagination/completeness, permissions and provenance. Native FOCUS support or automatic conversion must not be assumed. Research those capabilities against current official sources before defining the adapter. This task did not perform that provider API research or connect to either account.

## Next frontend and backend design scope

The owner has authorized starting frontend and backend interface design. The following is an initial work order, not approved endpoints, schemas or a completed screen redesign.

1. **Connection and collection status:** design the screen and API together around provider/account scope, source profile, acquisition status, last received period, missing/late coverage, quarantine reason and supported next action. Existing local states are evidence; do not convert them into unverified green “connected” statuses. Include NAVER and KT as unverified setup profiles until capability research is complete.
2. **Validation and publication:** define which authorized operator may inspect evidence and confirm/publish a pinned revision. The UI must distinguish receipt, validation, coverage and publication. Define stale-revision, permission-denied, partial-source and unknown-outcome responses. Internal CLI actor strings must not become caller-supplied authority.
3. **Cost/usage lists and detail:** define separate operator and customer response models. Reuse exact decimal strings, unit/currency/basis, snapshot pins and coverage distinctions where supported. The existing customer profile is usage-only; monetary fields require separate approved contracts and projections.
4. **Jobs and exports:** define durable job identity, status, cancellation/retry semantics, expiry, current-authority download checks and restart behavior before representing asynchronous completion in the frontend. Reuse existing library semantics only where their limits match the product need.
5. **Shared API contract:** document operation purpose, caller scope, request/response fields, nullability, pagination/snapshot handling, errors, idempotency for mutations and versioning. Derive the OpenAPI/typed frontend contract after domain review; none is generated or finalized by this audit.
6. **Multilingual design:** carry forward English/Arabic priority, Korean market importance, UAE-informed layout and the researched terminology rule from HB-61. Compare the same source-status and list/detail tasks in all three languages. Language must not change authorization, currency or document classification.

The first representative end-to-end design is source connection → collection status → validation details → published usage list. The broader product overview, invoices and customer portal must align with that contract but must not display synthetic monetary functionality as connected backend behavior. This sequence supports concrete interface design while the full visual redesign remains a separate deliverable.

## Checks executed for this assessment

| Check | Result |
| --- | --- |
| `go test ./modules/customeraccess ./modules/customerusage` | PASS for both packages (0.375s and 1.239s). Ordinary tests do not establish live engine or provider integration; `TestEngineGateway` requires a dedicated profile and otherwise skips. |
| `tests/engine_probes/.venv/bin/python -m pytest -q tests/focus_import/test_connection_model.py tests/focus_import/test_semantics.py tests/focus_import/test_customer_usage.py` | 94 passed in 0.67s. These selected tests avoid the destructive `stack` fixture. |
| Source/entrypoint inspection | Python CLI commands and Go subprocess bridge traced; no matching product HTTP/server/OpenAPI definitions in the inspected runtime paths. Provider-name search found no NAVER/KT profile implementation in the inspected paths. |
| Historical engine evidence | Existing HB-30/HB-34 reports record synthetic database/bridge verification. Those historical executions were read, not rerun or recertified here. |
| Current live CSP or engine end-to-end run | NOT_RUN. No cloud credentials, customer data, database reset, schema mutation or deployment used. |
| Independent architecture review | PASS: separate read-only interface_audit reviewer traced runtime source and reviewed the final report; no blocking findings. Reviewer did not rerun root tests. |
| Document and staged-scope checks | PASS: required frontmatter, local links, complete one-file staged diff and whitespace inspected; no unrelated files or sensitive data staged. |

This is a readiness assessment and design handoff. It does not complete the frontend redesign, production collector layer, HTTP/API implementation, persistence scaling, security certification or billing acceptance. The ticket remains open pending the reviewed PR and authorized merge; no merge authorization is inferred.
