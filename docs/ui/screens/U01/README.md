---
type: ScreenImplementation
title: U01 commercial overview implementation
description: Links the planned overview contract to its current complete viewport candidate and verification boundaries.
status: draft
---

# Current screen candidate

The [U01 specification](spec.md) remains the planning contract. The [HB-85 commercial composition](../../../features/HB-85/commercial-overview.md) implements separate money meanings and a compact U01 title/review area, an amount strip and next-work/trend workspace inside the HB-76 service shell. Source code is in `apps/console/src/modules/operations/pages/overview`; canonical route is `/home`.

[Verification](../../../features/HB-85/verification.md) records actual tests and review limits. This candidate remains synthetic with sales/allocation, authoritative receipts, deadlines and assignees unconnected. Per-screen review remains available and pending; this document grants no owner FIX or full U01 completion.


# Build and typography

From `apps/console`, run `npm run build`, `node tools/verify-build.mjs`, `npm test` and `npm start`. Open `http://127.0.0.1:8787/#/home?lang=ko`. U01 overrides load last through `src/main.tsx`. Real KR/Arabic 600 faces use the existing locked local Fontsource dependencies and generated licenses. Their shared availability may affect other screens requesting 600; visual acceptance of those screens remains separate.

The authoritative design/build guides were read from the authorized companion workspace; they are not published by this ticket. Do not substitute an incomplete guide in a fresh clone. The [composition record](../../../features/HB-85/commercial-overview.md) defines this scoped candidate's type/layout rules and [verification](../../../features/HB-85/verification.md) records tested combinations.
