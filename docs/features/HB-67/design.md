---
type: Design
title: Modular console foundation and intake increment
description: Defines the implemented first React increment and its composition, review and data boundaries.
---

# Owner correction

The owner rejected the candidate's arbitrary IA/layout changes. This document describes an incomplete implementation, not an accepted replacement design. Internal app/modules/shared/extensions/profiles boundaries remain accepted; the existing HB-63 top navigation, header, secondary navigation and per-screen designs remain the UI contract. See [corrected verification outcome](verification.md). The implementation workspace and reduced navigation must not be treated as an approved product direction.

# Scope and ownership

[HB-67](https://linear.app/hybrid-billing/issue/HB-67/implement-modular-react-console-foundation-and-source-intake-slice) implements two React screens after the owner accepted the shallow architecture in [HB-66 PR 62](https://github.com/myblueton-sw/hybrid_billing/pull/62), commit c9c8a8be8faf01d491176829e0b10f3ca009b10b. Root owns implementation and QA; a separate read-only reviewer checks design and actual changes. Machine: mac_mini. This branch starts from main b48b23c; it does not merge pending design PRs.

The source intake and implementation index exercise app/modules/shared/extensions/profiles without creating empty future packages. The 112-screen plan is not represented as 112 completed React screens. HB-63 remains the baseline for unreplaced screens; no owner FIX or supersession is inferred.

# Corrected screen contract

The shared catalog and shell copy are derived from HB-63 commit 4d51763458b314fd766998c05fb32a1c40c99e72. The app restores preview notice, linked logo, account/jobs/locale controls, six ordered domains, searchable secondary navigation, breadcrumb/title/metadata, review before content and related links. All 112 route identities and registry links remain. Unported content is explicitly pending; a route inventory does not establish screen implementation. Customer navigation stays restricted to customer routes as a UI boundary only.

`contents` restores the 24-card design directory, search, area/review filters, counts and paging. `overview` reuses this directory as in the baseline. Intake restores source-context, search/provider/status/sort/refresh, table/cards/columns/reset/import, grouped page/all-result selection with exclusions and local selection review, original source/amount/status/evidence columns, and 25/50/100 page sizes. Amounts remain unavailable. Scope changes invalidate selection/detail and cancel stale requests. Expiration blocks results/actions until refresh; retry returns to the normal synthetic scenario.

The default adapter is unconnected. Opt-in synthetic mode uses 57 fixture records. Fixture queries alone may enter hash history. Import selects a local file but performs no upload or server validation. Header account selection is limited to two synthetic scopes and jobs remain an empty local boundary; the full baseline account picker/job implementation remains pending. Profile/mode demonstration is separate review tooling below the screen rather than a replacement header.

Catalog contracts/data live in shared, not app internals. Standard and Evidence reuse the same source module; the versioned evidence slot renders an actual extension component. This is not tenant authorization or private customer packaging.

Review fingerprints include source dependencies, shell account, locale/width, profile/mode and actual screen state. Same-tab save notifications refresh directory counts. Previous evidence is preserved; no owner FIX is inferred. Existing canonical implementation IDs are restored, including:

- `contents:field:design-directory-search`, `contents:field:design-directory-group`, `contents:field:design-directory-status`, paging and per-card navigation IDs.
- `index-intake:region:layout`, `index-intake:section:purpose`, `index-intake:field:record-search`, `index-intake:field:provider-filter`, `index-intake:field:status-filter`, `index-intake:field:sort`, `index-intake:field:page-size`.
- `index-intake:field:page-select`, `index-intake:field:row-selection`, `index-intake:action:all-select`, `index-intake:action:review-selection`, `index-intake:action:record-detail`, `index-intake:action:retry-load` and visible column data IDs.

The earlier candidate's conflicting item names are not the canonical design IDs. This correction changes both implemented screen fingerprints; no old FIX transfers. Dialogs belong to their owning screen and remain local synthetic behavior. Canonical visual rules and baseline CSS are retained; this correction does not establish pixel-exact parity of every unported state.

# Deferred work

Production adapters, authenticated tenant/permission enforcement, remaining distinct React screen implementations, real customer installation packaging, durable owner acceptance, native-language review and Windows/Linux execution remain outside this increment. No payment, deployment or merge permission is implied. Operational commands and licensing are in [console README](../../../apps/console/README.md).

# Owner-requested header control refinement

The owner identified that the header's plain contribution link, native language selector and temporary account buttons did not form a finished control design. The correction preserves logo geometry/clear space, IA order and canonical tokens while applying consistent 44px minimum controls, square borders, spacing and focus treatment. Language selection uses a styled listbox with keyboard navigation and Escape focus return rather than an OS-rendered popup. The account dialog adds search, current selection and empty feedback for the same two synthetic scopes. Closing clears the picker search. Mobile controls use two columns; Arabic remains RTL with isolated account IDs and unchanged LTR branding.

This refinement is authorized by the owner's header feedback and is distinct from restoring the prior baseline verbatim. It affects shared-shell fingerprints for both implemented screens. It does not finish production account search, job integration or unported screen bodies.
