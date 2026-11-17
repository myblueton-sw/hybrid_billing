---
type: ImplementationPlan
title: O03 contract read and lifecycle API over modules/contracts
description: Phase 4d of the pre-launch plan (HB-257). Adds synthetic sample contracts linked to billing profiles, a contract list and detail read, and administrator commands with a reason that suspend, resume, renew or end a contract, withdraw a future event, record final evidence, settle and reopen, all through the modules/contracts registry. Proration and money stay not computed in pre-launch.
status: final
---

# Scope and ownership

[HB-280](https://linear.app/hybrid-billing/issue/HB-280) adds the API behind O03 (contract and service lifecycle, ADM06 in `docs/design-modules/04-work/r2/administration.md`). Base: main `1ad3fef`. The O03 screen follows in a separate ticket.

**Ownership**
- Root (Claude Code) owns the plan, the implementation and the tests.
- An independent read-only reviewer reviews the design and the change. Contract lifecycle drives billable intervals and final settlement, so the review is DBA style: data accuracy, ordering, idempotency, tenant boundary and permission.
- Machine: mac_mini.

**Allowed files**
- `apps/api/server`: new `contracts.go`, `contractseed.go` and their tests; command registration in `server.go`; seed step in `seed.go`; routes in `boundary.go`; limits in `server.go`.
- `docs/features/HB-280/*`.
- No module changes.

# Observed baseline

**`modules/contracts`** (registry, lifecycle, prorate)
- `Amend(ws, id, expected, write, withdrawals, events, cancel)` adds or withdraws lifecycle events in one batch against the expected contract sequence. The first batch creates the contract.
  - Events are `start`, `change` (with a rate revision), `renew`, `suspend`, `resume` and `end`, on distinct days, and must replay validly from `not_started`.
  - A batch that leaves the lifecycle ended (or empty) moves settlement to `pending_final`.
  - While `pending_final`, a batch that reopens the lifecycle needs a cancel reason (`recorded_in_error` or `contract_continues`).
  - Settled contracts refuse amendments (`ErrState`).
- `RecordEvidence` records `secured` evidence (reference, covered-until day, gaps it resolves) or `missing` evidence (reason, owner, follow-up). Secured evidence requires `pending_final`.
- `Settle(asOf)` requires `pending_final`, no unresolved gap, and, for an ended lifecycle, latest evidence that is secured, newer than the last batch and any reopening, covering the end day, with `asOf` on or after the end day.
- `Reopen(reason)` moves `settled` back to `pending_final`.
- Every write takes a write ID. The same write ID with the same content replays; with different content it conflicts.
- Reads: `Sequence`, `Lifecycle(at)`, `StateOn(at, day)`, `SegmentsAt(at, period)`, `Settlement(at)`, `Records`.
- `Prorate` exists but needs prices and a decimal profile, which pre-launch does not configure.

**`apps/api`**
- Organizations and billing profiles are seeded (HB-272). There is no contract data or route.
- Commands are resolved against the server revision. A stale screen gets `revision_conflict` from the command registry before the handler runs.

# Design

## Contract metadata and seed

The server keeps app-level metadata per contract: workspace, ID, display name, billing profile, timezone (`UTC`) and billing cycle (`monthly`). The registry holds only lifecycle and settlement, so the metadata lives in a small server store next to it, the same split as relationship plans.

The sample seed (only with `--seed sample`) creates five contracts in the boot workspace, each through `Amend` with real events:

| ID | Profile | Lifecycle | Settlement |
|---|---|---|---|
| `sample-contract-01` | north | start 2026-01-01 (revision 1), change 2026-07-01 (revision 2) | none |
| `sample-contract-02` | north | start 2026-01-01, suspend 2026-09-01 | none |
| `sample-contract-03` | south | start 2026-03-01, end 2026-10-01 | pending final, one missing-evidence gap |
| `sample-contract-04` | south | start 2026-02-01, end 2026-08-01 | settled from secured evidence |
| `sample-contract-05` | north chargeback | start 2027-01-01 | none (not started) |

Each seeded write records the seed as its actor (`seed`, no login) and reason `sample seed`. Names end in "(sample)". No real customer data. Sample-contract-03 gets missing evidence (`source_unavailable`, owner `sample-admin`, follow-up `sample-followup-03`, write `cw-03-gap`). Sample-contract-04 gets secured evidence (`sample-final-04`, covered until 2026-08-01, write `cw-04-final`) and then settles as of 2026-08-01. Each contract also gets its terms as `organizations.ContractRevision` rows (customer, seller, path, component IDs; two revisions for sample-contract-01). The dates are absolute, so sample-contract-05 stops being "not started" after 2027-01-01; that is accepted for a sample.

Limits: `Contracts 200`, `Writes 200`, `Reserve 20`, `Batch 8` per contract.

## Reads (admin only, `answer.Workspace()` only)

`GET /api/v1/contracts?on=YYYY-MM-DD` (default today UTC):
- One row per contract in the workspace, sorted by ID, capped at 200: ID, name, profile ID and name, buyer, payer, issuer, path, currency, timezone, cycle, state on the day, lifecycle revision on the day, settlement state, sequence, start day, end day if ended, number of open gaps.
- `impact`: `money`, `proration`, `publication`, `erp` and `final_amount_decision` are all `not_computed`.
- `end` is the last held event's day when the held lifecycle ends (the first day the contract is ended; intervals are `[start, end)`), whatever the `on` day.

`GET /api/v1/contracts/{id}?from=&until=`:
- Unknown, malformed or other-workspace IDs return 404 `unsupported_command`, the same body as an unknown route. A command against them gets `permission_denied` from the command registry, as for other unresolved targets.
- `from` and `until` default to the first day of the current UTC month and the first day of the next month. `until` must be after `from`; the span is at most 366 days.
- Body: the same summary row, the contract terms by revision (customer, seller, path and components with kind, provider-payment party and seller-billable flag), the held events (ID, kind, day, rate revision, and a withdraw hint with blocker and whether a cancel reason is needed), segments for the period, settlement (state, reopenings, evidence rows with sequence, write ID (the value `resolves` takes), kind, reference, covered-until, resolves, reason, owner, follow-up, resolved), the last 100 write records (sequence, time, kind, events added, events withdrawn, cancel reason, as-of, reopen reason, resulting state, actor ID and login, reason), and `actions`: for each command, whether it is allowed now, a blocker code if not, and for lifecycle commands the earliest allowed day.

## Commands (installation administration, reason required, target is the contract ID)

| Command | Fields | Module call |
|---|---|---|
| `contract.suspend` | `from` | `Amend` with a `suspend` event |
| `contract.resume` | `from` | `Amend` with a `resume` event |
| `contract.renew` | `from` | `Amend` with a `renew` event |
| `contract.end` | `from` | `Amend` with an `end` event |
| `contract.withdraw` | `event`, `cancel` (optional enum) | `Amend` withdrawing that event |
| `contract.evidence` | `kind`, `reference`, `covered_until`, `resolves`, `reason`, `follow_up` | `RecordEvidence`; owner is the acting user |
| `contract.settle` | `as_of` | `Settle` |
| `contract.reopen` | `reason` (`late_source`, `late_credit`, `correction`) | `Reopen` |

Rules:
- `Resolve` returns not found unless the workspace is the boot workspace and the contract exists there.
- The handler reads the current contract sequence and passes it as the expected sequence. The command registry has already checked the server revision under the domain lock, so a stale screen is refused before this point.
- Event days (`from`) must be today UTC or later and at most 3,660 days ahead. `end from` is the first day the contract is ended. Retroactive lifecycle changes would move finalized periods, which pre-launch cannot check.
- Only `renew`, `suspend`, `resume` and `end` events dated today or later can be withdrawn. `start` and `change` are never withdrawn in pre-launch, because withdrawing a start cancels the contract and creating or cancelling contracts is out of scope.
- Withdraw checks before the module call that the event is held and withdrawable (`invalid_request`, `event`) and that a cancel reason is given exactly when settlement is `pending_final` and the result is no longer ended (`invalid_request`, `cancel`).
- Missing evidence is refused on a `settled` contract (`invalid_stage`), so a settled contract never has an open gap; a late source after settlement is recorded by reopening first.
- Secured evidence: `covered_until` must be on or before today (`invalid_request`, `covered_until`), and the held end day must not be after today (`invalid_stage`). Evidence cannot claim days that have not happened.
- Evidence fields are checked per kind before the module call: secured takes `reference`, `covered_until` and optional `resolves`; missing takes `reason` (the gap-reason enum) and `follow_up`. A field of the other kind is `invalid_request` on that field. `resolves` is a comma-separated list of evidence write IDs (at most 8, no repeats).
- `reference`, `follow_up` and IDs use the module's ID alphabet (at most 128 bytes), not URLs or sentences. The rationale is bounded free text (256 bytes) shown to administrators and held in memory only, the same as relationship plans. Neither is meant for customer data, but nothing can prove a value is not a name (residual).
- `as_of` must be on or before today UTC.
- Event and write IDs are generated server side (`ev-` and `cw-` plus random hex). The actor and reason are recorded only after the module write succeeds.

Error mapping:

| Cause | Code | Field |
|---|---|---|
| bad field shape, past date, evidence field rules, withdraw precheck | `invalid_request` | the field |
| `ErrRejected` from `Amend` (lifecycle order) | `invalid_stage` | none (the command registry passes fields only with `invalid_request`) |
| `ErrState` (settled contract, secured evidence outside `pending_final`, settle or reopen in the wrong state) | `invalid_stage` | |
| `ErrRejected` from `RecordEvidence` (a `resolves` entry is not an open gap) | `invalid_request` | `resolves` |
| `ErrRejected` from `Settle` (open gap, evidence missing, stale or short) | `prerequisite_failed` | |
| `ErrLimit` | `prerequisite_failed` | |
| `ErrConflict` (not reachable under the domain lock) | `revision_conflict` | |
| read before the write fails (sequence, lifecycle, settlement) | `dependency_unavailable` | |
| `ErrInvalid` or any other error from the write | `internal_error` | |

The `actions` hints dry-run the module rules without writing:
- Lifecycle commands: the hint is for the earliest allowed day only (today, or the day after the last held event if later). Because the candidate follows every held event, the module accepts it exactly when the lifecycle's final state allows the kind. Blockers: `settled`, `ended`, `not_active`, `not_suspended`, `horizon`. The module may also accept an event between today and a held future event (for example a suspend before a future end); the screen offers only the earliest-day form.
- Withdraw: each held event is dry-run removed. Blockers: `not_withdrawable`, `settled`, `breaks_lifecycle`; `cancel_required` is reported separately. The command action is `nothing_withdrawable` when no event qualifies.
- Evidence has two hints, `contract.evidence.secured` (`not_pending_final`, `end_not_reached`) and `contract.evidence.missing` (`settled`).
- Settle: every `Settle` condition (`not_pending_final`, `open_gaps`, `evidence_needed` when the latest evidence is not secured, not newer than the last batch and the last reopening, or does not cover the end day, `end_not_reached`).
- Reopen: `settled` (`not_settled`).
- Every command hint, and each per-event withdraw hint, also reports `write_limit` when the contract's write log is full: the reserve applies to suspend, end, missing evidence and reopen, as in the module. The aggregate withdraw action keeps `nothing_withdrawable` when every event is blocked, whatever the reasons.
- A contract whose held lifecycle is empty cannot be reached through this API (start is never withdrawn and contracts are not created here). If one exists, reads fail closed with `dependency_unavailable` for the whole list rather than guessing a state.
A property test checks that whenever a hint says allowed, the command on the earliest day is not refused with `invalid_stage` or `prerequisite_failed`. The command remains the authority.

Fault hooks: `contract-read` and `contract-sequence` fail reads (`dependency_unavailable`); `contract-amend`, `contract-evidence`, `contract-settle` and `contract-reopen` make the write return `ErrInvalid` (`internal_error`).

## Out of scope (owner residuals)

- Proration, prices, money, publication and ERP impact (`not_computed`).
- Creating a contract or a `change` event (rate revisions need the pricing work in phase 6).
- Two-person approval for lifecycle changes (ADM06 separates operator and reviewer; HB-274 already requires a second administrator for relationship changes). Pre-launch uses one administrator with a reason. This needs the owner's acceptance and is recorded as a residual.
- `settled` here means the final evidence is settled in `modules/contracts`. ADM06 also requires an authorized final amount decision (`settlement.approve_final`), which pre-launch does not compute; the API reports `final_amount_decision: not_computed`. Owner residual.
- Free-text rationale and ID-shaped references could still carry a name; administrators are told not to enter customer data. Owner residual.
- Timezones other than UTC; access-removal plans; late-source window policy.
- Persistence across restarts (all registries are in memory in pre-launch).

# Verification plan

- Unit and handler tests for every command, read and error mapping, including the other-workspace and non-admin cases, idempotent replay through the command registry, and `revision_conflict` from a stale revision.
- Specific cases: a future end gives `pending_final` before the end day and refuses secured evidence until then; withdrawing an end while `pending_final` with no, wrong and each valid cancel reason, with open gaps kept; withdrawing a start; settle with `as_of` before the end day; reopen then settle without new evidence; secured evidence outside `pending_final`; the write limit with the reserve for suspend, end, missing evidence and reopen; an event dated today versus yesterday; resolving the seeded gap through the write ID.
- Property test: random command sequences through the API keep every hint sound (allowed implies not refused with `invalid_stage` or `prerequisite_failed`), keep the list state equal to `StateOn`, and never settle with an open gap.
- `go test -race -count=40`, cross builds (linux, windows, darwin), 100% statement coverage of the new files, mutation testing of `contracts.go`.
- Independent change review, then `verification.md`.
