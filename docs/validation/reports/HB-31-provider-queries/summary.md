---
type: VerificationReport
title: Six-provider FOCUS billing query verification
description: Records actual provider and service query results against independent source controls and the limits of future ingestion validation.
---

# Result

Tracking: HB-31. Accountable owner: Seung Woo Park. Machine: mac_mini. Base: merged PR26 commit `8153975c19fe27e32569e5416ef5f9298362ec08`.

**PASS:66 query scenarios through142 actual CLI commands**, including five expected rejection commands. **PASS:16 pure profile/physical-type probes and2 oracle unit tests.** These are separately scoped results, not a full product regression or provider conformance certificate. The default sample's complete journal/status is equal before and after the query run, at sequence58. No database reset, collection/publication change or product source modification was performed.

| Provider | All sample rows | Authorized company rows | Represented services | Query result | Source class |
| --- | ---: | ---: | ---: | --- | --- |
| AWS | 942 | 942 | 24 | PASS | Observed public sample |
| Microsoft Azure | 51 | 51 | 6 | PASS | Observed public sample |
| Google Cloud | 115 | 115 | 4 | PASS | Observed public subset |
| Oracle Cloud Infrastructure | 7 | 7 | 3 | PASS | Observed public sample |
| Alibaba Cloud | 4 | 3 | 1 | PASS | Deliberately synthetic |
| VMware | 4 | 3 | 1 | PASS | Deliberately synthetic |

All1123 source rows are accounted for: company view1121 across north-a1112 and north-b9, foreign company2. Identical company display labels do not grant cross-company access. All39 represented service filters, provider aliases, daily/monthly buckets and full paging agree with independent original source values. This covers these represented services, not every supplier catalog entry.

# What was actually compared

The standalone runner invokes the normal lifecycle CLI and its current PostgreSQL/ClickHouse/S3-backed control/query path. It reads existing published data; it does not re-ingest a newly downloaded provider export. A separate oracle has no production decode/control/semantic imports. It reads the pinned original union and each of the three retained monthly CSV deliveries, verifies their byte hashes against current delivery descriptors and proves their lexical row union matches the1123-row pack.

Every returned page is checked against full-filter controls. Every row's window/batch/file-local ordinal, tenant UUID/key, original lexical source dictionary and normalized money/time/scope fields must agree with its original. Row multiplicity and page identity checks prevent loss, permutation, duplicated pages or cross-tenant substitution. Exact decimal arithmetic uses100-digit precision with arithmetic rounding/inexact traps and integer coefficients. NULL-present/missing counts are compared separately from actual zero; unit prices are checked per row, not summed. Provider/service/currency/consumed-unit/pricing-unit boundaries remain separate.

Tenant reporting periods independently localize ChargePeriodStart to Asia/Seoul or UTC. September billing-window and single-day-overlap selections both return1005 company rows, demonstrating the documented delivery-window-overlap contract rather than row clipping/proration. The exclusive October boundary returns1 row; the disjoint August range returns0. Missing/wrong snapshots, changed provider/audience snapshot pins and an unauthorized importer query reject as expected.

Five offline dimension summaries derived from verified returned source_json agree with independently derived source summaries: provider/service/billing account and period; service/subaccount; charge category/class; service category; resource/SKU. These are test analytics, not newly implemented native SQL endpoints or proof of invoice/discount agreement. The [specification map](specification-map.md) links the official use cases and identifies their additional predicates/dimensions.

For a concrete money check, authorized company BilledCost totals retain separate currencies: AWS USD18.0066386184; Azure USD1.97651418586; GCP USD3.108526; OCI USD0.53707392473; VMware USD2.5; Alibaba USD1.25 and CNY11. No conversion or cross-currency total is calculated. These are public/synthetic sample values, not customer bills.

# Profile and transport results

The actual stored sample remains exploratory FOCUS1.0, with no consumed semantic context attached. Twelve synthetic1.2/1.3 provider cases separately prove declared-version provider-column selection and distinct host/publisher/invoice-issuer roles. They are generic contract probes, not vendor exports or database integration tests for those versions.

Two synthetic probes preserve the observed GCP sample's lowercase `usage` spelling and confirm strict1.2/1.3 profiles reject it; the original1.0 sample is not relabelled or rewritten. Two more representative physical-schema probes confirm fail-closed rejection of a DOUBLE Parquet money column and a nested extension. Those last probes PASS as rejection guards while exposing real compatibility limits; they do not claim actual AWS/GCP export compatibility.

# Evidence and review

- Final query run: `.local/runs/HB-31-provider-queries-third/{commands.json,summary.json}`.
- Pure profile results: `.local/runs/HB-31-provider-profiles.json`; PyArrow reports local CPU-discovery sandbox warnings but the probe command exits0 and all16 assertions pass.
- Oracle command: `python3 tests/focus_import/provider_queries/check_oracle.py`,2 tests PASS.
- [Machine-readable evidence](evidence.json) pins command/profile/source hashes, all scenario outcomes, aggregate values and historical attempts. Raw responses and public account/resource identifiers are not committed.
- [Plan and scope](../../../work/HB-31-provider-focus-queries/brief.md), [source/run instructions](specification-map.md), [future collection validation review](validation-review.md).

The first attempt failed because the test oracle treated removing zero fractional digits as rounding. The second failed because it assumed union-global ordinals survived monthly partitioning, although the source multiset already matched. Both were test-oracle errors; they were corrected and independently re-reviewed before the final full run. They are retained as failed attempts, not product defects or partial passes.

Independent DBA design and corrected-code review, primary-source content review and preliminary QA/change review report PASS. The validation applicability report also passes independent DBA/change review. Final DBA execution-evidence review, independent QA offline replay and actual nine-file staged-diff review report PASS. Reviewers did not rerun database operations; oracle unit tests were executed by the writer. No GitHub CI success or submitted GitHub review is inferred from these local reviews.

# Future collection verdict

Reuse the tested invariants as bounded ingestion acceptance checks; do not hardcode this pack's39 services or1123 rows as production rules. Existing admission, semantic, storage, confirmation/publication and query guards cover much of that chain. The [validation review](validation-review.md) identifies the remaining acceptance conditions: an explicit reviewed version/capability profile; supported exact transport; complete-window/multipart evidence; correct cumulative replacement semantics; and independent supplier controls for completeness or financial reconciliation.

Current results do not certify full FOCUS applicability, native supplier connectors/exports, actual OCI/VMware export versions, economic-line uniqueness, supplier-side completeness, invoice reconciliation, production authentication, scale/HA or installation readiness. No real customer files or cloud credentials were used. Product remains a customer-operated installation under the existing MIT license.
