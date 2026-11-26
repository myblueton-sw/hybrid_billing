---
type: ImplementationPlan
title: HYB-323 M01 select everyone matching a filter for bulk revocation (HYB-294 Part B)
description: Part B of the approved HYB-294 plan, rebased on the merged Part A. A new staged command resolves the M01 filter on the server into a caller-owned filter draft on the HYB-299 transport; prepare re-evaluates the stored filter and excludes people who no longer match; a bounded count read precedes selection; the manifest digest covers the filter for filter drafts only; two new installation settings; the anchor dead-end copy; an incremental latest-outcome index; HTTP scale measurements at 5,000, 20,000 and 100,000. Approved after design review round 3; D-7 amended by the visual addendum's root decisions.
status: stable
tags: [HYB-323, HYB-294, M01, identity, bulk-revocation, filter-wide-selection, IAM-07, UI-04, HB-03, HB-34, HB-42, HB-41, HYB-299]
sources:
  - resource: ../HYB-294/plan.md
  - resource: ../HYB-294/visual-spec.md
  - resource: ../HYB-294/verification.md
  - resource: ../HYB-299/plan.md
  - resource: ../HYB-299/verification.md
  - resource: ../HB-257/plan.md
  - resource: ../../ui/screens/M01/spec.md
  - resource: ../../planning/v1.16/retention-access.md
  - resource: ../../planning/v1.16/ui-design.md
  - resource: ../../planning/v1.16/requirements.md
  - resource: ../../planning/v1.16/acceptance-trace.json
---

# HYB-323 M01 select everyone matching a filter (HYB-294 Part B)

## 0. Status, scope and sequence

- **Ticket.** Linear HYB-323 (team HYB, state In Progress, label `machine:mac_mini`), read through `scripts/linear_workspace.py` on 2026-11-18 because the Linear MCP answered 401. The ticket carries one comment: the HYB-294 change-review residual N5 (incremental latest-outcome index). This plan covers both.
- **Branch and tree.** `feat/HYB-323-m01-select-matching` in `/private/tmp/hb323`, on main `3bdb750` (includes HYB-294 Part A, PR #227). Nothing is committed by the implementer.
- **Baseline.** The approved HYB-294 plan (`../HYB-294/plan.md`, status stable): D7, D8, the Part B sections (`:568-680`), the review amendments (H2, M3, M4, M6, L3, L6, N1) and Q2–Q5, Q8. Where this plan says "as HYB-294", that text is binding; this plan adds only what the merged Part A code forces or leaves open (section 11).
- **Sequence.** this plan → independent design review (architect/security for the command, permission and digest; DBA-style for the index and scale) → ui-designer Part B visual addendum (`docs/features/HYB-323/visual-spec.md`, filling the room HYB-294 `visual-spec.md` §9 left) → TW pass on the copy table → implementation → change review, render review, TW review → browser QA → `verification.md` → root PR.
- **Ownership.** Root: PM, planner, integration, PR. Implementer (this agent, after review): every file in section 9. Read-only reviewers as above, plus QA review of the browser record. One writer per file.

## 1. Planning cross-check (requirement text quoted from the source)

| ID / source | Text (verbatim) | Part B obligation | Where in this plan |
| --- | --- | --- | --- |
| M01 spec `docs/ui/screens/M01/spec.md:104` | "목록은 고정 snapshot/revision과 cursor로 조회하고 전체 자료를 브라우저 메모리로 읽지 않는다. 현재 페이지와 필터 전체 선택을 분리하며 전체 선택은 서버 target manifest·건수·허용 scope를 고정한다. 대상 변경은 새 snapshot으로 표시하고 승인된 범위를 조용히 갱신하지 않는다." | Separate page and filter-wide actions; the server fixes targets, count and scope; drift is shown, never silently applied | D-1, D-2, D-3, D-7 |
| M01 spec `:75` | "선택 범위 회수 \| 대량 회수 권한 \| 고정 manifest·역할 상한 \| 보안 효력 확정 후 청크 cleanup \| 대상/제외/효력 범위 확인" | Bulk right; fixed manifest; chunked cleanup after the deny; confirm targets, exclusions and effect | D-1, D-3, D-8 (cleanup limit), D-7 preview |
| M01 spec `:112` | "대량 파괴적 선택은 대상 요약을 본 후 확인 단계로 이동한다." | The fixed summary precedes the confirm step | D-7 |
| M01 spec `:81` | "탭 전환은 필터와 선택 범위를 유지하고 미저장 초안은 이탈 영향을 안내한다." | A fixed list survives tab changes and reloads | D-7 (recovery from the server) |
| M01 spec `:63` | "서버 검색과 안정 정렬을 사용한다. 표시명 변경에도 canonical ID로 선택을 유지한다." | The fixed set holds user IDs | D-1 |
| M01 spec `:25` | "해당 화면의 … scope를 서버에서 검증하고 상세·검색·합계·다운로드에 동일하게 적용한다." | Count and resolution use the read's scope check | D-1, D-4 |
| M01 trace `:118`, `:120` | "HB: HB-03, HB-34, HB-42。" / "인증·접근 추가 인수: IAM-01~IAM-08 중 이 화면의 생성/신뢰/역할/회수/복구/대량 경로를 연결한다." | Bulk path linked to IAM-07 | sections 1, 8 |
| IAM-07 `retention-access.md:116` (trace `acceptance-trace.json:1947`, HB-42) | "\| IAM-07 \| 10만 사용자 조회·초대·동기화·회수의 격리·재개·차단 revision 검증 \|" | Revocation of 100,000: isolation, deny before ledger, counts invariant, measured | D-8, section 7 |
| `retention-access.md:65` | "10만 사용자도 서버 검색·cursor·고정 선택 manifest로 관리한다." | Fixed selection manifest at 100,000 | D-1, D-8 |
| UI-04 `ui-design.md:547` (trace `:2064`, HB-41) | "\| UI-04 대량 탐색 \| 10만 사용자·10만 이상 목록과 월 누적 상세에서 필터/선택/페이지/합계 기준 일치, 브라우저 전량 적재 없음 \|" | Filter, selection and total bases agree; no client ID array | D-1, D-4, D-7 |
| `ui-design.md:420` | "현재 페이지 합계와 전체 선택 범위 합계를 분리. … 집계가 아직 준비 안 됐으면 0 대신 대기" | Count is "at least n" or exact, never a fake total; waiting state while counting | D-4, D-7 |
| `ui-design.md:421` | "현재 페이지와 필터 전체 선택을 다른 동작으로 제공. 전체 선택은 서버 manifest에 대상·제외·기준개정 고정. 클라이언트가 10만 ID 배열을 들고 다니지 않음" | Separate action; targets, exclusions and base revision in the manifest; no client ID array | D-1, D-3; Q4 (chosen exclusions) stays open |
| `ui-design.md:422` | "필터/업무 공간/기간 변경은 기존 선택을 유지할지 명시적으로 구분하며 다른 범위로 몰래 확대하지 않음. 실행 직전 개정·권한 재검사" | Filter change after fixing keeps the fixed list and says so; re-check before execute | D-2, D-7 |
| `ui-design.md:423` | "샘플 50행 통과를 전체 10만 행 통과로 표시하지 않음" | Preview pages are labelled as pages of the full list | D-7 (existing targets paging) |
| W03 `ui-design.md:278-279` | "[필터 결과 100,000개 전체 선택]" / "전체 선택 후: 대상 100,000개 고정 · 생성 10:00 [선택 해제]" | Count on the action; fixed count, fixed time, release action | D-7 |
| `retention-access.md:98`, RET-05 `:122` | "삭제·회수 확정 응답 전 독립 결정 원장의 내구 commit을 보장한다." | Phase B stays before the answer (HYB-294 D8) | D-8 |
| `retention-access.md:76` | "인증 비밀·불필요 프로필 제거" | Search text masked for non-preparers; no login in logs | D-6 |
| HB-03 `requirements.md:53` | "## HB-03 행위 권한과 위임" — "조회·원가 열람·정책 변경·승인·발행·다운로드·전달을 분리한다." | Bulk right plus `administer_installation`; owner-only drafts | D-5 |
| HB-42 `requirements.md:749-753` | "…권한 부여·변경·정지·탈퇴·회수까지 관리한다." | Revocation lifecycle | whole plan |
| HB-34 (M01 trace) | async work | Cleanup job fits 100,000 targets | D-8 |
| HB-257 phase row `HB-257/plan.md:166` | "HB-263/264 (M01) … M01–M05" | M01 is in the phase 1–5 build | — |
| HYB-299 R1 `HYB-299/plan.md:1251` | "Filter-wide selection and selections above 5,000 people wait for HYB-294, which must reuse the staging transport." | Reuse the transport, no new command route | D-1 |
| HYB-294 verification N5 (`../HYB-294/verification.md:96`) and the HYB-323 Linear comment | "reindex walks every kept executed item on each prune" | Incremental index, measured | D-9 |

Expected status after HYB-323: M01 `:104`, `:75` and W03 covered; IAM-07 revocation at 100,000 measured (invitation and sync remain outside, as HYB-294); UI-04 covered for M01; `ui-design.md:421` **partial**: targets and base revision fixed, system exclusions fixed, user-chosen exclusions open (Q4, HYB-324). Since [HYB-324](../HYB-324/implementation.md) the exclusions chosen by the person are built, so this line of the planning document is covered.

## 2. Facts from the merged code (`3bdb750`)

- `bulkrevoke.Registry.Add` creates drafts with `Selection: Explicit` only (`modules/bulkrevoke/registry.go:121`). `Prepare` hard-codes `Explicit` in the digest (`manifest.go:87`). `Digest` length-prefixes every field with an 8-byte big-endian length (`manifest.go:14-21`).
- Replay rebuilds jobs with `Selection: bulkrevoke.Explicit` (`apps/api/server/identityledger.go:267`), although the ledger header records no selection. A replayed filter job would therefore claim "explicit".
- `prepareBulk` refuses with `prerequisite_failed` both when the anchors would be stranded (`bulk.go:33-35`) and, through `bulkError`, when nothing is left to revoke (`ErrEmpty`, `bulkstage.go:66`). Refusal fields are copied only for `invalid_request` (HYB-300 review H2), so the console cannot tell these two apart. The console shows one "blocked" state for any refusal (`BulkPrepare.tsx:229`).
- The latest-outcome index is rebuilt from every kept executed item on every `prune` (`latestindex.go:10-33`), which `Execute`, `Discard` and `Restore` call (`registry.go:175`, `:240`, `manifest.go:146`). It orders jobs by **number**, not by execution time; the incremental index must keep that rule.
- Prepare impact costs per target: `sessions.byLogin` scans every session (`session.go:111-121`), `pickupsOf` scans every pickup (`bulkimpact.go:181-193`), and each proposal store scans its entries (`bulkimpact.go:24-117`). So prepare is O(targets × (sessions + pickups + proposals)).
- Cleanup: chunks of 100 items (`bulkcleanup.go:16`) in one job whose task limit is 100 (`bulkcleanup.go:22`). **A job with more than 10,000 deny-confirmed items cannot be created**: `jobs.Create` refuses it (`modules/jobs/registry.go:160`) and the job stays `cleanup_not_started`. The explicit maximum (5,000) never reaches this; the filter maximum (up to 100,000) does.
- `Registry.List` copies every view including `Items`, `Targets` and `Exclusions` before dropping them (`registry.go:202-221`), and `listBulk` then calls `Get` (another full copy) per row (`bulkread.go:225-226`).
- Settings: `withDefaults` applies defaults per group only when every number in the group is zero (`settings.go:107-118`). `user_filter_scan` sits in its own `userSettings` group and is not shown on any screen (HYB-294 deviation D6). Flags are registered from all three lists (`cmd/hb-api/main.go:95`).
- Commands: undeclared fields are refused with `invalid_request` (`modules/commands/execute.go:84-93`); every mutating command passes HYB-293 step-up (`stepup.go:41`); the bulk definitions require a rationale (`bulkstage.go:101`).
- The users read evaluates filters with `userMatcher.matches(row, member, now)` (`userfilter.go:187-202`) on rows built by `s.row` (`users.go:296-313`), over two segments (`dir.order`, then `dir.departed`), fetched through `Workspace.People` in chunks of at most 256 (`users.go:212-243`).

## 3. Decisions

### D-1. `bulk_revocation.add_matching` (HYB-294 D7, unchanged in substance)

- **Definition.** Target `bulk-revocations`, permission `revoke_access_in_bulk` (installation), mutating, rationale required, plus `administer_installation` in `bulkCommand.Apply` (`bulkstage.go:53-61`). Defined through the existing `define` helper with `stage: true`.
- **Fields.** `q` (text), `status`, `membership`, `role`, `period`, `signin`, `invitation`, `holder` (enum, values from `userFilterValues`, the single source), `method` (text, always refused). `after`, `limit` and `workspace` are not declared, so the command registry refuses them with `invalid_request`.
- **Validation.** The handler turns the given fields into `url.Values` and calls the read's parser, so the read and the command share one parser. Today `parseUsersQuery` returns only a code (`userfilter.go:93-125`), so the command could not name the field (review F6). It gains a second result, the offending parameter name: `parseUsersQuery(values) (usersQuery, commands.Code, string)`. The users read ignores the name and keeps answering `invalid_request` with no fields, as merged (the 422 in `x-reads` stays without `x-fields` except for `after`). The command answers `commands.Fail(commands.InvalidRequest, name)` for an empty value, a bad `q` or an enum value the registry did not already refuse, and `commands.Fail(commands.UnsupportedCapability)` for `method`. Tests pin the field name for each key, and that the read's 422 body is byte-identical to `3bdb750` for the same bad inputs.
- **Resolution, under the domain write lock** (commands run under it, `commands.go:209`):
  1. Walk **both segments** (login order, then departed) through `People` chunks of 256, with **no scan budget**, evaluating `userMatcher.matches` at one `now` (the command's server time). Stop as soon as the `max_matching + 1`-th match is found; exactly `max_matching` matches is accepted, `max_matching + 1` is refused (F6).
  2. Nobody matches → `prerequisite_failed` (module `ErrEmpty`). More than `-bulk-revocation-max-matching` → `prerequisite_failed` (`ErrTooMany`). The count is not returned on refusal.
  3. `bulkrevoke.AddMatching` creates a **new** draft owned by the caller with `Selection = "filter"`, the filter terms, `Matched`, `FixedAt` and the base revision (`FixedRevision = s.Revision()` before the answer's `s.Next()`). The open-draft and pending limits apply (`ErrOpenDraft`, `ErrPending` → `prerequisite_failed`). There is no `draft` field: a filter list is always created whole.
  4. Answer `completed`, state `draft`, fields `draft`, `count`, `matched`, `fixed_at`, `fixed_revision` and one field per applied filter key (`filter.status`, …; `filter.q` carries the search text, which only the caller sees).
- The caller can match their own filter; they are kept in the set and excluded at prepare with reason `self`, as HYB-294 Q5 states.
- `add_targets` with `draft` naming a filter draft → module `ErrState` → `invalid_stage`.
- Trust hold: one row `"bulk_revocation.add_matching": {holdAllow, "stages a reduction and changes no access"}` (`trusthold.go:43` pattern).

**One walker for three callers.** A new `s.walkPeople(actor, login, departed []string, now, visit)` in `userfilter.go` (or a small new `userwalk.go`) owns the chunked `People` loop and row building. The list read (`usersScan.segment`, with its budget and limit), the count read (D-4) and `add_matching` (no budget) all use it, so "a person matches the selection only if they would match in the list" holds by construction. The Part A property test and filter tests are the regression guard for the refactor.

### D-2. Prepare re-checks the filter (HYB-294 H2)

- `prepareBulk` reads the draft view (new module accessor returning the draft's `View` and its users). For a filter draft it rebuilds a `userMatcher` from the stored terms and evaluates each member with `s.row(u, mp, now)` and `matches(row, mp, now)` from the same `everyone()` roster, at the prepare's `now`, where `mp` is `&m` when `m, ok := r.members[id]` is present and **`nil` when the ID has no (non-removed) membership** (review F1). That is the same nil `People` returns for a non-member (`modules/identity/people.go`), so `membership=none` matches, and `role` and `period` never match, exactly as in the list.
- **Equivalence test (F1).** For every generated person, including non-members, removed memberships, suspended memberships, ended periods and departed people, the row and the match result built from the roster equal those built from `People`, for every filter key and value (`membership` all four values, `role`, `period`, and the rest).
- **Classification order** for a filter draft (clarifies HYB-294, which listed only "re-check first"): `self` → `not_found` → `no_longer_matches` → `no_membership` → `already_revoked` → target. The `not_found` predicate is **unchanged** from `classify` (`bulkimpact.go:155`): unknown, not human, departed, **or `effect == suspend_user` and no membership** (review F7). It runs before the re-check; a filter list only inserts `no_longer_matches` between `not_found` and `no_membership`. Explicit lists keep the existing order with no re-check. `self` stays first so the caller's own exclusion is always reported as `self`; `not_found` before the re-check keeps a departed person's reason truthful (a departed row has no login, so most filters would otherwise report "no longer matches").
- New exclusion reason `ExcludedNoLongerMatches = "no_longer_matches"`. It is counted, listed by the owner-only exclusions read, and shown in the preview.
- Nobody is ever added at prepare; the draft never widens.
- **Nothing left to revoke on a filter list** answers `invalid_stage`, not `prerequisite_failed` (deviation X-3). Explicit lists keep `prerequisite_failed` for `ErrEmpty`, unchanged. Reason: on a filter list `prerequisite_failed` then means exactly "this list would leave no anchor administrator", so the anchor dead-end copy (D-7) is truthful; `invalid_stage` shares its remedy (discard and select again) with an expired or discarded draft, which already answers `invalid_stage` (`bulk.go:18`).
- **Execute does not re-run the filter** (review F8). Between prepare and execute, the HYB-299 hold rule applies unchanged: a target whose user epoch, user state, membership version, membership state or role changed since preview is held `changed_since_preview` (`changedSince`, `bulk.go:44-49`). Filter terms that can change **without** touching those fields are not re-checked at execute: time-based terms (`period` ends soon or ends, `membership` ended), `holder`, `signin`, `invitation` and `q` (an email change). Such a person is still revoked, because the decision was confirmed on the preview's fixed list. The preview and the copy say "at {time}" for every filter fact and never say or imply "still matches at execution"; the confirm dialog is unchanged.
- Anchor rule unchanged: `strandsAnchors` over the roster (`periods.go:49-68`: administrator, effective, active user, open-ended period, live installation grant). `mayChangeRole` unchanged (`permission_denied`).

### D-3. Manifest and digest (HYB-294 D7, L6)

- `types.go`: `Selection` constants `Explicit` (existing), `Filter = "filter"`, `NotRecorded = "not_recorded"`; `type FilterTerm struct{ Key, Value string }`; View gains `Filter []FilterTerm`, `Matched int`, `FixedAt time.Time`, `FixedRevision string`; `Limits.MaxMatching` (required > 0).
- `AddMatching` validates: workspace and owner IDs; 1 ≤ `len(users)` ≤ `MaxMatching` (else `ErrEmpty`/`ErrTooMany`); unique valid IDs; terms with distinct keys in strictly ascending key order, non-empty keys and values, each value at most 64 bytes; non-zero `FixedAt`; non-empty `FixedRevision`. Anything else is `ErrInvalid`.
- **Canonical filter.** `CanonicalFilter(terms)` returns `len(key):key len(value):value` per term in key order (decimal lengths), absent keys omitted, exactly the HYB-294 L6 encoding. Property test: no two distinct term lists encode equally.
- **Digest.** `Prepare` passes the draft's own selection. For `explicit` the field sequence is byte-for-byte today's (pinned vector test, plus a test-local copy of the current `Digest` as an oracle over random inputs). For `filter`, after `selection` the digest adds the fields `CanonicalFilter`, `Matched`, `stamp(FixedAt)` and `FixedRevision`. Including the base revision is a clarification (C-1): `ui-design.md:421` fixes "기준개정" in the manifest.
- **Equal-manifest reuse** (`manifest.go:88-92`) compares digests; two fixes of one filter differ by `FixedAt`, so they never collapse (intended, HYB-294 D7).
- Ledger header and records unchanged; `RequestDigest` covers the filter. **Replay** sets `Selection = NotRecorded` instead of `Explicit` for every replayed job (deviation X-1): the header never recorded the basis, for explicit lists either.

### D-4. Bounded count read `GET /api/v1/identity/users/count` (HYB-294 M4, N1)

- New `userscount.go`, route in `boundary.go`. Same admission as the list (`s.admitted`: session, CSRF, `administer_installation`), the same `answer.Workspace() == s.bootWorkspace` check, `s.domain.RLock`.
- Parameters: exactly the list's filters through `parseUsersQuery`; `after` and `limit` are refused (`invalid_request`), `method` is `unsupported_capability`, `workspace` is `invalid_request`.
- Walk with the shared walker, stopping at the first of: end of list; `user_count_scan` people examined; `max_matching + 1` matches.
- Answer `{count, complete, over_limit, limit, scan_budget, list_version, server_time}` (review F6, exact rules):
  - `over_limit` is true if and only if `max_matching + 1` matches were found. Then `count = max_matching + 1`, which is a lower bound, and the walk stops at that person.
  - `complete` is true if and only if the walk examined the last person of the departed segment (or both segments were empty). Both can be true at once: when the `max_matching + 1`-th match is the last person, `count = max_matching + 1`, `complete = true`, `over_limit = true`; the console shows only the "too many" copy whenever `over_limit` is true.
  - Exactly `max_matching` matches with `complete = true` is `over_limit = false` and selectable.
  - Neither flag: `count` matches among the first `user_count_scan` people, a lower bound ("at least {n}").
  - The scan budget and the match stop are checked after each examined person, the match stop first, so a tie at the same person reports `over_limit`. Recorded in `openapi.go` under `x-reads["/identity/users/count"]` with its refusals (same table as `/identity/users`, without 409).
- No count appears in the list read (HYB-294 N1 unchanged).

### D-5. Tenant and permission checks

- Workspace only from `answer.Workspace()` / `ctx.Workspace`, asserted equal to `s.bootWorkspace` (read: `users.go` L1 check; command: `bulkCommand.Resolve`). Client-sent `workspace` is refused in every form.
- Count read: `administer_installation`. `add_matching`, prepare, execute, discard: `revoke_access_in_bulk` plus `administer_installation`; execute also requires a human caller (unchanged). Step-up applies to `add_matching` as a mutating command (`stepup.go:41`), no change.
- Drafts are owner-only: a non-owner gets `NotFound` → `permission_denied`, the same as an unknown draft (HYB-299 N4).
- **Who reads frozen lists** (review F4). The bulk reads (`listBulk`, `readBulk`, `readBulkItems`) admit through `bulkReader` → `s.admitted`, i.e. **`administer_installation` only** (`bulkread.go:190-196`); the bulk right is not required to read. So every holder of `administer_installation` reads prepared and executed lists, their rows, detail and items (with target logins), as since HYB-299. This plan does not change who reads; it adds that the structured filter values are shown to the preparer only (D-6). The targets and exclusions pages stay owner-only (`manifest.go` `Targets`/`Exclusions` check `Owner`).
- Negative tests: another workspace's people (direct registry call) never match; a member session, no session, missing CSRF, no bulk right, no `administer_installation`; `workspace` in three forms; a non-owner's prepare, discard and exclusions read of a filter draft. **Masking tests (F4)** for two non-preparers, one holding the bulk right and one holding `administer_installation` without it: the jobs list rows, the job detail and the items read show `filter.q` empty with `search_masked: true`, and the other filter keys; the preparer sees the text.

### D-6. Reads

- `bulkRow` gains `selection` (`explicit`, `filter`, `not_recorded`), `matched`, `fixed_at`, `fixed_revision`, `filter` (object with the `userFilter` keys) and `search_masked` (bool). For a caller who is not the preparer, `filter.q` is empty and `search_masked` is true when a search was applied (HYB-294 L3, Q8).
- **Masking is a display rule, not secrecy** (review F5). The search text still exists elsewhere: the manifest digest covers the canonical filter including `q` (a short `q` can be confirmed by recomputing the digest from a guess, given the other digest inputs, which are not all exposed but must not be relied on as a secret); the common command journal stores `add_matching`'s field values, including `q`, as it stores every command's fields (`modules/commands` `Fields: copyFields(values)`, `execute.go:379`); and the console link may carry it (HYB-294 Q8). Q8 and residual R-B2 say so.
- `readBulk` adds the key `no_longer_matches` to `exclusions` **for filter lists only**, so explicit detail answers stay unchanged.
- `Registry.List` builds summary views without copying `Items`, `Targets` and `Exclusions`; `listBulk` keeps one full `Get` per shown row (needed by `bulkRow`'s ledger fields). Pure efficiency change, guarded by the existing list tests and an equality test against the previous behaviour.
- Logs: `bulk_revocation_matched` with the draft ID, matched count and the filter **keys** (never values, never logins), following `HYB-299/plan.md:798`.

### D-7. Console

Detailed layout comes from the ui-designer addendum; the behaviour is fixed here.

- **Action placement.** With `can_bulk_revoke`, on the Users and Membership tabs (not Invitations, which has no checkbox column), the selection bar shows "Select everyone who matches these filters" inside `div.m01-selactions` directly after the count (HYB-294 visual spec §9). With no filter applied it reads "Select everyone in the list" (HYB-294 Q5 label; new [TW] string).
- **Count step.** Activating the action expands an inline region in the bar and calls the count read with the current filters: "Counting people who match…" while waiting (no number shown, `ui-design.md:420`); then "Select all {n} who match" (complete) or "Select all who match (at least {n})" (lower bound); `over_limit` shows the "too many" copy with no confirm button; a complete zero shows "Nobody matches these filters now."; a failure shows the shared failure text with Retry. The count runs only on activation, never on a filter change.
- **Replace confirmation.** With explicit picks, the confirm button is preceded by "This replaces your {n} selected people with everyone who matches." and needs a second activation.
- **Fixing.** The console sends `add_matching` with the current filters, a fresh expected revision (`reload()`, as discard does) and a system rationale "Selected everyone who matched the filters on the revocation screen" in the user's language (the discard precedent, `bulk-copy.ts:159`; Q-B1). The user's own reason is still required at prepare, which is the decision step. Refusals: too many, none match, open draft (the existing `openDraft` copy with "Open Revocation jobs"), pending limit (existing `pendingLimit`).
- **Fixed list.** On success the explicit selection is cleared and the bar shows "{matched} people fixed at {time} for these filters" with the stored filter as read-only chips (`.m01-chips` markup, chip text from the Part A chip model), or "{m} people fixed at {time} ({n} when counted)" when the fixed count differs from the counted one; then "Prepare revocation" and "Discard list". While a fixed list is open, the row checkboxes and the select-loaded control are not rendered, and the bar says "Discard the list to choose people one by one." (Q-B4; new [TW] string). A later filter change shows "The fixed list keeps its own filters. Changing the filters above does not change it."
- **Survives tabs and reloads.** The fixed list is kept with the explicit selection in the module-level `kept` state, and recovered from the server on first load: the console reads `GET /identity/bulk-revocations?limit=1`; drafts rank first and only the caller's own are visible (`registry.go:186-221`), so an own open draft with `selection = filter` restores the bar.
- **Prepare (BulkFlow, filter mode).** No staging. Step 1 keeps effect and reason; the selection summary shows the fixed chips, time and count instead of the ID list. Submit sends `prepare` on the draft.
  - `prerequisite_failed` → the **anchor dead-end**: title "Preview refused", the HYB-294 M6 copy "This list would leave no administrator who can manage access. Narrow the filters, for example Workspace role: Member." (KO/AR as HYB-294), no administrator list and no "Remove from selection" (a filter list cannot deselect, Q4), and "Discard list". After discarding, the user returns to the selection bar with the filters intact. Superseded in part by [HYB-324](../HYB-324/implementation.md): the owner of a filter list can now remove chosen people, in a removal mode that brings the checkbox column back for that purpose only; the dead-end names removal as a remedy and offers a way back to the list, still without a list of administrators.
  - `invalid_stage` → "This list can no longer be prepared: it expired, or nobody in it can be revoked now. Discard it and select again." with "Discard list" (new [TW] string; D-2).
  - `permission_denied` → the existing failure text.
- **Preview.** `PreviewBody` adds, for filter lists, the fixed facts: chips, fixed time, matched, "{n} no longer match the filters and are not in the list.", and "People who match after {time} are not in this list. Select again for a new list." The exclusion summary adds the `no_longer_matches` row. Execute and the confirm dialog are unchanged.
- **Revocation jobs tab** (`RevocationJobs.tsx`, deviation X-2): each job row states its selection basis ("Chosen people", "Everyone matching filters", "Selection basis not recorded"; [TW]); for filter lists the row carries the fixed filters as **one text line** (term texts joined by " · "), and the **read-only chips appear in the job detail** (visual addendum A-2, root decision 2 of `visual-spec.md` §17). Another administrator sees "Search applied" instead of the text, in the row line and in the detail chip.
- **Superseded copy.** `b.filterWide` and its test and QA expectation (`qa299.mjs:363`) are removed (`product-terminology.md`); the regression run uses a copy of `qa299.mjs` with that one expectation changed.
- **Settings disclosure** (Q-B3): the M01 settings rows add `bulk_revocation_max_matching` (identity group) and the two users-read budgets `user_filter_scan` and `user_count_scan`, each labelled, so every changeable default is readable on the screen (owner rule, 2026-11-17).

### D-8. Phase B, cleanup and scale (HYB-294 D8)

- Phase B stays under the domain write lock before the answer (`retention-access.md:98`, RET-05). No change to `decisionledger.BatchLimit`.
- **Cleanup must scale (deviation X-4; review F2, root decision: fix it).** Facts on `3bdb750` (`bulkcleanup.go:84-157`):
  - the job's task limit is 100 tasks of 100 items, so more than 10,000 items cannot get a cleanup job (`bulkcleanup.go:22`, `modules/jobs/registry.go:160`);
  - every task calls `s.bulk.Get`, a full `copyView` of the job (all items) (`bulkcleanup.go:111`), under the domain write lock (`:87-94`);
  - every item calls `Cleaning` and `Cleaned`, each through `update`, which recounts every item (`count(e.v)`, `job.go:42`), so a task is O(chunk × items) and the job O(items²);
  - `cleanPerson` scans every session per item (`sessionsOf` → `sessions.byLogin`) and every pickup twice (`dropPickups`, `directory.go:199-205`).

  Changes:
  1. **Task limit** = `ceil(max_matching / cleanupChunk)`, at least 100 (1,000 at 100,000). A test at 10,001 deny-confirmed items fails on `3bdb750` and passes after.
  2. **Span accessor.** New `bulkrevoke.Registry.Span(workspace, id, from, to) (View-without-items header fields, []Item, error)` copies only the task's items (plus `DenyAt`, `ID`, `Workspace`). `cleanTask` uses it instead of `Get`.
  3. **Batched, incremental counts.** New `CleanedBatch(workspace, id, indexes, cleanup)` and `CleaningBatch` apply a task's state changes in one module call, and `Counts` is maintained incrementally by outcome transition (unprocessed → succeeded/failed) instead of `count(e.v)` per change. The per-item checkpoint stays (resume after a lease loss marks at most one chunk again; `Cleaning` is already idempotent for `CleanupRunning`). A property test asserts that after any sequence of `Header`, `Record`, `Cleaning`, `Cleaned`, `Failed` and the batch forms, the incremental `Counts` equals `count(v)` (kept as the test oracle).
  4. **Lookups in cleanup.** A sessions-by-login index inside `sessions` (maintained by its add and drop) replaces the scan in `byLogin`; `directory` keeps pickups indexed by target user and by session (maintained in `directory.go`, its only writer per the Part A source test), so `dropPickups` by target or by session is O(matches). Equality tests against the scanning forms on random data.
  5. **Measured** at 5,000, 20,000 and 100,000 targets: lock hold per task (one chunk of 100), total drain time, and the drain against the job life of 1 h. These are the Q-B5 evidence.
- **Prepare impact indexes (HYB-294 M3).** Once per prepare: sessions by login (one pass over `s.sessions.byID`), pickups by target user and by session, and each proposal store counted by proposer in one pass. `impact` then looks each target up. Results equal the per-target functions (test comparing both on random data).

### D-9. Incremental latest-outcome index (HYB-294 N5, Linear comment)

- Keep `latestIndex[workspace][user]` as the answer, and add `refs[workspace][user]`: the kept executed jobs that list the user as a non-excluded, non-held item, as `latestRef{number, index}` sorted by number, highest first. `latestIndex[w][u] == refs[w][u][0]`.
- **Execute:** after the items are built, for each indexed item insert its ref into the user's list by number (usually at the front) and update `latestIndex`. Cost O(items of the new job).
- **Prune:** `prune` returns the evicted executed entries; for each, remove its refs from its users' lists (evicted jobs are the lowest numbers, so usually the tail) and recompute `latestIndex` for those users only. Cost O(items of evicted jobs).
- **Restore:** an existing entry with the same number is un-indexed before being replaced; each restored job is indexed. Discard and expiry never touch the index (they never evict executed jobs except through `prune`).
- **Tie-break and eviction order (review F9).** The oracle (`latest_oracle_test.go`) takes, per user, the **highest-numbered executed job** that lists the user as a non-excluded, non-held item, and within that job the **first** such item index. A user appears at most once per job (targets and exclusions are disjoint), but the index keeps the "first index" rule explicitly. Executed jobs that are still unfinished are indexed (the oracle includes every `Executed` state) and are never evicted; when they finish, a later `prune` may evict **older-numbered finished jobs out of order**, or jobs from the **middle** of a user's list (because numbers are assigned at draft creation and execution order differs). Removal is therefore by binary search on number, not tail-only.
- Memory: one ref (16 bytes plus slice overhead) per retained indexed item, beside the much larger `Item` it points at; measured.
- Tests: the existing `latest_test.go` cases; a property test over random sequences of execute, discard, restore and eviction (including out-of-number-order execution and repeated restores) comparing `LatestFor` with the full-scan oracle `latest_oracle_test.go` after every step, including sequences where a job stays unfinished, later finishes (all items cleaned or failed) and is followed by `prune` that evicts it or a lower- or middle-numbered job, and executions in a number order different from creation; a work-count test that the items visited per execute equal the new job's plus the evicted jobs' items (test-only counter through an unexported field read in the package test, no production hook).

## 4. Settings (changeable installation defaults)

| Setting | Flag | Group | Unit | Default | Bounds | Cross-check |
| --- | --- | --- | --- | --- | --- | --- |
| `bulk_revocation_max_matching` | `-bulk-revocation-max-matching` | identity (shown with the bulk settings) | people | 5,000 | 1–100,000 | ≥ `bulk_revocation_max_targets`, else `ErrConfig` naming this setting |
| `user_count_scan` | `-user-count-scan` | users (beside `user_filter_scan`) | people | 20,000 | 1,000–100,000 | none |
| `bulk_revocation_retained_people_max` (F3) | `-bulk-revocation-retained-people-max` | identity | people | 1,100,000 (root decision 2026-11-19 from the measurement; the smallest value that keeps the shipped defaults starting) | 10,000–20,000,000 | `(2 × bulk_revocation_jobs_kept + bulk_revocation_pending) × bulk_revocation_max_matching` ≤ this, else `ErrConfig` naming the cap and the three inputs (G2) |

- Source `default` or `start_flag`, as every HYB-298/299 setting; validated in `server.New`; flag registration comes from the lists (`main.go:95`). `main_test.go` covers bounds, form, usage text and the cross-check.
- The group rule (`withDefaults`) means tests that set one member of a group must set the new member too; those test edits are listed in verification.
- The explicit limit `bulk_revocation_max_targets` (500, 1–5,000) is unchanged.
- **Retained memory bound (review F3; round 2 G1, root decision option b).** What stays in memory:
  - up to `jobs_kept` finished executed jobs, each one `Item` per target and excluded person (`prune`, `registry.go:255-272`);
  - **also up to `jobs_kept` ended lists** (discarded or expired) beside them, because `prune` keeps `Kept` of each kind, and an ended list keeps its draft IDs or its prepared `Targets` and `Exclusions` (option b: what ended lists keep is **not** changed);
  - up to `pending` open lists (drafts: one map entry per ID; prepared manifests: one `Target` per person).

  Worst case: `(2 × jobs_kept + pending) × max_matching` people (`max_matching ≥ max_targets`, so explicit lists are bounded too). Executed jobs that are still unfinished are never pruned and can exceed `jobs_kept`; that is residual R-B3.
- **Measured** at 100,000: bytes per retained executed item (Go heap delta after `Execute` and cleanup, including the index ref), bytes per pending-draft ID, bytes per person of an **ended prepared list** (prepared, then discarded), and bytes per person of an **open prepared list**; the larger of the open-draft and open-prepared figures is used as `bytes_per_open_person` (round 3 note). Verification states the bound `jobs_kept × max_matching × bytes_per_item + jobs_kept × max_matching × bytes_per_ended_person + pending × max_matching × bytes_per_open_person` for the defaults (100, 20, 5,000) and for the maxima.
- **Default (G2).** The measured proposal must keep every configuration with `jobs_kept ≤ 100` at the default `pending` (20) and `max_matching` (5,000) starting: `(2 × 100 + 20) × 5,000 = 1,100,000`, so the default is at least 1,100,000. Root set the default to exactly 1,100,000 after the measurement (about 870 MB worst case at the measured bytes per item; see verification).
- **Start refusal (G2).** `server.New` returns `ErrConfig` naming `bulk_revocation_retained_people_max`; the `hb-api` start log line and the flag usage text name the cap and all three inputs, `bulk_revocation_jobs_kept`, `bulk_revocation_pending` and `bulk_revocation_max_matching`, with the formula. `main_test.go` pins the usage text and a server test the error and log fields.
- **Upgrade behaviour change (G2).** Some configurations accepted on `3bdb750` are refused at start. Example: `jobs_kept 1000`, `pending 200`, `max_matching 5000` (the minimum, since it must be at least `max_targets 5000`) gives `(2,000 + 200) × 5,000 = 11,000,000`. Remedies: lower `-bulk-revocation-jobs-kept`, `-bulk-revocation-pending` or `-bulk-revocation-max-matching`, or raise `-bulk-revocation-retained-people-max` after checking the host's memory. Verification records this example and the remedies as an upgrade behaviour change. Q-B6.

## 5. Contract changes (summary)

| Kind | Change |
| --- | --- |
| Command | `bulk_revocation.add_matching` (fields in D-1); refusals `invalid_request` (fields), `unsupported_capability` (`method`), `prerequisite_failed` (none match, too many, open draft, pending limit), `permission_denied`, step-up `reauthentication_required` |
| Command | `bulk_revocation.add_targets` on a filter draft → `invalid_stage` |
| Command | `bulk_revocation.prepare` on a filter list: `prerequisite_failed` = anchors only; nothing to revoke → `invalid_stage`; new exclusion reason `no_longer_matches` |
| Read | `GET /api/v1/identity/users/count` (D-4), `x-reads` entry |
| Read | bulk job rows: `selection` (now also `filter`, `not_recorded`), `matched`, `fixed_at`, `fixed_revision`, `filter`, `search_masked`; detail `exclusions.no_longer_matches` for filter lists |
| Digest | filter fields for filter drafts only; explicit unchanged |
| Read (existing rows changed, F10) | replayed jobs (`origin: ledger_replay`) now report `selection: "not_recorded"` instead of `"explicit"`; recorded in the OpenAPI `x-reads` notes for the bulk reads and in the console's basis label; a test pins it |
| Settings | section 4 table, including the retained-people cross-check that can refuse a previously accepted start configuration |
| OpenAPI | the new command appears with the command definitions; the count read under `x-reads`; the bulk read entries note the `selection` values `explicit`, `filter`, `not_recorded` and that replayed rows changed from `explicit` to `not_recorded`; the pinning test extended |

## 6. Tests

**`modules/bulkrevoke`** (100% statement coverage kept)
- `AddMatching`: every validation branch; open draft; pending limit; `MaxMatching` boundary; `Add` on a filter draft → `ErrState`; owner-only visibility.
- Digest: pinned explicit vector; the oracle copy of today's `Digest` equals the new one for explicit over random inputs; for filter drafts, changing any of the terms, `Matched`, `FixedAt` or `FixedRevision` changes the digest; an explicit digest is refused for a filter list (`ErrDigest`).
- `CanonicalFilter` property: injective over random term lists with values containing `:`, digits and separators.
- Incremental index: D-9 tests.
- `property_test.go` extended with filter drafts and the `no_longer_matches` reason; counts invariant (`HYB-299/plan.md:1080-1084`).

**`apps/api/server`** (100% statement coverage of changed and new files)
- `bulk_matching_test.go`: each refusal in D-1/D-5; the fixed set equals a brute-force evaluation; a person created after fixing who matches is absent at prepare; drift (role change, period end) excluded as `no_longer_matches`, a changed-but-matching member stays; classification order (self, departed); anchor stranding → `prerequisite_failed` with the draft still a draft; all no longer matching → `invalid_stage`; masking for another holder; replay shows `not_recorded` and keeps the counts invariant; trust-hold row; log line carries keys only (no value, no login).
- Count read: exact, lower bound (small `user_count_scan`), `over_limit` stopping at limit + 1, refusals, runs under `RLock` (a concurrent list read proceeds), bounds of the setting.
- **Property test against an independent model** (extends `users_property_test.go`'s population generator): random populations (up to 2,000), random filters over all keys, random `max_matching` and `user_count_scan`. Assertions: (1) `add_matching`'s fixed set equals an independent model written in the test (its own predicate code, not `userMatcher`, with `strings.EqualFold`-based search as in Part A); (2) the count answer equals the model's prefix count under the three stop rules; (3) after random mutations (role, membership state, period end, departure, link changes, time advance), prepare's `no_longer_matches` set equals the model's members-no-longer-matching set and targets ∪ exclusions = the fixed set; (4) the manifest digest equals an independent recomputation from the model's targets, exclusions and filter fields. Each assertion is shown to fail on a deliberately broken predicate or stop rule before it is trusted.
- Impact indexes equal the per-target functions; cleanup at 10,001 items (fails on `3bdb750`).
- `-race -count=40` on the new tests; the Part A suites and `bulk_*_test.go` unchanged and passing.

**Mutation** (scratch copy, session `GOCACHE` cleaned per batch; tests run `-count=40` first): `modules/bulkrevoke` `registry.go` (AddMatching, Add refusal), `manifest.go` (digest branch, prepare selection), `latestindex.go` (incremental index), `types.go` (limits); server `bulkstage.go`, `bulk.go` (re-check, codes), the walker, `userscount.go`, `bulkread.go` (masking, exclusions key), `bulkimpact.go` (indexes), `bulkcleanup.go` (limit, span, batches), `job.go` (incremental counts, batch transitions), `session.go` and `directory.go` (indexes), `settings.go` (cross-checks), `identityledger.go` (replay selection). Every survivor is killed or explained by its readers.

**Console**: `bulk-model.test.ts` (count label choice, replace confirmation, fixed-differs line, recovery from a job row), `identity-bulk.test.ts` (command fields, count read parsing, new row fields), render tests EN/KO/AR for the bar states, the fixed line, the dead-end and the preview facts; `tsc`, `npm test`, build.

## 7. Scale measurements (recorded numbers, not thresholds; `-short` skips)

`bulk_scale_http_test.go`, through the HTTP handler, 100,000 people built as in Part A (invite-and-accept pairs; Part A took 66 s). Lists of 5,000, 20,000 and 100,000 matched people (`-bulk-revocation-max-matching 100000`). Side data as HYB-294 M3: open sessions for 10% of targets, pickups for 1%, pending proposals in each store for 1%.

Record:
- count read at 20,000 and 100,000 people (`user_count_scan` 20,000 and 100,000) and the latency of a concurrent write command waiting on it;
- `add_matching`, prepare (impact before and after the D-8 indexes) and execute durations; phase A and phase B separately; ledger syncs; ledger bytes per item; time to answer;
- latency of a concurrent users read waiting on the lock during execute;
- the D-9 index: per-execute index time with 100 kept jobs × 5,000 and with kept 100,000-item jobs, before (rebuild) and after (incremental); retained memory per indexed item;
- `List`, `Get` and `bulkRow` copy costs with kept large jobs (before and after the D-6 `List` change);
- cleanup at 5,000, 20,000 and 100,000: task count, lock hold per task, total drain time against the 1 h life, before (on a scratch copy of `3bdb750` logic where it can run at all, i.e. 5,000) and after (F2, Q-B5);
- bytes per retained executed item, per pending-draft ID and per person of an ended prepared list at 100,000, and the resulting bound for the defaults and the maxima (F3, G1, Q-B6).

Assertions only: isolation; the deny happens before the first ledger write; the counts invariant; the cleanup job is created at 100,000 and drains to completion.

## 8. Browser QA and regressions

- **New runner `qa323.mjs`** (scratchpad; helpers copied from `qa294.mjs`): live `hb-api` from this tree with a 300-person harness, `-user-count-scan 1000` for the lower-bound case and `-bulk-revocation-max-matching 2` for the too-many case on separate runs. EN, KO and AR at 1440, 1280, 1024 and 390.
- **Every new check fails first:** the runner is run against the `3bdb750` build (no action exists, so every Part B check fails) and a self-test phase breaks each property on a live DOM (removed fixed line, coloured chip border, unisolated AR time, hidden count wording) before the real run.
- Scenarios: the action and its label with and without filters; counting state; exact and "at least" labels; `over_limit` before trying; none match; the replace confirmation; the fixed line and "({n} when counted)" after a change between counting and fixing; filters changed after fixing; checkboxes absent while fixed; recovery after a tab change and a reload; prepare with the fixed facts; **drift** (a second session changes one member's role between fixing and preparing → "1 no longer match"); the **anchor dead-end** (a filter that includes every anchor administrator) with Discard list returning to the bar; expired list (`-bulk-revocation-manifest-minutes 5` with a clock-advanced harness, or recorded as not run if the harness cannot advance time); execute and the job row's basis; another administrator sees "search applied"; keyboard order and live-region announcements; the HYB-299 layout checks (no page scroll, header/cell parity, no repeated block, AR isolates, no word joins, neutral borders).
- **Regressions** (scratchpad runners, each on its own ports): `qa294.mjs` (Part A), `qa299.mjs` groups A–E, L, N, P (copy with the `filterWide` expectation removed), `qa264on294.mjs`, M02 `qa297.mjs`, M03 `qa298on297.mjs`, HYB-293 `qa293on297.mjs`, P01–P05 `qa315bodyon297-full.mjs` (E with `--seed none`).

## 9. Allowed files

- `modules/bulkrevoke`: `types.go`, `registry.go`, `manifest.go`, `latestindex.go`, `job.go` (span accessor, batch cleanup transitions, incremental counts; F2); tests (new `cleanup_counts_test.go` property test), (`drafts_test.go`, `manifest_test.go`, `latest_test.go`, `property_test.go`, new `matching_test.go`, `index_property_test.go`).
- `apps/api/server`: `bulkstage.go`, `bulk.go`, `bulkimpact.go`, `bulkread.go`, `bulkcleanup.go` (limit, span, batches, lookups; F2), `session.go` (sessions-by-login index; F2), `directory.go` (pickup indexes; F2), `userfilter.go` (walker, shared parse use), `users.go` (walker adoption), new `userscount.go`, `boundary.go` (route), `settings.go`, `openapi.go`, `trusthold.go` (one row), `identityledger.go` (replay selection, X-1); tests (`bulk_matching_test.go`, `users_count_test.go`, `bulk_scale_http_test.go`, the property test, and the listed settings test edits).
- `apps/api/cmd/hb-api/main_test.go` (flags come from the lists; `main.go` only if usage text needs it).
- Console: `service/identity-bulk.ts` and test; `users/Selection.tsx`, new `users/Matching.tsx`, `BulkPrepare.tsx`, `RevocationJobs.tsx` (X-2), `Live.tsx` (wiring only, X-2), `bulk-model.ts` and test, `bulk-copy.ts`, `filter-model.ts` (export the chip model for a stored filter, if not already reusable), `bulk.css`, render tests; `package.json` test list.
- Docs: `docs/features/HYB-323/*`, `docs/ui/terminology/product-glossary.md` (new section), HYB-294 `visual-spec.md` §9 note only if root asks (the addendum lives under HYB-323).
- Scratchpad: `qa323.mjs`, the `qa299` copy, the mutation driver.

## 10. Copy (EN / KO / AR)

The HYB-294 Part B table (`../HYB-294/plan.md:593-607`) is adopted unchanged as the starting point. New strings forced by this plan, marked **[TW]**, are drafts for the TW and native-language review:

| Key | EN | KO | AR |
| --- | --- | --- | --- |
| selectAll (no filter) [TW] | Select everyone in the list | 목록의 모든 사람 선택 | تحديد كل من في القائمة |
| counting [TW] | Counting people who match… | 조건에 맞는 사람을 세는 중… | جارٍ عدّ المطابقين… |
| fixedNoChecks [TW] | Discard the list to choose people one by one. | 사람을 한 명씩 고르려면 목록을 폐기하세요. | تجاهل القائمة لاختيار الأشخاص واحدًا تلو الآخر. |
| listClosed [TW] | This list can no longer be prepared: it expired, or nobody in it can be revoked now. Discard it and select again. | 이 목록은 더 이상 준비할 수 없습니다. 만료되었거나 지금 회수할 수 있는 사람이 없습니다. 폐기하고 다시 선택하세요. | لم يعد ممكنًا تحضير هذه القائمة: انتهت صلاحيتها أو لا يوجد فيها من يمكن إبطاله الآن. تجاهلها وحدّد مرة أخرى. |
| matchRationale [TW] | Selected everyone who matched the filters on the revocation screen | 회수 화면에서 필터에 맞는 사람 모두 선택 | تحديد كل المطابقين لعوامل التصفية في شاشة الإبطال |
| basis [TW] | Chosen people / Everyone matching filters / Selection basis not recorded | 직접 선택 / 필터에 맞는 모든 사람 / 선택 기준 기록 없음 | أشخاص مختارون / كل المطابقين لعوامل التصفية / أساس التحديد غير مسجّل |
| reason no_longer_matches [TW] | No longer matches the filters | 더 이상 필터에 맞지 않음 | لم يعد يطابق عوامل التصفية |
| settings labels [TW] | People per filter-wide revocation / People examined per search page / People examined per count | 필터 전체 회수당 인원 / 검색 페이지당 확인 인원 / 집계당 확인 인원 | الأشخاص لكل إبطال حسب التصفية / الأشخاص المفحوصون لكل صفحة بحث / الأشخاص المفحوصون لكل عدّ |
| retainedPeopleMax label [TW] (G3) | People kept in revocation lists in memory | 메모리에 보관하는 회수 목록 인원 | الأشخاص المحفوظون في قوائم الإبطال في الذاكرة |
| retainedPeopleMax hint [TW] (G3) | Counts finished jobs, ended lists and lists waiting to run, each up to the people per filter-wide revocation. | 완료된 작업, 종료된 목록, 실행 대기 중인 목록을 각각 필터 전체 회수당 인원까지 셉니다. | يشمل المهام المكتملة والقوائم المنتهية والقوائم بانتظار التنفيذ، كلٌّ منها حتى عدد الأشخاص لكل إبطال حسب التصفية. |

The setting key `pending` appears only in the formula and the flag name; user-facing text says "lists waiting to run" (G3).

The glossary gains a "M01 select everyone matching (HYB-323)" section: fixed list, filter-wide selection, no longer matches, selection basis.

## 11. Deviations from the HYB-294 plan, and clarifications

- **X-1.** Replay sets `selection = not_recorded` for every replayed job (`identityledger.go:267` today says `explicit`). HYB-294 assumed replay "keeps no selection"; it keeps a wrong one. `identityledger.go` was not in the HYB-294 Part B file list.
- **X-2.** `Live.tsx` (wiring) and `RevocationJobs.tsx` (selection basis on job rows) are touched; HYB-294 listed neither, but its QA step "the job row with `selection`" needs the second, and the first passes the fixed list into `BulkFlow`. New UI goes into new `Matching.tsx`.
- **X-3.** On a filter list, "nothing left to revoke" answers `invalid_stage` instead of `prerequisite_failed`, so the console can show the anchor dead-end copy truthfully (refusal fields carry no cause; `modules/commands` is not changed).
- **X-4.** Cleanup scales (F2): task limit from `max_matching`, span accessor, batched incremental counts, session and pickup indexes. At `3bdb750` a list above 10,000 people never gets cleanup, and below that the drain is quadratic in items. `job.go`, `session.go` and `directory.go` join the files.
- **X-7.** New setting `bulk_revocation_retained_people_max` and its start-time cross-check (F3).
- **X-8.** `parseUsersQuery` returns the offending parameter name; the read's answers are unchanged (F6).
- **X-5.** One shared walker for the list, the count and the command (refactor of Part A's `usersScan`).
- **X-6.** `Registry.List` no longer copies items it then drops (HYB-294 said "no change unless the numbers call for one"; the copy is pure waste, so it is changed and measured).
- **C-1.** The filter digest also covers the base revision (`ui-design.md:421`).
- **C-2.** Classification order on a filter list puts `self` and `not_found` before `no_longer_matches` (D-2).
- **C-3.** `add_matching` answers the applied filter as `filter.<key>` fields (the command result is a flat string map).

## 12. Owner questions (each ships with the stated default, changeable)

Carried from HYB-294 with their defaults: Q2 (`-user-count-scan` 20,000), Q3 (`-bulk-revocation-max-matching` 5,000; the measured 100,000 hold is reported for acceptance), Q5 (no-filter selection allowed), Q8 (search text shown to the preparer only; **this is a display rule, not secrecy**: the digest covers the canonical filter including `q`, the command journal stores the field values, and the console link may carry it, F5). **Q4 stays an open finding, not a default**: a filter list cannot remove chosen people (`ui-design.md:421` "제외"); the follow-up is **HYB-324** (already open). Part B ships without it, and the cross-check marks `ui-design.md:421` exclusions as **partial**.

New:
- **Q-B1. Rationale of `add_matching`.** Default: a fixed system text in the user's language; the user's reason is required at prepare. Alternative: a reason field in the select-matching region.
- **Q-B2. Empty filter list at prepare.** Default: `invalid_stage` with the "can no longer be prepared" copy (X-3). Alternative: a shared ambiguous refusal text.
- **Q-B3. Settings shown.** Default: show `bulk_revocation_max_matching`, `user_filter_scan` and `user_count_scan` in the M01 settings disclosure. Alternative: keep the scan budgets unshown (Part A D6).
- **Q-B4. Checkboxes while a list is fixed.** Default: not rendered, with a sentence. Alternative: rendered and disabled (a control that does nothing; `usability.md` argues against it).
- **Q-B5. Cleanup at 100,000.** Default: one cleanup job with a scaled task limit and the F2 changes. Evidence: the measured per-task lock hold and total drain at 5,000, 20,000 and 100,000. If the drain exceeds the job life, the choice between a longer life and several jobs goes to the owner with the numbers.
- **Q-B6. Retained people bound.** Default: `-bulk-revocation-retained-people-max` 1,100,000 people (root decision 2026-11-19 after the measurement: about 870 MB worst case), changeable at start; configurations whose `(2 × jobs_kept + pending) × max_matching` exceeds it are refused with the cap and the three inputs named. Alternative: no bound (memory grows with `jobs_kept × max_matching`).

## 13. Facts, assumptions, unknowns

- **Facts:** the citations in section 2, read on `3bdb750` on 2026-11-18; the planning text in section 1, read from the primary checkout (`docs/planning/v1.16/` and `docs/ui/screens/M01/spec.md` are not tracked, so they are absent from this worktree).
- **Assumptions:** A1, the shared walker keeps Part A's results (guarded by the Part A property test). A2, `holdsRight` per target (`trustevidence.go:229`) is cheap enough at 100,000 without an index; measured, and indexed only if the numbers say so. A3, phase B remains linear (HYB-294 A3).
- **Unknowns:** the count read and write-lock latencies at 100,000; phase B at 100,000 over HTTP; bytes per ledger item; cleanup drain at 100,000; fsync latency on this host; the final KO and AR wording.

## 14. Residuals expected after HYB-323

HYB-294 R3–R8, R11 stand. **R-B1:** execute does not re-run the filter; time-based and sign-in, invitation, holder or email drift between preview and execute is not detected (F8). **R-B3:** unfinished executed jobs are never pruned, so retained people can exceed the bound while cleanup is unfinished (G1). **R-B2:** masking of the search text is a display rule; the digest, the command journal and links still carry it (F5). Q4 open finding (chosen exclusions). Replay shows no selection basis (R4, now truthful). Phase B lock hold at 100,000 is measured, not removed (R5, Q3). Ledger capacity for repeated 100,000-person revocations (R6) gets a measured bytes-per-item figure. The browser harness stays at about 300 people; 100,000 is measured in Go only.

## Review rounds

### Design review round 1 (2026-11-18): CHANGES REQUIRED (3 Medium, 7 Low, no High)

| Finding | Severity | Decision | Where |
| --- | --- | --- | --- |
| F1. The prepare re-check passes a nil member for an ID not in `r.members`; the People-vs-roster equivalence test must cover non-members (`membership=none`, `period`, `role`) and departed people | Medium | Accepted: nil for non-members stated; equivalence test over every key and value, including non-members, removed, suspended, ended and departed | D-2 |
| F2. Cleanup does not scale (full-view copy per task, full recount per item, session and pickup scans, 100-task limit) | Medium | Root decision, fix it: task limit, span accessor, batched incremental counts with a property test, session and pickup indexes; `job.go`, `session.go`, `directory.go` added; per-task hold and drain measured as Q-B5 evidence | D-8, section 7, section 9, X-4, Q-B5 |
| F3. Retained memory unbounded at `jobs_kept × max_matching` | Medium | Root decision: measure bytes per retained executed item and per pending-draft ID at 100,000; state the bound; new changeable setting `bulk_revocation_retained_people_max` with a start-time cross-check (provisional 2,000,000, replaced by the measured proposal); Q-B6 | section 4, section 7, X-7, Q-B6 |
| F4. Frozen lists are readable by any `administer_installation` holder | Low | Stated; masking tests for an administrator without the bulk right on rows, detail and items | D-5 |
| F5. Masking is a display rule, not secrecy | Low | Stated with the digest, the command journal and links; Q8 and R-B2 | D-6, section 12, section 14 |
| F6. Count when `over_limit` and the tie at exactly limit + 1; field-named refusal path for the parser | Low | `count = limit + 1` lower bound; both flags can be true; match stop checked first; `parseUsersQuery` returns the parameter name, the read's answers unchanged | D-1, D-4, X-8 |
| F7. `not_found` predicate unchanged, including `suspend_user` without membership, before `no_longer_matches` | Low | Stated | D-2 |
| F8. Execute does not re-run the filter | Low | Stated with the drifting terms; preview and copy say "at {time}" only; R-B1 | D-2, section 14 |
| F9. Incremental index tie-break and non-tail evictions | Low | First item in the highest-numbered job; removal by binary search; unfinished → finished → prune sequences in the property test | D-9 |
| F10. Replay changes existing rows from `explicit` to `not_recorded` | Low | Listed as a change to existing rows in section 5 and the OpenAPI notes | section 5 |
| Q4 follow-up | — | HYB-324 is the open follow-up; `ui-design.md:421` exclusions marked partial | section 1, section 12 | Since [HYB-324](../HYB-324/implementation.md) the exclusions chosen by the person are built, so this line of the planning document is covered.

### Design review round 2 (2026-11-18): CHANGES REQUIRED narrowly (F1–F10 resolved)

| Finding | Decision | Where |
| --- | --- | --- |
| G1. The bound omits ended lists, which `prune` keeps up to `Kept` beside `Kept` executed jobs, with their IDs, targets and exclusions | Root decision option b: the bound is `(2 × jobs_kept + pending) × max_matching`; what ended lists keep is unchanged; unfinished executed jobs above `jobs_kept` are residual R-B3; the scale test also measures an ended prepared list | section 4, section 7, section 14 |
| G2. The refusal must be operable | The error, start log and usage text name the cap and the three inputs; verification records the upgrade behaviour change with the example (11,000,000) and remedies; the measured default is at least 1,100,000 so `jobs_kept ≤ 100` at default pending and max matching always starts | section 4, Q-B6 |
| G3. Terminology | `pending` only in the formula; user text "lists waiting to run"; [TW] label and hint rows for the new setting | section 10 |

### Design review round 3 (2026-11-19): APPROVE

- Non-blocking note folded in: verification also measures an open prepared list and uses the larger per-person figure for open lists (section 4).
- D-7 amended per the visual addendum root decision (`visual-spec.md` §17 item 2): job rows carry one text line; chips are in the job detail.
