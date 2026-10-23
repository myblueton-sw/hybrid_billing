---
type: Runbook
title: Store and query official FOCUS demo data by tenant
description: Runs the bounded public-demo path with immutable tenant source assignments, exact decimal values, scoped detail pages and aggregate queries.
status: draft
---

# Tenant-aware FOCUS data

For the richer six-provider/company-view sample currently left running after its verification, use the [six-provider workflow](six-provider-samples.md). This page retains the official-only profile and its commands. Both use the same disposable namespace and immutable initialization policy; an incompatible existing profile is rejected rather than silently replaced. Test resets are documented below and are not a product configuration migration.

Tracking: [HB-27](https://linear.app/hybrid-billing/issue/HB-27). Owner: Seung Woo Park. Machine during implementation: `mac_mini`. [Plan](../../work/HB-27-tenant-focus/brief.md). This is a local trusted-operator CLI using actual public demo rows and explicitly synthetic tenant assignments. Local actor selection is not product authentication. Do not expose it as a customer API.

## Start and execute

Use the existing [verification stack](../../../deploy/local/verification/compose.yaml), generated ignored environment and pinned [probe Python environment](../../../tests/engine_probes/pyproject.toml). Run from the repository root. Keep credentials and downloaded rows out of Git. The new path uses `hb_focus` PG/CH namespaces and S3 bucket `hb-focus-originals`; the old `hb_local` synthetic path stays independent.

```sh
docker compose -f deploy/local/verification/compose.yaml up -d --wait
tests/engine_probes/.venv/bin/python tools/focus_data.py fetch-demo
tests/engine_probes/.venv/bin/python tools/focus_data.py versions
tests/engine_probes/.venv/bin/python tools/focus_data.py init-demo --file .local/datasets/focus/official-1000.csv
tests/engine_probes/.venv/bin/python tools/focus_data.py demo --file .local/datasets/focus/official-1000.csv
tests/engine_probes/.venv/bin/python tools/focus_data.py query --actor reader-demo-a --limit 20
tests/engine_probes/.venv/bin/python tools/focus_data.py query --actor reader-demo-b --provider Oracle --limit 20
```

Fetch verifies exact pinned official bytes before local rename. Initialization provisions roles/tenant properties/exact source assignments; it needs local admin access. `demo` is an explicitly labelled synthetic approval scenario: original intake -> exact full-file preflight -> durable reservation -> original/CH storage -> independent row/control verification -> digest confirmation -> atomic publication. Its subsequent effects run as separate importer/approver roles, not admin. An identical replay returns the same batch without inserting rows or changing pointers. Expected results: A942 AWS rows; B58 rows (Microsoft51 + Oracle7). Full source union is1000; equal rows are retained by original ordinal.

The individual steps are available without automatic scenario approval:

```sh
tests/engine_probes/.venv/bin/python tools/focus_data.py ingest --actor importer --version 1.0 --revision 1 --file .local/datasets/focus/official-1000.csv
tests/engine_probes/.venv/bin/python tools/focus_data.py confirm --actor approver --batch <batch-id> --manifest-digest <manifest-digest>
tests/engine_probes/.venv/bin/python tools/focus_data.py publish --actor approver --batch <batch-id>
```

Replace placeholders with ingest output. Revisions are local snapshot intake identifiers, not a claim about provider correction order. Changed bytes under the same revision conflict; incomplete/changed snapshot bytes under a new revision also reject against the provisioned exact snapshot profile. New source content/assignments/properties require a reviewed configuration transition; this increment has no reconfiguration or cleanup command.

## Tenant properties and query contract

`init --config <ignored-json-file>` accepts an explicit validated trusted configuration instead of the pinned `init-demo` provisioning recipe. The schema is enforced by [model.validate_config](../../../modules/focus_import/model.py). Required source properties: stable source key, declared mapping version, positive configuration revision, complete snapshot SHA-256/row count/domain-multiplicity digest and tenant list. Each tenant supplies unique key/reader actor, display label, active state, IANA reporting timezone, reporting currency, allowed source currencies, retention-days metadata and exact provider/account/subaccount/billing-start/end tuples. Do not derive customer ownership from source account names or an uploaded tenant column. Overlap, unknown source tuples and denied currencies fail before storage.

The pinned demo recipe makes 73 exact source-domain assignments. A uses Asia/Seoul reporting labels and180-day retention metadata; B uses UTC and365 days. These are synthetic test choices. Reporting currency is metadata only; no FX conversion occurs. UTC timestamps and original lexical timestamps remain available; unzoned1.0 demo timestamps use the mapping profile's explicit UTC interpretation. Charge intervals need not nest inside billing periods (delayed billing exists). Retention metadata is returned with `retention_enforcement=false`; there is no deletion/expiry claim.

Queries derive tenant scope from the server-owned actor binding, then use a tenant-specific PG/CH role. Caller tenant IDs are not accepted. PG current pointers select the exact immutable batch before CH access. Shared original keys/manifests and other tenant data are never returned. Source columns including account/resource IDs are visible only inside authorized tenant detail rows; they are public demo values, not real-customer attribution evidence. Original shared source bytes are coordinator-only.

Detail output is bounded to1–100 rows and contains original source ordinal, exact decimal strings, NULL provenance, original lexical row JSON and reporting-timezone period labels. `groups` covers all rows matching the filter, not only the current page. `--provider` and `--service` are exact-value filters after tenant authorization. `next_after` is the last returned original ordinal; continue with both `--after <next_after>` and `--expected-batch <batch_id>`. A changed current batch rejects continuation instead of mixing pages across snapshots. An offset without batch pin rejects.

Cost and quantity outputs use integer coefficients at scale18. Money remains grouped by source currency; quantity by exact units. Unknown currency remains its own explicit group where configured. Unit prices are never summed. Missing values retain token reasons, present/missing counts and NULL sums for all-missing groups. Maximum local file16 MiB/10000 rows; Decimal(38,18), per-value coefficients and absolute batch additive totals strictly below10^38. Excess scale/magnitude rejects; no monetary float/rounding is used. Decimal256 CH expressions widen exact verification sums only.

## Failure and verification

Exit0 indicates command success; exit2 rejected admission/permission/state; exit3 dependency or unknown-operation failure. Raw driver errors and credentials are suppressed. Pending/unknown storage reservations survive process failure and block subsequent source reuse and query. There is no automatic lease takeover, retry deletion or manual-record-erasure recovery instruction. Confirmation and publication recheck exact manifest/config/mapping and current tenant policies. All tenant pointers switch in one PG transaction only after every slice verifies. Concurrent policy changes serialize against tenant read locks.

```sh
tests/engine_probes/.venv/bin/python -m pytest tests/focus_import -q
```

Tests require the pinned file at `.local/datasets/focus/official-1000.csv` (fetch first). Integration fixtures reset only disposable `hb_focus` PG/CH tables/policies and restore local binding file bytes; running tests removes this path's current demo state. Existing `hb_local`/historical probes and other projects are untouched. Run `init-demo`/`demo` afterwards to leave a queryable sample. Synthetic S3 test originals remain local without a cleanup claim.

## Evidence class and remaining scope

Registry mappings1.0/1.2/1.3 are dynamically selected and explicitly labelled `exploratory-query-only`. The actual official data remains version1.0; it is never relabelled1.2/1.3. Mapping tests for1.2/1.3 use transformed in-memory copies, not certified provider exports. This path does not replace the existing strict synthetic1.2/1.3 validation path or authorize billing/pricing/issuance.

The full5.48M-row gzip is retained/profilable but is outside this bounded importer. Production source/customer attribution, full FOCUS conformance, live collectors, API/UI/user authentication, financial acceptance, actual currency conversion, retention execution, quarantine/re-request, safe deletion/recovery, HA and capacity remain subsequent work. Leave verification services running; `docker compose -f deploy/local/verification/compose.yaml down` stops them without removing volumes.
