---
type: ImplementationPlan
title: Candidate-backed requests in the B09 approval inbox
description: A bounded historical projection with truthful amount basis and separate original author and requester.
status: draft
---

# Scope and ownership

Continue HYB-340 after 76b6805, machine:mac_mini. Root owns implementation; independent refusal_review reviewed the proposed projection and remains read-only. Technical-writing review owns terminology evidence, actual Claude CLI ui-designer reviews changed renders. Owned files: approvalrows/read tests, B09 contract/parser/copy/result details/fixtures/tests, scoped browser tooling and ticket evidence. B09's existing screen/item identity, full-width layout and review control remain. This increment connects actual records; approve/reject screen actions follow under the same full W-23 goal.

# Read and projection contract

Derive a bounded union of the existing approvalIndex and validated financialCharges directly under the existing domain lock. Do not create another retained index. The combined number of entries must not exceed approvalCap (1000), matching the single module request registry's capacity, and neither collection may exceed that bound. Scope entries to the authorized workspace, reject duplicate IDs and corrupt same-workspace bindings rather than silently omitting them, and deterministically sort all rows by ID. Keep existing B09 read permission, paging, snapshots and permission revision checks.

Use one shared historical state/decision projector. Existing demonstration gross rows retain their basis and producer_unbound explanation, including their blocked pending projection. Service-backed StageCharge rows use rated_candidate_total, never inferred net/tax/gross; project recorded Pending as pending. Their new explanation states that current prerequisites and permission were not rechecked for this historical read. ERP applicability, authority, expiry and eligibility remain unknown; historical approval grants no execution permission. PolicyAt continues to validate the exact historical single-approval revision.

Return original author and requester separately, recorded request digest through evidence.revision and exact policy_revision. Add requester and policy_revision to new server rows, including known module facts on existing samples. Frontend parsing accepts both the prior exact record shape and the new paired fields; unknown extra fields and unknown amount bases still refuse. Old records with neither new field remain readable and show unknown requester/policy rather than fabricated values. New fields must appear together and contain a valid principal and positive safe policy revision. This is an additive synthetic read contract, not a production version negotiation claim.

# UI and acceptance

Extend the closed amount-basis union/parser with rated_candidate_total. Use reviewed EN/KO/AR labels for rated candidate total, requester, recorded policy revision and the current-check warning. Display original author/requester and policy revision in the existing result details without reinterpreting permissions. read_only=true and commands=unconnected continue to describe this screen's action binding; actual authenticated command endpoints already exist. Keep action integration separately explicit.

Verify mixed sample/service rows and exact amount strings, different requester/original author, Pending/approved/rejected/superseded histories, duplicate/corrupt/bounded union refusal, foreign workspace exclusion, snapshot invalidation after requests/decisions, parser old/new compatibility and malformed paired fields. Run real B09 EN/KO/AR desktop/tablet/mobile rendering with service-bound synthetic records, including long IDs and unknown current prerequisites, then independent code/TW/Claude re-review. No owner FIX, main merge, production deployment or full W-23 completion is implied.

# Reviewed visual specification and references

Actual Claude CLI ui-designer inspected four earlier B09 baseline renders and the current details/copy/style rules before UI changes. It directed root to add requester immediately after author and recorded policy revision after updated-at within the existing facts grid; preserve layout/CSS and focus; show current-check uncertainty as a neutral paragraph before facts, not as a failed-validation blocker; and update the synthetic notice for mixed environment records. Independent terminology review supplied the final EN/KO/AR terms, overriding the designer's unreviewed translation candidates. The original warning remains a typed API explanation but is displayed separately from blockers. The B09 Screen help also changes examples to environment records.

Root rechecked [Microsoft approval request documentation](https://learn.microsoft.com/en-us/power-automate/approve-reject-requests) and [GitHub deployment review documentation](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/review-deployments). Adopt the inbox-to-detail pattern and explicit target/actor distinction. Reject their financial meaning, deployment effects, bypass paths, assets, colors and permission defaults. Claude relied on root's reference summary; it did not independently browse those pages. Its preliminary specification is /private/tmp/hyb340-projection-design-review.txt, not final-render acceptance.
