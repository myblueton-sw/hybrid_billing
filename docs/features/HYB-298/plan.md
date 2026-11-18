---
type: ImplementationPlan
title: M03 grant changes through fixed-revision approval requests, and revocations confirmed by a durable decision ledger in the installation
description: Fix ticket for finding M3 of the HYB-291 planning cross-check. Grant additions and grant revocations become approval requests that a second administrator approves against a fixed revision. Grant and delegation revocations become revocation records that reach deny confirmed, cleanup running and revocation complete only after an fsynced, hash-chained ledger file in the installation's data directory commits the decision, and show decision unconfirmed otherwise. Request limits and lifetime become installation settings with defaults.
status: stable
sources:
  - resource: docs/ui/screens/M03/spec.md
  - resource: docs/planning/v1.16/requirements.json
  - resource: docs/planning/v1.16/detailed-requirements.json
  - resource: docs/planning/v1.16/retention-access.md
  - resource: docs/features/HB-266/plan.md
  - resource: docs/features/HB-267/verification.md
  - resource: docs/reviews/HYB-291-planning-cross-check/report.md
  - resource: docs/reviews/HYB-301-trace-classification/report.md
---

# Scope and ownership

[HYB-298](https://linear.app/hybrid-billing/issue/HYB-298) fixes finding M3 of the [HYB-291 planning cross-check](../../reviews/HYB-291-planning-cross-check/report.md) (`report.md:122`, `:148`, `:349`). Base: main `3e36101`. Branch: `feat/HYB-298-m03-approvals`.

**Owner direction of 2026-11-17.** Implement the spec behaviour as written. Where a step depends on something that does not exist yet, the product blocks or shows the state truthfully, and this plan records the dependency. Nothing is simplified silently.

**Owner decisions of 2026-11-17** (in chat, relayed by root):
- **D-O1.** Open values ship as defaults that the installation can change. They are installation settings (see Installation settings).
- **D-O2.** The decision record store goes into the installed environment now. It is a durable ledger file (see Decision ledger).
- **D-O3.** No emergency revoke path for now (Q1, owner-deferred).

**Ownership**
- Root (Claude Code) owns the plan, implementation, tests and browser QA.
- A separate read-only reviewer reviews the design and then the change, with a security focus. The change touches how permissions are granted and revoked, a public command contract and durable security records (AGENTS.md core rules; `design-first.md`). A security design review of the previous draft is in progress; its findings will be folded in under Review history.
- The `ui-designer` role (AI, read-only) leads the visual design of the new sections and compares the final renders.
- A separate read-only technical-writing review covers the new terms and the EN, KO and AR copy (`product-terminology.md`).
- Machine: mac_mini.

**Allowed files**
- New module `modules/accesschange/*`.
- New module `modules/decisionledger/*`, including the build-tagged lock files.
- Neither module directory exists on this base. Check both again before scaffolding.
- `apps/api/server/access.go` and new files beside it: `accessrequests.go`, `revocations.go`, `ledger.go`, `settings.go` and their tests. `access_test.go` and the reader tests are updated where they call the removed commands.
- `apps/api/server/readers.go`: one response header only (cleanup item C4).
- `apps/api/server/seed.go`: opening the ledger and replaying it in `Start` only.
- `apps/api/server/admin.go`: definition list only. `apps/api/server/server.go`: the `Config` field and wiring only.
- `apps/api/cmd/hb-api/main.go` and `main_test.go`: setting flags only.
- OpenAPI fixtures under `modules/commands/testdata`, if the generated document is pinned there.
- `apps/console/src/service/admin.ts` and its test.
- `apps/api/server/admin.go` handler bodies, for the Q6 increase checks only: `createUser` and the invite role check, `reinstateUser`, the `membershipStep` reinstate step, and `changeRole`. Also `apps/api/server/users_test.go`.
- M01 console copy (`apps/console/src/modules/identity/pages/users/*` copy file), for the EN, KO and AR refusal message only.
- `apps/console/src/modules/identity/pages/access/*`, including new files for the request, revocation and settings sections and their copy.
- `apps/console/package.json`, test list only.
- `modules/auth/listing.go` and `modules/gate/gate.go`, read-only additions only, changing no decision (review M3, L2):
  - `Registry.Generation() uint64`, the deny revision;
  - an `Effective` flag on `DelegationView`, computed by the same chain rules as a decision (`modules/auth/decide.go:33-41`, `:43-70`).
- `docs/ui/terminology/product-glossary.md`: new rows only.
- `docs/features/HYB-298/*`. The closure of HB-266 and HB-267 residuals is recorded here; their records are not edited (review L7).

**Not in scope**
- Support sessions (HB-36, `M03/spec.md:85`).
- Delegation creation.
- Bulk revocation (IAM-07).
- An emergency single-person grant revocation (D-O3).
- Off-host replication of the decision ledger.
- Persistence of grants, requests and sessions.
- Changing settings from the UI.
- Step-up reauthentication. HYB-293 covers it (see Dependencies).

# Observed baseline

Facts from the code on `3e36101`. Line numbers refer to that commit.

**Grants apply directly.**
- `access.grant` adds a subject grant in one step (`apps/api/server/access.go:350-408`).
- `access.revoke_grant` closes a grant in one step (`access.go:333-342`).
- `access.revoke_delegation` closes a delegation in one step (`access.go:343-347`).
- All three use `administer_installation` (`access.go:410-418`).
- The only separation today is that the caller cannot grant to themselves or revoke their own grant (`access.go:335`, `:353`).

**Duplicate grant.** HB-267 recorded L5: after a `partial` failure the settled reload renews the attempt, so a second submit can create a second grant (`docs/features/HB-267/verification.md:42`). The causes:
- The console renews the idempotency attempt whenever the page revision changes (`apps/console/src/modules/identity/pages/access/Live.tsx:69`, `:95-96`).
- The server's `grant` target `grants` resolves at every revision (`access.go:263-266`).
- `AddGrant` uses a fresh random ID on every call (`access.go:396`).
- The module accepts any number of grants with identical terms (`modules/auth/registry.go:253-287`).

**What revocation does today.**
- `RevokeGrant` and `RevokeDelegation` close the record and bump the registry generation (`modules/auth/registry.go:289-307`, `:422-440`, `:135`).
- Decision stamps carry the generation (`modules/auth/types.go:265-272`), so a stamp issued before a revocation no longer rechecks.
- The reader route decides on every request and holds no cached permission (`apps/api/server/readers.go:121-126`).
- Console sessions hold no permission state (`apps/api/server/session.go:26-32`).
- Proxied reader responses keep whatever cache headers the Python reader sends; only cookies and CORS headers are removed (`readers.go:92-99`). `If-None-Match` is forwarded to the reader (`readers.go:72`).
- `apps/api` imports no job runner.
- Every domain store is in memory. No durable ledger exists anywhere in `apps/api` or `modules`.

**Installed environment.**
- `hb-api` reads its configuration from command-line flags only (`apps/api/cmd/hb-api/main.go:30-38`). There is no configuration file.
- The data directory (`-data`, default `.hb-api`) is created `0700`. On a non-Windows system it is refused when it is a symlink, not a directory, group- or world-accessible, or not owned by the process user (`apps/api/server/seed.go:50-63`; ownership check in `owner_unix.go`).
- On Windows the ownership check returns true (`owner_windows.go`) and mode bits are not checked (`seed.go:59`).
- Secret files are written exclusively with `0600` (`seed.go:65-72`, `readers.go:62-69`).
- `Start` runs before the listener serves (`main.go:63-71`).
- The seed recreates the users and the seed grants with deterministic IDs on every start: `reader-console`, `reader-<user>`, `installation-console` and `installation-<user>` (`seed.go:219-246`). Grants created through commands get random IDs (`access.go:396`).

**Two-person patterns that already exist.**
- authpolicy proposals:
  - an equal pending proposal by the same proposer is returned instead of duplicated (`modules/authpolicy/registry.go:157-170`);
  - the proposer cannot approve (`registry.go:200-201`);
  - only the proposer can withdraw (`registry.go:233-237`);
  - an applied change drops the other pending proposals of the workspace (`modules/authpolicy/changes.go:116-125`).
- The API layer around them:
  - shows `can_approve` and `can_withdraw` per row (`apps/api/server/policy.go:118-129`);
  - maps refusals (`policy.go:174-185`);
  - requires a rationale on every command (`policy.go:289-296`).
- Retention proposals bind the approver to a reviewed impact digest (`apps/api/server/retentioncmds.go:157-182`, `:184-197`). They map self-approval to `prerequisite_failed` and limits to `prerequisite_failed` (`retentioncmds.go:37-55`).
- `modules/approvals` has proposals with reject and superseded states. Its `grant` means a billing approval authority with a currency limit (`modules/approvals/types.go:39-41`, `admin.go:87-93`), not an access grant.

**Executor facts this design relies on.**
- A mutating command needs an expected revision equal to the resolved one, or it gets `revision_conflict` (`modules/commands/execute.go:356-357`).
- Idempotency is per installation, client, command and key (`execute.go:340`).
- A handler error together with a non-empty outcome is `internal_error` with the operation ID (`execute.go:420-425`).
- `self_approval_denied` is not a handler code (`modules/commands/types.go:71-74`).
- An accepted outcome is `202` with a job ID and no business state (`modules/commands/execute.go:184-185`; `apps/api/server/errors.go:106-108`). It still advances the revision (`apps/api/server/commands.go:155-163`).

**Platform facts (Go 1.25.7, the toolchain on this machine).**
- `(*os.File).Sync` uses `F_FULLFSYNC` on macOS, falling back to `fsync` on `ENOTSUP` (`$GOROOT/src/internal/poll/fd_fsync_darwin.go:15-28`).
- It is `fsync` on Linux and `FlushFileBuffers` on Windows.
- `syscall.Flock` exists on macOS and Linux.
- The stdlib `syscall` package has no `LockFileEx` on Windows. It is reachable through `syscall.NewLazyDLL("kernel32.dll")`.

**Sample seed.** Two administrators (`admin`, `admin2`) and one member (`apps/api/server/seed.go:24-27`). Both administrators hold `view focus_data rows` and `administer_installation`. No delegations are seeded; delegations exist only in tests (`access_test.go:174-178`).

# Requirement trace

Sources:
- `docs/ui/screens/M03/spec.md:43-47`, `:65-66`, `:80-86`, `:101-115`, `:133-159`;
- `requirements.json` HB-03, HB-23, HB-33, HB-42;
- `detailed-requirements.json` AR-09, DB-08, QA-04;
- `retention-access.md:59-63`, `:98`, `:110-117`.

These planning inputs exist in the authorized local workspace, not in the Git tree.

Status meanings follow the [cross-check report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). "After" is the target of this ticket; no acceptance test has run.

| ID / spec line | Requirement (short) | After HYB-298 | Status |
| --- | --- | --- | --- |
| `M03/spec.md:83` | Grant change approval request: grant authority, self-elevation and ceiling checks, fixed-revision request, add or remove scope confirmed | Add and revoke become requests. The proposer cannot be the subject. The approver cannot be the proposer, and for an add cannot be the subject. Ceiling and days are checked at proposal and at approval. The approver must send the request digest. A changed basis supersedes the request. | covered (pre-launch grant pairs) |
| `M03/spec.md:84`, `:47`, `:65-66` | Delegation revoke: received → durable ledger commit and deny revision → token, cache and job cleanup → complete. A failed or unknown commit is never a success. | A revocation record for grant and delegation revocations. The deny is applied, then the decision is appended to the installation's ledger file and fsynced. A commit gives `deny_confirmed` → `cleanup_running` → `revocation_complete`. A failed, unknown or refused commit gives `decision_unconfirmed`. Sequence and watermark are shown in the expert detail. | covered |
| `M03/spec.md:109-113` | User-facing revocation states | All five states are rendered with the spec copy. `revocation_complete`, `decision_unconfirmed` (reason `ledger_isolated`, by starting in isolation) and the isolation state are reachable in the browser. The `ledger_stopped` and `ledger_full` reasons are asserted through injected faults and a small bound in API tests. `received`, `deny_confirmed` and `cleanup_running` pass inside one command, because the deny, the commit and the cleanup are synchronous today. They are tested in the module and rendered in console unit tests. | covered |
| `M03/spec.md:56`, `:83` | Grantor ceiling: the grantor cannot give more than they hold | The proposer and the approver must each be allowed the granted pair, checked through `Explain` at proposal and at approval (review M5) | covered |
| `M03/spec.md:106-107` | blocked, error copy | `blocked` when the ceiling is exceeded or no approver is eligible; `error` for a superseded request | covered |
| `M03/spec.md:115` | Job enum; an unknown result is `waiting` with a reason; retry only where proven | Job state per revocation: `succeeded` when complete; `waiting` with reason `decision_unconfirmed`. No retry button (R2). | covered |
| HB-03 | Separate author and approver; no leakage on ID tampering or revocation | Author and approver separation for grant changes. Tampered request IDs are refused. Revoked access is refused on the next request and stays revoked across a restart. | deferred (registry-wide leakage acceptance not built) |
| HB-04 | Setup role kept apart; operational read after setup revoke | Unchanged from HYB-301 | not connected |
| HB-23 | Request and decision as separate commands; self, AI and concurrent-change refusals; withdrawal | Separate propose, approve, reject and withdraw commands. Human only. A basis change supersedes. | covered for M03 grant changes; billing approvals are out of scope |
| HB-33 | One command registry; API-only cycle; unsupported is never success | The new commands are in the common registry and OpenAPI. `decision_unconfirmed` is never returned as completed. | covered (these commands) |
| HB-36 | Support sessions | Not built | deferred |
| HB-42 | Grant, change and revoke lifecycle; immediate block | Two-person grant changes. Revocation blocks at the next decision and is durably recorded. Bulk paths, MFA and external identity are not built. | covered for M03 grant change and revocation; deferred elsewhere |
| AUD-AR-09 | Delegation chain, re-delegation, support actor | Delegation revoke shows the descendants it ends. Delegation creation and support sessions are not built. | deferred |
| AUD-DB-08 | Setup permission with a revocation condition | Unchanged from HYB-301 | deferred / not connected |
| AUD-QA-04 | Idempotency scope, payload conflict 409, expected revision; approval request differs from human approval | Executor rules apply to every new command. Approval requires a human principal. OAuth and the client-user intersection are not built. | deferred (OAuth) |
| IAM-05 (`retention-access.md:114`) | Permission reduction reaches sessions, keys, jobs and files | Cleanup inventory per revocation (see Cleanup). External detection delay is not connected. | deferred |
| IAM-07 (`:116`) | 100k revocation isolation and resume | Not built | deferred |
| IAM-08 (`:117`) | Step-up for sensitive commands | HYB-293 covers the new commands through its fail-closed rule | covered by HYB-293 when merged |
| IAM-01–04, IAM-06 | Bootstrap, sign-in protocols, identity reuse, MFA recovery | Not touched | out of scope |
| `retention-access.md:59` | Authoritative deny and version first, then sessions, tokens, keys, caches and pending jobs | The deny is applied before the ledger commit. Cleanup runs only after `deny_confirmed`. | covered |
| RET-05 / `retention-access.md:98` | Durable commit of an independent decision ledger before a confirmed answer; monotonic sequence and watermark; re-apply before opening the API after a restore; a missing or unverifiable ledger stays isolated; independently retained and replicated | Covered: fsync before a confirmed answer; sequence and hash chain as tamper evidence; watermark; in-place integrity check and torn-tail recovery on open; replay before the listener serves; access-change isolation when the ledger is missing or unverifiable after one was created. After a restart, earlier decisions are labelled "From an earlier server run". Not covered locally: truncation at a record boundary and rollback of the whole data directory, which need an independent copy (R1, review N-H1). | partially covered; deferred (independent copy and replication, R1) |

# Decision

## Where the code lives

**`modules/accesschange`** holds approval requests and revocation records for access changes.
- It is stdlib-only, like `modules/authpolicy` (`modules/authpolicy/source_test.go:39-45`), and knows nothing about `modules/auth`.
- It defines a `Ledger` interface: `Commit(Decision) (Receipt, error)`.
- `apps/api` composes it with `gate.Administration` through callbacks.

**`modules/decisionledger`** holds the durable file: record format, hash chain, integrity check, append with fsync and the cross-platform lock.
- It is stdlib-only (`os`, `syscall`, `crypto/sha256`, `encoding/json`), with build-tagged `lock_unix.go` and `lock_windows.go`.
- It is a separate module because durable storage and its platform code are a different responsibility and change for different reasons than the request workflow. The workflow module stays pure and property-testable.
- `retention-access.md:98` names the same ledger for deletion tombstones. This plan does not wire retention to it.

Alternatives considered for the workflow:

| Option | Why not |
| --- | --- |
| Inside `modules/auth` | `auth` is the decision core. A workflow there enlarges the most security-critical review surface, and only `gate` may import `auth` (`modules/auth/source_test.go`). |
| Reuse `modules/approvals` | Its proposals are billing approval authorities with currency limits and a decimal profile (`modules/approvals/types.go:39-41`, `admin.go:68-74`). |
| Reuse `modules/authpolicy` | An approval drops every other pending proposal in the workspace (`changes.go:116-125`), so unrelated policy changes would cancel grant requests. |
| Only in `apps/api` | The separation and state rules would be untestable without HTTP. |

The workflow adopts the authpolicy shape (propose, approve, withdraw, equal-proposal return) and adds reject, expiry, superseded and the retention-style digest confirmation.

## Approval request model

**Change.** A request carries one change:
- `add`: subject, action, kind, fields, customer selector and days;
- `revoke`: a grant ID.

It is encoded canonically and hashed with SHA-256 under the label `access_change_v1`, like `modules/approvals/digest.go`.

**Basis (the fixed revision).** `apps/api` computes the facts the change was judged against and passes them in. The module stores their encoding.
- `add`:
  - the covering console ceiling grant's ID, version and `NotAfter` (`access.go:119-127`). The choice is deterministic: the latest `Until`, then the lowest grant ID. The existing loop already yields this order: `Grants()` sorts by ID (`modules/auth/listing.go:52`), and `ceiling()` keeps that order (`access.go:121-125`). The loop replaces the chosen `Until` only when a later one is strictly greater (`access.go:375`), so the first, lowest-ID grant wins a tie. Verified in code for review L8.
  - the subject's membership version and state (`identity.MemberInfo.Version`, `modules/identity/workspace.go:240`);
  - the subject's user epoch and user state (review L9);
  - the absence of an active equivalent grant, meaning the same subject, action, kind, fields and customer, not closed and not expired.
- `revoke`: the grant ID and version, and that the grant is not closed.
- Both kinds (review M5):
  - the proposer's membership version, role and user epoch;
  - whether the proposer holds an effective `administer_installation` grant;
  - for an `add`, whether `Explain` allows the proposer the granted pair (grantor coverage, `M03/spec.md:56`, `:83`).
  - A change to any of these supersedes the request, so a proposer who loses administration or demotes after proposing cannot have the request applied.

**Request revision.** The request revision is the digest over the change and the basis. The read shows it. The approve command must send it back in the field `digest`, so the approver confirms the exact content and basis they reviewed (the pattern of `retentioncmds.go:157-182`).

At approval, `apps/api` recomputes the basis. If it differs, the module records `superseded` and the command answers `invalid_stage`. Reads compute the same comparison without changing state, and show `current: false` with the spec's `error` copy (`M03/spec.md:107`).

**States.** `pending` → `applied` | `rejected` | `withdrawn` | `expired` | `superseded`. All end states are final.

**Separation and self-elevation.**
- **Human actors only.** Every actor must be human.
  - The `apps/api` handler is the layer that checks it: `answer.Kind()` must be `identity.Human`. A non-human actor maps to `prerequisite_failed`; this is the only mapping.
  - The module repeats the check as defence in depth, with reason `not_human`, which maps to the same code.
- The proposer may not be the subject of an `add`, or the holder of the grant in a `revoke`. This keeps the existing caller rules (`access.go:335`, `:353`).
- The approver and the rejecter may not be the proposer.
- For an `add`, the approver may not be the subject.
- **Revoking one's own grant.**
  - The holder of the grant may *approve* its revocation, because a revocation only reduces privilege (root decision on Q2). The approver still differs from the proposer.
  - The holder may never *reject* it (review H1). The read gives `can_reject: false`, and the command refuses with `prerequisite_failed`. Otherwise a holder could keep their own access by rejecting its removal.
- **Revoke guards** carried from today (`access.go:335`; review M1). `request_revoke` refuses client grants (`ClientID != ""`) and `administer_installation` grants with `prerequisite_failed`. The guard runs at proposal and again at approval.
- **Grantor coverage** (`M03/spec.md:56`; review M5). For an `add`, `Explain` must allow the granted pair to the proposer at proposal, and to both the proposer and the approver at approval. Nobody grants what they do not hold.
- Proposing, approving, rejecting and withdrawing all need `administer_installation`, which the executor checks with a fresh decision.
- Mutual proposals still need a third person as approver, so no elevation path exists without two distinct administrators other than the subject.

**Check order** (review L1). Every action checks in this order:
1. human actor;
2. separation, holder and grantor rules;
3. state and lazy expiry;
4. digest;
5. basis.

The first failure answers. A refusal changes no state. The only exceptions are the lazily computed `expired` and the basis outcome `superseded`, which are recorded because they are final facts, not effects of the caller.

**Ceiling checks** happen at proposal and again at approval:
- the pair is inside the console ceiling (`access.go:373-382`);
- `focus_data` requires customer `all` (`access.go:383-390`);
- `days` is at least 1, and the expiry rule below guarantees `approval time + days ≤ ceiling NotAfter`.

**Eligible approvers.** These are active human members with the administrator role and an effective `administer_installation` grant, minus the proposer, and for an `add` minus the subject.
- The read shows the count per request.
- When the count is 0, the request shows the spec's `blocked` copy with "no eligible approver" (`M03/spec.md:106`).
- Example: in the sample seed, a grant to `admin2` proposed by `admin` has no eligible approver. It is still accepted, so a third administrator can approve it later.

**Expiry.** `expires_at = min(proposed_at + request lifetime, ceiling NotAfter − days)`. The request lifetime is the installation setting `access_request_hours`, 168 by default.
- Expiry is evaluated lazily. A read shows `expired`, and any action on an expired request records `expired` and answers `invalid_stage`.
- A proposal whose expiry would already be past is refused with `invalid_request` on `days`.

**Duplicates.**
- A proposal equal to a pending one, with the same change code and the same proposer, returns the existing request (`authpolicy/registry.go:163-165`).
- Any other pending request for the same subject and pair, or for the same grant revocation, gives `invalid_stage`.

**Withdraw and reject.** Only the proposer withdraws. Any eligible approver can reject, with a rationale that the read shows. The holder of a grant may not reject its revocation (H1).

**Applying.**
- `Approve(number, approver, basis, apply func() (string, error))` runs `apply` under the module lock.
- On success it records `applied`, the approver, the time and the result reference: the grant ID for an add, the revocation ID for a revoke.
- If `apply` fails, the request stays `pending`.
- Lock order is always `s.domain` (write) → `accesschange` mutex → auth registry mutex → `decisionledger` mutex. Nothing calls back up this chain, so no cycle exists.

**Limits.** Limits come from the installation settings.
- Pending per proposer and pending per workspace refuse a new *request* with `prerequisite_failed`, as retention does (`retentioncmds.go:51-52`).
- Ended requests kept and revocations kept only prune.
- A revocation is never refused because of a record limit (review M4). That covers the approval of a revoke request and a delegation revocation.
- A revocation record is pruned only once it is final and confirmed (`revocation_complete`). A record in `decision_unconfirmed` is never pruned. The list may therefore exceed the setting, bounded by one record per grant or delegation.
- The module validates the limits it receives as well, as defence in depth.

## Installation settings

The former open values O-1 and O-2 ship as named installation settings with validated bounds (D-O1).

**Mechanism.**
- They are `hb-api` command-line flags. That is the configuration mechanism the installation already uses (`main.go:30-38`).
- No configuration file is added. A second mechanism would need its own format, permissions and precedence rules, and nothing else needs one yet.
- Each flag is registered with `fs.Func` and parsed as a canonical decimal integer: no sign, no leading zero, at most 5 digits. `flag.Int` would accept `0x10` and `-5`.
- After `fs.Parse`, `fs.Visit` records which flags were given. The source of each setting is therefore `start_flag` or `default`.
- The values go into a new `server.Config.Access` (`AccessSettings`).
- A zero `Config.Access` means all defaults, like `Config.Identity` (`server.go:181-183`). This keeps existing tests and callers unchanged.

**Validation.** `server.New` validates every bound and relation. If any fails, it returns `ErrConfig` naming the setting. `run` returns that error before listening, and the process exits non-zero (`main.go:55-58`). A server never starts with an out-of-bound value.

| Setting | Flag | Default | Bounds |
| --- | --- | --- | --- |
| Request lifetime in hours | `-access-request-hours` | 168 | 1–720 |
| Pending requests per proposer | `-access-pending-per-proposer` | 25 | 1–100 |
| Pending requests per workspace | `-access-pending-per-workspace` | 200 | 1–1000, and at least the per-proposer value |
| Ended requests kept in memory | `-access-ended-kept` | 1000 | 50–10000 |
| Revocations kept in memory and in the read | `-access-revocations-kept` | 1000 | 50–10000 |
| Ended requests shown in the read | `-access-ended-shown` | 50 | 1–200, and at most the ended-kept value |
| Decision ledger file bound in MiB | `-decision-ledger-max-mib` | 256 | 1–4096, and not below the current file size, checked at `Start`. Within one record of the bound, the ledger starts `full` (N-M2, R3-L2). |

`-access-revocations-kept` bounds the in-memory list only. The ledger file is append-only and is never pruned by this setting.

`-accept-new-decision-ledger` is a one-time operator switch, not a setting (see Decision ledger). It is not shown among the settings.

**Display.**
- The read returns `settings[]`, each with `name`, `value`, `unit`, `default`, `min`, `max` and `source`.
- M03 shows them read-only in an "Approval settings" disclosure under the request section.
- The source is in words: "Installation default" or "Set when the server started".
- A line explains that a value is changed by restarting the server with a different value.
- There is no edit control.

## Revocation state machine

A revocation record exists for every grant revocation (created when its request is approved) and every delegation revocation. It holds:
- the target kind, ID and version;
- the requester, and the approver for grants;
- the state, with the time of each transition;
- the ledger receipt: sequence and watermark, or the failure reason;
- the cleanup items;
- its origin: `this_process` or `ledger_replay`.

```
received ──deny applied, ledger committed──► deny_confirmed ──► cleanup_running ──► revocation_complete
   │
   └──deny applied, commit failed / unknown / ledger not usable──► decision_unconfirmed
```

**Order inside one command,** under the domain write lock (review M2, option a):
1. **Apply the deny:** `RevokeGrant` or `RevokeDelegation` with the basis version.
   - If this fails, no revocation record is created.
     - For a grant, the approval request stays `pending`.
     - The command answers the named failure: `dependency_unavailable`, `invalid_stage` or `revision_conflict`.
     - A rerun is an ordinary new attempt, for grants and for delegations alike.
2. **Record `received`,** with the deny revision: the auth generation after the deny (`Registry.Generation()`, review L2).
3. **Commit:** `Ledger.Commit(decision)`.
   - **With an advancing receipt:** `deny_confirmed`. Then cleanup: `cleanup_running`, the items run synchronously, then `revocation_complete`. An advancing receipt has a sequence greater than the last one seen and a watermark that differs from it.
   - **With a non-advancing receipt:** `decision_unconfirmed` with `commit_unknown` (L2).
   - **Without a receipt:** `decision_unconfirmed`, with the reason `commit_failed`, `commit_unknown`, `ledger_isolated`, `ledger_stopped` or `ledger_full`.
   - **Never fails open.** A revocation is never refused because of the ledger state. In `isolated`, `stopped` and `full` the deny is applied in this process, and the command answers `accepted`. The record reads `decision_unconfirmed`.
   - **Nothing touched in isolation.** In `isolated` the commit is refused without opening, creating or writing any ledger file. Only `-accept-new-decision-ledger` creates one.

**Job state** for every state (review L5):

| Revocation state | Job state |
| --- | --- |
| `received`, `deny_confirmed`, `cleanup_running` | `running` |
| `revocation_complete` | `succeeded` |
| `decision_unconfirmed` | `waiting`, reason `decision_unconfirmed` |

There is no `failed` job state. A revocation whose deny fails creates no record and returns no job ID (M2).

The job ID is the revocation ID. There is no read by job ID. The state is read from `revocations[]` in `GET /api/v1/identity/access`. No `/jobs/{id}` route is added.

**Delegations ended by a grant revocation** (review M3). A delegation stops applying when its root delegator loses the direct grant that backs it, or when its delegate no longer holds any grant in the workspace (`modules/auth/decide.go:40`, `:63`; `registry.go:341-349`).
- **Before approval.** The approve panel lists the active delegations where the grant holder is the delegator or the delegate, under the heading "May end with this revocation".
- **After the deny.** The revocation record stores the delegations whose `Effective` flag went from true to false, as `ended_delegations`.
- **Delegation revocation.** The record also stores the descendants of a revoked delegation.
- **Listing.** The delegation listing shows a delegation that is not effective as **ended by the chain**, not active. A new state, `ended`, joins `stateOf` (`access.go:103-113`, `:171`) whenever `Effective` is false and the delegation is neither closed nor expired.
- **Cleanup row.** The delegations ended by the chain form cleanup item C9.

**Deny before commit** (root decision on Q3). The deny is applied in-process before the ledger result is known, following `retention-access.md:59`, the authoritative deny and version first. Blocking is the safe direction. The spec forbids *reporting* success on a failed or unknown commit (`M03/spec.md:47`, `:84`), and the state then reads `decision_unconfirmed`, never success.

**What each state shows:**

| State | Reachable in the product | Shown |
| --- | --- | --- |
| `received` | Passes inside the command | Spec copy (`:109`) |
| `deny_confirmed` | Passes inside the command | Spec copy (`:110`), with ledger sequence and watermark in the expert detail (`:66`) |
| `cleanup_running` | Passes inside the command | Spec copy (`:111`), with per-item progress over the fixed item count |
| `revocation_complete` | Yes | Spec copy (`:112`). Lists every item with its final state, separates the items outside the controlled scope, and shows sequence and watermark. Job state `succeeded`. |
| `decision_unconfirmed` | Yes, on a failed, unknown or refused commit | Spec copy (`:113`), plus the ledger state line (for example "The decision ledger failed its integrity check at line 12"). Job state `waiting`, reason `decision_unconfirmed`. The target row shows "Revocation not confirmed", not "Revoked" (`:47`). Cleanup shows "Not started: waits for a confirmed decision", with the planned scope. |

**Effect preview.** When a subject has a revocation that has not reached `deny_confirmed`, the preview adds: "A revocation for this person is not confirmed. This result reflects this server process only."

**No retry command.** There is no retry command for `decision_unconfirmed` in this ticket (residual R2).
- After a write failure, the ledger file may end in a partial line, so appending again in the same process is unsafe.
- After an integrity failure, the file needs operator repair (R7).
- A retry would therefore not be provable (`M03/spec.md:115`).

## Decision ledger

The always-not-connected ledger of the first draft is replaced by an append-only file in the installation's data directory (D-O2). The former open value O-3 is resolved as a local file.

**What it protects, and what it does not** (review N-H1).
- The hash chain gives *tamper evidence* against corruption and in-place edits of records. It is not protection.
- A process with write access to the data directory can rewrite a consistent chain, cut the file at a record boundary, or roll back the whole directory, and none of that is detectable locally.
- Detecting truncation and rollback needs an independent copy, which is R1.

**Location.**
- The file is `<data>/decisions/ledger.jsonl`.
- `Start` creates `decisions/` with `0700` after `checkData` (`seed.go:75`), and refuses it with the same rules as the data directory: symlink, not a directory, group- or world-accessible, or not owned by the process user (`seed.go:50-63`).
- Creating `decisions/` syncs the data directory. Creating the file syncs `decisions/` (N-L2).

**Opening on macOS and Linux** (N-L1).
- The file is opened with `O_RDWR|O_APPEND|O_CREATE|O_NOFOLLOW` and mode `0600`.
- `Fstat` on the opened descriptor must show a regular file, permissions without group or world bits, the process user as owner, and `Nlink == 1`. Anything else refuses `Start`.

**Opening on Windows.**
- There is no `O_NOFOLLOW`, so the check is an `Lstat` before opening and `os.SameFile` after.
- Mode bits and ownership are not enforced (`owner_windows.go`). Owner-only access depends on the data directory's ACL as the installer sets it (R8).

**Durability.** It is claimed for a local disk only. Network file systems, exFAT and removable media may not honour `fsync`, `F_FULLFSYNC` or `FlushFileBuffers`. That is assumption A5, and installing on such a volume is not a tested combination.

**Lock** (N-M3). One process at a time, standard library only.
- **macOS and Linux:** `syscall.Flock(fd, LOCK_EX|LOCK_NB)`.
- **Windows:** `LockFileEx` through `syscall.NewLazyDLL("kernel32.dll")`, with `LOCKFILE_EXCLUSIVE_LOCK|LOCKFILE_FAIL_IMMEDIATELY` over the range from offset 0 to the maximum offset (`0xFFFFFFFF`, `0xFFFFFFFF`).
  - No `golang.org/x/sys` dependency is added, so no licensing review is needed.
  - There is no lock-file fallback.
- **One handle.** The lock is taken on the opened handle *before* the integrity read. Every read and append goes through that same locked handle. On Windows the byte-range lock is mandatory for other handles.
- **Release.** The operating system releases the lock when the process exits, so no stale lock survives a crash.
- **Lock held:** `Start` fails with `ErrStart`: "the decision ledger is in use by another process".
- **Within the process,** a mutex serialises appends.

**Records.** One JSON object per line, encoded from a fixed struct. The 2 KiB record limit and the 4 KiB line limit are constants fixed by the record format (N-M2). Every record has these fields:

| Field | Content |
| --- | --- |
| `v` | `1` |
| `seq` | 1, 2, 3, … with no gaps |
| `prev` | SHA-256 hex of the previous record's line bytes without the newline; 64 zeros for the genesis record |
| `at` | RFC 3339 UTC with nanoseconds |
| `kind` | `genesis`, `recovery` or `decision` |

The **genesis** record (`seq` 1, N-H1) adds:
- `ledger_id`: 128 random bits in hex;
- `created_at`;
- `replaces`: the replaced ledger's ID when known, otherwise `unknown`;
- `replace_reason`: `missing` or `integrity_failed`.

The last two are present only when an operator accepted a new ledger. The read shows `ledger_id` and `created_at`. The earlier "foreign installation" check is dropped: the installation ID is the constant `prelaunch` (`server.go:31`) and proves nothing.

A **recovery** record (N-M1) adds `truncated_bytes` and `recovered_at`.

A **decision** record adds:

| Field | Content |
| --- | --- |
| `workspace` | Workspace ID |
| `decision` | `revoke_grant` or `revoke_delegation` |
| `target`, `target_version` | The grant or delegation ID and the version that was closed |
| `revocation` | The revocation record ID |
| `request`, `request_digest` | For grants: the approval request and its revision |
| `requested_by`, `approved_by` | Internal user IDs |
| `deny_generation` | The auth generation after the deny, which is the deny revision (review L2) |
| `ended_delegations` | IDs of delegations ended by the chain rules or as descendants |

Records hold no rationale text, logins, customer IDs, scope or credentials. They do hold internal user IDs, and those are kept as long as the ledger is. The retention period for them against `retention-access.md:76` (user, identity and access audit data) is open value O-6 (N-L6).

**Append.**
1. Build the line with `seq = last + 1` and `prev = hash(last)`.
2. Write the line and its newline in one `Write` through the locked handle.
3. Call `Sync`.

Only a nil `Sync` yields a receipt with `sequence` and `watermark`. The outcomes:
- A write error or short write gives `commit_failed`.
- A `Sync` error after a full write gives `commit_unknown`: the bytes may or may not be durable.
- Either way, the ledger enters `stopped` for the rest of the process.

**File bound** (N-M2). The installation setting `decision_ledger_max_mib` (flag `-decision-ledger-max-mib`) is 256 by default, with bounds 1–4096.
- At `Start`, a configured bound below the current file size refuses `Start`, naming the setting. It is a configuration error.
- A ledger that is at the bound, or within one 2 KiB record of it, starts in `full` (review R3-L2). Privilege increases are refused, and revocations read `decision_unconfirmed`. The server is never left unable to start.
- An append that would pass the bound is not written. The ledger becomes `full`.

**Watermark.** `<seq>:<sha256 hex of the last record line>`. A new ledger's watermark points at its genesis record.

**Integrity check on open.** It runs after the lock is taken and reads through the locked handle.
- **Torn tail** (N-M1). A final segment without a newline is a torn tail. It was never confirmed, because confirmation follows the newline and the `Sync`. The check:
  1. truncates the file to the last verified line boundary;
  2. calls `Sync`;
  3. appends a `recovery` record, then calls `Sync` again.

  The read shows the recovery: when it happened and how many bytes were removed.
- **Every other line** must:
  - end in a newline;
  - decode with no unknown field;
  - have `v` 1;
  - have a `seq` one above the previous line's, starting at 1;
  - have a `prev` equal to the hash of the previous line;
  - be a genesis record exactly at `seq` 1.

  The first violation sets `integrity_failed` with its line number. This covers a byte flip, a deleted, reordered or duplicated line, a complete but altered last line, and an oversize line.

**Missing ledger and isolation** (N-H1, faithful to `retention-access.md:98`, "원장이 없거나 … 격리 상태로 유지").
- **The marker.** The first successful ledger creation writes a marker file `<data>/decision-ledger.created`, outside `decisions/`, with mode `0600`. It holds the ledger ID and the creation time. Writing it syncs the data directory.
- **First start ever.** No marker and no ledger: the ledger is created with a genesis record, then the marker is written.
- **Ledger without a marker,** for example an upgrade from this ticket's first build: the ledger is verified, and the marker is written from its genesis record.
- **Isolation.** The marker exists, and the ledger is missing, fails verification, or has a different ledger ID than the marker. `hb-api` then starts in **access-change isolation**:
  - **Privilege increases are refused** with `dependency_unavailable`. That is `access.request_grant` at proposal, `access.approve_request` for an add request, and the identity increases listed under "Privilege increases while the ledger cannot record".
  - **Revocations are applied in this process only.** `access.request_revoke` can be proposed. `access.approve_request` for a revoke request and `access.revoke_delegation` both apply the deny and answer `accepted`. The record reads `decision_unconfirmed` with `ledger_isolated`.
  - **No file is touched.** The commit is refused without opening, creating or writing a ledger file.
  - **Other actions** keep working: rejects, withdrawals and reads.
  - **The M03 ledger line** says in words, for example: "Grant additions are paused: the decision ledger is missing. Revocations made now are applied in this server process only and will not be re-applied after a restart. An operator must restore the ledger or accept a new one."
  - **Suspension** stays available as an immediate cutoff of a person: `identity.suspend_user` and `identity.suspend_membership`.
- **Accepting a new ledger.** The operator leaves isolation only by restarting with `-accept-new-decision-ledger`. That run:
  1. renames any existing file to `ledger.jsonl.replaced-<UTC time>` and keeps it;
  2. creates a new ledger whose genesis record names the replaced ledger ID (or `unknown`) and the reason;
  3. rewrites the marker.
- **The flag is single-use.** It refuses `Start` when there is nothing to replace, so it cannot be left on.
- **Consequence of a new ledger.** The earlier decisions are no longer replayed. Seed grants revoked under the old ledger come back after this restart. M03's ledger line states this for a replaced ledger, and R7 records it.

**Ledger states.**

| State | Cause | Effect |
| --- | --- | --- |
| `ok` | — | — |
| `recovered` | `ok`, after a torn-tail recovery at this start | Shown until the next start |
| `isolated` | Missing ledger, or a verification failure, with the marker present | Access-change isolation |
| `stopped` | A failed write or `Sync` in this process | See below |
| `full` | The file bound is reached, including a ledger that is at the bound or within one record of it at `Start` | See below |

In `isolated`, `stopped` and `full`:
- **Privilege increases** are refused (`dependency_unavailable`), at proposal and again at approval.
- **Revocations** are applied in this process only. They read `decision_unconfirmed` with `ledger_isolated`, `ledger_stopped` or `ledger_full`.
- **After a restart,** revocations applied under `isolated` are not replayed, because nothing was recorded. Under `stopped` or `full` the same holds: the failed or unwritten record is not replayed.

**Privilege increases while the ledger cannot record** (binding root decision on Q6, review R3-M2). A privilege increase is refused whenever a revocation could not be recorded. In `isolated`, `stopped` and `full` these are refused with `dependency_unavailable`:

| Command | Handler | Refused when |
| --- | --- | --- |
| `access.request_grant` | `accessrequests.go` | Always, at proposal |
| `access.approve_request` | `accessrequests.go` | The request is an add request |
| `identity.reinstate_user` | `reinstateUser`, `admin.go:288` | Always |
| `identity.reinstate_membership` | `membershipStep(… "membership_active" …)`, `admin.go:390` | Always |
| `identity.change_role` | `changeRole`, `admin.go:314` | The new role is administrator |
| `identity.create_user` | `createUser`, `admin.go:212` | The role is administrator |
| `identity.invite` | `inviteExisting`, `admin.go:380` | The role is administrator |

Decreases keep working in all three states:
- `identity.suspend_user`, `identity.suspend_membership`, `identity.depart_user`, `identity.remove_membership`;
- a role change away from administrator;
- every revocation.

The check is one server function, `ledgerBlocksIncrease()`, called at the start of each listed handler after its own validation. Command IDs are unchanged.

**The read** reports the state, `ledger_id`, `created_at`, the record count, the watermark, the time of the check, the problem line, any recovery, a replacement, and the relative file name. It never shows the absolute path.

## Restart: durable ledger with in-memory state and the synthetic seed

The ledger survives a restart. Grants, requests, revocation records and sessions do not: they are in memory and rebuilt by the seed.

`Start` runs these steps before the listener serves, as `retention-access.md:98` requires ("re-apply before opening the API"):
1. **Seed.** The seed grants come back with their fixed IDs and are open.
2. **Open the ledger:** take the lock, run the integrity check and torn-tail recovery, then the marker and isolation rules.
3. **Replay.** For each verified decision record of this workspace whose target grant or delegation exists and is open, close it. This is fail closed: a seed grant revoked before the restart is revoked again before anyone can use it.
   - Replay never closes a client grant or an `administer_installation` grant, even if a record names one (N-L3). Such a record is shown as "Not re-applied: protected grant". The M1 guards mean the product never writes one.
   - Records whose target no longer exists are left as they are. Grants made through requests had random IDs and are gone with the in-memory state.
   - In isolation, nothing is replayed from the failed file. Its targets keep their seed state, and the isolation line says so.
4. **Rebuild** up to `access_revocations_kept` revocation records from the most recent decision records, with origin `ledger_replay` (review L6).
   - The read gives each one `recorded_at`. M03 labels them "From an earlier server run".
   - Their state is `revocation_complete`. The cleanup items are:
     - C1, C2 and C6: done, because the restart ended every stamp, session and in-flight request;
     - C3, C4 and C8: not applicable;
     - C5 and C7: outside the controlled scope.
   - Each record also says "Re-applied at start", "Target not present after restart" or "Not re-applied: protected grant".
   - Request IDs from the earlier run are shown as text, not links.
5. **Counters** (N-L4). The counters for revocation IDs (`revocation-<n>`) and request IDs (`access-request-<n>`) continue after the highest number found in the ledger. A new ID never collides with a rebuilt one.
6. **Requests** from before the restart are gone. The read returns `started_at`, and M03 says: "Approval requests made before the server started at … are not kept."

**Additions are not recorded.** Grant additions are not written to the ledger, and granted access does not survive a restart. That is the same for every pre-launch store (R3). The ledger records denials, which is what the spec requires to be durable.

## Cleanup inventory (designated scope)

Cleanup runs only after `deny_confirmed`. Each item records `done`, `not_applicable` or `outside_scope`, with a reason. `revocation_complete` means every item in the designated scope is done or not applicable. The items outside the controlled scope are listed apart and never counted as cleaned (`retention-access.md:59`).

| Item | Today | Recorded as |
| --- | --- | --- |
| C1 Decision stamps | The deny bumps the generation; old stamps fail `Recheck` (`auth/registry.go:135`, `auth/types.go:265-272`) | done (by the deny) |
| C2 Console sessions | Sessions hold no permission; every reader request decides again (`readers.go:121-126`) | not applicable |
| C3 Service credentials and API keys | Grants go to human subjects only (`access.go:129-143`); console client grants are never changed by a command | not applicable |
| C4 HTTP caches of reader responses | This ticket sets `Cache-Control: no-store` on every proxied reader response (`readers.go:92-99`) | not applicable after this ticket |
| C5 Copies already shown in an open page or downloaded | Cannot be recalled | outside scope |
| C6 In-flight reader requests | A request admitted before the deny completes is not cancelled | outside scope (done after a restart) |
| C7 Jobs started under the revoked access | No job runner is wired into `apps/api` | outside scope |
| C8 Search indexes, projections, exports | None is keyed to grants | not applicable |
| C9 Delegations that depend on the revoked grant or delegation | The chain rules end them at the next decision (`decide.go:40`, `:63`); the listing shows them as ended | done (listed by ID) |

## Duplicate-grant fix

**Server.**
- At most one pending request exists per subject and pair.
- An add is refused with `invalid_stage` while an active equivalent grant exists. This is checked at proposal and again in the basis at approval.
- A request applies at most once: the transition to `applied` happens under the same lock as the apply callback.

**Console.** After a `partial` (an internal error with an operation ID), the form shows "Outcome unknown. Check Approval requests below." and reloads. A resubmit by the same proposer returns the same pending request.

**Regression test.**
- On main, two `access.grant` calls with different keys create two grants.
- After this ticket, two `access.request_grant` calls give one request.
- Two approvals, the second after an injected fault following the apply (`s.faulty("access-approve")`, the mechanism of `access.go:399` and `admin.go:78`), give one grant.

## Root amendments after the security change review (2026-11-18)

These take precedence over earlier text in this plan where they differ.

1. **Final newline lost (review m1).** A torn tail that decodes as a complete record, chains to the previous record and passes its `sum` is not truncated. The ledger is isolated with `integrity_failed`; a confirmed record is never discarded as torn.
2. **Replay errors (m2).** A revoke that fails during replay refuses start with an operator error. It is never labelled `reapplied`.
3. **Grant row state (m3, public contract).** A grant closed by a revocation whose decision is not confirmed reads `revocation_unconfirmed` in `grants[].state`, not `revoked`, so an API-only client does not read the target as revoked. The console reads the server state directly.
4. **Delegations (m4).** Unconfirmed delegation revocations are handled like grants in the console: they stay in the open list with the unconfirmed sentence.
5. **Isolated ledger replays the verified prefix (D1).** When the ledger is isolated, revocations from records that verified before the first failing line are still re-applied at start. This fails closed: a revoked grant never comes back because a later line is damaged. The read still shows `isolated` and its reason, and no new record is written until the operator acts. This is a root default under the owner's direction (defaults, changeable) and is listed for owner review.
   - The record on the first failing line, and every record after it, is not re-applied, so those revocations come back while isolated. The exception is a confirmed final record that lost only its newline: it passed every check and is re-applied.
   - A ledger whose marker names another ledger replays nothing, even when it also has a damaged line.
   - `-accept-new-decision-ledger` starts from an empty prefix, as documented.
6. **Round-2 review (2026-11-18).** Unconfirmed revocations read `revocation_unconfirmed` on delegation rows as well as grant rows (an additive enum value on `grants[].state` and `delegations[].state`).
7. **Durable before replay (from the HYB-297 design review).** At start the ledger calls `Sync` after verification and before any decision is replayed. A `Sync` failure still replays every verified record, because revocations are the restrictive direction, and leaves the ledger `stopped`, so increases are refused and no new record is written. The read labels the replay "replayed from records not confirmed durable". (Corrected 2026-11-18 after the HYB-297 design review round 3: not replaying would bring revoked grants back.) fsync error semantics are OS-dependent (residual).
8. **Copy after the terminology and render reviews (2026-11-18).** Copy must never state what the server did not report.
   - **Ended delegations:** the ended-delegation line reads "Ended: the access behind it no longer applies". This replaces "was revoked" in the Console section.
   - **No eligible approver:** a request with no eligible approver shows only the no-approver sentence, not the spec's blocked causes.
   - **EN:** the section heading is "Access change requests", the glossary term.
   - **KO:**
     - The form and button read "권한 추가 요청".
     - The out-of-date sentence keeps the spec text and adds "반려하거나 철회한 뒤 다시 요청하세요."
     - The watermark reads "결정 워터마크".
   - **AR defaults:**
     - الإبطال for revocation;
     - بصمة الطلب for the request revision digest;
     - دفتر القرارات for the decision ledger, with قيد/قيود for its records;
     - بانتظار الاعتماد for pending;
     - counted plurals through the shared helper.
   - These words remain open for owner and native review.
9. **Residuals (recorded, not fixed here):**
   - m5: delegation `Effective` checks backing over the whole scope, not per resource; it must be fixed before delegation creation ships.
   - m6: there is no single-process lock while the ledger is missing or isolated, and the lock is released briefly during replacement.
   - n1: the marker file is read without an owner or mode check.
   - n2: the digest does not include the request number.
   - Windows locking, opening and durability are verified by cross-build only, and directory sync is a no-op on Windows.

## API

**Removed commands:** `access.grant` and `access.revoke_grant`. This is a public contract change. The console, the tests and the OpenAPI fixtures change together.

**Commands.** All use the permission `administer_installation` on `installation`, `Mutating: true` and `Rationale: true` (at most 256 bytes, `access.go:326`).

| Command | Target | Fields | Result |
| --- | --- | --- | --- |
| `access.request_grant` | `grants` | `subject`, `action`, `kind`, `fields`, `customer`, `days` (`access.go:412-413`) | `completed`, state `requested`, fields `request`, `digest` |
| `access.request_revoke` | grant ID | — | `completed`, state `requested`, fields `request`, `digest` |
| `access.approve_request` | request ID | `digest` (text, required) | add: `completed`, state `granted`, fields `request`, `grant`. Revoke: `accepted` (202), job ID = revocation ID, fields `request`, `revocation`. |
| `access.reject_request` | request ID | — | `completed`, state `rejected` |
| `access.withdraw_request` | request ID | — | `completed`, state `withdrawn` |
| `access.revoke_delegation` (kept ID) | delegation ID | — | `accepted` (202), job ID = revocation ID, field `revocation` |

**Why revocations answer `accepted`.** The answer separates the receipt ID from the completion state (`M03/spec.md:78`). The final state, `revocation_complete` or `decision_unconfirmed`, is read from the access read. This holds even when the commit succeeded within the same command, so that one contract covers both outcomes and a `completed` answer never stands for a revocation (`:84`).

**Request IDs** are `access-request-<n>`, with the canonical number rules of `policy.go:165-172`.

**`Resolve`:**
- the workspace must be `s.bootWorkspace`;
- the request must exist in that workspace;
- grant and delegation targets resolve as today (`access.go:257-277`);
- the revision is `s.Revision()`;
- anything else is `NotFound`, which the executor answers as `permission_denied`.

**Refusals**

| Condition | Code |
| --- | --- |
| Field validation, including a stale `days` and a wrong `digest` | `invalid_request` with the field |
| The proposer is the subject or the grant holder; the approver or rejecter is the proposer; the approver of an add is the subject; the rejecter is the holder of a grant being revoked (H1); the revoke target is a client grant or an `administer_installation` grant (M1); the grantor does not hold the pair (M5); a withdraw by a non-proposer; a non-human actor; a pending-request limit is reached | `prerequisite_failed` |
| Request not pending; basis changed (records superseded); active equivalent grant; another pending request for the target; grant or delegation already closed | `invalid_stage` |
| Unknown or other-workspace request, grant or delegation | `permission_denied` (via `Resolve`) |
| Auth registry or roster unavailable | `dependency_unavailable` |
| Ledger `isolated`, `stopped` or `full`: a privilege increase (see "Privilege increases while the ledger cannot record") | `dependency_unavailable`; the read's `ledger.state` names the reason |
| Ledger `isolated`, `stopped` or `full`: a revocation | Not refused. It answers `accepted`, the deny is applied in this process, and the record reads `decision_unconfirmed`. |
| Expected revision differs | `revision_conflict` (executor) |
| Same key with a different payload | `idempotency_conflict` (executor) |

A ledger failure is not a command refusal. The deny has been applied, so the command answers `accepted`, and the record shows `decision_unconfirmed`. A deny failure is a refusal with no record (M2).

**Journal and logs.**
- The common command journal records every call (`execute.go:240-241`).
- The module keeps the business record: proposer, approver or rejecter, times, digest, rationales and the result reference.
- The ledger keeps the durable decision record.
- Log lines carry IDs, codes and ledger states only, never rationale text.
- A ledger failure logs one error line: `decision_ledger_failed`, with the state and the sequence.

**Read.** `GET /api/v1/identity/access` adds:
- `you`, `started_at`;
- `settings[]`: `name`, `value`, `unit`, `default`, `min`, `max`, `source`;
- `requests[]`:
  - `id`, `change` (`add` | `revoke`), `subject`, `grant`, `action`, `kind`, `fields`, `customer`, `days`;
  - `digest`, `state`, `current`, `proposer`, `proposed_at`, `expires_at`, `reason`;
  - `decided_by`, `decided_at`, `decision_reason`, `result`;
  - `eligible_approvers` (a count), `can_approve`, `can_reject`, `can_withdraw`;
- `revocations[]`:
  - `id`, `origin` (`this_process` | `ledger_replay`), `recorded_at`, `target_kind`, `target`, `target_present`, `state`;
  - `ended_delegations`, `deny_generation`;
  - `job_state` (`running` | `waiting` | `succeeded`), `job_reason`;
  - `requested_by`, `approved_by`, `received_at`, `deny_at`;
  - `receipt` (`sequence`, `watermark`, or `reason`), `verified`;
  - `cleanup[]` (`item`, `state`, `reason`), `completed_at`;
- each delegation row gains `effective`, and its `state` can now be `ended` (M3);
- `ledger`: `state` (`ok` | `recovered` | `isolated` | `stopped` | `full`), `isolation_reason` (`missing` | `integrity_failed` | `id_mismatch`), `ledger_id`, `created_at`, `replaces`, `records`, `watermark`, `checked_at`, `problem_line`, `recovery` (`at`, `truncated_bytes`), `file` (`decisions/ledger.jsonl`).

Pending requests come first, then ended requests, newest first, up to `access_ended_shown`. Everything is read under `s.domain.RLock` (`access.go:147-148`).

**Tenant checks.**
- Every read and command needs `administer_installation` in the installation workspace.
- The subject must be in the returned `subjects` set.
- The workspace comes from the session, never from the body.
- Ledger replay applies only records of this installation and workspace.
- Tests cover cross-workspace request IDs and tampered IDs.

## Console

`Live.tsx` is 519 lines today. New files keep each file to one responsibility:
- `Requests.tsx`: the request section and the approve, reject and withdraw panels;
- `Revocations.tsx`: the revocation section and the ledger line;
- `Settings.tsx`: the read-only settings disclosure;
- `request-copy.ts`;
- `request-model.ts` and its test.

**Section order.**
1. Grants to people.
2. Approval requests, with the Approval settings disclosure.
3. Revocations.
4. Console ceiling.
5. Delegations.

**Grant form.** The title and button become "Request access". Success says "Request sent. Another administrator must approve it."

**Request revocation.** The grant row's button becomes "Request revocation". While a revoke request is pending, the row says "Revocation requested. Access stays active until it is approved."

**Approval request section.**
- Rows show, in words, the change, the person, the access, the scope, the days, the proposer, the expiry, a short digest, the state and the eligible approvers.
- Ended requests sit in a disclosure, with the note about requests before `started_at`.
- **Approve panel.** It shows the full fixed content, with an explicit "Adds" or "Removes" scope line (`M03/spec.md:83`), the digest and a required reason. It is shown only when `can_approve` is true.
- **Reject panel.** It requires a reason.
- **Withdraw.** Proposer only.
- An out-of-date request shows the spec `error` copy and offers only reject and withdraw.
- No eligible approver shows the `blocked` copy.

**Revocation section.**
- Each record shows the state words with the spec copy, the job state, "Re-applied at start" or "Target not present after restart" for replayed records, and the cleanup items in three groups: done, not applicable, and outside the controlled scope.
- An expert detail disclosure shows the sequence and watermark (`:66`).
- A ledger line sits above the list.
  - Normally it reads "Decision ledger: N records, last record …", with the ledger ID and creation date in the expert detail.
  - It names a torn-tail recovery when there was one.
  - It names isolation, a stopped ledger or a full ledger in words, with the problem line and what stays available.
  - After a replacement it says that decisions recorded under the replaced ledger are no longer re-applied, so seed access revoked earlier is active again.
  - The hash chain is described as "detects damage to the file", never as protection.
  - The request form and the approve panel are disabled with that reason while additions are refused.

**States reachable in the browser:**
- `revocation_complete`: any approved grant revocation.
- Isolation and `decision_unconfirmed`: start the server on a ledger file whose chain was altered, with the marker present, then approve a grant revocation.
- Recovery: start the server after cutting the last line in half.
- `decision_unconfirmed` is not reachable in the browser without fault injection. It is rendered in console unit tests and asserted in API tests.
- Replayed records: after a restart.

`received`, `deny_confirmed` and `cleanup_running` are rendered in console unit tests from fixtures.

**Delegation impact.**
- The delegation revoke panel lists the descendant delegations it will end, for the "subsequent delegation impact" confirmation (`:84`), before the reason field.
- The approve panel of a grant revoke request lists the delegations that may end with it (M3).
- Each revocation record lists the delegations that actually ended.
- Delegations ended by the chain appear among ended delegations as "Ended: the access behind it was revoked", never as active.

**Errors.**
- `invalid_stage` and `prerequisite_failed` reload the page and explain the current state of that target.
- `partial` shows "Outcome unknown".
- The command client also returns `status` and `job_id` (`admin.ts:392-404`).

**Presentation.**
- State is shown in words and weight, never colour alone, and with no decorative borders (`usability.md`; `service-screen-style.md`).
- After any submit, focus moves to the affected row or section heading, and the live region announces the state.
- Panels return focus to the invoking button (`Live.tsx:416-419` pattern).
- Below 1280 px, panels open as a step.

**Copy.** EN, KO and AR.
- KO uses the spec state copy verbatim. EN and AR are drafts pending the technical-writing review and native-language review.
- New glossary rows: "access change request" (KO 권한 변경 승인 요청), "request revision", "revocation", "decision record not confirmed", "decision ledger", "watermark" and "installation setting".
- Each row records its forbidden implications. For example, "decision unconfirmed" means neither "access still allowed" nor "revoked", and "outside the controlled scope" does not mean "cleaned".

# Dependencies

- **HYB-293 (step-up, in progress).**
  - Its rule is `s.mutating[command] && !exempt[command]` with an empty exempt set, and its coverage test enumerates the registry (`/private/tmp/hb293/docs/features/HYB-293/plan.md:201-205`, `:506-510`). The five new commands are covered without change.
  - The ticket that merges second rebases. The command count goes from 63 to 66. The Q6 checks add no command and change no command ID, so they do not change the count.
  - If HYB-293 merges first, the API test fixtures need a fresh authentication, and the console panels use the shared `stepUp` hook.
- **Persistence of grants and requests.** Not available (R3). The restart behaviour above is designed for it and states it.

# Verification plan

**`modules/accesschange`**
- Unit tests:
  - every transition and refusal;
  - equal-proposal return and one pending request per target;
  - lazy expiry at the boundary (exactly at `expires_at` and 1 ns before);
  - a basis change gives `superseded`, and a wrong digest is refused;
  - an apply failure leaves the request pending;
  - a test ledger answering committed, failed, unknown and a non-advancing receipt (which gives `commit_unknown`);
  - a deny failure creates no record and leaves the request pending (M2);
  - cleanup runs only from `deny_confirmed`;
  - limits taken from settings, including invalid limits being refused;
  - a `decision_unconfirmed` record is never pruned, and a revocation succeeds at and over the record limit (M4);
  - the refusal order of L1, with one case per pair of adjacent checks;
  - no state change on any refusal path.
- Module-level concurrency under `-race`, because the domain lock serialises the HTTP path:
  - concurrent approve, reject and withdraw on one request, where exactly one wins;
  - concurrent proposals for one target, where one stays pending.
- Property tests over random operation sequences against a small model:
  - no request is applied by its proposer, or for an add by its subject;
  - apply is called exactly once per applied request;
  - states move only along the graph;
  - at most one pending request exists per target;
  - no revoke request is rejected by the holder of its grant (H1);
  - no revocation reaches `deny_confirmed` or later without an advancing receipt.
- Source test: stdlib imports only, no comments, no binary fractions, at most 1,500 lines per file.

**`modules/decisionledger`**
- Fault injection through an internal file interface (`Write`, `Sync`, `Close`):
  - a write error gives `commit_failed`;
  - a short write gives `commit_failed`;
  - a `Sync` error gives `commit_unknown`;
  - after any of these the ledger is `stopped`, and the next commit is refused without touching the file;
  - an append past the file bound is not written, and the ledger is `full`.
- Torn tail (N-M1):
  - a final segment without a newline is truncated to the last boundary, synced and followed by a `recovery` record, and the state is `recovered`;
  - a complete but altered last line is `integrity_failed`, not recovered.
- Corrupted chain, each detected with its line number:
  - a byte flip, a deleted line, reordered lines, a duplicated sequence;
  - an unknown field, a missing or misplaced genesis record, an oversize line.
- Marker and isolation (N-H1):
  - first start creates the genesis record and the marker;
  - a ledger without a marker writes the marker;
  - with the marker present, a missing ledger, a failed check or a different ledger ID each give `isolated`;
  - `-accept-new-decision-ledger` keeps the old file under a new name, writes a genesis record with `replaces` and the reason, and rewrites the marker;
  - the same flag with nothing to replace refuses `Start`.
- File bound: a setting below the current size plus one record refuses `Start`.
- Restart reread: append, close, reopen; the record count, ledger ID and watermark are equal, and the next sequence continues.
- Lock order: the integrity read and every append use the locked handle; a second open fails before reading (N-M3).
- Unix opening: a symlinked file is refused through `O_NOFOLLOW`, a hard-linked file through `Nlink`, and a `0644` file through `Fstat` (N-L1).
- Directory sync after creating `decisions/`, the file and the marker, checked through an injected directory syncer (N-L2).
- Permissions on macOS and Linux:
  - a `0755` directory, a `0644` file and a symlinked file or directory are each refused;
  - a created file is `0600` in a `0700` directory.
  - Another owner cannot be produced without root, so that case is recorded as UNVERIFIED.
- Concurrent appends under `-race`: N goroutines produce strictly increasing sequences and a valid chain on reopen.
- Lock: a second open of the same file fails with `ErrLocked` on macOS and Linux, run on this machine. The Windows lock and permissions are covered by cross build only; their tests are UNVERIFIED unless a Windows runner is available.
- Property test: any sequence of appends reopens to the same watermark, and any single-byte mutation of a non-empty file is detected.
- Source test as above.

**`apps/api`**
- Permissions: a member gets 403 on the read and on each command.
- Flows with the sample seed:
  - `admin` proposes for `member`, `admin2` approves, and the member reads the collected-data route;
  - reject and withdraw;
  - an add for `admin2` reports no eligible approver;
  - a revoke approved by the holder ends in `revocation_complete` with sequence 1;
  - a delegation revoke gives `accepted`, `revocation_complete` and its descendants.
- Every refusal row, the duplicate-grant regression, and tampered or other-workspace IDs.
- A request superseded by each basis change:
  - the ceiling;
  - the subject's membership, epoch or state;
  - the proposer's role, membership or administration grant (M5, L9).
- `request_revoke` on `reader-console` and on `installation-<SampleAdmin>` is refused at proposal. A revoke request whose target becomes such a grant is refused at approval (M1).
- Grantor coverage: a proposer whose own reader grant was revoked cannot propose that pair, and an approver without it cannot approve (M5).
- Chain-ended delegations (M3), using test delegations:
  - revoking the delegator's backing grant lists the delegation in `ended_delegations` and shows it as `ended`;
  - revoking the delegate's only grant does the same;
  - a revoked delegation's descendants show as `ended`.
- The holder's reject is refused, and the holder's approve succeeds (H1).
- Restart: revoke `reader-admin2` and complete it, then build a new `Server` on the same data directory:
  - the grant is closed before `Start` returns;
  - the record is listed with origin `ledger_replay`;
  - a request-made grant's record says the target is not present.
- Ledger failures:
  - a corrupted file with the marker present gives `isolated` in the read. Then:
    - grant-addition proposals and approvals answer `dependency_unavailable`;
    - a revoke request can be proposed and approved, and `access.revoke_delegation` runs; each answers `accepted`, the target is closed, and the record reads `decision_unconfirmed` with `ledger_isolated`;
    - no ledger file is created or written (directory listing and modification times unchanged);
    - rejects, withdrawals and reads work;
    - after a further restart still in isolation, that revocation is not replayed;
  - an injected write fault gives `stopped`: a grant addition is refused at proposal and at approval, and a revocation applies the deny and reads `decision_unconfirmed` with `ledger_stopped`;
  - a full ledger behaves the same with `ledger_full`, including a ledger within one record of the bound at `Start`, which starts `full` instead of refusing to start;
- Q6 increases (R3-M2). In each of `isolated`, `stopped` and `full`, every command in the increases table answers `dependency_unavailable`. `create_user` and `invite` with role member, and every decrease, still succeed. Tests in `users_test.go`.
  - a held lock makes `Start` fail.
- Replay never closes `reader-console` or an `installation-*` grant named in a hand-written record (N-L3).
- After a restart, new revocation and request IDs continue after the highest numbers in the ledger (N-L4).
- Ledger records contain no rationale or login (byte search over the file).
- `Cache-Control: no-store` on proxied reader responses.
- OpenAPI and route registration.
- AST test (review L3): in non-test files of `apps/api/server`, `AddGrant`, `RevokeGrant` and `RevokeDelegation` are called only from:
  - the approve path;
  - the revocation path;
  - the ledger replay;
  - the seed (`seed.go:219-246`).
  - Test files and the exported `Administration()` accessor (`server.go:286`) are explicitly allowed, and the test lists them by name.
- `-race` tests over HTTP: a read during a revocation, and an approve against a concurrent revoke of the covering grant. Because the domain lock serialises commands, the contention tests live in the modules (above).
- Leak test: logs contain no rationale.

**`apps/api/cmd/hb-api`**
- Every setting flag:
  - its default with source `default`;
  - a valid value with source `start_flag`;
  - out-of-bound, signed, hex, leading-zero and over-length values exit with an error before listening;
  - relation violations (workspace below proposer, shown above kept) exit with an error;
  - `-decision-ledger-max-mib` below the current file size exits with an error. A bound within one record of the size starts with the ledger `full`.

**Console**
- Unit tests for the request and revocation models and copy:
  - ordering, state text for all five revocation states and the replay origin, and the item grouping;
  - `can_*` gating, including `can_reject` false for the holder;
  - the attempt kept after `partial`;
  - accepted-result parsing;
  - the two idempotency cases named apart: a resend with the same key and payload shows the original result (`replay: true`), while the same key with a different payload shows the 409 `idempotency_conflict` message;
  - settings source text.
- Typecheck and build.

**Coverage, mutation and builds**
- 100% coverage of the changed Go code.
- Mutation testing of both new modules and the new server files, cleaning the session GOCACHE between batches.
- Cross builds for linux, darwin and windows.

**Browser QA** on the production build against a live API with the sample seed, using two browser contexts (`admin`, `admin2`):
- propose, approve, reject, withdraw;
- an out-of-date request;
- no eligible approver;
- a grant revocation reaching `revocation_complete` with sequence and watermark;
- a restart showing the replayed record and the not-kept requests note;
- a restart on an altered ledger showing isolation: grant additions disabled with the reason, a grant revocation approved and shown as `decision_unconfirmed` with the "this process only" line, and the M01 refusal message for reinstating a user;
- a restart after cutting the last line in half, showing the recovery;
- a restart with `-accept-new-decision-ledger`, showing the replacement note;
- the Approval settings disclosure with one value set by flag;
- the effect preview note;
- EN, KO and AR at 1440 and 390;
- keyboard order, focus return and computed borders.

Delegation revocation is not reachable in the browser, because no delegations are seeded. Expiry is unit-tested only.

**Reviews**
- The security design review of this plan (in progress for the previous draft).
- The `ui-designer` comparison.
- The technical-writing review.
- An independent change review.

# Root decisions on the questions (2026-11-17)

- **Q1.** No emergency revoke. Owner-deferred (D-O3). Grant revocation goes through an approval request, as `M03/spec.md:83` writes it.
  - An immediate single-person cutoff of a person still exists through `identity.suspend_user` and `identity.suspend_membership` (M01). Suspension is checked from the next permission check onwards (`retention-access.md:61`), and it ends every grant's effect at once.
  - The remaining delay, for removing one grant without suspending the person, is residual R9.
- **Q2.** A grant holder may approve the revocation of their own grant, but may never reject it (review H1). Revocation only reduces privilege, and the approver still differs from the proposer.
- **Q3.** The deny is applied in-process before the ledger commit (fail closed). The state reads `decision_unconfirmed`, never success, whenever the commit fails, is unknown or is refused.
- **Q4.** Delegation revocation stays limited to installation administrators (review L4 wording). What is missing for the spec's "original delegation management authority" (`M03/spec.md:84`) is a delegation-management permission and delegation creation, not a sign-in path. Residual R4.
- **O-1, O-2.** Installation settings with defaults (D-O1).
- **O-3.** Resolved as a local ledger file (D-O2).
- **Q5 and Q6** (2026-11-18, binding after security reviews 2 and 3). In `isolated`, `stopped` and `full`:
  - every privilege increase listed under "Privilege increases while the ledger cannot record" is refused;
  - revocations are applied in this process only and read `decision_unconfirmed`.
  - Revocation never fails open.
- **O-5** (2026-11-18, narrowed by review N-M2). Only the file bound is an installation setting, `-decision-ledger-max-mib`. The 2 KiB record and the 4 KiB line stay constants, because the record format fixes them.
- **Windows locking** (2026-11-18, settled by review N-M3). `LockFileEx` is reached through `syscall.NewLazyDLL("kernel32.dll")` in the standard library. No `golang.org/x/sys` dependency is added, so no licensing review is needed, and there is no lock-file fallback.

# Assumptions (not verified)

- A1. Nothing outside the console and the tests calls `access.grant` or `access.revoke_grant`. To be checked by repository search at implementation time.
- A2. `identity.MemberInfo.Version` changes when a membership is removed and recreated. To be confirmed in `modules/identity` before it is used in the basis.
- A3. `no-store` does not break the console's use of the reader responses. `If-None-Match` is forwarded today (`readers.go:72`). To be checked in browser QA.
- A4. `LockFileEx` through `kernel32.dll` and `FlushFileBuffers` behave as documented on a Windows host. Not run here; cross build only.
- A5. The data directory is on a local disk that honours `fsync`, `F_FULLFSYNC` and `FlushFileBuffers`. Network file systems, exFAT and removable media are not a tested combination (N-L2).

# Open values

- O-4. The cleanup scope for keys, jobs and files when those features exist.
- O-6. How long the internal user IDs in ledger decision records are kept, against the user, identity and access audit class of `retention-access.md:76`. Today they are kept as long as the ledger, with no deletion path (N-L6).

# Questions for the owner

None open. Q6 is decided (see Root decisions).

# Residuals

- R1. The ledger is a single local file. There is no off-host independent copy or replication (`retention-access.md:98`), and no backup or restore procedure for it. Four cases are not detectable without one (R3-L1):
  - truncation at a record boundary;
  - rollback of the whole data directory, marker included;
  - the marker and the ledger both deleted, which looks like a first start;
  - the marker deleted and a different valid ledger put in place.

  In the last two, the visible sign is the genesis `created_at` (and `ledger_id`) shown on the M03 ledger line.
- R2. No retry command for `decision_unconfirmed`.
- R3. Grants, requests and sessions are in memory. A restart loses them and recreates the seed. Only revocations persist, through the ledger.
- R4. Support sessions, delegation creation, a delegation-management permission and bulk revocation are not built.
- R5. Cleanup items C5–C7 are outside the controlled scope.
- R6. The KO, EN and AR copy needs native-language review. Screen acceptance (FIX) for M03 is the owner's alone.
- R7. There is no repair tool for a ledger that failed its integrity check.
  - The only exit from isolation is `-accept-new-decision-ledger`.
  - After it, the old decisions no longer replay, and seed access revoked under the old ledger is active again. M03's ledger line says so.
  - The old file is kept under its `replaced-` name for an investigation.
- R8. On Windows, owner-only access depends on the installer's directory ACL, there is no directory sync, and there is no `O_NOFOLLOW` (an `Lstat` and `SameFile` check instead). None of this is verified here.
- R9. Urgent grant revocation waits for a second administrator (D-O3).
- R10. Settings can be changed only by restarting with different flags. There is no UI and no configuration file.

# Review history

**Round 1 security design review** (of the draft with an unconnected ledger and provisional values): CHANGES REQUIRED. All findings are folded into this revision. This revision has not been re-reviewed.

| Finding | Severity | Resolution in this revision |
| --- | --- | --- |
| H1: the holder could reject the revocation of their own grant | High | The holder may approve, never reject; `can_reject` false; `prerequisite_failed`; property test |
| M1: today's revoke guards were missing on requests | Medium | Client and `administer_installation` grants refused at proposal and at approval |
| M2: a deny failure left a `received` record, which contradicted the request state | Medium | Option a: no record, the request stays pending, the named failure is answered; the contradicting text is removed |
| M3: delegations ended by a grant revocation were not shown | Medium | `ended_delegations`, the `ended` listing state, the approve panel list, cleanup row C9, read-only `DelegationView.Effective` |
| M4: unconfirmed records could be pruned, and a limit could refuse a revocation | Medium | Only confirmed final records are pruned; a revocation is never refused for a record limit |
| M5: proposer standing and grantor coverage were missing from the basis | Medium | Proposer membership, role, epoch and administration grant are in the basis; `Explain` coverage at proposal and approval |
| L1: check order | Low | Order defined; refusals change no state except lazy `expired` and `superseded` |
| L2: ledger record content and non-advancing receipts | Low | `deny_generation`, `ended_delegations`; a non-advancing receipt gives `commit_unknown` |
| L3: AST allowances | Low | Seed, tests and the `Administration()` accessor listed |
| L4: wording of Q1 and Q4 | Low | Rewritten |
| L5: job reads and job states | Low | No read by job ID, stated; job state defined for every state; no `failed` job |
| L6: RET-05 after a restart | Low | Earlier-run records labelled; durable deny covered by the D-O2 ledger, not "not connected" |
| L7: editing earlier tickets' records | Low | HB-266 and HB-267 records are not edited |
| L8: ceiling choice | Low | Latest `Until`, then lowest ID |
| L9: subject epoch and state | Low | Added to the basis |
| Nits | — | One non-human mapping (handler, `prerequisite_failed`); 409 versus replay console tests; the preview note keyed to `deny_confirmed`; module-level concurrency tests |

**Round 2 security design review** (of the revision with the ledger file and settings): CHANGES REQUIRED. All findings are folded into this revision, which has not been re-reviewed.

| Finding | Severity | Resolution in this revision |
| --- | --- | --- |
| N-H1: the ledger was described as protection; there was no rule for a missing ledger; the foreign-installation check proved nothing | High | Tamper-evidence wording. Genesis record with a random ledger ID. Marker file and access-change isolation. One-time `-accept-new-decision-ledger`. RET-05 partially covered. Re-opening consequence stated in R7 and on the ledger line. |
| N-M1: torn tail | Medium | Truncate to the last boundary, `Sync`, append a `recovery` record, show it; everything else is `integrity_failed` |
| N-M2: bounds and full or stopped behaviour contradicted each other | Medium | Only the file bound is a setting, checked at `Start`; full or stopped refuses grant additions only; revocations still deny |
| N-M3: Windows lock | Medium | `LockFileEx` through `NewLazyDLL`, standard library only, no fallback; lock taken before the integrity read; one locked handle; range to the maximum offset |
| N-L1: opening checks | Low | `O_NOFOLLOW` and `Fstat` (regular, mode, owner, `Nlink == 1`) on Unix |
| N-L2: directory sync and volumes | Low | The data directory and `decisions/` are synced on creation; A5 limits durability to local disks |
| N-L3: replay of protected grants | Low | Replay never closes client or `administer_installation` grants |
| N-L4: ID collisions | Low | Counters continue after the highest ledger numbers |
| N-L5: scope of the Q5 block | Low | M03 grant additions checked at proposal and at approval; identity increases decided in round 3 (Q6) |
| N-L6: actor IDs kept in the ledger | Low | Open value O-6 against `retention-access.md:76` |
| L8 premise | — | Verified: `listing.go:52` sorts by ID, and `access.go:375` keeps the first maximum |

**Round 3 security design review:** design sound. Two binding root decisions appended after round 2 contradicted the body; they are now stated in the body.

| Finding | Severity | Resolution in this revision |
| --- | --- | --- |
| R3-M1: revocations were refused in isolation (fail open) | Medium | Revocations apply the deny in this process and answer `accepted` with `decision_unconfirmed` (`ledger_isolated`); no file is touched in isolation; only increases are refused; the test is inverted; reachable in the browser |
| R3-M2: Q6 left as a question | Medium | Decided: identity increases, plus `create_user` and `invite` as administrator, are refused in the three states; allowed files and tests extended |
| R3-L1: undetectable marker cases | Low | Added to R1, with the genesis date as the visible sign |
| R3-L2: an unstartable server at the bound | Low | Starts `full`; only a bound below the file size refuses `Start` |
| R3-L3: "commits" wording | Low | Replaced with "is refused" and "is applied in this process only" |

History note: the root decisions of 2026-11-18 ("revocation never fails open" and Q6), first appended after round 2, are folded into the body above.

# Security design review round 4 clarifications (binding, nits)

- Torn-tail recovery at start follows the same bound rule: if appending the recovery record would pass the file bound, the ledger starts in `full` after truncating the torn tail, and the read shows both the recovery and the full state.
- Every identity increase refused by `ledgerBlocksIncrease()` changes no identity state; a test asserts the identity read and revision are unchanged after each refused increase, mirroring the access-change refusal test.
