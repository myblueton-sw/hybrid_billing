---
type: ImplementationPlan
title: P05 release evidence API over modules/releasetrace and modules/perfgate with synthetic sample requirements, builds, test results, support paths and load runs seeded through the modules' own flows
description: Phase 5i of the pre-launch plan. Adds a release evidence read for the five P05 tabs (matrix, requirements, artifacts, load, decisions) and administrator commands with a reason to approve and withdraw retirement proposals and to propose a requirement or load profile retirement. Nothing activates a support path, records customer proof, approves a load budget, records a result, revokes a harness or claims a release; activation, support decisions, revocation, evidence files and real measurements are shown as not connected.
status: stable
tags: [HYB-290, pre-launch, release-evidence, api]
sources:
  - resource: ../HB-257/plan.md
  - resource: ../HB-254/plan.md
  - resource: ../HB-251/plan.md
  - resource: ../HB-288/plan.md
  - resource: ../HB-288/verification.md
  - resource: ../HB-286/plan.md
---

# Scope and ownership

[HYB-290](https://linear.app/hybrid-billing/issue/HYB-290) adds the API behind P05 (support combinations and release evidence). Tickets moved from Linear team HB to team HYB; HYB-n is the old HB-n, so earlier documents cite HB-n. Base: main `6aabb35`. The P05 screen follows in a separate ticket. The plan follows the merged [HB-288 plan](../HB-288/plan.md) and [verification](../HB-288/verification.md) (retention) and the [HB-286](../HB-286/plan.md) recovery precedent, and binds the modules designed in [HB-254](../HB-254/plan.md) (release trace) and [HB-251](../HB-251/plan.md) (scale gates).

**Ownership**
- Root (Claude Code) owns the plan and integration. Implementation may be delegated to one writer in this worktree.
- An independent read-only reviewer reviews the design and the change, with a security focus (actor kinds, two-person rules, no system actor or result recording from requests, nothing that reads as a release, activation or passed measurement, no exposure of harness identities or customer fields).
- Machine: mac_mini.

**Allowed files**
- `modules/releasetrace/exports.go` (new) and `exports_test.go`: two read-only exports, listed under Module exports. No rule changes. `modules/perfgate` is not changed.
- `apps/api/server`:
  - new `evidence.go` (read, release trace part), `evidenceload.go` (read, load part), `evidencecmds.go` (commands and error mapping), `evidenceseed.go` (seed and server store), and tests `evidence_api_test.go`, `evidence_commands_test.go`, `evidence_edges_test.go`;
  - registration in `admin.go` (after `s.retentionDefinitions()`, `admin.go:361`), the route in `boundary.go` (after `boundary.go:39`), the registries and store in `server.go` (next to `server.go:97-98,175-176,234`), the seed step in `seed.go` (after `seed.go:109`);
  - the seed step count in `world_test.go:57` (16 to 17).
- `docs/features/HYB-290/*`.
- No other module, no console file, no `screens.json` change. `docs/ui/screen-binding-register.md` is not updated, as in HB-286 and HB-288 (residual).

**Non-goals**
- No release, deployment, signing, cutover or support-path activation, from requests or the seed. `releasetrace.ChangeActivate` is used nowhere and `Deactivate` is never called.
- No customer proof (`ChangeProof`) anywhere; no customer or environment identifier exists in the sample.
- No harness revocation: `RevokeHarness` of either module is never called.
- No test result or load run recorded from a request; no system or assistant actor from a request; no harness binding from a request.
- No load budget approval (`perfgate.ChangeApproval`) anywhere, so no load verdict and no load gate can read `passed`.
- No requirement, implementation, build, path, reconciliation, profile, hardware or budget proposal from requests: these assert facts about code, data, hardware or targets that only real evidence or the owner can supply.
- No evidence file, log or artifact store; digests are shown, contents are not connected.
- No loading of the real planning trace (`release-gates.json`, `acceptance-trace.json`); requirements are sample records in a sample namespace.
- No durable decision ledger; registers, store and journal reset with the process (HB-257 residual).

# Observed baseline (facts from code)

## `modules/releasetrace`

- **Registry.** `NewRegistry(clock)` refuses a nil clock (`registry.go:107-114`). Bounds are module constants (`registry.go:8-24`): 10,000 requirements, 1,000 paths, 1,000,000 result keys, 10,000 builds, 100 harnesses, 120 reconciliation periods per path, 16 implementations kept per requirement, 64 interfaces, 256 tests per requirement, 256 requirements per path, pending proposals 1,000 in all and 250 per proposer, applied proposals kept 1,000.
- **Identifiers and text.** IDs are 1 to 128 bytes of `[A-Za-z0-9._:-]` (`common.go:15-26`); free text is 1 to 200 bytes of valid UTF-8 without controls (`common.go:28-38`, unexported). Builds are `MAJOR.MINOR.PATCH` decimals (`common.go:62-73`); digests are 64 lowercase hex, commits 40 or 64 (`changes.go:34-36`); periods `YYYY-MM` (`changes.go:38-48`).
- **Actors** (`registry.go:129-143`, `results.go:32-51`). `human()` returns `ErrInvalid` for a malformed actor, `ai_not_allowed` for an assistant, `not_human` for a system actor. Human only: `Propose`, `Approve`, `Withdraw` (`registry.go:155-241`), `RevokeHarness` (`results.go:70-85`), `Deactivate` (`reads.go:49-64`). System only, and only the identity bound to an unrevoked harness: `Record` (`results.go:32-68`, `not_harness`).
- **Proposals.** Kinds `requirement`, `retire`, `implemented`, `build`, `harness`, `proof`, `path`, `reconcile`, `activate` (`types.go:21-29`).
  - **Silent duplicate.** The same proposer proposing the same encoded change returns the existing pending proposal (`registry.go:166-170`).
  - `Approve` refuses an unknown number (`ErrNotFound`), a non-pending proposal (`not_pending`) and self-approval (`self_approval`), re-runs `check`, applies (`registry.go:190-220`). **A repeat approval by the same approver of an applied proposal returns the view without an error** (`registry.go:201-202`). Applied proposals beyond 1,000 are deleted (`registry.go:214-218`).
  - `Withdraw` is limited to the proposer (`not_proposer`) and a pending proposal (`not_pending`) and deletes it (`registry.go:222-241`). `Proposal(number)` is the only proposal read; there is no list (`registry.go:243-251`).
- **Checks** (`changes.go:185-325`, pure: no writes):
  - `retire` (`changes.go:255-279`): requirement registered (`ErrNotFound`), not retired (`retired`); without a replacement only for a non-common requirement (`common_needs_replacement`); a replacement must be registered, unretired, not already a replacement, registered later, and as strong (common and blocking preserved) (`weaker_replacement`).
  - `proof` needs a registered build and a tested requirement (`changes.go:221-233`); `reconcile` needs a closed month and a new period (`changes.go:281-296`); `activate` needs a known build, two consecutive matched reconciliations no older than three months, a passing counted result per category, every path requirement tested, and no person of the reconciliations or counted harnesses as proposer or approver (`changes.go:298-325`, `results.go:140-165`).
- **Requirement state** (`results.go:97-123`): `designed` without an implementation; `failing` when a counted latest result failed; `implemented` until every test has a counted result; `customer_proven` with a proof for the build, else `tested`. A result counts only when the build was registered after the latest implementation record, the harness is not revoked, and the harness's proposer and approver overlap none of the build's, the requirement's or the latest implementation's people (`results.go:7-30,87-95`). `Record` refuses a self-verified result at once (`results.go:49-50`).
- **Two-person consequence.** Because of the overlap rule, a counted result needs at least four distinct human IDs: two who register requirements, implementations and builds, and two others who bind the harness. The module's tests use eight (`world_test.go:13-24`).
- **Reads.** `Requirement(key, build)` with per-test `Counted`, `Passed`, `Passes`, `Failures` (`reads.go:3-23`); `Path(id)` with state, build, proof digest and reconciliations (`reads.go:25-37`); `Results(namespace, test, build)` with harness ID, time, counts and `Counted` (`reads.go:39-47`, `results.go:167-175`); `Gate(release)` with `Ready`, sorted `Problems`, per-key `States`, `Templates`, `RetiredBlocking` (`gate.go:18-100`). There is no read for builds, harnesses, requirement lists, path lists or result lists. `TestState` does not say whether a result exists; a result always increments `Passes` or `Failures` (`results.go:62-66`), so `Passes+Failures == 0` means none.
- **Gate details.** A requirement retired without a replacement is listed under `RetiredBlocking` whether or not it is selected (`gate.go:44-47`). A path's `path_evidence` problem is absent exactly when every path requirement is not retired and tested at the build and every category has a counted passing result (`gate.go:85-87,102-114`). Gate never passes a path that is not active (`gate.go:72-73`), and only `ChangeActivate` makes a path active (`changes.go:380-382`).
- **`RevokeHarness`** is single-person, immediate and silent on repeat (`results.go:79-84`); there is no un-revoke and no read of the revoked flag. Not used here.
- **Source rules** (`source_test.go:11-46`): no comments, no imported path with a dot, no binary-fraction word, at most 1,500 lines. `go test -cover ./modules/releasetrace/` reports 100.0% (run on this base).

## `modules/perfgate`

- **Registry.** `NewRegistry(clock)` refuses a nil clock (`registry.go:81-87`). Bounds (`registry.go:8-26`): 1,000 profiles, 100 hardware records, 10,000 budgets, 100 harnesses, 1,000 runs per profile and 100,000 in all, page 1,000; P0 floors: more than 1,000,000 rows a day, at least 30,000,000 a month, at least 100,000 a run.
- **Actors** as in releasetrace (`registry.go:102-116`); `Record` is system only and bound-harness only (`runs.go:39-75`); `RevokeHarness` is single-person, immediate and silent on repeat (`runs.go:77-92`), not used here.
- **Proposals.** Kinds `profile`, `hardware`, `budget`, `approval`, `harness`, `retire` (`types.go:21-26`). Silent duplicate (`registry.go:139-143`) and silent repeat approval (`registry.go:174-175`) as in releasetrace; `Withdraw` deletes (`registry.go:195-214`); `Proposal(number)` only (`registry.go:216-224`).
- **Checks** (`changes.go:130-212`): `profile` must reach its gate's P0 floor (`below_p0`); `budget` must name every required metric and, for daily inflow and catch-up, a throughput at least the profile's daily rate (`missing_metric`, `below_p0`); `approval` needs the active budget and an approver who is neither the budget's proposer nor its approver (`not_active`, `same_people`); `retire` needs a registered, unretired profile (`ErrNotFound`, `retired`).
- **Budget states.** Approving a `budget` proposal makes it `proposed` and active for its profile; only an applied `approval` makes it `approved` (`changes.go:224-229`).
- **Verdicts** (`runs.go:128-175`), in order: `other_data`, `incomplete`, `incorrect`, `revoked`, `self_verified`, `missing_metric`, `unbudgeted`, `other_hardware`, `over_budget` (any measured value above its limit, or throughput below it), `provisional` (budget only proposed), `predates_budget`, `passed`. Verdicts are recomputed on every read against the profile's current active budget. `Record` refuses a run whose harness people overlap the profile's or the active budget's people (`runs.go:59-60`, `self_verification`); with an approved budget that needs five distinct human IDs.
- **Reads.** `Profile(id)` with the retired flag, `Hardware(id)`, `Budget(id)`, `ActiveBudget(profile)`, `Run(number)`, `Runs(profile, after, limit)` oldest first, `Gate(gate, build)`, `Readiness(build)` (`reads.go:3-160`). Gate status: `failing` when a live profile's latest run for the build fails, else `not_run` when the gate has no live profile or one has no run for the build, else `provisional` unless every live profile passed (`reads.go:94-123`). There is no list of profiles, hardware, budgets or harnesses.
- **Source rules** as in releasetrace. `go test -cover ./modules/perfgate/` reports 100.0% (run on this base).

## Importers

No package outside the two modules imports either of them (searched `apps/` and `modules/`).

## Server and commands (as in HB-288)

- Commands run under the server's domain write lock and move the revision on every completed mutating command. `Resolve` receives no actor and runs before the journal lookup; a target that no longer resolves is `permission_denied` on resend. Handler-coded failures are not journaled; `internal_error` is journaled and replayed.
- Only `invalid_request` carries fields. The rationale is capped at 256 bytes (`directory.go:13`).
- Command targets are command IDs: 1 to 128 bytes of `[A-Za-z0-9._:-]`, no `/` (`modules/commands/types.go:297-310`). A requirement key `namespace/id` therefore cannot be a target as is.
- Reads use `admitted` (`users.go:42`), which decides `AdministerInstallation` on the installation kind. Fault hooks use `faulty` (`admin.go:78`) through `moduleCall` (`integrationseed.go:174-179`); `stamp` renders RFC 3339 seconds (`users.go:40`); `installKept` = 200 (`installseed.go:18`); `installSeedReason` (`installseed.go:20`); `notConnected`, `stateConfigured`, `stateNotConfigured` (`integrations.go:18-20`); `alreadyProposed`, `notConsoleKind` (`recovery.go:16-17`); `ownProposal` (`installation.go:22`); `canonicalNumber` (`retentioncmds.go:58-61`); `personOf` (`retention.go:328`); eligible approvers `s.retentionApprovers(verifier, now)` (`retention.go:843-855`), reused unchanged.
- Sample users: `SampleAdmin`, `SampleAdmin2`, `SampleMember` (`seed.go:22-26`); only the two administrators hold the installation grant (`seed.go:229-230`).

## P05 design

`screens.json` (`apps/console/src/modules/administration/screens.json:199-247`) names the tabs `matrix`, `requirements`, `artifacts`, `load`, `decisions` and the actions: link evidence, review differences, record a support decision, request activation review (approval). The P05 spec (`docs/ui/screens/P05/spec.md` in the authorized local workspace; not in this tree) requires separate design, implementation, test and customer-proof columns and forbids a pass from mapping alone (`spec.md:33,59`), different text for none, unknown, not aggregated, error and a real zero (`spec.md:63`), fixed snapshots with cursor paging for lists (`spec.md:103`), a `revoked` state that hides details and follow-up commands after access is withdrawn (`spec.md:97`), never shows other customers' existence or counts (`spec.md:25`), says mock success is not live evidence (`spec.md:74`) and leaves the first supported combinations, performance hardware and latency targets, evidence validity and activation accountability open (`spec.md:137-139`). The preview `apps/console/src/modules/administration/pages/release-evidence/Screen.tsx:3-7,142-155` states that requirement traces and execution artifacts are not connected and that no measurements are connected.

Planning context (local untracked workspace, not this tree): `release-gates.json` lists 43 gates, all `NOT_IMPLEMENTED`, status "proposal; no release approved"; `acceptance-trace.json` is "design trace only; not executable tests or pass evidence"; `scale-validation.md:54` says every SCALE test is not run and that undecided targets make a result "measured, release performance decision pending", not a pass.

# Design

## Registries

`releasetrace.NewRegistry(s.clock)` and `perfgate.NewRegistry(s.clock)` built in `server.New` after the retention registry, as fields `trace` and `perf`; store field `ev` from `newEvidenceStore()`. The modules have no configurable limits.

## Module exports (`modules/releasetrace/exports.go`)

A reviewed module change, read-only, no rule change:
- `func ValidText(text string) bool { return validText(text) }`, so the server checks a retirement reason with the module's rule and `ErrInvalid` stays unreachable from requests.
- `func (r *Registry) Check(c Change, proposer, approver string) error`: under `enter`/`leave`, `clean(c)` (`ErrInvalid` when malformed), then `r.check(c, proposer, approver)`. `check` is pure (`changes.go:185-325`), so `Check` changes nothing. It skips what `Propose` and `Approve` add around `check` (duplicate return, limits, actor kind, pending state, self-approval), so a command the read shows as allowed may still answer `prerequisite_failed` (`ErrLimit`).

perfgate needs no export: its only console kind, `retire`, depends on facts `Profile(id)` returns.

Tests (`exports_test.go`): `ValidText` agrees with `Propose` of a `retire` on the boundary cases (empty, 200 and 201 bytes, controls, invalid UTF-8); `Check` returns what `Propose` (approver empty) and `Approve` return for every refusal and error of every kind and leaves proposals, requirements, paths and results unchanged; 100% statement coverage and `TestSourceRules` pass.

## Sample seed (only with `--seed sample`; one counted step `release-evidence`)

**Actors**
- Humans `SampleAdmin` (A1) and `SampleAdmin2` (A2) register every requirement, implementation, build, path, reconciliation, profile, hardware record and budget: A1 proposes, A2 approves.
- **Two sample reviewers bind the harnesses** (root decision 1): `sample-evidence-reviewer-1` proposes and `sample-evidence-reviewer-2` approves the release trace harness and the load harness. They are human-kind module actors, used in the seed only, never directory users, never accepted from requests; they cannot sign in, and the read shows them with `login: null` and `directory: false`. They are needed because a counted result or an accepted run requires harness people disjoint from the people who registered what is tested (baseline, two-person consequence). **Recorded deviation from the [HB-257 plan](../HB-257/plan.md) seed actor rule** (the seed acts as the bootstrap administrator or as named system actors), guarded by the AST test like the system actors.
- System actors built only inside `seedEvidence`: `sample-trace-runner` (the release trace harness identity) and `sample-perf-runner` (the load harness identity). They only call `Record`. Their IDs are never exposed.
- All digests and commits are `sha256` hex of fixed labels such as `sample:design:sample-common-xlsx`, truncated to 40 characters for commits. They name no real file, commit or data set.

**Release trace steps, in order, with T the seed time and M the calendar month of T** (namespace `sample`):
1. Requirements (A1 proposes, A2 approves), each with source `sample-source`, owner `sample-owner`, interface `sample-api`, design digest of its label:

   | ID | Common | Blocking | Tests (category) |
   | --- | --- | --- | --- |
   | `sample-common-xlsx` | yes | no | `t-xlsx-amount` (amount), `t-xlsx-isolation` (isolation) |
   | `sample-approval` | no | no | `t-approval` (approval), `t-duplicate` (duplicate) |
   | `sample-recovery` | no | yes | `t-bulk` (bulk), `t-recovery` (recovery) |
   | `sample-erp-template` | no | no | `t-erp-template` (none) |
   | `sample-api-export` | no | no | `t-api-export` (none) |
   | `sample-legacy-report` | no | no | `t-legacy` (none) |

2. Build `0.0.1` (commit of `sample:commit:0`).
3. Implementation records for every requirement except `sample-erp-template` (commit of `sample:commit:1`).
4. Build `0.0.2` (commit of `sample:commit:2`), registered after the implementations.
5. Harness `sample-trace-harness`, identity `sample-trace-runner` (reviewer 1 proposes, reviewer 2 approves).
6. Paths (A1, A2): `sample-path-api` (provider `sample-provider`, contract `sample-contract`, role `sample-role`, version `sample-1`, issuance `api`, no template, requirements `sample-approval`, `sample-common-xlsx`, `sample-recovery`); `sample-path-xlsx` (same provider, contract, role and version, issuance `xlsx`, template `sample-template-xlsx`, requirements `sample-common-xlsx`, `sample-erp-template`).
7. Reconciliations (A1, A2) for month M−1: `sample-path-api` matched (0 differences); `sample-path-xlsx` not matched (3 differences). One period only, so no path is ever two-period reconciled.
8. Results recorded by `sample-trace-runner`, evidence digest of `sample:result:<test>:<build>:<n>`:
   - build `0.0.2`: `t-xlsx-amount` pass, `t-xlsx-isolation` pass, `t-approval` fail then pass, `t-duplicate` pass, `t-recovery` pass, `t-bulk` fail;
   - build `0.0.1`: `t-xlsx-amount` pass (not counted: the build predates the implementation).
9. Pending: A1 proposes retiring `sample-legacy-report` without a replacement, reason `Sample scope change`; A2 can approve it from the console.

Resulting states at `0.0.2`: `sample-common-xlsx` and `sample-approval` `tested`; `sample-recovery` `failing`; `sample-api-export` and `sample-legacy-report` `implemented`; `sample-erp-template` `designed`. Nothing is `customer_proven`. At `0.0.1` every implemented requirement reads `implemented`.

**Sample candidate (server store).** `{version: sample-candidate-1, build: 0.0.2, requirements: sample-api-export, sample-approval, sample-common-xlsx, sample-recovery, paths: sample-path-api}`. Its gate reads `ready: false` with `blocking`, `not_tested` (twice), `path_evidence` and `path_inactive`. It is a gate check, not a release.

**Load steps, in order** (A1 proposes, A2 approves unless stated):
1. Hardware `sample-hw-small`: 8 CPUs, 32,768 MiB, 500 GiB, `ssd`, spec digest of its label.
2. Profiles: `sample-e2e` (`e2e`, 100,000 rows a run), `sample-daily` (`daily_inflow`, 1,200,000 rows a day), `sample-month` (`monthly_close`, 30,000,000 rows a month), `sample-resume` (`interrupt_resume`, 100,000 rows a run, 1 failure injection); dataset and expected digests of their labels. No `catch_up` profile.
3. Budgets on `sample-hw-small`: `sample-budget-e2e` for `sample-e2e` (completion 600,000 ms, max RSS 4,294,967,296 bytes, p95 500 ms, p99 1,000 ms, throughput 200 rows/s) and `sample-budget-daily` for `sample-daily` (completion 86,400,000 ms, max RSS 4,294,967,296 bytes, p95 500 ms, p99 1,000 ms, throughput 20 rows/s, above the 14 the daily rate requires). Both stay `proposed`: no approval change is ever proposed.
4. Harness `sample-perf-harness`, identity `sample-perf-runner` (reviewer 1, reviewer 2).
5. Runs recorded by `sample-perf-runner` on build `0.0.2` with all eight non-resume metrics:
   - `sample-e2e`: 100,000 rows, correct output, completion 420,000 ms, p95 300 ms, p99 800 ms, throughput 240 rows/s, all within the budget → `provisional`;
   - `sample-daily`: 1,200,000 rows over 86,400 s, correct output, measured throughput 13 rows/s (below the 20 limit) and p99 1,500 ms (above the 1,000 limit); both limits fail → `over_budget`;
   - `sample-month`: 1,000,000 rows → `incomplete`;
   - `sample-resume`: no run.
6. Pending: A2 proposes retiring `sample-month`; A1 can approve it from the console.

Readiness at `0.0.2`: `e2e` provisional, `daily_inflow` failing, `monthly_close` failing, `interrupt_resume` not run, `catch_up` not run; `ready: false`. All values are synthetic and are not performance results.

Nothing is built, signed, deployed, measured, reconciled, revoked or activated; every record is a module fact about sample identifiers.

## Server store (`evidenceseed.go`)

The modules cannot list what the read needs, so the server keeps, separate from the other stores:
- **Requirements:** key, in registration order; the 1-based position is the command target number. Seed only, never removed.
- **Builds:** ID and commit, in registration order (the module has no build read). Seed only.
- **Paths, profiles, hardware, budgets:** IDs. Seed only.
- **Harnesses:** module (`trace` or `perf`) and ID; never the identity. Seed only.
- **Result keys** `(namespace, test, build)` for trace results and nothing for load runs (`Runs` lists them). Seed only.
- **Candidate:** the sample candidate above, or none.
- **Proposals:** module, number, kind, created time, proposer, rationale, closed by, closing rationale and the last module view; one list for both modules, kept like `installStore.keep` (at most 200, pending never dropped). Lookup is by module and number.
- **`store_retired` is defensive only**, as `retired` in HB-288: should a module answer `ErrNotFound` or `not_pending` for an entry stored as `pending` (reachable only by a direct module call in a test), the entry becomes `store_retired` and the command answers `invalid_stage`. The name keeps it apart from a retired requirement or profile.

Facts the modules can read (requirement views and states, path views, result views, gate problems, profiles, budgets, runs, verdicts, readiness) are never stored.

**Visibility.** Both registers are installation-wide: they hold no tenant, workspace or customer. The read and every command serve only the boot workspace administrator; another workspace gets `not_configured` and every target is not found (root decision 3).

## Read (admin only)

`GET /api/v1/release-evidence`, admitted like the retention read, under the domain read lock. For a workspace other than the boot workspace it returns `{revision, now, state: not_configured}`. For the boot workspace it always returns the full shape. Any module error or roster failure makes the read answer `dependency_unavailable`.

General rules:
- **Origin.** Every section carries `origin: sample_seed` when the server store was filled by the seed and `origin: empty` otherwise. Nothing in the read is a release, an activation, a customer proof or a measured result.
- Counts are exact for the stored records and every enumerated key is present, so `0` is a real zero. A source the API does not have is the string `not_connected`, never `0` or an empty list. A value that does not apply or does not exist yet is `null` (for example `latest_outcome: null` when a test has no result, `template: null`, `active_budget: null`).
- Every capped list carries `total` and `more`. Caps: 50 rows per list unless stated. There is no cursor paging (residual).
- Times are RFC 3339 UTC. People are `{id, login, directory}`: `login` is `null` and `directory` false when the directory does not know the ID (the sample reviewers).
- **Candidate.** `candidate: {version, build, release: not_connected}` or `null`. The candidate build is the default build of every per-build value. **Without a candidate** (`--seed none`): `candidate`, the requirements tab `gate`, the load `readiness`, every path's `blockers` and `last_observed`, every requirement's `tests` outcomes and every profile's `status` are `null`, `states` is `{}`, and `Gate` and `Readiness` are never called.
- The read lists eligible approvers once with the admitted session's verifier (`s.retentionApprovers`, `retention.go:843-855`). The blocker `no_eligible_approver` means there is no eligible approver other than the reader (an active member with an active user holding `AdministerInstallation`), so no second person could approve what the reader proposes.

**`matrix`** (stored paths by ID):
- `id`, `provider`, `contract`, `role`, `version`, `issuance`, `template` (or `null`), `state` (`designed`, `active`), `build` (or `null`);
- `requirements`: per path requirement `{label: "namespace/id", target, state}` at the candidate build;
- `columns`: `design` (requirements registered), `implementation` (with a commit), `test` (`tested` or `customer_proven`), `customer_proof` (`customer_proven`), each `{count, of}` over the path's requirements, plus `failing`;
- `categories`: all six categories, each `true` when a path requirement has a test of that category whose result at the candidate build is counted and passed (from `Requirement` views). A test asserts that `path_evidence` is absent exactly when every path requirement is tested, none is retired and every category is `true` (`gate.go:102-114`);
- `reconciliations`: the last 12 periods `{period, matched, differences}` with `total` and `more`;
- `last_observed`: the latest result time among the path's stored result keys at the candidate build, or `null`;
- `blockers`: the problems of `Gate({candidate version, candidate build, the path's requirements, [path]})` whose subject is the path (`path_*`) or one of its requirements, as `{problem, subject}`; global problems appear only in the requirements tab gate;
- `activation`, `support_decision`, `customer_proof_intake`: `not_connected`.

**`requirements`** (stored requirements in key order):
- `target` (`trace-requirement-<n>`), `label`, `namespace`, `id`, `source`, `owner` (module ID, not a person), `interfaces`, `common`, `blocking`, `design` (digest), `commit` (latest or `null`), `retired`, `replacement` (label or `null`);
- `states`: build ID to state for every stored build (at most the newest 10, `builds_more`), so the screen can compare builds (P05 "review differences") without a query;
- `tests` at the candidate build: `{id, category (or null), counted, latest_outcome: "pass" | "fail" | null, passes, failures}` (`null` when `passes + failures == 0`);
- `retire`: pending retirement proposal number or `null`;
- `replacements`: the stored requirements `m ≠ n` for which `Check(retire{key, replacement: m}, reader, "")` returns `nil`, as `{target, label}`, at most 20 in key order, with `total` and `more`;
- `can_retire` with blockers in order `retired`, `already_proposed`, `no_eligible_approver`, then `Check(retire without replacement, reader, "")`: `nil` gives `can_retire: true, replacement_required: false`; `common_needs_replacement` gives `replacement_required: true` and `can_retire: true` when `replacements` is not empty, else `can_retire: false` with blocker `no_replacement`; any other result is the blocker;
- `gate`: the candidate's `Gate` result: `ready`, `problems` (at most 100, `total`, `more`), `retired_blocking` (labels), `templates`, with `release: not_connected`.

**`artifacts`**:
- `builds`: `{build, commit, order}`;
- `harnesses`: `{module, id, revocation: not_connected}`;
- `results` (stored keys, newest first by time): `{namespace, test, build, counted, latest_outcome: "pass" | "fail", evidence (digest), harness, at, passes, failures}`;
- `files`, `logs`, `environment`, `content`: `not_connected` (digests have no stored content).

**`load`**:
- `hardware`: `{id, cpus, memory_mib, disk_gib, storage, spec}`;
- `profiles` (stored, by ID): `target` (`perf-profile:<id>`), `id`, `gate`, `axes` (all thirteen axes as integers), `dataset`, `expected`, `retired`, `active_budget` (`{id, state, hardware, limits: {metric: value}, proposer, approver, approval_by, approved_by, approved_at}` with `null` for an absent person or time, or `null`), `status` at the candidate build (from `Readiness`), `runs` (the newest 20 from `Runs(profile, 0, 1000)` with `total` and `more`: `{number, build, hardware, rows, duration_s, mismatches, interrupted, metrics (all nine keys, null when not measured), harness, at, verdict}`), `retire` (pending number or `null`), `can_retire` with blockers `retired`, `already_proposed`, `no_eligible_approver`, `target_too_long` (a profile ID over 115 bytes);
- `readiness`: `Readiness(candidate build)`: `ready` and per gate `{gate, status, profiles, retired}`;
- `measurement`, `planning_targets`: `not_connected` (no harness measures real load; the P05 planning targets are copy, not data).

**`decisions`**:
- `proposals` (both modules, newest first): `module` (`release_trace`, `perf_gate`), `number`, `target` (`trace-proposal-<n>`, `perf-proposal-<n>`), `kind`, `state` (`pending`, `applied`, `withdrawn`, `store_retired`), `created_at`, `proposer`, `approver`, `rationale`, `closed_by`, `closing_rationale`, `summary` by kind:
  - release trace: requirement `{label}`, retire `{label, replacement, reason}`, implemented `{label, commit}`, build `{build, commit}`, harness `{harness}`, path `{path}`, reconcile `{path, period, matched, differences}`, proof `{label, build, period}`, activate `{path, build}` (proof and activate only ever appear by injection);
  - load: profile `{profile, gate}`, hardware `{hardware}`, budget `{budget, profile, hardware}`, approval `{budget}`, harness `{harness}`, retire `{profile}`;
- `can_approve` with blockers in order `not_pending`, `store_retired` (from the stored view), `not_console_kind` (every kind but `retire`), `own_proposal`, then for release trace `Check(change, proposer, reader)` and for load the profile's retired flag (`retired`);
- `can_withdraw` with blockers `not_pending`, `store_retired`, `not_proposer`;
- `support_decisions`, `activation_review`: `not_connected`; `ledger: in_memory`.

**Never exposed:** harness identities (`sample-trace-runner`, `sample-perf-runner`), a proof's customer and environment (never seeded; dropped even when injected), any secret, token or key (none exist in these modules), and anything of another workspace (none exists).

## Commands (`release_evidence.*`, reason required, actor = the caller as a human)

| Command | Target | Fields | Module call |
| --- | --- | --- | --- |
| `release_evidence.approve` | `trace-proposal-<n>` or `perf-proposal-<n>` (retire only) | none | `trace.Approve(n, caller)` or `perf.Approve(n, caller)` |
| `release_evidence.withdraw` | `trace-proposal-<n>` or `perf-proposal-<n>` | none | `trace.Withdraw` or `perf.Withdraw` |
| `release_evidence.retire_requirement` | `trace-requirement-<n>` | `reason` (text, required), `replacement` (ID, optional, a `trace-requirement-<m>` target) | `trace.Propose(retire, caller)` |
| `release_evidence.retire_profile` | `perf-profile:<id>` | none | `perf.Propose(retire, caller)` |

**Permission (a deviation from P05).** Every command requires `AdministerInstallation` on the installation kind. P05 names evidence-recording, domain-decision and activation-request permissions; the gate has none. Definitions are built from one permission value per group (proposals: approve, withdraw; retirements: retire_requirement, retire_profile).

**Resolve** (store only, no module call; not found unless the workspace is the boot workspace):
- a proposal number in canonical decimal (`canonicalNumber`) with a stored entry of that module;
- a requirement position in canonical decimal within the stored list;
- a stored profile ID.

Stored records are never removed and closed proposals leave only by the 200 bound, so a resend of a successful command replays.

**Actors.** Every module actor is `{ID: ctx.User, Kind: KindHuman}`. No command builds a system or assistant actor, records a result or run, binds or revokes a harness, proposes any kind but `retire`, approves any kind but `retire`, or calls `Deactivate`.

**One mutation per command.** Each command makes exactly one mutating module call (`Propose`, `Approve` or `Withdraw`), and it is the last fallible step: prechecks, module reads (`Requirement`, `Profile`, `Check`) and fault hooks run before it, and nothing fallible runs after it (results come from the call's own return).

**Server prechecks** (fields only on `invalid_request`; a rationale over 256 bytes is `invalid_request` without fields):
- `approve`: the stored entry must be `pending`, else `invalid_stage` (covers the silent repeat approval, `registry.go:201-202` in releasetrace and `registry.go:174-175` in perfgate, and `store_retired`/`withdrawn`); the kind must be `retire`, else `invalid_stage` (`not_console_kind`), decided without a module call. Self-approval is refused by the module (`prerequisite_failed`); a release trace retirement whose replacement became weaker is refused by the module's re-check (`invalid_stage`).
- `withdraw`: the stored entry must be `pending`, else `invalid_stage`.
- `retire_requirement`:
  - `reason` must pass `releasetrace.ValidText`, else `invalid_request` on `reason`;
  - `replacement`, when given, must resolve to a stored requirement other than the target, else `invalid_request` on `replacement`;
  - `Requirement(key, candidate build)` retired is `invalid_stage`; a pending retirement entry for the key from anyone is `invalid_stage` (`already_proposed`; also covers the module's silent duplicate, `registry.go:166-170`);
  - the module refuses a common requirement without a replacement (`common_needs_replacement`) and a weaker replacement (`weaker_replacement`): `invalid_stage`.
- `retire_profile`: `Profile(id)` retired is `invalid_stage`; a pending retirement entry for the profile from anyone is `invalid_stage` (`already_proposed`; also covers the silent duplicate, `registry.go:139-143`).

**Error mapping** (one function per module, same table)

| Module result | Code |
| --- | --- |
| `Refusal` `self_approval` or `not_proposer` | `prerequisite_failed` |
| `Refusal` `ai_not_allowed` or `not_human` | `internal_error` (unreachable) |
| Any other `Refusal` (`not_pending`, `retired`, `common_needs_replacement`, `weaker_replacement`, …) | `invalid_stage` |
| `ErrNotFound` after the prechecks (defensive; the entry becomes `store_retired` when stored as pending) | `invalid_stage` |
| `ErrConflict` (only by injection) | `invalid_stage` |
| `ErrLimit` (defensive: requests propose at most one pending retirement per requirement or profile, far below the 250-per-proposer bound) | `prerequisite_failed` |
| `ErrInvalid` | `internal_error` (the server built a bad input) |

These codes carry no fields; the read's blockers explain them. A `not_pending` refusal for an entry stored as `pending` also marks it `store_retired`.

**Revision on `internal_error`.** As in HB-288: `internal_error` is journaled, so the first answer moves the revision once and a resend with the same key replays without moving it; with the one-mutation invariant that move never accompanies a partial module change.

**Results.** `fields` `module`, `proposal`, `target`; `business_state` the proposal state.

# Not connected

Shown as `not_connected` in the read and absent from the commands:
- support-path activation, deactivation and the activation review request (P05 "request activation review");
- support decisions (supported, limited, unsupported, unverified) as records (P05 "record support decision"); the module has no such record;
- customer proof intake;
- harness revocation (both modules) and harness binding from the console;
- evidence linkage from a real artifact store: files, logs, execution environment, evidence content (P05 "link evidence");
- recording test results or load runs from a harness outside the seed;
- requirement, implementation, build, path, reconciliation, profile, hardware and budget proposals from the console;
- load budget approval with a distribution, and therefore any `passed` load verdict or load gate;
- real load measurement and the planning targets as data;
- the real planning trace (43 gates in `release-gates.json`, acceptance trace) and its import;
- a durable decision ledger;
- the P05 `revoked` state (`spec.md:97`): the read refuses a caller without the grant, but there is no separate revoked-access view.

# Residuals for the owner

- First supported combinations, performance hardware and latency targets, evidence validity and activation accountability are owner decisions (P05 `spec.md:137-139`); every seeded value is synthetic.
- The two sample reviewers are module-only human IDs that are not directory users; a recorded deviation from the HB-257 seed actor rule (seed only). Adding real reviewer accounts is later work.
- With two administrators, retirements are the only two-person decisions the console can complete; budget approval would need a fifth person and is not connected.
- Harness revocation is not connected; a compromised harness cannot be revoked from the console in this build.
- No cursor paging or fixed snapshot for lists (P05 `spec.md:103`); lists are capped with `total` and `more`.
- `AdministerInstallation` for every command is a recorded deviation; P05's separate permissions and AU read access are later work.
- The release trace and load registers are installation-wide; a multi-workspace design must decide who sees them.
- Linking load readiness to the release gate and commit ancestry between builds and implementations remain HB-254 residuals.
- Persistence: registers, store and journal reset on restart; closed proposal entries beyond 200 no longer resolve.
- `docs/ui/screen-binding-register.md` is not updated, as in HB-286 and HB-288.
- All release gates and SCALE-01 to SCALE-12 remain NOT_RUN at system level.

# Verification plan

- **Module:** `exports_test.go` as under Module exports; `go test -cover` at 100% for `modules/releasetrace` and, unchanged, `modules/perfgate`; `TestSourceRules`.
- **API read:**
  - shape with the sample seed; boot workspace with `--seed none` (empty lists, `candidate`, `gate` and `readiness` `null`, `states` `{}`, `not_connected` sources, `origin: empty`, and a fault hook on `trace-gate` and `perf-readiness` that would fail the read proves neither is called); another workspace `not_configured`; non-admin member `permission_denied`;
  - every value against the modules, read directly by the test: `Requirement`, `Path`, `Results`, `Gate`, `Profile`, `Hardware`, `ActiveBudget`, `Runs`, `Readiness`, `Proposal`;
  - seeded states: tested, failing, implemented, designed at `0.0.2`; `implemented` everywhere at `0.0.1`; the `0.0.1` result row reads `counted: false, latest_outcome: pass`; `t-approval` passes 1 failures 1; `latest_outcome: null` for tests without results; no `customer_proven`;
  - matrix columns balance against the requirement states; categories agree with `path_evidence` on both paths and after injected results that complete or break a category and after an injected retirement of a path requirement; blockers per path; `sample-path-xlsx` `path_reconciliation`;
  - candidate gate `ready: false` with exactly `blocking`, `not_tested` ×2, `path_evidence`, `path_inactive`; readiness `ready: false` with the five gate statuses above; the `sample-daily` run reads `over_budget` with throughput 13 and p99 1,500;
  - **no reachable state through the API reads `ready: true` (release gate or readiness), a path `active`, a requirement `customer_proven`, a load verdict `passed` or a load gate status `passed`** (root decision 4; asserted after every command in the command tests);
  - blockers: `already_proposed`, `not_console_kind` (an injected pending `implemented` proposal), `own_proposal`, `not_pending`, `store_retired`, `not_proposer`, `no_eligible_approver` (after suspending the other administrator), `retired`, `target_too_long` (an injected profile with a 116-byte ID; 115 bytes stays retirable);
  - replacements: `sample-common-xlsx` reads `replacement_required: true`, `replacements` empty, `can_retire: false` with `no_replacement`; after an injected newer common requirement it lists that one and `can_retire` turns true; a non-common requirement lists only newer, unretired, unused and at least as strong ones, matching `Check` per candidate;
  - a read leaves every proposal, requirement, result, run and verdict unchanged.
- **API commands:**
  - each command's success, with transitions asserted through module reads:
    - A2 approves the seeded requirement retirement: `sample-legacy-report` retired; the candidate gate's `problems` and `ready` are unchanged (it is not selected) and `retired_blocking` gains `sample/sample-legacy-report` (`gate.go:44-47`);
    - A1 approves the seeded profile retirement: `monthly_close` becomes `not_run`, the profile listed under retired;
  - approve: self-approval (`prerequisite_failed`), a repeat approval refused with the revision unchanged, approval after withdraw, a non-retire kind refused without a module call while its proposer can withdraw it;
  - retire_requirement: bad and long reasons, unknown and self replacement (`invalid_request` on the field), a common requirement without a replacement, a weaker replacement, `already_proposed` from either administrator including identical content, retired target;
  - retire_profile: retired profile, duplicate;
  - `store_retired` (defensive): a stored pending entry withdrawn or approved directly in the module, then `approve` and `withdraw` through the API answer `invalid_stage` and the entry reads `store_retired`;
  - `ErrLimit` (defensive): the caller's 250 pending proposals filled by direct module proposals, then `retire_requirement` and `retire_profile` answer `prerequisite_failed`;
  - unknown, non-canonical and other-module targets, other-workspace and non-admin callers: `permission_denied`;
  - idempotent replay after state changes; an `internal_error` (fault hook on the mutating call) moves the revision once, leaves the modules unchanged, and its replay leaves the revision unchanged; a fault on each read before the mutating call leaves the modules unchanged; stale revision;
  - the mapping tables through a direct test, including unreachable codes.
- **Leak test.** The JSON of the read and every result contains neither harness identity, no `customer` or `environment` field or value of an injected proof, no token or key, and the reviewers only as `{id, login: null, directory: false}`.
- **AST tests** over all non-test files in `apps/api/server`:
  - `releasetrace.KindSystem`, `perfgate.KindSystem` and both `KindAssistant` appear only in `seedEvidence`; `evidence.go`, `evidenceload.go` and `evidencecmds.go` contain no `"system"` or `"assistant"` literal;
  - in `evidencecmds.go` every `releasetrace.Actor` and `perfgate.Actor` literal takes its `ID` from `ctx.User` and nothing else;
  - the selector `Record` on `s.trace` or `s.perf`, whether called or taken as a method value, appears only inside `seedEvidence`;
  - `RevokeHarness` and `Deactivate` are never selected on `s.trace` or `s.perf`, called or as method values;
  - `releasetrace.ChangeActivate`, `releasetrace.ChangeProof` and `perfgate.ChangeApproval` appear nowhere; outside `seedEvidence` no `Change` literal of either module sets `Kind` to anything but `ChangeRetire`;
  - the reviewer IDs and runner identities are used only in `seedEvidence`;
  - the registered `release_evidence.*` IDs are exactly the four above, all mutating.
- **Fault hooks:** `release-evidence-read`, `roster`, and `trace-<call>` / `perf-<call>` for every module call the server makes (propose, approve, withdraw, check, requirement, path, results, gate, record, profile, hardware, active_budget, runs, readiness). The server never calls `Proposal`; seed failure at each step; concurrent reads, commands and `Check` calls under the race detector.
- **Runs:** `go test -race -count=40 -timeout 45m -run Evidence ./apps/api/server`; cross builds linux/amd64, windows/amd64, darwin/arm64; 100% statement coverage of `apps/api/server` and `modules/releasetrace`; mutation testing of the four new server files and `modules/releasetrace/exports.go`, with survivors killed or shown equivalent with a reason.
- **Checks:** `gofmt`, `go vet`, the 1,500-line limit, no comments.
- **Reviews:** independent design and change review; `verification.md`.

# Assumptions and root decisions

Assumptions (not verified in code):
- The P05 spec and planning citations come from the authorized local workspace copy, not this tree.
- A read with roughly 30 module calls per path and profile, plus one `Check` per requirement pair for replacements, stays fast enough at the seeded size; caps bound the output for larger stores.

Root decisions (design review round 1), folded into the body above:
1. Harness reviewers are two module-only sample actors (`sample-evidence-reviewer-1/2`), used in the seed only, never directory users and never accepted from requests; a recorded deviation from the HB-257 seed actor rule, guarded by the AST test like the other system actors (Sample seed, Actors; residuals).
2. `release_evidence.revoke_harness` is dropped: it is single-person and irreversible in this build. The read does not need a revoked flag, so neither module gets a `HarnessRevoked` export and perfgate is unchanged; harness rows read `revocation: not_connected` (Module exports; artifacts; Not connected).
3. Synthetic `tested` states and failing, provisional, over-budget or incomplete load verdicts may be shown on records labelled `origin: sample_seed`; nothing may read ready, active, customer-proven or passed, including a load verdict or load gate status (Read; verification).
4. Console scope: retirement proposals (requirement, profile) with approve and withdraw only. Budgets, budget approval, path activation, customer proof and support decisions stay owner decisions or not connected (Commands; Not connected).
5. The allowed files are accepted, including the `world_test.go` step count and the `evidence.go` / `evidenceload.go` split (Allowed files).

# Design review round 2 clarifications (binding)

- A. The read's `Check` calls for `can_retire` and `replacements` pass a fixed valid reason constant (as `retention.go` does for its release check), because `clean` refuses an invalid reason (`modules/releasetrace/changes.go:108`); a test asserts a seeded requirement's `can_retire` and `replacements` come from `Check` returning nil, not from `ErrInvalid`.
- B. `replacements` excludes requirements already named as the replacement by a pending retirement in the store; approval still re-checks and answers `invalid_stage` if the state changed.
- C. Building `replacements` costs one `Check` per pair of stored requirements; stored requirements come only from the seed, so this stays small. A future import of requirements must revisit this cost.

# Planning cross-check amendments (binding, 2026-11-17)

A read-only audit of this plan against `docs/ui/screens/P05/spec.md` and `docs/planning/v1.16` found these gaps; they are part of the plan.

- **Trace.** This ticket serves P05 requirement IDs HB-04, HB-19, HB-24, HB-25, HB-26, HB-38, HB-39, HB-40, HB-41, AUD-DB-01 and AUD-DB-09. verification.md has a "Planning cross-check" section listing each as covered, deferred or not connected, with the reason.
- **AUD-DB-01** (standard XLSX common release scope) is deferred: this ticket supplies only the gate side over sample requirements (`common_missing`, the `common_needs_replacement` refusal, `replacements` and the `no_replacement` blocker, templates only for problem-free paths). Corrected at change review round 2; it first read "covered".
- **AUD-DB-09** (an event test matrix per commitment profile, with unsupported events recorded rather than passed) is not connected in this build; listed in the not-connected section and the residuals.
- **Reason not run.** Each test row without a result for the candidate build carries `not_run_reason: "no_result_recorded"`; a row whose latest result is not counted carries `not_counted_reason` from the module (for example the build predating the implementation). The spec's evidence drawer reason (`P05/spec.md:37`) reads from these fields; never `null` without a reason.
- **Search and filters** (provider, contract or role, version, issuance path, evidence state; `P05/spec.md:61-63`) are not built; residual next to "no cursor paging".
- **Concurrent collect, close and delete load** (`P05/spec.md:41`, SCALE-11) has no profile in the seed or the modules; residual.
