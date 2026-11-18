---
type: DesignSpecification
title: HYB-297 M02 trust change workflow visual specification
description: Layout source for the M02 policies view in service mode after the trust change workflow lands. Written by the ui-designer role (read-only). The HYB-297 plan wins on copy, states and truthfulness; this spec wins on layout.
status: stable
tags: [HYB-297, M02, trust-change, visual-spec]
sources:
  - resource: plan.md
  - resource: ../../ui/screens/M02/spec.md
  - resource: ../../ui/screens/M02/design.json
---

# HYB-297 visual specification: M02 trust change workflow (service mode, policies view)

**Author:** ui-designer role, read-only. The environment reports the model as Claude Opus 5.5 (`claude-opus-5-5`).

**Status:** this is a design proposal. It is not owner acceptance, and a later visual PASS would not be owner acceptance either.

**Root decisions on §11 (2026-11-18):**
- Item 1: the propose form opens from an opener; it is not always open.
- Item 2: states are shown as text only; no state icons.
- Item 4: the dialog secondary button is "Go back".
- Item 5: after activation, show only the counts from the response; the full invalidation table appears after sign-in.
- Item 6: the expected new version is shown labelled "expected".
- Item 7: the Arabic rationale uses `span[dir=auto]`.
- Items 3 and 8 go to the technical-writing review and rendered QA.

**Precedence:** where this spec and `docs/features/HYB-297/plan.md` (approved after design review round 5) differ on copy, states or truthfulness, the plan wins. Copy marked **[TW]** below is a new string the layout needs. The plan does not define these strings, so the technical-writing review must supply or approve them. I am not supplying final copy.

---

## 0. Evidence basis and limits

**Read directly:**
- docs/features/HYB-297/plan.md, the whole file, including the Console design section, C1–C6, the state copy table, the hold table and copy, and the enrolment and isolation copy.
- docs/ui/screens/M02/spec.md, read in full. design.json was grepped only; its layout is `form` and its six `trust.*` actions match spec.md.
- Rules:
  - .agents/rules/service-screen-style.md
  - .agents/rules/design-system.md
  - .agents/rules/screen-review.md
  - .agents/rules/usability.md
  - docs/design-build-guide.md
  - docs/design-system.md (grepped for shape, icons, states and RTL)
- The current live M02 implementation:
  - apps/console/src/modules/identity/pages/authentication/Live.tsx
  - apps/console/src/modules/identity/pages/users/live.css (the shared `.live-*` base)
  - apps/console/src/app/screen-groups/identity.tsx:137-166 (wrapper `.remaining-security[data-security-screen="M02"]`)
  - apps/console/src/service/Reauthenticate.tsx and service.css:306-337 (the HYB-293 dialog)
  - apps/console/src/shared/styles.css:187-189 (global `bdi{direction:ltr;unicode-bidi:isolate}`) and :1033-1066 (global `dialog`, opaque backdrop, `.dialog-actions`)
  - apps/console/src/app/service-shell.css:34-85 (rail 256px, main padding 32px; only `.service-navigation` is sticky, the header is not)
  - packages/design-system/styles/tokens.css (`--hb-radius:0`, `--border-row`, `--border-control`, `--border-header`)
  - The icon set in packages/design-system/assets/icons, which holds **only** `chevron-down.svg` and `close.svg`
- Precedent:
  - docs/features/HYB-315/visual-spec.md and references.md
  - docs/features/HB-265/verification.md (the earlier M02 design-comparison findings: H1 comparison clipping, M1 unlabelled mobile history)

**Not available or not done:**
- **No current M02 screenshots were supplied and I viewed none.** The "current state" observations below come from source reading only. HB-265 renders are said to be in `the HB-265 QA renders (not in the repository)`; they were not assigned and I did not open them.
- **I had no browsing tool, so I opened no external reference.** §13 lists the plan's references as UNVERIFIED. Root must open them and record what they show before screen code, as was done for HYB-315.
- I did not read the frontend-design skill, because none was supplied in this task.
- I did not read live-copy.ts, live-model.ts, docs/ui/design-system/terminology.md or the glossary.
- I ran no commands, builds or browsers. Nothing here is render-verified.

**Container widths:** these are inferred from service-shell.css (1440 − 256 − 64 = 1120; 1280 − 256 − 64 = 960) and match the measured P04 values in the HYB-315 spec. At 390 I assume the same 358px (rail on top, 16px gutter) as P04. Neither has been measured on M02 (§12 check 1).

### Current-state observations (from source)

1. The policies view is `.live-users` with a wide split (`clamp(360px,34%,440px)` panel at viewport ≥1280, `useWide`). At wide widths the **propose form is permanently open** in the panel.
2. Current facts are a single-column label|value `dl.live-facts-grid` with nine rows.
3. Pending proposals are `article.live-proposal` blocks, each with an inline comparison table. History is a five-column table.
4. **Defect to carry forward and fix:** the rationale renders as `<bdi>{row.reason}</bdi>` (Live.tsx:352). The global `bdi{direction:ltr}` therefore forces Arabic rationales left-to-right.
5. Approve and withdraw are inline steps in the panel, not modal dialogs. The plan now requires modal dialogs (C1–C6).

---

## 1. Composition principles

1. **What is in force comes before what is proposed.** The reading spine is:
   1. conditions (hold, isolation and the like);
   2. current sign-in rules;
   3. trust changes, made up of the holders line, the list and the detail;
   4. actions, then confirmations.
2. **A workflow ledger, not a dashboard.** There are no progress bars, steppers with fills, percentages, ticks, crosses, green, amber or red. A state is a word plus its plan sentence. Counts are plain numbers.
3. **Only the server can make something read active, approved or activated.**
   - The client never infers a later state.
   - `activation_unknown` must never contain the word "Activated".
   - An expected digest or version computed in the browser is always labelled expected.
4. **Emphasis (weight 600, ink) is reserved for:**
   - problems: blocked items, stale evidence, the hold and isolation leads;
   - permanence sentences;
   - "everyone is signed out, including you".

   Nothing positive gets emphasis.
5. **Full viewport width with a capped measure.**
   - Regions span `.m02-live`.
   - Prose is capped at 68ch.
   - Inputs are capped at 440px (minutes inputs at `10ch + 26px`, as now).
   - Buttons are content width and 44px high at ≥768px viewport, and 100% width below that.
6. **Neutral geometry only.**
   - Separators are 1px `--border-row`. Controls use `--border-control`. The dialog border is the global `--hb-line-header`.
   - Radius 0, no shadows, no coloured edge strips, no coloured selection outlines.

---

## 2. Page structure (policies view, `data-security-view="policies"`)

```
.remaining-security[data-security-screen="M02"]        (existing wrapper; dir/lang set there)
  div.m02-live                                           container-type:inline-size; min-inline-size:0
    p.visually-hidden[role=status][aria-live=polite]     single announcer (async results only)
    section.m02-conditions        (only when a condition exists; §6)
    section.m02-current           Current sign-in rules (§3)
    section.m02-changes           Trust changes (§4, §5)
      .m02-changes-head           h2 + holders line / enrolment block + rights line + notes + opener
      div.m02-split[data-mode=split|step][data-open=true|false]
        div.m02-lists             h3 Open (n) + table; h3 Closed (n) + table
        aside.m02-detail          detail panel, or the propose/revise form (§5, §7)
```

- **Placement decision.** The trust workflow sits **below** the current rules and **replaces** today's "Pending proposals" and "History" sections. Activated changes appear in the Closed list, so the separate history table is removed. Current rules stay outside the split and use the full width, so what is in force is always visible above any proposal.
- **Split mode is decided by one source:** the existing `useWide()` (viewport ≥1280px, which gives `.m02-live` ≥960px). JS sets `data-mode`, and CSS keys only on that attribute, so the CSS and JS cannot disagree.
  - Split applies at 1440 and 1280. Step mode applies below 1280.
- **The other views** (`providers`, `test`, `keys`, `recovery`, `sync`) are unchanged.

### Vertical rhythm

- Sections inside `.m02-live` are separated by `gap: 40px` (the current `.live-sections` value).
- Inside a section, h2 → content is 12px, and sub-blocks are separated by 24px.
- There is no rule above the first block after a heading.

---

## 3. Current sign-in rules (`section.m02-current`)

```
.m02-toolbar (flex wrap; space-between; align-items:flex-end; gap 12px 16px; margin 0 0 16px)
  .live-heading: h2#m02-current "Current sign-in rules" (20/600, tabindex=-1) + p.live-meta "Loaded at …"
  .live-tools: button.live-secondary "Reload"
.m02-current-grid  (grid; 1 col; gap 24px 48px)
  @container (min-width: 880px) { grid-template-columns: minmax(0,1fr) minmax(0,1fr) }
  div.m02-group  h3 [TW "Rules"]  dl.m02-facts
       Sign-in methods · MFA · Maximum session age · Password confirmation (step-up) · Effective window
  div.m02-group  h3 [TW "Trust"]  dl.m02-facts
       Trust version (epoch) "g.n" · Policy revision · Origin · Activated at ·
       Proposed by · Validated by · Approved by · Activated by
p.m02-ledger      (14px, ink, 68ch): ledger state, records, generation [TW sentence template]
ul.m02-notes      (14px, muted, 68ch): origin notes (seed / replayed at start / carried from …)
p.live-window     (existing act_until / stale line, unchanged)
```

- **`dl.m02-facts > div`:**
  - `display:grid; grid-template-columns:minmax(140px,40%) minmax(0,1fr); gap:4px 16px; padding-block:10px; border-block-start:1px solid var(--border-row)`.
  - dt is 14px muted; dd is 16px ink.
  - Below a group width of 420px the row becomes one column, with dt above dd.
  - At 1120 and 960 the two groups sit side by side (about 536 and 456px each). At 358 they stack.
- **Trust version value:** `<bdi dir="ltr" class="live-mono m02-epoch">{generation}.{number}</bdi>`.
  - Build it from the two integer fields as strings.
  - **Never use `Intl`/`toLocaleString` and never parse the raw decimal string into a Number** (plan, Objects).
  - The raw decimal string appears only in the expert detail (§5.8).
- **Absent actor or time** (for example no validator on the seed genesis): show the muted word [TW "Not recorded"]. Never show "—" or a blank.
- **No policy yet:** keep the existing `p.live-empty` "no policy" sentence in place of the groups. The Trust group still shows version `g.0`.

---

## 4. Trust changes: header, holders, list

### 4.1 Header block (`.m02-changes-head`)

Order:
1. `.m02-toolbar`: h2#m02-changes [TW "Trust changes"] (20/600, tabindex −1), plus the opener `button.live-primary.m02-propose` [spec label "Propose trust change"] at inline-end.
   - Render the opener **only** when `rights.propose` is true and no hold, isolation or stopped ledger applies. Otherwise do not render it; never render it disabled.
2. `p.m02-holders` (16px, 68ch):
   - With qualifying_people ≥ 3: the plan line "N administrators qualify for trust changes; a change needs 3."
   - With fewer than 3: **the enrolment block** instead, `div.m02-enrolment`, which holds the plan copy as one paragraph with its first sentence wrapped in `<strong>` (600). There is no box, no border and no icon.
3. `p.m02-rights` (14px, ink) [TW]: the caller's trust rights in words, for example "Your trust rights: propose, validate". With none, use [TW "You hold no trust rights."]. This explains missing buttons before the user opens a change.
4. `p.m02-unavailable-line` (16px, 600 on the first sentence): appears when trust commands are refused by a hold, isolation or a stopped ledger. Its text is the `trust_unavailable` failure copy from the plan, shortened to its first sentence plus "See the notice at the top." [TW].
5. `p.m02-note` (14px muted, 68ch): the partial-state note "Changes from before the server started at … are not kept" (plan, Restart step 6), whenever replayed or earlier-run changes exist.

Spacing: 8px between items 2–5, and 24px before the split.

### 4.2 Lists (`div.m02-lists`, `container-type:inline-size`)

There are two groups, each `h3` (16/600, margin 24px 0 8px) plus one `table.m02-table`:
- **[TW "Open"] (n):** `activation_unknown` first, then `independently_approved`, `validated` and `draft`, newest first within each state.
- **[TW "Closed"] (n):** `activated`, `rejected`, `cancelled` and `superseded`, newest first.
- An empty group shows `p.live-empty` [TW "No open changes."]. A real 0 count reads "0".

**Full table** (no selection, list container ≥720px, so 1120 and 960). Settings: `table-layout:fixed`, a colgroup, a visually hidden caption, td padding-block 12px, `vertical-align:top`, th 14/600, and `border-block-start:1px solid var(--border-row)` per row.

| Col | Width | Content |
| --- | --- | --- |
| Change | 18% | `button.m02-open` "Change {n}" (600, underline, 44px hit area via padding-block 10px, `text-align:start`); under it `span.m02-sub` (14px muted) [TW "Earlier run, started {date time}"] when replayed |
| State | 22% | `span.m02-state-word` [TW short label per state, see §11]; weight 600 only for `activation_unknown` and for validation result `blocked`. Under it, 14px, "Evidence changed" [TW] (600) when `evidence_current:false` |
| Proposed by | 16% | `<bdi dir="ltr" class="live-mono">` proposer |
| Based on | 14% | Version `g.n` (bdi, mono) and 14px "Policy {r}" |
| Next step | 30% | The trust-model next-step sentence (16px). `why_not` reasons are in words |

- Px at 1120: 202 / 246 / 179 / 157 / 336. At 960: 173 / 211 / 154 / 134 / 288.

**Split mode with a selection** (`data-open="true"`; the list is about 435px at 1440 and about 371px at 1280):
- Render a **reduced, single-cell row**, chosen by state rather than hidden by CSS: line 1 "Change {n}" opener, line 2 the state word, line 3 the next step at 14px.
- Proposer and base are in the detail.
- No column headers are visible; the caption stays.

**Stacked** (list container <720px with no selection, which in practice is ≤767px viewport):
- One block per row, with the Change opener first.
- Then each other cell as `td[data-label]::before` (14px muted label on its own line; `content: attr(data-label)`; no colon concatenation in AR).

**Selected row:**
- `tr[aria-current="true"]` with `background:var(--surface-hover)`.
- A visible `span.m02-tag` [TW "Selected"] after the opener (`aria-hidden="true"`; `aria-current` carries the meaning).
- No outline other than transient focus.

`.m02-tag`: `display:inline-block; padding:0 6px; border:1px solid var(--border-control); font-size:14px; line-height:22px; font-weight:600; color:var(--hb-ink)`. Use it only for "Selected". State words are plain text, not tags.

---

## 5. Detail (`aside.m02-detail` in split mode; `section.m02-step` in step mode)

### 5.1 Split grid

```
.m02-split                     { display:grid; grid-template-columns:minmax(0,1fr); gap:32px; align-items:start; }
.m02-split[data-mode="split"][data-open="true"]
                               { grid-template-columns: minmax(320px,2fr) minmax(0,3fr); }
aside.m02-detail               { min-inline-size:0; container-type:inline-size;
                                 border-inline-start:1px solid var(--border-row); padding-inline-start:32px;
                                 position:sticky; inset-block-start:16px; max-block-size:calc(100dvh - 32px);
                                 overflow-y:auto; overscroll-behavior:contain; }
aside.m02-detail[tabindex="0"]:focus-visible { outline-offset:-2px; }
```

- **Widths:** the detail column is about 653px at 1440 (621px of content) and about 557px at 1280 (525px of content).
- **Why the detail is the wider pane.** This reverses P05. Here the list is short and bounded, while the detail carries:
  - the comparison;
  - nine validation items with reasons;
  - 64-hex digests;
  - impact, people, history and invalidation.
- **Without a selection** there is one column and no empty panel.
- **DOM order** is list first, then detail. In AR the panel lands on the left through the grid. No `order` and no floats.
- **Sticky offset:** 16px. The shell header is not sticky (service-shell.css), so no header offset is needed. Verify in render (§12 check 4).

### 5.2 Step mode (<1280 viewport, with a selection)

Render only the following; do not render the current rules, the changes header or the lists:
- `button.live-back.m02-back` [existing "Back" copy], 44px, underline;
- `p.m02-condition-short`, the first sentence (600) of each active condition from §6;
- the detail.

At 768–1279 the step content uses the full width (about 704–1120px). Its prose stays at 68ch and its `dl.m02-sum` turns two-column at a step container ≥720px.

### 5.3 Header and actions (top of the detail; spec: "primary actions under the title")

```
.m02-detail-head   flex wrap; space-between; gap 8px 16px
  h2.m02-detail-title (20/600, tabindex=-1): "Change {n}"  + span.m02-sub earlier-run line
  button.service-button-link.m02-close "Close" (split only; 44px; underline)
p.m02-state-line     the plan state sentence (16px, 68ch). Weight 600 for activation_unknown, superseded,
                     and for the "blocked" sentence when the last validation result is blocked.
p.m02-stale          (600, 68ch) plan copy "The evidence changed after validation. …" when evidence_current:false
p.m02-next           [TW "Next step:"] + next-step words (16px)
div.m02-actions      flex wrap; gap 12px; margin-block 16px 0
p.m02-why            one paragraph per why_not reason (16px ink, 68ch)
```

**Actions** come from the `can` flags. Each button opens a dialog. Revise opens the form in the panel or step. **At most one `live-primary`:**
- The primary is the first true flag in this order: `activate` → `approve` → `validate`.
- `revise`, `reject` and `cancel` are always `live-secondary`.
- Button labels are the spec action labels:
  - "Activate approved trust change"
  - "Independently approve trust change"
  - "Validate trust change"
  - "Reject trust change"
  - "Cancel trust proposal"
  - [TW "Revise"]
- Order: primary, then Revise, Validate (when secondary), Reject, Cancel.
- A command that is not allowed is **not rendered**. Its reason appears in `p.m02-why`; never use a disabled button.

### 5.4 Settings comparison (`section.m02-compare`, h3 [TW "Settings"])

- A table with columns Setting 36% | [existing "Current"] 32% | [existing "Proposed"] 32%.
- A changed value is weight 600 and followed by the existing "(changed)" word. Never use an arrow glyph.
- It stays a table at detail content ≥440px (both split widths and step M). Below 440px it is `dl.live-compare-stack`, as HB-265 H1 decided, so it is never clipped.
- Under isolation or a stopped ledger, add a 14px line under the table [TW "Current values may be older than the last activated change."]. This mirrors the plan's isolation sentence.

### 5.5 Validation evidence (`section.m02-validation`, h3 [TW "Validation"])

1. **Result line** `p.m02-result`:
   - [TW "Validated at {time} by {who}"], or [TW "Blocked at {time} by {who}: {k} checks blocked"] (600), or the muted [TW "Not validated"].
   - Never leave it blank.
2. The `p.m02-stale` repeat is not needed here; it is already in the header.
3. **`table.m02-findings`** (caption [TW "Checks"]):

| Col | Width | Content |
| --- | --- | --- |
| Check | 34% | Item name in words (Test login, Recovery path, Issuer and stable subject, Provider diagnostics, External identity status, Record size) |
| Result | 22% | "Verified" / "Not applicable" / "Blocked" as plain words. Only "Blocked" is 600. No tick or cross, no colour |
| Reason | 44% | Localized reason sentence. Under it, the server code in `<bdi dir="ltr" class="live-mono">` at 14px muted (for example `recovery_unverified`, `no_provider_connection`). For the recovery item, the missing role in words |

- Rows keep this fixed order so changes can be compared. A blocked row does not move to the top; the result line already says how many are blocked.
- Below 480px of the detail container, stack the rows: check name (600), then "Result: …" and the reason paragraph.
4. **Informational facts** (`dl.m02-sum`, not in the table, because they are not verified/blocked judgements):
   - MFA weakening ("Weakens MFA" 600, or "Unchanged");
   - affected members (count);
   - open sessions now (count, labelled live, [TW "now"]).

### 5.6 Impact (`section.m02-impact`, h3 [TW "Impact if activated"])

- `p` (600): [TW sentence for `signs_out_everyone`, "Everyone in this workspace is signed out at activation."].
- `dl.m02-sum`: Members N · Open sessions now M.
- `details.m02-members`:
  - summary [TW "Affected members ({n})"], 44px;
  - body `ul.m02-ids` (`columns:18ch auto; column-gap:24px; list-style:none; padding:0`). Each `li` is a bdi mono ID.
  - When there are more than 100 members, add the line [TW "Showing 100 of {n}."].

### 5.7 People and history

- **`section.m02-people`:** `dl.m02-sum` with Proposed by, Validated by, Approved by and Activated by. Each value is a bdi mono ID plus a 14px time; an absent value is the muted [TW "Not yet"].
- **`section.m02-history`:** `ol.m02-history` in chronological order (oldest first, the workflow narrative), up to 20 entries.
  - Each `li` has `border-block-start:1px solid var(--border-row)` and padding-block 12px.
  - Line 1: state word (600), then " · " (outside isolates), the actor (bdi mono) and the time.
  - Line 2: the rationale in `span.m02-free[dir="auto"]` (`unicode-bidi:isolate; overflow-wrap:anywhere`), 68ch. **Never inside `bdi`** (fixes the current Live.tsx:352 defect).
  - Include the plan's "approval invalidated: evidence changed" entry as a history line.

### 5.8 Expert detail (`details.m02-expert`, summary [TW "Technical details"], 44px)

- `dl.m02-sum` with wide rows (dt above dd; dd mono 14px; `overflow-wrap:anywhere`; no ellipsis):
  - full change ID;
  - change revision;
  - base version `g.n` plus the raw epoch decimal string;
  - base policy;
  - old digest; new digest; change digest; evidence digest;
  - ledger sequence and watermark (activated);
  - `superseded_by`.
- A 64-hex digest is about 538px in 14px Plex Mono: one line at 1440 (621px of content) and two lines at 1280 (525px).

### 5.9 Outcome sections (state-specific, placed after §5.3 and before §5.4)

- **`activated`:** `section.m02-activation`:
  - `dl.m02-sum` with New version `g.n`, Ledger sequence and Activated at;
  - h3 [TW "Invalidation progress"] and `table.m02-invalidation` with Item 40% | Result 20% | Detail 40%.
    - Results are words from the plan table: `done`, `in_progress`, `unknown`, `not_applicable`, `outside_scope`.
    - "In progress" and "Unknown" are 600.
    - It stacks below 480px.
- **`activation_unknown`:** `section.m02-waiting`:
  - the plan copy as paragraphs split at sentence boundaries, with the first sentence 600;
  - `dl.m02-sum` with Job (bdi mono `trust-activation-…`), Job state "Waiting", and Reason (`activation_unknown` / `publish_failed` in words plus the code);
  - **no retry button.**
- **`superseded`:** the state sentence (600), plus the fact [TW "Superseded by"] → `button.m02-open` "Change {m}" (it selects that change) and version `g.n`.
- **`rejected` / `cancelled`:** the state sentence only; the rationale is in the history.

---

## 6. Conditions (`section.m02-conditions`, top of the policies view)

The section appears only when at least one condition holds:
- `aria-labelledby` points to a visually hidden h2 [TW "Notices"].
- Each condition is a `div.m02-condition`: padding-block 16px, `border-block-end:1px solid var(--border-row)` (the first also gets `border-block-start`).
- All paragraphs are 16px and 68ch. The **first sentence is wrapped in `<strong>`** and the remaining sentences are separate `p` elements.
- No box fill, no icon, no coloured edge.

Order when several apply:
1. **Trust hold** (activation_unknown copy from the plan). Add the fact line "Since {time}" (`hold.since`) and the change link "Change {n}" (selects it).
2. **Ledger isolated:** the plan copy, plus "(problem at line X)" for `integrity_failed` / `epoch_order`.
3. **Ledger stopped / not confirmed durable:** the plan label "replayed from records not confirmed durable", inside a [TW] sentence frame.
4. **Reset on replacement:** "Sign-in rules were reset to the installation defaults when the decision ledger was replaced on …".
5. **Carried, may be older:** "The carried configuration may be older than the last activation (problem at line X)."

**What remains usable** must be readable at a glance during the hold:
- **[Recommendation, TW decision]** Render the hold's "Until the server restarts, only …" sentence and its "Invitations cannot …" sentence as two `ul` lists under [TW "Still possible"] and [TW "Not possible until restart"]. The items are the plan's clauses verbatim.
- If TW rejects the restructuring, keep the plain paragraphs.
- The rest of M02 stays readable during the hold: current rules, the lists, the detail, and the providers and other views. Only trust actions are absent (§4.1 item 4, §5.3).

In step mode the full conditions are replaced by `p.m02-condition-short` (§5.2).

---

## 7. Propose and revise form (in the panel or step, not a dialog)

- **Placement.** Pressing "Propose trust change" with no selection opens the form in `aside.m02-detail` (split) or as a step (narrow).
- **Revise** opens the same form, prefilled, titled [TW "Revise change {n}"].
- **Deviation from current behaviour:** the form is no longer permanently open at wide widths, because the panel now belongs to the detail. Root must accept this change (§11 item 1).
- **Order:**
  1. h2 (tabindex −1, focused on open);
  2. `p.live-impact` lead;
  3. `service-summary` error summary (focused on a client error);
  4. maximum age (minutes input `10ch + 26px`);
  5. step-up `fieldset.live-roles` with radios;
  6. step-up minutes;
  7. the "values match" note;
  8. `div.live-buttons` with `live-primary` [TW "Review proposal"] and `live-secondary` "Cancel".
- **The rationale moves into C1** (the plan puts a rationale field in each confirmation), so the form has no reason field.
- "Review proposal" validates the fields, then opens C1.

---

## 8. Confirmation dialogs C1–C6

### 8.1 Shared shell (`dialog.m02-confirm`, `showModal()`)

```
dialog.m02-confirm  inline-size:min(720px, calc(100vw - 32px)); max-block-size:90dvh; padding:24px;
                    overflow:auto; (global border 1px --hb-line-header, radius 0, opaque backdrop kept)
  h2#m02-confirm-title (20/600, tabindex=-1; focused on open)   "<Action> — Change {n}" [TW pattern, no dash glyph in AR: use a localized template]
  p.m02-confirm-lead   (16px, 68ch) what this command does
  div.service-summary  (role=alert, tabindex=-1)  failure or field errors, when present
  dl.m02-confirm-facts grid 2 cols at dialog content ≥600px (gap 4px 32px); .m02-wide rows span both;
                       each row dt 14px muted / dd 16px; digests dd mono 14px overflow-wrap:anywhere
  [section-specific blocks]
  div.m02-consequence  paragraphs, 68ch; permanence and sign-out sentences 600
  p.m02-boundary       cancellable boundary sentence (16px)
  div.service-field.live-reason  label, hint, error, textarea rows=4, max-inline-size 440px, byte counter 14px muted
  div.live-buttons.m02-confirm-actions  [live-primary <action label>] [live-secondary "Go back" TW]
```

**Rules:**
- **Every dialog shows these common facts first:**
  - Change "Change {n}" plus the full ID (mono);
  - Based on version `g.n`;
  - Policy revision R;
  - Change revision k;
  - current state word.

  This is what the plan's "target, epoch, revision" requirement and the spec's "target, revision, impact, cancellable boundary" requirement call for.
- **The secondary is [TW "Go back"], never "Cancel".** In C6 a "Cancel" button next to "Cancel trust proposal" is ambiguous.
- **Buttons are primary first, at inline-start.** This is the `.live-buttons` page pattern and the P05 confirm pattern. The HYB-293 Reauthenticate dialog uses end-aligned Cancel-then-Confirm. That dialog is not changed; the difference is recorded here.
- **Escape, Go back and the backdrop:**
  - Escape and "Go back" close the dialog without executing. Block both while a request is in flight, the same way `cancellable(busy)` does in Reauthenticate.
  - The backdrop does not close the dialog.
- **Focus return:**
  - Focus returns to the opener.
  - If the opener no longer exists after a reload, focus goes to `h2.m02-detail-title`.
- **Step-up (HYB-293).**
  - If the command answers `reauthentication_required`, the Reauthenticate dialog opens **above** the confirm dialog. The confirm dialog stays open, with its primary showing busy.
  - On success the same request is resent once and the Reauthenticate dialog returns focus to the confirm primary.
  - Two stacked modal dialogs are UNVERIFIED in render (§12 check 9).
- **Narrow (<768px viewport):** `inline-size:100vw; max-inline-size:100vw; block-size:100dvh; max-block-size:100dvh; margin:0; padding:16px; border:0`.
  - The facts are one column.
  - Buttons are 100% width and stacked in DOM order.
  - **Actions stay at the end of the scroll flow, never sticky**, so they never cover errors (spec).
  - The textarea uses `inline-size:calc(100% - 6px); margin-inline:3px` so the focus ring is not clipped.

### 8.2 Per-confirmation content (in this order, after the common facts)

| ID | Specific facts and blocks | Consequence / permanence (600 where marked) | Boundary | Primary |
| --- | --- | --- | --- | --- |
| C1 propose / revise | Workspace; Route "Local password"; Members affected (count). Comparison table Current / Proposed (§5.4 widths; stacked <440px). Wide rows: Current configuration digest; Candidate digest with the 14px muted label "expected" and the plan's note that the server value is authoritative. Revise adds "Change revision k → k+1" as two facts, with no arrow glyph | If activated later: **everyone in the workspace is signed out (600)**; lockout risk is checked at validation (test login, recovery) [TW] | Creates a draft only; active sign-in rules do not change; you can revise or cancel until activation [TW, from plan state copy] | "Propose trust change" / [TW "Revise trust change"] |
| C2 validate | Wide: change digest. Two lists: h3 [TW "Checked"] (Test login, Recovery path, Record size) and h3 [TW "Not applicable"] (Issuer and stable subject, Provider diagnostics, External identity status, each with its reason) | **Your current session is used as the test login under the proposed rules (600)** [TW]. A blocked result keeps the change a draft | Validation records evidence only; it does not approve or activate | "Validate trust change" |
| C3 approve | Proposed by (bdi); Validated by, at. Wide: change digest; evidence digest (both full). Your authority: [TW "You hold the approval right and did not propose this change."]. Recovery finding result plus reason (lockout impact). Members N; open sessions now M | **Everyone is signed out at activation (600)**. Activation rechecks authority and the old version | Approval is cleared if the change is revised or the evidence changes; reject and cancel stay possible until activation | "Independently approve trust change" |
| C4 activate | Wide: approved change digest; evidence digest. Proposed by, Validated by, Approved by (at), Activated by: [TW "You"]. Base version `g.n`; [TW "Expected new version"] `g'.n'` labelled expected (computed as `max(E0+1, (ledger.generation<<32)+1)` from the separate fields, display only; the server is authoritative). Affected members: count plus `details` list (§5.6). Test login and recovery findings (lockout risk) | **Everyone in this workspace is signed out, including you (600).** Open sessions now M. **Undoing this needs a new trust change with a newer version (600)** | **After you confirm, this change cannot be rejected or cancelled (600)** | "Activate approved trust change" |
| C5 reject | Proposed by; state word | Active sign-in rules do not change. **A rejected change cannot be reopened; propose again if needed (600)** [TW] | Rejection is final | "Reject trust change" |
| C6 cancel | State word | Active sign-in rules do not change. **A cancelled proposal cannot be reopened (600)** [TW] | Cancellation is final | "Cancel trust proposal" |

### 8.3 Refusals inside a dialog

| Failure | Dialog behaviour |
| --- | --- |
| `invalid_request` with a field (rationale) | Error summary linking to the textarea; the primary stays |
| `self`, `epoch`, `approval_stale`, `evidence_stale`, `needs_approval`, `trust_unavailable`, `invalid_stage`, `permission_denied` (plan refusal copy) | `service-summary` at the top (focused) with the plan copy. **The primary is removed**, because the facts shown are no longer trustworthy. The secondary becomes [TW "Reload and close"]: it reloads, closes and moves focus to the detail heading |
| `idempotency_conflict`, `revision_conflict` | Same as the row above |
| `unauthenticated` | Close the dialog and show the existing sign-in-again state |

### 8.4 After the command

| Command | What follows |
| --- | --- |
| Propose, validate, approve, reject, cancel | Close; reload; announce through the status region; focus `h2.m02-detail-title` of the affected change (the new change after propose) |
| Activate, `completed` | The caller's session has ended (`your_session_ended:true`). Replace the whole view with `section.m02-after` (role=status, h2 tabindex −1, focused): [TW "Activated"], then the plan "activated" sentence, `dl` with new version `g.n`, ledger sequence, sessions ended, sessions remaining, then `button.live-primary` "Sign in again" |
| Activate, `accepted` (`activation_unknown`) | The same section with the plan activation_unknown copy, the job ID and "Waiting". **No retry**; "Sign in again" only |

- **Deviation from the plan, for root to decide:** "After activation" lists invalidation items, but the read is no longer reachable once the session has ended. `section.m02-after` therefore shows only the `sessions_ended` and `sessions_remaining` counts from the response. The full invalidation table appears in the change detail after sign-in.

---

## 9. Whole-view states

| State | Presentation |
| --- | --- |
| Loading | The spec sentence in `p[role=status]` under the current-rules heading; `aria-busy` on `.m02-live`. No skeleton numbers |
| Load error | The existing `.live-error` (its `--hb-danger` heading is an error state) plus Try again / Sign in again |
| Denied | The existing `Unavailable` block |
| **Revoked** | `.live-unavailable` with the spec revoked copy and Reload. **Lists, detail, dialogs, selection and the propose form are removed from the DOM**; any open dialog is closed (plan: cleared on `permission_denied` / `unauthenticated`) |
| Empty (no changes) | Both groups show their empty sentences; the opener remains if allowed |
| Partial | The §4.1 item 5 note and the §6 conditions; never presented as zero or complete |

---

## 10. Focus order, keyboard, RTL, narrow

### 10.1 Tab order (split mode)

1. shell tabs;
2. Reload;
3. any condition links ("Change {n}");
4. Propose opener;
5. Open list openers (row order), then Closed list openers;
6. the aside scroll container (`tabindex=0`);
7. Close;
8. the detail action buttons;
9. links in the outcome section;
10. `details` summaries (affected members, technical details).

- **Opening a change** focuses `h2.m02-detail-title`.
- **Close, Back, or Escape** while focus is in the panel and no dialog is open returns focus to that row's "Change {n}" opener.
- **URL or programmatic selection** moves no focus ring. Add `:focus:not(:focus-visible){outline:none}` only on tabindex −1 headings, notices and summaries.
- **The global `:focus-visible`** (2px `--hb-focus`, offset 2px) is never overridden on controls.
- **Status announcements** go through the single polite region, once per command result. Rows are never re-announced on reload.

### 10.2 RTL and multilingual

- **Logical properties only.** The panel divider is `border-inline-start`, so in AR the panel is on the left with the divider on its right.
- **bdi dir="ltr" values:**
  - versions `g.n`, digests, IDs, codes, job IDs and actor IDs are `bdi dir="ltr"` (mono where listed);
  - dates use the existing `when()` helper;
  - separators (" · ") stay outside isolates.
- **Free text** (rationales) uses `span.m02-free[dir=auto]`, never `bdi`.
- **AR** keeps the console line-height of 1.8; do not set line-height in `.m02-*` running text. **KO** gets `word-break:keep-all` on headings and prose.
- **No glyphs:** no →, ←, ✓ or ✗ in any locale (design-system.md bans Unicode symbol icons).
- **Count sentences** are full per-locale templates with the number inside, never assembled from fragments.

### 10.3 Narrow (390, `.m02-live` ≈ 358)

- Order: conditions → current rules (single column, dt above dd) → changes head (opener full width) → stacked lists → step detail.
- Buttons are 100% width. The 44px minimum applies to openers, summaries, Back and Close.
- No horizontal scroll anywhere.
- Digests wrap at about 40 characters per line.

---

## 11. Problems in the plan and decisions root must take (layout and states)

1. **Propose form no longer always open at wide widths (Medium, root decision).** The panel is now the detail. I specify an opener (§7). If root keeps the always-open form, there is no room for list-detail at 1280.
2. **"State (text and icon)" (Medium, owner/root).** The design system supplies only `chevron-down` and `close` icons and bans Unicode symbols. I specify **text-only** states, which meets "not colour alone". Adding state icons needs canonical SVGs from the design system owner; inventing them is not allowed.
3. **Short state labels and new headings are missing from the plan copy (Medium, TW).** The [TW] items include:
   - list labels (Draft, Validated, Independently approved, Activated, Rejected, Cancelled, Superseded, and for `activation_unknown` a label that must not contain "Activated", for example "Activation not confirmed");
   - section headings, "Go back", dialog leads and boundaries.
4. **Dialog secondary label (Low).** "Cancel" next to "Cancel trust proposal" is ambiguous; use "Go back".
5. **Invalidation after activation (Medium, truthfulness).** The caller is signed out, so only the response counts can be shown; the full table appears after sign-in (§8.4).
6. **C4 "new epoch" (Low).** The value is the server's; display it labelled "expected", or show only "newer than g.n" if root prefers no client computation.
7. **Arabic rationale direction (existing defect, Medium).** Live.tsx:352 wraps the rationale in `bdi`; replace it with `span[dir=auto]`.
8. **Two stacked modals with step-up (Low, UNVERIFIED).** This needs a render and keyboard check.

---

## 12. Rendered-QA checklist (measurable)

For every render, record: viewport, locale, state, and the measured widths of `.m02-live`, `.m02-lists` and `aside.m02-detail`.

**Layout and widths**
1. `.m02-live` is about 1120 at 1440, about 960 at 1280 and about 358 at 390. `document.scrollingElement.scrollWidth <= clientWidth` holds on every render, including the open dialogs.
2. **No selection:** the full five-column tables at 1440 and 1280. No `th` text overflows its cell; check "Proposed by" and "Based on" in EN and AR.
3. **Selection at 1440 and 1280:**
   - the list is about 435 and 371px and the detail about 653 and 557px;
   - the reduced single-cell rows;
   - the detail divider's computed `border-inline-start` is 1px in the `--border-row` colour (`#dce1e5`);
   - in AR the detail is on the left.
4. **Sticky detail:** it stays in view while the page scrolls. Its bottom content can be reached with the mouse wheel and with the keyboard (Tab to the aside, then the arrow keys). Nothing is hidden under a header.
5. **Step mode at 390, and at 1024 if tested:**
   - the current rules, changes header and lists are absent from the DOM;
   - Back comes first;
   - focus returns to the same opener.

**States, findings and conditions**
6. **Validation findings:**
   - nine items in total (six checks in the table, three informational);
   - the result words are plain;
   - only "Blocked" is 600;
   - each reason shows the code in mono;
   - at 390 the rows are stacked.
7. **Truthfulness sweep:**
   - In EN, KO and AR, inside `.m02-live` and the dialogs, the words for active, approved and activated appear **only** for changes whose server state is `independently_approved` or `activated`, and in the current-rules "Activated by/at" facts.
   - `activation_unknown` rows never contain the activated word.
   - Expected digests and versions carry the "expected" label.
   - There are no ✓ ✗ → ← characters.
8. **Conditions:**
   - The hold, isolation, durable-unconfirmed, reset and carried blocks render in §6 order, with the first sentence 600 and 68ch (computed `max-inline-size` ≤ 68ch).
   - During the hold, no trust action buttons or Propose opener are in the DOM.
   - The enrolment block appears when qualifying_people < 3.

**Dialogs, refusals and controls**
9. **Dialogs C1–C6:**
   - exactly one `.live-primary` each;
   - the common facts come first;
   - full 64-hex digests with no ellipsis;
   - the permanence sentences are 600;
   - the textarea is ≤440px wide at 1440;
   - Escape and Go back do not execute;
   - focus returns to the opener;
   - focus is trapped inside the dialog;
   - at 390 the dialog is full screen, its buttons are not sticky and the focus ring is not clipped;
   - the step-up dialog stacks over C2 and restores focus to the confirm primary.
10. **Refusal:** a stubbed `evidence_stale` in C4 removes the primary and shows "Reload and close".
11. **Buttons:** content width with a 44px minimum height at ≥768px viewport, never stretched across the container; 100% width at 390.

**Styling, language and focus**
12. **Computed-style sweep over `.m02-live *` and `dialog.m02-confirm *`:**
    - every border colour resolves to `--border-row`, `--border-control`, `--hb-line-header` (the dialog) or `--hb-ink` (an invalid field only);
    - `border-radius` is 0;
    - there is no `box-shadow`;
    - no `border-inline-start` is wider than 1px;
    - backgrounds are only `--surface-page`, `--surface-hover` (the selected row) or `--surface-primary` (the primary button).
13. **RTL (AR):**
    - rationales are in `span[dir=auto]`, not `bdi`;
    - `g.n` and the digests read left-to-right;
    - Latin digits stay inside the version isolate;
    - line-height is 1.8;
    - the panel is on the left.
14. **KO:** `keep-all` with no mid-word breaks in headings, state lines or the enrolment block.
15. **Focus:** a visible 2px `--hb-focus` ring on the detail heading after a keyboard open, on the confirm title, on the error summary and on `section.m02-after`. There is no ring after a URL-driven selection.
16. **Revoked:** after a stubbed `permission_denied`, the DOM contains no list rows, detail, dialog, form or "Selected" tag.
17. **Not covered by this checklist:** owner acceptance, terminology correctness, native-speaker review and the real `activation_unknown` (render tests only, per the plan).

---

## 13. External references

Root opened the three references on 2026-11-18. What was observed, adopted and rejected is in [references.md](references.md). The ui-designer could not open them; the HYB-315 precedents (GOV.UK Summary list, GOV.UK Error summary, Android list-detail) apply as recorded in HYB-315.

## 14. Tokens, type, classes, implementation item IDs

**Tokens (existing only):**
- `--hb-ink`, `--text-muted`;
- `--border-row`, `--border-control`, `--hb-line-header` (the global dialog);
- `--surface-page`, `--surface-hover`, `--surface-primary`, `--text-on-primary`;
- `--hb-focus`, `--hb-font-mono`;
- `--hb-danger` (only through `.live-error`).

No colour literals, radii, shadows or new tokens.

**Type:**
- h2 20/600; h3 16/600; body 16; small and labels 14; mono 14;
- prose `max-inline-size:68ch`.

**Spacing:** 4 / 8 / 12 / 16 / 24 / 32 / 40 / 48.

**Classes** (all scoped under `[data-security-screen="M02"] .m02-live`, or under `dialog.m02-confirm` for the dialogs; reuse `live-*`, `service-field`, `service-summary` and `service-button-link`):

| Area | Classes |
| --- | --- |
| Frame and state | `m02-live`, `m02-conditions`, `m02-condition`, `m02-condition-short`, `m02-current`, `m02-toolbar`, `m02-current-grid`, `m02-group`, `m02-facts`, `m02-epoch`, `m02-ledger`, `m02-notes`, `m02-note` |
| Changes and list | `m02-changes`, `m02-changes-head`, `m02-propose`, `m02-holders`, `m02-enrolment`, `m02-rights`, `m02-unavailable-line`, `m02-split`, `m02-lists`, `m02-table`, `m02-open`, `m02-sub`, `m02-state-word`, `m02-tag` |
| Detail | `m02-detail`, `m02-step`, `m02-back`, `m02-close`, `m02-detail-head`, `m02-detail-title`, `m02-state-line`, `m02-stale`, `m02-next`, `m02-actions`, `m02-why`, `m02-compare`, `m02-validation`, `m02-result`, `m02-findings`, `m02-sum`, `m02-wide`, `m02-impact`, `m02-members`, `m02-ids`, `m02-people`, `m02-history`, `m02-free`, `m02-expert`, `m02-activation`, `m02-invalidation`, `m02-waiting` |
| Dialogs and after | `m02-confirm`, `m02-confirm-lead`, `m02-confirm-facts`, `m02-consequence`, `m02-boundary`, `m02-confirm-actions`, `m02-after` |

**Proposed implementation item IDs** (screen-review.md). Root must reconcile them with any existing M02 IDs and must not renumber those:
- `M02:trust:conditions`, `M02:trust:current-rules`, `M02:trust:holders`, `M02:trust:enrolment-block`, `M02:trust:propose-opener`, `M02:trust:propose-form`;
- `M02:trust:list-open`, `M02:trust:list-closed`, `M02:trust:detail`, `M02:trust:validation`, `M02:trust:impact`, `M02:trust:history`, `M02:trust:expert`, `M02:trust:invalidation`, `M02:trust:waiting`;
- `M02:trust:confirm-c1` … `M02:trust:confirm-c6`;
- `M02:trust:after-activation`, `M02:trust:revoked`.

---

## Files referenced

- docs/features/HYB-297/plan.md
- docs/ui/screens/M02/spec.md
- docs/ui/screens/M02/design.json
- apps/console/src/modules/identity/pages/authentication/Live.tsx
- apps/console/src/modules/identity/pages/users/live.css
- apps/console/src/app/screen-groups/identity.tsx
- apps/console/src/app/service-shell.css
- apps/console/src/shared/styles.css
- apps/console/src/service/Reauthenticate.tsx
- apps/console/src/service/service.css
- packages/design-system/styles/tokens.css
- docs/features/HB-265/verification.md
- docs/features/HYB-315/visual-spec.md
- docs/features/HYB-315/references.md
- .agents/rules/service-screen-style.md
- .agents/rules/design-system.md
- .agents/rules/screen-review.md
- .agents/rules/usability.md
- docs/design-system.md
- docs/design-build-guide.md

## Open uncertainties

- Root decisions on §11 items 1, 2, 5 and 6.
- The [TW] strings.
- External references unopened (§13).
- No current M02 screenshots supplied.
- The 390 container width and the stacked-modal behaviour are unverified.
