---
type: ImplementationPlan
title: Jobs, tasks, leases with fencing, checkpoints, retries and conditional cancellation
description: Defines the first model of work item W-34, a registry of jobs over sealed task manifests in which a task is worked only under a lease with a fencing token, an unknown outcome waits for an explicit resolution, and the six top-level job states are stored apart from the detail.
status: draft
---

# Scope and ownership

[HB-180](https://linear.app/hybrid-billing/issue/HB-180/w-34-jobs-tasks-leases-with-fencing-checkpoints-retries-and) is the first increment of work item W-34 (requirement HB-34, with the job-state part of QA-05). It is the ninth increment built under the owner's instruction of 2026-11-12. Base: main `c4d59c2`. Root (Claude Code) owns planning, implementation, tests and integration. A separate read-only reviewer reviews this plan before implementation and the change afterwards, because the model decides whether work that bills is done once, and is keyed by tenant. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-13:

- Requirement HB-34: queued, running, waiting, succeeded, failed and cancelled are separated from business detail states, and an acceptance or an unknown result is not completion. Events may arrive duplicated and out of order. Proposed direction: Job, Task, Checkpoint and Lease, a transactional outbox and an event inbox; a task lease with a fencing generation blocks an old worker's result; retry budget, backoff, retained failures and conditional cancellation. Scale: concurrency and fairness per job type, customer, source and ERP; a chunk's result and its checkpoint are committed together; overall success is published only after the manifest is confirmed complete.
- Acceptance of HB-34: no loss, no doubled billing and no state regression on a killed worker, a duplicate dequeue, an expired lease, a partial success, an event replay or expiry, and a permission withdrawal. Linked source cases, all NOT_RUN: OPS01, SIM09, EXT03, EXT04, EXT05.
- Open decisions of HB-34, not made: the task lease, queue delay targets, retry and event retention, and the cancellation boundary.
- QA-05: the top-level job states are exactly queued, running, waiting, succeeded, failed, cancelled; partial is a business detail or aggregate and a partial completion is never shown as overall success; an unknown ERP result is waiting with a reason. The rest of QA-05 concerns event envelopes and webhook security. AUD-QA-05 is NOT_RUN.
- DB-12 (periodic ERP reconciliation) and RT-01 (source event envelope) are mandatory detail contracts of W-34 and concern events and connectors.
- Existing code: `modules/commands` (HB-176) returns an accepted outcome with a job identifier and holds no job; nothing in the repository holds jobs. Everything is in memory; the business database and the queue technology are not selected.
- The planning sources are untracked local files of the primary checkout and were read there.

# Decision

One Go package, `modules/jobs`, standard library only, with an in-memory registry. It records jobs, hands out task leases and accepts task reports; it runs no work, starts no goroutine, calls no handler and has no entry point wired in this increment. Workers, which do not exist yet, are callers.

Not in this increment, each mandatory for W-34 and left to follow-up tickets: the transactional outbox, the event inbox, event envelopes and webhook security (QA-05), the source event envelope (RT-01) and periodic reconciliation (DB-12).

## Jobs and their manifest

A job belongs to one workspace and has a caller-assigned identifier, a type (an identifier, the unit of concurrency), and a manifest: the complete, ordered list of its tasks, fixed when the job is created. A task has an identifier unique in the job, an opaque input reference, and a flag saying whether it is repeatable; the flag's zero value is "not repeatable".

`Create(workspace, job, type, policy, tasks)` writes the job with all its tasks or nothing. An empty manifest and a task identifier used twice are refused. Creating a job whose identifier exists in the workspace returns the stored job without writing when type, policy and tasks, in their order, are identical, and is a conflict otherwise; this is decided before the workspace's job limit is looked at, so that the caller that received a job identifier from a command can always repeat the creation. A job identifier of another workspace is never looked at.

The manifest is sealed by creation: no task is added or removed later, so "all tasks done" is decidable. Stated limit: a job whose list of chunks is discovered while working needs a planning job that creates it; an open-then-seal variant is not built, and success is defined on "the sealed manifest" so that such a variant is not precluded.

`policy` is required in full, without defaults, because every member is the owner's open decision: lease duration (positive); the longest total life of one lease including extensions (at least the duration); attempts per task (the budget, from 1 to a limit given at construction); the backoff base (positive) and its cap (at least the base). The delay before the next attempt after attempt `n` is `base × 2^(n−1)`, saturating at the cap.

A type is any identifier; a job of a type that no worker asks for stays `queued` and nothing detects that (stated limit).

## Task states

| State | Meaning |
| --- | --- |
| `pending` | Not leased; eligible when its not-before time has come |
| `leased` | Held under a lease with an expiry |
| `succeeded` | Final |
| `failed` | Final: the budget is used up or a failure was reported as permanent. The last failure code is kept |
| `unknown` | The outcome of the last attempt is not known. Never granted again unless a `retry` resolution, or its holder's `Fail` confirming that nothing happened, comes first |
| `cancelled` | Final: not leased when the job was cancelled; earlier attempts may exist |

## Attempts and the fencing token

An attempt is a granted lease, counted when it is granted. A task's attempt count never exceeds the budget: a grant is refused, and so is a `retry` resolution, when the count equals the budget. A `retry` resolution does not itself count; the grant that follows it does, so an operator's retry draws on the same budget.

Every grant draws a token from one sequence of the registry that only increases. The token, not the attempt number, is the fence: it is unique across tasks, jobs and workspaces, so a tuple that belongs to another task or another workspace can never match. A worker write names workspace, job, task and token, and all four must match the task's latest grant.

The token is also what a worker passes to a system it calls, where that system can check that it only increases. The idempotency key towards such a system is (job, task), stable across attempts; the token is only the fence. That a worker stops its outward effect when it has lost its lease cannot be enforced here (stated limit).

A lease records the worker identifier it was granted to; the identifier is recorded in the transition and is not checked on later writes (worker authority is not modelled).

## Leasing

`Lease(worker, types, limit)` first sweeps expired leases of every job, then returns up to `limit` leases for tasks of the given job types, each with workspace, job, task, the input reference, the last checkpoint, the token, the attempt number and the expiry. A task is eligible when it is `pending`, its not-before time has come, its attempts are below the budget, its job is not terminal and has no cancellation requested, and both concurrency limits allow it.

Concurrency limits, required at construction: leased tasks per job type, and leased tasks per job type within one workspace, the second not greater than the first. When the two are equal one workspace can hold every slot of a type until its leases end, which the longest lease life bounds (stated).

Selection, one lease at a time until the limit is reached or nothing is eligible: among the pairs (workspace, type) of the requested types that have an eligible task, take the pair with the fewest leased tasks; among those the pair whose last grant is oldest by the registry's grant sequence, a pair never granted counting as oldest; among those the one with the earliest created job; then, inside the pair, the eligible task of the earliest created job in manifest order. "Earliest created" is by a creation sequence of the registry, not by clock time, which can tie. Inside a pair the order is strictly oldest job first, intended: a large old job is worked before newer ones of the same workspace and type, except while its tasks wait out a backoff. Because the pair just served becomes the most recently granted, a single worker that takes one task at a time alternates between workspaces instead of draining the oldest job first. The rule is deterministic.

Provisional: the unit of fairness is the pair of workspace and type, so a workspace whose jobs are spread over several types gets a share per type. Fairness by customer, by source and by ERP, which the requirement also names, needs attributes that jobs do not carry yet.

`Extend(lease)` moves the expiry of a current lease that has not expired to now plus the lease duration, but not beyond the grant time plus the longest lease life.

## What a worker reports

Calls on the registry are serialized, so of two that concern one task the first decides. An attempt is open from its grant until one of these closes it: an accepted `Complete`; an accepted `Fail`; a resolution; a later grant of the same task. A task becomes final only through one of the first three or by being cancelled while it has no open attempt, so a final task never has an open attempt and every task of a terminal job is closed to reports. While the latest attempt is open, its holder may report, also after the lease expired or was ended by a cancellation, because no other worker has been given the task and the holder's first-hand knowledge is better than none. `Checkpoint` and `Extend` are different: they need a lease that has not expired.

- `Checkpoint(lease, checkpoint)` stores an opaque checkpoint identifier with the task. The next lease of the task returns it.
- `Complete(lease, checkpoint)` makes the task `succeeded` and stores its final checkpoint in the same write. This is the registry's half of the requirement's "chunk result and checkpoint committed together"; atomicity with the business result needs the store that is not selected yet.
- `Fail(lease, code, permanent)` means the worker confirms that the attempt had no effect. A permanent failure, or one at the end of the budget, makes the task `failed` with the code. Otherwise the task becomes `pending` with a not-before time of now plus the backoff, also when a sweep had already made it `pending` (the write is then recorded once, with the new not-before time) or `unknown`. When a cancellation has been requested, an accepted retryable `Fail` makes the task `cancelled` instead of `pending`, because nothing is granted any more. A worker that cannot confirm "no effect" must report `Unknown`, never a retryable `Fail`; this is an obligation of the worker that the registry cannot check.
- `Unknown(lease, code)` makes the task `unknown`, ends its lease, which frees its concurrency slot, and leaves the attempt open: the worker called something and cannot tell whether it took effect. If it learns the outcome later it may still report `Complete` or `Fail`. An `Unknown` for a task that is already `unknown`, whether by an earlier report, by a sweep or by a cancellation, writes nothing and the first code is kept.

Repeating a report: a `Complete` or `Fail` that carries the token of the attempt it closed, with the same content, writes nothing and answers as a repeat with the state the task has now, so that a worker that lost the response can tell success from a lost lease. The state can differ from the one the first answer gave: a retryable `Fail` answered `pending`, and its repeat after a cancellation answers `cancelled`. This holds only while that attempt is the task's latest; the task keeps the closing report of its latest attempt and no older one. Every other write is refused as stale and changes nothing beyond the sweep that precedes every write: a token that is not the latest, including a repeat after a later grant; a `Complete` with another checkpoint or after an accepted `Fail`; any report after a resolution closed the attempt.

## Expired leases

A lease is expired when the registry's time is at or after its expiry. The registry's time never goes backwards: a clock reading earlier than the last one is replaced by the last one.

Expired leases are swept before anything else in every operation that writes: for the addressed job in a worker write, `Cancel` and `Resolve`; for every job in `Lease` and in `Sweep()`, an explicit operation for a housekeeper, since nothing here runs on a timer. A sweep changes only the jobs it covers. For an expired lease:

- a repeatable task with budget left returns to `pending`, with a not-before time of the sweep time plus the backoff (measured from the sweep, not from the expiry);
- a repeatable task at the end of its budget becomes `unknown` with the code `lease_expired`. A sweep never makes a task final: the holder may have finished, and a final `failed` would refuse its own late `Complete` and show a done effect as a failure;
- a task that is not repeatable becomes `unknown` with the code `lease_expired`: the worker may have died after its effect, so repeating it could bill twice.

In every case the attempt stays open for its holder as described above. Consequences, stated because they follow from the rule and are not softened by it:

- "repeatable" means that the effect is idempotent at its target: a repeatable task's effect can occur as many times as the budget allows;
- a worker that reports `Complete` after its lease expired is accepted if the task has not been granted again, and is stale if it has; in the second case the effect happened and a second worker is repeating it, which is safe only for a repeatable task;
- which task kinds are repeatable is decided where jobs are defined, not here (provisional).

A read sweeps nothing. It returns the stored state, the registry's time and the number of the job's leases that are past their expiry and not yet swept, so that a job whose workers have all died is not shown as healthy `running` without a sign.

## Unknown outcomes

An `unknown` task is resolved only by `Resolve(workspace, job, task, token, resolution)`, where the token is that of the attempt whose outcome is unknown, so that a late or repeated resolution cannot hit a later attempt, with one of: `succeeded` (the effect was confirmed), `failed` (confirmed not to have happened and not to be repeated), or `retry` (confirmed not to have happened; back to `pending` at once if the budget allows, else refused). `retry` is refused when a cancellation has been requested. A resolution closes the attempt; a worker's report that arrives after it is stale, so after a `retry` of a task that is not repeatable the only protection against a second effect is the target's (job, task) idempotency key, which is why `retry` asserts that nothing happened. Repeating a resolution with the same token and the same value writes nothing and succeeds; with another value it is refused. Who may resolve, and on what evidence, belongs to the commands that will wrap this operation.

## Job states, stored apart from the detail

The top-level state is stored and is exactly one of the six. It is recomputed and stored in the same write as every change, by this ordered rule, the first line that applies:

| Order | Holds when | State |
| --- | --- | --- |
| 1 | A cancellation was requested and no task is `leased` or `unknown` | `cancelled` |
| 2 | A cancellation was requested | `waiting`, reason `cancel_pending` |
| 3 | Every task is `succeeded` | `succeeded` |
| 4 | No task is `pending`, `leased` or `unknown` | `failed` |
| 5 | No task is `pending` or `leased` | `waiting`, reason `outcome_unknown` |
| 6 | No lease was ever granted | `queued` |
| 7 | Otherwise | `running` |

The inputs are the task states, whether a cancellation was requested, and whether any lease was ever granted; with a manifest that is never empty the rule is total and gives exactly one state.

Consequences, each pinned by tests:

- overall success needs the whole sealed manifest; a job with succeeded and failed tasks is `failed`, and the detail shows the counts, so a partial completion is never `succeeded`;
- an unknown outcome never becomes a terminal state by itself: while any task is `unknown` the job is `waiting` or `running`, never `succeeded`, `failed` or `cancelled`;
- terminal states are final and nothing about a terminal job changes afterwards. `queued` is never re-entered. `running` and `waiting` may alternate, which is not a regression: both are non-terminal;
- there is no fail-fast: a permanently failed task leaves the job `running` until its other tasks end.

Beside the state the job stores an attention count, the number of its `unknown` tasks, because line 7 shows `running` while other tasks are still pending or leased: a large job would otherwise hide an unknown outcome until everything else drained. This narrower `waiting` is provisional and a literal deviation from QA-05's "unknown result is waiting"; a reader that must not miss an unknown outcome reads the attention count.

The detail is a separate value: the number of tasks in each task state, and per task its state, attempts, latest token, last code and checkpoint.

## Conditional cancellation

`Cancel(workspace, job, reason)` takes a reason from a closed list (`requested`, `permission_withdrawn`, `superseded`). Its condition is the job's state, not a revision, so that it cannot lose a race against workers indefinitely: the checks run in this order: a job for which a cancellation is already recorded succeeds without writing when the reason is the same, also once it is `cancelled`, and is refused when the reason differs; otherwise a terminal job is refused; otherwise the cancellation is accepted.

In one write, after the sweep: the cancellation is recorded as requested; every `pending` task without an open attempt becomes `cancelled`; every `leased` task, and every `pending` task whose latest attempt is still open (its lease expired and its holder has not reported), becomes `unknown` with the code `cancelled_while_open`, any lease ended and the attempt still open, because its worker may already have had an effect. No task is granted afterwards. The job is `cancelled` at once when nothing was leased or unknown, and otherwise `waiting` with the reason `cancel_pending` until each unknown task is closed by its holder's late report or by a resolution with `succeeded` or `failed`.

So a cancelled job never carries an unknown outcome or a pending task, and "cancelled" does not claim that a leased task did not happen: a job whose leased tasks all complete late ends `cancelled` with those tasks `succeeded`, because line 1 of the rule precedes line 3. The boundary is the owner's open decision and this is the conservative reading: new work stops at once, and the job does not reach a terminal state while an effect may be unaccounted for.

A permission withdrawal is one reason for cancellation. The registry holds no permission and rechecks none: the command that acts on a caller's behalf decides through the gate, and a worker's own authority is not modelled yet (stated limit; the acceptance's permission-withdrawal case is NOT_RUN at system level).

## Revisions, time and records

Every job has a revision that increases by one with every write that changes its state, its attention count or a task's state. `Extend` and `Checkpoint` overwrite the lease expiry and the checkpoint and change neither the revision nor the record list, so their number is not bounded by anything and bounds nothing.

Every job keeps an append-only list of transitions (revision, time, task if any, worker if any, what happened, the resulting job state). The list needs no limit of its own: per attempt there are at most a grant, an expiry sweep, a first `Unknown`, and one closing report or resolution, and per task one cancellation, so the list holds at most four entries per attempt and one more per task, plus one for the job's creation and one for its cancellation; tasks and budget are both limited. Writes that change nothing (repeats, a second `Unknown`) add no entry. No write is ever refused for lack of room in it.

The registry's clock is injected and read at most once per call, before the sweep; a call that is answered without looking at any lease (a repeated creation or cancellation, a refusal for a malformed or unknown job) does not read it. The clock is called while the registry's lock is held, so a clock that blocks stalls every caller and one that calls back into the registry deadlocks (stated limit).

## Tenant boundary and limits

Every operation on a job takes the workspace and addresses only that workspace's jobs; a job of another workspace is not found, and a worker write whose workspace, job, task and token do not all match one grant is stale. `Lease` and `Sweep` cross workspaces by design and look at every stored job, terminal ones included, which costs time in proportion to all tasks of all jobs for every lease granted while nothing is evicted (stated limit); because workers and the housekeeper serve all of them; a lease names its workspace. The registry authorizes nothing and must not be exposed directly (stated limit).

Required and positive at construction, without defaults: jobs per workspace, tasks per job, the largest budget a policy may ask for, leased tasks per type, leased tasks per type and workspace, and leases per `Lease` call. Terminal jobs are never evicted in this increment; retention is the owner's open decision, and a full workspace refuses new jobs (stated limit, an exposure blocker like the one of the command layer).

Identifiers follow the project's identifier rule (the package has its own copy, stated). Input references, checkpoints and codes are identifiers, never free text.

## What is not decided here

Lease duration and life, retry budget and backoff (required parameters); queue delay targets (not modelled); retention of terminal jobs and of failures; the cancellation boundary; which tasks are repeatable; the narrower `waiting`; fairness by customer, source and ERP; who may cancel and resolve; worker identity and authority.

# Allowed files

`modules/jobs/*` (new); `docs/features/HB-180/*`.

# Verification

- Literal table tests with a test clock: creation, its repetition also in a full workspace, an empty manifest, repeated task identifiers, each invalid policy member; leasing order under both concurrency limits, across workspaces with a single worker taking one task at a time, and across several types; extension up to the longest lease life; checkpoint, completion, retryable and permanent failure with the exact backoff times and the saturated cap; the budget used up by failures, by expiries and by operator retries; an unknown outcome and each resolution; a repeated `Complete` and `Fail`; every stale write (an old token, a token of another task or workspace, a closed attempt, `Checkpoint` and `Extend` on an expired lease) refused with nothing changed; expiry exactly at the boundary; a clock that goes backwards; a `Complete` in the same call as the sweep of the last attempt, accepted; a second `Unknown`; a late retryable `Fail` on a swept task with its new not-before time; a repeat after a later grant, stale; a resolution with an old token, repeated, and with another value.
- The acceptance situations at package level: a killed worker (lease expiry) for a repeatable and for a non-repeatable task; a late `Complete` accepted when the task was not granted again and stale when it was; a duplicate dequeue (two `Lease` calls never return the same task while a lease holds); a partial success never shown as succeeded; a permission withdrawal as a cancellation whose leased tasks keep the job `waiting` until they are accounted for; no state regression.
- Job states: each line of the ordered rule, each of the six states reached, the attention count, the detail counts, the overdue count on a read, and nothing changing on a terminal job.
- Cancellation: repeated, with another reason, on a terminal job; pending and leased tasks; a cancellation over a lease that expired and was not yet swept; late reports and resolutions afterwards; `retry` refused; a retryable `Fail` during `cancel_pending` making the task `cancelled`; a cancellation over a swept repeatable task followed by its holder's late `Complete`; a repeated `Cancel` after the job became `cancelled` at once.
- Limits and the tenant boundary.
- Properties with a fixed seed over random histories of creations, leases, worker writes with current and stale tokens, clock advances, sweeps, resolutions and cancellations, checked by oracles that do not replay the transition function: a ledger of grants in the test from which it follows that no task has two grants open for leasing at once, attempts never exceed the budget, tokens strictly increase, every accepted worker write carried the task's latest token, a task that is not repeatable is never granted again unless a retryable `Fail` or a `retry` resolution came between, and no task becomes `succeeded` twice; the concurrency limits hold after every `Lease`; the stored job state equals the ordered rule written again in the test, with "ever granted" and "cancellation requested" taken from the test's ledger; the transition list stays within its bound; a terminal job has no `pending`, `leased` or `unknown` task; a task whose `Complete` was accepted is `succeeded` ever after; a terminal job never changes; `queued` is never re-entered; the detail counts sum to the manifest.
- Concurrency under the race detector: many goroutines leasing and completing, with every task completed exactly once.
- Mutation check by a script with per-row results appended to files that are never overwritten.
- `gofmt -l`, `go vet ./...`, `go test -race -count=1 ./...`.

Remaining NOT_RUN at system level after this ticket: OPS01, SIM09, EXT03, EXT04, EXT05, AUD-QA-05, AUD-DB-12, AUD-RT-01. This ticket does not complete W-34.

# Deferred

The transactional outbox and the event inbox with deduplication and ordering; event envelopes and webhook security (QA-05); the source event envelope (RT-01); periodic ERP reconciliation (DB-12); real workers, handlers and the command wiring of accepted outcomes to jobs; worker authority; an open-then-seal manifest; retention; persistence and the queue technology.

# Design review resolution

The independent design review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker, 5 HIGH, 6 MEDIUM and 2 LOW, and judged the fencing rule sound and the plan not implementable until the first five were corrected. All thirteen were adopted in the sections above.

| Finding | Resolution |
| --- | --- |
| D1 the transition limit could strand work and block resolution | No limit on the list: it is bounded by construction; `Extend` and `Checkpoint` add nothing to it; no write is refused for room |
| D2 fairness could starve a workspace | Ties go to the pair granted least recently; selection across types defined; single-worker test |
| D3 a terminal cancelled job could carry unknown outcomes | A cancellation with leased tasks is `waiting` with `cancel_pending` and becomes `cancelled` only when none is unknown |
| D4 cancellation and resolution could conflict indefinitely | `Cancel` is conditional on the job's state, `Resolve` on the token of the unknown attempt |
| D5 attempt accounting | An attempt is a grant, counted at the grant, bounded by the budget; operator retries consume it |
| D6 a retryable failure re-running a task that is not repeatable | `Fail` defined as confirmed no effect; the obligation stated; the property added; the idempotency key stated |
| D7 sweep coverage | Every writing operation sweeps; explicit `Sweep`; overdue count on reads; boundary and monotonic time defined |
| D8 late and repeated reports discarded | The holder of the latest open attempt may still report; repeated closing reports return the stored outcome; consequences of "repeatable" stated |
| D9 the state table not total | Ordered rule; empty manifests refused |
| D10 an unknown outcome behind `running` | Attention count stored beside the state; the deviation labelled |
| D11 tuples colliding across tasks and workspaces | Registry-wide token sequence; the worker identifier recorded |
| D12 creation holes | Duplicates, order, policy validation, the full-workspace repeat, type typos and sealing stated |
| D13 overstated or dropped items | The registry half of the commit, customer fairness, equal limits, backoff from the sweep and no fail-fast stated |

# Design re-review resolution

The narrow re-review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker, 2 HIGH, 5 MEDIUM and 4 LOW; nine of the thirteen first findings resolved and four partly. All eleven new findings were adopted in the sections above; the text after them was not reviewed a third time as a design.

| Finding | Resolution |
| --- | --- |
| E1 a sweep making a task final against its holder's late report | A sweep never makes a task final: at the end of the budget the task becomes `unknown` |
| E2 a retryable `Fail` during a cancellation leaving a dead pending task | It makes the task `cancelled` |
| E3 a cancellation closing a swept task whose holder may still report | Such a task becomes `unknown` on cancellation |
| E4 repeated `Cancel` contradictory | Order of the checks stated |
| E5 writes that change nothing; the bound | A second `Unknown` writes nothing; the late retryable `Fail` defined; the bound recounted |
| E6 a repeat after a later grant | Stale; only the latest attempt's closing report is kept |
| E7 the table against the rule | Reworded |
| E8 reports after a resolution; repeated resolutions | Stale, stated with its consequence; identical resolutions idempotent |
| E9 selection details | Creation sequence, the pair as the unit, oldest job first and the sweep's cost stated |
| E10 lease end on `Unknown`; cancelled with succeeded tasks | Stated |
| E11 wording of the operator's retry | Fixed |

# Change review resolution

The independent change review of 2026-11-13 (read-only, nothing executed) returned PASS with findings: no blocker, no HIGH, 3 MEDIUM and 5 LOW. It found no sequence in which two leases are valid for one task, a stale write is accepted, the budget is exceeded, a task that is not repeatable is granted again without a confirmation, a task succeeds twice or leaves `succeeded`, a final task has an open attempt, or a job of another workspace is reached.

Simplifications made after the first mutation run, each checked by the review: selection keeps no set of pairs already offered, because on a tie the earlier job of a pair stays; a grant does not clear the stored closing report, because it is read only on a closed attempt and every closing overwrites or clears it; the repeat check of `Fail` does not ask whether the stored report was a completion, because a completion stores no code and a code is never empty; the backoff needs no final cap, because the loop never doubles past it. Kept as defence although nothing reaches them: the refusal to lease from a job with a cancellation requested, and the budget check at the grant.

| Finding | Resolution |
| --- | --- |
| C1 the budget not checked at the grant | Restored at the eligibility check, as the plan says |
| C2 a token of the same job and task in another workspace untested | Test added for every worker write and for `Resolve` |
| C3 the record asserted for four of eight events | Test added over a history with an unknown report, an expiry, a resolution and a cancellation |
| C4 a repeated `Fail` answering an outdated state | A repeat answers the task's present state; stated above; test added |
| C5 a stored token never read | Removed |
| C6 the clock under the lock and not read on every call | Stated above |
| C7 property oracles weaker than claimed | The test's ledger keeps each lease's expiry and holding; every final task is checked for a closed attempt; the errors of a resolution are classified |
| C8 small gaps and the mutation evidence | Tests added for a retryable failure at the end of the budget during a pending cancellation and for equal limits; the mutation run is in the verification record and its per-row files stay in the session's scratch directory (stated limit there) |
