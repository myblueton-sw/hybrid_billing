---
type: OperationsPlan
title: Reload pinned FOCUS samples for local preview
description: Recreates an isolated query-only sample environment and verifies the authenticated console path.
status: draft
---

# Scope and authority

[HYB-336](https://linear.app/myblueton/issue/HYB-336), owner Seung Woo Park, machine mac_mini. The owner explicitly requested reloading the previous FOCUS dataset. Existing dataset, credential configuration, Python environment, HB containers and HB volumes were absent; reader/storage ports were closed. Root owns execution/evidence; separate reviewer checks source lineage, scoped reads and provisioning sequence. Existing missing-companion waiver applies.

# Plan

Use a new `hb336-preview` Compose project, existing pinned PostgreSQL/ClickHouse/SeaweedFS definitions and alternative loopback ports 25432/28123/28333. Do not touch other projects or attempt to recover an unknown namespace. Generate only local ignored synthetic credentials; keep values out of Git, output and documentation. Install the locked Python dependencies.

Download the exact checksum-pinned official FOCUS 1.0 data and reconstruct the same six-provider pack: 1115 public rows plus 8 explicitly synthetic VMware/Alibaba rows. Keep downloaded files/provenance outside Git. Use existing validated initialization, lifecycle registration, ingestion, confirmation and publication tools with local scenario roles. These are synthetic operational approvals, not human financial acceptance. Do not fabricate current provider collection.

Connect loopback readers 8806/8810 through the existing Go session gate and shared-secret boundary; enable the console's gated proxy. Verify total 1123 source rows, authorized company 1121 rows, currencies kept separate, provider coverage and actual browser query success after normal sign-in. No reset/deletion/test suite that clears the loaded namespace. Record startup and handoff instructions, evidence and any limitation in a reviewed PR. No merge/deployment authorization is inferred.

# Independent review constraints

Separate reviewer PASS: initialize an empty installation, activate monthly lifecycle, then describe/register/collect/confirm/publish each actual billing-month partition. Do not first ingest a legacy snapshot or repeatedly register the full pack. Expected authorized counts: north-a 1112, north-b 9, company-north 1121, foreign 2. Preserve all source occurrences and provenance. Runtime journal was checked absent before creation. Synthetic received-time policies use hot 30/total 365/grace 0/backup 30 days and reject-before 2020-01-01; these are preview scenario settings, not legal retention guidance. No lifecycle deletion/retirement or destructive fixtures run.
