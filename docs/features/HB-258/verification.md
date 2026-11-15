---
type: VerificationRecord
title: HB-258 pre-launch API service skeleton verification
description: Actual checks run for phase 1 of the pre-launch plan (apps/api), plus review outcomes and residuals.
status: final
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
