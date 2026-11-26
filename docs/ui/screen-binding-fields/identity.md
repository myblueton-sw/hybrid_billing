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
- **Commands.** Every command goes through `POST /api/v1/commands/{command}` (`boundary.go:25`), sent by `command()` (`admin.ts:976`). The answer envelope is `resultBody` (`errors.go:41-52`); `outcomeOf` (`admin.ts:796-805`) maps `status`, `business_state`, `job_id`, `result_revision` and `fields`. "Result `fields.x`" below means `Outcome.fields.x`.
- **Failures.** A refused read or command becomes an `AdminError` through `failureOf` (`admin.ts:841-851`, code table `admin.ts:811-833`). The message comes from the screen's describe function named in each row.
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

Rewritten on 2026-11-27 for HYB-304 after the list of trust changes moved from the policy read to bounded reads (increments M2a to M2e of the [M02 query design](../../features/HYB-304/m02-query-design.md); composition in the [M02 composition](../../features/HYB-304/m02-composition.md)). Paths under `authentication/` are `apps/console/src/modules/identity/pages/authentication/`; `service/` is `apps/console/src/service/`; server files are under `apps/api/server/`. A reference names a function and the line it starts at. Routes are registered at `boundary.go:33-35`.

Reads:

| Read | Server | Client and parser | Used for |
| --- | --- | --- | --- |
| `GET /api/v1/identity/policy` | `listPolicy` (`policy.go:145`), built by `policyPage` (`policy.go:179`), shape `policyPage` (`policy.go:105`): `revision`, `now`, `signed_in_at`, `authenticated_at`, `act_until`, `step_up_seconds`, `you`, `started_at`, `trust`, `policy`, `ledger`, `rights`, `holders`, `waiting_activation`, `earlier_run_kept`; rows `trustRow` (`policy.go:56`), `trustLedgerRow` (`:78`), `rightsRow` (`:89`), `holdersRow` (`:96`), `policyRow` (`:40`) over `revisionRow` (`:27`) | `policy()` (`service/admin.ts:988`), `policyPageOf` (`:583`) with `revisionOf` (`:538`) and `epochOf` (`:533`) | the command revision, the notices, the current rules, the changes head, the current side of every settings comparison; it carries no list of changes and no change |
| `GET /api/v1/identity/policy/changes` | `queryTrustChanges` (`trustlists.go:88`) through `openList` (`accesslists.go:133`) and `closeList` (`:181`); row `listedChange` (`trustlists.go:35`) built by `listed` (`:65`); page `listPage` (`accesslists.go:57`) | `trustChanges` (`service/trust-changes.ts:96`), `trustChangePageOf` (`:61`), `listedTrustChangeOf` (`:47`) | the open and the closed list |
| `GET /api/v1/identity/policy/changes/{id}` | `readTrustChange` (`trustlists.go:118`), shape `trustChangeDetail` (`:59`): `change`, `revision`, `observed_at`; `change` is `changeRow` (`trustrows.go:81`) built by `row` (`:190`) | `trustChange` (`service/trust-changes.ts:110`), `trustChangeDetailOf` (`:68`) with `trustChangeOf` (`service/admin.ts:550`) | the detail panel, the revise form's starting values, the confirm dialog's facts about a change |

The two change reads come from `trustChangeClient` (`service/trust-changes.ts:94`). The policy read is called by `LivePolicy` (`authentication/Live.tsx:69`) for the policies view only.

The list read takes `state`, `q`, `limit` and `after` (`parseListQuery`, `accesslists.go:71`). `state` is one of the eight states or a group (`trustChangeGroups`, `trustlists.go:20`): `open` is `draft`, `validated`, `independently_approved` and `activation_unknown`; `closed` is `activated`, `rejected`, `cancelled` and `superseded`. `q` matches a part of the change identifier. A page answers `rows`, `next`, `count` (`value`; `certainty` is always `exact` on this list), `reference_at`, `observed_at` and the echoed `query`, in the order `created_desc` (newest first). A continuation is bound to the registry's run and version (`trustBasis`, `trustlists.go:84`) and answers `revision_conflict` once a change was recorded since.

**What a list row carries and what only the detail read carries.** A list row is `id`, `number`, `state`, `change_revision`, `old_epoch`, `old_policy`, `candidate`, `proposer`, `validator`, `approver`, `activator`, `superseded_by`, `earlier_run`, `run_started_at`, `validation_result`, `evidence_current`, `invalidation_state`, `decide_in_detail`, `can.revise`, `can.validate`, `can.reject`, `can.cancel`, `why_not.approve`, `why_not.activate` and `act_until`. It does not carry evidence currency: `evidence_current` is `not_evaluated` (a change in `draft`, `validated` or `independently_approved` that has a validation) or `not_applicable`, and `why_not` is computed with the evidence taken as current, so it never says `evidence_stale`. It does not carry `can.approve`, `can.activate`, the impact, the validation's time, digest and items, the activation, the job, the invalidation items, the history, the digests or `superseded_epoch`. `decide_in_detail` is true for `draft`, `validated` and `independently_approved` and false for every other state, that is `activation_unknown` and the four closed states (`listed`, `trustlists.go:65`; `Open`, `modules/trustchange/types.go:160`). The detail read answers all of these (`row`, `trustrows.go:190`; `actions`, `:231`; `whyApprove`, `:152`; `whyActivate`, `:168`).

### Views marked unavailable in pre-launch

| View/area | Displayed field, column or state | Source | Rendered at |
| --- | --- | --- | --- |
| Providers view | Method table: password active; SAML, OIDC, LDAP not available; a button to the policies view | fixed rows; no read | `Providers` (`authentication/Live.tsx:37`), chosen in `LivePolicy` (`:69`) |
| Test login view | Not available | fixed copy | `Unavailable` (`authentication/Live.tsx:25`), chosen in `LivePolicy` |
| Keys view | Not available | fixed copy | as above |
| Sync view | Not available, go to Users | fixed copy | as above |
| Recovery view | Not available, go to Users | fixed copy | as above |

### Page states

All in `LivePolicy` (`authentication/Live.tsx:69`) unless another component is named.

| View/area | Displayed field, column or state | Source | Rendered at |
| --- | --- | --- | --- |
| Policies | Loading | no policy page yet and the policy read in flight | `LivePolicy` |
| Policies | Load failed, with Try again; or the session-ended sentence with Sign in again; a line that the shown data is kept when a page is still shown | the policy read failed; `unauthenticated` takes its sentence from `policyFailure` (`authentication/live-model.ts:56`) | `LivePolicy` |
| Policies | Denied on first load | the policy read refused `permission_denied` before any page was shown | `Unavailable` (`authentication/Live.tsx:25`) |
| Policies | Rights removed, with Reload | the policy read refused `permission_denied` after a page had been shown; the page, the open change, the form and the dialog are dropped | `Revoked` (`authentication/TrustChanges.tsx:388`) |
| Policies | Busy while reloading | the policy read in flight | `LivePolicy` (the view's root); the Reload button in `CurrentRules` (`authentication/TrustChanges.tsx:57`) |
| Policies | Done announcement after a command | command copy with the change's number | `LivePolicy` |
| Narrow step view | Back to trust changes; the load failure; short notices; the form or the detail | shown below the wide breakpoint (`useWide`, `users/Live.tsx:48`) while the form or a change is open; the notices are the first sentence of each notice below, without the hold notice when the shown change is in `activation_unknown` | `LivePolicy`; `ShortConditions` (`authentication/TrustChanges.tsx:53`) |

In the step view the notices, the current rules and both lists stay mounted and hidden, so the filter and the page of each list are kept while a panel is open. The changes section (head, filter row, lists, panel) is drawn only while a policy page is present.

### Notices

Component `Conditions` (`authentication/TrustChanges.tsx:27`); which notices apply comes from `conditions` (`authentication/trust-model.ts:80`), their sentences from `conditionText` (`:99`). Every source is the policy read.

| Area | Displayed field, column or state | Source |
| --- | --- | --- |
| Hold | Held after an unconfirmed activation; since | `trust.hold` present; `trust.hold.since` |
| Hold | "Open the change waiting for activation" | shown when `waiting_activation` is not empty; it opens that identifier through the detail read. `waiting_activation` is the identifier of a kept change in `activation_unknown`, or empty (`Summary`, `modules/trustchange/pages.go:82`) |
| Isolated | Trust ledger isolated; the problem line; an older record was ignored | `ledger.state = "isolated"`; `ledger.problem_line` when `ledger.isolation_reason` is `integrity_failed` or `epoch_order`; `trust.older_record_ignored` |
| Not durable | Not confirmed as durable | `trust.durable_unconfirmed` or `ledger.not_durable` |
| Stopped | Ledger stopped | `ledger.state = "stopped"` with no hold, and only when the not-durable notice does not apply |
| Reset | Trust reset at | `trust.reset_at` |
| Carried | Carried from line | `trust.origin = "carried"` with `trust.carried_line` above zero |

### Current rules

Component `CurrentRules` (`authentication/TrustChanges.tsx:57`). Every source is the policy read.

| Area | Displayed field, column or state | Source |
| --- | --- | --- |
| Head | Loaded at; Reload | `now` |
| Rules | No policy | `policy` is null |
| Rules | Sign-in methods | `policy.methods` through `methodsText` (`authentication/live-model.ts:42`) |
| Rules | MFA | `policy.mfa` |
| Rules | Maximum sign-in age | `policy.max_age_seconds` through `duration` (`authentication/live-model.ts:9`) |
| Rules | Step-up for policy changes | the `policy.step_up[]` entry with `action = "policy.change"` and no MFA (`policyStep`, `authentication/live-model.ts:21`); the default of 900 seconds when absent (`stepText`, `:27`; `defaultStepUp`, `:25`) |
| Rules | Effective step-up window | `step_up_seconds`, or 900 when it is not above zero (`effectiveStep`, `authentication/live-model.ts:32`) |
| Trust | Version | `trust.epoch_generation`, `trust.epoch_number` through `currentVersion` (`authentication/trust-model.ts:14`) |
| Trust | Policy revision | `trust.policy_number`; "not recorded" when `policy` is null |
| Trust | Origin | `trust.origin` |
| Trust | Stricter fields: the names, none, or unknown | `trust.stricter_fields`; the parser gives null unless it is a list of strings, shown as unknown |
| Trust | Activated at | `trust.activated_at`; "not recorded" when empty |
| Trust | Proposed, validated, approved, activated by | `trust.proposer`, `trust.validator`, `trust.approver`, `trust.activator` |
| Ledger line | Ledger records and generation | `ledger.records`, `ledger.generation` |
| Note | Carried from | `trust.carried_from` when `trust.origin = "carried"` |
| Window line | You may act until; or sign-in too old | `act_until` when not empty; too old when `now` is after `act_until` (`stale`, `authentication/live-model.ts:46`) |

### Changes head

Component `ChangesHead` (`authentication/TrustChanges.tsx:105`). Every source is the policy read.

| Area | Displayed field, column or state | Source |
| --- | --- | --- |
| Propose button | shown or not | `rights.propose` and trust changes available (`canPropose`, `authentication/trust-model.ts:74`) |
| Holders line | qualifying people | `holders.ready` true (`enrolled`, `authentication/trust-model.ts:169`); `holders.qualifying_people` |
| Enrolment blocked | lead, missing roles, qualifying people | `holders.ready` false; `holders.missing[]`; `holders.qualifying_people` |
| Rights line | your trust rights, or none | `rights.propose`, `rights.validate`, `rights.approve`, `rights.activate` through `rightsList` (`authentication/trust-model.ts:164`) |
| Unavailable line | trust changes unavailable | `trust.hold` present, or `ledger.state` not `ok`, `recovered` or `full` (`unavailable`, `authentication/trust-model.ts:70`) |
| Note | changes before this start are not kept, with the start time | `earlier_run_kept`, or `trust.origin` `ledger_replay` or `carried`; `started_at` |

### Filter row

Component `Filters` (`authentication/TrustLists.tsx:98`) inside `TrustLists` (`:125`); the filter is held by `TrustLists`, not in the address.

| Area | Displayed field, column or state | Source |
| --- | --- | --- |
| Change ID | input with a hint | typed; accepted while the trimmed text fits 64 UTF-8 bytes (`searchFits`, `authentication/trust-list-model.ts:17`); sent as `q` when not empty |
| State | All, then `draft`, `validated`, `independently_approved`, `activation_unknown`, `activated`, `rejected`, `cancelled`, `superseded` under their state names | fixed choices from `trustGroups` (`service/trust-changes.ts:6`); decides which section is drawn and its `state` |
| Buttons | Apply filters, Reset filters | applying or resetting starts both lists again at their first page |

### Open and closed lists

`TrustLists` draws up to two sections, each its own paged read: `PagedSection` (`authentication/TrustLists.tsx:89`) through `usePagedList` (`access/paged.tsx:17`), drawn by `Section` (`authentication/TrustLists.tsx:65`). Which section is drawn and with which `state` comes from `sectionState` (`authentication/trust-list-model.ts:11`):

- With no state chosen, the open section reads `state=open` and the closed section `state=closed`.
- With one state chosen, only the section whose group holds that state is drawn, and it reads `state=<that state>`; the other section is not drawn.
- The open section sends `limit=50` (`openPageSize`, `authentication/trust-list-model.ts:8`) and the closed section `limit=25` (`closedPageSize`, `:9`). Both send the same `q`.

Both lists start again at their first page whenever the policy read succeeds again (after a command, after Reload) and whenever filters are applied: `LivePolicy` advances a generation on each successful policy read and `TrustLists` combines it with a count of applied filters.

| Area | Displayed field, column or state | Source |
| --- | --- | --- |
| Heading | "Open (n)" or "Closed (n)" only while the section's read is ready; the bare title while a read is in flight or has failed | `count.value` of that section's read, the exact number of matches for the current filter |
| Loading | status line | the section's read in flight |
| Failure | title for the open or the closed list; wording; Reload, or Sign in again; First when not on the first page | the section's read failed; `ListFailure` (`access/paged.tsx:49`); wording from `readFailure` (`access/Permissions.tsx:80`): denied, "query changed" for `revision_conflict` or `invalid_request`, otherwise unavailable. One section's failure leaves the other section and the current rules |
| Order, open section | sorted by state priority: `activation_unknown`, `independently_approved`, `validated`, `draft`, then the higher number, then the higher identifier | derived in the browser (`byStatePriority`, `authentication/trust-list-model.ts:29`) only when the group is whole on one page: the first page with an empty `next` (`wholeOnOnePage`, `:25`) |
| Order note, open section | "Newest first. The open changes fill more than one page, so they are not sorted by state." | shown when the open section is not whole on one page; the rows are then in the service's order |
| Order, closed section | newest first | always the service's order; no note |
| Empty | no open changes, no closed changes | `rows` empty with no filter |
| No match | no open change, or no closed change, matches the filters | `rows` empty with a search or a state applied (`filtered`, `authentication/trust-list-model.ts:21`) |
| Evidence note, open section | "Whether a change's evidence is still current is checked when you open it." | shown when a listed row is in `validated` or `independently_approved` (`evidenceLeftToDetail`, `authentication/trust-list-model.ts:48`) |
| Reference time | list reference time under each table | `reference_at`; `Reference` (`access/paged.tsx:78`) |
| Paging | First, Previous, Next; after moving, focus rests on the reference line | `next`; drawn only off the first page or when `next` is not empty; `Paging` (`access/paged.tsx:66`) |

### Changes table

Component `ChangeRows` (`authentication/TrustLists.tsx:18`). Every source is a row of the list read.

| Area | Displayed field, column or state | Source |
| --- | --- | --- |
| Column headings | Change, State, Proposed by, Based on, Next step | fixed copy |
| Change | "Change N", which opens the detail | `number`; `id` is the key and what the detail read is asked for; the number is passed to the panel for its heading |
| Change | Selected tag and current-row mark | derived: the row's `id` is the open change |
| Change | From an earlier run, with that run's start | `earlier_run`, `run_started_at` |
| State | state name; emphasised for `activation_unknown` and for a `draft` whose validation was blocked | `state`; `validation_result = "blocked"` (`listedBlocked`, `authentication/trust-list-model.ts:33`). The list has no "evidence changed" line |
| Proposed by | user ID, or not recorded | `proposer` |
| Based on | version and policy revision | `old_epoch.generation`, `old_epoch.number` through `versionOf` (`authentication/trust-model.ts:10`); `old_policy` |
| Next step | next step | derived by `listedNext` (`authentication/trust-list-model.ts:37`) from `state`, `validation_result` and `invalidation_state`; it never says the evidence is stale, which only the detail can |
| Reduced table | one cell with the change, the state and the next step | as above; used at the wide layout while a panel is open beside the list |

Not displayed from a row: `change_revision`, `old_epoch.value`, `candidate`, `validator`, `approver`, `activator`, `superseded_by`, `evidence_current`, `invalidation_state` other than through the next step, `decide_in_detail`, `can`, `why_not`, `act_until`. No action button is drawn in the list; actions are offered in the detail from the detail read's `can`.

### Detail panel read and states

`useChangeDetail` (`authentication/TrustDetail.tsx:23`) reads the detail once for each opened identifier, again for each generation of the policy read, and again on Try again. The policy read's generation advances on every successful policy read, so the detail is read again after each command that returns to this view and after Reload, as both lists are. No detail is read outside the policies view or while the after-activation screen is shown.

| Area | Displayed field, column or state | Source |
| --- | --- | --- |
| Openers | a list row; the hold notice's button; the superseded change's successor button; the change a command answered | the row's `id` with its `number`; `waiting_activation`; the detail's `superseded_by`; result `fields.change` |
| Heading before the answer | "Change N", or the identifier in mono | the number passed by the list row or known from the command; the identifier when opened from the hold notice or as a successor |
| Loading | "Loading the change…"; Close in the head at the wide layout | no answer yet for the open identifier (`shownRead`, `authentication/TrustDetail.tsx:17`); `DetailPending` (`:39`) |
| Failure | "The change could not be loaded"; wording; Try again, or Sign in again; Close | the detail read failed otherwise than `invalid_request`; wording from `readFailure` (`access/Permissions.tsx:80`) |
| No longer kept | "This change is no longer kept. Older changes are removed when the list reaches its limit."; one Close | the detail read refused `invalid_request` (`goneFrom`, `authentication/TrustDetail.tsx:13`). The service answers `invalid_request` for an unknown identifier and also for a malformed one, a request with a query string and a session of another workspace (`readTrustChange`, `trustlists.go:118`); the screen shows this sentence for all of them |
| Leaving a change no longer kept | Close, or Back in the step view, reads the policy again and so both lists, and rests focus on the changes heading | `LivePolicy` |
| Shown while read again | an earlier loaded change stays; actions are ignored; the panel is marked busy at the wide layout. An earlier failure or no-longer-kept result does not stay: loading is shown | a loaded answer for the same identifier from an earlier read (`shownRead`; the `stale` flag of `useChangeDetail`) |

The detail body is drawn only while a policy page is present, because it compares against `policy` and reads `trust.hold` and `ledger.state` from it.

### Change detail

Component `Detail` (`authentication/TrustChanges.tsx:319`). Unless a row says "policy read", every source is `change` of the detail read.

| Area | Displayed field, column or state | Source |
| --- | --- | --- |
| Heading | "Change N"; from an earlier run | `number`; `earlier_run`, `run_started_at` |
| State sentence | the state; evidence changed since validation or approval; validation blocked | `stateSentence` (`authentication/trust-model.ts:118`) from `state`, `validation.result` (`blocked`, `:110`) and `validation.evidence_current` false on a `validated` or `independently_approved` change (`stale`, `:114`) |
| State sentence | "see the notice" for a waiting activation | `state = "activation_unknown"` at the wide layout with `trust.hold` of the policy read present |
| Next step | next step, with the stale case | `nextStep` (`authentication/trust-model.ts:124`) from `state`, `validation.result`, `validation.evidence_current`, `invalidation.state` |
| Action buttons | Activate, Approve, Validate, Revise, Reject, Cancel | `can.activate`, `can.approve`, `can.validate`, `can.revise`, `can.reject`, `can.cancel` through `actions` (`authentication/trust-model.ts:153`) |
| Why not | why you cannot approve or activate | `why_not.approve`, `why_not.activate` through `whyNot` (`authentication/trust-model.ts:138`); only for `draft`, `validated` and `independently_approved`, and not when the evidence is stale; for `independently_approved` only `why_not.activate` is used |
| Outcome, activated | new version, ledger sequence, activated at | `activation.epoch`, `activation.ledger_sequence`, `activation.at`; `Outcome` (`authentication/TrustChanges.tsx:252`) |
| Outcome, activated | session-ending progress: item, result, ended and remaining, reason | `invalidation.items[].item`, `result`, `ended`, `remaining` (for `console_sessions`), `reason` |
| Outcome, activation unknown | job, waiting, reason | `job.id`, fixed "waiting", `job.reason` shown twice: worded and raw |
| Outcome, superseded | superseded by: the successor's identifier and version; "Open the replacing change" | `superseded_by`, `superseded_epoch`; the button opens that identifier through the detail read. The successor's number is not shown |
| Settings compare | current and proposed maximum age, step-up and MFA | current: `policy` of the policy read; proposed: `candidate`; `Compare` (`authentication/TrustChanges.tsx:131`) |
| Settings compare | current may be older | `ledger.state` of the policy read is `isolated` or `stopped` |
| Validation | not validated | `validation` is null; `Findings` (`authentication/TrustChanges.tsx:160`) |
| Validation | blocked at, by, with the number of blocked checks; or validated at, by | `validation.result`, `validation.at`, `validator`, the `validation.items[]` whose `result` is `blocked` |
| Validation | check, result, reason | `validation.items[].item`, `result`, `reason`, `detail`, in the fixed order of `checkItems` (`authentication/trust-model.ts:183`); wording from `findingReason` (`:173`) |
| Validation | MFA weakening, affected subjects, session impact | the items `mfa_weakening` (`reason`), `affected_subjects` and `session_impact` (`detail`) |
| Impact | section shown | only for `draft`, `validated` and `independently_approved`; `Impact` (`authentication/TrustChanges.tsx:205`) |
| Impact | signs out everyone; "Not computed. This is not zero." | `impact` is null: the service could not read the facts for this change |
| Impact | signs out everyone; members; sessions now | `impact.signs_out_everyone`, `impact.members`, `impact.sessions` |
| Impact | affected member IDs; showing a part of them | `impact.affected_ids[]` (at most 100: `maxAffectedShown`, `trustevidence.go:33`); the part note when `impact.members` exceeds the listed IDs |
| People | proposed, validated, approved by; activated by, or attempted by for a waiting activation | `proposer`, `validator`, `approver`, `activator`; "not yet" when empty |
| History | state, by, at, rationale; "approval invalidated" | `history[].state`, `by`, `at`, `rationale`; `History` (`authentication/TrustChanges.tsx:236`) |
| Technical details | change ID, change revision, base version with the raw epoch, base policy, old, new and change digests, evidence digest, ledger sequence and watermark, superseded by | `id`, `change_revision`, `old_epoch`, `old_epoch.value`, `old_policy`, `old_digest`, `new_digest`, `change_digest`, `validation.evidence_digest`, `activation.ledger_sequence`, `activation.watermark`, `superseded_by`, `superseded_epoch` |

### Propose form, confirm dialog and result

A change named in this part is the detail read's `change` that the panel showed when the action was pressed; the list rows are not used.

| Area | Displayed field, column or state | Source |
| --- | --- | --- |
| Propose form | maximum age and step-up, prefilled | typed; prefilled from the revised change's `candidate`, or from `policy` of the policy read for a new proposal; `ProposeForm` (`authentication/TrustActions.tsx:18`) |
| Propose form | title: new proposal, or "revise" with the change's number | fixed copy; the revised change's `number` |
| Propose form | values typed before the review, when the dialog closes back to the form | typed (`initial` of `ProposeForm`); they take precedence over the prefill above |
| Step view | the panel's label: the form's title, or "Change N", or the identifier when the number is unknown | the detail's `number`, else the number passed by the opener, else the open identifier (`panelLabel` in `LivePolicy`, `authentication/Live.tsx:69`) |
| Propose form | same as current | derived against `policy` of the policy read (`sameAs`, `authentication/TrustActions.tsx:14`) |
| Confirm | title: the action with the change's number | the action's label (`primaryLabel`, `authentication/TrustConfirmBody.tsx:9`); the change's `number`, 0 for a new proposal (`ConfirmDialog`, `authentication/TrustConfirm.tsx:18`) |
| Confirm | change, base version, policy revision, change revision, current state | the change's `number`, `id`, `old_epoch`, `old_policy`, `change_revision`, `state`; for a new proposal `trust.epoch_generation`, `trust.epoch_number` and `trust.policy_number` of the policy read; `ConfirmBody` (`authentication/TrustConfirmBody.tsx:24`) |
| Confirm (propose) | workspace | the session's `workspace` (`service/api.ts`), not the policy read |
| Confirm (propose) | route: local password | fixed copy |
| Confirm (propose) | members affected | `impact.members` of the revised change when above zero, else `holders.members` of the policy read; "not recorded" when zero |
| Confirm (propose) | change revision, from and to | the revised change's `change_revision` |
| Confirm (propose) | old digest | `trust.digest` of the policy read |
| Confirm (propose) | expected new digest | derived: SHA-256 of the candidate's encoding in the browser (`configDigest`, `authentication/trust-model.ts:65`; `candidateOf`, `:29`); marked expected, the service is authoritative |
| Confirm (propose) | proposed settings compare | the derived candidate against `policy` of the policy read |
| Confirm (validate) | change digest; what is checked and what is not applicable | `change_digest`; fixed lists |
| Confirm (approve) | proposed by, validated by and at, change and evidence digests, lockout finding, members, sessions now | `proposer`, `validator`, `validation.at`, `change_digest`, `validation.evidence_digest`, the item `recovery_path`, `impact.members`, `impact.sessions`; "not computed" when `impact` is null; the "your authority" row is fixed copy |
| Confirm (activate) | digests, people, expected version, sessions now, test login and lockout findings, affected IDs | as above with `approver` and the item `test_login`; the expected version is derived from `trust.epoch_generation`, `trust.epoch_number` and `ledger.generation` of the policy read (`expectedNext`, `authentication/trust-model.ts:18`); `impact.affected_ids[]`; "to be activated by: you" is fixed copy |
| Confirm (reject, cancel) | proposed by, for reject | `proposer` |
| Confirm | lead, consequences, boundary | fixed copy |
| Confirm | command | `trust.propose`, `trust.validate`, `trust.approve`, `trust.activate`, `trust.reject`, `trust.cancel` (`commandOf`, `authentication/TrustConfirm.tsx:14`); target `trust` for a new proposal, else the change's `id`; expected revision `revision` of the policy read; values from `valuesFor` (`authentication/trust-model.ts:230`): `trust.epoch` and `trust.policy_number` of the policy read, the change's `change_revision`, `change_digest` and `validation.evidence_digest`, the typed minutes; sent by `command()` (`service/admin.ts:976`) |
| Confirm | failure; Reload and close | wording from `trustFailure` (`authentication/trust-model.ts:196`); a busy answer while trust changes are unavailable is shown as unavailable (`asTrustFailure`, `:191`); a failure in `fatalFailures` (`:189`), or an `invalid` failure that names any field other than `rationale`, ends the dialog's use (`ConfirmDialog`, `authentication/TrustConfirm.tsx:18`): the reason input and the command button are dropped, the body drops the "your authority" and lockout rows (approve) and the test-login and lockout rows (activate), and only Reload and close is left, which reads the policy again |
| Confirm | signed out during the command | `unauthenticated`: the dialog closes and the page shows the session-ended failure |
| After a command other than activate | done announcement; the change opened in the panel | result `fields.change` (`trustOutcome`, `trust.go:143`), else the confirmed change's `id`; the number in the announcement is the confirmed change's `number`, or for a new proposal the last part of the answered identifier read as a number; then the policy read, both lists and the detail are read again |
| After activation | activated, waiting, or unclear | result `status`, `business_state`, `job_id` through `afterKind` (`authentication/TrustAfter.tsx:9`); `AfterActivation` (`:15`) replaces the whole view; nothing is read again |
| After activation | new version | result `fields.epoch_generation` and `fields.epoch_number`, else `fields.epoch` (`activateTrust`, `trust.go:276`) |
| After activation | ledger sequence, sessions ended, sessions remaining | result `fields.ledger_sequence`, `fields.sessions_ended`, `fields.sessions_remaining` |
| After activation, waiting | job, waiting | result `job_id` |
| After activation | the sentences of each kind; Sign in again for every kind; no facts when unclear | fixed copy; the kind from `afterKind` |

**Parsed but not displayed on M02:** from the policy read `signed_in_at`, `authenticated_at`, `you`, `trust.epoch` (used only as a command value), `trust.change`, `trust.hold.reason`, `policy.number`, `policy.epoch`, `policy.proposer`, `policy.validator`, `policy.approver`, `policy.activator`, `policy.at`, `ledger.watermark`, `ledger.generation_stated`, `holders.needed`, `holders.enrolment`; `ledger.isolation_reason` only decides whether the problem line is shown. From the detail read `revision`, `observed_at`, and the change's `candidate.methods`, `impact.weakens_mfa`, `job.state`, `act_until`, `activation.epoch.value` and `superseded_epoch.value` (only the version from `versionOf` is shown); the change's `invalidation.state` is used only through the next step. From every list page `search_input`, `query`, `observed_at`, `revision`, `order`, `work_units`, `scan_stopped` and `count.certainty`, besides the row fields named under the changes table.

## M03 Access grants and delegations

Rewritten on 2026-11-27 for HYB-304 after the grants view moved to bounded reads. Paths under `access/` are `apps/console/src/modules/identity/pages/access/`; `service/` is `apps/console/src/service/`; server files are under `apps/api/server/`. A reference names a function and the line it starts at. Routes are registered at `boundary.go:34-41`.

Reads:

| Read | Server | Client and parser | Used for |
| --- | --- | --- | --- |
| `GET /api/v1/identity/access` | `listAccess` (`access.go:134`), shape `accessPage` (`access.go:40`): `revision`, `catalog`, `catalog_incomplete`, `now`, `you`, `started_at`, `settings`, `ledger` | `access()` (`service/admin.ts:991`), `catalogOf`, `settingsOf` (`service/access-read.ts:4-5`), `ledgerOf` (`service/admin.ts:760`) | the command revision, the request form's choices, the settings disclosure, the ledger line; it carries no list of grants, delegations, people, requests or revocations |
| `GET /api/v1/identity/access/permissions` | `queryPermissions` (`accessquery.go:278`), row `permissionRow` (`accessquery.go:54`) | `accessPermissions` (`service/access-query.ts:70`), `permissionOf` (`:34`) | the one list of grants and delegations |
| `GET /api/v1/identity/access/permissions/grant/{id}`, `/delegation/{id}` | `readPermission` (`accessquery.go:401`), shape `permissionDetail` (`accessquery.go:120`) | `accessPermission` (`service/access-query.ts:102`) | the revoke panel's list of delegations that would stop |
| `GET /api/v1/identity/access/requests` | `queryAccessRequests` (`accesslists.go:203`), row `listedRequest` (`accesslists.go:50`) over `changeRequestRow` (`accessrequests.go:359`) | `accessRequests` (`service/access-lists.ts:118`), `listedRequestOf` (`:35`) | waiting and ended requests, the decision panel |
| `GET /api/v1/identity/access/revocations` | `queryAccessRevocations` (`accesslists.go:259`), row `revocationRow` (`revocations.go:29`) built by `revocationRow` (`revocations.go:153`) | `accessRevocations` (`service/access-lists.ts:123`), `listedRevocationOf` (`:55`) | revocation records |
| `GET /api/v1/identity/access/subjects` | `queryAccessSubjects` (`accesssubjects.go:142`), row `listedSubject` (`accesssubjects.go:46`) | `accessSubjects` (`service/access-lists.ts:128`) | the person picker |
| `GET /api/v1/identity/access/preview` | `previewAccess` (`access.go:166`), row `decisionRow` (`access.go:50`) | `preview()` (`service/admin.ts:998`), `previewsOf` (`service/access-read.ts:6`) | the effect check's result |

Every list read answers `rows`, `next`, `count` (`value` with `certainty` `exact` or `lower_bound`), `reference_at`, `observed_at` and the echoed `query`.

### Views and page states

| View/area | Displayed field, column or state | Source | Rendered at |
| --- | --- | --- | --- |
| Roles view, Support view | Not available in pre-launch, go to grants | fixed copy | `LiveAccess` (`access/Live.tsx:381`), `Unavailable` (`:369`) |
| Grants view | Denied | the whole read refused `permission_denied` | `LiveAccess` |
| Grants view | Load failed, Try again or Sign in again | the whole read failed; wording from `describeFor("load")` (`access/Live.tsx:31`) | `LiveAccess` |
| Grants view | Loading | no whole page yet | `LiveAccess` |
| Grants view | Done notice, under the list heading or the requests heading | command copy; the place follows the command | `LiveAccess`; `RequestSection` (`access/Requests.tsx:163`) |
| Effects view | Loading or load failed | as above | `LiveAccess` |

The requests section, the revocations section and the request form are drawn only while the whole read is present, because they take `started_at`, `settings`, `ledger`, `catalog`, `now` and `revision` from it.

### Grants and delegations list

Component `PermissionList` (`access/Permissions.tsx:228`); rows `Rows` (`:162`); filters `Filters` (`:103`) from `permissionFilters` (`:25`), kept in the address as `acc_*`; composition in `docs/features/HYB-304/m03-list-composition.md`.

| Area | Displayed field, column or state | Source |
| --- | --- | --- |
| Filters | Search, holder user ID, action, customer ID, legal entity ID, source, state, ends before | sent as `q`, `subject`, `action`, `customer`, `legal_entity`, `source`, `state`, `expires_before`; action choices are the distinct `catalog[].action` of the whole read |
| Count line | Matching count, exact or at least; list reference time | `count.value`, `count.certainty`, `reference_at` |
| Column Holder | sign-in name and user ID; "Console" with "Console ceiling" for a console-client grant; "Delegated by" for a delegation | `subject_login`, `subject` (empty for a console grant), `delegator_login`, `delegator` |
| Column Access | action and fields, the resource kind below | `action`, `fields`, `kind` |
| Column Scope | customer, legal entity, and contract and period when restricted | `scope.customer`, `scope.legal_entity`, `scope.contract` (each `any` or `id`), `scope.period` |
| Column Source | Grant or Delegation; can delegate further; the parent when folded | `source`, `redelegate`, `parent` |
| Column Original delegation | the parent delegation or None | `parent` |
| Column Validity | from and until | `not_before`, `not_after` |
| Column Effective permission | the state with its reason; "Revocation requested" | `state`, `effective`, `revocation_requested`; wording from `effectiveLines` (`access/Permissions.tsx:94`) |
| Column Actions | Request revocation; Your grant; Set by the installation; Revoke | `source`, `subject` against the session user, an empty `subject`, `state` through `revocable`, `revocation_requested` |
| Paging | First, Previous, Next | `next`; 50 a page |
| States | Loading, no match, no match yet with more to scan, failed with Reload and First, filters that cannot be read | the read's state; `readFailure` (`access/Permissions.tsx:80`) |

Not displayed from a row: `id` (used as the row's key and in its action's target), `client` (a console grant is recognised by an empty `subject`), `version`.

### Request access form

Component `GrantForm` (`access/Live.tsx:46`).

| Area | Displayed field or state | Source |
| --- | --- | --- |
| Shown | the form | a `catalog` entry of the whole read with at least a day before `until`, counted from `now`, whose access has wording; in the narrow layout only after Request access is pressed |
| Additions paused | warning; the button disabled | `ledger.state` not `ok` or `recovered` |
| Person | search, results, chosen line, the unconfirmed-revocation sentence | `PersonPicker` (`access/PersonPicker.tsx:12`) over the subjects read: `q`, 25 a page; rows `user_id`, `login`, `revocation_unconfirmed`; the signed-in person is left out here |
| Person | No eligible person | the first page of the subjects read holds no one else and has no `next` |
| Access | choice, the hint about the console's own limits, and a second hint when the catalog could not be listed in full | `catalog[].action`, `fields`, `kind`; `catalog_incomplete` |
| In place of the form | a note that no access to grant could be listed | `catalog_incomplete` with no usable `catalog` entry |
| Customer | all customers | fixed: the form sends `customer = "all"` |
| Days | value, hint with the maximum | typed; the maximum from the chosen entry's `until` and `now` |
| Impact sentence | derived from the chosen person, entry and days | none |
| Command | `access.request_grant` with `subject`, `action`, `kind`, `fields`, `customer`, `days`, and `revision` | `command()` (`service/admin.ts:976`) |
| Failure | days refused; other refusals | `invalid_request` with field `days`; `describeRequest` (`access/Requests.tsx:24`) |

### Revoke panel

Component `Revoke` (`access/Live.tsx:206`), opened from a row of the list.

| Area | Displayed field or state | Source |
| --- | --- | --- |
| Person, access, customer, status | the row's facts | the permissions row the panel was opened from |
| Delegations that would stop applying | count, lead for a grant or a delegation, checked time, up to ten identifiers and the rest behind a disclosure; none; checking; could not be checked with Try again | `would_end` and `observed_at` of the detail read of that grant or delegation; `WouldStop` (`access/Live.tsx:166`) |
| Command | `access.request_revoke` for a grant, `access.revoke_delegation` for a delegation, with the identifier and `revision` | `command()`; the button does not wait for the detail read |
| Failure | refusals | `describeRequest` or `describeFor("revoke")` |

### Access change requests

Component `RequestSection` (`access/Requests.tsx:163`); rows `Rows` (`:115`); composition in `docs/features/HYB-304/m03-requests-composition.md`. Two lists, each through `usePagedList` (`access/paged.tsx:17`): `state=pending` 50 a page, `state=ended` 25 a page.

| Area | Displayed field or state | Source |
| --- | --- | --- |
| Heading | waiting count, exact or at least | the pending read's `count` |
| Row | add or revoke, the person | `change`, `subject_login` or `subject` |
| Row | proposed by and at | `proposer_login` or `proposer`, `proposed_at` |
| Row | request ID and short digest | `id`, `digest` |
| Row | access, data, customer, days | `action`, `fields`, `kind`, `customer`, `days` through `scopeLine` (`access/Requests.tsx:37`) |
| Row | state; expires; eligible approvers, none; out of date | `state`, `expires_at`, `eligible_approvers`, `current` |
| Row | decided by and at, the decision's reason | `decided_by_login` or `decided_by`, `decided_at`, `decision_reason` |
| Row | Approve, Reject, Withdraw | `can_approve`, `can_reject`, `can_withdraw` |
| Under each list | list reference time; First, Previous, Next when there is more than a page | `reference_at`, `next`; `Reference` (`access/paged.tsx:78`), `Paging` (`:66`) |
| Note | requests before this start are not kept | `started_at` of the whole read |
| Ended list | collapsed, the summary with the count exact, at least, or could not be loaded; hidden when its first page is empty and has no `next` | the ended read's `count` or failure |
| States | loading, none waiting, failed with Reload and First; one list's failure leaves the other | `ListFailure` (`access/paged.tsx:49`) |
| Settings disclosure | setting, value, range and default, source | `settings[]` of the whole read; `SettingsDisclosure` (`access/Settings.tsx:7`) |
| Decision panel | change, person, access, days, request, expires, reason, digest; what may end when approving a revocation | the request row the panel was opened from, with `may_end[]`; `DecidePanel` (`access/Requests.tsx:54`), `Facts` (`:42`) |
| Decision panel | command | `access.approve_request` with `digest`, `access.reject_request`, `access.withdraw_request`, with `revision` |

Not displayed from a row: `grant`, `result`.

### Revocations and decision ledger

Component `RevocationSection` (`access/Revocations.tsx:102`); composition in `docs/features/HYB-304/m03-revocations-composition.md`. Ten records a page.

| Area | Displayed field or state | Source |
| --- | --- | --- |
| Heading | revocation count, exact or at least | the read's `count` |
| Ledger line | status lines and details | `ledger` of the whole read; `LedgerLine` (`access/Revocations.tsx:14`) |
| Record heading | "Grant: access for person" or "Delegation to person: access"; the resource kind and identifier below; the identifier alone when the target is gone | `target_kind`, `target`, `target_subject_login` or `target_subject`, `target_action`, `target_fields`, `target_resource_kind`; `Target` (`access/Revocations.tsx:36`) |
| Record | state name and job | `state`, `job_state` |
| Record | decision not confirmed in this process | `state = "decision_unconfirmed"` |
| Record | earlier run, recorded at, replay; or received at | `origin`, `recorded_at`, `replay`, `received_at` |
| Record | delegations ended | `ended_delegations[]` |
| Record | cleanup progress and groups | `cleanup[].item`, `state`, `reason`, `refs`; `Cleanup` (`access/Revocations.tsx:49`) |
| Record | details: ID, request, sequence, watermark, deny revision, failure | `id`, `request`, `receipt.sequence`, `receipt.watermark`, `deny_generation`, `receipt.reason` |
| Under the list | list reference time; paging when there is more than a page | `reference_at`, `next` |
| States | loading, none, failed with Reload and First; the ledger line stays above a failed list | the read's state |

Not displayed from a row: `history`, `target_present`, `target_version`, `job_reason`, `requested_by`, `approved_by`, `deny_at`, `verified`, `completed_at`.

### Effects preview view

Component `Effects` (`access/Live.tsx:263`).

| Area | Displayed field or state | Source |
| --- | --- | --- |
| Person | the person picker; the unconfirmed-revocation sentence under the chosen person | the subjects read; `revocation_unconfirmed` of the chosen row |
| Data kind | choice when there is more than one; a hint, or a different refusal, when the catalog could not be listed in full | distinct `catalog[].kind` of the whole read; `catalog_incomplete` |
| Customer | optional, sent as `customer` | typed |
| States | start hint, cleared after a change, loading, failed; only the latest request is shown | derived |
| Result | title with the person; data kind, customer, checked at; allowed count | the chosen row's `login` or `user_id`; request values; checked at is browser time; `decisions[].allowed` |
| Result table | action, field, allowed or not, why | `decisions[].action`, `field`, `allowed`, `reason`, `delegations[]` |

**Parsed but not displayed on M03:** `you` of the whole read; ledger `replace_reason`, `replaced_file`; the permission detail's `workspace`, `revision` and `row`; every list page's `search_input`, `query`, `observed_at`, `revision`, `order`, `work_units` and `scan_stopped`. The preview read's `revision`, `subject`, `kind` and `customer` are not parsed: `preview()` returns only its `decisions`.

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
| Credential rotation: `replaces`, `replaced_by`, `rotation` on a credential row; `credential_overlap_minutes`, `credential_overlap_max_minutes`, `secret_pickup_minutes` in the limits; `can_rotate`, `rotate_step_up_state` in the credential detail eligibility; `service.rotate_credential` | `serviceCredentialView` and `queryServiceLimits`; `currentCredentialEligibility` (the same facts the command enforces); `serviceCommand.rotate`, which requires a current declaration as issuance does and records its pickup through `servicedelivery.go` | `credentialOf`, `limitsOf`, `credentialDetailOf` (contradictory rotation fields and flags rejected) | `RotateControl` in the credential detail of `CredentialWorkspace` with a review step; `CredentialStatus` shows the rotation facts; `issueExec` resends a held rotation as a rotation; see [integration with main](../../features/HYB-304/main-integration.md) |
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
- **Server fields that no console parser reads:** `impact.api_keys` and `impact.pending_jobs` (`bulkread.go:85, 87`, not in `detailOf`; the preview shows fixed "not applicable" copy instead, `users/BulkPrepare.tsx:73, 77`); trust command result `fields.reason` and `fields.your_session_ended` (`trust.go:315, 321`), `fields.change_revision` (`trust.go:144`); set-role-period result `fields.role_until` (`periods.go:130`); bulk result `fields.digest`, `fields.targets` and `fields.count` (`bulk.go:40, 105`, `bulkstage.go:86`).
- **Parser type wider than the read.** Checked again on 2026-11-27 for two of the three reads this entry named: `access` no longer has row arrays, and its remaining arrays are checked field by field (`service/access-read.ts:4-5`); `me` is parsed by `selfPageOf` under `validatedRead` (`service/admin.ts:1006-1008`). The earlier statement that the service-accounts read cast its row arrays without checking each field was not re-checked in this change.

## HYB-327 reconciliation

The preceding gap inventory records the original mapping review. HYB-327 validates O01 and identity response rows at the read boundary. M02 now displays the fields affected by the restored/default comparison, distinguishing an explicit empty list from unavailable data. M01 selection is already bound by the filter-selection follow-up; API-key and pending-job impact now require explicit server states. See [verification](../../features/HYB-327/verification.md) for executed checks and limitations. Unrelated unread result fields remain outside this change.

## HYB-304 M05 advanced confirmation and authority follow-up

`SessionSelection.tsx` hides a definitively refused initial confirmation synchronously and resets its reference/preview/checkbox for explicit review. A missing submitted reference remains an unknown outcome with the same fingerprint locked in `session-intent`; it is never converted to an empty successful result. Building completion/refusal focuses the stable dialog heading. `admin.ts` and the shared root remove sensitive reads on actual401 or semantic authentication loss; stale prior-session prompts cannot retire a successor session or authorize a retry. See the [L01 shared boundary mapping](./sign-in.md#hyb-304-authenticated-boundary-follow-up).

Large actual result pages and later-arrival exclusion pages were traversed independently with exact counts and no duplicate handles. Pending/failed cleanup render variants use explicitly intercepted responses over a real accepted operation; those images do not claim a real backend cleanup failure. Current verification and retained limitations are in [advanced evidence](../../features/HYB-304/advanced-evidence.json). Parent M04/M03/M02 and owner acceptance remain open.
