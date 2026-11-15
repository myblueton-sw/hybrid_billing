---
type: ImplementationPlan
title: Recovery generations with an external write lock, in-flight call fencing, verified recovery manifests, external reconciliation fences and ordered re-application of protective events
description: First increment of work item W-37. A register gives each installation a recovery generation. Declaring a restore point under dual control opens a new generation and locks every external writer at once. Permits carry generation fencing tokens, so old workers and the previous issuer are refused. Restored data is verified against the restore point's manifest by item inventory and contiguous range checksums for every domain. Results that external systems confirmed after the restore point are reconciled into numbering high-water fences and delivery fences. Protective events recorded after the restore point, such as deletions, purges, holds and revocations, must be applied again in order. Only a dual-controlled unlock resumes external writes, and only when everything is complete.
status: draft
---

# Scope and ownership

[HB-246](https://linear.app/hybrid-billing/issue/HB-246) starts work item W-37 (requirement HB-37; HB-37-D/I/V). This increment covers the recovery core:
- recovery generations;
- the external write lock;
- the recovery manifest and verification;
- the reconciliation job;
- re-deletion after restore.

The detail contracts QA-02, QA-06 and QA-09 are already implemented: HB-178, HB-184, HB-190, HB-214, HB-216 and HB-242. Migration import, meaning past documents imported as read-only history without moving secret keys, is increment 2.

Base: main `2988ecb`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the design and the change, because the register decides when external writes such as ERP submission, issuance and delivery may resume after a recovery. Machine: mac_mini.

Allowed files: `modules/recovery/*` and `docs/features/HB-246/*`.

# Observed baseline

- **HB-37-D.** Restore the database, evidence, keys, policies, modules, numbering, submissions and the outbox together. Block the previous issuer. Do not resume external writes before re-reconciling externally confirmed results.
- **HB-37-I.** RecoveryGeneration, an external write lock, a recovery manifest and a reconciliation job. Deletion includes re-deletion on backup and restore.
- **HB-37-V.**
  - Detect and block external re-issue, double delivery and evidence loss in these cases: ERP success after the restore point, a live old worker, migration re-import and data expiry.
  - At scale, use artifact inventories, range checksums and chunk reconciliation.
  - Check the state and numbering ledgers in full.
- **HB-245.** The installer records backup checkpoints, with a digest and a scope, that a restore point can name.
- **Sibling conventions.** The sibling modules provide the conventions for proposals, dual control, actors and bounds.

# Decision

A new package `modules/recovery` follows the sibling conventions:
- one mutex and an injected clock that never goes backward;
- `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (with `Refusal{Reason}`) and `ErrLimit`;
- check order: form, lookup, repeat, rule, room;
- assistants refused as `ai_not_allowed` and system actors as `not_human` where a human is required;
- numbered proposals approved by a second human with a full recheck, withdrawal by the proposer, caps per scope and per proposer, and bounded history.

It imports only the standard library.

## Invariants of the register itself

- **Outside the restore.** The recovery register is outside every restored domain. Its generation, fences, call records and re-apply log are never restored from a backup.
- **Monotonic generation.** The generation never decreases.
- **Persistence.** Persistence (a residual) must keep both properties. Until it does, serving after a recovery is an exposure blocker.
- **Previous issuer.** A previous issuer that runs against its own copy of the register, for example on an old site or in a split brain, is not refused by fencing alone. Unlock therefore also requires an attestation, for each external system, that the credentials the previous issuer held were rotated or revoked (below).

## Installations, writers and external systems

| Proposal kind | Payload | Effect |
| --- | --- | --- |
| `installation` | `Installation{ID, Tenant, Skew}` | Registered installation: generation 1, state `serving`. `Skew` is 0–1 h, the allowed clock difference between backup, register and external systems. By convention the ID equals the installer site ID; linking the two is a residual |
| `writer` | `Writer{Installation, ID, Kind, MaxCall}` | A system actor allowed to perform external writes. `Kind` is `erp`, `issuance`, `delivery`, `payment` or `message`. `MaxCall` is 1 s–1 h, the longest one external call may take. At most 16 writers per installation |
| `external` | `External{Installation, ID, Kind}` | An external system whose confirmations must be reconciled. At most 16 per installation. One registered during a recovery starts pending |
| `verifier` | `Verifier{Installation, ID}` | The one system actor that reports restored inventories, reconciliation pages and re-applications. Replacing it is a proposal, and a recovery in progress keeps its state |
| `begin` | `RestorePoint{Installation, Generation, Backup, ManifestDigest, AsOf, Manifest}` | Opens a recovery. `Generation` must equal the current generation |
| `unlock` | `Unlock{Installation, Generation}` | Ends a lock or a recovery |
| `prune` | `Prune{Installation, Before}` | Drops only re-apply log entries recorded at or before `Before`. The proposers attest that no restorable backup older than that exists; the register cannot check this. Refused while a recovery is open. Fences are never pruned in this increment: they stand for records a restore may have lost, which later backups also lack until increment 2 re-imports them. A consumer acknowledgement hook that would make fences prunable is a residual |
| `abandon` | `Abandon{Installation, Recovery}` | Abandons the restore point of an open recovery that cannot complete, for example a corrupt backup. It increments the generation, records the abandoned attempt in history, and keeps the fences, the log and the original `StoppedAt`. The recovery itself stays open without a restore point, so `unlock` is refused (`no_restore_point`) until a new `begin` sets one and every condition is met |

- **Identities.** Writers and verifiers are system actors. An identity serves one installation, is never reassigned, and is tombstoned when replaced.
- **Removing writers.** `RemoveWriter` by one human takes effect at once.
- **Locking.** `Lock(installation, reason)` by one human takes effect at once, only in state `serving`. During a recovery the installation is already stopped, so `Lock` is refused (`recovering`). The reason is plain text of 1–200 bytes. Locking:
  - sets the state to `locked`;
  - increments the generation, which fences every earlier token;
  - records `StoppedAt`.
- **Pending begins.** At most one `begin` proposal may be pending per installation. Any generation change retires the pending begins of that installation.
- **Recovery IDs.** Each recovery has an ID (`recovery-<n>`). Domains, pages and the pending window are keyed to the recovery ID, not to the generation.

## Permits, calls and fencing

- **Permits.** `Permit(installation, writer, actor)` is called by the writer itself. It requires state `serving` and returns `Token{Generation, Sequence, Expiry}`. The expiry is `MaxCall` after issue, and the sequence is recorded with its writer. Each writer keeps at most 1,000 live sequences, and its own oldest are evicted first. Other writers are never affected.
- **Calls.** `Call(installation, writer, token, Target{Workspace, Key, Series, Number}, actor)`. `Workspace` and `Key` are required, while `Series` and `Number` are optional: both are present or both are absent, for example both absent for a delivery. The call is the single choke point before any external call, called by the writer. It refuses when:
  - the generation is not current (`fenced`);
  - the state is not `serving` (`locked`);
  - the sequence belongs to another writer (`not_writer`);
  - the token has expired (`expired`);
  - the key is delivery-fenced for that workspace (`fenced_key`);
  - the number is at or below the numbering high-water mark of `(workspace, series)` (`below_high_water`).

  Otherwise it records an open call with a deadline of `MaxCall` and returns the call ID and the deadline.
- **Deadline obligation.** Adapters must check the deadline with a monotonic clock just before sending, and must pass it as the request timeout. They should prefer external idempotency keys where the external system supports them. A paused process can still send after its deadline, which is why coverage adds `Skew`. This is an accepted limit.
- **Completion.** `Done(installation, call, outcome, actor)` closes the call. The outcome is `confirmed`, `failed` or `unknown`, and a repeat of the same outcome is idempotent.
- **Call caps.** Each writer may hold at most 1,000 open calls. Calls past their deadline are reclaimed with outcome `unknown`. Closed calls are kept in a ring of 10,000 per installation.
- **Calls during a lock or recovery.** A call that is still open when writes stop is in flight. Reconciliation must cover its deadline (below). Its outcome is reconciled from the external system, never assumed.
- **Consumers.** The consumers that must route through `Call` are `modules/issuance`, `modules/delivery`, `modules/webhooks`, `modules/erpadapters` and `modules/jobs`. Wiring them is a residual, and an exposure blocker for serving after a recovery.
- **Fence keys.** Keys use the consumers' own identifiers. `Series` and `Number` use issuance's series ID and `int64` value.

## Restore points and manifests

- **Domains.** The closed domain set is `database`, `evidence`, `keys`, `policies`, `modules`, `numbering`, `submissions`, `outbox` and `deletions`.
- **Restore point.** A restore point names:
  - the backup digest;
  - `ManifestDigest`, which is computed at backup time and named in the backup evidence;
  - `AsOf`, the quiesce start of the backup checkpoint, which must not be in the future;
  - the manifest.

  The register recomputes the manifest digest, as SHA-256 over its length-prefixed canonical encoding, and requires it to equal `ManifestDigest` (`manifest_mismatch`). Binding the digest to the installer's checkpoint evidence is the adapter's job and is recorded as a residual.
- **Chunks.** Each domain has an `Inventory{Items, Chunks}`. The checksum contract is:
  - items are sorted bytewise by canonical key;
  - each chunk is `{Start, End, Count, Checksum}` with `Start < End` and `Count = End − Start`;
  - chunks partition `[0, Items)` contiguously;
  - the checksum is the SHA-256, in lowercase hex, of the length-prefixed item encodings of that range;
  - an empty domain has `Items = 0` and no chunks, and is allowed for a fresh installation;
  - at most 10,000 chunks per domain.
- **Opening a recovery.** Approving `begin` requires no open recovery with a restore point. It:
  - increments the generation;
  - sets the state to `recovering`;
  - records `StoppedAt`, unless a lock already set it, and `BeganAt`;
  - resets every external system to pending;
  - stores the restore point.

  It also refuses when `AsOf − Skew` is at or before the last pruned boundary (`pruned`). A new `begin` after an `abandon` sets a new restore point on the open recovery, and resets its domains, pages and external systems.

  History keeps only the manifest digest and the statuses, never the chunks.
- **Phases.** A recovery runs `verify`, then `reapply`, then `ready`.
  - **Verify.** `Report(installation, domain, Inventory, actor)` by the verifier is accepted only in the `verify` phase. It compares the report with the manifest chunk by chunk. The domain becomes `verified` when every value is equal; otherwise it is `mismatch`, with the differing chunk indexes (at most 100 returned). Every chunk is checked, without sampling. A later report replaces the earlier one.
  - **Re-apply.** When every domain is verified, the first re-application moves the recovery to `reapply`. Reports are refused from then on (`phase`), because re-application changes the inventories.

## External reconciliation

- **Pages.** `Reconcile(installation, external, Page, actor)` is called by the verifier only, and only during a recovery (`not_recovering`). A `Page` is `{Number, From, Through, Final, Rotated, Items}`:
  - `Number` is sequential from 1 per external system per recovery. Repeating a page number with identical content is idempotent, and with different content is `ErrConflict`. Pages after `Final` are refused (`not_pending`);
  - pages must be contiguous: each page's `From` is at or before the previous page's `Through`;
  - `Through` must not be in the future;
  - `Items` holds up to 10,000 entries `{Workspace, Key, Kind, At, Series, Number}`;
  - `Kind` is `submitted`, `issued`, `delivered` or `paid`;
  - `At` must not be in the future.
- **Applying.** Every item of every page is applied, whatever its time:
  - `issued` and `submitted` items raise the high-water mark of `(workspace, series)`;
  - every kind enters the delivery fence for `(workspace, key)`.

  Fences only grow, and applying is idempotent. A page is applied all or nothing: if a cap would be exceeded, nothing is applied (`ErrLimit`).
- **Reconciled.** An external system becomes `reconciled` with its `Final` page, when the union of its pages for this recovery meets all of these:
  - `From` is at or before `AsOf − Skew`;
  - `Through` is at or after `StoppedAt + max(MaxCall of the writers) + Skew`;
  - `Through` is at or after the latest deadline, plus `Skew`, of any call still open when writes stopped;
  - the final page attests `Rotated`, meaning the previous issuer's credentials for that system were rotated or revoked.

  Otherwise the final page is refused (`coverage`).
- **Fence reads.** `HighWater(installation, workspace, series)` and `Fenced(installation, workspace, key)` read the fences.
- **Fence caps.** 1,000 series and 1,000,000 keys per installation, and 10,000,000 keys across the register. A full fence cap blocks a recovery permanently until the residual consumer-acknowledgement hook exists. That is an accepted exposure blocker.
- **Calls at begin.** At begin, every call that is still open, or was recorded `confirmed` or `unknown` after `AsOf − Skew`, is fenced automatically.
  - Auto-fencing is best effort, because the closed-call ring is bounded. External reconciliation remains the authority.
  - Fenced `unknown` items are listed in `Recovery`. One that the external system shows never happened is handled manually under a new key, because fences never shrink.

## Ordered re-application of protective events

- **Recording.** `Record(installation, Event{Kind, Domain, Key}, actor)`. The register stamps `At` with its own clock. The call may be called by any writer or the verifier, in every state. `Kind` is `delete`, `purge`, `hold`, `extend`, `revoke_key` or `revoke_access`. Recording is write-ahead: callers record before they change any bytes or grants. The callers that must record are:
  - evidence `DeleteArtifact`, `Purge`, legal holds and retention extension;
  - key revocation;
  - access revocation;
  - W-43 retention.

  Wiring them is a residual.
- **Pending entries.** During a recovery, the pending window is the events with `At` in `(AsOf − Skew, BeganAt of the current attempt]`. Only events recorded since that `BeganAt` are excluded, including records made by the current re-application. Events recorded during an abandoned attempt are pending again; re-applying their duplicates is harmless.
- **Unapplicable events.** `Unapplicable(installation, sequence, reason, actor)` by the verifier marks a pending event that can no longer be applied, for example a hold on a bundle that does not exist after the restore.
  - It is refused for `revoke_key` and `revoke_access` (`must_apply`), because revoking something absent is a no-op and must still be reported as re-applied.
  - It is recorded as evidence loss and listed in `Recovery`.
  - The `unlock` proposal must acknowledge it with `Unlock.Lost`, the SHA-256 of the ordered list of unapplicable sequences.
- **Reapplying.** `Reapplied(installation, sequence, actor)` by the verifier must name the next pending entry in recorded order (`out_of_order`). Repeats are idempotent, and an unknown sequence is `ErrNotFound`. Re-applying them in order keeps HB-190's "purge, then delete" sequence and restores holds and revocations.
- **Caps.** 1,000,000 events per installation and 10,000,000 across the register. Pruning is a dual-controlled `prune` proposal. With a full log, `Record` refuses (`ErrLimit`), and the caller must not delete without a record. Until an operator prunes, a full log blocks deletion, and that permanent-block risk is accepted.

## Unlock

An `unlock` proposal for the current generation is dual-controlled. Whether the conditions apply depends on whether a recovery is open, not on the state. With an open recovery, proposing and approving both require:
- the recovery has a restore point;
- every domain is `verified`;
- every external system is `reconciled` for this recovery;
- every pending event is re-applied or marked unapplicable;
- `Lost` equals the digest of the unapplicable list;
- at least one external system is registered for each writer kind that has ever had a registered writer, including removed writers.

Without a recovery (a plain lock), unlock requires only that no call is still open within its deadline. A plain lock already incremented the generation, so tokens issued before it stay fenced.

Applying an unlock:
- sets the state to `serving`;
- closes the recovery into history;
- leaves the fences and the log in place.

`Recovery(installation)` reads the status:
- the phase;
- each domain's status;
- each external system's status;
- the pending events;
- the open calls;
- the generation;
- `StoppedAt` and the restore point.

## Bounds

| Limit | Maximum |
| --- | --- |
| Installations | 100,000 |
| Writers per installation | 16 |
| Live sequences per writer | 1,000 |
| Open calls per installation | 10,000 |
| External systems per installation | 16 |
| Chunks per domain | 10,000 |
| Pending `begin` proposals per installation | 1 |
| Items per page | 10,000 |
| Pages per external system per recovery | 1,000 |
| Pending `begin` proposals, register-wide | 1,000 |
| Series per installation | 1,000 |
| Fenced keys | 1,000,000 per installation, 10,000,000 in total |
| Events | 1,000,000 per installation, 10,000,000 in total |
| Recovery history | 100 per installation (digests only) |
| Pending proposals | 1,000 per scope |

## Residuals and exposure blockers

- Persistence that keeps the register outside restored domains, with a generation that never decreases.
- Wiring `Call`, `Done` and `Record` into the consumers.
- Storage and external adapters that query external systems, compute checksums and bind the manifest digest to installer checkpoint evidence.
- Migration import (increment 2), which uses the fences and the log.
- Lost records. Local records of externally confirmed items that the restore lost are fenced, but the documents themselves must be re-imported by increment 2.
- Number gaps. Whether gaps from `below_high_water` are acceptable for each series is for the issuance owner to decide.
- Pending begin payloads. They are capped globally at 1,000 pending `begin` proposals.
- HB-37-V cases still NOT_RUN at system level: migration re-import, data expiry, and live ERP after the restore point.
- RPO and RTO targets per scale, key recovery procedures, and screens.

# Verification plan

Literal tests:
- **Form and bounds.** Every form rule and bound.
- **Proposals.** Proposal rules.
- **Identities.** Writer, external and verifier identity binding.
- **Fencing.**
  - Permits and calls across lock, begin and unlock.
  - A live old worker holding a token from the previous generation is refused after a restore.
  - Removed writers, wrong writers and expired tokens are refused.
  - Eviction is per writer.
  - Fenced keys and numbers below the high-water mark are refused at `Call`.
  - A call open across a begin needs reconciliation coverage, and `Call` races begin under `-race`.
- **Manifests.**
  - Partition validation.
  - Manifest digest mismatch.
  - Verified, mismatch and repair reports.
  - The phase rule.
  - Empty domains.
  - No sampling.
- **Reconciliation.**
  - Page order.
  - Coverage: `From`, `Through` equal to `AsOf`, `StoppedAt + MaxCall`, and in-flight deadlines.
  - The `Rotated` attestation.
  - All-or-nothing caps.
  - Items before `AsOf`, which are applied anyway.
  - Workspace scoping.
  - Reset on begin.
  - The verifier as the only reporter.
- **Round-2 additions.**
  - Abandon, then a new begin.
  - Re-application that records again.
  - Events recorded during `recovering`.
  - `Lock` refused while `recovering`.
  - Call caps and reclaiming.
  - Auto-fencing at begin.
  - Unapplicable events, with `Lost`.
  - Page rules: gaps, a future `Through`, pages after `Final`, repeats.
  - Retirement of a stale pending begin.
  - Unlock after `abandon`, refused.
  - A begin after `abandon`, with a revocation recorded in between becoming pending.
  - Fences never pruned.
  - A call still open at begin fenced.
  - The pruned boundary with `Skew`.
  - `Unapplicable` refused for revocations.
  - The kind-coverage condition at unlock.
  - `pruned`.
  - Repeats of `Done` and `Report`.
- **Re-application.**
  - Every event kind, in order, with skew.
  - Recording in every state.
  - Pruning.
  - A full log refusing `Record`.
- **Unlock.** Refusal for each missing condition, and success when everything is complete.
- **Isolation.** Isolation between installations.

Property test: random sequences of permits, deletions, begin, reports, reconciliations, re-applications and unlocks across two installations. They run against an independent model of generation, state, fences and pending conditions, with these invariants:
- no check succeeds for a token of a past generation;
- no permit or check succeeds while the state is not `serving`;
- unlock never succeeds with a pending condition, or without a restore point;
- the generation never decreases;
- fences only grow;
- no `Call` succeeds for a fenced key or a number at or below the high-water mark.

Checks:
- `-race -count=10`, a 386 build, gofmt and vet;
- the whole repository with `-race`;
- 100% coverage and mutation testing.

# Review history
- Round 1 (independent reviewer, read-only): FAIL. Findings and what changed:
  - H1. Added `StoppedAt`, `MaxCall`, call tracking with `Call` and `Done`, and coverage rules.
  - H2. Every item is applied, and re-application uses a skew.
  - H3. The register's own invariants and the `Rotated` attestation are stated.
  - H4. `Call` is the single choke point, and its consumers are named.
  - H5. Re-application is an ordered log of protective events.
  - M1. Only the verifier reconciles.
  - M2. Pages, a `Final` page, and a reset on begin.
  - M3. A manifest digest and a checksum contract.
  - M4. Phases.
  - M5. Writer-bound, expiring tokens, and a lock increments the generation.
  - M6. Per-writer eviction, one pending begin, digest-only history, and global caps.
  - M7. All-or-nothing pages, pruning, and the accepted permanent-block risk.
  - M8. Recording in every state, write-ahead, with named sources.
  - M9. Workspace-scoped fences.
  - L1. Begin is bound to the current generation.
  - L2. Edge cases defined.
  - L3. NOT_RUN cases listed.
  - L4. The installation-to-site convention is stated.
- Round 2 (same reviewer): FAIL. Findings and what changed:
  - N1. Added `abandon`.
  - N2. The pending window is bounded by `BeganAt`.
  - N3. `Lock` is refused while recovering, state is keyed to the recovery ID, and unlock conditions depend on an open recovery.
  - N4. A deadline obligation, and `Skew` added to coverage.
  - N5. Added `Unapplicable` with `Lost`.
  - N6. Page rules defined.
  - N7. Per-writer call caps, reclaiming and a closed-call ring, plus auto-fencing at begin.
  - N8. Fence pruning, with the accepted risk.
  - N9. The register stamps `At`.
  - N10. Pending begins are retired on a generation change.
  - N11. Added `pruned` and the attestation note.
  - N12–N15. Recorded.
- Round 3 (same reviewer): FAIL. Findings and what changed:
  - R1. After `abandon` the recovery stays open, with `no_restore_point`.
  - R2. `prune` removes only log entries, never fences.
  - R3. Open calls are fenced at begin, best effort, and `unknown` items are listed.
  - R4. The pruned boundary includes `Skew`.
  - R5. `must_apply` for revocations, and `Lost` is a digest.
  - R6. Wording.
  - R7. The kind coverage moved to unlock.
  - R8. `Skew` added to the in-flight bound.
  - R9. `StoppedAt` is kept.
  - R10. The bound is in the table.
- Round 4 (same reviewer): PASS. Text fixes made before implementation:
  - T1. `begin` needs no open recovery with a restore point, and the pending window is per attempt.
  - T2. The fence cap is an exposure blocker.
  - T3. Tests and an invariant added.

# Change review resolution

The change review (independent reviewer, read-only) returned FAIL with M1–M4 and lows. Resolved:
- M1. Closed calls now record their close time. Auto-fencing at begin covers calls that started or were closed (confirmed or unknown) after `AsOf − Skew`.
- M2. Stopping writes (lock or begin from `serving`) snapshots the writer kinds and the longest `MaxCall`. Coverage uses the larger of the snapshot and current writers. Kind coverage at unlock includes the snapshot kinds, so removing a writer during a recovery cannot weaken either check.
- M3. Added a concurrent test: `Permit` and `Call` race `begin` under `-race`.
- M4. The property model now covers:
  - auto-fencing at begin;
  - final pages that fail coverage;
  - `Unapplicable`, including `must_apply` for `revoke_key`, with a modelled `Lost` digest.

  Calls are still closed immediately in the property test. Open-call coverage is tested by literal tests.
- L1. Auto-fencing checks the key and series room it actually needs. Skipped candidates are counted in `Recovery.Skipped`.
- L2. `Abandon` carries the generation it was proposed under.
- L3. The phase stays `verify` until the first re-application, matching the `Report` gate. With nothing to re-apply it is `ready`, and reports are still accepted until a re-application starts.
- L4. `Unknown` entries are kept across attempts.
- L5. `Done` after a reclaim keeps `unknown` and returns `ErrConflict`. This is accepted: the item stays fenced and listed for manual handling.
- L6. `Recovery` adds `OpenCalls`, `ManifestDigest` and `PointGeneration`.
- L7. Added a test for the exact coverage boundary.
- L8. `begin` checks the future `AsOf` before the generation, and `Reconcile` checks contiguity before the page cap.

The re-review (same reviewer, read-only) returned FAIL with R1–R3. Resolved:
- R1. Kind coverage at unlock now counts every writer ever registered, including writers removed before writes stopped. The stop snapshot is kept only for the longest `MaxCall`.
- R2. The race test now asserts that begin was applied and that every call that succeeded is fenced, which shows no call slipped past begin.
- R3. `Unknown` is deduplicated by call ID across attempts. `Skipped` keeps adding up across attempts.
