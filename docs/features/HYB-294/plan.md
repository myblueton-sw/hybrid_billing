---
type: ImplementationPlan
title: HYB-294 M01 server search, filters and filter-wide selection
description: Fix ticket for finding H3 of the HYB-291 planning cross-check. The M01 users read gains server search, the spec filters, a stable login order, cursor paging with a scan budget and truthful counts, on indexed identity listings; the screen binds them in a filter band with chips and link parameters; filter-wide selection resolves a filter snapshot on the server into the HYB-299 staging draft. Delivered as two ordered parts. Revised after design review rounds 1 and 2.
status: stable
tags: [HYB-294, M01, identity, search, filters, bulk-revocation, IAM-07, UI-04, HB-03, HB-42, HYB-291, HYB-299, HYB-300]
sources:
  - resource: ../../ui/screens/M01/spec.md
  - resource: ../../planning/v1.16/retention-access.md
  - resource: ../../planning/v1.16/ui-design.md
  - resource: ../../planning/v1.16/requirements.md
  - resource: ../HB-263/plan.md
  - resource: ../HB-264/verification.md
  - resource: ../HYB-299/plan.md
  - resource: ../HYB-299/verification.md
  - resource: ../HYB-299/references.md
  - resource: ../HYB-300/plan.md
  - resource: ../HYB-300/verification.md
  - resource: ../../reviews/HYB-291-planning-cross-check/report.md
  - resource: ../../reviews/HYB-301-trace-classification/report.md
---

# HYB-294 M01 server search, filters and filter-wide selection

## Requirement

- **Ticket.** HYB-291 finding H3: M01 has no server search or filters (`docs/reviews/HYB-291-planning-cross-check/report.md:50`, `:117`, `:145`). The users read only pages (`HB-263/plan.md:104-105`), and the deferral is recorded against IAM-07 (`HB-264/verification.md:176`).
- **Spec.** The list order is user ID, name, login realm, membership, role period, status, revocation effect (`docs/ui/screens/M01/spec.md:59`). The filters are workspace, status, authentication method, role and expiry (`:61`). The spec asks for server search and a stable sort, with selection kept by canonical ID across name changes (`:63`). Tab changes keep the filters and the selection (`:81`). Large lists:
  - reads use a fixed snapshot or revision and a cursor, and never load everything into the browser;
  - page selection and filter-wide selection are separate, and filter-wide selection fixes a server target manifest, count and permitted scope;
  - a change in targets is shown as a new snapshot and never silently updates an approved scope (`:104`).
- **Scope header.** The installation and workspace are fixed at the top of the screen. The server checks scope, and applies it the same way to detail, search, totals and downloads (`:25`).
- **Planning.**
  - `retention-access.md:65`: the screen offers user search, status, membership, role, authentication method and sessions. 100,000 users are managed with server search, a cursor and a fixed selection manifest.
  - `retention-access.md:116` (IAM-07): lookup, invitation, sync and revocation of 100,000 users stay isolated and resumable and are checked against the blocking revision.
  - `ui-design.md:547` (UI-04): with 100,000 users, the filter, selection, page and total bases agree, and nothing is loaded whole into the browser.
  - The large-table contract is `ui-design.md:412-424`:
    - a stable ID as the tie-breaker;
    - no unsupported sorts;
    - filters shown as chips and text, with a single clear-all;
    - the reference time of a state list;
    - cursor expiry;
    - page totals and selection totals kept separate;
    - filter-wide selection as a server manifest of targets, exclusions and base revision, so the client never carries 100,000 IDs;
    - a change in filters never silently widens a selection.
  - The wireframes are W03 (`ui-design.md:270-292`, "필터 결과 100,000개 전체 선택") and W07 (`ui-design.md:366-379`).
- **IDs.** IAM-07 and UI-04 are not in `requirements.md`. They are defined at `retention-access.md:116` and `ui-design.md:547` and traced in `acceptance-trace.json:1947`, `:2064`. HB-03 (`requirements.md:53`) applies to the server-side scope.
- **Root notes for this ticket:**
  - reuse the HYB-299 staging transport for filter-wide selection (`HYB-299/plan.md:389-406`, `:1173-1179`; residual R1 `:1251`);
  - revisit the phase-B lock hold time (`HYB-299/plan.md:1177`; numbers in `HYB-299/verification.md:148`);
  - reuse HYB-300 Part A (`HYB-300/plan.md:122-123`, `:209-256`).

## Size and proposed split

The change is too large for one reviewable PR. It touches identity listings, the users read, a new bulk command, a digest change, settings and the console. Proposed order:

1. **Part A: indexed listings, server search, filters and paging, and the console filter band.** This part has no bulk change. It closes `M01/spec.md:61-63` and the read half of IAM-07 and UI-04.
2. **Part B: filter-wide selection.** Part B depends on part A's filter evaluator. It covers:
   - `bulk_revocation.add_matching` on the HYB-299 draft;
   - a filter snapshot in the manifest and digest;
   - the filter-wide limit setting;
   - the phase-B measurement at 100,000;
   - the selection bar in the console.

   It closes `M01/spec.md:104` and HYB-299 R1.

Each part has its own design review, change review, QA and PR under HYB-294. If the owner prefers sub-issues, part B becomes a new ticket. The Linear MCP returned 401 in this session, so sub-issues are not created here.

## Ownership

- Root: PM, planner and integration. Root owns implementation and the decisions below.
- Independent read-only reviewers:
  - architect or security review for tenant scope, the new command and the digest change (`design-first.md`: permission and public-contract changes);
  - a DBA-style review of the index and paging correctness;
  - the ui-designer render review;
  - TW review of copy in EN, KO and AR (`product-terminology.md`);
  - QA review of the browser record.
- One writer per file. Part B starts after part A merges.

## Current behaviour (facts, `e360bf7`)

**Read**
- `GET /api/v1/identity/users` takes only `after` and `limit` (1–100, default 50) (`apps/api/server/users.go:111-125`). It runs under `s.domain.RLock` and requires `administer_installation` (`:127-147`, `:82-109`).
- `usersPage` first builds a map of every membership (`roster`, `admin.go:130-151`). It then pages `Installation.Users` 100 at a time, skipping service users, until `limit` humans are collected (`users.go:149-181`). The order is by user ID. User IDs are random: `"user-" + hex(random(8))` (`admin.go:247`).
- **Cost.**
  - `Installation.Users` and `Workspace.Members` copy and sort every ID on every call (`modules/identity/installation.go:153-172`; `workspace.go:258-284`; `page`, `registry.go:392-397`).
  - `roster` therefore sorts N IDs N/100 times per request.
  - `everyone()` repeats this for users (`bulkimpact.go:124-143`) and runs inside `prepare` and `execute` under the domain write lock (`bulk.go:20`, `:81`).
  - At 100,000 users this is about 1,000 full sorts per call. **This is not measured.** Part A measures it first (baseline benchmark) before changing it.
- The login and email of a person live in `apps/api`, not in identity (`directory.go:16-19`, `:41-43`). The directory holds humans only: `dir.add` is called at `admin.go:251`, `seed.go:101`, `:172`, `:184` and `signin.go:473`. `userOf(login)` is a linear scan (`directory.go:66-73`).
- **Departure removes the directory entry** (review H1). `departUser` deletes the person from `s.dir.people`, `holders`, `invites` and `links` (`admin.go:325-342`, `:337`). The identity user stays with state `departed`. A departed row therefore has no login, which follows the profile-minimisation rule (`retention-access.md:76`: 인증 비밀·불필요 프로필 제거).
- Users are never deleted from the identity registry. There is no `delete(r.users…)`, and users are inserted only at `boundary.go:26`, `installation.go:43` and `workspace.go:307`. A removed membership stays in the map with state `removed` (`registry.go:276-282`). Memberships are inserted only at `workspace.go:133`.

**Scope**
- There is one workspace per installation: `s.bootWorkspace` is set at start (`seed.go:89`), and no command creates a workspace.
- The users read is installation-level. It lists every human user and shows membership in the caller's workspace only.

**Authentication methods**
- Every human is linked through `LoginTriple`, with connection `local-password` (`signin.go:89-91`; `seed.go:30`).
- `authpolicy` names `ldap`, `oidc`, `password` and `saml` (`modules/authpolicy/types.go:23-26`), but apps/api only evaluates `MethodPassword` (`commands.go:244`, `signin.go:272`).
- SAML, OIDC and LDAP sign-in are not built (`HB-264/verification.md:171`).
- What varies per person is the sign-in setup:
  - password set (`users.go:191`);
  - a live set-password link or reset link (`:195-197`; `admin.go:255`, `:287`, `:298`);
  - a pending invitation (`:192-194`).

**Console**
- M01 loads `api.users("", 100)`, adds pages with "Show more users" (`Live.tsx:509`, `:529`), and keeps every loaded row in memory.
- The Invitations tab filters the loaded rows on the client (`rowsFor`, `Live.tsx:353-355`). This is the client-side filtering that UI-04 rules out.
- The selection bar says filter-wide selection is not available (`bulk-copy.ts:20-21`; `Selection.tsx`).
- M01 receives no link parameters. Tab changes keep the query string (`app/screen-groups/identity.tsx:51-60`). O01 receives `params` and `onParams` through `changeProps` (`app/screen-groups/work.tsx:25-39`).

**Bulk (HYB-299)**
- Drafts are explicit only (`Selection: Explicit`, `modules/bulkrevoke/registry.go:120`; `types.go:69`).
- `Add` bounds a draft by `MaxTargets` (`registry.go:99-145`). The digest covers effect, selection, targets and exclusions (`manifest.go:31-43`).
- `-bulk-revocation-max-targets` defaults to 500, bounded 1–5,000. The bound comes from staging batches inside the command burst (`HYB-299/plan.md:810`).

**Phase B**
- Execute runs under the domain write lock (`commands.go:209-211`).
- Phase A suspends in one identity call (`bulk.go:97`). Phase B writes a header, then item records in chunks of 256 (`ledgerChunk`, `bulk.go:12`, `:126-174`; `decisionledger.BatchLimit = 256`, `RecordLimit = 2048`, `modules/decisionledger/types.go:10`, `:15`).
- HYB-299 recorded these engine-level numbers (`HYB-299/verification.md:148`):
  - identity deny of 100,000 memberships: about 65 ms;
  - ledger header plus 100,000 items: about 2.5 s, with 392 syncs;
  - stage, prepare and execute of the manifest: about 69 ms.

  The HTTP path was tested only at 5,000. `server.WriteTimeout` is 30 s (`apps/api/cmd/hb-api/main.go:77`). The ledger defaults to 256 MiB (`settings.go:60`).
- Every command first runs `sweepRoles`, which scans every membership of the workspace (`commands.go:212`; `modules/identity/periods.go:60-74`). This is O(N) per command and not measured at 100,000.

## Decisions

### D1. Tenant scope and the workspace filter

- The read stays installation-level, with the caller's workspace fixed. The server takes the workspace only from `answer.Workspace()`, as today, and never from the query (`tenant-security.md`: a client-sent tenant ID is not trusted).
- **The `workspace` query parameter is not accepted.** A request carrying it gets `invalid_request`, whatever the value, so a probe cannot learn whether another workspace exists.
- The screen shows the workspace as fixed scope text under the title, not as a control (`M01/spec.md:25`).
- The spec's "업무 공간" filter is realized as **membership in this workspace** (the `membership` filter below). It covers people who have a membership and people who don't. With one workspace per installation this is the full meaning that data supports.
- A workspace selector across several workspaces, and an AM view limited to one workspace's members, need several workspaces and role-scoped grants. Neither exists (the same residual as `HYB-300/plan.md:133-135`). They are recorded under Residuals, not built.
- **Two workspace sources, asserted equal** (review L1). The read's scope is `answer.Workspace()` (`users.go:98-99`). Membership data comes from `s.bootWorkspace` (`admin.go:138`; `bulkstage.go:33`). The handler checks `answer.Workspace() == s.bootWorkspace` and answers `permission_denied` otherwise, so the two can never diverge silently. A test sets `s.bootWorkspace = "elsewhere"`, as the existing API tests do (`installation_api_test.go:222`).
- **Permission** (review L3). `administer_installation` is the scoped right for search, the same as for the list today (`users.go:98-107`). No separate search right exists, and none is added.
- Negative tests:
  - a request with `workspace=` of the caller's own ID, of another ID, and of a malformed ID: all `invalid_request`;
  - two installations' rows never mix (a direct call, as in HB-272);
  - a member session is refused;
  - no session is refused.

### D2. Filter set and semantics

Every filter is optional. Filters combine with AND. Values within one filter are single-choice, which keeps the chips simple. Every filter is evaluated on the server, on the same data the row shows.

| Parameter | Values | Meaning (evaluated at `server_time`) | Spec |
| --- | --- | --- | --- |
| `q` | 1–64 bytes, trimmed | Case-insensitive substring of the login, the contact email or the user ID. Matching uses simple Unicode case folding (`strings.EqualFold`-compatible folding of both sides). A departed person has no login or email (H1), so only the user ID can match. | `:63` server search; `retention-access.md:65` |
| `status` | `active`, `suspended`, `departed` | Account (user) state | `:54`, `:61` "상태" |
| `membership` | `active`, `suspended`, `ended`, `none` | Membership in this workspace. The four values partition every row exactly once (review M1):<br>- `active`: the HYB-299 **effective** predicate (state active and the period not ended);<br>- `suspended`: state suspended, for any reason, including a role period ended and already swept (`suspended_for = role_period_ended`);<br>- `ended`: state active, but the role period has ended and the sweep has not yet run, so the membership is not effective;<br>- `none`: no membership, or a removed one.<br>A property test asserts that each generated row matches exactly one value. | `:51`, `:54`, `:61` "업무 공간" |
| `role` | `administrator`, `member` | Workspace role of a membership in any state; people with `membership=none` never match | `:61` "역할" |
| `period` | `open`, `bounded`, `ends_soon`, `ended` | The row's existing `role_period` value (`users.go:64-75`). `ends_soon` uses the installation setting `role_period_warning_days` (14), which the chip states ("ends within 14 days"). `bounded` means it has an end that is not soon. | `:52`, `:61` "만료" |
| `signin` | `password_set`, `set_link`, `reset_link`, `not_set` | Password present; a live set-password link; a live reset link; no password and no live link | `:61` "인증 방식" (see D3) |
| `invitation` | `pending` | A live invitation or a live set-password link. This is exactly today's `rowsFor` rule, moved to the server. | Invitations tab |
| `holder` | `true` | Holds a live installation grant (`dir.holds`) | the existing column |
| `method` | any | Refused with `unsupported_capability` (D3) | `:61` |

- **Validation.** Any unknown parameter, unknown value, repeated parameter or empty value gets `invalid_request`. No partial application.
- **Response additions:**
  - `filters`, which echoes the applied values;
  - `scan_stopped`, `scan_budget`, `order` (`login`, then departed by user ID) and `list_version` (D4);
  - `auth_methods`: `{connected: ["password"], not_connected: ["saml","oidc","ldap"]}`;
  - `warning_days`.

  The existing fields are unchanged. The key allowlist test is extended. There is no session data and there are no tokens (`M01/spec.md:33`).

### D3. Authentication method

- **Fact:** every human signs in with an installation password. SAML, OIDC and LDAP are not connected (Current behaviour).
- A method filter would match everyone or no one. Offering it would be a control that does nothing (`usability.md`).
- Following the HYB-300 provider precedent (`HYB-300/plan.md:124-127`):
  - `method` is refused with `unsupported_capability`;
  - the read reports `auth_methods`;
  - the band shows a line of text instead of a control.
- What does vary is sign-in setup, and the spec separates authentication status from identity and membership (`:33`). The `signin` filter covers it.
- When a second connection exists, `method` becomes a filter on the identity triple's connection. The triple is already stored (`types.go:120-124`). That is recorded under Residuals.

### D4. Order, cursor, scan budget and truthful counts

- **Order: two segments** (review H1, root decision).
  1. **People with a directory entry**, by login ascending in byte order.
     - Logins are unique per installation (`directory.go:57-58`, `taken`), and no command renames one.
     - The user ID is still the explicit tie-breaker in the comparison (`ui-design.md:416`). It never decides today, because logins are unique, but it keeps the order total if that ever changes.
  2. **Departed people**, after every person with a login, by user ID ascending.
     - Departure deletes the login (`admin.go:337`). The plan does **not** keep a departed person's login only for ordering, because that would keep an unnecessary profile field (`retention-access.md:76`).
     - Owner question Q9 carries this default.
  - Order by user ID only is not offered, because random IDs are no order for a person. Other sorts are not offered (`ui-design.md:416`).
- **Index** (review L5). `apps/api` keeps, in the directory:
  - a sorted slice of user IDs in login order (`dir.order`);
  - a sorted slice of departed user IDs (`dir.departed`);
  - a `byLogin` map, which replaces the linear `userOf`.

  These are updated **in place** by binary-search insert and delete: O(N) copy per change, no dirty rebuild, so a read never sees a stale index.
  - `dir.add` inserts into `order`.
  - `departUser` moves the ID from `order` to `departed`, at the same place as the existing deletes (`admin.go:333-338`).
    - It inserts into `departed` **regardless of `c.known`** (review N4). Today the directory deletes run only when `c.known` (`admin.go:333`), and a departed identity human must still appear in the departed segment.
    - A test departs a person with no directory entry.
  - The directory changes only under the domain write lock. Reads hold `RLock`.
- **Directory field access** (review L7). `people` becomes reachable only through `dir.add` and `dir.remove`, and `departUser` calls `remove`. A source test fails if any file other than `directory.go` writes the unexported `people`, `order`, `departed` or `byLogin` fields.
  - The same test covers `invites`, `links` and `holders`, and the `s.passwords.Put` and `Remove` call sites (review N5). The other password-state mutators found in `apps/api/server` (`Commit`, `Issue`, `Release`, `Reserve`) are checked at implementation and added to the test if they change what `password_set` or a link reports. Every one of them must also bump the directory version, so the list version cannot miss a change a filter reads.
- **Cursor.**
  - `after` is the user ID of the last examined person, never a login. That keeps personal data out of query strings and logs.
  - The server finds the cursor person's segment and position: the login through `byLogin`, then `sort.Search` in `order`, or `sort.Search` in `departed`.
  - So that the server can tell which segment a cursor belongs to, the cursor carries a segment prefix (`a:` or `d:`) before the user ID. It is still opaque to the console.
  - **List changed, start again** (review N2, N3). This answer is given in exactly one case: the cursor was issued in the login segment (`a:`) and its person has departed since.
    - The read answers `revision_conflict` (HTTP 409, next action "inspect") with field `after`.
    - It goes through a new read-side error writer that can carry fields. The existing `failCode` carries none.
    - The answer is recorded in `openapi.go`, and a test asserts its status, code, next action and field.
    - The console keeps the loaded rows and shows "The list changed while you were paging. Start again from the first page." with "Start again", which reloads page one with the same filters.
  - A malformed `after`, an unknown user ID, a non-human ID, or a `d:` cursor whose person is not departed gets `invalid_request` (N3).
  - **No duplicate rows in the console** (N3). The console merges each new page into its rows **by user ID**, replacing an existing row. A person who departs while the user pages, and reappears in the departed segment, is shown once. This is a console guarantee, covered by a console test and an e2e step, not a server guarantee.
- **Scan budget.** This is the HYB-300 pattern (`HYB-300/plan.md:122`).
  - A request examines at most `user_filter_scan` people, a new installation setting: default 1000, bounds 100–10,000.
  - When the budget runs out, the page can be shorter than `limit`. `next` is then the last examined user ID and `scan_stopped` is true.
  - When the list ends within the budget, `scan_stopped` is false and `next` is empty, even if the budget was used exactly.
  - Unfiltered pages stop at `limit`.
- **Identity data per page.** Candidates are fetched in chunks of up to 256 IDs through one new identity call per chunk (D5). The cost is O(budget + log N) per request.
- **No snapshot.** The users list is a state list, so it shows its reference time (`ui-design.md:417`).
  - Each page returns `server_time` and a **users list version** (`list_version`, review M5). This replaces the global `revision`, which changes with every command of any screen.
    - The list version is the identity registry's `r.generation` plus the directory version (review N5). The per-index counters of D5 serve only the index slices.
    - The directory version is bumped by `add`, `remove` and every change to a field a filter reads: password set or removed, link issued or used, invitation, holder.
  - When a later page's `list_version` differs from the first page's, the console keeps the rows and shows "The list changed after the first page. Reload for a consistent list. Changes that come only from time passing, such as a role period ending, are not detected."
  - It does not merge silently.
  - Rows are evaluated at their own read, and permission is checked on every page (`ui-design.md:418`).
- **Counts never claim a total** (`usability.md`; HYB-300 copy amendment 7, `HYB-300/plan.md:731`):
  - "{n} shown; more may exist";
  - "{n} shown; end of list";
  - after a scan stop, "Searched up to {login}." with "Continue searching";
  - for a departed cursor, "Searched up to user {id}.".
- **No count in Part A.** The list read never gives a total. The bounded count before selecting belongs to filter-wide selection and ships in Part B (D7; review N1).

### D5. Identity listings (modules/identity, additive, read-only)

- **Sorted ID indexes inside the registry:**
  - one for users;
  - one per workspace for membership keys.
- **One write path per index** (review M2). Every user insert goes through a single helper, `r.putUser`:
  - Human creation at `boundary.go:26` (bootstrap) and `installation.go:43`;
  - the service user at `workspace.go:307`;
  - any future delete, through `r.dropUser`.

  Every membership insert goes through `r.putMembership` (`workspace.go:133`).
  - Each helper updates its sorted slice in place and bumps a **per-index version counter**, `usersVersion` or `membersVersion[workspace]`. It is deliberately not `r.generation`, which changes on every access-relevant mutation (`registry.go:114`).
  - Each in-place state change that a listing reports (user state, membership state, role, period) also bumps the listing version, without touching the slice.
  - A source test fails if `r.users[...] =` or `r.memberships[...][...] =` appears outside the helpers. Tests check index correctness after Bootstrap, CreateUser, CreateServiceUser and Accept.
- **Paging.** `Users` and `Members` keep their signatures and results, but page by `sort.Search` on the index instead of sorting a copy. `Members` skips removed memberships while it fills the page. The existing tests prove the results are unchanged.
- **New `(*Workspace) People(actor Verifier, workspace string, ids []string) ([]Person, error)`** (review L2: the verifier form, like `Workspace.Members`, `workspace.go:258`).
  - It requires an administrator verifier for `workspace`, under one lock.
  - It takes at most `PageSize×4` IDs (256).
  - For each known ID it returns the `UserInfo` and, when the membership in `workspace` exists and is not removed, the `MemberInfo`, with `Effective` computed at the registry clock. Unknown IDs are absent.
  - It answers `ErrInvalid` for a bad workspace, a bad ID or too many IDs.
- The identity part of `list_version` is the existing `Subjects().Generation()` (`modules/identity/registry.go:216`; `bulk.go:100`), so no new function is needed (M5, N5). The per-index counters are internal to the index slices and are not exposed.
- **Read-side costs fixed together** (review M3):
  - `roster()` and `everyone()` use the indexed `Members` and `Users`.
  - `findUser` (`admin.go:111-128`) uses a new indexed point lookup instead of paging every user.
  - The users read takes the latest bulk outcome only for the page's IDs. A new `bulkrevoke.Registry.LatestFor(workspace, ids)` replaces the full `Latest` map (`users.go:156`; `modules/bulkrevoke/job.go:182`).
  - Prepare's impact computation indexes sessions, pickups and pending proposals by user **once per prepare**. Today it may scan them per target (`bulkimpact.go`).
  - The costs of copying a `bulkrevoke` View (`copyView` of every target on `Get` and `Visible`) are recorded in the part B scale test, with no change unless the numbers call for one.

### D6. How the tabs use the filters (`M01/spec.md:79`, `:81`)

| Tab | Server request | Band |
| --- | --- | --- |
| Users | All filters from the link | Full band |
| Membership | All filters. Same rows, membership columns | Full band. A role or period filter implies membership; the chip text says so. |
| Invitations | All filters plus a fixed `invitation=pending` | Full band. A fixed, non-removable chip reads "Pending invitations". |
| Revocation jobs | No users read is affected | The band is not shown. One line reads "Filters apply to Users, Pending invitations and Workspace membership." The link keeps them. |

- Filters live in link parameters and survive tab changes: `q`, `st` (status), `ms` (membership), `role`, `per` (period), `si` (sign-in), `adm` (`1`).
- An invalid parameter is dropped, with the plural bad-link notice from HYB-300 (`HYB-300/plan.md:729`).
- `app/screen-groups/identity.tsx` passes `params` and `onParams` as `work.tsx` does. `onView` already keeps the query.
- **Explicit selection across filter changes.** It is kept by user ID (`HYB-299/plan.md:383-384`). A filter change never adds anyone. Selected people outside the new result are listed under the existing "not in the loaded rows" disclosure (`Selection.tsx`), so the selection never widens silently (`ui-design.md:422`). The console can tell only "not loaded", so it uses that text and never claims a filter mismatch.
- **The `q` value in links** (review L3, N6). The facts:
  - search text may appear in the console link, as in O01, so a link can carry part of a login;
  - the API receives `q` in the query string of the read;
  - the server does not log request URLs (no logging call in `apps/api/server` uses `r.URL`, repository search on `e360bf7`), and its log lines carry IDs, counts and codes (`HYB-299/plan.md:798`).

  Q8 asks whether this is acceptable.

### D7. Filter-wide selection (part B) through the HYB-299 staging transport

This reuses HYB-299's transport and adds no route (`HYB-299/plan.md:406`, `:1176`).

- **New command `bulk_revocation.add_matching`.**
  - Target `bulk-revocations`, the same definition pattern as `add_targets` (`bulkstage.go:98-110`).
  - Permission: `revoke_access_in_bulk`, plus `administer_installation` in the handler. Mutating, with a rationale. Step-up comes from HYB-293.
  - Fields: the D2 filter fields as optional `text` or `enum` fields, validated by the same function as the read. `method` is refused with `unsupported_capability`.
  - It uses the same filter evaluator as the read (`userfilter.go`), so a person matches the selection only if they would match in the list.
- **Behaviour, under the domain write lock:**
  1. It evaluates the filter over **all** people in login order. There is no scan budget: a command must fix the complete set (`M01/spec.md:104`).
  2. It refuses with `prerequisite_failed` when nobody matches (`ErrEmpty`), or when the match count exceeds `-bulk-revocation-max-matching`.
     - The count is not returned on refusal: refusal fields are copied only for `invalid_request` (`HYB-300/plan.md:689`).
     - The console says "More people match than one revocation list allows ({limit}). Narrow the filters."
  3. It creates a **new** draft owned by the caller, with `Selection = filter`. It stores the canonical filter, `Matched`, the domain revision and `FixedAt` (`server_time`).
     - The canonical filter is a **length-prefixed encoding** (review L6): for each key in a fixed order, `len(key):key len(value):value`, with absent filters omitted. No value can forge a separator.
     - The one-open-draft and pending limits apply unchanged (`HYB-299/plan.md:402`).
     - An existing explicit draft must be discarded first.
     - `add_targets` on a filter draft is refused with `invalid_stage`. No mixing, so the selection basis stays one thing.
  4. Answer: `completed`, state `draft`, fields `draft`, `count`, `matched`, `filter`, `fixed_at`.
- **Prepare re-checks the filter** (review H2, root decision). Between `add_matching` and `prepare`, people change.
  - At prepare, the stored filter is evaluated again at `server_time` for every draft member, with the same evaluator.
  - Members who no longer match are **excluded with the new reason `no_longer_matches`**. They are counted with the other exclusions, listed in the owner's exclusions read, and shown in the preview: "{n} no longer match the filters and are not in the list."
  - Members who still match are classified as before: self, not-found, no-membership and already-revoked exclusions.
  - People who start matching later are still **not** added, because the draft never widens. The preview says so, and "Select again" makes a new list (`M01/spec.md:104`).
- **Execute and discard are unchanged.** Anchor checks, holds for targets changed since preview, and the owner-only rules are kept.
- **A filter list that would leave no anchor administrator** (review M6). Prepare refuses it with `prerequisite_failed`, as for an explicit list (`bulk.go:33-35`; `HYB-299/plan.md:433`).
  - An explicit list shows the selected administrators to deselect. A filter list cannot deselect (Q4), so the blocked copy says: "This list would leave no administrator who can manage access. Narrow the filters, for example Workspace role: Member." Superseded in part by [HYB-324](../HYB-324/implementation.md): the owner of a filter list can now remove chosen people, in a removal mode that brings the checkbox column back for that purpose only; the dead-end names removal as a remedy and offers a way back to the list, still without a list of administrators.
  - KO: "이 목록으로는 접근을 관리할 관리자가 남지 않습니다. 필터를 좁히세요. 예: 업무 공간 역할: 구성원"
  - AR: "لن يبقى بهذه القائمة أي مسؤول يمكنه إدارة الوصول. ضيّق عوامل التصفية، مثل: الدور في مساحة العمل: عضو"
  - A Go test and a QA step cover it.
- **Manifest and digest.**
  - `View` gains `Filter`, `Matched` and `FixedAt`.
  - `Prepare` today hard-codes `Explicit` in the digest (`modules/bulkrevoke/manifest.go:87`). It changes to use the draft's own selection, plus `Filter`, `Matched` and `FixedAt` when the selection is `filter` (review L6).
  - Explicit digests are byte-for-byte unchanged, which a pinned vector test proves.
  - **Equal-manifest reuse** (`manifest.go:88-92`) compares digests. Because `FixedAt` is in the filter digest, two separate fixes of the same filter never collapse into one manifest. That is intended: each fix is its own snapshot.
  - The ledger header is unchanged. Its `RequestDigest` covers the filter, so there is **no ledger format change**.
  - A job rebuilt by replay shows "Selection basis not recorded" for filter lists. Today replay keeps no selection either.
- **Reads.**
  - Bulk job rows gain `selection`, `matched`, `fixed_at` and `filter`.
    - The structured filter values are shown to the preparer only (review L3).
    - Other holders see the filter keys, with the search text replaced by "search applied", because `q` can hold part of a login.
  - The prepared preview (owner only) shows the fixed filter as chips, the fixed time, `matched`, the `no_longer_matches` count and the other classification counts.
  - `targets` paging is unchanged (HYB-299 amendment, `HYB-299/plan.md:1382-1388`).
- **Exclusions chosen by the user: a deferral the owner must accept** (review M4, the HYB-300 Q1 precedent, `HYB-300/plan.md:129`, `:675`). Built since under [HYB-324](../HYB-324/implementation.md); the question is no longer open.
  - `ui-design.md:421` puts targets **and exclusions** in the manifest. HYB-299's exclusions are system reasons only.
  - HYB-294 does not let a person deselect individuals from a filter list. Until the owner accepts this, it is an **open finding, not a shipped default**.
  - The proposed follow-up ticket is "M01 filter list: remove chosen people (`bulk_revocation.remove_targets` on a filter draft, exclusion reason `chosen`)".
  - Meanwhile the user narrows the filter, or discards the list and selects again. M6 depends on the same gap.
- **New setting `-bulk-revocation-max-matching`:**
  - people per filter-wide list;
  - default 5,000, bounds 1–100,000;
  - source `default` or `start_flag`, like HYB-299's settings (`HYB-299/plan.md:800-815`);
  - cross-check: at least `bulk_revocation_max_targets`.

  The explicit limit (500, 1–5,000) is unchanged, because its bound comes from staging batches. Filter-wide selection sends one command, so its limit is bounded by lock time instead (D8).
- **Console selection bar.**
  - With `can_bulk_revoke`, the bar shows a second action, "Select everyone who matches these filters", next to "Select the {n} loaded people".
  - Opening the action calls the bounded count read (below) and labels the confirm button "Select all {n} who match" or "Select all who match (at least {n})". When the count answers `over_limit`, the action shows the "too many" copy at once, so the user learns it before trying.
  - When the fixed set differs from the count, the fixed line says so: "{m} people fixed at {time} ({n} when counted)".
- **Bounded count before selecting** (review M4; placed in Part B by review N1).
  - `GET /api/v1/identity/users/count` takes the same filters and the same admission as the list. It runs under `s.domain.RLock`.
  - It stops at the first of these:
    - the end of the list;
    - `user_count_scan` people examined, a new setting: default 20,000, bounds 1,000–100,000;
    - `-bulk-revocation-max-matching` + 1 matches.
  - It answers `{count, complete, over_limit, list_version, server_time}`:
    - `complete: true` when the scan reached the end of the list, so `count` is exact;
    - `over_limit: true` when more people match than one list allows;
    - otherwise `count` is the matches found so far, and the screen says "at least {n}".
  - The console calls it only when the user opens the select-matching action, not on every filter change.
  - The count is not a fixed set. The set and its exact count are fixed by `add_matching` (W03, `ui-design.md:278-279`).
  - Explicit picks are cleared only after an inline confirmation: "This replaces your {n} selected people with everyone who matches."
  - After success the bar reads "{matched} people fixed at {time} for these filters" with the chips, then "Prepare revocation" and "Discard list". Changing filters afterwards shows "The fixed list keeps its own filters", and the list stays as it is.

### D8. Phase-B lock hold time (HYB-299 `plan.md:1177`)

- **Kept under the domain write lock, before the answer.**
  - Phase B is what makes a revocation replay after a restart. The header carries no target list, so moving the item records after the answer would let an acknowledged revocation be lost on a crash before they are written. That reverses HYB-298's rule that a decrease is applied and then committed, or shown `decision_unconfirmed`.
  - It would also break the planning rule that the independent decision ledger's durable commit comes **before** the answer confirming a revocation (`retention-access.md:98`: "삭제·회수 확정 응답 전 독립 결정 원장의 내구 commit을 보장한다"; acceptance RET-05, `:122`) (review L8).
  - Holding the lock does not delay the deny for anyone. Every authenticated path waits on the domain lock, so no request runs on the old access while phase B writes. The cost is latency for other people's requests.
- **Expected hold.** Linear in targets from the recorded engine numbers: about 25 µs and 0.004 syncs per item. **These are estimates, not measurements:**

| Targets | Item syncs | Phase B, est. |
| --- | --- | --- |
| 500 | 3 | ~13 ms |
| 5,000 (default filter limit) | 21 | ~125 ms |
| 20,000 | 80 | ~0.5 s |
| 100,000 | 392 | ~2.5 s |

- **Part B measures the whole command at the HTTP level** at 5,000, 20,000 and 100,000. The measurement covers:
  - `add_matching`;
  - `prepare`, which includes `everyone()` after D5;
  - `execute`: the sweep, phase A, phase B, and the time to answer.

  These are recorded numbers, not thresholds. The default limit stays at 5,000 regardless.
- **Owner question Q3** asks whether to accept the measured 100,000 hold.
- **Not in this ticket.** Raising `decisionledger.BatchLimit` (256, `types.go:15`) to cut syncs changes the HYB-298 write bound and needs its own durability review. It is a follow-up only if the measured hold is not accepted.
- **Ledger capacity.** 100,000 item records use roughly 100,000 × the measured record size of the 256 MiB default. Part B records the measured bytes per item. The preview does not refuse on space, because revocations are never refused by ledger state (HYB-298). A residual notes that a few 100,000-person revocations can fill the default ledger.

## Part A: indexed listings, server search, filters and paging

### Allowed files

- `modules/identity`: `registry.go`, `installation.go` and `workspace.go` (the index fields and invalidation; `Users` and `Members` paging); new `people.go`; tests (`people_test.go`, a listing property test, `scale_test.go` additions). **No write path changes** beyond invalidation.
- `apps/api/server`:
  - `users.go`;
  - new `userfilter.go` (the parser and evaluator, shared with part B);
  - `directory.go` (the `order` and `departed` indexes, `byLogin`, `remove`, the directory version);
  - `admin.go` (`roster`, `users` and `findUser` through the index; `departUser` calls `dir.remove`);
  - `bulkread.go`, plus `LatestFor` in `modules/bulkrevoke/job.go` (the page-scoped latest outcome, M3);
  - `bulkimpact.go` (`everyone` through the index only);
  - `settings.go` and `server.go` (one setting);
  - `openapi.go`;
  - tests (`users_filter_test.go`, `users_property_test.go`, `users_scale_test.go`).
- `apps/api/cmd/hb-api/main.go` and `main_test.go`: one flag.
- `apps/console/src/service/admin.ts` and `admin.test.ts`: `users(query)` and the page fields.
- `apps/console/src/modules/identity/pages/users/`:
  - `Live.tsx` (wiring);
  - new `filters.tsx`, `filter-model.ts`, `filter-model.test.ts` and `filter-copy.ts`;
  - `live-copy.ts` (count lines);
  - `bulk.css` (band styles).

  `live.css` is **not** changed, because it is shared with M03–M05 and O01–O03 (`HYB-299/plan.md:157`).
- `apps/console/src/app/screen-groups/identity.tsx`: `params` and `onParams` for M01 only.
- `apps/console/package.json`: the test list.
- `docs/ui/terminology/product-glossary.md`: new rows.
- `docs/features/HYB-294/*` and the scratchpad QA runner.

### API

`GET /api/v1/identity/users?q=&status=&membership=&role=&period=&signin=&invitation=&holder=&after=&limit=` with the semantics in D2 and D4.

| Setting | Flag | Unit | Default | Bounds |
| --- | --- | --- | --- | --- |
| `user_filter_scan` | `-user-filter-scan` | people | 1000 | 100–10,000. The upper bound stays until the scale test measures larger budgets. |

### Console

- **Filter band** (new `filters.tsx`, modelled on O01's `filters.tsx`, not imported from it, so O01 is not touched):
  - Search: "Search login, email or user ID", submitted on Enter or Search. It is too long above 64 bytes (UTF-8, `utf8Length`).
  - Selects:
    - Account status;
    - Membership in this workspace;
    - Workspace role;
    - Role period;
    - Sign-in setup.
  - One checkbox, "Installation administrators only".
  - The sign-in method line (text).
  - Under the title, the scope line "Workspace: {workspace}", fixed.
- **Layout.**
  - The band is collapsible below 1280px (`M01/spec.md:112`; O01 Part A) and uses the full viewport width (`service-screen-style.md`).
  - The reading order is title → scope → tabs → filters → list → detail → actions (`M01/spec.md:114`).
  - It uses neutral borders and no accent strips.
- **Chips** for each applied filter, with "Clear all filters" (`ui-design.md:417`). A chip is a button whose accessible name includes "Remove".
- **List.**
  - Pages of 50 from the server with the query; "Show more users" follows `next`.
  - Counts and the scan stop are worded as in D4. The selection model is unchanged.
  - A change of filters replaces the rows (new first page) and announces "List updated: {count line}" in the live region.
- **States** (`M01/spec.md:89-98`):
  - loading;
  - a filtered empty, "No people match these filters in your permitted scope.";
  - an unfiltered empty (the existing text);
  - error with retry;
  - a next-page failure that keeps the loaded rows and offers retry;
  - denied without the band;
  - the list-version notice from D4;
  - the "list changed, start again" refusal from D4, with the Start again action.

### Copy drafts (EN / KO / AR)

These are AI drafts for TW and native-language review (`product-terminology.md`). The terms follow the existing `live-copy.ts` and `period-copy.ts` and the O01 count lines.

| Key | EN | KO | AR |
| --- | --- | --- | --- |
| scope | Workspace: {name} | 업무 공간: {name} | مساحة العمل: {name} |
| search | Search login, email or user ID | 로그인·이메일·사용자 ID 검색 | البحث باسم الدخول أو البريد أو معرّف المستخدم |
| status | Account status | 계정 상태 | حالة الحساب |
| membership | Membership in this workspace | 이 업무 공간 소속 | العضوية في مساحة العمل هذه |
| noMembership | No membership | 소속 없음 | بلا عضوية |
| membershipEnded | Role period ended, not yet suspended | 역할 기간 종료(정지 전) | انتهت مدة الدور (قبل التعليق) |
| restart | The list changed while you were paging. Start again from the first page. / Start again | 페이지를 넘기는 동안 목록이 바뀌었습니다. 첫 페이지부터 다시 시작하세요. / 처음부터 다시 | تغيّرت القائمة أثناء التنقل بين الصفحات. ابدأ من جديد من الصفحة الأولى. / البدء من جديد |
| departedRow | Departed (login removed) | 퇴사(로그인 삭제됨) | مغادر (أزيل اسم الدخول) |
| role | Workspace role | 업무 공간 역할 | الدور في مساحة العمل |
| period | Role period | 역할 기간 | مدة الدور |
| open / bounded / soon / ended | No end date / Has an end date / Ends within {n} days / Role period ended | 종료일 없음 / 종료일 있음 / {n}일 안에 종료 / 역할 기간 종료 | بلا تاريخ انتهاء / له تاريخ انتهاء / تنتهي خلال {n} يومًا / انتهت مدة الدور |
| signin | Sign-in setup | 로그인 설정 | إعداد تسجيل الدخول |
| set / setLink / resetLink / notSet | Password set / Set-password link outstanding / Reset link outstanding / No password and no link | 비밀번호 설정됨 / 비밀번호 설정 링크 발급됨 / 재설정 링크 발급됨 / 비밀번호·링크 없음 | تم تعيين كلمة المرور / رابط تعيين كلمة المرور صادر / رابط إعادة التعيين صادر / لا كلمة مرور ولا رابط |
| holder | Installation administrators only | 설치 관리자만 | مسؤولو التثبيت فقط |
| method | Everyone signs in with a password set at this installation. Single sign-on (SAML, OIDC) and directory sign-in (LDAP) are not connected. | 모든 사용자는 이 설치에서 설정한 비밀번호로 로그인합니다. SSO(SAML·OIDC)와 디렉터리 로그인(LDAP)은 연결되어 있지 않습니다. | يسجّل الجميع الدخول بكلمة مرور معيّنة في هذا التثبيت. تسجيل الدخول الأحادي (SAML وOIDC) وتسجيل الدخول عبر الدليل (LDAP) غير متصلين. |
| clearAll | Clear all filters | 필터 모두 제거 | مسح كل عوامل التصفية |
| shownMore / shownAll | {n} shown; more may exist / {n} shown; end of list | {n}명 표시, 더 있을 수 있음 / {n}명 표시, 목록 끝 | المعروض: {n}؛ قد يوجد المزيد / المعروض: {n}؛ نهاية القائمة |
| searched | Searched up to {login}. / Continue searching | {login}까지 검색했습니다. / 이어서 검색 | تم البحث حتى {login}. / متابعة البحث |
| searchedDeparted | Searched up to user {id}. | 사용자 {id}까지 검색했습니다. | تم البحث حتى المستخدم {id}. |
| filteredEmpty | No people match these filters in your permitted scope. | 허용 범위에서 이 필터에 맞는 사람이 없습니다. | لا يوجد أشخاص يطابقون عوامل التصفية هذه ضمن نطاقك المسموح. |
| changed | The list changed after the first page. Reload for a consistent list. Changes that come only from time passing, such as a role period ending, are not detected. | 첫 페이지 이후 목록이 바뀌었습니다. 일관된 목록을 보려면 다시 불러오세요. 역할 기간 종료처럼 시간이 지나서 생기는 변화는 감지하지 않습니다. | تغيّرت القائمة بعد الصفحة الأولى. أعد التحميل للحصول على قائمة متسقة. التغييرات الناتجة عن مرور الوقت فقط، مثل انتهاء مدة الدور، لا تُكتشف. |
| jobsNote | Filters apply to Users, Pending invitations and Workspace membership. | 필터는 사용자·대기 중인 초대·업무 공간 소속에 적용됩니다. | تُطبَّق عوامل التصفية على المستخدمين والدعوات بانتظار القبول والعضوية في مساحة العمل. |

The AR plural forms follow the existing `plural` helpers. Logins, IDs and dates sit in their own `bdi`, keeping the HYB-299 QA checks (b) and (c) (`HYB-299/verification.md:125`).

### Part A tests

**Go, modules/identity**
- `Users` and `Members` give unchanged results:
  - the existing tests;
  - a property test against the old sort-and-page implementation, kept in the test file as the oracle.
- The index is invalidated at each insert point. A removed membership is skipped and does not shorten a page wrongly.
- `People`: unknown IDs, the limit, a bad workspace, a non-administrator verifier, `Effective` at a period end, and isolation across two workspaces.
- Index helpers (M2):
  - the index and the version are correct after Bootstrap, CreateUser, CreateServiceUser and Accept;
  - the version is unchanged by a session open or close;
  - the source test forbids direct map writes.
- `-race`: concurrent first readers of a newly invalidated index.
- 100% coverage of the new code.

**Go, apps/api**
- Each filter alone and combined, against the sample seed plus test users with each sign-in setup, each period state, a removed membership and a departed account.
- `q` on login, email and ID; case folding with non-ASCII (KO and AR test logins); 65 bytes refused.
- `method` refused with `unsupported_capability`; `workspace` refused in all three forms; unknown, repeated and empty parameters refused.
- The scan budget, with small injected settings:
  - exhausted exactly at the end: `scan_stopped` false and `next` empty;
  - exhausted before any match: zero rows, `scan_stopped` true and `next` set;
  - resuming without gaps or duplicates.
- `after` that is unknown or malformed: `invalid_request`.
- A person created between pages appears exactly once if their login sorts after the cursor, and not at all otherwise. The page's `list_version` changes; a time-only period end does not change it.
- Departure (H1):
  - `status=departed` returns departed people by user ID, after every person with a login;
  - an unfiltered walk ends with the departed segment;
  - a departure of the **cursor person** between pages gives `revision_conflict` (409, next action inspect) with field `after` (N2);
  - an unknown or non-human `after`, or a `d:` cursor whose person is not departed, gives `invalid_request` (N3);
  - departing a person without a directory entry still places them in the departed segment (N4);
  - a departure of another person between pages keeps paging valid, and the person is not listed twice;
  - `dir.remove` keeps `order`, `departed` and `byLogin` consistent;
  - the departed row carries no login or email.
- `membership`: every generated row matches exactly one of the four values, which the property test covers (M1).
- The workspace-source check (L1): with `s.bootWorkspace` changed, the read is refused.
- `LatestFor` returns the same per-person outcome as `Latest` for the page's IDs (M3).
- `findUser` gives the same result as the paged scan.
- A filter that matches a person whose period ends between pages: evaluated at each page's `server_time`.
- **Property test** (`users_property_test.go`): random populations (up to 2,000 people, random states, periods, links and logins), random filters and budgets. It pages to the end and compares the concatenation with a brute-force filter of all people in the **two-segment order**: login order, then departed people by user ID. Cursors are compared **per segment** (N3):
  - equal sets;
  - no duplicates;
  - order preserved;
  - every `next` strictly increasing.
- The invitations filter equals the old `rowsFor` rule on the same data.
- No session, a member session, CSRF missing; a registry fault gives `dependency_unavailable`; flag parsing and bounds; the key allowlist; 100% statement coverage of the changed files.

**Scale test** (`users_scale_test.go`, skipped with `-short`)
- Build 100,000 people with memberships through the public identity API in invite-and-accept pairs, so pending invitations stay near zero.
- If that is still superlinear (the HYB-299 D3 concern, `HYB-299/verification.md:298`), fall back to an engine-level module test with a recorded deviation.
- **Record**, before and after D5:
  - first page, a deep page (cursor at 90%), and a sparse filter that exhausts the budget;
  - `roster()`;
  - `everyone()`;
  - `sweepRoles` per command;
  - allocations.

  These are numbers, not pass thresholds. One assertion: a page does not call the sort-everything path, which a counting fault hook detects.
- Benchmarks `BenchmarkUsersPage` and `BenchmarkUsersFiltered` at 10,000 and 100,000.

**Console**
- `filter-model.test.ts`: link parameter parsing and dropping, chips, the canonical query, count lines.
- Render tests in EN, KO and AR for the band, the method line, the chips, the filtered empty state and the scan-stop line.
- `admin.test.ts` for the query and the fields.
- `npm test`, `tsc` and the build.

**Mutation**
- On the changed Go lines in `modules/identity` and on `userfilter.go`, `users.go` and `directory.go`.
- Each surviving mutant is explained by its readers or killed (memory: a surviving mutant is not dead code). This time the server mutation run is required, not skipped as in `HYB-299/verification.md:259`; the scope is limited to the new files to keep it feasible.
- Clean the session GOCACHE per batch (memory: mutation runs fill the disk).

**Browser QA**
- Against a live `hb-api` with the sample seed, and a QA harness that adds 300 synthetic people with `-user-filter-scan 100`.
- EN, KO and AR at 1440, 1280, 1024 and 390.
- The HYB-299 checks:
  - no page horizontal scroll;
  - header and cell counts equal;
  - no repeated block;
  - date and login isolates in AR;
  - no word joins;
  - neutral computed borders.
- Focus order; live-region announcements; filters surviving tab changes; the invitations fixed chip; the jobs-tab note; the scan stop; "Show more"; the next-page failure; the list-version notice; the "list changed, start again" refusal after a departure between pages; departed rows last; the bad-link notice; the denied state.
- Regressions: the M01 runner (HYB-299 `qa299.mjs` groups A–E, L, N), M03–M05 and P01–P05, because `admin.ts` changes.

## Part B: filter-wide selection

### Allowed files

- `modules/bulkrevoke`:
  - `types.go`: selection `filter`, the View fields, the `MaxMatching` limit, exclusion reason `no_longer_matches`;
  - `registry.go`: `AddMatching`, and the refusal of `Add` on a filter draft;
  - `manifest.go`: the digest, and `Prepare` using the draft's selection;
  - tests, including a pinned explicit digest vector, `drafts_test.go` and `property_test.go`.
- `apps/api/server`:
  - `bulkstage.go`: the definition and `addMatching`;
  - `bulk.go`: the prepare re-check (H2) and the blocked filter case (M6);
  - `bulkimpact.go`: per-prepare indexes of sessions, pickups and proposals (M3);
  - new `userscount.go`: the bounded count read under `RLock`, its route in `boundary.go`, its setting, and `openapi.go` (M4, N1);
  - `bulkread.go`: fields, and the search-text masking for anyone but the preparer (L3);
  - `settings.go`, `server.go` and `openapi.go`;
  - `trusthold.go` (one row: allowed during the trust hold, because it changes no access, like `add_targets`; `HYB-299/plan.md:704`, `:1283`, Q10);
  - tests (`bulk_matching_test.go`, `bulk_scale_http_test.go`).
- `apps/api/cmd/hb-api/main.go` and `main_test.go`: one flag.
- `apps/console/src/service/identity-bulk.ts` and its test.
- `users/Selection.tsx`, `BulkPrepare.tsx` (the fixed-filter facts in the preview), `bulk-model.ts` and its test, `bulk-copy.ts`, `bulk.css`.
- The glossary, and `docs/features/HYB-294/*`.

### Copy drafts (part B)

| Key | EN | KO | AR |
| --- | --- | --- | --- |
| selectMatching | Select everyone who matches these filters | 이 필터에 맞는 사람 모두 선택 | تحديد كل من يطابق عوامل التصفية هذه |
| replaceConfirm | This replaces your {n} selected people with everyone who matches. | 선택한 {n}명을 필터에 맞는 모든 사람으로 바꿉니다. | سيحلّ هذا محل الأشخاص المحددين ({n}) بكل من يطابق. |
| fixed | {n} people fixed at {time} for these filters | {time}에 이 필터로 {n}명을 고정했습니다 | تم تثبيت {n} شخصًا في {time} لعوامل التصفية هذه |
| fixedDiffers | {m} people fixed at {time} ({n} when counted) | {time}에 {m}명을 고정했습니다(집계 당시 {n}명) | تم تثبيت {m} شخصًا في {time} (كان العدد {n} عند العدّ) |
| keepsOwn | The fixed list keeps its own filters. Changing the filters above does not change it. | 고정 목록은 자체 필터를 유지합니다. 위 필터를 바꿔도 목록은 바뀌지 않습니다. | تحتفظ القائمة الثابتة بعوامل التصفية الخاصة بها. تغيير عوامل التصفية أعلاه لا يغيّرها. |
| notAdded | People who match after {time} are not in this list. Select again for a new list. | {time} 이후에 조건에 맞게 된 사람은 이 목록에 없습니다. 새 목록은 다시 선택하세요. | الأشخاص الذين يطابقون بعد {time} ليسوا في هذه القائمة. حدّد مرة أخرى لإنشاء قائمة جديدة. |
| tooMany | More people match than one revocation list allows ({limit}). Narrow the filters. | 한 회수 목록에 넣을 수 있는 인원({limit}명)보다 많은 사람이 해당됩니다. 필터를 좁히세요. | يطابق أشخاص أكثر مما تسمح به قائمة إبطال واحدة ({limit}). ضيّق عوامل التصفية. |
| noneMatch | Nobody matches these filters now. | 지금 이 필터에 맞는 사람이 없습니다. | لا أحد يطابق عوامل التصفية هذه الآن. |
| discardList | Discard list | 목록 폐기 | تجاهل القائمة |
| selectCounted | Select all {n} who match / Select all who match (at least {n}) | 조건에 맞는 {n}명 모두 선택 / 조건에 맞는 사람 모두 선택(최소 {n}명) | تحديد كل المطابقين ({n}) / تحديد كل المطابقين ({n} على الأقل) |
| noLongerMatch | {n} no longer match the filters and are not in the list. | {n}명은 더 이상 필터에 맞지 않아 목록에서 제외했습니다. | لم يعد {n} يطابقون عوامل التصفية، فهم ليسوا في القائمة. |
| blockedFilter | This list would leave no administrator who can manage access. Narrow the filters, for example Workspace role: Member. | 이 목록으로는 접근을 관리할 관리자가 남지 않습니다. 필터를 좁히세요. 예: 업무 공간 역할: 구성원 | لن يبقى بهذه القائمة أي مسؤول يمكنه إدارة الوصول. ضيّق عوامل التصفية، مثل: الدور في مساحة العمل: عضو |
| searchApplied | search applied | 검색 조건 적용됨 | تم تطبيق بحث |

The existing `filterWide` sentence (`bulk-copy.ts:20-21`) is removed together with its tests and QA expectations. That is the superseded-copy rule in `product-terminology.md`.

### Part B tests

**Go**
- `add_matching`:
  - equals the read's brute-force filter set;
  - is refused with `prerequisite_failed` when nobody matches, when more match than the limit, with an open draft, or at the pending limit;
  - `add_targets` on a filter draft gives `invalid_stage`;
  - a non-owner sees `NotFound` as before;
  - is refused without the bulk right, or without `administer_installation`;
  - `method` gives `unsupported_capability`; `workspace` gives `invalid_request`.
- Isolation: a person in another workspace's data (a direct call) is never matched.
- The digest:
  - the explicit vector is unchanged;
  - a filter digest changes with the filter, the matched count or the fixed time;
  - an execute with an explicit digest for a filter list is refused.
- The fixed set is not widened: a person created after `add_matching` who matches is absent from prepare. A target changed after preview is held (the HYB-299 path).
- Drift before prepare (H2): between `add_matching` and `prepare`, one member's role changes and another's period ends, so both stop matching.
  - Both are excluded with `no_longer_matches`.
  - The counts invariant holds, and the preview and the exclusions read show them.
  - A member who changes but still matches stays a target.
- A filter list that strands the anchor (M6) is refused with `prerequisite_failed`, and the draft stays unfrozen.
- Count read (M4):
  - exact with `complete: true` when the scan ends;
  - "at least" with `complete: false` at a small injected `user_count_scan`;
  - `over_limit: true` at max-matching + 1, stopping there;
  - runs under `RLock` (a concurrent read proceeds);
  - the same filters and refusals as the list;
  - the setting's bounds.
- Masking (L3): another holder of the bulk right sees "search applied" in place of `q`; the preparer sees the value.
- Canonical encoding (L6): values containing separators, colons and digits do not collide (a property test over random filter pairs).
- Replay: a filter job rebuilt from its header shows the "not recorded" selection basis, and its counts invariant holds.
- The trust-hold row; the setting's flag, bounds and cross-check; the property test in `modules/bulkrevoke` extended with filter drafts.

**HTTP scale test** (D8, skipped with `-short`)
- 5,000, 20,000 and 100,000 matched people.
- Record:
  - `add_matching`, prepare and execute durations;
  - phase A and phase B separately;
  - the number of syncs;
  - bytes per item;
  - the latency of a concurrent users read waiting on the lock;
  - the `bulkrevoke` View copy costs (M3);
  - the count read at 20,000 and 100,000 people, and the latency of a concurrent write command while counting (N1).
- **Realistic side data** (M3): about one open session per person for 10% of the targets, outstanding pickups for 1%, and pending proposals in each registered proposal store for 1%. Prepare's impact computation then pays its real cost.
- Assert isolation, deny before the first ledger write, and the counts invariant (`HYB-299/plan.md:1080-1084`).
- At 100,000 the test runs with `-bulk-revocation-max-matching 100000`.

**Console**
- `bulk-model.test.ts`: replace-confirmation logic, fixed-list state.
- `identity-bulk.test.ts`: the command and the fields.
- Render tests in EN, KO and AR.

**Mutation**
- On the changed lines in `modules/bulkrevoke` and the new server files, as in part A.

**Browser QA** (EN, KO and AR at 1440, 1280, 1024 and 390)
- Select matching:
  - the counted label, exact and "at least" (with a small `-user-count-scan`);
  - the "too many" copy shown before trying (`over_limit`);
  - the "({n} when counted)" line after a change between counting and fixing;
  - replacing explicit picks with confirmation;
  - the fixed line;
  - filters changed after fixing.
- Prepare:
  - the preview with the fixed filter facts;
  - **drift**: change one member's role in a second session between fixing and preparing, and the preview shows "1 no longer match";
  - the **anchor-blocked filter list**, with the blocked copy.
- Refusals: too many (with `-bulk-revocation-max-matching 2`); none match.
- Discard; execute and the job row with `selection`; another administrator sees "search applied".
- Re-run the part A QA and the HYB-299 runner.

## Verification common to both parts

- `gofmt`; `go vet` for darwin, linux and windows; linux/386 and windows/amd64 builds.
- `go test ./...`.
- `-race -count=40` on the changed packages' new tests.
- 100% statement coverage of the changed server files and the new module code.
- No floating-point type in the new Go tests (the HYB-299 source rule).
- No file above 1,500 lines. `Live.tsx` is 687 lines today, and new UI goes into new files.
- `verification.md` with the planning cross-check below, filled with actual results. Facts, numbers and unrun checks are stated as such.

## Out of scope

- A workspace selector across several workspaces, and role-scoped (AM) user views. Neither multiple workspaces nor role-scoped grants exist (D1).
- An authentication method filter on the connection. Only password exists (D3).
- Sort keys other than login.
- An exact total in the list read. The bounded count read gives "at least {n}" (D4).
- Exclusions chosen by the user in a filter list. This is an open finding until the owner accepts the deferral (Q4).
- Raising `decisionledger.BatchLimit` (D8).
- Search, filters and paging on M02–M05 and on the operations screens (HYB-301 rows 1–2, `HYB-301-trace-classification/report.md:316-317`).
- The login realm column, a display name and the full spec column order. No display name field exists; HYB-299 R8 stays.
- Invitation and sync at 100,000 (the rest of IAM-07).
- Saved filters (`requirements.md:547`, "저장 필터 공유", an open decision).

## Residuals

- R1. Only one workspace exists, so the "workspace" filter means membership in the caller's workspace. Cross-workspace views wait for workspace creation and role-scoped grants.
- R2. The sign-in method is stated as text. The method filter waits for a second connection (SAML, OIDC or LDAP; IAM-02).
- R3. The read has no immutable snapshot. Pages are evaluated at their own time, and a changed revision is announced, not prevented.
- R4. Filter-wide lists are fixed at selection. Their filter is not in the ledger. A replayed job shows the selection basis as not recorded.
- R5. Phase B holds the domain lock, at about 2.5 s for 100,000 (estimate; measured in part B). Other requests wait. The follow-up is to raise the batch bound after its own durability review, if Q3 is not accepted.
- R6. The ledger capacity for repeated 100,000-person revocations: bytes per item are measured, and nothing refuses on space (HYB-298).
- R7. `sweepRoles` scans every membership on every command (measured in part A). An expiry index is a follow-up if the numbers warrant it.
- R8. EN, KO and AR copy are AI drafts that need TW and native-language review. Screen acceptance (FIX) is the owner's.
- R9. The scan budget upper bound stays at 10,000 until measured.
- R10. Departed people sort last by user ID and can be found only by ID, because their login is removed at departure (H1; Q9).
- R11. The list-version notice does not detect changes that come only from time passing, such as a role period ending or a link expiring. The copy says so (M5).

## Planning cross-check

| ID | Source | What it requires here | Covered by | Expected status after HYB-294 |
| --- | --- | --- | --- | --- |
| M01 filters | `M01/spec.md:61` | Workspace, status, auth method, role, expiry | D2: `membership` (workspace, D1), `status`, `signin` plus the method as not connected (D3), `role`, `period` | covered; workspace limited to one (R1); method not connected (R2) |
| M01 search and sort | `M01/spec.md:63` | Server search, stable sort, selection by canonical ID | `q`; login order; selection by user ID (HYB-299) | covered |
| M01 tabs | `M01/spec.md:79`, `:81` | Filters and selection kept across tabs | Link parameters; D6 | covered |
| M01 scope | `M01/spec.md:25` | Scope fixed and checked on the server for search and totals | D1 | covered |
| M01 states | `M01/spec.md:89-98` | Empty, error and the rest, distinct from zero | Filtered empty, scan stop, next-page failure | covered |
| M01 large data | `M01/spec.md:104` | Cursor; page and filter-wide selection separate; fixed manifest, count and scope; no silent update | Part A cursor, budget and the list-changed refusal; part B `add_matching`, the fixed draft, the digest, and the prepare re-check that only narrows (H2) | covered after part B |
| M01 bulk action | `M01/spec.md:75` | Fixed manifest, target confirmation | HYB-299 prepare and preview, plus the fixed filter facts | covered |
| retention-access | `retention-access.md:65` | Search, status, membership, role, method, sessions; 100,000 managed with search, cursor and fixed manifest | D2, D3, D7 | covered except sessions (M05) and the method (R2) |
| IAM-07 | `retention-access.md:116` | 100,000: isolation, resume, block revision | Scale tests in both parts; isolation tests; the cursor resumes | partly covered: lookup and revocation; invitation and sync not built |
| UI-04 | `ui-design.md:547`, `:412-424` | Filter, selection, page and total bases agree; no full load | Server pages and budget; separate selection bases; counts never claim totals | covered for M01; other screens deferred (HYB-301) |
| W03 / W07 | `ui-design.md:270-292`, `:366-379` | Search and filters in a band; "select all N" | Band; the bounded count on the select action (exact, or "at least") | covered; "at least" beyond the count budget |
| Manifest exclusions | `ui-design.md:421` | Targets **and exclusions** fixed in the manifest | System exclusions, including `no_longer_matches`; exclusions chosen by the user are not built | open finding until the owner accepts (Q4) |
| Profile minimisation | `retention-access.md:76` | Remove unnecessary profile data | A departed person's login is not kept for ordering (H1) | covered |
| Durable commit before answer | `retention-access.md:98`; RET-05 `:122` | The ledger commit comes before the revocation answer | Phase B kept before the answer (D8) | covered |
| HB-03 | `requirements.md:53` | Scope checked on the server | D1; negative tests | covered (admin-only, as today) |
| HB-42 | M01 trace `:118` | Bulk block first | Unchanged deny-first; D8 | covered |
| HYB-299 R1 | `HYB-299/plan.md:1251` | Filter-wide on the staging transport; above 5,000 | D7, D8 | covered after part B |

## Owner questions (each ships with the stated default, changeable)

- **Q1. Order.** Default: login ascending, with no other sort. Alternative: also by account status or by role-period end, which needs further indexes.
- **Q2. Count budget (Part B).** Default: `-user-count-scan` 20,000 (1,000–100,000). Up to the budget, the count before selecting is exact; past it, the action reads "at least {n}". The count runs only when the select action is opened.
- **Q3. Filter-wide limit and lock hold.** Default: `-bulk-revocation-max-matching` 5,000 (bounds 1–100,000). Raising it accepts the measured phase-B hold (about 2.5 s estimated at 100,000) during which other requests wait.
- **Q4. Exclusions chosen by the user in a filter list: acceptance needed, not a default** (review M4; the HYB-300 Q1 precedent).
  - HYB-294 does not let a person remove individuals from a filter list (`ui-design.md:421`).
  - The owner is asked to accept the deferral and the follow-up ticket "M01 filter list: remove chosen people".
  - Until then it is an open finding.
  - It also affects the anchor-blocked case (M6): the user can only narrow the filters, for example to Workspace role: Member.
- **Q5. Filter-wide selection with no filter.** Default: allowed (the whole installation, self excluded at prepare, anchor rule enforced), labelled "everyone in the list". Alternative: require at least one filter or search.
- **Q6. The "ends soon" window.** Default: the existing `role_period_warning_days` setting (14). Alternative: a user-chosen "ends within N days" filter.
- **Q7. Scan budget.** Default: `-user-filter-scan` 1000 (100–10,000), as for O01.
- **Q8. Search fields and search text in links.** Default:
  - search covers login, contact email and user ID;
  - the search text may appear in the console link, as in O01;
  - bulk job rows show it to the preparer only, and others see "search applied" (review L3).

  Alternatives: login and ID only, to keep email out of search; or keep search text out of links.
- **Q9. Departed people in the order.** Default: departed people sort last, by user ID, without keeping their login (`retention-access.md:76`). Alternative: keep a minimised sort key, at the cost of a retained profile field.

## Facts, assumptions and unknowns

**Facts** are the `file:line` citations above, read on `e360bf7` on 2026-11-18.

**Assumptions** (to be checked at implementation):
- A1 (corrected after review H1). Every identity human who has **not departed** has a directory entry. Departed people have none (`admin.go:337`) and are listed in their own segment. Part A adds an invariant test: non-departed humans equal the directory entries, and departed humans equal `dir.departed`.
- A2. Logins never change while a person is in the directory. No rename command exists. A test fails if one is added without updating the index.
- A3. The phase-B time is linear in targets (D8 table). It is measured in part B.

**Unknowns:**
- the current users read cost at 100,000 (measured first in part A);
- `sweepRoles` per command at 100,000;
- bytes per ledger item;
- fsync latency on the QA host;
- final KO and AR wording.

## External references

- PatternFly, Bulk selection (https://www.patternfly.org/patterns/bulk-selection/, opened 2026-11-18).
  - **Observed:** a bulk selector offering "Select page" and "Select all" across pages; "Select all" may be omitted if not supported; the count label counts selections across all pages. The page does not say what happens to a selection when filters change.
  - **Adopted:** separate page and filter-wide actions, and a count across pages.
  - **Decided here instead:** the filter-change rule (D6, D7).
- Microsoft Learn, Manage user profile information (https://learn.microsoft.com/en-us/entra/fundamentals/how-to-manage-user-profile-info, opened 2026-11-18).
  - **Observed:** bulk edit on All users is limited ("select up to 60 users at a time").
  - **Adopted:** an explicit, stated limit per list (D7 setting).
- HYB-299 references 1–6 (`HYB-299/references.md`), in particular Carbon's batch action bar. Select-all across pages was rejected there pending HYB-294 and is adopted here through the server manifest.
- The filter band pattern follows the merged O01 Part A (`HYB-300/plan.md:219-256`, amendments `:725-732`). The ui-designer render review records any further references it opens.

## Review rounds

### Design review round 1 (2026-11-18): FAIL (2 high, 6 medium, 8 low)

Root's decisions, folded in above:

- **H1. Departed people.** `departUser` deletes the directory entry (`admin.go:337`), so the earlier A1 was false.
  - Departed rows come last, by user ID, with an explicit tie-breaker. A departed person's login is not kept for ordering (`retention-access.md:76`; Q9).
  - The index is updated at `departUser` (`dir.remove`). The cursor stays the user ID, with a segment prefix.
  - When the cursor person changed segment or vanished, the read answers `invalid_stage` (field `after`), and the console offers "Start again".
  - Tests: departure between pages, `status=departed`, and the order with departed rows.
  - Sections: Current behaviour; D2 (`q`); D4; Part A allowed files, states, copy and tests; A1; R10; Q9.
- **H2. Filter drift before prepare.** Prepare re-evaluates the stored filter for every draft member. People who no longer match are excluded with `no_longer_matches`, counted and shown in the preview. There is a Go test and a QA step. Sections: D7; Part B allowed files, copy and tests.
- **M1.** Membership value `ended` (active state, period ended, not yet swept), so the four values partition every row. Property test. Sections: D2; Part A tests; copy.
- **M2.** One insert/delete helper per index, bumping a per-index version counter (not `r.generation`). A source test forbids direct writes. Invalidation is tested at Bootstrap. Sections: D5; Part A tests.
- **M3.**
  - `LatestFor` for the page's IDs only;
  - per-prepare indexes of sessions, pickups and proposals;
  - `findUser` through the index;
  - `bulkrevoke` copy costs recorded;
  - the part B scale test carries realistic side data.

  Sections: D5; both allowed-files lists; Part B tests.
- **M4.** A bounded count read ("at least {n}", or exact when the scan ended) under `-user-count-scan`, shown before select-matching. Q4 is now a deferral the owner must accept, an open finding with a follow-up ticket proposal (HYB-300 Q1 precedent). Sections: D4; D7; cross-check; Q2; Q4.
- **M5.** A users-scoped `list_version` (identity listing version plus directory version) for the list-changed notice. The copy says time-based changes are not detected. Sections: D4; D5; copy; R11.
- **M6.** A filter list that would leave no anchor administrator gets blocked copy naming a narrower filter (EN/KO/AR), with a test and a QA step, and is mentioned in Q4. Sections: D7; Part B copy and tests.
- **L1.** Both workspace sources are stated and asserted equal (D1).
- **L2.** `People` takes the verifier form (D5).
- **L3.** `administer_installation` is the scoped right for search. `q` may appear in links. Job rows show the search text to the preparer only (D1, D6, D7; Q8).
- **L4.** The contradictory "outside the current filters" sentence is dropped (D6).
- **L5.** In-place insert and delete in the sorted slices instead of a dirty rebuild (D4, D5).
- **L6.** Length-prefixed canonical filter encoding. `Prepare` stops hard-coding `Explicit` (`manifest.go:87`). `FixedAt` versus equal-manifest reuse is stated (D7).
- **L7.** Directory fields are writable only through `add` and `remove`, with a source test (D4).
- **L8.** D8 cites `retention-access.md:98` and RET-05 (`:122`).

### Design review round 2 (2026-11-18): FAIL narrowly (no High)

Root's decisions, folded in above. Root approves the plan on this fold after a targeted diff check, so the status is `approved`.

- **N1. The count read.**
  - It moves to Part B: the description, the files and the tests. Part A has no count.
  - It runs under `s.domain.RLock`.
  - It stops at the first of the end of the list, `user_count_scan`, or max-matching + 1, and answers `over_limit`, so the console shows "too many" before the user tries.
  - The Part B scale test measures it at 20,000 and 100,000, together with the latency of a concurrent write.
  - New copy: "{m} fixed at {time} ({n} when counted)".
  - Sections: D4, D7, Part B allowed files, copy and tests, Q2.
- **N2. The restart answer** is `revision_conflict` (409, next action inspect) with field `after`, through a read-side error writer that carries fields. It is recorded in `openapi.go`, with a test of status and next action. Sections: D4, Part A tests.
- **N3. Cursor edge cases.**
  - An unknown or non-human `after` is `invalid_request`. Only "issued in the login segment, now departed" gives the restart answer.
  - The console replaces rows by user ID, so a person who departs while the user pages is not shown twice. This is a console and e2e test, not a server guarantee.
  - The property test uses the two-segment order and compares cursors per segment.
  - Sections: D4, Part A tests.
- **N4.** `departUser` inserts into `departed` regardless of `c.known`, with a test. Sections: D4, Part A tests.
- **N5.** `r.generation` is the identity part of `list_version`. The per-index counters serve only the index slices. The directory source test extends to `invites`, `links`, `holders` and the `passwords` Put and Remove call sites. Sections: D4.
- **N6.** "Searched up to user {id}." is added to the copy table. The URL and log statement is reworded as a fact. Sections: D6, copy.
