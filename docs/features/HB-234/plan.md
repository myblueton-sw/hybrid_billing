---
type: ImplementationPlan
title: Notification incidents with recurrence links, per-recipient quiet hours, digests, re-notification, snooze and alternates, blocking that ignores notification settings, and permission checks at dispatch and link opening
description: Defines the second increment of work item W-30, a register of notification incidents keyed by condition and subject that keeps first and last occurrence and repeat counts, opens a new linked incident when a resolved condition recurs, plans notifications per recipient under quiet hours, digest cadence, per-severity re-notification, snooze and alternates, keeps blocking work blocked whatever the settings, and re-checks the recipient's permission when a notification is planned and when its link is opened, without claiming that sent messages can be recalled.
status: draft
---

# Scope and ownership

[HB-234](https://linear.app/hybrid-billing/issue/HB-234) continues work item W-30 (requirement HB-30; QA-08; AUD-QA-08) after HB-233 (`modules/overview`). Base: main `686cbb4`. Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviews the plan and the change, because notifications carry customer subjects to people whose permissions can change. Machine: mac_mini.

Out of scope, recorded as residuals: real mail, chat and push transports and handoff to `modules/delivery`, message templates, the `modules/auth` permission adapter, an administrator capability for other people's settings, flap guards for conditions that resolve and recur repeatedly, the U03 payment reminder and dunning-hold policy (before due, on the due date, overdue), the HB-03 and HB-31 links of QA-08, closing cycles (QA-03, W-31), bulk commands, persistence and screens (U03, M05).

# Observed baseline

- QA-08: NotificationIncident keeps first and last occurrence, repeat count, resolution and recurrence links; a recurrence after resolution is a new incident; quiet hours, digest cadence, per-severity re-notification, snooze and alternate recipients are configurable; blocking work stays; the current permission is checked at sending and at link viewing; recalling external messages is not guaranteed.
- AUD-QA-08: a recurrence after resolution is not hidden by the earlier resolution; blocking holds during quiet hours and snooze; permission revocation applies right before sending and when a link is opened.
- `modules/delivery` already models honest transport outcomes; this register decides who should be told what and when.

# Decision

A new package `modules/incidents` follows the sibling conventions: injected clock, one mutex, workspace isolation, `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (`Refusal{Reason}`), `ErrLimit`; check order form, lookup, repeat, rule, room; assistants refused everywhere. `NewRegistry(Config{Permissions, Zones}, Limits{Workspaces, Incidents, Signals, Recipients, Notifications}, clock)`. `Permissions` answers `Allowed(workspace, person, subject) (bool, error)`; a false answer is a refusal and an error is reported separately as unavailable, never as a revocation. `Zones` loads IANA zones through the injected loader only (portable zone data). Times inside the register are UTC.

## Incidents

`Raise(workspace, Signal{ID, Condition, Subject, Severity, Blocking, Occurred}, actor)` by a system actor. Severity is `info`, `warning` or `critical`; `Occurred` may not be after now (`future_occurrence`). The same signal identifier with the same content returns the incident it was recorded on; with other content it is `ErrConflict`.

- A signal whose occurrence is at or before the resolution time of a resolved incident of that condition and subject is recorded as a late signal on the earliest such resolved incident (late count and times only); its count, severity, resolution and finality are unchanged and nobody is notified. This rule applies first, even when a newer incident is open.
- Otherwise, with an open incident for the condition and subject, the signal joins it: the count grows, the first occurrence becomes the earlier and the last the later of the two, severity becomes the highest seen and blocking stays set once set.
- Otherwise a new incident opens with the next workspace number and links as `Recurs` to the most recent resolved incident of the same condition and subject, so recurrences form a chain and never reopen or overwrite a resolution.

`Resolve(workspace, id, Resolution{Reason, Evidence}, actor)` by a human permitted on the incident's subject (`not_permitted`; adapter failure `permission_unavailable`) or a system actor, with a reason from a closed set (`fixed`, `false_alarm`, `superseded`, `other`) and an evidence reference. It records the resolution time; a resolved incident is final. `Blocked(workspace, subject, actor)` by a system actor, or a human permitted on that subject, lists the open blocking incidents of the subject and ignores every notification setting, snooze and quiet hour, so work stays blocked until resolution.

## Recipients and settings

Recipients manage only their own settings in this increment (`actor == recipient`, `not_self` otherwise); an administrator capability for other people's settings is a residual. `Subscribe(workspace, Subscription{Recipient, Condition, Minimum}, actor)` records a subscription to a condition at or above a severity. `Configure(workspace, Settings{Recipient, Zone, Quiet, Digest, Renotify, Alternate, Away}, actor)` sets an IANA zone; optional quiet hours as a half-open window `[start, end)` of local wall-clock minutes evaluated at the current instant (start equal to end is invalid; a window may cross midnight; DST gaps and overlaps need no extra rule); a digest cadence (`immediate`, `hourly`, `daily`); per-severity re-notification intervals (zero means never again, otherwise at least 5 minutes); an alternate recipient for while `Away` is set (not the recipient itself). Recipients without settings use UTC, `immediate`, no quiet hours and no re-notification.

`Snooze(workspace, incident, until, actor)` by a human who is subscribed to the incident's condition and permitted on its subject, for an open incident, with `until` after now and at most 30 days ahead; a repeat replaces the earlier snooze. A person who currently receives forwarded copies as the alternate of an away subscriber may also snooze the incident for those copies. Snoozing never changes `Blocked`, and it applies only to copies the snoozing person would receive through their own subscriptions (including copies forwarded from them to an alternate) or as an alternate, never to anybody else's subscription.

## Planning and dispatch

`Plan(workspace, actor)` by a system actor computes, at the current time and in a deterministic order, which incidents are due for which target, and records the result all-or-nothing:

- the target of a subscription is its recipient, or the recipient's alternate while the recipient is away; forwarding is one hop only, and when the alternate is away too the copy is withheld as `no_target`;
- copies for the same target and incident from several subscriptions are merged;
- snoozed copies are skipped until the snooze ends; escalation to `critical` does not end a snooze;
- during the target's quiet hours only `critical` incidents are due;
- an incident already handed off to the target is due again only after the target's re-notification interval for its current severity, or at once when its severity rose since that handoff; an incident in a notification for the same target that is planned but not yet handed off, and planned less than one hour ago, is not planned again, except that an escalation to `critical` is planned at once in a new immediate notification;
- `critical` incidents are always delivered immediately; other incidents go immediately or into the target's digest, which is due no earlier than the last digest handoff plus the cadence (missed runs collapse into one digest, never a backlog);
- right before recording, the target's permission on each subject is checked; refused or unavailable incidents are withheld, and a withholding is recorded only when the state of that target and incident changes (from due to withheld or between `refused`, `unavailable` and `no_target`);
- each run creates at most one immediate notification and at most one digest per target; a notification whose incidents were all withheld is not created.

`Dispatch(workspace, notification, actor)` by a system actor is the last step before a transport: it re-checks, for each incident, that it is still open, not snoozed for that target, not inside the target's quiet hours unless `critical`, not handed off to the target in any other notification since this one was planned (unless its re-notification interval has passed since then or its severity rose), and that the target is still permitted; only the eligible incidents are returned, the rest are recorded as withheld under the same state-change rule and limit as planning, and the notification is marked handed off at that time exactly once (a repeat returns the same result; a notification with nothing eligible is handed off empty and sends nothing). Re-notification and digest timing count from handoff, not planning. Handing the result to `modules/delivery` and real transports are residuals.

`Open(workspace, notification, incident, actor)` by the human target: an actor who is not the target, or an incident not listed in that handed-off notification, gets the same `not_target` before any incident lookup; otherwise the permission is re-checked at that moment (`permission_revoked`, or `permission_unavailable` on adapter failure) and the incident's current state is returned. Revoking a permission withholds future dispatches and refuses link opening, but this register never claims that a message already handed to a transport can be recalled.

The `Notifications` limit counts notifications and withholding records together.

# Verification plan

Literal tests for every refusal and repeat; late signals after resolution; dispatch withholding after a revocation between planning and handoff; alternates that are away; DST gap and overlap and a half-hour-offset zone; recurrence after resolution as a new linked incident; repeat counts and first/last occurrence with out-of-order signals; blocking unchanged by quiet hours and snooze; quiet hours crossing midnight in non-UTC zones and DST; critical incidents during quiet hours; re-notification intervals and severity increases; digests; alternates while away; permission withheld at planning and refused at opening; a property test of random signals, resolutions, settings and plans against an independent model of what is due; `-race -count=10`, a 386 build, gofmt, vet, the whole repository with `-race`, 100% coverage and mutation testing.

# Review history

- Design review round 1: FAIL (3 high, 5 medium, 2 low). Folded in: `Dispatch` re-checks right before handoff and timing counts from handoff (1); late signals after resolution stay on the resolved incident, first/last as min/max, recurrence chains (2); self-only settings and subscriptions, permitted resolution and blocked queries (3); `not_target` before lookup and unavailable distinct from revoked, snooze rules (4); one-hop alternates, merged copies, snooze scope (5); state-change withholding records, atomic deterministic planning (6); half-open quiet windows, defaults, injected zones (7); digest anchoring, critical bypass, one notification per target per run, flap guards a residual (8); residuals and wording (9, 10).
- Design review round 2: FAIL (2 medium, 3 low). Folded in: late signals are recorded on the earliest resolved incident whose resolution covers them, before joining an open one (A); critical escalations bypass pending notifications and pending notifications go stale after one hour, with duplicates dropped at dispatch (B); dispatch re-checks quiet hours (C); dispatch withholdings follow the planning rules (D); alternates may snooze forwarded copies (E).
- Design review round 3: PASS. Folded in: dispatch drops an incident handed off to the same target by any other notification since this one was planned, whatever the dispatch order (F).

# Change review resolution

Change review round 1: FAIL (2 medium findings against the plan, 2 test-quality findings, several low). Resolved:

- A merged copy records every subscription behind it; dispatch withholds it as snoozed only when every route is snoozed, so a snooze never affects another person's own subscription (1).
- A planned digest that is not yet handed off anchors the cadence (it goes stale after one hour like other pending notifications), so lagging dispatch never produces a second digest (2); an empty digest handoff does not reset the cadence (3).
- `IncidentOf` and `Withholdings` take an actor and are system-only (4).
- `no_target` is evaluated after the snooze check, and planning a forwarded copy clears the recipient's `no_target` state so the cycle is recorded again (5); permission answers are cached per target and subject within a run (6).
- Plan clarification: at dispatch, quiet-hour deferrals and duplicates already handed off are skipped without a withholding record, because they are not withheld but deferred or already delivered (7).
- Dispatch re-checks that each route is still current (the recipient itself, or its alternate while it is away) and withholds stale routes as `no_target` (8).
- `Resolve` checks permission before the repeat check deliberately, so a person who lost permission cannot even confirm a resolution (9).
- Subscriptions are bounded per workspace (`Limits.Subscriptions`) (10).
- The property model now compares last occurrence, recurrence links and blocking (with blocking signals) (11); literal tests cover merged-copy snoozes, route changes, digest lag, empty digests, stale pending digests, critical and digest items for several targets in order, withholding transitions and the `no_target` cycle, rises while pending, late signals on an earlier resolved incident, read gates, the subscription limit and a refused dispatch that leaves the notification pending (12).

Change review round 2: PASS. Applied the low item L1: a forwarded route at dispatch also requires that the alternate is not away. Recorded L2 as a residual: once a pending digest is older than one hour a second digest may be planned, so two digests can go out within one cadence if both are dispatched late.
