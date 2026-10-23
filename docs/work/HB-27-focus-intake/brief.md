---
type: WorkBrief
title: HB-27 FOCUS-first file intake
description: Extend the common bounded input path to exact Parquet without changing tenant ownership or claiming full FOCUS conformance.
---

# Scope and ownership

The owner prioritizes FOCUS common ingestion before cloud-native connectors. HB-27 and PR24 remain the execution boundary, with root writing source, tests and documents. Separate DBA/contract review and QA validate design and changes. Machine: mac_mini. No merge is authorized by this brief.

# Input contract

Version is never inferred from a filename or coincidentally compatible headers. The caller declares a supported version (currently 1.0, 1.2, 1.3); the registry chooses column names dynamically. This remains an exploratory projection, not complete FOCUS conformance. CSV behavior is preserved. Parquet is detected by its PAR1 signature rather than filename, and must have a valid footer. Every mapped column must exist exactly once. Unknown versions fail closed.

Parquet supports scalar UTF-8 strings, integers, exact decimals, booleans for extra fields, and microsecond-or-coarser timestamps. Mapped numerics accept string, integer or decimal, never floating point; mapped timestamps accept strings or timestamps, with UTC as the existing source timezone. Other mapped fields require strings. Nested, binary, floating-point and unsupported logical types fail closed, including extra columns. Missing numeric values remain NULL; typed nulls have a distinct missing reason. All columns remain in source_json as exact lexical strings or JSON null; original bytes remain the authoritative typed evidence.

Limits: 16 MiB compressed input, 16 MiB declared uncompressed column chunks, 10,000 rows, 256 Parquet columns, 65,536 UTF-8 bytes per cell and 16 MiB cumulative decoded cells. Bound metadata decoding and reject unsupported/encrypted/corrupt input through sanitized errors. No external file or dataset discovery occurs.

# Storage and compatibility

Decode both formats into the same normalized rows before assignment, registration and publication. Original object suffix follows the detected format for new writes; existing CSV object keys, retained manifests, backups and journal entries remain valid. New manifests pin input format alongside file digest and mapping. No DB migration or change to fact columns is required. The registered file digest continues to bind the exact original representation; changing format requires a new delivery/revision. Scheduler discovers .csv and .parquet files and still only collects registered digests.

# Validation

Pure tests cover all declared versions and CSV/Parquet projection equivalence, high precision, negative/zero/null, wrong-version headers, malformed files, duplicate/missing columns, float/nested rejection and size bounds. Real PG/ClickHouse/S3 tests use synthetic inputs through CLI describe/register/collect/confirm/publish/query, read back exact controls and original bytes, verify tenant/company isolation, retry and archive/restore compatibility. Existing regressions run once after scoped tests. Restore the existing sample runtime from its independent verified backup after disposable tests.

# Exclusions

No live cloud API/export discovery, full-standard certification, commercial settlement, invoice generation or private customer sample ingestion. Public 1.0 samples and synthetic newer-version fixtures must be identified separately. Independent design review precedes implementation and actual-diff review precedes completion.

# Technical references

[Apache Arrow ParquetFile](https://arrow.apache.org/docs/python/generated/pyarrow.parquet.ParquetFile.html) documents single-file metadata, bounded Thrift decoding and record-batch iteration. Validate against the locally installed PyArrow21.0.0 API. [FOCUS1.3](https://focus.finops.org/docs/specification/v1-3/) is the standard reference; the existing [consumed-column design](../../architecture/data-ownership/focus-consumed-columns.md) retains the broader product semantic gates. This increment does not close those gates.
