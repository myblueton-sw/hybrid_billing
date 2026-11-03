---
type: NavigationDesign
title: Approved commercial service information architecture
description: Implements the owner's eight business destinations with separate administration, shared Assistant access and unchanged canonical screen contracts.
status: draft
---

# Decision and scope

The owner instructed implementation of the eight-menu proposal after the planning-purpose review. This authorizes the navigation composition, not owner FIX, full functionality, main merge or deployment. Root owns implementation and QA. Existing screen IDs, owning modules, routes and authority remain unchanged. The active presentation source is apps/console/src/app/service-ia.ts; backend module ownership is not derived from a menu group.

# Business navigation

| Destination | Canonical screens |
| --- | --- |
| Overview | U01, U02, U03, U04 |
| Costs and usage | S04, B01, B02, B06, B07 |
| Organizations and contracts | O01, O02, O03 |
| Allocation and pricing | B03, B04, B05 |
| Settlement and billing | B08, B09, B10, I01, B11, I04 |
| Accounting and ERP | I02, I03 |
| Data and connections | S01, S02, S03, S05, S06 |
| Reports and audit | R01, R02 |

Administration is a lower separated group: M01–M04 and P01–P07. A01 is an authorized header entry, M05 stays in the account menu, and C01–C03/L01–L03 retain separate customer/prelogin presentation. U03 keeps its authorized customer projection. All 49 screens are accounted for once in canonical placement; view projections are not new screens.

Navigation and breadcrumbs use the same mapping. Labels distinguish cost exploration, internal allocation, selling policy and independent monthly-recognition/ERP work. Tasks within one canonical screen remain its tabs or subviews: the I02 menu destination explicitly includes monthly recognition, submission and reconciliation, rather than inventing three identical destinations. Payment terms, receipt freshness and due/overdue views remain existing screen requirements rather than new implemented modules.

# Interaction and boundaries

Only allowed registry entries appear; empty sections disappear. The current section opens when the active screen changes. Other disclosure choices persist within the mounted navigation. Header Assistant and Notifications use the same allowed entries as business navigation. Mobile utilities remain reachable through the menu. All navigation uses the existing guarded transition and preserves allowed shared scope and service mode.

No monetary, permission, posting, invoice or collection behavior changes. Accounting navigation does not force customer billing to precede monthly recognition. No unsupported screen capability becomes implemented by appearing in this menu. Developer simulation remains explicitly separated and grants no authority.

# Design and review

The shell composition and external references remain those recorded in service-shell.md. This change applies the owner-approved grouping to the same neutral full-width shell; it imports no external assets or dependencies. The earlier seven-group research candidate remains distinct review evidence, qualified by planning-purpose-review.md; this document is the current implemented navigation decision.

Independent design and actual IA implementation reviews PASS for coverage, visibility filtering, breadcrumbs and route/authority preservation. Shared source revision changes require current-scope owner review; previous FIX is not copied.

# Validation

75 console tests and typecheck/build PASS; existing bundle-size warning remains. Registry partition test covers all 49 IDs exactly once and checks separate accounting, policy and administration placement. Browser QA PASS on 18 EN/KO/AR desktop/tablet/mobile renders, eight business groups plus lower administration, all 41 sidebar destinations and all five data destinations including S05, preserved service navigation, developer separation, mobile Assistant/Notifications and Enter-key accounting-group expansion. Claude CLI visual review found no blockers in four representative renders; its unobserved mobile utility and S05-visibility conditions were checked in browser. Other workflow-body findings remain outside this navigation change. macOS execution is not Windows/Linux certification. No new production integration is claimed.
