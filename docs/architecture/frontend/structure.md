---
type: FrontendArchitecture
title: Whole-product frontend architecture and directory design
description: Maps the complete screen inventory to a modular React application with explicit state, data, review and portable delivery boundaries.
status: draft
---

# Decision and evidence

[HB-66](https://linear.app/hybrid-billing/issue/HB-66/design-whole-frontend-architecture-and-screen-directory-ownership) responds to the owner's request to design the whole frontend and its file/directory structure before implementation. Root owns the design and mapping; gap_review independently reviews architecture and scope. Accountable owner: Seung Woo Park. Machine: mac_mini.

Use one modular React application, with TypeScript proposed for implementation contracts. Group screens by business responsibility, not by visual widget or provider alone. Keep screen-specific composition and behavior explicit. Reuse shared UI and canonical design assets; do not render the entire product through one catalog-driven form/table component. React is an [existing owner decision](../current-design-decisions.md); TypeScript and the physical paths here are design selections, not installed dependencies. Bundler, router, server-state library, test framework and exact versions remain implementation decisions against actual runtime support requirements. This document does not select Next.js, SSR, microfrontends or a cloud service.

The complete design input is [HB-63 at 4d5176](https://github.com/myblueton-sw/hybrid_billing/tree/4d51763458b314fd766998c05fb32a1c40c99e72/docs/design-modules/00-scalable-console/r1). It contains 112 route identities, screen profiles, bounded synthetic interactions and local review evidence. It is a separate open PR, not a merged product app. The [route ownership map](screen-ownership.json) and [screen directory](screens.md) cover every input route once, preserving registry IDs, baseline pages and related routes. They describe proposed paths, not files that already exist or screens that have passed product acceptance.

The authorized workspace also contains the planning repository-structure proposal and required companion rules. Earlier language/tooling uncertainty is superseded only where current owner decisions establish a choice. This frontend design refines the frontend placement; it does not replace backend module ownership or claim that all companion inputs are published. Missing inputs must be obtained before dependent implementation.

# Alternatives and boundaries

| Choice | Reason | Cost / boundary |
| --- | --- | --- |
| One app with feature modules | Shared shell, identity context, brand and review workflows with one build; fits the current single product | Import rules and feature ownership must prevent a large coupled application |
| Separate frontend per provider | Rejected as baseline: provider details reuse source/contract workflows | Provider-specific forms remain inside the owning feature; support still needs per-provider evidence |
| Microfrontends | Deferred: no established independently released UI teams or deployment need | Avoid duplicate session state, assets and cross-app version contracts |
| Universal metadata renderer | Rejected as the full implementation: catalog metadata lacks complete workflow semantics | Metadata remains useful for routing, navigation, traceability and review indexing |
| One shared global business store | Rejected: unrelated workflow state leaks and stale scoped results become hard to reason about | Keep route/filter, request, editable draft and review state in their actual owners |
| Browser computes billing outcomes | Rejected: presentation is not the billing ledger or authorization authority | Exact values and authoritative command state come from validated server contracts |

# Proposed source tree

Only create directories when an implementation adds real files. The paths below are the target layout, not an instruction to create empty scaffolding.

```text
apps/console/
  package.json                 app dependencies and portable commands
  tsconfig.json                selected compiler and import boundaries
  index.html                   browser entry, no embedded credentials
  src/
    main.tsx                   bootstrap and root error handling
    app/
      App.tsx                  assembly only
      routes/                  route registration, lazy entry points, not screen logic
      navigation/              domain menu, search, breadcrumbs, return destinations
      layout/                  workspace shell, public entry shell, portal shell
      providers/               session, locale, runtime configuration composition
    features/
      identity/                login, invitations, identity policy and sessions
      onboarding/              first setup and readiness
      operations/              close workbench and assigned tasks
      organizations/           relationships, customer context and change impact
      sources/                 collection, supplier evidence and reconciliation intake
      analytics/               cost exploration, pivot and reporting
      pricing/                 purchase/customer contracts, rules, seats and allocation
      calculation/             run revisions, explanation and reconciliation
      issuance/                financial approval, document lifecycle and assignment
      delivery/                ERP, file handoff and disclosure delivery
      history/                 historical close, correction and audit views
      customer-portal/         customer-authorized bill and inquiry views
      administration/          installation, license, lifecycle and integrations
      assistant/               constrained conversation, proposals and job navigation
      design-review/           index, review controls, evidence and scope comparison
    shared/
      ui/                      fields, dialogs, bounded table, picker, feedback
      api/                     transport, cancellation and response/error decoding
      session/                 current authenticated context and invalidation signals
      i18n/                    locale selection, shared messages, direction helpers
      formatting/              exact-value presentation, timestamps, identifier isolation
      styles/                  app-level semantic layout rules
    testing/                   synthetic adapter setup and shared test helpers
  tools/                       portable build/config/preview entry points as needed
packages/
  design-system/               existing canonical assets, tokens, provenance and notices
  contracts/                   existing contracts; future API schema only when agreed
  sdk/                         generated clients only when a real schema/generator exists
tests/
  e2e/console/                 user journeys across feature boundaries
  contracts/console/           real adapter/schema and failure-boundary checks
  fixtures/console/            small synthetic scenarios and independent expectations
docs/ui/screens/<screen-id>/   screen spec, item IDs, state matrix and acceptance evidence
docs/architecture/frontend/    this architecture and complete ownership map
output/                       generated artifacts and raw verification evidence
```

`packages/design-system` remains the token/asset authority. Keep React UI primitives app-local initially; extract a shared `packages/ui` only when another real consumer needs it. Do not create competing tokens or copy the supplied archive's unreviewed Money/Icon implementations. Existing `tools/design` and static design modules remain their own review artifact until accepted replacement scopes are finalized.

# Feature and file responsibilities

A typical feature grows by responsibility, not by adding every file in advance:

```text
features/sources/
  routes.ts                    public route descriptors / lazy screen references
  screens/index-intake/
    Screen.tsx                 screen-specific layout and actions
    state.ts                   screen state and transitions when nontrivial
    Screen.test.tsx            meaningful component/scenario checks
  components/                  source-domain widgets shared by actual source screens
  model/                       query scope, draft validation and data projection
  api/                         source contract adapters and query/command orchestration
  messages/                    en, ko, ar feature message catalogs
```

Use canonical route IDs as screen directory names. A filename change must not change screen/item identity. Component files use descriptive PascalCase names; ordinary modules use descriptive kebab-case names except established entry filenames. Each directly maintained source/test file stays below 1,500 physical lines, and responsibility splitting happens earlier when useful. No explanatory source comments; use types, names, tests and design records.

Dependencies flow from app composition to features and from features to shared modules/design-system. Shared modules never import app or features. Feature internals do not directly import another feature's internals. Cross-feature navigation uses route IDs/descriptors; cross-feature orchestration belongs in app composition or an explicit owner contract. Do not move domain logic into shared/utils to bypass a cycle. Feature route descriptors and explicitly exported types are the narrow public surface. Proposed import checking should be added with the selected tooling; it is not installed by this document.

A screen can compose multiple concerns while keeping one primary owner. For example pricing owns the price-rule editor; issuance owns financial approval; design-review owns visual acceptance. A customer portal may reuse a primitive but must not import internal-cost screen projections or broad cached responses. One app bundle is not an authorization boundary. Customer-sensitive data must never be fetched and hidden as a substitute for server-side disclosure.

# Whole-screen ownership

[screens.md](screens.md) lists all 112 routes and their proposed `Screen.tsx` paths. The owner map is keyed by screen ID, not label or array position. It preserves related-screen links and original baseline mappings, including multiple screens referring to the same legacy registry ID. Navigation groups remain separate from code ownership: regrouping a menu is not permission to move contract authority.

Special ownership choices include cost pivot/raw explorer/reports under analytics, organization relationships under organizations, approvals under issuance, ERP/file/publication delivery under delivery and assistant under its own feature. Customer-facing variants stay in customer-portal rather than reusing an internal financial screen with hidden columns. The design overview/index and synthetic scenario selector belong to the review experience; they are not ordinary financial operations.

Every screen gets a dedicated `docs/ui/screens/<screen-id>/spec.md` when developed. Preserve and link prior registry-based specs instead of overwriting their meanings. Each spec records purpose, user/context, entry/return routes, exact layout sections, fields/validation, table columns/actions, permission and data states, API dependencies, implementation items, locale/layout variants and acceptance scope. Existing semantic item IDs remain stable. Newly discovered items receive new scoped IDs; they are not numbered by display position. This architecture map is not a substitute for those individual designs.

# Page layout and design system

Use the established Fusion AI assets and IBM Plex fonts from local locked assets; semantic tokens, square geometry, no shadow/gradient/translucent overlay, 16px console body and responsive dimensions remain the design baseline. Read the actual [HB-63 design guide](https://github.com/myblueton-sw/hybrid_billing/blob/4d51763458b314fd766998c05fb32a1c40c99e72/docs/design-system.md) before writing UI. Typography, breakpoints and brand provenance are not redefined here.

| Region | Owner / behavior |
| --- | --- |
| Header | Shell: brand, locale, authenticated workspace context and session controls |
| Primary/secondary navigation | App: domain navigation and searchable screen links; keyboard and mobile behavior |
| Screen heading | Screen: title, scope, freshness/revision and task-specific actions |
| Main content | Screen: purpose-specific form, evidence, bounded table, comparison or investigation layout |
| Feedback | Screen + shared primitives: inline validation, loading/empty/error/denied states and retry |
| Related work | Screen: explicit next/return destinations without losing designed filter state |
| Review strip | Review composition: screen ID, exact design revision, local/canonical status and consistently located review button |

Review controls appear on every reviewable screen in the design-review experience, visually separate from task commands. They are not required as implementation bookkeeping in ordinary production tasks. Separately identified dialog/detail/state units have their own review scope; otherwise they are explicit items of their owning screen. Focus return, keyboard order and error announcements are part of the screen spec, not inferred from a screenshot.

# State and data ownership

| State | Location | Lifetime / invalidation |
| --- | --- | --- |
| Locale, authenticated workspace and session facts | App providers/shared session | Session changes invalidate protected query data, selections, previews and streams |
| Shareable navigation/filter state | Route controller | Only allowlisted non-sensitive IDs/filter parameters; no credentials or customer search text in URLs by default |
| Server result and query identity | Feature query adapter | Keyed by authenticated context, relevant permission/disclosure revision and full query scope; bounded pages |
| Editable user draft | Owning screen/feature | Explicit leave/reset behavior; no default persistent storage of sensitive draft values |
| Bulk selection and calculation preview | Owning feature | Bound to filter/snapshot/contract scope; change invalidates before command availability |
| Job status | Feature adapter using authoritative operation reference | Poll/subscribe lifecycle stops on route/session change; queued is not completed |
| Review record | design-review storage adapter | Screen/dependency revision + item/variant scope; independent of financial command state |

A scope change increments request identity, aborts obsolete requests where possible and rejects late results even if abort races. Purge protected caches on sign-out or authorization change; do not reuse one user's response for another. Persisted review metadata must exclude customer data. Browser back/forward restores designed filters/navigation, but never revives an invalid preview, permission or command receipt.

All network calls pass through shared transport; feature adapters decode actual agreed schemas and keep business error interpretation with the feature. Shared transport handles cancellation and session-related transport signals, not business approval. No endpoint, JSON field, idempotency guarantee or numeric policy is invented here. Unavailable schemas/services are explicit integration blockers for that path; synthetic mode is visibly labeled and separate from production adapters. An API failure never silently falls back to fixtures or simulated command success. Transport must not automatically retry financial mutations. Monetary DTOs retain exact decimal strings, explicit currency and source revision; client formatting does not change their precision or provenance.

Commands distinguish preflight validation, submitted/pending, authoritative completion, rejection and unknown outcome. On a timeout after submission, do not blindly resubmit a monetary/issuance command. Use the command's agreed idempotency and status-reconciliation contract; if missing, block that integration and expose uncertainty. A financial state cannot be inferred from local UI state, an HTTP success alone or a job's queue acknowledgment. Server checks current authorization and prerequisites on every protected operation.

# Review and FIX design

The [HB-63 screen-review rule](https://github.com/myblueton-sw/hybrid_billing/blob/4d51763458b314fd766998c05fb32a1c40c99e72/.agents/rules/screen-review.md) is authoritative. Design-review state is Draft → Ready for review → owner-confirmed FIX for an explicit revision and scope. An authorized change preserves the old acceptance, marks the new candidate Change pending, and returns it to Ready for review after assessment. No test or agent can approve for the owner.

Separate the review UI, immutable revision metadata, storage adapter and exported evidence. Revision fingerprints cover actual screen code, relevant shared components, tokens, copy and contract dependencies. Shared changes identify affected screens; no-impact changes record an assessment rather than automatically forcing all screens through review. Confirmation pins the displayed revision/item/locale/layout/state scope; changing it invalidates the open confirmation.

Browser storage is the initial local evidence option already present in the design preview. Failed/corrupt/unavailable persistence is visible and must not show persisted FIX. Local storage is not authenticated owner identity or canonical project acceptance. Exported non-sensitive evidence is reconciled into ticket-linked Git records. An authenticated central review service would require a separate contract; this design does not invent or claim it. Implementation status and design-review status are independent fields in screen documentation and review index.

# Installation and operation

Deliver a static frontend artifact with local assets and validated public runtime configuration. API location/base path may be configured without recompiling the app; configuration contains no secrets. Use base-path-aware asset and navigation URLs. Direct screen links must resolve through the configured static-host fallback or an explicitly selected hash router; test reload and browser back/forward under a non-root base path. Do not select a router by silently dropping the current route IDs. Same-origin hosting is the preferred integration shape, with the server's actual session/CSRF/CORS contract defined before binding authentication. A browser cannot hold provider credentials or execute privileged collectors.

Use portable Node-based development/build/preview entry points and lockfiles; installation does not require Bash, Homebrew, registry edits, keychain, an OS service manager or Docker. Node is a development/build dependency, not a required runtime on an end user's browser device. The deployed artifact can be served by a compatible static host or the product server; optional host adapters must not become the only path.

Specify supported runtime/browser versions when selecting tooling. The first implementation must provide equivalent install, build, start, stop and verification commands across the declared Windows/macOS/Linux targets, including paths with spaces and configurable ports/base paths. Build output goes to an owned generated directory with an asset/license manifest. Offline delivery needs packaged assets and declared dependency preparation; an npm lockfile alone is not an offline installer. Upgrade/rollback must keep asset/config/API compatibility explicit and avoid mixing old HTML with new hashed assets. Production backend installation remains separately scoped.

Current host evidence is from the earlier macOS static preview, not this proposed React application. Windows/Linux and React runtime portability remain UNVERIFIED. Do not claim support for every OS from standards-based design.

# Implementation sequence and gates

| Increment | Concrete deliverable | Required evidence |
| --- | --- | --- |
| Foundation | Select locked tools, React bootstrap, local assets, portable build/preview, shared error/session/locale composition | Clean setup/build on declared hosts; honest support matrix; asset checks; no secret config |
| Vertical screen slice | One bounded source-intake path with actual screen spec and explicit synthetic/real adapter boundary | Loading/empty/denied/failure/retry, back navigation, stale response rejection, EN/KO/AR rendering |
| Shared review integration | Review strip and index with scoped revision/storage/export behavior | Isolated owner-record simulations, save failure, changed revision and shared-impact cases |
| Domain expansion | Dedicated screens in ownership map, prioritized by real contract readiness | Per-screen item trace and relevant journeys; no generic route count as completion |
| Authoritative integration | Connect agreed identity/query/command contracts by feature | Independent security/financial review, negative scope tests, unknown-outcome reconciliation |
| Acceptance/finalization | Owner reviews individual screen scopes and agreed replacement modules | Git-reviewable evidence, update links/generators, remove only actually superseded copies |

Design can proceed on unavailable integrations with explicit states and deferred items. Contract absence blocks dependent behavior, not unrelated screen design. Independent layout/state design remains necessary for every screen; existing catalog titles and form metadata are inputs, not full specifications. Avoid copying synthetic fixture labels or example months as real production defaults.

# Verification strategy and status

Co-locate component/state tests with their owning feature; cross-feature journeys live in tests/e2e/console. Keep independent expected financial values and contract fixtures outside production projection logic. Use the reviewed hb-react-screens and hb-screen-regression skills alongside repository QA, not as substitutes for actual execution.

Required scenarios include late responses after scope changes, revoked sessions and cache clearing, bounded paging/selection, command timeout reconciliation, review persistence failure, changed accepted dependencies, EN/KO/AR keyboard/focus/overflow and direct-link/back navigation. Product tests must use isolated synthetic identities and records; never issue real bills or set owner acceptance.

This increment is architecture and complete route ownership design. [Verification](verification.md) records document/coverage/independent-review checks. No React source directories, dependency installation, product screenshot, runtime test, owner FIX, merge or deployment are introduced here.
