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

# Next executable scope

1. Define and review bounded protected-setup discovery for selected contract plus configuration ID/digest. Reuse current governed Inspect, human/session/console boundaries. Keep retained phases separate from module observations and current action eligibility. Use ten-row pages, scoped snapshots and output-size admission.
2. Resolve the evidence-capture contract before exposing protected writes. No existing source binds channel/change evidence IDs to owned verified artifacts. Define source, immutable reference/content identity, owner/scope, access/retention and the independent human attestation. An upload, hash or URL is not proof of customer instruction, account ownership or contact control.
3. Connect protected rules, channel registration/admission and requested/verified/approved references after those boundaries pass review.
4. Connect O03 payment terms and independently approved applicability bindings. Use explicit net days, exact term revision and effective dates; no invented default calendar or terms.
5. Complete missing M03 business-right provisioning, then verify a truly unseeded operator journey to B09 readiness and a second applicable invoice reusing unchanged setup. Keep durable production recovery and configuration replacement as explicit remaining scopes.

# Discovery evidence and traps

Existing private service entry points: financialissuanceprotectedrules.go (proposeProtectedRules/confirmProtectedRules), financialissuancechannel.go (registerChannel/admitChannel/currentChannel), financialissuancechangewrites.go (requestProtectedChange/verifyProtectedChange/decideProtectedChange). Public setup currently exposes configuration operations only. configuredChangeSubject in financialissuancesetuporigin.go derives recipient from contract customer, account from billing-profile issuer and template from configured kind; do not accept caller-selected subjects or invent tax-field support.

Provenance is an enum, not a separate registry. protected.Ref contains ID/fingerprint. modules/evidence offers custody/retention but is not connected here as verification evidence; release_evidence commands are unrelated. These are discovery findings, not an accepted new public contract.

Channel maturity is evaluated at change request time. Waiting after an immature request does not repair it. Channel custodians cannot request/verify through their own channel; independent admission alone is insufficient. modules/protected Of/Trail settle state, so a promised nonmutating read needs an explicit observation policy or new nonsettling snapshot. Never promote service uncertainty merely because module state appears complete, adopt foreign records or hide retained history solely because an original actor lost a right.

Required next tests include Inspect/grant loss/foreign scope, module-only records, absent versus unavailable rules, unchanged unknown phases, retained read after actor revocation, exact request-time maturity boundary, self/custodian/channel separation, stale configuration/cursors, bounded output and hold-safe reads.

# Local runtime boundary

The owner confirmed login at http://127.0.0.1:8830/?service=api. Vite serves /private/tmp/hyb341 and proxies the existing API at port 8820. Autofill is runtime-only local preview configuration; no new login implementation or committed credentials. The API uses /private/tmp/hyb330-api and in-memory credential/domain registries. It was not rebuilt or restarted for the latest setup endpoints, and new endpoint/grant availability is not claimed. Do not assume these processes or runtime-only values exist on the receiving machine. Do not copy credentials into tickets, Git or this handoff.

The old /private/tmp/hyb341-session-handoff.md is stale. This tracked handoff, current planning/evidence documents and the final HYB-341 ticket checkpoint supersede it for continuation. On transfer, release machine:mac_mini; the receiving machine must claim ownership before edits.
