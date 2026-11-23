---
type: ImplementationPlan
title: Issuance authority provisioning on M03
description: Reuse the result authority provisioning panel on M03 for the protected setup, payment term and draft authority namespaces, with a revised permission-area navigation and per-area copy.
status: draft
---

# Scope and evidence

HYB-341, machine:mac_mini. This is the screen part of item 5 of [the handoff](handoff.md). The server commands and reads for `issuance_protected_authority`, `issuance_term_authority` and `issuance_draft_authority` are merged ([provisioning](issuance-authority-provisioning-contract.md), [reads](issuance-authority-reads-contract.md)), and the console client under `identity/pages/access/result-authority` is domain-aware and wires no screen. Until a screen exists, a governor can provision these capabilities only by calling commands directly.

Root owns planning, implementation and integration. The ui-designer role reviewed this plan from source and documents, without renders; a read-only technical-writing review covers the new terms; the owner performs rendered verification. No deployment or real-account grant is implied.

# Decision: reuse the panel body, revise the navigation

M03 has a provisioning panel for the result domain, reached from the grants view through the "Permission area" navigation: proposal history, inspection, decision review, the proposal form with its review step, and the submissions of the session. Its [plan](result-provisioning-screen-plan.md) records the check-answers and select references and the composition review. The three issuance domains have the same protocol, reads and row shapes, so the panel body is reused with a domain. The capability list is an auto-fitting grid and takes six items without change; labels of this length already exist in the result area.

The navigation is a design change, not three more entries. It grows from two links to five, and today it has no visible current state, no list markup and no controlled wrapping.

## What is parameterised per area

The panel is not literally unchanged. These result-specific values become values of the area:

- the title and introduction;
- the client-refusal sentence, which exists in both `Area.tsx` and the panel;
- the screen item identifiers reported to the shell;
- capability labels, because `approve` and `inspect` are capability names in more than one namespace and `approve` is also an action label;
- the proposal identifier prefix. The form builds the identifier as `result-authority-<request>`, and the identifier is shown in the history row and the detail. An issuance area uses its own prefix (`issuance-protected-authority-`, `issuance-terms-authority-`, `issuance-drafts-authority-`); the result prefix is unchanged. The server accepts any valid request identifier as a proposal identifier, so the prefix is a display convention only;
- the component key, which today contains only session fields. The area is added so that switching between two issuance areas discards the previous area's history, draft and stores.

The shared labels for kinds, states and actions are reused unchanged, and the result area's copy is unchanged.

## Navigation

External references, inspected on 2026-11-23:

| Source | Observed | Adopted | Rejected |
| --- | --- | --- | --- |
| [MOJ Design System, Sub navigation](https://design-patterns.service.justice.gov.uk/components/sub-navigation/) (status "To be reviewed") | "Use this component when you have a second level of navigation"; `nav` with a label, `ul`, `li`, links; the current link carries `aria-current="page"`; not for primary or global navigation | The structure and `aria-current="page"` for a second-level navigation inside one screen | Its default label and its styling, which the page does not describe |
| [GOV.UK Design System, Service navigation](https://design-system.service.gov.uk/components/service-navigation/) | A list of links in a labelled `nav`; the active item is bold as a fallback and marked with `aria-current`; a script-driven Menu toggle on small screens | A weight change as part of the current state | The service-identity bar, the Menu toggle and the script: five links fit as a stacked list on a small screen without a disclosure |

Neither source gives item-count limits or a wrapping rule for about five links; the rules below are this plan's own and are verified by render.

- Structure: one flat `nav` labelled "Permission area" containing a list of five links, with its own class instead of the button-row utility it uses today. No grouping at five entries.
- Labels and order: Workspace access, ERP result permissions, Protected issuance settings permissions, Payment terms permissions, Issuance draft permissions. The existing two keep their place; the issuance areas follow setup order. The link text equals the area's title.
- Selection: the existing `area` query parameter with the values `issuance-protected`, `issuance-terms` and `issuance-drafts`; `result` and the default are unchanged. An unknown value shows Workspace access with that entry current. `area` is ignored outside the grants view.
- From 640 px: a wrapping row of whole entries, start-aligned, with one neutral separator under the block; each link at least 44 px high.
- Current entry: neutral background and semibold weight, no underline and no border change, matching the panel's selected history row. The bold width is reserved so that selecting an entry cannot re-wrap the row.
- Below 640 px: a stacked list of full-width links separated by neutral lines.
- Direction: logical properties only; the row mirrors in Arabic.
- Focus: the shared focus ring of the security screens applies to these links; the render check confirms it is not clipped.
- Leaving with unsent input: the links are hash links, and the shell's navigation guard intercepts hash changes while the panel reports itself dirty, asking "Discard unsaved input?". That is the shell's confirmation, not the panel's own discard prompt, which stays on the in-panel buttons. Read from `shared/navigation.ts` and `App.tsx`; the render check confirms it fires for these links.
- Client refusal: the navigation stays visible above the refusal sentence, so the user is not stranded.

## Focus on the primary button

The panel sets its focus outline to the current text color, and the primary button's text is white, so the ring on "Send proposal" and "Send decision" may be white on the page. A rule of equal specificity in the security stylesheet may or may not win, depending on load order. The panel's focus outline is set to the shared focus-ring color explicitly. This changes the result area as well; the impact is one outline color on focus and is included in the result area's re-render.

# Implementation items

Stable identifiers, scoped to M03. Each area `<a>` is `issuance-protected`, `issuance-terms` or `issuance-drafts`.

| Item | Purpose and behavior |
| --- | --- |
| `M03:authority-area:nav` | Five entries and their current states, desktop row and mobile stack |
| `M03:authority-area:unknown-area` | Fallback to Workspace access |
| `M03:authority-area:switch-guard` | The shell's confirmation when leaving with unsent input |
| `M03:<a>-authority:header` | Title and introduction |
| `M03:<a>-authority:client-refusal` | Refusal sentence with the navigation above it |
| `M03:<a>-authority:history` | Proposal history, ten rows per page, observed instant, next page |
| `M03:<a>-authority:inspection` | Full record of a selected proposal |
| `M03:<a>-authority:decision-review` | Approve, reject, withdraw or resolve an unknown decision, as the recorded actors allow |
| `M03:<a>-authority:proposal-fields` | Kind, capability from the server's policy list, validity, and for a person: recipient and ceiling |
| `M03:<a>-authority:proposal-review` | Read-only summary before sending |
| `M03:<a>-authority:attempts` | Submissions of this session for this area |

States of the reused panel, listed explicitly: loading; empty history; read error with technical reference; policy load error and retry; options load error and reload; validation summary; no eligible recipients; no matching active ceiling, which is the first-run state for a person permission; stale snapshot on the next page; refusals for hold, re-authentication and capacity; expired and superseded rows; action-start error; refused; outcome unknown with retry; busy; mobile list and detail panes.

Impact on the result area: the shared panel, the navigation and the focus color are its dependencies and change. Its copy, identifiers and behavior do not. Its review status is not known to this plan; the owner's review scope below includes its re-render.

# Copy

| Key | English | Korean | Arabic |
| --- | --- | --- | --- |
| area `issuance-protected` | Protected issuance settings permissions | 보호된 발행 설정 권한 | صلاحيات إعدادات الإصدار المحمية |
| area `issuance-terms` | Payment terms permissions | 납부 조건 권한 | صلاحيات شروط الدفع |
| area `issuance-drafts` | Issuance draft permissions | 발행 초안 권한 | صلاحيات مسودات الإصدار |
| protected `register_channel` | Register verification channel | 검증 채널 등록 | تسجيل قناة التحقق |
| protected `admit_channel` | Accept verification channel | 검증 채널 수락 | قبول قناة التحقق |
| protected `request_change` | Request protected settings change | 보호된 설정 변경 요청 | طلب تغيير الإعدادات المحمية |
| protected `verify_change` | Verify protected settings change | 보호된 설정 변경 검증 | التحقق من تغيير الإعدادات المحمية |
| protected `approve_change` | Approve protected settings change | 보호된 설정 변경 승인 | اعتماد تغيير الإعدادات المحمية |
| protected `inspect` | Inspect protected settings evidence | 보호된 설정 근거 조회 | عرض أدلة الإعدادات المحمية |
| term `record` | Record payment terms | 납부 조건 기록 | تسجيل شروط الدفع |
| term `approve` | Approve payment terms | 납부 조건 승인 | اعتماد شروط الدفع |
| term `bind` | Apply payment terms to a contract | 계약에 납부 조건 적용 | تطبيق شروط الدفع على عقد |
| term `approve_binding` | Approve applying payment terms to a contract | 계약에 납부 조건 적용 승인 | اعتماد تطبيق شروط الدفع على عقد |
| term `inspect` | Inspect payment terms evidence | 납부 조건 근거 조회 | عرض أدلة شروط الدفع |
| draft `submit` | Submit issuance draft for approval | 발행 초안 승인 요청 | إرسال مسودة الإصدار للاعتماد |
| draft `approve` | Approve issuance draft | 발행 초안 승인 | اعتماد مسودة الإصدار |
| draft `inspect` | Inspect issuance draft evidence | 발행 초안 근거 조회 | عرض أدلة مسودة الإصدار |

Introduction, one per area with its subject ("protected issuance settings", "payment terms", "issuance drafts"):

- English: "Manage proposals for console permission ceilings and individual permissions for payment terms. Each permission is time-limited and is granted only after another eligible governor approves the proposal. This is a synthetic internal workflow."
- Korean: "납부 조건에 대한 콘솔 권한 상한과 사용자별 권한 제안을 관리합니다. 권한에는 유효기간이 있으며, 다른 적격 관리자가 제안을 승인해야 부여됩니다. 합성 데이터 기반의 내부 절차입니다."
- Arabic: "أدر مقترحات سقوف صلاحيات وحدة التحكم وصلاحيات الأفراد الخاصة بشروط الدفع. لكل صلاحية مدة محددة، ولا تُمنح إلا بعد أن يعتمد المقترح مسؤول آخر مؤهل. هذا مسار داخلي ببيانات اصطناعية."

The exact validity limit is not written into copy; the proposal form shows the server's value. The client-refusal sentence per area keeps the result sentence's structure with the area's name.

Terminology notes:

- "Verification channel" follows the protected writes contract and the contracts screen's "verification through an existing registered channel"; "channel" alone already names webhook subscriptions and notification channels in the glossary.
- "Accept" is used for the admit step so that one panel does not use one verb for accepting a channel, approving a change, approving terms and approving a proposal. No existing user-facing label for this step was found in the contracts setup copy.
- "Bind" is shown as "apply to a contract"; the internal term is not user language.
- "Submit … for approval" is used because submitting a draft creates work awaiting approval and issues nothing.
- The Korean term for payment terms is 납부 조건, as in the glossary, the M01 labels, the drafts approval inbox and the planning fields. 지급 조건 is used in the contracts screen and its live copy, the review-and-issue screen, the credit cases evidence panels and the shared contracts finance copy. That inconsistency exists today, is outside this change and needs its own terminology decision.
- In the history row a capability label follows the permission type without a "Capability" caption, so "Person permission · Approve payment terms" above "Awaiting decision" could be misread as terms awaiting approval. The row is checked in context after rendering; changing the row would also change the result area and is a separate scoped change.
- These labels are also the natural labels for the HYB-346 action rows and are added to the glossary once reviewed. This is a role-based review, not native-language approval.

# Verification

Root, unit tests: the area parameter mapping including the unknown value; every capability the client lists for an area has a label in that area; the panel of each area issues only its namespace's commands; the proposal identifier prefix per area; an area switch resets the panel; the result area's existing tests unchanged.

Root, rendered checks, recorded with what was actually inspected:

- Navigation in English, Korean and Arabic at 1440, about 1280, near 1024 and 390 px: current state on an issuance area and on Workspace access, a keyboard focus capture, and computed border, outline and background values on the current and the focused link.
- Capability list for the protected area (six capabilities) and the payment terms area (longest labels) in the three languages at desktop and mobile widths; the draft area in at least one language.
- Per area: the history row with its longest label, the detail and the review summary.
- States: empty history, client refusal, unknown area fallback and no matching ceiling.
- A focus capture of the primary button with its computed outline color.
- The result area re-rendered and compared.

Owner: screen review of the three areas and the changed navigation as one declared scope, in the variants listed above, with the deferred items below. Nothing here records owner acceptance.

# Limits and deferred items

- History is unfiltered, forward-only and ten rows per page; with capabilities re-provisioned every 24 hours, pending proposals scatter across pages within days. The owner decision recorded in the reads contract (pending-only or pending-first view, pruning or capacity) is open, and the design review recommends settling it before real use. It is a deferred item of the owner's review scope.
- The folder and wrapper class still say `result-authority`; renaming them is a mechanical follow-up kept out of this change so the diff stays reviewable.
- P02 and the issuance setup screens do not link to these areas yet.
- Navigation grouping is revisited only if the areas exceed about seven.
- The frontend-design skill was not supplied to the design review, and no render existed when it ran.
- No implementation, render or review result beyond the source-based design review is claimed by this draft.
