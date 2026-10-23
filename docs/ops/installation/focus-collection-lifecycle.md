---
type: Runbook
title: Recurring FOCUS collection and tenant lifecycle
description: Runs registered complete-window delivery, accumulated queries and guarded archive, deletion, backup and recovery on the trusted local verification stack.
status: draft
---

# Local operating boundary

Tracking: HB-27, owner Seung Woo Park, machine mac_mini. This extends the [six-provider sample path](six-provider-samples.md). Use the pinned Python environment and running verification PostgreSQL18, ClickHouse26.8 and private S3 service. Commands below use `tests/engine_probes/.venv/bin/python tools/focus_lifecycle.py`. This is a trusted single-host CLI with explicit scenario actors, not production authentication, a live provider API adapter or a capacity certification. No real cloud credentials or customer data are needed.

The importer accepts registry-declared FOCUS versions; the bundled mappings are1.0/1.2/1.3 and remain exploratory-query-only. Provider name alone does not establish a native adapter or compatibility. GCP public source and deliberately synthetic VMware/Alibaba fixtures retain their existing provenance. Registration asserts source completeness locally; the software cannot infer provider-side completeness from a file or verify its truth without an authoritative source adapter.

# Register and collect

Initialize the existing tenant configuration with `tools/focus_data.py init --config PATH`. Then activate lifecycle once with `activate --actor operator --grain day --policies PATH`. The policy JSON maps every configured tenant key to an explicit approved policy: `revision`, `basis` (`received_at` or `window_end`), `hot_days`, `total_days`, `grace_days`, `backup_days`, UTC `reject_before`, and `approved_by`. No legal duration is supplied by the product. Initial adoption of older batches has unknown receipt time; a received-time policy cannot archive/delete those batches until an evidenced origin is available. Window-end policy uses recorded source billing bounds.

1. Run `describe --file FILE --version 1.0 --start UTC --end UTC --grain day --revision 1 --complete-through UTC --finalized`; save the returned `descriptor` object, not its enclosing response, to a local file.
2. Run `register --actor operator --file FILE --version 1.0 --descriptor DESCRIPTOR`. Registration pins exact file digest, row count, domain multiplicity, mapping, ownership, source window, completeness/as-of and economic revision.
3. Run `collect --actor importer --delivery ID --file FILE`, or `run-due --actor importer --inbox DIRECTORY` for already registered CSV files. Durable registration survives process restart. One verified batch awaits independent publication at a time; pending files are not dropped when byte digests coincide.
4. Run `confirm --actor approver --delivery ID --manifest-digest DIGEST`, using the actual ingest result. Then `publish --actor approver --delivery ID`. Source bytes, typed facts and controls are verified before publication.

`watch --actor importer --inbox DIRECTORY --interval 60` repeats the registered inbox scan; `--cycles 1` runs one bounded cycle. Run it in the operator's process supervisor for ongoing collection. It never invents completeness, registers unknown files or bypasses approval. Confirmation/publication remains a separate role. No automatic recovery or retry occurs after an unknown storage effect.

A source chooses UTC day or month windows. Distinct intersecting windows are rejected before publication; exact-window revisions increase and compare the predecessor. Daily windows accumulate. Monthly complete-to-date refreshes replace the previous current revision of that month; they are never added together. Historical revisions remain retained. Explicit header-only complete corrections clear a window. Equal source rows remain distinct occurrences. Versions can vary per registered delivery if the registry mapping and tenant-domain ownership permit them.

# Accumulated query

Use `query --actor READER --grain day` or `--grain month`. Configured company actors see exactly their authorized tenant union. Currency and quantity-unit groups remain distinct; there is no currency conversion or price summation. Day/month buckets use each tenant's timezone and source charge-start attribution, with no interval proration. Reporting timezone and intake UTC calendar are separate.

`--start`/`--end` select overlapping **delivery windows**, not a precise usage-time filter. The response explicitly labels this range basis and includes complete-through/finalized coverage. Late charge dates can differ from the billed window. A selection is bounded to10000 rows and the full-selection exact numeric envelope; an oversized selection rejects rather than truncating totals. Select a smaller delivery interval when needed. This local bounded implementation does not establish planned production capacity.

Use `--limit`, returned `next_cursor`, and `--snapshot` for stable pages. The cursor contains window/batch/ordinal; correction, policy, membership or relevant lifecycle changes invalidate the snapshot. Counts and totals cover the complete selected authorized set, not only the page. A deleted current scope makes the tenant/company result explicitly unavailable; it is never silently omitted. Filter to an unaffected delivery interval if appropriate.

After periodic windows exist, legacy `tools/focus_data.py query` rejects with `period_query_required`; its old single-pointer result must not be mistaken for accumulated history.

# Archive, holds and deletion

`status --actor operator` lists slice identities (`BATCH_UUID:TENANT_UUID`), policies, windows, deliveries, backups and actual deletion residuals. Archive a hot-eligible slice with `archive --actor operator --slice ID`; private typed Parquet must pass byte/hash/schema/row/count/exact-control checks before hot rows are removed. Period queries can read the verified archive under current tenant/company authority. `recall --actor operator --slice ID` rebuilds hot rows without resetting retention origins. Re-archive is idempotent against identical content. Corrupt or missing bytes deny access instead of returning zero.

Use `hold` or `reference` with `--actor operator --slice ID --id KEY --reason TEXT`; release with `--release`. Holds/dependencies serialize with deletion and policy changes. A hold after committed deletion returns too-late; it cannot promise to preserve removed bytes.

Run `preview --actor operator [--slice ID ...]`, `approve --actor lifecycle-approver --manifest-digest DIGEST`, then `delete --actor operator --manifest-digest DIGEST`. The fixed manifest pins scope, policies and restrictions. Any relevant change invalidates approval. A source/data retention expiry is eligibility, not an immediate deletion instruction. Only synthetic/public local verification data is authorized here.

Deletion separately reports active row absence, mutation completion, inactive CH parts, object-version uncertainty, shared originals and managed backup copies. Tenant A expiry cannot remove B's shared original or held evidence. Full PG/journal snapshots are shared source evidence while other tenants depend on exact configuration/recovery. Per-slice deletion therefore reports retained shared-control metadata and does not claim policy-complete all-copy erasure. Application input files, downloaded public samples and externally copied files are caller-owned, outside managed artifact cleanup.

# Backup, restore and retirement

`backup --actor operator` stores a private digest-verified package with PG manifests/dependencies, typed slice Parquet and original source bytes. Each slice's backup deadline derives from the immutable approved origin, not backup creation. Shared packages expose earlier slice deadlines and remain a residual until their last deadline. `prune-backups --actor operator` removes eligible managed packages with absence read-back. Expired copies cannot be restored merely because cleanup has not run.

`restore --actor operator --backup ID` verifies the current independent local journal/head and mirrored objects, rejects rollback/gaps/unknown dispatched effects, rebuilds missing PG authority schema if needed behind a busy gate, then replays current tombstones/revocations/holds before restoring only eligible evidence/projections. Missing originals and archives are reconstructed from verified backup bytes; absent current dependencies or corrupt evidence leave the operation closed. Restore never issues bills or resurrects deleted slices. A host-wide disaster recovery/SLO certification is not supplied by a journal on this host, even though its DB-rollback independence is tested.

The coordinator serializes every trusted legacy/new entry point using one host lock and PG policy locks. It records a durable intent before external effects. A killed coordinator, timeout, failed mirror or unknown dispatched request blocks subsequent operations and restore. Do not delete the journal, clear a busy flag, expire a lease or reinitialize around an unresolved operation. This implementation deliberately provides no automatic unknown-effect takeover. Actual route fencing/terminal reconciliation remains an operator recovery prerequisite requiring a separately reviewed case.

When **all** source slices/dependencies/holds/backups are released and active objects/CH physical parts are absent, `retire-preview --actor operator` produces a full source-cleanup manifest. Approve it with the distinct lifecycle approver and execute `retire --actor operator --manifest-digest DIGEST`. Unexpected local temporary artifacts or unknown versions block retirement. It durably denies the installation, commits PG source-payload removal and verifies it through a separate connection, removes generated bindings and historical financial journal snapshots, and retains only a minimal terminal deny anchor. This installation then permanently rejects activation/import/restore. Completion covers the enumerated managed application payloads; physical media erasure, engine WAL remnants and unmanaged copies are not certified. Never use source retirement to delete another namespace or shared verification volume.

# Verification evidence

See [HB-27 lifecycle validation](../../validation/reports/HB-27-lifecycle/summary.md). Test-only Python API clock injection supplies deterministic boundary cases; the CLI uses the server clock. Runtime roles receive no maintenance deletion privileges. Scenario actor names and source registration are trusted local operator choices, not authenticated financial approvals.
