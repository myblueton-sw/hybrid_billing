---
type: ImplementationPlan
title: Installer upgrades with owned resources, upgrade previews, quiesced backup checkpoints and dual-controlled software rollback
description: Increment 2b of work item W-36. When a run completes, the register records the resources it created as owned by the installation. Releases declare the versions they upgrade from. An upgrade preview distinguishes replaced, retained, created, detached, removed and adopted resources, keeps every data volume under its logical name, and blocks foreign, missing, remapped or moved resources. Each upgrade run stops writers and records a backup checkpoint for W-37 recovery. When an upgrade fails before the new version touches data, a dual-controlled, previewed software rollback returns the site to its previous release. Data is never written back here, because data restore belongs to W-37.
status: draft
---

# Scope and ownership

[HB-245](https://linear.app/hybrid-billing/issue/HB-245) completes the HB-36-I upgrade and restore procedures at the level of the installer register, extending `modules/installer` from HB-243.

Business-data recovery belongs to W-37 (HB-37: recovery generations, external write locks, reconciliation). It is out of scope here. This plan never writes data back. Its rollback only reinstalls the previous release's software, and only before any new-release service has started.

Base: main `e07771c`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the design and the change, because the register decides what an upgrade may change on a customer host and when a rollback is safe. Machine: mac_mini.

Allowed files: `modules/installer/*` and `docs/features/HB-245/*`.

# Observed baseline

The HB-243 register covers first installation only:
- every existing resource is a conflict;
- a site with a completed run is refused as `already_installed`.

Upgrade was deferred because observations carry no ownership labels, and labels on a host can be forged. The register's own run records are the trustworthy source of ownership: it knows which `volume:` and `service:` steps, with which resolved names and ports, a completed run executed.

# Decision

## Boundary with W-37

This increment never writes data back. Restoring data volumes is business-data recovery, which needs W-37 first: recovery generations, an external write lock, and reconciliation before writes resume.

What this increment provides:
- **Checkpoint.** Every upgrade stops writers and takes a backup checkpoint. The run records the backup's digest, its scope (the owned data volumes) and the probe that took it, as evidence W-37 will consume.
- **Software rollback.** A rollback after a failed upgrade reinstalls the previous release's software.
- **Data touched.** The site-level flag `data_touched` is set as soon as any upgrade run reaches its first `service:` step, meaning the step before it completed. From then on a target service may have started and changed data directly or over the network, whether or not it mounts a volume. The flag stays set until an upgrade completes. Rollback is refused while it is set (`data_touched`), and only a completing upgrade or W-37 recovery may proceed.
- **No service overlap.** Because rollback is possible only before any target service started, no new-release service is ever left running next to the previous release.

## Owned resources

When an install, upgrade or rollback run completes, the register records `Installation{Version, Services, Volumes, Ports}`:
- **Services.** Each service has a name and its resolved ports.
- **Volumes.** Each host volume has its logical release volume name and a data flag, and may also be detached.
- **Ports.** The resolved `number/protocol` keys of the current services.
- **Source.** The record is derived only from the register's own run steps and resolved layout, never from observations.
- **Bounds.** At most 32 detached volumes may accumulate, counting only detached volumes present in the current observation. A preview that would exceed this is `ErrLimit`. When an upgrade completes, detached volumes absent from that run's latest observation are dropped from the record, so an operator can free the cap by removing them outside the product.

Ownership is matched by the register record and the resource name. A resource removed out of band and recreated under the same name cannot be told apart, because observations carry no unforgeable identity. This is an accepted limit. Every preview states the ownership basis as `register_record_and_name`, and the limit is a residual.

- **Site states.** A site is `installed`, or `upgrade_failed`.
  - **Entering.** The site becomes `upgrade_failed` when an upgrade run ends `failed`, `cancelled` or `invalidated` after completing at least one change step. Change steps are `quiesce`, `volume:`, `remove:` and `service:`.
  - **Leaving.** The state lasts until an upgrade or rollback completes, however later runs end.
  - **Accumulated resources.** Adoption and leftovers cover the union of resources created, and services removed, by every failed run since the site was last `installed`. The union is stored on the site, together with each created volume's logical name, the backup records and the latest touching version, so purging run history never loses them. `volume_remapped` also applies to adoption.
- **Reads.** `Installed(site)` returns the record and the state. Upgrade history `{From, To, Run, Backup, Kind}` is capped at 100 entries per site.

## Releases

- **Upgrade sources.** `Release` gains `UpgradeFrom`: 0–16 versions, strictly ascending, each lower than `Version`.
- **Encoding.** The signed encoding tag becomes `hybrid-billing/release/v2` and includes the new list. The known-answer tests are re-pinned and cross-checked independently.

## Upgrade preview

`PreviewUpgrade(site, version)` requires:
- the site is installed (`not_installed`), or in state `upgrade_failed`;
- no run is active (`run_open`);
- the release is published and not revoked;
- the platform is supported;
- a fresh observation;
- the target is higher than the installed version (`direction`) and lists it in `UpgradeFrom` (`not_upgradable`);
- while `data_touched` is set, the site records the latest touching version. That is the target of the most recent run that reached its first `service:` step, updated on every such run. The target must equal it, or list it in `UpgradeFrom` (`data_form`). The installed version must also be listed, as above. Data in the newest form is therefore never handed to an older release.

Target resources resolve through the site's overrides for the target version, with `config_mismatch` as in HB-243.

| Section | Action or conflict |
| --- | --- |
| Services | `replace`, when owned. Otherwise `create`, or conflict `service_exists`. An owned service absent from the target is `remove`: it is stopped and unregistered, and its volumes are kept |
| Ports | `reuse`, when owned by the same service. Conflict `port_moved`, when owned by another service. Otherwise `bind`, or `port_in_use`. Duplicates give `port_duplicate` |
| Volumes | `retain`, when owned under the same logical name. Conflict `volume_remapped`, when owned or detached under a different logical name. `reattached`, when previously detached under the same logical name. Otherwise `create`, or `existing_volume`. An owned non-data volume absent from the target is `detach`. Duplicates give `volume_duplicate` |
| Data | Every owned data volume must map to a target data volume with the same host and logical name, with effect `retained`. Otherwise conflict `data_volume_removed`. New target data volumes have effect `new`. Reattached volumes have effect `stale_content` |
| Drift | An owned service or volume missing from the observation gives conflict `owned_missing`, except a service that a failed run's completed `remove:` step removed |
| Adoption | In state `upgrade_failed`, resources that failed runs' completed steps created (the stored union) are listed as `adopt_created`, taken from the run record, instead of conflicts |
| Permissions | Derived permissions of the target not yet granted |

The hash covers:
- the kind tag `upgrade`;
- the installed record and the site state;
- the adopted resources;
- every input and section, as in HB-243.

A preview changes nothing.

## Upgrade runs

- **Acceptance.** An `upgrade` proposal `{Site, Version, Hash}` is dual-controlled. Proposal and approval recompute the upgrade preview and require an equal hash and no conflicts, as `checkPlan` does in HB-243.
- **Encoding.** The proposal encoding includes the new kinds and the `Run` field, so the dedup cannot merge different proposals.
- **Starting.** `StartRun` recomputes the preview for the acceptance's kind and requires an equal hash and no conflicts, for every kind. The acceptance is then consumed.
- **Steps.**
  1. `preflight`;
  2. `quiesce`, whose evidence states that writers are stopped;
  3. `backup`;
  4. one `artifact:` step per target artifact;
  5. one `volume:` step per created volume;
  6. one `remove:` step per removed service;
  7. one `service:` step per target service;
  8. `verify`.
- **Backup.** The `backup` evidence digest, the backup scope and the probe are recorded on the run and shown in its view.
- **Data touched.** The site's `data_touched` flag is set when the step before the first `service:` step completes.
- **Completion.** Completing `verify` replaces the installation record with the target layout (the port set is replaced), keeps retained, detached and reattached volumes owned, clears `upgrade_failed`, and appends history.
- **Resume.** Resuming counts as satisfied:
  - every owned resource;
  - every resource created by a completed step;
  - every service removed by a completed `remove:` step, for `owned_missing`;
  - every service whose `service:` step completed.

  Resuming otherwise follows HB-243, including the requirement of an observation newer than the interruption.
- **Invalidation.** Release revocation, probe changes and configuration changes invalidate active runs, as in HB-243.
- **Retirement.** Retirement now covers every acceptance kind (`plan`, `upgrade`, `rollback`).

## Software rollback

- **Preview.** `PreviewRollback(site)` requires:
  - the state is `upgrade_failed`. A site that is merely `installed` gets `not_needed`, which also covers a failed upgrade that never completed a change step: the previous release was never replaced, so it is restarted outside the product;
  - the site's `data_touched` flag is clear (`data_touched`);
  - no backup is required, because software rollback writes no data. The run view lists every recorded backup;
  - no run is active;
  - the installed version's release is not revoked (`revoked`);
  - a fresh observation;
  - the run has not already been rolled back (`rolled_back`).
- **Layout.** The rollback layout is the installation record itself: names and ports come from the record, never from current overrides.
- **Preview sections.**
  - `restart` for each recorded service, including services that failed runs removed;
  - `reuse` for its ports;
  - `retain` for its volumes;
  - the leftovers, meaning resources the failed runs created (the stored union) that the record does not use, as information;
  - recorded ports in use are `reuse`, because the previous release's own services may still listen; this falls under the accepted name-based ownership limit;
  - conflicts: `owned_missing`, except for services that failed runs removed.
- **Hash.** The hash covers:
  - the kind tag `rollback`;
  - the record;
  - every failed run in the union, and every recorded backup digest with its scope (an absent backup is encoded as an empty, length-prefixed value);
  - the latest touching version and the `data_touched` flag;
  - the leftovers, the observation content, the revision and the probe.
- **Acceptance.** A `rollback` proposal `{Site, Run, Hash}` is dual-controlled. Approval and `StartRun` recompute the preview and require an equal hash.
- **Consumption.** At most one rollback acceptance per failed run is consumed at a time. A second approved acceptance is refused at `StartRun` while a rollback of that run is active or completed (`rolled_back`). If a rollback run fails or is cancelled, a new acceptance may retry it, because software rollback writes no data.
- **Steps.**
  1. `preflight`;
  2. one `artifact:` step per artifact of the installed release, with pinned digests;
  3. one `service:` step per recorded service;
  4. `verify`.
- **Completion.** Completing the run marks the failed runs as rolled back, returns the site to `installed` and appends history. The data is unchanged. Leftover volumes and artifacts stay in place. No leftover services exist, because no target service ever started.
- **Run fields.** A rollback run records the installed version, the current revision and the current probe. If that version is revoked, the sweep invalidates the run.

## Other rules

- **Bootstrap codes.** `IssueBootstrap` is refused (`not_allowed_state`) on upgrade and rollback runs, after an explicit check of the run kind.
- **Install runs.** Install runs, observations and diagnostics are unchanged.

## Residuals

- **W-37.** Data restore from the recorded backups, the external write lock and reconciliation. W-37 is also the way out for a detached data volume: such a volume blocks every upgrade whose target does not declare it, even after it has been removed from the host. The backup scope may name detached data volumes that are not observed. It is evidence only.
- **Host backup.** Real host backup and quiesce tooling, and checking the backup content.
- **Ownership.** Matching by name, as above.
- **Leftovers.** Removing detached volumes and leftovers, which the operator decides outside the product.
- **Not covered.** Downgrades, persistence and screens.

# Verification plan

Literal tests:
- **Recording and preview.**
  - Installation records after install, upgrade and rollback.
  - Every preview action and conflict: `replace`, `create`, `remove`, `reuse`, `bind`, `retain`, `detach`, `data_volume_removed`, `owned_missing`, foreign conflicts and overrides.
  - `not_installed`, `not_upgradable` and `direction`.
  - The hash inputs.
- **Upgrade runs.**
  - Acceptance recheck.
  - Step order, with `backup` before change steps.
  - Completion updating the record and history.
  - Resume with owned and created resources.
  - Invalidation.
- **Rollback.**
  - Every rollback rule, including `data_touched` and `not_needed` (no change step completed).
  - Recheck at `StartRun` after a configuration change, a probe change, revocation and a new conflicting observation.
  - Two approved rollback acceptances.
  - A retry after a failed rollback run.
  - Steps taken from the record after overrides changed.
  - An artifact digest mismatch on rollback.
  - Leftovers reported.
- **Further round-2 tests.**
  - An interruption during the first `service:` step, after which rollback is refused.
  - Two failed upgrades in a row.
  - Both previews after a `remove:` step.
  - Re-upgrade targets after data was touched (`data_form`).
  - `quiesce` coming before `backup`.
  - `stale_content`.
  - The ownership basis shown in previews.
  - Dropping detached volumes on completion, and freeing the cap after an out-of-band removal.
  - The sequence U1 to B, then U2 to C, then a target of B refused with `data_form`.
  - Rollback after a failure between `quiesce` and `backup`.
  - Adoption that triggers `volume_remapped`.
  - Accumulated records surviving the purge after 100 runs.
  - Rollback while recorded ports are still in use.
- **Failed upgrades and resources.**
  - A re-upgrade after a failed upgrade, with `adopt_created`, including when the installed release is revoked.
  - Resume after `remove:` and `service:` steps.
  - `volume_remapped`, `port_moved`, `reattached`.
  - The lookalike-ownership limit.
  - Retirement of every acceptance kind.
  - The detached-volume bound.
  - Bootstrap refusal on upgrade and rollback runs.
- **Encoding.** Updated release known-answer tests.

Property test: the HB-243 model extended with upgrade and rollback operations across two sites and three releases. It checks the installation record, the run states and these invariants:
- the installation version changes only on a completed upgrade;
- an owned data volume never leaves the owned set;
- no rollback completes twice for one failed run;
- no rollback starts while any failed run since the last completed transition has reached a `service:` step.

Checks:
- `-race -count=10`, a 386 build, gofmt and vet;
- the whole repository with `-race`;
- 100% coverage and mutation testing.

# Review history
- Round 1 (independent reviewer, read-only): FAIL. Findings and what changed:
  - H1. Data restore removed; it belongs to W-37. A `quiesce` step was added, the backup scope is recorded, software rollback is allowed only before `data_touched`, and a rollback is `not_needed` without a change step.
  - H2. Rollback now has a preview and hash. `StartRun` recomputes the preview for every kind, retirement covers every kind, rollback has artifact steps, and its layout comes from the record.
  - M1. Resume now treats removed and replaced services as satisfied.
  - M2. Added the `upgrade_failed` state with `adopt_created`, so a re-upgrade works even when the installed release is revoked.
  - M3. Rollback acceptances are single-use, with a recheck at start and defined retries.
  - M4. Name-based ownership is an accepted limit, named in previews and recorded as a residual.
  - M5. Logical volume names are recorded, with `volume_remapped`.
  - M6. The backup scope and probe are recorded.
  - L1. Detached volumes are capped, and the port set is replaced on completion.
  - L2. Added `reattached` with `stale_content`.
  - L3. Release tag bumped to v2.
  - L4. Added `port_moved`.
  - L5. Explicit run-kind check for bootstrap codes.
  - L6. Rollback run fields and states defined.
- Round 2 (same reviewer): FAIL. Findings and what changed:
  - N1. `data_touched` is now set when a run reaches its first `service:` step, for any service.
  - N2. `data_touched` is now a site-level flag; `upgrade_failed` persists, and adoption and leftovers cover the union of failed runs.
  - N3. Services removed by failed runs are excluded from `owned_missing` and restarted by rollback.
  - N4. Added `data_form`.
  - N5. No leftover services can exist, because rollback requires that no target service ever started.
  - N6. Wording fixed.
  - N7. Detached volumes that are no longer observed are dropped on completion.
  - N8. `quiesce` is a change step.
  - N9. Remapping on reattach is defined.
  - N10. The encoding includes the new fields.
  - N11. Tests added.
  - The H2 remainder: proposal and approval now recompute the preview.
- Round 3 (same reviewer): FAIL. Medium items P1 and P2, plus lows. What changed:
  - P1. The latest touching version is tracked.
  - P2. The cap counts only observed detached volumes.
  - L-a. The `owned_missing` exemption is now stated in the rollback preview.
  - L-b. Rollback no longer needs a backup.
  - L-c. Accumulated sets are stored at site level.
  - L-d. Adoption keeps logical names.
  - L-e. Recorded ports are `reuse`.
- Round 4 (same reviewer): PASS. Low items folded in:
  - the union wording;
  - the rollback hash, which now encodes an absent backup, covers every failed run and includes the touch state;
  - the additional tests.

# Change review resolution

The change review (independent reviewer, read-only) returned FAIL with M1, plus lows and weak tests. Resolved:
- M1 and L1. A detached or adopted volume missing from the observation is now `create`, with effect `new` for data, so a `volume:` step creates it again. It is never assumed present.
- L2. Data volumes created by failed runs are in the backup scope. When a later upgrade completes without targeting them, they are kept in the record as detached data volumes, so W-37 keeps a record of possibly written data. Such a volume blocks later upgrades that do not target it (`data_volume_removed`), which is the safe outcome.
- L3. `Installed` lists every recorded backup digest. The rollback run view does not repeat them.
- L4. Rollback history leaves `From` empty and lists the rolled-back runs in `Runs`.
- L5. Removed the unused `mounts` and the redundant `ended` guard. Rolled-back marks are capped at 100.
- L6. Rollback run ports come from the record.
- Weak tests:
  - W1. A known-answer test for a non-empty `UpgradeFrom` (cross-checked in Python), and a sensitivity test.
  - W2. A sensitivity table for the transition hash.
  - W3. A rollback dedup test.
  - W4. An exact conflict assertion.
  - W5. Rollback rechecks after a probe change, a revocation and a host change.
  - W6. The property test stays on one site, with the observation derived from register state. It does not explore forged or missing resources, which the literal tests cover. This is recorded as a residual.
- Mutation survivors from the first run were closed:
  - edge-bound tests for history, backups and failed runs;
  - an order test for history;
  - a `sortConflicts` unit test;
  - an exact-cap test;
  - permission exclusion;
  - acceptance-form tests;
  - install cancellation never marking a failed upgrade.

  The redundant `s.observation == nil` branch was removed.

The re-review (same reviewer, read-only) returned PASS. Its notes were resolved:
- a rollback proposal on an installed site is now asserted;
- the run steps for a missing reattached volume are asserted;
- the residual for detached data volumes is recorded.
