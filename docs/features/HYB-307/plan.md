---
type: ImplementationPlan
title: M05 end other sessions against a fixed session manifest with a session generation check, access blocked before cleanup
description: Plan for HYB-307. "End all other sessions" on M05 ends exactly the confirmed sessions that are still the caller's own, holds out and reports sessions that started after the list (with an End action) and reports listed sessions already gone, ends the module session inside the deny with a denied-verifier guard, cleans up afterwards with per-session results, and shows blocking apart from cleanup in EN, KO and AR. No decision-ledger record, because sessions never survive a restart.
status: stable
tags: [HYB-307, security, identity, sessions, M05, IAM-07, HB-42, HB-03]
sources:
  - resource: docs/reviews/HYB-301-trace-classification/report.md
  - resource: docs/planning/v1.16/retention-access.md
  - resource: docs/planning/v1.16/requirements.md
  - resource: docs/planning/v1.16/ui-design.md
  - resource: docs/ui/screens/M05/spec.md
  - resource: docs/ui/screen-binding-fields/identity.md
  - resource: docs/ui/screen-binding-register.md
  - resource: docs/features/HB-270/plan.md
  - resource: docs/features/HB-271/plan.md
  - resource: docs/features/HYB-293/plan.md
  - resource: docs/features/HYB-298/plan.md
  - resource: docs/features/HYB-299/plan.md
---

# Scope and ownership

Linear ticket HYB-307 (team HYB): "IAM-07 / M05 spec: end-others must work against a fixed session manifest with a generation check and show blocking apart from cleanup. Implement." It comes from row 4 of "Required and missing" in the [HYB-301 trace classification](../../reviews/HYB-301-trace-classification/report.md) (`report.md:319`). That row says the behaviour was not recorded before.

- **Base.** main `113c0d5`. **Branch.** `feat/HYB-307-m05-end-other-sessions`. **Machine.** `mac_mini`.
- **Owner direction of 2026-11-17.** Build the spec behaviour as written. Where a step depends on something missing, block or show the state truthfully and record the dependency. Never simplify silently. Provisional values ship as installation defaults that can be changed.
- **Ownership.**
  - Root (Claude Code) owns the plan, the implementation, the tests and the integration.
  - A separate read-only reviewer (`independent-reviewer`) does the security design review of this plan, and later the change review. The design review is required because this changes session revocation (permissions) and the public `/api/v1/me` contract.
  - `ui-designer` (read-only) gives visual direction for the changed M05 sessions view and its states, then reviews the rendered screens.
  - A tech-writer review (read-only) covers the new copy and the glossary rows.
  - QA writes and runs the browser runner.
- **Allowed files** (implementation):
  - `apps/api/server`: `session.go`, `self.go`, a new `selfend.go`, one call added in `bulkcleanup.go` (`drainCleanup`), the `s.denied` check in `commands.go` (`access`), denied-first eviction in `session.go` and `users.go` (`probe`), `openapi.go`, and their tests (`self_test.go`, `stepup_test.go`, `reauth_test.go`, new `selfend_test.go`, `selfend_race_test.go`, `selfend_property_test.go`).
  - `apps/console/src/service/admin.ts` (the `SelfPage` type, the `me` parser and `endSessions`) and `apps/console/src/service/admin.test.ts` (F8).
  - `apps/console/src/modules/identity/pages/my-security/`: `Live.tsx`, `live-model.ts`, `live-copy.ts`, `live-model.test.ts`, and a new `EndOthers.tsx` if `Live.tsx` would grow too much.
  - Docs: `docs/features/HYB-307/*`, the M05 rows in `docs/ui/screen-binding-fields/identity.md`, the M05 entry in `docs/ui/screen-binding-register.md:205`, and new rows in `docs/ui/terminology/product-glossary.md`.
- **Not in scope:**
  - The single-session "End" flow keeps its contract, except for the shared deny-then-cleanup path described below.
  - Password change keeps ending every other session unconditionally (`self.go:288`).
  - Recent access history is HYB-306.
  - Server search on M05 is HYB-294 row 1, and not this ticket.

# Requirement cross-check

## Verbatim sources

| ID | Source line | Verbatim |
| --- | --- | --- |
| IAM-07 | `docs/planning/v1.16/retention-access.md:116` | `\| IAM-07 \| 10만 사용자 조회·초대·동기화·회수의 격리·재개·차단 revision 검증 \|` |
| M05 action | `docs/ui/screens/M05/spec.md:72` | `\| 다른 세션 종료 \| 본인 재인증·세션 종료 권한 \| 본인 다른 세션의 권한 효력을 우선 차단하고 정리 진행 \| 선택 수/현재세션 포함 여부 확인; 효력 차단과 정리 완료 분리 \|` |
| M05 bulk | `docs/ui/screens/M05/spec.md:95` | `10만 사용자 전체를 불러오지 않고 본인 UserID로 범위를 고정한다. 본인 세션·최근 접근이 많으면 cursor+기간 필터를 사용한다. '다른 세션 전체'는 고정 대상 manifest와 최신 세션 generation 검사 후 회수하며 회수 효력과 청크 정리를 구분한다. 알림 설정 저장은 버전 조건을 가진다.` |
| M05 field | `docs/ui/screens/M05/spec.md:51` | `\| 현재 세션·기기 \| session-list \| 예 \| 본인 종료만 \| 현재세션 표시·발급/최근사용/만료/회수효력; 상세 위치는 최소화 \| 개인 \|` |
| M05 interaction | `docs/ui/screens/M05/spec.md:77` | `입력 오류는 상단 요약과 필드를 연결하고 첫 오류로 초점을 이동한다. 반복 클릭은 같은 요청으로 처리한다. 성공 토스트만으로 업무 완료를 표시하지 않고 결과 객체/작업 상태를 보여준다. 재시도는 멱등성과 현재 상태를 확인한 허용 명령만 제공한다.` |
| M05 partial | `docs/ui/screens/M05/spec.md:85` | `\| partial \| 접근 차단은 적용되었지만 일부 세션 정리가 진행 중입니다. \|` |
| M05 error | `docs/ui/screens/M05/spec.md:87` | `\| error \| 변경 결과를 확인하지 못했습니다. 다시 제출하기 전에 현재 연결·세션 상태를 확인해 주세요. \|` |
| M05 revoked | `docs/ui/screens/M05/spec.md:88` | `\| revoked \| 현재 세션의 접근이 종료되었습니다. 다시 로그인해 주세요. \|` |
| M05 reauth | `docs/ui/screens/M05/spec.md:89` | `\| reauth-required \| 이 변경을 계속하려면 현재 계정으로 다시 인증해 주세요. \|` |
| M05 states | `docs/ui/screens/M05/spec.md:91` | `권한 회수 시 보이는 민감값·이전 미리보기·선택 캐시를 제거하고 같은 데이터 재다운로드를 막는다. 오류에는 비민감 요청 참조와 안전한 다음 조치만 포함한다. 0·없음·미집계·알 수 없음은 별개이며 미확인 값을 0으로 채우지 않는다.` |
| M05 keyboard | `docs/ui/screens/M05/spec.md:101` | `- 키보드: 비밀번호 관리자·붙여넣기 지원; 해제/세션 종료 대화상자에서 대상 이름과 영향 읽기; 성공 후 해당 수단 행/상태 메시지로 초점.` |
| Revocation order | `docs/planning/v1.16/retention-access.md:59` | `정지/권한 회수 시 authoritative deny/version을 먼저 갱신하고 세션·refresh token·API 키·권한 캐시·대기 작업에 반영한다. …` (first sentence quoted) |
| HB-42 bulk | `docs/planning/v1.16/requirements.md:757` | `**대량 처리:** 사용자·멤버십·감사는 서버 필터와 커서 조회한다. 대량 초대/회수는 고정 scope·청크·재개를 사용하고 보안 차단 버전을 먼저 활성화해 배치 지연 중 기존 접근을 허용하지 않는다.` |
| UI plan | `docs/planning/v1.16/ui-design.md:154` | `\| M05 내 보안·로그인 연결 \| 본인 \| 로그인수단·본인 소속·MFA·세션·최근 접근, 비밀번호 변경/허용identity 연결·해제/다른 세션 종료 \| …` (first three cells quoted) |

## Cross-check table

| Clause | What it requires | Today (`113c0d5`) | This plan |
| --- | --- | --- | --- |
| `:95` 고정 대상 manifest | The sessions to end are fixed before the end runs | Ends whatever sessions exist when the request arrives (`self.go:122-131`) | The confirmed handles are the manifest. The client sends them with the list-time token, and the server classifies and removes them in one critical section and never extends the set (§2, §4) |
| `:95` 최신 세션 generation 검사 | Refuse when the sessions changed since the manifest | No session generation exists | A per-login session generation, shown as an opaque token, that changes on every add or remove. A change is reported, not refused (hold, F3): only confirmed sessions are ended, newcomers are `not_included` and gone ones `already_gone` (§1, §2) |
| `:95` 회수 효력과 청크 정리를 구분 | Revocation effect and cleanup are separate | Both happen together inside `endOne` (`self.go:116-120`) | Deny = removal from the session store and module `EndSession` under the lock before the answer, with `s.denied` guarding a failed module end. Cleanup = module retries, pickups and attempt counters, with per-session results (§4, §5) |
| `:72` 권한 효력을 우선 차단하고 정리 진행 | Block first, then clean | Same as above | Same as above. The answer is 202 with state `cleanup_running`, or `complete` |
| `:72` 선택 수/현재세션 포함 여부 확인 | The confirmation shows the count and whether the current session is included | Shows the count only (`Live.tsx:190`, `live-copy.ts:175-180`) | The confirmation lists the fixed sessions (reference, signed in) and says "This session is not included and stays signed in" (§9) |
| `:72` 본인 재인증 | Own reauthentication | Step-up on the route (`stepup.go:32`, `self.go:138`) | Unchanged (HYB-293), and it also applies to a replay (§6) |
| `:72` 효력 차단과 정리 완료 분리 (screen) | Blocking and cleanup completion are shown apart | One "Ended N" notice (`live-copy.ts:183-191`) | Two lines, "Access blocked" and "Cleanup", with per-session results and spec copy `:85` while cleanup is not complete (§9) |
| `:85` partial | Copy for blocked-but-cleaning | Not reachable | Shown while `cleanup_running`, and as the base sentence of `cleanup_incomplete` |
| `:87` error | An unknown result is never treated as success | "unknown result" copy exists (`Live.tsx:178`) | Kept. The step reloads the list. A resubmit with the same manifest is an idempotent replay (§6) |
| `:88` revoked | The ended session sees the revoked copy | Existing signed-out view (`Live.tsx:285-295`) | Unchanged; tested from the ended browser |
| `:77` 반복 클릭은 같은 요청 | Repeated clicks are the same request | Busy guard only; a server-side repeat ends any new sessions | Busy guard, plus a server replay keyed by the manifest digest. A repeat never ends a session outside the confirmed list (§6) |
| `:77` 결과 객체/작업 상태 | Show the result object, not a toast | Toast-like notice | An operation row (`session_ends[]` in `GET /me`) is shown until complete (§8, §9) |
| `:91` 0·없음·알 수 없음 별개 | Unknown is never filled as zero or done | — | Cleanup items are `pending`, `done`, `already_ended` or `failed`; nothing unconfirmed reads as done |
| `:101` 대상 이름과 영향 읽기; 상태 메시지로 초점 | The dialog reads its targets and impact; focus moves to the status | Impact sentence only | The list is read in the dialog; focus moves to the operation status (§9) |
| IAM-07 회수 … 차단 revision 검증 | The blocking revision is checked | No revision | The answer and the read carry `list_generation`, `current_generation` and `generation_changed`, and tests verify them against the live token (§2, §4, Tests) |
| IAM-07 격리·재개 | Isolation and resumption of the revocation | — | Isolation: the deny touches only the caller's own login. Resumption: module-end retries with backoff and a budget, under the `s.denied` guard, and a restart ends every session (§5, §7). The 100k-user scale aspect is scoped to the caller's own UserID (`:95`) |
| HB-42 `:757` 보안 차단 버전을 먼저 활성화 | Block before any batch delay | — | The deny and the generation bump happen before the answer and before any cleanup (§4) |
| `retention-access.md:59` deny/version first | The authoritative deny and version first | — | Same as above |

# Current behaviour (facts from code at `113c0d5`)

- **Session store.** `apps/api/server/session.go:36-41` keeps an in-memory map from verifier to session (`login`, `created`, `authenticated`, CSRF, `trustEpoch`). It has no per-login version. `put` (`:78-88`), `drop` (`:135-139`) and `prune` (`:53-61`) change it under `sessions.mu`.
- **Opening sessions.** Only `signIn` opens sessions (`signin.go:253-264`), and it holds `s.domain.Lock()`. Eviction of the oldest session at the module limit happens inside `openSession` (`session.go:154-162`).
- **Removing sessions.** Sessions are removed by:
  - `endOne` (`self.go:116-120`): module `EndSession`, then `sessions.drop`, then the session's pickups are dropped;
  - logout (`signin.go:299-304`);
  - `access` on stale, unfresh or refused sessions (`commands.go:139-158`);
  - the bulk cleanup (`bulkcleanup.go:146-156`);
  - administrator paths (`users.go:345`, `:359`) and trust activation (`trust.go:326`);
  - step-up failures (`reauth.go:25`, `:63`) and the `commands.go:217` path;
  - pruning at sign-in (`signin.go:263`).
- **The module side.** `modules/identity` holds its own session record with a serial (`boundary.go:52-99`) and limits live sessions per user to `SessionsPerUser` 10 (`server.go:154`, `boundary.go:85`). `current` (`registry.go:264-271`) treats idle (30 minutes) and absolute (12 hours) expiry as not live.
- **The read.** `GET /api/v1/me` (`self.go:92-114`) lists the caller's sessions with a 16-hex `handle` (`self.go:51-54`), `current`, `signed_in` and `ends`. It has no generation.
- **End others.** `POST /api/v1/me/sessions/end` (`self.go:133-168`) runs under `s.domain.Lock()`:
  - admission (`self.go:56-73`), then step-up (`self.go:138`), then the body `{session}` (`self.go:141`);
  - `"others"` calls `endOthers` (`self.go:122-131`), which ends **every** session of the login other than the current one at that moment, including sessions opened after the person looked at the list;
  - each end is `endOne`, so the module end, the store removal and the pickup removal happen together;
  - the answer is 200 `{ended: n}` (`self.go:167`) with no result object, generation or manifest;
  - a repeat ends anything new, so it is not idempotent.
- **The console.** `EndStep` (`my-security/Live.tsx:157-197`) shows the impact sentence with a count (`:190`, `live-copy.ts:175-180`), calls `api.endSessions("others")` (`Live.tsx:172`, `admin.ts:998-1001`) and shows "Ended N" (`live-copy.ts:183-191`). `endOutcome` (`live-model.ts:41-49`) maps errors.
- **Tests that send the old body:** `stepup_test.go:163`, `:171`, `:375`, `:474`; `reauth_test.go:338`; `self_test.go:251-295`. Browser runners `qa271body.mjs:102` and the HYB-293 runners (`qa293body.mjs:263`, `qa293on297.mjs:307`, `:400`, `qa293on298.mjs:305`, `:398`) also drive it.
- **Not in the OpenAPI document.** The `/me` routes are not in `Document()` (`openapi.go:14-39`). Only commands and `x-reads` for `/identity/users` are documented.

# Design

Revised after security design review round 1 (see Review history). The findings are cited as F1–F11.

## §1 Session generation (per login)

- **Internal counter.** `sessions` gains `seq uint64` and `gen map[string]uint64`, both guarded by `sessions.mu`.
  - Every change to the set of a login's sessions sets `gen[login] = ++seq`: `put`, and `drop` or `prune` of an entry that existed. One deny of a manifest sets it once.
  - When a login has no sessions left, its entry is deleted. A later session gets a fresh `++seq`, so a value never repeats for a login, and the map stays bounded by the logins that have sessions.
  - **What does not change it:** a reauthentication (`reauthenticated`, `session.go:90-99`), reads, and module-only idle expiry. Idle expiry leaves the store entry until `access` drops it, and that drop changes the generation.
- **Opaque value shown to the client (F4).**
  - `token map[string][16]byte` sits next to `gen`. It is a fresh random 128-bit value (`crypto/rand`) every time `gen[login]` changes.
  - `GET /me` exposes only the token, as 32 lowercase hex characters, in `sessions_generation`.
  - The client can only compare it for equality. It reveals neither the global `seq` nor how much activity other logins had.
  - The internal counter is kept for ordering, tests and the log line.
- **Atomic snapshot (F2).** `sessions.snapshot(login) (token, counter, []session)` returns the token, the counter and the login's sessions in one hold of `sessions.mu`. It replaces the separate `byLogin` and `get` calls in `ownSessions` (`self.go:75-90`) for `GET /me` and for the end. The list and the generation shown therefore always belong together.
- **Why per login.** The identity module's registry generation changes for every identity event in the installation. A check against it would treat some other person's suspension as a change to the caller's sessions. The spec scopes the check to the person's own sessions (`:95` 본인 UserID로 범위를 고정).
- **Alternative considered: the module serial.** Rejected. It counts opens only, so a removal would not show, and apps/api does not keep it.
- **Login and user hypothesis.** Sessions are keyed by login (`session.go:32`), and the operation is kept per internal user ID. The plan assumes one login per user and one user per login, which `LoginTriple` and the directory give today. Residual R7.

## §2 Fixed session manifest (hold semantics, F3)

- **What is fixed.** The handles the person confirmed in the dialog. The console re-reads `GET /me` when the confirmation step opens and sends exactly the handles it listed, together with the list's `sessions_generation`.
- **What is ended.** Each confirmed handle that is still one of the caller's own other sessions at the deny. The set is **never extended**. A session that is not in the confirmed list is never ended by this request.
- **What is reported.**
  - **Ended:** confirmed handles that were still present.
  - **`already_gone`:** confirmed handles that are gone by the deny. They are counted as such and never as done by this operation. This is the per-session result; `already_ended` is reserved for the C3 cleanup state (N3).
  - **`not_included`:** the caller's other sessions that are present at the deny but were not confirmed, which means they appeared after the list was taken. Each is reported with its handle and signed-in time, and an "End" action. The action is the single-session end, which fixes its target by handle.
  - **`generation_changed`:** a boolean that is true when the list-time token differs from the token before the deny. The answer carries both values, `list_generation` and `current_generation` (the token after the deny).
- **This follows the HYB-299 precedent.** There, a target changed since the preview is *held* and reported, never processed silently (HYB-299 execute step 5). Here, the confirmed list is processed, the newcomers are held out and shown, and nothing outside the confirmed list is touched. Root decision on F3. It replaces the first draft's refusal.
- **The current session** is never in a manifest. A current handle in `handles` is a 422.
- **Manifest fields** (server-side, in the operation record):
  - `operation`: `session-end-<n>`, from a server counter;
  - `user`: the internal user ID;
  - `kind`: `others` or `one`;
  - `list_generation`: the token from the request;
  - `current_generation`: the token after the deny;
  - per session: the verifier (never serialised), `handle`, `signed_in` and `result`: `ended`, `already_gone` or `not_included`;
  - `count`: the number ended;
  - `digest`: SHA-256 under the label `self_session_end_v1` over the kind, `list_generation` and the sorted confirmed handles, in a canonical encoding (the pattern of `modules/approvals/digest.go`).
- **Single session.** `{session: "<handle>"}` keeps its contract: the handle alone fixes the target. It goes through the same deny, operation and cleanup path, with a manifest of one. An unknown handle is 422 `invalid_request`, as today.

## §3 Request contract and order of checks

`POST /api/v1/me/sessions/end`, end others:

```json
{"session": "others", "generation": "<32 hex>", "handles": ["<16 hex>", "..."]}
```

- **Decoding.** A typed decoder replaces `decodeStrings` for this route:
  - `DisallowUnknownFields`, and the existing body cap with 413;
  - 1 to 64 handles, each 16 lowercase hex characters, with no duplicates;
  - `generation` is 32 lowercase hex characters.
  - The manifest-free form `{session: "others"}` is refused with 422.
- **Order**, all under `s.domain.Lock()`. The first failure answers, and a failure changes no session, operation or counter.
  1. Admission: cookie, CSRF and `s.access` (`self.go:56-73`).
  2. Step-up for route `sessions/end` (`self.go:138`, HYB-293).
  3. Decode and validate the body. A failure is 422 `invalid_request`, fields `generation` or `handles`.
  4. **Replay.** If the latest operation of this user has the same digest, return it (§6).
  5. **Deny** through one call, `sessions.endManifest(login, current, handles)` (§4). It returns the per-handle classification. The token compare, the classification and the removal happen in one critical section (F2).
- **A handle that is not the caller's.** A handle that never was one of the caller's own sessions is indistinguishable from one that has ended. Both are reported as `already_gone`, so nothing about other people is revealed and no 422 is needed (F7). The 422 cases are only structural (step 3) and the current handle.
- **The 422 on end-others reloads (F7).** If a 422 with field `handles` or `generation` ever reaches the console, for example from a stale client build, it reloads `GET /me` and shows the list-changed state rather than a generic error. A browser test covers it.

## §4 Deny step (authoritative block, F1 and F2)

The deny runs inside the domain write lock, after §3 step 4.

1. **`sessions.endManifest(login, current, handles)`.** It holds `sessions.mu` once and does all of this:
   - it reads the login's sessions and token;
   - it classifies each confirmed handle as present (to end) or absent (`already_gone`), and each unconfirmed other session as `not_included`;
   - it removes the present ones from the map;
   - it sets the generation once if anything was removed;
   - it returns the classification, the list-time match and the new token.

   **When it returns `false`.** It returns `false` only when the caller's own session is no longer in the map, for example because a lock-free path dropped it between admission and here. Then the answer is 401 `unauthenticated`, nothing is removed, no operation is created and the counter is not changed (F2). A changed generation is *not* a failure. It is reported (§2).
2. **The module session is ended inside the deny (F1, root decision).** For each removed verifier, still under the domain lock:
   - `identity.Boundary().EndSession(v)` is called;
   - on success, or on `ReasonSession`, the module refuses that verifier from now on, so every authority check that consults the module refuses it at once;
   - on `ErrUnavailable` or any other error, the verifier goes into `s.denied`, a set guarded by `sessions.mu` (§5). Entries are keyed by verifier and indexed by login, so eviction and the probe can find a login's denied verifiers. The set has a hard cap of `maxSessions` (`session.go:20`). When it is full, the deny still removes the session, and further sign-ins for the affected login fail closed with `dependency_unavailable` until the sweep frees entries. A denied verifier is never silently dropped from the set (N2).
3. **`s.access` checks `s.denied` first** (`commands.go:138`). A verifier in it is refused with `unauthenticated`, whatever the module answers. **After the module answers success**, `access` checks `s.denied` and store presence again (`sessions.get` of the verifier) before admitting, so a deny that lands between the first check and the module answer is still refused (N1). The entry is removed only when a retried module end succeeds or returns `ReasonSession`, or when the worker-tick sweep finds that `blocked_at + 12h` (the absolute lifetime) has passed (N2).
4. The operation is recorded with state `cleanup_running` and `blocked_at` (server clock). The current generation token is the one returned by step 1.
5. Then:
   - `s.counter++` when anything was ended;
   - the log line `sessions_ended` with user, operation, ended count, already-ended count, not-included count, internal counters before and after, and `generation_changed`. No handles, verifiers, tokens or logins are logged.
6. The remaining cleanup (§5) is queued, and the worker is woken through `s.wake` (`bulkcleanup.go:79`).
7. **Answer: 202**, carrying the operation object (§8). It is `accepted` because cleanup is still running. When nothing was ended (every confirmed handle already gone), the answer is 202 with state `complete` and `count: 0`, and the not-included list is still returned.

**Paths that look up a session without the domain lock (F1).** Each resolves the cookie through `s.lookup` (`session.go:171-181`), then calls `s.access`, which consults `s.denied` and the module:

| Path | Line | After the deny |
| --- | --- | --- |
| Proxied and served reads | `readers.go:112` | `lookup` fails after the removal. A request that resolved before the removal reaches `access`, where the module refuses (or `s.denied` does) |
| Current-session read | `boundary.go:119` | Same |
| Logout | `signin.go:293` | `lookup` fails, so it only clears the cookie. A request that resolved before it then takes the domain lock, and `EndSession` and `drop` are no-ops |
| Command entry | `commands.go:174` | `lookup` fails. A request that resolved before it is rechecked under the lock (`commands.go:215-218`), finds no session and is refused |

- **Mid-stream reads.** A proxied read that passed `access` before the deny and is still streaming its answer is **outside scope**. It finishes. This is the same classification as HYB-299 B8 (`bulkcleanup.go:38`, `in_flight_until_restart`). It is listed as cleanup item C6.
- **A pending-window test per path.** With the worker in manual mode and the module end forced to fail (so the verifier is only in `s.denied`), each of the four paths is driven with an ended cookie. Each must refuse it.

**The current session.** It is never in the manifest and never touched. An end of others whose own session another device has just ended fails admission with 401, and the console shows the revoked copy (`:88`).

**Pickups after the deny.** A pickup is bound to a session verifier, and redeeming it needs that session to resolve. Once the deny has run, the pickup cannot be redeemed, even before cleanup deletes it. A test checks this.

## §5 Cleanup (after the answer)

- **Items per ended session.** The per-session disclosure shows every row, including the outside-scope ones (F10).

| Item | Action | Result |
| --- | --- | --- |
| C1 Access by the session cookie | Removed from the store in the deny (§4) | `done`, `deny_applied` |
| C2 Step-up evidence | Lives on the store entry | `done` with C1 |
| C3 Module session record | `EndSession` in the deny. Retried only if that failed | `done` on success or `ReasonSession` (`already_ended` when the deny itself got `ReasonSession`); `pending` while retrying; `failed` otherwise |
| C4 Secret pickups issued to the session | `dropPickups(p.session == v)` | `done` (cannot be redeemed from the deny on) |
| C5 Password attempt counter | `delete(s.failures, v)` | `done` |
| C6 Requests in flight at the deny | — | `outside_scope`: mid-stream reads finish (HYB-299 B8). Commands are rechecked under the lock |
| C7 Pages already shown in that browser | — | `outside_scope`: cannot be recalled. That browser shows the revoked copy on its next request |
| C8 Jobs and API keys | — | `not_applicable`: no job runs under a session, and people hold no API keys |

- **The C3 retry (F6).** The verifiers in `s.denied` form an in-memory queue on `Server`. `drainCleanup` (`bulkcleanup.go:84-96`) also calls `drainSelfEnds`, under the domain lock, so the retry runs on the existing worker tick. Tests drain it through the manual-mode seam.
  - **Backoff:** a retry runs no earlier than 1 s, then 4 s, then 16 s after the previous attempt. That is 4 attempts in total, including the one in the deny, all within about 30 s.
  - **Outcomes:** success or `ReasonSession` gives `done`. `ErrUnavailable` waits for the next attempt. **Any other error gives `failed` at once**, reason `unexpected_error`, and is never mapped to done. After the last attempt the item is `failed`, reason `retry_budget_spent`.
  - **A failed item stays in `s.denied`** until the worker-tick sweep removes it at `blocked_at + 12h` (N2). That lifetime is a cap on the module record, so the session cannot become usable before the record itself expires.
- **Eviction and probe (F5).**
  - With F1, the module record is normally gone at the deny, so it no longer counts towards the 10-session limit.
  - For a verifier still in `s.denied`, eviction in `openSession` (`session.go:154-162`) and `probe` (`users.go:350-362`) first try `EndSession` on the caller login's denied verifiers before choosing the oldest live one. A success there completes that C3.
  - Tests: sign-in at the limit with a denied, unended module record; and the probe likewise.
- **Operation states.**

```
received ──deny──► cleanup_running ──every C3 done/already_ended──► complete
                         │
                         └──any C3 failed──► cleanup_incomplete
```

- `received` exists only inside the request.
- There is no failed operation state. A refused request creates no operation, and a cleanup failure never undoes the deny.
- **Counts invariant**, at every read: `count = done + already_ended + failed + pending`, where `count` is the number of sessions actually ended by this operation (result `ended`), and `already_ended` is the C3 state only (N3). `not_included` and confirmed-but-absent handles are listed apart and are not in `count`.
- **"Complete" never implies recall (F10).** The result shows C6 and C7 as outside scope next to "cleanup is complete".

## §6 Idempotency and repeated requests

- **Client.** The busy guard (`Live.tsx:167`) stays.
- **Server.** Each user keeps the latest 5 operations (constant). A request whose digest equals the latest operation's digest gets that operation back, with `replay: true`, the same 202 status, and no new deny.
- A replay still passes admission and step-up first (as in `TestReplayNeedsStepUp`, `stepup_test.go:244`).
- A repeat after the first answer has a matching digest, so it is a replay and ends nothing. A repeat that misses the replay window (an older operation) re-runs §4. Every confirmed handle is then gone, so it reports them as `already_gone` and ends nothing new, because the set is never extended.
- **After an unknown result** (`:87`), the console reloads `GET /me` and shows the latest operation. It never resubmits on its own.

## §7 Durability and the decision ledger

**Decision: no decision-ledger record for session ends.**

- **The ledger's purpose.** It makes a deny survive a restart by replaying it before the API opens (HYB-298 `plan.md:533-556`, HYB-299 Durability).
- **Sessions never survive a restart.** The store is in memory (`session.go:36-41`), and the identity module's session records are rebuilt empty by the seed. After a restart no ended session can resolve, so there is nothing to replay.
- **The owner rule.** "Required ledgers persist in the installation data directory" (2026-11-17) applies to a ledger that a requirement needs. IAM-07 and M05 `:95` need an ordered deny with a generation check, not a durable record of it.
- **Never refused because of the ledger.** In `isolated`, `stopped`, `full` and in the operator-recovery state (`commands.go:220`), ending one's own sessions keeps working. These tests are labelled **regression guards**: today's route is not a command and does not consult the ledger, so they pin that it never starts to.
- **Audit.** The structured log line only, as HB-270 decided. Owner question Q2.
- **Trigger to revisit.** If sessions are ever persisted across restarts, this decision is wrong (residual R1).

## §8 API and OpenAPI changes

- **`GET /api/v1/me`** adds:
  - `sessions_generation` (opaque 32-hex token, F4);
  - `session_ends[]`, newest first, at most 5, each with:
    - `operation`, `kind`, `count`;
    - `state` (`cleanup_running`, `complete`, `cleanup_incomplete`);
    - `blocked_at`, `list_generation`, `current_generation`, `generation_changed`, `digest`;
    - `cleanup`: `{done, already_ended, failed, pending}`;
    - `sessions[]`: `{handle, signed_in, result: ended|already_gone, items: [{item, state, reason}]}`;
    - `not_included[]`: `{handle, signed_in}`.
- **`POST /api/v1/me/sessions/end`:**
  - the new body for `others` (§3);
  - the answer changes from 200 `{ended}` to 202 with the same operation object, plus `replay`;
  - a single handle answers the same object with `kind: "one"`.
- **Codes.** All exist already; none is added.

| Condition | Status | Code | Next action |
| --- | --- | --- | --- |
| No or unknown session, or the caller's own session is gone at the deny | 401 | `unauthenticated` | answer |
| CSRF mismatch | 403 | `csrf_refused` (`CodeCSRF`) | none |
| Access refused (suspension, policy age) | as `s.access` | its code | as mapped |
| Step-up needed | 403 | `reauthentication_required` | reauthenticate |
| Step-up rule needs MFA | 422 | `prerequisite_failed` | correct |
| Body over the cap | 413 | `body_too_large` | none |
| Malformed body, missing or badly formed generation or handles, duplicates, more than 64 or none, the current handle, unknown single handle | 422 | `invalid_request` (fields where they apply) | correct |
| Accepted, held parts reported, or replayed | 202 | — | — |

- `revision_conflict` is no longer used by this route; a changed generation is reported, not refused (F3).
- **OpenAPI.** `Document()` gains `x-self` entries for `GET /me` and `POST /me/sessions/end`: the body schema, the answer schema, `x-step-up: true` on the POST, and `x-codes`. This follows the HYB-294 `x-reads` pattern (`openapi.go:38`) and is pinned by a golden test. It is a public contract change, covered by the independent change review.

## §9 Console (M05 sessions view)

- **Opening the confirmation** for "End all other sessions" (`Live.tsx:429`) re-reads `GET /me` and freezes that page for the dialog. The dialog shows:
  - the title (existing `endAllTitle`);
  - the list of the confirmed sessions: reference (`ltr(handle)`) and signed-in time, read in order by screen readers;
  - "List taken at {time}", from the read's `now`;
  - the sentence "This session is not included and stays signed in.";
  - the button "End {n} other sessions".
- **Result object.**
  - Lines: "Access blocked: {time}", and "Cleanup: {done} of {count} done, {pending} in progress, {failed} not finished".
  - A per-session disclosure lists C1–C8 with the outside-scope rows (F10).
  - When `generation_changed` is true or `not_included` is non-empty, a held block lists each new session with an "End" button (the single-session flow) and the changed copy.
  - It polls `GET /me` every 2 s for up to 30 s while `cleanup_running`, then shows the last read with "Check again". No fake progress.
  - Focus moves to the operation status (`:101`).
- **A 422 on end-others** reloads the list and shows the changed copy (F7).
- **Copy.** EN is wording for TW review. KO follows the spec where the spec has it. KO and AR are AI drafts pending native-language review. The AR term for the deny follows the glossary's حجب (`product-glossary.md:69`, "رقم إصدار الحجب"); TW confirms it (F9).

| State | EN | KO | AR (draft) |
| --- | --- | --- | --- |
| Confirmation, scope | This session is not included and stays signed in. | 이 세션은 포함되지 않으며 로그인 상태가 유지됩니다. | هذه الجلسة غير مشمولة وتبقى مسجّلة الدخول. |
| Confirmation, list time | List taken at {time} | 목록 기준 시각 {time} | وقت أخذ القائمة {time} |
| Button | End {n} other sessions | 다른 세션 {n}개 종료 | إنهاء {n} من الجلسات الأخرى (AR plural forms) |
| Held: new sessions | Only the sessions in your list were ended. {k} sessions started after the list was taken and are still signed in. Review them below and end any you do not recognise. If you did not sign in elsewhere, change your password to sign out everywhere else. | 목록에 있던 세션만 종료했습니다. 목록을 가져온 뒤 시작된 세션 {k}개는 로그인 상태가 유지됩니다. 아래에서 확인하고 모르는 세션은 종료하세요. 다른 곳에서 로그인한 적이 없다면 비밀번호를 변경해 다른 모든 곳에서 로그아웃하세요. | لم تُنهَ إلا الجلسات الموجودة في قائمتك. بدأت {k} جلسة بعد أخذ القائمة ولا تزال مسجّلة الدخول. راجعها أدناه وأنهِ أي جلسة لا تعرفها. إن لم تسجّل الدخول من مكان آخر، فغيّر كلمة المرور لتسجيل الخروج من كل مكان آخر. |
| Held: already gone | {j} sessions in your list had already ended. ({j} counts the per-session result `already_gone`.) | 목록의 세션 {j}개는 이미 종료되어 있었습니다. | كانت {j} جلسة من قائمتك قد انتهت مسبقًا. |
| Blocked, cleanup running (spec `:85`) | Access is blocked, but cleanup of some sessions is still in progress. | 접근 차단은 적용되었지만 일부 세션 정리가 진행 중입니다. | تم حجب الوصول، لكن تنظيف بعض الجلسات لا يزال جاريًا. |
| Complete | Access is blocked for {n} sessions and cleanup is complete. Pages already shown in those browsers and requests already in progress cannot be recalled. | 세션 {n}개의 접근을 차단했고 정리가 끝났습니다. 해당 브라우저에 이미 표시된 화면과 진행 중이던 요청은 회수할 수 없습니다. | تم حجب الوصول لـ {n} جلسة واكتمل التنظيف. لا يمكن استرجاع الصفحات المعروضة مسبقًا في تلك المتصفحات ولا الطلبات الجارية. |
| Cleanup incomplete | Access is blocked for all {n} sessions; none of them can be used. Cleanup of {k} could not finish. | 세션 {n}개 모두 접근이 차단되어 사용할 수 없습니다. 그중 {k}개의 정리를 끝내지 못했습니다. | تم حجب الوصول لجميع الجلسات ({n})، ولا يمكن استخدام أيٍّ منها. تعذّر إكمال تنظيف {k}. |
| Unknown result (spec `:87`) | We could not confirm the result. Before submitting again, check your current sessions. | 변경 결과를 확인하지 못했습니다. 다시 제출하기 전에 현재 연결·세션 상태를 확인해 주세요. | تعذّر تأكيد النتيجة. قبل الإرسال مجددًا، تحقّق من جلساتك الحالية. |
| Revoked (spec `:88`) | Existing signed-out copy | Existing | Existing |
| Reauth (spec `:89`) | Existing HYB-293 password confirmation dialog | Existing | Existing |

- **Glossary rows**, KO and AR marked as AI drafts:
  - `session-access-blocked`: 접근 차단 / Access blocked / حجب الوصول. The session can no longer be used. It is not the same as cleanup complete.
  - `session-cleanup`: 세션 정리 / Session cleanup / تنظيف الجلسة. Removal of a blocked session's leftover records. Never shown as done while unconfirmed. It never recalls shown pages or in-flight requests.
  - `session-not-included`: 목록 이후 시작된 세션 / Started after the list / بدأت بعد القائمة.
  - `session-list-time`: 목록 기준 시각 / List taken at / وقت أخذ القائمة.

  The old `ended` and `endAllImpact` copy (`live-copy.ts:175-191`) is replaced.
- **Layout.** Uses the existing service-screen table and status patterns at full viewport width, with no decorative borders or accent strips (`service-screen-style.md`). At 390 px the dialog list and the held list become stacked rows. `ui-designer` gives direction before implementation and reviews the rendered screens.

## §10 Other callers that end sessions (F11)

| Caller | Line | Decision | Generation | Operation |
| --- | --- | --- | --- | --- |
| Password-attempt limit ends the current session | `self.go:212` | Stays synchronous through `endOne` | Changes (`drop`) | None |
| Fifth wrong current password | `self.go:259` | Same | Changes | None |
| Password change ends every other session | `self.go:288` | Stays synchronous and unconditional (`endOthers`) | Changes once per session | None. The page shows "Your other sessions were signed out", as today |
| Step-up attempt limit, and the reauth commit failure | `reauth.go:25`, `:63` | Synchronous `endOne` | Changes | None |
| Stale session at command entry | `commands.go:217` | Synchronous `endOne` | Changes | None |
| Administrator ends a person's sessions (`endSessions`) | `users.go:342-348` | Synchronous | Changes | None |
| Probe eviction | `users.go:350-362` | Synchronous, and ends denied verifiers first (F5) | Changes | None |
| Sign-in eviction | `session.go:154-162` | Same as the probe | Changes | None |
| Trust activation | `trust.go:326` | Synchronous | Changes | None |
| Bulk cleanup | `bulkcleanup.go:146-156` | Unchanged (HYB-299) | Changes | None |

- `endOne` itself also clears `s.denied` for its verifier.
- **Tests:** for each row, the caller's generation token changes, no `session_ends` entry is created, and a manifest that was listed before the change is reported as `generation_changed` with the affected handle `already_gone`.

# Threats and abuse cases

| # | Case | Handling |
| --- | --- | --- |
| T1 | An attacker with the password signs in after the person opened the list | The confirmed sessions are ended. The attacker's new session is held out, reported as `not_included` with an End action, and the changed copy recommends a password change. It is never ended silently, and never passed over silently either |
| T2 | The attacker keeps signing in while the person ends sessions | Each end still ends every confirmed session; it is never refused. The newcomers are shown each time. Ending everything unconditionally is a password change (`self.go:288`). Residual R2 |
| T3 | Forged handles or generation | Structural problems give 422. Foreign or unknown handles read as `already_gone`. Only the caller's own sessions are ever read |
| T4 | Replay of another user's digest | Operations are looked up per user only |
| T5 | Two of the person's own browsers end others at the same time | Serialised by the domain lock. If the first ends the second's session, the second gets 401. Otherwise the second reports the first's ended sessions as `already_gone` |
| T6 | Cleanup touches a session opened after the deny | Impossible: cleanup acts only on removed manifest verifiers (random, 256-bit) |
| T7 | Lock-free paths during the pending window | §4 table. The module end happens inside the deny, `s.denied` covers a failed end, and there is a test per path |
| T8 | Ledger isolation or operator recovery blocks self-protection | Never (§7, regression guards) |
| T9 | Leakage | The opaque token reveals no global counter (F4). Logs carry counts and internal counters only |

# Tests

- **Unit and contract** (`selfend_test.go`):
  - the decoder table, and every row of the codes table;
  - the step order (step-up before body, replay before deny);
  - the 202 shape and its parity with `GET /me`;
  - the single-handle path through the operation;
  - the `x-self` OpenAPI golden test;
  - the old body is refused;
  - the updated callers: `stepup_test.go:163`, `:171`, `:375`, `:474`, `reauth_test.go:338`, `self_test.go:251-295`.
- **Hold semantics (F3):**
  - exact-set: only confirmed and present handles are ended;
  - a newcomer is `not_included` and still resolves;
  - a confirmed handle that is gone is `already_gone`;
  - a foreign handle is `already_gone`, and the other login's session is untouched;
  - both generation tokens are returned and `generation_changed` is correct;
  - the set is never extended.
- **The `endManifest` false path (F2).** The caller's own session is dropped between admission and the deny, through a test seam. The answer is 401, and the sessions, operations, counter and generation are all unchanged.
- **Opaque generation (F4):**
  - the token is 32 hex and changes on every own add or remove;
  - another login's sign-ins and ends do not change the caller's token;
  - no token value repeats for a login over 10,000 changes.
- **Pending window (F1).** The module end is forced to fail, so the verifier is only in `s.denied`. Then `readers.go:112`, `boundary.go:119`, `signin.go:293` and `commands.go:174` each refuse the ended cookie, the pickup cannot be redeemed, and the current session works. A second run without the fault checks that the module refuses at once. A third run uses a seam between the module answer and admission: the deny lands there, and the post-answer recheck refuses (N1).
- **Retries (F6):**
  - the backoff schedule under the fake clock;
  - `ErrUnavailable` retries, then `failed` with `retry_budget_spent`;
  - an unexpected error gives `failed` at once, never done;
  - `ReasonSession` gives done or `already_ended`;
  - `s.denied` is kept until the sweep at `blocked_at + 12h`. A sweep test uses the fake clock: an entry is present before the deadline and gone after one tick past it. A cap test fills the set to `maxSessions` and checks that sign-in for that login fails closed and that no entry is dropped (N2).
- **Eviction and probe (F5).** At the 10-session limit with one denied, unended record: sign-in and the probe both end the denied record first and succeed.
- **Other callers (F11).** One test per row of §10.
- **Ledger and recovery states.** Regression guards in `isolated`, `stopped`, `full` and operator recovery.
- **Property and model test** (`selfend_property_test.go`). Random sequences over several logins: sign-in, logout, access drop, prune, end-one, end-others (fresh or stale token, honest, partial or foreign handles), drain, injected module failures and other-caller ends, compared with a model. Invariants:
  - I1: a login's internal generation strictly increases on every add or remove, and its token changes with it and never repeats;
  - I2: an end-others ends exactly the confirmed handles that are present. Every other session of the caller is unchanged and is reported `not_included` when present. Confirmed handles that are absent are `already_gone`;
  - I3: after a 202, no ended session resolves and the current session resolves;
  - I4: `count = done + already_ended + failed + pending`, over the sessions with result `ended` only; `already_gone` and `not_included` are outside `count`;
  - I5: no other login's sessions or token change;
  - I6: a replay ends nothing;
  - I7: a refused request (422, 401) changes nothing.
- **Concurrency** (`selfend_race_test.go`, with `go test -race`):
  - N goroutines sign in as the same login while end-others runs with a token read before them. Exactly the confirmed handles are ended, and every newcomer is alive and reported.
  - **Lock-free removals (F2):** goroutines drive the four paths in the §4 table with live and ended cookies, causing `access` drops through `sessions.drop` without the domain lock, during end-others. The snapshot and classification stay consistent and the counts invariant holds.
  - Concurrent end-others from two own sessions (T5).
  - The worker drain running concurrently with reads and new ends.
  - Commands: `go test -race ./apps/api/server -run 'SelfEnd|EndOwn|StepUp|Reauth'`, and the full package with `-race` once.
- **Mutation targets.** Run with the HYB-293 script pattern, cleaning the session `GOCACHE` per batch. Each must be killed:
  - classification (present, absent, newcomer swapped);
  - set extension (ending newcomers);
  - the token refresh on change;
  - the bump in `put`, `drop` and `prune`;
  - the single critical section split into two;
  - the false path creating an operation;
  - the module end moved out of the deny;
  - the `s.denied` check in `access`, and the second check after the module answer (N1);
  - the denied-first eviction;
  - the backoff bound;
  - unexpected error mapped to done;
  - the `already_ended` (C3) and `already_gone` (per-session) mappings, kept apart;
  - the count-invariant terms;
  - the replay digest match;
  - step-up before replay.

  A surviving mutant is traced to every reader before any code is deleted.
- **Console unit tests** (`live-model.test.ts` and `apps/console/src/service/admin.test.ts`):
  - the parsing of the new read and answer fields, with no casts for the new arrays;
  - the held and changed mapping;
  - a 422 reloads;
  - the AR plural forms;
  - polling stops at a final state or after 30 s.
- **Commands to record** in `verification.md`:
  - `go vet ./...`, `go test ./...`, and the `-race` runs;
  - console `npm run typecheck`, `npm test`, `npm run build`;
  - the line-count check (no maintained file over 1,500 lines).

# Browser QA

- **New runner `qa307.mjs`**, kept in the session scratchpad next to the existing runners and recorded in `verification.md`. It uses the `qa271.mjs` harness. Each check is run against a build of base `113c0d5` first and must **FAIL** there:
  - Q1: the confirmation lists the N other sessions with references and "This session is not included". Base shows a count only.
  - Q2: hold. Browser B signs in after browser A opened the confirmation. A confirms. Every listed session is ended (their browsers show the revoked copy), B is still signed in, A shows the held block listing B with an End button, and ending B from there works. On base, B is ended silently.
  - Q3: after a successful end, the result object shows "Access blocked", then "cleanup is complete", with the outside-scope rows in the disclosure. On base only "Ended N" is shown.
  - Q4: partial. A stubbed `GET /me` returns `cleanup_running`, then `cleanup_incomplete`, and the spec `:85` copy and the incomplete copy appear.
  - Q5: unknown result. A stubbed aborted POST gives the `:87` copy and a reload with no automatic resubmit. A resubmit returns the replayed operation.
  - Q6: a stubbed 422 `invalid_request` (fields `handles`) reloads the list and shows the changed state (F7).
  - Q7: EN, KO and AR at 1440 and 390, with RTL, no horizontal scroll, keyboard-only confirmation, focus on the status, and no decorative or coloured edge borders.
- **Regression.** Updated for the new contract and run in EN, KO and AR, with the results recorded:
  - `qa271.mjs` with `qa271body.mjs` (M05);
  - the HYB-293 runners that stub `/me/sessions/end` (`qa293body.mjs`, `qa293on297.mjs`, `qa293on298.mjs`).

# Dependencies

- HYB-293 step-up: merged; reused unchanged.
- HYB-298 and HYB-299 ledger and state naming: read only. No ledger code changes.
- HYB-306 (recent access history) touches the same view and `selfPage`. Whichever merges second rebases.

# Residuals

- **R1.** No durable record of self-service session ends (§7). Correct while sessions are memory-only. It must be revisited if sessions persist.
- **R2.** Someone who knows the password can keep opening new sessions while the person ends the listed ones. Each new session is shown with an End action and the copy recommends a password change, but "end others" alone cannot outrun repeated sign-ins. Password change is the unconditional route.
- **R3.** A C3 that failed after the budget leaves a module record until it expires or the server restarts. `s.denied` refuses it meanwhile, and eviction and the probe try it first.
- **R4.** `sessions.snapshot` and `byLogin` scan the whole store (at most `maxSessions` 10,000, `session.go:20`). Bounded and not per person; not changed.
- **R5.** Operations do not survive a restart. After a restart every session has ended.
- **R6.** The KO and AR copy, including حجب, are AI drafts until native-language and TW review.
- **R7.** The plan assumes one login per user and one user per login (§1). If one person can hold several logins (for example linked identities), the manifest must become per user, and this plan must be revised.

# Owner questions (root defaults apply until decided)

| # | Question | Default shipped |
| --- | --- | --- |
| Q1 | When sessions start after the list was taken: hold (end the confirmed list, show the newcomers with an End action), or refuse everything until the person reviews? | Hold (root decision on F3, following the HYB-299 precedent). Behaviour, not a setting |
| Q2 | Should self-service session ends also get a durable audit record (for example a non-replayed ledger kind)? | No. The log line only (§7) |
| Q3 | A confirmed session that ended elsewhere meanwhile | Reported `already_gone`; the rest proceeds (with Q1) |
| Q4 | Retry schedule (1, 4 and 16 s; 4 attempts) and operations kept per person (5) | Engineering constants. They can become installation settings with these defaults if the owner wants them changeable |

# Verification and review sequence

1. Security design review of this plan by `independent-reviewer`. Round 1 findings are below. This revision goes back to the same reviewer for round 2.
2. Visual direction from `ui-designer` for the dialog, the held block and the result object. TW reviews the copy table, including حجب.
3. Implementation on the ticket branch, within the allowed files.
4. Tests, the race runs and the mutation runs, each recorded with its command.
5. Browser QA: the new runner fails on base and passes on the branch, and the regressions pass.
6. Independent change review, with a security focus. Fixes are re-reviewed.
7. Then the PR, an authorized merge with read-back, and ticket closure only after the merge is confirmed.

# Review history

## Review round 1 (security design review, CHANGES REQUIRED: 3 Major, 2 Medium, 6 Minor)

| # | Finding | Resolution in this revision |
| --- | --- | --- |
| F1 | The deny left the module session live until cleanup, while lock-free paths consult the module | Root decision: `EndSession` runs inside the deny under the lock. A failed end goes to `s.denied`, which `access` checks first. The four lock-free lookup paths are listed; mid-stream proxied reads are `outside_scope` (as `bulkcleanup.go:38`). One pending-window test per path (§4, §5, Tests) |
| F2 | The snapshot and the check were not atomic | `sessions.snapshot` under one `sessions.mu` hold. Compare, classification and removal in one critical section. The false path is defined: 401, nothing changed, no operation, no counter change. A race test with lock-free removals (§1, §4, Tests) |
| F3 | Refusing departed from the planning and the HYB-299 hold precedent | Root decision: hold. End exactly the confirmed handles still present. Newcomers are `not_included` with an End action, gone ones are `already_ended`, both tokens are returned, and the set is never extended. T1, T2, R2, the changed copy, Q2 and I2 rewritten (§2, §9, Threats, Tests, Q1/Q3) |
| F4 | The global counter leaked other logins' activity | An opaque random 128-bit token per login, replaced on every change and compared for equality only. The internal counter is kept. A test that other logins do not change it (§1, Tests) |
| F5 | Eviction and the probe could miss records whose cleanup is pending | With F1 these are mostly gone. Eviction and the probe end the denied verifiers first. Tests (§5, §10) |
| F6 | No retry backoff, and unclear error mapping | Backoff of 1, 4 and 16 s with 4 attempts. Any error other than `ReasonSession` or `ErrUnavailable` gives `failed`, never done (§5) |
| F7 | A handle-set 422 left the person at a generic error | Foreign or unknown handles read as `already_ended`. Any 422 on end-others reloads and shows the changed state. Browser check Q6 (§3, §9) |
| F8 | `admin.test.ts` was missing from the allowed files | Added (Scope) |
| F9 | The AR term for the deny | حجب, following `product-glossary.md:69`. TW to confirm (§9, R6) |
| F10 | "Complete" could imply recall | The disclosure shows C6 and C7 as outside scope, and the complete copy says so (§5, §9) |
| F11 | The other session-ending callers were not classified | §10 table, with tests |
| — | Ledger-state tests | Labelled regression guards (§7) |
| — | The login and user one-to-one hypothesis | Residual R7 |


## Review round 2: APPROVE (N1–N3 folded in)

| # | Note | Folded in |
| --- | --- | --- |
| N1 | `access` must re-check `s.denied` and store presence after the module answer too | §4 step 3, with a seam test (Tests, Pending window) and a mutant |
| N2 | Remove `s.denied` entries by a sweep at `blocked_at + 12h`, key them by login as well, and cap the set | Worker-tick sweep. Entries indexed by login for eviction and the probe. Hard cap `maxSessions`, failing closed. Sweep and cap tests (§4, §5, Tests) |
| N3 | Keep the per-session result apart from the C3 state | Per-session result renamed `already_gone`; `already_ended` is the C3 state only. I4 counts only sessions actually ended. Copy `{j}` reads `already_gone` (§2, §5, §9, Tests) |

Status set to stable after round 2. The F3 and F7 rows above use the round-1 name `already_ended` for what is now `already_gone`.
