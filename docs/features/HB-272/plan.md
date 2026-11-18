---
type: ImplementationPlan
title: O01 organization and relationship read API with read-only module listings and a synthetic sample seed
description: |
  Phase 4a of the pre-launch plan (HB-257).
  - Adds read-only listings to modules/organizations (parties and billing profiles) and to modules/relationships (member timelines of a revision, paged).
  - Assembles both registries in apps/api with a synthetic sample seed.
  - Serves an organizations read with relationship targets on a chosen day.
status: stable
---

# Scope and ownership

[HB-272](https://linear.app/hybrid-billing/issue/HB-272) adds the API behind O01 (organization and relationship explorer). Base: main `297e285`. The O01 screen follows in HB-273. Relationship changes (O02) and contracts (O03) are later tickets.

**Ownership**
- Root (Claude Code) owns the plan, the implementation and the tests.
- An independent read-only reviewer reviews the design and the change. Relationships drive billing and visibility scope, so the review covers tenant boundaries and data accuracy (a DBA-style focus).
- Machine: mac_mini.

**Allowed files**
- `modules/organizations`: a new `listing.go` and its tests.
- `modules/relationships`: a new `listing.go` and its tests.
- `apps/api/server`:
  - new `organizations.go`, `orgseed.go` and their tests;
  - assembly in `server.go`;
  - the route in `boundary.go`;
  - the seed step in `seed.go`.
- `docs/features/HB-272/*`.

# Observed baseline

**modules/organizations**
- Holds workspaces (operating profile and state).
- Holds parties: legal entities and internal units, with a parent and a state.
- Holds billing profiles: issuer, buyer, payer, recipient, issuance path, currency and state.
- Holds contract revisions.
- Lookup is by ID only. There is no listing. Parties have no display names.

**modules/relationships**
- Holds dated relationship timelines per workspace, in revisions. The kinds are `hierarchy`, `payer`, `contract_pool`, `allocation_pool` and `view_group`.
- Each timeline is (kind, member, entries), where an entry is (target, interval).
- Changes go through a plan: draft, add chunks, seal (validated), activate.
- Reads (`TargetOn`, `TargetsOn`, `MembersOn`, `Segments`, `Chain`) need a known member or target. Nothing lists all timelines of a kind.

**apps/api**
- Assembles only identity, gate, commands, authpolicy and passwords.
- Neither registry is assembled.

# Design

## Module listings (read-only, additive)

**`organizations.(*Registry).Parties(workspaceID string) ([]Party, error)`**
- Returns every party of the workspace, sorted by ID. Parties are filtered by an exact match on the workspace key, not by prefix.
- Returns `ErrNotFound` for an unknown workspace.
- Returns copies, so callers cannot change the registry.

**`organizations.(*Registry).BillingProfiles(workspaceID string) ([]BillingProfile, error)`**
- The same rules: sorted by ID, `ErrNotFound` for an unknown workspace, copies only.

**`relationships.(*Registry).Timelines(workspaceID string, number uint64, kind Kind, after string, limit int) ([]Timeline, error)`**
- Lists the timelines of one kind in revision `number`, sorted by member.
- Returns the page after `after`, of at most `limit` items, where `1 ≤ limit ≤ Limits.PageSize`.
- Entries are copied.
- Lookup goes through the existing `at`, so an unknown workspace at revision 0 gives the empty revision and an empty page, as `Active` and the other reads already behave.
- Errors:
  - `ErrNotFound` for a revision that does not exist;
  - `ErrInvalid` for an unknown kind, a bad `after` or a bad limit.
- Entries are deep-copied, because revisions share entry slices.
- Every page scans and sorts that kind's timelines. This is fine for pre-launch and is a residual for scale.

**Day formatting.** `relationships.Day` gains an `ISO()` method that returns `YYYY-MM-DD`, in the same `listing.go`.

Both registries keep their own locks. No write path changes.

## Assembly and seed (apps/api)

`New` builds both registries:
- `organizations.NewRegistry(Installation)`;
- `relationships.NewRegistry(limits, s.clock)`, with limits sized for pre-launch: page size 100, depth 32, and plan limits large enough for the seed.

Creating the organizations workspace is an always-run Start step, using `s.Workspace()` with operating profile `msp`. Only this step calls `PutWorkspace`; the sample step adds parties, profiles and relationships only. With `--seed=none` this is the empty workspace, with no parties and no relationship revision, so the read answers with empty lists rather than an error.

The sample data is added by a sample-only step, `organizations`, which runs after the policy step. All data is synthetic: no real company names, no real cloud account IDs, and every ID carries a `sample-` prefix. The seed contains:
- **Workspace:** `sample-workspace`, operating profile `msp`.
- **Legal entities:**
  - the seller `sample-le-seller`;
  - two customers, `sample-le-north` and `sample-le-south`.
- **Internal units:**
  - `sample-unit-north-ops` and `sample-unit-north-data`, under north;
  - `sample-unit-south-web`, under south.
- **Billing profiles:**
  - one external-invoice profile per customer, in USD:
    - the seller is the issuer;
    - the customer is the buyer and the recipient;
    - the payer is the customer's legal entity, matching the `payer` relationship target;
  - one internal-chargeback profile:
    - issuer north;
    - buyer `sample-unit-north-data` (the same legal entity, a different party);
    - no payer and no recipient.
- **Source accounts:** six relationship members named `sample-acct-01` to `sample-acct-06`.
- **An activated relationship revision**, built through Draft, AddChunk, Seal and Activate, so validation runs:
  - `hierarchy`:
    - account to unit;
    - unit to legal entity;
    - one account moves from north-ops to north-data on 2026-10-01, to show history.
    - The dated hierarchy drives the tree. A seed test checks that each unit's hierarchy target equals its `Party.ParentID`.
  - `payer`: each account to the legal entity that pays.
  - `allocation_pool`: two accounts in a shared pool `sample-pool-shared`.
  - `view_group`: one account visible to both customers' view groups.
- **Display names:** a seed-only label map in apps/api, for example "North Holdings (sample)". The modules have no name field. Names are labels for the screen, not module data.

With `--seed=none`, the workspace and the empty registries are created and nothing else.

## `GET /api/v1/organizations`

**Admission:** the same as the other administration reads (`admitted`): a session, CSRF, and the installation administration action. This is a pre-launch decision; see the residuals.

**Workspace:** every registry call uses `answer.Workspace()` from `admitted`, never a constant. Identity allows members only in the bootstrap workspace, so a session admitted to a different workspace cannot be created in pre-launch. The isolation test therefore calls the read with another populated workspace directly; the module tests also cover isolation.

**Query:**
- `on=YYYY-MM-DD` (default: today, UTC);
- `revision`: the relationships revision to read (default: the active one). It must be a number no greater than the active revision; otherwise the request gets `invalid_request`. Paging across pages stays on one revision; the cursor response echoes it.
- `kind`, one of the five kinds (default `hierarchy`);
- `after` and `limit` page the members of that kind:
  - `limit` runs from 1 to 99 (default 50);
  - the handler asks for `limit+1` to detect a next page.

**Response:**
- `revision` (the API revision counter);
- `relationship_revision`: the relationships revision read;
- `active_revision`: the current active revision;
- `on`;
- `workspace`: `{id, profile}`;
- `parties`: id, name, kind, parent and state;
- `profiles`: id, name, issuer, buyer, payer, recipient, path, currency and state;
- `kind`;
- `members`: one row per timeline of that kind, each with:
  - `member` and `name`;
  - `targets`: `[{id, name}]` on the chosen day, sorted. This uses the meaning of `TargetsOn`, so a `view_group` member can have several targets and the other kinds zero or one.
  - `history`: every entry as `{target, name, from, until}`, with ISO dates.
    - `from` is inclusive and `until` is exclusive (the day the entry stops applying).
    - `until` is empty for an open end.
    - On a move, the old entry's `until` equals the new entry's `from`.
- `next`: the cursor.

**Bounds:**
- Parties and profiles are capped at 500 each. Above the cap the read fails closed with `dependency_unavailable`, and a later ticket adds paging.
- Members are paged by the relationships page size.

**Errors:**
- A bad `on`, `kind`, `after`, `limit` or `revision` gives `invalid_request`.
- A registry failure gives `dependency_unavailable`.

# What is not included

- Changes to parties, profiles or relationships. Relationship changes belong to O02 (HB-274/275).
- Contracts, which belong to O03.
- Per-customer scope filtering. In pre-launch only installation administrators can read, and they see the whole workspace. Real scoped roles (AM, BL, AU) need gate kinds and grants, which a later ticket adds.
- Search and lazy tree loading beyond paging. A later `parent=` parameter can use `MembersOn` to load one tree level. For now, hierarchy members are paged alphabetically, with accounts and units mixed.

# Review history

- **Change review (PASS):**
  - Fixed: L2 (`workspace` now `{id, profile}`), L3 (an active-read failure is not reported as an invalid pin) and L4 (the limit is derived from the page size).
  - Recorded: L1 (direct-call isolation test, reason above), L5 (`covering` mirrors `Interval.contains`; a drift risk) and L6 (the sample deliberately shows one account visible to both customers).
- **Design round 2 (PASS):** N1–N5 (single workspace creation, `revision` validation, the read revision echoed, module-level workspace tests, and unattributed accounts) are folded in above.
- **Design round 1 (FAIL):**
  - M1: `view_group` can have several targets, so the read returns a `targets` list.
  - M2: `until` is exclusive, with an ISO formatter and a boundary test.
  - M3: the workspace is always created, and `Timelines` reuses `at`.
  - M4: every call uses `answer.Workspace()`, with a test for a different workspace.
  - L1–L7 are addressed above.

# Tests and verification

**Module tests**
- With two populated workspaces, `Parties`, `BillingProfiles` and `Timelines` return only the requested workspace's rows.
- Listing order, copies and unknown workspaces.
- `Timelines` paging: limits, cursor, unknown kind or revision, and entries copied.

**API tests**
- The seed contents and their validation through Seal.
- The read:
  - default day and default kind;
  - the shared view-group account showing both groups;
  - the boundary day and the day before it;
  - a session for a different workspace sees nothing;
  - a pinned revision;
  - limit 99, and a limit of 100 refused;
  - a day before and after the move, showing the target change;
  - each kind;
  - paging;
  - invalid queries;
  - a member refused;
  - no session;
  - `--seed=none`;
  - a fault while reading.

**Checks**
- race with count=40;
- linux/386 and Windows builds;
- 100% coverage of the new apps/api files and the module listings;
- mutation testing of the changed lines;
- gofmt and vet.

# Residuals for the owner

- O01 is readable only by installation administrators in pre-launch. Role-scoped access (AM, BL, AU, customer scope) is not built.
- Names come from a seed label map, not from module data.
- The hierarchy listing is flat, not a lazy tree.
- A member appears only under kinds where it has a timeline. The screen shows an empty `targets` on the chosen day as unattributed, not as zero.
- Each `Timelines` page scans every timeline of that kind.
- The sample data is synthetic and resets on restart. It deliberately includes one account visible to both customers' view groups, and a pool shared across customers, as examples. These are not recommended patterns.
