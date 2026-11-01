---
type: ArchitectureResearch
title: Platform extension patterns and customer-specific assembly
description: Compares professional platforms and defines a shallow modular frontend with controlled installation profiles and tenant configuration.
status: draft
---

# Conclusion and scope

The owner clarified that simplicity must coexist with expansion and customer-specific assembly. A flat pages/components layout alone does not define how customer variations compose. A deep feature/model hierarchy is also unnecessary. Use a shallow modular application with one explicit assembly point, reusable business modules, narrowly defined extension slots and validated profiles. This is a project design judgment informed by the observations below, not a universal React standard or adoption of any external platform runtime.

This refines the unaccepted HB-66 draft in place. The updated [structure](structure.md) and [112-screen ownership map](screen-ownership.json) remain the current design proposal. No customer forks, runtime plugin loader or product code are implemented by this research.

# External evidence

Research date: 2026-11-01. Only official platform documentation and original repository trees were used for technical conclusions. Documentation URLs are mutable observations at that date; repository snapshots below identify immutable inspected revisions. No platform was installed or benchmarked.

| Platform | Observed structure / extension mechanism | Applicable lesson / limit |
| --- | --- | --- |
| [Backstage app](https://backstage.io/docs/frontend-system/architecture/app/) and [extensions](https://backstage.io/docs/frontend-system/architecture/extensions/) | An app instance assembles contributed features; extensions have identity, inputs and attachment relationships | Keep assembly explicit and modules separately owned. Its full extension tree/framework is more machinery than this initial product needs. |
| [Medusa widgets](https://docs.medusajs.com/learn/fundamentals/admin/widgets) and [UI routes](https://docs.medusajs.com/learn/fundamentals/admin/ui-routes) | React widgets attach to declared zones; added pages live under admin routes; plugins can share both across apps | A few stable page slots plus dedicated pages allow customization without cloning whole screens. Explicit widget IDs survive moves. Current docs deprecate most before/after zone suffixes from v2.17.2; do not copy outdated examples. |
| [Grafana app plugins](https://grafana.com/developers/plugin-tools/key-concepts/plugin-types-usage) and [versioned extensions](https://grafana.com/developers/plugin-tools/how-to-guides/ui-extensions/versioning-extensions) | App plugins can bundle pages/panels/data sources; extension point major versions permit migration between contracts | Version extension contracts and test compositions. Grafana's organization enablement is not Hybrid Billing's tenant permission model. |
| [Strapi Admin API](https://docs.strapi.io/cms/plugins-development/admin-panel-api) | A plugin has an admin entry interface with registration/bootstrap/translation functions and UI injection APIs | Use a small module entry descriptor, with owned translations and declared contributions. Do not inherit arbitrary patching or lifecycle complexity. |
| [Shopify app extensions](https://shopify.dev/docs/apps/build/app-extensions) | Extensions target defined UI surfaces; app configuration and extensions are versioned together | Release a resolved, reproducible composition and retain rollback evidence. Hosted distribution and Shopify authentication are not adopted. |
| [Odoo 18 registries](https://www.odoo.com/documentation/18.0/developer/reference/frontend/registries.html) | Ordered registries provide lookup points for fields/views/actions/services | Explicit IDs and ordering matter. This is a versioned documentation example, not a statement about the latest Odoo or a choice of its frontend framework. |

Original tree checks: Backstage at `24deeed9605d2afa54c2e7456912317fa8a1dcee` contains `packages/app/src/App.tsx` and catalog-owned components/extension entries under `plugins/catalog/src/alpha`. The [actual app entry](https://github.com/backstage/backstage/blob/24deeed9605d2afa54c2e7456912317fa8a1dcee/packages/app/src/App.tsx) combines plugins and overrides; its alpha imports are observed examples, not dependency recommendations. Medusa at `7e797dfd591d25d9dded71b5ab8bbc7b89958f17` contains business routes and colocated screen components alongside shared components under [dashboard/src](https://github.com/medusajs/medusa/tree/7e797dfd591d25d9dded71b5ab8bbc7b89958f17/packages/admin/dashboard/src). Both recursive GitHub tree responses were complete, not truncated. These are focused structure checks, not full source audits.

Retrieval limits: the guessed `grafana/plugin-examples` repository lookup returned 404; Grafana conclusions use its official documentation instead. Backstage's generic architecture URL failed, while the specific app/extensions pages succeeded. The Odoo registry statement came from the official search-index excerpt; direct registry/assets opens timed out. Medusa's directory web view failed but the pinned GitHub API tree succeeded. No failed lookup is counted as a successful source inspection.

# Keep the source tree shallow

```text
apps/console/src/
  app/                  shell, routes and assemble.ts
  modules/
    pricing/
      module.ts         public contribution descriptor
      pages/            dedicated screens; helpers/tests beside the screen
      api.ts            split only when actual responsibility grows
    issuance/
    customer-portal/
  shared/               common UI, transport, session, locale and formatting
  extensions/           reviewed additions, named by capability rather than customer
  profiles/             public reusable presets and schema; no customer records
```

The domain owners in the 112-screen map remain useful responsibilities, not a demand for 15 empty module packages. Each module starts with its entry and actual pages. Avoid mandatory model/hooks/services/providers subtrees per module. An extension stays local until independently versioned reuse justifies a package. Customer-private extension code may live in a separately controlled package/repository and enter an approved build; no customer-private source is published in the common bundle or repository.

# Three different kinds of variation

| Need | Mechanism | Example |
| --- | --- | --- |
| Different visible options/defaults | Validated configuration | Available dashboard blocks, locale preference, allowed column visibility |
| Additional behavior or presentation | Reviewed extension targeting a declared slot/API | Extra evidence panel or a delivery-mapping editor for an agreed ERP contract |
| Different installed product capabilities | Release composition | A self-hosted installation includes agreed modules and approved extension versions |

Do not equate an installation purchaser, workspace/tenant, MSP customer and user. An installation profile selects shipped code; server-authenticated scoped configuration selects available behavior for the actual business context. A tenant cannot add code by submitting a profile ID. Profiles do not supply customer pricing formulas, license grants, permissions or identity. Those remain authoritative backend contracts.

Prefer reusable capability names and presets over `customer-a/` and `customer-b/` screen clones or long `if (customerId)` branches. A unique requirement first needs a classified difference and owner: configuration, extension or genuine core product capability. The assembler does not repair duplicated semantics after the fact.

# Assembly contract

The following is a proposed internal frontend contract, not a public plugin SDK or API already implemented.

1. The build reads explicitly approved local module/extension imports and a validated installation preset. Each descriptor supplies stable ID, version, required host/slot/API capabilities, dependencies and contributed routes/slots/message namespaces. The build pins exact dependency artifacts in its lockfile and records their digests; compatibility ranges alone are not reproducible.
2. Validate required dependencies, version compatibility, cycles, duplicate route/item/extension IDs, unknown slots and unauthorized replacement attempts. Reject conflicts; do not use import order or last-writer-wins to choose behavior. Slots declare cardinality and deterministic ordering, with stable ID as an ordering tie-breaker only when the slot permits multiple contributors.
3. Resolve an immutable composition manifest recording host version, selected modules/extensions, dependency and asset digests, schema/profile revision and applicable contract versions. Generate routes, navigation and slot bindings from this one resolved composition. Lazy imports refer only to bundled approved code, never URLs supplied by tenant config.
4. At runtime, validate public configuration against a versioned schema. Use documented precedence for allowed presentation values: product defaults, installation preset, then authenticated scoped settings. Replace arrays by explicit keyed selection rather than recursively merging arbitrary objects. Reject unknown keys and invalid values; do not accept scripts, arbitrary CSS/HTML or executable imports.
5. Actual availability is constrained by the shipped composition, backend capability/entitlement results, authenticated permission/disclosure scope and current prerequisites. A presentation override cannot widen any of these. Missing or stale authority disables protected actions until refreshed. Server authorization remains mandatory regardless of browser checks.
6. Identity, scope, permission or effective-profile revision changes invalidate related requests, caches, selections and previews together. Unmount obsolete contributions and reset their state. Back navigation cannot restore a previous scope's composition data or command eligibility.

Start with additive evidence/detail panels and explicit extra routes. Protected core controls for financial approval, issuance, exact totals, permission decisions and owner FIX cannot be replaced or suppressed by generic extensions. Configuration may not hide required warnings, scope or authoritative totals. Broader replacement needs a separately reviewed contract, not an escape hatch such as replaceComponent(anyId).

A module provides a narrow exported descriptor and public interfaces. Extensions use declared slot props and host APIs, not deep module imports, global store access or unrestricted raw record dumps. The backend still filters data per caller. Reviewed same-origin JavaScript is trusted application code: React context, TypeScript and an error boundary are not a security sandbox. Untrusted third-party code is outside this baseline; adding it requires an isolated execution and data/message authorization design. Do not claim ordinary plugins prevent data exfiltration.

# Failure, release and verification

Optional presentation extensions may fail with a visible local error while the rest of the page remains usable. If a declared required capability fails, block its dependent workflow and show the reason; never silently fall back to an alternate provider or simulated approval. Error containment is distinct from security isolation.

Test the default composition and every actually shipped installation profile, plus targeted incompatible versions, cycles, collisions, missing required capabilities, denied permissions and scope-switch races. Pairwise checks can supplement these tests; they cannot certify every arbitrary combination. Each supported composition has its own release evidence and ownership. Limit supported combinations instead of allowing unconstrained customer patches.

Upgrade validates profile/schema and slot migrations before activation. Preserve the previous resolved composition and configuration revision. Rollback is permitted only when compatible with current backend contracts and data migrations; restoring an old UI is not authority to restore old access or billing state. Use staged validation with isolated fixtures before activating a profile. No runtime extension marketplace, remote updater or microfrontend federation is needed initially.

Screen review fingerprints include the effective composition/profile and relevant extension versions as well as screen/shared dependencies. FIX is scoped to those effective variants; profile changes do not inherit blanket acceptance. Keep non-sensitive digests in review records, not tenant data. OS-independent install/build/run principles still apply; composition does not introduce a required OS shell or cloud service.

# Concrete synthetic assembly examples

- Installation P uses the standard source/pricing/issuance modules and a file-delivery capability. Installation Q uses the same modules plus an approved ERP mapping extension. Their common screens are maintained once; changing the ERP extension only affects mapped consumers and its dependent tests/review scope.
- Two customers within P use different server-authorized disclosure and display settings. No separate frontend build is needed merely for those settings, and neither customer receives another customer's data. Turning a display setting on cannot enable an unavailable or unauthorized command.
- If Q requires confidential extension code, package it only into Q's controlled release artifact. Do not ship it to all installations and assume a hidden menu protects the code. The private package still follows the same versioned host contracts and review/testing gates.

These examples are proposed scenarios, not existing customer arrangements. First implement one actual extension need and two synthetic profiles to prove the composition seam; do not create a full plugin framework before it has a consumer.
