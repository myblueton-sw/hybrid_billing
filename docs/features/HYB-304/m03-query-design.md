---
type: DesignPlan
title: M03 bounded access queries (HYB-304 increment C)
description: Design of the bounded, independently paged reads for the M03 access screen under the common contract of the HYB-304 plan, split into reviewable increments.
status: draft
---

# Scope and ownership

HYB-304 increment C, machine:mac_mini. Root owns design, code and integration; the independent reviewer and the ui-designer are read-only. This document is the design to be reviewed before any code. It follows the common contract of [the HYB-304 plan](plan.md) and reuses the mechanics built for M04 ([query contract](m04-queries.md)). It changes who can read nothing: every read stays behind installation administration, as `listAccess` is today.

# Facts

- `GET /api/v1/identity/access` (`apps/api/server/access.go` `listAccess`) answers one page holding every grant, every delegation, every eligible subject, change requests, every kept revocation, the action catalog, the settings rows, the caller, the start time and the decision ledger row. `GET /api/v1/identity/access/preview` answers one effect preview. Both are admitted for installation administrators only (`access.go:175`).
- Grants and delegations come from the auth registry through `gate.Administration.Grants()` and `Delegations()` (`modules/auth/listing.go:40-66`), which copy and sort the whole population on every call. `Active` and `Effective` are computed there at the module's current time; `Effective` also depends on membership and on a person being an active human. Grants for installation actions are left out of the grant rows (`access.go:194-197`). A delegation cannot carry an installation action: the registry refuses one (`modules/auth/registry.go:385`).
- The registry's `Generation()` advances on every stored change, including a change of a subject (`registry.go:136` and its nine callers). It does not advance when a validity period begins or ends, since that changes nothing stored.
- Change requests are not filtered per caller: every installation administrator sees all pending requests and the most recent ended ones up to the setting `EndedShown` (50 by default, at most `EndedKept`), newest number first; only each row's eligibility depends on the caller (`accessrequests.go:418-431`). The rows are built from `accessFacts`, which reads the whole member roster and pages through every user on each call (`accessrequests.go:28-47`).
- `revocationRows` returns every kept revocation, up to `RevocationsKept`, each with its history and cleanup items (`apps/api/server/revocations.go:148-167`). Revoking a grant ends its dependent delegations on the server (`EndedDelegs`).
- The accesschange registry has no version that advances on every change: its counters are the last request and revocation numbers, which move only when one is created. An approval, rejection, withdrawal or a revocation's progress changes a row without moving them.
- `grantees` reads the whole member roster (`access.go:155-169`). The catalog (`ceiling`) walks every grant (`access.go:145-153`). `ledgerRow` reads only the ledger's status.
- The M03 specification (`docs/ui/screens/M03/spec.md`, in the authorized local workspace) asks for the columns subject, action, target scope, source, original delegation, validity and effective permission; the filters subject, action, customer or legal entity, expiry and permission source; server search with a stable order; selection kept by canonical ID; and lists read by a fixed snapshot or revision with a cursor, never the whole population in the browser.
- The console screen (`apps/console/src/modules/identity/pages/access/Live.tsx`, 576 lines) reads the one page and derives its four views (roles, grants, effects, support) from it.
- The M04 source basis includes the service revision (`servicequery_cursor.go` `serviceBasis`), so any write in the installation ends an M04 cursor.
- Not read for this design: the accesschange module's internals beyond what is cited; the M03 specification beyond the lines the plan cites; how membership reaches the auth registry beyond the fact that subject changes advance its generation.

# Design

## Reads

Five bounded reads replace the lists of the one page. Each is admitted as `listAccess` is, answers only for the caller's workspace, and follows the common contract: an allowlist of parameters with one value each; a bounded search; one stable order with a canonical tie-breaker; a signed cursor bound to the list, the normalized filters, the caller and session, the reference time and the source basis; a scan budget separate from the page size; a count that says whether it is exact or a lower bound; `409` with the restart convention when the cursor no longer fits.

| Read | Rows | Filters | Order |
| --- | --- | --- | --- |
| `GET /identity/access/permissions` | grants and delegations as one list with `source` (`grant`, `delegation`), the original delegation where there is one, validity, state at the reference time and the effective flag | `q` over subject and identifier; `subject`; `action`; `customer`; `legal_entity`; `source`; `state` (`active`, `scheduled`, `expired`, `closed`); `expires_before` | source, then identifier |
| `GET /identity/access/permissions/grant/{id}` and `…/delegation/{id}` | one grant or one delegation with its eligibility for revocation | — | — |
| `GET /identity/access/requests` | pending requests and ended requests, as today | `state` (`pending`, `ended`); `q` over number and subject | number descending, as today |
| `GET /identity/access/revocations` | kept revocations | `q` over target | newest first |
| `GET /identity/access/subjects` | people who may hold a grant, for the pickers | `q` over login; exact `id` | login, then identifier |

Visibility is kept as it is. Every installation administrator sees every row; the caller changes only a row's eligibility. The requests read keeps the `EndedShown` cap: it does not page into ended requests the screen cannot reach today. Installation-action grants stay out of the permissions list, and their detail read answers exactly as an unknown identifier does, as does an identifier of another workspace or of the other source. Filters and counts therefore tell an administrator nothing the one page does not tell today.

`GET /identity/access` keeps what is not a population: the catalog, the settings rows, the caller, the start time, the ledger row and the reference time. Its `grants`, `delegations`, `subjects`, `requests` and `revocations` fields are removed in the last increment, once the screen no longer reads them. The preview read is unchanged.

## Basis, reference time and cursor

The cursor mechanics of M04 (`servicequery_cursor.go`: sealing, opening, the walk with its scan budget, the page writer) are made generic rather than copied: the area-specific parts are the parsed query and the source basis.

The access basis is the auth registry's generation, a new accesschange version that advances on every change of a request or a revocation (creation, approval, rejection, withdrawal, expiry handling, job and cleanup progress), and whether a trust hold is in force. It does not include the service revision: an unrelated write elsewhere should not end an access cursor. A cursor from before a change of basis answers `409`.

A list is evaluated at its reference time, fixed when the first page is read and carried in the cursor. The state of a permission (`active`, `scheduled`, `expired`, `closed`) and its `effective` flag are computed with validity evaluated at that time, and with membership and person standing as the registry stores them when the page is read; a change of membership or standing advances the registry's generation, which ends the cursor with `409`. The module's bounded reads take the time as a parameter, and so does the registry's check that a delegate still holds in the workspace, which depends on validity. A validity period that begins or ends after the reference time therefore changes no row of that list and needs no `409`; the page says its reference time, and a fresh query gives a fresh one. The detail read of one permission is evaluated at its own observation time and says so, as M04's detail reads do, because eligibility for a command must be current.

## Module reads and scan bounds

- **Auth registry.** `GrantPage(after, limit, at)` and `DelegationPage(after, limit, at)` in identifier order, with counts, so that the server does not copy and sort the population per call. If the registry cannot give identifier order without a full sort, it gains an ordered index, as the identity module did for credentials. The whole-list functions stay for effect preview and eligibility.
- **Accesschange registry.** Bounded reads of pending and ended requests in number order and of revocations newest first, and the version described above.
- **People.** Request rows and subject rows need a person's login and standing. They are looked up for the rows emitted only, through the identity module's existing bounded member and user reads (or a by-identifier read where one exists), never by reading the whole roster per call as `accessFacts` and `grantees` do. The subjects read pages the roster with the identity module's cursor.
- **Catalog.** The main read keeps the catalog, which walks every grant today. It moves to a count-bounded walk over the grant index with the scan budget; if the budget ends first the catalog says it is incomplete rather than presenting a partial ceiling as whole. Whether the catalog can be derived without reading grants is checked in C1.

Every read charges its scan budget for rows examined, not rows emitted, and reports `scan_stopped` and a lower-bound count when it stops early, as M04 does.

## Dependent effects

M03 has no bulk action on the screen. Revoking a grant ends its dependent delegations on the server; that cascade stays there and is not changed by this design. A revocation's affected delegations are shown from the revocation row, as today.

## Screen

The four views stay. The grants view becomes the permissions list with the specification's columns and filters, a selected row's detail read by identifier, and paging controls, built from the M04 pieces (`QueryFilters`, `PageInfo`, `ReadBoundary`, `Picker`, the scoped read) moved to a shared place rather than imported across page folders. Requests and revocations page independently. Subject pickers search the subjects read. Commands and their independent approval are not changed; a command's eligibility comes from the detail read of its target, not from whether the row is on the current page.

## Increments

1. **C1a.** Generic cursor mechanics extracted from M04 with no behaviour change: M04's tests unchanged and passing, its rotation runner rerun. One commit, nothing else in it.
2. **C1b.** The auth registry's bounded reads with the time parameter, and the accesschange version and bounded reads. Module tests, including the time parameter through the delegation chain. A by-identifier member and user read is added to the identity module if none exists. Server reads not yet changed.
3. **C2.** The permissions read and its two detail reads, with tests for every filter, the cursor's bindings and refusals, the reference-time rule, hidden-equals-unknown, the scan budget and the count certainty.
4. **C3.** The permissions list on the screen, with the shared query pieces. Rendered check.
5. **C4.** Requests, revocations and subjects reads with their bounded people lookups, and their use on the screen.
6. **C5.** Removal of the five population fields from `GET /identity/access`, and the bounded catalog.

Each increment has its own independent review; C2, C4 and C5 change a public read.

# Decided for C1 and C2

- **Settings.** The page size, scan budget and cursor lifetime reuse the M04 settings (`ServiceQueryPage`, `ServiceQueryScan`, `ServiceQuerySeconds`) under their existing bounds; no access-specific settings are added.
- **`expires_before`** is an instant in RFC 3339 form, compared with the stored end of validity; a date would need a time zone the screen does not have.
- **States** are those `stateOf` returns today, evaluated at the reference time.

# Progress

- **C1a, 2026-11-26.** `apps/api/server/querycursor.go` holds the signed token, the binding check, the expiry and the scan budget; `servicequery_cursor.go` uses them. Token format, domain strings, key and checks are unchanged. The scan limit is now read once when a walk begins instead of on each use; access settings are fixed at startup (`server.go:302`), so the two are the same. Verified: the whole server package passes; the M04 rotation runner passes (55 checks); new unit tests for the generic pieces. Independent review: approve, behaviour preserved.

- **C1b, auth registry, 2026-11-26.** The registry keeps its grants and delegations in identifier order as they are added (nothing removes one) and gains `GrantPage`, `DelegationPage`, `GrantAt`, `DelegationAt` with the evaluation time as a parameter, and `Counts`; the gate module exposes them. `Grants()` and `Delegations()` now walk the same order instead of sorting per call and return what they did. Tests: pages of several sizes equal the whole list; cursors between and past identifiers; copies do not share terms; validity evaluated at the given time through the delegation chain while standing is read as stored and a change of standing advances the generation. The accesschange version and reads and the by-identifier people read are the rest of C1b.

# Open for the owner

- **C5 removes fields from an existing public read.** The plan's common contract requires it ("a new paged endpoint alone is insufficient if the old screen entry still downloads all rows"), and M04 did the same to the services read. It is listed here because a removal is a contract change; it will be asked before C5 is built.
- **One list for grants and delegations.** The specification's columns include source and original delegation, which reads as one list. The screen today shows them as two tables. This design follows the specification; it is a screen decision and will be put to the ui-designer and the owner before C3. The server read serves either: the `source` filter gives each table.
- **Selection across a whole filter.** The specification's fixed server target manifest for "select all matching" applies to bulk actions. M03 has no bulk action today, so none is designed here.

# Review

Independent design review (role-based AI reviewer separate from the writer, by reading, nothing built), 2026-11-26: approve with conditions. Four blocking findings, all taken into this revision: the basis could not see most row changes and time-driven state was unaddressed; the detail read had no identifier namespace or hidden-equals-unknown rule; scan bounds for requests, subjects and the main read were not designed; the facts misstated request visibility and omitted the `EndedShown` cap. Its notes on the service revision in the basis, the request order and the delegation cascade are also taken in. Bounded re-review, same day: approve, with one sentence corrected to say that membership and standing are read as stored and guarded by the generation, and a by-identifier people read added to C1b.
