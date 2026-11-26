---
type: DesignOptions
title: Releasing issuance setup records that can no longer progress
description: Verified facts about setup records that block further work and cannot be released through the service, with options for governed release, discard, retirement and withdrawal.
status: draft
---

# Outcome and ownership

HYB-345, machine:mac_mini. This document puts a decision before the owner. It decides nothing and changes no product code; it adds one server test that pins three of the facts. Any operation chosen here ends or replaces a control record and needs its own design with independent financial and security review.

# Verified facts

Read from main at 84cff28 on 2026-11-24. Rows marked "test" are pinned by `TestIssuanceSetupRecordsThatCannotBeReleased`, which passes.

| Fact | Evidence |
| --- | --- |
| The protected module can discard a pending rules proposal, retire a channel and withdraw a change | `modules/protected/registry.go:167` (`Discard`), `:279` (`Retire`), `modules/protected/changes.go:287` (`Withdraw`) |
| The service calls none of the three for issuance setup; it uses the module's propose, confirm, register, admit, request, verify, approve and read operations only | every use of `a.fields.` in `apps/api/server` |
| While a change for a subject and field is requested or verified and not decided, a new request for the same subject and field is refused as a failed prerequisite | test; `financialissuancechangewrites.go:107` |
| Such an open change ends by itself: a requested change at its effective time, which the rules may allow up to 366 days ahead; a verified change at the earlier of its effective time and the verification time plus the validity period (up to 30 days) | `modules/protected/changes.go:52`, `:62`; `modules/protected/types.go:22`, `:24` |
| While a retained change has a phase other than recorded, that is while its request, verification or decision has an unknown outcome, every new request for the same subject and field is refused, with no time limit | test for the refusal, although in the tested case the open-change check would refuse as well; the absence of a time limit is by reading; `financialissuancechangewrites.go:101` |
| Only the person who sent the step with the unknown outcome can complete it, by sending the same step again. With several approvals required, only the last approver whose decision is unknown can complete it, and other approvers are refused meanwhile. The same holds for a channel whose registration or acceptance has an unknown outcome | `financialissuancechangewrites.go:69`, `:183`, `:275`; `financialissuancechannel.go:152`, `:220`; the real-server run in the HYB-341 handoff shows it for a channel acceptance |
| For the payment account the subject is the issuing entity, which every configuration of the same billing profile shares | HYB-341 protected writes contract |
| The service retains one rules record for the installation. Once a proposal exists, a proposal with another identifier is refused as stale evidence, whether the first is unconfirmed, has an unknown outcome or is confirmed | test (after confirmation); `financialissuanceprotectedrules.go:99` |
| So the rules of an installation cannot be discarded before confirmation and cannot be revised after it | follows from the row above |
| Confirming the rules rechecks the proposer's standing as an installation administrator, so a proposal whose proposer has left can be neither confirmed nor replaced | `financialissuanceprotectedrules.go:50`, `:165`; `TestIssuanceRulesConfirmationRechecksSuspendedProposer` |
| After confirmation, the proposer and the confirmer of the rules are rechecked as active installation administrators whenever the rules are used: registering a channel, every check of a channel, requesting a change and every check of a change. Since the rules cannot be replaced, either person leaving stops all of these with no end | `financialissuanceprotectedaccess.go:128`, `financialissuancechannel.go:61`, `:136`, `financialissuancechangewrites.go:35`, `financialissuancechange.go:236` |
| The module's discard, retire and withdraw accept any human actor and apply no permission or separation rule of their own; the proposer can discard their own proposal | `modules/protected/registry.go:177` to `:188`, `:289` to `:295`, `modules/protected/changes.go:303` to `:318` |
| The service treats a retired channel as stale, so retiring a channel would break every change that used it | `financialissuancechannel.go:105` |
| Stuck channels and ended changes keep counting against the installation's record limit | `financialissuancechannel.go:163`, `financialissuancechangewrites.go:93` |
| A channel cannot be retired through the service, and a scheduled change cannot be withdrawn before its effective time | no service call of `Retire` or `Withdraw` |
| The module allows withdrawing a requested or verified change, and a scheduled one only before its effective time and only when no later change exists for the setting; it records who ended it | `modules/protected/changes.go:303` to `:318` |
| The module records who retired a channel and refuses a second retirement by another person | `modules/protected/registry.go:289` |
| The same kind of dead end exists outside the protected module: payment terms awaiting approval whose author has left, and a draft whose prepared review belongs to a reviewer who has left | HYB-344 comments; `docs/features/HYB-349/lapsed-grant-options.md` |
| Restarting the service clears all of it, because the pre-launch service keeps these records in memory | HYB-341 handoff |

# What can get stuck today

| Record | How it gets stuck | Effect | Ends by itself |
| --- | --- | --- | --- |
| Rules proposal | proposer or pinned confirmer leaves; outcome unknown and the sender does not return | no rules, so no channel can be registered or accepted and no change requested in the installation | no |
| Confirmed rules | the values turn out wrong | every later change follows them | no |
| Confirmed rules | the proposer or the confirmer stops being an active installation administrator | no channel can be registered or accepted and no change requested, verified or decided | no |
| Channel, unknown outcome | the registrar or the pinned admitter does not return | the channel cannot be completed by anyone else | no |
| Channel | registered or accepted in error; registrar or admitter leaves | stays in the list; changes through it need both people to hold their permissions | no |
| Change, open | not verified or not decided | blocks the setting for its subject | yes, at the effective time, or earlier once verified (validity period) |
| Change, unknown outcome | the sender does not return | blocks the setting for its subject | no |
| Change, scheduled | approved in error | takes effect at its time | no |

# Options

Each operation below is separate; the owner may choose any subset. Where the module already has the effect, it applies no permission or separation rule of its own, so the service would have to supply all of the authority and separation.

1. Withdraw an open change. Let a person end a requested or verified change. Decide who: the requester only, or also a holder of the approval permission who is not otherwise involved. The module already supports it and records who ended it. Costs: a new command, a refusal set, an entry in the change history, a statement or reason to retain, and the question of what happens to the statements the change consumed.
2. Withdraw a scheduled change before it takes effect. Same operation for a change approved in error, within the module's limits. Costs: as option 1, plus the separation rule (it undoes an approval, so one person should not be able to do it alone).
3. Release a step with an unknown outcome. Let someone other than the original sender settle a record whose request, verification or decision has an unknown outcome, either by completing the read-back or by ending the change. Costs: the hardest to make safe, because the outcome in the module is by definition not known to the service record; needs the recovery semantics of each step written out, and a rule for who may act for an absent person.
4. Retire a channel. Let a person mark a channel as no longer usable. The module supports it. Costs: the service treats a retired channel as stale, so every change that used it would stop passing its checks unless that is redefined; decide the effect on a scheduled change that has not taken effect; decide who may retire.
5. Discard an unconfirmed rules proposal. Let an administrator other than the proposer discard a proposal that cannot be confirmed, so that a new one can be made. The module supports discarding a pending proposal. Costs: the service keeps a single rules record, so it must also learn to hold a history; separation between discarder and the next proposer.
6. Revise confirmed rules. Let administrators propose and confirm a new revision. The module keeps rules by revision, creates a pending next revision when rules are active and activates it on confirmation, and each change keeps the rules it was made under. Costs: the largest of these; every record stores the digest of the rules it was made under, and reads compare against the current rules, so the meaning of "made under earlier rules" must be defined for channels, changes and drafts.
7. Stop rechecking the proposer and confirmer of confirmed rules, or allow another pair of administrators to confirm them again. Removes the dead end of the row above without revising the values. Costs: it is the question of HYB-349 applied to rules, and should be decided with it.
8. Leave all of it for now and record that the only remedy in the pre-launch service is a restart, with these operations as preconditions of durable storage. No change. Costs: with durable storage a stuck record would be permanent.

# Recommendation

Options 1 and 5 first: they remove two dead ends that an ordinary mistake or a departure produces, and the module already implements both effects. Option 1 matters more than the validity period suggests, because a change that is requested and never verified blocks its setting until its effective time, which may be up to the rules' maximum lead ahead. Options 3 and 7 next, together with the decision in HYB-349, because all three ask who may act, or whose standing matters, when the original person is absent; the departure of a rules administrator stops every protected step of the installation, which makes option 7 or option 6 necessary before durable storage. Options 2, 4 and 6 before durable storage. Option 8 is acceptable only while a restart clears the records. This is root's reading and not a decision.

# What each chosen operation needs

A contract in the style of the HYB-341 protected writes contract: the command and its fields, who may send it and who may not, what is retained, the refusals, the effect on consumed statements, recovery after an unknown outcome of the release itself, the read changes that show it, the M01 impact of a pending release, the O03 screen steps with EN/KO/AR copy, tests including the separation rules, and independent review.

# Not checked

- The planning requirements for withdrawing or correcting a protected change after approval.
- Whether a consumed statement can or should be freed when its step is withdrawn.
- The payment terms registry's own withdraw or end operations and the draft review dead end, beyond the references above.
- Whether lapse of an open change is written back to the service record or only observed on read.

# Review

Independent design review (role-based AI reviewer separate from the writer, by reading, nothing executed), 2026-11-24: FAIL with two findings — the document said an open change ends when the validity period runs out, although a change that is requested and not verified ends only at its effective time; and it omitted that the proposer and confirmer of confirmed rules are rechecked at every use, so that either leaving stops every protected step. Corrected with the reviewer's wording and references, together with its additions: channels with an unknown outcome, the module's lack of authority rules, the effect of retiring a channel, the record limit, the weakness of one test assertion and an eighth option. Judged to hold: the other facts read from code, the test's assertions on the open change and on the rules, the description of the options and the labelling of the recommendation. Not verified by the reviewer: the references to the HYB-341 and HYB-344 records and the statement about a restart. Corrective re-review of the two findings: PASS, with no false statement introduced. Still not verified: whether issuance or consumption reads also fail on the rules recheck, and that no code path removes ended changes or stuck channels from the retained records.
