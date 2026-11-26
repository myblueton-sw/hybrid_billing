---
type: ImplementationPlan
title: B09 financial decision journey
description: Current checks, explicit decision confirmation and truthful uncertain outcomes for governed charge requests.
status: draft
---

# Scope and ownership

HYB-340, machine:mac_mini. Root owns implementation/integration; refusal_review owns read-only financial/security review, platform_refusals owns terminology/evidence review, and actual Claude CLI ui-designer leads visual composition and final rendered review. Apply hb-design-completion-loop, hb-react-screens and hb-screen-regression. This plan depends on the independently reviewed financial_charge.prepare producer. Owned files are B09 components/client/contracts/tests/styles, the server B09 capability declaration and associated tests, scoped browser tooling and evidence. Do not revise unrelated screens or create owner acceptance.

# Journey and capability

A service-backed StageCharge record retains its historical facts. Add a decision section only for records carrying the reviewed service basis and current-check marker; this marker never grants eligibility. An explicit Check current conditions action calls authenticated financial_charge.prepare for the selected request digest. Bind preparation to the actual session identity/workspace/CSRF in memory, selection ID, digest and read scope. Render exact amount/currency, original author, requester, recorded policy, checked-at observation and separate approval/rejection availability. No previous preparation survives selection/query/session change or failed refresh. Unknown action outcomes or malformed response fields fail closed.

Eligible actions open an inline confirmation form with a required rationale and, for rejection, the existing closed reason selection. Show target ID, exact amount/currency and decision meaning beside the final Confirm approval or Confirm rejection button. The confirmation stores immutable preparation revision/digest and a fresh attempt identity. Disable duplicate submission synchronously before awaiting network work; preserve the same attempt for ambiguous-result inspection/retry. Confirmation does not imply execution, invoicing, posting, payment or ERP acceptance. Other records retain explicit unavailable action copy.

While submission is in flight, avoid accidental action or close-induced loss of its status; expose pending state accessibly. A matching validated command receipt confirms only the recorded historical decision. Refresh the inbox after settled success and invalidate preparation. Known refusals clear eligibility and require a new current check. A timeout, network error, malformed success or partial/unknown operation is Outcome unknown, not failed or succeeded. Offer an explicit same-attempt retry using identical command, target, digest, rationale, reason, expected revision and idempotency key; never auto-retry or mint a new attempt for that unknown action. Session changes invalidate the client and require sign-in/current historical inspection; never reuse CSRF or prepared eligibility across sessions. A historical list state can explain what is recorded but cannot identify an uncertain attempt as successful. Current command-time authorization and server serialization remain authoritative.

# State and accessibility

Own decision state once above desktop/mobile detail copies; use one shared state controller and visible form instances without divergent request state. Preserve full-width layout, neutral separators, square geometry and visible focus. Use explicit labels/status text, logical RTL layout, isolated IDs/money, native form controls, a focusable error summary and correct focus restoration. No decorative accent borders. Locale changes affect copy only. Register decision state in the screen review fingerprint and dirty-state boundary. Existing list reads remain read-only operations, but their commands capability declaration must stop claiming the screen has no actions; accept only an explicit reviewed closed capability, with compatibility for historical read-only payloads.

# References and validation

Root re-read [Microsoft approval requests](https://learn.microsoft.com/en-us/power-automate/approve-reject-requests) and [GitHub deployment reviews](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/review-deployments). Adopt selected-record detail, visible target, rationale and explicit final confirmation. Reject optional rationale, deployment effects, bypass paths, email actions, assets, colors and permission defaults. No external content is copied.

Before code obtain independent behavior review, actual Claude layout specification against current renders and terminology review. Test real authenticated synthetic request-to-prepare-to-approve/reject, read-only/decision separation, author/requester exclusion, amount limit, hold, reauthentication, stale revision/evidence, revoked permission, wrong digest, late responses, session changes, duplicate clicks and ambiguous response/same-attempt recovery. Render EN/KO/AR at 1440/1024/390 with long IDs, keyboard/focus/error states and computed border/overflow checks. Re-review changed code and actual renders; retain native-language and owner acceptance as distinct pending gates. Full W-23 and downstream execution remain active.

# Independent lifecycle conditions

Independent financial review requires a session-keyed in-memory attempt controller whose lifetime survives same-session screen, navigation and locale remounts. Selection/query changes invalidate preparation, never pending or unknown attempts. Retain the frozen payload and original request/operation reference for each unresolved request. A same-attempt retry refusal (including permission loss, expired key or unknown stored operation) cannot turn the earlier unknown effect into failure or unlock a fresh attempt. No browser storage of sessions or decisions is introduced. A document reload discards ephemeral client evidence; explicitly disclose that limitation and require fresh historical inspection without claiming the prior attempt failed. Current authorization can prevent recovery.

Require both the explicit server capability and service-record marker before showing decision controls. Validate preparation identity/digest, exact amount/currency, author/requester and policy against the selected record and validate its revision format. Validate successful receipts against the frozen command/target/digest and expected recorded state. A post-success inbox refresh failure leaves the decision success intact with a separate stale-list notice. Only the visible responsive form may be interactive; use unique IDs and error-summary links. Add navigation-away/back, locale remount, pending responsive resize, unknown-to-retry-permission-refusal and success-to-refresh-failure coverage before acceptance.

# Reviewed terminology candidates

Independent AI domain/TW review supplied these candidates; rendered, native-language and owner acceptance remain pending.

| Meaning | EN | KO | AR |
| --- | --- | --- | --- |
| Current check | Check decision conditions | 결정 조건 확인 | التحقق من شروط القرار |
| Final approval | Confirm approval | 승인 확정 | تأكيد الاعتماد |
| Final rejection | Confirm rejection | 반려 확정 | تأكيد الرفض |
| Unknown effect | Outcome unknown | 처리 결과 미확인 | نتيجة العملية غير معروفة |
| Identical retry | Retry the same submission | 동일한 제출 다시 시도 | إعادة محاولة الإرسال نفسه |
| Eligible observation | Available at last check | 마지막 확인 시 가능 | متاح عند آخر تحقق |

Unknown outcome explanation: The decision may have been recorded. Retrying uses the same submission; it does not create a new attempt. / 결정이 기록되었을 수 있습니다. 다시 시도하면 새 시도를 만들지 않고 동일한 제출을 재전송합니다. / ربما سُجّل القرار. تُعيد المحاولة إرسال الطلب نفسه ولا تُنشئ محاولة جديدة.

Successful decision with refresh failure: The decision was recorded, but the list could not be refreshed. Reload the list. / 결정은 기록됐지만 목록을 새로 불러오지 못했습니다. 목록을 다시 불러오세요. / سُجّل القرار، لكن تعذّر تحديث القائمة. أعد تحميل القائمة.

# Actual visual design and integration decisions

Actual Claude CLI ui-designer read both plans, existing B09 source/style/tokens and seven baseline PNGs and returned /private/tmp/hyb340-decision-screen-design-review.txt. It did not browse references or certify final visuals. Adopt its section after KindBoundary and before recorded facts, explicit recorded-facts heading, exact amount/actor summary, per-action availability rows, inline confirmation, neutral feedback/error-summary boxes, unique responsive IDs and visible-element focus resolution. Keep controls readable within full-width sections and preserve existing long amount/ID behavior. Correct the duplicate neutral rule in the changed tablet detail boundary if confirmed by the rendered implementation.

Root resolves its open choices against the mandatory contracts: use neutral existing buttons with stronger type for final confirmation, without a colored button-border exception or additional owner approval flow. The persistent session-scoped controller required by independent financial review owns unresolved attempts above Screen remounts; a Screen-local hook alone is insufficient. Selection/query changes never erase an unresolved attempt. Original request/operation references may appear under technical details, never CSRF/session secrets. Do not include rationale in review fingerprints or persisted evidence. Success receipts remain in a page-level result area if the record leaves the current filter; failed refresh does not change confirmed decision status. The current historical-uncertainty explanation remains visible and is distinct from current preparation.

Use the reviewed terminology candidates above instead of unreviewed designer placeholder copy. Record rendered scope honestly: core checked/reject-confirmation/validation/unknown/success states need EN/KO/AR ×1440/1024/390, with other async/refusal states at EN1440 and AR390, plus resize/focus/long values. This is a verification plan, not proof of those renders. Frontend-design composition guidance is subordinate to the existing full-width, neutral and local-asset system; no novelty styling or new fonts is introduced.
