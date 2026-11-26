---
type: DesignPlan
title: M03 revocations on a paged read — composition
description: The ui-designer's composition for the revocations section of the M03 grants view once it reads kept revocations a page at a time, recorded before implementation.
status: draft
---

# Scope and ownership

HYB-304 increment T4 of [the screen transition](m03-screen-transition.md), machine:mac_mini. Specified by the ui-designer role on 2026-11-26 by reading `Revocations.tsx` and its strings, applying the pattern of [the requests composition](m03-requests-composition.md) without rereading it. Nothing was built or rendered, no external reference was consulted, and it is not owner acceptance. Root implements it. The ledger line is unchanged and still comes from the whole read.

# Page size and paging

Ten records a page: each record is a tall card, and the installation keeps a bounded number of completed revocations, so one page is the normal case. Paging controls (First, Previous, Next, the shared buttons and disabled look) appear below the list only when it has more than one page.

# Count and reference time

- The count goes in the heading's meta line, hidden while loading and after a failure. Exact: "1 revocation" / "{n} revocations", "회수 {n}건", and the counted Arabic forms of the existing revocation unit. Lower bound: "At least {n} revocations / 회수 {n}건 이상 / عدد عمليات الإبطال: {n} على الأقل".
- The reference line ("List reference time") sits directly under the list, before the paging controls, whenever the read succeeded, the empty state included. It is separate from the ledger line, whose "checked" time is a different time.
- Order of the section: heading and meta; the lead; the ledger line; the list; the reference line; paging when needed.

# Loading, empty and failure

Loading: the existing "Loading access…" as a status line where the list goes. Empty: the existing "No revocations yet.", then the reference line. Failure: title "Revocations could not be loaded / 회수 목록을 불러오지 못했습니다 / تعذّر تحميل عمليات الإبطال", the shared read wording, Reload, and First when not on the first page. The ledger line stays visible above a failed list, since it carries the warning that grant additions are paused. The requests and permissions lists are unaffected.

# Records and the target line

Records stay cards in the ordered list at every width: each is a short report, not a row of comparable columns. The card's heading now comes from the record's own facts.

- **A grant that still exists.** The existing "Grant: {access} for {who}", where who is the record's holder sign-in name, or the user identifier when it has none. A second, secondary line: the resource kind, a middle dot, the grant identifier in mono.
- **A delegation that still exists.** New: "Delegation to {who}: {access} / {who}님에게 한 위임: {access} / تفويض إلى {who}: {access}", who being the delegate. The second line as for a grant, with the delegation identifier.
- **A target that is gone** (the facts are empty): the existing "Grant {id}" or "Delegation {id}", with no second line.

Everything else in a card is unchanged.

# Focus

After First, Previous or Next, focus moves to the list's reference line, as in the requests section. After a revocation started elsewhere the list reloads to its first page, where the newest record appears, and existing flows keep their focus target.

# Left open by the specification, decided by root

- **Numbering across pages.** The list's markers are not shown today (`access-records` is styled without them), so nothing continues across pages.
- **A flow that targets one record.** Root found none: the records carry `tabIndex={-1}`, but no flow focuses a record by identifier; flows that end in a revocation focus the section heading. No lookup by identifier is added.

# To capture for the rendered review

English, Korean and Arabic at 1440 with and without the side panel, and 390: a grant still present, a delegation still present with the delegate's name, a grant gone, a record whose decision is not confirmed, a ledger-replay record; the heading count exact and as a lower bound; a second page; the empty state with its reference line; a failed list with the ledger line above it; a ledger state that is not ok with records below; focus on the reference line after Next.
