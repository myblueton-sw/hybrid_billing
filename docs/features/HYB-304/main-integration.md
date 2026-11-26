---
type: IntegrationRecord
title: HYB-304 M04 branch integrated with main and credential rotation
description: How the bounded M04 service-account branch was merged with main after credential rotation landed there, what was decided, what was verified and what is left for the owner.
status: draft
---

# Scope and ownership

HYB-304, machine:mac_mini, branch `feat/HYB-304-identity-list-paging` (PR 251). The branch was written on 2026-11-20 and 2026-11-21 by another agent and stopped as a draft on the server coverage criterion. On 2026-11-26 the owner chose, when asked in the working session, to accept the 28 uncovered units recorded in [the coverage residual assessment](m04-coverage-residuals.md) and to integrate with main. This record covers that integration, made by the Claude root session.

# Facts at the start

- The branch was 28 commits ahead of and 510 behind main. A merge of main `ddf09982` reported 46 conflicting hunks in 24 files.
- Main had received credential rotation (HYB-334): a module call that replaces a credential, the command `service.rotate_credential`, rotation fields in the whole-population services read, and a rotation mode of the old M04 issue form.
- The branch had removed that whole-population read and rebuilt M04 on bounded reads, canonical detail reads with server-computed eligibility, credential and expiry indexes in the identity module, a delivery index for secret pickups and a declaration that issuance requires.

# What was done

Lists both sides extended (settings, hold table, struct fields, test lists, glossary rows) were joined. The rest:

- **Identity module.** `ReplaceCredential` now stores the new credential through `putCredential`, moves the replaced credential in the expiry index when its end is brought forward, and advances the credential query version. `credentialInfos` returns the rotation-aware `credentialInfo`.
- **Reads.** The bounded credential rows and the credential detail carry `replaces`, `replaced_by` and `rotation`; the limits carry the overlap bounds and the pickup time; the API description says so. The whole-population read stays removed.
- **Rotate command.** It records its pickup through `putPickup` and `putDelivery`. Before this, a rotated credential's pickup read as never collected; two of main's rotation tests failed on exactly that and pass now.
- **Eligibility.** The credential detail read gains `can_rotate` and `rotate_step_up_state`, additive fields. `can_rotate` is true when the credential is active, has started and has not been replaced, and the account could be issued a credential by this caller at this observation; it is computed from the facts the command enforces.
- **Console.** The strict readers take the new fields and reject contradictions. The attempt store freezes a rotation's credential and overlap, hands them to the sender, and settles a rotation only on `credential_rotated` and an issue only on `credential_issued`. `issueExec` sends a held attempt as what it was; before this the branch resent every held attempt as an issue. `RotateControl` in the credential detail takes the new validity, the overlap and a reason, shows a review step and then sends. The credential status shows the rotation facts. The held-attempt list opens the rotate form for a rotation. Copy is main's `rotate-copy.ts`, unchanged.
- **Superseded.** `apps/console/tools/credential-rotation-qa.mjs` drove the old M04 form and is removed; `tools/m04-rotation-qa.mjs` replaces it.

# Open for the owner

- **Rotation now requires a current declaration.** On this branch issuing a credential requires an eligible owner and a current declaration (`requireServiceDeclaration`). Rotation issues a credential, so the rotate command now makes the same check; without it rotation would be a way around the branch's rule. On main before this merge a rotation needed no declaration. This narrows what main allowed and can be reversed by removing one check.
- **The rotate form does not keep an unsent draft across navigation.** The branch's issue form keeps its draft in the recovery store; the rotate form keeps its fields in the component. A held or sent rotation is kept, as any attempt is.
- **`can_rotate` and `rotate_step_up_state` are new public read fields.** They are additive and were reviewed independently under the owner's standing rule for additive fields.

# Verification

See the pull request for the run results of this integration; they are recorded there with the revision they were run on.

# Not verified

- Rotation under a trust hold, and with a step-up requirement, in the browser.
- A held rotation retried from the held-attempt list in the browser; the store's behaviour is covered by unit tests.
- The other M04 runners of this branch (`m04-*.mjs`) were not rerun after the integration.
- Owner screen acceptance.
