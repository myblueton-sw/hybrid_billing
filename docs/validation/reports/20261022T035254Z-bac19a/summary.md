---
type: Validation Report
title: HB-27 engine probe run 20261022T035254Z-bac19a
description: Summarizes an actual local run of the HB-27 engine behavior probes against PostgreSQL 18.6, ClickHouse 26.8.33.10 and SeaweedFS 4.48, with outcomes and the design changes they require.
status: draft
---

# Engine probe run 20261022T035254Z-bac19a

Tracking: Linear ticket [HB-27](https://linear.app/hybrid-billing/issue/HB-27). Machine: `mac_mini`. Plan: [HB-27 engine probes](../../plans/HB-27-engine-probes.md). Manifest: [evidence manifest](evidence-manifest.json). Design under test: [platform data build design](../../../architecture/platform-data-build-design.md).

Run on 2026-10-22 from 03:52:55 to 03:52:56 UTC, Docker Engine 28.3.2, pinned image digests, synthetic values only. pytest: 51 cases passed; the pytest status only means each probe ran and recorded an outcome. Outcomes: 47 CONFIRMS, 3 CONTRADICTS, 1 NOT_RUN. Every rejection is classified by its exact ClickHouse error code or S3 error code; any other error, including setup errors, is recorded as NOT_RUN. Earlier runs on the same day were setup and probe-correction iterations and are not counted.

## Outcomes

| Probe | Outcome | Observed |
| --- | --- | --- |
| EP-01 Decimal128 sum | CONFIRMS (2) | Two `Decimal(38,0)` values of 10^38 − 1 summed to `-140282366920938463463374607431768211458` (that is 2·(10^38 − 1) − 2^128) with no error, with `decimal_check_overflow` 1 and 0 |
| EP-02 Decimal64 sum | CONFIRMS | `sum` over `Decimal(18,6)` returns `Decimal(38,6)`, exact `1800000000000` |
| EP-03 Excess scale | CONFIRMS (2) | `1.1234567` and `1.1234564` into `Decimal(18,6)` accepted silently as `1.123456` (truncation), from CSV and through `s3()` |
| EP-04 NULL and empty | CONFIRMS (15), CONTRADICTS (3) | See the matrix below |
| EP-05 Missing Parquet column | CONFIRMS (2) | Default fills NULL silently; `input_format_parquet_allow_missing_columns = 0` rejects with code 8 |
| EP-06 `join_use_nulls` | CONFIRMS | Default gives `0`; locked profile gives NULL |
| EP-07 Row policies | CONFIRMS (4), NOT_RUN (1) | Configured server: a user without a matching policy gets code 497 "Table hb.facts has row policies, but none of them are for the current user", the unmatched-row-policy error. Default server: that user reads every row. A tenant-A user sees zero tenant-B rows on both. The effective `access_control_improvements` values are not exposed in `system.server_settings` (NOT_RUN). The error text shows the throw behavior is active; whether `users_without_row_policies_can_read_rows` also takes effect was not separately tested |
| EP-08 Reader bypass | CONFIRMS (12) | `s3`, `url`, `remote`, `file`, `INSERT INTO FUNCTION`, `ALTER … DELETE`, `ATTACH PARTITION` and `MOVE PARTITION` to another table, `system.named_collections`: code 497. Changing `join_use_nulls`, `input_format_null_as_default` or `serialize_query_plan`: code 452 |
| EP-09 Synchronous delete | CONFIRMS | With `lightweight_deletes_sync = 2`, zero stale-attempt rows remain immediately; 100,000 current rows kept |
| EP-10 Parquet export | CONFIRMS | `cost` `decimal128(18,6)` non-nullable and `amount` `decimal128(38,10)` nullable, both asserted |
| EP-11 SeaweedFS | CONFIRMS (3) | `If-None-Match: *` second write rejected (`PreconditionFailed`); versioning keeps two versions; deleting a COMPLIANCE-locked version rejected (`AccessDenied`) |
| EP-12 Named collection | CONFIRMS | Reads work. After a missing-object error, a wrong-credential `s3()` call and a failed login, no credential (including the probe user password) appears in `system.query_log`, `system.text_log` or the error texts |
| EP-13 PostgreSQL RLS | CONFIRMS (2) | Role is not superuser, BYPASSRLS or owner; no scope gives 0 rows; `SET LOCAL` scope gives only that tenant; after `COMMIT` 0 rows and `current_setting` is `''` |

### EP-04 matrix (row 1 of each file)

| Settings | Column | CSV `\N` | CSV empty field | Parquet NULL |
| --- | --- | --- | --- | --- |
| Default | Nullable | NULL | NULL | NULL |
| Default | non-Nullable | `0` | `0` | `0` |
| Design-locked (`empty_as_default 0`, `null_as_default 0`) | Nullable | NULL | **`0`** | NULL |
| Design-locked | non-Nullable | rejected (117) | **`0`** | rejected (349) |
| Candidate (`empty_as_default 1`, `null_as_default 0`) | Nullable | NULL | NULL | NULL |
| Candidate | non-Nullable | rejected (117) | **`0`** | rejected (349) |

The three CONTRADICTS are the bold cells. No tested setting makes an empty CSV field fail for a non-Nullable column.

## Design changes for the physical schema step

From recorded probe outcomes:

1. **Missing values.** Use `input_format_csv_empty_as_default = 1` with `input_format_null_as_default = 0`, and declare every consumed money and quantity column `Nullable`. This is the only tested combination that never turns NULL or an empty field into zero (EP-04). Whether a value is required is enforced by the pre-publication NULL-count check, never by a non-Nullable column type, because an empty CSV field becomes `0` in a non-Nullable column under every tested setting.
2. **Decimal128 sums wrap silently** (EP-01). The per-column precision bound and the Python recomputation of every pricing total are mandatory. `decimal_check_overflow` does not protect `sum`.
3. **Excess scale truncates silently** (EP-03). The Python manifest step is the only detector; it compares exact source strings, not ClickHouse-loaded values.

From setup iterations of this run, observed while building the stack and not recorded as probe outcomes:

4. **Loader grants.** `s3()` with a named collection needs `S3`, `NAMED COLLECTION ON <name>` and `CREATE TEMPORARY TABLE`. Users that receive SQL grants must be SQL-managed; XML-defined users cannot receive grants (error 495).
5. **Container config.** Mounting `config.d` replaces the image's own listen configuration; the server config must set `listen_host` explicitly.
6. **SeaweedFS single node.** Each bucket allocates volumes; the volume limit must be sized for the expected buckets (`-volume.max`); without it, uploads failed with "no writable volumes".

## Not covered

Conditional create in customer S3 services, real provider Parquet money types, which providers export FOCUS 1.3, production, HA, performance and capacity: NOT_RUN, as listed in the plan. The defaults ClickHouse service relies on the image entrypoint for its user setup; that service exists only to observe default behavior.
