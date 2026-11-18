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
| Preview | API keys: not applicable | fixed copy `b.noKeys`; the server's `impact.api_keys` (`bulkread.go:85`) is not read | none | `users/BulkPrepare.tsx:73` |
| Preview | Pending work per store, rechecked or not | `impact.pending[].store`, `count`, `rechecked` (`bulkread.go:70-72, 86`) | `identity-bulk.ts:114` | `users/BulkPrepare.tsx:75-76` |
| Preview | Pending jobs: not applicable | fixed copy `b.noJobs`; the server's `impact.pending_jobs` (`bulkread.go:87`) is not read | none | `users/BulkPrepare.tsx:77` |
| Preview | Service users kept | `impact.service_users` (`bulkread.go:88`) | `identity-bulk.ts:113` | `users/BulkPrepare.tsx:79` |
| Preview | Administrators left afterwards | `impact.anchors_after` (`bulkread.go:89`) | `identity-bulk.ts:113` | `users/BulkPrepare.tsx:81` |
| Preview | Other workspaces | `impact.other_workspaces` (`bulkread.go:90`) | `identity-bulk.ts:113` | `users/BulkPrepare.tsx:82` |
| Preview | Excluded by reason | `exclusions` map, keys `self`, `not_found`, `no_membership`, `already_revoked` (`bulkread.go:98`) | `identity-bulk.ts:106-107` | `users/BulkPrepare.tsx:85-86` |
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
| Impact (open changes) | Signs out everyone | `impact.signs_out_everyone` (`trustrows.go:31`); the parser treats a missing value as true | `admin.ts:536` | `authentication/TrustChanges.tsx:279` |
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

Read: `GET /api/v1/identity/services` (`boundary.go:34`), client and parser `admin.ts:916-930`, called at `service-accounts/Live.tsx:551` for the accounts and credentials views (`:556`). Server shape `servicesPage` (`services.go:47-53`) with `serviceRow` (`services.go:30-39`), `credentialRow` (`services.go:22-28`) and `serviceLimits` (`services.go:41-45`). The service and credential rows are cast without per-field normalisation (`admin.ts:921`). Secret pickup: `GET /api/v1/identity/links/{ref}` (`boundary.go:30`), `admin.ts:931-937`, server answer `secret`, `credential` (`users.go:221-222`).

### Views and page states

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Usage view | Not available in pre-launch, go to accounts | fixed copy | none | `service-accounts/Live.tsx:569` |
| Transfer view | Not available in pre-launch, go to accounts | fixed copy | none | `service-accounts/Live.tsx:570` |
| Both views | Denied | services read refused `permission_denied` | `admin.ts:755` | `service-accounts/Live.tsx:571-573` |
| Both views | Load failed, Try again or Sign in again | failed services read; body from `describeFor("create")` (`service-accounts/Live.tsx:26-53`) | `admin.ts:776-783` | `service-accounts/Live.tsx:574-582, 649, 687` |
| Both views | Loading | derived: no page yet | none | `service-accounts/Live.tsx:648, 686` |
| Both views | Loaded at | `now` (`services.go:52`) | `admin.ts:928` | `service-accounts/Live.tsx:640, 661` |
| Both views | Done notice | derived: command copy | none | `service-accounts/Live.tsx:548, 647, 668` |

### Accounts view

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Header | Count of limit | derived: number of `services[]`; limit `limits.services` (`services.go:44`) | `admin.ts:921, 926` | `service-accounts/Live.tsx:639` |
| Accounts | No services | derived: no non-retired service (`service-accounts/live-model.ts:18-20`) | `admin.ts:921` | `service-accounts/Live.tsx:650` |
| Accounts table | Column headings | fixed copy | none | `service-accounts/Live.tsx:458-463` |
| Accounts table | Service ID | `services[].id` (`services.go:31`) | `admin.ts:921` | `service-accounts/Live.tsx:467` |
| Accounts table | Owner, created by | `owner_login` else `owner`; `creator_login` else `creator` (`services.go:34-37`) | `admin.ts:921` | `service-accounts/Live.tsx:468` |
| Accounts table | Status | `state` (`services.go:32`) via `serviceState` (`service-accounts/live-model.ts:30-32`) | `admin.ts:921` | `service-accounts/Live.tsx:469` |
| Accounts table | Suspended note or active credentials of limit | `state`; count of `credentials[].state = "active"` (`service-accounts/live-model.ts:22-24`); `limits.credentials_per_service` (`services.go:43`) | `admin.ts:921, 925` | `service-accounts/Live.tsx:470` |
| Accounts table | Manage | derived | none | `service-accounts/Live.tsx:471-473` |
| Accounts | Retired disclosure | derived: `state = "retired"` | `admin.ts:921` | `service-accounts/Live.tsx:652-654` |
| Create form | Name with `svc-` prefix, ID preview | typed `name`; preview derived (`service-accounts/live-model.ts:6-10`) | none | `service-accounts/Live.tsx:154-164` |
| Create form | Owner choice, no owners | `owners[].user_id`, `login` (`services.go:50`) | `admin.ts:922` | `service-accounts/Live.tsx:165-176` |
| Create form | Credentials after creation | fixed copy | none | `service-accounts/Live.tsx:177` |
| Create form | Command | `service.create` with `name`, `owner` | `admin.ts:871` | `service-accounts/Live.tsx:133` |
| Create form | Name taken, owner refused | `invalid_request` with `fields` `name`, `owner` | `admin.ts:754, 778` | `service-accounts/Live.tsx:136-144` |
| Create form | Limit reached | `prerequisite_failed` → `refused`; limit `limits.services` | `admin.ts:752, 926` | `service-accounts/Live.tsx:31, 153` |
| Create form | New service opened | derived: `svc-` + typed name; the command result is not read | none | `service-accounts/Live.tsx:134, 605` |
| Detail | Heading | `id` | `admin.ts:921` | `service-accounts/Live.tsx:195` |
| Detail | Status, owner, creator, credentials of limit | `state`, `owner_login`/`owner`, `creator_login`/`creator`, active credential count, `limits.credentials_per_service` | `admin.ts:921, 925` | `service-accounts/Live.tsx:196-201` |
| Detail | Suspend, reinstate, retire buttons; retired note | derived from `state` | `admin.ts:921` | `service-accounts/Live.tsx:191, 202-206` |
| State step | Service, status, owner | `id`, `state`, `owner_login`/`owner` | `admin.ts:921` | `service-accounts/Live.tsx:239, 242` |
| State step | Command | `service.suspend`, `service.reinstate`, `service.retire` | `admin.ts:871` | `service-accounts/Live.tsx:234` |
| State step | Partial retire | `internal_error` with `operation_id` → `partial` | `admin.ts:759, 781` | `service-accounts/Live.tsx:43, 241` |

### Credentials view

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Credentials | No services | derived: no service to choose (`service-accounts/live-model.ts:50-53`) | `admin.ts:921` | `service-accounts/Live.tsx:688-693` |
| Credentials | Service choice with state | `services[].id`, `state` | `admin.ts:921` | `service-accounts/Live.tsx:696-701` |
| Credentials | Pre-launch note: credentials cannot authenticate | fixed copy | none | `service-accounts/Live.tsx:702` |
| Credentials | Active of limit | derived active count; `limits.credentials_per_service` | `admin.ts:925` | `service-accounts/Live.tsx:703` |
| Credentials | None for this service | derived: no active credential (`service-accounts/live-model.ts:26-28`) | `admin.ts:921` | `service-accounts/Live.tsx:704` |
| Credentials table | Column headings | fixed copy | none | `service-accounts/Live.tsx:488-492` |
| Credentials table | Credential ID | `credentials[].id` (`services.go:23`) | `admin.ts:921` | `service-accounts/Live.tsx:498` |
| Credentials table | Valid from, until | `not_before`, `not_after` (`services.go:24-25`) | `admin.ts:921` | `service-accounts/Live.tsx:499` |
| Credentials table | Status: active, on hold, revoke required, revoked, expired | `state`, `usable` (`services.go:26-27`) and service retired via `credentialState` (`service-accounts/live-model.ts:34-40`) | `admin.ts:921` | `service-accounts/Live.tsx:495, 500` |
| Credentials table | Revoke | derived | none | `service-accounts/Live.tsx:501-503` |
| Credentials | Ended disclosure | derived: non-active credentials | `admin.ts:921` | `service-accounts/Live.tsx:706-708` |
| Unstored reminder | Credential and its service, Revoke, Dismiss | derived: credentials issued in this browser session whose secret was not acknowledged (`service-accounts/Live.tsx:108, 536-537, 625`); service looked up in `services[].credentials` | `admin.ts:921` | `service-accounts/Live.tsx:669-685` |
| Unstored reminder | Leave-page warning | derived: reminder list not empty | none | `service-accounts/Live.tsx:562-567` |
| Issue form | Blocked: retired, suspended, limit | derived `issueBlock` from `state` and active count against `limits.credentials_per_service` (`service-accounts/live-model.ts:44-48`); or server `refused` | `admin.ts:752, 921, 925` | `service-accounts/Live.tsx:265, 296-302` |
| Issue form | Valid for days, ends on | typed `days`; maximum `limits.credential_days` (`services.go:42`); end derived from `now` | `admin.ts:924, 928` | `service-accounts/Live.tsx:258, 267, 307-317` |
| Issue form | Command | `service.issue_credential` with `days` | `admin.ts:871` | `service-accounts/Live.tsx:278` |
| Issue form | Held retry after a lost or unclear answer | `dependency_unavailable`, unavailable or `partial` keep the expected revision and attempt (`service-accounts/live-model.ts:55-59`) | `admin.ts:759-760, 829` | `service-accounts/Live.tsx:78-82, 320` |
| Issue form | Issued: credential and secret reference | result `fields.credential`, `fields.secret_ref` (`services.go:367`); malformed values are held as unavailable | `admin.ts:742` | `service-accounts/Live.tsx:280-286` |
| Reveal | Issued notice, service, credential, valid until | `credential` from the result; `not_after` of that credential after reload, else the derived end | `admin.ts:742, 921` | `service-accounts/Live.tsx:382-384, 613` |
| Reveal | Loading | derived while the pickup read runs | none | `service-accounts/Live.tsx:342, 387` |
| Reveal | Secret, shown once | pickup `secret` (`users.go:222`), checked against `^hbsk_` and the credential ID | `admin.ts:933-936` | `service-accounts/Live.tsx:343-346, 388-396` |
| Reveal | Pre-launch note, acknowledgement | fixed copy; local checkbox | none | `service-accounts/Live.tsx:397-401, 371-379` |
| Reveal | Show secret again | first pickup failure `unavailable` | `admin.ts:780, 829` | `service-accounts/Live.tsx:348, 404-409` |
| Reveal | Secret gone | other pickup failure, credential mismatch, no reference, or page hidden | `admin.ts:935` | `service-accounts/Live.tsx:344, 348, 356, 366-370, 410` |
| Reveal | Done, Not stored, Revoke | derived from reveal state | none | `service-accounts/Live.tsx:411-415` |
| Revoke step | Credential, service, status | credential ID; row looked up in `services[].credentials` for `credentialState` | `admin.ts:921` | `service-accounts/Live.tsx:428, 439, 442` |
| Revoke step | Command | `service.revoke_credential` | `admin.ts:871` | `service-accounts/Live.tsx:434` |
| Revoke step | Already revoked | `invalid_stage` → `stage` | `admin.ts:753` | `service-accounts/Live.tsx:33-35, 441` |

**Parsed but not displayed on M04:** service `version`.

## M05 My security

Read: `GET /api/v1/me` (`boundary.go:35`), client and parser `admin.ts:938-956`, called at `my-security/Live.tsx:268` for every view except recovery and notifications (`:278`). Server shape `selfPage` (`self.go:34-49`) with `selfSession` (`self.go:22-27`) and `selfIdentity` (`self.go:29-32`). The `identities` and `sessions` arrays are cast without per-field normalisation (`admin.ts:946, 949`). Calls outside the command path: `POST /api/v1/me/password` (`boundary.go:48`, `admin.ts:957-959`) and `POST /api/v1/me/sessions/end` (`boundary.go:49`, `admin.ts:960-963`, answer `ended`, `self.go:167`).

### Views and page states

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Recovery view | Not available in pre-launch, forgot-password guidance, go to account | fixed copy | none | `my-security/Live.tsx:296` |
| Notifications view | Not available in pre-launch, go to sessions | fixed copy | none | `my-security/Live.tsx:297` |
| All views | Signed out or locked out | `me` read `unauthenticated`, or a password change or end-session answer that signs the person out | `admin.ts:757` | `my-security/Live.tsx:272, 285-295` |
| All views | Loading | derived: no page yet | none | `my-security/Live.tsx:313` |
| All views | Load failed, Try again | any other failed `me` read | `admin.ts:776-783` | `my-security/Live.tsx:314-320` |
| All views | Loaded at | `now` (`self.go:48`) | `admin.ts:954` | `my-security/Live.tsx:303` |
| Sessions view | Done notice | derived: end-session result copy | none | `my-security/Live.tsx:265, 430` |

### Account view

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Account | Sign-in email | `login` (`self.go:36`) | `admin.ts:942` | `my-security/Live.tsx:335` |
| Account | Contact email or not recorded | `email` (`self.go:37`) | `admin.ts:943` | `my-security/Live.tsx:336` |
| Account | User ID | `user_id` (`self.go:35`) | `admin.ts:941` | `my-security/Live.tsx:337` |
| Account | Password set or not | `password_set` (`self.go:41`) | `admin.ts:947` | `my-security/Live.tsx:338` |
| Account | Profile changes not available note | fixed copy | none | `my-security/Live.tsx:340` |
| Access | Workspace | `workspace` (`self.go:38`) | `admin.ts:944` | `my-security/Live.tsx:459` |
| Access | Role | `role` (`self.go:39`) via `roleName` (`users/live-model.ts:14-16`) | `admin.ts:945` | `my-security/Live.tsx:460` |
| Access | Installation administrator | `installation_administrator` (`self.go:42`) | `admin.ts:948` | `my-security/Live.tsx:461` |
| Access | Role note | fixed copy | none | `my-security/Live.tsx:463` |

### Sign-in methods view and password change

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Methods table | Column headings | fixed copy | none | `my-security/Live.tsx:372-376` |
| Methods table | Method | fixed copy `m.password`; `identities[].method` (`self.go:30`, always `local-password` at `self.go:103`) is used only as a row key | `admin.ts:946` | `my-security/Live.tsx:379-380` |
| Methods table | Signs in as | `identities[].login` (`self.go:31`) | `admin.ts:946` | `my-security/Live.tsx:381` |
| Methods table | Status | `password_set` | `admin.ts:947` | `my-security/Live.tsx:382` |
| Methods | Linking other methods not available note | fixed copy | none | `my-security/Live.tsx:388` |
| Password form | No password to change | `password_set` false (`my-security/live-model.ts:12-14`) | `admin.ts:947` | `my-security/Live.tsx:114` |
| Password form | Current, new, confirm | typed; sent as `current`, `new` | none | `my-security/Live.tsx:133-136`, call `:89` |
| Password form | Local checks: missing, rule, same, mismatch | derived (`my-security/live-model.ts:6-10`; `passwordProblem` with `login`) | `admin.ts:942` | `my-security/Live.tsx:73-81, 131` |
| Password form | Wrong current password | `current_password_wrong` → `wrong` | `admin.ts:763` | `my-security/Live.tsx:97-100` |
| Password form | New password refused by the rule | `password_form` → `form` | `admin.ts:765` | `my-security/Live.tsx:101-103` |
| Password form | Locked, limited, conflict, busy, result unknown | `password_locked`, `rate_limited`, `revision_conflict`, `dependency_unavailable`, other (`my-security/live-model.ts:22-38`) | `admin.ts:751, 758, 760, 764` | `my-security/Live.tsx:104-107, 132` |
| Password form | Changed | successful answer | `admin.ts:957-959` | `my-security/Live.tsx:115-125` |

### Sessions view

| View/area | Displayed field, column or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Header | Session count | derived: number of `sessions[]` | `admin.ts:949` | `my-security/Live.tsx:428` |
| Header | End all other sessions | derived: at least one session with `current` false (`my-security/live-model.ts:16-18`) | `admin.ts:949` | `my-security/Live.tsx:429` |
| Sessions | Stale-list note with idle minutes | `idle_minutes` (`self.go:44`) | `admin.ts:950` | `my-security/Live.tsx:434` |
| Sessions table | Column headings | fixed copy | none | `my-security/Live.tsx:204-209` |
| Sessions table | This or other session, reference | `sessions[].current`, `handle` (`self.go:23-24`) | `admin.ts:949` | `my-security/Live.tsx:213-214` |
| Sessions table | Signed in | `sessions[].signed_in` (`self.go:25`) | `admin.ts:949` | `my-security/Live.tsx:215` |
| Sessions table | Ends | `sessions[].ends` (`self.go:26`) | `admin.ts:949` | `my-security/Live.tsx:216` |
| Sessions table | Sign out here, or End | derived from `current` | `admin.ts:949` | `my-security/Live.tsx:217-221` |
| Sessions | No other sessions | derived | none | `my-security/Live.tsx:436` |
| Limits | Idle limit | `idle_minutes` | `admin.ts:950` | `my-security/Live.tsx:413` |
| Limits | Longest session | fixed copy `m.longestValue`, not from the read | none | `my-security/Live.tsx:414` |
| Limits | Maximum sign-in age | `max_age_seconds` (`self.go:45`); none when zero | `admin.ts:951` | `my-security/Live.tsx:415` |
| Limits | You signed in, last confirmed | `signed_in_at`, `authenticated_at` (`self.go:46-47`) | `admin.ts:952-953` | `my-security/Live.tsx:416-417` |
| Limits | Recent access history not available note | fixed copy | none | `my-security/Live.tsx:419` |
| End step | Reference, signed in, ends (one session) | `handle`, `signed_in`, `ends` of the chosen session | `admin.ts:949` | `my-security/Live.tsx:165, 189` |
| End step | Impact (one, or all others with count) | derived count of other sessions | none | `my-security/Live.tsx:166, 190` |
| End step | Call | `POST /api/v1/me/sessions/end` with `session` = handle or `others` | `admin.ts:960-963` | `my-security/Live.tsx:172` |
| End step | Ended N | answer `ended` (`self.go:167`) | `admin.ts:962` | `my-security/Live.tsx:173` |
| End step | Session already gone | `invalid_request` → `invalid` (`my-security/live-model.ts:41-49`) | `admin.ts:754` | `my-security/Live.tsx:177` |
| End step | Limited, step-up needed, result unknown | `rate_limited`, `reauthentication_required` or cancelled, other | `admin.ts:758, 766, 842` | `my-security/Live.tsx:178, 188` |

**Parsed but not displayed on M05:** none found beyond `identities[].method`.

## Unknowns and mismatches

- **Unknown.** None of the rows above is marked unknown.
- **Server fields that no console parser reads:** `trust.stricter_fields` (`policy.go:69`, not in `policyPageOf`, `admin.ts:568-573`); `jobs[].selection` and `job.selection` (`bulkread.go:39`, not in `jobOf`, `identity-bulk.ts:83-96`); `impact.api_keys` and `impact.pending_jobs` (`bulkread.go:85, 87`, not in `detailOf`; the preview shows fixed "not applicable" copy instead, `users/BulkPrepare.tsx:73, 77`); `revocations[].target_version` (`revocations.go:35`, not in the `Revocation` type, `admin.ts:167-190`, although the spread at `admin.ts:900-901` keeps it at run time); trust command result `fields.reason` and `fields.your_session_ended` (`trust.go:314, 320`), `fields.change_revision` (`trust.go:143`); set-role-period result `fields.role_until` (`periods.go:130`); bulk result `fields.digest`, `fields.targets` and `fields.count` (`bulk.go:40, 105`, `bulkstage.go:86`).
- **Parser type wider than the read.** `access`, `services` and `me` cast their row arrays without checking each field (`admin.ts:891-894, 921, 946, 949`), unlike `userRowOf`, `trustChangeOf` and the bulk parsers. A missing field arrives as `undefined` rather than an empty string or zero.
