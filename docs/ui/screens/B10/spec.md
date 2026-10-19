---
type: Screen Specification
title: B10 Billing application history
description: Trace applied coverage, remainder and reversals from source through charge lines and submissions to detect duplicate billing.
status: draft
sources:
  - resource: ../../../planning/v1.16/ui-design.md
  - resource: ../../../planning/v1.16/detailed-requirements.json
  - resource: ../../../planning/v1.16/provider-profiles.md
  - resource: ../../../contracts/commands/planning-actions.md
---

# B10 Billing application history

## Purpose, roles and provenance

Trace applied coverage, remainder and reversals from source through charge lines and submissions to detect duplicate billing.

Intended audience: admin. Roles: BL, AU.

HB-16 translates the complete prior local screen contract and integrates planning fixes. Original spec SHA-256: `d276af243f765dfffa40e4783a87b5da1ad54f1326f9ad6b6041c6c0f2d45460`. Original design JSON SHA-256: `648394cd9c3e73519729e248937ac00bf9e30e0c098ada8778b2f2562198be96`. Original source references above remain authoritative evidence; root retains the raw source snapshot. This document does not declare all 49 screens or the entire planning baseline complete.

Requirements: HB-21, HB-22, HB-29.

Acceptance references: AUD-DB-03.



## Regions and reading order

Proposed layout: `table-detail`.

### Ledger summary

Show applied, remaining and reversed amounts by currency.

### Trace path

Source to allocation to sales to line to API/file submission.

### Duplicate assessment

Include coverage already applied by ERP orders or prepayments; no direct ledger editing.

### Design status

Static proposal; product unimplemented; API disconnected; tests not run.
Use synthetic amounts and data only.
AUD links: AUD-DB-03

## Field contract

| Label | Type | Required when | Editability | Validation | Sensitivity |
| --- | --- | --- | --- | --- | --- |
| Contract, component and period | reference+interval | Always | Read only | Fix business duplication scope | Contract |
| Source to allocation to sales to charge line | reference-chain | Always | Read only | Preserve canonical references | Restricted evidence |
| Applied, remaining and reversed amounts | money | Always | Read only | Preserve currency and partial coverage | Sales |
| Existing ERP order and prepayment reference | reference | When applicable | Read only | A new submission ID cannot bypass existing billing | ERP contract |
| Correction and original reference | reference | When applicable | Read only | Never overwrite the original | Internal |
| Coverage identity and units | reference+unit | Claim detail | Read only except explicit command draft | Protected unknown outcome; quantity reversal does not rebill; money-only credit does not reopen quantity | Restricted business evidence |
| Coverage state and external evidence | enum+reference | Claim detail | Read only except explicit command draft | Protected unknown outcome; quantity reversal does not rebill; money-only credit does not reopen quantity | Restricted business evidence |
| Reserved, applied, reversed and available coverage | decimal-coverage | Claim detail | Read only except explicit command draft | Protected unknown outcome; quantity reversal does not rebill; money-only credit does not reopen quantity | Restricted business evidence |
| Linked correction and original Claim | reference-pair | Claim detail | Read only except explicit command draft | Protected unknown outcome; quantity reversal does not rebill; money-only credit does not reopen quantity | Restricted business evidence |

Sensitive fields: Contract, component and period, Source to allocation to sales to charge line, Applied, remaining and reversed amounts, Existing ERP order and prepayment reference.

## Tabs, filters and columns

Tabs: Application ledger / Partial billing / Reversals and corrections / External references.

Filters: Contract, Period, Component, Claim state, Submission path, Original document.

Columns: Claim ID → Contract and period → Component → Source coverage → Applied amount → Remainder → Submission and document → Correction reference.

## Action contract

All identifiers below are proposed canonical planning IDs, not deployed APIs. Existing English permission tokens are preserved; descriptive permissions require the command catalog mapping, not an invented grant.

| Action | Proposed ID | Permission | Preconditions | Result | Confirmation |
| --- | --- | --- | --- | --- | --- |
| Trace application path | Existing action; catalog mapping required | Billing history read | Permitted contract and period | Link original source through external document | None |
| View duplicate conflict | Existing action; catalog mapping required | Billing history read | Conflict record exists | Show existing Claim and external billing evidence | None |
| Open correction review | Existing action; catalog mapping required | Correction request | Issued document and reason available | Open a B11 case draft | Confirm original and affected scope |

## User-facing states

| State | Message |
| --- | --- |
| loading | Checking scope and permissions and loading Billing application history. |
| empty | No billing application records exist for this scope. |
| partial | Only part of the data or work is ready. Show ready, missing, held, failed and unprocessed scopes separately; this is not full completion or zero. |
| blocked | The same coverage is already billed or the external outcome is unknown. Reconcile before proceeding. |
| error | The request could not be completed. Check the request ID and failed scope, then choose a safe retry or handoff. |
| revoked | Access to this scope has changed. Remove sensitive details, previews and selections, stop affected actions and downloads, and query permitted scope again. Recall of externally delivered copies is not guaranteed. |
| coverage_protected_unknown | The external outcome is unknown. This coverage remains protected and cannot be rebilled. |

## Claim coverage and correction navigation

Read the proposed [Claim and numeric contract](../../../contracts/billing/claim-and-numeric.md). Display canonical coverage identity, source intervals or quantities/units, business deduplication scope, Claim revision, reserved/applied/reversed coverage, remaining available coverage and external references separately from amounts. Coverage is not inferred from a display label, page total, new submission ID or monetary credit.

Expose available, reserved, externally pending/unknown, applied, release-pending/released and reversed/corrected coverage with their authoritative evidence; these are business categories bound to the canonical contract, not a replacement Job enum. Pending/unknown external outcomes retain protected coverage. No timeout, local job failure or arbitrary expiry makes that coverage billable again. Reservation release requires proven nonapplication or authoritative cancellation, current rights and version checks. Link unresolved outcomes to I02 reconciliation and original-referencing corrections to B11/I01.

Quantity reversal does not automatically rebill. Any rebilling requires explicit claim.authorize_replacement for reconciled replacement-pending coverage followed by ordinary approval/reservation; reversal disposition may instead be cancelled or superseded and unavailable. Reversal subsets are checked against all prior effective reversals: a partly overlapping second reversal conflicts even if scalar cumulative quantity is below the original application. Money-only credits do not reopen quantity. Preserve original Claim, linked correction and tax/currency/amount/quantity dimensions; never directly edit the ledger. Display unknown remainder as unknown, not zero or available. Numeric precision, scale, rounding and deterministic remainder policy remain proposed under the numeric contract, with customer values unset.

Acceptance (NOT_RUN): concurrent partial claims cannot overlap; a fresh external key cannot rebill protected coverage; unknown submission retains reservation; credit-only adjustment does not release quantity; quantity reversal cannot auto-submit; correction links retain originals; currency/unit differences cannot be combined.


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

- [B08: Calculation results](../B08/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [B11: Corrections and credits](../B11/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [I02: Submission state](../I02/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [S05: Original evidence](../S05/spec.md). Pass only allowed canonical context and recheck permission at destination.

## Source acceptance scenarios

All scenarios below remain NOT_RUN.

- AUD-DB-03 (DB-03): A fresh platform submission ID cannot reissue coverage already reflected in ERP orders/prepayments. Duplicate purchase documents and unknown external-ledger outcomes hold submission.

## Unresolved values and verification limits

- Partial coverage identity contract; proposed geometry in claim-and-numeric.md awaits closure.
- Reconciliation method for existing external orders.

Field lengths, precision/scale policy values, permission enums, actual API addresses, nullable details, concrete error codes, page sizes, latency targets and supported product versions require detailed contract closure. The proposed command and numeric contracts define planning behavior, not chosen customer defaults. Current provider/ERP support, actual credentials and live data connections remain unverified. Static document/JSON checks cannot substitute for product behavior, accessibility, performance or live integration evidence.
