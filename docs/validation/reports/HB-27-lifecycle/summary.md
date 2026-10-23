---
type: Validation Report
title: HB-27 recurring FOCUS collection and tenant lifecycle execution
description: Records actual bounded daily/monthly accumulation, isolation, archive, deletion-failure and latest-control backup recovery evidence.
status: stable
---

# Result and review boundary

Tracking: HB-27. Owner: Seung Woo Park. Machine: mac_mini. Root owns implementation, integration and actual engine/CLI execution; independent DBA, architecture/security and QA review are separate from root verification. Design initially failed on missing temporal/authority/lifetime boundaries, then passed after refinement. Actual-change review found missing-original/archive restore, journal rollback, backup-clock reset, re-archive retry, retirement ordering/inventory and restore preflight defects; these were fixed and independently re-reviewed. No GitHub-hosted review or CI run is asserted.

Final full suite `20261023T105939Z-12d7cb`: **122 passed**, zero failures/errors/skips. This comprises77 pure/model/configuration cases and45 actual PG18/CH26.8/S3 integration cases. Regression run `20261023T100923Z-028cfe` reproduced four failures before the important recovery fixes; it is historical FAIL evidence, not a pass. Intermediate120/119/17-case runs do not replace the final execution.

# Observed behavior

- Two complete daily windows accumulate, a correction changes only its window, explicit empty corrections clear an old window, and replay/stale/conflicting revisions cannot double count. Monthly cumulative refresh replaces its predecessor. Overlapping/granularity-incompatible registrations reject; equal source row occurrences remain distinct.
- Registry-declared mappings remain dynamic. Pure cases cover1.0/1.2/1.3 and a temporary test registry entry; actual public sample ingestion is1.0. Whole-selection exact coefficient bounds, currency/unit separation, NULL provenance, timezone/DST attribution and composite snapshot paging are tested.
- Actual archive/recall/re-archive validates Parquet schema/bytes/hash/row digest/count and exact dimensional controls; copy/tier changes preserve origins. Corrupt archive bytes deny access. Holds, references and changed policies invalidate stale deletion approval; tenant A deletion preserves B's shared source and exact rows.
- Actual archive/delete effects followed by synthetic lost acknowledgement remove the selected CH rows, preserve the other tenant, leave an independently reopened pending intent/delete-committed tombstone, and block query, recall, restore and legacy initialization. No lease-based takeover or fabricated successful deletion follows.
- Backup clocks derive from immutable policy origins, including per-slice shared-copy deadlines. Expired unpruned backups reject restore. Latest tombstone, hold and revocation survive old-checkpoint/schema restoration. Required originals and archived Parquet are restored from verified backup bytes; missing/corrupt dependencies fail closed. Invalid backup preflight cannot poison a missing-schema recovery.
- Source retirement rejects untracked temporary files/backups/holds/physical-part residuals. Tests verify a successful empty-source cleanup removes PG/config and historical financial journal payload, retains a minimal permanent deny anchor, and blocks reinitialization/restore. Late transaction failure or interrupted compaction leaves a pending independent deny, never a terminal removal receipt.

# Actual sample CLI and final runtime

Baseline run `20261023T105442Z-88bde1`:42 successful commands downloaded/prepared/ingested/confirmed/published and queried the1123-row six-provider sample.

Periodic run `20261023T105825Z-e7c355`: **67 successful CLI commands**. It registers and publishes March2024=115, September2024=1007 and October2024=1 rows, repeats September as a new revision, exercises registered inbox/watch, archive/recall/re-archive, actual metadata/data backup restore, filtered queries and every current page. Current total remains1123. Original lexical-row multiset equals the unchanged derived six-provider input; original provider samples are not silently rewritten. The company union1121 excludes foreign2. All six additive measures and missing counts match independent Fraction calculations within exact currency/unit groups. Company filters: AWS942, Azure51, GCP115, OCI7, VMware3, Alibaba3. VMware/Alibaba remain explicitly synthetic.

The periodic CLI deletion preview had **zero eligible targets**; no destructive CLI deletion is claimed. Destructive effects and failure boundaries were exercised by the45 actual engine integration cases.

After the final122-case suite reset the disposable `hb_focus` namespace, the actual maintenance CLI restored backup `1697dd80-edf1-4219-9d23-8532cf9de4f1` under latest journal sequence44. Four final CLI queries and a status readback verified north-a1112, north-b9, foreign2, company1121 and unchanged complete grouped controls; all retained slices are hot. The default runtime journal and backup were preserved through isolated test journals. The original `hb_local` namespace was untouched. Services and this known sample remain available.

# Reproduction and limitations

Use the [collection/lifecycle runbook](../../../ops/installation/focus-collection-lifecycle.md). Run the suite with `tests/engine_probes/.venv/bin/python -m pytest tests/focus_import`; it deliberately resets only the disposable `hb_focus` PG/CH namespace. Preserve the latest independent journal and verified backup first; restore through the documented maintenance command afterward. Never bypass an unknown operation or remove a journal to reseed. Test clocks are explicit Python API inputs; CLI deadlines use server time.

[Evidence JSON](evidence.json) records run IDs, artifact/source hashes, exact counts and scope. Raw source rows, generated account-bearing config, private catalog/journal packages and command output stay ignored/local. The committed inputs remain synthetic fixtures plus aggregate/hash provenance.

This is bounded trusted single-host execution, not deployed IAM, native/live cloud collection, FOCUS financial conformance, a production UI/API, real billing/ERP or scale certification. Queries are limited to10000 selected rows and explicitly use delivery-window overlap for their optional interval; reporting uses charge-start timezone attribution without proration. Unknown external effects require separately reviewed actual fencing/reconciliation. Local independent controls prove DB-rollback resistance, not host-disaster redundancy. Shared control/PG evidence and backup copies remain explicit per-tenant deletion residuals until governed source retirement. Managed-payload cleanup does not certify physical disk/WAL erasure, object-store internals or caller-owned/exported copies. The complete product requirements remain broader than this local path.
