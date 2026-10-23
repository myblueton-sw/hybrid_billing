---
type: Validation Report
title: HB-27 official FOCUS demo inventory
description: Records exploratory profiling of the exact official composite FOCUS 1.0 artifact and independent decimal controls before defining the supported 1.2 and 1.3 consumer schema.
status: draft
---

# Official demo inventory

Tracking: [HB-27](https://linear.app/hybrid-billing/issue/HB-27). Owner: Seung Woo Park. Machine: `mac_mini`. Observed on 2026-10-23 by the separate `focus_inventory` researcher. See [manifest](evidence-manifest.json), [profile](profile.json), [independent controls](independent-controls.json), [provenance](provenance.json) and [checks](checks.json).

The official [sandbox](https://focus.finops.org/docs/tools-assets/tools/query-sandbox/) links to [FOCUS Sample Data](https://github.com/FinOps-Open-Cost-and-Usage-Spec/FOCUS-Sample-Data/tree/adbdd17a132984d6e8583c149c236d2199c3f5bc). The full artifact is `FOCUS-1.0/focus_data_table.csv.gz`, 496,305,507 bytes, SHA-256 `c198cda74cbde4b11b9be289815bda7e9ac62042f1d023ad9bc8fbf7bff1018c`, matching its Git LFS object. It is a composite 1.0 example, including AWS preview input, not a supported 1.2/1.3 export. Use it for exploratory profiling only; product ingestion requires a separately reviewed adapter or a supported-version fixture. No implicit version expansion, fixture adoption or converted original is claimed.

Attribution: FOCUS Sample Data, FinOps Open Cost and Usage Specification, a Series of Joint Development Foundation Projects, LLC; [source and revision](https://github.com/FinOps-Open-Cost-and-Usage-Spec/FOCUS-Sample-Data/tree/adbdd17a132984d6e8583c149c236d2199c3f5bc), [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/). Original downloaded bytes were unchanged. This report contains derived aggregate observations, not copied source rows. The [official origin account](https://www.finops.org/insights/focus-sandbox/) identifies anonymized AWS/Google/Oracle inputs and Microsoft toolkit sample data. No downloaded data rows are committed.

## Observed coverage

| ProviderName | Rows | Distinct services |
| --- | ---: | ---: |
| AWS | 5,181,336 | 63 |
| Google Cloud | 115 | 4 |
| Microsoft | 270,492 | 36 |
| Oracle | 36,416 | 14 |
| Total | 5,488,359 | 117 |

All observed service names/counts are in the profile. Datadog occurs in 273 AWS rows with ServiceName `Datadog` and PublisherName `Datadog Inc.`; this proves demo presence only, not native Datadog export or external collection. The 1,000/10,000 subsets omit Google; the 100,000 subset includes only two Google rows, so the full artifact is the coverage baseline.

Strict UTF-8 CSV decoding passed. All 44 lexical headers are recorded, including nonstandard unprefixed `Id`; CSV supplies no physical decimal types. Both empty strings and literal `NULL` occur. Costs reach observed scale 11, consumed quantity scale 15 and list unit price scale 17; these observations cannot set universal policy bounds. Per-provider periods, row duration and identifier cardinalities are recorded without identifier values. Google covers March 2024; Microsoft September 1–19; AWS September; Oracle September charges with September/October billing periods. Completeness/finality and provider revision ordering remain unknown. The sample `Id` is not an established provider economic identity.

## Exact control totals

These are sums of present USD values, not complete invoice or selling-price truth. Missing counts below are within USD only.

| Measure | Exact USD sum | Missing USD values |
| --- | ---: | ---: |
| BilledCost | 103121.71404690744 | 0 |
| ContractedCost | 73348.46173154618 | 36,189 |
| EffectiveCost | 74923.43740661209 | 0 |
| ListCost | 105016.12219995464 | 259 |

227 Oracle rows have empty currency and remain a separate group: their present costs are zero, ContractedCost is missing in all 227. Total missing ContractedCost is 36,416. Neither an all-missing sum recorded as zero nor a present-value sum establishes completeness. 115 Google rows use lowercase `usage`; no Correction ChargeClass was observed. These gaps are not silently repaired or described as 1.2/1.3 conformance.

Two standard-library scans used Decimal precision 100: rowwise addition versus distinct lexical decimal value times frequency, grouped independently by provider/currency/measure. Row counts and all 20 cost groups matched. The manifest pins original bytes, method, script hashes and aggregate artifacts; scratch scripts and source bytes have local-only retention, while source URL/hash allow reacquisition. Root has not independently rerun the full scans.

## Acceptance limits

Profiling and control comparison: PASS. Full specification validation, product DB ingestion, marts, selling-contract/tenant mapping, live connectors, writer fencing, policy outages and capacity: NOT_RUN. The current [demo plan](../../plans/HB-27-focus-demo-baseline.md) still governs fixture adoption. The [mapping](../../../architecture/data-ownership/focus-consumed-columns.md) distinguishes official version semantics from these observed 1.0 values.
