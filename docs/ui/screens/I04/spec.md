---
type: Screen Specification
title: I04 Disclosure policy and delivery packages
description: Create new files using only fields permitted for the recipient, then validate every row for disclosure and amounts before delivery.
status: draft
sources:
  - resource: ../../../planning/v1.16/ui-design.md
  - resource: ../../../planning/v1.16/detailed-requirements.json
  - resource: ../../../planning/v1.16/domain-contracts.md
  - resource: ../../../planning/v1.16/retention-access.md
  - resource: ../../../contracts/commands/planning-actions.md
---

# I04 Disclosure policy and delivery packages

## Purpose, roles and provenance

Create new files using only fields permitted for the recipient, then validate every row for disclosure and amounts before delivery.

Intended audience: admin. Roles: BL, AP, Disclosure policy owner.

HB-16 translates the complete prior local screen contract and integrates planning fixes. Original spec SHA-256: `894a38132b649cc68f7372800768b4b982c4605254e60f24e0fb77ebf0acc4a4`. Original design JSON SHA-256: `ac6291fe63843b77fb4344b18cc38db678117e20813ebfd29345e42ecc9f991f`. Original source references above remain authoritative evidence; root retains the raw source snapshot. This document does not declare all 49 screens or the entire planning baseline complete.

Requirements: HB-06, HB-24, HB-25, HB-26, HB-27, HB-28, HB-29, HB-34, HB-41, HB-43.

Acceptance references: AUD-DB-01, AUD-DB-05, AUD-DB-06, AUD-DB-07.

## Role interpretation

This is an admin screen for BL, AP and disclosure-policy owners, not a dedicated customer-public surface. The source role glossary distinguishes IA identity/access, OP installation/operations, AM organization administration, BL billing operations, AP eligible approval/issuance, PR contract/rate policy, CP internal cost and AU audit reading. Actual action, entity, customer and period authorization remains mandatory.

## Regions and reading order

Proposed layout: `workbench`.

### Policy on the left

Field classifications, exclusion reasons and inference risk from combinations.

### Public copy in the center

Recipient view and totals without costs; public references instead of source locators.

### Validation and delivery on the right

Full validation results, ERP automatic-delivery controls and current permission checks; independent of issuance state.

## Field contract

| Label | Type | Required when | Editability | Validation | Sensitivity |
| --- | --- | --- | --- | --- | --- |
| Disclosure purpose | enum | Always | Draft | Separate internal and customer purposes | Business |
| Recipient scope | reference[] | Always | Draft | Intersection of contract and current permissions | Personal |
| Allowed fields and aggregation | allowlist | Always | Policy draft | Exclude newly unclassified fields by default; check inference conflicts | Internal |
| Policy, generator and schema version | reference | Always | Read only | Fix sorting, date, sign and precision | Business |
| Input and file hash | digest | Always | Read only | Never replace a finalized artifact | Internal |
| Full row and amount validation | evidence | At delivery | Read only | Sampling alone cannot pass | Financial |
| Delivery path | enum | Always | Before delivery | Require evidence of control over automatic ERP routes | Business |
| Access expiry | datetime/policy | For links | Policy authority | Distinguish revocability and exposure window | Business |
| Hold type and target manifest | enum+reference | Hold request, release or status | Read only except explicit command draft | Separate billing/delivery/collections effects; unknown is not effective/released | Restricted business evidence |
| Hold and per-target revisions | reference+revision-map | Hold request, release or status | Read only except explicit command draft | Separate billing/delivery/collections effects; unknown is not effective/released | Restricted business evidence |
| Hold effect and external confirmation | enum+reference | Hold request, release or status | Read only except explicit command draft | Separate billing/delivery/collections effects; unknown is not effective/released | Restricted business evidence |

Sensitive fields: Recipient scope, Allowed fields and aggregation, Input and file hash, Full row and amount validation.

## Tabs, filters and columns

Tabs: Policy revision / Public preview / Packages / Delivery history.

Filters: Purpose, Recipient, Policy version, Verification state, Delivery state.

Columns: Package ID → Target document → Recipient → Policy revision → Rows and totals → Validation → Delivery.

## Action contract

All identifiers below are proposed canonical planning IDs, not deployed APIs. Existing English permission tokens are preserved; descriptive permissions require the command catalog mapping, not an invented grant.

| Action | Proposed ID | Permission | Preconditions | Result | Confirmation |
| --- | --- | --- | --- | --- | --- |
| Generate public copy | Existing action; catalog mapping required | Public-file generation permission | Recipient, policy and inputs fixed | Create a new file without internal original links | Confirm fields, targets and purpose |
| Validate all rows | Existing action; catalog mapping required | Validation permission BL/AP | Generated artifact exists | Check fields, filenames, ZIP contents, cache and amounts | None |
| Request delivery | Existing action; catalog mapping required | Delivery permission | Full validation passed, current recipient rights and route verified | Accept delivery job | Confirm recipients, channel and file hash |
| Redeliver same package | Existing action; catalog mapping required | Delivery permission | Issued document unchanged and authority valid | Retry delivery only | Confirm duplicate delivery targets |
| Revoke link access | Existing action; catalog mapping required | Disclosure revocation permission | Current link scope | Block managed access; no guarantee of recalling external copies | Confirm affected links |
| Request hold | holds.request | Current scoped authority for the hold type and target action | Explicit type, target revisions, reason/evidence and required policy | Report per-target requested/effective/released/pending/unknown effect without auto-resume | Confirm type-specific effect; due dates and balances do not change |
| Request hold release | holds.release | Current scoped authority for the hold type and target action | Explicit type, target revisions, reason/evidence and required policy | Report per-target requested/effective/released/pending/unknown effect without auto-resume | Confirm type-specific effect; due dates and balances do not change |
| Inspect hold status | holds.status | Scoped hold read for the authorized hold and target | Current authorized hold/case ID and target scope; no new mutation rationale or approval evidence | Read current revisions, as-of, hold effects and external uncertainty without mutation | None; read-only status inspection |

## User-facing states

| State | Message |
| --- | --- |
| loading | Checking scope and permissions and loading Disclosure policy and delivery packages. |
| empty | No items match the current permitted scope and filters. Check scope and filters. |
| partial | Only part of the data or work is ready. Show ready, missing, held, failed and unprocessed scopes separately; this is not full completion or zero. |
| blocked | Disclosure validation or automatic ERP delivery controls are unproven. Delivery is blocked. |
| error | Package generation or delivery failed. Retry only the affected stage without reissuing the invoice. |
| revoked | Access to this scope has changed. Remove sensitive details, previews and selections, stop affected actions and downloads, and query permitted scope again. Recall of externally delivered copies is not guaranteed. |
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

- [I01: Open I01](../I01/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [I03: Open I03](../I03/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [C02: Open C02](../C02/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [P04: Open P04](../P04/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [R02: Open R02](../R02/spec.md). Pass only allowed canonical context and recheck permission at destination.

## Source acceptance scenarios

All scenarios below remain NOT_RUN.

### DB-01 / AUD-DB-01

Source locators: B1013, B1051, B1535. Standard settlement XLSX, customer XLSX, result registration and the common API/file submission ledger are baseline common capabilities. Only specific ERP upload templates and actual activation depend on product/version evidence. An API-only customer does not remove common file-feature tests. Complete first reconciliation with standard XLSX and result registration without API connectivity; never label an unverified ERP template verified. Release gates block omission of common capabilities.

### DB-05 / AUD-DB-05

Source locator: B0992. Diagnose automatic ERP attachments, mail and portal disclosure as connection capabilities. Without proof of recipient-specific field/attachment controls, do not activate automatic delivery; allow only approved routes. Platform export validation alone does not prove safe ERP delivery. An ERP template automatically attaching an internal cost file blocks automatic-delivery activation independently of issuance/posting.

### DB-06 / AUD-DB-06

Source locators: B0726, B1011, B1037. Supply source as nonexecutable attachments. Defend HTML previews and formula injection; block PDF renderer external resource requests, XLSX macros and external data connections. Distinguish machine raw values from safe viewing derivatives so derivatives cannot be mistaken for originals. Verify format, precision and safety in actual supported spreadsheets. Test HTML/formula payloads, external-URL PDFs and macro/external-link XLSX for no external execution/calls while preserving long IDs, decimals and machine raw values.

### DB-07 / AUD-DB-07

Source locators: B0999, B1001. ExportArtifact fixes input, disclosure policy, generator and schema versions, stable sort keys, dates, timezones, decimal places, encoding, delimiters and signs. Separate volatile metadata such as generation time from semantic data. Redownload of a finalized artifact returns the same stored bytes; regeneration compares the designated normalized data result. Changing chunk/worker/retry order under the same version preserves data results. Generator changes produce a new version and cannot silently replace finalized files.

## Unresolved values and verification limits

- Allowed fields and inference-conflict policy.
- Direct link permission and expiry.
- Controllable ERP delivery paths.

Field lengths, precision/scale policy values, permission enums, actual API addresses, nullable details, concrete error codes, page sizes, latency targets and supported product versions require detailed contract closure. The proposed command and numeric contracts define planning behavior, not chosen customer defaults. Current provider/ERP support, actual credentials and live data connections remain unverified. Static document/JSON checks cannot substitute for product behavior, accessibility, performance or live integration evidence.
