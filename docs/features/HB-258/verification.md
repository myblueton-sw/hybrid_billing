---
type: VerificationRecord
title: HB-258 pre-launch API service skeleton verification
description: Actual checks run for phase 1 of the pre-launch plan (apps/api), plus review outcomes and residuals.
status: stable
---

# Scope

[HB-258](https://linear.app/hybrid-billing/issue/HB-258) adds `apps/api`, following [plan.md](plan.md):
- `cmd/hb-api` is a loopback-only program with server timeouts, a header limit and graceful shutdown.
- `server` assembles identity, gate, authpolicy and commands with one clock and the sized limits.
- The request boundary checks Host, `Sec-Fetch-Site`, Origin and the body limit, and recovers panics.
- One domain lock and counter serve as the revision of every target.
- Sessions use a cookie (`Path=/api/v1`) and a stable CSRF token.
- The generic command endpoint uses the module's wire schema and status table, with a token-level JSON decoder.
- `GET /session`, `/health` and the adapted `/openapi.json` are served.
- The seed loader consumes the bootstrap (sample), or writes a one-time activation link to an exclusive 0600 file (none).
- The console's Vite proxy sends `/api/v1` to the service.

No screen and no business command is bound yet.

Base: main `96fd245`. Machine: mac_mini.

# Checks run (2026-11-16, this machine)

| Check | Result |
| --- | --- |
| `gofmt -l`, `go vet ./apps/api/...` | clean |
| `go test -race -count=10 ./apps/api/...` | pass |
| `go test -count=40 ./apps/api/server` | pass (no order-dependent tests) |
| `GOOS=linux GOARCH=386 go build`, `GOOS=windows go vet` | pass |
| `go test -race ./...` | every package passes |
| Statement coverage | `apps/api/server` 100.0%; `apps/api/cmd/hb-api` 82.5% (`main` itself, signal wiring, is not unit-tested; `run` is) |
| Console `npm run typecheck`, `npm run build` | pass (worktree uses the primary checkout's `node_modules` through a symbolic link) |
| Manual start, sample seed | `GET /api/v1/health` 200; foreign Host 421; `GET /session` without a cookie 401; `/openapi.json` served |
| Through the real Vite dev proxy (port 5179) | `/api/v1/health` 200; a POST with the proxy's own origin (not in the allowed list) 403 `origin_refused`; a POST with an allowed origin and no session 401 `unauthenticated` |

# Mutation testing

| Run | Mutants | Did not build | Survived |
| --- | --- | --- | --- |
| First | 200 | — | 54 (mostly configuration constants) |
| Final code | 188 | 5 | 2 |

**Killing the first run's survivors.**
- Literal tests pin every limit, default origin and bound.
- Exact-boundary tests:
  - the 1 MiB body, at the limit and one byte over;
  - `Retry-After` rounding, including two-digit seconds;
  - session idle at 29:59 and 30:01;
  - the `as_of` format;
  - the document version.
- Branches that made no difference to the result were simplified.

**After the final run.** A two-digit `Retry-After` case killed one of the two survivors, checked by hand. The other is equivalent: a sort tie that cannot happen, because duplicate field names are refused earlier.

# Reviews

- **Design review** (independent reviewer, security focus, read-only): round 1 FAIL (M1–M6), round 2 PASS with lows folded in.
- **Change review** (a separate independent reviewer, security focus, read-only): PASS, with lows L1–L7 folded in.

Details are in plan.md.

# Residuals

- Login, passwords and the activation screen: HB-259.
- Stale session records that are never accessed again stay until restart. HB-259 adds pruning at login.
- State resets on restart, by design.
- TLS, and on Windows the data-directory ownership and file modes, are not enforced.
- Handlers must call `Revision()` and `Next()` only under the domain lock.

# Planning cross-check (HYB-291)

Added by the [HYB-291 planning cross-check](../../reviews/HYB-291-planning-cross-check/report.md) on 2026-11-17. A record added by HYB-291; it changes no result above except where marked. The findings come from a read-only auditor; the HYB-291 writer agent (delegated by root) wrote this trace, the residual notes and the copy changes; the HYB-291 change review is their first independent check.

## Requirement trace

Source: No screen spec. IDs derived from the phase 1 scope in `docs/features/HB-257/plan.md` (State, Seed, Audit). Status meanings are in the [cross-check report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). No acceptance test was run.

| ID | Status | Reason | Record |
| --- | --- | --- | --- |
| HB-33 | deferred | Covered: commands run through the common command path with idempotency keys and expected revisions; OpenAPI is served. Not built: CLI, AI and external consumer parity, 202 jobs for bulk work, and an API-only business cycle. | HB-258/plan.md |
| HB-03 | deferred | Addressed within pre-launch scope for the assembled gate and command path; registry-wide and leakage acceptance not built. | HB-258/plan.md |
| HB-30 | deferred | Closed by HYB-292: the in-memory audit journal keeps 8 hours, matching HB-257:109. Still deferred: durable storage; the journal resets with the process. | HYB-292/verification.md |
| HB-39 | deferred | Closed by HYB-292 for the command limits: `TestEightHourSession` drives one user through 28,800 mutations at one per second and on past the retention; the limits keep the newest records. Still deferred: the rest of HB-39 (large-scale processing budgets). | HYB-292/verification.md |
| HB-36 | deferred | Loopback service and cross builds exist; TLS, packaging and Windows file modes are residuals. | HB-258/verification.md Residuals |
| HB-42 | deferred | Covered: the identity bootstrap is armed with a generated verifier written only to a 0600 file. Not covered: the rest of the user identity and access lifecycle (later tickets). | HB-258/plan.md |
