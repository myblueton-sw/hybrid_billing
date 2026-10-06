---
type: Screen Specification
title: B09 Approval inbox and result cards
description: Check current authority and fixed results for approval or rejection, keeping author and approver distinct.
status: draft
sources:
  - resource: ../../../planning/v1.16/ui-design.md
  - resource: ../../../planning/v1.16/detailed-requirements.json
  - resource: ../../../planning/v1.16/provider-profiles.md
  - resource: ../../../contracts/commands/planning-actions.md
---

# B09 Approval inbox and result cards

## Purpose, roles and provenance

Check current authority and fixed results for approval or rejection, keeping author and approver distinct.

Intended audience: admin. Roles: AP, CP.

HB-16 translates the complete prior local screen contract and integrates planning fixes. Original spec SHA-256: `074c4d8d91dd79523c7ac545f956359beae2648f6d490ee4ea5a93be2b9fcc25`. Original design JSON SHA-256: `ddb23aa02fc13c0b96a6a4eb2c358c486ab36c5821d66ab45f6d6261918d9fca`. Original source references above remain authoritative evidence; root retains the raw source snapshot. This document does not declare all 49 screens or the entire planning baseline complete.

Requirements: HB-03, HB-23, HB-24, HB-28, HB-32, HB-15.

Acceptance references: AUD-AR-01, AUD-AR-06, AUD-AR-07, AUD-QA-01, ERPC-01, ERPC-02, ERPC-03, ERPC-04, ERPC-05, ERPC-06, ERPC-07, ERPC-08.

## Monthly recognition approval details

Approval type monthly_recognition is required and read-only for monthly recognition approval. Bind the separately fixed result's legal entity, ledger, attribution/posting periods, currency, amount basis, batch hash, latest validation and external effect. Its target, hash and permissions differ from customer billing/issuance approval. Review these details before approving that batch only.

## Regions and reading order

Proposed layout: `workbench`.

### Monthly ERP recognition and quarterly billing reconciliation

Monthly transmission and actual posting confirmation are independent of billing.
Current monthly net plus approved adjustments not yet included equals quarterly comparison; never count an adjustment twice.

### Decision summary

Fix whose target is approved for which period and currency amount.

### Mandatory validation

Reconciliation, payment-detail-change verification, original eligibility and limit status.

### Approval area

Recheck current authority; approval itself does not mean issuance or delivery.

### Design status

Static proposal; product unimplemented; API disconnected; tests not run.
Use synthetic amounts and data only.
AUD links: AUD-AR-01, AUD-AR-06, AUD-AR-07, AUD-QA-01

### Approval stages

Billing-result approval may target a fixed calculation before an ERP draft exists. Final ERP amount approval follows external draft retrieval and reconciliation and binds that amount and external revision. Other policy approval targets a policy revision. Distinguish ERP not applicable from applicable but unknown. Unknown mandatory values block that stage and are never replaced with zero or not applicable.

## Field contract

| Label | Type | Required when | Editability | Validation | Sensitivity |
| --- | --- | --- | --- | --- | --- |
| Action, target and period | reference+interval | Always | Read only | Currently approvable scope | Contract |
| Author and actual approver | principal-reference | Always | Read only | Self-approval policy; no impersonation through delegation | Principal |
| Input and result hashes | digest | Always | Read only | Approval invalid if current result differs | Internal |
| Change evidence and blockers | reference-set | Always | Read only | Mandatory blockers cannot be bypassed | Restricted evidence |
| Rejection reason and handoff | text+principal | When applicable | Edit | Hand off only to an authorized owner | Internal |
| Approval type | enum | Always | Read only | Distinguish billing result, final ERP amount and other policy | Internal |
| Before/after amounts and taxes | money | For amount approval | Read only | Check amounts and limits by currency | Sales |
| Final ERP amount and external revision | money+revision | At final ERP amount approval | Read only | Reconciled ERP draft and current revision; required unknown values block | ERP amounts |
| Monthly recognition and quarterly reconciliation hash | read-only | For recognition-linked contracts | Read only | Permitted entity, contract and ERP scope; snapshot, currency and posting state | Internal accounting and contract |
| Unknown postings and unapproved adjustments | read-only | For recognition-linked contracts | Read only | Permitted entity, contract and ERP scope; snapshot, currency and posting state | Internal accounting and contract |
| Approval type monthly_recognition | enum | For monthly recognition approval | Read only | Separate target, hash and authority from customer billing approval | Internal accounting and contract |
| Input eligibility and candidate revision | enum+revision | Before eligibility or dependent amount approval | Read only except explicit command draft | Bind exact immutable evidence and current authority; no waiver | Restricted business evidence |
| Confirmation actor and evidence digest | principal+digest | Before eligibility or dependent amount approval | Read only except explicit command draft | Bind exact immutable evidence and current authority; no waiver | Restricted business evidence |
| Source, mapping and profile revisions | reference-set | Before eligibility or dependent amount approval | Read only except explicit command draft | Bind exact immutable evidence and current authority; no waiver | Restricted business evidence |
| Independent cross-validation and comparison digest | reference+digest | Before eligibility or dependent amount approval | Read only except explicit command draft | Bind exact immutable evidence and current authority; no waiver | Restricted business evidence |

Sensitive fields: Action, target and period, Author and actual approver, Change evidence and blockers, Before/after amounts and taxes, Final ERP amount and external revision.

## Tabs, filters and columns

Tabs: Recognition cadence and reconciliation / Pending / My decisions / Delegation and substitution / Approval history.

Filters: Entity and customer, Action, Period, Amount range, Author, Deadline, Blockers.

Columns: Monthly recognition and quarterly reconciliation hash → Unknown postings and unapproved adjustments → Approval ID → Action and target → Author → Period → Amounts by currency → Differences and warnings → Authority and limit → Deadline.

## Action contract

All identifiers below are proposed canonical planning IDs, not deployed APIs. Existing English permission tokens are preserved; descriptive permissions require the command catalog mapping, not an invented grant.

| Action | Proposed ID | Permission | Preconditions | Result | Confirmation |
| --- | --- | --- | --- | --- | --- |
| Approve monthly recognition | Existing action; catalog mapping required | Recognition approval for entity, ledger and limits with separation of duties | monthly_recognition type; matching batch hash, attribution/posting periods, currency, amount basis and current validation | Approve this batch only, separately from customer issuance approval | Confirm target, period, amounts and external effects |
| Approve | Existing action; catalog mapping required | Approval for the specific action | Separation of duties, current hashes, limits and mandatory checks satisfied | Record approval of a fixed result, separately from issuance | Final confirmation of action, target, amounts and hashes |
| Reject | Existing action; catalog mapping required | Approval for the specific action | Valid request and reason | Record rejection and revision path | Confirm rejection reason |
| Hand off to eligible owner | Existing action; catalog mapping required | Approval handoff | Target approval authority and expiry verified | Change owner who will approve under their own identity | Confirm deadline and authority |
| Compare evidence | Existing action; catalog mapping required | Approval evidence read | Permitted evidence available | Show result versus ERP amount differences | None |
| Inspect input evidence | input.inspect | Existing scoped evidence-read permission | Currently permitted exact candidate revision | Read input confirmation and validation evidence; no mutation | None |

## User-facing states

| State | Message |
| --- | --- |
| loading | Checking scope and permissions and loading Approval inbox and result cards. |
| empty | No approval requests currently need action. |
| partial | Only part of the data or work is ready. Show ready, missing, held, failed and unprocessed scopes separately; this is not full completion or zero. |
| blocked | Authority, amount limits, hashes or mandatory checks are not satisfied. Approval cannot bypass them. |
| error | The request could not be completed. Check the request ID and failed scope, then choose a safe retry or handoff. |
| revoked | Access to this scope has changed. Remove sensitive details, previews and selections, stop affected actions and downloads, and query permitted scope again. Recall of externally delivered copies is not guaranteed. |
| recognition-unreconciled | Monthly posting or billing reconciliation is incomplete. Check missing evidence, differences and adjustments. |
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

- [B08: Compare results](../B08/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [S05: Approval evidence](../S05/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [I02: ERP state](../I02/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [M03: Delegation scope](../M03/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [S06: Worker input confirmation](../S06/spec.md). Pass only allowed canonical context and recheck permission at destination.

## Source acceptance scenarios

All scenarios below remain NOT_RUN.

- AUD-AR-01 (AR-01): Run identical CSP inputs under direct-contract and resale routes. Direct contracts contain only service fees; resale contains only approved eligible sales components.
- AUD-AR-06 (AR-06): Cover customer exception/default conflicts, holidays, missing published rates, API failure, manual import, markup disguised as raw rates and duplicate conversion of same-currency fees. Already-converted provider amounts must not convert again; retain default discounts/fees-before-conversion order.
- AUD-AR-07 (AR-07): Target-total adjustments create explainable difference lines without overwriting source or existing ChargeLine. Changed results cannot submit under prior approval.
- AUD-QA-01 (QA-01): Reject bank-account changes based only on a new email and LLM-generated accounts. Reflect verified existing-channel checks and separate approval only from the effective date.

## Monthly ERP recognition and quarterly billing

Follow the [recognition source](../../../planning/v1.16/erp-recognition-cadence.md) and [English payment and recognition contract](../../../contracts/billing/payment-and-recognition.md). Customer billing cadence and ERP recognition cadence are separate. ERPC-01 through ERPC-08 remain NOT_RUN.

The Recognition cadence and reconciliation tab shows monthly batches, actual external posting, quarterly linkage, adjustments and unreconciled states. Missing or unknown months are not zero or completed. Corrections/retransmissions retain separate ERP execution/approval authority; inquiry/read access does not authorize execution. Show unsupported journal types as unsupported/manual handoff, not substitute customer invoices. Internal monthly recognition journals are not customer-public by default.

Before quarterly billing, verify support for offset/replacement/reversal of already-recognized amounts and the resulting external net. Current monthly net plus only approved adjustments not already included forms the quarterly comparison; distinct keys alone cannot prevent duplicate economic recognition. Unresolved differences and unknown external outcomes block final completion. Even an explicit issuance-gate exception does not erase unreconciled status. Monthly recognition transmission does not start payment terms. On small screens summarize monthly recognition, quarterly billing, difference and verification time with access to details. A zero arithmetic difference is not proof of complete external posting evidence.


## Unresolved values and verification limits

- Self-approval and multi-approver policy; trust changes prohibit self-approval.
- Amount limits and approval validity.

Field lengths, precision/scale policy values, permission enums, actual API addresses, nullable details, concrete error codes, page sizes, latency targets and supported product versions require detailed contract closure. The proposed command and numeric contracts define planning behavior, not chosen customer defaults. Current provider/ERP support, actual credentials and live data connections remain unverified. Static document/JSON checks cannot substitute for product behavior, accessibility, performance or live integration evidence.
