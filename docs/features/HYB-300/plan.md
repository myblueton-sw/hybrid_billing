---
type: ImplementationPlan
title: HYB-300 O01 server filters, create change plan handoff and view group drafts
description: Fix ticket for finding M4 of the HYB-291 planning cross-check. O01 gains server-side filters and search with stable paging, a create change plan action that hands canonical IDs, an effective date and the relationship revision to O02, and view group drafts that grant nothing and become one O02 change plan; delivered as three ordered parts.
status: stable
tags: [HYB-300, O01, O02, HB-272, HB-273, HB-274, HB-275, HYB-296, HYB-291, relationships, view-group]
sources:
  - resource: ../../ui/screens/O01/spec.md
  - resource: ../../ui/screens/O02/spec.md
  - resource: ../../planning/v1.16/requirements.md
  - resource: ../../planning/v1.16/domain-contracts.md
  - resource: ../../planning/v1.16/ui-design.md
  - resource: ../../planning/v1.16/acceptance-trace.json
  - resource: ../HB-272/plan.md
  - resource: ../HB-273/plan.md
  - resource: ../HB-274/plan.md
  - resource: ../HYB-296/plan.md
  - resource: ../../reviews/HYB-291-planning-cross-check/report.md
---

# HYB-300 O01 server filters, create change plan handoff and view group drafts

## Requirement

Owner decision 2026-11-17 (faithful to planning), recorded on HYB-300: the O01 spec requires two actions and five filters. HYB-291 recorded them as missing (`docs/reviews/HYB-291-planning-cross-check/report.md:55`, finding M4 at `:123`, action row `:149`).

- The actions are view group draft and create change plan (`O01/spec.md:72-73`).
- The filters are as-of date, legal entity, relationship kind, unattributed and provider (`O01/spec.md:61`).

They are built in the API and the screen and linked to O02 plans. Design review and change review are required.

The O01 spec and the O02 spec are local planning inputs (`docs/ui/screens/O01/spec.md`, `docs/ui/screens/O02/spec.md`). They are untracked in the authorized workspace. Line numbers below refer to those files.

Base: main `867a3e9`. Branch `feat/HYB-300-o01-group-draft-filters`. Machine: mac_mini.

## Size and proposed split

One implementer cannot review, test and run browser QA on this as one change. Three areas change: a public read contract, two public command contracts (one of them new) and both O01 and O02. The plan therefore has three parts, each with its own sub-ticket, PR, design review and change review. Each part depends on the one before it.

| Order | Sub-ticket (to be created under HYB-300) | Content | Depends on |
| --- | --- | --- | --- |
| 1 | HYB-300 part A: O01 server filters and paging | Organizations read filters and search, a targets listing, a member lookup; the console's accounts, groups and history views read server pages | — |
| 2 | HYB-300 part B: create change plan handoff | An O01 action, O02 prefill, an optional `base` field on `relationship.propose` | A (the panel and member lookup) |
| 3 | HYB-300 part C: view group drafts | A draft store, four commands, two reads, settings, O01 drafts UI, an O02 group change plan | A (member search), B (the handoff and `base`) |

Lazy loading of the tree is a separate follow-up ticket (see Out of scope), not part of HYB-300.

## Ownership

- Root (Claude Code) owns the plan, the implementation, the tests and browser QA. This follows the owner's standing instruction of 2026-11-12.
- Reviews are read-only:
  - an independent design reviewer (`independent-reviewer`) for each part, because each part changes a public contract, a tenant boundary or a permission (AGENTS.md core rules);
  - a change reviewer for each part, with a DBA-style focus on dates, revisions and attribution in part A and on plan building in part C;
  - the `ui-designer` role for the O01 and O02 render review;
  - a separate technical-writing reviewer for the copy (`.agents/rules/product-terminology.md`).
- One writer per file. No concurrent writers.

## Evidence and decisions

### D1. What a view group is in this codebase

**Facts**
- `modules/relationships` models the view group as the relationship kind `view_group` (`modules/relationships/types.go:27`):
  - the member is a source account or other relationship member;
  - the target is the group ID;
  - the kind is non-exclusive, so a member can be in several groups (`types.go:41`);
  - group IDs and member IDs must be disjoint (`types.go:43-45`, `validate.go:137-142`);
  - `Union` gives the deduplicated members of several groups on a day (`registry.go:231-256`). This is the HB-02 rule "조회 그룹은 원천/배분 ID 합집합으로 중복을 제거한다" (`requirements.md:45`).
- The planning domain lists `GroupMembership` in the organization and source account domain, beside `RelationshipRevision` (`domain-contracts.md:20`). For organizations and accounts the only planned mutation is "Create/validate/apply change plan" with "Expected revision, fixed targets and pool impact" (`domain-contracts.md:67`). O02's change kind distinguishes "연결/해제/이동/그룹" (connect, release, move, group) and requires a fixed target manifest (`O02/spec.md:49-50`).
- Groups have no entity of their own: no name, no state, no registry. Names come from the seed label map (`apps/api/server/orgseed.go:10-31`).
- `relationship.propose` changes one member and accepts only a target that already exists in the active revision (`relplans.go:227-240`). A new group cannot be created today.
- Nothing outside `modules/relationships` reads `view_group` or `Union` (a search of `apps` and `modules` finds only the seed, the read and propose). In this build a view group therefore feeds no permission decision and no billing amount.

**Decision**
- A view group stays a set of dated `view_group` relationship timelines. Membership changes only through a change plan.
- A **view group draft** is a new, minimal pre-launch record in `apps/api`, held in memory like the plan store (`relplans.go:46-89`). It holds:
  - the group: either an existing `view_group` target, or a new canonical ID generated by the server (`group-` + 16 hex digits). The ID is never derived from the name;
  - for a new group only, a draft display name (the field contract allows a name draft, `O01/spec.md:50`). A draft on an existing group takes no `name`; a `name` field there is refused with `invalid_request` `name`. The draft name becomes the group's label only when the plan from that draft is activated (Labels in part C). This is owner question Q8;
  - a set of members;
  - the author, the creation time and the last change time;
  - the plans created from it.
- A draft changes no relationship revision, no grant and no gate decision. "Create change plan" turns it into one O02 change plan that holds every member's join (D2). That plan is the atomic unit that HB-02 asks for ("의존 풀 단위로 원자 공개", `requirements.md:43`).
- The draft adds members only. Removing a member from a group is the existing O02 single-member "leave" change. This is owner question Q3.

**Alternatives rejected**
- *A group registry module with states and permissions.* Planning names no such aggregate and has no group mutation other than a change plan.
- *One single-member plan per member.* It is not atomic, and the open-plan limit of 16 (`server.go:136-138`) would cap a group at 16 members.

### D2. Create change plan: what is handed to O02 and where it is checked again

- **Single relationship.** The selected-row panel in every O01 view gets a "Create change plan" action. It navigates to `#/organization-changes?view=targets&kind=<k>&member=<id>&from=<day>&base=<R>` (plus `lang`). The values are:
  - `kind`: the view's relationship kind;
  - `member`: the canonical ID;
  - `from`: the as-of date, sent only when it is today or later;
  - `base`: the pinned relationship revision R.

  `kind` is part of the relationship's identity: the same member has separate timelines per kind (`types.go:121-125`), so the handoff names the kind with the member, never the member alone. Names are never passed. This is the rule "허용된 canonical 대상 ID·기간·revision만 전달한다" (`O01/spec.md:81`, `O02/spec.md:82`).
- **View group draft.** The draft's panel navigates to `#/organization-changes?view=targets&draft=<id>&from=<day>&base=<R>`.
- **Checks in O02.** O02 checks again:
  - its own reads are admitted, and a refusal shows the existing denied block (`changes/Live.tsx:403-405`);
  - the commands check permission, expected revision and idempotency as today (`changes/Live.tsx:426-446`).
- **Holding the revision.** "대상 revision 확보" (`O01/spec.md:72`) is enforced on the server.
  - `relationship.propose` gains an optional `base` field. When it is given and is not the active relationship revision, the command refuses `revision_conflict`. The refusal carries no field: the command registry copies refusal fields only for `invalid_request` (`modules/commands/execute.go:401-411`), and `modules/commands` is not changed.
  - `view_group.propose` makes `base` required.
  - On any `revision_conflict`, O02 reloads, compares the `base` it carries with `active_revision` and shows the base notice when they differ. Otherwise it shows today's conflict sentence.
  - O02 never moves the base silently (`O01/spec.md:102`). It shows that the revision changed and offers "Base on revision A". Only that button changes `base`.
- The expected revision of the command stays the API revision from O02's own read (`changes/Live.tsx:428`). `base` is a second, relationship-level check, the "기대 revision" of `O02/spec.md:51`.

### D3. Server filters and paging; the client-built tree

- The organizations read (`organizations.go:97-201`) gains these filters, combined with AND:
  - `entity`;
  - `unattributed`;
  - `subject`;
  - `q` (server search, `O01/spec.md:63`);
  - `member` (exact lookup);
  - `target`.

  `kind` already exists and is the relationship kind filter. `on` is the as-of date.
- **Paging stays stable.** Rows are sorted by member ID, with the `after` cursor on one pinned revision (as in HB-272).
- **The account set.** The registry deletes a timeline that has no entries (`validate.go:86-91`), and `Timelines` lists only members that have a timeline of the requested kind (`listing.go:22-27`). Accounts 01, 02, 05 and 06 have no view group timeline (`orgseed.go:73-77`). A per-kind listing would therefore silently omit them from "not in a view group" and would drop a payer-only account from the accounts view. The server defines the **account set** as the union, across all five kinds, of members that are not registered parties. With `subject=account`, rows come from that set. `unattributed` means: in the requested subject set, with no target of the requested kind on `on`, whether or not a timeline of that kind exists.
- **Scan budget.** A filtered request examines at most `org_filter_scan` candidate members, an installation setting with a default of 1000. When the budget runs out, the page can be shorter than `limit`. `next` is then the last examined member and `scan_stopped` is true, so the browser never needs the whole listing. When the list ends within the budget, `scan_stopped` is false and `next` is empty, even if the budget was used up exactly.
- **Cost per request.** The module keeps a sorted key index per (revision, kind), built once on first use and cached; revisions are immutable after activation, so the cache never goes stale. A page starts with `sort.Search` on the cursor, so one request costs about O(budget + log N), not a scan of the kind.
- **Provider.** No source-account-to-provider link exists. There is no account registry (`HB-273/plan.md:135`), and the provider connection domain (`modules/connectors`) has no account mapping. The filter is therefore declared not connected:
  - the read reports `provider_filter: "not_connected"`;
  - a `provider` parameter is refused with `unsupported_capability`;
  - the screen states the reason as text instead of showing a filter that does nothing (`usability.md`).
- **Server pages in the console.** The accounts, groups and history views read their own server pages with Load more. The 10-page automatic follow (`organizations/Live.tsx:13-14, 93`) remains only in the tree view.
- **The tree: a deferral the owner must accept.** The tree is still built on the client (`HB-273/plan.md:193`), which conflicts with UI-04 and `O01/spec.md:102`. HYB-300 does not resolve that conflict; it is not a shipped default. The plan asks the owner to accept the deferral (Q1) and to open a follow-up ticket for a lazy tree (`parent=` through `MembersOn`, `registry.go:196`) right after part C. The reason it is not in HYB-300: the ticket names the actions and filters, and the lazy tree changes the tree model, its completeness rules and its accessibility contract (`HB-273/plan.md:112-162`). That is a separate design. Until the owner accepts, the conflict stays an open finding, not a residual.

### D4. Permission

- Today every relationship command and read requires the installation administration action (`relplans.go:419`, `organizations.go:209`).
- The spec asks for "관계 변경 권한 AM" and "그룹 관리 권한" (`O01/spec.md:72-73`). The gate has no such action (`modules/auth/types.go:21-31`), and role-scoped access is a recorded residual (`HB-272/plan.md:210`).
- **Default (Q2):** the new commands and reads also require the installation administrator. A group management action is added together with role-scoped grants in a later ticket.
- The new handlers never call the access, grant or gate registries. The command framework still checks the caller's permission through the gate before any handler runs; that is the only gate use.

## Part A: server filters, search and paging

### Allowed files

- `modules/relationships`: `listing.go` and `listing_test.go` (the listings and benchmarks), and the revision struct in `registry.go` for the cached index (one unexported field and its builder). No write path changes.
- `apps/api/server`:
  - `organizations.go`;
  - new `orgfilter.go` and `orgtargets.go`;
  - new `orgsettings.go`;
  - their tests;
  - `boundary.go` (one route);
  - `server.go` (the `Config.Org` field and wiring only).
- `apps/api/cmd/hb-api/main.go` and `main_test.go`: flags only.
- `apps/console/src/service/admin.ts` and `admin.test.ts`: `organizations()` query fields and `targets()`.
- `apps/console/src/modules/organizations/pages/organizations/`:
  - `Live.tsx`, `live-model.ts`, `live-model.test.ts`, `live-copy.ts`, `live.css`;
  - new `filters.tsx` and `live-render.test.tsx`.
- `apps/console/package.json`: the test list.
- `docs/ui/terminology/product-glossary.md`: rows for the new terms.
- `docs/features/HYB-300/*` and the scratchpad QA runner.

### Module listings (additive, read-only)

- **Index.** Each revision gets a lazily built, cached index: per kind, the sorted member IDs and the sorted target IDs, plus the sorted union of member IDs across all kinds.
  - Readers hold only `r.mu.RLock` (`listing.go:13-14`, `registry.go:120-122`, `:196-198`), so the index is not built under that lock as a plain field write. It is built inside a per-revision `sync.Once` (`index.once.Do(build)`), and readers read the result only after `Do` returns. This also covers the shared empty revision (`registry.go:54`, `:78-79`), which every unknown workspace reads.
  - `apply` builds a fresh revision for a plan (`validate.go:62`). It never copies the index or its `Once` from the base revision, so a revision activated later builds its own index.
  - Draft and sealed plans never share an index; only activated revisions are read through it, and they are immutable.
- `(*Registry) TimelineOf(workspaceID string, number uint64, kind Kind, member string) (Timeline, bool, error)`:
  - an exact lookup of one member's timeline of one kind, entries deep-copied; `false` when the member has no timeline of that kind;
  - `ErrInvalid` for an unknown kind or a bad ID;
  - `ErrNotFound` for a missing revision. An unknown workspace at revision 0 gives the empty revision, as `Timelines` does (`listing.go:18`).
- `(*Registry) IsTarget(workspaceID string, number uint64, kind Kind, id string) (bool, error)`: whether `id` is a target of that kind in the revision, from the index. It replaces the page scan in `checkIDs` (`relplans.go:227-240`) for the new commands.
- `(*Registry) Targets(workspaceID string, number uint64, kind Kind, after string, limit int) ([]string, error)`: the distinct targets of one kind, sorted, after `after`; `1 ≤ limit ≤ PageSize`, with the same errors as `Timelines`.
- `(*Registry) Members(workspaceID string, number uint64, after string, limit int) ([]string, error)`: the sorted union of member IDs across all five kinds, after `after`. Party exclusion is done in `apps/api`, because parties live in `modules/organizations`.
- `Timelines` moves onto the index as well, so its existing per-page scan (an HB-272 residual) is removed. Its results do not change; the existing tests prove that.
- A Go benchmark (`BenchmarkMembersPage`, `BenchmarkTargetsPage`) runs at 10,000 and 100,000 members and records ns per page after the first build. The verification record states the measured numbers.

### `GET /api/v1/organizations` additions

Existing parameters are unchanged. New parameters:

| Parameter | Values | Meaning |
| --- | --- | --- |
| `subject` | `account`, `party`, `any` (default) | `any` keeps today's rows: members with a timeline of `kind`. `account`: the account set (D3), the members of any kind that are not registered parties, paged through `Members`; a row is returned even when it has no timeline of `kind`, with empty `targets` and `history`. `party`: registered parties with a timeline of `kind`. |
| `entity` | a legal entity party ID of the workspace, in any state | The member is the entity, or the entity is on the member's dated hierarchy chain on `on` at the read revision (`Chain`, `registry.go:209-229`). A member with no hierarchy target on the day matches no entity; the `unattributed` filter finds those. The registry parent is not used, because it is undated (`HB-273/plan.md:120-122`). A suspended entity is accepted: a read is a past or present view, and a suspended entity can still have had members on the as-of date (decision L8). Only commands refuse inactive parties (`relplans.go:223`). |
| `unattributed` | `true` | The member is in the requested subject set and has no target of the requested `kind` on `on`, including when it has no timeline of that kind at all. |
| `q` | 1–64 bytes | Case-insensitive substring of the member ID or its label. |
| `member` | repeated, 1 to 50 IDs | Exact lookup through `TimelineOf` for the requested `kind`. The rows come back in ID order. An unknown member is absent, not an error. `after` is not allowed with `member`. At 50 IDs of 128 bytes the query is about 7 KB, within the 8 KB request-line limit that common proxies apply; a test sends 50 maximal IDs. |
| `target` | an ID | The member has this target on `on` (`MembersOn`), paged by member ID. |
| `provider` | any | Refused `unsupported_capability` (D3). |

The response gains these fields:
- `filters`, which echoes the applied values: `{subject, entity, unattributed, q, member_count, target}`;
- `provider_filter: "not_connected"`;
- `scan_stopped` (a boolean);
- `scan_budget` (the setting's value).

Errors:
- A bad value, an `entity` that is not a legal entity party of the workspace, `member` combined with `after`, or more than 50 `member` values give `invalid_request`.
- A registry failure gives `dependency_unavailable`.

Every call keeps using `answer.Workspace()` (`organizations.go:218`).

### `GET /api/v1/organizations/targets`

- **Query:** `kind` (required, one of the five kinds), `on`, `revision`, `after`, `limit` (1–99, default 50), `q`.
- **Rows:** `{id, name, members_on}`, where `members_on` is the member count on `on` at the read revision.
- **Response fields:** `revision`, `relationship_revision`, `active_revision`, `on`, `kind`, `targets`, `next`, `scan_stopped`.
- **Admission and errors:** the same as the organizations read.
- This listing replaces the client-side inversion of group members (`live-model.ts:111-128`).

### Setting

`org_filter_scan`:
- flag `-org-filter-scan`;
- unit: timelines;
- default 1000;
- bounds 100 to 10000. The upper bound stays at 10000 until the benchmark and browser QA measure larger budgets.

It follows the HYB-298 pattern (`settings.go:15-121`): `fs.Func` with a canonical decimal, `ErrConfig` before listening, and the source `default` or `start_flag`. A zero `Config.Org` means all defaults. Rows reuse `settingRow` (`settings.go:101-121`) through a shared builder rather than a second row type. Cross-setting relations are checked in `OrgSettings.validate()`, like `AccessSettings.validate()` (`settings.go:81`). Part C adds settings to the same `OrgSettings`.

### Console (O01)

**Filter panel**
- A new `filters.tsx` serves the accounts, groups and history views. It is collapsible below 1280px (`O01/spec.md:110`). Its reading order is scope, then tabs, then filters, then list (`O01/spec.md:112`). It contains:
  - **Legal entity:** a select of the legal entity parties from the read, plus "All legal entities".
  - **Relationship kind:** the five kinds. The accounts view's default is hierarchy.
  - **Only unattributed:** now sent to the server.
  - **Search name or ID:** submitted on Enter or Search.
  - **Provider:** a text line, not a control: "Not available yet: source accounts are not linked to provider connections."
- Applied filters show as chips with "Clear all filters" (`ui-design.md:417`).
- Filters live in URL parameters (`entity`, `rk`, `ua`, `q`). They survive a tab change (`O01/spec.md:79`).

**Accounts view**
- Each page is one server page with `subject=account` and the filters, `limit=50`.
- The Organization and Payer columns are joined by one `member=` read per other kind for that page's members, at the same revision. When the relationship kind is not hierarchy or payer, a column "{kind} on {date}" shows that kind's targets.
- Load more appends the next page. The count reads "{n} shown; more may exist" or "All {n} shown". After `scan_stopped`, it reads "Searched through {id}. Continue searching".
- The model functions `accounts()` and `loaded()` are adapted so that a member beyond the cursor is never shown as absent (`HB-273/plan.md:116-119`).

**Groups view**
- The view group or allocation pool choice reads `/organizations/targets`.
- Selecting a group opens the panel with its members, read through `target=` and paged.
- "Not in a view group on {date}" becomes a paged read with `kind=view_group&unattributed=true&subject=account`. Through the account set it now includes accounts with no view group timeline (Accounts 01, 02, 05 and 06 in the seed).
- The persistent note stays (`live-copy.ts:59`).
- **Filters in the groups view.** The relationship kind is the view group or allocation pool choice; the panel's kind select is not shown here. Legal entity, search and unattributed apply to the member lists (a group's members and the "not in a group" list), not to the group list; the group list takes only search (`q` on the targets read). The chips say so: a chip reads "Members: {entity}" in this view. Group counts (`members_on`) are always unfiltered and are labelled "Members on {date}, all".

**History view**
- The member is found by server search, and the timeline is read with `member=`. This closes the HB-273 residual "no member filter" (`HB-273/plan.md:194`).

**Tree view**
- Unchanged. Its filter area shows the as-of date and the note "Filters apply to the accounts, groups and history views. The tree shows the loaded organization structure."

**Selection across tabs**
- The selected member stays selected across tabs by canonical ID (`O01/spec.md:79`), and the panel shows it when the new view has it loaded. When it is not on the loaded page, the panel says "Selected: {id}. Not on the loaded pages of this view." and keeps the selection.
- The accounts view's page selection (part C) is kept across tabs and cleared when a filter, the date or the revision changes, with an announcement "Selection cleared because the list changed."

**States**
- Every view keeps the loading, empty, error and revoked sentences of `O01/spec.md:89-96`.
- A filtered empty state gets its own text: "No items match these filters in your permitted scope on {date}."

### Part A tests

- **Module:**
  - `TimelineOf` and `Targets`: workspace isolation with two populated workspaces, order, cursor, limits, unknown kind or revision, copies.
  - 100% coverage of the new functions.
- **API:**
  - each filter alone and combined;
  - `entity` across the 2026-10-01 move (Account 02 stays under North) and for a unit;
  - an `entity` that is a unit, unknown or in another workspace gives `invalid_request`;
  - `unattributed` per kind, including the view-group outside list;
  - `q` matching the ID and the label, case-insensitively;
  - `member`: order, unknown member, the limit, and refusal with `after`;
  - `target` with the account that is in both groups;
  - the account set: Accounts 01, 02, 05 and 06 appear in `kind=view_group&unattributed=true&subject=account`; an account that has only a payer timeline (added in the test) appears in `subject=account` with `kind=hierarchy` as unattributed;
  - the scan budget with injected small settings:
    - exhausted exactly at the end of the list: `scan_stopped` false and `next` empty;
    - exhausted before any match: a zero-row page with `scan_stopped` true and `next` set;
    - the cursor resuming without gaps or duplicates;
  - a property test that pages through random workspaces, filters and budgets and compares the union of pages with a brute-force filtered set;
  - an activation between two pages of a pinned read: later pages stay on the pinned revision and `active_revision` reports the new one;
  - labels changing between pages (in part C, a label map entry written by an activation between two pages) while paging with `q`: rows are matched by the label at read time, the cursor is the member ID, and no row is repeated;
  - `member` with 50 maximal IDs (URL length) and 51 refused;
  - the benchmarks run (not asserted);
  - the index under concurrency (N1): many goroutines read the same revision for the first time at once under `-race`; the empty revision too; a revision activated after a read of its base builds its own index and returns the new members;
  - `provider` refused;
  - targets listing counts on and off the move day;
  - a different workspace sees nothing (a direct call, as in HB-272);
  - no session and a member session refused;
  - a registry fault;
  - flag parsing and bounds.
- **Console:**
  - model tests for the page join, cursor progress and chips;
  - render tests (new `live-render.test.tsx`) for the filter panel and the provider line in EN/KO/AR, and the filtered empty state.

## Part B: create change plan handoff

### Allowed files

- `apps/api/server/relplans.go` (the `base` field and check only) and a new `relplans_base_test.go`.
- `apps/console/src/modules/organizations/pages/organizations/{Live.tsx,live-copy.ts,live-render.test.tsx}`.
- `apps/console/src/modules/organizations/pages/changes/{Live.tsx,live-model.ts,live-model.test.ts,live-copy.ts,live-render.test.tsx}`.
- `apps/console/src/app/screen-groups/work.tsx`: pass `onDirty` to O02 (N2), and its test.
- `docs/ui/terminology/product-glossary.md`, `docs/features/HYB-300/*`.

### Server

- `relationship.propose` gains the optional field `{Name: "base", Type: commands.Integer}`, so the OpenAPI command schema types it. The handler still checks it: presence is `raw, ok := ctx.Fields["base"]`, so a field sent as "" is present and refused. A present value is parsed with `strconv.ParseUint(raw, 10, 64)` and must be canonical (`FormatUint` equals the input), as the read does (`organizations.go:129-135`). Anything else refuses `invalid_request` `base`.
- **Order of checks:**
  1. `validChange`;
  2. the effective date is not in the past;
  3. when `base` is present, it parses, else `invalid_request` `base`;
  4. the active revision is read (`dependency_unavailable`);
  5. **new:** when `base` is given and is not `active`, refuse `revision_conflict` (no field, D2);
  6. `checkIDs`, then the rest as today.
- Without `base` the behaviour is unchanged. The command catalogue and the OpenAPI command schema gain the field. This is a public contract change, which is why the design review is required.

### Console

**O01**
- The panel of a selected node or row gets the button "Create change plan". The kind is:
  - hierarchy in the tree;
  - the relationship kind in the accounts view;
  - the group kind in the groups view;
  - the history kind in the history view.
- The button is shown only when the selection is a loaded member or party at the pinned revision. A row with "Not loaded yet" has none.

**O02 prefill**
- On first render of the targets view, O02 reads `kind`, `member`, `from` and `base` once, then clears them from the URL with `onParams`.
- The member uses "Enter ID" mode when it is not among the loaded choices. The action and the target are not prefilled; the user chooses them.
- A `from` in the past is dropped and the hint is shown.
- A prefill note (`role=status`) is shown. When `base` is not `page.active_revision`, a notice is shown with the button "Base on revision {a}".

**Refusal**
- Any `revision_conflict` reloads. If the carried `base` differs from the reloaded `active_revision`, the base notice is shown; otherwise today's conflict sentence. The form is kept in both cases.

**Unsaved input**
- O02 reports its form as dirty through `onDirty` (part C, Unsaved input). A prefilled form counts as dirty only after the user changes it.

### Part B tests

- **Go:**
  - `base` equal to the active revision proposes;
  - `base` stale gives `revision_conflict` and no plan is created (the plan store and open plan count are unchanged);
  - an activation between O02's load and propose (injected impact evidence activates another plan): the propose with the old `base` gives `revision_conflict` and creates nothing;
  - `base` absent keeps the old path;
  - `base` values "-1", "01", present but empty (""), and 2^64 ("18446744073709551616") give `invalid_request` `base`;
  - the order against a past date (still `invalid_request` `from`) and an unreadable active revision (`dependency_unavailable`).
- **Console:**
  - prefill parsing: unknown kind ignored, a past date dropped, an ID not in the choices;
  - the base notice and its button;
  - the conflict handling: base notice when the reloaded active revision differs from the carried base, the plain conflict sentence when it does not (`react`, `live-model.ts:168-193`);
  - O01 button presence per view in EN/KO/AR render tests;
  - `onDirty` wiring: `work.tsx` passes `onDirty` to O02; O02 reports true after a user change and false after propose succeeds or the form is cleared.

## Part C: view group drafts

### Allowed files

- `apps/api/server`:
  - new `groupdrafts.go` (store and reads), `groupdraftcmds.go` (commands) and their tests;
  - `relplans.go`: `planMeta.draft`, `planRow.draft`, the shared label lookup and chunked building;
  - `organizations.go` and `orgtargets.go`: label lookup only;
  - `orgsettings.go`;
  - `admin.go`: the definition list only;
  - `boundary.go`: two routes;
  - `server.go`: wiring.
- `apps/api/cmd/hb-api/main.go` and `main_test.go`: flags.
- `apps/console/src/service/admin.ts` and `admin.test.ts`: the `CommandId` union, `viewGroupDrafts()` and `draftMembers()`.
- `apps/console/src/modules/organizations/pages/organizations/`: new `drafts.tsx`, `drafts-model.ts`, `drafts-model.test.ts`; `Live.tsx`, `live-copy.ts`, `live.css`, `live-render.test.tsx`.
- `apps/console/src/modules/organizations/pages/changes/{Live.tsx,live-model.ts,live-model.test.ts,live-copy.ts,live-render.test.tsx}`.
- `apps/console/src/app/screen-groups/work.tsx`: pass `onDirty` to O01 (N2), and its test.
- `apps/console/package.json`, `docs/ui/terminology/product-glossary.md`, `docs/features/HYB-300/*`.

### Store

**Record**
- Each draft has: `workspace`, `id` (`vgd-` + 16 hex digits), `group`, `newGroup`, `name`, `members` (a sorted set), `author`, `created`, `updated`, `discarded`, `plans` (plan IDs, oldest first) and `seq`.
- It is keyed by workspace and ID, like `planKey` (`relplans.go:81`).

**Name**
- Only for a new group: after trimming, 1 to 80 characters, with no control characters. Duplicate names are allowed, because the ID identifies the draft. A draft on an existing group has no name; `name` is refused there (`invalid_request` `name`) on `draft` and `edit`.

**State** (derived, never stored)
- The latest plan's state is read from the relationships registry with `s.rels.Plan(ws, planID)` (`modules/relationships/plan.go:284`), not from the plan store, because `prunePlans` can remove a closed plan's `planMeta` (`relplans.go:96-110`). The registry never deletes plans, so `s.rels.Plan` always finds a plan this draft created; an error there is answered `dependency_unavailable` by the read.
- `discarded` when the draft is discarded;
- otherwise from the latest plan: `proposed` while that plan is draft or sealed, `activated` when it was activated;
- otherwise `open`, also when the latest plan was rejected or discarded.
- When `relationship.activate` succeeds for a plan with a `draft`, it records the activated revision on the draft and in the label map (Labels).

**Open count (L6)**
- The open-drafts limit counts drafts in state `open` or `proposed`. Discarded and activated drafts do not count.

**Labels (M1)**
- A compact, workspace-scoped **label map** holds `(group ID → name, activated revision)`. An entry is written once, when `relationship.activate` succeeds for a draft plan on a new group, from that draft's name. It is never pruned and never overwritten by a newer or open draft.
- `s.label(ws, id)` returns `orgName(id)`, and when that is empty, the label map's name for `id`.
- Before activation, a draft name is never a group label. It appears only in draft and plan contexts, labelled as a draft name: the drafts read (`name`), the plans read's `draft.name`, and O02's group plan lead ("draft name: {name}"). The organizations and targets reads show the group's ID for a group no activated draft named.
- `s.label` replaces `orgName` in the organizations read, the targets read and the plans read (`relplans.go:504`, `:521`).

**Pruning (L4)**
- Discarded drafts beyond `view_group_drafts_kept` are pruned, oldest first.
- Activated drafts are bounded by a separate setting, `view_group_activated_kept` (default 1000). Beyond it the oldest activated drafts lose their body and member list; the label map entry stays, so group labels never fall back to IDs. In pre-launch activation is held (HYB-296), so the bound is not reached.
- Open and proposed drafts are never pruned; the open limit bounds them.

**Unsaved input (M4, N2)**
- The new draft form and the draft edit form are dirty while they differ from their starting values. O01 and O02 report this through the console's existing `onDirty` prop (`shared/domain-screen-ui/work/contracts.ts:11`); `work.tsx` passes it to `LiveOrganizations` and `LiveChanges`, which today receive no `onDirty` (`app/screen-groups/work.tsx:43`, `:49`). The pages add no `beforeunload` of their own.
- **Who renders the O01 tabs.** `App.tsx` renders the screen tabs (`planning-tabs`, `App.tsx:1063-1110`). O01 is not a `liveReadWorkbench` screen (`App.tsx:178-179`), so a tab click calls `canLeave()` (`App.tsx:210-222`), which asks "Discard unsaved input?" while dirty. The O01 page renders no tablist of its own.
- **Guards that cover each path:**
  - tab clicks: `canLeave()` in `App.tsx`;
  - sidebar and other app navigation: `navigate()`, which calls `canLeave()` (`App.tsx:223-233`);
  - any hash change, including the page's own `onView` and `onParams` and the "Create change plan" navigation to O02: the navigation guard set from `dirty` (`App.tsx:288-301`, `shared/navigation.ts:21-38`), which reverts the hash when the user declines;
  - browser unload: the app's `beforeunload` handler (`App.tsx:265-274`).
- **Inline notice for in-page actions.** The date, the filters and "Create change plan" are URL-backed in this plan, so the guard above already asks; an inline notice there would ask twice. The inline notice is used only for in-page actions that do not change the URL: selecting another draft or closing a form in the draft panel, and any filter or date control that the implementation applies without a URL change. Its wording follows the app's family: "Discard unsaved input?" with "Discard" and "Keep editing". "Keep editing" returns focus to the form. This narrows root's N2 list (date, filters, Create change plan) to avoid a double prompt; recorded as a deviation for root's check.
- The same applies to a changed O02 form, including a prefilled one after the user edits it.

### Commands

All four commands have:
- `Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}`;
- `Mutating: true` and a rationale;
- resolution only in `s.bootWorkspace` (as in `relplans.go:273-288`);
- the expected revision taken from the API revision.

| Command | Target | Fields | Rules | Result |
| --- | --- | --- | --- | --- |
| `view_group.draft` | `view-group-drafts` | `name` (Text: required for a new group, refused for an existing one); `group` (ID, optional: an existing group); `members` (Text, required: newline-separated IDs, 1–31 per call so that one value fits the 4096-byte limit, `server.go:131`) | Open drafts (L6) below `view_group_drafts_open`, else `prerequisite_failed`. `group`, when given, is a `view_group` target in the active revision (`IsTarget`), else `invalid_request` `group`. Each member is a source account (in the account set, not a party), not a target of any kind (`IsTarget` per kind) and not repeated, else `invalid_request` `members`. For each member, `buildTimeline(entries, "join", group, today)` runs on its current `view_group` timeline (`TimelineOf`), and the built length must not exceed `EntriesPerTimeline` (64, `server.go:137`); else `invalid_request` `members`. A member already in the group whose timeline already has 64 entries is accepted, because the join does not lengthen it (N4). An unreadable active revision gives `dependency_unavailable`. | `draft`, `group`; state `open` |
| `view_group.edit` | draft ID | `name` (new group only), `add`, `remove` (all optional, at least one) | The draft is `open` (not discarded and no open plan), else `invalid_stage`. `add` follows the member rules, including the built-length check. `remove` must name current members. The result must have 1 to `view_group_draft_members` members, else `invalid_request` `remove` or `prerequisite_failed` `add`. | state `open` |
| `view_group.discard_draft` | draft ID | — | `invalid_stage` while a plan from it is open. The user discards that plan in O02 first; a hidden cascade would conflict with O02's own record. | state `discarded` |
| `view_group.propose` | draft ID | `from` (Text date, required, today or later); `base` (Integer, required; presence and canonical form checked as in part B) | See the list after this table. | `plan`; state `sealed` or `rejected` |

`view_group.propose` checks, in order:
1. The draft is `open`, else `invalid_stage`.
2. A valid date that is not in the past, else `invalid_request` `from`.
3. The active revision is readable, else `dependency_unavailable`.
4. `base` is the active revision, else `revision_conflict` (no field, D2).
5. The members are validated again against the active revision, including the per-timeline entry check (`invalid_request` `members`). An existing group is still a target (`IsTarget`, `invalid_request` `group`). A new group ID is still unused as a target of any kind or as a member (`invalid_stage`).
6. `discardStale` runs as in propose. The open-plan limit gives `prerequisite_failed`.
7. For every member, `buildTimeline(entries, "join", group, from)` (`relbuild.go:68`) on its `TimelineOf(view_group)` entries, with the actual `from`. A built timeline longer than `EntriesPerTimeline` refuses `invalid_request` `members` before any plan is drafted.
8. `Draft`, then `AddChunk` in chunks of `TimelinesPerChunk`, then `Seal`. The plan's total entries can exceed `EntriesPerPlan` (50,000, `server.go:138`) for large groups with long histories; Seal then rejects the plan with the rule `plan_limit` (`modules/relationships/plan.go:180-181`). The plan is stored as `rejected`, the draft stays `open`, and O02 shows the existing rejected state with the rule sentence; the copy row `planLimit` adds "This change is too large for one plan. Remove members from the draft and propose again."
9. A failure after the draft discards it and answers `errPlanPartial`, as in propose (`relplans.go:359-367`). Members already in the group are left to the module, which ignores unchanged timelines. A plan where nothing changes is rejected with `no_change` (`modules/relationships/plan.go:184-193`).
10. `planMeta{change: {kind: view_group, action: join, member: "", target: group, from}, draft: id}` is stored.

The plan touches only `view_group` timelines. A test asserts that its `kinds` is exactly `[view_group]`. This keeps the rule that a view group change never changes a contract pool or a payer (`ui-design.md:183`, `docs/ui/screen-binding-register.md:70`).

**Activation** is the O02 path and is held by HYB-296 until impact evidence exists. A view group change still goes through it, because the O02 progress model applies to every plan. Whether a view group change has money impact is owner question Q4; the default keeps the hold.

### Reads

`GET /api/v1/organizations/view-group-drafts?after=&limit=` (`limit` 1–50, default 50):
- Order: open and proposed drafts first, then the rest, newest first, as in the plans read (`relplans.go:588-600`).
- Response fields:
  - `revision`, `active_revision`;
  - `settings`: the `settingRow` rows (`settings.go:101-121`) for every `OrgSettings` value;
  - `open` (L6 count), `open_limit`;
  - `drafts`: `[{id, name, group, group_label, new_group, member_count, members (first 50 as {id, name}), author, author_login, created, updated, state, activated_revision, plan: {id, state, closed, blocked} | null}]`. `name` is the draft name (empty for an existing group); `group_label` is `s.label` of the group, so it is the ID for a new group until its plan activates (M1);
  - `next`.

`GET /api/v1/organizations/view-group-drafts/{id}/members?after=&limit=`:
- paged members `{id, name}`;
- an unknown draft answers `unsupported_command`, like the contract read (`admin.ts:706-713`).

Both reads are admitted like the other organization reads and use `answer.Workspace()`.

The plans read: `planRow` gains `draft: {id, name, group, new_group, member_count}`, empty for a single-member plan. `member_name` is empty for a group plan.

### Settings (added to `OrgSettings`)

| Setting | Flag | Default | Bounds |
| --- | --- | --- | --- |
| `view_group_drafts_open` | `-view-group-drafts-open` | 16 | 1–200 |
| `view_group_draft_members` | `-view-group-draft-members` | 1000 | 1–10000 (`TimelinesPerPlan`) |
| `view_group_drafts_kept` | `-view-group-drafts-kept` | 200 | 50–10000 |
| `view_group_activated_kept` | `-view-group-activated-kept` | 1000 | 100–100000 |

`OrgSettings.validate()` also checks that `view_group_draft_members` is not above `TimelinesPerPlan` and that `org_filter_scan` is at least the read page size (99). The drafts read shows the values and their sources. Changing them is a start flag, not a UI action, as in HYB-298.

### No permission is granted (`O01/spec.md:73`)

- The new handlers never call the access, grant or gate registries. The command framework's own permission check for the caller runs before the handler, as for every command; it decides only whether the administrator may run the command.
- A test runs `view_group.draft`, `edit` and `propose`, and an activation with injected impact evidence (the HYB-296 seam `s.plans.impact`). After them the grants list, the access read and a gate decision for a member session are unchanged.
- Part of GH04 (`acceptance-trace.json:809`) is covered: membership change does not change permission. Its full scope (past documents and pending files) is not connected.

### Console (O01 groups tab)

**Drafts section**
- With the view group kind, a "View group drafts" section sits above the group list. Its table has the columns Name, Group ("New group" or the existing group's name), Members, State and Change plan (a link to the O02 plan).
- The note is persistent and shown before any form: "A view group is not ownership. Creating or editing a draft grants no access and changes no relationship. Members join only when a change plan is activated."

**New view group draft**
- An inline form, not a dialog, so that the 390px flow works. It has:
  - Name;
  - Group: New group, or an existing group chosen from the targets listing;
  - Members: server search (`subject=account`, `q`), with Add buttons, a selected list with Remove, and "Enter IDs";
  - Reason (the existing rationale component).
- At most 31 members per save. The form says so, and more are added from the draft's panel in batches.
- The error summary and field focus follow the O02 form (`changes/Live.tsx:604-608`).

**Accounts view selection**
- Row checkboxes and "Select all on this page".
- A bar reads "{n} selected on this page · Start a view group draft". It opens the form with those members.
- Selection across the whole filter is not offered (see Residuals).

**Draft panel**
- Facts and paged members.
- Edit (name, add, remove).
- Discard draft, with an inline confirmation stating the effect.
- "Create change plan", which navigates as in D2.
- The panel shows only the actions that the draft's state allows. Absent actions are not rendered.

### Console (O02)

**Group draft handoff**
- With `draft` in the URL, the targets view shows "Change plan for a view group draft" with:
  - the name, the group and the member count;
  - the first 50 members;
  - the base notice;
  - the effective date and the reason;
  - "Propose".
- Propose runs `view_group.propose` and then opens the impact view, as propose does (`changes/Live.tsx:507-525`).

**Group plan detail**
- The lead is "Join view group {group}: {n} members from {date}", where {group} is the group's label (its ID for a new group), followed by "Draft name: {name}" for a new group (M1). The member table comes from `row.members`, the existing 50-row list.
- The HYB-296 held block and the four-step progress apply unchanged.

### Copy drafts (EN / KO / AR)

KO and AR are drafts for the technical-writing review. KO uses 공급사, the owner's term (`product-glossary.md:168`).

| Key | EN | KO | AR |
| --- | --- | --- | --- |
| createPlan | Create change plan | 변경 계획 만들기 | إنشاء خطة تغيير |
| filters | Filters | 필터 | عوامل التصفية |
| entity | Legal entity | 법인 | الكيان القانوني |
| allEntities | All legal entities | 모든 법인 | كل الكيانات القانونية |
| relKind | Relationship kind | 관계 종류 | نوع العلاقة |
| search | Search name or ID | 이름 또는 ID 검색 | ابحث بالاسم أو المعرّف |
| provider | Provider | 공급사 | المزوّد |
| providerNote | Not available yet: source accounts are not linked to provider connections. | 아직 사용할 수 없습니다. 원천 계정이 공급사 연결과 이어져 있지 않습니다. | غير متاح بعد: الحسابات المصدرية غير مرتبطة باتصالات المزوّدين. |
| clearAll | Clear all filters | 필터 모두 해제 | مسح كل عوامل التصفية |
| loadMore | Load more | 더 불러오기 | تحميل المزيد |
| shownMore | {n} shown; more may exist | {n}개 표시, 더 있을 수 있음 | عُرض {n}؛ قد يوجد المزيد |
| shownAll | All {n} shown | {n}개 모두 표시 | عُرضت كلها ({n}) |
| scanStopped | Searched through {id}. Continue searching | {id}까지 검색했습니다. 계속 검색 | تم البحث حتى {id}. متابعة البحث |
| filteredEmpty | No items match these filters in your permitted scope on {date}. | {date} 기준 허용 범위에서 이 필터에 맞는 항목이 없습니다. | لا عناصر تطابق عوامل التصفية هذه ضمن نطاقك المسموح في {date}. |
| treeFilters | Filters apply to the accounts, groups and history views. The tree shows the loaded organization structure. | 필터는 원천 계정·조회 그룹·귀속 이력 보기에 적용됩니다. 트리는 불러온 조직 구조를 보여줍니다. | تنطبق عوامل التصفية على عروض الحسابات والمجموعات والسجل. تعرض الشجرة بنية المؤسسة المحمّلة. |
| prefilled | Prefilled from Organizations, based on relationship revision {r}. | 조직·관계 탐색에서 가져온 값입니다. 관계 개정 {r} 기준입니다. | قيم مأخوذة من استكشاف المؤسسات، استنادًا إلى مراجعة العلاقات {r}. |
| baseChanged | The relationship revision changed from {r} to {a} after you opened Organizations. Check the current relationships, then base the plan on revision {a}. | 조직·관계 탐색을 연 뒤 관계 개정이 {r}에서 {a}(으)로 바뀌었습니다. 현재 관계를 확인한 뒤 개정 {a} 기준으로 계획하세요. | تغيّرت مراجعة العلاقات من {r} إلى {a} بعد فتح استكشاف المؤسسات. راجِع العلاقات الحالية ثم اعتمد المراجعة {a} أساسًا للخطة. |
| useBase | Base on revision {a} | 개정 {a} 기준으로 하기 | الاعتماد على المراجعة {a} |
| fromPassed | The as-of date {d} has passed. Choose an effective date from today. | 기준일 {d}은(는) 지났습니다. 오늘 이후의 적용일을 고르세요. | تاريخ المرجع {d} قد مضى. اختر تاريخ سريان من اليوم فصاعدًا. |
| drafts | View group drafts | 조회 그룹 초안 | مسودات مجموعات العرض |
| newDraft | New view group draft | 새 조회 그룹 초안 | مسودة مجموعة عرض جديدة |
| notOwnership | A view group is not ownership. Creating or editing a draft grants no access and changes no relationship. Members join only when a change plan is activated. | 조회 그룹은 소유권이 아닙니다. 초안을 만들거나 고쳐도 접근 권한이 부여되지 않고 관계도 바뀌지 않습니다. 변경 계획이 활성화되어야 구성원이 추가됩니다. | مجموعة العرض ليست ملكية. إنشاء المسودة أو تعديلها لا يمنح أي صلاحية وصول ولا يغيّر أي علاقة. ينضم الأعضاء فقط عند تفعيل خطة تغيير. |
| newGroup | New group | 새 그룹 | مجموعة جديدة |
| perSave | Up to 31 members per save. Add more from the draft afterwards. | 한 번에 구성원을 최대 31개까지 저장합니다. 나머지는 초안에서 추가하세요. | حتى 31 عضوًا في كل حفظ. أضف المزيد من المسودة لاحقًا. |
| selectedBar | {n} selected on this page · Start a view group draft | 이 페이지에서 {n}개 선택 · 조회 그룹 초안 시작 | {n} محدد في هذه الصفحة · بدء مسودة مجموعة عرض |
| stOpen / stProposed / stActivated / stDiscarded | Draft / Change plan open / Activated / Discarded | 초안 / 변경 계획 진행 중 / 활성화됨 / 폐기됨 | مسودة / خطة تغيير مفتوحة / مُفعّلة / مُتجاهَلة |
| discardDraft | Discard draft | 초안 폐기 | تجاهل المسودة |
| discardEffect | The draft is closed. No relationship or access changes. | 초안이 닫힙니다. 관계와 접근 권한은 바뀌지 않습니다. | تُغلق المسودة. لا تتغيّر أي علاقة أو صلاحية وصول. |
| groupLead | Join view group {group}: {n} members from {date} | 조회 그룹 {group} 가입: {date}부터 구성원 {n}개 | الانضمام إلى مجموعة العرض {group}: {n} عضوًا اعتبارًا من {date} |
| draftName | Draft name: {name} | 초안 이름: {name} | اسم المسودة: {name} |
| planLimit | This change is too large for one plan. Remove members from the draft and propose again. | 한 계획에 담기에는 변경이 너무 큽니다. 초안에서 구성원을 줄인 뒤 다시 제안하세요. | هذا التغيير أكبر من أن تتسع له خطة واحدة. أزِل أعضاء من المسودة ثم اقترح مجددًا. |
| entriesFull | One or more members already have the most entries a view group relationship can hold. | 하나 이상의 구성원이 조회 그룹 관계에 담을 수 있는 최대 항목 수에 이미 도달했습니다. | بلغ عضو واحد أو أكثر الحدّ الأقصى للإدخالات في علاقة مجموعة العرض. |
| discardInput | Discard unsaved input? (the app's existing sentence, `App.tsx:214-218`) | 저장하지 않은 입력을 버릴까요? | هل تريد تجاهل المدخلات غير المحفوظة؟ |
| discard / keepEditing | Discard / Keep editing | 버리기 / 계속 편집 | تجاهل / متابعة التحرير |
| selCleared | Selection cleared because the list changed. | 목록이 바뀌어 선택을 해제했습니다. | أُلغي التحديد لأن القائمة تغيّرت. |
| selNotLoaded | Selected: {id}. Not on the loaded pages of this view. | 선택: {id}. 이 보기에서 불러온 페이지에 없습니다. | المحدد: {id}. ليس ضمن الصفحات المحمّلة في هذا العرض. |
| membersAll | Members on {date}, all | {date} 구성원, 전체 | الأعضاء في {date}، الكل |

Items for the technical-writing review (L9):
- **Provider in KO.** 공급사 follows the owner's B03 decision (`product-glossary.md:168`). The glossary's shared label row still needs a 공급사 entry for O01; the TW review adds or confirms it.
- **KO counts.** Members are accounts, not people, so the counter is 개 ("구성원 {n}개"), never 명. The TW review checks every count in the set.
- **AR "discarded".** مُتجاهَلة follows HYB-296's use of تجاهل for discard. Whether the feminine agreement fits the draft noun (مسودة) in every position is for the TW review.
- **"Draft name" versus group name.** The copy must never present a draft name as the group's name before activation (M1).
- The glossary gains rows for view group draft, draft name, create change plan, provider filter not connected and the filtered empty state.

### Part C tests

**Go**
- Each command's rules and refusal codes from the table, including:
  - the 31-member field limit and the member cap from an injected setting;
  - a party, a group ID or a pool ID as a member;
  - a repeated member;
  - an unknown existing group;
  - discard with an open plan;
  - edit while proposed;
  - a stale `base`;
  - a new group ID collision (a test seam);
  - the open-plan limit;
  - chunking across two chunks (with injected small relationship limits);
  - a partial failure via `s.fault` (`plan-draft`, `plan-chunk`);
  - a `no_change` rejection when every member is already in the group;
  - a member whose `view_group` timeline already has 64 entries refuses `invalid_request` `members` on `draft`, `edit` and `propose`;
  - a plan above `EntriesPerPlan` (injected small limits) is stored as `rejected` with `plan_limit`, and the draft stays `open`;
  - `name` refused for a draft on an existing group, required for a new one.
- The plan has only `view_group` in `kinds`. The plans read `draft` block. Workspace isolation (direct call).
- Labels (M1):
  - before activation, the organizations, targets and plans reads show the new group's ID, and only the drafts read and the plan's `draft.name` carry the draft name;
  - after activation with injected impact evidence, the label is the activated draft's name;
  - a second, newer draft on the same group never changes that label;
  - after the activated draft's body is pruned (small `view_group_activated_kept`), the label stays (label map, N3).
- Draft state: derived from `s.rels.Plan`, also after the plan's `planMeta` is pruned from the plan store (more than 200 closed plans).
- Built-length check (N4): a member already in the group with 64 entries is accepted on `draft`, `edit` and `propose`; a member not in the group with 64 entries is refused.
- Open count (L6): open and proposed count, discarded and activated do not.
- No grant or gate change (above).
- Pruning of discarded and activated drafts against their settings.
- Flags, bounds and the cross-setting checks in `validate()`.
- Activation with injected impact evidence creates the group in the next revision, and `Union` returns its members.

**Console**
- The drafts model (state labels, allowed actions per state, batching to 31, the unsaved test).
- `onDirty` wiring: `work.tsx` passes `onDirty` to O01; O01 reports true for a changed draft form and false after save, discard or "Discard" in the notice.
- The inline notice: shown for in-page actions that do not change the URL (draft panel selection, closing a form) while dirty; "Keep editing" keeps input and focus; "Discard" discards; no notice when nothing changed. No page-level `beforeunload` exists (a test asserts the page adds no listener).
- Selection across tabs: kept by ID, the "not on the loaded pages" line, cleared with an announcement when a filter changes.
- Render tests in EN/KO/AR: the drafts section and its note, the form errors, the panel actions per state, the O02 group handoff block, the group plan lead with the draft name under the held block.

## Verification common to all parts

- `go test` with race and count=40 on `apps/api/server` and `modules/relationships`. 100% statement coverage of `apps/api/server`, and of the new module functions.
- Mutation testing on the changed Go lines, with every surviving mutant explained by its readers or killed (memory: a surviving mutant is not dead code). Clean the session GOCACHE per batch.
- linux/386 and Windows builds; gofmt and vet.
- Console `npm test` and typecheck.
- **Browser QA** against a live `hb-api` with the sample seed: EN, KO and AR at 1440, 1280 and 390. Checks:
  - no horizontal page scroll;
  - focus order scope → tabs → filters → list → detail → actions (`O01/spec.md:112`);
  - live region announcements;
  - neutral computed borders (`service-screen-style.md`).

  Scenarios:
  - **Part A:** each filter on the 2026-10-01 move; the shared view-group account in both groups; the provider line; Load more; the scan stop with a small `-org-filter-scan`.
  - **Part B:** create a change plan from each view; O02 prefilled; a stale base after another change.
  - **Part C:** create, edit and discard a draft; create its change plan; the held state in O02; discard the plan and propose again.
- Re-run the O02 runner (`qa275body.mjs`) and the M01–M05 and P01–P05 regressions, because shared CSS and `admin.ts` change.
- `verification.md` gets a "Planning cross-check" section listing covered, deferred and not-connected IDs.

## Out of scope

- **Lazy tree loading** (`O01/spec.md:31-33, 102`; UI-04). A follow-up ticket to be created: "O01 lazy tree by `parent=` and O02 choices without full load". `readChoices` in O02 also loads every member.
- **Selection across the whole filter** with a server target manifest, count and fixed scope (`O01/spec.md:102`; `ui-design.md:183`). It needs the manifest and snapshot job of the bulk pattern (`ui-design.md:270-292`), which no module provides yet.
- **The provider filter's producer:** a source account registry linked to provider connections (S01; HB-04, HB-05).
- **Group management and AM relationship change permissions** (D4). Role-scoped reads for AM, CP, BL and AU.
- **Removing members through a group draft** (Q3), renaming an activated group, and a group registry.
- **The amount, publication and ERP impact of a group change, and B09 approval.** Step 2 of HYB-296 (phase 6).
- **Paging of parties and profiles** (still capped at 500, `organizations.go:182`).

## Residuals for the owner

- Admin-only, as in HB-272. The permissions the spec names do not exist in the gate.
- The client-built tree is an open finding until the owner accepts the deferral (Q1). Only then does it become a residual with its follow-up ticket.
- The provider filter is stated as not connected.
- Activated drafts beyond `view_group_activated_kept` are pruned, and their groups then show IDs, until a group registry exists.
- Drafts, names and plans are in memory and reset on restart, like every pre-launch store. The command journal records the commands.
- A view group change is held at activation like every plan (HYB-296). Groups created by a draft cannot become effective in pre-launch until step 2 of HYB-296.
- The relationships registry never deletes plans, so registry plans grow without bound in a long-running process; `prunePlans` trims only the API's `planMeta`. This predates HYB-300 and is recorded here because group plans add to it.
- The cached index costs memory per activated revision that is read. Old revisions' indexes are built only when a pinned read asks for them; no eviction is added until measured.
- The sample data still links customers deliberately (`HB-272/plan.md:215`).

## Planning cross-check

| ID | Source | What it requires here | Covered by | Status after HYB-300 |
| --- | --- | --- | --- | --- |
| HB-01 | `requirements.md:19-35` | Workspace, legal entity and group are distinct; the profile grants nothing | D1; `entity` keyed by legal entity parties; drafts are workspace-scoped | covered |
| HB-02 | `requirements.md:37-51` | View group separate from hierarchy and payer; dated changes; group changes through validated revisions; union deduplication | D1, part C plan building, a view_group-only plan, the `Union` test | covered; the move/release acceptance is partly covered (no amount impact, held activation) |
| HB-03 | `requirements.md:53-65` | Actions separated; scope checked on the server; no leak on group movement | D4; no-grant test; server-side filters | partly covered: admin-only (residual) |
| HB-14 | `requirements.md:249-263` | Pools separate from view groups | allocation pool unchanged; a group plan touches only view_group | covered for separation; allocation runs are not in scope |
| UI-04 | `ui-design.md:547` | Filter, page and selection agree; no full load into the browser | part A server pages, scan budget | covered for accounts, groups and history; the tree and selection across the whole filter are deferred |
| GH03 | `acceptance-trace.json:794` | Duplicate view groups do not raise billed amounts | no consumer of view_group in billing (D1 fact) | not connected (no billing yet) |
| GH04 | `acceptance-trace.json:809` | Group move or release: permission, past documents, pending files | no-grant test | partly covered (permission only) |
| O01 filters | `O01/spec.md:61, 63` | As-of date, legal entity, relationship kind, unattributed, provider; server search, stable sort | part A | covered; provider not connected |
| O01 actions | `O01/spec.md:70-73` | Relationship detail; create change plan with a held revision; group draft with no grant and the not-ownership notice | parts B and C | covered; permission is admin-only |
| O01 tabs | `O01/spec.md:77, 79` | Tree → accounts → groups → history; filters and selection kept across tabs | existing tabs; URL filter parameters | covered |
| O01 handoff | `O01/spec.md:81`; `O02/spec.md:82` | Canonical ID, period and revision only; checked again on arrival | D2 | covered |
| O01 large data | `O01/spec.md:102` | Cursor reads, no full load, page versus filter selection | part A; page selection in part C | partly covered (tree and selection across the filter deferred) |
| O02 fields | `O02/spec.md:49-51` | Change kind including group; fixed manifest; expected revision | part C group plan; `base` | covered |
| Domain | `domain-contracts.md:18, 20, 67` | GroupMembership, change plan only, expected revision | D1, D2 | covered |
| Register | `screen-binding-register.md:70` | Leaving a group keeps the contract pool | view_group-only plan test | covered for the relationship part; allowance preservation is not addressed (HYB-291) |

## Owner questions (each ships with the stated default)

- **Q1. Lazy tree: acceptance needed, not a default.** HYB-300 leaves the client-built tree, which conflicts with UI-04 and `O01/spec.md:102`. The owner is asked to accept that deferral and the follow-up ticket right after part C. Without acceptance, the tree conflict blocks closing HYB-300 as complete against M4's filter scope.
- **Q2. Permission for drafts and change plans.** Default: installation administrator, as for O02. A group management action comes with role-scoped grants.
- **Q3. Removals in a group draft.** Default: a draft adds members only. Removal stays the O02 single-member "leave". If the owner wants removals in drafts, a later change adds a `leave` set to the same draft and plan.
- **Q4. Activation hold for view group plans.** Default: held like every plan (HYB-296), because the spec's four-step progress applies to every change plan. A view group may change visibility but not amounts; the owner may decide that its money impact is "not applicable", which would need its own evidence rule.
- **Q5. Settings.** Default values: `org_filter_scan` 1000 (bounds 100–10000 until measured), `view_group_drafts_open` 16, `view_group_draft_members` 1000, `view_group_drafts_kept` 200, `view_group_activated_kept` 1000. Each is changeable by start flag within its bounds.
- **Q6. Legal entity filter meaning.** Default: the dated hierarchy chain on the as-of date. An account whose payer is the entity but whose hierarchy is elsewhere does not match; the payer column shows the payer.
- **Q7. Members of a view group.** Default: source accounts only. Allocation pool IDs as members (HB-02 "원천/배분 ID 합집합") are not offered until allocation runs exist.
- **Q8. Group naming.** Default: a new group's name is drafted in the draft and becomes its label only when that draft's plan activates. A draft on an existing group cannot rename it, and no other rename path exists. The owner may want a separate group rename change (through a change plan or a group registry); that is not built.
- **Q9. Suspended legal entities in the filter.** Default: allowed, because the read can look at a past date when the entity was active (L8). The owner may prefer to hide suspended entities from the select; the server would still accept them.

## Review rounds

- **Design review round 1 (2026-11-18): FAIL.** Root's decisions on the findings, folded in above:
  - H1, accounts with no timeline of the requested kind were silently omitted: built. The server-side account set (union of non-party members across all five kinds) and the new `unattributed` meaning are in D3 and part A; tests for Accounts 01, 02, 05 and 06 and for a payer-only account are added.
  - H2, refusal fields are copied only for `invalid_request` (`modules/commands/execute.go:401-411`): `revision_conflict` carries no field; O02 compares its carried base with the reloaded active revision. `modules/commands` is not changed.
  - M1, labels: a new group's label comes only from the draft whose plan activated it; draft names appear only in draft and plan contexts; a draft on an existing group takes no name (refused); Q8 added.
  - M2, cost per request: a cached sorted index per revision and kind with `sort.Search`, the scan budget bounded at 10000, benchmarks.
  - M3, module calls and entry limits: `TimelineOf` per kind, new `IsTarget`, a per-member check against `EntriesPerTimeline`, and the `plan_limit` rejection with its sentence.
  - M4, unsaved input and selection: the leave notice and selection behaviour across tabs, with tests.
  - L1: the client-built tree is a deferral the owner must accept (Q1), not a default.
  - L2: `member` capped at 50, with a URL length test.
  - L3: `base` parsed with `ParseUint` in canonical form; tests for "-1" and 2^64.
  - L4: draft state from `s.rels.Plan`, a pruned plan record handled, activated drafts bounded by a setting.
  - L5: `settingRow` reused; cross-setting checks in `validate()`.
  - L6: the open-drafts count defined.
  - L7: the meaning of filters and chips in the groups view.
  - L8: suspended entities are allowed in the read filter, with the reason (Q9).
  - L9: TW items listed under the copy table.
  - L10: "handlers never call the gate registries" wording.
  - L11: `kind` in the handoff as part of the relationship's identity.
  - Added tests: scan budget exhausted exactly at the end, a zero-row page with `scan_stopped`, an activation between pages, labels changing while paging with `q`, the M1 label rules, a member with full entries, URL length, the leave notice, an activation between O02's load and propose, and a property test against a brute-force filtered set.
- **Design review round 2 (2026-11-18): FAIL, narrowly (no High).** Root's decisions, folded in above; root approves the plan on this fold:
  - N1, data race: the per-revision index is built inside a per-revision `sync.Once`; `apply` never copies it; concurrency tests under `-race`.
  - N2, unsaved input: the console's existing guard via `onDirty` (`work.tsx` added to parts B and C); no page-level `beforeunload`; the inline notice only for in-page actions, in the app's wording ("Discard unsaved input?"). O01's tabs are rendered by `App.tsx` and guarded by `canLeave()`. Deviation for root to check: date, filters and "Create change plan" are URL-backed here, so the app's navigation guard already covers them and the inline notice is not shown for them, to avoid a double prompt.
  - N3: a never-pruned, workspace-scoped label map; `view_group_activated_kept` prunes only draft bodies and member lists.
  - N4: `buildTimeline` runs at draft and edit time and the built length is checked; a member already in the group with 64 entries is accepted.
  - N5: `base` stays Integer in the command schema, with presence via `ctx.Fields` and the canonical `ParseUint` check; a present-but-empty test.
  - N6: the pruned-plan "unknown" branch is dropped; the unbounded growth of registry plans is a residual.
  - Nit: the `plan_limit` rejection is folded into the Seal step.

## Unknowns

- The final KO and AR wording, which the TW review decides.
- Whether the owner wants the three parts as Linear sub-issues or as three PRs on HYB-300. The Linear MCP returned 401 in this session, so the sub-tickets are not created yet; the scripts path in local tooling is the fallback.
- No external console was opened for the filter panel pattern. The render review records the references it actually opens, as HB-273 did.

## Root amendments after the Part A reviews (2026-11-18)

Part A merged under HYB-300; Parts B and C continue as HYB-318 and HYB-319, the lazy tree as HYB-320. These supersede the matching Part A text above.

1. Selection is kept in a `sel` link parameter (pattern-checked on the client), because the app remounts O01 per tab. The history view's relationship kind uses the shared `rk`; a legacy `kind` is mapped to `rk` once and removed, and "View history" keeps `rk`.
2. The groups view has no unattributed filter (members of a group are attributed to it by definition; the not-in-group list covers unattributed accounts). Search applies to the group list only; legal entity applies to members. The group-member empty state says "No members of this group match the member filters on {date}." with a member filter and "No members in this group on {date}." without.
3. `member` cannot be combined with `limit` or `after` (invalid_request). The targets read refuses `provider` (501 `unsupported_capability`) and the organization-only filters.
4. `member=` with `subject=account` accepts an account that has a timeline of any kind and returns empty entries for the requested kind (consistent with the account-set rule).
5. A refused read with a pinned revision is retried once without the pin; the revision-reset notice appears only when that retry succeeds, otherwise the refused `entity`, `member` or `sel` is removed with the bad-link notice.
6. A relationship-kind chip alone does not narrow the accounts list, so the filtered-empty sentence does not blame it (`narrowing()`).
7. Copy follows the technical-writing review: "Organization since", "{n} shown; end of list", "Groups shown: {n}; …", "All members on {date}", "Searched up to {id}.", one loading sentence per list, no count at zero ("No items shown." announced), plural bad-link notice.
8. Tables use automatic layout at full width; dates and IDs never wrap; cells wrap only between words at 768px and above; the overflow wrapper is a fallback only.

## Root amendments after the Part B reviews (HYB-318, 2026-11-18)

These supersede the matching Part B text above; details are in `../HYB-318/visual-spec.md` §12 and `../HYB-318/verification.md`.

1. O02 compares the carried base with the active revision on every plans read, blocks Propose on the client while the base is stale (`eBase`), and moves the base only through "Base on revision N", which clears the propose attempt and does not mark the form dirty.
2. The arrival block takes focus without `role=status`; programmatic focus targets show no outline.
3. A refused propose shows exactly one message: the base-refused notice when the base differs from the reloaded active revision, otherwise the plain conflict sentence.
4. "Create change plan" is absent (not disabled) on legal entities, group panels, unloaded selections and history without a timeline. The server still accepts a legal entity as a hierarchy member (HYB-322).
5. Copy follows the technical-writing review: the form is "this change" until Propose; screen and field names match actual titles; numbers and attached words never break across lines.
6. The Current aside shows its loading state until the shared choices read settles.

## Root amendments after the Part C reviews (HYB-319, 2026-11-18)

These supersede the matching Part C text above; details are in `../HYB-319/visual-spec.md` §16 and `../HYB-319/verification.md`.

1. **Separation of duties.** A draft records everyone who created or edited it; propose copies them into the plan; `relationship.activate` refuses (`prerequisite_failed`) when the activator is the proposer or any editor, and the plans read shows `own_proposal` to them.
2. **Plan snapshot.** Each plan proposed from a draft stores the draft name, group and member count at propose time, so a closed group plan keeps its title after the draft is pruned.
3. **Duplicate and invalid adds.** Before saving, the console drops IDs already in the draft and IDs that are not source accounts, and states both counts; the server stays authoritative.
4. **Refusals without fields.** `view_group.propose` refusals for members or the group arrive without a field (the registry copies only declared fields); O02 states that the draft was not accepted as it is. An unknown draft answers `permission_denied` from the registry; O01 then shows that the draft no longer exists.
5. **Hold rows.** `view_group.discard_draft` is allowed during a trust hold; `view_group.draft`, `edit` and `propose` are refused.

