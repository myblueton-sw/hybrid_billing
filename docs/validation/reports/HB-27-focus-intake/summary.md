---
type: ValidationReport
title: HB-27 FOCUS-first CSV and Parquet intake
description: Records exact common-file ingestion across declared versions, real storage checks and legacy sample recovery.
---

# Result

Implemented bounded CSV/Parquet intake through the existing tenant-aware FOCUS projection. Registry-selected versions 1.0, 1.2 and 1.3 share the same registration, independent confirmation, publication, query and lifecycle path. This is exploratory ingestion, not complete FOCUS conformance. Owner/accountable assignee: Seung Woo Park. Machine: mac_mini. Tracking: HB-27, PR24. [Work brief](../../../work/HB-27-focus-intake/brief.md); [operating contract](../../../ops/installation/focus-collection-lifecycle.md); [hashed evidence](evidence.json).

# Verified behavior

- New Parquet manifests pin the detected format; original bytes retain their proper suffix. Legacy manifests default to CSV without reinterpretation.
- Exact decimal/integer inputs preserve 18-place costs, credits and zero. Floating point is rejected. Typed NULL remains NULL with distinct provenance. All source columns are preserved as lexical strings/null, backed by the original physical file.
- Input/schema/type/expansion checks occur before ingestion writes. Metadata limits and one-row iteration reduce allocations but do not create an OS-enforced memory sandbox.
- Registered-only inbox collection accepts both formats. Version is caller-declared, never inferred from compatible headers or filename.
- Synthetic per-version CLI tests execute describe/register/run-due/confirm/publish/query/replay/backup/restore against real PostgreSQL18, ClickHouse26.8 and S3. Independent expected totals are company30, north10 and south20. Test assertions also inspect 18-place detail, missing values, tenant provider separation, original bytes, archived query results and recovery of a removed original object.

# Executed checks

Focused run `20261023T141451Z-edd757`: 20 PASS (17 pure + 3 integration). Full run `20261023T141645Z-ed7780`: **217 PASS (158 pure + 59 integration)**, zero failures/errors/skips. The preceding sandbox attempt `20261023T141601Z-4117ab` had 158 pure passes and 59 setup errors because localhost connections were denied; the authorized-access rerun passed. Raw JUnit/text logs remain in ignored `.local/runs/<run>/`; hashes are recorded in evidence.

Default-runtime run `20261023T144022Z-744a94` executes six successful CLI commands: restore the existing graph-inclusive CSV backup, query the company and all three tenant actors, and read lifecycle status. Counts remain company1121, tenantA1112, tenantB9, foreign2 (1123 overall); journal sequence56. This proves old-format recovery with the new code. It does not turn the existing public sample into a newer-version source export. Raw command results remain ignored and operator-managed.

# Independent review

Separate DBA design review, actual-change DBA review, architecture/security change review and focused QA report PASS. Full-regression and restored-runtime QA read-backs also report PASS. Reviews are local independent reviews, not GitHub-hosted approvals or remote CI checks. Root owns edits and integration; reviewers performed no DB mutations.

# Remaining boundaries

Only public FOCUS1.0 data and synthetic newer-version inputs are verified here. Full consumed-column applicability/conformance, live CSP export/API adapters, provider-side completeness, destination timestamp-range certification, production authentication/scale and multi-host recovery remain outside this increment. VMware remains an on-prem/private-cloud source. MSP post-invoice settlement remains deferred. No private customer documents, source rows, account-bearing runtime manifests, secrets, raw backups or journals are committed. PR24 remains unmerged; HB-27 is not Done.
