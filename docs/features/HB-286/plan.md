---
type: ImplementationPlan
title: P03 recovery API over modules/recovery with a synthetic sample installation in recovery and a locked standby installation
description: Phase 5e of the pre-launch plan. Adds a recovery read for the five P03 tabs (restore sets, isolation, external reconciliation, customer migration, resume decision) and administrator commands with a reason to approve unlock proposals, withdraw recovery proposals, lock an installation and propose an unlock. Nothing restores, prunes or deletes data, contacts a network or acts as a writer or verifier after the seed; customer migration is shown as not connected.
status: final
---

# Scope and ownership

[HB-286](https://linear.app/hybrid-billing/issue/HB-286) adds the API behind P03 (recovery, migration and execution state). Base: main `c7b4f3e`. The P03 screen follows in a separate ticket. The plan follows the merged [HB-282 plan](../HB-282/plan.md), [HB-284 plan](../HB-284/plan.md) and their verification records.

**Ownership**
- Root (Claude Code) owns the plan and integration. Implementation may be delegated to one writer in this worktree.
- An independent read-only reviewer reviews the design and the change, with a security focus (actor kinds, two-person rules, tenant scoping, no restore, prune or delete effects, no exposure of data-derived checksums or tokens).
- Machine: mac_mini.

**Allowed files**
- `modules/recovery/exports.go` (new): four exported wrappers, plus `exports_test.go`. No rule changes:
  - `ManifestDigest(manifest []Inventory) string` returns `manifestDigest(manifest)`;
  - `LostDigest(lost []uint64) string` returns `lostDigest(lost)`;
  - `(*Registry) Check(c Change) error` is a dry run of the checks `Propose` and `Approve` run (described under Design);
  - `(*Registry) Status(id string) (state string, generation uint64, stoppedAt time.Time, err error)` is a read of an installation's state, generation and stop time.
- `apps/api/server`:
  - new `recovery.go` (read), `recoverycmds.go` (commands and error mapping), `recoveryseed.go` (seed and server store), and their tests;
  - registration in `admin.go` (`registerUsers`), the route in `boundary.go`, the registry in `server.go`, the seed step in `seed.go`;
  - the seed step count in `world_test.go` (14 to 15).
- `docs/features/HB-286/*`.
- No other module (`modules/migration`, `modules/balances`, `modules/exports` included), no console file and no `screens.json` change.

# Observed baseline

## `modules/recovery`

- **Registry.** `NewRegistry(clock)` takes only a clock and refuses a nil one (`registry.go:131-136`). There is no `Limits` value: every bound is a module constant (`registry.go:8-32`): installations 100000, writers 16 and externals 16 per installation, 1000 live permits per writer, 1000 open calls per writer and 10000 per installation, 10000 closed calls, 100 history rows, 1000 pending proposals per scope, 1000 pending begins in total, skew ≤ 1 hour, writer `MaxCall` 1 second to 1 hour, 100 mismatch indices.
- **Identifiers and text.** IDs use `[A-Za-z0-9._:-]`, at most 128 bytes (`common.go:15-26`). Free text is 1 to 200 bytes of valid UTF-8 without C0, DEL or C1 controls (`common.go:28-38`). Digests are 64 lowercase hex characters (`common.go:40-51`).
- **Actors.** `human()` returns `ErrInvalid` for a malformed actor, `ai_not_allowed` for an assistant and `not_human` for a system actor (`registry.go:151-165`).
- **Changes are proposals.** `Propose(change, human)` then `Approve(number, other human)` (`registry.go:186-260`).
  - Kinds: `installation` (ID, tenant, skew), `writer` (ID, kind `erp`/`issuance`/`delivery`/`payment`/`message`, `MaxCall`), `external` (ID, kind), `verifier` (ID), `begin` (a restore point), `unlock` (generation and a lost digest), `prune` (a time), `abandon` (recovery ID and generation) (`types.go:21-28`, `changes.go:71-106`).
  - A scoped proposal needs the installation to exist (`ErrNotFound`, `registry.go:196-198`).
  - The same proposer proposing the same change returns the existing pending proposal (`registry.go:200-205`). One proposer may hold at most `(1000+3)/4` pending proposals per scope, and an installation may hold one pending `begin` (`ErrLimit`, `registry.go:209-217`).
  - `Approve` refuses an unknown number (`ErrNotFound`), a non-pending proposal (`not_pending`) and self-approval (`self_approval`), re-runs `check` and applies (`registry.go:228-260`). **A repeat approval by the same approver of an applied proposal returns the view without an error** (`registry.go:239-240`).
  - Applied proposals beyond 1000 per scope are deleted (`registry.go:254-258`).
  - `Withdraw` is limited to the proposer (`not_proposer`) and a pending proposal (`not_pending`), and deletes the proposal (`registry.go:262-283`).
  - `Proposal(number)` is the only proposal read; there is no list (`registry.go:285-293`).
  - Every generation bump deletes the installation's pending `begin` proposals (`registry.go:269-272`, `registry.go:311-324`); a later `Approve` of such a number is `ErrNotFound` (`flow_test.go`, `TestBeginRules`).
- **Checks** (`changes.go:137-208`):
  - `installation`: `ErrConflict` for a taken ID; `writer` and `verifier`: `identity_taken` for an ID used anywhere (identities are never released, `changes.go:132-135,236-246`); `external`: `ErrConflict` for a taken ID.
  - `begin`: `ErrInvalid` for an as-of in the future, `generation`, `manifest_mismatch` (the stated digest is not `manifestDigest(manifest)`), `open_recovery`, `pruned`. The restore point needs the nine domains in the module's order, each a contiguous chunk list with 64-hex checksums (`changes.go:13-16,18-30,59-69`).
  - `unlock`: `generation`, then `checkUnlock` (`recovery.go:434-470`): without a recovery, `not_recovering` unless `locked` and `open_calls` while calls are open; with one, `no_restore_point`, `incomplete` (an external not final, a domain not verified or an event pending), `lost` (the digest is not `lostDigest` of the lost sequences), `kind_coverage` (a writer kind, removed writers included, without an external of that kind).
  - `prune`: `ErrInvalid` for a future time, `open_recovery`. `abandon`: `no_restore_point`, `generation`.
  - `checkUnlock` calls `reclaim`, which closes calls past their deadline as `unknown` (`recovery.go:435`, `calls.go:19-25`); `Recovery` does the same on every read (`recovery.go:479`).
- **Application** (`changes.go:228-267`): an installation starts `serving` at generation 1; `begin` stops writers, bumps the generation, enters `recovering`, fences open and recent confirmed or unknown calls and lists unconfirmed targets as unknown (`recovery.go:41-89`); `unlock` records history and returns to `serving` without a bump (`recovery.go:105-111`); `abandon` bumps, records history and drops the restore point while the state stays `recovering` (`recovery.go:98-103`, `TestAbandonAndPrune`); `prune` deletes decision events up to a time (`changes.go:251-263`).
- **Direct human operations.**
  - `Lock(id, reason, human)` needs valid text but does not store it. **On a `locked` installation it returns nil without a change**; on `recovering` it refuses `recovering`; otherwise it bumps the generation, enters `locked` and records the stop (`calls.go:171-195`).
- **System paths.** `Permit`, `Call` and `Done` need the writer's own system actor (`calls.go:41-47,56-153`); `Report`, `Reconcile`, `Reapplied` and `Unapplicable` need the installation's verifier system actor (`recovery.go:113-130,132-422`); `Record` needs a writer or the verifier as a system actor (`recovery.go:293-315`). Final reconciliation needs pages covering from before the as-of to at least the stop time plus the longest `MaxCall` plus skew, and not after the module clock (`recovery.go:253-262`), so it cannot complete at the instant of a seed.
- **Reads.** `Recovery(id)` returns the current recovery (state, generation, stop time, domains, externals, pending events, lost sequences, open calls, unknown targets, skipped count, phase, restore point fields) or `ErrNotFound` when none is open (`recovery.go:472-517`). `History(id)` (`recovery.go:519-527`), `HighWater(id, workspace, series)` and `Fenced(id, workspace, key)` (`calls.go:222-246`). There is no read of an installation's state, generation, writers, externals or verifier outside a recovery.
- **Errors:** `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrLimit`, and `Refusal{Reason}` wrapping `ErrRejected` (`types.go:8-14,81-117`).
- **Fixtures** (`world_test.go:88-149`): `ann` proposes and `bob` approves; writers `w-erp` (1 minute) and `w-iss` (2 minutes), verifier `v-1`, externals `erp-x` and `iss-x`; manifests of nine domains with two chunks whose checksums are `sha256(domain+"a"|"b")`; the restore point digest is `manifestDigest(m)`; unlock uses `lostDigest(current.lost)`. `manifestDigest` and `lostDigest` are unexported, so the server cannot build a valid `begin` or `unlock` without wrappers; known answers are pinned in `boundary_test.go:8-19`.
- **Source rules** (`source_test.go:11-46`): no comments, no imports with a dot, at most 1,500 lines per file.

## Other modules for P03 tabs

- **Customer migration** (P03 tab `migration`): `modules/migration` imports legacy documents and keys through sources, importers (system actors, `not_importer`) and batches with an `Evidence` dependency (`modules/migration/types.go:21-125`); `modules/balances` records opening balances (`modules/balances/lots.go:20`). Neither is bound by the API (`apps/api/server` imports only `authpolicy`, `commands`, `contracts`, `events`, `gate`, `identity`, `installer`, `organizations`, `passwords`, `plugins`, `relationships`, `webhooks`). P03's migration package action exports a customer scope (P03 spec, action table). This ticket shows the tab as `not_connected`; see Out of scope.

## Server and commands

- Commands run under the server's domain write lock and the revision moves on every completed mutating command (`commands.go:196-206`).
- `Resolve` receives no actor and runs before the idempotency journal lookup (`modules/commands/execute.go:307` before `execute.go:342-353`). A target that no longer resolves is `permission_denied` (`execute.go:330-332`), so a resend after a lost response replays only while the target still resolves. Replays skip the revision check (`execute.go:350-358`). Failures with a handler code are not journaled (`execute.go:399-418`); `internal_error` is not a handler code, so for a mutating command it is journaled and a resend replays it (`execute.go:420-425`).
- Only `invalid_request` carries fields, and only declared field names (`execute.go:399-413`). Handler codes are `unsupported_capability`, `invalid_stage`, `revision_conflict`, `prerequisite_failed`, `dependency_unavailable`, `permission_denied` and `invalid_request` (`modules/commands/types.go:71-74`). The command layer limits rationale to 2000 bytes (`execute.go:70`, `server.go:114`); the server caps it at `maxRationale` = 256 (`directory.go:13`).
- Admin reads use `admitted`, which requires `AdministerInstallation` for the console client (`users.go:42-67`). The gate has no recovery-specific action (`modules/gate/types.go:45-46`).
- Fault hooks use `faulty` (`admin.go:78`) through `moduleCall` (`integrationseed.go:174-179`).
- The sample seed runs one counted step per domain (`seed.go:86-113`) and the test asserts 14 steps (`world_test.go:57`). The sample workspace is `sample-workspace` (`seed.go:20`), a valid module ID.
- The HB-282 and HB-284 AST tests cover only their own files and package qualifiers (`installation_edges_test.go:192-217`, `integration_edges_test.go:415`).

## P03 design

The P03 design (`docs/ui/screens/P03/spec.md` in the authorized local workspace; not in this repository tree) names the tabs restore sets, isolation check, external reconciliation, customer migration and resume decision (`spec.md:80`), the columns restore ID, as-of, generation, evidence check, decision watermark, external reconciliation and public lock (`spec.md:60`), and five actions with separate recovery verification, recovery execution, customer-scope export, ERP reconciliation and resume permissions (`spec.md:70-76`). Its audience is OP, AU and permitted BL (`spec.md:21`). RPO/RTO, the decision ledger and the migration format are open (`spec.md:132-136`).

# Design

## Registry and limits

`recovery.NewRegistry(s.clock)` built in `server.New` next to the installer. The module has no configurable limits; the sample uses two installations, three writers, two externals, one verifier, two calls, two events and ten proposals, far inside the constants in `registry.go:8-32`. Server bounds: the proposal store keeps at most 200 entries (pending ones never dropped) and the read shows the newest 50; open calls, unknown targets, pending events and history rows are each shown up to 50 with a total count.

## Exported recovery wrappers (`modules/recovery/exports.go`)

- `func ManifestDigest(manifest []Inventory) string { return manifestDigest(manifest) }` (seed only, to build the sample restore point).
- `func LostDigest(lost []uint64) string { return lostDigest(lost) }` (the read shows the digest the unlock proposal must confirm).
- `func (r *Registry) Check(c Change) error`:
  - under the module lock, with the module clock, `clean(c)`; `ErrInvalid` when it is malformed;
  - `ErrNotFound` when a scoped change names an unknown installation (the same guard as `registry.go:196-198`, which also keeps `check` from reading a nil installation);
  - otherwise `r.check(c, now)`.
  - It creates, changes and deletes no proposal and no installation fact. The only effect is `reclaim` inside `checkUnlock`, which `Recovery` also performs on every read.
  - It skips what `Propose` does around `check`: the duplicate return for the same proposer and content and the proposal limits (`registry.go:200-217`). A command the read shows as allowed may therefore still answer `prerequisite_failed` (`ErrLimit`).
- `func (r *Registry) Status(id string) (string, uint64, time.Time, error)`: under `enter`/`leave`, `located(id)` (`ErrInvalid` for a malformed ID, `ErrNotFound` for an unknown one), then the site's `state`, `generation` and `stoppedAt`. No `reclaim`, no other effect. It is the authority for state, generation and stop time in the read, the lock precheck and the unlock generation.
- A reviewed module change with tests showing that:
  - the wrappers equal the pinned known answers in `boundary_test.go:8-19`;
  - `Check` returns what `Propose` (for a new change) and `Approve` (for a pending one) return, for every refusal listed in the baseline, and changes no proposal view, history, generation or state;
  - `Status` returns `serving`/1 for a new installation, `locked` with a bumped generation and the lock time after `Lock`, `recovering` after `begin`, `serving` with a cleared stop time after `unlock`, and `ErrInvalid`/`ErrNotFound` for malformed and unknown IDs, with no state change;
  - `modules/recovery` keeps 100% statement coverage and passes `TestSourceRules`.

## Sample seed (only with `--seed sample`; one counted step `recovery`)

**Actors and seed-only identities**
- The first sample administrator proposes and the second approves, unless noted.
- System actors built only inside `seedRecovery`: the writer `sample-erp-writer` and the verifier `sample-verifier`. They never appear in a request path.
- `sample-issuance-writer` and `sample-standby-writer` are registered identifiers only and never act.
- No key material exists in this domain; digests are SHA-256 hex of fixed synthetic labels.

**Installation `sample-primary`** (tenant = the boot workspace, skew 1 minute), approved, then:
- writers `sample-erp-writer` (`erp`, 1 minute) and `sample-issuance-writer` (`issuance`, 2 minutes); verifier `sample-verifier`; externals `sample-erp-ledger` (`erp`) and `sample-issuance-register` (`issuance`), each proposed and approved;
- writer activity by `sample-erp-writer`: one permit; a call on (boot workspace, `sample-invoice-1`, series `sample-invoice-series`, number 1) reported `confirmed`; a call on (boot workspace, `sample-invoice-2`, series `sample-invoice-series`, number 2) left open. The permit token is discarded;
- decision events: `delete` on domain `evidence`, key `sample-bundle-1`, recorded by `sample-erp-writer`; `revoke_access` on domain `database`, key `sample-user-1`, recorded by `sample-verifier`;
- restore point: generation 1, backup reference SHA-256 hex of `sample-backup-1`, as-of the seed time minus 1 hour, manifest of the nine domains in module order, each with 10 items in chunks [0,4) and [4,10) whose checksums are SHA-256 hex of `sample-<domain>-a` and `sample-<domain>-b`, digest from `recovery.ManifestDigest`. Proposed and approved as `begin`: `recovery-1`, `recovering`, generation 2; the confirmed call is fenced, and the open call is fenced and listed unknown;
- verifier reports for the nine domains: eight match; `outbox` is reported with the second chunk checksum of `sample-outbox-tampered`, so it reads `mismatch` at chunk 1. The recovery stays in phase `verify` with two pending events, both externals pending and one unknown target.

**Installation `sample-standby`** (tenant = the boot workspace, skew 1 minute), approved, then writer `sample-standby-writer` (`erp`, 1 minute), approved; `Lock` by the first administrator (generation 2, `locked`, no open calls).

**Pending proposal:** unlock of `sample-standby` at generation 2 with `recovery.LostDigest(nil)`, proposed by the second administrator (the first can approve). Approval returns it to `serving`; the lock command can then lock it again, so the full two-person `lock_release` flow works on synthetic data.

The seed discards every permit token and call view. Nothing is restored, reconciled, pruned or sent; every value is labelled sample data.

## Server store (`recoveryseed.go`)

The module has no list read, so the server keeps, separate from the HB-282 and HB-284 stores, only what the module cannot list. State, generation and stop time are not stored; they come from `Status`.
- **Installations:** ID, tenant, skew; writer IDs; externals (ID, kind); verifier ID; a lock log (by, rationale, time).
- **Proposals:** number, kind, installation, created time, proposer, rationale, closed by, closing rationale, and the last module view (with the restore point manifest kept server side only). Kept like `installStore.keep`.
- **Store rules.** The server updates the store only after the module call succeeds, with one exception:
  - `Approve` of `installation`, `writer`, `external` or `verifier` (seed only) adds the record;
  - `Lock` appends to the lock log;
  - `Withdraw` keeps the entry as `withdrawn`;
  - the exception: when `Approve` or `Withdraw` of a pending entry answers `ErrNotFound` (the module deleted the proposal, `registry.go:254-258,311-324`), the entry is marked `retired` and the command answers `invalid_stage`, so the read stops offering actions. `retired` arises only when a test injects a pending `begin` into the module, because no console path and no seed step leaves one pending.
- Installations are never removed from the store, and closed proposals leave only by the 200-entry bound, so a resend after a lost response replays (see Resolve).

## Read (admin only, boot workspace only)

`GET /api/v1/recovery` is admitted like the installation read. It shows only installations whose tenant equals the reader's workspace, and only in the boot workspace; proposals are shown only when their installation passes the same rule. For another workspace, or when no installation qualifies (`--seed none`), it returns `{revision, now, state: not_configured}`. Otherwise it returns `revision`, `now`, `proposals` and one section per tab, each with its own `state`. State, generation and stop time come from `Status`.

**`sets`** (one row per installation):
- installation ID, state, generation;
- `recovery`: ID, phase, restore point as-of, point generation, backup reference and manifest digest (64-hex integrity values that identify a backup; they are not credentials or contents), and per domain: domain, status (`pending`, `verified`, `mismatch`) and mismatch chunk indices; or `none` when no recovery is open;
- `history`: recovery ID, manifest digest, generation, outcome, lost count;
- `begin` and `abandon`: `not_connected` (no restore-set intake; see Out of scope).
- Manifest chunk checksums, item counts per chunk and the restore point from proposal views are never returned.

**`isolation`** (one row per installation):
- state, generation and stop time (the module's in-flight horizon has no read and is not shown);
- `can_lock` with blockers `locked`, `recovering`;
- writers: identifiers only;
- verifier ID (an identifier, not a credential);
- open calls: ID, writer, deadline, and the target (key, series, number) only when its workspace is the boot workspace; calls on other workspaces are omitted and only the boolean `other_scope_present` says that some exist (no count). The installation-wide view is an administrator scope; a count per foreign workspace is not shown;
- `permits`: `not_shown` (permits exist in the module but are never read or returned).

**`reconciliation`** (one row per installation with an open recovery):
- externals: ID, kind, status (`pending`, `reconciled`);
- unknown targets scoped as above, each with `fenced` from `Fenced` and, for a series, the high-water number from `HighWater`;
- skipped count;
- `reconciler`: `not_connected`.

**`migration`:** `{state: not_connected}`, with `modules: ["migration", "balances"]` as the modules a later ticket binds.

**`resume`** (one row per installation):
- `mode`: `recovery_resume` while a recovery is open, `lock_release` for a `locked` installation without one (an unlock that ends a plain lock, not a recovery resume), `none` when `serving`;
- phase (`verify`, `reapply`, `ready`, or `not_recovering`);
- pending decision events: sequence, kind, domain, time (no key);
- lost sequences and `lost_digest` from `recovery.LostDigest` (of the lost list, or of none without a recovery);
- `can_propose_unlock` with blockers, in order: `already_proposed` (a pending unlock entry for the installation exists, by anyone); then the result of `Check` for the unlock the reader would propose (generation from `Status`, the shown digest): `not_recovering`, `open_calls`, `no_restore_point`, `incomplete`, `lost`, `kind_coverage`, `generation`, or `not_found`/`invalid` for those errors.

**`proposals`** (newest first, at most 50):
- number, kind, installation, state (`pending`, `applied`, `withdrawn`, `retired`), created time, proposer and approver (ID and login), rationale, closed by and closing rationale;
- a summary: installation ID and skew; writer ID; external ID and kind; verifier ID; restore point as-of, generation and manifest digest; unlock generation and lost digest;
- `can_approve` and `can_withdraw` with blockers: `not_pending` and `retired` from the stored view (no module call for a non-pending entry); `not_console_kind` for approval of any kind other than `unlock`; `own_proposal` and `not_proposer`; for other pending entries the result of `Check` on the stored change.

No row contains a manifest chunk checksum, a permit token, a decision event key, an item list or any target outside the boot workspace.

## Commands (`recovery.*`, reason required, actor = the reader as a human)

| Command | Target | Fields | Module call |
| --- | --- | --- | --- |
| `recovery.approve` | `recovery-proposal-<n>` (unlock proposals only) | none | `Approve(n, reader)` |
| `recovery.withdraw` | `recovery-proposal-<n>` | none | `Withdraw(n, reader)` |
| `recovery.lock` | `recovery-installation:<id>` | none | `Lock(id, "Administrator lock", reader)` |
| `recovery.propose_unlock` | `recovery-installation:<id>` | `lost` (required text) | `Propose(unlock{id, generation, lost}, reader)` |

**Permission (a deviation from P03).** Every command requires `AdministerInstallation` on the installation kind. P03 names separate verification, execution, export, ERP reconciliation and resume permissions (`spec.md:70-76`); the gate has no such actions. Definitions are built in two groups (proposals: approve, withdraw, propose_unlock; containment: lock), each from its own permission value, so a later split is one line per group.

**Resolve.** `Resolve` receives no actor, so it reads only the server store and calls no module. It returns not found unless the workspace is the boot workspace and the target is in the store:
- a proposal number in canonical decimal, whose installation's tenant equals the workspace;
- an installation ID whose tenant equals the workspace.

Because installations are never removed and closed proposal entries are kept (bounded at 200), a resend of a successful command resolves and replays. `Apply` then calls the module with the human reader.

**Actors.** Every module actor is `{ID: ctx.User, Kind: recovery.KindHuman}`. No command builds a system or assistant actor. There is no command that permits, calls, records, reports, reconciles, reapplies, declares an event unapplicable, begins, abandons, prunes or registers an installation, writer, external or verifier.

**Lock reason.** The module validates the lock reason but does not store it (`calls.go:177-179`). The server passes the fixed text `Administrator lock`, so a rationale that is valid for the command layer but not module text can never produce `ErrInvalid`; the rationale is kept in the lock log.

**Server prechecks** (fields only on `invalid_request`):
- rationale over 256 bytes: `invalid_request` without fields;
- `approve`: the stored change kind must be `unlock`, decided from the stored view without a module call; otherwise `invalid_stage` (`not_console_kind`). Proposals of other kinds come only from the seed or from tests; `withdraw` stays available for any kind to its proposer.
- `propose_unlock`: `lost` must be 64 lowercase hex characters (`invalid_request` on `lost`); when no recovery is open (`lock_release`), `lost` must equal `recovery.LostDigest(nil)` (`invalid_request` on `lost`). The generation comes from `Status`.
- **Silent no-ops.** The module succeeds silently on three calls; the server refuses them first with `invalid_stage`, so the revision does not move:
  - `approve` on a non-pending entry (`registry.go:239-240`), decided from the stored view, including `retired` and `withdrawn`;
  - `lock` unless `Status` reads `serving`; on `locked` the module is silent (`calls.go:184-186`), on `recovering` it refuses;
  - `propose_unlock` while any pending unlock entry exists for the installation, from any proposer, including the same proposer with the same content, which the module would return silently (`registry.go:200-205`) (`already_proposed`).

**Error mapping**

| Module result | Code |
| --- | --- |
| `Refusal` `self_approval` or `not_proposer` | `prerequisite_failed` |
| `Refusal` `ai_not_allowed`, `not_human`, `not_writer` or `not_verifier` | `internal_error` (the server built a non-human actor or called a system path; unreachable) |
| Any other `Refusal` (`not_pending`, `locked`, `recovering`, `generation`, `not_recovering`, `open_calls`, `no_restore_point`, `incomplete`, `lost`, `kind_coverage`, `identity_taken`, …) | `invalid_stage` |
| `ErrNotFound` after the prechecks (a deleted or pruned proposal) | `invalid_stage` |
| `ErrConflict` | `invalid_stage` |
| `ErrLimit` | `prerequisite_failed` |
| `ErrInvalid` | `internal_error` (the server built a bad input) |

These codes carry no fields, so the read's blockers explain them. `internal_error` is journaled (`execute.go:420-425`): a resend with the same key replays it and the revision does not move.

**Results.** `fields` carry `proposal` and `target` for proposal commands, `installation` and `target` for `lock`; `business_state` is the proposal state or the installation state.

# Out of scope (residuals)

- **Restore and data effects.** Restore-set registration and inspection, isolated restore requests (`begin`), `abandon` (it leaves the installation `recovering` without a restore point, `recovery.go:98-103`, and only a new `begin` continues), and `prune` (it deletes decision events, `changes.go:251-263`). They need a backup catalog, a restore runtime and a decision on RPO/RTO.
- **Non-human paths.** Writer permits, calls and outcomes; verifier reports, reconciliation pages, reapplication and unapplicable declarations; decision event recording. They need authenticated writer and verifier identities. Registration of installations, writers, externals and verifiers from the console.
- **Sample limits.** The `sample-primary` recovery cannot progress after the seed: the verifier is not connected, and final reconciliation needs time to pass after the stop. Its resume stays blocked (`incomplete`); `sample-standby` carries the working two-person `lock_release` flow, not a recovery resume. Open calls past their deadline become `unknown` on the next read (`reclaim`) without a revision change.
- **Customer migration.** Binding `modules/migration` (legacy document import with importer system actors and an evidence dependency), `modules/balances` opening balances (QA-09), and the P03 customer-scope export package (customer data leaves the platform). The tab reads `not_connected`; a later ticket binds `modules/migration` read-only first.
- **Permissions and audience.** Separate gate actions for verification, execution, export, ERP reconciliation and resume; AU and BL read access. Until then the read and every command use `AdministerInstallation`.
- **Writer removal** is not exposed: single administrator and irreversible (`changes.go:132-135`; removed writers still count for `kind_coverage`); needs a two-person design.
- **Lock release.** An unlock that ends a plain lock (`lock_release`) checks only the generation, the `locked` state and open calls (`recovery.go:437-444`); it checks no decision watermark, domain verification or external reconciliation.
- **Persistence**, a decision ledger independent of the database, and network or host effects: none exist here.
- Closed proposal entries evicted after 200 no longer resolve, so a very late resend of a command on them answers `permission_denied`.

# Verification plan

- **Module:** `exports_test.go`:
  - the digest wrappers equal the known answers in `boundary_test.go:8-19`;
  - `Check` matches `Propose` and `Approve` across every refusal and error in the baseline, including `ErrNotFound` for an unknown installation without a panic, and leaves proposals, history, generation and state unchanged;
  - `Status` transitions and errors as described under Design;
  - `go test ./modules/recovery/` at 100% statement coverage; `TestSourceRules` passes.
- **API read:**
  - the shape with the sample seed and with `--seed none` (`not_configured`); an administrator of another workspace gets `not_configured` and a non-admin member `permission_denied`;
  - state, generation and stop time equal `Status` (and `Recovery()` while a recovery is open) after the seed and after each lock and unlock;
  - every blocker above, including `already_proposed`, `not_console_kind`, `retired`, the `Check` reasons and the `not_pending` rows built without a module call;
  - `resume.mode` is `recovery_resume` for `sample-primary`, `lock_release` for `sample-standby` and `none` after its unlock;
  - a call on another workspace (registered directly in the module) is omitted and sets `other_scope_present` without a count;
  - an installation with another tenant (registered directly in the module and the store) and its proposals are not shown;
  - `permits` reads `not_shown`;
  - advancing the test clock past the open call's deadline changes the isolation section (the call leaves the open calls) and the reconciliation section (the target stays unknown and fenced) without a revision change;
  - a read leaves every seeded proposal unchanged, compared through `Proposal(n)` before and after.
- **API commands:**
  - each command's success; self-approval; withdraw by a non-proposer; approve after withdraw; approve of an applied proposal by its approver refused without moving the revision;
  - the three silent no-ops refused as `invalid_stage` with the revision unchanged, including a second `propose_unlock` by the same administrator with the same content and one by the other administrator while an unlock is pending;
  - `approve` of a pending `begin` and a pending `prune` injected into the module refused as `invalid_stage` without a module call, while `withdraw` by their proposer succeeds;
  - a pending entry deleted in the module (an injected `begin` retired by `lock`) answers `invalid_stage` on approve or withdraw, becomes `retired` and loses its actions in the read;
  - `propose_unlock` on `sample-primary` refused with `invalid_stage` (`incomplete`), and on `sample-standby` after its unlock (serving) with `invalid_stage` (`not_recovering`);
  - `lock` on `recovering`; unlock with a wrong `lost`, with a stale generation, with open calls, and a full cycle lock → propose unlock → approve → lock on `sample-standby`, with each transition asserted through `Status`;
  - a malformed `lost` with the field name, and in `lock_release` a well-formed `lost` other than `LostDigest(nil)` with the field name; a long rationale;
  - `prerequisite_failed` from `ErrLimit` on `propose_unlock` although the read showed it allowed (proposal limits are outside `Check`);
  - unknown targets (non-canonical numbers, unknown installation, a `recovery-writer:` target), an installation or proposal of another tenant, and other-workspace and non-admin callers answer `permission_denied`;
  - idempotent replay of each successful command after the target's state changed (the target still resolves); an `internal_error` (fault hook) replayed with the same key and the revision unchanged; stale revision;
  - the mapping table, including the unreachable codes, through a direct test of the mapping function.
- **Leak test.** The JSON of the read and of every command result contains none of: manifest chunk checksums (hex, base64, raw), including the tampered `outbox` checksum, the `sample-<domain>-a`/`-b` and `sample-outbox-tampered` labels, permit tokens or sequences, decision event keys (`sample-bundle-1`, `sample-user-1`), or a target from another workspace.
- **AST tests** over all non-test files in `apps/api/server`:
  - `recovery.KindSystem` and `recovery.KindAssistant` appear only in `seedRecovery`; `recovery.go` and `recoverycmds.go` build no non-human actor;
  - calls on the `s.recovery` field to `Permit`, `Call`, `Done`, `Record`, `Report`, `Reconcile`, `Reapplied`, `Unapplicable` appear only in `seedRecovery`, and `Reconcile`, `Reapplied`, `Unapplicable` and `RemoveWriter` appear nowhere; `ManifestDigest` appears only in `seedRecovery`;
  - outside `seedRecovery`, no `recovery.Change` composite literal sets `Kind` to anything other than `recovery.ChangeUnlock`;
  - no `"system"` or `"assistant"` string literal in `recovery.go` or `recoverycmds.go`;
  - the registered `recovery.*` command IDs are exactly `recovery.approve`, `recovery.lock`, `recovery.propose_unlock`, `recovery.withdraw`, all mutating.
- **Fault hooks:** `recovery-read` and `recovery-<call>` for every module call (propose, approve, withdraw, lock, check, status, recovery, history, high_water, fenced, permit, call, done, record, report); seed failure paths for each step; concurrent reads, commands and `Check` calls under the race detector.
- **Runs:**
  - `go test -race -count=40 -timeout 30m -run Recovery ./apps/api/server`;
  - cross builds for linux/amd64, windows/amd64 and darwin/arm64;
  - 100% statement coverage of the new server files, and `server` overall;
  - mutation testing of the three new server files and `exports.go`; survivors are killed by a test or shown to be equivalent, with a reason.
- **Checks:** `gofmt` and `go vet` clean; the 1,500-line limit per file holds.
- **Reviews:** independent design and change review; `verification.md`.

# Design review round 2 clarifications

- `propose_unlock` checks the installation state before the `lost` digest: a `serving` installation answers `invalid_stage` (`not_recovering`) whatever the digest.
- A `recovering` installation without a restore point (after an abandon, not reachable from the console or the seed) reads `mode: recovery_resume` with the blocker `no_restore_point`.

