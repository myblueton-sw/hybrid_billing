---
type: DesignSpecification
title: HYB-315 P05 visual specification
description: Binding visual source for the P05 release evidence screen in service mode, written by the ui-designer role and amended by root; the plan wins on copy, truthfulness and state wording.
status: stable
tags: [HYB-315, P05, release-evidence]
sources:
  - resource: plan.md
---

Root amendments 2026-11-18: where this spec and the HYB-315 plan differ on copy, truthfulness or state wording, the plan wins; this spec wins on layout only. The plan section "Root amendments after review (2026-11-18)" overrides this spec on layout as well, including §4.3 column widths, §4.5 step tables, the stacking threshold for small registers and §14.4.

HYB-315 visual specification: P05 "Support combinations and release evidence" (/release-evidence), live service mode

Author: ui-designer role, read-only. The model reports itself as Claude Opus 5.5 (claude-opus-5-5). This is a design proposal, not owner acceptance, and a later visual PASS would not be owner acceptance either.


---

# 0. Evidence basis and limits

**Directly observed (files read):**
- /private/tmp/hb315/docs/features/HYB-315/plan.md (whole file)
- /Users/home/project/hybrid_billing/docs/ui/screens/P05/spec.md, design.json and wireframe.svg
- /Users/home/project/hybrid_billing/.agents/rules/service-screen-style.md
- /Users/home/project/hybrid_billing/docs/design-system.md (brand and state rules)
- /Users/home/project/hybrid_billing/apps/console/src/modules/administration/pages/retention/live.css (the P04 precedent)
- /Users/home/project/hybrid_billing/apps/console/src/modules/identity/pages/users/live.css (the shared `.live-*` base: buttons, tables, facts, back link, 440px field cap)
- /Users/home/project/hybrid_billing/apps/console/src/shared/styles.css: the global focus rule is `outline: 2px solid var(--hb-focus); outline-offset: 2px`. The rail is 246px; at ≤1279px it is 210px and the gutter 24px. At ≤1023px the rail moves on top, and at ≤767px the gutter is 16px.
- The preview Screen.tsx at /Users/home/project/hybrid_billing/apps/console/src/modules/administration/pages/release-evidence/Screen.tsx
- /private/tmp/hb315/docs/features/HB-289/verification.md. It records the measured `.p04-live` width as 1120px at 1440, 960px at 1280 and 358px at 390. This spec uses those widths. P05 has not been measured.

**Not available:**
- No current P05 live render or screenshot was supplied, and I did not view any. The current service-mode P05 is the preview.
- I did not read docs/design-build-guide.md, design-system.md under .agents/rules, screen-review.md, usability.md, terminology.md, the P03 recovery CSS or service.css in full. I grepped service.css only for tokens.
- I ran no browser and no commands. Nothing here is render-verified.

**Wireframe observations (direct):**
- At 1440 the list card is about 759px and the detail card about 391px wide.
- In that split, the list table shows only 5 of the 8 spec columns (조합, 요구 ID, 설계, 구현, 시험). I use this as support for a reduced column set beside the panel.
- Wireframe traits that must NOT be adopted:
  - rounded corners (rx 4–8);
  - the blue 3px active-tab underline bar;
  - the blue "설계 예시" chip;
  - a filled primary "증거 연결" button (that action is not connected);
  - the filter bar (no filters exist).

---

# 1. Composition principles for this screen

1. **Evidence ledger, not dashboard.** No hero numbers, progress bars, gauges, percentages, coverage bars or charts. Counts are always "{n} of {m}" text, never a fill. A full bar for "2 of 2 designed" would read as done.
2. **One reading spine.** The order is frame → tab heading → lead → primary list → detail → actions. Width is used for parallel reference (list beside detail, side-by-side small registers), not for decoration.
3. **Absence is typographic.** Every non-value (not evaluated, unknown, not connected, not measured, not run, not counted) is a muted word, never blank, a dash or 0. A real 0 looks like any other number.
4. **No positive emphasis anywhere.** Nothing gets success styling. The only weighted emphasis goes to problems, permanence and the unexpected-value sentence, and all of it stays in ink.
5. **Full width, capped measure.** Regions span `.p05-live`. Prose is capped at 68ch, fields at 440px and buttons at their content width (44px high) above 767px viewport.

---

# 2. Container sizes and measured basis

`div.p05-live { display:block; min-inline-size:0; container-type:inline-size; }`

| Viewport | Rail and gutter | `.p05-live` width | Container | Detail mode (this spec, see §4) |
| --- | --- | --- | --- | --- |
| 1440 | 246 + 2×32 | ~1120 (P04 measured) | L | side by side |
| 1280 | 210 + 2×24 (≤1279 rule; 1280 is outside it, so 246 + 64) | ~960 (P04 measured) | M | **side by side (deviation, §4.1)** |
| 390 | rail on top, gutter 16 | ~358 (P04 measured) | S | step |

- Table breakpoints stay as in the plan: S <720, M 720–1079, L ≥1080, all as `@container` rules on the table's own container (§3). Exception (root amendment 2026-11-18, design review V2): the reduced table beside an open panel (`.p05-split[data-open="true"] .p05-table.p05-reduced`) never stacks; it keeps its reduced columns at any list width, and only step mode replaces it.
- The only `@media (max-width:767px)` uses are:
  - the 3px control inset;
  - full-width buttons and openers;
  - the scope-facts reflow;
  - the shell tab grid override.

---

# 3. Containers within containers

- `.p05-live`: the page container. It decides the frame grid and the split mode.
- `.p05-list` (`container-type:inline-size; min-inline-size:0`): the list region. Every table inside decides S/M/L and nowrap from this container, never from `.p05-live`.
- `aside.p05-detail` (`container-type:inline-size; min-inline-size:0`): the detail panel or step. Its inner tables stack below 720px of the panel's own width.
- `.p05-reg` (artifacts registers, load profile articles): each is its own container.

**Rule:** nowrap on IDs applies at `@container (min-width:1080px)` of the **nearest** container. In split mode the list region is about 600px, so IDs there must wrap at `/` (see §11).

---

# 4. Evidence detail: split and step

## 4.1 Threshold (deviation from plan decision 4; root must accept or reject)

**Proposal:** side-by-side when `.p05-live` is **≥ 940px**, step mode below. The table S/M/L thresholds are unchanged.

**Why:**
- spec.md:109 asks for side-by-side on desktop, and 1280 is a mainstream desktop width;
- at 960px the split still gives a list of about 563px and a panel of 365px, enough for the reduced list (§4.3) and the stacked panel content;
- at 1080 the rule would put every 1280 desktop into step mode, which the spec describes as the mobile behaviour.

**If root keeps 1080:** everything else in this spec stands. 1280 then renders the step mode, and Group F keeps its current expectation. Either way `useAvailableWidth` and the `@container` rule must use the **same** number. Name it `--p05-split` in the copy and tests, even though CSS container queries cannot use variables.

## 4.2 Split grid

```
.p05-split            { display:grid; grid-template-columns:minmax(0,1fr); gap:32px; align-items:start; }
@container (min-width:940px) {
  .p05-split[data-open="true"] { grid-template-columns: minmax(0,1fr) clamp(360px, 38%, 460px); }
}
```

- Panel width is 426px at 1120 and 365px at 960, so the list is about 662px and 563px.
- Without a selection the grid is one column and the list uses the full width with the full table. No empty panel is drawn and no "Select a combination…" box takes 40% of the screen.
  - Recommended deviation: show "Select a combination to see its evidence." as a one-line muted sentence under the tab lead instead.
  - The plan's empty-panel text is acceptable only if root insists. It wastes 426px.
- **Panel:** `border-inline-start:1px solid var(--border-row); padding-inline-start:32px;`. This is the neutral divider used by `.live-users`, not an accent; it never uses a brand colour and is never thicker than 1px.
- **Sticky:** `position:sticky; inset-block-start:16px; max-block-size:calc(100dvh - 32px); overflow-y:auto; overscroll-behavior:contain;`.
  - The scroll container is the aside itself. Give it `tabindex="0"` so a keyboard user can scroll it. It shows the global focus outline (inset with `outline-offset:-2px` so the sticky edge does not clip it).
  - Check whether the shell header is sticky. If it is, `inset-block-start` must equal its height plus 16px. This is UNVERIFIED and needs a render check.
- **DOM order:** list first, then panel, as in reading order. In RTL the panel lands on the left automatically through the grid. No `order` or `float`.

## 4.3 Reduced list beside an open panel (`.p05-split[data-open="true"] .p05-table`)

The full 8-column matrix cannot fit 563–662px. The plan's container rule would turn the whole list into stacked cards at the moment a row is opened, which is a jarring reflow and loses comparison across rows. Instead, a dedicated reduced column set is rendered (a different `colgroup` and cell set chosen by the `open` state, not by CSS hiding):

| Tab | Reduced columns (width) |
| --- | --- |
| Matrix | Combination: path ID, provider and issuance sub-line, path state, Evidence link (46%) · Test (18%) · Customer proof (16%) · Blockers count "{n} problems" (plural-aware), "Not evaluated" for null, the rule 1 sentence for `[]`, never "0 problems" (20%) |
| Requirements | Requirement: label, tags, Evidence link (48%) · State at the candidate build (28%) · Tests summary (24%) |

**The panel must then repeat every column hidden from the reduced list (plan gap, §14.2).**
- Matrix panel: the five combination facts, requirement references with states, design, implementation, test with failing, customer proof with the intake sentence, last observed, and the blockers in full.
- Requirements panel: the source, owner reference and interfaces facts and the implementation commit.

## 4.4 Step mode (below the threshold)

- With a selection, render only:
  - the shared frame (§5);
  - the tab heading;
  - a "Back to the list" link (`.p05-back`: 44px min block size, underline, 16px, ink, no arrow glyph);
  - the detail (`section.p05-step`, not `aside`).
- The tab lead, gate block, compare control, list and not-connected actions are **not rendered**. Plan decision 4 is ambiguous here; this spec decides it.
- Without a selection, render the normal list view.

## 4.5 Panel and step internal layout

- **Header row (`.p05-detail-head`):** flex, wrap, `justify-content:space-between`, gap 8px 16px.
  - `h2.p05-detail-title` (20px/600, `tabindex=-1`), containing the ID in mono 16px/600.
  - Sub-line (14px muted): provider · issuance path (each an ltr isolate). The middle-dot separator sits **outside** the isolates.
  - "Close" (`a.p05-close`, 44px hit area, underline) at inline-end, in split mode only.
- **Fact list (`dl.p05-sum`, GOV.UK summary-list shape without borders between rows):**
  - Each row is `div` with `grid-template-columns:minmax(120px,38%) minmax(0,1fr)`, gap 4px 16px, padding-block 8px. dt is 14px muted; dd is 16px.
  - `div.p05-sum-wide` (digests, commits, blocker lists): dt above dd, dd mono 14px with `overflow-wrap:anywhere`. 64 hex is about 538px, so in a 360–426px panel it wraps to 2 lines. No truncation or ellipsis, because the value must be copyable.
  - No rule above the first fact. Groups are separated by `h3` (16px/600, margin 24px 0 8px) and spacing, with no borders between groups.
- **Step mode at M:** `dl.p05-sum` becomes `grid-template-columns:repeat(2,minmax(0,1fr))` at `@container (min-width:720px)`, with wide rows spanning both columns. At S it is a single column with dt above dd.
- **Categories (`dl.p05-cats`):** 6 items.
  - Panel: one column.
  - Step at M: 3 columns. Step at S: 1 column.
  - Each item: the category word (600), then the state sentence (16px regular).
  - "No counted pass at the candidate build" and "Counted pass recorded at the candidate build" have identical weight. Never use a tick or a cross.
- **Reconciliations:** a small table (Period | Matched | Differences, 40/35/25%) that stacks below 720px. In the panel it is always stacked as one line per period: "2026-10 · Not matched · 3 differences".
- **Not-connected block (`ul.p05-nc`):** each item is the label (600) followed by "Not connected" (muted). No borders, no buttons.
- **Tests (requirements detail):**
  - At panel and S width: stacked blocks. Each block is `div.p05-test`, padding-block 12px, `border-block-start:1px solid var(--border-row)` from the second block on.
    - Line 1: test ID mono 14px/600.
    - Line 2: `dl.p05-inline` (flex wrap, gap 4px 24px) with Category · Outcome · Counted · Passes · Failures.
    - Line 3: the reason sentence (16px, 68ch).
  - At step M (≥720): a table with Test 22 | Category 14 | Outcome 14 | Counted 12 | Passes 8 (end-aligned) | Failures 8 (end-aligned) | Reason 22.
- **States by build:** a two-column list (build mono | state words), newest first.

---

# 5. Shared frame (every tab)

```
.p05-scope        (flex wrap; gap 16px 24px; align-items:flex-end; padding-block-end:16px;
                   margin-block-end:24px; border-block-end:1px solid var(--border-row))
  dl.p05-scope-facts: Workspace | Data revision | Now (UTC) | Release candidate
  button.live-secondary "Reload" (44px)
.p05-frame        (grid; 1 column; gap 8px 48px; margin-block-end:24px)
  .p05-frame-main : p.p05-statement (bold lead sentence + rest, 16px, 68ch)
                    p.p05-candidate (16px, 68ch)
  .p05-frame-notes: p.p05-note ×3 (origin, open decisions, permission; 14px muted, 68ch, margin 0 0 8px)
@container (min-width:1080px) { .p05-frame { grid-template-columns: minmax(0,1.1fr) minmax(0,1fr); } }
```

- **Release candidate fact in the scope bar:** "{version} at {build}" (each an ltr isolate), or "None selected" (muted).
  - The full candidate sentence stays in `.p05-candidate` so the gate-only caveat is read.
- **At L,** the statement and candidate lines sit beside the three notes. This cuts the preamble from about 9 lines to about 4. At M and S it stacks: main first, then notes.
- **Origin note:** with `sample_seed`, the first words "Sample records." are weight 600. Do not use a tag or chip, because the wireframe's blue chip is not adopted.
- **The tab heading (h2, 20px/600) follows the frame.** Then come the lead (`p.p05-lead`, 16px, 68ch, margin 0 0 24px) and the matrix blocked sentence as a second lead paragraph (600 for its first sentence).
- **Notice and failure:**
  - `#p05-notice` (`role=status`, `tabindex=-1`): padding 16px 0, `border-block:1px solid var(--border-row)`, 16px, 68ch. Success messages are ink, with no green and no icon.
  - The failure box uses the shared `.live-error` (its existing `--hb-danger` heading is acceptable as an error state) and adds the spec error sentence as a second paragraph.
- **Whole-page states** (loading, not_configured, unknown state, denied, revoked, unauthenticated, failure) render inside `.p05-live` after the scope bar as `.live-unavailable` (68ch). No skeleton numbers and no spinners with percentages. Loading is the sentence only, with `aria-busy` on the screen.
- **Revoked:** the frame keeps only the scope bar and the revoked block. No tables, forms, panels or "Close" links remain in the DOM.

---

# 6. Tab layouts

All tables follow the same pattern:
- `div.live-table > table.p05-table` with `table-layout:fixed`, a colgroup and a caption (visually hidden when a heading names it);
- one `tbody.p05-row` per record with `border-block-start:1px solid var(--border-row)`;
- an optional `tr.p05-follow > td[colspan]` for row notes and forms;
- td padding-block 12px, `vertical-align:top`, th 14px/600.

Numbers are `tabular-nums` and end-aligned at M and L, start-aligned when stacked.

## 6.1 Support matrix

**Order:** lead → blocked sentence → `.p05-split` (list, panel) → `h3` "Not connected in this build" with `ul.p05-nc` (Link evidence / Record support decision / Request activation review, each followed by "Not connected in this build" in muted text).

**Full table, L (list region ≥1080, i.e. 1440 with no panel open):**

| Col | Width | Content |
| --- | --- | --- |
| Combination | 16% | Path ID mono 600 (row title) → `dl.p05-combo` stacked: Provider, Contract, Role, Version, Issuance (dt 14px muted inline + dd; one fact per line) → Template or "No template" → path state words → `a.p05-evidence` "Evidence" |
| Requirement IDs | 15% | `ul` of labels (link when target is well-formed); state words below each in 14px; "Showing 50 of {total}." |
| Design | 7% | "{count} of {of}" |
| Implementation | 12% | "{count} of {of}" + 14px "with a recorded commit" |
| Test | 10% | "{count} of {of} tested at the candidate build" / "{failing} failing" (600 when > 0) |
| Customer proof | 10% | "{count} of {of}" + 14px muted "Intake not connected" (see §14.4) |
| Last observed | 11% | Date and time as two nowrap parts, stacked |
| Blockers | 19% | `ul.p05-problems`: problem words (600) + subject mono 14px on its own line; an empty `[]` is not shown as "no problem": it renders the rule 1 unexpected-value sentence (root amendment, V1); "Not evaluated" |

- The widths are based on the EN header "Implementation" (about 112px at 14px/600) needing about 128px with padding. Root must verify that no th text overflows (checklist §13).
- **M (720–1079, i.e. 1280 with no panel):** 7 columns. "Last observed" folds under Test as a 14px sub-line headed "Last observed". Widths: Combination 19 · Requirements 16 · Design 8 · Implementation 14 · Test 13 · Customer proof 11 · Blockers 19.
- **S (<720) and 390:** stacked labelled rows. Path ID 16px/600 first, then each column as `td[data-label]::before` (14px/600 label above the value). The Evidence link is last in the row, 44px hit area. Combination facts form a 2-column `dl` at ≥480px of the list, one column below that.
- **Split (1440 and, under §4.1, 1280):** the reduced list from §4.3.

## 6.2 Requirement trace

**Order:** lead → `.p05-gate-row` → `.p05-split` (list, panel).

**`.p05-gate-row`:**
- A grid with `grid-template-columns: minmax(0,7fr) minmax(0,5fr)` and gap 48px at `.p05-live` ≥1080. Stacked below that.
- **Left, `section.p05-gate`:**
  - h3 "Gate check for {version} at {build}: not ready, {n} problems."
  - Problems table: Problem 55% | Subject 45%. At S it stacks.
  - `dl.p05-sum`: Retired without replacement · Templates (wording and rule 1 handling as in the plan: "None. A template is listed only for an active combination…") · Release: Not connected.
- **Right, `section.p05-compare`:**
  - h3 "Compare builds (partial)" and its 14px muted hint (68ch).
  - `div.p05-pair`: two native selects, "Compare build" and "with build", each 44px high and `inline-size:12ch + 40px` (builds are short semver).
  - The result table: Requirement | State at {from} | State at {to}. No arrow glyph (§14.3). An "unchanged" result shows the sentence instead.

**Requirement list, full (L and M, no panel):**

| Col | Width | Content |
| --- | --- | --- |
| Requirement | 24% | Label mono 600 + `<wbr>` after "/" → tags row (`.p05-tags`: Common, Blocking, Retired) → "Replaced by {label}" / "No replacement" → Evidence link |
| Register references | 22% | dl: Source, Owner reference (with hint "A register ID, not a person." in 14px muted, shown once in the th as a sub-line, not per row), Interfaces |
| State at the candidate build | 18% | Words |
| Implementation | 16% | Commit (12 chars, mono), or "No implementation recorded" |
| Tests | 20% | "{n} tests" / "{a} pass recorded · {b} fail recorded · {c} no result" (one line per count at M) |

- **Action, at every width:** in `tr.p05-follow`, never as a column.
  - Contents: the opener `button.live-secondary.p05-opener` "Propose retirement" (44px, padding-inline 16px, nowrap), or the pending sentence with its link, or the blocker sentence (16px, 68ch).
  - The follow row is indented to align with the first column's text (padding-inline 0).
- **Why action is never a column:** blocker sentences such as "This requirement is common, so it needs a replacement, and no requirement can replace it now." run to 9 lines in a 20% column. They also need the 68ch measure, and the form opens in the same place.
- **S:** stacked labelled rows; the follow row comes last.

## 6.3 Test artifacts

**Order:** lead → `dl.p05-nc-strip` → `.p05-regs` → h3 "Results" → results table.

- **`dl.p05-nc-strip`:** 4 items (Evidence files, Logs, Execution environment, Evidence content: "Not connected").
  - `repeat(4,1fr)` at ≥1080, `repeat(2,1fr)` at ≥720, one column at S.
  - dt 14px muted, dd 16px muted "Not connected". No borders.
- **`.p05-regs`:** a grid of two `section.p05-reg`, Builds and Harnesses.
  - `repeat(2,minmax(0,1fr))`, gap 48px, at `.p05-live` ≥880, so both 1440 and 1280. Stacked below that.
  - Builds table: Build 30 | Commit 50 (wraps in its own cell, 40 hex) | Order 20 (end-aligned) "Registered {order}".
  - Harnesses table: Register 30 | ID 45 | Revocation 25 "Not connected".
  - These are two sibling sections, not two tables inside one panel. They are small registers. Each is its own container and stacks under 720px.
- **Results (L and M):**

| Col | Width | Content |
| --- | --- | --- |
| Test | 20% | Test ID mono 600, namespace 14px sub-line |
| Build | 9% | Mono |
| Outcome | 12% | "Pass recorded" / "Fail recorded" as plain text, not tags |
| Counted | 22% | "Counted" / "Not counted" + reason sentence in 14px |
| Passes | 8% | Number, end-aligned |
| Failures | 8% | Number, end-aligned |
| Harness | 10% | Mono |
| Recorded at | 11% | Two nowrap parts |

  Follow row: "Evidence digest" dt (14px muted) + 64 hex mono 14px, `overflow-wrap:anywhere`. On one line at L and M; it wraps at S.
- **S:** stacked rows; the digest is the last labelled item.

## 6.4 Load profiles

**Order:**
1. lead;
2. `.p05-notes-block` with three 14px paragraphs: the planning targets, the SCALE-11 sentence, "Real load measurement: Not connected.";
3. `section.p05-readiness`;
4. `section.p05-hw`;
5. one `article.p05-profile` per profile.

- **Readiness:**
  - h3 "Load gate at build {build}: not ready."
  - Table: Gate 20 | Status 16 | Profiles 44 | Retired profiles (not counted) 20.
  - The Profiles cell lists one line per profile: `{profile}` · status words · "Run #{n}", or "No run at this build" when the status is not_run · "Failures since budget approval: Not connected" (never a failure count; root amendment, V1/M2).
  - Stacked at S.
- **Hardware:**
  - `dl.p05-strip` with ID, CPUs, Memory, Disk, Storage: `repeat(5,1fr)` at L, 3 columns at M, 2 at S.
  - Values 16px/600 tabular, not 20px. They are synthetic, so they are not given P04 strip prominence.
  - The spec digest follows as a `p05-sum-wide` row.
- **Profile article:**
  - `padding-block:32px`, `border-block-start:1px solid var(--border-row)`. No rule above the first article directly after the hardware heading.
  - `.p05-head` (flex wrap, baseline, gap 8px 16px): h3 profile ID mono 16px/600 · gate words · status words · "Retired" tag.
  - **Body:** `.p05-profile-grid`, `minmax(0,5fr) minmax(0,7fr)`, gap 48px, at article container ≥1080 (1440). One column at 1280 and below.
    - **Left:** `dl.p05-sum` with Status at the candidate build, Dataset digest (wide), Expected digests (wide). Then h4 "Active budget" and `dl.p05-sum`: state sentence (600 for "Proposed, not approved"), each limit with unit, Proposer, "Registration approved by" (perfgate Approver), and "Budget approval: Not connected" in place of Approval by / Approved by (root amendment, V1/M4).
    - **Right:** h4 "Axes" and `dl.p05-axes`.
      - Columns: `repeat(4,1fr)` at L, `repeat(3,1fr)` at ≥720, `repeat(2,1fr)` at S.
      - Each item: dt 14px muted (label with unit), dd 16px/600 tabular; a present 0 reads "0".
  - **Runs:** h4 "Runs", then a core table.

| Col | Width |
| --- | --- |
| Run | 8% (end-aligned number) |
| Build | 12% |
| Verdict | 26% (words; 600 for every non-provisional verdict) |
| Rows | 14% (end-aligned) |
| Duration | 12% "{n} s" |
| Recorded at | 14% |
| Harness | 14% |

    Follow row per run: `dl.p05-metrics` with Hardware, Mismatches, Interrupted, and the nine metrics.
    - `grid-template-columns:repeat(auto-fill,minmax(180px,1fr))`, gap 8px 24px, dt 14px muted, dd 16px tabular.
    - `null` reads "Not measured" (muted). A missing key reads "Unknown".
    - The client computes no budget comparison and adds no "over limit" marker beyond the server's verdict words.
  - **Why core columns plus a follow row:** 19 run fields cannot form a table at 960 or 1120 without 60px columns. P04's stacked-at-M runs produced 5,000px pages at 1280 (HB-289 round 1), so the table stays a table at M.
  - **Action:** `tr`-equivalent `.p05-actions` at the end of the article, the same as §6.2.

## 6.5 Decision history

**Order:**
1. lead;
2. `dl.p05-strip` with Support decisions, Activation review, Ledger: `repeat(3,1fr)` at ≥720, 1 column at S, values 16px;
3. h3 "Waiting for approval ({n})" and `ol.p05-proposals`;
4. h3 "Closed ({n})" and `ol.p05-proposals`.

Splitting pending and closed under two headings makes the plan's "pending first" visible.

- **`li.p05-proposal`:**
  - `border-block-start:1px solid var(--border-row)`, padding-block 16px.
  - Grid at ≥720: `minmax(220px,28%) minmax(0,1fr)`, gap 8px 32px. One column at S.
  - **Meta column:**
    - register words (14px muted);
    - h4 "#{number}" (mono 16px/600);
    - kind words;
    - state tag;
    - proposer;
    - created time (two nowrap parts).
  - **Body column:**
    - summary (16px, 68ch, placeholders isolated);
    - "Reason:" + `span.p05-free` (pending rows);
    - closing facts (14px);
    - the "Applied means recorded…" sentence (14px muted) on applied rows only;
    - actions: `div.p05-act` (flex wrap, gap 12px) with Approve and/or Withdraw as `live-secondary` 44px buttons, then the blocker or state sentence (68ch).
- **Selected (`proposal=` from the URL):** `background:var(--surface-hover)` plus a `.p05-tag` "Selected" with `aria-hidden`; `aria-current` carries the meaning.

---

# 7. State presentation

| State | Visual | Never |
| --- | --- | --- |
| Real value / real 0 | Ink, 16px, tabular | Muted, blank, "–" |
| Not evaluated (no candidate) | `.p05-absent`: `color:var(--text-muted)`, regular, full word "Not evaluated" (in a narrow cell; the full sentence is in the lead or candidate line) | "Unknown", 0, blank |
| Unknown / Unknown (code) | `.p05-absent` + code in mono 14px inside its own ltr isolate | Blank |
| Not connected | `.p05-absent` "Not connected" | "None", "0", a disabled button |
| Not measured (metric null) | `.p05-absent` | 0 |
| Not run | Words "Not run" in ink (it is a state, not an absence) plus its reason sentence | "Pending", a clock icon |
| Not counted | "Not counted" ink + reason sentence 14px | Strike-through, grey-out of the counts |
| Pass recorded / Fail recorded | Plain text, ink, regular (Fail recorded 600 when it causes a problem) | Tags, ticks, crosses, green or red |
| Failing / Over budget / other failing verdicts / problems | Ink, 600 | `--hb-danger` colour, red tag |
| Provisional | Ink, regular + its qualifier sentence | Amber |
| Unexpected value (rule 1) | `p.p05-unexpected`: ink, 600, full sentence, in place of the state word; code in mono | Showing the positive value in any form |
| Blocked lead (matrix) | Lead paragraph, first sentence 600 | Banner, coloured box |
| Pending proposal | `.p05-tag` "Pending" | Amber chip |
| Applied | Plain text "Applied" (not a tag) + the "recorded, releases nothing" sentence | Success styling |
| Retired / Common / Blocking | `.p05-tag` | — |

`.p05-tag`:
```
display:inline-block; padding:0 6px; border:1px solid var(--border-control);
font-size:14px; line-height:22px; font-weight:600; color:var(--hb-ink); white-space:nowrap;
```
Inside td it wraps, as P04 does. Use tags only for register flags and the proposal state. Never put a sample result in a tag.

---

# 8. Command forms and confirms

- **Location:** forms open in the row's follow row (requirements), in the article's `.p05-actions` (profiles) or in the proposal body (decisions). An open form hides its opener and siblings.
- **Container:** `.p05-form-wrap` (`max-inline-size:none`). Inside, `form.p05-form` is a grid with gap 16px and `justify-items:start`.
- **Order:**
  1. `h3.p05-form-title` (16px/600, margin 16px 0 0), e.g. "Propose retirement of sample/sample-approval";
  2. the error summary (shared `service-summary`, focused on a client failure);
  3. fields, each `service-field`: label 16px/600 → hint 14px muted (between label and control, GOV.UK order) → control → inline error;
  4. `div.live-buttons`: "Review" (`live-primary`, the only primary in this step) and "Cancel" (`live-secondary`).
- **Fields:**
  - Retirement reason: `input`, `inline-size:100%; max-inline-size:440px; min-block-size:44px`.
  - Replacement: a native `select` with the same caps, 44px high. Its placeholder option is "Choose a replacement" and nothing is pre-selected.
  - Reason for this action: `textarea` with 4 rows and the same width cap. Show a byte counter as a 14px muted line under the field: "{n} of 256 bytes".
- **Invalid fields:** `border:2px solid var(--hb-ink)` (the shared pattern) plus the error text. No red border alone.
- **Confirm (`section.p05-confirm`, gap 16px):**
  1. `h3.p05-confirm-title` (focused);
  2. `dl.live-facts` with 2 columns at ≥720 of its container, `max-inline-size:880px`, digests and replacements spanning both;
  3. `div.p05-impact` (68ch) with the effect sentences as separate paragraphs. The permanence sentence ("Approval retires … permanently …") is weight 600, and so are the "does not mean the load passed" and "does not mean its tests passed" sentences;
  4. `div.live-buttons`: "Confirm" (`live-primary`, the only primary) and "Cancel".
  - The resend variant adds `p.p05-repeat` ("This repeats your earlier request exactly as it was sent.") above the facts.
- **Row note (`.p05-row-note`, `tabindex=-1`, `outline-offset:2px`):** paragraphs at 68ch, with the pending-attempt sentence first.
- **Width rules:**
  - ≥768px viewport: buttons are content width with a 44px minimum height and padding-inline 20px, so they never stretch across 1120px.
  - <768px viewport: `.live-buttons` stacks and buttons take 100% width. Inputs use `inline-size:calc(100% - 6px); margin-inline:3px` so the focus ring is not clipped (the P04 fix).

---

# 9. Focus and keyboard visuals

- Rely on the global `:focus-visible` rule (2px `--hb-focus`, offset 2px). Do not override its colour, and do not add a persistent outline to selected rows.
- **Programmatic focus targets** (`tabindex=-1`): `h2.p05-detail-title`, `#p05-notice`, `.p05-row-note`, `h3.p05-confirm-title`, the error summary.
  - Their `:focus-visible` ring must be visible after keyboard-initiated moves.
  - Add `:focus:not(:focus-visible){outline:none}` only on these headings and notes, never on controls.
- **Sticky panel scroll container** (`aside.p05-detail[tabindex=0]`): `outline-offset:-2px`.
- **Selected list row:** `tbody.p05-row[aria-current]`.
  - `background:var(--surface-hover)`, plus the visible `.p05-tag` "Selected" (`aria-hidden`) after the row title.
  - This is a non-colour indicator, as required by service-screen-style.md. The Evidence link keeps its text so focus can return to it.
- **Links in table cells** (`.p05-link`, `.p05-evidence`): `display:inline-flex; align-items:flex-start; min-block-size:44px; text-decoration:underline; text-underline-offset:4px; color:var(--hb-ink)`.
  - Nothing else signals a link, so links have no colour change. This matches P04.
- **Escape** has no visual of its own. Focus returns to the row's Evidence link and the ring shows there.

---

# 10. RTL and multilingual rules

- **Logical properties only** (`inline-start/end`, `block-*`). No `left`/`right`, `float` or `text-align:left`. The panel's border is `border-inline-start`, so it is on the panel's leading edge in both directions.
- **ltr isolates:**
  - IDs, labels, paths, digests, commits, builds, targets and codes are `<bdi dir="ltr">`, each with the `.live-mono` class where monospace.
  - Use one isolate per compound value, e.g. "sample/sample-erp-template", or `path>ns/id` as a single isolate.
  - Separators (" · ", "of", commas) stay outside isolates in locale text.
- **Free text** (rationales, retirement reasons, closing reasons): `span.p05-free[dir=auto]` with `unicode-bidi:isolate; overflow-wrap:anywhere`. **Never `<bdi>`**, because the global `bdi{direction:ltr}` forces Arabic left-to-right.
  - Do not place free text inside any element that has `.live-mono` or a `bdi` ancestor.
- **Dates:** two nowrap parts (date | time), each FSI-isolated, as in P04's Stamp. Whether ar-AE renders Latin or Arabic-Indic digits follows the existing Intl helper. UNVERIFIED; record it in the render pass.
- **"{n} of {m}":** a whole localized sentence template per locale (AR word order), with numbers inside it. Do not assemble it from fragments.
- **No direction glyphs.** No →, ←, ✓ or ✗ in any locale (design-system.md bans Unicode symbol icons). The build comparison uses table columns (§6.2).
- **Typography:**
  - Arabic line-height 1.8 (the console override); do not set line-height in `.p05-*` for running text.
  - KO uses `word-break:keep-all` for prose and headings.
  - No italics in any locale.
- **Numbers in RTL:** end-aligned in columns (which means visually left in AR); `tabular-nums`.
- **Shell tabs at <768px** (five tabs): `.planning-main:has(.p05-live) .planning-tabs` is a 2-column grid, as in P04, plus `> :last-child:nth-child(odd){grid-column:1/-1}` so "Decision history" is not orphaned at half width. 44px min height, text start-aligned. The active tab uses the shell's neutral indicator; P05 adds none.

---

# 11. ID wrapping

- Requirement labels `namespace/id`: insert `<wbr>` after "/" (inside the isolate). Segments do not break at hyphens at L of the nearest container.
- Path IDs and test IDs: nowrap at L of the nearest container, `overflow-wrap:anywhere` below.
- Digests (64 hex) and commits: `overflow-wrap:anywhere` everywhere, inside their own fact row or follow row. Never in a narrow column.

---

# 12. Tokens and class names

**Tokens (existing only):**
- `--hb-ink`, `--text-muted`;
- `--border-row`, `--border-control`;
- `--surface-page`, `--surface-hover`, `--surface-primary`, `--text-on-primary`;
- `--hb-focus`;
- `--hb-font-mono`;
- `--hb-danger` (only through the shared `.live-error` heading).

No colour literals, shadows, radii, gradients or new tokens.

**Type:**
- tab h2 20/600; h3 16/600; h4 14/600;
- body 16; small and labels 14; mono 14 (16 in headings);
- prose `max-inline-size:68ch`.

**Spacing:** 4 / 8 / 12 / 16 / 24 / 32 / 48. Sections are separated by 32px with a 1px `--border-row` top rule, and there is no rule above the first block after a heading or lead.

**Classes (p05 prefix; reuse shared `live-*` base classes; no `p04-` names):**
- Frame and states: `p05-live`, `p05-scope`, `p05-scope-facts`, `p05-frame`, `p05-frame-main`, `p05-frame-notes`, `p05-statement`, `p05-candidate`, `p05-note`, `p05-lead`, `p05-absent`, `p05-unexpected`, `p05-tag`, `p05-tags`, `p05-free`, `p05-stamp`, `p05-nobr`, `p05-num`.
- Tables and links: `p05-table`, `p05-row`, `p05-follow`, `p05-reduced`, `p05-link`, `p05-evidence`, `p05-back`, `p05-close`.
- Split and detail: `p05-split`, `p05-list`, `p05-detail`, `p05-step`, `p05-detail-head`, `p05-detail-title`, `p05-sum`, `p05-sum-wide`, `p05-inline`, `p05-cats`, `p05-test`.
- Matrix: `p05-combo`, `p05-problems`, `p05-nc`, `p05-nc-strip`.
- Requirements: `p05-gate-row`, `p05-gate`, `p05-compare`, `p05-pair`.
- Artifacts and load: `p05-regs`, `p05-reg`, `p05-readiness`, `p05-hw`, `p05-strip`, `p05-profile`, `p05-profile-grid`, `p05-head`, `p05-axes`, `p05-metrics`, `p05-actions`.
- Decisions: `p05-proposals`, `p05-proposal`, `p05-meta`, `p05-body`, `p05-act`.
- Commands: `p05-form-wrap`, `p05-form`, `p05-form-title`, `p05-confirm`, `p05-confirm-title`, `p05-impact`, `p05-repeat`, `p05-row-note`, `p05-opener`, `p05-vanished`; `#p05-notice`.
- Column helpers: `p05-c{n}` (only the percentages listed above).

Every rule is scoped under `.p05-live`, except the single shell-tab `:has()` override.

---

# 13. Must never appear

- **Coloured borders and strips:**
  - coloured edge strips or `border-inline-start` accents on notices, panels, rows, tabs or the detail panel (the panel divider is 1px `--border-row` only);
  - coloured selection outlines;
  - the wireframe's blue tab underline, chips or rounded cards.
- **Positive signals:**
  - green, ticks or check badges;
  - "Passed", "Ready", "Active", "Supported", "Released" or "Customer-proven" as a state, in any language;
  - a positive server value rendered even partially (e.g. "passed" next to the unexpected sentence).
- **Fill-style counts:** progress bars, meters, percentages, donuts, sparklines or traffic lights, including for "{n} of {m}" counts or axes.
- **Buttons where nothing is connected:**
  - Buttons, or disabled buttons, for not-connected actions (Link evidence, Record support decision, Request activation review). These are text only.
  - Approve or Withdraw buttons on requirement or profile rows, in the detail panel, or on non-retire proposals.
- **Too many or too wide primaries:**
  - more than one primary button in any form step or confirm;
  - buttons stretched across the container at ≥768px viewport.
- **Wrong data presentation:**
  - an empty selection panel occupying 38–40% of the width (see §4.2);
  - blank cells, "—" or "0" standing for absent values;
  - truncated digests with an ellipsis;
  - free text inside `<bdi>`;
  - harness or runner identities.
- **Layout faults:** horizontal page scroll at any tested size; overlapping columns.

---

# 14. Problems found in the plan (layout and states)

Historical (root, 2026-11-18): these items describe the plan before its revisions; the plan now carries the fixes and wins on wording.

1. **Split list too narrow (High).** At 1440 the list region beside a 40% panel is about 640–660px, under the 720px table threshold. The plan's rule would flip the matrix and requirements tables into stacked cards whenever a row opens. Fix: the reduced column set (§4.3), rendered by state.
2. **Detail content gap (High).** Plan decision 4's matrix detail lists only Support since, categories, reconciliations and not-connected items. The requirements detail omits source, owner reference and interfaces. With a reduced list, the hidden columns would be visible nowhere. The panel must repeat them (§4.3).
3. **Arrow in the build comparison (Medium).** "{from state} → {to state}" as one isolate is ambiguous in RTL and conflicts with design-system.md's ban on Unicode symbols. Fix: a three-column result table (§6.2).
4. **Customer-proof sentence repeated per cell (Medium, root decision).** Rule 6 puts "Customer proof intake is not connected, so this count cannot rise in this build." in every row of a 10% column, about 8 lines per cell. Proposal: state the full sentence once, as a column-header sub-line (th, 14px regular) and in the panel, and use the short "Intake not connected" in each cell. This still satisfies "the column always says". Root should confirm this reading of the binding rule.
5. **Split threshold (Medium, root decision).** At 1080, a 1280 desktop gets the mobile-style step, against spec.md:109 "Desktop: side by side". Proposal: 940px (§4.1).
6. **Sticky panel height (Medium).** A sticky panel taller than the viewport hides its own bottom content. The plan specifies sticky without a height cap or keyboard scrolling. Fix in §4.2. The shell header offset is UNVERIFIED.
7. **Step-mode content not specified (Low).** The plan does not say whether the lead, gate block and compare control render with the detail in step mode. Decided in §4.4: they do not.
8. **Action column (Low).** Requirement and profile actions are in-row cells in the plan. Long blocker sentences need the 68ch follow row (§6.2).
9. **Runs table (Medium).** "Capped and stacked" with 19 fields repeats the HB-289 5,000px-stack failure at M. Fix: core columns plus a metric follow row (§6.4).
10. **IDs nowrap "only at L" (Low).** This must key on the nearest container. Labels also need a `<wbr>` after "/", or 26-character labels overflow 15% columns (§11).
11. **Selected-row indicator (Low).** `aria-current` with a background tint only is colour-only. Add the visible "Selected" tag (§9).
12. **Empty panel text (Low).** Drawing an empty 426px aside wastes width. Use a one-line hint instead (§4.2).
13. **Five shell tabs at 390 (Low).** The P04 2-column clone orphans the fifth tab. Fix in §10.
14. **Not checked:** I did not check the reasonableness of the copy or the terminology. That belongs to the tech-writer. The KO tab label "시험 artifact" mixes scripts, which is noted only.

---

# 15. External references

Superseded (root, 2026-11-18): root opened these references; the binding record is [`references.md`](references.md), as transcribed into the HYB-315 plan. The "unverified" marks below are historical.

I had no browsing tool, so **every reference is UNVERIFIED (not opened).** The plan's gate ("no screen code before they are recorded") is therefore **not satisfied by this spec**. Root must open each one and record the observed pattern, or replace it.

| Reference | Pattern relied on | Adopt | Reject |
| --- | --- | --- | --- |
| GOV.UK Design System: Summary list, https://design-system.service.gov.uk/components/summary-list/ (unverified) | Key–value rows for facts | `dl.p05-sum` shape; key 38% / value | Row borders between every pair; "Change" links |
| GOV.UK: Error summary and hint placement, https://design-system.service.gov.uk/components/error-summary/ (unverified) | Summary at top linking to fields; hint between label and input | Form order in §8 | Red left border on the summary (conflicts with the owner's no-accent rule) |
| Material Design 3: List–detail canonical layout, https://m3.material.io/foundations/layout/canonical-layouts/list-detail (unverified) | Two panes at expanded width, single pane with navigation at compact width | Split and step model, reduced list beside the detail | Rounded surfaces, elevation, colour-filled selection |
| Azure DevOps: Requirements traceability, https://learn.microsoft.com/en-us/azure/devops/pipelines/test/requirements-traceability (unverified) | Requirement → tests with outcome counts, including not-run | Tests summary as separate counts; not-run as its own state | Green/red outcome colouring, pass-percentage bars |
| GitLab: Release evidence, https://docs.gitlab.com/ee/user/project/releases/release_evidence.html (unverified) | Evidence collected separately from the release | The statement that evidence is not release; the digest as identity | Any "release" affordance |
| Grafana k6: Thresholds, https://grafana.com/docs/k6/latest/using-k6/thresholds/ (unverified) | Per-metric threshold, verdict per run | Metric list with units under each run; verdict words | Pass/fail tick output, coloured threshold results |

---

# 16. Render-comparison checklist for root QA

Record per render: viewport, locale, tab, measured `.p05-live`, `.p05-list` and `.p05-detail` widths, and the container label.

1. **Widths:**
   - `.p05-live` is about 1120 at 1440, about 960 at 1280 and about 358 at 390.
   - The split appears at the agreed threshold (940 or 1080), on both the JS and CSS side.
   - `document.scrollingElement.scrollWidth <= clientWidth` on every render.
2. **Matrix with no selection:**
   - 1440: 8 columns. 1280: 7 columns, with Last observed under Test.
   - No th text overflows its cell. Inspect the "Implementation" header in EN.
   - 390: stacked labelled rows; the Evidence link is 44px.
3. **Matrix split:**
   - Opening `sample-path-xlsx` at 1440 gives the reduced 4-column list. The table does not turn into stacked cards.
   - The panel is at inline-end (left in AR), with a 1px `--border-row` divider. Check its computed colour equals the row-border token.
   - The panel shows design 2 of 2, implementation 1 of 2, test 1 of 2, 0 failing, customer proof 0 of 2 with the intake sentence, four blockers, six categories, "Not matched, 3 differences" and "Support since: Not connected".
   - No buttons in the panel.
4. **Sticky panel:** scroll the list. The panel stays in view and its bottom content is reachable by mouse wheel and keyboard (Tab to the aside, then the arrow keys).
5. **Selected row:** `aria-current` present; background `--surface-hover`; the visible "Selected" tag; no outline other than transient focus.
6. **Step mode at 390** (and at 1280 if 1080 is kept):
   - The list, gate block and compare control are absent from the DOM with a selection.
   - "Back to the list" is above the h2.
   - Focus returns to the same Evidence link.
7. **Requirements:**
   - The gate block and compare control sit side by side at 1440 and stack at 1280 and 390.
   - The compare result is a three-column table, with no arrow glyph.
   - Action sentences sit in follow rows within 68ch.
   - Labels break only after "/".
8. **Artifacts:**
   - The four "Not connected" items as text.
   - Builds and Harnesses side by side at 1440 and 1280.
   - Each digest is on one line at L and M and wraps at 390, with no ellipsis.
   - The `t-xlsx-amount@0.0.1` not-counted reason.
9. **Load:**
   - The runs table has 7 core columns at 1440 and 1280, with metrics in a follow-row grid. Null metrics read "Not measured".
   - Axes in 4, 3 and 2 columns. Hardware values 16px, not 20px.
   - Page height at 1280 recorded. Flag anything over about 2× the 1440 height.
10. **Decisions:**
    - "Waiting for approval" and "Closed" headings.
    - The meta and body grid at ≥720.
    - Approve and Withdraw are content-width secondary buttons at 1440 and 1280, and full width at 390.
    - "Applied" is not a tag.
11. **Confirms:**
    - Exactly one `.live-primary` in each form step and each confirm.
    - The permanence sentence at weight 600.
    - Fields at most 440px wide at 1440.
    - The focus ring is unclipped at 390 (3px inset).
12. **Computed-style sweep:**
    - Every `border-*-color` on `.p05-live *` resolves to `--border-row`, `--border-control` or `--hb-ink` (invalid fields only).
    - No `border-inline-start` wider than 1px.
    - No `border-radius` other than 0, no `box-shadow`, no background other than `--surface-page`, `--surface-hover` or `--surface-primary` (primary button).
13. **Text sweep:** in EN, KO and AR there are no positive state words. There are no ✓ ✗ → ← characters inside `.p05-live`.
14. **RTL (AR):**
    - Free text is in `span[dir=auto]` and not inside `bdi`.
    - IDs read correctly as ltr isolates.
    - Dates are not split inside the year.
    - The panel is on the left.
    - Line-height is 1.8.
    - The scope facts and tags mirror correctly.
15. **KO:** `keep-all` on headings and prose; no mid-word breaks in the tab heading or lead.
16. **Focus:**
    - A visible 2px `--hb-focus` ring on the detail heading after keyboard open, on `#p05-notice` after success, on the row note after refusal and on the confirm heading.
    - No ring after URL-driven selection.
17. **Shell tabs at 390:** a 2-column grid, with the fifth tab spanning both columns.
18. **Not verified by this checklist:** owner acceptance, terminology correctness and real-data behaviour.

---

Uncertainties to resolve before implementation:
- Root decisions on §14 items 4 (the customer-proof sentence per cell) and 5 (the 940px split threshold).
- Whether the shell header is sticky, which sets the panel offset.
- The ar-AE digit system.
- The actual th widths in EN.
- All external references are unopened.

Files referenced:
- /private/tmp/hb315/docs/features/HYB-315/plan.md
- /Users/home/project/hybrid_billing/docs/ui/screens/P05/spec.md
- /Users/home/project/hybrid_billing/docs/ui/screens/P05/design.json
- /Users/home/project/hybrid_billing/docs/ui/screens/P05/wireframe.svg
- /Users/home/project/hybrid_billing/.agents/rules/service-screen-style.md
- /Users/home/project/hybrid_billing/docs/design-system.md
- /Users/home/project/hybrid_billing/apps/console/src/modules/administration/pages/retention/live.css
- /Users/home/project/hybrid_billing/apps/console/src/modules/identity/pages/users/live.css
- /Users/home/project/hybrid_billing/apps/console/src/shared/styles.css
- /Users/home/project/hybrid_billing/apps/console/src/modules/administration/pages/release-evidence/Screen.tsx
- /private/tmp/hb315/docs/features/HB-289/verification.md