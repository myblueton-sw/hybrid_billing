---
type: ImplementationPlan
title: Scale gate register with P0 profiles, hardware records, budgets that stay proposals until approved with hardware and a distribution, harness-recorded measured runs and a readiness read that never shows NOT_RUN as passing
description: Second increment of work item W-39. A register holds the HB-39-V P0 scale gates as profiles with independent axes, hardware records and per-gate metric budgets. Budgets stay proposed until a second human approves them with a distribution evidence digest. Bound harnesses record measured runs. Each run gets a verdict that can be passed only under an approved budget on the same hardware, with exact correctness and the profile's scale actually reached. A readiness read lists every P0 gate that has not passed.
status: stable
---

# Scope and ownership

[HB-251](https://linear.app/hybrid-billing/issue/HB-251) is the second increment of work item W-39, after HB-250. It covers the recording side of HB-39-V:
- **P0 gates.**
  - an end-to-end run of at least 100,000 rows;
  - sustained inflow above 1,000,000 rows a day;
  - a monthly close of at least 30,000,000 rows;
  - interruption and resume;
  - catch-up.
- **Recorded measures.**
  - correctness;
  - maximum RSS;
  - DB IO, WAL and lock waits;
  - p95 and p99;
  - throughput;
  - completion time and resume time.
- **Rules.**
  - SLA numbers stay proposals until approved together with hardware and the real distribution.
  - NOT_RUN is never shown as passing.

It does not run any measurement. A harness runs the load and records the results here.

Base: main `9a04497`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the design and the change, because the register decides when a release gate counts as passed. Machine: mac_mini.

Allowed files: `modules/perfgate/*` and `docs/features/HB-251/*`.

# Observed baseline

- **HB-39-V.** The P0 gates are listed above. Scale axes are customers, accounts, shared pools, skew and fan-out, each set independently, plus concentrated arrival, accumulated retention, failure recovery and concurrent closes. Correctness, maximum RSS, DB IO/WAL/lock, p95/p99, throughput, completion time and resume time are recorded. SLA numbers are proposals until approved with hardware and real distributions. NOT_RUN must not be marked as passing.
- **Source plan.** For performance acceptance, the hardware, monthly row counts, concurrency and warm or cold conditions are fixed, and p50/p95/p99, throughput, memory and impact are measured on the same hardware and data. The 10,000-customer target is a design goal, not a measured result.
- **Siblings.**
  - `modules/simulation` (HB-248) binds harness identities and refuses self-verification.
  - `modules/publication` (HB-250) provides the bounded building blocks that the gates will exercise.

# Decision

A new package `modules/perfgate` follows the sibling conventions:
- one mutex and an injected clock that never goes backward;
- `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (with `Refusal{Reason}`) and `ErrLimit`;
- check order: form, lookup, repeat, rule, room;
- assistants refused as `ai_not_allowed` and system actors as `not_human` where a human is required;
- numbered proposals approved by a second human with a full recheck, and withdrawal by the proposer;
- register-wide pending caps: 1,000 in total and 250 per proposer;
- 1,000 applied proposals kept. Withdrawn proposals are deleted at once.

It imports only the standard library.

## Proposals

| Kind | Payload | Effect |
| --- | --- | --- |
| `profile` | `Profile` | An immutable scale profile for one gate |
| `hardware` | `Hardware` | An immutable hardware record |
| `budget` | `Budget` | A proposed budget for a profile, which becomes the profile's active budget |
| `approval` | `Approval{Budget, Distribution}` | Approves a proposed budget |
| `harness` | `Harness{ID, Identity}` | Binds a system identity that may record runs |
| `retire` | `Retire{Profile}` | Retires a profile. It leaves the gate status, but its history stays readable, `Record` refuses it with `retired`, and the gate view still lists its latest status under `Retired` |

`RevokeHarness(id, actor)` lets one human revoke a harness at once.

## Gates and profiles

The gate set is closed:

| Gate | Required scale |
| --- | --- |
| `e2e` | `RowsRun` of at least 100,000 |
| `daily_inflow` | `RowsDay` above 1,000,000 |
| `monthly_close` | `RowsMonth` of at least 30,000,000 |
| `interrupt_resume` | `RowsRun` of at least 100,000 |
| `catch_up` | `RowsDay` above 1,000,000 |

`Profile{ID, Gate, Axes, Dataset, Expected}`:
- **Axes.** `Axes` holds these non-negative integers, each at most 10¹²:
  - `RowsRun`, `RowsDay`, `RowsMonth`;
  - `Customers`, `Accounts`, `SharedPools`;
  - `SkewPermille`, at most 1,000;
  - `FanOut`, `ConcurrentCloses`;
  - `PeakWindowMinutes` (concentrated arrival);
  - `RetentionMonths` (accumulated retention);
  - `FailureInjections`;
  - `BacklogRows`.
- **P0 floor.** A profile below its gate's required scale is refused with `below_p0`, so a gate can never be met by a smaller test. The floors are:
  - `interrupt_resume` also needs `FailureInjections` of at least 1;
  - `catch_up` also needs `BacklogRows` of at least 1,000,000.

  Minimums for the other axes are owner decisions and a residual.
- **Data and answer.**
  - `Dataset` is the hex digest of the input data.
  - `Expected` is the hex digest of the independently expected result.

  Both are fixed when the profile is registered under dual control, so correctness never rests on what a harness reports. A new dataset or answer is a new profile.
- **Immutability.** Profiles are immutable, and a new ID is a new profile.

## Hardware

`Hardware{ID, CPUs, MemoryMiB, DiskGiB, Storage, Spec}`:
- `CPUs`, `MemoryMiB` and `DiskGiB` are from 1 to 10⁹.
- `Storage` is `ssd`, `hdd` or `network`.
- `Spec` is the hex SHA-256 of the full hardware description.

Hardware records are immutable.

## Budgets

`Budget{ID, Profile, Hardware, Limits}`:
- **Limits.** `Limits` lists `Limit{Metric, Value}` strictly ascending by metric, with values from 0 to 10¹⁸. The metric set is closed:

  | Metric | Unit | A run meets it when |
  | --- | --- | --- |
  | `max_rss_bytes` | bytes | value ≤ limit |
  | `db_io_bytes` | bytes | value ≤ limit |
  | `wal_bytes` | bytes | value ≤ limit |
  | `lock_wait_ms` | milliseconds | value ≤ limit |
  | `p95_ms` | milliseconds | value ≤ limit |
  | `p99_ms` | milliseconds | value ≤ limit |
  | `throughput_rows_s` | rows per second | value ≥ limit |
  | `completion_ms` | milliseconds | value ≤ limit |
  | `resume_ms` | milliseconds | value ≤ limit |

- **Required metrics.** Every budget needs `max_rss_bytes`, `p95_ms`, `p99_ms`, `throughput_rows_s` and `completion_ms`. An `interrupt_resume` budget also needs `resume_ms`. A missing one is refused with `missing_metric`.
- **Rate floor.** A `daily_inflow` or `catch_up` budget needs a `throughput_rows_s` limit of at least the profile's `RowsDay` divided by 86,400, rounded up. Otherwise it is refused with `below_p0`.
- **States.** A budget registered through `budget` is `proposed`. It becomes the profile's active budget, replacing any earlier one; earlier budgets stay readable.
- **Approval.** An `approval` change makes it `approved`. It needs `Distribution`, the hex SHA-256 of the real distribution evidence, and is only possible for the profile's active budget (`not_active` otherwise).
- **Separation.** The approval's approver must differ from the budget's proposer and approver, so at least three people stand behind an approved SLA. Otherwise it is refused with `same_people`.
- **Immutability.** An approved budget never changes, and approving it again is `ErrConflict`. A new SLA is a new budget, which starts as proposed again.

## Runs

`Record(Run{Profile, Hardware, Build, Dataset, Output, Rows, DurationS, Mismatches, Interrupted, Metrics}, actor)`:
- **Actor.** It is accepted only from a system actor bound to a registered, unrevoked harness (`not_harness`); an assistant is refused with `ai_not_allowed`.
- **No self-verification.** `Record` is refused with `self_verification` when the person who proposed or approved the harness is also any of these:
  - the person who proposed or approved the profile;
  - the active budget's proposer or approver;
  - the person who proposed or approved that budget's approval.

  The verdict rechecks this against the budget active when it is read.
- **Fields.**
  - `Build` is `MAJOR.MINOR.PATCH`.
  - `Dataset` is the input digest and `Output` the produced result digest.
  - `Rows` is the number of rows processed.
  - `DurationS` is the measured wall-clock duration in seconds.
  - `Mismatches` is the count of rows that differ, 0 or more.
  - `Interrupted` says whether the run was interrupted and resumed.
  - `Metrics` lists `Measure{Metric, Value}` strictly ascending, with each metric from the closed set at most once.
- **Numbering.** Runs are numbered from 1 and record the harness and the time.
- **Verdict.** The verdict is computed when read, in this order:

  | Verdict | When |
  | --- | --- |
  | `other_data` | the run's `Dataset` differs from the profile's |
  | `incomplete` | the run fell short of the profile's own scale, or of the measures its gate requires (see below) |
  | `incorrect` | `Output` differs from the profile's `Expected`, or `Mismatches` is above 0 |
  | `revoked` | its harness has since been revoked |
  | `self_verified` | the harness's people now overlap the active budget's people |
  | `missing_metric` | any of the eight metrics other than `resume_ms` is missing, or `resume_ms` is missing for `interrupt_resume`. Every run must carry them, whatever the budget limits |
  | `unbudgeted` | the profile has no active budget |
  | `other_hardware` | the run's hardware is not the active budget's hardware |
  | `over_budget` | any limit is not met. A metric limited by the budget but missing from the run is `missing_metric` |
  | `predates_budget` | every limit is met, but the run was recorded before the active budget was approved |
  | `provisional` | every limit is met, but the active budget is only proposed |
  | `passed` | every limit is met under an approved active budget, by a run recorded after the approval |

  "After" is decided by a register-wide event sequence that every applied change and every run takes, not by timestamps, so a run and an approval in the same instant are still ordered.

  A run is `incomplete` when any of these holds:
  - `Rows` is below the profile's own axis: `RowsRun` for `e2e` and `interrupt_resume`, `RowsDay` for `daily_inflow`, `RowsMonth` for `monthly_close`, and `BacklogRows` for `catch_up`;
  - a `daily_inflow` run has `DurationS` under 86,400;
  - a `daily_inflow` or `catch_up` run processed fewer rows per day than the profile's `RowsDay`, that is `Rows × 86,400 < RowsDay × DurationS`, or has a `DurationS` of 0;
  - an `interrupt_resume` run was not `Interrupted`.

  Because the verdict is computed when read, a run counts against the budget that is active now, never a stale one. A budget fitted to earlier measurements cannot turn those runs into passes (`predates_budget`). Replacing a budget sends earlier runs back to `unbudgeted`, `provisional` or `predates_budget`.

## Gate status and readiness

Statuses are per build, so the gates cannot pass on a mix of builds or on a build older than the release candidate.
- **Profile status.** For a build, a profile's status is the verdict of its latest run of that build, or `not_run` when it has none. The view also shows the number of that build's runs not passed since the active budget's approval, so retries until a pass stay visible.
- **Gate status.** For a build, over the gate's profiles that are not retired, in this order:
  - `failing` when any profile's verdict is `other_data`, `incomplete`, `incorrect`, `revoked`, `self_verified`, `missing_metric`, `other_hardware` or `over_budget`;
  - `not_run` when the gate has no profile that is not retired, or any such profile is `not_run`;
  - `provisional` when any is `unbudgeted`, `predates_budget` or `provisional`;
  - `passed` only when every profile of the gate is `passed`.
- **`Readiness(build)`** returns every gate with its status and its per-profile statuses. It is ready only when all five P0 gates are `passed` for that build. `not_run`, `provisional`, `predates_budget` and `unbudgeted` are never counted as passing.

## Reads

Every read returns a copy:
- `Profile`, `Hardware`, `Budget`, `ActiveBudget(profile)`;
- `Run(number)`;
- `Runs(profile, after, limit)`, oldest first, with a limit of at most 1,000;
- `Gate(gate, build)`, `Readiness(build)` and `Proposal`.

## Bounds

| Bound | Limit |
| --- | --- |
| Profiles | 1,000 |
| Hardware records | 100 |
| Budgets | 10,000 |
| Harnesses, revoked ones included | 100 |
| Runs kept per profile | 1,000. The oldest is evicted, and reading it is `ErrNotFound` |
| Runs in the register | 100,000 retained runs; evicted runs do not count. A run that would grow the total beyond it is `ErrLimit`; a run on a profile already at its own cap evicts and is accepted. Retired profiles keep their runs as history, and those runs count |
| Identifiers | 1–128 bytes of `[A-Za-z0-9._:-]` |

## Residuals

- Running the measurements: the harness, data generation at scale (a W-38 residual) and the system-level P0 gates themselves, which stay NOT_RUN.
- p50 and warm or cold conditions are recorded only through the hardware `Spec` and the profile ID. Separate fields for them are a residual.
- A profile with a wrong `Expected` digest fails its gate until it is retired and replaced under dual control.
- Per-gate minimums for the other axes; the duration and lag rules for `catch_up`; how failures are injected.
- A harness can retry until a run passes. The failure count is shown, but retries are not limited.
- Persistence and screens.
- Owner decisions not confirmed by the owner:
  - every SLA number;
  - what a source row means;
  - the inflow ceiling and peak window;
  - RPO and RTO;
  - the hardware;
  - the closed metric set.

# Verification plan

Literal tests:
- **Forms and bounds.** Every form rule and bound, at the limit and one past it.
- **Proposals.** Proposal rules, actor kinds and recheck at approval.
- **Profiles.** `below_p0` for each gate, at its floor and one below.
- **Budgets.** `missing_metric` for each gate. Approval by a third person, `same_people`, `not_active`, and replacement.
- **Runs.**
  - Every verdict, in precedence order, including `other_data`, `self_verified` and `predates_budget`.
  - Every metric required on every run.
  - The `daily_inflow` duration.
  - Per-build statuses and the failure count.
  - Direction for throughput.
  - Approval or replacement of a budget changing earlier runs' verdicts.
  - Harness binding and revocation.
  - Eviction and the total cap.
- **Gates.** Gate status and readiness, including that `not_run` and `provisional` are never passing.

Property test: random profiles, budgets, approvals and runs. An independent model of verdicts, gate statuses and readiness must match after every step.

Checks: `-race -count=10`, the 386 build, gofmt, vet, `go test -race ./...`, 100% statement coverage and mutation testing.

# Review history

- Round 1 (independent reviewer, read-only): FAIL. Findings and what changed:
  - H1. The dataset and expected digests are fixed in the dual-controlled profile, and runs get `other_data` and `incorrect`.
  - H2. `self_verification` at `Record`, and `self_verified` on read.
  - M1. `predates_budget`.
  - M2.
    - Every metric is required on every run.
    - New axes, with floors for failure injection and backlog.
    - A rate floor, and a one-day duration for `daily_inflow`.
    - Runs are compared with the profile's own axis.
  - M3. Statuses and readiness are per build.
  - L1. `failing` takes precedence.
  - L2. Re-approval is a conflict, and the total cap counts retained runs.
  - L3. A failure count since approval, with retries as a residual.
- Round 2 (same reviewer): FAIL. Findings and what changed:
  - N1. The per-day rate is checked against the profile's `RowsDay` for `daily_inflow` and `catch_up`, and the budget rate floor is derived from it.
  - N2. The `Record` signature now matches the field list.
  - N3. The approval's proposer is in the separation set.
  - N4. A register-wide event sequence decides "after".
  - N5. Dual-controlled profile retirement.
- Round 3 (same reviewer): PASS. Low items folded in:
  - L-a. Retired profiles' statuses stay visible in the gate view.
  - L-b. `Record` refuses a retired profile with `retired`.

# Change review resolution

The change review (independent reviewer, read-only) returned FAIL. Resolved:
- H1. The per-day rate is compared in 128 bits (`bits.Mul64`), so large axes and durations cannot overflow into a false pass. Literal tests cover it at the bounds.
- M1. The total cap refuses only runs that would grow the total.
- M2. Tests cover a budget approver overlapping the harness, at `Record` and on read.
- M3. Tests cover each P0 floor condition separately.
- M4. The property model now covers:
  - all five gates;
  - wrong data digests, dropped metrics and interruption;
  - a second harness whose people overlap a budget approval, both refused at `Record` and `self_verified` on read;
  - `Readiness` over all gates.

  Its rate check uses `math/big`, independently of the code.
- L1. A metric limited by the budget but missing from the run is `missing_metric`.
- L2. `ActiveBudget` reads under one lock.
- L3. `Record` checks the harness binding before the profile lookup, as in `modules/simulation`, so unbound callers cannot probe profile IDs. This deviation from the check order is deliberate.
- L4. Gate-level tests cover `other_data`, `missing_metric` and `self_verified` as failing.

The re-review (same reviewer, read-only) returned PASS with no regression.
