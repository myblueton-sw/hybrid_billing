---
type: ImplementationPlan
title: Protected issuance setup and payment terms on O03
description: Composition, items, copy proposals and delivery slices for wiring the protected setup, statement and payment term reads and commands into the O03 documents and payment terms views.
status: draft
---

# Scope and evidence

HYB-341, machine:mac_mini. This is the screen part of items 1 to 4 of [the handoff](handoff.md). The reads and commands are merged and no screen uses them: protected setup discovery ([contract](issuance-protected-setup-discovery-contract.md)), statements and the sample reference catalog ([contract](issuance-setup-evidence-contract.md)), the seven protected writes ([contract](issuance-protected-writes-contract.md)) and payment terms ([contract](issuance-payment-terms-contract.md)). The console decoders and clients exist under `organizations/pages/contracts/setup` (`protected.ts`, `attestation.ts`, `protected-writes.ts`, `terms.ts`, `client.ts`).

Root owns planning, implementation and integration. The ui-designer role produced the composition on 2026-11-24 from source and contract documents, without renders and without browsing; the external references below were fetched by root and given to it as summaries. Sections marked "root" are root decisions that differ from or add to the proposal. This plan is not rendered verification, native-language review or owner acceptance, and it grants no deployment or real-account permission. No accepted (FIX) record exists for O03 under `docs/ui/acceptance`, so no fixed revision is changed.

# Facts the composition is built on

Read from the contracts and the merged decoders.

- Every read and command is scoped to the selected contract, one confirmed configuration and that configuration's digest.
- Every read, and every receipt of a protected or term write, carries `current_eligibility` `unchecked`; a statement receipt carries `assurance` `human_attestation` instead. Nothing on the screen is called ready, eligible, verified, mature, usable or proven.
- Every record keeps three layers apart: the retained state, which may be an unknown state; what the module holds now; and an actor the service pinned before an effect the module does not yet show.
- A statement is an accountable claim by its author. An unused or withdrawn statement supports nothing.
- No read returns what the current user may do for these commands. A refusal code after sending is the only guidance.
- Every read on these views needs the protected Inspect right or the terms read right. A person who holds only a step capability can call the command but cannot read the rows this screen needs to choose a target.
- The service refuses the registrar admitting their own channel, the requester verifying or deciding their own change, the verifier deciding it, and a term's or binding's author approving it. It also refuses a channel's registrar or admitter requesting or verifying through that channel, and deciding; the row of a change does not name them, so that case appears only as a refusal.
- Reads are allowed during an installation hold and every write is refused. Every write is subject to step-up.
- These cannot be undone: the first protected rules proposal of the installation; the net days of a recorded term; a binding, which occupies its dates for the contract, permanently when it has no end date; a consumed statement. A change left in an unknown state blocks its subject and field.
- Protected rules are installation-wide and are reached through a contract.
- A change request needs a channel admitted at least the rules' channel age earlier. The channel row states `request_not_before` only for a channel admitted under the current rules.
- References are thirteen sample codes. The user never types an identifier, fingerprint or digest; the console generates new identifiers.

# External references

Fetched on 2026-11-24. The GOV.UK Tabs page could not be fetched (certificate error) and is not used.

| Source | Observed | Adopted | Rejected |
| --- | --- | --- | --- |
| [GOV.UK Task list](https://design-system.service.gov.uk/components/task-list/) | Each task has a name and a status; not for a service "that needs to be completed in a specific order"; "should not be used as a way of showing users their answers" | Nothing | The component: the setup is order-dependent, done by several people, and no status of completion is known to the screen |
| [GOV.UK Complete multiple tasks](https://design-system.service.gov.uk/patterns/complete-multiple-tasks/) | For transactions over several sessions; "split it up into steps"; statuses on every task | Splitting the work into separately addressable parts | Per-part statuses; only recorded counts are shown |
| [GOV.UK Summary list](https://design-system.service.gov.uk/components/summary-list/) | Key and value facts in a `dl`; not for tabular data; summary cards for "multiple summary lists that all describe the same type of thing"; row action links need hidden context text | `dl` for the rules and the term; three cards for the three protected fields; hidden context on repeated row actions | Its styling |
| [GOV.UK Check answers](https://design-system.service.gov.uk/patterns/check-answers/) | One check page immediately before confirming; the submit button "clearly shows the action it performs"; nothing is complete until confirmed | The existing inline check step of the documents view, with headed groups | The two-thirds page width |
| [MOJ Timeline](https://design-patterns.service.justice.gov.uk/components/timeline/) (status "To be reviewed") | A record of events in date order with title, date and who; most recent first; not a task list or progress indicator | An event list inside a change row | Using it to show progress or pending work |
| Carbon data table and the check-answers evidence recorded in [the draft journey plan](issuance-draft-journey-plan.md) | Already adopted for the configuration history | Paged tables with stacked labelled rows on narrow panels | No change |

None of these sources gives small-screen rules. The responsive rules below are this plan's own and are verified by render.

# Information architecture

## Documents view

The view gains a second-level navigation with five parts, selected by a `part` query parameter. An unknown value shows the first part.

| `part` | Part | Content |
| --- | --- | --- |
| absent or `settings` | Issuance settings | The existing five sections, unchanged |
| `rules` | Protected rules | Rules block |
| `channels` | Verification channels | Channel table |
| `changes` | Protected settings changes | Three field cards and the change table of one field |
| `statements` | Statements | Statement table |

Reasons: the work spans days and at least five people, each returning for one step; four paged tables on one page lose their place when reads reload after a write; and the pattern guidance asks to split many tasks into steps. The navigation reuses the structure fixed for M03 in [the authority screen plan](issuance-authority-screen-plan.md): a labelled `nav` with a list of links, the current link marked with `aria-current="page"`, neutral background and semibold weight, a wrapping row from 640 px and a stacked list below it, 44 px targets. It shows no status. Beside a part's name it may show the recorded count from the summary read, and shows none when that read is denied or failed.

## Scope (root)

The proposal carried the configuration in the URL. No read returns one configuration by identifier, and the digest must come from the history read, so the selected scope is kept in session memory per contract instead, like the unsent input of the documents view.

- A history row in state `confirmed` that was recorded with setup context gets the action "Use these settings", with the configuration named in hidden text; the same selection scopes the payment terms view. The merged O03 copy calls these records issuance settings, so the action does too.
- A row recorded without setup context is not offered. The discovery contract makes it readable, but no contract decides whether the writes support it. This is listed as an open question.
- When the first history page is the whole history and holds exactly one such row, it is selected automatically.
- A scope bar above the navigation of parts 2 to 5, and at the top of the payment terms view, shows the setup date, the recorded contract revision and billing profile version, the state, and the identifier and digest under "Details".
- Without a scope, parts 2 to 5 and the payment terms view show one sentence and a link to the settings part. A full page reload loses the selection, as it loses unsent input.
- Changing contract, date or session clears the scope.

## Payment terms view

One page replaces the "Not connected" placeholder: scope bar, term, bindings of the contract, results of this session.

# Shared frame

- Each part has a heading with Reload, a purpose line of at most 72 characters per line, and a meta line "Read at <time> · Current eligibility not checked".
- Spacing, body size, container query, tables and forms follow `documents.css`: 32 px between sections, 16 px inside, 16 px body, forms within 40 rem, the 56 rem container breakpoint, stacked labelled rows below it.
- The three layers appear the same way everywhere, as three labelled lines: "Recorded state" with a neutral tag, "Observed now" in plain text, and "Pinned, not yet observed" with the actor, rendered only when present.
- An unknown state is written out, for example "Registration outcome unknown". It never uses success or failure wording and is never conveyed by color.
- Read states reuse the existing feedback block: loading, denied, stale, failed, each with its code. A denied read stays inside its section and says: these records could not be read; reading them may need the permission that M03 lists as "Inspect protected settings evidence" (for the payment terms view, the Inspect permission of the payment terms area); without them no step can be started from this screen. The refusal also covers a configuration that is unknown to the service, so the sentence says "may". It makes no claim about what the user holds.
- When the session user is recorded on a record as an actor the next step excludes, the action of that step is replaced by a fact, for example "You registered this channel. The service refuses acceptance by the registrar." The comparison uses the row's own fields against the session user: `rules_proposer`, `registrar`, `requester`, `verifier` and the approvers of recorded decisions, `term_author`, and a binding's `author`. This states a recorded fact and a rule, not a permission.
- People are shown by identifier, as the configuration history already shows them; no read on this view returns a person's name. Identifiers, instants, dates, codes and numbers of days are isolated left-to-right.

# Parts

## Protected rules

A summary list. First group: approvals required, verification validity, maximum lead time and minimum channel age, shown in hours or days with the exact seconds under details. Second group: the three layers, then proposed by and at, confirmed by and at. A fixed line states that the rules apply to the whole installation and are reached here through this contract.

- `none` with `absent`: "No protected rules recorded" and the Propose action.
- `none` with `unowned`: "The module holds rules this service did not record"; no values, no action.
- `proposed`: Confirm, or the proposer fact.
- An unknown state: recovery under the rule of [Recovery](#recovery).

## Verification channels

Purpose line: channels through which a change can be verified; a change request needs a channel accepted at least the minimum channel age earlier. A counts line gives the total and, separately, records of another origin and module entries without a record, both "not listed".

Table columns: Channel (subject kind and channel kind; identifier under details) | Record state | People and times (registered, accepted, retired) | Earliest change request | Action. The fourth column shows `request_not_before` or "Not stated", with a note under the table that it is stated only for a channel accepted under the current rules. There is no countdown. Row actions: Accept, or recovery for an unknown state, or the registrar fact; a retired channel has no action. Register sits under the table.

## Protected settings changes

Three summary cards, one per field (document recipient, document template, payment account): subject kind, changes recorded, other origin, module entries without a record, and the latest scheduled change with its effective time, captioned as recorded history and not a statement that a draft can use it. Each card has one action, "Show changes".

The table of the selected field, in retained order: Change (provenance, instruction kind; identifier under details) | Record state (phase, recorded state, observed now, pinned actors) | Takes effect | Verification channel | Action. A row disclosure "Events" lists observed facts, most recent first: requested, verification with the verifier's stated outcome, each decision, ended. Pinned actors stay in the record state cell. Notices in words: a module entry without a record refuses later requests for the field; a change in an unknown phase blocks further requests for its subject and field.

The decision deadline depends on the validity fixed into the change at request time, which the row does not return. The screen shows the effective time only.

## Statements

Purpose line: accountable claims written by the people who acted; a statement is its author's claim, not proof. One table: Author | Recorded | Purpose | Names (record type and identifier) | State | Action. States: "Not used — supports no record", "Used", "Withdrawn — supports no record". The statement text and the locator are in a row disclosure. Withdraw is offered only on the session user's own unused statement.

Channel and change rows embed no statement. No read returns statements by target, and joining them would mean walking up to a hundred pages.

## Payment terms

Term, as a summary list: "Net <N> calendar days", revision, the three layers, author and written at, approver and approved at. Empty: "No payment terms recorded for this configuration" and Record.

Bindings, as a table led by "Every binding of this contract is listed, because each occupies its dates": From and to ("No end date") | Net days | Time zone | Reason | Origin (this configuration, another configuration with its identifier, billing profile) | Record state | People and times | Action. Approve is offered only on a row made under the selected configuration; a binding of another configuration or of a billing profile is listed without an action. The configuration's document zone is shown above the table as the zone a new binding uses.

# Write flows

All flows use the protocol the documents view already has: input is frozen, a check step shows it, one request is sent, the outcome is kept for the session, a request whose response never arrived is retried unchanged, reads reload after settlement. A write or a lapse changes the snapshot of a paged read, so paging restarts at the first page after a reload. A row action selects the target and moves focus to one step form under the table; the form starts with the target's facts.

## Statement and action (root confirms the proposal)

A step that needs a statement is one flow with two visible requests. The statement is written in the form of the step it supports, because it is bound to one target and, for acceptance and verification, to the target's digest, and because a separate statement screen would produce unused claims against the limit of ten per person. The check step has two groups, "Statement you will record" and "Action it supports", and says that two requests are sent, that the statement is recorded first and stays on record even if the action is refused, and that it can be withdrawn only while unused. The outcome shows both lines.

- The action is sent only after the statement request has settled as recorded. A statement request with an unknown outcome is settled first, by the unchanged retry or by recovery; the action waits.
- Action cleanly refused, with no earlier attempt of the same operation left unknown: the entry reads "Statement recorded, not used" and offers Withdraw. A new attempt of the step in this session reuses that statement when the target and, where bound, the target digest are the same.
- Action with an unknown outcome: the service may have marked the statement used in the same step that reserved the record. The entry reads "Statement recorded; whether it was used is not known", offers no Withdraw, and offers only the unchanged retry or recovery of the action.
- Reuse is limited to a statement recorded in this session. No read returns statements by target or the digest a statement was bound to, so an unused statement from an earlier session is visible only in the statements part and cannot be attached to a step from this screen. Open question 2 covers this.

## Recovery

Two different things follow an unknown outcome, and the documents view already has both.

- Unchanged retry: the same request with the same key, for a request whose response did not arrive. The server replays a stored result for that key, including a stored uncertain error, so this alone never completes a record the server left in an unknown state.
- Recovery: a new request with a new key and the same identifiers and inputs, labelled "Recover recorded attempt" as in the merged copy. The service accepts it only from the actor of the unknown step: the proposer, registrar, requester or author for a first step left unknown; the pinned confirmer, admitter, verifier or approver for those steps; the actor of the pending decision.

The rows do not return the inputs a recovery must repeat: the reason, the sample reference code, the instruction form and its channel, the statement identifier, the pending outcome or verdict. Recovery is therefore offered only when both hold: the session user is the actor named above for the record's unknown state, and the session still holds the frozen input of the original request. Otherwise the row shows the unknown state, the pinned actor and the blocking notice as facts, with no action. Recovery from another session needs a data contract and is open question 10. Which inputs the server compares was read from the contracts, not from server code.

## Fields

| Step | Fields in order | Notice at the decision |
| --- | --- | --- |
| Propose rules | Approvals (1, 2 or 3); validity, maximum lead time, minimum channel age as a whole number with a unit of hours or days, no defaults, bounds as hints; reason | The first proposal cannot be replaced or discarded for this installation |
| Confirm rules | Recorded values; reason | None |
| Register channel | Subject kind; channel kind; sample reference (the two samples of that subject, marked as samples); statement; where the source is kept (optional); reason | None |
| Accept channel | Target facts; statement; where the source is kept | A used statement stays used |
| Request change | Sample reference; verification channel with each option's earliest request time; provenance; instruction (in person, through a registered channel with a second channel that is not the verification channel, unregistered sender); effective date and time, hinted as at least one minute ahead and within the maximum lead time of the recorded rules; statement; where the source is kept; reason | None |
| Verify change | Outcome (confirmed through the channel, denied); statement; where the source is kept | A used statement stays used |
| Decide change | Verdict; closed reason when rejecting | None |
| Record term | Net days, a whole number from 0 to 365 with no placeholder and no suggestion; reason | Net days cannot be corrected after recording; this configuration holds one term |
| Approve term, approve binding | Target facts only | None |
| Apply term to the contract | From date; an explicit choice between an end date and no end date, neither preselected; reason (initial, renegotiation, correction); time zone, read-only | A binding cannot be withdrawn; without an end date it occupies every later date of this contract |

The verification channel choice lists every channel of the scope, which means reading every channel page, at most ten.

The statement hint asks the person to describe what they did and not to enter account, contact or identifier values.

The effective date and time are entered and shown in UTC, labelled as UTC, as every instant on this view already is (root; open question below).

## After sending

- Success: "Recorded. Current eligibility was not checked.", with the receipt state.
- Clean refusal: "Refused. This request recorded nothing.", the code, and a sentence per code. For `evidence_stale` the sentence also covers a statement that cannot be used and a separation refusal that arrives under this code. For `prerequisite_failed` it lists the possible causes for the step without asserting one: channel age, the one-minute margin of request, verification and decision, the limit of ten unused statements per person and the total statement limit, overlapping dates, a time zone that differs from the contract's existing bindings, and capacity. During an installation hold the sentence says that writes are refused while reads continue.
- Unknown: the existing "Outcome unknown" entry with the unchanged retry, and recovery under the rule above.

The check button of these flows is not the merged "Check permission and review": no permission is checked for these commands before sending. It reads "Review before sending".

# Responsive and direction rules

- 1440 px: tables full width; summary lists in two columns; field cards three across; forms within 40 rem; number and date inputs 8 to 14 rem.
- 1024 px: expected below the 56 rem container breakpoint and therefore stacked; confirmed by render.
- 390 px: stacked labelled rows, one card per row, navigation as a vertical list, content-width buttons of at least 44 px.
- Logical properties only. No colored border, edge strip, radius or shadow; state by text.

# Implementation items

`o03-documents:` `nav:parts`; `scope:select`, `scope:bar`; `rules:summary`, `rules:layers`, `rules:installation-notice`, `rules:action:propose`, `rules:action:confirm`; `channels:table`, `channels:counts`, `channels:not-before`, `channels:paging`, `channels:action:register`, `channels:action:accept`; `changes:field-cards`, `changes:table`, `changes:events`, `changes:blocking-notice`, `changes:paging`, `changes:action:request`, `changes:action:verify`, `changes:action:decide`; `statements:table`, `statements:text`, `statements:action:withdraw`; `references:sample-labels`; `flow:statement-and-action`, `flow:check`, `flow:outcomes`, `flow:recovery`; `state:loading`, `state:denied`, `state:stale`, `state:failed`, `state:no-scope`; `fact:recorded-by-you`.

`o03-payment-terms:` `scope:bar`; `term:summary`, `term:layers`, `term:action:record`, `term:action:approve`; `bindings:table`, `bindings:paging`, `bindings:action:apply`, `bindings:action:approve`; `flow:check`, `flow:outcomes`, `flow:recovery`; `state:loading`, `state:denied`, `state:stale`, `state:failed`, `state:no-scope`, `state:empty`.

Every item starts as planned and not implemented. Design review status and implementation status are tracked per slice in this document.

# Delivery slices

Each slice is one branch, one review and one merge, and each ends with a rendered check in English, Korean and Arabic at 1440, 1024 and 390 px with intercepted synthetic responses.

| Slice | Content | A user can then |
| --- | --- | --- |
| 1 | Part navigation, scope selection, scope bar, no-scope state | Choose a confirmed configuration and move between parts |
| 2 | Rules, read-only | See the installation's protected rules |
| 3 | Channels, read-only | See the verification channels |
| 4 | Field cards and the change table with events, read-only | See the summary and every change of each protected field |
| 5 | Statements and sample labels, read-only | See statements apart from the records they name |
| 6 | Payment terms view, read-only, with the Korean view label changed to 납부 조건 in `live-copy.ts` and the screen records in the same change | See the term and every binding of the contract |
| 7 | Record and approve term; apply and approve binding | Complete payment terms |
| 8 | Propose and confirm rules | Establish the rules |
| 9 | Register and accept channel, with the statement flow and Withdraw | Establish a channel |
| 10 | Request, verify and decide change | Schedule a protected reference |

Root moved the term writes ahead of the protected writes: they need no statement and no waiting period, so they prove the write frame before the statement flow is built on it. Slices 7 to 10 carry mutations and take an independent source review of the client boundary; every slice takes the ui-designer render review and the technical-writing review of its new copy.

# Copy proposals

Proposals for the technical-writing review; not final copy. The register, accept, request, verify, record, approve and apply actions reuse the capability labels of the M03 section of the glossary without change. The decision step has no capability label of its own that fits both verdicts: its row action is the neutral "Decide change" and its two choices use the approve (승인, اعتماد) and reject (반려, رفض) wording already merged in the M03 provisioning panel.

| Concept | English | Korean | Arabic |
| --- | --- | --- | --- |
| Part 1 | Issuance settings | 발행 설정 | إعدادات الإصدار |
| Part 2 | Protected settings rules | 보호된 설정 규칙 | قواعد الإعدادات المحمية |
| Part 3 | Verification channels | 검증 채널 | قنوات التحقق |
| Part 4 | Protected settings changes | 보호된 설정 변경 | تغييرات الإعدادات المحمية |
| Part 5 | Statements | 진술 | الإفادات |
| View | Payment terms | 납부 조건 | شروط الدفع |
| Bindings | Payment terms applied to this contract | 이 계약에 적용된 납부 조건 | شروط الدفع المطبَّقة على هذا العقد |
| Layer 1 | Recorded state | 기록된 상태 | الحالة المسجَّلة |
| Layer 2 | Observed now | 현재 관찰된 상태 | المرصود حالياً |
| Layer 3 | Pinned, not yet observed | 지정됨, 아직 관찰되지 않음 | معيَّن، لم يُرصد بعد |
| Unknown state | Registration outcome unknown | 등록 처리 결과 미확인 | نتيجة التسجيل غير معروفة |
| Channel registered | Registered, awaiting acceptance | 등록됨, 수락 대기 | مسجَّلة، بانتظار القبول |
| Channel admitted | Accepted | 수락됨 | مقبولة |
| Change requested | Requested | 요청됨 | تم طلبه |
| Change after verification | Verification recorded, awaiting decision | 검증 기록됨, 결정 대기 | سُجِّل التحقق، بانتظار القرار |
| Change scheduled | Scheduled | 예정됨 | مجدول |
| Change rejected | Rejected | 반려됨 | مرفوض |
| Change lapsed | Lapsed | 기한 경과 | انقضت المهلة |
| Statement unused | Not used | 미사용 | غير مستخدمة |
| Statement used | Used | 사용됨 | مستخدمة |
| Statement withdrawn | Withdrawn | 철회됨 | مسحوبة |
| Withdraw | Withdraw statement | 진술 철회 | سحب الإفادة |
| Decide | Decide change | 변경 결정 | البتّ في التغيير |
| Scope action | Use these settings | 이 설정 사용 | استخدام هذه الإعدادات |
| Scope bar | Selected issuance settings | 선택한 발행 설정 | إعدادات الإصدار المحددة |
| Recovery | Recover recorded attempt | 기록된 시도 복구 | استعادة المحاولة المسجلة |
| Sample reference | Sample recipient A | 샘플 수신자 A | مستلم تجريبي A |

Sample labels follow one form for the thirteen codes: "Sample" with the use (recipient, document template, payment account, customer channel, document kind channel, entity channel) and the letter, and "Sample unregistered sender".

Conflicts the review must settle:

- 지급 조건 and 납부 조건. The O03 view label and the baseline screen use 지급 조건; M03, M01, the draft inbox and planning use 납부 조건. Proposal: 납부 조건 on this view, changed in slice 6. The payer labels (지급 주체, 지급자) are a separate concept and do not follow.
- 확인 and 검증. Proposal: 확인 for confirming settings and rules, 검증 for verifying a change or a channel, never mixed.
- The module state `confirmed` of a change and the verification outcome `confirmed` collide with the settings state and with the rule against calling anything verified. The table above avoids both words for the change state.
- The command decides and may reject, while the capability is labelled as approving.
- The M03 Inspect label says evidence (근거). A statement must not be called evidence on this view.
- The parties view already has a section called Terms next to Payment terms.

# Limits and open questions

For the owner or a data contract; current behavior is as stated above until decided.

1. A person with a step capability and no Inspect right can do nothing on this screen. Either the governor grants Inspect with every step capability, which is what the screen tells the user, or a minimal target read is designed.
2. Statements cannot be shown on the record they support, and an unused statement of an earlier session cannot be reused, without a read by target that returns the bound digest.
3. A confirmed configuration recorded without setup context is not offered as scope.
4. The zone in which the effective date and time are entered: UTC here; the document zone is the alternative.
5. The decision deadline of a change is not shown.
6. A governor with no contract that has a confirmed configuration cannot reach the rules.
7. A term has no display name; the screen names it by its net days.
8. A terms reader without the protected Inspect right cannot obtain a scope, because the configuration history needs that right.
9. The selected scope does not survive a full reload.
10. A record left in an unknown state cannot be recovered from a session that does not hold the original input, because no read returns that input.

Not verified at this stage: computed borders of the reused feedback, tag and note classes on these parts; the content width at 1024 px; every label in context. These are checked by render in each slice.

# Slice status

| Slice | Implementation | Checks and review |
| --- | --- | --- |
| 1 | Implemented: `setup/scope.ts`, `Parts.tsx`, `parts.css`, `part-copy.ts`, an Action column in the settings history, the `part` parameter in `LiveDetail.tsx`. Items `nav:parts`, `scope:select`, `scope:bar`, `state:no-scope` | Five scope tests; render check `tools/o03-setup-parts-qa.mjs`, 54 cases in English, Korean and Arabic at 1440, 1024 and 390 px with intercepted synthetic reads; ui-designer render and copy review recorded below |
| 2 | Implemented, read-only: `setup/Rules.tsx`, `rules-copy.ts`, `part-store.ts`. Items `rules:summary`, `rules:layers`, `rules:installation-notice`, and the loading, denied, stale and failed states of this part. No action; propose and confirm arrive in slice 8 | Five tests for the part read store and the duration text; render check `tools/o03-setup-rules-qa.mjs`, 99 cases: seven rule states and four refusals in English, Korean and Arabic at 1440, 1024 and 390 px; ui-designer render and copy review recorded below |
| 3 | Implemented, read-only: `setup/Channels.tsx`, `channels-copy.ts`, `page-store.ts`. Items `channels:table`, `channels:counts`, `channels:not-before`, `channels:paging`. No action; register and accept arrive in slice 9 | Three tests for the paged read store; render check `tools/o03-setup-channels-qa.mjs`, 63 cases: seven channel states in one table, the empty state, two pages and four refusals in English, Korean and Arabic at 1440, 1024 and 390 px; ui-designer render and copy review recorded below |
| 4 | Implemented, read-only: `setup/Changes.tsx`, `changes-copy.ts`. Items `changes:field-cards`, `changes:table`, `changes:events`, `changes:blocking-notice`, `changes:paging`. No action; request, verify and decide arrive in slice 10 | Render check `tools/o03-setup-changes-qa.mjs`, 72 cases: eight change states in one table with opened events, switching the protected setting, the empty state, two pages, four refusals of the change read and a refused summary read, in English, Korean and Arabic at 1440, 1024 and 390 px; ui-designer render and copy review recorded below |
| 5 | Implemented, read-only: `setup/Statements.tsx`, `statements-copy.ts`. Items `statements:table`, `statements:text`, `references:sample-labels`. No action; Withdraw arrives in slice 9 | Two tests for sample labels and state wording; render check `tools/o03-setup-statements-qa.mjs`, 72 cases: five statements with opened text, the sample reference list, the empty state, two pages, four refusals of the statement read and a refused sample read, in English, Korean and Arabic at 1440, 1024 and 390 px; ui-designer render and copy review recorded below |
| 6 | Implemented, read-only: `setup/PaymentTerms.tsx`, `terms-copy.ts`; the payment terms view of `LiveDetail.tsx` no longer shows the not-connected placeholder; the Korean view title is 납부 조건. Items `o03-payment-terms:scope:bar`, `term:summary`, `term:layers`, `bindings:table`, `bindings:paging` and the loading, denied, stale, failed, no-scope and empty states. No action; the term writes arrive in slice 7 | Two tests for the net-days text; render check `tools/o03-payment-terms-qa.mjs`, 99 cases: five term states, five applications in one table, two pages, four refusals, a refused settings history, no scope and choosing one, in English, Korean and Arabic at 1440, 1024 and 390 px; ui-designer render and copy review recorded below |
| 7 | Implemented: `setup/step-store.ts` (session store for step commands: one unresolved request at a time, unchanged retry, recovery under a new key, unsent input kept per form), `term-flow.ts` (which step is offered in which state and the four intents), `TermWrites.tsx`, an Action column in the applications table. Items `term:action:record`, `term:action:approve`, `bindings:action:apply`, `bindings:action:approve`, `flow:check`, `flow:outcomes`, `flow:recovery` | Seventeen tests for the store and the term flow; render check `tools/o03-payment-terms-writes-qa.mjs`, 27 flows against an intercepted service that keeps state: record with an invalid then a valid entry, approve, apply without an end date, approve an application, a clean refusal, an unknown outcome while the read fails (the result, the retry and the unload guard remain), steps disabled until it is settled, the unchanged retry and a recovery, in English, Korean and Arabic at 1440, 1024 and 390 px; independent source review and ui-designer review recorded below |
| 8 to 10 | Not implemented | None |

Slice 1 differs from the plan in three ways, each accepted by the render review: the scope action and bar say "issuance settings"; the navigation stacks by a container query at 40 rem of the documents panel, because the panel sits beside the contract list, where the plan gave 640 px; and parts 2 to 5 use the existing not-connected wording until their slices land. The navigation shows no counts yet.

Slice 2 shows a pending or unknown rules record before its values and heads the values "Proposed rule values — not confirmed", so that unconfirmed values are not read as the rules in force. Durations are shown in the largest whole unit and never rounded, with exact seconds under the details. The panel beside the contract list is narrower than 56 rem at 1440 px, so the two groups stack there and sit side by side at 1024 px, the reverse of what the plan expected. A lighter scope bar was suggested by the review and is left for the owner's screen review.

Slice 3 has four columns and no Action column until slice 9. Its table stacks below 44 rem of the panel, so the panel at 1440 px shows the table. The note about the earliest change request time sits above the table.

Slice 4 shows the first protected setting's changes on arrival and marks its card "Shown below"; the other cards carry "Show changes", which moves focus to the table heading. The events list is numbered in the order the events happened and shows the latest first. A refused summary read leaves the change table usable, and a refused change read leaves the cards. A request whose outcome is unknown and that is not observed is listed as attempted, not as requested. The step outcome line appears only for an unknown phase, and the table carries its own read time.

Slice 5 lists the statement step, the record it names, its state, author and recording time; the text and the place where the source is kept are in a disclosure of the row, shown as written with its line breaks. Only a used statement carries a tag. The thirteen sample references are listed under the table by label and use; their codes are not shown. With every part connected, the not-connected wording of parts 2 to 5 is removed.

Slice 6 reads the settings history on the payment terms view only to select or keep the scope; when that read is refused the view says so and links to the issuance settings. The view states that it calculates no due date. An application made under other issuance settings shows their identifier; one made under the selected settings does not repeat it. The Korean view title changes to 납부 조건 in `live-copy.ts`; the frozen baseline screen `Screen.tsx` still uses 지급 조건 and is not changed here.

Slice 7 adds a "Steps" block under the applications. It states that the screen cannot tell which permissions the user holds. Record is offered while no terms exist; approval while the recorded and observed state are both awaiting approval, and replaced by a fact when the session user recorded the terms; applying once the terms are approved and observed approved; approval of an application on its row, only for a row made under the selected settings. Every request goes through a review step and is sent once. The expected revision is that of the read the step was built from, and a recovery uses the revision of the current read. A reload after a request moves focus to the part heading, and the results of the session are listed below the view, newest first. Recovery is offered only on a result of this session with an unknown outcome, which satisfies both conditions of the recovery rule: the session user sent it, and the session holds its input. Unsent input is kept in session memory per form, contract and issuance settings; it is kept after a refusal, cleared once its request is recorded directly or through a retry or recovery, and discarded by Cancel. The results of the session and the unload guard are rendered by the view itself and do not depend on the terms read, so an unknown outcome stays visible and retryable when a reload fails; recovery reads the current revision of the request's own settings first. While any request of the session is unresolved, every step is disabled and unresolved requests of other contracts are listed too, because the one-at-a-time rule is session-wide. A record left unknown by another session shows no action; the results block says that only the sending session can retry or recover.

# Review record

- Composition: ui-designer role, 2026-11-24, from source and contracts in the plan worktree at main `7894a33`; a proposal, with ten open questions and four unverified points. They are carried above, merged where they overlap, except its question about sample labels, which the copy proposals answer, and its unverified point about a warning icon, which no part of this plan uses.
- Root changes to the proposal: scope held in session memory, not in the URL; configurations without setup context not offered; term writes ahead of protected writes; the read-only slices split so that each adds one table; UTC stated for the effective time; the denied sentence names the Inspect permission and M03.
- Independent design and terminology review, 2026-11-24, role-based AI reviewer separate from the writer, by reading: FAIL with four blocking findings. Corrected here: recovery is a new request with a new key, offered only to the actor of the unknown step and only while the session holds the original input, under the merged label; the statement is not called unused and cannot be withdrawn after an action with an unknown outcome, and an unknown statement request settles before the action; reuse of a statement is limited to this session; the unknown-state and sample labels follow merged copy. Suggestions applied: the further separation refusals, hold and step-up, paging after reload, the refusal causes and the effective-time hint, the receipt of a statement, the wording "may need", observed instead of module wording, the scope action label, the neutral decision action, people shown by identifier, the split of slices 3 and 4, and the label change in slice 6. Not checked by the reviewer: server code, the external pages, renders, native-language acceptance.
- Corrective re-review of the four findings, same reviewer, 2026-11-24, by reading: PASS, all four resolved, no new blocking contradiction. Its residual notes are applied above, except that the remaining "module" wording in user-facing sentences of the parts is left to the terminology review of each slice.
- Slice 1 render and copy review, ui-designer role, 2026-11-24, from 17 of 72 renders, the measurements and source: FAIL with three findings (a stacked label over the disclosure line of the history table at 1440 px, a scope note that named the wrong reload and omitted contract and date, focus lost after both new actions). Corrected; corrective re-review from nine regenerated renders: PASS. Not seen by the reviewer: the rendered "Selected" state, the read-error path of parts 2 to 5, the channels, changes and statements renders. Whether the overlap existed before this slice was not determined. Left for a later slice: announcing a selection made by mouse or automatically. Role-based AI review; no native-language or owner acceptance.
- Slice 2 render and copy review, ui-designer role, 2026-11-24, from 23 renders, measurements and source: FAIL with three findings (an observation worded as "nothing recorded" under a recorded unknown state, a refusal sentence asserting one cause, proposed or unknown values presented like the rules in force). Corrected; corrective re-review from eight regenerated renders including the error block: PASS, with one edge case fixed afterwards without re-review as the reviewer allowed (the plain values heading whenever the rules are observed in effect). Not seen by the reviewer: Arabic renders of two corrected strings, the opened details, the stale and failed blocks. Role-based AI review; no native-language or owner acceptance.
- Slice 3 render and copy review, ui-designer role, 2026-11-24, from 13 renders and source: FAIL with two copy findings (the channel age rule misstated in Korean and Arabic and hard to parse in English; "no longer current" asserting an order the read does not give). Corrected with its suggestions; corrective re-review from five regenerated renders: PASS. Column widths of 22, 26, 32 and 20 percent were applied afterwards on its suggestion. Text at 390 px was not legible to the reviewer at the scale it was shown; structure only. Role-based AI review; no native-language or owner acceptance.
- Slice 4 render and copy review, ui-designer role, 2026-11-24, from 12 renders and source: FAIL with one finding (the events list stated a request as an event where none was observed). Corrected with its suggestions; corrective re-review from five regenerated renders and crops: PASS. Korean and Arabic table text was only partly legible to the reviewer at the scale shown and was judged mainly from the copy file; 390 px for structure only. The sentence that an entry without a record refuses verifications and decisions as well as requests rests on the discovery contract, not on the writes contract or server code. Role-based AI review; no native-language or owner acceptance.
- Slice 5 render and copy review, ui-designer role, 2026-11-24, from 11 renders and source: PASS, with one Korean string to correct before merge and no re-review required (the missing source location read as "not displayed"). Corrected, together with its suggestions: a wider statement column, the sample list as a fact list, State before the record named, the headers "Statement for" and "Purpose", and the Korean purpose line. The render check was rerun afterwards; the reviewer did not see those later renders. Text at 390 px was not legible to the reviewer at the scale shown. Role-based AI review; no native-language or owner acceptance.
- Slice 6 render and copy review, ui-designer role, 2026-11-24, from 12 renders and source: FAIL with one finding (the net-days value joined a number to a fixed unit word, wrong for one day and for most Arabic counts). Corrected with number-aware formatting and a unit test, together with its suggestions; corrective re-review from four regenerated renders: PASS. Only the value 30 was rendered; other counts rest on the formatter and the test. Text at 390 px was not legible to the reviewer at the scale shown. Role-based AI review; no native-language or owner acceptance.
- Slice 7 independent source review (role-based AI reviewer separate from the writer, by reading, nothing executed), 2026-11-24: FAIL with one blocking finding — the results of the session, the retry and the unload guard were mounted only while the terms read was loaded, so an unknown outcome disappeared when the following reload failed. Corrected by rendering the results and the guard from the view itself; its suggestions were applied (two-way settlement of an original and its recovery, drafts cleared when their request is recorded, open-ended refusal sentences, a caught store limit, steps disabled while a request is unresolved, unresolved requests of other contracts listed) and tests added. Corrective re-review: PASS; the notes it asked to correct before merge are applied (the store-limit sentence, a busy flag during recovery, the sent time kept on the entry). Not checked by the reviewer: server code, the render check script, Korean and Arabic wording.
- Slice 7 render and copy review, ui-designer role, 2026-11-24, from 17 renders: FAIL with one finding (an unknown outcome also carried a sentence saying the request was not completed). Corrected with its suggestions; corrective re-review from twelve regenerated crops: PASS on that finding, with one item to fix before merge and no further review required — the steps disabled while a request is unresolved now state why and reference that sentence. Moving the record and approve steps next to the terms they act on was suggested and is left for the owner's screen review. Role-based AI review; no native-language or owner acceptance.
