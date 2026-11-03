---
type: ImplementationDesign
title: HB-93 bounded source-data inspection
description: Defines actual local S04 source-row inspection and separates verified metadata from warehouse integrity and production authorization.
status: draft
---

# Scope and ownership

HB-93 uses pending HB-142 as its explicit dependency. Root owns implementation and QA; planning_boundary_review independently reviewed authority and numeric boundaries. Its design verdict is conditional PASS, not implementation approval. The local synthetic approver is an inspection identity, not a production BL/CP/AU grant. No publication, export or original-object download is added.

# Contract

Every request enters the existing read-only inspection transaction and rechecks capability before returning. Candidate listing is bounded to 25. An exact release and manifest digest bind each 50-row page. The cursor is a stored ordinal, not a display index. Page queries use the tenant-scoped ClickHouse reader and parameterized scope, release, ordinal and limit. Aggregate ordinal count, distinct count and range must match the declared manifest count. Pending loads are unavailable. Manifest hash, registry, table, scope, file identity and policy revision are checked before warehouse access.

This pins metadata identity, not a transactionally immutable warehouse snapshot. Current warehouse content is not independently certified by this read. The UI states that limitation; S06 remains the explicit consistency-check action. Quarantine and unresolved operations remain visible blockers and never turn a source row into billing approval.

The response allowlist contains account/service/resource identifiers, charge interval, consumed/pricing quantity and units, separately named costs, currency and charge category. Arbitrary JSON, tags, raw storage keys and unknown future fields are omitted. Decimal values retain six-place strings; null remains null. No sum or FX conversion is performed.

# Screen items

| Item ID | Behavior |
| --- | --- |
| S04:scope:local-inspection | Actual fixed server scope, period and diagnostic-only status |
| S04:list:releases | Bounded received-release selection; exact deep links fail closed |
| S04:table:normalized | At most 50 rows, stored ordinal identity, separate billed/effective and units |
| S04:page:snapshot | Manifest-bound continuation; clear selection on page/release change |
| S04:detail:row | Canonical release plus ordinal; keyboard focus return |
| S04:state:unavailable | Failed/denied/stale reads clear protected data |
| S04:review:scope | Existing revision-scoped owner review; no automated FIX |

Use the canonical full-width shell, neutral borders, local fonts and EN/KO/AR layouts. Mobile cards retain exact amounts; a separate scrollable table exposes all columns. Cloudscape's [table view](https://cloudscape.design/patterns/resource-management/view/table-view/) was consulted for bounded pagination, clear loading/error states and full-page data hierarchy; no external styling or code is imported.

# Deferred and verification

Production permission mapping, cross-source filters/aggregates, conversion histories, AI fields, export, durable query snapshots and large-volume acceptance remain separate contracts. HB-94 owns safe S05 evidence access. A source-data page must not pretend that unconnected evidence links or exports work.

The scoped implementation and verification are recorded below. Full planning parity and owner design acceptance remain pending. macOS runtime evidence does not certify other operating systems.

# Implementation and review evidence

Implemented the scoped contract above. Source fields outside the allowlist are omitted. Profile classification is specific to S04. Exact direct links, 50-row continuation, row details and return focus are functional; the unrelated candidate renderer remains unchanged.

Independent backend source review: PASS (planning_boundary_review). Independent frontend review initially found the final ordinal boundary, incomplete response validation and detail-return behavior. A subsequent review found Escape interference with an open modal. All were corrected; scoped re-review: PASS (gap_review). A browser regression opens and cancels the owner-review dialog without confirming it.

Actual verification on macOS arm64, Node 26.7.0, Python 3.11.7 and Chromium headless shell 1243:

- Console suite: 64 PASS, including exact nullable response and 10,000th-row boundary cases.
- Source service suite: 8 PASS. Includes real isolated PostgreSQL/ClickHouse reads, cross-scope/loader denial, HTTP origin/method controls, stale/corrupt metadata, final authority recheck and unchanged release/publication counts. Mock warehouse cases cover 50+1 pagination and malformed ordinal sets; a 10,000-row production load was not run.
- Typecheck, production build, 242-file build-manifest hash verification and scoped prohibited-style scan: PASS. Existing large-bundle warning remains; no performance acceptance inferred.
- Browser QA: six EN/KO/AR desktop/mobile renders PASS; actual rows, detail focus, browser Back/Forward, Escape, modal isolation, denied-data clearing, malformed response, empty list and viewport overflow checked. Evidence is generated under `output/validation/HB-93`, excluded from Git.
- Claude CLI visual review identified timestamp wrapping, RTL value grouping and table/control typography. The scoped source screen was corrected and re-rendered. Final scoped visual re-review: PASS for the corrected issues. The review does not certify overall design quality or native-language acceptance.

No owner FIX, main merge, deployment, production authorization, independent financial approval or cross-OS execution is claimed. Shared shell code only selects the new S04 implementation and its accurate local-service banner; visual overrides are scoped to the live S04 screen. Other screen acceptance records are retained and are not automatically renewed.
