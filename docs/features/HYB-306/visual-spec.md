---
type: DesignSpecification
title: HYB-306 M05 recent access history visual specification
description: Layout source for the HYB-306 "Recent access history" region on the M05 Sessions view. Covers its placement after the session list, the "Recent session ends" region and the Session limits panel; the record table (time, event with reason, session, device); the retention, size-limit, coverage and minimisation statements; every note state; the period filter and Show more paging; loading, failed, none-kept and empty states with null kept apart from zero; focus, keyboard and announcements; RTL; widths 1440/1280/1024/390; rendered-QA checks; and what must not appear. Written by the ui-designer role (read-only). The HYB-306 plan wins on copy, states and truthfulness; this spec wins on layout.
status: draft
tags: [HYB-306, M05, access-history, sessions, retention, visual-spec]
sources:
  - resource: plan.md
  - resource: ../HYB-307/visual-spec.md
  - resource: ../../ui/screens/M05/spec.md
  - resource: ../../ui/terminology/product-glossary.md
---

# HYB-306 visual specification: M05 recent access history (service mode)

**Author:** ui-designer role, read-only. The environment reports the model as Claude Opus 5.5 (`claude-opus-5-5`).

**Status:** this is a draft design proposal, not owner acceptance. A later visual PASS would not be owner acceptance either. It would also not certify the design quality of anything outside the inspected scope.

**Precedence:**
- Where this spec and `docs/features/HYB-306/plan.md` differ on copy, states or truthfulness, the plan wins.
- This spec wins on layout, hierarchy, sizing, focus and responsive behaviour.
- The HYB-307 visual spec (§16 root decisions included) stays binding for "Recent session ends". Nothing here changes that region.

**Markers:**
- **[TW]** marks a string that the layout needs and the plan does not define, or a plan string that needs technical-writing review. EN drafts only; not final copy.
- **[R-n]** marks a decision for root (§12).
- **KO verbatim** marks KO taken word for word from `docs/ui/screens/M05/spec.md`.

---

## 0. Evidence basis and limits

**Read directly:**
- the plan: approved, four design rounds; §1, §6, §7, the copy table, Browser QA and residuals were read in full;
- the HYB-307 visual spec (in full);
- `my-security/Live.tsx` (lines 380–512), `end.css` (in full), and `live-copy.ts`/`end-copy.ts` (grepped);
- `users/live.css`: lines 1–110 and 268–407;
- `users/bulk.css`: lines 378–437, the HYB-332 container-query stacking pattern;
- M05 spec lines :22, :34, :59–61, :77, :91, :95, :102 and :119 (grepped);
- glossary line 66 (`password-confirmation`);
- `.agents/rules/service-screen-style.md`.

**Screenshots inspected** (HYB-307 round r2, Sessions view): `M05_result_en_1440.png`, `M05_result_ar_1440.png`, `M05_result_en_390.png` and `M05_held_ko_390_2x_crop.png`. I did not open the `qa307shots-branch/` set or compare it with r2.

**Observed in those renders (direct):**
1. At 1440 the list column is about 707px (measured from image scale) and the Session limits panel about 381px. A 1px neutral rule separates them.
2. The panel ends with "**Pre-launch:** recent access history is not available." (EN, AR) and is short: it ends at about y=1000 of a 3425px page.
3. "Recent session ends" (h3 16/600) sits under the Sessions toolbar. With five operations, two of them open, it pushes the session table to about y=3200 at 1440 and about y=4700 at 390.
4. At 390 everything is one column: the session table is stacked and Session limits comes after it.
5. AR mirrors correctly, and handles and times read left to right.
6. KO keep-all wrapping holds in the 390 crop.

**Inferred, not measured:** that `.live-users` is 1120 / 960 / 960 / 358 at 1440 / 1280 / 1024 / 390 (HYB-307 §0 table, consistent with observation 1).

**Not available:**
- no frontend-design skill was supplied;
- no external reference was opened (§11);
- I did not read `docs/design-system.md` or `design-build-guide.md` in this session;
- nothing here is render-verified.

---

## 1. Composition principles

1. **A ledger, newest first.** It is one quiet table with neutral hairlines. There are no charts, counters, badges, icons, colours, timelines or grouping cards.
2. **The truth statements come before the rows.** What is kept, for how long, from when and what is never recorded sits directly under the heading. Problem notes follow, then the filter, then the table. The rows are read inside those statements.
3. **Problems are carried by words and weight, never by colour.**
   - Weight 600 is used only for refused sign-ins, failed password confirmations, "recording stopped", "deletion paused" and "deletion overdue".
   - Nothing positive is emphasised.
4. **The region is separate from "Recent session ends".**
   - Different heading level: this region uses an h2 at 20/600; "Recent session ends" is an h3 at 16/600 inside Sessions.
   - Different place: this region is a full-width band below both columns.
   - Different name: the term "history" is never used for the ends operations.
5. **Full width with a capped measure.**
   - The section spans the whole `.live-users` grid.
   - Prose is capped at 72ch.
   - Controls are content width and 44px high and are never stretched at ≥768px.
6. **Neutral geometry.**
   - 1px `--border-row` separators.
   - The existing `.service-status` and `.service-summary` boxes (1px `--border-control`).
   - Radius 0, no shadows, and no inline-start or inline-end strips.
   - The global focus ring is unchanged.

---

## 2. Placement and page structure

```
.remaining-security[data-security-screen="M05"][data-security-view="sessions"]
  div.live-users[data-live=sessions][data-wide]
    p.visually-hidden[role=status][aria-live=polite]     single announcer (existing say())
    div.live-list.live-sections
      section[aria-labelledby=mysec-sessions-title]      toolbar, changed note, Recent session ends,
                                                         pre-launch note, session table (unchanged)
      section limits                                      narrow only (unchanged, minus historyNote)
    aside.live-panel                                      wide only: Session limits (minus historyNote)
    section.mysec-hist[aria-labelledby=mysec-history-title]   NEW, grid-column: 1 / -1
  dialog.mysec-end                                        (unchanged)
```

- **DOM order** is Sessions, then Limits, then History at every width, so reading order matches visual order.
- **Wide:** the history band starts below the taller of the two columns.
- **Narrow:** the band follows Session limits.
- **Grid and spacing:**
  - `[data-security-screen="M05"] .live-users > .mysec-hist { grid-column: 1 / -1; min-inline-size: 0; }`
  - The 32px `.live-users` gap applies above the section.
  - The section has `border-block-start: 1px solid var(--border-row); padding-block-start: 24px`. This neutral hairline ends the two-column zone. It is a separator, not an accent.
- **Removed:**
  - `prelaunch(m.historyNote(l))` at `Live.tsx:468`;
  - the `historyNote` key in `live-copy.ts`.
  - The panel then ends after its facts.
- **Component and styles:**
  - The region is its own component, `AccessHistory.tsx`. `Live.tsx` only mounts it, and stays near its current length.
  - Styles go in `my-security/history.css`, scoped under `[data-security-screen="M05"]`.
  - `users/live.css` and `end.css` are not edited.
- **Why not inside the list column:**
  - At 1280 that column is about 568px, which would force stacked rows on a desktop.
  - The history would also compete with the ends region for the same column.
  - Spanning gives 1120 / 960 / 960 / 358, so the four-column table works at every desktop width (§9).
  - This matches the plan's "at full width".

---

## 3. Region anatomy

```
section.mysec-hist  (container-type:inline-size; container-name:m05hist)
  div.mysec-hist-head   display:flex; flex-wrap:wrap; align-items:baseline; gap:4px 16px
    h2#mysec-history-title.live-title (20/600, margin 0)   plan "Recent access history"
                                                           (KO 최근 접근 기록, AR سجل الوصول الأخير)
    p.live-meta  [TW "Loaded at {time}"]   (existing w.loadedAt wording; from the read's `now`, with seconds)
  div.mysec-hist-policy   margin-block-start:12px; max-inline-size:72ch            (§4)
  div.service-status.mysec-hist-notes   margin-block-start:16px; max-inline-size:80ch  (§5; only when notes exist)
  div.mysec-hist-controls  margin-block-start:24px                                    (§6)
  div.security-table-scroll.live-table.mysec-hist-table  margin-block-start:16px       (§7)
     or p.live-empty (empty period) / p.live-empty (none kept)                         (§8)
  div.mysec-hist-more  margin-block-start:16px                                         (§6.3)
```

- Vertical rhythm: heading → policy 12px → notes 16px → controls 24px → table 16px → more 16px.
- Spacing scale is 4 / 8 / 12 / 16 / 24 / 32 only.

---

## 4. Statements (`div.mysec-hist-policy`)

These are always shown when a read has succeeded, in this order. Each is its own `p` with `margin:0`; consecutive `p` have 8px between them.

| # | Content | Type | Condition |
| --- | --- | --- | --- |
| 1 | Plan Retention copy, sentence 1: "Kept for {days} days." + when `retention_source = start_flag`: " (set for this installation)" | 16/400, `--hb-ink` | always |
| 2 | Plan Retention copy, sentences 2–end (hidden at once; deleted within one day while the server runs, otherwise at the next start before it accepts sign-ins; legal hold; backups) | 14/400, `--hb-ink` | always |
| 3 | Plan "Size limit" line with `{effective}` | 14/600 | only when `effective_days < retention_days` |
| 4 | Plan Coverage "History available from {time}." | 14/400 | `available_from` non-null |
| 4' | Plan "No history is kept yet." | 14/400 | `available_from` null |
| 5 | Plan Minimisation "Network addresses and locations are not recorded." + plan Not shown "Expired sessions are not listed as events." | 14/400, muted (`--text-muted`) | always |

- **[TW]** Rows 1 and 2 need the plan's single Retention string split into two per-locale keys at the sentence boundary. The meaning must not change. KO and AR are AI drafts (plan R9).
- Row 2 at 14px, capped at 72ch, runs about four lines in EN at 1440. It stays visible and is never collapsed into a disclosure: the plan says these statements are always visible.
- `{days}` and `{effective}` use the locale's plural forms (AR: every form).
- No statement claims more than the plan:
  - no "permanently deleted";
  - no "secure";
  - no "complete history";
  - no "all devices";
  - nothing about backups beyond the plan sentence.

---

## 5. Notes (`div.service-status.mysec-hist-notes`)

- **Container:** one existing `.service-status` box: 1px `--border-control`, no edge strip, padding as existing. It holds `ul` (list-style none, padding 0); each `li` is one sentence, with 8px between them.
- **When shown:** the box is rendered only when at least one note applies. It is never shown as an empty box, and there is no "No problems" sentence.
- **Not a live region.** It has `aria-labelledby` pointing to a visually hidden h3 [TW "Notices about this history"].

**Fixed order and weight:**

| Order | Source | Copy (plan unless marked) | Weight |
| --- | --- | --- | --- |
| 1 | `recording.state = stopped` | Recording stopped, `{time}` = `since` | 600 |
| 2 | note `purge_deferred` | Deletion deferred | 600 |
| 3 | note `deletion_overdue` | Deletion overdue with `{days}` | 600 |
| 4 | note `held` (own count) | Held with `{count}`, `{days}` | 400 |
| 5 | note `early_deletion` | Early deletion with `{time}` = `to` | 400 |
| 6 | note `dropped` with `count` ≥ 1 | Dropped (own count) | 400 |
| 6' | note `dropped` with `count` null | Possibly missing (no count), `{from}`, `{to}` | 400 |
| 7 | note `unclean_stop` | Unclean stop, `{time}` = `from` | 400 |
| 8 | note `damaged`, count non-null | Damaged with `{count}` | 400 |
| 8' | note `damaged`, count null | Possibly missing (no count) | 400 |
| 9 | note `recovered_tail` | [TW "The end of a stored file was incomplete around {time} and was repaired. Events just before that may be missing."] | 400 |
| 10 | note `future_file` | [TW "Some stored events are dated after the server's current time. The server clock may have been changed."] | 400 |

**Null versus zero (spec `:91`):**
- A note with `count: null` never prints a number and never prints "0". It always uses the "may be missing" sentence.
- A note whose `count` is 0 is not rendered. The server should not send one; if it does, it is ignored and nothing is announced. [R-3] confirms this.
- An unknown `kind` makes the parser fail, which gives the failed state (§8). It is never silently dropped.

**Formatting and repeats:**
- All `{time}`, `{from}` and `{to}` use the same local time format with seconds as the table (§7.2), each wrapped in `bdi dir=ltr` in AR.
- Several notes of the same kind each get their own `li`, in `from` order newest first.

---

## 6. Controls

### 6.1 Period filter (`div.mysec-hist-controls`)

```
div.mysec-hist-controls   display:flex; flex-wrap:wrap; align-items:flex-end; gap:12px 32px
  div.mysec-hist-period
    label[for=mysec-hist-period] (14/600, display:block, margin-block-end:4px)
          [TW EN "Period"]; KO verbatim "최근 접근 기간"; AR [TW "فترة الوصول"]
    select#mysec-hist-period (native; min-block-size:44px; inline-size:auto; min-inline-size:14rem;
          max-inline-size:20rem; 1px --border-control; radius 0; font 16)
      options (plan): Last 24 hours · Last 7 days · Last 30 days · Everything kept
  div.mysec-hist-scope   (max-inline-size:56ch)
    p (14)  [TW "Workspace: {id}"]   (id via existing ltr(), bdi dir=ltr)
    p (14, muted) [TW "Events from your other workspaces are shown when you are signed in to them.
                  Refused sign-ins are shown in every workspace."]
```

- **Default:** "Everything kept" [R-1]. The first read sends no `from`, and the page is bounded by the cursor (`limit` 50). "Last N" is computed as `from = now − N` using the `now` of the most recent successful read, never the browser clock.
- **Applying a change:** the change applies on `change`. There is no Apply button: a native select changed by keyboard fires `change` only on commit, and this is a single read-only filter. Applying:
  - discards the cursor and the loaded rows;
  - keeps focus on the select;
  - replaces the table area with the loading line (§8);
  - announces once (§10.3).
- **Coverage line:** when the chosen `from` is before `available_from`, `p.live-meta` under the select reads [TW "Nothing before {time} is kept."]. Otherwise there is no line.
- **No other filter controls.** There is no event-type filter, no search and no date-range picker. The workspace is a fixed fact, not a control (plan §7, spec `:60` "본인 소속 업무 공간" is satisfied by the session scope, R6).

### 6.2 Reload

- The region has no Reload button of its own.
- The Sessions toolbar Reload and every end result (HYB-307 `showOp` after a 202, and `closeEnd` reload paths) also reload page 1 of the history, keeping the selected period. This makes "Session ended: by you" appear (QA Q3).
- Such reloads move no focus and announce nothing.
- No polling.

### 6.3 Show more (`div.mysec-hist-more`)

- **Button:** `button.live-secondary.mysec-hist-more-btn`, content width, 44px, rendered only while `next` is non-null. Label: plan "Show more"; [TW] alternative "Show older events", which says the direction.
- **Busy:** label [TW "Loading older events…"], `aria-busy=true`, disabled, same width or wider. No spinner.
- **On success:** rows are appended below the existing ones, keyed by `at` + kind + handle + position, with no re-sort. Focus:
  - if `next` is still non-null, focus stays on the button;
  - if `next` is null, the button is removed and `p.mysec-hist-end[tabindex=-1]` [TW "No older events in this period."] (14, muted) takes focus.
- **On failure (not 401):**
  - the loaded rows stay;
  - `div.service-summary.mysec-hist-more-failed[tabindex=-1]` under the table shows the plan's Load error copy with the request reference;
  - then `button.live-secondary` "Try again" (existing `c.tryAgain`), which resends the same cursor;
  - focus moves to the summary.
- **On 401:** the existing signed-out view.

---

## 7. Record table (`.mysec-hist-table`)

### 7.1 Structure

```
div.security-table-scroll.live-table.mysec-hist-table
  table[role=table]  inline-size:100%; table-layout:fixed
    caption.visually-hidden  [TW "Recent access history, newest first"]
    colgroup: Time 20% · Event 40% · Session 20% · Device 20%
    thead th (14/600): [TW "Time"] · [TW "Event"] · [TW "Session"] · [TW "Device"]
    tbody tr (min-block-size:56px; border-block-end:1px solid var(--border-row); vertical-align:top)
      td.mysec-hist-time    (14, tabular-nums)
      td.mysec-hist-event   line 1 label (16); line 2 reason (14, --hb-ink) when present
      td.mysec-hist-session (14)
      td.mysec-hist-device  (14)
```

- `data-label` is set on every `td` except the first (for stacking, §9).
- Rows have no actions, checkboxes, links or hover fill. They are not focusable. The history is read-only evidence; ending a session stays in the session table and the end dialog.

### 7.2 Cells

**Time:**
- Local time with seconds, from `at`, as two `span.mysec-nw` parts (`white-space:nowrap`): date, then time. They sit on one line when the cell fits and wrap between the parts otherwise.
- Use the existing `when()` with a seconds variant, the same as the HYB-307 operation times.
- AR: wrapped in `bdi dir=ltr`.
- No relative time ("2 hours ago"), no tooltips.

**Event:**
- Line 1 is the plan event label. Weight 600 for `sign_in_refused`, `sign_in_refused_more` and `password_confirmation_failed`; 400 for all other events.
- Line 2 is the reason as its own short fragment (`display:block`, 14).
- [TW] The plan writes "Session ended: by you". The layout needs label and reason as separate per-locale keys, without the colon.

| kind / reason | Line 1 (plan) | Line 2 |
| --- | --- | --- |
| `signed_in` | Signed in / 로그인 / تسجيل الدخول | — |
| `sign_in_refused` `wrong_password` / `locked` / `access_refused` | Sign-in refused / 로그인 거부 | Wrong password / Account locked / Access refused (plan) |
| `sign_in_refused` `credentials_changed` | Sign-in refused | [TW "Sign-in details changed during sign-in"] |
| `sign_in_refused` `session_unavailable` | Sign-in refused | [TW "Session could not be opened"] |
| `sign_in_refused_more` | plan "{count} more refused sign-ins" (plural forms; AR all forms) | [TW "Not listed one by one on this day"] |
| `password_confirmed` | Password confirmed / 비밀번호 재확인 (glossary `password-confirmation`) | — |
| `password_confirmation_failed` | Password confirmation failed / 비밀번호 재확인 실패 | `wrong_password` [TW "Wrong password"]; `attempt_limit` plan "Too many wrong passwords" |
| `signed_out` | Signed out / 로그아웃 | — |
| `session_ended` | [TW "Session ended"] | plan: By you · Password changed · Too many wrong passwords · Access changed · By an administrator · Session limit · Sign-in rules changed (`trust_change`) · Bulk revocation; [TW] `password_reset` "Password reset"; [TW] `role_period_ended` "Role period ended" (glossary `role-period`) |
| `password_changed` | Password changed / 비밀번호 변경 | — |

**Session:**
- Line 1 is the state word: `current` gives plan "This session"; `active` and not current gives plan "Signed in now"; otherwise no line 1.
- Line 2 is the reference: `span.live-mono` (14) `bdi dir=ltr` 16-hex handle, muted only when a line 1 exists.
- With no `session`, show "—" plus visually hidden [TW "No session"].
- [R-2] covers this two-line form.

**Device:**
- [TW] formatter from the fixed vocabulary only: "{Browser} on {OS}", for example "Chrome on macOS".
- Both values `unknown`: plan "Unknown device".
- One value `unknown`: [TW "Unknown browser on {OS}"] or [TW "{Browser} on an unknown system"].
- `other`: [TW "Other browser"] or [TW "other system"].
- `sign_in_refused_more` has no device: "—" plus visually hidden [TW "Not recorded"].
- Brand names stay Latin in every locale and are wrapped in `bdi` in AR.

### 7.3 Never in the table

- IP, network, location or a raw User-Agent string;
- login ID, email or display name;
- `seq`, the cursor, or a record count of any kind for the whole history;
- other people's events;
- a workspace column (every row is this workspace or workspace-less, per F8).

---

## 8. States (null, zero and failure are distinct)

| State | Region content below the heading | Not shown |
| --- | --- | --- |
| Loading, first read | `p.live-meta` [TW "Loading your access history…"]; no statements yet (they come from the read) | statements, notes, controls, table, empty sentence |
| Loading after a period change | statements, notes and controls stay; the table area shows the same loading line | old rows, the Show more button |
| Read failed (network, 5xx, parse failure, unknown enum) | `div.service-summary.mysec-hist-failed` with the plan's Load error copy (request reference in `bdi` mono) + "Try again" (`live-secondary`, 44px) | statements, notes, table and the empty sentence. A failure is never shown as empty or as "no history" |
| 401 / revoked | the existing page-level signed-out view; the history rows, cursor and period state are cleared with the page (spec `:91`) | everything in the region |
| None kept (`available_from` null) | statements with row 4'; notes if any; controls; then `p.live-empty` plan "No history is kept yet." in place of the table | the table and the "No access events in this period." sentence |
| Empty period (`events` empty, `available_from` set) | statements, notes, controls; `p.live-empty` plan "No access events in this period." | the table head |
| Recording stopped | note 1 (600) **above** the controls, and the table or empty sentence as usual. When stopped and empty, both are shown, so emptiness is never read as "nothing happened" | — |
| Rows present | the table and Show more per §6.3 | — |

A real zero only appears inside a note count (never 0) or a plural. No "0 events" summary exists anywhere.

---

## 9. Responsive layout

The container is `section.mysec-hist`, named `m05hist`.

| Viewport | `.mysec-hist` (inferred) | Table | Policy / controls |
| --- | --- | --- | --- |
| 1440 | 1120 | 4 columns: Time 224 · Event 448 · Session 224 · Device 224; time on one line in EN | policy 72ch (about 540px at 14px), left aligned; controls on one line |
| 1280 | 960 | 4 columns, 192 / 384 / 192 / 192; time may wrap date / time | same |
| 1024 | 960 | same as 1280; Session limits sits in the list column above the band | same |
| 390 | 358 | stacked (below) | controls stack: period block above scope block; the select becomes `inline-size:100%`, the existing ≤767px control convention; Show more 100% width |

**Stacked mode, `@container m05hist (max-width: 599px)`** (the HYB-332 pattern):
- `thead` is visually hidden.
- `table`, `tbody`, `tr` and `td` use `display:block`.
- `tr` has `padding-block:12px; padding-inline:12px; border-block-end:1px solid var(--border-row)`.
- `td` has `padding:2px 0; border:0; block-size:auto`.
- `td[data-label]::before { content: attr(data-label) ": "; color: var(--text-muted); font-size:14px }`, except the first cell. Time stands first with no label, then Event at 16, then Session and Device.
- The DOM order is unchanged.

**Overflow:**
- `.security-table-scroll` is the only element allowed to scroll horizontally.
- At ≥600 container width no row may need it. Handles at 14px mono are about 135px; the narrowest Session column (192px) fits them.

---

## 10. Focus, keyboard and announcements

### 10.1 Tab order (in the region)

1. The period select.
2. Show more, or Try again.

The summary and the end line are programmatic focus targets only. Everything is reachable by keyboard alone; there are no hover-only affordances.

### 10.2 Focus moves

| Trigger | Target |
| --- | --- |
| Period change | stays on the select |
| Show more succeeded, `next` non-null | stays on the button |
| Show more succeeded, `next` null | `p.mysec-hist-end` |
| Show more failed | `div.mysec-hist-more-failed` |
| Try again (first load) succeeded | `h2#mysec-history-title` (tabindex=-1) |
| Background reloads (toolbar Reload, end results) | none |

- Use `:focus:not(:focus-visible){outline:none}` only on the tabindex=-1 heading, summary and end line, never on controls.
- The global `--hb-focus` ring is never overridden.

### 10.3 Announcements

All announcements go through the existing single polite announcer (`say()`); no new live regions.

| Event | Announced once |
| --- | --- |
| Period change, rows loaded | [TW "Showing {period}: {n} events, newest first."], + [TW "More are available."] when `next` is non-null. `{n}` is the caller's own rows on the screen, never a total |
| Period change, empty | the empty or none-kept sentence |
| Show more | [TW "{k} older events added."] |
| Any user-initiated read failed | the Load error copy |
| First load, background reloads | nothing |

Notes and statements are never announced. They are read in place, in DOM order before the table.

---

## 11. RTL, KO and external references

**RTL and KO:**
- Logical properties only.
- AR:
  - the table and the controls mirror;
  - times, handles, workspace IDs and request references are in `bdi dir=ltr`;
  - line-height stays at the console's 1.8;
  - every count uses `plural()` with zero, one, two, few, many and other.
- KO:
  - `word-break: keep-all` on h2, labels, statements, notes, event labels and reasons;
  - the KO filter label is verbatim "최근 접근 기간" (spec `:60`).
- No arrows, ticks, crosses, italics or icons.

**External references (not opened; root to open and record per `service-screen-style.md`):**
- Google Account "Recent security activity";
- GitHub Settings "Security log";
- Microsoft account "Recent activity";
- GOV.UK Design System "Table" and "Select".

**Assumed from them, inference only:**
- a newest-first event list with time, event and device;
- no editing actions on history rows.

**Rejected in advance by the plan (Q3):** any IP, location or map display such services may show. Report any contradicting pattern.

---

## 12. Decisions for root

- **[R-1] Default period "Everything kept".**
  - The first read then needs no client-side clock, every later `from` uses the server `now`, and paging already bounds the fetch.
  - The alternative, "Last 30 days", needs a `GET /me` `now` before the first history read.
- **[R-2] Two-line Session cell.**
  - This shows the state word above the reference, where the plan says "or". The reference is kept so a row can be matched to the session table and the ends region.
  - Rejecting it means showing only the word when one applies.
- **[R-3] Note counts.** A note with `count` 0 is not rendered. Confirm that the server never sends one.
- **[R-4] Missing plan copy.** The plan copy table has no strings for:
  - `recovered_tail` and `future_file`;
  - the reasons `credentials_changed`, `session_unavailable`, `password_reset`, `role_period_ended`;
  - the `sign_in_refused_more` line 2;
  - the device fallbacks.

  The [TW] drafts above are placeholders; the plan and TW must own the final text.
- **[R-5] Retention string split.** The plan's Retention string is split into a 16px lead sentence and a 14px body (§4) for hierarchy, with no wording change.
- **[R-6] Item ID reconciliation.** Proposed item IDs (`screen-review.md`), not renumbering the existing M05 IDs:
  - `M05:history:region`, `M05:history:retention`, `M05:history:size-limit`, `M05:history:coverage`, `M05:history:minimisation`, `M05:history:notes`;
  - `M05:history:period-filter`, `M05:history:workspace-fact`, `M05:history:table`, `M05:history:show-more`;
  - `M05:history:state-loading`, `M05:history:state-failed`, `M05:history:state-none-kept`, `M05:history:state-empty`.

**Critique against the brief:**
- The band below both columns can be far down the page when several end operations are open: at 1440 it would start at about y=3300 in the r2 result render. The ends region already collapses older operations (HYB-307 §5.6), and the Sessions tab is the IA entry point, so I add no jump link. If the owner finds it too deep, the fallback is an in-page link in the Sessions toolbar, which is a new IA element and needs root's decision.
- The always-visible retention paragraph is long. Splitting it into a lead and a 14px body keeps it truthful without hiding it.

---

## 13. Rendered-QA checklist

Record for every render: the viewport, locale, state, and the measured widths of `.live-users`, `.mysec-hist` and the table columns.

**Layout**
1. `.mysec-hist` spans the full `.live-users` width (about 1120 / 960 / 960 / 358) and starts below both the list column and the panel. Its computed `grid-column` covers both tracks.
2. The panel and the narrow limits section no longer contain "recent access history is not available" in any locale.
3. `scrollWidth <= clientWidth` on the document in every state. At ≥600 container width the table wrapper does not scroll.
4. The table has 4 columns at 1440, 1280 and 1024 and is stacked at 390. In stacked mode the first cell has no label and the others carry their `data-label`.
5. The select and the buttons are 44px high. At ≥768 they are content width, the select is 14rem–20rem, and no button is wider than its label plus 40px.
6. Prose is capped: policy `max-inline-size` ≤ 72ch; notes ≤ 80ch.
7. The order below the heading is policy → notes → controls → table or empty → more.

**Truthfulness**
8. No IP-like string (`\d+\.\d+\.\d+\.\d+`, `:`-hex groups), no "Mozilla/", no city or country name, and no login ID or email anywhere in the region's DOM.
9. No total count in the region. Note counts only appear when non-null, and a null count shows the "may be missing" sentence with no digit.
10. A failed read never shows "No access events in this period." or "No history is kept yet."
11. With recording stopped and an empty period, both the stopped note and the empty sentence are present.
12. "Kept for {days} days" matches `retention_days`. "(set for this installation)" appears only for `start_flag`. The size-limit line appears only when `effective_days < retention_days`.
13. The "Recent session ends" region is unchanged (the `qa307.mjs` assertions pass). Its heading text never contains "history"/"기록"/"سجل", and this region's heading never contains "ends".

**States** (stubbed reads per QA Q6)
14. Each of the following renders with its plan copy, in §5 order, KO and AR included:
    - recording stopped, dropped with a count, possibly missing, damaged, early deletion, held, deferred, overdue, unclean stop;
    - recovered_tail and future_file with [TW] copy;
    - the size-limit line, empty period, none kept, failed load, and Show more paging to the end line.

**Styling sweep over `.mysec-hist *`**
15. Border colours are only `--border-row` and `--border-control`, and no inline-start or inline-end border differs from the block borders. `border-radius` is 0 and there is no `box-shadow`. Backgrounds are only `--surface-page`. There is no `animation-name` other than none.

**Focus, RTL and KO**
16. Every move in §10.2 lands as listed, with a visible ring after keyboard moves. Keyboard alone can change the period and page to the end.
17. Exactly one announcement per §10.3 event, and none on first load or on background reloads (checked in the announcer text log).
18. AR:
    - mirrored table;
    - times, handles and references read left to right;
    - plurals correct for 0, 1, 2, 3, 11 and 100 in "{count} more refused sign-ins" and the note counts.
19. KO: no mid-word breaks in the labels, event names, reasons or statements at 390 and 1280.
20. **Not covered:** owner acceptance, terminology correctness, native-speaker review, and screen-reader behaviour beyond the announcer and reading-order checks.

---

## 14. Tokens, type and classes

**Tokens (existing only):** `--hb-ink`, `--text-muted`, `--border-row`, `--border-control`, `--surface-page`, `--hb-focus`, `--hb-font-mono`. No new tokens, literal colours, radii or shadows.

**Type:**
- h2 20/600;
- event label and the lead statement 16;
- everything else 14;
- `th` 14/600;
- mono 14;
- `tabular-nums` on times.

**Classes:** `mysec-hist`, `mysec-hist-head`, `mysec-hist-policy`, `mysec-hist-notes`, `mysec-hist-controls`, `mysec-hist-period`, `mysec-hist-scope`, `mysec-hist-table`, `mysec-hist-time`, `mysec-hist-event`, `mysec-hist-session`, `mysec-hist-device`, `mysec-hist-more`, `mysec-hist-more-btn`, `mysec-hist-more-failed`, `mysec-hist-failed`, `mysec-hist-end`. Also reused: `mysec-nw`, `live-*`, `service-status`, `service-summary`.

## Open uncertainties

- [R-1] to [R-6]; every [TW] string; the KO and AR drafts (plan R9).
- Widths are inferred, not measured on this region.
- External references were not opened.
- Nothing is render-verified.
- Only four r2 screenshots were inspected; the branch set was not compared.

## 15. Root decisions (2026-11-19)

Binding over the matching proposals.

1. **Placement (full-width band below both columns):** accepted.
2. **[R-1]:** default period "Everything kept" accepted.
3. **[R-2]:** two-line Session cell (state word above the reference) accepted; it amends the plan's "or".
4. **[R-3]:** the server never sends a note with count 0 (tested); the console ignores one if it arrives.
5. **[R-4]:** the [TW] drafts are adopted as drafts; the plan copy table gains these keys and the technical-writing review owns the final text.
6. **[R-5], [R-6]:** accepted.
7. **Depth on the page:** no jump link in this ticket; root reviews the renders.
8. **References (§11):** not opened; no new layout pattern beyond the existing M05/M01 table, notice and stacked-table patterns, so none is recorded as evidence.
