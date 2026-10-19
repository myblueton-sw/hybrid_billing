---
type: Screen Specification
title: S06 Reconciliation and differences
description: Compare expected and actual amounts with the same currency, meaning and scope, and resolve differences with evidence.
status: draft
sources:
  - resource: ../../../planning/v1.16/ui-design.md
  - resource: ../../../planning/v1.16/detailed-requirements.json
  - resource: ../../../planning/v1.16/provider-profiles.md
  - resource: ../../../contracts/commands/planning-actions.md
---

# S06 Reconciliation and differences

## Purpose, roles and provenance

Compare expected and actual amounts with the same currency, meaning and scope, and resolve differences with evidence.

Intended audience: admin. Roles: BL, AP, CP.

HB-16 translates the complete prior local screen contract and integrates planning fixes. Original spec SHA-256: `afca06def277689139b4d15ec753a3079bf2b845966de2ed70fdde9223e0138c`. Original design JSON SHA-256: `d1bbed9693561fc97ea439f719d0a488da8aef72f652a790508b1254fecfe53a`. Original source references above remain authoritative evidence; root retains the raw source snapshot. This document does not declare all 49 screens or the entire planning baseline complete.

Requirements: HB-09, HB-12, HB-14, HB-19, HB-20, HB-23.

Acceptance references: AUD-AR-03, AUD-AR-08, AUD-DB-04.



## Regions and reading order

Proposed layout: `workbench`.

### Reconciliation summary

Expected, actual and difference by original currency; unexplained difference count.

### Difference table

Keep cost, sales and allocation reconciliation meanings separate.

### Evidence panel

Independent pricing requires a sales contract; cost-linked pricing requires confirmed source evidence.

### Design status

Static proposal; product unimplemented; API disconnected; tests not run.
Use synthetic amounts and data only.
AUD links: AUD-AR-03, AUD-AR-08, AUD-DB-04

## Field contract

| Label | Type | Required when | Editability | Validation | Sensitivity |
| --- | --- | --- | --- | --- | --- |
| Reconciliation basis and snapshot | reference | Always | Draft selection | Cost meaning, currency and period match | Cost |
| Expected, actual and difference | money | Always | Read only | Decimal reconciliation by currency | Cost and sales |
| Whole document and platform-attributable amount | money | When applicable | Read only | Evidence for excluding unrelated ERP lines | ERP amounts |
| Tax and credit classification | enum | When applicable | Evidence classification edit | Matches source and policy | Contract |
| Difference reason and evidence | text+reference | Always | Edit | Do not conceal source errors with arbitrary adjustments | Restricted evidence |
| Owner and resolution status | principal+enum | Always | Edit within authority | Resolution requires verification results | Internal |
| Input eligibility and candidate revision | enum+revision | Before eligibility or dependent amount approval | Read only except explicit command draft | Bind exact immutable evidence and current authority; no waiver | Restricted business evidence |
| Confirmation actor and evidence digest | principal+digest | Before eligibility or dependent amount approval | Read only except explicit command draft | Bind exact immutable evidence and current authority; no waiver | Restricted business evidence |
| Source, mapping and profile revisions | reference-set | Before eligibility or dependent amount approval | Read only except explicit command draft | Bind exact immutable evidence and current authority; no waiver | Restricted business evidence |
| Independent cross-validation and comparison digest | reference+digest | Before eligibility or dependent amount approval | Read only except explicit command draft | Bind exact immutable evidence and current authority; no waiver | Restricted business evidence |

Sensitive fields: Reconciliation basis and snapshot, Expected, actual and difference, Whole document and platform-attributable amount, Tax and credit classification, Difference reason and evidence.

## Tabs, filters and columns

Tabs: Source reconciliation / Sales reconciliation / Allocation reconciliation / ERP reconciliation.

Filters: Period, Currency, Reconciliation type, Difference reason, Owner, Status.

Columns: Reconciliation ID → Scope and basis → Expected amount → Actual amount → Difference → Explanation state → Owner → Blockers.

## Action contract

All identifiers below are proposed canonical planning IDs, not deployed APIs. Existing English permission tokens are preserved; descriptive permissions require the command catalog mapping, not an invented grant.

| Action | Proposed ID | Permission | Preconditions | Result | Confirmation |
| --- | --- | --- | --- | --- | --- |
| Reconcile again | Existing action; catalog mapping required | Reconciliation execution | Inputs fixed to the same scope | Create a new verification result | Confirm affected scope |
| Register difference evidence | Existing action; catalog mapping required | Difference review | Row and document locators available | Link explanation and evidence | None |
| Request adjustment review | Existing action; catalog mapping required | Adjustment request | Reason, evidence and affected amounts available | Open a separate adjustment draft | Confirm amount and tax impact |
| Verify resolution | Existing action; catalog mapping required | Reconciliation confirmation | Required differences explained | Record verification pass or fail | Confirm remaining blockers |
| Inspect input evidence | input.inspect | Existing scoped evidence-read permission | Currently permitted exact candidate revision | Read input confirmation and validation evidence; no mutation | None |
| Confirm and publish input | input.confirm_publish | Scoped input.confirm plus publication authority under approved role mapping | Awaiting confirmation; expected candidate and scope revisions, exact evidence digest and current authority | Atomically record confirmation decision and publish eligible input; failure returns explicit error/blocked state without implicit rejection | Confirm candidate, evidence, scope and publication effect |
| Reject input candidate | input.reject | Scoped input confirmation authority; separate from amount approval | Current actor, exact unpublished normalized_candidate/validating/blocked/awaiting_confirmation and expected revision, evidence digest and rejection reason; reject conflicts after publication | Record rejection reason and preserve candidate history; do not publish | Confirm source, mapping, profile, comparison digest and affected downstream scope |

## User-facing states

| State | Message |
| --- | --- |
| loading | Checking scope and permissions and loading Reconciliation and differences. |
| empty | No reconciliation targets exist in this scope. This differs from a matched zero amount. |
| partial | Only part of the data or work is ready. Show ready, missing, held, failed and unprocessed scopes separately; this is not full completion or zero. |
| blocked | Unexplained differences or missing mandatory evidence prevent an approval request. |
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

- [S05: Original evidence](../S05/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [B08: Calculation comparison](../B08/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [I02: ERP state](../I02/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [U04: Billing anomalies](../U04/spec.md). Pass only allowed canonical context and recheck permission at destination.

## Source acceptance scenarios

All scenarios below remain NOT_RUN.

- AUD-AR-03 (AR-03): OCR decimal/currency errors block confirmation despite upload success; original comparison and verification precede eligible revision publication.
- AUD-AR-08 (AR-08): Permit suspected new charges/benefits to be reported even when totals reconcile. Distinguish confirmed-normal from adjustment-needed and trace actual correction/resolution evidence.
- AUD-DB-04 (DB-04): Reconcile only platform-attributable totals in invoices mixing platform usage charges with independent ERP products and in split documents for one submission.

## Unresolved values and verification limits

- Reconciliation tolerance and exemption authority; mandatory independent input checks cannot be waived.
- Required evidence by cost type.

Field lengths, precision/scale policy values, permission enums, actual API addresses, nullable details, concrete error codes, page sizes, latency targets and supported product versions require detailed contract closure. The proposed command and numeric contracts define planning behavior, not chosen customer defaults. Current provider/ERP support, actual credentials and live data connections remain unverified. Static document/JSON checks cannot substitute for product behavior, accessibility, performance or live integration evidence.
