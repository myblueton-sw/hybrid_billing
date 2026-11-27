---
type: DesignPlan
title: M03 grants and delegations as one list — composition
description: The ui-designer's composition for the one permissions list of the M03 grants view, with its columns, filters, paging, headings and focus, recorded before implementation.
status: stable
---

# Final status

Checkpoint record. The work it names as remaining or not yet built was completed later in HYB-304, which closed on 2026-11-27 ([closing audit](closing-audit.md)).

# Scope and ownership

HYB-304 increment C3, machine:mac_mini. The owner decided on 2026-11-26 that grants and delegations are shown as one list ([query design](m03-query-design.md)). This is the composition specified by the ui-designer role on 2026-11-26 by reading the screen's code; nothing was built or rendered for it, no external reference was consulted (still owed under the service screen style), and it is not owner acceptance. Root implements it. The requests, revocations and ceiling sections and the side panel are not part of this increment.

# Columns

Eight columns in the owner's order. A cell's second line is secondary text.

| # | Header (EN / KO / AR) | Grant | Delegation |
| --- | --- | --- | --- |
| 1 | Holder / 보유자 / صاحب الصلاحية | sign-in name; user identifier below. A console-client grant: "Console / 콘솔 / وحدة التحكم" with the existing "Console ceiling" below | the delegate's sign-in name; "Delegated by {name} / {name}님이 위임 / فوّضه {name}" below |
| 2 | Access (existing) | the existing access text; the kind label below | same |
| 3 | Scope / 범위 / النطاق | "Customer: All or identifier" and "Legal entity: All or identifier" always; contract and period only when restricted | same |
| 4 | Source / 권한 출처 / المصدر | "Grant / 권한 부여 / منح" | the existing "Delegation"; "can be redelegated" below when allowed |
| 5 | Original delegation / 원 위임 / التفويض الأصلي | "None / 없음 / لا يوجد", secondary | the parent delegation's identifier in mono, or "None" |
| 6 | Validity / 유효 기간 / مدة الصلاحية | "From {date} / {date}부터 / من {date}"; the existing "until" line below; the screen's date format with UTC and the exact value in a title | same |
| 7 | Effective permission / 실효 권한 / الصلاحية الفعلية | the existing state line with its reason; a state other than active stays bold; "Revocation requested" as a third line when a request is pending | same; a delegation ended through its chain uses the existing wording |
| 8 | Actions (visually hidden header) | "Request revocation" under today's rules; "Your grant" for one's own; nothing when a request is pending; for a console-client grant "Set by the installation / 설치 환경에서 설정 / يحدده التثبيت", quiet | "Revoke" when it can be revoked |

Widths at full width: 15 / 17 / 15 / 9 / 10 / 12 / 13 / 9 percent; cells wrap; neutral row dividers; the selected row keeps the neutral selected background. Below a list width of 960px, as beside the open panel, the original delegation moves into Source as a third line and Validity into Effective permission, leaving six columns. Below a list width of 600px each row becomes a card in the order holder, access, scope, source, validity, effective permission, action.

# Filters

Offered now, in this order: search (hint: "Grant or delegation ID, or holder user ID"); holder user identifier as a text field (hint: "Exact user ID. Choosing a person by name is not available yet."); action as a select; customer identifier; legal entity identifier; source (All, Grant, Delegation); state (All, Active, Not yet valid, Revocation not confirmed, Expired, Revoked, Ended through delegation chain); "Ends before" as a date, read as the start of that day in UTC. Korean and Arabic strings are the designer's and are in the implementation's copy file.

All states are shown by default; ended rows are told apart by their bold state. Nothing is sent until "Apply filters" or Enter in a text field; "Reset filters" clears every field and reads the first page; both reset the cursor trail; filters live in the address, as on M04. At wide widths the filters are an always-visible grid; at 390 they collapse under "Filters ({n} applied)", open when any is applied.

# Paging and count

Above the table: the count ("Matching count: 37", or "Matching count, at least: 50" while more remain) and the list's reference time, in small secondary text. Below it: First, Previous, Next; First and Previous disabled on the first page, Next when there is no cursor. The page size is fixed at 50. An empty last page says there are no matches; an empty page with more to scan says so and keeps Next enabled. A failed read uses the shared read boundary with Reload and First.

Removed: the "N active grants" and "N active delegations" counts, which cannot be computed across pages, and the "Loaded at" line, replaced by the reference time.

# Headings and focus

One heading: "Grants and delegations / 권한 부여·위임 / المنح والتفويضات", focusable, with the request button and Reload in its toolbar. The "Grants to people" and "Delegations" headings, the separate delegations section and both collapsed "ended" lists are removed. After Apply, Reset or a paging button, focus moves to the count line; on a failure, to the failure box. Row actions return focus to their button as today, or to the heading when the row is no longer on the page.

# Decided by root where the specification left a choice

- **Default state filter.** The specification preferred a "Current" default (active, not yet valid, revocation not confirmed) if the server accepted a set of states. The read takes one state. All states are shown by default in this increment; a "current" value is noted as a possible addition to the read.
- **Action options.** The select lists the actions of the console's catalog, which is what a request can name.

# To capture for the rendered review

English, Korean and Arabic at 1440 with the panel closed, 1440 with it open, and 390: a first page containing a console-client grant, a person's grant, one's own grant, a delegation with a parent that allows redelegation, one ended through its chain, an expired and a revoked row, a pending revocation request, a "does not apply" row and a restricted contract or period; three filters applied; a lower-bound count; the last page; both empty states; a read failure; focus on the count line after Apply and after Next; the six-column layout beside the panel.
