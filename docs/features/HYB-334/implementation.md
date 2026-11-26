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

# Step 3, console (2026-11-25)

- On the M04 Credentials view a live credential of an active account whose end no rotation has fixed offers "Rotate" beside "Revoke". The action is hidden while an attempt is held for the account.
- The rotation uses the issue form in a rotation mode: validity of the new credential, the time the current credential stays valid (prefilled with the installation default, with its bounds), and the reason. The first submit validates and shows a review; only the second sends. The review names the credential that will end and its end in local time with the same instant in UTC beneath it, and states that the new secret is shown once and must be collected within the pickup time, that the rotation cannot be undone, that the screen does not know where the credential is used and does not check work in progress, and that a suspected exposure calls for revoking.
- The end shown before confirming is computed in the browser from the last read, while the service counts from the command, so it is labelled "Estimated end" with the note that the exact time is set at confirmation. When the credential already ends sooner the end is exact and labelled "Ends at".
- After sending, the existing held-attempt handling applies: an unknown outcome keeps the attempt, and "Try again" resends the same request with the attempt's own credential and values; a success leads to the one-time reveal. The held attempt remembers that it is a rotation, so the texts of the attempts region and the announcements say so.
- Rows: the replaced credential shows "Rotated · ends" with its end and "Replaced by", the new one "Replaces".
- A refused rotation (`invalid_stage`) says that the credential can no longer be rotated. The suspended and limit states use the issue form's texts and offer "Cancel rotation".
- The strict parser of the services read accepts the new fields and rejects an empty link, a rotation value on a credential that is not active, `ending` without `replaced_by` and `replacing` without `replaces`.
- The column widths of the M04 credential table had never applied, because another page's stylesheet set the same selector later; they are now more specific.

## Checks of step 3

`tsc` clean; console tests 1370, pass 1367, fail 0, skipped 3, with new tests of the overlap bounds, the end computation, the UTC text, the rotate condition, the parser and the held attempt's memory of a rotation.

Rendered check `apps/console/tools/credential-rotation-qa.mjs` (removed when HYB-304 rebuilt M04; replaced by `tools/m04-rotation-qa.mjs`, see `docs/features/HYB-304/main-integration.md`), against the API and console of this change (dev server, loopback, sample seed, a data directory the service created, synthetic data only, stopped afterwards): exit code 0, 106 checks passed, none failed, no page error; one run of the final script. It creates a service account, issues a credential and rotates it through the screen: the form with the installation default; an overlap below the pickup time refused before sending; the review's facts and focus in English, Korean and Arabic at 1440, 1024 and 390; nothing sent before the confirmation; one request for the chosen credential with the entered values; the service's answer; the new secret revealed; the rows of both credentials; no second rotation offered on the bounded credential; no credential cell wider than its column. Not covered by it: an unknown outcome and its retry, a refused rotation, the suspended and limit states, and keyboard order beyond the focus checks; the first two are covered by unit tests of the store and the model only.

Screenshots looked at by the writer: the review in English and Arabic at 1440, the Arabic time lines in a crop, the list in Korean at 390 and in English at 1440.

## Review of step 3

Independent change review (independent-reviewer role, AI, by reading, nothing executed): approve with conditions. Two blocking findings, both about statements on the screen that could be false, corrected: a refused rotation showed the sentence about retired accounts; and the end time was presented as exact although it is computed from the last read. Confirmed by reading: the request cannot name another credential or other values than those reviewed, a retry sends the identical request, the parser's rules match what the server sends. Bounded re-review: approve. Its two small remarks are left as they are: the lead says the current credential stays valid "only for the time you set", although one that ends sooner stays valid for less, which the screen states in that case; and the hint says "from now" for a time counted from confirmation, which the estimated-end note states.

Composition and copy review (ui-designer role, AI, by reading the copy and the rendered screenshots): first FAIL with four blocking findings, all corrected: the Arabic UTC text was scrambled; the review and the list showed the same instant as different dates, so both local and UTC time are shown; the sentence about the end not being moved back was ambiguous; the row of a replaced credential showed no end time. Its strings were adopted unchanged. Re-review: FAIL on one new blocking defect, an identifier running over the row buttons at 1440, corrected (the column widths above), with three string corrections adopted. Final re-review: PASS, the overflow closed in the screenshots it opened.

Left as the reviewer marked not blocking: the three buttons of the review wrap two and one at 1440; the Arabic local time line uses the screen's pickup-time form and not the list's sentence form; in the list at 1440 a time and its morning or afternoon marker can break onto separate lines in Korean and Arabic, in the existing validity column as well as in the new sub-line.

# Remaining

- The owner's screen review of the rotation on M04. No review above is a screen acceptance.
- The residuals of the plan are unchanged: nothing detects that a workload still uses the old credential when it ends, nothing checks work in progress, the pickup stays bound to the issuing session, and records live in process memory in the pre-launch service.
- Not built, as separate tickets under the plan's owner questions: a distinct rotation permission, recording where a credential is used, a prompt for an expiring credential.
