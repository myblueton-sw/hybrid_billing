---
type: Visual Spec
title: HYB-300 Part A visual spec for the O01 filter panel, search and server paging
description: Layout, type, spacing, states, keyboard order and responsive behaviour for the O01 server-side filters, search, active-filter chips, count certainty, Load more and the scan-stopped state, at 1440, 1280 and 390 in EN, KO and AR, with root decisions; room is left for the Part C view group actions.
status: stable
tags: [HYB-300, O01, visual-spec, filters, paging, ui-designer]
sources:
  - resource: plan.md
  - resource: references.md
  - resource: ../HB-273/plan.md
---

# HYB-300 Part A: O01 filters, search and server paging (visual spec)

Author: the ui-designer role (Claude, read-only), 2026-11-18. Root decisions are in §15 and override the proposals they answer. This is a design proposal, not a review PASS or owner acceptance. O01 spec references point to the local planning input `docs/ui/screens/O01/spec.md`.

## 0. Evidence

Current renders from the latest regression set (`O01_accounts_en_1440`, `O01_groups_en_1440`, `O01_accounts_ar_390`, `O01_groups_ko_390`) show:
- The scope row (date, Show, Today, revision and workspace facts, Reload) below the App tabs, closed by a neutral hairline. It stays.
- At 1440 the content area is about 1120px: list about 707px, panel about 380px (`clamp(360px, 34%, 440px)`). Date-bearing account headers wrap to two lines; a fifth column would make this worse.
- The only filter today is a lone "Only unattributed" checkbox; no grouping and no applied-filter summary.
- The count reads "9 members loaded", which describes the load, not completeness.
- At 390 each table cell stacks label/value; one account is about 290px tall.
- No coloured strips or coloured selection outlines.

## 1. Composition

App tabs → scope row → filter band (new) → list/panel grid. The band spans the full content width above list and panel; fields keep intrinsic widths and wrap, never stretch. Hierarchy comes only from neutral hairlines, spacing and weight: no box, no tint, no coloured edges. Inside the list column (about 576px at 1280) four controls would take three rows; across the band they take one or two.

## 2. Filter band at ≥1280px

**Container:** `section.o01-filters` with `aria-labelledby="o01-filters-title"`; `h2#o01-filters-title` "Filters" 16px/600, `margin-block: 0 12px`; `padding-block-end: 16px`, `margin-block-end: 24px`, `border-block-end: 1px solid var(--border-row)`; no top border.

**Field row** `.o01-filter-fields`: `display:flex; flex-wrap:wrap; gap:16px 24px; align-items:flex-end`; labels above controls (`.service-field`); all controls `min-block-size:44px`.

| # | Control | Element | Inline size | Apply |
| --- | --- | --- | --- | --- |
| 1 | Search name or ID | `form[role=search]`: `input type=search dir=auto maxlength=64 autocomplete=off` + secondary button "Search" | input 320px | Enter or Search, never per keystroke |
| 2 | Legal entity | `select`: "All legal entities", then names; suspended reads "{name} · Suspended" | 280px | on change |
| 3 | Relationship kind | `select`, five kinds, default Organization hierarchy | 240px | on change |
| 4 | Only unattributed | checkbox, 44px row | auto | on change |

At 1440 EN the row is about 1,214px, so the checkbox wraps to a second row; do not shrink fields to force one row.

**Provider line** `p.o01-provider`: below the fields, `margin-block-start:12px`, `max-inline-size:68ch`; `<span class="o01-provider-label">Provider</span>` (600) then muted 16px "Not available yet: source accounts are not linked to provider connections." Not focusable; no disabled control, icon or tooltip.

**Applied filters** `div.o01-applied`: `margin-block-start:16px`, rendered only with at least one non-default filter. Lead "Applied filters:" (14px muted), `ul` of chips (`flex-wrap; gap:8px`), then "Clear all filters" as `.service-button-link` on the same wrapping line.

**Chip:** `button type=button`, text "{Field}: {value}" plus `packages/design-system/assets/icons/close.svg` at 16px (never a Unicode ×); accessible name "Remove filter {Field}: {value}"; `min-block-size:44px` (§15 D4), `padding-inline:12px 8px`, `1px solid var(--border-control)`, `background: var(--surface-subtle)`, radius 0, 14px text; ID values in `<bdi dir="ltr">` mono; hover and active change the neutral background only; focus uses the focus token.

**Chips exist only for non-default values** (entity, non-hierarchy kind, unattributed, search). The as-of date is scope: never a chip, never reset by Clear all. Groups view prefixes: "Members: {entity}", "Members: only unattributed", "Groups: {q}".

## 3. Filter band below 1280px

- `button.o01-filters-toggle` replaces the heading: `aria-expanded`, `aria-controls="o01-filter-body"`, "Filters" or "Filters · {n} applied", 600, 44px, leading design-system chevron that mirrors in RTL and rotates when open.
- Collapsed on each first view load; open after the user opens it for that view (§15 D8).
- Applied chips and Clear all render below the toggle, outside the collapsible body.
- 768–1279px: desktop flex-wrap, widths capped by `min(100%, …)`. Below 768px: `flex-basis:100%`, `inline-size:min(100%, 440px)`; the Search button on its own line, auto width.
- After a change on mobile the body stays open, focus stays on the control, and the live region announces the count (§7).

## 4. List header and count certainty

`div.o01-list-head { display:flex; flex-wrap:wrap; align-items:baseline; justify-content:space-between; gap:8px 24px; margin-block-end:16px }`.

Start slot: the title h2 and a 14px muted meta `p`:

| Situation | Text |
| --- | --- |
| `next` present or `scan_stopped` with rows | "{n} shown; more may exist · Relationships on {date}" |
| list ended | "All {n} shown · Relationships on {date}" |
| groups | "{n} groups shown; …" in the same pattern |

Never "of N", never a total. The date moves into the meta line; account headers become "Organization", "Payer", "{kind}", "Entry from" (§15 D3).

End slot is reserved for Part C and renders nothing in Part A.

Accounts columns: colgroup 30/26/26/18 without a kind column; 26/20/20/20/14 with it. At 390 the stacked `data-label` keeps the full dated wording.

## 5. Load more and scan stopped

`div.o01-more { display:flex; flex-wrap:wrap; align-items:center; gap:8px 16px; margin-block-start:16px }`.
- `next` present: secondary "Load more" at inline-start, auto width, 44px, never full width. While loading: same size, "Loading…", `aria-disabled="true"`, rows kept. List ended: no button.
- `scan_stopped`: `role="status"` text "Searched through {id}." (ID in `bdi` mono) and a separate secondary button "Continue searching"; text `max-inline-size:68ch`. With zero rows it replaces the table: "No matches yet. Searched through {id}." plus Continue searching. The empty state is never claimed.
- Next page failed: rows kept; `role="alert"` "The next page could not be loaded. The rows shown are kept." plus "Try again" (the spec's partial state).
- After Load more or Continue, focus moves to the first appended row's `.live-open`; if nothing was appended, focus stays on the button. Live region: "{k} more shown. {n} shown in total; more may exist."
- At 390 a compact copy of the count text repeats next to Load more.

## 6. Per view

- **Tree:** the band shows only the muted note "Filters apply to the accounts, groups and history views. The tree shows the loaded organization structure." No fields and no toggle.
- **Accounts:** all four controls plus the provider line.
- **Groups:** the View group / Allocation pool radio fieldset moves into the band as the first control (§15 D7); the kind select is hidden; search labelled "Search groups by name or ID" filters the group list; legal entity and Only unattributed sit in a nested borderless fieldset with legend "Filter members" (14px/600); the note "A view group grants neither ownership nor access." stays above the table; the group count header reads "Members on {date}, all".
- **History:** the kind select and a member search (`subject=any`, `q`); results as a `ul` of up to 10 buttons "{name} {id}" with Load more; choosing one collapses to "Member: {name} {id}" with link-button "Change", focus to `#o01-history-title`. Entity, unattributed and the provider line are omitted (§15 D5).

## 7. States

| State | Rendering |
| --- | --- |
| Initial loading | existing `role=status`; band and chips render from URL values at once |
| Filter or search changed | clear the table, "Loading {view} for {date}…" (status); no dimming or overlay; last request wins |
| Filtered empty | `.live-empty` "No items match these filters in your permitted scope on {date}." then "Clear all filters" |
| Unfiltered empty | existing per-view sentences |
| Scan stopped, zero rows | §5 |
| Next page failed | §5 |
| First-page error | existing `.live-error`; filters and date kept |
| Bad URL parameter | field reset to default with `role=status` "A filter in the link was not valid and was removed." |
| Denied or revoked | existing; band not rendered |
| Revision newer | existing `.o01-newer` |

Every state is text; colour never carries a state alone.

## 8. Selection

- Selected row keeps `.live-selected` (neutral background) and `aria-current="true"`; no coloured outline.
- Selected but not loaded: panel's first line `p[role=status]` "Selected: {id}. Not on the loaded pages of this view."; the selection is not cleared.
- A filter or date change keeps the single selected member by ID; only Part C's page multi-selection clears.
- Panel stays `clamp(360px, 34%, 440px)` at ≥1280; below 1280 the existing step-in detail and Back flow.

## 9. Keyboard and reading order

DOM order = visual order: App tabs, scope row (date, Show, Today, Reload; accepted deviation, §15 D2); the toggle below 1280; search and Search; legal entity, kind (groups: radios, search, member fieldset); Only unattributed; provider line (not focusable), chips, Clear all; list heading (`tabIndex=-1`), rows, Load more or Continue; panel.
- Removing a chip moves focus to the next chip, else previous, else the search input (desktop) or toggle (mobile). Clear all moves focus to the search input or toggle. Both announce "Filter removed. {count text}."
- The live region announces after each read: "{count text}, {n} filters applied."
- All interactive elements show the focus token, 2px outline, 2px offset.

## 10. Locales and RTL

- KO keeps `word-break: keep-all`; AR uses its Plex family at line-height 1.8, so chips use `min-block-size` only.
- RTL: band and chips flow from the right, close icon at inline-end, chevron mirrored, Load more at inline-start; logical properties only.
- IDs, revisions and the scan-stop ID in `<bdi dir="ltr">`; dates via `<time>` and the existing `day()`; search `dir="auto"`.
- Field widths in px; labels wrap within the field.

## 11. Breakpoints

| Width | Content | Band | List/panel |
| --- | --- | --- | --- |
| 1440 | ~1120px | inline, two rows in EN and AR, chips below | ~707 / ~380 |
| 1280 | ~960px | inline, two rows | ~576 / 360; "Entry from" must not wrap mid-date |
| 768–1279 | — | collapsible, fixed widths wrap | single column, step-in detail |
| 390 | 358px | collapsible, fields full width up to 440, buttons auto | stacked cells; count repeated by Load more |

No width may produce horizontal page scroll.

## 12. References

Opened and recorded by root in `references.md`: MOJ "Filter a list", Cloudscape filter patterns, Cloudscape table view.

## 13. Must not appear

Coloured edge strips; coloured outlines on rows, chips or the panel; coloured tab or band borders; a disabled or greyed Provider select, tooltip-only explanation or "coming soon" badge; colour-only states; totals ("of N", percentages, progress bars) when unknown; stretched inputs or buttons at ≥768px; a centred max-width container; dimming overlays; shadows, rounded corners, Unicode symbol icons or emoji; any Part C control in Part A.

## 14. Designer's critique

Meets all five spec filters (four live, provider stated in text), explicit-submit search, truthful counts and resumable scan stops, chips with Clear all, full-width layout without stretched controls, neutral hierarchy, the mobile collapsible panel, selection by canonical ID and a reserved Part C slot. Risks: two band rows at 1440; the groups view's split between group search and member filters needs render verification; per-change reads are unmeasured; 390 account cards remain tall (a compact mobile row would change HB-273's verified table and is not proposed); the tab-before-scope order predates Part A.

## 15. Root decisions (2026-11-18)

1. **D1 apply model:** selects and the checkbox apply on change, announced in the live region; search applies on submit. MOJ's Apply button is rejected (references.md §1); revisit only if measured reads prove costly.
2. **D2 order:** the existing App tabs → scope order is accepted as a recorded deviation from O01 spec line 112; no change in Part A.
3. **D3 dates out of headers:** adopted; the dated `data-label` stays on mobile. Copy changes go to the technical-writing review.
4. **D4 chip hit size:** 44px everywhere, consistent with the other O01 controls.
5. **D5 history view:** entity, unattributed and the provider line are omitted there.
6. **D6 new strings:** listed in §2–§7; EN as written, KO and AR drafts for the technical-writing review.
7. **D7 groups band:** the View group / Allocation pool radios move into the band.
8. **D8 panel memory below 1280:** collapsed on each first view load.
9. **D9 close icon:** `packages/design-system/assets/icons/close.svg` exists and is used.
10. **D10 references:** three opened and recorded in `references.md`.
11. **D11 provider line in the groups view (from the technical-writing re-check):** kept. The sentence is true there too, and the member filters would include a provider filter once source accounts are linked to provider connections.
12. **Tables (render rounds 1–3):** the §4 colgroup percentages are dropped in favour of automatic layout; dates and IDs never wrap; cells wrap only between words at 768px and above.

