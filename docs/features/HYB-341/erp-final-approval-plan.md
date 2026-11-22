---
type: ImplementationPlan
title: Governed ERP final-amount approval using current owned results
description: Separate ERP-final governance and authority from charge approval while binding exact current result evidence.
status: draft
---

# Original outcome and ownership

HYB-341, machine:mac_mini, continues 096ac90 and the original W-23 scope. Root owns implementation and integration. refusal_review provides independent financial/security review; platform_refusals reviews terminology/evidence. The existing approvals registry remains the only financial approval store. No external ERP command, approval consumption, issuance, delivery, merge or production authorization is implied by policy configuration.

# Observed gap and sequence

The module already supports StageERPFinal, but service policy, amount authority, direct-human permission and request permission are currently bound to charge_approval. The seeded ERP-final policy is demonstrative, not service-owned governance evidence. Reusing charge grants or adopting the seed would silently widen authority. Implement separate ERP-final policy governance, separate amount limits, dedicated direct-human approval and console/human request domains, current subject production, authenticated review/request/decision, and downstream revalidation. Existing charge clients and historical receipts must retain their current meaning.

# First policy prerequisite

Extract a private closed stage configuration selecting only charge_approval or erp_final_approval and separate service-owned policy binding maps. Preserve existing financial_policy command names, fields, targets, receipt bytes and charge-only wrappers. Add separately named erp_final_policy.propose/confirm/inspect/current commands; callers cannot select a stage by injecting a body field. Reuse the same governors, independent proposer/confirmer checks, policy revision monotonicity, current authority, step-up, hold, output bounds and actual approvals module. Initial current-policy inspection may expose the seeded policy as current module configuration, but the owned-current-policy consumer must refuse it until an independently confirmed service-owned proposal exists.

Both stage maps share the actual module proposal-ID namespace: cross-stage collisions must refuse, never adopt another stage's proposal. Policy required digests remain explicit caller choices from the existing closed set, with result mandatory; configuring a policy does not establish that a future subject can supply all required evidence. Add exact source-rule exceptions only if a reviewed shared mutation helper moves; do not broadly bypass the financial confirmation guard. Pending ERP-final proposals must be counted distinctly in fallible impact enumeration with original-proposer current recheck. Product-facing impact wording and rendered checks follow the existing terminology/design rules.

# Subsequent subject boundary

A final subject must bind independent source/packet/proposal and confirmation evidence, complete application receipt and exact live ledger, current chain head and monotonic result watermark, governed document/preparation/mapping facts, upstream current charge approval and the owned ERP-final policy revision. Use a stable economic target across replacement revisions with revision-specific evidence digests. A replacement invalidates fresh final approval eligibility without deleting prior decisions.

The initial synthetic profile may explicitly define reported gross as exact net plus reported tax: fixture USD15 plus USD1 produces USD16, distinct from charge net. This is a reviewed test profile, not production tax/accounting policy or proof of external ERP issuance. Unsupported FX, adjustment, currency or tax semantics must remain unavailable. Authorship and contributor exclusion require independent resolution before subject/decision code; the requester cannot replace the original reported-result author or erase upstream capture authorship.

# Verification and boundaries

Policy acceptance covers unchanged charge wire/behavior, stage isolation, cross-stage ID collisions, unsupported stages, seed refusal by owned consumers, independent confirmation, stale/conflicting revisions, malformed fields, held and reauthentication paths, revoked proposer/caller before cached replay, capacity/output bounds and fallible impact. No policy action creates grants, amount authority, result application or external commands. Reuse actual registry/session fixtures and run relevant existing charge/source-rule regressions. This prerequisite does not substitute for complete StageERPFinal, issuance, delivery, persistence or the owner goal.

# Independent investigation

refusal_review inspected the existing service/module/planning evidence and recommended the sequence above. The subsequent independent prerequisite design review passed the closed-stage helper and separate ownership model before implementation. Independent source review of the implemented prerequisite also passed: stage isolation, shared-ID collision refusal, seed ownership refusal, pre-cache caller/proposer rechecks, exact confirmation guard exception and coordinated hold/step-up/impact behavior. The reviewer inspected, but did not execute, the scoped race log (PASS, 16.719s). This approval excludes full final-amount authority, subject production and decisions. Existing missing-companion-file waiver remains in force; no files are reconstructed.

# Current implementation verification

The scoped server race run covering ERP-final policy, charge policy, financial confirmation, financial subjects, source rules, hold and step-up passed in 16.719s. An initial guard run exposed the explicit command-count expectation still set to 139; it was corrected to 141 for the two new mutations while preserving explicit row checks. Eight M01 preview render unit tests passed, including distinct final/charge policy labels and counts in EN/KO/AR. Independent source/domain terminology review passed the new labels, without native-language certification.

Actual browser verification is tracked separately. Its first fixture attempt assumed charge policy revision 2, but the existing result fixture already governs that stage; the fixture now reads each current stage revision and proposes its successor. A subsequent compile check caught an integer formatter mismatch, corrected to strconv.Itoa. Neither failure changed product policy or weakened revision checks. This remains an uncommitted prerequisite increment; no final financial approval or completion claim is made.

The corrected actual M01 browser run passed all 18 EN/KO/AR × 1440/1024/390 × reported-recheck variants, with separately asserted final/charge policy counts, no local clipping or page overflow, correct RTL and no execution requests or page runtime errors. Captures and machine evidence are retained outside Git at /private/tmp/hyb341-final-policy-impact. Screenshot-based independent visual review remains pending; browser assertions alone do not certify appearance.

# Next amount-authority prerequisite: independently reviewed design

The next change uses separate ERP-final amount-authority bindings in the existing approvals registry, closed stage helpers and erp_final_authority.propose/confirm/revoke/inspect. Existing charge commands, wrappers and financial-limit- IDs retain their meaning; ERP-final authorities use erp-final-limit- IDs. Reject shared proposal-ID and derived authority-ID collisions before mutation. Preserve canonical exact limits, currency, finite date ranges, current human recipients, independent proposer/confirmer/recipient exclusion, monotonic AuthorityAt status, capacity and output preflight. Confirmations recheck current caller, original proposer and recipient before cached replay. Revocation requires the current governor and owned authority, not continuing eligibility of the original proposer or recipient. Hold permits authority reduction; escalation and reauthentication remain explicit. Pending impact stays separate. No gate decision/request grants are created.

refusal_review independently passed this design by source inspection. The subsequent final-decision consumer must additionally prove that one service-owned confirmed, unrevoked authority covers the entire amount for the exact actor, stage, currency and date. Current module covering checks do not establish service ownership; finding a small owned authority while relying on a larger unowned seed would be unsafe. Keep exact numeric coverage in a narrowly reviewed approvals-module API rather than duplicating adapter money arithmetic. Apply ownership coverage at approval and downstream revalidation, preserving rejection semantics. This requirement is not fulfilled by policy or amount-authority provisioning alone.

Acceptance includes stage isolation, both collision directions, unknown stages, canonical/date boundaries, all three actor exclusions, identical-key authority-loss refusal, revocation after proposer/recipient loss, monotonic status, no gate grants, output/capacity atomicity and separate fail-closed impact. Root remains sole writer; independent reviewer remains read-only. Amount-authority implementation has not started at this checkpoint.

# Rendered review and corrective evidence

The actual Claude CLI ui-designer invocation completed with a bounded initial visual PASS for EN1440/KO390/AR390, but the charge-policy row was offscreen. Root added separate policy-row crops and explicit per-row count, overflow, direction and computed-border evidence. The repeated 18-scenario browser run passed in 14.689s at /private/tmp/hyb341-final-policy-impact-reviewed. The actual corrective Claude CLI review passed the six final/charge EN1440/KO390/AR390 crops. It identified two non-blocking evidence improvements: use unequal policy counts and include the scenario recheck flag in each policy evidence record. Both were implemented in the browser fixture/tool; the injection still only changes the unknown-application note, never policy semantics.

Independent technical-writing review also inspected all six row crops, glossary and plan and passed the bounded meaning/readability/RTL review. These reviews do not certify native-language, owner acceptance, full M01, keyboard focus or every viewport image. Visual CLI transcripts remain outside Git at /private/tmp/hyb341-final-policy-visual-review.txt and /private/tmp/hyb341-final-policy-visual-followup.txt. Console TypeScript/build passed; existing bundle-size/configuration warnings remain. Additional same-key module-governor removal and reverse cross-stage ID collision tests passed with the race detector in 1.893s. No additional product changes followed the independent source review.

Final corrective browser assertions passed in 21.537s across all 18 scenarios with 36 policy records and unequal charge=2/final=1 counts; each record now identifies its reported-recheck scenario. Evidence is /private/tmp/hyb341-final-policy-impact-distinct. The full server package suite passed in 199.975s at the policy implementation checkpoint; the additional two race regression tests and browser fixture refinements were checked separately. No CI, merge or deployment success is implied.
