---
type: Reference
title: HYB-297 external references
description: External professional references for the M02 trust change workflow layout, opened by root on 2026-11-18, with what was observed, adopted and rejected.
status: final
tags: [HYB-297, M02, references]
sources:
  - resource: visual-spec.md
---

# HYB-297 external references (opened by root, 2026-11-18)

1. **GitHub Docs, About protected branches.** https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches (opened)
   - **Observed:**
     - With "Dismiss stale pull request approvals when new commits are pushed", an approval is dismissed when the diff changes from what the reviewer approved, and the pull request cannot merge until someone approves again.
     - "Require approval of the most recent reviewable push" needs one approval from someone other than the last pusher.
     - Administrators can bypass the rules unless bypassing is disallowed.
     - The page does not state a general rule against self-approval.
   - **Adopt:**
     - A change in evidence or a revision clears approval, and the screen says that approval is needed again.
     - The approver must be a different person from the proposer.
   - **Reject:**
     - Any administrator bypass; trust rights are separate from installation administration.
     - Coloured check badges.

2. **Microsoft Learn, Approve or deny requests for Microsoft Entra roles in PIM.** https://learn.microsoft.com/en-us/entra/id-governance/privileged-identity-management/pim-approval-workflow (opened)
   - **Observed:**
     - Approvers cannot approve their own requests.
     - Approving or denying opens a page where the approver enters a business justification before submitting.
     - The first approver's decision resolves the request.
     - Administrators can cancel pending requests.
   - **Adopt:**
     - A rationale is required in every confirmation.
     - Self-approval is refused.
     - One decision resolves a step.
   - **Reject:**
     - A fixed 24-hour expiry, because M02 has none.
     - Email notification, because nothing in M02 sends messages.

3. **HashiCorp Terraform, Command: plan.** https://developer.hashicorp.com/terraform/cli/commands/plan (opened)
   - **Observed:**
     - `plan` previews changes without carrying them out.
     - A saved plan can later be passed to `apply`.
     - Changes made after a plan can make the result differ from what the plan showed, so the page advises re-checking before applying.
     - The page does not say what `apply` does with a saved plan whose state has since changed.
   - **Adopt:**
     - Validation is a step separate from activation.
     - Activation is bound to the approved digest.
     - Activation is refused when the base or the evidence changed, instead of applying a stale review.
   - **Reject:**
     - A coloured +/- diff.
     - Any automatic apply.
