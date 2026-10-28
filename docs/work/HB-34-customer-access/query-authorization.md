---
type: SecurityImplementationPlan
title: Internal customer query authorization increment
description: Defines a Go query boundary with current scoped grants and revocation-aware result release while authentication remains deferred.
---

# Scope and owners

HB-34; base main `341f93475cf4cdbcaa13b1cd54822f9892b01bfd`; accountable owner Seung Woo Park; machine mac_mini. Root owns implementation, tests and integration. A separate architecture/security reviewer owns design and actual-change review. Owned paths: root `go.mod`, `modules/customeraccess/`, this work directory and `docs/validation/reports/HB-34-query-authorization/`. No existing database, Python lifecycle implementation or local credentials are changed.

This is the first Go backend increment. It implements an internal query service and a current-authority in-memory store, exercised with a synthetic reader and trusted test identity verifier. It is not a public server, login implementation, durable production authority store, FOCUS adapter or complete customer projection/export feature. The interface is consumed by the internal query service and executable Go tests; there is no unused HTTP route or mock login shipped as authentication.

# Boundary and invariants

The service accepts a context and a typed request containing exact company/tenant pairs, selected field IDs and a bounded row limit. It does not accept a subject, role, permission list or authoritative actor ID in the request. A server-installed verifier resolves the context into installation, stable internal subject ID, credential epoch and explicit validity deadline. A missing/unavailable verifier, empty identity, wrong installation, stale epoch or expired identity denies before the data reader. This trusted adapter is a future authentication integration point; calling it successfully in a synthetic test is not external identity proof.

The authority store owns active companies, tenant-to-company assignments and active subjects with explicit scoped grants. All identifiers are exact opaque IDs; display labels/email/payer ancestry never grant membership. Grants bind company, tenant, action, fields and an explicit expiry. The query service fixes the action to `usage.read`; an export-only grant does not grant query access. There are no wildcard or role-name shortcuts. Multiple grants may combine fields only within the same exact company/tenant/action. Every requested scope and field must be allowed or the whole request is denied; no partial union is returned.

Configuration is trusted internal input. Store updates validate and deep-copy data under a mutex, preserve prior state on invalid updates, and advance scoped versions. Company/tenant inactivity, reassignment, subject inactivity, grant removal, expiry and credential-epoch change revoke relevant authority. Store unavailability denies and changes its availability generation. Restart creates an empty store; production persistence, distributed updates and rollback-resistant authority restoration remain unimplemented.

Each record update supplies its expected version (zero for creation); stale updates fail atomically. Subject credential epochs are positive and never decrease. Reactivating an inactive subject requires a strictly higher epoch, so disabling and re-enabling cannot revive its old credential. A rejected update leaves both data and version unchanged. Explicit new grants are permitted by current trusted configuration; request stamps still invalidate reads spanning any relevant update.

Construction requires an explicit immutable field catalog, clock and positive maximum row/scope/field limits. These are internal configuration, not product defaults. Both checks use the server clock; expiry equality denies. Caller input is copied before authorization, and the reader receives a separate copy. The trusted reader transfers result ownership on return and must not mutate it concurrently. Released results are validated and copied under the authority read lock. Economic revision, period, currency, basis and customer-safe content semantics require the future reviewed source/profile adapter.

Callers must not concurrently mutate request slices during initial validation/copy. The supplied clock must be concurrency-safe and non-reentrant into the store; it is trusted server infrastructure. Row/scope/field counts are bounded, but string byte sizes and reader execution time are not independently bounded here. A future adapter must supply appropriate resource budgets and honor context cancellation before customer exposure.

The service authorizes before calling the reader and reauthorizes before releasing a result. It records subject, requested-company/tenant and availability versions; intervening relevant changes reject the result even if grants later appear equivalent. Unrelated subjects/tenants do not change that stamp. No lock is held while the reader executes, so revocation is not delayed by a slow read. Release validation under the store read lock is the internal linearization point; an already released response cannot be recalled. The reader receives copied authorized selection only, with a propagated cancellation context.

The bounded reader result must contain only requested scope pairs and exactly the requested fields, with no duplicate row identity per tenant and no more than the requested row limit. Unexpected/missing fields, foreign rows or malformed scope fail the entire result. Values remain opaque strings here: no floating-point conversion, pricing calculation, production disclosure profile or source-cost-to-selling-price mapping is introduced. This validates the selected-field boundary but does not certify the meaning of a field's contents. No arbitrary filter/group/SQL, aggregate totals, pagination, caches or export path is exposed in this increment.

Errors exposed by the query service are uniform denied/unavailable/cancelled outcomes without subject existence, foreign identifiers or raw verifier/reader error text. There is no anonymous fallback. Reader failures return no partial rows.

# Verification and next boundaries

Use standard-library Go tests and `go test -race ./...`: positive A/B and explicit multi-tenant/MSP grants; unknown/forged identity and scope; wrong company/installation; stale epoch; expired identity/grant; wrong action; per-scope field intersection; inactive company/tenant/subject; malformed updates; authority outage; deep-copy ownership; foreign/extra/missing/duplicate/oversized reader output; reader error/cancellation; revocation and revoke-then-regrant during a paused read; unrelated updates; concurrent A/B requests without scope contamination. Fixed synthetic values are independent expectations. Tests must assert denied requests never reach the reader and revoked results never escape.

After this bounded increment, connect reviewed disclosure profiles and real source adapters, then protected export/download work. Authentication remains deferred under the [sequencing decision](brief.md). I18/I19 external end-to-end certification remains NOT_RUN. Deployment, customer data and real accounts are outside scope.

# Internal integration contract

The implementation is `modules/customeraccess`. Construct `NewStore` with an installation ID, an explicit field catalog and a server clock. Populate companies, tenants and subjects through the trusted `PutCompany`, `PutTenant` and `PutSubject` methods. Retain each returned version for the next expected-version update; zero is valid only for creation. There is no deletion or client-facing configuration endpoint. State is process-local and starts empty. `SetAvailable(false)` denies queries; making the store available again cannot release results read across that transition.

Construct `NewService` with a trusted `VerifyFunc`, that store, a trusted `ReadFunc` and explicit `Limits`. Call `Query(ctx, Selection)`; the request contains only scopes, fields and row limit. The verifier must establish the principal outside client-controlled query input. The synthetic test verifier is confined to test files and is not a production authentication mechanism. Reader output is an owned bounded row batch; an empty batch is valid, and no total/completeness claim is made. Never expose the store update methods or install a verifier that trusts request actor IDs in a customer handler.

`ErrConflict` and `ErrInvalidPolicy` are trusted configuration errors. Query errors are `ErrDenied`, `ErrUnavailable` or context cancellation/deadline errors; source/verifier messages are suppressed. Result values remain exact opaque strings. The current tests use synthetic field names and values, not a certified customer disclosure profile.

The independent design review initially rejected missing credential rollback protection. Positive nondecreasing epochs, higher epochs on reactivation and atomic expected-version updates resolved the finding before implementation. Independent actual-change review and runtime QA subsequently passed; see the [verification report](../../validation/reports/HB-34-query-authorization/results.md).
