---
type: Visual Spec
title: HYB-294 Part A visual spec for the M01 users filter band, server search and paging
description: Layout, type, spacing, states, keyboard order and responsive behaviour for the M01 server-side search, filters, chips, count certainty, Show more, scan stop, list-changed and start-again states, departed rows and the HYB-299 selection bar under filter changes, at 1440, 1280, 1024 and 390 in EN, KO and AR. Room is left for the HYB-323 select-matching action.
status: stable
tags: [HYB-294, M01, visual-spec, filters, search, paging, ui-designer]
sources:
  - resource: plan.md
  - resource: ../../ui/screens/M01/spec.md
  - resource: ../HYB-300/visual-spec.md
  - resource: ../HYB-300/references.md
  - resource: ../HYB-299/visual-spec.md
---

# HYB-294 Part A: M01 filters, search and server paging (visual spec)

The ui-designer role (Claude, read-only) wrote this on 2026-11-18. The environment reports the model as Claude Opus 5.5 (`claude-opus-5-5`). This is a design proposal. It is not a review PASS and it is not owner acceptance. On copy, states and truthfulness the approved plan (`plan.md` D1–D7 and the copy table) wins. On layout this spec wins. **[TW]** marks a string the layout needs that the plan does not define. **[R-n]** marks a decision for root (§16).

## 0. Evidence and limits

**Read directly:**
- `docs/features/HYB-294/plan.md`, all 845 lines;
- `docs/ui/screens/M01/spec.md`, all of it;
- `docs/features/HYB-300/visual-spec.md` and `references.md`;
- `docs/features/HYB-299/visual-spec.md`;
- on the `hb294` worktree: `users/Live.tsx`, `Selection.tsx`, `bulk.css`, and the first 32 lines of `bulk-copy.ts`;
- grepped only: `live.css`, `live-copy.ts`, the class names in O01 `filters.tsx`, and `app/screen-groups/identity.tsx`;
- the rules `service-screen-style.md`, `usability.md` and `design-system.md`.

**Not available:**
- **No current M01 screenshots or renders were supplied, and none exist under `docs/features`** (a glob for `*.png` found nothing). Everything I say about the current state comes from reading the source, not from images.
- **No frontend-design skill was supplied** with this task.
- **I opened no external page**, because I have no browsing tool.
- I ran no builds, browsers or tests. Every width below is calculated, not measured.

**Current state, from the source:**
1. The order inside `.live-users` is: announcer, `Conditions`, then `div.live-list` (toolbar with h2, count, "Loaded at", Reload and Create user; then the selection bar; the table; "Show more users"), then `aside.live-panel` only when something is open (HYB-299 R-1).
2. The count is `w.count(n)` over the loaded rows. Invitations are filtered on the client by `rowsFor`.
3. The paging button calls `api.users(next, 100)`, and new rows are appended with no merge by ID.
4. The selection bar always shows the `filterWide` note: "Selecting every person in a search result is not available yet. Load more rows to add them."
5. **Inferred risk, not verified:** at ≥1280 with a row open, `.live-users` is a two-column grid, and no rule spans `section.m01-conditions` across both columns. Auto-placement may put the list into the panel column when a condition shows. The new band must not repeat this (§2).
6. The screen title and the tabs are rendered by the shell, above `.remaining-security`. A scope line inside `LiveUsers` therefore renders **after** the tabs.

## 1. Composition

The band follows the same pattern as O01 so the console stays consistent. It is a full-width band above the list, never a left column. Fields keep their own fixed widths and wrap; they never stretch.

M01 order: scope line → conditions → filter band (fields, sign-in method line, applied chips) → list toolbar (title, count certainty, order line, Loaded at, Reload, Create user) → selection bar → table → paging area. The panel is unchanged.

Hierarchy comes only from neutral 1px hairlines, spacing and weight. There are no boxes, tints, coloured edges or shadows, and the radius is 0.

**Why the band spans the whole content width:** with a row open at 1280 the list is 568px, and seven controls would take five rows. Across 960–1120px they take two or three. When a row opens, the band stays the same width, so filters do not jump.

## 2. Structure and grid placement

```
div.live-users.m01-live
  p.visually-hidden[role=status]                 existing single announcer
  p.m01-scopeline                                (§3)  grid-column:1/-1
  section.m01-conditions                         existing; add grid-column:1/-1 (§0 item 5)
  section.m01-filters[aria-labelledby]           (§4)  grid-column:1/-1
  div.live-list.m01-list                         existing
     .live-toolbar  (count, order line)           (§6)
     p.m01-listnote / div.m01-restart?           no; these go in the paging area (§7)
     section.m01-selbar                          existing (§9)
     table                                        existing columns (§10)
     div.m01-more                                (§7)
  aside.live-panel                               existing
```

- In `bulk.css`, set `[data-security-screen="M01"] .m01-scopeline, .m01-filters, .m01-conditions { grid-column: 1 / -1 }`. Do not touch `live.css`.
- **The band is not rendered:**
  - in step mode with a detail or step open (`stepping`);
  - in the bulk flow (`data-bulk != none`), because the frozen selection is not driven by filters;
  - in the denied state.

## 3. Fixed scope line (`p.m01-scopeline`)

- Content: `<span class="m01-scopelabel">` with the plan's "Workspace:" prefix (14px, muted), then the value (16px/600, ink).
- `margin-block: 0 16px`, `max-inline-size: 68ch`. It is not focusable, has no control and no chevron.
- The value is the workspace name, **if the users read returns one**. Otherwise it is the workspace ID in `<bdi dir="ltr" class="live-mono">`. Do not invent a name [R-1].
- It shows in all four tabs, including Revocation jobs.
- Recorded deviation: the spec order is title → scope → tabs (`spec.md:114`), but the shell puts the scope after the tabs. This is the same deviation root accepted for O01 (HYB-300 §15 D2) [R-2].

## 4. Filter band at ≥1280px

### 4.1 Container

`section.m01-filters`:
- `aria-labelledby="m01-filters-title"`;
- `h2#m01-filters-title` "Filters" [TW, reuse the O01 string], 16px/600, `margin-block: 0 12px`;
- `padding-block-end: 16px`, `margin-block-end: 24px`, `border-block-end: 1px solid var(--border-row)`. No top border and no fill.

### 4.2 Fields

`div.m01-filter-fields`: `display:flex; flex-wrap:wrap; gap:16px 24px; align-items:flex-end`. Each field is a `.service-field` with its label above the control. Every control has `min-block-size: 44px`; no control has a fixed height, so AR at line-height 1.8 still fits.

DOM order equals visual order:

| # | Control | Element | Inline size | Applies |
| --- | --- | --- | --- | --- |
| 1 | Search: plan "Search login, email or user ID" | `form[role=search]` holding `input type=search dir=auto autocomplete=off` and secondary "Search" [TW]. 8px gap. Too long above 64 UTF-8 bytes: inline field error plus summary, no truncation | input `304px` | Enter or Search only, never per keystroke |
| 2 | Account status | `select`: All [TW] · Active · Suspended · Departed | `184px` | on change |
| 3 | Membership in this workspace | `select`: All · Active · Suspended · "Role period ended, suspension pending" · "No membership" (final strings: plan copy table and `filter-copy.ts`) | `336px` | on change |
| 4 | Workspace role | `select`: All · Administrator · Member | `184px` | on change |
| 5 | Role period | `select`: All · No end date · Has an end date · "Ends within {n} days" (n from `warning_days`) · Role period ended | `242px` | on change |
| 6 | Sign-in setup | `select`: All · Password set · Set-password link outstanding · Reset link outstanding · No password and no link | `288px` | on change |
| 7 | Installation administrators only | `label.live-check`, 20px checkbox, 44px row | auto (about 268px in EN) | on change |

- Controls use fixed `inline-size: Npx` with `max-inline-size: 100%`; fields use `inline-size: min-content` so labels wrap to the control width (`text-wrap: balance`).
- **Wrapping.** Fields keep the fixed inline sizes in the table (Membership 336px, Role period 242px after the TW option text) and wrap in DOM order; they never stretch or shrink to change the row split. The row composition is language-dependent and is not specified. Rendered at 1440 the band is two field rows (1–3 / 4–7) in EN; at 1280 and 1024 it is at most three field rows in EN, KO and AR (observed: EN 1–2 / 3–5 / 6–7; KO and AR 1–3 / 4–7). Acceptance criteria at ≥768px: fixed widths, Search first, controls bottom-aligned per row, no closed value clipped in any locale, at most three field rows at a 960px content width, no horizontal scroll.
- Field widths come from the longest EN option plus 52px for padding and the arrow. KO is shorter. AR must be checked in the render, so that the closed select does not truncate "انتهت مدة الدور (بانتظار التعليق)" (§15).

### 4.3 Sign-in method line (`p.m01-method`)

- It sits below the fields: `margin-block-start: 12px`, `max-inline-size: 68ch`.
- Content: `<span class="m01-method-label">` "Sign-in method" [TW] (14px/600, ink), then the plan sentence "Everyone signs in with a password set at this installation. Single sign-on (SAML, OIDC) and directory sign-in (LDAP) are not connected." (14px, muted). Expect two to three lines.
- It is plain text: no control, no disabled select, no tooltip, no icon, no badge.

### 4.4 Applied filters (`div.m01-applied`)

- `margin-block-start: 16px`. It renders only when at least one chip exists.
- Content, on one wrapping line: the lead "Applied filters:" [TW, reuse the O01 string] (14px, muted), then `ul.m01-chips` (`flex-wrap`, `gap: 8px`, `list-style: none`, `padding: 0`), then "Clear all filters" (plan copy) as `.service-button-link` with a 44px hit area.
- **Removable chip:**
  - `button type=button`, with the text "{Field}: {value}" plus `packages/design-system/assets/icons/close.svg` at 16px (never ×);
  - accessible name "Remove filter {Field}: {value}" [TW, reuse the O01 string];
  - `min-block-size: 44px`, `padding-inline: 12px 8px`, `border: 1px solid var(--border-control)`, `background: var(--surface-subtle)`, radius 0, 14px text;
  - hover changes the neutral background only.
- **Chip texts:**
  - Search: "Search: {q}". `q` goes in `<bdi dir="auto">`, not `ltr`, because KO and AR search text is valid.
  - The selects use "{Field}: {option}". Period uses the n-day wording.
  - The checkbox chip reads "Installation administrators only".
  - In the Membership tab, a role or period chip gets the suffix [TW "(people with a membership)"] (plan D6).
- **Fixed chip on Invitations:** `span.m01-chip-fixed` "Pending invitations" (plan). Same metrics, no icon, not a button, not focusable. It is always first, and is not counted in "{n} applied".
- **Clear all filters:**
  - It renders only when at least one removable chip exists.
  - It never removes the fixed chip and never changes the selection.

## 5. Filter band below 1280px (1024, 768, 390)

- `button.m01-filters-toggle` replaces the visible h2. The h2 stays in the DOM, visually hidden, so the region keeps its name.
  - `aria-expanded`, `aria-controls="m01-filter-body"`.
  - Text: "Filters", or "Filters · {n} applied" [TW], where n counts removable filters only.
  - 600 weight, 44px, a leading design-system chevron that mirrors in RTL and rotates when open.
- **Collapsed** on the first load of each tab [R-3, matching O01 D8].
- **Chips and Clear all render below the toggle, outside the collapsible body.** So applied filters are always visible, including the fixed Invitations chip.
- **768–1279px:** the ≥1280 widths and wrapping apply.
- **Below 768px:**
  - fields use `flex-basis: 100%` and `inline-size: min(100%, 440px)`;
  - the Search button sits on its own line at auto width;
  - the method line uses the full 358px.
- After a change, the body stays open, focus stays on the control, and the live region announces (§11).

## 6. List toolbar: count certainty and order

The existing `.live-toolbar` and `.live-heading` stay. The `p.live-meta` lines are 14px and muted. In order:

1. **Count line**, which replaces `w.count` and `invitationCount`. The plan has these two strings:
   - "{n} shown; more may exist" when `next` is present, including after a scan stop;
   - "{n} shown; end of list" when `next` is empty.

   It never shows "of N", a total, a percentage or the word "all". In the Invitations tab the noun stays "people" unless TW supplies an invitation variant [TW].
2. **Order line** [TW "By login. Departed people last."]. It makes the fixed order visible, because there are no sort controls.
3. "Loaded at {time}" (existing).

The Reload and Create user positions are unchanged. Reload loads page one again with the same filters.

## 7. Paging area (`div.m01-more`)

`div.m01-more`: `display:flex; flex-wrap:wrap; align-items:center; gap:8px 16px; margin-block-start:16px`. It replaces the bare `.live-more` button.

| Condition | Content |
| --- | --- |
| `next`, not scan-stopped | Secondary "Show more users" (existing), auto width, 44px, at inline-start. While loading, the label reads "Loading…" with `aria-disabled="true"`, and the rows are kept. |
| `scan_stopped` with rows | `p[role=status]`: plan "Searched up to {login}." (login in `<bdi dir="ltr">`), or for a departed cursor plan "Searched up to user {id}." (mono). Then a separate secondary "Continue searching". "Show more users" is **not** rendered, so there is one forward action. Text `max-inline-size: 68ch`. |
| `scan_stopped`, zero rows on the first page | It replaces the table: [TW "No matches yet."] plus the "Searched up to …" sentence and "Continue searching". The filtered-empty state is never claimed. |
| List ended | No button. The count line says "end of list". |
| Next page failed | Rows are kept. `role="alert"` [TW, reuse O01 "The next page could not be loaded. The rows shown are kept."] plus "Try again". |
| `list_version` changed on a later page | Rows are kept. `p[role=status].m01-listnote` (68ch) with the plan "changed" text (three sentences; the first is 600). Then secondary "Reload", which runs the toolbar's action. The Show more or Continue control stays available. |
| 409 `revision_conflict` on `after` | Rows are kept. `div[role=alert].m01-restart`: plan "The list changed while you were paging. Start again from the first page." (first sentence 600) and secondary "Start again". Show more is removed. Focus moves to "Start again". |

- **Rows merge by `user_id`.** A repeated ID replaces its row in place. A person who departs while the user is paging is never shown twice.
- **After Show more or Continue,** focus moves to the first appended row's `.live-open`. If nothing was appended, focus stays on the button.
- **At 390,** a compact copy of the count line sits next to the button (`span.live-meta`).

## 8. Per tab

| Tab | Band | Notes |
| --- | --- | --- |
| Users | All seven controls, the method line and chips | — |
| Workspace membership | All seven controls, the method line and chips | Role and period chips carry the membership suffix. |
| Pending invitations | All seven controls, the method line, and chips with the fixed "Pending invitations" chip first | The filtered-empty state here uses the plan's `filteredEmpty` text, not `noInvitations`, whenever any removable filter is set. |
| Revocation jobs | No band and no toggle | `p.m01-note` (14px, muted, 68ch) with the plan's `jobsNote`, **only when at least one filter is in the link** [R-4]. No chips. The scope line stays. |

**TW check:** `jobsNote` names "Users, Pending invitations and Workspace membership", which are the panel titles. If the shell tab labels differ, the note must use the tab labels the user actually sees.

## 9. Selection bar under filter changes (HYB-299, unchanged component)

- **Selection is kept by user ID** across filter changes, Show more, Reload and tab changes. A filter change never adds anyone.
- **The header checkbox** and the narrow `m01-select-loaded` checkbox keep their "Select the {n} loaded people" name. Here `n` is the rows of the current filtered result, so selecting from it never widens past what is visible.
- **People outside the new result** are listed in the existing `details.m01-notloaded` disclosure, with the existing "not in the loaded rows" wording. It never says "do not match", because the console only knows "not loaded" (plan D6).
  - Compute and render the disclosure only when the list status is `ready`. While a filtered read is in flight, keep the previous disclosure; do not show every selected person as not loaded and then pull them back.
- **The `filterWide` note stays unchanged in Part A.** It is still true. HYB-323 removes it.
- **Room for HYB-323:** the "Select everyone who matches these filters" action goes inside `div.m01-selactions` directly after the count. The fixed-list line goes in the bar as a `flex-basis: 100%` row and reuses the `.m01-chips` chip markup, read-only. **Part A renders no placeholder, empty slot or disabled control for it.**
- **Invitations:** no checkbox column. The existing `p.m01-selline` stays.

## 10. Table: columns, departed rows and header/cell parity

**Columns:** the column sets and containers are unchanged from HYB-299 §3.2.
- **L** (list ≥880px: 1440 or 1280 with no row open, and 1024): ☐ · User · Account status · Workspace role · Role period · Sign-in setup.
- **M** (520–879px: 707 or 568 with a row open, and 768): ☐ · User (with the sign-in sub-line) · Account status · Role and period.
- **Membership tab:** L is ☐ · User · Workspace role · Role period · Membership status · Installation administrator. M is ☐ · User · Membership status · Role and period.
- There are no sortable headers and no arrows (the order line in §6 carries the order).

**Departed rows** come last, in server order, with no separator row, so no `colspan` row breaks parity.
- The user cell's `.live-open` button shows the user ID in `<bdi dir="ltr" class="live-mono">`. Under it, `span.live-sub` carries plan "Departed (login removed)". This applies in every tab, because the Membership tab has no account status column [R-5].
- Every other cell is rendered, never blank:
  - "—" plus a visually hidden word where a value is absent;
  - the role cell keeps "No membership";
  - the period cell keeps "—" plus the visually hidden "No role period".

**The membership value `ended`** (period over, not yet swept) shows as the existing period "Role period ended" (600) in the period cell. The membership status cell needs a word for it [TW]; it must not read "Active".

**Parity rule:** at L, M and the stacked layout, the number of visible `th` equals the number of visible `td` in every row. That includes departed rows, rows with no membership and the caller's own row (`m01-self`). Hidden duplicates use `display:none`, as in HYB-299.

**HYB-331 amendment (2026-11-19):** on the Membership tab the membership status column already says "No membership", so a row with no membership no longer repeats it in the role cell. The Workspace role cell (L) and the role line of the Role and period cell (M and stacked) show "—" plus the visually hidden "No role" [TW] (KO "역할 없음", AR "بلا دور"); the membership status cell keeps "No membership". The Users and Invitations tabs, which have no membership status column, and the row detail keep "No membership" in the role value. This replaces the "role cell keeps 'No membership'" bullet above for the Membership tab only.

## 11. States

| State | Rendering |
| --- | --- |
| Initial load | The band and chips render at once from the link. Existing `role=status` text. No skeletons and no numbers. |
| Filter or search changed | Clear the table body. Show [TW "Loading users…"] (status). No dimming or overlay. The last request wins. The selection bar stays. |
| Filtered empty | `p.live-empty`: plan "No people match these filters in your permitted scope." Then "Clear all filters" (link-button). On Invitations with only the fixed chip, use the existing `noInvitations`. |
| Unfiltered empty | Existing "No users yet." |
| Scan stopped, zero rows | §7 |
| First-page error | Existing `.live-error`, with "Try again". The band and chips stay. |
| Next-page failure, list changed, restart | §7 |
| Bad link parameter | The field resets. `role=status`, with the HYB-300 bad-link notice (the plural form), above the fields. |
| Denied or revoked | Existing; no band and no scope line. |
| Open detail person is not in the new result (wide) | [R-6] Recommended: close the panel, keep focus on the filter control, and announce [TW "The person you had open is not in the filtered list."]. Do not show stale row data. |

Every state is text. Colour never carries a state alone.

## 12. Keyboard, focus and announcements

**Tab order:**
1. shell tabs;
2. conditions;
3. the toggle (below 1280);
4. search, then Search;
5. status, membership, role, period, sign-in setup, the administrators checkbox;
6. chips, then Clear all;
7. Reload, Create user;
8. the selection bar;
9. the table (header checkbox; for each row, its checkbox and then its login button);
10. the paging area;
11. the panel.

The method line is not focusable.

**Focus moves:**
- Removing a chip moves focus to the next chip, else the previous chip, else the search input (desktop) or the toggle (narrow).
- Clear all moves focus to the search input or the toggle.

**Live region (single, polite):**
- after each first-page read: [TW "List updated: {count line}. {k} filters applied."];
- when k > 0 changed: append "{k} selected people are not in the loaded rows";
- after a chip removal: [TW "Filter removed. {count line}."].

**Focus ring:** all controls use the global 2px `--hb-focus` ring with a 2px offset. Do not suppress it.

## 13. Locales and RTL

- **KO:** add `.m01-filters` and `.m01-more` to the existing `keep-all` selector.
- **AR:** keep line-height 1.8. Chips and controls use `min-block-size` only.
- **RTL:**
  - the band and chips flow from the right;
  - the close icon sits at inline-end;
  - the chevron mirrors;
  - the paging button sits at inline-start (the right);
  - logical properties only.
- **Isolation:**
  - logins, IDs and `{id}` in "Searched up to user {id}" use `bdi dir=ltr`;
  - `q` uses `dir=auto`;
  - dates use the existing `Stamp` and `day()`;
  - no arrow glyphs.
- AR plurals use the existing `counted` helpers.

## 14. Breakpoints (calculated, not measured)

| Viewport | `.live-users` | Band | List |
| --- | --- | --- | --- |
| 1440 | 1120 | open; two rows (950 / 1,036); method line; chips | L at 1120 with no row open; M at 707 with a row open |
| 1280 | 960 | open; at most three field rows (language-dependent, §4.2) | L at 960; M at 568 |
| 1024 | 960, rail on top | collapsible; when open, at most three field rows (language-dependent, §4.2) | L at 960; step detail |
| 390 | 358 | collapsible; fields `min(100%,440px)`; buttons at auto width | stacked; count repeated beside Show more |

No width may produce horizontal page scroll. At ≥768px no input or button stretches to fill a row.

## 15. Rendered-QA checks specific to this spec

1. At all four widths in EN, KO and AR, no field's closed value is truncated. Check the longest options in AR: membership "ended" and sign-in "set link".
2. The band is 1120 / 960 wide whether or not a row is open. The list does not move into the panel column, including with a condition showing (§0 item 5).
3. `th` and `td` counts are equal at L and M and in the stacked layout, with departed, no-membership and self rows present.
4. Chips and toggle hit areas are ≥44×44. The fixed chip is not focusable and is not removed by Clear all.
5. The computed-border sweep from HYB-299 §15 item 18 is extended to `.m01-filters *` and `.m01-scopeline`.
6. After a filter change, a selection of 5 stays 5. The not-loaded disclosure lists those outside the result and does not flicker while loading.

## 16. Decisions for root

1. **R-1 scope value:** does the users read return a workspace name? If not, show the ID. Do not invent a name.
2. **R-2 scope after tabs:** accept it as a recorded deviation from `spec.md:114`, as for O01.
3. **R-3 band memory below 1280:** collapsed on each first tab load (O01 D8).
4. **R-4 jobs note:** show it only when filters are present, or always (the plan reads as always). I recommend only when present.
5. **R-5 departed sub-line:** show it in every tab, or only in Membership. In the Users tab it duplicates the "Departed" status, but it explains why an ID shows instead of a login.
6. **R-6 open person filtered out:** close the panel and announce (recommended), or keep the panel open with a "not in the loaded rows" line and no data.
7. **Apply model:** search on submit, everything else on change (O01 D1). Per-change read cost is unmeasured at 100,000 people.
8. **Page size:** 50 per read (the plan says 50; the code calls 100).
9. **New [TW] strings:**
   - "Filters";
   - "Search";
   - "All" per select;
   - "Sign-in method";
   - "Applied filters:";
   - the chip removal name;
   - the membership chip suffix;
   - "Filters · {n} applied";
   - the order line;
   - "No matches yet.";
   - the next-page failure;
   - "Loading users…";
   - the live-region templates;
   - the open-person-filtered sentence;
   - the membership-status word for `ended`;
   - the "people" or invitation noun in the count line;
   - checking the jobs-note tab names.
10. **Similar labels:** "Role period ended, not yet suspended" (membership) sits beside "Role period ended" (period). TW should confirm users can tell them apart.
11. **Implementation item IDs (proposed; reconcile with the existing M01 IDs without renumbering):**
    - `M01:users:scope-line`;
    - `M01:users:filter-band`;
    - `M01:users:filter-search`;
    - `M01:users:filter-status`, `-membership`, `-role`, `-period`, `-signin`, `-holder`;
    - `M01:users:method-line`;
    - `M01:users:filter-chips`;
    - `M01:users:count-line`;
    - `M01:users:order-line`;
    - `M01:users:show-more`;
    - `M01:users:scan-stopped`;
    - `M01:users:list-changed`;
    - `M01:users:restart`;
    - `M01:users:departed-row`;
    - `M01:invitations:fixed-chip`;
    - `M01:jobs:filter-note`.

## 17. External references

Reuse the three references root opened for HYB-300 (`HYB-300/references.md`): MOJ "Filter a list", Cloudscape filter patterns, and Cloudscape table view. Their adopted and rejected points apply unchanged:
- chips with "Remove", and Clear all;
- server filtering over the full permitted set;
- progressive loading instead of numbered pages;
- no Apply button;
- no left-column panel.

**Optional, not opened by me:** a user-directory filter reference that root may open and record, such as the Microsoft Entra admin center "All users" list filters (Microsoft Learn). Nothing is adopted from it until root records it.

## 18. Must not appear

- coloured edge strips, coloured chip, row, band or tab borders, or coloured selection outlines;
- a disabled or greyed authentication-method select, a tooltip-only explanation, or a "coming soon" badge;
- a workspace selector;
- totals ("of N", "all N", percentages, progress bars), or numbered pagination;
- sortable headers or sort arrows;
- per-keystroke search, or an Apply button;
- stretched inputs or buttons at ≥768px, or a centred max-width container;
- dimming overlays, skeleton numbers, shadows, rounded corners, Unicode ×, arrows or emoji;
- "No users yet." while a filter is set;
- any claim that a selected person "does not match";
- a departed person's login or email;
- any HYB-323 control or placeholder;
- duplicate rows after paging.

## 19. Critique against the owner's brief

**Meets:**
- full-viewport width with content-width controls;
- neutral hierarchy;
- every spec filter: workspace as fixed scope plus a membership filter, status, method stated as text, role, and period, plus sign-in setup and administrators;
- truthful counts and resumable scan stops;
- the restart and list-changed states;
- selection that never widens and is disclosed honestly;
- room for Part B with no fake control;
- header/cell parity preserved.

**Risks:**
- The band is about 330px tall at 1440 and about 400px at 1280 before the list starts. The sign-in method line wraps to two or three lines. Showing the h2 only below 1280 would save about 36px, at the cost of consistency with O01.
- At 1280 the administrators checkbox sits alone on a third row.
- All widths are calculated, and no current render was viewed.
- AR select truncation and the grid-span risk (§0 item 5) need rendered verification.
- The KO and AR copy are AI drafts that need TW and native review. Screen acceptance (FIX) belongs to the owner.

Files referenced:
- /private/tmp/hb294/docs/features/HYB-294/plan.md
- /Users/home/project/hybrid_billing/docs/ui/screens/M01/spec.md
- /private/tmp/hb294/docs/features/HYB-300/visual-spec.md
- /private/tmp/hb294/docs/features/HYB-300/references.md
- /private/tmp/hb294/docs/features/HYB-299/visual-spec.md
- /private/tmp/hb294/apps/console/src/modules/identity/pages/users/Live.tsx
- /private/tmp/hb294/apps/console/src/modules/identity/pages/users/Selection.tsx
- /private/tmp/hb294/apps/console/src/modules/identity/pages/users/bulk.css
- /private/tmp/hb294/apps/console/src/modules/identity/pages/users/bulk-copy.ts
- /private/tmp/hb294/apps/console/src/modules/identity/pages/users/live.css
- /private/tmp/hb294/apps/console/src/app/screen-groups/identity.tsx
- /Users/home/project/hybrid_billing/.agents/rules/service-screen-style.md
- /Users/home/project/hybrid_billing/.agents/rules/usability.md
- /Users/home/project/hybrid_billing/.agents/rules/design-system.md

## 20. Root decisions (2026-11-18)

Binding over the matching proposals.

1. **R-1:** show the workspace name only if the users read returns one; otherwise the ID. No invented name.
2. **R-2:** accepted as a recorded deviation from `spec.md:114`, as for O01 (HYB-300 D2).
3. **R-3:** collapsed on each first tab load below 1280.
4. **R-4:** the Revocation jobs note shows only when at least one filter is in the link.
5. **R-5:** the departed sub-line shows in every tab.
6. **R-6:** close the panel, keep focus on the filter control and announce.
7. **Apply model:** search on submit, everything else on change (O01 D1).
8. **Page size:** 50 per read, as the plan says.
9. **[TW] strings:** drafted by the implementer in EN/KO/AR, then fixed by the technical-writing review.
10. **Similar labels:** for the technical-writing review.
11. **Item IDs:** adopted as proposed; reconcile with existing M01 IDs without renumbering.
12. **References:** the HYB-300 references apply (`../HYB-300/references.md`); no new reference is opened for Part A.
13. **Band wrapping (round-2 render review):** the §4.2 wrapping text and the §14 band cells for 1280 and 1024 are replaced by the ui-designer's amendment: fixed widths (Membership 336px, Role period 242px), language-dependent row split, acceptance criteria instead of a calculated split.
