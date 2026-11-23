---
type: Handoff
title: Reusable issuance setup handoff
description: Completed implementation, preserved verification and next operator prerequisites for transfer to another machine.
status: active
---

# Transfer boundary

The owner requested commit, ticket update and main merge on 2026-11-23, then transfer to another environment. This grants integration, not deployment, real issuance, delivery or payment. HYB-341 remains functionally incomplete. Continue from fetched main after the recorded merge; do not restore missing historical companion material or repeat passed reviews. The owner performs browser verification. Root implements and tests logic; changed designs still receive scoped Claude ui-designer review. Native-language and owner acceptance are not inferred from source reviews.

# Completed scope

Governed ERP result reconciliation and B09 issuance draft discovery, context, submission, saved review, independent approval and owned recovery are implemented. Reusable configuration backend context/history/options/action/propose/confirm endpoints are implemented and reviewed. O03 documents now includes selected-date source facts, retained history, explicit synthetic preset/languages, proposal/confirmation check-answers and exact retry/recovery results. Inputs remain in memory per session/contract/date across tab navigation, not across a full reload.

Latest feature commit before main integration: 235a424. All 34 setup logic tests passed; console typecheck/build passed with the existing chunk-size warning. Independent source review and actual Claude source/terminology corrective review passed. The detailed scopes and limitations remain in issuance-setup-screen-plan.md and issuance-setup-write-client-plan.md. Earlier rendered B09 evidence used seeded prerequisites and does not establish the unseeded operator journey.

Main integration includes the required PR chain 246, 248, 245, 249, 250 and 252. Unrelated open licensing, accessibility and service-account PRs are excluded. Main's existing HYB-304 session changes are preserved. Manual conflicts were field/test unions and distinct appended document sections, with no intentional authorization override. Ticket checkpoints record final merge/test results.

# Progress since the transfer (2026-11-23, machine:mac_mini)

Work resumed from main c8aebd6 on this machine and merged the following through reviewed pull requests. Each contract document records its own review rounds, executed checks and limits; this list only orients.

| Handoff item | Merged | Documents and code |
| --- | --- | --- |
| 1. Protected-setup discovery | PR 253, 254, 255 | [discovery contract](issuance-protected-setup-discovery-contract.md); `modules/protected` `Peek` / `PeekTrail`; reads `issuance_setup.protected`, `protected_channels`, `protected_changes`; console decoders in `setup/protected.ts` |
| 2. Evidence capture | PR 257, 258, 259, 260 | [evidence contract](issuance-setup-evidence-contract.md); attest and withdraw commands, `issuance_setup.attestations`, `issuance_setup.references`; console client in `setup/attestation.ts` |
| 3. Public protected writes | PR 261, 262, 264 | [protected writes contract](issuance-protected-writes-contract.md); seven commands in `financialissuanceprotectedwrites.go` consuming attestations; console client in `setup/protected-writes.ts` |
| 4. Payment terms | PR 265, 266, 267 | [payment terms contract](issuance-payment-terms-contract.md); read `issuance_setup.terms` and four commands in `financialissuancetermwrites.go`; console client in `setup/terms.ts` |
| 5. Authority provisioning | PR 268, 269, 270, 271, 272 | [provisioning contract](issuance-authority-provisioning-contract.md) and [reads contract](issuance-authority-reads-contract.md); namespaces `issuance_protected_authority`, `issuance_term_authority`, `issuance_draft_authority` with six commands plus `history` and `context` each; journey test `issuanceauthorityjourney_test.go` |

Separate issues, by owner instruction that anything outside the checklist gets its own ticket: HYB-343 (two server tests failing on main, fixed in PR 256, which also added pending issuance settings to M01 bulk revocation impact), HYB-342 (unkeyed digest hypothesis, open), HYB-344 (remaining M01 impact gaps, open), HYB-345 (release, discard and withdrawal operations for protected setup records, open), HYB-346 (labels for issuance grants on existing M03 and P02 rows, open).

Not done for items 1 to 5: no screen is wired to any of these reads or commands, and that composition needs scoped ui-designer and terminology review; the owner has verified no screen. The console client for the provisioning namespaces and the M03 screen plan (`issuance-authority-screen-plan.md`, not yet in the tree) were in review when this section was last updated and are not merged by it.

What the journey test established: from a sample installation with no issuance grant, confirmed rules, an admitted channel, an approved term and binding and a scheduled payment-account change are reachable with every capability granted through the public provisioning commands. After the 24-hour channel age all grants have expired, and a change request stays refused until the channel's registrar and admitter are provisioned again as well as the requester, verifier and approver. The contract and its confirmed configuration are still created by package-internal helpers in that test.

Owner-visible residuals are recorded in the evidence and protected writes contracts and await the owner: the reading of independent attestation as each separated actor attesting their own step; retained free-text statements; a change left in an unknown state blocking its subject and field until restart; the irrevocable first protected rules proposal. The payment terms, provisioning and reads contracts add theirs, among them: one irrevocable net-days value per configuration; whether governors may receive step capabilities; whether one person may hold both sides of a separated flow; the 24-hour maximum grant validity; expiry-only console ceilings; and unfiltered, unpruned proposal history.

# Next executable scope

1. Merge the domain-aware provisioning console client after its source review, then the M03 screen plan after its design and terminology review, then implement the three M03 permission areas.
2. O03 documents and payment-terms composition for items 1 to 4, led by the ui-designer role, then owner screen verification.
3. Extend the journey to B09 readiness and a second applicable invoice reusing unchanged setup, including the draft capabilities through real commands. Configuration replacement, release operations and durable production recovery remain explicit separate scopes.
4. HYB-346 labels once the capability terms of the screen plan are reviewed.

# Discovery evidence and traps

Existing private service entry points: financialissuanceprotectedrules.go (proposeProtectedRules/confirmProtectedRules), financialissuancechannel.go (registerChannel/admitChannel/currentChannel), financialissuancechangewrites.go (requestProtectedChange/verifyProtectedChange/decideProtectedChange). Public setup currently exposes configuration operations only. configuredChangeSubject in financialissuancesetuporigin.go derives recipient from contract customer, account from billing-profile issuer and template from configured kind; do not accept caller-selected subjects or invent tax-field support.

Provenance is an enum, not a separate registry. protected.Ref contains ID/fingerprint. modules/evidence offers custody/retention but is not connected here as verification evidence; release_evidence commands are unrelated. These are discovery findings, not an accepted new public contract.

Channel maturity is evaluated at change request time. Waiting after an immature request does not repair it. Channel custodians cannot request/verify through their own channel; independent admission alone is insufficient. modules/protected Of/Trail settle state, so a promised nonmutating read needs an explicit observation policy or new nonsettling snapshot. Never promote service uncertainty merely because module state appears complete, adopt foreign records or hide retained history solely because an original actor lost a right.

Required next tests include Inspect/grant loss/foreign scope, module-only records, absent versus unavailable rules, unchanged unknown phases, retained read after actor revocation, exact request-time maturity boundary, self/custodian/channel separation, stale configuration/cursors, bounded output and hold-safe reads.

# Local runtime boundary

The owner confirmed login at http://127.0.0.1:8830/?service=api. Vite serves /private/tmp/hyb341 and proxies the existing API at port 8820. Autofill is runtime-only local preview configuration; no new login implementation or committed credentials. The API uses /private/tmp/hyb330-api and in-memory credential/domain registries. It was not rebuilt or restarted for the latest setup endpoints, and new endpoint/grant availability is not claimed. Do not assume these processes or runtime-only values exist on the receiving machine. Do not copy credentials into tickets, Git or this handoff.

The old /private/tmp/hyb341-session-handoff.md is stale. This tracked handoff, current planning/evidence documents and the final HYB-341 ticket checkpoint supersede it for continuation. On transfer, release machine:mac_mini; the receiving machine must claim ownership before edits.
