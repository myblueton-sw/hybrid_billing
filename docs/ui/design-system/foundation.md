---
type: DesignSystemReference
title: Screen foundation entry point
description: Routes screen work to the adopted brand foundations while preserving large-data and interaction obligations.
status: draft
---

# Current visual source

[Active design foundations](../../design-system.md) replace this earlier palette/layout proposal for new and revised screen work. Canonical Fusion AI SVGs, supplied tokens, local IBM Plex fonts, square geometry and explicit 16px/RTL/responsive overrides are defined there. Use the [build and work guide](../../design-build-guide.md); do not copy an older palette or serve an unbuilt source folder. Accepted v10 modules are not silently restyled or deleted.

## Interaction obligations retained

- Present purpose, scope, state, filters, action area and evidence/history in a readable sequence. Keep labels and visible focus; placeholders do not replace labels. Status always has text.
- Use bounded server-oriented search, sorting and cursor pages for production. Current page and all-filtered selection are distinct, tied to authorized scope/snapshot, and reviewed before execution. Exact, estimated, unknown, not received and confirmed zero are different states.
- Large exports, recalculation and deletion need async status, partial failure and safe recovery. The design prototype only simulates these; UI duplicate protection is not server idempotency.
- Show currency, measurement unit, precision and source/version. Never infer currency or add unlike currencies without an approved conversion basis. Missing evidence is not zero.
- Keep business approval/issuance/deletion checks visible: entity, target, amount/currency, version, role separation, prior issuance, uncertain external outcomes, retention/hold and reversibility as applicable. Do not invent contract rules from a UI label.
- Required fields need format/help/error guidance. Preserve input across recoverable validation errors; do not retain secrets as ordinary draft text. Concurrent version changes require comparison/review.
- Mobile layouts preserve selection scope, count and primary action. Use local horizontal scroll only for genuinely two-dimensional tables; do not compress complex approval matrices into illegible cells.
- The accessibility target remains WCAG 2.2 AA; normal text contrast 4.5:1, large text 3:1 and meaningful non-text boundaries 3:1 require verification. Accessibility review includes contrast, keyboard/focus, reading order, errors, assistive technology and alternatives to drag-only interaction. Static images alone do not establish conformity.

The current source and browser verification record live in [the scalable console module](../../design-modules/00-scalable-console/r1/verification.md). Native-language, full assistive-technology and production workflow acceptance remain separate.
