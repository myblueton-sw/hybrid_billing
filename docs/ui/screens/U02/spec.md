---
type: Screen Specification
title: U02 Closing list and calendar
description: Manage contract deadlines and completion conditions, resolve blockers, and preserve exemption, postponement and reopening history.
status: draft
sources:
  - resource: ../../../planning/v1.16/ui-design.md
  - resource: ../../../planning/v1.16/detailed-requirements.md
  - resource: ../../../planning/v1.16/retention-access.md
  - resource: ../../../contracts/commands/planning-actions.md
---

# U02 Closing list and calendar

## Purpose, roles and provenance

Manage contract deadlines and completion conditions, resolve blockers, and preserve exemption, postponement and reopening history.

Intended audience: admin. Roles: BL settlement operator, AP approver, CP customer policy administrator.

HB-16 translates the complete prior local screen contract and integrates planning fixes. Original spec SHA-256: `18d33f7ddd3e4ff8f6a88611d25190412272153c94d90ec32f72c62da5a8e742`. Original design JSON SHA-256: `93131812be27490944f159a21e047390f85ef9397a40377a6ebe417bddbd8af7`. Original source references above remain authoritative evidence; root retains the raw source snapshot. This document does not declare all 49 screens or the entire planning baseline complete.

Requirements: HB-15, HB-23, HB-30, HB-31, HB-34, HB-24.

Acceptance references: AUD-QA-03, ERPC-01, ERPC-02, ERPC-03, ERPC-04, ERPC-05, ERPC-06, ERPC-07, ERPC-08, UI-02, UI-06.

## Entry and completion

Enter from a U01 closing card or U03 deadline event. Scope is permitted per-contract ClosingCycle; internal Chargeback and external-issuance contracts have different completion conditions. Show each cycle as completed, incomplete or exempt with evidence and trace required changes, validation and approval.

Calendar cell counts include permitted targets only and open the same ClosingCycle detail as the list. Mobile uses date-ordered agenda; change forms open separately and preserve original/new-date confirmation. Provide a complete list alternative to the calendar, explicit date-arrow navigation instructions and readable table sorting/selection. Compare original/current deadlines in aligned desktop columns. For over 100,000 closing targets, use period/entity/owner filters and cursors. Bulk assignment/validation is a job with separate success/hold/failure/unprocessed counts; never combine approval units to bypass authority. Flow: closing blocker, evidence screen, rerun validation, then approval/issuance/delivery checks. Correct issued documents through I01's original-referencing flow.

## Regions and reading order

Proposed layout: `table-detail`.

### Monthly ERP recognition and quarterly billing reconciliation

Monthly transmission and actual posting confirmation are independent of billing.
Current monthly net plus approved adjustments not yet included equals quarterly comparison; never count an adjustment twice.

### Period and mode

Switch list/calendar while preserving filters.

### Closing targets

Use text to distinguish deadlines, exceeded deadlines and blockers from completion, exemption and holds.

### Selected target conditions

Show mandatory stages, evidence, owner and next action.

### Deadline and state history

Preserve original deadlines and all postponements, exemptions, reopenings and target changes.

## Field contract

| Label | Type | Required when | Editability | Validation | Sensitivity |
| --- | --- | --- | --- | --- | --- |
| Contract and cycle | reference | Always | Read only | Bind to fixed target manifest | Transactions |
| Original deadline | datetime | Always | Immutable | Retain original deadline and delay after postponement | Business |
| Current deadline | datetime | Always | Permitted change draft | Calendar revision, timezone, change reason and impact review | Business |
| Calendar rule | reference | Always | Approved policy selection | Block without fixed-day/business-day, holiday, month-end, leap-year and DST rules | Policy |
| Completion conditions | checklist | Always | Read or rerun verification | Source, allocation, calculation, approval, issuance and delivery; internal routes have their own conditions | Business |
| Exemption reason and evidence | text-reference | When applicable | Exemption authority | Separate from completed; require evidence, accountability and allowed scope | Business |
| Owner | scoped-user | Always | Assignment authority | Hand off pending approvals and notification responsibility; reject inactive users | Personal |
| Denominator changes | revision-history | Always | Read only | Record target additions, exemptions and holds with revisions | Business |
| Change reason | textarea | For postponement, reopening or exemption | Required editing | Reject empty reason; never overwrite history | Business |
| Monthly recognition and quarterly billing deadlines | read-only | For recognition-linked contracts | Read only | Permitted entity, contract and ERP scope; verify snapshot, currency and posting state | Internal accounting and contract |
| Monthly posting and unreconciled status | read-only | For recognition-linked contracts | Read only | Permitted entity, contract and ERP scope; verify snapshot, currency and posting state | Internal accounting and contract |
| Operational stage and schedule ID | enum+reference | Schedule change request | Read only except explicit command draft | Bind cycle and exact stage; exclude payment extension from generic mutation | Restricted business evidence |
| Expected schedule and calendar revision | revision-pair | Schedule change request | Read only except explicit command draft | Bind cycle and exact stage; exclude payment extension from generic mutation | Restricted business evidence |

Sensitive fields: Customer closing, Blocked amount, Owner, Exemption evidence.

## Tabs, filters and columns

Tabs: Recognition cadence and reconciliation / List / Calendar / Selected closing detail / Deadline change history.

Filters: Customer or organization, Contract, Legal entity, Settlement cycle, Owner, Stage, Overdue deadline, Issuance path.

Columns: Monthly recognition and quarterly billing deadlines → Monthly posting and unreconciled status → Customer and contract → Cycle → Original and current deadline → Local timezone → Remaining or exceeded time → Stage → Blockers → Exempt or held → Owner.

## Action contract

All identifiers below are proposed canonical planning IDs, not deployed APIs. Existing English permission tokens are preserved; descriptive permissions require the command catalog mapping, not an invented grant.

| Action | Proposed ID | Permission | Preconditions | Result | Confirmation |
| --- | --- | --- | --- | --- | --- |
| Recheck conditions | Existing action; catalog mapping required | closing.validate | Fixed targets and current permission | Create a completion-condition verification job | Not automatic amount calculation or issuance; confirm target and snapshot |
| Hand off owner | Existing action; catalog mapping required | closing.assign and target-owner eligibility | Current assignment scope | Transfer unfinished work, pending approvals and notification responsibility | Confirm recipient and unfinished count; retain prior history |
| Request deadline change | schedule.change_request | closing.schedule.write and approval policy | Selected cycle, operational stage and schedule revision | Request a new stage deadline revision after impact review | Confirm old/new local and UTC times, affected cycles and preserved delay |
| Request exemption | Existing action; catalog mapping required | closing.exempt scope and policy | Evidence and current exemption scope | Request approval for a distinct evidenced exemption state | Exemption is not completion or amount deletion; confirm reason and evidence |
| Request reopening | Existing action; catalog mapping required | closing.reopen and current state | Current target state permits governed path | Before issuance create new calculation/approval; after issuance link an original-referencing correction | Never overwrite finalized content; confirm target, impact and path |
| Open approval inbox | Existing action; catalog mapping required | approval.read/request | Related fixed result available | Open related fixed result in B09 | A deadline cannot automatically approve or issue |

## User-facing states

| State | Message |
| --- | --- |
| loading | Checking scope and permissions and loading Closing list and calendar. |
| empty | No closing targets match the filters. |
| partial | Only part of the data or work is ready. Show ready, missing, held, failed and unprocessed scopes separately; this is not full completion or zero. |
| blocked | Unknown ERP outcomes, differences or mandatory file failures prevent closing. |
| error | Closing conditions could not be verified. Preserve the existing closing state. |
| revoked | Access to this scope has changed. Remove sensitive details, previews and selections, stop affected actions and downloads, and query permitted scope again. Recall of externally delivered copies is not guaranteed. |
| exempt | This target is exempt based on evidence. Count it separately from completed targets. |
| recognition-unreconciled | Monthly posting or billing reconciliation is incomplete. Check missing evidence, differences and adjustments. |
| schedule_change_pending | This stage-specific change awaits required approval. Other deadlines remain unchanged. |

## Stage-specific schedule changes

The proposed [command contract](../../../contracts/commands/planning-actions.md) defines `schedule.change_request`. Target identity is the ClosingCycle plus explicit operational stage and schedule ID, never a bare cycle deadline. Bind authenticated scope, expected schedule revision, original/current/proposed local and UTC timestamps, timezone, calendar-policy revision, reason, affected targets and approval policy. The confirmation names the selected stage and unaffected stages.

Collection, calculation, approval, issuance, delivery and payment are distinct stages. A collection deadline change cannot silently move approval or delivery. Payment due-date extension is excluded from generic operational schedule mutation and must route to the separately governed `payment.extension_request` and approved payment-term extension process with its own document/receivable authority, approval and immutable original due date. Monthly-recognition closing and quarterly-billing closing also retain separate cadence and target identity. Select no calendar, grace or extension default here.

The request creates a proposed revision pending the required approval; it does not execute billing or complete closing. The separate proposed `schedule.apply_approved` applies the exact approved digest with separately authorized approval/execution permission and compare-and-swap against the expected stage/schedule revision. A request receipt is never application. Stale revision, changed calendar evidence or authority requires reload and new review. Same-key/same-payload retry rechecks current authority and returns the original permitted result; different payload conflicts. Preserve initial deadline and all delays, reasons, approvals, exemptions and reopenings. Missing required calendar policy blocks rather than substituting a date.

Acceptance (NOT_RUN): same cycle/different stages remain distinct; old-stage revision cannot mutate a new schedule; payment extension cannot use this generic action; retries do not create multiple revisions; DST/holiday policy changes force review; approval or deadline arrival cannot issue or complete the cycle.


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

- [U01: Home](../U01/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [U03: Jobs and deadline notifications](../U03/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [S06: Reconciliation](../S06/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [B08: Calculation](../B08/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [B09: Approval](../B09/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [I01: Documents](../I01/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [R02: History](../R02/spec.md). Pass only allowed canonical context and recheck permission at destination.

## Source acceptance scenarios

All scenarios below remain NOT_RUN.

- UI-02, UI-06, AUD-QA-03: Exempt is not completed; postponement never erases original deadlines or delay history. Preserve explicit calendar rules across invalid dates, leap years, DST and group moves. Deadlines and read notifications cannot automatically issue or complete closing.

## Monthly ERP recognition and quarterly billing

Follow the [recognition source](../../../planning/v1.16/erp-recognition-cadence.md) and [English payment and recognition contract](../../../contracts/billing/payment-and-recognition.md). Customer billing cadence and ERP recognition cadence are separate. ERPC-01 through ERPC-08 remain NOT_RUN.

The Recognition cadence and reconciliation tab shows monthly batches, actual external posting, quarterly linkage, adjustments and unreconciled states. Missing or unknown months are not zero or completed. Corrections/retransmissions retain separate ERP execution/approval authority; inquiry/read access does not authorize execution. Show unsupported journal types as unsupported/manual handoff, not substitute customer invoices. Internal monthly recognition journals are not customer-public by default.

Before quarterly billing, verify support for offset/replacement/reversal of already-recognized amounts and the resulting external net. Current monthly net plus only approved adjustments not already included forms the quarterly comparison; distinct keys alone cannot prevent duplicate economic recognition. Unresolved differences and unknown external outcomes block final completion. Even an explicit issuance-gate exception does not erase unreconciled status. Monthly recognition transmission does not start payment terms. On small screens summarize monthly recognition, quarterly billing, difference and verification time with access to details. A zero arithmetic difference is not proof of complete external posting evidence.


## Unresolved values and verification limits

- Holiday calendar, business-day and DST values.
- Deadline change and exemption approval roles.
- Escalation stages and timing.

Field lengths, precision/scale policy values, permission enums, actual API addresses, nullable details, concrete error codes, page sizes, latency targets and supported product versions require detailed contract closure. The proposed command and numeric contracts define planning behavior, not chosen customer defaults. Current provider/ERP support, actual credentials and live data connections remain unverified. Static document/JSON checks cannot substitute for product behavior, accessibility, performance or live integration evidence.
