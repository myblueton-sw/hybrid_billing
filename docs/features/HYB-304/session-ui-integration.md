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

# Bounded account response contract for review

Replace `/me`'s `sessions` and `session_ends` arrays with `current_session` (the existing queried-session row), `recorded_session_count` (exact retained canonical owner/epoch/current-workspace index length, including distinctly inactive/unavailable records) and `accepted_operation_count` (exact retained accepted operations in that same scope). These are recorded counts, not active-device counts or cleanup completion. Preserve the account, policy and time fields. Keep the existing 32-hex `sessions_generation` as an opaque legacy compatibility token obtained without a population snapshot; dedicated query pages retain their separate canonical generation. New UI termination uses selection references and does not construct the legacy browser manifest.

Authorize the canonical owner before counts. Read index length/version and compatibility token under the session lock; use the accepted operation owner index under the domain lock. No full session copy or expanded operation construction occurs in `/me`. The current-session row remains independently available when query filters/pages omit it. A count grants no object access. Legacy request compatibility remains capped and current authority is rechecked at acceptance.

Update SelfPage strict parsing, account/password/history consumers, test fixtures and OpenAPI together with the rendered session consumer. Replace legacy tests' use of `/me` population arrays with explicit dedicated session/operation read helpers; keep existing effect/retry/retention assertions and add a real `/me` bounded-response regression with many sessions/large operation results. Test-only reconstruction must not enter product code. This response change cannot be merged separately while the rendered consumer still expects the removed arrays.

# One-session uncertainty and shared reference proposal

The new UI should use a server-owned selection reference for a single known session as well as for all other sessions. Keeping the legacy one-handle command as the only single-session UI submission loses the operation identity when its response is lost. Avoid reconstructing an operation or guessing a digest from internal implementation details.

Extend preparation with the mutually exclusive strict body `{session: handle}` alongside `{}` and `{selection, after}`. After current canonical admission, resolve that known handle through the existing owner index; missing/foreign is404, current caller is422. Store exactly that handle/private verifier and minimal preview row, with kind `one`, canonical source generation, original caller and the same expiry/limits. A one-target record seals immediately, charges one record plus one target, and can be reused only for the identical live scoped target/source. It shares the per-caller pending and global limits; it cannot create an uncharged alternate path. The existing all-other route continues to build boundedly with kind `others`. Return the explicit kind in preparation/preview.

The selected termination body remains `{selection,digest,count}`. Fingerprint the stored kind and immutable targets; use that kind for application and operation results. For kind `one`, later unrelated sessions are not not-included results, matching the existing one-handle semantics. Recheck original caller/current authority/expiry at effects and replay the same accepted reference before prep expiry/live generation comparison. Keep both legacy request variants supported for API compatibility, but the new UI does not use them. Single-session uncertainty now follows the identical reference read-back/replay flow and cannot widen its target.

Keep submission uncertainty in a small in-memory store keyed by the stable authenticated AdminClient object plus canonical user/workspace/current-session handle. It survives component, view and locale remounts within that authenticated client; a new client/login is isolated. Store only reference/fingerprint/operation/status, never cookies, CSRF values or source populations. Full-document reload and process restart are explicit retention limits; do not claim durable browser intent. A new preparation/termination is locked while submitting, unknown, recovering or unavailable. Accepted result state permits later independent actions. Stale async responses must not replace a later intent or scope. No automatic fresh preparation after an unknown result.

# Remaining original session-tag binding

The former full-population view tagged a currently present session as started after the latest confirmed other-session list. Keep that requirement under bounded queries; do not infer its absence from an unloaded not-included page. The initial bounded UI currently shows a separate paged not-included result view but has no cross-page row-tag producer yet. This remains unfinished before M05 integration.

Proposed binding for independent design review: add `not_included_operation` to queried-session rows (empty when no retained matching fact). Maintain a canonical owner-scoped ordered subset of accepted kind-others operations so lookup of the latest retained one is constant work. Its immutable heldPositions map tests the current handle without loading result rows. Publish/forget update this subset under the domain lock; this is bounded fixed index metadata per already-charged record, not a new uncharged target population. The current accepted-operation owner version must also bind session-query continuation, because a tag can change even when no live target was removed. Missing/foreign operations produce no tag; known-ID detail rechecks the current canonical scope. Use the same admitted current-session producer where appropriate.

The UI renders the localized not-included-in-confirmed-list tag only from this explicit returned operation reference, and can recover that accepted operation by its already scoped known-ID route rather than downloading all held rows. Preparation previews remain frozen observations and do not present a tag as a live eligibility check. Verify all-other acceptance/eviction with no session removal, many newer one-session operations, caller/epoch/workspace isolation, cursor invalidation and bounded row/byte behavior before claiming this legacy item preserved.

The independent design review rejected temporal wording for this producer: legacy explicit manifests may omit a session that already existed. The operation reference proves confirmed-set exclusion, not creation after the reference time. Use the same neutral exclusion wording for result summaries and row tags; no active-session claim is inferred from historical not-included records. This meaning correction preserves the visible exclusion fact without inventing temporal evidence.

# Inherited history browser bound

Independent completion audit found no accepted exception to the bounded-browser rule for access history. The baseline Show more appended every response and rendered the complete accumulated array. The bounded follow-up and actual failure/pass evidence are recorded in [UI progress](./ui-progress.md#bounded-inherited-history-follow-up). Preserve the existing API, period, admission, recording/retention notes and truthful failure behavior, but display one bounded history page and keep at most a bounded continuation trail. Expose navigation without inventing stronger snapshot guarantees than HYB-306 provides. Cover replacement, previous navigation, period reset, failure/retry and delayed prior-scope responses before full M05/UI-04 completion.
