---
type: VerificationRecord
title: Jobs, leases and cancellation verification
description: Records what was executed and reviewed for the first model of work item W-34, with results, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-180](https://linear.app/hybrid-billing/issue/HB-180/w-34-jobs-tasks-leases-with-fencing-checkpoints-retries-and) adds the Go package `modules/jobs` from main `c4d59c2`: jobs over a task manifest sealed at creation; leases granted fairly across workspaces under two concurrency limits, each with a token from one increasing sequence that fences every later write; attempts bounded by a budget, with backoff; expired leases swept into `pending` or `unknown` and never into a final state; the holder of the latest open attempt still heard after its lease ended; unknown outcomes that wait for a resolution; a cancellation that stops new work at once and becomes `cancelled` only when every open outcome is accounted for; and the six top-level states stored beside a separate detail and an attention count. See the [plan](plan.md).

Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviewed the plan twice before implementation and the implemented change afterwards (see Independent review). Machine: mac_mini.

Nothing calls the package: there are no workers, handlers, commands or storage for it, and it must not be exposed directly. The outbox, the event inbox, event envelopes and webhook security (QA-05), the source event envelope (RT-01) and periodic reconciliation (DB-12), all mandatory for W-34, are not part of this ticket. OPS01, SIM09, EXT03, EXT04, EXT05, AUD-QA-05, AUD-DB-12 and AUD-RT-01 remain NOT_RUN at system level, and W-34 is not complete.

# Executed checks

Executed on 2026-11-13 in the ticket worktree on macOS (Darwin 25.2.0, arm64) with the repository's Go toolchain. No other operating system was tested.

| Check | Result |
| --- | --- |
| `gofmt -l .` | No output |
| `go vet ./...` | No output |
| `go test -race -count=1 ./...` | All twelve packages `ok`, including `modules/jobs`; run again after the last correction |

The acceptance situations of HB-34 that this package can show, each a literal test with a test clock (`registry_test.go`):

| Situation | What the test shows |
| --- | --- |
| A killed worker | A lease that expires returns a repeatable task to the queue after the backoff and makes a task that is not repeatable `unknown`, with the job `waiting` and never granted again until someone answers |
| A duplicate dequeue | A task is never leased while its lease holds; once it was granted again, the first worker's completion, failure and unknown report are stale and change nothing |
| An expired lease whose worker did finish | Its completion is accepted when the task was not granted again, also in the same call as the sweep of its last attempt |
| A partial success | One task succeeded and one failed gives `failed` with the counts in the detail, never `succeeded` |
| An unknown result | `waiting` with the reason `outcome_unknown` when nothing else remains, and an attention count while other work continues |
| A permission withdrawal | A cancellation with that reason stops every grant at once, keeps the job `waiting` with `cancel_pending` while leased work is unaccounted for, and ends `cancelled` only afterwards |
| No state regression | A terminal job is unchanged by every later call; `queued` is never re-entered |

Further literal tests:

- creation: repetition with identical content, also in a full workspace; seven kinds of other content as conflicts; five invalid policy members; eight malformed requests; the limits on budget, tasks and jobs; the same identifier in two workspaces;
- backoff: the exact delays, the cap reached exactly, and no overflow;
- a task through three attempts with the exact not-before times, the checkpoint handed to the next attempt, the budget used up, repeated reports with the same and with other content, and the whole record of that history;
- expiry: the instant before and at the boundary; extension up to the longest lease life; the backoff measured from the sweep; a second unknown report keeping the first code; a late retryable failure on a swept task; a clock that goes backwards;
- unknown outcomes: each resolution, its repetition, another value, an old token, the token of zero, a retry beyond the budget, the holder learning the outcome after reporting it unknown, an operator's retries drawing on the budget;
- cancellation: before any lease; over leased, swept and pending tasks; over leases that expired and were not swept; late reports and resolutions afterwards; a retryable failure during a pending cancellation, with budget left and at its end; repetition with the same and with another reason, also once cancelled;
- selection: a single worker taking one task at a time alternating between two workspaces; both limits inside one call; several types; the pair granted longer ago first; equal limits letting one workspace hold every slot and no more;
- a token of the same job and task in another workspace refused by every worker write and by `Resolve`;
- the record of a history with an unknown report, an expiry, a resolution and a cancellation, entry by entry with task, worker and resulting state;
- the ordered state rule on fifteen combinations; a registry that is not built.

Property test (`property_test.go`, fixed seed): 300 histories of 120 random steps over creations, leases, worker writes with current and old tokens, clock advances, sweeps, resolutions and cancellations. The oracles do not replay the registry's transitions: the test keeps a ledger of grants with each lease's expiry and whether it is still held, and writes the state rule a second time. After every step: the stored state equals the rule over the reported task states with "ever granted" and "cancellation requested" taken from the ledger; attempts equal the ledger's grants and never exceed the budget; tokens only increase; no task is leased while the ledger says its lease holds; a task that is not repeatable is granted again only after a retryable failure or a retry resolution; every accepted worker write and resolution carried the task's latest token; a task whose completion was accepted stays `succeeded` and no task succeeds twice; both concurrency limits hold; the record stays within its bound and its length equals the revision; no final task has an open attempt; a terminal job never changes and holds nothing pending, leased or unknown; `queued` is never re-entered. Reached: 3,962 grants, 481 of them of a task granted before; 4,921 accepted worker writes, 1,747 of them repeats; 4,378 stale writes, 513 by an old token; 344 resolutions; 811 cancellations; 1,016 jobs ending terminal (284 succeeded, 184 failed, 548 cancelled).

Concurrency under the race detector: eight goroutines leasing and completing 120 tasks of three workspaces, each task completed exactly once and each repeated completion answered as a repeat.

Limits of these tests: the property does not look at the wait reason, the not-before times, checkpoints or the order of selection, which only the literal tests pin; its limit counts read the registry's own view; the concurrency test uses a fixed clock, so expiry, cancellation and resolution are not exercised under contention; expectations in the tables were worked out by hand.

# Mutation check

114 single-line mutations of `modules/jobs` were applied one at a time by a script, and the package's tests were run after each. The per-row results were appended to files that were never overwritten.

History:

- Before the change review the list had 109 rows. First run: 99 caught, 7 not caught, 3 did not compile. Of the 7, two showed tests that did not test what they named and were corrected (the backoff after an expiry was only tested with a sweep at the very instant of the expiry; a resolution with the token of zero on a task that was never granted was not tested). Four showed code that could not change a result, which was removed: a final cap in the backoff, a set of pairs already offered in the selection, the clearing of the stored closing report at a grant, and a test in the repeat check of `Fail`. While removing them, the clearing of a stored resolution at a grant was removed too; an existing test failed at once, because without it a later attempt's resolution was answered as a repeat of an earlier one, so it was restored and given a row of its own. One, the refusal to lease from a job with a cancellation requested, is defence that nothing reaches. After rewriting the 3 and running the affected rows again: 108 of 109 caught.
- After the change review's corrections, 6 rows were rewritten for the changed code and 5 were added. All 114 were run again from the start.
- Result: 112 of 114 caught, 2 not caught.

The 2 that are not caught are both checks kept as defence, which no sequence of calls reaches:

| Mutation | Why no test distinguishes it |
| --- | --- |
| 49, leasing from a job for which a cancellation was requested | A cancellation leaves no pending task: pending tasks become `cancelled` or `unknown`, a retryable failure afterwards gives `cancelled`, and a retry resolution is refused |
| 110, the budget not checked at the grant | A task is `pending` only with budget left: the sweep, a retryable failure and a retry resolution each check the budget before making it pending. The change review asked for the check at the grant as well |

| No. | Fault introduced | Result |
| --- | --- | --- |
| 1 | A policy without a lease duration | Caught |
| 2 | A life shorter than the lease | Caught |
| 3 | A policy without attempts | Caught |
| 4 | A policy without a backoff | Caught |
| 5 | A cap below the base | Caught |
| 6 | The backoff doubled once too often | Caught |
| 7 | The backoff overflows | Caught |
| 8 | The cap not applied when the next doubling would pass it | Caught |
| 9 | A linear backoff | Caught |
| 10 | A workspace limit above the type limit | Caught |
| 11 | No limit on the budget required | Caught |
| 12 | Cancelled with unknown outcomes | Caught |
| 13 | Cancelled with leased tasks | Caught |
| 14 | Cancelled at once whatever is open | Caught |
| 15 | A pending cancellation shown by the ordinary rule | Caught |
| 16 | The wrong reason for a pending cancellation | Caught |
| 17 | A partial result shown as succeeded | Caught |
| 18 | Failed with unknown outcomes | Caught |
| 19 | Waiting although work is under way | Caught |
| 20 | An unknown outcome never shown as waiting | Caught |
| 21 | Never queued | Caught |
| 22 | Queued again when nothing is leased | Caught |
| 23 | Cancelled is not terminal | Caught |
| 24 | Identifier length unbounded | Caught |
| 25 | The registry's time goes backwards | Caught |
| 26 | The attention count not kept | Caught |
| 27 | The revision does not move | Caught |
| 28 | A lease holds at its expiry | Caught |
| 29 | A task that is not repeatable returns to the queue on expiry | Caught |
| 30 | An expired task pending beyond the budget | Caught |
| 31 | No backoff after an expiry | Caught |
| 32 | The backoff measured from the expiry | Caught |
| 33 | A sweep makes a task final | Caught |
| 34 | A sweep closes the attempt | Caught |
| 35 | An empty manifest | Caught |
| 36 | A task twice in a manifest | Caught |
| 37 | An input that is text | Caught |
| 38 | A repeated creation with another type | Caught |
| 39 | A repeated creation with another policy | Caught |
| 40 | A repeated creation with other tasks | Caught |
| 41 | A conflicting creation reported as a repeat | Caught |
| 42 | A budget above the limit | Caught |
| 43 | Tasks per job unbounded | Caught |
| 44 | Jobs per workspace unbounded | Caught |
| 45 | Jobs counted across workspaces | Caught |
| 46 | Jobs keyed without their workspace | Caught |
| 47 | Leases not counted | Caught |
| 48 | A type that was not asked for leased | Caught |
| 49 | Leased after a cancellation was requested | Not caught |
| 50 | The limit per type ignored | Caught |
| 51 | The limit per workspace ignored | Caught |
| 52 | Leased during its backoff | Caught |
| 53 | Not leased at the instant its backoff ends | Caught |
| 54 | An unknown task leased again | Caught |
| 55 | A newer job taken before the older one of its pair | Caught |
| 56 | Ties go to the oldest job | Caught |
| 57 | The pair with more leases preferred when granted longer ago | Caught |
| 58 | Leases per call unbounded | Caught |
| 59 | A type asked for twice | Caught |
| 60 | Leasing without a sweep | Caught |
| 61 | The same token for every grant | Caught |
| 62 | The last grant of a pair not noted | Caught |
| 63 | Attempts not counted | Caught |
| 64 | A lease as long as its longest life | Caught |
| 65 | A job with a lease still queued | Caught |
| 66 | The resolution of an earlier attempt kept | Caught |
| 67 | Worker writes without a sweep | Caught |
| 68 | A token of zero accepted | Caught |
| 69 | Any token accepted | Caught |
| 70 | An extension without a lease | Caught |
| 71 | Extensions without an end | Caught |
| 72 | An extension added to the old expiry | Caught |
| 73 | A checkpoint without a lease | Caught |
| 74 | A repeated completion with another checkpoint | Caught |
| 75 | A repeated completion refused | Caught |
| 76 | A completed attempt stays open | Caught |
| 77 | The final checkpoint not stored | Caught |
| 78 | A completion not recorded | Caught |
| 79 | A repeated failure with another code | Caught |
| 80 | A repeated failure with another permanence | Caught |
| 81 | Any failure accepted as a repeat | Caught |
| 82 | A retry beyond the budget after a failure | Caught |
| 83 | A permanent failure retried | Caught |
| 84 | A retryable failure pending after a cancellation | Caught |
| 85 | No backoff after a failure | Caught |
| 86 | A failed attempt stays open | Caught |
| 87 | The closing failure not kept | Caught |
| 88 | An unknown report on a closed attempt | Caught |
| 89 | A second unknown report overwrites the first | Caught |
| 90 | A repeated resolution with another value accepted as a repeat | Caught |
| 91 | A resolution of a task that is not unknown | Caught |
| 92 | A retry after a cancellation | Caught |
| 93 | A retry beyond the budget | Caught |
| 94 | A failed resolution recorded as success | Caught |
| 95 | A retry delayed | Caught |
| 96 | A resolved attempt stays open | Caught |
| 97 | A cancellation written twice | Caught |
| 98 | Another reason accepted as a repeat | Caught |
| 99 | A terminal job cancelled | Caught |
| 100 | A cancellation without a sweep | Caught |
| 101 | A swept task closed by a cancellation | Caught |
| 102 | A leased task untouched by a cancellation | Caught |
| 103 | A cancellation closes the attempt | Caught |
| 104 | Pending tasks survive a cancellation | Caught |
| 105 | Any cancellation reason | Caught |
| 106 | Overdue leases not shown | Caught |
| 107 | A lease at its expiry not overdue | Caught |
| 108 | The record aliases what it returns | Caught |
| 109 | An unknown resolution value | Caught |
| 110 | The budget not checked at the grant | Not caught |
| 111 | A repeated failure report tells an outdated state | Caught |
| 112 | A resolution not kept for its repeat | Caught |
| 113 | Every transition recorded as a grant | Caught |
| 114 | An expiry recorded without its worker | Caught |

# Deviations from the plan

- A repeated closing report answers with the state the task has now, not with the state the first answer gave; the plan was changed to say so after the change review.
- A cancellation adds one entry to the record for the job, not one per task.
- The clock is not read by calls that are answered without looking at a lease.

# Provisional choices and undecided matters

The owner has not decided these; the implementation uses the labelled rules of the plan: lease duration, lease life, budget and backoff as required parameters without defaults; the cancellation boundary (new work stops at once, open attempts become `unknown`, the job is `cancelled` only when they are accounted for); which tasks are repeatable, with "not repeatable" as the default; `waiting` shown only when nothing else remains, with the attention count beside it, a literal deviation from QA-05's wording; fairness by workspace and type only; no retention or eviction of terminal jobs; no queue delay targets; who may cancel and resolve, and worker authority, not modelled.

# Limits

- The package authorizes nothing and is reachable only in process; it must not be exposed before commands and workers exist for it.
- Exposure blocker: terminal jobs are never evicted, a full workspace refuses new jobs, and every lease costs time in proportion to all stored tasks.
- That a worker stops its outward effect when its lease is lost, reports `Unknown` when it cannot confirm that nothing happened, and marks only idempotent work as repeatable cannot be checked here. A repeatable task's effect can occur as many times as the budget allows.
- After a `retry` resolution of a task that is not repeatable, the only protection against a second effect is the target's idempotency key.
- A job of a type that no worker asks for stays `queued` unnoticed; a job whose workers all died shows its overdue leases on a read but is swept only by a call.
- The clock is called while the registry's lock is held.
- State is in memory and lost on restart, including tokens.
- The per-row mutation results are in the session's scratch files, not in the repository.

# Independent review

The design review of 2026-11-13 returned PASS with findings (5 HIGH, 6 MEDIUM, 2 LOW) and judged the plan not implementable until the first five were corrected; it was rewritten. The re-review of the same day returned PASS with findings (2 HIGH, 5 MEDIUM, 4 LOW), all adopted; the text after it was not reviewed again as a design, but the change review read the final plan against the code.

The change review of the same day returned PASS with findings: no blocker, no HIGH, 3 MEDIUM and 5 LOW. It found no sequence in which two leases are valid for one task, a stale write is accepted, the budget is exceeded, a task that is not repeatable is granted again without a confirmation, a task succeeds twice, a final task has an open attempt, or a job of another workspace is reached, and it checked each simplification made after the first mutation run. All eight findings were handled as the plan's change review resolution lists. The corrections made after it (the budget check at the grant, the present state in a repeated answer, the removal of an unread field, and tests) were not reviewed again; they are covered by the tests and by the full mutation run after them.

All three reviews were read-only and executed nothing: every executed result in this record is the author's.
