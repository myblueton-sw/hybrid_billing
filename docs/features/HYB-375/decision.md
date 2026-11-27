---
type: DecisionRecord
title: HYB-375 — a failed restore of earlier trust changes stops the service start
description: What the trust ledger replay does when the trust changes of earlier runs cannot be restored, and why.
status: draft
---

# Scope and ownership

[HYB-375](https://linear.app/myblueton/issue/HYB-375), machine:mac_mini, root (Claude). Written 2026-11-27 from main `984ae039`. Found by the independent change review of HYB-304 M2a.

# Facts

- At start, `replayTrust` (`apps/api/server/trustledger.go`) rebuilds the activated trust changes of earlier runs from the decision ledger and restores them into the new run's registry (`trustchange.Registry.Restore`, `modules/trustchange/registry.go`). It discarded the error of `Restore` and of `trustchange.New`.
- `Restore` returned on the first invalid view, keeping the views before it and skipping `prune`, so a partly failed restore left a registry that was neither the earlier state nor empty, and the server did not learn of it.
- A view is invalid when its identifier is malformed, belongs to the current run, repeats an identifier already kept or listed, or is not activated. The views come from the ledger's own activation records, so an invalid view means the ledger's history and the registry disagree.
- Every start step returns an error that stops the start with the step's name (`seed.go`, step `trust-replay`); `trustchange.Draw` in the same function already did.

# Decision

1. `Restore` is all or nothing: it checks every view first (including repeats within the list), then adds them all and prunes; on an invalid view it changes nothing and returns `ErrInvalid`.
2. The server returns the error, so the start stops at step `trust-replay` with "restore the trust changes of earlier runs: …". The error of `trustchange.New` is returned as well.

Stopping is chosen over starting with a partial or empty history because the trust change list is the record administrators use to see which policy changes were activated and by whom; starting while silently missing part of it would misstate that record. The ledger is append-only and verified, so nothing can be corrected in place. The error names the start step `trust-replay`; the recovery is the existing one for a decision ledger that cannot be used: start with a successor ledger and either carry the current trust record into it (`--accept-new-decision-ledger` with `--carry-trust-from`) or accept a trust reset (`--accept-trust-reset`), as designed in [HYB-297](../HYB-297/plan.md) and [HYB-298](../HYB-298/plan.md). Today no legitimate ledger produces a refused view (independent review, by reading: identifiers come from verified activation records of the boot workspace, every earlier run is excluded from the drawn run, and one change is activated at most once), so this path is reached only by corruption that passed verification or a future second writer.

# Verification

- `modules/trustchange`: `TestRestoreRefusesTheWholeListWhenOneViewIsInvalid` (a valid view followed by an invalid one, and an identifier repeated within the list): `ErrInvalid`, nothing kept, version unchanged. Existing restore tests pass.
- `apps/api/server`: trust, ledger, replay and seed tests pass; the whole API suite is run before merge.

# Not verified

- A start against a real ledger that holds an invalid earlier change: no such ledger is known; the case is reached only through the module test.
