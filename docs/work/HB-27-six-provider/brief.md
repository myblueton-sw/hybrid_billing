---
type: Implementation Plan
title: Six-provider sample ingestion and authorized company views
description: Extends the local sample path with real GCP demo rows, labelled VMware and Alibaba fixtures and explicit multi-tenant company-query grants.
status: draft
---

# Six-provider sample acceptance

Tracking: HB-27. Owner: Seung Woo Park. Machine: mac_mini. Root sole writer; independent DBA/security design/change review and QA. Extend existing PR24, not a new product stack. Owner requests sample-data execution for AWS/Azure/GCP/OCI/VMware/Alibaba and integrated views when multiple tenants belong to one company. No real-account, deployed authentication, payment or merge permission.

## Data and fixture scope

Keep unchanged official1000 FOCUS1.0 rows. Extract all115 Google Cloud rows from pinned official full gzip by streaming CSV, preserving lexical column values, original full-file hash and original ordinals. Derived selected-file CSV is labelled a subset, not unchanged original bytes. Preserve both original official1000 and full-source hash, selection ordinals, derived-row digest, output hash and provider counts in local provenance. Fetch/verify pinned original1000 and full gzip (reuse verified local cache with explicit provenance or reacquire); sample builder must validate hashes before extraction.

No verified reusable official VMware or Alibaba billing sample was located in the focused official/source search; do not claim universal absence. Add four deliberately synthetic FOCUS-shaped1.0 rows for each, with own synthetic accounts/amounts and row-level provenance. These demonstrate the platform sample contract, not native provider collection, exporter correctness or pricing truth. Keep official labels AWS/Microsoft/Google Cloud/Oracle; canonical display/filter aliases map Azure/OCI/GCP explicitly without altering original values. Dataset/response evidence class distinguishes observed public rows from synthetic additions. Input version stays1.0; generic1.2/1.3 mapping remains independent.

The bounded complete dataset is official1000 +GCP115 +synthetic8 =1123 rows. Provision three synthetic tenants: north-a owns official AWS/Microsoft/GCP and two VMware/two Alibaba rows; north-b owns official Oracle and one VMware/one Alibaba row; foreign owns one VMware/one Alibaba row. Exact domains drive assignment, not file tenant claims. North-a/north-b share immutable company_id; foreign has a different company_id even if company labels coincide. North integrated expected1121; foreign2. Distinguish currency and quantity units; include CNY and USD to prove no mixed-currency sum or automatic conversion.

Disposable hb_focus test fixtures may reset only that documented namespace; final fixture reseeding installs this new complete dataset and preserves all previously demonstrated official1000 source rows in the final sample. Historical manifests/raw originals remain in S3 without deletion claims; resetting test namespace is not a product migration/recovery feature. Existing hb_local untouched. Existing immutable configuration init continues rejecting unexpected changes; no silent source remap or general profile replacement API is added.

## Authorized integrated query

Tenant properties optionally include company_id/company_label; company ID alone grants nothing. Extend trusted initial configuration with optional explicit views: unique reader actor, company_id, active state and nonempty unique tenant key allowlist. Validate every referenced tenant belongs to that exact company ID, reject duplicate aliases/unknown scopes/cross-company membership. Absence of views preserves existing behavior. No arbitrary tenant list from a runtime query is accepted.

Add checksummed migration002 with server-owned audience and audience_member tables, relational tenant FKs, active state and inert locking column. Only provisioning admin writes memberships; runtime importer/approver may read for consistency checks, tenant readers cannot view another audience. Integrated role derived from immutable view actor identity, not local binding tenant substitutions. Role-bound PG RLS grants the view only its allowlisted tenant rows/slices/pointers plus its audience/membership rows. Separate CH view reader has the same explicit tenant OR-list row policy and locked settings, no writer/external-function privileges. Validate effective grants and unexpected policies as existing init does. Audience source/company/member set is pinned and checked against trusted server configuration; local binding hints alone cannot widen it.

Integrated query acquires the source gate shared lock, then audience and all allowed tenant policy rows in deterministic order; policy/membership changes serialize against reads. Recheck active actor/view/tenants and exact company IDs. All selected PG current pointers must agree with the same stable sealed source batch. Materialize each exact authorized slice via the view CH role and verify its row digest/controls. Aggregate only the resulting authorized union. Never combine financial measures across currencies or quantities across units. Return per-tenant counts/identity and company view metadata; retain provider/account/tenant breakdown. Shared source original keys remain coordinator-only. Company labels are display only. An inactive/missing member denies the complete view rather than silently dropping it.

Pagination remains original source ordinal +expected batch; ordinals uniquely cover a batch across tenant slices. No duplicate summation when tenant/source rows coincide lexically. Unknown storage reservations block individual and company reads; import/confirm/publish remains all-tenant atomic and replay-safe. Provider query filters accept canonical aliases via a reviewed fixed mapping; raw source labels preserved in rows.

Independent DBA/security design reviews: PASS before code, requiring every membership INSERT/UPDATE/DELETE to lock the parent audience FOR UPDATE, while readers hold parent FOR SHARE; discovered-row locks alone do not prevent phantom membership. Audience identity/expected member set is immutable; active-state changes use the parent row. Reject duplicate union ordinals and actor aliases across tenant/view/reserved namespaces.

## Owned files and verification

Root edits model/authority/warehouse/service +new focused sample/view helpers if needed; tools/focus_data.py; migration002; synthetic fixtures/provenance/alias contract; tests and runbook/report/README. No source/test file1500 lines. No explanatory source comments. Public source rows/account IDs and full raw commands stay ignored; commit only synthetic fixtures and aggregate/hash provenance.

Actual CLI build samples -> trusted init -> ingest -> distinct confirmation/publication -> individual and company queries. Verify all six providers, actualGCP115 source rows, all1123 unique ordinals and North1121/foreign2 disjointness; direct unrestricted PG/CH view-role SELECT excludes foreign; partial reader cannot escalate to company/other tenant through bindings, labels, filters or tenant-list input. Test mixed currencies/units, independent Fraction controls, overlap/member tamper, inactive/mid-query policy withdrawal, expected-batch pages, duplicate replay and existing unknown/rollback regressions. Rerun prior37 tests plus new relevant pure/integration cases. Independent reviews and actual QA required; final local sample left queryable and PR24 rewritten with truthful per-provider evidence classes. No live connector/native compatibility certification from synthetic input.
