---
type: Review Report
title: HB-27 local execution design change review and QA
description: Records independent reviews, fixes, actual local execution evidence and the remaining product boundaries for the first dynamic-version FOCUS path.
status: draft
---

# Local implementation verification

Tracking: [HB-27](https://linear.app/hybrid-billing/issue/HB-27). Accountable owner: Seung Woo Park. Machine: `mac_mini`. Root owns all code/doc edits and integration. User authorized local implementation and dynamic supported-version behavior on 2026-10-23. Correct workspace/team/project and unclaimed machine were verified; mac_mini was claimed before implementation. Existing HB-27 branch/PR 23 is reused; no duplicate ticket or production permission was inferred.

## Independent sequence

DBA `physical_design_review` and security/architecture reviewer `pr22_review` reviewed the [execution plan](../../work/HB-27-local-path/brief.md) before executable changes: PASS after account/period source admission, exact local aggregate bounds and intentional mapping-fanout clarifications. That acceptance did not certify code.

Initial actual change review: **FAIL**. Root corrected and obtained actual re-review for:

| Finding | Correction / final evidence |
| --- | --- |
| Received unusable input discarded | Immutable quarantine originals/receipts and scoped hold, no fact load or publication; full-file authorization admission before any write |
| Early parse failure concealed later off-scope rows | Independent complete bounded account/subaccount/period scan for known/unknown versions; repeated under reservation/current capability locks |
| Quarantine upload completion lifted business protection | Persistent unresolved quarantine blocks confirm/publish/query/export; no automatic resolution |
| Capability changed between admission and storage | Complete current capability comparison and re-admission before PUT; injected race proves no write and unknown reservation |
| Unit prices incorrectly added | Registry sum/none flags, no price totals, per-row checksum, exact integer product/nonnegative checks; conservative local host applicability |
| Runtime-required registry fields/types missing | Complete runtime core validation before side effects, four costs require sum, unsafe/unknown predicates and ambiguous fields reject |
| Existing role/profile/ACL bypass | PG elevated attributes/membership/ownership/effective privileges checked, exact restricted grants, expected policies repaired; CH grants and effective locked settings checked, extra policies fail |
| Scoped constraints / NULL controls | Composite decision FK; all-missing sum NULL, present/missing counts, direct populated-table CH cross-tenant assertion |
| Accepted NULLs bypassed by tests | Positive Tax/Purchase/Unused/Correction cases for both versions and actual nullable CH/Parquet roundtrips |

Final independent DBA change review: **PASS**, no blocking findings. Security/architecture actual change review: **PASS**, no blocking findings. Separate QA `focus_inventory`: **PASS** after its initial missing-NULL coverage FAIL was resolved. QA read-back confirmed 49 passed (36 pure, 13 integration), zero failures/errors/skips; all 14 recorded source hashes match; CLI replay release/export equality and 2/0 tenant query counts match. Reviewers did not mutate DBs or rerun the integration suite. Root executed the actual runtime checks; QA had independently run the earlier 28-case pure suite before re-review.

## Actual execution and limits

Final independent staged-scope review: **PASS**, exactly 25 scoped files; all staged bytes match reviewed working files and all 14 recorded source hashes match. No unrelated assets, raw logs, environment files or generated data are staged; credential/token scans and staged whitespace checks passed. A fixture-local attributes file preserves exact CRLF CSV bytes. The owner-requested repeat run `20261023T051133Z-3b474d` again passed all 49 tests and CLI replay/tenant checks; its digests and current sample release are in the linked validation report.

[Run 20261023T045921Z-c60f96](../../validation/reports/20261023T045921Z-c60f96/summary.md) records 49 passing tests and actual CLI versions/demo/replay/query commands against local PostgreSQL/ClickHouse/SeaweedFS. Original bytes, exact source/row/CH controls, sealed CAS, immutable export and Parquet type/null/count/hash checks passed. A test-only extra registry version proves data-driven dispatch without modifying runtime version branches. Production support is not inferred from it.

Six local generated secrets and derived application credentials were rotated after a failing-test representation exposed values; redacting repr and CLI error suppression are implemented. Rotation/restart/authenticated read-back is root evidence; no values are recorded. This does not claim historical session output was erased.

Runtime code is scoped to `modules/focus_local/`, `tools/local_focus.py`, one local migration and version-contract JSONs; fixtures/tests are synthetic. Existing engine probes and their historical results stay unchanged. No new dependency was installed; the existing pinned probe environment supplies the local driver dependencies. Each source/test file is below 1,500 physical lines and contains no added explanatory comments. Working-tree and staged checks must exclude unrelated local baseline assets and generated/raw data.

[Runbook](../../ops/installation/local-focus-path.md) states exact commands and local identity/profile limits. This implements a bounded local path, not the complete physical schema or product. Actual rebuild import, quarantine resolution/re-request, true fenced recovery/deletion, deployed policy/access services, user authentication, live providers/ERP, customer data, HA/performance and financial approvals remain outside this increment. HB-27 stays In Progress. Publication/read-back and machine handoff are recorded in the ticket after pushing the reviewed commit; no merge is claimed by this report.
