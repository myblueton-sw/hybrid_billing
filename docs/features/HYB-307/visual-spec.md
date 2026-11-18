---
type: DesignSpecification
title: HYB-307 M05 end other sessions visual specification
description: Layout source for the HYB-307 changes to the M05 Sessions view. It covers the entry point, the confirmation dialog with a fixed session list, the result object that shows "Access blocked" apart from "Cleanup", sessions held because they started after the list, sessions already gone, single-session End, failure and step-up states, polling, focus, RTL, widths and rendered-QA checks. Written by the ui-designer role (read-only). The HYB-307 plan wins on copy, states and truthfulness; this spec wins on layout.
status: stable
tags: [HYB-307, M05, sessions, IAM-07, visual-spec]
sources:
  - resource: plan.md
  - resource: ../../ui/screens/M05/spec.md
  - resource: ../../ui/screen-binding-fields/identity.md
  - resource: ../HYB-299/visual-spec.md
  - resource: ../HYB-298/plan.md
  - resource: ../HYB-293/plan.md
  - resource: ../../ui/terminology/product-glossary.md
---

# HYB-307 visual specification: M05 end other sessions (service mode)

**Author:** ui-designer role, read-only. The environment reports the model as Claude Opus 5.5 (`claude-opus-5-5`).

**Status:** this is a draft design proposal. It is not owner acceptance. A later visual PASS would not be owner acceptance either, and it would not certify the design quality of anything outside the inspected scope.

**Precedence:** where this spec and `docs/features/HYB-307/plan.md` differ on copy, states or truthfulness, the plan wins. This spec wins on layout, hierarchy, sizing, focus and responsive behaviour.
- **[TW]** marks a string that the layout needs and the plan does not define, or a plan string that needs technical-writing review. I supply EN drafts only so the layout can be built. They are not final copy.
- **[R-n]** marks a decision for root (§13).
- **KO verbatim** marks a KO string taken word for word from `docs/ui/screens/M05/spec.md`.

---

## 0. Evidence basis and limits

**Read directly:**
- the plan, `/private/tmp/hb307/docs/features/HYB-307/plan.md` (in full, revision after security round 1);
- `docs/ui/screens/M05/spec.md` (in full; lines :51, :72, :77, :85, :87–89, :91, :95, :101 and :102 are used);
- the current console, `apps/console/src/modules/identity/pages/my-security/Live.tsx` (467 lines), `live-copy.ts` and `live-model.ts`;
- the shared `users/live.css`: lines 1–135, and lines 600–645 (the M05 `mysec-*` rules);
- `service/service.css`: `.service-summary`, `.service-status` and `.service-reauth`;
- `service/Reauthenticate.tsx`: grepped (a native `dialog.service-reauth` with `showModal()`);
- the HYB-298 plan, Console section (`plan.md:699-768`), and the HYB-293 plan (grepped for the dialog and the step-up flow);
- the HYB-299 visual spec, in full, as the layout precedent;
- the product glossary, grepped for the revocation, outside-scope, حجب and session rows;
- the M05 rows of `docs/ui/screen-binding-fields/identity.md`;
- rules: `.agents/rules/service-screen-style.md`, `design-system.md`, `screen-review.md` and `usability.md`.

**Not available or not done:**
- **I received no current M05 screenshots or renders and viewed none.** Statements about the current state come from reading the source, not from inspecting renders. This means I could not do the brief's "inspect actual current screenshots before proposing" step.
- **No frontend-design skill was supplied.** None exists under `/private/tmp/hb307`.
- **I had no browsing tool and opened no external reference.** §12 lists candidate references for root to open and record. Nothing in this spec is claimed as observed from them.
- I did not read `docs/design-system.md` or `docs/design-build-guide.md` in this session. Tokens below are limited to those already used by `live.css`, `service.css` and the HYB-299 spec.
- I ran no commands, builds or browsers. Nothing here is render-verified.

**Container widths.** These are inferred from `live.css:1-9` (`.live-users` grid, 32px gap, panel `clamp(360px,34%,440px)`) and the shell widths that the HYB-299 spec records. They were not measured on M05.

| Viewport | `.live-users` | `useWide()` | List column | Panel (`aside.live-panel`) |
| --- | --- | --- | --- | --- |
| 1440 | 1120 | true | 707 | 381 (Session limits) |
| 1280 | 960 | true | 568 | 360 |
| 1024 | 960 | false | 960 | none; limits below the list |
| 390 | 358 | false | 358 | none; limits below the list |

### Current-state observations (from source)

1. The Sessions view is a list column (toolbar, notice, pre-launch note, sessions table) plus, when wide, a panel that shows Session limits. Ending one session or all others replaces the panel content with an inline `.live-step` (`Live.tsx:443`). Below 1280 the step replaces the whole list (`stepping`, `Live.tsx:350`, `:444-449`).
2. The end-others confirmation shows only a count sentence (`live-copy.ts:175-181`). It does not show the sessions, the time the list was taken, or a separate statement that this session is excluded.
3. The result is one notice, "Ended N" (`Live.tsx:405`, `:430`, `live-copy.ts:183-191`). It does not separate blocking from cleanup and does not show per-session results.
4. The confirmation is not a dialog, so the spec's focus trap (`:102`) and "read targets and impact in the dialog" (`:101`) are only partly met today.
5. `.service-status` and `.service-summary` use a full 1px neutral `--border-control` box with no edge strip (`service.css:115-136`). This is compliant and is reused.

---

## 1. Composition principles

1. **Blocking and cleanup are two separate facts, not one success.** Every result shows two labelled lines, "Access blocked" and "Cleanup", with equal visual weight. Nothing is rendered as a single "done" message (spec `:72`, `:77`).
2. **The confirmed list is the contract.** The confirmation shows the exact frozen list with "List taken at {time}". The result reports against that list: what was ended, what had already ended, and what started afterwards.
3. **Held sessions come before detail.** Sessions that started after the list are a possible security signal (plan T1). They appear directly under the two lines, above the per-session detail, with their End actions.
4. **A ledger, not a dashboard.** The result has:
   - plain tabular counts;
   - no progress bars, `<progress>`, percentages, spinners, countdowns, ticks or crosses;
   - no green, amber or red.

   State is carried by words and weight (600 for problems only).
5. **Full width with capped measure.** Regions span the list column. Prose is capped at 68ch. Buttons are content width and 44px high. Buttons sit directly after the content they act on and are never pushed to the far edge of the row.
6. **Neutral geometry.**
   - Separators are 1px `--border-row`. Boxed notices use the existing 1px `--border-control` `.service-status` and `.service-summary`. The dialog uses the global dialog border.
   - Radius is 0, with no shadows and no edge strips or coloured outlines.
   - The global focus ring is unchanged.
7. **Weight 600 is used only for:**
   - the "This session is not included" sentence;
   - "Not finished" and the failed reasons;
   - the first sentence of the held paragraph;
   - the cannot-be-recalled sentence.

   Nothing positive is emphasised.

---

## 2. Page structure (Sessions view)

```
.remaining-security[data-security-screen="M05"][data-security-view="sessions"]
  div.live-users[data-live=sessions][data-wide]
    p.visually-hidden[role=status][aria-live=polite]     single announcer (existing)
    div.live-list.live-sections
      section[aria-labelledby=mysec-sessions-title]
        .live-toolbar        h2 "Sessions" + count + Loaded at | tools: End all other sessions · Reload
        p.service-status.mysec-changed[tabindex=-1]       only after a list-changed reload (§8.2)
        section.mysec-ends   "Recent session ends" (§5); only when session_ends[] is non-empty
        div.live-note-rule   pre-launch stale note (existing)
        div.mysec-sessions   sessions table (existing) + held tag (§4.3)
        p.live-empty         "You have no other sessions." (existing)
      section limits          (narrow only, existing)
    aside.live-panel          Session limits (wide only). Never replaced by an end step any more.
  dialog.mysec-end            end confirmation, others or one (§3)
```

**Change from today.** Both end confirmations move from the inline panel or step into one modal `dialog.mysec-end` with two variants. Reasons:
- the spec calls it a dialog that reads targets and impact and traps focus (`:101`, `:102`);
- the frozen list must not be confused with the live table behind it;
- the result then sits in the same place at every width.

The `stepping` branch for the Sessions view (`Live.tsx:350`, `:444-449`) and the `end`/`endAll` modes no longer render panel content. The Sign-in methods view keeps its password-change step unchanged. [R-1] covers the fallback.

**Vertical rhythm:**
- toolbar → changed notice or ends region: 16px;
- ends region → pre-launch note: 32px;
- inside the ends region: 24px between operations and 12px from a heading to its content;
- inside an operation: 16px between blocks.

---

## 3. Confirmation dialog (`dialog.mysec-end`)

### 3.1 Shell

Use the native `<dialog>` with `showModal()`, the same mechanism as `Reauthenticate.tsx`.
- `inline-size: min(640px, calc(100vw - 32px))`, `max-block-size: 90dvh`, `overflow: auto` and `padding: 24px`.
- It uses the global dialog border and background. Do not add a blur, shadow, radius or new backdrop treatment.
- Below 768px viewport width it is full screen: `inline-size:100vw; max-inline-size:100vw; block-size:100dvh; max-block-size:100dvh; margin:0; padding:16px; border:0`. Buttons are stacked at 100% width in DOM order and are not sticky. This is the HYB-299 §5.2 pattern.
- The backdrop does not close the dialog. Escape and Cancel close it, except while a request is in flight. While a request is in flight, the `cancel` event is prevented and Cancel is not rendered.
- `aria-labelledby` points to the h2 and `aria-describedby` points to `p.mysec-end-lead`.

### 3.2 Variant "others": entered from "End all other sessions"

```
h2#mysec-end-title (20/600, tabindex=-1, focused on open)   existing endAllTitle "End all other sessions?"
p.live-meta[role=status]          while the fresh read runs: [TW "Getting your current sessions…"]
div.service-summary[role=alert][tabindex=-1]   failures (§3.5)
p.mysec-end-lead (16, 68ch)       [TW "Access ends for the {n} sessions listed below. Whoever uses them is
                                   signed out on their next request."]
p.mysec-end-scope (16/600, 68ch)  plan "This session is not included and stays signed in."
p.live-sub                        [TW "This session: reference {ref}"]  (ref = current handle, mono, bdi ltr)
div.mysec-manifest-head           h3 (16/600) [TW "Sessions to end ({n})"]
                                  p.live-meta  plan "List taken at {time}"   (time from the read's `now`)
div.live-table.mysec-manifest     table, caption = the h3 text (visually hidden)
                                  cols: Reference 40% · Signed in 30% · Ends by 30%
p.mysec-end-note (14 muted, 68ch) [TW "Sessions that start after this list was taken are not ended.
                                   They are shown in the result so you can end them."]
div.live-buttons
  button.live-primary  plan "End {n} other sessions"      busy: [TW "Ending sessions…"], aria-busy, disabled
  button.live-secondary existing "Cancel"
```

- **The list is frozen.** On open, the dialog renders the h2 and the status line at once, then re-reads `GET /me` (plan §9). It fills the list from that read only. Later polls or reloads of the page behind never change the open dialog.
- **The handles sent are exactly the rows rendered**, with the read's `sessions_generation`. The primary is not rendered until the list is present.
- **Rows:**
  - Reference uses `span.live-mono > bdi[dir=ltr]` with the 16-hex handle.
  - Signed in and Ends by use the existing `when()`.
  - The rows have no row actions and no checkboxes, so nothing can be deselected. The spec asks for 선택 수 and current-session inclusion to be confirmed (`:72`), not for per-row choice.
- **Order** is the read's order, which is newest signed-in first if the server already sorts that way. Root confirms the order. Rows are not re-sorted on the client.
- **Read outcomes:**
  - **Zero other sessions:** the lead, list and primary are replaced by the existing "You have no other sessions." sentence, and the secondary becomes [TW "Close"].
  - **Read failed:** `.service-summary` shows the existing `loadFailed` title and `loadRetry` body, then `button.live-secondary` "Try again" (existing `c.tryAgain`) and Cancel.
  - **Read answered 401:** the dialog closes and the existing signed-out view is shown.
- **Size:** at most 9 other sessions under the module limit of 10, and the contract allows up to 64. The table does not scroll on its own; the dialog scrolls. At 9 rows the content is about 640px high at 1440 (estimated, not measured).

### 3.3 Variant "one": entered from a row's End or a held session's End (§5.4)

This is the same shell and keeps the current content, so the single-session flow is consistent with the others flow:
- h2: existing endOneTitle "End this session?";
- `dl.live-facts`: Reference · Signed in · Ends by (existing);
- `p.mysec-end-lead`: existing endOneImpact;
- buttons: existing "End session" and "Cancel".

No list-time line is shown, because the handle alone fixes the target (plan §2).

### 3.4 Step-up (HYB-293)

- After the primary is pressed, a 403 `reauthentication_required` opens `dialog.service-reauth` above `dialog.mysec-end` through the shared step-up client. The primary stays busy.
- On success the client resends the same frozen request once. On cancel, `.service-summary` shows the existing `c.reauthText` copy, the primary is restored and focus moves to the summary.
- **422 `prerequisite_failed`** (an MFA rule): the summary shows the existing HYB-293 mapping and the primary is removed.
- Stacking two top-layer modals is **UNVERIFIED**. It is a rendered-QA item (§11).

### 3.5 Outcomes of the POST

| Answer | Presentation | Focus |
| --- | --- | --- |
| 202 (`replay` false) | Close the dialog. Render the operation from the answer as the newest one in §5, then reload `GET /me` and start polling if `cleanup_running`. | The operation's h4 (§9) |
| 202 `replay: true` | As above, plus the `p.live-sub` line under the operation heading: [TW "This is the result of your earlier request. Nothing new was ended."] | The operation's h4 |
| 401 `unauthenticated` | Close the dialog and show the existing signed-out view (spec `:88`; see §13 P-7). | The existing autofocused "Sign in again" |
| 422 `invalid_request`, kind `others` | Close the dialog, reload `GET /me` and show the list-changed notice (§8.2). Never a generic error (plan F7). | `p.mysec-changed` |
| 422 `invalid_request`, kind `one` | Close the dialog, reload and show the existing `endGone` copy in `p.mysec-changed`. | `p.mysec-changed` |
| 429 `rate_limited` | Show the existing `endLimited` copy in the summary. The primary stays. | The summary |
| Network failure, abort, 5xx or any other code | Show the plan's unknown-result copy (spec `:87`, KO verbatim) in the summary. Remove the primary and change the secondary to [TW "Reload and close"]. It closes the dialog, reloads, and shows the newest operation if there is one. Never resubmit automatically (plan §6). | The summary, then the operation h4 or `#mysec-sessions-title` after reload |
| Cancel or Escape | Close. | The invoking button. If it no longer exists, `#mysec-sessions-title` |

---

## 4. Entry point and sessions table

### 4.1 "End all other sessions"

- It stays `button.live-secondary.mysec-end-others` in `.live-tools`, before Reload. It is rendered only when at least one other session exists (existing rule).
- It is content width and 44px high. At ≤767px the existing `.live-tools` stacking rules apply.
- It is secondary, not primary. The page has no primary action, and the gateway to an access-ending action should not be urged visually. The dialog carries the only primary.
- The label "End all other sessions" is kept. See [TW] P-6: under the hold model "all" means all at list time.

### 4.2 Table

The table is unchanged (`mysec-col-a…d`, 36 / 24 / 22 / 18%).

### 4.3 Held tag in the table

For each row whose handle is in the `not_included[]` of the newest `kind: "others"` operation that is still present:
- After "Other session" in `td.live-user`, render `span.mysec-tag` [TW, glossary `session-not-included` "Started after your list"].
- `.mysec-tag`: `display:inline-block; margin-inline-start:8px; padding:0 6px; border:1px solid var(--border-control); font-size:14px; line-height:22px; font-weight:600; color:var(--hb-ink); white-space:nowrap`. This is the HYB-299 `m01-tag` pattern: a neutral border and no fill.
- At ≤767px the tag wraps under the label (`display:block; inline-size:max-content; margin:4px 0 0`).
- The row has no background change. `--surface-hover` (`.live-selected`) is no longer used in this view because no panel step is open.

---

## 5. Result region: recent session ends (`section.mysec-ends`)

### 5.1 Structure

```
section.mysec-ends[aria-labelledby=mysec-ends-title]
  h3#mysec-ends-title (16/600)          [TW "Recent session ends"]
  p.live-meta (14, 68ch)                [TW "The latest 5 are kept until the service restarts."]
  ol.mysec-ops (list-style:none; padding:0; margin:12px 0 0)
    li.mysec-op  × up to 5, newest first   (border-block-start:1px solid var(--border-row); padding-block:16px)
      details.mysec-op-wrap [open per §5.6]
        summary.mysec-op-summary (min-block-size:44px)
          h4#mysec-op-{operation} (16/600, tabindex=-1, display:inline)  kind others: [TW "Other sessions ended"];
                                                                         kind one: [TW "One session ended"]
          span.live-meta  "· {Access blocked time} · {cleanup state word}"   (one line, wraps)
        div.mysec-op-body
          p.live-sub  operation reference  (mono, bdi ltr "session-end-7"); replay line (§3.5) when replay
          div.mysec-op-state  (§5.2)
          dl.mysec-op-lines   (§5.3)          ← the two lines
          section.mysec-held  (§5.4)          kind others only
          p.mysec-gone        (§5.5)
          section.mysec-op-sessions (§5.7)
          p.mysec-checked + Check again (§7)
```

- Region placement: directly under the toolbar and above the pre-launch note and the table. The result is what the person acted for, and focus lands there.
- **Name.** The region is not "access history", which is HYB-306's term. Its heading must not suggest a complete log.

### 5.2 State sentence (`div.mysec-op-state`, 16px, 68ch, each sentence its own `p`)

| `state` | Paragraphs | Weight |
| --- | --- | --- |
| `cleanup_running` | Spec `:85` (plan EN/AR; KO verbatim "접근 차단은 적용되었지만 일부 세션 정리가 진행 중입니다.") | 400 |
| `complete`, `count > 0` | Plan "Access is blocked for {n} sessions and cleanup is complete." + plan "Pages already shown in those browsers and requests already in progress cannot be recalled." as a second `p` | 400; the second `p` 600 |
| `complete`, `count = 0` | [TW "No session needed blocking: every session in your list had already ended."] (P-2) | 400 |
| `cleanup_incomplete` | Plan "Access is blocked for all {n} sessions; none of them can be used." + plan "Cleanup of {k} could not finish." as a second `p` | first 400, second 600 |

[R-2] concerns whether spec `:85` is also shown in `cleanup_incomplete`.

### 5.3 The two lines (`dl.mysec-op-lines`)

```
dl.mysec-op-lines   max-inline-size:72ch; margin:16px 0 0
  div  (display:grid; grid-template-columns:minmax(10rem,max-content) minmax(0,1fr); column-gap:24px;
        padding-block:12px; border-block-start:1px solid var(--border-row))  — last div also border-block-end
    dt (16/600)  [TW glossary session-access-blocked "Access blocked"]
    dd (16)      "{time}" (blocked_at, when()) + " · " + [TW "{n} sessions"] (existing sessionsWord)
  div
    dt (16/600)  [TW glossary session-cleanup "Cleanup"]
    dd           state word: running [TW "In progress"] (400) · complete [TW "Complete"] (400) ·
                 incomplete [TW "Not finished"] (600)
                 then dl.mysec-counts (below)
```

**`dl.mysec-counts`** follows the HYB-299 §6.1 counts strip:
- `display:flex; flex-wrap:wrap; gap:4px 24px; margin-block-start:8px`. Each `div` has `min-inline-size:9ch`.
- dt is 14px muted; dd is 16/600 `tabular-nums`.
- **Items:** [TW "Sessions"] {count} · [TW "Done"] {done} · [TW "Already cleared"] {already_ended} · [TW "In progress"] {pending} · [TW "Not finished"] {failed}.
- All five are always shown, and a real 0 reads "0" (spec `:91`).
- **Under the strip,** `p.mysec-equation` (16px, 68ch) gives the plan line as a whole per-locale sentence that includes the already-cleared term (P-1). For example: [TW "{count} sessions: {done} done, {a} already cleared, {pending} in progress, {failed} not finished."]
- No bar, no percentage and no ratio glyph.
- **When `count = 0`,** the Cleanup dd reads [TW "Nothing to clean up"], with no strip.
- At list container <520px (390), dt sits above dd (`grid-template-columns:minmax(0,1fr)`, row-gap 4px).

### 5.4 Held sessions (`section.mysec-held`, kind `others` only)

The block is rendered when `not_included[]` is non-empty.

```
section.mysec-held[aria-labelledby]   margin-block-start:16px
  h5 (16/600)  [TW glossary session-not-included "Started after your list ({k})"]
  p (68ch)     plan Held copy; first sentence 600 ("Only the sessions in your list were ended."),
               the rest 400 in the same paragraph per locale
  button.service-button-link   [TW "Go to Change password"] → onView("methods"), 44px hit area
  ul.mysec-held-list (list-style:none; padding:0)
    li  (display:flex; flex-wrap:wrap; align-items:center; gap:8px 24px; padding-block:8px;
         border-block-start:1px solid var(--border-row))
      span  reference (mono, bdi ltr)
      span.live-sub  [TW "Signed in {time}"] (existing signedInAt)
      still present in the latest read → button.live-secondary.live-revoke "End" (existing m.end)
                                          + visually hidden " reference {handle}"; data-session={handle}
      no longer present → span (14) [TW "No longer signed in"]  — no button
```

- **Generation changed, nothing to list.** When `generation_changed` is true and `not_included[]` is empty, render only `p.live-sub` [TW "Your sessions changed after the list was taken."] in place of the block.
- **End** opens the "one" variant (§3.3). Cancel returns focus to that End button.
- "No longer signed in" is derived from absence in the current read. It does not claim that the session was ended by you or that it expired.

### 5.5 Already gone (`p.mysec-gone`, 16px, 68ch, 400)

Shown when at least one confirmed session has `result: already_ended`. It uses the plan copy "{j} sessions in your list had already ended." Those sessions also appear in §5.7.

### 5.6 Which operations are open

A `details` is `open` on first render when any of these is true:
- it is the newest operation;
- its state is not `complete`;
- it has a held session that is still present.

The others are collapsed to the summary line. A user toggle is kept across polls; key the open state by `operation`.

### 5.7 Per-session results (`section.mysec-op-sessions`)

```
h5 (16/600)  [TW "Sessions in this request ({m})"]
ol.mysec-op-list (list-style:none; padding:0)
  li.mysec-op-row (border-block-start:1px solid var(--border-row); padding-block:12px;
                   container: grid minmax(12rem,34%) minmax(0,1fr), gap 4px 24px at list container ≥520px;
                   one column below)
    div  reference (mono, bdi ltr, 16) ; span.live-sub "Signed in {time}"
    div  result line (16): ended → per-session cleanup word from C3:
              done / already_ended → [TW "Access blocked · cleanup complete"]
              pending              → [TW "Access blocked · cleanup in progress"]
              failed               → [TW "Access blocked · cleanup not finished"] (600)
         already_ended (session) → [TW "Had already ended"] (400)
         details.mysec-items  summary [TW "Cleanup details for {ref}"] (44px; ref in bdi)
```

**`details.mysec-items` body:** groups in this fixed order. A group is rendered only when it has items, except "Outside the controlled scope", which is always rendered for ended sessions (plan F10).

| Group h6 (14/600) [TW] | Item states | Row |
| --- | --- | --- |
| Done | `done`, C3 `already_ended` | item name (16) + reason line (14 muted) when present; C3 `already_ended` reason [TW "The sign-in service had already closed it."] |
| In progress | `pending` | item name + [TW "Retrying"] |
| Not finished | `failed` | item name (600) + reason: `unexpected_error` [TW "Stopped after an unexpected error. Access stays blocked."], `retry_budget_spent` [TW "Stopped after 4 attempts. Access stays blocked."] |
| Not applicable | `not_applicable` | C8 + plan reason |
| Outside the controlled scope (glossary `outside-controlled-scope`; KO 통제 범위 밖, AR خارج النطاق المتحكَّم فيه) | `outside_scope` | C6, C7 with plan reasons; the C7 row is 600 |

- Each group is `ul` (list-style none, `padding:0`, rows `padding-block:6px`). Prose is capped at 68ch.
- **Item names** are all [TW] drafts:
  - C1 "Access through this session";
  - C2 "Recent password confirmation for this session";
  - C3 "Session record in the sign-in service";
  - C4 "Secret pickups issued to this session";
  - C5 "Wrong-password count for this session";
  - C6 "Requests already in progress";
  - C7 "Pages already shown in that browser";
  - C8 "Jobs and API keys".
- **A session that had already ended** shows [TW "Nothing to clean up: this session had already ended."] in place of the groups.

---

## 6. Single-session End: consistency

- A single End now produces an operation (`kind: "one"`), which renders in the same §5 region with the same two lines, counts strip and per-session row.
- Nothing announces "Ended N" any more. The old `ended` and `endAllImpact` strings are removed (plan §9).
- **Kind `one` never renders the held block or the already-gone line,** even if the server returns `not_included` for it (P-3).
- The table row's End and a held session's End open the same dialog variant.

---

## 7. Polling and the "checked" line

- **While the newest open operation is `cleanup_running`,** poll `GET /me` every 2 s for at most 30 s from the 202, which is at most 15 reads. Use the existing `sequence()` ordering guard. Stop polling when:
  - every shown operation is final;
  - 30 s have passed;
  - a read fails or answers 401;
  - the view changes or the component unmounts.
- **`p.mysec-checked`** (14px muted) sits at the end of the operation body: [TW "Checked at {time}"], from the read's `now`. It updates on each poll and is not a live region.
- **After 30 s still running:**
  - the line becomes [TW "Cleanup was still in progress at {time}."];
  - `button.live-secondary` [TW "Check again"] (content width, 44px) follows it. It does one read and restarts the 30 s window if the operation is still running.
  - There is no countdown and no animation.
- **A poll fails (not 401):**
  - polling stops;
  - the line reads [TW "Could not check cleanup. Last checked at {time}."], followed by "Check again";
  - the page-level `.live-error` is **not** shown, and the page and the result stay as they are.
- **A poll answers 401:** show the existing signed-out view.
- **Each poll re-renders the table and the region with stable keys** (handle, operation). If a poll removes the element that has focus (for example a held End button whose session ended elsewhere), move focus to that operation's h4.

---

## 8. Other states

### 8.1 Page-level (unchanged)

The following are unchanged: Loading (existing `role=status` sentence), Load error (existing `.live-error`), Signed out or locked out (existing `.live-unavailable` view), and the reauthentication dialog (spec `:89`, HYB-293 copy).

### 8.2 List changed after a 422 (`p.service-status.mysec-changed`, tabindex=-1)

- **Kind `others`:** [TW "Your sessions changed, so nothing was ended. The list has been reloaded. Check it and try again."]
- **Kind `one`:** existing `endGone`.
- The notice is cleared on the next Reload, the next dialog open or a view change.

### 8.3 Signed out by your own action elsewhere

This applies when another browser ends this session. The next request or poll answers 401, and the existing signed-out view appears (spec `:88`; see P-7).

---

## 9. Focus, keyboard and announcements

### 9.1 Tab order (Sessions view, wide)

1. Toolbar: End all other sessions, then Reload.
2. The changed notice (programmatic focus only).
3. The ends region, for each operation:
   - the summary;
   - the Go to Change password link;
   - the held End buttons;
   - the cleanup-details summaries;
   - Check again.
4. The sessions table: Sign out, then each row's End.
5. The panel (limits; no controls).

### 9.2 Focus moves

| Trigger | Target |
| --- | --- |
| Open either dialog | `h2#mysec-end-title` (it stays there after the list loads) |
| Cancel or Escape | The invoker (`.mysec-end-others`, or `[data-session=…]` in the table or the held list), else `#mysec-sessions-title` |
| 202 or replay | `h4#mysec-op-{operation}` (inside its `summary`; the `details` is forced open) |
| 422 | `p.mysec-changed` |
| Failure inside the dialog | `div.service-summary` |
| Reload and close | The newest operation's h4, else `#mysec-sessions-title` |
| Step-up cancelled | The dialog summary |
| Focused element removed by a poll | That operation's h4 |

- The dialog traps focus natively (`showModal`).
- Add `:focus:not(:focus-visible){outline:none}` only on `tabindex=-1` headings and notes, never on controls.
- Never override the global `--hb-focus` ring.

### 9.3 Announcements

All announcements go through the single polite region (`say()`); the visible lines are not live regions.

| Event | Announced once |
| --- | --- |
| 202 | The §5.2 state sentence. When `not_included` is non-empty, add [TW "{k} sessions started after your list and are still signed in."] |
| Replay | The §3.5 replay line, then the state sentence |
| State change found by a poll (running → complete or incomplete) | The new state sentence |
| 422 | The §8.2 copy |
| Polling stopped at 30 s | The "still in progress" line |

Do not announce on every poll, and never re-announce rows.

### 9.4 Dialog reading order

The order is h2, then the lead, then the scope sentence, then the list heading and its time, then the table, then the note. The table caption repeats the heading so the list is announced with its count (spec `:101`, "대상 이름과 영향 읽기").

---

## 10. Responsive and RTL

| Width | Region | Dialog |
| --- | --- | --- |
| 1440 (list 707) | §5.7 rows two columns (240 / 443); counts strip on one line in EN | 640px; table cols 237 / 178 / 178 (content 592) |
| 1280 (list 568) | rows two columns (193 / 351); counts strip wraps to two lines in KO and AR (allowed) | 640px |
| 1024 (list 960) | rows two columns (326 / 610); limits below the list | 640px |
| 390 (list 358) | rows one column; dt above dd in §5.3; held rows wrap the End under the reference; the table stacks through the existing `.live-table` ≤767px rules with `data-label` | full screen; manifest table stacked through `.live-table` rules with `data-label` on Signed in and Ends by; buttons 100% stacked |

- Use logical properties only.
- **AR:**
  - the dialog, lists and grids mirror;
  - the held tag sits at inline-end of "Other session" (left);
  - handles, the operation reference and times use `bdi dir=ltr` (handles and references in mono);
  - line-height stays at the console's 1.8;
  - plurals use the existing `plural()` forms (`otherSessions`, `sessionsWord`), and the button text needs AR plural forms (P-11);
  - the deny term is حجب (`حجب الوصول`, glossary `:69`), confirmed by TW.
- **KO:** `word-break: keep-all` on headings, the state sentences and the dt labels. The counts dt labels must not break inside a word.
- No → ← ✓ ✗ glyphs and no italics. Count sentences are whole per-locale templates. Separators use " · ", with the middle dot outside isolates.

---

## 11. Rendered-QA checklist (measurable)

For every render, record the viewport, the locale, the state, and the measured widths of `.live-users`, `.live-list`, `aside.live-panel`, `.mysec-ends` and `dialog.mysec-end`.

**Layout**
1. `.live-users` is about 1120 / 960 / 960 / 358 at 1440 / 1280 / 1024 / 390, and the list column is about 707 / 568 / 960 / 358.
2. `document.scrollingElement.scrollWidth <= clientWidth` holds on every render, including the open dialog and the expanded cleanup details. No `.mysec-*` element overflows except a table scroll wrapper.
3. The dialog `inline-size` is ≤640px at ≥768. At 390 it is 390×viewport height, with non-sticky buttons stacked at 100%.
4. Buttons are 44px high and content width at ≥768; no button is wider than its label plus 40px. "End all other sessions" is not stretched.
5. The computed `max-inline-size` of the prose is ≤68ch and of `dl.mysec-op-lines` ≤72ch.
6. The panel shows Session limits while the dialog is open and after the result. No end step renders in the panel.

**Truthfulness**
7. Before a 202, neither the page nor the dialog contains the words for blocked, ended, complete or done about this request in any locale. The exceptions are the dialog title and the button verbs.
8. "Complete" or "cleanup is complete" appears only when `state = complete`. Whenever it appears, the cannot-be-recalled sentence and the "Outside the controlled scope" group (C6, C7) are present in the DOM.
9. The words recall, recalled, revoked everywhere and signed out everywhere (and their KO and AR equivalents) never appear in the end-sessions UI. The only exception is the plan's held sentence, which recommends a password change.
10. **Counts:** Sessions = Done + Already cleared + In progress + Not finished, on screen. Held and already-gone sessions are outside the strip.
11. There is no `<progress>`, `%`, spinner animation (`animation-name` is none in `.mysec-*`) or countdown.

**Hold scenario (with plan Q2)**
12. Browser B signs in after A opened the dialog. A's dialog list does not change. After confirmation:
    - A shows the held block listing B with an End button;
    - the B row in the table has the "Started after your list" tag;
    - ending B from the held block opens the "one" dialog and produces a `kind: one` operation with no held block;
    - the earlier B row then reads "No longer signed in".

**States**
13. `cleanup_running` (stubbed) shows spec `:85` with KO verbatim and the "Checked at" line. The line updates without moving focus. Exactly one announcement is made at 202 and one at the state change.
14. After 30 s still running, "Check again" appears and no further requests are made; this is verified in the network log.
15. `cleanup_incomplete` shows "Not finished" at 600 and the failed reason in the details.
16. A stubbed abort shows the `:87` copy with KO verbatim. The primary is removed, "Reload and close" is present, and there is no automatic POST.
17. A stubbed 422 on others closes the dialog, reloads, and focuses `p.mysec-changed`.
18. Replay shows the replay line.

**Styling sweep over `.mysec-*` and `dialog.mysec-end *`**
19. Checks:
    - border colours are only `--border-row`, `--border-control` or the global dialog border;
    - no `border-inline-start` or `border-inline-end` is wider than 1px, and none differs from the block borders;
    - `border-radius` is 0 and there is no `box-shadow`;
    - backgrounds are only `--surface-page` or `--surface-primary` (the primary button);
    - `.service-status` and `.service-summary` show no edge strip.

**Focus, RTL and KO**
20. Every move in §9.2 lands as listed, with a visible ring after keyboard moves. Keyboard alone can open the dialog, read the list, confirm, complete step-up and reach the operation heading.
21. The step-up dialog stacks above `dialog.mysec-end`. Escape closes only the top modal, and focus returns into `dialog.mysec-end`. This is UNVERIFIED until rendered.
22. **AR:**
    - mirrored layout;
    - handles, references and times read left to right;
    - the tag position is mirrored;
    - line-height 1.8;
    - plural forms are correct for 1, 2, 3 and 11 sessions.
23. **KO:** no mid-word breaks in the dt labels, the counts or the state sentences.
24. **Not covered:** owner acceptance, terminology correctness, native-speaker review, and screen-reader behaviour beyond the live-region and reading-order checks.

---

## 12. External references (not opened)

I opened no external source. Before implementation, root must open, record (source, observed pattern, adopted, rejected) and verify these candidates under `service-screen-style.md`:
- Google Account "Your devices" (per-device sign-out);
- GitHub Settings → Sessions;
- Microsoft account "Sign out everywhere";
- the GOV.UK Design System Summary list, Details and Notification banner components.

**What this spec assumes from them is inference only:**
- a per-session list with an explicit marker for the current session;
- details disclosures for secondary evidence;
- the absence of success-coloured banners.

Any pattern that the opened sources contradict must be reported back for revision.

---

## 13. Problems and decisions for root

- **[R-1] Modal confirmation for both variants (recommended).**
  - It meets spec `:101-102` and keeps the result in one place at every width.
  - The fallback is to keep the inline panel and step for both. The frozen list then sits in the 360–381px panel, and the spec's focus-trap requirement stays unmet. Record that if chosen.
  - This changes the existing single-End interaction, which the qa271 and HYB-293 runners drive.
- **[R-2] Spec `:85` in `cleanup_incomplete`.**
  - The plan uses it as the base sentence. It says cleanup is "진행 중" (in progress), which is untrue once cleanup has stopped.
  - I recommend showing `:85` only for `cleanup_running`; the incomplete copy's first sentence already states the blocking.
  - If root keeps the plan, render `:85` first and then the incomplete copy.
- **P-1.** The plan's cleanup line "{done} of {count} done, {pending} in progress, {failed} not finished" omits `already_ended`, so on screen the numbers do not add up to `count`. §5.3 adds "Already cleared" [TW].
- **P-2.** There is no copy for `complete` with `count = 0`. "Access is blocked for 0 sessions" would be misleading. §5.2 adds [TW] text.
- **P-3.** For `kind: one`, every other session would satisfy the plan's `not_included` definition. The console suppresses the held block for `one`. The server should return an empty array, which is for the security reviewer to confirm.
- **P-4.** The plan's allowed files contain no CSS file, and `users/live.css` is shared and not listed. The `mysec-*` rules need a scoped stylesheet, for example `my-security/end.css`. That file must be added to the allowed files.
- **P-5.** C3 `already_ended` (an item state) and the session result `already_ended` mean different things. The display words must differ: "Already cleared" for the item and "Had already ended" for the session [TW].
- **P-6 [TW].** The entry label "End all other sessions" contains "all" while the hold model ends only the listed sessions. Keep it with the dialog's explanation, or change it to "End other sessions".
- **P-7 [TW].** The plan keeps the existing signed-out body for the revoked state. The KO spec `:88` text is "현재 세션의 접근이 종료되었습니다. 다시 로그인해 주세요." The existing KO is "이 세션이 종료되었습니다. 계속하려면 다시 로그인하세요." The brief asks for KO verbatim where the spec has it. Changing it affects every signed-out path on M05.
- **P-8.** The plan's unknown-result EN differs from the existing `unknownResult`. Use the plan string for this flow (KO verbatim `:87`).
- **P-9 [TW].** Whether "List taken at {time}" shows seconds. Two lists taken within the same minute are otherwise indistinguishable.
- **P-10.** The order of manifest rows comes from the server. Confirm that it is stable (signed-in descending).
- **P-11 [TW].** The AR button "End {n} other sessions" needs plural forms. The plan draft "إنهاء {n} من الجلسات الأخرى" is wrong for 1 and 2.
- **P-12.** The "Recent session ends" region overlaps conceptually with HYB-306's recent access history. Keep the names distinct.

---

## 14. Tokens, type, classes and proposed item IDs

**Tokens (existing only):** `--hb-ink`, `--text-muted`, `--border-row`, `--border-control`, `--surface-page`, `--surface-primary`, `--text-on-primary`, `--hb-focus`, `--hb-font-mono`, and `--hb-danger` only through the existing `.service-summary h2`. No new tokens, literal colours, radii or shadows.

**Type:**
- h2 20/600; h3, h4 and h5 16/600; h6 14/600;
- body 16; meta and sub-lines 14; mono 14 in tables and 16 in the dialog reference cell;
- `tabular-nums` on counts.

**Spacing:** 4 / 8 / 12 / 16 / 24 / 32.

**Classes**, scoped under `[data-security-screen="M05"]` and `dialog.mysec-end`. Reuse the existing `live-*` and `service-*` classes.
- Dialog: `mysec-end`, `mysec-end-lead`, `mysec-end-scope`, `mysec-end-note`, `mysec-manifest-head`, `mysec-manifest`.
- Table: `mysec-tag`.
- Region: `mysec-changed`, `mysec-ends`, `mysec-ops`, `mysec-op`, `mysec-op-wrap`, `mysec-op-summary`, `mysec-op-body`, `mysec-op-state`, `mysec-op-lines`, `mysec-counts`, `mysec-equation`, `mysec-held`, `mysec-held-list`, `mysec-gone`, `mysec-op-sessions`, `mysec-op-list`, `mysec-op-row`, `mysec-items`, `mysec-checked`.

**Proposed implementation item IDs** (`screen-review.md`). Root must reconcile them with the existing M05 IDs and must not renumber those.
- Entry and table: `M05:sessions:end-others-entry`, `M05:sessions:held-tag`.
- Dialog: `M05:dialog:end-others`, `M05:dialog:end-one`, `M05:dialog:manifest-list`, `M05:dialog:list-time`, `M05:dialog:current-excluded`.
- Result: `M05:result:ends-region`, `M05:result:state-sentence`, `M05:result:access-blocked-line`, `M05:result:cleanup-line`, `M05:result:cleanup-counts`, `M05:result:held-sessions`, `M05:result:already-ended`, `M05:result:session-rows`, `M05:result:cleanup-items`, `M05:result:polling`.
- States: `M05:state:list-changed`, `M05:state:end-unknown`, `M05:state:end-replay`.

## Open uncertainties

- [R-1], [R-2] and P-1 to P-12.
- All [TW] strings, and the KO and AR drafts (plan R6).
- No current M05 renders were supplied or viewed.
- Widths are inferred from source.
- External references were not opened.
- The stacked modals are unverified.
- Nothing is render-verified.

---

## Files referenced (absolute)

- /private/tmp/hb307/docs/features/HYB-307/plan.md
- /Users/home/project/hybrid_billing/docs/ui/screens/M05/spec.md
- /private/tmp/hb307/apps/console/src/modules/identity/pages/my-security/Live.tsx
- /private/tmp/hb307/apps/console/src/modules/identity/pages/my-security/live-copy.ts
- /private/tmp/hb307/apps/console/src/modules/identity/pages/my-security/live-model.ts
- /private/tmp/hb307/apps/console/src/modules/identity/pages/users/live.css
- /private/tmp/hb307/apps/console/src/service/service.css
- /private/tmp/hb307/apps/console/src/service/Reauthenticate.tsx
- /private/tmp/hb307/docs/features/HYB-299/visual-spec.md
- /private/tmp/hb307/docs/features/HYB-298/plan.md
- /private/tmp/hb307/docs/features/HYB-293/plan.md
- /private/tmp/hb307/docs/ui/terminology/product-glossary.md
- /private/tmp/hb307/docs/ui/screen-binding-fields/identity.md
- /Users/home/project/hybrid_billing/.agents/rules/service-screen-style.md
- /Users/home/project/hybrid_billing/.agents/rules/design-system.md
- /Users/home/project/hybrid_billing/.agents/rules/screen-review.md
- /Users/home/project/hybrid_billing/.agents/rules/usability.md

## 15. External references opened by root (2026-11-19)

| Source | Observed pattern | Adopted | Rejected or not established |
| --- | --- | --- | --- |
| GitHub Docs, "Viewing and managing your sessions" (docs.github.com/en/authentication/keeping-your-account-and-data-secure/viewing-and-managing-your-sessions) | Sessions listed under Settings > Sessions, grouped web and mobile; per-session "See more" details; revocation is per session ("Revoke session"); revoking a mobile session also removes it as a second factor | Per-session End actions kept for the table and for held sessions; per-session details behind a disclosure | The page shows no "end all" action, no current-session marker and no confirmation or result copy, so none of those is claimed from it |
| GOV.UK Design System, "Summary list" (design-system.service.gov.uk/components/summary-list/) | Key and value rows on a `dl`; row actions with visually hidden text naming the target; row borders help connect a row with its action; use a table for tabular data | `dl` key/value lines for "Access blocked" and "Cleanup"; visually hidden " reference {handle}" on End buttons; neutral row hairlines kept; the manifest stays a table | Summary cards are not used |

Google Account "Your devices" and Microsoft "Sign out everywhere" were not opened; nothing is adopted from them.

## 16. Root decisions (2026-11-19)

Binding over the matching proposals.

1. **R-1:** accepted — both end confirmations use the modal `dialog.mysec-end`. The qa271 and HYB-293 runners are updated as regressions.
2. **R-2:** accepted — spec `:85` only for `cleanup_running`; `cleanup_incomplete` uses its own copy.
3. **Naming after plan review round 2 (N3):** the per-session result is `already_gone` (display "Had already ended" [TW]); `already_ended` is only the C3 cleanup state (display "Already cleared" [TW]). Every `result: already_ended` for a session in this spec reads `already_gone`.
4. **P-1, P-2, P-5:** accepted as specified; the cleanup line includes the already-cleared term and the count = 0 state has its own copy.
5. **P-3:** the server returns an empty `not_included` for `kind: one`; the console also suppresses the held block for `one`.
6. **P-4:** a scoped stylesheet `apps/console/src/modules/identity/pages/my-security/end.css` is added to the allowed files; shared `users/live.css` is not edited.
7. **P-6:** the entry label becomes "End other sessions" [TW]; "all" is dropped because the hold model ends the listed sessions only.
8. **P-7:** the M05 revoked body uses the spec `:88` KO wording verbatim; EN/AR follow [TW]. Every signed-out path on M05 is re-checked in QA.
9. **P-8:** the plan's unknown-result string is used for this flow (KO verbatim `:87`).
10. **P-9:** "List taken at {time}" shows seconds.
11. **P-10:** the server orders the manifest and session list by signed-in time, newest first, then handle; tested.
12. **P-11, P-12, all [TW] strings:** for the technical-writing review before browser QA; the region heading stays "Recent session ends", distinct from HYB-306's access history.
13. **Render review round 1 (amends §5.3 and §11.10):** the counts strip drops the "Sessions" item (the count is already in the Access blocked line and in the equation sentence) and uses a fixed grid of 4 columns at a dd width of 380px or more, else 2×2, so it never wraps 4+1. §11.10 becomes: Done + Already cleared + In progress + Not finished = the count in the Access blocked line. The manifest Reference column is 12.5rem and dates render as nowrap date and time parts. `dl.mysec-op-lines` has no measure cap. The held tag is always on its own line. "Change your password" is a navigation link, not a button.
