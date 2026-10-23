---
type: Validation Report
title: Official FOCUS data stored and queried by tenant
description: Records successful exact storage and scoped queries of unchanged official public demo rows with synthetic tenant properties and independent reviews.
status: draft
---

# Tenant-aware official demo execution

Tracking: [HB-27](https://linear.app/hybrid-billing/issue/HB-27). Owner: Seung Woo Park. Machine: `mac_mini`. Root owns implementation and actual runtime execution. [Evidence manifest](evidence.json), [reviewed plan](../../../work/HB-27-tenant-focus/brief.md), [runbook](../../../ops/installation/tenant-focus-data.md). The owner's acceptance is actual official-demo storage and query with tenant properties. The [earlier rejected intake](../HB-27-official-demo-intake/summary.md) is historical, not the final implementation outcome.

## Actual result

**PASS:** unchanged official FOCUS1.0 public CSV -> preserved shared original -> full exact assignment/validation -> immutable CH facts -> independent digest/control checks -> explicit synthetic confirmation -> atomic PG publication -> tenant-scoped detail pages and aggregate queries. CLI run `20261023T063824Z-f67dc9`: all18 commands exit0. `fetch-demo` freshly obtained755423 bytes with SHA-256 `e91e5ac7edf01ed2c9d926f37ef7dc1ae2aae97956fea8da6c9ee488b1c2839e`. The complete pinned1000-row artifact is processed, not a silently truncated subset of the large full dataset.

| Synthetic tenant | Actual source rows | Source providers | Reporting timezone | Retention metadata |
| --- | ---: | --- | --- | ---: |
| demo-a | 942 | AWS942 | Asia/Seoul | 180days |
| demo-b | 58 | Microsoft51, Oracle7 | UTC | 365days |

These assignments are synthetic owner-test configuration, not evidence of real customer ownership. Server-provisioned73 exact provider/account/subaccount/billing-period domains form a disjoint complete source partition. Reporting currency USD is metadata; conversion and retention execution are explicitly false. The importer preserves declared1.0, lexical original values/NULL reasons, exact Decimal(38,18) and source ordinals. No enum repair, source-account replacement or1.2/1.3 relabelling occurs.

CLI traversal retrieved all942 unique A ordinals and all58 unique B ordinals; their combined coverage is1000. Costs and quantities were independently compared to Fraction sums of original lexical values after paging. Per-tenant four-cost exact coefficients/missing counts are retained in evidence; quantities remain dimensionally grouped by exact units. B preserves7 missing contracted-cost values; all-missing groups remain NULL. Cross-provider filters under the wrong tenant return0; service filter returns matching authorized rows. Detail/aggregate verification checks CH typed values, per-row digest and widened exact CH sums. Unit prices are never totaled.

Published batch `18b2fb2c-aab7-4806-a837-6aa5de5217eb`, manifest SHA-256 `d70b0f7942b5b4e218daa1b4cd8028d374a96aed73b6bbc843379897cb30d643`. Identical CLI demo replay returns this same batch without inserting facts or changing current pointers. Services are deliberately left running with this queryable sample. The existing hb_local synthetic sample is preserved; this implementation does not modify its modules/schema.

## Tests, reviews and corrections

Root executed `tests/engine_probes/.venv/bin/python -m pytest tests/focus_import -q` with JUnit output in run `20261023T063643Z-870c19`: **37passed**, zero failures/errors/skips (24pure +13actual PG/CH/S3 integration). Tests include2000 exact-number generated Fraction-oracle cases; dynamic mapping dispatch; complete snapshot admission; provider-qualified identical account identity; NULL provenance; source-original equality; role-bound PG/CH negatives including empty tenant; source-composite FK negatives; confirmation/revision/policy/mapping denials; partial insert/unknown blocking; all-tenant publication rollback; binding-file tamper; policy withdrawal serialization; CH excessive grants; revision replay and batch-pinned pagination.

Independent DBA and security/architecture design reviews: PASS before code. Initial actual-change review: FAIL. Root corrected partial snapshots replacing full-source coverage, source pointer/predecessor FKs, reader binding-file authority and policy withdrawal during materialization. Focused independent re-reviews: PASS. Inert CHECK-constrained lock_token column privileges permit row locks without policy/property edits. Source and processing/profile checks repeat before verified state. Final independent DBA, security and QA read-back: PASS; all11 source hashes, JUnit37 and CLI18 successful exits match. Reviewers independently checked complete942/58 ordinal coverage and exact Fraction controls; QA had earlier independently run23pure tests before final additions. Final reviewers performed read-only code/evidence checks; root owns DB-mutating test execution.

Earlier sandboxed runtime attempts failed at local-service access, not at successful ingestion. The first authorized integration suite found persisted CH row policies after DROP DATABASE in test setup:7failures/22passes; fixture teardown now removes only its disposable namespace policies. That test setup failure was not repaired by weakening runtime policy checks. Final37-case execution above supersedes it. No new dependencies or environment secret rotation occurred in this increment.

## Bounds and remaining work

This is actual stored/queryable official public data under a local exploratory profile. Tenant selection is trusted operator tooling, not deployed user authentication. The new generic mapping registry labels1.0/1.2/1.3 exploratory/query-only;1.2/1.3 tests transform in-memory copies and do not certify real provider exports. Existing strict synthetic-path results remain distinct. No billing, pricing, issuance, FX or financial completeness acceptance is inferred.

Current immutable provisioning pins an exact complete snapshot, assignment/property revision and expected domain multiplicity; changed source bytes or configuration require a reviewed transition. Local bounds16MiB/10000rows/Decimal(38,18) with checked absolute integer coefficient totals below10^38 are not production provider limits. Full5.48M-row ingestion, live collectors, complete customer/source management, API/UI/authentication, retention/deletion/recovery, quarantine/re-request dispatch, HA and capacity remain future scope. The unchanged original contains multiple tenants and stays coordinator-only; readers receive no shared original key/download capability.

Raw CLI commands include public row identifiers and remain ignored/local-only; only aggregate summary, source hashes and artifact digests are committed. No persistence guarantee is made for raw artifacts. Attribution: FOCUS Sample Data by FinOps Open Cost and Usage Specification, a Series of Joint Development Foundation Projects, LLC; [pinned official source](https://github.com/FinOps-Open-Cost-and-Usage-Spec/FOCUS-Sample-Data/tree/adbdd17a132984d6e8583c149c236d2199c3f5bc), [CC BY4.0](https://creativecommons.org/licenses/by/4.0/). Original source bytes are unchanged; this report is derived execution metadata.
