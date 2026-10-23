---
type: Validation Report
title: Official FOCUS demo download and unchanged intake check
description: Distinguishes a successful fresh official FOCUS 1.0 sample download from rejected intake into the bounded 1.2 and 1.3 synthetic local path.
status: draft
---

# Official demo intake check

Tracking: [HB-27](https://linear.app/hybrid-billing/issue/HB-27). Owner: Seung Woo Park. Machine: `mac_mini`. Root applied QA and tech-writer responsibilities for this owner-requested verification on 2026-10-23. Scope is download, unchanged-file intake and evidence; no runtime code, policy, identity, registry or numeric-profile changes. [Execution evidence](evidence.json), [previous full inventory](../HB-27-focus-demo-inventory/summary.md), [local path](../../../ops/installation/local-focus-path.md).

## Source and actual results

The official [FOCUS sandbox](https://focus.finops.org/docs/tools-assets/tools/query-sandbox/) links to [FOCUS Sample Data](https://github.com/FinOps-Open-Cost-and-Usage-Spec/FOCUS-Sample-Data). The current main revision read through GitHub API is `adbdd17a132984d6e8583c149c236d2199c3f5bc`, unchanged from the earlier inventory. The repository exposes FOCUS-1.0; this check found no bundled 1.2/1.3 official fixture there.

| Check | Actual result |
| --- | --- |
| Fresh HTTP download | PASS: original `FOCUS-1.0/focus_sample.csv`, 755,423 bytes, SHA-256 `e91e5ac7edf01ed2c9d926f37ef7dc1ae2aae97956fea8da6c9ee488b1c2839e`, equal to pinned provenance |
| Strict UTF-8 CSV scan | PASS: 1,000 rows, 44 unique headers, no malformed row widths; AWS 942, Microsoft 51, Oracle 7 |
| Retained full artifact hash | PASS: previously downloaded gzip, 496,305,507 bytes, SHA-256 `c198cda74cbde4b11b9be289815bda7e9ac62042f1d023ad9bc8fbf7bff1018c`; not freshly downloaded or fully rescanned here |
| Installed version enumeration | PASS: actual CLI lists only 1.2 and 1.3; registry has no 1.0 entry |
| Official unchanged-file intake | REJECTED: actual CLI exits 2 with `source_scope`; official demo accounts differ from the seeded synthetic connection |
| Existing published sample | PASS: before/after query outputs are identical: sealed 1.3 release, two rows, unchanged controls and scope revision |
| Official demo DB ingestion/publication/export | NOT_RUN beyond rejected admission; no successful ingestion claim |

The executable command was `tests/engine_probes/.venv/bin/python tools/local_focus.py ingest --actor loader-a --version 1.0 --revision 2 --file .local/runs/hb27-official-intake-20261023/focus_sample.csv`. Query/versions/ingest/query exits and JSON results are retained in evidence. The initial sandboxed query returned dependency unavailable; the same scratch wrapper under authorized local-service access completed successfully. The download itself succeeded with curl. Raw artifacts and the scratch wrapper are local-only and not promised to persist; no official source rows are committed.

Independent read-only QA `focus_inventory`: PASS. The reviewer checked sample/full-artifact hashes, CSV counts, evidence digest, registry/header gaps, recorded CLI outputs, admission control flow, metadata/links and exactly two scoped staged files. HTTP/API retrieval and CLI execution remain root-recorded evidence; the reviewer did not rerun network/runtime calls or mutate databases.

## Interpretation and next boundary

Admission checks account/period authorization before version dispatch. Therefore the observed error is `source_scope`, not an executed `unsupported_version` result. Independently, installed registry inspection proves 1.0 is unsupported. Header comparison finds both `HostProviderName` and `ServiceProviderName` missing for 1.3; required-header presence for 1.2 alone proves no semantic compatibility. Do not relabel 1.0 input as 1.2/1.3 or replace source accounts/amounts to force a pass.

The full artifact exceeds current 8 MiB/10,000-row local bounds. Earlier profiling also observed precision beyond Decimal(18,6), literal NULL tokens and cross-account/period/provider data. These are additional design requirements, not defects repaired in this verification. A real official-demo ingestion increment needs a separately reviewed version adapter or explicit supported-version source, source capability provisioning and precision/size/multiplicity profile. It must preserve original evidence and establish provider/account/period partition controls before loading. Existing 49 synthetic-path passes do not certify this integration.

Attribution: FOCUS Sample Data by FinOps Open Cost and Usage Specification, a Series of Joint Development Foundation Projects, LLC; [pinned source](https://github.com/FinOps-Open-Cost-and-Usage-Spec/FOCUS-Sample-Data/tree/adbdd17a132984d6e8583c149c236d2199c3f5bc), [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/). Original downloaded bytes remain unchanged; this report contains derived counts and execution metadata only.
