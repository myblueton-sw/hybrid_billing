---
type: Validation Report
title: HB-27 first local execution 20261023T045921Z-c60f96
description: Records 49 passing local tests and real CLI ingestion, confirmation, publication, scoped query and verified Parquet export with dynamic supported-version contracts.
status: draft
---

# First executable local path

Tracking: [HB-27](https://linear.app/hybrid-billing/issue/HB-27). Machine: `mac_mini`. Owner: Seung Woo Park. [Manifest](evidence-manifest.json), [runbook](../../../ops/installation/local-focus-path.md), [review](../../../reviews/HB-27-local-path/verification.md). Inputs are labelled synthetic CSVs, not provider exports or the official 1.0 demo.

Run `20261023T045921Z-c60f96`: **PASS**. Actual command `tests/engine_probes/.venv/bin/python -m pytest tests/local_focus -q` with JUnit output: 49 passed, zero failures/errors/skips (36 pure, 13 actual PG/CH/S3 integration cases). The manifest records exact source/fixture hashes, tools, start/end, successful command exits and local raw-artifact digests. Tests reset only the disposable hb_local PG/CH namespaces; no production/customer data or other project state is involved.

## Observed behavior

Owner-requested rerun `20261023T051133Z-3b474d`: **PASS**, the same 49 tests with zero failures/errors/skips and successful versions/demo/replay/tenant-query commands. The final local published release is now `38d7a50c-0d82-402c-930b-decef7e9048a`; the registry and export digests below are unchanged. Root used the pinned probe Python environment; invoking the wrapper with system Python first failed before collection because psycopg was absent. Local-only evidence SHA-256: `evidence.json` = `211508e65293beb9f88234f5871fc7dad17b35ca76dc6374f67c122a2df2ed9a`, `tests.xml` = `4d0a6a587c21f5f4a8a3a0d33abbb876313c030b59c6109c6fb2ef060bda347d`, retained under `.local/runs/20261023T051133Z-3b474d/` without a persistence guarantee. Executable source and fixture bytes are unchanged from the manifest.

- Bundled support is discovered from validated registry entries: 1.2 and 1.3. Both have successful real ingestion/publication/query/export cases. A test-only 9.9 entry loads dynamically without a version branch; it is not bundled support.
- Exact source integer-coefficient controls match loaded rows and independent CH grouping/sums, without floating-point monetary arithmetic. Tax NULL quantities survive actual CH and Parquet with present=0, missing=2 and NULL sum. Unit prices are excluded from additive totals and verified per row.
- Scope account/period admission scans every row before storage, including unknown versions. Authorized unusable originals quarantine immutably and keep the source scope held; denied source scope is not stored.
- Wrong confirmation digest, loader confirmation permission, stale registry/policy, conflicting source revision and predecessor CAS races fail as asserted. A CAS race has exactly one winner. Pending/unknown insert and quarantine-policy-race cases retain blocking reservations after session loss or withdrawal.
- Forced PG RLS, composite references, direct CH tenant-reader negatives and reader write/external-function/locked-setting denials were checked. Reinitialization rejects altered elevated role, effective policy-update grant, CH settings and extra reader grants.
- Generic CLI demo and its identical replay use the same release `46f6a7db-b4ec-457b-acf4-4fcc33518287`, registry digest `f46aea7294381123ccf02b2a43b260dbace0348e0cdd8483ed833b9c1a1db4a4`, and export SHA-256 `defd1ba8ec4641dd87faf7a7def99c58efa139256ba256d1f88e45471db706d5`. Published version 1.3 has two rows. Tenant A query returns two; tenant B query returns zero. Original byte equality, immutable create and exported Parquet schema/count/control/hash roundtrip passed.

The four cost measures each total 0.200000 USD in this synthetic scenario, consumed quantity 5 Hours and pricing quantity 2 Hours. These values are independent fixed fixture expectations, not provider/customer invoice truth. Local profile is Decimal(18,6), at most 10,000 rows and 8 MiB; it cannot activate a production numeric profile.

## Corrections and limits

An initial test-collection syntax error and later integration failures were corrected before this run: CH row-policy replacement syntax; PG SELECT FOR UPDATE privilege limited via inert CHECK-constrained lock_token; UTC-aware source versus naive UTC driver return values; test restoration of the complete CH setting profile. Earlier FAIL runs are not claimed as passes. Independent reviews also drove quarantine admission/hold/capability-race guards, effective-role/ACL verification, nonadditive prices, exact product predicates and positive NULL coverage.

An early failing pytest fixture repr exposed local generated connection secrets in session output. The environment object now redacts repr; six generated local secrets and derived app credentials were rotated, four verification containers restarted, and PG/CH/S3 authenticated read-back passed without printing values. No exposed value is stored in this report or committed artifacts. Historical session output is not claimed erased; old local values are no longer active.

This is local scaffold acceptance only. Synthetic persona selection is not production authentication. Real provider profiles/exports, complete source/customer mapping, Go access/policy APIs, React UI, full financial workflows, quarantine resolution/re-request dispatch, actual fenced takeover/destructive deletion, independent restore-control journal, actual rebuild import, HA and capacity remain NOT_RUN or unimplemented. The inspected official FOCUS 1.0 demo remains exploratory; no live connector or full FOCUS certification is asserted. The local stack and final published synthetic sample are deliberately left running.
