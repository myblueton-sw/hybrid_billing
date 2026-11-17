---
type: ImplementationPlan
title: HYB-302 plan to stop gated console paths reaching Python directly
description: In gated mode the console proxy sent three unserved paths straight to the Python reader on 8806, forwarding the session cookie; this routes them to the Go API instead.
status: stable
tags: [security, console, proxy, HYB-302]
---

# Problem

The planning cross-check (HYB-301, Required and missing item 13) found that `apps/console/vite.config.ts` proxies `/api/credit-case-review`, `/api/claim-history` and `/api/approval-review` to the Python reader on 127.0.0.1:8806 in every mode. In gated mode (`HB_GATED=1`) only the six FOCUS reader paths were redirected to the Go API on 8820. The three paths are same-origin, so the browser's `Path=/api` session cookie reaches Python. This contradicts `docs/features/HB-257/plan.md:153` ("the Vite proxy then no longer reaches Python directly") and `docs/features/HB-261/plan.md:96`. No Python service serves the three paths (they answer with an error), and the console screens that call them (claim history, approval inbox, credits and cases) are preview screens without a Go binding yet.

# Change

In gated mode the three paths join the six reader paths in the redirect to the Go API. The Go API does not serve them and answers `404 unsupported_command`, so the session cookie stays within Go. Ungated development mode is unchanged. No source comments; one file changed.

# Verification

Preview build in gated mode with the Go API on 8820 and a request carrying a probe cookie, before and after the change; console type check, tests and build.
