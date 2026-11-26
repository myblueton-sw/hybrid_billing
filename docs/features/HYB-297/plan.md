---
type: ImplementationPlan
title: M02 trust change workflow with propose, validate, independent approval and activation under a strictly newer trust epoch
description: Replaces the M02 propose-then-approve-applies policy flow with the specified trust change workflow. Six commands (trust.propose, trust.validate, trust.approve, trust.activate, trust.reject, trust.cancel) bind each change to the expected old trust epoch, policy revision and digests, require three distinct people with four distinct trust rights, commit every activation with its session invalidation intent to the installation decision ledger before it takes effect, fail closed on unknown or isolated ledger states, and report invalidation progress separately. Revised after security design review round 1.
status: stable
tags: [HYB-297, security, identity, trust-change, M02, M03, IAM-02, IAM-05, IAM-06, IAM-08]
sources:
  - resource: ../../ui/screens/M02/spec.md
  - resource: ../../ui/screens/M02/design.json
  - resource: ../../contracts/commands/planning-actions.md
  - resource: ../../contracts/security/access-and-trust.md
  - resource: ../../reviews/HYB-291-planning-cross-check/report.md
  - resource: ../HB-265/plan.md
  - resource: ../HYB-293/plan.md
---

# Scope and ownership

[HYB-297](https://linear.app/hybrid-billing/issue/HYB-297) implements the M02 trust change workflow as specified (`docs/ui/screens/M02/spec.md:96-131`). It fixes the HYB-291 finding M2: "the draft, validated, independently approved, activated trust flow is collapsed into propose, then approve, which applies" (`docs/reviews/HYB-291-planning-cross-check/report.md:51`, `:121`, `:147`).

Base: main `be1bfa3` (after HYB-293). Branch: `feat/HYB-297-trust-change-workflow`. Worktree: `/private/tmp/hb297`. Machine: mac_mini. The branch is rebased on main after HYB-298 merges (see Dependency).

**Owner directions in force (2026-11-17, relayed by root).**
- Faithful to the planning: build the spec behaviour as written. Where a dependency is missing, block truthfully and say so. Never simplify silently.
- Provisional values ship as changeable installation defaults.
- Required decision records persist in the installation data directory.
- No emergency revoke path yet. This plan also adds no emergency trust bypass (`M02/spec.md:128`).

**Ownership**
- Root (Claude Code) owns the plan, the implementation, the tests and browser QA.
- A separate read-only reviewer reviews this design and later the change, with a security focus. The change touches permissions, M03 grant paths, authentication policy, session invalidation, durable decision records and the public command contract (AGENTS.md core rules; `design-first.md`). Round 1 returned FAIL; see "Design review round 1".
- The `ui-designer` role (AI, read-only) leads the visual design of the trust change sections and compares the final renders (`service-screen-style.md`).
- A separate read-only technical-writing review checks the new terms and the EN, KO and AR copy (`product-terminology.md`).

## In scope

- New stdlib-only module `modules/trustchange`: the trust change state machine, actor separation, digests, epoch binding and supersession.
- `modules/authpolicy`: revisions can no longer be proposed or approved through `Propose`/`Approve`. A revision is published only through `Publish`, which requires a strictly newer trust epoch and takes the policy number explicitly on replay. Two pure judgement exports for validation.
- `modules/auth` and `modules/gate`: four new installation-kind actions, `trust_propose`, `trust_validate`, `trust_approve` and `trust_activate`, for human administrators only. `administer_installation` confers none of them. An exported predicate `auth.InstallationAction(Action) bool` (re-exported by the gate).
- **M03 grant paths (review B1).** Every M03 path that today excludes only `administer_installation` uses the predicate instead. Files after the HYB-298 rebase: `apps/api/server/access.go` (ceiling, listing, revoke target validation), `apps/api/server/accessrequests.go` (`ceilingFor`), `apps/api/server/ledger.go` (`protectedGrant`, the replay guard), and `apps/api/server/integrations.go` (`:334`, the same pattern, checked and changed if it lists grants).
- `modules/auth` (and a re-export in `modules/gate`): a new read-only export `ExplainInstallation(subject, client, request, now)` for the trust evidence (review N3, R3-5). The existing `Explain` refuses every installation action and takes no client (`modules/auth/listing.go:73-77`), so it cannot serve. The new export applies the installation-action rule and the human-kind rule that auth itself enforces for installation actions (`modules/auth/registry.go:264`), intersects the subject's grants with the client's grants, checks validity at the given `now` (not the registry clock used by `consistent`, `decide.go:274`), skips delegations because installation actions cannot be delegated (`registry.go:375`), and changes no decision. The administrator-role check is not repeated here: the gate's role check (`gate.go:109`) needs an identity answer, so the evidence binds the role from identity facts instead (Validation). Exactly two callers are allowed, checked by an AST test: the gate re-export wrapper, and `apps/api/server/trustevidence.go` through that wrapper. It is never reachable from the M03 preview (`previewAccess`) or any M03 path.
- `modules/commands`: four refusal codes from the planning error table become catalog codes, and `self_approval_denied` becomes a handler code with its existing status and next action.
- `modules/decisionledger` (added by HYB-298): a `trust_activation` decision kind, and a ledger **generation** `g` in the marker and the genesis record (review N7; backward compatible: a marker or genesis without `g` reads 0).
- `apps/api/server`: the six `trust.*` commands replace `policy.propose`, `policy.approve` and `policy.withdraw`. The policy read gains the trust change model. Sessions carry `trust_epoch`. Activation commits to the ledger first, then publishes, supersedes and invalidates. A trust hold fails closed on an unknown activation. Start replays the latest activation.
- The sample seed gains a third administrator and trust-right grants. The bootstrap administrator receives the four rights.
- Console M02 live screen: the trust change list, detail, commands, confirmations and states in EN, KO and AR.

## Not in scope

- SAML, OIDC and LDAP providers, provider test login, key rotation, external status synchronization and their states `error`, `external_status_unknown` and `sync_failed` (not connected; HB-265 `plan.md:177-178`, `:200-208`).
- An emergency trust policy or recovery bypass (`M02/spec.md:128`, owner direction).
- Enrolment of further trust-right holders (granting trust rights through M03 or any command). Trust grants come only from the seed and the bootstrap. This is a truthful block for non-sample installations (review M4; R4; Q3).
- Durability of drafts, validations, approvals, rejections and cancellations of changes that were not activated (provisional deviation, R2, Q4).
- Off-host replication of the ledger (inherits HYB-298 R1).
- Named owners approving proof and recovery procedures (`access-and-trust.md:28`; R10).
- `modules/trust` (HB-237 connection trust for pushes and destinations) is a different concept and is not touched.

# Dependency on HYB-298 (decision)

**HYB-297 is implemented after HYB-298 merges. It extends HYB-298's ledger and its M03 paths; it does not create its own ledger.**

Facts:
- HYB-298 is in progress in `/private/tmp/hb298` on `feat/HYB-298-m03-approvals` (uncommitted changes, `git status` on 2026-11-18). Its plan is `status: draft` and makes the ledger "the decision record store … in the installed environment now" (`/private/tmp/hb298/docs/features/HYB-298/plan.md:25`).
- Its `modules/decisionledger` accepts only `revoke_grant` and `revoke_delegation` decisions (`modules/decisionledger/types.go:23-24`, `record.go:78-80` in that worktree), with a 2 KiB record limit (`types.go:10`).
- It owns `<data>/decisions/ledger.jsonl`, the lock, the marker, isolation, the integrity check and the replay order in `Start` (HYB-298 `plan.md:376-555`).
- Its M03 grant code excludes only `administer_installation` (`apps/api/server/access.go:148`, `:195`; `accessrequests.go:86`; `ledger.go:72-74` in that worktree).

Decision and reasons:
1. **Trust activations are recorded in that ledger.** Without a durable record a restart would bring back the seed policy at an old epoch, a silent rollback the spec forbids (`M02/spec.md:126`). A second ledger would duplicate locking, integrity, isolation and replacement and give two sequences for one installation.
2. **Sequence after HYB-298.** HYB-297 edits `modules/decisionledger` and HYB-298's M03 files. One writer per file (AGENTS.md). Implementation of those steps starts after HYB-298 is merged and read back.
3. **Before HYB-298 merges** only this plan's review and the parts with no contact with HYB-298 files may be written (`modules/trustchange`, the `authpolicy`, `auth`, `gate` and `commands` changes, console models and copy). Nothing merges before the rebase.
4. **If HYB-298 does not merge,** activation stays blocked and the screen says "Activation needs the decision ledger, which this installation does not have yet". B1 still has to be fixed on whatever M03 code exists on main (`access.go:122`, `:161`, `:335` on this base) before the four rights are added; the rights are never added without the M03 guard.
5. **Start-time `Sync` (reviews N6, R3-3).** HYB-298 amendment 7 (`/private/tmp/hb298/docs/features/HYB-298/plan.md:600`, corrected 2026-11-18): `Start` calls `Sync` after verification and before any replay. If that `Sync` fails, every verified record is still replayed, the ledger is `stopped`, increases are refused, and the read labels the replay "replayed from records not confirmed durable". HYB-297 follows it: the latest verified trust configuration is replayed too, every `trust.*` command is refused while the ledger is `stopped`, and M02 shows the same label.
6. **Marker generation (review N7).** HYB-297 extends HYB-298's marker and genesis format with the generation `g` after the rebase (declared scope in `modules/decisionledger`).
7. **Rebase notes (R3-6).** With the B1 guard, an M03 revoke of an `administer_installation` grant also changes from HYB-298's `prerequisite_failed` to the generic `permission_denied` (not found). HYB-298's API tests and the M03 console copy for that refusal are updated in this ticket.

Lock order after both tickets (review m8): `s.domain` (write) → `trustchange` mutex → `authpolicy` mutex → identity boundary mutex → auth registry mutex → `sessions` mutex → `decisionledger` mutex. HYB-298 fixes `s.domain` → `accesschange` → auth → `decisionledger` (HYB-298 `plan.md:264`). The evidence is computed by apps/api before the `trustchange` call, so no identity, gate or session read happens while the `trustchange` mutex is held. The sweep runs after `trustchange.Activate` returns. Nothing calls back up the chain.

Command count after both tickets, enumerated from the HYB-298 worktree registry plus this plan: 73 (identity 12, access 6, service 6, relationship 3, contract 8, installation 3, integrations 15, recovery 4, retention 6, release_evidence 4, trust 6). The exact list is re-enumerated from the running registry at implementation time. HYB-293's step-up coverage test enumerates the registry (`apps/api/server/stepup.go:40`).

# Requirement cross-check

Sources: `M02/spec.md:24-26` (HB-03, HB-36, HB-42; IAM-01 to IAM-08), `design.json` (`requirement_ids`, `acceptance_ids`, `command_contract`), `docs/planning/v1.16/requirements.json`, `detailed-requirements.json`, `acceptance-trace.json:1893-1962` (each IAM ID maps to HB-42, NOT_RUN), `retention-access.md:41`, `:47`, `:49`, `:98`, `:110-117`, `planning-actions.md:52-58`, `:162-164`, `access-and-trust.md:24-30`, `:66-67`. The `docs/planning/v1.16` files are in the authorized local workspace, not yet in Git; their line numbers refer to those files.

Status meanings follow HYB-291 (`report.md:163-169`): **covered**, **deferred**, **not connected**, **not applicable**, and here **blocked** (built, but cannot complete in a given installation until an owner decision). No acceptance test has run.

| ID / source | Requirement (summary) | How HYB-297 satisfies it | Status |
| --- | --- | --- | --- |
| `M02/spec.md:96`, `planning-actions.md:52` | `trust.propose`: current revision and expected old epoch; draft only; confirmation shows proposed scope and old/new digests | `expected_epoch`, `expected_policy`; result `draft`; confirmation C1 | covered |
| `M02/spec.md:97`, `planning-actions.md:53` | `trust.validate`: draft at expected revision; validated or blocked evidence; confirmation shows validation scope and evidence | Findings recorded; confirmation C2 | covered for local password (Validation) |
| `M02/spec.md:98`, `planning-actions.md:54` | `trust.approve`: validated, digest, current old epoch, approver ≠ proposer; confirmation of exact digest, authority and session/lockout impact | Binds both digests; `self_approval_denied`; confirmation C3 | covered |
| `M02/spec.md:99`, `planning-actions.md:55` | `trust.activate`: approved digest, old epoch, separation and authority rechecked; atomic publication, newer epoch, durable invalidation intent; confirmation of digest, affected principals, session/lockout impact | Ledger commit first, then publish at a strictly newer epoch; confirmation C4 | covered (needs HYB-298) |
| `M02/spec.md:100-101`, `planning-actions.md:56` | `trust.reject`, `trust.cancel` with rationale, before activation only | Both, with confirmations C5 and C6 | covered |
| `M02/spec.md:115-120` | States draft … cancelled | Rendered with the spec copy | covered |
| `access-and-trust.md:26` | Superseded terminal state | `superseded` | covered |
| `M02/spec.md:110` | `blocked`: test login or recovery unverified; hold activation | Validation `blocked`; enrolment block copy | covered |
| `M02/spec.md:111`, `:113-114` | `error`, `external_status_unknown`, `sync_failed` | No provider or external identity | not connected |
| `M02/spec.md:112` | `revoked` clears sensitive detail | On `permission_denied` or `unauthenticated` the list, detail and selection are cleared | covered |
| `M02/spec.md:108-109` | `empty`, `partial` | `partial` for earlier-run changes not kept and for an unusable ledger | covered |
| `M02/spec.md:126` | Proposal binds route, epoch, revision, digests, identity binding, proofs, diagnostics, recovery, external status, session impact, affected subjects | Local password items; issuer, diagnostics and external status `not_applicable` | covered for local password; rest not connected |
| `M02/spec.md:126` | Candidate or evidence change invalidates validation and approval | Revision resets to draft; evidence digest binds user state, user epoch, kind and effective gate decisions | covered |
| `M02/spec.md:126` | Rollback is a new change with a newer epoch | No restore command; a replacement ledger raises the generation, so epochs never restart (N7) | covered locally (R1) |
| `M02/spec.md:128` | Distinct rights; no self-approval; activator authority rechecked; no emergency bypass | Four rights; three people; gate recheck on every command and replay | covered; **blocked** in non-sample installations until enrolment is decided (M4) |
| `M02/spec.md:130` | Atomic commit of new epoch and durable invalidation intent; progress and unknown completion shown | Ledger record carries the intent; per-item progress; trust hold on unknown | covered |
| `M02/spec.md:130` | Concurrent proposals cannot both activate | Domain lock, module check, `Publish` strictly newer | covered |
| `M02/spec.md:130` | Idempotent retry rechecks authorization; mismatched payload conflicts | Executor order (`modules/commands/execute.go:324-351`) | covered (regression) |
| `M02/spec.md:130` | Stale evidence needs fresh review | Superseded or explicit re-validation and re-approval | covered |
| `M02/spec.md:130` | Reject/cancel retain reasons and history | Kept in memory only; lost on restart | **provisional deviation** (R2, Q4) |
| `M02/spec.md:132` | Acceptance list | API tests for each; external unknown not connected | covered except external unknown |
| `M02/spec.md:93` | Request policy activation through the trust workflow | Proposal form posts `trust.propose` | covered |
| `M02/spec.md:63`, `retention-access.md:47` | MFA and reauthentication evidence in the revision | Unchanged from HYB-293; validation blocks an unmeetable MFA requirement | covered (MFA not connected) |
| `M02/spec.md:65`, `retention-access.md:49` | Recovery path verified; last administrator; rollback does not revive suspended accounts | Recovery finding as defined in Validation; suspended and not-yet-activated accounts never count | covered for local password |
| `M02/spec.md:143` | Writes check authority, expected revision, idempotency key | Executor plus trust fields | covered |
| `M02/spec.md:149` | Unknown outcome is `waiting` with a reason; retry only where safe | `activation_unknown` answers `accepted` with job state `waiting`, no retry | covered |
| HB-03 | Separate authority; author ≠ approver; no leak on tampering or revocation | Distinct rights, three people, generic not-found; trust rights cannot be granted or revoked through M03 (B1) | covered for trust changes |
| HB-36 | Installation operation and support | Ledger in the data directory; no support sessions | durable record covered; support deferred |
| HB-42 | Identity and access lifecycle, immediate block | Activation and the trust hold end sessions before the next request | covered for password sessions |
| IAM-01 (`retention-access.md:110`) | Bootstrap and invitation replay | Bootstrap now also grants trust rights; regression run of the bootstrap tests | regression (behaviour otherwise untouched) |
| IAM-02 (`:111`) | Local, SAML, OIDC, LDAP; no SSO bypass | Local password only | not connected |
| IAM-03, IAM-04 (`:112-113`) | Identity reuse; token validation | No federated identity | not connected |
| IAM-05 (`:114`) | Suspension and permission loss reach sessions | Trust activation and hold end sessions; suspension changes evidence | covered for trust changes |
| IAM-06 (`:115`) | Recovery and last-administrator protection not a bypass | No bypass; recovery finding blocks lockout candidates | covered within pre-launch |
| IAM-07 (`:116`) | 100k-user isolation | Not built | deferred |
| IAM-08 (`:117`) | Sensitive commands need MFA and reauthentication policy | All `trust.*` step-up (HYB-293) | covered for password sessions |
| AUD-QA-04 | Idempotency, 409, stale revision; approval request ≠ human approval | Executor rules; human only | covered (OAuth not connected) |
| AUD-AR-09 | Delegation and support identity | Trust rights are installation kind and not delegable | not applicable |
| `retention-access.md:98` | Durable commit before confirmation; monotonic sequence and epoch; replay before serving; isolation on missing or unverifiable ledger | `Sync` before `activated`; epoch per record and generation in the marker and genesis; replay in `Start`; trust commands refused while isolated | covered locally; independent copy deferred (R1) |
| `access-and-trust.md:28` | Legitimate-authority proof; named owners approve proof and recovery procedures | Proof is the validator's password session only; no named owner approval | deferred (R10) |
| `access-and-trust.md:30` | New epoch; old sessions, step-up evidence and queued privilege cannot proceed; propagation failure fails closed | Session `trust_epoch` check, sweep, trust hold on unknown | covered |
| TR-01 (`access-and-trust.md:66`), TC-01 (`planning-actions.md:162`) | Self-approval or generic admin activation fails | `self_approval_denied`; `administer_installation` grants no trust right; M03 cannot grant one | covered |
| TR-02 (`:67`), TC-02 (`:163`) | Newer epoch and durable invalidation; old-epoch work blocks even if propagation lags | Epoch check in `access` and under the lock in `execute` | covered |
| TC-03 (`:164`) | Changed proof or old epoch, concurrent replacement or rollback needs fresh approval and newer epoch | Evidence and epoch binding; supersession; new epoch for rollback | covered |

# Current behaviour (facts on `be1bfa3`)

**authpolicy proposals apply on approval.**
- `Registry.Propose` stores a pending proposal; an equal pending proposal from the same proposer is returned (`modules/authpolicy/registry.go:145-183`).
- `Registry.Approve` refuses self-approval (`:200-201`), checks the approver against the candidate (`:206-210`), then applies the revision and drops every other pending proposal of the workspace (`:211-214`; `changes.go:116-125`).
- `Registry.Withdraw` is proposer-only (`registry.go:233-237`).
- States are `pending`, `applied` and `withdrawn` (`types.go:42-44`). There is no epoch, validation, separate activation or rejection.

**apps/api policy commands.**
- `policy.propose`, `policy.approve`, `policy.withdraw` require `administer_installation` (`apps/api/server/policy.go:289-296`).
- `policy.approve` answers `applied` (`policy.go:272-280`). Refusals map to `invalid_stage`, `prerequisite_failed` or `dependency_unavailable` (`policy.go:174-185`).
- Proposal rationales and numbers are in `s.records` in memory (`server.go:115`; `policy.go:135-155`).
- `GET /api/v1/identity/policy` requires `administer_installation` (`users.go:42-67`) and returns `policy`, `proposals` and step-up times (`policy.go:64-133`).

**Step-up special case (HYB-293).** `stepUpAction` maps `policy.*` to `policy.change` (`stepup.go:42-47`). For `policy.approve` the window also takes the candidate's rule and maximum age (`stepup.go:64-72`). The candidate is looked up by proposal number without a workspace check (`stepup.go:67`).

**Sessions.** A session holds `created` and `authenticated`, no trust epoch (`session.go:26-33`, `:77-87`). `access` runs the current policy on every request (`commands.go:136-153`, `:214-219`). `command` calls `access` before the domain lock (`commands.go:185`); `execute` re-reads the session only for step-up (`commands.go:196-212`; `stepup.go:75-89`). Sign-in runs under the domain lock (`signin.go:253-283`). Secret pickups record the issuing session (`admin.go:208`).

**Persistence.** Policy, proposals and sessions are in memory. The sample seed recreates the policy on every start through `Propose` by `admin` and `Approve` by `admin2` (`seed.go:181-200`). No durable record exists on this base.

**Permissions.** The action set is closed (`modules/auth/types.go:20-39`). `installationAction` is unexported and true only for `administer_installation` (`types.go:47`). The gate requires a human administrator for that action only (`modules/gate/gate.go:109`). Handler failure codes exclude `self_approval_denied` (`modules/commands/types.go:71-74`), whose catalog entry is 403 with next action `none` (`types.go:55`).

**M03 grant paths filter on one action, not on the installation kind (corrects the first draft; review B1).** On this base:
- the ceiling offered to M03 excludes `g.Terms.Action != gate.AdministerInstallation` (`apps/api/server/access.go:122`);
- the M03 listing skips only `administer_installation` grants (`access.go:161`);
- `access.revoke_grant` refuses only `administer_installation` targets (`access.go:335`);
- `previewKind` excludes the installation kind from previews only (`access.go:189`).
In the HYB-298 worktree the same single-action test is at `access.go:148` and `:195`, `accessrequests.go:86` (`ceilingFor`) and `ledger.go:72-74` (`protectedGrant`, the replay guard). A new installation-kind action would therefore be offered, listed, requestable, revocable and replayable through M03. The first draft's statement "M03 hides installation-kind grants today" was wrong.

**Identity facts used by the evidence.** Suspending a user bumps the user epoch and changes the user state (`modules/identity/installation.go:61-80`); `UserInfo` carries `Kind`, `State` and `Epoch` (`installation.go:154`). HYB-298 already binds membership version, role, state, user state, user epoch, kind in its approval basis (`/private/tmp/hb298/apps/api/server/accessrequests.go:51-52`).

**Bootstrap.** A non-sample installation has one administrator, created by the bootstrap, who receives `administer_installation` (`signin.go:448-453`). Further administrators come from invitations and receive no installation grant.

**Console.** `Live.tsx` posts `policy.propose` (`apps/console/src/modules/identity/pages/authentication/Live.tsx:82`) and `policy.approve` or `policy.withdraw` (`:154`). Types in `apps/console/src/service/admin.ts:25-35`, `:209-219`, `:264-266`.

**Sample seed.** Two administrators and one member (HYB-298 `plan.md:124`, citing `seed.go:24-27`).

# Target design

## Trust rights (permissions)

| Right | Commands |
| --- | --- |
| `trust_propose` | `trust.propose`, `trust.cancel` (own changes only) |
| `trust_validate` | `trust.validate` |
| `trust_approve` | `trust.approve`, `trust.reject` |
| `trust_activate` | `trust.activate` |

- All four are installation-kind actions. `auth.InstallationAction(action)` is exported and true for `administer_installation` and the four trust rights; the gate re-exports it and `gate.go:109` uses it, so all five require a human administrator.
- `administer_installation` grants none of the trust rights, and no trust right grants it (`planning-actions.md:58`). The policy read still requires `administer_installation`.
- Every grant needs the console client grant too, as for the existing installation grants (`seed.go:234-250`).
- **M03 guard (B1).** Every M03 site listed under Current behaviour uses `gate.InstallationAction` instead of the single-action test:
  - ceiling and `ceilingFor`: installation-kind grants are never offered, so `access.request_grant` for any trust right is refused with the generic refusal the module gives for a pair outside the ceiling;
  - listing: installation-kind grants are not listed;
  - revoke target validation (`access.request_revoke`, `access.approve_request` for a revoke request, and any direct revoke path left on main): an installation-kind grant resolves as **not found**, so the executor answers its generic `permission_denied` exactly as for an unknown grant ID, disclosing nothing about its existence (review N4);
  - replay guard: `protectedGrant` is true for any installation-kind grant, so ledger replay never closes one;
  - `integrations.go:334` excludes installation-kind grants with the same predicate (review N5).
- **Seed.** `admin`, `admin2` and `admin3` each hold all four rights (`trust-<right>-<user>`), plus `trust-<right>-console` client grants.
- **Bootstrap.** The bootstrap administrator receives the four rights with `administer_installation` (`signin.go:448-453`). Root default, open for the owner.
- **Validity.** Trust grants have no expiry by default: they are issued with the maximum validity the gate accepts. Root default, open for the owner.
- **Enrolment blocks (review M4).** There is no command that grants a trust right. A non-sample installation therefore has exactly one holder and **can never approve or activate a trust change** until the owner decides an enrolment procedure. This is a truthful block, shown on the screen ("Trust changes need three administrators who hold trust rights. This installation has 1. How further administrators receive trust rights is not decided yet.") and listed in R4 and Q3.

## Actor rule

- **Three distinct people** by internal user ID: proposer ≠ approver, proposer ≠ activator, approver ≠ activator. Root default following the ticket wording; the spec's "approver/activator" grouping (`access-and-trust.md:24`) would allow approver = activator (Q1).
- **Validator:** any holder of `trust_validate`, including the proposer.
- **Rejecter:** a holder of `trust_approve` other than the proposer. **Canceller:** the proposer, holding `trust_propose`.
- **One administrator with rights:** can propose, validate (result `blocked` on recovery) and cancel.
- **Two:** cannot pass validation (recovery needs three people), so nothing is approved or activated.
- **Rechecked every time.** Rights come from the gate on every command, including idempotent replays (`execute.go:324-351`). A changed display name, role label, successful login or new identity does not change the internal user ID and confers nothing (`M02/spec.md:128`).

## Objects and digests

A **trust change** (`modules/trustchange`) has:
- target `trust-<run>-<n>`: `<run>` is 8 random hex characters drawn at each process start and `<n>` counts from 1 (review m9). At start the drawn prefix is compared with the run prefixes of all replayed change IDs and redrawn on a match (review N8), so IDs are never reused across restarts, including IDs of changes that were never activated. Replayed changes are labelled "earlier run" with that run's `started_at`, which the activation record carries (`run_started_at`). The screen shows "Change <n>" and the full ID in the expert detail;
- **Epoch values (reviews N7, R3-4).** An epoch is the integer `(g << 32) + n`: `g` is the ledger generation from the marker; `n` counts within that generation. `n = 0` means no policy yet in generation `g`; the first published policy of a generation (the seed genesis, a carried-forward record or the first activation) has `n = 1`. `Publish` and activation refuse `n ≥ 2^32`. The read carries `g` and `n` as two separate integer fields (`epoch_generation`, `epoch_number`), plus the raw value as a decimal string; the console formats "g.n" from the two fields only and never from a JavaScript Number of the raw value.
- `workspace`, `route` (`local_password`);
- `change_revision`, starting at 1;
- the base `old_epoch`, `old_policy` (authpolicy revision number) and `old_digest`;
- the `candidate` and `new_digest`;
- `state`, actors and a history of transitions with time, actor and rationale.

**No policy yet** (reviews m10, R3-4): an installation without a published policy has epoch `g << 32` (`n = 0`) and policy number 0. `expected_epoch` is then that value (`0` in generation 0), `expected_policy` is `0`, and `old_digest` is the digest of the empty encoding.

Digests (SHA-256, length-prefixed canonical encoding as in `modules/authpolicy/common.go:74-126`):
- `config digest` = `trust_config_v1` over the canonical revision encoding.
- `change_digest` = `trust_change_v1` over workspace, route, change ID, change revision, old epoch, old policy, old digest, new digest and proposer.
- `evidence_digest` = `trust_evidence_v1` over the change digest and the evidence facts (Validation). It has no times and no session counts.

## State machine

```
            propose                  validate (no blocker)        approve                  activate (ledger receipt, publish ok)
  (none) ──────────► draft ──────────────────────► validated ─────────────► independently_approved ──────────────────► activated
                      │ ▲  validate (blocker): stays draft, result blocked      │                       │
                      │ └──────────── propose on target (revise): draft, change_revision + 1 ◄──────────┘
                      │                                                                                 └─ activate, write unknown or publish fault ──► activation_unknown (trust hold)
  draft | validated | independently_approved ── reject ──► rejected
  draft | validated | independently_approved ── cancel ──► cancelled
  draft | validated | independently_approved ── another change activates on the same old epoch ──► superseded
  validated | independently_approved ── validate again after an evidence change ──► validated (approval cleared)
```

| Transition | Who | Preconditions (under the domain lock) | Effect |
| --- | --- | --- | --- |
| propose (new) | `trust_propose` | `expected_epoch`, `expected_policy` current; candidate valid and different from the current config; no trust hold; ledger not isolated; bounds | `draft`, revision 1 |
| propose (revise) | the proposer | state draft, validated or independently_approved; `expected_change_revision` matches; old epoch current | candidate replaced, revision + 1, validation and approval cleared, `draft` |
| validate | `trust_validate` | state draft, validated or independently_approved; `expected_change_revision` and `change_digest` match; old epoch and policy current | findings recorded; `validated` or stays `draft` with result `blocked`; re-validation clears approval with history "approval invalidated: evidence changed" |
| approve | `trust_approve`, ≠ proposer | state `validated` (any other state, including a never-validated or blocked draft: `invalid_stage`, review m11); both digests match stored; recomputed evidence equals stored; old epoch and policy current | `independently_approved` |
| activate | `trust_activate`, ≠ proposer, ≠ approver | state `independently_approved` (otherwise `independent_approval_required`); digests match stored and recomputed; old epoch and policy current; ledger `ok` or `recovered`; record fits | see Activation |
| reject | `trust_approve`, ≠ proposer | state draft, validated or independently_approved; `expected_change_revision` matches | `rejected` |
| cancel | the proposer | same | `cancelled` |
| supersede | inside an activation | other open change, same old epoch | `superseded` with "superseded by <change>, epoch <e>" |

Rules:
- **Terminal:** `activated`, `rejected`, `cancelled`, `superseded`, and `activation_unknown` (resolved only by the next start).
- **Ledger isolated, ledger `stopped` (including a failed start-time `Sync`), or trust hold:** every `trust.*` command is refused with `dependency_unavailable` (reviews M2, M3, R3-3). A `full` ledger refuses `trust.activate` only.
- **Epoch-stale:** `authority_epoch_conflict`; a fresh proposal is needed (`M02/spec.md:130`).
- **Evidence-stale:** approve and activate answer `evidence_stale`; the read shows `evidence_current: false` and "The evidence changed after validation. Validate again; approval is needed again."
- **No silent rebasing.** Nothing moves a change to a new epoch.
- **Duplicate draft:** same proposer, candidate and old epoch → `invalid_stage` (`field: candidate`).
- **Bounds** (constants, resource limits): 25 open changes per proposer, 100 per workspace, 200 kept, 20 history entries per change (as authpolicy today, `registry.go:19-21`; `policy.go:19`).

## Validation

The spec binds identity binding, proof, diagnostics, recovery, external status, session impact and affected subjects (`M02/spec.md:126`). Pre-launch has one route, local password. Each item is `verified`, `not_applicable` with a reason, or `blocked` with a reason; a blocked item holds the change (`M02/spec.md:110`).

Evidence is computed by apps/api before the module call, with **one captured `now`** (review m3) used for every judgement and gate decision in that command.

| Item | Check | Result |
| --- | --- | --- |
| Test login | The validator is the test principal. Their session authentication (password, no factors, `authenticated`) is judged under the candidate at `now` by `authpolicy.Judge(principal, authentication, candidate, now)`. | `verified`, or `blocked: test_login_failed` with the authpolicy reason |
| Recovery path (review m4) | See the definition below. | `verified`, or `blocked: recovery_unverified` with which role is missing |
| Issuer and stable subject | Password only (`policy.go:259`). | `not_applicable: local_password` |
| Provider diagnostics | No provider connection. | `not_applicable: no_provider_connection` |
| External identity status | No external identity. | `not_applicable: no_external_identity` |
| MFA weakening | Old and new MFA scope compared. | `weakens_mfa` or `unchanged`, informational |
| Affected subjects | All members of the workspace. | Count and up to 100 internal user IDs |
| Session impact | Console sessions of the workspace now open. | Count, live; not in the digest |
| Record size | The activation record must fit 2 KiB (HYB-298 `types.go:10`). | `blocked: record_limit` (unreachable through the API) |

**Recovery path, exactly** (review m4). A person *qualifies* when all hold at `now`:
1. human user, user state active and activated (not invited or pending), membership active, role administrator;
2. a usable password credential exists for their login in `modules/passwords` (set, not awaiting a reset link);
3. not blocked by an authpolicy block plan (`authpolicy` `blocked`, `evaluate.go:35-36`);
4. admitted under the candidate by `authpolicy.Reachable` (password allowed; any MFA requirement satisfiable with no factors, which in pre-launch means MFA `none`);
5. the effective gate decision for the right in question is allowed for that person together with the console client (see Evidence).

The recovery item is `verified` only if there exist **three distinct qualifying people** holding `trust_propose`, `trust_approve` and `trust_activate` respectively, **and** at least one qualifying holder of `trust_validate` (who may be one of the three). This is what a later correction or rollback needs.

**Evidence digest facts** (review M1). For every administrator of the workspace, sorted by user ID:
- membership version, membership state and role;
- user state, user epoch and kind (`UserInfo`, `modules/identity/installation.go:154`; suspension bumps the epoch, `:61-80`), following HYB-298's basis (`accessrequests.go:51-52`);
- password credential usable (yes/no), authpolicy block (yes/no), admitted under the candidate (yes/no);
- for each of the four rights, the effective decision at `now` from `auth.ExplainInstallation(person, ConsoleClient, request, now)` (review N3): allowed or the refusal reason. This covers grant validity and expiry, closed grants, the installation-action and human-administrator rules and the client-grant intersection;
- the IDs and versions of the grants that produce an allowed decision.
Plus: current epoch, policy number, member count and the change digest. A suspension, a user-level suspension that leaves membership unchanged, a role change, a grant revocation, a grant expiry or a lost client grant changes the digest.

If the owner decides a separate test-login feature is mandatory even for local password (Q2), the test-login item becomes `blocked: not_connected`.

## Activation

All steps run in the `trust.activate` handler under the domain write lock.

1. **Check** rights, three-person rule, state, digests (stored and recomputed), `expected_epoch` and `expected_policy`, no trust hold, ledger `ok` or `recovered`, `authpolicy.CanPublish(workspace, E1)`.
2. **Intent.** `E1 = max(E0 + 1, (g << 32) + 1)` for the current ledger generation `g` (Restart; review R4-3). In generation `g`, no policy yet is `E0 = g << 32` (`n = 0`), so the first activation after a replacement without a carried record gets `n = 1`. Affected sessions: every console session of the workspace with `trust_epoch < E1`, i.e. all of them, including the activator's.
3. **Commit** the `trust_activation` record. Only a nil `Sync` gives a receipt.
4. **Apply** (receipt only), in order:
   1. `authpolicy.Publish(workspace, candidate, E1, policy number, actors)`.
   2. Mark the change `activated` with the receipt.
   3. Supersede every other open change of the workspace.
   4. Sweep: for each affected session, `identity.EndSession`, `sessions.drop`, and drop the secret pickups issued by that session (review m5). Re-scan and count sessions still below `E1`.
   5. Record the invalidation progress.
5. **Answer** `completed`, `activated`, with `epoch`, `ledger_sequence`, `sessions_ended`, `sessions_remaining`, `your_session_ended: true`.

**Ledger refusal before the write, or `commit_failed`.** Nothing is applied; `dependency_unavailable`. Copy (review m14): "Nothing was activated. The decision ledger stopped accepting records. Restart the server. If the ledger cannot be verified at the next start it is isolated, trust changes stay unavailable, and an operator must restore it or accept a new one."

**`commit_unknown`** (`Sync` failed after a full write; review M2). Fail closed:
- Nothing is published in memory (applying it would let a restart without the record revert to an older epoch).
- The workspace enters a **trust hold** until restart (reviews N2, R3-1). Every console session of the workspace is ended (sweep as above). The hold never refuses a command whose only effect is to reduce access, stop a harmful effect, withdraw a pending proposal or place a hold (`retention-access.md:59`, authoritative deny first; HYB-298 `plan.md:335`, `:507-510`, `:524-527`). Everything else is refused with `dependency_unavailable`.
  - **Hold table** (`trusthold.go`, `holdTable`): one row per registered command ID, enumerated from the HYB-298 worktree registry; re-enumerated at implementation time and any new ID classified by the same rule. Production is default-deny: an ID absent from the table is refused.

| Command | During the hold | Reason |
| --- | --- | --- |
| `identity.suspend_user`, `identity.suspend_membership`, `identity.depart_user`, `identity.remove_membership` | allowed | reduce access |
| `identity.change_role` | allowed only when the new role is `member` | a decrease; to administrator is an increase (HYB-298 `:524-527`) |
| `identity.revoke_invitation` | allowed | reduces pending access (`admin.go:419` in the HYB-298 worktree) |
| `identity.create_user`, `identity.invite`, `identity.reinstate_user`, `identity.reinstate_membership` | refused | add or restore access |
| `identity.issue_set_link`, `identity.issue_reset_link` | refused | issue a credential path |
| `access.request_revoke`, `access.revoke_delegation` | allowed | reduce access; recorded `decision_unconfirmed` (ledger `stopped`) |
| `access.approve_request` | allowed only for a revoke request | add requests are increases (HYB-298 `:524-527`) |
| `access.reject_request`, `access.withdraw_request` | allowed | end a pending request (`accessrequests.go:434-435`) |
| `access.request_grant` | refused | increase |
| `service.suspend`, `service.retire`, `service.revoke_credential` | allowed | reduce service access (`services.go:386-390`) |
| `service.create`, `service.reinstate`, `service.issue_credential` | refused | add or restore access |
| `recovery.lock` | allowed | containment (`recoverycmds.go:183`, `:188`) |
| `recovery.withdraw` | allowed | withdraws a pending proposal |
| `recovery.approve`, `recovery.propose_unlock` | refused | could lift containment |
| `retention.place_hold`, `retention.stop_run` | allowed | place a hold; stop a deletion run (`retentioncmds.go:395-399`) |
| `retention.withdraw` | allowed | withdraws a pending proposal |
| `retention.approve`, `retention.propose_policy`, `retention.propose_release` | refused | could release a hold or start deletion |
| `integrations.plugin_stop`, `integrations.plugin_disable`, `integrations.subscription_pause`, `integrations.subscription_revoke` | allowed | stop an outbound effect (root classification by the rule; Q6) |
| `integrations.plugin_withdraw` | allowed | withdraws a pending proposal |
| `integrations.plugin_propose`, `integrations.plugin_approve`, `integrations.subscription_propose`, `integrations.subscription_approve`, `integrations.subscription_request_resume`, `integrations.subscription_approve_resume`, `integrations.letter_request_redeliver`, `integrations.letter_approve_redeliver`, `integrations.letter_request_dismiss`, `integrations.letter_approve_dismiss` | refused | start, resume or change delivery |
| `installation.withdraw` | allowed | withdraws a pending proposal |
| `installation.approve`, `installation.propose_plan` | refused | change the installation |
| `relationship.discard` | allowed | discards a pending plan |
| `relationship.propose`, `relationship.activate` | refused | change organization structure |
| `contract.suspend`, `contract.resume`, `contract.renew`, `contract.end`, `contract.withdraw`, `contract.evidence`, `contract.settle`, `contract.reopen` | refused | business changes, not access reductions |
| `release_evidence.withdraw` | allowed | withdraws a pending proposal |
| `release_evidence.approve`, `release_evidence.retire_requirement`, `release_evidence.retire_profile` | refused | change release evidence |
| `trust.*` (all six) | refused | the trust workflow is frozen until restart |
| `POST /me/sessions/end`, `POST /auth/logout`, `POST /me/password`, `POST /me/reauthenticate` | allowed | end sessions; password change; password confirmation is needed because the allowed decreases are step-up commands (HYB-293) |
| `POST /auth/login` | allowed, without invitation acceptance | during the hold sign-in skips `acceptPending` (`signin.go:265`), so a user signs in to existing memberships only and opens a read-only session for this table (review R4-1) |
| `POST /auth/redeem` | set link: refused (`dependency_unavailable`); reset link (`PurposeReset`): allowed | a set link accepts invitations and activates an account (`signin.go:399-412`), an increase. A reset link only rotates an existing account's credential and ends its sessions (`signin.go:405-408`), never raises privilege, and matches `/me/password` being allowed (root default after round 5, Q6) |
| `POST /auth/link` | allowed | unauthenticated link peek; reads only |
| `POST /api/v1/commands/{command}` | per the command rows above | — |

The hold test enumerates `postRoutes` (`stepup.go:29-38`) as well as the registry, and fails when a route has no row and reason in this table (R4-1). `TestPostRoutesAreClassified` (`stepup_test.go:137-166`) is extended to match every non-GET method registered in `boundary.go`, not only the literal `"POST "` (R5-2).

  - Where the check runs (R5-1): commands in `execute` under the domain write lock; the `/me` handlers under the lock they already take; **sign-in** under the domain write lock already held in `signIn` (`signin.go:254`), before `acceptPending` (`:265`); **redeem** in an optional fast path before `Reserve`, and a mandatory re-check after taking the write lock (`signin.go:393`) and before `passwords.Commit` (`:399`) and `acceptPending` (`:410`). On a refusal the deferred `Release` (`signin.go:382`) returns the link unconsumed, so it stays usable after the restart. A test enumerates the registry and fails when any registered ID is missing from the table, and asserts the exact allowed and refused lists above with their reasons, so it is not satisfied by default-deny alone.
  - Sign-in remains possible so administrators can read the state and make decreases.
- The change becomes `activation_unknown`. The command answers `accepted` (202) with job ID `trust-activation-<change>`; the read reports the job state `waiting` with reason `activation_unknown` (`M02/spec.md:149`; HYB-298's accepted precedent for `decision_unconfirmed`). No retry button.
- Copy: "The activation could not be confirmed as written. Everyone in this workspace was signed out. Until the server restarts, only changes that reduce access, stop an effect, withdraw a pending proposal or place a hold are possible, plus ending sessions, password changes and using an already issued password reset link. Invitations cannot be accepted and new password reset links cannot be issued, so account setup waits for the restart. At the next start the server applies the activation only if its record is readable."
- The ledger is `stopped`, so any later ledger commit in this process, including a revocation, is refused or unconfirmed per HYB-298; a later `trust.activate` answers `dependency_unavailable`.
- Root default, open for the owner (Q6).

**`Publish` failure after a receipt** (review m6). Unreachable by construction, because `CanPublish` runs in step 1 under the same lock. If it still fails: the same trust hold as `commit_unknown`, the change becomes `activation_unknown` with reason `publish_failed`, one error log line, and the next start replays the durable record. Tested through an injected fault.

**Ledger record** (`decision: trust_activation`): `workspace`, `change`, `change_revision`, `change_digest`, `evidence_digest`, `old_epoch`, `new_epoch`, `old_policy`, `new_policy`, `old_digest`, `new_digest`, `config` (methods, MFA, maximum age in seconds, step-up rules), `proposed_by`, `validated_by`, `approved_by`, `activated_by`, `approved_at`, `invalidate_below_epoch`, `affected_sessions`. No rationale, logins or tokens. Internal user IDs are kept as long as the ledger (HYB-298 O-6).

**Invalidation progress** (`M02/spec.md:118`, `:130`):

| Item | Result |
| --- | --- |
| Console sessions below the new epoch | `done` with `ended` and `remaining` from the re-scan, or `in_progress` if `remaining > 0` |
| Password confirmation evidence | `done`: held in the session record |
| Secret pickups issued by ended sessions | `done`: dropped by the sweep |
| Requests admitted before activation and still running (reader proxy reads) | `unknown`: a read that passed `access` before the lock may still finish (`readers.go:111-126`) |
| Service credentials (M04) | `not_applicable`: no apps/api route calls `identity.Boundary.AccessByCredential` (`modules/identity/boundary.go:142`; the only callers are module tests), so service credentials authenticate nothing in pre-launch (review N9) |
| Refresh tokens, queued privileged commands, cached permissions | `not_applicable`: none exist; the executor is synchronous and rechecks per request |
| Copies delivered to browsers | `outside_scope` (`M02/spec.md:112`) |

**Enforcement independent of the sweep.**
- `sessions.put` records `trust_epoch` (review m13) under the domain lock.
- `access` refuses a session whose `trust_epoch` is below the workspace's current epoch, ends it and answers `unauthenticated` (`commands.go:136-153`, `readers.go:116`, `boundary.go:117`). The trust hold is enforced per command by the hold table in `execute` and in the `/me` handlers, not in `access`, so reads stay available.
- `execute` re-checks the caller's session, `trust_epoch` and the hold after taking the lock, for every command (closes the gap at `commands.go:185`).
- `/me/reauthenticate` runs `access` first, so it cannot revive an old-epoch session.

## Concurrency

- Two changes on one epoch: the first activation supersedes the second inside the lock; the second gets `invalid_stage` or `authority_epoch_conflict`; `Publish` independently refuses an epoch not strictly newer.
- Activation and sign-in are serialized (`signin.go:253-255`).
- Activation and a queued command: refused by the re-check in `execute`.
- Same key twice: replay after the gate recheck; a new key: `invalid_stage`.

## Restart and persistence

Durable: activation records in `<data>/decisions/ledger.jsonl` and the ledger generation in the marker and genesis (HYB-298 files). In memory: changes that were not activated, rationales, histories, sessions and the trust hold.

`Start` (after HYB-298's ledger open and revocation replay):
1. **Ledger isolated, for any reason** (missing with the marker present, integrity failure, ID mismatch, epoch order; reviews M3, N1, R4-4): the latest `trust_activation` record **in the verified prefix** is published at its epoch and policy number, origin `ledger_replay`. For `integrity_failed` the verified prefix ends before the first failing line; for `epoch_order` it ends at the last record before the first epoch violation. Only when the verified prefix has no trust record does the seed genesis apply (sample seed) or no policy (empty installation). Every `trust.*` command is refused with `dependency_unavailable`, and the trust line reads: "The decision ledger cannot be verified. The sign-in rules shown may be older than the last activated change. Trust changes are unavailable until an operator restores the ledger or accepts a new one." For `integrity_failed` and `epoch_order` it adds "(problem at line X)". Other commands follow HYB-298's isolation rules (increases refused, decreases allowed).
2. **Ledger usable:** read the `trust_activation` records of the boot workspace. Epochs must increase strictly; the first violation stops replay at the last good record and the workspace enters isolation as in step 1 (reason `epoch_order`).
3. **No records:** the seed publishes its genesis policy at epoch `(g << 32) + 1` with origin `seed`; the empty installation publishes none (`signin.go:268-272`).
4. **Records:** the latest record's configuration is published at its epoch and **its policy number** (review m9), origin `ledger_replay`. The seed genesis is not published.
5. **Ledger generation (reviews N7, R3-2, R3-4).** The marker carries `g` (0 for a marker without it). When an operator accepts a new ledger (`-accept-new-decision-ledger`), the new marker and genesis get `g + 1`.
   - **Carry forward (R3-2, R4-2).** A record is *carryable* only if the replaced file's genesis ledger ID equals the marker's ledger ID and the record's workspace is the boot workspace. Then the latest verified carryable `trust_activation` configuration is written into the new ledger as its first trust record, at epoch `((g + 1) << 32) + 1`, with `origin: carried` and `carried_from` = the replaced ledger ID and that record's sequence; actors are copied from the original record. For `integrity_failed` and `epoch_order` the new genesis and the M02 line also say "The carried configuration may be older than the last activation (problem at line X)." (R4-4). Start then publishes it as usual.
   - **ID mismatch or unusable marker (R4-2).** The replaced file is treated as having no carryable record. Start is refused unless the operator passes either `-accept-trust-reset` or `-carry-trust-from <ledger-id>`, which names the replaced file's genesis ledger ID explicitly; the named ID must match that file's genesis, and the new genesis records the operator's choice. The workspace check still applies.
   - **No carryable trust record** in the replaced file (or the file is missing): start is refused unless the operator also passes `-accept-trust-reset`, which acknowledges that the seed genesis (or no policy) becomes active. The genesis records the acknowledgement, and M02 shows "Sign-in rules were reset to the installation defaults when the decision ledger was replaced on …".
   - **Generation source (R4-5).** If the marker is missing or unreadable but the current ledger verifies, `g` is taken from its genesis, as in HYB-298's ledger-without-marker path (HYB-298 `plan.md:482`); no flag is needed. Only if the marker is unusable **and** the current ledger is missing or unverifiable is start refused unless the operator passes `-trust-generation N`. `N` is validated against every retained `ledger.jsonl.replaced-*` file and any readable genesis: start is refused if `N ≤ max(every g found, every highest epoch >> 32)`. The genesis records that `N` was operator-stated.
   - Deleting both the marker and the ledger is HYB-298's first-start path: it restarts at `g = 0` with no warning (R1).
6. A record is replayed only if it is verified (N6). If the start-time `Sync` fails (HYB-298 amendment 7, R3-3), the latest verified trust configuration is still replayed, the ledger is `stopped`, every `trust.*` command is refused, and M02 labels the trust line "replayed from records not confirmed durable". A `commit_unknown` record that is verified at the next start is the latest record and applies; one that is not is absent. The read says "Changes from before the server started at … are not kept" and shows the replayed epoch, origin and the earlier run's `started_at`.
7. Invalidation for a replayed activation is `complete`: the restart ended every session.

## Step-up interaction (HYB-293)

- All six `trust.*` commands are mutating, so the registry rule makes each step-up (`stepup.go:40`).
- `stepUpAction` maps `trust.*` to `policy.change` (`stepup.go:42-47`).
- The candidate-window special case moves from `policy.approve` to `trust.validate` (`stepup.go:18`, `:64-72`): `min(D, current rule, candidate rule, candidate MaxAge)`. The candidate lookup is scoped to the caller's workspace (review m3): a change in another workspace, or an unknown or malformed target, contributes nothing and the executor then answers its generic not-found.
- MFA: a candidate MFA step-up rule would fail closed without a prompt (HYB-293). It is unreachable through the API, which proposes MFA `none` and no MFA rule (`policy.go:259-262`); stated, not tested through HTTP.
- Step-up stays before permission (HYB-293 "Ordering").
- After activation the activator's session is ended; a retry with the same key from a new session replays after the gate recheck.

# API contract

## Commands

`POST /api/v1/commands/{command}` with `{request_id, target_id, idempotency_key, expected_revision, rationale, body}` (`admin.ts:392-399`). `expected_revision` stays `s.Revision()`. Rationale is required on all six.

| Command | Target | Body fields | Result |
| --- | --- | --- | --- |
| `trust.propose` | `trust` (new) | `max_age_minutes`, `step_up_minutes` (`policy.go:251-262`), `expected_epoch`, `expected_policy` (`0` without a policy) | `draft`; `change`, `change_revision`, `change_digest` |
| `trust.propose` | change ID (revise) | the same plus `expected_change_revision` | `draft` |
| `trust.validate` | change ID | `expected_change_revision`, `change_digest`, `expected_epoch` | `validated` or `blocked`; `evidence_digest`, `blocked_items` |
| `trust.approve` | change ID | `change_digest`, `evidence_digest`, `expected_epoch` | `independently_approved` |
| `trust.activate` | change ID | `change_digest`, `evidence_digest`, `expected_epoch`, `expected_policy` | `completed` `activated` (`epoch`, `ledger_sequence`, `sessions_ended`, `sessions_remaining`, `your_session_ended`), or `accepted` with job `trust-activation-<change>` (`activation_unknown`) |
| `trust.reject` | change ID | `expected_change_revision` | `rejected` |
| `trust.cancel` | change ID | `expected_change_revision` | `cancelled` |

## Refusals

| Code | HTTP | Next action | When |
| --- | --- | --- | --- |
| `self_approval_denied` (existing catalog entry, newly allowed as a handler code; status and next action unchanged, review m2) | 403 | `none` (`types.go:55`) | proposer approves, rejects or activates; approver activates |
| `authority_epoch_conflict` (new) | 409 | `inspect_current_state` | expected epoch or policy not current; change's old epoch not current |
| `approval_stale` (new) | 409 | `inspect_current_state` | `change_digest` or `expected_change_revision` mismatch |
| `evidence_stale` (new) | 409 | `inspect_current_state` | `evidence_digest` mismatch or recomputed evidence differs |
| `independent_approval_required` (new) | 422 | `correct_request` | activation of a change not `independently_approved` |

Existing codes:
- `invalid_stage` (422): approval of anything not `validated`, including a never-validated or blocked draft (m11); terminal, superseded or `activation_unknown` changes; duplicate draft.
- `permission_denied` (403): right missing, not a human administrator, cancel by a non-proposer, or an unknown or other-workspace target (executor generic not-found).
- `invalid_request` (422) with the field; `dependency_unavailable` (503): ledger not usable, isolated, trust hold, bounds; `reauthentication_required` (403); `idempotency_conflict` and `revision_conflict` (409).

`policy.propose`, `policy.approve`, `policy.withdraw` are removed (`unsupported_command`, 404). The OpenAPI document lists the new commands and codes and marks each `trust.*` operation `x-step-up: true`. Adding the four codes is a public contract change covered by this review.

## Read: `GET /api/v1/identity/policy`

Permission unchanged (`administer_installation`).

```
revision, now, signed_in_at, authenticated_at, act_until, step_up_seconds, you,   (unchanged)
started_at,
trust: { epoch: "<decimal string>", epoch_generation, epoch_number, policy_number, digest, origin: seed|this_process|ledger_replay|carried, carried_from, durable_unconfirmed, activated_at,
         proposer, validator, approver, activator, change, hold: null | { reason, since } },
policy: { …revisionRow, number, epoch, proposer, approver, activator, at },
ledger: { state, isolation_reason, records, watermark, generation },
rights: { propose, validate, approve, activate },
holders: { qualifying_people, needed: 3, enrolment: "not_decided" },
changes: [ {
  id, number, state, change_revision, change_digest, old_epoch, old_policy, old_digest, new_digest,
  candidate, proposer, validator, approver, activator, superseded_by,
  validation: { result, at, evidence_digest, evidence_current, items: [ { item, result, reason, detail } ] },
  impact: { members, sessions, affected_ids (≤100), weakens_mfa, signs_out_everyone: true } | null,   (null when the change is not open or its facts could not be read; HYB-314)
  activation: { epoch, ledger_sequence, watermark, at } | null,
  job: null | { id, state: "waiting", reason: "activation_unknown" | "publish_failed" },
  invalidation: { state, items: [ { item, result, ended, remaining, reason } ] } | null,
  can: { revise, validate, approve, activate, reject, cancel },
  why_not: { approve, activate },
  act_until,
  history: [ { state, by, at, rationale } ]
} ]
```

`proposals` is removed. Rationales appear only on this admin-only read and never in logs or the ledger.

# Console design (M02 live screen)

**Visual lead.** The `ui-designer` role specifies the layout from current renders and the references below; root implements; the designer compares final renders. Full viewport width, neutral separators only, no colored edge strips or outlines, state as text and icon, visible focus (`service-screen-style.md`).

**External references** (to be read and recorded before layout; nothing below is an observation yet):
- GitHub Docs, protected branches and required reviews (dismissal of stale approvals).
- Microsoft Learn, Privileged Identity Management approve or deny requests (justification and history).
- Terraform plan and apply, Google Cloud Organization Policy dry run (validation separate from apply).

**Structure** (policies view, `Live.tsx:235-406` refactored):
- **Current sign-in rules**: facts plus trust epoch, origin, activator, the ledger line and, when present, the trust hold or isolation line.
- **Holders line**: "N administrators qualify for trust changes; a change needs 3." With fewer, the enrolment block copy (M4).
- **Trust changes list** with state (text and icon), proposer, epochs and the next required step in words.
- **Detail panel** (right on desktop, separate step on mobile, `M02/spec.md:159-161`): current and candidate side by side (stacked on mobile), digests in an expert detail, validation items, impact, actors, history with rationales, and the caller's action.
- **After activation**: result, epoch, ledger sequence, invalidation items, then "Sign in again". After `activation_unknown`: the waiting copy, no retry.
- **Revoked**: list, detail and selection cleared.

**Confirmations** (review M5; `M02/spec.md:96-101`, `:126`, `:143`). Each is a modal dialog with focus trap and return, Escape closes without executing, and a rationale field. Each shows target, epoch, revision and the cancellable boundary.

| ID | Command | Shows before submission |
| --- | --- | --- |
| C1 | propose / revise | Proposed scope (workspace, route local password, affected members count), current and candidate settings side by side, old digest and the new digest the server will compute (computed client-side from the same canonical encoding only for display, labelled "expected"; the server value is authoritative), "Based on epoch E, policy revision R", and the lockout and sign-out consequences if later activated |
| C2 | validate | Validation scope (which items are checked, which are not applicable and why), that the validator's own session is the test login, the change digest being validated |
| C3 | approve | Exact change digest and evidence digest, the approver's authority (their right and that they are not the proposer), session impact (members and open sessions, "everyone is signed out at activation"), lockout impact from the recovery item, and that activation still rechecks |
| C4 | activate | Approved digest, proposer, validator and approver, affected principals (count and list), session impact including "you", lockout risk, new epoch, and that undoing it needs a new change with a newer epoch |
| C5 | reject | Change, current state, that active trust is unchanged, rationale |
| C6 | cancel | Change, current state, that active trust is unchanged, rationale |

**User-facing states.** EN is the spec copy where given (`M02/spec.md:107-120`); KO and AR are AI drafts for the terminology review.

| State | EN | KO (draft) | AR (draft) |
| --- | --- | --- | --- |
| draft | Trust change is a draft; active trust is unchanged. | 신뢰 변경 초안입니다. 현재 적용 중인 신뢰 설정은 바뀌지 않았습니다. | تغيير الثقة مسودة؛ إعدادات الثقة النشطة لم تتغير. |
| validated | Exact proposal evidence is validated; independent approval is required. | 제안 내용의 근거가 검증되었습니다. 다른 관리자의 독립 승인이 필요합니다. | تم التحقق من أدلة الاقتراح بدقة؛ يلزم اعتماد مستقل. |
| independently_approved | Independent approval is recorded; activation still rechecks current authority and old epoch. | 독립 승인이 기록되었습니다. 활성화할 때 현재 권한과 이전 epoch를 다시 확인합니다. | تم تسجيل الاعتماد المستقل؛ يعيد التفعيل التحقق من الصلاحية الحالية والحقبة السابقة. |
| activated | The new epoch and invalidation intent are committed. Inspect remaining invalidation progress. | 새 epoch와 무효화 의도가 확정되었습니다. 남은 무효화 진행 상황을 확인하세요. | تم تثبيت الحقبة الجديدة ونية الإبطال. راجع تقدم الإبطال المتبقي. |
| rejected | The proposal is rejected; active trust is unchanged. | 제안이 반려되었습니다. 현재 신뢰 설정은 바뀌지 않았습니다. | تم رفض الاقتراح؛ إعدادات الثقة النشطة لم تتغير. |
| cancelled | The proposal is cancelled; active trust is unchanged. | 제안이 취소되었습니다. 현재 신뢰 설정은 바뀌지 않았습니다. | تم إلغاء الاقتراح؛ إعدادات الثقة النشطة لم تتغير. |
| superseded (ED-02) | Another change was activated on the same epoch. This proposal can no longer be activated; propose again if it is still needed. | 같은 epoch에서 다른 변경이 활성화되었습니다. 이 제안은 더 이상 활성화할 수 없으니 필요하면 다시 제안하세요. | تم تفعيل تغيير آخر على الحقبة نفسها. لا يمكن تفعيل هذا الاقتراح؛ اقترح مجددًا إذا لزم. |
| activation_unknown | The activation could not be confirmed as written. Everyone in this workspace was signed out. Until the server restarts, only changes that reduce access, stop an effect, withdraw a pending proposal or place a hold are possible, plus ending sessions, password changes and using an already issued password reset link. Invitations cannot be accepted and new password reset links cannot be issued. At the next start the server applies the activation only if its record is readable. | 활성화 기록이 저장되었는지 확인할 수 없습니다. 이 작업 공간의 모든 사용자가 로그아웃되었습니다. 서버를 다시 시작할 때까지는 권한 축소, 효과 중지, 대기 중인 제안 철회, 보존 조치만 할 수 있으며 세션 종료, 비밀번호 변경, 이미 발급된 비밀번호 재설정 링크 사용은 가능합니다. 초대를 수락하거나 새 비밀번호 재설정 링크를 발급할 수 없습니다. 다음 시작 시 기록을 읽을 수 있는 경우에만 활성화가 적용됩니다. | تعذر تأكيد كتابة التفعيل. تم تسجيل خروج الجميع في مساحة العمل هذه. حتى إعادة تشغيل الخادم، لا يمكن إلا تقليل الوصول أو إيقاف أثر أو سحب اقتراح معلّق أو وضع حجز، إضافةً إلى إنهاء الجلسات وتغيير كلمة المرور واستخدام رابط إعادة تعيين صادر مسبقًا. لا يمكن قبول الدعوات ولا إصدار روابط جديدة لإعادة تعيين كلمة المرور. عند البدء التالي يطبّق الخادم التفعيل فقط إذا كان سجله قابلًا للقراءة. |
| blocked | Test login or recovery path is unverified. Hold policy activation. | 테스트 로그인 또는 복구 경로가 확인되지 않았습니다. 정책 활성화를 보류합니다. | لم يتم التحقق من تسجيل الدخول التجريبي أو مسار الاسترداد. أوقف تفعيل السياسة. |
| enrolment block (M4) | Trust changes need three administrators who hold trust rights. This installation has N. How further administrators receive trust rights is not decided yet. | 신뢰 변경에는 신뢰 권한을 가진 관리자 3명이 필요합니다. 이 설치에는 N명이 있습니다. 다른 관리자에게 신뢰 권한을 부여하는 방법은 아직 정해지지 않았습니다. | تتطلب تغييرات الثقة ثلاثة مسؤولين يملكون صلاحيات الثقة. لدى هذا التثبيت N. لم يُحدَّد بعد كيف يحصل مسؤولون آخرون على صلاحيات الثقة. |
| ledger isolated (M3) | The decision ledger cannot be verified. The sign-in rules shown may be older than the last activated change. Trust changes are unavailable until an operator restores the ledger or accepts a new one. | 결정 원장을 확인할 수 없습니다. 표시된 로그인 규칙이 마지막으로 활성화된 변경보다 이전 것일 수 있습니다. 운영자가 원장을 복구하거나 새 원장을 승인할 때까지 신뢰 변경을 사용할 수 없습니다. | لا يمكن التحقق من سجل القرارات. قد تكون قواعد تسجيل الدخول المعروضة أقدم من آخر تغيير مُفعَّل. تغييرات الثقة غير متاحة حتى يستعيد المشغّل السجل أو يقبل سجلًا جديدًا. |
| loading, empty, partial, revoked | Spec copy (`M02/spec.md:107-109`, `:112`) | 사양 문구의 초안 번역 | ترجمة مسودة لنص المواصفة |

"Epoch" stays a specialist term in KO and AR pending TW review; the main surface may say "trust version" (KO 신뢰 버전, AR إصدار الثقة) if the review agrees.

**Refusal copy.** `policyFailure` (`live-model.ts:72-100`) and `AdminFailure` gain `self`, `epoch`, `approval_stale`, `evidence_stale`, `needs_approval` and `trust_unavailable` (for `dependency_unavailable` with a hold or isolation), each with a reload and a plain next step.

**Glossary** (new rows only): trust change, trust version (epoch), validate, independent approval, activate, supersede, invalidation progress, activation unknown, trust hold, trust rights.

**Preview screen** `authentication/Screen.tsx` is unchanged.

# Tests and verification

## `modules/trustchange`
- Every transition from every state, including refusals; terminal states accept nothing.
- Three-person rule for every pair; rejecter ≠ proposer; canceller = proposer; approving a never-validated or blocked draft gives `invalid_stage`.
- Revision resets validation and approval.
- Digest binding: each bound field changes the digest; equal inputs give equal digests.
- Activation calls the commit callback once, after all checks; error leaves the change unchanged; unknown gives `activation_unknown`; receipt supersedes exactly the other open changes on the same epoch.
- Change IDs with different run prefixes never collide; bounds; duplicate draft.
- **Property tests** over random command sequences by three to five administrators:
  - at most one `activated` change per old epoch;
  - activated epochs strictly increase;
  - no activated change has a repeated person among proposer, approver and activator;
  - no change reaches `independently_approved` or `activated` without a non-blocked validation whose evidence digest equals the one current at that step;
  - terminal states never change.
- Source test: stdlib only.

## `modules/authpolicy`
- `Propose` and `Approve` refuse `ChangeRevision`; plans and lifts unchanged.
- `Publish` refuses an epoch not strictly newer and an invalid revision; it sets the given policy number on replay; `RevisionView` carries `Epoch` and `Activator`.
- `Judge(…, now)` and `Reachable` agree with `Evaluate` for every reason (table and property test).

## `modules/auth`, `modules/gate`, `modules/commands`
- The four rights are valid installation-kind actions; `InstallationAction` is true for exactly the five installation actions; refused for a member, a service subject and an AI principal; `administer_installation` and the trust rights do not imply each other.
- The four new codes have the listed status and next action; `self_approval_denied` keeps 403 and `none`; all five are accepted from handlers and appear in OpenAPI.

## `modules/decisionledger`
- A `trust_activation` record round-trips, is shaped and chained, fits with one rule and is refused oversized.
- A genesis and marker with generation `g` round-trip; older ones without it read as `g = 0`; an operator-stated generation is recorded as such.
- Mixed revocation and trust records replay in order; HYB-298's tests stay green.

## M03 guard (review B1; after the HYB-298 rebase)
- `access.request_grant` for each of the four trust rights is refused (the pair is outside the ceiling), with the same body as for any other pair outside the ceiling.
- `access.request_revoke` (and any direct revoke path left on main) on a trust grant gets the generic refusal given for an unknown grant.
- The M03 listing and ceiling exclude every installation-kind grant.
- Ledger replay of a record naming a trust grant does not close it (`protectedGrant`), shown as "Not re-applied: protected grant".
- A source test: no apps/api file compares `Terms.Action` with `gate.AdministerInstallation` for exclusion purposes; the predicate is used instead.

## `apps/api/server` (fixed clock)
- **Happy path** with `admin`, `admin2`, `admin3`: epoch 1 → 2; policy published; every session and its pickups ended; `your_session_ended`; a new sign-in has `trust_epoch` 2.
- **Spec acceptance (`M02/spec.md:132`) and TC-01..TC-03:**
  - self-approval, activation by the proposer, activation by the approver: `self_approval_denied`;
  - an `administer_installation` holder without trust rights: `permission_denied` on every `trust.*` command;
  - old epoch: `authority_epoch_conflict`; old-epoch session: `unauthenticated`;
  - simultaneous activation from two goroutines: exactly one `activated`;
  - candidate mutation: revise after approval → `draft`; old digest → `approval_stale`;
  - revoked approver grant, approver grant expiry, approver user-level suspension (membership unchanged), approver client grant removed: each gives `evidence_stale` (review M1);
  - revoked or suspended activator: `permission_denied` or `unauthenticated`;
  - timeout: injected `Sync` failure → `accepted`, job `waiting` `activation_unknown`, policy and epoch unchanged, every session ended, mutating commands and the `/me` posts refused with `dependency_unavailable`, reads and sign-in allowed; restart on the same directory applies the activation if written and not otherwise;
  - `commit_unknown` on a revocation (HYB-298 path), then `trust.activate`: `dependency_unavailable` (test gap);
  - injected `Publish` failure after a receipt: the same hold, reason `publish_failed`; restart applies the record (m6);
  - rollback to the earlier configuration gets a newer epoch;
  - external unknown: not connected (recorded).
- **Validation:** recovery blocked with two qualifying people; each of the five qualification conditions failing in turn (not activated, no usable password, authpolicy block, not admitted, gate refusal); test login blocked through a crafted session time; `not_applicable` items present; evidence-stale after role, membership, user state, grant and client-grant changes of any administrator.
- **No policy:** `expected_epoch` and `expected_policy` `0` accepted on an empty installation; non-zero refused.
- **Enrolment block:** an empty installation after bootstrap has one qualifying person; validation is `blocked: recovery_unverified`; the read reports `enrolment: not_decided`.
- **Idempotency:** replays after the gate recheck; a replay by a user who lost the right is refused; same key with another body → `idempotency_conflict`; a refusal reserves nothing.
- **Epoch enforcement:** `access` refuses old-epoch sessions on a read, a command, the reader proxy and `/me/reauthenticate`; a command queued during activation is refused; sign-in during activation gets the new epoch.
- **Ledger states:** `full` refuses activation only; `stopped` (including after a failed start-time `Sync`) and `isolated` (missing, integrity, ID mismatch, epoch order) refuse all `trust.*` commands and show the line. Isolation with a trust record before the failing line and another after it: the record **before** the failing line is published at its epoch and policy number; the one after is not; trust commands are refused (review N1, inverted from round 1). Isolation with no trust record in the prefix: seed genesis or no policy.
- **Replacement (N7):** `-accept-new-decision-ledger` writes generation `g + 1`; the next epoch is at least `(g + 1) << 32` and above every replaced epoch; with the marker unreadable, start is refused without `-trust-generation`; display renders "g.n".
- **Restart:** epoch, policy number, configuration, actors and origin `ledger_replay` return; a forced run-prefix collision with a replayed change ID is redrawn (N8); replayed changes show "earlier run" with that run's `started_at`; the seed genesis is not published when records exist; a record unreadable after the start-time `Sync` is not replayed (N6).
- **Carry forward (R3-2, R4-2):** accept a new ledger over a file with a verified carryable trust record: the new ledger's first trust record is at `((g + 1) << 32) + 1` with `carried_from`; the configuration is unchanged after restart. Not carried, start refused without a flag: genesis ledger ID ≠ marker ID; marker unusable; record in another workspace. With `-carry-trust-from` naming the right ID: carried and recorded; naming a wrong ID: refused. Over a file without a carryable record: refused without `-accept-trust-reset`; with it, the seed genesis or no policy applies and M02 shows the reset line. `integrity_failed` and `epoch_order` cases: the carried record is the last one before the problem line, and the genesis and M02 line carry the "may be older (problem at line X)" text.
- **Epoch after replacement (R4-3):** with no carried record and no seed, the first activation in generation `g` is `(g << 32) + 1`; with a carried record it is `(g << 32) + 2`. **Generation source (R4-5):** marker unreadable and ledger verifying: `g` from the genesis, no flag needed; both unusable: `-trust-generation` required and validated.
- **Operator-stated generation (R3-4):** `-trust-generation N` refused for `N` at or below any genesis `g` or highest-epoch generation in the current or any retained replaced file; accepted above; `n ≥ 2^32` refused by `Publish` and activation; the read's `epoch_generation` and `epoch_number` match the raw string for a generation above 2^21 (beyond JavaScript's exact integer range for the raw value).
- **Start-time `Sync` failure (R3-3):** injected failure: the latest verified trust configuration is published, the label reads "replayed from records not confirmed durable", and every `trust.*` command is refused.
- **Trust hold table (N2, R3-1, R4-1):** every allowed row succeeds during the hold (including `service.suspend`, `service.retire`, `service.revoke_credential`, `identity.revoke_invitation`, `access.reject_request`, `access.withdraw_request`, `recovery.lock`, `retention.place_hold`, `retention.stop_run`, a role decrease and a revoke approval) and every refused row answers `dependency_unavailable` (including a role increase, an add approval and `identity.issue_reset_link`); sign-in during the hold opens a session without accepting a pending invitation; `/auth/redeem` with a set link answers `dependency_unavailable` and changes nothing, and with a reset link succeeds; a race test starts the hold between `Reserve` and `Commit` of a set-link redemption (through the existing `s.pause` hook) and asserts the refusal and that the link is still unconsumed and redeemable after a restart; the test fails when a registry ID or a `postRoutes` entry (`stepup.go:29-38`) is missing from the table, and compares both sides with the reasoned lists; an unlisted synthetic command is refused (default-deny).
- **`ExplainInstallation` (N3, R3-5):** agrees with `Decide` for every installation action, human and service subjects, with and without the client grant, expired and not-yet-valid grants, at a given `now`; a delegation naming an installation action is never consulted (none can exist, `registry.go:375`); changes no state; AST test: exactly two callers, the gate re-export wrapper and `trustevidence.go` through it; the M03 preview and M03 paths cannot reach it; the administrator role in the evidence comes from identity facts.
- **M03 revoke of an installation-kind grant (N4):** `access.request_revoke` on a trust grant and on `administer_installation` gives a body byte-identical to an unknown grant ID (`permission_denied`). **Integrations (N5):** the `integrations.go:334` listing excludes trust grants.
- **Server-level properties** (test gap; random interleavings of sign-in, activation, commands, reads, restarts with injected ledger faults):
  - no request is admitted for a session whose `trust_epoch` is below the current epoch;
  - the published configuration changes only through a receipted activation, the seed genesis or replay;
  - after any restart the published epoch is never below the last durably written trust epoch, unless the ledger is isolated, in which case every `trust.*` command is refused.
- **Step-up:** every `trust.*` command is step-up; `trust.validate` uses the candidate window from the caller's workspace only; `policy.*` are `unsupported_command`.
- **Leak:** generic refusals for unknown, other-workspace and malformed targets; ledger, logs and journal contain no rationale, logins or tokens.
- **Source and AST tests:** `authpolicy.Publish` is called only from the activation handler, the seed genesis and replay; no non-test apps/api file calls `authpolicy` `Propose`, `Approve` or `Withdraw` (review m7); `sessions.put` is the only writer of `trust_epoch`; `trustchange.Activate` is called only from the activation handler.
- **Race:** `-race -count=40` over activation, sign-in, commands, password confirmation, logout and reads.
- **Mutation testing** of changed lines in `modules/trustchange`, `modules/authpolicy`, the M03 guard sites, `apps/api/server/trust*.go`, `stepup.go`, `commands.go`, `session.go`, with the session `GOCACHE` cleaned per batch; survivors traced to every reader before any deletion.
- 100% coverage of changed and new Go code; `gofmt`, `go vet`; linux/amd64, linux/386, windows/amd64, darwin/arm64 builds.

## Console unit tests
- `trust-model.test.ts`: next-step text per state and `why_not`, `can` flags, digests, invalidation items, stale evidence, holders and enrolment lines, hold and isolation lines.
- Confirmation content for C1–C6.
- Failure maps for the new failures in EN, KO and AR.
- Render tests for `activated`, `activation_unknown`, `superseded`, `blocked`, enrolment block, isolation and `revoked`.
- `admin.test.ts`: command IDs and read shape.

## Browser QA (production build against a live `hb-api`, runner `qa297`)
- **A.** Full path in EN: `admin` proposes (C1), `admin2` validates (C2) and approves (C3), `admin3` activates (C4); everyone is signed out; signing in shows epoch 2.
- **B.** Separation: the proposer sees no approve or activate action; the approver sees no activate action; a stubbed direct post shows the `self_approval_denied` copy.
- **C.** Two approved changes; activating one supersedes the other.
- **D.** Blocked and stale: suspend `admin3` in M01, validate → `blocked`; reinstate, validate, approve, change `admin3`'s role, activate → `evidence_stale` copy.
- **E.** Reject (C5) and cancel (C6) with rationale; history.
- **F.** Restart after an activation: "Replayed at start" and the "not kept" note.
- **G.** Ledger isolation: start on an altered ledger with the marker; isolation line; trust actions unavailable.
- **H.** Empty installation (no seed): bootstrap administrator sees the enrolment block.
- **I.** M03: no trust right is offered in the grant request form and no trust grant is listed.
- **J.** Password confirmation on `trust.validate` past the window; byte-identical resend.
- **K.** EN, KO and AR at 1440 and 390; RTL; no horizontal scroll at 390; keyboard only, confirmation focus trap and return, Escape; live region; neutral computed borders; axe on the detail panel and confirmations.
- `activation_unknown` and the trust hold are covered by API and render tests only.

## Regression
M01 `qa264`, M02 `qa265r` (replaced by `qa297` where the flow changed), M03 (`qa267` or HYB-298's runner), M04 `qa269`, M05 `qa271`, O01 `qa273`, O02 `qa275`, O03 `qa281`, P01 `qa283full`, and the P02, P03, P04 and P05 runners. IAM-01: the bootstrap and activation tests rerun, as bootstrap now grants trust rights. Changes to runner expectations caused by the third sample administrator are listed with the reason.

# Threats and abuse cases

| Threat | Mitigation | Test |
| --- | --- | --- |
| One person completes a change | Three distinct user IDs; gate rights | Separation tests, property test |
| Granting trust rights to oneself or others through M03 (B1) | Installation-kind predicate on ceiling, `ceilingFor`, listing, revoke validation and replay | M03 guard tests, QA I |
| Generic administration activates (TC-01) | `administer_installation` confers no trust right | Permission test |
| Same human with two accounts | Accounts are created by administrators; not detectable | Residual R5 |
| Approve one candidate, activate another | `change_digest`; revision resets approval | Candidate-mutation test |
| Facts change between approval and activation (suspension, user-level suspension, grant expiry, client grant) | Evidence binds user state, user epoch, kind and effective gate decisions | Evidence-stale tests |
| Replay of an old activation | Stored result only, after the gate recheck; new key meets the epoch check | Replay tests |
| Two activations on one epoch | Lock, module check, `Publish` strictly newer | Concurrency and property tests |
| Old session keeps acting | `trust_epoch` in `access` and under the lock; sweep | Epoch tests, server property |
| Lockout candidate | Test-login and recovery items | Validation tests |
| Seed or generic path bypasses the workflow | `Propose`/`Approve` refuse revisions; AST tests on `Publish` and on `Propose`/`Approve`/`Withdraw` callers | AST tests |
| Timeout read as success, or divergence between disk and memory | `activation_unknown`, trust hold, sessions ended, no retry; replay at start | Fault tests |
| Restoring an older trust configuration through ledger isolation or replacement (M3, N1, N7) | Isolation publishes the latest verified trust record and refuses trust commands; replacement raises the generation | Isolation and replacement tests |
| A trust hold used to block containment | The hold allows every decrease and revocation; registry-enumerating test | Hold-split tests |
| Evidence computed with a gate decision that ignores the client or validity | `ExplainInstallation` with client intersection and explicit `now`; AST-restricted caller | `ExplainInstallation` tests |
| Ledger edited or truncated at a record boundary, directory rolled back | Chain is tamper evidence for edits only | Residual R1 |
| Disclosure of other workspaces' changes, including through the step-up candidate lookup | Executor generic not-found; workspace-scoped lookup; admin-only read | Leak and step-up tests |
| Rationale or credentials in logs or ledger | No rationale in records; ID-only log lines | Byte-search test |
| Draft flooding | Bounds | Bounds test |
| AI or system principal acting | Gate human-administrator rule; module kind check | Permission tests |
| Step-up bypass | Registry rule (HYB-293) | Step-up tests |

# Residuals

- **R1.** Ledger rollback, truncation at a record boundary and loss of the data directory are undetectable without an independent copy (HYB-298 R1). Under isolation the published rules are those of the latest verified record, which may be older than the last activation. If both the marker and the current ledger are unusable, the operator-stated generation is trusted once it passes validation against every retained file; if the operator also deleted the retained files, any `N > 0` is accepted (R4-5). Deleting both the marker and the ledger restarts at `g = 0` with no warning (HYB-298's first-start path). A replacement without a carryable trust record reverts the rules only with `-accept-trust-reset`, or carries an operator-named ledger's record with `-carry-trust-from`.
- **R2.** Changes not activated, and all rationales and histories, are lost on restart. This is a provisional deviation from "retain reasons and history" (`M02/spec.md:130`), open as Q4.
- **R3.** Only the local password route exists; issuer binding, provider diagnostics and test login, key rotation and external status are not connected.
- **R4.** No enrolment: trust rights come only from the seed and the bootstrap. A non-sample installation can never approve or activate a trust change until the owner decides enrolment (Q3). Trust grants have no expiry by default; revocation of a trust right is possible only by suspending the holder or changing their role in M01, since M03 now excludes installation-kind grants.
- **R5.** Separation is by internal user ID; one person with two administrator accounts is not detected.
- **R6.** Reads admitted just before an activation may complete (`unknown` item).
- **R7.** Every activation signs out everyone in the workspace (Q5).
- **R8.** The trust hold after an unknown activation pauses every non-decreasing mutating command in the workspace until restart; decreases and revocations stay available (Q6). Invitation acceptance, set-link redemption and `identity.issue_reset_link` are refused, so account setup and issuing new reset links wait for the restart; an already issued reset link can still be redeemed (R4-1, R4-6, round 5 default).
- **R9.** The sample seed has three administrators; runner expectations adjusted with reasons. KO and AR copy is an unreviewed AI draft. Screen acceptance belongs to the owner.
- **R10.** Legitimate-authority proof is only the validator's password session; named owners approving proof and recovery procedures (`access-and-trust.md:28`) do not exist.
- **R11.** Downgrading to a build with HYB-298 but without HYB-297 isolates the ledger, because that build does not accept `trust_activation` records or the generation field.
- **R12.** Allowed hold rows that stop an outbound integration effect (`plugin_stop`, `plugin_disable`, `subscription_pause`, `subscription_revoke`) are root classifications under the hold rule, beyond the reviewer's named list (Q6).

# Root defaults (open for the owner)

| Default | Value | Owner question |
| --- | --- | --- |
| Activator separation | Three distinct people | Q1 |
| Test login for local password | Validator's session judged under the candidate | Q2 |
| Bootstrap administrator receives the four trust rights | Yes | Q3 |
| Trust grant validity | No expiry (gate maximum) | Q3 |
| History of non-activated changes | In memory only (deviation from `:130`) | Q4 |
| Sign-out scope at activation | Everyone in the workspace | Q5 |
| Trust hold after an unknown activation | End all sessions; refuse `trust.*`, HYB-298's increase table and other non-decreasing mutations; allow every decrease, revocation, ending sessions, logout, password change and password confirmation | Q6 |
| Ledger isolation or `stopped` (including a failed start-time `Sync`) | Publish the latest verified trust record; refuse all `trust.*` commands; other commands follow HYB-298's rules | Q6 |
| Trust hold table | Allow every command that only reduces access, stops a harmful effect, withdraws a pending proposal or places a hold (table in Activation); refuse the rest; default-deny for unknown IDs | Q6 |
| Epoch across a ledger replacement | Generation counter `(g << 32) + n`; latest trust configuration carried forward; `-accept-trust-reset` when none exists; operator-stated generation validated against retained files when the marker is unusable | Q7 |

# Questions for the owner

1. **Q1.** Keep three distinct people, or allow the approver to activate as the spec's "approver/activator" grouping permits?
2. **Q2.** Accept the validator's password session as the local-password test login, or block activation until a separate test-login feature exists?
3. **Q3.** Decide trust enrolment: how further administrators receive trust rights (for example two-person M03 requests limited to holders), whether bootstrap holds them, and whether trust grants expire. Until then non-sample installations cannot complete any trust change.
4. **Q4.** Record rejections and cancellations (without rationale) durably, to meet `M02/spec.md:130` across restarts?
5. **Q5.** Keep signing out everyone on every activation, or later invalidate only step-up evidence for tightening-only changes?
6. **Q6.** Confirm the trust hold table: in particular the allowed rows `service.suspend`, `service.retire`, `service.revoke_credential`, `identity.revoke_invitation`, `access.reject_request`, `access.withdraw_request`, `recovery.lock`, `retention.place_hold`, `retention.stop_run`, the pending-proposal withdrawals, the root-classified integration stops, and the refusal of set-link redemption with reset-link redemption allowed; and that contract changes stay refused. During isolation or a stopped ledger, should anything beyond trust commands and HYB-298's increase table be held?
7. **Q7.** Accept the generation counter with carry-forward of the latest trust configuration, the `-accept-trust-reset` acknowledgement, and an operator-stated generation validated against retained files?

# Files to change (expected sizes)

No file exceeds 1,500 lines.

| File | Change | Expected lines |
| --- | --- | --- |
| `modules/trustchange/types.go`, `registry.go`, `activate.go`, `digest.go` (new) | module | ~170, ~380, ~170, ~130 |
| `modules/trustchange/registry_test.go`, `activate_test.go`, `property_test.go`, `source_test.go` (new) | tests | ~450, ~380, ~280, ~50 |
| `modules/authpolicy/registry.go`, `evaluate.go`, `types.go` | refuse revisions; `Publish`, `CanPublish`, `Judge`, `Reachable`; `Epoch`, `Activator` | 275 → ~330; 92 → ~135; 171 → ~180 |
| `modules/authpolicy/*_test.go` | updated and new | each < 400 |
| `modules/auth/types.go` and tests | four actions; exported `InstallationAction` | 301 → ~320; +~90 |
| `modules/auth/listing.go` (or a new `explain.go`) and tests; `modules/gate/gate.go` re-export | `ExplainInstallation` (N3) | +~60; +~150 |
| `modules/gate/gate.go`, `types.go` | predicate use and re-exports | 296 → ~300 |
| `modules/commands/types.go` and tests, testdata | codes and handler set | 317 → ~330; +~60 |
| `modules/decisionledger/types.go`, `record.go`, `ledger.go` and tests (after HYB-298) | trust record; generation in marker and genesis | +~110; +~180 |
| `apps/api/server/access.go`, `accessrequests.go`, `ledger.go`, `integrations.go` (after HYB-298) | M03 guard (B1, N4, N5) | a few lines each |
| `apps/api/server/access_test.go`, `accessrequests_test.go`, `ledger_test.go` | M03 guard tests | +~120 total |
| `apps/api/server/policy.go` | read only | 297 → ~270 |
| `apps/api/server/trust.go`, `trustevidence.go`, `trustledger.go`, `trusthold.go` (new) | commands; evidence and recovery; record, replay, generation, progress; hold table | ~390, ~260, ~230, ~110 |
| `apps/api/server/trust_test.go`, `trust_validate_test.go`, `trust_activate_test.go`, `trust_restart_test.go`, `trust_property_test.go` (new) | API and server-property tests | each < 650 |
| `apps/api/server/policy_test.go`, `stepup.go`, `stepup_test.go` | read tests; action map and scoped candidate window | 446 → ~250; 136 → ~145; 569 → ~610 |
| `apps/api/server/session.go`, `commands.go` | `trust_epoch`, sweep with pickups, hold checks | 170 → ~210; 219 → ~245 |
| `apps/api/server/seed.go`, `signin.go`, `server.go`, `admin.go`, `errors.go`, `openapi.go`, `world_test.go` | third admin, trust grants, genesis after ledger open, bootstrap rights, wiring, codes | small to moderate; all < 500 |
| `apps/console/src/service/admin.ts`, `admin.test.ts` | types, IDs, failures | 612 → ~690; 316 → ~370 |
| `.../authentication/Live.tsx` | refactored wiring | 406 → ~300 |
| `.../authentication/TrustChanges.tsx`, `TrustActions.tsx`, `TrustConfirm.tsx` (new) | list and detail; forms; confirmations C1–C6 | ~350, ~330, ~260 |
| `.../authentication/trust-model.ts`, `trust-model.test.ts`, `trust-copy.ts`, `trust-render.test.tsx` (new) | model, copy, tests | ~220, ~280, ~340, ~180 |
| `.../authentication/live-copy.ts`, `live-model.ts`, `live-model.test.ts` | removed proposal copy; new failures | each < 220 |
| `apps/console/package.json` | test list | one line |
| `docs/ui/terminology/product-glossary.md` | new rows | +10 rows |
| `docs/features/HYB-297/plan.md`, `verification.md` | plan; verification record | — |

# Facts, assumptions and unknowns

**Facts** are the file:line citations above, read on `be1bfa3` and in the HYB-298 worktree on 2026-11-18.

**Assumptions** (not verified):
- A1. HYB-298 merges with `modules/decisionledger` and its M03 files substantially as in its worktree today. If the API or the guard sites change, the affected steps are re-reviewed.
- A2. No executor step after the handler depends on the caller's identity session (read at `modules/commands/execute.go:378-385`, `:399-433`); confirmed by the sweep-in-handler test.
- A3 (round 1) was false and is removed: `Explain` refuses installation actions and has no client (`modules/auth/listing.go:73-77`). Replaced by the declared `ExplainInstallation` export (N3).
- A4 is verified: `passwords.Registry.Has(key)` reports whether a login has a credential without verifying a password (`modules/passwords/registry.go:108-112`).
- A5. A third sample administrator does not break HYB-298's seed grants or replay.

**Unknowns:** owner answers to Q1–Q7; the visual layout until the `ui-designer` specification and references are read; final KO and AR terms until TW review.

# Design review round 1 (2026-11-18)

Separate read-only reviewer, security focus. Result: **FAIL** (1 blocker, 5 major, 14 minor, plus test gaps). Root decisions relayed on 2026-11-18. All findings are folded into this revision; it has not been re-reviewed.

| Finding | Severity | Where the plan now handles it |
| --- | --- | --- |
| B1: M03 filters on `administer_installation`, not the installation kind; trust rights could be requested, listed, revoked and replayed | Blocker | Current behaviour (corrected fact); In scope; Trust rights "M03 guard"; Dependency point 4; Tests "M03 guard"; QA I; Threats; Files |
| M1: evidence must bind user state, user epoch, kind and the effective gate decision per right | Major | Validation "Evidence digest facts"; Tests (user-level suspension, grant expiry, client grant); Threats |
| M2: `activation_unknown` must fail closed | Major | Activation "`commit_unknown`" (trust hold, `accepted` with job `waiting`); `access` and `execute` hold checks; States; Tests; R8; Q6 |
| M3: isolation could restore an older trust configuration; replacement restarted epochs at 1 | Major | Restart steps 1 and 5 (refuse `trust.*` while isolated, epoch floor in genesis); States; Tests; Threats; R1; Q6, Q7 |
| M4: enrolment blocks every non-sample installation | Major | Trust rights "Enrolment blocks"; Actor rule; holders line and enrolment copy; Cross-check status `blocked`; Tests; QA H; R4; Q3 |
| M5: spec confirmations not designed; cross-check over-claimed | Major | Console "Confirmations" C1–C6; cross-check rows for `:96-101` cite them; `:130` history row now a provisional deviation |
| m1: omitted sources | Minor | Cross-check: `retention-access.md:49`, `access-and-trust.md:28` (R10) and `:30`, TC-01..TC-03; IAM-01 as regression |
| m2: no undeclared change to `self_approval_denied` | Minor | Refusals: status 403 and next action `none` unchanged |
| m3: scope the candidate lookup; one captured `now`; MFA unreachable | Minor | Step-up interaction; Validation (`Judge(…, now)`) |
| m4: exact recovery definition | Minor | Validation "Recovery path, exactly" |
| m5: pickups and service credentials | Minor | Activation sweep step 4; invalidation table |
| m6: `Publish` failure after a receipt | Minor | Activation "`Publish` failure after a receipt"; Tests |
| m7: AST test on `Propose`/`Approve`/`Withdraw` callers | Minor | Tests "Source and AST tests" |
| m8: lock order and evidence outside the module mutex | Minor | Dependency "Lock order" |
| m9: no ID reuse across restarts; replay restores the policy number | Minor | Objects (run prefix); Restart step 4; Tests |
| m10: no-policy installations | Minor | Objects "No policy yet"; Commands; Tests |
| m11: code for approving a never-validated draft | Minor | State machine (approve row) and Refusals: `invalid_stage` |
| m12: Q4 as a provisional deviation | Minor | Cross-check row `:130`; R2; Root defaults; Q4 |
| m13: field name | Minor | `trust_epoch` throughout |
| m14: `commit_failed` copy | Minor | Activation "Ledger refusal … `commit_failed`" |
| Test gap: `commit_unknown` revocation then activation | — | Tests "timeout" list |
| Test gap: isolation with the trust record after the failing line | — | Tests "Ledger states" |
| Test gap: server-level properties | — | Tests "Server-level properties" |

**Not fully resolvable in the plan:** M4 is resolved only as a truthful block; making non-sample installations able to complete a trust change needs the owner's enrolment decision (Q3). The clock dependence noted after round 1 is removed by N7.


# Design review round 2 (2026-11-18)

Same reviewer, security focus. Result: **FAIL** (no blocker, three majors, six minors). Root decisions relayed on 2026-11-18. All findings are folded into this revision; it has not been re-reviewed. Round 1 entries above that mention a clock-based epoch floor or a seed-only isolation fallback are superseded by N1 and N7.

| Finding | Severity | Where the plan now handles it |
| --- | --- | --- |
| N1: isolation should publish the latest verified trust record, not fall back to the seed | Major | Restart step 1; Tests "Ledger states" (inverted: the record before the failing line is active); Threats; R1; Root defaults (Ledger isolation) |
| N2: the trust hold must never refuse decreases | Major | Activation "`commit_unknown`" (refused and allowed lists, `holdAllows`, citing HYB-298 `plan.md:335`, `:507-510`, `:524-527` and `retention-access.md:59`); Enforcement; `activation_unknown` copy in EN, KO, AR; Tests "Trust hold split"; Threats; R8; Root defaults and Q6 |
| N3: A3 false; `Explain` refuses installation actions and has no client | Major | In scope (`ExplainInstallation`); Validation "Evidence digest facts"; Tests "`ExplainInstallation`"; Threats; Files; Assumptions (A3 removed, A4 verified) |
| N4: installation-kind grants must resolve as not found in M03 revoke paths | Minor | Trust rights "M03 guard" (revoke target validation); Tests (byte-identical `permission_denied`) |
| N5: `integrations.go:334` | Minor | Trust rights "M03 guard"; Tests; Files |
| N6: start-time `Sync` moved into HYB-298 | Minor | Dependency point 5; Restart step 6; "readable at start" copy |
| N7: generation counter instead of a clock floor | Minor | In scope; Dependency point 6; Objects "Epoch values" (`(g << 32) + n`, shown as "g.n"); Activation step 2; Restart steps 3 and 5; Read (`ledger.generation`); Tests; R1; Root defaults and Q7; Files |
| N8: run prefix collision and "earlier run" label | Minor | Objects (redraw on match, `run_started_at`); Tests "Restart" |
| N9: service credentials reason | Minor | Invalidation table |

**Not resolvable in the plan:** none of the round 2 findings. The open owner decisions remain Q1–Q7, and M4 stays a truthful block until enrolment is decided (Q3).


# Design review round 3 (2026-11-18)

Same reviewer, security focus. Result: **FAIL** (no blocker, three majors, three minors). Root decisions relayed on 2026-11-18. All findings are folded into this revision; it has not been re-reviewed. The round 2 `holdAllows` description is superseded by the hold table.

| Finding | Severity | Where the plan now handles it |
| --- | --- | --- |
| R3-1: the hold refused safe-direction commands; its test was vacuous | Major | Activation "`commit_unknown`" (rule, two-sided hold table with a reason per row, default-deny for unknown IDs); copy; Tests "Trust hold table" (missing-ID failure, both sides compared, synthetic unknown ID refused); R12; Root defaults; Q6 |
| R3-2: a replacement ledger could revert the trust configuration | Major | Restart step 5 "Carry forward" and `-accept-trust-reset`; Read (`origin: carried`, `carried_from`); Tests "Carry forward"; R1; Root defaults; Q7 |
| R3-3: start-time `Sync` failure (HYB-298 amendment 7 corrected) | Major | Dependency point 5; State machine rule (stopped refuses `trust.*`); Restart step 6; Read (`durable_unconfirmed`); Tests "Start-time `Sync` failure" and "Ledger states"; Root defaults |
| R3-4: generation validation and epoch edges | Minor | Objects "Epoch values" and "No policy yet" (`n = 0`, `n ≥ 2^32` refused, separate `epoch_generation` and `epoch_number`); Restart step 5 (operator-stated generation validated against retained files; deleting marker and ledger restarts at 0); Tests; R1; R11 (downgrade isolates) |
| R3-5: `ExplainInstallation` callers and rules | Minor | In scope (two callers, delegations skipped, human-kind rule in auth, role from identity facts); Tests "`ExplainInstallation`" |
| R3-6: rebase notes | Minor | Dependency point 7 (`administer_installation` revoke refusal becomes `permission_denied`; HYB-298 tests and M03 copy follow) |

**Not resolvable in the plan:** none of the round 3 findings. The hold table must be re-enumerated from the registry at implementation time (the plan's table is from the HYB-298 worktree), and the integration-stop rows (R12) and the whole table remain owner-confirmable (Q6). M4 stays a truthful block until enrolment is decided (Q3).


# Design review round 4 (2026-11-18)

Same reviewer, security focus. Result: **FAIL** with two narrow majors; all round 3 findings resolved. All round 4 findings are folded into this revision; it has not been re-reviewed.

| Finding | Severity | Where the plan now handles it |
| --- | --- | --- |
| R4-1: sign-in accepted invitations and set-link redemption activated accounts during the hold; the test did not cover routes | Major | Hold table rows for `POST /auth/login` (no `acceptPending`, `signin.go:265`), `POST /auth/redeem` (refused, `signin.go:399-412`), `POST /auth/link` and the command route; the route enumeration note; handler checks; Tests "Trust hold table"; R8; copy in EN, KO, AR; Q6 |
| R4-2: carry-forward without checking the ledger ID and workspace | Major | Restart step 5 "Carry forward" (carryable only when genesis ID equals marker ID and workspace is the boot workspace) and "ID mismatch or unusable marker" (`-accept-trust-reset` or `-carry-trust-from <ledger-id>`, recorded in the genesis); Tests "Carry forward"; R1 |
| R4-3: first activation after a replacement could get `n = 0` | Minor | Activation step 2 (`E1 = max(E0 + 1, (g << 32) + 1)`, no policy is `g << 32`); Tests "Epoch after replacement" |
| R4-4: "verified" for `integrity_failed` and `epoch_order`; older-configuration wording | Minor | Restart step 1 (prefix definitions, "(problem at line X)"); Restart step 5 carried-record wording; Tests |
| R4-5: retained files deleted; marker unreadable while the ledger verifies | Minor | Restart step 5 "Generation source" (`g` from the verifying genesis, HYB-298 `plan.md:482`; flag only when both are unusable); R1; Tests |
| R4-6: `identity.issue_reset_link` refused during the hold | Minor | R8; hold copy in EN, KO, AR |

**Not resolvable in the plan:** none. Open owner questions remain Q1–Q7; M4 stays a truthful block until enrolment is decided (Q3).


# Design review round 5 (2026-11-18)

Same reviewer, security focus. Result: **PASS**, with two minors and one root default change, folded in below. Status set to "approved after design review round 5 (2026-11-18)" on root's instruction. This is design approval only; the change review, verification and owner acceptance are still pending.

| Item | Where the plan now handles it |
| --- | --- |
| R5-1: where the hold check runs in sign-in and redeem; link stays unconsumed on refusal | Activation "`commit_unknown`" (check locations: `signin.go:254`, `:265`, `:382`, `:393`, `:399`, `:410`); Tests "Trust hold table" (race between `Reserve` and `Commit`) |
| R5-2: route classification test covers every non-GET method | Hold route note (`TestPostRoutesAreClassified`, `stepup_test.go:137-166`) |
| Q6 default: reset-link redemption allowed during the hold; set-link redemption refused | Hold table `POST /auth/redeem` row; hold copy in EN, KO, AR; R8; Q6 |

## Root amendments after the change reviews (2026-11-18)

From the independent security, render and technical-writing reviews of the implementation; decided by root under the owner's standing instruction, all fail closed. Owner questions Q1–Q7 stay open.

- **A1 (S2, revised after the security re-review N3).** On `id_mismatch` with a readable marker of generation G, a record from a lower generation is not published as it is. For each sign-in rule the stricter of the installation default and the ignored record's value applies (shorter session limits, MFA required if either requires it, sign-in methods the intersection, or the ignored record's methods if the intersection is empty), and the read reports `older_record_ignored`. Step-up rules are tightened per action the same way (shorter limit, MFA if either requires it), capped at the session limit; the changed fields are reported in `trust.stricter_fields`. On an installation without the sample seed the ignored record's configuration applies at G<<32+1, number 1, compared against a base with no limits, with the same `stricter_fields` and `older_record_ignored` reporting.
- **A2 (S4).** A re-validation with the same digest and no blocked item keeps the approval. Only a changed digest or a blocked result invalidates it.
- **A3 (S5).** A duplicate activation is refused at commit. In a crafted ledger it isolates the ledger with `epoch_order`, the prefix ending before it, and `-accept-new-decision-ledger` recovers. Start does not fail without a way out.
- **A4 (S1).** The floor rises with each retained generation (g + 1) and with the highest generation in the unverified current file. A stated generation must be at least floor + 1.
- **A5 (S8, S9).** The holders line uses the recovery finding (`holders.ready`). The validate candidate window applies only to holders of `trust_validate`.
- **A6 (copy).** These EN strings changed:
  - the blocked sentence;
  - the stale sentences, one per state;
  - the `activation_unknown` lead: "could not confirm that the activation record was saved", "stop a running operation such as a deletion run", "place a retention hold";
  - "Qualifying administrators now: N.";
  - "Restored from the decision ledger at start";
  - the durable-unconfirmed label;
  - the after-activation sentence.
- **A7 (security re-review N1, N2).** A ledger generation is below 2^32−1 in every source (genesis, marker, `-trust-generation`, retained files, the unverified current-file scan); a genesis or trust record at or above the bound in the current ledger is not well shaped and isolates the ledger; a marker at or above it is unusable; retained files and the unverified scan ignore such lines; `-trust-generation` accepts canonical 1 to 2^32−2; default and stated successors refuse to reach the bound (`ErrGenerations`) rather than saturating. The unverified scan counts only canonical, well-shaped lines and is streamed with the ledger's line and size limits.
- **A8 (security re-review N6).** A same-digest re-validation writes its own history entry `revalidated_approval_kept` ("Validated again; the approval is kept.").
- **A9 (security re-review N5).** The policy read adds `holders.missing` (the rights no qualifying administrator holds); with `ready:false` the enrolment notice shows the missing-rights sentence.
- **A10 (copy).** User-facing EN strings follow the technical-writing reviews (rounds 1–2, items C1–C30 and T1–T11 recorded in verification.md); where this plan quotes earlier EN copy, the strings in `trust-copy.ts` are authoritative.
