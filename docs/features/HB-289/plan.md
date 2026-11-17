---
type: ImplementationPlan
title: Live P04 retention screen in the console service mode, bound to the HB-288 retention API
description: Phase 5h of the pre-launch plan (HB-257 scope). In service mode, P04 shows retention policies with their pending proposals and impact, usage and expiry per data class over the registered sample items, holds with what they cannot preserve and shared references, deletion runs with per-store counts, archive restore as not connected, and the in-memory deletion records with the decision watermark. It offers the six HB-288 commands (approve, withdraw, propose a policy, place a hold, propose a hold release, stop a run) only where the read allows them. Nothing is deleted, archived, restored or exported from the screen, and no decision is shown as durably confirmed; the screen says so.
status: draft
tags: [HB-289, pre-launch, retention, console, P04]
sources:
  - resource: ../HB-288/plan.md
  - resource: ../HB-288/verification.md
  - resource: ../HB-287/plan.md
  - resource: ../HB-287/verification.md
  - resource: ../HB-285/plan.md
  - resource: ../HB-285/verification.md
---

# Scope and ownership

[HB-289](https://linear.app/hybrid-billing/issue/HB-289) binds P04 (data retention and deletion) to the [HB-288](../HB-288/plan.md) API ([verification](../HB-288/verification.md)). Base: main `d1ac7d8`. It follows the merged [HB-287 plan](../HB-287/plan.md) and [verification record](../HB-287/verification.md) (P03, which passed design review, render comparison and change review) and the code that plan produced (`apps/console/src/modules/administration/pages/recovery/`, `apps/console/src/service/recovery.ts`, the P03 branch in `apps/console/src/app/screen-groups/operations.tsx:55-59`), and the [HB-285](../HB-285/plan.md) P02 precedent. Every rule HB-285 and HB-287 ended with is part of this design from the start (see "Rules carried"). The P04 design is `docs/ui/screens/P04/spec.md`; it is in the authorized local workspace and not in this repository tree, so its line citations refer to that copy.

**Ownership**
- Root (Claude Code) owns the plan, integration, browser QA and verification. Implementation may be delegated to one writer in this worktree.
- The `ui-designer` role (AI, read-only) writes the visual specification, confirms or replaces the external references, and compares the final renders.
- The tech-writer role (AI, read-only) reviews the EN/KO/AR terminology in context (`.agents/rules/product-terminology.md`).
- An independent read-only reviewer reviews the design and the change. Focus:
  - nothing implies a deletion, an archive, a restore, an export, a running worker or a durably confirmed decision;
  - a hold is never shown as preserving data that is already deleted, being deleted or partly deleted (P04 `spec.md:85`);
  - single-administrator effects (place hold, stop run) and two-person effects (policy, release) are shown truthfully, including the case where a hold can never be released;
  - the policy-shortening impact sent on approval is exactly the impact the confirm showed;
  - unknown, not connected, other-scope and zero are never confused;
  - commands are offered only where the read allows them, and parsing fails closed.
- Machine: mac_mini.

**Allowed files**
- `apps/console/src/service/retention.ts` (types and hardened parsers) and `retention.test.ts` (including the command envelopes and resend identity).
  - It may import `count` and `hint` from `./integrations` read-only (`integrations.ts:173-193`).
- `apps/console/src/service/admin.ts`: `retention()` and the six command IDs (`retention.approve`, `retention.withdraw`, `retention.propose_policy`, `retention.place_hold`, `retention.propose_release`, `retention.stop_run`) in `CommandId` (`admin.ts:245-302`). The existing `access()` read (`admin.ts:410-420`) is used unchanged for the hold-owner list.
- `apps/console/src/modules/administration/pages/retention/`:
  - new `Live.tsx` (host), `LivePolicies.tsx`, `LiveExpiry.tsx`, `LiveHolds.tsx`, `LiveDeletion.tsx`, `LiveRestore.tsx`, `LiveDecisions.tsx`, `LiveProposals.tsx` (the proposal list shared by three tabs), `LiveCommand.tsx` (the shared command form and row note) and `LiveHoldForm.tsx` (the place-hold fields and owner list);
  - `live-model.ts`, `live-model.test.ts`, `live-copy.ts`, `live-codes.ts` (vocabularies and blocker sentences), `live-format.tsx` and `live.css`.
  - Any file nearing 1,500 lines is split by responsibility (`.agents/rules/code-structure.md`).
  - The existing preview `Screen.tsx` stays for the unconnected mode.
- `apps/console/src/app/screen-groups/operations.tsx`: the live branch for P04.
- `apps/console/package.json`: the test list (`src/service/retention.test.ts`, `src/modules/administration/pages/retention/live-model.test.ts`).
- `docs/features/HB-289/*`.
- No API, module, shell, P01, P02, P03, M03 or M04 file changes.
  - P04 imports only these read-only helpers: `count`, `hint` (`service/integrations.ts:173-193`), `useWide` (`identity/pages/users/Live.tsx:21`), `fsi`, `Num`, `ListOr` (`integrations/live-format.tsx:14`, `29`, `55`), and `identity/pages/users/live.css`.
  - P04's own `live-format.tsx` re-creates, with `p04-` classes (visual specification §1.2–1.3):
    - `Free` in place of P02's `Prose`: free text in `<span dir="auto" class="p04-free">` with `unicode-bidi: isolate`, never `<bdi>` (P02's `Prose` uses `<bdi>`, `integrations/live-format.tsx:38-40`);
    - `Who` with the `p04-login` class (P02's emits `p02-login`, `integrations/live-format.tsx:33-36`);
    - `href`, which builds `#/retention?…` (P02's builds `#/integrations?`, `integrations/live-format.tsx:61-64`; P04's path is `/retention`, `screens.json` P04 `path`);
    - `Stamp`, `Rich`, `ltr`, `Known` (P03's emit `p03-` classes, `recovery/live-format.tsx:25-47`).
  - A model test asserts that every href the screen builds starts with `#/retention?`.
  - P04 keeps its own attempt store and command form, so P01, P02, P03 and P04 attempts never mix.

**Non-goals**
- No deletion plan proposal or approval, no worker step, retry, backup-expiry record, archive, tiering or archive restore (all `not_connected` in the read; HB-288 `plan.md:277-286`).
- No impact preview before proposing a policy: the API computes the impact only for a pending shortening (`retention.go:784-792`).
- No filters, server search, cursors, drawers or before/after comparison from the P04 spec (`spec.md:69-75`, `89-93`, `121-125`); see Residuals.
- No change to the shell tab bar, the shared `bdi` style or the review control.

# What the API offers (merged HB-288 code)

## Read: `GET /api/v1/retention`

Route: `boundary.go:39`. The read is admin only, through `admitted` (`retention.go:892`), under the domain read lock (`retention.go:890-891`).

**Whole-page states**
- **Another workspace than the boot workspace:** `{revision, now, state: "not_configured"}` and nothing else (`retention.go:897-899`).
- **Any module or roster error:** `dependency_unavailable` for the whole read (`retention.go:901-904`, `857-864`). The console maps it to `busy` (`admin.ts:320`).
- **Otherwise:** `{revision, now, state: "configured", policies, expiry, holds, deletion, restore, decisions, proposals}` (`retention.go:305-316`, `867`). With `--seed none` the boot workspace still gets the full shape with seven policy rows without a policy and empty lists (`retention_api_test.go:514-557`).

**General rules (HB-288 `plan.md:156-164`)**
- Counts are exact for the population, and every enumerated key is present (`zeroCounts`, `retention.go:415-421`), so a present `0` is a real zero.
- `population` is `registered_sample_artifacts` on every artifact count block (`retention.go:22`, `460`, `567`): the counts cover the registered items visible to the reader, not all data of the installation.
- A source the API does not have is the string `not_connected`, never `0` or an empty list.
- Every capped list carries `total` and `more`; the cap is 50 (`retention.go:17`).
- People are `{id, login}` with `login: null` when the directory does not know the ID, or `null` for no person (`retention.go:46-49`, `328-337`).
- Times are RFC 3339 UTC seconds (`stamp`); impact `as_of` keeps fractional seconds (`retention.go:734`, `790`).

**`policies`**: seven rows in `retention.Classes` order `claim, conversation, final, identity, projection, source, temporary` (`modules/retention/types.go:91`, `retention.go:819-841`). Each row (`retention.go:65-75`):
- `class`, `target` (`retention-class:<class>`);
- `policy`: `{revision, period_days, grace_days, proposer, approver, approved_at}` or `null` (no policy: never due);
- `origin: "artifact_anchor"`, `scope: "not_connected"`, `archive_rule: "not_connected"` (`retention.go:823`);
- `pending`: the pending policy proposal number for the class, or `null`;
- `can_propose` / `propose_blocked`, whose only blocker is `already_proposed` (`retention.go:833-837`).

**`expiry`**: seven class groups (`retention.go:95-105`, `457-480`):
- `class`, `population`, `counts` with the seven keys `eligible, held, referenced, not_due, no_policy, in_run, deleted` (`retention.go:40`), `total`, `more`;
- `artifacts`, at most 50 by ID; `usage`, `growth`, `bytes`: `not_connected`.
- Each artifact (`retention.go:77-93`, `434-455`): `id`, `anchor`, `due_at` (`null` without a policy), `stores`, `access` (`ok`, `blocked`, `deleted`; `modules/retention/types.go:61-63`), `standing` (one count key, decided in the order deleted, in run, no policy, not due, held, referenced, eligible; `retention.go:390-406`), `held` (the hold flag on its own), `damaged`, `run` (`{target: "retention-run-<n>", label: "run/<n>"}` or `null`) with `other_scope` (true when the owning run is not the reader's; `retention.go:364-372`), `shared` (another tenant is listed; no ID or count), `decision: "not_connected"` only when `access` is not `ok` (omitted otherwise, `retention.go:89`, `408-413`), `holdable` / `hold_blocked` (`target_too_long` when `retention-artifact:<id>` exceeds 128 bytes) and `hold_target` (empty when not holdable).

**`holds`** (`retention.go:150-156`, `565-596`): `total`, `more`, `rows` (the reader's holds, newest first, at most 50), `tenant_target` (`retention-tenant:<workspace>`) and `references`.
- Each hold (`retention.go:119-136`, `521-563`): `id`, `target` (`retention-hold:<id>`), `owner`, `reason` (the hold's own reason), `placer`, `placed_at`, `rationale` (the command reason), `released`, `scope: {artifacts, workspace, classes}` (visible artifact IDs only; `workspace` true when the hold covers the reader's workspace as a tenant), `counts` with `not_preserved, outcome_pending, partially_deleted`, `counts_scope`, `outcomes` (at most 50) and `outcomes_more`, `release` (pending release number or `null`), `can_propose_release` / `release_blocked`.
- **`counts_scope`** is `workspace`, or `other_scope` with `counts: null` when the hold reaches artifacts or tenants the reader cannot see (`retention.go:539-547`; HB-288 `verification.md:33`).
- **`outcomes`** list only covered items the hold cannot preserve or whose outcome is pending: deleted (`not_preserved`), damaged (`partially_deleted`) or being deleted (`outcome_pending`) (`modules/retention/records.go:164-175`, `177-185`). Preserved items are never listed. Each is `{artifact, outcome, stores}` with store results `deleted` or `failed`.
- **Release blockers, in order** (`retention.go:482-499`): `released`; `already_proposed` (a pending release from anyone); `no_eligible_approver` (fewer than two administrators other than the placer, counted from the roster with the installation-administration grant, `retention.go:843-855`); then the module's dry run with the reader as proposer, through `retentionReason` (`retention.go:339-354`): `placer`, `not_found`, `released`, `limit`, `conflict`, `invalid`.
- **`references`**: `{population, total, more, rows: [{id, class}]}`, the visible items another item requires; the requirer is never named (`retention.go:584-594`).

**`deletion`** (`retention.go:200-208`, `677-696`): `total`, `more`, `rows` (the reader's runs, newest first, at most 50), and `worker`, `retry`, `backup_expiry`, `propose_plan`: `not_connected`.
- Each run (`retention.go:178-198`, `645-675`): `run` (reference), `class`, `revision` (the policy revision used), `state` (`running`, `stopped`, `done`, `complete`; `types.go:56-59`), `decision: "not_connected"`, `next_chunk`, `chunks`, `chunk_size`, `lease_seconds`, `reason`, `damaged` (count), `proposer`, `approver`, `stopper` (or `null`), `counts` with the eight outcomes `pending, deleting, deleted, failed, held, referenced, not_due, stopped` (`retention.go:41-42`), `stores`, `artifacts` (`{total, more, rows: [{id, outcome}]}`), `can_stop` / `stop_blocked`.
- `stores`: `{store, target, deleted, failed, excluded, pending}`, sorted by store. `excluded` counts manifest items whose outcome is held, referenced, not due or stopped and that have no result for the store; `pending` is the module's pending minus `excluded`, so target = deleted + failed + excluded + pending (`retention.go:598-633`; P04 `spec.md:61`).
- `stop_blocked` is `""` for `running`, `stopped` for a stopped run, otherwise `finished` (`retention.go:635-643`).

**`restore`**: `{state: "not_connected"}` (`retention.go:210-212`, `867`).

**`decisions`** (`retention.go:231-238`, `698-728`): `watermark` (installation-wide), `tombstones: {window: 1000, total, more, rows}` (only the last 1,000 sequences are scanned; of those, visible ones newest first, at most 50), `ledger: "in_memory"`, `epoch`, `durable_commit`, `restore_check`: `not_connected`. Each tombstone: `sequence`, `artifact`, `class`, `run` (reference or `null` with `other_scope`), `at`, `decision: "not_connected"`.

**`proposals`** (`retention.go:279-303`, `754-817`): visible entries, newest first, at most 50, with `total` and `more`.
- Row: `number`, `target` (`retention-proposal-<n>`), `kind` (`policy`, `plan`, `release`), `state` (`pending`, `applied`, `withdrawn`, or the server's `retired`, `retentionseed.go:20`), `created_at`, `proposer`, `approver`, `rationale`, `closed_by`, `closing_rationale`, `summary`, `impact_required`, `impact_now`, `can_approve` / `approve_blocked`, `can_withdraw` / `withdraw_blocked`.
- `summary` has exactly one non-null member by kind (`retention.go:253-277`, `737-747`): `policy: {class, period_days, grace_days, impact_at_proposal: {as_of, count, digest} | null}`, `plan: {class, artifact_count, chunk_size, lease_seconds, reason}`, `release: {hold, reason}`.
- **Impact.** For a pending policy proposal that is a shortening now (period + grace below the current revision's, `retention.go:749-752`), `impact_required: true` and `impact_now: {as_of, count, digest, workspace_count}` from the module's `Impact` at the read time (`retention.go:784-792`). `count` and `digest` are installation-wide; `workspace_count` is the reader's workspace only. The count is the not-yet-deleted items of the class that are due under the proposed total but not under the current one, held and required items included (`modules/retention/changes.go:66-88`). Otherwise `impact_required: false` and `impact_now: null`.
- **Approve blockers** (`retention.go:754-777`): non-pending → `not_pending` (or `retired`) for both actions; `not_console_kind` for plans; `own_proposal`; then the module's dry run with an empty impact, where `impact_required` is treated as no blocker: `placer`, `released`, `not_found`, `limit`, `conflict`, `invalid` (and, unreachably from the read, `impact_changed`).
- **Withdraw blockers:** `not_pending`, `retired`, `not_proposer`.

**Never exposed:** claim tokens, lease deadlines, another tenant's identifiers or counts, any content (HB-288 `plan.md:208`).

## Commands

All six are `retention.<kind>` (`retentioncmds.go:14-27`, `382-401`):
- Each takes a reason, and `expected_revision` is the read's `revision`.
- Each needs `AdministerInstallation` on the installation kind (`retentioncmds.go:386-388`), a recorded deviation from P04's separate permissions (HB-288 `plan.md:221`).
- The handler refuses a reason over 256 bytes with a fieldless `invalid_request` (`retentioncmds.go:356-357`, `directory.go:13`); the command layer allows 2,000 bytes (`server.go:120`), so the screen enforces 256 itself.
- Every module actor is the reader as a human (`retentioncmds.go:359`). Each command makes exactly one mutating module call, last (HB-288 `plan.md:233`).
- A success moves the revision (`retentioncmds.go:379`).

| Command | Target | Fields | Result `business_state` / `fields` | Server prechecks and refusals |
| --- | --- | --- | --- | --- |
| `retention.approve` | `retention-proposal-<n>` | `impact_as_of` (timestamp), `impact_count` (integer), `impact_digest` (text), all optional (`:390-391`) | `applied`; `proposal`, `target` (`:152-155`) | not pending or a plan → `invalid_stage` without a module call (`:186-188`); impact fields on a release → `invalid_request` on the first given field; a missing or malformed one on a policy → `invalid_request` on that field (`impact_digest` 64 lowercase hex, `impact_count` ≥ 0) (`:157-182`); self-approval → `prerequisite_failed`; `impact_required`, `impact_changed`, `placer`, `released` → `invalid_stage` (`:37-56`) |
| `retention.withdraw` | `retention-proposal-<n>` | — | `withdrawn`; `proposal`, `target` | not pending → `invalid_stage` (`:199-203`); not the proposer → `prerequisite_failed` |
| `retention.propose_policy` | `retention-class:<class>` | `period` (integer, required), `grace` (integer, required) (`:393-394`) | `pending`; `proposal`, `target` | `period` outside 1–36,500 or `grace` outside 0–365 → `invalid_request` on the field (`:210-227`); a pending policy proposal for the class from anyone (`already_proposed`) → `invalid_stage` (`:228-230`); values equal to the current revision (`unchanged`) → `invalid_stage` (`:231-234`); limit → `prerequisite_failed` |
| `retention.place_hold` | `retention-artifact:<id>` or `retention-tenant:<workspace>` | `owner` (ID, required), `reason` (text, required), `class` (enum of the seven classes, optional) (`:395-397`) | `held` when all three counts are zero, else `partially_preserved`; `hold`, `target`, `not_preserved`, `outcome_pending`, `partially_deleted` (`:307-315`) | `owner` not an active member with an active user (`grantees`, `access.go:129-143`) → `invalid_request` [`owner`]; `reason` failing the module's text rule → [`reason`]; `class` with an artifact target → [`class`] (`:263-279`); 100 or more live holds in the workspace → `prerequisite_failed` (`:280-289`, `:32`); a roster failure → `dependency_unavailable` |
| `retention.propose_release` | `retention-hold:<id>` | `reason` (text, required) (`:398`) | `pending`; `proposal`, `target` | `reason` invalid → `invalid_request` [`reason`]; released or a pending release from anyone → `invalid_stage` (`:318-330`); the placer proposing → `invalid_stage` (`placer`) |
| `retention.stop_run` | `retention-run-<n>` | — | the run state after the stop (`done` for the sample); `run` (`run/<n>`), `target`, `stopped` (count) (`:338-352`) | state not `running` → `invalid_stage` (`:340-346`) |

Line references in the table are to `retentioncmds.go` unless named.

**Hold and release text rule.** `retention.ValidText`: 1 to 200 bytes of valid UTF-8 without C0 controls, DEL or C1 controls (`modules/retention/common.go:12-13`, `29-38`). A line break is a control, so these are single-line texts.

**Hold IDs.** `place_hold` increments the counter before any check, then uses `retention-hold-<n>`, skipping IDs the module already knows (`retentioncmds.go:246-261`). On a fresh sample API the first console hold is `retention-hold-1`.

**Stop effect.** `Stop` marks the run stopped, and every still-pending item in a chunk no worker has claimed gets the outcome `stopped` and leaves the run, so the run no longer blocks its access (`modules/retention/runs.go:250-275`, `36-39`, `reads.go:3-15`). Items already deleted stay deleted; claimed chunks are not recalled.

**Hold coverage.** An artifact hold covers its item; a workspace hold covers every registered item of that tenant, of the listed classes or all classes, including items registered later (`modules/retention/changes.go:99-109`).

**Error mapping and fields**
- **Mapping** (`retentioncmds.go:37-56`): `self_approval`, `not_proposer` and `ErrLimit` → `prerequisite_failed`; actor-kind refusals and `ErrInvalid` → `internal_error` (unreachable); every other refusal, `ErrNotFound` and `ErrConflict` → `invalid_stage`.
- **Codes that carry fields.** Only `invalid_request` carries fields, and from a handler only declared names: `impact_as_of`, `impact_count`, `impact_digest`, `period`, `grace`, `owner`, `reason`, `class`. Shape validation may also name `rationale`, `target_id`, `expected_revision`, `idempotency_key`, `request_id`, `client`, an undeclared field name or the excess marker (`modules/commands/execute.go:53-107`). Every other code is fieldless, so the read's blockers explain refusals.
- **Revision.** An outdated `expected_revision` is `revision_conflict` before `Apply` (`execute.go:356-358`).
- **`internal_error` is journaled.** The first moves the revision; a resend with the same key replays it (HB-288 `plan.md:270`).
- **Console failure classes** (`admin.ts:310-335`): `invalid`, `refused`, `stage`, `denied`, `conflict`, `busy`, `limited`, `stale`, `unauthenticated`; `internal_error` becomes `partial` only with an operation ID, otherwise `unavailable`.
- **Resolve** reads only the server store (`retentioncmds.go:63-145`): a canonical proposal number whose entry is visible; a class in the list; a visible stored artifact or the reader's own tenant; a hold of the workspace; a canonical run number whose run is of the workspace; and only in the boot workspace. Anything else is `permission_denied`.

**Replay: every target stays resolvable.** Artifacts, holds and runs are never removed from the store, and closed proposal entries leave only beyond 200 kept entries (`retentionseed.go:97-108`, `installseed.go:18`). So a resend with the same key and body after a lost response returns the stored result, even after the revision moved. A `place_hold` resend with a **new** key places a second hold (HB-288 `plan.md:148`); the screen reuses the key for identical values, so only an edited resend can do that, and the form says so.

**Repeats are refused, not replayed.** A fresh approval of a non-pending proposal, a second policy or release proposal for the same class or hold, and a second stop are `invalid_stage` without moving the revision.

## Sample seed (who did what)

`--seed sample`, one counted step `retention` (`retentionseed.go:319-429`). First administrator A1 `sample-admin` = `admin@example.invalid`; second A2 `sample-admin-2` = `admin2@example.invalid`; member `sample-member` = `member@example.invalid` (`seed.go:22-26`). Command reasons read "Sample data" (`installseed.go:20`). The expected read is asserted in `retention_api_test.go:370-510`.

- **Policies** (#1–#3, proposed by A1, approved by A2): `source` 365 + 30, `final` 3650 + 0, `temporary` 7 + 0. `claim`, `conversation`, `identity`, `projection` have no policy (`retention_api_test.go:385-392`).
- **Expiry counts** (`retention_api_test.go:398-405`): `source` 7 items (deleted 1, in run 1, held 1, referenced 1, not due 3); `temporary` 3 (deleted 1, in run 2); `final` 1 not due; `claim`, `conversation`, `projection` 1 each without a policy; `identity` 0.
  - `sample-source-1` deleted (decision not connected); `sample-source-2` blocked, in `run/1`, damaged; `sample-source-3` required by `sample-final-1`; `sample-source-4` held; `sample-temporary-1` deleted and held; `-2`, `-3` blocked in `run/2` and held (`retention_api_test.go:414-421`).
- **Holds** (`retention_api_test.go:433-450`):
  - `sample-hold-temporary`, placed by the member, owner the member, reason "Sample export review hold", scope this workspace with class `temporary`; counts not preserved 1; outcome `sample-temporary-1` not preserved; release #6 pending; blocker `already_proposed`.
  - `sample-hold-dispute`, placed by A1, owner A2, reason "Sample billing dispute hold", item `sample-source-4`; all counts 0; blocker `no_eligible_approver` for both administrators (only A2 remains besides the placer A1).
  - References: `sample-source-3` (source).
- **Runs** (`retention_api_test.go:452-466`): `run/1` done (class source, revision 1, chunk size 2, lease 3600 s, "Sample expiry deletion"; database 4 = 2 deleted + 2 excluded; object store 2 = 1 deleted + 1 failed; `finished`). `run/2` running (temporary, object store 3 = 1 deleted + 2 pending; Stop offered).
- **Decisions** (`retention_api_test.go:472-476`): watermark 2; tombstones 2 (`sample-temporary-1` in `run/2`, `sample-source-1` in `run/1`).
- **Proposals** (`retention_api_test.go:482-485`, `509`):
  - #1–#3 applied policies; #4, #5 applied plans (no action for anyone);
  - **#6, pending release** of `sample-hold-temporary`, **proposed by A1** (`retentionseed.go:417-421`): A1 sees `own_proposal` and Withdraw; A2 sees Approve and `not_proposer`;
  - **#7, pending policy** `source` 180 + 30, a shortening, **proposed by A2** (`retentionseed.go:422-426`): A1 sees Approve with `impact_required` and `impact_now` count 2, workspace count 2; A2 sees `own_proposal` and Withdraw.
- **After A1 approves #7:** `source` revision 2 is 180 + 30; `sample-source-5` (300 days) and `-6` (250 days) become `eligible`, so `source` reads eligible 2, not due 1. Nothing is deleted.
- **After A1 stops `run/2`:** `done` with 2 stopped; object store 3 = 1 deleted + 2 excluded (HB-288 `verification.md:21`); `sample-temporary-2`, `-3` lose the run, so their access is `ok` and their standing `held`.

# Design

The design follows the P03, P02, P01, O01–O03 and M01 live patterns:
- full viewport width; neutral rules and tags, no coloured edge strips (`.agents/rules/service-screen-style.md`);
- EN, KO and AR copy;
- the shared failure box and the reason field (UTF-8 bytes, 256), with an error summary;
- layout switches by container size on `.p04-live` (see Layout); `useWide` (1280px viewport) is imported only where a component needs a JavaScript width decision, as in P03;
- dates through `Intl` in UTC, kept whole with date and time parts (`recovery/live-format.tsx:15-29`);
- IDs, digests and targets in `<bdi dir=ltr>` monospace;
- free text (hold reasons, release reasons, rationales) in `<span dir="auto">` with `unicode-bidi: isolate`, not `<bdi>`, because the global `bdi { direction: ltr }` rule (`shared/styles.css:187`) forces Arabic reasons to left-to-right (an HB-287 residual). This is a P04-scoped choice; the shared rule is not changed.

Classes use the `p04-` prefix.

## Design decisions and alternatives

1. **Hold-owner list from `GET /api/v1/identity/access` `subjects`.** That list is built by the same `grantees` function the server checks the owner against (`access.go:129-143`, `153-159`), so a listed owner is acceptable by construction. Rejected: a free-text owner ID (typing errors, no login shown); the reader only (cannot name the accountable owner P04 asks for, `spec.md:58`, `85`); the P01 users list (a different rule: it includes suspended users and memberships). The list is read when the place-hold form opens. A failed read shows "The list of possible owners could not be read." with Try again, and the form cannot be sent. An empty list reads "No active member can own a hold." and cannot be sent. `admin.ts` maps a missing `subjects` to `[]` (`admin.ts:418`), so an absent list fails closed as "no owner".
2. **Proposals are shown on the tab of their kind:** policy proposals under Policies, release proposals under Holds, plan proposals (read-only) under Deletion. Rejected: a seventh list outside the registered tabs. Each tab says, when `proposals.more` is true, "Only the newest 50 proposals are read; older ones are not shown on any tab."
3. **The impact sent is the read's `impact_now`, verbatim**, frozen with the confirm. The screen never recomputes, edits or rounds it. Rejected: sending no impact and relying on a refusal (the module requires a reviewed impact for a shortening, `modules/retention/changes.go:123-165`, HB-288 `plan.md:54`).
4. **No plan proposal from the console, and plan proposals are fully read-only.** The API refuses plan approval and offers no plan command (`retentioncmds.go:186`). Withdraw of a pending plan is allowed by the API to its proposer, but the screen does not offer it (design review L5): plans are not connected, so their whole lifecycle stays off this screen.
5. **Root decisions on the open questions (folded in).** The owner list comes from `/identity/access` subjects (decision 1); the decisions tab keeps the neutral sentence that sequence numbers can be skipped, which prevents gaps from reading as lost records and adds nothing beyond the numbers themselves (HB-288 residual); proposals stay on their own tabs (decision 2); KO and AR reuse P03's hold terms 보존 보류 / تعليق الحذف (see Terminology citations); the shell-scoped `.planning-tabs` override at 390 is accepted as a residual, as in P03.

## Host, remount and URL state

- **Live branch.** `operations.tsx` sets `live = !!service && (p.screenId === "P01" || p.screenId === "P02" || p.screenId === "P03" || p.screenId === "P04")` (`operations.tsx:19`). The P04 branch returns before the P01 fallback (`operations.tsx:61`, `if (live && service)`, which would otherwise render P01 for P04; visual specification §1.1), and that fallback is changed to check `p.screenId === "P01"` explicitly. The render pass asserts `data-screen-id="P04"` on every P04 render. For P04 the branch returns:
  - `<div className="work-domain remaining-security" data-screen-id="P04" data-source-state="live" dir lang key={`P04:${locale}`}>` around `LiveRetention`;
  - with props `locale`, `view`, `params`, `session` and `onSignOut`, as for P03 (`operations.tsx:55-59`).
  - The key is locale only. Hooks stay unconditional, and the live branch never reports `source: unconnected` (`operations.tsx:33`).
- **Remount per view (rule).** `RemainingScreens` keys `RevisionScope` by screen, locale, view and scope parameters (`RemainingScreens.tsx:92`), so every tab change remounts `LiveRetention` and reloads the read. Only the module-level store (attempts, drafts, kept refusals) survives.
- **URL state:**
  - `view=` is one of `policies`, `expiry`, `holds`, `deletion`, `restore`, `decisions` (`screens.json` P04 tabs); anything else falls back to `policies`.
  - Optional selections, each validated against the read and otherwise ignored:
    - `class=<class>` (policies, expiry): one of the seven classes and present in that tab's rows;
    - `artifact=<id>` (expiry): `[A-Za-z0-9._:-]{1,128}` and present in a group's shown artifacts;
    - `hold=<id>` (holds): same pattern, present in the shown holds;
    - `run=<n>` (deletion): canonical decimal, a shown run labelled `run/<n>`;
    - `proposal=<n>` (policies, holds, deletion): canonical decimal, a shown proposal of that tab's kind.
  - The selection is derived from the URL on every render, never stored in state; scroll-into-view happens once per distinct value; the selected row is marked `aria-current` (`recovery/Live.tsx:52-53`, `79-83`).
  - Cross links, canonical IDs only:
    - a policy row's pending number → `view=policies&proposal=<n>`;
    - an artifact's run → `view=deletion&run=<n>` (only for a reference, never for `other_scope`);
    - a run artifact → `view=expiry&class=<class>&artifact=<id>`;
    - a hold's pending release → `view=holds&proposal=<n>`; a release proposal → `view=holds&hold=<id>`;
    - a tombstone's run → `view=deletion&run=<n>`; its item → expiry, when the item is in the read.
  - Keeping the selection across shell tab changes and the unsaved-draft warning on a tab switch are residuals, as in P03 (HB-287 `verification.md:49`).
- **Focus and live region:** after a success, the status message (`#p04-notice`, `role=status`) is focused; a refusal focuses the row note; Cancel returns focus to the opener; error-summary links move focus to the field; the reload sets `aria-busy` on the screen.
- **Stale responses.** A request sequence drops stale read responses (`recovery/Live.tsx:55-76`). A command form ignores a second send while busy. The owner-list read has its own sequence.
- **Store.** The attempt and draft store (module level, keyed by command and target) is scoped by workspace, user and session (`setScope`, as `recovery/live-model.ts:426-452`) and cleared by `forget()` on sign-out and on a scope change.

## Shared frame (every tab)

- **Scope bar:** workspace, data revision, now (UTC), Reload.
- **Statement:**
  - "**Records decisions only.** Nothing is deleted, archived, restored or exported from this screen in this build, and no deletion worker runs."
  - KO: "**결정만 기록합니다.** 이 빌드에서는 이 화면에서 아무것도 삭제·아카이브·복원·반출하지 않으며 삭제 작업자도 실행되지 않습니다."
  - AR: "**تسجيل القرارات فقط.** لا يُحذف أي شيء ولا يُؤرشف ولا يُستعاد ولا يُصدَّر من هذه الشاشة في هذا الإصدار، ولا تعمل أي جهة حذف."
- **Values note** (small):
  - "No legal or contractual retention period has been decided. Periods here are values entered by administrators; in the sample data they are synthetic."
  - KO: "법적·계약상 보관 기간은 아직 정해지지 않았습니다. 여기의 기간은 관리자가 입력한 값이며, 샘플 데이터의 값은 가상 값입니다."
  - AR: "لم تُحدَّد بعد أي مدة احتفاظ قانونية أو تعاقدية. المدد هنا قيم يُدخلها المسؤولون، وهي في بيانات العينة قيم افتراضية."
- **Permission note** (small):
  - "Every command here currently needs installation administration. Separate policy, hold, deletion and evidence-access permissions are not available yet, and auditors cannot open this screen yet."
  - KO: "현재 이 화면의 모든 명령에는 설치 관리 권한이 필요합니다. 정책·보존 보류·삭제·증빙 접근 권한은 아직 별도로 제공되지 않으며, 감사 조회자는 아직 이 화면을 열 수 없습니다."
  - AR: "تتطلب كل الأوامر هنا حاليًا صلاحية إدارة التثبيت. صلاحيات السياسات وتعليق الحذف والحذف والوصول إلى الأدلة غير متاحة بشكل منفصل بعد، ولا يستطيع المدققون فتح هذه الشاشة بعد."
- **Whole-page states:**
  - first load: "Loading retention data… Uncounted values are never shown as 0." (P04 `spec.md:105`) with no numbers;
  - reload: `aria-busy`;
  - `not_configured`: "Retention is not configured in this workspace";
  - an unknown `state`: "The retention read returned an unknown state (code)";
  - denied: "Retention is not available to you" / "You do not have permission to manage retention in this installation.";
  - unauthenticated: "Your session has ended." with "Sign in again" (calls `forget()`);
  - failure (`busy`, `unavailable`, `limited` with the wait) with Try again.
- **Read-only.** When `revision` is empty, commands are disabled and the tab reads "Commands are unavailable because the data revision is unknown. Reload."

## Tabs (the registered P04 keys)

### 1. Retention policies (`policies`)

- **Lead:**
  - "A policy sets how long each data class is kept, counted from each item's start time. Approving a policy adds a revision; it deletes nothing. Policy scope by jurisdiction or contract and archive rules are not connected in this build."
  - KO: "정책은 각 자료 등급을 항목별 기산 시각부터 얼마나 보관할지 정합니다. 정책을 승인하면 새 revision이 추가될 뿐 아무것도 삭제하지 않습니다. 이 빌드에서는 관할·계약별 정책 범위와 아카이브 기준이 연결되어 있지 않습니다."
  - AR: "تحدد السياسة مدة الاحتفاظ بكل فئة بيانات بدءًا من وقت بدء كل عنصر. الموافقة على سياسة تضيف مراجعة جديدة ولا تحذف شيئًا. نطاق السياسة حسب الاختصاص أو العقد وقواعد الأرشفة غير متصلة في هذا الإصدار."
- **Policies table**, one row per class, in the read order:
  - **Data class:** the class in words.
  - **Revision:** the number; "No policy" when `policy` is `null`.
  - **Period / Grace / Total:** "{p} days", "{g} days", "{p+g} days"; without a policy: "No policy: items of this class never become due" across the three cells.
  - **Approved:** approver (login, falling back to ID) and time, with the proposer below.
  - **Starts from:** `artifact_anchor` → "Each item's recorded start time"; another value "Unknown (code)".
  - **Scope / Archive rule:** `not_connected` → "Not connected"; another value "Unknown (code)".
  - **Pending / Action:** the pending number as a link, or **Propose a change** when allowed (see Commands), or the blocker sentence.
- **Policy proposals** (kind `policy`) below the table (see Proposals).

### 2. Usage and expiry (`expiry`)

- **Lead:**
  - "Shows which registered items are due for deletion and why the others are not. Usage, growth and storage size are not connected, and nothing is deleted from here."
  - KO: "등록된 항목 중 삭제 대상과 그렇지 않은 이유를 보여줍니다. 사용량·증가량·저장 크기는 연결되어 있지 않으며 여기에서는 아무것도 삭제하지 않습니다."
  - AR: "يعرض العناصر المسجّلة المستحقة للحذف وسبب عدم استحقاق غيرها. الاستخدام والنمو وحجم التخزين غير متصلة، ولا يُحذف أي شيء من هنا."
- **Population note:** `registered_sample_artifacts` → "Counts cover only the sample items registered in the retention register for this workspace, not all data in the installation." (KO "건수는 이 업무 공간에 대해 보관 레지스트리에 등록된 샘플 항목만 포함하며 설치의 전체 데이터가 아닙니다." AR "تشمل الأعداد فقط عناصر العينة المسجّلة في سجل الاحتفاظ لمساحة العمل هذه، وليس كل بيانات التثبيت."). Another value reads "Counted population: Unknown (code)".
- **Decision note:** "No durable decision record is connected. A deletion or access block shown here is recorded only in the in-memory register and is not confirmed." (KO "내구성 있는 결정 기록이 연결되어 있지 않습니다. 여기에 표시된 삭제나 접근 차단은 메모리 내 레지스트리에만 기록되었으며 확정되지 않았습니다." AR "لا يوجد سجل قرارات دائم متصل. أي حذف أو منع وصول معروض هنا مسجّل فقط في السجل داخل الذاكرة وغير مؤكد."). P04 `spec.md:45`, `116`.
- **One group per class:**
  - heading: the class in words and "{total} items";
  - a count strip with the seven standings, each labelled in words, in the order Due for deletion, On hold, Required by another item, Not due yet, No policy, In a deletion run, Recorded as deleted. A missing key reads "Unknown"; a present 0 reads "0";
  - Usage, Growth, Storage size: "Not connected";
  - an empty group with a real total 0: "No registered items in this class.";
  - "Showing 50 of {total}." when `more`.
- **Items table** (per group), columns:
  - **Item:** ID (ltr); a neutral tag "Shared with another workspace" when `shared` (no other workspace is named);
  - **Start · Due:** start time; due time, or "Never (no policy)" when `due_at` is `null`;
  - **Stores:** store IDs, or "None";
  - **Access:** `ok` "Accessible", `blocked` "Blocked", `deleted` "Recorded as deleted". The decision line below is rendered from the `decision` field value, not from the access value: `not_connected` → "Decision record: not connected (not confirmed)"; any other non-empty value → "Decision record: Unknown (code)"; field missing while `access` is not `ok` → "Decision record: Unknown"; missing while `ok` → no line. The same rule applies to the `decision` field of runs and deletion records;
  - **Standing:** in words; neutral tags "On hold" when `held` and the standing is not `held`, and "Partly deleted (damaged)" when `damaged`;
  - **Run:** the run label as a link, "Owned by a run that is not shown in this workspace" when `other_scope` (the run may belong to another workspace; no ID is shown), "None" otherwise;
  - **Action:** **Place hold** when allowed; `target_too_long` reads its sentence.

### 3. Holds and references (`holds`)

- **Lead:**
  - "A hold stops deletion of covered items that still exist. It cannot bring back items that are already deleted, being deleted or partly deleted. Releasing a hold needs two administrators who did not place it: one proposes, the other approves."
  - KO: "보존 보류는 대상 항목 중 아직 남아 있는 항목의 삭제를 막습니다. 이미 삭제되었거나 삭제 중이거나 일부만 삭제된 항목은 되살리지 못합니다. 보류를 해제하려면 보류를 등록하지 않은 관리자 두 명이 필요합니다. 한 명이 제안하고 다른 한 명이 승인합니다."
  - AR: "يمنع تعليق الحذف حذف العناصر المشمولة التي ما زالت موجودة. ولا يمكنه إعادة العناصر المحذوفة أو قيد الحذف أو المحذوفة جزئيًا. يتطلب رفع التعليق مسؤولَين لم يضعاه: أحدهما يقترح والآخر يوافق."
- **Place a workspace hold** (opener above the list) when `tenant_target` is well-formed (see Commands).
- **One block per hold**, newest first:
  - **Facts:** ID; state ("In force" / "Released"); owner; placed by and when; hold reason (free text); action reason (`rationale`, free text);
  - **Covers:** "This workspace" (with "all classes" or the class list in words) when `scope.workspace`; the item IDs when `scope.artifacts` is non-empty; both when both. An empty scope with `workspace` false reads "Nothing in this workspace".
  - **What this hold cannot preserve** (a released hold reads "While in force, this hold could not preserve…", because the module keeps counting covered items after release, `records.go:195-202`):
    - `counts_scope: workspace`: "{n} already deleted (not preserved)", "{n} with a deletion outcome pending", "{n} partly deleted", each with a real 0 shown as 0;
    - `counts_scope: other_scope` (counts `null`): "Not shown: this hold also covers data outside this workspace.";
    - an unknown scope or a missing count: "Unknown".
    - The outcomes table (item, outcome in words, store results: the store ID as the only ltr isolate, then "deleted" or "failed" in words) with "Only items this hold cannot preserve, or whose deletion outcome is pending, are listed. Items it does preserve are not listed." "More items are not shown." when `outcomes_more`. "None listed." when empty.
  - **Release:** the pending release number as a link; **Propose release** when allowed; otherwise the blocker sentence (hidden for `released`).
- "Showing the newest 50 of {total}." when `more`; "No holds in this workspace." when empty with a real 0.
- **Required items** (`references`): ID and class in words; "Another item requires this one, so it is not deleted while that item exists. The requiring item is not named." "None." when empty; "Showing 50 of {total}." when `more`. The population note as on expiry.
- **Release proposals** (kind `release`) below (see Proposals).

### 4. Deletion runs (`deletion`)

- **Lead:**
  - "Deletion runs were recorded by the sample data. Proposing or approving a deletion plan, the deletion worker, retries and backup expiry are not connected in this build, so a running run does not progress. Stopping a run releases the items no worker has claimed; it restores nothing."
  - KO: "삭제 작업은 샘플 데이터로 기록되었습니다. 이 빌드에서는 삭제 계획 제안·승인, 삭제 작업자, 재시도, 백업 만료가 연결되어 있지 않아 실행 중인 작업도 진행되지 않습니다. 작업을 중지하면 작업자가 아직 맡지 않은 항목이 작업에서 풀려날 뿐 아무것도 복원하지 않습니다."
  - AR: "سُجّلت عمليات الحذف من بيانات العينة. اقتراح خطة حذف والموافقة عليها وجهة الحذف وإعادة المحاولة وانتهاء النسخ الاحتياطية غير متصلة في هذا الإصدار، لذلك لا تتقدم العملية الجارية. إيقاف العملية يحرر العناصر التي لم تتسلمها أي جهة بعد، ولا يستعيد شيئًا."
- **Not connected list:** Deletion worker, Retry, Backup expiry, Deletion plan proposals: each `not_connected` → "Not connected"; another value "Unknown (code)".
- **One block per run**, newest first:
  - **Facts:** run label; class; policy revision; state in words; "Decision record: not connected (not confirmed)"; chunks "{next} of {chunks} chunks finished (a stop finishes unclaimed chunks)" (`next_chunk` counts finished chunks in order, `runs.go:337-339`); chunk size; lease "{n} s"; plan reason (free text); damaged count; proposed by, approved by, stopped by ("Not stopped" when `null`). With a stopper the run adds "Stopped by {who}: unclaimed items were released, not deleted."
  - **State words** follow the module's rule (`runs.go:341-350`): `running` "Running (no worker connected)"; `stopped` "Stopped (a claimed chunk is still open)"; `done` "Finished, not complete" with "Not every item was deleted, or backup expiry is not recorded for every deleted item." below; `complete` "Closed without failures" with "This does not by itself mean any item was deleted: a run whose items were all skipped or stopped also closes this way." below. No state word claims a deletion.
  - **Outcome counts:** the eight outcomes in words; a missing key reads "Unknown".
  - **Per store:** store, target, deleted, failed, excluded, pending, all `tabular-nums`, with "Target = deleted + failed + excluded + pending. Excluded items are held, required, not due or stopped." The screen checks the sum; a mismatch or any unknown number reads "These counts do not add up; treat them as unknown." (P04 `spec.md:61`). Deleted at an active store is never shown as backup expiry: "Backup copies are not tracked (backup expiry not connected)."
  - **Items:** ID (link to expiry) and outcome in words; "Showing 50 of {total}." when `more`.
  - **Stop run** when allowed; `stopped` and `finished` are hidden (the state says it).
- "No deletion runs in this workspace." when empty with a real 0; "Showing the newest 50 of {total}." when `more`.
- **Plan proposals** (kind `plan`) below, **fully read-only** (root decision L5): neither Approve nor Withdraw is offered on this screen, even to the proposer whose withdraw the read allows. The list reads "Deletion plans are not connected: these were recorded by the sample data. Approving or withdrawing them is not available here."

### 5. Archive restore (`restore`)

- `not_connected`:
  - "Archive restore is not connected in this build. No archive exists, nothing is restored, and no temporary copy is made."
  - KO: "이 빌드에서는 아카이브 복원이 연결되어 있지 않습니다. 아카이브가 없으며 아무것도 복원하지 않고 임시 사본도 만들지 않습니다."
  - AR: "استعادة الأرشيف غير متصلة في هذا الإصدار. لا يوجد أرشيف، ولا تتم أي استعادة، ولا تُنشأ أي نسخة مؤقتة."
- Another state: "Unknown (code)". No command is offered on this tab.

### 6. Decision records and restore freshness (`decisions`)

- **Lead:**
  - "Deletion records and the decision watermark come from the in-memory retention register, which resets with the process. An independent durable decision ledger, epoch, durable commit and restore freshness checks are not connected, so no deletion is shown as confirmed."
  - KO: "삭제 기록과 결정 워터마크는 프로세스와 함께 초기화되는 메모리 내 보관 레지스트리에서 읽습니다. 독립 결정 원장, epoch, 내구 확정, 복원 최신성 검사가 연결되어 있지 않아 어떤 삭제도 확정으로 표시하지 않습니다."
  - AR: "تأتي سجلات الحذف وعلامة القرار المائية من سجل الاحتفاظ داخل الذاكرة، الذي يُعاد تعيينه مع العملية. سجل القرارات الدائم المستقل والحقبة والتثبيت الدائم وفحوص حداثة الاستعادة غير متصلة، لذلك لا يُعرض أي حذف على أنه مؤكد."
- **Facts:** Decision watermark (installation-wide) "{n}"; Ledger `in_memory` → "In memory (not durable)"; Epoch, Durable commit, Restore freshness check: "Not connected"; unknown values "Unknown (code)".
- **Deletion records table:** sequence, item, class in words, run (link, or "Owned by a run that is not shown in this workspace"), time, "Decision record: not connected".
  - "Only the last {window} deletion records of the installation are scanned; {total} of them are in this workspace." "Showing the newest 50." when `more`. "No deletion records in this workspace." when empty with a real 0.
  - "Sequence numbers are installation-wide, so numbers can be skipped."

## Proposals (shared list, filtered by kind)

- Pending first, then closed, newest first. Each row: number; kind in words; summary in words; state; proposer and created time; the proposal reason on pending rows.
- `applied` rows show the approver (`approver`, login falling back to ID) with the closing reason; `withdrawn` rows show the withdrawer (`closed_by`) with the closing reason; `retired` rows show neither, read "Retired: the retention register no longer holds this proposal" and offer nothing.
- "Applied means recorded in the retention register; it deletes nothing." (KO "적용됨은 보관 레지스트리에 기록되었다는 뜻이며 아무것도 삭제하지 않습니다." AR "«مطبّق» تعني أنه سُجّل في سجل الاحتفاظ، ولا يحذف شيئًا.")
- "No {kind} proposals." when the filtered list is empty.
- **Summaries** (placeholders isolated; an empty value reads "Unknown"):

| Kind | EN | KO | AR |
| --- | --- | --- | --- |
| policy | Set {class} to {p} days plus {g} days grace (total {t} days) | {class} 보관 기간을 {p}일, 유예 {g}일(합계 {t}일)로 설정 | ضبط {class} على {p} يومًا مع فترة سماح {g} يومًا (المجموع {t} يومًا) |
| plan | Delete {n} {class} items in chunks of {c} (lease {s} s): {reason} | {class} 항목 {n}건을 {c}건 단위로 삭제(점유 {s}초): {reason} | حذف {n} من عناصر {class} على دفعات من {c} (مدة الحجز {s} ث): {reason} |
| release | Release hold {hold}: {reason} | 보존 보류 {hold} 해제: {reason} | رفع تعليق الحذف {hold}: {reason} |

- A policy summary with `impact_at_proposal` adds "Impact when proposed (installation-wide): {count} items as of {as_of}". Its count and digest are installation-wide like `impact_now.count` (HB-288 `plan.md:203`); the render pass asserts the label on #7. The `impact_now` block (pending shortenings) reads "Impact now: {count} items in the installation become due earlier, {workspace_count} of them in this workspace (computed {as_of})." followed by a fact row "Impact digest" with the 64-hex value (visual specification §11).
- **Approve** is shown only when allowed (see Commands); **Withdraw** only when allowed.

## Commands: forms, confirms and availability

Every form has an action heading, the reason field ("Reason for this action", 1–256 UTF-8 bytes after trimming), an error summary, Review → a confirm step with exactly one primary button (Confirm) and Cancel. An open form hides its opener and sibling openers (HB-285 rule). The confirm restates what the read showed; its facts are frozen into the attempt's pre-send snapshot before the first send.

### Propose a policy change (`retention.propose_policy`, target `retention-class:<class>`)
- **Offered** when `can_propose`/`propose_blocked` allow it, the class is known and the target equals `retention-class:` + class.
- **Fields:** Period (days) and Grace (days), prefilled with the current values when a policy exists, empty otherwise.
  - **Unknown current policy** (a `policy` object whose `period_days`, `grace_days` or `revision` does not parse): no prefill; the equal-to-current check is skipped (the server's `unchanged` refusal is then the guard); the confirm reads "Current policy: Unknown" and "Whether this shortens the period is unknown; if it does, the approving administrator must review the impact."; the revision in the impact sentence is omitted ("Approval adds a new revision"). A `policy` that is not an object and not `null` is treated the same way. The approve confirm likewise reads "Current policy: Unknown" and relies on `impact_required` from the read.
  - Accept only `^[0-9]{1,5}$` for period and `^[0-9]{1,3}$` for grace; send the canonical decimal (`String(Number(v))`), so "007" is sent as "7".
  - Period 1–36,500: "Enter whole days from 1 to 36,500." Grace 0–365: "Enter whole days from 0 to 365." (`retentioncmds.go:28-29`, `220-227`).
  - Both equal to the current revision: "These are the current values. Change the period or the grace." No request is sent (the server's `unchanged`, `retentioncmds.go:231-234`).
- **Confirm:** class; current "{p} + {g} = {t} days" or "No policy"; proposed "{p} + {g} = {t} days"; the change "{d} days shorter" / "{d} days longer" / "Same total".
  - Impact: "Another administrator must approve it. Approval adds revision {r+1} for {class}; it deletes nothing, because deletion plans are not connected."
  - When the proposed total is shorter than the current: "Because this shortens the period, the approving administrator must review how many items become due earlier. That count is computed after you propose; it cannot be previewed here."
- **Blocker:** `already_proposed` → "Policy proposal #{n} for this class is already waiting for approval." / "Your policy proposal #{n} for this class is waiting for another administrator's approval." ("Your" is decided only from that proposal row: its `approve_blocked` is `own_proposal` or its withdraw is allowed.)

### Approve (`retention.approve`, target `retention-proposal-<n>`)
- **Offered** when `can_approve`, the kind is `policy` or `release`, and the target matches `^retention-proposal-[1-9][0-9]*$` and equals the prefix plus `number`.
- **Policy approval narrowing** (defensive; the server stays authoritative):
  - when `impact_required` is true, `impact_now` must parse: `as_of` a valid time, `count` and `workspace_count` safe non-negative integers, `digest` 64 lowercase hex. Otherwise Approve is not offered and the row reads "The impact could not be read, so this cannot be approved here. Reload.";
  - when `impact_required` is false, `impact_now` must be `null`; no impact fields are sent.
- **Policy confirm:** the summary, proposer and their reason; the class's current policy from the policies row.
  - Not a shortening: "Approval adds this revision now. The total does not get shorter, so no impact review is required. Nothing is deleted."
  - Shortening: facts "Become due earlier (installation)" {count}, "In this workspace" {workspace_count}, "Computed" {as_of}, "Impact digest" (64 hex, its own fact row, wrapping; visual specification §1.5, §11), then: "**Review the impact.** As of {as_of}, {count} items in the installation that are not due now would become due under this period, {workspace_count} of them in this workspace. Held and required items are included in the count and stay protected. Approval sends exactly these values and the digest above; the retention register refuses them if the impact has changed or is more than one hour old. Nothing is deleted: deletion plans are not connected."
  - KO: "**영향을 검토하세요.** {as_of} 기준으로 현재 삭제 대상이 아닌 항목 중 {count}건(설치 전체)이 이 기간에서는 삭제 대상이 되며, 그중 {workspace_count}건이 이 업무 공간의 항목입니다. 보류·참조 중인 항목도 건수에 포함되며 계속 보호됩니다. 승인하면 이 값과 위의 요약값을 그대로 보냅니다. 영향이 바뀌었거나 1시간이 지났으면 보관 레지스트리가 거부합니다. 삭제 계획이 연결되어 있지 않아 아무것도 삭제되지 않습니다."
  - AR: "**راجع الأثر.** اعتبارًا من {as_of}، سيصبح {count} من العناصر في التثبيت غير المستحقة الآن مستحقة بموجب هذه المدة، منها {workspace_count} في مساحة العمل هذه. تشمل الأعداد العناصر المعلّقة والمطلوبة وتبقى محمية. ترسل الموافقة هذه القيم والملخص أعلاه كما هي، ويرفضها سجل الاحتفاظ إذا تغيّر الأثر أو مضت عليه أكثر من ساعة. لا يُحذف أي شيء: خطط الحذف غير متصلة."
  - The "more than one hour" rule is the module's impact window (HB-288 `plan.md:43`, `54`).
- **Release confirm:** the hold ID, its owner, what it covers and its current "cannot preserve" counts; "Approval releases hold {hold} now. Items it covered are no longer protected by it and may be deleted by a later deletion run. Nothing is deleted now, because deletion plans and workers are not connected."
- **After a reload while the form is open** (conflict or refusal), the policy confirm uses the reloaded `impact_now`. When its `count` or `digest` differs from the reviewed values, the confirm adds "The impact changed since you reviewed it: {old} → {new} items." (one isolate for "{old} → {new}", shown above the facts) and needs a new Confirm. A new `as_of` alone is shown without that sentence.
  - **Where the reviewed values live (L13).** A `revision_conflict` or refusal is a definite answer, so the attempt and its snapshot are discarded. The reviewed `count` and `digest` are therefore kept separately, in the module-level draft store under `retention.approve` + target as `reviewed: {count, digest}`: set when the confirm is shown, compared after every reload, and cleared on success, on Cancel, on a definite answer other than `conflict` or `stage`/`refused`, and by `forget()`. They are never sent; only the impact on screen at Confirm is sent.

### Withdraw (`retention.withdraw`)
- **Offered** when `can_withdraw` and the target is well-formed. Confirm: "Withdrawing closes this proposal. Nothing else changes."

### Place hold (`retention.place_hold`)
- **Targets:** an artifact row's `hold_target` (offered when `holdable`/`hold_blocked` allow it and `hold_target` equals `retention-artifact:` + id within 128 bytes) or `holds.tenant_target` (offered when it matches `^retention-tenant:[A-Za-z0-9._:-]+$` within 128 bytes and equals `retention-tenant:` + the session workspace).
- **Fields** (`LiveHoldForm.tsx`):
  - **Owner** (required): a native `<select>` of the access `subjects`, in the order returned, with a first placeholder option "Choose an owner" (value `""`, no default choice). Option text (visual specification §1.6, §11): `\u2066{login} ({id})\u2069`; a `login` that is `""` or missing falls back to `\u2066{id}\u2069`. A subject whose `user_id` is not 1–128 bytes of `[A-Za-z0-9._:-]` (the command `ID` field rule, `modules/commands/execute.go:31-32`) is left out of the list. Nothing chosen: "Choose an owner." in the error summary. Hint: "The person accountable for this hold." See decision 1.
  - **Hold reason** (required, single-line input): trimmed, 1–200 UTF-8 bytes, no control characters: "Enter a hold reason.", "The hold reason is too long (at most 200 bytes).", "Remove line breaks and control characters."
  - **Data class** (workspace target only): "All classes" (no `class` sent) or one class. Never sent with an artifact target.
  - The action reason (rationale) as for every command. The two texts are labelled so they are not confused: the hold reason is stored with the hold; the action reason records why you placed it.
- **Confirm:** the target ("Item {id}" with its class, access and standing, or "This workspace" with the class choice), the owner, the hold reason.
  - "**Takes effect now; no second approval: you alone place this hold.** It stops deletion of covered items that still exist. It cannot bring back items already deleted, being deleted or partly deleted; the result lists them."
  - For an item recorded as deleted: "This item is already recorded as deleted. The hold cannot preserve it." For a damaged item: "This item is partly deleted. The hold cannot restore what is gone." For an item in a run: "This item is in deletion run {run}. The hold protects it only if the run has not started deleting it."
  - For a workspace hold: "It covers every registered item of this workspace{ of class X}, including items registered later."
  - Release: "Releasing it later needs two other administrators: one to propose and one to approve. If this workspace has fewer than two administrators besides you, this hold cannot be released."
  - KO: "**지금 바로 적용되며 두 번째 승인이 없습니다. 이 보존 보류는 귀하 혼자 등록합니다.** 대상 항목 중 아직 남아 있는 항목의 삭제를 막습니다. 이미 삭제되었거나 삭제 중이거나 일부만 삭제된 항목은 되살리지 못하며, 결과에 그 항목이 표시됩니다."
  - AR: "**يسري الآن دون موافقة ثانية: أنت وحدك تضع هذا التعليق.** يمنع حذف العناصر المشمولة التي ما زالت موجودة. ولا يمكنه إعادة العناصر المحذوفة أو قيد الحذف أو المحذوفة جزئيًا، وتُدرج في النتيجة."
- **Blocker:** `target_too_long` → "This item's identifier is too long to name in a command, so it cannot be held here."
- **Edits after an unknown outcome:** when a pending attempt exists, the form adds "If you change anything, a new hold is placed in addition to any earlier one."
- **`stale` on place hold (M4).** An `idempotency_conflict` or expired key can follow a request that was applied. For place hold the generic "Start again" is not used:
  - the screen reloads, keeps the attempt's "The last request may have been applied." warning and does not renew the attempt by itself;
  - after the reload it looks for a hold placed by the reader (`placer.id` equals the session user) whose owner ID, hold reason and scope (the artifact in `scope.artifacts`, or `scope.workspace` with the same class list) equal the frozen values; if one is listed it reads "Hold {id}, placed by you with the same target, owner and reason, is already listed. It may be the result of your earlier request." with a link to `view=holds&hold=<id>`;
  - a new attempt is made only through an explicit "Place another hold" action, whose form shows the edit warning above and starts a new key.
  - Other commands keep the generic "Start again" (their repeats are refused, not duplicated).

### Propose release (`retention.propose_release`, target `retention-hold:<id>`)
- **Offered** when `can_propose_release`/`release_blocked` allow it and `target` equals `retention-hold:` + id within 128 bytes.
- **Fields:** Release reason (single line, same rule as the hold reason) and the action reason.
- **Confirm:** the hold, its owner, scope and "cannot preserve" counts; "Another administrator who did not place this hold must approve it. Approval releases the hold; nothing is deleted now."

### Stop run (`retention.stop_run`, target `retention-run-<n>`)
- **Offered** when `can_stop`/`stop_blocked` allow it and `run.target` matches `^retention-run-[1-9][0-9]*$` with `run.label` equal to `run/` + the same number.
- **Confirm:** run, class, state, "{pending} items not yet claimed" from `counts.pending` ("Unknown" when missing).
  - "**Takes effect now; no second approval: you alone stop this run.** Items in chunks no worker has claimed leave the run with the outcome Stopped and are no longer blocked by it. Items already deleted stay deleted; stopping is not a restore. Chunks already claimed are not recalled." (P04 `spec.md:86`; `runs.go:250-275`)
  - KO: "**지금 바로 적용되며 두 번째 승인이 없습니다. 이 작업은 귀하 혼자 중지합니다.** 작업자가 맡지 않은 묶음의 항목은 '중지됨'으로 작업에서 빠지고 더 이상 이 작업 때문에 차단되지 않습니다. 이미 삭제된 항목은 그대로이며 중지는 복원이 아닙니다. 이미 맡은 묶음은 회수하지 않습니다."
  - AR: "**يسري الآن دون موافقة ثانية: أنت وحدك توقف هذه العملية.** تخرج عناصر الدفعات التي لم تتسلمها أي جهة من العملية بنتيجة «موقوف» ولا تبقى محجوبة بسببها. تبقى العناصر المحذوفة محذوفة، والإيقاف ليس استعادة. لا تُسترد الدفعات التي تم تسلمها."

### Availability parsing (fail closed)
- Every pair uses P02's `hint` (`integrations.ts:184-188`): allowed only when the flag is `true` and the blocker is exactly `""`; a true flag with a non-empty blocker is not allowed; a non-string blocker, or an empty blocker with the flag not true, is `missing`. Pairs: `can_propose`/`propose_blocked`, `holdable`/`hold_blocked`, `can_propose_release`/`release_blocked`, `can_stop`/`stop_blocked`, `can_approve`/`approve_blocked`, `can_withdraw`/`withdraw_blocked`.
- **Targets** are used only when they match their pattern, equal the value built from the row's ID, and are 1–128 bytes of `[A-Za-z0-9._:-]` (`modules/commands/types.go:299-310`). Otherwise no command is offered and the row reads "No command is available for this row."
- **All commands:** the page `revision` must be non-empty.

### Blocker display policy and sentences
- **Hidden only:** `not_pending` on closed rows; `not_proposer`; `released` on release; `stopped` and `finished` on Stop. These produce no blocker line, but the row note carries a state-based sentence so no row is silently missing an action (L1):
  - closed proposal: the state tag plus the applied, withdrawn or retired line (no extra sentence);
  - `not_proposer` on a pending row: "Only the proposer can withdraw it.";
  - `released`: "This hold is released; there is nothing to release.";
  - `stopped`: "This run is stopped; a claimed chunk is still open.";
  - `finished`: "This run has finished, so it cannot be stopped."
- **Explained:** every other code, plus `not_pending` on a pending row (unexpected). Every sentence has EN, KO and AR copy in `live-codes.ts`. An unknown code reads "This is not available now (code)"; `missing` reads "This is not available now."

| Code | Where | EN |
| --- | --- | --- |
| `already_proposed` | propose policy, propose release | Policy proposal #{n} … / Release proposal #{n} for this hold is already waiting for approval. ("Your …" form as above; without a row: "A proposal is already waiting for approval.") |
| `own_proposal` | approve | You proposed this. Another administrator must approve it. |
| `not_pending` | approve, withdraw (pending row) | This is no longer waiting for a decision. Reload. |
| `retired` | approve, withdraw | The retention register no longer holds this proposal. Nothing can be done with it here. |
| `not_console_kind` | approve | Deletion plans cannot be approved or withdrawn from the console in this build. |
| `placer` | approve / propose release | You placed this hold, so you cannot approve its release. / You placed this hold, so you cannot propose its release. Two other administrators must propose and approve it. |
| `no_eligible_approver` | propose release | Fewer than two administrators besides the person who placed this hold can act, so its release cannot be proposed and approved. |
| `released` | approve | This hold is already released. |
| `impact_changed` | approve | The impact changed after it was reviewed. Reload and review it again. |
| `target_too_long` | place hold | This item's identifier is too long to name in a command, so it cannot be held here. |
| `not_found` | approve, propose release | The retention register no longer knows this hold or proposal. |
| `limit` | approve, propose release | A retention register limit is reached. |
| `conflict` | approve, propose release | Another record already uses this identifier. |
| `invalid` | approve, propose release | The retention register could not check this request. |

## Attempts, refusals and reactions

- **Attempts.** A request ID and an idempotency key are kept per command and target in the scoped store. They are reused only to resend identical values (fields and reason) after an unknown outcome (network failure or `busy`), even if a reload moved the revision. Any edit or definite answer discards them, including `partial`. A pending attempt shows "The last request may have been applied. Check the list, or send it again."
- **Pre-send snapshot.** Before its first send an attempt stores:
  - the listed proposal numbers (for `propose_policy` and `propose_release`, for the duplicate check);
  - what its confirm stated: for approve the summary, current policy and the impact (`as_of`, `count`, `workspace_count`, `digest`) or the hold facts; for propose policy the current and proposed values; for place hold the target, owner (ID and login), hold reason, class and item standing; for propose release the hold facts and release reason; for stop the run, state and pending count;
  - context for vanished-row notices (proposal number, hold, run, class or item).
- **Resend.** A pending attempt keeps its command usable even if the reloaded row no longer allows it. The resend skips every read-based check; only the frozen target must still be well-formed. It reuses the frozen target, fields, reason and `expected_revision`, and its confirm restates the frozen snapshot with "This repeats your earlier request exactly as it was sent." An approve resend sends the frozen impact, never the reloaded one.
- **Drafts.** The reason, hold reason, release reason, period and grace are kept per command and target while the session lasts. The owner choice is not a draft (the list is re-read). Impact values are never drafts.
- **Reactions:**
  - **`invalid` with fields:**
    - `period`, `grace`, `owner`, `reason` → that field (owner: "This person can no longer own a hold. Choose another." and the owner list is re-read; `reason` maps to the hold reason or the release reason);
    - `rationale` → the reason field;
    - `impact_as_of`, `impact_count`, `impact_digest` → "The impact could not be sent as read. Reload and review the impact again." and a reload; never the reason field;
    - `class` and any other field (`target_id`, `expected_revision`, `idempotency_key`, `request_id`, `client`, the excess marker, an unknown name) → "The request could not be sent as built. Reload and try again."
  - **`invalid` without fields** → the reason field ("The reason is too long" over 256 bytes, otherwise "Check the reason").
  - **`stage` or `refused`:** reload, then the row's current blocker sentence. When the reloaded row shows no blocker (or there is no row), a kept per-command sentence:
    - approve: "The retention register refused this approval. Nothing changed. For a shorter period, the impact may have changed or be more than an hour old: review it again.";
    - withdraw: "The retention register refused this withdrawal. Nothing changed.";
    - propose policy: "The policy proposal was refused. Nothing changed. Another proposal may already be waiting, the values may equal the current policy, or you may have too many pending proposals.";
    - place hold: `refused` (`prerequisite_failed`) is definite: "The hold was not placed. Nothing changed. This workspace already has 100 holds in force, or the retention register's hold limit is reached." (`retentioncmds.go:280-289`; module `ErrLimit` at 10,000 live holds, `records.go:154-156`); `stage` (an identifier collision, injection only): "The hold was not placed. Nothing changed. Reload and try again.";
    - propose release: "The release proposal was refused. Nothing changed. The hold may already be released, a release may already be waiting, or you placed it.";
    - stop run: "The stop was refused. Nothing changed. The run may already be stopped or finished."
    - The kept sentence clears on the next definite answer for that command and target, or when the reloaded row shows a blocker or closes. Kept sentences for rows that left the read are dropped (P03 `pruneRefusals`).
  - **`conflict`** → reload, keep the reason and fields, and ask to confirm again (approve: with the reloaded impact, see above). If the reloaded read no longer offers the command, the form closes, focus moves to the row note, and the row's state or blocker explains why.
  - **`denied` on a command:** reload. If the read succeeds: "This proposal / hold / run / item / class is no longer available." If the read is denied, the screen-level denied block. With a pending attempt and a proposal missing from the reloaded read (a closed entry evicted beyond 200; not reachable with the sample seed): "Your earlier request may have been applied. The proposal is no longer listed." and the attempt is discarded.
  - **Pending attempts whose row vanished** get a tab-level notice: "This item is no longer listed. Your earlier request may have been applied." with Dismiss.
  - **Other failures:** `unauthenticated` → sign in again; `limited` → the wait; `busy` or `unavailable` → retry with the same attempt; `stale` → "Start again" (renew the attempt), except place hold (see Place hold, M4); `partial` → the operation text with its ID, a reload, and the attempt discarded; anything else → "Failed".
- **Duplicate results.** A `propose_policy` or `propose_release` whose returned `proposal` is in the snapshot announces "This was already proposed as #{n}". Through the real API this is not expected (duplicates are `invalid_stage`, `retentioncmds.go:228-230`, `328-329`); it is covered by model and service tests with a stubbed success.
- **Counted sentences (L11).** Every sentence with a count ("{n} items …", "Showing 50 of {total}", the cannot-preserve parts, the stop result) chooses its form with `Intl.PluralRules` for the locale. `live-codes.ts` holds full sentences per category: EN `one`/`other`; KO `other` only; AR all six (`zero`, `one`, `two`, `few`, `many`, `other`), written as whole sentences rather than a noun swapped into one template. Model tests cover AR 0, 1, 2, 3, 11 and 100.
- **Successes** name the result in words:
  - approve policy: "Policy proposal #{n} applied. Nothing was deleted."; approve release: "Release proposal #{n} applied. Hold {hold} is released. Nothing was deleted." ("is released" only when the reloaded read shows it released);
  - withdraw: "Proposal #{n} withdrawn.";
  - propose policy: "Policy proposal #{n} is waiting for another administrator's approval.";
  - place hold, `held`: "Hold {hold} is in force."; `partially_preserved`: "Hold {hold} is in force, but it cannot preserve {a} items already deleted, {b} items whose deletion outcome is pending and {c} items partly deleted." (only non-zero parts; an unparsable count reads "Unknown");
  - propose release: "Release proposal #{n} for hold {hold} is waiting for approval by another administrator who did not place the hold.";
  - stop: "{run} stopped: {n} unclaimed items were released, not deleted. Nothing was restored." followed by the run's state words from the reloaded read (for the sample `run/2`, `done` → "Finished, not complete");
  - unknown result states read "Unknown (code)".

## Unknown, zero, not connected and other scope

- The page `revision` is opaque, shown and sent as is; empty reads "Unknown" and disables commands.
- Numbers are safe non-negative integers or `null` (P02 `count`, `integrations.ts:173-175`); `null` reads "Unknown"; a real 0 reads "0". This covers `revision` (policy and run), `period_days`, `grace_days`, `pending`, `release`, every count map value, `total`, `next_chunk`, `chunks`, `chunk_size`, `lease_seconds`, `damaged`, store numbers, `watermark`, `window`, `sequence`, `number`, `artifact_count`, impact counts. A uint64 beyond the safe range reads "Unknown" and a proposal or run number beyond it offers no command.
- Count maps: a missing or non-numeric key reads "Unknown" for that key only.
- Booleans (`held`, `damaged`, `shared`, `other_scope`, `released`, `more`, `outcomes_more`, `scope.workspace`, `impact_required`, every `can_*`) are `null` when not a boolean; a null `can_*` fails closed; a null `more` reads "There may be more."
- Missing arrays are `null` and read "Unknown"; empty arrays read the tab's empty sentence.
- Times that are empty, unparseable or year 1 read "Unknown"; `due_at: null` reads "Never (no policy)".
- `not_connected` reads "Not connected", never "None" or "0". `other_scope` on a run reference reads "Owned by a run that is not shown in this workspace", never "None". `counts: null` with `counts_scope: other_scope` reads the other-scope sentence, never 0.
- People: `null` reads "None" only for `stopper` ("Not stopped") and `approver`/`closed_by` on pending rows (not shown); otherwise "Unknown". A `null` login falls back to the ID.
- An unknown enum value reads "Unknown (code)", never blank.

## Layout (defers to the visual specification)

The ui-designer's visual specification (`hb289-visual-spec.md`, kept with the ticket's working notes; not owner acceptance) is the layout authority. Where this plan and the specification differ on layout, the specification wins. Its binding points:
- **Wrapper and container.** `div.p04-live` with `container-type: inline-size` inside the live wrapper; full width, no max-width page wrapper, preview grid or aside. All rules are scoped under `.p04-live` with `p04-` classes (§2).
- **Container sizes, not viewport sizes.** S below 720px, M 720–1079px, L 1080px and up, measured on `.p04-live` (the content is about 290px narrower than the viewport because of the shell rail; §0, §1.4). Every layout switch (fact grids, count strips, the hold grid, proposal grid, table stacking, nowrap IDs, confirm fact columns) is an `@container` rule.
- **Viewport media only for four edge cases** at `@media (max-width: 767px)`: the 3px control inset, full-width buttons, the shell tab grid `.planning-main:has(.p04-live) .planning-tabs` (six tabs as three rows of two, at least 44px each; a clone of `recovery/live.css:543-554`, a residual), and the scope facts at full width (§2).
- **Tables.** `div.live-table > table.p04-table` with a `colgroup` of percentage classes, one `tbody.p04-row` per data row with an optional follow row for the row note or form; stacked into labelled rows below 720px, and `p04-table.p04-wide` (policies, expiry items) below 1080px; the shell `th` rule overridden as `recovery/live.css:408-414`; IDs nowrap only at L (§2, §5).
- **Per tab:** column sets and widths for policies (§6) and expiry items (§7), the hold grid (§8), run articles with Stop right after the outcome counts and the store and item tables stacked and capped (§9), restore and decisions (§10), proposals, forms and confirms (§11).
- **Type, rhythm, borders, tags and controls** (§3): prose capped at 68ch; neutral `--border-row`/`--border-control` only; tags only for states; controls 44px high at natural width with 440px field caps.
- **Empty values** (§14): Unknown, real zero, Not connected and other scope are styled and worded distinctly.
- **RTL** (§13): logical properties only; one ltr isolate per compound value; dates as two nowrap parts; free text `span.p04-free[dir=auto]`; KO `keep-all`.
- **Lessons carried from P02/P03:** no overlapping tables (stacked, never side by side); dates never broken inside the year; IDs not broken at hyphens at L; no rule above the first fact of a block; one primary button during a confirm; choice controls no wider than 21px inline with their labels.

## Accessibility and focus

- Reading order: title → scope bar → statement and notes → tab heading → lead → lists → actions (P04 `spec.md:133`).
- Each table has a caption (visually hidden where the heading already says it) and `scope` on header cells; stacked rows keep the labels.
- Status is never colour only: standings, access, outcomes and states are words, with neutral tags.
- Count strips are `<dl>` lists; each number has its label.
- The owner `<select>` and the class `<select>` are native controls with visible labels.
- Focus rules as in Host; a confirm step focuses its heading; the error summary is focused on a client failure and links to fields.

## Vocabularies (EN / KO / AR)

All are in `live-codes.ts` with an "Unknown (code)" fallback; tests check every key in every language. KO and AR are AI drafts pending terminology review.

| Group | Code | EN | KO | AR |
| --- | --- | --- | --- | --- |
| Class | source | Source records | 원천 자료 | سجلات المصدر |
| Class | final | Final documents and manifests | 확정 문서·매니페스트 | المستندات والبيانات النهائية |
| Class | claim | Duplicate-prevention records | 중복 방지 기록 | سجلات منع التكرار |
| Class | identity | Identity and access records | 신원·접근 기록 | سجلات الهوية والوصول |
| Class | conversation | Assistant conversations | Assistant 대화 | محادثات المساعد |
| Class | temporary | Temporary files | 임시 파일 | الملفات المؤقتة |
| Class | projection | Caches and projections | 캐시·프로젝션 | ذاكرات التخزين المؤقت والإسقاطات |
| Standing | eligible | Due for deletion | 삭제 대상 | مستحق للحذف |
| Standing | held | On hold | 보존 보류 중 | معلّق الحذف |
| Standing | referenced | Required by another item | 다른 항목이 참조 | مطلوب لعنصر آخر |
| Standing | not_due | Not due yet | 아직 기한 전 | لم يحن موعده |
| Standing | no_policy | No policy | 정책 없음 | لا توجد سياسة |
| Standing | in_run | In a deletion run | 삭제 작업 중 | ضمن عملية حذف |
| Standing | deleted | Recorded as deleted | 삭제로 기록됨 | مسجّل كمحذوف |
| Access | ok | Accessible | 접근 가능 | متاح |
| Access | blocked | Blocked | 차단됨 | محجوب |
| Access | deleted | Recorded as deleted | 삭제로 기록됨 | مسجّل كمحذوف |
| Hold outcome | not_preserved | Already deleted (not preserved) | 이미 삭제됨(보존 불가) | محذوف بالفعل (غير محفوظ) |
| Hold outcome | outcome_pending | Deletion outcome pending | 삭제 결과 대기 | نتيجة الحذف معلّقة |
| Hold outcome | partially_deleted | Partly deleted | 일부 삭제됨 | محذوف جزئيًا |
| Store result | deleted / failed | Deleted / Failed | 삭제됨 / 실패 | محذوف / فشل |
| Run state | running | Running (no worker connected) | 실행 중(작업자 미연결) | قيد التشغيل (لا توجد جهة متصلة) |
| Run state | stopped | Stopped (a claimed chunk is still open) | 중지됨(맡은 묶음 진행 중) | موقوفة (دفعة مستلمة ما زالت مفتوحة) |
| Run state | done | Finished, not complete | 종료됨(미완료) | انتهت، غير مكتملة |
| Run state | complete | Closed without failures | 실패 없이 종료됨 | أُغلقت دون إخفاقات |
| Run outcome | pending | Not yet claimed | 아직 맡지 않음 | لم يُتسلَّم بعد |
| Run outcome | deleting | Being deleted | 삭제 중 | قيد الحذف |
| Run outcome | deleted | Deleted | 삭제됨 | محذوف |
| Run outcome | failed | Failed | 실패 | فشل |
| Run outcome | held | Skipped: on hold | 제외: 보존 보류 | مستبعد: معلّق الحذف |
| Run outcome | referenced | Skipped: required | 제외: 참조됨 | مستبعد: مطلوب |
| Run outcome | not_due | Skipped: not due | 제외: 기한 전 | مستبعد: لم يحن موعده |
| Run outcome | stopped | Stopped | 중지됨 | موقوف |
| Hold state | false / true (released) | In force / Released | 유지 중 / 해제됨 | سارٍ / مرفوع |
| Counts scope | workspace / other_scope | (counts shown) / Not shown: also covers data outside this workspace | (건수 표시) / 표시 안 함: 이 업무 공간 밖 자료도 포함 | (تُعرض الأعداد) / غير معروضة: يشمل بيانات خارج مساحة العمل هذه |
| Origin | artifact_anchor | Each item's recorded start time | 항목별 기록된 기산 시각 | وقت البدء المسجّل لكل عنصر |
| Connection | not_connected | Not connected | 연결되지 않음 | غير متصل |
| Ledger | in_memory | In memory (not durable) | 메모리 내(비내구) | في الذاكرة (غير دائم) |
| Decision | not_connected | Decision record: not connected (not confirmed) | 결정 기록: 미연결(미확정) | سجل القرار: غير متصل (غير مؤكد) |
| Population | registered_sample_artifacts | Registered sample items in this workspace | 이 업무 공간에 등록된 샘플 항목 | عناصر العينة المسجّلة في مساحة العمل هذه |
| Proposal kind | policy / plan / release | Retention policy / Deletion plan / Hold release | 보관 정책 / 삭제 계획 / 보존 보류 해제 | سياسة الاحتفاظ / خطة حذف / رفع تعليق الحذف |
| Proposal state | pending / applied / withdrawn / retired | Pending / Applied / Withdrawn / Retired | 승인 대기 / 적용됨 / 철회됨 / 폐기됨 | بانتظار الاعتماد / مطبّق / مسحوب / مُسقَط |
| Result state | held / partially_preserved | In force / In force, not all preserved | 유지 중 / 유지 중(일부 보존 불가) | سارٍ / سارٍ مع عدم حفظ الكل |

**Terminology citations.** One term per concept (`.agents/rules/product-terminology.md`; `docs/ui/design-system/terminology.md` rules 1, 5, 9):
- "Hold" uses KO "보존 보류" and AR "تعليق الحذف" as P03's event kind `hold` (`recovery/live-codes.ts:99`, HB-287 `plan.md:584`); the registered tab label "보류·참조" stays. The product glossary's billing "hold" uses AR "إجراءات التعليق" (`docs/ui/terminology/product-glossary.md:111`) for a different concept (payment and inquiry holds), so the two are not merged; this is a terminology review question.
- "Deletion record" (tombstone) reuses P03's domain `deletions` "Deletion records / 삭제 기록 / سجلات الحذف" (HB-287 `plan.md:568`); "Decision watermark / 결정 워터마크 / علامة القرار المائية" reuses P03 (HB-287 `plan.md:674`).
- "Workspace" KO "업무 공간" follows the spec's "설치·업무 공간" (`spec.md:25`).
- The tab and title labels are the registered KO labels (`screens.json` P04: 보관 정책 / 사용량·만료 / 보류·참조 / 삭제 작업 / 아카이브 복원 / 결정 기록·복원 최신성; title 데이터 보관·삭제). EN view titles: "Retention policies", "Usage and expiry", "Holds and references", "Deletion runs", "Archive restore", "Decision records and restore freshness". AR: "سياسات الاحتفاظ", "الاستخدام والانتهاء", "التعليقات والمراجع", "عمليات الحذف", "استعادة الأرشيف", "سجلات القرار وحداثة الاستعادة".
- The `claim` class is a duplicate-prevention record (HB-256 `plan.md:42`), not an insurance or payment claim; its label avoids "claim".
- "Revision" is kept in Latin script in KO, as in other admin screens.

## External references (opened 2026-11-17)

Root opened each page with WebFetch on 2026-11-17. The fetch returns the page text converted by a small model, so these notes summarise that text; screenshots were not inspected. **Gate:** these references are recorded before implementation starts, and the ui-designer's render comparison checks the adopted patterns against them.
- **Google Vault, How retention works** (https://support.google.com/vault/answer/2990828, redirected to https://knowledge.workspace.google.com/vault/retention/how-retention-works). Observed: "Holds take precedence over retention rules"; once data is purged it cannot be retrieved; retention rules keep data for a number of days or indefinitely, while holds keep data until the hold is removed; the default rule covers a whole service, and custom rules are scoped by organizational unit. Adopted: holds listed separately from policies, a hold preventing deletion regardless of the period, and the statement that nothing purged is brought back. Rejected: indefinite retention and organizational-unit scoping (the module has neither).
- **Microsoft Purview, Learn about retention policies and labels** (https://learn.microsoft.com/en-us/purview/retention). Observed: the principles of retention, "retention always takes precedence over permanent deletion, and the longest retention period wins"; eDiscovery holds preserve content until an administrator releases them; Preservation Lock stops anyone making a policy "less restrictive". Adopted: shortening treated as the change that needs a reviewed impact, and holds winning over due dates. Rejected: Preservation Lock and label publishing (not in the module).
- **Microsoft Purview, Disposition of content** (https://learn.microsoft.com/en-us/purview/disposition). Observed: reviewers per stage (up to five stages); content is deleted only after the final reviewer approves; an optional auto-approval timeout (7–365 days) disposes items with no action; pending and disposed items are separate lists. Adopted: a reviewer separate from the proposer and pending versus closed proposal groups. Rejected: auto-approval after a timeout and multi-stage chains.
- **Amazon S3, Locking objects with Object Lock** (https://docs.aws.amazon.com/AmazonS3/latest/userguide/object-lock.html). Observed: "A legal hold provides the same protection as a retention period, but it has no expiration date"; "Legal holds are independent from retention periods and are placed on individual object versions"; governance mode lets users with a special permission bypass retention. Adopted: the hold shown per item as a flag separate from standing (the "On hold" tag beside the standing word). Rejected: governance-mode bypass and single-person hold removal (release here needs two administrators who did not place it).
- **GOV.UK Design System, Summary list** (https://design-system.service.gov.uk/components/summary-list/). Observed: a `<dl>` of key–value rows to "show information as a list of key facts", not for tabular data; optional row actions such as "Change" with visually hidden context; borders between rows help users who zoom. The page says nothing about mobile layout. Adopted: labelled key–value facts (confirm facts, hold and run facts) with neutral row rules. Rejected: per-row Change links (the confirm has Back to edit instead). The stacked table rows at S are the ui-designer's choice and are not supported by this page.

## Rules carried from the HB-283, HB-285 and HB-287 reviews

All are part of this design from the start:
- **Host and remount:** live branch before the preview wrapper, locale-only key, hooks unconditional, remount per view.
- **URL state** validated; **focus** to the status after success, the row note after refusal, the opener after Cancel; summary links to fields.
- **Stale responses:** the request sequence (also for the owner list).
- **Store:** scoped by workspace, user and session, cleared on sign-out; pre-send snapshots; resend only with identical values; the resend confirm restates the frozen snapshot and skips read-based narrowing.
- **Mobile:** stacking at 390 without horizontal scroll; one primary button during a confirm; an open form hides its opener and siblings and has an action heading.
- **RTL:** one ltr isolate per compound ID; FSI for dates; dates kept whole.
- **Unknown versus zero and empty;** a missing list is never shown as empty; year-1 times read "Unknown".
- **Blockers:** a sentence for every code with an unknown fallback; rows without a well-formed target offer no command; a missing number fails closed; the proposer's reason on pending rows and in the approve confirm; logins fall back to IDs; "Showing the newest 50."
- **Refusals:** the kept sentence when the reloaded blocker is empty, and its clearing; command `denied` → reload → "no longer available"; a vanished pending attempt gets a tab-level notice.
- **Truthful statements:** what each decision does and does not do; single-administrator effects say so.
- **Terminology:** user-facing copy follows `.agents/rules/product-terminology.md`, the product glossary and `docs/ui/design-system/terminology.md`.

# Verification plan

- **Service tests** (`retention.test.ts`):
  - hardened parsers for every section: missing arrays, wrong types, unknown enums, unsafe and negative integers, uint64 beyond the safe range, `null` people and logins, missing count keys, `counts: null` with `other_scope`, `decision` absent on `ok` items, `due_at: null`, run references with mismatched target and label, `summary` with the wrong member for its kind, `impact_now` present with `impact_required` false;
  - whole-page `not_configured`, an unknown page state, `restore` `not_connected` and an unknown restore state, `--seed none` shapes (empty lists with real zeros);
  - command envelopes: approve with and without the three impact fields (sent verbatim, `as_of` with its fraction), propose policy with canonical integers, place hold with and without `class`, propose release with `reason`, withdraw and stop without fields;
  - resend identity; result fields (`proposal`, `target`, `hold`, the three counts, `run`, `stopped`);
  - stubbed `invalid_stage`, `prerequisite_failed`, `invalid_request` with each declared field, and `internal_error` with an operation ID mapped to `partial`.
- **Model tests** (`live-model.test.ts`):
  - availability parsing fails closed for all six pairs;
  - the blocker display policy: hidden-only codes hidden; every explained code with its sentence in EN, KO and AR; `already_proposed` naming the pending number, own and other, and the fallback; `placer` wording per action; `missing` and the unknown fallback;
  - target patterns and equality: proposal, class, artifact (109 and 110 bytes of ID), tenant (equal and not equal to the session workspace), hold, run (target/label mismatch, non-canonical);
  - propose-policy validation: 0, 1, 36,500, 36,501, grace 365 and 366, "1.5", "-1", "007" sent as "7", equal to current refused on the client, no policy (any values allowed);
  - the shortening sentence in the propose confirm only when the total is shorter;
  - approve narrowing: `impact_required` with an unparsable `impact_now` → not offered with its sentence; `impact_required` false with a non-null `impact_now` → not offered;
  - the approve impact confirm with known values; the "impact changed" sentence only when count or digest differ after a reload; the resend sends the frozen impact, not the reloaded one;
  - hold reason and release reason: empty, whitespace, 200 and 201 bytes (multi-byte KO and AR), a line break, a C1 control;
  - place-hold confirm sentences for deleted, damaged and in-run items and for a workspace hold with and without a class; `class` never sent with an artifact target;
  - owner list: failed read, empty list, a list; no default selection;
  - stop confirm with a known and an unknown pending count;
  - the per-store sum check: balanced, unbalanced, an unknown number;
  - unknown versus zero versus not connected versus other scope for every numeric field, count map, boolean, list and time; `due_at: null`; a `null` `more`;
  - every vocabulary key in EN, KO and AR; no copy claims a deletion, archive, restore, export or confirmed decision except statements that none happens;
  - reactions: every field mapping, fieldless `invalid`, kept sentences per command and their clearing, `denied` → "no longer available" per target kind, the evicted-proposal sentence, the vanished-attempt notice, the place-hold edit warning with a pending attempt;
  - the attempt store (identical values only, scoped by session, cleared by `forget`, discarded on `partial`);
  - the duplicate announcement with a stubbed success, and none when a resend replays a stored success;
  - the URL selection validation for every parameter and tab; every href the screen builds starts with `#/retention?` (M1);
  - `Free` renders `span[dir=auto]`, never `bdi`; `Who` uses `p04-login`;
  - run state words for `running`, `stopped`, `done`, `complete` and an unknown state, the chunks sentence, and the stopper sentence present only with a stopper (M2);
  - the decision line from the field value: `not_connected`, another code, missing while blocked or deleted, missing while ok (L2);
  - the hidden-only state sentences (L1); the released-hold heading (L4); plan proposals offer no command even with `can_withdraw` true (L5);
  - owner options: login fallback to ID, an invalid `user_id` left out, the placeholder and "Choose an owner." (L6);
  - unknown current policy in propose and approve confirms (L8);
  - plural forms through `Intl.PluralRules` for EN, KO and AR (0, 1, 2, 3, 11, 100) (L11);
  - the reviewed impact kept across a conflict in the draft store and cleared on success, Cancel and other definite answers (L13);
  - the place-hold stale reaction: no automatic renewal, the matching-hold sentence with and without a match, the explicit new attempt (M4);
  - the place-hold `refused` and `stage` sentences.
- **Checks:** `npx tsc --noEmit`, `npm test`, `npm run build`; changed files under 1,500 lines; no comments.
- **Playwright QA** against the real API (`hb-api --seed sample`, built from this branch's base) with the console preview (`HB_GATED=1 vite preview`). Each group starts on a fresh API with an empty data directory (as `qa287run.sh`), so no group depends on another or on order.
  - **Render pass R** (fresh API, no commands), every tab × EN/KO/AR × 1440/390 (36 renders), as the first administrator:
    - `data-screen-id="P04"` and `data-source-state="live"` on the wrapper (never P01; L14); no horizontal scroll; `dir` per locale; computed borders of tags, notices and selected rows neutral; one primary button in an open confirm (one confirm opened per tab that offers a command); the measured `.p04-live` width and its container size (S/M/L) recorded per render (visual specification §0, §2);
    - the statement, values note, permission note and each tab's lead and not-connected sentences present;
    - policies: `source` revision 1, period 365, grace 30, total 395, pending #7 sentence; `claim` "No policy"; #7 shows "Impact when proposed (installation-wide): 2 items" (L12) and the "Impact digest" fact row;
    - expiry: `source` counts deleted 1, in run 1, held 1, referenced 1, not due 3, eligible 0; `identity` "No registered items"; `sample-source-1` "Recorded as deleted" with "Decision record: not connected"; `sample-temporary-2` standing "In a deletion run" with the "On hold" tag; the population note;
    - holds: `sample-hold-temporary` "1 already deleted (not preserved)" and outcome `sample-temporary-1`; `sample-hold-dispute` counts 0 and the `no_eligible_approver` sentence; references `sample-source-3`;
    - deletion: `run/1` "Finished, not complete" with "2 of 2 chunks finished", database 4 = 2 + 0 + 2 + 0; `run/2` "Running (no worker connected)" with Stop; plan proposals #4, #5 with no Approve or Withdraw and the "not available here" sentence (L5); not-connected list;
    - restore: the not-connected sentence;
    - decisions: watermark 2, two deletion records with "not connected", "In memory (not durable)";
    - dates not broken inside the year; IDs not broken at hyphens at 1440.
  - **Group A, as the first administrator** (`admin@example.invalid`):
    - #7 shows Approve; #6 shows "You proposed this" and Withdraw;
    - approve #7: the confirm shows count 2, workspace count 2, a 64-hex digest and "Nothing is deleted"; the request body carries exactly the read's `impact_as_of`, `impact_count`, `impact_digest` (captured by route); after it `source` reads revision 2, 180 + 30, and expiry `source` reads eligible 2, not due 1;
    - propose a `claim` policy 3650 / 0 → #8 pending, "Your policy proposal #8…"; withdraw #8 → withdrawn; Propose offered again;
    - the client refuses `final` 3650 / 0 as the current values with no request sent;
    - place hold on `sample-source-5`, owner `admin2@example.invalid` → "Hold retention-hold-1 is in force."; the new hold reads placer `admin@example.invalid` and the `no_eligible_approver` sentence;
    - place hold on `sample-source-1`: the confirm says it is already recorded as deleted; the result reads "cannot preserve 1 item already deleted" (singular via `Intl.PluralRules`; `retention-hold-2`);
    - the owner select shows the placeholder "Choose an owner" and options as "login (ID)"; Review with no owner shows "Choose an owner.";
    - stop `run/2`: the confirm states 2 not yet claimed and "stopping is not a restore"; the notice reads "run/2 stopped: 2 unclaimed items were released, not deleted. Nothing was restored." and "Finished, not complete"; after it `run/2` reads "Finished, not complete" (state `done`, HB-288 `verification.md:21`), "3 of 3 chunks finished (a stop finishes unclaimed chunks)", "Stopped by admin@example.invalid: unclaimed items were released, not deleted.", outcome Stopped 2, object store 3 = 1 + 0 + 2 + 0, no Stop button and the row note "This run has finished, so it cannot be stopped."; expiry `sample-temporary-2` reads "Accessible" and "On hold".
  - **Group B, as the second administrator** (`admin2@example.invalid`), with the first administrator only where stated:
    - #7 reads "You proposed this" with Withdraw; `source` reads "Your policy proposal #7…"; #6 shows Approve;
    - approve #6: the release confirm ("may be deleted by a later deletion run"); after it `sample-hold-temporary` reads Released, its heading reads "While in force, this hold could not preserve…" and still lists `sample-temporary-1` as not preserved; no Propose release, and the row note "This hold is released; there is nothing to release.";
    - withdraw #7 → withdrawn; `source` offers Propose;
    - propose `source` 400 / 30 (longer) → #8; as the first administrator approve #8: the confirm says no impact review is required and the request carries no impact fields; `source` reads revision 2, 400 + 30;
    - as the second administrator place a workspace hold, class Final documents, owner `member@example.invalid` → `retention-hold-1`, covers "This workspace · Final documents", the `no_eligible_approver` sentence;
    - no closed proposal offers any action; plan proposals #4, #5 read applied with the "not available here" sentence.
  - **Group D, the release flow through the real API** (M3; fresh API):
    - as the first administrator withdraw #6 → withdrawn; `sample-hold-temporary` then offers Propose release (the member placed it, so both administrators are eligible);
    - propose release with the release reason "Sample review finished again" → #8 pending; the request body (captured by route) carries `reason` in `body` and the rationale separately; the hold reads "Release proposal #8 is waiting for approval."; #8 reads "You proposed this" with Withdraw;
    - as the second administrator, #8 shows Approve; approve it → applied; `sample-hold-temporary` reads Released with the "While in force…" heading.
  - **Group C, failure paths** (first administrator unless stated):
    - **Revision conflict:** open Approve #7 with a reason in tab 1; in tab 2 stop `run/2`; confirm in tab 1 → conflict, reload, reason kept, the impact shown again without the "impact changed" sentence; confirm again → applied.
    - **Lost response after apply** (fetch the real response, then abort): place hold on `sample-source-6`; the pending sentence shows; the resend returns the stored `held` result for the same hold ID; holds total is exactly 3.
    - **Lost request before the server** (abort without fetching): propose a `claim` policy; the resend creates the proposal once.
    - **Stale key on place hold** (M4; stub `idempotency_conflict` by route on the resend of a hold whose first request was applied): the screen reloads, keeps "The last request may have been applied.", names the matching hold placed by you, and offers only "Place another hold", whose form shows the edit warning.
    - **Stubbed refusals by route:** `invalid_stage` on Withdraw #6 → focus on the row note with the kept withdrawal sentence; `invalid_request` [`owner`] on place hold → the owner field error via the summary and a re-read of the owner list; `invalid_request` [`impact_digest`] on approve → "could not be sent as read" and a reload, the reason field untouched; `internal_error` with an operation ID on propose policy (`final`, 3000 / 0) → the operation text and a reload. (No hold offers Propose release to the first administrator on the sample: the seeded holds read `already_proposed` and `no_eligible_approver`.)
    - **Read failures:** a stubbed `dependency_unavailable` on the retention read → the failure box with Try again, then the page; a stubbed failure of the access read → the owner list sentence and no send.
    - **Client checks:** a reason over 256 bytes; a whitespace-only reason; a hold reason of 201 bytes; a hold reason with a line break; period 0 and 36,501; grace 366.
    - **Focus:** status after a success; opener after Cancel.
    - **Drafts:** a hold reason typed into Place hold survives a tab change to policies and back.
    - **Member:** `member@example.invalid` sees the denied block.
    - **URL state:** invalid `view`, `class`, `artifact`, `hold`, `run` and `proposal` values are ignored; `view=holds&hold=sample-hold-dispute` and `view=deletion&run=2` are marked `aria-current`.
  - **Regressions**, each on a fresh API: P03 (`qa287run.sh` R, A, B, C); P02 (`qa285run.sh` A–D); P01 (qa283full); O03 (qa281); O02 (qa275); O01 (qa273); M01 (qa264).
  - Console errors during QA: none expected except the deliberate aborts.
- **Reviews and record:** ui-designer visual specification and render comparison; tech-writer terminology review in context; independent design and change review; `verification.md`.

# Residuals

- **Not connected in this build:** deletion plan proposal and approval, the deletion worker (claim, proceed, report, retry), backup expiry, archive, tiering and archive restore, usage, growth and byte counts, policy scope by jurisdiction or contract and archive rules, impact preview before proposing, the durable decision ledger, epoch, durable commit and restore freshness checks.
- **Holds:** with two administrators, a hold placed from the console can never be released (`no_eligible_approver`); only the seeded member-placed hold shows the full release flow (HB-288 `verification.md:42`). Live holds are capped at 100 per workspace, and the screen cannot pre-count them because the list is capped at 50 and includes released holds.
- **Installation-wide figures:** the watermark and the impact count and digest are installation-wide; sequence gaps reveal other-tenant deletions (HB-288 `plan.md:292-293`).
- **Permissions and audience:** every command uses installation administration; P04's separate policy, hold, deletion and evidence-access permissions and AU read access are later work.
- **Not built from the P04 spec:** filters, server search, stable sort and cursors; the selected-row drawer, evidence view and before/after comparison; links to S05, P03, R02, U03 and I04; the 100-million-row scale behaviour (`spec.md:69-75`, `89-99`, `121-125`).
- **Shell:** keeping the selection across shell tab changes and the unsaved-draft warning on a tab switch; the scoped `.planning-tabs` and `th` overrides; the global `bdi` rule (P04 avoids it for free text; P02 and P03 still have it).
- **Bounds:** at most 50 proposals, holds, outcomes per hold, references, runs, items per run, items per class and deletion records are read, with totals; closed proposals evicted beyond 200 no longer resolve.
- **Persistence:** the register, store and journal reset with the API process.
- **Reviews:** owner FIX of the screen; KO and AR terminology review of every AI-drafted string, in particular hold (KO 보존 보류, AR تعليق الحذف versus the glossary's إجراءات التعليق), deletion record, decision watermark, the class labels, "Due for deletion", "Recorded as deleted" and "Running (no worker connected)".
- **External references** were opened by root through WebFetch (text only, no screenshots); the ui-designer did not open them.
- **Single-workspace seed:** the rendering of `other_scope` run references, `shared` items, holds with `counts: null` (`counts_scope: other_scope`) and other-tenant tombstones is covered by model tests only; the sample seed has one workspace, so no QA render shows them.
- **Run state wording:** "Finished, not complete" replaces the coordinator's suggested "Finished (not every item was deleted)", because `done` also covers a run whose items were all deleted but whose backup expiry is not recorded (`runs.go:341-350`).

# Root decisions on the open questions

1. The hold owner list comes from `GET /identity/access` subjects, built by the same function `place_hold` checks; a failed, empty or missing list blocks sending.
2. The decisions tab keeps the neutral sentence that sequence numbers can be skipped; it adds nothing beyond what the numbers already show (HB-288 residual), and it prevents gaps from reading as lost records.
3. Proposals stay on their own tabs (policies, holds, deletion read-only).
4. KO and AR reuse the P03 hold terms (보존 보류 / تعليق الحذف); the glossary's payment-hold term is a different concept. Owner terminology review remains a residual.
5. The shell-scoped `.planning-tabs` override at 390px is accepted as a residual, as in P03.

# Design review round 2 clarifications (binding)

1. Group D: the first administrator (proposer) sees "Your release proposal #8 for this hold is waiting for another administrator's approval."; the second administrator sees the plain "Release proposal #8 is waiting for approval." form.
2. Group C: the steps run in a fixed order and the stale-key step targets a distinct item (`sample-source-7`); hold totals are asserted as the seeded count plus one per real hold placed so far.
3. Place-hold `partially_preserved` copy: "This hold cannot preserve {a} items already deleted or {c} partly deleted; {b} items have a deletion outcome pending." Pending items are never described as lost.
4. Residual: signing out clears the attempt store, so after signing in again the "may have been applied" warning is gone; the holds list is the place to check, and the place-hold form says so in its pending-attempt note.
5. The L1 state sentences for hidden-only blockers go in the row note's kept-sentence paragraph, not in `ul.p04-blocked`; the visual spec's "no line" applies to the blocked list only.
6. Runs follow the decision field-value rule (not a hard-coded sentence). Citations for run states are `modules/retention/runs.go:335-347`. Without withdrawing #6, no hold offers Propose release to the first administrator (see Group D).
