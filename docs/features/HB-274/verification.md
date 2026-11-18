---
type: VerificationRecord
title: HB-274 verification of the O02 relationship change API
description: Tests, property tests, coverage, race, cross builds, mutation results and review outcomes for proposing, validating, activating and discarding relationship plans. Run on mac_mini on 2026-11-17; the test clock is fixed at 2026-11-16.
status: stable
---

# Scope

This record verifies [HB-274](https://linear.app/hybrid-billing/issue/HB-274) against its [plan](plan.md). Every check was run on mac_mini (macOS) with Go 1.25.7, using synthetic data only.

# Results

| Check | Result |
| --- | --- |
| `gofmt -l` and `go vet` | clean |
| `go test -race ./server` (apps/api) | pass, 100.0% statement coverage |
| `-race -count=40` on the new tests | pass |
| `GOOS=linux GOARCH=386` and `GOOS=windows` builds | pass |
| Mutation testing of `relplans.go` and `relbuild.go` (219 mutants on the final code) | 198 killed and 21 survived. Three survivors (reversed sequence order, a page-size boundary, and the self-target check) were then killed by added tests; each was re-run on its own. The other 18 are listed below. |

**Equivalent survivors**
- Five fault-hook error values that are swapped.
- Four `Required` flags that the handler validates again.
- Three zero-value checks: a party lookup, a date that does not parse (which yields a zero date that the `from` check refuses), and the workspace or state filter in stale cleanup.
- An early exit from paging that only saves time.
- A `found` result returned together with an error, which callers ignore.
- `<=` in place of `<` in a sort over unique keys.
- An activation state condition that is still caught by the module.
- `ChangedMembers` called on closed plans, whose errors are ignored.

**Property test.** It generates 400 random timelines. For each one it checks:
- the module accepts the output (the real `AddChunk` canonicalisation);
- every day before `from` is unchanged;
- on and after `from`, the action means what it says: `set` gives exactly the target, `end` gives nothing, `join` includes the target, and `leave` excludes it.

This test found a defect in `join`, which is now fixed.

**What the API tests cover**
- **Propose and activate.**
  - A second administrator activates a proposal, and attribution moves on the effective date.
  - The proposer cannot activate their own proposal.
  - Activating twice, or discarding an activated plan, is refused.
  - The open-plan count reaches zero after activation.
  - The read lists the newest plan first.
- **Refusals with the field named.**
  - Action and kind combinations, a missing or extra target, self-targets, dates, and past dates.
  - Unknown members, and unknown, wrong-kind or inactive targets.
  - A member is refused, and so is an unknown plan.
  - Today is allowed.
- **Plan states.**
  - Rejected plans for no change and for a cycle, with their violations.
  - Joining and leaving a view group.
  - Stale plans, expired plans, and their automatic discard.
  - Discard, and discarding twice.
- **Limits and paging.**
  - The open-plan limit, and 17 failed proposals followed by a success.
  - The metadata cap keeps open plans, and the read always shows them.
  - A member on a later page.
  - The entry limit.
  - A difference with an end date.
- **Failure paths.**
  - A fault at each read and module step.
  - An empty proposer fails closed, in the command and in the read.
  - Resolve against another workspace.

# Reviews

These reviews were done by AI agents, not the owner.

- **Design review** (independent reviewer, read-only, focused on data accuracy, tenant boundaries and permissions):
  - Round 1: FAIL, with five Mediums: orphaned drafts, merging entries with the same target, the retroactive check at activation, the metadata cap with the two-person rule, and unvalidated IDs.
  - Round 2: PASS.
- **Change review:** PASS, with six Lows. All were applied, including a stronger property test that found the `join` defect. H1 is recorded as a residual.

No review is owner acceptance.

# Residuals

These are listed in the plan:
- the API is admin-only;
- two-person approval is between accounts;
- money, publication and ERP impact are not computed;
- there is no scheduling or partial resume;
- each plan covers one member;
- effective dates cannot be in the past;
- new pools and groups cannot be created;
- looking up a member is O(n);
- module plan records grow until restart;
- the party kind of a hierarchy target is not checked.

# Planning cross-check (HYB-291)

Added by the [HYB-291 planning cross-check](../../reviews/HYB-291-planning-cross-check/report.md) on 2026-11-17. A record added by HYB-291; it changes no result above except where marked. The findings come from a read-only auditor; the HYB-291 writer agent (delegated by root) wrote this trace, the residual notes and the copy changes; the HYB-291 change review is their first independent check.

## Requirement trace

Source: `docs/ui/screens/O02/spec.md:117` (HB-274 is the API, HB-275 the screen). Status meanings are in the [cross-check report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). No acceptance test was run.

| ID | Status | Reason | Record |
| --- | --- | --- | --- |
| HB-02 | deferred | Covered: one-member set, end, join and leave plans are drafted, sealed and activated by two people. Not built: allowance preservation, past effective dates, money impact and resuming failed units (HB-14, HB-15, HB-21, HB-23 rows). | HB-274/plan.md |
| HB-14 | deferred | New pools and groups cannot be created; allowance preservation (`screen-binding-register.md:70`) is not addressed. | HB-274/plan.md:253-265 |
| HB-15 | deferred | Effective dates cannot be in the past; there is no scheduling. | HB-274/plan.md:253-265 |
| HB-21 | not connected | Money, publication and ERP impact are not computed, though `O02/spec.md:54` makes it required. | Planning cross-check residuals below |
| HB-23 | deferred | Two-person approval is between accounts; no B09 approval request (`O02/spec.md:72`), scheduling or resume. | HB-275/plan.md:145-155 |

## Planning cross-check residuals (HYB-291)

- **Activation without money impact.** `O02/spec.md:54` makes the money, publication and ERP impact required and `:72` routes the change through a B09 approval request. Plans activate without either. Owner acceptance is needed.
