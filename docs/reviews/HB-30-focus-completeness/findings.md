---
type: ReviewReport
title: Common FOCUS ingestion completeness audit
description: Identifies missing common collection controls and confirmed defects in the bounded installation-based FOCUS implementation.
---

# Conclusion

**FAIL for complete common FOCUS ingestion.** The merged implementation provides a tested exploratory CSV/Parquet projection, storage/query path and local lifecycle controls. It does not yet provide complete version-aware, independently scoped, omission-detecting common ingestion. Earlier conversational wording that common collection was complete was too broad. The 217 previous passing tests remain valid evidence for their assertions; they do not establish omitted requirements.

Tracking: [HB-30](https://linear.app/hybrid-billing/issue/HB-30). Baseline: `0bc7907beda435c968bfde945043b1974f46c7a5`, containing reviewed PR24 head `dec73f8cf4db708132ca2aa9b81fb214edc32813`. Root owns integration and documents. Independent DBA and temporal QA reviews challenge the broader completion claim. Customer-operated installation is the product delivery model; no hosted SaaS control plane is required by this audit. Internal company/tenant authorization still applies. Billing SaaS-origin inputs is distinct from hosting Hybrid Billing as SaaS.

# Scope status matrix

| Area | Current evidence/status |
| --- | --- |
| Version-selected CSV/Parquet projection and exact NULL/decimal handling | Implemented bounded exploratory path; prior217 tests apply to tested assertions |
| Tenant/company queries and local archive/backup/restore | Prior bounded verification; no new engine test in this audit |
| Independent connection/dataset/window accumulation | Missing common capability; F01 |
| Expected account/calendar coverage and missing-versus-zero status | Missing common control; F02 |
| Version-specific consumed semantics | Partial; common19-field projection is not a complete applicability gate; F03/F06 |
| Fair collection of registered inbox artifacts | Confirmed starvation defect; F04 |
| Safe rejected receipt/admitted quarantine and replacement resolution | Missing in common path; F05 |
| Finality/as-of reopening policy | Unspecified control; F07 |
| Production installer, live connectors and production recovery readiness | Not certified by this audit; outside implementation changes |

# Prioritized findings

## F01 — P1: independent source files share one replacement window

**Missing product capability, reproduced in actual registration plus modeled selection.** Window identity hashes only start/end/grain ([temporal.py:31](../../../modules/focus_import/temporal.py#L31)). The single source journal indexes this key, compares one revision sequence and replaces one current pointer ([collection.py:44](../../../modules/focus_import/collection.py#L44), [55](../../../modules/focus_import/collection.py#L55), [105](../../../modules/focus_import/collection.py#L105)). It does not identify a registered connection/dataset/economic coverage scope as required by the [physical design](../../architecture/data-ownership/focus-physical-schema.md).

Reproduction: register AWS September revision1 and model it sealed; an independently complete Microsoft file for September revision1 is rejected `delivery_revision_or_pending`. Revision2 is accepted into the same window with the AWS delivery as predecessor. Modeling the normal new current pointer, actual `selected_slices` selects only the Microsoft batch. No engine publication/deletion was executed. Current behavior therefore requires an externally combined all-provider snapshot per window; it cannot independently accumulate ordinary provider files. The installation also chooses one intake grain globally, rather than one per connection.

Minimum resolution: stable connection/dataset/coverage identity in windows, revisions and pointers, or an explicit verified multipart assembly contract. Provider display names alone are not sufficient identity. Validate independent same-period sources and different source grains before claiming CSP-independent common collection.

## F02 — P1: missing accounts and periods are not detected

**Missing common control.** File row counts and domain digests are calculated from observed input; validation checks present rows, not an independently expected account universe ([temporal.py:75](../../../modules/focus_import/temporal.py#L75)). A finalized north-only file is accepted with a two-tenant account configuration. Operator completeness assertions remain intentional; the missing control is evidence distinguishing omitted accounts from explicit zero-usage corrections.

A separate pure probe models published September1/3 and registered September2. `selected_slices` returns only the two published windows ([collection.py:113](../../../modules/focus_import/collection.py#L113)); query coverage is assembled only from selected slices ([163](../../../modules/focus_import/collection.py#L163)). Never-registered dates cannot be enumerated without an expected calendar. Missing, late, pending, unsupported and known-zero are not a complete status model. This falls short of [period/account coverage checks](../../architecture/platform-data-build-design.md#L64) and the [detail completeness requirement](../../architecture/usage-detail-lifecycle.md).

Minimum resolution: expected accounts/windows, evidence-backed explicit-zero status, due times and received/validated/published coverage watermarks per connection. Keep intentional empty complete corrections possible. Do not claim this audit observed DB row deletion or an upstream omission.

## F03 — P1: version-specific consumed semantics are not connected to common intake

**Documented exploratory limit; incomplete as product collection.** The common registry maps19 source columns in every declared version ([model.py:23](../../../modules/focus_import/model.py#L23)). Compared with the separate synthetic consumed registry, common1.2 omits9 mapped semantic columns and common1.3 omits11: charge category/class, service category, SKU price identity, commitment status, invoice issuer, publisher, resource and tags;1.3 additionally lacks independent legacy-provider and host-provider fields. They may remain in `source_json` but are not typed, validated or available as consumed semantics. See the [documented/proposed first-path consumed-column design](../../architecture/data-ownership/focus-consumed-columns.md).

Seven mutated synthetic1.3 inputs pass common `decode` but fail the separate strict parser: missing BilledCost, negative unit price, price-product mismatch, quantity without unit, missing SKU price identity, lowercase charge category and missing required host for this fixture. The common parser preserves exact values/NULL and exploratory classification; it does not establish applicability/conformance PASS. The strict parser is a different path and has additional defects below, so simply connecting it is not a complete resolution.

Minimum resolution: one explicitly reviewed consumed-field/applicability registry per declared version/capability, including absent-by-version reasons and restrictions on dependent functions. Keep raw exploratory intake distinct from product/financial eligibility. Full provider conformance certification and supplier invoice reconciliation are separate judgments.

## F04 — P1: registered inbox file can starve indefinitely

**Confirmed local behavior defect.** `run_due` sorts and truncates the directory to100 entries before checking registered digests ([tools/focus_lifecycle.py:28](../../../tools/focus_lifecycle.py#L28)). With100 earlier unregistered files plus registered `zzz.csv`, two real scheduler-function calls both return no collected files, one registered file waiting and no awaiting publication; collection is never invoked. Repeating the scan does not advance. The test uses in-memory inbox/read/access doubles; no external I/O was attempted.

Minimum resolution: a durable fair scan position or direct registered artifact lookup, bounded per cycle without repeatedly excluding the same pending suffix. Include stale/unregistered files and more than100 entries in regression acceptance. This finding applies even to the bounded local workflow.

## F05 — P1: common rejected receipts and resolution workflow are absent

**Missing common contract.** Invalid projection/delivery input raises before creation of a batch or original-object preservation ([service.py:65](../../../modules/focus_import/service.py#L65), [101](../../../modules/focus_import/model.py#L101)). The common schema has no rejected receipt/quarantine/re-request state; `tools/focus_lifecycle.py` has no such command. A separate `focus_local` synthetic path has limited quarantine, which does not close the common path requirement. The gap includes safe rejection metadata and admitted-unusable evidence handling; it is not a requirement to store every rejected payload. The [build design](../../architecture/platform-data-build-design.md) calls for immutable rejected/quarantined receipt evidence, bounded replacement authorization, preserved predecessor and explicit escalation.

Minimum resolution: installation-scoped receipt disposition with an admission boundary. Unauthorized, prohibited or out-of-allowed-scope input must be rejected with safe metadata only; do not preserve its raw content merely because it failed validation. For admitted but semantically unusable input, retain the immutable original in restricted quarantine and provide reviewed replacement/re-request transitions. Protected originals follow authorized retention and disclosure controls. External notifications need their own authorization; no messaging is performed or implied here. This is source inspection, not a new runtime failure test.

## F06 — P2: valid source scope and strict parsing still need review

**Confirmed parser outcomes, with standard applicability supplied by the pinned design.** A source row with NULL SubAccountId is rejected `required_source_scope`; every mapped source header is mandatory ([intake.py:20](../../../modules/focus_import/intake.py#L20), [model.py:111](../../../modules/focus_import/model.py#L111)). The consumed design marks subaccount attribution conditional. Account-only charges need explicit authorized mapping without fabricated subaccounts or whole-payer access.

The separate strict parser accepts `ServiceCategory=not-a-category`, Tags arrays and duplicate JSON keys. Tags numeric scalar `0.1000000000000000001` becomes `0.1`; duplicate keys collapse to the last value ([focus_local/contract.py:225](../../../modules/focus_local/contract.py#L225), [1.3 registry](../../../packages/contracts/focus/1.3.json)). This is metadata normalization loss in the separate strict path, not observed billed-cost loss in the common exact-decimal path. Both parsers also accept `1E+2`; common1.3 accepts a naive space-separated timestamp. These need declared-version format gates, separated from intentional legacy-import tolerance.

Minimum resolution: conditional scope attribution, pinned enum/format checks, unique scalar Tags object validation and lossless numeric handling. Validate the exact pinned specification predicates rather than promoting all synthetic-profile checks to universal rules.

## F07 — P2: finality/as-of regression has no explicit reopening decision

**Policy gap, not a claim that provisional data is forbidden.** Actual `register` with an in-memory journal accepts a higher revision that reduces completeness from a finalized month through October1 to nonfinal through September10 ([collection.py:55](../../../modules/focus_import/collection.py#L55)). Revision and sealed-predecessor checks pass; no explicit reopening/correction policy is consulted. Existing provisional views are intentional.

Minimum resolution: decide and enforce whether lower as-of/unfinalizing revisions need an explicit authorized reopening reason, preserving the original completion history. Avoid silently representing a correction as unchanged source finality.

# What remains valid and what was not checked

- Existing bounded tests establish version-selected CSV/Parquet projection, exact Decimal(38,18) coefficient checks, NULL preservation, current publication, tenant/company authorization, hierarchy snapshots and tested local archive/backup/restore behavior. No new regression claim follows solely from broader missing requirements.
- UTC calendar day/month intake, whole-window replacement, charge-start reporting in tenant timezone without proration and delivery-window overlap filtering are implemented limitations. They do not provide arbitrary billing calendars or a precise usage-time range filter.
- Input and full-selection limits remain10000 rows. Pagination does not certify large historical selection/export capacity. Current tests do not establish the installation production volume target.
- A supported installer/upgrader, installation authentication and service supervision, actual external source completeness, supplier invoice reconciliation, production writer fencing/recovery and physical erasure certification remain outside tested acceptance. They are self-hosted readiness requirements, not requests to turn this repository into SaaS.
- Live CSP connectors, private customer workbooks/PDFs and MSP post-invoice settlement were excluded. No private customer inputs or live CSP credentials were accessed. No database, default runtime, local services or sample1123-row contents were changed by this audit.

# Reproduction and review evidence

Run from the repository using the existing pinned environment:

```sh
tests/engine_probes/.venv/bin/python -B docs/reviews/HB-30-focus-completeness/reproduce.py
```

The committed [reproduction source](reproduce.py) executes21 synthetic observations without DB connections; [observed results](probes.json) retain exact case/input hashes and distinguish fake-journal registration/modeled selection from engine effects. These are audit observations, not21 product acceptance passes. The fixture baseline is `tests/fixtures/focus/synthetic-1.3.csv` via the existing strict test helper; its SHA is recorded in results. Existing217-test evidence was inspected, not rerun. Source and evidence hashes are recorded in [evidence.json](evidence.json).

Independent DBA review concludes FAIL for full common coverage and PASS for the documented bounded subset. Independent temporal QA confirms F01/F02/F04/F07 with pure probes. Independent README/licensing review passes installation wording and unchanged MIT terms. Final independent report/content review and21-observation reproduction read-back report PASS; this audit does not fix findings or certify installation readiness.
