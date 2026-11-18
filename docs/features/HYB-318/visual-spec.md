---
type: Visual Spec
title: HYB-318 visual spec for the O01 Create change plan handoff and the O02 prefill and base revision states
description: Placement, labels, states, layout, focus order, locale and responsive rules for the O01 "Create change plan" action, O02's arrival with prefilled values, the read-only carried base revision, the base-changed notice with "Base on revision N" and the refused-propose state, with root decisions.
status: final
tags: [HYB-318, HYB-300, O01, O02, visual-spec, handoff, revision, ui-designer]
sources:
  - resource: ../HYB-300/plan.md
  - resource: ../HYB-300/visual-spec.md
  - resource: ../HYB-296/plan.md
  - resource: references.md
---

# HYB-318: O01 "Create change plan" → O02 prefill (visual spec)

Author: the ui-designer role (Claude, read-only), 2026-11-18; root decisions in §12 are binding. HYB-318 is Part B of the approved HYB-300 plan ("Part B: create change plan handoff" and its root amendments). O01 and O02 spec references point to the local planning inputs under `docs/ui/screens/`. This is a design proposal, not a render PASS or owner acceptance.

## 0. Evidence

- O01 renders (`hb300work/shotsA`): the panel at 1440 is about 380px with h2, a facts `dl` and one secondary 44px action "View history"; at 390 AR the panel is a step-in view and the action stretches to full width (existing). The groups view's group panel lists members as text; the history view has no panel.
- O02 renders (`hb300work/regshots3/o02`): scope row, hairline, h2 "Change target", a grid of the fields column (`minmax(320px,440px)`) and a "Current" aside with a neutral inline-start rule. The stale wording uses a bold lead "Base changed." inside `.live-note-rule`.
- Existing KO copy uses 리비전 for revision; the plan's Part B KO drafts used 개정.
- No render of a prefilled O02 form exists yet.

## 1. Composition

The handoff is navigation from a detail to a form, not a command. O01 gets one secondary action in the existing detail action row; O02 gets one neutral note block between its heading and the form and one read-only fact at the top of the fields column. Base versus active revision is always stated in text. No dialog, toast, overlay or coloured edge.

## 2. O01: Create change plan

**Placement:** the detail panel's action row (`div.live-buttons`, `flex-wrap:wrap; gap:8px 16px`), before "View history". Not in the list-header end slot (reserved for Part C) and not as a row action.

**Element:** `<a id="o01-create-plan" href="#/organization-changes?view=targets&kind=…&member=…[&from=…]&base=…[&lang=…]" aria-describedby="o01-plan-hint">` styled as the secondary button: `display:inline-flex; align-items:center; min-block-size:44px; padding-inline:20px; border:1px solid var(--border-control); border-radius:0; color:var(--hb-ink); background:transparent; text-decoration:none; inline-size:auto`; hover changes the neutral background only; focus token 2px/2px. Label "Create change plan", no icon. Neither action is primary.

**Hint:** `p#o01-plan-hint.live-note` below the row, 14px muted, `max-inline-size:60ch`, `margin-block:8px 0`; revisions in `<bdi dir="ltr">`, dates in `<time>`.

| View | Placement | Kind carried | Rendered when |
| --- | --- | --- | --- |
| Tree | member panel action row | `hierarchy` | the selected node is in the loaded tree at the pinned revision and can be a hierarchy member (§12 R8) |
| Accounts | member panel action row | the view's `rk` | the selected row is on the loaded pages |
| Groups, outside-list member | member panel action row | the group kind | the member is in the loaded outside list |
| Groups, a group selected | none | — | never; a group is a target |
| History | an action row after the end note in the history section | `rk` | a timeline is loaded |

**States:** absent, never disabled, when it cannot apply (not-loaded selection, no selection, group panel, history without a timeline, loading, error, denied, revoked). No busy state. With an older pinned revision R the link carries R and O02 shows the base notice. No inline Leave/Stay for this URL navigation; the app guard covers unsaved input.

## 3. O02 arrival (targets view)

Order: scope row; `#o02-notice` (empty on arrival); h2 "Change target"; new `div#o02-handoff.live-note-rule` (`tabIndex=-1`, no live role, `border-block-start:1px solid var(--border-row)`, `padding-block:16px`, `margin-block-end:24px`, children `max-inline-size:68ch`); form grid.

Handoff block content:
- always the `prefilled` sentence;
- with a past `from` in the link, the `fromPassed` sentence;
- when the carried base differs from `page.active_revision`: `<strong>Base changed.</strong>` then `baseChanged`, and below (`margin-block-start:12px`) `button#o02-use-base` "Base on revision {a}", secondary, auto width, 44px.

Read-only base fact as the first child of the fields column: `dl.live-facts`, dt "Based on relationship revision", dd `{r}` plus " · active" or " · not active; revision {a} is active". Not focusable; rendered only while a base is carried.

| Field | Prefill | Editable | Notes |
| --- | --- | --- | --- |
| Relationship kind | `kind` from the link; unknown ignored | yes | changing it clears change and target, as today |
| Change | not prefilled | yes | |
| Member | the canonical ID | yes | mode resolved once after the first choices read settles: in the choices → select shows "Name (id)"; otherwise ID mode with the ID; before choices settle a one-option select `{id}` plus the loading-choices line; never switched after a user edit |
| Target | not prefilled | yes | |
| Effective from | `from` if today or later, else empty plus `fromPassed` | yes | |
| Reason | empty | yes | |
| Base | `base` from the link | no; only via "Base on revision {a}" | |

Link values are read once and removed from the URL (`lang` kept). The "Current" aside shows the prefilled member's current targets from O02's own read.

## 4. Base on revision N

- Compared on arrival and on every plans read that changes `active_revision` while a base is carried.
- Click: base := active; notice removed; fact reads "{a} · active"; `baseAdopted` announced in `#o02-announce`; focus to `#o02-current-title` (`tabIndex=-1`). The click clears the propose attempt so the idempotency key is never reused with a different base, and does not mark the form dirty.
- Never automatic.
- Propose while the base is stale: client validation adds the `eBase` summary problem linking to `#o02-use-base`; the server check stays authoritative.

## 5. Refused propose (`revision_conflict`)

Reload the plans read, then compare. If the carried base differs from the reloaded active revision: no `#o02-notice`; the handoff block shows `<strong>Base changed.</strong>` + `baseRefused`; focus `#o02-handoff`; entries kept; no error summary. Otherwise today's conflict sentence in `#o02-notice`. Never both.

## 6. Unsaved input

O02 reports dirty through `onDirty` only after a user edit; prefill and base adoption are not edits. Shell navigation, hash changes, Back and unload go through the app guard; no page-level `beforeunload` and no inline Leave/Stay. Browser Back from O02 restores O01's URL and selection (`sel`), and the action is present again.

## 7. Widths

| Width | O01 | O02 |
| --- | --- | --- |
| 1440 | panel ~380px; both actions on one row in EN/KO/AR; hint wraps 2–3 lines | handoff block full row, text ≤68ch; base button at inline-start; fields 440px with the base fact on top |
| 1280 | panel 360px; one row | same |
| 768–1279 | step-in detail; actions wrap, auto width | the existing O02 grid keeps fields and the Current aside side by side where they fit (rendered at 1024 without clipping); otherwise single column |
| 390 | step-in detail; both actions follow the existing mobile `.live-buttons` behaviour alike | text wraps; base button like Propose; fact stacks dt over dd; no horizontal scroll |

No stretched inputs or buttons at ≥768px; no centred container.

## 8. Focus and keyboard

1. O01: row → panel heading/facts → Create change plan → View history. History: table → end note → Create change plan.
2. Enter activates the link; the app guard runs.
3. O02: after the plans read and prefill (not waiting for choices) focus moves once to `#o02-handoff`.
4. Then: Base on revision (if shown) → kind radios → change radios → member → ID toggle → target and toggle → Effective from → Reason → Propose.
5. After "Base on revision": focus to the Current heading, announced.
6. After a refused propose: `#o02-handoff` (base case) or `#o02-notice` (plain conflict).
7. Programmatic focus on the block shows no persistent outline (`:focus-visible` only).

## 9. Locales and RTL

Logical properties only; AR runs from the right (Create change plan, then View history). IDs, revisions and the member ID input in LTR isolation, mono. KO `keep-all`, 리비전 throughout. AR line-height 1.8, `min-block-size` only.

## 10. Copy

Superseded by the technical-writing review of 2026-11-18: the strings in `apps/console/src/modules/organizations/pages/organizations/live-copy.ts` (`createPlan`, `planHint`, `planHintNoDate`) and `.../changes/live-copy.ts` (`prefilled`, `prefilledNoBase`, `baseChanged`, `baseRefused`, `useBase`, `baseAdopted`, `baseFact`, `baseActive`, `baseNotActive`, `eBase`, `fromPassed`, `linkDropped`) and the glossary HYB-318 section are authoritative. Principles from that review: the form is "this change" until Propose succeeds (never "the plan"); the base notice states the base and the active revision rather than claiming when it changed, except in `baseRefused`; screen and field names match their actual titles and labels in each language (KO "조직 변경 계획 화면", "적용 시작일"); the AR base action avoids الاعتماد, which means Approval on O02; a number and its attached word or particle never break across lines. KO and AR remain drafts for native review.

## 11. References

Recorded by root in `references.md`.

## 12. Root decisions (2026-11-18)

1. **R1:** `organizations/live.css` and `changes/live.css` are added to HYB-318's allowed files.
2. **R2:** an anchor, matching O02's `o01Href` precedent.
3. **R3:** confirmed; O01 omits a past as-of date, `fromPassed` covers hand-edited or post-midnight links, `planHintNoDate` covers O01.
4. **R4:** adopted; Propose is blocked with the `eBase` summary while the carried base is stale. Server check stays.
5. **R5:** adopted.
6. **R6:** adopted; compared on every plans read that changes `active_revision`.
7. **R7:** focus on `#o02-handoff` only, no `role=status` (supersedes the plan's `role=status`).
8. **R8:** the implementer checks the domain: if a legal entity cannot be a hierarchy member, the action is absent on legal-entity nodes; record the finding.
9. **R9:** carry the older pinned revision; O02 shows the base notice.
10. **R10:** adopted.
11. **R11:** 리비전 in all Part B KO copy.
12. **R12:** adopted (history action after the end note).
13. **R13:** adopted (no action on a group panel).
14. **R14:** unknown `kind` and malformed `member` are ignored; a non-canonical or missing `base` means no base is carried and the base fact is absent; one status line "A value in the link was not valid and was removed." when anything was dropped. A dropped base is acceptable because propose then sends no base and the server applies its normal active-revision check.

## 13. Must not appear

Display names, the target or the action in the handoff URL; a prefilled target or change; silent or automatic rebasing; toasts, dialogs or overlays; coloured strips, outlines or tinted notices; a disabled Create change plan; the action on not-loaded selections, group panels or a history view without a timeline; a second Leave/Stay prompt or a page-level `beforeunload`; two refusal messages at once; stretched controls at ≥768px or a centred container; Unicode arrows, emoji, rounded corners or shadows; any Part C control; wording that claims a plan exists before Propose succeeds.
15. **Render rounds (2026-11-18):** programmatic focus targets (`#o02-handoff`, `#o02-current-title`) show no outline; numbers and attached words stay on one line; O02 keeps its two-column grid at 1024 where it fits (§7 amended).
