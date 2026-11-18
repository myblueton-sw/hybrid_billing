---
type: ImplementationPlan
title: Step-up reauthentication for every mutating administration command, with holder-only password re-entry and a console reauthentication dialog
description: Fix ticket for finding H2 of the HYB-291 planning cross-check. The API refuses every registered mutating command, and the M05 session-ending endpoint, when the session's last password proof is older than the step-up window. A session-holder password re-entry endpoint refreshes that proof and cannot be blocked by third parties. Every console screen that posts commands opens a reauthentication dialog in EN, KO and AR and resends the same frozen request once.
status: stable
tags: [HYB-293, security, identity, step-up, reauthentication, IAM-08]
sources:
  - resource: ../../reviews/HYB-291-planning-cross-check/report.md
  - resource: ../../planning/v1.16/retention-access.md
  - resource: ../../ui/screens/M01/spec.md
  - resource: ../../ui/screens/M05/spec.md
  - resource: ../../architecture/security-boundaries/identity-session-permissions.md
---

# Scope and ownership

[HYB-293](https://linear.app/hybrid-billing/issue/HYB-293) fixes finding H2 of the [HYB-291 planning cross-check](../../reviews/HYB-291-planning-cross-check/report.md) (`report.md:50`, `:116`, `:144`). The finding: M01 and IAM-08 require reauthentication before sensitive actions such as suspending a user. Today step-up applies only to the policy commands. Every other command checks only the policy's maximum authentication age (`apps/api/server/commands.go:208-213`, `policy.go:82-90`).

Owner direction (2026-11-17): stay faithful to the planning documents. Where they leave a value undecided, root decided a provisional value on 2026-11-17 (see Decisions). Those values stay owner-reviewable.

**Scope.** Every command registered as mutating through `Server.Register` requires a recent authentication. That is 63 commands today, in ten registry prefixes: `identity`, `policy`, `access`, `service`, `relationship`, `contract`, `installation`, `integrations`, `recovery` and `retention`. The `release_evidence.*` commands join automatically when HYB-290 merges. The M05 session-ending endpoint also requires it.

Base: `origin/main` `3e36101`. Branch: `feat/HYB-293-step-up`. Machine: mac_mini.

**Ownership**
- Root (Claude Code) owns the plan, implementation, tests and browser QA.
- The `ui-designer` role (AI, read-only) specifies the reauthentication dialog and compares the renders.
- A separate read-only reviewer reviews the design and then the change, with a security focus: sessions, credentials, permissions and a public error contract.
- A separate read-only technical-writing review checks the new EN, KO and AR copy and the glossary row.

**Allowed files**

`modules/passwords`:
- `registry.go`: one new export, `VerifyHolder`;
- its tests (`registry_test.go`, `property_test.go`).

`apps/api/server`:
- `session.go`: the `authenticated` time and its atomic setter;
- `commands.go`: `access`, `command`, `execute` and `fresh`;
- `policy.go`: `window`, `policyPage` and `actorOf`;
- `self.go`: step-up on ending sessions, and `authenticated_at` in `/me`;
- `signin.go`: `login` in the sign-in response;
- `boundary.go`: the route, and `login` in the session body;
- `errors.go` and `openapi.go`: the new code and its next action;
- new files `stepup.go` (rule, window, refusal) and `reauth.go` (the endpoint), with tests.

`apps/console/src/service`:
- `admin.ts` and its test;
- `api.ts`;
- `context.ts`;
- `Service.tsx`;
- `copy.ts`;
- `service.css` (dialog-scoped rules only);
- new files `Reauthenticate.tsx`, `reauth-model.ts` and its test.

Every console file that builds `adminClient` switches to the step-up client:
- identity: `users`, `access`, `service-accounts`, `authentication` and `my-security` `Live.tsx`;
- `organizations/pages/{organizations,changes,contracts}/Live.tsx`;
- `administration/pages/{installation,integrations,recovery,retention}/Live.tsx`.

The command-posting components and failure copy maps get a `reauth_cancelled` mapping and copy:
- `users/live-copy.ts`;
- `organizations/pages/changes/live-copy.ts`;
- `organizations/pages/contracts/{LiveActions.tsx,live-model.ts}`;
- `administration/pages/installation/{LiveProposals.tsx,live-codes.ts,live-model.ts}`;
- `administration/pages/{integrations,recovery,retention}/{LiveCommand.tsx,live-model.ts}`;
- the identity `access` and `service-accounts` `Live.tsx` failure maps;
- the corresponding `live-model.test.ts` files.

M02 and M05 copy:
- `identity/pages/authentication/{live-copy.ts,live-model.ts,live-model.test.ts}`;
- `identity/pages/my-security/{live-copy.ts,Live.tsx}`.

Other files:
- `apps/console/package.json` (test list only);
- `docs/ui/terminology/product-glossary.md` (one new row);
- `docs/features/HYB-293/*`.

`modules/commands`, `modules/authpolicy` and `modules/identity` stay unchanged.

# Requirement trace

Statuses follow the HYB-291 meanings (`report.md:163-169`):
- **covered:** the whole criterion within pre-launch scope;
- **deferred:** not built and recorded;
- **not connected:** depends on something pre-launch does not connect.

Planning acceptance tests stay NOT_RUN; these statuses describe planned behaviour. Screen specs other than M02 are in the authorized local workspace, not yet tracked in Git, and their line numbers refer to those files.

| ID / source | Text (summary) | This ticket | Status |
| --- | --- | --- | --- |
| IAM-08 (`retention-access.md:117`; `acceptance-trace.json:1956-1962`, HB-42) | Only sessions meeting the per-method MFA evidence and reauthentication policy run sensitive commands | Step-up on every mutating command for password sessions. MFA rules fail closed without a prompt. | covered for password sessions; MFA not connected |
| M01 `spec.md:73` | Suspend user: last-administrator protection and **reauthentication** | `identity.suspend_user` requires step-up | covered |
| M01 `spec.md:67` | Every write checks expected revision, idempotency key and current permission | The resend keeps the same key, payload and expected revision; the refusal reserves nothing | covered (regression tests) |
| M01 `spec.md:114` | Dialogs trap focus and return it; status through a live region | Reauthentication dialog | covered |
| M05 `spec.md:47` | Current password / reauthentication just before a sensitive change | The dialog's password field; password change keeps its own current-password field | covered |
| M05 `spec.md:68` | Password change requires reauthentication | Already met: the current password is verified on every change (`self.go:196-292`) | covered (HB-270) |
| M05 `spec.md:69-71` | Link, unlink and MFA management require reauthentication | None of these exist in pre-launch (`HB-270/plan.md`, "Excluded") | not connected |
| M05 `spec.md:72` | End other sessions: own reauthentication | `POST /api/v1/me/sessions/end` requires step-up | covered |
| M05 `spec.md:89` | State `reauth-required`: authenticate again with the current account to continue | Dialog copy | covered |
| M05 `spec.md:100` | Mobile keeps reauthentication | Dialog at 390 px | covered |
| M05 `spec.md:119`, M03 `spec.md:165` | Reauthentication validity undecided | Root decision: 15 minutes, provisional | covered with a provisional value |
| M02 `spec.md:63`; `retention-access.md:47` | The policy revision holds MFA and reauthentication evidence for sensitive commands | A server default that a policy rule can only tighten. Only `policy.change` rules can be proposed through the API (`policy.go:255-262`). | deferred (configuration in M02) |
| M02 `spec.md:51`, `:113` | Unknown external status holds sensitive commands | No external identity | not connected |
| `retention-access.md:41` | SSO alone is not MFA; require reauthentication | No SSO | not connected |
| `retention-access.md:53` | Linking requires reauthentication | `identity.LinkIdentity` has its own 15-minute check (`modules/identity/boundary.go:198-208`); no API path uses it | not connected |
| IAM-05 (`retention-access.md:114`) | Suspension and permission loss reach sessions | Reauthentication re-runs `access`; it cannot revive revoked access | covered (regression) |
| IAM-06 (`retention-access.md:115`) | Password recovery and last-administrator protection are not a bypass | Reauthentication needs the current password. It never resets the lockout, raises permission or skips the gate. | covered within pre-launch |
| HB-42 (`requirements.json:3703`) | Session, MFA, recovery and immediate-block design | Session authentication time and step-up | covered (session); MFA not connected |
| HB-03 (`requirements.json:146`) | Separate permissions; author versus approver | Step-up only adds a condition; gate decisions are unchanged | covered (no change to separation) |
| QA-04 (`detailed-requirements.json:822`) | Idempotency key scope; same key with a different payload is a 409; stale revision refused | Byte-identical resend; a test pins no 409 after a refusal | covered (regression) |
| L01 `spec.md:34`, `:48`, `:65`, `:76` | MFA challenge at sign-in | No MFA factor | not connected |
| Identity boundary design (`identity-session-permissions.md:18`, `:24`, `:55`) | Sensitive finance and access commands need recent step-up evidence; the session keeps an authentication/step-up time; the step-up lifetime is an owner value | Every mutating finance and administration command is in scope. The session keeps `authenticated`. The window is a provisional root decision. | covered (pre-launch commands); value provisional |

**A design principle, not a trace claim.** AUD-DB-02 and AUD-QA-07 (`detailed-requirements.json:373`, `:935`) forbid endless reauthentication loops on S01 and S02. The console likewise resends at most once.

# Observed baseline

**Session times**
- An `apps/api` session has `created`, set at sign-in (`session.go:26-32`, `:76-86`).
- `created` drives:
  - the maximum authentication age in `access` (`commands.go:136-153`, `:208-213`);
  - the absolute-lifetime prune (`session.go:51-59`);
  - the M02 `act_until` (`policy.go:103-110`, `:124-128`);
  - the authpolicy authentication time (`policy.go:222-237`).
- Nothing refreshes `created`. M02 tells the user to sign out and sign in again (`authentication/live-copy.ts:84`, `:92-93`; `live-model.ts:35-39`).

**Step-up today**
- authpolicy judges a step-up rule only for the evaluated action: `step_up_required` when the age is strictly over the rule, then `step_up_mfa_required` (`modules/authpolicy/evaluate.go:48-58`).
- Rules accept any valid action ID, at most 64 per revision, with an age from 1 minute to the revision's maximum age (`changes.go:23-27`).
- apps/api passes `policy.change` only inside the policy commands (`registry.go:116-128`, `:205-208`). `access` passes the empty action (`commands.go:209-210`).
- The policy module's refusal reaches the client as `prerequisite_failed` (`policy.go:174-185`). The console cannot distinguish it (`apps/console/src/service/admin.ts:317-341`).

**The identity module's own clock.** `modules/identity` has a 15-minute reauthentication limit (`server.go:115`; HB-258 "Assembly"). It is checked only by `LinkIdentity` and `UnlinkIdentity`, against the identity session's opened time (`boundary.go:198-208`). That clock cannot be refreshed. Even after this ticket's reauthentication, it would still refuse a linking call through a session opened more than 15 minutes ago. No API path calls it, so the divergence is recorded, not fixed.

**Registry.** `Server.Register` records `s.mutating[id] = d.Mutating` (`server.go:294-302`). All 63 registered commands are mutating. Enumerated from a sample-seed server on this base:
- `access` (3);
- `contract` (8);
- `identity` (12);
- `installation` (3);
- `integrations` (15, `integrationcmds.go:398-416`);
- `policy` (3);
- `recovery` (4);
- `relationship` (3);
- `retention` (6);
- `service` (6).

There is no `organization` command prefix: the organizations screens post `relationship.*` and `contract.*`.

**Execution order** (`commands.go:165-206`; `modules/commands/execute.go:216-380`)
1. Cookie, then CSRF, then body shape.
2. `access`.
3. Domain write lock.
4. Executor:
   1. gate admission;
   2. rate token;
   3. journal reservation;
   4. shape;
   5. resolve;
   6. gate decision;
   7. idempotency lookup, where a stored result replays (`execute.go:350`);
   8. expected revision (`:356`);
   9. gate recheck;
   10. handler.

   Handler failure codes are a fixed set (`modules/commands/types.go:71-74`). `s.commands.Execute` has one caller, `execute` (`commands.go:201`).

**Password verification**
- `passwords.Verify` reads the record, derives outside the mutex and compares in constant time. Under the mutex it then:
  - returns `locked` while the per-login lock holds;
  - resets the failure count on a match;
  - otherwise counts the failure and locks at 5 (`modules/passwords/registry.go:114-150`).
- M05 password change (`self.go:196-292`) uses `Verify`, the per-session attempt map `s.failures` (5, `self.go:18`, `:169-194`), the global unauthenticated limiter `s.limit` and the derivation pool `derive`.

**Console.**
- `adminClient.command` sends `{request_id, target_id, idempotency_key, expected_revision, rationale, body}` (`admin.ts:392-399`). Screens hold an `Attempt` in a ref (`identity/pages/users/Live.tsx:107-111`).
- The console builds `adminClient` in twelve files. Commands are posted from nine components:
  - `users`, `access`, `service-accounts` and `authentication` `Live.tsx`;
  - `organizations/pages/changes/Live.tsx`;
  - `organizations/pages/contracts/LiveActions.tsx`;
  - `administration/pages/installation/LiveProposals.tsx`;
  - `administration/pages/{integrations,recovery,retention}/LiveCommand.tsx`.

  M05 posts `/me/password` and `/me/sessions/end` (`admin.ts:476-479`).
- Native modal dialogs already exist (`shared/ReviewPanel.tsx:206`, `modules/sources/pages/live-imports/UploadReview.tsx:20`).

# Decisions (root, 2026-11-17; provisional, owner-reviewable)

| Topic | Decision |
| --- | --- |
| Window (was O-1) | 15 minutes from the last password authentication, with no reset per action. It matches the existing identity setting (`server.go:115`). The specs leave it open (M05 `:119`, M03 `:165`, identity boundary `:55`). |
| Command list (was O-2, Q5) | Every mutating registered command, plus `POST /me/sessions/end`. One auditable rule, not a per-family judgement. |
| Policy rules (was O-3) | A policy step-up rule can only tighten the 15-minute default: the window is `min(default, rule)`. M02's `step_up_minutes = 0` reads as "default", and the screen says so. |
| Configuration in M02 (O-4) | Server default only in pre-launch; residual. |
| Revocations (was Q3) | No exemption. Incident containment also needs a recent authentication. |
| Token rotation (Q1) | No cookie or CSRF rotation in this ticket, to keep the byte-identical resend. Recorded as a residual. |
| Journal (Q2) | Step-up refusals and reauthentications are log lines without credentials. Journaling them is a residual tied to HYB-292. |
| Password change (Q4) | Does not refresh the authentication time (consistent with HB-270). |
| Third-party blocking (review H1) | Reauthentication must not be blockable by third parties: no global limiter token, and holder-only verification that ignores the per-login lock. |

# The step-up rule

**Commands.** `stepUp(command) = s.mutating[command] && !exempt[command]`, evaluated at runtime. The exempt set for commands is **empty**.
- The rule fails closed: anything registered later through `Server.Register` with `Mutating: true` requires step-up until someone exempts it explicitly, in reviewed code.
- An unregistered command ID is not mutating, so it reaches the executor and gets `unsupported_command`.

**Non-command POST routes.** These are classified in an enumerated table that a test checks against every `POST` route registered in `boundary.go`.

| Route | Step-up | Reason |
| --- | --- | --- |
| `POST /api/v1/commands/{command}` | by the rule above | — |
| `POST /api/v1/me/sessions/end` | yes | M05 `spec.md:72` |
| `POST /api/v1/me/password` | no | verifies the current password itself (M05 `:68`) |
| `POST /api/v1/me/reauthenticate` | no | it is the reauthentication |
| `POST /api/v1/auth/login` | no | no session yet |
| `POST /api/v1/auth/logout` | no | only reduces access |
| `POST /api/v1/auth/link` | no | unauthenticated link peek |
| `POST /api/v1/auth/redeem` | no | unauthenticated link redemption |

Reads, including the secret pickup `GET /api/v1/identity/links/{ref}`, are not step-up. The issuing command already passed step-up, and pickups are bound to the issuing session.

**Rule names in a policy revision.** These are authpolicy action IDs:
- `policy.change` applies to the three `policy.*` commands. This is the existing name, and the only one the API can propose (`policy.go:259-262`).
- `command.sensitive` applies to every other step-up command and to `sessions/end`. This is a neutral name, not tied to one family.

No API path can create a `command.sensitive` rule in pre-launch. The resolution honours one if a revision holds it, for example a revision approved directly through authpolicy in tests.

**Window**
- **Default.** `window(action) = min(D, rule.MaxAge)` when the session workspace's current revision has a rule for the action, otherwise `D`. D is 15 minutes. A policy can only tighten the window, never loosen it.
- **`policy.approve` (review M1, M-B).** The window is `min(D, current rule, candidate rule, candidate MaxAge)`: the `policy.change` rule of the current revision, the `policy.change` rule of the candidate revision and the candidate revision's maximum age, the same terms `window()` uses (`policy.go:82-90`). Terms that are absent are left out.
  - The dead-end case: a candidate rule tighter than the caller's freshness. The authpolicy check would answer `approver_locked_out`, a `prerequisite_failed` the dialog cannot fix. Instead it becomes a reauthentication prompt, after which approval succeeds.
- **Boundary.** The window is satisfied while `now − authenticated ≤ window`, the same strictness as `evaluate.go:53`.
- **Fail closed.** An `authenticated` time later than `now` fails closed. In practice it is unreachable, because `access` ends such sessions first (HB-265).
- **MFA (review M2).** If the matching rule requires MFA, under the current revision or for `policy.approve` the candidate, a password cannot satisfy it. The check then refuses with `prerequisite_failed` (422, `next_action: correct_request`) **without a prompt**, for every step-up command and `sessions/end`. The API cannot propose such a rule (`policy.go:259`), so this is unreachable through the API. Tests reach it with a revision approved directly through authpolicy, for both a policy and a non-policy command.

# Server design

## Authentication time

**New field.** `session` gains `authenticated time.Time`. `sessions.put` sets it to `created`.

**Atomic setter (review L3).** `sessions.reauthenticated(v, at) bool` updates the field inside `sessions.mu` only if the session is still present. It returns `false` otherwise, and the caller then answers 401. Only `put` and `reauthenticated` write the field.

**One authentication time.** Every authpolicy evaluation uses `authenticated`:
- `fresh`;
- `actorOf`;
- M02 `act_until`.

This matches the HB-255 meaning of the maximum age: the age of the authentication.

**Consequences.**
- Reauthentication also renews the maximum-age clock. The 12-hour absolute lifetime still bounds every session, through `created` and identity's own expiry.
- With `actorOf` passing `authenticated`, the policy module's own `policy.change` check agrees with the pre-check. The pre-check is never looser, because it applies `min(D, rule)`.

**Alternative (rejected).** The maximum age on `created` and step-up on `authenticated`. Rejected because authpolicy takes a single `At`.

**Unchanged:** `created`, the session list's "signed in" time (`self.go:110`), the prune, and M02's `signed_in_at`.

**New fields**
- M02 gains `authenticated_at`.
- `/me` gains `authenticated_at` (review L6).
- `GET /api/v1/session` and the sign-in response (`loginBody`, `signin.go:198-202`, `:236`) gain `login`, the caller's own login (review L4), so the dialog can offer a username hint to password managers.

**Password change.** It does not refresh `authenticated`.

## Step-up check and refusal

**Where.**
- In `execute`, after taking the domain write lock and before `s.commands.Execute`.
- In `endSelfSessions`, after `self` admits the caller under the lock.

**What it does.**
1. Re-read the session with `s.sessions.get(caller)` under the lock.
2. Resolve the window as above. For `policy.approve` it reads the candidate through `s.policy.Proposal(n)`; a target that does not parse or does not exist uses the current-revision window only.
3. Compare against `s.clock()`.

**Refusal: HTTP 403**

```
{"code":"reauthentication_required","retry":false,"next_action":"reauthenticate","request_id":"<body request_id>"}
```

- The response carries `Cache-Control: no-store` and the usual headers.
- It has no `fields`, no times and no target information. It is identical for any target and any role.
- 403, not 401, because the session stays valid.
- **Request ID (review M4).** The command refusal echoes the body's `request_id` exactly as command results do: the value if it is a valid ID, otherwise `commands.Marker` (`modules/commands/types.go:312-317`). The module's `marked` and `validID` are unexported, so apps/api has its own request ID check, and a test pins its parity with the module's ID rule. `sessions/end` echoes the `X-Request-Id` header like the other `/me` responses.

**Contract (review M4)**
- `boundaryCode` gains a `next` field (`errors.go:19-26`).
- `reauthentication_required` is listed with status 403 and next action `reauthenticate`.
- `Document()` appends `reauthenticate` to the `next_action` enum of the Error schema, uses each boundary code's own `next` in `x-errors` (`openapi.go:25-28`), and marks each step-up command's operation with `x-step-up: true`.
- A test reads the published document.

**What a refusal does not do**
- It does not reach the executor. It is therefore:
  - not journaled;
  - not counted against the command rate;
  - not reserved in the idempotency store;
  - not a domain change (the counter is unchanged).
- It writes one log line, `step_up_required`, with only the user ID and the command or route.

**Idempotency and expected revisions**
- **Fresh request.** Nothing is reserved, so the resend with the same key, payload and expected revision runs as a first execution.
- **Replay.** A replay of an already completed command is also refused until reauthentication. Then it returns the stored result with `replay: true`, matching M02 `spec.md:130`.
- **Counter.** Reauthentication success does not bump `s.counter`. Bumping it would turn every resend into `revision_conflict`.
- **Stale revision.** If another administrator changed the domain meanwhile, the resend gets `revision_conflict`, and the existing screen handling applies.

**Ordering (declared).** Step-up is checked before permission. A member with an old session is asked to reauthenticate, and the executor then answers `permission_denied`.
- Nothing about the target is revealed, and the command catalogue is public.
- Alternative (rejected): duplicating the gate decision before step-up.

**Alternatives for the check**

| Option | Why not chosen |
| --- | --- |
| A new code in `modules/commands`, checked in a handler wrapper | A module change. The handler runs after the replay, so a replay would skip step-up. Kept as the route if journaling of refusals becomes required (HYB-292). |
| Reuse `prerequisite_failed` | The console cannot tell it from business refusals. |
| A password field in every command body | The password would cross the command decoder, digest and executor paths. It also contradicts "recent". |

## Holder verification in `modules/passwords` (review H1)

**New export.** `VerifyHolder(k Key, password string) string` returns `ok` or `wrong`.
1. Under the mutex, read the record (or the dummy record for an unknown key) and the account pointer.
2. Derive outside the mutex.
3. Compare in constant time with `subtle.ConstantTimeCompare`.
4. Under the mutex again:
   - if the account was replaced or removed meanwhile, return `wrong`;
   - otherwise return the comparison result.

**It never reads or writes the lockout state.** It does not touch `failures`, `first` or `locked`. So:
- a lock set by others does not refuse the holder;
- the holder's wrong attempts neither extend nor create a lock;
- a correct password does not reset the count or the lock.

**Security argument**
- The endpoint is reachable only with a live session cookie and its CSRF token. That session was opened by proving the password, and it is still current under `access`: maximum age, absolute lifetime, membership and suspension.
- The per-login lock and the global limiter exist to bound guessing by parties who hold no session. Applying them here would let anyone who knows a login, or who floods sign-in, block a legitimate holder's sensitive work. That is what the review requires to be impossible.
- Guessing through this endpoint is bounded per session:
  - the 5th wrong attempt, counted together with M05 password-change attempts in `s.failures`, ends the session;
  - a new session needs the password at sign-in, where the per-login lock and the limiter still apply.
- Derivation stays bounded by the shared `derive` pool (4 at a time, context-aware wait).

**Unchanged.** `Verify` keeps its behaviour for sign-in and password change.

## Reauthentication endpoint: `POST /api/v1/me/reauthenticate`

**Admission.** As for M05 (`self.go:55-72`): session cookie, CSRF and `access`. The login comes only from the session record.

**Steps**
1. **Decode before the lock (review L2).** `decodeStrings(r.Body, "password")`, exactly one key with a size cap, as `changePassword` does (`self.go:198`).
2. Under the domain lock:
   - admit the caller;
   - reject a bad body (422, or 413);
   - reserve a per-session attempt in `s.failures`. At 5, end the session, bump `s.counter` as `changePassword` does, and answer 401.
   - read the login's `generation`.
3. **No global limiter token (review H1a, L8).** Run `VerifyHolder` through `derive`, outside the lock. If derivation is unavailable, answer 503 `dependency_unavailable` and give the attempt back.
4. Under the domain lock:
   - **Session liveness first (L-A).** If the session no longer exists, answer 401 `unauthenticated`, before the wrong-password branch, as `changePassword` does (`self.go:246-249`). No counter bump.
   - **Wrong password:** 422 `current_password_wrong`. When this was the 5th wrong attempt, end the session, bump `s.counter` and answer 401 `unauthenticated` instead.
   - **Access.** Run `access` again (suspension, membership and maximum age); on failure, answer with its code.
   - **Generation changed** (a concurrent password change in this session): 409 `revision_conflict`.
   - **Otherwise:**
     - `s.sessions.reauthenticated(v, now)`; if it returns false, answer 401;
     - clear this session's attempt count;
     - answer 204 `no-store`.

     No counter bump on success.
5. Log `reauthenticated` or `reauthentication_failed`, with the user ID and the result class only.

**There is no `password_locked` result**, because holder verification ignores the lock.

**What it never does**
- log, journal, echo or store the password;
- rotate the cookie or the CSRF token;
- touch other sessions;
- touch the per-login lockout.

**Constant time**
- `VerifyHolder` always derives, with the dummy record when needed, and compares in constant time.
- The endpoint has no early return that depends on the password.
- Correct, wrong and unknown-record paths take the same derivation.

# Console design

## Step-up client

`adminClient(csrf, fetch, stepUp?)` takes an optional `stepUp: () => Promise<boolean>`.

**In `send`, for a POST refused with `reauthentication_required`:**
- If `stepUp` is present:
  - await it;
  - on `true`, resend the byte-identical request once: same path, body string, `X-Request-Id`, CSRF token and idempotency key;
  - on `false`, throw `AdminError("reauth_cancelled")`.
- A second refusal on the resend throws `AdminError("reauth")`. There is no loop.
- Without `stepUp`, the refusal maps to `reauth`.
- A `prerequisite_failed` from the MFA fail-closed path maps to the existing `refused`.

**Hook.** `useAdminClient()` reads `csrf` and `stepUp` from `ServiceContext` and memoizes the client on `[csrf, stepUp]`. `stepUp` itself is a stable `useCallback` in `Service.tsx`.

**Where the hook is used.** All twelve files that build `adminClient` use it, so every command-posting screen and M05 get the dialog:
- identity M01–M05;
- organizations `changes` and `contracts`, and `organizations`;
- administration `installation`, `integrations`, `recovery` and `retention`.

**Copy for both failures.** Each failure map gives `reauth_cancelled` and `reauth` their own neutral copy (change review L1). The wording is in the dialog copy table below (rows `reauth_cancelled` and `reauth`).

The copy is neutral because the refused request may have been a replay of an earlier, completed attempt (review L5). The form stays as it was. A second submit opens the dialog again with the same attempt.

## Reauthentication dialog

**Ownership.** `Service.tsx` owns one dialog and exposes `stepUp`. Concurrent callers share one pending promise, which settles exactly once; a guard ignores later settle calls (review L7).

**Structure**
- A native `<dialog>` opened with `showModal()`. Opened last, it sits in the top layer above any screen modal that is already open, such as the review panel or the upload review.
- `aria-labelledby` the heading and `aria-describedby` the explanation.
- `dir` follows the locale.
- No coloured borders or accent strips.

**Copy (one user-facing term, "password confirmation", after the terminology review; M05 `spec.md:89`; neutral for replays; KO uses the ~해 주세요 register)**

| Item | EN | KO | AR (draft) |
| --- | --- | --- | --- |
| Heading | Confirm your password | 비밀번호 재확인 | تأكيد كلمة المرور |
| Body | To continue, enter the password for your current account again. | 계속하려면 현재 계정의 비밀번호를 다시 입력해 주세요. | للمتابعة، أدخل كلمة مرور حسابك الحالي مرة أخرى. |
| Account label | Sign-in email (shared sign-in wording) | 로그인 이메일 | البريد الإلكتروني لتسجيل الدخول |
| Password hint | After 5 wrong passwords in a row in this session, you will be signed out. | 이 세션에서 비밀번호를 연속 5번 틀리면 로그아웃됩니다. | إذا أدخلت كلمة مرور خاطئة 5 مرات متتالية في هذه الجلسة، فسيُسجَّل خروجك. |
| Empty | Enter your password. | 비밀번호를 입력해 주세요. | أدخل كلمة المرور. |
| Wrong | The password is not correct. | 비밀번호가 올바르지 않습니다. | كلمة المرور غير صحيحة. |
| Changed (409) | Your password was changed while it was being confirmed. Enter your current password and try again. | 확인하는 동안 비밀번호가 변경되었습니다. 현재 비밀번호를 입력해 다시 시도해 주세요. | تغيّرت كلمة مرورك أثناء تأكيدها. أدخل كلمة المرور الحالية وأعد المحاولة. |
| Busy or unknown | We could not confirm your password. Try again shortly. | 비밀번호를 확인하지 못했습니다. 잠시 후 다시 시도해 주세요. | تعذّر تأكيد كلمة المرور. أعد المحاولة بعد قليل. |
| `reauth_cancelled` | Your password was not confirmed, so this request was not completed. Try again to continue. | 비밀번호 확인이 완료되지 않아 이 요청은 처리되지 않았습니다. 다시 시도해 주세요. | لم تُؤكَّد كلمة المرور، لذا لم يكتمل هذا الطلب. أعد المحاولة للمتابعة. |
| `reauth` | Your password was confirmed, but this request was not completed. Try again to continue. | 비밀번호는 확인했지만 이 요청은 완료되지 않았습니다. 다시 시도해 주세요. | تم تأكيد كلمة المرور، لكن هذا الطلب لم يكتمل. أعد المحاولة للمتابعة. |

**Fields**
- A read-only `autocomplete="username"` field with the session's `login`, in `<bdi dir="ltr">`.
- A password field (`type="password"`, `autocomplete="current-password"`).
- The password hint above, linked with `aria-describedby`.
- Buttons: Confirm and Cancel.

**Password handling (as in HB-271)**
- The input has no `name`, is uncontrolled and is read through a ref at submit.
- It is cleared in `finally` after every response, and on close, cancel, unmount and locale change.
- The value never goes into React state, the URL, storage, notices, errors or the console.
- The form is `method="post"` with no `action`, and the submit handler calls `preventDefault` first.
- Submit is disabled and re-entry ignored while a request is pending.

**Focus and keys (review L7)**
- On open, focus moves to the password field, and Tab stays inside the modal.
- Escape and Cancel resolve `false`.
- **While a request is pending**, the `cancel` event is prevented and Cancel is disabled. The dialog cannot settle twice, and a late response is not lost.
- On close, focus returns to the recorded opener, even when the opener sits inside another modal.
- Errors use `aria-invalid` and `aria-describedby`.
- Each result is announced once. A successful confirmation is not announced by the dialog, because the screen announces the result of the resend. A wrong or empty password is announced through the dialog's polite region only when focus is already on the password field; otherwise focus moves to the field, which reads the linked error.

**Outcomes**

| Endpoint result | Dialog |
| --- | --- |
| 204 | Close, resolve `true`; the screen shows its normal result after the resend |
| `current_password_wrong` | Field error "The password is not correct." No remaining count. |
| `unauthenticated` | Close. The existing sign-in-again path with the neutral "Your session has ended. Sign in again." A 401 can mean the 5th wrong attempt, suspension, the maximum age or a session ended elsewhere, so the wording names no cause and makes no lockout claim (change review L2). |
| `revision_conflict` | The "Changed" copy above (the generation counter is per login and also moves on reset-link redemption, so the copy does not say "in this session"). |
| `busy`, `unavailable` or unknown | The "Busy or unknown" copy above. The dialog stays open. |

**M02**
- When `now > act_until`, the actions stay enabled.
- The stale line becomes "You'll be asked to confirm your password before changing the policy."
- The sign-out link and the stale-row text go (`authentication/live-copy.ts:84`, `:92-93`; `live-model.ts:35-39`).
- The step-up hint (`live-copy.ts:46-47`) states: "Blank or 0 uses the 15-minute default. A value under 15 minutes is stricter; a longer value still allows 15 minutes." The second choice reads "Set a shorter time".
- The labels become "Recent password confirmation for policy changes" and, for the effective window, "Password confirmation window".
- The maximum-age hint and the approval impact count from "the last sign-in or password confirmation".
- The current-policy view shows the effective window.
- `act_until` comes from `authenticated_at`.

**M05**
- The account view adds "Password last confirmed" with `authenticated_at`.
- "Workspace sign-in age" (`my-security/live-copy.ts:204`) says it counts from the last sign-in or password confirmation.

**Glossary.** A reviewed row:
- concept `password-confirmation`, with reauthentication and step-up as internal aliases;
- KO "비밀번호 재확인", EN "Confirm your password", AR "تأكيد كلمة المرور" (candidate);
- sources named by path (`docs/ui/screens/M05/spec.md:47`, `:89`; IAM-08);
- scope: every command-posting screen, M02 and M05.

The row also states that a confirmation restarts the "Sign-in valid for" clock, that five consecutive wrong passwords in one session (shared with M05 password change) end that session without locking password sign-in, and what the term does not mean: it is not a new sign-in, does not change the password, grants no permission and is not MFA. KO and AR are AI drafts pending native-language review.

# Tenant and permission boundaries

- **Own session only.** Reauthentication acts only on the caller's own session. The login and workspace come from the session record, never from the request.
- **Step-up only adds a condition.** The gate decision, delegation checks and self-approval rules run unchanged in the executor.
- **Freshness is per session.** Reauthenticating in one browser does not refresh the user's other sessions.
- **Revoked access stays revoked.** Reauthentication re-runs `access`, so a suspended user, a removed membership or a role change still fails (IAM-05, IAM-06).
- **Policy scope.** The policy is read for the session's own workspace; pre-launch has one.
- **No client time is trusted.**
- **Refusals reveal nothing about the target.**
- **Service credentials** do not use the cookie command path.

# Tests and verification

## modules/passwords
- **Holder verification:**
  - `VerifyHolder` known-answer;
  - unknown key returns `wrong`;
  - a concurrent `Put` returns `wrong`.
- **Locked login.** A login locked by 5 failed `Verify` calls still returns `ok` from `VerifyHolder` with the right password. The lock expiry, failure count and window start are unchanged afterwards.
- **No lock effects.** Any number of wrong `VerifyHolder` calls never creates, extends, unlocks or resets the lock. This is a property test against the existing model: `VerifyHolder` is a no-op on lock state.
- **Timing.** The `VerifyHolder` medians for correct, wrong and unknown stay within a factor of 1.5 over 20 runs. Skipped with `-short`.
- 100% coverage and mutation testing of the new function.

## apps/api/server (fixed clock)
- **Coverage from the registry.** The test iterates `s.mutating` on a sample-seed server and checks, for every registered command:
  - mutating and not exempt: refused at window + 1 ns and accepted at exactly the window and after reauthentication;
  - non-mutating or exempt: listed in the exempt table.

  The count (63 on this base) is asserted from the registry, not from a hard-coded list. A synthetic mutating definition registered in the test is refused (fail closed).
- **No family runs without step-up.** A `contract.*` command with an old session is refused.
- **Route table.** A source test scans `boundary.go` for every `POST` route and requires each to appear in the enumerated table with its classification. `sessions/end` is refused when old.
- **Refusal contract**
  - status 403, the code, `next_action: reauthenticate`, `no-store`;
  - the body `request_id` echoed (a valid ID and an invalid one, which gives `commands.Marker`);
  - identical bodies for an existing and a missing target, and for an administrator and a member;
  - the published OpenAPI document lists the code, the `reauthenticate` next action and `x-step-up` on every step-up operation.
- **Nothing recorded.** A refusal adds no journal record, consumes no command rate token, reserves no idempotency key and leaves the counter unchanged. The resend after reauthentication succeeds with no `idempotency_conflict` and no `revision_conflict`.
- **Replay.** Complete a command, let the window lapse, then replay: refused before reauthentication, `replay: true` after.
- **Window**
  - D without a rule.
  - A `command.sensitive` rule shorter than D is honoured; a longer one has no effect.
  - The same for `policy.change` on `policy.propose`.
  - `policy.approve` uses the minimum with the candidate. The dead-end cases: a candidate rule of 5 minutes, and a candidate with a 5-minute maximum age and no rule, each with a session reauthenticated 10 minutes ago, give `reauthentication_required`, not `prerequisite_failed`, and approval succeeds after reauthentication.
  - The M02 `act_until` and `authenticated_at`.
- **MFA fail closed.** A rule requiring MFA gives `prerequisite_failed` with no prompt, for `policy.propose`, for `identity.suspend_user` and for `sessions/end`.
- **Reauthentication endpoint**
  - success refreshes only this session;
  - wrong password;
  - the 5th wrong attempt, mixed with password-change attempts, ends the session and bumps the counter;
  - derive unavailable;
  - body shape, size, no session and bad CSRF;
  - the session ended between verification and commit gives 401 (the atomic setter);
  - a wrong password after the session ended during derivation gives 401 with no counter bump (liveness first);
  - suspended between reserve and commit;
  - generation change gives 409;
  - the maximum age renewed but bounded by the 12-hour absolute lifetime;
  - `/me` and the sign-in and session bodies carry `login` and `authenticated_at` as specified.
- **Not blockable (H1)**
  - The global limiter is exhausted by failed sign-ins, and reauthentication still succeeds.
  - The login is locked by 5 failed sign-ins, and reauthentication with the right password succeeds. Sign-in stays locked with the same expiry.
  - 4 wrong reauthentication attempts do not lock sign-in.
- **Leak**
  - A captured `slog` handler, every response body, the command journal and the OpenAPI document never contain the password bytes.
  - The log lines carry only their declared keys.
- **Timing.** Endpoint medians for correct and wrong passwords are within a factor of 1.5 over 20 runs. Skipped with `-short`.
- **Race.** Concurrent reauthentication, step-up commands, logout, password change and session ending, under `-race -count=40`.
- **Source and AST tests (`go/ast`)**
  - assignments to the `authenticated` field occur only in `sessions.put` and `sessions.reauthenticated`;
  - `sessions.reauthenticated` is called only from the reauthentication handler;
  - `passwords.VerifyHolder` is referenced only in `reauth.go`, after `self` admission and `reserveAttempt` (M-A);
  - `login` and `changePassword` still call `passwords.Verify`, with the per-login lock, and `setPassword` calls neither `Verify` nor `VerifyHolder` (M-A);
  - `s.commands.Execute` is called only in `execute`;
  - no logger call in `reauth.go` takes the decoded body or the password.
- **Builds and checks**
  - 100% coverage of the changed and new code;
  - mutation testing of the changed lines;
  - linux/386 and Windows builds;
  - gofmt; vet.

## Console unit tests
- `failureOf` maps `reauthentication_required`.
- `send`:
  - resends once with an identical body and headers;
  - does not resend on cancel;
  - does not loop;
  - leaves GETs untouched;
  - shares one prompt.
- `useAdminClient` memoization.
- The dialog model: single settle, and no cancel while pending.
- Each screen's failure map covers `reauth_cancelled` and `reauth` in EN, KO and AR.
- The M02 model without the stale sign-out.

## Browser QA
On the production build against a live `hb-api`, in EN, KO and AR at 1440 and 390.
- **Real waiting past D**, end to end in EN:
  - suspend a user (M01);
  - one administration family (recovery lock).

  Each checks:
  - the dialog opens;
  - a wrong password gives the field error;
  - the correct password applies the change once (one journal operation);
  - focus returns.
- **At least one case per family, with the first response stubbed to the refusal.** The resend goes to the live server. Recorded as stubbed:
  - M02 policy approve;
  - M03 grant;
  - M04 credential issue;
  - M05 end other sessions;
  - organizations changes (`relationship.*`);
  - contracts (`contract.*`);
  - installation proposals;
  - integrations;
  - retention.
- **Other cases**
  - Cancel and Escape, with the neutral copy and the form kept;
  - Escape ignored while pending;
  - the dialog over an open review-panel modal, with focus return;
  - the 5th wrong attempt signs the user out;
  - busy;
  - a fresh session shows no dialog.
- **Accessibility and layout**
  - keyboard only;
  - axe on the open dialog;
  - RTL in AR;
  - no horizontal scroll at 390;
  - neutral computed borders.
- **Password handling**
  - the field is empty after every response;
  - no password in storage, the URL or the console;
  - in the network log, the password appears only in the reauthentication request body.
- **Regression.** Rerun the M01–M05, O01–O03 and administration screen QA.

# Residuals

- **Guessing with a stolen session.** A holder-only check means a stolen session cookie and CSRF token give 5 password guesses per stolen session. Those guesses do not count toward the per-login lock, which bounds guesses at sign-in. Before this ticket, the M05 password change already allowed 5 guesses per session, but they counted toward the lock. This is a deliberate trade for H1.
- **Still blockable.**
  - **Sign-in.** A third party who knows a login can still lock sign-in for that login (HB-259 and HB-270 residual). Reauthentication of existing sessions stays available.
  - **Password change.** It keeps `Verify` and the global limiter, so a third party can still block a holder's password change.
  - **Derivation pool.** The pool of 4 is shared with sign-in. A flood of unauthenticated attempts, bounded by the limiter at a burst of 30 plus one token every 2 seconds (`signin.go:30-31`), can delay reauthentication while the request context waits for a slot, but cannot refuse it outright unless the request is cancelled.
- **No rotation.** No cookie or CSRF token rotation on reauthentication (Q1; OWASP recommends rotation, see External references). Mitigations:
  - the cookie is `HttpOnly` and `SameSite=Strict`;
  - the CSRF token is per session;
  - the API is loopback-only.
- **Not journaled.** Step-up refusals and reauthentications are logged, not journaled (Q2; HYB-292).
- **Ordering.** Step-up is checked before permission (declared).
- **No M02 setting.** No policy configuration of `command.sensitive` through M02 (O-4).
- **Identity clock.** The identity module's own 15-minute clock (`LinkIdentity` and `UnlinkIdentity`) is not refreshed by reauthentication. It is unreachable now and must be reconciled when linking is built.
- **No MFA.** No MFA, SSO, external status hold, linking or MFA management (not connected).
- **Translations.** KO and AR copy is an unreviewed AI draft.
- **M05 preview copy.** The M05 preview screen (`identity/pages/my-security/Screen.tsx`, unchanged from main) still says "reauthentication"; only the live screens use "password confirmation".
- **Acceptance.** Screen acceptance (FIX) belongs to the owner alone.

# External references

Consulted 2026-11-17; rechecked by root on 2026-11-18 (WebFetch summaries). The OWASP row was corrected: rotation comes from the risk-events section.

| Source | Observed | Adopted | Not adopted |
| --- | --- | --- | --- |
| OWASP Authentication Cheat Sheet, "Require Re-authentication for Sensitive Features" (https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html) | Require the current credentials before changing sensitive account details or approving sensitive transactions. Token rotation is not in this section; the separate "Re-authentication After Risk Events" section says to invalidate sessions and rotate tokens after re-authentication. | Password re-entry against the bound local credential | Token rotation (residual) |
| GitHub Docs, "Sudo mode" (https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/sudo-mode) | High-impact actions affecting access need authentication even while signed in. A 2-hour period, reset by each sensitive action. | Reauthentication within a session for access-affecting actions | The 2-hour period and the reset |
| Microsoft Learn, "Conditional Access adaptive session lifetime policies" (cited in HB-265) | Fresh authentication can be required for sensitive actions; frequent prompts cause fatigue | A window, at most one prompt per refused request | The "every time" mode |

# Questions remaining for the owner

1. **Q1.** Rotate the session cookie and CSRF token on reauthentication in a follow-up? It needs pickup re-binding and a client-side token update before the resend.
2. **Q2.** Journal step-up refusals and reauthentications when HYB-292 sizes the journal?
3. **O-4.** Make `command.sensitive` configurable in M02 after pre-launch?

Q3 (revocation exemption), Q4 (password change refresh), Q5 and O-2 (command scope), and O-3 (policy rule handling) are resolved in Decisions.

# Review history

**Design review round 1** (separate reviewer, security focus): CHANGES REQUIRED. All findings are folded into this revision.

| Finding | Resolution |
| --- | --- |
| H1: third parties could block reauthentication | No global limiter. Holder-only `VerifyHolder` that ignores the per-login lock. Security argument, tests and honest residuals. |
| H2: body did not match the all-mutating decision | Title, scope, trace, registry-derived command scope, neutral rule name, every command-posting screen, inverted family test, registry-generated coverage test, per-family QA, decisions resolved |
| M1: approval dead end | `policy.approve` uses the minimum with the candidate's window; dead-end test |
| M2: MFA rule on non-policy commands | Fail closed with `prerequisite_failed`, no prompt, for every command; non-policy test |
| M3: classification | Runtime rule `s.mutating && !exempt`; enumerated POST-route table checked by a test |
| M4: OpenAPI and request ID | `next` on `boundaryCode`, enum extended, document test; the body `request_id` echoed |
| L1–L8 and nits | A policy can only tighten; decode before the lock; atomic setter; `login` in sign-in; neutral replay copy; `authenticated_at` in `/me` and M05 copy; dialog pending, settle, stacking and memoization; the identity clock divergence; AST wording; KO heading aligned with M05 `:89` |

**Design review rounds 2 and 3** (same reviewer): round 2 asked for four clarifications, all folded into the body: holder-only source tests for `VerifyHolder` (M-A), the candidate maximum age in the `policy.approve` window (M-B), session liveness first in reauthentication (L-A), and nits (the integrations count, the apps/api request ID check, the limiter wording). Round 3: PASS.
