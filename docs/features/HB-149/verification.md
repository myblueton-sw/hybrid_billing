---
type: VerificationRecord
title: Collected sample baseline for screen refinement
description: Separates retained source integrity and authorized query correctness from receipt visibility and unverified production intake capabilities.
status: draft
---

# Scope and planning decision

[HB-149](https://linear.app/hybrid-billing/issue/HB-149/verify-collected-sample-ingestion-before-screen-by-screen-refinement) follows the owner's priority: use the complete planning baseline, postpone onboarding, establish the collected sample baseline first, then refine one screen at a time. Root owns execution and this report; a separate read-only reviewer checks data/security meaning and results. Source baseline: main `8e10d9dada03e31114d9e861603382bfc134ab5d`. See the [verification plan](plan.md).

This audit concerns already retained public/synthetic exploratory FOCUS 1.0 data. It performs no collection, import, initializer, reset, journal repair, permission/schema change or financial operation. A sealed sample publication is not certification of live provider collection or billable customer data.

# Retained originals and receipt visibility

Fresh operator-scoped, read-only manifest/S3 checks found three current sealed deliveries. Every original exactly matches its retained monthly CSV bytes, manifest hash, declared byte count and delivery row count.

| Delivery month | Original rows | Company rows | Foreign rows | Whole-file company receipt |
| --- | ---: | ---: | ---: | --- |
| 2024-03 | 115 | 115 | 0 | Visible |
| 2024-09 | 1007 | 1005 | 2 | Excluded |
| 2024-10 | 1 | 1 | 0 | Visible |
| Total | 1123 | 1121 | 2 | 2 receipts |

Before runtime repair, the actual S03 browser showed 2 receipts; after repair, the protected receipt GET returned 2. Current rendered verification is UNVERIFIED: two native recapture attempts failed with ScreenCaptureKit `-3811`. `receipt_metadata.legacy_candidate` and `legacy_owned` admit whole-file metadata only when every nonzero manifest slice belongs to the reader. Current September manifest ownership proves why its whole-file receipt is excluded. This preserves the foreign boundary; it does not establish whether permitted row queries succeed. Whole-file receipt counts and authorized fact counts are different measures. S01's protected registration GET returns 0, because these historical legacy deliveries do not constitute registered connections.

The combined 1123-row CSV, official 1000-row subset and Google 115-row subset match the previously committed HB-27 SHA256 pins. Providers represented are AWS, Microsoft Azure, Google Cloud, Oracle, VMware and Alibaba Cloud; VMware/Alibaba additions are deliberately synthetic fixtures. The company API provenance breakdown is 1000 official-public, 115 official-public-subset and 6 synthetic-contract-fixture rows; the two excluded foreign rows are synthetic. This is a sample authorization model, not evidence of real company ownership. The large upstream gzip is absent and was not downloaded or independently revalidated. Retained subset verification does not certify current upstream availability or completeness.

# Local runtime failure and repair boundary

The first fresh correctness run stopped after status succeeded and the first company query returned unavailable. The protected overview GET also failed. Direct host ClickHouse HTTP returned an empty reply, while its container-internal ping succeeded. Docker inspection found ClickHouse configuration and SeaweedFS startup bind mounts still referencing the removed HB-85 preview worktree; ClickHouse's configuration mount was empty. Root's preceding worktree cleanup missed these running-container references.

Repair completed using the tracked canonical compose file and overview-port override, recreating only the two affected local service containers. Inspection confirmed canonical bind sources and the same `hb-verification_ch-data` / `hb-verification_sw-data` volumes, pinned images and localhost ports. PostgreSQL was not recreated. No down/volume deletion, data initialization or reimport occurred. The initial failed result is retained separately from subsequent verification. All 85 source/control/previous-result files matched their before hashes after repair, independently rechecked by the reviewer. All three S3 originals were verified again.

# Query verification and storage boundaries

After restoration, protected overview GET returned 200 with 1121 authorized/selected rows and 3 deliveries; receipt GET returned 200/2 and connection GET 200/0. Root separately observed the same three routes through the actual console proxy returning 200 with those counts; the pinned API artifact records direct reader requests, and the reviewer did not independently rerun the proxy checks. Every response declares `financial_authority:false`; completeness remains `operator-asserted-unverified`. The restored correctness run passed all 66 scenarios / 142 status-query commands, including 5 expected rejections. Independent result review passed; Git integration is recorded in the ticket only after remote read-back. There are 15 hot slices and no archived slices. No Parquet artifact was created or read: archived Parquet verification is not applicable to this retained state and remains unverified as a storage capability.

The independent lexical/decimal oracle matched the three monthly CSV union to the pinned combined sample, then compared every returned row and exact numeric coefficient, grouped totals, present/missing counts, currencies/units and tenant-timezone periods. Full pagination preserved identities and row multiplicity: company 1121 rows/12 pages, member A 1112/12, member B 9/1 and foreign 2/1. Six-provider, service, daily and unknown-selection cases passed. Snapshot-required/changed and importer-denied cases rejected as expected. September window and mid-window overlap each returned 1005 company rows; the exclusive October boundary returned 1 and the preceding window returned 0. These are delivery-window overlap semantics, not row clipping or proration. Offline account/resource derivations were checked by digest; no new native analytics endpoint is implied.

Operator status before/after was identical at journal sequence 58. All 85 preserved file hashes were unchanged after the final query. The wrapper allowlisted only status/query and replaced the historical runner's execute method to avoid per-query timing and raw response logging; assertions still consumed full responses in memory. Old HB-31 results remain unchanged. The failed run and restored run use separate directories. No Graphify comparison or benchmark ran.

Curated [machine-readable evidence](evidence.json) pins the original runner/oracle, local wrapper and safe fresh summaries. Full local evidence remains under `.local/runs/HB-149-sample-intake-restored`; the prior failure remains under `.local/runs/HB-149-sample-intake-current`. Git includes counts, hashes and verdicts only, not source bodies, account/resource identifiers or credentials. This is a correctness result, not a throughput or scalability result.

# Independent review and document QA

The separate read-only reviewer passed the final result/diff review: six artifact hashes and runner/oracle/wrapper pins matched; 66 cases, 142 commands and 5 expected rejections agreed; eight curated queries and API/storage summaries matched their safe source artifacts; all 85 preserved hashes were unchanged; and the first failure remained present. The reviewer checked the three staged documents/curated JSON files for raw identifiers, source bodies, credentials and unrelated assets. The console proxy observation remains explicitly root-attributed. No expensive query suite was rerun for this review. Root checked evidence arithmetic, OKF types, affected local links and staged whitespace; all passed.

# Screen sequence and acceptance limits

Start with S03 collection status and reconciliation: distinguish receipts, permitted stored rows, source evidence class, completeness expectations and unavailable capabilities. Next use S04 source rows and field details to expose exact decimals, missing versus zero values, provenance and period meaning. Cost/allocation/billing screens follow that baseline; onboarding remains deferred.

Planning trace: HB-05 source/capability and HB-06 immutable-original/lineage checks are bounded to retained samples. HB-07 requires exact values, currencies/units, NULLs and period semantics. HB-34 asynchronous worker guarantees and HB-39 large-scale processing/performance budgets are unverified by this audit. No owner screen FIX, native-language acceptance, production authentication, live connector readiness or cross-platform runtime certification is claimed. Only macOS arm64 with the prepared local Python/container environment is exercised.
