---
type: ImplementationDesign
title: Service shell reset and developer-tool separation
description: Rebuilds the shared product entry and navigation after the owner rejected screen-directory tooling as a commercial service.
status: draft
---

# Owner correction and responsibility

The owner explicitly rejected the previous results as commercial service screens and requested a restart. Screen search and Work home were identified as development convenience, not the intended customer experience. Existing test and review passes remain narrow technical evidence; they do not establish product acceptance. Root owns PM, implementation and QA. Claude CLI supplies visual composition and independent reviewers inspect boundaries and changes.

HB-76 is the shared-shell implementation ticket. Its branch starts from current origin/main with pending HB-91 dependencies explicitly incorporated. This is the first corrective slice of the commercial-service rebuild, not a claim that all 49 screens or production operations are complete.

# Product and developer boundary

The historical shell slice below has been extended by the owner-authorized [commercial IA](commercial-ia.md), which is now the current navigation composition. Earlier direct U01–U04 placement and domain-group ordering describe the first corrective slice, not the active eight-group mapping.

Service entry is the integrated billing overview at canonical U01 /home. U02–U04 retain their business destinations without a Work home category header. Other canonical business groups and visibility checks remain. No screen search, screen IDs, legacy mapping, role simulation or screen directory appears in ordinary service navigation.

Explicit review=1 enables developer presentation only. Direct design routes require that mode; the flag grants no backend identity or authority. Leaving the mode resets simulation and effective service state does not inherit it. Contribution help remains separately reachable. The compact per-screen review control retains exact revision and scope under the owner's existing requirement; no assistant-created FIX.

Shared navigation preserves language, data mode and local service selection, but discards object IDs, snapshots, row ordinals and screen-specific filters. Existing dirty-navigation protection remains. Unsupported data paths remain visibly unconnected or synthetic; changing the shell does not establish authentication, commercial readiness or live overview aggregation.

# Design evidence

Consulted official Carbon global-header and left-panel guidance on 2026-11-03:
- https://www.carbondesignsystem.com/building-blocks/core/patterns/global-header
- https://www.carbondesignsystem.com/building-blocks/core/components/ui-shell-left-panel/guidelines

Adopt consistent product navigation, task grouping, no third navigation tier and context-preserving transitions. Do not import Carbon code/assets or override the canonical Fusion AI brand, full-width layout, neutral square borders, local IBM fonts or EN/KO/AR directionality.

# Verification and remaining work

Independent boundary design review conditionally passed with explicit developer gating, simulation separation, preserved service mode and unchanged backend authority. Implementation review and rendered validation remain pending.

The commercial-service rebuild still requires complete business forms/actions, persisted setup and handoff contracts, authentic identity/workspace integration, real integrated overview aggregation, production permission checks and complete task journeys. These are required product work, not replaced by shell polish or screenshot counts. This record does not declare them complete.

# Implemented shell and evidence

ServiceNavigation now renders direct U01–U04 business links and canonical domain groups without screen search or IDs. The default entry is labelled Billing overview. Developer mode explicitly gates directory routes, simulation controls, legacy mapping and data-mode controls. Contribution remains available through help. Effective service audience/persona/scenario ignore developer simulation, and mode exit clears it. Shared navigation retains service=local while discarding object-specific context. Existing owner review remains compact with its required ID/revision and modal scope.

Claude CLI read the installed frontend-design skill and supplied the composition proposal. Root retained U02–U04 direct links and the existing exact-revision review contract instead of removing required metadata or inventing a new drawer. The visual review gave conditional PASS for the shell with no shell blockers, requesting desktop menu-height and mobile-utility confirmation. The sidebar now stretches with the page while its navigation sticks and scrolls independently; tablet layout removes the unused rail column. Body workflows remain outside this shell slice.

Independent authority implementation review initially failed loss of exit controls in the prelogin developer preview. Developer controls and a guarded Return to service action were restored; targeted source re-review PASS. This never establishes authentication or customer authorization.

Actual verification on macOS arm64, Node 26.7.0 and Chromium headless shell 1243:
- 74 console tests PASS, including direct developer-route gating, contribution boundaries and service-query preservation.
- Typecheck/build PASS with the existing bundle-size warning.
- 18 EN/KO/AR renders across overview and connection screens at 1440, 900 and 390 px PASS for overflow, logo, service/developer visibility and review access.
- Actual connections-to-onboarding navigation retains local-service mode; directory denial without review=1, developer exit simulation reset, prelogin return, mobile menu Escape and language-control presence PASS.
- Desktop rail/page height equality and sticky navigation are asserted in the browser suite.
- Screenshots and raw reports remain under output/validation/HB-76, excluded from Git.

No commercial-service completion, owner FIX, main merge or deployment is claimed. The shared source revision changes candidate review scopes; prior acceptance is not carried forward. Full workflow redesign and backend capability gaps remain active product work.
