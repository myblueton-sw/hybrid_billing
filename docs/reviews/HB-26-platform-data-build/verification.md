---
type: Verification Record
title: HB-26 platform data build design review record
description: Records owner decisions, external architecture review rounds and residuals for the build-level platform data design of the first FOCUS path.
status: draft
---

# HB-26 review record

Tracking: Linear ticket [HB-26](https://linear.app/hybrid-billing/issue/HB-26). Owner: Seung Woo Park. Machine: `mac_mini`. Branch `feature/HB-26-platform-data-build-design` from main `0bc39aa`. Date: 2026-10-22. Artifact: [platform data build design](../../architecture/platform-data-build-design.md).

## Owner decisions in conversation (2026-10-22)

- ClickHouse as analytical store and single bulk engine; DuckDB not used, including inside the FOCUS validator.
- The product is a FOCUS consumer; the official validator is not embedded; intake checks only what is consumed.
- Checks before collection and before publication; no ClickHouse staging database.
- FOCUS 1.2 and 1.3 supported through one internal model.
- Unusable received files are quarantined and re-requested.
- Replay and restore ordering must be guaranteed.
- Sealed releases are also exported to Parquet (option A), keeping the Parquet foundation.

## External architecture review

Two independent read-only reviewers: an external Codex reviewer and a Claude architecture reviewer that verified external facts by web search (ClickHouse decimal, null, row-policy and delete semantics; ClickHouse 26.8 LTS; Apache-2.0 licences; SeaweedFS 4.42 compatibility; OpenTelemetry ClickHouse exporter maturity; provider FOCUS 1.2 support).

| Revision | Codex | Claude architect |
| --- | --- | --- |
| 1 | FAIL (7 BLOCKING) | PASS-WITH-FIXES (7 BLOCKING) |
| 2 | FAIL (4 BLOCKING) | PASS-WITH-FIXES (2 BLOCKING) |
| 3 | FAIL (replacement path, journal tail, confirmation atomicity) | PASS-WITH-FIXES (2 BLOCKING) |
| 4 | PASS as build-level overview | PASS; one NON-BLOCKING journal commit-marker item applied by root afterwards and not re-reviewed |

## Residuals

- Implementation is gated on accepted physical designs, in particular the irreversible deletion boundary, writer fencing and policy-server outage behavior.
- Deferred detail is listed in the design's "Detailed in the physical design step" and open items sections.
- Supersession of the HB-20 Iceberg and Spark preference and of the "engine unselected" statements in the decision register is applied only after acceptance of this design.
- Unverified: ClickHouse sum behavior at the Decimal64 to Decimal128 boundary, excess-scale text parsing, ClickHouse Parquet DECIMAL export, conditional-create support in SeaweedFS and customer S3 services, provider Parquet money types, which providers export FOCUS 1.3.
- Nothing was built or executed; all behavior is NOT_RUN.
