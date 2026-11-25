---
type: DesignOptions
title: Later issuance steps and invoices when an earlier actor's grant has lapsed
description: Verified facts, consequences and options for the rule that every later issuance step, and every invoice draft review, rechecks the current permission of the people who performed the earlier steps.
status: draft
---

# Outcome and ownership

HYB-349, machine:mac_mini. This document puts a decision before the owner. It decides nothing and changes no product code; it adds one server test that pins a fact. Any change to these checks alters permission behaviour and the financial approval path and needs independent design review first.

# The rule as built

Each issuance record keeps the people who acted on it, and each later use of the record checks again that every one of them is an active person who still holds the permission they used. The check is on current standing, not on standing at the time they acted.

# Verified facts

Read from main at 4e856cc on 2026-11-24.

| Fact | Evidence |
| --- | --- |
| A grant of an issuance permission lasts at most 24 hours | `apps/api/server/server.go:381`, `:384`, `:387`, `MaxGrantValidity: 24 * time.Hour` for the payment terms, protected settings and draft domains; enforced in `modules/internal/grantauthority/shape.go:61` |
| A grant that has expired fails these checks, and so does a suspended person; the two are distinguishable in the code, as are an expired and a revoked grant | `financialissuanceprotectedaccess.go:30` and `:62`, `financialissuancetermaccess.go:30`, `modules/auth/listing.go:49`, `apps/api/server/grantauthority.go:192` |
| A verification channel may be used for a change request only after it has been accepted for at least the rules' channel age, which cannot be set below 24 hours | `modules/protected/types.go:25` and `:138`; the age is checked at verification against the request time, `modules/protected/changes.go:215` |
| So the grants under which a channel was registered and accepted have always ended before the channel can be used, and both people must be granted again before any change request through it | follows from the two rows above; shown by `TestIssuanceSetupJourneyThroughPublicProvisioningOnly`, where the request stays refused after 24 hours until the registrar and the admitter are provisioned again |
| Every use of a change rechecks its requester, its verifier and each approver once recorded, the registrar and admitter of its channel, and the proposer and confirmer of the protected rules | `financialissuancechange.go:235` to `:286` (`changeStanding`, `:247`, `:250`, `:264`, `:269`), `financialissuancechannel.go:71`, `:91`, `financialissuanceprotectedaccess.go:128`; suspension tests in `financialissuancestepimpact_test.go` |
| Approving payment terms rechecks their author; applying them rechecks author and approver; approving the application rechecks the person who applied | `financialissuanceterms.go:45`, `financialissuancetermbinding.go:60` to `:75`; real-server run in the HYB-341 handoff, where applying terms was refused until the author's and approver's grants were made again |
| Preparing the review of an invoice draft, and approving it, recheck: for each of the three protected changes it uses, everyone in the row above; the author and approver of the payment terms and of their application; the draft's submitter; and, on administrator standing and not on a 24-hour grant, the proposer and confirmer of the configuration | `financialissuancedraftreviewwrites.go:47` to `:56`, `:134` to `:146`, `financialissuanceconfiguredconsumption.go:50`, `:66`, `financialissuanceconsumptionrefs.go:11` to `:39`, `financialissuanceconsumptionterms.go:31`, `financialissuancedraftreview.go:131`, `financialissuanceapplication.go:73` |
| Submitting a draft does not recheck the setup actors; it rechecks the configuration's governors and the live financial source | `financialissuanceapplication.go:92` |
| No path reviews or approves a draft without these rechecks: an existing review identifier returns its stored result, and a prepared review cannot be approved without the full recheck. No `issuance_draft` command after approval exists in this tree | `financialissuancedraftreviewwrites.go:33`, `:123`, `trusthold.go:57` |
| With setup complete and a draft submitted, removing the access of any one of the requester, the verifier or the approver makes the review preparation fail with `permission_denied`, and no review is retained | `TestIssuanceDraftReviewRechecksSetupActors`, which passes |
| The same test fails the review when the admitter loses access. By reading, references are evaluated before coverage, so that refusal comes from the channel check; the outcome alone does not distinguish it from the payment terms check, since the admitter also approved the terms | same test; `financialissuancedraftreviewwrites.go:52`, `:56`, `financialissuancechannel.go:91` |
| The same test fails the review when the draft's submitter loses access. The submitter is also the channel registrar in that fixture, and a submitter-related check runs before any setup check, so this case shows nothing about setup actors | same test; `financialissuancedraftreview.go:124` or `:131`, not isolated |
| A person refused for this reason receives `permission_denied`, the same code as when their own permission is missing; the screens show their general permission sentence | `financialissuanceprotectedaccess.go:47` to `:73`; O03 copy in `terms-copy.ts`, `changes-copy.ts` |

The test removes access by suspending membership. That a lapsed grant fails the same check is established by reading (`protectedActor` requires an applied grant that is active now) and, for channel custodians and for payment terms, by the two time-based runs cited above; it was not shown by a time-based test for the draft review.

# Consequences

- Setup cannot be finished inside one grant window. The channel age alone forces at least one renewal of the registrar's and the admitter's grants.
- Later invoices need the setup people again. A draft for a later invoice reuses the setup, but its review is refused unless the people who registered and accepted the channels, requested, verified and approved the three protected changes, and recorded, approved and applied the payment terms, hold their permissions at that moment: at least five distinct people per protected change, plus those of the payment terms. With grants of at most 24 hours this means granting each of them again, through two governors each time, for every grant window in which a draft review is prepared or approved. Drafts reviewed inside one window share the grants, as `TestIssuanceJourneyReachesApprovedDraftsForTwoInvoices` shows.
- A person who has left blocks invoicing. If one of those people is suspended or removed, no later draft can be reviewed with that setup. The same holds for the administrators who proposed and confirmed the configuration and the protected rules, whose standing as administrators is rechecked. The service has no operation to replace the setup or to release such a record (HYB-345).
- The refused person cannot tell why. The reviewer holds the right permission and is told that permission is denied.

# Options

1. Keep the rule and state it. Document that setup actors must hold current grants at every use, and add guidance text naming this possibility in the permission sentence. No change to checks. Costs: the operational burden above remains; the screens still cannot say whose grant lapsed.
2. Keep the rule and make the cause visible. Return a distinguishable refusal code, or name the earlier step whose actor no longer qualifies, when a later step fails for this reason. Costs: a public-contract change on several commands; it discloses to the caller that a named or unnamed other person lost a permission, which needs a decision on how much to disclose.
3. Recheck that the earlier actor is still an active member, not that they still hold the grant. A step stays valid when its actor's time-limited grant has run out, and is still blocked when the person is suspended or removed. Costs: this changes what the checks protect; a deliberate revocation of a grant would no longer stop later use, unless revocation is distinguished from expiry, which the code can do; with grants of at most 24 hours, not renewing is the usual way a duty is withdrawn, so a person whose duty ended without suspension or explicit revocation would remain valid indefinitely; needs financial and security review.
4. Check standing at the time of the act only. Later steps verify the retained record and not the present standing of its actors. Costs: the largest change in protection; removing a person no longer affects what they completed; M01 impact rows that say "rechecked" would become false for these records.
5. Lengthen the permitted grant validity for these domains. Reduces how often grants must be made again without changing the checks; it is one limit per domain. Costs: long-lived high-trust permissions; it does not help when a person leaves.
6. Let a current holder of the permission re-endorse or replace a retained step whose actor no longer qualifies. Keeps the rule and gives a way forward when a person leaves. Costs: new governed operations with their own separation rules; it belongs with HYB-345.

Options 1, 2 and 6 can be combined with any of the others. Options 3 to 5 are alternatives.

# Recommendation

Decide the question in two parts. For the steps of one setup record (verify, decide, accept, approve, apply), keep the present rule and add option 2, because there the people are still working together and a visible cause is what is missing. For the use of completed setup by later invoices, the present rule makes each later invoice depend on at least five distinct people per protected change, plus those of the payment terms, being granted again within a day; the planning text read does not require this, as the cross-check below records; option 3 with revocation kept distinct from expiry is the candidate to examine. This is root's reading and not a decision.

# Planning cross-check

Read on 2026-11-24 from `docs/planning/v1.16` in the authorized local workspace; several of these planning files are not yet in the published repository. Quotations are translated from the Korean planning text.

| Planning statement | Where | Bearing on this question |
| --- | --- | --- |
| Changes to a payment account, document recipient, tax identifier or template "require separate verification, approval and a validity period"; the references before and after, the verifier and approver, and the evidence of confirmation through an existing registered channel are preserved; "the control itself is mandatory even though the responsible people and the specific periods are undetermined" | `detailed-requirements.md`, QA-01 | Requires separation, preserved evidence and a validity period. Does not say that the verifier or approver must still hold a permission when the change is later used |
| Acceptance: such a change is "reflected in documents only on the effective date after confirmation through the existing channel and separate approval" | `detailed-requirements.md`, AUD-QA-01 | The condition for use is the completed confirmation and approval and the effective date |
| Re-approval is required when the input, policy, exchange rate, attribution or ERP draft changes | `requirements.md`, HB-23, requirement | Concerns changes to what was approved |
| "Long-running work checks the permission and the target hash again"; the approval subject binds the "eligible approver" | `requirements.md`, HB-23, implementation direction (a proposal, not the requirement) | Concerns the person performing the work, and eligibility when approving; no later time is stated |
| Acceptance: external execution is blocked on "withdrawal" right after approval | `requirements.md`, HB-23, acceptance | Ambiguous between withdrawing the approval and withdrawing a permission; the nearest statement to a contrary reading |
| "The current permission is checked again at job start, chunk boundaries, external submission and download resume"; a change of permission version invalidates waiting jobs and caches | `requirements.md`, HB-03 | Concerns the person performing the work and work still waiting, not completed approvals |
| Queued work started by a person rechecks the current permission; whether processing that was already approved continues is decided per command; the actor recorded in approval history is unchanged | `retention-access.md` | States that continuing after a permission change is a per-command decision |

Reading: the planning requires a recheck of the person acting now and of the thing that was approved, and leaves to each command whether already approved processing continues after a permission change. No statement was found that requires the people who performed earlier steps to hold their permissions at the time of a later step or of a later invoice. The rule as built therefore goes beyond the planning text. It is not contradicted by it either, so whether to keep it remains a decision.

Two numbers that drive the consequences are fixed in code and are not in the planning text, which leaves the periods undetermined: the maximum grant validity of 24 hours (`apps/api/server/server.go`, one literal per permission domain) and the minimum channel age of 24 hours (`modules/protected/types.go:25`). The owner's instruction of 2026-11-17 on open values was that provisional values ship as installation defaults that can be changed without a code change. By that instruction the maximum grant validity, at least, is a candidate to become an installation setting with bounds; this is option 5 in a form the owner asked for in general terms in another context, and it still needs its own design and review because it lengthens high-trust permissions.

Not searched: the source chapters cited by QA-01 (B1500) and the HB-257 baseline outside `docs/planning/v1.16`.

Review of this section (role-based AI reviewer separate from the writer, by reading, 2026-11-24): PASS; the reviewer found no contradicting statement in the six planning files it searched and supplied four further statements, which are included above.

# Not checked

- The source chapters behind QA-01 and the HB-257 baseline; the planning documents under `docs/planning/v1.16` were read and are summarised above.
- Whether a public operation to revoke an issuance grant exists.
- Which people the financial source read rechecks at submission.
- A time-based test of the draft review after grants lapse; the finance fixtures of that test also depend on time-limited grants.
- Whether issuing, delivering or recording payment recheck the same people again.
- The channel registrar in isolation at draft review.

# Review

Independent design review (role-based AI reviewer separate from the writer, by reading, nothing executed), 2026-11-24: FAIL with three findings — "at every invoice" overstated a recheck that happens per grant window; the set of rechecked people omitted the channel custodians, the draft submitter, the administrators of the configuration and the rules, and the approval step; and the note on the registrar and admitter cases of the test was inaccurate in both directions. Corrected with the reviewer's wording and references, including its confirmation that an expired grant fails the checks and that expiry, revocation and suspension are distinguishable. Judged to hold: the other facts, the three clean cases of the test, the absence of a path around the recheck, the options and the labelling of the recommendation. Corrective re-review of the three findings: PASS.
