---
type: ImplementationPlan
title: Retention register with class policy revisions anchored at an immutable start, requires and mentions dependencies, fast legal holds released under dual control, preview-first chunked deletion runs that re-check before every chunk, and tombstones with a watermark for restore checks
description: First increment of work item W-43. Artifacts are registered with a data class, tenants and an immutable anchor. Class policies set retention periods. Dependencies and legal holds make an artifact ineligible. Deletion runs fix a manifest, block access, and let a worker delete per chunk and store after a re-check. Every deletion leaves a tombstone with a monotonic sequence, and a deleted artifact can never be registered again.
status: stable
---

# Scope and ownership

[HB-256](https://linear.app/hybrid-billing/issue/HB-256) is the first increment of work item W-43 (requirement HB-43). It covers:
- **From HB-43-I.** RetentionPolicyRevision, ArtifactDependency, LegalHold, DeletionPlan/Run and DeletionTombstone, executed as impact preview → permission, hold and reference re-check → logical block → chunked physical deletion → verification. The retention of final evidence and minimal duplicate-prevention records is a separate class.
- **From HB-43-V.** Deletion racing with approvals and holds, interruption mid-run, and re-exposure after a backup restore (through the tombstone watermark), and duplicate billing from re-receiving deleted source (re-registration is refused).
- **From AR-05 and QA-10.** Assistant conversations are their own class. Deleting a conversation never deletes business evidence, because evidence only *mentions* a conversation.
- **From retention-access.md.** The anchor is never reset by re-collection, copying or archiving (the anchor is immutable; re-collection under a deterministic ID is idempotent). A shortening is approved only after an impact preview. A shared payer source is deleted only after every requiring reference is gone. A hold on an already deleted artifact is never reported as preserving it. Counts reconcile: target = deleted + held + referenced + failed + pending. Completion waits for backup expiry.

It does not cover:
- the actual periods, legal hold owners and deletion targets. These are owner and legal decisions, and every period here is configured by a revision;
- storage adapters, physical deletion, archive verification and key destruction. The worker reports what it did;
- applying tombstones to a restored database. This module provides the ordered tombstones and watermark that a restore procedure must replay;
- persistence and screens.

Base: main `f0c7f78`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the design and the change, because deleting evidence and tenant data is irreversible. Machine: mac_mini.

Allowed files: `modules/retention/*` and `docs/features/HB-256/*`.

# Observed baseline

- `modules/assistant` already applies its own conversation retention (`conversation_retention`) inside the Assistant. This module is the installation-wide register; wiring the Assistant's expiry to it is a residual.
- No module records retention classes, holds, dependencies or tombstones.
- Modules in this repository use only the standard library, a single mutex, human-only proposals with `ai_not_allowed` and `not_human`, and system actors for workers.

# Decision

## Classes

The closed class list, from the retention table:

| Class | Meaning |
| --- | --- |
| `source` | received source and normalized detail |
| `final` | final billing, policy, FX and calculation manifests |
| `claim` | claim, received identity and external document links (minimal duplicate-prevention records) |
| `identity` | user profile, identity and access audit |
| `conversation` | Assistant business conversations, attachments and tool-run references |
| `temporary` | temporary uploads, previews, exports and failed files |
| `projection` | caches, search and projections |

Backups, old object versions and replicas are not a class. They are covered by the backup expiry wait.

## Artifacts

A system actor (the ingesting component) registers an `Artifact`:
- `ID`;
- `Class`;
- `Tenants`, a strictly ascending list of 1 to 64 tenant IDs, so a shared payer source lists every tenant;
- `Anchor`, the time from which retention runs;
- `Stores`, a strictly ascending list of 1 to 16 store IDs where its bytes live.

The class, anchor and stores are immutable. A system actor may add a tenant later (`AddTenant`), so a shared source that gains a customer is covered by that customer's holds; tenants are never removed, and a deleted artifact is refused with `deleted`.

Registering the same ID again with the same content is idempotent; with different content it is `ErrConflict`. Registering an ID that has a tombstone is refused with `deleted`. This prevents re-received source from coming back only when callers use the deterministic source or claim identity as the ID, which is a caller obligation; ingestion that assigns fresh IDs is not protected here. An anchor after the clock is a form error.

## Dependencies

A system actor records `Reference{From, To, Kind}` between two registered artifacts that are not deleted:
- **`requires`**: `To` must exist while `From` exists. For example, a final manifest requires its source.
- **`mentions`**: `From` only names `To`. For example, an evidence record mentions an Assistant conversation.

An artifact with a `requires` reference from an artifact that is not deleted is ineligible (`referenced`). `mentions` never blocks. A reference to or from an artifact whose outcome is `deleting` or `failed`, or that is damaged (see below), is refused with `deleting`. A reference to a pending, blocked artifact is accepted, and the next claim re-check excludes it. Self-references are form errors and repeated references are idempotent.

## Policy revisions

A `Policy{Class, Period, Grace}` revision is proposed by a human and approved by a different human. `Period` is 1 to 36,500 days and `Grace` is 0 to 365 days. Revisions are numbered per class.

An artifact is due at time `t` when `Anchor + Period + Grace ≤ t` under the class's current revision. A class with no revision is never due.

**Shortening.**
- A revision whose `Period + Grace` is shorter than the current one is a shortening. When it is proposed, the module computes its `Impact{AsOf, Count, Digest}` with `AsOf` set to the clock, and stores it in the proposal. The proposer cannot choose `AsOf`.
- The impact set is the artifacts of the class, not deleted, that are due at `AsOf` under the new values and not due under the current ones. The digest is the SHA-256 of their IDs in order, each length-prefixed.
- The `Impact(class, period, grace)` read computes the same at the clock and also returns per-tenant counts and how many of the set are held or referenced.
- The approver of a shortening must present the impact it reviewed, read at most one hour before approval. The module recomputes the set at that impact's `AsOf`; if the count or digest differs, the approval is refused with `impact_changed`. So the approver binds the impact as of approval, not the proposer's stale one.
- A lengthening or first revision carries no impact. Bytes are not counted here.

## Legal holds

A `Hold{ID, Owner, Reason, Artifacts, Tenants, Classes}` covers:
- the listed artifacts;
- plus every artifact of a listed tenant, restricted to the listed classes when any are listed.

At least one artifact or tenant is required, and classes without tenants are a form error. Every listed artifact must be registered (`ErrNotFound`).

**Placing.** A single human places a hold. It is never refused for the state of the artifacts it covers: it takes effect at once for everything covered, because preserving is the safe direction. The result lists, for every covered artifact that is not fully preserved, its outcome:
- `not_preserved`: already deleted;
- `outcome_pending`: claimed and being deleted;
- `partially_deleted`: failed with at least one store deleted.

The result carries the counts of each outcome. The artifacts themselves, with their per-store state, are read in pages of up to 1,000 with `HoldOutcomes(id, after, limit)`, recomputed when read, so a pending outcome later shows as `not_preserved` or `partially_deleted`. Nothing reports a deleted artifact as preserved.

**Releasing.** A hold is released by a proposal and approval of two humans, neither of whom placed it.

## Eligibility

An artifact is eligible when it is registered, not deleted, not owned by a run, and due, and when:
- no active hold covers it (`held`);
- no artifact that is not deleted requires it (`referenced`).

`Preview(class, after, limit)` returns the eligible artifacts of the class in ID order after `after`, at most `limit` of 1 to 10,000. It also returns whether more follow, and counts of due artifacts that are held or referenced.

## Deletion runs

**Plan.** A human proposes a `DeletionPlan{Class, Artifacts, Chunk, Lease, Reason}`:
- `Artifacts` is a strictly ascending list of 1 to 100,000 IDs, normally from `Preview`;
- the chunk size is 1 to 1,000;
- `Lease` is 1 minute to 24 hours, how long a claim stays exclusive.

A different human approves it.

**Approval.**
- Every artifact must be registered (`ErrNotFound`), of the plan's class (`ErrInvalid`), not deleted (`already_deleted`) and not owned by another run (`in_plan`); otherwise the whole approval is refused.
- The run records the class's current policy revision number.
- Each artifact is re-checked: held, referenced and not-due artifacts get those outcomes and are released at once. The rest are `pending`, blocked at once (`Access` returns `blocked`) and owned by the run.
- The run gets the ID `run/<n>`, with `n` a sequence that only increases.

**Outcomes.** Each manifest artifact has one outcome. The count equation always holds:

target = deleted + held + referenced + not_due + stopped + failed + deleting + pending.

An artifact is owned by the run while its outcome is `pending`, `deleting` or `failed`. Every other outcome releases ownership.

**Damage.** Once any store of an artifact is reported deleted, the artifact is *damaged*. The flag is never cleared. A damaged artifact that is not fully deleted is always `blocked` for `Access` and refuses `requires` references, whatever its outcome, ownership or holds. A hold shows it as `partially_deleted`.

**Chunks and leases.**
- Chunks are formed over the manifest order; chunk `i` covers manifest positions `i*chunk` to `min((i+1)*chunk, s) - 1`.
- A system actor calls `Claim(run, i)`. Chunks are claimed in order: claiming a chunk after the next unclaimed one is `out_of_order`.
- `Claim` re-checks every `pending` artifact of the chunk under the class's current revision. Held, referenced and not-due artifacts get those outcomes and are released. The rest become `deleting`.
- The claim returns a fencing token and the `deleting` IDs, and holds the chunk for `Lease`.
- Claiming a chunk again within its lease is refused with `leased`. After the lease expires, a new claim:
  - issues a new token, so reports with the old token are refused with `stale_token`;
  - does not re-check anything: every `deleting` artifact stays `deleting`, because the earlier worker may have deleted bytes it never reported.
- `Retry(run, i)` is a claim of a done chunk's `failed` artifacts. It re-checks each one; held, referenced and not-due ones get those outcomes and lose ownership, but a damaged one stays blocked by its damage. The rest become `deleting` again under a new token and lease. A retry does not affect claiming later chunks: the next chunk can be claimed once every earlier chunk has been done once.
- **Proceed.** Before deleting an artifact's bytes in each store, the worker should call `Proceed(run, i, token, id)`. It is false when a hold now covers the artifact. (New `requires` references to a `deleting` artifact are already refused.) The worker then reports that store as `failed`, and the next retry re-checks it. A hold placed during a live lease is shown as `outcome_pending` until the reports resolve it.

**Reports.**
- A system actor calls `Report(run, i, token, store, deleted, failed)`, normally with the chunk's current token. With the current or an older token of the same chunk, a report may record `deleted` results for any artifact of the chunk that lists the store and is not fully deleted, whatever its outcome now, because a deletion is a fact that is safe to record. It sets the damage flag, and if every store is then deleted the artifact becomes `deleted`, in the reporting run and in the run that owns it, with a tombstone under the owner. `failed` results need the current token (`stale_token` otherwise) and an artifact that is `deleting` (`not_in_chunk` otherwise). `deleted` and `failed` are disjoint and strictly ascending.
- Per (artifact, store), `deleted` is terminal. `failed` may later become `deleted`. A `deleted` store reported as `failed` is refused with `already_deleted`, and an identical repeat is idempotent.
- An artifact becomes `deleted` once every one of its stores is deleted. It then gets a tombstone. When every store has reported and at least one failed, it becomes `failed`.
- A chunk is done when none of its artifacts is `deleting`. The next chunk can be claimed once every earlier chunk has been done once.

**Stop.** A single human may stop a run, because stopping is the safe direction. Every `pending` artifact becomes `stopped` and is released. A claimed chunk still resolves by its reports. No new chunk can be claimed, but retries of `failed` artifacts remain possible. A stop does not restore deleted bytes.

**Backup expiry.**
- A human records `BackupExpired(store, before)`, meaning every backup and object version of that store taken before `before` has expired.
- `before` never moves backward and is never after the clock.
- A deleted artifact is complete when, for every one of its stores, its deletion time is earlier than that store's recorded `before`.

**Run states.**

| State | When |
| --- | --- |
| `running` | not stopped, and some chunk is not done |
| `stopped` | stopped, and some chunk is not done |
| `done` | every chunk done (or stopped), and some artifact is `failed` or some deleted artifact is not complete |
| `complete` | every chunk done (or stopped), no `failed`, no damaged artifact that is not fully deleted, and every deleted artifact complete |

Only `running` and `stopped` runs count as unfinished for the cap of 100. A `done` run can still return to `complete` through retries and backup expiry. Ownership (`in_plan`) follows the outcomes above, not the run state.

`Run(id)` returns the outcome counts, the number of damaged artifacts that are not fully deleted, the next chunk, the policy revision, the state and per-store counts. For each store `s`, its target is the number of manifest artifacts that list `s`, and target = deleted + failed + unreported, where each artifact's result for `s` is `deleted`, `failed` or none.

## Tombstones and watermark

Each deletion appends a `Tombstone{Sequence, Artifact, Class, Run, At}`, where `Sequence` starts at 1 and increases by one each time. `Tombstones(after, limit)` returns up to 1,000 in sequence order, and `Watermark()` returns the latest sequence.

A restore procedure must replay every tombstone up to the watermark before reopening access. `Restored(applied)` is true only when `applied` equals the watermark.

This register is the authority for blocked and failed artifacts too. After a restore, callers keep checking `Access` from this register, which is kept outside the restored database, before serving any bytes.

Tombstones are kept as the minimal record that refuses re-registration. Whether and when they may expire is an owner and legal decision, listed as a residual.

## Access

`Access(id)` returns:
- `ok`;
- `blocked` (owned by a run, including `failed`, or damaged and not fully deleted);
- `deleted`;
- or `ErrNotFound`.

Callers must check it before serving bytes. A held artifact that is not damaged is `ok`.

## Actors

- Registering, adding tenants, referencing, claiming, retrying and reporting are for system actors. Humans get `not_system` and assistants `ai_not_allowed`.
- Policies, plans, approvals, holds, releases, stops and backup expiry are for humans. `Proceed` is a read for the worker. System actors get `not_human` and assistants `ai_not_allowed`.
- A proposer cannot approve its own proposal (`self_approval`). Only the proposer may withdraw it.

Authorization (who may manage retention for which tenants) is the caller's obligation through `modules/auth`. This module records actor IDs.

## Reads

Every read returns a copy: `Artifact(id)`, `Policy(class)`, `Impact`, `Preview`, `Hold(id)`, `Run(id)`, `Access(id)`, `Tombstones`, `Watermark`, `Restored`, `Proposal`.

## Bounds

| Bound | Limit |
| --- | --- |
| Registered and tombstoned artifacts | 10,000,000 (`ErrLimit` for new IDs) |
| References | 10,000,000 |
| Tenants and stores per artifact | 64 and 16 |
| Holds active | 10,000; artifacts listed per hold 10,000; tenants and classes per hold 64 and 7 |
| Unfinished runs | 100; artifacts per run 100,000; chunk 1 to 1,000 |
| Preview page | 1 to 10,000 |
| Pending proposals | 1,000 in all, 250 per proposer; applied kept 1,000 |
| Identifiers | 1–128 bytes of `[A-Za-z0-9._:-]` (run IDs `run/<n>`) |
| Text | 1–200 bytes of UTF-8, with no control characters |

## Errors

`ErrInvalid` (form), `ErrNotFound`, `ErrConflict` (a changed re-registration), `ErrLimit`, and `ErrRejected` wrapped in a `Refusal` with a reason.

## Residuals

- Retention periods, grace, hold owners, deletion targets, backup windows and tombstone expiry are owner and legal decisions. Every rule here is provisional, including:
  - one person placing a hold, stopping a run or recording backup expiry;
  - two people releasing a hold;
  - the shortening impact check;
  - failed artifacts staying blocked until a retry deletes them.
- Byte counts in impact previews.
- A worker that ignores `Proceed` can still delete bytes after a hold during its lease; the hold then shows `outcome_pending` and the reports record the truth. Testing worker adherence belongs with the storage adapters.
- Storage adapters, physical deletion, archive verification, object versions, caches and key destruction.
- Replaying tombstones into a restored database, and keeping this register outside it.
- Deterministic artifact IDs at ingestion; wiring `modules/assistant` conversation expiry, ingestion, evidence and claim modules to register artifacts and references.
- Tenant-scoped authorization and screens; persistence.
- Store results are kept per artifact, so a later run's reports also change an earlier run's per-store counts for the same artifact. Each run still balances.
- Hold reads scan every registered artifact under the single lock; an index by tenant and class is a residual for the persistence layer.
- RET-01 to RET-08, AUD-AR-05 and AUD-QA-10 are NOT_RUN at system level.

# Verification plan

Literal tests:
- **Forms and bounds.** Every form rule and bound.
- **Artifacts.** Idempotent and conflicting registration; refusal after a tombstone; future anchors.
- **References.** `requires` blocks and `mentions` does not; `deleting` serialization; a conversation deleted while evidence mentions it.
- **Policies.** Due times at the exact boundary; grace; shortening with a matching impact and with a changed count or digest at the same `AsOf`.
- **Holds.** Artifact, tenant and class scope; outcomes `not_preserved`, `outcome_pending` and `partially_deleted`, recomputed on read; tenants added later; dual-controlled release.
- **Runs.**
  - Approval refusals and re-check, and `in_plan`.
  - Block before any chunk.
  - Claims: in order, out of order, `leased`, lease expiry with a new token, and `stale_token`.
  - Holds, references and lengthened policies arriving between approval and claim.
  - Reports: per store, transitions, idempotent repeats, failures and retries, including a retry refused by a hold placed after a failure.
  - Count reconciliation, overall and per store, after every step.
  - Stopping.
  - Per-store backup expiry and every run state.
- **Tombstones.** Sequence, paging, watermark and `Restored`.
- **Concurrency.** Concurrent registration, holds, claims and reports under `-race`.

Property test: random artifacts, references, policies, holds, runs, claims, reports and clock moves. An independent model of eligibility, access, counts and tombstones must match, and the counts must always add up to the target.

Checks: `-race -count=10`, the 386 build, gofmt, vet, `go test -race ./...`, 100% statement coverage and mutation testing.

# Review history

- Round 1 (independent reviewer, DBA focus, read-only): FAIL. Findings and what changed:
  - H1. Retries are claims that re-check holds; a held, partly deleted artifact stays blocked and is shown as `partially_deleted`.
  - H2. Holds are never refused for the state of covered artifacts; per-artifact outcomes are returned and recomputed on read.
  - M1. Runs record the policy revision; claims re-check due-ness under the current revision (`not_due`); approval refuses deleted or foreign manifest entries.
  - M2. The impact check uses a fixed `AsOf`, a digest of the set and per-tenant and held/referenced breakdowns; bytes are a residual.
  - M3. Backup expiry is per store.
  - M4. Per-store counts reconcile.
  - M5. Claims have leases and fencing tokens; per-store report transitions are defined.
  - M6. A single human can stop a run.
  - M7. A run state table, unfinished meaning running or stopped, and ownership by outcome.
  - M8. Re-registration protection needs deterministic IDs (a caller obligation); tombstone expiry is a residual.
  - L1–L6. Tenants can be added; "being deleted" defined; paged preview; `Access` stays authoritative after restore; classes need tenants; more provisional items.
- Round 2 (same reviewer): FAIL. Findings and what changed:
  - N1. A never-cleared damage flag keeps partly deleted artifacts blocked and unreferenceable.
  - N2. Lease expiry re-checks nothing, and stale reports may still record deletions.
  - N3. The module sets the proposal's `AsOf`, and the approver binds an impact read within the last hour.
  - N4. Hold outcomes are counted and paged.
  - N5. The per-store equation is stated.
  - N6. Retries do not gate later chunks.
  - N7. A `Proceed` check for the worker, with the remaining race as a residual.
  - N8. `AddTenant` refuses deleted artifacts.
- Round 3 (same reviewer): PASS. Low items folded in:
  - L1. Chunk gating wording made consistent.
  - L2. Stale reports record late deletions for any chunk artifact that is not fully deleted.
  - L3. Damaged artifacts that are not fully deleted keep a run from `complete`, and are counted.
  - L4. `Proceed` checks holds only; worker adherence is a residual.

# Change review resolution

- Change review round 1 (separate independent reviewer, DBA focus, read-only): FAIL. Findings and what changed:
  - M1. A stale report from one run could delete an artifact owned by another run and leave that run stuck. On full deletion, both the reporting run and the owning run now record `deleted`. The tombstone names the owning run, and the owner's chunk is finished if it was claimed.
  - L1. Re-registration with a subset of the stored tenants is idempotent.
  - L2. `deleted` results are accepted for any chunk artifact that lists the store and is not fully deleted, with the current or an older token. `not_in_chunk` applies only to `failed` results.
  - L3. Per-artifact store results are a stated residual.
  - L4. Hold checks scan only live holds; the artifact scan in hold reads is a residual.
  - L5. An unreachable `stopped` claim refusal was removed.

- Change review round 2 (same reviewer): PASS. Low items:
  - N1. A cross-run deletion is counted as `deleted` in both runs, while one tombstone exists. Sums of `deleted` across runs can exceed the watermark; the watermark is the count of deletions.
  - N2. A partial cross-run deletion settles only the reporting run; the owner settles on its next report or reclaim. Kept as a residual.
  - N3. The run-count checks in the test helper hold by construction; cross-run behaviour is covered by literal tests, not the property model.
