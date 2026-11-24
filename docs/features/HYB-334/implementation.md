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
- Pruning keeps an ended credential while its replacement is live, so the pair stays readable. Step 2 widened this: an ended replacement is also kept while the credential it replaced is live. A live credential can therefore keep two ended ones, and the ended credentials kept stay within the existing bound plus twice the live credentials of the account, which corrects the bound stated in the plan.
- The recheck of a service answer (`modules/identity/boundary.go`, `Recheck`) additionally refuses when the credential's validity does not hold at the time of the recheck, so an answer obtained before a rotation is not accepted after the old end. Human answers are unchanged. No production path obtains a service answer today.

Difference from the plan, stricter: the module refuses a new credential that starts later than now, where the plan left "valid from now" to the caller, and an old credential that has not started yet.

## Checks

`go vet` clean; the tests of `modules/identity` pass, with ten new tests in `replace_test.go`, among them a randomized property that no sequence of add, revoke and replace extends a credential's end, which requires at least 100 successful replacements over its 40 seeds; the tests of `modules/commands` and `modules/assistant`, which use the module, pass; the whole `apps/api/server` test package passed on the change before the last renaming and test-only corrections, and its service and credential tests after them.

Fourteen hand-made changes that each remove one rule (the recheck of the current validity, the pruning guard, the second replacement, the extension of the old end, the suspended and the retired account, an old end in the past, the limit counted with the old credential, each direction of a dangling link, each direction of a mutual link, a later start, the workspace of the credential) each made a test fail; two of them did so only after a test was added for them.

## Review

Independent security change review (independent-reviewer role, AI, by reading, nothing executed): approve with conditions, nothing blocking. Conditions and test findings, all adopted: links must be mutual, because an identifier reused after pruning could otherwise be shown as the other half of a pair and keep an ended credential from pruning; a new credential that starts later must be refused or the rule tested in step 2; a test of a retired account passed for another reason; no check that nothing changed after the state refusals; the property test did not show that any replacement succeeded; no test of an administrator of another workspace naming this workspace's credential. Bounded re-review of the corrections by the same reviewer: approve; its two small remarks (a helper's name, the scope of one unchanged check) were adopted and not reviewed again.

# Step 2, server command and reads (2026-11-24)

- Command `service.rotate_credential`, routed by a credential target like revoke, with the same standing as issue and revoke. Fields: `days` (required, bounds as for issue) and `overlap_minutes` (optional). Result fields: `credential`, `secret_ref`, `replaced`, `replaced_until`. The secret is handled as for issue: the module receives the verifier only, and the secret is collected once through the pickup bound to the issuing session.
- Refusals, each leaving everything unchanged: no standing or an unknown credential (permission_denied), a suspended account or the limit of live credentials counted with the old one (prerequisite_failed), `days` or `overlap_minutes` out of bounds (invalid_request with the field), an old credential that is not live or was already replaced (invalid_stage), a stale revision (revision_conflict), an unavailable dependency.
- Overlap as installation settings that can be changed at start: `credential_overlap_minutes` (default 1440) and `credential_overlap_max_minutes` (default 10080), each from 10 to 43200, the maximum not below the default. At run time the lower bound is the pickup validity (10 minutes), the maximum is held to the credential validity and the default is held between the two.
- `GET /api/v1/identity/services`: a credential row carries `replaces` and `replaced_by` while the counterpart is kept, and `rotation`, which is `ending` on an active credential that was replaced and `replacing` on an active credential whose replaced credential is still active. The limits carry the two overlap values and `secret_pickup_minutes`. The OpenAPI description of the read states the fields. The integrations read is unchanged.
- During a trust hold the command is refused, like issue; revoke stays allowed.

## Checks of step 2

`go build`, `go vet`; the whole `apps/api/server` test package and the command-line tests pass, with six new tests in `services_rotate_test.go`; the tests of `modules/identity`, `modules/commands` and `modules/assistant` pass. Ten hand-made removals of server rules (the suspended account, the limit, the reported end held to the old end, the lower bound, the maximum held to the credential validity, each rotation value shown too long, the pickup's account, the maximum, the unit) and the removal of the widened pruning rule each made a test fail.

## Review of step 2

Independent security change review (independent-reviewer role, AI, by reading, nothing executed): approve with conditions, no security defect. Conditions, both adopted: the OpenAPI description of the read lacked the new fields; and a bounded credential lost its link and its `ending` value once its revoked replacement was pruned, so a later rotation would be refused without the screen knowing why, corrected in the module by keeping an ended replacement while the replaced credential is live. Test remarks adopted: a retired-account case that passed for another reason, and a default held below its configured value. Bounded re-review by the same reviewer: approve; its correction of the pruning bound and its remark on the precedence of `ending` are included above and in the OpenAPI text.

Not covered by a test, as the reviewer listed: a refusal of the command under an actual trust hold beyond the table of commands, authentication of both credentials while the account is suspended, and the branch for a credential whose account is not listed.

# Remaining

- Step 3, console: the M04 action, the form and review step, the rows of both credentials, copy in English, Korean and Arabic with terminology review, unit tests and rendered checks.
- The owner's screen review. The residuals of the plan are unchanged.
