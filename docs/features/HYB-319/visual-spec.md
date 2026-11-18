---
type: Visual Spec
title: HYB-319 visual spec for O01 view group drafts, page multi-selection and the O02 group draft handoff
description: Layout, type, spacing, states, focus order, locale and responsive rules for page multi-selection in the O01 accounts and not-in-a-view-group lists, the "Start a view group draft" flow, the drafts list, the draft panel with batched saves, discard, the O02 group draft handoff as the propose confirmation, draft-name labelling and the persistent not-ownership note, with decisions for root.
status: final
tags: [HYB-319, HYB-300, O01, O02, visual-spec, view-group, draft, bulk-selection, ui-designer]
sources:
  - resource: ../HYB-300/plan.md
  - resource: ../HYB-300/visual-spec.md
  - resource: ../HYB-318/visual-spec.md
  - resource: ../HYB-296/plan.md
  - resource: ../../ui/screens/O01/spec.md
  - resource: ../../../.agents/rules/service-screen-style.md
  - resource: ../../../.agents/rules/usability.md
---

# HYB-319: O01 view group drafts (visual spec)

Author: the ui-designer role (Claude, read-only), 2026-11-18. The model is claude-opus-5-5, as the environment reports it. This document is a design proposal. It is not a render PASS and it is not owner acceptance. Where this spec and the approved plan (`HYB-300/plan.md`, Part C and its amendments) differ, the difference is listed in §15 for root to decide. The plan applies until root decides.

## 0. Evidence and what was not available

**Read directly**
- The HYB-300 plan: Part C, D1, D2, Q3, Q5, Q8, the review rounds and the amendments for Parts A and B.
- The visual specs for HYB-300 Part A and HYB-318.
- The HYB-296 held-state wording.
- O01 spec lines 43–112.
- Current code under `apps/console/src/modules/organizations/pages/organizations/` (`Live.tsx`, `live.css`, `live-copy.ts`).
- O03 `LiveActions.tsx:264-278`. This is the precedent for a capped checkbox set with a stated maximum.
- The rules `service-screen-style.md` and `usability.md`.

**Inferred from the code (no render)**
- The groups view list column holds, in order:
  1. the list head "Groups and pools";
  2. the short note "A view group grants neither ownership nor access." (`viewNote`);
  3. the group table;
  4. the "Not in a view group on {date}" section, a `ul.o01-plain` of open buttons.
- The accounts view is a table with an empty end slot in `.o01-list-head`.
- The panel is `clamp(360px, 34%, 440px)` at 1280px and wider; below that it is a step-in view with Back.
- No checkbox or multi-selection exists in O01 today.

**Not available**
- No current O01 or O02 screenshots were supplied to me, and none were found under the HYB-319 worktree's `docs/features`. The sizes in §11 are derived from Part A's recorded measurements (content about 1120px at 1440 and about 960px at 1280). They must be checked on actual renders.
- I opened no external sites. The references in §13 are for root to open and record.

## 1. Composition

A view group draft is a working list that sits next to the relationship data. It is not a new place in the IA. Three surfaces carry it, and all three reuse existing O01 structures:

1. **The picker.** The accounts view table and the groups view "Not in a view group" list get row checkboxes. A selection bar sits in the list header's reserved end slot.
2. **The cart.** The draft form or draft panel opens in the existing detail panel. At 1280px and wider, while a draft form or draft is open, the panel widens to `clamp(400px, 42%, 560px)`. Picker and cart stay side by side, so the user can load more pages, check more rows and press "Add {n} to the draft".
3. **The register.** A "View group drafts" table sits in the groups view (view group kind only), between the list head or note and the group table.

Hierarchy comes only from neutral hairlines (`--border-row`), spacing, weight and explicit controls. A checked row is shown by its checkbox only. The existing `.live-selected` neutral background stays reserved for the row whose detail is open. There are no coloured edges, tinted notices, dialogs, toasts, overlays or sticky bars.

## 2. Page multi-selection

### 2.1 Where it appears

| List | Checkboxes | Bar location |
| --- | --- | --- |
| Accounts view table (any `rk`) | yes | `.o01-list-head` end slot |
| Groups view, "Not in a view group on {date}" (view group kind) | yes | a new `.o01-list-head.o01-sub-head` around the existing h3 and its meta line |
| Groups view, allocation pool kind | no | — |
| Group member list in a group panel, tree, history | no | — |

Checkboxes render only after the drafts read has loaded once for the session. That read supplies `open`, `open_limit` and the `view_group_draft_members` setting. If the drafts read fails, the checkboxes are absent and the bar slot shows nothing. The list itself still works.

### 2.2 Row checkbox

**Table (≥768px)**
- A first column `th.o01-check` with `inline-size: 44px; padding-inline: 0; text-align: center`.
- The cell holds `<input type="checkbox">` inside a 44×44 `label.o01-check-hit` (`display: grid; place-items: center`).
- The control is 20px (`inline-size` and `block-size`) with `accent-color` left at the system default. Do not set a brand colour on it.
- Accessible name: `aria-label` "Select {name} {id}". The ID is in the name because display names can repeat.
- The header cell holds a tri-state checkbox ("Select all shown rows"). Set `indeterminate` when some rows are checked.

**Stacked table (<768px)**
- The checkbox cell renders first in the card, aligned to the inline start, with a visible 14px label "Select" next to it. The 44px hit area spans the label.
- The header checkbox is not reachable in this layout. The bar's "Select all shown" link-button replaces it.

**Outside list (`ul.o01-plain`)**
- Each `li` becomes `display: flex; align-items: center; gap: 8px; min-block-size: 44px`.
- Order: the checkbox hit area (44px), then the existing open button, then the mono ID.
- The accessible name is the same as in the table.

**Checked versus current**
- A checked row gets no background, outline or weight change.
- The current (detail-open) row keeps `.live-selected` and `aria-current`.
- A row can be both.

### 2.3 Selection bar (end slot)

`div.o01-selbar`:
- `display: flex; flex-wrap: wrap; align-items: center; justify-content: flex-end; gap: 8px 16px; min-block-size: 44px`.
- It renders whenever checkboxes render, so the first check does not shift the list.

| Selection | Content, in DOM order |
| --- | --- |
| 0 | `p.o01-selcount` "None selected" (14px muted); "Select all shown" (`service-button-link`, <768px only) |
| n ≥ 1, no draft open | "{n} selected of {k} shown" (16px, 600 on the number via `<strong>`); "Clear selection" (`service-button-link`); "Start a view group draft" (`live-secondary`, 44px, auto width) |
| n ≥ 1, a draft form or open draft is in the panel | the count; "Clear selection"; "Add {n} to the draft" (`live-secondary`) |
| Open-drafts limit reached, no draft open | the count; "Clear selection"; `p` "The limit of {limit} open drafts is reached. Discard an open draft to start another." (`max-inline-size: 48ch`). The Start button is absent. |
| n = per-draft cap | the count; then "Limit reached: up to {max} members per draft." Unchecked checkboxes become `disabled`, as in the O03 precedent. The reason is stated as text in the bar, so this is not an unexplained disabled control. |
| Selection cleared by a list change | "Selection cleared because the list changed." for one render, then "None selected". It is also announced. |

- `{k}` is the number of loaded rows, the same "shown" concept as Part A's count text. The bar never shows a filter total and never offers "Select all {N} matching". That is out of scope (plan, Out of scope).
- The cap `{max}` is `view_group_draft_members` from the drafts read, minus members already in the open form or draft.
- "Select all shown" and the header checkbox select loaded rows up to the cap. If the cap cuts the set short, the bar states the cap line.

### 2.4 Lifetime of the selection

- **Kept:** across Load more and Continue searching, and across opening or closing row details.
- **Cleared, with `selCleared` announced:** when a filter, the search, the as-of date, the relationship kind, the group kind or the pinned revision changes.
- **Lost without a prompt:** on a tab change. O01 remounts per tab, and a selection is not unsaved input (§15 R5).
- **Consumed by Start:** Start moves the checked IDs into the new draft form and clears the checkboxes. This is announced as "Moved {n} accounts into the new draft." The form is then the only place where pending members exist.
- **Consumed by "Add {n} to the draft":** the IDs are added to the form's or draft's pending-add list, without duplicates, and the checkboxes clear. This is announced as "{n} added to the draft. {d} were already in it." The second sentence appears only when `d` > 0.

## 3. New view group draft (form in the panel)

**Opening the form**
- "Start a view group draft" opens the form from the picker.
- "New view group draft" in the drafts table head opens it empty.

**Container:** `section.o01-draft` with `aria-labelledby="o01-draft-title"`, inside the panel. At 1280px and wider the panel gets `data-mode="draft"`, and the grid column becomes `clamp(400px, 42%, 560px)`. Content inside is limited by `max-inline-size: 520px`.

**Order, top to bottom**

1. `h2#o01-draft-title` "New view group draft" (`tabIndex=-1`, existing panel h2 size).
2. The not-ownership note (§8), in `div.live-note-rule`, `margin-block: 12px 24px`, `max-inline-size: 60ch`.
3. The error summary (the O02 form pattern), rendered only after a failed save.
4. `fieldset` "Group" (legend 16px/600). Radios "New group" and "Existing group", each in a 44px row.
   - **New group:** the field "Draft name".
     - `input dir="auto" maxlength="80"`, `inline-size: min(100%, 440px)`, 44px high.
     - Hint (14px muted, `max-inline-size: 60ch`): "Becomes the group's name only when its change plan is activated. Until then the group is shown by its ID."
   - **Existing group:** the Part A history-pick pattern.
     - `SearchForm` "Search groups by name or ID" against the targets read.
     - Up to 10 result buttons "{label} {id}".
     - A choice collapses to "Group: {label} {id}" with the link-button "Change".
     - No name field is rendered. The server refuses `name` on an existing group.
5. `section` "Members to add".
   - h3 16px/600, with a meta line (14px muted): "{n} in this draft · up to {max}".
   - `SearchForm` "Search source accounts by name or ID" (`subject=account`, `q`).
   - Results as an `ul` of 44px rows: name, mono ID, and an "Add" `live-secondary` button at the inline end (`margin-inline-start: auto`). A row already in the list shows the text "Added" instead of the button.
   - Results use Load more as in Part A.
   - The link-button "Enter IDs" reveals:
     - a `textarea dir="ltr"`, mono, 6 rows, `inline-size: min(100%, 440px)`, with the hint "One source account ID per line.";
     - the secondary button "Add IDs".
     - Client-side, it removes blanks and duplicates and reports "{a} added, {d} already in the list". The server stays authoritative on validity.
   - The pending list is `ul.o01-draft-members`: 44px rows with a `border-block-end` hairline, each with name, mono ID and "Remove from draft" (`service-button-link`) at the inline end.
   - The list shows the first 50 rows, then "Show 50 more". No inner scroll box.
6. The reason (existing rationale component).
7. The batch line (14px muted): "Saved in batches of up to 31 members. Progress is shown while saving."
8. The action row (`.live-buttons`, gap 8px 16px): "Save draft" (primary, auto width), then "Cancel" (secondary).

**Saving in batches**
- Call 1 is `view_group.draft` with up to 31 members and the name or group.
- Each further call is `view_group.edit` with `add` of up to 31 members.
- `p.o01-draft-progress role="status"` reads "Saving: {done} of {total} members saved." The denominator is exact because the client holds the list. There is no percentage and no bar.
- While saving, "Save draft" reads "Saving…" with `aria-disabled="true"`, and "Cancel" is absent. Rows stay visible.

**Result**
- **All saved:** the panel becomes the draft panel (§5). Focus moves to its h2, and "Draft saved with {n} members." is announced.
- **First call refused:** nothing was created. The form stays, with the error summary linked to the field (`name`, `group` or `members`).
  - A `members` refusal does not name the member. The summary reads "One or more of these members cannot join this group: a member may be a group, a pool or a party, or may already have the most entries a view group relationship can hold." The summary links to the members list.
  - The plan's `entriesFull` sentence is used when the server's refusal rule identifies that case.
- **Later call refused or failed:** the draft exists. The panel becomes the draft panel in state open, with the block `role="alert"`:
  - "{done} of {total} members were saved. {rest} were not saved."
  - then the reason sentence;
  - then the button "Try the remaining {rest} again".
  - The unsaved members stay in the pending-add list, and the panel stays dirty.

## 4. Drafts table (groups view, view group kind)

**Placement and heading**
- The table sits after the groups list head and its short `viewNote`, and before the group table.
- `section.o01-drafts` with `aria-labelledby="o01-drafts-title"`, `margin-block-end: 32px`, `padding-block-end: 24px`, and a `border-block-end` hairline.
- Its head is a `.o01-list-head.o01-sub-head`:
  - start slot: h3 "View group drafts" (18px/600), with the meta line "Open drafts: {open} of {limit} · {count text}";
  - end slot: "New view group draft" (`live-secondary`). It is absent when the open limit is reached; the meta line then carries the reason.

**Columns** (automatic layout, as amended in Part A; dates and IDs never wrap):

| Column | Content |
| --- | --- |
| Draft | `button.live-open`: the draft name (new group) or the group label (existing group), in `<bdi>`. Sub-line: the mono draft ID `vgd-…`. |
| Group | "New group" with the mono group ID as a sub-line, or "Existing group" with the mono ID |
| Members | `member_count` |
| State | the state text (§6). Sub-line (14px muted) for open after a closed plan: "Last plan rejected" or "Last plan discarded" |
| Change plan | a link "Plan {plan id}" (mono, `bdi`) to the existing O02 plan locator, or "None" |

**Paging, empty state and selection**
- The first page uses `limit=10`, followed by Load more (`MoreBar`). With the open limit of 16, this keeps the group table near the top (§15 R7).
- Empty: `p.live-empty` "No view group drafts." The New button stays in the head.
- Selecting a row opens the draft panel. This selection is in-page (no URL change, per plan). It clears the selected group or member in the panel, and the inline notice applies (§9).

## 5. Draft panel

`section.o01-draft` (same container and width mode as §3).

1. h2 (`tabIndex=-1`): the draft name (new) or the group label (existing).
2. Sub-line (14px muted): "View group draft · {state}".
3. The not-ownership note (§8).
4. The state block (§6), when the state has one.
5. `dl.live-facts`:

| Term | Value |
| --- | --- |
| Draft name | new group only |
| Group | "New group" plus the mono ID, or the label plus the mono ID |
| Members | "{n} of up to {max}" |
| State | the state text |
| Created | `{author_login}` · `<time>` |
| Last changed | `<time>` |
| Change plan | a link, or "None" |
| Activated in relationship revision | `{r}` in `bdi`; activated state only |

6. **Name (new group, open state).** Below the facts, the field "Draft name" with its current value and the same hint as in §3.
7. **Members.** h3 "Members", with the meta line "{n} in this draft" and paged reads from the members read.
   - Rows as in §3.
   - In the open state each row has "Remove from draft". Pressing it marks the row with the text "Will be removed" and swaps the button to "Undo". There is no strikethrough and no colour change.
   - In other states the rows have no actions.
8. **Add members (open state only).** The same search, Enter IDs and pending-add list as §3, titled "Members to add".
   - At the cap, the area is replaced by "This draft has the most members a draft can hold ({max}). Remove members to add others, or create a change plan."
9. **Unsaved summary (dirty only).** `p` "Unsaved changes: {a} to add, {r} to remove{, name changed}." (600 lead "Unsaved changes:"), followed by "Save changes" (primary) and "Discard changes" (secondary).
   - Batching: up to 31 adds and up to 31 removes per `edit` call. The name goes in the first call.
   - Calls are ordered so the count stays between 1 and the maximum after each call (§15 R4).
   - Progress reads "Saving: {done} of {total} changes saved." Partial failure is handled as in §3.
10. **Actions** (open and not dirty), in `.live-buttons`:
    - "Create change plan": the Part B `o01-plan-link` anchor styling, secondary, 44px. It goes to `#/organization-changes?view=targets&draft=<id>&from=<day>&base=<R>` (`from` only when the date is today or later).
    - "Discard draft" (secondary).
    - While dirty, both are absent. In their place: "Save or discard your changes before creating a change plan." (14px muted).
11. **Discard confirmation (inline).** The action row is replaced by `div.live-note-rule.o01-confirm` (`tabIndex=-1`, focused):
    - `p` with `<strong>Discard this draft?</strong>`, then `discardEffect` "The draft is closed. No relationship or access changes.";
    - "Discard draft" (secondary) and "Keep draft" (secondary).
    - "Keep draft" restores the row and returns focus to "Discard draft".
    - After a confirmed discard, the panel shows the discarded state, focus moves to h2, and "Draft discarded." is announced.

## 6. States

| State | State text | Block (a neutral `live-note-rule` paragraph, `max-inline-size: 60ch`) | Editable | Actions |
| --- | --- | --- | --- | --- |
| open, never proposed | Draft | — | yes | Create change plan, Discard draft |
| open, last plan rejected | Draft | "The last change plan {link} was rejected: {rule sentence}. Change the draft and create a change plan again." For `plan_limit`, the rule sentence is the plan's `planLimit` sentence. | yes | same |
| open, last plan discarded | Draft | "The last change plan {link} was discarded. Nothing changed. You can change the draft and create a change plan again." | yes | same |
| proposed, held (`blocked` = impact) | Change plan open | "Change plan {link} was created from this draft. Activation is held: the amount, publication and ERP impact of this change has not been calculated. To change or discard this draft, discard the plan in Organization change plans first." (the HYB-296 lead wording) | no | "Open change plan" (link) |
| proposed, not held | Change plan open | "Change plan {link} is open. To change or discard this draft, discard the plan first." | no | Open change plan |
| activated | Activated | "Activated in relationship revision {r}. The members joined the group through change plan {link}." | no | Open change plan |
| activated, body pruned | Activated | adds "The member list of this draft is no longer kept." The members section is absent. | no | Open change plan |
| discarded | Discarded | "This draft was discarded on {date}. Nothing changed." | no | none |
| open-drafts limit reached | — | the bar and the drafts head line (§2.3, §4) | — | Start and New are absent |
| per-draft cap reached | — | §5 item 8; §2.3 cap line | — | — |
| drafts or members read failed | — | the existing `.live-error` with "Try again"; filters and selection kept | — | — |
| draft no longer exists (`unsupported_command`) | — | "This draft no longer exists. It may have been removed after it was discarded." The panel shows only this sentence and Back. | — | — |
| command `invalid_stage` | — | Reload the draft, then: "This draft changed before your save. It was reloaded; your unsaved changes are kept below." Pending changes are kept where the new state allows them; otherwise they are dropped and the count is stated. | — | — |
| command `prerequisite_failed` | — | the matching limit sentence | — | — |
| `dependency_unavailable` | — | "The relationships could not be read. Nothing was saved. Try again." with Try again | — | — |
| denied or revoked | — | existing; no draft surfaces render | — | — |

Every state is stated in text. The state column never relies on colour, an icon or a badge alone.

## 7. O02: the group draft handoff is the propose confirmation

This follows HYB-318 §3. The block is `div#o02-handoff.live-note-rule` (`tabIndex=-1`, focused on arrival, no live role). Its children are limited to `max-inline-size: 68ch`.

1. `<strong>Change plan for a view group draft</strong>`.
2. `dl.live-facts`: Draft name (new group only), Group (new or existing, with the mono ID), Members `{n}`.
3. The not-ownership note sentence (§8).
4. The effect sentence: "Propose creates one change plan that adds all {n} members to this group from the effective date. The plan is checked, then held: activation waits until the amount, publication and ERP impact of the change has been calculated."
5. Members: h3 "Members (first {m} of {n})", an `ul` in a CSS grid `repeat(auto-fill, minmax(240px, 1fr))` at 768px and wider, single column below. Each item is name plus mono ID.
6. The base notice and "Base on revision {a}", exactly as in HYB-318.
7. Below the block: the existing Effective from, Reason and Propose fields. Kind, change, member and target are not rendered for a draft handoff.

**After Propose**
- The impact view opens, as for propose.
- The group plan detail lead reads "Join view group {group}: {n} members from {date}". `{group}` is the label, which is the ID for a new group before activation.
- The next line, for a new group, is "Draft name: {name}" (14px, `<span class="o01-provider-label">Draft name:</span>` pattern).
- The HYB-296 held block and the four-step progress follow, unchanged.

## 8. The not-ownership note

- **The full sentence** (`notOwnership`) is persistent. It always appears in three places:
  - at the top of the new-draft form;
  - at the top of the draft panel;
  - inside the O02 draft handoff block.
- It is never dismissible and never collapsed.
- **The short `viewNote`** stays above the groups table. The two are not stacked in the same column (§15 R3).

## 9. Unsaved input

- The form or panel is dirty when the name, the group choice, pending adds or pending removes differ from the starting values. The dirty state is reported through `onDirty`.
- **URL-changing actions use the app guard only.** These are tab, sidebar, date, filters, "Create change plan" and selecting an outside member or a row (`sel`).
- **In-page actions use the inline notice.** These are selecting another draft, selecting a group row (if not URL-backed), "Cancel", "Back" (step-in, see below) and "New view group draft" while another is dirty.
  - Notice: `div.live-note-rule.o01-leave` at the top of the panel, below h2, `tabIndex=-1` and focused.
  - Content: `<strong>Discard unsaved input?</strong>`, then the buttons "Discard" and "Keep editing".
  - "Keep editing" returns focus to the last edited control, or to the form h2.
  - "Discard" performs the pending action.
- **Step-in Back below 1280px** hides the panel but keeps the form state, so it needs no notice (§15 R6). The bar then offers "Add {n} to the draft", and the list head shows the link-button "Return to the draft".
- No page-level `beforeunload`.

## 10. Focus and keyboard

1. Reading order: scope row → filter band → list head (title, meta, the bar's controls) → rows (checkbox, open button) → Load more → panel (h2, facts, form controls in DOM order, summary, actions).
2. Space toggles a checkbox. The header checkbox toggles all shown rows within the cap.
3. Start, or New: focus moves to `#o01-draft-title`.
4. Save succeeds: focus moves to the draft panel h2, and the result is announced. Save is refused: focus moves to the error summary.
5. "Add {n} to the draft": focus stays on the bar button, and the result is announced.
6. "Remove from draft" and "Undo" swap in place, and focus stays on the swapped button.
7. Discard confirmation: focus goes to the block. "Keep draft" returns it to "Discard draft".
8. Live region (`#announce`): changes to the selection count (debounced, 500ms), save progress milestones (each batch), results, and selection cleared.
9. All controls show the focus token: 2px outline, 2px offset. Programmatic focus targets (h2, notices) show no outline (`:focus-visible` only), as decided in Part B.

## 11. Widths

| Width | Picker | Panel | Notes |
| --- | --- | --- | --- |
| 1440 (content ~1120) | list ~690px with the panel in draft mode (~470px); the checkbox column is 44px; the bar fits on the title row in EN, and may wrap under the title in AR | `clamp(400px, 42%, 560px)` ≈ 470px; fields 440px | name, search and textarea never exceed 440px; buttons auto width |
| 1280 (~960) | list ~535px; the accounts table may wrap between words only, and dates and IDs never wrap | ≈ 403px; fields `min(100%, 440px)` | check that "Entry from" still does not wrap mid-date with the extra column |
| 1024 (768–1279) | single column; the bar sits on the title row; the header checkbox is present | step-in with Back; form fields keep px widths | no side-by-side |
| 390 | stacked cards with the "Select" checkbox row first; the bar wraps under the title: count on line 1, buttons on line 2, auto width | step-in; fields `min(100%, 440px)`; Save and Cancel follow the existing mobile `.live-buttons` behaviour | the O02 member grid is single column; no horizontal page scroll |

No centred container. No inputs or buttons stretched at 768px and wider.

## 12. Locales and RTL

**Layout**
- Logical properties only.
- In AR, the checkbox column sits at the right (inline start), the bar at the left (inline end), and remove buttons at the inline end.

**IDs, dates and numbers**
- IDs (`vgd-…`, `group-…`, accounts, plans) and revisions are in `<bdi dir="ltr">` mono, `white-space: nowrap`.
- Dates use `<time>` with the existing `day()`.
- The draft name input and display use `dir="auto"` and `bdi`.
- A number and its attached word never break across a line (Part B rule), for example "{n}개" and "{n} عضوًا".

**Script-specific rules**
- KO: `keep-all`. Accounts are counted with 개 (never 명), and revision is 리비전 (Part B R11).
- AR: line-height 1.8, `min-block-size` only.

**Copy drafts.** KO and AR are drafts for the technical-writing review; they are not final copy.

| Key | EN | KO | AR |
| --- | --- | --- | --- |
| selNone | None selected | 선택 없음 | لا شيء محدد |
| selCount | {n} selected of {k} shown | 표시된 {k}개 중 {n}개 선택 | تم تحديد {n} من {k} معروضة |
| selectAllShown | Select all shown | 표시된 항목 모두 선택 | تحديد كل المعروض |
| clearSel | Clear selection | 선택 해제 | مسح التحديد |
| startDraft | Start a view group draft | 조회 그룹 초안 시작 | بدء مسودة مجموعة عرض |
| addToDraft | Add {n} to the draft | 초안에 {n}개 추가 | إضافة {n} إلى المسودة |
| selCap | Limit reached: up to {max} members per draft. | 한도 도달: 초안당 구성원 최대 {max}개. | تم بلوغ الحد: حتى {max} عضو لكل مسودة. |
| openLimit | The limit of {limit} open drafts is reached. Discard an open draft to start another. | 열린 초안 한도 {limit}개에 도달했습니다. 새로 시작하려면 열린 초안을 폐기하세요. | تم بلوغ حدّ المسودات المفتوحة ({limit}). تجاهل مسودة مفتوحة لبدء أخرى. |
| draftNameLabel | Draft name | 초안 이름 | اسم المسودة |
| nameHint | Becomes the group's name only when its change plan is activated. Until then the group is shown by its ID. | 변경 계획이 활성화되어야 그룹 이름이 됩니다. 그 전에는 그룹을 ID로 표시합니다. | يصبح اسمَ المجموعة فقط عند تفعيل خطة التغيير. وحتى ذلك الحين تُعرض المجموعة بمعرّفها. |
| existingGroup | Existing group | 기존 그룹 | مجموعة موجودة |
| membersToAdd | Members to add | 추가할 구성원 | الأعضاء المراد إضافتهم |
| removeFromDraft / willRemove / undo | Remove from draft / Will be removed / Undo | 초안에서 빼기 / 빠질 예정 / 되돌리기 | إزالة من المسودة / ستُزال / تراجع |
| batchHint | Saved in batches of up to 31 members. Progress is shown while saving. | 구성원을 최대 31개씩 나누어 저장하며 진행 상황을 표시합니다. | يُحفظ الأعضاء على دفعات حتى 31 عضوًا، مع عرض التقدّم أثناء الحفظ. |
| saving | Saving: {done} of {total} members saved. | 저장 중: 구성원 {total}개 중 {done}개 저장됨. | جارٍ الحفظ: حُفظ {done} من {total} عضوًا. |
| partialSave | {done} of {total} members were saved. {rest} were not saved. | 구성원 {total}개 중 {done}개를 저장했습니다. {rest}개는 저장하지 못했습니다. | حُفظ {done} من {total} عضوًا. لم يُحفظ {rest}. |
| retryRest | Try the remaining {rest} again | 남은 {rest}개 다시 시도 | إعادة محاولة الـ{rest} المتبقية |
| pendingSummary | Unsaved changes: {a} to add, {r} to remove. | 저장하지 않은 변경: 추가 {a}개, 빼기 {r}개. | تغييرات غير محفوظة: إضافة {a}، إزالة {r}. |
| saveDraft / saveChanges / discardChanges | Save draft / Save changes / Discard changes | 초안 저장 / 변경 저장 / 변경 취소 | حفظ المسودة / حفظ التغييرات / تجاهل التغييرات |
| keepDraft | Keep draft | 초안 유지 | الإبقاء على المسودة |
| draftEffect | Propose creates one change plan that adds all {n} members to this group from the effective date. The plan is checked, then held: activation waits until the amount, publication and ERP impact of the change has been calculated. | 제안하면 구성원 {n}개 전체를 적용 시작일부터 이 그룹에 추가하는 변경 계획 하나가 만들어집니다. 계획은 검사된 뒤 보류됩니다. 변경의 금액·공개·ERP 영향이 계산될 때까지 활성화되지 않습니다. | يُنشئ الاقتراح خطة تغيير واحدة تضيف كل الأعضاء ({n}) إلى هذه المجموعة اعتبارًا من تاريخ السريان. تُفحص الخطة ثم تُعلَّق: ينتظر التفعيل احتساب أثر التغيير على المبالغ والنشر وERP. |

The remaining keys (`drafts`, `newDraft`, `notOwnership`, `newGroup`, the `st*` set, `discardDraft`, `discardEffect`, `groupLead`, `draftName`, `planLimit`, `entriesFull`, `discardInput`, `discard` / `keepEditing`, `selCleared`) come from the plan's copy table. The state sentences in §6 are EN only here; they need KO and AR drafts in the same review.

## 13. External references for root to open and record

I did not open these. Root opens them, records exact pages, adopted and rejected points in `docs/features/HYB-319/references.md`, and replaces any that are unavailable.

1. **Carbon Design System, Data table usage: batch actions and selection.** https://carbondesignsystem.com/components/data-table/usage/
   - Expected to adopt: the row checkbox column, the header tri-state checkbox, and a count with actions while a selection exists.
   - Expected to reject: Carbon's coloured batch-action toolbar that overlays the header. This spec puts the bar in the existing end slot with neutral styling.
2. **AWS Cloudscape, Table view (selection).** https://cloudscape.design/patterns/resource-management/view/table-view/ (already recorded for HB-153 and HYB-300)
   - To re-check the statement that selection applies to rows visible on the page, and the counter in the header.
   - Adopt: page-bounded selection and the counter. Reject: user preferences.
3. **Microsoft Learn, Add or remove group members in Microsoft Entra ID.** https://learn.microsoft.com/en-us/entra/fundamentals/how-to-manage-groups
   - Observe the pattern of picking many members into a group through a searchable picker with a selected list before committing.
   - Adopt: search → add → selected list → commit.
   - Reject: immediate membership change on save. Here the save only edits a draft, and membership changes only through a held change plan.

Optional fourth: GOV.UK Design System, "Add multiple things" or "Check answers" (https://design-system.service.gov.uk/patterns/check-answers/). It may support the O02 handoff as a review-before-commit step.

## 14. Must not appear

- **Selection.**
  - Selection across the whole filter: "Select all {N} matching", totals, "of N" or percentages.
  - A disabled "Start a view group draft". It is absent when it cannot apply, with the reason stated in text.
  - Coloured, outlined or tinted checked rows.
  - A sticky or floating selection bar.
- **Draft names and wording.**
  - A draft name as a group label anywhere outside draft and plan contexts: groups table, targets search results, history, tree, or the O02 targets choices.
  - A draft name without the "Draft name" label in O02.
  - Any wording that says members "join", "are added to the group" or "get access" on save.
  - "Remove from group", or any removal path into the plan (Q3). Only "Remove from draft".
  - Renaming an existing group.
  - An "Activate" action in O01.
- **Discard.** A hidden cascade that discards an open plan from O01.
- **Chrome and layout.**
  - Dialogs, modals, toasts or overlays.
  - Coloured edge strips, outlines or tinted notices.
  - Rounded corners, shadows, Unicode × or arrows, emoji.
  - Stretched controls at 768px and wider; a centred max-width container; inner scroll boxes for member lists.
- **Unsaved input.**
  - A second unsaved-input prompt for URL-backed actions.
  - A page-level `beforeunload`.
- **Labels and identifiers.** Invented permission, owner or customer labels; display names in any handoff URL.

## 15. Designer's critique and decisions for root

**Critique against the brief**
- The design covers: page selection with an exact, honest count and cap; a draft created from the picker; edit, discard and propose; truthful batch progress with a fixed denominator (O01 spec line 104); every derived state; the persistent not-ownership note where the action happens; draft-name containment (M1/Q8); and unsaved-input handling split by URL behaviour. IA and IDs are unchanged.
- **Risks:**
  - The wider draft-mode panel squeezes the accounts table at 1280. It must be checked on a render.
  - The drafts table above the group list adds height before the groups.
  - The batched save leaves a partial draft on failure. This is stated, but users may not expect it.
  - The server's `members` refusal does not name the failing member, so error precision is limited.
  - Nothing has been verified on a render yet.

**Decisions**
1. **R1. Batching.** The client chains `draft` then `edit` calls (31 per call) with progress, as this task asks. The plan's text is "add more from the draft afterwards". I recommend chaining.
2. **R2. Wide draft-mode panel** `clamp(400px, 42%, 560px)` at 1280px and wider, versus keeping the standard 360–440px panel. I recommend the wider panel. Fall back if 1280 renders cramp the accounts table.
3. **R3. Note placement.** The short `viewNote` stays above the groups table; the full `notOwnership` sentence goes in the form, the panel and the O02 handoff. The alternative is to replace `viewNote` with the full sentence.
4. **R4. Edit batch ordering.** Up to 31 adds and 31 removes per call, ordered to keep the member count between 1 and the maximum after each call. Root confirms against the `edit` rules.
5. **R5. Selection versus unsaved input.** A selection is not dirty input and is lost on a tab change. Alternatively it could be kept in a link parameter. I recommend not adding one.
6. **R6. Step-in Back.** Back keeps the form state (no notice), plus "Return to the draft". This deviates from the plan's "closing the panel" notice for the step-in Back case only. "Cancel" still asks.
7. **R7. Drafts table page size** 10 with Load more.
8. **R8. Draft column header "Draft"**, with the new or existing distinction in the Group column. This satisfies M1 because the table is a draft context. If the technical-writing review requires literal labelling, use the header "Draft name" and render existing-group drafts as "Existing group: {label}".
9. **R9. Bar copy.** The plan's single string "{n} selected on this page · Start a view group draft" is split into "{n} selected of {k} shown" and a separate button, to match Part A's "shown" vocabulary under Load more.
10. **R10. Cap behaviour.** Unchecked checkboxes become `disabled` at the cap, with the reason stated in the bar (O03 precedent). The alternative is to refuse the check with an announcement.
11. **R11. Checkbox rendering depends on a successful drafts read** (for limits). Accounts view checkboxes are hidden if that read fails.
12. **R12. Add to an existing open draft from the picker** ("Add {n} to the draft") is available only while that draft's panel is open. There is no separate "Add to draft…" chooser.
13. **R13. References.** Open the §13 sources and record them before implementation. Supply EN, KO and AR renders at 1440, 1280, 1024 and 390 for my follow-up comparison.

## 16. Root decisions (2026-11-18)

Binding over the matching proposals and over the HYB-300 plan's Part C text where they differ.

1. **R1:** the client chains `view_group.draft` then `view_group.edit` calls of up to 31 members with exact progress and partial-failure handling.
2. **R2:** the wider draft-mode panel `clamp(400px, 42%, 560px)` at ≥1280; fall back to the standard panel if the 1280 accounts table cramps (render check).
3. **R3:** adopted (short `viewNote` above the groups table; the full not-ownership sentence in the form, the panel and the O02 handoff).
4. **R4:** adopted; the implementer confirms the ordering against the `edit` rules and records it.
5. **R5:** adopted; a selection is not unsaved input and no link parameter is added.
6. **R6:** adopted; step-in Back keeps the form state with "Return to the draft"; Cancel still asks.
7. **R7:** page size 10 with Load more.
8. **R8:** column header "Draft"; the technical-writing review may change it.
9. **R9:** adopted ("{n} selected of {k} shown" plus a separate button).
10. **R10:** adopted (unchecked boxes disabled at the cap with the reason in the bar).
11. **R11:** adopted.
12. **R12:** adopted.
13. **R13:** references opened and recorded in `references.md`; renders at EN/KO/AR × 1440/1280/1024/390 are required for the render review.
14. **B2 at 1280 (render review):** the wide draft panel appears only in the groups view; in the accounts view at 1280–1439 while checkboxes show, the panel is `clamp(340px, 34%, 440px)` so the six-column kind variant fits; the checkbox column is a true 44px.
15. **Drafts list:** compact two-line rows when the section is narrower than 760px; the table where it fits.

16. **State word (technical-writing review row 20):** the open state reads "Open" / "작성 중" / "مفتوحة" (supersedes "Draft" in §6); the compact row says "No change plan" when there is none.
