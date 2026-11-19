---
type: DesignSpecification
title: HYB-299 M01 role periods and bulk revocation visual specification
description: Layout source for the M01 service-mode changes in HYB-299. Covers selection in the users table, the bulk revocation steps, the Revocation jobs tab, role period display and editing, refusals and clock conditions. Written by the ui-designer role (read-only). The HYB-299 plan wins on copy, states and truthfulness; this spec wins on layout.
status: stable
tags: [HYB-299, M01, bulk-revocation, role-period, visual-spec]
sources:
  - resource: plan.md
  - resource: ../../ui/screens/M01/spec.md
  - resource: ../../ui/screens/M01/design.json
  - resource: ../HYB-315/visual-spec.md
  - resource: ../HYB-297/visual-spec.md
---

# HYB-299 visual specification: M01 role periods and bulk revocation (service mode)

**Author:** ui-designer role, read-only. The environment reports the model as Claude Opus 5.5 (`claude-opus-5-5`).

**Status:** this is a design proposal. It is not owner acceptance, and a later visual PASS would not be owner acceptance either.

**Root decisions on §12 (2026-11-18):**
- R-1: the panel opens on demand.
- R-2: the caller's own row has no checkbox.
- R-3: selection is kept in memory only.
- R-4: there is no Back link while sending.
- R-5: on a staging failure the draft is kept, and resume or discard is offered. Resume is idempotent.
- R-6: executed targets are removed from the selection.
- Items 2, 4, 5, 6 and 9 follow the recommendations:
  - discard and stage again after deselecting, with copy;
  - "not in the loaded rows";
  - less than a day reads "Ends today at {time}", with KO and AR plurals;
  - "open" instead of "(selected)";
  - cleanup progress is counted in items.
- Item 3: the plan adds a paged read of the targets of a prepared manifest (plan amendment).
- Item 12: `offers()` is fixed in the model.

**Precedence:** where this spec and `docs/features/HYB-299/plan.md` (approved after design review round 5) differ on copy, states or truthfulness, the plan wins. This spec wins on layout. **[TW]** marks a string the layout needs that the plan does not define. The technical-writing review must supply or approve those strings; I am not supplying final copy. **[R-n]** marks a decision for root (§12).

---

## 0. Evidence basis and limits

**Read directly:**
- `docs/features/HYB-299/plan.md`: lines 1–921 and 1125–1152 (scope, design, Durability, API contract, settings, Console, Copy, browser QA). I did not read the threats table, the residual list, owner questions or the dependency sections in full.
- `docs/ui/screens/M01/spec.md` and `design.json`, both in full.
- Current M01 code on main `d102fb4`:
  - `apps/console/src/modules/identity/pages/users/Live.tsx` (550 lines) and `live.css` (644 lines, shared);
  - `live-model.ts` in full; `live-copy.ts` grepped only.
- Shell and global styles:
  - `apps/console/src/app/screen-groups/identity.tsx:41-64` (the M01 wrapper `.remaining-security[data-security-screen="M01"]`, with `dir` and `lang`);
  - `app/service-shell.css` (rail 256px, main padding 32px; the rail moves on top at ≤1279px; padding is 16px at ≤767px; the header is not sticky);
  - `shared/styles.css` (checkbox 16px with `accent-color:var(--navy)`; the global `:focus-visible` is 2px `--hb-focus` with offset 2px; the global `dialog`; `tbody tr:hover`);
  - `service/service.css` (`.service-summary`, `.service-status`);
  - `screen-ui/security.css` (`.security-table-scroll`).
- Rules: `.agents/rules/service-screen-style.md`, `design-system.md`, `screen-review.md` and `usability.md`. `docs/design-system.md` and `docs/design-build-guide.md` were grepped only.
- Precedents: `docs/features/HYB-315/visual-spec.md` and `docs/features/HYB-297/visual-spec.md`.

**Not available or not done:**
- **No current M01 screenshots or renders were supplied, and I viewed none.** Statements about the current state come from reading the source.
- **No frontend-design skill was supplied.** None exists under the project `.claude`.
- **I had no browsing tool, so I opened no external reference** (§13).
- I ran no commands, builds or browsers. Nothing here is render-verified.

**Container widths** (inferred from service-shell.css; they match the measured P04 values cited in the HYB-315 spec; not measured on M01):

| Viewport | `.live-users` width | `useWide()` | Mode |
| --- | --- | --- | --- |
| 1440 | 1120 | true | split |
| 1280 | 960 | true | split |
| 1024 | 960 (rail on top) | false | step |
| 768 | 704 | false | step |
| 390 | 358 | false | step, stacked rows |

### Current-state observations (from source)

1. At ≥1280 the users view is always a two-column grid: the list, then a `clamp(360px,34%,440px)` panel. With no row open, the panel shows the "pick" sentence. **The list is therefore 707px at 1440 and 568px at 1280, even when nothing is open.**
2. The table has four columns (User 36%), no selection and no role period column.
3. The open row is marked with `.live-selected` (`--surface-hover`) and visually hidden text "(selected)" (`live-copy.ts:45`). **That word collides with the new checkbox selection** (§3.6).
4. Single-person actions are inline steps in the panel (`.live-step`), not dialogs.
5. The Revocation jobs tab is a fixed unavailable block (`Live.tsx:446-455`).
6. At ≤767px the table stacks through media rules in the shared `live.css`, and `thead` is visually hidden.

---

## 1. Composition principles

1. **Effect before progress.** The order is: who is selected → what will change → confirmation → what the server confirmed. Nothing on screen reads "revoked", "completed" or "succeeded" before the server says so.
2. **A ledger, not a dashboard.** Counts are plain tabular numbers in words and weight. There are no progress bars, `<progress>`, percentages, meters, ticks, crosses, countdowns or green/amber/red. Staging progress is text, so it cannot be read as revocation progress.
3. **The checkbox is the selection indicator.** A checked native checkbox carries the state. Checked rows get no background, outline or colour. `--surface-hover` stays reserved for the row whose detail is open.
4. **Full width, capped measure.** Regions span `.live-users`. Prose is capped at 68ch, inputs at 440px (the date-time input at `16rem`), and buttons are content width and 44px high at ≥768px. Buttons in action rows sit next to the content they act on. They are not pushed to the far edge of a 1120px row.
5. **Neutral geometry.** Separators are 1px `--border-row`, controls use `--border-control` and the dialog uses `--hb-line-header`. Radius 0, no shadows, no edge strips, no coloured selection outlines.
6. **Emphasis (600, ink) only for:**
   - problems: blocked, held, failed, unprocessed, decision not confirmed, ended, "Ends in";
   - permanence and scope sentences;
   - the first sentence of each condition.

   Nothing positive is emphasised.

---

## 2. Page structure

```
.remaining-security[data-security-screen="M01"][data-security-view=users|invitations|membership|revocation]
  div.live-users.m01-live  [data-wide] [data-open=true|false] [data-bulk=none|effect|preview]
    p.visually-hidden[role=status][aria-live=polite]          single announcer (existing)
    section.m01-conditions          only when a condition exists (§9); every view
    -- views users / membership / invitations, data-bulk="none":
    div.live-list.m01-list          container-type:inline-size; container-name:m01list
      .live-toolbar                 existing: h2 + Loaded at + Reload + Create user
      section.m01-selbar            selection bar (§3.4); users and membership only
      p.m01-selline                 invitations only: one line (§3.7)
      table                         with the checkbox column (§3)
      Show more users               existing
    aside.live-panel                detail / single-action steps (existing), incl. role period (§8)
    -- users / membership with data-bulk != "none":
    section.m01-bulk                full width; replaces list AND panel (§4, §5)
    -- view revocation:
    section.m01-jobs                (§7)
```

**Where the new pieces sit:**
- **Role periods** sit in the existing table, as a column, and in the detail facts. Changing a period is an action in the detail's Membership group, which opens an inline `.live-step` in the panel like every other single-person action.
- **Selection** sits in the users and membership tables, with the selection bar between the toolbar and the table.
- **The bulk flow** is a full-width workspace that temporarily replaces the list and the panel inside the same tab. There is no new tab and no new route.
  - Why replace the list: it freezes what the user sees while targets are staged; it gives the preview's impact groups room; and it makes the flow identical at every width (spec `M01/spec.md:112`).
  - The selection is kept, and "Back to selection" restores the list.
- **Jobs** go in the existing fourth tab, "Revocation jobs", which replaces the unavailable block.

**[R-1] Panel on demand (recommended).** In M01's users, invitations and membership views, render `aside.live-panel` only when a detail or step is open. With no row open, use one column, so the list is 1120 / 960 and the full column set fits (§3.2).
- The "pick" sentence becomes a one-line `p.live-meta` under the heading.
- Scope it as `[data-security-screen="M01"] .live-users[data-wide="true"][data-open="false"] { grid-template-columns:minmax(0,1fr) }` in `bulk.css`. The shared `live.css` stays unchanged.
- It changes the qa264 expectations.
- If root rejects it, the M column set (§3.2) is used at every wide width.

**Vertical rhythm:**
- conditions → toolbar: 24px; toolbar → selection bar: 16px; selection bar → table: 16px;
- inside `.m01-bulk` and job detail sections: 32px between sections, 12px from h3 to content;
- no rule above the first block after a heading.

---

## 3. Users and Membership tables with selection

### 3.1 Checkbox column

The column is rendered only when `can_bulk_revoke` is true and no operator-recovery state applies. Otherwise the column, the bar and the line are absent from the DOM; they are not disabled.

```
<col class="m01-col-check">                      inline-size:56px
th.m01-check  > label.m01-hit > input[type=checkbox] + span.visually-hidden
td.m01-check  > label.m01-hit > input[type=checkbox] + span.visually-hidden
.m01-hit { display:flex; align-items:center; justify-content:center;
           min-inline-size:44px; min-block-size:44px; margin-block:-10px; cursor:pointer; }
.m01-check input { inline-size:20px; block-size:20px; }   (24px under @media (pointer:coarse))
th.m01-check, td.m01-check { padding-inline:6px; vertical-align:top; }
```

- The −10px block margin centres the 44px hit area on the first text line (td padding 14px, line 24px).
- The **row checkbox** is labelled with the plan copy "Select {login}" (login in an ltr isolate; the user ID when there is no login).
- **The caller's own row [R-2]:** render no checkbox. Show `span.m01-self` (14px muted) "You" [TW], with the visually hidden reason [TW "You cannot revoke your own access."].
  - The server excludes `self` anyway.
  - Showing it up front prevents a guaranteed exclusion.
  - If root rejects this, render a normal checkbox; the preview then lists the exclusion.
- **Header checkbox:** labelled with the plan copy "Select the {n} loaded people", where `n` is the number of rows in this tab that have a checkbox. It never uses a word for "all". Its states:
  - `checked` when every loaded selectable row is selected;
  - `indeterminate` (set in JS) when some are;
  - unchecked otherwise.

  Activating it when checked deselects only the loaded rows. Selected people outside the loaded rows stay selected.
- **Row click does not toggle selection.** The login button keeps opening the detail, so the two actions never conflict.

### 3.2 Column sets (container query on `.m01-list`)

| Set | List container | Where it occurs | Columns (users view) |
| --- | --- | --- | --- |
| **L** | ≥880px | 1440/1280 with no row open (under R-1); 1024 step mode (960) | ☐ 56px · User 26% · Account status 14% · Workspace role 16% · **Role period 22%** · Sign-in setup (auto, about 191px at 1120) |
| **M** | 520–879px | 1440 with a row open (707), 1280 with a row open (568), 768 (704) | ☐ 56px · User 36% (login, email, then Sign-in setup as a 14px muted sub-line) · Account status 20% · **Role and period** [TW header] (auto: role, then the period lines) |
| **S** | ≤767px viewport | 390 | stacked rows (§3.8) |

Membership view:
- **L:** ☐ · User 28% · Workspace role 18% · Role period 22% · Membership status 18% · Installation administrator (auto).
- **M:** ☐ · User 36% · Role and period 34% · Membership status (auto). "Installation administrator" folds into a sub-line under the role.

Implementation:
- Render every cell once and hide or show the fold sub-lines and the folded columns with `@container m01list (max-width:879px)` (`display:none` on the `col`, `th` and `td` of a folded column, and on `.m01-fold` sub-lines at L).
- Hidden duplicates are removed from the accessibility tree by `display:none`.
- The M widths at 568 are 56 / 184 / 102 / 226.
- `table-layout:fixed` is kept, with the last column auto so the fixed 56px plus the percentages never sum above 100%.

### 3.3 Role period and revocation effect in cells

**Role period cell (L), or the period lines under the role (M).** Each value is two nowrap parts, so a date never breaks inside itself:

| `role_period` | Line 1 | Line 2 (14px muted) |
| --- | --- | --- |
| `open` | "No end date" (ink, regular; this is a real value, not an absence) | — |
| `bounded` | "Until {date}" | "{time} (UTC{offset})" |
| `ends_soon` | "Ends in {n} days" (**600**) | "Until {date}, {time} (UTC{offset})" |
| `ended` | "Role period ended" (**600**) | "Ended {date}, {time}" |
| no membership | the existing "No membership" text in the role cell; the period cell is "—" with a visually hidden [TW "No role period"]. Never blank. | |

- The plan's single string "Until 30 Nov 2026, 18:00 (UTC+09:00)" is split at the comma into two lines. That is a layout change, not a copy change; TW confirms that the split reads correctly in KO and AR.
- **"Ends in" is computed once per load from `server_time`.** It never ticks, never animates and never uses `setInterval`. The "Loaded at" meta is its base.

**Revocation effect sub-line.** It sits under the state it explains:
- `suspend_membership`: under the role (L) or the membership status;
- `suspend_user`: under the account status.

Its contents are the plan copy, "Suspended by bulk revocation {n}" plus "· decision not confirmed" (600 for that part), or "Role period ended". It is a 14px `span.live-sub`, and "bulk revocation {n}" is plain text, not a link (the link is in the detail, §8.1). The middle dot sits outside isolates.

### 3.4 Selection bar (`section.m01-selbar`)

```
section.m01-selbar[aria-labelledby=m01-selbar-h]
  display:flex; flex-wrap:wrap; align-items:center; gap:12px 24px;
  padding-block:12px; border-block:1px solid var(--border-row); margin-block:0 16px;
  h3#m01-selbar-h.visually-hidden            [TW "Selection for revocation"]
  label.live-check.m01-select-loaded          narrow only (§3.8); hidden at ≥768 via display:none
  p.m01-selcount[tabindex=-1]                 plan "37 selected" + limit, e.g. [TW "37 selected (at most 500)"]
                                              the number in <strong>; 16px
  div.m01-selactions (flex wrap, gap 12px)    button.live-secondary "Prepare revocation"; button.live-secondary "Clear"
  p.m01-selnote   (flex-basis:100%; 14px muted; 68ch)   plan filter-wide note, always shown
  p.m01-limit     (flex-basis:100%; 600; 68ch)          plan limit sentence, only at the limit
  details.m01-notloaded (flex-basis:100%)                only when some selected IDs are not in the loaded rows (§3.6)
```

- **With 0 selected:** `p.m01-selcount` is muted [TW "No one selected. Use the checkboxes to choose people."]. "Prepare revocation" and "Clear" are not rendered (never disabled), and the filter-wide note stays.
- **"Prepare revocation" is secondary, not primary.** The toolbar's "Create user" is the page primary, and the gateway to a destructive flow should not be visually urged. The flow's own steps carry their primaries.
- The buttons sit **directly after the count**, with no `margin-inline-start:auto`, so they stay within the same visual group at 1120px.
- **The bar is static, not sticky.** A sticky bar would cover focused rows (WCAG 2.4.11) when the list is long. The count is also announced (§10.3).

### 3.5 Limit behaviour

The limit is `-bulk-revocation-max-targets` (default 500) from the users read `settings[]`.
- **At the limit,** unchecked row checkboxes get `aria-disabled="true"` and keep their focusability. Activating one does not check it; `aria-describedby` points to `p.m01-limit`, and the limit sentence is announced. Never use native `disabled`: it removes the control from the tab order, so the reason cannot be found.
- **The header checkbox** refuses the whole action when selecting the loaded rows would pass the limit, with the same sentence. It never selects the first N silently.

### 3.6 Selection persistence and people outside the loaded rows

- The selection is a `Set` of user IDs held in `LiveUsers` state, above the view switch. It survives "Show more users", the Reload button and tab switches (spec `:81`).
- **[R-3] Browser reload.** I recommend in-memory only. A full page reload clears the selection, and the bar's empty state then shows. Do not write user IDs to `localStorage` or `sessionStorage` without a root decision: they are personal identifiers, and the storage boundary would need stating (`screen-review.md`, `tenant-security.md`).
- **People outside the loaded rows.** After Reload only the first page is loaded, so a selected person beyond it is *not loaded*, not *gone*.
  - The plan's "No longer in the list" would be untruthful for them (§12 item 4).
  - Layout: `details.m01-notloaded`, with summary [TW "{k} selected people are not in the loaded rows"] (44px). The body is `ul.m01-ids` (`columns:22ch auto; column-gap:24px; list-style:none; padding:0`).
  - Each `li` holds the login, or the ID as a `bdi dir=ltr` mono value, and a `button.service-button-link` [TW "Remove"] with the visually hidden login.
  - They stay counted, so the server decides.
- **Open-row marker.** Change the visually hidden "(selected)" on the open row to [TW "(open)"]. Add the visible `span.m01-tag` [TW "Open"] (`aria-hidden`; `aria-current` carries the meaning) after the login, as P05 and M02 do.
  - `.m01-tag`: `display:inline-block; padding:0 6px; border:1px solid var(--border-control); font-size:14px; line-height:22px; font-weight:600; color:var(--hb-ink); white-space:nowrap`.

### 3.7 Invitations tab

There is no checkbox column; the plan puts selection on Users and Membership only. If the selection is not empty, render `p.m01-selline` (14px, 68ch) [TW "{n} selected on the Users and Membership tabs."] under the toolbar. There is no Prepare button here.

### 3.8 Narrow (390)

- **Stacked row grid.** This overrides the shared stacked rules, scoped in `bulk.css`:
  - `tr { display:grid; grid-template-columns:44px minmax(0,1fr); column-gap:8px; }`;
  - `td.m01-check { grid-column:1; grid-row:1 / span 8; padding:0; }`;
  - every other `td { grid-column:2; }`.

  The checkbox sits beside the login, and the labelled cells follow beneath it.
- The `thead` checkbox is hidden with the thead. Instead, `label.live-check.m01-select-loaded` with the same accessible name ("Select the {n} loaded people") is shown inside the selection bar, as the first item, at 44px. Only one of the two exists in the accessibility tree at a time, through `display:none`.
- In the bar, `div.m01-selactions` stacks with full-width buttons, and the count comes first.
- The role period shows as the labelled cell "Role period: …", with its two parts on two lines.

---

## 4. Bulk flow, step 1: effect, reason and staging (`section.m01-bulk[data-bulk=effect]`)

```
section.m01-bulk (container-type:inline-size; container-name:m01bulk)
  button.live-back "Back to selection"        44px, underline, at the top at every width
  h2#m01-bulk-title (20/600, tabindex=-1, focused on open)   plan "Prepare revocation"
  p.m01-stepline (14px muted)                 [TW "Step 1 of 3: effect and reason"]  (text only; no stepper graphics)
  div.service-summary                         failures (§4.3), focused
  div.m01-bulk-grid   1 col; at @container m01bulk (min-width:880px): minmax(0,7fr) minmax(0,5fr); gap 48px
    form.m01-effect-form
      fieldset.live-roles.m01-effects
        legend (16/600)                       [TW "Effect"]
        label (44px row, radio 20px) + span.m01-opt (600) + p.m01-opt-hint (14px, 68ch, indented 32px)
          suspend_membership (checked by default)  hint: plan "Other workspaces are not changed"
          suspend_user                              hint: [TW from the plan effect table: suspends the account in every workspace]
      Reason (existing Reason component; textarea max 440px; byte counter)
      p.m01-progress[tabindex=-1]              staging line (§4.2); absent before sending
      div.live-buttons
        button.live-primary "Preview impact"   (the only primary)
    section.m01-selsum (aria-labelledby)
      h3 [TW "Selected people ({n})"]
      p (16px, 68ch, first sentence 600)       [TW "Nothing has been revoked. The server checks every person when you preview."]
      details.m01-sellist  summary [TW "Show the {n} selected people"] (44px)
        ul.m01-ids  (login or ID, bdi ltr)
```

The ID of the reason field is `m01-bulk-reason`. Do not reuse `live-reason`; the panel may also be mounted.

### 4.1 Widths

| Container | Layout |
| --- | --- |
| 1120 | form about 597px (fields capped at 440); summary about 427px |
| 960 | 509 / 363 |
| ≤879 | one column, summary after the form |

### 4.2 Staging line (`p.m01-progress`, 16px, 68ch)

- **While sending:**
  - the plan copy "Sending {sent} of {n} people";
  - the fields become `readonly`, and the radios get `aria-disabled` and are not changeable;
  - "Preview impact" is replaced by nothing, and its busy state lives in the line;
  - "Back to selection" is not rendered while sending [R-4];
  - focus moves to the line (`tabindex=-1`).
- **`rate_limited`:** the plan copy "The server is limiting requests. Sending continues in a few seconds ({sent} of {n} sent)." There is no countdown number and no animation.
- **`revision_conflict`:** the resend is silent. The line keeps its count, because the plan defines no copy. [TW optional: "Another change happened on the server. Sending the same people again."]
- **The HYB-293 step-up dialog** may open on the first batch. Staging resumes after it.
- **Announcements** go through the single region only at start, at a rate-limit pause and at the end. The visible line is not a live region, so per-batch updates are not read aloud.
- **No `<progress>` element and no percentage.**

### 4.3 Staging and prepare failures (in `div.service-summary`, focused)

- **Staging failure:** h2 [TW "Preview not prepared"]. Then the reason, the plan's staged count ("{k} of {n} people were sent"), and the plan sentence that nothing has been revoked (600).
  - Then **either** `button.live-secondary` [TW "Discard the draft"] **or** the sentence that it was discarded, depending on R-5.
- **Anchor rule blocked (`prerequisite_failed`, field `draft`):**
  - h2 [TW "Preview refused"];
  - `p` with the plan/spec `blocked` copy (600 first sentence);
  - h3 [TW "Administrators in your selection"] and `ul.m01-admins`, one `li` per selected person shown as administrator in the loaded rows: login (bdi), "Administrator" and the role period line, plus `button.live-secondary` [TW "Remove from selection"] (44px; content width at ≥768px).
  - Label the list as "in your selection", never "the administrators that block". Which person is an anchor depends on server data (§12 item 2).
- **Second open draft or pending limit (`prerequisite_failed`):** the plan copy, plus a link to the Revocation jobs tab where the open draft can be discarded (§7.3).
- **`permission_denied` or `unauthenticated`:** the whole-view states (§11).

---

## 5. Bulk flow, step 2: preview, and step 3: confirmation

### 5.1 Preview (`section.m01-bulk[data-bulk=preview]`; the same component is reused in the job detail for prepared manifests, §7.4)

```
button.live-back "Back to selection"           (leaves the manifest pending; focus returns to "Prepare revocation")
h2#m01-bulk-title (focused)                    [TW "Impact preview"] + span.live-meta "Bulk revocation {n}"
p.m01-stepline                                 [TW "Step 2 of 3: check the impact"]
p.m01-when (14px)                              plan "Prepared at 14:02 · valid until 14:32"
p.m01-lead (16px, 68ch; first sentence 600)    effect scope sentence [TW, e.g. "Suspends membership in this workspace for 37 people."]
dl.m01-counts                                  People to revoke · Excluded · Duplicates removed (§6.1 style)
div.m01-preview-grid   1 col; @container m01bulk (min-width:880px): minmax(0,1fr) minmax(0,1fr); gap 48px
  section.m01-impact   h3 [TW "Impact"] + p.live-meta "Computed at {time}"
  section.m01-excl     h3 [TW "Excluded ({n})"]
section.m01-targets    h3 [TW "People to revoke ({n})"]  (§5.3; depends on §12 item 3)
section.m01-outside    h3 [TW "Not changed by this revocation"]  ul (16px, 68ch) — outside-scope items
div.live-buttons       button.live-primary "Continue to confirmation"; button.live-secondary "Discard"
```

**Impact (`section.m01-impact`).** Groups follow the plan order. Each group is an `h4` (14/600, margin 16px 0 4px) plus `dl.m01-sum`:
- `div` rows use `grid-template-columns:minmax(140px,44%) minmax(0,1fr)`, gap 4px 16px, padding-block 8px, with no rule between rows;
- dt is 14px muted; dd is 16px tabular;
- below 420px of the section, dt sits above dd.

| h4 [TW] | Rows (plan content) |
| --- | --- |
| People and roles | People · Administrators · Members |
| Rights | Installation administrators · Trust-right holders · Bulk revocation right holders |
| Sessions and keys | Open console sessions · Outstanding secret pickups · API keys: plan "Not applicable: people hold no API keys" |
| Pending work | One row per proposal store: count, then "Rechecked at approval" (regular) or plan "Approval recheck not confirmed" (**600**) on its own line. Then Pending jobs: plan "Not applicable: …" |
| Service users | Count, then the plan line "Service users they own keep running. Review them in Service accounts." with "Service accounts" as a link to M04 (the spec allows the link) |
| Administrators afterwards | The anchor count [TW sentence]; 600 when it is 1 |
| Scope | For `suspend_membership`: plan "Other workspaces are not changed" |

**Exclusions (`section.m01-excl`):**
- `dl.m01-sum` by reason: Self, Not found, No membership, Already revoked. All four rows are shown, and a real 0 reads "0".
- Then `ul.m01-people` (paged, 50), each `li` with the login or ID and the reason word.
- `button.live-secondary` [TW "Show more excluded people"] while `next` exists, with the line [TW "Showing {a} of {b}."].

**Confirm-step entry:** "Continue to confirmation" opens the dialog (§5.2).

**Discard:**
- It sends `bulk_revocation.discard` and returns to the list with the selection kept.
- It announces [TW "The prepared list was discarded. Nothing was revoked."].
- Focus goes to `p.m01-selcount`.

**Expired** (now ≥ `expires_at` by server time at the last read, or `invalid_stage`):
- the summary block [TW "This preview expired. Prepare it again."];
- "Continue to confirmation" is not rendered;
- the actions become `live-primary` [TW "Prepare again"] and "Back to selection".

### 5.2 Confirmation dialog (`dialog.m01-confirm`, `showModal()`)

This follows the HYB-297 shell.

```
dialog.m01-confirm  inline-size:min(720px, calc(100vw - 32px)); max-block-size:90dvh; padding:24px; overflow:auto
  h2#m01-confirm-title (20/600, tabindex=-1, focused)     [TW "Confirm bulk revocation"]
  div.service-summary (role=alert, tabindex=-1)            errors
  dl.m01-confirm-facts   2 cols at dialog content ≥600px (gap 4px 32px); .m01-wide spans both
     List: "Bulk revocation {n}" + ID (bdi mono) · Effect · People to revoke · Excluded ·
     Prepared by · Prepared at / valid until · (wide) Digest: first 12 hex, mono, with 14px label [TW "first 12 characters"]
  p.m01-scope (600, 68ch)        effect scope sentence
  h3 [TW "Not changed"] + ul      outside-scope items (16px, 68ch)
  div.m01-consequence (68ch)      [TW "Access ends at once for these people and their sessions end."] (600);
                                  reversal [TW from the plan effect table: one person at a time]
  Reason (prefilled from preparation, editable; textarea rows=4; max 440px; id m01-confirm-reason)
  label.live-check (44px; checkbox 20px)   plan "I checked the people, the exclusions and the effect scope"
  div.live-buttons.m01-confirm-actions
     button.live-primary  plan "Revoke access for {n} people"
     button.live-secondary [TW "Go back"]
```

- **Exactly one primary.** It is never disabled. On submit, a missing checkbox or reason produces the error summary, linked to the field.
- **The secondary is "Go back".** Escape and "Go back" close without executing. Both are blocked while the request is in flight. The backdrop does not close the dialog.
- **Focus** returns to "Continue to confirmation", or to `h2#m01-bulk-title` if that button no longer exists.
- **Step-up:** Reauthenticate opens above the dialog, and the confirm primary shows busy. On success the same request is resent once, and focus returns to the primary. The stacked modals are UNVERIFIED.
- **Refusals inside the dialog** (as HYB-297 §8.3):
  - for `invalid_stage` (expired or executed), a wrong `digest`, `prerequisite_failed` (anchor at execute), `revision_conflict`, `idempotency_conflict` or `permission_denied`: the plan copy goes in the focused summary, **the primary is removed**, and the secondary becomes [TW "Reload and close"];
  - for anchor, use the spec `blocked` copy.
- **On `accepted`:**
  - close the dialog and switch to the Revocation jobs tab with the job open;
  - focus the job title;
  - announce the job state in words. Use `accepted` wording only, never "revoked" [TW "Revocation job started. Access is blocked first; cleanup continues."], subject to TW against the plan.
  - The executed targets are removed from the selection [R-6].
- **Narrow (<768px):** `inline-size:100vw; max-inline-size:100vw; block-size:100dvh; max-block-size:100dvh; margin:0; padding:16px; border:0`.
  - The facts are one column.
  - Buttons are stacked at 100% width in DOM order, and are **not sticky**.
  - The textarea uses `inline-size:calc(100% - 6px); margin-inline:3px`.

### 5.3 Target list in the preview

`section.m01-targets` is a table with Person 40% · Role 20% · Role period 25% · Installation administrator 15%, paged at 50 with "Show more". It is stacked at S.
- This needs a read of the prepared manifest's targets. The plan's items read has no pre-execution outcome for targets (§12 item 3).
- If root does not add one, render only the counts and the plan's exclusion list, and record that the spec's "대상 확인" is satisfied by count only.

---

## 6. Shared result patterns

### 6.1 Counts strip (`dl.m01-counts`)

- `display:flex; flex-wrap:wrap; gap:8px 32px; margin-block:12px 16px`. Each `div` has `min-inline-size:11ch`.
- dt is 14px muted; dd is **16px/600 `tabular-nums`**. There are no hero numbers and no 20px values.
- **Job order:** People (fixed denominator) · Succeeded · Held · Failed · Unprocessed.
  - A 1px `--border-row` inline-start separator (`padding-inline-start:32px`) sets off **Excluded (not counted)** as the last item.
  - That neutral separator is the only border in the strip.
- **Under the strip,** `p.m01-equation` (16px, 68ch) gives the full sentence [TW template, e.g. "37 people: 30 succeeded, 2 held, 1 failed, 4 unprocessed."], built per locale with the numbers inside it.
- No bar, no percentage and no ratio glyph.

### 6.2 State words

| Server value | Words (plan / [TW]) | Weight |
| --- | --- | --- |
| manifest `draft` | [TW "Draft, not prepared"] | 400 |
| `prepared` | [TW "Prepared, not executed"] + "valid until …" | 400 |
| `executed` + `job_state=running` | [TW "Running"] | 400 |
| `waiting` / `decision_unconfirmed` | [TW "Waiting: decision not confirmed"] + the HYB-298 sentence | **600** |
| `waiting` / `cleanup_not_started` | [TW "Waiting: cleanup not started"] | **600** |
| `succeeded`, failed = 0 | [TW "Finished"] + held > 0 sentence [TW "{h} people were held and not processed."] | 400 |
| `succeeded`, failed > 0 | the spec `partial` copy, then the spec `error` copy as a second paragraph (§12 item 8) | first sentence **600** |
| `discarded` / `expired` | [TW] | 400 |
| item `succeeded` | [TW "Revocation complete"] | 400 |
| item `held` | plan "Held: changed after preview" + plan "prepare again" sentence | **600** |
| item `failed` | [TW "Cleanup failed"] + reason | **600** |
| item `unprocessed` | [TW "Not processed yet"] / plan "unprocessed: record not confirmed" | **600** |
| item `excluded` | plan "Excluded" + reason | 400 |

"Succeeded", "complete" and "revoked" words appear only where the server value is `succeeded` or `revocation_complete`.

---

## 7. Revocation jobs tab (`section.m01-jobs`)

```
.live-toolbar   h2#m01-jobs-title "Revocation jobs" + p.live-meta "Loaded at …" + Reload
p.m01-note (14px muted, 68ch)   plan "Prepared lists from before the server started at … are not kept" (when started_at applies)
div.m01-jobs-split [data-mode=split|step] [data-open]
  div.m01-jobs-list (container-type:inline-size)
    h3 [TW "Not yet executed ({n})"]  table.m01-jobs-table   (caller's drafts, then prepared manifests)
    h3 [TW "Jobs ({n})"]              table.m01-jobs-table   (newest first)
    details.m01-ended  summary [TW "Discarded and expired ({n})"] (44px) → table
    button.live-secondary "Show more" while next
  aside.m01-job  (detail)
details.m01-settings  summary [TW "Installation settings"] (§7.6)
```

### 7.1 Split

- `.m01-jobs-split[data-mode=split][data-open=true] { grid-template-columns:minmax(320px,2fr) minmax(0,3fr); gap:32px }`. The list is about 435 / 371px and the detail about 653 / 557px at 1440 / 1280.
- `aside.m01-job`:
  - `border-inline-start:1px solid var(--border-row); padding-inline-start:32px`;
  - `position:sticky; inset-block-start:16px; max-block-size:calc(100dvh - 32px); overflow-y:auto; overscroll-behavior:contain`;
  - `tabindex=0`, with `outline-offset:-2px` on focus-visible.
- No empty panel. The mode comes from `useWide()` through `data-mode` only.

### 7.2 Jobs table

**Full** (list container ≥720px, no job open):

| Col | Width | Content |
| --- | --- | --- |
| Job | 26% | `button.m01-open` "Bulk revocation {n}" (600, underline, 44px via padding-block 10px), sub-line the effect in words, and [TW "From an earlier server run"] when `origin=ledger_replay` |
| State | 24% | §6.2 words; reason line 14px |
| People | 30% | the §6.1 equation sentence (16px); "+ {e} excluded" 14px muted |
| When | auto | prepared: "Valid until {time}"; executed: "Executed {date}" / "{time}", two nowrap parts |

- **With a job open:** reduced single-cell rows (opener, state word, the people sentence at 14px). There are no visible headers; the caption stays.
- **Stacked at 390,** with labelled cells.
- **The open row:** `--surface-hover` + `.m01-tag` [TW "Open"].

**HYB-332 amendment (2026-11-19):** "stacked at 390" is implemented as a container rule. When the jobs list container (`div.m01-jobs-list`) is below 600px, which includes the list column of the split layout with a job open (about 435px at 1440), the table stacks into labelled blocks like `.m01-settings`. With a job open, the split rules still hide every cell except the Job cell and show its compact lines. Stacked rows keep a 12px inline inset so text stays inside the open-row fill. In the stacked form `thead` is visually hidden, each row is a block with a row separator, and every cell after the Job cell starts with its column label ("{label}: ", 14px muted). Headers of the jobs and settings tables set `overflow-wrap: normal` (overriding the inherited `anywhere`), so a header word is not split. `span.m01-opener` is a block, so its box holds the 44px opener button. Stacked cells here and in the stacked `.m01-settings` table take their content height (`block-size: auto`), overriding the global 44px table-cell row height, so long setting names and job lines are not cut.

### 7.3 Job detail (executed)

```
.m01-detail-head   h2#m01-job-title (20/600, tabindex=-1) "Bulk revocation {n}" + sub-line effect ; "Close" (split only)
p.m01-state-line   §6.2 (68ch)
dl.m01-counts + p.m01-equation      (§6.1)
dl.m01-sum   Prepared by · Prepared at · Executed by · Executed at · Access blocked at (deny_at) ·
             Decision ledger: "Records {first}–{last}" (bdi mono) or "Not confirmed: {reason}" (600) ·
             Cleanup: [TW "{done} of {total}"] (§12 item 9)
section.m01-scope   h3 [TW "What cleanup covers"]
   three h4 groups: [TW "Done by the revocation"] · [TW "Not applicable"] · [TW "Outside the controlled scope"]
   each: ul of plan B-items in words with their plan reason (16px, 68ch); B10 (copies already shown or downloaded) 600
section.m01-items   h3 [TW "People ({n})"]
   div.m01-filter: service-field label [TW "Show"] + native select (44px, inline-size:calc(22ch + 40px))
       options: All · Succeeded · Held · Failed · Unprocessed · Excluded, each with its count
   ol.m01-items
   p.live-meta [TW "Showing {a} of {b}."] + button.live-secondary [TW "Show more people"]
details.m01-expert summary [TW "Technical details"]: manifest ID, full digest (mono, overflow-wrap:anywhere), watermark
```

**`li.m01-item`:**
- `border-block-start:1px solid var(--border-row); padding-block:12px`.
- Grid at item container ≥520px: `minmax(180px,36%) minmax(0,1fr)`, gap 4px 24px. One column below that.
  - **Left:** login (600, bdi), user ID (14px mono, bdi), role · role period (14px).
  - **Right:** the outcome word (§6.2) and reason; the revocation state words; the record "Record {sequence}" (mono) or "Not confirmed: {reason}" (600); `details.m01-cleanup` with summary [TW "Cleanup for {login}"] (44px) and the person's `cleanup[]` in the same three groups; the `replay_note` (14px).
- A list, not a table, because five fields plus a nested cleanup do not fit 557–653px as columns.

**Draft or prepared detail:**
- **Draft:** the facts plus `button.live-secondary` [TW "Discard the draft"] (owner only).
- **Prepared:** the §5.1 preview body (container-driven, so it stacks in the 557–653px pane), with "Continue to confirmation" (primary) and "Discard".
- **Expired or discarded:** facts only.

**No retry button anywhere** (spec `:100`).

### 7.4 Step mode (<1280)

With a job open, render only the toolbar heading, `button.live-back` [TW "Back to revocation jobs"] and the detail as `section.m01-job-step`.

### 7.5 Empty state

- Jobs: `p.live-empty` [TW "No revocation jobs."].
- Each group with 0 items shows its own empty sentence. A real 0 count reads "0".

### 7.6 Installation settings (`details.m01-settings`; also inside the role period form, §8.2)

- Summary 44px.
- Body: a table with Setting 34% · Value 14% (tabular, with unit) · Default 14% · Allowed 18% ("{min}–{max}") · Source 20% (words for `default` / `start_flag`). It stacks at S.
- Read-only, with no controls.

---

## 8. Role periods in the detail and the forms

### 8.1 Detail facts (`dl.live-facts`, after "Workspace role")

- **dt [TW plan "Role period"]**, dd as two lines:
  - "From {date}, {time} (UTC{offset})";
  - "Until …", or "No end date", or "Ended {date}, {time}" (600).
- An "Ends in {n} days" line (600) when `ends_soon`.
- "· decision not confirmed" (600) when `ledger=unconfirmed`.
- **dt [TW "Latest revocation"]** when `revocation` is present: `a.m01-link` "Bulk revocation {n}" (underline, 44px hit area). It switches to the jobs tab with that job open. The state words come after it.
- **The Membership action group** gains `button.live-secondary` "Change role period" (plan copy), placed after "Change role".
  - Rendered only when the server-side rules can allow it (not self, membership present).
  - A period-ended membership shows "Change role period" and "Remove", **not** "Reinstate" or "Change role". The plan refuses those two; under the existing `offers()` they would render.

### 8.2 Change role period form (`form.live-step.m01-period-form` in the panel or step)

Order:
1. `h2` (focused) plan "Change role period" + `p.live-who` login.
2. `p.m01-clock` (16px, 68ch) when `clock_behind`/`clock_ahead`: the plan sentence "The host clock differs from the server's time; role periods use the server's time."
3. The error summary.
4. `p.m01-current` (16px): [TW "Current: {period line}"].
5. **`fieldset.live-roles.m01-end`**, legend [TW "Role ends"], radios plan "End date" / "No end date", **pre-filled from the stored end**.
   - When the setter has an end, "No end date" is `disabled` as the plan requires.
   - Its reason `p.m01-why` (14px, ink) [TW "Your own role ends on {date}, so you cannot set no end date."] sits directly under that radio and is linked by `aria-describedby`.
6. **When "End date" is chosen:** `div.service-field.m01-when-field`:
   - label [TW "End date and time"];
   - hint (14px muted) with the plan copy "Times are in your browser's time zone (UTC{offset})";
   - `input[type=datetime-local][step=60][dir=ltr]`, `inline-size:16rem; max-inline-size:calc(100% - 6px); min-block-size:44px`. Its `min` is the server time + 1 minute and its `max` is the earlier of the server time + max days and the setter's end, both converted to local time. These are guidance only; the server validates.
   - under it, `p.m01-utc` (14px): [TW "Sent as"] + `bdi dir=ltr` mono "2026-11-30T09:00Z", updated as the user types;
   - `dl.m01-limits` (14px, dt above dd at panel width): [TW "Server time now"] · [TW "Latest allowed end"] · [TW "Your own role end"] (only when it limits).
7. `p.live-impact.m01-period-impact` (68ch), recomputed from the stored versus the chosen value:
   - the plan shortening sentence;
   - or the plan lengthening/restoring sentence, with its first sentence 600.
8. `details.m01-settings` (§7.6), collapsed.
9. The Reason (existing).
10. `div.live-buttons`: `live-primary` "Change role period" and `live-secondary` "Cancel" (the existing single-action pattern in M01).

- **Panel width** (360–440px): every control fits. No control is wider than 440px.
- **Task wording versus the plan:** the brief says "input in UTC with the user's local time shown". The plan specifies a local input converted to UTC. The plan wins. Step 6 shows both values: the local input and the UTC value sent.

### 8.3 "Role ends" in create, invite and change role

- The same `fieldset.m01-end` and date-time field (steps 5–6), placed after `RoleChoice` and before the reason.
- On change role it is pre-filled from the stored end.
- On create and invite it starts as "No end date", or with the setter's own end when the setter has one (plan).

### 8.4 Results and refusals for period writes

| Outcome | Placement |
| --- | --- |
| `completed`, `durable` absent or `yes` | Existing `.service-status` notice in the detail [TW done copy] |
| `durable: "no"` / `"unknown"` | `.service-status` with the plan sentence. The first sentence ("The change is in effect now…") is 600, and the restart sentence is a second paragraph |
| `ledger: unconfirmed` (a decrease) | Notice [TW], and the row and detail show "· decision not confirmed" |
| Cap (`invalid_request`, field `role_until`) | Inline field error under the date-time input, plus the summary link. Copy [TW], naming the limiting date |
| Bounds (not a whole minute, past, or beyond the maximum) | Same placement |
| Anchor (`prerequisite_failed`) | `FailureBox` with the spec `blocked` copy, and the administrator involved (the target) named in the body |
| Trust hold (HYB-297) | `FailureBox` with HYB-297's hold copy |
| Ledger cannot record an increase, or wall lead (`dependency_unavailable`) | `FailureBox` with HYB-298's extended `busy` copy |
| Equal end (`invalid_stage`) | `FailureBox` [TW "The end is unchanged."] |
| Reinstate or change role on an ended period (`invalid_stage`) | `FailureBox` with the plan copy pointing to "Change role period" |

---

## 9. Conditions (`section.m01-conditions`, top of every M01 view)

- Rendered only when at least one condition applies, with `aria-labelledby` pointing to a visually hidden h2 [TW "Notices"].
- Each `div.m01-condition`: padding-block 16px, `border-block-end:1px solid var(--border-row)` (the first also gets `border-block-start`).
- Paragraphs are 16px and 68ch. The first sentence is in `<strong>`, and each later sentence is its own `p`. No fill, icon or coloured edge.

Order:
1. **Wall lead beyond the tolerance:** the plan EN/KO/AR copy, plus `dl.m01-inline` (14px; flex wrap; gap 4px 24px) with [TW "Host clock"] and [TW "Server time"], both in ltr isolates.
2. **Ledger not recording** (`isolated`, `stopped`, `full`): HYB-298 copy.
3. **Clock base not recorded:** [TW sentence; the plan names only the state "clock base not recorded"], plus the chosen base (`ledger` / `wall`) and both times.
4. **Clock base chosen and recorded:** a 14px muted fact line only, not a condition, placed under the toolbar [TW].
5. **Clock differs** (`clock_behind` / `clock_ahead`): the plan sentence. This is a diagnostic, so the first sentence is not 600. It is repeated in the period form (§8.2).

**In step mode with an open bulk step or detail,** render `p.m01-condition-short` (the first sentence of each, 600) under the back link.

---

## 10. Focus, keyboard and announcements

### 10.1 Tab order (users view, split)

1. shell tabs;
2. condition links;
3. Reload, Create user;
4. the selection bar: the select-loaded checkbox (narrow only), Prepare revocation, Clear, the not-loaded summary and its Remove buttons;
5. the table: the header checkbox, then for each row its checkbox and then its login button;
6. Show more users;
7. the panel.

### 10.2 Focus moves

| Trigger | Focus target |
| --- | --- |
| Prepare revocation | `h2#m01-bulk-title` |
| Back to selection | Prepare revocation |
| Preview impact | `p.m01-progress` |
| Prepared | the preview h2 |
| Failure | the summary |
| Continue | the dialog h2 |
| Go back / Escape | Continue to confirmation |
| Accepted | `h2#m01-job-title` |
| Discard | `p.m01-selcount` |
| Remove from selection (blocked list) | the next Remove button, or the summary when none is left |
| Close / Back in jobs | that row's opener |

Add `:focus:not(:focus-visible){outline:none}` only on `tabindex=-1` headings, notes and summaries, never on controls. The global 2px `--hb-focus` ring is never overridden.

### 10.3 Keyboard and live region

- Space toggles a checkbox, and Tab and Shift+Tab move. There is no Shift-click range selection and no row-click toggle. Range selection is not provided and is not claimed.
- **Selection count:** announced through the single polite region, debounced at 500ms after the last change (plan "37 selected"). The limit sentence is announced at once.
- **Job state:** announced once per change on reload. Rows are never re-announced.

---

## 11. Whole-view states

| State | Presentation |
| --- | --- |
| Loading | Existing sentence (`role=status`); no skeleton numbers |
| Load error | Existing `.live-error` |
| Denied | Existing `.live-unavailable` |
| Revoked / unauthenticated | Existing signed-out handling. Selection, the bulk section, the dialog and the job detail are removed from the DOM, and any open dialog is closed |
| Operator recovery | `.live-unavailable` with the plan sentence in every view. **No command controls, no checkbox column, no selection bar, no bulk section**; read-only lists stay |
| `can_bulk_revoke` false | No checkbox column, bar or line; the jobs tab is still readable (the read uses `administer_installation`) |

---

## 12. Problems in the plan and decisions for root

1. **Staging failure contradiction (Medium).** `plan.md:846` says a staging failure *discards* the draft, while `:833` says it *offers* to discard it. §4.3 supports both; root picks one [R-5].
2. **Deselecting after the anchor refusal (High).** The draft stays unfrozen, but there is no remove-target command, and the caller may hold only one open draft.
   - After "Remove from selection", the console must discard the draft and stage again.
   - The plan does not say this, and the user should be told [TW "Preview again to send the new selection."].
   - Also, which selected administrators block depends on server data: the anchor count and the role ends. The client list can only say "administrators in your selection".
3. **No read of the targets before execution (High, spec `:75` "대상 확인").** Items have the outcomes `excluded|held|succeeded|failed|unprocessed` only. The preview can list exclusions but not the people to be revoked. Either add a target outcome for prepared manifests or accept count-only confirmation explicitly.
4. **"No longer in the list" is untruthful after a list reload (Medium).** Only page 1 is reloaded. Use [TW "not in the loaded rows"] (§3.6); only the server can say a person is gone.
5. **"Ends in {n} days" for less than one day (Medium, TW).** Rounding is undefined, and there is no copy for n = 0. Plurals in KO and AR are needed (the plan uses `shared/plural.ts` for AR).
6. **"(selected)" means the open row today (Low).** It collides with checkbox selection. Rename it to "open" (§3.6).
7. **Selection across browser reloads (Medium) [R-3].** Recommend in-memory only.
8. **Partial and error copy on the same job (Low, TW).** Both describe failed cleanup items. I show both, partial first.
9. **The unit of `cleanup {done,total}` is undefined (Low).** Is it items or chunks? The label depends on it.
10. **Selection after execute (Low) [R-6].** Recommend removing the executed targets.
11. **Panel on demand (Medium) [R-1]**, **no checkbox on the caller's own row (Low) [R-2]**, and **no Back link while sending (Low) [R-4]**.
12. **`offers()` would render Reinstate and Change role on a period-ended membership** that the plan refuses (§8.1). Fix it in `live-model.ts` (logic, not layout).
13. **"Back to selection" from a prepared preview** leaves a pending manifest until it expires. It is visible in the jobs tab ("Not yet executed"); no other reminder is specified.

---

## 13. External references

Root opened the references on 2026-11-18. What was observed, adopted and rejected is in [references.md](references.md): Entra PIM, GitHub member removal, Google Workspace suspension, Carbon data table, GOV.UK check answers and GOV.UK checkboxes. Okta was not opened and nothing is adopted from it.

## 14. Tokens, type, classes and item IDs

**Tokens (existing only):**
- `--hb-ink`, `--text-muted`;
- `--border-row`, `--border-control`, `--hb-line-header` (the dialog);
- `--surface-page`, `--surface-hover` (the open row only), `--surface-primary`, `--text-on-primary`;
- `--hb-focus`, `--hb-font-mono`;
- `--hb-danger` only through the existing `.live-error` and `.service-summary` headings.

The native checkbox fill comes from the global `accent-color:var(--navy)`, which is an explicit control state. No literals, radii, shadows, gradients or new tokens.

**Type:**
- h2 20/600; h3 16/600; h4 14/600; body 16; labels and sub-lines 14; mono 14;
- prose 68ch; `tabular-nums` on counts.
- AR keeps the console line-height of 1.8 (none is set in `.m01-*`). KO uses `word-break:keep-all` on headings and prose. No italics.

**Spacing:** 4 / 8 / 12 / 16 / 24 / 32 / 48.

**RTL:**
- Logical properties only. The checkbox column is at inline-start (the right in AR), and the job detail pane is on the left in AR.
- IDs, digests, sequences, UTC values and the `UTC+09:00` offsets use `bdi dir=ltr`. Each date and time part is an FSI isolate (the P04 Stamp pattern). The date-time input is `dir=ltr`.
- Free text (rationales) uses `span.m01-free[dir=auto]`, never `bdi`.
- No → ← ✓ ✗ glyphs. Count sentences are whole per-locale templates.

**Classes** (all in `users/bulk.css`, scoped under `[data-security-screen="M01"]` or `dialog.m01-confirm`; reuse `live-*`, `service-*`):
- Frame: `m01-live`, `m01-conditions`, `m01-condition`, `m01-condition-short`, `m01-inline`, `m01-note`.
- Selection: `m01-list`, `m01-col-check`, `m01-check`, `m01-hit`, `m01-self`, `m01-fold`, `m01-tag`, `m01-selbar`, `m01-selcount`, `m01-selactions`, `m01-selnote`, `m01-limit`, `m01-notloaded`, `m01-ids`, `m01-select-loaded`, `m01-selline`.
- Bulk: `m01-bulk`, `m01-stepline`, `m01-bulk-grid`, `m01-effect-form`, `m01-effects`, `m01-opt`, `m01-opt-hint`, `m01-progress`, `m01-selsum`, `m01-sellist`, `m01-admins`, `m01-when`, `m01-lead`, `m01-counts`, `m01-equation`, `m01-preview-grid`, `m01-impact`, `m01-sum`, `m01-excl`, `m01-people`, `m01-targets`, `m01-outside`.
- Confirm: `m01-confirm`, `m01-confirm-facts`, `m01-wide`, `m01-scope`, `m01-consequence`, `m01-confirm-actions`.
- Jobs: `m01-jobs`, `m01-jobs-split`, `m01-jobs-list`, `m01-jobs-table`, `m01-open`, `m01-ended`, `m01-job`, `m01-job-step`, `m01-detail-head`, `m01-state-line`, `m01-items`, `m01-item`, `m01-filter`, `m01-cleanup`, `m01-expert`, `m01-settings`, `m01-free`.
- Period: `m01-period-form`, `m01-end`, `m01-why`, `m01-when-field`, `m01-utc`, `m01-limits`, `m01-current`, `m01-period-impact`, `m01-clock`, `m01-link`.

**Proposed implementation item IDs** (`screen-review.md`). Root must reconcile them with the existing M01 IDs and must not renumber those:
- Users and selection: `M01:users:select-column`, `M01:users:select-loaded`, `M01:users:selection-bar`, `M01:users:selection-limit`, `M01:users:selection-not-loaded`, `M01:users:filter-wide-note`, `M01:users:role-period-column`, `M01:users:revocation-effect`.
- Detail and period: `M01:detail:role-period`, `M01:detail:latest-revocation`, `M01:action:change-role-period`, `M01:form:role-ends`.
- Bulk flow: `M01:bulk:effect`, `M01:bulk:staging`, `M01:bulk:blocked`, `M01:bulk:preview`, `M01:bulk:exclusions`, `M01:bulk:targets`, `M01:bulk:confirm`.
- Jobs: `M01:jobs:list`, `M01:jobs:detail`, `M01:jobs:counts`, `M01:jobs:items`, `M01:jobs:cleanup-scope`, `M01:jobs:settings`.
- Conditions and states: `M01:conditions:clock`, `M01:conditions:ledger`, `M01:state:operator-recovery`.

---

## 15. Rendered-QA checklist (measurable)

For every render, record the viewport, locale, view, state, and the measured widths of `.live-users`, `.m01-list`, `aside.live-panel`, `.m01-bulk` and `aside.m01-job`.

**Layout and widths**
1. `.live-users` is about 1120 / 960 / 960 / 704 / 358 at 1440 / 1280 / 1024 / 768 / 390. On every render, including the open dialog and the expanded details, `document.scrollingElement.scrollWidth <= clientWidth` holds, and no `.m01-*` element has `scrollWidth > clientWidth` except the intended table scroll wrapper.
2. **Column sets:**
   - L, 6 columns including the checkbox, at a list width of ≥880 (1440/1280 with no row open under R-1, and 1024);
   - M at 707 and 568 (a row open) and at 704;
   - stacked at 390.
   - No `th` text overflows its cell (EN "Sign-in setup", "Role period"; AR; KO).
   - The checkbox column is 56px.
3. **Hit area:** the bounding box of each `label.m01-hit` is ≥44×44, and the checkbox is 20px (24px under a coarse pointer).
4. **Bulk section:**
   - full width, with no list or panel in the DOM;
   - fields ≤440px and the date-time input 16rem;
   - computed prose `max-inline-size` ≤68ch;
   - buttons content width and 44px at ≥768px, 100% at 390;
   - the preview grid has two columns at 1120 and 960 and one at 704 and 358.
5. **Confirm dialog:**
   - width ≤720px at 1440;
   - exactly one `.live-primary`;
   - the digest shown is 12 hex characters with its label;
   - full screen at 390 with non-sticky buttons;
   - the focus ring is not clipped.
6. **Jobs split:** list about 435 / 371 and detail about 653 / 557. The divider's computed `border-inline-start` is 1px in the `--border-row` colour. In AR the detail is on the left.

**Selection behaviour**
7. Select 3 people, then Show more users, then select 2 more, then switch to Membership: the same 5 are checked. On Invitations the 5-count line shows. After Reload, 5 are counted and the people outside the loaded rows are listed with Remove.
8. The header checkbox is indeterminate with a partial selection. Its accessible name contains the loaded count and no word for "all" in EN, KO or AR.
9. With the limit set to 3, the 4th checkbox does not check: it is `aria-disabled`, still focusable, and announces the sentence. The header checkbox refuses the whole action and selects nothing partially.
10. Checked rows have no background, outline or border change; only the checkbox changes. The open row has `--surface-hover` plus the "Open" tag.

**Truthfulness**
11. During staging, preview, confirmation and the `accepted` announcement, `.m01-live` and the dialog contain no word for revoked, complete or completed, or succeeded, in any locale. The only exception is the spec's negated `partial` sentence. Those words appear only beside server values `succeeded` or `revocation_complete`.
12. There is no `<progress>`, no `%` character and no ✓ ✗ → ← inside `.m01-live` or the dialog. The staging line is text only.
13. **Counts invariant on screen:** the dd values for People equal Succeeded + Held + Failed + Unprocessed. Excluded is outside the sum and labelled "not counted".
14. "Ends in" does not change after 60 seconds without a reload, and no timer re-renders the table.

**Role period**
15. All four period variants render in the column and the detail. "Ends in" and "Role period ended" are 600. Dates are not split inside a date or time part.
16. **The form:**
    - "No end date" is disabled with a visible reason when the setter has an end;
    - the input has `step=60`, `min` and `max`;
    - the zone line and the "Sent as … Z" line update;
    - a cap refusal places an inline error under the input, linked from the summary.

**Conditions and styling**
17. Conditions render in the §9 order, with the first sentence 600 (the clock-differs line is not 600) and the 68ch cap.
18. **Computed-style sweep over `.m01-live *` and `dialog.m01-confirm *`:**
    - border colours are only `--border-row`, `--border-control`, `--hb-line-header` or `--hb-ink` (an invalid field only);
    - no `border-inline-start` wider than 1px;
    - `border-radius` 0;
    - no `box-shadow`;
    - backgrounds only `--surface-page`, `--surface-hover` (the open row) or `--surface-primary`.

**Focus, RTL and KO**
19. **Focus:** every move in §10.2 lands as listed, with a visible ring after keyboard moves and no ring after URL-driven selection. The Escape and Go back focus return works with the step-up dialog stacked.
20. **RTL (AR):**
    - the checkbox column is on the right;
    - IDs, digests and `UTC+…` read left to right;
    - rationales are in `span[dir=auto]`, not `bdi`;
    - line-height is 1.8.
21. **KO:** `keep-all` gives no mid-word breaks in the bar, the step lines or the state lines.
22. **Not covered:** owner acceptance, terminology correctness, native-speaker review, real ledger failures beyond the plan's QA scenarios, and screen-reader behaviour beyond the live-region checks.

---

## Files referenced

- docs/features/HYB-299/plan.md
- docs/ui/screens/M01/spec.md
- docs/ui/screens/M01/design.json
- apps/console/src/modules/identity/pages/users/Live.tsx
- apps/console/src/modules/identity/pages/users/live.css
- apps/console/src/modules/identity/pages/users/live-model.ts
- apps/console/src/modules/identity/pages/users/live-copy.ts
- apps/console/src/app/screen-groups/identity.tsx
- apps/console/src/app/service-shell.css
- apps/console/src/shared/styles.css
- apps/console/src/service/service.css
- apps/console/src/modules/identity/screen-ui/security.css
- docs/features/HYB-315/visual-spec.md
- docs/features/HYB-297/visual-spec.md
- .agents/rules/service-screen-style.md
- .agents/rules/design-system.md
- .agents/rules/screen-review.md
- .agents/rules/usability.md
- docs/design-system.md
- docs/design-build-guide.md

## Open uncertainties

- Root decisions R-1 to R-6, and §12 items 2, 3, 5, 8 and 9.
- All [TW] strings.
- External references unopened (§13).
- No current M01 renders were supplied.
- Container widths are inferred, not measured.
- The stacked step-up and confirm modals are unverified.
- The ar digit system in dates is unverified.
