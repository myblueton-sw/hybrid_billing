---
type: Verification
title: HYB-307 verification record, M05 end other sessions against a fixed session manifest
description: Commands run and results observed for the HYB-307 implementation and its review round 1 and round 2 fixes, on the branch at base 113c0d5, uncommitted. Only observed numbers are recorded; checks that were not run are listed as not run.
status: stable
tags: [HYB-307, verification, security, identity, sessions, M05, IAM-07]
sources:
  - resource: docs/features/HYB-307/plan.md
  - resource: docs/features/HYB-307/visual-spec.md
---

# Scope

The work verified here is HYB-307 in the worktree `/private/tmp/hb307`: branch `feat/HYB-307-m05-end-other-sessions`, base main `113c0d5`, not committed. It follows [plan.md](plan.md), which is stable after security design review round 2, and [visual-spec.md](visual-spec.md) §16, including root's item 13.

**Reviews so far**
- Security change review round 1: PASS with 4 Minor findings, all applied below.
- Render and copy review round 1: FAIL. Every finding was applied below.
- A re-review of this round has not been done.
- TW review, native-language review and owner acceptance have not been done.

**Machine and environment**
- mac_mini.
- Go commands ran with `GOCACHE` set to the session scratchpad (`gocache307`). The mutation scripts cleaned it every 10 mutants.

# What changed

**Server (`apps/api/server`)**
- **`session.go`, `sessionend.go`.** The session store now has the following:
  - A per-login internal counter and an opaque 128-bit token. Both change on every put, drop and prune.
  - `snapshot` and `endManifest`, each running in one `sessions.mu` critical section.
  - A denied-verifier set indexed by login, capped at `maxSessions`, with a per-login overflow marker and a sweep at the 12-hour session lifetime.
- **`selfend.go`.** `POST /api/v1/me/sessions/end` uses the manifest with hold semantics:
  - The deny calls module `EndSession` under the domain lock. A failed end goes to `s.denied`.
  - Retries back off at 1, 4 and 16 s, for 4 attempts in total.
  - The latest 5 operations are kept per user.
  - The answer is 202.
- **`commands.go`.** `access` checks `s.denied` first. After the module answers, it checks `s.denied` and store presence again. The test seam `accessHook` now receives the verifier.
- **Eviction and probe.** `openSession` and `probe` settle denied verifiers first. Sign-in answers 503 while the login is overflowed.
- **`self.go`.** `GET /me` adds `sessions_generation` and `session_ends`.
- **`bulkcleanup.go`.** The worker tick drains the retries.
- **`openapiself.go`.** `x-self` entries for the two routes.

**Console**
- `service/self-end.ts` parses the new read and answer fields. `admin.ts` gets `endOthers` and `endOne`.
- M05 `my-security/`:
  - `EndOthers.tsx`: the modal dialog for both variants, and the "Recent session ends" region;
  - `end-copy.ts`: the copy;
  - `end.css`: the scoped styles;
  - `Live.tsx`: the wiring, the held tag, the list-changed notice and polling;
  - `live-model.ts`: helpers;
  - `live-copy.ts`: superseded strings removed, and the revoked body uses the spec `:88` KO wording.

**Docs**
- Glossary: rows for `session-access-blocked`, `session-cleanup`, `session-not-included`, `session-list-time` and, this round, `secret-pickup`.
- The M05 field map rows and the M05 register line.

## Review round 1 fixes

### Security change review (PASS, 4 Minor)

| # | Fix | Proof that the new check fails first |
| --- | --- | --- |
| C1 | `openapiself.go` gives every status its own entry, with `x-code`, `x-codes`, `x-next-action` and `x-retry`. POST: 401, 403 (`reauthentication_required` with next action `reauthenticate`; codes `csrf_refused` and `reauthentication_required`), 413 `body_too_large`, 422 (`invalid_request` and `prerequisite_failed`), 500 `internal_error`, 503 `authority_unavailable` (`retry_later`). GET `/me`: 401, 403 `csrf_refused`, 500, 503. `TestOpenAPISelfPaths` checks every entry and the response count | `prove307.py` rows 1–2: the old 403 entry, and the missing 500 and 503 on POST, each make the test FAIL (`prove307.txt`) |
| C2 | Root decision: the `s.denied` release in `endOne` is restored, because it is reachable. `execute` calls `endOne(caller)` when the caller's verifier is no longer in the store (`commands.go:225-227`). `endOne` now ends the module record through `endModule` and calls `settleEnded`: a final success (`done` or `already_ended`) settles the pending retry and releases the guard; anything else leaves the retry pending. The earlier entry in this record calling the release unreachable was wrong. Tests drive `execute` with a denied verifier: `TestEndOneSettlesADeniedVerifier`, `TestEndOneKeepsAPendingRetryWhileTheModuleIsDown`, `TestEndOneUnexpectedErrorLeavesTheRetryPending`, `TestEndOneSettlesAnAlreadyClosedRecord` | `prove307.py` rows 3–4: an `endOne` without the release, and the first draft's unconditional release, each FAIL |
| C3 | When the denied set is full, no retry entry is created. C3 reads `failed`, reason `denied_set_full`, with copy in EN, KO and AR. Test: `TestDeniedOverflowCreatesNoRetry` | `prove307.py` row 5: the pre-fix retry on overflow FAILS |
| C4 | `TestSelfEndInFlightRequestsAreRefused` holds a request at `accessHook`, after the module answered, on the `readers.go` and `boundary.go` paths. The deny runs while it is held; on release the request gets 401, and the session is not put back. The earlier after-deny HTTP checks were relabelled as stale-cookie checks (`TestSelfEndStaleCookiePerPath`) | `prove307.py` row 6: without the recheck after the module answer, the test FAILS |

All six proof rows read `FAILS_AS_EXPECTED` and `restored=True`.

### Render and copy review (FAIL)

**Layout**
- **L1.** The counts strip no longer has a "Sessions" item. It is a grid of 4 columns when the dd is at least 380px wide (container query), else 2×2. QA §11.10 follows root's §16 item 13.
- **L2.** In the dialog, the Reference column is `12.5rem`. Signed in and Ends by render as two nowrap parts, date then time, using separate `Intl` calls (`datePart`, `timePart`).
- **L3.** `dl.mysec-op-lines` has no measure cap.
- **L4.** The held tag is always on its own line.
- **L5.** "Change your password" is styled as a link: no border or background, underlined, 44px hit area.
- **L6.** The operation reference is 14px and muted.
- **L7.** Confirmed correct; no change. In the AR 1440 render the older operation was open because its `not_included` list still held sessions that were present. Each earlier language iteration leaves its own session signed in. That meets the §5.6 rule.

**Copy**
- **C1–C9** replaced exactly as given, in EN, KO and AR:
  - no 회수 in the outside-scope wording;
  - count-neutral Arabic for the held, announcement, already-gone and not-finished lines;
  - no "لـ" before digits;
  - en-US "recognize", and a singular form for one session;
  - KO and AR result titles that differ from the entry button;
  - 미완료;
  - آخر عمليات إنهاء الجلسات;
  - أُغلقت مسبقًا;
  - "Change your password" / 비밀번호 변경하기 / تغيير كلمة المرور.
- **C10.** C4 KO is now "이 세션에 발급된 비밀값 수령 링크", with a new glossary row `secret-pickup`.
- **C11.** `already_closed` KO is now "로그인 서비스에서 이미 종료된 기록입니다.".
- **Two further changes made during QA:**
  - The polling-stopped announcement now formats its time. It used to announce the raw ISO value.
  - After a cancelled step-up, focus now lands on the dialog's failure summary (§9.2), not the primary button. Check S3c below.
- **Copy test.** The new test in `live-model.test.ts` was run against each of the 9 old strings, restored one at a time. Every run failed as expected, and the file was restored each time.

# Commands and observed results (final pass, after all fixes)

| Command | Result |
| --- | --- |
| `gofmt -l apps/api` | No files |
| `go vet ./...` ×3 | Clean, 3 runs |
| `go test ./...` | 73 packages `ok`, 0 failures; `apps/api/server` `ok` in 125.114 s |
| `go test -race -count=5 -run 'SelfEnd\|EndOne\|DeniedOverflow\|DeniedSet\|AccessRechecks\|OtherSessionEnders\|SignInEviction\|SessionTokens\|SessionListOrder\|OpenAPISelf' ./apps/api/server` | `ok` in 26.852 s |
| `go test -race -timeout 60m ./apps/api/server` | `ok` in 1178.684 s; 0 `DATA RACE` reports |
| `npx tsc --noEmit` | Clean |
| `npm test` | 776 tests, 776 pass, 0 fail |
| `npm run build` | Built |

**Earlier passes this round, recorded as observed.** These ran before the last `settleEnded` simplification:
- `go vet` ×3 clean;
- `go test ./...` with 73 packages `ok`;
- the selfend race tests ×5 `ok`;
- the full-package race run `ok` in 1190.581 s with 0 `DATA RACE` reports.

**The first round** also recorded a full race run `ok` in 1279.953 s. An earlier run was terminated by a signal after 1105 s and is not counted.

# Mutation testing

**Round 1 (40 mutants).** 38 killed. The two rows that did not end killed:
- #17, `access` without the first `s.denied` check, **survives**. The recheck after the module answer refuses the same verifier, and the two orders differ only when the identity registry is unbuilt, which cannot happen after Start.
- #40, removing the `endOne` release, is superseded by C2: the release is restored and now killed (round 2, row 1).

**Rounds 2 to 4 (newly changed lines).** Results are in `mutation307-round2.txt`, `-round3.txt` and `-round4.txt` in the scratchpad.
- **Round 2: 12 mutants, 9 killed.** The 3 survivors were in `settleEnded`. Two of them guarded unreachable branches: a pending early return that changed nothing, and a release for a denied verifier without a retry entry. `deny` always creates a retry entry, and the overflow path adds no denied entry. Those branches were removed. The third survivor was killed by the new `TestEndOneUnexpectedErrorLeavesTheRetryPending`.
- **Round 3: 5 mutants on the simplified code, 3 killed.**
- **Round 4.** "Accepts already ended" was killed by the new `TestEndOneSettlesAnAlreadyClosedRecord`. "Release without a retry" survives. It is equivalent: no denied entry exists without a retry entry.
- **Covered mutants in rounds 2 to 4, all killed in their final form:**
  - `endOne` settles;
  - `settleDenied` settles;
  - no retry on overflow;
  - overflow reads failed;
  - the hook receives the verifier;
  - the 403 next action;
  - the 413 entry;
  - the GET 403 entry;
  - `x-codes` include the alternatives;
  - only final success settles;
  - already ended settles.
- Every file was restored and checked by SHA-256 after every mutant.

# Browser QA

**Setup.** Each run used a fresh `hb-api` data directory, the scratchpad proxy `qa298proxy.mjs`, and Playwright 1.56.1 with headless Chromium. Ports were 4541–4547 and 8871–8877.

| Runner | Build | Result |
| --- | --- | --- |
| `qa307.mjs` (Q1–Q7, plus R1–R7 and S1–S5 added this round: 92 checks) | base `113c0d5` | 0 pass, **92 fail**: every check fails on base |
| `qa307.mjs` | branch, before the last `settleEnded` simplification | **92 pass**, 0 fail |
| `qa307.mjs` | branch, final binary | **92 pass**, 0 fail |
| `qa271on307.mjs` (M05 regression) | branch | 62 pass, 0 fail |
| `qa293on307.mjs` (HYB-293 step-up regression) | branch | Groups B 32/0, C 27/0, D 54/0, A 21/0 (pass/fail) |
| `qa271.mjs` and `qa293on297.mjs` | base (first round) | 62/0; B 32/0, C 27/0, D 54/0, A 21/0 |

## Checks added this round

- **R1.** The counts strip has four items in one row at 1440 in EN, KO and AR, and is 2×2 at 390.
- **R2.** The manifest Reference column is 200px, and every date and time part renders on one line.
- **R3.** The two lines end at the same right edge as the operation body.
- **R4.** The held tag is a block.
- **R5.** "Change your password" has no border and is underlined.
- **R6.** The operation reference is 14px.
- **R7.** In each language at 1440 and 390, a sweep finds no radius, no shadow, no border wider than 1px and no horizontal scroll.
- **S1.** Count = 0 (real: the listed sessions were ended individually before the confirmation) shows its own sentence and "Nothing to clean up".
- **S2.** A real server replay (the same POST sent twice by the route) shows the replay line.
- **S3.** Stacked step-up dialog:
  - a. the step-up dialog stacks as a second `:modal` over `dialog.mysec-end`;
  - b. Escape closes only the step-up dialog;
  - c. focus lands on the dialog's failure summary;
  - d. a later confirmation completes the end.
- **S4.** With `/me` stubbed to stay `cleanup_running`, polling stops after 30 s with "Check again", and makes no reads in the next 5 s. "Check again" makes one read (2 requests observed, including the stub's fetch).
- **S5.** `cleanup_incomplete` in KO at 390 shows 미완료 and the failed reason.

## Focus moves and announcements (observed in the branch run, `EVIDENCE` lines in `qa307-branch.out`)

- **Dialog open:** focus is on `h2#mysec-end-title`.
- **After a 202:** focus is on `h4#mysec-op-session-end-N`. The live region reads, for example, "Access is blocked for 2 sessions and cleanup is complete. 1 session started after your list and is still signed in." The KO and AR runs gave the corresponding sentences, for example "تم حجب الوصول عن الجلسات (3) واكتمل التنظيف. عدد الجلسات التي بدأت بعد قائمتك ولا تزال مسجّلة الدخول: 1.".
- **Replay:** "This is the result of your earlier request. Nothing new was ended. Access is blocked for 2 sessions and cleanup is complete."
- **Step-up stacked:** the open dialogs are `mysec-end:modal` and `service-reauth:modal`, with focus on `input#reauth-password`. After Escape, focus is on `div.service-summary`.
- **Polling stopped:** after 17 reads. The visible line and the live region both read "Cleanup was still in progress at Nov 19, 2026, 2:24:32 AM.".
- **`scrollWidth − clientWidth`:** 0 in EN, KO and AR at 1440 and at 390. The sweep found 0 issues in each.

## Screenshots

34 files in `qa307shots-branch/` in the session scratchpad. They cover:
- the dialog and result in EN, KO and AR at 1440 and 390;
- 2× crops at 390 of the dialog and the held result in EN, KO and AR, and of the incomplete result in KO;
- the held result at 1440 and 390;
- count 0, replay, list changed after a 422, unknown result with "Reload and close", step-up stacked over the dialog, polling stopped with "Check again", and `cleanup_running` in KO.

# Review round 2 fixes

**Reviews.** Security change review: PASS, with one advisory. Render and copy review: FAIL on copy only; the layout passed.

| # | Fix | Proof that the new check fails first |
| --- | --- | --- |
| Advisory | The POST 403 entry lists `csrf_refused` and `reauthentication_required`, which need different next actions. Every error entry now carries `x-next-actions`, a map from each code to its own next action: `csrf_refused` → none, `reauthentication_required` → `reauthenticate`, `prerequisite_failed` → `correct_request`. `x-next-action` keeps the primary code's action. `TestOpenAPISelfPaths` checks that every code listed in `x-codes` has a next action, and checks the 403 and 422 maps | `prove307-round2.txt` row 1: the round-1 entry fails |
| F1 | The `denied_set_full` reason now uses the reviewer's EN, KO and AR text. It no longer says sign-ins "wait until it clears". It says they are refused for 12 hours after the stop, and that the list is the service's. "Access stays blocked" is proved by `TestSelfEndInFlightIsRefusedWhenTheDeniedSetIsFull`: with the denied set full and the module end failing, the verifier is neither denied nor ended in the module, yet a request held at `accessHook` on the `readers.go` and `boundary.go` paths is refused with 401 by the store-presence recheck | `prove307-round2.txt` row 2: without the presence recheck the test fails. The copy is pinned in `live-model.test.ts`, which fails on the old string |
| F2 | The failure summary in the dialog has a title for each outcome. The unknown result is "Result not confirmed" / "결과 미확인" / "لم تُؤكَّد النتيجة". Rate-limited, cancelled or refused step-up is "Nothing was ended" / "종료하지 않았습니다" / "لم يُنهَ أي شيء". A failed read keeps the load-failed title | Unit test fails on the old title. Browser Q5 and S3 titles fail on the round-1 build |
| F3 | When count = 0, the "Access blocked" value reads "None: every session in your list had already ended", with the given KO and AR | Unit test fails on the old value. Browser S1 fails on the round-1 build |
| F4 | `stateSentence` and the `StateSentence` component check count = 0 first, so `completeNone` is used in every state. `stateSentence` moved to `end-copy.ts` so the unit test can import it. The polling-stopped evidence was re-shot with consistent fixture data: rows have C3 `pending`, and count-0 operations stay `complete`. New checks: S4 "running rows read cleanup in progress" and S4 "count-0 operations never show the in-progress sentence" | The unit test fails when the guard comes after the running branch. The two S4 checks pass on the round-1 build as well, because they verify the corrected fixture rather than changed code |
| F5 | `TestEndedSessionsCarryAllEightCleanupItems` checks that the server sends C1–C8, in order, for every ended session, and no items for an already-gone session. The KO incomplete fixture now has all eight items; S5 checks that the expanded row lists 8. The KO 390 2× crop was re-shot | `prove307-round2.txt` row 3: removing C8 from the server answer fails. S5 also passes on the round-1 build, because it checks the fixture |
| F6 | An already-gone session shows "Nothing to clean up: this session had already ended." inline, with no disclosure. The unreachable branch in `Items` was removed | Browser S1 (inline sentence, no `details`) fails on the round-1 build |

All three `prove307-round2.txt` rows read `FAILS_AS_EXPECTED` and `restored=True`. The 4 console reversals (F1 EN, F4 guard order, F2 title, F3 value) each failed as expected, and the file was restored each time.

## Round 2 results

| Command | Result |
| --- | --- |
| `gofmt -l apps/api` | No files |
| `go vet ./...` | Clean |
| `go test ./apps/api/...` | 2 packages `ok` |
| `go test -race -count=5 -run 'SelfEnd\|EndOne\|DeniedOverflow\|DeniedSet\|AccessRechecks\|OtherSessionEnders\|SignInEviction\|SessionTokens\|SessionListOrder\|OpenAPISelf\|EndedSessionsCarry' ./apps/api/server` | `ok` in 24.421 s |
| `npx tsc --noEmit` | Clean |
| `npm test` | 777 tests, 777 pass, 0 fail |
| `npm run build` | Built |
| `qa307.mjs` (99 checks) on the round-1 build (saved `dist307r1`, `hb-api307r1`) | 95 pass, 4 fail. The 4 failures are the new code checks: Q5 title, S1 value, S1 inline and S3 title. The other 3 new checks verify fixtures and pass on both builds |
| `qa307.mjs` on the round-2 build | **99 pass**, 0 fail |
| `qa271on307.mjs` | 62 pass, 0 fail |
| `qa293on307.mjs` (run on ports 4547 and 8877, after the proxy incident below) | Groups B 32/0, C 27/0, D 54/0, A 21/0 (pass/fail) |

**Proxy incident.** During the first round-2 HYB-293 run, the HYB-323 implementer stopped this run's proxy on port 4546 by mistake and restarted it. That run, which passed B 32/0, C 27/0, D 54/0 and A 21/0, is not counted. The restarted proxy was stopped, and the run was repeated on other ports.

**Re-shot evidence** in `qa307shots-r2/`:
- `M05_polling_stopped_en_1440.png`: every row reads "Access blocked · cleanup in progress", and the count-0 operation is collapsed with "Nothing to clean up";
- `M05_incomplete_ko_390_2x_crop.png`: 완료 (C1, C2, C4, C5), 미완료 (C3), 해당 없음 (C8) and 통제 범위 밖 (C6, C7);
- `M05_count0_en_1440.png`.

The full-repository `go test ./...` and the full-package race run were not repeated this round. The Go changes this round were limited to the OpenAPI entries and to tests.

# Residuals and not run

- **Sessions table dates (L2, pre-existing).** The main sessions table still breaks dates inside the value ("Nov 19, 2026, 1:15 / AM"). Recorded as a residual with no change, by root decision.
- **Mutants #17 and round-4 "release without a retry"** survive, as equivalent in reachable states (see Mutation testing).
- **Reviews.** No re-review of the round-2 changes has been done. The KO and AR strings and all [TW] strings are AI drafts.
- **Screen reader.** Behaviour beyond the live-region text and focus targets was not tested.
- **Field map line numbers.** Rows in `screen-binding-fields/identity.md` that this ticket did not touch still carry pre-change line numbers.

## Root final verification (2026-11-19, rebased on main 9f66b78)

- Reviews: plan security review rounds 1–2 (APPROVE with N1–N3 folded in); security change review round 1 PASS (C1–C4) → round 2 PASS; render and copy review round 1 FAIL → round 2 FAIL (copy) → round 3 PASS.
- Rebase conflicts with HYB-323 in `apps/api/server/openapi.go` (kept HYB-323's `x-reads` and this ticket's `x-self`) and `docs/ui/screen-binding-fields/identity.md` (kept HYB-323's unread-fields line and this ticket's parser line); `bulkcleanup.go` merged without conflict.
- Root re-ran: gofmt clean; `go vet ./...` clean; `go test -short ./...` all packages ok; `-race -count=3` on the session, denied, end, bulk and cleanup tests ok; server coverage 100.0%; console tsc clean, npm test 792/792, build ok; OKF status test ok; `qa307.mjs` on the rebased build: no failures, the one aborted request is the expected Q5 case.
