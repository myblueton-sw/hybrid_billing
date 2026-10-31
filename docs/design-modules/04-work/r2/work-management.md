---
type: DesignSpecification
title: Calendar notification and work management
description: Defines stage schedules notification rules and unified job handling with safe recovery.
status: draft
---

# Context

Inherits the [shared screen contract](../../11-settings/r2/README.md). Sources: HB-58 OPS01–03; v1.16 B1359, B1376–B1396, B1420/B1423; [canonical commands](../../../contracts/commands/planning-actions.md) and [U02](../../../ui/screens/U02/spec.md). These textual states extend the existing calendar/queue concept without claiming visual completion.

# U02 — operational calendar (OPS01)

Entry: Work → Calendar; scoped operators see collection, calculation, approval, issuance, delivery and payment as separate stage rows. Fields: cycle/schedule ID, stage, original/current deadline, timezone and UTC instant, business-calendar revision, holiday/month-end rule, DST ambiguity resolution, owner/alternate, completion predicate, dependency state, hold and revision. Multiple named schedules per stage remain distinct. Missing calendar policy blocks scheduling; ambiguous/nonexistent local time requires explicit approved resolution, never an implicit timezone fallback.

Flow: select exact schedules → propose date → preview affected cycles and downstream suggestions → approved digest → apply → inspect history and next deadline. `schedule.change_request` proposes; `schedule.apply_approved` applies the separately authorized approved revision. No schedule operation approves amounts or issues/delivers a document. Payment due dates use the separate payment extension contract and finance permission; moving delivery never extends payment terms. Dependency suggestions require separate changes, not a cascading silent rewrite.

Walkthrough T01: a holiday-adjusted collection date moves while approval and issuance remain unchanged unless separately reviewed. A conflicting revision requires reinspection. A missed deadline stays overdue with its original date, assigned owner and recovery action. Reopening completed work requires a separately authorized business action and evidence; a date edit cannot clear completion, holds or blockers. Payment-extension application, calendar policy edit and reopen bindings remain **unresolved** here.

# U02/P02/U03 — operational notification rules (OPS02)

Entry: Calendar → Rules, or authorized U03 → Operational rules. P02 supplies channel bindings; M05 supplies personal preferences only. Fields: rule ID/revision, scope, stage/schedule kinds, notification kind, relative offsets, recipient roles, ordered escalation conditions, alternate owner, channel references, template revision and rationale. Show installation approval/separation policy. Source D-7/D-3/D-1 offsets are templates, not chosen defaults. Missing eligible recipient or policy blocks application until resolved.

Flow: `notification_rule.draft` → `notification_rule.preview_impact` → `notification_rule.approve`/`notification_rule.reject` → `notification_rule.apply`. Preview resolves recipients under current permissions, holds, confirmed receipt and applicable state; inaccessible targets appear only as generic unresolved counts. Apply binds the exact approved revision to future events. Existing events retain their rule revision. At dispatch recheck recipient activity/rights, target state and suppression; route to the permitted alternate/escalation, never a former member.

P02 channel fields: purpose, owner, approved destination reference, credential reference/version, supported notification kinds, last attempt/success and current health. A test requires an approved destination and non-sensitive payload; connection success grants no billing authority. Channel test/configuration bindings are **unresolved**. Failure or unknown delivery stays visible in U03; do not claim receipt or resend a possibly delivered message without channel-specific reconciliation. Closed-network operation keeps the required in-app workflow available; optional integrations are not mandatory product plugins.

Walkthrough T02: an approval-waiting event reaches an eligible owner; read, acknowledgement, owner acceptance and blocker resolution are recorded separately. A read notification cannot close a blocker. If the owner loses access before dispatch, the next eligible alternate is used or unresolved-recipient is shown. Collections holds and confirmed payment suppress only the covered reminders; unknown receipt remains unknown. Recurring events use event/rule identity and deduplication policy; missing recurrence policy blocks automatic repeated dispatch rather than inventing timing. Personal snooze cannot remove operational escalation.

# U03/U04 — inbox and unified jobs (OPS03)

Entry: Work → Inbox/Jobs; operators see only authorized targets. Tabs separate notifications, assigned cases and jobs. Fields: job/operation ID, kind (import, calculation, export, ERP, restore), immutable targets, initiator/owner, queued/start/as-of/end, technical state, business result, progress completeness, checkpoint, retryability/cancellation capability, result revision and evidence. Filters include customer, period, kind, owner and state; favorites/recent views remain personal preferences with current authority.

Actions: inspect original result; retry eligible failed unit; request cancellation of cancellable work; assign/accept a case under explicit permission; follow a blocker to its owning screen. Job acceptance never means business success. Cancellation reports whether execution stopped and identifies already committed effects; it does not undo a financial action. A failed polling request leaves last-known status stale, not failed execution. Reconnect queries the original operation before retry. Job query/retry/cancel/case assignment bindings are **unresolved** and must preserve each domain command's idempotency.

Walkthrough T03: a fixed 20-target import finishes with 18 accepted and 2 held. Display succeeded technical processing plus partial business outcome and reasons; do not call all 20 complete. Retrying eligible held inputs cannot resubmit accepted units or confirm billing inputs. An ERP timeout shows waiting with outcome_unknown and a reconciliation action, not Retry issuance. A possible anomaly is labelled suspected until evidence confirms a discrepancy. Unauthorized targets disappear without leaking labels/counts; saved filters and recent history do not preserve revoked access.

# U04 — suspected anomaly review detail (OPS03)

Entry: an authorized U03 anomaly item or Work → Anomalies. Fields: event ID, customer/account/period scope, detection rule/revision, comparison basis and baseline revision, observed/as-of time, evidence completeness, suspected amount/currency (unknown distinct from zero), severity, review state, owner, rationale and linked reconciliation/correction case. The operator must be able to inspect permitted evidence before confirming a conclusion. Detection is not proof of a financial discrepancy.

Flow: inspect fixed comparison evidence → accept review ownership → classify as confirmed normal, requires reconciliation or insufficient evidence → record reason/evidence → send a scoped reconciliation/adjustment request to the owning financial screen when needed → resolve only after the linked authoritative evidence supports closure. A normal classification cannot change amounts; an adjustment request is not approval/application and must follow the registered adjustment lifecycle. For issued/pinned/unknown-dispatch scope, route to governed correction/reconciliation instead of an unissued adjustment. Review state remains separate from technical job state.

Walkthrough T04: a spike caused by a known service change is reviewed against complete comparable evidence and resolved as normal with reason. Missing baseline or unknown amount keeps review waiting for evidence and assigned to an owner, not resolved or called a confirmed discrepancy. A confirmed difference opens the financial case and remains linked until its outcome is verified. Concurrent evidence/revision changes invalidate the old resolution preview. Anomaly query/owner/classification/resolution bindings are unresolved; current rights and shared revision rules apply throughout.
