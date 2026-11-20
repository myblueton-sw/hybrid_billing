---
type: ImplementationPlan
title: HYB-304 M05 bounded session UI integration
description: Defines the remaining session query, immutable confirmation and operation recovery consumer work without declaring visual acceptance.
status: draft
sources:
  - resource: ./plan.md
  - resource: ./selection-integration.md
  - resource: ../HYB-306/visual-spec.md
  - resource: https://support.google.com/accounts/answer/3067630?hl=en
  - resource: https://support.microsoft.com/en-us/accounts-billing/work-school/view-your-work-or-school-account-sign-in-activity-from-my-sign-ins
---

# Ownership and design boundary

Root owns source, integration and verification on machine mac_mini. Existing M05 account/password/history sections and neutral full-width structure remain the visual baseline. New session controls, preparation/confirmation and paged operation details are changed M05 candidate items; no previous owner FIX is carried forward. Separate source/security and TW/render review remain required.

The actual Claude CLI ui-designer invocation returned a weekly usage limit, resetting November 24 at 07:00 Asia/Seoul. No Claude design or rendered review succeeded for this change. The root-authored proposal below is an implementation plan, not a substitute claim of completed Claude leadership or owner acceptance. Continue independent contract/state work while that limitation is recorded.

# External pattern evidence

Root opened the official Google Account device/session help and Microsoft My Sign-ins help on 2026-11-20. Google distinguishes sessions from physical devices and exposes a separate detail/sign-out action. Adopt explicit session identity and current/signed-out state; reject inferred device counts or locations because this query has no such producers. Microsoft separates recent sign-in activity and expandable evidence. Retain HYB-306's dedicated history region, and page operation details independently of the session table. These references inform hierarchy and terminology; no asset, code or third-party styling is copied.

# Session list and navigation

The session view consumes `/me/sessions` rather than accumulating `/me` rows. Keep one bounded page and a bounded continuation history; query parameters restore normalized filters and page size on browser navigation. Cursor invalidation restarts from the first page with an explicit message. An obsolete request cannot overwrite a later query or canonical user/workspace/caller scope.

Show current workspace as read-only scope. Provide current/other/all, search, since/until, apply, clear, applied-filter removal and page size. Preserve nanosecond timestamp values in serialized filters; never round a supplied API timestamp through JavaScript Date. Empty dates mean no bound; invalid date/search input is explained before sending. Use the canonical stable order and server-supplied reference time. Exact/lower-bound counts and scan-stopped state remain distinct from page length and zero. Current-session row and known-ID detail remain available independently of the selected page.

# Immutable confirmation and uncertain results

End-all-others prepares the server-owned set independently of the displayed filters/page. While building, expose observed progress and continuation; no final count or percentage. A sealed count/digest and paged immutable preview describe exactly the confirmed recorded set. Never collect all handles in browser memory. Explain that later arrivals are excluded and missing targets are reported separately. Current session is never part of this action.

Retain one submitted reference/digest/count across transport uncertainty. Disable new preparation/another termination while its outcome is unknown. Read back the selection reference/operation; if an operation exists, recover it rather than creating a replacement. Explicit replay sends the identical fingerprint and does not construct a fresh set. A missing/evicted reference means result unavailable, never zero or permission to repeat effects. Scope loss clears sensitive pages but does not assert success or undo effects. Confirmation and read-back responses reject stale request identities.

# Results and layout

The operation region pages accepted summaries; each summary uses recorded/ended/gone/not-included counts and independent cleanup status. Unconfirmed fields stay unavailable. Expanded results and not-included rows are separately bounded pages. Read-back from another current session of the same owner uses accepted operation IDs; original-caller selection mutation restrictions still apply. Manual refresh and bounded polling preserve request identity and focus.

Use the existing full-width service grid, typography, square controls and neutral separators. At1440/1280, group filters into compact labeled rows above the table; at 1024 allow wrapping; at 390 stack fields/actions and existing session cards. Do not stretch each field to the viewport or introduce a centered fixed-width page. Preserve Arabic RTL and isolate handles/dates LTR. No decorative colored edge or persistent colored selection outline. Keyboard order follows scope, filters, results, paging and end action. Modal focus enters its title, remains contained, and returns to the invoker or results heading after resolution. Unknown/pending/error states provide explicit status text and safe next actions.

# Exit criteria

M05 remains incomplete until `/me` no longer emits full session populations or expanded operation arrays, the actual screen consumes the dedicated query/selection/result routes, and meaningful source/state/transport/authorization/browser-history regressions pass. Verify EN/KO/AR at 1440/1280/1024/390 with actual API fixtures, late responses, failure/retry, unknown-result locking, keyboard/focus, local fonts, computed neutral borders and containment. Native-language, full assistive-technology and owner review remain separate. No partial foundation merge closes HYB-304 or satisfies UI-04.
