---
type: VisualSpecification
title: M04 credential issues with an unknown outcome, never-collected secrets and locked held forms
description: Binding visual and interaction specification for HYB-333 in the M04 console. It covers where the two new reminder regions go, the pickup state on credential rows, every issue and create form phase, copy that never claims certainty it does not have, focus, announcements, RTL, layout at 1440, 1280, 1024 and 390 px, and rendered QA.
status: draft
tags: [HYB-333, M04, service-accounts, credentials, secret-pickup, visual-spec, ui-designer]
sources:
  - resource: ./plan.md
  - resource: ../HYB-316/verification.md
  - resource: ../../ui/screens/M04/spec.md
  - resource: ../../ui/terminology/product-glossary.md
  - resource: ../../design-system.md
---

# Status, authorship and evidence

- Author: the AI ui-designer role (Claude Opus 5.5), read-only. This is a design proposal, not owner acceptance. Screen acceptance (FIX) belongs to the owner alone.
- **Renders inspected (direct observation)**, from the HYB-316 session scratchpad `reg316shots-m04fix/` and `reg316shots-pre/`:
  - `M04_credentials_en_1440`, `M04_unstored_en_1440`, `M04_issue_retry_en_1440`, `M04_pickup_gone_en_1440`;
  - `M04_credentials_ar_1440`, `M04_credentials_en_390`;
  - `held.png`.
- **Observed baseline**:
  - At 1440 the list column is about 707 px and the side panel about 380 px. The panel appears at ≥1280 px.
  - The credential table uses fixed columns of 30/34/20/16 %. "Valid" already wraps "9:02 / AM", and credential IDs break after `cred-`.
  - "Secrets not confirmed as stored" is a neutral 1 px bordered box (`service-summary live-unstored`). It sits between the toolbar and the service select, with rows of the form text … [Revoke][Dismiss].
  - The held issue form shows a danger-colored "Could not save this change" box and leaves days and reason editable.
  - At 390 the table stacks into labelled blocks with full-width Revoke buttons.
- **Not available**:
  - no frontend-design skill was supplied;
  - no external service was browsed in this session;
  - no 1024 or 1280 render exists;
  - no KO or AR render of the held form exists.
- **External reference patterns** come from model knowledge and were **not re-verified** in this session. Root must re-check them under `.agents/rules/service-screen-style.md` before recording them as evidence:
  - GitHub (new personal access token) and AWS IAM (create access key) both show a secret once, warn that it cannot be retrieved later, and offer revoke or delete as the recovery. **Adopted**: revoke-and-reissue as the next step, and no secret or reference in lists.
  - Stripe API idempotency documentation: retrying with the same key returns the stored result, and keys expire after a window. **Adopted**: "Try again sends the same request" and a lapsed state after the window.
  - GOV.UK Design System error summary and notification banner: the heading states the situation, the body says what to do, and focus moves to the summary. **Adopted**: heading-first reminder boxes and a focus target.
  - IBM Carbon inline notifications carry a colored inline-start accent strip. **Rejected**: the owner prohibits accent edges.

# 1. Mandatory constraints mapped to this change

| Constraint | Application here |
| --- | --- |
| No decorative colored borders or accent strips | The two new regions, the held, lapsed and sending blocks, and the row sub-lines use only `1px solid var(--border-control)` boxes or `var(--border-row)` separators. All four sides are identical. No `box-shadow`, no tinted background, no colored heading. |
| Full viewport width | The regions span the full list column. Only paragraph text is capped (68ch). There is no centered max-width container. Buttons keep their intrinsic width at ≥768 px. |
| Neutral state by wording | Severity is carried by headings and sentences, not by color. The danger color (`--hb-danger`) is **not** used for unknown, lapsed or never-collected states, because none of them is an error the person made or a confirmed failure. |
| Visible focus | The existing focus token on every new button, read-only input, `tabIndex=-1` heading and block. |
| Preserved | IA (Accounts / Credentials / Usage / Transfer), screen ID M04, the canonical logo, local IBM Plex, square corners, the existing `data-credential`, `.live-unstored`, `#svc-cred-title` and `#svc-list-title`. |

# 2. Credentials view: placement and hierarchy

Top to bottom inside `section[aria-labelledby="svc-cred-title"]`:

1. Toolbar: `h2#svc-cred-title`, "Loaded at …", then [Issue credential] (narrow only) and [Reload]. Unchanged.
2. `notice` (`.service-status`). Unchanged.
3. **"Credential issues to check"** (`section.live-attempts`, new). This comes first because it is the only region that can still produce a secret, for a few minutes ("Show secret" or "Try again").
4. **"Credentials whose secret was never collected"** (`section.live-never`, new). This is a server fact that survives a reload. Its next step is Revoke.
5. "Secrets not confirmed as stored" (`.service-summary.live-unstored`, existing and unchanged, still hidden in `reveal` mode). A credential that is in the never-collected list is removed from this list.
6. Service select, then the pre-launch note, the count, the credential table and "Ended credentials". All unchanged except §3.

**Spacing between regions**
- Each region box has `margin-block: 0 16px`, the same as `.live-unstored`. The last region before the select keeps 16 px, and the select keeps its existing top spacing.
- Never wrap the regions in an extra outer box. Never collapse them into tabs.
- Regions 3 to 5 are siblings with identical box geometry, so the rhythm reads as one "needs attention" stack without a new container.

**Region box (both new regions)**, scoped `[data-security-screen="M04"]`:
- `border: 1px solid var(--border-control)`, `padding: 16px`, background inherited (page surface), square corners.
- Heading: `h3`, 16px/24px, weight 600, color `var(--hb-ink)`, `margin: 0 0 8px`. Ids: `svc-attempts-title` and `svc-never-title`, both `tabIndex={-1}`. The section has `aria-labelledby` set to its heading id.
- Lead paragraph: 16px/24px, `var(--text-muted)`, `max-inline-size: 68ch`, `margin: 0 0 8px`.
- List: `ul` with no bullets and no padding. `li`: `display:flex; flex-wrap:wrap; align-items:center; justify-content:space-between; gap: 8px 16px; padding-block: 12px`. `li + li` gets `border-block-start: 1px solid var(--border-row)`. The first `li` has no top rule.
- Row text: `flex: 1 1 32ch; max-inline-size: 68ch; min-inline-size: 0`, 16px/24px, ink.
- Row buttons: `.live-row-buttons` (existing; `flex:none`, gap 8px). Every button is `min-block-size: 44px`, padding `0 16px`, intrinsic width at ≥768 px.

## 2.1 "Credential issues to check" (`live-attempts`)

- Shown when the store holds at least one entry for (user, any service). It stays visible in `reveal` mode, because consumed entries are already removed.
- Rows are ordered by first send time, oldest first. Each row is `li[data-attempt="<service>"]`.
- The lead is `attemptsDiscardNote` [TW]. It is shown only while at least one row offers Discard (`unknown` or `lapsed`).

| Phase | Row text (service and credential IDs in `bdi dir="ltr" class="live-mono"`) | Buttons, in order (inline start → end) |
| --- | --- | --- |
| `sending` | `attemptSending` [TW], weight 400 | none. The row text has `aria-busy="true"`. |
| `unknown` | `attemptUnknown` [TW]. The first clause up to the colon, "Issue for {service}: outcome unknown", is weight 600. | [Open issue form] `live-secondary` · [Discard] `live-secondary` |
| `issued` | `attemptIssued` [TW] | [Show secret] **`live-primary`** · [Revoke] `live-secondary` with `data-attempt-revoke="<service>"` and a visually hidden " {credential}" |
| `lapsed` | `attemptLapsed` [TW]. The first clause, "Issue for {service}: the outcome can no longer be checked.", is weight 600. | [Discard] `live-secondary` |

- Every button carries a visually hidden suffix with the service ID (and the credential ID for Revoke), as the existing Revoke and Dismiss buttons do.
- Show secret is the only primary button in the list column, which is justified because it is time-limited. Never render two primaries in one row.

## 2.2 "Credentials whose secret was never collected" (`live-never`)

- Shown when any active credential in the read has `secret_pickup = never_collected`.
- Rows are sorted by service ID, then credential ID (canonical and stable). Each is `li[data-never="<credential>"]`.
- The lead is `neverLead` [TW], always shown.
- Row text, on two lines:
  - line 1: `cred-…` and `svc-…`, both mono isolates, separated by " · ", weight 600;
  - line 2: `v.until(l, when(l, not_after))` (existing string), 14px, muted, `display:block; margin-block-start: 4px`.
- Button: [Revoke] `live-secondary` with `data-never-revoke="<credential>"` and a visually hidden " {credential}". It opens the revoke step with the reason `revokeNeverReason`.
- **Volume bound.** Show at most 5 rows. If there are more, rows 6 and later go into a `details.live-ended` placed directly after the list, inside the box. Its summary is `neverMore` [TW new] and it uses the existing `.live-ended summary` styling (44 px). The region never pushes the table off-screen by more than about 5 rows.

## 2.3 Accounts view note

- Shown when the store holds at least one entry in `sending`, `unknown`, `issued` or `lapsed`.
- Placement: directly after the accounts toolbar and notice, before the table.
- Markup: `div.live-note-rule`, the existing neutral top rule with 16 px padding. Inside it, `p` contains `accountsAttemptsNote` [TW], one space, then `button.service-button-link` labelled `goCredentials` [TW new].
- The button switches to the credentials view and focuses `#svc-attempts-title`.
- Text is capped at 68ch. No box, no color.

# 3. Credential row: pickup state

The status cell `td[data-label=Status]` stacks blocks in this order:
1. The state ("Active", "On hold", "Revoke required"). Unchanged.
2. The existing sub-line, if any (for example `cOnHoldSub`). Unchanged.
3. The pickup sub-line (new), `span.live-sub` plus the classes below, `display:block; margin-block-start: 4px`.

| `secret_pickup` | Sub-line | Style |
| --- | --- | --- |
| `open` | `pickupOpen` [TW] with `{time}` = `when(l, secret_pickup_until)` in an `iso()` isolate | class `live-pickup`: 14px/20px, `var(--text-muted)`, weight 400. The date and time have separate `white-space: nowrap` spans inside an LTR isolate, so the timestamp can break between date and time at narrow desktop columns. Credential IDs wrap within their own cells (combined render-review correction). |
| `never_collected` | `pickupNever` [TW] | class `live-pickup-never`: 14px/20px, `var(--hb-ink)`, **weight 600**. This overrides `[data-security-screen="M04"] .live-strong .live-sub { font-weight:400 }`. |
| `collected`, absent, or a non-active row | nothing | The cell is exactly as today, so qa269's "Active" assertions still hold. |

- Never add a pickup claim to the Accounts table, the Usage tab, or the "Ended credentials" table.

## 3.1 Credential table columns (M04 credential tables only)

- Add the class `live-creds` to the `CredentialTable` wrapper.
- Under `[data-security-screen="M04"] .live-creds`, at ≥768 px, set `col-a 27%`, `col-b 29%`, `col-c 28%`, `col-d 16%`.
- The Accounts table keeps 30/34/20/16.
- Purpose: at a 568 px list column (1280 with the panel open), Status gets about 159 px instead of about 114 px, and `pickupOpen` wraps to at most 4 lines in EN.
- The ≤767 px stacked layout is unchanged.

# 4. Issue form (side panel at ≥1280, panel step below 1280)

Common to every phase:
- `h2` "Issue credential" (`tabIndex -1`), then the service ID line, then the phase block, then the fields, then the buttons.
- The phase block, when present, is `div.live-held` with `id="svc-held-lead"` on its lead paragraph and `tabIndex={-1}` on the block. It has the same box as §2: `border-control`, 16px padding, `margin-block: 16px 24px`, h3 16/24 weight 600 ink, body 16/24 ink, 68ch.
- **`live-held` is never `role="alert"` and never uses `--hb-danger`.**

## 4.1 Idle

- Unchanged from today: lead, then fields, then [Issue credential] `live-primary`.

## 4.2 Sending

- Phase block:
  - no h3;
  - body is `sendingLead` [TW new] ("Sending this request. The values are locked until the server answers.").
- Days input and reason textarea:
  - `readOnly`;
  - `aria-describedby` = existing hint ids + `svc-held-lead`.
- Locked style, `[data-security-screen="M04"] .live-step input[readonly], textarea[readonly]`:
  - `background: var(--surface-hover)`;
  - `color: var(--hb-ink)` (**not muted**: these are the request's values and must keep body contrast);
  - border unchanged (`--border-control`);
  - `cursor: default`;
  - still focusable, with the focus token visible.
- The "Ends …" line and the byte counter stay.
- Buttons:
  - [Saving…] `live-primary`, `disabled`, `aria-busy="true"` (existing).
  - [Discard and start over] `live-secondary` is shown **only when this send is a retry** (`retry=true`), and is then `disabled`.
  - Add `[data-security-screen="M04"] .live-secondary:disabled { opacity:.7; cursor: not-allowed }`.
  - A disabled Discard is never shown on a first send.

## 4.3 Unknown (held)

- Phase block:
  - h3 `heldTitle` [TW new] "Outcome unknown";
  - p#svc-held-lead `heldLead` [TW];
  - p#svc-locked-note `lockedNote` [TW new].
- The block **replaces** the FailureBox for every answer whose phase-table row yields `unknown` (`busy`, `unavailable`, `partial` on a first send, `limited`, `unauthenticated`, and a `reauth` that is kept).
- The texts "Could not save this change" and `issueRetry` ("…so you don't issue twice") must not render in this phase.
- If the failure carries a request ID or stable code, render it as a 14px muted line at the end of the block (existing format). Never render the key, the `secret_ref` or the secret.
- Inputs: locked as in §4.2. `aria-describedby` also includes `svc-locked-note`.
- Buttons, in one `.live-buttons` row with a 12px gap: [Try again] `live-primary` (existing `c.tryAgain`), then [Discard and start over] `live-secondary` (`startOver` [TW]).

## 4.4 Issued

- Form visible (the plan's five-condition "visible" predicate): the Reveal opens automatically (existing Reveal, unchanged).
- Form not visible: no form change. The entry shows in §2.1.
- Submit is disabled while the entry is `sending` or `issued`.

## 4.5 Lapsed

- Phase block:
  - h3 `lapsedTitle` [TW new] "Outcome can no longer be checked";
  - p `attemptLapsed` [TW].
- Inputs are **not** rendered. Below the block, a `Facts` list shows the requested values for recognition, using existing labels:
  - `validFor` → "{n} days";
  - Reason → the reason text, or "—".
- Button: [Discard and start over] `live-secondary` only. There is no Try again and no primary button.

## 4.6 Removed after a definitive answer

- First send: the existing FailureBox and `describeFor("issue")` text apply. That text speaks only for this request, so its "nothing was saved" is truthful there.
- **Retry** refused (`conflict`, `refused`, `stage`, `invalid` inside the window):
  - the FailureBox title is `refusedTitle` (existing);
  - the body is `retryRefused` [TW new];
  - `conflict` "…so nothing was saved" is **not** used for a retry.
- The inputs unlock and the form returns to Idle with the values kept.

## 4.7 Discard and start over

- Renews the attempt.
- Inputs become editable and keep their values.
- The phase block disappears.
- Focus moves to `#svc-days`.

# 5. Create form

- The same phase blocks as §4, with an id prefix `svc-create-`: `#svc-create-held-lead` uses `createHeldLead` [TW], and the block titles are the same keys.
- Sending and held:
  - the name input (`#svc-name`) and the reason are `readOnly` with the locked style;
  - **the owner `<select>` is replaced by `<input id="svc-owner" readOnly class="live-select">`** whose value is the owner's login (bdi LTR). The label association and focusability stay. `select[disabled]` is forbidden: it drops out of the focus order and is not read as a value;
  - `aria-describedby` includes `svc-create-held-lead`, plus `svc-create-locked-note` when held.
- The ID preview line ("ID: svc-…") stays visible.
- Lapsed:
  - h3 `lapsedTitle` plus `createLapsed` [TW new];
  - a Facts list of name, owner and reason;
  - [Discard and start over] only.
- Success, including a replay, goes to the detail (unchanged).
- `errNameTakenMine` [TW]:
  - rendered as the `#svc-name-error` field error and in the error Summary;
  - `{id}` in a mono LTR isolate;
  - the Summary link focuses `#svc-name`;
  - otherwise the existing "name taken" text.

# 6. Focus and keyboard

| Event | Focus goes to |
| --- | --- |
| A phase becomes `unknown` or `lapsed` while the form is visible | `div.live-held` (`tabIndex -1`), once per transition (focusKey pattern) |
| A phase becomes `unknown`, `issued` or `lapsed` while the form is **not** visible | Focus is not moved. See §7. |
| Open issue form (row) | Selects the service and opens the panel; focus goes to the IssueForm `h2` |
| Show secret (row) | The Reveal `h2#svc-reveal-title` (existing) |
| Discard (row) | The next row's first button. If there is none, the previous row's. If the region is gone, `#svc-cred-title`. |
| Discard and start over (form) | `#svc-days` (issue) or `#svc-name` (create) |
| Revoke from any source, then Cancel or narrow Back | `from` if it is on screen, else the view heading, per the plan's `returnFocus` (`data-credential`, `data-unstored-revoke`, `data-never-revoke`, `data-attempt-revoke`) |
| Successful Revoke from a region row | `#svc-cred-title`. The row is gone and the notice announces the result. |
| `goCredentials` (accounts note) | `#svc-attempts-title` |

- Tab order follows DOM order: toolbar, attempts, never, unstored, select, table, then the panel.
- Read-only inputs stay in the tab order.
- Disabled buttons are skipped.

# 7. Announcements (`p.visually-hidden[role=status][aria-live=polite]`, existing)

- A transition while the form is not visible announces, once each:
  - to `unknown`: `attemptUnknown`;
  - to `issued`: `attemptIssued`;
  - to `lapsed`: `attemptLapsed`.
- A transition while the form is visible announces nothing extra, because focus moves to the block.
- Sending: the existing `aria-busy` only. No polite chatter.
- Never-collected region: no announcement on load or reload, because it is static server state. It is discoverable by heading.
- Allowed in any live region or `aria-*`: the service ID, the credential ID, times. **Forbidden**: the secret, any `hbsk_` value, the `secret_ref`, the idempotency key and the request body hash.

# 8. RTL (AR)

- Use only logical properties (`margin-inline`, `padding-block`, `border-block-start`, `inset-inline`). In AR, row buttons sit at the inline end (left), and text starts on the right.
- Every ID and time placeholder is an isolate: `bdi dir="ltr"` for IDs, and `iso()` in strings. The " · " separator sits between two isolates, so the IDs never reorder.
- Locked inputs keep their `dir` (days `dir="ltr"`; reason `dir="auto"`, existing).
- The pickup sub-line wraps right-aligned in the status cell. In AR, the nowrap time isolate must not overflow the 28% column at 1280 (QA check).

# 9. Layout by width

| Width | List column (approx.) | Regions | Form |
| --- | --- | --- | --- |
| 1440 | about 707 px, panel about 380 | Text and buttons on one line per row; the text column caps at 68ch | Side panel. Buttons have intrinsic width, side by side, start-aligned. |
| 1280 | about 568 px, panel 360 | Text (min 32ch) and buttons still fit one line in EN. KO and AR may wrap the buttons below the text (start-aligned, gap 8px). | Side panel, same as 1440 |
| 1024 | full content width, no panel | Rows on one line | Panel replaces the list as a step with Back (existing) |
| 390 | full width | Each `li` stacks: the text, then `.live-row-buttons` as a column. Each button is `inline-size:100%`, 44px, gap 8px. | Step. Buttons are full width and stacked: primary first, then Discard and start over. |

- No horizontal page scroll at any width.
- The ≤767 rules live under the existing `@media (max-width: 767px)` in `live.css`, M04-scoped.

# 10. Must not appear

- The secret, any `hbsk_` prefix, the `secret_ref` or pickup reference, or the idempotency key, anywhere in the DOM outside `#svc-secret`, in `aria-*`, in live regions, in titles or tooltips, or in the console.
- "Collected", "stored" or any time for `secret_pickup = collected`. Owner question Q3 defaults to showing nothing.
- Any of these for an unknown, lapsed, discarded or retry-refused attempt: "not issued", "nothing was issued", "nothing was saved", "failed", "Could not save this change", "check the list first so you don't issue twice".
- "Revoked" or "unusable" implied by "Secret never collected". The state cell keeps "Active" or "On hold".
- A colored edge, colored heading, tinted box or `--hb-danger` on the new regions, held blocks or sub-lines.
- A disabled "Discard and start over" on a first send.
- Stretching buttons across the column at ≥768 px.

# 11. Copy keys used ([TW] = needs technical-writing review; KO and AR are AI drafts)

- From the plan: `pickupOpen`, `pickupNever`, `neverTitle`, `neverLead`, `revokeNeverReason`, `attemptsTitle`, `attemptSending`, `attemptUnknown`, `attemptIssued`, `attemptLapsed`, `attemptsDiscardNote`, `openIssue`, `discardAttempt`, `startOver`, `heldLead`, `revokeNotShownReason`, `accountsAttemptsNote`, `createHeldLead`, `errNameTakenMine` [TW].
- New in this spec, all [TW]:

| Key | EN | KO | AR |
| --- | --- | --- | --- |
| `heldTitle` | Outcome unknown | 결과 확인 안 됨 | النتيجة غير معروفة |
| `lapsedTitle` | Outcome can no longer be checked | 결과를 더 이상 확인할 수 없음 | لم يعد بالإمكان التحقق من النتيجة |
| `sendingLead` | Sending this request. The values are locked until the server answers. | 요청을 보내는 중입니다. 서버가 응답할 때까지 값을 바꿀 수 없습니다. | جارٍ إرسال هذا الطلب. القيم مقفلة إلى أن يردّ الخادم. |
| `lockedNote` | The values are locked so that Try again sends exactly the same request. | 다시 시도가 같은 요청을 보내도록 값을 잠갔습니다. | القيم مقفلة كي تُرسل إعادة المحاولة الطلب نفسه تمامًا. |
| `retryRefused` | Try again was refused because service accounts changed. This answer covers only the repeated request: if the first request went through, its credential is in the list. | 서비스 계정이 바뀌어 다시 시도가 거부되었습니다. 이 응답은 다시 보낸 요청에만 해당합니다. 첫 요청이 처리되었다면 그 자격 증명이 목록에 있습니다. | رُفضت إعادة المحاولة لأن حسابات الخدمة تغيّرت. يخص هذا الرد الطلب المُعاد فقط: إن نُفِّذ الطلب الأول فبيانات اعتماده في القائمة. |
| `createLapsed` | The outcome of this request can no longer be checked. If {id} is in the list, it may come from this request. | 이 요청의 결과를 더 이상 확인할 수 없습니다. 목록에 {id}이(가) 있다면 이 요청으로 만들어졌을 수 있습니다. | لم يعد بالإمكان التحقق من نتيجة هذا الطلب. إن ظهر {id} في القائمة فقد يكون ناتجًا عن هذا الطلب. |
| `goCredentials` | Go to Credentials | '자격 증명'으로 이동 | الانتقال إلى «بيانات الاعتماد» |
| `neverMore` | {n} more | {n}개 더 보기 | {n} أخرى |

- TW flags:
  - AR `discardAttempt` and the existing `dismiss` are both "تجاهل" in adjacent regions and need distinct AR words;
  - the AR secret term: the glossary's "استلام السر" vs the plan's "القيمة السرية";
  - KO spacing "비밀 값" vs "비밀값" (plan Q5);
  - AR plural forms for `{n}` in `accountsAttemptsNote` and `neverMore`.

# 12. Rendered QA (root runs; the ui-designer compares afterwards)

Capture EN, KO and AR at 1440, 1280, 1024 and 390 for each of the following:
- a. Credentials view with all three regions present (attempt rows `unknown`, `issued` and `lapsed`; at least 2 never-collected rows; 1 unstored row).
- b. Never region with 6 or more rows (the disclosure).
- c. A table row with `open`, a row with `never_collected`, and an on-hold row with `never_collected` (two sub-lines).
- d. Issue form in sending (retry), unknown and lapsed.
- e. Retry refused (`retryRefused`).
- f. Create form held, with the owner as a read-only input.
- g. `errNameTakenMine`.
- h. Accounts view note.

Checks:
1. `getComputedStyle` on `.live-attempts`, `.live-never`, `.live-held` and their `li`:
   - all four border colors equal `--border-control` (or `--border-row` for separators);
   - the border widths are equal;
   - `box-shadow: none`;
   - the background is not tinted;
   - h3 color equals ink.
2. Locked inputs: `readOnly === true`, the computed `color` equals ink, contrast ≥ 4.5:1 against `--surface-hover`, and the focus ring is visible when tabbed.
3. No horizontal scroll (`scrollWidth <= clientWidth` on the document and on `.live-list`).
4. In AR, row buttons sit at the inline end, and IDs and times render LTR inside the RTL sentence.
5. Buttons are ≥44 px high. At ≥768 px, no button spans more than 50% of its region.
6. The pickup sub-line is at most 4 lines in EN at 1280, with no break inside the time isolate.
7. A DOM, `aria-*` and live-region scan finds no `hbsk_`, no `secret_ref` value and no `Idempotency` value. The strings in §10 are absent in their forbidden phases.
8. The focus-table rows in §6 are asserted by `document.activeElement`.
9. qa269 regression: the `[data-credential]` count and the `.live-unstored` checks are unchanged.

# 13. Critique against the owner's brief

- **Meets**:
  - neutral hierarchy by headings and wording, with no accent edges;
  - full-width list with capped text measure;
  - intrinsic-width controls;
  - truthful wording for unknown and lapsed states, and for retry refusals;
  - visible focus and a defined focus return;
  - EN/KO/AR and RTL by isolates;
  - no secret-adjacent data on screen.
- **Trade-offs**:
  - Three stacked reminder boxes can push the table down at 1280. This is bounded by the 5-row cap and by `lapsed` and `issued` entries being removed. It was accepted over merging the regions, because each region has a different evidence source (page memory, server read, person's acknowledgement).
  - The table rebalance touches one existing layout (§3.1). Pre-existing wrapping in the "Valid" column remains a residual outside this ticket.
- **Open**:
  - the external references need re-verification;
  - the frontend-design skill was not applied;
  - TW review is pending;
  - owner questions Q1 to Q6 in the plan.

# 14. Root decisions (2026-11-19)

Binding over the matching proposals and the plan where they conflict.

1. Findings 1–4 adopted: the held form uses the neutral "Outcome unknown" block (no FailureBox, no danger colour, no "check the list first" advice); `sendingLead` replaces `heldLead` while sending; a refused retry uses `retryRefused`, never "nothing was saved"; the create form's owner select becomes a read-only input while sending or held.
2. Finding 5: the new [TW] keys are adopted as drafts for the technical-writing review.
3. Finding 6: AR `discardAttempt` must differ from the existing `dismiss` (TW to choose; draft «إلغاء المحاولة»); the AR secret term follows the glossary row («استلام السر» family) unless TW decides otherwise; KO keeps the glossary spelling «비밀값» (Q5 answered by the glossary).
4. Finding 7 / §3.1: the M04 credential tables rebalance to 27/29/28/16; the Accounts table is unchanged.
5. External references in §0 are model knowledge and are not recorded as evidence; no new layout pattern is introduced beyond P02/M04's existing neutral boxes, row lists and Facts lists.
