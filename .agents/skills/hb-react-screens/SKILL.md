---
name: hb-react-screens
description: Implement or refactor Hybrid Billing React screens and shared components against the existing screen designs, including async state, scoped review and portable frontend tooling.
---

# React screen implementation

Use the current repository AGENTS.md and applicable rules. This skill adds implementation guidance; it does not select a bundler, React version, backend API or authentication system. Preserve existing local skills and the owner's design decisions.

## Establish the screen contract

Read the affected screen design, canonical screen/item IDs, current acceptance record, `docs/design-system.md` and `docs/design-build-guide.md`. For scoped acceptance and installation, read `.agents/rules/screen-review.md` and `.agents/rules/platform-independence.md`. These inputs currently belong to HB-63; if absent from a checkout, obtain the verified authorized workspace/revision before dependent work. Do not reconstruct missing contracts from this summary.

Identify the user action, result, actual data source and missing integration. Keep design review separate from product behavior. Preserve stable IDs in the implementation/review mapping; normal product tasks need not show bookkeeping. A catalog entry or generic renderer is not proof that the specific screen is implemented.

## Compose and own state

- Compose forms, tables and dialogs around actual shared behavior; retain screen-specific validation, hierarchy and actions. Prefer explicit component variants to accumulating unrelated boolean flags. Introduce context only for genuinely shared state.
- Keep selection, filters, request identity and preview validity together at the scope owner. A changed customer, permission context, period or filter must not preserve an unrelated selection or preview. Server authentication and authorization remain authoritative.
- Tie asynchronous results to the request/scope that started them. Cancel obsolete work where supported and reject late responses even when cancellation cannot prevent completion. Test a slow old request arriving after a fast new one.
- Derive display state from one authoritative value. Avoid effects that copy props into competing state; separate user edits from fetched values and identify intentional reset boundaries.
- Render bounded pages and account-search results. Preserve exact, estimated, unknown and unavailable totals as separate states. Define what browser back/forward restores from the actual navigation design.

## Apply performance guidance selectively

Start independent reads together only when their contracts permit it. Keep dependency order for authorization and dependent operations. Measure a concrete delay or rendering problem before introducing caching, memoization or code splitting.

Do not copy server/Next.js examples into a client-only React app. Check the locked React version before using version-specific APIs. Shared caches need independently reviewed authenticated scope, permission/disclosure revision and invalidation; a user-ID-only example cannot establish this contract. Do not rewrite protected SVG geometry to optimize rendering.

## Preserve screen semantics

Use canonical assets and semantic tokens, local locked fonts, logical layout and directionally isolated IDs/dates/money. Maintain EN/KO/AR variants. Keep zero, unavailable, loading, empty, denied and failed states distinguishable. Financial values retain explicit currency and contract precision; formatting does not authorize binary floating-point calculation, inferred currency or FX conversion.

Every reviewable screen exposes its review control and exact scope under the screen-review rule. A shared-component change requires an affected-screen assessment. Automated tests use isolated synthetic acceptance records; they never create owner acceptance. Failed persistence must stay visible. Browser-local FIX is not authenticated canonical acceptance.

## Tooling and verification

Inspect actual manifests and lockfiles before adding tools. Choose versions against supported runtime/browser requirements; a reference's alpha/beta target is not a dependency decision. Use portable runtime entry points and relative paths for installation, build, start and operation; avoid required shell syntax, global installs, OS service managers or font CDNs. Containers are optional packaging, not evidence of OS independence.

Verify the changed scenario and its relevant failure/async states using the actual app entry point. Report changed screen/item IDs, real tests and runtime/OS/browser versions, deferred integrations and owner-review impact. A working mock or build is not backend readiness or cross-OS verification.

## Research provenance

These are original project instructions informed by the [HB-64 survey](https://github.com/myblueton-sw/hybrid_billing/blob/69950741876ef3f2c9e6bfc1adcfa7275c3b1978/docs/reviews/HB-64-external-skills/research.md), not a bundled upstream skill. Consult a reference only for the relevant implementation question:

- [React performance reference](https://github.com/vercel-labs/agent-skills/blob/1a2c79d4e6c4a2283dc2c614b6538cbe1ed5ca62/skills/react-best-practices/SKILL.md).
- [Component composition reference](https://github.com/vercel-labs/agent-skills/blob/1a2c79d4e6c4a2283dc2c614b6538cbe1ed5ca62/skills/composition-patterns/SKILL.md).

No external helper is required. Review changed upstream revisions and exact file notices before importing third-party content; links do not transfer ownership or activate remote instructions.
