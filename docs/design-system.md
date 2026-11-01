---
type: DesignSystem
title: Hybrid Billing active design foundations
description: Defines the adopted brand assets and visual rules, explicit console overrides and boundaries of the supplied design-system reference.
status: draft
---

# Active scope and source

The owner's supplied hybrid_billing-design-system.zip is the source of the Fusion AI logo, navy/teal palette, IBM Plex family, square geometry and line-icon rules. HB-63 adopts a static foundation subset in [packages/design-system](../packages/design-system/README.md) for the [scalable console](design-modules/00-scalable-console/r1/README.md). The original archive remains untouched. Its setup instructions do not select a production stack or approve every React component.

The canonical paths are `packages/design-system/assets`, `packages/design-system/styles/tokens.css` and the explicit `console.css` overrides. [Provenance](../packages/design-system/provenance.json) records source paths and SHA-256 for unchanged supplied assets. New screens must use these sources rather than copying a palette or inventing a replacement mark. [Work rules](../.agents/rules/design-system.md) and [build procedures](design-build-guide.md) connect this guide to actual execution.

## Brand and visual rules

| Concern | Active rule |
| --- | --- |
| Logo | Supplied Fusion AI: subscription/usage halves with four-point joining spark. Use hb-lockup-color.svg on light header; supplied reverse/mono variants only for their appropriate surfaces. |
| Orientation | Logo remains LTR and is placed at inline-start, including the right side of an RTL header. Do not mirror its geometry or translate/re-typeset its wordmark. |
| Size and space | Mark minimum 16px; horizontal lockup minimum 20px mark height. Clear space on every side is half the mark height. Current header uses a 48px source viewBox height and 24px clear space. |
| Color | Brand navy #1b3a5c and teal #0f7c86; slate neutrals and semantic status tokens from tokens.css. Application CSS references variables, not color literals. |
| Shape | Zero-radius controls, panels, tables and chips; hairline borders. No shadows, gradients, blur, glass or translucent overlays. The modal backdrop is opaque. Logo geometry is not altered to match control shape. |
| Icons | Supplied 24px-grid, 1.75-stroke line SVGs. No emoji, Unicode symbol icons, generic currency glyphs, flags, religious/national symbols or decorative photography. |
| Fonts | Locally built IBM Plex Sans, Sans Arabic, Sans KR and Mono. No runtime CDN. Browser system fallback remains available on a load failure, which must be reported rather than certified as the intended font. |
| States | Status text accompanies color. Empty, not loaded, error and confirmed zero remain distinct. Visible focus uses the supplied focus token. No animated decorative motion. |

Do not automatically apply the original archive's React components. Its Money implementation uses binary numbers/implicit rounding and does not validate missing currency; Icon interpolates titles into SVG; Rail/LinkTabs keyboard/default navigation, responsive grids and Field typing need review. These are recorded source observations, not fixes implemented by this static adoption.

## Explicit console overrides

The supplied base tokens remain byte-identical. `styles/console.css` sets the current console's body/table type to 16px and Arabic line-height to 1.8. These retain the earlier UAE-informed legibility decision instead of silently adopting the archive's 14px body and 12px table defaults. Six TNB domains and searchable secondary navigation remain the current design proposal; they are not a claim of owner-final acceptance of every screen.

The console uses responsive content/sidebar/table/picker dimensions instead of the supplied fixed 205px rail and 76px header. Its breakpoints are 640/768/1024/1280/1536px; desktop content gaps use 24px. Component-specific maximum heights, minimum hit areas and mobile wrapping remain local layout values. Common spacing uses the supplied exact spacing tokens where available; no automatic rounding to a new spacing grid is performed. These layout overrides do not permit a new color palette, round controls or shadow effects.

## Locale and data meaning

English US, Arabic UAE and Korean are first-class screen variants. Arabic uses RTL layout and its own IBM Plex family; Korean uses keep-all. Identifiers, ISO dates, codes and financial representations stay directionally isolated. Preserve exact amounts with explicit ISO currency and source precision; currency is never inferred and unconverted currencies are never aggregated. The current prototype shows unavailable values where a financial contract is absent and implements no money calculation.

Country-specific wording follows the [HB-61 terminology evidence](https://github.com/myblueton-sw/hybrid_billing/blob/2694eab/docs/ui/design-system/terminology.md), currently a separate PR. Translation alone is insufficient: validate the concept against local CSP, IT operations and billing usage, record the source and have local practitioners review candidate copy. Arabic Saudi and UAE conventions must not be treated as interchangeable by default. Native-language acceptance remains pending.

Large-data design requires bounded pages, searchable account selection, snapshot/filter-bound bulk selection and explicit count certainty. The prototype's 100,000-account and 1,000,000-record populations are synthetic frontend models. They do not establish production throughput, persistence, authorization or tenant isolation.

## Acceptance boundary

The owner requested application of the supplied design rules and logo. This guide records that adoption for current work; it does not supersede accepted v10 artifacts or certify UAE regulatory/federal-brand conformity. Review English/Arabic/Korean screenshots, focus behavior, local fonts, logo geometry/spacing, overflow and data states. [Verification](design-modules/00-scalable-console/r1/verification.md) distinguishes actual checks from pending business/API/native-language acceptance.


## Owner screen-review decision (2026-11-01)

Under [HB-63](https://linear.app/hybrid-billing/issue/HB-63/implement-multilingual-whole-product-scalable-screen-redesign), the owner requested individual screen designs with implementation-item IDs, followed by a review-confirmation control on every screen. Confirmed screens become FIX for the accepted revision and declared variants. The [screen-review rule](../.agents/rules/screen-review.md) defines stable IDs, owner evidence, scope, storage honesty and change/re-review handling.

Design FIX and implementation completion are separate. An owner-authorized change creates a new candidate without overwriting the prior acceptance. Common-component changes identify every affected fixed screen. Screen acceptance does not authorize merge, deployment or financial operations.

The console now exposes a per-screen review control and stable IDs for the currently displayed implementation items. Confirmation stores scoped local evidence and supports JSON export; a canonical authenticated acceptance store is not implemented. No screen is declared owner-fixed by implementation or QA. See the [frontend design record](design-modules/00-scalable-console/r1/frontend-design.md).


## OS-independent installation and operation

The owner requires installation and operation without dependence on a particular OS. The [platform rule](../.agents/rules/platform-independence.md) governs all designs. Frontend preview tooling uses portable Node/npm commands and standard browser APIs, with local fonts/assets and explicit storage capability failures. macOS execution evidence does not certify Windows, Linux, every browser or production installations; those combinations remain unverified until tested.
