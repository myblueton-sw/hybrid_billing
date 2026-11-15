---
type: ImplementationPlan
title: Simulation core with dataset and scenario manifests, dual-controlled independent expectations, reproducible runs, a commitment event matrix and integration proof kept apart from demo passes
description: First increment of work item W-38. A register records dataset manifests with source, hash, licence notice and origin, and scenario manifests that fix seed, virtual clock, versions and the event list. Expected values come from an independent calculation, are approved by a second human and can never be engine output. Runs are compared check by check, and a different output for the same inputs is flagged as non-reproducible. A DB-09 matrix links every commitment event to a scenario, an unsupported record or an open decision. Integration proof is recorded separately and no simulation run can set it.
status: final
---

# Scope and ownership

[HB-248](https://linear.app/hybrid-billing/issue/HB-248) is the first increment of work item W-38 (requirement HB-38). It covers:
- the HB-38-I manifests: source, hash, licence, coverage, version, seed, clock, events and expected values;
- source cases SIM01 (source and format), SIM02 (repeatable runs), SIM03 (independent reconciliation) and SIM10 (demo kept apart from real integration);
- the DB-09 matrix and its acceptance case, AUD-DB-09.

It does not cover:
- QA-10 demo isolation: credentials, egress, environment separation and reset (SIM08). This is increment 2.
- Generators at scale (W-39).
- Replay, mock ERP and mailbox drivers.
- Persistence and screens.

Base: main `a67549f`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the design and the change, because the register decides which billing amounts count as verified. Machine: mac_mini.

Allowed files: `modules/simulation/*` and `docs/features/HB-248/*`.

# Observed baseline

- **HB-38.** Fixed scenarios run on the same core as production. A public partial month must never become the answer for a real invoice. Acceptance requires:
  - identical seed, clock and version giving identical results;
  - independent expected values;
  - a DEMO or synthetic mark;
  - blocked external sends;
  - SIM01–SIM10.
- **Source plan, chapter 106.** A `ScenarioManifest` fixes these fields:
  - `scenario_id` and `fixture_version`;
  - the seed;
  - the virtual start time and time zone;
  - the organisation and account relation version;
  - the policy and FX versions;
  - the event list;
  - the expected-value version.

  Generated data records `synthetic` and the generator version. Expected values come from an independently prepared calculation sheet or separate verification logic, approved before use. Billing-engine output is never stored as the answer.
- **SIM rows.**
  - SIM01: public source hash and notice kept; converted, synthetic and version identified.
  - SIM02: the same seed, clock and rules give the same ordered statement and amounts.
  - SIM03: expected totals per currency, document and period match.
  - SIM10: a demo pass and real CSP, ERP or product connection proof are recorded separately.
- **DB-09.** A test matrix per commitment profile covers these events:
  - full, partial and no upfront payment;
  - utilisation of 0 and 100%, and overage;
  - concurrent commitments;
  - a mid-month start, expiry or move;
  - a change of sharing scope;
  - a late credit;
  - a refund or exchange;
  - a correction after cancellation;
  - negative savings;
  - an FX change;
  - a partial CSV error.

  An event that the source does not support is recorded with evidence and is not replaced by a pass. An undecided amount or policy does not remove the need for a test.
- **No existing code.** No module records scenarios, expectations or runs. The Python fixtures under `tests/` are test inputs, not a register.

# Decision

A new package `modules/simulation` follows the sibling conventions:
- one mutex and an injected clock that never goes backward;
- `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (with `Refusal{Reason}`) and `ErrLimit`;
- check order: form, lookup, repeat, rule, room;
- assistants refused as `ai_not_allowed` and system actors as `not_human` where a human is required;
- numbered proposals approved by a second human with a full recheck, withdrawal by the proposer, register-wide pending caps (total, and per proposer at a quarter of the total, as in the siblings), and bounded history.

It imports only the standard library. It is constructed with `NewRegistry(clock)`.

The register holds test assets only, never customer data. Datasets are public, converted from public, or synthetic. The register does not run the engine: a harness runs it and records the result through `Record`.

## Proposals

| Kind | Payload | Effect |
| --- | --- | --- |
| `dataset` | `Dataset` | An immutable dataset manifest |
| `scenario` | `Scenario` | An immutable scenario manifest |
| `expectation` | `Expectation` | A new expectation version for a scenario. Its proposer is the author |
| `profile` | `Profile{ID}` | A commitment profile. It creates one matrix cell per DB-09 event, all `open` |
| `cell` | `Cell` | Sets how one matrix cell is covered |
| `harness` | `Harness{ID, Identity}` | Binds a system identity that may record runs |
| `target` | `Target{ID, Kind, Scenarios}` | An integration target with the scenarios that make up its demo |
| `proof` | `Proof{Target, Environment, Evidence}` | Records real integration proof for a target |

Scopes:
- `dataset`, `scenario`, `profile`, `target` and `harness` use the register scope.
- `expectation` uses the scenario's scope.
- `cell` uses the profile's scope.
- `proof` uses the target's scope.

Each kind is rechecked in full at approval.

## Harnesses

- **Registration.** A `harness` proposal, `Harness{ID, Identity}`, binds a system actor identity to a harness. An identity belongs to at most one harness, and a repeat is refused with `identity_taken`.
- **Who may record.** Only a system actor whose ID is a registered harness identity can call `Record`:
  - a human is refused with `not_harness`;
  - an assistant is refused with `ai_not_allowed`;
  - an unbound system identity is refused with `not_harness`.
- **No self-verification.** A harness records who proposed and who approved it. `Record` is refused with `self_verification` when either of those humans is the author or the approver of the scenario's active expectation. The people who set the answer therefore cannot have set up the harness that checks it.
- **Revocation.** `RevokeHarness(id, actor)` lets one human revoke a harness at once. A revoked harness can never record again. Its ID and identity stay recorded, so reusing either is refused with `ErrConflict` or `identity_taken`.

## Datasets (SIM01)

`Dataset{ID, Version, Origin, Digest, Source, License, Notice, Parent, Tool, ToolVersion, Seed, Coverage}`:
- **`Version`** is `MAJOR.MINOR.PATCH`.
- **`Digest`** is the SHA-256 of the dataset bytes, as 64 lowercase hex characters.
- **`Coverage`** lists 1–64 coverage tags (identifiers), strictly ascending.
- **`Origin`** is `public`, `converted` or `synthetic`:
  - **`public`** needs a `Source`, a `License` (an identifier) and a `Notice`.
    - `Source` is an `https://` URL of at most 200 bytes, made of printable ASCII, with no user information (`@`), query (`?`) or fragment (`#`).
    - `Notice` is the hex SHA-256 of the retained notice text.

    `Parent`, `Tool`, `ToolVersion` and `Seed` must be empty.
  - **`converted`** needs these, with `Seed` and `Source` empty:
    - a registered `Parent` whose origin is `public` or `converted`;
    - a converter `Tool` and `ToolVersion`;
    - the `License` and `Notice` copied from the parent. A mismatch is refused with `notice_changed`, and approval rechecks this.
  - **`synthetic`** needs a generator `Tool`, a `ToolVersion` and a `Seed`, a canonical decimal `uint64`. `Source`, `License`, `Notice` and `Parent` are empty.
- **Notice text.** The notice text and the dataset bytes themselves are kept in the evidence store by their digests. The register holds only the digests, and their transfer is a residual.
- **Immutability.** A dataset can never be changed or removed. A new version is a new ID. A repeated ID is `ErrConflict`, including one that differs only in content.
- **Reads.** `Dataset(id)` returns the manifest. `Lineage(id)` returns the chain of parents back to the public original.

## Scenarios

`Scenario{ID, FixtureVersion, Purpose, Topology, Deployment, Seed, Start, Zone, Relations, Policy, FX, Datasets, Events, EventCount, Profile, Covers}`:
- **`Purpose`** is `format` or `billing`. A `format` scenario checks format and queries. A `billing` scenario checks amounts.
- **`Topology`** is `single`, `group` or `msp`. **`Deployment`** is `cloud`, `onprem` or `mixed`.
- **`Seed`** is a canonical decimal `uint64`.
- **`Start`** is the virtual start time: non-zero, stored in UTC, with a year from 2000 to 2199.
- **`Zone`** is an IANA zone name of 1–64 bytes made of letters, digits, `/`, `_`, `+` and `-`. The register does not load time zone data, so the name is recorded, not resolved.
- **`Relations`, `Policy` and `FX`** are version identifiers.
- **`Datasets`** lists 1–64 registered dataset IDs, strictly ascending.
- **`Events`** is the hex SHA-256 of the event list. **`EventCount`** is from 0 to 10,000,000.
- **`Profile` and `Covers`.**
  - `Profile` is an optional registered commitment profile.
  - `Covers` lists the DB-09 events the scenario exercises: 0–18 names from the closed set, strictly ascending.
  - `Covers` must be empty when `Profile` is empty.
  - Only `billing` scenarios may name a profile.
- **Billing data.** A `billing` scenario may reference only `synthetic` datasets. A public or converted dataset is refused with `public_answer`, so a public partial month never becomes an invoice answer.
- **Immutability.** Scenarios are immutable, like datasets.

## Expectations (SIM03 and the chapter 106 rule)

`Expectation{Scenario, Version, Method, Evidence, Checks}`:
- **`Version`** starts at 1 and is exactly one more than the scenario's latest approved version. A gap or a repeat is `ErrConflict`.
- **Approved before use.** An expectation can be proposed or approved only while the scenario has no recorded run. Otherwise it is refused with `runs_recorded`.
  - So the expected values are fixed before the engine ever runs the scenario. The active version at the first run is frozen. `Current` reports it as `Expected`, and each run records the version it used.
  - This is how the chapter 106 rule, that the expected version is part of the manifest, is met.
  - **Corrections.** A wrong expected value for inputs that have already run cannot be corrected in this increment. Its scenario keeps failing, which is the safe direction. A correction needs a new fixture with different engine inputs, for example a regenerated event list. This limit is accepted and recorded as a residual.
  - **Fingerprint.** Each scenario has a fingerprint: the digest of the fields that are engine inputs. These are `Topology`, `Deployment`, `Seed`, `Start`, `Zone`, `Relations`, `Policy`, `FX`, `Datasets`, `Events` and `EventCount`. `Purpose`, `Profile` and `Covers` do not change what the engine produces, so they are left out. An expectation is also refused with `runs_recorded` when any scenario with the same fingerprint has recorded runs. A copy of a scenario that has already run therefore cannot be given answers taken from those runs.
  - **What this proves.** Within the register, no run output exists for these inputs before the expectation is fixed. The register cannot see engine runs made outside it, or a manifest changed only to escape the fingerprint. Independence from such runs rests on the method and the approver, and this is a residual.
- **`Method`** is `manual_sheet` or `independent_logic`.
- **`Evidence`** is the hex SHA-256 of the calculation sheet or the logic's output.
- **`Checks`** lists 1–256 `Check{Name, Kind, Value, Currency, Document, Period}`, strictly ascending by name:

  | Kind | `Value` | `Currency` |
  | --- | --- | --- |
  | `amount` | a canonical decimal: `-?(0|[1-9][0-9]*)(\.[0-9]*[1-9])?`, at most 40 bytes, never `-0` | three uppercase ASCII letters |
  | `count` | a canonical decimal integer of 0 or more, at most 19 digits | empty |
  | `digest` | 64 lowercase hex characters | empty |
  | `status` | an identifier whose first character is a lowercase letter | empty |

  - Each value has exactly one written form, so exact string equality is numeric equality.
  - `Document` is an optional document identifier. `Period` is an optional `YYYY-MM`, with a month from 01 to 12.
- **Purpose.** `amount` checks are allowed only in `billing` scenarios.
- **Totals.** A `billing` expectation needs at least one `amount` check. For each currency used by its `amount` checks, it also needs one with empty `Document` and `Period`, which is the currency total required by SIM03.
- **Author and approver.** The author is the proposer and must be a human. The approver is a second human. Neither can be a system actor.
- **Reads.** The latest approved version is active, and every version stays readable.

## Runs (SIM02 and SIM03)

`Record(Run{Scenario, Engine, Rules, Output, Results}, actor)`:
- **Preconditions.**
  - The actor is a registered harness (see Harnesses).
  - The scenario must have an active expectation. Otherwise `Record` is refused with `no_expectation`, so no unchecked run is ever kept.
- **Fields.**
  - `Engine` is the engine version, `MAJOR.MINOR.PATCH`.
  - `Rules` is the hex digest of the rule set.
  - `Output` is the hex SHA-256 of the canonical ordered output.
  - `Results` lists 0–256 `Result{Name, Value, Currency}`, strictly ascending by name, in the same written forms as checks.
- **Numbering.** Runs are numbered from 1 and record the harness, the time and the expectation version.
- **Results digest.** The register computes a digest of `Results` with the canonical encoding.
- **Reproducibility key.** The key is `(Scenario, Engine, Rules)`; the scenario fixes the seed, clock, versions and events.
  - The first run under a key records its `Output` and results digest.
  - A later run that differs in either marks the key `non_reproducible` permanently, recording the number of the run that caused it.
  - A key with at least two runs and no mark is `reproduced`. A key with one run is `single`.
  - The key state is read live, so marking also affects earlier runs.
- **Comparison.** Each check gives `match`, `mismatch` (with the value received) or `missing`. Results with no matching check are counted as `Unexpected`. An `amount` matches only when both the value and the currency are equal.
- **Kept per run.** A run keeps one outcome code per check, names taken from the expectation, and the received values of its first 8 mismatches. It does not keep the raw results. Further mismatches show their outcome without the value.
- **Copied answer.** If `Output` equals the expectation's `Evidence`, the run records `copied` and fails.
- **Verdict.** The verdict is computed when read, in this order:

  | Verdict | When |
  | --- | --- |
  | `revoked` | the run's harness has since been revoked |
  | `non_reproducible` | the run's key is marked |
  | `fail` | any check is not `match`, there are unexpected results, or the run is `copied` |
  | `unconfirmed` | the run would pass but its key is `single` |
  | `pass` | the run would pass and its key is `reproduced` |

- **Current.** The current verdict is taken at the highest engine version recorded for the scenario.
  - At that version, the latest run of each rule set is read, and the worst verdict is reported, in the order `revoked`, `non_reproducible`, `fail`, `unconfirmed`, `pass`. Engine versions are compared numerically, so `10.0.0` is higher than `9.0.0`. A passing run on an older rule set or an older engine therefore never masks a failure.
  - `Current(scenario)` returns the verdict, plus the `Engine`, `Rules` and run number of the run that decided it, or `none` when there is no run. It also reports the scenario's `Expected` version.
  - A passing run on an older engine never replaces a newer engine's result.

## Commitment event matrix (DB-09)

The event set is closed:
- `full_upfront`, `partial_upfront`, `no_upfront`;
- `utilization_zero`, `utilization_full`, `overage`;
- `concurrent_commitments`;
- `mid_month_start`, `mid_month_expiry`, `mid_month_move`;
- `share_scope_change`;
- `late_credit`;
- `refund`, `exchange`;
- `cancel_then_correct`;
- `negative_savings`;
- `fx_change`;
- `partial_csv_error`.

Each profile has one cell per event. `Cell{Profile, Event, Coverage, Scenario, Checks, Dataset, Evidence, Decision}`, where `Coverage` is one of these:
- **`tested`** needs all of the following:
  - a registered `billing` scenario with an active expectation, whose `Profile` is the cell's profile and whose `Covers` includes the event;
  - 1–32 check names, strictly ascending, all present in that active expectation;
  - for every event except `partial_csv_error`, at least one `amount` check among them.

  `Dataset`, `Evidence` and `Decision` are empty.
- **`unsupported`** needs a registered `Dataset` of origin `public` or `converted` (otherwise `wrong_origin`), the source whose limitation it is, and `Evidence`, the hex digest of the source's statement or the observed limitation. The other fields are empty.
- **`undecided`** needs `Decision`, an identifier of the open owner decision. A `Scenario` link and `Checks` are optional; if given, they are validated as for `tested`. `Dataset` and `Evidence` are empty.

A cell can be set again by a new proposal. Each cell keeps a revision count and a history of its last 16 changes. Each history entry records the coverage replaced and the status it had at that moment. Each cell also keeps a permanent count of the changes that replaced a coverage whose status was `failed`. So a failure that is covered over by a change of coverage is never lost, even after the history rolls over. A failure later cleared by a passing run, with no change of coverage, is not counted.

The status of each cell is computed when read, in this order:

| Status | When |
| --- | --- |
| `open` | no coverage has been set |
| `unsupported` | coverage is `unsupported` |
| `undecided` | coverage is `undecided`. A test is still required, whatever its linked scenario shows |
| `failed` | coverage is `tested` and the current verdict is `revoked`, `fail` or `non_reproducible` |
| `passed` | coverage is `tested`, the current verdict is `pass`, and every linked check is in the expectation the current run used |
| `pending` | coverage is `tested` in any other case: `none` (no run), `unconfirmed`, or a linked check missing |

`Matrix(profile)` returns the 18 cells in event order, each with its status, coverage, revision count, history, failure count, and current engine and rules.

`Summary()` returns the count of cells in each status across all profiles. There is no single "complete" flag, and `unsupported` and `undecided` are never counted as passed.

## Integration proof (SIM10)

- **Targets.** A `Target{ID, Kind, Scenarios}` has `Kind` `csp`, `erp` or `product`, and 1–64 registered scenario IDs, strictly ascending.
- **Proof.** A `Proof{Target, Environment, Evidence}` names an `Environment` identifier and `Evidence`, the hex digest of the connection evidence.
  - A proof is refused with `run_output` when `Evidence` equals any recorded run's output digest, or any expectation's `Evidence`, anywhere in the register.
  - Recording proof again replaces the earlier record and keeps a count.
  - `Record` and every other run path never touch proof.
- **Environment class.** Environments are not registered or classified until increment 2 (QA-10). So recorded proof reads as `proven_unclassified`, never `proven`, and nothing in this increment can show a target as proven in production.
- **Read.** `Status(target)` returns two separate fields:
  - `Demo`, based on the listed scenarios' current verdicts:
    - `passed` when every one is `pass`;
    - `failed` when any is `revoked`, `fail` or `non_reproducible`;
    - `pending` otherwise.
  - `Integration`: `not_proven` or `proven_unclassified`, with the environment, the evidence, the approver, the time and the count.

  A demo pass never sets `Integration`, and proof never sets `Demo`.

## Canonical encoding

Every value is a length-prefixed field: an 8-byte big-endian length, then the bytes. Integers are canonical decimal, and times are UTC Unix nanoseconds.
- **Fingerprint.** `hybrid-billing/simulation-scenario/v1`, `Topology`, `Deployment`, `Seed`, `Zone`, `Relations`, `Policy`, `FX`, `Events`, `Start`, `EventCount`, then the number of datasets and each dataset's `Digest`, sorted bytewise. Using digests means that re-registering the same bytes under a new ID does not escape the rule.
- **Results digest.** `hybrid-billing/simulation-results/v1`, the number of results, then each result's `Name`, `Value` and `Currency`.

Both are SHA-256 in lowercase hex, and known-answer tests are cross-checked against an independent Python encoder.

## Reads

Every read returns a copy:
- `Dataset`, `Lineage`, `Scenario`;
- `Expectation(scenario, version)`;
- `Run(number)`;
- `Runs(scenario, after, limit)`, oldest first, with a limit of at most 1,000;
- `Current(scenario)`;
- `Matrix`, `Summary`, `Status`;
- `Proposal(number)`.

Each run view carries:
- its verdict, its outcomes, `Unexpected` and `Copied`;
- the expectation version;
- the key state, and the run that marked the key.

## Bounds

| Bound | Limit |
| --- | --- |
| Datasets | 10,000 |
| Scenarios | 10,000 |
| Expectation versions per scenario | 16 |
| Runs per scenario | 1,000 |
| Runs in the register | 200,000 |
| Reproducibility keys | 200,000 |
| Harnesses, revoked ones included | 100 |
| Profiles | 1,000 |
| Targets | 1,000 |
| Pending proposals, register-wide | 1,000 |
| Pending proposals per proposer, register-wide | 250 |
| Applied proposals kept | 1,000. Withdrawn proposals are deleted at once, as in the siblings |
| Identifiers | 1–128 bytes of `[A-Za-z0-9._:-]`, as the sibling rule |
| Source URL | 200 bytes |

Memory estimates:
- **Runs.** A run keeps at most 256 outcomes. The names are shared with the expectation, and a received value is kept only for a mismatch. Each run keeps 256 one-byte outcome codes and at most 8 mismatched values of about 170 bytes each, about 1.7 KB. So 200,000 runs need roughly 350 MB at worst.
- **Pending proposals.** These are about 1,000 × 100 KB, roughly 100 MB.

Runs are never pruned in this increment. When a cap is reached, `Record` returns `ErrLimit`.

Every recorded output digest and every expectation evidence digest are kept in register-wide sets, so the `run_output` check is exact.

## Residuals

- Demo isolation (QA-10, SIM08): increment 2. This includes registering and classifying environments, after which proof can read as `proven`.
- Transferring notice texts, dataset bytes and calculation sheets into `modules/evidence`.
- `origin: synthetic` is declared, not proven. The register cannot tell whether the bytes behind a digest were generated.
- Generators at scale and stream generation (W-39).
- Replay, mock ERP, mailbox, on-premises history and permission-failure drivers.
- SIM04–SIM07 and SIM09, which need the harness to run real scenarios.
- Wiring a harness to the engine.
- Voiding runs. A run recorded with a wrong engine version stays the highest version until a higher one is recorded. Likewise, a failing latest run of an old rule set keeps `Current` at `fail` for that engine version, even after a corrected rule set passes. Both cases fail safe.
- Correcting an expectation for inputs that have already run (see Corrections).
- Engine runs made outside the register (see the fingerprint rule).
- Runs of a harness revoked later still count toward reproducibility keys, which were set when they were recorded. A fake reference output can therefore mark honest runs `non_reproducible`, which fails safe. A fake second run could also lift an honest single run to `reproduced`. Rebuilding keys without revoked runs is a residual.
- Persistence.
- Screens.
- Owner decisions not confirmed by the owner:
  - the public sample adoption commit;
  - who approves synthetic expectations;
  - the performance hardware.

# Verification plan

Literal tests:
- **Form and bounds.** Every form rule and every bound, at the exact limit and one past it.
- **Proposals.** Proposal rules, actor kinds and recheck at approval.
- **Datasets.**
  - The rules for each origin.
  - Lineage, and `notice_changed`.
  - Immutability.
- **Scenarios.**
  - The rules for `billing` and `format`, including `public_answer`.
  - Zone and start bounds.
- **Expectations.**
  - Version sequence.
  - `runs_recorded` at proposal and at approval.
  - The totals rule.
  - Purpose rules.
  - Author and approver kinds.
- **Runs.**
  - `match`, `mismatch` and `missing` outcomes, `Unexpected`, and amounts with a different currency.
  - `copied`.
  - `no_expectation`.
  - `non_reproducible` marking earlier and later runs.
  - `single`, `reproduced` and `unconfirmed`, including the run that caused a mark.
  - A results difference under the same `Output`.
  - `Current` with older engines.
  - Harness binding, `self_verification` and revocation.
  - The fingerprint rule.
  - The worst verdict across rule sets, and `revoked`.
  - The mismatched-value limit.
  - Paging.
- **Matrix.**
  - Every cell status, in precedence order.
  - `Profile` and `Covers` linkage, and the `amount` rule.
  - Revision count, history and failure count.
  - Linked checks missing.
  - Summary counts.
  - Unsupported and undecided never counted as passed.
- **Proof.**
  - Demo and integration statuses kept apart.
  - `run_output` against runs and expectations.
  - `proven_unclassified`.
  - Replacement.
- **Concurrency.** Concurrent `Record` calls under `-race`.

Property test: random sequences of expectations, runs (with repeated and changed outputs) and cell coverage over two scenarios. An independent model of verdicts, keys, current verdicts and cell statuses must match the register's reads after every step.

Checks: `-race -count=10`, the 386 build, gofmt, vet, `go test -race ./...`, 100% statement coverage and mutation testing.

# Review history

- Round 1 (independent reviewer, read-only): FAIL. Findings and what changed:
  - H1. Expectations are fixed before the first run (`runs_recorded`), and the expected version is frozen at that run.
  - H2. Only registered harness identities record, with `self_verification` refused.
  - M1. Scenarios declare `Profile` and `Covers`. `tested` cells need the link and an `amount` check.
  - M2. `Current` follows the highest engine version.
  - M3. A results digest is part of the key. There are `single`, `reproduced` and `unconfirmed` states, and the marking run is recorded.
  - M4. `unsupported` names a dataset. `undecided` keeps an optional link. Cells keep revision history.
  - M5. Lower caps, register-wide pending caps, and runs keep outcomes only.
  - M6. Proof reads as `proven_unclassified` until increment 2.
  - L1. Register-wide digest sets that include expectation evidence.
  - L2. Resolved by H1.
  - L3. `Document`, `Period` and the currency-total rule.
  - L4. Notice text retention stated.
  - L5. Precedence order stated.
  - L6. `status` values start with a lowercase letter.
  - L7. `Source` is restricted to an https URL.
  - L8. Identifier rule wording corrected.
- Round 2 (same reviewer): FAIL. Findings and what changed:
  - N1. The scenario fingerprint rule. The independence claim is limited to runs in the register, and outside runs are a residual.
  - N2. `self_verification` compares the harness's proposer and approver. `RevokeHarness` was added.
  - N3. `Current` reports the worst verdict across rule sets at the highest engine. Voiding is a residual.
  - L1. `unsupported` needs a public or converted dataset. Cell history keeps 16 entries with their status.
  - L2. Runs cap of 20,000, and corrected estimates.
  - L3. The wording now says register-wide caps.
  - L4. Names aligned. `none` maps to `pending`.
- Round 3 (same reviewer): FAIL. Findings and what changed:
  - R1. The fingerprint covers engine inputs only. The correction limit is accepted and recorded.
  - R2. Runs keep outcome codes and 8 mismatched values. Caps are 200,000 runs and 200,000 keys.
  - R3. Runs of a revoked harness read `revoked` and fail closed.
  - R4. The rule-set case is added to the residual, and numeric version order is stated.
  - R5. A permanent failure count per cell. The verification wording is fixed.
- Round 4 (same reviewer): PASS. Low items folded in:
  - S1. Recorded as a residual.
  - S2. Punctuation fixed.
  - S3. What the failure count counts is stated.

# Change review resolution

The change review (independent reviewer, read-only) found no code defect against the plan. It returned FAIL for test gaps M1 and M2, plus lows. Resolved:
- M1. Tests cover a human, and an assistant, using a bound harness ID.
- M2. Tests cover both cross-role `self_verification` cases.
- L1. `Current` reports the `Expected` version.
- L2. Withdrawn proposals are deleted at once, as in the siblings. The bounds table now says so.
- L3. `Record` checks the harness binding before the scenario lookup, so unbound callers cannot probe scenario IDs.
- L4. An expectation whose `Evidence` equals a recorded run output is refused with `run_output`, at proposal and at approval.
- L5. Verdicts are computed without building outcomes. `Matrix` and `Summary` compute each scenario's current verdict once per call.
- L6. The property model now covers:
  - `copied` outputs;
  - `MarkedBy`;
  - `Current.Run`, the run that decided the verdict.
- L7. Tests cover the proof and expectation rechecks at approval, and concurrent `Propose`, `Approve`, `RevokeHarness` and `Matrix`.
- Hypothesis. The fingerprint now hashes dataset digests, and a test shows that a renamed copy of a dataset does not escape it.

The re-review (same reviewer, read-only) returned PASS with one low, N1: digests were hashed in dataset-ID order, so renamed copies in another order escaped. The digests are now sorted, and a test covers it.
