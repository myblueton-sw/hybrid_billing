---
type: ImplementationPlan
title: M04 credential issue with an unknown outcome leaves a truthful trace on the server and in the console
description: Fix plan for HYB-333. The services read reports, for each active credential, whether its one-time secret pickup is still open, was collected, or was never collected. The console keeps a credential issue whose outcome is unknown or still in flight outside the issue form, lists it as a reminder, guards the page against unloading while it is held, holds unknown create outcomes, and returns focus to the button that opened a revoke step.
status: stable
tags: [HYB-333, M04, service-accounts, credentials, secret-pickup, idempotency, security]
sources:
  - resource: ../HYB-316/verification.md
  - resource: ../HB-268/plan.md
  - resource: ../HB-269/plan.md
  - resource: ../HYB-293/plan.md
  - resource: ../../ui/screens/M04/spec.md
  - resource: ../../planning/v1.16/retention-access.md
  - resource: ../../planning/v1.16/detailed-requirements.md
  - resource: ../../ui/terminology/product-glossary.md
---

# Scope and ownership

[HYB-333](https://linear.app/hybrid-billing/issue/HYB-333) carries the residuals of [HYB-316](../HYB-316/verification.md):

- **N2:** a credential issue can end with an unknown outcome (lost response). If the person then leaves deliberately, the held attempt is dropped and nothing truthful remains. Ways to leave: Back at narrow width, a change in the service selector, a view switch, a Revoke button, a browser reload or closing the tab. The server may already have committed a credential whose one-time secret was never shown. The in-memory "Secrets not confirmed as stored" reminder does not include that credential and does not survive a reload. The `beforeunload` guard runs only for unstored secrets.
- **N5:** `CreateForm` uses `holdUnknown = false`. After a lost create response, the key is renewed, and a retry can report "name taken" for the person's own committed create.
- **Focus:** Cancel on a revoke step opened from the "Secrets not confirmed as stored" reminder sets focus to the credential row. When that row is not on screen, it falls back to `#svc-list-title`, which does not exist in the credentials view, so focus is lost.

Base: `main` `73745d6` (includes the HYB-316 fix). Branch: `fix/HYB-333-m04-unknown-issue`. Worktree: `/private/tmp/hb333`. Machine: mac_mini.

**Ownership**
- Root (Claude Code) owns the plan, the implementation, the tests and the browser QA.
- **Independent security design review (required)** before implementation, then a change review of the actual diff. A separate read-only reviewer (`independent-reviewer`) does both. Focus areas: secret and pickup-reference exposure, session binding, idempotent replay, and the new public read field. Design review rounds 1–3 are recorded under "Review history"; round 3 approved the design. The change review of the diff has not run.
- The `ui-designer` role (AI, read-only) specifies the two new reminder regions and the locked held form, then compares the renders.
- A separate read-only technical-writing review checks the EN, KO and AR copy and the glossary rows.

**Allowed files**
- `apps/api/server`:
  - `directory.go`: the collected map and its sweep;
  - `services.go`: issue, revoke, retire and the read rows;
  - `users.go`: `collect` marks a credential pickup as collected;
  - `openapi.go`: an `x-reads` entry for `/identity/services`;
  - a new `services_pickup_test.go`, kept separate so `services_test.go` stays focused.
- `apps/console/src/service/admin.ts` and `admin.test.ts`: the new row fields, the new `limits.replay_seconds`, and their validation.
- `apps/console/src/service/Service.tsx`: release the issue-attempt guard before a deliberate sign-out (`signOutIn`, line 418) and before the session-ended navigation (`onSignedOut`, line 436). Root decision F5.
- `apps/console/src/modules/identity/pages/service-accounts/`:
  - `Live.tsx`, `live-model.ts`, `live-model.test.ts`, `live-copy.ts`;
  - new `issue-attempts.ts` and `issue-attempts.test.ts`.
- `apps/console/src/shared/unload-intent.ts` and `unload-intent.test.ts` (new), and `apps/console/src/app/App.tsx` (its `beforeunload` handler only). Root decision N5.
- `apps/console/package.json`: the test list only.
- `apps/console/src/modules/identity/pages/users/live.css`: M04-scoped rules only, if the reminder regions need them.
- `apps/console/tools/service-accounts-qa.mjs` and the console README line that describes it.
- `docs/ui/terminology/product-glossary.md`: the new rows.
- `docs/features/HYB-333/*`.

# Planning text and cross-check

The planning documents below exist in the authorized local workspace and are not yet in Git. Quotes are verbatim.

| # | Source | Verbatim text | How this plan meets it |
| --- | --- | --- | --- |
| P1 | `docs/ui/screens/M04/spec.md:37` (자격 상세) | "secret 참조/버전/만료만; 클립보드·로그 비밀 노출 금지" | The new read field states only the pickup state and the pickup end time. It never contains the secret, its prefix-stripped form, or the pickup reference. Nothing new is written to logs, storage or the clipboard. |
| P2 | `M04/spec.md:53` (field contract) | "\| 자격 참조·버전 \| secret reference \| 예 \| 발급/회전 명령 \| 목록은 원문 비밀 열람 불가 \| 비밀 참조 \|" | The list shows a pickup state, not the secret. Only the existing Reveal step shows the secret, once. |
| P3 | `M04/spec.md:68` (action contract) | "모든 쓰기는 expected revision·멱등 키·현재 권한을 확인하고 접수 ID와 완료 상태를 구분한다." | The held issue keeps its original `request_id`, idempotency key and expected revision outside the form. An unknown outcome is shown as unknown, never as issued or not issued. |
| P4 | `M04/spec.md:100` | "결과 불명은 waiting+reason으로 표시한다. 재시도 버튼은 재시도 가능성이 입증된 단계에만 제공한다." | "Try again" appears only while a replay of the same key is possible. After the key's replay window (`idempotency_key_expired`), the entry says the outcome can no longer be checked and offers no retry. |
| P5 | `M04/spec.md:97` (error state) | "\| error \| 자격 교체 확인에 실패했습니다. 유효 버전과 사용처를 확인하고 새 비밀을 로그에 남기지 마세요. \|" | No secret or reference enters logs, the console, notices or live regions (tested). |
| P6 | `M04/spec.md:64` | "없음·미확인·미집계·오류와 실제 0을 다른 텍스트로 표현한다." | "Outcome unknown", "Secret waiting for pickup", "Secret never collected" and the plain active state are distinct texts. A credential with no recorded pickup state shows no claim. |
| P7 | `M04/spec.md:81` | "탭 전환은 필터와 선택 범위를 유지하고 미저장 초안은 이탈 영향을 안내한다." | A held or in-flight issue survives a view switch, a service change and Back. It is listed as a reminder, and a page unload is guarded. |
| P8 | `M04/spec.md:114` (Keyboard) | "dialog는 초점을 가두고 닫으면 호출 요소로 돌려준다." | Cancel or Back on a revoke step returns focus to the button that opened it: a row, the unstored reminder or the never-collected list. The fallback is the current view's heading. |
| P9 | `M04/spec.md:74` | "\| 자격 폐기 \| 폐기 권한 \| 대상 자격 현재 버전 \| deny 우선 확정·후속 정리 \| 연결 중단 범위 확인 \|" | Revoke stays the next step for a credential whose secret nobody holds. Automatic revocation is an owner question (Q1). |
| P10 | `M04/spec.md:132` (미정값) | "- 자격 유형·발급 전달 방식" | Delivery stays HB-268's one-time pickup. This plan adds no new delivery path (Q2). |
| P11 | `planning/v1.16/retention-access.md:24` | "`ServiceAccount`와 그 `Credential`은 사람 로그인과 분리한다. API 키·작업자 자격증명에는 소유 조직·관리 책임자·scope·만료·회전·폐기 상태가 있다." | A pickup state is added to the credential's state facts. Rotation does not exist in the API, so the next step is revoke and issue (Q4). |
| P12 | `retention-access.md:65` | "비밀번호·LDAP bind secret·토큰은 열람 목록에 표시하지 않는다." | Same as P1. The read is tested to contain neither the secret nor the reference. |
| P13 | `planning/v1.16/detailed-requirements.md:403` (QA-04) | "Idempotency-Key 범위는 설치/client/명령이고 동일 키의 다른 요청 hash는409다. 변경은 ETag/If-Match 등 expected revision 조건을 요구한다." | The retry resends the same body, key and expected revision. A replay returns the stored outcome. A first execution with an old revision is refused as `revision_conflict`. The console never changes the body of a held attempt (its inputs are locked). |

Prior design facts used:
- [HB-268](../HB-268/plan.md):
  - the pickup lasts 10 minutes, is bound to the issuing session, and is deleted on first read;
  - suspend, retire, revoke, sign-out and session end drop pickups;
  - its residual: "the listing does not mark it, so HB-269's copy advises revoking credentials whose secret was not saved."
- [HB-269](../HB-269/plan.md): the hold rules, the unstored marks, `beforeunload` and `pagehide` for a shown secret, and the residual "a held issue attempt is lost if the form remounts".
- [HYB-293](../HYB-293/plan.md) and the HYB-316 verification: a `reauthentication_required` refusal never reaches the idempotency journal, so resending the same attempt after password confirmation is a first execution.

# Observed baseline (code facts)

**Server**
- `services.go` `issue`:
  - adds the credential through `AddCredential`;
  - writes `pickup{secret, credential, session, target, expires: now+10m}` under a random 16-byte `ref`;
  - returns only `credential` and `secret_ref` in the outcome.
- `users.go` `collect` (`GET /api/v1/identity/links/{ref}`):
  - runs under `s.domain.Lock()`;
  - refuses an expired pickup or another session with `permission_denied`;
  - deletes the pickup and returns `{secret, credential}`.
- Pickups are also deleted by:
  - `directory.prune` at the start of each service command (expiry);
  - `dropPickups` on suspend, retire, revoke, sign-out (`signin.go`), session end (`self.go`, `selfend.go`, `users.go endSessions`), period end (`periods.go`) and bulk cleanup.
- **After a pickup is gone, the server cannot tell "read" from "expired or dropped".** No per-credential delivery fact is stored. The `credentialRow` in the read has `id`, `not_before`, `not_after`, `state` and `usable` only.
- The server has every fact needed to know it:
  - an open pickup exists in `dir.pickups` (keyed by ref, with `credential`);
  - a successful read happens in exactly one place (`collect`).
- **Command engine order** (`modules/commands/execute.go:340-358`):
  - the idempotency lookup (replay, `idempotency_conflict`, in-flight `dependency_unavailable`, `idempotency_key_expired`) runs before the expected-revision check;
  - so a held retry either replays the committed outcome, or, if nothing was committed, runs as a first execution and meets `revision_conflict` once anything else has changed;
  - `Resolve` and the permission decision run before the lookup, so a lost permission answers `permission_denied` even when the first request committed.
- Replay window: `Replay: time.Hour` (`server.go:164`). After that, the key answers `idempotency_key_expired`, which the console maps to `stale` (`admin.ts:797`). Once the stored operation itself is gone, the same key is treated as new, so a retry would run as a first execution and meet `revision_conflict`. That answer then says nothing about whether the first request committed.
- Sessions:
  - `s.sessions` (`session.go`) holds the server's session records. `get` (line 146) returns a record only while it is present.
  - `signin.go:267` prunes records older than `sessionAbsolute`.
  - `oldest` (`session.go:104`, used at `session.go:219` and `users.go:333`) evicts the oldest session when a login reaches its session limit.
  - Idle expiry is enforced by the identity module through `s.access` in `admitted` (`users.go:96-123`), so a record can still be present in `s.sessions` after its idle end.
  - `collect` admits the caller through `admitted`, so a pickup whose session can no longer be admitted cannot be collected, even if the pickup has not reached `expires`.
- `credentialRows` (`services.go`) is also used by the integrations read (`integrations.go:355`), which only counts usable rows.
- `Revision()` is the command counter. `collect` does not change it, and this plan keeps it that way.
- Identity state, credentials and pickups are in memory and reset on restart (`openapi.go:13`). A restart removes the credentials themselves, so no pickup fact can outlive its credential.

**Console**
- `useCommand` owns an `attemptState` tracker per component (`Live.tsx:55-99`). `IssueForm` passes `holdUnknown = true`. `CreateForm`, `StateStep` and `RevokeStep` pass false.
- `pinIssue` keeps the panel open across a breakpoint change, but any unmount (Back, a selector change, a view switch, a Revoke button) drops the tracker and the held attempt.
- An issue that succeeds after its form unmounted loses its `secret_ref`: `onIssued` belongs to the unmounted form. The pickup then expires unread.
- The unstored marks (`pending`, a module-level map per user) survive a view switch but not a reload. Their `beforeunload` listener is registered only while `LiveServices` is mounted and the list is non-empty.
- Editing an input of a held issue form renews the attempt and silently drops the hold.
- The focus effect falls back to `#svc-list-title` in both views (`Live.tsx:544-548`, the fallback on line 546).
- The unstored marks are the module-level `pending` map (`Live.tsx:109`).
- The `qa269` runner counts `[data-credential]` elements and checks `.live-unstored` counts. New elements must not use either.

# Design

## Alternatives considered

| Option | Outcome |
| --- | --- |
| A. Console-only reminder for unknown issues | Rejected alone: lost on reload or tab close, invisible to other administrators and other sessions. |
| B. Server-side pickup state on the credential row | **Chosen.** The server alone knows whether a pickup was read, and the read survives reloads and is visible to every administrator. |
| C. Keep the held or in-flight attempt outside the form, in page memory per user and service, with a reminder and an unload guard | **Chosen with B.** This is the only way to still get the secret after leaving and coming back within the pickup window, through a replay of the same key. |
| D. Persist the held attempt in `sessionStorage` to survive a reload | Rejected. A replay returns `secret_ref`, which the same session cookie can exchange for the secret. Storage would widen who can obtain it, and HB-269 keeps nothing secret-adjacent in storage. |
| E. Return the open pickup's `ref` in the services read to its issuing session, to allow "Show secret" after a reload | Rejected by default (Q2). It puts a bearer reference into a broad, repeated read. Revoke and issue again is cheap and safe. |
| F. Revoke a credential automatically when its pickup closes unread | Not in this ticket (Q1). It changes the credential lifecycle and the audit meaning of a revoke. Pre-launch credentials cannot authenticate anyway (HB-268 residual). |

## Server: pickup state per credential

**Data**
- `directory.collected map[string]bool`, keyed by credential ID.
- `issue` sets `collected[id] = false` in the same domain write that adds the credential and its pickup.
- `collect` sets `collected[p.credential] = true` for a credential pickup, before writing the response, under the existing lock. A link pickup (M01) is unchanged.
- No secret, verifier or reference is stored in this map. It does not change `Revision()`.

**Where the derivation lives (F10).**
- A new function `pickupStates(rows []credentialRow, now time.Time)` is called only from `listServices`, after `credentialRows`.
- `credentialRows` itself is unchanged, so the integrations read (`integrations.go:355`) neither computes nor exposes the new fields. The `credentialRow` fields are `omitempty`, so they stay absent there.
- A test asserts that the integrations read body carries no `secret_pickup`.

**Derivation in `GET /api/v1/identity/services`.** For each credential row whose `state` is `active`:
- **`open`**: a pickup for this credential exists, `now < expires`, and its issuing session is still live (F3).
  - The read is under `RLock`, so it builds a credential-to-pickup index from `dir.pickups` and ignores expired entries without deleting them.
  - It also sets `secret_pickup_until` to `min(expires, the session's absolute end, the session's idle end as of this read)` (N4). This is an estimate, not a promise: activity can move the idle end later, and a sign-out, eviction or permission change can end the pickup sooner. The copy says "estimated" (see `pickupOpen`).
- **Live session** means all of:
  - `s.sessions.get(p.session)` returns a record;
  - the identity module would still admit it, meaning neither the idle end nor the absolute end has passed;
  - a side-effect-free gate decision for that session's answer allows `AdministerInstallation` on the installation kind, the same request `admitted` makes (`users.go:112-113`) (N4, root decision).

  So `open` means that `collect` by the issuing session would succeed at the moment of the read.
  - The implementer confirms that `gate.Decide` records nothing (no audit, counter or cache write).
  - If it does record something, the implementer adds a read-only variant instead and records that in the verification.
  - The check must have no side effects: it must not touch the idle clock or end the session.
  - If the module has no read-only admission check, the implementer adds one read-only method to `modules/identity` (an allowed-file addition, recorded in the verification) instead of calling `s.access`.
  - This makes `open` mean exactly "`collect` would still admit the issuing session". A pickup whose session was pruned (`signin.go:267`), evicted (`oldest`), idle-expired or absolutely expired reads `never_collected`, even if its own `expires` has not passed.
- **`collected`**: there is no open pickup and `collected[id]` is true. This means the server returned the secret once to the issuing session. It does **not** mean anyone stored it, because a lost response looks the same.
- **`never_collected`**: there is no open pickup and `collected[id]` is false. The pickup ended unread. Any of these ends it:
  - it expired;
  - the issuing session signed out, ended, was pruned, was evicted, or passed its idle or absolute end;
  - the account was suspended or retired;
  - a role-period end or a bulk cleanup dropped it.

  No one can obtain this secret from the server any more. A pickup that is still stored but whose session is no longer live counts as ended (F3).
- **No entry:** both fields are omitted, so the read makes no claim. By construction this cannot happen for an active credential; a test asserts it.
- **Rows that are not active** (`expired`, `revoked`) never carry the fields.

**Bound and sweep**
- `revoke_credential` deletes the entry.
- `retire` deletes entries for the credentials it revokes.
- `issue` deletes entries of the same service user whose credential is no longer active, using the credential list it already reads for the limit check.
- Entries are only created by `issue`, so each service user holds at most `credentials_per_service` (10) live entries plus the one being added. Expired ones linger only until that service user's next issue, revoke or retire. Across 200 service users that is at most about 2,200 entries.

**Replay.** A replayed issue returns the stored outcome and does not touch the map, so there is no second entry or second credential. If the pickup was already read, the replayed `secret_ref` answers `permission_denied` and the console shows "can no longer be shown" (unchanged).

**Contract** (public read; additive):

```
credentials[]: {
  id, not_before, not_after, state, usable,
  secret_pickup?:       "open" | "collected" | "never_collected"
  secret_pickup_until?: RFC 3339 time, only with "open"
}
```

- `limits` gains `replay_seconds`: the command registry's replay window (`commands.Limits.Replay`, 3600 today), read from the same configuration the registry uses. This is root decision F2: the console takes the window from the server and never hard-codes it.
- `openapi.go` adds `x-reads["/identity/services"]` describing these fields and `limits.replay_seconds`, their enum, when they are present, and the sentence "`collected` means the server returned the secret once; it does not confirm the secret was stored."
- No command, outcome field, error code or permission changes. The read keeps its admission: a current session, `AdministerInstallation`, and the domain read lock. Another administrator sees the same states, but cannot collect another session's pickup (unchanged).

## Console: the issue attempt outside the form

**`issue-attempts.ts`** is a new, React-free module-level store per signed-in user, keyed by service ID. It holds one entry per service:

```
{ service, track: attemptState, days, reason,
  phase: "sending" | "unknown" | "issued" | "lapsed",
  retry: boolean,            true while the current send resends an unknown attempt
  firstSentMono: number,     performance.now() at the attempt's first send
  firstSentWall: number,     Date.now() at the attempt's first send
  issued?: { ref, credential, until } }
```

**Phase table (root decision F2).** The rows cover every `AdminFailure` value. "First" is a send of a new attempt; "retry" resends an `unknown` attempt with the same `request_id`, key, body and expected revision. "Remove" means the answer proves the attempt did not commit: the engine's replay lookup runs before these checks (`execute.go:340-358`), and the retry is still inside the replay window. One test per row.

| Answer (code) | After a first send | After a retry |
| --- | --- | --- |
| Success, first execution or replay | `issued` | `issued` |
| `busy` (`dependency_unavailable`, including in-flight), `unavailable` (network, malformed or no answer) | `unknown` | `unknown` |
| `partial` (`internal_error` with an operation ID) | `unknown` | `lapsed`. A stored failure replays the same answer, so retrying cannot help; never removed. |
| `limited`, `unauthenticated` (`rate_limited`, `unauthenticated`, `csrf_refused`) | `unknown`. The console cannot prove the request never reached execution. | `unknown` (phase kept) |
| `reauth` | The HYB-293 dialog resends the same frozen request once; the phase follows that answer. A second `reauth` after that one resend is treated as `reauth_cancelled` (N7). | Same; a second `reauth` keeps `unknown` |
| `reauth_cancelled` | Remove. A step-up refusal never reaches the journal (HYB-316 verification). | `unknown` (phase kept) |
| `conflict` (`revision_conflict`) | Remove | Remove while `elapsed < window − margin`; otherwise `lapsed`, never "not issued" |
| `stale` (`idempotency_conflict`) | `lapsed` | `lapsed` |
| `stale` (`idempotency_key_expired`) | `lapsed` | `lapsed` |
| `refused`, `stage`, `invalid` | Remove | Remove (within the window, as for `conflict`) |
| `denied` (`permission_denied`) | Remove | `lapsed`. Permission is checked before the replay lookup, so the answer is ambiguous. |
| `wrong`, `locked`, `form`, `self`, `epoch`, `approval_stale`, `evidence_stale`, `needs_approval` (not produced by `service.issue_credential`) | `lapsed` | `lapsed` |

**Replay window**
- `window` is `limits.replay_seconds` from the services read; `margin` is `min(60 s, window / 10)`.
- `firstSentMono` and `firstSentWall` are set at the first send and never changed by retries; `elapsed` is defined below (N1).
- The client-measured time since the first send overstates the server's time since the commit, because the commit happens after the send. Only the network delay of the retry is unaccounted for; the margin covers it.
- **Elapsed time (N1):**
  - The entry records both `firstSentMono` (`performance.now()`) and `firstSentWall` (`Date.now()`).
  - `elapsed = max(performance.now() − firstSentMono, Date.now() − firstSentWall)`.
  - A suspended laptop or a throttled background tab can stop the monotonic clock while server time moves on. Taking the larger value never understates the time.
  - A wall clock moved backwards cannot shorten `elapsed` below the monotonic value.
- Before a retry, the store checks `elapsed ≥ window − margin`. If so, it moves the entry to `lapsed` **without sending**.
- A missing or non-positive `replay_seconds` counts as an unknown window: a retry is still allowed, but a `conflict` answer to a retry becomes `lapsed`, never removed.

**Sending is locked (F1)**
- **Renew is split (N3):**
  - `attemptState` and the store expose a user-triggered `renew`, which is refused while sending.
  - The failure path first leaves `sending` (`idle`), and only then applies its own renew. So a definitive failure still renews the attempt, in every form that uses `useCommand`: `IssueForm`, `CreateForm`, `StateStep` and `RevokeStep`.
  - The failure handling of `useCommand` moves into a pure `settleFailure(track, error, holdUnknown)` in `live-model.ts`, so the order can be unit-tested.
- While the phase is `sending`, the user-triggered `renew` and Discard are refused. `IssueForm` and `CreateForm` show their inputs `readOnly`, each with `aria-describedby` pointing to the held lead (`heldLead` or `createHeldLead`), and disable "Discard and start over".
- This closes the window where typing during a slow request renewed the attempt, so a lost response would be retried with a new key.

**Other store transitions**
- **Reveal and revoke:** an `issued` entry is consumed only by Show secret, which opens the Reveal, removes the entry and adds the unstored mark (unchanged rule), or by a successful Revoke from its row (F6).
- **Read refresh (F4):** on each services read, an `issued` entry is dropped when its credential row is not `active` with `secret_pickup = open`. That includes `collected`, `never_collected`, no claim, revoked, expired, or the row missing. The secret can no longer be shown through that entry; the never-collected region or the row takes over.
- **Fresh reads only (N2):**
  - The store keeps a read sequence. `LiveServices` takes a number from it when a services read starts, which extends the existing `order` sequence.
  - When an entry becomes `issued`, it records the current sequence value (`issuedAfter`).
  - A read may drop the entry only if it started after that point (`readSeq > issuedAfter`).
  - A read that started before the outcome arrived and finished after it can predate the credential, so it never drops the entry.
- **Discard:** removes an `unknown` or `lapsed` entry. It is not offered for `issued`, and is disabled while `sending`.
- The guard is re-evaluated after every transition.

**Store is the single writer (F6)**
- The command continuation writes only to the store, never to component state. So an issue that completes after its form unmounted still records `issued` with its `ref`.
- `LiveServices` subscribes to the store and derives every display from it: the form's held state, the reminder rows, and opening the Reveal.
- **The one automatic consumption:** when an entry becomes `issued` and its form is visible, the parent invokes that entry's Show secret action. Otherwise the entry waits for the person.
  - **"Visible" (N7)** means all of:
    - the view is `credentials`;
    - the mode kind is `form`;
    - `current.id` equals the entry's service;
    - `panelShown(wide, true, open)` is true;
    - `document.visibilityState === "visible"`.
  - If the page is hidden when the entry becomes `issued`, the entry waits for the person.
- **Submit guard:** Submit is refused, and the submit button disabled, while the entry for that service is `sending` or `issued`. A second issue cannot start while the first one's secret is still waiting.

**`IssueForm`**
- Obtains its tracker and values from the store for `(session.user, service)`. A remount for the same service therefore restores the held attempt, the days, the reason and "Try again".
- While the phase is `sending` or `unknown`:
  - the days and reason inputs are `readOnly` with `aria-describedby` to the held lead;
  - in `unknown`, the form shows "Try again" (primary) and "Discard and start over" (secondary), plus a lead that explains the replay;
  - this replaces HB-269's "editing an input starts a new attempt" for held issues only, so a request body cannot change under a held key.
- In `lapsed`, the form shows the lapsed text and "Discard and start over" only.

**Unload guard.** The store owns one module-level `beforeunload` listener.
- It is registered while any entry for the current user is `sending`, `unknown` or `issued`, or while the unstored marks are non-empty. `lapsed` entries do not guard.
- The unstored marks move into the same store, so the guard no longer depends on `LiveServices` being mounted.
- **Deliberate leave covers every guard (N5, root decision).** A new module `apps/console/src/shared/unload-intent.ts` exports `leaveDeliberately()` and `leaving()`.
  - Each of the three `beforeunload` handlers returns without `preventDefault` while `leaving()` is true:
    - the store's guard;
    - the Reveal's own guard (`Live.tsx:362-366`);
    - the App dirty guard (`App.tsx:265-274`).
  - `Service.tsx` calls `leaveDeliberately()` right before `location.assign` in `signOutIn` and in `onSignedOut`. These are the only two callers (L2).
  - The flag is cleared on `pageshow` with `event.persisted` true (a back-forward cache restore) and whenever a new session is set in `Service.tsx` (L1), so a restored or newly signed-in page is guarded again.
  - Allowed files gain `apps/console/src/app/App.tsx` (that handler only) and the new `unload-intent.ts` with its test.
  - A Reveal that was showing a secret still drops it on `pagehide` (unchanged).
- **Release (F5):** `Service.tsx` calls the store's `release(user)` before the deliberate sign-out navigation (`signOutIn`) and before the session-ended navigation (`onSignedOut`).
  - It removes the listener and clears that user's entries and marks.
  - The server drops the session's pickups at sign-out, so those pickups become `never_collected`, which the region shows after the next sign-in.
  - Neither navigation then raises a `beforeunload` prompt.

**Reminder region "Credential issues to check"**
- Shown at the top of the credentials view. A one-line note with a link to it is shown in the accounts view.
- One row per entry, with the service ID in an LTR isolate:
  - `sending`: an "Issuing…" text, no actions;
  - `unknown`: "Open issue form" (selects the service and opens the panel at narrow width) and "Discard";
  - `issued`: "Show secret" (opens the Reveal with the stored `ref`, `credential` and `until`) and "Revoke" (opens the revoke step with the reason "Secret not shown at issue");
  - `lapsed`: "Discard" only.
- **Selectors:** the region uses the class `live-attempts`. Each row uses `data-attempt="<service>"`, and its Revoke button uses `data-attempt-revoke="<service>"`. They never use `.live-unstored` or `data-credential`, so `qa269` counts are unaffected.
- No reference, key or secret appears in the DOM, in `aria-*` attributes or in live regions. Only the service ID and the credential ID do.

**Never-collected region "Credentials whose secret was never collected"**
- Derived from the services read, across all services. It is the trace that survives a reload or a tab close.
- Lists every active credential with `secret_pickup = never_collected`, with its service, end date and "Revoke" (prefilled reason "Secret never collected").
- Each row's Revoke button uses `data-never-revoke="<credential>"`.
- An entry that is also in the unstored marks is listed only here, because the server fact is stronger.

**Credential row status**
- `never_collected`: the status cell keeps "Active" or "On hold" and adds the strong sub-line "Secret never collected".
- `open`: adds "Secret waiting for pickup until {time}".
- `collected` and no claim: unchanged, so `qa269`'s "credential listed … Active" still holds.

## CreateForm holds unknown outcomes (N5)

- `CreateForm` uses `holdUnknown = true`. On an answer whose phase-table row gives `unknown`, it keeps the attempt and the expected revision and shows "Try again". It follows the same phase table as issue, except that there is no `issued` phase: success goes to the detail.
- While sending and while held, the name, owner and reason inputs are read-only, with `aria-describedby` to `createHeldLead` (F1).
- A "Discard and start over" button renews the attempt. It is disabled while sending, and `attemptState.renew` refuses during a send.
- Submitting pins the panel open, mirroring `pinIssue`, so a breakpoint change does not unmount a held create.
- A replay of a committed create returns `service_created`, and the form proceeds to the detail as for a first success.
- **After a deliberate leave** (the hold is not lifted out of the form for create, see Residuals): a later create with the same name gets `invalid_request` on `name`.
  - If the reloaded page lists `svc-<name>` with `creator` equal to the signed-in user, the field error reads "svc-{name} already exists and you created it. Your earlier request may have gone through; check the list."
  - Otherwise it keeps the existing "name taken" text.
  - This is a display rule over existing read data, not an inference about which request created it.

## Focus return for revoke

- The `revoke` mode gains `from`: the selector of the invoking button. Each Revoke source has its own attribute (F7), so a selector matches exactly one button:
  - `[data-credential="id"]`: the row's Revoke button (existing);
  - `[data-unstored-revoke="id"]`: the unstored reminder's Revoke button (new);
  - `[data-never-revoke="id"]`: the never-collected region;
  - `[data-attempt-revoke="service"]`: the issue-attempt row;
  - the Reveal's "I didn't store it: revoke" and its "gone" Revoke button: `[data-credential="id"]`. The Reveal is gone once the revoke step opens, so focus returns to that credential's row.
- Cancel and narrow Back focus `from` if it is on screen.
- Otherwise they focus the current view's heading: `#svc-cred-title` in credentials and `#svc-list-title` in accounts.
- The generic fallback in the focus effect becomes view-aware in the same way.
- A pure helper `returnFocus(from, credentialsView)` returns the selector list in order. Its unit test covers each of the five sources, including the Reveal case, in both views.

## Copy (EN / KO / AR)

The KO and AR texts are AI drafts pending native-language review. KO keeps M04's current spacing "비밀 값". The glossary's secret-pickup row uses "비밀값 수령 링크" (Q5). Placeholders are wrapped in the existing `iso()` isolates.

| Key | EN | KO | AR |
| --- | --- | --- | --- |
| `pickupOpen` | Secret not collected yet · pickup open, estimated to end {time} | 비밀 값 미수령 · 수령 가능, 예상 종료 {time} | لم تُستلَم القيمة السرية بعد · الاستلام متاح، ويُقدَّر انتهاؤه في {time} |
| `pickupNever` | Secret never collected | 비밀 값 수령 안 됨 | لم تُستلَم القيمة السرية |
| `neverTitle` | Credentials whose secret was never collected | 비밀 값이 수령되지 않은 자격 증명 | بيانات اعتماد لم تُستلَم قيمتها السرية |
| `neverLead` | No one received the secret of these credentials, and it can't be shown any more. Revoke each one, and issue a new credential if you need one. | 이 자격 증명의 비밀 값은 아무도 받지 않았고 더 이상 표시할 수 없습니다. 각각 폐기하고, 필요하면 새로 발급하세요. | لم يستلم أحد القيمة السرية لبيانات الاعتماد هذه، ولم يعد بالإمكان عرضها. أبطِل كلًّا منها، وأصدر بيانات اعتماد جديدة عند الحاجة. |
| `revokeNeverReason` | Secret never collected | 비밀 값 수령 안 됨 | لم تُستلَم القيمة السرية |
| `attemptsTitle` | Credential issues to check | 확인이 필요한 자격 증명 발급 | عمليات إصدار بيانات اعتماد تحتاج إلى تحقق |
| `attemptSending` | Issuing a credential for {service}… | {service} 자격 증명 발급 중… | جارٍ إصدار بيانات اعتماد لـ {service}… |
| `attemptUnknown` | Issue for {service}: outcome unknown. The server may already have issued a credential. | {service} 발급 결과 확인 안 됨. 서버가 이미 자격 증명을 발급했을 수 있습니다. | إصدار لـ {service}: النتيجة غير معروفة. ربما أصدر الخادم بيانات الاعتماد بالفعل. |
| `attemptIssued` | Credential {credential} was issued for {service}. Its secret has not been shown yet. | {service}에 자격 증명 {credential}이(가) 발급되었습니다. 비밀 값은 아직 표시되지 않았습니다. | صدرت بيانات الاعتماد {credential} لـ {service}. لم تُعرض قيمتها السرية بعد. |
| `attemptLapsed` | Issue for {service}: the outcome can no longer be checked. If a credential of this account shows "Secret never collected", revoke it. | {service} 발급 결과를 더 이상 확인할 수 없습니다. 이 계정에 '비밀 값 수령 안 됨' 자격 증명이 있으면 폐기하세요. | إصدار لـ {service}: لم يعد بالإمكان التحقق من النتيجة. إن ظهرت لبيانات اعتماد هذا الحساب عبارة «لم تُستلَم القيمة السرية» فأبطِلها. |
| `attemptsDiscardNote` | Discarding removes this reminder only. Any credential the server issued stays in the list. | 버리면 이 알림만 사라집니다. 서버가 발급한 자격 증명은 목록에 그대로 남습니다. | التجاهل يزيل هذا التذكير فقط. تبقى أي بيانات اعتماد أصدرها الخادم في القائمة. |
| `openIssue` | Open issue form | 발급 양식 열기 | فتح نموذج الإصدار |
| `discardAttempt` | Discard | 버리기 | تجاهل |
| `startOver` | Discard and start over | 버리고 새로 시작 | تجاهل والبدء من جديد |
| `heldLead` | The outcome of this request is unknown. Try again sends the same request. If the server already issued the credential, you get that credential and its secret, not a second one. | 이 요청의 결과를 확인하지 못했습니다. 다시 시도하면 같은 요청을 보냅니다. 서버가 이미 발급했다면 두 번째 자격 증명이 아니라 그 자격 증명과 비밀 값을 받습니다. | نتيجة هذا الطلب غير معروفة. تؤدي إعادة المحاولة إلى إرسال الطلب نفسه. إن كان الخادم قد أصدر بيانات الاعتماد فستحصل عليها وعلى قيمتها السرية، لا على بيانات اعتماد ثانية. |
| `revokeNotShownReason` | Secret not shown at issue | 발급 시 비밀 값 미표시 | لم تُعرض القيمة السرية عند الإصدار |
| `accountsAttemptsNote` | {n} credential issues need checking on the Credentials tab. | 자격 증명 탭에서 확인이 필요한 발급이 {n}건 있습니다. | توجد {n} عمليات إصدار بيانات اعتماد تحتاج إلى تحقق في تبويب «بيانات الاعتماد». |
| `createHeldLead` | The outcome of this request is unknown. Try again sends the same request and does not create a second account. | 이 요청의 결과를 확인하지 못했습니다. 다시 시도하면 같은 요청을 보내며 계정을 두 번 만들지 않습니다. | نتيجة هذا الطلب غير معروفة. تؤدي إعادة المحاولة إلى إرسال الطلب نفسه ولا تُنشئ حسابًا ثانيًا. |
| `errNameTakenMine` | {id} already exists and you created it. Your earlier request may have gone through; check the list. | {id}은(는) 이미 있으며 내가 만든 계정입니다. 이전 요청이 처리되었을 수 있으니 목록을 확인하세요. | {id} موجود بالفعل وقد أنشأته أنت. ربما نُفِّذ طلبك السابق؛ تحقّق من القائمة. |

- AR plural forms for `{n}` follow the existing `count` helper pattern. The implementer checks this with TW.
- "Outcome unknown" follows the glossary pattern of "Activation not confirmed" (`activation-unknown`).

**Glossary rows to add**
- `credential-secret-pickup-state` / `secret_pickup` `open`, `collected`, `never_collected`.
  - Forbidden implications: "collected" is not "stored"; "never collected" is not "revoked" and not "unusable".
- `issue-outcome-unknown`.
  - Forbidden implications: not "failed" and not "issued".

# Data flow and failure conditions

| Situation | Server trace | Console trace (same page) | After reload |
| --- | --- | --- | --- |
| Issue response lost, person stays and retries | Replay returns the same outcome. The pickup is open, then collected. | `unknown`, then `issued`, then Reveal | Row has no pickup claim, or `collected` |
| Lost, then Back, a selector change, a view switch or Revoke | Pickup `open` for up to 10 minutes, then `never_collected` | `unknown` entry. "Open issue form" restores the held form. | Never-collected region (after the pickup ends) |
| In flight, then the person leaves, and the response arrives | `open` | `issued` entry with "Show secret" | `never_collected` region after 10 minutes |
| Lost, then reload or tab close | `open`, then `never_collected` | The unload guard prompts first | Row sub-line, then the region, with Revoke |
| Lost, and the first request did not commit | No credential | Retry runs once. Success issues, `revision_conflict` removes the entry. | Nothing to show (truthful) |
| Retry once `elapsed ≥ replay_seconds − margin` | Unchanged | `lapsed` without sending | Region, if the credential was issued |
| Typing during a slow request, then the response is lost | Whatever the first request did | Inputs locked while `sending`; the retry carries the same key | As above |
| Sign-out or session end with an open pickup | `never_collected` (pickups dropped; any surviving pickup reads `never_collected` once its session is not live) | `release(user)` clears the store and the guard before navigating | Region |
| `issued` entry, but its pickup closes before Show secret | `never_collected` | The next read drops the entry (F4) | Region |
| Collect response lost | `collected` (cannot be told apart) | Reveal "can no longer be shown" + Revoke; unstored mark kept | Row unchanged; the unstored mark is lost (residual) |

# Tests (fail-first) and verification

## Go: `apps/api/server/services_pickup_test.go`

Each test fails on `main` `73745d6`, because the fields are absent.

1. **Issue with no collect:** the row has `secret_pickup = "open"` and `secret_pickup_until = start+10m`. The raw read body contains neither the secret nor the `secret_ref`.
2. **After a collect:** `collected`. A failed collect (another session, after expiry, a second read) does not mark it collected.
3. **Clock advanced past 10 minutes, no collect:** `never_collected`, both before and after a later service command prunes the pickup.
4. **Pickup dropped without a read:** `never_collected` for each of:
   - sign-out of the issuing session;
   - session end through M05;
   - suspend, then the same row after reinstate;
   - a bulk revocation cleanup that drops the issuing administrator's pickups (`bulkcleanup.go`) (F9);
   - a role-period end that drops them (`periods.go:221`) (F9).
4a. **Session no longer live while the pickup is stored (F3):** `never_collected`, and `collect` refuses, for each of:
   - idle expiry of the issuing session, with the pickup's own 10 minutes not yet passed (a shorter configured idle time, or the test clock);
   - absolute expiry;
   - eviction through `oldest` when the same login opens sessions up to the limit;
   - `sessions.prune` at sign-in;
   - the issuing administrator loses `AdministerInstallation` (installation grant removed or expired) while the session stays live (N4).

   `secret_pickup_until` equals `min(expires, absolute end, idle end)`. A test sets each of the three as the smallest in turn.
   The read must not change the session's idle clock: a test reads repeatedly and then shows the idle end unchanged.
4b. **Integrations read (F10):** the `/integrations` body carries no `secret_pickup`, and its usable counts are unchanged.
4c. **Replay window field:** `limits.replay_seconds` equals the registry's configured `Replay` (3600 with the default configuration, and a changed value in a test configuration).
5. **Rows that are not active:** revoked and expired rows carry neither field. Retire and revoke delete entries. A later issue sweeps ended entries (white-box size check). The map never exceeds live credentials plus one per service user.
6. **Replay:**
   - a replayed issue with the same key creates no second credential and no second entry;
   - after a collect, the replay's `secret_ref` answers 403 and the state stays `collected`.
7. **Other administrator:** another administrator's read shows the same state for the first session's pickup and cannot collect it.
8. **Invariant:** the property test runs random sequences of issue, collect, clock advance, revoke, suspend, reinstate, retire, sign-out, M05 session end and new sign-ins up to the session limit (F9). After every step:
   - every active credential row has `secret_pickup`;
   - `open` is reported exactly when a `collect` by the issuing session would succeed at that moment;
   - `collected` only follows a successful `collect`.
   It uses the existing `identity_property_test.go` helpers where they fit.
9. **Lock and revision:**
   - a collect does not change `Revision()`;
   - `go test -race` covers concurrent reads with collect.
10. **OpenAPI:** `x-reads["/identity/services"]` lists the enum and the presence rules.

Checks:
- `gofmt` and `go vet`;
- `go test ./apps/api/... ./modules/...` and `-race -count=20` on the new tests;
- statement coverage of the changed functions;
- mutation testing of the changed lines in `services.go`, `users.go` and `directory.go`, cleaning the session `GOCACHE` per batch;
- 386 and Windows cross builds.

## Console unit tests

**`issue-attempts.test.ts`** (new; added to the `npm test` list)
- A second `entry(user, service)` returns the same tracker and values, which models a remount.
- `sending` followed by an unknown failure keeps the same `request` and `key` and the original expected revision.
- **F1:**
  - `renew` and Discard are refused while `sending`, both in the store and in `attemptState`, so `CreateForm`'s own tracker refuses them too;
  - a renew attempted during a send, followed by an unknown failure and a retry, sends the same key.
  - Fails on `main`, where `attemptState.renew` always renews.
- **F2:** one test per row of the phase table, for both a first send and a retry, including:
  - `idempotency_conflict` gives `lapsed`;
  - `partial` on a retry gives `lapsed`;
  - `limited` and `unauthenticated` keep the phase;
  - `reauth_cancelled` on a first send removes the entry and on a retry keeps `unknown`;
  - `conflict` on a retry is removed inside the window and gives `lapsed` at or after `window − margin` (injected clock);
  - a retry at or after `window − margin` gives `lapsed` with no send (the fake command client records zero calls);
  - a missing `replay_seconds` makes a retry `conflict` give `lapsed`.
- A success delivered after the entry's form is gone records `issued` with `ref`. The continuation writes only to the store, so the subscriber is notified and no component setter is needed (F6).
- **F6:** Submit is refused while the service's entry is `sending` or `issued`. `issued` is consumed only by Show secret or a successful Revoke.
- **F4:**
  - a read refresh drops an `issued` entry whose row is `collected`, `never_collected`, without a claim, revoked, expired or missing;
  - it keeps one whose row is `open`;
  - the guard is re-evaluated afterwards.
- Discard removes `unknown` and `lapsed`, is refused for `issued`, and is refused while `sending`.
- The guard predicate is true for `sending`, `unknown`, `issued` or non-empty unstored marks, and false otherwise. The listener is added once and removed at zero.
- **F5:** `release(user)` removes the listener and that user's entries and marks.
- **N1:**
  - with an injected wall clock that advances past `window − margin` while the injected monotonic clock does not move, the retry gives `lapsed` with no send;
  - with a wall clock that goes backwards while the monotonic clock passes the limit, the result is also `lapsed`.
- **N2:**
  - a read that started before the entry became `issued` and returned a row without the credential (stale) does not drop the entry;
  - a read started afterwards with the same row content does.
- **N3:** `settleFailure` tests, each followed by the next send:
  - for `holdUnknown` false (`CreateForm` after a definitive answer, `StateStep`, `RevokeStep`) and true (`IssueForm`), a definitive failure renews while the tracker was sending, so the next send has a new key;
  - an unknown failure with `holdUnknown` true keeps the key;
  - a user `renew` during a send is still refused.
  - Fails on the round-1 design, where renew during sending was refused for the failure path too.
- **N7:** a second `reauth` after the single resend removes the entry after a first send and keeps `unknown` after a retry. The "visible" predicate is true only when all five conditions hold, including `visibilityState`.
- **`unload-intent.test.ts`:**
  - after `leaveDeliberately()`, a dispatched cancelable `beforeunload` is not `defaultPrevented` by the store guard, the Reveal guard or the App handler (each handler tested as a function);
  - **L1:** after a `pageshow` event with `persisted` true, and after a new session is set, `leaving()` is false again and the guards prevent unload again;
  - **L2:** a source test asserts that `leaveDeliberately(` occurs in `apps/console/src` only in `Service.tsx`, exactly twice, inside `signOutIn` and `onSignedOut` (excluding the module itself and tests).
- Users are isolated.

**`live-model.test.ts`**
- `pickupLabel` for each state, including no claim.
- `neverCollected(page)` across services, excluding non-active rows.
- `returnFocus` order for each of the five invokers (row, unstored reminder, never-collected region, attempt row, and Reveal → `[data-credential="id"]`) in both views (F7).
- The create hold uses the `keepsHold` rule.
- `takenByMe(page, name, user)`.

**`admin.test.ts`**
- `services()` keeps a valid `secret_pickup`, `secret_pickup_until` and `limits.replay_seconds`. A missing or non-positive `replay_seconds` becomes 0, meaning unknown.
- It drops unknown values or a malformed time to "no claim".
- It never copies a field named `secret` or `secret_ref` into the row.

Fail-first: the console tests import exports that do not exist on `main`, so they fail there and pass after the change. The browser tool below provides the behavioural fail-first evidence against the pre-fix bundle.

## Browser: `tools/service-accounts-qa.mjs`, new `--only` scenarios

The four existing scenarios stay unchanged. Each new scenario runs against a real API (`--seed sample`, a fresh mode-0700 data directory) and asserts:
- the same `request_id` and `idempotency_key` wherever a retry is expected;
- that no `secret_ref` or `hbsk_` value appears in `localStorage`, `sessionStorage`, the DOM outside `#svc-secret`, or the console.

**Harness rules (F8)**
- **Fresh page per scenario:** each scenario opens a new page in the same signed-in context and closes it at the end. One scenario's store cannot leak into the next.
- **In-app navigation only for "leave":** scenarios that leave and return use the console's own controls: the M04 tab links, the service select, the Back button and the Revoke buttons. They never use `page.goto`, which would reload and empty the page-memory store.
- **No-reload marker:**
  - before leaving, the tool sets `window.__hb333 = <random token>`;
  - after returning, it asserts the token is unchanged;
  - a scenario whose marker is gone fails as "reloaded", not as a product failure.
- **Deterministic dialogs:**
  - every page has a `dialog` handler that records the dialog and fails the scenario on any unexpected dialog;
  - only the `reload` scenario arms a one-shot handler, before calling `page.reload()`. That handler asserts `dialog.type() === "beforeunload"` and accepts it;
  - the sign-out step asserts that no dialog appeared, which checks the F5 release.
- **Synthetic answers are labelled:** where a scenario route-fulfils an answer (F2 rows), the result line says "synthetic" and records the real request that was intercepted.

| Scenario | Steps | Expected |
| --- | --- | --- |
| `leave` | Lost issue response. Switch to accounts (the note is visible), then back to credentials. | `unknown` row in "Credential issues to check". "Open issue form" shows locked inputs and "Try again". The retry has the same key and shows the secret. |
| `selector` | Lost response. Select another service, then the first again. | Same as `leave`, reached through the selector |
| `revokebtn` | Lost response. Click Revoke on another credential's row, then Cancel. | Held form restored, focus on that row, same key on the retry |
| `backnarrow` | At 1000 px: lost response, then Back. | Reminder row present. Open the form again and retry with the same key. |
| `inflightleave` | Delay the issue response. Switch view while it is in flight. Also hold a services read that started before the release (N2), then release the issue response, then release the held read. | The `issued` row survives the stale read (N2 assertion). "Show secret" shows the secret. The unstored mark is added. |
| `reload` | Lost response. Dispatch a cancelable `beforeunload` (`defaultPrevented` must be true). Reload with a dialog handler that accepts and records the type `beforeunload`. | After the reload the row shows "Secret not collected yet · pickup open, estimated to end …". Sign out and sign in. The row shows "Secret never collected" and is listed in the region. Revoke from the region succeeds; focus reaches `#svc-cred-title`. |
| `remindercancel` | Two services. Leave a secret unacknowledged on service A, select service B, click Revoke in the unstored reminder, then Cancel, at 1440 and 390 px. | Focus on `[data-unstored-revoke="<id>"]` |
| `revealcancel` | On the Reveal, choose "I didn't store it: revoke", then Cancel. | Focus on `[data-credential="<id>"]` (F7) |
| `typesending` (F1) | Delay the issue response. While it is pending, type into days and reason (they must be read-only and unchanged) and try Discard (disabled). Then lose the response and retry. | Values unchanged, the same key on the retry, the secret shown. Repeated for `CreateForm` with a delayed `service.create`. |
| `idemconflict` (F2) | Lose the response. On the retry, route-fulfil `409 idempotency_conflict` (synthetic). | Entry `lapsed`, no "Try again", the lapsed text, Discard only |
| `replaypartial` (F2) | Lose the response. On the retry, route-fulfil `500 internal_error` with an `operation_id` (synthetic). | Entry `lapsed`, never removed |
| `window` (F2) | Route-modify the services read so `limits.replay_seconds` is 10 (synthetic). Lose the response, wait 10 s, then select Try again. | No second POST is sent; the entry becomes `lapsed` |
| `issueddrop` (F4, N6) | Delay the issue response and leave while it is in flight. Release the response (`issued` row). From a second page in the same browser context (the same session), revoke that credential on the M04 credentials view. Then reload the list on the first page with its Reload button. | The `issued` row disappears; the credential is listed as revoked; no `beforeunload` guard remains (a dispatched cancelable event is not prevented). The first session stays live throughout. |
| `signoutheld` (N5) | (a) Lost response, so an `unknown` entry is held: sign out. (b) Issue and stay on the Reveal with the secret shown: sign out. | In both cases no dialog of any kind appears, the sign-in page loads, and after signing in again the credential from (a), if issued, shows "Secret never collected". |
| `create` | Lost `service.create` response, then Try again. | Same key, "created" (not "name taken"), inputs locked while held |
| `createleave` | Lost create response, switch view, return, and create the same name. | Field error "… already exists and you created it …" |
| `copy` | Render both regions, the locked forms and the row sub-lines in EN, KO and AR at 1440 and 390 px. | Screenshots for the `ui-designer` and TW reviews. No horizontal scroll. Computed borders neutral. |

Pre-fix evidence: the implementer runs the new scenarios against the `main` bundle first and records the failures, then runs them on the fix. Required: 20 consecutive passing runs of the full tool after the fix.

## Regressions

- `tsc --noEmit`, `npm test` and `npm run build`.
- `qa269` (M04; expected 49/0, three runs). Its `[data-credential]` count and `.live-unstored` checks must keep passing. That is the reason for the separate classes and attributes.
- `qa293on307` groups B, C, D and A, because every command screen shares the reauthentication flow.
- The M01, M02 and M03 QA, only if `live.css` changes.
- The runners `qa269` and `qa293on307` are uncommitted. They are in the session scratchpad used for the HYB-316 verification. The implementer records the exact runner file used.
- The new `beforeunload` guard can block Playwright `page.goto` in older runners while an attempt is held. Any such hang is recorded as a runner interaction, not hidden.

# Owner questions (with the default applied until answered)

1. **Q1. Revoke automatically when a pickup closes unread?**
   - Default: no. The row and the region show "Secret never collected" with Revoke as the next step.
2. **Q2. Let the issuing session collect an open pickup again after a reload** (option E, or option D)?
   - Default: no. After a reload, the path is revoke and issue a new credential.
3. **Q3. Show "collected" with a time in the list?**
   - Default: no time is kept or shown. `collected` adds no sub-line, because it cannot mean "stored".
4. **Q4. A rotation command.** The M04 action contract lists "자격 회전 요청", but the API has none.
   - Default: the next step is revoke, then issue. Rotation is ticketed as [HYB-334](https://linear.app/hybrid-billing/issue/HYB-334).
5. **Q5. KO spacing:** "비밀 값" (current M04 copy) or "비밀값" (glossary secret-pickup row).
   - Default: keep M04's "비밀 값" and flag it to TW.
6. **Q6. Locking the inputs of a held issue or create**, which replaces HB-269's "editing starts a new attempt".
   - Default: lock, with "Discard and start over".

# Residuals

- **Ambiguous `denied`.** A held retry refused with `permission_denied` (lost permission or a deleted target) cannot tell whether the first request committed, because `Resolve` and the permission check run before the replay lookup. The entry becomes `lapsed`, and the server state is the trace.
- **`collected` is not "stored".** It cannot be distinguished from a lost collect response. The unstored marks (page memory) still cover this within the page, but not after a reload.
- **Held create.** It is kept across breakpoint changes, but not lifted out of the form across a deliberate leave. Its durable trace is the account itself in the list, plus the "you created it" error text.
- **Restart.** Credentials, pickups and the collected map are in memory and reset on restart (as in all pre-launch phases). If credentials become persistent later, this map must persist with them.
- **Tested environment.** The browser tool is planned to run only on mac_mini with chrome-headless-shell 1243. A fresh data directory is needed per run.
- **Original request still in transit.** The original request can still be on its way when the console decides. For example, it is delayed in a proxy or queued, so the server has not reserved its key when the retry arrives.
  - The retry then runs as a first execution. If it is refused definitively (for example `revision_conflict`), the console removes the entry as not issued.
  - The delayed original can still arrive later and commit a credential, or replay the retry's stored answer if the bodies match.
  - The same applies after a `lapsed` decision or a Discard.
  - The console cannot observe this. The truthful trace is the server state: the credential appears with `secret_pickup` `open`, then `never_collected`, in the row and the never-collected region.
  - The removal copy therefore states only what the retry's answer says, never "no credential was issued".
- **Synthetic answers.** The `idemconflict`, `replaypartial` and `window` scenarios rely on route-fulfilled or route-modified answers, not on a real one-hour wait or a real stored failure. The real paths are covered by the Go and unit tests.
- **Copy review.** The KO and AR copy is an AI draft pending native-language review. Screen acceptance (FIX) is the owner's alone.

# Review history

## Review round 1: independent security design review (CHANGES REQUIRED)

Reviewer: the separate read-only security design reviewer, relayed by root. Root made decisions F2 and F5. The plan text above has been revised for every finding. A re-review has not run yet.

| ID | Severity | Finding | Change in this plan |
| --- | --- | --- | --- |
| F1 | High | Typing during a slow request renewed the attempt, so a lost response was retried with a new key. | Inputs read-only with `aria-describedby` to the held lead, and Discard disabled, while `sending`, in `IssueForm` and `CreateForm`. The store and `attemptState` refuse `renew` while sending. Unit test, plus browser scenario `typesending`. |
| F2 | Medium-High (root decision) | The failure handling was not exhaustive, and some answers wrongly meant "not issued". | Phase table over every `AdminFailure` for first send and retry: `idempotency_conflict` → `lapsed`; `partial` on retry → `lapsed`, never removed; `limited` and `unauthenticated` keep the phase; `revision_conflict` near or after the window → `lapsed`. `firstSentMono` and `firstSentWall` are recorded, and no retry is sent once `window − margin` has passed. The window comes from the new `limits.replay_seconds`. One test per row, plus scenarios `idemconflict`, `replaypartial` and `window`. |
| F3 | Medium | `open` was claimed for a pickup whose issuing session could no longer collect it. | `open` only when the issuing session is live (`s.sessions.get` and a side-effect-free admission check); otherwise `never_collected`. Go tests 4a cover idle expiry, absolute expiry, eviction and prune, and check that the idle clock is unchanged. |
| F4 | Medium | An `issued` entry could outlive its pickup and keep the guard up. | Each read drops an `issued` entry whose row is not active with `open`; a successful Revoke from the attempt row removes it; the guard is re-evaluated. Unit tests, plus scenario `issueddrop`. |
| F5 | Medium (root decision) | The guard would prompt on a deliberate sign-out or on the session-ended navigation. | `Service.tsx` added to the allowed files. `release(user)` runs before `signOutIn` and `onSignedOut`. The `reload` scenario handles its dialog deterministically. |
| F6 | Medium | Continuations wrote to component state, and an `issued` entry could be consumed implicitly. | Continuations write only to the store; the parent derives the display from the store. Submit is refused while the service's entry is `sending` or `issued`. `issued` is consumed only by Show secret (including the single automatic case) or Revoke. |
| F7 | Low | Revoke sources shared attributes, and a revoke opened from the Reveal had no invoker. | Distinct `data-unstored-revoke`, `data-never-revoke` and `data-attempt-revoke`. A revoke opened from the Reveal uses `from = [data-credential="id"]`. The `returnFocus` test covers all five sources; scenario `revealcancel`. |
| F8 | Low | The browser scenarios could pass by reloading, carried state between scenarios, and handled dialogs nondeterministically. | In-app navigation with a no-reload marker, a fresh page per scenario, a deterministic dialog policy, and F1 and F2 scenarios with labelled synthetic answers. |
| F9 | Low | Bulk cleanup and role-period end were not tested; the property test lacked session ends. | Go tests for `bulkcleanup.go` and `periods.go:221`. The property test includes sign-out, M05 session end and eviction. |
| F10 | Low | Wrong `Live.tsx` line references; the derivation location was not stated (integrations); Q4 had no ticket. | Line references corrected (`55-99`, `544-548`, `109`). The derivation lives in `listServices` via `pickupStates`, with `credentialRows` unchanged and test 4b for `integrations.go:355`. Q4 cites HYB-334. |

## Review round 2: independent security design review (CHANGES REQUIRED: 3 Medium, 4 Low)

Reviewer: the same separate read-only reviewer, relayed by root. Root made decisions N4 and N5. The plan text above has been revised for every finding, and the in-transit-original residual is recorded.

| ID | Severity | Finding | Change in this plan |
| --- | --- | --- | --- |
| N1 | Medium | The monotonic clock alone can stop (system sleep, a throttled tab) while server time moves on, so the replay window could be understated. | `elapsed = max(performance.now() delta, Date.now() delta)`. Unit tests: the wall clock advances while the monotonic clock does not; the wall clock goes backwards. |
| N2 | Medium | A services read that started before the outcome arrived could drop a fresh `issued` entry. | A read sequence: an entry records `issuedAfter`, and only reads started after it may drop it. Unit test with a stale read; the `inflightleave` scenario asserts that the `issued` row survives a held stale read. |
| N3 | Medium | Refusing `renew` while sending also blocked the failure path's own renew, so a definitive failure kept the old key. | The failure path leaves `sending` first and then renews; only the user-triggered renew is refused while sending. The pure `settleFailure` is tested for `IssueForm`, `CreateForm`, `StateStep` and `RevokeStep`. |
| N4 | Low (root decision) | `open` did not account for permission, and `secret_pickup_until` read as a promise. | The `open` derivation adds a side-effect-free `AdministerInstallation` gate decision for the session's answer. `secret_pickup_until = min(expires, absolute end, idle end)`. The copy says "estimated to end". Go tests for losing the grant and for each minimum; property test 8 unchanged. |
| N5 | Low (root decision) | A deliberate sign-out could still be blocked by the Reveal's guard or the App dirty guard. | A shared `unload-intent` flag is set by `Service.tsx` before both navigations, and all three handlers honour it. Unit test, plus the `signoutheld` scenario (an `unknown` entry held; a secret revealed). |
| N6 | Low | `issueddrop` depended on eviction or idle timing. | It now revokes the credential from a second page in the same session, with the first session live. |
| N7 | Low | A stale line 30; a second `reauth` after the resend was undefined; "visible" for the automatic Show secret was vague. | Line 30 updated. A second `reauth` is handled like `reauth_cancelled` (removed after a first send, `unknown` kept after a retry). "Visible" is defined as the credentials view, mode `form`, `current.id` equal to the entry's service, `panelShown`, and `document.visibilityState === "visible"`. Unit tests. |

## Review round 3: APPROVE

The reviewer approved the design with four Lows folded in:

| ID | Severity | Finding | Change in this plan |
| --- | --- | --- | --- |
| L1 | Low | The deliberate-leave flag could stay set after a back-forward cache restore or a new sign-in. | Cleared on `pageshow` with `persisted` and when a new session is set; unit test. |
| L2 | Low | Nothing stopped other code from calling `leaveDeliberately`. | Source test: only the two `Service.tsx` paths call it. |
| L3 | Low | Leftover `firstSent` references after N1. | `firstSentMono`, `firstSentWall` and `elapsed` used throughout. |
| L4 | Low | The pending approval line. | Replaced by this record. |

**Review round 3: APPROVE.** This is a design approval. The change review of the actual diff, the `ui-designer` visual spec (`docs/features/HYB-333/visual-spec.md`, commissioned by root) and the TW copy review are still to come.
