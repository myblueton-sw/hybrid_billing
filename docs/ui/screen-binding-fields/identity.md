---
type: Screen Binding Field Map
title: Field bindings of the identity administration screens M01 to M05
description: For each column, field and state of the service-mode identity screens M01, M02, M03, M04 and M05, the module read or command and JSON field it comes from, or the client-side derivation, with the parser line and the console line that renders it.
status: stable
tags: [HYB-295, screen-binding, M01, M02, M03, M04, M05]
sources:
  - resource: ../screen-binding-register.md
  - resource: ../../features/HB-257/plan.md
  - resource: ../../features/HB-264/verification.md
  - resource: ../../features/HB-265/verification.md
  - resource: ../../features/HB-267/verification.md
  - resource: ../../features/HB-269/verification.md
  - resource: ../../features/HB-271/verification.md
  - resource: ../../features/HYB-297/verification.md
  - resource: ../../features/HYB-298/verification.md
  - resource: ../../features/HYB-299/verification.md
---

# Identity administration field bindings

This is part of the [screen binding register](../screen-binding-register.md#identity-administration-m01m05), as `HB-257/plan.md:177-178` requires: it records which screen fields come from which module read or command. The register entries list the reads and commands per screen; this file adds the field level. It is based on main `82c9272`. Owner acceptance is pending for every screen here.

- **Short paths.** `admin.ts` is `apps/console/src/service/admin.ts`; `identity-bulk.ts` is `apps/console/src/service/identity-bulk.ts`. Page paths start at `apps/console/src/modules/identity/pages/`, so `users/Live.tsx` is `apps/console/src/modules/identity/pages/users/Live.tsx`. Go files (`users.go`, `policy.go`, `access.go` and so on) are under `apps/api/server/`; routes are registered in `boundary.go:21-59`.
- **Commands.** Every command goes through `POST /api/v1/commands/{command}` (`boundary.go:24`), sent by `admin.ts:871`. The answer envelope is `resultBody` (`errors.go:41-52`); `outcomeOf` (`admin.ts:735-744`) maps `status`, `business_state`, `job_id`, `result_revision` and `fields`. "Result `fields.x`" below means `Outcome.fields.x`.
- **Failures.** A refused read or command becomes an `AdminError` through `failureOf` (`admin.ts:776-783`, code table `admin.ts:750-772`). The message comes from the screen's describe function named in each row.
- **Copy.** Headings, labels and messages are fixed copy (`w.*`, `b.*`, `p.*`, `ty.*`, `a.*`, `q.*`, `v.*`, `s.*`, `m.*` from the page's `*-copy.ts`). They are not service data unless the Source column names a JSON field.
- **Typed values.** Values the person types are request fields of the command named in the row, not read fields.
- **Derived.** "derived" means the value is computed in the console; the cited line is the computation.

## M01 Users and access

Read: `GET /api/v1/identity/users?limit=100` (`boundary.go:25`), client `admin.ts:854-867`, called at `users/Live.tsx:509` (first page) and `:529` (more). Server shape: `usersPage` (`users.go:45-53`) with `userRow` (`users.go:30-43`), `memberRow` (`users.go:14-23`), `revocationRef` (`users.go:25-28`), embedded `clockRows` (`clockrows.go:22-30`) and `settingRow` (`settings.go:145-153`). Row parser `userRowOf` (`admin.ts:96-100`) with `membershipOf` (`admin.ts:60-75`); `clockOf` (`admin.ts:77-90`); `settingsOf` (`admin.ts:92-94`).

### List frame and states

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| All list views | Loading | derived: `status === "loading"` before the first page | none | `users/Live.tsx:655` |
| All list views | Load failed, with Try again | any failed users read; body from `failureText` (`users/live-model.ts:52-81`) | `admin.ts:776-783` | `users/Live.tsx:656-665` |
| All list views | Session ended, with Sign in again | users read refused `unauthenticated` (401, `csrf_refused`) | `admin.ts:757, 780` | `users/Live.tsx:660-661` |
| All list views | Denied (no administration right) | users read refused `permission_denied` | `admin.ts:755` | `users/Live.tsx:560-568` |
| Users and membership views | Empty list | derived: zero rows after `rowsFor` | none | `users/Live.tsx:668` (`w.empty`) |
| Invitations view | Empty list | derived: no row with `invitation_expires` or `link_purpose = "set"` (`users/Live.tsx:353-355`) | none | `users/Live.tsx:668` (`w.noInvitations`) |
| Toolbar | Title (Users, Invitations, Membership) | derived from the view | none | `users/Live.tsx:571, 638` |
| Toolbar | Count | derived: number of rows shown | none | `users/Live.tsx:629, 639` |
| Toolbar | Loaded at | derived: browser time when the page arrived (`users/Live.tsx:513`) | none | `users/Live.tsx:640` |
| Toolbar | Reload | repeats the users read | `admin.ts:854` | `users/Live.tsx:644` |
| Toolbar | Create user button | shown outside the membership view and while not stranded (`recovery_state.stranded`) | `admin.ts:88` | `users/Live.tsx:645-647` |
| List | Show more | `next` (`users.go:47`) | `admin.ts:860` | `users/Live.tsx:674` |
| Notices | Recovery stranded | `recovery_state.stranded` (`clockrows.go:18`) | `admin.ts:88` | `users/Live.tsx:456` |
| Notices | Host clock ahead beyond tolerance | `wall_lead_beyond_tolerance.wall_time`, `.server_time` (`clockrows.go:13-14, 28`) | `admin.ts:87` | `users/Live.tsx:457-463` |
| Notices | Decision ledger down | `ledger_state` in `isolated`, `stopped`, `full` (`users.go:51`) | `admin.ts:864` | `users/Live.tsx:450, 464` |
| Notices | Clock base not recorded | `clock_base.choice` not `default` and `clock_base.recorded` false (`clockrows.go:6, 9`) | `admin.ts:86` | `users/Live.tsx:451, 465` |
| Notices | Clock differs | `clock_behind`, `clock_ahead` (`clockrows.go:25-26`) | `admin.ts:84-85` | `users/Live.tsx:466` |
| Panel | Stranded instead of panel | `recovery_state.stranded` | `admin.ts:88` | `users/Live.tsx:682` |

### Users view table

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Users table | Column headings | fixed copy | none | `users/Live.tsx:381` |
| Users table | User (opens detail) | `users[].login`, else `users[].user_id` (`users.go:31-32`) | `admin.ts:96-100` | `users/Live.tsx:397` |
| Users table | Contact email under the user | `users[].email` (`users.go:33`), shown when it differs from the login | `admin.ts:96-100` | `users/Live.tsx:401` |
| Users table | Open tag on the selected row | derived: selected row | none | `users/Live.tsx:400` |
| Users table | Account status | `users[].state` (`users.go:34`) via `accountName` (`users/live-model.ts:18-20`) | `admin.ts:96-100` | `users/Live.tsx:423` |
| Users table | Suspended by bulk revocation N | `users[].revocation.bulk`, `.state` (`users.go:26-27`) and `membership.suspended_for` (`users.go:21`) | `admin.ts:72, 98-99` | `users/Live.tsx:363-371`, used at `:423, :427` |
| Users table | Decision unconfirmed on that effect | `users[].revocation.state = "decision_unconfirmed"` | `admin.ts:99` | `users/Live.tsx:368` |
| Users table | Workspace role | `users[].membership.role` (`users.go:15`); no membership shows "No membership" (`users/live-model.ts:14-16`) | `admin.ts:66` | `users/Live.tsx:425` |
| Users table | Membership suspended / role period ended | `membership.state`, `membership.suspended_for` (`users.go:16, 21`) | `admin.ts:67, 72` | `users/Live.tsx:426` |
| Users table | Installation administrator | `users[].installation_administrator` (`users.go:41`) | `admin.ts:96-100` | `users/Live.tsx:428` |
| Users table | Role period | `membership.role_period`, `role_until`, `suspended_for`, `ledger` (`users.go:19-22`) and `server_time` for days left (`clockrows.go:23`) via `PeriodCell` | `admin.ts:69-73, 82` | `users/RolePeriod.tsx:22-47`, used at `users/Live.tsx:430` |
| Users table | Role period ledger unconfirmed | `membership.ledger = "unconfirmed"` | `admin.ts:73` | `users/RolePeriod.tsx:25` |
| Users table | Sign-in setup | `link_purpose`, `link_expires`, `invitation_expires`, `password_set` (`users.go:37-40`) via `setup` (`users/live-model.ts:22-27`) | `admin.ts:96-100` | `users/Live.tsx:431`, folded copy `:402` |
| Users table | Role and period (medium width) | same fields as Workspace role, Role period, Membership suspended, Installation administrator | as above | `users/Live.tsx:432-437` |
| Users table | Selection checkbox | shown when `can_bulk_revoke` (`users.go:52`) and not stranded | `admin.ts:865` | `users/Live.tsx:572, 394`; cell `users/Selection.tsx:18-28` |
| Users table | "You" instead of a checkbox | derived: `user_id` equals the session user | none | `users/Selection.tsx:11-16` |

### Membership and invitations view tables

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Membership table | Column headings | fixed copy | none | `users/Live.tsx:380` |
| Membership table | User, contact email | `login`/`user_id`, `email` | `admin.ts:96-100` | `users/Live.tsx:397, 401` |
| Membership table | Workspace role | `membership.role` | `admin.ts:66` | `users/Live.tsx:412` |
| Membership table | Role period | as Users table Role period | `admin.ts:69-73` | `users/Live.tsx:413` |
| Membership table | Membership status, with bulk effect | `membership` null, `membership.state`; `revocation` | `admin.ts:61, 67, 98-99` | `users/Live.tsx:414` |
| Membership table | Installation administrator (yes or dash) | `installation_administrator` | `admin.ts:96-100` | `users/Live.tsx:415` |
| Membership table | Role and period (medium width) | as above | as above | `users/Live.tsx:416-419` |
| Invitations table | Column headings | fixed copy | none | `users/Live.tsx:378` |
| Invitations table | Rows | derived filter: `invitation_expires` set or `link_purpose = "set"` | none | `users/Live.tsx:353-355` |
| Invitations table | Role | `membership.role` | `admin.ts:66` | `users/Live.tsx:406` |
| Invitations table | Invitation expires | `invitation_expires` (`users.go:38`) | `admin.ts:96-100` | `users/Live.tsx:407` |
| Invitations table | Set link issued until / not issued | `link_purpose`, `link_expires` (`users.go:39-40`) | `admin.ts:96-100` | `users/Live.tsx:408` |
| Invitations view | Selected elsewhere note | derived: selection size while selection is on | none | `users/Live.tsx:654` |

### Selection bar

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Selection bar | Shown | `can_bulk_revoke` and not stranded, outside the invitations view | `admin.ts:865` | `users/Live.tsx:650-653` |
| Selection bar | Select loaded rows (header checkbox) | derived from loaded rows minus self (`users/Live.tsx:573`) | none | `users/Selection.tsx:31-42` |
| Selection bar | Selected count and limit | derived count; limit is setting `bulk_revocation_max_targets` from `settings[]` (default 500) | `admin.ts:92-94` | `users/Live.tsx:545`; `users/Selection.tsx:53-55, 63` |
| Selection bar | Prepare, Clear | derived: count above zero | none | `users/Selection.tsx:56-61` |
| Selection bar | Selected but not loaded | derived; names come from rows loaded earlier (`users/Live.tsx:512, 532`) | none | `users/Selection.tsx:64-76` |

### User detail and actions

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Detail | Heading | `login`, else `user_id` | `admin.ts:96-100` | `users/Live.tsx:329` |
| Detail | Done notice from the last action | derived: action copy (`actionDone`) or role-period result lines | none | `users/Live.tsx:328` |
| Detail | Contact email | `email` | `admin.ts:96-100` | `users/Live.tsx:331` |
| Detail | Account status | `state` | `admin.ts:96-100` | `users/Live.tsx:332` |
| Detail | Workspace role | `membership.role` | `admin.ts:66` | `users/Live.tsx:333` |
| Detail | Role period from | `membership.role_from` (`users.go:18`) | `admin.ts:69` | `users/RolePeriod.tsx:57` |
| Detail | Role period | as Users table Role period | `admin.ts:69-73` | `users/RolePeriod.tsx:58` |
| Detail | Latest revocation (opens the job) | `revocation.bulk` | `admin.ts:99` | `users/RolePeriod.tsx:62-69` |
| Detail | Membership status | `membership`, `membership.state` | `admin.ts:61, 67` | `users/Live.tsx:335` |
| Detail | Sign-in setup | as Users table Sign-in setup | `admin.ts:96-100` | `users/Live.tsx:336` |
| Detail | Installation administrator (only when true) | `installation_administrator` | `admin.ts:96-100` | `users/Live.tsx:337` |
| Detail | User ID | `user_id` | `admin.ts:96-100` | `users/Live.tsx:338` |
| Detail | Blocked: self or departed | derived by `offers` from session user and `state` (`users/live-model.ts:33-34`) | none | `users/Live.tsx:340-341` |
| Detail | Account, membership and sign-in action buttons | derived by `offers` from `state`, `membership`, `role_period`, `suspended_for`, `password_set`, `login`, `link_purpose`, `invitation_expires` (`users/live-model.ts:31-50`) | none | `users/Live.tsx:314-325, 344-346` |
| Detail | Notes: administrator reset, suspended reset, promotion blocked | derived (`users/live-model.ts:44-48`) | none | `users/Live.tsx:323, 345-346` |
| Confirm | Heading, person | action copy; `login` or `user_id` | none | `users/Live.tsx:156-157` |
| Confirm | Impact sentence | derived from action, current `membership.role` and chosen role | none | `users/Live.tsx:160` |
| Confirm | Role choice (invite, change role) | typed `role`; administrator offered unless `link_purpose` is set | none | `users/Live.tsx:120, 163` |
| Confirm | Role ends: server now, latest end, your own end | `server_time`; setting `role_period_max_days` (default 366); own `membership.role_until` of the session user (`users/Live.tsx:544`) | `admin.ts:82, 92-94` | `users/RolePeriod.tsx:74-112`, used at `users/Live.tsx:164` |
| Confirm | Reason with byte counter | typed `rationale` | none | `users/Live.tsx:53-73, 167` |
| Confirm | Command | `identity.` + action (`users/Live.tsx:51`), body `role`, end fields | `admin.ts:871` | `users/Live.tsx:142-143` |
| Confirm | One-time link after issue set/reset link | result `fields.link_ref` (`admin.go:219`) | `admin.ts:742` | `users/Live.tsx:144, 584-585` |
| Confirm | Failure box | refused command; `failureText` (`users/live-model.ts:52-81`); settled failures reload (`users/Live.tsx:108-110`) | `admin.ts:776-783` | `users/Live.tsx:159` |
| Create | Sign-in email, contact email, role, role ends, reason | typed; sent as `login`, `email`, `role`, end fields to `identity.create_user` | none | `users/Live.tsx:232-236`, command `:210` |
| Create | Login already used | `prerequisite_failed` → `refused` | `admin.ts:752` | `users/Live.tsx:213` |
| Create | Field refused by server | `invalid_request` with `fields` (`role_until`, `email`, `login`) | `admin.ts:778` | `users/Live.tsx:214-216` |
| Create | New user, then set-link pickup | result `fields.user_id` and `fields.link_ref` (`admin.go:219, 259`) | `admin.ts:742` | `users/Live.tsx:211, 610` |
| Link pickup | Loading | derived while `GET /api/v1/identity/links/{ref}` runs (`boundary.go:30`) | `admin.ts:1092-1097` | `users/Live.tsx:255, 280` |
| Link pickup | One-time link | `url` (`users.go:225`); the client rejects a URL that is not `/activate#<43 chars>` | `admin.ts:1094-1095` | `users/Live.tsx:285` |
| Link pickup | Copy result | derived: clipboard write outcome | none | `users/Live.tsx:288-290` |
| Link pickup | Link expires | `link_expires` of the row whose `login` matches | `admin.ts:96-100` | `users/Live.tsx:613, 291` |
| Link pickup | Unavailable, show link again | pickup read refused as `unavailable` | `admin.ts:780, 829` | `users/Live.tsx:257, 296-301` |
| Link pickup | Link gone | any other pickup failure, or no `link_ref` | none | `users/Live.tsx:257, 264, 302` |
| Role period form | Clock differs | `clock_behind`, `clock_ahead` | `admin.ts:84-85` | `users/RolePeriod.tsx:193` |
| Role period form | Current period | as Users table Role period | `admin.ts:69-73` | `users/RolePeriod.tsx:196` |
| Role period form | Shorter or longer | derived from `role_until` and the typed end (`users/period-model.ts:61`) | none | `users/RolePeriod.tsx:161, 198-200` |
| Role period form | Settings disclosure: name, value and unit, default, range, source | `settings[].name`, `value`, `unit`, `default`, `min`, `max`, `source` (`settings.go:146-152`) | `admin.ts:92-94` | `users/RolePeriod.tsx:120-143`, used at `:201` |
| Role period form | Command | `identity.set_role_period` with `end` and end fields | `admin.ts:871` | `users/RolePeriod.tsx:170` |
| Role period form | Saved / not durable / unknown / ledger unconfirmed | result `fields.durable` (`increases.go:88`) and `fields.ledger` (`periods.go:130`, `increases.go:88`) | `admin.ts:742` | `users/RolePeriod.tsx:171-172`; notice `users/Live.tsx:618` |
| Role period form | Unchanged or ended refusal, blocked | `invalid_stage` → `stage`; `prerequisite_failed` → `refused` | `admin.ts:752-753` | `users/RolePeriod.tsx:184-188, 195` |

### Select everyone matching a filter (HYB-323)

Read: `GET /api/v1/identity/users/count` (`boundary.go:26`, `userscount.go`) through `bulkClient.count` (`identity-bulk.ts:174`, path built at `:133`, parsed at `:128`). Command: `bulk_revocation.add_matching` (`bulkstage.go:150`), sent at `users/Matching.tsx:176` with the users query's filter values (`identity-bulk.ts` `matchValues`).

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Selection bar | Select everyone who matches / everyone in the list | derived: `can_bulk_revoke`, Users or Membership view, no fixed list; label from the applied filters | `admin.ts:865` | `users/Matching.tsx:142`; `users/Live.tsx:813` |
| Match region | Counting, exact, at least, too many, none match | count read `count`, `complete`, `over_limit`, `limit`, `scan_budget`, `server_time` | `identity-bulk.ts:128` | `users/Matching.tsx:200-246` (`data-state` on `#m01-match`) |
| Match region | Replace confirmation | derived: selected people above zero | none | `users/Matching.tsx` (`#m01-replace`) |
| Match region | Refusal cause (open draft, none, too many, pending limit, combined) | command refused, then `bulk-revocations?limit=1` `open_draft`, `pending` and a second count read (`match-model.ts` `refusalCause`) | `identity-bulk.ts:120-126` | `users/Matching.tsx:189` |
| Fixed bar | Fixed line, when counted | command result `fields.matched`, `fixed_at`; the last complete count | `match-model.ts` `fixedOfFields` | `users/Matching.tsx:103`; `users/Live.tsx:808` |
| Fixed bar | Fixed filters chips, base revision, draft number | result `fields.filter.<key>`, `fixed_revision`, `draft`; after a reload the own open draft row `filter`, `fixed_revision`, `id` | `match-model.ts` `fixedOfJob`; `identity-bulk.ts:90-104` | `users/Matching.tsx:103-138`; recovery at `users/Live.tsx:596` |
| Fixed bar | Discard list | `bulk_revocation.discard` on the draft | none | `users/Live.tsx` `discardFixed` |
| Fixed bar | Remove people from the list | `bulk_revocation.remove_targets` on the draft with `users` (the people selected in removal mode, in batches); result `fields.removed`, `already`, `ignored`, `chosen` | `match-model.ts` `removalOf` | `users/Live.tsx` `removeSelected`; `users/Matching.tsx` `FixedBar` (`removal`) |
| Fixed bar | Removed by you | the last result's `fields.chosen`; after a reload the own open draft row `chosen` | `identity-bulk.ts` `jobOf`; `match-model.ts` `fixedOfJob` | `users/Matching.tsx` `FixedBar` |
| Users table | No checkbox column while fixed | derived: fixed list open | none | `users/Live.tsx` (`sel.on`) |
| Prepare | Everyone matching filters summary | fixed list | none | `users/BulkPrepare.tsx` (filter mode) |
| Prepare | Anchor dead-end, list closed | `bulk_revocation.prepare` refused `prerequisite_failed` / `invalid_stage` on a filter list | `admin.ts:783-785` | `users/BulkPrepare.tsx:216, 329` |
| Preview | Fixed list facts, arithmetic, no longer match | `job.selection`, `filter`, `fixed_at`, `matched`, `fixed_revision`, `counts`; `exclusions.no_longer_matches` | `identity-bulk.ts:90-104, 104-118` | `users/BulkPrepare.tsx:71, 94`; `users/Matching.tsx:63` |
| Jobs view | Selection basis line, Fixed {at} | `jobs[].selection`, `filter`, `search_masked`, `fixed_at` | `identity-bulk.ts:90-104` | `users/RevocationJobs.tsx:187, 202`; `users/Matching.tsx:53` |
| Job detail | Basis, fixed chips, Search applied | `job.selection`, `filter`, `search_masked`, `matched`, `fixed_at`, `fixed_revision` | `identity-bulk.ts:90-104` | `users/RevocationJobs.tsx:117-140`; `users/Matching.tsx:81` |
| Settings | Max matching, retained people, the two scan budgets, hints | `settings[]` rows `bulk_revocation_max_matching`, `bulk_revocation_retained_people_max`, `user_filter_scan`, `user_count_scan` | `admin.ts:92-94` | `users/RolePeriod.tsx:132` |

### Bulk revocation: prepare and preview

Reads: `GET /api/v1/identity/bulk-revocations` (`boundary.go:26`), `/{id}` (`:27`), `/{id}/items` (`:28`), `/{id}/targets` (`:29`), all through `admin.ts:868` (`read`) and `bulkClient` (`identity-bulk.ts:138-169`). Server shapes in `bulkread.go:12-162`.

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Prepare | Effect choice, reason | typed `effect` (`suspend_membership` or `suspend_user`), `rationale` | none | `users/BulkPrepare.tsx:314-324` |
| Prepare | Selected people list | derived: selected IDs; names from loaded user rows | none | `users/BulkPrepare.tsx:333-340` |
| Prepare | Sending progress, waiting after rate limit, resent after conflict | derived from `bulk_revocation.add_targets` calls in batches (`identity-bulk.ts:214-250`); draft ID from result `fields.draft` (`bulkstage.go:86`) | `identity-bulk.ts:229` | `users/BulkPrepare.tsx:325-330` |
| Prepare | Not prepared: open draft or pending limit | refused staging; cause read from list `open_draft` (`bulkread.go:60`) | `identity-bulk.ts:99` | `users/BulkPrepare.tsx:217-218, 270-277` |
| Prepare | Not prepared: other failure, count staged | derived staged count; `failureText` body | none | `users/BulkPrepare.tsx:278-283` |
| Prepare | Resume, Discard draft | derived: draft ID known; discard sends `bulk_revocation.discard` | `admin.ts:871` | `users/BulkPrepare.tsx:285-288, 244` |
| Prepare | Refused: administrators in selection, with their periods | `bulk_revocation.prepare` refused; administrators derived from loaded rows `membership.role`; periods via `PeriodCell` | `admin.ts:752` | `users/BulkPrepare.tsx:229, 291-311`; `users/Live.tsx:593` |
| Prepare | Command | `bulk_revocation.prepare` with `effect`; result `fields.bulk` (`bulk.go:40`) | `admin.ts:742` | `users/BulkPrepare.tsx:224-226` |
| Preview | Heading with list number | `job.id` via `number` (`users/bulk-model.ts:105-108`) | `identity-bulk.ts:91` | `users/BulkPrepare.tsx:345` |
| Preview | Expired banner, Prepare again | derived `expired` from `job.state`, `job.expires_at` and detail `server_time` (`users/bulk-model.ts:99-101`) | `identity-bulk.ts:91, 109` | `users/BulkPrepare.tsx:347, 350-351` |
| Preview | Prepared at, valid until | `job.prepared_at`, `job.expires_at` (`bulkread.go:41-42`) | `identity-bulk.ts:91` | `users/BulkPrepare.tsx:55-56` |
| Preview | Scope sentence | `job.effect`, `job.counts.targets` | `identity-bulk.ts:88, 92` | `users/BulkPrepare.tsx:58` |
| Preview | To revoke, excluded, duplicates | `job.counts.targets`, `excluded`, `duplicates` (`bulkread.go:13-15`) | `identity-bulk.ts:77-81` | `users/BulkPrepare.tsx:60-62` |
| Preview | Impact computed at | `impact.computed_at` (`bulkread.go:76`) | `identity-bulk.ts:111` | `users/BulkPrepare.tsx:67` |
| Preview | People and roles | `impact.people`, `administrators`, `members` (`bulkread.go:77-79`) | `identity-bulk.ts:111` | `users/BulkPrepare.tsx:69` |
| Preview | Rights | `impact.installation_administrators`, `trust_right_holders`, `bulk_right_holders` (`bulkread.go:80-82`) | `identity-bulk.ts:112` | `users/BulkPrepare.tsx:71` |
| Preview | Sessions, pickups | `impact.sessions`, `pickups` (`bulkread.go:83-84`) | `identity-bulk.ts:113` | `users/BulkPrepare.tsx:73` |
| Preview | API keys: not applicable | `impact.api_keys`: explicit `not_applicable` uses `b.noKeys`; absent/unrecognized is unknown | `identity-bulk.ts` `detailOf` | `users/BulkPrepare.tsx:73` |
| Preview | Pending work per store, rechecked or not | `impact.pending[].store`, `count`, `rechecked` (`bulkread.go:70-72, 86`) | `identity-bulk.ts:114` | `users/BulkPrepare.tsx:75-76` |
| Preview | Pending jobs: not applicable | `impact.pending_jobs`: explicit `not_applicable` uses `b.noJobs`; absent/unrecognized is unknown | `identity-bulk.ts` `detailOf` | `users/BulkPrepare.tsx:77` |
| Preview | Service users kept | `impact.service_users` (`bulkread.go:88`) | `identity-bulk.ts:113` | `users/BulkPrepare.tsx:79` |
| Preview | Administrators left afterwards | `impact.anchors_after` (`bulkread.go:89`) | `identity-bulk.ts:113` | `users/BulkPrepare.tsx:81` |
| Preview | Other workspaces | `impact.other_workspaces` (`bulkread.go:90`) | `identity-bulk.ts:113` | `users/BulkPrepare.tsx:82` |
| Preview | Excluded by reason | `exclusions` map, keys `self`, `not_found`, `no_membership`, `already_revoked`, and for a filter list `no_longer_matches` and `chosen` (`bulkread.go`, `readBulk`); the count of `chosen` is also shown under the excluded count in the preview and in the confirmation as people who keep their access | `identity-bulk.ts` `detailOf` | `users/BulkPrepare.tsx` `PreviewBody`, `ConfirmDialog` |
| Preview | Excluded people | `/targets?kind=excluded` → `exclusions[].user_id`, `login`, `reason`, `total`, `next` (`bulkread.go:144-155`) | `identity-bulk.ts:155-162` | `users/BulkPrepare.tsx:31-46` |
| Preview | People to revoke | `/targets` → `targets[].login`, `user_id`, `role`, `installation_administrator`, `total`, `next` (`bulkread.go:134-142, 157-162`) | `identity-bulk.ts:130-134, 163-167` | `users/BulkPrepare.tsx:90-100` |
| Preview | Preparer only may see people | derived: targets not loaded (job opened by someone else) | none | `users/BulkPrepare.tsx:101`; `users/RevocationJobs.tsx:234` |
| Preview | Not changed by this list | `outside_scope[]` codes (`bulkread.go:99`) | `identity-bulk.ts:109` | `users/BulkPrepare.tsx:102-105` |
| Preview | Continue, Discard | derived; Continue re-reads the detail and stops if expired | `identity-bulk.ts:146-148` | `users/BulkPrepare.tsx:252-261, 349-354` |
| Confirm dialog | List number and ID, effect, to revoke, excluded, prepared by, prepared at, valid until, digest (first 12) | `job.id`, `effect`, `counts.targets`, `counts.excluded`, `preparer`, `prepared_at`, `expires_at`, `digest` (`bulkread.go:35-47`) | `identity-bulk.ts:83-96` | `users/BulkPrepare.tsx:158-167` |
| Confirm dialog | Scope, outside scope, consequence, reversal | `job.effect`, `counts.targets`, `outside_scope[]`; consequence and reversal are copy | `identity-bulk.ts:88, 109` | `users/BulkPrepare.tsx:168-171` |
| Confirm dialog | Reason, confirmation checkbox | typed `rationale`; local checkbox | none | `users/BulkPrepare.tsx:172-176` |
| Confirm dialog | Command | `bulk_revocation.execute` with `digest`; job from result `job_id` (`bulk.go:105`) | `admin.ts:740` | `users/BulkPrepare.tsx:139-141` |
| Confirm dialog | Final failure, Reload and close | `stage`, `invalid`, `refused`, `conflict`, `stale`, `denied` | `admin.ts:750-772` | `users/BulkPrepare.tsx:149-151, 157, 178-179` |

### Bulk revocation: jobs view

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Jobs view | Notices | the users read clock and ledger fields (same as list notices) | `admin.ts:77-90` | `users/Live.tsx:555` |
| Jobs view | Loading | derived: no list yet and no error | none | `users/RevocationJobs.tsx:277` |
| Jobs view | Load failed, Try again | any failed list or detail read | `admin.ts:776-783` | `users/RevocationJobs.tsx:276` |
| Jobs view | Session ended | `unauthenticated` | `admin.ts:757` | `users/RevocationJobs.tsx:253-255` |
| Jobs view | Denied | `permission_denied` | `admin.ts:755` | `users/RevocationJobs.tsx:256-258` |
| Jobs view | Loaded at | derived: browser time of the list read | none | `users/RevocationJobs.tsx:217, 271` |
| Jobs view | Jobs before this start not kept | `started_at` (`bulkread.go:58`) | `identity-bulk.ts:99` | `users/RevocationJobs.tsx:275` |
| Jobs view | No jobs yet | derived: empty `jobs` | `identity-bulk.ts:100` | `users/RevocationJobs.tsx:288` |
| Jobs view | Not executed, executed, ended groups and empty lines | derived by `split` on `jobs[].state` (`users/bulk-model.ts:91-97`) | `identity-bulk.ts:87` | `users/RevocationJobs.tsx:259, 290-300` |
| Jobs table | List number (opens job) | `jobs[].id` via `number` | `identity-bulk.ts:91` | `users/RevocationJobs.tsx:187` |
| Jobs table | Effect | `jobs[].effect` (`bulkread.go:38`) | `identity-bulk.ts:88` | `users/RevocationJobs.tsx:190` |
| Jobs table | From an earlier run | `jobs[].origin = "ledger_replay"` (`bulkread.go:37`) | `identity-bulk.ts:91` | `users/RevocationJobs.tsx:191` |
| Jobs table | State | derived `jobWords` from `state`, `job_state`, `job_reason`, `counts.failed` (`users/bulk-model.ts:53-64`) | `identity-bulk.ts:87, 89, 92` | `users/RevocationJobs.tsx:192, 195` |
| Jobs table | People | draft: `staged` (`bulkread.go:48`); executed: count equation; otherwise `counts.targets`; plus `counts.excluded` | `identity-bulk.ts:77-81, 94` | `users/RevocationJobs.tsx:173-175, 193, 196` |
| Jobs table | When | `expires_at` for prepared, else `executed_at` (`bulkread.go:42, 44`) | `identity-bulk.ts:91-92` | `users/RevocationJobs.tsx:176-177, 197` |
| Jobs view | More jobs | `next` (`bulkread.go:62`) | `identity-bulk.ts:100` | `users/RevocationJobs.tsx:301` |
| Jobs view | Settings disclosure | `settings[]` of the list (`bulkread.go:64`) | `identity-bulk.ts:100` | `users/RevocationJobs.tsx:308` |
| Job detail | Heading, effect | `job.id`, `job.effect` | `identity-bulk.ts:88, 91` | `users/RevocationJobs.tsx:108-109` |
| Job detail (prepared) | Preview body as above; Continue and Discard for the preparer only | `job.preparer` against the session user | `identity-bulk.ts:91` | `users/RevocationJobs.tsx:97, 112-121` |
| Job detail (draft) | State line, Discard draft for the preparer | `job.state`, `job.preparer` | `identity-bulk.ts:87, 91` | `users/RevocationJobs.tsx:123-127` |
| Job detail (discarded, expired) | State line | `job.state` | `identity-bulk.ts:87` | `users/RevocationJobs.tsx:128-129` |
| Job detail (executed) | Partial, unconfirmed, not started, waiting, finished, held | derived `jobWords`; held from `counts.held` | `identity-bulk.ts:79, 89` | `users/RevocationJobs.tsx:32-43, 132` |
| Job detail (executed) | Counts and equation | `counts.targets`, `succeeded`, `held`, `failed`, `unprocessed`, `excluded` (`bulkread.go:13-19`) | `identity-bulk.ts:77-81` | `users/RevocationJobs.tsx:19-30, 133` |
| Job detail (executed) | Prepared by and at, executed by and at, blocked at | `preparer`, `prepared_at`, `executor`, `executed_at`, `deny_at` (`bulkread.go:40-45`) | `identity-bulk.ts:91-92` | `users/RevocationJobs.tsx:135-139` |
| Job detail (executed) | Ledger records or not confirmed | `ledger.first_sequence`, `last_sequence`, `reason` (`bulkread.go:23-26`) | `identity-bulk.ts:93` | `users/RevocationJobs.tsx:140-141` |
| Job detail (executed) | Cleanup done of total | `cleanup.done`, `cleanup.total` (`bulkread.go:30-31`) | `identity-bulk.ts:94` | `users/RevocationJobs.tsx:142` |
| Job detail (executed) | From an earlier run | `origin` | `identity-bulk.ts:91` | `users/RevocationJobs.tsx:144` |
| Job detail (executed) | What a bulk revocation covers | fixed copy codes B1-B10 | none | `users/RevocationJobs.tsx:76-85, 145` |
| Job detail (executed) | People, outcome filter | `/items?outcome=` → `total`, `next`, `items[]` (`bulkread.go:127-131`) | `identity-bulk.ts:149-154` | `users/RevocationJobs.tsx:146-157` |
| Job item | Login, user ID, role | `items[].login`, `user_id`, `role` (`bulkread.go:115-117`) | `identity-bulk.ts:123` | `users/RevocationJobs.tsx:52-54` |
| Job item | Outcome and reason | derived `itemWords` from `outcome`, `state`, `receipt.reason` (`users/bulk-model.ts:68-81`); `reason` | `identity-bulk.ts:124-125` | `users/RevocationJobs.tsx:57-60` |
| Job item | Ledger record or not confirmed | `receipt.sequence`, `receipt.reason` (`bulkread.go:109-111`) | `identity-bulk.ts:125` | `users/RevocationJobs.tsx:61` |
| Job item | Cleanup groups | `cleanup[].item`, `state` (`bulkread.go:103-104`) via `groups` (`users/bulk-model.ts:83-89`) | `identity-bulk.ts:126` | `users/RevocationJobs.tsx:62-69` |
| Job item | Replay note | `replay_note` (`bulkread.go:124`) | `identity-bulk.ts:126` | `users/RevocationJobs.tsx:70` |
| Job detail (executed) | Technical: job ID, digest, watermark | `id`, `digest`, `ledger.watermark` | `identity-bulk.ts:91-93` | `users/RevocationJobs.tsx:159-164` |
| Job detail | Discard command | `bulk_revocation.discard` | `admin.ts:871` | `users/RevocationJobs.tsx:100` |

**Parsed but not displayed on M01:** `users[].epoch`, `membership.version`, `clock.wall_time`, `clock_base.ledger_time`/`wall_time`, `recovery_state.records`; bulk list `can_prepare`, `pending`, `ledger_state`, `clock`; `job.recorded_at`, `cleanup[].reason`, item `role_until`, target `role_until`, `user_state`, `membership_state`.

## M02 Authentication policy and trust changes

Read: `GET /api/v1/identity/policy` (`boundary.go:31`), client `admin.ts:883-885`, parser `policyPageOf` (`admin.ts:550-583`) with `trustChangeOf` (`admin.ts:517-548`), `revisionOf` (`admin.ts:505-513`) and `epochOf` (`admin.ts:500-503`); called at `authentication/Live.tsx:96` for the policies view only (`:116`). Server shape `policyPage` (`policy.go:105-120`) with `trustRow` (`policy.go:56-76`), `trustLedgerRow` (`policy.go:78-87`), `rightsRow` (`policy.go:89-94`), `holdersRow` (`policy.go:96-103`), `policyRow` (`policy.go:40-49`) and `changeRow` (`trustrows.go:81-109`).

### Views marked unavailable in pre-launch

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Providers view | Method table: password active, SAML, OIDC, LDAP not available | fixed rows; no read | none | `authentication/Live.tsx:34-64`, chosen at `:117` |
| Test login view | Not available | fixed copy | none | `authentication/Live.tsx:118` |
| Keys view | Not available | fixed copy | none | `authentication/Live.tsx:119` |
| Sync view | Not available, go to Users | fixed copy | none | `authentication/Live.tsx:120` |
| Recovery view | Not available, go to Users | fixed copy | none | `authentication/Live.tsx:121` |

### Page states

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Policies | Loading | derived: no page yet | none | `authentication/Live.tsx:184-189` |
| Policies | Load failed, Try again or Sign in again | failed policy read; `policyFailure` for `unauthenticated` (`authentication/live-model.ts:56`) | `admin.ts:776-783` | `authentication/Live.tsx:163-171` |
| Policies | Denied on first load | `permission_denied` before any page | `admin.ts:755` | `authentication/Live.tsx:132` |
| Policies | Rights removed (denied after a page was shown) | `permission_denied` on a later read | `admin.ts:755` | `authentication/Live.tsx:107-110, 129-131`; `authentication/TrustChanges.tsx:447-457` |
| Policies | Busy while reloading | derived: `status === "loading"` | none | `authentication/Live.tsx:173`; reload button `authentication/TrustChanges.tsx:67` |
| Narrow step view | Short notices above the panel | as Notices below, first sentence only | as below | `authentication/TrustChanges.tsx:52-54`, used at `authentication/Live.tsx:178` |

### Notices

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Notices | Held after an unconfirmed activation, since, link to the change | `trust.hold`, `hold.since` (`policy.go:51-54, 75`); change with `state = "activation_unknown"` | `admin.ts:552, 572, 526` | `authentication/TrustChanges.tsx:26-50`; rule `authentication/trust-model.ts:82` |
| Notices | Trust ledger isolated, problem line, older record ignored | `ledger.state = "isolated"`, `isolation_reason`, `problem_line` (`policy.go:79-81`); `trust.older_record_ignored` (`policy.go:68`) | `admin.ts:575, 571` | `authentication/trust-model.ts:83-86, 102`; `authentication/TrustChanges.tsx:38` |
| Notices | Not durable | `trust.durable_unconfirmed` (`policy.go:65`) or `ledger.not_durable` (`policy.go:86`) | `admin.ts:571, 576` | `authentication/trust-model.ts:87, 103` |
| Notices | Ledger stopped | `ledger.state = "stopped"` without a hold | `admin.ts:575` | `authentication/trust-model.ts:88, 104` |
| Notices | Trust reset at | `trust.reset_at` (`policy.go:67`) | `admin.ts:571` | `authentication/trust-model.ts:89, 105`; `authentication/TrustChanges.tsx:29` |
| Notices | Carried from line | `trust.origin = "carried"`, `carried_line` (`policy.go:62, 64`) | `admin.ts:570` | `authentication/trust-model.ts:90, 106` |

### Current rules

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Current rules | Loaded at | `now` (`policy.go:107`) | `admin.ts:561` | `authentication/TrustChanges.tsx:65` |
| Current rules | No policy | `policy` null (`policy.go:115`) | `admin.ts:557, 560` | `authentication/TrustChanges.tsx:81` |
| Current rules | Sign-in methods | `policy.methods` (`policy.go:28`) via `methodsText` (`authentication/live-model.ts:42-44`) | `admin.ts:508, 560` | `authentication/TrustChanges.tsx:74` |
| Current rules | MFA | `policy.mfa` (`policy.go:29`) | `admin.ts:509` | `authentication/TrustChanges.tsx:75` |
| Current rules | Maximum sign-in age | `policy.max_age_seconds` (`policy.go:30`) | `admin.ts:510` | `authentication/TrustChanges.tsx:76` |
| Current rules | Step-up for policy changes | `policy.step_up[]` entry with `action = "policy.change"` and no MFA (`policy.go:21-25`); default 900 s when absent (`authentication/live-model.ts:21-30`) | `admin.ts:511` | `authentication/TrustChanges.tsx:77` |
| Current rules | Effective step-up window | `step_up_seconds` (`policy.go:111`), default 900 (`authentication/live-model.ts:32-34`) | `admin.ts:565` | `authentication/TrustChanges.tsx:78` |
| Trust | Version | `trust.epoch_generation`, `epoch_number` (`policy.go:58-59`) via `currentVersion` (`authentication/trust-model.ts:14-16`) | `admin.ts:569` | `authentication/TrustChanges.tsx:85` |
| Trust | Policy revision | `trust.policy_number` (`policy.go:60`); "not recorded" without a policy | `admin.ts:569` | `authentication/TrustChanges.tsx:86` |
| Trust | Origin | `trust.origin` (`policy.go:62`) | `admin.ts:570` | `authentication/TrustChanges.tsx:87` |
| Trust | Activated at | `trust.activated_at` (`policy.go:66`) | `admin.ts:571` | `authentication/TrustChanges.tsx:88` |
| Trust | Proposed, validated, approved, activated by | `trust.proposer`, `validator`, `approver`, `activator` (`policy.go:70-73`) | `admin.ts:571-572` | `authentication/TrustChanges.tsx:89-92` |
| Trust | Ledger records and generation | `ledger.records`, `ledger.generation` (`policy.go:82, 84`) | `admin.ts:575-576` | `authentication/TrustChanges.tsx:96` |
| Trust | Carried from | `trust.carried_from` (`policy.go:63`) | `admin.ts:570` | `authentication/TrustChanges.tsx:97` |
| Trust | You may act until / sign-in too old | `act_until`, `now` (`policy.go:107, 110`) via `stale` (`authentication/live-model.ts:46-48`) | `admin.ts:561, 564` | `authentication/TrustChanges.tsx:98` |

### Changes list

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Changes head | Propose button | `rights.propose` (`policy.go:90`) and not unavailable (`authentication/trust-model.ts:70-76`) | `admin.ts:578` | `authentication/TrustChanges.tsx:109` |
| Changes head | Holders ready, qualifying people | `holders.ready`, `qualifying_people` (`policy.go:97, 99`) | `admin.ts:579` | `authentication/TrustChanges.tsx:111-112` |
| Changes head | Enrolment blocked, missing roles | `holders.ready` false, `holders.missing[]` (`policy.go:100`) | `admin.ts:579-580` | `authentication/TrustChanges.tsx:113-121` |
| Changes head | Your trust rights | `rights.propose`, `validate`, `approve`, `activate` (`policy.go:90-93`) via `rightsList` (`authentication/trust-model.ts:170-173`) | `admin.ts:578` | `authentication/TrustChanges.tsx:122` |
| Changes head | Trust changes unavailable | `trust.hold` or `ledger.state` not `ok`, `recovered`, `full` | `admin.ts:552, 575` | `authentication/TrustChanges.tsx:123` |
| Changes head | Changes before this start not kept | any `changes[].earlier_run`, or `trust.origin` `ledger_replay`/`carried`; `started_at` (`policy.go:113`) | `admin.ts:531, 567, 570` | `authentication/TrustChanges.tsx:104, 124` |
| Lists | Open and closed counts, empty lines | derived grouping by `changes[].state` (`authentication/trust-model.ts:7-8, 110-114`) | `admin.ts:526, 581` | `authentication/TrustChanges.tsx:188-198` |
| Changes table | Column headings | fixed copy | none | `authentication/TrustChanges.tsx:152-155` |
| Changes table | Change N (opens detail) | `changes[].number`, `id` (`trustrows.go:82-83`) | `admin.ts:528` | `authentication/TrustChanges.tsx:161` |
| Changes table | From an earlier run | `earlier_run`, `run_started_at` (`trustrows.go:98-99`) | `admin.ts:531` | `authentication/TrustChanges.tsx:163` |
| Changes table | State, evidence changed | `state` (`trustrows.go:84`); stale from `validation.evidence_current` (`trustrows.go:22`) via `stale` (`authentication/trust-model.ts:120-122`) | `admin.ts:526, 533` | `authentication/TrustChanges.tsx:133-141, 173` |
| Changes table | Proposed by | `proposer` (`trustrows.go:92`) | `admin.ts:529` | `authentication/TrustChanges.tsx:174` |
| Changes table | Based on version and policy | `old_epoch.generation`, `.number`, `old_policy` (`trustrows.go:87-88`) | `admin.ts:528-529` | `authentication/TrustChanges.tsx:129-131, 175` |
| Changes table | Next step | derived `nextStep` from `state`, validation result, `invalidation.state` (`authentication/trust-model.ts:130-140`) | `admin.ts:526, 532-542` | `authentication/TrustChanges.tsx:176` |
| Changes table (reduced) | One cell with change, state, next | as above | as above | `authentication/TrustChanges.tsx:169` |

### Change detail

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Detail | Heading, earlier run | `number`, `earlier_run`, `run_started_at` | `admin.ts:528, 531` | `authentication/TrustChanges.tsx:392-395` |
| Detail | State sentence; hold see-notice | derived `stateSentence` (`authentication/trust-model.ts:124-128`); `trust.hold` | `admin.ts:526, 552` | `authentication/TrustChanges.tsx:398-400` |
| Detail | Next step | derived `nextStep` | as above | `authentication/TrustChanges.tsx:401` |
| Detail | Action buttons | `can.revise`, `validate`, `approve`, `activate`, `reject`, `cancel` (`trustrows.go:60-67`) via `actions` (`authentication/trust-model.ts:159-168`) | `admin.ts:543` | `authentication/TrustChanges.tsx:402-407` |
| Detail | Why you cannot approve or activate | `why_not.approve`, `why_not.activate` (`trustrows.go:69-72`) via `whyNot` (`authentication/trust-model.ts:144-155`) | `admin.ts:544` | `authentication/TrustChanges.tsx:408` |
| Outcome (activated) | New version, ledger sequence, activated at | `activation.epoch`, `ledger_sequence`, `at` (`trustrows.go:34-39`) | `admin.ts:537` | `authentication/TrustChanges.tsx:315-319` |
| Outcome (activated) | Session ending progress | `invalidation.items[].item`, `result`, `ended`, `remaining`, `reason` (`trustrows.go:47-58`) | `admin.ts:539-542` | `authentication/TrustChanges.tsx:320-341` |
| Outcome (activation unknown) | Job, waiting, reason | `job.id`, `job.reason` (`trustrows.go:41-45`) | `admin.ts:538` | `authentication/TrustChanges.tsx:346-355` |
| Outcome (superseded) | Superseded by change and version | `superseded_by`, `superseded_epoch` (`trustrows.go:96-97`) | `admin.ts:530-531` | `authentication/TrustChanges.tsx:357-370` |
| Settings compare | Current and proposed maximum age, step-up, MFA | current `policy`; proposed `candidate` (`trustrows.go:91`) | `admin.ts:529, 560` | `authentication/TrustChanges.tsx:200-227, 410-413` |
| Settings compare | Current may be older | derived: `ledger.state` `isolated` or `stopped` | `admin.ts:575` | `authentication/TrustChanges.tsx:387, 224` |
| Validation | Not validated | `validation` null (`trustrows.go:100`) | `admin.ts:519, 532` | `authentication/TrustChanges.tsx:231` |
| Validation | Blocked or validated at, by | `validation.result`, `at` (`trustrows.go:19-20`), `validator` | `admin.ts:533, 530` | `authentication/TrustChanges.tsx:237-239` |
| Validation | Check, result, reason | `validation.items[].item`, `result`, `reason`, `detail` (`trustrows.go:11-16`), fixed check order (`authentication/trust-model.ts:189`), text from `findingReason` (`authentication/trust-model.ts:179-187`) | `admin.ts:534` | `authentication/TrustChanges.tsx:240-264` |
| Validation | MFA weakening, affected subjects, session impact | items `mfa_weakening` (`reason`), `affected_subjects` and `session_impact` (`detail`) | `admin.ts:534` | `authentication/TrustChanges.tsx:265-269` |
| Impact (open changes) | Not computed | `impact` is `null` when the change is not open or its facts could not be read (HYB-314); the screen then says "Not computed. This is not zero." and still shows the sign-out sentence | `admin.ts` `trustChangeOf` | `authentication/TrustChanges.tsx` `Impact` |
| Impact (open changes) | Signs out everyone | `impact.signs_out_everyone` (`trustrows.go:31`), always true when the impact is computed | `admin.ts:536` | `authentication/TrustChanges.tsx:279` |
| Impact (open changes) | Members, sessions now | `impact.members`, `sessions` (`trustrows.go:27-28`) | `admin.ts:536` | `authentication/TrustChanges.tsx:281-282` |
| Impact (open changes) | Affected member IDs, showing N of M | `impact.affected_ids[]` (`trustrows.go:29`), `members` | `admin.ts:536` | `authentication/TrustChanges.tsx:284-290` |
| People | Proposed, validated, approved, activated or attempted by | `proposer`, `validator`, `approver`, `activator` (`trustrows.go:92-95`) | `admin.ts:529-530` | `authentication/TrustChanges.tsx:419-426` |
| History | State, by, at, rationale; approval invalidated | `history[].state`, `by`, `at`, `rationale` (`trustrows.go:74-79`) | `admin.ts:546` | `authentication/TrustChanges.tsx:295-309` |
| Technical | Change ID, revision, base version and raw epoch, base policy, old, new and change digests, evidence digest, ledger sequence and watermark, superseded by | `id`, `change_revision`, `old_epoch.value`, `old_policy`, `old_digest`, `new_digest`, `change_digest`, `validation.evidence_digest`, `activation.ledger_sequence`, `activation.watermark`, `superseded_by`, `superseded_epoch` (`trustrows.go:82-97`) | `admin.ts:528-537` | `authentication/TrustChanges.tsx:428-442` |

### Propose form, confirm dialog and result

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Propose form | Maximum age and step-up, prefilled | typed; prefilled from the revised change's `candidate` or the current `policy` | `admin.ts:505-513` | `authentication/TrustActions.tsx:21-25, 62-68` |
| Propose form | Same as current | derived against `policy` (`authentication/TrustActions.tsx:14-16`) | none | `authentication/TrustActions.tsx:33, 39, 69` |
| Confirm | Change, base version, policy revision, change revision, current state | the change's `number`, `id`, `old_epoch`, `old_policy`, `change_revision`, `state`; for a new proposal `trust.epoch_*` and `trust.policy_number` | `admin.ts:528, 569` | `authentication/TrustConfirmBody.tsx:35-43` |
| Confirm (propose) | Workspace | session `workspace` (`GET /api/v1/session`), not the policy read | `apps/console/src/service/api.ts:101` | `authentication/TrustConfirmBody.tsx:52`; passed at `authentication/Live.tsx:204` |
| Confirm (propose) | Members affected | `impact.members` of the change, else `holders.members` (`policy.go:98`) | `admin.ts:536, 579` | `authentication/TrustConfirmBody.tsx:29, 54` |
| Confirm (propose) | Old digest | `trust.digest` (`policy.go:61`) | `admin.ts:569` | `authentication/TrustConfirmBody.tsx:56` |
| Confirm (propose) | Expected new digest | derived: SHA-256 of the candidate encoding in the browser (`authentication/trust-model.ts:42-68`); marked expected, server authoritative | none | `authentication/TrustConfirm.tsx:38-43`; `authentication/TrustConfirmBody.tsx:57` |
| Confirm (propose) | Proposed settings compare | derived candidate (`authentication/trust-model.ts:29-34`) against `policy` | none | `authentication/TrustConfirmBody.tsx:60-63` |
| Confirm (validate) | Change digest; what is checked and not applicable | `change_digest`; fixed lists | `admin.ts:528` | `authentication/TrustConfirmBody.tsx:67-80` |
| Confirm (approve) | Proposed by, validated by and at, change and evidence digests, lockout finding, members, sessions | `proposer`, `validator`, `validation.at`, `change_digest`, `validation.evidence_digest`, item `recovery_path`, `impact.members`, `impact.sessions` | `admin.ts:528-536` | `authentication/TrustConfirmBody.tsx:83-95` |
| Confirm (activate) | Digests, people, expected version, sessions, test login and lockout findings, affected IDs | as above plus `approver`; expected version derived from `trust.epoch_*` and `ledger.generation` (`authentication/trust-model.ts:18-25`) | `admin.ts:528-536, 569, 576` | `authentication/TrustConfirmBody.tsx:98-121` |
| Confirm (reject, cancel) | Proposed by (reject) | `proposer` | `admin.ts:529` | `authentication/TrustConfirmBody.tsx:125-131` |
| Confirm | Lead, consequences, boundary | fixed copy | none | `authentication/TrustConfirmBody.tsx:132-140` |
| Confirm | Command | `trust.propose`, `validate`, `approve`, `activate`, `reject`, `cancel` (`authentication/TrustConfirm.tsx:13-15`); target `trust` or the change ID; values from `valuesFor` (`authentication/trust-model.ts:236-255`) | `admin.ts:871` | `authentication/TrustConfirm.tsx:63-64` |
| Confirm | Failure, Reload and close | `self_approval_denied`, `authority_epoch_conflict`, `approval_stale`, `evidence_stale`, `independent_approval_required` and others via `trustFailure` (`authentication/trust-model.ts:202-222`); busy while unavailable becomes `trust_unavailable` (`authentication/trust-model.ts:197-200`) | `admin.ts:767-771` | `authentication/TrustConfirm.tsx:74-77, 96, 101` |
| Confirm | Signed out during the command | `unauthenticated` | `admin.ts:757` | `authentication/TrustConfirm.tsx:69-73`; `authentication/Live.tsx:153` |
| After a non-activate command | Done announcement, change opened | result `fields.change` (`trust.go:143`) | `admin.ts:742` | `authentication/Live.tsx:142-152` |
| After activation | Activated, waiting or unclear | result `status`, `business_state`, `job_id` (`errors.go:43, 47-48`) via `afterKind` (`authentication/TrustAfter.tsx:9-13`) | `admin.ts:738-740` | `authentication/TrustAfter.tsx:22-28`; chosen at `authentication/Live.tsx:122-128` |
| After activation | New version | result `fields.epoch_generation`, `fields.epoch_number`, else `fields.epoch` (`trust.go:317-318`) | `admin.ts:742` | `authentication/TrustAfter.tsx:21, 38` |
| After activation | Ledger sequence, sessions ended, sessions remaining | result `fields.ledger_sequence`, `sessions_ended`, `sessions_remaining` (`trust.go:319-320`) | `admin.ts:742` | `authentication/TrustAfter.tsx:39-41` |
| After activation (waiting) | Job, waiting | result `job_id` (`trust.go:314`) | `admin.ts:740` | `authentication/TrustAfter.tsx:33-34` |

**Parsed but not displayed on M02:** `signed_in_at`, `authenticated_at`, `you`, `trust.epoch` (used only as a command value), `policy.number`, `policy.epoch`, `policy.proposer`..`policy.at`, `ledger.watermark`, `ledger.generation_stated`, `holders.needed`, `holders.enrolment`, `impact.weakens_mfa`, change `act_until`.

## M03 Access grants and delegations

Read: `GET /api/v1/identity/access` (`boundary.go:32`), client and parser `admin.ts:886-909`, called at `access/Live.tsx:383` for the grants and effects views (`:388`). Server shape `accessPage` (`access.go:72-85`) with `grantRow` (`access.go:28-41`), `delegationRow` (`access.go:43-58`), `ceilingRow` (`access.go:60-65`), `subjectRow` (`access.go:67-70`), `changeRequestRow` (`accessrequests.go:343-369`), `revocationRow` (`revocations.go:29-53`) and `ledgerRow` (`ledger.go:20-35`, parsed by `ledgerOf`, `admin.ts:712-733`). Preview read: `GET /api/v1/identity/access/preview` (`boundary.go:33`), `admin.ts:910-915`, response key `decisions` (`access.go:275`, rows `access.go:87-93`). The grants, delegations and subjects arrays are cast without per-field normalisation (`admin.ts:891-894`).

### Views and page states

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Roles view | Not available in pre-launch, go to grants | fixed copy | none | `access/Live.tsx:391` |
| Support view | Not available in pre-launch, go to grants | fixed copy | none | `access/Live.tsx:392` |
| Grants view | Denied | access read refused `permission_denied` | `admin.ts:755` | `access/Live.tsx:393-397` |
| Grants view | Load failed, Try again or Sign in again | failed access read; body from `describeFor("load")` (`access/Live.tsx:25-38`) | `admin.ts:776-783` | `access/Live.tsx:398-406, 457` |
| Grants view | Loading | derived: no page yet | none | `access/Live.tsx:456` |
| Effects view | Loading or load failed | as above | as above | `access/Live.tsx:408` |
| Grants view | Done notice | derived: command copy | none | `access/Live.tsx:379, 455` |

### People with access

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Header | Active grants count | derived: current grants with `state = "active"` and `effective` | `admin.ts:891` | `access/Live.tsx:447` |
| Header | Loaded at | `now` (`access.go:78`) | `admin.ts:895` | `access/Live.tsx:448` |
| Header | Request access button (narrow) | derived: first `catalog` entry has days left (`access/live-model.ts:46-49`) | `admin.ts:893` | `access/Live.tsx:451` |
| Grants | No grants | derived: no current person grant (`access/live-model.ts:29-33`) | none | `access/Live.tsx:458` |
| Grants table | Column headings | fixed copy | none | `access/Live.tsx:207-212` |
| Grants table | Person | `grants[].subject` (`access.go:30`) shown as `subjects[].login` (`access.go:68-69`) via `loginOf` (`access/who.tsx:4-6`) | `admin.ts:891, 894` | `access/Live.tsx:219` |
| Grants table | Access, data and customer | `grants[].action`, `fields`, `kind`, `scope.customer` (`access.go:32-35`) via `accessText`, `scopeText` (`access/live-model.ts:57-64`) | `admin.ts:891` | `access/Live.tsx:220` |
| Grants table | Status and until/starts/ended | `state`, `effective`, `not_before`, `not_after` (`access.go:36-39`) via `stateLine` (`access/live-model.ts:14-27`) | `admin.ts:891` | `access/Live.tsx:215, 221` |
| Grants table | Your grant | derived: `subject` equals the session user | none | `access/Live.tsx:224` |
| Grants table | Revocation requested | derived: pending request with `change = "revoke"` and `grant` equal to the row (`access/request-model.ts:9-11`) | `admin.ts:899` | `access/Live.tsx:216, 225` |
| Grants table | Request revocation button | derived: `state` `active` or `pending` (`access/live-model.ts:12`) | none | `access/Live.tsx:226-227` |
| Grants | Ended grants disclosure | derived: `state` `revoked`, `expired` or `ended` (`access/live-model.ts:8-10`) | none | `access/Live.tsx:459-461` |

### Request access form and revoke step

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Request form | Shown | first `catalog` entry with at least one day left before `until` (`access.go:60-65`) | `admin.ts:893` | `access/Live.tsx:43-45, 63, 430` |
| Request form | Additions paused | `ledger.state` not `ok` or `recovered` (`access/request-model.ts:13-15`) | `admin.ts:907, 718` | `access/Live.tsx:61, 98, 129-130` |
| Request form | No eligible person | derived: `subjects` minus self is empty | `admin.ts:894` | `access/Live.tsx:44, 101` |
| Request form | Person choice | `subjects[].user_id`, `login`, excluding self | `admin.ts:894` | `access/Live.tsx:103-109` |
| Request form | Access and data kind | `catalog[0].action`, `fields`, `kind` | `admin.ts:893` | `access/Live.tsx:113` |
| Request form | Customer: all customers | fixed: the form always sends `customer = "all"` | none | `access/Live.tsx:80, 114` |
| Request form | Duration days, hint with maximum and ceiling end | typed `days`; maximum derived from `catalog[0].until` and `now` | `admin.ts:893, 895` | `access/Live.tsx:116-125` |
| Request form | Impact sentence | derived from chosen person, catalog entry and days | none | `access/Live.tsx:126` |
| Request form | Command | `access.request_grant` with `subject`, `action`, `kind`, `fields`, `customer`, `days` | `admin.ts:871` | `access/Live.tsx:79-80` |
| Request form | Days refused, failure box | `invalid_request` with field `days`; others via `describeRequest` (`access/Requests.tsx:21-32`) | `admin.ts:754, 778` | `access/Live.tsx:83-84, 100` |
| Revoke step | Person | grant `subject` via `loginOf`, or delegate login | `admin.ts:891-894` | `access/Live.tsx:150, 176` |
| Revoke step | Access, customer, status | grant `action`, `fields`, `kind`, `scope.customer`, `stateLine`; delegation access and status text passed from the row | `admin.ts:891-892` | `access/Live.tsx:179-183` |
| Revoke step | Delegations ended below | derived from `delegations[].parent` and `state` (`access/request-model.ts:113-122`) | `admin.ts:892` | `access/Live.tsx:172, 185-190` |
| Revoke step | Command | `access.request_revoke` (grant) or `access.revoke_delegation` (delegation) | `admin.ts:871` | `access/Live.tsx:160` |
| Revoke step | Failure box | `describeRequest` or `describeFor("revoke")` | `admin.ts:776-783` | `access/Live.tsx:178` |

### Approval requests

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Requests | Waiting count, none waiting | derived: `requests[].state = "pending"` (`access/request-model.ts:5-7`) | `admin.ts:899` | `access/Requests.tsx:166, 169` |
| Requests table | Column headings | fixed copy | none | `access/Requests.tsx:117-122` |
| Requests table | Add or revoke, person | `change`, `subject` (`accessrequests.go:345-346`) | `admin.ts:899` | `access/Requests.tsx:129-130` |
| Requests table | Proposed by and at | `proposer`, `proposed_at` (`accessrequests.go:356-357`) | `admin.ts:899` | `access/Requests.tsx:131` |
| Requests table | Request ID and short digest | `id`, `digest` first 12 (`access/request-model.ts:40-42`) | `admin.ts:899` | `access/Requests.tsx:132` |
| Requests table | Access, data, customer, days | `action`, `fields`, `kind`, `customer`, `days` (`accessrequests.go:348-352`) via `scopeLine` (`access/Requests.tsx:34-37`) | `admin.ts:899` | `access/Requests.tsx:134` |
| Requests table | State | `state` (`accessrequests.go:354`) via `requestState` (`access/request-model.ts:64-66`) | `admin.ts:899` | `access/Requests.tsx:136` |
| Requests table | Expires | `expires_at` (`accessrequests.go:358`) | `admin.ts:899` | `access/Requests.tsx:137` |
| Requests table | Eligible approvers, no approver | `eligible_approvers` (`accessrequests.go:364`) | `admin.ts:899` | `access/Requests.tsx:138, 140` |
| Requests table | Out of date | `current` false (`accessrequests.go:355`) | `admin.ts:899` | `access/Requests.tsx:139` |
| Requests table | Decided by and at, decision reason | `decided_by`, `decided_at`, `decision_reason` (`accessrequests.go:360-362`) | `admin.ts:899` | `access/Requests.tsx:141-142` |
| Requests table | Approve, Reject, Withdraw | `can_approve`, `can_reject`, `can_withdraw` (`accessrequests.go:365-367`) | `admin.ts:899` | `access/Requests.tsx:145-149` |
| Requests | Requests before this start not kept | `started_at` (`access.go:80`) | `admin.ts:897` | `access/Requests.tsx:170` |
| Requests | Ended requests disclosure | derived: non-pending requests | `admin.ts:899` | `access/Requests.tsx:171-173` |
| Settings disclosure | Setting, value, range and default, source | `settings[].name`, `value`, `unit`, `min`, `max`, `default`, `source` (`access.go:81`; row `settings.go:145-153`) | `admin.ts:898` | `access/Settings.tsx:7-35`, used at `access/Requests.tsx:174` |
| Decide panel | Facts: change, person, access, days, request, expires, reason, revision digest | `change`, `subject`, `action`, `fields`, `kind`, `customer`, `days`, `proposer`, `proposed_at`, `expires_at`, `reason`, `digest` | `admin.ts:899` | `access/Requests.tsx:39-49, 96` |
| Decide panel | May end (approve a revoke) | `may_end[]` (`accessrequests.go:368`) | `admin.ts:899` | `access/Requests.tsx:97-102` |
| Decide panel | Command | `access.approve_request` (with `digest`), `access.reject_request`, `access.withdraw_request` | `admin.ts:871` | `access/Requests.tsx:68, 77` |
| Decide panel | Failure box | `describeRequest` | `admin.ts:776-783` | `access/Requests.tsx:95` |

### Revocations and decision ledger

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Ledger | Status lines: ok with records, isolated, stopped, full, unknown, not durable, recovered, replaced, still available | `ledger.state`, `isolation_reason`, `problem_line`, `records`, `watermark`, `not_durable`, `recovery.at`, `recovery.truncated_bytes`, `replaces` (`ledger.go:15-34`) via `ledgerLines` (`access/request-model.ts:82-107`) | `admin.ts:712-733` | `access/Revocations.tsx:10-15` |
| Ledger | Details: ledger ID, created, watermark, checked, file | `ledger_id`, `created_at`, `watermark`, `checked_at`, `file` (`ledger.go:23-24, 29-30, 34`) | `admin.ts:720-731` | `access/Revocations.tsx:16-25` |
| Revocations | None | derived: empty `revocations` | `admin.ts:900` | `access/Revocations.tsx:97` |
| Revocation record | Target | `target_kind`, `target` (`revocations.go:33-34`); grant person and access looked up in `grants` and `subjects` | `admin.ts:900-906` | `access/Revocations.tsx:32-38, 64` |
| Revocation record | State name and job | `state`, `job_state` (`revocations.go:38, 42`) via `stateWords`, `jobWords` (`access/request-model.ts:44-51`) | `admin.ts:900-906` | `access/Revocations.tsx:65-66` |
| Revocation record | Decision unconfirmed in this process | `state = "decision_unconfirmed"` | `admin.ts:900-906` | `access/Revocations.tsx:67` |
| Revocation record | Earlier run, recorded at, replay; or received at | `origin`, `recorded_at`, `replay`, `received_at` (`revocations.go:31-32, 37, 47`) | `admin.ts:900-906` | `access/Revocations.tsx:68-70` |
| Revocation record | Delegations ended | `ended_delegations[]` (`revocations.go:40`) | `admin.ts:903` | `access/Revocations.tsx:71` |
| Revocation record | Cleanup progress and groups | `cleanup[].item`, `state`, `reason`, `refs` (`revocations.go:11-16`) via `cleanupGroups`, `settled`, `itemWords` (`access/request-model.ts:30-38, 58-62`) | `admin.ts:904` | `access/Revocations.tsx:40-57, 72-73` |
| Revocation record | Details: ID, request, sequence, watermark, deny revision, failure | `id`, `request`, `receipt.sequence`, `receipt.watermark`, `deny_generation`, `receipt.reason` (`revocations.go:19-21, 30, 41, 46`) | `admin.ts:900-906` | `access/Revocations.tsx:74-84` |

### Ceiling and delegations

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Console ceiling | Rows | derived: grants with `client` set and `state = "active"` (`access/live-model.ts:35-37`) | `admin.ts:891` | `access/Live.tsx:413, 473` |
| Console ceiling | Access and data | `action`, `fields`, `kind` | `admin.ts:891` | `access/Live.tsx:475` |
| Console ceiling | Customer | `scope.customer` | `admin.ts:891` | `access/Live.tsx:476` |
| Console ceiling | Active until | `not_after` | `admin.ts:891` | `access/Live.tsx:477` |
| Delegations | Active count, none | derived: delegations not in a closed state (`access/live-model.ts:39-44`) | `admin.ts:892` | `access/Live.tsx:489, 491` |
| Delegations table | Delegator → delegate | `delegator`, `delegate` (`access.go:45-46`) via `loginOf` | `admin.ts:892, 894` | `access/Live.tsx:506` |
| Delegations table | From parent, may redelegate | `parent`, `redelegate` (`access.go:47-48`) | `admin.ts:892` | `access/Live.tsx:507` |
| Delegations table | Access, data and customer | `action`, `fields`, `kind`, `scope.customer` (`access.go:49-52`) | `admin.ts:892` | `access/Live.tsx:508` |
| Delegations table | Status | `state`, `not_before`, `not_after` (`access.go:53-55`) via `stateLine` | `admin.ts:892` | `access/Live.tsx:502, 509` |
| Delegations table | Revoke | derived: `state` `active` or `pending` | none | `access/Live.tsx:510-511` |
| Delegations | Ended disclosure: ID, ended by chain or state | `id`, `state` | `admin.ts:892` | `access/Live.tsx:519-527` |

### Effects preview view

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Form | Person | `subjects[]` of the access read | `admin.ts:894` | `access/Live.tsx:270-274` |
| Form | Data kind (choice when more than one) | derived: distinct `catalog[].kind` | `admin.ts:893` | `access/Live.tsx:240, 276-290` |
| Form | Customer (optional) | typed; sent as `customer` | none | `access/Live.tsx:291-295` |
| Preview | Unconfirmed revocation for this person | derived: a revocation in `received` or `decision_unconfirmed` whose target belongs to the person (`access/request-model.ts:17-26`) | `admin.ts:900` | `access/Live.tsx:299` |
| Preview | Start hint, cleared after a change | derived | none | `access/Live.tsx:300-301` |
| Preview | Loading | derived | none | `access/Live.tsx:303` |
| Preview | Preview failed | failed preview read | `admin.ts:776-783` | `access/Live.tsx:304-309` |
| Preview | Stale result dropped | derived: only the latest request is shown (`access/Live.tsx:247-248, 256, 260`) | none | `access/Live.tsx:256` |
| Result | Title with person | derived: chosen subject login | none | `access/Live.tsx:313` |
| Result | Data kind, customer, checked at | request values; checked at is browser time (`access/Live.tsx:257`) | none | `access/Live.tsx:315` |
| Result | Allowed count | derived from `decisions[].allowed` | `admin.ts:914` | `access/Live.tsx:302, 316` |
| Result table | Action | `decisions[].action` (`access.go:88`) | `admin.ts:914` | `access/Live.tsx:332` |
| Result table | Field | `decisions[].field` (`access.go:89`) | `admin.ts:914` | `access/Live.tsx:333` |
| Result table | Allowed or not allowed | `decisions[].allowed` (`access.go:90`) | `admin.ts:914` | `access/Live.tsx:334` |
| Result table | Why | `decisions[].reason`, `delegations[]` (`access.go:91-92`) via `reasonText` (`access/live-model.ts:66-83`) | `admin.ts:914` | `access/Live.tsx:335` |

**Parsed but not displayed on M03:** `you`, grant and delegation `version`, `scope.legal_entity`/`contract`/`period`, request `grant` (used only to match), `result`, revocation `history`, `target_present`, `job_reason`, `requested_by`, `approved_by`, `deny_at`, `verified`, `completed_at`, ledger `replace_reason`, `replaced_file`; preview `revision`, `subject`, `kind`, `customer`.

## M04 Service accounts

**HYB-304 connected integration candidate, not merge-ready.** The actual M04 entry now uses independent bounded account/credential/pending pages, canonical details, bounded metadata pickers, complete declaration forms and session-owned immutable command recovery. The obsolete population client/parser/types have been removed; P02 credential metadata parsing remains separate. Actual authenticated creation, versioned declaration editing, issuance, one-time collection acknowledgement and revocation passed. Known/off-filter selection and a subsequent sixty-variant read-state/credential-hold matrix were rendered in EN/KO/AR at four widths. Eighteen mounted interaction scenarios now include stale reads, real cursor conflict, immutable replay, keyboard/focus and issuance authentication recovery; sixty-four additional authentication captures distinguish actual password confirmation from injected prerequisite metadata. Remaining full journey/state parity, coverage reconciliation and final integration remain open. See [interaction checkpoint](../../features/HYB-304/m04-consumer.md#interaction-and-read-error-checkpoint-after-ced673a).

The tables below replace obsolete service/population line anchors with the current producer, client and integration boundary. Current code symbols are used because recovery refactoring changed line positions. See [query contract](../../features/HYB-304/m04-queries.md), [strict client](../../features/HYB-304/m04-console.md), [recovery design/status](../../features/HYB-304/m04-recovery.md), [declaration producer](../../features/HYB-304/m04-declarations.md) and [operator settings](../../ops/installation/service-catalog.md#declaration-commands-and-limits).

### Current producers and connected page/form binding

| Field or behavior | Actual producer | Strict client | Current integration boundary |
| --- | --- | --- | --- |
| Canonical account ID, state, version, responsible owner/creator and labels | `servicequery_accounts.go:serviceAccountView`; bounded `GET /identity/services` | `service-query-read.ts:accountPageOf`; `serviceQueryClient.serviceAccounts` | `AccountWorkspace`, `AccountTable` and `AccountDetailView`; independent `svc_a_` query/selection, no first-row fallback or embedded population |
| Purpose, owning organization, exact catalog version and selected audience/scope pairs | `servicedeclaration.go`; `serviceDeclarationView` | `declarationOf` inside account/detail/history parsing | `DeclarationForm` supplies every required field; current owner is read-only during declaration updates |
| Recorded/missing/unavailable declaration and organization label availability | `serviceDeclarationView`; known `readServiceAccount` | `declarationOf`, `accountDetailOf` | `DeclarationForm`, `AccountDetailView` and picker exact-ID reads preserve explicit missing/unavailable/current catalog distinctions |
| Credential aggregates and earliest unexpired expiry | Bounded nested account scan at fixed `reference_at`; known detail explicitly not_evaluated | `summaryOf`; decimal strings and exact sum invariant | Account list exposes reference aggregates; canonical issue controls use observed `eligibility.live_credentials` and decimal limit, never loaded row counts. Recent use remains unavailable |
| Exact admission count and configured account limit | `serviceAdmission`; list/details independently of filters, including retired accounts | `admissionOf`; `AccountPage`/`AccountDetail` | `AccountWorkspace` displays exact recorded/limit separately from filtered exact/lower-bound count |
| Independent credential rows and service/credential IDs, validity, fixed state, current_state, usable metadata | `servicequery_credentials.go`; bounded `GET /identity/service-credentials` | `credentialOf`; `serviceCredentials` | `CredentialWorkspace` and `CredentialTable` use `svc_c_` query/known selection; explicit issuance target remains separate. `CredentialStatus` keeps lifecycle separate from service state and active-but-unusable hold/revoke guidance; `CredentialPeriod` shows both validity endpoints. Usable is not request authentication or an effective grant |
| Active delivery observation, estimated pickup end | `serviceCredentialView`; `servicedelivery.go`; `pickupLive` with side-effect-free session peek | Optional secret_pickup/secret_pickup_until; canonical `serviceCredential` | `RecoveryRegion` observes exact locally recorded credentials; `PendingCollection` owns its independent bounded workspace page and live pickup annotations |
| Current issue/reinstate/declare/suspend/retire/revoke prerequisites | `currentServiceEligibility`, `currentCredentialEligibility` in `servicequery_detail.go` | `accountDetailOf`, `credentialDetailOf`; contradictory flags rejected | `AccountDetailView`, `IssueControl`, `KnownRevoke` and `ActionForm` use canonical current detail; explicit reauthentication refreshes without submitting; unsupported MFA and unavailable authentication evidence have distinct explanations and passive Reload |
| Bounded owners, organizations, audiences and scopes with full restrictions | `servicequery_choices.go` and exact known-ID handlers | `serviceOwners`, `serviceOrganizations`, `serviceAudiences`, `serviceScopes` and canonical detail methods | `Picker` uses one bounded page and an independent exact known-ID read, including the new metadata-only `serviceAudience` route. `ScopeDefinition` displays complete structured declared restrictions |
| Retained declaration versions or not_recorded/version_unavailable | `queryServiceDeclarations` | `serviceDeclarations`; `declarationPageOf` | `DeclarationHistory` pages retained versions and names unavailable versions; current definitions never reconstruct historical catalog restrictions |
| Search/filter/cursor/reference/count certainty/settings | `servicequery_model.go`, `servicequery_cursor.go`, actual settings/CLI catalog | Strict page parsers; exact search_input correlation separate from normalized query.q | `navigation`, `QueryFilters`, `useScopedRead` and bounded cursor-only trails preserve other view parameters. Explicit First clears an expired/source-conflicting cursor; Reload preserves it. Read errors use M04 read-specific explanations and sanitized server diagnostics, with prior rows hidden |
| Effective command value/rationale/body byte limits | `queryServiceLimits` exposes decimal `command_value_bytes`, effective min(256, configured rationale), and existing1048576-byte body limit | `limitsOf` requires exact/bounded metadata | `admitsDeclaration` and `admitsCommand` check UTF8 and actual serialized JSON escaping before retaining/sending a draft |
| Create and update declaration | `serviceDefinitions`, `service.create` and `service.declare`; purpose/organization/catalog_version/scopes required, plus expected declaration_version for update | Existing command envelope and CommandId union | `DeclarationForm`, `declarationFields` and `commandIntents`; complete frozen envelope/identity and original revision survive refresh/locale/view changes |
| Issue, suspend/reinstate/retire, revoke commands and partial/unknown outcomes | Existing command registry; identity/step-up/hold/reference checks remain authoritative | `AdminClient.command`, scoped issue store | `IssueControl` and direct original-issue recovery use stored values/window; `ActionForm` matches frozen command and target before any current-target callback. The server remains authoritative |
| Usage history, credential request authentication and owner transfer | No connected producer/adapter in this increment | None | Usage/transfer views remain explicitly unavailable; metadata must not fabricate these capabilities |

### Implemented recovery binding

| Area | Source and invariant | Current consumer |
| --- | --- | --- |
| Session owner | `ServiceRoot` synchronously updates sessionRef and supplies stable matchesSession; `ServiceAccountRecoveryProvider` owns the committed client/session layout token and effect-created store | Context-only wrapper around authenticated service children; no DOM addition. `LiveServices` gates until a matching live store exists |
| Navigation and scope | Same client/user/workspace/CSRF retains the store across locale/view remounts; layout cleanup invalidates/disposes replacement scope before passive cleanup | `RecoveredServices`, `IssueControl`, `RecoveryRegion` and `commandIntents` use the scoped stores; no module-global issues/afterView singleton |
| Local required/byte/integer errors and keyboard recovery | Existing current metadata limits, naming/whole-day rules and reference preflight remain authoritative | `form-validation`, `DeclarationForm`, `ActionForm` and `IssueControl` identify actual fields, preserve rejected values and link/focus summary targets. Owner/organization fieldsets exist during loading/failure; selects have separate IDs. Canonical credential used/limit and recorded-account/limit are isolated LTR in all locales, separate from filtered matching count. [Scoped actual validation](../../features/HYB-304/m04-form-errors.md) |
| Command error/outcome meaning | Existing command/issuance error, retained generic hadUnknown and existing issuance refusal.retry; ErrorDiagnostics sanitizes code/request | `CommandFailure` keeps possible original completion distinct from latest refused retry, gives target/context-specific first refusal and separate confirmation copy. `IntentNotice` owns one phase heading; no raw message, global access-loss or save/user implication. [Verification and unchanged F2 boundary](../../features/HYB-304/m04-command-errors.md) |
| Unsent state/revocation rationale | `action-draft.ts:actionDrafts` retains one exact immutable action/target/reason; same-lease canonical refresh/view remount preserves, replacement/auth loss disposes | `ActionForm` rechecks current first-submit bounds/eligibility, keeps refused drafts and original unknown intents separate, never silently replaces another target/action. Explicit discard clears only captured identity; Cancel closes while preserving. [Actual verification](../../features/HYB-304/m04-action-draft.md) |
| Unsent issuance draft | `issue-draft.ts:issueDrafts` retains exactly one immutable target/days/reason under the current authenticated lease; byte bounds and identity-safe clearing | `IssueControl` restores across canonical refresh/locale/view, offers explicit other-target keep/discard and rechecks current first-submit eligibility. Provider disposes on scope/auth loss and guards unload; state/revocation rationale uses its separate action draft |
| Issuance retry | `issue-attempts.ts:submit` supplies the original copied frozen days/rationale and original revision/attempt to Exec | IssueControl and direct original retry send those supplied values, not mutable caller values; held unknown/lapsed behavior remains |
| Exact reconciliation | `AdminClient.serviceCredential` against the current expected workspace; scoped read sequence plus exact service/credential ID | `RecoveryRegion` observes locally recorded issued entries. Page absence, stale/mismatched/failed reads and never_collected/not-open alone retain the reference |
| Authoritative terminal result | Current revoked/expired or returned-once observation of the retained issued entry | Retires only that matching entry. Returned-once adds a canonical service+credential unstored marker; acknowledged markers are not recreated by repeated reads |
| Opening Reveal | `issue-attempts.ts:open` copies the exact retained ref/credential without consuming it | Reveal retains canonical identity; its initial browser-clock expiry is explicitly estimated until a matching canonical credential read supplies actual not_after. Updating that metadata does not retrigger collection; opening alone does not assert returned or stored |
| One-time transport | `secret-collection.ts` shares one pending-only promise for exact service/credential/ref; current scope and visible document checked before intent and immediately before dispatch | `Reveal` calls scoped collect through the original AdminClient.secret. StrictMode replay attaches to the same pending operation; unsent hidden requests are not unknown deliveries |
| Successful or unknown collection | Valid matching response retires issued reference and adds an unstored marker; failed/malformed/mismatched response retains unknown reference without automatic retry | Current visible observer may display the secret once; explicit user retry is required after unknown collection. No resolved secret is retained in store state |
| Hidden/unmounted observer | Observer generation and scope/visibility checked before display; pagehide invalidates generation and clears the display | A pending response cannot repopulate a hidden/BFCache-restored view. Actual native BFCache remains unverified; fixture pagehide is controlled simulation |
| Secret display and storage acknowledgement | Validated secret and exact credential match; secret exists only in the current Reveal component's displayed state | Existing shown-once help, local storage checkbox and Done; no clipboard/log/browser-storage integration added |
| Revocation meaning | Only the shown-secret action explicitly declares not-stored; other reveal states do not establish storage | Unknown-storage Revoke opens with empty rationale; shown I-did-not-store action retains the explicit user's reason |
| Unstored reminder | Scope-owned canonical Marker {service, credential}, independent of loaded rows | Reminder display/revoke uses the recorded service, with no current-selection/page fallback; Dismiss is local and is not server proof of storage |
| Cleanup and unload | Store activate/dispose, actual auth-loss subscription, deliberate leave, scoped pending cancellation | Accidental unload guarded while necessary; session/client replacement, authenticated unmount and explicit release fence late callbacks and clean subscriptions |

Recovery status copy distinguishes recorded-not-returned/currently-not-open from permanent deletion. Generic failed/hidden observations do not assert returned or stored; confirmed-return text requires an observed matching validated response. KO/AR copy remains candidate/source-reviewed rather than native/practitioner approved. No secret, pickup reference, request key or rationale is added to reminder DOM attributes or browser storage.

Isolated actual React StrictMode/Chromium fixtures exercise this recovery lifecycle with synthetic transport and simulated visibility/pagehide. They are not complete M04 rendered journeys, native BFCache, native assistive-technology or owner FIX acceptance. The actual issuance authentication matrix adds twelve real password confirmations across EN/KO/AR at1440/1280/1024/390 and actual sign-out/new sign-in, as recorded in [issuance draft verification](../../features/HYB-304/m04-issue-draft.md). The isolated provider suite now has twenty scenarios, including action-only retention and scope cleanup. Full state/form parity, coverage reconciliation, final review and authorized merge of draft PR251 remain pending.

## M05 My security

HYB-304 candidate read: bounded `GET /api/v1/me`, requested by `AdminClient.me` and strictly parsed by `account-read.ts:selfPageOf`. `self.go:readSelf` exposes account/policy metadata, a separately available `current_session`, canonical recorded-session/accepted-operation counts and the opaque legacy generation token; it does not emit session populations or expanded operation arrays. Identity fields are strictly parsed. Dedicated query, selection and operation consumers are mapped below. Password changes use `AdminClient.changePassword`; selected termination uses `sessionQueryClient.endSelectedSessions`. Both legacy request variants remain API-compatible but are not used by the new screen.

### Views and page states

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Recovery view | Not available in pre-launch, forgot-password guidance, go to account | fixed copy | none | `my-security/Live.tsx` |
| Notifications view | Not available in pre-launch, go to sessions | fixed copy | none | `my-security/Live.tsx` |
| All views | Signed out or locked out | `me` read `unauthenticated`, or a password change or end-session answer that signs the person out | `account-read.ts:selfPageOf` | `my-security/Live.tsx` |
| All views | Loading | derived: no page yet | none | `my-security/Live.tsx` |
| All views | Load failed, Try again | any other failed `me` read | `account-read.ts:selfPageOf` | `my-security/Live.tsx` |
| All views | Loaded at | `now` (`self.go:48`) | `account-read.ts:selfPageOf` | `my-security/Live.tsx` |

### Account view

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Account | Sign-in email | `login` (`self.go:36`) | `account-read.ts:selfPageOf` | `my-security/Live.tsx` |
| Account | Contact email or not recorded | `email` (`self.go:37`) | `account-read.ts:selfPageOf` | `my-security/Live.tsx` |
| Account | User ID | `user_id` (`self.go:35`) | `account-read.ts:selfPageOf` | `my-security/Live.tsx` |
| Account | Password set or not | `password_set` (`self.go:41`) | `account-read.ts:selfPageOf` | `my-security/Live.tsx` |
| Account | Profile changes not available note | fixed copy | none | `my-security/Live.tsx` |
| Access | Workspace | `workspace` (`self.go:38`) | `account-read.ts:selfPageOf` | `my-security/Live.tsx` |
| Access | Role | `role` (`self.go:39`) via `roleName` (`users/live-model.ts:14-16`) | `account-read.ts:selfPageOf` | `my-security/Live.tsx` |
| Access | Installation administrator | `installation_administrator` (`self.go:42`) | `account-read.ts:selfPageOf` | `my-security/Live.tsx` |
| Access | Role note | fixed copy | none | `my-security/Live.tsx` |

### Sign-in methods view and password change

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Methods table | Column headings | fixed copy | none | `my-security/Live.tsx` |
| Methods table | Method | fixed copy `m.password`; `identities[].method` (`self.go:selfIdentity`, always `local-password` at `self.go:readSelf`) is used only as a row key | `account-read.ts:selfPageOf` | `my-security/Live.tsx` |
| Methods table | Signs in as | `identities[].login` (`self.go:31`) | `account-read.ts:selfPageOf` | `my-security/Live.tsx` |
| Methods table | Status | `password_set` | `account-read.ts:selfPageOf` | `my-security/Live.tsx` |
| Methods | Linking other methods not available note | fixed copy | none | `my-security/Live.tsx` |
| Password form | No password to change | `password_set` false (`my-security/live-model.ts:12-14`) | `account-read.ts:selfPageOf` | `my-security/Live.tsx` |
| Password form | Current, new, confirm | typed; sent as `current`, `new` | none | `my-security/Live.tsx`, call `:89` |
| Password form | Local checks: missing, rule, same, mismatch | derived (`my-security/live-model.ts:6-10`; `passwordProblem` with `login`) | `account-read.ts:selfPageOf` | `my-security/Live.tsx` |
| Password form | Wrong current password | `current_password_wrong` → `wrong` | `account-read.ts:selfPageOf` | `my-security/Live.tsx` |
| Password form | New password refused by the rule | `password_form` → `form` | `account-read.ts:selfPageOf` | `my-security/Live.tsx` |
| Password form | Locked, limited, conflict, busy, result unknown | `password_locked`, `rate_limited`, `revision_conflict`, `dependency_unavailable`, other (`my-security/live-model.ts:22-38`) | `account-read.ts:selfPageOf` | `my-security/Live.tsx` |
| Password form | Changed | successful answer | `account-read.ts:selfPageOf` | `my-security/Live.tsx` |

### Sessions view

HYB-304 is an in-progress replacement of the former full-population account read. Current candidate bindings follow; this table does not declare M05 complete or owner-accepted. Historical HYB-307 bindings remain in Git at commit 5fbd7791127d7b369a03a5f11baaf33a0d3ddc1e.

| View/area | Displayed field, column or state | Producer | Parser and rendered consumer |
| --- | --- | --- | --- |
| Current session | Independent reference, login time and state | `/me.current_session`; `self.go:readSelf` and canonical session index | `account-read.ts:selfPageOf`; `LiveSessions.tsx:ScopedSessions` |
| Sessions table | Current/other, handle, signed-in/expiry time, state, permitted End action | `GET /me/sessions.rows[]` and known-ID `/me/sessions/{handle}`; current canonical owner/epoch/workspace only | `session-query.ts:sessionPageOf/sessionRowOf`; `LiveSessions.tsx:SessionRows`; `SessionSelection.tsx` rechecks the known target |
| Confirmed-set exclusion tag | Neutral not-included label and View confirmed-list operation | Query/detail/account `not_included_operation`; constant-work latest retained accepted-others subset and immutable held membership | Strict `sessionRowOf`; `LiveSessions.tsx:ExclusionTag`; scoped `SessionOperations.tsx:ReferencedOperation` known-ID summary, without traversing held pages |
| Filters | Current/other/all, search, from/before timestamps and page size | Strict server query allowlist; canonical normalized query; 1–100 rows | `session-query-model.ts`; `LiveSessions.tsx` form, applied filters and URL/history |
| Count and paging | Exact/lower-bound recorded count, this-page row count, scan stop, reference time, previous/next/first | `count`, `scan_stopped`, `reference_at`, signed caller/query/session-generation/owner-operation-change-bound cursor | `session-query.ts`; bounded page state and at most 32 prior continuation links in `LiveSessions.tsx` |
| Empty/error | No matching rows on this page; invalid query; changed/expired cursor; read failure | Explicit response/count basis, never page-length inference of the full population | `LiveSessions.tsx`; `query-copy.ts` |
| Limits | Idle, longest session, max sign-in age, signed-in and authenticated times | Account/policy metadata; longest session is `/me.absolute_minutes`, the server's absolute session lifetime (`self.go`, `sessionAbsolute`) | `account-read.ts`; `LiveSessions.tsx` limits panel |
| End entry | All other recorded sessions in current workspace, independent of displayed filters/page | Explicit preparation kind `others`, not browser handles | `SessionSelection.tsx`; `query-copy.ts:fullScope` |
| One-session entry | One currently eligible own known handle | Dedicated known-ID admission then exclusive `{session:handle}` preparation | `session-query.ts:sessionDetail/prepareSessionSelection`; `SessionSelection.tsx` |
| Preparation/preview | Kind, reference time, building progress vs sealed exact count/digest, immutable bounded rows | `POST /me/sessions/selections`, `GET /me/sessions/selections/{reference}` | `selectionOf`; `SessionSelection.tsx`; no full target array in browser |
| Submission | Exact selection reference/digest/count for either kind | `POST /me/sessions/end`; same canonical caller, admission and step-up at effects | `sessionQueryClient.endSelectedSessions`; `session-intent.ts` |
| Uncertain result | New actions locked; same-reference read-back/replay, retained-result unavailable | Original selection/operation identity, nullable unconfirmed outcomes | `session-intent.ts`, `IntentRecovery`; in-memory client/scope retention, not document-reload durability |
| Result summaries | Recorded/ended/gone/not-included counts, access vs cleanup state | Dedicated bounded operation list and known-ID summary; constant-work server counters | `sessionOperationOf`; `SessionOperations.tsx`; unknown is never zero/completed |
| Result rows | Ended vs already-gone and eight fixed cleanup items | `/me/sessions/operations/{operation}/items?view=results`; bounded rows | `SessionOperations.tsx:OperationRows/ResultRow`; per-item categories and already-cleared distinctions grouped by done/pending/failed/not-applicable/outside-scope |
| Not included | Separate bounded confirmed-set exclusion rows and known-ID End action | Same item route with `view=not_included`; never added to the confirmed set | `SessionOperations.tsx`; fresh one-session preparation before effects |
| Cleanup refresh | Manual check, 2-second polling of displayed first-page running operations for at most 30 seconds | Owner operation version and current summary; no whole-population poll | `SessionOperations.tsx`; explicit stopped/error state |
| Scope/lifecycle | Hide prior-scope data synchronously, discard late read responses, preserve only scoped submitted intent | Canonical account admission and current API/read token | `scoped-read.ts`; tagged records in `Live`, `LiveSessions`, selection and operation readers; guarded parent callbacks |

**Remaining verification before M05 integration:** the current candidate now binds cross-page confirmed-set exclusion tags and effective preparation/query limits with exact expiry; finish their final integrated review and advanced failure/cleanup/building/large-result variants. These are tracked in [the integration plan](../../features/HYB-304/session-ui-integration.md), not removed from the acceptance scope. Inherited history now retains one page and at most 32 continuation parents; see the focused [history follow-up](../../features/HYB-304/ui-progress.md#bounded-inherited-history-follow-up). Final failure/cleanup/building/large-result browser variants, source/coverage/mutation checks and owner review remain distinct pending work.

The current candidate binds query `settings[]` to `SessionSettings` (three expected rows), and preparation `settings[]` to its preparation scope (six expected rows). Each missing known setting is explicitly unavailable; effective values, ranges/defaults and source are distinct and read-only. `SessionSelection` binds exact `expires_at` to `PreparationExpiry` without Date conversion or local admission. Shared capacity includes not-included rows and retry references, not free slots or session totals. Actual scoped verification remains in the HYB-304 evidence records.

### HYB-306 recent access history

Implementation candidate; combined verification is pending. The new read is `GET /api/v1/me/access-history`, handled by `apps/api/server/accesshistory.go`: `readHistory` and `parseHistoryQuery`. It takes identity from `self`, never a caller-supplied user ID.

| Area | Field or behavior | Producer | Consumer |
| --- | --- | --- | --- |
| History rows | `events[].at`, `kind`, `reason`, `workspace`, `session`, `device`, `count` | `modules/accesshistory`: `Read`; server `historyEvent` mapping | `service/access-history.ts`: `eventOf`; `AccessHistory.tsx`: `HistoryRow`; `history-copy.ts` |
| Period | `from`, fixed current workspace; server also accepts `to` | `parseHistoryQuery`; per-user/per-workspace record sequences | `history-model.ts`: `fromFor`; period control |
| Paging | `before`, `next`; two own-sequence positions, default 50, max 100 | `encodeCursor`, `decodeCursor`, `accesshistory.Read` | `AdminClient.accessHistory`; `AccessHistory.tsx`: one response page, first/previous/next, max32 parent cursors, identical-direction retry and fixed `from` while paging |
| Retention | `retention_days`, `retention_source`, `period_since`, `effective_days` | durable retention state and installation start settings | `accessHistoryOf`; retention and size-limit statements |
| Coverage | `available_from`, `now` | oldest visible own record and server clock | coverage statement, period gap, loaded time |
| Recording | `recording.state`, `since`, `reason` | history writer status | stopped-recording note; never represented as an empty successful history |
| Loss, holds and deletion | `notes[].kind`, `from`, `to`, `count` | history storage, purge and persisted hold markers; attributable own counts only | `history-model.ts`: `noteLines`; ordered note region |
| Scope revocation | clear rows and cursor on unauthenticated answer; remount by user/workspace/session | authenticated self read and API error | `AccessHistory.tsx`: canonical/client keyed content and layout-invalidated read token; outer key in `my-security/LiveSessions.tsx` |
| Storage | append-only day files, retention and person state in installation data directory | `modules/accesshistory`; `Server.openHistory`, `Server.Close` | survives server restart within the plan's incarnation and storage limits |

No network address, raw User-Agent, login name, email or bearer secret is stored or displayed by this feature. Device values are fixed browser/OS families. Old held records stay hidden; backup deletion is not claimed.

**Parsed but not displayed on M05:** `/me.recorded_session_count`, `/me.accepted_operation_count`, opaque legacy `sessions_generation`, `identities[].method`, history `period_since`, and `recording.reason` (the recording state and start time drive the visible notice).

## Unknowns and mismatches

- **Unknown.** None of the rows above is marked unknown.
- **Server fields that no console parser reads:** `trust.stricter_fields` (`policy.go:69`, not in `policyPageOf`, `admin.ts:568-573`); `impact.api_keys` and `impact.pending_jobs` (`bulkread.go:85, 87`, not in `detailOf`; the preview shows fixed "not applicable" copy instead, `users/BulkPrepare.tsx:73, 77`); `revocations[].target_version` (`revocations.go:35`, not in the `Revocation` type, `admin.ts:167-190`, although the spread at `admin.ts:900-901` keeps it at run time); trust command result `fields.reason` and `fields.your_session_ended` (`trust.go:314, 320`), `fields.change_revision` (`trust.go:143`); set-role-period result `fields.role_until` (`periods.go:130`); bulk result `fields.digest`, `fields.targets` and `fields.count` (`bulk.go:40, 105`, `bulkstage.go:86`).
- **Parser type wider than the read.** `access`, `services` and `me` cast their row arrays without checking each field (`admin.ts:891-894, 921`, and `identities` in `me`), unlike `userRowOf`, `trustChangeOf` and the bulk parsers. A missing field arrives as `undefined` rather than an empty string or zero.

## HYB-327 reconciliation

The preceding gap inventory records the original mapping review. HYB-327 validates O01 and identity response rows at the read boundary. M02 now displays the fields affected by the restored/default comparison, distinguishing an explicit empty list from unavailable data. M01 selection is already bound by the filter-selection follow-up; API-key and pending-job impact now require explicit server states. See [verification](../../features/HYB-327/verification.md) for executed checks and limitations. Unrelated unread result fields remain outside this change.

## HYB-304 M05 advanced confirmation and authority follow-up

`SessionSelection.tsx` hides a definitively refused initial confirmation synchronously and resets its reference/preview/checkbox for explicit review. A missing submitted reference remains an unknown outcome with the same fingerprint locked in `session-intent`; it is never converted to an empty successful result. Building completion/refusal focuses the stable dialog heading. `admin.ts` and the shared root remove sensitive reads on actual401 or semantic authentication loss; stale prior-session prompts cannot retire a successor session or authorize a retry. See the [L01 shared boundary mapping](./sign-in.md#hyb-304-authenticated-boundary-follow-up).

Large actual result pages and later-arrival exclusion pages were traversed independently with exact counts and no duplicate handles. Pending/failed cleanup render variants use explicitly intercepted responses over a real accepted operation; those images do not claim a real backend cleanup failure. Current verification and retained limitations are in [advanced evidence](../../features/HYB-304/advanced-evidence.json). Parent M04/M03/M02 and owner acceptance remain open.
