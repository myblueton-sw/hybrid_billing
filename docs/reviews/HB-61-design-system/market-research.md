---
type: Research
title: Billing interface research for US and Middle East markets
description: Official product evidence, regional design guidance, and open validation questions for the supplied design system and the full screen inventory.
status: draft
---

# Billing interface research

Research date: 2026-10-31. Tracking: [HB-61](https://linear.app/hybrid-billing/issue/HB-61/import-owner-supplied-design-system-r1-package). Writer: root; machine: mac_mini.

## Scope and evidence boundary

The owner requested research before a navigation or visual-style decision. This report does not select TNB, LNB, RNB, a hybrid shell, dimensions, or a palette. The supplied package remains evidence for evaluation; this report does not adopt it as a product rule or import runtime components.

Evidence categories are **observed** (published product screenshots visually inspected), **documented** (official written guidance), **local** (workspace source inspected), and **unverified** (not tested or inaccessible). Published screenshots establish an example, not current live behavior or regional user preference. Marketing-site navigation is excluded as evidence of application navigation. This is desk research, not a representative market survey or a US/GCC user study.

Terminology used here: TNB means primary navigation across the top; LNB means primary or secondary navigation at the side; RNB can mean a right-side navigation rail or a contextual panel. A utility header, account-detail tab strip, and product-wide TNB are different structures. In RTL, an inline-start sidebar may appear on the right without becoming a different information architecture.

## Product and design-system comparisons

| Source and evidence | What was established | What it does not establish |
| --- | --- | --- |
| [Kill Bill Kaui guide](https://docs.killbill.io/latest/userguide_kaui), observed: “Homepage Labeled” and “Account Submenu” images | Published homepage has a top utility/search area with account creation, administration, and tenant/user controls. Account detail has horizontal tabs for account, subscriptions, invoices, payments, timeline, tags, and custom fields. No persistent left rail is visible in those two images. | These are guide screenshots, not a tested live deployment. Account tabs are not proof of a product-wide TNB or of market preference. |
| [ERPNext Workspace](https://docs.frappe.io/erpnext/workspace), documented | Desk has a persistent categorized sidebar. Workspaces expose module dashboards, shortcuts, and links to records/reports; customization is documented. | A module-heavy ERP is a useful comparator, not proof that billing needs its hierarchy. |
| [Lago workflow](https://getlago.com/resources/how-lago-works), documented | Customer, subscription, usage, invoice, and credit-note workflows provide a billing-domain comparator; the product describes self-hosted use and integrations. | The page does not establish exact current application navigation, mobile behavior, or Arabic support. |
| [Lago revenue analytics](https://getlago.com/platform/revenue-analytics), documented | Revenue analysis, filtering/export, and currency-aware breakdowns provide reporting comparison points. | Published material is not an interaction or reconciliation test. |
| [Stripe Dashboard search](https://docs.stripe.com/dashboard/search), documented | Cross-resource search is a discovery mechanism alongside navigation. | Search documentation does not settle TNB versus LNB. |
| [Stripe customer portal](https://docs.stripe.com/customer-management), documented | A separate customer-facing surface covers invoices, account/payment details, and subscriptions. | These capabilities are not new Hybrid Billing requirements. The portal is not evidence for an operator console shell. |
| [Carbon UI shell](https://www.carbondesignsystem.com/building-blocks/core/components/ui-shell-left-panel/guidelines), documented | Header, left panel, and right panel can be used independently. A header alone is described for simple products. The left panel is optional product navigation; the right panel supplies additional system actions/content. Carbon recommends a left panel for more than five secondary items or frequent secondary switching and page tabs beyond its supported two navigation tiers. | This is Carbon's design guidance, not a measured US-market preference or a universal menu-count rule. |
| [Cloudscape app layout](https://cloudscape.design/components/app-layout/?tabId=usage), documented | Its shell accommodates collapsible side navigation, tools panels, drawers, and a split panel. | Component availability does not establish the correct default visibility or placement for this product. |

The inspected examples preserve TNB and horizontal navigation as research candidates. They also establish side-navigation and contextual-panel alternatives. The sample cannot support a ranking by popularity, productivity, or geographic preference.

## Navigation questions to test with the same tasks

The following are research hypotheses, not adopted designs.

| Candidate | Evidence anchor | Unresolved task-level question |
| --- | --- | --- |
| Top primary navigation with page-level tabs | Carbon header-only guidance; Kaui horizontal account tabs are a separate, narrower example | Can the actual top-level groups and Arabic labels remain visible at small widths and increased zoom without concealing frequent destinations? |
| Side navigation with a utility header | ERPNext workspace; Carbon left panel | Does persistent hierarchy reduce wrong-destination selections during repeated cross-module billing work, and what table width does it consume? |
| Top domain navigation plus side subnavigation | Carbon supports composing shell elements | Can users distinguish domain, organization, and record context without duplicate menu levels? This exact composition was not tested in Hybrid Billing. |
| Contextual right-side panel | Carbon right panel; Cloudscape tools/split panels | Can evidence, assistance, or record details remain available without obscuring the active table or confusing navigation with an action? |
| RTL inline-start navigation | W3C direction guidance below | Does visual placement, keyboard order, and mixed Arabic/Latin content remain coherent? Right placement alone is not evidence for a separate RNB architecture. |

Candidate comparisons need equivalent content and tasks: find a billing exception, inspect its source evidence, return to the same filtered list, compare before/after results, and locate a customer document. Measure completion, wrong destination, scope mistakes, backtracking, and overflow. No participant results or timings have been collected.

## Dimensions, density, typography, and tone

| Topic | Supplied package: local observation | External comparison and limit |
| --- | --- | --- |
| Shell dimensions | Rail 205px, header 76px, horizontal content padding 32px; responsive rules at 900px and 600px | [UAE layout guidance](https://designsystem.gov.ae/guidelines/layout) uses breakpoints 640/768/1024/1280/1536px and recommends 24px gutters for larger screens, with smaller-screen adjustments. These are government-web conventions, not measurements of billing users' devices. |
| Text density | Body 14px, table/label 12px, small text 11px, footer 10px; body line height 1.5, Arabic 1.7 | [UAE typography](https://designsystem.gov.ae/guidelines/typography) documents a 16px base, recommends at least 16px unless context warrants less, and uses Roboto/Noto Kufi Arabic. This differs from the supplied density; legibility at zoom and Arabic shaping remains untested. |
| Typeface | IBM Plex Sans, Sans Arabic, Sans KR, and Mono references | [IBM typeface guidance](https://www.ibm.com/design/language/typography/typeface/) documents multilingual Plex families. Font availability is not evidence of regional preference or verified loading in this application. |
| Palette | Navy `#1b3a5c`, teal `#0f7c86`, white page, light-gray navigation/table surfaces, dark body text | [UAE colour guidance](https://designsystem.gov.ae/guidelines/colour-system) explicitly ties its core palette to federal identity. Its gold palette is not evidence that Middle Eastern commercial billing software should use gold. No regional preference claim for navy/teal is established. |
| Shape and decoration | Zero radius, no shadow; borders and typographic hierarchy carry separation | These are supplied design choices. This research has no comparative evidence that square or rounded controls improve billing work. |
| Focus and state | Focus token 3px with 3px offset; separate status colors exist | Token presence alone does not verify focus visibility, keyboard operation, color contrast, or non-color state labels. |

The local source is `design-system-reference/hybrid_billing/packages/design-system/src/styles/` (`tokens.css`, `base.css`, `components.css`, and `fonts.css`). This workspace evidence is not yet part of the tracked baseline; a fresh clone may not contain it. Existing `docs/ui/design-system/foundation.md` differs in typography, palette, rounding, and dimensions. Neither document is replaced by this report.

Screen size research must distinguish viewport dimensions, content width, and device resolution. No customer device analytics were available. A future comparison can sample narrow/mobile, tablet, laptop, and wide desktop viewports, plus 200%/400% zoom; those samples would be test conditions rather than declared support or market share. Dense financial tables require their own overflow and context checks.

## US and Middle East applicability

US-oriented enterprise design references and global billing products are comparators, not evidence of what US users prefer. Similarly, UAE government guidance cannot represent all Middle Eastern countries or private-sector buyers. Saudi DGA's design-system site could not be reliably retrieved during this research; its detailed guidance is unverified.

[W3C direction guidance](https://www.w3.org/International/questions/qa-html-dir) distinguishes language from direction and documents document-level RTL and mixed-direction handling. Arabic names beside Latin account IDs, invoice references, ISO currency codes, and negative amounts therefore need rendered examples and copy/paste checks. This research does not decide numeral preferences, translation scope, or financial formatting rules for all countries.

[WCAG contrast guidance](https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum.html) sets 4.5:1 for ordinary text and 3:1 for qualifying large text. [Target-size guidance](https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum.html) specifies a 24 CSS-pixel minimum with exceptions, including spacing conditions. [Reflow guidance](https://www.w3.org/WAI/WCAG21/Understanding/reflow) addresses a 320 CSS-pixel width and exceptions for content requiring two-dimensional layout; a table exception does not exempt the whole page. These are evaluation criteria, not a claim that the supplied system passes.

For an open, customer-operated product, the research backlog includes local font delivery, translated extensions, configurable branding, long plugin labels, keyboard operation, and installations without public CDN access. These are evaluation questions, not new committed features. Existing product licensing and third-party notices remain unchanged.

## Full screen-inventory research map

The local `docs/ui/screen-registry.json` contains 49 entries. This table maps every ID to a research concern; it does not claim that every screen has been rendered or user-tested. Stale implementation/authorization flags in that local registry are not used to override the owner's later planning-completion declaration.

| Screen IDs | Family | Evidence connection and remaining inspection |
| --- | --- | --- |
| C01, C02, C03 | Customer costs, documents, questions | Stripe portal establishes a separate customer surface; compare document discovery and narrow-screen use. Inquiry flow remains product-specific. |
| L01, L02, L03 | Login, invitation, recovery | No comparable authentication flow was tested; inspect Arabic input, validation, focus, and recovery states locally. |
| U01, U02, U03, U04 | Operational overview, close calendar, tasks, exceptions | ERPNext workspaces provide a dashboard/shortcut comparator; test navigation from exception to evidence and back. Calendar behavior is not verified by that source. |
| O01, O02, O03 | Organization relationships and contracts | Kaui demonstrates account context and tabs; organization graph and change-impact flows remain product-specific. |
| S01, S02, S03, S04, S05, S06 | Source onboarding, import, usage, evidence, reconciliation | Lago provides usage/integration context; inspect long tables, incomplete imports, conflicts, and evidence detail in this product. |
| B01, B02, B03, B04, B05, B06, B07 | Cost pools, assets, allocation, rates, FX, commitments, marketplace | No directly equivalent complete comparator was inspected; examine editable grids, precision display, preview boundaries, and missing-versus-zero states. |
| B08, B09, B10, B11 | Billing runs, approvals, history, credits/inquiries | Lago and Kaui provide invoice/credit/account context; approval and before/after comparison need local scenario inspection. |
| I01, I02, I03, I04 | Invoices, ERP exchange, file handoff, disclosure | Invoice comparators exist; export status, partial failure, disclosure scope, and handoff recovery remain untested. |
| R01, R02 | Reports and audit | Lago analytics and Kaui record navigation are comparators; inspect currency labels, filter persistence, evidence links, and audit readability. |
| A01 | Assistant | Cloudscape contextual tools are a layout comparator only; no equivalent assistant workflow or output reliability was tested. |
| M01, M02, M03, M04, M05 | Users, identity, roles, service accounts, personal settings | Kaui shows administration controls, not an equivalent permission model. Inspect discoverability and clear scope without inferring authorization from menu visibility. |
| P01, P02, P03, P04, P05, P06, P07 | Installation, extensions, recovery, retention, release evidence, demo, licensing | Self-hosted product comparisons establish relevance; no full installation/operator UX comparator was tested. Inspect progress, diagnostics, failure recovery, and long technical identifiers. |

## Findings and unresolved decisions

1. Top-based and side-based structures both have official precedents. The evidence does not justify excluding TNB or selecting LNB by default.
2. Customer self-service and operator administration have different task scope in the inspected products. Their navigation must be compared separately before making a shared-shell decision.
3. The supplied package has concrete tokens that can be evaluated, but the research has not certified its density, accessibility, RTL behavior, responsive interactions, or financial display components.
4. Government identity palettes and design-system breakpoints are source-specific conventions. Neither establishes regional taste or device prevalence.
5. Further confidence would require equivalent screen prototypes, representative US and Arabic-speaking users, and observed task results. The full inventory above identifies coverage and gaps; this desk-research report is not a completed usability study.

No product code, base-style rule, menu specification, financial contract, permission boundary, or licensing policy is changed by this research. Verification and independent-review status are recorded in [verification.md](verification.md).
