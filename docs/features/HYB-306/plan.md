---
type: ImplementationPlan
title: M05 recent access history kept in the installation data directory with a provisional, changeable retention period
description: Plan for HYB-306. The server records each person's sign-ins, refused sign-ins, password confirmations, sign-outs and session ends in append-only day files under the installation data directory, shows them to that person only on M05 under "Recent access history", keeps them for a provisional 90-day installation default, deletes older records unless a P04 legal hold covers them, and states truthfully what is kept, held, deleted, missing or still in backups. Revised after security and data design review round 1.
status: stable
tags: [HYB-306, security, identity, retention, M05, HB-43, HB-42, HB-03]
sources:
  - resource: docs/reviews/HYB-301-trace-classification/report.md
  - resource: docs/planning/v1.16/requirements.md
  - resource: docs/planning/v1.16/retention-access.md
  - resource: docs/planning/v1.16/ui-design.md
  - resource: docs/ui/screens/M05/spec.md
  - resource: docs/ui/screen-binding-fields/identity.md
  - resource: docs/ui/screen-binding-register.md
  - resource: docs/ui/terminology/product-glossary.md
  - resource: docs/features/HB-270/plan.md
  - resource: docs/features/HB-271/plan.md
  - resource: docs/features/HYB-298/plan.md
  - resource: docs/features/HYB-307/plan.md
---

# Scope and ownership

Linear ticket HYB-306 (team HYB): "HB-43 / M05 spec: recent access history is not built (HB-270/plan.md:125); its retention period is an open value. Implement with a provisional period recorded as an owner decision." It comes from row 3 of "Required and missing" in the [HYB-301 trace classification](../../reviews/HYB-301-trace-classification/report.md) (`report.md:318`, criterion `:133-135`).

- **Base.** main `73745d6` (HYB-307 merged). **Branch.** `feat/HYB-306-m05-access-history`. **Machine.** `mac_mini`.
- **Owner direction of 2026-11-17.** Build the spec behaviour as written; block or show truthfully when a dependency is missing; never simplify silently. Provisional values ship as installation defaults that can be changed. Required ledgers persist in the installation data directory.
- **Source availability.** `docs/planning/v1.16/*` and `docs/ui/screens/M05/spec.md` exist in the authorized local workspace and are not in Git. Line numbers below are from that workspace on 2026-11-19.
- **Ownership.**
  - Root (Claude Code) owns the plan, implementation, tests and integration.
  - `independent-reviewer` (read-only) does the **required** security and data design review of this plan, then the change review. Required because this adds a stored personal-data ledger, a retention and deletion behaviour, a self-scoped read and a public `/api/v1/me` contract.
  - `ui-designer` (read-only) gives visual direction for the new M05 region and reviews the rendered screens.
  - Tech-writer review (read-only) covers the copy table and the glossary rows.
  - QA writes and runs the browser runner.
- **Allowed files** (implementation):
  - New package `modules/accesshistory/` (record codec, day-file store, index, writer, device families, file lock) and its tests.
  - `apps/api/server`: new `accesshistory.go` (server glue, read handler, purge tick), new `sessionending.go` (the session-end and module-end helpers, F3, G1, G2; the existing HYB-307 `sessionend.go` keeps the store's manifest code) and `accesshistory_test.go`, `accesshistory_property_test.go`, `accesshistory_race_test.go`, `sessionending_test.go` (the guard test); session-end and recording changes in `signin.go`, `reauth.go`, `self.go`, `selfend.go`, `users.go`, `session.go`, `commands.go`, `trust.go`, `bulkcleanup.go`, `periods.go`, `admin.go`; the route in `boundary.go`; a read-only hold query in `retention.go` and an exported coverage query in `modules/retention` (`exports.go`, reusing `hold.covers`, F6, G8); `settings.go`; `seed.go` (`Start`); `ledger.go` (`Close`); `openapiself.go`; existing tests that need the new seam.
  - `apps/api/cmd/hb-api/main.go` and `main_test.go` (new setting group and the one-time flags).
  - `apps/console/src/service/admin.ts` and `admin.test.ts` (types, parser, `accessHistory`).
  - `apps/console/src/modules/identity/pages/my-security/`: new `AccessHistory.tsx`, `history-copy.ts`, `history-model.ts`, `history-model.test.ts`, `history.css` if needed; `Live.tsx` (mount the region, remove the "not available" note); `live-copy.ts` (remove `historyNote`).
  - Docs: `docs/features/HYB-306/*`, the M05 rows in `docs/ui/screen-binding-fields/identity.md` (`:595-673`), the M05 entry in `docs/ui/screen-binding-register.md:199-205` and `:656`, new rows in `docs/ui/terminology/product-glossary.md`.
- **Not in scope.** An administrator view of other people's access ("비정상 로그인 감사", `retention-access.md:65`, M01). MFA, recovery and identity linking. P04 retention policy revisions over this history (residual R4); P04 legal holds **are** honoured (§3, F6). The HYB-307 end flow and its "Recent session ends" region keep their contract; only their session-end calls move into the shared helper (F3).

# Requirement cross-check

## Verbatim sources

| ID | Source line | Verbatim |
| --- | --- | --- |
| HB-43 req | `requirements.md:771` | `**요구사항:** 사용자 추가 지시: 데이터 종류·업무 공간·계약·증빙 관계별 보관·아카이브·삭제 정책을 관리한다. 보관기간·법적 보류·실제 물리 삭제·백업 만료를 구분한다.` |
| HB-43 open | `requirements.md:779` | `**선행 요구:** HB-03, HB-08, HB-21, HB-37. **결정 필요:** 분류별 보관 기준일·기간·법적 보류 책임자·백업 만료·삭제 목표시간·개인정보 최소화·암호화 키와 공유 범위.` |
| Legal period | `retention-access.md:69` | `법정 보관기간은 이 문서에서 임의 지정하지 않는다. 관할·계약·문서 성격에 맞게 책임자가 확정할 결정값이며 법률 검증을 했다는 뜻이 아니다.` |
| Class row | `retention-access.md:76` | `\| 사용자 프로필·identity·접근 감사 \| 소속 종료/사건일 기준, 업무 actor 참조와 보안 조사 \| 인증 비밀·불필요 프로필 제거, 필요한 actor 참조 최소화 \|` |
| Backups | `retention-access.md:80` | `\| 백업·객체 이전 버전·복제·이관본 \| 실제 복원 가능 창과 저장소별 만료 \| 활성 삭제와 별도 상태. 백업 만료 전 모든 사본 삭제 완료로 표시 금지 \|` |
| Policy change | `retention-access.md:82` | `` `RetentionPolicyRevision`은 범위, 등급, 기산점, 기간, hot/archive, 삭제 방식·유예, 책임자, 승인, 적용일을 가진다. 기산점은 재수집·복사·아카이브 이동으로 초기화하지 않는다. 정책 단축·소급 적용은 삭제 건수/바이트/고객/증빙 영향 미리보기 후 관리 권한·승인 정책을 거친다. 사용자 삭제 기능 구현 요청이 실제 고객 데이터 즉시 삭제 지시는 아니다. `` |
| Account existence | `retention-access.md:41` | `… 로그인·재설정 응답은 계정 존재를 드러내지 않게 설계한다.` (last sentence quoted) |
| Admin audit | `retention-access.md:65` | `화면에는 사용자 검색/상태/소속/역할/인증 방식/세션, … 비정상 로그인 감사를 제공한다. 비밀번호·LDAP bind secret·토큰은 열람 목록에 표시하지 않는다. …` (excerpt) |
| M05 purpose | `M05/spec.md:22` | `본인이 자신의 인증 수단·세션·최근 접근과 개인 알림 선호를 관리하고 안전하게 계정 접근을 유지한다.` |
| M05 permission | `M05/spec.md:28` | `- 권한: 현재 사용자·업무 공간·대상·행위·필드 권한을 서버에서 재검사한다. 클라이언트 필터나 역할명만으로 허용하지 않는다. 권한 밖 객체의 존재·건수·자동완성·파일명을 노출하지 않는다.` |
| M05 sessions | `M05/spec.md:34` | `3. **세션**: 현재/다른 세션과 최근 접근을 최소 개인정보로 보여주고 회수 효과를 설명한다.` |
| M05 field | `M05/spec.md:51` | `\| 현재 세션·기기 \| session-list \| 예 \| 본인 종료만 \| 현재세션 표시·발급/최근사용/만료/회수효력; 상세 위치는 최소화 \| 개인 \|` |
| M05 tabs | `M05/spec.md:59` | `- 탭/하위 보기: 내 계정 / 로그인 수단 / 추가 인증·복구 / 세션·최근 접근 / 알림 설정.` |
| M05 filters | `M05/spec.md:60` | `- 필터: 현재/다른 세션 / 최근 접근 기간 / 본인 소속 업무 공간.` |
| M05 columns | `M05/spec.md:61` | `- 열/요약 값: 로그인 제공자 / 연결 상태 / MFA 근거 / 마지막 사용 / 기기/세션 / 만료·회수 상태.` |
| M05 states | `M05/spec.md:91` | `권한 회수 시 보이는 민감값·이전 미리보기·선택 캐시를 제거하고 같은 데이터 재다운로드를 막는다. 오류에는 비민감 요청 참조와 안전한 다음 조치만 포함한다. 0·없음·미집계·알 수 없음은 별개이며 미확인 값을 0으로 채우지 않는다.` |
| M05 bulk | `M05/spec.md:95` | `10만 사용자 전체를 불러오지 않고 본인 UserID로 범위를 고정한다. 본인 세션·최근 접근이 많으면 cursor+기간 필터를 사용한다. …` (first two sentences quoted) |
| M05 open | `M05/spec.md:119` | `미정: 본인 변경 가능한 프로필 항목; MFA/복구 수단·재인증 유효시간; credential 변경 시 세션 회수 범위; 필수 알림 예외와 개인 선호 상한; 최근 접근 보존기간/기기 정보 수준.` |
| UI plan | `ui-design.md:154` | `\| M05 내 보안·로그인 연결 \| 본인 \| 로그인수단·본인 소속·MFA·세션·최근 접근, 비밀번호 변경/허용identity 연결·해제/다른 세션 종료 \| 기존 재인증+새 신원 증명·최후수단 보호·이메일 일치 병합 금지. HB-03·42 \|` |
| Prior record | `HB-270/plan.md:125` | `- A recent access history beyond the session list.` (under "Excluded, and marked unavailable on M05") |

## What the sources do and do not settle

- **Settled:** the history is the person's own (`:22`, `:95` 본인 UserID), shown with minimal personal data (`:34`), with a period filter and cursor (`:60`, `:95`), detailed location minimised (`:51`), "unknown" never shown as zero (`:91`); its class is access audit, counted from the event date, with secrets removed (`retention-access.md:76`); backup copies are never reported deleted (`:80`); a shortening needs an impact preview and authority (`:82`); responses never reveal whether an account exists (`:41`).
- **Not settled (open values, `M05/spec.md:119`, `requirements.md:779`):** the retention period, the device information level, the deletion target time, and **which events count as "최근 접근"**. The spec never enumerates the events. The event set in §1 is a root proposal, recorded as owner question Q1, not a reading of the spec.

## Cross-check table

| Clause | Requires | Today (`73745d6`) | This plan |
| --- | --- | --- | --- |
| `:22`, `:34`, `ui-design.md:154` 최근 접근 | A recent access view on M05 | Not built; the limits panel says "recent access history is not available" (`live-copy.ts:180`, `Live.tsx:468`) | Region "Recent access history" on the sessions view (§7), fed by `GET /api/v1/me/access-history` (§6) |
| `:95` 본인 UserID로 범위 고정 | Only the caller's own records | — | The server takes the user from the authenticated session; the read has no user parameter and refuses unknown parameters (§6) |
| `:28` 권한 밖 객체의 존재·건수 | Nothing about other people, not even counts | — | Records of other people are never read, counted or hinted at: the cursor is the person's own record ordinal (F1), note counts are the person's own or absent (F2), and refused sign-ins for unknown logins are not recorded at all (§1, §4, §6) |
| `:41` 계정 존재 비노출 | Sign-in answers never reveal account existence | Same answer for unknown login and wrong password (`signin.go:226-228`) | Recording is a non-blocking enqueue after the answer code is chosen; the answer neither waits for nor depends on the write, so its status and timing do not change with account existence (§4, T2) |
| `:60`, `:95` 기간 필터·cursor | Period filter and cursor | — | `from`/`to` and an opaque `before` cursor, page size 50 (§6) |
| `:51` 상세 위치는 최소화; `:119` 기기 정보 수준 | Minimal device and no detailed location | — | Coarse browser and OS family only, from a fixed vocabulary; no IP address, network, location or full User-Agent (§1, Q3) |
| `:61` 기기/세션 | A device/session column | Session column only in the session table | Columns Time, Event, Session, Device (§7) |
| `:91` 0·없음·알 수 없음 별개 | Empty, unknown and failed are distinct | — | Separate states for "no events in this period", "not recorded" gaps, "recording stopped" and "could not load" (§7) |
| `retention-access.md:76` 사건일 기준; 인증 비밀 제거 | Event-date basis; no secrets | — | The retention clock counts from each record's own time, which a copy, restart or reload never resets; no secret is ever written (§1, §3) |
| `:779`, `:119` 기간 | A period is an owner decision | — | Installation setting `access_history_days`, default 90, range 1–1095, shown on M05 with its source; owner question Q2 (§3) |
| `:779` 삭제 목표시간 | A deletion target | — | Records past the period are hidden at once and deleted at the next UTC day boundary while the server runs, otherwise at the next start before serving; an overdue or deferred deletion is shown (§3, F7, F9) |
| `:80` 백업 만료 전 삭제 완료 표시 금지 | Backups are a separate state | — | The retention statement says copies in backups of the data directory remain until those backups expire (§7) |
| `:82` 정책 단축 영향 미리보기·권한 | A shortening is previewed and authorised | — | A shorter period refuses `Start` and prints the impact (records, bytes, people count) until the operator restarts with `-accept-access-history-shortening=N`, where N must equal the new period (§3, F10); the second-person approval of P04 is not available (R7) |
| `:771` 보관·법적 보류·물리 삭제·백업 구분 | Four distinct facts | — | Period (setting), legal hold (P04 holds on the `identity` class stop deletion, persisted as held markers, F6), physical deletion (day-file deletion or rewrite, overdue note) and backup (statement) are four distinct states (§3) |
| Owner direction: ledgers persist | The history survives restarts | — | Append-only, fsynced day files under `<data>/access-history/` (§2) |
| `HB-270/plan.md:125` | Recorded as excluded | Excluded | Built; the register and field map are updated |

# Current behaviour (facts from code at `73745d6`)

- **Sign-in** (`signin.go:206-242`): the body is checked, then `passwords.Verify` runs off-lock in `derive`; a non-OK result answers 401 `unauthenticated` (`:226-228`) for both unknown logins and wrong passwords; `signIn` (`:257-292`) opens the session under `s.domain.Lock()` and may refuse through `Access` or the policy (`:283-289`), also 401. The global limiter guards every attempt (`signin.go:42-58`, `:167`).
- **Logout** (`signin.go:294-313`) ends the module session and drops the store entry.
- **Password confirmation** (`reauth.go:10-53`) logs `reauthenticated` or `reauthentication_failed` with a result (`wrong`, `ended`, `access`, `conflict`, `unavailable`).
- **Session ends are scattered** (F3). A store session is ended at these sites:
  - `endOne` (`self.go:114-119`), called from `self.go:126` (`endOthers`, password change `self.go:251`), `self.go:175` and `:222` (self-service attempt limit), `reauth.go:25` and `:63`, `commands.go:227`, `trust.go:326`;
  - `endSessions(login)` (`users.go:317-321`), called from administrator departure (`admin.go:327`), role-period end (`periods.go:219`) and a password reset through a link (`signin.go:428`);
  - evictions at the module limit: `openSession` (`session.go:223-224`) and `probe` (`users.go:337-338`);
  - `access` (`commands.go:139-168`): denied verifier, stale trust epoch, not fresh, refused by the module;
  - the bulk revocation cleanup (`bulkcleanup.go:170-171`);
  - the M05 deny through `endModule` (`selfend.go:195-203`);
  - logout (`signin.go:304-305`);
  - the refused sign-in rollback (`signin.go:283-284`), which ends a session that never reached the person.
  Module-only probe sessions that never enter the store are ended at `signin.go:433`, `:443` and `seed.go:160-179`.
- **No persistence.** Sessions, the directory and self-service state are in memory and reset on restart (`HB-270/plan.md:161-163`). Only the decision ledger persists (`modules/decisionledger`, HYB-298).
- **User IDs.** People added at runtime get `user-<16 random hex>` (`admin.go:236-240`). Seeded and bootstrap IDs are fixed and are added again at every start (`seed.go:101`, `:172`, `:184`; bootstrap activation `signin.go:477` with `s.bootUser`). So the same ID **can** belong to a different human after a restart, for example a new operator who activates `EmptyAdmin` (F4; the first draft's "never reused" claim was wrong).
- **Data directory** (`-data`, default `.hb-api`) is checked by `checkData` (`seed.go:50-63`): created `0700`; refused when a symlink, not a directory, group/world accessible, or not owned by the process user (non-Windows).
- **Settings** are integer installation settings with name, flag, unit, default and range (`settings.go:43-82`), parsed by `numberFlag` (`main.go:161`), grouped (`allSettings`, `withDefaults`, `settings.go:95-126`), and shown with source `default` or `start_flag` (`settings.go:11-13`, `:199-201`).
- **Clock.** `Start` first runs `checkData`, then `setClockBase` (`seed.go:75-81`, `clockbase.go:22`), which compares the wall clock with decision-ledger stamps; `leadBeyond` (`clock.go:84`) reports an unaccepted lead beyond the skew tolerance.
- **Shutdown.** `run` defers `s.Close()` before serving, and `srv.Shutdown` returns before the deferred `Close` runs (`main.go:63-90`), so `Close` follows the HTTP shutdown. On a `Serve` error, `Close` also runs after `Serve` has returned.
- **P04 holds** (`apps/api/server/retention.go`, `modules/retention`) carry `Tenants` and `Classes` (`types.go:147-154`) and are kept in memory; they do not survive a restart.
- **Server listen address** must be loopback (`main.go:48`). The remote address the API sees is therefore the local proxy, and no trusted forwarded-address configuration exists.
- **M05 sessions view** (`Live.tsx:416-498`): the sessions table, the HYB-307 "Recent session ends" region (`EndsRegion`, title `end-copy.ts:84`), and the limits panel with the "not available" note (`Live.tsx:468`).

# Design

## §1 What is recorded

**Events** (root proposal, Q1). One record per event, for the person whose login it concerns:

| Kind | When | Fields beyond the common ones |
| --- | --- | --- |
| `signed_in` | `signIn` returns ok | `workspace` (from the answer), `session` handle, `device` |
| `sign_in_refused` | A sign-in for an **existing** login fails after the password was judged | `reason` (below); `device`, classified for every refused attempt (F15). The only kind with no workspace |
| `password_confirmed` | `commitReauth` returns "" | `workspace`, `session` |
| `password_confirmation_failed` | `commitReauth` returns `wrong`, or the attempt limit ends the session | `reason`: `wrong_password` or `attempt_limit`; `workspace`, `session` |
| `signed_out` | Logout of a live session | `workspace`, `session` |
| `session_ended` | `endSession` (§4) ends a store session | `reason` (below); `workspace`, `session` |
| `password_changed` | `changePassword` succeeds | `workspace`, `session`; the ended sessions get their own `session_ended` with reason `password_changed` |

Every session-bound record carries the **session's own workspace** (`session.workspace`), never the reader's (F8).

`sign_in_refused` reasons, one per refusal path in `login` and `signIn` (F15):
- `wrong_password`: `Verify` is not OK and not locked (`signin.go:226`);
- `locked`: `Verify` returns `passwords.ResultLocked`;
- `credentials_changed`: the login's generation changed between the check and `signIn` (`signin.go:260`), for example a password reset in between;
- `access_refused`: password correct, then `Access` or the policy refused (`signin.go:273-289`);
- `session_unavailable`: `openSession` failed for a reason other than store overflow (`signin.go:268-271`).
- Store overflow answers 503 `dependency_unavailable` (`signin.go:231-234`) and is **not** recorded: no person-specific decision was made.

`session_ended` causes (F3), one per site in "Current behaviour":

| Cause | Sites |
| --- | --- |
| `ended_by_you` | M05 end one or others (`selfend.go` deny) |
| `signed_out` is its own kind | logout (`signin.go:304`) |
| `password_changed` | `endOthers` from `changePassword` (`self.go:251`) |
| `password_reset` | `endSessions` from a reset link (`signin.go:428`) |
| `attempt_limit` | `self.go:175`, `:222`; `reauth.go:25`, `:63` |
| `administrator` | administrator departure (`admin.go:327`) and administrator session ends (`users.go:317-321`) |
| `role_period_ended` | `periods.go:219` |
| `session_limit` | evictions in `openSession` and `probe` |
| `access_changed` | `access` branches, mapped by the exact refusal value (G3, J4): stale trust epoch (`commands.go:144`) is `trust_change`; for a module `Access` refusal (`commands.go:168`): `identity.ReasonUser` (`user_not_eligible`) and `identity.ReasonOperatorBlock` (`operator_block`) give `administrator`; `identity.ReasonMembership` (`no_active_membership`) gives `access_changed`; `identity.ReasonRolePeriod` (`role_period_ended`) gives `role_period_ended`; any other identity reason gives `access_changed`. For `fresh` failing (`commands.go:152`, `commands.go:252-257`): `authpolicy.ReasonStale` (`stale`, the maximum sign-in age) is expiry; any other refusal (`blocked`, `method_not_allowed`, `issuer_not_allowed`, `mfa_required`, `future_authentication`, …) gives `access_changed`. Also `commands.go:227` (stale session at command entry) is `access_changed` |
| Expiry records nothing | `identity.ReasonSession` (`session_not_current`, which the module returns for idle and absolute expiry, `registry.go:264-277`), `authpolicy.ReasonStale`, the prune at sign-in (`signin.go:267`), and the denied-verifier branch (already recorded as `ended_by_you` at the deny) record nothing; the region says expired sessions are not listed |
| `trust_change` | `trust.go:326` |
| `bulk_revocation` | `bulkcleanup.go:170` |

Expiry is not an event: an expired session is simply gone, and the region says so.

**Not recorded:**
- Sign-ins for logins that do not exist (no person to attach them to, and no trace that would differ by account existence).
- `dependency_unavailable` and rate-limited attempts (no password was judged). The global limiter bounds the rate of all attempts.
- Password set through an activation or reset link, and administrator-initiated resets (R10).

**Common fields** (every record): `v` (format 1), `seq` (installation-wide, strictly increasing across restarts, internal only and never sent to a client, F1), `at` (server clock `s.clock()`, UTC, milliseconds), `user` (internal user ID), `kind`, `reason` (fixed vocabulary or empty).

**Person incarnation** (F4). Because a fixed user ID can pass to a different human, each record belongs to an incarnation:
- An `incarnation {user}` system record is written when a person is added at runtime (`admin.go:240`) and when an account is activated, including the bootstrap activation (`signin.go:477` and the activation path of `setPassword`, `signin.go:431-434`).
- Re-adding seeded people at `Start` (`seed.go:101`, `:172`, `:184`) writes no marker: the sample seed recreates the same sample identity.
- **Synchronous and fail-closed** (G4, J2, J3). The marker is not an ordinary queued record. It is appended as one line to the **person-state file** `people.log` (§2) and fsynced before the add or activation changes anything, under the history-state lock, which is separate from the writer and from purge and rewrite work. `floor_seq` is allocated atomically from the writer's `seq` counter in the same step (J3), so it is above every `seq` already assigned, including records still queued. The in-memory floor (`floor[user] = floor_seq`) is set in that step, so the read hides earlier records at once, even when the writer has stopped or its queue is blocked. When the writer runs, an `incarnation` system record is also queued.
- **Order** (round 3). The marker is written **before** the irreversible step, and a failed write answers 503 `dependency_unavailable` with nothing changed:
  - bootstrap activation: before `Bootstrap` (`signin.go:473`);
  - invitation acceptance (`setPassword` with an activation link): before `passwords.Commit` (`signin.go:421`);
  - runtime add: before the identity add (`admin.go:236-240`).
  Because nothing was consumed, the same activation link and the same invite link still work after the 503. A marker written before a step that then fails leaves an extra floor, which only hides that user ID's earlier records (the safe direction).
- **Lock order.** The domain lock, then the history-state lock; never the reverse. The history-state lock guards `people.log` appends and the floors; purge, day-file rewrite and early deletion run on the writer goroutine and never take it while doing file work. The single exception is the `people.log` compaction, which holds it across its read, rewrite and rename (K1).
- The read returns only the caller's records with `seq` above the floor for that user ID. Earlier records stay on disk until the period ends and are never shown to the new incarnation.
- **Unseeded installations** (G5). With `-seed none`, every start re-arms the bootstrap and the operator activates `EmptyAdmin` again, which writes a new marker; that administrator's earlier history is then hidden after every restart. This is the safe reading (a different human may be activating) and is residual R12.

**Never stored:** passwords or any part of them, cookies, session tokens, verifiers, CSRF tokens, pickup tokens, the login ID, email, display name, the requested workspace of a failed sign-in (attacker-chosen text), IP addresses, forwarded headers, location, the raw User-Agent. The session `handle` is stored: it is the 16-hex public reference already shown on M05, derived one-way from the 256-bit verifier (`self.go:51-54`), and cannot be used to act on a session by anyone but its owner.

**Device family** (`modules/accesshistory/device.go`). The User-Agent is read once at the request, classified and discarded:
- `browser` ∈ `edge`, `chrome`, `firefox`, `safari`, `other`, `unknown`;
- `os` ∈ `windows`, `macos`, `linux`, `ios`, `android`, `chromeos`, `other`, `unknown`;
- the classifier reads at most the first 512 bytes, matches fixed tokens in a fixed order, and never stores or echoes the input. An absent header is `unknown`; an unrecognised one is `other`.

**Per-person refused sign-in cap.** At most 100 `sign_in_refused` records per person per UTC day. Further refusals that day are counted in memory and written as one `sign_in_refused_more` record with `count` at the day boundary or at `Close`. After a crash the count may be lost; the unclean-stop note (§2) covers that window. This keeps one targeted person's flood from evicting everyone else's history (T4). An engineering constant (Q9).

## §2 Storage in the installation data directory

**Alternatives considered.**
- *Reuse the decision ledger.* Rejected: it is a hash-chained, never-pruned decision record whose isolation states block access changes (HYB-298). Access history needs pruning by age and must never put access decisions into isolation.
- *One file per person.* Rejected: up to 100k people (`:95`) means 100k files and a directory scan per purge.
- *One installation-wide append-only file per UTC day* (chosen): append is sequential, purge deletes whole files, and the load rebuilds a per-person index.

**Layout** (all under `<data>/access-history/`, checked with the `checkData` rules at `Start`, created `0700`; creating it syncs `<data>`):
- `YYYY-MM-DD.log` (`0600`), one per UTC day of the record's `at`. Creating a file syncs the directory.
- `retention.json` (`0600`): the effective period and its history `[{days, since}]`, the highest `seq` written (F14), the previous run's last record time and any pending clock deferral (J1), and the held markers (§3, F6). It stays small. Written by temp file, `Sync`, rename and directory sync.
- `people.log` (`0600`, J2): append-only, one checksummed line per entry (same line format and per-file checks as the day files), fsynced per append. It holds the incarnation entries `{user, floor_seq, at}` and the release audit entries. A person add appends one line; nothing large is rewritten.
  - **Compaction at purge:** an incarnation entry is dropped once no kept record of that user has a `seq` below its floor (nothing left to hide). The file is rewritten on the writer goroutine (temp, `Sync`, rename, directory sync) only when compaction removes entries. **The history-state lock is held across the whole read, rewrite and rename** (K1), so no marker append can land in the old file after it was read; this is cheap because the file is small after compaction. This is the one place where file work runs under that lock.
  - **Damage and torn tail at `Start`** (K2): a damaged line in the middle of `people.log` (bad checksum, bad JSON, unknown entry kind) refuses `Start` with `ErrConfig`, naming the file and the line number, because a lost floor could show a new holder an earlier holder's records. A torn tail (no final newline) is truncated to the last newline and synced: its add or activation never answered success, because the answer waits for the fsync.
  - **Audit bound:** each release audit entry is kept for at least the access history period plus 7 years, then compacted into a count per year (owner question Q11).
  - **Scale:** append time is measured at 100k users (100k entries) and recorded in `verification.md`.
- `lock`: an exclusive OS lock held for the life of the server (`flock` on Unix, `LockFileEx` on Windows), following `modules/decisionledger/open_unix.go` and `open_windows.go`. A second process on the same directory fails `Start`. The few lock lines are duplicated rather than moving decision-ledger code (R11).

**Per-file checks** (F12), as the decision ledger applies to its file: every day file, `*.held-damaged` file, `retention.json`, `people.log` and `lock` is opened without following symlinks and refused when it is a symlink, not a regular file, has a link count above 1, is group- or world-accessible, or is not owned by the process user (non-Windows). A refused file refuses `Start` and is named in the message. On Windows, mode and ownership are not checked (HYB-298 R8), and a directory sync is a no-op that never refuses `Start`.

**Handles** (F12). The writer keeps open only the handle of the day it is appending to. Handles of other days are closed after each batch, and every handle of a file is closed before that file is deleted or rewritten, so deletion also works on Windows.

**Record line.** `<crc32c, 8 lowercase hex> <compact JSON>\n`, at most 1 KiB. The CRC covers the JSON bytes. The JSON uses only the fields of §1 with fixed key names.

**System records** carry `user: ""` unless stated, and are never shown as a person's events. They produce the notes of §6: `run_started`, `run_stopped`, `dropped {count, from, to, per_user}`, `recording_stopped`, `recovered_tail`, `early_deletion {before, per_user}`, `purged {through, files, records}`, `period_changed {from_days, to_days}`, `incarnation {user}` (F4), `held {hold, day}` and `hold_released {hold}` (F6), `purge_deferred {reason}` (F7).

**Writer.**
- Recording calls `history.offer(rec)`: a non-blocking send on a bounded queue (4,096). It never blocks, never takes the domain lock and is safe to call while holding it.
- One goroutine takes batches (up to 256 records or 50 ms), appends each line to its day file and calls `Sync` once per touched file. `seq` is assigned in the writer.
- A record enters the in-memory index **only after its `Sync` succeeds**. The screen never shows a record that could vanish in a crash.
- Queue full: the record is dropped and counted **per user** (records with a user) and in total; the counts and time range are written as one `dropped` system record when the queue drains (F2).
- Write, short-write or `Sync` error: the writer stops for the rest of the run (`recording: stopped`, `since`), because the file may end in a partial line (as in HYB-298). Further offers are counted as dropped in memory, per user. It tries once to append a `recording_stopped` record; if that fails, the state is still shown from memory. Sign-in, sign-out and session ends keep working (Q4).
- **`Close`** (F11, G12), from `Server.Close` (`ledger.go:74`), which runs after `srv.Shutdown` has returned (`main.go`, see "Current behaviour"). `Shutdown` waits for active handlers, but background workers (`every` goroutines, `bulkcleanup.go:203-204`, and the purge timer) can still end sessions, so `Close` first closes `s.stop` and **waits** for those workers (a `sync.WaitGroup` added to `every`), then closes the history. HTTP handlers that outlive `Shutdown`'s context are not claimed to be stopped:
  1. stop accepting; any later `offer` is counted as dropped, per user;
  2. drain the queue for at most 2 s; records still queued after that are counted as dropped, per user;
  3. append one `dropped` record with those counts (if any), then the pending `sign_in_refused_more` counts, then `run_stopped`; `Sync`; update `retention.json` with the highest `seq` and the last record time (G9); release the lock;
  4. an `offer` that arrives after step 3 cannot be written: it is counted and logged once (`access_history_offer_after_close`, count only); the next start shows no note for it, so the claim is limited to "recorded unless offered after the history closed".
  A test pins the order with a handler offering during shutdown.

**Load at `Start`** (F7): after `checkData` **and after `setClockBase`** (`seed.go:75-81`), and before the listener serves:
1. Take the lock. Read `retention.json` (absent means first start: write it with the configured period and `seq` 0).
2. Apply the period-change rule (§3). A refused shortening returns an `ErrConfig` error and the server does not start.
3. Read the day files in name order. Ignore files whose name is not a valid date; a file dated after `now + clock skew tolerance` is read but reported as a `future_file` note.
4. **Torn tail:** a file not ending in `\n` is truncated to its last newline, synced, and a `recovered_tail` record is appended. **Damaged line** in the middle (bad CRC, bad JSON, unknown kind or enum, over 1 KiB): skipped and counted in a `damaged` note, per user when the user field can still be parsed; never shown.
5. **Sequence continuation** (F14): the next `seq` is above both the highest `seq` read from the files and the highest recorded in `retention.json`, so deleted or truncated files never lead to a reused `seq`.
6. **Unclean stop:** a `run_started` without a later `run_stopped` gives an `unclean_stop` note spanning the last record of that run to the new `run_started`. It has no count: what was lost is unknown (F2).
7. Build the index from records within the period and after each user's latest `incarnation`; append `run_started`; `Sync`.
8. Check the size bound against the period (F5, §3).
9. Purge (§3) before the listener serves, unless the clock rule defers it.

An unsafe directory or file, a lock held by another process, an unreadable file or a failed `Sync` at load refuses `Start` with `ErrConfig` and a message naming the path; the operator fixes it. Damaged lines and torn tails do not refuse.

**Index.** `map[user][]entry` with compact entries (`seq`, `ordinal`, `at`, `kind`, `reason`, `workspace` interned, `handle [8]byte`, `device` codes, `count`), each slice in `seq` order, guarded by its own `RWMutex`. `ordinal` is the record's position in its own sequence (F1, G11, root decision): one sequence per **(user, workspace)** for session-bound records, and a separate sequence per user for refused sign-ins (workspace-less). It is stored in each personal record as `n` with the sequence key, assigned by the writer, restarted at 1 after each incarnation floor, and continued at load from the highest `n` kept in that sequence. A cursor therefore reflects only the caller's own events in the caller's own workspace scope; the read merges the two sequences of the scope (§6). If all records of a sequence were deleted the counter may restart, and an old cursor then simply returns newer records of the same sequence. The workspace interning table is bounded by the workspaces that exist. Memory and load time are bounded by the file bound (§3) and **measured** at the maximum (F5); the estimates in Q5 are not results.

**Durability** is claimed for a local disk only, as HYB-298 assumption A5. Windows mode bits are not enforced (HYB-298 R8).

## §3 Retention period, size bound and deletion

**Settings** (F13): new group `historySettings` in `settings.go`, added to `allSettings`, to the group loop of `withDefaults` (so an all-zero group takes its defaults) and to the list `accessFlags` iterates in `main.go`. The source vocabulary is the existing one: `default` or `start_flag` (`settings.go:11-13`).

| Setting | Flag | Unit | Default | Range |
| --- | --- | --- | --- | --- |
| `access_history_days` | `-access-history-days` | days | 90 (provisional, Q2) | 1–1095 |
| `access_history_max_mib` | `-access-history-max-mib` | MiB | 256 | 16–1024 (maximum lowered from 4096 until measured, F5) |
| `access_history_clock_gap_days` | `-access-history-clock-gap` | days | 7 (provisional, Q12) | 1–365 |

One-time flags:
- `-accept-access-history-shortening=N` (F10): N must equal the configured new `access_history_days`. A missing flag, or an N that differs, refuses a shortening. The flag is refused when no shortening is pending, so it cannot be left in a start script by mistake.
- `-release-access-history-marker=<marker ID>` with `-release-access-history-reason=<text>` (G6): releases one persisted held marker from an earlier run (see below).

**Period basis.** Each record's own `at`. A restart, reload, copy or setting change never resets it (`retention-access.md:82`).

**Changing the period.**
- *Longer or equal:* applied at `Start`; `retention.json` gains `{days, since: now}` and a `period_changed` record is written. Records already deleted do not come back; `available_from` (§6) shows the real coverage.
- *Shorter:* `Start` is refused with `ErrConfig` and a message: "access_history_days would shorten the access history from {old} to {new} days and delete {records} records ({bytes} bytes) of {people} people; restart with -accept-access-history-shortening={new} to apply". The counts come from a dry run over the loaded index and file sizes, excluding held records. With a matching flag, the change is applied, recorded and purged before serving. People are counted, never listed.

**Legal holds** (F6, G8, root decision option a). Before deleting or rewriting any day file, the purge asks the P04 registry for live holds that cover class `identity` for a tenant, **using P04's own predicate**: `hold.covers` (`modules/retention/changes.go:97-106`) is split so that a new exported `Registry.HoldsCovering(tenant, class)` applies the same tenant-and-class rule, including that a tenant hold with empty `Classes` covers every class. There is no installation-wide hold concept in P04, so none is invented.
- **Tenant IDs are workspace IDs.** P04 compares hold tenants with the reader's workspace (`apps/api/server/retention.go:540-542`, `:446-447`), which supports that reading. It is recorded as verified by reading the code only; a test pins it.
- A session-bound record is **held** when a covering hold names its workspace.
- A refused sign-in (workspace-less) is held whenever **any** covering hold exists, because it cannot be tied to one workspace.
- A day file with no held record is deleted. A day file with held records is **rewritten** with only its held records (temp file, `Sync`, rename, directory sync); the unheld records are deleted on schedule.
- **Held markers** (G6). Each observed hold becomes a marker in `retention.json` (not in a purgeable day file): a unique marker ID (`hm-<16 random hex>`), the hold ID, a snapshot of the hold (owner, reason, tenants, classes, placed time), the run ID and time it was first observed, and the days and files it keeps. A hold ID that P04 reuses after a restart is a new marker; markers are matched by marker ID, never by hold ID alone.
- **Release.** A marker is cleared through P04 only when it was first observed **in the current run** and P04 now shows that hold released. A marker from an earlier run is cleared only by the one-time start flag `-release-access-history-marker=<marker ID>` together with `-release-access-history-reason=<text>`. The flag is refused when no such marker exists, when the reason is empty, or when no marker is pending. Each release is appended to the release audit in `people.log` (marker ID, snapshot, reason, time, run ID), kept as described in §2 (J2). A `hold_released` system record is also written.
- **Logging** (G7). At `Start` and at every purge, each marker is logged (`access_history_held`): marker ID, snapshot, first observed, days beyond the period, records and bytes kept. Held bytes are included in the shortening refusal message and in the size-bound messages.
- Held records past the period stay hidden from the M05 read; the person sees a held note with their own count (§6).

**Clock rule** (F7). The purge uses `s.clock()` only after `setClockBase` has settled it, and is **deferred** (with a `purge_deferred` note and records still hidden by the cutoff, which is reversible) when:
- `leadBeyond()` reports an unaccepted wall-clock lead; or
- at `Start`, the server clock is ahead of the previous run's last record time, persisted in `retention.json` (G9), by more than the **clock gap** setting `access_history_clock_gap_days` (J1), whatever the decision ledger holds. The 24-hour skew tolerance is not used here, because an installation that was simply switched off for a weekend is not a clock fault.
- **Persisted deferral** (J1, root decision). The deferral is written to `retention.json`: the pre-jump reference time (the previous run's last record time) and the server time logged at the refusal. It stays pending across restarts until accepted: a restart without the flag deletes nothing and defers again with the same persisted values.
The second deferral ends only with the one-time flag `-accept-access-history-clock=<RFC 3339 time>`, whose value must equal the **persisted** logged server time (to the minute); it is refused when no deferral is pending. The first ends when the operator accepts the clock with the existing `-clock-base` flag. The `Start` purge runs after the decision-ledger step of `Start` (replay and clock settlement). A forward jump therefore never deletes files at `Start` on an unverified clock.

**Serialization and crash safety** (G10).
- Purge, held-record rewrite and early deletion run **on the writer goroutine** (as a queued control task), so no append can interleave with a delete or rewrite of the same file.
- At `Start`, leftover temp files (`*.tmp`) in `access-history/` are deleted and the directory is synced before any read. A crash between the temp file's `Sync` and the rename therefore leaves the original file intact.
- Damaged lines in a day file that a covering hold keeps are copied byte-for-byte into a held file `YYYY-MM-DD.held-damaged` instead of being dropped by the rewrite (J5): these files pass the same per-file checks, count toward the size bound and the held bytes, are listed against their marker, and are deleted when that marker is released.
- If the oldest days are entirely held and the size bound is reached, nothing is deleted early and the writer stops with the distinct reason `full_held`.

**Purge** (`history.purge(now)`), at `Start` and at each UTC day boundary on a timer, with an hourly retry while overdue or deferred:
1. `cutoff = now − days`.
2. Entries with `at < cutoff` are removed from the index at once; the read never returns them, even if a file still holds them.
3. Every day file whose whole day ends at or before `cutoff` (`day + 24h ≤ cutoff`) is deleted, or rewritten with only its held records; the directory is synced. A `purged` system record notes the range and counts.
4. A deletion, rewrite or directory sync that fails leaves `deletion_overdue {since}` in memory, retried hourly; the M05 read shows it (§6). Hidden records stay hidden.

While the server runs, purges at each day boundary keep every unheld record younger than `days + 1 day`. While it is stopped nothing is deleted; the next `Start` purges before serving (F9). The screen copy says exactly that.

**Size bound.** Before a batch would take the total size past `access_history_max_mib`, the oldest whole day file is deleted early (or rewritten with its held records only), its entries leave the index, and an `early_deletion {before}` record is written with per-user counts (Q5). If the newest day alone exceeds the bound, the writer stops (`recording: stopped`, reason `full`) rather than delete today's records.

**Bound versus period** (F5). At `Start` and at each day-boundary purge the server computes the observed rate (bytes per day over the last 7 kept days) and the **effective coverage** = `max_mib / rate`, capped at `days`. When the effective coverage is below `days`, `Start` logs a warning (`access_history_bound_short`, with both values) and the M05 statement shows the effective coverage (§7). The read carries `effective_days`.

**What the purge never claims.** Copies of the data directory in backups, snapshots or replicas are outside the server's control (`retention-access.md:80`); the statement says so.

## §4 Recording points and their safety

- A helper `s.recordAccess(kind, user, workspace, reason, v, r)` builds the record (device from `r`, handle from `v`) and calls `offer`. It is called **after** the HTTP answer code is chosen and never changes it.
- **Refused sign-in attribution.** `login` resolves `s.dir.userOf(login)` under `s.domain.RLock()` for every non-OK result, existing or not; only a found user produces an offer. Both paths do the same map lookup and the offer never waits, so the answer and its timing do not depend on account existence (T2). A test drives both with the writer blocked.
- **One session-end helper file** (F3, G1, G2). `sessionend.go` already exists (HYB-307: `snapshot`, `endManifest`, `deny`, `sweep`), so the helpers go into a **new file `sessionending.go`**:
  - `s.endSession(v, cause)`: module end, store removal, pickup removal, one `session_ended` record, only when a store session was actually removed (a repeated end records nothing);
  - `s.dropSession(v, cause)`: store removal and record without a module end (the `s.denied` path in `access`);
  - named module-only helpers: `s.rollbackOpened(v)` (the refused sign-in rollback, `signin.go:283-284`), `s.endProbe(v)` (module-only probe sessions: `signin.go:433`, `:443`, `resettable`, `seed.go:160-179`), `s.retryModuleEnd(v)` (the HYB-307 cleanup retries and the denied sweep, `selfend.go:341`, `:393`, `:421`). `Server.endModule` (`selfend.go:195-203`, with its fault seams) moves into `sessionending.go` and becomes `retryModuleEnd`'s body. These record nothing: no store session of a person ends there;
  - the M05 deny keeps `endManifest` (store) and records `ended_by_you` in `denyManifest` (`selfend.go:334`) from `result.ended`, one record per ended session, none for `already_gone` or `not_included`.
  - Every site listed in "Current behaviour" calls one of these helpers: `endOne` and its callers, `endSessions` (administrator departure, role-period end, reset link), the evictions in `openSession` and `probe`, every branch of `access`, the bulk cleanup, `trust.go:323-326` (`Server.sweep`), and logout (which records `signed_out` instead of `session_ended`).
  - `cause` is a typed enum; the compiler finds every call.
  - **Guard test** (`sessionending_test.go`, G2): it type-checks the package with `go/types` and fails if any call resolves to the identity boundary's `EndSession`, `Server.endModule`, or the session store's `drop`, `remove`, `prune`, `endManifest` from a function outside `sessionending.go`. Two calls are allowed **as expiry**: `sessions.prune` at sign-in (`signin.go:267`) and `sessions.sweep` (`selfend.go:403`), which remove only expired sessions or denied verifiers and record nothing. Calls inside methods of the store type itself (`session.go`, `sessionend.go`) are the store implementation and are allowed by receiver type. There are no other per-function exceptions. The guard skips `_test.go` files. Because `go/types` resolves the receiver, `directory.remove` and `directory.prune` are not confused with the store's methods.
- **No new lock order.** `offer` uses only the queue; the index lock is taken only by the writer and the read.
- **Logging.** The existing structured log lines stay. A writer failure logs `access_history_stopped` with the error class and path, never record contents.

## §5 Restart and the rest of the server

- History survives a restart; sessions do not. A handle from an earlier run is never `active` after a restart.
- The decision-ledger states (`isolated`, `stopped`, `full`, operator recovery) never affect recording or the read; regression guards pin it.
- No record is replayed into server state. The history is evidence, never a source of access decisions.

## §6 API: `GET /api/v1/me/access-history`

- **Admission:** `s.self` (cookie, CSRF header, `s.access`), as `readSelf` (`self.go:56-73`). No step-up: it reads the caller's own records and changes nothing.
- **Scope, server-side only:** `user = answer.User()`, current incarnation only (F4). Records are returned when their stored workspace equals `answer.Workspace()`, or when they are refused sign-ins, the only workspace-less kind (F8). Session-bound records from the person's other workspaces are not returned in this session and are not counted (R6).
- **Query:** `limit` 1–100 (default 50); `before` an opaque cursor; `from`, `to` RFC 3339 instants with `from ≤ to`. Any other parameter, a repeated parameter or a bad value is 422 `invalid_request` with `fields`.
- **Cursor** (F1, G11). The read merges two sequences: the caller's (user, session workspace) sequence and the caller's refused sign-in sequence. The cursor is base64url of the pair of positions `{w: n, r: n}` reached in each, each validated as 0–20 digits. It never carries the installation-wide `seq`, a time from another person's record, or a position in any other workspace's sequence. Pages are cut by these positions, so neither another person's activity nor the caller's activity in another workspace changes the cursor or the page.
- **Answer 200,** `Cache-Control: no-store`:

```json
{
  "events": [{"at": "...", "kind": "signed_in", "reason": "", "workspace": "...",
              "session": {"handle": "16hex", "current": true, "active": true},
              "device": {"browser": "chrome", "os": "macos"}, "count": 0}],
  "next": "cursor-or-null",
  "retention_days": 90, "retention_source": "default|start_flag", "period_since": "...",
  "effective_days": 90,
  "available_from": "time-or-null",
  "recording": {"state": "recording|stopped", "since": "time-or-null", "reason": "write_failed|full|null"},
  "notes": [{"kind": "unclean_stop|dropped|damaged|recovered_tail|early_deletion|deletion_overdue|purge_deferred|held|future_file",
             "from": "...", "to": "...", "count": "integer-or-null"}],
  "now": "..."
}
```

- **Note counts** (F2). `count` is the caller's own count where the loss or hold is attributable to a person (`dropped`, `damaged` with a readable user, `early_deletion`, `held`); a note whose window holds no attributable loss of the caller's but an unattributable loss gets `count: null` and reads "some events in this window may be missing". `unclean_stop`, `recovered_tail`, `purge_deferred`, `deletion_overdue` and `future_file` never carry a count. An installation-wide total is never returned.

- `events` newest first by the person's ordinal `n` (F1). `session` is null for kinds without a session. `current` is the caller's own handle; `active` is a live session of the caller now (from `sessions.snapshot`).
- `available_from` is the oldest record still kept for this person in this scope, or null when there is none. It never claims coverage before the history existed.
- `notes` are installation-wide facts with no personal data, returned when their range overlaps the period kept.
- **Codes:** 401 `unauthenticated`; 403 `csrf_refused`; `s.access` codes as mapped; 422 `invalid_request`. None is new.
- **OpenAPI** (F17): `selfPaths()` in `openapiself.go` gains the path `/api/v1/me/access-history` with a `get` operation whose `operationId` is `me.access_history.read`, following `me.sessions.end`; the query schema, the answer schema and enums, and the error responses built with `selfError`. It appears under `x-self` through `openapi.go:40`, pinned by the golden test. Public contract change, covered by the change review.

## §7 Console (M05 sessions view)

- **Placement.** A new `<section aria-labelledby="mysec-history-title">` after the sessions section in the sessions view, at full width. It is separate from "Recent session ends" (`EndsRegion`), which keeps its title and its in-memory operations. The limits panel loses the "not available" note.
- **Heading:** "Recent access history" (KO 최근 접근 기록, AR سجل الوصول الأخير), the terms the current note already uses (`live-copy.ts:180`).
- **Controls:** period select (Last 24 hours, Last 7 days, Last 30 days, Everything kept), sent as `from`; the workspace shown as a fixed fact ("Workspace: {id}") with the note that other workspaces' history is shown when signed in there; "Show more" while `next` is non-null.
- **Table** (stacked labelled rows below 600 px, following the HYB-332 pattern): Time (local, with seconds) | Event (label plus reason) | Session ("This session", "Signed in now", or the reference `ltr(handle)`, or "—") | Device ("Chrome on macOS"; "Unknown device").
- **Statements, always visible under the heading:**
  - retention: the copy below with `{days}` and, when `retention_source` is `start_flag`, "(set for this installation)";
  - effective coverage (F5), only when `effective_days < retention_days`: the "Size limit" line below;
  - held, deferred and overdue notes when present;
  - coverage: "History available from {time}." or "No history is kept yet.";
  - "Network addresses and locations are not recorded."
- **States** (`:91`): loading; empty period ("No access events in this period."); could not load (error copy with request reference, never shown as empty); recording stopped; each note kind; deletion overdue.
- **Revocation** (`:91`): on a 401 or revoked state, the region's rows and cursor are cleared with the rest of the page.
- **Layout.** Service-screen table and status patterns at full viewport width; no decorative borders or accent strips (`service-screen-style.md`). `ui-designer` gives direction before implementation and reviews the rendered EN, KO and AR screens.

**Copy** (EN for TW review; KO and AR are AI drafts pending native-language review):

| Key | EN | KO | AR (draft) |
| --- | --- | --- | --- |
| Heading | Recent access history | 최근 접근 기록 | سجل الوصول الأخير |
| Retention (F9) | Kept for {days} days. Older events are hidden at once. While the server is running they are deleted from it within one day; if the server is stopped, they are deleted when it next starts, before it accepts sign-ins. Events under a legal hold are kept until the hold is released. Copies in backups of the server's data folder remain until those backups expire. | {days}일 동안 보관합니다. 그보다 오래된 기록은 즉시 숨깁니다. 서버가 실행 중이면 하루 안에 서버에서 삭제하고, 서버가 중지되어 있으면 다음 시작 시 로그인을 받기 전에 삭제합니다. 법적 보류 대상 기록은 보류가 해제될 때까지 보관합니다. 서버 데이터 폴더의 백업에 있는 사본은 해당 백업이 만료될 때까지 남습니다. | يُحتفَظ به لمدة {days} يومًا. تُخفى الأحداث الأقدم فورًا. أثناء تشغيل الخادم تُحذَف منه خلال يوم واحد؛ وإذا كان الخادم متوقفًا، تُحذَف عند تشغيله التالي قبل أن يقبل تسجيلات الدخول. تُحفَظ الأحداث الخاضعة لحجز قانوني حتى رفع الحجز. تبقى النسخ الموجودة في النسخ الاحتياطية لمجلد بيانات الخادم حتى انتهاء صلاحية تلك النسخ. |
| Size limit (F5) | Because of this server's size limit, about the last {effective} days are currently kept. | 이 서버의 크기 한도 때문에 현재 최근 약 {effective}일의 기록만 보관됩니다. | بسبب حدّ الحجم في هذا الخادم، يُحتفَظ حاليًا بنحو آخر {effective} يومًا فقط. |
| Coverage | History available from {time}. | {time}부터의 기록이 있습니다. | السجل متاح منذ {time}. |
| Minimisation | Network addresses and locations are not recorded. | 네트워크 주소와 위치는 기록하지 않습니다. | لا تُسجَّل عناوين الشبكة ولا المواقع. |
| Empty | No access events in this period. | 이 기간에는 접근 기록이 없습니다. | لا توجد أحداث وصول في هذه الفترة. |
| Load error | We could not load your access history. Try again. Request reference: {ref} | 접근 기록을 불러오지 못했습니다. 다시 시도해 주세요. 요청 참조: {ref} | تعذّر تحميل سجل الوصول. حاول مرة أخرى. مرجع الطلب: {ref} |
| Recording stopped | Since {time} this server has not been able to record access events. Events after that are missing. | {time} 이후 이 서버가 접근 기록을 저장하지 못하고 있습니다. 그 이후의 기록은 없습니다. | منذ {time} لا يستطيع هذا الخادم تسجيل أحداث الوصول. الأحداث بعد ذلك مفقودة. |
| Unclean stop | The server stopped unexpectedly around {time}. Events just before that may be missing. | {time} 무렵 서버가 예기치 않게 중지되었습니다. 그 직전의 기록이 빠졌을 수 있습니다. | توقّف الخادم بشكل غير متوقّع نحو {time}. قد تكون الأحداث التي سبقت ذلك مباشرة مفقودة. |
| Dropped (own count, F2) | {count} of your access events between {from} and {to} could not be recorded. | {from}부터 {to} 사이의 내 접근 기록 {count}건을 저장하지 못했습니다. | تعذّر تسجيل {count} من أحداث الوصول الخاصة بك بين {from} و{to}. |
| Possibly missing (no count, F2) | Some events between {from} and {to} may be missing. | {from}부터 {to} 사이의 일부 기록이 빠졌을 수 있습니다. | قد تكون بعض الأحداث بين {from} و{to} مفقودة. |
| Held (F6) | {count} of your events older than {days} days are kept because of a legal hold. They are not shown here. | {days}일보다 오래된 내 기록 {count}건을 법적 보류 때문에 보관하고 있습니다. 여기에는 표시하지 않습니다. | يُحتفَظ بـ{count} من أحداثك الأقدم من {days} يومًا بسبب حجز قانوني، ولا تُعرَض هنا. |
| Deletion deferred (F7) | Deletion of old events is paused until the server's clock is confirmed. Older events stay hidden. | 서버 시각이 확인될 때까지 오래된 기록 삭제를 멈췄습니다. 오래된 기록은 계속 숨깁니다. | أُوقِف حذف الأحداث القديمة مؤقتًا حتى يُؤكَّد وقت الخادم. تبقى الأحداث الأقدم مخفية. |
| Damaged | {count} stored events could not be read and are not shown. | 저장된 기록 {count}건을 읽을 수 없어 표시하지 않습니다. | تعذّرت قراءة {count} من الأحداث المخزّنة ولا تُعرَض. |
| Early deletion | Events before {time} were deleted early because the access history reached its size limit. | 접근 기록이 크기 한도에 도달해 {time} 이전 기록을 일찍 삭제했습니다. | حُذفت الأحداث السابقة لـ{time} مبكرًا لأن سجل الوصول بلغ حدّ الحجم. |
| Deletion overdue | Events older than {days} days are hidden, but the server has not yet been able to delete them. | {days}일보다 오래된 기록은 숨겼지만 서버가 아직 삭제하지 못했습니다. | الأحداث الأقدم من {days} يومًا مخفية، لكن الخادم لم يتمكّن بعد من حذفها. |
| Not shown | Expired sessions are not listed as events. | 만료된 세션은 기록에 표시되지 않습니다. | لا تُدرَج الجلسات المنتهية الصلاحية كأحداث. |

Event labels: Signed in / 로그인 / تسجيل الدخول; Sign-in refused: wrong password, account locked, access refused / 로그인 거부: 비밀번호 오류, 계정 잠김, 접근 거부; Password confirmed / 비밀번호 재확인; Password confirmation failed / 비밀번호 재확인 실패; Signed out / 로그아웃; Session ended: by you, password changed, too many wrong passwords, access changed, by an administrator, session limit, sign-in rules changed, bulk revocation; Password changed / 비밀번호 변경; "{count} more refused sign-ins". "Password confirmed" follows the glossary row `password-confirmation` (`product-glossary.md:66`). AR plural forms for every count.

**Glossary rows** (KO and AR marked as AI drafts): `access-history` (최근 접근 기록 / Recent access history / سجل الوصول الأخير): the person's own record of sign-ins, refused sign-ins, password confirmations, sign-outs and session ends on this server, kept for the installation's period. Not an administrator audit, not proof of who used the device, not the list of active sessions, and not the "Recent session ends" operations. Also `access-history-period` (보관 기간) and `access-history-note` (기록 누락 안내).

# Threats and abuse cases

| # | Case | Handling |
| --- | --- | --- |
| T1 | Reading another person's history | The user comes only from the session; no parameter names a user; unknown parameters are refused; a two-person test checks nothing crosses |
| T2 | Account enumeration through sign-in timing or answers | Same answer code; recording is a non-blocking enqueue after the answer is chosen, behind the same map lookup for every login; test with the writer blocked |
| T3 | Injection of text into the victim's view | Nothing user-supplied is stored except the classified device family; the requested workspace of a failed sign-in is not stored |
| T4 | Flooding refused sign-ins to evict history | Global sign-in limiter; per-person daily cap of 100 records plus an aggregate count; early deletion removes the oldest day only and says so |
| T5 | Secrets reaching disk | Allowlisted fields only; a test scans every byte written for the test passwords, cookies, CSRF tokens, verifiers and login IDs |
| T6 | A second process or a symlinked directory | Exclusive lock; `checkData` rules; `Start` refused |
| T7 | Tampering by someone with write access to the data directory | CRC detects accidental damage only; deliberate edits are not detectable locally (R1) |
| T8 | Silent loss | Every loss path writes or shows a note (dropped, stopped, unclean stop, damaged, early deletion); the model test checks that each offered event is present, purged or covered by a note |
| T9 | Retention bypass by shortening at restart | Refused without the one-time acceptance and an impact count |
| T10 | Cross-workspace exposure | Session-bound records carry their session's workspace; only that workspace's records and refused sign-ins are returned (F8) |
| T11 | Learning about other people through the cursor, page boundaries or note counts (F1, F2) | The cursor is the caller's own ordinal; pages are cut by that ordinal; notes carry only the caller's own counts or none. A test interleaves another person's events and checks that the caller's cursors, pages and notes are byte-identical with and without them |
| T12 | A new holder of a reused fixed user ID reads the earlier holder's history (F4) | Incarnation markers at activation and runtime add; the read starts after the latest marker |
| T13 | Deleting held evidence (F6) | P04 holds on `identity` stop deletion; held markers persist across restarts and are never assumed released |
| T14 | A wrong clock deletes valid records (F7) | Purge after the clock settles; deferred on an unaccepted lead or an unverified forward jump |

# Tests

- **Module unit and fault tests** (`modules/accesshistory`):
  - codec round trip; CRC rejects every single-byte change; over-length, unknown kind and unknown enum lines are damaged;
  - device classifier table, including empty, oversized and hostile User-Agents;
  - fault injection with a `faultyFile` seam as in `decisionledger/faults_test.go`: write error, short write, disk full, `Sync` error, rename and directory-sync errors for `retention.json`, remove and rewrite errors during purge, lock contention, unsafe directory mode, symlink, foreign owner (Unix);
  - per-file checks (F12): a symlinked, hard-linked, group-readable or foreign-owned day file, `retention.json` or `lock` refuses `Start` (Unix); on Windows a no-op directory sync does not refuse `Start`; non-current handles are closed before a delete or rewrite (a test that fails if a handle is still open at delete);
  - `seq` continuation (F14): delete every day file, reopen, and the next `seq` is above the one in `retention.json`;
  - torn tail truncation and the `recovered_tail` record; damaged middle line; future-dated file;
  - writer: queue overflow counts per user and the `dropped` record; stop after a write error; nothing indexed before `Sync`; unclean stop detected on reopen;
  - `Close` (F11): records still queued after the 2 s drain and offers made after `Close` began are counted per user into one `dropped` record written before `run_stopped`; a server test starts `Close` while a handler offers and checks the order, and that `Close` runs only after `srv.Shutdown` returned.
- **Retention with a fake clock:**
  - the cutoff instant (a record exactly at `cutoff` is kept, one millisecond older is hidden);
  - the day boundary purge leaves no file with a record older than `days + 1 day`;
  - a failed delete gives `deletion_overdue`, the hourly retry clears it;
  - a clock moved backwards appends to an earlier day file and never purges early;
  - lengthening keeps deleted records deleted and `available_from` truthful;
  - shortening refuses `Start` with correct counts; `-accept-access-history-shortening=N` applies only when N equals the new period, a different N refuses, and the flag without a pending shortening refuses (F10);
  - size bound: early deletion of the oldest day, then the `full` stop when one day exceeds the bound;
  - bound versus period (F5): a rate that makes `max_mib / rate` shorter than `days` logs the warning at `Start` and sets `effective_days`;
  - holds, round 2 (G6–G8, G10): a tenant hold with empty `Classes` holds identity records; a hold on another tenant does not; refused sign-ins are kept while any covering hold exists; a stale release flag (unknown marker, empty reason, nothing pending) refuses `Start`; a hold ID reused by P04 after a restart creates a new marker and does not clear the old one; a P04 release clears only markers first observed in the current run; the release audit survives purges; the marker log line carries the snapshot, days beyond the period, records and bytes; held bytes appear in the shortening and size messages; damaged lines under a hold go to the held file; all-held oldest days at the bound stop with `full_held`; a crash between the temp `Sync` and the rename (fault seam) leaves the original file, and the leftover temp file is deleted at `Start`; purge and appends never interleave (race test);
  - round 4: adds and activations racing a `people.log` compaction (`-race`) never lose a marker and never leave a floor missing after reopen (K1); a damaged middle line in `people.log` refuses `Start` with `ErrConfig` naming the file and line, and a torn tail is truncated and `Start` succeeds with every answered marker present (K2);
  - round 3: after a 503 from the marker write, the same activation link and the same invite link still succeed, and nothing was bootstrapped, committed or added; `floor_seq` taken while the writer queue is blocked is above every queued `seq`, and those records stay hidden after they are written (J3); `people.log` faults (write, `Sync`, torn tail, damaged line, per-file checks); compaction drops an entry only when no kept record lies below its floor; audit entries survive until the period plus 7 years and then become counts (fake clock) (J2); one test per refusal value in the J4 mapping; `*.held-damaged` files counted in size and held bytes, checked like day files, and deleted on release (J5); defer, restart without the flag: nothing deleted and the same deferral pending; the flag checked against the persisted time; a gap below `access_history_clock_gap_days` does not defer (J1);
  - clock, round 2 (G9): the forward-jump check uses the previous run's last record time from `retention.json`, also when the decision ledger has stamps; `-accept-access-history-clock` with a mismatching value or with nothing deferred refuses `Start`;
  - cursor per (user, workspace) (G11): the caller's activity in another workspace does not change the cursor or pages;
  - `Close` order (G12): a background worker ending a session during `Close` is recorded before `run_stopped`, and an offer after the history closed is counted and logged;
  - legal holds (F6): a P04 hold on `identity` for workspace A keeps A's expired records (the file is rewritten with only them) and deletes workspace B's; the held marker survives a restart in which P04 no longer has the hold, and the records stay; `-release-access-history-marker=<marker ID>` with a reason releases it and the next purge deletes them; a hold on another class does nothing; the person sees the held note with their own count;
  - clock (F7): load and purge run after `setClockBase`; with `leadBeyond` true no file is deleted and `purge_deferred` is shown; a forward jump beyond the tolerance at `Start` with no decision-ledger stamps defers deletion while records past the cutoff stay hidden; accepting the clock with `-clock-base` ends the deferral;
  - deletion timing while stopped (F9): advance the fake clock past the period while the server is stopped; the next `Start` deletes before the listener serves.
- **Property and model test** (`modules/accesshistory/property_test.go` and `apps/api/server/accesshistory_property_test.go`). Random sequences of offers for several users and workspaces, clock advances, purges, clean and unclean restarts, setting changes and injected faults, compared with a model. Invariants:
  - H1: a read returns only the caller's records of the current incarnation, in the caller's session workspace, plus the caller's refused sign-ins;
  - H2: no returned record is older than the cutoff;
  - H3: every record whose `Sync` succeeded is present after reopen unless purged or deleted early;
  - H4: no record is shown before its `Sync`;
  - H5: pages concatenate to the filtered list, newest first, without duplicates, under concurrent appends;
  - H6: `seq` strictly increases across restarts;
  - H7: every offered event is present, purged, held, or covered by a note;
  - H8: the caller's cursors, pages and notes are identical whether or not other people's events are interleaved (F1, F2, T11);
  - H9: no held record is ever deleted while its hold or persisted marker exists (F6).
- **Server tests** (`accesshistory_test.go`):
  - one test per recording point, refusal reason and cause in §1 and §4, including that unknown logins and store overflow record nothing, and that every failed sign-in carries a device family (F15);
  - the session-end guard test (F3, G2): a direct `EndSession`, `endModule`, `drop`, `remove`, `prune` or `endManifest` call added outside `sessionending.go` fails it (a planted-violation test proves the guard fires); the M05 deny records exactly one `ended_by_you` per `result.ended` entry and none for `already_gone` or `not_included` (G1); each `access` branch records its mapped cause, the denied branch records nothing new, and an expiry refusal records nothing (G3); one test per site in "Current behaviour", including administrator departure, role-period end (`role_period_ended`), reset link (`password_reset`), both evictions, each `access` branch, bulk cleanup, the M05 deny and logout;
  - incarnation (F4, G4): activate `EmptyAdmin` after a restart and see none of the earlier holder's records; a runtime-added person starts empty; a seeded person keeps history across a restart; with the writer stopped the marker is still written to `retention.json` and the floor hides earlier records; with the `retention.json` write failing, the add or activation answers 503 and changes nothing;
  - other-workspace session events (F8): a person with sessions in workspaces A and B sees A's session events only in A and B's only in B, and refused sign-ins in both;
  - the answer code and the absence of waiting with the writer blocked, for existing and unknown logins (T2);
  - the read: two people, two workspaces, `current` and `active`, cursor, period filter, the parameter table, `no-store`, CSRF, 401 after the session ends;
  - restart on the same data directory: history kept, old handles not active, notes shown;
  - decision-ledger states never affect recording (regression guards);
  - the `x-self` OpenAPI golden test;
  - the secrets scan (T5).
- **Concurrency** (`accesshistory_race_test.go`, `go test -race`): sign-ins, refused sign-ins, ends, reads and purges together; `go test -race ./modules/accesshistory ./apps/api/server -run 'AccessHistory'` and the full packages once.
- **Scale** (F5): load files at the **maximum** bound (1024 MiB) and at the default (256 MiB), and record load time and index memory in `verification.md`; a 100k-person spread for read latency. If the maximum takes more than 30 s to load or more than 1 GiB of index, the maximum is lowered to the largest measured size that meets both, and Q5 is updated with the measured numbers.
- **Mutation targets** (HYB-293 script pattern, cleaning the session `GOCACHE` per batch). Each must be killed:
  - the cutoff comparison and the day-file selection;
  - the CRC check;
  - the user filter and the workspace filter in the read;
  - index-after-`Sync` ordering;
  - the `seq` continuation at load;
  - the torn-tail offset;
  - early deletion picking the oldest day;
  - the shortening guard;
  - the unknown-login skip;
  - overflow counting and note emission;
  - the cursor comparison, the use of `n` rather than `seq`, and the `limit` bounds;
  - the incarnation filter;
  - the hold check and the rewrite that keeps held records;
  - the clock deferral;
  - the shortening flag value comparison;
  - the helper's "only when a store session was removed" condition;
  - the per-person cap;
  - the device default `unknown`.

  A surviving mutant is traced to every reader, including traces, before any code is deleted.
- **Console unit tests** (`history-model.test.ts`, `admin.test.ts`): the parser without casts; empty versus error versus stopped mapping; each note; AR plural forms; the period select to `from`; "Show more" stops at a null cursor.
- **Commands to record** in `verification.md`: `go vet ./...`, `go test ./...`, the race runs; console `npm run typecheck`, `npm test`, `npm run build`; the line-count check (no maintained file at 1,500 lines or more; `Live.tsx` stays near 512).

# Browser QA

- **New runner `qa306.mjs`** in the session scratchpad, on the `qa307.mjs` harness. Each check runs first against a build of base `73745d6` and must **FAIL** there:
  - Q1: the sessions view has a "Recent access history" region, separate from "Recent session ends", with the retention, coverage and minimisation statements. Base shows the "not available" note.
  - Q2: browser A signs in; browser B fails once with a wrong password and then signs in; A's history shows A's sign-in ("This session"), the refused sign-in (wrong password) and B's sign-in ("Signed in now"), each with a device family.
  - Q3: A ends B through "End all other sessions"; the history gains "Session ended: by you" and the "Recent session ends" operation still appears in its own region.
  - Q4 (with `-seed sample`, G5): restart the API on the same data directory, once cleanly and once with SIGKILL; after signing in again, the earlier events are still listed, none is "Signed in now", and only the killed run shows the unclean-stop note.
  - Q5: start with `-access-history-days 30`; the statement shows 30 days "set for this installation". Restart with 10 days: `Start` is refused with the impact message; with `-accept-access-history-shortening=9` it is refused again; with `=10` it starts and shows 10 days.
  - Q6: stubbed reads for recording stopped, dropped with an own count, possibly missing (no count), damaged, early deletion, held, deletion deferred, deletion overdue, the size-limit line, an empty period and a failed load (not shown as empty); "Show more" paging.
  - Q7: EN, KO and AR at 1440 and 390 px, RTL, no horizontal scroll, keyboard-only use of the filter and "Show more", no decorative or coloured edge borders.
- The purge itself is proven by the fake-clock Go tests; the browser cannot move the server clock.
- **Regression** in EN, KO and AR, results recorded: `qa307.mjs` (end other sessions) and `qa271on307.mjs` (M05 overall), from `/private/tmp/claude-501/-Users-home-project-hybrid-billing/ff7545a1-604d-4178-88ba-25b4b88aab02/scratchpad/`.

# Dependencies

- HYB-307 (merged): its session-end calls move into `endSession`/`dropSession` (F3); its contract and the "Recent session ends" region do not change, and `qa307.mjs` must still pass.
- P04 retention (`retention.go`): one read-only hold query is added (F6); no change to P04 behaviour.
- HYB-298 decision ledger: patterns only (directory checks, lock, fault seams); no code change.
- HYB-304 (M05, Todo) touches the same page; whichever merges second rebases.

# Residuals

- **R1** (F16, corrected). No tamper evidence: someone with write access to the data directory can edit or remove records undetected. The first draft said a hash chain was incompatible with pruning; that was wrong, because a chain per day file is compatible with whole-file deletion (a held-record rewrite would start a new chain). It is not built in this ticket; owner question Q10.
- **R2.** Records in the last batch before a crash can be lost; the unclean-stop note says so but cannot count them.
- **R3.** No administrator review of access ("비정상 로그인 감사", `retention-access.md:65`). A removed person's history is kept until the period ends and nobody can read it.
- **R4.** P04 retention policy revisions do not govern this history, and the P04 `identity` class row is unchanged. P04 legal holds **do** stop deletion (F6). P04 holds are in memory, so a hold placed and lost in a run where no purge observed it is not protected; held markers cover only holds a purge has observed.
- **R5.** Device detail is coarse; no network address is recorded, because the server is loopback-only and has no trusted proxy configuration.
- **R6.** History of the person's other workspaces is shown only when signed in to that workspace.
- **R7.** A shortening is authorised by the installation operator at start, not by the P04 two-person approval.
- **R8.** Durability on local disks only (HYB-298 A5); Windows permissions rely on the installer's ACL (HYB-298 R8).
- **R9.** KO and AR copy are AI drafts until native-language and TW review.
- **R10.** The password set through an activation or reset link, and administrator-initiated resets, are not recorded as events; the sessions a reset link ends are recorded with cause `password_reset`.
- **R11.** The file-lock lines duplicate `decisionledger` rather than sharing a package.
- **R12** (G5). With `-seed none`, each restart's re-activation of the bootstrap administrator writes an incarnation marker, so that administrator's earlier history is hidden after every restart.
- **R13** (G12). An offer made after the history closed is counted and logged only; it is not shown on M05.

# Owner questions (root defaults ship until decided)

| # | Question | Default shipped |
| --- | --- | --- |
| Q1 | Which events make up "최근 접근"? The spec does not list them | Sign-ins, refused sign-ins for existing logins, password confirmations and their failures, sign-outs, session ends with cause, password changes (§1) |
| Q2 | Retention period | 90 days, installation setting `access_history_days`, 1–1095. No legal minimum or maximum was assessed (`retention-access.md:69`) |
| Q3 | Device information level | Browser and OS family only; no IP, network, location or full User-Agent |
| Q4 | When the history cannot be written, should sign-in be refused? | No. Sign-in continues and M05 shows "recording stopped" |
| Q5 | When the size bound is reached: delete the oldest day early, or stop recording? And the bound itself | Delete the oldest day early and say so; stop only if one day exceeds the bound. Default 256 MiB, maximum 1024 MiB. Unverified estimate at about 150 bytes per line: 256 MiB ≈ 1.8 million records, 1024 MiB ≈ 7 million; index memory and load time are filled in from the scale measurement (F5), and the maximum is lowered if the measurement fails the 30 s / 1 GiB limits |
| Q6 | Who authorises a shorter period? | The operator, with the one-time start flag after the impact message |
| Q7 | Should legal holds stop this purge? | Yes (root decision on F6): P04 holds covering `identity` stop deletion. Held markers are persisted in `retention.json` and **outlive P04's in-memory holds**: after a restart a marker stays until the operator releases it by marker ID with a reason (G6, G7) |
| Q8 | Delete a person's history when they are removed? | No; it ages out with the period (`retention-access.md:76`, security investigation) |
| Q9 | Engineering constants: refused sign-in cap 100 per person per day, queue 4,096, batch 256 or 50 ms, page 50 (max 100), `Close` drain 2 s | As listed; they can become installation settings with these defaults |
| Q10 | Tamper evidence: a per-day hash chain (R1) | Not in this ticket |
| Q11 | How long is the hold release audit kept? | At least the access history period plus 7 years per entry, then compacted into yearly counts (J2) |
| Q12 | Gap after which a start is treated as a forward clock jump | 7 days, installation setting `access_history_clock_gap_days`, 1–365 (J1) |

Q2 is recorded as an owner decision pending, with the default shipped, in `verification.md` and in the register entry.

# Verification and review sequence

1. Security and data design review of this plan by `independent-reviewer`; findings folded in and re-reviewed until it approves.
2. Visual direction from `ui-designer` (`visual-spec.md`); TW review of the copy table and glossary rows.
3. Implementation on the ticket branch within the allowed files.
4. Tests, fault and fake-clock runs, race runs, scale measurement and mutation runs, each recorded with its command.
5. Browser QA: `qa306.mjs` fails on base and passes on the branch; the regressions pass.
6. Independent change review with a security and data focus; fixes re-reviewed.
7. PR, authorized merge with read-back, then ticket closure only after the merge is confirmed.

# Review history

## Review round 1 (security and data design review): CHANGES REQUIRED (2 High, 6 Medium, Lows)

| # | Severity | Finding | Resolution in this revision |
| --- | --- | --- | --- |
| F1 | High (root decision) | The cursor carried the installation-wide `seq`, revealing other people's activity | The cursor is the person's own ordinal `n`; `seq` is internal (§2 Index, §6 Cursor); T11, H8 |
| F2 | Medium | Notes carried installation-wide counts | Per-person counts where attributable, otherwise no count and "some events may be missing" (§2 Writer, §6 Note counts, copy) |
| F3 | High | Session ends were recorded only through `endOne`; many sites end sessions directly | One helper `endSession`/`dropSession` in `sessionend.go` for every site, causes `role_period_ended` and `password_reset` added, AST guard test with a named allow-list; `periods.go`, `admin.go`, `selfend.go` allowed (§1, §4, Tests) |
| F4 | Medium | "An ID is never reused" was wrong for seeded and bootstrap IDs | Claim corrected; incarnation markers at activation and runtime add hide earlier records (§1, Current behaviour, T12, tests) |
| F5 | Medium | The size bound could silently undercut the period; memory at the maximum unmeasured | Rate-based effective coverage with a `Start` warning and an on-screen line; maximum lowered to 1024 MiB pending measurement; numbers in Q5 (§3, Tests) |
| F6 | Medium (root decision, option a) | Legal holds were not honoured | P04 `identity` holds stop deletion via a read-only query; held day files rewritten with held records only; persisted held markers, operator release flag; held note (§3, T13, H9, tests) |
| F7 | Medium | Load and purge could run before the clock settled; forward jumps undefined | Load after `setClockBase`; no purge while `leadBeyond`; forward jump at `Start` without ledger stamps defers deletion with a note (§2, §3, T14, tests) |
| F8 | Medium | Session-bound records lacked the session's workspace | Every session-bound record stores its session's workspace; only refused sign-ins are workspace-less; test (§1, §6, T10) |
| F9 | Low | The copy promised deletion within a day even while stopped | Copy says: within a day while running, otherwise at the next start before serving (§3, §7) |
| F10 | Low | A bare boolean shortening flag could be left set | `-accept-access-history-shortening=N`, mismatch and no-pending refused (§3, tests, QA Q5) |
| F11 | Low | Records queued at `Close` and late offers were lost silently | Counted per user into a `dropped` record before `run_stopped`; `Close` confirmed after HTTP shutdown (§2, tests) |
| F12 | Low | Per-file checks, Windows directory sync, open handles at delete | Decision-ledger per-file checks; Windows no-op sync never refuses `Start`; non-current handles closed before delete (§2, tests) |
| F13 | Low | Source vocabulary and settings grouping | `default`/`start_flag`; group added to `allSettings` and `withDefaults` (§3, §6) |
| F14 | Low | `seq` could repeat after files were deleted | Highest `seq` persisted in `retention.json` (§2, test) |
| F15 | Low | Device not classified on every failure; other refusal paths undefined | Device for every failed sign-in; reasons `credentials_changed`, `session_unavailable`; overflow 503 not recorded (§1, tests) |
| F16 | Low | R1 rationale wrong | Corrected; Q10 with default "not in this ticket" |
| F17 | Low | Stale line references and OpenAPI naming | Line references rechecked at `73745d6`; OpenAPI via `selfPaths()` with `operationId` `me.access_history.read` (§6) |

## Review round 2: CHANGES REQUIRED (1 High, several Medium); F1, F2, F5, F8–F17 pass

| # | Severity | Required | Resolution in this revision |
| --- | --- | --- | --- |
| G1 | Medium | `sessionend.go` already exists; record `ended_by_you` from `result.ended` | New file `sessionending.go`; `denyManifest` records one per `result.ended` entry, none for `already_gone`/`not_included`; test (§4, Tests) |
| G2 | Medium | Module-only ends as named helpers; guard by method resolution with prune and sweep as expiry; no other exceptions | `rollbackOpened`, `endProbe`, `retryModuleEnd` in `sessionending.go`; `go/types` guard over `EndSession`, `endModule`, `drop`, `remove`, `prune`, `endManifest`; prune and sweep allowed as expiry; planted-violation test (§4) |
| G3 | Medium | Map `access` branches by reason; expiry records nothing | Branch-to-cause table; expiry records nothing; test per branch (§1, Tests) |
| G4 | Medium | Incarnation marker synchronous before the answer, or fail closed; in-memory floor | Written to `retention.json` with temp, `Sync`, rename before success; 503 on failure; floor set in the same step; tests with the writer stopped and the write failing (§1) |
| G5 | Low | `-seed none` residual; QA names the seed | R12; QA Q4 uses `-seed sample` |
| G6 | High | Unique marker IDs with hold snapshots; P04 release only for markers of this run; release by marker ID with reason; audit outside day files; tests | Done as written (§3 Legal holds, flags, Tests) |
| G7 | Medium | Log markers at `Start` and every purge; held bytes in messages; Q7 wording | Done (§3, Q7) |
| G8 | Medium | Reuse P04's coverage predicate; no installation-wide hold; refused sign-ins held by any covering hold; confirm tenant IDs are workspace IDs | `HoldsCovering` reuses `hold.covers`; installation-wide removed; tenant = workspace supported by `retention.go:540-542` (code reading, pinned by a test) (§3) |
| G9 | Medium | Compare with the previous run's last record in `retention.json`; value-bound acceptance flag; drop the "no stamp" qualifier; purge after the decision-ledger step | Done (§3 Clock rule, §2 `Close`, Tests) |
| G10 | Medium | Serialize purge with the writer; delete temp files at `Start`; held damaged lines; `full_held`; crash test between `Sync` and rename | Done (§3 Serialization, Tests) |
| G11 | Medium (root decision) | Ordinal per (user, workspace); refused sign-ins on their own sequence | Done; two-sequence cursor (§2 Index, §6 Cursor, Tests) |
| G12 | Low | Stop background workers before closing history; count and log late offers; soften the claim | WaitGroup on `every`; late offers counted and logged; claim limited; R13 (§2 `Close`) |

This revision was reviewed in round 3.

## Review round 3: close; CHANGES REQUIRED on J1, J2 (Medium) plus lows

| # | Severity | Required | Resolution in this revision |
| --- | --- | --- | --- |
| Marker order | Medium (reviewer answer) | Marker before `Bootstrap` (`signin.go:473`), before `passwords.Commit` (`:421`), before the identity add (`admin.go:236-240`); links still usable after a 503; lock order; lock separate from purge work | Done (§1 Order and Lock order, Tests) |
| J1 | Medium (root decision) | Persist the deferral; flag checked against the persisted time; restart without the flag deletes nothing; new changeable clock-gap default | Deferral persisted in `retention.json`; `access_history_clock_gap_days` 7 days (1–365); Q12; tests (§3) |
| J2 | Medium (root decision) | Incarnations and release audit in their own append-only, checksummed, fsynced file; compaction; audit bound; 100k-user measurement | `people.log` with compaction at purge, period plus 7 years then counts (Q11), measurement recorded (§2) |
| J3 | Low | `floor_seq` allocated atomically from the writer's counter | Done; test with a blocked queue (§1) |
| J4 | Low | Exact refusal values per cause | Mapping table by `identity.Reason` and `authpolicy` values; `ReasonSession` and `ReasonStale` record nothing; test per value (§1) |
| J5 | Low | `*.held-damaged` files deleted on release, counted, checked | Done (§3) |
| G2 notes | Low | Move `endModule` into `sessionending.go`; guard skips `_test.go` | Done (§4) |

Round 3 items: all PASS (reviewed in round 4).

## Design review round 4: APPROVE pending K1–K2 (root text check)

| # | Required | Resolution in this revision |
| --- | --- | --- |
| K1 | `people.log` compaction holds the history-state lock across the whole read, rewrite and rename; race test with adds during compaction | Done (§2 `people.log`, §1 Lock order, Tests) |
| K2 | A damaged middle line in `people.log` refuses `Start` with `ErrConfig` naming the file and line; a torn tail is truncated; tests for both | Done (§2 `people.log`, Tests) |

The approval takes effect when root's text check confirms K1 and K2. Next: the `ui-designer` visual spec, then the implementation go-ahead from root.

Root text check (2026-11-19): K1 and K2 confirmed in §1 Lock order, §2 people.log (lines 183, 207–208) and the tests (line 442). Design review round 4: APPROVE.


# Combined verification repair decisions

The resumed implementation review found and repaired clock-deferral bypass in size cleanup, non-durable observed hold markers before day rewrite, inconsistent read-state synchronization, and loss attribution across workspace/incarnation boundaries. Independent design and actual-change re-review covered these corrections.

- Destructive purge and size cleanup both require accepted time. A full history that cannot safely append while deletion is deferred refuses startup or stops recording; it does not delete protected data to make room.
- Newly observed hold markers and damaged-file associations are persisted and synced before source files change. A failed marker snapshot prevents destructive cleanup.
- Writer-owned file operations publish read-visible maintenance snapshots under the history lock. Sequence snapshots use the offer lock without retaining it across file I/O; person-state operations preserve the documented lock order.
- Internal `Record.src` stores the earliest source sequence of a summarized refused-sign-in count. Delayed predecessor summaries cannot become successor events or exact losses. `Record.p` attribution keys encode `[user, workspace, firstSourceSeq, lastSourceSeq]`; they remain bounded by source-record size and describe original events rather than the later summary record.
- Exact loss/hold counts require matching workspace and a source range entirely above the current incarnation floor. Wholly predecessor ranges are suppressed; legacy or mixed-incarnation ambiguity is masked as unknown without predecessor-only timestamps. Refused sign-ins remain workspace-less within the same user's current incarnation.
- Hidden predecessor rows neither allocate nor restore the successor's per-scope cursor ordinal, including queued rows and restart reload.
- The test fixture advances its injected wall time under `serverClock.mu` before its wall-source mutex. This makes a synthetic step atomic across the clock's paired reads; product clock behavior is unchanged. The original eight-hour regression failed before this fixture repair and passed 50 consecutive runs after it.

See `verification.md` for actual commands, failures, mutation evidence and remaining limits.

- `retention.HoldsCovering` returns detached copies of its three mutable scope slices. A read snapshot cannot mutate an authoritative legal hold; the filter/sort/released-state contract is unchanged. This repair received independent design and actual-change review with a fail-first mutation-isolation regression.
