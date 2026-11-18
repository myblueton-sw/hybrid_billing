---
type: ImplementationPlan
title: Live O01 organization and relationship explorer in the console service mode, bound to the HB-272 read API
description: Phase 4b of the pre-launch plan (HB-257). In service mode, O01 shows four read-only views over one pinned relationship revision and an as-of day. The tree view lists legal entities, units and accounts. The accounts view joins hierarchy and payer. The groups view covers view groups and allocation pools. The history view shows one member's history. Unattributed and incomplete states are stated in words.
status: stable
---

# Scope and ownership

[HB-273](https://linear.app/hybrid-billing/issue/HB-273) binds O01 to the [HB-272](../HB-272/plan.md) read API. Base: main `8c4fa4f`.

**Ownership**
- Root (Claude Code) owns the plan, implementation, tests and browser QA.
- The `ui-designer` role (AI, read-only) wrote the visual specification and compares the final renders.
- A separate read-only reviewer reviews the design and the change. The focus is data accuracy: dates, revisions and attribution.
- Machine: mac_mini.

**Allowed files**
- `apps/console/src/service/admin.ts` and its test: `organizations()`.
- `apps/console/src/modules/organizations/pages/organizations/{Live.tsx,live-copy.ts,live-model.ts,live-model.test.ts,live.css}`: new.
- `apps/console/src/app/screen-groups/work.tsx`: the live branch for O01 when the service context is present.
- `apps/console/package.json`: the test list.
- `docs/features/HB-273/*`.

# Design

The layout follows the designer specification. It uses the identity live-screen patterns:
- the full viewport width;
- a list with a side panel at 1280px and wider, and stepped below that with Back;
- neutral rules only, no decorative coloured borders;
- EN, KO and AR copy.

## Shared scope

**The scope form**
- An as-of date field (UTC, native date input), plus Show and Today. The date is applied on submit.
- Facts: the relationship revision (marked "active" when it is the active one), and the workspace with its operating profile.
- Reload, which re-reads the active revision.

**One revision for every view**
- The first read omits `revision`. Every later read, including other views and other kinds, passes `revision=R`.
- If a response reports `active_revision > R`, a neutral status notice offers "Show revision N". It resets the pages and the selection, then reads at N.
- If a pinned read fails with `invalid_request`, the screen restarts from the active revision and announces it.

**Paging**
- Each kind is read with `limit=99`, following `next` automatically for up to 10 pages, all at R.
- Past the cap, or after a mid-paging failure, the view is marked "Incomplete" and offers Load more or Continue loading from the last cursor.
- Counts read "N loaded". They are never presented as totals.

**Stale responses** are dropped with a request sequence whenever the date, kind or revision changes.

## Views

**tree** (kind `hierarchy`)

How a node's parent is found:
- a party that is a hierarchy member: its target on the day;
- a party that is not a member: `party.parent`;
- a member that is not a party: a source account, placed under its target.

The tree is laid out as follows:
- Roots are parties without a parent.
- Children are sorted by kind (units, then accounts), then by name, then by ID.
- Members with an empty `targets` list go in a final "No organization on {date}" section, and the section is omitted when there are none. While the load is incomplete, the section is shown with the "may be on pages not yet loaded" note.
- Targets that match no known node become a node "Target not in this workspace listing".

How the tree behaves:
- It is a nested disclosure list with toggle buttons that carry `aria-expanded`, plus Expand all and Collapse all.
- Selecting a node opens the panel. The panel shows the parent on the day, the state, "Since" for an account, and the billing profiles where the party is the issuer, buyer, payer or recipient. A "View history" link opens the history view.

**accounts**
- The screen reads `hierarchy`, then `payer` at the same R. Rows are the members that are not parties, joined by member.
- **Organization** shows one of three things:
  - the day's target;
  - "Unattributed": a hierarchy timeline exists but has no target on the day;
  - "No hierarchy relationship": the account has no hierarchy timeline at all.
- **Payer** follows the same pattern.
- **Since** is the start of the covering hierarchy entry.
- An "Only unattributed" filter.
- Accounts are not linked to billing profiles, because the API defines no such link.

**groups**
- A choice between view group and allocation pool. Contract pool belongs to O03 and is not shown here.
- Members are inverted by target. A member with several targets appears in each of its groups.
- Members without a target on the day are listed under "Not in a … on {date}".
- A persistent note: a view group grants neither ownership nor access, and pool membership does not confirm an allocation result.

**history**
- A kind select and a member select, filled from the members loaded for that kind.
- A table of target, From, "Until (not included)", and "Applies" for the entry that covers the as-of date.
- A note explains the exclusive end.
- A member that is not found in the loaded pages is reported as not found. The API has no member filter, which is a residual.

## States

| State | What the screen shows |
| --- | --- |
| First load | Loading, with no numbers |
| Reload or paging | Rows stay, with `aria-busy` |
| Empty | Distinct messages for no data, no activated revision, and a kind with no members |
| Denied | The unavailable block |
| Unauthenticated | Sign in again |
| Failure | An alert with Try again; the date and the view are kept |
| Invalid date | A field error |

## External references

The designer specification recalls three consoles, but it did not open them. Root verifies two of them and records the exact URLs before the PR:
- the AWS Organizations console (hierarchy with expand and collapse, ID beside name, list and hierarchy views);
- Azure management groups (tenant root, Type column, expand all).

# Design review round 1 (FAIL) and the resulting rules

These rules apply to every view.

- **Loading progress.**
  - Each kind tracks whether its load completed and the last loaded member, which is the cursor.
  - A member whose ID sorts after the cursor of an incomplete kind is "Not loaded yet". It is never shown as "No … relationship", "Unattributed", a registry fallback parent, or "not found" in history.
  - Parties whose relationship may sit on an unloaded page are listed under "Not placed yet", not under a parent.
- **Current registry versus the relationship revision.**
  - Parties and billing profiles come from the current registry. They are not pinned and not dated, and the panel says so.
  - A party that is not a hierarchy member shows "Registered parent (current registry)", never "Parent on {date}".
  - If the server state (`revision`) changes between pages or kinds, a notice says the pages may not match and offers Reload.
- **Several targets.**
  - When hierarchy or payer has more than one target on the day, the row reads "Conflicting attribution (N targets)" and lists all of them. Nothing is picked.
  - Members in conflict go in their own tree section.
- **Tree completeness.**
  - Every loaded party and member appears exactly once, either in the tree or in one of these sections: unattributed, conflict, not placed, or cycle.
  - A party that is a hierarchy member with no target on the day is unattributed. It does not fall back to its registry parent.
  - Unknown targets and parents become "Target not in this workspace listing" nodes.
  - Roots are all nodes without a parent, including internal units.
  - Nodes that cannot be reached because of a cycle go under "Cycle detected".
- **Accounts.**
  - Rows are the union of non-party members from hierarchy and payer.
  - Accounts with no relationship at all cannot be shown, because there is no account registry. This is a residual.
- **Restarting after `invalid_request`.**
  - Only a pinned read restarts once, without the revision.
  - If that read also fails, the error is shown, so the screen cannot loop.
- **Dates.**
  - The client always sends `on`.
  - "Applies" uses from ≤ on < until.
  - The account column is "Entry from" (the start of the covering entry), not "Since".
- **Tree accessibility.**
  - Each node has a toggle with `aria-expanded`, `aria-controls` and the node name, plus a separate select button with `aria-current`.
  - Collapsed children are not rendered.
  - Expand all and Collapse all keep focus on the button.
  - Revision and incompleteness notices use `role=status`.

# Design review round 2 (FAIL) and the fixes

**N1: roots during an incomplete hierarchy load.** Every party that is not a hierarchy member is checked against the cursor, with or without a registry parent. One not yet loaded goes under "Not placed yet" and is never shown as a root.

**N2: hidden subtrees.** The unattributed, conflict, not-placed and cycle sections render as expandable subtrees. Their loaded children therefore stay visible, and the "exactly once" test walks every section.

**Lows**
- One restart per user action.
- "Only unattributed" filters strictly.
- Filtered and incomplete states have their own empty messages.
- `aria-current` is on the select button, and `aria-controls` is set only when a node is expanded.
- Rows in every section can be selected.
- "Entry from" is empty for a member in conflict.

# Tests and verification

**Unit tests**
- The model:
  - tree building, including party parents by day, accounts under units, unattributed members and unknown targets;
  - the account join with both kinds of absence;
  - group inversion with a member in several groups;
  - the history entry that covers the date;
  - date validation;
  - the revision-change detection.
- The client query: already added.

**Browser QA against a live `hb-api`**
- Each view:
  - the default date;
  - the date before and on the 2026-10-01 move, where Account 02 changes unit;
  - the shared view-group account appears in both groups;
  - the pool;
  - the history of Account 02, showing the exclusive end;
  - "View history" navigation.
- The member denied view.
- EN, KO and AR at 1440 and 390, checking:
  - no horizontal scroll;
  - focus moves to the detail heading when stepping, and Back returns it;
  - computed borders are neutral.
- Rerun the M01–M05 QA, because shared CSS is not changed but is used.

# Residuals for the owner

- Admin-only in pre-launch, inherited from HB-272.
- The tree is built on the client from flat pages, and is complete only when the whole load completes.
- The API has no member filter for history and no `parent=` parameter.
- No link between accounts and billing profiles.
- The contract pool is left for O03.
- KO and AR terms (지급 주체, 미귀속, 내부 비용 청구 and others) need a glossary and technical-writing review.
- The sample data deliberately links customers.
- Screen acceptance (FIX) is the owner's alone.
