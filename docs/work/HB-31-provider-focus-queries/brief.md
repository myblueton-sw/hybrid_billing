---
type: TestPlan
title: HB-31 provider billing queries against FOCUS use cases
description: Defines independent source controls and actual provider query verification without equating sample imports with live provider coverage.
---

# Goal and ownership

Ticket HB-31; accountable owner Seung Woo Park; root owns research, test/report writing and integration on machine:mac_mini. Base is merged PR26 commit `8153975c19fe27e32569e5416ef5f9298362ec08`. Separate read-only DBA reviews amount/unit/account/time semantics and test design/results; separate QA checks final evidence. No production code change is planned. Allowed edits: this plan, `tests/focus_import/provider_queries/` and `docs/validation/reports/HB-31-provider-queries/`. Existing user files and runtime controls must be preserved.

# Source and expected-value method

Read official FOCUS versioned use cases and supplier export guidance; distinguish FOCUS's common schema/query semantics from supplier-specific service catalog names and export availability. Record retrieved URLs and version-dependent fields, including provider/host/invoice issuer changes. Existing six-provider pack is declared exploratory FOCUS1.0; AWS/Azure/GCP/OCI rows are observed public samples, VMware/Alibaba rows are labelled synthetic. Do not relabel these as1.2/1.3 or claim all vendor service types are represented.

Build independent expected aggregates directly from pinned lexical source rows with Python Decimal and explicit NULLs, without importing production decode/controls/semantic functions. Compare full row-preservation digests, exact costs and quantities by provider/service/currency/unit/account/charge/billing and charge period. Keep incompatible currencies and units separate; no invented currency conversion, provider-specific invoice reconciliation or reallocation of charges across dates.

DBA design review PASS requires exact arithmetic with rounding traps, full page identity/multiplicity and immutable snapshot checks, tenant UUID binding, tenant-local periods and clear native/offline distinctions. Initial local attempts exposed test-oracle errors: Decimal's exact-integral conversion signalled Rounded for trailing zeros; original union ordinals were incorrectly assumed to be retained across monthly delivery partitioning. The corrected oracle uses non-signalling equality after trapped arithmetic and verifies each retained monthly CSV digest against its current delivery descriptor, then binds window/batch/original ordinal and tenant UUID. It independently proves the partition union equals the pinned1123-row pack, including lexical source fields. These are test fixes, not product or runtime mutations.

# Execution

1. Inspect runtime and source provenance; make read-only queries against restored local sample data. Capture sequence and publication state before and after.
2. Page actual CLI lifecycle queries for each provider, each represented service and permitted tenant/company audience. Verify aliases, empty/unknown filters, snapshot pinning, per-row tenant identity and foreign-tenant exclusion.
3. Compare provider/service/account/period results against independently computed source controls and exact raw field retention, including credits, purchases, adjustments or nulls only where actually present.
4. Map selected official use cases (provider/service charges, account attribution, daily/monthly cost, resource/SKU and charge categories) to fields and available product entry points. Report absent fields, missing server filters or insufficient sample coverage explicitly, never mark derived offline analytics as a native product feature.
5. Inspect existing1.2/1.3 consumed-profile tests; if additional runtime writes are necessary, first define isolated namespace/backup/restore procedures and obtain relevant independent design review. Default sample/runtime is not reset for read-only verification.

# Acceptance and boundaries

PASS means specified sample queries match independently derived source values in actual backing stores and retain tenant boundaries. FAIL means a reproducible disagreement; UNVERIFIED means missing source/coverage or unavailable runtime. Full provider conformance, every catalog service, native/live cloud collection, real invoice comparison, production authentication, scale/HA and MSP settlement are outside this verification. Final evidence includes exact commands, version/source hashes, outcomes and limitations; independent DBA and QA review precedes PR. Merge is not authorized by this new task.

# Collection validation follow-up

The owner additionally requests review of the future collection validation logic after OCI/VMware checks. Root traces existing admission, consumed semantics, warehouse verification, confirmation/publication and query checks. The report distinguishes generic controls suitable for future acceptance from this pack's provider/service/row-count assertions. Independent DBA/change review checks monetary and tenant-boundary conclusions. Record transport, upstream completeness and invoice-evidence gaps explicitly; no production validation implementation is silently added to this test/report ticket.
