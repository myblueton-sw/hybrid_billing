---
type: VerificationRecord
title: HYB-298 verification — M03 approval requests, revocation states and the decision ledger
description: What was built against the HYB-298 plan, the checks that were run with their observed results, the rebase deltas against main be1bfa3, deviations from the plan and what remains unverified.
status: final
sources:
  - resource: plan.md
---

# Scope

This record covers the implementation of [the plan](plan.md).
- The plan was written on `3e36101` and implemented on main `be1bfa3`.
- Root then rebased it onto main `07458be`, which includes HYB-315 P05 (see the second rebase section).
- A delegated agent wrote the code; root re-ran the checks marked "root". The closure of HB-266 and HB-267 residuals is recorded here; their records are not edited.

# Rebase deltas (plan base `3e36101` to `be1bfa3`)

Only positions and counts changed. No behaviour or contract conflict was found.

- `access.go` was unchanged between the two bases; the plan's `access.go` line references still apply.
- `admin.go`: the definition list gained `evidenceDefinitions()` (HYB-290). The access definitions stay in the same position; the Q6 handler line numbers are unchanged.
- `seed.go`: the sample seed gained a `release-evidence` step (HYB-290). The sample seed now has 17 counted steps. The ledger step runs after every seed step and counts only when replay closes something, so `world_test.go` still expects 17.
- `server.go`: HYB-290 added the `trace`, `perf` and `ev` fields and two registry steps. The `Config.Access` field, the `changes`, `ledger` and `startedAt` fields and the `accesschange` registry step were added after them.
- Step-up (HYB-293): `execute` checks `s.stepUp(command)`, which is `s.mutating[command] && !exempt[command]` with an empty exempt set. The step-up check runs before `Execute` under the domain lock. The five new commands are mutating, so they are covered without change. `TestEveryRegisteredMutatingCommandRequiresStepUp` enumerates the registry and checks at least 63 commands; it passes. The console resend after a password confirmation reuses the same request ID, idempotency key and payload, including the approve `digest`; a refused step-up records nothing, so the resend is the first execution.
- Console: `access/Live.tsx` uses `useAdminClient()` from HYB-293. The new panels use the same client, so they get the shared step-up dialog. `admin.ts` command results moved to `outcomeOf`; the line numbers in the plan's `admin.ts:392-404` reference moved.

# Checks run

| Check | Command | Observed |
| --- | --- | --- |
| Format | `gofmt -l apps modules` | no files |
| Vet | `go vet ./...`; `GOOS=linux,windows,darwin go vet ./modules/decisionledger` | clean |
| Tests | `go test ./...` | all packages ok |
| Race | `go test -race` on `modules/accesschange`, `modules/decisionledger`, `modules/auth`, `modules/gate`, `apps/api/server`, `apps/api/cmd/hb-api` | all ok except the pre-existing `TestRun` flake in `hb-api` (see below; also fails on main) |
| Coverage | `go test -cover` | `accesschange` 100.0%, `decisionledger` 100.0%, `apps/api/server` 100.0%; all new and changed functions in `auth/listing.go` and `gate/gate.go` 100% (packages 97.4% and 97.1%, the rest is pre-existing); `hb-api` new code covered, the package is 89.7% because `main` and two pre-existing error branches were already uncovered |
| Cross builds | `GOOS/GOARCH go build ./apps/api/cmd/hb-api` | linux/amd64, windows/amd64, darwin/arm64 ok |
| Console | `tsc --noEmit`, `npm test`, `npm run build` | ok; 570 tests pass (after the amendments) |
| File size | `wc -l` on changed files | largest changed file 824 lines |

# Browser QA

Production build of the console served from the worktree `dist` through a QA-only static and proxy server (port 8820 was held by another session's server, and the console proxy target is fixed at 8820). Fresh `hb-api` built from this worktree, on a new mode-0700 data directory under the scratchpad that was deleted afterwards. Two browser contexts (`admin`, `admin2`). Runner: `qa298.mjs` with `qa298run.sh` in the session scratchpad. Screenshots: `qa298shots/` in the session scratchpad (44 files).

| Phase | Server start | Checks |
| --- | --- | --- |
| A | `-access-request-hours 72` | 44 pass, 0 fail |
| B | restart, same data | 15 pass, 0 fail |
| C | restart after cutting the last ledger line in half | 13 pass, 0 fail |
| D | restart after flipping one byte of line 1, marker present | 24 pass, 0 fail (after the amendments; includes `revocation_unconfirmed` in EN, KO and AR and in the API read) |
| E | restart with `-accept-new-decision-ledger` | 14 pass, 0 fail |

Covered: section order; approval settings with one value set by flag and no edit control; propose, approve, reject and withdraw; focus return after cancel and focus moves after approve; a grant revocation reaching `revocation_complete` with sequence 2 and the watermark; an out-of-date request offering only reject and withdraw; the holder approving their own revocation; no eligible approver with the `blocked` copy; neutral computed borders on the new record, ledger, settings and panel surfaces; replayed records with "From an earlier server run", "Re-applied at start", "Target not present after restart" and the not-kept requests note; the torn-tail recovery line; isolation with the request form disabled with the reason, an approved revocation shown as `decision_unconfirmed` with "this server process only", the target row "Revocation not confirmed" and the effect preview note; the M01 refusal message when reinstating a user in isolation; the replacement note and the replaced file kept. Each phase rendered EN, KO and AR at 1440 and 390 with no horizontal scroll and the right direction; at 390 a request panel opens as a step. 0 page or console errors.

QA findings fixed during the run:
- The actions column of the grants table clipped "Request revocation" and long logins overflowed in the request table. Column widths and wrapping were adjusted in `access/access.css`.
- A grant whose revocation was not confirmed moved into the collapsed "Ended grants" disclosure, hiding "Revocation not confirmed". Such grants now stay in the current list without a revoke action.

# Amendments after the security change review (2026-11-18)

Applied per "Root amendments after the security change review" in [the plan](plan.md).

- **m1, final newline lost.** `modules/decisionledger/record.go` (`next`, `confirmedTail`, `verify`): a final segment without its newline that is a complete, chained record with a valid `sum`, as it stands or with its last byte removed, isolates the ledger with `integrity_failed` at that line. It is never truncated as a torn tail. Tests: `TestAConfirmedFinalRecordIsNeverTruncated` (newline removed; newline replaced by another byte): isolated at line 3, file bytes unchanged, no recovery.
- **D1, verified prefix.** `ledger.go` `check` keeps the decisions verified before the first failing line, and `Decisions` returns them while isolated. Commits stay refused and no file is written. Only the `integrity_failed` case has a prefix: a missing ledger has none, and an `id_mismatch` ledger is not replayed, because the whole file belongs to another ledger. Tests: `TestIsolationKeepsTheVerifiedPrefix` (module), `TestIsolationReappliesTheVerifiedPrefix` (server: a damaged line 3 keeps the line 2 revocation applied across two restarts, the damaged record is not applied, the read stays `isolated` with problem line 3, a grant addition is refused, a new delegation revocation reads `ledger_isolated`, and the data directory is unchanged).
- **m2, replay errors.** `apps/api/server/ledger.go` `replayOne` and `replayLedger` return the `RevokeGrant` or `RevokeDelegation` error. `Start` is then refused with `ErrStart` naming the revocation and record, the error is logged as `decision_ledger_replay_failed`, and the ledger is closed so its lock is released. The fault point `replay` exists for tests. Test: `TestReplayFailureRefusesStart` (grant and delegation records).
- **m3, grant row state.** `apps/api/server/access.go` (`closedState`, `unconfirmedTargets`, the grant and delegation loops): a grant, or a delegation, closed by a revocation in `received` or `decision_unconfirmed` reads `revocation_unconfirmed`. A confirmed or replayed revocation still reads `revoked`. Applying it to delegations as well goes slightly beyond m3, which names only grants; it keeps m4 on the server state. Console: `stateLine` (now in `access/live-model.ts`) maps `revocation_unconfirmed`, and any unknown state, to "Revocation not confirmed" (EN/KO/AR in `live-copy.ts` `stUnconfirmed`/`stUnconfirmedWhy`). The console no longer joins revocations for the row state; the join remains only for the effect-preview note by subject. Tests: API assertions in `TestLedgerIsolation`, `TestRestartReplaysRevocations` and `TestIsolationReappliesTheVerifiedPrefix`; console model test for EN/KO/AR and unknown states.
- **m4, delegations.** `people` and `delegationsOf` keep every state except `revoked`, `expired` and `ended` in the open list. Revoke buttons appear only for `active` or `pending` rows, for grants and delegations alike. Test: `live-model.test.ts` "unconfirmed revocations stay open for grants and delegations, and unknown states read unconfirmed".

# Round-2 review changes (2026-11-18)

- **N1.** `modules/decisionledger/record.go` `confirmedTail` returns the verified record (the whole segment, or the segment without its last byte when the newline was replaced), and `verify` accepts it into the prefix before marking that line as the problem. Tests: `TestAConfirmedFinalRecordIsNeverTruncated` now expects two decisions; `TestAFinalRecordWithoutItsNewlineStaysApplied` (server) asserts the grant revoked by that final record stays closed after a restart, the ledger reads `isolated` at line 2 with no recovery, and the file is unchanged.
- **N2.** `ledger.go` `check`: a damaged line together with a marker that names another ledger (`marked && s.ledgerID != "" && mark.LedgerID != s.ledgerID`) isolates as `id_mismatch` with no prefix. When the damage is in the genesis line, so that no ledger ID can be read, the result is `integrity_failed` with an empty prefix. Tests: `TestAForeignMarkerWithADamagedLineReplaysNothing` (module), `TestAForeignMarkerAndADamagedLineReplayNothing` (server: nothing re-applied, no replayed rows).
- **Sync at start (plan amendment 7, corrected).** After verification and before any decision is replayed, `check` calls `Sync` on the locked handle (`startSync` in `modules/decisionledger/ledger.go`).
  - On a failure every verified record is still replayed, because revocations are the restrictive direction; skipping them would make revoked grants usable again.
  - A verified ledger, or one with a torn tail, becomes `stopped`. Start proceeds: increases are refused with `dependency_unavailable`, and no record is written (no truncation, no recovery record, no marker). An isolated ledger stays `isolated` and still replays its verified prefix.
  - `Status.NotDurable` is set, the read returns `ledger.not_durable: true`, and M03 shows "Revocations were replayed from records not confirmed durable …" in EN, KO and AR.
  - `Options.FailStartSync` lets the server's test-only fault hook (`ledger-start-sync`) inject the failure.
  - Tests:
    - `TestASyncFailureAtStartStopsButKeepsEveryVerifiedRecord` (module: verified ledger, torn tail, isolated prefix);
    - `TestTheStartSyncOption`;
    - `TestAStartSyncFailureStillReplaysAndStops` (server: the revoked grant stays revoked, the ledger reads `stopped` with `not_durable`, a grant addition gets 503, and the record count is unchanged);
    - console model and `admin` parsing tests.
- **Server test nit.** `TestReplayFailureRefusesStart` closes the ledger it re-opens.
- **`TestHeldLockFailsStart`.** The test now keeps the first server referenced (`defer f.s.Close()`). Without that, the garbage collector could finalize the first server's ledger file under load, which releases the lock, and the second `Start` then succeeded. This was observed once in a full run. A running `hb-api` keeps its server referenced for its whole life, so the product is not affected.
- **`TestRun`.** `apps/api/cmd/hb-api/main_test.go` now polls the health route until it answers, with a 30-second deadline, instead of a fixed 1-second wait. The HYB-298 flag test uses the same polling. `go test -race -count=5 ./apps/api/cmd/hb-api` passes.

# Render and terminology review changes (2026-11-18, plan amendment 8)

- **Render.**
  - The login helpers moved to `access/who.tsx`. Request rows show the login on its own line (`access-who`), and the action buttons sit in a `div.access-actions` inside a plain cell: a column on desktop, a wrapping row at 767 px and below.
  - `.live-sections` uses `minmax(0, 1fr)`, record headings wrap anywhere, and cleanup and ended-delegation lists use "–" markers at `inset-inline-start` with `role="list"`.
  - The impact list has a 24 px bottom margin. Setting identifiers break only after "_". "Request revocation" does not wrap at 768 px and above. The paused primary button shows a `not-allowed` cursor. The record heading isolates the login in `bdi dir="ltr"`.
  - Browser QA now measures `scrollWidth - clientWidth` of `.live-sections`, `.access-ledger`, `.access-record` and the requests table in every language and width; it was 0 in all.
- **Terminology.** Reviewer items C1–C26, H1 and H5 are applied in `request-copy.ts`, `live-copy.ts` and `users/live-copy.ts`.
  - The changes include the verified-prefix sentence, "not confirmed as recorded", "tries to record", "no longer applies", only the no-approver sentence for zero approvers, and "Any revocations replayed at start". Stopped and full now say "Revocations made now … will not be re-applied after a restart", and the replacement note says "any sample access revoked under it".
  - EN uses singulars and the heading "Access change requests". AR uses counted plurals through the shared `shared/plural.ts` `counted`, which retention now re-exports. AR terms: الإبطال, بصمة الطلب, رقم إصدار الحجب, دفتر القرارات with قيد/قيود, انتهت مدته, مدة سريان الطلب, في الانتظار, and the range "من … إلى …". KO: 권한 추가 요청, the spec out-of-date sentence plus "반려하거나 철회한 뒤 다시 요청하세요.", 새 원장으로 교체, the '회수' section references, and 결정 워터마크. M01 busy copy names the M03 screen.
  - The glossary rows for access change request, request revision, revocation, decision not confirmed, decision ledger and watermark are updated.
  - These words remain open for owner and native review.

# Mutation testing

The coordinator ran 791 mutants on the tree before the round-2 fixes. 715 were killed and 76 survived. For each survivor, the kill was confirmed by applying that mutant to the current tree and running the named test (scratchpad `mutcheck.py` with `mut-server.json`, `mut-mod.json`, `mut-more.json` and `mut-last.json`; every file was restored afterwards). Line numbers are those of the survivor list.

Killed by a new or strengthened test:

| Survivors (index: file:line) | Test |
| --- | --- |
| 2, 5, 10, 14 (`main.go:44`, `:57`, `:107`, `:108`) | `TestAccessSettingFlags`, which now calls the extracted `accessFlags` and `settingNumber` directly |
| 44, 45 (`accessrequests.go:66`) | `TestOnlyAdministratorsAreEligibleApprovers` |
| 48, 49 (`:72`), 163 (`:300`) | `TestCustomerScopedRequests` |
| 77 (`:106`) | `TestRequestsAreSupersededByBasisChanges/ceiling tie with a lower ID` |
| 116 (`:219`) | `TestCeilingNeedsFullScopeAndGranteesNeedActiveState` (suspended membership) |
| 122 (`:223`) | `TestGrantValidation` (now compares the refused field list exactly) |
| 179, 180, 181 (`:377`) | `TestMayEndListsOnlyEffectiveDelegationsOfTheHolder` |
| 199, 205 (`:405`), 213 (`:406`) | `TestRequestRowFlags` |
| 217, 218, 219 (`:420`, `:422`) | `TestRequestOrderAndShownLimit` |
| 256 (`ledger.go:52`) | `TestHeldLockFailsStart` (step counter unchanged by a failed ledger open) |
| 266–281 (`ledger.go:75`–`91`) | Code changed in round 1 (replay now returns errors). The same value mutations on the new code (`present` and `closed` for absent, protected, open and already-closed grants and delegations, and dropping the closed-delegation case) are killed by `TestReplayCountsOnlyWhatItCloses`. The `== nil` and `!= nil` mutants (271, 280) are gone; errors are covered by `TestReplayFailureRefusesStart`. |
| 299 (`ledger.go:165`) | `TestRevokedDelegationEndsItsDescendants` (ledger record says `revoke_delegation`) |
| 302 (`ledger.go:180`) | `TestLedgerMissingWithMarkerIsolates` (`created_at` is empty) |
| 312 (`revocations.go:80`) | `TestRevokingTheDelegatesOnlyGrantEndsTheDelegation` (an unrelated effective delegation is not listed) |
| 361 (`requests.go:67`), 529–532 (`types.go:89`) | `TestLimits` (one day; the smallest valid limits) |
| 381, 385, 389, 391, 402, 415, 424, 429 (`requests.go:101`–`137`) | every refusing `Propose` in the module tests now asserts `existing == false` and an empty view |
| 482 (`requests.go:314`) | `TestNumber` (`"12"` without the prefix) |
| 629 (`ledger.go:238`), 633 (`:247`) | `TestExactlyOneRecordOfRoomIsEnough` |
| 661 (`ledger.go:325`) | `TestCommitFaultsStopTheLedger/full write with an error` |
| 686 (`open_unix.go:32`) | `TestUnixPlatformEdges` (a named pipe is refused) |
| 699, 700 (`open_unix.go:56`) | `TestUnixPlatformEdges` (syncing a missing directory returns `fs.ErrNotExist`) |
| 772 (`record.go:143`) | `TestAnOverlongLineIsNotReadAsARecord` (a valid 4,096-byte record followed by a stray byte) |
| 780–784 (`record.go:149`) | `TestEachRecordConditionIsCheckedOnItsOwn` (one correctly summed record failing each condition alone: format, sequence, previous hash, genesis at line 2, non-UTC time, shape) |

Code changed (the mutated code no longer exists):

- 88 (`accessrequests.go:139`): the `accesschange.ErrInvalid` branch in `mapChange` was removed. `mapAccess` already maps any error that is not a gate error to `invalid_request`; `TestMapChange` still asserts that mapping.
- 95 (`:166`): `found := false` became `var found bool`. Every branch of the switch assigns it.
- 290 (`ledger.go:138`): the server-side trim of rebuilt records was removed. `accesschange.Registry.Restore` prunes to `RevocationsKept`, keeping the newest complete records, which is the same result. `TestReplayClosesDelegationsAndHonoursTheKeptLimit` still asserts 50.

Equivalent, with the argument:

- 13 (`main.go:107`, dropping `raw == ""`): `strconv.Atoi("")` always fails and the function returns `err == nil`, so `""` still gives `(0, false)`. The only reader is the flag callback, which produces the same error text.
- 78 (`accessrequests.go:106`, never returning `none`): the absent case is never stored. A proposal needs a ceiling (otherwise `invalid_request` on `action`), and a revoke target must exist (`Resolve`); in-memory grants are never deleted. So an absent value at approval or in a read always differs from the stored present value in both versions, and `current` and the superseded outcome are the same. The basis string itself is reachable only through the digest, which is computed at proposal, when the fact is present.
- 330 (`revocations.go:156`, `&&` to `||` in `verified`): the reachable states are complete with no reason and a sequence above 0 (this process or replay), and unconfirmed with a reason and sequence 0. `received` with no reason and sequence 0 exists only inside one command and is never read. Both formulas give the same value for every readable state. The only reader is the read API.
- 433 (`requests.go:162`, `<` to `<=` in the sort): the sorted values are distinct map keys, so the two comparisons agree for every pair.
- 543 (`auth/listing.go:89`, also checking per-field backing for actions other than `download`): only `view`, `view_cost` and `download` carry fields. For `view` and `view_cost`, `classView(field)` is the action itself (`validTerms` forbids cost fields on `view` and general fields on `view_cost`), and each single-field term is within the terms already backed. The only reader is `DelegationView.Effective`.
- 588, 591 (`decisionledger/ledger.go:126`, `:130`, returning `marked = true` with an error): the only caller returns at once on any error, and the flag is never read.
- 691 (`open_unix.go:32`, dropping the `f.Stat()` error check): unreachable. `Stat` on a descriptor that was just opened fails only on an `fstat` failure (EBADF or EIO), which cannot be produced in a test. With the check removed, `info` would be nil and the next condition would panic.
- 698 (`open_unix.go:48`, dropping the symlink test in `checkDir`): `checkDir` receives `Lstat` output, and a symlink's `Lstat` mode never has `IsDir`, so `!info.IsDir()` refuses it in both versions.
- 768 (`record.go:99`, `decodeLine` returning `ok = true` on a decode failure): it returns the zero `line` with `V == 0`, which `next` rejects through `l.V == FormatV1`. `decodeLine` has no other caller.

# Noted for later

- After the HYB-297 rebase, revoke refusals on installation-kind grants (including `administer_installation`) change from `prerequisite_failed` to `permission_denied`. No action in this ticket; `TestRevokeRules` and the M1 guard tests will need that change then.

# Regression runners

- M03 (`qa267.mjs`, HB-267): fails at its line 74 by design. It waits for the notice "Access granted to member@example.invalid" from the removed direct `access.grant` command. HYB-298 replaces that flow with a request ("Request sent. Another administrator must approve it.") that a second administrator approves; the replacement flow is covered by the HYB-298 runner above. The checks before line 74 ran. This failure is not reproducible on main, where the direct grant exists.
- HYB-293 (`qa293full.mjs`) on this build: group A 21 pass, 0 fail; group B 28 pass, 2 fail; group C 27 pass, 0 fail; group D 54 pass, 0 fail. Both group B failures come from its "M03 grant" family, which stubs `/commands/access.grant`, the removed command: no step-up dialog appears for it, and "every reauthentication succeeded" then counts 8 instead of 9. Group B rerun with only that family retargeted to `/commands/access.request_grant` and the new notice (`qa293on298.mjs`): 32 pass, 0 fail. These failures are not reproducible on main, where `access.grant` exists.

# Deviations from the plan

- **Last-line tamper evidence.** Every record carries an extra `sum` field (SHA-256 of the record encoded with an empty `sum`), and a line must re-encode byte for byte. The plan's field list has no self-check, but the plan also requires that "a complete but altered last line is `integrity_failed`" and that "any single-byte mutation of a non-empty file is detected"; the last line has no successor whose `prev` would catch it. A mutation of the final newline isolates the ledger with `integrity_failed` when the final line is otherwise a complete verified record (m1/N1); that record stays in the prefix and is never truncated. A mutation inside a final segment that was never a complete record reads as a torn tail and `recovered`. The property test asserts "never reads as `ok`".
- **Windows append.** Go's `O_APPEND` on Windows drops `GENERIC_WRITE`, so the torn-tail truncation would fail. On Windows the file is opened `O_RDWR` without `O_APPEND`, and every append seeks to the end first through the locked handle. Unix keeps `O_APPEND|O_NOFOLLOW`. Windows is cross-built only.
- **Replace reason.** `replace_reason` records the isolation reason as it was, so it can also be `id_mismatch`, not only `missing` or `integrity_failed`.
- **Bound at start.** The decisionledger verification bullet says "a setting below the current size plus one record refuses Start"; the body and R3-L2 say only a bound below the file size refuses and a bound within one record starts `full`. The body was followed.
- **Ledger line.** "last record …" shows the record number, because the ledger does not keep a time for the last record apart from the decision records.
- **Stopped state in API tests.** API tests reach `stopped` by closing the ledger handle, which the ledger treats as stopped. The write, short-write, sync and seek faults that stop the ledger are asserted in the module tests through the injected file interface.
- **Delegation `Effective`.** It applies the link conditions of the decision chain (closed, validity, active humans, memberships at join, delegate holds a grant) and requires the root delegator's direct backing (`backedDirectly`, plus the view-class grant for `download`), without a concrete resource. Delegation terms cannot change after creation, so the structural chain checks made at creation are not repeated.
- **Files outside the allowed list.** `server_test.go` and `readers_test.go` close the first server before starting a second one on the same data directory, because the ledger lock now refuses a second open. Four console service tests (`contracts`, `installation`, `integrations`, `recovery`) compare only `state` and `fields` of a command result, because `Outcome` gained `status` and `job_id`. `access/live-model.ts` and its test changed to list `ended` delegations and keep unconfirmed revocations visible. `access/access.css` is new.
- **M01 copy.** The refusal for an identity increase is the shared `dependency_unavailable` message. Its EN, KO and AR text now names the decision ledger as a possible cause and says suspension still works.

# Residuals from the amendment section

- m5: delegation `Effective` checks backing over the whole delegation scope, not per resource. It must be fixed before delegation creation ships.
- m6: there is no single-process lock while the ledger is missing or isolated, and the lock is released briefly during replacement.
- n1: the marker file is read without an owner or mode check.
- n2: the request digest does not include the request number.
- Windows locking, opening and durability are verified by cross-build only, and directory sync is a no-op on Windows.
- D1 is a root default listed for owner review.
- What a failed `fsync`, `F_FULLFSYNC` or `FlushFileBuffers` leaves on disk depends on the OS and file system (for example, whether dirty pages are dropped, and whether a later `fsync` reports the earlier failure). The start-up `Sync` makes the ledger `stopped` and labels the replay on any reported failure, but it cannot detect a failure the OS does not report.

# Not done or not verified

- Mutation testing was run by the coordinator (791 mutants, 76 survivors; see "Mutation testing"). The rerun of the survivors on this tree is the coordinator's.
- Windows locking, opening checks and durability: cross build only (A4, R8). Another file owner on Unix needs root: UNVERIFIED.
- Independent security change review, `ui-designer` comparison and technical-writing review were not run in this task. KO and AR copy are AI drafts pending terminology and native-language review (R6).
- `go test -race ./apps/api/cmd/hb-api`: the pre-existing `TestRun` fails intermittently under `-race` with "connection refused". `server.New` takes about 1.0 s under `-race` both on main `be1bfa3` and on this branch (measured 0.95–1.12 s), and `TestRun` waits only 50 × 20 ms for the listener. Reproduced on a temporary main worktree (1 of 3 runs failed); on this branch at load around 9–11 it failed 4 of 4. `Start` is about 40 ms slower here because of the ledger fsyncs, but it runs after the listener is open, so it cannot cause this refusal. The HYB-298 tests in that package pass under `-race`. `TestRun` was not changed.

# Second rebase (be1bfa3 to 07458be, HYB-315 merged)

Root rebased the tree onto `07458be`.

**Changes made during the rebase:**
- **`apps/console/package.json`:** the one conflict, in the test list. Both sets of tests were kept.
- **`src/service/release-evidence.test.ts:151`:** one test from HYB-315 now compares only `state` and `fields`. This is the same change this ticket applied to four other service tests, because command results now carry `status` and `job_id`.
- **`service/admin.ts`:** merged without conflict.

**Root checks on the final tree:**
- `gofmt -l`: nothing listed. `go vet ./...` in `apps/api`: clean.
- `go test ./...`: all packages pass in `apps/api` and `modules`.
- Console: `tsc` is clean, `npm test` passes 625 of 625, and the build is ok.

**Regression on the rebased tree** (implementer; a fresh `hb-api` was built from this worktree for each run):

| Runner | Results (pass/fail) |
| --- | --- |
| P05 (HYB-315) | R 481/0, F 132/0, A 17/0, B 9/0, C1 2/0, C2 3/0, C3 30/0, E 142/0 (with `--seed none`), P 12/0 |
| P04 | R 559/0, A 28/0, B 20/0, C 46/0, D 11/0 |
| P03 | R 288/0, A 23/0, B 12/0, C 24/0 |
| P02 | A 162/0, B 32/0, C 10/0, D 16/0 |
| P01 | 46/0 |
| M01 `qa264` | 51/0 |
| HYB-293 | A 21/0, B 32/0, C 27/0, D 54/0 |

- HYB-293 group B used a copy of its runner retargeted to `access.request_grant`.
- **Final HYB-298 browser phases:** A 65, B 21, C 19, D 30, E 20, a total of 155 of 155. These include the measurement of `scrollWidth - clientWidth` for both access tables at 1440 and 1024 in EN, KO and AR, which was 0 in every case.

# Reviews

All reviewers were read-only and separate from the writer. A review PASS is not owner acceptance.

**Security change review**
1. Round 1: PASS, with six minors (m1–m6), two nits and one design note (D1).
2. Round 2: PASS, with two new minors (N1: a final record without its newline was not replayed; N2: a foreign marker with a damaged line).
3. Round 3: PASS, with four nits; three were applied. All findings and root decisions are recorded in plan amendments 1–8.

**Rendered review (ui-designer)**
1. Round 1: FAIL (overlapping logins, the actions cell, Arabic clipping at 390, and minors).
2. Round 2: FAIL (the "Request revocation" button was clipped).
3. Round 3: FAIL (the column rebalance broke the requests table).
4. Round 4: PASS.

**Terminology review (tech-writer role)**
1. Round 1: FAIL, 26 findings. Six of them stated facts the server had not reported.
2. Round 2: FAIL (leftover terms and plural forms).
3. Round 3: FAIL on one Arabic dual form, which was applied exactly as proposed.

The M01 Arabic items (إلغاء used for revocation, a glued لـ, and معلّق) moved to HYB-299. The KO 결정 워터마크 rendering remains the owner's choice.

# Mutation testing (root)

Root ran the first full mutation run on four copies of the worktree, on the tree before the round-2 fixes.
- **Result:** 791 mutants, 715 killed.
- **Survivors:** the 76 survivors are classified in the mutation section above.
- **Equivalence claims:** root checked the written argument for each survivor classified as equivalent or unreachable. The kills claimed for new tests were confirmed by the implementer, who applied each mutant on its own. Root did not run the full mutation set again on the final tree.

# Planning cross-check

The plan's requirement table covers HB-03, HB-04, HB-23, HB-33, HB-36, HB-42, AUD-AR-09, AUD-DB-08 and AUD-QA-04, each against its requirement text. Root checked the built behaviour against each row.

**Covered as stated:**
- Separate propose, approve, reject and withdraw commands for M03 grant changes. Approval is human only, the proposer cannot approve or be the subject, and a basis change supersedes the request (HB-23, HB-03 for author/approver separation).
- The new commands are in the common registry and step-up. `decision_unconfirmed` is never returned or shown as completed, and the grant row reads `revocation_unconfirmed` (HB-33).
- Revocation blocks access at the next decision. It is durably recorded in the installation-local decision ledger and re-applied at start, including the verified prefix of an isolated ledger and a start where `Sync` failed (HB-42 for M03).

**Unchanged:** HB-04 and AUD-DB-08 (setup role, see HYB-301).

**Still deferred, as the table states:**
- Registry-wide leakage acceptance (HB-03).
- Support sessions (HB-36).
- Bulk paths, MFA and external identity (HB-42 elsewhere). Bulk revocation is HYB-299.
- Delegation creation and support actors (AUD-AR-09).
- OAuth and the client–user intersection (AUD-QA-04).

Nothing in this ticket changes those statuses.
