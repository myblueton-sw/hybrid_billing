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

Screens for items 1 to 5: the three issuance permission areas are on M03 (PR 274, 276, 277; [screen plan](issuance-authority-screen-plan.md)), and every read and command of items 1 to 4 is on the O03 documents and payment terms views in ten slices (PR 278 to 288; [O03 screens plan](issuance-setup-screens-plan.md), which records each slice's checks, reviews and open notes). All of it is a candidate: the render checks used intercepted synthetic responses, only the flows listed below ran against a real server, no native speaker reviewed the Korean or Arabic copy, and the owner has reviewed no screen.

Real-server run of the O03 screens (`TestIssuanceSetupActualScreenBrowser` with `tools/o03-setup-live-qa.mjs`, 2026-11-24; English, Korean and Arabic, each against its own fresh server, at 1440 only): a browser with separate sessions, every request forwarded to an isolated in-process server, ran two phases, and the test checks the retained records and their actors after each. Phase one: the protected settings rules proposed and confirmed, one verification channel registered and accepted with statements, where the service is interrupted once after the acceptance: the screen shows an unknown outcome and guards leaving, retrying the exact request returns the same stored outcome, and recovering the recorded attempt completes the acceptance with the statement used; payment terms recorded and approved. Phase two, after the test clock advanced 24 hours and the grants were made again: a payment account change requested and then denied by the verifier, which ends it with no decision offered; a change requested with a statement, verified with a statement and rejected; a further change for the same setting requested, verified and approved, by three further people; an unused statement withdrawn by its author; and the terms applied to the contract and that approved. The test also checks that every statement recorded through the screen is used by its step. Apart from the injected interruption and its replay, no request of the acting sessions was refused and no page error occurred in the passing run. The test is skipped without its browser configuration, so the ordinary suite does not run it. Not run against a real server: an unknown outcome on steps other than channel acceptance, a statement left unused by a refused step, the narrower widths. In Korean and Arabic the run drives the same steps by their translated button names and checks the retained records; the sentences shown on screen are asserted in English only. The withdrawn statement was recorded by the test through the public command, not through the screen.

Findings of that run, none changed here:

- The contract and organization reads need installation administration, which the workspace administrator role does not carry and which the sample installation gives only to its three seeded administrators. A step holder without it cannot open these screens, and a change needs five distinct people, so the run gave step capabilities to administrators and gave two extra people installation administration through the seed helper, not a public command (HYB-348).
- A later step is refused as a denied permission when an earlier actor's grant has lapsed: applying terms needs the term's author and approver to hold their permissions still, as a change request needs the channel's registrar and acceptor to. The screen can only show its general permission sentence (HYB-349).
- The payment terms view needs the protected settings read permission and the separate payment terms read permission.

What the journey tests established (`issuanceauthorityjourney_test.go`, `issuanceauthorityjourneydraft_test.go`):

- From a sample installation with no issuance grant and no issuance configuration, two approved invoice drafts are reached with every issuance setup, provisioning and draft step sent as a public command: the configuration is proposed and confirmed through `issuance_setup`, every capability is granted through the public provisioning commands, rules, three verification channels (customer, issuing entity, document kind), three scheduled changes (recipient, payment account, template), the term and its binding are made through `issuance_setup`, and the draft is submitted, reviewed and approved through `issuance_draft` by a submitter and a separate reviewer.
- The second invoice reuses the setup: after a second final amount is approved, a second draft is approved with the same configuration and binding, with no new configuration, channel, change, statement, term or binding and no further issuance grant. Both invoices are handled inside one grant validity window (12 hours in the fixture); the test does not advance time between them beyond what the finance fixtures do.
- After the 24-hour channel age the 12-hour grants made before the wait have expired (by the fixture's validity, not asserted), and a change request stays refused until the channel's registrar and admitter are provisioned again as well as the requester, verifier and approver.
- A draft can be submitted while a protected reference is missing; preparing its review is refused as a failed prerequisite (422) and no review is retained. A submitter who holds no approval capability is refused permission to prepare the review; the self-approval rule itself is not exercised by these tests. The member's submission is refused (403) before the draft capabilities are granted.

Limits of these tests: the five additional people are created and activated, the member's password set and every session opened by test helpers that call the identity module and the server directly, not by public user or sign-in commands; the financial source of each invoice (candidate capture, charge and final-amount approval, ERP result) comes from the existing test fixtures, which use their own commands and some package-internal functions, not from this journey; the tests read retained state for the configuration record, the draft digest and assertions; issuance, delivery and payment are outside them; time is advanced by the test clock.

Review of the draft journey test and of these statements (role-based AI reviewer separate from the writer, by reading, nothing executed), 2026-11-24: FAIL with four findings, all statements saying more than the tests assert (public commands alone; no review retained; the self-review attribution; nobody can submit). Corrected by wording and by tighter assertions; corrective re-review: PASS. Open suggestion not adopted: the grant and setup baselines are taken after the finance fixtures, so a fixture adding issuance grants before the first draft would go unseen. Not checked by the reviewer: transitive finance helpers, and whether preparing a review depends on the validity of the change actors' grants.

Owner-visible residuals are recorded in the evidence and protected writes contracts and await the owner: the reading of independent attestation as each separated actor attesting their own step; retained free-text statements; a change left in an unknown state blocking its subject and field until restart; the irrevocable first protected rules proposal. The payment terms, provisioning and reads contracts add theirs, among them: one irrevocable net-days value per configuration; whether governors may receive step capabilities; whether one person may hold both sides of a separated flow; the 24-hour maximum grant validity; expiry-only console ceilings; and unfiltered, unpruned proposal history.

# Next executable scope

1. Owner screen review of the O03 documents and payment terms views and of the M03 permission areas, with the open questions and review notes of [the O03 screens plan](issuance-setup-screens-plan.md).
2. The real-server run of the O03 screens does not yet cover an unknown outcome on steps other than channel acceptance or the narrower widths.
3. Configuration replacement, release operations and durable production recovery remain explicit separate scopes.
4. Separate issues, with their state on 2026-11-24:
   - HYB-344 (M01 impact gaps): items 2 to 5 merged as candidates (PR 292, 293, 298, 299); item 1, rows for the people who must perform the next step, awaits a design decision.
   - HYB-346 (labels of issuance grants on M03 and P02): labels merged (PR 294); rendered check and two wording decisions open.
   - HYB-347 (M03 history row, navigation caption, refusal heading): items 1 to 3 merged as candidates (PR 295, 296); item 4, wording decisions, open.
   - HYB-342, HYB-345, HYB-348, HYB-349: each has a document of verified facts and options and awaits the owner's decision; see the next section.

# Decisions waiting for the owner

Four documents were prepared on 2026-11-24. Each records facts read from code, a test where one was added, options, root's reading, an independent review and what was not checked. None of them decides anything.

| Question | Document | What it found |
| --- | --- | --- |
| How a step holder who is not an installation administrator reaches the O03 setup views | [HYB-348 options](../HYB-348/step-holder-screen-access-options.md) | The service already allows every setup read and step to such a person; only the console's path through the contract list stops them. A change needs five distinct people and the sample installation has three who can open the screen |
| Whether later steps and later invoices keep rechecking the current permission of the people who performed earlier steps | [HYB-349 options](../HYB-349/lapsed-grant-options.md) | Grants last at most 24 hours and a channel must age at least 24 hours, so grants must always be made again; every draft review rechecks everyone involved in setup; the planning text does not require this; both numbers are code literals |
| How a setup record that can no longer progress is released | [HYB-345 options](../HYB-345/release-options.md) | The module can discard, retire and withdraw but the service uses none; rules can be neither discarded nor revised, and the departure of either of their administrators stops every protected step |
| Whether returned digests may reveal withheld values | [HYB-342 assessment](../HYB-342/digest-oracle-assessment.md) | Unkeyed digests cover withheld values; low consequence while references are synthetic samples; to be closed before real references exist |

The first three are one question seen from three sides: what happens when the people involved in setup change, lapse or leave. They are better decided together, in this order: HYB-349 first, because it fixes whose standing matters at later steps; then HYB-345, whose options for an absent person depend on that answer; then HYB-348, which is independent of the other two in its first option and can also be taken first if the owner wants the screens usable by separated people soonest. HYB-342 is independent of all three.

Smaller decisions recorded on their tickets: the M01 rows for people who must perform the next step (HYB-344 item 1); "View cost" as the action name of inspect grants and the Korean wording of synthetic resource kinds (HYB-346); "허용 작업" against "권한", "납부 조건" against "지급 조건" and "확인" against "검증" (HYB-347 item 4).

# Discovery evidence and traps

Existing private service entry points: financialissuanceprotectedrules.go (proposeProtectedRules/confirmProtectedRules), financialissuancechannel.go (registerChannel/admitChannel/currentChannel), financialissuancechangewrites.go (requestProtectedChange/verifyProtectedChange/decideProtectedChange). Public setup currently exposes configuration operations only. configuredChangeSubject in financialissuancesetuporigin.go derives recipient from contract customer, account from billing-profile issuer and template from configured kind; do not accept caller-selected subjects or invent tax-field support.

Provenance is an enum, not a separate registry. protected.Ref contains ID/fingerprint. modules/evidence offers custody/retention but is not connected here as verification evidence; release_evidence commands are unrelated. These are discovery findings, not an accepted new public contract.

Channel maturity is evaluated at change request time. Waiting after an immature request does not repair it. Channel custodians cannot request/verify through their own channel; independent admission alone is insufficient. modules/protected Of/Trail settle state, so a promised nonmutating read needs an explicit observation policy or new nonsettling snapshot. Never promote service uncertainty merely because module state appears complete, adopt foreign records or hide retained history solely because an original actor lost a right.

Required next tests include Inspect/grant loss/foreign scope, module-only records, absent versus unavailable rules, unchanged unknown phases, retained read after actor revocation, exact request-time maturity boundary, self/custodian/channel separation, stale configuration/cursors, bounded output and hold-safe reads.

# Local runtime boundary

The owner confirmed login at http://127.0.0.1:8830/?service=api. Vite serves /private/tmp/hyb341 and proxies the existing API at port 8820. Autofill is runtime-only local preview configuration; no new login implementation or committed credentials. The API uses /private/tmp/hyb330-api and in-memory credential/domain registries. It was not rebuilt or restarted for the latest setup endpoints, and new endpoint/grant availability is not claimed. Do not assume these processes or runtime-only values exist on the receiving machine. Do not copy credentials into tickets, Git or this handoff.

The old /private/tmp/hyb341-session-handoff.md is stale. This tracked handoff, current planning/evidence documents and the final HYB-341 ticket checkpoint supersede it for continuation. On transfer, release machine:mac_mini; the receiving machine must claim ownership before edits.
