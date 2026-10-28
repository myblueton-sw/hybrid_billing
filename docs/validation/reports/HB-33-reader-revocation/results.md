---
type: SecurityValidationReport
title: Reader SQL revocation and restore verification
description: Records the I17b correction, isolated engine regression results and remaining customer authentication boundaries.
---

# Verdict and scope

[HB-33](https://linear.app/hybrid-billing/issue/HB-33/enforce-reader-sql-revocation-and-preserve-denial-through-restore) fixes the direct internal SQL revocation failure recorded by HB-32. Thirteen isolated engine cases, thirteen pure revocation tests and eight existing selector regressions PASS. The base behavior was reproduced: after product revocation, existing and new PostgreSQL readers and reused ClickHouse credentials still returned three rows. This report does not certify external customer access; that work is tracked in [HB-34](https://linear.app/hybrid-billing/issue/HB-34/implement-authenticated-customer-query-and-export-boundaries).

The [plan](../../../work/HB-33-reader-revocation/brief.md) defines ownership and boundaries. Root implemented the core and ran engine verification; a separate implementer wrote the isolated harness; an independent DBA/security reviewer reviewed design and source changes and independently ran the thirteen pure tests. Accountable owner Seung Woo Park; machine mac_mini. Base is merged PR28 at `79866f4388b8052d64443b974e4d1eec9cedacf9`.

# Changed behavior

Revocation durably records denial intent before enforcing engine access. PostgreSQL removes schema/table/column privileges and terminates affected sessions only in the current database. ClickHouse removes fact SELECT, installs a false row policy and synchronously cancels active queries for the verified generated reader identities. Success requires both engine steps and zero remaining ClickHouse queries. An engine error leaves a pending operation; same-target retry resumes it. A verified adjacent journal/checkpoint interruption can be reconciled without repairing unrelated pending operations.

Tenant revocation also denies every containing company audience completely; unrelated tenant readers retain access. Direct audience revocation leaves member readers unchanged. Ordinary initialization and both existing-schema and missing-schema restores receive durable denial before provisioning; denied readers never receive protected grants. Independent review found an existing-schema restore gap during implementation, which was corrected and re-reviewed before final verification.

# Actual verification

Final run 4 used PostgreSQL18.6, ClickHouse26.8.33.10 on loopback18129, database `hb33_reader_revocation_20261028_r4`, bucket `hb33-reader-revocation-20261028-r4` and unique generated tenant/view identities. Namespace and role preflight passed. No default or HB-32 namespace was reset. Default runtime status/bindings hashes, sequence58 and shared ingest/approve role attributes/memberships were preserved.

| Cases | Observed outcome |
| --- | --- |
| R01–R02 | Injected ClickHouse failure left durable pending denial and blocked product reads; retry completed and an independently observed active fact-reading query returned QUERY_WAS_CANCELLED. |
| R03–R05 | Preopened prepared tenant/audience sessions and a repeatable-read transaction terminated with PG57P01. Fresh protected queries returned PG42501 and ClickHouse access denial. Product access was denied. |
| R06–R08 | A2 and B retained exact row counts1 and2 across SQL/product paths. Duplicate revocation and ordinary reinitialization preserved denial. |
| R09–R10 | A pre-revocation backup did not restore authority. A separately old-style-revoked company-B reader first demonstrably retained two SQL rows; existing-schema restore then denied existing/new view credentials without denying the B tenant reader. |
| R11 | Dropped only the verified owned HB-33 schema, restored the old backup and confirmed tenant/dependent/direct audience denial plus unaffected reader access. |
| R12–R13 | Injected checkpoint failure after durable intent and after durable completion. Normal loading rejected the one-entry mismatch; same-target retry reconciled. A different target could not repair pending intent. |

Commands used the pinned `tests/engine_probes/.venv/bin/python`: `tests/focus_import/reader_revocation/verify.py --phase <phase> --run-number 4` for seed, baseline, verify, missing-schema and checkpoint, in that order. The missing-schema phase intentionally destroys only the proven owned test schema. Do not rerun seed or baseline on a completed namespace. `-m pytest tests/focus_import/test_reader_revocation.py -q` passed13 tests; `tests/focus_import/msp_isolation/check_selector.py` passed8; `git diff --check` passed. No CI execution is claimed.

The [evidence manifest](evidence-manifest.json) records final cases, core/source digests, artifact hashes/lengths and preservation observations. Every final phase matches the five current core source digests. The checkpoint runner's path variable was corrected after verify/missing-schema succeeded; individual phase artifacts retain their actual harness digests. Raw backups, bindings, credentials and runtime files are not committed.

Final independent source/evidence review PASS: the reviewer matched all ten current source hashes, all seven retained artifact hashes/lengths, all five completed phase records and the thirteen final engine cases. The reviewer did not rerun database operations. No blocking discrepancy remained.

# Earlier attempts and limitations

Earlier namespaces were preserved rather than reset. Initial baseline observations needed Decimal-to-integer serialization in the harness. The first cancellation implementation incorrectly treated the driver's command summary as a result table; it now reads raw JSON and handles a successful empty response only with a zero-process check. Test probes then exposed a missing JOIN alias, the engine's per-block sleep bound and a too-short query that completed before cancellation under load. The final bounded20-second probe was independently timed, observed active, and actually cancelled. The final checkpoint dispatcher initially referenced an undefined path variable; its corrected phase passed. These attempts are not counted as successful full runs. Their ClickHouse containers are stopped with data retained; final run4 remains available for review.

Cancellation is limited to exclusively provisioned generated internal readers with the validated fact-table grant profile. Current grant inspection does not prove the historical authority of arbitrary shared users. Already delivered/completed responses cannot be revoked retroactively. Failed non-revocation restore operations can remain unresolved and fail closed; this change is not a general restore recovery redesign. No real customer accounts, remote deployment or public authentication endpoint was used.

External identity, customer-safe projection, portal/search/cache/export/download and queued-worker authorization remain unimplemented/unverified here. The existing [identity design](../../../architecture/security-boundaries/identity-session-permissions.md) leaves the initial provider/profile and session/step-up bounds unresolved. HB-34 records those required inputs; no authentication policy was silently selected.
