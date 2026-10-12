---
type: Screen Specification
title: S02 Onboarding and owner handoff
description: Separate setup permissions from operational read access and verify evidence from first data through reconciliation and activation.
status: draft
sources:
  - resource: ../../../planning/v1.16/ui-design.md
  - resource: ../../../planning/v1.16/detailed-requirements.json
  - resource: ../../../planning/v1.16/provider-profiles.md
  - resource: ../../../contracts/commands/planning-actions.md
---

# S02 Onboarding and owner handoff

## Purpose, roles and provenance

Separate setup permissions from operational read access and verify evidence from first data through reconciliation and activation.

Intended audience: admin. Roles: OP, AM, BL.

HB-16 translates the complete prior local screen contract and integrates planning fixes. Original spec SHA-256: `fafdd44b5ee29c366df3720fe22aa600e11d3388d0f28a625b9a59e96f25d3f2`. Original design JSON SHA-256: `6f23a8d7e0d397ba05e0e1175f590eb84c581a7ca4a135be4783c51e40a50d26`. Original source references above remain authoritative evidence; root retains the raw source snapshot. This document does not declare all 49 screens or the entire planning baseline complete.

Requirements: HB-01, HB-04, HB-05, HB-07, HB-09, HB-25, HB-36, HB-41.

Acceptance references: AUD-DB-02, AUD-DB-05, AUD-DB-08, AUD-QA-07.



## Regions and reading order

Proposed layout: `wizard`.

### Progress stages

Check successful authentication separately from first data, completeness and reconciliation.

### Permission handoff

Separate setup owner, export writer and reader into three rows.

### Readiness

Distinguish no usage, first-generation wait, period restrictions and insufficient permissions.

### Design status

Static proposal; product unimplemented; API disconnected; tests not run.
Use synthetic amounts and data only.
AUD links: AUD-DB-02, AUD-DB-05, AUD-DB-08, AUD-QA-07

## Field contract

| Label | Type | Required when | Editability | Validation | Sensitivity |
| --- | --- | --- | --- | --- | --- |
| Operating type and connection | reference | Always | Initial edit | Connection scope matches business purpose | Contract |
| Setup owner | principal-reference | Always | Handoff edit | Currently authorized for this task | Principal |
| Temporary setup permissions | permission-set | Always | Plan edit | Least privilege and revocation time | Security |
| Service writer and collector reader | reference | Always | Read after validation | Revoking setup access must not remove the writer | Security |
| Estimated cost and data interval | money+interval | When applicable | Read only | Separate query and storage costs; show unknown values | Cost |
| Handoff expiry and reason | datetime+text | When applicable | Edit | Task limited; exclude secrets | Internal |
| Stage evidence and configuration hash | reference | Always | Read only | Matches current configuration revision | Restricted evidence |
| Metric cohort and version | reference+revision | When measurement is displayed | Read only except explicit command draft | Immutable cohort; unknown/censored distinct from zero; pause union counted once; no chosen thresholds | Restricted business evidence |
| Observation cutoff and fixed denominator | datetime+count | When measurement is displayed | Read only except explicit command draft | Immutable cohort; unknown/censored distinct from zero; pause union counted once; no chosen thresholds | Restricted business evidence |
| Connection and first verified billing rates | separate-rates | When measurement is displayed | Read only except explicit command draft | Immutable cohort; unknown/censored distinct from zero; pause union counted once; no chosen thresholds | Restricted business evidence |
| Raw and active duration with interval evidence | duration+reference-set | When measurement is displayed | Read only except explicit command draft | Immutable cohort; unknown/censored distinct from zero; pause union counted once; no chosen thresholds | Restricted business evidence |

Sensitive fields: Operating type and connection, Setup owner, Temporary setup permissions, Service writer and collector reader, Estimated cost and data interval, Stage evidence and configuration hash.

## Tabs, filters and columns

Tabs: Path selection / Setup and permissions / First data / First reconciliation / Activation review.

Filters: Stage status, Owner, Waiting reason, Provider.

Columns: Stage → Actor → Required permissions → Estimated cost → Completion evidence → Last check → Next action.

## Action contract

All identifiers below are proposed canonical planning IDs, not deployed APIs. Existing English permission tokens are preserved; descriptive permissions require the command catalog mapping, not an invented grant.

| Action | Proposed ID | Permission | Preconditions | Result | Confirmation |
| --- | --- | --- | --- | --- | --- |
| Start path | Existing action; catalog mapping required | Connection setup | Provider and contract selected | Create a stage checklist | None |
| Hand off owner | Existing action; catalog mapping required | Handoff permission | Target owner and expiry specified | Create a task-limited handoff draft | External delivery is a separate user action |
| Recheck affected stages | Existing action; catalog mapping required | Diagnostic permission | Configuration change scope confirmed | Create fresh evidence only for affected stages | Confirm cost and scope |
| Request activation review | Existing action; catalog mapping required | Activation request | First data, reconciliation and required gates satisfied | Move to approval review | Confirm unsupported scope |
| Inspect onboarding measurements | onboarding.metrics.read | Scoped onboarding read | Fixed cohort revision, policy and observation cutoff | Read separate connected and first-verified outcomes with fixed denominator | None |

## User-facing states

| State | Message |
| --- | --- |
| loading | Checking scope and permissions and loading Onboarding and owner handoff. |
| empty | Onboarding has not started for the selected connection. |
| partial | Only part of the data or work is ready. Show ready, missing, held, failed and unprocessed scopes separately; this is not full completion or zero. |
| blocked | Required reconciliation evidence is missing. Data receipt alone cannot activate the connection. |
| error | The request could not be completed. Check the request ID and failed scope, then choose a safe retry or handoff. |
| revoked | Access to this scope has changed. Remove sensitive details, previews and selections, stop affected actions and downloads, and query permitted scope again. Recall of externally delivered copies is not guaranteed. |

## Onboarding measurement contract

The proposed `onboarding.metrics.read` uses scoped onboarding-read authority, fixed cohort revision and observation cutoff. Display connection success and first verified billing success as separate metrics, with their own event evidence and numerators. Authentication or data receipt is not first verified billing. Each measurement cohort binds a canonical cohort ID, immutable target manifest/denominator, inclusion rule, cutoff/as-of time, timezone, metric definition version and policy version. Deleting failures, holds, unsupported or waiting targets or changing a screen filter cannot improve the original denominator; changed eligibility creates a new cohort/version with an explained comparison.

Show numerator, fixed denominator and count of unknown/censored targets. A zero denominator is not a zero-percent success rate. Before verified evidence exists, first verified billing remains pending/unknown. Distinguish raw elapsed duration from permitted active duration. If active duration is defined to exclude pauses, subtract the union of qualifying pause intervals clipped to the observation interval exactly once; overlaps cannot be double-subtracted. Missing starts, ends, observation cutoff or required state evidence produce unknown/censored values instead of zero or fabricated durations. Show incomplete cases separately from completed-duration summaries.

Metric cohort selection, inclusion/exclusion policy, thresholds, pause eligibility, target durations and customer defaults remain unresolved policy choices. The UI exposes definition/version and evidence, not a selected threshold or success claim. Acceptance (NOT_RUN): overlapping pauses counted once; cutoff changes create a new immutable observation; retry cannot add duplicate successes; failures remain in denominator; unsupported and unverified states do not become zero; connection-only success cannot count as first verified billing.


## Authorization, editing and navigation

This is a static design proposal. Product, API, persistence, login, external integration, issuance, deletion and delivery are not implemented or executed by this document. Buttons and states describe planned behavior; only synthetic examples are permitted. The [planning gate](../../../planning/v1.16/planning-gate.md) remains authoritative. [Design data](design.json) is a static artifact input, not a deployed API or executable UI. No user acceptance, accessibility, security, provider, ERP or performance test is claimed.

Roles identify intended users, not global grants. Recheck the authenticated installation/workspace, customer, legal entity, contract, field, period, current actor and session permissions on every read, command and download. UI filters cannot authorize access. Never disclose unauthorized object existence, counts, totals, autocomplete, filenames or costs. Operator status does not grant cost access; support viewing does not grant billing approval. Keep installation/workspace and permitted scope visible; technical canonical IDs belong in evidence detail while the main surface uses user terminology.

Conditional fields are mandatory whenever applicable. A JSON `required=false` permits nonapplicability, not skipped validation. Hide genuinely inapplicable inputs and explain missing required policy. Read-only values come from ledgers or validation results; the UI must not create or overwrite them. Money, exchange rates and quantities use decimal strings. Periods use inclusive start/exclusive end and the contract timezone. Preserve currency and meaning; never combine cost, sales or allocation amounts or different currencies without the separately permitted display policy. Exclude secrets from storage, logs, clipboard and examples.

Draft saving is an explicit user action; autosave and automatic recovery are not assumed implemented. Leaving edits offers retain/discard; a revision conflict requires comparison with the latest revision and a new draft. All writes check current authority, expected revision and idempotency key. Repeated clicks and retries reference the existing operation, and acceptance IDs are distinct from completion. A retry must be permitted under current state and authority, including access to any prior protected result. Confirmation shows target, period, revision, impact and the cancellable boundary; it represents the business contract, not an invented extra approval. An error summary links fields and focuses the first error. Success requires a result object or job state, not only a toast.

Use server search and stable sorting. Canonical IDs keep selection stable when display names change. Filter changes create a fresh snapshot and invalidate prior selections; preserve the same valid snapshot, filters and focus on return from details. Distinguish current-page selection from all matching targets. Within the same scope, tab changes retain filter/selection context and warn about unsaved drafts. Detail drawers include evidence/original references, before/after comparisons and command/job results. Hide inapplicable views or explain nonapplicability without fabricated values. Navigation passes only currently permitted canonical IDs, period, revision and snapshot/as-of references. URLs contain no secrets, raw personal information or sensitive amounts. Destination screens recheck permissions.

## State and failure interpretation

`partial` is readiness or a business-result classification, not a top-level Job state. Top-level jobs use `queued/running/waiting/succeeded/failed/cancelled`; unknown outcomes use `waiting` with a reason. Distinguish genuine zero, none, not aggregated, not measurable, unsupported, not applicable and unknown. Never replace unknown with zero. Revocation removes sensitive values, previews and selection caches and prevents further protected downloads; previously delivered external copies cannot be guaranteed recalled. Errors include stable nonsensitive code, request ID, field location, failed scope and safe next action, never secrets, customer source bodies or stack traces. Offer retry only where that stage is demonstrably safe.

## Volume, snapshots and concurrency

Plan for at least 100,000-row verification workloads, over one million incoming records per day and at least 30 million accumulated monthly rows. These are design premises, not upper limits or measured guarantees. Read bounded pages using server permission filters, stable sort and snapshot cursors; never load all source data into browser memory. Server totals use the same snapshot, currency and meaning; current-page totals are not overall totals. Show scope, revision, as-of time, completeness and aggregation delay, including different card timestamps.

Cursor expiry requires resynchronization and a new query. All-target actions preview and bind a server target manifest, allowed scope and count before acceptance. Scope, permission, target or relevant revision changes invalidate affected selections; never silently replace approved scope or results with new data. Screen filters cannot shrink shared-pool denominators. Calculation, diagnostics, export, restoration, full generation, inspection, cleanup and revocation use jobs consistent with their business meaning. Show progress with a fixed known denominator and successful, held, failed and unprocessed targets plus resumable checkpoints; unknown denominators cannot produce invented percentages. Samples do not replace full validation. Large exports split by file/customer boundaries with manifests. Recheck current permissions on resume and immediately before downloads. Never directly edit immutable originals or approved results.

## Responsive and keyboard behavior

Desktop uses a fixed top scope/period, left filters/tabs, a central table or editor and right detail. Keep scope and primary actions under the title and preserve identifying columns and currency/amount/status context during horizontal scrolling. Compare old and candidate results side by side with currency and basis fixed. Closing detail returns focus to the selected row. Concrete breakpoints, widths, fonts, color tokens, contrast and touch sizes remain unverified visual-design work.

Mobile uses scope/blocker summary, essential fields, primary action, then details in a single column. Open detail as a separate step and use collapsible filters. Cards summarize canonical ID, amount and state; a separate horizontally scrollable table retains all columns. Before/after comparison stacks vertically with clear headings. Fixed bottom actions cannot obscure errors or amounts. Do not remove required evidence, permissions, fields, actions, confirmations or results due to width. Bulk destructive actions require target-summary review before confirmation.

Read order is title, scope, tabs, filters, list, detail and actions. Use meaningful button/link names, labeled landmarks and heading hierarchy. Tab/Shift+Tab navigate basic elements, arrows navigate tabs, and Enter/Space activate buttons. Trees announce expansion and the current node. Dialogs trap focus and restore it to the caller; Escape closes without confirming execution. Errors link to their fields and cannot rely only on tooltips. Status uses text and icons, not color alone. Announce asynchronous results with a live region without repeatedly reading every row update. Charts have equivalent table summaries.

## Common acceptance plan

All scenarios are **NOT_RUN**: unauthorized rows/fields/counts remain hidden; filter changes invalidate selection; stale revisions and expired snapshots require recovery; revocation during work stops protected actions; details and dialogs preserve keyboard round-trip focus; mobile preserves confirmation and evidence; idempotent replay still checks current access. Static parsing, structure, reference and source comparisons do not establish running UI, server permissions, downloading, sending, large-volume processing, rendering, accessibility or integration success. Product implementation requires the owner's separate planning-complete declaration.


## Linked screens

- [S01: Edit connection](../S01/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [S06: First reconciliation](../S06/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [P05: Release evidence](../P05/spec.md). Pass only allowed canonical context and recheck permission at destination.

## Source acceptance scenarios

All scenarios below remain NOT_RUN.

- AUD-DB-02 (DB-02): Wrong roles, periods, contracts and source kinds produce restricted/unsupported guidance and alternate-data routes for each profile, not endless reauthentication/invitation. Never require root or blanket administrator credentials.
- AUD-DB-05 (DB-05): An ERP template that automatically includes an internal cost file blocks automatic-delivery activation independently of issuance/posting.
- AUD-DB-08 (DB-08): Stale diagnostics or changed configurations cannot activate. Reject expired or other-task handoff links. After setup revocation, retain only authorized operational reads.
- AUD-QA-07 (QA-07): Revoking setup access does not stop export writers. Unsupported scope routes to an unsupported path rather than reauthentication. Explain storage/query costs and expected intervals separately.

## Unresolved values and verification limits

- Revalidate permissions and costs for actual provider combinations.
- Per-stage diagnostic expiry and activation approver.

Field lengths, precision/scale policy values, permission enums, actual API addresses, nullable details, concrete error codes, page sizes, latency targets and supported product versions require detailed contract closure. The proposed command and numeric contracts define planning behavior, not chosen customer defaults. Current provider/ERP support, actual credentials and live data connections remain unverified. Static document/JSON checks cannot substitute for product behavior, accessibility, performance or live integration evidence.
