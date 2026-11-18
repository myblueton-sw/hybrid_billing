---
type: ImplementationPlan
title: Session gate in front of the collected FOCUS readers, with a shared secret the gated reader requires and a console proxy switch
description: Phase 2c of the pre-launch plan (HB-257). The API service forwards the six collected FOCUS reader paths to the loopback readers only after the session, identity access and a gate view permission are checked, and the reader can require a shared secret so local clients cannot go around the gate.
status: stable
---

# Scope and ownership

[HB-261](https://linear.app/hybrid-billing/issue/HB-261) is the third part of phase 2 of the [pre-launch plan](../HB-257/plan.md), which says:

> Phase 2 puts them behind the Go session. `apps/api` forwards `/api/focus-*` … only after the session and the read permission (gate action `view`, with the focus data kind added to the gate catalogue in phase 2) are checked. The Python services require a random shared secret header that only `apps/api` sends.

Base: main `84685ef`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the change with a security focus. Machine: mac_mini.

Allowed files:
- `apps/api/**`;
- `apps/console/vite.config.ts` (proxy only);
- `modules/focus_import/overview_http.py`;
- `tools/focus_overview.py`;
- `tests/focus_overview/test_http.py`;
- `docs/features/HB-261/*`.

Out of scope: the local onboarding service on port 8793 (`?service=local`), which serves its own pages from its own origin.

# Observed baseline

- `modules/focus_import/overview_http.py` serves six GET paths:
  - on port 8806 (overview server): `focus-overview`, `focus-commitments` and `focus-marketplace`;
  - on port 8810 (receipts server): `focus-receipts`, `focus-connections` and `focus-collection-status`.

  It requires `Host` to be its own origin and `X-HB-Local-Client: 1`. It allows `Origin` only from itself and the UI port, and `Sec-Fetch-Site` only `same-origin` or `none`.
- The console's Vite proxy sends those paths directly to the readers, so any local client can reach them without a session.
- `apps/api` sessions use the cookie `hb_session` with `Path=/api/v1`. The browser therefore does not send it with `/api/focus-*` requests.
- At the time of this ticket, the readers' data stack (PostgreSQL, ClickHouse and SeaweedFS on ports 25432, 28123 and 28333) was not running. The owner's long-running readers on 8806 and 8810 returned `503 collected_unavailable` for every path.

# Decision

## Gate in `apps/api`

**Routes.** For each configured reader path, `apps/api` registers a GET route:
- the defaults are the six paths above, with their default ports;
- `--reader path=http://127.0.0.1:port` replaces the defaults;
- each target must be `http://127.0.0.1:<port>`, with no path, query or user info. The reader binds only 127.0.0.1 and checks `Host` exactly.

**Checks, in order:**
1. a session cookie, else `401 unauthenticated`;
2. `identity.Access`, mapped as for commands;
3. a gate decision for action `view`, kind `focus_data` and field `rows` in the session's workspace, else `403 permission_denied`.

The catalogue always contains this kind and field.

**Forwarding.** It uses `httputil.ReverseProxy` with `Rewrite`:
- **Target.** The target URL, and `Host` set to the target's host.
- **Headers sent.** Only `Accept`, `Accept-Language` and `If-None-Match` are forwarded, plus `X-HB-Local-Client: 1` and `X-HB-Gate-Secret: <secret>`. Nothing else goes through:
  - no cookie, `Origin`, `Sec-Fetch-*` or CSRF header;
  - no client-supplied secret;
  - no `X-Forwarded-*` header.
- **Response.** `Set-Cookie` and `Access-Control-*` headers are removed from the reader's response.
- **Errors.** An unreachable reader is `503 dependency_unavailable`.

**Methods.** Every method except GET, including HEAD, gets `404 unsupported_command`.

**Timeout.** The reader transport has a 25-second response-header timeout, so a hung reader cannot hold a request past the server's write timeout. Environment proxies are ignored.

**Request ID.** An unreachable reader's 503 carries the request's ID.

**Secret.** At start, `apps/api` writes a new random 32-byte secret (base64url) to `reader-secret` in its data directory: the old file is removed first, and the new one is created exclusively with mode 0600. This happens whenever readers are configured.

**Seed.** With the sample seed, the console client and both sample administrators get view grants on `focus_data` in the sample workspace, valid for 30 days of uptime. With `--seed=none` there are no reader grants, so the readers answer 403 until grants can be managed (phase 3).

**Cookie path.** The session cookie path widens from `/api/v1` to `/api`, so the browser sends it with the reader paths.

## Reader secret

- `OverviewServer` gains an optional `gate_secret_file`; `tools/focus_overview.py` gains `--gate-secret-file`.
- When it is set, every request must carry exactly one `X-HB-Gate-Secret` header that equals the file's content (stripped), compared with `hmac.compare_digest`. Otherwise the answer is `403 origin_denied` and no query runs.
- The file is re-read when its modification time, size or inode changes. A missing file, or a secret shorter than 32 bytes, refuses everything.
- When it is not set, the reader behaves exactly as before, so the existing review workflow is unchanged.

## Console proxy

`vite.config.ts` routes the six reader paths to `apps/api` (port 8820) when `HB_GATED=1`. Otherwise it keeps the direct reader routes for the existing review shell.

The pre-launch start uses `HB_GATED=1` and starts the readers with `--gate-secret-file <apps/api data>/reader-secret`.

# Residuals

- **The start script.** One command that starts `apps/api`, the readers with the secret file and the console with `HB_GATED=1` comes with phase 10.
- **Non-gated mode.** Without `HB_GATED=1`, a signed-in browser's `/api` cookie also travels through the Vite proxy to the direct readers. The readers ignore cookies.
- **Data stack.** The real reader data stack was not running, so no reader returned 200 during this ticket. The gate's forwarding was verified against a test upstream, and against the real reader's own 503.
- **Local onboarding service.** The service on 8793 stays outside the gate.
- **The gate needs two opt-in switches.** It holds only when the console runs with `HB_GATED=1` and the readers run with `--gate-secret-file`. Until the phase 10 start script sets both, a reader started without the secret can still be reached directly by a local client. The secret is now generated by `apps/api`, not by a start script as the phase plan first said.
- **Same-user trust boundary.** Any process running as the same OS user can read `reader-secret`.
- **No workspace binding in the reader.** The reader serves one fixed dataset whatever the session's workspace. Only the sample workspace has grants today, so this is not reachable, but granting a second workspace would expose the same data across tenants. This is a tenant-security residual: the reader must be bound to the workspace before any second workspace gets reader grants.
- **Widened cookie path.** With `Path=/api`, the HttpOnly session cookie also goes to other local listeners' `/api/*` paths on 127.0.0.1, because cookies are not port-scoped. That includes the onboarding service on 8793 and the direct readers when the console is not gated. None of them reads or logs cookies.

# Verification plan

- **Go tests:**
  - routes;
  - the order of the checks;
  - an unauthenticated request, an ended session and a missing grant never reaching the upstream;
  - exactly which headers are forwarded and stripped, including a forged client secret;
  - response header filtering;
  - an unreachable upstream;
  - configuration validation;
  - the secret file's mode, length and replacement failure;
  - the `--reader` flag;
  - 100% coverage of `apps/api/server`.
- **Python tests:**
  - the secret required, wrong, short, rotated, missing and duplicated;
  - the ungated reader unchanged.
- **Manual end to end:**
  - `hb-api` with `--reader` pointing at a gated reader started from the worktree code, on a spare port, with the primary checkout's local configuration;
  - anonymous requests (401), signed-in requests (forwarded), and direct requests without the secret (403);
  - the Vite dev proxy with `HB_GATED=1` returning the gate's 401 even with `X-HB-Local-Client: 1`.

# Review history

- 2026-11-16, design and change review together (separate independent reviewer, security focus, read-only).
  - This review came after implementation, not before it, which departs from the required design-first order. It is recorded here and in the verification record.
  - Result: PASS, with lows folded in:
    1. a response-header timeout;
    2. the request ID on the 503;
    3. HEAD refused;
    4. the residuals for the opt-in switches and the same-user boundary;
    5. the workspace-binding residual;
    6. grant validity decoupled from links (30 days);
    7. targets limited to 127.0.0.1;
    8. the cookie-path residual widened to all local listeners;
    9. no-op test lines removed, and tests added for hop-by-hop and forwarded headers, HEAD, the request ID and secret replacement on restart.

