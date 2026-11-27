---
type: DesignPlan
title: M03 access change requests on paged reads — composition
description: The ui-designer's composition for the requests section of the M03 grants view once it reads pending and ended requests a page at a time, recorded before implementation.
status: stable
---

# Scope and ownership

HYB-304 increment T3 of [the screen transition](m03-screen-transition.md), machine:mac_mini. Specified by the ui-designer role on 2026-11-26 by reading `Requests.tsx` and `request-copy.ts`; the role did not reopen the permissions list or its composition in that pass and relied on the patterns it had specified and reviewed there. Nothing was built or rendered, no external reference was consulted, and it is not owner acceptance. Root implements it. The decision panel is unchanged.

# Two lists

"Waiting" and "Ended" stay two lists, each with its own read (`state=pending`, `state=ended`, newest first), cursor, count and reference time; not one list with a state control. Pending rows carry the actions and must always be visible; ended rows are history and stay collapsed; a state control would hide the actionable rows whenever history is looked at. Both are read when the section loads, so the ended summary can show its count. Rows use the sign-in names they carry; where a row carries none, the user identifier is shown alone.

# Page size and paging

Pending 50 a page; ended 25. Paging controls (First, Previous, Next, the permissions list's buttons and disabled look) appear below a list only when it has more than one page: a next cursor, or not on its first page. This differs from the permissions list on purpose: under one or two requests three disabled buttons are clutter.

# Counts and reference time

- **Pending.** The count stays in the heading's meta line: the existing "{n} waiting" when exact; when a lower bound, new "At least {n} waiting / 대기 {n}건 이상 / عدد الطلبات بانتظار الاعتماد: {n} على الأقل". Hidden while loading or failed. One reference line under the table, small and secondary, the existing "List reference time".
- **Ended.** The count is in the disclosure's summary: the existing "Ended requests ({n})" when exact; a lower bound "Ended requests (at least {n}) / 종료된 요청 (최소 {n}건) / الطلبات المنتهية ({n} على الأقل)"; a failed read "Ended requests (could not be loaded) / 종료된 요청 (불러오지 못함) / الطلبات المنتهية (تعذّر التحميل)"; loading, no number; exact zero, the disclosure is hidden as today. Inside, one reference line above the table. No "Matching count" line: nothing is filtered.
- The note that requests are not kept across a restart stays between the pending list and the disclosure.

# Loading, empty and failure

- **Pending.** Loading: the existing "Loading access…" as a status line. Empty: the existing "no requests" text. Failure: title "Access change requests could not be loaded / 권한 변경 승인 요청을 불러오지 못했습니다 / تعذّر تحميل طلبات تغيير الصلاحيات", the read wording the permissions list uses for its body, Reload, and First when not on the first page.
- **Ended.** Loading and failure inside the disclosure; failure title "Ended requests could not be loaded / 종료된 요청을 불러오지 못했습니다 / تعذّر تحميل الطلبات المنتهية", same body and actions.
- One list's failure does not affect the other or the permissions list; the pending actions stay usable when only the ended read fails.

# After a decision

Pending and ended return to their first page and the disclosure keeps its open or closed state; the permissions list reloads its current page with its filters and shows its own failure if its cursor no longer fits. The existing confirmation text is shown as a status line directly under the requests heading, announced through the polite live region, until the next action or reload; it belongs to the section, not to a row, since the row usually moves to the ended list. Focus goes to the requests heading. A failed decision stays in the panel as today; Cancel returns focus to the row's action.

# Widths

- 1440 without the panel: fixed-layout table, change and person 30, access 30, status 25, actions 15 percent; 8px inline cell padding.
- 1440 beside the panel (about 700px): 28 / 28 / 26 / 18; action buttons stacked with a 4px gap at their natural width, wrapping; text breaks at words, identifiers and digests anywhere.
- 390: cards in the order change and person (bold), proposer, identifier and digest, access, status lines, actions at natural width; paging buttons stack full width.
- Neutral row dividers; the selected row keeps the neutral selected background.

# Left open by the specification, decided by root

- **Paging controls only when needed.** Taken as specified; making them always visible is one condition if the owner prefers consistency with the permissions list.
- **Ended page size 25.** Taken as specified.

# To capture for the rendered review

English, Korean and Arabic at 1440 without the panel, 1440 with it, and 390: two pending rows (one out of date, one with no eligible approver); pending empty; pending beyond one page with the lower-bound line and paging; ended collapsed with an exact count and with a lower bound; ended open on its second page; an ended read failure with the summary saying so; a pending read failure with the ended list working; after a rejection, the message under the heading, focus on the heading and the row in the ended list; after an approval while the permissions list is on its second page.
