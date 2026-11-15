---
type: ImplementationPlan
title: Pre-launch API service skeleton with module assembly, a domain lock and counter as the revision of every target, a session-bound generic command endpoint, and a seed loader that consumes the bootstrap
description: Phase 1 of the pre-launch plan (HB-257). A standard-library Go program serves the console on loopback, assembles identity, gate, authpolicy and commands with one clock, serializes every domain change under one lock with one counter, runs mutating commands from a session cookie with CSRF and idempotency, and starts from a synthetic seed or an empty installation with a one-time activation link in a 0600 file.
status: final
---

# Scope and ownership

[HB-258](https://linear.app/hybrid-billing/issue/HB-258) is phase 1 of the [pre-launch plan](../HB-257/plan.md). It builds the service skeleton that every later phase extends. It binds no screen and registers no business command. Login, passwords and the session gate in front of the Python endpoints come in HB-259.

Base: main `96fd245`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the design and the change with a security focus, because this is the request boundary. Machine: mac_mini.

Allowed files:
- `apps/api/**`;
- `apps/console/vite.config.ts` (proxy entry only);
- `docs/features/HB-258/*`.

# Observed baseline

- **identity.** `identity.New(installation, limits, clock)` builds the registry. `Installation().ArmBootstrap(verifier, validity, workspace)` arms the bootstrap, and `Boundary().Bootstrap(verifier, user, triple, email)` consumes it. `Boundary().Access(verifier, workspace)` returns an `Answer` and refreshes the session's idle time. A `Verifier` is a 32-byte digest that the caller supplies, so raw secrets never enter the module.
- **gate.** `gate.New(registry, catalog)` returns the gate and its `Administration` (clients and grants). Actions are the closed set in `modules/auth`, and kinds and fields come from the catalogue.
- **commands.** `commands.New(gate, limits)` builds the registry.
  - `Register(definition)` adds commands.
  - `Execute(call)` admits a call only with a current `Answer`. It applies idempotency per installation, client and command, and checks the expected revision against `Resolve`.
  - It returns a `Result` with a stable error code.
  - `OpenAPI(title, version)` generates the document.
- **authpolicy.** `authpolicy.NewRegistry(clock)` builds the registry.
- **Console proxy.** The console's `vite.config.ts` lists one proxy entry per Python path, each with `changeOrigin: true`. So the `Host` a backend sees is the target's own host and port, while the browser's `Origin` is the console's.
- **Source rules.** No module imports a package path with a dot, and every module is standard library only. `apps/api` imports the modules by their full path, so it uses a separate source-rule test that allows module imports and forbids everything else outside the standard library.

# Decision

## Program and configuration

`apps/api/cmd/hb-api` parses these flags:

| Flag | Default | Rule |
| --- | --- | --- |
| `--listen` | `127.0.0.1:8820` | The host must be a loopback address (`127.0.0.1`, `::1` or `localhost`); anything else is refused at start |
| `--console-origin` | `http://127.0.0.1:5173`, `http://localhost:5173`, `http://127.0.0.1:4173`, `http://localhost:4173` (the dev server and `vite preview`) | Exact origins; giving the flag replaces the defaults; up to 4 |
| `--data` | `./.hb-api` | Created with mode 0700 if missing. If it exists, it must be a directory, not a symbolic link, owned by the current user, with no group or other permissions (checked on Unix-like systems; on Windows these checks and file modes are not enforced, a stated residual). It holds only the one-time link file; no state |
| `--seed` | `sample` | `sample` or `none` |

The program logs one JSON line per event to stderr, never with secrets, cookies, tokens, verifiers or request bodies. It exits non-zero with a clear message on any start error.

Server limits:
- read-header timeout 5 s, read timeout 15 s, write timeout 30 s, idle timeout 60 s;
- request body 1 MiB and header size 16 KiB.

## Request boundary

Every request passes, in order:
1. **Host.** The `Host` header must be in the allowed set built from the bound port `p`. Otherwise `421 misdirected_request`.
   - Listening on `127.0.0.1` or `localhost`: the set is `127.0.0.1:p` and `localhost:p`.
   - Listening on `::1`: the set is `[::1]:p` and `localhost:p`.
2. **Origin.**
   - A mutating method (`POST`, `PUT`, `PATCH`, `DELETE`) must carry an `Origin` equal to one of the console origins. Otherwise `403 origin_refused`.
   - A request whose `Sec-Fetch-Site` is present and is anything other than `same-origin` or `none` is refused the same way. Pages on another loopback port are `same-site` and are refused.
   - `GET /api/v1/session` must also carry the header `X-HB-Client: console`. A cross-origin page can only send that header after a CORS preflight, which never succeeds.
   - No `Access-Control-*` header is ever sent.
3. **Recovery.** A panic in a handler becomes `500 internal_error` with only the request ID. The stack is logged without request content, and the domain lock is released by `defer`. `http.ErrAbortHandler` is re-panicked so the server aborts the response as Go intends.
4. **Request ID.** For commands, the request ID is the client's `request_id` field (an identifier), as in the module's schema. It is echoed back but never trusted for any decision. Other requests get a random 16-byte ID. Every response returns it in `X-Request-Id`.

Responses are JSON with `Cache-Control: no-store`, `X-Content-Type-Options: nosniff` and `Referrer-Policy: no-referrer`.

Errors use the module's `Error` schema: `code`, `retry`, `next_action`, `request_id`, and optionally `operation_id`, `fields` and `retry_after_ms`.
- Module codes keep the module's statuses.
- The boundary adds four codes: `misdirected_request` (421), `origin_refused` (403), `csrf_refused` (403) and `body_too_large` (413). Each has `retry: false` and `next_action: none`, and all four are listed in the served document.

## Assembly

`apps/api/server` builds one installation at start:
- **identity:**
  - installation ID `prelaunch`;
  - bootstrap validity 1 hour, invitation validity 7 days, credential validity 30 days;
  - session idle 30 minutes and absolute 12 hours;
  - reauthentication 15 minutes;
  - attempts 5, pending invitations 1,000, sessions per user 10, credentials per service user 10, page size 100.
- **gate:** the catalogue holds the kinds and fields that registered commands need. Phase 1 registers only the kind `installation` for the health and session reads. One console client, `console`, is put active.
- **authpolicy:** an empty registry. Phase 2 seeds the policy.
- **commands**, with every limit set. They are sized so that a reviewer issuing one command per second never reaches a cap: retention keeps the last hour, and older records are swept.

  | Limit | Value |
  | --- | --- |
  | Fields, Value, Rationale | 64 fields, 4,096 bytes per value, 2,000 bytes |
  | Journal, Operations | 1,200,000 each |
  | Workspaces | 2 |
  | WorkspaceJournal, WorkspaceOperations | 600,000 each |
  | Users | 20, more than the seeded users |
  | UserJournal, UserOperations | 30,000 each, more than the rate limit's sustained 4 commands per second over the 2-hour retention (28,800), so even a runaway client loop is throttled by the rate limit, never locked out by the cap |
  | Replay | 1 hour |
  | Retention | 2 hours, longer than replay, so a key reused after an hour gets `idempotency_key_expired` |
  | Burst, Refill | 60, with a refill every 250 ms |

  These satisfy the module's sharing rule: 2 × 600,000 ≤ 1,200,000 and 20 × 30,000 ≤ 600,000. The server is built from this exact configuration at start, and a test builds it too.
- **Clock.** All modules share the system clock in UTC.

## Domain lock and counter

The server holds one `sync.RWMutex` (the domain lock) and one `uint64` counter.

- **Write mode** is taken by the command endpoint around the whole `Execute` call, by each seed step and by each worker step. After the step, the counter is incremented if and only if the step may have changed domain state. For a command, that holds when all three are true:
  - the definition is mutating;
  - the result is not an idempotent replay;
  - the status is `completed` or `accepted`, or the error carries an operation ID. An operation ID means `Apply` was reached, so state may have partly changed even though the result is `internal_error`.

  Reading commands never bump the counter.
- **Read mode** is taken by every read endpoint that returns a revision. The data and the counter are read together under it.
- **Revision.** The revision of every target is the counter written as a decimal string (`r<n>`). `Resolve` of every registered handler returns it. A changing outcome must carry `ResultRevision` equal to `r<n+1>`, the value the counter takes when the step ends. Handlers get it from the server's `next()` helper, and the next read returns exactly that revision.
- **Named failures.** A handler that returns a named failure from `Apply` must leave domain state unchanged. Such a result carries no operation ID, so the counter does not move. The test-only handler has a case for this.
- **No re-entry.** Handlers, seed steps and worker steps never take the lock themselves. The lock is held by the endpoint layer or by the seed or worker driver only. A handler that calls back into the endpoint layer is a design error, refused at review.
- **Not domain changes.** Session refresh by `Access` (taken outside the lock, before the command lock), rate limiting, and idempotency and audit bookkeeping (inside `Execute`, but not counted).

## Sessions

Phase 1 defines the session record that phase 2 creates at login. It is held in memory by `apps/api`:
- a token of 32 random bytes, sent to the browser base64url-encoded in the cookie `hb_session` (`HttpOnly`, `SameSite=Strict`, `Path=/api/v1`, `Secure` when the request arrived over TLS);
- the identity `Verifier`, which is the SHA-256 of the raw token bytes, so the raw token is never kept;
- the workspace;
- a CSRF token of 32 random bytes, made at login and kept for the life of the session.

The cookie path is `/api/v1`, so the cookie is never forwarded to the Python endpoints.

The server looks a session up by the verifier, not the token.

`GET /api/v1/session` returns `401 unauthenticated` without a valid session. With one, it calls `Access`, refreshing the idle time, and returns:
- the user, workspace and role;
- the session's CSRF token, so a reloaded console or a second tab can continue. The token is not rotated, so tabs do not invalidate each other.

## Command endpoint

`POST /api/v1/commands/{command}` with a JSON body in the module's request schema: `request_id`, `target_id`, `body` (an object of string field values) and, for mutating commands, `idempotency_key` and `expected_revision`, plus `rationale` where the command requires one.

1. The session cookie must resolve to a session (`401 unauthenticated`).
2. The `X-CSRF-Token` header must equal the session's CSRF token, compared in constant time (`403 csrf_refused`).
3. `Access` returns the `Answer`.
   - A refusal is `401 unauthenticated`, and the server drops its session record.
   - `identity.ErrUnavailable` is `503 authority_unavailable`, and `identity.ErrInvalid` (which a valid stored workspace never causes) is `500 internal_error`.
4. Under the write lock, `Execute` runs with that `Answer`, the client `console`, the request ID, the command, and the target, key, expected revision, rationale and fields from the body.
5. The HTTP status is `Result.Err.Status` for errors (the module's own table), 200 for completed and 202 for accepted. A `rate_limited` error adds `Retry-After` in whole seconds, rounded up from `retry_after_ms`. The body follows the module's `Result` and `Error` schemas.

The body is parsed with a token-level decoder that refuses unknown fields and duplicate keys. A body that is not exactly one JSON object is `422 invalid_request`, the same code and status the module uses for a malformed call.

## Other endpoints

- `GET /api/v1/health` returns `{"status": "ok"}` without a session. It holds no secret and no activity count.
- `GET /api/v1/openapi.json` returns the module's generated document without a session, adapted for this endpoint: `servers` set to `/api/v1`, the description saying the path is served on loopback for pre-launch, and the four boundary codes added to `x-errors` and to the `Error.code` enum. The conformance test also validates a boundary error body against the document. A test builds a request from the served document for a test command and sends it to the real handler.

## Seed loader

- **Steps.** A seed is an ordered list of named steps. Each step runs under the write lock and returns whether it changed domain state. A failing step stops the start with its name.
- **Activation link file.** In both modes the server generates a random 32-byte bootstrap secret and arms the bootstrap with its SHA-256 for the bootstrap workspace. That workspace is `sample-workspace` with the sample seed, and `workspace` with `--seed=none`.
- **`--seed=none`.**
  - The server writes `activation.txt` into the data directory, holding the one-time link `<first console origin>/activate#<secret>`, with the secret base64url-encoded without padding.
    - The file is created exclusively (`O_CREATE|O_EXCL`), with mode 0600.
    - The server logs only the file's path.
  - A fragment keeps the secret out of request logs; the console reads it in the browser.
  - The file is deleted at the next start, and phase 2 deletes it when the link is used.
- **`--seed=sample`.**
  - The first step consumes the bootstrap itself and creates the administrator `sample-admin`. Its identity triple uses connection `local-password`, the installation as issuer and login ID `admin`, with email `admin@example.invalid`.
  - No activation file is written in phase 1. Phase 2 adds the password-set links for the seeded users.
- **Seed data.** Seed data uses only names marked as samples and the reserved `.invalid` domain.
- **Old link files.** Any older `activation.txt` is removed at start, before a new one is written.

## Console proxy

`vite.config.ts` gains one entry, `/api/v1` → `http://127.0.0.1:8820` with `changeOrigin: true`. The existing Python entries are unchanged in this phase.

## Code layout

Each file is under 1,500 lines, with no explanatory comments.

| File | Contents |
| --- | --- |
| `apps/api/cmd/hb-api/main.go` | flags, start, signals |
| `apps/api/server/server.go` | assembly and the domain lock |
| `boundary.go` | middleware |
| `session.go` | session records |
| `commands.go` | the command endpoint |
| `seed.go` | the seed driver and the bootstrap step |
| `errors.go` | status mapping |
| `source_test.go` | the source rule for apps |

## Residuals

- Login, passwords and the activation screen (HB-259).
- Business commands and screens (later phases).
- State resets on every restart, by design.
- TLS: the service is loopback only, so `Secure` cookies apply only when it is put behind TLS later.
- On Windows, data-directory ownership and file modes are not enforced.

# Verification plan

Literal tests through `net/http/httptest`, on the real handler:
- **Host and Origin.** Every Host and Origin rule:
  - allowed hosts: `127.0.0.1:p`, `localhost:p` and `[::1]:p`;
  - a foreign host;
  - a mutating request without an `Origin`, and one with a foreign `Origin`;
  - a `Sec-Fetch-Site` of `same-site` and of `cross-site`, both refused;
  - `GET /session` without `X-HB-Client`;
  - no `Access-Control-*` header on any response.
- **Limits and recovery.** Body and header limits; panic recovery; the request ID.
- **Sessions.** Unauthenticated, ended and valid sessions; CSRF missing, wrong and correct; the CSRF token stays the same across `GET /session` calls and across two tabs; the `Set-Cookie` path is `/api/v1`.
- **Command endpoint.** Body forms; mapping of each result code to its status; idempotent replay; expected-revision conflict.
- **Domain lock.** Two interleaved commands and a stale read. A session refresh does not change the counter. A replayed result does not change it. A test-only command that reports a change does.
- **Seed.** Both modes; the 0600 mode and the contents of the link file; deletion at the next start; a failing step stops the start.
- **Configuration.** A non-loopback `--listen` is refused at start.
- **Configuration.** The real configuration builds every module at start (identity, gate, commands limits).
- **Audit sizing.** With a test clock, a user issues one command per second for 90 simulated minutes. No cap is reached, because the sweep keeps the journal under the user limit. A separate test fills a lowered cap and checks `dependency_unavailable`.
- **Counter.** A reading command does not change the counter. A handler that changes state and then fails, and a handler that panics, do bump it. A changing outcome's `ResultRevision` equals the next read's revision.
- **Errors and contract.** Every module code is mapped to its status from the module's table. The served document builds a request that the real handler accepts.
- **Data directory.** An older link file is removed first; the exclusive create then refuses only a file that appears after that removal. A symbolic-link directory and group-readable permissions are refused.
- **Concurrency.** Concurrent commands and reads under `-race`.

Checks:
- `gofmt`, `vet`, `go test -race ./...`;
- 100% statement coverage of `apps/api/server`;
- mutation testing of the boundary and the lock logic;
- the 386 build;
- `npm run typecheck` and `npm run build` for the proxy change;
- a request through the real Vite proxy, checking Host and Origin handling;
- a manual start with each seed mode, recorded with `curl` output.

# Review history

- Round 1 (independent security reviewer, read-only): FAIL. Findings and what changed:
  - M1. HTTP statuses come from the module's error table, `Retry-After` is added, and body errors are 422 `invalid_request`.
  - M2. The wire format follows the module's schema, and the served document is adapted and tested against the handler.
  - M3. `CredentialValidity` is set, and the start configuration is tested.
  - M4. Every commands limit is set and satisfies the sharing rule; retention is 1 hour, with a load test and a cap test.
  - M5. The counter bumps only for mutating, non-replayed results that completed, were accepted or reached `Apply`.
  - M6. Changing outcomes carry `r<n+1>`.
  - L1–L9:
    - `Sec-Fetch-Site` must be same-origin or none, `GET /session` needs a custom header, and no CORS headers are sent.
    - One CSRF token per session; cookie path `/api/v1`.
    - Explicit allowed hosts; more default console origins.
    - Link format, exclusive create and data-directory checks, plus a Windows residual.
    - `ErrUnavailable` maps to 503, and ended sessions are dropped.
    - Health carries no count.
    - `ErrAbortHandler` is re-panicked.
- Round 2 (same reviewer): PASS. Low items folded in:
  - R2-L1. Cookie path `/api/v1` everywhere, with a test.
  - R2-L2. A stable CSRF token, tested across tabs.
  - R2-L3. The boundary tests are listed.
  - R2-L4. The `Error.code` enum is extended.
  - R2-L5, R2-L7. Retention 2 hours, longer than replay; user caps above the sustained rate limit over retention.
  - R2-L6. Named failures leave state unchanged.
  - R2-L8. Removal order for the link file.
  - R2-L9. `ErrInvalid` maps to 500.


# Change review resolution

- Change review round 1 (separate independent reviewer, security focus, read-only): PASS with lows, folded in:
  - L1. `OpenSession` takes whether the request is secure and sets `Secure`.
  - L2. Panics log their stack, and net/http's own errors go through the JSON logger.
  - L3. The session slot is reserved before the identity session is opened, and released if that fails, so a full store never leaves an orphaned identity session. Stale records of sessions that are never touched again are a residual for HB-259's login.
  - L4. Tighter tests:
    - five valid origins;
    - the cap error code;
    - Host checked before Origin;
    - every module rule compared with the served document's error table.
  - L5. `run()` tests cover flag, listen, seed, origin, port and data errors, a real start, the header limit and shutdown.
  - L7. Trailing data over 1 MiB is 413. Command results always carry the module's request marker in `X-Request-Id`.
  - L6. `Revision()` and `Next()` read the counter without locking and are only for handlers called under the domain lock. This is a stated rule for later phases.
