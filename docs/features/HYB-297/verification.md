---
type: VerificationRecord
title: HYB-297 verification — M02 trust change workflow
description: Rebase deltas against main d102fb4, the review fixes (security S1–S9, render R-*, copy C1–C30, QA gaps), the second review round (render, QA clipping check, security N1–N8 and F7, copy T1–T11), the checks run with their observed results, regressions per runner, mutation results, deviations, plan amendments and residuals.
status: stable
sources:
  - resource: plan.md
  - resource: visual-spec.md
---

# Scope

This record covers [the plan](plan.md) on `feat/HYB-297-trust-change-workflow`, first based on main `d102fb4`, which already includes HYB-298 and HYB-315. Root then rebased it onto main `9699fa1` (HYB-292) as the local commit "wip HYB-297"; the second-round fixes are uncommitted on top of it.
- Root wrote the Go code.
- A fork of root wrote the first console code. Root then changed it after QA and after the reviews.
- Three independent reviews ran:
  - security: PASS with minors;
  - ui-designer render: FAIL at 390;
  - technical writing: FAIL.
- Every item from those reviews is listed below with its outcome.

# Rebase deltas (plan written on `be1bfa3` plus the HYB-298 worktree)

None of these changes a behaviour or a contract the plan depends on.

1. The marker is `{ledger_id, created_at}`. `g` was added to the marker and the genesis with `omitempty`; a missing value reads as 0.
2. HYB-298 has only `missing`, `integrity_failed` and `id_mismatch`. `epoch_order` was added in ledger verification; it also covers a duplicate activation of the same change (S5).
3. HYB-298 keeps no verified prefix on `id_mismatch`. A separate `Verified()` serves only the trust replay, so HYB-298's revocation replay is unchanged.
4. An unreadable marker next to a ledger that verifies isolates as `id_mismatch`. It does not take HYB-298's ledger-without-marker path. The generation then comes from the readable genesis (`Prior.Next`).
5. The start-time `Sync` behaves as HYB-298 amendment 7 says.
6. There is no direct `access.revoke_grant`. `access.request_revoke` resolves installation-kind grants as not found; replay still reports them as protected.
7. An M03 revoke of an installation grant now gets `permission_denied` instead of `prerequisite_failed`. No console copy depended on the old refusal.
8. The command count is 70 on the base and 73 now.
9. `ledgerBlocksIncrease` is unchanged.
10. The five single-action exclusion sites use `gate.InstallationAction`, pinned by a source test.

# Review fixes

## Security review

| Item | Outcome | Tests |
| --- | --- | --- |
| S1 generation floor | Fixed. A retained file of generation g contributes g + 1 (from its genesis and its highest epoch). The current file is scanned line by line without chain verification for its highest generation. The error now reads "must be at least N, above every generation found in …". | `TestStatedGeneration` (1 and 2 refused with "at least 3" after a destroyed generation-2 ledger); `TestTheUnverifiedCurrentFileRaisesTheFloor`; `TestSuccessorDecides` (floor 1); `TestDefaultSuccessorAndFloors` (floor 10); `TestReplayFailsClosedOnACraftedLedger` (successor 7) |
| S2 ID mismatch | Fixed. With a readable marker of generation G, a record of a lower generation is not published and its changes are not restored. The seed applies, or no policy. The read reports `older_record_ignored` and the isolation notice says so. | `TestAnOlderMismatchedLedgerIsNotPublished` (marker g 3, and a copied-back older ledger); `TestIsolationPublishesTheVerifiedPrefix` (the same generation still publishes); console `trust-model.test.ts` (isolation text with the older line) |
| S3 after-activation | Fixed. The activated view shows only for `completed` with `activated`. `accepted` with a job shows the waiting view. Anything else shows a neutral "could not be confirmed; reload". The component is now `TrustAfter.tsx`. | `trust-render.test.tsx`: "the after-activation view trusts only a completed activated answer" |
| S4 re-validation | Fixed. The same evidence digest with no blocked item keeps the approval and writes a plain validated entry. A changed digest or a blocked result clears it with `approval_invalidated`. | `TestRevalidationWithUnchangedEvidenceKeepsTheApproval` (module); `TestRevalidationWithTheSameEvidenceKeepsTheApproval` (API); `TestEvidenceStale` (a changed digest clears) |
| S5 duplicate activation | Fixed. Commit refuses a second activation of a change with `commit_failed`. Verification isolates at a crafted duplicate (`epoch_order`, the prefix ends before it), and `-accept-new-decision-ledger` recovers. Start no longer fails without a way out. | `TestADuplicateActivationIsolatesAndCanBeReplaced`; `TestReplayFailsClosedOnACraftedLedger` (commit refusal) |
| S6 concurrency | Fixed. At module level, 50 rounds of two concurrent `Activate` calls each give exactly one success and one `invalid_stage`. At server level, the activator sessions are kept above the sweep; the loser's exact refusal is `409 revision_conflict`, from the executor's expected-revision check. A sequential test covers `authority_epoch_conflict` and `invalid_stage`. | `TestConcurrentActivationsOnOneEpoch`; `TestConcurrentActivationLoserGetsTheRevisionConflict`; `TestTheSecondActivationOnAnEpochIsRefusedWithAFreshRevision`. Mutants: removing supersession is killed at module and server level. Removing the `checkBound` epoch check is killed. Removing `Publish`'s strictly-newer refusal is killed by `TestPublish`; it survives at server level, where earlier checks refuse first. |
| S7 missing tests | Added. `access.request_grant` for each trust right gives `422 invalid_request [action]`. A revoke of a trust grant returns the same body as an unknown grant, byte for byte apart from the request ID. Recovery fails on its own for a membership that is not active, and for a candidate that admits nobody. | `TestM03RefusesTrustRights`; `TestRecoveryConditionsOnTheirOwn` |
| S8 holders line | Fixed. `holders.ready` is the recovery finding, and the console's `enrolled()` uses it. | `TestHoldersAreReadyOnlyWithARecoveryPath` (three validate-only holders: qualifying 3, ready false); `trust-model.test.ts` |
| S9 step-up probing | Fixed. The candidate window applies only to callers who hold `trust_validate`, so an unknown ID and an existing ID get the same answer. | `TestStepUpIgnoresTheCandidateWithoutTheValidateRight`; `TestValidateUsesTheCandidateWindow` |

## Render review

| Item | Outcome |
| --- | --- |
| R-B1 rows overlap at 390 | Fixed. Every `.m02-*` table cell has `block-size: auto` (plus `min-block-size: 0` when stacked). Headers are reset: no background, `position: static`, auto height. |
| R-B2 clipped text | Fixed with `white-space: normal` on tables in `.m02-live` and the dialog. |
| R-M1 | C3 "Validated by" now uses `when()`. |
| R-M2, R-M3 | Covered by C7 and C8. |
| R-M4 | After a refusal, the dialog drops the "Your authority", lockout and test-login rows, removes the primary button and offers "Reload and close". Render test added. |
| R-M5 | The invalidation table has a 40/20/40 colgroup. Result words do not break; `anywhere` is kept only on mono values. |
| R-M6 | Colgroups follow the spec: comparison 36/32/32 (§5.4) and findings 34/22/44 (§5.5). The fix list had the numbers swapped; the spec wins. |
| R-m1, R-m2 | Fixed. |
| R-m3 | Covered by C9. |
| R-m6 | Only the first sentence of the unavailable line is weight 600. |
| R-m7 | The C3 lockout row shows the recovery result and its reason. |
| R-m11 sidebar offset | Not reproduced in the viewport-sized layout checks (detail side and divider pass at 1440 and 1280). Not investigated separately. |
| QA gap | Added overlap and clipping checks to the K group (see Checks), dialog captures at top and bottom, and the new group L (propose and revise forms, hold condition, `activation_unknown` render with a stubbed read, loading, load error). |

QA also found a truthfulness bug: a change in `activation_unknown` showed "Activated by" under People. It now reads "Activation attempted by", with a render test.

## Copy

C1–C5 and C7–C30 are applied as written in `trust-copy.ts`, `trust-model.ts` and the components, and the glossary epoch row is updated for C16.
- **C2:** the stale sentence replaces the state line (600), and the separate stale line is removed. `fail.evidence_stale` now matches the approved-state text.
- **C9:** closed rows read "No further step." or "Propose a new change if still needed.". An activated change with invalidation still in progress reads "Check invalidation progress.".
- **C10:** the ledger line uses `counted()` and no longer shows the raw state enum.
- **C26:** applied. The progress item counts the one-time secret pickup links that the server drops for each ended session (`dropPickups`).
- **Not adopted,** as the fix list says: KO "기술 세부 정보" versus "전문가 상세", and the EN "epoch" wording.
- KO and AR remain drafts.

# Second review round

## Render

| Item | Outcome | Evidence |
| --- | --- | --- |
| B2 table overflow | Fixed. `security.css` gives `.security-table-scroll table` `min-inline-size: 430px`; the M02 tables and stacked blocks now set `min-inline-size: 0` and `max-inline-size: 100%`. | qa297 K 130/0 with the repaired clipping check |
| m1 header band and size | Fixed. The thead background and the global sticky `th` styling are neutralised under `.m02-live` and the dialog. The shell rule `#content [data-remaining-screen] th` (16px) is overridden at the same specificity, so header text is 14px. | qa297 K "validated line in ink, no header band, 14px headers" in 9 renders |
| m2 validated line muted | Fixed. "Validated at … by …" is ink; only "Not validated" is muted. | same K check |
| m7 lockout reason | Fixed. C3 and C4 show the recovery result and the reason `recovery_verified` in EN, KO and AR. | `round-two details` test in `trust-render.test.tsx` |
| N1 disclosure marker | Fixed. `summary` is `display: list-item` with 10px block padding and a 44px minimum height. | qa297 K "disclosure marker kept" |
| N2 hold sentence twice | Fixed. When the selected change is in `activation_unknown` and a hold is active, the split detail points to the notice, and the stacked view drops the hold condition from the short conditions. | qa297 L "the hold text appears once"; render test |
| N3 actor IDs split | Fixed. Actor IDs render through `actor()`: `bdi dir="ltr"`, nowrap. Digests stay wrappable, because a nowrap 64-character digest overflowed the 390 dialog. | qa297 K 390 dialog checks |
| N4 load error copy | Fixed with the given EN, KO and AR sentence. An expired session keeps the sign-in body. | qa297 L "load error" |
| N5 loading position | Fixed. The loading sentence is under the current rules heading. | qa297 L "loading state under the current rules heading" |
| Empty Reason label | Fixed. A finding with no reason renders an empty cell without `data-label`. | render test |
| Form rhythm | 24px below the propose-form lead. | screenshot |
| Captures | Added the self-refusal dialog captures (top and bottom) and a viewport-only 390 dialog capture. The 390 dialog is checked to be full screen. | qa297 K |

## QA clipping check

`overlapAndClip` in `qa297.mjs` now checks that each text-bearing element's range lies within its nearest overflow or dialog container, or within the viewport, and that no ancestor has `scrollWidth > clientWidth`. Run against the tree before the CSS fix, it failed: K 100 passed, 9 failed (scratchpad `qa297-K-before-cssfix.out`). For example, at 390 the `security-table-scroll` wrapper overflowed by 72px, and at 1280 by 59px. After the fix, K passes 130/0.

## Security

| Item | Outcome | Tests |
| --- | --- | --- |
| N1 generation bound | Fixed. `decisionledger.MaxGeneration = 2^32−1`. A genesis or trust record at or above it is not shaped, so the ledger is isolated. A marker at or above it is unusable. The unverified scan ignores such lines. The default and stated successors refuse to reach the bound with `ErrGenerations`. `-trust-generation` accepts only canonical numbers from 1 to 2^32−2. | `TestGenerationBounds` (2^32−2, 2^32−1, 2^32, MaxUint64, crafted MaxUint64 genesis); `TestStatedGenerationBounds`; `TestAccessSettingFlags` cases in `main_test.go` |
| N2 streaming scan | Fixed. `unverifiedHighest` reads through `bufio` with a `LineLimit` buffer and stops at `MaxBytes`. It counts only lines that pass `decodeLine` and `shaped`. | `TestGenerationBounds`: an over-long line, the byte limit, an over-long final line, a trust record above the genesis |
| N3 stricter of ignored record and default | Fixed. Per field: the shorter session limit; MFA if either requires it; sign-in methods are the intersection, or the record's methods when the intersection is empty; and per action the shorter step-up limit, with MFA if either requires it. Step-up limits are capped at the session limit. The read reports `trust.stricter_fields`. | `TestTheIgnoredOlderRecordStillTightensTheDefault`; `TestAnOlderMismatchedLedgerIsNotPublished` now expects the stricter 1 h |
| N4 unclear button | Fixed: "Sign in again". | render test "after-activation view" |
| N5 enrolment with three people | Fixed. `holders.missing` lists the rights without a qualifying holder. With `ready: false` the notice shows the `missingRoles` sentence. | render test "enrolment block"; `TestHoldersAreReadyOnlyWithARecoveryPath` checks `missing` |
| N6 kept approval | Fixed. A same-digest re-validation records `revalidated_approval_kept`, labelled in EN, KO and AR. | `activate_test.go`; render test |
| N7 step-up user | Fixed. `stepUpCheck` takes the user from `call.Answer.User()`; the route path passes its known user. | `stepup_test.go` |
| N8 dead write | Removed; this record's amendment wording is corrected. | — |
| F7 membership not activated | Added a standalone test: an invited, not yet activated administrator does not qualify. | `TestPendingAdministratorDoesNotQualify` |

## Copy

T1–T11 were applied as written in the fix list. T1: `qualifying()` counts people who can use at least one trust right, which matches the new label. T2: `evidence_stale` shows the stale-validation text for approve and the current text for activate. T4: the glossary gains "Blocked" and "Activation attempted by". T9: `records()` reuses `units.records` from the M03 copy, now exported.

# Plan amendments

Root added A1–A6 to plan.md as "Root amendments after the change reviews (2026-11-18)". After the second round, root revised A1 and added A7 (generation bound) and A8 (the kept-approval history entry). Still to add to the plan text:
- A1 for an installation without the sample seed: the ignored record's configuration applies at G<<32+1, number 1, compared against a base with no limits. `stricter_fields` and `older_record_ignored` are reported the same way.
- A7: a genesis at or above the bound in the current ledger isolates it, because the record is not shaped. The bound is not ignored there. `-trust-generation` accepts 1 to 2^32−2.
- A8: the entry is `revalidated_approval_kept`.
- `holders.missing` lists the trust rights that no qualifying person holds.
- The T1–T11 EN copy wherever plan.md quotes the old text.

# Final round (root's round-3 notes)

Root re-verified the round-2 tree, and it passed: vet, `go test`, 100% coverage on the four packages, tsc, `npm test` 648/648 and the build.

| Item | Outcome | Test |
| --- | --- | --- |
| N3 gap without the sample seed | Fixed. Before, an ignored older record left no policy, so administrators had no session-age limit. Now `stricter()` is applied against a base with no limits: no methods, MFA none and the maximum duration. It publishes the record's configuration at G<<32+1, number 1, and reports `stricter_fields` and `older_record_ignored`. `stricter()` itself is unchanged. A base with a zero session limit would have kept 0 as the limit, so the base uses the maximum duration. | `TestWithoutTheSampleSeedTheIgnoredRecordStillApplies` |
| T3 | AR `fail.approval_stale` is now "عُدِّل التغيير بعد تحميله. أعد التحميل وراجع التعديل الحالي." | — |
| Optional copy | AR `holdSeeNotice` adopted. The history label `revalidated_approval_kept` is now "Validated again, approval kept" / "재검증, 승인 유지" / "أُعيد التحقق، والاعتماد قائم". | render test "round-two details" |

Final-round checks, all run:
- gofmt clean; vet clean with GOOS=linux, darwin and windows.
- `go test` ok with 100.0% coverage on decisionledger, authpolicy, trustchange and apps/api/server; `-race` ok on apps/api/server (314 s).
- tsc clean; `npm test` 648/648; build ok.
- `qa297`: A 17/0, B 3/0, C 3/0, D 3/0, E 2/0, I 3/0, K 130/0, L 21/0, F1 2/0, F 2/0, G 3/0, H 2/0.
- Mutation on the changed lines (`trustledger.go:238`, `244`, `245`): 4 mutants, all killed. Line 245 has no operator to mutate. Its maximum-duration base is pinned by the test's `max_age` field expectation.

# Checks (second-round tree, observed)

| Check | Result |
| --- | --- |
| gofmt | clean |
| go vet | clean with GOOS=linux, darwin and windows over `./...` |
| `go test ./...` | all packages ok. Once, `TestTimingIndistinguishable` failed under parallel package load (locked 101 ms against 56 ms). It then passed with `-count=5` alone and in a full rerun of the package. It is outside this change. |
| `go test -race` | ok on decisionledger, trustchange, authpolicy, apps/api/server (363 s) and cmd/hb-api |
| Coverage | 100.0% on authpolicy, decisionledger, trustchange and apps/api/server |
| Console | tsc clean; `npm test` 648/648; `npm run build` ok |
| Browser QA `qa297` | A 17/0, B 3/0, C 3/0, D 3/0, E 2/0, I 3/0, K 130/0, L 21/0, F1 2/0, F 2/0, G 3/0, H 2/0 |

H first failed 1 of 2: the runner still expected the old enrolment sentence. With the runner updated to T1, it passed 2/0.

# Checks (first-round tree, observed)

| Check | Result |
| --- | --- |
| gofmt | clean |
| go vet | clean; also clean with GOOS=linux, windows and darwin over `./modules/...` and `./apps/api/...` |
| `go test ./...` | all packages ok |
| `go test -race` | ok on decisionledger, trustchange, authpolicy and apps/api/server (335 s). Before the fixes it was also ok on auth, gate, commands and hb-api, and with `-count=40` on the concurrency tests. |
| Coverage | 100.0% on authpolicy, decisionledger, trustchange and apps/api/server. In auth, gate and commands the uncovered functions are the same ones as on the base. |
| Cross builds | linux/amd64, windows/amd64, darwin/arm64 and linux/386 ok. This run was before the fixes; the fixes add no platform code. |
| Console | tsc clean; `npm test` 647/647; `npm run build` ok |
| Browser QA `qa297` (final build) | 168 passed, 0 failed |

`qa297` per group:

| Group | Pass/fail |
| --- | --- |
| A | 17/0 |
| B | 3/0 |
| C | 3/0 |
| D | 3/0 |
| E | 2/0 |
| I | 3/0 |
| K | 109/0 |
| L | 19/0 |
| F1 | 2/0 |
| F | 2/0 |
| G | 3/0 |
| H | 2/0 |

- K covers EN, KO and AR at 1440, 1280 and 390. Every list, detail and dialog render was checked for page `scrollWidth`, overlapping cells and clipped text.
- The data directories were created with mode 0700 and deleted afterwards.
- Screenshots: scratchpad `qa297shots` (69 files) with an `INDEX.md`.

# Regressions (second-round tree)

| Runner | Result |
| --- | --- |
| M01 `qa264on297` | 51/0 |
| HYB-298 `qa298on297` | A 67/0, B 21/0, C 19/0, D 30/0, E 20/0 |
| HYB-293 `qa293on297` | B 32/0, C 27/0, D 54/0, A 21/0 |
| P01 `qa283full` | 46/0 |
| P02 `qa285fullon297` | A 162/0, B 32/0, C 10/0, D 16/0 |
| P03 `qa287fullon297` | R 288/0, A 23/0, B 12/0, C 24/0 |
| P04 `qa289fullon297` | R 559/0, A 28/0, B 20/0, C 46/0, D 11/0 |
| P05 `qa315bodyon297` | R 481, F 132, A 17, B 9, C1 2, C2 3, C3 30, P 12, E 142, every group exit 0 |

- The first pass used a stale driver. HYB-293, P01 and P02 A could not start, because port 8820 was in use at that moment. P02–P05 ran the pre-`on297` runners. Both were rerun with the right runners.
- HYB-293 B then failed 1 check: "M02 validate stubbed refusal then live resend" saw 403,0. The runner's done-selector also matched the muted "Not validated" line, so it stopped before the resend completed. With the selector narrowed to a non-muted result, B passed 32/0.
- No HTTP 429 appeared in any runner after HYB-292's 1 s refill.

# Regressions (first-round tree, hb-api built from it)

| Runner | Result |
| --- | --- |
| M01 `qa264on297` | 51/0 |
| HYB-298 `qa298on297` | A 67/0, B 21/0, C 19/0, D 30/0, E 20/0 |
| HYB-293 `qa293on297` | B 32/0, C 27/0, D 54/0, A 21/0 |
| P01 `qa283full` | 46/0 |
| P02 `qa285fullon297` | A 162/0, B 32/0, C 10/0, D 16/0 |
| P03 `qa287fullon297` | R 288/0, A 23/0, B 12/0, C 24/0 |
| P04 `qa289fullon297` | R 559/0, A 28/0, B 20/0, C 46/0, D 11/0 |
| P05 `qa315bodyon297` | R 481/0, F 132/0, A 17/0, B 9/0, C1 2/0, C2 3/0, C3 30/0, P 12/0, E 142/0 (`--seed none`) |

Before these runner changes, every failure came from a runner assumption; none was a product bug:
- `qa264`: two count checks failed.
- `qa298` group A: "no eligible approver" failed.
- HYB-293: B had 3 failures, C 2, D 1, caused by the removed `policy.*` screen and by `member` being read by position.
- P02 D, P03 C and P05 C3: "member denied" failed, because the third link (now admin3) was signed in as "member".
- P04: R had 2 failures, A, B and C 1 each, caused by the positional `member` and by admin3 counting as an eligible approver.

These are the runner-only changes, made in scratchpad copies:
- **M01 `qa264` → `qa264on297`:** expects 4 users, since there is now a third sample administrator.
- **HYB-298 `qa298` → `qa298on297`:**
  - Group A suspends sample-admin-3 before the "no eligible approver" request and reinstates it after, so the original case is still exercised.
  - Phase E passes `--accept-trust-reset`.
- **HYB-293 `qa293full` → `qa293on297`:**
  - Group B drives the M02 trust screen: propose through C1, then validate through C2 under a stubbed step-up.
  - Group D reads the AR digits from the trust propose form.
  - The M03 step is retargeted to `access.request_grant`, as in HYB-298.
- **P02–P05 and HYB-293 runners:** they took `member` as the third activation-link line, which is now admin3. The `on297` copies look logins up by name.
- **P04 copy:** suspends sample-admin-3 at start, so its "fewer than two administrators besides the placer" cases are still exercised.

# Mutation

- **First run** (tree before the review fixes): 294 mutants. 12 survivors were killed by new tests; each was confirmed by applying the mutant alone.
- **Final run** (final tree, new code included): 532 operator mutants.
  - Files: `modules/trustchange`, `authpolicy/publish.go`, `decisionledger/trust.go` and `successor.go`, and in `apps/api/server`: `trust.go`, `trusthold.go`, `stepup.go`, `trustevidence.go`, `trustledger.go`, `trustrows.go`.
  - Outcome: 513 killed, 8 rejected by the compiler or vet (contradictory conditions), 11 survived.
- **Six survivors were killed afterwards** by tests added after the snapshot; each was confirmed by applying the mutant alone:
  - `successor.go` with `found` starting as `true`: killed by a file with no genesis or trust lines.
  - `trust.go:95` `||`→`&&`: killed by an identity session that has no console session.
  - `trust.go:149` `>`→`>=`: killed by a rationale exactly at the limit, which must be accepted.
  - Three `Required` flags: killed by `TestTrustFieldsAreRequired`.
- **Four survivors are equivalent.** For each, the input that would tell the mutant apart cannot occur:
  - `registry.go:97` (`>`→`>=` on the history cap): at exactly 20 entries the slice keeps the same 20.
  - `successor.go:107` (sort `<`→`<=`): retained file names are unique.
  - `trust.go:73` (`||`→`&&`): the executor's integer form only lets through values such as `-0` or `-1`, which fail `ParseUint` and also differ from `FormatUint`, so both conditions are true together.
  - `trust.go:282` (`!ok || CanPublish` → `&&`): an exhausted generation yields epoch 0, which `CanPublish` already refuses.
- **One survivor is not killed:** `trust.go:98`, `Fits` with `<=`→`<`. Only a record that encodes to exactly 2048 bytes would tell it apart, and no test builds one. Low risk, because `Commit` applies its own `> RecordLimit` check.

## Second round

- 59 operator mutants on the lines changed in the second round, with `+ 1`, `min`/`max` and `!` added to the operators: 56 killed, 3 survived.
- `successor.go:84` (`max`→`min` on trust records in the unverified scan): killed by a new case, a trust record above the genesis generation. Confirmed by applying the mutant alone.
- `trustledger.go:138` (`!known`→`known`): the later generic bound check hid it. Killed by asserting the specific `trust-generation must be below` error. Confirmed by applying the mutant alone.
- `main.go:109` (first `||`→`&&`) is equivalent. A failed `ParseUint` returns 0 or MaxUint64, which `n == 0` or `n >= MaxGeneration` already refuses.

# Deviations

- admin3 holds `administer_installation`, so it can read M02 and activate a change. Go tests that assumed only one other holder use a new `withoutThirdAdmin()` helper.
- `commands.Definition.Refusals` declares the new refusal codes per command.
- Fields added to the API:
  - `holders.members` and `holders.ready`;
  - `trust.older_record_ignored`;
  - `epoch_generation` and `epoch_number` in the activate result.
- `epoch_exhausted` is a new validation blocking reason.
- The set-link race test checks that the link is unconsumed and still usable once the hold ends, rather than after a restart, because links are in memory.
- The hold notice is paragraphs, not lists.
- The confirmation dialog relies on the native modal for its focus trap.

# Residuals and not done

- Owner questions Q1–Q7 are still open. KO and AR copy are drafts for native review.
- Not run in a browser:
  - QA J, password confirmation past the window;
  - axe;
  - the step-up dialog stacked on a confirmation;
  - a real `activation_unknown` (only stubbed render and API tests).
- R-m11 was not investigated separately.
- No HTTP 429 was seen in any runner after the 1 s refill (HYB-292).
- When an older record is ignored, its configuration still applies, with or without the sample seed. A crafted record that requires MFA, or has an MFA step-up rule, can therefore lock administrators out until the operator replaces the isolated ledger. Crafting such a record requires write access to the data directory.

# Planning cross-check (root, 2026-11-18)

The requirement table in plan.md ("Requirement cross-check": M02 spec lines 24-26 and 96-131, HB-03, HB-36, HB-42, IAM-01 to IAM-08, retention-access.md 41-117, HYB-291 finding M2) was re-checked against the merged behaviour after all review rounds. Status per row is unchanged: covered for trust changes (HB-03, IAM-05, IAM-06 within pre-launch, IAM-08 for password sessions), not connected (IAM-02 to IAM-04, MFA), deferred (IAM-07), regression only (IAM-01). HYB-291 M2 (propose-then-approve collapsing the trust flow) is closed: draft, validated, independently approved and activated are separate states held by three distinct people.

# Root final verification (2026-11-18, rebased on main 867a3e9)

- Rebase onto 867a3e9 (HYB-296) conflicted only in `apps/console/package.json` (test list; both sets kept) and `docs/ui/terminology/product-glossary.md` (both sections kept).
- `gofmt` clean, `go vet ./...` ok, `go test ./...` ok, coverage 100.0% on `apps/api/server`, `modules/trustchange`, `modules/decisionledger`, `modules/authpolicy`; `tsc` clean; `npm test` 658/658; build ok.
- Reviews: security change review rounds 1-3 (PASS; last gap N3 fixed in the final round); render review rounds 1-3 (PASS; last minor N3, the validator ID split at 390, fixed by root with a render test "the validated line keeps the validator ID in one isolated unit"); technical-writing review rounds 1-3 (PASS after T3).
- Open from the render review: the "Reload and close" button in B-self-refusal-en-1440-bottom shows a light fill, most likely the programmatic focus after the refusal; not changed.
- Owner questions Q1-Q7 remain open with their defaults; KO and AR copy remain drafts for native review.

