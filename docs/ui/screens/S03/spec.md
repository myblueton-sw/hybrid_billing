---
type: Screen Specification
title: S03 Import, conflicts and completeness
description: Check source files and events for duplication, corrections and missing partitions, and publish only eligible input revisions.
status: draft
sources:
  - resource: ../../../planning/v1.16/ui-design.md
  - resource: ../../../planning/v1.16/detailed-requirements.json
  - resource: ../../../planning/v1.16/provider-profiles.md
  - resource: ../../../contracts/commands/planning-actions.md
---

# S03 Import, conflicts and completeness

## Purpose, roles and provenance

Check source files and events for duplication, corrections and missing partitions, and publish only eligible input revisions.

Intended audience: admin. Roles: OP, BL.

HB-16 translates the complete prior local screen contract and integrates planning fixes. Original spec SHA-256: `caed22102c13593bf67e4973d1591d417c9c386fddd85a7b32a99bdf44d4d988`. Original design JSON SHA-256: `190827d02fb49c60fa622bf9e3cf53956ef4cb7f57f5048d83f07ce0d31fbf75`. Original source references above remain authoritative evidence; root retains the raw source snapshot. This document does not declare all 49 screens or the entire planning baseline complete.

Requirements: HB-05, HB-06, HB-07, HB-34, HB-39.

Acceptance references: AUD-AR-02, AUD-AR-03, AUD-AR-05, AUD-RT-01.



## Regions and reading order

Proposed layout: `workbench`.

### Input status

Separate original, normalized and OCR candidates.

### Completeness assessment

Compare received bytes, rows, partitions and watermark with expected coverage.

### Quarantine review

Exclude same-key different-content, forged and tenant-mismatched data from eligible manifests. Inputs containing prompt or response bodies are not accepted as normal billing artifacts; never preserve bodies in raw storage, parser staging, error logs or support bundles. Retain only nonsensitive rejection reasons and receipt metadata references and reacquire allowed source fields without bodies.

### Design status

Static proposal; product unimplemented; API disconnected; tests not run.
Use synthetic amounts and data only.
AUD links: AUD-AR-02, AUD-AR-03, AUD-RT-01, AUD-AR-05

## Field contract

| Label | Type | Required when | Editability | Validation | Sensitivity |
| --- | --- | --- | --- | --- | --- |
| Source connection and period | reference+interval | Always | Edit before receipt | Authenticated scope and period match | Account identifiers |
| File or batch reference | artifact-reference | Always | New receipt | Validate signature, size, compression and schema | Source evidence |
| Provenance, recipient and original number | text | When applicable | Edit before receipt | Capture original invoice provenance | Contract |
| Original hash and revision | digest+revision | Always | Read only | Quarantine same key with different content | Internal |
| Watermark and partition manifest | reference | Always | Read only | All partitions, periods and row counts match | Internal |
| OCR candidate and verification evidence | reference | When applicable | Verifier assessment | Exclude from confirmed input until checked against original | Source amounts |
| Correction reference and reason | reference+text | When applicable | Edit before receipt | Validate supersedes target and connection scope | Internal |
| Input eligibility and candidate revision | enum+revision | Before eligibility or dependent amount approval | Read only except explicit command draft | Bind exact immutable evidence and current authority; no waiver | Restricted business evidence |
| Confirmation actor and evidence digest | principal+digest | Before eligibility or dependent amount approval | Read only except explicit command draft | Bind exact immutable evidence and current authority; no waiver | Restricted business evidence |
| Source, mapping and profile revisions | reference-set | Before eligibility or dependent amount approval | Read only except explicit command draft | Bind exact immutable evidence and current authority; no waiver | Restricted business evidence |
| Independent cross-validation and comparison digest | reference+digest | Before eligibility or dependent amount approval | Read only except explicit command draft | Bind exact immutable evidence and current authority; no waiver | Restricted business evidence |

Sensitive fields: Source connection and period, File or batch reference, Provenance, recipient and original number, OCR candidate and verification evidence.

## Tabs, filters and columns

Tabs: Import jobs / Quarantine and conflicts / Completeness / Extraction verification.

Filters: Source connection, Period, Job state, Revision, Quarantine reason, Verification state.

Columns: Import ID → Source and period → Rows and bytes → Verified volume → Duplicates and quarantine → Completeness → Job state.

## Action contract

All identifiers below are proposed canonical planning IDs, not deployed APIs. Existing English permission tokens are preserved; descriptive permissions require the command catalog mapping, not an invented grant.

| Action | Proposed ID | Permission | Preconditions | Result | Confirmation |
| --- | --- | --- | --- | --- | --- |
| Receive data | Existing action; catalog mapping required | Source import | Allowed billing fields and connection verified | Create a job awaiting verification | Confirm target, bytes and duplication policy |
| View quarantine reason | Existing action; catalog mapping required | Import read | Quarantine record exists | Show only nonsensitive errors and metadata references; no excluded-body link | None |
| Record extraction verification | Existing action; catalog mapping required | Evidence verification | Original is available for comparison | Bind verification evidence to a new revision | Confirm verified amount and document number |
| Resume failed segment | Existing action; catalog mapping required | Import resume | Valid checkpoint and current permission | Resume while retaining completed segments | Confirm resumed segment |
| Inspect input evidence | input.inspect | Existing scoped evidence-read permission | Currently permitted exact candidate revision | Read input confirmation and validation evidence; no mutation | None |

## User-facing states

| State | Message |
| --- | --- |
| loading | Checking scope and permissions and loading Import, conflicts and completeness. |
| empty | No import records match. Start an import or change the period. |
| partial | Only part of the data or work is ready. Show ready, missing, held, failed and unprocessed scopes separately; this is not full completion or zero. |
| blocked | Input partitions are missing or extracted values are unverified. They cannot be published as confirmed input. |
| error | The request could not be completed. Check the request ID and failed scope, then choose a safe retry or handoff. |
| revoked | Access to this scope has changed. Remove sensitive details, previews and selections, stop affected actions and downloads, and query permitted scope again. Recall of externally delivered copies is not guaranteed. |
| normalized_candidate | This normalized revision is a candidate; it is not eligible for billing. |
| awaiting_confirmation | Required checks passed. The authorized worker must explicitly confirm this exact evidence in S06. |
| eligible | This exact revision has evidence-bound worker confirmation and atomic publication. |
| rejected | This candidate was rejected. Preserve the reason and create a new revision for corrections. |

## Confirmed input eligibility

The proposed [planning command contract](../../../contracts/commands/planning-actions.md) defines `input.inspect`, `input.confirm_publish` and `input.reject`. These are canonical planning IDs, not deployed endpoints or new permission grants. S03 collects candidates and evidence; S06 owns the explicit worker confirmation and atomic confirmation/publication action; B09 reads the resulting evidence as an amount-approval prerequisite. An amount approval never supplies or waives input confirmation.

The flow is `received → normalized_candidate → validating`, then proceeds to `blocked` for missing, conflicting or stale evidence, or `awaiting_confirmation` after all required checks pass. Only the currently authorized worker's explicit `input.confirm_publish` may atomically create the confirmation and publish that exact immutable revision as `eligible`. No intermediate confirmed-but-unpublished success or eligible-without-confirmation state is allowed. `input.reject` accepts only unpublished normalized_candidate, validating, blocked or awaiting_confirmation at the expected revision. It records reason and preserves candidate history; correcting a rejection creates a new candidate revision, not an edit to prior published evidence. A reject racing publication either wins before publication or conflicts after eligibility; it cannot turn eligible history into rejected. Withdrawal/correction of published eligibility requires the separate governed impact/financial path. `input.inspect` reads only currently permitted evidence and never confirms as a side effect.

Bind the actor, authenticated scope, candidate revision, immutable original/source hashes and manifest, parser/normalization/mapping versions, provider profile revision, comparison result/digest and completeness evidence. Mandatory independent cross-validation must be complete for the same scope, currency, units and period. Agreement between two calculations sharing the same erroneous extraction is not independent proof. Model confidence, LLM advice, successful upload, OCR completion and matching aggregate totals do not replace original comparison or worker confirmation. No waive control is offered.

Source, mapping, parser, profile, scope, evidence or comparison changes invalidate pending confirmation and dependent eligibility for new downstream use. Preserve prior finalized versions for audit; changed inputs require a new candidate and complete revalidation. Confirmation binds expected revision using compare-and-swap, with the current actor, permissions and applicable checks re-evaluated at commit. Concurrent evidence changes cannot race publication. A lost response retries the same idempotency key and payload after current authorization; payload mismatch conflicts, stale revision requires reload/review, missing checks remain blocked, and revoked access returns a generic denial without disclosing the prior result. Display request ID and safe next action, never a success inferred from timeout.

Acceptance scenarios (NOT_RUN): confirmation without independent evidence fails; evidence changes between inspection and confirmation fail; simultaneous confirm/reject cannot publish a rejected revision; lost-response replay returns only a currently authorized original result; amount approval cannot override blocked/awaiting input; old published versions remain auditable while changed candidates require new confirmation.


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

- [S01: Connection diagnostics](../S01/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [S05: Compare original](../S05/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [U03: Track jobs](../U03/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [S06: Worker input confirmation](../S06/spec.md). Pass only allowed canonical context and recheck permission at destination.

## Source acceptance scenarios

All scenarios below remain NOT_RUN.

- AUD-AR-02 (AR-02): Reject wrong certificates/tenants, forged, expired or replayed Push, changed DNS destinations, prohibited redirects and metadata addresses. Receive only approved internal connections. Explicit private destinations never authorize metadata or loopback access.
- AUD-AR-03 (AR-03): OCR decimal/currency errors block confirmation despite upload success. Publish a new eligible input revision only after original comparison and verification.
- AUD-RT-01 (RT-01): Reject missing mandatory semantics, tenant mismatch and invalid correction references. Returned source results cannot include unapproved documents or unauthorized costs. Grace expiry, unlocking and late sources preserve past finalized results.
- AUD-AR-05 (AR-05): A body-containing source fixture must not persist in raw storage, parser staging, billing storage, ordinary logs, support bundles, model input or customer exports. Keep only nonsensitive rejection references and reacquire allowed fields.

## Unresolved values and verification limits

- Allowed file and decompression limits.
- Signature clock skew and replay retention.
- OCR verifier assignment policy.

Field lengths, precision/scale policy values, permission enums, actual API addresses, nullable details, concrete error codes, page sizes, latency targets and supported product versions require detailed contract closure. The proposed command and numeric contracts define planning behavior, not chosen customer defaults. Current provider/ERP support, actual credentials and live data connections remain unverified. Static document/JSON checks cannot substitute for product behavior, accessibility, performance or live integration evidence.
