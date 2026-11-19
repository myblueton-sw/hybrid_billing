---
type: DesignSpecification
title: HYB-310 P02 integration rights visual specification
description: Layout, hierarchy, states and draft EN copy for the P02 integration-rights changes. Covers the rights line, withheld areas, availability reasons, the closed-on-approval outcome, your pending items with withdraw, held redeliveries and re-approval, the no_right and no_access outcomes, the admin3 stop-only view, RTL, the 1440/1280/1024/390 widths, keyboard and announcements, and rendered-QA checks. Written by the ui-designer role (read-only). The HYB-310 plan wins on copy, states and truthfulness; this spec wins on layout.
status: draft
tags: [HYB-310, P02, permissions, visual-spec]
sources:
  - resource: plan.md
  - resource: ../../ui/screens/P02/spec.md
  - resource: ../../ui/terminology/product-glossary.md
---

# HYB-310 visual specification: P02 integration rights

**Author:** ui-designer role, read-only. The model is Claude Opus 5.5 (`claude-opus-5-5`).

**Status:** this is a design proposal. It is not owner acceptance, and a later visual-review PASS would not be owner acceptance either.

**Precedence:** where this spec and `docs/features/HYB-310/plan.md` (after security review round 3) differ on copy, states or truthfulness, the plan wins. Copy marked **[TW]** is either new copy that the layout needs or plan copy that this spec questions. The technical-writing review supplies or approves the final text. KO and AR are not drafted here, except where the plan already gives them.

---

## 0. Evidence basis and limits

**Read directly:**
- `docs/features/HYB-310/plan.md`, the whole file. This includes Console (P02), Read, Author withdrawal, Starter recheck, Grants and Browser QA.
- `docs/ui/screens/P02/spec.md`, the whole file, especially :23, :33, :37, :70-76, :100, :102 and :116.
- `.agents/rules/service-screen-style.md`, `usability.md` and `screen-review.md`.
- Console source:
  - `apps/console/src/modules/administration/pages/integrations/Live.tsx`, `live.css` and `live-copy.ts`;
  - `LiveCommand.tsx:214-245` (`RowNote`, `blockerLines`);
  - `apps/console/src/service/service.css:130-136` (`.service-status`: a full 1px neutral `--border-control` box, not an edge strip);
  - `apps/console/src/app/service-shell.css:35, 77, 135-177`. The rail is 256px with 32px main padding. The rail stacks on top at ≤1279px, and main padding is 16px at ≤767px.
- Glossary rows `bulk-revocation-right` (:81) and `trust-rights` (:242), the precedent for wording right groups.
- Format precedent: `docs/features/HYB-297/visual-spec.md`.

**Screenshots inspected (current P02 build, before HYB-310):**
- `reg309shots/p02/render-clients-en-1440.png`, `render-modules-en-1440.png`, `render-channels-en-1440.png`, `render-events-en-1440.png`;
- `render-channels-ar-390.png`;
- `flow-D-member-denied-en-1440.png`;
- `hb267shots/_access_lang_en_view_grants_1440.png`. This is an older M03 screen that predates the live rights display. It gives no evidence for an M03 rights list.

**Not available or not done:**
- **External references:** I had no browsing tool and opened none. `service-screen-style.md` requires recorded external references for layout revisions. This change adds no new layout pattern: it reuses P02's existing row-note, notice, denied and list patterns. Root must still record that judgement, or the references, before calling the screen compliant. Reference status: UNVERIFIED.
- **frontend-design skill:** not supplied in this task, so not read.
- **Not read:** `.agents/rules/design-system.md`, `docs/design-system.md`, `live-model.ts`, `live-codes.ts`, `LiveModules.tsx`, `LiveProposals.tsx`, `LiveChannels.tsx`, `LiveEvents.tsx`, `LiveClients.tsx`, and the M03 live access page and its labels. "M03 consistency" below rests only on the plan's statement that M03 shows no integration right.
- No KO screenshot was inspected. No 1024 or 1280 renders exist.

**Container widths** (inferred from `service-shell.css`, not measured):

| Viewport | Rail | Content width |
| --- | --- | --- |
| 1440 | 256px, side | 1120px. Matches the screenshot: content runs x=288–1408. |
| 1280 | 256px, side | 960px |
| 1024 | stacked on top | 960px |
| 390 | stacked on top | 358px |

### Current-state observations (from the screenshots and source)

1. **Page spine:** h1 → review strip → tabs (Clients · Modules · Channels · Events) → scope bar (Workspace ID · Data revision · Now (UTC) · Reload, inline-end aligned) → statement ("**Records decisions only.** …", 16px) → permission line (14px muted, max 68ch) → view h2 → area lead → area sections.
2. **The deviation line:** "Every command here currently needs installation administration…" sits at `Live.tsx:137` (`.p02-permission`), 14px muted, 24px below. This is the slot the rights line replaces.
3. **Unavailable actions are already expressed as text, not as disabled buttons.** For example, on Modules proposal 9: "**Approve:** You proposed this. Another administrator must approve it." On Events: "**Approve redelivery:** …" / "**Request redelivery:** …" / "**Approve dismissal:** …". These are `RowNote` → `ul.p02-blocked`: a neutral top rule per item, 8px padding, bold action label and colon. **This is the pattern for every new reason in this spec.**
4. **The denied block** (member): it keeps the tab strip; below it are h2 "Integrations are not available to you" and one paragraph. No scope bar.
5. **No decorative edges anywhere:** tags are 1px neutral boxes, the notice is a full neutral box, selected tabs are underline and weight. Primary buttons are solid dark, secondary buttons are outlined.
6. **At 390 the tabs become a 2×2 grid** (`live.css:618-629`); actions stack as full-width buttons. In AR, mono IDs stay LTR inside RTL text.

---

## 1. Composition principles

1. **What you can do comes before what exists** (`usability.md:3`). The caller's integration rights are stated once, high on the page, before any area content. Every unavailable action then says why in the row where the action would be.
2. **Withheld is a fourth state, not empty, zero or an error.** A withheld area shows one sentence and nothing derived from the area. It must look different from "No clients." (empty), from 0, and from the error block.
3. **One reason per unavailable action.** The first reason in plan order wins: `no_right` → `requester_lost_right` → state reason. Never stack two reasons for one action.
4. **No new visual vocabulary.** No icons (the icon set has only chevron and close), no lock badges, no tinted panels, no colored edges. Hierarchy comes from position, weight, size and neutral rules.
5. **Withholding is a display boundary.** Nothing on screen may suggest secrecy or confidentiality (plan F8).

---

## 2. Implementation item IDs (screen-scoped, stable)

| ID | Item |
| --- | --- |
| `p02:status:rights` | Rights line (held rights) |
| `p02:status:rights-none` | Rights line (no integration right) |
| `p02:status:rights-unknown` | Rights line (the read has no `rights`) **[new; see §4.1]** |
| `p02:state:area-withheld` | Withheld panel per tab |
| `p02:list:own-pending` | "Your pending items" inside a withheld tab |
| `p02:action:withdraw-own` | Withdraw on own pending rows (in-row and in-list) |
| `p02:reason:no-right` | Per-action `no_right` line |
| `p02:reason:requester-lost-right` | Approve-action line when the starter has lost the right |
| `p02:outcome:closed-lost-right` | Approval that closed the entry |
| `p02:state:redelivery-held` | Held redelivery row |
| `p02:action:redelivery-reapprove` | "Approve again" and its confirm step |
| `p02:outcome:no-right` | Refusal after reload, right lost |
| `p02:outcome:no-access` | Reload returns 403; switch to the denied block |

---

## 3. Page spine after HYB-310 (all tabs)

The order is unchanged except for one replaced line:

1. h1, review strip, tabs: unchanged. **Tab labels stay as they are on every tab, withheld or not.** No suffix, badge or icon. The panel explains.
2. Scope bar: unchanged.
3. Statement ("Records decisions only. …"): unchanged.
4. **Rights line** `p02:status:rights`: replaces `.p02-permission` in the same slot (§4.1).
5. `#p02-notice` (command outcome): unchanged slot and style.
6. Error block, if any.
7. View h2 (`#p02-view-title`, tabIndex −1): **always rendered**, including on withheld tabs.
8. Area content: either the normal area or the withheld panel (§4.2).

---

## 4. Components

### 4.1 Rights line — `p02:status:rights`, `-none`, `-unknown`

**Placement:** directly under the statement, in the slot of `.p02-permission`. It is page-level, so it shows on every tab with the same text.

**Markup:**
```
<p class="p02-rights" id="p02-rights">
  <strong>{lead}</strong> {list}.
</p>
```
- This is not a live region and not a heading.
- `{list}` is built with fixed separators over the held rights' **labels**, never their IDs: EN `; `, KO `, ` and AR `، ` (accepted copy-review correction).
- List order is fixed as in plan "Rights": API client administration; Module administration, Module approval, Module stop; Subscription and channel administration, approval, stop; Event reprocessing.

**Type and spacing:**

| Property | Value |
| --- | --- |
| Font size | 14px |
| Line height | inherit (as `.p02-permission`) |
| Lead weight | 600 |
| Text color | `--hb-ink` for the whole line, not muted. This line is now actionable information, so it outranks the old deviation note. |
| Measure | `max-inline-size: 68ch` |
| Margin | `margin-block: 0 24px` |
| Border, background | none |

**Copy:**

| State | EN (plan, unless marked) |
| --- | --- |
| Some held | "**Your integration rights:** {list}." |
| None held | "You hold no integration rights. Integration rights are given only when the installation is set up; they can't be granted here yet." In this state, `<strong>` wraps the first sentence. |
| `rights` missing or not an object | **[TW, new]** "Your integration rights could not be read, so no integration action is offered here. Reload to try again." |

**Why `-unknown`:** the plan treats a page without `rights` as "every right false". The none-held sentence would then claim "You hold no integration rights", which may be untrue. Unknown is not none (`usability.md:5`). The behaviour stays fail-closed: every right false, no action offered. Only the sentence differs.

**Expected length:** all eight held is about 230 characters, three lines at 68ch on 1440. admin3 is one line. That is acceptable; do not truncate or collapse it into a disclosure.

### 4.2 Withheld area — `p02:state:area-withheld`

Applies when `page[area].state === "not_permitted"`. The areas are clients, modules, channels and events. Channels and Events share the rule, so both tabs are withheld together.

**Render inside the panel, in this order and nothing else:**
1. View h2 (unchanged).
2. `<p class="p02-area-withheld">`: **"You don't hold a right for {area}. Nothing from this area is shown."** (plan). `{area}` uses the view-title words "API clients", "Modules", "Subscriptions and channels", "Events and redelivery", not the short tab labels.
3. If own pending items exist for this area: §4.6 "Your pending items".

**Must not render on a withheld tab:**
- the area lead (`clientsLead`, `modulesLead`, `channelsLead`, `eventsLead`);
- `readOnly`;
- the vanished-items list;
- the `unknown (state)` line;
- `sectionEmpty`;
- any h3, table, count, form opener or "not connected" note belonging to the area.

**Style:** this deliberately differs from empty and error.

| Property | Value |
| --- | --- |
| Font size, color | 16px, `--hb-ink` |
| Measure | `max-inline-size: 68ch` |
| Padding | `padding-block: 16px` |
| Rules | `border-block: 1px solid var(--border-row)`, neutral top and bottom |
| Margin | `margin-block: 0 32px` |
| Not allowed | background, tint, icon, inline-start border, `role="alert"` |

**Class name:** use the new class `p02-area-withheld`. Do **not** reuse `.p02-withheld` (`live.css:218`), which has another meaning.

**Semantics:**
- Plain paragraph, no live role; the tab change already moves focus to the h2 (§7).
- Add `data-state="not_permitted"` to the panel root for QA selectors.

### 4.3 Availability reason `no_right` — `p02:reason:no-right`

- **Button:** not rendered. Never a disabled button.
- **Reason line:** rendered through the existing `RowNote` → `ul.p02-blocked > li`, in the same slot and order where the button or state reason would be.
- **Line format:** "**{action label}:** Needs the {right label} right. You don't hold it." The text part is the plan's copy. The `{action label}` prefix follows the existing pattern.
  - Action labels are the existing button words: "Approve", "Withdraw", "Stop", "Disable", "Pause", "Revoke", "Request resume", "Approve resume", "Request redelivery", "Approve redelivery", "Request dismissal", "Approve dismissal", "Approve again".
  - For the two openers, the label is the opener text without the object: "Propose a change" and "Propose a subscription".
- **Right label:** sentence case inside the sentence, not bold and not quoted ("Needs the Module approval right."). **[TW]** Lower-case it mid-sentence ("the module approval right") if the TW review prefers. Apply one rule across EN.
- **Which actions get a line:** each action key that the read returns as `no_right` for that row, but only for keys the current console already renders on that row (as a button or a state reason). The console must not invent lines for action keys it never shows on that row type.
  - Example: on an active subscription the console today shows Pause and Revoke only (screenshot), so Approve gets no line there.
- **Openers withheld by `no_right`:**
  - Modules "Propose a change to {plugin}": the button is replaced by one `p02-blocked` line in the same place, under the installation facts.
  - Channels "Propose a subscription": the h3 stays, and the button is replaced by the line.
- **Order within a row:**
  1. The `RowNote` list first, with each unavailable action's single reason in the existing button order.
  2. Then the buttons the caller may use, in the existing order.
  3. Reasons are never interleaved between buttons. This matches the existing P02 composition (corrected after render review round 1; see §13 item 7).

### 4.4 Reason precedence (binding)

For each action, show exactly one line, from the first match:
1. `no_right`;
2. `requester_lost_right` (approve actions only);
3. the existing state reason (`blocker` words, `not_pending`, own-proposal and similar).

**No-right overlap:** when the reader lacks the right, the console must not also show the own-proposal sentence ("You proposed this…") for that action.

**Dry-run extra sentence:** `blockerExtra` (the unverified sentence) is appended only to state reasons, never to `no_right`.

### 4.5 Starter lost the right — `p02:reason:requester-lost-right`

- **Slot:** the approve button is not rendered. One `p02-blocked` line replaces it.
  - The action label is "Approve" (plugin or subscription), "Approve resume", "Approve redelivery", "Approve dismissal" or "Approve again".
- **Plan copy:** "The person who started this no longer holds the right. It can't be approved and will be closed."
- **[TW / root decision D1]:** "will be closed" is a promise the console cannot keep. With the approve button hidden, nothing in the console triggers the closure; only a stale page or a direct API call does.
  - **Default:** "**Approve:** The person who started this no longer holds the right, so it can't be approved."
  - Add the existing "Withdraw" button for the author when it applies; nothing else.
  - **Alternative for root:** keep the approve button and label it to close, which makes the closure reachable. That changes the plan's "not offered" rule, so it is root's call.
- **Row state tag:** unchanged ("Pending" or "Proposed").
- **Must not:** grey out, strike through or visually dim the row. The row stays fully readable.

### 4.6 Your pending items — `p02:list:own-pending`, `p02:action:withdraw-own`

**Where:** only inside a **withheld** tab (§4.2 item 3), filtered to that tab's area:
- plugin proposals → Modules;
- subscription proposals and resume requests → Channels;
- redelivery and dismissal requests → Events.

The API clients tab never has items. In a visible area, own items stay in their normal rows with an in-row Withdraw button; they are not duplicated.

**Structure:**
```
<section class="p02-own" aria-labelledby="p02-own-title">
  <h3 id="p02-own-title">Your pending items</h3>
  <p class="p02-small">{lead}</p>
  <ul class="p02-list">
    <li class="p02-proposal" id="p02-own-{request-id}">
      <div class="p02-meta">{kind label} · <bdi class="live-mono">{request or proposal ID}</bdi>
        <span class="p02-tag">Pending</span></div>
      <div class="p02-body">
        <p>{target line}</p>
        <button class="live-secondary" data-opener=…>Withdraw</button>
        {inline confirm when open}
      </div>
    </li>
  </ul>
</section>
```
- **Layout:** reuses `.p02-proposal`. At ≥1024 that is two columns (`minmax(220px, 28%) | 1fr`, 32px gap); below 1024 it is a single column.
- **Spacing:** `margin-block-start: 8px` after the withheld paragraph. The h3 is 16px/600, as `.p02-live h3`.
- **Copy:**
  - h3: **[TW]** "Your pending items".
  - Lead: **[TW]** "You can withdraw work you started, even without a right for this area."
  - Kind labels: **[TW]** "Plugin change proposal", "Subscription proposal", "Resume request", "Redelivery request", "Dismissal request".
  - Target line: **[TW]** "For {target}", with `{target}` in `bdi.live-mono`.
  - Show only what `own_pending` carries (kind, target, state, request ID). No dates, reasons, approvers or module facts, because the read does not supply them for withheld areas.
- **Withdraw button:**
  - Secondary style with auto width. A neutral outlined button, not destructive styling (withdrawal reduces risk).
  - Accessible name: "Withdraw {kind label} {ID}" (EN **[TW]**).
  - Uses the existing inline confirm (`.p02-confirm`): h3 "Confirm withdrawal", facts (kind, ID, target), the impact line, primary "Withdraw {kind} {ID}" and secondary "Back to edit". **Labelling the secondary "Cancel" here is a [TW] suggestion**; this confirm has no reason field.
  - Impact copy: **[TW]** "Withdrawing closes this request. Nothing else changes." For proposals, the existing `withdrawImpact` applies.
- **After success:**
  - `#p02-notice`: **[TW]** "{Kind label} {ID} withdrawn."
  - The item leaves the list after reload. If the list empties, remove the whole section; do not show an empty sentence.
- **One kind per item:** a redelivery request and a dismissal request on the same letter are two items with two IDs, each with its own Withdraw. A Withdraw never offers the other kind (plan N1).

### 4.7 Closed on approval — `p02:outcome:closed-lost-right`

**Trigger:** an approval returns business state `closed` with `reason: requester_lost_right`.

**Notice:**
- `#p02-notice` (existing `.service-status`, role=status, focused).
- Text: "{Object}: Not approved. It was closed because the person who started it no longer holds the right."
  - The sentence after the colon is plan copy. The `{Object}:` prefix is **[TW]**, for example "Proposal 8:", "Subscription sample-payments-feed:", "Dead letter delivery-3:".
- It must never contain "Nothing was changed".

**Row after reload:**
- **Plugin proposal:** moves to "Closed (n)" with state tag **[TW]** "Closed". This must be a different word from "Withdrawn" and "Applied". The body gains the existing "Closing reason" fact with **[TW]** "The person who started it no longer holds the right. Found when {approver} tried to approve (request {request ID})." The approver and request ID are `bdi` mono, LTR.
- **Subscription (proposed → closed):** state tag "Closed". Its "History and signing keys" fold gets a history row with the same wording.
- **Resume or letter requests:** the request disappears from "Dead-letter decisions" or the subscription's resume line. Show the closure history only if the read returns a closed-request record. **Do not invent a row from client memory.** If the read has no such record, the notice is the only trace (truthful; record it in verification).

### 4.8 Held redelivery and re-approval — `p02:state:redelivery-held`, `p02:action:redelivery-reapprove`

**Location:** the Events → Deliveries table. The row is a queued redelivery.

**Main row:**
- State column tag: **[TW]** "Held". Uses the existing `.p02-tag` 1px neutral box. Not color-coded.
- Reason column: **[TW]** "Approver no longer holds the right".

**Follow row** (`tr.p02-follow`, spanning the table, same as the Dead-letter decisions block):
1. h4 **[TW]** "Redelivery decision".
2. Fact lines (`p02-small`): "Requested by {requester}" and "Approved by {approver}" (existing phrasing), `bdi` IDs.
3. Status sentence (plan copy, `p` at 16px, max 68ch): "Held: the approver no longer holds the Event reprocessing right. Another holder must approve it again."
4. Then **one** of these:
   - Primary button "Approve again" (plan; KO "다시 승인", AR "اعتماد مجددًا"). Solid dark like "Approve", auto width.
   - Or a `p02-blocked` line, by precedence:
     1. `no_right`;
     2. `requester_lost_right`;
     3. **[TW]** "**Approve again:** You requested this redelivery. Another holder must approve it." or "**Approve again:** You approved it before. Another holder must approve it."

**Confirm step** (existing inline `.p02-confirm`):
- h3 **[TW]** "Confirm re-approval".
- Facts: Redelivery (ID, mono), Event, Subscription, Requested by, Previously approved by.
- Impact **[TW]**: "The redelivery is released to the queue. Nothing is sent: the dispatcher is not connected in this build."
- Primary **[TW]** "Approve redelivery {ID} again"; secondary "Back to edit" (or "Cancel", see §4.6).

**Outcomes:**
- Success notice **[TW]**: "Redelivery {ID} approved again and queued. Nothing is sent."
- Closure when the requester has lost the right: §4.7 notice with "Redelivery {ID}:" as the prefix.

**Closed redelivery** (requester lost the right, closed by dispatch): show it only if the read lists it, with tag "Closed" and the reason **[TW]** "Requester no longer holds the right". No buttons.

### 4.9 Refusal outcomes — `p02:outcome:no-right`, `p02:outcome:no-access`

**`no_right`** (the reload shows the right gone):
- Shown in the row's `RowNote` "kept" slot (`p.live-note-rule`), as existing refusals are. Focus moves to `.p02-row-note`, which carries `role=status` when focused (existing).
- Text: "You no longer hold the {right label} right. Nothing was changed." (plan)
- The reloaded row then shows the `no_right` line (§4.3) instead of the button, so the row tells the same story twice. **This is accepted**: one is the outcome, the other is the current state.
- The rights line updates silently with the reload.

**`no_access`** (the reload returns 403):
- The page switches to the existing denied block: the tab strip stays, scope bar and spine are removed. h2 "Integrations are not available to you", then the existing paragraph.
- In this transition only, insert a second paragraph before the existing one:
  - **[TW]** "Your last action was not carried out. Nothing was changed." The plan's `no_access` wording is "the command message says nothing was changed".
  - Style: 16px `--hb-ink`, max 68ch, `role="status"`.
- Focus moves to the denied h2 (add `tabIndex={-1}` to `#p02-denied`).
- **[TW] option:** align the existing denied paragraph with `P02/spec.md:100` (revoked: details and further commands are not shown; ask an authorized person). If adopted, do not promise a grant path: integration rights cannot be granted (plan R2). Only re-promotion or reinstatement on M01 restores access.

**`authority_unavailable` (503 on reload):**
- The existing error block ("failed" + "Try again") applies.
- **Check (§9 item 12):** while that block shows, the previously loaded page must not offer commands built from the stale `rights`. Set `canCommand` false until a successful reload.
- If root judges this out of scope, record it as a residual. Do not hide it.

---

## 5. admin3 view (stop and reprocess only) — expected composition

The rights line reads "**Your integration rights:** Module stop, Subscription and channel stop and Event reprocessing." (`ListFormat` EN).

**Clients:** h2 "API clients" plus the withheld paragraph (§4.2). Nothing else: no lead, no Clients, Console grants, Service users or OAuth sections.

**Modules** (visible through `stop_modules`):
- The lead, licences, plugin workspace, publishers and versions table are unchanged.
- Installation: **Stop** and **Disable** buttons as today.
- In place of "Propose a change to sample-ledger-export": "**Propose a change:** Needs the Module administration right. You don't hold it."
- Pending proposals: "**Approve:** Needs the Module approval right. You don't hold it." This replaces the own-proposal sentence, which admin3 would not see anyway because they are not the proposer. No Withdraw button, because the items are not admin3's own.

**Channels:**
- Active subscription: **Pause** and **Revoke** buttons. If the console renders "Request resume" for that state, add "**Request resume:** Needs the Subscription and channel administration right. You don't hold it."
- Proposed subscription: "**Approve:** Needs the Subscription and channel approval right. You don't hold it." plus the **Revoke** button if the read allows it.
- "Propose a subscription" h3 followed by its `no_right` line.

**Events:** **Request redelivery** and **Request dismissal** available; approvals follow the existing separation reasons.

**Spine:** the statement, scope bar and notice are identical to admin.

---

## 6. Responsive

| Width | Content | Rights line | Withheld panel | Pending items | Buttons |
| --- | --- | --- | --- | --- | --- |
| 1440 | 1120 | 68ch, 14px | 68ch, 16px | 2 columns, 28% / 1fr | auto width, min-height 44px |
| 1280 | 960 | same | same | 2 columns | auto |
| 1024 | 960, rail stacked | same | same | 2 columns (`.p02-proposal` ≥1024) | auto |
| 390 | 358 | full width; wraps naturally | full width | 1 column | auto in rows; **full-width only in the confirm step** (existing `live.css:586-590`) |

- **No max-width page container.** The 68ch caps apply only to prose.
- **Tables** (Deliveries with a "Held" row) keep their existing behaviour: fixed layout at ≥768; stacked label cells at <768. The held follow row spans all cells, and its content is capped at 68ch.
- **At 390:** the "Held" tag and "Approver no longer holds the right" reason go in their own labelled stacked cells (`data-label`), as other cells do.

---

## 7. RTL (AR)

- **Logical properties only;** no new physical left or right values. The new rules (`.p02-rights`, `.p02-area-withheld`, `.p02-own`) use only `margin-block`, `padding-block`, `border-block` and `max-inline-size`.
- **Right labels and lists:** AR right labels come from the plan's table, joined with the fixed Arabic comma separator `، ` (Arabic comma only). Never splice EN separators into AR.
- **Mono and isolation:** request IDs, proposal numbers, targets, user logins and right-free technical IDs are isolated LTR with `bdi.live-mono` or `iso()`. Arabic prose (closure reason, withheld sentence) must **not** be wrapped in `bdi`; the global `bdi{direction:ltr}` would force it LTR (the HYB-297 defect class).
- **Colons:** the bold-label pattern "**Label:** text" uses the same ASCII colon as the existing AR blockers, for consistency with the rendered AR P02.

---

## 8. Keyboard, focus and announcements

- **Reading and Tab order** follow §3: tabs → Reload → (rights line, not focusable) → notice → area. In a withheld tab: h2 → (paragraph) → pending-item Withdraw buttons in list order.
- **Tab change:** focus moves to `#p02-view-title`, as today. The screen reader then reads the withheld paragraph in normal order. No extra live announcement.
- **Unavailable actions:** never a focusable disabled button. The reason is plain text, read in order.
- **After a command:**

  | Outcome | Focus goes to | Role |
  | --- | --- | --- |
  | success, closed-on-approval, withdrawal, re-approval | `#p02-notice` | `status` |
  | `no_right` refusal | the row's `.p02-row-note` | `status` |
  | `no_access` | denied h2; the inserted paragraph | `status` |

  Do not use `role=alert` for any of these. They are outcomes, not errors.
- **Confirm steps:**
  - Opening one moves focus to its h3, as existing.
  - "Back to edit" or "Cancel" returns focus to the opener (`data-opener`, existing `useOpen`).
  - After a withdrawal removes the item, focus goes to `#p02-notice`, not to a removed node.
- **Focus ring:** the existing visible focus indication, unchanged. Do not add persistent outlines to the withheld panel or the pending list.

---

## 9. Rendered-QA checks (binding for `qa310p02.mjs` and screenshots)

Run each check in EN, KO and AR at 1440 and 390. Run 1280 and 1024 in EN, one tab each, for wrap and column checks.

1. **Rights line text and placement:**
   - admin/admin2 show all eight labels in the plan's order; admin3 shows exactly three.
   - The old "Every command here currently needs installation administration…" text is absent from the DOM on every tab.
2. **Withheld Clients tab (admin3):**
   - The panel contains only the h2, the withheld paragraph and, if present, `.p02-own`.
   - Assert that none of these exist: `table`, `.p02-section`, `.live-empty`, `.p02-lead`, the text "No clients", "No service users", and any digit inside the panel apart from item IDs.
3. **Withheld panel computed style:**
   - `border-inline-start-width` equals `border-inline-end-width` (0).
   - `border-block-*` color equals `--border-row`.
   - `background-color` is transparent.
   - No `::before` content.
4. **`no_right` lines:**
   - Modules pending proposal (admin3) shows "Needs the Module approval right" and has no Approve button in the DOM.
   - Channels proposed subscription likewise.
   - The "Propose a change" and "Propose a subscription" lines replace their buttons.
5. **Precedence:** with admin2 as proposer and admin3 as reader, the proposal row shows only the `no_right` line, not "You proposed this…".
6. **Group F (lost right):**
   - The closed-on-approval notice has the exact plan sentence and does not contain "Nothing was changed".
   - After reload, the Closed group lists the proposal with tag "Closed", the closing reason, the approver and the request ID.
   - Y shows the `requester_lost_right` line and no Approve button.
7. **Member after demotion:** the denied block; no scope bar; no rights line.
8. **`no_access` transition** (only if producible in the browser; otherwise record it as not browser-verified):
   - the denied h2 has focus;
   - the "Nothing was changed" paragraph is present.
9. **Held redelivery:**
   - The tag "Held" is present along with the follow row and either "Approve again" or the precedence line.
   - The browser cannot produce a held state without a dispatcher (plan R9). If only seed or test fixtures can produce it, record it as **not browser-verified** and do not claim it.
10. **Your pending items:** not producible in the browser (plan, Browser QA item 5). Record it as not browser-verified; the console unit tests cover rendering.
11. **RTL (AR):**
    - Request IDs and targets compute `direction: ltr`.
    - Arabic sentences compute `direction: rtl`.
    - The rights list uses the Arabic separators.
12. **Stale commands after 503:** after a forced 503 reload, no action button triggers a command (§4.9). If not implemented, record it as a residual.
13. **Global neutrality:** no element added by HYB-310 has a non-neutral border color, a one-sided border, an accent background or an icon. Verify computed styles, not only screenshots.
14. **390 tabs:** the 2×2 tab grid is unchanged; a withheld tab's label is identical to a visible tab's label.
15. **Scope:** record which tabs, states, locales and widths were inspected. A PASS on these is not owner acceptance and does not audit unchanged legacy P02 sections.

---

## 10. Must not appear

- **Secrecy wording:** "confidential", "secret", "hidden for security", "protected", "restricted data", or anything implying that the withheld facts are secret from installation administrators (plan F8, R3).
- **Fake empties:** empty-state sentences, "0", "None", "No …" or counts inside a withheld area.
- **Error look:** error styling, `role=alert`, or the error block on a withheld area.
- **Decorative treatment:** lock icons, badges on tabs, tinted or striped panels, colored or one-sided borders, accent outlines on rows or selection.
- **Raw IDs:** raw right IDs (`stop_modules` and so on) in user-facing text. They may appear in `data-*` attributes for QA only.
- **Disabled buttons** standing in for `no_right` or `requester_lost_right`.
- **Wrong closure wording:** "Nothing was changed" for a closed-on-approval outcome. "Withdrawn" or "Rejected" as the state of a lost-right closure.
- **Grant promises:**
  - links or prompts to request, grant or enrol integration rights on M03, or anywhere else;
  - "ask your administrator to grant" (no grant path exists, plan R2/Q5);
  - any change to M03 labels or rows (plan: M03 shows no integration right).
- **Invented data:**
  - client-side history rows for closures the read did not return;
  - any claim that a redelivery was sent, or that the hold was verified end to end.
- **Leaked state reasons:** state reasons for an action the reader lacks the right for. They can reveal state the reader did not ask about, and the plan orders `no_right` first.

---

## 11. Open items for root and TW

| # | Item | Default in this spec |
| --- | --- | --- |
| D1 | Plan copy "It can't be approved and will be closed." is not reachable from the console (approve hidden). | Drop "will be closed" (§4.5); root may instead keep a closing approve action. |
| D2 | `p02:status:rights-unknown` is new copy and a new state. | Add it (§4.1); fail-closed behaviour is unchanged. |
| D3 | `{Object}:` prefix on the closed-on-approval notice. | Add it (§4.7). |
| D4 | A second paragraph in the denied block for `no_access`. | Add it (§4.9). |
| D5 | Right labels mid-sentence: capitalisation. | TW decides one rule. |
| D6 | Stale commands while a 503 error shows. | `canCommand` false; otherwise record a residual. |
| D7 | Closure trace for resume and letter requests when the read has no closed-request record. | Notice only; no invented row. |
| D8 | Secondary label in the withdraw-only confirm: "Back to edit" vs "Cancel". | TW decides. |

---

## 12. Critique against the owner's brief

**Meets:**
- **Full-viewport width is kept:** no new container. Prose caps (68ch) apply only to sentences, and buttons stay auto-width except in mobile confirm steps.
- **Hierarchy uses position, weight and neutral rules only;** no decorative edges.
- **The square, neutral vocabulary** of the current build is reused unchanged.
- **IA is preserved:** same tabs, same order, same screen ID and spine. Only one line is replaced.
- **"Unknown" and "withheld" are kept distinct** from empty and 0 (`usability.md:5`).
- **The P02 spec is honoured:** :33 (separate rights are visible as separate labels), :100 (revoked: no details, no further commands) and :116 (live-region outcomes, no color-only state).

**Risks and weaknesses:**
- **Repetition:** per-action `no_right` lines can repeat the same right on many rows, for example a reader without `approve_modules` and many pending proposals. I accepted this for explicitness on an admin screen. If renders show it dominating the Modules tab, the fallback is one area-level sentence under the lead ("You can't approve or propose module changes here: …") with per-row lines suppressed. That is a design change for root to approve after seeing renders, not a default.
- **Long rights line** (three lines at 1440 for all eight). Grouping by area would be shorter but would deviate from the plan's copy; left to TW.
- **Unverified evidence:** external references are UNVERIFIED (§0), and no 1024 or 1280 render was inspected. This spec is not render-verified.

## 13. Root decisions (2026-11-19)

Binding over the matching proposals.

1. **D1:** accepted as the default — the starter-lost line reads "**Approve:** The person who started this no longer holds the right, so it can't be approved." The approve button stays hidden; the closure happens only when a stale page or direct call attempts the approval (§4.7 covers that outcome).
2. **D2, D3, D4, D7:** accepted as specified.
3. **D5, D8:** for the technical-writing review.
4. **D6:** accepted — `canCommand` is false while the error block from a failed reload shows; tested.
5. **Repetition risk (§12):** keep per-row lines; root reviews the renders before any area-level fallback.
6. **References:** this change reuses P02's existing row-note, notice, denied and list patterns and adds no new layout pattern; no new external reference is required for it. The P02 references recorded for HB-285 apply.
7. **§4.3 order (render review round 1):** the existing P02 pattern puts the reason list before the buttons; §4.3 is corrected to match the implementation.
