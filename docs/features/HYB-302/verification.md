---
type: VerificationRecord
title: HYB-302 verification of the gated console proxy
description: Gated mode now sends every console proxy path to the Go API only, checked by a unit test, a before-and-after preview probe and a security-focused independent review. Run on mac_mini on 2026-11-17.
status: stable
tags: [security, console, proxy, HYB-302]
---

# Results

| Check | Result |
| --- | --- |
| `npx tsc --noEmit` | clean |
| `npm test` | 535 pass, 0 fail (new `proxy-config.test.ts`: every gated target is the Go API on 8820, no gated target on 8793, 8806 or 8810, all nine proxied paths and `/api/v1` present, ungated routes kept) |
| `npm run build` | builds |
| Gated preview probe before the change (Go API on 8820, request with a probe cookie) | `/api/claim-history`, `/api/approval-review`, `/api/credit-case-review` answered by the Python reader on 8806: 403 `origin_denied` |
| Gated preview probe after the change | the same three paths answered by Go: 404 `unsupported_command`; `/api/focus-overview` answered by Go's session gate: 401 `unauthenticated`; `/api/v1/health` 200 |

The change: `apps/console/proxy-config.ts` builds the proxy; in gated mode every known path points to the Go API (an allow-list, so a Python route added later cannot bypass the gate), and ungated development mode is unchanged. `vite.config.ts` uses it and includes it in the build revision. The three paths' preview screens treat 404 as their unconnected state (`claim-history/api.ts`, `approval-inbox/api.ts`, `credits-cases/api.ts`).

# Review

Security-focused independent review (read-only, AI, not owner acceptance): PASS with three Lows; the allow-list and the test were applied and re-reviewed PASS; the build-revision nit was applied.

# Planning cross-check

- `docs/features/HB-257/plan.md:153` ("the Vite proxy then no longer reaches Python directly") now holds for the gated console proxy.
- HYB-301 Required and missing item 13: done.

# Residuals

- Cookies are not port-scoped: a browser still sends the `/api` session cookie on direct navigation to 127.0.0.1:8793, 8806 or 8810, and the 8793 onboarding service serves `/api/local-*` outside the gate (recorded in `docs/features/HB-261/plan.md:92, 96`; HYB-291 M9, owner decision).
- Hypothesis, not verified: the dev server's `fs.allow` covers the repository root, so a Go API data directory created inside the repository could be readable through the dev server's file route. The documented runs use `vite preview`; start `hb-api` with a data directory outside the repository.
