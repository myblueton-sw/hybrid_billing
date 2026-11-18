---
type: ImplementationPlan
title: Live O02 relationship change and impact review in the console service mode, bound to the HB-274 change API
description: Phase 4c of the pre-launch plan (HB-257). In service mode, O02 proposes a one-member relationship change from an effective date, shows the sealed plan's changed members, violations and impact (money, publication and ERP stated as not computed), lets a second administrator activate it or anyone discard it with a reason, and lists results. Scheduling and partial resume stay out.
status: stable
---

# Scope and ownership

[HB-275](https://linear.app/hybrid-billing/issue/HB-275) binds O02 to the [HB-274](../HB-274/plan.md) API. Base: main `a66d95a`.

**Ownership**
- Root (Claude Code) owns the plan, implementation, tests and browser QA.
- The `ui-designer` role (AI, read-only) writes the visual specification and compares the final renders.
- A separate read-only reviewer reviews the design and the change. The focus is permissions (two-person activation), dates and revisions, and that nothing unknown is shown as zero.
- Machine: mac_mini.

**Allowed files**
- `apps/console/src/service/admin.ts` and its test: `plans()`, plus the three command IDs.
- `apps/console/src/modules/organizations/pages/changes/{Live.tsx,live-copy.ts,live-model.ts,live-model.test.ts,live.css}`: new. The directory name `changes` is free.
- `apps/console/src/app/screen-groups/work.tsx`: the live branch for O02 when the service context is present.
- `apps/console/package.json`: the test list.
- `docs/features/HB-275/*`.

# What the API offers

These facts are from the merged HB-274 code:
- `GET /api/v1/organizations/plans` returns `revision` (the server revision to send as `expected_revision`), `active_revision`, `open_plans`, `open_limit`, `you`, and `impact` with `money`, `publication` and `erp`, each `not_computed`.
- It also returns up to 50 plans, newest first, with every open plan included. Each plan has its change, names, state, closing reason, base, staleness, kinds, violations, changed members (with before, after and the differing ranges), `can_activate` and `blocked`.
- `relationship.propose` takes the target `relationships` and the fields kind, member, action, target and from. It answers with `business_state` `sealed` or `rejected`, and `fields.plan`.
- `relationship.activate` and `relationship.discard` take the plan ID as the target. Activate answers with `fields.revision`.
- Refusals:
  - `invalid_request`, which names the field when one is known (a reason that is too long names none);
  - `prerequisite_failed`: the caller's own proposal, or the open-plan limit;
  - `invalid_stage`: stale, expired or already closed;
  - `internal_error` with an operation ID: stopped after a change;
  - `permission_denied`.
- The organizations read (HB-272) gives the parties and each kind's members and targets, used to fill the form choices.

# Design

The screen follows the O01 live patterns:
- the full viewport width;
- on impact and approval, a list with a side panel at 1280px and wider, and stepped below that with Back;
- on result, a full-width table with the detail stepped below it at every width, because a five-column table does not fit the list rail (visual review round 1, H1);
- neutral rules only;
- EN, KO and AR copy;
- the date and revision handling from O01.

## Steps

The spec's tabs become four views in the URL (`view=` plus `plan=`):
1. **Change target** (`targets`)
2. **Impact review** (`impact`)
3. **Approval** (`approval`)
4. **Results** (`result`)

These are the tab keys already registered for O02 in `modules/organizations/screens.json`, at path `/organization-changes`.

A step indicator above the views shows the selected plan's progress in words: Proposed → Sealed or Rejected → Activated or Discarded. Colour is never the only signal.

A left list of **open plans** (sealed, newest first) appears on the impact and approval views. It shows each plan's member, change and effective date. Selecting one sets `plan=`. The count reads "N of L open" from `open_plans` and `open_limit`, and is hidden when the limit is 0 or missing.

## Change target

**Form fields**
- **Relationship**: hierarchy, payer, contract pool, allocation pool or view group.
- **Change**: set or end for exclusive kinds; join or leave for view groups. Changing the relationship resets the change.
- **Member**: a select of the parties and the members of every loaded kind (see M4 below), with names and IDs. A member the select cannot show is entered by ID through "Enter an ID", and the server checks it.
- **Target**: required for set, join and leave; hidden for end. The choices follow the API's own rules:
  - hierarchy: active parties other than the member;
  - payer: active legal entities other than the member;
  - pools and groups: targets found in the history of any loaded member of that kind.
  
  A target that does not exist cannot be created here; this is stated in the field hint.
- **Effective from**: a UTC date, today or later. The minimum is today in UTC, and the hint says the change applies from the start of that day.
- **Reason**: required, up to 256 characters, checked before sending.

**Current relationship.** Before submit, the form shows the member's current targets on the effective date and today, from the same organizations read. This is labelled "Current, from the organizations listing", not as a computed result.

**Propose**
- Propose sends the command with the current attempt (M1) and `expected_revision` from the last plans read.
- On `sealed` or `rejected`, it goes to the impact view for the new plan.
- On `invalid_request`, the named field shows the error and receives focus; the error summary lists it.
- On `prerequisite_failed`, the plans are read again and a status explains that the open-plan limit is reached, with the list.
- On `revision_conflict`, the plans read reloads and the form keeps its values.

## Impact review

For the selected plan:
- A summary: the member, the relationship, the change, the target, the effective date, the base revision, the proposer and the reason.
- **Changed members**, following M2: the dated change first, then the differing ranges ("from D", or "from D until E (not included)"), then the undated targets anywhere in the timeline before and after. On mobile these are stacked under headings.
- **Kinds**: changed members, entries before and after, earliest change.
- **Violations** for a rejected plan, in words per rule. Unknown rules show the rule code.
- **Impact on money, publication and ERP**: three rows, each "Not computed". The note says this pre-launch build does not compute them and that nothing is zero. No amounts are shown.
- A truncated plan says that more members changed than are shown.
- A stale plan says the active revision moved and the plan must be proposed again.

## Approval

- Shown for the selected sealed plan.
- Activate needs a reason and is enabled only when the state is `sealed` and `can_activate` is true. Otherwise the blocked reason is stated (an unknown value reads "Activation is not available for this plan"):
  - "You proposed this change. Another administrator must activate it.";
  - "The active revision changed after this was proposed.";
  - "The effective date has passed."
- The confirm area restates what changes and from when. Activate sends the plan ID with `expected_revision`.
- On success, the result view opens for the plan with the new revision and a link to O01 at the effective date and that revision.
- Discard is available on any sealed plan, with a reason, through a confirm step. Discarding is permanent for that plan and is stated as such.
- Refusals follow M5. `internal_error` with an operation reads: "Stopped after a change. Operation {id}. Check the results before trying again."

## Results

- Closed plans (activated, rejected, discarded), newest first.
- Columns: member, change, effective from, state, the closing reason in words (activated as revision N, rejected, discarded, discarded because stale, discarded because the date passed), proposer and created.
- Selecting a row shows its impact detail read-only.
- An activated plan links to O01 with `on` and `revision`.

## States

| State | What the screen shows |
| --- | --- |
| First load | Loading, with no numbers |
| Reload | Rows stay, with `aria-busy` |
| Empty | "No open changes" with a link to Change target; "No closed changes yet" in results |
| Denied | The unavailable block |
| Unauthenticated | Sign in again |
| Failure | An alert with Try again; the form keeps its values |
| Plan not found | "Not among the latest plans, or no longer kept", with a link to the list |

Stale responses are dropped with a request sequence. Commands disable their button while sending. Attempts follow M1 and L-C.

## External references

The ui-designer recalled these references from memory. Neither the designer nor root opened them, so they are unverified:
- **HCP Terraform run states** (https://developer.hashicorp.com/terraform/cloud-docs/run/states).
  - Adopted: the plan is fixed before apply; confirm and discard each need a comment and an explicit confirm.
  - Rejected: coloured badges.
- **GitHub deployment environments: required reviewers and "Prevent self-review"** (https://docs.github.com/en/actions/managing-workflow-runs-and-deployments/managing-deployments/managing-environments-for-deployment#required-reviewers).
  - Adopted: self-approval is blocked and stated in a sentence.
- **AWS Organizations: moving an account** (https://docs.aws.amazon.com/organizations/latest/userguide/move_account_to_ou.html).
  - Adopted: the current parent is shown next to the destination, and only existing parents can be chosen.
- **GOV.UK Design System: error summary** (https://design-system.service.gov.uk/components/error-summary/).
  - Adopted: a linked summary above the form.

## Out of scope (residuals)

These are left out and recorded as residuals:
- scheduling;
- resuming failed units;
- the B09 approval request;
- computed money, publication and ERP impact;
- filters beyond state;
- bulk targets;
- past effective dates;
- creating new pools or groups.

# Design review round 1 (FAIL) and the resulting rules

The review found five Mediums and five Lows. The plan now follows these rules:

**M1. Attempts.** A command attempt holds the request ID and idempotency key.
- A new attempt starts whenever:
  - the plans `revision` changes;
  - any field or the reason is edited;
  - the result is `stale` or `partial`.
- The same attempt is reused only to resend identical values after a network failure or `busy`.
- A propose that answers `partial` says that no plan was kept and the change must be proposed again.

**M2. Before and after.**
- The changed-member table leads with the dated change: "From D: target", or "From D: no target" for end and leave.
- It then lists the differing ranges.
- The `before` and `after` columns are labelled "Targets anywhere in the timeline, before" and "…after", because the server sends them undated.
- IDs map to party or member names, and fall back to the ID, never a blank.
- A model test covers a payer move, where before is [A] and after is [A, B].

**M3. Closed plans.** Rejected and discarded plans show "Changed members are not kept for rejected or discarded plans" in place of the table. Kind counts are shown only when `kinds` is present.

**M4. Choices.**
- The form loads every kind from the organizations read at today's date, paging up to the O01 cap of 10 pages of 99.
- Members are the parties plus the members of every loaded kind.
- Pool and group targets come from the history of every loaded member of that kind.
- Payer targets exclude the member, as all target lists do.
- Both member and target have "Enter an ID".
- If any kind did not load completely, the form says the lists may be incomplete.
- `invalid_request` becomes a field error and the options reload:
  - member: "Not found in the active revision";
  - target: "Not usable as a target for this relationship in the active revision".

**M5. Activate and discard refusals.**
- `revision_conflict`: the plans are read again and the reason is kept. The confirm step is shown again against the updated plan. Nothing is resent automatically.
- `permission_denied`: the unavailable block.
- `limited`: the wait from Retry-After.
- `busy`: Try again, with the same attempt.
- `stale`: a new attempt.
- `invalid_stage`: the plan is read again and its new reason shown.
- `refused`: the own-proposal text.
- `partial`: the operation text.

A model function maps each failure to its outcome, and the test covers every case.

**L1.** A polite live region announces sealed, rejected, activated, discarded and reloaded. Focus moves to the view heading after propose or activate, to the error summary on failure, and to the ID input when a member or target was entered by ID.

**L2.**
- `invalid_request` with no field is attached to the reason.
- The reason is checked on the client against the same limit (256 characters, `maxRationale` in apps/api/server/directory.go).
- The `action` copy also covers "too many entries for this member".

**L3.** The open-plan count and limit come from `open_plans` and `open_limit`. A refused propose reloads the plans.

**L4.** Unknown values fail closed:
- When `can_activate` is false and `blocked` is unknown, Activate stays disabled, with "Activation is not available for this plan".
- An impact value other than `not_computed` shows as "Unknown". It is never shown as 0.

**L5.** A plan in the URL that is not in the list reads "Not among the latest plans, or no longer kept".

# Design review round 2 (PASS) and the Lows applied

- **L-A.** Superseded body text was edited in place.
- **L-B.** The test list is extended; see below.
- **L-C.** After an unknown outcome (a network failure or `busy`), the next submit reuses the same attempt with the same `expected_revision`. A reload does not renew the attempt until that resend has an answer. The server then either replays the original answer or applies it once.
- **L-D.** The copy for a refused member or target follows the server's checks; see M4.
- **L-E.** The client parse is hardened:
  - a missing or empty impact value is "unknown";
  - null member elements are dropped;
  - the open count is hidden when the limit is 0.
- **L-F.** The step indicator and pinning are adjusted:
  - A rejected plan's third step is "not applicable", and an unknown state shows "unknown" steps.
  - `blocked()` and the enable condition require `sealed`.
  - The form's organization reads pin the first `relationship_revision`, as O01 does. The server re-checks on submit either way.

# Tests and verification

**Model tests** (`live-model.test.ts`)
- The action choices for each kind.
- The target candidates for each kind, from parties and members.
- Mapping each blocked reason, closing reason and violation rule to copy.
- Formatting differing ranges with the exclusive end.
- The step state for each plan state.
- The partition into open plans and results.
- The failure reaction for every code and command (M5), and when an attempt is kept or renewed (M1, L-C).
- The closed-plan member text (M3) and the dated change lead with names (M2).
- History-based pool targets and the incomplete flag (M4).
- An unknown impact or blocked value (L4).
- `invalid_request` with no field (L2).
- Null member elements (L-E).

**Client tests**
- `plans()` parsing with missing fields.
- Command IDs.
- The `invalid_request` field passed through.

**Browser QA** (Playwright against `vite preview` with the API binary and the sample seed)
- Admin 1 proposes moving `sample-acct-01` to `sample-unit-north-data` from a future date: the plan is sealed and the impact shows before and after, and "Not computed".
- Admin 1 sees the own-proposal block. Admin 2 activates, and the O01 link shows the new attribution on that date.
- A proposal with no change is rejected and its violation is shown.
- A past date is refused with the field error.
- A discard.
- A stale plan after another activation.
- A member is denied.
- EN, KO and AR at 1440 and 390, with no horizontal scroll and the right direction.
- Keyboard focus to the field in error.
- Neutral borders.
- Regression scripts qa273 (O01) and qa264 (M01).

**Other checks:** `npx tsc --noEmit`, `npm test` and `npm run build`.

# Residuals for the owner

- The FIX for O02.
- KO and AR terminology, for example 활성화 versus 적용 and 폐기.
- Impact is not computed.
- Two-person approval is between accounts, not verified people.
- There is no scheduling or resume.
- Admin-only.
