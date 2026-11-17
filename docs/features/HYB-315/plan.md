---
type: ImplementationPlan
title: Live P05 support combinations and release evidence screen in the console service mode, bound to the HYB-290 release evidence API
description: Binds the P05 support combinations and release evidence screen in the console service mode to the HYB-290 read and its four retirement commands, showing every record truthfully as sample evidence with nothing reading ready, active, customer-proven or passed.
status: final
tags: [HYB-315, pre-launch, release-evidence, console, P05]
sources:
  - resource: ../HYB-290/plan.md
  - resource: ../HYB-290/verification.md
  - resource: ../HB-289/plan.md
  - resource: ../HB-289/verification.md
  - resource: ../HB-287/plan.md
  - resource: ../HB-287/verification.md
  - resource: ../HB-257/plan.md
  - resource: ../../reviews/HYB-291-planning-cross-check/report.md
  - resource: ../../reviews/HYB-301-trace-classification/report.md
---

# Scope and ownership

[HYB-315](https://linear.app/hybrid-billing/issue/HYB-315) binds P05 (support combinations and release evidence, path `/release-evidence`, `screens.json` P05) to the [HYB-290](../HYB-290/plan.md) API ([verification](../HYB-290/verification.md)). Base: main `19d4ec1`. Branch: `feat/HYB-315-release-evidence-screen`. Worktree: `/private/tmp/hb315`.

It follows the merged [HB-289 plan](../HB-289/plan.md) and [verification](../HB-289/verification.md) (P04), the [HB-287](../HB-287/plan.md) P03 precedent, and the code those plans produced:
- `apps/console/src/modules/administration/pages/retention/` (host, per-tab views, shared command form, model, copy, codes, format, CSS);
- `apps/console/src/service/retention.ts`, `retention.sample.ts` and `retention.test.ts`;
- the P04 branch in `apps/console/src/app/screen-groups/operations.tsx:62-67`.

Every rule HB-285, HB-287 and HB-289 ended with is part of this design from the start (see "Rules carried").

The P05 design is `docs/ui/screens/P05/spec.md`, `design.json` and `wireframe.svg`. They are in the authorized local workspace and not in this repository tree, so their line citations refer to that copy.

**Dependency on HYB-293 (binding).** [HYB-293](https://linear.app/hybrid-billing/issue/HYB-293) (step-up reauthentication) is not merged at this base. It changes `apps/console/src/service/admin.ts`, adds `useAdminClient()` in `service/context.ts`, a shared reauthentication dialog and the `reauthNotDone` / `reauthRefused` copy, and makes every registered mutating command, including the four `release_evidence.*` commands, require a recent authentication (HYB-293 `plan.md:21`). **Implementation of this ticket starts only after HYB-293 merges.** The branch is then rebased on that main, and this plan's "Step-up" section is checked against the merged code before the first line of screen code. If the merged contract differs from the HYB-293 plan cited here, the difference goes to root before implementation, not into a silent adaptation.

**Ownership**
- Root (Claude Code) owns the plan, integration, browser QA and verification.
  - Implementation may be delegated to one writer in this worktree. That writer is the only writer of the allowed files.
- The `ui-designer` role (Claude CLI, read-only) writes the visual specification before implementation and compares the final renders. It confirms or replaces the external references.
  - The specification is committed with this plan as [`visual-spec.md`](visual-spec.md). It is not owner acceptance.
- The tech-writer role (AI, read-only) reviews the EN/KO/AR terminology in context (`.agents/rules/product-terminology.md`).
- An independent read-only reviewer reviews the design (this plan) and the change. Focus:
  - nothing reads as a release, an activation, a supported combination, a customer proof, an approved load budget or a passed measurement;
  - unreachable positive values (`ready: true`, path `active`, `customer_proven`, verdict or gate status `passed`) fail closed and are never shown as positive;
  - a profile retirement that turns a failing load gate into "not run" is explained, so it does not read as a fixed failure;
  - a requirement retirement states its permanence and its gate effect, with and without a replacement;
  - unknown, not connected, not evaluated (no candidate), not run, not counted and real zero are never confused;
  - commands are offered only where the read allows them, and parsing fails closed;
  - the step-up path keeps the attempt and never duplicates a proposal.
- Machine: mac_mini.

**Allowed files**
- `apps/console/src/service/release-evidence.ts` (types and hardened parsers), `release-evidence.test.ts`, and the fixture `release-evidence.sample.ts`.
  - The fixture is a non-test module imported by both test files. It is not imported from a test file (HB-289 change review round 1: a fixture imported from a test file registered its tests twice).
  - The file name is checked free before creation (`ls apps/console/src/service`; no `release-evidence.*` and no `evidence.*` exists at this base).
  - It may import `count` and `hint` from `./integrations` read-only (`integrations.ts:173-175`, `184-188`).
- `apps/console/src/service/admin.ts`:
  - `releaseEvidence(request = attempt().request)`, a `GET /release-evidence` read placed next to `retention()` (`admin.ts:545-547`). The optional request ID lets the screen show the ID it sent (Shared frame, error detail);
  - the four command IDs `release_evidence.approve`, `release_evidence.withdraw`, `release_evidence.retire_requirement` and `release_evidence.retire_profile` in `CommandId` (`admin.ts:246-309`).
  - Nothing else in `admin.ts`. The step-up client is HYB-293's.
- `apps/console/src/modules/administration/pages/release-evidence/`:
  - new `Live.tsx` (host), `LiveMatrix.tsx`, `LiveRequirements.tsx`, `LiveArtifacts.tsx`, `LiveLoad.tsx`, `LiveDecisions.tsx`;
  - new `LiveCommand.tsx` (the shared command form, confirm and row note) and `LiveRetireForm.tsx` (the reason and replacement fields);
  - new `live-model.ts`, `live-model.test.ts`, `live-copy.ts`, `live-codes.ts` (vocabularies and blocker sentences), `live-format.tsx` and `live.css`.
  - Any file nearing 1,500 lines is split by responsibility (`.agents/rules/code-structure.md`).
  - The existing preview `Screen.tsx` stays unchanged for the unconnected mode.
- `apps/console/src/app/screen-groups/operations.tsx`: the live branch for P05.
- `apps/console/package.json`: the test list (`src/service/release-evidence.test.ts`, `src/modules/administration/pages/release-evidence/live-model.test.ts`).
- `docs/features/HYB-315/*`.
- No API, module, shell, `screens.json`, P01–P04 or M01–M05 file changes.
  - P05 imports only these read-only helpers: `count`, `hint` (`service/integrations.ts`), `useWide` (`identity/pages/users/Live.tsx:21`), `useAdminClient` (HYB-293, `service/context.ts`), `c` and `retryPhrase` (`service/copy.ts`, including HYB-293's `reauthText`), and `identity/pages/users/live.css`.
  - P05's own `live-format.tsx` re-creates `Free`, `Who`, `Stamp`, `ltr`, `Known`, `Num`, `ListOr`, `Template`, `MoreLine` and `href` with `p05-` classes. P04's versions emit `p04-` classes and build `#/retention?` links (`retention/live-format.tsx:44`, `61`, `66`, `89`), so they are not imported.
  - P05 keeps its own attempt store and command form, so P01–P05 attempts never mix.

**Non-goals**
- No release, activation, deactivation, support decision, customer proof, evidence linkage, harness binding or revocation, result or run recording, budget or budget approval: none has a command (HYB-290 `plan.md:281-295`).
- No requirement, implementation, build, path, reconciliation, profile or hardware proposal from the console.
- No server search, filters, stable server sort, cursor paging or fixed snapshot (P05 `spec.md:61-63`, `103`); the API has none (HYB-290 `verification.md:56`, tracked in HYB-304 and HYB-305).
- No links to S02, P06, I02, I03 or U03 (P05 `spec.md:82-86`): none of them has a live binding.
- No change to the shell tab bar's files, the shared `bdi` style or the review control. P05 changes the tab bar's layout below 768px only through one scoped rule in its own `live.css`: `.planning-main:has(.p05-live) .planning-tabs` (two columns, with the fifth tab spanning both).
- `docs/ui/screen-binding-register.md` is not updated; the catch-up is HYB-295 (HB-288 `plan.md:300`, HYB-290 `verification.md:59`).

# Observed baseline (facts from code at `19d4ec1`)

## Console

- **Live branch.** `operations.tsx:20` sets `live` for P01–P04 only. P02, P03 and P04 return their live wrappers at `:50-67`. P01 returns at `:68-73`, and the preview chooser at `:74-89` maps P05 to the preview `ReleaseEvidence`. P05 in service mode therefore renders the preview today.
- **Preview.** `pages/release-evidence/Screen.tsx` (162 lines):
  - a `Ledger` whose title changes per view (`:52-99`);
  - a `Draft` form "Review evidence linkage" with combination, requirement, artifact and reason fields (`:8-46`, `:100-108`). No command exists for it.
  - panels saying requirement traces and execution artifacts are not connected (`:3-7`), no measurements are connected (`:140-146`), and mock success is not live evidence (`:151-156`).
  - Its EN column "Customer validation" (`:88`) names the concept the API calls customer proof.
- **Client.** `adminClient(csrf)` (`admin.ts:354`) posts `{request_id, target_id, idempotency_key, expected_revision, rationale, body}` (`:392-404`).
  - Failure classes come from `codes` (`:317-333`). `failureOf` maps `internal_error` to `partial` only with an operation ID, otherwise to `unavailable` (`:335-342`).
  - The P04 host builds the client with `useMemo(() => adminClient(session.csrf))` (`retention/Live.tsx:49`). HYB-293 replaces this with `useAdminClient()` in twelve files (HYB-293 `plan.md:56-59`, `398-403`).
- **Helpers.** `count` accepts safe non-negative integers only (`integrations.ts:173-175`). `hint(allowed, blocked)` allows only `true` with `""`; a missing or non-string blocker is `missing` (`:184-188`).
- **Remount.** `RemainingScreens` keys the screen by screen, locale, view and scope parameters, so every tab change remounts the host and reloads the read (HB-289 `plan.md:204`).

## API (`GET /api/v1/release-evidence`, route `boundary.go:40`)

- **Admission.** The read is admin only through `admitted` under the domain read lock (`evidence.go:739-746`).
  - Another workspace than the boot workspace gets `{revision, now, state: "not_configured"}` and nothing else (`evidence.go:748-750`).
  - Any module or roster error makes the whole read `dependency_unavailable` (`evidence.go:752-755`). The console maps it to `busy`.
- **Page.** `{revision, now, state: "configured", candidate, matrix, requirements, artifacts, load, decisions}` (`evidence.go:241-251`, `718-736`).
- **Origin.** Every section carries `origin`: `sample_seed` when the store was seeded, else `empty` (`evidence.go:23-24`, `715-717`).
- **Candidate.** `{version, build, release: "not_connected"}` or `null` (`evidence.go:61-65`, `719-722`). Without a candidate:
  - per-path `columns.test`, `columns.customer_proof`, `columns.failing`, `categories`, `blockers` and `last_observed` are `null`, and requirement references have `state: null` (`evidence.go:391-405`);
  - per-requirement `tests` is `null` and `states` is `{}` (`evidence.go:512-523`);
  - the requirements `gate` is `null` (`evidence.go:542-544`);
  - load `readiness` and every profile `status` are `null` (`evidenceload.go:197-212`, `160-162`).
- **Lists.** `{total, more, rows}`; `more` is `total > rows` (`evidence.go:31-48`). Caps: 50 per list (`:18`), 10 build states (`:19`), 20 replacements (`:20`), 100 gate problems (`:21`), 12 reconciliation periods (`:22`), 20 runs per profile (`evidenceload.go:10`).
- **People.** `{id, login, directory}`; `login: null, directory: false` for an ID the directory does not know, such as the sample harness reviewers; `null` for no person (`evidence.go:55-59`, `262-271`).

### `matrix.paths[]` (`evidence.go:97-116`, `349-421`)

| Field | Meaning |
| --- | --- |
| `id`, `provider`, `contract`, `role`, `version`, `issuance` | The combination (strings) |
| `template` | ERP or file template ID, or `null` |
| `state` | `designed` or `active` (`releasetrace` path states); only `ChangeActivate` makes a path active, and no API path uses it |
| `build` | The path's active build, or `null` |
| `requirements` | `{total, more, rows: [{label: "namespace/id", target: "trace-requirement-<n>" or null, state: requirement state at the candidate build or null}]}` |
| `columns` | `design {count, of}`, `implementation {count, of}`, `test {count, of} or null`, `customer_proof {count, of} or null`, `failing` integer or `null` |
| `categories` | All six categories (`amount, approval, bulk, duplicate, isolation, recovery`) to `true` / `false`, or `null` |
| `reconciliations` | Newest first, at most 12: `{period: "YYYY-MM", matched, differences}` |
| `last_observed` | The latest result time at the candidate build for the path's tests, or `null` |
| `blockers` | `[{problem, subject}]` from the gate check of the path with its requirements, or `null` |
| `activation`, `support_decision`, `customer_proof_intake` | `not_connected` |

Problem codes are `unknown`, `common_unregistered`, `retired`, `common_missing`, `not_tested`, `blocking`, `path_inactive`, `path_build`, `path_reconciliation`, `path_requirement`, `path_evidence` (`modules/releasetrace/types.go:55-65`). Subjects are `namespace/id`, a path ID, `path>namespace/id` for `path_requirement`, `namespace/id>namespace/id` for a retired requirement with a replacement, or `""` for `common_unregistered` (`gate.go:9-16`, `62`, `82`).

### `requirements` (`evidence.go:139-175`, `498-558`)

- `requirements.rows[]`: `target` (`trace-requirement-<n>`, the 1-based registration position), `label`, `namespace`, `id`, `source`, `owner` (a module ID, not a person), `interfaces[]`, `common`, `blocking`, `design` (64-hex digest), `commit` (or `null`), `retired`, `replacement` (label or `null`).
- `states`: build to state for at most the newest 10 stored builds, with `builds_more`. States are `designed`, `implemented`, `failing`, `tested`, `customer_proven` (`types.go:46-50`).
- `tests` (at the candidate build) is `{total, more, rows}` or `null`. Each row is `{id, category (or null), counted, latest_outcome: "pass" | "fail" | null, passes, failures, not_run_reason, not_counted_reason}`.
  - `not_run_reason` is `no_result_recorded` exactly when there is no result (`evidence.go:444-445`).
  - `not_counted_reason` is set only when a result exists and is not counted: `no_implementation`, `build_predates_implementation` or `not_counted_by_module` (`evidence.go:429-442`, `evidenceseed.go:29-32`, `169-176`).
- `retire`: the pending retirement proposal number or `null`.
- `replacements`: `{total, more, rows: [{target, label}]}`, the requirements the module would accept as a replacement now, at most 20. Requirements already named as the replacement in a pending retirement are excluded (`evidence.go:460-476`).
- `replacement_required`, `can_retire`, `retire_blocked`. Blockers in order (`evidence.go:478-496`):
  - `retired`, then `already_proposed`, then `no_eligible_approver`;
  - then the module check without a replacement. `common_needs_replacement` gives `replacement_required: true`, with `can_retire: true` when replacements exist, else blocker `no_replacement`. Any other module reason, or `not_found`, is the blocker.
- `gate` (or `null`): `{ready, problems: {total, more, rows}, retired_blocking: [labels], templates: [IDs], release: "not_connected"}` (`evidence.go:163-169`, `545-557`).

### `artifacts` (`evidence.go:177-212`, `560-605`)

- `builds` newest first: `{build, commit, order}`.
- `harnesses`: `{module: "release_trace" | "perf_gate", id, revocation: "not_connected"}`.
- `results` newest first: `{namespace, test, build, counted, latest_outcome: "pass" | "fail", evidence (64-hex), harness, at, passes, failures, not_counted_reason}`.
- `files`, `logs`, `environment`, `content`: `not_connected`.

### `load` (`evidenceload.go:15-108`, `150-224`)

- `hardware`: `{id, cpus, memory_mib, disk_gib, storage: "ssd" | "hdd" | "network", spec (64-hex)}`.
- `profiles[]`:
  - `target` (`perf-profile:<id>`), `id`, `gate` (`e2e`, `daily_inflow`, `monthly_close`, `interrupt_resume`, `catch_up`), `axes` (13 integers), `dataset`, `expected` (digests), `retired`;
  - `active_budget`: `{id, state: "proposed" | "approved", hardware, limits: {metric: value}, proposer, approver, approval_by, approved_by, approved_at}` or `null`;
  - `status`: the profile's status in the readiness at the candidate build, or `null`. Values are a verdict or `not_run`;
  - `runs`: newest first, at most 20, each `{number, build, hardware, rows, duration_s, mismatches, interrupted, metrics: nine keys with integer or null, harness, at, verdict}`;
  - `retire` (pending number or `null`), `can_retire`, `retire_blocked`: `retired`, `already_proposed`, `no_eligible_approver`, `target_too_long` (an ID over 115 bytes), in that order (`evidenceload.go:172-182`).
- `readiness` (or `null`): `{ready, gates: [{gate, status, profiles: [{profile, status, run, failures}], retired: [...]}]}`.
- `measurement`, `planning_targets`: `not_connected`.
- Verdicts: `other_data`, `incomplete`, `incorrect`, `revoked`, `self_verified`, `missing_metric`, `unbudgeted`, `other_hardware`, `over_budget`, `predates_budget`, `provisional`, `passed`. Gate statuses: `not_run`, `failing`, `provisional`, `passed`. Metrics: `max_rss_bytes`, `db_io_bytes`, `wal_bytes`, `lock_wait_ms`, `p95_ms`, `p99_ms`, `throughput_rows_s`, `completion_ms`, `resume_ms` (`modules/perfgate/types.go:42-71`).

### `decisions` (`evidence.go:214-239`, `607-701`)

- `proposals`: both modules, newest first by store order, at most 50.
  - Each row: `module`, `number`, `target` (`trace-proposal-<n>` or `perf-proposal-<n>`), `kind`, `state` (`pending`, `applied`, `withdrawn`, `store_retired`), `created_at`, `proposer`, `approver`, `rationale`, `closed_by`, `closing_rationale`, `summary`, `can_approve`, `approve_blocked`, `can_withdraw`, `withdraw_blocked`.
  - Summaries by kind (`evidence.go:607-636`, `evidenceload.go:226-243`):
    - release trace: `requirement {label}`, `retire {label, replacement, reason}`, `implemented {label, commit}`, `build {build, commit}`, `harness {harness}`, `path {path}`, `reconcile {path, period, matched, differences}`, `proof {label, build, period}`, `activate {path, build}`;
    - load: `profile {profile, gate}`, `hardware {hardware}`, `budget {budget, profile, hardware}`, `approval {budget}`, `harness {harness}`, `retire {profile}`.
- Approve blockers in order (`evidence.go:638-667`):
  - a closed row: `not_pending` (or `store_retired`) for both actions;
  - `not_console_kind` for every kind but `retire`;
  - `own_proposal`;
  - for release trace, the module check of the change with the reader as approver: `retired`, `weaker_replacement`, `common_needs_replacement`, `not_found`;
  - for load, `retired` when the profile is already retired.
- Withdraw blockers: `not_pending`, `store_retired`, `not_proposer`. A pending non-retire proposal is withdrawable by its proposer (`evidence.go:646-652`). Such a proposal exists only by injection.
- `support_decisions`, `activation_review`: `not_connected`; `ledger: "in_memory"`.

### Commands (`evidencecmds.go`)

| Command | Target | Body fields | Server prechecks and module refusals |
| --- | --- | --- | --- |
| `release_evidence.approve` | `trace-proposal-<n>` or `perf-proposal-<n>` | none | Not pending, or a kind other than `retire`: `invalid_stage` without a module call (`:125-134`). Self-approval: `prerequisite_failed`. A retirement whose replacement became weaker, or a requirement or profile already retired: `invalid_stage`. |
| `release_evidence.withdraw` | same | none | Not pending: `invalid_stage` (`:136-145`). Not the proposer: `prerequisite_failed`. |
| `release_evidence.retire_requirement` | `trace-requirement-<n>` | `reason` (text, required), `replacement` (ID, optional, a `trace-requirement-<m>` target) (`:225-226`) | `reason` failing `releasetrace.ValidText`: `invalid_request` [`reason`] (`:149-152`). An unknown replacement, or the target itself: `invalid_request` [`replacement`] (`:153-160`). Already retired, or a pending retirement from anyone: `invalid_stage` (`:165-171`). `common_needs_replacement`, `weaker_replacement`: `invalid_stage`. |
| `release_evidence.retire_profile` | `perf-profile:<id>` | none (`:227`) | Retired, or a pending retirement from anyone: `invalid_stage` (`:177-189`). |

- **Rationale.** Every command takes a rationale. Over 256 bytes is a fieldless `invalid_request` (`evidencecmds.go:193-195`).
- **Module text rule.** `reason` must be 1 to 200 bytes of valid UTF-8 with no C0 control, DEL or C1 control (`modules/releasetrace/common.go:28-38`). The server does not trim it.
- **Results.** `business_state` is the proposal state; `fields` are `module`, `proposal` (decimal) and `target` (`evidencecmds.go:120-123`).
- **Error mapping** (`evidencecmds.go:26-66`):
  - `self_approval` and `not_proposer` → `prerequisite_failed`; `ErrLimit` → `prerequisite_failed`;
  - every other refusal, `ErrNotFound` and `ErrConflict` → `invalid_stage`;
  - `ErrInvalid` and actor-kind refusals → `internal_error`.
  - Only `invalid_request` carries fields.
- **Permission.** Every command needs `AdministerInstallation` on the installation kind (`:220-221`), a recorded deviation from P05's separate permissions (HYB-290 `plan.md:240`).
- **Resolve** reads only the store and only in the boot workspace (`:68-118`). Anything else is `permission_denied`.
- **Replay.** Stored records are never removed and closed proposals leave only beyond 200 kept entries, so a resend with the same key and body replays the stored result (HYB-290 `plan.md:247`).
- **Permanence.** Neither module has an un-retire change (`releasetrace/types.go:21-29`, `perfgate/types.go:21-26`), so an applied retirement cannot be reversed in this build.

### Gate effects of a retirement (module facts)

- A requirement retired **without** a replacement is listed in `retired_blocking` of every gate check, selected or not (`gate.go:44-47`).
- A retired requirement selected in the candidate adds a `retired` problem and is no longer checked for `blocking` (`gate.go:33-34`, `44-56`).
- A retired load profile leaves the gate's live profiles. Its gate status can move from `failing` to `not_run` (HYB-290 `plan.md:327`: after the seeded `sample-month` retirement, `monthly_close` becomes `not_run`).

## Sample seed (who did what)

`--seed sample`. A1 is `sample-admin` (`admin@example.invalid`), A2 is `sample-admin-2` (`admin2@example.invalid`), and the member is `sample-member` (`member@example.invalid`). Seeded command reasons read "Sample data". The expected read is asserted in `evidence_api_test.go:225-360`.

- **Candidate:** `sample-candidate-1` at build `0.0.2`, release `not_connected`.
- **Paths:**
  - `sample-path-api`: three requirements; blockers include `path_evidence` and `path_inactive`.
  - `sample-path-xlsx` (template `sample-template-xlsx`): design 2 of 2, implementation 1 of 2, test 1 of 2, customer proof 0 of 2, failing 0. Blockers: `not_tested` `sample/sample-erp-template`, `path_evidence`, `path_inactive`, `path_reconciliation`. One reconciliation for the previous month: not matched, 3 differences (`evidence_api_test.go:225-230`).
- **Requirements** at `0.0.2` (`evidence_api_test.go:235-244`):
  - `trace-requirement-1` `sample-common-xlsx`: tested; `replacement_required`, no replacements, blocker `no_replacement`;
  - `-2` `sample-approval`: tested; can retire; replacements `sample-recovery`, `sample-erp-template`, `sample-api-export`, `sample-legacy-report`;
  - `-3` `sample-recovery`: failing (blocking); can retire;
  - `-4` `sample-erp-template`: designed; can retire; `t-erp-template` reads `not_run_reason: no_result_recorded`;
  - `-5` `sample-api-export`: implemented; can retire;
  - `-6` `sample-legacy-report`: implemented; `already_proposed` with `retire: 19`.
- **Gate:** `ready: false`, five problems: `blocking` `sample/sample-recovery`, `not_tested` ×2, `path_evidence` and `path_inactive` on `sample-path-api`. `retired_blocking` and `templates` are empty.
- **Artifacts:** builds `0.0.2` (order 2) and `0.0.1` (order 1); harnesses `sample-trace-harness` and `sample-perf-harness`; seven results. The `t-xlsx-amount@0.0.1` result reads `counted: false` with `not_counted_reason: build_predates_implementation`.
- **Load:**
  - hardware `sample-hw-small`;
  - `sample-e2e` provisional (budget proposed);
  - `sample-daily` over budget (throughput 13 below 20, p99 1,500 above 1,000);
  - `sample-month` incomplete, `already_proposed` with `retire: 9`;
  - `sample-resume` not run, no runs.
  - Readiness: `e2e` provisional, `daily_inflow` failing, `monthly_close` failing, `interrupt_resume` not run, `catch_up` not run; `ready: false`.
- **Decisions:** 28 proposals.
  - `perf-proposal-9`, a pending profile retirement **proposed by A2**: A1 sees Approve and `not_proposer`; A2 sees `own_proposal` and Withdraw.
  - `trace-proposal-19`, a pending requirement retirement of `sample-legacy-report` without a replacement, reason "Sample scope change", **proposed by A1**: A1 sees `own_proposal` and Withdraw; A2 sees Approve.
  - The harness proposals name `sample-evidence-reviewer-1` and `-2` with `login: null, directory: false`.
- **After A2 approves #19:** `sample-legacy-report` is retired; the gate keeps `ready: false` and its five problems, and `retired_blocking` gains `sample/sample-legacy-report` (HYB-290 `plan.md:326`).
- **After A1 approves perf #9:** `sample-month` is retired; readiness `monthly_close` reads `not_run` (HYB-290 `plan.md:327`).

# Design

The design follows the P04, P03, P02 and P01 live patterns:
- full viewport width; neutral rules and tags, no coloured edge strips (`.agents/rules/service-screen-style.md`);
- EN, KO and AR copy;
- the shared failure box and the rationale field (UTF-8 bytes, 256), with an error summary;
- layout switches by container size on `.p05-live`; `useWide` only where a component needs a JavaScript width decision;
- dates through `Intl` in UTC, kept whole;
- IDs, digests, commits, builds, labels and targets in `<bdi dir=ltr>` monospace;
- free text (rationales, retirement reasons) in `<span dir="auto" class="p05-free">` with `unicode-bidi: isolate`, never `<bdi>`, because the global `bdi { direction: ltr }` rule forces Arabic text left-to-right (HB-289 `plan.md:186`).

Classes use the `p05-` prefix.

## Truthfulness rules (owner directions, binding)

1. **Never ready, active, customer-proven or passed.** The screen has no copy that states, as a positive fact, that a gate is ready, a path is active, a requirement is customer-proven, a load verdict or load gate passed, a combination is supported, or anything is released.
   - **Unreachable values fail closed.** These cannot occur through the API (HYB-290 `plan.md:320`). If the parser meets one, it renders a neutral "Unexpected value from the server ({code}). This build cannot record it, so it is not accepted here." in place of the state word. `{code}` is a neutral field locator (`gate.ready`, `readiness.ready`, `path.state`, `path.build`, `path.blockers`, `requirement.state`, `run.verdict`, `gate.status`, `budget.state`, `budget.approval`, `customer_proof.count`, `gate.templates`, `proposal.state`), never the server value:
     - `gate.ready: true` or `readiness.ready: true`;
     - a path `state: "active"` or a non-null path `build`;
     - a requirement state `customer_proven`;
     - a verdict or gate status `passed`;
     - a budget `state: "approved"`;
     - a budget in `state: "proposed"` with a non-null `approval_by`, `approved_by` or `approved_at`. Only an applied budget approval sets them (`modules/perfgate/changes.go:228`, `evidenceload.go:47-49`, `130-132`), and budget approval is not connected;
     - a path `columns.customer_proof.count` above 0. Customer proof intake is not connected, so the count can only be 0;
     - a non-empty `gate.templates`. A template is listed only for a path with no problems (`gate.go:88-89`), and a path without problems must be active;
     - a path whose `blockers` is `[]` (an empty list, not `null`). Every path that is not active carries `path_inactive` (`gate.go:72-73`);
     - a gate problem or path blocker `path_build`. It is raised only for an active path (`gate.go:72-75`);
     - a proposal of kind `activate`, `proof` (release trace) or `approval` (load) in state `applied`. None of these is proposed by the seed or any command (HYB-290 `plan.md:35`, `36`, `39`: no activation, no customer proof and no budget approval anywhere).
   - In each case the row, cell or block is treated as not ready, not active, not proven, not approved and not passed for every other purpose. The positive value itself is never rendered, not even beside the unexpected sentence.
     - A customer proof count above 0 reads the unexpected sentence in place of the count.
     - Templates read the unexpected sentence in place of the list.
     - `blockers: []` reads the unexpected sentence in place of the "no problem" sentence.
     - `path_build` reads the unexpected sentence in place of its problem words.
     - An applied activate, proof or approval proposal reads the unexpected sentence in place of its state and closing facts.
   - A model test asserts each value in this list, and the render pass sweeps the page text.
2. **Sample origin.** With `origin: sample_seed`, every tab shows "Sample records. Every record here was created by the sample data. None is a release, an activation, a customer proof or a measurement of real load." With `origin: empty`: "No records. The release trace and load registers are empty in this installation." An unknown origin reads "Record origin: Unknown (code)".
3. **Test outcomes on sample records** may be shown as recorded facts (HYB-290 root decision 3), worded as "Pass recorded" and "Fail recorded", never "Passed" or "Failed" as a status. The requirement state `tested` reads "Tested at this build". The suffix " (sample results)" is added only when the section's `origin` is `sample_seed`.
4. **Not connected** reads "Not connected", never "None", "0" or blank.
5. **Not evaluated.** A value that is `null` because no candidate exists reads "Not evaluated: no release candidate is selected", never "Unknown" and never 0.
6. **Customer proof** (root decision on visual spec §14.4). Each cell shows "{count} of {of}" with the short muted line "Intake not connected". The full sentence "Customer proof intake is not connected, so this count cannot rise in this build." appears once as a 14px muted note directly under the matrix lead, before the table, and again in the evidence detail panel. The column header carries no sub-line (amended 2026-11-18 after the rendered review: the header sub-line wrapped to about 8 lines in a 10% column). A 0 is shown as 0; a count above 0 is rule 1.
7. **Provisional values.** This screen introduces no provisional value of its own. Caps, byte limits and the planning targets are server rules or planning statements, not installation settings.
   - The owner decisions P05 leaves open are shown as open in the frame note: the first supported combinations, performance hardware and latency targets, evidence validity and activation accountability (`spec.md:137-139`).
   - Budgets and limits in the read are sample records, not defaults.

## Design decisions and alternatives

1. **Approve and Withdraw only on the decisions tab.** Requirement and profile rows with a pending retirement show "Retirement proposal #{n} is waiting for approval." with a link to `view=decisions&proposal=trace-<n>` (or `perf-<n>`).
   - Rejected: a second Approve button on the requirement or profile row. Two forms for one command and target would share one attempt store entry from two places.
2. **Non-retire pending proposals are read-only.** The API lets a proposer withdraw a pending non-retire proposal (`evidence.go:646-652`), but such proposals exist only by injection and their whole lifecycle is not connected. The screen offers no command for any kind but `retire`, even when `can_withdraw` is true. This is the P04 plan-proposal rule (HB-289 decision 4). The row reads "Proposals of this kind are recorded by the sample data. Approving or withdrawing them is not available here."
3. **Replacement is an explicit choice.** The retirement form has a native `<select>` with the placeholder "Choose a replacement" (value `""`) and no default selection.
   - When `replacement_required` is false, the first real option is "No replacement".
   - When `replacement_required` is true, "No replacement" is not offered.
   - Options are the `replacements.rows` labels, in the read order.
   - Rejected: defaulting to "No replacement". That silently chooses the variant that lists the requirement as retired without replacement in every gate check.
4. **The evidence detail (`spec.md:35-37`) follows the spec: beside the list on desktop, a separate step on narrow widths** (`spec.md:29`, `109`, `111`; root decision 6). It applies to the matrix (`path=`) and the requirements tab (`requirement=`). The visual specification §4 is binding for its layout.
   - **Mode by available width, not window width.**
     - `live-format.tsx` provides `useAvailableWidth(ref)`, a `ResizeObserver` on `.p05-live`. The split threshold is **940px** of `.p05-live` (root decision on visual spec §4.1 and §14.5), the constant `splitWidth = 940` in `live-model.ts`.
     - The CSS rule is `@container (min-width: 940px)` on `.p05-live`. The JS and CSS use the same number, and a model test reads both (the CSS file text) and asserts they match.
     - **Rounding.** The JS compares `Math.floor(contentRect.width) >= 940`. `floor(w) >= 940` holds exactly when `w >= 940`, which is what the CSS `min-width` query tests, so the two cannot disagree at fractional widths.
     - **Before the first measurement.** The host measures `.p05-live` with `getBoundingClientRect()` in a layout effect before the first paint, so no frame renders in the wrong mode. A width of 0 (detached or hidden) counts as below the threshold, so step mode applies.
     - `useWide` (a viewport check) is not used for this decision.
     - Measured P04 widths (HB-289 `verification.md:27`): about 1120px at a 1440 window and about 960px at 1280 both give side by side; about 358px at 390 gives the step.
   - **Side by side (`.p05-live` ≥ 940px) with a selection.**
     - `.p05-split[data-open="true"]` is a two-column grid: the list region `.p05-list` and `aside.p05-detail` at `clamp(360px, 38%, 460px)` (visual specification §4.2), labelled by its heading.
     - **Reduced list.** Beside the open panel the list renders a reduced column set chosen by the open state, not by CSS hiding (visual specification §4.3):
       - matrix: Combination (path ID, provider and issuance sub-line, path state, Evidence link), Test, Customer proof, and Blockers:
         - "{n} problems" for a non-empty list, with the count chosen through `Intl.PluralRules`;
         - "Not evaluated" for `null`;
         - the rule 1 unexpected sentence for `[]`.
         - It never reads "0 problems".
       - requirements: Requirement (label, tags, Evidence link), State at the candidate build, Tests summary.
     - **The reduced table never stacks.** `.p05-split[data-open="true"] .p05-table.p05-reduced` stays a table at every width at which side-by-side mode applies. No `@container` rule stacks it, and only step mode replaces it (visual specification §4.3, amended 2026-11-18).
     - **The panel repeats every column hidden from the reduced list** (visual specification §14.2), as listed under each tab's "Evidence detail".
     - The list stays visible; the selected row keeps its place and is marked `aria-current`, with a background tint and a visible "Selected" tag (`aria-hidden`; visual specification §9).
     - **Sticky panel with a height cap** (visual specification §4.2):
       - `position: sticky` with `max-block-size: calc(100dvh - 32px)` and `overflow-y: auto`;
       - the aside is the scroll container and has `tabindex="0"`, so a keyboard user can scroll it with the arrow keys;
       - its focus ring is inset (`outline-offset: -2px`);
       - the top offset accounts for the shell header if that header is sticky (unverified; checked in the render pass).
     - Two tables are never placed side by side: the panel holds fact lists and stacked test blocks.
   - **Side by side without a selection.** The grid is one column and the list uses the full width with its full column set. **No empty panel is drawn.** A one-line muted hint under the tab lead reads "Select a combination to see its evidence." (matrix) or "Select a requirement to see its evidence." (requirements).
   - **Step (`.p05-live` < 940px) with a selection** (visual specification §4.4). Only these are rendered:
     - the shared frame;
     - the tab heading;
     - a "Back to the list" link;
     - the detail (`section.p05-step`).
     - The tab lead, hint, gate block, build comparison, list and not-connected action list are not rendered (removed, not hidden).
   - **Step without a selection:** the normal list view.
   - **Opening and closing.** Each row has an "Evidence" link (`href` with `path=` or `requirement=`; `aria-controls` the panel in side-by-side mode). The panel has "Close" and the step has "Back to the list". Both link to the same view without the selection.
   - **Keyboard and focus.**
     - Activating "Evidence" moves focus to the detail heading (`tabindex=-1`).
     - "Close", "Back to the list" and Escape inside the detail return focus to that row's "Evidence" link. Its ID is kept in a ref; when the row is gone, focus goes to the tab heading.
     - A selection that arrives through the URL (a cross link or a reload) opens the detail without moving focus.
     - The panel is not a dialog: it does not trap focus, and Tab moves from the list to the panel in reading order (title → scope → tabs → list → detail → actions, `spec.md:113`).
     - A width change across the threshold keeps the selection. Focus stays on the focused element if it still exists, otherwise moves to the detail heading.
   - **Commands stay with the list.** The panel and the step show the row's retirement state and pending sentence, but no command form, so one command and target has one form. Actions sit in follow rows, never in a column (decision 8).
   - **Content:** see the matrix and requirements tabs ("Evidence detail").
   - Rejected: an inline follow row (an earlier draft). It does not keep the list and detail side by side on desktop as the spec requires.
   - Rejected: the 1080px threshold of the previous draft. It puts a 1280 desktop (about 960px available) into the step meant for mobile (visual specification §4.1).
5. **"Review differences" (`spec.md:72`) is a read-only build comparison** on the requirements tab.
   - Two native selects, "Compare build" and "with build", list the build IDs present in the `states` maps. At most 10 newest builds are read; with `builds_more` the control says "Only the newest 10 builds can be compared."
   - The result is a three-column table: Requirement | State at {from} | State at {to}. It lists only the requirements whose state differs, and reads "No requirement changed state between these builds." when none differ. No arrow or other direction glyph is used in any locale (visual specification §6.2, §14.3).
   - Without a candidate (`states` empty) the control is replaced by "Not evaluated: no release candidate is selected."
   - **Partial (root decision 9).** The control is labelled "Compare builds (partial)". Its hint says: "Shows how requirement states differ between two builds. Comparison with an earlier support decision is not built, because support decisions are not connected." The not-built part is listed in Residuals and the cross-check.
6. **Revoked access** (`spec.md:97`). When a read answers `denied` after an earlier read of this mount succeeded, the screen discards the page data, the forms, the drafts and the attempts for this screen, and shows the spec's revoked state: "Your access has been withdrawn. Details, files and follow-up commands are not shown. Ask a person with access." A first-load `denied` shows the denied block. The API has no separate revoked read, so this covers the visible behaviour only (residual).
   - **Revoked flag lifecycle.**
     - The flag is module-level, scoped like the attempt store (workspace, user, session).
     - **Set** when a read or a command answers `denied` after a successful read in the current scope (any mount).
     - **Effect:** while it is set, a new mount whose first read is `denied` shows the revoked state, not the plain denied block, because access was withdrawn during this session.
     - **Cleared** by the next successful read in the same scope (Reload and Try again stay available), by `forget()` on sign-out, and by a scope change.
     - In a new session, or for another user or workspace, the flag starts unset, so a first-load `denied` shows the denied block.
     - The revoked state keeps only the scope bar (with Reload) and the revoked block; no table, form, panel or "Close" link stays in the DOM (visual specification §5).
7. **The blocked state** (`spec.md:95`) is the matrix lead whenever any path has blockers or is not active: "Execution evidence is missing or the scale or conditions are not met. No support combination can be activated, and activation is not connected in this build."
8. **Actions in follow rows, never a column** (visual specification §6.2, §14.8). On the requirements tab, the opener ("Propose retirement"), the pending sentence or the blocker sentence sits in the row's follow row (`tr.p05-follow`), within the 68ch measure, at every width. On the load tab, they sit in the profile article's `.p05-actions`. The retirement form opens in the same place.

## Host, remount and URL state

- **Live branch.**
  - `operations.tsx:20` adds `p.screenId === "P05"` to `live`.
  - A P05 branch returns before the P01 branch (`:68`). For P05 it returns `<div className="work-domain remaining-security" data-screen-id="P05" data-source-state="live" dir lang key={`P05:${locale}`}>` around `LiveReleaseEvidence`, with props `locale`, `view`, `params`, `session`, `onSignOut`.
  - The P01 branch stays explicit (`p.screenId === "P01"`, already so at `:68`). The render pass asserts `data-screen-id="P05"` on every P05 render.
- **Remount per view.** Every tab change remounts the host and reloads the read. Only the module-level store (attempts, drafts, kept refusals, the revoked flag) survives.
- **URL state:**
  - `view=` is one of `matrix`, `requirements`, `artifacts`, `load`, `decisions` (`screens.json` P05 tabs). Anything else falls back to `matrix`.
  - Optional selections are validated against the read and otherwise ignored. They are derived from the URL on every render, never stored in state. Scroll-into-view happens once per distinct value. The selected row is marked `aria-current`.
    - `path=<id>` (matrix): `[A-Za-z0-9._:-]{1,128}`, a shown path;
    - `requirement=<n>` (requirements): canonical decimal, a shown row whose target is `trace-requirement-<n>`; it also opens that row's evidence detail (decision 4);
    - `path=` likewise opens the matrix evidence detail.
    - **Fact:** the `RevisionScope` key is `JSON.stringify([screenId, locale, view, audience, scope, customer, contract, period, le, ccy, asof])` (`apps/console/src/app/RemainingScreens.tsx:83-92`). The selection parameters are not in it, so opening or closing a detail does not remount the host or reload the read.
    - `from=<build>&to=<build>` (requirements): `^[0-9]+\.[0-9]+\.[0-9]+$`, both present in the `states` keys and different;
    - `build=<id>` (artifacts): a shown build;
    - `profile=<id>` (load): a shown profile;
    - `proposal=trace-<n>` or `proposal=perf-<n>` (decisions): canonical decimal, a shown proposal of that module.
  - **Cross links**, canonical IDs only:
    - a matrix requirement reference with a target → `view=requirements&requirement=<n>`;
    - a requirement or profile row's pending retirement → `view=decisions&proposal=trace-<n>` / `perf-<n>`;
    - a retire proposal's summary label → `view=requirements&requirement=<n>` when exactly one shown requirement has that label; a profile retirement → `view=load&profile=<id>`;
    - an artifacts result row → `view=requirements&requirement=<n>` only when exactly one shown requirement lists that namespace and test ID; otherwise no link.
  - A model test asserts that every href starts with `#/release-evidence?`.
- **Focus and live region.** After a success, the status message (`#p05-notice`, `role=status`) is focused. A refusal focuses the row note. Cancel returns focus to the opener. The evidence detail follows decision 4. Error-summary links move focus to the field. A reload sets `aria-busy` on the screen.
- **Stale responses.** A request sequence drops stale read responses (as `retention/Live.tsx:64-85`). A command form ignores a second send while busy.
- **Store.** The attempt and draft store (module level, keyed by command and target) is scoped by workspace, user and session (`setScope`, as in P04). It is cleared by `forget()` on sign-out, on a scope change and on revoked access.

## Shared frame (every tab)

- **Scope bar** (directly under the title, `spec.md:109`): workspace, data revision, now (UTC), release candidate ("{version} at {build}" or "None selected"), and Reload (visual specification §5). At a `.p05-live` width of 1080px or more, the statement and candidate lines sit beside the three notes (visual specification §5).
- **Candidate line:**
  - with a candidate: "Release candidate {version} at build {build}. This is a gate check only; releasing is not connected." `release` other than `not_connected` reads "Release: Unknown (code)";
  - without: "No release candidate is selected. Gate checks, test outcomes and load readiness are not evaluated."
- **Statement:**
  - "**Shows evidence records only.** Nothing on this screen releases, activates or supports a combination, records a customer proof, links an evidence file or measures load in this build."
  - KO: "**증거 기록만 보여줍니다.** 이 빌드에서는 이 화면에서 출시·개통·지원 판정, 고객 실증 기록, 증거 파일 연결, 부하 측정을 하지 않습니다."
  - AR: "**يعرض سجلات الأدلة فقط.** لا تُصدر هذه الشاشة في هذا الإصدار أي شيء ولا تفعّل أي مجموعة دعم ولا تعتمدها، ولا تسجّل إثباتًا لدى العميل، ولا تربط ملف دليل، ولا تقيس الحمل."
- **Origin note:** see Truthfulness rule 2.
- **Open decisions note** (small): "Owner decisions still open: the first supported combinations, performance hardware and latency targets, how long evidence stays valid, and who is accountable for activation." KO "아직 정해지지 않은 결정: 최초 지원 조합, 성능 장비와 지연 목표, 증거 유효기간, 개통 책임자." AR "قرارات المالك التي لم تُحسم بعد: مجموعات الدعم الأولى، وأجهزة الأداء وأهداف زمن الاستجابة، ومدة صلاحية الأدلة، والمسؤول عن التفعيل."
- **Permission note** (small): "Every command here currently needs installation administration. Separate evidence-recording, support-decision and activation-request permissions are not available yet, and auditors cannot open this screen yet." KO and AR as P04's permission note pattern.
- **Whole-page states:**
  - first load: "Loading support combination and release evidence… Uncounted values are never shown as 0." (`spec.md:92`), with no numbers;
  - reload: `aria-busy`;
  - `not_configured`: "Release evidence is not configured in this workspace.";
  - an unknown `state`: "The release evidence read returned an unknown state (code).";
  - denied: "Release evidence is not available to you" / "You do not have permission to view release evidence in this installation.";
  - revoked: see decision 6;
  - unauthenticated: "Your session has ended." with "Sign in again" (calls `forget()`);
  - failure (`busy`, `unavailable`, `limited` with the wait) with Try again. The failure box adds the spec's error sentence: "The evidence could not be checked. Do not treat an earlier result as still valid; check the verification state again." (`spec.md:96`).
- **Spec states mapped** (`spec.md:88-99`; design review L2):
  - **loading** (`:92`): the first-load sentence above.
  - **empty** (`:93`): each list's "No {items} are recorded." sentence with a real 0. The spec's "check the scope and filters" clause is dropped because no filters exist (deviation, residual).
  - **partial** (`:94`, `:99`): not applicable. Every P05 command acts on one proposal, requirement or profile and has no multi-item job, so no partial business result exists. The async job enum (`:99`) has no counterpart in the API (deviation, recorded).
  - **blocked** (`:95`): decision 7.
  - **error** (`:96`): the failure box with the spec sentence.
  - **revoked** (`:97`): decision 6.
  - **Error detail** (`:99`): every failure box and every command failure shows:
    - the stable code (`AdminError.code`, or the failure class when no code is returned) in an ltr isolate;
    - the request ID;
    - the field location, through the error summary links for `invalid` with fields;
    - the next action in words: Try again, Reload, Sign in again, wait {n}, or Start again.
    - For commands, the request ID is the attempt's `request`. For the read, the host creates the request ID and passes it to `releaseEvidence(request)` (see Allowed files), so the ID shown is the one sent in `X-Request-Id`.
    - No stack trace, secret or customer text is shown.
  - **Main action under the title** (`:109`): the spec's main actions (link evidence, record support decision, request activation review) are not connected. The scope bar therefore holds only Reload, and the three actions are listed as text on the matrix tab (deviation, recorded).
- **Read-only.** When `revision` is empty, commands are disabled and the tab reads "Commands are unavailable because the data revision is unknown. Reload."

## Tabs (the registered P05 keys)

Registered KO tab labels (`screens.json` P05): 지원 행렬 / 요구 추적 / 시험 artifact / 부하 프로파일 / 판정 이력. EN view titles: "Support matrix", "Requirement trace", "Test artifacts", "Load profiles", "Decision history". AR: "مصفوفة الدعم", "تتبع المتطلبات", "مواد الاختبار", "ملفات تعريف الحمل", "سجل القرارات". The title is the registered "지원 조합·출시 증거" / "Support combinations and release evidence" / "مجموعات الدعم وأدلة الإصدار".

### 1. Support matrix (`matrix`)

- **Lead:** "Each row is a support combination: provider, contract, account role, product version and issuance path. Design, implementation, test and customer proof are separate columns; mapping a requirement to a combination never counts as a pass. Activation and support decisions are not connected." Then the blocked sentence (decision 7).
- **Table** (full column set when no detail is open; the reduced set of decision 4 beside an open panel), one row per path, columns in the spec order (`spec.md:59`): combination → requirement IDs → design → implementation → test → customer proof → last observed → blockers. Widths and the M fold of "Last observed" under Test follow visual specification §6.1.
  - **Combination:** provider, contract, role, version and issuance path as labelled facts (each an ltr isolate); template, or "No template"; the path ID; the state:
    - `designed` → "Designed, not active";
    - `active` → the unexpected sentence (rule 1);
    - another value → "Unknown (code)".
  - **Requirement IDs:** each `label` as a link when `target` is well-formed. The state in words below it, or "Not evaluated" when `state` is `null`. "Showing 50 of {total}." when `more`.
  - **Design, Implementation:** "{count} of {of}". Implementation is "with a recorded commit".
  - **Test:** "{count} of {of} tested at the candidate build" and "{failing} failing", or "Not evaluated".
  - **Customer proof:** "{count} of {of}" with the short line "Intake not connected" (rule 6), or "Not evaluated". The full intake sentence is the note under the matrix lead (rule 6). A count above 0 is rule 1.
  - **Last observed:** the time; `null` with a candidate → "No result recorded for this combination's tests"; without a candidate → "Not evaluated".
  - **Blockers:** each problem in words with its subject (ltr); `path_build` is rule 1. `[]` is rule 1 (it needs an active path). `null` → "Not evaluated".
- **Evidence detail** (decision 4: beside the list when `.p05-live` is at least 940px, a separate step below that; opened with the row's "Evidence" link or `path=`). Heading: the path ID with its provider and issuance path. The internal layout follows visual specification §4.5.
  - **Every matrix column** (repeated because the reduced list hides most of them, visual specification §14.2):
    - the five combination facts, template and state;
    - the requirement references with their states;
    - design; implementation; test with failing; last observed;
    - customer proof with the full intake sentence;
    - the blockers in full.
  - **Support since:** "Not connected: no combination has been activated." (`spec.md:37` "지원 시점").
  - **Test cases** (`spec.md:35-37`, design review M5): for each path requirement whose target matches a row of `requirements.rows`, the requirement label followed by that row's `tests` as stacked test blocks. Each block has the test ID, category, outcome, counted, passes, failures and the not-run or not-counted reason, worded exactly as on the requirements tab.
    - A path requirement with no matching row (a `null` target, or a row beyond the requirements list's 50) reads "Test cases for {label} are not in this read."
    - `tests: null` reads "Not evaluated".
  - **Test categories:** six rows, each "Counted pass recorded at the candidate build" (`true`) or "No counted pass at the candidate build" (`false`); `null` → "Not evaluated". A missing key reads "Unknown".
  - **Reconciliations:** period, "Matched" / "Not matched", differences. "No reconciliation recorded." when empty with a real 0. "Showing the newest 12 of {total}." when `more`. The note: "Two consecutive matched months are required before activation can be proposed; activation is not connected."
  - Activation, support decision and customer proof intake: "Not connected" each.
- "No support combinations are recorded." when empty with a real 0 (the spec's empty state, `spec.md:93`; its "check scope and filters" clause is dropped because no filters exist, a recorded deviation). "Showing 50 of {total}." when `more`.
- **Not-connected actions** below the table: "Link evidence", "Record support decision" and "Request activation review" (`spec.md:71-74`) are listed as text, each with "Not connected in this build", never as buttons.

### 2. Requirement trace (`requirements`)

- **Lead:** "Requirements, their implementation and tests at the release candidate build, and the gate check for the candidate. Requirements are sample records; the planning trace (HB and AUD IDs) is not loaded."
- **Gate block** (when `gate` is not `null`):
  - "Gate check for {version} at {build}: not ready, {total} problems." For `ready: true`, rule 1 applies.
  - Problems table: problem in words, subject (ltr). `common_unregistered` has an empty subject and reads "Whole register" in the subject cell. `path_build` is rule 1. "Showing 100 of {total}." when `more`.
  - "Retired without replacement" (`retired_blocking`): labels, or "None.".
  - "Templates usable with no problems": "None. A template is listed only for an active combination with no problems, and no combination can be active in this build." A non-empty list is rule 1.
  - "Release: Not connected".
  - `gate: null` → "Not evaluated: no release candidate is selected."
- **Compare builds** (decision 5), beside the gate block when `.p05-live` is at least 1080px and stacked below that (visual specification §6.2).
- **Table**, one row per requirement in read order:
  - **Requirement:** label (ltr); "Common" and "Blocking" neutral tags when true; "Retired" tag with "Replaced by {label}" or "No replacement";
  - **Source / Owner reference / Interfaces:** module IDs (ltr). The owner header reads "Owner reference" with the hint "A register ID, not a person."
  - **State at the candidate build:** in words, or "Not evaluated";
  - **Implementation:** the commit (ltr, first 12 characters with the full value in the detail), or "No implementation recorded";
  - **Tests:** "{n} tests" with the counts of pass recorded, fail recorded and no result; `null` → "Not evaluated";
- **Action** (decision 8, in the follow row): **Propose retirement** when allowed; a pending `retire` → the pending sentence with its link; otherwise the blocker sentence.
  - `replacement_required` that is not a boolean fails closed: no command, "No command is available for this row."
  - `can_retire: true` with `replacement_required: true`, but every listed replacement dropped as malformed: no command, "No usable replacement is listed. Reload."
- **Evidence detail** (decision 4; opened with the row's "Evidence" link or `requirement=`). Heading: the requirement label.
  - It repeats the columns the reduced list hides: source, owner reference (with "A register ID, not a person."), interfaces, and the implementation commit in full.
  - It then shows:
    - the design digest (64 hex, own fact row, wrapping);
    - the retirement state, worded as in the list: "Retired" with "Replaced by {label}" or "No replacement", or "Not retired";
    - the pending sentence when a retirement is pending;
    - "Evidence files: Not connected".
  - **States by build:** build → state words, newest first. "Only the newest 10 builds are shown." when `builds_more`.
  - **Tests table:** test ID; category in words, or "No category"; outcome; counted; passes and failures; reason.
    - Outcome: `pass` → "Pass recorded"; `fail` → "Fail recorded"; `null` → "No result".
    - Counted: "Counted" or "Not counted".
    - Reason: `not_run_reason` `no_result_recorded` → "Not run: no result is recorded for this build". `not_counted_reason`:
      - `no_implementation` → "Not counted: no implementation is recorded";
      - `build_predates_implementation` → "Not counted: this build was registered before the latest implementation";
      - `not_counted_by_module` → "Not counted by the release trace register" (no cause is guessed);
      - another value → "Unknown (code)".
    - A row with a result and neither reason reads no reason line. A row with no result and no `not_run_reason` reads "Reason: Unknown" (fail closed; the API always sends one, `evidence.go:444-445`).
    - "Showing 50 of {total}." when `more`.
- "No requirements are recorded." when empty with a real 0.

### 3. Test artifacts (`artifacts`)

- **Lead:** "Builds, test harnesses and recorded test results. Digests name sample labels: no evidence file, log, execution environment or content is stored, so none can be opened."
- **Not connected list:** Evidence files, Logs, Execution environment, Evidence content: each `not_connected` → "Not connected"; another value "Unknown (code)".
- **Builds:** build (ltr), commit (ltr), order "Registered {order}". "No builds are recorded." when empty.
- **Harnesses:** module in words ("Release trace" / "Load"), ID (ltr), "Revocation: Not connected". Harness identities are never in the read and never shown.
- **Results** (newest first): test, namespace, build, outcome ("Pass recorded" / "Fail recorded"), "Counted" or "Not counted" with the reason words as on the requirements tab, passes and failures, evidence digest (64 hex, wrapping), harness, time. "No test results are recorded." when empty with a real 0. "Showing 50 of {total}." when `more`.

### 4. Load profiles (`load`)

- **Lead:** "Load profiles, their budgets, recorded runs and the load gate at the release candidate build. All values are synthetic sample records, not measurements. A budget stays proposed: budget approval is not connected, so no load verdict or gate can pass in this build."
- **Planning targets** (copy, `planning_targets: not_connected`): "Planning targets: more than 1,000,000 incoming records a day and at least 30,000,000 accumulated a month. 100,000 records is an intermediate test size. These targets are not connected as data." (preview `Screen.tsx:142-145`; `spec.md:41`, `103`).
  - It adds: "No profile for concurrent collection, close and deletion exists." (`spec.md:41`, SCALE-11).
- **Measurement:** "Real load measurement: Not connected."
- **Readiness block** (or "Not evaluated"):
  - "Load gate at build {build}: not ready." For `ready: true`, rule 1 applies.
  - Per gate: gate in words, status in words, the profiles with their status, and retired profiles listed separately as "Retired profiles (not counted)".
  - **Run and failures** (design review M2; `modules/perfgate/reads.go:74-89`):
    - `run` is the latest run of the profile at the candidate build. With status `not_run`, `run` is 0 and reads "No run at this build", never "Run #0".
    - `failures` counts only runs recorded after an applied budget approval whose verdict is not passed (`reads.go:84-85`). Budget approval is not connected, so the count is always 0 and says nothing about the runs. It is shown as "Failures since budget approval: Not connected", never as "0 failures".
    - Failing statuses are the module's set (`reads.go:91-92`): `other_data`, `incomplete`, `incorrect`, `revoked`, `self_verified`, `missing_metric`, `other_hardware`, `over_budget`. A gate with any of them reads Failing. `unbudgeted`, `predates_budget` and `provisional` are not failing.
- **Hardware:** ID, CPUs, memory "{n} MiB", disk "{n} GiB", storage in words, spec digest.
- **One block per profile:**
  - ID, gate in words, dataset and expected digests, "Retired" tag when retired.
  - Status at the candidate build in words, or "Not evaluated".
  - **Axes:** a `<dl>` of the 13 axes with labels; a present 0 reads "0".
  - **Active budget:**
    - `null` → "No budget is active for this profile.";
    - `proposed` → "Proposed, not approved. Budget approval is not connected, so runs against it stay provisional at best.";
    - `approved` → rule 1.
    - Limits per metric with units.
    - **People** (design review M4; `modules/perfgate/changes.go:224-228`):
      - "Budget proposed by" for `proposer`;
      - "Registration approved by" for `approver`, who approved registering the budget, not the budget itself;
      - "Budget approval: Not connected" in place of `approval_by`, `approved_by` and `approved_at` while they are `null`. A non-null value on a proposed budget is rule 1.
  - **Runs** (newest first), as core columns plus a metric follow row (visual specification §6.4, §14.9):
    - core columns: run number, build, verdict in words, rows, duration "{n} s", recorded at, harness;
    - the follow row: hardware, mismatches, interrupted "Yes"/"No" and the nine metrics with units (`null` → "Not measured", a missing key "Unknown"). The client compares no metric with a limit.
    - "No runs are recorded." when empty with a real 0. "Showing the newest 20 of {total}." when `more`.
  - **Action** (decision 8, in `.p05-actions` at the end of the article): **Propose retirement** when allowed; a pending retire → the pending sentence with its link; otherwise the blocker sentence.
- **Verdict words**, none positive:
  - `provisional` "Provisional: within a proposed budget, not approved";
  - `over_budget` "Over performance budget";
  - `incomplete` "Incomplete: fewer rows than the profile";
  - `incorrect` "Incorrect output";
  - `other_data` "Different dataset";
  - `revoked` "Harness revoked";
  - `self_verified` "Self-verified: not accepted";
  - `missing_metric` "Missing metric";
  - `unbudgeted` "No budget";
  - `other_hardware` "Different hardware";
  - `predates_budget` "Run predates the budget";
  - `passed` → rule 1.
- **Gate status words:** `not_run` "Not run"; `failing` "Failing"; `provisional` "Provisional"; `passed` → rule 1.
- "Harness revoked" is a load verdict, not the screen's revoked access state; the copy keeps the two apart.

### 5. Decision history (`decisions`)

- **Lead:** "Support decisions and activation review are not connected. This tab lists the proposals recorded in the release trace and load registers. The registers are kept in memory and reset with the process."
- **Facts:** Support decisions: Not connected; Activation review: Not connected; Ledger `in_memory` → "In memory (not durable)"; unknown values "Unknown (code)".
- **Proposals:** under two headings, "Waiting for approval ({n})" and "Closed ({n})", each newest first (visual specification §6.5). A selected proposal (`proposal=`) has `aria-current`, a background tint and a visible "Selected" tag. Each row has:
  - register in words ("Release trace" / "Load"), number, kind in words, summary in words, state, proposer and created time;
  - the proposal reason (`rationale`) on pending rows.
  - `applied` rows of kind `activate`, `proof` or `approval` are rule 1.
  - Other `applied` rows show the approver with the closing reason. `withdrawn` rows show the withdrawer (`closed_by`) with the closing reason. `store_retired` rows read "No longer held by the register. Nothing can be done with it here." and offer nothing.
  - "Applied means recorded in the register; it releases and activates nothing."
- **People.** A person with `directory: false` reads "{id} (not a directory user)". A `null` login otherwise falls back to the ID.
- **Summaries** (placeholders isolated; an empty or missing value reads "Unknown"):

| Kind | EN |
| --- | --- |
| requirement | Register requirement {label} |
| retire (trace) | Retire requirement {label}, replaced by {replacement} / with no replacement: {reason} |
| implemented | Record the implementation of {label} at commit {commit} |
| build | Register build {build} at commit {commit} |
| harness | Register test harness {harness} |
| path | Register support combination {path} |
| reconcile | Record reconciliation of {path} for {period}: matched / not matched, {differences} differences |
| proof | Customer proof for {label} at {build} for {period} (not connected) |
| activate | Activation of {path} at {build} (not connected) |
| profile | Register load profile {profile} for {gate} |
| hardware | Register hardware {hardware} |
| budget | Propose budget {budget} for {profile} on {hardware} |
| approval | Budget approval for {budget} (not connected) |
| harness (load) | Register load harness {harness} |
| retire (load) | Retire load profile {profile} |

- **Approve** only when allowed (see Commands); **Withdraw** only when allowed.
- "No proposals are recorded." when empty with a real 0. "Showing the newest 50 of {total}." when `more`.

## Commands: forms, confirms and availability

Every form has:
- an action heading;
- the rationale field "Reason for this action" (1–256 UTF-8 bytes after trimming);
- an error summary;
- Review, then a confirm step with exactly one primary button (Confirm) and Cancel.

An open form hides its opener and sibling openers. The confirm restates what the read showed; its facts are frozen into the attempt's pre-send snapshot before the first send.

### Propose requirement retirement (`release_evidence.retire_requirement`, target `trace-requirement-<n>`)

- **Offered** when `hint(can_retire, retire_blocked)` allows it and `target` matches `^trace-requirement-[1-9][0-9]*$` within 128 bytes.
- **Fields** (`LiveRetireForm.tsx`):
  - **Retirement reason** (required, single-line input). It is stored with the proposal; the action reason records why you propose it.
    - Trimmed, then 1–200 UTF-8 bytes with no control characters (the module rule).
    - Errors: "Enter a retirement reason.", "The retirement reason is too long (at most 200 bytes).", "Remove line breaks and control characters."
  - **Replacement** (decision 3). Options whose target does not match the pattern, or equals the requirement's own target, are left out.
    - Nothing chosen: "Choose a replacement, or No replacement." When required: "This requirement is common, so it needs a replacement. Choose one."
    - "Only the first 20 possible replacements are listed." when `replacements.more`.
    - Hint: "A replacement must be registered later than this requirement, not retired, not already a replacement, and at least as strong (common and blocking are kept)."
  - The action reason.
- **Body:** `{reason}` with the trimmed retirement reason, plus `replacement: <target>` only when a replacement is chosen.
- **Confirm:**
  - the requirement (label, common and blocking tags, state at the candidate build or "Not evaluated", tests summary); the replacement (label, with its state), or "No replacement"; the retirement reason.
  - "Another administrator must approve it. Approval retires {label} permanently: this build has no way to undo a retirement. Nothing is released or activated."
  - Without a replacement: "Without a replacement it is listed as retired without replacement in every gate check, and a release candidate that includes it reports it as retired."
  - With a replacement: "{replacement} takes its place in gate checks. If the replacement is no longer at least as strong when the proposal is approved, the approval is refused."
  - When the requirement is blocking and its state is `failing` or not tested: "Retiring it removes its blocking check from the gate. This does not mean its tests passed." (`gate.go:44-56`)
- **Blocker sentences:** see the table below.

### Propose load profile retirement (`release_evidence.retire_profile`, target `perf-profile:<id>`)

- **Offered** when `hint(can_retire, retire_blocked)` allows it, `target` equals `perf-profile:` + `id`, and the target is 1–128 bytes of `[A-Za-z0-9._:-]`.
- **Fields:** the action reason only. The API has no retirement reason field for a profile (`evidencecmds.go:227`).
- **Confirm:**
  - profile, gate in words, status at the candidate build, budget state, number of runs.
  - "Another administrator must approve it. Approval retires the profile permanently; its runs stay listed. The load gate stops counting it, so the gate may read Not run instead of Failing. That does not mean the load passed." The last sentence is weight 600.
  - **The latest verdict, stated in words** (design review M3). When the profile's `status` is a verdict (not `not_run` and not `null`): "Its latest run at this build reads {verdict words}. Retiring it does not fix that run."
    - The words come from the verdict vocabulary, so `incomplete` (the seeded `sample-month`) reads "Incomplete: fewer rows than the profile", `over_budget` reads "Over performance budget", and so on for every verdict.
    - When the verdict is in the module's failing set (`modules/perfgate/reads.go:91-92`), a second sentence follows: "This run counts as failing in the load gate; after retirement the gate no longer counts it."
    - `not_run` reads "No run is recorded at this build." `null` (no candidate) reads "Not evaluated."
    - `passed` is rule 1. An unknown verdict reads "Unknown (code)".
  - The approve confirm for a profile retirement uses the same sentences.

### Approve (`release_evidence.approve`, target `trace-proposal-<n>` / `perf-proposal-<n>`)

- **Offered** when `hint(can_approve, approve_blocked)` allows it, `kind` is `retire`, `module` is `release_trace` or `perf_gate`, `target` equals the module's prefix plus `number`, and `number` is a safe integer.
- **Confirm:** the summary, proposer and their reason.
  - For a requirement retirement: the requirement's current row facts from the requirements tab (state, common, blocking) and the replacement's, with the retirement confirm's effect sentences.
  - For a profile retirement: the profile's current facts and the profile retirement effect sentences.
  - If the row the summary names is not in the read: "{label} is not listed in this read; its current state is unknown."

### Withdraw (`release_evidence.withdraw`)

- **Offered** when `hint(can_withdraw, withdraw_blocked)` allows it, `kind` is `retire` and the target is well-formed.
- **Confirm:** "Withdrawing closes this proposal. Nothing else changes."

### Availability parsing (fail closed)

- Every pair uses `hint`: `can_retire`/`retire_blocked` (requirements and profiles), `can_approve`/`approve_blocked`, `can_withdraw`/`withdraw_blocked`.
- Targets are used only when they match their pattern, equal the value built from the row's identity, and are 1–128 bytes of `[A-Za-z0-9._:-]`. Otherwise no command is offered and the row reads "No command is available for this row."
- All commands need a non-empty page `revision`.

### Blocker display policy and sentences

- **Hidden only:** `not_pending` on closed rows; `not_proposer`; `retired` on the retire action of a retired row; `store_retired` (its own row sentence).
  - These produce no blocker line. The row note carries a state sentence so no row silently lacks an action:
    - `not_proposer` on a pending row: "Only the proposer can withdraw it.";
    - `retired` on a requirement or profile: "Retired; there is nothing to retire."
- **Explained:** every other code, plus `not_pending` on a pending row (unexpected). Every sentence has EN, KO and AR copy in `live-codes.ts`. An unknown code reads "This is not available now (code)"; `missing` reads "This is not available now."

| Code | Where | EN |
| --- | --- | --- |
| `already_proposed` | retire requirement, retire profile | "Retirement proposal #{n} is already waiting for approval." / "Your retirement proposal #{n} is waiting for another administrator's approval." ("Your" only when that proposal row reads `own_proposal`.) Without the number: "A retirement proposal is already waiting for approval." |
| `no_eligible_approver` | retire requirement, retire profile | No other administrator could approve a retirement now, so it cannot be proposed. |
| `no_replacement` | retire requirement | This requirement is common, so it needs a replacement, and no requirement can replace it now. |
| `common_needs_replacement` | approve | The requirement is common and the proposal names no replacement. |
| `weaker_replacement` | approve | The replacement is no longer at least as strong, or is retired or already a replacement. Withdraw and propose again. |
| `retired` | approve | The requirement or profile is already retired. |
| `own_proposal` | approve | You proposed this. Another administrator must approve it. |
| `not_pending` | approve, withdraw (pending row) | This is no longer waiting for a decision. Reload. |
| `store_retired` | approve, withdraw | The register no longer holds this proposal. Nothing can be done with it here. |
| `not_console_kind` | approve | Proposals of this kind cannot be approved or withdrawn from the console in this build. |
| `target_too_long` | retire profile | This profile's identifier is too long to name in a command, so it cannot be retired here. |
| `not_found` | retire requirement, approve | The register no longer knows this requirement or its replacement. |

## Attempts, refusals and reactions

- **Attempts.** A request ID and an idempotency key are kept per command and target in the scoped store.
  - They are reused only to resend identical values (fields and rationale) after an unknown outcome (network failure or `busy`), or after a step-up prompt, even if a reload moved the revision.
  - Any edit or definite answer discards them, including `partial`.
  - A pending attempt shows "The last request may have been applied. Check the list, or send it again."
- **Pre-send snapshot.** Before its first send, an attempt stores:
  - the listed proposal numbers (for the retirements, for the duplicate check);
  - what its confirm stated;
  - context for vanished-row notices (proposal number, requirement label, profile ID).
- **Resend.** A pending attempt keeps its command usable even if the reloaded row no longer allows it. The resend skips read-based checks; only the frozen target must still be well-formed. It reuses the frozen target, fields, rationale and `expected_revision`. Its confirm restates the frozen snapshot with "This repeats your earlier request exactly as it was sent."
- **Drafts.** The rationale and the retirement reason are kept per command and target while the session lasts. The replacement choice is not a draft: the list is re-read.
- **Step-up (HYB-293 contract, `plan.md:389-405`; checked against the merged code before implementation):**
  - the host takes the client from `useAdminClient()`; it never builds `adminClient` itself;
  - a `reauthentication_required` answer opens the shared dialog in the client. On confirmation the client resends the byte-identical request once, with the same request ID and idempotency key;
  - `AdminError("reauth_cancelled")` (dialog cancelled) and `AdminError("reauth")` (refused again after reauthentication) map to the reaction `reauth`:
    - the form shows `sc.reauthText(l, failure)` as an alert;
    - the attempt and its key are kept (as the HYB-293 draft keeps them for P04);
    - a second Confirm opens the dialog again with the same attempt.
  - The step-up refusal records nothing on the server (HYB-293 `plan.md:518`), so keeping the key cannot duplicate a proposal. A replay of a completed command is also refused until reauthentication and then returns the stored result (HYB-293 `plan.md:304`).
- **Reactions:**
  - **`invalid` with fields:**
    - `reason` → the retirement reason field;
    - `replacement` → the replacement field ("This replacement is no longer accepted. Choose another.") and a reload;
    - `rationale` → the action reason field;
    - any other field (`target_id`, `expected_revision`, `idempotency_key`, `request_id`, `client`, the excess marker, an unknown name) → "The request could not be sent as built. Reload and try again."
  - **`invalid` without fields** → the action reason field ("The reason is too long" over 256 bytes, otherwise "Check the reason").
  - **Row note precedence** (design review L1). After a definite answer and the reload, the row note shows in this order, and only the first applicable entry:
    1. **An earlier attempt was pending.** If an attempt of the same command and target was pending before this send (a resend, a send after an unknown outcome, or a renewed attempt after "Start again" that inherited the flag), the note reads "The register refused this request. Your earlier request may have been applied: check the list." It never says "Nothing changed", because the earlier send may have changed something.
    2. **The reloaded row's blocker sentence**, when the row shows a blocker or has closed.
    3. **The kept per-command sentence below**, only when neither of the above applies.
  - **`stage` or `refused`:** reload, then the row note by the precedence above. The kept per-command sentences:
    - approve, `refused`: "You cannot approve your own proposal. Nothing changed.";
    - approve, `stage`: "The register refused this approval. Nothing changed. The proposal may no longer be pending, the requirement or profile may already be retired, or the replacement may no longer be at least as strong.";
    - withdraw, `refused`: "Only the proposer can withdraw it. Nothing changed."; withdraw, `stage`: "The register refused this withdrawal. Nothing changed.";
    - retire requirement, `stage`: "The retirement was not proposed. Nothing changed. It may already be retired or proposed, it may need a replacement, or the replacement may be too weak."; `refused`: "The retirement was not proposed. Nothing changed. You have too many pending proposals in the register.";
    - retire profile: the same two forms for a profile.
    - The kept sentence clears on the next definite answer for that command and target, or when the reloaded row shows a blocker or closes. Kept sentences for rows that left the read are dropped.
  - **`conflict`** → reload, keep the rationale and fields, and ask to confirm again. If the reloaded read no longer offers the command, the form closes, focus moves to the row note, and the row's state or blocker explains why.
  - **`denied` on a command:** reload.
    - If the read succeeds: "This proposal / requirement / profile is no longer available."
    - If the read is denied, decision 6 (revoked).
    - With a pending attempt and a proposal missing from the reloaded read (evicted beyond 200; not reachable with the sample seed): "Your earlier request may have been applied. The proposal is no longer listed." The attempt is discarded.
  - **Pending attempts whose row vanished** get a tab-level notice: "This item is no longer listed. Your earlier request may have been applied." with Dismiss.
  - **Other failures:**
    - `unauthenticated` → sign in again;
    - `limited` → the wait, keeping the attempt and the pending sentence (HB-289 residual: the P04 wait message hid it);
    - `busy` or `unavailable` → retry with the same attempt;
    - `stale` → "Start again" (renew the attempt). This is safe: a repeated retirement is refused as `already_proposed`, and a repeated approve or withdraw is refused as not pending.
      - **The earlier-attempt-was-pending flag carries across "Start again"** (design review round 2, L-b). An `idempotency_key_expired` or `idempotency_conflict` can follow a request that was applied. The renewed attempt therefore inherits the flag from the attempt it replaces, and a later `stage` or `refused` answer uses precedence entry 1, never "Nothing changed".
      - The flag clears only on a success, on Cancel, or when the reloaded read shows the row closed or with a blocker that explains it.
    - `partial` → the operation text with its ID, a reload, and the attempt discarded;
    - `reauth` → see Step-up;
    - anything else → "Failed".
- **Duplicate results.** A retirement whose returned `proposal` is in the snapshot announces "This was already proposed as #{n}." Through the real API this is not expected (`evidencecmds.go:169-171`, `183-185`). It is covered by model and service tests with a stubbed success.
- **Counted sentences.** Every sentence with a count uses `Intl.PluralRules` for the locale, with whole sentences per category (EN `one`/`other`, KO `other`, AR all six). Model tests cover AR 0, 1, 2, 3, 11 and 100.
- **Successes:**
  - retire requirement: "Retirement proposal #{n} for {label} is waiting for another administrator's approval.";
  - retire profile: "Retirement proposal #{n} for load profile {profile} is waiting for another administrator's approval.";
  - approve: "Proposal #{n} applied. {label} is retired. Nothing was released or activated." ("is retired" only when the reloaded read shows it retired);
  - withdraw: "Proposal #{n} withdrawn.";
  - unknown result states read "Unknown (code)".

## Unknown, zero, not evaluated and not connected

- The page `revision` is opaque, shown and sent as is; empty reads "Unknown" and disables commands.
- Numbers are safe non-negative integers or `null` (`count`); `null` reads "Unknown", a real 0 reads "0". This covers every count, `of`, `failing`, `differences`, `order`, `passes`, `failures`, `number`, `retire`, axes, CPUs, sizes, limits, `rows`, `duration_s`, `mismatches`, metric values, `run` and `total`. A uint64 beyond the safe range reads "Unknown", and a proposal number beyond it offers no command.
- Metrics: `null` reads "Not measured", a missing key "Unknown".
- Booleans (`common`, `blocking`, `retired`, `counted`, `matched`, `interrupted`, `directory`, every category, `more`, `builds_more`, `replacement_required`, every `can_*`) are `null` when not a boolean. A null `can_*` fails closed; a null `more` reads "There may be more."; a null category reads "Unknown".
- A field documented as `null` without a candidate reads "Not evaluated" only when `candidate` is `null`. With a candidate, the same `null` reads "Unknown". This keeps "not evaluated" from hiding a server fault.
- Missing arrays read "Unknown"; empty arrays read the tab's empty sentence.
- Times that are empty, unparseable or year 1 read "Unknown".
- `not_connected` reads "Not connected". An unknown enum value reads "Unknown (code)", never blank.
- Digests are shown only when they match `^[0-9a-f]{64}$`; commits only when they match `^[0-9a-f]{40}([0-9a-f]{24})?$`. Otherwise "Unknown (malformed value)".

## Layout (binding visual source: `visual-spec.md`)

**The ui-designer's visual specification [`visual-spec.md`](visual-spec.md) is the binding visual source** for composition, widths, containers, type, spacing, state styling, form layout, focus visuals, RTL and the render checklist. It was written by the ui-designer role (Claude CLI, read-only) on 2026-11-18 and is committed with this plan as `visual-spec.md`; root's reference record is [`references.md`](references.md). It is not owner acceptance, and the implementer receives it with this plan.

**Precedence (binding).**
- **The plan wins on copy, truthfulness rules and state wording.** Where a specification cell's content or wording differs from this plan (Truthfulness rules, the tabs, Vocabularies, Commands), the plan's text is used. The specification decides only where and how that text is laid out.
- **The specification wins on layout only.**
- The root decisions below settle the items the specification left open.
- **Amendments.** Root amended the specification on 2026-11-18 so that it agrees with this plan at these points:
  - a header stating this precedence;
  - the readiness cells read "No run at this build" and "Failures since budget approval: Not connected";
  - the budget labels read "Registration approved by" and "Budget approval: Not connected";
  - an empty `[]` blockers value renders the rule 1 sentence;
  - the reduced table never stacks;
  - §15 (external references) is marked superseded by root's references record (see External references).
- Any further difference found during implementation goes to root, and the plan's wording applies until root decides.

**Root decisions on its open items (folded into this plan):**
- the split threshold is 940px of `.p05-live`, in JS and CSS (§4.1; decision 4);
- the customer proof sentence appears once in the note under the matrix lead and once in the panel, with "Intake not connected" per cell (§14.4; rule 6);
- beside an open panel the list is reduced and the panel repeats every hidden column (§4.3, §14.2; decision 4);
- there is no empty panel: a one-line hint is shown instead (§4.2, §14.12);
- the step renders only the frame, tab heading, back link and detail (§4.4, §14.7);
- actions sit in follow rows, never in a column (§6.2, §14.8; decision 8);
- runs are core columns plus a metric follow row (§6.4, §14.9);
- the build comparison is a three-column table with no arrow glyph (§6.2, §14.3; decision 5);
- the selected row shows a visible "Selected" tag (§9, §14.11);
- the fifth shell tab spans both columns at 390 (§10, §14.13);
- the panel is sticky with a height cap and keyboard scrolling (§4.2, §14.6).

**Its §14 fixes are adopted:**
1. reduced list;
2. panel repeats hidden columns;
3. three-column comparison;
4. customer proof sentence once;
5. 940px split;
6. sticky height cap;
7. step content;
8. action follow rows;
9. runs follow row;
10. nearest-container nowrap with `<wbr>` after "/" in labels;
11. Selected tag;
12. no empty panel;
13. fifth tab spanning.
- Item 14 (copy and terminology) belongs to the tech-writer.

**Binding points in short** (details in the specification):
- **Containers.**
  - `.p05-live` decides the frame grid and the split mode.
  - `.p05-list`, `aside.p05-detail`, each `.p05-reg` and each profile article are their own inline-size containers.
  - Table S/M/L (S < 720, M 720–1079, L ≥ 1080) and ID nowrap key on the **nearest** container (§3, §11).
  - **Exception:** the reduced table beside an open panel (`.p05-split[data-open="true"] .p05-table.p05-reduced`) never stacks; only step mode replaces it.
- **Viewport media only** at `@media (max-width: 767px)`: the 3px control inset, full-width buttons, the scope-facts reflow and the shell tab grid with the fifth tab spanning (§2, §10).
- **No dashboard devices.** No progress bars, meters, percentages or charts: counts are "{n} of {m}" text (§1, §13).
- **States** follow the specification's state table (§7). Absent values use `.p05-absent` muted words. A real 0 is ink. The unexpected value is `p.p05-unexpected` in ink, weight 600, never showing the positive value.
- **Tokens:** existing tokens only. No colour literals, radii, shadows or gradients; the panel divider is a 1px `--border-row` (§4.2, §12, §13).
- **Forms and confirms:** §8. One primary per step; the permanence and "does not mean … passed" sentences at weight 600; a 440px field cap; a byte counter on the action reason.
- **RTL and multilingual:** §10. Logical properties only; one ltr isolate per compound value; free text `span.p05-free[dir=auto]`; no direction or tick glyphs; KO `keep-all`.
- **Lessons from P02–P04:** no overlapping tables; dates never broken inside the year; IDs not broken at hyphens at L; no rule above the first fact of a block.

## Accessibility and keyboard

- Reading order: title → scope bar → candidate line → statement and notes → tab heading → lead → lists → details → actions (`spec.md:113`).
- The shell tabs keep their arrow-key behaviour; P05 adds no tab widget of its own.
- Every table has a caption (visually hidden where the heading says it) and `scope` on header cells; stacked rows keep the labels.
- **Evidence detail** (decision 4):
  - the "Evidence" link names its row ("Evidence for {label}", visually "Evidence");
  - the panel is `aside` with `aria-labelledby`; it is not modal and does not trap focus;
  - explicit activation focuses the detail heading; Close, Back to the list and Escape return focus to the row's link; URL selection does not move focus;
  - in step mode the list is not rendered, so assistive technology does not reach hidden rows;
  - the sticky panel is a scroll container with `tabindex="0"`; Tab reaches it, and the arrow keys and Page Up/Down scroll its content. Its focus ring is inset;
  - the selected row has a visible "Selected" tag as well as `aria-current`, so selection is not shown by colour alone;
  - opening or closing the detail is announced through the existing status region only when focus does not move.
- Status is never colour only: states, verdicts, outcomes and problems are words, with neutral tags.
- Count strips and axes are `<dl>` lists; each number has its label.
- The replacement select and the build compare selects are native controls with visible labels.
- A confirm step focuses its heading; the error summary is focused on a client failure and links to fields; Cancel returns focus to the opener. The HYB-293 dialog owns focus while open and returns it to the confirm button.

## Vocabularies (EN / KO / AR)

All are in `live-codes.ts` with an "Unknown (code)" fallback; tests check every key in every language. **KO and AR are AI drafts pending terminology review.**

| Group | Code | EN | KO | AR |
| --- | --- | --- | --- | --- |
| Requirement state | designed | Designed, not implemented | 설계됨(미구현) | مُصمَّم، غير منفَّذ |
| Requirement state | implemented | Implemented, not tested | 구현됨(시험 미완료) | منفَّذ، لم يُختبر |
| Requirement state | failing | Failing at this build | 이 빌드에서 실패 | فاشل في هذا الإصدار |
| Requirement state | tested | Tested at this build (+ " (sample results)" for `sample_seed`) | 이 빌드에서 시험됨 (+ "(샘플 결과)") | مُختبَر في هذا الإصدار (+ "(نتائج عينة)") |
| Requirement state | customer_proven | (rule 1: unexpected value) | (규칙 1) | (القاعدة 1) |
| Path state | designed | Designed, not active | 설계됨(개통 안 됨) | مُصمَّم، غير مفعّل |
| Category | amount / approval / bulk / duplicate / isolation / recovery | Amounts / Approval / Bulk processing / Duplicates / Isolation / Recovery | 금액 / 승인 / 대량 처리 / 중복 / 격리 / 복구 | المبالغ / الاعتماد / المعالجة المجمّعة / التكرار / العزل / الاسترداد |
| Problem | not_tested | Not tested at this build | 이 빌드에서 시험 안 됨 | لم يُختبر في هذا الإصدار |
| Problem | blocking | Blocking requirement not tested | 차단 요구사항 시험 안 됨 | متطلب حاجب لم يُختبر |
| Problem | retired | Retired requirement in the candidate | 후보에 폐지된 요구사항 포함 | متطلب منتهٍ العمل به ضمن المرشّح |
| Problem | common_missing | Common requirement missing from the candidate | 후보에 공통 요구사항 누락 | متطلب مشترك غير مُدرج في المرشّح |
| Problem | common_unregistered | No common requirement registered | 등록된 공통 요구사항 없음 | لا يوجد متطلب مشترك مسجّل |
| Problem | unknown | Unknown requirement or combination | 알 수 없는 요구사항 또는 조합 | متطلب أو مجموعة غير معروفة |
| Problem | path_inactive | Combination not active | 조합 미개통 | المجموعة غير مفعّلة |
| Problem | path_build | Combination active at another build | 다른 빌드로 개통된 조합 | المجموعة مفعّلة على إصدار آخر |
| Problem | path_reconciliation | Latest reconciliation not matched | 최근 대사 불일치 | آخر مطابقة غير متطابقة |
| Problem | path_requirement | Combination requirement missing from the candidate | 후보에 조합 요구사항 누락 | متطلب المجموعة غير مُدرج في المرشّح |
| Problem | path_evidence | Test evidence incomplete for the combination | 조합의 시험 증거 불충분 | أدلة الاختبار غير مكتملة للمجموعة |
| Outcome | pass / fail / null | Pass recorded / Fail recorded / No result | 통과 기록됨 / 실패 기록됨 / 결과 없음 | سُجّل نجاح / سُجّل إخفاق / لا نتيجة |
| Reason | no_result_recorded | Not run: no result is recorded for this build | 미실행: 이 빌드의 결과 기록 없음 | لم يُنفَّذ: لا توجد نتيجة مسجّلة لهذا الإصدار |
| Reason | no_implementation | Not counted: no implementation is recorded | 미반영: 구현 기록 없음 | غير محتسب: لا يوجد تنفيذ مسجّل |
| Reason | build_predates_implementation | Not counted: this build was registered before the latest implementation | 미반영: 최근 구현 전에 등록된 빌드 | غير محتسب: سُجّل الإصدار قبل آخر تنفيذ |
| Reason | not_counted_by_module | Not counted by the release trace register | 미반영: 레지스트리가 인정하지 않음 | غير محتسب لدى السجل |
| Module | release_trace / perf_gate | Release trace / Load | 출시 추적 / 부하 | تتبع الإصدار / الحمل |
| Gate | e2e / daily_inflow / monthly_close / interrupt_resume / catch_up | End-to-end run / Daily inflow / Monthly close / Interruption and resume / Catch-up | 단일 작업 종단 / 일 유입 / 월 마감 / 중단·재개 / 지연분 처리 | تشغيل شامل / التدفق اليومي / الإقفال الشهري / الانقطاع والاستئناف / تعويض التأخر |
| Gate status | not_run / failing / provisional | Not run / Failing / Provisional | 미실행 / 실패 / 잠정 | لم يُنفَّذ / فاشل / مؤقت |
| Budget state | proposed | Proposed, not approved | 제안됨(미승인) | مقترحة، غير معتمدة |
| Storage | ssd / hdd / network | SSD / HDD / Network storage | SSD / HDD / 네트워크 스토리지 | SSD / HDD / تخزين شبكي |
| Proposal state | pending / applied / withdrawn / store_retired | Pending / Applied / Withdrawn / No longer held | 승인 대기 / 적용됨 / 철회됨 / 보관 안 됨 | بانتظار الاعتماد / مطبّق / مسحوب / لم يعد محفوظًا |
| Action | retire | Propose retirement | 폐지 제안 | اقتراح إنهاء العمل |
| Connection | not_connected | Not connected | 연결되지 않음 | غير متصل |
| Ledger | in_memory | In memory (not durable) | 메모리 내(비내구) | في الذاكرة (غير دائم) |
| Origin | sample_seed / empty | Sample records / No records | 샘플 기록 / 기록 없음 | سجلات عينة / لا سجلات |

The 13 axes and 9 metrics get labels with units in `live-codes.ts` (for example `rows_day` "Rows a day", `p95_ms` "95th percentile latency (ms)", `throughput_rows_s` "Throughput (rows/s)").

**Terminology citations and questions** (`.agents/rules/product-terminology.md`; `docs/ui/design-system/terminology.md` rules 1, 5, 9):
- "Support combination" reuses the preview's terms "지원 조합" / "مجموعة الدعم" (`Screen.tsx:84`).
- "Customer proof" follows `spec.md:33` "실증" and HB-41 "실제 고객 실증", KO "고객 실증". The preview's EN "Customer validation" and AR "التحقق لدى العميل" (`Screen.tsx:88`) are replaced in live mode by "Customer proof" and AR "إثبات لدى العميل", so validation is not confused with proof. Terminology review question.
- "Retire" uses KO "폐지" and AR "إنهاء العمل". These avoid P04's proposal state "폐기됨 / مُسقَط" (`store_retired` here reads "No longer held") and the AR "مسحوب" used for withdrawn. Terminology review question.
- "Harness revoked" (verdict) and "access withdrawn" (spec revoked state) use different words in every language.
- The decisions tab EN title "Decision history" replaces the preview's "Support decision history" (`Screen.tsx:70-74`), because support decisions are not connected. The registered KO label 판정 이력 stays.
- Proposal states `pending / applied / withdrawn` reuse P04's KO and AR (HB-289 `plan.md:583`).

## Root amendments after review (2026-11-18)

Root decisions on the change review, the rendered review and the terminology review of the first implementation. They take precedence over earlier text in this plan and in `visual-spec.md` where they differ. KO and AR words chosen here are installation defaults recorded for owner and native review, not final copy.

1. Unexpected-value sentence: `{code}` is the field locator listed in Truthfulness rule 1; the positive value is never rendered.
2. Customer proof: rule 6 as amended (note under the matrix lead, no header sub-line).
3. Layout:
   - Tests in the detail panel and in step mode are stacked blocks at every width; the 720-939px step table of visual-spec §4.5 is withdrawn.
   - Two- and three-column registers (`.p05-gate`, `.p05-compare`, `.p05-reg` tables) stack only below 480px of their container.
   - Reduced matrix columns: Combination 40%, Test 24%, Customer proof 16%, Blockers 20%.
   - Harness IDs wrap; the per-row tests summary renders one count per line without separators.
   - Panel requirement references: the state line stays with its own link.
   - Row hover tint is removed inside P05 tables so only the selected row is tinted.
   - Focus moved after a pointer open does not show a ring; a keyboard open does.
   - "{n} of {m}" counts are start-aligned.
4. Copy (EN):
   - The budget approval fact is the label "Budget approval" with the value "Not connected".
   - Headings and the load lead say "Performance budget" (meaning trap: an unqualified budget reads as money in a billing product).
   - Reconciliation summaries use the counted plural for differences.
   - The applied note adds: "An applied budget proposal records a proposed budget; budget approval is not connected."
   - A refused command on a proposal that has closed shows its note on the closed row and moves focus there.
   - An approve confirm for a profile missing from the read reads Unknown, not "Not evaluated".
   - A server `invalid_request` on the reason shows the client check text for that reason, or "Check the retirement reason."
   - The tests summary says the counts cover the rows shown when the list is truncated.
   - A malformed `ready` uses the neutral title.
   - Reauthentication cancel keeps the attempt with its own sentence: "Password confirmation was not completed, so this request was not sent."
5. KO defaults:
   - 공급사 (provider).
   - 해지 (harness revocation).
   - 불일치 건수 (mismatch count).
   - 통과 횟수 / 실패 횟수 (count columns).
   - 구현됨(미시험).
   - 종단 간 실행.
   - 피크 구간(분).
   - WAL(미리 쓰기 로그).
   - 성능 판정 (load verdict column).
   - 성능 예산.
   - 귀하의 폐지 제안 #n.
   - The verdict line "이 빌드의 최근 실행 판정: {verdict}. 폐지해도 그 실행 결과는 바뀌지 않습니다."
   - No space before "(" in "(샘플 결과)".
   - 출시 추적 레지스트리 in the not-counted reason.
6. AR defaults:
   - البنية for build; الإصدار for release only; نسخة for product version.
   - بصمة for digest.
   - الإبطال / مُبطَلة for harness revocation.
   - المقترح for the proposal noun.
   - ملفات السجل for logs.
   - مبدئي for provisional.
   - معدل المعالجة for throughput.
   - تشغيل من البداية إلى النهاية for end-to-end.
   - ميزانية الأداء.
   - Retirement always names its object (إنهاء العمل بالمتطلب, …به).
   - The tests line uses count labels (النجاحات المسجّلة / الإخفاقات المسجّلة / بلا نتيجة).
   - The not-counted reason names سجل تتبع الإصدار.
7. Settled after the first fix round:
   - A `path_build` gate problem uses the locator `path.blockers`; a profile status `passed` uses `run.verdict`; a readiness gate status `passed` uses `gate.status`.
   - Combination IDs and requirement labels may break at hyphens at L when they would otherwise overflow their column; no overflow outranks the spec's no-hyphen-break preference.
   - Registers narrower than 480px stack at every width.
   - AR issuance path reads "مسار تسليم نتائج الفوترة" (`modules/releasetrace/types.go:42-44`: api, file or xlsx channel for issued billing results; "invoices" would add a document class the API does not state); open for owner confirmation.
8. Connection fields outside rule 1 (release, activation, support decision, customer proof intake, revocation): any value other than the expected not-connected or known neutral value renders the neutral unexpected sentence with its field locator (`release`, `activation`, `support_decision`, `intake`, `revocation`), never "Unknown ({value})". The same applies to the other not-connected facts (artifacts, the decisions ledger and the origin note).
9. Settled after the round-2 terminology review:
   - Applied `perf_gate:budget` rows read "Registration approved by {who}".
   - Every other applied kind keeps "Approved by {who}".
   - EN verdict "Over performance budget".
   - The retirement confirm repeats the truncated-tests sentence.
   - KO defaults:
     - 성능 예산 everywhere a budget is meant;
     - 성능 판정 in the load lead;
     - "WAL(미리 쓰기 로그, 바이트)";
     - "표시된 시험만 집계한 수입니다.";
     - "이 성능 예산 기준 실행은 잠정 판정까지만 받을 수 있습니다".
   - AR defaults:
     - بانتظار الاعتماد for pending everywhere;
     - retirement names its object in every sentence;
     - the proposal number follows the proposal noun.
   - The P05 reauthentication-cancel sentence stays; aligning the shared sentence is an owner question.
10. Settled after round 3 (2026-11-18):
   - State-like fields (path state, requirement state, budget state, proposal state, run verdict, gate status, test outcome and page state) render the neutral unexpected sentence with their locator for any value outside their known set, never "Unknown ({value})". Descriptive codes such as kind, module, category and reason keep "Unknown ({code})".
   - `load.planning_targets` is read through the same fixed-value rule.
   - The customer proof cell and the matrix note follow `customer_proof_intake`.
   - AR retirement confirmations put the proposal number after the proposal noun.
   - The final layout polish uses the full matrix at L at Combination 15, Requirement IDs 17, Design 6, Implementation 12, Test 12, Customer proof 10, Last observed 11, Blockers 17 (%), and Results at L at Test 15, Build 10, Outcome 13, Counted 19, Passes 7, Failures 7, Harness 18, Recorded at 11 (%), where the reviewer's numbers overflowed.
   - The reduced matrix stays 40/24/16/20.
   - Requirement labels break only after "/" at L.
11. Open for the owner and native review (also: AR ملفات السجل versus register, AR verb-form dt labels): every KO and AR word in items 5 and 6, and the Arabic word for customer proof (kept as إثبات لدى العميل).

## External references (opened by root, 2026-11-18)

**Gate satisfied.** Root opened these on 2026-11-18 through WebFetch (text summaries; no rendered page or screenshot was inspected). Root recorded them in [`references.md`](references.md), transcribed here. This satisfies the reference gate of `.agents/rules/service-screen-style.md` before screen code. The ui-designer's render comparison checks the adopted patterns against them. The visual specification's own §15 (references the ui-designer could not open) is superseded by this section.

1. **GOV.UK Design System, Summary list.** https://design-system.service.gov.uk/components/summary-list/ (opened)
   - **Observed:**
     - rows of key (`dt`), value (`dd`) and optional actions, for key facts and metadata, not tabular data;
     - borders are on by default, and a no-border variant exists for lists without actions, with a caution that borders help users who zoom or use magnification;
     - missing values link to where they are entered;
     - narrow-screen behaviour is not described.
   - **Adopted:** the `dl` key/value shape for facts (`p05-sum`), with no row actions. Row separation stays readable under zoom, through spacing or the 1px row border.
   - **Rejected:** "Change" links.
2. **GOV.UK Design System, Error summary.** https://design-system.service.gov.uk/components/error-summary/ (opened)
   - **Observed:**
     - the summary sits at the top of main (below a back link, above the h1) and focus moves to it;
     - each link goes to its field and matches the inline message wording;
     - the page gives no rule for hint placement for single fields.
   - **Adopted:** summary placement, focus and link wording.
     - The hint between the label and the control is the project's existing `service-field` pattern, not a claim from this page.
   - **Rejected:** any coloured accent edge.
3. **Material Design 3, List-detail canonical layout.** https://m3.material.io/foundations/layout/canonical-layouts/list-detail
   - **Not available:** HTTP 404 on 2026-11-18. Replaced by the Android Developers page below.
4. **Android Developers, List-detail layout.** https://developer.android.com/develop/ui/compose/layouts/adaptive/list-detail (opened)
   - **Observed:**
     - with enough space, the detail pane is shown beside the list;
     - with limited space, the selected item's detail takes the whole space, and only one pane is visible at a time;
     - back in the single pane returns to the list;
     - the selection is kept across configuration changes.
   - **Adopted:**
     - split or single-pane step by available width;
     - "Back to the list";
     - the selection kept across the layout change (decision 4).
   - **Rejected:** animations, elevation, rounded surfaces.
5. **Azure DevOps, Requirements traceability.** https://learn.microsoft.com/en-us/azure/devops/pipelines/test/requirements-traceability (opened)
   - **Observed:**
     - requirements are linked to test results;
     - the Requirements quality widget lists requirements in scope with a pass rate and a failed-test count, and links the failed count to the tests;
     - requirements without any associated test are tracked.
   - **Adopted:**
     - per-requirement test counts, with the detail reachable from the row;
     - requirements and tests without results shown explicitly.
   - **Rejected:** pass-rate percentages. P05 shows counts and "no result" states, never a rate.
6. **GitLab, Release evidence.** https://docs.gitlab.com/ee/user/project/releases/release_evidence.html (opened)
   - **Observed:**
     - a JSON snapshot separate from the release, taken at collection time;
     - a release can have several snapshots;
     - each snapshot is identified by its collection timestamp. The page does not mention a digest.
   - **Adopted:** evidence is separate from the release and is a dated snapshot.
     - Showing the digest as the evidence identity comes from the HYB-290 API, not from GitLab.
   - **Rejected:** any release affordance.
7. **Grafana k6, Thresholds.** https://grafana.com/docs/k6/latest/using-k6/thresholds/ (opened)
   - **Observed:**
     - per-metric pass/fail criteria such as `p(95) < 200`;
     - each threshold is reported with a tick or a cross;
     - a run passes only if all thresholds pass.
   - **Adopted:** metrics listed with units per run, and the verdict as words from the server.
   - **Rejected:**
     - tick and cross output;
     - any client-side threshold evaluation, because budget approval is not connected.

## Rules carried from the HB-283, HB-285, HB-287 and HB-289 reviews

- **Host and remount:** live branch before the preview wrapper, locale-only key, hooks unconditional, remount per view, `data-screen-id` asserted.
- **URL state** validated; **focus** to the status after success, the row note after refusal, the opener after Cancel; summary links to fields.
- **Stale responses:** the request sequence.
- **Store:** scoped by workspace, user and session and cleared on sign-out; pre-send snapshots; resend only with identical values; the resend confirm restates the frozen snapshot and skips read-based narrowing.
- **Fixtures:** the shared fixture is a non-test module (HB-289 change review round 1: a fixture imported from a test file registered its tests twice).
- **Mobile:** stacking at 390 without horizontal scroll; one primary button during a confirm; an open form hides its opener and siblings and has an action heading.
- **RTL:** one ltr isolate per compound ID; FSI for dates; dates kept whole; free text in `span[dir=auto]`.
- **Unknown versus zero and empty;** a missing list is never shown as empty; year-1 times read "Unknown".
- **Blockers:** a sentence for every code with an unknown fallback; rows without a well-formed target offer no command; a missing number fails closed; the proposer's reason on pending rows and in the approve confirm; logins fall back to IDs.
- **Refusals:** the row note precedence (an earlier pending attempt first and never "Nothing changed", then the reloaded blocker, then the kept sentence) with the flag carried across "Start again", and the kept sentence's clearing; command `denied` → reload → "no longer available"; a vanished pending attempt gets a tab-level notice.
- **Truthful statements:** what each decision does and does not do, including permanence.
- **Terminology:** user-facing copy follows `.agents/rules/product-terminology.md`, the product glossary and `docs/ui/design-system/terminology.md`.

# Planning cross-check

Source: `docs/ui/screens/P05/spec.md:115-133` and `design.json` (`requirement_ids`, `acceptance_ids`), with requirement texts from `docs/planning/v1.16/requirements.json` and `detailed-requirements.json`. Status meanings follow the [HYB-291 report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). These are the planned statuses; `verification.md` restates them after the build. No acceptance test is run. **No ID is covered**: every ID's requirement text asks for real evidence, real paths or real measurement that this build does not have.

| ID | Planned status | Requirement (from the text) | Built by this screen | Not built, and why |
| --- | --- | --- | --- | --- |
| HB-04 | deferred | Authentication, operational read, first data, full scope, FOCUS fit, reconciliation and billing activation are different states; support per provider, contract, account role and data kind is shown with evidence. | Each combination shows provider, contract, role, version and issuance path with separate design, implementation, test and customer proof columns, the reconciliation periods and the gate blockers; activation never reads as reached. | Data kind is not a combination field in the API. Activation, the operational read test after a permission is revoked, and onboarding states (HB-04's main subject, other screens) are not connected. |
| HB-19 | deferred | Accept RI/SP/CUD commitment results; P05's part is DB-09. | Nothing for commitments. | The commitment event matrix (AUD-DB-09 below) does not exist in the API or the modules. |
| HB-24 | deferred | ERP common submission ledger; P05's part is DB-01 (the release gate must not drop common features). | The gate block shows `common_missing` and `common_unregistered` problems in words; a common requirement cannot be retired without a replacement, and the screen says so. | The submission ledger itself (phase 7) and its tests in the real trace. |
| HB-25 | deferred | Odoo and SAP proof; a mock ERP pass is not compatibility certification. | No path reads active and no requirement reads customer-proven; the screen states that sample results are not proof, and unexpected positive values fail closed. | ERP paths, product and version proof, and customer proof intake. |
| HB-26 | deferred | Standard XLSX and customer files; ERP upload formats are verified per product and version. | The sample XLSX combination shows its template with its blockers, so an unverified template never reads verified. The gate's template list reads that a template appears only for an active combination with no problems, which cannot occur in this build; a non-empty list fails closed (rule 1). | The XLSX files, result registration and their tests in the real trace. |
| HB-38 | deferred | Simulator and independent expected values; DEMO or synthetic marking. | Every section carries the API's `origin` label, and every tab shows the sample origin note for it (rule 2); "(sample results)" is added to the tested state only for `sample_seed` sections. | Scenario manifests with seed, clock and expected values, and demo reset (P06, not yet ticketed). |
| HB-39 | deferred | 1M+ daily and 30M+ monthly as P0 gates; not run is never shown as passed. | Load profiles, budgets, runs, verdicts and readiness, never passed; "Not run" and "Not measured" are distinct from 0; the planning targets are shown as copy, not data. | Budget approval, real measurement at scale, the concurrent collect, close and delete profile (SCALE-11), and system-level gate runs. |
| HB-40 | deferred | Reuse licences and commercial rights; distribution notices and licence boundaries are verified. | Nothing. | Release evidence for distribution notices and licence boundaries is not in the API (P07 is a separate screen). |
| HB-41 | deferred | Link requirement ID, source, owner, API or screen, tests, evidence and blocking; design, implementation, test pass and customer proof are different states. | Requirement rows with source, owner reference, interfaces, design digest, commit, tests with not-run and not-counted reasons, blocking and retirement; separate state columns. | The real planning trace (HB and AUD IDs), API and screen links per requirement, an accountable person (the owner is a register ID), customer proof intake, and two-period parallel reconciliation before activation. |
| AUD-DB-01 | deferred | First reconciliation through standard XLSX and result registration without an API; unverified ERP templates never shown as verified; the release gate blocks dropping common features. | The gate's common-requirement problems in words; the `no_replacement` and `common_needs_replacement` sentences; the sentence that templates appear only for active, problem-free combinations, which cannot occur, with a non-empty list failing closed (rule 1). | Standard XLSX reconciliation and result registration, ERP template verification state, the real trace. |
| AUD-DB-09 | not connected | An event test matrix per commitment profile; unsupported events recorded rather than passed. | Nothing. | No such matrix exists in the seed, the modules or the API. |

Deviations from the spec recorded as residuals: the empty-state filter clause (`spec.md:93`); the partial state and job enum, not applicable (`:94`, `:99`); the main actions under the title, not connected (`:109`); filters, server search and cursor paging (`spec.md:61-63`, `103`); links to S02, P06, I02, I03, U03 (`spec.md:82-86`); the evidence, support-decision and activation actions (`spec.md:71`, `73-74`); "review differences" against an earlier decision (`spec.md:72`; only build-to-build state comparison is built); the revoked state as a server state (`spec.md:97`); evidence validity and revalidation (`spec.md:52`); the measurement field and concurrency (`spec.md:53`, `41`); accountable person for decisions (`spec.md:55`).

# Verification plan

- **Service tests** (`release-evidence.test.ts`):
  - hardened parsers for every section: missing arrays, wrong types, unknown enums, unsafe and negative integers, uint64 beyond the safe range;
  - `null` people, `directory: false` people, `null` logins;
  - missing count keys and category keys; metrics with `null` and with missing keys; malformed digests and commits;
  - the no-candidate shape (`--seed none`, `origin: empty`): every documented `null` read as not evaluated, and the same `null` with a candidate read as unknown;
  - whole-page `not_configured` and an unknown page state;
  - command envelopes: retire requirement with and without `replacement` (reason trimmed), retire profile, approve and withdraw without fields; resend identity; result fields (`module`, `proposal`, `target`);
  - stubbed `invalid_stage`, `prerequisite_failed`, `invalid_request` with `reason`, `replacement`, `rationale` and another field, and `internal_error` with an operation ID mapped to `partial`;
  - with HYB-293 merged: a stubbed `reauthentication_required` through `useAdminClient`'s client resends byte-identically once; cancel gives `reauth_cancelled`.
- **Model tests** (`live-model.test.ts`):
  - **Truthfulness (rule 1, design review M1 and L5):** each value in rule 1's list renders the unexpected sentence, never the positive value, and is treated as not ready, not active, not proven, not approved and not passed. One case per value:
    - `ready: true` (gate and readiness);
    - path `active` and a non-null path `build`;
    - `customer_proven`;
    - verdict and gate status `passed`;
    - budget `approved`;
    - a proposed budget with each of `approval_by`, `approved_by` and `approved_at` non-null;
    - `customer_proof.count` 1;
    - `gate.templates` with one ID;
    - `blockers: []` beside `blockers: null`;
    - `path_build` as a gate problem and as a path blocker;
    - an applied `activate`, `proof` and `approval` proposal. No copy key in any language contains a positive "ready", "active", "passed", "supported", "released" or "customer-proven" claim except in statements that none happens (a word-list test over `live-copy.ts` and `live-codes.ts`, reviewed by the tech-writer).
  - Availability parsing fails closed for all three pairs.
  - Target patterns and equality: proposal per module, requirement, profile (115 and 116 bytes of ID), replacement options (own target and malformed targets dropped).
  - Retirement reason: empty, whitespace, 200 and 201 bytes (multi-byte KO and AR), a line break, a C1 control; trimmed before sending.
  - Replacement choice: placeholder only, "No replacement" offered only when not required, required error, `more` sentence.
  - Confirm sentences: with and without a replacement; blocking and failing requirement; profile retirement with a failing verdict; a summary row missing from the read.
  - Non-retire pending proposals offer no command even with `can_withdraw` true.
  - The blocker display policy with every code in EN, KO and AR; `already_proposed` own and other and without a number; `missing` and the unknown fallback.
  - The build comparison: two builds, identical states, `builds_more`, invalid URL builds, no candidate.
  - Unknown versus zero versus not evaluated versus not connected versus not measured for every numeric field, list, boolean, time and metric.
  - Reactions: every field mapping, fieldless `invalid`, kept sentences per command and their clearing, `denied` → "no longer available" per target kind, revoked handling (store cleared), the evicted-proposal sentence, the vanished-attempt notice, `limited` keeping the pending sentence, `reauth` and `reauth_cancelled` keeping the attempt.
  - The attempt store: identical values only, scoped by session, cleared by `forget`, discarded on `partial`.
  - The duplicate announcement with a stubbed success, and none when a resend replays a stored success.
  - URL selection validation for every parameter and tab; every href starts with `#/release-evidence?`.
  - `Free` renders `span[dir=auto]`, never `bdi`.
  - The evidence detail mode (design review L3):
    - side by side at an available width of 940px or more, step below it;
    - 939.99 gives step, 940 and 940.4 give side by side, and `floor` matches the CSS `min-width` query;
    - `splitWidth` equals the number in `live.css`'s `@container` rule;
    - a 0 width before measurement gives step; the decision comes from the measured `.p05-live` width, not the viewport; the focus target after open, close, Escape and Back; the selection kept across a mode change; the list not rendered in step mode with a selection.
  - The "Harness revoked" and access-withdrawn strings differ in every language.
  - The matrix evidence detail lists the test cases of each path requirement from `requirements.rows`, and the "not in this read" sentence for a requirement without a row (M5).
  - Readiness (M2): `run` 0 with `not_run` reads "No run at this build"; `failures` always reads "Failures since budget approval: Not connected", and never "0 failures".
  - Profile retirement confirm (M3): one case per verdict, including `incomplete` ("Its latest run at this build reads Incomplete: fewer rows than the profile. Retiring it does not fix that run."); the failing-set second sentence exactly for the module's eight failing verdicts; `not_run` and `null`.
  - Budget people (M4): "Registration approved by" for `approver`, and "Budget approval: Not connected" while the approval fields are `null`.
  - Edge cases (L3): `common_unregistered` with an empty subject reads "Whole register"; `replacement_required: null` offers no command; `can_retire: true` with every replacement malformed offers no command and reads its sentence.
  - Row note precedence (L1): after a resend refused as `stage` or `refused`, the note never contains "Nothing changed".
  - Revoked flag (L4): set after a denied read following a success; a new mount's first `denied` reads revoked while the flag is set and denied when it is not; cleared by a successful read, `forget()` and a scope change.
  - "(sample results)" appears only for `origin: sample_seed`.
  - Error detail (L2): a read failure shows the code, the request ID passed to `releaseEvidence(request)` and the next action; a command failure shows the attempt's request ID.
  - Plural forms through `Intl.PluralRules` for EN, KO and AR (0, 1, 2, 3, 11, 100).
  - Every vocabulary key in EN, KO and AR.
- **Checks:** `npx tsc --noEmit`, `npm test`, `npm run build`; changed files under 1,500 lines; no comments.
- **Playwright QA** against the real API (`hb-api --seed sample`, built from this branch's base after the HYB-293 rebase) with the console preview (`HB_GATED=1 vite preview`).
  - Each group starts on a fresh API with an empty mode-0700 data directory, so no group depends on another or on order.
  - The runners and screenshots are session working files, not committed (`docs/artifact-layout.md`: PNG is output). `verification.md` records their location, counts and the inspected scope.
  - **Render pass R** (fresh API, no commands), as the first administrator: every tab × EN/KO/AR × 1440/390, plus 1280 for matrix and requirements. A screenshot per render.
    - `data-screen-id="P05"` and `data-source-state="live"` (never P01); no horizontal scroll; `dir` per locale; computed borders of tags, notices and selected rows neutral; the measured `.p05-live` width and container size recorded per render;
    - one primary button in an open confirm (one confirm opened per tab that offers a command);
    - the statement, origin note, open decisions note, permission note, candidate line and each tab's lead and not-connected sentences present;
    - the page text contains no "Passed", "Ready", "Active", "Customer-proven", "Supported" or "Released" as a state (KO and AR equivalents from the word list);
    - matrix: `sample-path-xlsx` reads 2 of 2, 1 of 2, 1 of 2 tested, 0 failing, customer proof 0 of 2 with the intake sentence, four blockers in words, the reconciliation "Not matched, 3 differences"; the blocked sentence; the three not-connected actions as text;
    - requirements: gate "not ready, 5 problems"; `sample-common-xlsx` reads the `no_replacement` sentence; `sample-legacy-report` reads "Your retirement proposal #19…" for A1; the `t-erp-template` detail reads "Not run: no result is recorded for this build"; the build comparison `0.0.1` → `0.0.2` lists the changed states;
    - artifacts: `t-xlsx-amount` at `0.0.1` reads "Not counted: this build was registered before the latest implementation"; the four not-connected lines; no runner identity in the DOM;
    - load: `sample-daily` "Over performance budget" with throughput 13 and p99 1,500 in the run's metric follow row; `sample-resume`'s readiness line reads "No run at this build"; no "0 failures" text anywhere, and "Failures since budget approval: Not connected" present; budget people read "Registration approved by admin2@example.invalid" and "Budget approval: Not connected"; `sample-resume` "No runs are recorded."; readiness with `monthly_close` Failing; budgets "Proposed, not approved"; the planning targets sentence and the SCALE-11 sentence;
    - decisions: 28 proposals total with "Showing the newest 50" absent; `perf-proposal-9` shows Approve; `trace-proposal-19` shows "You proposed this" and Withdraw; the sample reviewers read "(not a directory user)".
  - **Group A, as the first administrator** (`admin@example.invalid`):
    - approve `perf-proposal-9`: the confirm reads "Its latest run at this build reads Incomplete: fewer rows than the profile. Retiring it does not fix that run.", the failing-set sentence, and that the gate may read Not run instead of Failing, which does not mean the load passed (M3); after it `sample-month` is retired, readiness `monthly_close` reads Not run with `sample-month` under retired profiles, and its runs stay listed;
    - propose retirement of `sample-erp-template` with "No replacement" chosen explicitly → a new pending proposal; the row reads "Your retirement proposal #{n}…"; withdraw it from the decisions tab → withdrawn; Propose retirement is offered again;
    - propose retirement of `sample-approval` with replacement `sample-recovery` (captured body carries `reason` and `replacement: trace-requirement-3`, the rationale separately) → pending;
    - Review with no replacement chosen shows "Choose a replacement, or No replacement.";
    - `sample-common-xlsx` offers no retirement.
  - **Group B, as the second administrator** (`admin2@example.invalid`):
    - `trace-proposal-19` shows Approve; approve it → applied; `sample-legacy-report` reads Retired with "No replacement"; the gate still reads not ready with 5 problems and "Retired without replacement" lists `sample/sample-legacy-report`;
    - `perf-proposal-9` reads "You proposed this"; withdraw it → withdrawn; `sample-month` offers Propose retirement again;
    - closed proposals offer no action.
  - **Group C, failure paths** (first administrator unless stated):
    - **Revision conflict:** open Approve `perf-proposal-9` with a reason in tab 1; in tab 2 withdraw the A1-owned `trace-proposal-19`; confirm in tab 1 → conflict, reload, reason kept; confirm again → applied.
    - **Lost response after apply** (fetch the real response, then abort): approve `perf-proposal-9`; the pending sentence shows; the resend returns the stored `applied` result.
    - **Lost request before the server** (abort without fetching): propose retirement of `sample-api-export`; the resend creates exactly one proposal.
    - **Stubbed refusals by route:** `invalid_stage` on withdraw → focus on the row note with the kept sentence; `invalid_request` [`replacement`] → the replacement field error and a reload; `invalid_request` [`reason`] → the retirement reason field; `prerequisite_failed` on approve; `internal_error` with an operation ID → the operation text and a reload.
    - **Step-up** (HYB-293 merged; session older than the step-up window, or a stubbed `reauthentication_required`): Confirm opens the dialog; Cancel shows the "not confirmed" sentence and keeps the attempt; Confirm again and reauthenticate → the request is sent once with the same request ID and idempotency key, and one proposal exists.
    - **Read failures:** a stubbed `dependency_unavailable` → the failure box with the spec's error sentence and Try again; the member `member@example.invalid` sees the denied block; a stubbed `permission_denied` on a reload after a successful read → the revoked state, with no table, form or follow-up command in the DOM.
    - **Client checks:** a rationale over 256 bytes; a whitespace-only rationale; a retirement reason of 201 bytes; a retirement reason with a line break.
    - **Focus:** status after a success; opener after Cancel.
    - **Drafts:** a retirement reason survives a tab change and back.
    - **URL state:** invalid `view`, `path`, `requirement`, `from`/`to`, `build`, `profile` and `proposal` values are ignored; `view=requirements&requirement=4` opens that detail and is marked `aria-current`; `view=decisions&proposal=perf-9` is marked.
  - **Group E, empty installation** (`--seed none`): every tab reads "No records" and the no-candidate sentences; matrix, requirements, artifacts, load and decisions show their empty sentences with real zeros; no "Unknown" appears where "Not evaluated" is meant; no command is offered.
  - **Group F, evidence detail** (decision 4; visual specification §16 items 1–6), in EN, KO and AR, with a screenshot of each state:
    - **1440 (`.p05-live` about 1120px) and 1280 (about 960px):**
      - opening `sample-path-xlsx` and `sample-approval` shows the reduced list and the panel side by side;
      - at both 1440 and 1280, `.p05-table.p05-reduced` computes `display: table` and its rows `display: table-row`, so the reduced table has not stacked into cards;
      - the reduced Blockers cell reads "4 problems" for `sample-path-xlsx`, and no cell reads "0 problems";
      - the panel is at the inline end (left in AR), with a 1px `--border-row` divider;
      - the selected row has `aria-current` and the visible "Selected" tag;
      - there is no horizontal scroll;
      - the matrix panel repeats every hidden column (design 2 of 2, implementation 1 of 2, test 1 of 2, 0 failing, customer proof 0 of 2 with the full intake sentence, four blockers, last observed), the six categories, "Not matched, 3 differences", "Support since: Not connected", and the test cases of `sample-common-xlsx` and `sample-erp-template` (the latter "Not run: no result is recorded for this build");
      - the requirements panel repeats source, owner reference, interfaces and the commit;
      - no button or form is in the panel.
    - **Without a selection:** no panel; the one-line hint under the lead; the full column set (8 columns at 1440, 7 at 1280 with Last observed under Test).
    - **The mode follows `.p05-live`, not the window:** with the window at 1440 and `.p05-live` narrowed to 939px by a test-only style, step mode applies; at 940px, side by side. The measured widths are recorded.
    - **Sticky panel:** scrolling the list keeps the panel in view; Tab reaches the panel and the arrow keys scroll it to its last item.
    - **390:**
      - step mode renders only the frame, tab heading, "Back to the list" and the detail;
      - the lead, hint, gate block, comparison and list are absent from the DOM;
      - Back returns to the list.
      - Shell tabs: two columns, with the fifth tab spanning both.
    - **Keyboard:**
      - Tab reaches the row's "Evidence" link, and Enter focuses the detail heading;
      - Escape and Close (or Back in the step) return focus to the same link;
      - a cross link (`view=requirements&requirement=4`) opens the detail without moving focus or showing a focus ring.
    - **Resizing** across 940px with the detail open keeps the selection.
  - **Preview without the service** (L5): with no service context (the unconnected mode), `#/release-evidence` renders the unchanged preview `Screen.tsx` with `data-screen-id="P05"`, no `data-source-state="live"`, the `P05:state:unconnected` review state, and no request to `/api/v1/release-evidence`. EN, KO and AR at 1440 and 390.
  - **Regressions**, each on a fresh API: P04 (`qa289` R, A–D), P03 (`qa287run.sh` R, A, B, C), P02 (`qa285run.sh` A–D), P01 (qa283full). M01 and the O-screens run if the HYB-293 rebase touched their files.
  - Console errors during QA: none expected except the deliberate aborts.
- **Reviews and record:** ui-designer visual specification and render comparison; tech-writer terminology review in context; independent design and change review; `verification.md` with the planning cross-check restated from the built behaviour.
- **Screen-binding register.** `docs/ui/screen-binding-register.md` is not updated; `verification.md` records the P05 field-to-read and action-to-command map as input for HYB-295.

# Residuals

- **Not connected in this build:**
  - support-path activation and the activation review request; support decisions (supported, limited, unsupported, unverified) and their history; customer proof intake;
  - evidence linkage, files, logs, environment and content; harness binding and revocation; result and run recording;
  - requirement, implementation, build, path, reconciliation, profile, hardware and budget proposals; budget approval, so no load verdict or gate can pass;
  - real load measurement and the planning targets as data; the concurrent collect, close and delete profile;
  - the real planning trace; a durable decision ledger; the commitment event matrix (AUD-DB-09).
- **Not built from the P05 spec:** filters, server search, stable sort, cursors and fixed snapshots (HYB-304, HYB-305); links to S02, P06, I02, I03, U03; "review differences" against earlier support decisions (built only as the partial build-to-build state comparison of decision 5); an accountable person for decisions.
- **Evidence validity (spec field 검증 날짜·유효성, `spec.md:52`).** The API has result times (`at`, `last_observed`) but no validity period and no revalidation when a version or condition changes. The screen shows the times only and never says evidence is still valid; the validity period is an open owner decision (`spec.md:139`).
- **Revoked state:** the API has no revoked read; the screen covers the visible behaviour after a denied reload only.
- **Asymmetric retirement fields.** `retire_requirement` takes a module `reason` (1–200 bytes) and the rationale; `retire_profile` takes only the rationale (`evidencecmds.go:225-227`), so a profile retirement has no reason stored in the load register beyond the command rationale. The profile form shows only the action reason and says nothing about a stored retirement reason.
- **Profile retirement can turn a failing load gate into "not run"** (HYB-290 `plan.md:327`). The confirm states the latest verdict in words ("Its latest run at this build reads {verdict}. Retiring it does not fix that run."), adds the failing-set sentence for the module's failing verdicts, and says "The load gate stops counting it, so the gate may read Not run instead of Failing. That does not mean the load passed." (Commands, Propose load profile retirement.) The gate status is not otherwise annotated after approval; a later read shows the profile under "Retired profiles (not counted)" with its runs still listed.
- **Customer proof label.** The live screen uses the API meaning, "Customer proof" (KO 고객 실증, AR إثبات لدى العميل), for `columns.customer_proof` and `customer_proof_intake`. The preview's EN "Customer validation" and AR "التحقق لدى العميل" (`Screen.tsx:88`) stay only in the unconnected preview; aligning the preview is out of this ticket's files.
- **"Revoked" has two meanings.** The load verdict `revoked` (a revoked harness) reads "Harness revoked"; the spec's revoked access state reads "Your access has been withdrawn". The words differ in EN, KO and AR, and a model test asserts the two strings share no key.
- **Permissions and audience:** every command uses installation administration; P05's separate permissions and AU read access are later work.
- **Two administrators:** retirements are the only two-person decisions the console completes; with fewer than two eligible administrators every retirement reads `no_eligible_approver`.
- **Permanence:** an applied retirement cannot be reversed in this build.
- **Bounds:** at most 50 paths, requirements, tests per requirement, results, profiles, hardware and proposals; 20 runs per profile; 20 replacements; 12 reconciliation periods; 10 build states; 100 gate problems.
- **Persistence:** registers, store and journal reset with the API process.
- **Single-workspace seed:** another workspace's `not_configured` and the `store_retired`, `not_console_kind`, `weaker_replacement` and unexpected positive values are covered by model tests only.
- **Readiness failures** are counted by the module only after an applied budget approval, so the screen shows them as not connected rather than as a number (Tabs, Load profiles).
- **Spec states not built:** the empty-state filter clause (`spec.md:93`), because no filters exist; the partial state and the job enum (`:94`, `:99`), because no multi-item command or job exists; the main actions under the title (`:109`), because they are not connected.
- **Unverified layout facts** (visual specification "Uncertainties"):
  - whether the shell header is sticky, which sets the panel's top offset;
  - the ar-AE digit system;
  - the actual EN header widths.
  - All three are checked in the render pass.
- **Shell:** the scoped `.planning-tabs` (with the fifth tab spanning) and `th` overrides; keeping the selection across shell tab changes and the unsaved-draft warning on a tab switch (`spec.md:80`).
- **Reviews:** owner FIX of the screen; KO and AR terminology review of every AI-drafted string, in particular "retire" (폐지 / إنهاء العمل), "customer proof" (고객 실증 / إثبات لدى العميل), "support combination", the gate and verdict words, and "Decision history".
- **Screen-binding register:** catch-up in HYB-295.

# Assumptions (not verified in code)

- HYB-293 merges with the contract in its plan (`plan.md:389-405`): `useAdminClient()` in `service/context.ts`, failures `reauth` and `reauth_cancelled`, `sc.reauthText`, and the attempt kept for a resend. The draft worktree `/private/tmp/hb293` shows this shape for P04 (`retention/live-model.ts:350-383`, `Live.tsx:49`), but it is not merged and may change.
- `release_evidence.*` joins HYB-293's step-up set automatically because it registers as mutating (HYB-293 `plan.md:21`); this is checked in the rebased API before QA.
- External references: root opened them on 2026-11-18 through WebFetch text summaries. No rendered page or screenshot was inspected, and the Material Design 3 page returned 404. See External references.
- The P05 spec and planning citations come from the authorized local workspace copy, not this tree.
- `.p05-live` measures as `.p04-live` did: about 1120px at a 1440 window, 960px at 1280 and 358px at 390 (HB-289 `verification.md:27`). With the 940px threshold, 1440 and 1280 show the evidence detail side by side and 390 shows the step. Group F records the measured P05 widths.
- No longer an assumption: the `RevisionScope` key (`apps/console/src/app/RemainingScreens.tsx:83-92`) does not include the selection parameters (`path`, `requirement`, `from`, `to`, `build`, `profile`, `proposal`). That is a fact from the code (Host, remount and URL state). Only a later edit of that file could change it, and the rebase onto HYB-293 is checked for one.

# Root decisions (2026-11-18)

Root decided the plan's open questions, the visual specification's open items and the design review findings. Each decision is folded into the body; this list points to the sections that carry it.

**Plan open questions (first round)**
1. Unexpected positive values render as "Unexpected value" and are treated as negative: Truthfulness rule 1; Verification plan.
2. "Pass recorded" / "Fail recorded" and "Tested at this build", never "Passed": Truthfulness rule 3; Vocabularies.
3. Approve and Withdraw only on the decisions tab: Design decisions 1; Commands.
4. Non-retire pending proposals read-only, as P04: Design decisions 2.
5. The replacement is an explicit choice: Design decisions 3; Commands, Propose requirement retirement.
6. The evidence detail follows the spec, side by side on desktop and a separate step on narrow widths, decided by available width: Design decisions 4; Host, remount and URL state; Tabs 1 and 2; Layout; Accessibility; Verification plan, Group F.
7. Decisions tab EN title "Decision history": Tabs; Terminology citations.
8. Revoked state on a denied reload: Design decisions 6; Shared frame.
9. Build comparison built and labelled partial: Design decisions 5; Residuals; Planning cross-check.
10. Group E (`--seed none`): Verification plan.
11. KO 폐지 and AR إنهاء العمل as drafts for terminology review: Vocabularies; Residuals.
12. HB-19 and HB-40 stay deferred: Planning cross-check.

**Visual specification (second round).** `visual-spec.md` is the binding visual source, and its §14 fixes are adopted (Layout). Its open items were decided as follows:
13. Split threshold 940px of `.p05-live`, the same number in JS and CSS, with defined rounding and first-measurement behaviour: Design decisions 4; Verification plan.
14. The customer proof sentence once in the note under the matrix lead and in the panel, with "Intake not connected" per cell: Truthfulness rule 6; Tabs, Support matrix.
15. A reduced list beside the open panel, with the panel repeating every hidden column: Design decisions 4; Tabs 1 and 2, Evidence detail.
16. No empty panel; a one-line hint instead: Design decisions 4.
17. The step renders only the frame, tab heading, back link and detail: Design decisions 4.
18. Actions in follow rows, never a column: Design decisions 8; Tabs 2 and 4.
19. Runs as core columns plus a metric follow row: Tabs, Load profiles.
20. A three-column build comparison with no arrow glyph: Design decisions 5.
21. A visible "Selected" tag: Design decisions 4; Tabs, Decision history; Accessibility.
22. The fifth shell tab spans at 390: Layout; Verification plan, Group F.
23. A sticky panel with a height cap and keyboard scroll: Design decisions 4; Accessibility.

**Design review findings (CHANGES REQUIRED, second round)**
- M1, fail-closed list extended: Truthfulness rule 1; Tabs 1, 2 and 5; Planning cross-check (HB-26, AUD-DB-01); Verification plan.
- M2, readiness run and failures: Tabs, Load profiles; Residuals; Verification plan.
- M3, the profile retirement confirm states the verdict: Commands, Propose load profile retirement; Residuals; Verification plan, Group A.
- M4, budget people labels: Tabs, Load profiles; Verification plan.
- M5, test cases in the matrix evidence detail: Tabs, Support matrix; Verification plan, Group F.
- L1, row note precedence: Attempts, refusals and reactions; Verification plan.
- L2, spec states and error detail: Shared frame, "Spec states mapped"; Allowed files (`releaseEvidence(request)`); Planning cross-check deviations; Residuals.
- L3, edge cases: Design decisions 4 (rounding, first measurement); Tabs 2 (empty subject, `replacement_required`, filtered replacements); Verification plan.
- L4, revoked flag lifecycle: Design decisions 6; Verification plan.
- L5, verification additions: Verification plan (preview without the service, M1 to M5 cases).
- Nits:
  - one EN wording for `build_predates_implementation`: Tabs 2; Vocabularies; Verification plan;
  - `not_counted_by_module` without a guessed cause: Tabs 2; Vocabularies;
  - "(sample results)" only for `sample_seed`: Truthfulness rule 3;
  - a one-sentence description: front matter;
  - HB-38 "labelled per section": Planning cross-check.

**External references (third round).** Root opened them on 2026-11-18, and the reference gate is satisfied.
- The Material Design 3 page returned 404 and was replaced by the Android Developers list-detail page.
- Digest-as-identity is attributed to the API, not GitLab; Azure DevOps pass rates are rejected.
- See External references.

**Design review round 2 (CHANGES REQUIRED) and the visual specification amendment**
- V1, precedence (the plan wins on copy, truthfulness and state wording; the specification wins on layout only) and the 2026-11-18 amendment points: Layout.
- V2, the reduced table never stacks: Design decisions 4; Layout; Verification plan, Group F (computed `display: table` at 1440 and 1280).
- L-a, the reduced Blockers cell uses rule 1 for `[]` and never reads "0 problems": Design decisions 4; Verification plan, Group F.
- L-b, the earlier-attempt-was-pending flag carries across "Start again": Attempts, refusals and reactions.
- L-c: Assumptions (references opened by root; the `RevisionScope` key recorded as a fact); Host, remount and URL state; Rules carried (kept-sentence rule).
- L-d, the scoped `:has()` tab rule in P05's own CSS: Scope and ownership, Non-goals.
- Nits:
  - the requirement detail names "Replaced by {label}" / "No replacement": Tabs 2;
  - visual specification §15 superseded: External references;
  - the HYB-290 citation was checked and narrowed to `plan.md:35`, `36`, `39`: Truthfulness rule 1.

Spec/API mismatches stay in Residuals and Not connected:
- the retirement field asymmetry;
- profile retirement turning failing into not run;
- evidence validity;
- the customer proof label;
- the two meanings of "revoked";
- support decisions, evidence linkage, activation, filters, links, the owner reference and concurrency.
