---
type: Screen Specification
title: M02 Identity providers and login policy
description: Test organization authentication routes and trust contracts, verify recovery paths, then activate.
status: draft
sources:
  - resource: ../../../planning/v1.16/ui-design.md
  - resource: ../../../planning/v1.16/detailed-requirements.json
  - resource: ../../../planning/v1.16/domain-contracts.md
  - resource: ../../../planning/v1.16/retention-access.md
  - resource: ../../../contracts/commands/planning-actions.md
---

# M02 Identity providers and login policy

## Purpose, roles and provenance

Test organization authentication routes and trust contracts, verify recovery paths, then activate.

Intended audience: admin. Roles: IA, OP.

HB-16 translates the complete prior local screen contract and integrates planning fixes. Original spec SHA-256: `1172749b27c04c9f2047163b73b9fa73238655efd5efdfa79ece3db5353689ff`. Original design JSON SHA-256: `2ac0bc852df8b0c5f975bcb4c161848df8d5426b231e8943a66c269019dabcdf`. Original source references above remain authoritative evidence; root retains the raw source snapshot. This document does not declare all 49 screens or the entire planning baseline complete.

Requirements: HB-03, HB-36, HB-42.

Acceptance references: IAM-01, IAM-02, IAM-03, IAM-04, IAM-05, IAM-06, IAM-07, IAM-08.

## Role interpretation

This is an admin screen, not a dedicated customer-public surface. IA is identity/access administration; OP installation/operations; AM organization administration; BL billing operations; AP eligible approval/issuance; PR contract/rate policy; CP internal cost in the source role glossary; AU audit reading. These labels do not grant global access. Authentication method order is a validation-planning choice, never permission to remove local/SAML/OIDC/LDAP scope.

## Regions and reading order

Proposed layout: `form`.

### Method selection on the left

Distinguish direct LDAP from federated SSO.

### Trust inputs in the center

Method-specific conditional fields; never expose plaintext secrets in lists.

### Tests and recovery on the right

Assess subject, MFA and permissions separately; authentication failures must not disclose account existence.

### External-status verification and detection delay

Show per-connection event/synchronization/polling method, last attempt, last success, next check, maximum delay and failure reason. Check intervals and delay values are unset; do not invent defaults.
Unknown external status holds sensitive commands. A recheck receipt or ordinary successful login does not release the hold; evidence must verify current inactivity, groups and scope. Product-local suspension blocks from the next permission check.

## Field contract

| Label | Type | Required when | Editability | Validation | Sensitivity |
| --- | --- | --- | --- | --- | --- |
| Method and workspace route | enum+reference | Always | Draft | Distinguish local, SAML, OIDC and LDAP | Internal |
| Issuer and subject contract | text+policy | SAML/OIDC | Draft | Stable subject; no automatic email linking | Internal |
| Certificate and key reference | secret reference | External authentication | Draft | Verify trusted origin, expiry and rotation | Secret reference |
| OIDC redirect/audience | uri+text | OIDC | Draft | Exact registration; PKCE, state and nonce checks | Internal |
| LDAP endpoint, CA and search scope | uri+reference | Direct LDAP | Draft | TLS mandatory; reject empty bind; stable identity | Internal |
| Allowed methods and enforced SSO | policy | Always | Draft | No automatic local bypass during outage | Internal |
| MFA and reauthentication evidence | policy+evidence | Sensitive commands | Draft | An SSO label alone does not satisfy requirements | Internal |
| Group mapping and role ceiling | mapping | When synchronizing | Draft | Allowlist; no automatic JIT administrator grants | Internal |
| Recovery-path evidence | evidence | At activation | Verification record | Last-administrator access; no resurrection of revoked access | Internal |
| External-status verification method | enum+adapter contract | External identity | Policy draft | Distinguish events, synchronization and polling; login alone does not detect departures | Internal |
| Last success and next check | datetime pair | External identity | Read only | Separate last attempt from last success; missing signals do not imply active | Internal |
| Maximum permitted detection delay | duration policy | External identity | Policy draft | Unset value; determine through supported adapter and operational contract | Internal |
| Synchronization state and failure reason | enum+reason | External identity | Read only | Distinguish healthy, delayed, failed and unknown; unknown external status holds sensitive commands | Internal |
| Old and proposed trust epoch | revision-pair | Trust change | Read only except explicit command draft | Bind exact proposal and current authority; changed evidence invalidates independent approval | Restricted business evidence |
| Old/new configuration and identity evidence digests | digest-set | Trust change | Read only except explicit command draft | Bind exact proposal and current authority; changed evidence invalidates independent approval | Restricted business evidence |
| Proposer, independent approver and activator | principal-set | Trust change | Read only except explicit command draft | Bind exact proposal and current authority; changed evidence invalidates independent approval | Restricted business evidence |
| Session impact and invalidation status | reference+enum | Trust change | Read only except explicit command draft | Bind exact proposal and current authority; changed evidence invalidates independent approval | Restricted business evidence |

Sensitive fields: Method and workspace route, Issuer and subject contract, Certificate and key reference, OIDC redirect/audience, LDAP endpoint, CA and search scope, Allowed methods and enforced SSO, MFA and reauthentication evidence, Group mapping and role ceiling, Recovery-path evidence, External-status verification method, Last success and next check, Maximum permitted detection delay, Synchronization state and failure reason.

## Tabs, filters and columns

Tabs: Provider connections / Organization login policy / Test login / Keys and certificates / Recovery paths / External status synchronization.

Filters: Method, Workspace, Status, Certificate expiry.

Columns: Connection → Method → workspace route → Diagnostics → MFA evidence → Key expiry → Active revision.

## Action contract

All identifiers below are proposed canonical planning IDs, not deployed APIs. Existing English permission tokens are preserved; descriptive permissions require the command catalog mapping, not an invented grant.

| Action | Proposed ID | Permission | Preconditions | Result | Confirmation |
| --- | --- | --- | --- | --- | --- |
| Diagnose connection | Existing action; catalog mapping required | Authentication configuration permission | Allowed endpoint and secret references | Return error, version and trust validation results | Confirm external diagnostic scope |
| Test login | Existing action; catalog mapping required | Authentication test permission | Draft settings and test principal | Assess identity and MFA; not business approval | Confirm test scope |
| Request policy activation | Existing action; catalog mapping required | Policy approval permission | Diagnostics, recovery and enforced SSO verified | Plan activation of a new revision through the trust workflow | Confirm affected users and lockout risk |
| Plan key rotation | Existing action; catalog mapping required | Key management permission | Old and new key validity and verification | Plan rotation and failure recovery | Confirm interruption impact |
| Recheck external status | Existing action; catalog mapping required | Diagnostic permission for this authentication connection | Allowed adapter, scope and destination with current connection authority | Requery and reconcile external inactivity and group changes; update successful evidence and last-success time; retain sensitive-command hold while unknown | Confirm targets and external-call scope |
| Propose trust change | trust.propose | Scoped trust.propose | Current connection revision and expected old epoch; exact proposal and proof references | Create a draft proposal only; active trust unchanged | Review proposed scope and old/new digests |
| Validate trust change | trust.validate | Scoped trust.validate | Draft proposal at expected revision; exact configuration, proof and old epoch | Record validated or blocked evidence; no approval or activation | Review validation scope and evidence |
| Independently approve trust change | trust.approve | Scoped trust.approve by an independent authorized actor | Validated proposal at expected revision and digest; current old epoch; approver differs from proposer | Record independent approval of the exact proposal; active trust unchanged | Confirm exact digest, authority and session/lockout impact |
| Activate approved trust change | trust.activate | Scoped trust.activate plus current independent approved authority | Independently approved revision/digest; expected old epoch; current proposer/approver separation and approver/activator authority | Atomically publish configuration, strictly newer epoch and durable invalidation intent; report propagation separately | Confirm approved digest, affected principals and session/lockout impact |
| Reject trust change | trust.reject | Scoped trust.approve | Current proposal revision and rationale; before activation | Mark proposal rejected; active trust unchanged | Confirm rejection and rationale |
| Cancel trust proposal | trust.cancel | Authorized trust proposal cancellation scope | Current proposal revision and rationale; before activation | Mark proposal cancelled; active trust unchanged | Confirm proposal cancellation |

## User-facing states

| State | Message |
| --- | --- |
| loading | Checking scope and permissions and loading Identity providers and login policy. |
| empty | No items match the current permitted scope and filters. Check scope and filters. |
| partial | Only part of the data or work is ready. Show ready, missing, held, failed and unprocessed scopes separately; this is not full completion or zero. |
| blocked | Test login or recovery path is unverified. Hold policy activation. |
| error | Provider connection failed. Do not disable TLS validation or automatically switch to another login method. |
| revoked | Access to this scope has changed. Remove sensitive details, previews and selections, stop affected actions and downloads, and query permitted scope again. Recall of externally delivered copies is not guaranteed. |
| external_status_unknown | External account status is unknown. Sensitive commands remain held; request a status recheck. |
| sync_failed | External status synchronization failed. Check the last success and delay; do not assume the account is active. |
| draft | Trust change is a draft; active trust is unchanged. |
| validated | Exact proposal evidence is validated; independent approval is required. |
| independently_approved | Independent approval is recorded; activation still rechecks current authority and old epoch. |
| activated | The new epoch and invalidation intent are committed. Inspect remaining invalidation progress. |
| rejected | The proposal is rejected; active trust is unchanged. |
| cancelled | The proposal is cancelled; active trust is unchanged. |

## Trust change and activation

The proposed [command contract](../../../contracts/commands/planning-actions.md) uses `trust.propose`, `trust.validate`, `trust.approve`, `trust.activate`, `trust.reject` and `trust.cancel`. These are planning IDs, not deployed APIs. Existing diagnostic, test-login, activation-request, key-rotation and external-status actions remain, but activation must use this governed transition rather than a generic bypass.

A proposal binds workspace/provider route, expected old trust epoch and revision, old/new configuration digests, issuer/stable subject/identity binding, verified identity/MFA proof, diagnostics, recovery evidence, current external identity status, session impact and affected subjects. Show changes and lockout/revocation consequences before submission. States are `draft → validated → independently_approved → activated`, with explicit rejected/cancelled outcomes. Changing the candidate or bound evidence invalidates validation and approval. Verified provider key rotation under an unchanged trust root remains a separate validated rotation. Changes to the trust root, issuer or identity namespace, or weakened MFA, require this independent trust workflow; rollback is a new independently approved change with a strictly newer epoch, never resurrection of an old epoch or revoked grant.

Proposer, approver and activator have distinct rights. Self-approval is prohibited. Activation requires an independently approved proposal and an authorized activation actor whose current authority is rechecked against the old epoch at execution. A matching role name, successful test login or possession of the proposed new identity cannot confer approval or activate its own authority. Missing independent approval, stale epochs, unknown external identity status, failed recovery or insufficient current privileges block activation. An emergency policy remains separately unresolved; no generic recovery bypass is introduced.

Activation compares the old epoch/revision and atomically commits the new epoch with durable invalidation intent for affected sessions, credentials and cached permissions. Display invalidation progress and any unknown completion; do not claim all sessions revoked from acceptance alone. Concurrent proposals cannot both activate against the same old epoch. Idempotent retry rechecks present authorization before revealing prior protected results; mismatched payload conflicts. Stale evidence requires a fresh proposal/review, not silent rebasing. Reject/cancel are legal only before activation and retain reasons and history without mutating active trust.

Acceptance (NOT_RUN): reject self-approval, replay from old epochs and simultaneous activation; candidate mutation invalidates approval; revoked approver/activator is rechecked; timeout does not imply activation or completed invalidation; rollback gets a newer epoch; external unknown retains the sensitive-command hold.


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

- [M01: Open M01](../M01/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [M03: Open M03](../M03/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [L01: Open L01](../L01/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [L03: Open L03](../L03/spec.md). Pass only allowed canonical context and recheck permission at destination.
- [P01: Open P01](../P01/spec.md). Pass only allowed canonical context and recheck permission at destination.

## Source acceptance scenarios

All scenarios below remain NOT_RUN.

- IAM-01 through IAM-08: Link the applicable creation, trust, role, revocation, recovery and bulk paths. Actual authentication/access tests remain NOT_RUN.

## Unresolved values and verification limits

- Authentication adapter/product validation order; all required methods remain in scope.
- MFA and session lifetime.
- JIT, synchronization and emergency-access policy.
- External-status check interval, maximum detection delay and adapter evidence.

Field lengths, precision/scale policy values, permission enums, actual API addresses, nullable details, concrete error codes, page sizes, latency targets and supported product versions require detailed contract closure. The proposed command and numeric contracts define planning behavior, not chosen customer defaults. Current provider/ERP support, actual credentials and live data connections remain unverified. Static document/JSON checks cannot substitute for product behavior, accessibility, performance or live integration evidence.
