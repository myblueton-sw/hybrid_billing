---
type: Screen Specification
title: B11 Credits and billing inquiries
description: Manage balances and billing inquiries as separate states and link evidence-backed adjustments and approvals.
status: draft
sources:
  - resource: ../../../planning/v1.16/ui-design.md
  - resource: ../../../planning/v1.16/detailed-requirements.json
  - resource: ../../../planning/v1.16/provider-profiles.md
  - resource: ../../../contracts/commands/planning-actions.md
---

# B11 Credits and billing inquiries

## Purpose, roles and provenance

Manage balances and billing inquiries as separate states and link evidence-backed adjustments and approvals.

Intended audience: admin. Roles: BL, AP, CP.

HB-16 translates the complete prior local screen contract and integrates planning fixes. Original spec SHA-256: `618d38f51a8a04e7e484e81b1ec7ea4f673a54ae87276dbcbce02a132dca0390`. Original design JSON SHA-256: `f083b2c00cbd2666e0d94bd11f8bb04ce3b6dc3e6f89f503e791b165ee4231fb`. Original source references above remain authoritative evidence; root retains the raw source snapshot. This document does not declare all 49 screens or the entire planning baseline complete.

Requirements: HB-22, HB-23, HB-29, HB-31, HB-15, HB-30.

Acceptance references: AUD-AR-08, AUD-AR-10, AUD-DB-10, AUD-QA-09, PT-01, PT-02, PT-03, PT-04, PT-05, PT-06, PT-07, PT-08.



## Regions and reading order

Proposed layout: `workbench`.

### Payment terms, due dates and overdue

Receiving a dispute is not a payment exemption.
Separate confirmed credit application from the remaining balance after partial payment.

### Balance conservation

Opening plus increases minus application minus expiry plus/minus reversals equals closing; separate available and reserved.

### Inquiry review

Issue, owner, deadline and evidence; separate internal notes and public replies.

### Adjustment linkage

Credits, SLA and corrections require contract, approval and original references.

### Design status

Static proposal; product unimplemented; API disconnected; tests not run.
Use synthetic amounts and data only.
AUD links: AUD-AR-08, AUD-AR-10, AUD-DB-10, AUD-QA-09

## Field contract

| Label | Type | Required when | Editability | Validation | Sensitivity |
| --- | --- | --- | --- | --- | --- |
| Balance authority and as-of time | reference+datetime | Always | Read only | Stale external balances cause a hold | Financial |
| Opening, increases, application, expiry and reversals | money | Always | Read only | Verify conservation by currency | Financial |
| Migration ledger and included historical coverage | reference | When applicable | Migration draft | Do not count opening balance and reimported history twice | Restricted evidence |
| Case, disputed amount and deadline | reference+money+datetime | When applicable | Draft edit | An inquiry is not cancellation or refund | Personal and financial |
| Internal note and public reply | text | When applicable | Separate editing | No cost or margin in public replies | Internal and customer-public |
| Adjustment reason, contract and evidence | reference+text | When applicable | Draft edit | An outage alone cannot create an SLA discount | Restricted evidence |
| Confirmed allocation and reversal balance impact | read-only | For applicable external billing contracts | Read only | Check authority, currency, snapshot and receipt freshness; separate unknown and not applicable | Contract and customer financial |
| Dispute, collections hold and overdue state | read-only | For applicable external billing contracts | Read only | Check authority, currency, snapshot and receipt freshness; separate unknown and not applicable | Contract and customer financial |
| Hold type and target manifest | enum+reference | Hold request, release or status | Read only except explicit command draft | Separate billing/delivery/collections effects; unknown is not effective/released | Restricted business evidence |
| Hold and per-target revisions | reference+revision-map | Hold request, release or status | Read only except explicit command draft | Separate billing/delivery/collections effects; unknown is not effective/released | Restricted business evidence |
| Hold effect and external confirmation | enum+reference | Hold request, release or status | Read only except explicit command draft | Separate billing/delivery/collections effects; unknown is not effective/released | Restricted business evidence |

Sensitive fields: Balance authority and as-of time, Opening, increases, application, expiry and reversals, Migration ledger and included historical coverage, Case, disputed amount and deadline, Internal note and public reply, Adjustment reason, contract and evidence.

## Tabs, filters and columns

Tabs: Due dates and overdue / Balances / Reservations and application / Inquiries and disputes / Adjustments and reversals.

Filters: Payment due date, Overdue or unknown, Customer and contract, Period, Currency and unit, Balance authority, Case state, Owner, Deadline.

Columns: Confirmed allocation and reversal balance impact → Dispute, collections hold and overdue state → Balance and case ID → Customer and contract → Currency and unit → As-of time → Available and reserved → Disputed amount → Status → Owner and deadline.

## Action contract

All identifiers below are proposed canonical planning IDs, not deployed APIs. Existing English permission tokens are preserved; descriptive permissions require the command catalog mapping, not an invented grant.

| Action | Proposed ID | Permission | Preconditions | Result | Confirmation |
| --- | --- | --- | --- | --- | --- |
| Register inquiry | Existing action; catalog mapping required | Billing inquiry write | Contract, document and issue specified | Create a case with owner and deadline | None |
| Request adjustment approval | Existing action; catalog mapping required | Adjustment request | Reason, original, tax and balance verified | Send a fixed adjustment proposal to the approval inbox | Confirm amounts, balances and disclosure scope |
| Preview public reply | Existing action; catalog mapping required | Inquiry reply write | Public fields validated | Preview public reply; sending is separate | Confirm recipient and body |
| Record resolution | Existing action; catalog mapping required | Inquiry handling | Resolution evidence and related corrections linked | Resolve case while preserving separate financial state | Confirm unfinished financial actions |
| Inspect payment evidence | Existing action; catalog mapping required | Read authority for document and receipt evidence | Current permitted evidence scope | Read allowed evidence without changing receipts or credits | Distinguish acceptance from completion |
| Request hold | holds.request | Current scoped authority for the hold type and target action | Explicit type, target revisions, reason/evidence and required policy | Report per-target requested/effective/released/pending/unknown effect without auto-resume | Confirm type-specific effect; due dates and balances do not change |
| Request hold release | holds.release | Current scoped authority for the hold type and target action | Explicit type, target revisions, reason/evidence and required policy | Report per-target requested/effective/released/pending/unknown effect without auto-resume | Confirm type-specific effect; due dates and balances do not change |
| Inspect hold status | holds.status | Scoped hold read for the authorized hold and target | Current authorized hold/case ID and target scope; no new mutation rationale or approval evidence | Read current revisions, as-of, hold effects and external uncertainty without mutation | None; read-only status inspection |

## User-facing states

| State | Message |
| --- | --- |
| loading | Checking scope and permissions and loading Credits and billing inquiries. |
| empty | No balance or inquiry exists. Distinguish unknown balances from zero. |
| partial | Only part of the data or work is ready. Show ready, missing, held, failed and unprocessed scopes separately; this is not full completion or zero. |
| blocked | The balance is stale or concurrent reservations leave insufficient availability. Hold the adjustment. |
| error | The request could not be completed. Check the request ID and failed scope, then choose a safe retry or handoff. |
| revoked | Access to this scope has changed. Remove sensitive details, previews and selections, stop affected actions and downloads, and query permitted scope again. Recall of externally delivered copies is not guaranteed. |
| payment-unknown | Payment status could not be verified. Check the last confirmation time and due date. |
| hold_pending | The hold request is pending approval or authoritative external confirmation. |
| hold_effective | Evidence confirms this hold is effective for the displayed targets and type. |
| hold_release_pending | Release is requested; existing protection remains until confirmed. |
| hold_released | This hold is released; other holds remain and work does not automatically resume. |
| hold_external_unknown | External effect is unknown. Preserve protection and reconcile authoritative evidence. |

## Explicit billing, delivery and collections holds

The proposed [command contract](../../../contracts/commands/planning-actions.md) defines `holds.request`, `holds.release` and `holds.status`. A hold request fixes hold type (`billing`, `delivery` or `collections`), canonical target manifest, each target's expected revision, reason/evidence, actor/current authority, relevant approval policy and idempotency key. Preview affected and unaffected work before submission. Different hold types have separate effect boundaries: billing prevents the specified billing step, delivery stops controlled sending/disclosure paths, and collections pauses the specified reminder/collection activity. Inquiry state, a dispute or a successful read does not itself create a hold.

Show requested, pending approval/external confirmation, effective, rejected, release-requested, released and external-unknown effects separately from Job states. Per-target success, failure, pending, already-effective and conflict results must remain visible; a bulk receipt cannot imply all targets are held. Current permissions and expected revisions apply to request, release, status reads and retries. Hold permissions map to distinct hold.billing.request/release, hold.delivery.request/release and hold.collections.request/release plus scoped read; case resolution grants none. Local active/released state and external unsupported/requested/pending/confirmed-active/release-pending/confirmed-released/failed/unknown are separate. Local dispatch and hold/release serialize; delayed acknowledgements must match the current operation/revision. A stale target requires refreshed review; payload mismatch conflicts; denied or missing targets return generic outcomes without existence disclosure.

Release names the exact hold and affected targets and requires current authority, reasons, remaining blockers and any required approval. It does not automatically resume calculation, issue documents, resend packages, collect payment or advance closing. Subsequent work needs its own current-state command. External delivery/ERP/collection results may remain pending or unknown; retain conservative protection until authoritative evidence confirms the effect. Releasing one hold never removes another active hold. No hold or release changes original/effective payment due dates, contractual liability, confirmed receipts, credit balances or overdue calculation. Legal retention holds are separate and cannot be released by these operations. Payment extensions require the separate governed payment-term path.

Acceptance (NOT_RUN): mixed-authority bulk targets do not leak or all-pass; type-specific hold cannot stop/change an unrelated stage; concurrent revision change conflicts; unknown external effects cannot be reported effective/released; duplicate requests do not duplicate holds; release leaves other holds effective and does not auto-resume; dispute/hold never erases overdue or changes due dates.


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

- [B09: Adjustment approval](../B09/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [B10: Application history](../B10/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [U04: Anomalies](../U04/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [C03: Customer inquiries](../C03/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [P03: Opening balance migration](../P03/spec.md). Pass only allowed canonical context and recheck permission at destination.

## Source acceptance scenarios

All scenarios below remain NOT_RUN.

- AUD-AR-08 (AR-08): Suspected new charges/benefits can be raised despite matching totals. Distinguish normal confirmation from adjustment and trace actual corrections and resolution.
- AUD-AR-10 (AR-10): Block separate power rebilling for power-inclusive fixed fees and SLA discounts based only on outage events. Approved credits link to original documents and balances.
- AUD-DB-10 (DB-10): Unknown eligibility is not zero. Block automatic seller-fee pass-through, customer-document changes caused by settlement delay, and receipt completion without incoming funds.
- AUD-QA-09 (QA-09): Opening-balance migration and included historical transaction reimport cannot double increase balances. Currency conservation holds after concurrent use, expiry and reversals.

## Payment terms, due dates and overdue

Follow the [payment-term source](../../../planning/v1.16/payment-terms.md) and [English payment and recognition contract](../../../contracts/billing/payment-and-recognition.md). Contract terms such as Net 30/45 are examples, not universal defaults. Due dates start from issuance, with governed calendar details remaining explicit policy choices. PT-01 through PT-08 remain NOT_RUN.

The Due dates and overdue tab separates on-time, due-today, overdue, fully paid, unknown and not applicable. Only remaining balances after partial payment can be overdue. Disputes and collections holds are separate overlays; an inquiry is not a payment exemption. Only confirmed credit application changes the relevant balance. Unknown receipts stay out of confirmed overdue totals. Preserve authority, as-of time, receipt freshness and original/current due dates; filters and exports use the same snapshot, currency and permission scope. Mobile prioritizes due date, remaining balance, overdue/unknown status and confirmation time, with text rather than color alone.


## Unresolved values and verification limits

- Balance freshness threshold and reservation expiry.
- Inquiry response deadline and public reply approval.
- Credit validity and SLA formula.

Field lengths, precision/scale policy values, permission enums, actual API addresses, nullable details, concrete error codes, page sizes, latency targets and supported product versions require detailed contract closure. The proposed command and numeric contracts define planning behavior, not chosen customer defaults. Current provider/ERP support, actual credentials and live data connections remain unverified. Static document/JSON checks cannot substitute for product behavior, accessibility, performance or live integration evidence.
