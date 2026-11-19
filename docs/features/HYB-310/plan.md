---
type: Plan
title: HYB-310 P02 separate API, module and channel rights
description: Replaces installation administration on every P02 integration command and read with eight separate installation rights (API clients; module administration, approval and stop; channel administration, approval and stop; event reprocessing), withholds each P02 area from callers without its right, rechecks at approval time that the proposer or requester still holds the starting right, lets authors withdraw their own pending work without a current right, keeps the rights out of M03 and grants them all-or-nothing only through the sample seed and the bootstrap, and shows the caller's rights and the reason for each unavailable action in EN, KO and AR. Revised after security design review round 1.
status: stable
tags: [HYB-310, P02, HB-31, HB-33, HB-35, HB-42, permissions, security, separation-of-duties]
sources:
  - resource: ../../reviews/HYB-301-trace-classification/report.md
  - resource: ../HB-284/plan.md
  - resource: ../HB-285/plan.md
  - resource: ../HYB-297/plan.md
  - resource: ../HYB-298/plan.md
  - resource: ../HYB-299/plan.md
  - resource: ../HYB-309/plan.md
  - resource: ../../contracts/commands/planning-actions.md
  - resource: ../../contracts/security/access-and-trust.md
  - resource: ../../ui/terminology/product-glossary.md
---

# HYB-310 P02 separate API, module and channel rights

## Ticket, base and ownership

- **Ticket.** HYB-310: "HB-31 / P02 spec:33 and HB-284/plan.md:308-310,376: every integration command uses AdministerInstallation; split the API, module and channel permissions as specified; security-focused review." Source: item 14 of the "Required and missing" table in [the HYB-301 report](../../reviews/HYB-301-trace-classification/report.md) (`report.md:329`). Owner direction of 2026-11-17: faithful to planning.
- **Base.** main `9f66b78` (HYB-309 and HYB-323 merged). Branch `feat/HYB-310-p02-permission-split`. Worktree `/private/tmp/hb310`. Machine `mac_mini`.
- **Ownership.**
  - Root (Claude Code) owns this plan, the integration and the PR.
  - One implementer writes the code in this worktree.
  - **An independent read-only security reviewer reviews this design before any code is written** (`design-first.md`; AGENTS.md: permissions and public contracts). It changes who may run 15 commands and one read, adds five commands and an executor rule, adds rights to a closed catalogue, and changes the public command and read contracts. Rounds 1 and 2 returned CHANGES REQUIRED (see "Review round 1" and "Review round 2"); round 3 reviews this revision. A second security review covers the actual diff.
  - A separate read-only technical-writing review checks the right names and the EN, KO and AR copy (`product-terminology.md`: a permission meaning changes).
- **Allowed files.**
  - `modules/auth/`: `types.go`; tests `explain_test.go`, `installation_test.go`, plus a new `integration_rights_test.go`.
  - `modules/gate/`: `types.go`, `gate.go` (`CheckAuthor`, `RecheckAuthor`, F2/N2); tests `installation_test.go`, `gate_test.go`.
  - `modules/commands/`: `types.go`, `registry.go`, `execute.go`, `openapi.go` (the `Author` definition flag, F2); tests `commands_test.go`, `property_test.go`, `document_test.go`, `testdata/openapi.golden.json`.
  - `modules/plugins/`: `registry.go` (`Close` of a pending proposal with a system reason, F1/F2) and tests.
  - `modules/webhooks/`: `subscriptions.go`, `deliveries.go` (withdraw or close a proposed subscription, a resume request, a letter request; hold and re-approve a queued redelivery; `Dispatch` skips held deliveries, F1/F2/N5) and tests.
  - `apps/api/server/`:
    - code: `integrationcmds.go`, `integrations.go`, `trusthold.go` (four new withdrawal commands and the re-approval), `integrationseed.go` (dispatch through the wrapper), `bulkimpact.go`, `seed.go`, `signin.go`, `openapi.go`, `trustevidence.go` (only if `holdsRight` needs the role check, F1/F4), and a new `integrationrights.go` (right table, grants, read admission, starter recheck, dispatch wrapper);
    - tests: `integration_commands_test.go`, `integration_edges_test.go`, `integrations_api_test.go`, `access_test.go`, `accessrequests_test.go`, `bulk_test.go`, `trust_source_test.go` (only if a caller list changes), and a new `integration_rights_test.go` (split by responsibility if it nears 1,500 lines).
  - `apps/console/src/`: `service/integrations.ts`; `modules/administration/pages/integrations/` `live-model.ts`, `live-model.test.ts`, `live-codes.ts`, `live-copy.ts`, `Live.tsx`, `LiveClients.tsx`, `LiveModules.tsx`, `LiveProposals.tsx`, `LiveChannels.tsx`, `LiveEvents.tsx`, `LiveCommand.tsx`.
  - Docs: `docs/ui/terminology/product-glossary.md`; `docs/features/HYB-310/plan.md` and `verification.md`; the permission residual line in `docs/features/HB-284/plan.md:376` (pointer to this ticket only).
  - Not edited: the P02 screen spec and the planning documents (authorized local workspace, untracked); M03 console files (no M03 change is planned; see "M03").

## Planning text (verbatim) and cross-check

Planning documents under `docs/planning/v1.16/` and `docs/ui/screens/P02/spec.md` are read from the authorized local workspace (not in Git).

| # | Source | Verbatim text | This plan | Status |
| --- | --- | --- | --- | --- |
| P1 | `P02/spec.md:33` (통합별 탭) | "API/모듈/알림채널을 한 권한으로 자동 합치지 않음" | API, module and channel areas get separate rights; no right covers two areas; `administer_installation` no longer gates any P02 command or area | covered |
| P2 | `P02/spec.md:72` | "\| 등록 초안 \| 확장 관리 권한 \| 권한·manifest·목적 정의 \| 검증 대기 등록 \| 허용 기능/송신 범위 확인 \|" | `administer_modules` (plugin proposals) and `administer_channels` (subscription proposals). Registration of publishers, keys and versions has no command yet (HB-284 residual) | covered for existing commands |
| P3 | `P02/spec.md:73` | "\| 호환·송신 시험 \| 검증 권한 \| 승인 테스트 목적지 \| 호환/서명/권한 증거 \| 실데이터 송신 여부와 대상 확인 \|" | No compatibility or test-send command exists, so no right is created (`design-first.md`: no abstraction for a future consumer only) | **not covered**; R1, Q3 |
| P4 | `P02/spec.md:74` | "\| 활성화 요청 \| 적격 승인 권한 \| 실증·라이선스·범위 검증 \| 고정 버전 활성화 계획 \| 권한·외부 영향 확인 \|" | *Interpretation:* the result column "고정 버전 활성화 계획" (a fixed-version activation plan) is what an approved proposal is. So the step that makes an activation take effect (`plugin_approve`, `subscription_approve`, `subscription_approve_resume`) needs the area's approval right; filing the proposal needs the administration right (P5). The literal alternative is in Q2 | covered by interpretation; Q2 |
| P5 | `P02/spec.md:37` | "코어 DB 직접쓰기 금지·승인 요청≠사람승인·외부 앱 iframe 초기 제외" | Approval rights are separate actions from administration rights; neither implies the other; per-object proposer ≠ approver stays in the modules | covered |
| P6 | `P02/spec.md:75` | "\| 격리·중지 \| 운영 중지 권한 \| 대상/진행 작업 식별 \| 새 실행 차단·진행 결과 추적 \| 이미 나간 요청 취소 불가 안내 \|" | `stop_modules` (stop, disable) and `stop_channels` (pause, revoke), split by area because of P1 | covered; Q4 (revoke) |
| P7 | `P02/spec.md:76` | "\| 사건 재처리 \| 재처리 권한 \| 원사건/현재 scope·멱등검사 \| 중복 방지 재처리 \| 외부 효과/대상 확인 \|" | `reprocess_events` for all four dead-letter commands; request ≠ approval stays in the module | covered |
| P8 | `P02/spec.md:23` | "역할 이름만으로 전역 권한을 부여하지 않으며 실제 행위·법인·고객·기간의 권한을 검사한다." | Every command and every area is checked against the caller's own grant at that moment; the administrator role alone grants nothing in P02 | covered (scope dimensions: installation-wide, see "Scope") |
| P9 | `P02/spec.md:100` (revoked) | "접근 권한이 회수되었습니다. 상세·파일과 후속 명령을 표시하지 않습니다. 권한 있는 담당자에게 요청하세요." | A withheld area returns only `state: not_permitted`; no rows, counts or actions | covered |
| P10 | `ui-design.md:156` | "\| P02 API·플러그인·채널 \| OP·IA·범위 내 BL \| … 등록/시험/승인/격리/중지/재처리 \| …" | This plan chooses installation-kind rights, which need the administrator role (`gate.go:109`); that choice, not the planning text, excludes a billing operator (BL) | **partly**; R6, Q7 |
| P11 | `requirements.md:593` (HB-33) | "UI·CLI·AI·외부 소비자·플러그인은 하나의 업무 명령/권한 레지스트리를 따른다." | Rights are declared once in the command definitions and read from the same table for the read and OpenAPI | covered |
| P12 | `detailed-requirements.md:403` (QA-04) | "위임 유효 권한은 현재 사용자 권한과 client 권한의 교집합이다. … 승인요청과 사람승인권한은 별개다." | Each right needs both the person's grant and the console client grant; approval rights are separate | covered for the console client; OAuth clients not connected (R3) |
| P13 | `requirements.md:557, 563, 565` (HB-31) | "알림 읽음·수락·해결을 분리한다." / "…채널 실패·권한 철회를 시험하고…" / "**결정 필요:** 마감 달력·업무시간·알림 채널·에스컬레이션." | Channel rights are rechecked on every command and read, so a revoked right takes effect on the next request (tests T6). Notification channels for deadlines are not built; the P02 channel area is webhook subscriptions | covered for webhook channels; notification channels out of scope |
| P14 | `planning-actions.md:121` | "Channel connection or notification receipt grants no billing permission (B1387). … P02 supplies channel bindings; M05 and U03 remain personal." | Integration rights gate only `integrations.*` and the P02 read; a definition test proves no other command accepts them | covered |
| P15 | `requirements.md:753` (HB-42) | "사용자 계정 접근 ID 생성부터 초대·활성화·권한 부여·변경·정지·탈퇴·회수까지 관리한다." | Granting and revoking these rights in a running installation is not possible (same enrolment block as HYB-297 R4 and HYB-299 R6) | **not covered**; R2, Q5 |
| P16 | `retention-access.md:59` | "정지/권한 회수 시 authoritative deny/version을 먼저 갱신하고 … 민감 명령과 다운로드는 현재 상태를 확인한다." | Gate decision on every command, idempotent replay and read; a suspended person or closed grant is refused at once | covered |
| P17 | `retention-access.md:63` | "사람이 시작한 대기 작업은 현재 권한과 업무 소유권을 재검사한다. … 자동으로 더 강한 서비스 계정에 승격하지 않는다." | Every approval rechecks under the write lock that the proposer, resumer or requester still qualifies for the starting right; if not, the approval is refused (`requester_lost_right`) and the entry is closed. Already-approved processing is decided per command (see "Starter recheck"). Nothing runs as a service account | covered (F1) |
| P18 | `access-and-trust.md:24` | "Ordinary bounded membership administration authorizes none of those changes. … self-approval is prohibited." | Integration rights are not grantable through M03; self-approval refusals stay | covered |
| P19 | `requirements.md:629, 635` (HB-35) | "외부 코드의 DB 직접 쓰기와 관리자 DOM 실행은 금지한다." / "모듈 timeout/격리/중지/롤백이 다른 고객을 막지 않으며…" | Unchanged; stop rights keep stop and disable available under a trust hold (`trusthold.go:76-79`) | regression only |
| P20 | `HB-284/plan.md:308-310` | "**Permission (a deviation from P02).** Every command requires `AdministerInstallation` on the installation kind, the same permission as the other admin commands. - This deviates from P02, which keeps API, module and channel authority separate (`docs/ui/screens/P02/spec.md:33`) and names separate extension, verification, approval, stop and reprocess permissions (`spec.md:70-76`). The gate has no such actions yet." | The deviation is removed for extension (administration), approval, stop and reprocess; verification has no command (R1) | covered except verification |
| P21 | `HB-284/plan.md:376` | "**Permissions.** Separate gate actions for API, module and channel administration, stop and reprocess, as P02 requires. Until then every command uses `AdministerInstallation`." | Resolved; the line is updated to point to HYB-310 | covered |
| P22 | `HYB-301 report.md:329` | "\| 14 \| Separate API, module and channel permissions, plus stop and reprocess permissions, on P02 commands \| HB-31, HB-42; `P02/spec.md:33, 70-76` \| …" | This ticket | covered except R1, R2 |

## Current behaviour (base `9f66b78`)

### Commands

All 15 are defined in `apps/api/server/integrationcmds.go:389-417`. Three variables at `:394-396` each hold `{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}`.

| # | Command | Line | Target | Right today |
| --- | --- | --- | --- | --- |
| 1 | `integrations.plugin_approve` | `:399` | `plugin-proposal-<n>` | `administer_installation` |
| 2 | `integrations.plugin_withdraw` | `:400` | `plugin-proposal-<n>` | `administer_installation` |
| 3 | `integrations.plugin_propose` (kinds install, upgrade, rollback, enable, remove) | `:401-402` | `plugin:<id>` | `administer_installation` |
| 4 | `integrations.plugin_stop` | `:403` | `plugin:<id>` | `administer_installation` |
| 5 | `integrations.plugin_disable` | `:404` | `plugin:<id>` | `administer_installation` |
| 6 | `integrations.subscription_propose` | `:405-406` | `webhooks` | `administer_installation` |
| 7 | `integrations.subscription_approve` | `:407` | `webhook-subscription:<id>` | `administer_installation` |
| 8 | `integrations.subscription_pause` | `:408` | `webhook-subscription:<id>` | `administer_installation` |
| 9 | `integrations.subscription_request_resume` | `:409` | `webhook-subscription:<id>` | `administer_installation` |
| 10 | `integrations.subscription_approve_resume` | `:410` | `webhook-subscription:<id>` | `administer_installation` |
| 11 | `integrations.subscription_revoke` | `:411` | `webhook-subscription:<id>` | `administer_installation` |
| 12 | `integrations.letter_request_redeliver` | `:412` | `webhook-letter:<id>` | `administer_installation` |
| 13 | `integrations.letter_approve_redeliver` | `:413` | `webhook-letter:<id>` | `administer_installation` |
| 14 | `integrations.letter_request_dismiss` | `:414` | `webhook-letter:<id>` | `administer_installation` |
| 15 | `integrations.letter_approve_dismiss` | `:415` | `webhook-letter:<id>` | `administer_installation` |

### Reads

| Read | Line | Right today |
| --- | --- | --- |
| `GET /api/v1/integrations` (route `boundary.go:45`) | `integrations.go:716-735`, admission through `s.admitted` (`users.go:96-123`, gate check `:112`) | `administer_installation` for the whole page |
| API clients section | `clientsOf`, `integrations.go:339-366` (console client grants without installation-kind ones, `:342`; service users) | same |
| Modules section (publishers, versions, installations, proposals, permitted licences) | `modulesOf`, `integrations.go:493` | same |
| Channels and events sections | `channelsAndEvents`, `integrations.go:675-702` | same |
| Per-row action availability | `installRowOf` `:420-438`, `pluginBlockers` `:457-470`, `subscriptionActions` `:551-583` (calls `s.admits` at `:559` and `:576`, which reads destination state and subscriber permits), `letterActions` `:625-651` | state only; no right check |

### Facts the design rests on

- **Catalogue.** The action set is closed (`modules/auth/types.go:20-45`). `InstallationAction` (`types.go:53-59`) is true for six actions. For those, the gate requires a human administrator (`modules/gate/gate.go:109`), `AddGrant` requires a human subject (`modules/auth/registry.go:264`) and delegation is refused (`registry.go:375`).
- **M03 guard (HYB-297 B1).** M03 excludes every installation-kind action from the ceiling (`access.go:148`), the listing (`access.go:195`), revoke targets (`access.go:301`), request ceilings (`accessrequests.go:86`) and ledger replay (`ledger.go:87`). A new action added to `InstallationAction` is excluded everywhere with no M03 code change.
- **Executor.** The gate decision is taken on every call, including an idempotent replay. A missing right and an unknown target both give `permission_denied` (`modules/commands/execute.go:320-336`), so a refusal discloses nothing.
- **Grants live in memory.** The sample seed recreates them at every start (`seed.go:110-112`). The bootstrap grants `administer_installation`, the trust rights and the bulk right at activation (`signin.go:474-482`). The decision ledger never replays installation-kind grants (`ledger.go:87`). There is no stored grant to migrate.
- **Module-level separation already exists:**
  - plugin self-approval: `plugins/registry.go:234`;
  - subscription self-approval and self-resume: `webhooks/subscriptions.go:241`, `:306`;
  - letter request ≠ approval: `webhooks/deliveries.go:500`, `:537`.
- **Role and grants are separate.** The gate refuses an installation action while the person is not an administrator (`gate.go:109`), but it does not close the grant. Demotion to member suspends the use of every installation right; re-promotion within the grant's validity restores it with no new grant. Suspension of the user or membership, bulk revocation (HYB-299) and role-period expiry act the same way through identity facts.
- **Bulk impact.** `integration_proposals` is listed with `rechecked: false` (`bulkimpact.go:71`): pending plugin and subscription proposals of a departing person are counted but today stay approvable.
- **Dispatch.** No dispatcher is connected (`integrations.go:679`). The only `Dispatch` caller is the integration seed (`integrationseed.go:364`).
- **Trust hold.** It classifies stop, disable, pause and revoke as stops (allowed), and propose and approve as changes (refused) (`trusthold.go:76-85`). This is unchanged.
- **Console.** P02 states the deviation at `live-copy.ts:43-48`, shown at `Live.tsx:137`. A command refusal `permission_denied` is read as "gone" or "already applied" (`live-model.ts:605-610`).

## Design

### Rights

There are eight new installation-kind actions in `modules/auth/types.go`, re-exported in `modules/gate/types.go`. `InstallationAction` returns true for each, so there are 14 installation actions in total.

| Right (gate action) | Area | Commands | P02 area it opens |
| --- | --- | --- | --- |
| `administer_api_clients` | API | none today (no P02 API command exists) | API clients |
| `administer_modules` | module | 3 `plugin_propose` (2 `plugin_withdraw` is author-only, see "Author withdrawal") | Modules |
| `approve_modules` | module | 1 `plugin_approve` | Modules |
| `stop_modules` | module | 4 `plugin_stop`, 5 `plugin_disable` | Modules |
| `administer_channels` | channel | 6 `subscription_propose`, 9 `subscription_request_resume` (their withdrawals are author-only) | Subscriptions and channels; Events |
| `approve_channels` | channel | 7 `subscription_approve`, 10 `subscription_approve_resume` | Subscriptions and channels; Events |
| `stop_channels` | channel | 8 `subscription_pause`, 11 `subscription_revoke` | Subscriptions and channels; Events |
| `reprocess_events` | channel (events) | 12–15 (letter requests and approvals) and `redelivery_reapprove`; letter withdrawals are author-only | Subscriptions and channels; Events |

Rules:
- **No implication.** No right implies another. `administer_installation` implies none of them, and none of them implies `administer_installation`, a trust right or the bulk right. After this change no P02 command accepts `administer_installation`.
- **Withdrawal needs no current right (F2).** The author of pending work may always withdraw it, because withdrawal only reduces risk (the trust hold already allows it, `trusthold.go:80`). See "Author withdrawal". *Owner question Q6.*
- **Revocation of a subscription** sits with `stop_channels`. It blocks new deliveries (P6), and the trust hold already treats it as a stop (`trusthold.go:79`). Revocation is irreversible (`webhooks/subscriptions.go:334`; a revoked subscription is never resumed), so this default lets one stop-only holder remove a subscription permanently with no second person. *Owner question Q4: the alternative is `administer_channels`.*
- **Provisional IDs.** The eight action strings are provisional: HB-33 leaves "정식 명령/권한 ID" open (`requirements.md:601`). *Owner question Q10.*
- **Approval of a resume** is an approval (P4, P5). Requesting a resume is administration.
- **Both letter steps** (request and approve) need `reprocess_events`. The module keeps the requester ≠ approver rule.

**Scope.** Every right is installation-kind with the boot workspace and full customer, legal-entity, contract and period scope, as the existing installation grants are (`seed.go:278-279`).
- Integration objects carry no customer or contract today: plugin installations are per workspace, and subscriptions are per boot workspace.
- No workspace-kind right is introduced, because no P02 object has a narrower scope to check.

**Implementation shape.**
- A single table in `integrationrights.go` maps each `integrations.*` command ID to its right. `integrationDefinitions` reads the permission from that table and refuses to build (panics at start, covered by a test) if a command has no entry.
- The read admission, the availability reasons and the OpenAPI read section use the same table, so a command and its button can never disagree.

### Separation of duties

- **Required by planning (P5, P12, P18):**
  - approval is a separate right from administration (`approve_modules` and `approve_channels`, distinct actions);
  - no self-approval: the module refusals are unchanged, and `mapIntegration` keeps mapping them to `prerequisite_failed` (`integrationcmds.go:61-62`);
  - human only: the installation-action rule (`gate.go:109`), human subjects only (`registry.go:264`), not delegable (`registry.go:375`).
- **Not required by planning, so not enforced:** a ban on one person holding both the administration and the approval right of an area. Per-object proposer ≠ approver already prevents self-approval. *Owner question Q6.*
- **Stop and reprocess are separate** from approval: a stop-only operator can contain but cannot approve or re-enable. Re-enabling a stopped plugin goes through `plugin_propose` (`enable`) and `plugin_approve`; resuming a subscription goes through a request and an approval.

### Starter recheck at approval (F1, N4–N6; `retention-access.md:63`)

Root decision: implement. Each approval command, inside `Apply` under the domain write lock, runs in this order:

1. **State prechecks first (N6, R3-1).** Each gives `invalid_stage` and closes nothing:
   - `plugin_approve`: the proposal is pending (`integrationcmds.go:147`). The plugins module also returns an applied proposal to its own approver as a replay (`plugins/registry.go:230`); this precheck stops a second command before that, and an idempotent retry with the same key is served from the journal after the gate recheck, not re-applied.
   - `subscription_approve`: the subscription is proposed (the `silent` step at `:302`).
   - `subscription_approve_resume`: **new explicit check**: `current.Resumer != ""` and the subscription is paused or suspended (today `:305` has no silent check).
   - `letter_approve_redeliver` and `letter_approve_dismiss`: **new explicit check**: a pending request **of that kind** exists in `s.integ.redeliver` or `s.integ.dismiss` for the letter.
   - `redelivery_reapprove`: the redelivery is still held.
   - **Empty starter.** If the pending record has an empty proposer, resumer or requester, the command returns `internal_error` and closes nothing; it is never treated as a lost right.
2. **Starter recheck.** The person who started the work must still **qualify** for the starting right:

| Approval command | Starter | Starting right |
| --- | --- | --- |
| `plugin_approve` | proposal proposer | `administer_modules` |
| `subscription_approve` | subscription proposer | `administer_channels` |
| `subscription_approve_resume` | resume requester (`Resumer`) | `administer_channels` |
| `letter_approve_redeliver` | redelivery requester | `reprocess_events` |
| `letter_approve_dismiss` | dismissal requester | `reprocess_events` |
| `redelivery_reapprove` (N5) | redelivery requester | `reprocess_events` |

3. The module call, only if the starter qualifies.

- **Full explanation, fail closed (R3-2).** `starterQualifies` returns one of `qualifies`, `lost` or `unknown`. It uses the full explanation, not a boolean: `ExplainInstallation` through `holdsRight` and the identity facts for role, membership and user state.
  - If the explanation is authority unavailable, the command is refused `authority_unavailable` (503).
  - If the identity facts cannot be read, the command is refused `dependency_unavailable` (503).
  - In both cases nothing is closed and nothing is held. Only a definite `lost` closes or holds.
- **Qualifies** means all of the following:
  - `s.holdsRight(starter, right, now)` (`trustevidence.go:229`): installation explain with the console client, covering grant validity, closed grants and the client intersection;
  - the starter's membership in the boot workspace is effective with role administrator;
  - the user is active and human.
  The role part is needed because explain does not apply the gate's role rule (F4). It is taken from identity facts, as the trust evidence does, and `ExplainInstallation` keeps its two allowed callers (`trust_source_test.go:73`).
- **If the starter does not qualify (N4, root decision option a):** the approval **completes** as a closing.
  - Outcome: business state `closed`, fields `reason: requester_lost_right` and `target`. No refusal code, so no engine change; the result revision moves forward as for any applied command.
  - The entry is closed in the same lock with system reason `requester_lost_right`: `plugins.Close`, `webhooks.CloseSubscription` (proposed subscription), `CloseResume`, `CloseRedeliver` and `CloseDismiss`. The server's letter request maps are cleared, and the slot is freed (proposal counts and the duplicate check no longer see it).
  - The closure history records the triggering request ID and the approver who triggered it ("Closed because the person who started it no longer holds the right; found when {approver} tried to approve, request {id}"). The approver is recorded as the trigger, not as the one who closed or approved it.
  - The console shows the closure as what it is: "Not approved. It was closed because the person who started it no longer holds the right." It never says "Nothing was changed".
- **Read blocker.** The approve availability shows `requester_lost_right` (after `no_right`, before state reasons) when the starter does not qualify at read time. The read does not close anything.
  - **Disclosure (N8).** This blocker tells an approver that another named person currently lacks the right. Approvers already see the proposer's identity and are installation administrators, and M01 shows role and state of every person to them. This is judged acceptable and recorded; no owner question is raised.
- **Bulk impact.** `integration_proposals` becomes `rechecked: true` (`bulkimpact.go:71`). Its tally also counts pending resume requests and letter requests by their requester.
- **Already-approved processing** is decided per command. *Owner question Q11.*
  - **Active subscriptions continue.** Their approval was valid when given; a `stop_channels` holder can pause or revoke them.
  - **Enabled plugins continue**, for the same reason; `stop_modules` can stop them.
  - **Queued redeliveries (N5).** The server records the requester and approver of each redelivery.
    - **Requester lost the right:** the redelivery is closed (`webhooks.CloseRedeliver`, reason `requester_lost_right`); it is not re-approvable.
    - **Only the approver lost the right:** the redelivery is held for re-approval (`webhooks.HoldRedelivery`, reason `approver_lost_right`).
    - These checks run in a new wrapper `s.dispatchDeliveries`, under the same domain write lock as the `Dispatch` call it makes. If any starter or approver check is `unknown` (R3-2), the wrapper skips that dispatch round entirely: it does not call `Dispatch`, closes nothing and holds nothing, and reports the dependency failure. An AST test makes it the only caller of `webhooks.Dispatch`; the seed moves to it.
    - **The webhooks module itself refuses to lease a held delivery.** `Dispatch` skips held deliveries, shown by a module test, so a future caller that bypasses the wrapper still cannot send a held one.
    - **Re-approval** is a new command `integrations.redelivery_reapprove` on a new target prefix `webhook-redelivery:<redelivery-id>` with its own `Resolve` case (found only for a held redelivery in the boot workspace). Right: `reprocess_events`. It runs the starter recheck on the requester first (a lost requester closes it, as above). The re-approver must be neither the requester nor the lost approver (module refusal → `prerequisite_failed`). Success records the new approver and releases the hold (`webhooks.ReapproveRedelivery`). Trust hold: `holdRefuse, reasonChange`.
    - No dispatcher is connected today, so this is exercised only through the wrapper in tests and the seed. End-to-end delivery stays UNVERIFIED (R9).
  - **Dismissals** take effect at approval, so nothing remains queued.

### Author withdrawal (F2, N1–N3, N7)

Root decision: an author may withdraw their own pending work without the area's right. Root decision N3 (faithful to `P02/spec.md:100`, "접근 권한이 회수되었습니다. 상세·파일과 후속 명령을 표시하지 않습니다."): the author must still be an effective administrator; a person who lost administrator access gets no P02 command or read. A demoted author's pending items are closed by the starter recheck at the next approval attempt.

- **Engine change (`modules/commands`, N2).**
  - `Definition` gains `Author bool`. Registration accepts it only when `Mutating` is true and `Approval` is false; otherwise the definition is invalid.
  - The `Gate` interface used by the executor gains `CheckAuthor(answer, client, author string, request) Decision` and `RecheckAuthor(stamp, client, author string, request) Decision`. For an `Author` definition the executor calls `CheckAuthor` where it now calls `Decide`, and `RecheckAuthor` at commit time after the idempotency lookup, where it now calls `Recheck` (`execute.go:361-363`). A replay is protected by `CheckAuthor`, which runs before the journal lookup, so a stored result is returned only to a caller who passes the current author and identity rules; `RecheckAuthor` protects new calls between the decision and the apply (correction after the HYB-310 change review).
  - OpenAPI marks the operation `x-permission.author_only: true, right_not_required: true`; `Permission` stays the area's administration right so the definition validates and the catalogue shows the area.
- **`CheckAuthor` rules (`modules/gate`).** Allowed only when all hold:
  - `author` is not empty;
  - the identity answer confirms (active subject, current epoch, same workspace as the request);
  - the caller is human, with an **effective administrator role** (N3);
  - no support session is in use;
  - the client matches the immutable `Catalog.AuthorClient`, configured as `ConsoleClient` by the API server; an empty configuration disables author-only calls. The client must also be active and hold the matching direct grant. The constructor rejects a nonempty invalid client ID. No setter exists. This enforces the console client boundary (QA-04: delegated authority is the intersection with the client, `detailed-requirements.md:403`; no other client may act as the author);
  - the caller is `author`.
  The caller's grants are not consulted; the configured client's matching grant is required. Any failure gives `permission_denied`, the same as not found. `RecheckAuthor` applies the same rules to the stamp.
- **`Resolve` (N7, R3-3; root decision).** `AuthorID` comes from the server's own store, with no module call. Authorship is kept **per request**, not per object:
  - Every resume, redelivery and dismissal request gets its own request ID (`resume-request-<n>`, `letter-request-<n>`) when it is made. The live maps (`s.integ.redeliver`, `s.integ.dismiss`, the resumer record) point to the current request ID. A separate tombstone map, request ID → {author, kind, object, closed state}, is never overwritten or removed while the server runs.
  - The withdrawal commands for requests target the request, not the object: `webhook-resume-request:<request-id>` and `webhook-letter-request:<request-id>`, each with its own `Resolve` case. The read exposes the request ID on each request row and in `own_pending`.
  - Plugin proposals (`plugin-proposal-<n>`) and subscription proposals (`subscription-<n>`) already have unique, never-reused IDs; their author record is kept after withdrawal.
  - **Fresh-key withdrawal of an older request (root note 1).** A withdrawal with a new idempotency key on a request ID succeeds only while the live map still points to that same request ID. Otherwise it gives `invalid_stage` and leaves the newer request untouched.
  - **Tombstone cap (root note 2).** At most 10,000 tombstones are kept per kind (resume, redeliver, dismiss). When the cap is reached, the oldest whose request is no longer live is evicted. A replay of an evicted request ID resolves as not found and answers `permission_denied` (fail closed). This is a known limit.
  - **Sequential IDs (root note 3).** Request IDs are sequential, as `plugin-proposal-<n>` and `subscription-<n>` are; they reveal an approximate request count to administrators. Accepted.
  - So a replay always resolves to the original author, even after the object has a newer request by someone else. It is answered from the journal after `CheckAuthor` passes, not refused as not found and not attributed to the newer requester.
- **Withdrawal commands (N1, N7).** The handler kind fixes which record is the author's:

| Command | Target | Author record | Module call |
| --- | --- | --- | --- |
| `integrations.plugin_withdraw` (existing, now author-only) | `plugin-proposal-<n>` | proposal proposer | `plugins.Withdraw` |
| `integrations.subscription_withdraw` | `webhook-subscription:<id>` (proposed) | subscription proposer | `webhooks.WithdrawSubscription` |
| `integrations.subscription_withdraw_resume` | `webhook-resume-request:<request-id>` | resume requester (per request) | `webhooks.WithdrawResume` |
| `integrations.letter_withdraw_redeliver` | `webhook-letter-request:<request-id>` (kind redeliver) | redelivery requester (per request) | `webhooks.WithdrawRedeliver` |
| `integrations.letter_withdraw_dismiss` | `webhook-letter-request:<request-id>` (kind dismiss) | dismissal requester (per request) | `webhooks.WithdrawDismiss` |

  - `found()` (`integrationcmds.go:113-126`) gets explicit cases for the new kinds, so the `subscription_`/`letter_` prefix rules cannot misroute them.
  - `Apply` re-verifies under the lock that `ctx.User` is the requester **of that kind** (N1): the author of a dismissal request cannot withdraw a redelivery request on the same letter, and the reverse.
  - Each is classified `holdAllow, reasonPending` in `trusthold.go`, as `plugin_withdraw` is (`:80`).
- **Console.** "Withdraw" on the caller's own pending rows. An administrator who lacks the area's rights reaches them through `own_pending` (see "Read").

### Grants: seed, bootstrap and migration

- **Client grants.** Each right gets one console client grant, `integration-<right>-console`, with the validity of `installation-console` (`installationGrants`, 30 days, `directory.go:12`). Without it the gate refuses (client intersection, P12).
- **Sample seed.** It gets a new step `integration-grants` after `bulk-grants` (`seed.go:112`), with fault point `integration-grant`.
  - `admin` and `admin2` hold all eight rights.
  - `admin3` holds `stop_modules`, `stop_channels` and `reprocess_events` only: an operator profile, so the split can be seen in the browser.
  - `admin3` keeps `administer_installation` and the trust rights.
  - *Owner question Q8.*
- **All-or-nothing (F5; root decision).** `grantIntegration(user, rights, now)` adds the client grants (tolerating `ErrConflict`, as today), then the person grants one by one. If any person grant fails after k succeeded, the k are revoked (`RevokeGrant` with their versions) before the error returns, so the person holds either the whole profile or none of it. Client grants are kept: alone they give a person nothing. If a rollback itself fails, the error names it, the seed reports partial and the bootstrap logs "integration grants rolled back incompletely". In that state the read reports what each right actually is; it never assumes the profile.
- **Bootstrap.** At activation the bootstrap administrator receives all eight rights. The call comes after `grantBulk` (`signin.go:480`) with the same log-and-continue handling: a failure leaves the person with no integration right (fail closed) and is logged as "integration grants failed at activation".
- **Grant IDs.** `integration-<right>-<user>`, validity `installationGrants`.
- **Migration (who keeps what).**
  - Nothing is stored, so there is no data migration.
  - Every person who holds `administer_installation` today (the seed's three administrators and the bootstrap administrator) keeps it for all non-P02 commands.
  - After restart they hold the integration rights listed above, and only those.
  - Anyone holding `administer_installation` without an integration right loses P02 commands and P02 areas. That is the intended fail-closed result.
  - The idempotency journal is in memory, so there are no stored replays to reinterpret. A replay after the upgrade is rechecked against the new rights (`execute.go:324-336`).
- **Demotion and re-promotion (F4).** A holder demoted to member, suspended (user or membership), bulk-revoked or past their role period cannot use any integration right from the next request: the gate refuses installation actions for a non-administrator (`gate.go:109`) and for a suspended subject. Their grants are not closed. Re-promotion or reinstatement within the grant's validity restores every right with no new grant and no approval. This is today's behaviour for `administer_installation`, the trust rights and the bulk right; it is stated, not changed. Their pending work is closed at the next approval attempt (starter recheck).
- **Enrolment block.** No command grants or revokes an integration right; M03 cannot (see "M03"). A non-sample installation has exactly one holder, the bootstrap administrator. It therefore cannot approve its own proposals (unchanged from today: invited administrators get no `administer_installation` either). This is shown truthfully on P02 (see "Console").
  - *Residual R2; owner question Q5.*

### M03

- **Grant and approval flow.** The new rights are installation-kind, so the HYB-297 guard applies with no M03 code change:
  - they are not offered in the ceiling (`access.go:148`) or as a request ceiling (`accessrequests.go:86`), so `access.request_grant` for one is refused like any pair outside the ceiling;
  - they are not listed (`access.go:195`);
  - `access.request_revoke` on one resolves as not found, giving the generic `permission_denied` (`access.go:301`);
  - ledger replay never closes one (`ledger.go:87`);
  - the M03 preview answers with its existing line: "This check can't be previewed here. It depends on approval or installation rules." (`live-copy.ts:188` in the access page).
- **Display.** M03 shows no new row or label. Tests prove the exclusion (T9).
- **Eligible approver.** M03 eligible-approver logic stays tied to `administer_installation` (`accessrequests.go:57`). An integration right does not make anyone an M03 approver.
- **M03 label table.** No entry is added to the M03 action labels (`live-copy.ts:221-230`, access page), because the actions never reach M03. If the owner later chooses M03 enrolment (Q5), labels, ceiling and approval rules are a separate ticket with its own security review.

### Read: `GET /api/v1/integrations`

- **Admission.** A new `admitIntegrations` in `integrationrights.go` repeats the session, CSRF and access steps of `admitted` (`users.go:96-111`), then takes nine gate decisions (eight integration rights plus `administer_installation`) with the console client.
  - **One decision set (F6).** If any of the nine decisions is `authority_unavailable`, the whole read answers 503 `authority_unavailable`; no partial page is built from a mixed set. Otherwise the page is admitted if any one is allowed, else 403 `permission_denied`. The `rights` field, the area gating and every `no_right` flag are computed from this same set; no later per-row gate call can disagree with it.
  - Admitting on `administer_installation` alone shows an installation administrator a page that says they hold no integration rights, with every area withheld. *Owner question Q9.*
- **New top-level field** `rights`: an object of the eight right names to booleans, from those decisions at read time.
- **Per area.**
  - `clients`: present if `administer_api_clients`; otherwise `{ "state": "not_permitted" }` and nothing else.
  - `modules`: present if any of `administer_modules`, `approve_modules`, `stop_modules`; otherwise `not_permitted`.
  - `channels` and `events`: present if any of `administer_channels`, `approve_channels`, `stop_channels`, `reprocess_events`; otherwise `not_permitted`.
  - A withheld area is not computed at all. No module call is made for it, so it cannot leak counts, timings or errors. A dependency failure in a withheld area does not turn the page into an error.
  - `own_pending`: the caller's own pending items (kind, target, state), present even when the area is withheld, so an administrator without the area's right can still withdraw (F2). It is built only from the server store (`pluginEntry`, `subscriptionEntry` and the letter request maps), with **no module call**, so it cannot leak a withheld area's module state or fail because of it (N8). It lists nothing of anyone else.
  - **No member admission (N3).** Admission is only the nine decisions above. A person who is no longer an effective administrator gets 403, as `P02/spec.md:100` requires; their pending items are closed by the starter recheck.
- **Withholding is a display boundary, not confidentiality (F8).** The API clients area shows console client grants and service users. The same facts reach other administrators elsewhere: M03 lists console client grants (`access.go:148`, the ceiling), M04 lists service users and credentials under `administer_installation`, and the bulk revocation preview counts service users owned by targets (`bulkimpact.go`). Likewise, module and channel facts appear in history and audit reads. The area rule stops P02 from showing them to a caller without the right; it does not keep them secret from installation administrators. The glossary rows say the same.
- **Availability.**
  - Every action availability (`can_*`/`*_blocked` pairs and `actions` maps) checks the right first. If the reader lacks it, the action is `{allowed: false, blocked: "no_right"}`, before any state reason.
  - For a pending proposal the reader cannot approve, `approve_blocked` is `no_right` and the module recheck (`pluginBlockers`, `:468`) is skipped, so a non-approver does not trigger rechecks.
  - **F7.** Likewise `subscriptionActions` skips `s.admits` (`integrations.go:559` for approve, `:576` for approve resume) when the reader lacks `approve_channels`: no destination-state or subscriber-permit read is made on behalf of a caller who cannot approve.
  - Reason order per action: `no_right`, then `requester_lost_right` (approvals only), then the existing state reasons.
- **Unchanged.** The read stays under the domain read lock; nothing new is cached between requests.

### Refusal codes

No new code.

| Case | Code | Status |
| --- | --- | --- |
| Command without its right (including `administer_installation` alone, a trust right, the bulk right or another integration right) | `permission_denied` | 403 |
| Member role, service subject or AI principal holding a grant | `permission_denied` (gate `installation_scope` / `principal`) | 403 |
| Unknown target or another workspace's target, with or without the right | `permission_denied` | 403 |
| Right lost between first call and idempotent replay | `permission_denied` | 403 |
| Gate unavailable | `authority_unavailable` | 503 |
| Self-approval, own request | `prerequisite_failed` (unchanged mapping) | as today |
| Approval whose starter no longer qualifies | not a refusal: applied, business state `closed`, field `reason: requester_lost_right` (N4) | 200 |
| Withdrawal by a non-author, by the author of the other request kind, by a non-administrator, through a support session or another client, or of an unknown target | `permission_denied` | 403 |
| Re-approval by the requester or the lost approver | `prerequisite_failed` | 422 |
| Read with no integration right and no `administer_installation` | `permission_denied` | 403 |
| Read where any of the nine decisions is unavailable | `authority_unavailable` | 503 |

### OpenAPI

- `x-permission.action` of the 15 `integrations.*` operations changes from `administer_installation` to the right in the table above. It is generated from the definitions (`modules/commands/openapi.go:97`); no hand edit.
- `x-reads` (`apps/api/server/openapi.go:39`) gains `/integrations`:
  - `operationId: integrations.read`;
  - `x-admission` lists the nine rights, any one of which admits;
  - `x-area-rights` maps `clients`, `modules`, `channels` and `events` to their rights;
  - `x-withheld: "state not_permitted, no other field"`;
  - `x-blocked-first: "no_right"`;
  - 401 (`unauthenticated`), 403 (`CodeCSRF` and `permission_denied`) and 503 (`authority_unavailable` and `dependency_unavailable`) in the existing `readCodes` form, as for `/identity/users` (`openapi.go:75-79`).
- Author-only operations carry `x-permission.author_only: true` and `right_not_required: true`; the four new withdrawal operations and `integrations.redelivery_reapprove` appear in the generated paths.
- `x-changed` on `/integrations`: "before HYB-310 every integrations command and the whole read required administer_installation".

### Console (P02)

- **Parsing.**
  - `service/integrations.ts` parses `rights`; an unknown or missing key is read as `false`.
  - It accepts `not_permitted` as an area state.
  - A page without `rights` (older server) treats every right as false and every action as unavailable. No action is offered on a guess.
- **Rights line.** It replaces the deviation line (`live-copy.ts:43-48`):
  - "Your integration rights: {list}." / "내 연동 권한: {list}." / "صلاحيات التكامل لديك: {list}."
  - With none: "You hold no integration rights. Integration rights are given only when the installation is set up; they can't be granted here yet." / "연동 권한이 없습니다. 연동 권한은 설치를 설정할 때만 부여되며 아직 여기에서 부여할 수 없습니다." / "لا تملك أي صلاحيات تكامل. تُمنح صلاحيات التكامل عند إعداد التثبيت فقط، ولا يمكن منحها هنا بعد."
- **Withheld area.** The tab stays in place, so the layout is stable and the area is not hidden. Its panel shows:
  - "You don't hold a right for {area}. Nothing from this area is shown." / "{area}에 대한 권한이 없습니다. 이 영역의 내용은 표시하지 않습니다." / "لا تملك صلاحية لـ{area}. لا يُعرض شيء من هذه المنطقة."
  - This is not shown as empty, not as 0 and not as an error (`usability.md`).
- **Unavailable action (`no_right`).** The button is not offered. The row shows the reason:
  - "Needs the {right} right. You don't hold it." / "{right} 권한이 필요합니다. 이 권한이 없습니다." / "تتطلب صلاحية {right}، وأنت لا تملكها."
- **Refusal after reload.**
  - `deniedOutcome` (`live-model.ts:605`) gains a first branch: if the reloaded page shows the caller no longer holds the command's right, the outcome is `no_right`.
  - The message is "You no longer hold the {right} right. Nothing was changed." / "더 이상 {right} 권한이 없습니다. 아무것도 변경되지 않았습니다." / "لم تعد تملك صلاحية {right}. لم يتغيّر شيء."
  - If the reload itself returns 403 (the caller lost every right and `administer_installation`, or is no longer an administrator), the outcome is `no_access`: the page switches to the existing denied block ("Integrations are not available to you") and the command message says nothing was changed.
  - Otherwise the existing branches apply.
- **Starter lost the right.** An approve button with `requester_lost_right` is not offered; the row reads "The person who started this no longer holds the right. It can't be approved and will be closed." / "시작한 사람에게 더 이상 권한이 없습니다. 승인할 수 없으며 종료됩니다." / "لم يعد من بدأ هذا يملك الصلاحية. لا يمكن اعتماده وسيُغلق."
- **Closed by an approval attempt (N4).** When an approval returns business state `closed` with `reason: requester_lost_right`, the message is "Not approved. It was closed because the person who started it no longer holds the right." / "승인되지 않았습니다. 시작한 사람에게 더 이상 권한이 없어 종료되었습니다." / "لم يُعتمد. أُغلق لأن من بدأه لم يعد يملك الصلاحية." The row's history shows the closure with the request ID and the approver who triggered it. This is a change, so the "Nothing was changed" wording is never used for it.
- **Held redelivery** (re-approval button "Approve again" / "다시 승인" / "اعتماد مجددًا", command `integrations.redelivery_reapprove`). "Held: the approver no longer holds the Event reprocessing right. Another holder must approve it again." / "보류됨: 승인자에게 더 이상 이벤트 재처리 권한이 없습니다. 다른 권한 보유자가 다시 승인해야 합니다." / "معلّق: لم يعد المعتمِد يملك صلاحية إعادة معالجة الأحداث. يجب أن يعتمده صاحب صلاحية آخر مجددًا."
- **Withdraw** buttons on own pending rows, and a "Your pending items" list for areas the caller cannot see (see "Read" and "Author withdrawal"). Withdrawing a request of one kind never offers the other kind.
- **Right labels (EN / KO / AR)**, AI drafts for TW and native review:

| Right | EN | KO | AR |
| --- | --- | --- | --- |
| `administer_api_clients` | API client administration | API client 관리 | إدارة عملاء API |
| `administer_modules` | Module administration | 모듈 관리 | إدارة الوحدات |
| `approve_modules` | Module approval | 모듈 승인 | اعتماد الوحدات |
| `stop_modules` | Module stop | 모듈 중지 | إيقاف الوحدات |
| `administer_channels` | Subscription and channel administration | 구독·채널 관리 | إدارة الاشتراكات والقنوات |
| `approve_channels` | Subscription and channel approval | 구독·채널 승인 | اعتماد الاشتراكات والقنوات |
| `stop_channels` | Subscription and channel stop | 구독·채널 중지 | إيقاف الاشتراكات والقنوات |
| `reprocess_events` | Event reprocessing | 이벤트 재처리 | إعادة معالجة الأحداث |

  The area names reuse the existing tab titles (`live-copy.ts:36-40`): "API clients / API client / عملاء API", "Modules", "Subscriptions and channels", "Events and redelivery".
- **Glossary.** New rows in `product-glossary.md`:
  - `integration-rights`: the group term 연동 권한 / Integration rights / صلاحيات التكامل. Definition: eight installation rights separate from installation administration, the trust rights and the bulk revocation right. None implies another. They are granted only by the sample seed and the bootstrap today, and enrolment is not decided.
  - One row per right, with "Forbidden implications": not a billing permission (P14), not M03-grantable, not a role, and not confidentiality: a withheld area may still be visible to installation administrators on M03 or M04 (F8). The IDs are marked provisional (Q10).
  - The `bulk-revocation-right` and `trust-rights` rows are checked for consistency.
- **Unchanged.** Layout, widths and RTL handling are unchanged. The `service-screen-style.md` checks are re-run on the changed panels only.

## Implementation order

1. `modules/auth` and `modules/gate`: the eight actions, `InstallationAction`, `CheckAuthor`, tests (T1).
2. `modules/commands` and `modules/gate`: the `Author` flag, `CheckAuthor`/`RecheckAuthor` in the executor's gate interface, registration rule and OpenAPI, with tests (T15).
3. `modules/plugins` and `modules/webhooks`: close with a system reason, the withdrawal operations, redelivery hold and re-approval, with tests.
4. `integrationrights.go`: the command → right table, all-or-nothing `grantIntegration(user, rights, now)`, `admitIntegrations`, `readerRights`, `starterQualifies`, `dispatchDeliveries`.
5. `integrationcmds.go`: definitions read the table; delete the three `AdministerInstallation` variables; state prechecks then starter recheck and closing in approvals; the five withdrawal commands, `redelivery_reapprove` and the `webhook-redelivery:` target; `trusthold.go` entries.
6. `integrations.go`: one decision set, area gating, `rights` and `own_pending` fields, reason order, skipped recheck and `admits`.
7. `bulkimpact.go`, `integrationseed.go`, `seed.go`, `signin.go`.
8. `openapi.go`: the `x-reads` entry.
9. Server tests T2–T16, then a mutation run.
10. Console, console tests, glossary, browser QA.
11. `verification.md`; security review of the diff; HB-284 residual pointer.

## Tests

### Server and modules

- **T1 Catalogue (`modules/auth`, `modules/gate`).**
  - `InstallationAction` is true for exactly the 14 named actions; `explain_test.go` "exactly the six" becomes "exactly the fourteen", listed by name.
  - Each new right: grant to a service subject refused (`registry.go:264`); delegation refused (`:375`); member-role request refused at the gate (`installation_scope`).
  - No two of the 14 imply each other: a grant of one never allows another.
- **T2 Definition table.**
  - The 18 `integrations.*` definitions (15 existing, four new withdrawals and `redelivery_reapprove`) — 20 in all — have exactly the rights in "Rights" and the `Author` flag on exactly the five withdrawals, compared against a literal table written in the test, not against the production table.
  - No `integrations.*` definition uses `administer_installation`.
  - No non-`integrations.*` definition uses any of the eight rights (P14).
  - Building definitions with a command missing from the table fails.
- **T3 Command × right matrix (`integration_rights_test.go`).**
  - For each of the 15 existing commands (withdrawals: see T12) and each caller profile:
    - each of the eight rights alone;
    - for each command, **all rights except the command's own** (all other integration rights, `administer_installation`, the trust rights and the bulk right);
    - `administer_installation` alone;
    - all four trust rights plus the bulk right;
    - no installation right;
    - the command's right as a person grant **without** its client grant, and as a client grant **without** a person grant.
  - A fresh world gives a test administrator exactly that profile, with a target where the command can succeed.
  - Expected: `permission_denied` unless the profile holds both the person and the client grant of the command's right. With the right, the command is not refused for permission, and for the success fixture the business state matches the existing integration tests.
- **T4 Read matrix.**
  - For each single-right profile, for multi-right profiles (each pair within an area, each cross-area pair, the admin3 operator profile, all eight) and for `administer_installation` alone: `rights` is exactly that profile; areas are present or `not_permitted` by the union rule; a withheld area has no key other than `state`; every action in a present area outside the profile is `no_right`.
  - No right at all: 403.
  - **F6:** the gate is made unavailable for one of the nine decisions (each in turn): the whole read is 503 `authority_unavailable`, with no body fields beyond the error.
  - A fault injected in a withheld area's dependency does not fail the read.
  - **F7:** a reader without `approve_channels` causes no destination-state or permit read (counting stand-ins); with it, both are read.
- **T5 Negative and boundary.**
  - Expired person grant; expired or removed client grant; closed grant; other-workspace target; unknown target, with and without the right. All give `permission_denied` and an identical body apart from the request ID.
- **T6 Real revocation paths, per area (F4; P13, P16).** For one right in each area (`administer_api_clients` by read, `administer_modules`, `approve_channels`, `reprocess_events` by command):
  - role changed to member (`identity.change_role`), then back to administrator within validity: refused, then allowed again with no new grant;
  - membership suspended and reinstated; user suspended and reinstated: refused, then allowed;
  - bulk revocation (HYB-299 execute) including the holder: refused afterwards;
  - role period expiry by the forward-only server clock: refused at the instant;
  - registry `RevokeGrant` between two commands, and between an accepted command and its idempotent replay with the same key: the replay is refused, not served from the journal;
  - between two reads: the area becomes `not_permitted`.
- **T7 Separation.**
  - A holder of `administer_modules` and `approve_modules` cannot approve their own proposal (`prerequisite_failed`, unchanged).
  - A holder of `approve_modules` only cannot propose, and cannot withdraw someone else's proposal.
  - A `stop_modules`-only holder can stop and disable, and cannot re-enable through propose or approve.
  - The same holds for channels.
  - A `reprocess_events` holder cannot approve their own letter request.
- **T8 Seed and bootstrap (F5).**
  - Seed grants exactly the profiles in "Grants".
  - A failure injected after k person grants, for every k from 0 to 7: afterwards the person holds none of the eight (all k revoked), the seed reports partial, and the client grants remain.
  - A rollback failure injected after k: the error names it, and the read reports the rights actually held.
  - Bootstrap activation grants all eight; the same k-failure leaves the bootstrap administrator with `administer_installation`, no integration right and no P02 area, and logs the line.
- **T9 M03 exclusion.**
  - `access.request_grant` for each of the eight rights is refused as outside the ceiling, with the same body as any other pair outside the ceiling.
  - `access.request_revoke` on an `integration-*` grant gives the generic refusal.
  - The M03 listing and ceiling contain no integration right.
  - `protectedGrant` is true for each.
  - The M03 preview gives the existing "can't be previewed" answer.
- **T10 OpenAPI.** `x-permission` of each `integrations.*` operation equals T2's literal table, with `author_only` on the five withdrawals; `x-reads./integrations` exists with the specified keys; its 403 lists `CodeCSRF` and `permission_denied`, its 503 lists `authority_unavailable` and `dependency_unavailable`.
- **T11 Trust hold and step-up regression.** The existing `trusthold` and step-up tests pass with holders of the new rights; the four new withdrawals are allowed and `redelivery_reapprove` is refused under a trust hold.
- **T12 Starter recheck (F1, N4, N6).** For each of the six commands in the starter table and each loss path of T6 applied to the starter (demotion, membership or user suspension, bulk revocation, role-period expiry, `RevokeGrant` of the person grant, removal of the client grant):
  - the approval is applied with business state `closed` and `reason: requester_lost_right`, not refused;
  - the entry is closed with that system reason; its history records the request ID and the approver as trigger; the slot is freed (a new identical proposal or request is accepted); the revision moves forward;
  - the read showed `requester_lost_right` before the attempt, and the closed entry after it;
  - a starter who still qualifies: approval proceeds as today;
  - **no pending request (R3-1):** `subscription_approve_resume` with no resume request (and on an active subscription); each letter approval with no request of its kind (including when only the other kind is pending); `redelivery_reapprove` on a redelivery no longer held: each `invalid_stage`, nothing closed;
  - **empty starter (R3-1):** a pending record with an empty starter gives `internal_error` and closes nothing;
  - **injected faults (R3-2):** authority unavailable in the explanation → `authority_unavailable`; identity facts failing → `dependency_unavailable`; in both, the entry stays pending and the revision does not move;
  - **order (N6):** a target that is not pending gives `invalid_stage` even when the starter has lost the right, and nothing is closed; an idempotent retry of an applied approval returns the stored result after the gate recheck and never reaches `plugins.Approve`'s replay branch (`plugins/registry.go:230`) a second time;
  - the check runs under the write lock: a revocation racing the approval either precedes it (closed) or follows it (approved), never a mixed result (race test with `-race`).
- **T13 Already-approved processing (F1, N5).**
  - An active subscription whose approver or proposer loses the right stays active.
  - Queued redelivery, approver lost the right: `dispatchDeliveries` holds it; the read shows the hold; `redelivery_reapprove` by another current holder (not the requester, not the lost approver) releases it; by the requester or the lost approver → `prerequisite_failed`; by a holder when the requester has also lost the right → closed, not released.
  - Queued redelivery, requester lost the right: closed (`CloseRedeliver`), not held, and `redelivery_reapprove` finds no target (`permission_denied`).
  - Module test (`modules/webhooks`): `Dispatch` never leases a held delivery, with no server code involved.
  - `webhook-redelivery:` targets resolve only for held redeliveries of the boot workspace; a `webhook-letter:` target never resolves for `redelivery_reapprove`, and the reverse.
  - AST test: `webhooks.Dispatch` has exactly one caller, `dispatchDeliveries`, which holds the domain write lock.
  - **R3-2:** with an injected authority or identity failure, `dispatchDeliveries` skips the round: no `Dispatch` call, no close and no hold; the next healthy round proceeds.
- **T14 Author withdrawal (F2, N1–N3, N7).**
  - Each of the five withdrawals by its author succeeds while the author is an administrator holding no integration right.
  - **N1:** on a letter with both a redelivery request by A and a dismissal request by B, B's `letter_withdraw_redeliver` and A's `letter_withdraw_dismiss` give `permission_denied`; each author withdraws only their own kind.
  - **N3:** an author demoted to member, with a suspended membership or a suspended user: `permission_denied`; their item is closed at the next approval attempt.
  - **N2:** an author calling through a support session, or through a client other than the console, gets `permission_denied`; a definition with an empty `AuthorID` from `Resolve` refuses everyone.
  - By a non-author holding every right: `permission_denied`, same body as unknown target.
  - **Root note 1:** A withdraws req-1, files req-2, and a fresh-key withdraw on req-1 → `invalid_stage`, req-2 untouched.
  - **Root note 2:** with a lowered cap in the test, the oldest non-live tombstone is evicted and a replay of its ID answers `permission_denied`; a live request is never evicted.
  - **R3-3 sequence:** A requests a redelivery on letter L; A withdraws it (key K); B requests a redelivery on L; A retries with key K and the same body → A's stored result, with B's request untouched. The same holds for resume requests.
  - **N7 replay:** after a successful withdrawal, a retry with the same key and body returns the stored result (the author is still resolved from the server store); after the author is demoted, the same retry is refused by `CheckAuthor` before the journal lookup.
  - `own_pending` lists exactly the caller's pending items, including in withheld areas, and makes no module call (counting stand-ins).
  - A person who is no longer an administrator gets 403 on the read even with pending items.
- **T15 Engine (`modules/commands`, N2).** In `commands_test.go`:
  - registration refuses `Author` on a non-mutating definition and together with `Approval`;
  - the executor calls `CheckAuthor` (not `Decide`) for an `Author` definition and `RecheckAuthor` at commit after the idempotency lookup;
  - a replay whose `CheckAuthor` fails is refused before the journal lookup and not served from the journal;
  - OpenAPI marks `author_only`.
  In `property_test.go`: for random author/caller/client/session combinations, an `Author` command is allowed exactly when every `CheckAuthor` rule holds, at first call and at replay.
  In `modules/gate` tests: each `CheckAuthor` rule refused on its own.
- **T16 Admission does not change output (N9).** For each administrator profile of T4, the `rights`, area presence and `no_right` flags are identical whether or not the caller has own pending items, and whether admission came from `administer_installation` or from an integration right. `own_pending` is the only field that varies.

### Property tests

- **PT1 Matrix property.** For random subsets of the eight rights given to up to three administrators, random command sequences over the sample world, and random revocations, demotions and re-promotions in between: every right-checked command is refused with `permission_denied` exactly when the actor lacks its right at that moment; every approval closes the entry with `requester_lost_right` exactly when the starter does not qualify (and the target is pending); every withdrawal succeeds exactly when the actor is an effective administrator and the author of that kind. Otherwise the outcome equals the outcome for an all-rights actor in the same state. Uses the existing world harness (`world_test.go`) and a seeded generator; a failure prints the seed and the minimal sequence.
- **PT2 Read property.** For random subsets, the read's `rights` equals the subset, area presence is the union rule, and no `no_right` action is reported `allowed`.

### Mutation targets

- Each case of the `InstallationAction` switch.
- Each row of the command → right table.
- Each area predicate in `integrationsPage`.
- The `no_right`-first ordering in each availability function.
- The skipped recheck for non-approvers, and the skipped `s.admits` calls (`integrations.go:559`, `:576`).
- The any-unavailable → 503 rule of the read.
- Each part of `starterQualifies` (right, client grant, role, membership, user state), each starter mapping in the approval table, each R3-1 precheck, the empty-starter branch, the `unknown` → refuse branch and the dispatch-round skip (R3-2), the per-request tombstone lookup (R3-3), the state-precheck-before-recheck order, the close call, the `closed` outcome and the freed slot.
- `rechecked: true` for `integration_proposals` and the added tallies.
- The requester-versus-approver branch in `dispatchDeliveries`, the module's held-delivery skip in `Dispatch`, the re-approver exclusions and the `webhook-redelivery:` `Resolve` case.
- Each `CheckAuthor`/`RecheckAuthor` rule (empty author, identity, human, administrator role, support session, console client, author comparison); the executor's `Author` branch and its commit-time recheck; the registration rule for `Author`.
- The per-kind author check in `Apply` for `letter_withdraw_redeliver` and `letter_withdraw_dismiss` (N1), and the new `found()` cases.
- Each right in the seed profiles and in the bootstrap call; the rollback loop.
- The client grant creation.
- The console's `rights` parser (unknown key → false), the `no_right` and `no_access` branches of `deniedOutcome`, and the `requester_lost_right` row state.

Every survivor gets a reader trace before any deletion (memory: surviving mutant is not dead code). Clean the session `GOCACHE` per batch (memory: mutation runs fill the disk).

### Console unit tests (`live-model.test.ts`)

- Rights parsing; missing `rights` → all false.
- `not_permitted` area state.
- The `no_right` reason text in EN, KO and AR.
- `deniedOutcome` returns `no_right` when the reloaded page lacks the right, `no_access` when the reload returns 403, and the existing outcomes otherwise.
- `requester_lost_right` and held-redelivery texts in EN, KO and AR; `own_pending` withdrawal rows.

### Browser QA

Runners are in the scratchpad `/private/tmp/claude-501/-Users-home-project-hybrid-billing/ff7545a1-604d-4178-88ba-25b4b88aab02/scratchpad/`. Copies are made per ticket; the originals are not edited.

- **P02: `qa285fullon297.mjs` → `qa310p02.mjs`.**
  - Groups A–D are re-run with `admin` and `admin2`, which hold all rights. The only expected text change is the rights line replacing "Every command here currently needs installation administration…".
  - New group E, `admin3` (stop and reprocess only):
    - the API clients and Modules tabs, and the Subscriptions panel, show the rights line;
    - the API clients tab shows the withheld text;
    - Modules shows stop and disable on an enabled plugin, and "Needs the Module approval right" on a pending proposal;
    - pause and revoke are available on an active subscription;
    - letter requests are available.
  - EN, KO and AR at 1440 and 390, with RTL checks for AR.
  - Group D (member denied) is unchanged.
- **P02 licence regression: `qa309.mjs`.**
- **Lost right in the browser (group F).** Real paths through M01, no test hook:
  1. `admin2` proposes plugin change X and subscription Y, then a second change Z. `admin` loads P02, then demotes `admin2` to member on M01.
  2. `admin`'s stale P02 page posts approve on X: the result reads "Not approved. It was closed because the person who started it no longer holds the right."; the row shows the closure with the request ID and `admin` as trigger. After reload, Y shows the `requester_lost_right` reason with no approve button.
  3. `admin2`, now a member, opens P02: the denied block (`P02/spec.md:100`).
  4. `admin` re-promotes `admin2`; `admin2`'s full P02 page returns with no new grant, and `admin2` withdraws Z.
  5. Withdrawal by an administrator who holds no right of the area cannot be produced in the browser (no grant path changes a sample profile at run time), so it is API-only (T14) and recorded as not browser-verified.
  Run in EN, KO and AR.
- **M03 regression: `qa267.mjs` and `qa298on297.mjs`.** No integration right appears in the console client list, the grants list, the request form or the preview; the existing checks pass.
- **Screenshots** of the changed panels for the TW review and `screen-review.md`.

## Verification commands

- `go test ./modules/auth/... ./modules/gate/... ./modules/commands/... ./apps/api/...`
- `go vet ./...`
- The mutation run on the targets above.
- Console: `npm test` and `npm run build` in `apps/console`.
- The browser runners above against a worktree preview (memory: local tooling paths).
- Line counts of every changed file stay under 1,500 (`code-structure.md`).
- A comment check on the diff (`no-comments.md`).

## Residuals

- **R1 Verification right.** No verification right (P3). HB-284 already names the missing "verification" permission (`HB-284/plan.md:309`), and there is still no compatibility or test-send command. The right and the command come together in a later ticket.
- **R2 Enrolment.** No grant or revoke path for integration rights in a running installation (P15). Installation-kind grants are also outside the decision ledger (`ledger.go:86-88`, `protectedGrant`), so no durable revocation of one exists: removing a person's right means demotion or suspension, and re-promotion within validity restores it (F4). A non-sample installation has one holder and cannot complete any two-person P02 step. The same block applies to the trust rights (HYB-297 R4) and the bulk right (HYB-299 R6).
- **R3 API right.** `administer_api_clients` gates only the read: P02 has no API client command, OAuth clients are not connected, and service users are managed on M04 under `administer_installation` (`services.go:380`). Withholding the area is not confidentiality (F8).
- **R4 Bulk revocation preview.** It counts trust and bulk holders (`bulkimpact.go:238-246`) but not integration-right holders. Pending integration work of the targets is now counted as rechecked (F1). No planning text requires the holder count.
- **R5 Registration commands.** Commands for publishers, keys, versions, endpoints and marks remain absent (`HB-284/plan.md:369-375`). When they are added they take `administer_modules` or `approve_modules` through the same table, and their approvals join the starter recheck.
- **R6 Non-administrators.** A billing operator ("범위 내 BL", P10) cannot hold an integration right. This follows from this plan's choice of installation-kind rights (Q7), not from the planning text.
- **R7 Seed bypasses the gate.** The integration seed calls modules directly as `admin` and `admin2` (`integrationseed.go:309-310`, `:351`), not through the gate. Both hold the matching rights, so the seeded state is one they could have produced.
- **R8 Translations.** KO and AR copy are AI drafts pending native-language review.
- **R9 Redelivery hold end to end.** No dispatcher is connected, so the hold of a queued redelivery is verified only through `dispatchDeliveries` in tests and the seed; real delivery stays UNVERIFIED.

## Owner questions (defaults apply until answered)

- **Q1 Granularity.** Eight rights: one API right, administration, approval and stop per module and channel area, plus reprocessing. **Default: as specified.** The alternative is a single stop right across areas, which P1 argues against.
- **Q2 Activation request.** P02 `spec.md:74` reads "활성화 요청 | 적격 승인 권한 | 실증·라이선스·범위 검증 | 고정 버전 활성화 계획 | 권한·외부 영향 확인".
  - **Default (interpretation):** the result, "고정 버전 활성화 계획", is an approved proposal. So the approval right gates the step that makes it take effect (`plugin_approve`, `subscription_approve`, `subscription_approve_resume`), and the administration right gates filing it (`plugin_propose`, `subscription_propose`, `subscription_request_resume`). Rationale: `spec.md:37` ("승인 요청≠사람승인") and QA-04 (`detailed-requirements.md:403`) separate the request from the human approval, and the modules already enforce proposer ≠ approver.
  - **Literal alternative:** filing an activation request itself needs the eligible approval right. Mapping: `plugin_propose` with kinds `install`, `upgrade`, `rollback`, `enable` → `approve_modules`; `plugin_propose` `remove` → `administer_modules`; `subscription_propose` and `subscription_request_resume` → `approve_channels`; the approving step then also needs `approve_*` from a different person. Under it, administration rights alone could file almost nothing, and approval would be a peer review between approval-right holders.
- **Q3 Verification right.** **Default: none until a test command exists (R1).**
- **Q4 Subscription revocation.** Revocation is irreversible: a revoked subscription is never resumed. **Default: `stop_channels`**, which lets one stop-only holder remove a subscription permanently with no second person. Rationale: containment of a compromised subscriber must not wait; `trusthold.go:79` already treats it as a stop. **Alternative: `administer_channels`**, leaving pause (reversible) as the only stop-only containment.
- **Q5 Enrolment.** How do further administrators receive integration rights? **Default: seed and bootstrap only, with a truthful block; not through M03.** The same question is open as HYB-297 Q3 and HYB-299 Q5, and one answer should cover all three.
- **Q6 Combined holding and withdrawal.**
  - May one person hold both the administration and the approval right of an area? **Default: yes; per-object self-approval stays refused.** Alternative: refuse a grant profile holding both (a static separation rule no planning text requires).
  - Who may withdraw pending work? Options: (a) the author, with no current right; (b) the author while holding the area's administration right; (c) any holder of the administration right. **Default (root decision): (a)**, because withdrawal only reduces risk (as `trusthold.go:80` treats it), limited by root decision N3 to authors who are still effective administrators (`P02/spec.md:100`). A demoted author cannot withdraw; their items are closed at the next approval attempt, which completes as `closed` and frees the slot (N4).
- **Q7 Non-administrators.** Should a non-administrator (BL within scope) hold P02 access? **Default: no.** The exclusion is a consequence of this plan's installation-kind choice (R6); a workspace-kind read right is the alternative and needs its own review.
- **Q8 Seed profiles.** **Default: `admin` and `admin2` hold all eight; `admin3` holds `stop_modules`, `stop_channels` and `reprocess_events`.**
- **Q9 Page admission.** May `administer_installation` alone open the P02 page with every area withheld? **Default: yes, so the person sees why instead of a bare denial.** A person who is no longer an administrator is never admitted, even with pending items (N3).
- **Q10 Provisional right IDs.** HB-33 leaves "정식 명령/권한 ID·호환 패치 버전·SDK 언어·API 폐기 기간" open (`requirements.md:601`). **Default: the eight IDs are provisional**, marked so in the glossary and OpenAPI (`x-provisional` is already set on the document); renaming later follows the API deprecation decision.
- **Q11 Already-approved processing.** **Default:** active subscriptions and enabled plugins continue when their proposer or approver later loses the right (their approval was valid; stop rights can end them); queued redeliveries not yet delivered are held for re-approval; dismissals take effect at approval. Alternative: suspend active subscriptions automatically, which would make a demotion cause external outages without a decision.

## Review round 1

Security design review, 2026-11-19, read-only reviewer: CHANGES REQUIRED (1 High, 4 Medium, 7 Low). Root decisions are marked.

| # | Severity | Finding | Change in this revision |
| --- | --- | --- | --- |
| F1 | High | Approvals did not recheck that the proposer, resumer or requester still holds the starting right (`retention-access.md:63`); `integration_proposals` was `rechecked: false` | Root decision: implement. "Starter recheck at approval": recheck under the write lock with `holdsRight` plus role and state; `prerequisite_failed` with `requester_lost_right`; read blocker; `rechecked: true` with resume and letter requests tallied; per-command decision for already-approved processing (Q11) including held redeliveries; T12, T13, PT1, mutation targets |
| F2 | Medium | Withdrawal needed a current right; a lost-right entry stayed open | Root decision: author withdrawal without a current right (`Author` definition flag, `CheckAuthor`), three new withdrawal commands, author-only admission, closing and freeing the slot on `requester_lost_right`; Q6 rewritten with options; T14 |
| F3 | Medium | Row P4 omitted the result column; the literal reading was not described | P4 quotes "고정 버전 활성화 계획"; Q2 gives the literal alternative with its command mapping; default and rationale kept |
| F4 | Medium | Demotion and re-promotion behaviour unstated; tests used registry shortcuts | "Role and grants are separate" and "Demotion and re-promotion"; T6 uses real paths per area (role change and back, membership and user suspension, bulk revocation, role-period expiry); `ledger.go:86-88` under R2 |
| F5 | Medium | Partial grant profiles possible at seed and bootstrap | Root decision: all-or-nothing with rollback; T8 injects a failure after k grants |
| F6 | Low | Mixed availability in the nine decisions not specified | Any `authority_unavailable` → whole read 503; one decision set feeds `rights`, areas and `no_right`; T4 |
| F7 | Low | `s.admits` still ran for readers who cannot approve | Skipped at `integrations.go:559`, `:576`; T4; mutation target |
| F8 | Low | Withholding the API clients area implied confidentiality | Stated as a display boundary (M03 ceiling, M04 service users, bulk impact counts); glossary rows likewise; R3 |
| F9 | Low | Right IDs presented as final | Q10: provisional (`requirements.md:601`) |
| F10 | Low | BL exclusion attributed to planning | P10, R6 and Q7 say it follows this plan's installation-kind choice |
| F11 | Low | Q4 hid that revocation is irreversible | Q4 states the one-person permanent removal under the default |
| F12 | Low | Test and contract gaps | T3 per-command "all other rights" profiles and person-without-client / client-without-person; T4 multi-right profiles; OpenAPI 403 with `CodeCSRF`, 503 with `dependency_unavailable`; console `no_access` when the reload returns 403; R1 cites `HB-284/plan.md:309`; P20 quote includes "verification" |

## Review round 2

Security design review round 2, 2026-11-19: F3–F12 PASS; F1 and F2 right in direction, CHANGES REQUIRED (1 High, 4 Medium, 4 Low). Root decisions are marked.

| # | Severity | Finding | Change in this revision |
| --- | --- | --- | --- |
| N1 | High | One letter withdrawal command for both request kinds let the author of one kind withdraw the other | Split into `letter_withdraw_redeliver` and `letter_withdraw_dismiss`; the handler kind fixes the author; `Apply` re-verifies under the lock that the caller is the requester of that kind; T14 cross-kind case; mutation target |
| N2 | Medium | Author check had no commit-time recheck and no client, session or human rules | `CheckAuthor` and `RecheckAuthor` in the executor's `Gate` interface (recheck after the idempotency lookup); refuses empty author, non-human, support session, non-console client; `Author` valid only on mutating, non-approval definitions; T15 engine tests in `commands_test.go` and `property_test.go`; mutation targets |
| N3 | Medium | Member admission to the read contradicted `P02/spec.md:100` | Root decision: `CheckAuthor` requires an effective administrator role; member admission dropped; administrators without the area right still withdraw through `own_pending`; a demoted author's items close at the next approval attempt |
| N4 | Medium | A refusal that also closed the entry was misleading and needed an engine change | Root decision option a: the approval completes with business state `closed` and `reason: requester_lost_right`; closure history records the request ID and the approver as trigger; console shows the closure truthfully and never "Nothing was changed" |
| N5 | Medium | Redelivery hold did not separate requester and approver loss; target and leasing guarantees were server-only | Requester lost → close; only approver lost → hold, and re-approval runs the starter recheck; new `webhook-redelivery:` prefix with its own `Resolve` case and command `redelivery_reapprove`; the webhooks module refuses to lease a held delivery (module test); evaluated under the dispatch lock; R9 stays UNVERIFIED end to end |
| N6 | Low | Order of state prechecks and starter recheck unspecified | State prechecks (`invalid_stage`) run first; the plugins replay branch (`plugins/registry.go:230`) noted; T12 order cases |
| N7 | Low | `Resolve` source of `AuthorID` and replay after withdrawal unspecified; `resume_withdraw` could misroute | `AuthorID` from the server store with no module call, kept after withdrawal; T14 replay case; renamed `subscription_withdraw_resume`; explicit `found()` cases |
| N8 | Low | `own_pending` could call modules for withheld areas; the blocker discloses another person's right status | `own_pending` uses only the server store; the disclosure is stated and judged acceptable (approvers are installation administrators who already see role and state on M01) |
| N9 | Low | No test that admission paths leave output unchanged; no mutation targets for N1/N2 | T16; mutation targets for the per-kind author check and every `CheckAuthor`/`RecheckAuthor` rule |

## Review round 3

Security design review round 3, 2026-11-19: N1–N9 PASS; two Medium and one Low remain. **Review round 3: APPROVE pending R3-1..R3-3** (reviewer: approve once these are in).

| # | Severity | Finding | Change in this revision |
| --- | --- | --- | --- |
| R3-1 | Medium | Some approvals had no explicit "pending" precheck before the starter recheck, so a missing request could read as a lost right | Explicit `invalid_stage` prechecks for `approve_resume` (resumer set, paused or suspended), both letter approvals (a pending request of that kind), `redelivery_reapprove` (still held); an empty starter is `internal_error`, never a lost right; T12 "no pending request" cases on each |
| R3-2 | Medium | The starter recheck reduced the explanation to a boolean, so an outage could close or hold work | Three-way result; authority unavailable → `authority_unavailable`, identity facts failing → `dependency_unavailable`, nothing closed; `dispatchDeliveries` skips the round without closing or holding; injected-fault T12/T13 cases; mutation target |
| R3-3 | Low | Authorship per object let a replay resolve to a newer requester | Root decision: per-request IDs with a tombstone map separate from the live maps; request withdrawals target `webhook-resume-request:` / `webhook-letter-request:`; T14 sequence A requests, A withdraws, B requests, A replays → A's stored result |

## Design review round 4

Design review round 4: APPROVE (notes 1–3 folded in). Root decisions of 2026-11-19 (scratchpad `hb310-notes.md`): fresh-key withdrawal on an older request ID gives `invalid_stage`; tombstones capped at 10,000 per kind with fail-closed replay of evicted IDs; sequential request IDs accepted and noted.

## Design review

- Round 1: CHANGES REQUIRED, addressed. Round 2: CHANGES REQUIRED (F3–F12 PASS), addressed. Round 3: APPROVE pending R3-1..R3-3, applied. Round 4: APPROVE. Root runs a short confirmation and a ui-designer visual spec before implementation. The reviewer receives this plan and the cited code, not root's conclusions.
- Review of the actual change: required after implementation.
- TW review of the copy: required after rendering.
