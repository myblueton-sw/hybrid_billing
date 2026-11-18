---
type: Reference
title: HYB-299 external references
description: External professional references for M01 role validity periods and bulk revocation, opened by root on 2026-11-18, with what was observed, adopted and rejected.
status: stable
tags: [HYB-299, M01, references]
sources:
  - resource: plan.md
---

# HYB-299 external references (opened by root, 2026-11-18)

## 1. Microsoft Learn, Assign Microsoft Entra roles in PIM

https://learn.microsoft.com/en-us/entra/id-governance/privileged-identity-management/pim-how-to-add-role-to-user (opened)

**Observed:**
- An assignment is either permanent (no expiration date) or time-bound with a start and an end date and time.
- An assignment cannot last less than five minutes.
- At expiry, a time-bound Global Administrator assignment is not removed if it is the last one, and the last Global Administrator assignment cannot be removed. The page gives this reason: "to minimize the risk of administrators locking themselves out of the tenant by accident".
- An emergency access account with a permanent assignment is recommended.

**Adopted:**
- An optional end date. Omitted means no end, subject to the setter's cap.
- The anchor rule. At least one active administrator without an end must remain, so ends and removals that would strand administration are refused.

**Rejected:**
- Eligible and activation-based assignments. M01 has assigned roles only.
- Scheduled future starts (owner question Q9 defaults to none).
- Silently keeping an expired last administrator. HYB-299 refuses the change up front instead.

## 2. GitHub Docs, Removing a member from your organization

https://docs.github.com/en/organizations/managing-membership-in-your-organization/removing-a-member-from-your-organization (opened)

**Observed:**
- Members are selected with checkboxes. A "members selected" action menu offers removal.
- The administrator reviews the members who will be removed, then confirms.
- The article lists the side effects:
  - pending invitations the member sent are cancelled;
  - access to private forks is lost;
  - paid seats are not reduced automatically;
  - membership data is kept for three months.
- When membership is managed by an identity provider, removal happens there.

**Adopted:**
- Selection by checkbox with a visible selected count and one bulk action.
- A review step that lists exactly who is affected before confirmation.
- Side effects such as cancelled invitations and ended sessions are stated in the preview.

**Rejected:**
- Removal as a bulk effect. HYB-299 offers suspensions only (Q4).
- Restoring membership data on re-invitation.

## 3. Google Workspace Admin Help, Suspend a user temporarily

https://knowledge.workspace.google.com/admin/users/suspend-a-user-temporarily (opened; redirected from support.google.com/a/answer/33312)

**Observed:**
- A single user is suspended from the user list or the account page with a final Suspend click.
- Several users are suspended through a CSV upload. Guests cannot be suspended in bulk.
- Suspension blocks the user's access to services without deleting data. Shared documents stay shared, and an existing chat session can continue until it ends.
- Suspended accounts are still billed. The account can be restored at any time.

**Adopted:**
- Suspension as the default bulk effect, keeping data.
- The preview states what stops (sign-in, sessions, step-up evidence) and what does not (shared and owned data).
- Restoring access is an explicit, separate increase.

**Rejected:**
- A CSV upload path. Selection is in the screen and staged through the API (plan transport).
- Letting existing sessions continue. HYB-299 ends sessions immediately.

## 4. Carbon Design System, Data table usage

https://carbondesignsystem.com/components/data-table/usage/ (opened)

**Observed:**
- Selecting rows shows a batch action bar at the top of the table with actions that apply to every selected item.
- Row actions are disabled while batch mode is active.
- Cancel or deselecting all leaves batch mode.
- The header checkbox has checked, unchecked and indeterminate states.
- The page does not say whether select-all covers the current page or all rows, and gives no guidance on a selected count or on confirming destructive batch actions.

**Adopted:**
- A selection bar with the count and the bulk action next to it.
- The header checkbox selects the visible rows, with an indeterminate state.
- Row actions are disabled while the bulk flow is open.

**Rejected:**
- A coloured batch toolbar.
- Select-all across pages (filter-wide selection waits for HYB-294).

## 5. GOV.UK Design System, Check answers pattern

https://design-system.service.gov.uk/patterns/check-answers/ (opened)

**Observed:**
- Before submitting, users review their answers in summary lists grouped under headings.
- Skipped optional answers read "Not provided".
- The submit button states the action it performs.

**Adopted:**
- The preview step lists the facts and the people affected before confirmation.
- Missing facts read in words, not blank.
- The confirm button names the action and the count.

**Rejected:**
- Per-row "Change" links. Changes go back to the selection step.

## 6. GOV.UK Design System, Checkboxes

https://design-system.service.gov.uk/components/checkboxes/ (opened)

**Observed:**
- Checkboxes sit in a fieldset with a legend, the box to the left of its label (right in RTL), with item hints under the label.
- Error messages go after the legend and hint.
- Options are never pre-selected, and users are told how many they can select.
- The page does not discuss hit area.

**Adopted:**
- The effect radios and any checkbox groups follow this layout.
- The selection limit is stated next to the count.
- Nothing is pre-selected.

**Rejected:**
- The GOV.UK checkbox styling. The native control and design-system tokens are kept.
