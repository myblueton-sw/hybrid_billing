---
type: Design
title: Modular console foundation and intake increment
description: Defines the implemented first React increment and its composition, review and data boundaries.
---

# Owner correction

The owner rejected the candidate's arbitrary IA/layout changes. This document describes an incomplete implementation, not an accepted replacement design. Internal app/modules/shared/extensions/profiles boundaries remain accepted; the existing HB-63 top navigation, header, secondary navigation and per-screen designs remain the UI contract. See [corrected verification outcome](verification.md). The implementation workspace and reduced navigation must not be treated as an approved product direction.

# Scope and ownership

[HB-67](https://linear.app/hybrid-billing/issue/HB-67/implement-modular-react-console-foundation-and-source-intake-slice) implements seven distinct React screen bodies after the owner accepted the shallow architecture in [HB-66 PR 62](https://github.com/myblueton-sw/hybrid_billing/pull/62), commit c9c8a8be8faf01d491176829e0b10f3ca009b10b. Root owns implementation and QA; a separate read-only reviewer checks design and actual changes. Machine: mac_mini. This branch starts from main b48b23c; it does not merge pending design PRs.

The source intake and implementation index exercise app/modules/shared/extensions/profiles without creating empty future packages. The 112-screen plan is not represented as 112 completed React screens. HB-63 remains the baseline for unreplaced screens; no owner FIX or supersession is inferred.

# Corrected screen contract

The shared catalog and shell copy are derived from HB-63 commit 4d51763458b314fd766998c05fb32a1c40c99e72. The app restores preview notice, linked logo, account/jobs/locale controls, six ordered domains, searchable secondary navigation, breadcrumb/title/metadata, review before content and related links. All 112 route identities and registry links remain. Unported content is explicitly pending; a route inventory does not establish screen implementation. Customer navigation stays restricted to customer routes as a UI boundary only.

`contents` restores the 24-card design directory, search, area/review filters, counts and paging. `overview` reuses this directory as in the baseline. Intake restores source-context, search/provider/status/sort/refresh, table/cards/columns/reset/import, grouped page/all-result selection with exclusions and local selection review, original source/amount/status/evidence columns, and 25/50/100 page sizes. Amounts remain unavailable. Scope changes invalidate selection/detail and cancel stale requests. Expiration blocks results/actions until refresh; retry returns to the normal synthetic scenario.

The default preview uses visibly labeled synthetic mode with 57 fixture records. Explicit unconnected mode exposes no fixture work records. Fixture queries alone may enter hash history. Import selects a local file but performs no upload or server validation. Header account selection is limited to two synthetic scopes and jobs remain an empty local boundary; the full baseline account picker/job implementation remains pending. Profile/mode demonstration is separate review tooling below the screen rather than a replacement header.

Catalog contracts/data live in shared, not app internals. Standard and Evidence reuse the same source module; the versioned evidence slot renders an actual extension component. This is not tenant authorization or private customer packaging.

Review fingerprints include source dependencies, shell account, locale/width, profile/mode and actual screen state. Same-tab save notifications refresh directory counts. Previous evidence is preserved; no owner FIX is inferred. Existing canonical implementation IDs are restored, including:

- `contents:field:design-directory-search`, `contents:field:design-directory-group`, `contents:field:design-directory-status`, paging and per-card navigation IDs.
- `index-intake:region:layout`, `index-intake:section:purpose`, `index-intake:field:record-search`, `index-intake:field:provider-filter`, `index-intake:field:status-filter`, `index-intake:field:sort`, `index-intake:field:page-size`.
- `index-intake:field:page-select`, `index-intake:field:row-selection`, `index-intake:action:all-select`, `index-intake:action:review-selection`, `index-intake:action:record-detail`, `index-intake:action:retry-load` and visible column data IDs.

The earlier candidate's conflicting item names are not the canonical design IDs. This earlier correction changed the directory and intake screen fingerprints; no old FIX transfers. Dialogs belong to their owning screen and remain local synthetic behavior. Canonical visual rules and baseline CSS are retained; this correction does not establish pixel-exact parity of every unported state.

# Deferred work

Production adapters, authenticated tenant/permission enforcement, remaining distinct React screen implementations, real customer installation packaging, durable owner acceptance, native-language review and Windows/Linux execution remain outside this increment. No payment, deployment or merge permission is implied. Operational commands and licensing are in [console README](../../../apps/console/README.md).

# Owner-requested header control refinement

The owner identified that the header's plain contribution link, native language selector and temporary account buttons did not form a finished control design. The correction preserves logo geometry/clear space, IA order and canonical tokens while applying consistent 44px minimum controls, square borders, spacing and focus treatment. Language selection uses a styled listbox with keyboard navigation and Escape focus return rather than an OS-rendered popup. The account dialog adds search, current selection and empty feedback for the same two synthetic scopes. Closing clears the picker search. Mobile controls use two columns; Arabic remains RTL with isolated account IDs and unchanged LTR branding.

This refinement is authorized by the owner's header feedback and is distinct from restoring the prior baseline verbatim. It affects shared-shell fingerprints for every implemented screen. It does not finish production account search, job integration or unported screen bodies.

# Actual work-screen increment

The owner requested work screens rather than a directory/checklist. Default navigation now opens `operations`. Domain modules follow HB-66 ownership: operations owns `operations`, `core-index-close` and `my-tasks`; organizations owns `customer-detail`; issuance owns `review-issue`. The latter two declare their operations dependency. The shared queue retains each screen's canonical column contract rather than replacing the three workflows with directory cards.

Work queues provide period/provider/status/search filters, 25/50/100 paging, selection, empty/error/retry states and query-history restoration. Customer detail carries the exact synthetic case/account/period into overview, evidence and history tabs. Its evidence link clears conflicting intake filters and selects the corresponding source reference. Account changes reset scoped screen state and invalidate selected references/preparation; mismatched case or period produces an explicit missing-record boundary with no fallback customer.

Issuance exposes reviewer/issuer separation, source/recipient/payment prerequisite checks, revision and currency. Local preparation is invalidated by every relevant change, including changing back to an earlier value. Financial approval and issuance stay disabled; no monetary result is fabricated. Explicit unconnected mode disables work search and exposes no synthetic customer/issuance data. The preview banner distinguishes fixtures from a backend connection.

Baseline screen-layout styles, existing tokens, IA, canonical item IDs and review-before-content remain. Each screen fingerprints visible filters, selections, references and preparation state. This adds five work bodies to the existing directory/intake pair; it does not complete all 112 routes or establish owner FIX. Installation/start use the existing portable npm/Node commands without adding OS-specific product dependencies.

# Service-wide paired conception: owner correction

The owner rejected treating the directory/checklist or the operations slice as the service frontend, and explicitly requested a dedicated PM working with screen design and frontend development. This correction governs subsequent implementation; it does not claim that the current operations default has already changed.

Root remains accountable for scope, implementation and integration. Dedicated service_pm owns scope alignment; service_designer owns the visual/user-flow brief; service_frontend owns the implementation/state brief. Their initial work was read-only. They jointly confirmed that HB-66 assigns overview/contents/demo-scenarios to design-review, entry routes to identity and setup routes to onboarding. The existing app registers neither identity nor onboarding. Shrinking the review panel alone cannot resolve that product gap.

The complete service remains in scope: entry and identity, onboarding, operations, organizations, sources, analytics, pricing, calculation, issuance, delivery, history, customer portal, administration and Assistant. The 112-route catalog is traceability evidence, not a count of completed product screens. Existing IA, assets, tokens and shallow app/modules/shared/extensions/profiles structure remain the basis. Customer differences use composition and extensions rather than cloned screens.

## Proposed paired implementation order

1. Service entry and shared service experience: jointly specify entry-start, entry-login, entry-bootstrap, entry-invite, entry-challenge and entry-scope, using canonical designs and L01–L03 evidence. The designer defines screen-specific layout, hierarchy, actions and responsive states; frontend maps these to identity-owned screen components, app entry/workspace layouts and explicit state transitions. Preserve the existing six-domain workspace shell. Do not repurpose the design directory as a product home or make operations the universal entry.
2. Existing onboarding journey: setup-progress, setup-profile, setup-attribution, setup-connectors, setup-readtest, setup-schedule, setup-first and setup-readiness. Pair each screen with its actual next action, result and failure recovery.
3. Complete the remaining service journeys through the existing ownership map: sources and analytics; organizations and pricing; calculation, issuance, delivery and history; customer portal; administration and identity management; Assistant. These are coordinated work packages, not new menu groups or a reduced product scope. Existing operations work is assessed within this whole-service flow.

For each screen, the pair agrees on purpose, entry context, visible composition, principal action, destination, loading/empty/error/blocked states and adapter boundary before implementation. They then inspect the same rendered screen. PM verifies journey continuity and prevents technical traceability UI from replacing product UI. Per-screen review remains available, with exact ID/revision/status beside its compact control; item lists and evidence open on demand. No assistant creates owner FIX.

## Paired first-screen composition proposal

The designer and frontend specialist agree on this starting composition, derived from the access journey rather than owner FIX: entry-start presents the canonical logo, installation identity, actual environment status and language control; workspace sign-in is the principal action, with distinct invitation, first-administrator and recovery paths. The pre-workspace layout does not expose an authenticated business rail. Each access branch has its own fields and validation/recovery states. Entry-scope presents workspace, organization/legal entity and permitted role/scope with an explicit selection action and empty/expired/unavailable recovery. These fields require an authorized server response in production; local examples remain visibly synthetic. First setup proceeds to setup-progress; returning-user destinations require the role/context mapping described below.

## First package acceptance and unresolved boundaries

A returning internal user, invited customer and first-time administrator can each follow the defined local entry journey to the correct workspace/onboarding boundary. Keyboard, back/reload, mobile and EN/KO/AR states are checked on actual renders. A catalog page or generic form/table is not acceptance. Final landing-route mapping must be traced to the approved role/context evidence; registry references alone do not authorize an invented universal dashboard.

Local navigation and validation presentation may be implemented independently. Production authentication, bootstrap/invitation verification, workspace authorization, session persistence/revocation and protected deep-link restoration require agreed server contracts and independent security review. URL scope and local flags are not authentication. Preserve an explicit preview boundary without claiming a real identity or financial success. Portable npm/Node installation and local browser assets remain required; cross-OS execution remains unverified.

Evidence: HB-63 catalog and screen designs at immutable commit 4d51763458b314fd766998c05fb32a1c40c99e72; HB-66 structure.md, screens.md and screen-ownership.json in PR 62; authorized workspace docs/ui/flows/end-to-end.md and docs/ui/screens/L01/spec.md. Older planning-gate warnings in screen drafts do not override the later planning-completion decision; the drafts also do not establish owner FIX. PM, designer and frontend findings are conception evidence only, not executed screen verification.

# Service-entry implementation

The owner authorized proceeding with the paired conception. Identity now owns six dedicated entry screen components and its module descriptor; app owns EntryShell. Default navigation changes from operations to entry-start. The pre-workspace header contains the canonical brand and locale control, with no business rail/TNB. The start screen prioritizes sign-in and retains invitation, first-administrator and recovery branches. Dedicated forms retain canonical workspace/email/verification-reference fields; challenge and recovery are distinct views. Workspace selection provides search, empty state, explicit radio selection and a fixed synthetic scope.

No authentication request or session is created. Native form submission is prevented; local validation resets fields and reports the disconnected boundary. Editable values never enter URL, storage, review fingerprints/exports or diagnostics. No password or token input is introduced. Preview navigation is separate from request controls, visibly labeled and available only in synthetic mode. The fixed internal workspace preview leads to my-tasks; the customer and first-setup previews honestly reach pending bodies. These preview links do not resolve or redefine the production role-based home mapping.

The shared review control is compact, with ID/revision/status beside its trigger. Detailed item lists, storage disclosures and export move into its dialog; storage/digest failures remain visible outside it. Entry layouts place this auxiliary strip after principal content; existing workspace ordering remains. All shared consumers receive a new source fingerprint. Editable entry screens use a mount nonce plus edit revision, mode and non-sensitive variant, so changing or reloading a form cannot reuse an unrelated FIX without retaining identity values. Historical records remain exportable; reload requires renewed local review for those forms. Exported variants distinguish verification from recovery.

Design review and independent boundary review passed before implementation. Root remains the sole writer; service_frontend independently reviews source and service_designer inspects rendered variants. No owner FIX, real identity integration, deployment or merge is inferred. The complete service remains in scope; thirteen distinct bodies now exist, not 112 completed screens.

# Owner correction: no decorative colored borders

The owner rejected colored-border decoration in the service entry candidate. Do not add brand-colored edge strips to notices, colored outlines to selected cards, or accent borders to tabs in subsequent service screens. Use neutral separators, spacing, text hierarchy and explicit selection controls instead. Keyboard focus must remain visible and is an interaction indicator rather than decoration. This correction removes the entry notice/feedback accent edge and tinted fill, makes selection borders neutral, and identifies the active tab through weight and underline. Canonical logos/tokens remain unchanged.
