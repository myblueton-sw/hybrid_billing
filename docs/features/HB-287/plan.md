---
type: ImplementationPlan
title: Live P03 recovery screen in the console service mode, bound to the HB-286 recovery API
description: Phase 5f of the pre-launch plan (HB-257 scope). In service mode, P03 shows restore sets with domain verification and history, isolation with generations, stop times, writers, open calls and the lock log, external reconciliation with unknown targets, customer migration as not connected, and the resume decision with pending decision events, lost sequences and every recovery proposal. It offers the four HB-286 commands (approve an unlock, withdraw, lock, propose an unlock) only where the read allows them. Nothing is restored, pruned or deleted, and no host changes; the screen says so.
status: final
---

# Scope and ownership

[HB-287](https://linear.app/hybrid-billing/issue/HB-287) binds P03 to the [HB-286](../HB-286/plan.md) API ([verification](../HB-286/verification.md)). Base: main `411a758`. It follows the merged [HB-285 plan](../HB-285/plan.md), its [verification record](../HB-285/verification.md) and the code that plan produced (`apps/console/src/modules/administration/pages/integrations/`, `apps/console/src/service/integrations.ts`, the P02 branch in `apps/console/src/app/screen-groups/operations.tsx`). Every rule the HB-285 plan ended with after three design-review rounds is part of this design from the start. The P03 design is `docs/ui/screens/P03/spec.md`. It is in the authorized local workspace and not in this repository tree.

**Ownership**
- Root (Claude Code) owns the plan, integration, browser QA and verification. Implementation may be delegated to one writer in this worktree.
- The `ui-designer` role (AI, read-only) writes the visual specification and the external references, and compares the final renders.
- An independent read-only reviewer reviews the design and the change. Focus:
  - nothing implies a restore, a prune, a deletion, a host, network or credential change, or a running writer or verifier;
  - the single-administrator lock and the two-person unlock are shown truthfully;
  - a lock release is never presented as a checked recovery;
  - unknown values are never shown as zero;
  - commands are offered only where the read allows them, and parsing fails closed.
- Machine: mac_mini.

**Allowed files**
- `apps/console/src/service/recovery.ts` (types and hardened parsers) and `recovery.test.ts` (including the command envelope and resend identity).
  - It may import `count` and `hint` from `./integrations` read-only.
- `apps/console/src/service/admin.ts`: `recovery()` and the four command IDs (`recovery.approve`, `recovery.withdraw`, `recovery.lock`, `recovery.propose_unlock`).
- `apps/console/src/modules/administration/pages/recovery/`:
  - new `Live.tsx` (host), `LiveSets.tsx`, `LiveIsolation.tsx`, `LiveReconciliation.tsx`, `LiveMigration.tsx`, `LiveResume.tsx`, `LiveProposals.tsx` and `LiveCommand.tsx` (the shared command form);
  - `live-model.ts`, `live-model.test.ts`, `live-copy.ts`, `live-codes.ts` (vocabularies and blocker sentences), `live-format.tsx` and `live.css`.
  - Any file nearing 1,500 lines is split by responsibility.
  - The existing preview `Screen.tsx` stays for the unconnected mode.
- `apps/console/src/app/screen-groups/operations.tsx`: the live branch for P03.
- `apps/console/package.json`: the test list (`src/service/recovery.test.ts`, `src/modules/administration/pages/recovery/live-model.test.ts`).
- `docs/features/HB-287/*`.
- No API, module, P01, P02, M03 or M04 file changes.
  - P03 may import read-only helpers, for example `Known`, `fsi`, `ltr` and `stamp` from `integrations/live-format.tsx`, and `identity/pages/users/live.css`.
  - P03 keeps its own attempt store and command form, so P01, P02 and P03 attempts never mix.

# What the API offers (merged HB-286 code)

## Read: `GET /api/v1/recovery`

Route: `boundary.go:38`. The read is admin only, through `admitted` (`recovery.go:445`, `users.go:42`). It runs under the domain read lock (`recovery.go:443-444`).

**Whole-page states**
- **Another workspace, or no visible installation** (`--seed none`): `{revision, now, state: "not_configured"}` and nothing else (`recovery.go:450-453`). Installations are visible only in the boot workspace and only when their tenant equals it (`recovery.go:251-258`).
- **Any module read error:** `dependency_unavailable` for the whole read (`recovery.go:455-458`). The console maps it to `busy`.
- **Otherwise:** `{revision, now, state: "configured", sets, isolation, reconciliation, migration, resume, proposals}` (`recovery.go:202-212`, `414-417`).
  - Section states (`recovery.go:414-417`, `432`):
    - `sets`, `isolation` and `resume` are always `configured`;
    - `reconciliation` is `configured` only when at least one installation has an open recovery, otherwise `not_configured`;
    - `migration` is always `not_connected`.
- State, generation and stop time come from the module's `Status` (`recovery.go:264`, `recoveryseed.go:189-195`), never from a server copy.

**Availability** is a flat pair per action, `can_*` and `*_blocked`, where `can_*` is true exactly when the blocker is empty (`recovery.go:301`, `369`, `406`).

**`sets.rows`**, one per installation (`recovery.go:29-63`, `277-292`):
- `{installation, state, generation, recovery, history, history_total, begin: "not_connected", abandon: "not_connected"}`.
- `state` is `serving`, `locked` or `recovering` (`modules/recovery/types.go:34-36`).
- `recovery` is always an object (`recovery.go:278`, `281-282`):
  - `state: "none"` with every other field empty or zero and `domains: []` when no recovery is open;
  - `state: "open"` with `{id, phase, as_of, point_generation, backup_reference, manifest_digest, domains}`;
  - `phase` is `verify`, `reapply` or `ready` (`types.go:62-64`, `modules/recovery/recovery.go:508-515`);
  - each domain is `{domain, status, mismatch_chunks}`. The nine domains are `database`, `evidence`, `keys`, `policies`, `modules`, `numbering`, `submissions`, `outbox` and `deletions`, in module order (`types.go:48-56`). Status is `pending`, `verified` or `mismatch` (`types.go:58-60`). `mismatch_chunks` are zero-based chunk indices.
  - `backup_reference` and `manifest_digest` are 64-hex integrity values. They identify a backup; they are not credentials or contents (`HB-286 plan.md:195`).
- `history`: newest first, at most 50; `history_total` is the count the registry keeps (at most 100 per installation) (`recovery.go:287-290`). Each row is `{recovery, manifest_digest, generation, outcome, lost_count}`. `outcome` is `completed` or `abandoned` (`modules/recovery/recovery.go:100`, `107`).
- Chunk checksums, item counts and the restore-point manifest are never returned (`HB-286 plan.md:198`).

**`isolation.rows`**, one per installation (`recovery.go:65-96`, `294-320`):
- `{installation, state, generation, stopped_at, can_lock, lock_blocked, writers, verifier, open_calls_source, open_calls, open_calls_total, other_scope_present, permits: "not_shown", locks}`.
- `stopped_at` is empty while `serving` (the unlock clears it, `modules/recovery/recovery.go:110`).
- `lock_blocked` is the state itself when it is not `serving`: `locked` or `recovering` (`recovery.go:298-301`).
- `writers` are identifiers only. `verifier` is an identifier, or empty when none is registered (the standby has none, `recovery_api_test.go:162-166`).
- `open_calls_source` (`recovery.go:297`, `302-303`):
  - `recovery` while a recovery is open; then `open_calls` lists `{id, writer, deadline, key, series, number}` for calls in the reader's workspace, at most 50, with `open_calls_total`;
  - `not_shown` otherwise, with an empty list and a total of 0. **Open calls outside a recovery are not read**, so an empty list does not mean there are none.
- `other_scope_present` is a boolean only: calls on other workspaces exist; no count (`recovery.go:305-308`).
- `locks`: the lock log, `{by, by_login, rationale, at}`, in order (`recovery.go:316-318`).
- An open call past its deadline is closed as `unknown` on the next read (`reclaim`, `modules/recovery/recovery.go:479`, `calls.go:19-25`). It leaves `open_calls` **without a revision change**.

**`reconciliation.rows`**, one per installation **with an open recovery** (`recovery.go:98-120`, `322-348`, `426-429`):
- `{installation, externals, unknown_targets, unknown_total, other_scope_present, skipped, reconciler: "not_connected"}`.
- Externals: `{id, kind, status}`. `kind` is a writer kind: `erp`, `issuance`, `delivery`, `payment` or `message` (`types.go:38-42`). `status` is `pending` or `reconciled` (`types.go:71-72`).
- Unknown targets in the reader's workspace, at most 50, with the total (`recovery.go:329-346`). Each is `{key, series, number, fenced, high_water}`.
  - `fenced` comes from the module's `Fenced` (`recovery.go:339`).
  - `high_water` is a number only when `series` is set, otherwise `null` (`recovery.go:340-344`).
- `skipped` is a count.

**`migration`:** `{state: "not_connected", modules: ["migration", "balances"]}` (`recovery.go:27`, `416`).

**`resume.rows`**, one per installation (`recovery.go:129-139`, `350-371`):
- `{installation, mode, phase, pending_events, pending_total, lost, lost_digest, can_propose_unlock, unlock_blocked}`.
- `mode` (`recovery.go:18-20`, `351-362`):
  - `recovery_resume` while a recovery is open;
  - `lock_release` for a `locked` installation without one;
  - `none` otherwise (`serving`).
- `phase` is the recovery phase, or `not_recovering` without a recovery (`recovery.go:21`, `351`).
- Pending decision events: `{sequence, kind, domain, at}`, at most 50, with `pending_total`; no key (`recovery.go:355-359`). Kinds: `delete`, `purge`, `hold`, `extend`, `revoke_key`, `revoke_access` (`types.go:74-79`).
- `lost` is the list of lost sequences (unbounded; empty without a recovery). `lost_digest` is `recovery.LostDigest` of that list (`recovery.go:363`). It is the value `propose_unlock` must send.
- Unlock blockers, in order (`recovery.go:364-369`):
  1. `already_proposed`: a pending unlock entry for the installation exists, from anyone (`recoveryseed.go:105-112`);
  2. otherwise the module's dry run (`Check`) of the unlock the reader would propose, at the `Status` generation with the shown digest. `checkReason` maps it (`recovery.go:230-245`):
     - refusal reasons from `checkUnlock` (`modules/recovery/recovery.go:434-470`) and the generation check: `not_recovering`, `open_calls`, `no_restore_point`, `incomplete`, `lost`, `kind_coverage`, `generation`;
     - module errors: `not_found`, `limit`, `conflict`, `invalid`.

**`proposals`**, newest first, at most 50, only for installations visible to the reader (`recovery.go:433-437`). Each row (`recovery.go:155-175`, `400-408`):
- `{number, target: "recovery-proposal-<n>", kind, installation, state, created_at, proposer, proposer_login, approver, approver_login, rationale, closed_by, closed_by_login, closing_rationale, summary, can_approve, approve_blocked, can_withdraw, withdraw_blocked}`.
- `kind` is one of `installation`, `writer`, `external`, `verifier`, `begin`, `unlock`, `prune` and `abandon` (`types.go:21-28`).
- `state` is `pending`, `applied`, `withdrawn` (`types.go:30-32`) or the server's `retired` (`recoveryseed.go:21`, `131-136`).
- `summary` (`recovery.go:141-153`, `373-377`): `{installation, skew_seconds, writer, external, external_kind, verifier, as_of, point_generation, manifest_digest, unlock_generation, lost_digest}`. Only the fields of the row's kind are filled; the others are empty or 0. A prune or abandon summary carries only the installation.
- **Blockers** (`recovery.go:379-398`):
  - non-pending: `not_pending` for both actions, or `retired` for both on a retired entry; no module call;
  - approve on a pending row: `not_console_kind` for any kind other than `unlock`; then `own_proposal`; then the module's `Check` of the stored change (the reasons above);
  - withdraw on a pending row: `not_proposer` unless the reader proposed it. Withdraw stays available for any kind to its proposer.

No row carries a chunk checksum, a permit token, a decision-event key, an item list or a target from another workspace (`HB-286 plan.md:228`, `HB-286 verification.md:361`).

## Commands

All four are `recovery.<kind>` (`recoverycmds.go:13-21`, `178-190`):
- Each takes a reason, and the `expected_revision` is the read's `revision`.
- Each needs `AdministerInstallation` on the installation kind (`recoverycmds.go:182-183`). This is a recorded deviation from P03's five separate permissions (`HB-286 plan.md:239`).
- The handler refuses a reason over 256 bytes with a fieldless `invalid_request` (`recoverycmds.go:156-157`, `directory.go:13`). The command layer allows 2,000 bytes, so the screen enforces 256 itself.
- Every module actor is the reader as a human (`recoverycmds.go:159`).

| Command | Target | Fields | Result `business_state` / `fields` | Command-specific refusals |
| --- | --- | --- | --- | --- |
| `recovery.approve` | `recovery-proposal-<n>` | — | `applied`; `proposal`, `target` (`:92-95`, `:105`) | not pending (incl. `withdrawn`, `retired`, and a repeat by its own approver) or not `unlock` (`not_console_kind`) → `invalid_stage`, decided from the stored view without a module call (`:99-100`); self-approval → `prerequisite_failed`; any `Check` refusal (`generation`, `open_calls`, `incomplete`, …) → `invalid_stage`; a proposal the module deleted → `invalid_stage` and the entry becomes `retired` (`recoveryseed.go:131-136`, `157-162`); limit → `prerequisite_failed` |
| `recovery.withdraw` | `recovery-proposal-<n>` | — | `withdrawn`; `proposal`, `target` (`:108-117`) | not pending → `invalid_stage` (`:110-111`); not the proposer → `prerequisite_failed`; deleted in the module → `invalid_stage` and `retired` |
| `recovery.lock` | `recovery-installation:<id>` | — | `locked`; `installation`, `target` (`:131`) | `Status` not `serving` (`locked` would be a silent no-op in the module, `recovering` a refusal) → `invalid_stage` (`:121-127`) |
| `recovery.propose_unlock` | `recovery-installation:<id>` | `lost` (required text) (`:187`) | `pending`; `proposal`, `target` (`:146-151`) | checked in this order (`:137-145`): `serving`, or any pending unlock for the installation from anyone (`already_proposed`) → `invalid_stage`; `lost` not 64 lowercase hex, or a `locked` installation with `lost` other than `LostDigest(nil)` → `invalid_request` [`lost`]; then the module: refusals → `invalid_stage`, limit → `prerequisite_failed` |

Line references in the table are to `recoverycmds.go` unless named. The state check comes before the digest check: a `serving` installation answers `invalid_stage` whatever `lost` is (`HB-286 plan.md:335`, `recovery_commands_test.go:52`).

**Lock reason.** The module validates but does not store a lock reason. The server passes the fixed text `Administrator lock`, and keeps the administrator's reason in the lock log (`recoveryseed.go:20`, `181-187`).

**Error mapping and fields**
- **Mapping** (`recoverycmds.go:23-42`):
  - `self_approval`, `not_proposer` and `ErrLimit` → `prerequisite_failed`;
  - `ai_not_allowed`, `not_human`, `not_writer`, `not_verifier` and `ErrInvalid` → `internal_error` (unreachable: the server builds only human actors);
  - every other refusal, `ErrNotFound` and `ErrConflict` → `invalid_stage`.
- **Codes that carry fields.**
  - Only `invalid_request` carries fields, and from a handler only declared names: `lost` for `propose_unlock` (`modules/commands/execute.go:399-413`, `recoverycmds.go:144`).
  - Shape validation may also name `rationale`, `target_id`, `expected_revision`, `idempotency_key`, `request_id`, `client`, an undeclared field name, or the marker for excess fields (`execute.go:53-107`, `282-284`).
  - Every other code is fieldless (`HB-286 plan.md:272`), so the read's blockers explain refusals.
- **Revision.** A successful command moves the revision (`recoverycmds.go:175`). An outdated `expected_revision` is refused with `revision_conflict` before `Apply` (`execute.go:356-358`).
- **`internal_error` is journaled** (`execute.go:420-425`). The first one moves the revision like any operation, and a resend with the same key replays it with the same operation ID and no further move (`recovery_commands_test.go:310-323`).
- **Console failure classes:** `invalid` (invalid_request), `refused` (prerequisite_failed), `stage` (invalid_stage), `denied` (permission_denied), `conflict` (revision_conflict), `busy` (dependency_unavailable), `limited`, `stale` (idempotency), `unauthenticated`. `internal_error` becomes `partial` only with an operation ID, otherwise `unavailable` (`admin.ts:305-329`).
- **Resolve and unknown targets.** `Resolve` reads only the server store, without an actor (`recoverycmds.go:75-85`):
  - a proposal number in canonical decimal whose installation's tenant equals the workspace (`recoverycmds.go:44-55`);
  - an installation ID whose tenant equals the workspace, in the boot workspace (`recoverycmds.go:57-64`).
  - Anything else is `permission_denied` (`execute.go:330-332`), including non-canonical numbers, a `recovery-writer:` target and another tenant (`recovery_api_test.go:256-265`).

**Replay: every target stays resolvable.**
- Installations are never removed from the store (`HB-286 plan.md:187`). Proposal entries are kept, and only closed entries leave, once more than 200 are kept (`recoveryseed.go:77-88`, `installseed.go:18`).
- So a resend with the same idempotency key and body after a lost response resolves. The journal lookup comes before the revision check (`execute.go:342-358`), so it returns the stored result even after the state and the revision moved (`recovery_commands_test.go:282-309`, all four commands).
- The only non-replayable case is a closed proposal evicted beyond 200 entries. Its approve or withdraw resend is `permission_denied`. This cannot happen with the sample seed (ten proposals).
- **Repeats are refused, not replayed.** A fresh approval of a proposal that is no longer pending, a second lock and a second unlock proposal are refused as `invalid_stage` without moving the revision (`recovery_commands_test.go:51-54`, `61`, `69-70`).

## Sample seed (who did what)

`--seed sample`, one counted step `recovery` (`seed.go:108`, `recoveryseed.go:210-290`). First administrator `sample-admin` = `admin@example.invalid`; second `sample-admin-2` = `admin2@example.invalid`; member `member@example.invalid` (`seed.go:22-27`). Seed reasons read "Sample data" (`installseed.go:20`).

**Proposals #1–#9: proposed by the first administrator and approved by the second, all `applied`** (`recoveryseed.go:216-224`, `recovery_api_test.go:184-187`):
- #1 installation `sample-primary` (skew 60 s);
- #2 writer `sample-erp-writer` (`erp`, 1 minute);
- #3 writer `sample-issuance-writer` (`issuance`, 2 minutes);
- #4 verifier `sample-verifier`;
- #5 external `sample-erp-ledger` (`erp`); #6 external `sample-issuance-register` (`issuance`);
- #7 `begin` of `sample-primary`: restore point as of the seed time minus 1 hour, generation 1;
- #8 installation `sample-standby`; #9 writer `sample-standby-writer` (`erp`).

**#10, pending:** unlock of `sample-standby` at generation 2 with `LostDigest(nil)`, **proposed by the second administrator** (`recoveryseed.go:283-286`).
- The first administrator sees Approve and no Withdraw (`not_proposer`).
- The second sees `own_proposal` and Withdraw (`recovery_api_test.go:188-193`, `217-220`).

**`sample-primary`** is `recovering` at generation 2, recovery `recovery-1`, phase `verify` (`recovery_api_test.go:136-152`):
- eight domains `verified`; `outbox` `mismatch` at chunk index 1;
- both externals `pending`; one unknown target `sample-invoice-2` (series `sample-invoice-series`, number 2), fenced, high water 2;
- two pending decision events: 1 `delete` on `evidence`, 2 `revoke_access` on `database`; no lost sequences; empty history;
- resume `recovery_resume`, blocker `incomplete`; lock blocker `recovering`;
- open call `call-2` by `sample-erp-writer`, **deadline = seed time + 1 minute** (`recovery_api_test.go:157-161`). In a real-clock run, the call leaves `open_calls` on the first read after that deadline; the target stays unknown and fenced in reconciliation.

**`sample-standby`** is `locked` at generation 2, stopped at the seed time (`recovery_api_test.go:162-166`):
- the lock log has one row, by the first administrator, "Sample data";
- writer `sample-standby-writer`; no verifier; `open_calls_source: not_shown`;
- resume `lock_release`, blocker `already_proposed` (#10); lock blocker `locked`; no reconciliation row.

**After the first administrator approves #10** (`recovery_commands_test.go:33-50`):
- `sample-standby` is `serving` at generation 2 with an empty stop time; resume `none` with the blocker `not_recovering`; `can_lock` true.
- A lock moves it to `locked` at generation 3 and appends a lock-log row.
- A new unlock proposal (number 11) then needs `lost` = `LostDigest(nil)` and another administrator's approval.

# Design

The design follows the P02, P01, O01–O03 and M01 live patterns:
- full viewport width; neutral rules and tags, no coloured edge strips;
- EN, KO and AR copy;
- the shared failure box and the reason field (UTF-8 bytes, 256), with an error summary;
- `useWide` at 1280px;
- dates through `Intl` in UTC with first-strong isolation;
- IDs, digests and call targets in `<bdi dir=ltr>` monospace.

Classes use the `p03-` prefix.

## Host, remount and URL state

- **Live branch.** `operations.tsx` sets `live = !!service && (screenId === "P01" || screenId === "P02" || screenId === "P03")`. For P03 it returns, before the preview wrapper:
  - `<div className="work-domain remaining-security" data-screen-id="P03" data-source-state="live" dir lang key={`P03:${locale}`}>` around `LiveRecovery`;
  - with props `locale`, `view`, `params`, `session` and `onSignOut`, as for P02.
  - The key is locale only. Hooks stay unconditional, and the live branch never reports `source: unconnected` (the existing `if (live) return;` covers it).
- **Remount per view (rule).** The parent `RemainingScreens` keys `RevisionScope` by screen, locale, view and scope parameters (`RemainingScreens.tsx:82-93`), so every tab change remounts `LiveRecovery` and reloads the read, whatever the host key.
  - Component state (the page, notices, open forms) does not survive a tab change. Only the module-level store (attempts, drafts, kept refusals) survives.
  - Announcements are shown within one view. A cross-view link carries its target in the URL, never in component state.
- **URL state:**
  - `view=` is one of `sets`, `isolation`, `reconciliation`, `migration` or `resume`; anything else falls back to `sets`.
  - Optional selections, each validated against the read and otherwise ignored:
    - `installation=<id>` (sets, isolation, reconciliation and resume). The ID matches `[A-Za-z0-9._:-]{1,128}` and is present in that view's section rows.
    - `proposal=<n>` (resume). Canonical decimal, present in `proposals`.
  - The selection is derived from the URL parameters on every render, never stored in state. Scrolling into view happens once per distinct parameter value (the scroll-once flag is keyed on the parameter), and a selection change does not reload the read. The selected row is marked `aria-current`.
  - The P03 spec asks that a tab change keep the selection (spec:82). The tab bar belongs to the shell, which is outside the allowed files, so this ticket does not change it. The screen's own cross-view links carry `installation=`, and every view ignores an `installation=` without a row there. If the shell's tabs drop the parameter, that is recorded as a residual. The spec's unsaved-draft warning on a tab switch is a residual too; reasons survive a tab change in the draft store instead.
  - Each restore-set row links to its installation on isolation, reconciliation (only when a reconciliation row exists) and resume (`view=<v>&installation=<id>`).
  - A resume row whose blocker is `already_proposed` links to that proposal (`view=resume&proposal=<n>`).
  - A pending unlock proposal links to its installation's resume row.
  - Only canonical IDs are passed.
- **Focus and live region:**
  - After a successful command, a status message (`#p03-notice`, `role=status`) is focused.
  - A refusal focuses the row note.
  - Cancel returns focus to the opener.
  - Error-summary links move focus to the field.
  - The reload sets `aria-busy` on the screen.
- **Stale responses.** A request sequence (`seq` ref) drops stale read responses. A command form ignores a second send while busy.
- **Store.** The attempt and draft store (module level, keyed by command and target) is scoped by workspace, user and session (CSRF handle), as `setScope` in P01 and P02. It is cleared by `forget()` on sign-out and on a scope change.

## Shared frame (every view)

- **Scope bar:** workspace, data revision, now (UTC), Reload.
- **Statement:**
  - "**Records decisions only.** Nothing is restored, pruned or deleted, and no host, network or credential is changed from this screen in this build."
  - KO: "**결정만 기록합니다.** 이 빌드에서는 이 화면에서 아무것도 복원·정리·삭제하지 않으며 호스트·네트워크·자격 증명도 변경하지 않습니다."
  - AR: "**تسجيل القرارات فقط.** لا تتم من هذه الشاشة في هذا الإصدار أي استعادة أو تقليم أو حذف، ولا يُغيَّر أي مضيف أو شبكة أو بيانات اعتماد."
- **Permission note** (small, under the statement):
  - "Every command here currently needs installation administration. Separate recovery verification, recovery execution, customer export, ERP reconciliation and resume permissions are not available yet, and auditors and billing operators cannot open this screen yet."
  - KO: "현재 이 화면의 모든 명령에는 설치 관리 권한이 필요합니다. 복구 검증·복구 실행·고객 범위 반출·ERP 대사·재개 권한은 아직 별도로 제공되지 않으며, 감사 조회자와 청구 운영자는 아직 이 화면을 열 수 없습니다."
  - AR: "تتطلب كل الأوامر هنا حاليًا صلاحية إدارة التثبيت. صلاحيات التحقق من الاسترداد وتنفيذه وتصدير نطاق العملاء ومطابقة ERP والاستئناف غير متاحة بشكل منفصل بعد، ولا يستطيع المدققون ومشغلو الفوترة فتح هذه الشاشة بعد."
- **Whole-page states:**
  - first load: "Loading recovery state…" with no numbers;
  - reload: `aria-busy`;
  - `not_configured`: "Recovery is not configured in this workspace";
  - an unknown `state`: "The recovery read returned an unknown state (code)";
  - denied: "Recovery is not available to you" / "You do not have permission to manage recovery in this installation.";
  - unauthenticated: "Your session has ended." with "Sign in again" (calls `forget()`);
  - failure (`busy`, `unavailable` including `authority_unavailable`, `limited` with the wait) with Try again.
- **Section states.** A section with any state other than `configured` shows a sentence instead of its tables:
  - reconciliation `not_configured`: "No recovery is open, so there is nothing to reconcile.";
  - migration `not_connected`: its view (see below);
  - an unknown section state: "Unknown (code)".
- **Read-only.** When `revision` is empty, commands are disabled and the view reads "Commands are unavailable because the data revision is unknown. Reload."

## Views (the registered P03 tab keys)

### 1. Restore sets (`sets`)

- **Lead:**
  - "Nothing is restored here. Restore sets are read from the recovery registry. Registering a restore set, requesting an isolated restore, abandoning a recovery and pruning decision events are not connected in this build."
  - KO: "여기에서는 아무것도 복원하지 않습니다. 복원 세트는 복구 레지스트리에서 읽습니다. 이 빌드에서는 복원 세트 등록, 격리 복원 요청, 복구 중단, 결정 기록 정리가 연결되어 있지 않습니다."
  - AR: "لا تتم أي استعادة هنا. تُقرأ مجموعات الاستعادة من سجل الاسترداد. تسجيل مجموعة استعادة وطلب استعادة معزولة والتخلي عن الاسترداد وتقليم أحداث القرار غير متصلة في هذا الإصدار."
- **Restore sets table**, one row per installation, in the spec column order (restore ID → as-of → generation → evidence check → decision watermark → external reconciliation → public lock), after the installation ID:
  - **Installation:** the ID, with links to isolation, reconciliation and resume.
  - **Restore ID:** the recovery ID. With `recovery.state` `none`: "No recovery open".
  - **As of:** the restore point time. Not shown without a recovery.
  - **Generation:** "Installation {generation} · restore point {point_generation}". Without a recovery, only the installation generation.
  - **An open recovery without a restore point** (empty `as_of`, `manifest_digest` and domain list; reachable only after an abandon) reads "No restore point" for the as-of, the restore-point generation, the evidence check and the details, and its reconciliation cell reads "No restore point" too. Its zero `point_generation` is never shown.
  - **Evidence check:** "{v} of {n} domains verified", plus "{m} mismatched" and "{p} not checked yet" when non-zero. Counts come from the domain list; an empty domain list on an open recovery reads "No restore point".
  - **Decision watermark:** from the resume row of the same installation:
    - "{pending_total} decision events not yet reapplied";
    - "No decision events pending" when the total is a real 0 and a recovery is open. It never claims that events were reapplied;
    - with a recovery open, "{n} declared lost" is added from the resume `lost` list when it is non-empty ("Unknown" when the list is missing);
    - "Not checked outside a recovery" when the mode is `lock_release` or `none`;
    - "Unknown" when the resume row is missing.
  - **External reconciliation:** from the reconciliation row: "{r} of {n} external systems reconciled", or "Not applicable (no recovery open)" when there is no row.
  - **Public lock:** the installation state in words. `recovering` and `locked` read "Writes locked (in recovery)" and "Writes locked"; `serving` reads "Not locked".
- **Details disclosure per open recovery:**
  - phase in words; backup reference and manifest digest (64 hex, wrapping, ltr);
  - the domain table: domain in words, status in words, and for a mismatch:
    - "Chunk index {i} does not match" (indices counted from 0, as the registry reports them; several are joined with commas inside one ltr isolate);
    - an empty `mismatch_chunks` list: "Mismatch; no chunk index was reported (the item count differs).";
    - exactly 100 indices (the module's bound on reported indices): the list is prefixed "The first 100 mismatched chunk indices".
  - "Chunk checksums and item lists are not shown."
- **History** (per installation): recovery, manifest digest, generation, outcome in words, lost-sequence count (a real 0 reads "0").
  - `history_total` is the count the registry keeps (at most 100 per installation), not every recovery ever run. "Showing the newest 50 of {total} kept." when it exceeds the rows. "No completed or abandoned recoveries are kept." when empty.
- `begin` and `abandon` `not_connected` read "Not connected"; another value reads "Unknown (code)".

### 2. Isolation check (`isolation`)

- **Lead:**
  - "Locking records the installation as locked in the recovery registry and moves its generation, so the registry refuses write permits and calls under the old generation. It does not stop a host, cut a network or revoke a credential. No writer is connected to the registry in this build."
  - KO: "잠그면 복구 레지스트리에 설치가 잠김으로 기록되고 generation이 올라가, 레지스트리가 이전 generation의 쓰기 허가와 호출을 거부합니다. 호스트를 중지하거나 네트워크를 끊거나 자격 증명을 회수하지는 않습니다. 이 빌드에서는 레지스트리에 연결된 쓰기 주체가 없습니다."
  - AR: "يسجّل القفل التثبيت كمقفل في سجل الاسترداد ويرفع رقم الجيل، فيرفض السجل أذونات الكتابة والاستدعاءات تحت الجيل السابق. لا يوقف أي مضيف ولا يقطع أي شبكة ولا يلغي أي بيانات اعتماد. لا توجد جهة كتابة متصلة بالسجل في هذا الإصدار."
- **One block per installation:**
  - state in words, generation, stopped at:
    - a time; or "Not stopped" when empty and `serving`;
    - "Unknown" when empty and `locked` or `recovering`.
  - **Writers:** identifiers, or "No writers registered". "Removing a writer is not available here."
  - **Verifier:** the identifier, or "No verifier registered" when empty.
  - **Write permits:** `not_shown` reads "Not shown. Write permits are never shown on this screen."; another value reads "Unknown (code)".
  - **Open calls:**
    - source `recovery`: table of call ID, writer, deadline, and the target as one ltr isolate `{key} · {series} #{number}` (`{key}` alone when the series is empty). "No open calls in this workspace." when the list is empty and the total is a real 0. "Showing 50 of {total}." when the total exceeds the rows.
    - source `not_shown`: "Open calls are listed only while a recovery is open." It never reads "None".
    - an unknown source: "Unknown (code)".
    - `other_scope_present` true adds "Calls in other workspaces also exist. They are not shown." (no count). `false` reads nothing; a missing flag reads "Calls in other workspaces: Unknown".
    - "A call past its deadline is closed as unknown the next time the registry is read. This does not change the data revision."
  - **Lock log:** time, administrator (login, falling back to ID), reason. "No locks recorded." when empty.
  - **Lock** is shown only when `can_lock` is true and the target is well-formed (see Command availability).
    - Lock confirm (restates the installation, state and generation): "**Takes effect now; no second approval: you alone lock this installation.** Locking records {installation} as locked and moves its generation from {g} to {g+1}, so the recovery registry refuses write permits and calls under generation {g}. It does not stop a host, cut a network or revoke a credential. Releasing the lock later needs an unlock proposal and another administrator's approval."
    - When the generation is unknown, the sentence omits "from {g} to {g+1}" and "under generation {g}".
    - When the proposals list holds a pending `begin` row for the installation, the confirm adds: "Pending recovery-opening proposal #{n} will be retired by this lock and can no longer be approved or withdrawn." (The generation bump deletes pending `begin` proposals, `modules/recovery/changes.go:269-272`, `registry.go:311-324`.) The seed has none; it is covered by a model test.
    - KO: "**지금 바로 적용되며 두 번째 승인이 없습니다. 이 설치는 귀하 혼자 잠급니다.** 잠그면 {installation}이(가) 잠김으로 기록되고 generation이 {g}에서 {g+1}(으)로 올라가, 복구 레지스트리가 generation {g}의 쓰기 허가와 호출을 거부합니다. 호스트를 중지하거나 네트워크를 끊거나 자격 증명을 회수하지는 않습니다. 나중에 잠금을 해제하려면 잠금 해제 제안과 다른 관리자의 승인이 필요합니다."
    - AR: "**يسري الآن دون موافقة ثانية: أنت وحدك تقفل هذا التثبيت.** يسجّل القفل {installation} كمقفل ويرفع جيله من {g} إلى {g+1}، فيرفض سجل الاسترداد أذونات الكتابة والاستدعاءات تحت الجيل {g}. لا يوقف أي مضيف ولا يقطع أي شبكة ولا يلغي أي بيانات اعتماد. يتطلب رفع القفل لاحقًا اقتراح إلغاء قفل وموافقة مسؤول آخر."

### 3. External reconciliation (`reconciliation`)

- **Lead:**
  - "External results are not checked from this screen. The reconciler is not connected in this build, so unknown targets stay unknown and fenced, and nothing is resubmitted."
  - KO: "이 화면에서는 외부 결과를 확인하지 않습니다. 이 빌드에서는 대사 기능이 연결되어 있지 않아 미확인 대상은 미확인·차단 상태로 남고 아무것도 다시 제출하지 않습니다."
  - AR: "لا يتم التحقق من النتائج الخارجية من هذه الشاشة. أداة المطابقة غير متصلة في هذا الإصدار، لذلك تبقى الأهداف غير المعروفة غير معروفة ومحجوبة، ولا يُعاد إرسال أي شيء."
- Section `not_configured`: "No recovery is open, so there is nothing to reconcile."
- **One block per row** (installations with an open recovery):
  - **External systems:** ID, kind in words, status in words. "No external systems registered." when empty.
  - **Unknown targets:** key, series ("No series" when empty), number, fenced ("Fenced" / "Not fenced"), high water:
    - a number when the series is set;
    - "No series" when the series is empty and the value is `null`;
    - "Unknown" when the series is set and the value is not a number.
    - "Showing 50 of {total}." when the total exceeds the rows. "No unknown targets in this workspace." when empty with a real 0.
    - `other_scope_present` as in isolation.
  - **Skipped:** "{n} skipped" (a real 0 reads "0").
  - An open recovery without a restore point (see Restore sets) returns no externals; its block reads "No restore point" instead of "No external systems registered".
  - **Reconciler:** `not_connected` reads "Not connected"; another value "Unknown (code)".
- "An unknown target is resolved only by the reconciler from the original submission. This screen records no reconciliation."

### 4. Customer migration (`migration`)

- **`not_connected`:**
  - "Customer migration is not connected in this build. No migration package is prepared, no legacy document is imported, no opening balance is recorded and no customer data leaves the platform."
  - KO: "이 빌드에서는 고객 이관이 연결되어 있지 않습니다. 이관 패키지를 준비하지 않고, 이전 문서를 반입하지 않으며, 기초 잔액을 기록하지 않고, 고객 데이터가 플랫폼 밖으로 나가지 않습니다."
  - AR: "ترحيل العملاء غير متصل في هذا الإصدار. لا تُجهَّز أي حزمة ترحيل، ولا يُستورد أي مستند سابق، ولا يُسجَّل أي رصيد افتتاحي، ولا تغادر أي بيانات عملاء المنصة."
- **Modules to connect later:** `modules` in words ("Customer migration import", "Opening balances"); an unknown entry reads "Unknown (code)"; a missing list reads "Unknown".
- An unknown section state reads "Unknown (code)". No command is offered on this view.

### 5. Resume decision (`resume`)

- **Lead:**
  - "Approving an unlock records the decision in the recovery registry and returns the installation to serving. Nothing is restored, pruned or deleted, and no host changes. A lock release checks no decision watermark: it checks only the lock, the generation and open calls."
  - KO: "잠금 해제를 승인하면 복구 레지스트리에 결정이 기록되고 설치가 운영 상태로 돌아갑니다. 아무것도 복원·정리·삭제하지 않으며 호스트도 변경하지 않습니다. 잠금 해제는 결정 워터마크를 확인하지 않고 잠금·generation·열린 호출만 확인합니다."
  - AR: "تسجّل الموافقة على إلغاء القفل القرار في سجل الاسترداد وتعيد التثبيت إلى التشغيل. لا تتم أي استعادة أو تقليم أو حذف، ولا يتغير أي مضيف. رفع القفل لا يتحقق من علامة القرار المائية: يتحقق فقط من القفل والجيل والاستدعاءات المفتوحة."
- **One block per installation:**
  - mode in words, phase in words.
  - `mode` `none` adds "This installation is serving; there is nothing to resume or release."
  - `mode` `lock_release` adds "This installation is locked without a recovery. Releasing the lock does not verify any restore, decision watermark or external result."
  - **Pending decision events** (only with a recovery): sequence, kind in words, domain in words, time. "Decision event keys are not shown." "Showing 50 of {total}." when needed. "No pending decision events." when empty with a real 0.
  - **Lost sequences:** "None", or the numbers in one ltr isolate that wraps, separated by commas; after 50 numbers it ends with "and {n} more". A missing list reads "Unknown". The lost-sequence digest is shown (64 hex, wrapping) as "Confirmation digest".
  - **Propose unlock** is shown only when `can_propose_unlock` is true and the derived checks pass (see Command availability). Otherwise its blocker is handled by the display policy.
    - The confirm restates the installation, mode, state, generation, the lost sequences ("None" in `lock_release`) and the digest.
    - Impact, by mode:
      - `lock_release`: "Another administrator must approve it. Approval releases the lock and returns {installation} to serving at generation {g}. (When the generation is unknown, "at generation {g}" is omitted, here and in every confirm.) A lock release checks only the lock, the generation and open calls; it checks no decision watermark, domain verification or external reconciliation. Nothing is restored, pruned or deleted, and no host changes."
      - `recovery_resume`: "Another administrator must approve it. Approval confirms the listed lost sequences, records recovery {id} as completed and returns {installation} to serving. Nothing is restored, pruned or deleted, and no host changes."
- **Recovery proposals** (below the installations):
  - Pending first, then closed, newest first. Each row:
    - number, kind in words, a summary in words, state;
    - proposer and created time, and the rationale on pending rows;
    - `applied` rows show the approver (`approver_login`, falling back to `approver`) with the closing reason; `withdrawn` rows show the withdrawer (`closed_by_login`, falling back to `closed_by`) with the closing reason;
    - a `retired` row shows neither, reads "Retired: the recovery registry no longer holds this proposal" and offers nothing.
  - A note under the list: "Applied means recorded in the recovery registry." (KO "적용됨은 복구 레지스트리에 기록되었다는 뜻입니다.", AR "«مطبّق» تعني أنه سُجّل في سجل الاسترداد.")
  - "Showing the newest 50." when 50 rows return. "No recovery proposals." when empty.
  - Summaries per kind (fallback "Unknown (code)"):

| Kind | EN | KO | AR |
| --- | --- | --- | --- |
| installation | Register installation {installation} (clock skew {skew}) | 설치 {installation} 등록(시계 오차 {skew}) | تسجيل التثبيت {installation} (انحراف الساعة {skew}) |
| writer | Register writer {writer} for {installation} | {installation}에 쓰기 주체 {writer} 등록 | تسجيل جهة الكتابة {writer} لـ {installation} |
| external | Register external system {external} ({kind}) for {installation} | {installation}에 외부 시스템 {external}({kind}) 등록 | تسجيل النظام الخارجي {external} ({kind}) لـ {installation} |
| verifier | Register verifier {verifier} for {installation} | {installation}에 검증 주체 {verifier} 등록 | تسجيل جهة التحقق {verifier} لـ {installation} |
| begin | Open recovery of {installation} against the restore point as of {as_of} (generation {point_generation}, manifest {manifest_digest}); recorded in the registry, no data was restored | {as_of} 기준 복원 지점(generation {point_generation}, manifest {manifest_digest})에 대해 {installation} 복구 개시. 레지스트리에 기록만 되었으며 데이터는 복원되지 않음 | فتح استرداد {installation} مقابل نقطة الاستعادة بتاريخ {as_of} (الجيل {point_generation}، البيان {manifest_digest})؛ سُجّل في السجل ولم تُستعد أي بيانات |
| unlock | Unlock {installation} at generation {unlock_generation}, confirming lost-sequence digest {lost_digest} | {installation}을(를) generation {unlock_generation}에서 잠금 해제, 유실 순번 요약값 {lost_digest} 확인 | إلغاء قفل {installation} عند الجيل {unlock_generation}، مع تأكيد ملخص التسلسلات المفقودة {lost_digest} |
| prune | Prune decision events of {installation} | {installation}의 결정 기록 정리 | تقليم أحداث القرار لـ {installation} |
| abandon | Abandon the recovery of {installation} | {installation}의 복구 중단 | التخلي عن استرداد {installation} |

  - Every placeholder is isolated: IDs, generations and digests in ltr monospace, dates with FSI. `{skew}` reads "{n} s"; `null` reads "Unknown". `{kind}` uses the writer-kind vocabulary. An empty value reads "Unknown".
  - **Approve** is shown only when `can_approve` (in this build only on pending `unlock` rows of another administrator), and, as a defensive narrowing (the server stays authoritative), only when `summary.unlock_generation` equals the installation's current generation and `summary.lost_digest` equals its resume `lost_digest`. Otherwise the row reads "The installation changed after this was proposed. Reload or ask the proposer to withdraw it." The approve confirm restates the summary, the proposer and their rationale, and the installation's current mode, state and generation from its resume and isolation rows. Impact by that mode:
    - `lock_release`: "Approval releases the lock and returns {installation} to serving at generation {g}. A lock release checks only the lock, the generation and open calls; it checks no decision watermark, domain verification or external reconciliation. Nothing is restored, pruned or deleted, and no host changes."
    - `recovery_resume`: "Approval confirms the listed lost sequences, records recovery {id} as completed and returns {installation} to serving. Nothing is restored, pruned or deleted, and no host changes."
    - any other mode, or no resume row: Approve is not offered, and the row reads "The installation's resume state is unknown, so this cannot be approved here."
  - **Withdraw** is shown only when `can_withdraw`. Confirm: "Withdrawing closes this proposal. Nothing else changes."

## Command availability (fail closed)

- **Parsing.** Every pair (`can_lock`/`lock_blocked`, `can_propose_unlock`/`unlock_blocked`, `can_approve`/`approve_blocked`, `can_withdraw`/`withdraw_blocked`) uses P02's `hint` (`integrations.ts:184-188`):
  - allowed only when `can_*` is `true` and the blocker is exactly `""`;
  - `can_*` true with a non-empty blocker is not allowed;
  - a non-string blocker, or an empty blocker with `can_*` not true, is `missing`.
- **Targets.** Commands use a target only when it matches its pattern and its ID. Otherwise no command is offered and the row reads "No command is available for this row."
  - proposals: the row's own `target`, `^recovery-proposal-[1-9][0-9]*$` and equal to `recovery-proposal-` + `number`;
  - installations (built by the screen): `recovery-installation:` + the row's `installation`.
  - Every target must be 1–128 bytes of `[A-Za-z0-9._:-]` (`modules/commands/types.go:299-310`). An installation ID longer than 106 bytes therefore offers no command.
- **Read blockers** gate all four commands. The screen adds only these narrowing checks; the server remains the authority:
  - **Propose unlock:**
    - `lost_digest` must be 64 lowercase hex;
    - `mode` must be `lock_release` or `recovery_resume`;
    - in `recovery_resume`, the `lost` list must parse (otherwise the confirm cannot state what is recorded as lost);
    - `lost` is always the read's `lost_digest`, never computed or edited by the screen.
  - **Approve:** the installation's resume `mode` must be `lock_release` or `recovery_resume` (see the confirm).
  - **All commands:** the page `revision` must be non-empty.
- **Blocker display policy.** A blocked action is never shown as a button. Its blocker is either hidden or explained:
  - **Hidden only** (the row's state already says why):
    - `not_pending` on a closed row;
    - `not_proposer`;
    - `locked` on Lock, because the state reads Locked;
    - `not_recovering` on Propose unlock when the mode is `none`, because the mode sentence says it.
  - **Explained with a sentence:** `own_proposal`, `already_proposed`, `retired`, `not_console_kind`, `recovering`, `open_calls`, `no_restore_point`, `incomplete`, `lost`, `kind_coverage`, `generation`, `not_found`, `invalid`, `limit`, `conflict`, `missing` and any unknown code.
  - `not_pending` on a pending row and `not_recovering` with a mode other than `none` are unexpected and therefore explained.
  - `not_console_kind` is explained only on pending rows. Closed rows carry `not_pending` first.
- **Resend.** A pending attempt (unknown outcome) keeps its command usable even if the reloaded row no longer allows it.
  - The resend skips every read-based check above (availability, the propose-unlock and approve narrowing, the mode). Only the frozen target must still be well-formed.
  - It reuses the frozen target, values (`lost`), reason and `expected_revision`, and the journal returns the original result, because every target stays resolvable (see Replay).
  - Its confirm restates the frozen pre-send snapshot (mode, state, generation, impact sentence and lost list as they were before the first send), never the reloaded row, and says "This repeats your earlier request exactly as it was sent."

## Blocker sentences (every code, unknown fallback)

Every sentence has EN, KO and AR copy in `live-codes.ts`. Any other code reads "This is not available now (code)" with the code in an ltr isolate. `missing` reads "This is not available now."

| Code | Where | EN |
| --- | --- | --- |
| `own_proposal` | approve | You proposed this. Another administrator must approve it. |
| `not_proposer` | withdraw | (no sentence; Withdraw is hidden) |
| `not_pending` | approve, withdraw | (closed rows: no sentence) / (pending row, unexpected) This is no longer waiting for a decision. Reload. |
| `retired` | approve, withdraw | The recovery registry no longer holds this proposal. Nothing can be done with it here. |
| `not_console_kind` | approve | This kind of change cannot be approved from the console in this build. Its proposer can still withdraw it. |
| `already_proposed` | propose unlock | Unlock proposal #{n} is already waiting for another administrator's approval. / Your unlock proposal #{n} is waiting for another administrator's approval. (The number is the pending `unlock` row for the installation. "Your" is decided from the read only: that row's `approve_blocked` is `own_proposal`, or its withdraw is allowed. Without a row: "An unlock proposal is already waiting for approval.") |
| `locked` | lock | (no sentence; the state reads Locked) |
| `recovering` | lock | The installation is in recovery and already locked for writes in the recovery registry. It cannot be locked separately. |
| `not_recovering` | propose unlock, approve | (mode `none`: no sentence) / The installation is not locked or in recovery, so there is nothing to unlock. |
| `open_calls` | propose unlock, approve | Writer calls are still open. They close when the writer reports them or at their deadline. |
| `no_restore_point` | propose unlock, approve | The recovery has no restore point, so it cannot be resumed. |
| `incomplete` | propose unlock, approve | The recovery is not complete: a domain is not verified, an external system is not reconciled, or decision events are still pending. |
| `lost` | propose unlock, approve | The confirmed lost sequences do not match the recovery's current lost sequences. |
| `kind_coverage` | propose unlock, approve | A writer kind has no external system to reconcile against. |
| `generation` | propose unlock, approve | The installation's generation changed after this was proposed. |
| `not_found` | propose unlock, approve | The installation is no longer registered in the recovery registry. |
| `invalid` | propose unlock, approve | The recovery registry could not check this request. |
| `limit` | propose unlock, approve | A recovery registry limit is reached. |
| `conflict` | propose unlock, approve | Another record already uses this identifier. |

## Attempts, refusals and reactions

- **Attempts.** A request ID and an idempotency key are kept per command and target in the scoped store.
  - They are reused only to resend identical values (`lost` and reason) after an unknown outcome (network failure or `busy`), even if a reload moved the revision.
  - Any edit or any definite answer discards them, including `partial`. A journaled `internal_error` replays the same failure for the same key, so a new send needs a new key.
  - A pending attempt shows "The last request may have been applied. Check the list, or send it again."
- **Pre-send snapshot.** Before its first send, an attempt stores:
  - the listed proposal numbers (for `propose_unlock`, for the duplicate check);
  - what its confirm stated: the installation, mode, state, generation, impact sentence, recovery ID and lost list (for the resend confirm);
  - context for vanished-row notices: the installation and the proposal number.
- **Reason.** The reason is trimmed before sending. An empty or whitespace-only reason is rejected on the client ("Enter a reason"), as is one over 256 UTF-8 bytes.
- **Drafts.** The reason is kept per command and target while the session lasts. `lost` is never a draft; it always comes from the read, or from the frozen attempt on a resend.
- **Reactions:**
  - **`invalid` with fields:**
    - `lost` (propose unlock) → "The installation's lost sequences or state changed. Review the current state." and a reload. The form closes when the reloaded row no longer offers Propose unlock.
    - `rationale` → the reason field.
    - Any other field (`target_id`, `expected_revision`, `idempotency_key`, `request_id`, `client`, the excess marker or an unknown name) → "The request could not be sent as built. Reload and try again." It is never mapped to the reason field.
  - **`invalid` without fields** → the reason field ("The reason is too long" when it is over 256 bytes, otherwise "Check the reason").
  - **`stage` or `refused`:**
    - Reload, then the row's current blocker sentence.
    - When the reloaded row shows no blocker (or there is no row), a kept per-command sentence:
      - approve: "The recovery registry refused this approval. Nothing changed. Review the proposal and the installation's state.";
      - withdraw: "The recovery registry refused this withdrawal. Nothing changed.";
      - lock: "The lock was refused. Nothing changed. The installation may already be locked or in recovery.";
      - propose unlock: "The unlock proposal was refused. Nothing changed. Another unlock may already be waiting, or you may have too many pending proposals."
    - The kept sentence clears on the next definite answer for that command and target, or when the reloaded row shows a blocker or closes. Kept sentences for rows that left the read are dropped (P02 `pruneRefusals`).
  - **`conflict`** → reload, keep the reason, and ask to confirm again. If the reloaded read no longer offers the command, the form closes, focus moves to the row note, and the row's state or blocker sentence explains why.
  - **`denied` on a command:**
    - Reload. If the read succeeds: "This proposal is no longer available" or "This installation is no longer available".
    - If the read is denied, the screen-level denied block.
    - With a pending attempt and the proposal missing from the reloaded read (a closed entry evicted beyond 200; not reachable with the sample seed): "Your earlier request may have been applied. The proposal is no longer listed." The attempt is discarded.
  - **Pending attempts whose row vanished.** After every reload, a pending attempt whose target is no longer in the read is shown in a view-level notice above the lists: "This item is no longer listed. Your earlier request may have been applied." The notice has a Dismiss button and clears when the attempt is discarded.
  - **Other failures:**
    - `unauthenticated` → sign in again;
    - `limited` → the wait;
    - `busy` or `unavailable` → retry with the same attempt;
    - `stale` → "Start again" (renew the attempt);
    - `partial` → the operation text with its ID, a reload, and the attempt discarded;
    - anything else → "Failed".
- **Duplicate results.** A `propose_unlock` whose returned `proposal` is in the attempt's pre-send snapshot announces "This was already proposed as #{n}".
  - A resend compares against the snapshot taken before the first send, so a replayed stored success is announced as the new proposal it was, not as a duplicate.
  - Through the real API this message is not expected: a second unlock proposal is refused as `invalid_stage` (`already_proposed`, `recoverycmds.go:141-142`), and a stale tab gets `revision_conflict`. The message is kept as a defence and is covered only by model and service tests with a stubbed success.
- **Successes** name the result in words:
  - approve: "Unlock proposal #{n} applied. {installation} is serving." The installation part is added only when the reloaded read shows `serving`.
  - withdraw: "Proposal #{n} withdrawn."
  - lock: "{installation} is locked. No host was changed."
  - propose unlock: "Unlock proposal #{n} is waiting for another administrator's approval."
  - Unknown result states read "Unknown (code)".

## Unknown versus zero or empty

- The page `revision` is an opaque string, shown and sent as is. An empty one reads "Unknown" and disables commands.
- Numbers are finite safe integers or `null`; `null` reads "Unknown". A real 0 reads "0".
  - Non-negative: `generation`, `point_generation`, `history_total`, `lost_count`, `open_calls_total`, `unknown_total`, `skipped`, `pending_total`, `sequence`, `number` (proposal), `skew_seconds`, `unlock_generation` and each `lost` entry.
  - Signed: `number` (call and target) and `high_water`.
- `point_generation`, `as_of` and the digests are read only when `recovery.state` is `open`. `skew_seconds`, `unlock_generation` and the other summary fields are read only for their kind; the zero placeholders of other kinds are never shown.
- Missing arrays are `null` and read "Unknown"; empty arrays read "None" or the view's empty sentence. A `lost` list with any element that is not a safe integer is `null`.
- Booleans (`fenced`, `other_scope_present`, `can_*`) are `null` when not a boolean. A null `can_*` fails closed.
- Times that are empty, unparseable or year 1 read "Unknown". The exception is an empty `stopped_at` while `serving`, which reads "Not stopped".
- Empty strings read "Unknown", with these exceptions:
  - `verifier` (empty reads "No verifier registered");
  - `series` (empty reads "No series");
  - an empty approver on a pending row (not shown);
  - an empty closing rationale on a row closed without one.
- Logins fall back to IDs.
- An unknown enum value reads "Unknown (code)", never blank.

## Mobile, RTL and wrapping

- **Mobile.** Below 768px:
  - tables become stacked labelled rows; grids and forms become one column;
  - the restore-sets table keeps the spec order as labelled rows;
  - form and confirm buttons go full width.
  - Digests wrap with `overflow-wrap:anywhere` inside `<bdi dir=ltr>`.
  - No horizontal scroll at 390px.
- **RTL.** A compound identifier is one ltr isolate, never split into pieces in RTL:
  - `recovery-installation:` and `recovery-proposal-` targets;
  - call targets `{key} · {series} #{number}`;
  - "{g} → {g+1}" in the lock confirm;
  - lists of chunk indices and lost sequences.
  - Logins are isolated as ltr. Dates use FSI…PDI inside sentences. Reasons use `<bdi>` (first strong).

## Vocabularies (EN / KO / AR)

All are in `live-codes.ts` with an "Unknown (code)" fallback; tests check every key in every language. Korean and Arabic are AI drafts; the open terminology choices are listed under Out of scope.

| Group | Code | EN | KO | AR |
| --- | --- | --- | --- | --- |
| Section | configured | Configured | 설정됨 | مُعدّ |
| Section | not_configured | Not configured | 설정되지 않음 | غير مُعدّ |
| Connection | not_connected | Not connected | 연결되지 않음 | غير متصل |
| Shown | not_shown | Not shown | 표시하지 않음 | غير معروض |
| Installation state | serving | Serving | 운영 중 | قيد التشغيل |
| Installation state | locked | Locked | 잠김 | مقفل |
| Installation state | recovering | In recovery | 복구 중 | قيد الاسترداد |
| Public lock | serving | Not locked | 잠기지 않음 | غير مقفل |
| Public lock | locked | Writes locked | 쓰기 잠김 | الكتابة مقفلة |
| Public lock | recovering | Writes locked (in recovery) | 쓰기 잠김(복구 중) | الكتابة مقفلة (قيد الاسترداد) |
| Recovery | open | Open | 진행 중 | مفتوح |
| Recovery | none | No recovery open | 진행 중인 복구 없음 | لا يوجد استرداد مفتوح |
| Phase | verify | Verifying the restore | 복원 검증 중 | التحقق من الاستعادة |
| Phase | reapply | Reapplying decision events | 결정 기록 재적용 중 | إعادة تطبيق أحداث القرار |
| Phase | ready | Ready for a resume decision | 재개 판정 대기 | جاهز لقرار الاستئناف |
| Phase | not_recovering | Not in recovery | 복구 중 아님 | ليس قيد الاسترداد |
| Mode | recovery_resume | Resume after recovery | 복구 후 재개 | الاستئناف بعد الاسترداد |
| Mode | lock_release | Lock release | 잠금 해제(복구 없음) | رفع القفل |
| Mode | none | Nothing to resume | 재개할 대상 없음 | لا شيء للاستئناف |
| Domain | database | Database | 데이터베이스 | قاعدة البيانات |
| Domain | evidence | Evidence files | 증빙 파일 | ملفات الأدلة |
| Domain | keys | Keys | 키 | المفاتيح |
| Domain | policies | Policies | 정책 | السياسات |
| Domain | modules | Modules | 모듈 | الوحدات |
| Domain | numbering | Document numbering | 문서 번호 | ترقيم المستندات |
| Domain | submissions | Submissions | 제출 기록 | الإرسالات |
| Domain | outbox | Outbox | 발신 대기열(outbox) | صندوق الصادر |
| Domain | deletions | Deletion records | 삭제 기록 | سجلات الحذف |
| Domain status | pending | Not checked yet | 검증 전 | لم يُتحقق بعد |
| Domain status | verified | Verified | 검증됨 | تم التحقق |
| Domain status | mismatch | Mismatch | 불일치 | عدم تطابق |
| Writer kind | erp | ERP | ERP | ERP |
| Writer kind | issuance | Invoice issuance | 청구서 발행 | إصدار الفواتير |
| Writer kind | delivery | Delivery | 전달 | التسليم |
| Writer kind | payment | Payment | 결제 | الدفع |
| Writer kind | message | Messaging | 메시지 발송 | المراسلة |
| External status | pending | Not reconciled | 대사 전 | لم تتم المطابقة |
| External status | reconciled | Reconciled | 대사 완료 | تمت المطابقة |
| Fenced | true | Fenced | 차단됨 | محجوب |
| Fenced | false | Not fenced | 차단 안 됨 | غير محجوب |
| Calls source | recovery | From the open recovery | 진행 중인 복구 기준 | من الاسترداد المفتوح |
| Event kind | delete | Deletion | 삭제 | حذف |
| Event kind | purge | Purge | 영구 삭제 | محو نهائي |
| Event kind | hold | Hold | 보존 보류 | تعليق الحذف |
| Event kind | extend | Retention extension | 보존 기간 연장 | تمديد الاحتفاظ |
| Event kind | revoke_key | Key revocation | 키 회수 | إلغاء مفتاح |
| Event kind | revoke_access | Access revocation | 접근 권한 회수 | إلغاء الوصول |
| History outcome | completed | Completed | 완료 | مكتمل |
| History outcome | abandoned | Abandoned | 중단됨 | متروك |
| Proposal kind | installation | Installation | 설치 | التثبيت |
| Proposal kind | writer | Writer | 쓰기 주체 | جهة الكتابة |
| Proposal kind | external | External system | 외부 시스템 | نظام خارجي |
| Proposal kind | verifier | Verifier | 검증 주체 | جهة التحقق |
| Proposal kind | begin | Recovery opened (isolated restore record) | 복구 개시(격리 복원 기록) | فتح استرداد (سجل استعادة معزولة) |
| Proposal kind | unlock | Unlock | 잠금 해제 제안 | إلغاء القفل |
| Proposal kind | prune | Prune decision events | 결정 기록 정리 | تقليم أحداث القرار |
| Proposal kind | abandon | Abandon recovery | 복구 중단 | التخلي عن الاسترداد |
| Proposal state | pending | Pending | 승인 대기 | بانتظار الاعتماد |
| Proposal state | applied | Applied | 적용됨 | مطبّق |
| Proposal state | withdrawn | Withdrawn | 철회됨 | مسحوب |
| Proposal state | retired | Retired | 폐기됨 | مُسقَط |
| Migration module | migration | Customer migration import | 고객 이관 반입 | استيراد ترحيل العملاء |
| Migration module | balances | Opening balances | 기초 잔액 | الأرصدة الافتتاحية |
| Result state | applied, withdrawn, pending, locked | (as above) | | |

The spec's `partial` and its job enum are not P03 states in this API. No P03 value reads "Partly done".

## Tabs

The shell labels the tabs from `screens.json`:
- KO: 복원 세트 / 격리 검사 / 외부 대사 / 고객 이관 / 재개 판정.
- In-screen view titles:
  - EN: "Restore sets", "Isolation check", "External reconciliation", "Customer migration", "Resume decision";
  - KO: the registered labels;
  - AR: "مجموعات الاستعادة", "فحص العزل", "المطابقة الخارجية", "ترحيل العملاء", "قرار الاستئناف".

## External references

The ui-designer (AI, read-only) recalled these from memory; nobody opened them, so they are unverified:
- **Amazon RDS point-in-time restore** (https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_PIT.html). Adopted: the as-of time and the backup identity shown as facts before any action. Rejected: the restore-to-new-instance form and one-click restore.
- **Azure Resource Manager locks** (https://learn.microsoft.com/en-us/azure/azure-resource-manager/management/lock-resources). Adopted: a lock as recorded state with who, when and notes, and wording that it limits management, not running resources. Rejected: removal by a single person.
- **Heroku maintenance mode** (https://devcenter.heroku.com/articles/maintenance-mode). Adopted: the state in words with its consequence. Rejected: a toggle without a reason or a confirm.
- **AWS Backup multi-party approval** (https://docs.aws.amazon.com/aws-backup/latest/devguide/multipartyapproval.html). Adopted: a distinct second approver and a pending/closed list. Rejected: the approval-team UI.
- **Stripe payout reconciliation report** (https://docs.stripe.com/reports/payout-reconciliation). Adopted: unreconciled items listed separately with counts. Rejected: automatic matching and coloured pills.

The visual specification (ui-designer, AI) defines the `p03-` classes and the layout; root keeps it with the ticket's working notes. The domain disclosure opens by default when any domain is mismatched (root decision).

## Rules carried from the HB-283 and HB-285 reviews

All are part of this design from the start:
- **Host and remount:** live branch before the preview wrapper, locale-only key, hooks unconditional, remount per view.
- **URL state:** validated selections. **Focus and live region:** status focused after success, row note after refusal, opener after cancel, summary links to fields.
- **Stale responses:** the request sequence.
- **Store:** scoped by workspace, user and session, and cleared on sign-out. Pre-send snapshots; resend only with identical values.
- **Mobile:** stacking at 390 with no horizontal scroll; one primary button during a confirm; an open form hides its opener and siblings and has an action heading.
- **RTL:** one ltr isolate per compound ID; FSI for dates.
- **Unknown versus zero and empty.** A year-1 time reads "Unknown". A missing list is never shown as empty.
- **Blockers:**
  - a sentence for every blocker code, with an unknown fallback;
  - rows without a well-formed target offer no command; a missing number fails closed;
  - the proposer's rationale shows on pending rows and in the approve confirm;
  - logins fall back to IDs; "Showing the newest 50."
- **Refusals:** the kept sentence when the reloaded blocker is empty, and its clearing. On a command, `denied` → reload → "no longer available". A vanished pending attempt gets a view-level notice.
- **Terminology:** user-facing copy follows `.agents/rules/product-terminology.md`, the product glossary and `docs/ui/design-system/terminology.md`.
- **Truthful statements:** what each decision does and does not do. The lock is single-administrator and says so; a lock release checks no decision watermark and says so.

## Out of scope (owner residuals)

- **Not connected in this build:**
  - restore-set registration and inspection, isolated restore requests (`begin`), `abandon` and `prune` (restore, discard and delete effects);
  - writer permits, calls and outcomes; verifier reports, reconciliation pages, reapplication and unapplicable declarations; decision-event recording;
  - registration of installations, writers, externals and verifiers from the console;
  - the reconciler; network, host and credential effects; persistence; a decision ledger independent of the database.
- **Writer removal** is not offered: it is single-administrator and irreversible, and removed writers still count for `kind_coverage`. It needs a two-person design.
- **Lock release** checks no decision watermark, domain verification or external reconciliation. Open calls are listed only while a recovery is open, so a lock release's `open_calls` blocker cannot be shown with its calls.
- **Customer migration** is not connected: binding `modules/migration` read-only, `modules/balances` opening balances (QA-09), and the P03 customer-scope export package.
- **Permissions and audience:** separate recovery verification, execution, customer export, ERP reconciliation and resume permissions (the HB-286 deviation), and AU and BL read access. Until then the read and every command use `AdministerInstallation`.
- **Sample limits:**
  - `sample-primary` cannot progress after the seed; its resume stays `incomplete`;
  - only `sample-standby` carries the working two-person lock-release flow, which is not a recovery resume;
  - the open call leaves the list one minute after the seed.
- **Not built from the P03 spec:**
  - filters (restore time, generation, check state, lock reason), server search and stable sort;
  - the selected-row drawer, the evidence/original view and before/after comparison;
  - bulk snapshots and cursors (the lists are small and bounded at 50);
  - links to P04, I02, S05, B10, B11 and R02;
  - keeping the selection across shell tab changes, if the shell drops `installation=`, and the unsaved-draft warning on a tab switch (drafts survive in the store).
- **Bounds:** the screen sees at most 50 proposals, history rows, open calls, unknown targets and pending events per installation, with totals. Closed proposals evicted beyond 200 no longer resolve.
- **Reviews:** owner FIX of the screen.
- **Terminology** (owner KO and AR review and technical-writer review against `.agents/rules/product-terminology.md`, the product glossary and `docs/ui/design-system/terminology.md`; the proposed terms are used until then):
  - generation: KO "generation" kept in Latin script, AR "الجيل";
  - writer: KO "쓰기 주체", AR "جهة الكتابة";
  - fenced: KO "차단됨", AR "محجوب";
  - decision watermark: KO "결정 워터마크", AR "علامة القرار المائية";
  - lost sequences and their digest: KO "유실 순번", "요약값", AR "التسلسلات المفقودة", "ملخص";
  - outbox: KO "발신 대기열(outbox)";
  - lock release (mode) versus unlock (proposal kind): KO "잠금 해제(복구 없음)" versus "잠금 해제 제안", pending a clearer pair;
  - the `begin` kind: "Recovery opened (isolated restore record)", KO "복구 개시(격리 복원 기록)";
  - every other KO and AR string in this plan, which are AI drafts.

# Verification plan

- **Model tests** (`live-model.test.ts`):
  - availability parsing fails closed for all four flat pairs: `can_*` true with a blocker, non-string blocker, empty blocker with `can_*` false, missing keys;
  - the blocker display policy: each hidden-only code hidden, each explained code with its sentence; `not_pending` on a pending row and `not_recovering` with mode `lock_release` explained; `not_console_kind` explained only on pending rows; `already_proposed` naming the pending unlock number, own and other, and the fallback without a row;
  - target patterns: malformed, mismatched number, over 128 bytes, characters outside the ID set, an installation ID of 107 bytes;
  - propose-unlock narrowing: a malformed `lost_digest`, an unknown mode, an unparsable `lost` list in `recovery_resume`; `lost` sent equals the read's digest;
  - approve narrowing: no resume row, or an unknown mode → not offered with its sentence;
  - the lock confirm with a known and an unknown generation (the "from {g} to {g+1}" part omitted); the single-administrator sentence present in EN, KO and AR;
  - the approve and propose-unlock impacts per mode, with the no-watermark sentence only for `lock_release`;
  - the restore-sets joins (evidence counts, watermark text per mode, reconciliation counts, public lock) and their unknown fallbacks; "No decision events pending" with a real 0 and "{n} declared lost" from a non-empty lost list; an open recovery with an empty domain list reading "No restore point" for as-of, generation, evidence check and reconciliation;
  - mismatch wording: listed indices, an empty `mismatch_chunks` list, and exactly 100 indices;
  - the `begin` kind and summary, and the "Applied means recorded" note; no restore wording ("restored", "restore completed" and their KO and AR equivalents) in any EN, KO or AR copy except the statements that nothing is restored;
  - closed rows: applied shows the approver, withdrawn the withdrawer, retired neither;
  - "Your unlock proposal" decided only from `own_proposal` or withdraw availability, and "Unlock proposal #n" otherwise;
  - approve narrowing on `unlock_generation` and `lost_digest` mismatches;
  - the lock confirm's retired-begin sentence with and without a pending `begin` row;
  - resends after a state change, for approve and propose unlock: the read-based narrowing is skipped, only the frozen target is checked, and the confirm restates the frozen snapshot (mode, generation, impact, lost list), not the reloaded row;
  - the open-calls text per source (`recovery` empty with a real 0, `not_shown`, unknown), `other_scope_present` true, false and missing;
  - `high_water` with and without a series; `stopped_at` empty while serving versus locked;
  - reason trimming and whitespace-only rejection; UTF-8 byte counting; the `lost` and non-reason field mappings;
  - blocker sentences for every code in every language, the `missing` sentence and the unknown fallback;
  - proposal summaries per kind with the unknown fallback, and only the kind's own fields used; proposals ordered pending first, newest first; `retired` rows offering nothing;
  - unknown versus zero for every numeric field, null lists, null flags and a year-1 time;
  - every vocabulary key in EN, KO and AR, with no copy claiming a restore, a prune, a deletion, or a host, network or credential change;
  - failure reactions; the attempt store (identical values only, scoped by session, cleared by `forget`, discarded on `partial`);
  - the kept refusal sentence for every command when the reloaded blocker is empty, and its clearing;
  - command `denied` → reload → the per-target "no longer available" sentence, and the evicted-proposal sentence with a pending attempt;
  - the vanished-attempt notice;
  - the duplicate announcement against the snapshot with a stubbed success, and no announcement when a resend replays a stored success;
  - the URL selection validation.
- **Service tests** (`recovery.test.ts`):
  - hardened parsers for every section (missing arrays, wrong types, unknown enums, unsafe and negative integers, the `none` recovery object);
  - the whole-page `not_configured` and an unknown page state; reconciliation `not_configured`; migration `not_connected`;
  - the command envelope with a field (`propose_unlock` with `lost`) and without;
  - resend identity, and the result fields (`proposal`, `installation`, `target`);
  - stubbed `invalid_stage` and `prerequisite_failed` answers leading to the kept sentences; a stubbed `internal_error` with an operation ID mapped to `partial`.
- **Checks:** `npx tsc --noEmit`, `npm test`, `npm run build`.
- **Playwright QA** against the real API (`hb-api --seed sample`, built from this branch's base), with the console preview (`HB_GATED=1 vite preview`). Each group starts on a fresh API (restarted with an empty data directory), as `qa285run.sh` does, so no group's starting state depends on another group or on order.
  - **All five views** (Group A's fresh API, before its commands) in EN, KO and AR at 1440 and 390:
    - no horizontal scroll; `dir` per locale; neutral tag borders; one primary button during a confirm;
    - the statement, the permission note and each view's "not connected" sentence present;
    - restore sets: `sample-primary` shows `recovery-1`, "8 of 9 domains verified · 1 mismatched", "2 decision events not yet reapplied", "0 of 2 external systems reconciled" and "Writes locked (in recovery)"; the outbox detail reads chunk index 1; `sample-standby` shows "No recovery open" and "Not checked outside a recovery";
    - isolation: `sample-standby` reads Locked, generation 2, "No verifier registered", "Open calls are listed only while a recovery is open", and one lock-log row by `admin@example.invalid`. Permits read "Not shown".
    - **Open call (clock-dependent):** for `sample-primary`, using the read's `now` and the call's `deadline` from the API response the page used, `call-2` is asserted only when the deadline is at least 5 s after `now`, and its absence ("No open calls in this workspace.") only when `now` is at least 5 s after the deadline. Otherwise the check is skipped and logged. Either way, reconciliation still lists `sample-invoice-2` as fenced with high water 2.
    - migration reads the not-connected sentence with two modules;
    - resume: `sample-primary` reads the `incomplete` sentence and two pending events (Deletion on Evidence files, Access revocation on Database); `sample-standby` reads Lock release.
  - **Group A, as the first administrator** (`admin@example.invalid`):
    - proposal #10 (proposed by the second administrator) shows Approve and no Withdraw; `sample-standby` resume reads "Unlock proposal #10 is already waiting…";
    - `sample-primary` offers no Lock (the `recovering` sentence) and no Propose unlock (`incomplete`);
    - approve #10. The confirm states the lock release and "checks no decision watermark". After it:
      - `sample-standby` reads Serving, generation 2, "Not stopped" and "Nothing to resume" with no Propose unlock;
      - #10 reads applied with the approver `admin@example.invalid`;
    - Lock `sample-standby`. The confirm reads "Takes effect now; no second approval: you alone lock this installation" and "from 2 to 3". After it: Locked, generation 3, two lock-log rows, and no Lock button;
    - Propose unlock on `sample-standby`. The confirm states the lock release and the empty lost list. The result is #11, which reads "You proposed this" with Withdraw, and the resume row reads "Your unlock proposal #11…";
    - withdraw #11 → withdrawn; Propose unlock is offered again.
  - **Group B, as the second administrator** (`admin2@example.invalid`), with the first administrator only where stated:
    - #10 reads "You proposed this" with Withdraw and no Approve; `sample-standby` reads "Your unlock proposal #10…";
    - withdraw #10 → withdrawn; the resume row then offers Propose unlock (lock release);
    - propose unlock → #11 pending, "You proposed this";
    - as the first administrator, approve #11 → `sample-standby` Serving at generation 2;
    - as the second administrator, lock `sample-standby` → Locked, generation 3, with the lock-log row by `admin2@example.invalid`;
    - no closed row (#1–#11) offers any action, and no `begin` row offers Approve.
  - **Group C, failure paths** (first administrator unless stated):
    - **Revision conflict:** open Approve #10 with a reason in tab 1. In tab 2, move the revision with a P02 command (`subscription_pause` on `sample-erp-invoices`). Confirm in tab 1 → conflict, reload, reason kept, confirm again → applied.
    - **Lost response after apply,** by route: fetch the real response, then abort it. Lock `sample-standby` after the conflict step has applied #10 (the steps run in this order on the group's one API). The pending-attempt sentence shows. The resend returns the stored `locked` result from the journal (the target stays resolvable). The lock log still has exactly two rows.
    - **Lost request before the server:** route abort without fetching, on Propose unlock for the now locked `sample-standby`. The pending sentence shows; the resend creates the proposal once (#11).
    - **Client checks:** a reason over 256 bytes and a whitespace-only reason are caught on the client.
    - **Focus:** the status message is focused after a success; the row note after a refusal (the `incomplete` blocker after a stubbed `invalid_stage` by route); the opener after Cancel.
    - **Drafts:** a reason typed into Propose unlock survives a tab change to isolation and back.
    - **Member:** `member@example.invalid` sees the denied block.
    - **URL state:** an invalid `view`, `installation` or `proposal` in the URL is ignored; a valid `installation=sample-standby` on isolation is marked `aria-current`.
  - **Regressions**, each on a fresh API:
    - P02: `qa285run.sh` groups A–D;
    - P01: qa283full;
    - O03: qa281;
    - O02: qa275;
    - O01: qa273;
    - M01: qa264.
  - Console errors during QA: none expected, except the deliberate aborts.
- **Reviews and record:** ui-designer render comparison; independent design and change review; `verification.md`.

# Design review round 2 clarifications

- Focus after a refusal: stub `invalid_stage` on Approve #10 and assert focus on the row note carrying the kept sentence "The recovery registry refused this approval…" (not the `incomplete` blocker, which is never offered).
- The draft-survives-a-tab-change QA step runs right after the Lock step in Group C, using the Approve #10 reason before the conflict step, so Propose unlock availability does not matter.
- The Lock-confirm begin-retirement sentence has EN, KO and AR copy in `live-codes.ts` and a model test; after the lock the read still shows that begin entry as pending with Withdraw for its proposer until an approve or withdraw attempt marks it retired, and the test expects that.
- The external references are recalled, not opened; verification.md records them as unverified evidence (residual).

