---
type: ImplementationPlan
title: O02 relationship change API with drafted, validated and two-person-activated relationship plans
description: Phase 4c of the pre-launch plan (HB-257). Adds apps/api commands that draft a one-member relationship change and seal it through modules/relationships, activate a sealed plan by a second installation administrator against the expected revision and digest, and discard a plan, plus a plans read. Money, publication and ERP impact are reported as not computed in pre-launch.
status: stable
---

# Scope and ownership

[HB-274](https://linear.app/hybrid-billing/issue/HB-274) adds the API behind O02 (relationship change and impact review). Base: main `238b83c`. The O02 screen follows in HB-275.

**Ownership**
- Root (Claude Code) owns the plan, the implementation and the tests.
- An independent read-only reviewer reviews the design and the change. Relationship changes move billing attribution, so the review focuses on data accuracy and tenant boundaries (DBA style) and on permissions.
- Machine: mac_mini.

**Allowed files**
- `apps/api/server`: new `relplans.go` and its tests; registration in `admin.go`; the route in `boundary.go`.
- `docs/features/HB-274/*`.
- No module changes.

# Observed baseline

**`modules/relationships`**
- Changes are plans. `Draft(base)` must equal the active revision. `AddChunk` adds timelines. `Seal` validates the result against every rule: roles, payer chain, cycle, depth, the plan limit and no change.
  - A seal that fails returns `ErrRejected` with violations, and the plan becomes `rejected`.
  - A seal that succeeds returns a digest and a per-kind summary, and the plan becomes `sealed`.
- `Activate(digest, expected)` requires a sealed plan, the same digest, and `expected == active`, and it creates a new revision.
- `Discard` closes a draft or sealed plan.
- `Plan` and `ChangedMembers` read the result.
- At most `OpenPlans` (16 in apps/api) plans can be open per workspace.

**`apps/api`** (HB-272)
- Assembles the registry with a sample revision (number 1).
- Has the admin-only organizations read.
- Has no change path.

# Design

All commands use the installation administration permission and require a reason. The workspace is always `answer.Workspace()`, which the command context supplies.

## `relationship.propose` (target `relationships`)

**Fields**
- `kind`: one of the five kinds.
- `member`: an ID.
- `action`:
  - `set` or `end` for the exclusive kinds (`hierarchy`, `payer`, `contract_pool`, `allocation_pool`);
  - `join` or `leave` for `view_group`.
- `target`: required for `set`, `join` and `leave`.
- `from`: `YYYY-MM-DD`.

**Rules**
- `from` must be today (UTC) or later. Retroactive changes are refused in pre-launch with `invalid_request` on `from`, because finalized past periods cannot be checked yet.

**Building the new timeline** from the member's entries in the active revision:
- Entries ending on or before `from` are kept as they are.
- An entry spanning `from` is cut so that it ends at `from`. End dates are exclusive.
- What happens to entries starting on or after `from`, and what is added, depends on the action:

| Action | Later entries | Added |
| --- | --- | --- |
| `set` | dropped | one open entry `{target, From(from)}` |
| `end` | dropped | nothing |
| `join` | kept | an open entry for `target` from `from`, merged with any existing entry for it |
| `leave` | entries for `target` only are cut or dropped as above; all others are kept | nothing |

**Steps**
1. Draft plan `plan-<16 hex>` on the active revision.
2. Add the one timeline as chunk 0.
3. Seal with one chunk.
4. Record the plan metadata in apps/api: proposer, reason, creation time and the change fields.

**Outcome:** `state` (`sealed` or `rejected`), with `plan` (the ID). A rejected plan is still recorded, so its violations can be read. Both outcomes are results, not failures.

**Errors**
- A known member with no timeline of that kind can get one through `set` or `join`. This is how a new connection is made. Unknown members are refused; see the rules below.
- `end` or `leave` on a member with nothing to end gives the module's `no_change` violation.
- The open-plan limit gives `prerequisite_failed`.

## `relationship.activate` (target: plan ID)

**Preconditions**
- The plan is `sealed`.
- The caller is not the proposer (two-person rule). Otherwise `prerequisite_failed`.
- The plan base equals the active revision. Otherwise the plan is stale: `invalid_stage`. The proposer must discard it and propose again.

**Action:** `Activate(digest from Plan, expected = active)`. The outcome carries the new `revision`.

## `relationship.discard` (target: plan ID)

**Precondition:** the plan is `draft` or `sealed`; otherwise `invalid_stage`.

**Who:** the proposer or another administrator. Both are installation administrators.

## `GET /api/v1/organizations/plans`

**Admission:** admin-only, the same as the organizations read.

**Response**
- `revision` and `active_revision`.
- `impact`: `{money: "not_computed", publication: "not_computed", erp: "not_computed"}`, so the screen can show this honestly.
- `plans`, newest first, at most 50. Each plan carries:
  - `id`, `state`, `created`;
  - `proposer` and `proposer_login`;
  - `reason`;
  - `change`: `{kind, member, member_name, action, target, target_name, from}`;
  - `base` and `stale` (sealed and `base != active`);
  - `activated_revision`;
  - `kinds`: `[{kind, changed, entries_before, entries_after, earliest}]`;
  - `violations`: `[{kind, member, rule}]`;
  - `truncated`;
  - `members`, up to 50 per kind from `ChangedMembers`: `{member, name, differs: [{from, until}], before: [...], after: [...]}`.
- `can_activate`: true when the plan is sealed, not stale, and the reader is not the proposer.

**Bounds:** apps/api keeps metadata for every open plan and for the newest 200 closed plans; see the rules below. The module keeps its own records.

## Design review round 1 (FAIL) and the resulting rules

### Before `Draft`: validation

All of these checks run before the module is touched, so a refusal never leaves an open plan behind.

**Fields**
- `kind` and `action` are `commands.Enum` fields.
- `target` must differ from `member`.

**Known IDs** (F5)
- `member` must be either a known party or a member of any kind in the active revision.
- `target` depends on the kind:
  - `hierarchy`: a known party in state `active`;
  - `payer`: a known legal entity in state `active`;
  - `contract_pool`, `allocation_pool` and `view_group`: a target already present for that kind in the active revision.
- Anything else gives `invalid_request` on that field. Creating new pools or groups is a residual.

**Lookup and limits**
- The member's current entries are found by paging `Timelines` for the kind under the domain lock until the member is found or passed. This is O(n) per proposal, which is a residual.
- After the timeline is built, its entry count must not exceed the module limit, otherwise `invalid_request`.

### Building the timeline (F2)

1. Cut the entries as in the action table.
2. Add the new entry. For `join`, it is always open from `from`. Merging joins it to any earlier or later entry for the same target, so the member is in the group from `from` onward.
3. Sort the result and merge adjacent or overlapping entries for the same target.

The result always passes the module's `canonical` rule. Setting a member to the target it already has from `from` onward produces the same timeline, so Seal reports `no_change`.

Property tests check two things:
- the builder output always passes canonicalisation;
- every day before `from` is unchanged.

### After `Draft` (F1)

If `AddChunk` or `Seal` returns any error other than `ErrRejected`, the plan is discarded at once and the error mapped.

A test runs 17 failing proposals followed by one successful proposal.

### Retroactive changes at activation (F3)

`relationship.activate` refuses with `invalid_stage` when the stored `from` is before today (UTC, from the server clock). The read sets `can_activate` to false and gives `blocked: "effective_date_passed"`.

### Metadata (F4, F8)

- Metadata is keyed by workspace and plan ID.
- Metadata for open plans (`draft` and `sealed`) is never dropped. Only closed plans count towards the cap of 200.
- `Resolve` for activate and discard:
  - returns NotFound when the workspace is not the boot workspace;
  - returns NotFound when the metadata is missing.
- The two-person check fails closed when the proposer is empty.

### Stale plans (F6)

At each proposal, after field validation, sealed plans are discarded automatically in two cases:
- the base is no longer the active revision, recorded as `discarded_stale`;
- the `from` date has passed, recorded as `discarded_expired`. The read reports `open_plans` and `open_limit`.

### The read for closed plans (F7)

`ChangedMembers` is called only for `sealed` and `activated` plans. Rejected and discarded plans return their stored change and violations only. One closed plan can therefore never break the page.

### Error mapping (F9)

| Error | Code |
| --- | --- |
| `Draft` `ErrConflict` (plan ID collision) | `internal_error` |
| `Draft` `ErrLimit` (open plans) | `prerequisite_failed`. This is deliberate: an administrator can fix it by discarding plans. |
| `Activate` `ErrConflict` or `ErrState` | `invalid_stage` |
| `Discard` `ErrState` | `invalid_stage` |
| `AddChunk` `ErrInvalid`/`ErrLimit`, `Seal` `ErrConflict`/`ErrIncomplete`, `Activate` `ErrNotFound` | `internal_error`, after the plan is discarded. Validation should make these unreachable; fault injection tests them. |

# What is not included

- Computing money, publication or ERP impact. This is reported as `not_computed`.
- Approval through B09.
- Scheduling a future application. The `from` date is the effective date of the relationship itself; activation is immediate.
- Resuming failed dependency units.
- Changes that touch several members in one plan.

# Review history

- **Change review (PASS):** six Lows, all addressed:
  - L1: activation errors map to `invalid_stage`;
  - L2: fault hooks run before the module call;
  - L3: the read fails closed when the proposer is empty;
  - L4: open plans are always shown;
  - L5: refusals name the actual field;
  - L6: the property test runs every output through the module's own validation and checks the meaning on and after `from`.

  That last test found a defect: `join` on a member already in the group on `from` added nothing, so a membership ending later was not extended. `join` now always adds an open entry and merges. H1 (party-kind ordering for hierarchy targets, and member state) is a residual.
- **Design round 2 (PASS):** L1–L6. The superseded text is removed, an active state is required for parties, unreachable errors are mapped, expired plans are cleaned up, cleanup runs after validation, and the test list is extended.
- **Design round 1 (FAIL):** five Mediums, F1–F5 (orphaned drafts, the same-target merge, the retroactive check at activation, the metadata cap and the two-person rule, and unvalidated IDs), plus Lows F6–F10. All are addressed in "Design review round 1 (FAIL) and the resulting rules".

# Tests and verification

**Each action**
- Building the timeline for `set`, `end`, `join` and `leave`.
- Cutting entries at `from`.
- A plan that is rejected for a cycle, a role conflict or no change.
- A retroactive `from` refused.
- Invalid fields: kind, action or target combinations, and the date format.
- The open-plan limit.

**Activation**
- Refused for the proposer.
- Refused for a stale plan (a second plan activated first).
- Refused for a plan that is not sealed.
- A success, after which the organizations read shows the new target from `from`.

**Other paths**
- Discard.
- The read: shape, newest first, `can_activate`, `stale`, the impact fields, a member refused, and the metadata cap.

**Round-1 rules**
- Activation after `from` has passed is refused.
- The metadata cap never drops open plans.
- An unknown target, a wrong-kind target and an inactive target are refused.
- An unknown member is refused.
- Stale and expired plans are discarded automatically, and `open_plans` reports the count.
- Rejected and discarded plans appear in the read.
- An empty proposer fails closed.
- `Resolve` for another workspace returns NotFound.
- 17 failing proposals followed by a success.
- Property tests for the builder.
- Fault injection for the unreachable module errors.

**Checks**
- race at count=40;
- builds for linux/386 and Windows;
- 100% coverage of `relplans.go`;
- mutation testing of the changed lines;
- gofmt and vet.

# Residuals for the owner

- Admin-only.
- Two-person activation is between accounts, not verified people. This is the same trust boundary as M02.
- No amount, publication or ERP impact.
- No scheduling or partial resume.
- One member per plan.
- Hierarchy targets are not checked by party kind: any active party can be a parent, and the member's own state is not checked. Only cycles and depth are enforced.
- Changes cannot be retroactive in pre-launch.
- Plan metadata resets on restart.
- The module keeps every plan record it has ever made, so memory grows with each proposal. The bound is the per-user command rate limit, and the store resets on restart.
- New pools and view groups cannot be created; a change can only use targets that already exist.
- Finding one member's timeline is O(n) per proposal.

## Amendment 2026-11-18 (HYB-322)

A `member` that is a party must be an active internal unit and is allowed only for `hierarchy`; a legal entity is never a member, and payer, pool and view-group members are source accounts. Other members follow the rule above. See `../HYB-322/verification.md`.

