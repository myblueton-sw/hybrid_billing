---
type: DesignSystemReference
title: The console's data list pattern
description: One shared anatomy for the console's server-paged lists — filters, applied-filter chips, summary line, table, pager with rows per page, states and the list settings disclosure — with the references adopted and rejected.
status: draft
---

# Scope and ownership

Composed on 2026-11-27 by the ui-designer role (a role-based AI designer, by reading the source and the [references](list-pattern-references.md); no renders were supplied) after the owner asked that the service screens reach enterprise-level design quality. Root recorded it and implements it. It serves [HYB-376](https://linear.app/myblueton/issue/HYB-376) (page size choice and list settings), [HYB-378](https://linear.app/myblueton/issue/HYB-378) (applied-filter chips) and [HYB-379](https://linear.app/myblueton/issue/HYB-379) (list state in the address). It is not owner acceptance; each built increment gets a rendered designer review.

# Facts the pattern respects

Lists are server-paged by an opaque cursor (First, Previous, Next; no page numbers, no random access). A count is exact or a lower bound. Every list shows a reference time. A refused or expired cursor shows "query changed" with First. The page size is bounded by the installation's limit (default 50, 1 to 100), known from the entry read; while unknown no size is sent. Filters apply with an explicit Apply because the reads are costly; search is submit-based. Lists have master–detail panels at 1280 px and wider and a step view below. English, Korean and Arabic are first-class; identifiers and numbers stay left-to-right mono.

# Anatomy, top to bottom

Spacing uses the existing scale (2, 4, 8, 12, 16, 24, 40 px); separate lists or sections by 40 px.

1. **Header** (required). The existing `live-toolbar`: title and page-level actions at the inline end. M02 keeps its count headings; a lower-bound count reads "Open (at least n)" (a new plural form, terminology review). Headings take `tabIndex={-1}` as a focus fallback.
2. **Filter panel** (required). An explicit form with Apply and Reset (Carbon Filtering). New `.list-filters` and `.list-filter-grid` replace `.access-filter-grid` and the M02 grid: `repeat(auto-fill, minmax(14rem, 1fr))`, gap 16 px, each field at most 20rem wide. "Apply filters" and "Reset filters" follow after 12 px, `live-secondary`, natural width. Below a 600 px container the panel collapses behind "Filters (n applied)".
3. **Applied filters** (required when a filter is applied). 16 px below the panel; see Chips.
4. **Summary line** (required). New `.list-summary`, 12 px below: the count ("Matching count: 240" or "…at least: 240") and the reference time, `flex-wrap`, gap 4 px 24 px, 14 px muted, no separator glyph. `tabIndex={-1}` and a visually hidden polite status copy of the count. On M02 the reference time moves from below the table to here: it qualifies the count and rows, and the count is always shown (Carbon Search).
5. **Table** (required). 8 px below. Rows keep `min-block-size: 56px` (cells carry second lines); cells 12 px block and 16 px inline padding, at least 16 px between columns (Carbon Style), 8 px inline below a 960 px container. Header 14/600, body 16, sub-lines 14. Identifiers start-aligned mono in `<bdi dir="ltr">`; numbers end-aligned with tabular figures (USWDS, GOV.UK); dates and codes start-aligned; status as text with weight as the non-colour cue (Carbon Status). Hover always on with `var(--c-subtle)`. A selected row has `var(--surface-hover)`, `aria-current="true"` and the visible "Selected" tag (`.list-tag`, from `m02-tag`) — never the background alone (Atlassian). No sticky header. Optional: a neutral header background.
6. **States** (required), each replacing the table and its header (Carbon Empty states): no data (the list's sentence); no match ("No matches for these filters." and a "Clear filters" button); nothing on this page but more to scan (the existing "incomplete" copy with Next); loading (a text status line; optional static placeholder rows, no animation); failure (the existing box; a refused size keeps its wording).
7. **Pager** (required). Directly below the table; see Pager.
8. **List settings** (required on M02 and M03). Once per screen after the last list, 24 px above; see List settings.

# Chips

- "Label: value" with the filter field's label; the value in weight 600; identifiers and dates mono in `<bdi dir="ltr">`. Long values wrap and are never truncated. The page size is not a chip.
- `.list-chip`: an inline-flex `<li>`, 1 px `var(--border-control)`, square, `var(--surface-page)`, at least 44 px tall at every width (matching the console's 44 px buttons; revised at the first rendered review), 14 px text, `padding-inline: 12px 0`.
- `.list-chip-remove`: a 44 × 44 px button with an `aria-hidden` "×" at the inline end and a 1 px neutral `border-inline-start`; accessible name "{Remove filter} {Label}: {value}".
- The row is `<section class="list-applied">` with a visible 14 px lead, a wrapping `<ul>` (gap 8 px) and "Clear filters" as the last item, `live-secondary`.
- Removing a chip applies at once: a new query, first page, cursor and trail cleared, that draft field cleared. Focus goes to the next chip's remove button, otherwise the previous one; after the last chip or "Clear filters", to the summary line once loaded, else the list heading.
- At 390 px chips wrap full width. Logical properties only; the remove control is at the inline end.

| Text | English | Korean | Arabic |
| --- | --- | --- | --- |
| Lead | Applied filters | 적용된 필터 | عوامل التصفية المطبّقة |
| Remove control | Remove filter | 필터 제거 | إزالة عامل التصفية |
| Clear all | Clear filters | 필터 모두 해제 | مسح جميع عوامل التصفية |
| No match | No matches for these filters. | (to be written with the terminology review) | (to be written with the terminology review) |

# Pager

- `.list-pager`: flex, wrap, `justify-content: space-between`, gap 8 px 16 px, at least 56 px tall, 1 px `var(--border-row)` at the block end.
- Start group `.list-pager-size`: "Rows per page" (existing copy, 14 px regular) with a native select 44 px tall, then "Rows on this page: 25" / "이 페이지의 항목: 25" / "الصفوف في هذه الصفحة: 25" (label and value, so no Arabic plural agreement).
- End group: `<nav class="list-pager-nav" aria-label="{list title}, pages">` with First, Previous, Next (`live-secondary`, gap 8 px), disabled states as today, no arrow glyphs.
- **No range text** such as "1–25 of 240": a cursor arriving from the address has no known offset, and a page that stops scanning can hold fewer rows than its size, so the range would often be false. The count stays in the summary line (adapted from PatternFly's indeterminate pagination).
- The nav is hidden on a first page without a next cursor (GOV.UK); the bar is hidden when nav and select are both hidden.
- Options: 10, 25, 50, 100 up to the limit, plus the limit itself, sorted; the current value is the echoed `query.limit`, added if missing. Hidden with one option or an unknown limit.
- Lists with the select: M02 open and closed, M03 permissions and ended requests. Not M03 waiting requests (a short to-do list sized to show everything), revocations (10) or the person picker (25).
- Changing the size applies at once, restarts at the first page, clears the trail, keeps the filters and writes the size to the address (HYB-379). Focus stays on the select (WCAG 3.2.2); the count is announced; the newest request wins.
- After paging, focus goes to the summary line (no reference covers this; the changed content starts there and it says what changed — WCAG 2.4.3). This replaces M02's focus on the reference line.
- At 390 px the start group takes the first line and the nav the second, its three buttons sharing the width. This needs an inline-size container around the pager; a screen that adopts the pager provides one (`container-type: inline-size`).
- When the size was lowered to the limit, a line under the select: "The installation limits rows per page to {n}." / "설치 설정에 따라 페이지당 항목 수가 {n}개로 제한됩니다." / "يحدّ إعداد التثبيت عدد الصفوف في الصفحة بـ {n}."

# List settings

The existing settings table (Setting, Value, Range, Source, identifier in mono under each name), moved to a shared `.list-settings` `<details>`.

| Text | English | Korean | Arabic |
| --- | --- | --- | --- |
| Title | List settings | 목록 설정 | إعدادات القوائم |
| Lead | Set by the installation; they apply to every list on this screen. | 설치 시 정해지며 이 화면의 모든 목록에 적용됩니다. | يحدّدها التثبيت وتنطبق على جميع القوائم في هذه الشاشة. |
| `service_query_page_size` | Most rows per page | 페이지당 최대 항목 수 | الحد الأقصى للصفوف في الصفحة |
| `service_query_scan` | Records checked per read, at most | 조회당 최대 확인 항목 수 | الحد الأقصى للسجلات المفحوصة في كل قراءة |
| `service_query_lifetime_seconds` | Page position stays valid for | 페이지 위치 유효 시간 | مدة صلاحية موضع الصفحة |

`service_query_scan` is a budget of work units, one unit per record a read examines (`scanBudget` in the service), so "records checked" matches its meaning. The terminology review confirms the names.

# Per list

| List | Changes | Owner decision |
| --- | --- | --- |
| M02 open and closed | Shared filter grid; chips above both sections; summary line above each table (the reference time moves up); pager with select; focus after paging on the summary line; list settings after the closed section; lower-bound heading form | — |
| M03 permissions | Summary line replaces `.access-count`; pager with select replaces `.access-paging`; chips; "Selected" tag with `aria-current`; nav hidden on a single page | — |
| M03 requests | Waiting: summary and pager without select. Ended: with select | — |
| M03 revocations, person picker | Summary and pager without select; the picker keeps its list idiom | — |
| M03 screen | "List settings" beside "Approval settings" | — |
| M04 | Chips; summary line and pager replace `PageInfo` (fixing its nav label and the never-disabled First and Previous) | Moving its numeric size field into the pager select (maximum 100 there) changes its filter form |
| M05 sessions and operations | Chips replace `mysec-applied`, without a `limit` chip; summary line replaces `mysec-page-meta`; pager | Replacing its numeric size field with the select; Apply from primary to secondary weight |

Increments: shared components and CSS (`ListSummary`, `ListPager`, `AppliedFilters`, `ListSettings`, `list-pattern.css`); M03; M02; M05; M04. The new classes replace `.access-count`, `.access-paging`, `.mysec-applied`, `.mysec-page-meta`, the M02 filter grid and `.access-filter-grid`.

# References adopted and rejected

| Reference | Adopted | Adapted | Rejected |
| --- | --- | --- | --- |
| Carbon data table usage and style | toolbar above; hover always on; pager below with no gap; 16 px column gap | skeleton loading as optional static rows (no decorative motion) | 40 px rows (56 px multi-line rows kept); 14 px body (project 16 px kept) |
| Carbon accessibility, empty states, filtering, search | accessible table name; empty state replaces the table with one next step; batch Apply; applied count when the panel is closed; clear-all; always show the count; submit-based search; instant apply for the size select | — | sort (not applicable) |
| Carbon pagination | items-per-page select; bar below the table | — | "page x of y" and range text (cursor paging) |
| Carbon status indicators | text with a non-colour cue (weight) | — | — |
| PatternFly table, filters, pagination, empty state, primary-detail | content alignment and tabular figures; removable labels; "Clear filters"; mobile summary on top; selected state and back link | one chip per filter (filters are single-valued); indeterminate total as "at least"; "No results found" as "No matches" | pagination as the last toolbar item |
| USWDS table and pagination | number and date alignment; `nav` with a unique label | — | sticky header (scroll wrapper, short pages); hiding Previous on the first page (disabled controls keep positions) |
| GOV.UK table and pagination | caption and `scope`; hiding the pager on a single page | — | page number in the document title |
| Atlassian | selection never by highlight alone; row-specific action labels | — | coloured lozenges |
| Ant Design | separate no-data and no-results states | sizes 10, 25, 50, 100 (keeping 25) | "1–20 of 85" |
| Shopify | previous and next only; "Remove … filter"; reset the cursor when the query changes; the address owns the query | — | "Clear all" |

# Progress

- **L1** (2026-11-27, HYB-376 and HYB-378): the shared parts are built in `apps/console/src/shared/list/` (`ListSummary`, `ListPager`, `AppliedFilters`, `ListSettings`, `list-copy.ts`, `list-model.ts`, `list-pattern.css`) and applied to the M03 permissions list, with rows per page in the address (`acc_limit`) and "List settings" after the revocations section. Revised at the rendered review: chips are 44 px at every width; the pager label is 14 px regular. Verified: 1506 console tests, 1501 passed, 5 skipped; `tools/list-pattern-qa.mjs` 95 checks passed against the real service in three languages at 1440, 1024 and 390, measuring the summary spacing, the gap to the table and the chip height; the existing M03 permissions, catalog and revoke-panel checks pass. ui-designer rendered review: pass with four conditions (paragraph margins, the gap to the table, chip height, a shaded row that was pointer hover), resolved. Independent change review: approve with two blocking findings (Reset could not clear an invalid size in the address; focus was lost after the no-match clear) and two conditions (the shared parts imported page copy; an inert linter directive), resolved. Known: the hidden count announcement sits inside the focusable summary line, so a screen reader may read the count twice there; a focus flag can stay set when a size choice sends the same query. Next: the filter panel (fields at uneven heights, the lead on the chip row) and the narrow list column at 1440 against the full-viewport rule; then the M03 requests, revocations and picker, M02, M05 and M04.

- **L2** (2026-11-27): two ui-designer decisions on the M03 grants view. (1) The access request form opens on demand at every width: "Request access" in the toolbar at every width with `aria-expanded`, a Close control at 1280 and wider laid on the panel heading's row, focus to the panel heading on open and back to the invoker on close; at rest the list takes the full width (about 1120 px at 1440 instead of about 708 px). This changes behaviour established by an earlier plan (the form was always visible at 1280 and wider); no owner instruction was found behind that, so it was decided as a composition matter and is listed for owner screen acceptance. (2) The filter panel: a "Filters" heading (visually hidden in a container of 600 px or less, where the toggle labels it), label then input then hint, `.list-filter-grid` with `repeat(auto-fill, minmax(14rem, 20rem))` and gaps of 16 and 24 px, inputs 44 px, the buttons 16 px below, a neutral hairline under the panel; the "Choosing a person by name is not available yet" note joins the holder hint; the "Applied filters" lead shares the chip row at 768 px and wider. At the full width the permissions table showed words breaking in narrow columns; its eight columns now total 100 percent (holder 13, access 16, scope 13, source 8, original delegation 11, validity 13, effective 13, actions 13). Verified: list-pattern 95, M03 permissions 116, revoke panel 54 (full width measured, `aria-expanded`, focus on open, on a second click and after Close, Close never touching the heading text at 1440 and 1280 in English, Korean and Arabic), person picker 66, catalog 32, requests 90, revocations 57; console tests pass. ui-designer rendered review: pass, its condition (captures of the open panel) met. Independent change review: approve with conditions (the heading selector also matched error-box headings; the column widths totalled 95 percent; weak full-width check), taken. Known: with a panel open at 1280 the list is in its card layout, whose cells lack labels (the next increment); opening the form while a revoke or decide panel is open replaces it and drops a typed reason; after a failed reload the form can remount and take focus; `tools/candidate-access-qa.mjs` at the repository root still expects the always-visible form and a select person field (stale before this change for the field).

- **L3** (2026-11-27): the M03 permission cards (a container of 600 px or less: with a panel open at 1280, and at 390) label every value with its column name on its own line above the value (14 px, muted), keep the holder as the unlabelled card title at every width, and space their lines (cell padding 4 px, sub-lines 20 px, rows 16 px). The overlapping lines had a root cause outside the cards: the console's global `td { height: var(--ui-row-h) }` fixed each card cell at 44 px; the cards set `block-size: auto` (the other identity card layouts already override it). Verified: the revoke-panel check (59) asserts card mode at 1280 with the panel open in three languages, a label on every labelled cell, none on the holder and no overlapping text lines; the permissions check (116) passes. ui-designer rendered review: pass; independent change review: two corrections taken.

- **L4** (2026-11-27): the shared paging parts (`access/paged.tsx`) render `ListPager` and `ListSummary`, so the M03 waiting and ended requests, the M03 revocations and both M02 sections take the pattern: summary line above the table, pager below with "Rows on this page", the nav hidden on a single page, focus after paging on the summary line. Rows per page on ended requests (25) and M02 open (50) and closed (25), kept in component state until HYB-379; focus returns to the select after the list reloads. Revised at the rendered review: where a heading already states the count ("Open (n)", "n waiting", ended requests, revocations) the summary line shows only the reference time and keeps the hidden count announcement (`countInHeading`); the M02 evidence note moved above the table so the pager follows the table directly. Found on the way: `tools/page-limit-qa.mjs` had broken when the request form became on demand (L2) because it was not rerun then; from now on every increment reruns all M02 and M03 checks. Verified: every M02 and M03 rendered check passes (m02-lists 174 with a size change on the closed list, m02-detail 100, authentication 56, m03-permissions 116, list-pattern 95, m03-requests 90, m03-revocations 57, revoke panel 59, catalog 32, person picker 66) and page-limit 29 at a limit of 10. Reviews: ui-designer pass with two conditions, independent change review approve with two conditions (a module-level focus flag replaced by a per-pager ref; the revocations section made an inline-size container), all taken. Known: M02's table still differs from M03's (no header layer, tighter padding, no hover) — the M02 increment; a section's size resets when a one-state filter remounts it until HYB-379.

# Not decided by reading

Today's rendered spacing; whether the project's 44 px minimum applies to desktop chip controls (40 px meets WCAG 2.5.8); whether `--c-subtle` and `--surface-hover` differ enough to tell hover from selection; whether paging pushes or replaces history entries (HYB-379); native-language review of the Korean and Arabic copy; whether "Reset filters" and "Clear filters" should share one label.
