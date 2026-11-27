---
type: DesignPlan
title: M03 revoke panel, person picker and ceiling table — composition
description: The ui-designer's composition for the last screen increment of the M03 transition, with its strings in three languages, recorded before implementation.
status: stable
---

# Scope and ownership

HYB-304 increment T5 of [the screen transition](m03-screen-transition.md), machine:mac_mini. Specified by the ui-designer role on 2026-11-26 by reading `Live.tsx` (`GrantForm`, `Revoke`, `Effects`) and the copy files; the role did not read the transition plan. Nothing was built or rendered, no external reference was consulted, and it is not owner acceptance. Root implements it in two parts: T5a (section A and C) and T5b (section B). Terminology follows the users screen: "login / 로그인 ID / اسم الدخول", "user ID / 사용자 ID / معرّف المستخدم".

# A. Revoke panel: delegations that would stop applying

The block is shown for a grant and for a delegation, from one read per opened panel. The existing "Delegations that end with it" and "No delegation was made from it." no longer say the truth (the Korean says "child", false for a grant; children can exist yet not be listed) and are removed if nothing else uses them.

Placement as today: after the impact paragraph, before the reason.

| Part | EN | KO | AR |
| --- | --- | --- | --- |
| Heading | Delegations that would stop applying | 함께 적용되지 않게 되는 위임 | تفويضات ستتوقف عن السريان |
| Count beside it | 1 delegation / {n} delegations | {n}건 | تفويض واحد / تفويضان / {n} تفويضات / {n} تفويضًا |
| Lead, grant | Delegations that apply now and would stop applying if this revocation is approved: those this grant backs, and those its holder received if this is their last grant here. | 이 회수가 승인되면 적용되지 않게 되는, 지금 적용 중인 위임입니다. 이 권한이 뒷받침하는 위임과, 이 권한이 보유자의 마지막 권한일 때 보유자가 받은 위임이 포함됩니다. | التفويضات السارية الآن التي ستتوقف عن السريان إذا اعتُمد هذا الإبطال: ما يستند إلى هذه الصلاحية، وما تلقّاه صاحبها إن كانت آخر صلاحية له هنا. |
| Lead, delegation | Delegations that apply now and would stop applying when this delegation is revoked. | 이 위임을 회수하면 적용되지 않게 되는, 지금 적용 중인 위임입니다. | التفويضات السارية الآن التي ستتوقف عن السريان عند إبطال هذا التفويض. |
| Empty | No delegation that applies now would stop applying. | 지금 적용 중인 위임 중 함께 적용되지 않게 되는 위임은 없습니다. | لا يوجد تفويض سارٍ الآن سيتوقف عن السريان. |
| More | Show {n} more | {n}건 더 보기 | عرض {n} أخرى |
| Loading | Checking which delegations would stop applying… | 함께 적용되지 않게 되는 위임을 확인하는 중… | جارٍ التحقق من التفويضات التي ستتوقف عن السريان… |
| Failure | The delegations that would stop applying could not be checked. Revoking still works; the delegations that stop are listed with the revocation in the Revocations section. | 함께 적용되지 않게 되는 위임을 확인하지 못했습니다. 회수는 그대로 진행할 수 있으며, 함께 끝난 위임은 '회수' 섹션의 해당 회수에 표시됩니다. | تعذّر التحقق من التفويضات التي ستتوقف عن السريان. يمكن الإبطال مع ذلك، وتظهر التفويضات المنتهية مع الإبطال في قسم «الإبطال». |

- A small secondary line under the lead says when it was checked (the existing "Checked at" wording): for a grant the list reflects now, while the revocation takes effect only when approved.
- Identifiers: a list, one mono identifier a line, in the server's order; the first ten, the rest behind a disclosure "Show {n} more".
- Loading is a status line; the failure sits inside the block with "Try again".
- The submit button stays usable and does not wait for the read: a revocation is a safety action and must not be blocked by an informational read; the revocation's record later lists what actually ended.

# B. Person picker (request form and effect check)

A labelled search field with a Search button (Enter also searches and never submits the form), a status line, results as a list of buttons, and a chosen line with "Change"; the pattern of the P04 owner search. Not a combobox (fragile across screen readers and right-to-left text) and not a search with a select (a select cannot show the flag on each row).

Order inside the field: label (the existing "Person"); hint; error; chosen line; search row; status line; results.

| Part | EN | KO | AR |
| --- | --- | --- | --- |
| Hint | Search by login or user ID. | 로그인 ID 또는 사용자 ID로 검색합니다. | ابحث باسم الدخول أو معرّف المستخدم. |
| Searching | Searching… | 검색하는 중… | جارٍ البحث… |
| One page | {n} people | {n}명 | عدد الأشخاص: {n} |
| First page, more exist | Showing the first 25 by user ID. Search to find others. | 사용자 ID순으로 처음 25명을 표시합니다. 다른 사람은 검색하세요. | يُعرض أول 25 حسب معرّف المستخدم. ابحث للعثور على غيرهم. |
| Search, more than a page | More people match than are shown. Type more of the login or user ID. | 표시된 것보다 일치하는 사람이 더 있습니다. 로그인 ID나 사용자 ID를 더 입력하세요. | يطابق أشخاص أكثر مما هو معروض. اكتب المزيد من اسم الدخول أو معرّف المستخدم. |
| No match | No one matches “{q}”. | “{q}”와 일치하는 사람이 없습니다. | لا يطابق «{q}» أحدًا. |
| Only you (request form) | Only you match “{q}”. You can't request access for yourself. | “{q}”와 일치하는 사람은 본인뿐입니다. 자신에 대한 권한은 요청할 수 없습니다. | لا يطابق «{q}» غيرك، ولا يمكنك طلب صلاحية لنفسك. |
| Failure | People could not be searched. | 사람을 검색하지 못했습니다. | تعذّر البحث عن الأشخاص. |
| Failure, a person chosen | The person you chose is kept. | 선택한 사람은 그대로입니다. | يبقى الشخص الذي اخترته كما هو. |
| Change | Change | 변경 | تغيير |
| Keep | Keep this person | 이 사람 유지 | الإبقاء على هذا الشخص |
| Results label | Matching people | 검색된 사람 | الأشخاص المطابقون |
| Request form, flag | A revocation for this person is not confirmed; see the Revocations section. | 이 사람에 대한 회수가 확정되지 않았습니다. '회수' 섹션을 확인하세요. | إبطال يخص هذا الشخص غير مؤكَّد؛ راجع قسم «الإبطال». |

- Before any search the first page is shown (25, in user-identifier order), so a small workspace needs no typing; there is no minimum to type. More than a page asks to refine; there is no Next.
- The request form hides the signed-in person's row; if the first page holds no one else, the existing "no eligible person" text is kept and the search hidden. The effect check may list the signed-in person.
- A result row is a button: the login (the user identifier when there is none), then the user identifier in mono, then, when the flag is set, the existing "Revocation not confirmed" in secondary bold.
- The chosen line: the login in bold, the mono user identifier, "Change"; while changing, "Keep this person". When the flag is set a line follows: in the effect check the existing unconfirmed-preview sentence, replacing today's line; in the request form the new sentence above.
- Failure is an alert with "Try again"; a chosen person is kept.
- Keyboard and ARIA: the input is a search input described by the hint, the status and the error; choosing a result moves focus to "Change"; "Change" moves focus to the input; after a search focus stays in the input. The effect check clears its result when the person changes, as today.
- Sizes as P04: the search row up to 560px, input 44px tall growing from 24ch, button at its natural width; results up to 560px wide and 352px tall with neutral borders, rows at least 44px; at 390 the input takes the row and the button wraps; no native clear button.

# C. Console ceiling table

Removed; the permissions list's "Console" rows replace it and nothing else is needed. Its lead text pointed at the table and is removed. The request form gains a hint under the Access field: "Choices are limited to what the console itself may do: the Console rows in the list above. / 콘솔 자체가 할 수 있는 범위로 제한됩니다. 위 목록의 '콘솔' 행을 참고하세요. / تقتصر الخيارات على ما يُسمح لوحدة التحكم نفسها بفعله: صفوف «وحدة التحكم» في القائمة أعلاه." The existing days hint already states the ceiling's end date.

# Left open by the specification, decided by root

- **Whether approving a grant's revocation re-reads the list.** It does not: the approval panel already shows the request's own "may end" list from the requests read, and the block here carries the time it was checked.

# To capture for the rendered review

English, Korean and Arabic at 1440 (panel) and 390. A: a grant panel with two identifiers; a delegation panel with twelve (ten and "Show 2 more"); empty; loading; a failed read with the submit still enabled. B, for the request form and the effect check: the form opened with the first page; a search with three matches, one flagged; more than a page; no match; "only you"; a failure with a person chosen; a chosen person with the flag in each place; focus on "Change" after choosing by keyboard. C: the view without the ceiling section; the request form with the Access hint.
