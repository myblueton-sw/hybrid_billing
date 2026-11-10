---
type: VerificationPlan
title: Collected sample intake baseline before screen refinement
description: Maps the complete plan's intake requirements to a non-mutating source storage query and screen verification sequence.
status: draft
---

# Owner priority and scope

[HB-149](https://linear.app/hybrid-billing/issue/HB-149/verify-collected-sample-ingestion-before-screen-by-screen-refinement) starts from main 8e10d9d. The owner requires the complete product plan to govern screen-by-screen refinement, with onboarding deferred. First establish whether the already collected public/synthetic sample correctly survives original intake, normalization, storage and authorized query; only then design/connect screens using that baseline. Root owns the report and runtime verification. A separate read-only DBA/security reviewer checks the plan, source meaning and fresh results. Machine: mac_mini.

At initial inspection, S03 showed 2 wholly attributable receipts. This is not proof of the complete source inventory or complete ingestion. Historical evidence reports 1123 source rows, 1121 authorized company rows, 1112/9 member slices and 2 foreign rows across three monthly deliveries. Those are prior observations to recheck, not current successes. The mixed-ownership receipt-exclusion explanation remains a hypothesis until current manifests/windows and facts agree.

# Planning trace and observations to verify

| Requirement | Current verification target | Boundary |
| --- | --- | --- |
| HB-05 source/capability | Pin public versus deliberately synthetic sample provenance, represented providers/services and source version | No live connector or upstream completeness certification |
| HB-06 immutable originals/idempotency | Compare source/delivery byte hashes, row multiplicity, current publication lineage, stored originals/Parquet controls and query identities | No new ingestion/replay/reset or economic-line deduplication claim |
| HB-07 FOCUS meaning | Preserve source lexical rows, exact decimal coefficients, NULL versus zero, currency/unit and timestamp/reporting-period boundaries | Exploratory 1.0 remains 1.0; source cost is not a customer charge |
| HB-34 asynchronous outcomes | Distinguish receipt, verified/sealed publication, current facts and unsupported expectations | No worker confirmation or production job-success inference |
| HB-39 large-scale processing and performance budgets; S03/S04 | Explain receipt counts versus authorized fact counts and demonstrate the real GET/UI path | Bounded correctness prerequisite only; scale/load gates UNVERIFIED |

# Execution and preservation

1. Use the verified existing Docker storage ports and prepared Python environment. The old 8806 GET reader was stopped; restore that existing fixed-reader process for the current 8817 UI without changing source data or proxies. Keep 8810 retained metadata separate.
2. Record private before/after hashes of all retained control files and adopted local sample files. Do not log credentials, environment values or public account/resource identifiers into Git. Keep all old verification artifacts unchanged.
3. Reuse HB-31's independent lexical/decimal oracle and bounded correctness cases. Its historical execute method records per-command seconds, so do not run it unchanged. A local wrapper replaces only command execution/recording with the same assertions, no timing collection and no raw command-response or account/resource-identifier recording; use a new output directory and record both wrapper/original code hashes. Allowed CLI commands are only status/query. No A/B or per-query benchmark is run.
4. Verify the three retained monthly CSV byte pins and lexical union, each fully paged company/member/foreign query, provider/service and period cases, exact amount/quantity/price/NULL controls, page identity and expected permission/snapshot rejections. Do not infer current pass from the historical report.
5. Compare actual protected overview counts/scope to receipt metadata and registered connection metadata. Inspect the mixed-delivery reason without disclosing excluded whole-file fields to the company UI. Restore no hidden fixtures.
6. Keep S03 visibly open for owner inspection. Verify the actual screen's count/meaning and the downstream sample-read endpoint. Record UI/API shortcomings separately from stored-data correctness; implement no onboarding or material redesign in this baseline step.

# Completion and next screen sequence

Publish an English PASS/FAIL/UNVERIFIED report with immutable source/code references, actual current results, independent review and limits. The first screen refinement should be S03 receipt/completeness meaning using the verified baseline, then S04 source rows/field details and source reconciliation before higher-level cost/allocation/billing screens. Each later screen still needs its own scoped contract, design/copy review and rendered verification. No screen FIX, production readiness or permission/financial action follows from this audit. A discovered data defect blocks dependent claims rather than triggering a reset or data patch.

# Discovered local runtime repair

Fresh query failed before any fact comparison: host ClickHouse HTTP closed the connection while the container's internal ping succeeded. Docker inspection showed ClickHouse configuration and SeaweedFS startup bind mounts still referenced the removed HB-85 preview worktree; the mounted ClickHouse configuration directory was empty. Root's earlier worktree cleanup missed these active container references. This is a local environment failure, not evidence of corrupted source data.

After independent review, recreate only the existing SeaweedFS and ClickHouse service containers using the canonical compose plus overview-port override. Retain project name, image pins, environment, localhost ports and the existing named data volumes; do not run down, remove volumes, initialize, repair journals or reimport. Verify mount sources and volume names, then rerun correctness in a fresh output directory and compare all preservation hashes. The first failed run remains evidence.
