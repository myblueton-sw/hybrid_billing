---
type: DesignPlan
title: M02 trust changes on the bounded reads — composition
description: The ui-designer's composition for moving the M02 list of trust changes onto the bounded list and detail reads, with the copy in three languages.
status: stable
---

# Scope and ownership

HYB-304 increment M2d of the [M02 query design](m02-query-design.md), machine:mac_mini. The composition is the ui-designer's (role-based AI designer, by reading, 2026-11-27); root recorded it and implements it. Nothing is built by this document and it is not owner acceptance. It follows the composition of the M03 lists ([requests](m03-requests-composition.md)) and reuses `apps/console/src/modules/identity/pages/access/paged.tsx`.

# Decisions

Every decision is required unless marked optional.

## 1. Structure

- Two sections as today, "Open (n)" and "Closed (n)", each its own paged read: `state=open` and `state=closed`. The administrator acts on open changes; closed ones are history. One list with a group filter would hide the actionable rows while history is shown.
- Page size: open 100 (the read's maximum; the group holds at most 100 changes awaiting a decision plus any waiting activations), closed 25.
  - **Built with 50 for the open section** (2026-11-27). The read accepts a limit only up to the installation's page limit, whose default is 50 (range 1 to 100), so the real service refuses 100 with the default setting; the rendered check against the real service found it. An open group of 51 or more therefore spans pages and is shown in the service's order with the line of decision 2. The console does not know the installation's page limit on this screen; an installation that lowers it below 50 (or below 25 for the closed section) would refuse these reads. Tracked in HYB-376 with the same condition on the M03 lists.
- One filter row above both sections, under the changes heading:
  - Search, over the identifier only (numbers repeat across runs).
  - State: All, then the four open states, then the four closed states, with the existing state names. Choosing one state shows only the section that contains it; the other section is hidden, not shown empty.
  - "Apply filters" and "Reset filters" as on M03; Enter in the search field applies. Both lists restart at their first page.
- Optional: hide the filter row when no change is kept at all.

## 2. Order

- Open section: when the whole group is on one page (first page, no continuation), the browser sorts by state priority as today (activation unknown, approved, validated, draft, then newest). When it spans pages, the service's order is shown with one line above the table saying so.
- Closed section: always the service's order, newest first. This changes today's state-priority sort; for history newest first is the natural reading and needs no line.
- Not chosen: sorting only the first page of a group that spans pages, which would show a partial sort as complete.

## 3. Rows without evidence currency

- No per-row replacement for the "evidence changed" line. The Next column shows the next step with staleness assumed false; it is exact except in the stale case, which the detail corrects.
- One note under the open section's table, shown only when a validated or approved row is listed.
- The detail keeps staleness: its state sentence, the stale next step and the reasons approval or activation are not offered.

## 4. The detail as its own read

- While it loads the list stays visible; the opened row keeps `aria-current` and the "Selected" tag; wide widths use the reduced list beside the panel, narrow widths the step mode with "Back to trust changes".
- The panel shows its heading at once: the change's number from the row, or, when opened by identifier (the waiting notice, a successor), the identifier in mono until the detail arrives. Under it a status line; Close stays available.
- Failure: a title, the shared read-failure wording, "Try again" and "Close".
- Unknown identifier (`invalid_request`): a statement that the change is no longer kept, then "Close"; both lists reload to their first page. No "Try again".
- Focus: on open, the detail heading, while loading too. On close, the row's opener if it is on the current page, otherwise the changes section.
- Openers that no longer come from the list: the waiting notice's button and the superseded change's successor, which is shown as the identifier in mono after the existing "superseded by" label and opened by the detail read. The successor's number is no longer shown.

## 5. After a command

- Validate, approve, reject, cancel, revise, propose: announce as today; re-read the policy read; restart both lists at their first page keeping the filters (a kept cursor would answer 409); re-read the detail and focus its heading.
- Section counts are each list's exact count for the current filter; with a filter applied the headings still read "Open (n)" and "Closed (n)", n being the matches.
- Activate: the after-activation screen, unchanged.
- A 409 on a cursor: the M03 behaviour, the list failure with the "query changed" wording and "First". Not chosen: restarting automatically.

## 6. Paging, reference, empty states, widths

- Paging under a section's table only beyond one page; the reference line under each table; after paging, focus goes to that list's reference line.
- Empty: without a filter the existing sentences; with a search or state, the two new sentences.
- A list's failure does not affect the other list or the current rules.
- 1440 and 1280: the five columns as today, reduced single column beside the panel. 1024: the same if the wide breakpoint holds there, otherwise step mode. 390: step mode, the filter row stacked full width.
- Right-to-left: identifiers and epochs stay left-to-right mono; paging mirrors. Neutral borders only.

## 7. What the whole list gave

| Fact | Kept by |
| --- | --- |
| Waiting-activation notice | `waiting_activation`, with the new opener text |
| Earlier-run note | `earlier_run_kept` in place of looking through the list; the origin conditions stay |
| The successor of a superseded change | its identifier, opened by the detail read; its number is lost |
| The selected change | the detail read's result |
| Evidence-changed line and stale next step | the detail only |
| Open and closed counts | the lists' counts |

# Copy

| Use | English | Korean | Arabic |
| --- | --- | --- | --- |
| Search label | Change ID | 변경 ID | معرّف التغيير |
| Search hint | Part of the change identifier. | 변경 ID의 일부입니다. | جزء من معرّف التغيير. |
| Open group spans pages | Newest first. The open changes fill more than one page, so they are not sorted by state. | 최신순입니다. 진행 중인 변경이 한 페이지를 넘어 상태순으로 정렬하지 않습니다. | الأحدث أولًا. تتجاوز التغييرات المفتوحة صفحة واحدة، لذا لا تُرتَّب حسب الحالة. |
| Evidence note | Whether a change's evidence is still current is checked when you open it. | 변경의 근거가 아직 유효한지는 변경을 열 때 확인합니다. | يُتحقَّق مما إذا كانت أدلة التغيير لا تزال سارية عند فتحه. |
| Detail loading | Loading the change… | 변경을 불러오는 중… | جارٍ تحميل التغيير… |
| Detail failure title | The change could not be loaded | 변경을 불러오지 못했습니다 | تعذّر تحميل التغيير |
| No longer kept | This change is no longer kept. Older changes are removed when the list reaches its limit. | 이 변경은 더 이상 보관되지 않습니다. 목록이 한도에 이르면 오래된 변경이 삭제됩니다. | لم يعد هذا التغيير محفوظًا. تُحذف التغييرات الأقدم عندما تبلغ القائمة حدّها. |
| Waiting notice opener | Open the change waiting for activation | 활성화 대기 중인 변경 열기 | فتح التغيير المنتظر للتفعيل |
| Successor opener | Open the replacing change | 대체한 변경 열기 | فتح التغيير البديل |
| No open match | No open change matches the filters. | 필터와 일치하는 진행 중인 변경이 없습니다. | لا يوجد تغيير مفتوح يطابق عوامل التصفية. |
| No closed match | No closed change matches the filters. | 필터와 일치하는 종료된 변경이 없습니다. | لا يوجد تغيير مغلق يطابق عوامل التصفية. |
| Open list failure title | Open changes could not be loaded | 진행 중인 변경을 불러오지 못했습니다 | تعذّر تحميل التغييرات المفتوحة |
| Closed list failure title | Closed changes could not be loaded | 종료된 변경을 불러오지 못했습니다 | تعذّر تحميل التغييرات المغلقة |

State label, "All", "Apply filters", "Reset filters", "Try again", "Close", the read-failure wording and the "query changed" wording reuse existing strings.

# Not verified by the designer

- The wide breakpoint, so the behaviour at 1024; `trust.css`; any rendering.
- Whether the shared read-failure wording maps a cursor's 409 to "query changed", as M03 relies on.

# Implementation order

1. Model and list: row type adapted to the list row, the two paged sections, filters, order, notes, empty and failure states.
2. Detail read: loading, failure, no-longer-kept, focus, the two openers; refresh after commands.
3. Rendered check in three languages at four widths; ui-designer rendered review; guideline and accessibility pass.
