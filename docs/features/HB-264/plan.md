---
type: ImplementationPlan
title: Live M01 user administration screen in the console service mode, bound to the HB-263 API
description: Phase 3c of the pre-launch plan (HB-257). In service mode M01 renders a live screen backed by the identity read, the identity commands and the one-time link pickup. The unconnected preview stays for the review shell.
status: stable
---

# Scope and ownership

[HB-264](https://linear.app/hybrid-billing/issue/HB-264) binds M01 to the [HB-263](../HB-263/plan.md) API. Base: main `95eb7da`.

**Ownership**
- Root (Claude Code) owns the plan, implementation, tests and browser QA.
- The `ui-designer` role (AI, read-only) leads the visual design and compares the final renders.
- A separate read-only reviewer reviews the change.
- Machine: mac_mini.

**Allowed files**
- `apps/console/src/service/*`
- `apps/console/src/modules/identity/pages/users/*` (new live files beside the preview)
- `apps/console/src/app/screen-groups/identity.tsx`
- `apps/console/package.json` (the test list only)
- `docs/features/HB-264/*`

**Out of scope**
- Other screens.
- API changes. If one is needed, it becomes a separate ticket.

# Integration

**Service context.** `ServiceRoot` wraps the shell in a `ServiceContext` provider that carries the API client and the current session. Outside service mode the context is absent and every screen renders exactly as before.

**Selecting the screen.** `screen-groups/identity.tsx` renders `LiveUsers` for M01 when the context is present. Every other case renders the existing preview. The `data-security-phase` attribute is `live` for the live screen.

**API client** (`service/admin.ts`)
- `users(after, limit)`
- `command(id, target, expected, rationale, body)`
- `collect(ref)`

**Request rules**
- Every request sends the session's CSRF header and `X-Request-Id`.
- Each command gets a fresh idempotency key and request ID per user confirmation. A retry of the same confirmation reuses them.

**Errors.** Error bodies map to a closed set: `conflict`, `refused` (`prerequisite_failed`), `stage`, `invalid` (with fields), `denied`, `unauthenticated`, `limited` (with `Retry-After`), `unavailable`, and `partial` (`internal_error` with `operation_id`).

**State ownership**
- The screen owns the list page, the cursor stack, the selected user ID, the open action and the request sequence number.
- A response from an older request sequence is ignored.
- The selection is kept by user ID across reloads. If that user is no longer in the list, the panel shows the "select a user" prompt.
- After any command result, success or failure, the list is reloaded so the next command uses the current revision.

**Links**
- A `link_ref` from a command is collected immediately.
- The URL is held only in component state and shown once, with a copy button and the expiry time. It is never written to storage, the URL bar or logs.
- Closing the panel discards the URL.

**Tabs** (the shell's existing `view` keys)
- `users`: all users.
- `invitations`: the create form, and users with a pending invitation or a set link.
- `membership`: the role and membership state.
- `revocation`: a fixed "not available in pre-launch" state.

# External references (consulted 2026-11-16)

| Source | Observed | Adopted | Not adopted |
| --- | --- | --- | --- |
| Microsoft Learn, "How to create, invite, and delete users in Microsoft Entra ID" | "New user" sits above the All users list; the user is opened from the list, and the actions (delete, resend invitation) are in the user's details; the invitation state is shown in the details | Create user above the list; the login opens the details; actions and the invitation state live in the details | Emailed invitations: no message permission in pre-launch |
| GOV.UK Design System, Summary list | Key facts as rows; row actions at the row end with visually hidden context; not for tabular data | A facts list in the detail panel; hidden context on the row's open button | Per-row change links: actions are grouped below the facts |
| GitHub Docs, "Removing a member from your organization" | Row checkboxes, a bulk action menu, and a review step listing the affected members | A review step (confirmation) before each change | Bulk selection: bulk revocation is out of scope |

# Design

**Source.** The `ui-designer` role (AI, read-only) produced the specification on 2026-11-16. It worked from the current renders, the constraints, the references and the API contract. The spec covers:
- **Composition.** Toolbar, then list, then a detail panel at 1280px and wider; below that, a stepped detail.
- **Table.** The login is an underlined button that opens the details; the selected row gets a fill and `aria-current`.
- **Detail panel.** A facts list, then account, membership and sign-in action groups.
- **Disallowed actions.** Hidden and replaced by a reason line, never shown disabled.
- **Confirmation.** A step with impact text, a role choice where relevant, and a required reason of up to 256 characters.
- **Link pickup.** Shown once, with a copy button and the expiry time.
- **Revocation tab.** A "not available in pre-launch" state.
- **States.** Every state is in words; there are no coloured pills.
- **Copy.** EN, KO and AR, with a map onto the no-coloured-border rule.

**Root's decisions on the spec**
- **Impact copy, corrected against the server.**
  - Suspending an account signs the user out.
  - A reset link ends sessions only once it is redeemed.
  - Revoking an invitation leaves the set link working.
  - An invitation is accepted at the user's next sign-in.
  - A new set link replaces the earlier one.
- **Hidden where the server would refuse.** A reset link for a suspended user, and an invitation while one is pending.
- **Stepped layout from 1024 to 1279px,** the same as mobile.
- **Stacked rows below 768px,** using the table markup with a visually hidden header.
- **Paging.** 100 users per page, with "Show more users" when `next` is present.
- **Copy review.** All copy is a draft for the owner's language and terminology review.

# Verification plan

**Unit tests (`tsx --test`)**
- Error mapping.
- The request body shape.
- Fresh and reused idempotency keys.
- The late-response guard.
- Link handling: never persisted, and discarded on close.

**Browser QA** against the production build, with `apps/api` running the sample seed and Vite preview proxying `/api/v1`. Cases:
- EN, KO and AR at 1440 and 390;
- the users table, the detail panel, each action's confirmation;
- create-user with link pickup and copy, sign-in of the created user;
- conflict (a stale revision after another session's change);
- protection refusal (a reset link for an administrator), the revocation tab, empty and error states, keyboard flow;
- computed borders, checked against the no-coloured-border rule.

**Design comparison** of the final renders by the `ui-designer` role. Then an independent change review.

**Checks:** `npm run build`, the console unit test suite.

# Review history
