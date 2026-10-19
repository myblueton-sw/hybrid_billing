---
type: Screen Specification
title: U01 Settlement overview
description: Understand settlement progress and blockers within the assigned scope and open the next closing, approval or document action.
status: draft
sources:
  - resource: ../../../planning/v1.16/ui-design.md
  - resource: ../../../planning/v1.16/detailed-requirements.md
  - resource: ../../../planning/v1.16/retention-access.md
  - resource: ../../../contracts/commands/planning-actions.md
---

# U01 Settlement overview

## Purpose, roles and provenance

Understand settlement progress and blockers within the assigned scope and open the next closing, approval or document action.

Intended audience: admin. Roles: BL settlement operator, AP approver, CP customer policy administrator, AU auditor.

HB-16 translates the complete prior local screen contract and integrates planning fixes. Original spec SHA-256: `4a9692be701a7032fc1e0b6e77c1b72011b318c579519653469b4f009c2c2123`. Original design JSON SHA-256: `c2693641be9e14ba442cdb2ff5843b1ef002f6d08aff88486010e3645a9ecbc6`. Original source references above remain authoritative evidence; root retains the raw source snapshot. This document does not declare all 49 screens or the entire planning baseline complete.

Requirements: HB-01, HB-23, HB-30, HB-31, HB-39, HB-15.

Acceptance references: AUD-QA-03, PT-01, PT-02, PT-03, PT-04, PT-05, PT-06, PT-07, PT-08, SCALE-03, UI-01, UI-02.

## Entry and completion

Enter from the business user's default home or another workflow's Home link. Scope covers currently permitted organizations, customers, contracts, legal entities and periods. MSP, standalone-enterprise and group-company profiles alter the default view, never authority. Completion means understanding the different cost/sales/allocation meanings and closing denominator, then opening the highest-priority blocker or assigned work.

All tabs retain the same scope token. My work adds an assignee condition without extending permissions; clicking a card translates that card's aggregate conditions into detail filters. The dashboard offers no bulk amount-changing actions. Desktop order is scope, amount-meaning cards, closing/my-work columns and blocker list. Mobile places deadlines, blockers and next actions before charts. Card title/amount relationships must be accessible, and refresh must not move focus. Flow: U01 to target U02 closing, resolve blocker in its screen, track U03 job, then return to a new U01 snapshot.

## Regions and reading order

Proposed layout: `dashboard`.

### Payment terms, due dates and overdue

Due dates and overdue status by contract.
Separate unknown amounts from confirmed overdue totals.

### Scope and freshness

Fix workspace, current scope, period, currency and refresh state.

### Summary by amount meaning

Provide cost, sales and allocation cards separately under current permissions; do not aggregate unauthorized amounts or counts.

### Closing and blockers

Show completed, exempt and held targets with fixed denominators, upcoming or exceeded deadlines and unknown ERP outcomes.

### My work

Show currently permitted next actions for approval requests, differences, document failures and anomalies.

## Field contract

| Label | Type | Required when | Editability | Validation | Sensitivity |
| --- | --- | --- | --- | --- | --- |
| Business scope | scope-select | Always | Permitted scope only | Canonical IDs and period; update cards and selections together on scope change | Customer identifiers |
| Settlement period | period | Always | Queryable periods | Distinguish contract-local timezone and execution as-of time | Transactions |
| Total cost | decimal-summary | When applicable | Read only with cost permission | Separate currency and cost type; never add sales or allocations | Cost |
| Total sales | decimal-summary | When applicable | Read only with sales permission | Separate expected and confirmed amounts; totals by currency | Sales |
| Organization allocation | decimal-summary | When applicable | Read only with allocation permission | Show internal approval state, period and currency | Allocation |
| Progress | count-summary | Always | Read only | Fixed target/completion denominator; separate exemption, hold and addition history | Business |
| Blockers and unknown ERP outcomes | status-summary | Always | Read only | No other-customer counts; never infer ERP waiting as success or failure | Business |
| Display as-of time | datetime | Always | Read only | Show card snapshot and refresh delay | System |
| Display exchange rate | reference | When applicable | Permitted aggregate mode | Show policy/time only for converted totals; does not modify finalized documents | Amounts |
| Receivables and overdue amounts by currency | read-only | For applicable external billing contracts | Read only | Check current scope, authority, currency, snapshot and receipt freshness; unknown differs from not applicable | Contract and customer financial |
| Due, overdue and unknown counts | read-only | For applicable external billing contracts | Read only | Check current scope, authority, currency, snapshot and receipt freshness; unknown differs from not applicable | Contract and customer financial |
| As-of time and receipt freshness | read-only | For applicable external billing contracts | Read only | Check current scope, authority, currency, snapshot and receipt freshness; unknown differs from not applicable | Contract and customer financial |
| Favorite target and revision | reference+revision | Favorite actions | Read only except explicit command draft | Principal/workspace scoped; no sensitive cached snapshot | Restricted business evidence |

Sensitive fields: Cost, Sales amount, Allocation amount, Customer-specific blockers, Owner.

## Tabs, filters and columns

Tabs: Due dates and overdue / Current settlement / My work / Recent changes.

Filters: Payment due date, Overdue or unknown, Assigned customer or organization, Contract, Legal entity, Settlement period, Currency, Operating profile.

Columns: Receivables and overdue amounts by currency → Due, overdue and unknown counts → As-of time and receipt freshness → Customer or organization → Contract → Nearest deadline → Settlement stage → Blocking reason → Owner → As-of time.

## Action contract

All identifiers below are proposed canonical planning IDs, not deployed APIs. Existing English permission tokens are preserved; descriptive permissions require the command catalog mapping, not an invented grant.

| Action | Proposed ID | Permission | Preconditions | Result | Confirmation |
| --- | --- | --- | --- | --- | --- |
| Inspect closing | Existing action; catalog mapping required | closing.read | Current scope allowed | Open same-scope closing list in U02 | No confirmation for reads; retain snapshot and scope |
| View my work | Existing action; catalog mapping required | Own or delegated task permission | Current task scope allowed | Open assigned jobs and cases in U03 | Reading a notification does not resolve work |
| Inspect blocker evidence | Existing action; catalog mapping required | Target and evidence access | Current evidence scope allowed | Open each blocker in reconciliation, approval or submission detail | Do not disclose unauthorized counts or reasons |
| Open report | Existing action; catalog mapping required | report.read | Current report scope allowed | Analyze current amount meaning and scope in R01 | Do not automatically sum different meanings |
| Refresh overview | Existing action; catalog mapping required | Read permission | Current scope allowed | Fetch new snapshot and replace prior as-of time | Does not reexecute business commands; explain invalidated selections |
| View overdue detail | Existing action; catalog mapping required | Receivables read in current legal entity and customer scope | Current snapshot and overdue filter | Open R01 with the same snapshot overdue filter; export requires separate authority | Distinguish acceptance from completion |
| Add favorite | favorites.add | Current personal workspace and target access | Canonical target and current authority; expected personal favorite revision where applicable | Mutate/read only own favorite references; generic denied/unavailable result | Removal affects favorite only; no authority grant; regrant restoration follows approved policy |
| Remove favorite | favorites.remove | Current personal workspace and target access | Canonical target and current authority; expected personal favorite revision where applicable | Mutate/read only own favorite references; generic denied/unavailable result | Removal affects favorite only; no authority grant; regrant restoration follows approved policy |
| List favorites | favorites.list | Current personal workspace and target access | Canonical target and current authority; expected personal favorite revision where applicable | Mutate/read only own favorite references; generic denied/unavailable result | Removal affects favorite only; no authority grant; regrant restoration follows approved policy |

## User-facing states

| State | Message |
| --- | --- |
| loading | Checking scope and permissions and loading Settlement overview. |
| empty | No settlement targets are registered in this scope. |
| partial | Only part of the data or work is ready. Show ready, missing, held, failed and unprocessed scopes separately; this is not full completion or zero. |
| blocked | Some targets lack mandatory evidence. Open blocker details to act. |
| error | The overview could not be refreshed. Check the last successful as-of time. |
| revoked | Access to this scope has changed. Remove sensitive details, previews and selections, stop affected actions and downloads, and query permitted scope again. Recall of externally delivered copies is not guaranteed. |
| zero | The verified amount in this scope is zero. No-usage evidence is available. |
| payment-unknown | Payment status could not be verified. Check the last confirmation time and due date. |
| favorite_unavailable | This favorite is unavailable. Refresh permitted favorites or explicitly add an accessible target. |

## Favorites

The proposed [command contract](../../../contracts/commands/planning-actions.md) defines `favorites.add`, `favorites.remove` and `favorites.list`. Favorites are personal references scoped by the authenticated principal and workspace, with canonical customer target ID and expected favorite revision where applicable. They never store a permission grant, sensitive target snapshot or customer amount. Personal-preference write and customer read are separately required. Current customer permissions are checked at add, list, navigation and idempotent replay. Rename retains canonical identity; termination never transfers favorites to a recreated identity.

Duplicate addition is idempotent for the same principal/workspace/target. Removal changes only that personal favorite. Missing and forbidden targets receive a generic unavailable/denied result without exposing existence, names, counts or last-known metadata. Revoked targets are removed or suppressed from visible results under a bounded lifecycle policy; display no stale sensitive label. Later permission regrant obeys the approved versioned restoration policy. Automatic restoration is forbidden unless explicitly selected and approved in that policy; otherwise a fresh authorized addition is required. Missing policy blocks activation.

Favorite count bounds, order, retention/cleanup duration and exact removal-versus-suppression lifecycle remain unresolved. A bounded policy is required before implementation acceptance; do not invent an unlimited store or numeric default. Acceptance (NOT_RUN): cross-user/workspace listing rejected; revoked target not leaked; add/remove replay remains authorized; regrant does not silently restore; duplicate add does not create duplicate rows; missing and unauthorized outcomes are indistinguishable.


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

- [U02: Closing](../U02/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [U03: Jobs](../U03/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [U04: Anomalies](../U04/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [S06: Reconciliation](../S06/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [B09: Approval](../B09/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [I02: Submission](../I02/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [R01: Reports](../R01/spec.md). Pass only allowed canonical context and recheck permission at destination.

## Source acceptance scenarios

All scenarios below remain NOT_RUN.

- UI-01, UI-02, SCALE-03, AUD-QA-03: Do not sum cost, sales, allocations or different currencies together. Exemption, holds and target additions preserve progress denominators and original history. Cards and drill-down never reveal unauthorized customer counts or amounts.

## Payment terms, due dates and overdue

Follow the [payment-term source](../../../planning/v1.16/payment-terms.md) and [English payment and recognition contract](../../../contracts/billing/payment-and-recognition.md). Contract terms such as Net 30/45 are examples, not universal defaults. Due dates start from issuance, with governed calendar details remaining explicit policy choices. PT-01 through PT-08 remain NOT_RUN.

The Due dates and overdue tab separates on-time, due-today, overdue, fully paid, unknown and not applicable. Only remaining balances after partial payment can be overdue. Disputes and collections holds are separate overlays; an inquiry is not a payment exemption. Only confirmed credit application changes the relevant balance. Unknown receipts stay out of confirmed overdue totals. Preserve authority, as-of time, receipt freshness and original/current due dates; filters and exports use the same snapshot, currency and permission scope. Mobile prioritizes due date, remaining balance, overdue/unknown status and confirmation time, with text rather than color alone.


## Unresolved values and verification limits

- Card order by home profile.
- Refresh interval and delay target.
- Display conversion policy.

Field lengths, precision/scale policy values, permission enums, actual API addresses, nullable details, concrete error codes, page sizes, latency targets and supported product versions require detailed contract closure. The proposed command and numeric contracts define planning behavior, not chosen customer defaults. Current provider/ERP support, actual credentials and live data connections remain unverified. Static document/JSON checks cannot substitute for product behavior, accessibility, performance or live integration evidence.
