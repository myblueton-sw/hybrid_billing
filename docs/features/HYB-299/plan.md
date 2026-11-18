---
type: ImplementationPlan
title: M01 role periods and bulk revocation, with expiry and bulk decisions recorded in the installation decision ledger
description: Adds a role period (start and optional end) to every workspace membership, enforced by a time-aware effective-membership predicate and swept into a durable suspended state at its end, and a bulk revocation flow (targets staged in bounded batches into a caller-owned draft, frozen into a fixed manifest with an impact preview, confirmed, deny first, ledger job header and item records, chunked cleanup on modules/jobs, per-person outcomes) on the M01 users screen, built on the HYB-298 decision ledger and classified for the HYB-297 trust hold. Revised after security design review round 1.
status: stable
tags: [HYB-299, security, identity, M01, HB-03, HB-34, HB-42, IAM-05, IAM-07, IAM-08]
sources:
  - resource: docs/ui/screens/M01/spec.md
  - resource: docs/ui/screens/M01/design.json
  - resource: docs/planning/v1.16/requirements.md
  - resource: docs/planning/v1.16/retention-access.md
  - resource: docs/planning/v1.16/ui-design.md
  - resource: docs/planning/v1.16/domain-contracts.md
  - resource: docs/planning/v1.16/acceptance-trace.json
  - resource: ../HB-263/plan.md
  - resource: ../HB-264/plan.md
  - resource: ../HB-264/verification.md
  - resource: ../HYB-293/plan.md
  - resource: ../../reviews/HYB-291-planning-cross-check/report.md
  - resource: ../../reviews/HYB-301-trace-classification/report.md
---

# Scope and ownership

[HYB-299](https://linear.app/hybrid-billing/issue/HYB-299) builds the two M01 behaviours that HB-263 and HB-264 recorded as not built: role validity periods (`M01/spec.md:52`) and bulk revocation (`M01/spec.md:75`). Both go into the API and the screen. HYB-291 finding M10 names them (`docs/reviews/HYB-291-planning-cross-check/report.md:50`, `:129`).

- Base for this plan: main `07458be`. Branch: `feat/HYB-299-role-validity-bulk-revocation`. Worktree: `/private/tmp/hb299`. Machine: mac_mini.
- **Implementation starts only after HYB-298 and HYB-297 have merged.** The branch is then rebased on that main (see Dependencies and sequencing). Every `file:line` for current code below refers to `07458be` unless it names the HYB-298 or HYB-297 worktree.
- The M01 spec, design.json and the v1.16 planning files are in the authorized local workspace (`/Users/home/project/hybrid_billing/docs/...`), not in the Git tree of this base. Their line citations refer to that copy.

**Owner directions in force (2026-11-17, relayed by root).**
- D-1. Faithful to the planning: build the spec behaviour as written. Where a dependency is missing, block truthfully and say so. Never simplify silently.
- D-2. Provisional values ship as installation defaults that can be changed.
- D-3. Required decision records persist in the installation data directory.
- D-4. No emergency revoke path yet.
- D-5. Permissions and tenant boundaries need an independent design review.

**Ownership**
- Root (Claude Code) owns the plan, implementation, tests and browser QA.
- A separate read-only reviewer reviews this design and later the change, with a security focus (D-5; AGENTS.md core rules; `design-first.md`). The change touches permissions, membership state, durable security records and the public command contract.
- The `ui-designer` role (AI, read-only) leads the visual design of the selection, preview, confirmation and revocation-job views and compares the final renders (`service-screen-style.md`).
- A separate read-only technical-writing review checks the new terms and the EN, KO and AR copy (`product-terminology.md`).

## In scope

1. **Role periods.** Every workspace membership gets a role period: a start set by the server and an optional end. It is checked at every access check, swept into a suspended membership at its end, recorded in the decision ledger, and shown in the list and the detail.
2. **Bulk revocation.** Selection by canonical user ID, staged into a caller-owned draft in bounded batches (review B1), frozen into a fixed server manifest with exclusions and an impact preview, then confirmation and execution:
   - the deny is applied first for every target;
   - a job header record, then the item decisions, are committed to the HYB-298 decision ledger;
   - cleanup runs in chunks as `modules/jobs` tasks;
   - outcomes are reported per person with a fixed denominator;
   - the M01 "Revocation jobs" tab is replaced by a working view.
3. **A new installation-kind action** for bulk revocation (`M01/spec.md:75` "대량 회수 권한").
4. **The anchor administrator rule.** No command, sweep, bulk execution or replay may take the set of anchor administrators from non-empty to empty (restated after review m2; see "Anchor administrator rule").
5. **Time-aware effective membership** in `modules/identity` (review M1): a membership is effective only when its state is active and it has no end or `now < until`, wherever membership is checked.
6. **Ledger extension.**
   - New decision kinds, a job header record, a batch commit and format version 2 in `modules/decisionledger` (reviews M5, m11).
   - Replay of identity decisions at start, applying an increase only when its confirm record follows (review M3).
   - Intent, apply, confirm records for every increase that the identity replay could otherwise misapply (see Durability).
7. **Installation settings** for the new provisional values (D-2), on the HYB-298 mechanism.
8. **M01 AR copy corrections** requested by the coordinator (see "Console copy corrections in `users/live-copy.ts`").

## Not in scope

| Item | Why | Where |
| --- | --- | --- |
| Server search, filters and filter-wide ("select all results") selection | Needs server search and filters, which do not exist (`M01/spec.md:61-63`, `:104`) | HYB-294 (Todo). This plan blocks the selection truthfully and prepares the manifest model for it. |
| Durable records for the single-person commands `suspend_user`, `suspend_membership`, `remove_membership`, `depart_user` and a role demotion | Not in the ticket. They keep today's in-memory behaviour. | Residual R2, owner question Q6 |
| Emergency revoke path | D-4 | — |
| Two-person approval of bulk revocation | Not in the spec: `M01/spec.md:67` says the confirmation is an impact review, not an added approval step | Owner question Q3 |
| Scheduled future role start | The spec names an interval, not scheduling. A future start is a scheduled privilege increase with its own checks. | Owner question Q9 |
| Login realm column and full M01 column order | `M01/spec.md:59`; not part of this ticket | Residual R8 |
| Jobs, API keys and download links owned by a revoked person | No product job is run through a job runner in apps/api (`modules/jobs` exists but no app imports it; this ticket's cleanup becomes its first user), and humans hold no API keys (HYB-298 `plan.md:565`, `:569`) | Reported as not applicable or outside the controlled scope |
| A change to the shared command field limit (`Value: 4096`) or a new non-command route for target lists | Root decision on B1: staging through bounded commands instead | — |
| External directory leaver detection (`M01/spec.md:126`) | No external identity | — |
| Persistence of users, memberships, invitations, sessions and manifests | Every pre-launch store is in memory (HB-263 `plan.md:233`; HYB-298 R3) | Restart behaviour is stated below |

# Requirement cross-check

Every ID the M01 spec cites (`M01/spec.md:116-120`: HB-03, HB-34, HB-42, IAM-01–IAM-08), the M01 spec lines this ticket touches, and the planning text they depend on. "After" is this ticket's target. No acceptance test has run. Status meanings follow the [cross-check report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings).

| ID / source | Requirement (summary) | How it is satisfied after HYB-299 | Status |
| --- | --- | --- | --- |
| `M01/spec.md:52` | Role and validity period, required, editable within grant scope, grant ceiling and expiry checked | A role period on every membership. The start is set by the server; the end is optional and set by `role_until` or `identity.set_role_period`. Ceiling: an end cannot pass the setter's own role end or `-role-period-max-days`, and a setter with an end cannot write "no end" (review M2). Enforced by the time-aware effective predicate everywhere membership is checked (review M1), then swept. | covered, by interpretation: "required" is read as "every membership has a period"; whether an end date is mandatory is Q2 (review m9) |
| `M01/spec.md:37` | Detail drawer shows stable identity, role period, recent authentication, duties | The detail shows the role period (start, end or "No end date", "Ends soon"). Recent authentication and duties are unchanged. | covered for role period |
| `M01/spec.md:59` | Columns: user ID, display name, login realm, membership, role period, status, revocation effect | Role period and revocation effect are added. Login realm and the full order are not. | partially covered (R8) |
| `M01/spec.md:61` | Filters: workspace, status, auth method, role, expiry | Not built | deferred to HYB-294; the expiry filter uses this ticket's `role_until` |
| `M01/spec.md:63` | Server search, stable sort, selection kept by canonical ID | Selection is keyed by user ID across "Load more", reload and tabs | covered for selection; search is HYB-294 |
| `M01/spec.md:41` | Revocation result: effect time, separate session, key, cache and job cleanup progress; downloaded copies never shown as recalled | Job and per-person view: deny time, ledger sequence, cleanup items in three groups (done, not applicable, outside the controlled scope) | covered |
| `M01/spec.md:55` | Revocation revision and confirmation time; durable deny acknowledgement separate from cleanup | Bulk items and role-period ends carry the deny generation, ledger sequence and watermark. Cleanup is separate. | covered for bulk and role periods; single commands deferred (R2) |
| `M01/spec.md:75` | Selected-scope revocation: bulk revocation right; fixed manifest; role ceiling; security effect first, then chunked cleanup; confirm targets, exclusions and effect scope | `bulk_revocation.add_targets` (bounded batches into a caller-owned draft), `bulk_revocation.prepare` (freeze and digest), then `bulk_revocation.execute`, under the new action `revoke_access_in_bulk` | covered |
| `M01/spec.md:73` | User suspension: last-administrator protection and reauthentication | Reauthentication by HYB-293 (merged). Last-administrator protection is strengthened by the anchor rule. | covered |
| `M01/spec.md:79` | Tabs: users, invitations, membership and roles, revocation jobs | The "Revocation jobs" tab replaces the not-available state (`Live.tsx:446-455`) | covered |
| `M01/spec.md:95-98` | partial, blocked, error, revoked copy | partial and error on jobs; blocked on the anchor rule and ceiling; revoked through the existing signed-out handling | covered |
| `M01/spec.md:100` | Job enum; an unknown result is waiting with a reason; retry only where proven | Job state `running`, `waiting` (reason `decision_unconfirmed`) or `succeeded`. No retry button. | covered |
| `M01/spec.md:104` | Fixed snapshot and cursor; page selection separate from filter-wide selection; the latter fixes a server manifest, count and scope; changed targets shown as a new snapshot, never silently updated | Page selection (loaded rows) is built. Filter-wide selection is blocked with a truthful line until HYB-294. Targets that change after preview are **held** (보류), never silently updated, and the user prepares again. | covered for page selection; filter-wide blocked (HYB-294) |
| `M01/spec.md:106` | Asynchronous job, fixed denominator, success / held / failed / unprocessed, no invented percentage | Counts per job with `targets = succeeded + held + failed + unprocessed`, a property-tested invariant | covered |
| `M01/spec.md:112` | Mobile: destructive bulk selection shows the target summary, then a confirmation step | Preview step, then confirmation step, below 1280 px | covered |
| `M01/spec.md:114` | Keyboard order, dialog focus, live region, never colour alone | Applied to the new views | covered |
| `M01/spec.md:124-126` | Open values: invitation expiry and provisioning, whole-suspend owner, external leaver detection delay | Unchanged | not touched |
| HB-03 (`requirements.md:53-63`) | Separate actions; recheck rights at job start and chunk boundaries; permission version changes invalidate pending jobs and caches; no leakage on ID tampering or revocation | A separate bulk action. The executor makes a fresh decision at execute. Every cleanup chunk rechecks the target's state. The deny bumps the identity generation, so old Answers fail `Recheck`. Tampered and other-workspace IDs are excluded as `not_found`. | partially covered; registry-wide leakage acceptance not built (as HB-263/264) |
| HB-34 (`requirements.md:607-619`) | Job states; receipt is not completion; partial success; chunk result with checkpoint; permission withdrawal | Job states and per-item outcomes. Deny and ledger commit happen before the answer. Cleanup runs as `modules/jobs` tasks with lease, fencing token, checkpoint and retry budget (`modules/jobs/types.go:32-219`, `registry.go:74-544`; review m7). The job registry is in memory, and there is no transactional outbox or event inbox. | partially covered (R4) |
| HB-42 (`requirements.md:749-761`) | Lifecycle including role expiry and immediate block; bulk revocation with fixed scope, chunks and resume; security block version first, so a batch delay never allows existing access | Role expiry is enforced at the instant of the end. Bulk revocation applies every deny before the answer and before cleanup. Resume after a restart is replay of the ledger before the listener serves; cleanup needs no resume, because the restart ends all sessions. | covered for role periods and bulk; MFA and external identity not built |
| IAM-01 (`retention-access.md:110`) | Bootstrap, invitation replay, acceptance from another workspace | Not changed. An invitation now carries the role end. Bootstrap tests rerun because bootstrap also grants the new action. | regression |
| IAM-02, IAM-03, IAM-04 (`:111-113`) | Sign-in protocols, identity reuse, token checks | Not touched | out of scope |
| IAM-05 (`:114`) | Suspension, departure and permission reduction reach sessions, keys, jobs and files; external detection delay measured | Role expiry and bulk revocation reach Answers, sessions, step-up evidence and secret pickups. Keys: not applicable. Jobs: not wired. Files: outside scope. External detection: not connected. | partially covered |
| IAM-06 (`:115`) | Recovery and last administrator are not a bypass | The anchor rule closes expiry and bulk as a way to strand the installation. Recovery is not touched. | covered for the paths added here |
| IAM-07 (`:116`) | 100,000-user lookup, invitation, sync and revocation: isolation, resume, block revision | An engine-level test runs 100,000 targets through manifest, deny, ledger and replay, and records the lock-hold time. The HTTP path stages targets in batches of at most 4,096 bytes and is bounded by `-bulk-revocation-max-targets` (default 500, at most 5,000). HYB-294 adds filter-wide selection on the same staging transport. Invitation and sync at that scale are not built. | partially covered |
| IAM-08 (`:117`) | Only sessions meeting step-up may run sensitive commands | HYB-293's rule covers every mutating command, including the five new ones, with no change (`stepup.go:40`) | covered by HYB-293 |
| UI-04 (`ui-design.md:547`) | 100k lists: filter, selection, page and total bases agree | Selection basis is explicit and counted; filters are HYB-294 | deferred (HYB-294) |
| UI-07 (`ui-design.md:550`) | 100k revocations without bypass or false progress | No false progress: fixed denominator, `received` and `decision_unconfirmed` never shown as success. Engine at 100k. | partially covered |
| `retention-access.md:59` | Authoritative deny and version first, then sessions, tokens, keys, caches and pending jobs; downloaded copies are not shown as recalled | Order: deny, ledger, cleanup. Cleanup items are listed in three groups. | covered |
| `retention-access.md:61` | 100k bulk revocation: security block revision first, token cleanup in chunks; progress never delays the effect | The deny for every target is applied in one step before anything else | covered |
| `retention-access.md:65` | Screen: permission preview, bulk suspension | Impact preview and bulk suspension | covered |
| `retention-access.md:98` (RET-05) | Durable ledger commit before a confirmed answer; re-apply before opening the API; a missing or stale ledger isolates | Uses the HYB-298 ledger. Identity replay runs in `Start` before the listener. Isolation rules are inherited. | covered within HYB-298's limits (R1 of HYB-298) |
| `ui-design.md:150`, `:384`, `:450` | Bulk preview separates people, memberships and roles, sessions and keys, pending jobs, service-account handover and last-administrator exceptions; cleanup progress never read as block progress; workspaces outside the scope are kept | The preview groups below follow this list. The effect scope "this workspace" keeps the account. | covered for people, roles, sessions, keys, service users, last-administrator and scope. Pending work is **partially covered** (review m10): pending proposals are counted, but each store's own recheck of a revoked proposer at approval time is confirmed only for M03 (HYB-298 basis) and M02 (HYB-297). The other stores are listed as "approval recheck not confirmed" until each is checked at implementation. Pending jobs are "not applicable: none run". |
| `domain-contracts.md:79` | Filter-wide operations capture a fixed manifest and recheck rights; bulk detail is never an unbounded payload | Draft staged in bounded batches, frozen into a manifest with digest; recheck at execute and per chunk; items paged | covered |
| `identity-session-permissions.md:31`, `:45` (Git tree) | Rotate sessions after privilege change; commit the deny before reporting the revocation effective | Sessions are ended on role end and bulk; the deny is applied before any answer, and the ledger before "confirmed" | covered |
| AR-09, DB-08, QA-04 (`detailed-requirements.json`) | Cite HB-42 but list M03, M04, P01, P02 and S02, not M01 | Not touched | not applicable to M01 |

# Current behaviour (facts from code at `07458be`)

## Memberships and access

- A membership is a role, a state, a version and a join counter. It has no validity (`modules/identity/registry.go:61-66`).
- Roles are `member` and `administrator`; membership states are `active`, `suspended` and `removed` (`modules/identity/types.go:86-100`).
- `Validity` (`NotBefore`, `NotAfter`) exists, but only for invitations and service credentials (`types.go:137-146`; `workspace.go:22-30`, `:374-381`).
- `Invite` takes a role and the invitation validity (`workspace.go:22`). `Accept` creates the membership with that role and no end (`workspace.go:117-125`). Acceptance requires the inviter to still stand (`workspace.go:104-105`, `:15-20`).
- `ChangeRole`, `SuspendMembership`, `ReinstateMembership` and `RemoveMembership` require a verifier of an active administrator and the expected version (`workspace.go:130-217`). `transition` refuses a change that would leave no administrator (`ErrLastAdministrator`, `workspace.go:153-155`; `registry.go:307`).
- `Members` returns user, role, state, version and user state (`workspace.go:219-243`; `MemberInfo`, `types.go:233-239`).
- Membership is checked in several places, each on `state == active` only: `activeAdministrator` (`registry.go:272-275`), the administrator count behind `strands` (`registry.go:288-323`), `inviterStands` (`workspace.go:15-20`) used by `Accept` (`:104`), `Subjects.Member` (`registry.go:171`), `Access` and `Recheck`. apps/api builds its basis inputs (`protect`, HYB-298's request basis) from `MemberInfo`.
- `Access` requires an active membership and returns an Answer whose `notAfter` is the session's absolute expiry (`boundary.go:114-140`). `Recheck` refuses an Answer whose membership state, role or version changed, or whose time is past `notAfter` (`boundary.go:169-196`).
- apps/api calls `Access` on every authenticated request. A refusal drops the session (`apps/api/server/commands.go:136-152`).
- The gate allows `administer_installation` only to a human whose Answer role is administrator (`modules/gate/gate.go:109`).

## M01 API

- Every M01 command and the users read use `administer_installation` (`apps/api/server/admin.go:371-405`; `users.go:42-69`).
- Every command requires a rationale of at most 256 bytes (`admin.go:57-59`; `directory.go:13`).
- The twelve commands are listed at `admin.go:379-398`. None takes a period.
- `protect` refuses self-targeting, and refuses a change to an active administrator unless another active administrator holds the installation grant (`admin.go:153-169`). Role is not considered beyond "administrator".
- The users read pages human users with a limit of 1–100 (`users.go:71-85`, `:108-135`). Its rows have no period (`users.go:14-32`, `:137-153`).
- Installation grants last 30 days from their creation at start or bootstrap (`directory.go:11`; `seed.go:234-253`). The holder table records that expiry (`seed.go:250`), and `holds(user, now)` stops counting a lapsed holder (`directory.go:96`). Nothing renews a grant while the process runs (HB-263 `plan.md:230`).
- Field types are `id`, `decimal`, `integer`, `boolean`, `text`, `timestamp` and `enum`; there is no list type (`modules/commands/types.go:98-115`). A timestamp must match RFC 3339 with `Z` or an offset (`modules/commands/execute.go:19`, `:39-41`). Request bodies are limited to 1 MiB (`server.go:38`; `boundary.go:84`).
- **Every field value is limited to 4,096 bytes**, and a command has at most 64 fields (`CommandLimits`, `server.go:124-128`; checked at `modules/commands/execute.go:91`). A newline-separated list of real user IDs (`user-` + 16 hex, 21 bytes, plus a newline) therefore fits at most 186 IDs in one field (review B1).
- Commands are rate-limited per caller with a burst of 60 and a refill of one every 250 ms (`server.go:127`).
- `modules/jobs` exists: a job, task, lease, fencing-token, checkpoint, retry and wait-reason registry (`modules/jobs/types.go:32-219`; `registry.go:74-544`, `Create`, `Lease`, `Checkpoint`, `Complete`, `Fail`, `Unknown`, `Cancel`, `Job`). No package under `apps/` imports it (repository search on `07458be`). The first draft wrongly said "no job runner" (review m7).
- The command endpoint runs `s.execute` (`commands.go:196`). Sign-in and redeem take the domain write lock in `signIn` (`signin.go:253`) and `redeem` (`signin.go:347`).
- Step-up applies to every mutating command with an empty exempt set (`stepup.go:22`, `:40`).
- Ending a person's sessions is `endSessions(login)`: `EndSession`, drop and delete pickups (`users.go:181-187`).

## M01 screen

- Tabs `users`, `invitations`, `membership`, `revocation` (`apps/console/src/modules/identity/screens.json`, M01).
- The `revocation` tab is a fixed unavailable state (`users/Live.tsx:446-455`; copy `users/live-copy.ts:225-237`).
- The table has four columns and no selection (`Live.tsx:330-381`). The detail shows role and membership state, not a period (`Live.tsx:280-319`).
- `users/live.css` (644 lines) is shared with M03, M04, M05, O01, O02 and O03 (imports in `organizations/pages/{organizations,changes,contracts}/Live.tsx`, `identity/pages/{service-accounts,my-security}/Live.tsx`).

## Records of HB-263 and HB-264

- Bulk revocation was deferred: "It needs asynchronous jobs and manifests" (`HB-263/plan.md:224`; `HB-264/plan.md:61`).
- HB-42 is recorded as deferred: "role validity periods and bulk revocation are not" built (`HB-264/verification.md:169`, `:181`; `HB-263/verification.md:203`).
- IAM-07 search and filters are assigned to HYB-294 (`HB-264/verification.md:176`).

## In-flight work this plan builds on (read-only)

**HYB-298** (`/private/tmp/hb298`, implementation in progress on base `be1bfa3`).
- `modules/decisionledger`: an fsynced, hash-chained JSONL ledger in `<data>/decisions/ledger.jsonl`, with states `ok`, `recovered`, `isolated`, `stopped` and `full` (HYB-298 `plan.md:376-531`).
  - Decision kinds are only `revoke_grant` and `revoke_delegation` (`modules/decisionledger/types.go:23-24` and `record.go:79` in that worktree).
  - `Commit` takes one decision (`plan.md:170`, `:445-453`).
- Revocations are never refused by ledger state. The deny is applied first; a failed, unknown or refused commit reads `decision_unconfirmed` (`plan.md:324-336`, `:507-510`).
- Privilege increases are refused while the ledger cannot record (`ledgerBlocksIncrease()`, `plan.md:512-529`). In that worktree `admin.go` gains `increaseAllowed` before `createUser`, `inviteExisting`, `reinstateUser`, `changeRole` and the reinstate step.
- Replay at `Start` before the listener, with the verified-prefix and durable-before-replay amendments (`plan.md:533-555`, `:595-600`).
- Installation settings are `hb-api` flags with sources `default` and `start_flag`, shown read-only (`plan.md:273-306`).
- Under `users/`, its diff changes only `live-copy.ts` (the `busy` message). It adds `access/access.css` and does not change `users/live.css` (git diff of `/private/tmp/hb298` at the time of writing). The coordinator's note says HYB-298 touches the shared `users/live.css`. That is rechecked after the HYB-298 merge.
- Glossary: "revocation" is الإبطال (إبطال) in AR (`docs/ui/terminology/product-glossary.md:69-70` in that worktree).

**HYB-297** (`/private/tmp/hb297`, plan approved after design review round 5).
- Four installation-kind trust actions and the exported predicate `auth.InstallationAction` (`HYB-297/plan.md:39`, `:182`).
- A third sample administrator (`:45`).
- A trust hold with a default-deny hold table over every registered command ID and POST route. Its test fails when an ID has no row (`:314-352`).
  - M01 rows: suspend, suspend membership, depart and remove are allowed; `change_role` is allowed only to member; `revoke_invitation` is allowed; create, invite, both reinstates and both link commands are refused (`:318-322`).

**HYB-294** (Todo, not started): M01 server search and filters (`HB-264/verification.md:176`).

# Design

## Terms

- **Role period**: the interval in which a membership's role is in effect. KO 역할 기간, AR مدة الدور.
  - The start (`role_from`) is the time the current role took effect. The end (`role_until`) is optional; without it, the period has no end date.
- **Effective membership** (review M1): state `active`, and no end or `now < until`, where `now` is the server clock (Timezone and clock) (below).
- **Anchor administrator**: an active human user whose membership in the installation workspace is effective, has the administrator role and has a role period with **no end**, and who holds a live installation grant at `now`.
- **Draft**: a caller-owned, unfrozen list of target user IDs built by `bulk_revocation.add_targets`. KO 회수 대상 초안, AR مسودة الإبطال.
- **Bulk revocation**: one manifest, executed once, that applies one effect to a fixed set of people. It is a job with per-person outcomes. KO 대량 회수, AR الإبطال الجماعي.
- **Manifest**: the fixed list of targets and exclusions, with its digest. KO 회수 대상 목록(고정), AR قائمة الإبطال الثابتة.
- **Held** (보류, معلَّق المعالجة — AR draft, see Copy): a target that changed between preview and execution and was not processed.
- **Excluded**: a selected person who is not a target, with a reason.

## Part A. Role periods

### Data (modules/identity)

- `membership` gains `from time.Time`, `until time.Time` (zero means no end) and `suspendedFor SuspendReason`. `SuspendReason` is `""`, `role_period_ended` or `bulk_revocation`.
- `MemberInfo` gains `From`, `Until` and `SuspendedFor`.
- Who sets `from`:
  - `Accept` sets `from` to the acceptance time and `until` to the invitation's role end.
  - `ChangeRole` sets `from` to the change time.
  - A restoring period change sets `from` to the restore time.
- `invitation` gains `roleUntil`. `Invite` takes it, and refuses it when it is non-zero and not after `validity.NotBefore`. The invitation's own `NotAfter` becomes `min(NotAfter, roleUntil)` when `roleUntil` is set. An invitation can therefore never be accepted after its role would end.
- **One time-aware predicate** (review M1): `r.effective(m *membership, now) bool` = `m.state == MembershipActive && (m.until.IsZero() || now.Before(m.until))`. Every membership check in the module uses it instead of `state == MembershipActive`:
  - `activeAdministrator` (`registry.go:272-275`) and the administrator count used by `strands` (`registry.go:288-323`);
  - `inviterStands` (`workspace.go:15-20`), so `Accept` (`:104`) refuses an invitation whose inviter's period ended, before or after the sweep;
  - `Subjects.Member` (`registry.go:171`), so the gate and HYB-298's basis see an ended membership as not a member;
  - `transition`'s actor check (`workspace.go:139`) and `Members`'s actor check through `administrator` (`registry.go:277`);
  - `Access` (`boundary.go:114-140`): it refuses with `ReasonExpired` when not effective. Otherwise the Answer's `notAfter` is `min(session absolute, until)`;
  - `Recheck` (`boundary.go:169-196`), which also refuses through the existing `notAfter` test.
- `MemberInfo` gains `Effective bool`. apps/api uses `Effective`, never `State` alone, for `protect`, `activeAdministrators` (`admin.go:143-151`), the anchor count, the bulk preview and basis, and HYB-298's request basis (subject and proposer membership state).
- The end is therefore exact everywhere, within the clock tolerance (see Timezone and clock), not only in `Access`: it does not wait for the sweep.
- A source test in `modules/identity` fails if `MembershipActive` is compared outside `effective` and the state transitions. Its list of checked sites explicitly includes `CreateServiceUser` (`workspace.go:259`, the owner's membership) and `onlySeat` (`registry.go:341`), which both move to `effective` (round 2).
- New exported functions, in a new file `modules/identity/periods.go` so `workspace.go` (498 lines) stays one responsibility:

| Function | Caller check | Effect |
| --- | --- | --- |
| `Workspace.SetRolePeriod(actor, workspace, target, until, expected)` | Active administrator verifier, as `transition` | Sets or clears `until` with version bump and generation bump. When the membership is suspended for `role_period_ended` and the new end is later than now or absent, it also reactivates the membership and sets `from = now`. |
| `Installation.ExpireRoles(op Operator, workspace, now) []Expired` | Operator, like `SuspendUser` (`installation.go:47`) | Every active membership with `until ≤ now` becomes `suspended` with `role_period_ended`, version and generation bumped. It never strands: a membership whose suspension would leave no administrator is returned as `blocked` and not changed. By the anchor rule this cannot happen; it is a defence, logged as `role_expiry_blocked`. |
| `Installation.ApplyRecorded(op, workspace, Recorded)` | Operator | Replay only. It sets a membership state, user state or role end to a recorded value (see Replay). |
| `Workspace.ReinstateMembership` (existing) | — | Refused with a new reason `ReasonRolePeriod` when the membership is suspended for `role_period_ended`, **and** when it is suspended for any other reason but its `until` has passed (review m5): reinstating it would restore a membership whose period is over. A new period is needed instead. |
| `Workspace.ChangeRoleUntil` (new) | As `ChangeRole` | Refused with `ReasonRolePeriod` on a membership suspended for `role_period_ended` or whose `until` has passed (review m5). A demotion of an administrator whose end passed at T0 is therefore refused, even before the sweep; the person has no effective role to demote. |
| `Check*` dry-runs (new, review M4) | Same checks as the mutation | `CheckSetRolePeriod`, `CheckReinstateMembership`, `CheckReinstateUser` and `CheckSuspend*` run every precondition of the matching mutation and return its error without changing anything. |

- `ChangeRole` keeps its signature. A second function, `ChangeRoleUntil`, takes the end. The existing callers stay unchanged, and apps/api uses the new one.
- `Installation.ReinstateUser` is not changed: it restores the account only. A membership whose period ended stays not effective after it.
- `Members` keeps its signature and returns the new fields.

### Who sets a role period, and the limits

- The same people who can change roles: an installation administrator (`administer_installation`), through the commands below. A member cannot.
- **Self.** A person cannot set their own role period: rule 1 of `protect` (`admin.go:155`).
- **Ceiling** ("부여상한", `M01/spec.md:52`).
  - The end may not be later than the setter's own role end when the setter's period has one.
  - A setter whose period has no end is limited only by the setting below.
  - The setter's 30-day installation grant (`directory.go:11`) is **not** used as the ceiling, because it is recreated at every start and would cap every period at 30 days. Recorded as decision PD-4.
- **Bounds.**
  - `role_until` must be a timestamp in UTC (`Z`), on a whole minute.
  - It must be at least one minute after the server time.
  - It must be at most `-role-period-max-days` days after the server time.
  - Anything else is `invalid_request` with field `role_until`.
- **Anchor rule.** Setting an end on an administrator, or any command below that would leave no anchor administrator, is refused with `prerequisite_failed`. This is the spec's `blocked` copy (`M01/spec.md:96`).
- **Omitted means no end** (review M2). For every command that writes a period end (`identity.create_user`, `identity.invite`, `identity.change_role` and `identity.set_role_period`), an omitted `role_until`, or `end: none`, means "no end date". Nothing keeps a previous end implicitly. The console pre-fills the current end so that an unchanged form never removes it.
- **The cap applies to every write** (review M2). A setter whose own period has an end cannot write "no end", and cannot write an end later than their own. This covers `create_user`, `invite`, `change_role` and `set_role_period`. It is refused with `invalid_request`, field `role_until`.
  - At acceptance the cap is held by `inviterStands`: any change to the inviter's period bumps their membership version, so an invitation made under an older, longer period is not accepted (`workspace.go:18`).
- **One direction function for existing memberships** (review M2, narrowed by round 2 N1). `periodIncrease(old, new period, oldEffective bool) bool` classifies every period write on an **existing** membership: `identity.set_role_period` and `identity.change_role`. A write is an increase when:
  - it removes an end;
  - it sets a later end;
  - it restores a membership that is not effective.

  It is a decrease when it sets an end where there was none, or an earlier end, on an effective membership. An equal end is no change. For existing memberships the same function decides the trust hold, `ledgerBlocksIncrease`, and the increase sequence (Durability).
- **`change_role` combines both tests** (round 3 N1 note). A `change_role` write is an increase when the new role is administrator (the existing HYB-298 role test in `changeRole`, `increaseAllowed(role == identity.Administrator)`, `admin.go:332` on main `d102fb4`) **or** `periodIncrease` says the period gets longer. The period term is added **to that same `increaseAllowed` call** (`increaseAllowed(role == identity.Administrator || periodIncrease(…))`), not as a separate check (round 4). It is a decrease only when neither holds. The hold, `ledgerBlocksIncrease` and the increase sequence all use this combined test.
- **New memberships are classified by the role, not by the period** (round 2 N1). `identity.create_user`, `identity.invite` and the acceptance that creates the membership keep HYB-298's role rule (HYB-298 `plan.md:520-522`, `:872`): as administrator they are increases and are refused while the ledger cannot record; as member they still succeed in `isolated`, `stopped` and `full`.
  - Their `role_until` is validated (bounds, cap) but writes **no ledger record**.
  - A user created by command does not exist after a restart (HB-263 `plan.md:233`). A seed user can be re-invited only after an unrecorded removal, which itself reverts at restart. In that case the end is lost with the removal (R2).
  - `createUser`'s multi-step path (`admin.go:231-241`, `errPartial`) is **not** routed through intent, apply and confirm; it keeps its existing partial-failure handling (HB-263 `plan.md:146`).
- **Whether an end is mandatory** is owner question Q2. The root default keeps it optional, enforces the cap, and uses a 366-day maximum.

### Timezone and clock

- The server stores instants in UTC. **Every time-dependent rule uses one server clock built on monotonic elapsed time** (review m1; replaced after round 3 C1). The earlier "maximum of wall readings" design is withdrawn: it froze time after a backward step and pinned it after a forward step.
  - **Base.** At `Start`: `base` = the maximum of the wall clock at start and the latest ledger `at` within the tolerance (below). `startMono` is a monotonic reading taken at the same moment.
    - **Pre-scan** (round 4 D1). The base is computed by a raw pre-scan of the ledger file **before `modules/decisionledger` opens it**. The pre-scan reads every decodable line for its `at`, `kind` and, for `clock_base` records, its choice. It verifies nothing and writes nothing.
    - The server clock is then fixed, and every record written during start (`recovery`, replayed `role_period_ended`, `clock_base`) is stamped with that final server clock.
  - **Now, with forward-only catch-up** (round 4 A8). The clock keeps an `offset`, starting at 0, that only grows. On each read:
    1. `proj = base + elapsed`, where `elapsed` = `time.Since(startMono)` on Go's monotonic clock;
    2. `lead = wall − (proj + offset)`;
    3. if `0 < lead ≤ tolerance`, then `offset = wall − proj`;
    4. return `proj + offset`, with the monotonic reading stripped.

    Properties:
    - **It never moves back**, because `offset` only grows.
    - **It never freezes after a backward wall step**: `proj` keeps advancing with elapsed time.
    - **Repeated steps do not accumulate**: `offset` is set to the lead, not added to.
    - **Forward wall jumps within the tolerance are absorbed**, which is the restrictive direction (ends come sooner). This also catches up a monotonic clock that paused during host sleep while the wall clock advanced: role ends fire on the first read after resume.
  - **A wall lead beyond the tolerance is not absorbed** (root decision, A8). If `lead > tolerance`, the clock keeps `proj + offset`, and:
    - the read flags `wall_lead_beyond_tolerance` with both times;
    - the server logs `clock_wall_lead` once per occurrence. The log line and the read tell the operator to restart the server once the host clock is confirmed correct (round 5 note). Read copy, for TW review:
      - EN: "The host clock is ahead of the server's time by more than the allowed difference. Giving or restoring access is paused. Confirm the host clock is correct, then restart the server."
      - KO: "호스트 시계가 서버 시간보다 허용 범위를 넘어 앞서 있습니다. 권한 부여·복구가 중지되었습니다. 호스트 시계가 올바른지 확인한 뒤 서버를 다시 시작하세요."
      - AR (draft): "ساعة المضيف متقدمة على وقت الخادم بأكثر من الفرق المسموح به. أُوقف منح الوصول أو استعادته مؤقتًا. تحقق من صحة ساعة المضيف، ثم أعد تشغيل الخادم."
    - **privilege increases are refused** (`dependency_unavailable`) until restart, through the same check as `ledgerBlocksIncrease`. That covers the HYB-298 increase list, the period increases of this ticket and `change_role` to administrator, as for any unrecorded state. Decreases continue.
    - At the next start the base is recomputed (the wall clock, or C2 when the ledger is ahead).
  - **Stripping.** Every value produced by `s.now()` and compared with a stored instant has its monotonic reading stripped (`Round(0)`), or is compared through `UnixNano`. A stored instant (ledger `at`, `role_until`, invitation and link expiries) never carries a monotonic reading, so mixed comparisons cannot silently use different clocks.
  - **Restoring the base** (round 2 N5). The latest ledger `at` is the maximum over **every decodable line**, verified or not, including lines after an integrity failure and a torn tail that decodes. A clock set back across a restart therefore cannot reopen an ended period.
    - With a missing ledger, or a newly created one (first start, or `-accept-new-decision-ledger`), the base is the wall clock. That is residual R15.
  - **Ledger time ahead of the wall clock: fail closed** (round 3 C2). If the latest ledger `at` is more than `-clock-skew-tolerance-hours` (default 24, bounds 1–720) ahead of the wall clock at start, `Start` is **refused** with an explicit error naming both times:

    "The decision ledger's latest time 2027-03-01T10:00:00Z is more than 24 hours ahead of this host's clock 2026-11-18T09:00:00Z. Correct the host clock, or restart with -clock-base=ledger or -clock-base=wall."
    - `-clock-base=ledger` takes the ledger time as the base.
    - `-clock-base=wall` takes the wall clock as the base. Ended periods could then reopen up to the difference, and the operator accepts that.
    - The choice is written into the next ledger record (a `clock_base` record with both times and the choice, `v: 2`) and shown in the read.
    - The flag is accepted only when the condition holds. Otherwise `Start` refuses it, so it cannot be left on (the HYB-298 single-use pattern).
    - **Later starts honour the latest `clock_base`** (round 4 D2). Ledger times written **before** the latest `clock_base` record are judged by its choice:
      - `wall`: they are ignored for the base;
      - `ledger`: the recorded offset (ledger time minus wall time at that start) is honoured, so they are compared with `wall + offset`.

      Times written after it are judged normally. A second restart therefore needs no flag, unless a new lead beyond the tolerance appears.
      - **Only a verified choice counts** (round 5 E2). A `clock_base` choice is honoured only if its line lies in the range that `decisionledger` verifies once it opens. If the choice the pre-scan used turns out to be unverified (after an integrity failure, or in a torn tail), `Start` is **refused** with an explicit error naming the line: "The clock base choice at decision ledger line 412 could not be verified. Correct the host clock, or restart with -clock-base=ledger or -clock-base=wall." (root choice: refuse).
    - **Ledger not usable when the flag is used** (round 4 D3). If the ledger is `isolated`, `stopped` or `full` at that start, the flag is still accepted and the base is set as chosen. Nothing can be written, so the read shows "clock base not recorded", and privilege increases are refused, as for any unrecorded state. The next start without a recorded choice meets the same condition and needs the flag again.
  - **One clock everywhere** (round 2 N6). `s.clock` **is** this server clock, for every time-dependent rule in apps/api and the modules it constructs:
    - role periods and the sweep;
    - step-up age (`stepup.go`);
    - directory expiries (pickups, invitations, links, grant holders; `directory.go`);
    - session idle and absolute expiry;
    - command rate-limit tokens;
    - the gate's and `modules/auth`'s validity checks (constructed with the same clock);
    - `modules/identity` and `modules/jobs` leases.

    Today the only wall-clock default is `cfg.Clock = time.Now` (`server.go:167`), which `clock.go` replaces.
  - **Source test** (widened after round 3 C3). A source test over `apps/api/server` and `apps/api/cmd/hb-api` fails on `time.Now`, `time.Since`, `time.Until`, `time.After`, `time.Tick`, `context.WithDeadline` and `context.WithTimeout` outside `clock.go`. Three uses are listed by name and allowed, because they are HTTP and process deadlines, not business rules:
    - the `http.Server` timeouts (`main.go:74-75` on main `d102fb4`);
    - the shutdown `context.WithTimeout` (`main.go:83`);
    - HTTP and TLS deadlines set on connections, if any are added.
  - **Diagnostics.** Every read returns `server_time` (the server clock) and `wall_time`. When they differ by more than one minute, the read flags `clock_behind` (or `clock_ahead`), and M01 shows "The host clock differs from the server's time; role periods use the server's time." `clock_behind` and `clock_ahead` are diagnostics only and never feed a rule. The one exception is `wall_lead_beyond_tolerance`: it drives the refusal of privilege increases until restart (A8; round 5 E1).
- The console takes a local date and time from a standard `<input type="datetime-local">` and converts it to UTC with the browser's time zone (`platform-independence.md`). The form says which zone is used, for example "Times are in your browser's time zone (UTC+09:00)".
- Display uses the viewer's locale with the UTC offset shown.
- "Ends soon" compares `role_until` with `server_time`, not the browser clock. It applies within `-role-period-warning-days`.
- **Clock steps** (restated after round 3 C1).
  - **During a run** (reworded after round 5 E1), forward wall-clock steps within the tolerance are absorbed by the catch-up offset, so ends come sooner. Backward steps are not followed, and neither are leads beyond the tolerance, which are flagged and refuse increases. Throughout, the server clock keeps advancing with monotonic elapsed time: role ends due during a step still fire, and rate limits, step-up age, session expiry and job leases keep advancing.
  - **Across a restart**, a backward step is absorbed by the ledger-time base. A forward step moves the base forward, which ends periods early (the restrictive direction).
  - A ledger time beyond the tolerance refuses `Start` (C2). Residual R5.

### What happens at the end

1. **At the instant of the end.** `Access` and `Recheck` refuse. The person's next request is answered `unauthenticated`, and apps/api drops that session (`commands.go:150-151`). Step-up evidence lives on the session (`session.go:89`), so it goes with it. Pending M03 requests where the person is proposer are superseded by HYB-298's basis (HYB-298 `plan.md:204-208`). Their invitations stop being acceptable, because the inviter no longer stands (`workspace.go:104-105`).
2. **The sweep** (locations widened after review M1). It always runs under the domain write lock:
   - at `Start`, before the listener;
   - at the top of `s.execute` (`commands.go:196`), before any command's checks, so every command, the trust hold classification (review m4) and the bulk handlers (review m6) see swept state;
   - in `signIn` (`signin.go:253`) and `redeem` (`signin.go:347`), before invitation acceptance;
   - on a 30-second ticker. 30 seconds is an engineering constant, not a policy value.

   Reads run under the read lock and do not sweep, but they use the effective predicate, so they never show an ended membership as active. For each membership `ExpireRoles` returns, the sweep:
   - ends the person's sessions and drops their pickups (`endSessions`, `users.go:181-187`);
   - writes a ledger decision `role_period_ended` (see Durability);
   - bumps the domain revision once per sweep that changed anything.

   If the ledger cannot record, the expiry still applies (a decrease is never refused), and the person's row reads "Role period ended · decision not confirmed".
3. **Restoring.** Only `identity.set_role_period` with a later end, or no end, restores a membership suspended for `role_period_ended`. That is a privilege increase.
   - `identity.reinstate_membership` is refused for it, and for a bulk-suspended membership whose end has passed (review m5), with copy that points to the period.
   - `identity.change_role` is refused for both (`invalid_stage`).
   - `identity.remove_membership` stays available.

**Why suspend, not demote or remove** (decision PD-1).
- Demoting an administrator to member at the end would grant a role nobody chose for that period.
- Removing would lose the membership record and need a new invitation.
- Suspension ends all access, keeps the record, and is reversible only through an explicit new period. Owner question Q1.

### Display

- **Users and Membership tabs.** A new "Role period" column:
  - "No end date";
  - "Until 30 Nov 2026, 18:00 (UTC+09:00)";
  - "Ends in 3 days", in words and weight;
  - "Ended 18 Nov 2026, 09:00", for a membership suspended for `role_period_ended`.
- **Detail.** A "Role period" fact with start and end.
- **Action.** "Change role period" in the Membership group.

## Part B. Bulk revocation

### Permission

- A new installation-kind action `revoke_access_in_bulk` (KO 대량 회수 권한, AR صلاحية الإبطال الجماعي), added to `modules/auth` and re-exported by `modules/gate`. `auth.InstallationAction` (HYB-297) becomes true for it, so:
  - the gate requires a human administrator (`gate.go:109` after HYB-297);
  - M03 cannot grant, list or revoke it (HYB-297 B1 guard).
- The four bulk commands (`add_targets`, `prepare`, `execute`, `discard`) use this action as their definition permission. The handler also requires a fresh `administer_installation` decision for the caller, because the effect is the same as M01 suspension. Both are needed; either one missing is `permission_denied`.
- `administer_installation` does not imply the bulk action, and the bulk action does not imply it.
- **Holders.**
  - The sample seed grants it to `sample-admin` and `sample-admin-2`, with the installation grants' 30-day terms (`seed.go:234-253`).
  - Bootstrap activation grants it to the bootstrap administrator (`signin.go:448-453`).
  - Nothing else grants it. This is the same enrolment block as HYB-297 R4, and it is declared (R6; Q5).
- **The read.** `GET /api/v1/identity/bulk-revocations` requires `administer_installation`, like the users read. It returns `can_prepare` for the caller.

### Selection model

- The console keeps a set of selected **user IDs**, never names or row positions (`usability.md`; `M01/spec.md:63`). The set survives "Load more", reloads and tab switches (`M01/spec.md:81`).
- A selected person who no longer appears after a reload is marked "No longer in the list" and still counted, so the server decides.
- **Page selection.** The header checkbox selects the loaded rows only. Its label says "Select the 100 loaded people", never "all".
- **Filter-wide selection** (`M01/spec.md:104`) is not offered. The selection bar says "Selecting every person in a search result is not available yet. Load more rows to add them." The manifest model has a `selection` value `explicit` only. HYB-294 adds `filter` with a fixed filter snapshot (see Dependencies).
- **Limit.** At most `-bulk-revocation-max-targets` people in one draft, default 500, bound 1–5,000. The console stops selection at the limit with a sentence.

### Transport: staging targets into a draft (review B1, root decision)

A single command cannot carry the selection: every field value is limited to 4,096 bytes (`server.go:125`; `execute.go:91`), about 186 real user IDs. The shared field limit is **not** changed, and no new route is added. Targets are staged instead:

1. **`bulk_revocation.add_targets`**, target `bulk-revocations`, fields `users` (text, newline-separated IDs, at most 4,096 bytes like every field) and `draft` (optional ID).
   - Without `draft`, it creates a new draft owned by the caller, with ID `bulk-draft-<n>`.
   - With `draft`, it adds to that draft. Only the owner may add. For anyone else the draft resolves `NotFound` in `Resolve` (N4), which is the same answer as an unknown draft, so a draft's existence is not revealed.
   - Each ID must pass `validID`. Format failures are `invalid_request`, field `users`, and nothing in the batch is added.
   - IDs are **not** resolved or classified here, so the batch answer reveals nothing about which IDs exist. Duplicates within and across batches are counted and kept once.
   - The draft size (after removing duplicates) may not pass `-bulk-revocation-max-targets`. A batch that would pass it is refused whole with `prerequisite_failed`, field `users`.
   - Answer: `completed`, state `draft`; fields `draft`, `count`, `duplicates`.
   - The console splits the selection into batches by **byte length** (not by count), so each value stays at most 4,096 bytes. It sends them in order, each with the previous result's revision as the expected revision.
2. **`bulk_revocation.prepare`**, target the draft ID, field `effect`. It freezes the draft: it resolves and classifies every ID (below), computes the digest, and turns the draft into a prepared manifest under the same ID. A frozen draft accepts no more targets.
3. **`bulk_revocation.discard`**, target a draft or a prepared manifest. It drops it. Only a holder of the bulk action may discard. For a draft, only the owner may discard.
4. **Draft lifetime and limits.** A draft expires `-bulk-revocation-manifest-minutes` after its last addition. Drafts and prepared manifests together count against `-bulk-revocation-pending` per workspace, and against one open draft per caller (a second draft is `prerequisite_failed`).
   - **Expired drafts and manifests are removed lazily** (round 2 N8): at the top of every bulk command and in the bulk read's computation of limits. They then free their slots, so an abandoned draft never blocks its owner beyond its lifetime.
5. **Rate and step-up.** 5,000 targets at 186 per batch is 27 batches, inside the command burst of 60 (`server.go:127`). The maximum setting is bounded at 5,000 for this reason. Each batch is a mutating command, so HYB-293 step-up applies. One password confirmation covers the whole staging within its 15-minute window (`stepup.go:17`).
6. **HYB-294 must reuse this transport** for filter-wide selection: a fixed filter snapshot is resolved server-side into the same draft, so the manifest, digest and execution stay one path.

### Effect scope

The preparer chooses one effect (`M01/spec.md:75` "효력 범위"; `:54` "전체 정지와 한 조직 종료 분리"):

| `effect` | What it does | Reversal |
| --- | --- | --- |
| `suspend_membership` (default) | Suspends the person's membership in this workspace only, with `bulk_revocation`. Other workspaces are kept (`ui-design.md:450`). | `identity.reinstate_membership` (an increase) |
| `suspend_user` | Suspends the account in every workspace (`SuspendUser`) | `identity.reinstate_user` (an increase) |

Removal and departure are not bulk effects. Both are hard to reverse, and the spec's bulk action is a revocation of effect, not a deletion (decision PD-6; Q4).

### Prepare: the fixed manifest

`bulk_revocation.prepare` runs under the domain write lock, after the sweep at the top of `s.execute` (review m6). It never changes access, and every classification uses the **effective** membership state.

1. It resolves each draft ID against the installation users and the boot workspace membership.
2. It sorts each selected person into a target or an exclusion:

| Exclusion reason | When |
| --- | --- |
| `self` | The preparer. Rule 1 of `protect`. |
| `not_found` | Unknown, service user, departed, or not in this workspace. These cases are not told apart, so a tampered ID reveals nothing (`tenant-security.md`). |
| `no_membership` | `suspend_membership` and the person has no membership |
| `already_revoked` | The membership is not effective (suspended, or its period ended), or for `suspend_user` the account is not active |

3. **Anchor check over the whole manifest.** If suspending every target would leave no anchor administrator, preparation is refused with `prerequisite_failed`, field `draft`, and the draft stays unfrozen. The console shows the spec's `blocked` copy (`M01/spec.md:96`) and lists the selected administrators, so the user can deselect one. Silently dropping one administrator would choose for the user (decision PD-5).
4. **Role ceiling** (`M01/spec.md:75` "역할 상한"). The preparer must be allowed to change every target's role. An installation administrator may change both roles, so no target is excluded today. The check is written explicitly so a future narrower role cannot pass it.
5. **The manifest stores:**
   - ID `bulk-revocation-<n>` (the draft's number carried over), preparer (the draft owner), `prepared_at`, `expires_at = prepared_at + -bulk-revocation-manifest-minutes`;
   - effect, selection (`explicit`), and the domain revision at preparation;
   - per target: user ID, login, role, `role_until`, membership version, user epoch, user state, membership state, installation-administrator flag;
   - exclusions with reasons, and the duplicate count;
   - the **impact preview** (below);
   - the digest: SHA-256 under the label `bulk_revocation_v1` over the canonical encoding of effect, selection, sorted targets with their basis and sorted exclusions. It follows the approvals digest pattern (`modules/approvals/digest.go`).
6. **Equal pending manifest.** A preparation equal to a pending manifest of the same preparer, with the same effect and the same set, returns that manifest and discards the new draft (the authpolicy pattern, HYB-298 `plan.md:255`).
7. **Only the draft owner prepares.** Anyone else gets `permission_denied`.
8. **Answer:** `completed`, state `prepared`, fields `bulk`, `digest`, `targets`, `excluded`, `duplicates`.

**Impact preview** (`ui-design.md:450`, `:384`). Every count is computed at preparation and labelled with that time:
- people, and memberships by role (administrators, members);
- installation administrators, and holders of trust rights (HYB-297) and of the bulk action;
- open console sessions and outstanding secret pickups;
- API keys: "Not applicable: people hold no API keys" (HYB-298 C3);
- pending work:
  - pending requests proposed by these people in the stores that record a proposer by user ID: M02 trust changes, M03 access change requests, and the retention, recovery, installation, integration, relationship and release-evidence proposals. The implementation enumerates them, and the test fails if a registered proposal store is missing.
  - For each store, the preview says whether that store rechecks a revoked proposer at approval. "Rechecked" is confirmed for M03 (HYB-298 basis, `plan.md:204-208`) and M02 (HYB-297). Every other store reads "Approval recheck not confirmed" until it is verified at implementation (review m10).
  - Pending jobs: "Not applicable: no product job runs under a person's authority".
- service users owned by these people: count. They are **not** changed and are not promoted (`retention-access.md:63`; `ui-design.md:450`). Line: "Service users they own keep running. Review them in Service accounts."
- the last-administrator line: how many anchor administrators remain afterwards.
- workspaces outside the scope: for `suspend_membership`, "Other workspaces are not changed".

### Execute: deny first, then ledger, then cleanup

`bulk_revocation.execute`, target the manifest ID, field `digest` (required). It runs under the domain write lock, after the sweep at the top of `s.execute`. Every check below reads the **effective** state (review m6). A target whose period ended between preparation and execution is therefore held as `changed_since_preview`, because the sweep bumped its version.

**Checks, in order.** The first failure answers, and a refusal changes no state.
1. The executor has made a fresh decision for the bulk action, and the handler for `administer_installation`. Step-up is by HYB-293.
2. The actor is human. Otherwise `prerequisite_failed`.
3. The manifest is `prepared` and not expired. An expired manifest is recorded `expired` lazily and answers `invalid_stage`. An executed, discarded or expired manifest is `invalid_stage`.
4. The digest equals the manifest's. Otherwise `invalid_request`, field `digest`.
5. **Per-target basis recheck.** A target whose membership version, user epoch, state or role changed since preparation is **held** with reason `changed_since_preview`, never processed silently (`M01/spec.md:104`). A target who is the executor is held with `self`. The executor need not be the preparer (Q3).
6. **The anchor check is repeated** over the non-held targets. A failure is `prerequisite_failed`, and the manifest stays `prepared`.

**Phase A: deny, before answering** (`retention-access.md:61`; HB-42).
- New identity batch functions in `modules/identity/bulk.go`:
  - `Workspace.SuspendMemberships(actor, workspace, []Target{user, version}, reason)`;
  - `Installation.SuspendUsers(op, []Target{user, epoch})`.
- Each runs under one identity lock, checks every version first, applies all, bumps the generation once, and runs one stranding check. It is all-or-nothing.
- Because step 5 rechecked under the same domain lock, a module conflict means a defect. The command then answers the mapped code (`mapIdentity`, `admin.go:80-94`), nothing is applied, and no job is created.
- After phase A, every target's next request is refused and every older Answer fails `Recheck`.

**Phase B: ledger, before answering.**
- **Job header first** (review M5). The first record of the job is a `bulk_revocation_job` record, committed alone with its own `Sync`. It holds:
  - the manifest ID and digest, the effect, the preparer and the executor;
  - the counts of targets, held and excluded, and the deny generation.

  If the header is not confirmed, no item record is written: every item reads `decision_unconfirmed`, and the job is not rebuilt after a restart.
- Then one decision record per non-held target (kinds under Durability). Records are written through a new `Ledger.CommitBatch` in chunks of 256 records: one `Write` and one `Sync` per chunk. 256 lines of at most 4 KiB is at most 1 MiB per write.
- An advancing receipt for a chunk gives its items `deny_confirmed`.
- A failed, unknown or refused chunk gives its items **and every later item** `decision_unconfirmed`, with HYB-298's reason (`commit_failed`, `commit_unknown`, `ledger_isolated`, `ledger_stopped` or `ledger_full`). The ledger enters `stopped` as HYB-298 defines.
- **Never refused, never fails open.** In `isolated`, `stopped` and `full` the deny of phase A is already applied. Nothing is written, and every item reads `decision_unconfirmed` (HYB-298 `plan.md:335`, `:507-510`).

**Answer.** `accepted` (202). Job ID = manifest ID. Fields `bulk`, `targets`, `held`, `excluded`. It is `accepted`, not `completed`, because cleanup is still running and the receipt is not completion (HB-34; HYB-298 `plan.md:638`).

**Phase C: cleanup in chunks, after the answer, on `modules/jobs`** (review m7).
- **Why `modules/jobs`.** It already provides what HB-34 asks of a chunked job: tasks, leases with expiry, a fencing token against a stale worker, a checkpoint per task, a retry budget with backoff, and a job state derived from task states (`modules/jobs/types.go:32-219`; `registry.go:132`, `:217`, `:323`, `:345`, `:372`). Writing a second chunk tracker in apps/api would duplicate it.
  - Its registry is in memory, like every pre-launch store, so a restart loses the job. That is acceptable here: a restart ends every session and pickup, which is all the cleanup does (see Replay).
  - It has no outbox or event inbox (R4).
- At execute, after phase B, apps/api creates a jobs job with ID = manifest ID, kind `bulk_revocation_cleanup`, and one task per chunk of 100 confirmed items. A task's input is the list of item indexes.
  - Tasks are `Repeatable`, because ending sessions and deleting pickups are idempotent.
  - Policy: lease 30 s, life 1 h, 3 attempts, backoff 1 s to 10 s. These are engineering constants.
- A single per-server worker goroutine leases tasks. For each task it:
  1. takes the domain write lock;
  2. checks its fencing token is still current;
  3. cleans the chunk;
  4. checkpoints the last item index it finished;
  5. completes the task, and releases the lock.

  A failure is reported to `Fail` with `permanent = false` until the budget is spent. The retry is the registry's own, for an operation proven idempotent; there is still no user retry button (`M01/spec.md:100`).
- Only items in `deny_confirmed` are cleaned. Items in `decision_unconfirmed` show "Not started: waits for a confirmed decision" with the planned scope (HYB-298 rule).
- **Each chunk rechecks** each target's current state (HB-03 "청크 경계 재검사"). A target reinstated in the meantime is still cleaned only for sessions opened **before** the deny time; sessions opened after a reinstatement are never ended. A session's open time is compared with the item's `deny_at`.
- **Cleanup items per person.** Each records `done`, `not_applicable` or `outside_scope`, with a reason:

| Item | Result |
| --- | --- |
| B1 Access decisions and Answers | done by the deny (identity generation; `Recheck`) |
| B2 Console sessions opened before the deny | done: `EndSession`, drop |
| B3 Step-up evidence | done with B2 (it lives on the session) |
| B4 Secret pickups issued by or for the person | done: deleted |
| B5 Outstanding set and reset links | not applicable: a link does not give access while the account or membership is suspended. Kept so a reinstatement can use it. |
| B6 API keys | not applicable: people hold none |
| B7 Service users owned by the person | outside scope: not changed, never promoted; review in M04 |
| B8 In-flight requests admitted before the deny | outside scope (done after a restart) |
| B9 Jobs started by the person | outside scope: no product job runs under a person's authority in pre-launch; the only `modules/jobs` use is this cleanup |
| B10 Copies already shown or downloaded | outside scope: cannot be recalled (`M01/spec.md:41`) |

- An item whose task has spent its retry budget is `failed`, with the reason. There is no retry button (`M01/spec.md:100`). A restart ends every session (R7).
- **State mapping** (round 2 N7). The bulk job's state combines the jobs job with the ledger outcome, in this order of precedence:

| Condition | Bulk `job_state` | Reason / screen copy |
| --- | --- | --- |
| Any item `decision_unconfirmed` | `waiting` | `decision_unconfirmed` |
| `jobs.Create` failed after phases A and B | `waiting` | `cleanup_not_started` |
| jobs `Queued` or `Running` | `running` | — |
| jobs `Waiting` | `waiting` | the jobs wait reason (`outcome_unknown`, `cancel_pending`) |
| jobs `Succeeded` | `succeeded` | — |
| jobs `Failed` (some task spent its budget) | `succeeded` | screen copy `partial` (`M01/spec.md:95`); the items of the failed tasks are `failed` |
| jobs `Cancelled` | not reachable: nothing cancels this job (below) | — |

  There is **no `failed` bulk job state**. The deny and the ledger are already done; a cleanup failure is a partial outcome, never a failed revocation.
- **`jobs.Create` fails after phases A and B** (N7). The deny and the records stand. The items stay `deny_confirmed` with cleanup "Not started", and the job is `waiting` with reason `cleanup_not_started`. The answer is still `accepted`. The log line is `bulk_cleanup_not_started`. A restart ends every session and pickup, which completes the cleanup by restart (R7).
- **Cleanup is not cancelled when the executor loses permission** (N7). Cleanup only removes access remnants of people already denied. Stopping it because the executor was suspended would leave sessions that `Access` already refuses. That is the restrictive direction of `retention-access.md:63` (pending work rechecks the actor's authority; a system cleanup of a decrease is not escalated to a stronger account, and needs none). The worker runs as `system:bulk-cleanup` and is never promoted.
- The worker stops with `Server.Close` (HYB-298 adds `Close`). Tests drain it through a test seam.

**Per-person revocation state.** It reuses HYB-298's state names and job mapping (`plan.md:318-346`): `received` → `deny_confirmed` → `cleanup_running` → `revocation_complete`, or `decision_unconfirmed`.

**Job counts and outcomes** (`M01/spec.md:95`, `:106`). The denominator is fixed at execution: `targets` = non-excluded people in the manifest.

| Outcome | Meaning |
| --- | --- |
| `succeeded` | Deny confirmed and cleanup complete |
| `held` | Not processed: changed since preview, or self |
| `failed` | Deny confirmed, a cleanup item failed |
| `unprocessed` | Not yet complete: deny applied but unconfirmed, or cleanup not yet run |

- The invariant `targets = succeeded + held + failed + unprocessed` holds at every read, and is property-tested.
- Excluded people are listed apart and are not in the denominator.

**Job state.**
- `running` while any confirmed item is not yet cleaned.
- `succeeded` when every non-held item is `succeeded` or `failed`, with no unconfirmed decision. With any failed item the screen copy is `partial` (`M01/spec.md:95`), never "complete".
- `waiting` with reason `decision_unconfirmed` while any item is unconfirmed.
- There is no `failed` job state: a deny failure creates no job.

**Manifest states.** `draft` → `prepared` → `executed`, or `draft` / `prepared` → `discarded` | `expired`. All end states are final.
- `bulk_revocation.discard`: the owner may discard a draft; any holder of the bulk action may discard a prepared manifest. It changes no access.
- Executed manifests are kept up to `-bulk-revocation-jobs-kept`. A job with any `decision_unconfirmed` or `unprocessed` item is never pruned (HYB-298 M4).

**Idempotency.**
- The executor's idempotency per installation, client, command and key applies (HYB-298 `plan.md:115`).
- A second execute with a different key finds the manifest `executed` and answers `invalid_stage`, so a manifest applies at most once. The transition to `executed` happens under the same lock as phase A.
- A replay with the same key returns the stored `accepted` result.

**Separation of duties.**
- No second-person approval: the spec has a confirmation, not an approval (`M01/spec.md:67`, `:75`), and the effect is a decrease. Owner question Q3 asks whether a threshold should require two people.
- What applies:
  - the bulk action held separately from `administer_installation`;
  - a fresh decision at execute;
  - step-up;
  - self-exclusion;
  - the anchor rule. Note the asymmetry (review m3): it guarantees *an* anchor remains, and that anchor may be the actor alone. One administrator can therefore leave themselves as the only anchor. This is the same power as today's single suspensions, and it is recorded in the threats and as owner question Q11;
  - a digest-bound confirmation;
  - the journal and the ledger naming the preparer and the executor.

**Confirmation** (`M01/spec.md:75` "대상/제외/효력 범위 확인"). The console's confirmation step shows:
- target and exclusion counts;
- the effect scope sentence;
- the outside-scope items;
- the digest (first 12 hex characters);
- a required reason;
- a required checkbox: "I checked the people, the exclusions and the effect scope".

The button reads "Revoke access for 37 people". The rationale entered at preparation is offered as the editable default for execution, because each command requires its own rationale.

## Durability and the HYB-298 decision ledger

**Decision (answers the coordinator's questions).**
- Role-period ends and bulk revocations are **decreases**. They are **never refused** because of ledger state. They are **recorded durably** in the decision ledger whenever it can record, and read `decision_unconfirmed` when it cannot.
- Bulk revocation does **not** go through an approval request. HYB-298's approval applies to M03 grant changes as `M03/spec.md:83` writes it. M01's bulk action has a confirmation only.

**Why increases must also be recorded.**
- Identity state is in memory and recreated by the seed at every start (HB-263 `plan.md:233`). Only seed users exist after a restart, with fixed IDs (`seed.go:22-27`).
- If decreases were recorded and their reversals were not, replay would re-suspend a person who had been reinstated.
- So `identity.reinstate_user`, `identity.reinstate_membership`, and every `role_period_set` that `periodIncrease` classifies as an increase, write records too.

**Increases: dry-run, intent, apply, confirm** (reviews M3 and M4). An increase runs these steps under the domain write lock, in this order:
1. **Dry-run** (M4). One instant `t` (the server clock read once at the start of the command, monotonic reading stripped) is passed to the `Check*` function **and** to the mutation (round 2 N2), so the dry-run and the apply judge the same time. Every precondition of the mutation is checked under the same locks the mutation will hold:
   - apps/api's own checks (self, anchor, cap, `periodIncrease`, the trust hold, `ledgerBlocksIncrease`);
   - the identity module's `Check*` function for the mutation (Part A).

   Any refusal answers its code and **writes nothing**. A refused reinstatement therefore leaves no ledger record.
2. **Intent record.** An `increase_intent` record carries the kind, target, version and value. It is committed with `Sync`. If it is refused, fails or is unknown, nothing is applied and the command answers `dependency_unavailable`. On `commit_unknown` the ledger is `stopped`, as HYB-298 defines.
3. **Apply** the mutation with the same instant `t`.
   - **If the mutation refuses** (it returns a refusal or conflict, which step 1 should have caught; round 2 N2), an `increase_aborted` record naming the intent's sequence is committed. The ledger **stays open**. The command answers the mutation's mapped code. Replay never applies an intent followed by an abort. If the abort record itself is not confirmed, the intent has no confirm either, so replay ignores it anyway.
   - **Only a non-refusal failure** (a panic or an unexpected error type, a genuine defect) stops the ledger for the rest of the process. The command answers `internal_error` with its operation ID and logs `increase_apply_failed`. No confirm record can follow, so the intent is never replayed.
4. **Confirm record.** An `increase_confirmed` record names the intent's sequence, and is committed with `Sync`.
   - If it is confirmed, the command answers `completed`.
   - If not, the increase **stays applied in this process**, because it was already applied. The command still answers `completed`, with field `durable` (round 2 N3):
     - `durable: "no"` when the confirm was refused or `commit_failed`: nothing was written, so the confirm cannot exist. Copy: "The change is in effect now, but its record could not be confirmed. It will not survive a restart."
     - `durable: "unknown"` when the confirm is `commit_unknown`: the line was written but its `Sync` failed, so it may be on disk. Copy: "The change is in effect now, but its record could not be confirmed. It may or may not survive a restart."
     - The ledger is `stopped` by either failure, so later increases are refused until restart.

**Replay of increases** (M3; Q8 default). Replay applies an increase **only when its `increase_confirmed` record follows its intent in the replayed range**. An intent without a confirm is ignored. Replay therefore never applies an increase it cannot prove took effect.

**Decreases: apply, then commit.** Decreases (bulk items, `role_period_ended`, and shortening `role_period_set`) are applied first and committed after. A failed or unknown commit reads `decision_unconfirmed`, and the decrease is never refused (HYB-298 rule). A decrease record found at replay is applied even if it was never confirmed in its own run: that is the restrictive direction (HYB-298 amendment 7).

**Decreases only, in degraded replay** (M3). When the ledger is replayed from a verified prefix while isolated, or "replayed from records not confirmed durable" (HYB-298 amendments 5 and 7):
- replay applies **decrease records only**;
- every increase record, confirmed or not, is ignored;
- the replay line says "Reinstatements and later end dates from the earlier run were not re-applied because the ledger could not be fully verified".

**New decision kinds** in `modules/decisionledger`. They share HYB-298's record fields, plus new optional fields: `role`, `until`, `intent_sequence` and the job header counts. Records hold no rationale or login (HYB-298 `plan.md:443`).

| Kind | Written when | Target / version | Other fields | Direction |
| --- | --- | --- | --- | --- |
| `bulk_revocation_job` | Phase B, first, alone (M5) | manifest ID / — | `request_digest`, `effect`, `requested_by` (preparer), `approved_by` (executor; reused field), `targets`, `held`, `excluded`, `deny_generation` | header |
| `suspend_membership` | Bulk item, effect `suspend_membership` | user / membership version before | `revocation` = `bulk-revocation-<n>:<i>`, `request` = manifest ID, `request_digest`, `requested_by` = executor, `deny_generation` | decrease, apply then commit |
| `suspend_user` | Bulk item, effect `suspend_user` | user / epoch before | as above | decrease |
| `role_period_ended` | Sweep | user / membership version before | `role`, `until`, `revocation` = `role-end-<n>`, `requested_by` = `system:role-period` | decrease |
| `role_period_set` | A shortening of an existing membership by `set_role_period` or `change_role` (never written by `create_user`, `invite` or acceptance; N1) | user / membership version before | `role`, `until` | decrease, apply then commit |
| `increase_intent` | Any increase: `reinstate_user`, `reinstate_membership`, a lengthening, removal or restoring `role_period_set` | user / version or epoch before | `increase` (`reinstate_user`, `reinstate_membership`, `role_period_set`), `role`, `until` (RFC 3339 or `none`) | increase step 2 |
| `increase_confirmed` | Increase step 4 | user / — | `intent_sequence` | increase step 4 |
| `clock_base` | The first record after a start with `-clock-base` (C2), stamped with the final server clock (D1) | — / — | `ledger_time`, `wall_time`, `choice` (`ledger` / `wall`), `offset` for `ledger` (D2) | diagnostic; never replayed as a decision |
| `increase_aborted` | Increase step 3, when the mutation refuses (N2) | user / — | `intent_sequence` | ends the intent; replay ignores it |

- Validation in `record.go` becomes per kind.
- **Format version 2** (review m11). Records of the new kinds are written with `v: 2`; HYB-298's kinds keep `v: 1`. The reader of this ticket accepts both.
  - **Root decision (a), round 2 N9:** HYB-299 implements `ErrFormatNewer` in `modules/decisionledger` itself. From this ticket on, a reader that meets a record whose `v` is greater than it supports refuses `Start` with the explicit error "the decision ledger was written by a newer version". It does not isolate.
  - A binary with only HYB-298 cannot know about this. On a `v: 2` ledger it isolates on the first `v: 2` record. That is accepted for pre-launch and recorded as R11. Downgrade stays unsupported.
- **Batch commit.** `CommitBatch([]Decision) ([]Receipt, error)` writes the chunk's lines in one `Write` and calls `Sync` once. The outcomes match `Commit`. A batch that would pass the file bound is not written, and the ledger becomes `full`.
- **Torn tail.** A crash between `Write` and `Sync` can leave several complete but unconfirmed lines. HYB-298's integrity check verifies and replays them.
  - For decreases that is the restrictive direction.
  - An `increase_intent` without its `increase_confirmed` is ignored (M3).

**Replay at `Start`.** It runs after HYB-298's grant and delegation replay and before the listener. Isolation and verified-prefix rules are HYB-298's (`plan.md:595-598`), with the decreases-only rule above.
1. For each user present after the seed, compute the final value of each aspect from the applicable records. Applicable records are decreases, plus confirmed increases outside degraded replay. The last applicable record in sequence order wins:
   - user state, from `suspend_user` and a confirmed `reinstate_user`;
   - membership state, from `suspend_membership`, `role_period_ended`, a confirmed `reinstate_membership` and a confirmed restoring `role_period_set`;
   - role end, from `role_period_set`, confirmed period increases and `role_period_ended`.
2. Apply them through `Installation.ApplyRecorded`, in this order:
   - **Role guard.** A role-end record whose `role` differs from the seed role after restart is not applied: "Not re-applied: role differs after restart". Role changes themselves are not recorded (R2).
   - **Suspensions are always re-applied** (Q7 default). That covers `suspend_user`, `suspend_membership` and `role_period_ended`.
   - **Role-end records are skipped deterministically to keep an anchor** (Q7 default). If, after the suspensions, applying every role-end record would leave no anchor administrator, the role-end records of administrators are skipped one at a time, in ascending user-ID order, until an anchor exists. Each skipped record reads "Not re-applied: kept as the administrator without an end date".
     - Example: a command-created open-ended administrator existed when two seed administrators were given ends. After a restart that administrator is gone.
   - **Operator-recovery state.** If no anchor exists even with every role-end record skipped (the suspensions alone leave none):
     - the server still starts, in a visible **operator-recovery state**;
     - it logs `administration_stranded` with the record sequences;
     - the sign-in page and every read show "No administrator can manage this installation. Operator recovery is needed.";
     - every mutating command is refused with `dependency_unavailable`.

     There is no dedicated exit: administrator recovery is an owner decision (HB-263 `plan.md:227`). **`-accept-new-decision-ledger` is a de facto exit** (round 2 N10), at the cost of every recorded revocation, because the replaced ledger is no longer replayed. The operator-recovery sentence names this cost. Recorded as R12 and owner question Q7.
   - Records whose user is not present ("Target not present after restart") are left, as HYB-298 does.
3. Run the role sweep with the server clock. Periods that ended while the server was down end now, and are recorded.
4. **Rebuild jobs** from the `bulk_revocation_job` header records (M5). Up to `-bulk-revocation-jobs-kept` jobs are rebuilt, with:
   - the header's manifest ID, digest, effect, preparer, executor and counts;
   - origin `ledger_replay` and `recorded_at`;
   - per item "Re-applied at start", "Target not present after restart" or "Not re-applied: …".

   - Item records without a confirmed header are not rebuilt into a job, but are still replayed as decreases.
   - **A confirmed header whose item chunks are unconfirmed** (round 2, M5 gap). The rebuilt job lists every target counted in the header. Targets with a replayed item record read as above. Targets without one read **"unprocessed: record not confirmed"**: the deny was applied in the earlier run, but its record is not in the ledger, so it was not re-applied. The counts invariant holds with them in `unprocessed`.
   - Held and excluded people appear only as counts, because only the header records them.
   - Cleanup items B2–B4 read done, because the restart ended every session and pickup.
   - Manifests from the earlier run, their exclusions and their previews are gone. The view says "Prepared lists from before the server started at … are not kept".
5. **Counters** for `bulk-draft-<n>`, `bulk-revocation-<n>` and `role-end-<n>` continue after the highest number in the ledger (HYB-298 N-L4).
6. The server clock's base is set from the maximum `at` over every decodable line, or `Start` is refused when it is beyond the tolerance (Part A; m1, N5, C1, C2).

**Ledger states and M01 commands.**

| Command | `ok` / `recovered` | `isolated`, `stopped`, `full` |
| --- | --- | --- |
| `bulk_revocation.add_targets`, `bulk_revocation.prepare`, `bulk_revocation.discard` | Allowed | Allowed (no access change) |
| `bulk_revocation.execute` | Deny, header, items, cleanup | Deny applied; items `decision_unconfirmed`; answers `accepted` |
| Any period write on an existing membership that `periodIncrease` classifies as a decrease | Apply, then commit | Applied; `decision_unconfirmed` shown on the row |
| Any period write on an existing membership that `periodIncrease` classifies as an increase | Dry-run, intent, apply, confirm |
| `create_user`, `invite` with `role_until` (N1) | Role rule only; no record | As member: succeeds, writes nothing. As administrator: `dependency_unavailable` (HYB-298 role rule) | `dependency_unavailable` (HYB-298 `ledgerBlocksIncrease`) |
| `identity.reinstate_user`, `identity.reinstate_membership` | Dry-run, intent, apply, confirm (new) | `dependency_unavailable` (unchanged from HYB-298) |
| Sweep | Apply, then commit | Applied; unconfirmed |
| Any mutating command in the operator-recovery state | Refused, `dependency_unavailable` | Same |

## Classification for the HYB-297 trust hold and HYB-298 increases

The HYB-297 rule (`HYB-297/plan.md:313-314`): allow what only reduces access, stops a harmful effect, withdraws a pending proposal or places a hold; refuse the rest; deny by default. The hold table and its registry-enumerating test gain these rows:

| Command | During the trust hold | Reason |
| --- | --- | --- |
| `bulk_revocation.execute` | allowed | reduces access |
| `bulk_revocation.add_targets`, `bulk_revocation.prepare` | allowed | stage or freeze a reduction and change no access (root classification by the rule; Q10 default: allowed) |
| `bulk_revocation.discard` | allowed | withdraws a pending draft or manifest |
| `identity.set_role_period` | allowed only when `periodIncrease` says decrease | a shortening is a decrease. Lengthening, removing the end and restoring are increases. |
| `role_until` on `change_role` | the existing role-based row applies, **and** the write must also be a decrease by `periodIncrease` | an end never makes a refused command allowed. A demotion that removes an end is refused during the hold (M2). |
| `role_until` on `create_user`, `invite` | the existing role rows apply unchanged (`HYB-297/plan.md:321`: both refused during the hold) | a new membership is classified by its role (N1) |

- **How the hold decides direction** (review m4). The hold check runs in `s.execute` under the domain write lock, **after** the sweep. It loads the target's **stored** membership (end, state, effectiveness) and calls `periodIncrease(stored, requested)`.
  - It never trusts the field label: `end: at` with a later date is an increase.
  - The same function, with the same inputs and under the same lock, is called again by the handler, so the hold decision and the applied write cannot disagree.
- HYB-298's `ledgerBlocksIncrease()` list (`plan.md:514-522`) gains every period write on an existing membership that `periodIncrease` classifies as an increase (`set_role_period`, `change_role`). `create_user` and `invite` keep HYB-298's role rule (N1).
- `periodIncrease(old, new)` is tested over every pair of no-end, earlier, equal and later ends, on effective, suspended and period-ended memberships, and with "no membership" (create, invite). An equal end on `set_role_period` is not a change: `invalid_stage`.
- The command count after HYB-298 (66) and HYB-297 (72, by its plan) becomes 77 with the five new IDs: `identity.set_role_period` and the four `bulk_revocation.*` commands. The count is rechecked at implementation from the registry; the plan's number is not a test oracle. The HYB-293 step-up coverage test and the HYB-297 hold test enumerate the registry, so they pick up the new IDs and fail until each has a row.

## Anchor administrator rule (restated after review m2)

**The property.** No accepted command, sweep, bulk execution or replay step takes the set of anchor administrators from **non-empty to empty**, evaluated at the server-clock time of that step.

It is **not** "an anchor exists at all times". Installation grants lapse 30 days after they were created (`directory.go:11`; holder expiry recorded at `seed.go:250`; `holds`, `directory.go:96`), and nothing renews them while the process runs (HB-263 residual, `plan.md:230`). After 30 days of uptime, every holder has lapsed, and so has every anchor. That is time-driven, outside this rule, and recorded as residual R13. A restart recreates the seed grants and the bootstrap grant path.

**Where it is enforced.**
- `protect` (`admin.go:153-169`) changes the "another administrator remains" test: the remaining administrator must now be an anchor. This applies to `suspend_user`, `depart_user`, a demotion, `suspend_membership` and `remove_membership`, as today.
- It also applies to:
  - every period write on an administrator;
  - bulk preparation and execution;
  - the sweep (defence; it cannot trigger, because anchors have no end);
  - replay (Q7 rule above).
- Consequence: a suspension that today succeeds because another administrator remains is refused when every other administrator has an end date. This is intended, and is tested and stated in the copy.
- The sample seed's administrators have no end, so the existing tests keep their results.

**The asymmetry** (review m3). The rule keeps *an* anchor, and the actor may be that anchor. A single administrator can leave themselves as the only anchor. Owner question Q11 asks whether the actor may be the only remaining anchor. The root default is yes, the same as today's single suspensions.

# API contract

**Common.**
- All commands are `Mutating`, with `Rationale` at most 256 bytes.
- The workspace comes from the session (`s.bootWorkspace`), never from the body.
- `Operator.Reference` is the Answer's user (HB-263 `plan.md:130`).
- Step-up comes from HYB-293.

| Command | Permission | Target | Fields | Result |
| --- | --- | --- | --- | --- |
| `identity.set_role_period` (new) | `administer_installation` | user ID | `end` enum `at` / `none` (required); `role_until` timestamp (required when `at`, refused when `none`) | `completed`, state `role_period_set` or `role_period_restored`; fields `role_until`, `ledger` (`confirmed` / `unconfirmed`), `durable` (`yes` / `no` / `unknown`, increases only) |
| `identity.create_user`, `identity.invite`, `identity.change_role` (changed) | unchanged | unchanged | new optional `role_until` timestamp; **omitted means no end** (M2) | unchanged. On `change_role` a period increase follows the increase steps. `create_user` and `invite` are classified by role and write no record (N1). |
| `identity.reinstate_user`, `identity.reinstate_membership` (changed behaviour) | unchanged | unchanged | unchanged | Dry-run, intent, apply, confirm. Field `durable`. Refused for a membership whose period ended (m5). |
| `bulk_revocation.add_targets` (new) | `revoke_access_in_bulk` + handler `administer_installation` | `bulk-revocations` | `users` text (newline-separated IDs, at most 4,096 bytes); `draft` ID (optional) | `completed`, state `draft`; fields `draft`, `count`, `duplicates` |
| `bulk_revocation.prepare` (new) | same | draft ID | `effect` enum `suspend_membership` / `suspend_user` | `completed`, state `prepared`; fields `bulk`, `digest`, `targets`, `excluded`, `duplicates` |
| `bulk_revocation.execute` (new) | same | manifest ID | `digest` text (required) | `accepted` (202), job ID = manifest ID; fields `bulk`, `targets`, `held`, `excluded` |
| `bulk_revocation.discard` (new) | same | draft or manifest ID | — | `completed`, state `discarded` |

**`Resolve`** follows the existing pattern (`admin.go:34-53`). The workspace must be `s.bootWorkspace`. The draft or manifest must exist in it; `bulk-revocations` resolves for `add_targets`. **Draft ownership is checked in `Resolve`** (round 2 N4): a draft whose owner is not the caller resolves `NotFound`. That happens before the revision and idempotency checks, so a non-owner's draft answers exactly like an unknown ID: same code, same status, no revision or idempotency difference. `Resolve` identifies the caller by looking up the session for `s.caller` with `s.sessions.get(s.caller)` (`session.go:100`) to get its login and user. `s.caller` is set under the domain lock as HB-263 defines (`plan.md:92-98`). A missing session resolves `NotFound`. The revision is `s.Revision()`. Anything else is `NotFound`, which the executor answers as `permission_denied`.

**Refusals**

| Condition | Code |
| --- | --- |
| Bad `role_until` (not UTC, not a whole minute, not in the future, beyond the maximum, beyond the setter's own end, or omitted / `none` while the setter has an end); `none` with `role_until`; `users` empty, malformed or over 4,096 bytes (executor); wrong `digest` | `invalid_request` with the field |
| Self-target; the anchor rule (single, period or bulk); non-human actor; draft over `-bulk-revocation-max-targets`; a second open draft; pending draft and manifest limit | `prerequisite_failed` |
| Equal end; no membership; reinstating, or changing the role of, a membership whose period ended (m5); draft not `draft`; manifest not `prepared`, or expired | `invalid_stage` |
| Unknown or other-workspace user or manifest | `permission_denied` (via `Resolve`) |
| Caller lacks the bulk action, or lacks `administer_installation` in the handler | `permission_denied` |
| A draft not owned by the caller | `permission_denied` via `Resolve` `NotFound`, before revision and idempotency checks (N4) |
| Increase while the ledger cannot record; increase whose intent commit failed or is unknown; any mutating command in the operator-recovery state; identity or roster unavailable | `dependency_unavailable` |
| Increase refused by the mutation after a confirmed intent (abort record; ledger stays open; N2) | the mutation's mapped code |
| Increase whose apply failed with a non-refusal error (a defect; the ledger is stopped, no confirm record) | `internal_error` with operation ID |
| Expected revision differs | `revision_conflict` (executor) |
| Same key, different payload | `idempotency_conflict` (executor) |

**Reads**

- **`GET /api/v1/identity/users`** (changed, additive):
  - page: `server_time` (server clock), `wall_time`, `clock_behind` / `clock_ahead` (diagnostics), `clock_base` (`default` | `ledger` | `wall`, with the times when chosen, and `recorded` true/false; C2, D3), `wall_lead_beyond_tolerance` (both times or null; A8), `recovery_state`, `settings[]` (HYB-298 shape: `name`, `value`, `unit`, `default`, `min`, `max`, `source`), `ledger_state`, `can_bulk_revoke`;
  - `membership`: `role_from`, `role_until` (or empty), `role_period` (`open` | `bounded` | `ends_soon` | `ended`), `suspended_for` (`""` | `role_period_ended` | `bulk_revocation`), `ledger` (`confirmed` | `unconfirmed` | `""`) for the last recorded decision on this membership;
  - row: `revocation` (`{bulk, state}` of the latest bulk item for the person, or null).
  - The key allowlist test is extended. No session data and no tokens.
- **`GET /api/v1/identity/bulk-revocations?after=&limit=`** (new; limit 1–100, default 50). It returns:
  - `revision`, `server_time`, `started_at`, `can_prepare`, `next`;
  - `jobs[]` (drafts of the caller, then prepared manifests, then jobs): `id`, `state` (`draft` | `prepared` | `executed` | `discarded` | `expired`), `origin`, `effect`, `selection`, `preparer`, `prepared_at`, `expires_at`, `executor`, `executed_at`, `deny_at`, `digest`;
  - `counts` (`targets`, `excluded`, `duplicates`, `succeeded`, `held`, `failed`, `unprocessed`), `job_state`, `job_reason`;
  - `ledger` (`first_sequence`, `last_sequence`, `watermark`, or `reason`), `cleanup` (`done`, `total`), `recorded_at`.
  - Pending manifests come first, then jobs, newest first.
- **`GET /api/v1/identity/bulk-revocations/{id}`** (new): the job row plus `impact` (the preview groups with `computed_at`), `exclusions` (counts by reason), and the outside-scope list.
- **`GET /api/v1/identity/bulk-revocations/{id}/items?after=&limit=&outcome=`** (new; limit 1–100). Each item has:
  - `user_id`, `login`, `role`, `role_until`;
  - `outcome` (`excluded` | `held` | `succeeded` | `failed` | `unprocessed`), `reason`;
  - `state` (revocation state), `receipt` (`sequence`, `watermark`, or `reason`);
  - `cleanup[]` (`item`, `state`, `reason`), `replay_note`.
  - Paged so that detail is never an unbounded payload (`domain-contracts.md:79`).

Every read needs the session, the CSRF header and `administer_installation` (the `admitted` path, `users.go:42-69`). It runs under `s.domain.RLock`. OpenAPI lists the new commands and routes.

**Journal and logs.**
- The common journal records each command.
- The bulk module keeps the business record.
- The ledger keeps the durable decisions.
- Log lines carry IDs, counts, codes and ledger states, never rationale text or logins: `bulk_revocation_executed`, `role_period_ended`, `role_expiry_blocked`.

# Installation settings (D-2)

They use the HYB-298 mechanism: `hb-api` flags parsed as canonical decimals, validated by `server.New`, shown read-only with their source (HYB-298 `plan.md:277-306`). They appear in an "Installation settings" disclosure on the M01 Revocation jobs tab and in the role period form.

| Setting | Flag | Default | Bounds |
| --- | --- | --- | --- |
| Longest role period, days | `-role-period-max-days` | 366 | 1–3660 |
| Clock skew tolerance, hours (N6, C2) | `-clock-skew-tolerance-hours` | 24 | 1–720 |
| Clock base after a refused start (C2) | `-clock-base` | unset | `ledger` or `wall`; accepted only when the ledger time is beyond the tolerance (single use) |
| "Ends soon" window, days | `-role-period-warning-days` | 14 | 1–90, and at most the longest period |
| People per bulk revocation (draft size) | `-bulk-revocation-max-targets` | 500 | 1–5000 (staging batches stay inside the command burst of 60, `server.go:127`) |
| Draft and prepared list lifetime, minutes | `-bulk-revocation-manifest-minutes` | 30 | 5–1440 |
| Drafts and prepared lists pending per workspace | `-bulk-revocation-pending` | 20 | 1–200 |
| Bulk jobs kept | `-bulk-revocation-jobs-kept` | 100 | 10–1000 |

Chunk sizes (ledger 256, cleanup 100), the `modules/jobs` policy (lease 30 s, life 1 h, 3 attempts, backoff 1–10 s) and the sweep interval (30 s) are engineering constants, not policy values. They are named in code, and their effect is measured in the IAM-07 test.

# Console

**Files.** `users/Live.tsx` is 550 lines. New pieces go into their own files, and `Live.tsx` only wires them:
- `Selection.tsx`: checkbox column cells, selection bar, limit and missing-row notes;
- `BulkPrepare.tsx`: effect choice, preparation, preview and confirmation steps;
- `RevocationJobs.tsx`: the tab, the job list, job detail, items paging and the settings disclosure;
- `RolePeriod.tsx`: the period cell, the detail fact and the change form;
- `bulk-model.ts` and `period-model.ts`, each with a test;
- `bulk-copy.ts` and `period-copy.ts`;
- `bulk.css`.

The shared `users/live.css` is **not** changed, so M03, M04, M05, O01, O02 and O03 are unaffected. If HYB-298 merges a `live.css` change, this plan rebases on it without editing it further.

**Service client.** A new `apps/console/src/service/identity-bulk.ts` (with test) holds the three bulk reads, the new types and the **staging helper** `stage(ids, revision)`. The helper splits IDs into batches by UTF-8 byte length (each value at most 4,096 bytes) and sends `bulk_revocation.add_targets` in order with each result's revision.
- **On `revision_conflict`** (round 2 N8), for example when another administrator signs in, it reloads the revision and resends the same batch with the same idempotency key. The draft count from the answer shows whether the batch had already been added: duplicates are kept once, so a resend never double-counts.
- **On `rate_limited`** it waits for the server's retry hint and resumes, with the copy "The server is limiting requests. Sending continues in a few seconds (300 of 500 sent)."
- On any other failure it stops, reports how many were staged, and offers to discard the draft. This keeps `admin.ts` (~763 lines after HYB-298) to the commands and users read. `admin.ts` gains the five command IDs, the new row fields and `accepted` result handling, which HYB-298 adds.

**Users and Membership tabs.**
- A checkbox column appears when `can_bulk_revoke` is true. Each checkbox's label is "Select <login>".
- The header checkbox selects the loaded rows.
- The selection bar is a region with a heading. It shows "37 selected" with the limit, "Clear", "Prepare revocation", and the filter-wide note.
- A new "Role period" column. The revocation effect appears as a sub-line in the status cell:
  - "Suspended by bulk revocation 3";
  - plus "· decision not confirmed" when unconfirmed;
  - or "Role period ended".

**Prepare and preview.**
1. Effect choice: radio buttons with a sentence each, and the reason.
2. "Preview impact" stages the selection (the progress line reads "Sending 300 of 500 people") and then runs `prepare`. A staging failure discards the draft and shows what failed. Nothing has been revoked at that point, and the copy says so.
3. The preview shows:
   - counts;
   - the impact groups in the order above;
   - exclusions grouped by reason, with paging through items;
   - the outside-scope list;
   - "Prepared at 14:02 · valid until 14:32".
4. Buttons: "Continue to confirmation", "Discard" and "Back to selection".
- Below 1280 px, each step is a separate step (`M01/spec.md:112`).

**Confirm.** As described under Confirmation. The HYB-293 step-up dialog may appear and resend once. On success, the console switches to the Revocation jobs tab with the new job selected and announces its state.

**Revocation jobs tab.**
- The list shows each job's state in words, effect, counts, job state and when.
- The detail shows:
  - the counts line with the fixed denominator, never a percentage when unconfirmed;
  - the ledger line (sequence range or reason);
  - cleanup progress "done of total";
  - items paged and filterable by outcome;
  - the per-item cleanup in three groups.
- Replayed jobs say "From an earlier server run".
- Expired and discarded manifests sit in a disclosure.
- In the operator-recovery state, the tab and every M01 view show the recovery sentence and no command controls.

**Role period form.**
- Radio "End date" / "No end date", **pre-filled from the stored end** (M2). "No end date" is disabled with a reason when the setter has an end. With "End date": date and time inputs, the zone line, server time, the longest allowed end and the setter's own end when it limits.
- `clock_behind` shows its sentence above the form.
- Impact sentence:
  - shortening: "Access through this role ends at … Their sessions end then.";
  - lengthening or restoring: "This extends access. It is refused while the decision ledger can't record."
- `create_user`, `invite` and `change_role` forms gain a "Role ends" control. On `change_role` it is pre-filled from the stored end, so that submitting unchanged never removes it (M2). On create and invite it starts as "No end date", or as the setter's own end when they have one.
- `durable: "no"` shows "The change is in effect now, but its record could not be confirmed. It will not survive a restart." `durable: "unknown"` shows "… It may or may not survive a restart." (N3)

**States and errors.**
- `prerequisite_failed` on the anchor rule shows the spec's blocked copy and lists the administrators involved. `invalid_stage` reloads and explains. `dependency_unavailable` uses HYB-298's extended `busy` copy.
- `partial`, the spec's copy (`M01/spec.md:95`), shows for a job with failed items. `error` (`:97`) shows for cleanup failures, and keeps "Access blocking already confirmed stays in place".
- State is in words and weight, never colour alone. There are no decorative borders or edge strips (`service-screen-style.md`). Focus moves to the affected heading, and a live region announces states.

**External references.** To be opened and recorded by root before implementation, with the observed pattern, what is adopted and what is rejected (`service-screen-style.md`). They have **not** been consulted for this plan:
- GitHub organization member removal (already cited, `HB-264/plan.md:69`);
- Google Workspace Admin bulk user actions;
- Microsoft Entra role assignment start and end, with expiring-soon views;
- Okta bulk deactivate confirmation.

## Copy (EN, KO, AR)

- KO uses the spec copy verbatim where it exists (`M01/spec.md:93-98`). EN and AR are drafts pending the technical-writing review and native review (R9).
- AR follows the HYB-298 glossary: الإبطال (إبطال) for revocation; دفتر القرارات for the decision ledger; القرار غير مؤكَّد for decision not confirmed.

| Key | EN | KO | AR (draft) |
| --- | --- | --- | --- |
| Role period | Role period | 역할 기간 | مدة الدور |
| No end date | No end date | 종료일 없음 | بلا تاريخ انتهاء |
| Ends soon | Ends in {n} days | {n}일 후 종료 | تنتهي بعد {n} يوم (counted plural via `shared/plural.ts`) |
| Role period ended | Role period ended | 역할 기간 종료 | انتهت مدة الدور |
| Change role period | Change role period | 역할 기간 변경 | تغيير مدة الدور |
| Bulk revocation | Bulk revocation | 대량 회수 | الإبطال الجماعي |
| Revocation jobs (tab) | Revocation jobs | 회수 작업 | مهام الإبطال |
| Prepare revocation | Prepare revocation | 회수 준비 | تحضير الإبطال |
| Held | Held: changed after preview | 보류: 미리보기 후 변경됨 | مُستبعَد مؤقتًا: تغيّر بعد المعاينة |
| Excluded | Excluded | 제외 | مستثنى |
| Partial | Some items were processed. Check completed, held, failed and unprocessed separately. This is not a full completion. | 일부 업무 항목만 처리되었습니다. 완료·보류·실패·미처리를 각각 확인하세요. 전체 완료가 아닙니다. | عولج جزء من العناصر فقط. راجع المكتمل والمستبعَد مؤقتًا والفاشل وغير المعالَج كلًّا على حدة. هذا ليس اكتمالًا كاملًا. |
| Blocked | The last-administrator or grant-scope check did not pass. Check the targets and the remaining access. | 최후 관리자 또는 부여 범위 검사를 통과하지 못했습니다. 대상과 대체 접근을 확인하세요. | لم يجتز التحقق من آخر مسؤول أو من نطاق المنح. راجع المستهدفين والوصول البديل. |
| Error | Some cleanup after revocation failed. Access blocking already confirmed stays in place. | 회수 후속 정리가 일부 실패했습니다. 이미 확정한 접근 차단은 유지됩니다. | أخفق جزء من التنظيف اللاحق للإبطال. يظل حظر الوصول المؤكَّد سابقًا قائمًا. |
| Filter-wide not available | Selecting every person in a search result is not available yet. Load more rows to add them. | 검색 결과 전체 선택은 아직 제공하지 않습니다. 더 불러온 뒤 선택하세요. | لا يتوفر بعد تحديد كل الأشخاص في نتيجة بحث. حمّل مزيدًا من الصفوف لإضافتهم. |

The "Held" AR term avoids معلّق, which means Suspended on this screen (`users/live-copy.ts:57`). It is listed for TW review.

## Console copy corrections in `users/live-copy.ts` (coordinator request, 2026-11-18)

This file is touched anyway for bulk revocation. The line numbers are from the HYB-298 worktree and are confirmed again against the base after the HYB-298 merge.

- **:230 and :237.** AR uses إلغاء ("مهام الإلغاء", "الإلغاء الجماعي") for access revocation. Change them to the الإبطال family, as HYB-298 standardised (glossary `:69-70` in that worktree): "مهام الإبطال" and "الإبطال الجماعي".
  - `revocationTitle` and `revocationBody` themselves are replaced by the working tab. Any surviving use, and every new string, uses الإبطال.
  - Invitation cancellation (`revoke_invitation`) keeps إلغاء, because it cancels an invitation rather than revoking access. The TW review confirms this split.
- **:343.** "لـ${login}" glues the preposition to an isolated login. Use the "الموجّهة إلى ${login}" style: "تُلغى الدعوة الموجّهة إلى ${login} …".
- **:193.** "الروابط المعلّقة" uses معلّق, which means Suspended on the same screen (`:57`). Use "الروابط التي لم تُستخدم بعد".
  - **Root addition for TW review:** `:343` also says "الدعوة المعلّقة" for a pending invitation, the same conflict. It becomes "الدعوة التي لم تُقبل بعد" in the same change.
- **Search.** Before QA, search all M01 copy for إلغاء used for access revocation and for معلّق used for "pending", and record what was found (`product-terminology.md`).

**Glossary rows** (new): role period, ends soon, role period ended, bulk revocation, revocation list (manifest), held, excluded, effect scope, and the bulk revocation right. Each row records its forbidden implications:
- "held" is not "failed" and not "revoked";
- "excluded" is not "protected";
- "decision not confirmed" is neither "revoked" nor "still allowed";
- "outside the controlled scope" is not "cleaned".

# Threat and abuse cases

| Case | Control | Test |
| --- | --- | --- |
| An administrator bulk-suspends every other administrator to take over | Self excluded; the anchor rule keeps an anchor administrator. **Asymmetry (m3):** that anchor may be the attacker alone. This is the same power as today's single suspensions. Journal and ledger name the actor. | Anchor tests; Q3 (two-person threshold); Q11 (actor as the only anchor) |
| Lock-out: every administrator given an end date, then expiry | The anchor rule at every setter, bulk and replay; open-ended anchors never expire; grant lapse modelled separately (m2, R13) | Property test: no step takes anchors from non-empty to empty |
| Manifest swapped or changed between preview and execution (TOCTOU) | Digest-bound execute; per-target basis recheck holds changed targets; no silent update | Basis change per field gives `held`; wrong digest gives `invalid_request` |
| Executing twice with different idempotency keys | `executed` transition under the same lock as the deny | Concurrent and serial double-execute: one job |
| Tampered or other-workspace IDs in `users` | `not_found` without telling cases apart; the workspace comes from the session | Cross-workspace and unknown IDs |
| Extending one's own role period | Self-target refused | Test |
| Granting a period beyond one's own | Ceiling check | Test |
| Two administrators extending each other indefinitely | Allowed within authority; recorded; bounded by `-role-period-max-days` per change | Documented, not prevented (Q2) |
| Expiring person acts before the end (invites admins, issues links) | Invitations need the inviter to stand at acceptance, through the time-aware predicate, so it also holds between the end and the sweep (`workspace.go:104-105`; M1); pending M03 requests supersede (HYB-298 basis through `Effective`); links give no access while suspended | Window tests (end ≤ now < sweep): acceptance refused, M03 approval superseded, anchor count excludes the person |
| Using the window between the end and the sweep | One effective predicate in every check; sweep at the top of `s.execute`, sign-in and redeem (M1) | Window tests above |
| Removing an end by omission (demotion form submitted without `role_until`) | Omitted means no end, classified as an increase by `periodIncrease`; refused when the setter has an end, during the hold and while the ledger cannot record; console pre-fills (M2) | Tests per command, including demotion of an administrator ended at T0 |
| Answer or session used past the end | `notAfter = min(absolute, until)`; `Access` refuses at the instant | Fixed-clock tests at the end and 1 ns before |
| Clock moved back or forward | Server clock = ledger-time base + monotonic elapsed time; wall steps ignored during a run; a base beyond the tolerance refuses start (C1, C2) | Wall-step tests with a separate monotonic source; restart tests; R5 |
| Forged future ledger time pins the clock | `Start` refused beyond the tolerance; operator chooses the base explicitly, and the choice is recorded (C2) | Start test with a line beyond the tolerance |
| Restart resurrects revoked access | Ledger replay before the listener; isolation rules inherited | Restart tests |
| Replay drift (reinstatement not recorded) | Increases recorded as intent, apply, confirm | Sequence suspend, reinstate, suspend other, restart |
| Replay grants an increase that never took effect | Replay applies an increase only with its confirm record; degraded replay applies decreases only (M3) | Property test: after any restart each target is at least as restrictive as its last confirmed in-process state |
| A refused or failed increase leaves a replayable record | Dry-run before the intent; apply failure stops the ledger, so no confirm (M4) | A refused reinstatement leaves no record; an injected apply failure leaves an unconfirmed intent that replay ignores |
| Rebuilt job misreports its counts | Job header record written first, rebuilt from it (M5) | Restart test compares counts with the header |
| Replay strands the installation | Suspensions always re-applied; role-end records skipped deterministically to keep an anchor; otherwise the operator-recovery state (Q7) | Constructed record sets for each branch |
| Ledger full or broken used to block containment | Decreases never refused; increases refused | State tests per command |
| Cleanup ends sessions opened after a reinstatement | Session open time compared with `deny_at` | Race test |
| Large selection used for denial of service | Field limit of 4,096 bytes per batch, draft size limit, one open draft per caller, pending limit, command rate limit; lock time measured at 100k | HTTP test at the maximum; IAM-07 test |
| Probing which IDs exist through staging | `add_targets` does not resolve IDs; classification happens at `prepare` and reports `not_found` without telling cases apart; a non-owner's draft answers like an unknown one | Tests |
| Step-up bypass for bulk | Every new command is mutating, so HYB-293 applies | Registry-enumerating step-up test |
| Trust hold blocks containment | Execute and shortening allowed in the hold | Hold table test |
| Preview leaks data | Only data M01 already shows to installation administrators; no tokens; key allowlists | Allowlist tests |
| Service user promoted on a person's revocation | Never changed; shown as outside scope | Test asserts unchanged service users |

# Test plan

Every check is recorded as PASS, FAIL or UNVERIFIED with the command and its output in `docs/features/HYB-299/verification.md`.

**modules/identity** (new `periods_test.go`, `bulk_test.go`, `period_property_test.go`; existing tests updated for the new `Invite` argument)
- `Access` and `Recheck` at `until − 1 ns`, `until` and `until + 1 ns`. `notAfter` is the minimum.
- `SetRolePeriod`: set, shorten, lengthen, clear, restore from `role_period_ended`, equal end, wrong version, non-administrator actor.
- `ExpireRoles`: only due memberships change; `blocked` when it would strand; the generation is bumped once.
- `ReinstateMembership` is refused for `role_period_ended`, and for a bulk-suspended membership whose `until` has passed (m5). `ChangeRoleUntil` is refused for both (m5).
- **Time-aware predicate (M1).** In the window `until ≤ now` before `ExpireRoles` runs:
  - `activeAdministrator` is false and the administrator count excludes the person;
  - `inviterStands` is false, so `Accept` of their invitation is refused;
  - `Subjects.Member` reports no membership;
  - `MemberInfo.Effective` is false;
  - the actor checks of `transition` and `Members` refuse them.
  - A source test fails on any comparison with `MembershipActive` outside `effective` and the transitions, including `CreateServiceUser` (`workspace.go:259`) and `onlySeat` (`registry.go:341`). A window test shows that a service user cannot be created with an owner whose period ended, before the sweep.
- `Check*` dry-runs agree with the mutations: for random states, `Check` returns exactly the error the mutation would return, and changes nothing (M4).
- `Invite` with `roleUntil`: the invitation `NotAfter` is capped; acceptance after the end is refused.
- Batch suspensions: all-or-nothing; version conflict; one stranding check; generation bumped once.
- `ApplyRecorded` for each aspect.
- **Property test** over random sequences of commands, clock advances and sweeps against a model:
  - no Answer is valid at or after its role end;
  - a swept membership is suspended for `role_period_ended`;
  - the identity module never has zero administrators;
  - with a wall clock that may step in either direction, no Answer becomes valid again once its end has been reached on the server clock (m1, C1).

**modules/bulkrevoke** (new, stdlib only; name checked free with `ls modules/bulkrevoke` before scaffolding)
- Drafts: creation, additions by the owner only, duplicates across batches, refusal of a whole batch over the size limit, one open draft per caller, expiry from the last addition, freezing at prepare, no additions after freezing.
- Manifest building, exclusions per reason, duplicates, digest stability and sensitivity to every field, equal-manifest return, limits, expiry at the boundary, discard of drafts and manifests.
- Execute: state checks in order, holds per basis field, the outcomes invariant, unconfirmed chunks, no pruning of unfinished jobs.
- **Property tests:**
  - `targets = succeeded + held + failed + unprocessed` after every step;
  - no target outside the manifest is affected;
  - a manifest executes at most once;
  - no item reaches `deny_confirmed` without an advancing receipt;
  - cleanup never runs for an unconfirmed item.
- Source test: stdlib imports only, no comments, at most 1,500 lines per file.

**modules/decisionledger**
- Each new kind's validation: required and forbidden fields; `increase_intent` with `none`; `increase_confirmed` naming an earlier intent; a `bulk_revocation_job` header.
- Format version 2: new kinds are written with `v: 2`; `v: 1` and `v: 2` both reopen. A record with `v: 3` gives `ErrFormatNewer` at `Start`, with the explicit message, and no isolation and no file change (N9). The HYB-298-only isolation on `v: 2` is documented, not tested here (R11).
- `CommitBatch`: receipts; write error, short write and `Sync` error give `stopped`; a bound pass gives `full`; torn tail inside a batch.
- The property test is extended: any sequence of batches reopens to the same watermark.
- A HYB-298 ledger reopens unchanged with this binary.

**apps/api/server** (fixed clock; new `periods_test.go`, `bulk_test.go`, `bulk_edges_test.go`, `identityledger_test.go`; `users_test.go` extended)
- **Permissions.** For each new command and read: a holder of both actions; holders of only one; a member; no session; bad CSRF. The read without `administer_installation`.
- **Role periods.**
  - Every refusal row.
  - The ceiling.
  - The anchor rule on single commands, including the changed outcome when the other administrator has an end.
  - Expiry ends the session at the instant, through a real cookie request. Step-up evidence is gone.
  - The sweep records `role_period_ended`.
  - Restore through a later end.
  - `reinstate_membership` refused for a period-ended membership.
- **Bulk.**
  - Prepare, then execute, with the sample seed: the member is refused on the next request; counts; the cleanup worker drained; per-item cleanup groups.
  - Exclusions; blocked by the anchor rule at prepare and at execute.
  - Held after a reinstate, a role change or a suspension between prepare and execute.
  - Wrong digest; expired manifest; discard; double execute; the idempotent replay result.
  - `suspend_user` effect.
  - The impact preview counts, including the enumerated proposal stores; the test fails when a registered proposal store is not enumerated.
  - Service users unchanged.
- **Ledger.**
  - Records per kind with no rationale or login (byte search).
  - In `isolated`, `stopped` and `full`: execute answers `accepted` with `decision_unconfirmed`; a shortening applies and reads unconfirmed; extension, restore and both reinstates answer `dependency_unavailable` and change nothing (read and revision unchanged).
  - Increases (both reinstates, and each period increase): a refused intent, `commit_failed` and `commit_unknown` apply nothing.
  - A refused or failed confirm leaves the increase applied with `durable: "no"`; a `commit_unknown` confirm gives `durable: "unknown"`. Both stop the ledger. After a restart, the first is never re-applied, and the second is re-applied only if the confirm line verifies (N3).
  - A mutation refusal after a confirmed intent writes `increase_aborted`, keeps the ledger open, and answers the mapped code; a restart ignores that intent (N2).
  - `Check*` and the mutation receive the same instant: a test advances the clock between them and asserts the outcome is judged at the first instant (N2).
  - **A refused reinstatement leaves no ledger record (M4)**: byte-compare the file before and after.
  - An injected apply failure after a confirmed intent answers `internal_error`, stops the ledger, and leaves an intent with no confirm, which a restart ignores.
  - The job header is written before any item, and alone with its own `Sync` (M5). A failed header leaves every item `decision_unconfirmed` and writes no item record.
- **Restart.** A new `Server` on the same data directory:
  - a bulk-suspended seed member stays suspended before `Start` returns;
  - a reinstated one stays active;
  - a role end on `sample-member` is re-applied, and ended if due while down;
  - the role guard note;
  - Q7 branches: suspensions always re-applied; role-end records skipped in ascending user-ID order until an anchor exists; the operator-recovery state when suspensions alone leave none (every mutating command refused, the recovery sentence in reads);
  - an intent without a confirm is ignored; a confirmed increase is applied (M3);
  - degraded replay (isolated prefix, or not confirmed durable) applies decreases only (M3);
  - jobs rebuilt from header records with the header's counts (M5); item records without a header are replayed but not rebuilt into a job;
  - the clock base is restored as the maximum `at` over every decodable line, verified or not, so a clock set back across the restart does not reopen an ended period (m1, N5); with a missing or new ledger the wall clock is used (R15);
  - a line whose `at` is more than the tolerance ahead of the wall clock refuses `Start` with the error naming both times; `-clock-base=ledger` and `-clock-base=wall` each start, set the base as chosen and write a `clock_base` record; either flag without the condition refuses `Start` (C2);
  - counters continue after the highest ledger numbers.
- **Hold and step-up.**
  - The HYB-297 hold test with the new rows, including `set_role_period` in both directions.
  - **The hold decides direction from the stored end (m4):** an `end: at` request with a later date is refused during the hold; one with an earlier date is allowed; the decision is taken after the sweep, so a period that ended just before the request is classified as a restore (increase).
  - The HYB-293 coverage test over the registry (77 IDs by this plan's count).
  - `ledgerBlocksIncrease` for every period increase on an existing membership (`set_role_period`, `change_role`).
  - **New memberships (N1):** in `stopped` (and in `isolated` and `full`), `create_user` as member with a `role_until` succeeds, and the ledger file is byte-identical before and after. As administrator it answers `dependency_unavailable`. `invite` behaves the same. An injected `createUser` partial failure still answers `internal_error` through `errPartial`, with no ledger record.
- **Omitted means no end (M2).** For `create_user`, `invite`, `change_role` and `set_role_period`:
  - omitted or `none` is stored as no end;
  - it is refused when the setter has an end;
  - on an existing membership (`change_role`, `set_role_period`) it is classified as an increase when the old period had an end;
  - **demotion of an administrator whose end passed at T0 to member is refused** (`invalid_stage`), both before and after the sweep;
  - a demotion without `role_until` of an administrator with a future end is an increase: refused during the hold and while the ledger cannot record.
- **Window tests (M1)**, with end ≤ now < next sweep:
  - invitation acceptance by an invitee of the ended person is refused;
  - an M03 approval whose proposer's period ended is superseded;
  - the anchor count excludes the ended administrator;
  - `Subjects.Member` and the users read show the membership as not effective;
  - the sweep runs at the top of `s.execute`, in `signIn` and in `redeem`, verified with a fixed clock and no ticker.
- **Staging over HTTP (B1)**, with real-length IDs (`user-` + 16 hex):
  - a batch of exactly 4,096 bytes is accepted, and 4,097 bytes is `invalid_request`, field `users`;
  - a draft filled to `-bulk-revocation-max-targets` = 5,000 through real HTTP batches, each at most 4,096 bytes and inside the rate limit, then `prepare` and `execute` succeed;
  - one more ID is refused whole;
  - a non-owner's add, prepare or discard of a draft answers byte-for-byte like an unknown ID, also with a stale expected revision and with a reused idempotency key (N4);
  - unknown IDs are accepted in staging and appear only as `not_found` exclusions after `prepare`;
  - an expired draft is removed at the owner's next bulk command and frees the pending slot and the one-draft-per-caller slot (N8).
- **Bulk job states (N7):** every row of the state mapping, including jobs `Failed` → `succeeded` with `partial` copy; an injected `jobs.Create` failure after phase B leaves items `deny_confirmed`, cleanup not started, and the job `waiting` with `cleanup_not_started`; suspending the executor during cleanup does not cancel or stop it; a confirmed header with an unconfirmed item chunk rebuilds those items as "unprocessed: record not confirmed".
- **Cleanup on `modules/jobs` (m7):** one task per chunk; a stale lease's fencing token is refused; an injected failure is retried within the budget and then reported as `failed`; the checkpoint survives a retried task.
- **Race** (`-race -count=40`):
  - concurrent reads during execute and cleanup;
  - a reinstatement between cleanup chunks; sessions opened after it survive;
  - the sweep ticker against commands;
  - concurrent execute of one manifest.
- **IAM-07 scale test** (skipped with `-short`): 100,000 synthetic members through manifest, deny, ledger and replay.
  - It asserts isolation (no other workspace changed), the deny before the first ledger write, and the counts invariant.
  - It records the domain-lock hold time and the number of `Sync` calls. The numbers are recorded, not pass thresholds, and are input for HYB-294.
- **Clock (C1).** `clock.go` takes two injected sources: a wall source and a monotonic elapsed source. Tests use real monotonic readings from `time.Now()` values for the elapsed source, and a scripted wall source. With a backward wall step of one hour and a forward step of one day during a run:
  - `s.now()` keeps advancing by elapsed time and never moves back or freezes;
  - command rate-limit tokens refill, step-up age grows, session idle and absolute expiry fire, and `modules/jobs` leases expire on elapsed time;
  - a role end due during the step fires at its time (sweep and `Access`);
  - every compared value has no monotonic reading (`t == t.Round(0)`), and stored instants compare by `UnixNano`.
- **Catch-up (A8):**
  - a monotonic source that pauses while the wall clock advances (host sleep within the tolerance): role ends due during the pause fire on the first read after resume, through the sweep and `Access`;
  - a backward wall step after a catch-up: no freeze, no regression, `offset` unchanged;
  - repeated forward steps within the tolerance do not accumulate `offset`;
  - a forward jump beyond the tolerance: `wall_lead_beyond_tolerance` flagged in the read and logged with the restart advice, not absorbed; every privilege increase answers `dependency_unavailable` and a decrease still succeeds;
  - the wall reading used to compute the lead has its monotonic reading stripped (`w == w.Round(0)`), so the lead is a wall-to-wall difference (round 5).
- **Pre-scan and base records (D1, D2, D3):**
  - the base is computed before `decisionledger` opens, and `recovery`, replayed and `clock_base` records written at start carry the final server clock;
  - after a `-clock-base=ledger` start, a second restart without the flag starts, and the earlier times are compared with the recorded offset;
  - after a `-clock-base=wall` start, the earlier ahead times are ignored for the base;
  - with the ledger `isolated`, `stopped` or `full`, the flag is accepted, the read shows "clock base not recorded", and increases are refused;
  - a `clock_base` line after the first failing line (or in a torn tail) that the pre-scan used refuses `Start` with the error naming that line, and changes no file (E2).
- **`change_role` (N1):** the combined condition is one `increaseAllowed` argument at `admin.go:332`; a member-to-member change that lengthens the end is refused while the ledger cannot record.
- **Source test (C3):** no `time.Now`, `time.Since`, `time.Until`, `time.After`, `time.Tick`, `context.WithDeadline` or `context.WithTimeout` outside `clock.go` in `apps/api/server` and `apps/api/cmd/hb-api`, except the listed HTTP and shutdown deadlines (`main.go:74-75`, `:83` on main `d102fb4`).
- **Property tests in apps/api** (new `identity_property_test.go`, injected wall and elapsed sources, injected ledger faults). They run random sequences of single commands, period writes, bulk staging and execution, sweeps, wall-clock steps (forward and back), grant lapses at 30 days, ledger faults (refused, failed, unknown, full) and restarts. After every step:
  - **Restrictiveness (M3):** after any restart, every target's user state, membership state and role end is at least as restrictive as its last *confirmed* in-process state. That is: suspended if it was suspended; an end no later than the last confirmed end; never active through an unconfirmed increase.
  - **Anchor (m2):** no accepted step takes the set of anchor administrators from non-empty to empty, with grant lapse modelled from `directory.go:11` and `seed.go:250`. A lapse alone may empty it; that is reported, not failed.
  - **No record from a refusal (M4):** the ledger length changes only on accepted steps.
  - **Job header (M5):** every rebuilt job's counts equal its header.
- OpenAPI and route registration; JSON key allowlists for the new reads.

**apps/api/cmd/hb-api**: each new flag's default, `start_flag`, out-of-bound, signed, hex, leading-zero and relation values (HYB-298 pattern).

**Coverage, mutation, builds**
- 100% statement coverage of changed Go code.
- Mutation testing of `modules/bulkrevoke`, the changed identity files, the decisionledger changes and the new server files, cleaning the session GOCACHE between batches.
- Before deleting code a mutant survives in, trace every reader.
- Cross builds for linux, darwin and windows; vet; gofmt.

**Console** (`bulk-model.test.ts`, `period-model.test.ts`, `identity-bulk.test.ts`, render tests)
- Selection kept across pages, reloads and tabs; limit; missing rows.
- Staging helper: batches never exceed 4,096 bytes (including multi-byte safety), are sent in order with chained revisions, resume after `revision_conflict` (reload, same batch, same key) and after `rate_limited` (wait, then resume, with copy in EN, KO and AR), and stop at any other failure with an exact staged count (N8).
- Role forms pre-fill the stored end; "No end date" is disabled when the setter has an end.
- Local time to UTC conversion across a DST change and for UTC+03:00, UTC+09:00 and UTC−05:00.
- "Ends soon" from `server_time`.
- Outcome and job state text for every state, including replayed jobs.
- No percentage while unconfirmed.
- Accepted-result parsing.
- `can_*` gating.
- AR plural helper use.
- Typecheck and build; package.json test list.

**Browser QA** on the production build against a live API with the sample seed (after HYB-297, three administrators), with two browser contexts, EN, KO and AR at 1440 and 390:
1. Role period: set an end three minutes ahead on `sample-member`; the column, detail and "Ends soon". The member's context is signed out at the end, and the row reads "Role period ended". Restore with a later end.
2. Bulk: select the member and one administrator, preview with groups and exclusions (self), confirm through step-up, then the job reaches `succeeded`. The member's context is signed out.
3. Held: prepare, change a target in the other context, then execute. The item is held, and the copy says to prepare again.
4. Blocked: with every other administrator given an end, a selection of the remaining anchors is blocked with the spec copy.
5. Ledger isolated (altered ledger with marker): execute shows `decision_unconfirmed`; extension is refused with HYB-298's copy.
6. Restart: the earlier-run job and the replay notes; the suspended member stays suspended.
6a. Staging of a selection larger than one batch (flag set so that at least two batches are needed), with the progress line.
7. Filter-wide note; selection limit with a flag set low; the settings disclosure with one flag set.
8. Keyboard order, focus return, live region announcements; computed borders show no decorative or coloured edges; full viewport width.

**Regressions.** Rerun the existing runners with recorded changes in expectations:
- M01 `qa264` (all four tabs; the revocation tab expectation changes);
- M02 (`qa297`);
- M03 (HYB-298 runner);
- M04 `qa269`, M05 `qa271`;
- O01 `qa273`, O02 `qa275`, O03 `qa281` (shared `live.css` and `live-copy.ts`);
- P01 `qa283full`, and the P02, P03, P04 and P05 runners.
- Go: the full `apps/api` and module suites, including HYB-298's and HYB-297's.

**Reviews:** security design review of this plan (D-5); `ui-designer` design and render comparison; technical-writing review of copy and glossary; independent change review with security focus; re-review of fixes.

# Dependencies and sequencing

- **HYB-298 must merge first.** HYB-299 needs:
  - the decision ledger (states, `Commit`, replay hook, isolation);
  - `ledgerBlocksIncrease`;
  - the settings mechanism;
  - the `accepted` result handling in `admin.ts`;
  - the revocation state vocabulary;
  - the `users/live-copy.ts` change.

  HYB-299 changes HYB-298's `modules/decisionledger/record.go`, `types.go`, `ledger.go` and `apps/api/server/ledger.go` (replay order). That is only possible once they are on main.
  - **Format version (m11, N9 decision a).** HYB-299 adds `ErrFormatNewer` to `modules/decisionledger` after HYB-298 merges. HYB-298 is not asked to change. A HYB-298-only binary on a `v: 2` ledger isolates; that is accepted pre-launch (R11).
- **HYB-297 (approved) is expected to merge after HYB-298, and HYB-299 implements after it.** HYB-299 needs:
  - `auth.InstallationAction` for the new action;
  - the hold table and its enumerating test, which gain five rows;
  - the third sample administrator used in QA.

  If HYB-297 has not merged when HYB-298 has, HYB-299 waits; this is the root default. The alternative, carrying the predicate itself, would create a conflicting second definition.
- **HYB-293** is merged. Its step-up rule covers the new commands with no change (`stepup.go:40`).
- **HYB-294 should start after HYB-299 merges.** Overlap:
  - the users read (`users.go`) and the M01 table and toolbar (`Live.tsx`);
  - the expiry filter (`M01/spec.md:61`), which needs `role_until`;
  - filter-wide selection, which extends this manifest with `selection: filter`, a fixed filter snapshot and server-side count (`M01/spec.md:104`). **HYB-294 must reuse the staging transport** (`bulk_revocation.add_targets` into a draft, then `prepare`). The filter snapshot is resolved server-side into the draft. It does not change the shared field limit and adds no separate target-list route (B1);
  - for selections beyond `-bulk-revocation-max-targets` (up to 100,000), HYB-294 must revisit holding the domain lock through phase B, using the IAM-07 measurements recorded here. A likely change is moving phase B after the answer, with items visible as `received`.

  This note should be added to HYB-294's description.

# Files to change (after the HYB-298 and HYB-297 rebase; sizes are estimates, none over 1,500 lines)

| File | Change | Lines now → after (est.) |
| --- | --- | --- |
| `modules/identity/types.go` | Membership and member fields (`Effective`), `SuspendReason`, reasons | 255 → ~295 |
| `modules/identity/registry.go` | `membership` fields; `effective` predicate used by `activeAdministrator`, `administrators`, `Subjects.Member`, `administrator` | 389 → ~410 |
| `modules/identity/workspace.go` | `Invite` role end, `Accept` start and end, `inviterStands` via `effective`, `ChangeRoleUntil`, `Members` fields, reinstate refusal | 498 → ~570 |
| `modules/identity/boundary.go` | `Access`, `Recheck` via `effective`; Answer `notAfter` | 264 → ~280 |
| `modules/identity/periods.go` (new) | `SetRolePeriod`, `ExpireRoles`, `ApplyRecorded` | ~220 |
| `modules/identity/checks.go` (new) | `Check*` dry-runs (M4) | ~160 |
| `modules/identity/bulk.go` (new) | Batch suspensions | ~170 |
| `modules/identity/*_test.go` | New `periods_test.go` ~550, `bulk_test.go` ~400, `period_property_test.go` ~300; `workspace_test.go` 732 → ~780 | — |
| `modules/bulkrevoke/` (new) | `types.go` ~200, `manifest.go` ~260, `job.go` ~280; tests ~650 + ~450, `property_test.go` ~280, `source_test.go` ~50 | — |
| `modules/decisionledger/types.go`, `record.go`, `ledger.go` | Kinds (header, intent, confirm), fields, per-kind validation, format version 2, `CommitBatch` | ~124 → ~165, ~176 → ~290, ~415 → ~510; new `batch_test.go` ~380 |
| `modules/auth/types.go`, `modules/gate/types.go` | New action and predicate membership | small |
| `apps/api/server/periods.go` (new) | `set_role_period`, ceiling, `periodIncrease`, sweep and ticker | ~300 |
| `apps/api/server/clock.go` (new) | Server clock: pre-scan base, monotonic elapsed time, forward-only catch-up, lead flag, injected sources, `-clock-base` handling (C1, C2, A8, D1–D3) | ~170 |
| `apps/api/server/increases.go` (new) | Dry-run, intent, apply, confirm sequence (M3, M4) | ~180 |
| `apps/api/server/bulkstage.go` (new) | Drafts and `add_targets` (B1) | ~200 |
| `apps/api/server/bulk.go` (new) | Prepare, execute, discard handlers; preview; job header | ~400 |
| `apps/api/server/bulkread.go` (new) | Three reads | ~260 |
| `apps/api/server/bulkcleanup.go` (new) | Cleanup worker on `modules/jobs` (m7) | ~200 |
| `apps/api/server/identityledger.go` (new) | Identity records, replay with the confirm, decreases-only and Q7 rules, operator-recovery state, job rebuild from headers | ~360 |
| `apps/api/server/admin.go` | Anchor rule in `protect` via `Effective`, `role_until` on three commands, reinstates through `increases.go`, definitions | ~427 → ~510 |
| `apps/api/server/commands.go`, `signin.go` | Sweep at the top of `s.execute` (`:196`), in `signIn` (`:253`) and `redeem` (`:347`); operator-recovery refusal | +~20 each |
| `apps/api/server/users.go` | Row and page fields | 212 → ~270 |
| `apps/api/server/ledger.go` (HYB-298) | Call identity replay in order | +~10 |
| `apps/api/server/settings.go` (HYB-298) | Six settings | ~121 → ~190 |
| `apps/api/server/trusthold.go` (HYB-297) | Five rows; direction from `periodIncrease` on the stored end (m4) | +~25 |
| `apps/api/server/accessrequests.go` (HYB-298) | Basis uses `MemberInfo.Effective` (M1) | +~5 |
| `apps/api/server/seed.go`, `signin.go`, `boundary.go`, `server.go`, `openapi.go` | Grants of the new action, routes, config, OpenAPI | small |
| `apps/api/cmd/hb-api/main.go`, `main_test.go` | Six flags | +~40, +~90 |
| `apps/api/server/*_test.go` | New `periods_test.go` ~700, `window_test.go` ~400, `increases_test.go` ~450, `bulkstage_test.go` ~400, `bulk_test.go` ~750, `bulk_edges_test.go` ~550, `identityledger_test.go` ~750, `identity_property_test.go` ~400, `bulk_scale_test.go` ~200; `users_test.go` ~743 → ~850 | — |
| `apps/console/src/service/admin.ts`, `admin.test.ts` | Command IDs, row fields | ~763 → ~800 |
| `apps/console/src/service/identity-bulk.ts` (new) + test | Bulk reads and the staging helper | ~280 + ~260 |
| `apps/console/src/modules/identity/pages/users/Live.tsx` | Wiring, columns, tab | 550 → ~640 |
| `users/Selection.tsx`, `BulkPrepare.tsx`, `RevocationJobs.tsx`, `RolePeriod.tsx` (new) | Views | ~160, ~330, ~380, ~260 |
| `users/bulk-model.ts`, `period-model.ts` (new) + tests | Models | ~200 + ~260, ~150 + ~220 |
| `users/bulk-copy.ts`, `period-copy.ts` (new) | Copy | ~480, ~220 |
| `users/live-copy.ts` | Column headers; AR corrections | 346 → ~360 |
| `users/bulk.css` (new) | Styles; `live.css` unchanged | ~220 |
| `apps/console/package.json` | Test list | +3 entries |
| `apps/console/tools/` QA runner for HYB-299 | Browser QA | ~600 |
| `docs/ui/terminology/product-glossary.md` | New rows | +9 rows |
| `docs/features/HYB-299/plan.md`, `verification.md` | Records | — |

If any file approaches 1,500 lines at implementation, it is split by responsibility before review (`code-structure.md`).

# Facts, assumptions and unknowns

**Facts** are the `file:line` citations above, checked against `07458be` or against the named worktree on 2026-11-18.

**Assumptions (not verified):**
- A1. Seed user IDs and their memberships are recreated identically at every start (`seed.go:22-27` names them). The bootstrap administrator's behaviour after a restart with `--seed=none` is to be checked at implementation.
- A2. HYB-297 merges with the predicate, hold table and third administrator as its approved plan states.
- A3. The host wall clock is kept by the operator (NTP). In-run steps are ignored (monotonic elapsed time). At start, the base comes from the wall clock or the ledger within the tolerance (R5, R15).
- A8 (restated after round 4). Go's monotonic clock may pause during host sleep on some platforms. Within the tolerance the forward-only catch-up absorbs the gap, so role ends fire on the first read after resume. **Residual only above the tolerance:** a gap or wall lead larger than `-clock-skew-tolerance-hours` is not absorbed. It is flagged and logged, and privilege increases are refused until restart. Ends then follow monotonic time until the restart sets a new base.
- A4. `<input type="datetime-local">` is available in the supported browsers. Unsupported browsers get a visible text field with the expected format (`platform-independence.md`).
- A5. No downgrade from a HYB-299 ledger to an earlier binary is needed in pre-launch. Binaries from this ticket on refuse a newer format; a HYB-298-only binary isolates (N9, R11).
- A6. The domain-lock hold time for 500 targets (header plus two item chunks, three `Sync` calls) is acceptable. Measured, not assumed, in the scale test.
- A7. `modules/jobs` (`registry.go:74-544`) behaves as its tests state for leases, fencing and retries. It has no production user yet, so this ticket is its first integration.

**Unknowns:**
- fsync latency on the QA host per chunk;
- the exact proposal stores present after HYB-297 (enumerated at implementation, test-enforced), and which of them recheck a revoked proposer at approval (m10);
- none left for the format version (decided in N9).


# Residuals

- R1. Filter-wide selection and selections above 5,000 people wait for HYB-294, which must reuse the staging transport.
- R2 (widened after round 3 C3). Single-person decreases (suspend, suspend membership, remove, depart, demotion) are not recorded in the ledger. They revert to the seed state on restart, as today. This also covers **re-inviting a seed user after a removal**: the new membership's `role_until` is not recorded (new memberships write no record, N1), so after a restart the seed membership returns with no end. A follow-up ticket is the Q6 default.
- R3. Durability inherits HYB-298's limits: a single local file with no independent copy (HYB-298 R1).
- R4. Cleanup runs on the in-memory `modules/jobs` registry. There is no transactional outbox or event inbox (HB-34). A restart ends the job, along with every session it would have ended.
- R5 (restated after round 4 A8, D2, D3). The server clock is a base plus monotonic elapsed time plus a forward-only catch-up offset. Wall-clock steps back are ignored; steps forward within the tolerance are absorbed, which ends periods sooner (restrictive). A wall lead beyond the tolerance is flagged and logged, and increases are refused until restart (A8). Across a restart, a ledger time beyond the tolerance refuses `Start` until the operator corrects the host clock or chooses `-clock-base=ledger` or `-clock-base=wall`. The choice is recorded and honoured at later starts (D2), unless the ledger cannot record it (D3, increases refused). `-clock-base=wall` can reopen periods by up to the difference, and the operator accepts that.
- R6. Only the seed and bootstrap grant the bulk action; there is no enrolment (as HYB-297 R4). Q5 default: a truthful enrolment block.
- R7. No user retry for failed cleanup items. `modules/jobs` retries within its budget, and a restart clears sessions.
- R8. Login realm column and the full spec column order are not built.
- R9. EN and AR copy and the new KO strings need technical-writing and native-language review. Screen acceptance (FIX) is the owner's.
- R10. Pending jobs and keys are "not applicable" because those features do not exist. When they do, the cleanup inventory must be extended. Approval recheck of revoked proposers is confirmed only for M02 and M03 (m10).
- R11 (root decision a, N9). HYB-299 implements `ErrFormatNewer`, so later binaries refuse a newer ledger format at start. A HYB-298-only binary on a `v: 2` ledger isolates instead of refusing to start. This is accepted for pre-launch.
- R12 (restated, N10). The operator-recovery state (Q7) has no dedicated exit command. Administrator recovery is an owner decision (HB-263 `plan.md:227`). **`-accept-new-decision-ledger` is a de facto exit**: it replaces the ledger, so nothing is replayed and the seed state returns. The cost is every recorded revocation, grant and delegation revocations included (HYB-298 R7). The owner question stays open (Q7).
- R13. Installation grants lapse after 30 days of uptime, and with them every anchor (`directory.go:11`, `seed.go:250`). This is the HB-263 renewal residual; the anchor rule cannot prevent it (m2).
- R15 (N5, C2). With a missing or newly created ledger, the base is the wall clock, so a clock set back before such a start is not detected. With a ledger, a base beyond the tolerance fails closed (C2); within the tolerance, a forward ledger time up to 24 h ahead is accepted as the base.
- R14 (restated, N3). An increase whose confirm record fails stays applied in this process. When the confirm was refused or `commit_failed` (`durable: "no"`), it is lost at restart. When the confirm was `commit_unknown` (`durable: "unknown"`), it may or may not survive a restart, depending on whether the line reached the disk. Losing it is the restrictive direction (M3).

# Open owner questions (with root defaults applied until the owner decides)

- Q1. At the end of a role period, suspend the membership, or demote an administrator to member? **Default: suspend.**
- Q2. Should an end date be mandatory for administrator roles, or for all roles? **Default: the end is optional, the cap is enforced on every write, and the maximum is 366 days.**
- Q3. May one administrator both prepare and execute a bulk revocation, or should a second administrator confirm above a size threshold? **Default: a single administrator.**
- Q4. Bulk effects: should removal or departure be offered besides the two suspensions? **Default: suspensions only.**
- Q5. Who holds the bulk revocation right, and how are others enrolled? **Default: seed and bootstrap only, with a truthful enrolment block.**
- Q6. Should the single-person M01 decreases be durable in the ledger? **Default: a follow-up ticket.**
- Q7. Replay that would leave no anchor. **Default:**
  - always re-apply suspensions;
  - skip role-end records deterministically (ascending user ID) to keep an anchor;
  - otherwise start in a visible operator-recovery state with every mutating command refused.

  Its exit is open; `-accept-new-decision-ledger` is a de facto exit at the cost of every recorded revocation (R12, N10).
- Q8. What happens to an increase whose ledger commit is unknown? **Default (M3): replay applies an increase only when its confirm record follows. Unconfirmed increases are ignored at restart.**
- Q9. Should a role be schedulable to start in the future? **Default: no scheduled start.**
- Q10. Are `bulk_revocation.add_targets` and `bulk_revocation.prepare` allowed during the trust hold? **Default: allowed** (they change no access).
- Q11 (new, review m3). May the actor be the only remaining anchor after a bulk revocation or a single suspension? **Default: yes**, the same as today's single suspensions. The alternative is to require at least one anchor other than the actor.

# Root decisions in this plan (provisional, owner-reviewable)

- PD-1. A role-period end suspends the membership with reason `role_period_ended`; only a new period restores it. `reinstate_membership` and `change_role` are refused for it.
- PD-2. The end is enforced exactly, within the clock tolerance, by one time-aware effective predicate in every membership check. The sweep makes it durable and visible, and runs at the top of `s.execute`, in sign-in and redeem, at start and on a ticker.
- PD-3. The anchor property: no accepted step takes the anchors from non-empty to empty.
- PD-4. The ceiling is the setter's own role end, not the 30-day installation grant. A setter with an end cannot write "no end".
- PD-5. A bulk selection that would strand administration is refused, not silently trimmed.
- PD-6. Bulk effects are suspensions only.
- PD-7. Bulk revocation has a digest-bound confirmation, not an approval request.
- PD-8. Increases run dry-run, intent, apply, confirm, and replay only confirmed increases. Decreases apply before they commit, and replay always.
- PD-9. HYB-299 implements after HYB-298 and HYB-297, and HYB-294 after HYB-299 on the same staging transport.
- PD-10. Targets are staged in batches of at most 4,096 bytes into a caller-owned draft. The shared field limit is unchanged, and no route is added (B1, root decision).
- PD-11. Bulk cleanup uses `modules/jobs`.
- PD-12. One server clock: a ledger-time base plus monotonic elapsed time plus a forward-only catch-up within the tolerance, used by every rule (C1, A8). A wall lead beyond the tolerance refuses increases until restart. A ledger time beyond the tolerance refuses start (C2), and the recorded choice is honoured later (D2).

# Review history

## Design review round 1 (2026-11-18)

Security design review by a separate read-only reviewer. **Result: FAIL** (1 blocker, 5 major, 11 minor). Root decisions were relayed by the coordinator on 2026-11-18. Every finding is folded into this revision, listed below. **This revision has not been re-reviewed.**

| Finding | Severity | Resolution | Sections |
| --- | --- | --- | --- |
| B1: `CommandLimits().Value = 4096` (`server.go:125`; `execute.go:91`) caps the `users` field at about 186 IDs | Blocker | Root decision: staging. `bulk_revocation.add_targets` adds batches of at most 4,096 bytes to a caller-owned draft; `prepare` freezes it and computes the digest; `discard` drops it. The shared limit is unchanged, and no route is added. `-bulk-revocation-max-targets` bounds the draft. HTTP test at the maximum with real-length IDs. HYB-294 must reuse the transport. | Current behaviour (M01 API); Transport: staging targets into a draft; Prepare; API contract; Settings; Console (service client, Prepare and preview); Threats; Test plan (Staging over HTTP, console staging helper); Dependencies (HYB-294); Files; PD-10; R1 |
| M1: the end was enforced only in `Access` and `Recheck` | Major | One time-aware `effective` predicate in `activeAdministrator`, the administrator count, `inviterStands` and `Accept`, `Subjects.Member`, actor checks, `MemberInfo.Effective` and the basis inputs. Sweep at the top of `s.execute`, in `signIn` and in `redeem`. Window tests for invitation acceptance, M03 approval and anchor counts. | Current behaviour (memberships); Terms; Part A Data; What happens at the end (sweep); Threats; Test plan (identity predicate, Window tests); Files |
| M2: omitted `role_until` semantics and the cap on all writes | Major | Omitted or `none` means no end. It is refused when the setter has an end, on `create_user`, `invite`, `change_role` and `set_role_period`; acceptance is held by `inviterStands`. One `periodIncrease(old, new)` classifies every write for the hold, `ledgerBlocksIncrease` and commit-before-apply. Tests include the demotion of an administrator ended at T0. | Who sets a role period; Classification; API contract; Console role forms; Threats; Test plan (Omitted means no end) |
| M3: replay could apply an increase that never took effect | Major | Increases write an intent, apply, then a confirm record. Replay applies an increase only when its confirm follows. Isolated-prefix and not-durable replay apply decreases only. Property test: restrictiveness after any restart. Q8 default. | Durability (increases, replay); Threats; Test plan (Ledger, Restart, Property tests in apps/api); R14; Q8; PD-8 |
| M4: a failed apply after commit left a replayable record | Major | Dry-run of every precondition under the same locks before the intent (`Check*` in identity). A post-commit apply failure is a defect: the ledger stops, there is no confirm record, and it is never replayed. Test: a refused reinstatement leaves no record. | Part A Data (`Check*`); Durability (increases); API contract (internal_error); Threats; Test plan; Files (`checks.go`, `increases.go`) |
| M5: jobs rebuilt without a header | Major | Phase B writes a `bulk_revocation_job` header first (manifest digest, effect, preparer, executor, targets, held, excluded), alone with its own `Sync`. Rebuilt jobs use it. | Execute Phase B; Durability (kinds, replay step 4); Threats; Test plan |
| m1: wall-clock comparison | Minor | (Superseded by round 3 C1.) Monotonic high-water time for end comparison and the sweep, restored from the last ledger record at start. R5 restated. | Timezone and clock; Replay step 6; API reads (`server_time`, `wall_time`, `clock_behind`); R5; A3; PD-12 |
| m2: the anchor property was overstated | Minor | Restated as "no accepted step takes anchors from non-empty to empty". Grant lapse (`directory.go:11`, `seed.go:250`) is modelled in the property test and recorded as R13. | Anchor administrator rule; Current behaviour (grants); Test plan (Property tests); R13; PD-3 |
| m3: the actor may remain the only anchor | Minor | Asymmetry recorded in Separation of duties and Threats; owner question Q11 | Execute (Separation of duties); Anchor rule; Threats; Q11 |
| m4: the hold must not trust the field label | Minor | The hold decides direction in `s.execute`, under the domain lock, after the sweep, from the stored end via `periodIncrease`. Test included. | Classification; Test plan (Hold and step-up) |
| m5: role change and reinstate on ended periods were unspecified | Minor | `change_role` is refused on a `role_period_ended` suspension. `reinstate_membership` is refused after a bulk suspension whose `until` has passed. | Part A Data; What happens at the end; API refusals; Test plan (identity) |
| m6: bulk did not sweep first | Minor | `prepare` and `execute` run after the sweep at the top of `s.execute` and use effective state | Prepare; Execute |
| m7: wrong fact, "no job runner" | Minor | Corrected: `modules/jobs` exists (`types.go:32-219`) and no app imports it. Phase C now uses it for chunked tasks with lease, fencing, checkpoint and retry budget. | Not in scope; Current behaviour; Cross-check HB-34; Phase C; cleanup item B9; Settings note; Test plan; R4, R7; A7; PD-11 |
| m8: citation offsets | Minor | `retention-access.md:61` (100k bulk, deny first), `:63` (pending jobs, no promotion to service accounts), `:65` (screen) | Cross-check; Impact preview; Phase A |
| m9: spec :52 status | Minor | Marked covered, by interpretation, with Q2 | Cross-check |
| m10: pending-work row | Minor | Partially covered until each store's approval recheck of a revoked proposer is confirmed (M02 and M03 confirmed) | Cross-check; Impact preview; R10; Unknowns |
| m11: ledger downgrade | Minor | New kinds are written as format version 2. Superseded by round 2 N9: HYB-299 implements `ErrFormatNewer` itself; a HYB-298-only binary isolates (R11). | Durability (format version 2); Dependencies; Test plan (decisionledger); A5; R11 |

## Design review round 2 (2026-11-18)

Same reviewer, security focus. **Result: FAIL** on one major (N1) and ten minors. All round 1 findings are resolved or reduced. Root decisions were relayed by the coordinator on 2026-11-18. Every finding is folded into this revision, listed below. **This revision has not been re-reviewed.**

| Finding | Severity | Resolution | Sections |
| --- | --- | --- | --- |
| N1: new memberships were classified by `periodIncrease` | Major | `create_user`, `invite` and acceptance keep HYB-298's role rule (`plan.md:520-522`, `:872`). As member they succeed in `isolated`, `stopped` and `full`. Their ends are validated but never recorded, because non-seed users vanish at restart. `periodIncrease` applies to existing memberships only (`set_role_period`, `change_role`). The `role_period_set` kind row is fixed. `createUser`'s multi-step `errPartial` path is not routed through the increase sequence. Test: in `stopped`, `create_user` as member succeeds and the ledger is byte-identical. | Who sets a role period (direction function; new memberships); Durability kinds table; Ledger states table; Classification; API contract; Test plan (Hold and step-up; Omitted means no end) |
| N2: dry-run and apply could judge different instants; a refused apply stopped the ledger | Minor | One instant `t` is passed to `Check*` and the mutation. A mutation refusal after a confirmed intent writes `increase_aborted`, keeps the ledger open and answers the mapped code. Only a non-refusal failure is a defect stop. | Increases step 1 and 3; kinds table; Refusals; Test plan (Ledger) |
| N3: confirm `commit_unknown` copy was too definite | Minor | `durable: "no"` (refused or `commit_failed`): "will not survive a restart". `durable: "unknown"` (`commit_unknown`): "may or may not survive a restart". R14 restated. | Increases step 4; API contract; Console; Test plan; R14 |
| N4: draft ownership was checked in the handler | Minor | Checked in `Resolve`, which returns `NotFound` before the revision and idempotency checks. A non-owner's draft answers exactly like an unknown ID. | Transport step 1; `Resolve`; Refusals; Test plan (Staging over HTTP) |
| N5: high-water restore from the last verified record only | Minor | Restored as the maximum `at` over every decodable line, verified or not. A missing or new ledger uses the wall clock (R15). | Timezone and clock; Replay step 6; Test plan (Restart); R15 |
| N6: other components used the wall clock; no bound on future times | Minor | (Clock part superseded by round 3 C1 and C2.) `s.clock` is the high-water clock everywhere: step-up, directory, pickups, sessions, gate, auth, identity, jobs. A source test forbids `time.Now` outside `clock.go`. A ledger `at` more than `-clock-skew-tolerance-hours` (default 24) ahead of the wall clock is ignored for the high-water time and flagged `ledger_time_ahead`. | Timezone and clock; Settings; API reads; Test plan (Restart); R5 |
| N7: `modules/jobs` mapping, `Create` failure, permission loss, header gap | Minor | Explicit mapping table: jobs `Failed` → bulk `succeeded` with `partial` copy, and no failed bulk job state. `jobs.Create` failure → items `deny_confirmed`, cleanup not started, job `waiting` with `cleanup_not_started`. Cleanup is not cancelled when the executor loses permission (`retention-access.md:63`). A confirmed header with unconfirmed item chunks rebuilds those items as "unprocessed: record not confirmed". | Phase C (state mapping); Replay step 4; Test plan (Bulk job states) |
| N8: expired drafts, conflicts and rate limits during staging | Minor | Expired drafts and manifests are removed lazily and free their slots. The staging helper reloads and resends on `revision_conflict` with the same key, waits and resumes on `rate_limited`, with copy. | Transport step 4; Console service client; Test plan (Staging over HTTP; Console) |
| N9: format version ownership | Minor | Root decision (a): HYB-299 implements `ErrFormatNewer` in `modules/decisionledger`. A HYB-298-only binary on a `v: 2` ledger isolates (accepted pre-launch). The former request to HYB-298 is withdrawn. | Durability (format version 2); Dependencies; Test plan (decisionledger); A5; Unknowns; R11; round 1 m11 row |
| N10: no stated exit from operator recovery | Minor | `-accept-new-decision-ledger` is stated as a de facto exit at the cost of every recorded revocation. The owner question stays open. | Replay step 2; R12; Q7 |
| Source test coverage | Minor (additional note) | The `MembershipActive` source test explicitly covers `CreateServiceUser` (`workspace.go:259`) and `onlySeat` (`registry.go:341`), both moved to `effective`. | Part A Data; Test plan (identity) |

## Design review round 3 (2026-11-18)

Same reviewer, security focus. **Result: FAIL** on C1 only. N1–N10 are resolved, apart from the clock part of N6. Root decisions were relayed by the coordinator on 2026-11-18. Every finding is folded into this revision, listed below. **This revision has not been re-reviewed.**

| Finding | Severity | Resolution | Sections |
| --- | --- | --- | --- |
| C1: the high-water clock froze after a backward wall step and pinned after a forward one | Major | The server clock is now `base + time.Since(startMono)`, with `base` = the maximum of the wall clock at start and the latest ledger `at` within the tolerance. It never moves back, never freezes, and ignores wall steps during a run. `wall_time` and `clock_behind` are diagnostics only. Every value compared with stored instants has its monotonic reading stripped (`Round(0)`) or compares `UnixNano`. Tests inject a wall source separate from a monotonic elapsed source, run backward and forward wall steps with real monotonic readings, and assert that rate-limit tokens, step-up age, session expiry and job leases keep advancing and that role ends due during the step still fire. | Timezone and clock; Clock steps; Replay step 6; API reads; Threats; Test plan (Clock, Restart, Property tests); Files (`clock.go`); A3, A8; R5; PD-12 |
| C2: a ledger time far ahead could set the clock | Minor | Fail closed: `Start` is refused with an explicit error naming both times when the latest ledger `at` is more than the tolerance ahead of the wall clock. The operator restarts with `-clock-base=ledger` or `-clock-base=wall` (single use), and the choice is recorded in a `clock_base` ledger record and shown in the read. | Timezone and clock; Settings; API reads; Threats; Test plan (Restart); R5; R15 |
| C3: wording and test scope | Minor | R2 now covers re-inviting a seed user after removal (its `role_until` is not recorded and reverts at restart). `Resolve` names the `s.sessions.get(s.caller)` lookup (`session.go:100`). The source test forbids `time.Now`, `time.Since`, `time.Until`, `time.After`, `time.Tick`, `context.WithDeadline` and `context.WithTimeout` outside `clock.go`, with the HTTP and shutdown deadlines listed (`main.go:70-71`, `:79`). | New memberships; `Resolve`; Timezone and clock (source test); Test plan; R2 |
| N1 note | — | Increase on `change_role` = the new role is administrator (`admin.go:332` in the HYB-298 change) **or** the period gets longer; otherwise it is a decrease | Who sets a role period (`change_role` combines both tests) |

## Design review round 4 (2026-11-18)

Same reviewer, security focus. **Result: FAIL** on A8 only. C1, C3 and N1 are resolved; C2 is resolved with gaps D2 and D3. Root decisions were relayed by the coordinator on 2026-11-18. Every finding is folded into this revision, listed below. **This revision has not been re-reviewed.**

| Finding | Severity | Resolution | Sections |
| --- | --- | --- | --- |
| A8: a paused monotonic clock (host sleep) delayed role ends without bound | Major | Forward-only catch-up: an `offset` that only grows, set to `wall − proj` when `0 < lead ≤ tolerance`. It never moves back, never freezes after a backward step and does not accumulate; forward jumps within the tolerance are absorbed. A lead beyond the tolerance is not absorbed: it is flagged in the read, logged, and privilege increases are refused until restart (root decision). A8 is restated as the residual above the tolerance only. PD-2 now says "exact within the tolerance". | Timezone and clock (Now); Data (exactness); API reads; Test plan (Catch-up); A8; R5; PD-2; PD-12; Files |
| D1: base computed after the ledger opened; start-time records stamped inconsistently | Minor | A raw pre-scan of decodable lines computes the base before `decisionledger` opens. Every start-time record (`recovery`, replayed decisions, `clock_base`) is stamped with the final server clock. | Timezone and clock (Base); kinds table; Test plan |
| D2: a second restart after `-clock-base` | Minor | Ledger times before the latest `clock_base` record are judged by its choice: ignored for `wall`; compared with the recorded offset for `ledger`. A second restart needs no flag. Test included. | Timezone and clock (C2 block); kinds table (`offset`); Test plan; R5 |
| D3: flag used while the ledger cannot record | Minor | The flag is accepted. The read shows "clock base not recorded", and privilege increases are refused as for any unrecorded state. | Timezone and clock (C2 block); API reads; Test plan; R5 |
| N1 test placement | — | The `periodIncrease` term is added to the same `increaseAllowed` call at `admin.go:332` (main `d102fb4`), not as a separate check; test included | Who sets a role period; Test plan |
| `main.go` citations | — | Rechecked on main `d102fb4`: `http.Server` timeouts at `main.go:74-75`, shutdown `context.WithTimeout` at `:83` | Source test; Test plan |

## Design review round 5 (2026-11-18)

Same reviewer, security focus. **Result: PASS.** The notes below are folded in. Status set to approved after design review round 5.

| Item | Resolution | Sections |
| --- | --- | --- |
| E1: wording of in-run clock steps and the diagnostics line | Forward steps within the tolerance are absorbed. Only backward steps and leads beyond the tolerance are not followed. `wall_lead_beyond_tolerance` is stated as the one read flag that drives a rule (refusal of increases). | Timezone and clock (Diagnostics, Clock steps) |
| E2: an unverified `clock_base` choice | Honoured only within the range `decisionledger` verifies. If the pre-scan used an unverified choice, `Start` is refused with an error naming the line (root choice: refuse). Test included. | Timezone and clock (C2 block); Test plan |
| A8 test | Asserts the wall reading used for the lead has its monotonic reading stripped | Test plan (Catch-up) |
| Note: operator advice | The `clock_wall_lead` log line and the read tell the operator to restart once the host clock is confirmed correct; EN, KO and AR (draft) copy added for TW review | Timezone and clock (lead beyond the tolerance) |

## Root amendment after the visual spec (2026-11-18)

- **Targets of a prepared manifest.** The spec requires confirming the targets before execution (`M01/spec.md:75`, "대상 확인").
  - A prepared manifest exposes its frozen targets through a paged read.
  - The read is limited to the owner of the manifest, and a non-owner gets the same answer as for an unknown ID.
  - It lists each target's user ID and login and its classification at prepare time.
  - The preview and the confirmation show the first page with the total, and link to the next pages.
  - This is an additive read field. It needs no new command.
- **Staging failure.** The draft is kept, and the console offers resume (idempotent) or discard.

## Root amendments after the implementation reviews (2026-11-18)

These supersede the matching text above, including lines 250, 468, 746, 946 and 1056-1057; details are in verification.md (deviations D1–D18 and "Review fixes").

1. **Pending member records (D6).** A replayed suspension or period end for a person whose membership does not exist yet (an invited seed member) waits in memory and is applied when the invitation is accepted, before access is evaluated; the replay note reads `applies_when_membership_returns`, then "re-applied".
2. **Clock base (security F1).** `-clock-base` without a lead is accepted only provisionally; start is refused with `ErrConfig` when the chosen clock-base line verifies, so the flag cannot be left on. The base stays `max(latest, wall)`.
3. **`change_role` direction (F2).** An increase is a promotion to administrator or a later or removed end; the same role with a shorter end is a `role_period_set` decrease, allowed during a trust hold and a degraded ledger.
4. **Prepared lists are the preparer's (F3, default answer to Q3).** Only the preparer can execute or discard a prepared list and read its people; anyone else gets the unknown-ID answer and sees counts only. The executor is never held as `self` because the preparer is excluded at prepare.
5. **Omitted end on `change_role` (F4).** An omitted `role_until` keeps the stored end; removing an end needs the optional `end` field set to `none`; `end=at` needs a time. The setter's end-date limit applies to the kept end as well. Create and invite keep "omitted means no end, subject to the setter's cap". This is a public-contract change.
6. **New reads.** `GET /identity/bulk-revocations/{id}/targets?kind=excluded` (owner only, prepared or later lists; other `kind` values answer `invalid_request`); job rows carry `staged`; the jobs list pages with "Show more lists".
7. **Open-draft cause.** The console reads the open draft or the pending limit from the jobs page and shows only the applying cause.

