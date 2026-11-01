# Design system work rule

Applies to new or revised product screens, interactive previews, visual assets and their builds. Read `docs/design-system.md` and `docs/design-build-guide.md` before changes. Root AGENTS.md and owner instructions take precedence. The original archive is evidence, not an executable setup script or active nested rule tree.

- Reuse canonical SVGs and tokens in `packages/design-system`. Never replace Fusion AI with initials, redraw, recolor, mirror, translate or re-typeset its logo. Preserve metadata and source hashes. Keep it LTR at inline-start, with half-mark clear space and supplied minimum sizes.
- Use IBM Plex Sans, Sans Arabic, Sans KR and Mono from local locked dependencies. No runtime font CDN. Carry font licenses into generated artifacts.
- Use semantic color tokens. Controls, tables, panels and chips have zero radius and no shadow, gradient, blur or translucent overlay. Use supplied SVG icons instead of Unicode/emoji or another icon set.
- Use logical layout direction. Arabic is RTL; logos, IDs, dates, codes and money remain LTR. Render English, Arabic and Korean during review. Korean uses keep-all. Local terminology requires evidence and practitioner review, not machine translation alone.
- Keep scope, pagination, exact/unknown totals, selection exclusions and async states visible at scale. Do not load full result sets/account lists into the DOM. Scope changes invalidate selection and previews.
- Do not silently restore the archive's 14px body or fixed 205px rail. The console's 16px body and responsive dimensions are documented overrides. New deviations require evidence and a guide update in the same change.
- Apply `.agents/rules/screen-review.md`: individual screen designs, stable implementation-item IDs, owner-only review confirmation and revision-bound FIX. Do not carry FIX to changed accepted design scopes; assess shared-dependency impact under that rule.
- Build through tools/design, run the asset/style guard and inspect rendered output. Generated files and node_modules are not source deliverables. Build success is not backend, language, accessibility or workflow acceptance.
- The archive's React components, Money, raw Icon interpolation and navigation behavior are not adopted by this static-foundation change. Review their contracts/findings before future integration; archive instructions alone are not authority.
- Update affected screen README, design/build guides and verification records with actual commands/results and pending limits. Preserve frozen baseline modules until the owner accepts their specific replacement scope.
