---
type: DesignPlan
title: M03 screen — what replaces each read of the whole page
description: Every place the M03 grants view still reads a population from the whole access read, what replaces it, and the additive facts the bounded reads still lack.
status: draft
---

# Scope and ownership

HYB-304 increment C4c, machine:mac_mini. Root wrote this on 2026-11-26 from the console code on main `2dddcdec`; nothing was built for it. It plans how the M03 grants view stops reading the five population fields (`grants`, `delegations`, `subjects`, `requests`, `revocations`) of `GET /identity/access`, so that C5 of the [query design](m03-query-design.md) can remove them. It lists what must be added to the bounded reads first. Screen composition is the ui-designer's and is not decided here.

# What the view reads today

Paths are under `apps/console/src/modules/identity/pages/access/`.

| Use | Where | Population read |
| --- | --- | --- |
| Requests section: pending and ended rows, each person's sign-in name, the name in the decision's confirmation | `Requests.tsx:43-45`, `:78`, `:130-141`; `request-model.ts:6` | `requests`, `subjects` |
| The request being decided in the side panel | `Live.tsx:416` | `requests` |
| Revocations section rows | `Revocations.tsx:97-98` | `revocations` |
| A revocation's target: the grant's holder and access | `Revocations.tsx:34-36` | `grants`, `subjects` |
| Person choices in the request form and the effect check, and the sign-in name in the check's result | `Live.tsx:50`, `:254`, `:270` | `subjects` |
| Warning that a chosen person has a revocation not yet confirmed | `Live.tsx:297`; `request-model.ts:19-26` | `revocations`, `grants`, `delegations` |
| Delegations that end when one is revoked | `Live.tsx:207`; `request-model.ts:113-122` | `delegations` |
| The console ceiling table | `Live.tsx:414`; `live-model.ts:36` | `grants` |

Three helpers read populations and are called only from tests: `people` and `delegationsOf` (`live-model.ts`) and `pendingRevokeFor` (`request-model.ts`). Outside the view, the rendered check `apps/console/tools/access-approval-qa.mjs` reads `requests` and `grants` from the whole read, and the whole read's parser in `service/admin.ts` parses the five fields. No other screen calls `api.access()`.

What is not a population stays on the whole read: `revision`, `now`, `you`, `started_at`, `settings`, `ledger` and `catalog` (the catalog's bound is C5's).

# What replaces each

1. **Requests section and the decision panel.** `accessRequests` with `state=pending` and `state=ended`, each paged. Rows carry the three sign-in names. The decision panel takes the row from the list it was opened from. No new server fact.
2. **Revocations section.** `accessRevocations`, paged. The target's holder and access are not in a revocation row. **New additive fields** on the revocation row: `target_subject` (a grant's holder, a delegation's delegate), `target_subject_login`, `target_action`, `target_resource_kind` and `target_fields`, filled from the grant or delegation by identifier when it exists, empty otherwise; the existing `target_present` already says when it is gone. The row is built by one function for the whole read and the bounded read, so the fields appear on both; the binding entry covers both. A grant or delegation is only added and closed, never changed, so its holder and terms now are those at the revocation; only the sign-in name is read now. Nothing is shown that an administrator cannot already see.
3. **Person choices.** A search over `accessSubjects` in place of the full list; the composition (a combobox or a search field with results) is the ui-designer's. The effect check uses the same picker.
4. **Unconfirmed revocation warning.** The rule is "some revocation that is not confirmed targets a grant or delegation of this person". **New additive field** on the subjects row: `revocation_unconfirmed`, true under exactly that rule. The server builds, once per page, the set of holders of kept revocations that are not confirmed (each target resolved by identifier) and looks each row up in it. Revocations that are not confirmed are not pruned, so the scan has the read's limit and answers 503 above it.
5. **Delegations that end.** The browser's rule (`descendantsOf`) and what the server ends differ. The server ends the delegations that were effective before and are not after (`endedSince` in `revocations.go`); a delegation is effective only while every link of its chain holds and its root is backed by a grant. The browser lists children by parent that are neither revoked nor expired, so it includes pending children and children that already do not apply, and it shows nothing for a grant although revoking a grant can end the delegations it backs. The replacement follows the server: a **new additive field** `would_end` on the permission detail read of a grant and of a delegation, the identifiers of the delegations that are effective now and would not be if the target were revoked, the set `ended_delegations` then reports. The screen shows it for both kinds of target; this changes what the panel lists, toward what will happen, and is recorded for the owner at screen acceptance. The set is computed in the auth module by evaluating effectiveness with the target treated as closed. For a delegation target: its effective descendants by parent. For a grant target: the holder's root delegations that are effective now and would no longer be backed without this grant, judged against the holder's other grants by the same backing rule as `AddDelegation`; then their effective descendants by parent. This needs a lookup of delegations by parent, which R1 did not add, together with R1's lookups of a user's delegations and a subject's grants; it is part of T2, each lookup with the read's limit and 503 above it. Tested against `ended_delegations` after an actual revocation of each kind of target.
6. **Console ceiling table.** The console's grants are already rows of the one permissions list (holder "Console"). The separate table is removed, a composition change the owner sees at screen acceptance; the binding rows of the table are updated in the same change. The request form's choices come from the catalog, not from these grants.

# Increments

Each is one reviewed change; server increments are additive fields with independent review and a binding-document entry, as the owner allowed on 2026-11-24.

- **T1.** Server: the revocation row's target facts (item 2). Console client accepts them.
- **T2.** Server: `revocation_unconfirmed` on the subjects row (item 4); the auth lookup of delegations by parent and `would_end` on the detail read (item 5). Console client accepts them.
- **T3.** Screen: requests section and decision panel from `accessRequests` (item 1). ui-designer composition for paging two lists; rendered check.
- **T4.** Screen: revocations section from `accessRevocations` (item 2).
- **T5.** Screen: the person picker, the warning and what would end from the bounded reads; the ceiling table removed (items 3 to 6); the three unused helpers and their tests removed; `access-approval-qa.mjs` reads the bounded reads.
- After T5 the view reads no population from the whole read, and C5 can be put to the owner.

# Progress

- **T1, 2026-11-26.** `revocationRow` in `apps/api/server/revocations.go` adds `target_subject`, `target_subject_login`, `target_action`, `target_resource_kind` and `target_fields` to a revocation row, on the whole read and the bounded read alike, read by identifier at the current time and empty when the record is not present. The console's strict client requires and checks them. Tests: a revoked grant, a revoked delegation (the delegate is named), the whole read's row equal to the bounded read's, and a target that is not present. Verified: the whole server package passes (389 s); console TypeScript clean, 1485 tests, 1480 passed, 5 skipped. Independent review: approve. Not tested: a replayed revocation of a console-client grant (empty holder, action set); a holder whose user record is gone. The reviewer notes that a replayed record targeting an installation-action grant would show its holder here though the whole read's grants list hides such grants; no path creating such a record was found.

- **T2a, 2026-11-26.** `revocation_unconfirmed` on the rows of `GET /identity/access/subjects` only (the whole read's `subjects` and the retention owners keep their two fields): `modules/accesschange` `UnconfirmedRevocations(workspace, limit)`, and in `accesssubjects.go` the set of holders built once per call from the targets by identifier. The cursor's basis gains the change registry version and the access generation, since the flag depends on them. Tests: at three stages (a confirmed revocation only; a grant's revocation left unconfirmed on a stopped ledger; a delegation's) the flag of every listed person equals the browser's rule computed from the whole read, the lookup by identifier agrees with the list, a cursor from before the revocation is refused, and more unconfirmed revocations than the limit answers 503. The console's strict client requires the field. Verified: the whole server package passes (394 s); console TypeScript clean, 1485 tests, 1480 passed, 5 skipped. Independent review: approve. Known limit: when more unconfirmed revocations are kept than the scan limit the read answers 503, which also stops the read-only effect check's picker; such records arise only during a ledger outage within one process run. The access generation in the cursor's basis is not needed for the flag and makes a subjects cursor end on any permission change; kept for now. Not tested: an unconfirmed revocation that targets a console-client grant, which should flag no one.

# Risk

- The warning (item 4) and what would end (item 5) inform a person before they ask for access or revoke; a wrong answer misleads but the commands decide. The warning's rule moves from the browser to the server and is compared with the browser's rule in tests before the browser's is removed; what would end follows the server's rule instead and is compared with `ended_delegations` after an actual revocation.
- T3 to T5 change what the owner sees; each needs the ui-designer's rendered review and is not owner acceptance.

# Reviewed

Independent design review (role-based AI reviewer separate from the writer, by reading, nothing built), 2026-11-26: approve with conditions. One blocking finding, taken into this revision: the planned `descendants` field followed the browser's rule, which differs from what the server ends and ignores grant revocations. Also taken: the missed reads and the readers outside the view, the shared revocation row (so the new fields also appear on the whole read), the per-page shape and bound of the warning, the field name. The reviewer settled both unknowns of the first draft from the code: target facts now equal those at revocation, and excluding the caller from the choices is presentation, since the server already refuses a grant to oneself. Nothing here needs an owner decision before building; C5 does. Bounded re-review of the blocking finding, 2026-11-26: resolved with the grant-target backing stated explicitly in item 5. Not verified by the reviewer: realistic numbers of unconfirmed revocations and of delegation chains.
