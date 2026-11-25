---
type: Visual Spec
title: HYB-323 Part B visual addendum for M01 select everyone matching a filter
description: Layout, type, spacing, states, copy placement, keyboard, focus, announcements and responsive behaviour for the select-matching action, the bounded count, the fixed list, the filter-list prepare and preview, the anchor dead-end and the list-closed refusal, the job-row selection basis, masking of the search text, and the new M01 settings rows, at 1440, 1280, 1024 and 390 in EN, KO and AR.
status: stable
tags: [HYB-323, HYB-294, M01, visual-spec, filter-wide-selection, bulk-revocation, ui-designer]
sources:
  - resource: plan.md
  - resource: ../HYB-294/visual-spec.md
  - resource: ../HYB-294/plan.md
  - resource: ../HYB-300/references.md
  - resource: ../../ui/screens/M01/spec.md
---

# HYB-323 Part B: select everyone matching a filter (visual addendum)

The ui-designer role (Claude, read-only) wrote this on 2026-11-18. The environment reports the model as Claude Opus 5.5 (`claude-opus-5-5`). This is a design proposal. It is not a render-review PASS and it is not owner acceptance.

Precedence:
- On behaviour, truthfulness and refusal codes, the approved plan (`plan.md` D-1 to D-9, sections 10 and 12) wins.
- On layout, this addendum wins, and it amends HYB-294 `visual-spec.md` §9 only where §1 says so.
- **[TW]** marks a string drafted here for the technical-writing and native-language review.
- **[R-Cn]** marks a decision for root (§13).

## 0. Evidence and limits

**Read directly:**
- `docs/features/HYB-323/plan.md`, all of it;
- HYB-294 `visual-spec.md`, all of it, including §9, §20 and item 13;
- the HYB-294 `plan.md` Part B copy table (`:588-609`) and D7 (`:330-357`);
- `HYB-300/references.md`;
- the merged console source on `3bdb750`: `Selection.tsx`, `BulkPrepare.tsx`, `RevocationJobs.tsx`, `filters.tsx` (lines 1-104), `Live.tsx` (lines 680-814), `bulk.css`, `bulk-copy.ts`, `RolePeriod.tsx` `Settings`, and `filter-model.ts` `queryOf`;
- M01 `spec.md` `:92-128`;
- `ui-design.md:272-283` and `:416-425`;
- the rules `service-screen-style.md`, `design-system.md`, `usability.md` and `screen-review.md`;
- `docs/design-system.md`, grep only.

**Not available:**
- **No current M01 render or screenshot.** No image exists under `docs/features/HYB-294`, `HYB-299` or `HYB-323`, so every current-state statement below comes from the source.
- **No frontend-design skill** was supplied.
- **No browsing.** No external page was opened.
- **Nothing was run:** no builds, browsers or tests. Every width is calculated.

**Facts from the source that shape this addendum:**
1. The selection bar (`section.m01-selbar`) is a wrapping flex row: `gap: 12px 24px`, `padding-block: 12px`, neutral `border-block` hairlines.
   - Children: the narrow-only "select loaded" checkbox, the count `p.m01-selcount` (`tabIndex=-1`), `div.m01-selactions` (Prepare revocation, Clear; rendered only when someone is picked), and the full-width rows `p.m01-selnote` (the `filterWide` sentence), `p.m01-limit` and `details.m01-notloaded`.
   - Below 768px, `.m01-selactions` stacks its buttons at full width.
2. The Users and Workspace membership tabs send the same users query; only the columns differ (`queryOf`, `filter-model.ts:67-79`). Invitations adds `invitation=pending` and has no checkbox column.
3. `BulkFlow` shows one `blocked` state for any refusal, with an administrator list and "Remove from selection" (`BulkPrepare.tsx:291-311`).
4. `PreviewBody` lists four exclusion reasons in a fixed order (`:86`).
5. The job table has four columns. In split mode with a job open, only the first column shows, and the state and people move into `.m01-compact` sub-lines.
6. The settings table has five columns (Setting, Value, Default, Allowed, Source) and no hint column. Below 600px it becomes labelled blocks.

## 1. Amendments to HYB-294 `visual-spec.md` §9 (binding once root accepts)

- **A-1. Action order.** HYB-294 §9 placed the select-matching action "directly after the count".
  - Revised order: count → `div.m01-selactions` (Prepare revocation, Clear, when someone is picked) → `button.m01-match-open` (the new action).
  - Reason: Prepare acts on the current selection and is the user's next step once someone is picked. The new action *replaces* the selection, so it comes after the actions that keep it.
  - With nobody picked, the new action is the only control after the count, so it still sits directly after the count.
- **A-2. Job-row chips.** Plan D-7 puts read-only chips on job rows. 44px chips double the row height in a four-column table, and in the 320px compact column they wrap to five or more lines.
  - Job rows carry the filter as one 14px text line (§9.1).
  - The read-only chips appear in the job detail (§9.2).
  - [R-C5]
- **A-3. Checkbox column while a list is fixed.** Q-B4 says the checkboxes are "not rendered". This addendum removes the whole column (`col`, `th` and `td`), using the table's existing no-check layout. Superseded in part by [HYB-324](../HYB-324/implementation.md): the owner of a filter list can now remove chosen people, in a removal mode that brings the checkbox column back for that purpose only; the dead-end names removal as a remedy and offers a way back to the list, still without a list of administrators.
  - Empty cells would leave a 56px blank column that reads as a broken control.
  - The parity rule (HYB-294 §10) holds by construction.

## 2. Composition

The bar stays one neutral band between the list toolbar and the table. Part B adds three things to it:
- one secondary button;
- one inline region (`div.m01-match`) that expands below the bar's first row;
- a fixed-list form of the bar.

Hierarchy comes only from weight (600 on numbers and the first sentence of a refusal), 14px muted secondary lines, spacing and the existing neutral hairlines. Every text line has `max-inline-size: 68ch`. Buttons are auto width at ≥768px.

No new box, tint, coloured edge, shadow, radius or icon is introduced. The only icons are the existing `close.svg` and `chevron-down.svg`, which Part B does not use.

## 3. The select-matching action (state E: explicit or empty selection)

**Shown when:**
- `can_bulk_revoke` is true;
- the view is Users or Workspace membership;
- no fixed list is open;
- the console is not in bulk-flow or step mode.

It is not rendered in Invitations or Revocation jobs. Without the right, nothing renders: no disabled control and no explanation.

**Element:** `button.live-secondary.m01-match-open`, `type=button`, `aria-expanded`, `aria-controls="m01-match"`, `min-block-size: 44px`, auto width, 16px/400.

**Label:**
- One or more removable filters applied: "Select everyone who matches these filters" (plan, `selectMatching`).
- No removable filter applied: "Select everyone in the list" (`selectAll` [TW]).

On the Membership tab the label and the matched people are the same as on Users (source fact 2), so "in the list" is truthful there.

**The count line with nobody picked** stays the existing `b.none` text. Proposed revision [TW]: "No one selected. Use the checkboxes, or select everyone who matches." Otherwise the sentence names only one of the two routes. Optional; it applies only when the action renders.

**Calculated widths (EN, 16px Plex Sans, about 8.3px per character plus 32px padding):**
- The label is about 380px.
- At 1120 (1440, no row open) row 1 holds the count, the 24px gap, the action, and Prepare and Clear when someone is picked. The longest EN mix is about 1,010px, so it fits on one line.
- At 568 (1280 with a row open) the action wraps to its own line.
- Wrapping is expected and allowed; nothing shrinks.

## 4. Inline count region `div#m01-match.m01-match`

**Container:**
- `flex-basis: 100%`;
- `display: flex; flex-direction: column; gap: 8px`;
- `padding-block: 12px 0`;
- `border-block-start: 1px solid var(--border-row)`, the same neutral hairline as the bar;
- no fill.

It renders only while the trigger is expanded.

**Contents:**
- Text paragraphs: 16px, `max-inline-size: 68ch`, `margin: 0`.
- `div.m01-match-actions`: `display: flex; flex-wrap: wrap; gap: 12px`, buttons auto width with a 44px minimum.

The count runs only when the trigger is activated. A filter change while the region is open closes it, drops the count and ignores any late answer. The region never opens on its own.

| State | Text (top to bottom) | Buttons |
| --- | --- | --- |
| Counting | "Counting people who match…" (`counting` [TW]). No number, digit, bar or percentage (`ui-design.md:420`). | Cancel (reuse the shared cancel string) |
| Complete, n ≥ 1 | Meta line, 14px muted: "Counted at {time}. The number can change before the list is fixed." [TW] | **"Select all {n} who match"** (`selectCounted`), `live-secondary`, then Cancel |
| Lower bound (neither flag) | Meta line [TW]: "Counting stopped after checking {scan} people. At least {n} match. Selecting checks everyone, and the list is refused if more than {limit} match." | "Select all who match (at least {n})", then Cancel |
| `over_limit` (with or without `complete`) | 600: `tooMany` "More people match than one revocation list allows ({limit}). Narrow the filters." The `count` value is **not shown**: it is limit + 1 and would read as a total. | Close |
| Complete, n = 0 | `noneMatch` "Nobody matches these filters now." | Close |
| Count failed | `role=alert` block: the shared `failureText` title and body | Try again, Cancel |
| Someone picked (any confirmable state) | Above the buttons, 600: `replaceConfirm` "This replaces your {n} selected people with everyone who matches." The confirm button's `aria-describedby` points to it. | as the row above |
| Fixing (command in flight) | The confirm button reads "Selecting…" [TW] with `aria-disabled=true` (not `disabled`, so focus stays on it). Cancel is hidden. | — |

**Field meanings:**
- `{time}` is the count answer's `server_time`. It is never the browser clock.
- `{scan}` is `scan_budget` and `{limit}` is `limit`, both from the count answer.

**The replace confirmation, interpreted.** Opening the region is the first activation. The confirm button, read together with the visible sentence, is the second. No third step is added [R-C6].

**Refusals of `add_matching`** replace the region's text and keep the region open:

| Cause | Text | Buttons |
| --- | --- | --- |
| Too many | `tooMany` | Close |
| None match | `noneMatch` | Close |
| Open draft | existing `openDraft` | button-link "Open Revocation jobs", then Close |
| Pending limit | existing `pendingLimit` | button-link "Open Revocation jobs", then Close |
| Step-up | the existing HYB-293 reauthentication flow; after it, the region returns to its last confirmable state | — |
| Permission or other | `failureText` | Close |

Every refusal ends with "Nothing has been revoked." (600, existing `nothingRevoked`). The refusal block is `role=alert`, `tabIndex=-1`, and gets focus. How the console tells the four `prerequisite_failed` causes apart is a root decision [R-C1]. If the cause cannot be determined, show [TW] "The list was not fixed. Too many or no people may match, or you may already have an open list. Check Revocation jobs and the filters." Never pick one of the four by guessing.

## 5. Fixed-list bar (state F)

**Shown when** the caller has an own open draft with `selection = filter`: just fixed, kept across tabs in `kept`, or recovered on first load through `GET /identity/bulk-revocations?limit=1`.

**Bar contents, in DOM and visual order:**

1. **`p.m01-selcount`** (16px, `tabIndex=-1`), the fixed line:
   - `fixed` "{matched} people fixed at {time} for these filters". `{matched}` is `<strong>` (600, `tabular-nums`). `{time}` is `fixed_at` rendered through the existing `At` (`zone=false`, date shown when not today), inside `bdi.m01-part`.
   - When the matched count differs from the last count shown in this session: `fixedDiffers` "{m} people fixed at {time} ({n} when counted)".
   - With no filter: "{matched} people fixed at {time}: everyone in the list" [TW].
2. **`div.m01-selactions`:** "Prepare revocation" (`live-secondary.m01-prepare`, unchanged) and "Discard list" (`live-secondary`, `discardList`). Clear is not rendered.
3. **`div.m01-fixedfilters`** (`flex-basis: 100%`, `display: flex; flex-wrap: wrap; align-items: flex-start; gap: 8px`):
   - lead `span` "Fixed filters:" [TW], 14px muted, `line-height: 44px`;
   - `ul.m01-chips` of `span.m01-chip.m01-chip-fixed`: not focusable, no icon, no `button`, same metrics as the Part A fixed chip (44px, `padding-inline: 12px`, neutral `--border-control`, `--surface-table-head`).
   - Chip text comes from the Part A chip model over the **stored** terms. The Membership-tab suffix is not added, because the stored terms do not depend on the tab.
   - The search chip is `Search: <bdi dir=auto>{q}</bdi>` for the preparer, which state F always is.
   - No "Clear all" here.
4. **`p.m01-fixedmeta`** (14px muted, `flex-basis: 100%`): "{Bulk revocation n} · Base revision <bdi dir=ltr class=live-mono>{fixed_revision}</bdi>" ("Base revision" [TW]). Not focusable.
5. **`p.m01-selnote`** (14px muted): `fixedNoChecks` "Discard the list to choose people one by one."
6. **`p.m01-selnote.m01-ink`** (16px, `role=status`), only when the current filters differ from the stored terms: `keepsOwn` "The fixed list keeps its own filters. Changing the filters above does not change it." It is announced once per change from "same" to "different", not on every filter change.

**While state F is open:**
- the checkbox column (A-3), the narrow `label.m01-select-loaded`, `.m01-notloaded` and `.m01-limit` are not rendered;
- the trigger is not rendered.

**Discard list** sends `bulk_revocation.discard`. On success, state E returns with an empty selection. Focus goes to the trigger and the live region says "The fixed list was discarded. Nothing was revoked." [TW]. If discard answers `invalid_stage` (the list has already ended), treat it as discarded, with the same result [R-C7].

**Calculated heights.** At 1120, row 1 is about 44px; with two to four chips the chip row is 44px; the meta and note lines are 20px each. Total bar height is about 180px, against about 70px for state E.

## 6. Prepare, step 1, for a filter list (`BulkFlow` in filter mode)

The layout is unchanged: `m01-bulk-grid` 7fr / 5fr at a container width of 880px or more, one column below. The left form is unchanged (effect radios and reason). The right `section.m01-selsum` is replaced:

- h3: "Everyone matching filters ({matched})", the basis wording plus the count [TW].
- The fixed line (§5 item 1, without "for these filters"), then the read-only chip row (§5 item 3) and the meta line (§5 item 4).
- p: **"Nothing has been revoked."** followed by [TW] "When you preview, the server checks each person on the list against these filters again and excludes anyone who no longer matches." This replaces `serverChecks` in filter mode only.
- No `details.m01-sellist`. A filter list has no client ID array.

**Sending.** One `prepare` call, with no staging progress. `p.m01-progress` (`tabIndex=-1`, focused) reads "Checking {matched} people…" [TW].

**Refusals.** Both use the existing `div.service-summary[role=alert][tabIndex=-1]` above the grid, which gets focus.

| Code | Title (h2) | Body | Then | Buttons |
| --- | --- | --- | --- | --- |
| `prerequisite_failed` (**anchor dead-end**) | `refused` "Preview refused" | `blockedFilter`, first sentence 600 through `Emph`: "This list would leave no administrator who can manage access. Narrow the filters, for example Workspace role: Member." | p 600: "Nothing has been revoked." | "Discard list" (secondary) |
| `invalid_stage` (**list closed / nobody left**) | `notPrepared` "Preview not prepared" | `listClosed` "This list can no longer be prepared: it expired, or nobody in it can be revoked now. Discard it and select again." | p 600: "Nothing has been revoked." | "Discard list" |
| `permission_denied` and others | existing failure handling | — | — | — |

In filter mode, the dead-end has no administrator list, no "Remove from selection", no "Preview again" sentence and no "Resume sending". The submit button "Preview impact" is hidden while either summary shows. Superseded in part by [HYB-324](../HYB-324/implementation.md): the owner of a filter list can now remove chosen people, in a removal mode that brings the checkbox column back for that purpose only; the dead-end names removal as a remedy and offers a way back to the list, still without a list of administrators.

After Discard list, the flow returns to the Users or Membership bar with the filters as they were. Focus goes to `.m01-match-open` and the discarded sentence is announced.

The example in `blockedFilter` must use the rendered option label of the Workspace role "Member" option (`roleName`), so that the text matches what the select shows [R-C3].

## 7. Preview for a filter list (`PreviewBody` additions)

**Order:**
1. the `m01-when` facts;
2. the scope lead;
3. `m01-counts`;
4. **new `section.m01-fixedfacts`**;
5. the preview grid, the targets, and "Not changed".

The scope lead stays the strongest line, and the fixed facts sit between the counts and the detail.

**`section.m01-fixedfacts`** (`aria-labelledby`, h3 [TW] "Fixed list", 16px/600, `margin-block: 24px 8px`):

- `dl.m01-sum` (68ch, existing two-column `dt`/`dd` grid):

  | `dt` | `dd` |
  | --- | --- |
  | "Filters" [TW] | the read-only chip row, or "Everyone in the list" [TW] |
  | "Fixed at" [TW] | `Stamp(fixed_at)` |
  | "Matched when fixed" [TW] | `{matched}`, `tabular-nums` |
  | "Base revision" | mono `bdi dir=ltr` |

- p (16px): [TW] "Of the {matched} people fixed, {targets} will be revoked and {excluded} are excluded." This makes the arithmetic visible, so the excluded people never silently disappear.
- p (600), only when k > 0: `noLongerMatch`, revised [TW]: "{k} no longer match the filters and are excluded." The plan's "…and are not in the list" is untrue: they are in the fixed list and are excluded at preview [R-C2].
- p (16px, muted is not allowed here because it is a scope statement): `notAdded` "People who match after {time} are not in this list. Select again for a new list." `{time}` is `fixed_at`.

**Exclusion summary (`section.m01-excl`).**
- For filter lists the rows are, in classification order: Self, Not found, **No longer matches the filters** (`reason no_longer_matches` [TW]), No membership, Already revoked.
- Explicit lists keep the current four rows and order, byte for byte.
- `Excluded` rows show the reason text through `b.reason`, so the new key needs its entry.

Execute, the "Continue to confirmation" button and `ConfirmDialog` are unchanged. The dialog adds no filter text.

## 8. Words about time and execution (R-B1)

Every filter fact is stated at a time: "fixed at {time}", "counted at {time}", "no longer match … at preview" (implicit in the preview's "Computed at"), and "People who match after {time} are not in this list".

No string, label, hint or announcement may say or imply that execution re-applies, re-checks or re-runs the filter, that the list "stays up to date", or that people "still match". This applies on the bar, the preview, the dialog, job rows, job detail, settings and announcements. §12 lists the forbidden forms.

## 9. Revocation jobs: selection basis

### 9.1 Job rows (`JobTable`, first cell)

After the effect sub-line, add `span.live-sub.m01-line.m01-basis` (14px):

| Basis | Text |
| --- | --- |
| `explicit` | "Chosen people" |
| `filter` | "Everyone matching filters: " followed by the term texts joined by " · " (chip-model text, not chips) |
| `not_recorded` | "Selection basis not recorded" (not muted; it carries the R4 fact) |

- Search term, preparer: `Search: <bdi dir=auto>{q}</bdi>`.
- Search term, everyone else (`search_masked`): the plain text "Search applied" [TW, capitalised `searchApplied`].
- A filter list with no terms: "Everyone in the list" [TW].
- The line wraps with `overflow-wrap: anywhere` (existing); it is never truncated with an ellipsis.

**When column:** a filter draft shows "Fixed {at}" [TW] instead of "—". Prepared and executed rows are unchanged.

**Split mode, job open:** the basis line stays in the first cell above the `.m01-compact` lines, so it shows in the compact layout too.

### 9.2 Job detail

Draft, prepared and executed details get a `dl.m01-sum` block directly under the detail head:
- **Selection basis:** the basis word.
- **Filter lists also show:** "Filters" (read-only `.m01-chip-fixed` chips), "Fixed at", "Matched when fixed" and "Base revision".
- **Placement:** for prepared lists the block is the §7 `m01-fixedfacts` inside `PreviewBody` (not duplicated). For executed and draft details it sits before `Counts` or the state line.

**Masked chip for a non-preparer:** `span.m01-chip.m01-chip-fixed` with the text "Search applied". There is no `bdi`, no tooltip, no `title`, no disclosure and no "hidden" wording. Its accessible name is its text.

### 9.3 Masking rule

- The console shows `filter.q` only when it is non-empty.
- When `search_masked` is true, it shows "Search applied", and it never renders an empty "Search: " chip.
- This is a display rule (plan D-6, R-B2). No copy may call it private, secret, protected or hidden for security.

## 10. M01 settings rows (`Settings`, Revocation jobs tab)

**Row order**, inserted among the existing rows:
1. Longest role period
2. Clock skew tolerance
3. Ends soon window
4. People per bulk revocation
5. **People per filter-wide revocation** (`bulk_revocation_max_matching`)
6. Draft and prepared list lifetime
7. Lists waiting to run per workspace
8. Bulk jobs kept
9. **People kept in revocation lists in memory** (`bulk_revocation_retained_people_max`)
10. **People examined per search page** (`user_filter_scan`)
11. **People examined per count** (`user_count_scan`)

The labels are the plan's [TW] drafts.

**Hints** add a second line inside the Setting cell: `span.live-sub.m01-line`, 14px, ink not muted, `max-inline-size: 60ch`. No new column and no tooltip.
- `bulk_revocation_retained_people_max`: the plan G3 hint, "Counts finished jobs, ended lists and lists waiting to run, each up to the people per filter-wide revocation." The user-facing text never says "pending".
- `user_count_scan` [TW]: "Past this, the count before selecting shows "at least"."
- `user_filter_scan` [TW]: "Past this, the list shows "Searched up to" and Continue searching."

**Value cells.**
- Unchanged pattern: `<bdi dir=ltr>{value}</bdi> {unit}` in `.m01-num`.
- A seven-digit default such as 2,000,000 must render without wrapping inside the number. Add `white-space: nowrap` to the `bdi` in `.m01-num`; the unit may wrap.
- Below 600px the existing labelled-block layout applies, and the hint stays under the name.

## 11. Responsive layout, RTL, keyboard and announcements

### 11.1 Widths (calculated)

| Viewport | Bar width | Notes |
| --- | --- | --- |
| 1440 | 1120 (707 with a row open) | Row 1 on one line in EN with no picks. The region and the fixed rows are full bar width; text is 68ch (about 560px at 16px). |
| 1280 | 960 (568 with a row open) | At 568 the trigger wraps under the count. Chips wrap at 8px gaps. |
| 1024 | 960, rail on top | As 1280 with no row open. With a row open the screen is in step mode and the bar is not shown; state F persists and returns with the list. |
| 390 | 358 | Bar buttons stack at full width (existing `.m01-selactions` rule, extended to `.m01-match-actions` and `.m01-match-open`). Chips `max-inline-size: 100%`. The fixed-meta revision wraps (`overflow-wrap: anywhere`). |

No width may produce horizontal page scroll. At ≥768px no button or chip stretches.

### 11.2 RTL and locales

- Logical properties only. The bar, chips and buttons flow from the right.
- Times: `bdi.m01-part` (existing `At` and `Stamp`).
- Revision and IDs: `bdi dir=ltr` mono.
- `q`: `bdi dir=auto`.
- Numbers: the existing `counted` and `people` helpers for AR plurals in `fixed`, `fixedDiffers`, `selectCounted` and `replaceConfirm`. Never concatenate numbers into AR sentences outside those helpers.
- KO: `keep-all`, which already covers `.m01-selbar`, `.m01-bulk` and `.m01-jobs`. Add `.m01-match` only if it ever renders outside the bar.
- AR: line-height 1.8. Chips and buttons use `min-block-size` only, never a fixed height.

### 11.3 Tab order (bar, state E)

count (programmatic only) → Prepare revocation → Clear → select-matching trigger → region: confirm → Cancel/Close/Try again → table. Escape inside the region closes it and focuses the trigger.

### 11.4 Tab order (state F)

Prepare revocation → Discard list → table rows' login buttons (no checkboxes). Chips and meta lines are not focusable.

### 11.5 Focus moves

| Event | Focus goes to |
| --- | --- |
| Trigger activated | stays on the trigger |
| Count result arrives | stays put |
| Fix succeeds | `p.m01-selcount` (`tabIndex=-1`) |
| Fix refused | the refusal block |
| Discard | the trigger |
| Prepare refusal | the summary |
| Dead-end Discard | the trigger on the bar |

The global 2px `--hb-focus` ring with a 2px offset is never suppressed, except for the existing rule on programmatic `tabIndex=-1` headings and paragraphs.

### 11.6 Announcements (the single polite announcer)

- "Counting people who match…";
- the result sentence (the confirm label's text, or `tooMany`, or `noneMatch`);
- after fixing [TW]: "{fixed line}. Row checkboxes are hidden while the list is fixed.";
- the discarded sentence;
- `keepsOwn` once per change from same to different filters.

Alerts use `role=alert` and are not also sent to the announcer.

## 12. Rendered-QA checks (adds to the plan's `qa323.mjs` scenarios)

1. The trigger renders only with `can_bulk_revoke` on Users and Membership. It is absent, with no placeholder, in Invitations, Revocation jobs and without the right. The label switches between the filtered and unfiltered wording.
2. While counting, the region contains no digit.
3. `over_limit` shows no number; the lower-bound state names `{scan}` and `{limit}`; complete zero shows `noneMatch` and no confirm button.
4. With picks, `replaceConfirm` is visible, and the confirm button's `aria-describedby` resolves to it.
5. State F has no `th.m01-check`, `td.m01-check` or `col.m01-col-check`, and no `input[type=checkbox]` in the table or the bar. `th` and `td` counts are equal at L, M and the stacked layout.
6. State F chips are `span`s, not focusable, with no close icon. The fixed line, meta line and `fixedNoChecks` are present. Changing a filter shows `keepsOwn` once, and the chips do not change.
7. State F survives a tab change and a reload, recovered from the server.
8. Dead-end: no administrator list, no "Remove from selection"; Discard list returns to the bar with the same filters and focus on the trigger. Superseded in part by [HYB-324](../HYB-324/implementation.md): the owner of a filter list can now remove chosen people, in a removal mode that brings the checkbox column back for that purpose only; the dead-end names removal as a remedy and offers a way back to the list, still without a list of administrators.
9. `invalid_stage` shows `listClosed`.
10. Preview for a filter list: `m01-fixedfacts` present, the `no_longer_matches` exclusion row present, the arithmetic sentence holds (targets + excluded = matched). The explicit-list preview is byte-identical to `3bdb750` (snapshot).
11. Job rows show the three basis words. For a second administrator, the search shows "Search applied" and the text of `q` appears nowhere in the DOM, including the `aria-*` and `title` attributes.
12. The settings show the four new rows with hints. `2,000,000` does not break across lines at 390.
13. Text sweep in EN, KO and AR: none of the §14 forbidden forms appear.
14. Computed-border sweep (HYB-299 §15 item 18) extended to `.m01-match *`, `.m01-fixedfilters *`, `.m01-fixedfacts *` and `.m01-basis`: every border colour is a neutral token, with no inline-start accent edge.
15. AR: times and the revision are isolated, there are no word joins around `{n}`, and chips flow from the right. KO: `keep-all` applies in the region.
16. No horizontal scroll at 1440, 1280, 1024 or 390. At ≥768, no bar button is wider than its content plus padding.
17. Keyboard: the §11.3 order is reproduced with Tab only, Escape closes the region, and focus lands as in §11.5.

## 13. Decisions for root

- **R-C1.** `add_matching` refusals cannot be told apart by code. Proposed order:
  1. read `bulk-revocations?limit=1`; if `open_draft` is true, show `openDraft`;
  2. else re-run the count with the same filters; a complete zero shows `noneMatch`, and `over_limit` shows `tooMany`;
  3. else show `pendingLimit` only if the list read can confirm the limit; otherwise show the combined §4 sentence.

  This costs one extra read. Alternatively, add a refusal field (out of plan scope).
- **R-C2.** Revise `noLongerMatch` to "…and are excluded" (TW).
- **R-C3.** The KO `blockedFilter` example uses 구성원; the existing `members` string uses 멤버. Use the role option's rendered label in all locales.
- **R-C4.** AR terminology: the existing setting name uses «القوائم قيد الانتظار» and the G3 hint uses «القوائم بانتظار التنفيذ». Pick one (TW).
- **R-C5.** A-2: text terms on job rows and chips in the detail, versus chips on rows as plan D-7 says.
- **R-C6.** The replace confirmation counts opening the region as the first activation; no extra click.
- **R-C7.** Discard answering `invalid_stage` is treated as already discarded.
- **R-C8.** The revised `none` sentence in §3 (optional).
- **New [TW] strings:**
  - the counted-at meta line;
  - the lower-bound meta line;
  - "Selecting…";
  - the combined refusal sentence;
  - the everyone-fixed line;
  - "Fixed filters:";
  - "Base revision";
  - the discarded sentence;
  - "Everyone matching filters ({n})";
  - the filter-mode `serverChecks`;
  - "Checking {n} people…";
  - "Fixed list";
  - "Filters";
  - "Fixed at";
  - "Matched when fixed";
  - the arithmetic sentence;
  - "Everyone in the list";
  - "Search applied";
  - "Fixed {at}";
  - the two scan hints;
  - the after-fixing announcement.
- **Implementation item IDs (proposed; reconcile without renumbering):**
  - `M01:users:select-matching`
  - `M01:users:match-count`
  - `M01:users:match-confirm`
  - `M01:users:match-refusal`
  - `M01:users:fixed-list`
  - `M01:users:fixed-filters`
  - `M01:users:fixed-meta`
  - `M01:users:fixed-no-checks`
  - `M01:users:fixed-keeps-own`
  - `M01:users:discard-list`
  - `M01:bulk:fixed-summary`
  - `M01:bulk:anchor-dead-end`
  - `M01:bulk:list-closed`
  - `M01:bulk:preview-fixed-facts`
  - `M01:bulk:exclusion-no-longer-matches`
  - `M01:jobs:selection-basis`
  - `M01:jobs:masked-search`
  - `M01:settings:max-matching`
  - `M01:settings:retained-people-max`
  - `M01:settings:filter-scan`
  - `M01:settings:count-scan`

## 14. Must not appear

- Coloured borders, edge strips, tints or selection outlines on the region, the chips, the fixed bar, the facts or the job rows; rounded corners, shadows, Unicode ×, arrows or emoji.
- Any total, percentage, progress bar, ETA or digit while counting; a number with `over_limit`; "of N" or "all N" before `complete` is true.
- Disabled or greyed checkboxes, an empty checkbox column, or "Select loaded" while a list is fixed.
- A disabled select-matching button, a tooltip-only explanation or a "coming soon" note; the `filterWide` sentence anywhere.
- Any wording that execution re-runs, re-applies or re-checks the filter, that the list "stays current" or "updates", or that people "still match" at execution (R-B1).
- Any claim that a filter change updates the fixed list.
- The search text for a non-preparer, in text, `title` or `aria-*`; and any wording that masking is privacy, secrecy or protection (R-B2).
- An administrator list or "Remove from selection" in the filter-list dead-end; any "remove chosen people" control on a filter list (Q4, HYB-324 is open). Superseded in part by [HYB-324](../HYB-324/implementation.md): the owner of a filter list can now remove chosen people, in a removal mode that brings the checkbox column back for that purpose only; the dead-end names removal as a remedy and offers a way back to the list, still without a list of administrators.
- Invented authentication, integrations, metrics or acceptance claims.
- A centred max-width container, or stretched buttons at ≥768px.

## 15. External references

Only the references root opened for HYB-300 (`../HYB-300/references.md`) are cited:
- **MOJ "Filter a list":** applied filters shown as tags.
- **Cloudscape "Filter patterns":** filtering applies to the full collection, not the visible page.
- **Cloudscape "Table view":** progressive loading, and no claimed total.

Adopted from them:
- read-only chips that mirror the applied-filter tags;
- the count and the fixing run on the server over the full permitted set.

The "select all N that match this search" pattern in large mail and admin consoles is a familiar convention. **No source for it was opened by me**, so nothing is adopted on its authority. Root may open and record one (for example the Cloudscape table selection guidance) before render review.

## 16. Critique against the owner's brief

**Meets:**
- full-width bar with content-width controls and 68ch text;
- neutral hierarchy only;
- separate page and filter-wide actions (`ui-design.md:421`, W03);
- waiting instead of a fake number (`:420`);
- explicit statement that filter changes do not widen the list (`:422`);
- a summary before confirmation (`spec.md:112`);
- truthful time-bound wording (R-B1);
- honest masking (R-B2);
- every new changeable default readable on the screen;
- existing IA, IDs, chip metrics and focus ring kept.

**Risks:**
- In state F the bar grows to about 180px at 1440 and about 260px at 390 before the table starts.
- Removing the checkbox column changes the table's column rhythm mid-session. This is intended (A-3), but it is visible.
- R-C1 adds a read after a refusal.
- No current render was seen; every width is calculated.
- The KO and AR strings are drafts that need TW and native review.
- Screen acceptance (FIX) belongs only to the owner.

## 17. Root decisions (2026-11-19)

Binding over the matching proposals.

1. **A-1, A-3:** accepted.
2. **A-2 / R-C5:** accepted; job rows carry the fixed filters as one text line and the read-only chips appear in the job detail. This amends plan D-7.
3. **R-C1:** accepted as proposed, with no guessing: read `bulk-revocations?limit=1` (open draft → `openDraft`); else re-run the count with the same filters (complete zero → `noneMatch`, `over_limit` → `tooMany`); else `pendingLimit` only if the list read confirms the limit; otherwise the combined sentence. No new refusal field.
4. **R-C2:** "{k} no longer match the filters and are excluded." replaces the plan string.
5. **R-C3:** the blocked example uses the Workspace role option's rendered label in every locale.
6. **R-C4:** for the technical-writing review to pick one AR term for "lists waiting to run", applied to both the setting name and the hint.
7. **R-C6, R-C7:** accepted.
8. **R-C8:** adopted: "No one selected. Use the checkboxes, or select everyone who matches." when the action renders.
9. **[TW] strings:** drafted by the implementer in EN/KO/AR from this addendum, then fixed by the technical-writing review before browser QA.
10. **References:** no new reference is opened; the convention noted in §15 is not adopted on any authority.

