---
type: ImplementationPlan
title: Live M03 access screen in the console service mode, bound to the HB-266 access API
description: Phase 3f of the pre-launch plan (HB-257). In service mode M03 renders a live screen with a grants view (grants to people, the console ceiling, delegations, a grant form and revoke steps), an effects view (per-field effect preview), and roles and support views marked as not available in pre-launch.
status: final
---

# Scope and ownership

[HB-267](https://linear.app/hybrid-billing/issue/HB-267) binds M03 to the [HB-266](../HB-266/plan.md) API. Base: main `bbd2f22`.

**Ownership**
- Root (Claude Code) owns the plan, implementation, tests and browser QA.
- The `ui-designer` role (AI, read-only) leads the visual design and compares the final renders.
- A separate read-only reviewer reviews the change.
- Machine: mac_mini.

**Allowed files**
- `apps/console/src/service/admin.ts` and its test.
- `apps/console/src/modules/identity/pages/access/*`, which are new live files beside the preview.
- `apps/console/src/modules/identity/pages/users/live.css`, for shared styles.
- `apps/console/src/app/screen-groups/identity.tsx`.
- `apps/console/package.json`, test list only.
- `docs/features/HB-267/*`.

# External references (consulted 2026-11-16)

| Source | Observed | Adopted | Not adopted |
| --- | --- | --- | --- |
| Microsoft Learn, "List Microsoft Entra role assignments" (https://learn.microsoft.com/en-us/entra/identity/role-based-access-control/view-assignments) | Assignments list a principal, a role and a scope. In privileged identity management, eligible, active and expired assignments are shown apart, and direct and inherited assignments are distinguished. | Grants listed by person, with access, scope and state in words. Ended grants are kept apart from active ones. Delegated access is listed separately from direct grants. | Group-based assignment; bulk download. |
| Google Cloud, "Troubleshoot IAM permissions" (https://docs.cloud.google.com/policy-intelligence/docs/troubleshoot-access) | The inputs are a principal, a resource and a permission. The output is an overall access state with the granting or non-matching policies. Caveats list what the check does not evaluate. | The effects view takes a person, data and an optional customer. It returns one allowed or refused result per action and field, with a plain reason and the delegation chain. A standing "this is a preview" note says what is not evaluated (the console's own permission and a current sign-in). | Condition context, relevance ranking. |

# Design

The visual design follows the `ui-designer` specification of 2026-11-16 and reuses the M01 and M02 live patterns. It covers:
- the wide grid with a grant form in the side panel, and a stepped layout below 1280px;
- three sections in the grants view (grants to people, console ceiling, delegations), with ended items inside a disclosure;
- state shown in words and weight, with no colour;
- a revoke step that needs a reason;
- an effects view with a standing preview note, whose result is cleared whenever an input changes;
- roles and support views marked "not available in pre-launch";
- EN, KO and AR draft copy, pending the owner's language and terminology review.

# Verification plan

- Unit tests for the client and the model: state ordering, day limits, reason labels.
- Browser QA on the production build against a live API with the sample seed:
  - grant a member, revoke, the self-grant refusal, the effects preview before and after a grant;
  - ended grants, the roles and support views, the denied state;
  - EN, KO and AR at 1440 and 390;
  - focus, and computed borders.
- Design comparison by `ui-designer`, then the independent change review.

# Review history
