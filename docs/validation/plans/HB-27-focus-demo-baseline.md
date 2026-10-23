---
type: Verification Plan
title: FOCUS demo dataset baseline within HB-27
description: Consolidates the owner-requested demo-first data review into the existing physical design and local verification work without confusing sample evidence, synthetic probes and live provider collection.
status: draft
---

# FOCUS demo dataset baseline

Tracking: [HB-27](https://linear.app/hybrid-billing/issue/HB-27). Accountable owner: Seung Woo Park. Machine: `mac_mini`. Owner clarification: 2026-10-22. This plan extends the existing [engine probes](HB-27-engine-probes.md) and [platform data build design](../../architecture/platform-data-build-design.md); it does not create a parallel implementation ticket.

## Decision and scope

The first representative data review and end-to-end fixture baseline must use the actual FOCUS demo dataset. Specification descriptions, provider adoption lists and generated examples do not establish what the selected demo contains. Inspect and inventory the selected artifact before deriving its physical schema, expected aggregates or coverage claims.

The product scope includes CSP, SaaS and other technology billing sources, explicitly including Datadog. Inventory all providers and services actually present in the selected demo. Maintain a separate coverage backlog for requested providers or services absent from it. Neither demo presence nor FOCUS ecosystem participation proves that a working external connector or a native FOCUS export exists.

The HB-26 decisions remain: a FOCUS consumer, ClickHouse as the bulk engine, no embedded official validator, and checks of consumed fields. Product support for FOCUS 1.2/1.3 is distinct from the demo artifact's declared version. The earlier 1.0 sample reference is a candidate, not an inspected fixture or automatic authorization to expand supported versions. If the selected sample is outside the supported set, record the mismatch and decide explicitly whether to use it only for exploratory profiling, implement a reviewed compatibility adapter, or select another artifact. Never relabel converted or generated data as an original provider export.

## Required artifact inventory

| Evidence | Record before fixture adoption |
| --- | --- |
| Provenance | Official source URL, exact file/object path, revision or download date, SHA-256, byte size, license and required attribution; whether anonymized, synthetic or another documented origin |
| Dataset identity | Declared FOCUS version and dataset type, provider/service values actually observed; keep observed values separate from inferred aliases |
| Physical representation | File format, compression, encoding, column names and physical types, decimal precision/scale, null and empty representations, custom fields |
| Grain and coverage | Row count per artifact/provider, billing and charge periods, account/resource identifiers, observed row grain, overlap and available source/revision identity |
| Measures | Currency, consumed/pricing units, cost bases, charge categories and control totals computed with exact decimals; do not combine currencies or cost bases |
| Gaps | Missing fields and unsupported semantics, unknown completeness/finality, absent correction/commitment/credit cases, and no supplied tenant or selling-contract mapping |

Missing metadata stays unknown. Do not infer a FOCUS version from a filename alone or treat a sample's successful parse as full specification conformance. Preserve originals and distinguish any conversion output with its source digest and mapping revision. Do not commit downloaded rows until their rights and disclosure suitability are verified.

## Evidence classes and acceptance

| Class | Purpose | Required boundary |
| --- | --- | --- |
| Observed demo | Profile actual fields and values; derive expected source counts and per-currency totals; verify sample-backed ingestion, postprocessing and selected mart results | Claims name the exact artifact and consumed columns; only observed services and cases count as demo coverage |
| Synthetic supplement | Exercise absent duplicates, corrections, missing values, tenant mappings, selling contracts and lifecycle failures | Label generation recipe and expected result separately; never count synthetic coverage as provider sample evidence |
| Live connector | Later verify provider authentication, export/API access, pagination, scope, retry, incremental delivery and source completeness | Remains NOT_RUN until separately authorized and executed; public sample replay does not establish external collection success |

Review the actual demo inventory first. Then specify the consumed-column mappings and expected outcomes for collection, postprocessing and each selected mart using that inventory. Optional common detail remains optional; the demo does not impose a mandatory serial normalization pipeline. Supplier cost does not become customer selling price without the separate contract inputs.

Existing engine probes retain their synthetic scope and historical outcomes. They can inform safe loading and numeric behavior, but do not satisfy demo-backed ingestion or mart acceptance. Conversely, a demo run cannot replace engine edge cases, tenant isolation, retention/hold races, recovery or capacity tests.

## Status and handoff

This change records the owner direction and acceptance boundaries only. Exact demo artifact selection, download, row inspection, license verification, inventory, demo execution and live collection are NOT_RUN in this integration. Datadog presence in the demo is UNKNOWN. No provider coverage count, sample totals or runtime success is asserted.

HB-27 remains In Progress for physical design and verification. HB-26 is the completed build-design input; HB-24 owns the separate missing portable companion baseline. No duplicate FOCUS ticket was found in the current HB ticket title inventory, so the request is consolidated into HB-27 without closing unrelated tickets or merging their unfinished scope.
