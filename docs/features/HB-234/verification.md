---
type: VerificationRecord
title: Notification incidents verification
description: Records what was executed and reviewed for the second increment of work item W-30, a register of notification incidents with recurrence links and late signals, self-managed recipient settings, blocking that ignores notification settings, notification planning with quiet hours, digests, re-notification, snooze and one-hop alternates, and dispatch that re-checks routes, snoozes, quiet hours and permissions right before handoff. The record lists results, the mutation check, provisional choices and what remains unverified.
status: draft
---

# Delivered behavior

[HB-234](https://linear.app/hybrid-billing/issue/HB-234) adds the Go package `modules/incidents` from main `686cbb4`.

- **Incidents.** Signals from system actors join the open incident of their condition and subject (count, first and last occurrence, highest severity, blocking once set). A signal that occurred no later than a resolution is recorded as late on the earliest resolved incident covering it and changes nothing else; otherwise a new incident opens and links to the most recent resolved one, so recurrences form a chain and never reopen a resolution. Resolution needs permission on the subject (or a system actor) and is final. `Blocked` lists open blocking incidents and ignores every notification setting.
- **Recipients.** Subscriptions and settings are self-managed: zone, half-open quiet windows (also across midnight), digest cadence, per-severity re-notification and an alternate while away. Snoozes need a subscription (or an active forwarding) and permission, and apply only to the snoozing person's own routes.
- **Planning.** Deterministic, all-or-nothing runs merge copies per target and incident, forward one hop to alternates (`no_target` otherwise), respect snoozes and quiet hours (critical bypasses quiet hours and digests), re-notify after the interval or a severity rise, suppress incidents already pending (critical escalations excepted, pending notifications stale after one hour), anchor digests on the last non-empty handoff or a pending digest, cache permission answers per run and record withholdings only when their state changes.
- **Dispatch and opening.** Dispatch re-checks that each incident is open, that a route is still current and awake, quiet hours and permission, drops copies already handed off since planning, hands off once and starts re-notification and digest timing at handoff. Opening a link checks the target and listing before any lookup (`not_target`) and re-checks permission (`permission_revoked`, or `permission_unavailable` on adapter failure). No sent message is claimed to be recallable.

# Commands and results

All runs were made in the worktree `/private/tmp/hb234` on 2026-11-16 with Go 1.25.7 on darwin/arm64.

| Check | Result |
| --- | --- |
| `gofmt -l modules/` | no output |
| `go vet ./modules/incidents` | pass |
| `go test -race -count=10 ./modules/incidents` | pass |
| `GOOS=linux GOARCH=386 go test -c ./modules/incidents` | builds |
| `go test -race -count=1 ./...` | 47 packages, all pass |
| Coverage | `modules/incidents` 100.0% |
| Mutation testing | 479 mutants of `registry.go` and `plan.go` on the final code: 415 killed, 50 did not build, 14 survived |

A full run on the revision before the last review item left 18 survivors; new tests killed 3 of them, and the six mutants of the route check changed by that item were re-run on the final code and all killed. The 14 survivors were read and judged equivalent: `Round(1)` on clocks, signal and snooze times; a `[][3]` limits array; `>=` for an equal severity (no change); `<=` comparators reached only for distinct keys in incident, subscription and blocked sorts and `Start <= End` where equal windows are refused; and a 128-byte zone name no loader resolves.

Tests use a permission fake and the real zone loader. A property test over 40 seeds of 120 random steps (signals, resolutions, permission toggles, snoozes, clock advances, plan-and-dispatch cycles) compares every raised incident's identity, count, late count, first and last occurrence, severity, recurrence link and blocking, every blocked list and every delivered set with an independent model. Literal tests cover the scenarios listed in [the plan](plan.md), including DST gaps and overlaps and a half-hour-offset zone.

# Reviews

- Design review: three rounds by a separate read-only reviewer; FAIL, FAIL, PASS ([plan](plan.md)).
- Change review: two rounds by a fresh read-only reviewer; FAIL (merged-copy snoozes, digest lag), PASS with one low item applied.

# Provisional choices for the owner

- Severity names, resolution reasons, the 30-day snooze bound, 5-minute minimum re-notification, the one-hour staleness of pending notifications, critical bypassing quiet hours and digests, and escalation not ending a snooze.
- Recipients manage only their own settings; an administrator capability is not built.

# Not verified and residuals

Real transports and handoff to `modules/delivery`, message templates, the `modules/auth` permission adapter, administrator settings, flap guards, the U03 payment reminder and dunning-hold policy, the HB-03 and HB-31 links of QA-08, closing cycles (QA-03), bulk commands, persistence and screens. Once a pending digest is older than one hour a second digest may be planned, so two digests can go out within one cadence if both are dispatched late.
