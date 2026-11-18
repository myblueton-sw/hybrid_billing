---
type: ImplementationPlan
title: P04 retention API over modules/retention with synthetic sample artifacts, policies, holds, deletion runs and tombstones seeded through the module's own flows
description: Phase 5g of the pre-launch plan. Adds a retention read for the six P04 tabs (policies, expiry, holds, deletion, restore, decisions) and administrator commands with a reason to approve policy and release proposals, withdraw proposals, propose a policy revision, place a hold, propose a hold release and stop a deletion run. Nothing deletes, archives, restores, exports or touches a host after the seed; archive restore, deletion plans, workers, retries and backup expiry are shown as not connected.
status: stable
tags: [HB-288, pre-launch, retention, api]
sources:
  - resource: ../HB-257/plan.md
  - resource: ../HB-256/plan.md
  - resource: ../HB-286/plan.md
  - resource: ../HB-286/verification.md
---

# Scope and ownership

[HB-288](https://linear.app/hybrid-billing/issue/HB-288) adds the API behind P04 (data retention and deletion). Base: main `8abfe6b`. The P04 screen follows in a separate ticket. The plan follows the merged [HB-286 plan](../HB-286/plan.md) and [verification](../HB-286/verification.md) (recovery) and the HB-284 integrations precedent, and binds the module designed in [HB-256](../HB-256/plan.md).

**Ownership**
- Root (Claude Code) owns the plan and integration. Implementation may be delegated to one writer in this worktree.
- An independent read-only reviewer reviews the design and the change, with a DBA and security focus: actor kinds, two-person rules, tenant scoping, no deletion or access-block effects from requests, and no exposure of tokens or other tenants' identifiers.
- Machine: mac_mini.

**Allowed files**
- `modules/retention/exports.go` (new) and `exports_test.go`: three read-only exports, listed under Module exports. No rule changes.
- `apps/api/server`:
  - new `retention.go` (read), `retentioncmds.go` (commands and error mapping), `retentionseed.go` (seed and server store), and their tests `retention_api_test.go`, `retention_commands_test.go`, `retention_edges_test.go`;
  - registration in `admin.go` (next to `recoveryDefinitions`, `admin.go:361`), the route in `boundary.go` (next to `boundary.go:38`), the registry and store in `server.go` (next to `server.go:94-95,172,229`), the seed step in `seed.go` (after `seed.go:108`);
  - the seed step count in `world_test.go:57` (15 to 16).
- `docs/features/HB-288/*`.
- No other module, no console file, no `screens.json` change. `docs/ui/screen-binding-register.md` is not updated (residual; the catch-up is HYB-295; corrected by HYB-291).

**Non-goals**
- No physical, logical or simulated deletion caused by a request: no deletion plan proposal or approval, no claim, retry, report or proceed, no backup expiry record.
- No archive, hot/cold tiering or archive restore; no byte or growth accounting.
- No registration of artifacts or references from requests.
- No independent decision ledger, epoch or durable commit; the register is in memory and resets with the process (HB-257 residual).
- No new gate action; no AU read access.

# Observed baseline (facts from code)

## `modules/retention`

- **Registry.** `NewRegistry(clock)` refuses a nil clock (`registry.go:100-107`). Every bound is a module constant (`registry.go:8-31`): pending proposals 1000 in all and 250 per proposer (`maxPending`, `maxMine`), applied proposals kept 1000 (`maxDone`), 100 unfinished runs, 100,000 artifacts per run, chunk 1 to 1000, lease 1 minute to 24 hours, preview page 10,000, outcome and tombstone pages 1000, period 1 to 36,500 days, grace 0 to 365 days, impact window 1 hour.
- **Identifiers and text.** IDs are 1 to 128 bytes of `[A-Za-z0-9._:-]` (`common.go:16-27`); free text is 1 to 200 bytes of valid UTF-8 without C0, DEL or C1 controls (`common.go:29-38`). `validText` is unexported.
- **Actors** (`registry.go:122-148`). `human()` returns `ErrInvalid` for a malformed actor, `ai_not_allowed` for an assistant and `not_human` for a system actor; `system()` returns `not_system` for a human.
  - System only: `Register`, `AddTenant`, `Refer` (`records.go:34-129`), `Claim`, `Retry`, `Report`, `Proceed` (`runs.go:22-34,80-248`).
  - Human only: `Propose`, `Approve`, `Withdraw` (`registry.go:160-249`), `PlaceHold` (`records.go:137-162`), `Stop` (`runs.go:250-275`), `BackupExpired` (`runs.go:277-292`).
- **Proposals.** Kinds `policy`, `plan`, `release` (`types.go`, `changes.go:11-33`).
  - **Silent duplicate.** The same proposer proposing the same encoded change returns the existing pending proposal (`registry.go:173-179`).
  - `Approve` refuses an unknown number (`ErrNotFound`), a non-pending proposal (`not_pending`) and self-approval (`self_approval`), re-runs `check` and applies (`registry.go:198-228`). **A repeat approval by the same approver of an applied proposal returns the view without an error** (`registry.go:209-210`). Applied proposals beyond 1000 are deleted (`registry.go:223-226`).
  - `Withdraw` is limited to the proposer (`not_proposer`) and a pending proposal (`not_pending`) and deletes the proposal (`registry.go:230-249`).
  - `Proposal(number)` is the only proposal read; there is no list (`registry.go:251-259`).
- **Checks** (`changes.go:123-165`, pure: no writes):
  - `policy`: only on approval and only for a shortening (`Period+Grace` below the current revision's, `changes.go:52-55`), the reviewed impact must be at most one hour old (`impact_required`) and must match the recomputed count and digest at its `AsOf` (`impact_changed`).
  - `plan`: every artifact registered (`ErrNotFound`), of the plan class (`ErrInvalid`), not deleted (`already_deleted`) and not owned by a run (`in_plan`); at most 100 unfinished runs (`ErrLimit`).
  - `release`: the hold exists (`ErrNotFound`), is not released (`released`) and was placed by neither the proposer nor the approver (`placer`).
- **Application** (`changes.go:167-195`):
  - `policy` appends a revision numbered per class.
  - `plan` creates run `run/<n>` with `n` from an increasing counter (`changes.go:173-174`; documented in the HB-256 design). Each artifact gets `held`, `referenced` or `not_due` at once, or becomes `pending` and owned by the run, which **blocks `Access` immediately** (`changes.go:179-188`, `reads.go:12-13`). Approval does not return the run ID.
  - `release` marks the hold released and drops it from the live set.
- **Holds.** `PlaceHold` needs a valid hold (`records.go:131-135`: ID, owner ID, text reason, at least one artifact or tenant, classes only with tenants), refuses an existing ID with `ErrConflict` (`records.go:146-148`) and unknown artifacts with `ErrNotFound`, and is never refused for the state of what it covers. `HoldView.Counts` holds only the outcomes `not_preserved`, `outcome_pending`, `partially_deleted` of covered artifacts (`records.go:164-203`); keys with zero are absent. `HoldOutcomes(id, after, limit)` pages them with per-store results (`records.go:215-241`). There is no hold list.
- **Runs.** `Run(id)` returns the plan, policy revision, state (`running`, `stopped`, `done`, `complete`), next chunk, chunk count, damaged count, all eight outcome counts, per-store counts and the per-artifact outcome map (`runs.go:303-359`). There is no run list. `Stop` on a stopped run returns the view without a change; on a finished run it records the stopper and releases nothing (`runs.go:260-273`).
- **Backup expiry.** A deleted artifact is complete only when every store's deletion time is strictly before that store's recorded `before` (`runs.go:294-301`); `before` is never after the clock (`runs.go:285`).
- **Reports record deleted facts** even for a held artifact (`runs.go:188-199`; `TestLeasesAndStaleReports`).
- **Reads.** `Access`, `Artifact`, `Policy(class)` (current revision only, `ErrNotFound` without one), `Impact(policy)` (computed at the clock, with per-tenant counts and held and referenced counts), `Preview(class, after, limit)`, `Tombstones(after, limit)`, `Watermark()`, `Restored(applied)` (`reads.go:3-104`). `Preview` and `Impact` count across every tenant of the class (`reads.go:49-79`, `changes.go:66-88`). No read exposes whether a single artifact is held, referenced or due.
- **Errors:** `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrLimit`, and `Refusal{Reason}` wrapping `ErrRejected` (`types.go:8-14,93-99`).
- **Fixtures** (`world_test.go`): humans `ann`, `bob`, `cat`, systems `ingest`, `worker`; a run's ID is derived as `run/<runSeq>` in the test helper.
- **Source rules** (`source_test.go:11-46`): no comments, no imports with a dot, at most 1,500 lines. `go test -cover ./modules/retention/` reports 100.0%.
- **Importers.** No package outside `modules/retention` imports it.

## Server and commands (same as HB-286)

- Commands run under the server's domain write lock and move the revision on every completed mutating command. `Resolve` receives no actor and runs before the journal lookup; a target that no longer resolves is `permission_denied` on resend. Handler-coded failures are not journaled; `internal_error` is journaled and replayed (HB-286 baseline, `modules/commands/execute.go:307-425`).
- Only `invalid_request` carries fields. The server caps the rationale at 256 bytes (`directory.go:13`).
- Command targets must be command IDs: `[A-Za-z0-9._:-]`, no `/` (`modules/commands/types.go:299-310`, `execute.go:61-63`). A run ID `run/<n>` therefore cannot be a target as is.
- Field types include `Integer` (`^-?(0|[1-9][0-9]*)$`) and `Timestamp` (RFC 3339 with optional fraction) (`execute.go:18-19,29-51`).
- Admin reads use `admitted` (`users.go:42`); fault hooks use `faulty` (`admin.go:78`) through `moduleCall` (`integrationseed.go:174-179`); `stamp` renders RFC 3339 seconds (`users.go:40`); `validPlanHash` checks 64 lowercase hex (`installation.go:495`); `directory.loginOf` maps a user ID to a login (`directory.go:61`); `installKept` = 200 (`installseed.go:18`).
- Sample users: `SampleAdmin`, `SampleAdmin2`, `SampleMember` (`seed.go:22-26`).

## P04 design

`screens.json` names the tabs `policies`, `expiry`, `holds`, `deletion`, `restore`, `decisions` and the actions: impact preview, policy approval request, hold place/release request, deletion run request (approval, multi-stage), archive restore request. The P04 spec (`docs/ui/screens/P04/spec.md` in the authorized local workspace; not in this tree) requires separate states for acceptance, decision confirmed, access blocked, cleanup, active-store confirmation, backup expiry and policy completion (`spec.md:45,64,104-117`), says a hold on deleted data is never shown as preserved (`spec.md:85`), keeps other customers' existence and counts hidden (`spec.md:25`), distinguishes unknown from zero (`spec.md:75`), and leaves periods, deletion targets, backup expiry and hold approval responsibility open (`spec.md:157-161`). The preview page `apps/console/src/modules/administration/pages/retention/Screen.tsx:3-7` states that policies and durable decision records are not connected.

# Design

## Registry

`retention.NewRegistry(s.clock)` built in `server.New` next to the recovery registry. The module has no configurable limits; the sample uses 14 artifacts, 1 reference, 3 policy revisions, 2 holds, 2 runs and 7 proposals.

## Module exports (`modules/retention/exports.go`)

A reviewed module change, read-only, no rule change:
- `func ValidText(text string) bool { return validText(text) }`. The server checks hold and release reasons with the module's own rule, so `ErrInvalid` stays unreachable from requests.
- `func (r *Registry) Check(c Change, proposer, approver string, reviewed Impact) error`: under `enter`/`leave`, `clean(c)` (`ErrInvalid` when malformed), then `r.check(c, proposer, approver, reviewed, now)`. `check` is pure (`changes.go:123-165`), so `Check` changes nothing. It skips what `Propose` and `Approve` do around `check` (duplicate return, limits, actor kind, pending state, self-approval), so a command the read shows as allowed may still answer `prerequisite_failed` (`ErrLimit`).
- `type Standing struct { Access string; Due bool; DueAt time.Time; Held bool; Referenced bool; Damaged bool; Run string; Stores map[string]string }` and `func (r *Registry) Standing(id string) (Standing, error)`: under `enter`/`leave`, `ErrNotFound` for an unknown ID; `Access` as `Access(id)`; `DueAt` is `Anchor + (Period+Grace) days` under the current revision, zero without one; `Due` as `r.due`; `Held` as `r.held`; `Referenced` as `requiredBy[id] > 0`; `Damaged`; `Run` the owning run or empty; `Stores` a copy of the per-store results (`deleted`/`failed`, absent when unreported), as `HoldOutcomes` already returns for held artifacts (`records.go:234-238`). It is the only per-artifact source for the expiry, holds and deletion tabs, because `Preview` and `Impact` count across tenants and `RunView.Stores` folds excluded artifacts into `Pending` (`runs.go:321-331`).

Tests (`exports_test.go`): `ValidText` agrees with `PlaceHold` acceptance on the boundary cases; `Check` returns what `Propose` (approver empty) and `Approve` return for every refusal and error listed above and leaves proposals, policies, holds and runs unchanged; `Standing` across new, due, held (artifact and tenant holds, released holds), referenced (and after the requirer is deleted), pending, deleting, failed, damaged, deleted and no-policy artifacts, agreeing with `Preview`, `Access` and `HoldOutcomes`, and its `Stores` map is a copy (mutating it changes nothing); 100% statement coverage and `TestSourceRules` pass.

## Sample seed (only with `--seed sample`; one counted step `retention`)

**Actors**
- Humans: the first sample administrator (A1) and the second (A2) propose and approve as noted.
- The sample member places `sample-hold-temporary` (root decision 1), so A1 and A2, neither of them the placer, can complete the two-person release (`changes.go:160`). **Deviation from the [HB-257 plan](../HB-257/plan.md) seed rule** (`plan.md:106`: the seed acts as the bootstrap administrator or as named system actors): a non-administrator directory user acts in the seed. It happens only inside `seedRetention` (AST test); no request path lets a non-administrator place a hold.
- System actors built only inside `seedRetention`: `sample-retention-ingest` (register, refer) and `sample-retention-worker` (claim, proceed, report). They never appear in a request path. Claim tokens are discarded.
- Stores are the identifiers `sample-database` and `sample-object-store`. No bytes exist; every artifact is a labelled sample identifier.

**Steps, in order, with T the seed time** (tenants of every artifact = the boot workspace):
1. Register (ingest), anchor `T − age`:

   | Artifact | Class | Age (days) | Stores |
   | --- | --- | --- | --- |
   | `sample-source-1`, `sample-source-2` | source | 400 | both |
   | `sample-source-3`, `sample-source-4` | source | 400 | database |
   | `sample-source-5` | source | 300 | database |
   | `sample-source-6` | source | 250 | database |
   | `sample-source-7` | source | 30 | database |
   | `sample-final-1` | final | 100 | database |
   | `sample-claim-1` | claim | 500 | database |
   | `sample-temporary-1`, `-2`, `-3` | temporary | 20 | object store |
   | `sample-conversation-1` | conversation | 10 | database |
   | `sample-projection-1` | projection | 5 | object store |

2. Refer (ingest): `sample-final-1` requires `sample-source-3`.
3. Policies, each proposed by A1 and approved by A2: `source` 365 + 30 grace, `final` 3650 + 0, `temporary` 7 + 0. Values are synthetic, not legal periods. `claim`, `conversation`, `identity` and `projection` stay without a policy (never due).
4. A1 places `sample-hold-dispute`: owner A2, reason `Sample billing dispute hold`, artifacts `[sample-source-4]`.
5. Plan, proposed by A1 and approved by A2: class `source`, artifacts `sample-source-1..4`, chunk 2, lease 1 hour, reason `Sample expiry deletion` → `run/1`, revision 1: 1 and 2 `pending`, 3 `referenced`, 4 `held`.
6. Worker on `run/1`: claim chunk 0; proceed for 1 and 2; report `sample-database` deleted `[1, 2]`; report `sample-object-store` deleted `[1]`, failed `[2]` → `sample-source-1` deleted (tombstone 1), `sample-source-2` failed and damaged. Claim chunk 1 (nothing deleting, done). `run/1` reads `done` (one failure, one damaged).
7. Plan, proposed by A1 and approved by A2: class `temporary`, artifacts `sample-temporary-1..3`, chunk 1, lease 1 hour, reason `Sample temporary file expiry` → `run/2`. Worker claims chunk 0 and reports `sample-object-store` deleted `[sample-temporary-1]` → tombstone 2. Chunks 1 and 2 stay unclaimed: `run/2` reads `running`, two artifacts `pending` and blocked.
8. The sample member places `sample-hold-temporary`: owner the sample member, reason `Sample export review hold`, tenants `[boot workspace]`, classes `[temporary]`. Its counts show `not_preserved: 1` (`sample-temporary-1`), so the hold never reads as preserving deleted data.
9. Pending: A1 proposes the release of `sample-hold-temporary` (reason `Sample review finished`); A2 can approve it from the console.
10. Pending: A2 proposes `source` 180 + 30 (a shortening). The module stores its impact: `sample-source-5` and `sample-source-6` (count 2). A1 can approve it from the console with a reviewed impact.

**Run IDs.** After each plan approval the seed derives `run/<k>` from its own count of applied plans and confirms it with `Run(id)` whose plan equals the approved plan; a mismatch fails the seed. Requests never approve plans, so the count stays exact.

Nothing is deleted from any store, archived or exported; the deletion facts are module records about sample identifiers.

## Server store (`retentionseed.go`)

The module cannot list artifacts, holds, runs or proposals, so the server keeps, separate from the other stores:
- **Artifacts:** ID, class, tenants (copied at registration; `AddTenant` is never called, so the copy stays exact). Seed only.
- **Holds:** ID, workspace (the scope it was placed for), created time, rationale. Placed by the seed or `place_hold`.
- **Runs:** ID and workspace. Seed only.
- **Proposals:** number, kind, created time, proposer, rationale, closed by, closing rationale, and the last module view; kept like `installStore.keep` (at most 200, pending never dropped). `withdraw` keeps the entry as `withdrawn`.
- **`retired` is defensive only.** The module never removes a pending proposal behind the server: it evicts only applied ones (`registry.go:223-226`) and deletes a pending one only in its own `Withdraw` (`registry.go:245`), which the server performs and records. Should the module still answer `ErrNotFound` or `not_pending` for an entry the store holds as `pending` (reachable only by a direct module call in a test), the entry becomes `retired` and the command answers `invalid_stage`.

Facts that the module can read (policy revisions, hold state and counts, run state and outcomes, artifact standing, tombstones, watermark) are never stored.

**Hold IDs.** `place_hold` uses `retention-hold-<n>`. The counter advances on every attempt, whatever the outcome, and the server skips any `n` whose ID `Hold(id)` already knows, so an ID is never reused. A resend with the same idempotency key replays; a resend with a new key places a second hold, which is intended (holds are protective and cheap to release by the two-person flow).

**Visibility.** An artifact is visible to a workspace when the reader's workspace is the boot workspace and the artifact's tenants include it. A hold or run is visible when its stored workspace is the reader's. A proposal is visible when its policy class is valid (policies are installation-wide), its released hold is visible, or every artifact of its plan is visible.

## Read (admin only)

`GET /api/v1/retention`, admitted like the recovery read, under the domain read lock. For a workspace other than the boot workspace it returns `{revision, now, state: not_configured}`. For the boot workspace it always returns the full shape, with empty lists under `--seed none` (the console can still propose the first policy). Any module error makes the read answer `dependency_unavailable`, as in HB-286.

General rules:
- **Population.** Artifact counts cover the registered sample artifacts known to the server store and visible to the reader, not all data of the installation; every artifact count block carries `population: registered_sample_artifacts`.
- Counts are exact for that population and every enumerated key is present, so `0` is a real zero. A source the API does not have is the string `not_connected`, never `0` or an empty list. A value that does not apply is `null`.
- Every capped list carries `total` and `more`.
- **Run references.** A run is shown as `{target: "retention-run-<n>", label: "run/<n>"}`, the command target form plus the module ID, and only when the run is in the server run store for the reader's workspace. Otherwise the reference is `null` and the row carries `other_scope: true`. A shared artifact can be owned or deleted by another tenant's run (`registry.go:43`, `runs.go:217-228`).
- **Decision state.** Every run, every tombstone and every artifact whose access is `blocked` or `deleted` carries `decision: "not_connected"`: no independent durable decision record exists, so the screen cannot claim a confirmed deletion or a confirmed access block (P04 `spec.md:45,116`).
- Times are RFC 3339 UTC; `impact.as_of` keeps fractional seconds (see `approve`).
- People are shown as ID and login (`null` login when the directory does not know the ID).
- The read lists the boot workspace roster once with the admitted session's verifier (`s.roster(found.verifier)`, as `listServices` uses `s.grantees(found.verifier)` at `services.go:137-142`; `s.caller` is zero outside commands, `commands.go:199-200`) for the `no_eligible_approver` blocker; a test asserts the blocker reflects the real roster, not only the fault-hook path; a roster failure makes the read answer `dependency_unavailable`.

**`policies`** (seven rows, in `retention.Classes` order):
- `class`; `policy`: `{revision, period_days, grace_days, proposer, approver, approved_at}` or `null` (never due);
- `origin`: `artifact_anchor` (the module's immutable anchor); `scope`, `archive_rule`: `not_connected` (no jurisdiction, contract or hot/archive criteria in the module);
- `pending`: the pending policy proposal number for the class or `null`;
- `can_propose` with blocker `already_proposed`.

**`expiry`** (seven class groups):
- `counts` over the population: `eligible`, `held`, `referenced`, `not_due`, `no_policy`, `in_run`, `deleted`;
- `artifacts` (at most 50 per class by ID, with `total` and `more`): `id`, `anchor`, `due_at` (`null` without a policy), `stores`, `access` (`ok`, `blocked`, `deleted`), `standing` (one of the count keys, in `Preview`'s order: deleted, in run, no policy, not due, held, referenced, eligible), `held` (the hold flag on its own, because an artifact in a run or not yet due can also be held), `damaged`, `run` (a run reference or `null`, with `other_scope`), `shared` (true when another tenant is listed; no other tenant ID or count), `decision` when blocked or deleted, and `holdable` (false with blocker `target_too_long` when `retention-artifact:<id>` would exceed the 128-byte command target limit, that is an artifact ID over 109 bytes, `modules/commands/types.go:299-310`);
- `usage`, `growth`, `bytes`: `not_connected`.
- `Preview` is not used, because its held and referenced counts include other tenants.

**`holds`** (visible holds, newest first, at most 50, with `total` and `more`):
- `id`, `owner`, `reason`, `placer`, `placed_at`, `rationale`, `released`;
- `scope`: `artifacts` (IDs), `workspace` (true when the hold covers the reader's workspace as a tenant), `classes`;
- `counts`: `not_preserved`, `outcome_pending`, `partially_deleted`, all present;
- `outcomes`: `HoldOutcomes` at most 50 (`more` when a 51st exists): `artifact`, `outcome`, `stores` (store ID to `deleted`/`failed`);
- `release`: pending release proposal number or `null`; `can_propose_release` with blockers in order `released`, `already_proposed`, `no_eligible_approver` (fewer than two active members with an active user who also hold `AdministerInstallation` per `s.dir.holds(user, now)` (`directory.go:96-99`), other than the hold's placer, so no proposer and approver pair can exist), then `Check` (`placer`, `not_found`);
- `references`: visible artifacts whose standing has `Referenced` (ID and class only; the referrer is never named, because it may belong to another tenant).

**`deletion`** (visible runs, newest first, at most 50, with `total` and `more`):
- `run` (reference), `class`, `revision`, `state`, `decision: not_connected`, `next_chunk`, `chunks`, `chunk_size`, `lease_seconds`, `reason`, `damaged`, `proposer`, `approver`, `stopper` (or `null`);
- `counts`: all eight outcomes;
- `stores`: per store `target`, `deleted`, `failed`, `excluded`, `pending`. `RunView.Stores.Pending` counts every unreported artifact, including held, referenced, not-due and stopped ones (`runs.go:321-331`). The server splits it: `excluded` is the number of manifest artifacts listing the store whose outcome (`RunView.Outcomes`) is `held`, `referenced`, `not_due` or `stopped` and whose `Standing.Stores` has no result for the store; `pending` is the module's `Pending` minus `excluded`. Tests assert target = deleted + failed + excluded + pending (P04 `spec.md:61`);
- `artifacts`: at most 50 by ID with `outcome`, with `total` and `more`;
- `can_stop` with blockers `stopped`, `finished` (from the run state);
- `worker`, `retry`, `backup_expiry`, `propose_plan`: `not_connected`.

**`restore`:** `{state: not_connected}` (the module has no archive).

**`decisions`:**
- `watermark` (installation-wide, root decision 3);
- `tombstones`: `window: 1000` (only the last 1000 by sequence are scanned), of those the ones whose artifact is visible, newest first, at most 50, with `total` (visible in the window) and `more`: `sequence`, `artifact`, `class`, `run` (reference or `null` with `other_scope: true`), `at`, `decision: not_connected`;
- `ledger`: `in_memory`; `epoch`, `durable_commit`, `restore_check`: `not_connected`.

**`proposals`** (visible, newest first, at most 50, with `total` and `more`):
- `number`, `kind`, `state` (`pending`, `applied`, `withdrawn`, `retired`), `created_at`, `proposer`, `approver`, `rationale`, `closed_by`, `closing_rationale`;
- summary by kind: policy `{class, period_days, grace_days, impact_at_proposal: {as_of, count, digest} or null}`; plan `{class, artifact_count, chunk_size, lease_seconds, reason}`; release `{hold, reason}`;
- for a pending policy proposal that is a shortening now: `impact_required: true` and `impact_now` from `Impact(policy)`: `as_of`, `count`, `digest` (installation-wide, needed for approval; the only installation-wide impact figures), and `workspace_count` (`ImpactView.Tenants` for the reader's tenant only). `ImpactView.Held` and `.Referenced` count every tenant (`changes.go:66-88`) and are dropped; the server does not recompute the impact set itself, to avoid duplicating the module's due rule; otherwise `impact_required: false`, `impact_now: null`;
- `can_approve` with blockers in order: `not_pending`, `retired` (from the stored view, no module call); `not_console_kind` (plans); `own_proposal`; then `Check(change, proposer, reader, Impact{})` with `impact_required` treated as a requirement, not a blocker (`placer`, `released`, `not_found`);
- `can_withdraw` with blockers `not_pending`, `retired`, `not_proposer`.

**Never exposed:** claim tokens and lease deadlines, any identifier of another tenant (artifacts, holds, runs, tombstones, references, tenant IDs), per-tenant counts other than the reader's, any content (the module holds none), and any secret or key (none exist in this domain). The only installation-wide figures are the watermark and `impact_now.count`/`digest`.

## Commands (`retention.*`, reason required, actor = the reader as a human)

| Command | Target | Fields | Module call |
| --- | --- | --- | --- |
| `retention.approve` | `retention-proposal-<n>` (policy and release only) | `impact_as_of` (timestamp), `impact_count` (integer), `impact_digest` (text), all optional | `Approve(n, reader, impact)` |
| `retention.withdraw` | `retention-proposal-<n>` | none | `Withdraw(n, reader)` |
| `retention.propose_policy` | `retention-class:<class>` | `period` (integer, required), `grace` (integer, required) | `Propose(policy, reader)` |
| `retention.place_hold` | `retention-artifact:<id>` or `retention-tenant:<workspace>` | `owner` (ID, required), `reason` (text, required), `class` (enum of classes, optional) | `PlaceHold(hold, reader)` |
| `retention.propose_release` | `retention-hold:<id>` | `reason` (text, required) | `Propose(release, reader)` |
| `retention.stop_run` | `retention-run-<n>` (for `run/<n>`) | none | `Stop(run/<n>, reader)` |

**Permission (a deviation from P04).** Every command requires `AdministerInstallation` on the installation kind. P04 names separate policy-edit, policy-request, hold, deletion and evidence-access permissions; the gate has none. Definitions are built in three groups (proposals: approve, withdraw, propose_policy; holds: place_hold, propose_release; runs: stop_run), each from its own permission value.

**Resolve** (store only, no module call; not found unless the workspace is the boot workspace):
- a proposal number in canonical decimal whose entry is visible;
- a class in `retention.Classes`;
- a stored artifact visible to the workspace; a tenant equal to the workspace;
- a stored hold of the workspace; a run number in canonical decimal whose stored run is of the workspace.

Stored artifacts, holds and runs are never removed and closed proposals leave only by the 200 bound, so a resend of a successful command replays.

**Actors.** Every module actor is `{ID: ctx.User, Kind: retention.KindHuman}`. No command builds a system or assistant actor, registers, refers, claims, retries, reports, proceeds, records backup expiry or proposes or approves a plan.

**One mutation per command.** Each command makes exactly one mutating module call (`Propose`, `Approve`, `Withdraw`, `PlaceHold` or `Stop`), and it is the last fallible step: every precheck, roster read, module read (`Hold`, `Run`, `Policy`, `Check`) and fault hook runs before it, and nothing fallible runs after it (results come from the call's own return view). So a failure leaves no partial module change.

**Server prechecks** (fields only on `invalid_request`; rationale over 256 bytes is `invalid_request` without fields):
- `approve`:
  - the stored entry must be `pending`, else `invalid_stage` (covers the silent repeat approval, `registry.go:209-210`, and `retired`/`withdrawn`);
  - the kind must be `policy` or `release`, else `invalid_stage` (`not_console_kind`), decided without a module call;
  - the impact fields are all present or all absent; a missing or malformed one is `invalid_request` on that field (`impact_digest` 64 lowercase hex, `impact_count` ≥ 0); on a release they are `invalid_request` on the first given field;
  - the impact passed is what the reader sent, never recomputed by the server, so the module's review rule binds the approver. A stale or mismatched impact is `invalid_stage` (`impact_required`, `impact_changed`).
- `withdraw`: the stored entry must be `pending`, else `invalid_stage`.
- `propose_policy`:
  - `period` 1 to 36,500 and `grace` 0 to 365, else `invalid_request` on the field;
  - a pending policy entry for the class from anyone is `invalid_stage` (`already_proposed`); this also covers the module's silent duplicate (`registry.go:173-179`);
  - values equal to the current revision are `invalid_stage` (`unchanged`); a server rule, because the module would append an identical revision.
- `place_hold`:
  - `owner` must be an active member of the boot workspace with an active user (root decision 5): it must appear in `s.grantees(s.caller)` (`access.go:129-141`, used the same way for service owners at `services.go:278`), else `invalid_request` on `owner`. A roster failure answers `dependency_unavailable` (`identity.ErrUnavailable`, `admin.go:84`; precedent `access_test.go:331-334`);
  - `reason` must pass `retention.ValidText`, else `invalid_request` on `reason`;
  - `class` only with a tenant target, else `invalid_request` on `class`;
  - the hold is `{ID: retention-hold-<n>, Owner, Reason, Artifacts: [id]}` or `{…, Tenants: [workspace], Classes: [class] or none}`. Placing is never refused for covered state, matching the module; the result says what is not preserved.
- `propose_release`:
  - `reason` must pass `ValidText`, else `invalid_request` on `reason`;
  - `Hold(id)` released is `invalid_stage` (`released`); a pending release entry for the hold from anyone is `invalid_stage` (`already_proposed`). A placer proposing is refused by the module (`placer` → `invalid_stage`).
- `stop_run`: `Run(id)` state must be `running`, else `invalid_stage`. On `stopped` the module is silent and on `done`/`complete` it would only record a stopper (`runs.go:260-273`).

**Error mapping**

| Module result | Code |
| --- | --- |
| `Refusal` `self_approval` or `not_proposer` | `prerequisite_failed` |
| `Refusal` `ai_not_allowed`, `not_human` or `not_system` | `internal_error` (unreachable) |
| Any other `Refusal` (`not_pending`, `impact_required`, `impact_changed`, `placer`, `released`, `in_plan`, `already_deleted`, …) | `invalid_stage` |
| `ErrNotFound` after the prechecks (defensive; the entry becomes `retired` when stored as pending) | `invalid_stage` |
| `ErrConflict` (hold ID collision, only by injection) | `invalid_stage` |
| `ErrLimit` | `prerequisite_failed` |
| `ErrInvalid` | `internal_error` (the server built a bad input) |

These codes carry no fields; the read's blockers explain them. A `not_pending` refusal for an entry stored as `pending` also marks it `retired`.

**Revision on `internal_error`.** `internal_error` is journaled, so the first answer counts as a change and moves the revision (`commands.go:155-163`: a mutating, non-replayed result with an operation); a resend with the same key replays it and leaves the revision unchanged. With the one-mutation invariant above, that first move never accompanies a partial module change.

**Results.**
- Proposal commands: `fields` `proposal`, `target`; `business_state` the proposal state.
- `place_hold`: `fields` `hold`, `target`, `not_preserved`, `outcome_pending`, `partially_deleted` (decimal); `business_state` `held` when all three are zero, else `partially_preserved`.
- `stop_run`: `fields` `run`, `target`, `stopped` (count); `business_state` the run state after the stop.

# Not connected

Shown as `not_connected` in the read and absent from the commands:
- archive, hot/archive tiering and archive restore (the `restore` tab);
- usage, growth and byte counts;
- deletion plan proposal and approval (approval blocks access at once, `changes.go:179-188`, and no worker exists to finish a run);
- the deletion worker: claim, proceed, report and retry;
- backup expiry records (an attestation needing a backup catalog; runs never reach `complete` here);
- policy scope by jurisdiction or contract, and hot/archive criteria;
- an independent durable decision ledger, epoch and restore freshness checks.

# Residuals for the owner

- Retention periods, grace and hold accountability are owner and legal decisions; seeded and console-proposed values are synthetic.
- Plans, workers, retries and backup expiry need storage adapters and a backup catalog; `run/2` stays `running` until stopped, and `run/1` stays `done`.
- The watermark and the policy impact count and digest are installation-wide and shown to the boot workspace administrator; a multi-workspace design must scope them.
- Tombstone sequence gaps in the decisions tab reveal how many deletions of other tenants happened between visible ones; acceptable for the single pre-launch workspace, to be scoped with the watermark.
- With only two administrators, a hold placed by one of them through the console can never be released (`no_eligible_approver`); only the seeded member-placed hold shows the full release flow.
- The sample member acting in the seed deviates from the HB-257 seed rule (seed only).
- Real retention periods may need to stay read-only until the owner decides them (root decision 2); console proposals use synthetic values.
- `AdministerInstallation` for every command is a recorded deviation; P04's separate permissions and AU read access are later work.
- A hold on artifacts of another tenant, references across tenants and `AddTenant` are not exercised from requests; a referrer is never named.
- Persistence: the register, store and journal reset on restart. Closed proposal entries beyond 200 no longer resolve.
- `docs/ui/screen-binding-register.md` is not updated; the catch-up is HYB-295 (corrected by HYB-291).
- RET-01 to RET-08 remain NOT_RUN at system level.

# Verification plan

- **Module:** `exports_test.go` as under Module exports; `go test -cover ./modules/retention/` at 100%; `TestSourceRules`.
- **API read:**
  - shape with the sample seed; boot workspace with `--seed none` (seven policy rows with `null`, empty lists, `not_connected` sources); another workspace `not_configured`; non-admin member `permission_denied`;
  - every value against the module, read directly by the test (not through a fault hook): `Policy`, `Standing`, `Hold`, `HoldOutcomes`, `Run`, `Tombstones`, `Watermark`, `Proposal`;
  - standing order matches `Preview` for visible artifacts; `held` shown separately (`sample-source-4` reads `standing: held, held: true`; `sample-temporary-2` and `-3`, pending in `run/2` and covered by `sample-hold-temporary`, read `standing: in_run, held: true`); counts per class and per run balance; all keys present; `population`, `total` and `more` on every list, `window: 1000` on tombstones;
  - per-store counts: `run/1` `sample-database` reads target 4 = deleted 2 + failed 0 + excluded 2 + pending 0, and `sample-object-store` target 2 = deleted 1 + failed 1; `run/2` after the stop reads its two stopped artifacts as `excluded`; target = deleted + failed + excluded + pending on every store;
  - `decision: "not_connected"` on every run, every tombstone and every blocked or deleted artifact (`sample-source-1`, `-2`, `sample-temporary-1..3`), and absent on `ok` artifacts;
  - run references use `{target: retention-run-<n>, label: run/<n>}`;
  - `sample-hold-temporary` shows `not_preserved: 1`;
  - blockers: `already_proposed`, `not_console_kind`, `own_proposal`, `placer` (A1 on `sample-hold-dispute`), `no_eligible_approver` (after suspending one administrator, and on a hold placed by an administrator), `released`, `retired`, `not_pending`, `not_proposer`, `stopped`, `finished`, `target_too_long` (an injected artifact with a 110-byte ID; 109 bytes stays holdable); `impact_required` with `impact_now` on the seeded shortening, carrying only `as_of`, `count`, `digest`, `workspace_count`;
  - a roster failure (`roster` fault hook) answers `dependency_unavailable`;
  - another tenant registered directly in the module and the store (an artifact only of that tenant, a shared artifact, its hold, its run, its tombstone, a reference from its artifact, and a shared artifact owned and then deleted by that tenant's run): only `shared: true`, `referenced`, `run: null` with `other_scope: true` (in expiry and on the tombstone) appear; nothing else of that tenant;
  - advancing the test clock past the impact window changes `impact_now` without a revision change; a read leaves every proposal, hold and run unchanged.
- **API commands:**
  - each command's success, with transitions asserted through module reads;
  - approve: the seeded shortening by A1 with the read's `impact_now`; with no impact (`impact_required`), a stale one (clock past 1 hour), a wrong digest or count (`impact_changed`); partial or malformed impact fields with field names; impact fields on a release;
  - release approve by A2; self-approval; placer approval; approval after withdraw; a repeat approval refused with the revision unchanged; plan approval (an injected pending plan) refused without a module call while its proposer can withdraw it;
  - propose_policy: ranges, `already_proposed` from either administrator including the same content, `unchanged`;
  - place_hold on an artifact (including a deleted one, reporting `not_preserved`), on the tenant with and without a class; an owner unknown, suspended, removed or of another workspace (`invalid_request` on `owner`), a roster failure through the `roster` fault hook (`dependency_unavailable`), bad reason, class on an artifact target;
  - hold IDs: a failed attempt still advances the counter; an ID already known to `Hold` (injected) is skipped; a resend with the same key replays, and a resend with a new key places a second hold;
  - propose_release: on a released hold, a duplicate, by the placer;
  - stop_run on `run/2` (running → done with two `stopped`), then again (`invalid_stage`), and on `run/1` (`finished`);
  - `retired` (defensive): a stored pending entry withdrawn directly in the module by its proposer, then `approve` and `withdraw` through the API answer `invalid_stage` and the entry reads `retired`; a stored pending entry approved directly in the module, then `approve` through the API with the module answering `not_pending`, likewise (the HB-286 injection approach);
  - `ErrLimit` from `maxMine` maps to `prerequisite_failed`;
  - unknown, non-canonical and other-tenant targets, other-workspace and non-admin callers: `permission_denied`;
  - idempotent replay after state changes; an `internal_error` (fault hook on the mutating call) moves the revision once, leaves the module unchanged, and its replay with the same key leaves the revision unchanged; a fault on each read before the mutating call leaves the module unchanged; stale revision;
  - the mapping table through a direct test, including unreachable codes.
- **Leak test.** The JSON of the read and every result contains no claim token or lease deadline field, no identifier or tenant ID of the injected other tenant (including its run that deleted a shared artifact), and no other tenant count. The only installation-wide figures present are the watermark and `impact_now.count`/`digest`; adding the other tenant's artifacts changes nothing in the read beyond the `shared`, `referenced` and `other_scope` flags listed above.
- **AST tests** over all non-test files in `apps/api/server`:
  - `retention.KindSystem` and `retention.KindAssistant` appear only in `seedRetention`; `retention.go` and `retentioncmds.go` build no non-human actor and contain no `"system"` or `"assistant"` literal;
  - calls on the `s.retention` field to `Register`, `Refer`, `Claim`, `Proceed`, `Report` appear only in `seedRetention`; `AddTenant`, `Retry` and `BackupExpired` appear nowhere;
  - outside `seedRetention`, no `retention.Change` literal sets `Kind` to `retention.ChangePlan`;
  - `SampleMember` is used as a retention actor only in `seedRetention`;
  - the registered `retention.*` IDs are exactly the six above, all mutating.
- **Fault hooks:** `retention-read`, `roster`, and `retention-<call>` for every module call the server makes (propose, approve, withdraw, check, standing, artifact, policy, impact, hold, hold_outcomes, place_hold, run, stop, tombstones, watermark, register, refer, claim, proceed, report). The server never calls `Proposal` (the store keeps each call's returned view), so it has no hook; seed failure at each step; concurrent reads, commands and `Check`/`Standing` calls under the race detector.
- **Runs:** `go test -race -count=40 -timeout 45m -run Retention ./apps/api/server`; cross builds linux/amd64, windows/amd64, darwin/arm64; 100% statement coverage of the new server files and of `server`; mutation testing of the three new server files and `exports.go`, with survivors killed or shown equivalent with a reason.
- **Checks:** `gofmt`, `go vet`, the 1,500-line limit, no comments.
- **Reviews:** independent design and change review; `verification.md`.

# Assumptions and root decisions

Assumptions (not verified in code):
- The P04 spec citations come from the authorized local workspace copy, not this tree.
- The read uses the roster with the admitted session's verifier and `place_hold` with `s.caller` (set only inside commands); `no_eligible_approver` counts active members with an active user who hold the `AdministerInstallation` grant (`s.dir.holds`), not `MemberInfo.Role`.

Root decisions (design review round 1), folded into the body above:
1. The sample member places `sample-hold-temporary`, in the seed only, so the two sample administrators can complete the two-person release. No third administrator is added (Sample seed, Actors; residuals).
2. Policy proposals from the console are allowed with synthetic values only; whether real retention periods stay read-only until decided is an owner residual.
3. Installation-wide figures (watermark, impact count and digest) are shown only to the boot workspace administrator; other workspaces get `not_configured`.
4. The server keeps refusing a policy proposal identical to the current policy (`unchanged`).
5. A hold owner must be an active member of the boot workspace with an active user, not merely a directory login; otherwise `invalid_request` on `owner`.
