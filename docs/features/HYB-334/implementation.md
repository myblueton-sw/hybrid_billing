---
type: ImplementationRecord
title: M04 credential rotation, implementation record
description: What each implementation step of the credential rotation command changed, how it was checked and reviewed, and what remains.
status: draft
---

# Ownership

HYB-334, machine:mac_mini. The owner decided on 2026-11-24 that Claude implements this ticket, as an exception to the split of 2026-11-19 in which new implementation goes to Codex. The design is [the plan](plan.md), approved by its security design review; its five owner questions proceed with the defaults stated there. The work is done in three steps, each reviewed and merged on its own: the identity module, the server command and reads, the console.

# Step 1, identity module (2026-11-24)

Nothing calls the new operation yet; no command, read or screen changes in this step.

- `ReplaceCredential(actor, workspace, oldID, newID, verifier, validity, oldEnd)` in `modules/identity/workspace.go` adds the new credential and bounds the old one in one write. Every refusal returns before any change. It needs the same standing as `RevokeCredential`. It refuses an old credential that is unknown in the workspace, revoked, ended or not yet started, an account that is not active, an old credential that was already replaced once, an identifier that exists, the account's limit of live credentials counted with the old one, a new validity that is invalid, already over or starts later than now, and an old end that is not after the module's clock.
- The old end becomes the earlier of its present end and the given end; it is never extended. A credential that was replaced stays replaced, also after its replacement was pruned.
- A credential records the one it replaces and the one that replaced it. `CredentialInfo` returns each link only when the counterpart exists, belongs to the same account and points back, so an identifier reused after pruning is never shown as the other half of a pair.
- Pruning keeps an ended credential while its replacement is live, so the pair stays readable; the ended credentials kept stay within the existing bound plus the live credentials of the account.
- The recheck of a service answer (`modules/identity/boundary.go`, `Recheck`) additionally refuses when the credential's validity does not hold at the time of the recheck, so an answer obtained before a rotation is not accepted after the old end. Human answers are unchanged. No production path obtains a service answer today.

Difference from the plan, stricter: the module refuses a new credential that starts later than now, where the plan left "valid from now" to the caller, and an old credential that has not started yet.

## Checks

`go vet` clean; the tests of `modules/identity` pass, with ten new tests in `replace_test.go`, among them a randomized property that no sequence of add, revoke and replace extends a credential's end, which requires at least 100 successful replacements over its 40 seeds; the tests of `modules/commands` and `modules/assistant`, which use the module, pass; the whole `apps/api/server` test package passed on the change before the last renaming and test-only corrections, and its service and credential tests after them.

Fourteen hand-made changes that each remove one rule (the recheck of the current validity, the pruning guard, the second replacement, the extension of the old end, the suspended and the retired account, an old end in the past, the limit counted with the old credential, each direction of a dangling link, each direction of a mutual link, a later start, the workspace of the credential) each made a test fail; two of them did so only after a test was added for them.

## Review

Independent security change review (independent-reviewer role, AI, by reading, nothing executed): approve with conditions, nothing blocking. Conditions and test findings, all adopted: links must be mutual, because an identifier reused after pruning could otherwise be shown as the other half of a pair and keep an ended credential from pruning; a new credential that starts later must be refused or the rule tested in step 2; a test of a retired account passed for another reason; no check that nothing changed after the state refusals; the property test did not show that any replacement succeeded; no test of an administrator of another workspace naming this workspace's credential. Bounded re-review of the corrections by the same reviewer: approve; its two small remarks (a helper's name, the scope of one unchanged check) were adopted and not reviewed again.

# Remaining

- Step 2, server: the command `service.rotate_credential` routed by a credential target, the overlap default and maximum as installation configuration, the pickup of the new secret, the read fields `replaces`, `replaced_by`, `rotation` and the limits, with tests and an independent security change review.
- Step 3, console: the M04 action, the form and review step, the rows of both credentials, copy in English, Korean and Arabic with terminology review, unit tests and rendered checks.
- The owner's screen review. The residuals of the plan are unchanged.
