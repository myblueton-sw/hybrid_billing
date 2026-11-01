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

# Boundaries and behavior

`assemble` validates explicit module/profile/extension descriptors, versions, dependencies, unique routes and slot ownership before rendering. Standard and Evidence share the sources module; Evidence contributes its own renderer at sources/evidence/v1. Both examples are intentionally bundled. This is not plugin isolation, tenant authorization or a private-customer distribution solution.

The default adapter is unconnected. Opt-in synthetic mode has 57 records with bounded ten-row pages, search, missing/ready filters, detail and selection. Query scope changes invalidate selection/detail and abort old requests; an epoch also rejects late results. Hash history restores only explicit synthetic fixture queries. No real customer query may reuse this URL policy without a separate data review. Error, empty and loading states are distinct.

Both screens use canonical local tokens/logo/fonts. Review evidence covers immutable source/dependency fingerprints, locale, width, profile/mode, visible state (including unsubmitted input), items and explicit acknowledgment. Scope changes cannot reuse FIX. Exported variants describe non-sensitive state while the scope hash binds the exact state. Corruption, quota and read-back failures remain visible. Browser records are local evidence, not project acceptance; authenticated approval storage is deferred.

# Screen items

| Screen / item suffix | Location and expected behavior | Status |
| --- | --- | --- |
| contents:table:implementation | Main index distinguishes 2 implemented screens and 112 planned screens | Implemented |
| contents:action:open-intake | Open intake using current profile/mode/locale | Implemented |
| index-intake:field:scope | Sample workspace picker invalidates result selection/detail | Synthetic |
| index-intake:field:filter | Ready/missing/all filter resets page | Synthetic |
| index-intake:field:search | Fixture ID input; unsubmitted edits invalidate review scope | Synthetic |
| index-intake:action:search | Submit bounded fixture query into hash history | Synthetic |
| index-intake:table:records | At most ten rows, exact fixture total, stable IDs | Synthetic |
| index-intake:state:feedback | Loading, empty and error feedback without fabricated zero | Implemented |
| index-intake:state:unconnected | Default explicit no-backend message | Implemented |
| index-intake:field:scenario | Explicit synthetic normal/empty/error demonstration | Synthetic |
| index-intake:action:page | Previous/next with limits | Synthetic |
| index-intake:action:select | Select visible records only; no financial action | Synthetic |
| index-intake:action:inspect | Open inline record detail, owned by this screen review scope | Synthetic |
| index-intake:action:close-detail | Close visible detail | Implemented |
| index-intake:action:retry | Retry only when the request failed | Synthetic |
| index-intake:extension:evidence-guide | Evidence-only contribution, no source approval | Implemented |

IDs remain stable on reorder. Screen IDs retain the prior design route identity; this increment adds scoped item IDs and does not retire other baseline items. Review controls and shell choices are shared dependencies of both screens. Their changes affect both fingerprints. A no-impact dependency change still needs an impact assessment before reconciling prior acceptance.

# Deferred work

Production adapters, authenticated tenant/permission enforcement, remaining 110 React screens, real customer installation packaging, durable owner acceptance, native-language review and Windows/Linux execution remain outside this increment. No payment, deployment or merge permission is implied. Operational commands and licensing are in [console README](../../../apps/console/README.md).
