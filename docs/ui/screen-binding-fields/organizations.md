---
type: Screen Binding Field Map
title: Field bindings of the organization and contract screens O01 to O03
description: For each field, column and state of the service-mode screens O01, O02 and O03, the read or command and JSON field it comes from, or the client-side derivation, with the parser line and the console line that renders it.
status: stable
tags: [HYB-295, screen-binding, O01, O02, O03]
sources:
  - resource: ../screen-binding-register.md
  - resource: ../../features/HB-257/plan.md
---

# Organization and contract field bindings

This is part of the [screen binding register](../screen-binding-register.md#organizations-and-contracts-o01o03), as `HB-257/plan.md:177-178` requires. It is based on main `82c9272`. Owner acceptance is pending for every screen here.

- **Short paths.** `admin.ts` and `contracts.ts` are under `apps/console/src/service/`. Page paths start at `apps/console/src/modules/organizations/pages/`.
- **Server rows.** The JSON shapes are defined in `apps/api/server/organizations.go:22-81`, `orgtargets.go:13`, `relplans.go:513-541` and `contracts.go`.
- **Copy.** Headings, labels and state words are fixed copy (`o.*`, `c.*`, `k.*`) unless the Source column names a JSON field.

## O01 Organizations and relationships

`GET /api/v1/organizations` is parsed at `admin.ts:978-1000`. The `parties` and `profiles` arrays are cast to `OrgParty[]` and `OrgProfile[]` without a check of each field (`admin.ts:988-989`).

| View | Field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| All | As-of date, revision pin | `on`, `relationship_revision`, `active_revision` | `admin.ts:983-986` | `organizations/Live.tsx` tree heading; newer-revision notice derived by `live-model.ts:231` |
| All | Filter band (legal entity, relationship kind, unattributed, search) | request `entity`, `kind`, `unattributed`, `q`; echoed in `filters` | `admin.ts:993-996` | `organizations/Live.tsx:964-966`, `filters.tsx` |
| All | Legal entity choices | `parties[]` where `kind = legal_entity` | `admin.ts:988` | `organizations/Live.tsx:962` |
| All | Provider filter: not connected | `provider_filter` (value `not_connected`); the line itself is fixed copy | `admin.ts:997` | `organizations/filters.tsx:81` |
| All | Denied | `permission_denied` failure | `admin.ts:776` | `organizations/Live.tsx:476-477` |
| All | Load failed, with Try again or Sign in again | other failure | `admin.ts:776` | `organizations/Live.tsx:568-575` |
| Tree | Roots, and the nodes under one opened node | `GET /api/v1/organizations/tree` (`parent` absent or set) → `nodes[]` (`id`, `name`, `kind`, `state`, `source`, `children`), `next`, `scan_stopped` (`apps/api/server/orgtree.go`) | `admin.ts` `tree()`, parsed strictly by `organization-read.ts` `organizationTreeOf` | `organizations/tree.tsx` (`Branch`, `BranchView`); read through `organizations/Live.tsx` `readTree` |
| Tree | Whether a node can be opened | `nodes[].children` (`some`, `none`, `unchecked`); a node already on its own path is not opened | same | `tree.tsx` `canOpen` |
| Tree | Legal entities on a cycle | the same read with `group=detached` | same | `tree.tsx` `TreeGroup`, hidden when empty |
| Tree | Without organization on the day | `GET /api/v1/organizations` with `kind=hierarchy`, `unattributed=true` → `members[]`; each row is shown with `children` unknown | `admin.ts` `organizations()` | `tree.tsx` `unplaced`, `TreeGroup` |
| Tree | Heading facts, registry sentences, selected node's detail | `GET /api/v1/organizations` with a page size of one (`relationship_revision`, `workspace`, `parties`, `profiles`) and with `member=` for the selected node | `admin.ts` `organizations()` | `organizations/Live.tsx` `prime`, `useMemberFacts` |
| Tree | Not bound | `registry_empty` of the tree read is parsed and not used; the empty sentence still reads the party list | | |
| Tree | States of each list: loading, failure with Try again, nothing under the node, further page, stopped search, failed further page, node already shown on its path | status of that list's own read; `next`, `scan_stopped` | `use-pages.ts` | `tree.tsx` `BranchView` |
| Tree | Empty: no parties, no revision or none of the kind, shown only when the roots, the cycle list and the no-organization list are all empty; otherwise "Nothing is placed at the top of the tree" | derived from `parties`, `revision` of the page-size-one read and from the three lists | `admin.ts` `organizations()`, `tree()` | `organizations/Live.tsx` tree view (`beside`) |
| Accounts | Account name and ID | `members[].member`, `name` (`subject=account`) | `admin.ts:991` | `organizations/Live.tsx:795` |
| Accounts | Organization on the day | `members[].targets` of `kind=hierarchy`, joined by `member=` | derived, `live-model.ts:101-108` | `organizations/Live.tsx:796` |
| Accounts | Payer on the day | `members[].targets` of `kind=payer` | derived, `live-model.ts:107` | `organizations/Live.tsx:797` |
| Accounts | Other relationship kind | `members[].targets` of the chosen kind | derived, `live-model.ts:107` | `organizations/Live.tsx:798` |
| Accounts | Entry from | `members[].history[].from` that covers the day | derived, `live-model.ts:91`, `:108` | `organizations/Live.tsx:799` |
| Accounts | Empty, filtered empty, scan stopped, Load more | `members` empty, `scan_stopped`, `next` | `admin.ts:991-999` | `organizations/Live.tsx:778`, `:807-808` |
| Groups | Group name and members on the day | `GET /api/v1/organizations/targets` → `targets[].id`, `name`, `members_on` (`orgtargets.go:13`) | `admin.ts:1016` | `organizations/Live.tsx:854-855` |
| Groups | Group members | `GET /api/v1/organizations?target=` → `members[]` | `admin.ts:991` | `organizations/Live.tsx:829` |
| Groups | Not in a view group or pool | `GET /api/v1/organizations` (`unattributed`) → `members[]` | `admin.ts:991` | `organizations/Live.tsx:867`, `:877` |
| Detail | ID, kind, parent on the day, state | `parties[]` (`id`, `kind`, `parent`, `state`) or `members[].targets` | `admin.ts:988`, `:991` | `organizations/Live.tsx:648-664` |
| Detail | Billing profiles: role, path, currency | `profiles[]` (`issuer`, `buyer`, `payer`, `recipient`, `path`, `currency`) | `admin.ts:989` | `organizations/Live.tsx:655`, `:671-674` |
| Detail | Selection not in this view | URL `sel` not among the loaded rows | derived | `organizations/Live.tsx:685-686` |
| History | Target, from, until, applies on the day | `members[].history[]` (`target`, `name`, `from`, `until`) | `admin.ts:991` | `organizations/Live.tsx:932-937`; applies derived at `live-model.ts:87` |
| History | Conflict | derived: more than one history entry applies on the day | `live-model.ts:87` | `organizations/Live.tsx:943` |
| Drafts | Draft list: name or group, state, members, plan | `GET /api/v1/organizations/view-group-drafts` → `drafts[]` (`name`, `group`, `group_label`, `new_group`, `state`, `member_count`, `plan.id`) | `admin.ts:387-398`, `:1044` | `organizations/drafts.tsx:156-159`, `:201-207` |
| Drafts | Open drafts against the limit; member limit | `open`, `open_limit`, `settings[view_group_draft_members]` | `admin.ts:1041-1043` | `organizations/drafts.tsx:59` |
| Drafts | Draft facts: name, group, members, state, created by and at, changed | `drafts[]` (`name`, `group_label`, `member_count`, `state`, `author_login`, `author`, `created`, `updated`) | `admin.ts:392-394` | `organizations/drafts.tsx:765-770` |
| Drafts | Discarded, activated, pruned, plan rule | `discarded`, `pruned`, `plan.rule` | `admin.ts:394-396` | `organizations/drafts.tsx:686-698` |
| Drafts | Draft members | `GET /api/v1/organizations/view-group-drafts/{id}/members` → `members[]` (`id`, `name`) | `admin.ts:1055` | `organizations/drafts.tsx:268`, `:401` |
| Drafts | Save, edit or discard result | `view_group.draft`, `view_group.edit`, `view_group.discard_draft` command outcomes | `admin.ts:735` (`outcomeOf`) | `organizations/drafts.tsx:452`, `:455`, `:501`, `:565` |

## O02 Relationship change plans

| Area | Field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Header | Active revision; open plans against the limit | `GET /api/v1/organizations/plans` → `active_revision`, `open_plans`, `open_limit` | `admin.ts:1021-1029` | `changes/Live.tsx:767-768`, `:924-926` |
| Plan list | Member or group, action, kind, target, from, stale | `plans[]` → `member_name`, `change` (`kind`, `member`, `action`, `target`, `from`), `target_name`, `stale`; `draft` for group plans | `admin.ts:329-371` (`planOf`) | `changes/Live.tsx:933-941` |
| Plan | Title, draft name, ID | `draft` (`name`, `member_count`, `new_group`), `id` | `planOf` | `changes/Live.tsx:135-137` |
| Plan | Stale base; truncated | `stale`, `base`, `truncated`; `active_revision` | `planOf` | `changes/Live.tsx:139-140` |
| Plan facts | Member, change, target, from, state, base, proposer, at, reason | `change.*`, `member_name`, `target_name`, `state`, `base`, `proposer_login`, `proposer`, `created`, `reason` | `planOf` | `changes/Live.tsx:142-151` |
| Plan | Violations | `violations[]` (`kind`, `member`, `rule`) | `planOf` | `changes/Live.tsx:155-164` |
| Plan | Changed members (before and after) | `members[]` (`member`, `name`, `differs`, `before`, `after`) | `admin.ts:351` | `changes/Live.tsx:174-201` |
| Plan | Kinds table: changed, entries before and after, earliest | `kinds[]` | `planOf` | `changes/Live.tsx:224-232` |
| Plan | Impact: money, publication, ERP | `impact.money`, `impact.publication`, `impact.erp` (`computed`, `not_computed`, else unknown) | `admin.ts:293-295` | `changes/Live.tsx:131`, `:241-251` |
| Plan | Held block; progress steps | derived from `impact`, `blocked`, `can_activate`, `state` | `changes/live-model.ts:70-99` | `changes/Live.tsx:86`, `:107-109`, `:243`, `:272` |
| Plan | Only a sealed plan can be decided | `state` | `planOf` | `changes/Live.tsx:266` |
| Decision | Activate or discard result | `relationship.activate` → `fields.revision`; `relationship.discard` | `admin.ts:735` | `changes/Live.tsx:707-712` |
| Proposal form | Member, kind, target choices | `GET /api/v1/organizations`, all kinds, read page by page in the browser | `admin.ts:991` | `changes/Live.tsx:40-60` (`readChoices`), `:870`, `:886` |
| Proposal form | Base revision from O01 | URL handoff, checked against `active_revision` | derived | `changes/Live.tsx:604`, `:817`, `:838` |
| Proposal form | From date | typed, minimum today | none | `changes/Live.tsx:823`, `:898` |
| Proposal result | New plan ID | `relationship.propose` or `view_group.propose` → `fields.plan` | `admin.ts:735` | `changes/Live.tsx:687` |
| Group plan | Draft members | view group drafts and members reads | `admin.ts:1044`, `:1055` | `changes/Live.tsx:381-383` |
| Closed plan | Link to O01 at the activated revision | `change.from`, `activated_revision` | `planOf` | `changes/Live.tsx:956` |
| All | Denied | `permission_denied` | `admin.ts:776` | `changes/Live.tsx:456`, `:471` |

## O03 Contract lifecycle

| Area | Field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Scope | Today, revision, contract count | `GET /api/v1/contracts` → `today`, `revision`, length of `contracts` | `contracts.ts:136` | `contracts/Live.tsx:166-168` |
| Scope | Money and amount not computed | `impact` (`money`, `proration`, `publication`, `erp`, `final_amount_decision`) | `contracts.ts:107` | `contracts/Live.tsx:141`, `:172` |
| List | Name, ID | `contracts[].name`, `id` | `contracts.ts:113` | `contracts/Live.tsx:300-301` |
| List | Buyer, payer | `contracts[].buyer`, `payer`; names from `GET /api/v1/organizations` → `parties[]` | `contracts.ts:113`, `admin.ts:988` | `contracts/Live.tsx:303-304` |
| List | Path, period, state on the day, final settlement, open gaps | `path`, `start`, `end`, `state`, `settlement`, `open_gaps` | `contracts.ts:113` | `contracts/Live.tsx:305-312` |
| List | Filters (state, settlement, path, buyer) | applied in the browser to the loaded `contracts` | derived | `contracts/Live.tsx:146`, `:233` |
| List | Loading, empty, filtered empty | derived | none | `contracts/Live.tsx:148-155` |
| All | Denied; session ended; load failed | failure codes | `admin.ts:776` | `contracts/Live.tsx:119`, `:129`, `:176` |
| Detail | Name, ID, state, period, settlement | `GET /api/v1/contracts/{id}` → `contract` (`name`, `id`, `state`, `end`, `start`, `settlement`) and `today` | `contracts.ts:148` | `contracts/LiveDetail.tsx:37-50` |
| Detail | Parties: buyer, payer, issuer, path | `contract.buyer`, `payer`, `issuer`, `path` | `contracts.ts:113` | `contracts/LiveDetail.tsx:79-82` |
| Detail | Billing setup: profile, currency, timezone, cycle | `contract.profile_name`, `profile`, `currency`, `timezone`, `cycle` | `contracts.ts:113` | `contracts/LiveDetail.tsx:88-91` |
| Detail | Terms revisions: revision, customer, seller, path | `terms[]` | `contracts.ts:148` | `contracts/LiveDetail.tsx:102-107` |
| Detail | Events: from, kind, rate revision, withdraw or blocked reason | `events[]` (`from`, `kind`, `revision`, `withdraw_blocked`, `cancel_required`) | `contracts.ts:148` | `contracts/LiveDetail.tsx:138`, `:159-168` |
| Detail | Segments: from, until, days, state, revision | `segments[]` | `contracts.ts:148` | `contracts/LiveDetail.tsx:217-222` |
| Detail | Components: kind, provider payment responsible, seller billable | `terms[].components[]` | `contracts.ts:148` | `contracts/LiveDetail.tsx:248-250` |
| Detail | Documents, payment terms, recognition, rates: not connected | fixed copy for those views | none | `contracts/LiveDetail.tsx:27`, `:263-271` |
| Termination | End day, last event, open gaps, reopenings, sequence, final amount not computed | `contract.end`, `events`, `contract.open_gaps`, `reopenings`, `contract.sequence`; the final amount line is fixed copy | `contracts.ts:148` | `contracts/LiveDetail.tsx:283-291` |
| Termination | Evidence: kind, reference or gap, covered until, owner, follow-up, write | `evidence[]` | `contracts.ts:148` | `contracts/LiveDetail.tsx:302-311` |
| Termination | Write log: at, actor, kind, added, withdrawn, as of, reopen, settlement, write, reason | `records[]` | `contracts.ts:148` | `contracts/LiveDetail.tsx:329-341` |
| Actions | Offered or blocked, with earliest day | `actions[<command>]` (`allowed`, `blocked`, `earliest`) | `contracts.ts:149`, `:203` | `contracts/LiveActions.tsx:68`, `:130`, `:219-231` |
| Actions | Result and new settlement | command outcome `state` | `admin.ts:735` | `contracts/LiveActions.tsx:174` |
