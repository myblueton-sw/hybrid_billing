---
type: ImplementationPlan
title: Release trace with namespaced requirement records, computed states kept apart from customer proof, a support matrix activated only after two parallel-reconciled billing periods and required test categories, and a release gate that refuses to omit common file features
description: First increment of work item W-41. A register records each requirement with its source location, owner, interfaces, test IDs and blocking flag, under a namespace, so that results are never reused by name alone. Test results recorded by bound harnesses make a requirement's state computed per build: designed, implemented, tested or customer proven, kept apart. Support paths (provider, contract, account role, version, issuance path) are activated under dual control only after two consecutive matched parallel-reconciliation periods on real data, passing evidence for every required category and real-path proof. A release gate reads every selected and common requirement and path, and NOT_RUN never passes.
status: final
---

# Scope and ownership

[HB-254](https://linear.app/hybrid-billing/issue/HB-254) is the first increment of work item W-41 (requirement HB-41). It covers:
- **From HB-41-I.**
  - the RequirementTrace;
  - the support matrix;
  - activation after two billing periods of parallel reconciliation.
- **From HB-41-V.** Cutover only when every test passes for the selected scope (amount, isolation, approval, duplicate, recovery, bulk) and the real path is proven.
- **From DB-01.** The common file features are never omitted from a release, and an unverified ERP template is never shown as verified.
- **From the source plan.** Requirement IDs are namespaced, and a pass is never reused under the same name from another document.

It does not cover:
- running tests or reconciliations;
- the cutover itself;
- feeding `modules/perfgate` readiness or `modules/simulation` results automatically. Callers record those as evidence.
- persistence and screens.

Base: main `6de9b94`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the design and the change, because the gate decides what may be cut over to real customers. Machine: mac_mini.

Allowed files: `modules/releasetrace/*` and `docs/features/HB-254/*`.

# Observed baseline

- **HB-41.** Each requirement is linked to its source text, owner, API and screens, tests and evidence, and to whether it blocks a release. Design, implementation, passing tests and real customer proof are different states.
  - A RequirementTrace and a support matrix (provider × contract × account role × version × issuance path) are maintained.
  - Only the selected paths are activated, and only after two billing periods of parallel reconciliation on real data.
  - Cutover needs every amount, isolation, approval, duplicate, recovery and bulk test to pass for the selected scope, plus evidence that the real path works.
  - Each release profile links input volume, hardware, targets and measurement artifacts.
- **DB-01.** The standard settlement XLSX, customer XLSX, result registration and the shared API/file submission ledger are common initial features. Only specific ERP upload templates and real cutover are chosen per product and version proof. A customer who uses only the API still does not skip the common file tests. AUD-DB-01 requires that a release gate blocks the omission of common features.
- **Source plan, chapter 116.** Before implementation, the RequirementTrace registers requirement ID, text location, owner, API or screen, test IDs, evidence, state and release blocking. States are designed, implemented, tests passed and customer proven. Requirement IDs from earlier documents are told apart by namespace, and passing results are never reused by name alone.
- **Siblings.**
  - `modules/perfgate` (HB-251) and `modules/simulation` (HB-248) produce gate and proof results.
  - `modules/sandbox` (HB-249) classifies environments.

# Decision

A new package `modules/releasetrace` follows the sibling conventions:
- one mutex and an injected clock that never goes backward;
- the shared error set and `Refusal{Reason}`;
- check order: form, lookup, repeat, rule, room;
- assistants refused as `ai_not_allowed` and system actors as `not_human` where a human is required;
- numbered proposals approved by a second human with a recheck, and withdrawal by the proposer (withdrawn proposals are deleted at once);
- register-wide pending caps (1,000 in total, 250 per proposer) and 1,000 applied proposals kept.

It imports only the standard library.

## Requirements

A `requirement` proposal registers `Requirement{Namespace, ID, Source, Owner, Interfaces, Tests, Common, Blocking, Design}`:
- **Key.** `Namespace` (for example `planning-v1.16`) and `ID` (for example `HB-26`) together form the key. Two namespaces with the same `ID` are different requirements, and nothing is shared between them.
- **`Source`** is the identifier of the text location, such as a chapter or paragraph reference.
- **`Owner`** is the identifier of the accountable person or role.
- **`Interfaces`** lists 0–64 API or screen identifiers, strictly ascending.
- **`Tests`** lists 1–256 `TestRef{ID, Category}`, strictly ascending by ID. A test ID belongs to the requirement's namespace.
  - `Category` is empty, or one of the HB-41-V categories: `amount`, `isolation`, `approval`, `duplicate`, `recovery`, `bulk`.
- **`Common`** marks a feature that every release must include (DB-01).
- **`Blocking`** marks a requirement that blocks every release while it is not passing.
- **`Design`** is the hex SHA-256 of the design document. A requirement registered with it is `designed`.

Requirements are immutable.

**Retiring.** A `retire{Namespace, ID, Replacement, Reason}` proposal retires a requirement.
- **With a replacement.** The replacement must:
  - be registered after the retired requirement;
  - be unretired, and not the requirement itself;
  - not already replace another requirement, because each requirement replaces at most one;
  - keep at least the same flags. If the retired one is `Common` or `Blocking`, so is the replacement; otherwise the proposal is refused with `weaker_replacement`.

  Since a replacement is always newer and replaces only once, cycles are impossible. Two humans can still declare that one feature is replaced by another common feature registered later, which merges two features into one. That cannot be told apart from a genuine replacement here, and it is a stated residual.
- **Without a replacement.** Allowed only for a requirement that is not `Common`, with a stated reason, so that a `Blocking` requirement for a path that is never pursued can be lifted. The gate lists it under `RetiredBlocking`, so it stays visible.
- Retired requirements leave the common and blocking sets, and their replacements take over.
- A new namespace therefore never escapes a common or blocking requirement: those sets are evaluated across the whole register, not only the namespaces a release selects.

**Implementation.** An `implemented{Namespace, ID, Commit}` proposal records the 40- or 64-character commit that implements the requirement.
- A requirement keeps its last 16 implementations.
- Results count only for builds registered after the latest implementation record (see Builds). A reimplementation therefore makes earlier results NOT_RUN for that requirement.
- Ancestry between the build commit and the implementation commit is not checked here. This is a residual.

## Builds and test results

- **Builds.** A `build{Build, Commit}` proposal registers a product build (`MAJOR.MINOR.PATCH`) with its product commit, under dual control. It also records a register-wide sequence number. Results are accepted only for registered builds.
- **Harnesses.** A `harness` proposal binds a system identity. `RevokeHarness` lets one human revoke it at once.
- **Recording.** `Record(Result{Namespace, Test, Build, Passed, Evidence}, actor)` records a test outcome.
  - **Actor.** Only a bound, unrevoked harness may record.
  - **Unknown tests.** A test that no requirement of the namespace lists is refused with `unknown_test`, so a pass can never be attached by name from another namespace.
  - **No self-verification.** The record is refused with `self_verification` when the harness's proposer or approver is also any of these:
    - the proposer or approver of a requirement that lists the test;
    - the proposer or approver of that requirement's latest `implemented` record;
    - the proposer or approver of the result's `build`.

    The same check is made again on read. The people who define, implement or build something therefore never verify it.
- **Latest result.** For each (namespace, test, build), the register keeps the latest result, plus counts of passes and failures.
  - The latest is chosen from all results first.
  - If it came from a harness that has since been revoked, or fails the self-verification check, the test counts as NOT_RUN. An older pass never comes back.
- **Bound.** At most 1,000,000 keys are kept. Beyond that, a new key is `ErrLimit`; this is a stated residual. Updates to an existing key always succeed.

## Requirement state

A requirement's state for a build is computed when read:

| State | When |
| --- | --- |
| `designed` | there is no implementation record |
| `failing` | any listed test's counted latest result for the build failed |
| `implemented` | there is an implementation, and not every listed test has a counted passing result for the build, with none failing |
| `tested` | every listed test has a counted passing latest result for the build |
| `customer_proven` | `tested`, plus a `proof` for the same build |

**Counted results.** A result counts when it:
- is for a build registered after the latest implementation;
- came from an unrevoked harness;
- passes the self-verification check.

Anything else is NOT_RUN, and NOT_RUN is never a pass.

**Proof.** A `proof{Namespace, ID, Build, Environment, Customer, Period, Evidence}` proposal records customer proof.
- The register stores who proposed and approved it, and that the environment is production is their declaration.
- `Period` is `YYYY-MM`, and `Customer` is an identifier.
- Proof counts only for the exact `Build`, and only while the requirement is `tested` for that build.

The requirement view also shows each test's pass and failure counts for the build, so that a flaky test passing only on a rerun stays visible.

## Support matrix and activation

A `path` proposal registers `Path{ID, Provider, Contract, Role, Version, Issuance, Template, Requirements}`:
- **`Issuance`** is `api`, `file` or `xlsx`.
- **`Template`** is an optional ERP upload template identifier.
- **`Requirements`** lists 1–256 requirement keys that the path depends on.

A path starts `designed`.

**Parallel reconciliation.** A `reconcile{Path, Period, Evidence, Matched, Differences}` proposal records one billing period of parallel reconciliation on real data.
- `Period` is `YYYY-MM`.
- `Matched` must agree with `Differences`: `Matched` with 0 differences, or not matched with at least 1. Otherwise the proposal is `ErrInvalid`.
- A repeat of the same period is `ErrConflict`.
- **Ordering.** Periods are ordered by `Period`, not by when they were recorded. Only closed periods count: a period in or after the current month (by the injected clock) is refused with `future_period`.
- **Cap.** At most 120 periods are kept per path, and the oldest is dropped beyond that.

**Categories.** A category holds for a path and build when at least one test tagged with that category, in one of the path's requirements, has a counted passing latest result for the build. All six categories must hold. No human evidence can replace a test.

**Activation.** An `activate{Path, Build, Proof}` proposal activates the path when every rule holds:
- the two latest periods by `Period` are consecutive months, both matched, and the latest is within 3 months of the current month (otherwise `reconciliation_missing`);
- every category holds for the build (otherwise `category_missing`);
- every listed requirement is `tested` for the build (otherwise `requirement_not_tested`);
- neither the activation's proposer nor its approver bound a harness whose counted results back the path's requirements, or proposed or approved one of the path's reconciliation periods (otherwise `self_verification`).

`Proof` is the hex digest of the real-path proof. Activation records the build.

**Deactivation.** `Deactivate(path, reason, actor)` lets one human deactivate a path at once. Reactivation needs a new `activate`.

## Release gate

`Gate(Release{Version, Build, Requirements, Paths})` is a read:
- **Form.** `Requirements` lists 1–10,000 keys and `Paths` lists 1–1,000 path IDs, each strictly ascending.
- **Problems.** It returns every problem, sorted by problem then subject:

  | Problem | When |
  | --- | --- |
  | `unknown` | a requirement or path is not registered |
  | `common_unregistered` | the register holds no unretired `Common` requirement, so that the DB-01 set cannot be absent by mislabelling |
  | `retired` | a selected requirement is retired. Its replacement is named |
  | `common_missing` | a `Common` requirement anywhere in the register, not retired, is not selected. Common file features can therefore never be omitted, even for an API-only customer |
  | `not_tested` | a selected requirement is not `tested` for the build |
  | `blocking` | a `Blocking` requirement anywhere in the register, not retired, is not `tested` for the build, whether or not it is selected |
  | `path_inactive` | a selected path is not active |
  | `path_build` | a selected path was activated for a different build |
  | `path_reconciliation` | a selected path's latest recorded period is unmatched |
  | `path_requirement` | a selected path's requirement is not among the selected requirements |
  | `path_evidence` | a selected path's requirement is retired or not `tested` for the build, or a required category has no counted passing tagged test for the build |

- **Result.** `Ready` is true only with no problems.
- **Templates.** The result carries the state of each selected requirement. It shows as verified only the templates of selected paths that are active for this build and have no path problem. A `Path(id)` read never shows a template as verified, because verification depends on the build and the release.

## Reads

Every read returns a copy:
- `Requirement(namespace, id)`, with its state for a given build;
- `Path(id)`, with its state, reconciliations and categories;
- `Results(namespace, test, build)`, one key;
- `Proposal`.

## Bounds

| Bound | Limit |
| --- | --- |
| Requirements | 10,000 |
| Paths | 1,000 |
| Result keys | 1,000,000 (`ErrLimit` for new keys) |
| Builds | 10,000 |
| Implementations kept per requirement | 16 |
| Reconciliation periods per path | 120 |
| Harnesses | 100 |
| Requirements and paths per release | 10,000 and 1,000 |
| Identifiers | 1–128 bytes of `[A-Za-z0-9._:-]` |
| Text | 1–200 bytes of UTF-8, with no control characters |

## Residuals

- Running the tests and reconciliations. Classifying the proof environment as production (`modules/sandbox`). Wiring `modules/perfgate` readiness and `modules/simulation` results in as evidence.
- Release profiles linking input volume, hardware, targets and artifacts. These live in `modules/perfgate`, and linking them to a release is a residual.
- The cutover itself, the operational handover and the choice of the first release's supported combinations, acceptance owners, schedule and proof period. These are owner decisions.
- Commit ancestry between builds and implementations. A build registered after an implementation record can still point to an older commit; this is a trust assumption of the gate.
- Merging two features into one newly registered replacement, which two humans can declare.
- Making `failing` sticky until a human acknowledges it. For now, flaky tests are shown through their counts.
- Persistence and screens.
- A reconciliation period older than the 120 kept periods can be proposed again; it is inserted and dropped at once, with no effect.
- AUD-DB-01 and the HB-41-V system-level cases are NOT_RUN.

# Verification plan

Literal tests:
- **Forms and bounds.** Every form rule and bound.
- **Proposals.** Proposal rules and actor kinds.
- **Requirements.** Namespaces kept apart, `unknown_test`, and every state, including:
  - `failing`;
  - a revoked harness's latest result as NOT_RUN;
  - reimplementation making results NOT_RUN;
  - proof kept apart from `tested`, per build.
- **Self-verification.** At `Record` and on read.
- **Retirement.** `weaker_replacement`, and common and blocking requirements in another namespace.
- **Paths.**
  - Reconciliation: the two latest periods, consistency of `Matched` and `Differences`, unmatched periods after activation, and repeats.
  - Every activation refusal, including categories without tagged passing tests, and activation self-verification.
  - Deactivation.
  - Verified templates.
- **Gate.** Every problem, several problems at once, common features required for an API-only release, a blocking requirement that is not selected, and a ready release.
- **Concurrency.** Concurrent calls under `-race`.

Property test: random requirements, results, paths, reconciliations and releases. An independent model of states and gate problems must match.

Checks: `-race -count=10`, the 386 build, gofmt, vet, `go test -race ./...`, 100% statement coverage and mutation testing.

# Review history

- Round 1 (independent reviewer, read-only): FAIL. Findings and what changed:
  - H1. Common and blocking sets are register-wide, and there is retirement with replacement.
  - H2. `self_verification` at `Record`, on read and at activation.
  - H3. The latest result is chosen from all results, and a revoked latest is NOT_RUN.
  - H4. Categories come only from tagged passing tests.
  - M1. The two latest periods, `path_reconciliation`, and `Matched` consistency.
  - M2. Builds are registered, and reimplementation makes earlier results NOT_RUN.
  - M3. Proof is per build, with its people recorded.
  - M4. Templates are verified only per gate and build.
  - L1. At least one requirement and one path per release.
  - L2. Failure counts are shown, and stickiness is a residual.
  - L3. Bounds, latest-only result keys, and problem ordering.
- Round 2 (same reviewer): FAIL. Findings and what changed:
  - M1. Replacements must be newer, unretired, used once and not the requirement itself.
  - M2. Implementers and build approvers are in the self-verification set.
  - M3. Retirement without a replacement for non-common requirements, listed in `RetiredBlocking`.
  - M4. Periods are ordered by `Period`, `future_period` and a 3-month staleness rule are added, and the cap drops the oldest.
  - L1. `common_unregistered`.
  - L2. Reconciliation people are excluded from activation.
  - L3. Kept as a trust assumption.
- Round 3 (same reviewer): PASS. Low items folded in:
  - L1. The merge case is reworded as a residual.
  - L2. Only closed months can be recorded.

# Change review resolution

- Change review round 1 (separate independent reviewer, read-only): FAIL. Findings and what changed:
  - M1. A template could show as verified after a path's requirement stopped passing. Added `path_evidence`, checked on every gate.
  - M2. Read-side self-verification lacked a test that fails without it. Added a test where a harness binder later registers a requirement listing the same test, and a single-implementation implementer test.
  - M3. The property test covered states only. Added an independent model of gate problems, ordering, `RetiredBlocking` and states over random registers and releases.
  - L1. The unmatched-latest test now advances the clock instead of editing state.
  - L2. `ResultView.Counted` now means counted for every requirement that lists the test.
  - L3. The activation proof digest is kept on the path and shown in `PathView.Proof`.
  - L4. A retirement without replacement has subject `ns/id`.
  - L5. Kept as a residual.

- Change review round 2 (same reviewer): PASS. Low items folded in:
  - L-a. `Deactivate` clears the kept proof as well as the build.
  - L-b. The gate property test now also checks each selected requirement's state value. Active paths, `path_build` and templates stay covered by literal tests.
  - L-c. A meaningless assertion was removed.
