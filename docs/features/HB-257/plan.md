---
type: ImplementationPlan
title: Pre-launch phase plan for a Go API service that assembles the domain modules in memory with synthetic seed data, signs people in locally, and backs the console's unconnected screens
description: Phase plan requested by the owner on 2026-11-16. It defines the service runtime, in-memory state with a synthetic seed, sign-in, the screen binding order, the ticket sequence and what stays out, so that a pre-launch build runs end to end for the owner to review. Durable storage waits for the owner's database decision.
status: final
---

# Scope and ownership

The owner's request on 2026-11-16: "남은거 작업해야지 출시전 화면까지 하고 그 다음에 내가 보면서 개선해야되어서" ("the remaining work has to be done, up to the screens before launch, and then I will review and improve them").

[HB-257](https://linear.app/hybrid-billing/issue/HB-257) writes and reviews this phase plan. It contains no code. Each phase below becomes its own ticket that follows the mandatory sequence.

**Pre-launch** means:
- a build that one operator can start locally;
- someone can sign in to it;
- the console screens read and change state held by the Go modules, through the service, as defined in the completion criteria below.

State does not survive a restart in this phase. See the state section.

It is not a release. It grants no deployment, real accounts, payments, messages, customer data or production topology.

Root (Claude Code) owns planning, implementation and integration. Independent read-only reviewers review each design and change. Security and permission phases get a security-focused review, and billing and data phases a DBA-focused review. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-16 at main `f468d64`:

- **Go modules.** There are 66 standard-library Go packages under `modules/`. Each is an in-memory register with a clock and its own proposal rules. None is served over HTTP or persisted; there is no `package main` and no `net/http` import.
- **Live screen data.** Every `/api/*` call the console makes today goes to Python loopback servers:
  - `modules/focus_import/overview_http.py` on ports 8806 and 8810;
  - `modules/focus_local/http_service.py` on port 8793, through `tools/console_service.py`.
  - These read collected FOCUS sample data in PostgreSQL, ClickHouse and SeaweedFS, run from `deploy/local/verification/compose.yaml`.
- **Screen status.**
  - All 49 registered screens are routed in `apps/console`.
  - 9 read real data through Python.
  - B09, B10 and B11 call endpoints with no producer and fail closed.
  - 32 are verified, unconnected previews. `docs/features/HB-144/completion.json` lists them as `producer_binding: not-implemented`: A01, B02, C01–C03, I01–I04, L01–L03, M01–M05, O01–O03, P01–P07, R01, R02, S03 (older view), U03 and U04.
- **Owner decisions.** `docs/architecture/current-design-decisions.md` (2026-10-19) sets Go for the backend APIs and the policy server, Python for processing, and React for the frontend. It requires a customer-operated deployment with no mandatory cloud services, and Docker containers, with Compose proposed locally.
- **Open decisions.** The local, untracked v1.16 planning documents leave the database product and version, the migration tool and the production topology unadopted (`architecture.md`, `repository-structure.md`). The local proposal lists `postgres:18` as a candidate.
- **Existing building blocks.**
  - `modules/commands` (HB-176 and HB-177): a command registry and executor behind `modules/gate`, with idempotency, expected revisions, an audit journal, retention, rate limits and OpenAPI generation.
  - `modules/identity` (HB-172) and `modules/gate` (HB-173): users, sessions and permission decisions.
  - `modules/authpolicy` (HB-255): sign-in policy.
- **Planned layout.** The local, untracked `docs/planning/v1.16/repository-structure.md` places HTTP, authentication middleware and module assembly in `apps/api/`, and storage adapters in `adapters/storage/`, the latter "after the physical product is chosen".

# Decision

## Service runtime

- **Program.** `apps/api` is one Go program, `apps/api/cmd/hb-api`, on the standard library only. It uses `net/http` with the Go 1.22 method and path patterns.
- **Listening.**
  - It listens on `127.0.0.1` by default, with the port configurable.
  - Requests whose `Host` is not an allowed loopback name and port, including the console host the proxy forwards, are refused, and mutating requests whose `Origin` is not the allowed console origin are refused (DNS rebinding and cross-site requests).
  - The console's Vite proxy sends every `/api/*` path to it, so the cookie is same-origin.
- **Assembly.**
  - At start it builds every module register it serves, with one shared clock: identity, gate, authpolicy and commands from phase 1, and the rest as each phase adds them.
  - Phase 1 includes identity assembly and the bootstrap of the first administrator, so every later phase can sign in.
- **Mutations.**
  - Every mutating endpoint is a `modules/commands` definition. Its handler calls the owning module.
  - **One mutation path.** A *domain change* is a change to domain state made by a command, a pre-sign-in identity endpoint, a seed step or a worker step. Session refresh by `identity.Access`, rate-limit buckets, and idempotency and audit bookkeeping are not domain changes.
  - **Lock and counter.**
    - Every domain change runs under one installation-wide read-write lock, held in write mode for the whole `Execute` call, from the first `Resolve` through `Apply`.
    - Each applied domain change increments one installation-wide counter.
    - A worker step increments the counter only when the module call it makes reports a change (an applied proposal, a recorded result, a state transition). Tests check both directions: an active worker bumps the counter and an idle one does not.
    - Password hashing for login and password set runs outside the lock; the lock is taken only to record the result. Each login is still a domain change, so other users' open forms may see a conflict after someone signs in, and must reload.
    - A read that returns a revision takes its data and the counter under the same lock in read mode.
    - Handlers and workers never take the lock themselves (Go locks are not re-entrant). Only the endpoint layer and the worker loop take it.
    - Serializing all domain changes is acceptable for a single-operator pre-launch build.
    - Tests interleave two commands and a stale read, and check that a session refresh or an idle worker does not change the counter.
  - `Resolve` returns the current counter as the revision of every target. A change anywhere therefore invalidates outstanding expected revisions: clients may see spurious `revision_conflict` results and must reload, but an update can never overwrite a change the client did not see.
  - Where a module method takes its own expected version or epoch (for example the identity installation and membership versions), the handler reads that value from the module under the same lock and passes it. The module's own optimistic check therefore always passes, and the installation counter is the only guard against lost updates; reviews must not count the module check as a second guard.
  - This gives every endpoint idempotency, expected revisions, rate limits and the audit journal.
  - The executor's `Answer` comes from `identity.Access` with the request's session.
- **Identity calls.**
  - Identity calls made before sign-in (bootstrap, password login, activation and reset consumption) are plain endpoints. They have their own failure limiter keyed by account and client address.
  - Identity administration that the identity module authorizes by session verifier (invitations, roles, memberships, service users, credentials) is also wrapped as command definitions. Their handlers pass the caller's verifier to the identity method.
  - Installation-level user operations (`CreateUser`, `SuspendUser`, `ReinstateUser`, `DepartUser`) take a free-text operator and act on users across the installation. The gate's closed action set (`modules/auth`) has no action for them, and identity has no installation-wide role.
    - Phase 3 therefore starts with a separately reviewed change, with a security-focused review. It adds an installation administration action to `modules/auth` and the gate catalogue. It defines the installation administrator as an administrator of the bootstrap workspace, which in `modules/identity` is the only workspace with members. The change checks the bootstrap workspace ID explicitly rather than relying on that invariant, and a test fails if a second workspace ever gets an administrator.
    - Until that change is merged, these operations are not exposed.
    - Once it is, their command definitions use that action, are decided by the gate for the bootstrap workspace only, and fill `Operator.Reference` from the authenticated Answer's user, never from client input. A workspace administrator of any other workspace can never reach them.
- **Reads.** Reads call the module's read methods after the gate decides the read permission.
- **Representation.** JSON over HTTP. Errors use the stable catalogue from `modules/commands`, and decimals stay strings, following the numeric contract.
- **Contract.** The OpenAPI document is generated from the command registry and served at `/api/v1/openapi.json`.
- **Errors and timeouts.** The server sets request size and time limits, recovers panics into `internal` errors without stack traces in responses, and logs to stderr in structured form without secrets.
- **Portability.** It builds for Linux, macOS and Windows. The tested combination is macOS arm64, until CI exists.

## State: in memory, with synthetic seed data; durable storage waits for the database decision

The modules keep their state in memory and expose no serialization. Making that state durable needs either a storage design per module or a generic journal.

The independent review of the first draft of this plan showed that a generic journal replayed through the gate cannot be made deterministic without changing many modules:
- sessions and Answers are bound to the process;
- reads extend sessions;
- operation identifiers and other values are random;
- apply order is not append order.

Durable storage is therefore tied to the database decision, which is the owner's (open in the v1.16 planning documents). It will be designed per module with that decision.

For pre-launch:
- **State.** All state lives in memory for the life of the process. A restart starts from the seed again, and every session ends with it.
  - The console shows a fixed notice on every screen, in English, Korean and Arabic, that the build is a pre-launch build whose data resets on restart.
  - The notice's copy follows the terminology guide.
- **Seed.**
  - At start, a seed loader builds one synthetic installation: one workspace, organizations, contracts, users of each role, a policy, retention classes, sample billing periods, invoices, claims, cases and release-trace records.
  - It goes through the same mutation path as the endpoints, as the bootstrap administrator or as named system actors. It holds no real customer data, and every name is marked as sample.
  - Each phase extends the seed for its screens, and the seed is covered by tests.
  - The seed can be turned off to start from an empty installation (`--seed=none`).
- **Audit.** The command audit journal lives in memory too, and resets with the process. This is stated in the notice and in the residuals. Its capacity and retention limits (HB-177) are set in phase 1 to numbers sized for an eight-hour review session at 1 command per second (at least 28,800 records, with retention keeping the newest). A test drives the journal to that bound and checks that commands still run. The idempotency store (`Operations`) and the rate limits (`Burst`, `Refill`) are sized and tested the same way.

## Workers and system actors

Modules that expect system actors (retention workers, harnesses, ingestion) are driven, where a screen needs it:
- by a worker loop inside `apps/api`, under named system actors that the seed registers, through the same mutation path;
- or by a seed step.

A phase that needs a worker says which. A worker never runs against real external systems.

## Sign-in and sessions

**First administrator.**
- At start, `apps/api` arms the identity bootstrap with a verifier it generates.
- It writes a one-time activation link, holding that verifier, only to a file in the data directory with mode 0600, and prints only that file's path. The verifier never appears on the command line or in logs.
- Each link is invalidated as soon as it is used. The file is deleted when every link in it has been used or has expired, and at the next start.
- With `--seed=none`, the operator opens the link to create the first administrator (L02).
- With the default seed, the seed consumes the bootstrap itself and creates the administrator and the sample users. The 0600 file then holds one-time password-set links from `modules/passwords` (not the bootstrap verifier) for the administrator and each sample user. Each start writes new links, because state resets.

**Passwords.**
- Passwords live in a new module, `modules/passwords`, reviewed in phase 2. A password identity is the triple with connection `local-password`, the installation as issuer and the login ID as subject. The module maps it to a PBKDF2-HMAC-SHA256 hash from the standard library's `crypto/pbkdf2`:
  - a 16-byte random salt;
  - 600,000 iterations, the OWASP minimum, provisional;
  - stored with algorithm, iterations, salt and hash, and compared in constant time.
- Passwords are never logged and never returned.

**Login (L01).**
- Login checks the password, then `authpolicy.Evaluate` with the `password` method, then opens an identity session.
- For an unknown account it runs PBKDF2 against a dummy hash, so the response and its timing do not depend on whether the account exists.
- Failures are limited per account and globally. Through the local proxy every request comes from loopback, so `X-Forwarded-For` is never trusted.

**Session.**
- The session token is random, 32 bytes, in an `HttpOnly`, `SameSite=Strict` cookie, marked `Secure` when served over TLS.
- Each session has a CSRF token, returned at login and by `GET /api/v1/session` (so a reloaded console can get it again), that every mutating call must echo in a header.
- The login form itself carries a pre-session CSRF token bound to a short-lived cookie.

**Other.**
- Recovery (L03) is administrator-assisted only: an administrator issues a one-time reset that the user consumes. Self-service recovery is an owner decision.
- SSO, LDAP and MFA verifiers are residuals. Their policy hooks already exist in `modules/authpolicy`.

## Existing Python endpoints

The 9 screens that read collected FOCUS sample data stay on the Python services in this phase plan.
- Phase 2 puts them behind the Go session:
  - `apps/api` forwards `/api/focus-*`, `/api/local-*` and the other existing Python paths to the loopback Python ports only after the session and the read permission (gate action `view`, with the focus data kind added to the gate catalogue in phase 2) are checked. The Vite proxy then no longer reaches Python directly.
  - The Python services require a random shared secret header that only `apps/api` sends, so a local client cannot go around the gate. The phase 2 start script generates the secret into a 0600 file before starting either side. `apps/api` and the Python services read it from that file, and a Python service re-reads it when a request carries an unknown secret and the file's modification time has changed, so restarting `apps/api` alone with a new file still works.
- The Python services and the compose stack (PostgreSQL, ClickHouse, SeaweedFS) remain a development dependency for those screens, not an adopted database.
- Porting them to Go is a residual.

## Screen binding

Each screen binding replaces the preview's in-memory data with calls to `/api/v1`, keeps the reviewed layout and copy, and fails closed when the service is unreachable. The order follows dependencies:

| Phase | Tickets | Screens | Modules |
| --- | --- | --- | --- |
| 1 | HB-258 | none (service skeleton, assembly of identity, gate, authpolicy and commands, bootstrap, seed loader, health, OpenAPI) | commands, gate, identity, authpolicy |
| 2 | HB-259 (sign-in API), HB-260 (L01–L03), HB-261 (session gate in front of the Python readers) | L01, L02, L03, plus the session gate in front of the Python endpoints | identity, authpolicy, passwords (new) |
| 3 | HB-262 (installation administration action), HB-263/264 (M01), HB-265 (M02), HB-266/267 (M03), HB-268/269 (M04), HB-270/271 (M05) | M01–M05 | identity, auth, authpolicy |
| 4 | HB-272/273 (O01), HB-274/275 (O02), HB-280/281 (O03) | O01–O03 | organizations, relationships, contracts, customeraccess |
| 5 | HB-282/283 (P01), HB-284/285 (P02), HB-286/287 (P03), HB-288/289 (P04), HYB-290 (P05); P06 and P07 not yet ticketed | P01–P07 | installer, plugins, webhooks, recovery, retention, releasetrace, perfgate, simulation, sandbox, licensemode, thirdparty |
| 6 | not yet ticketed | B02, B09, B10, B11 | meters, approvals, protected, adjustments, claims, corrections, cases, balances |
| 7 | not yet ticketed | I01–I04 | issuance, delivery, erpledger, erpadapters, exports, disclosure |
| 8 | not yet ticketed | C01–C03 | customeraccess, issuance, cases |
| 9 | not yet ticketed | R01, R02, U03, U04, A01, S03 (older view) | analytics, history, operations, incidents, assistant |
| 10 | not yet ticketed | none (end-to-end pre-launch QA in English, Korean and Arabic, handover) | all |

The ticket column was corrected by [HYB-291](../../reviews/HYB-291-planning-cross-check/report.md) on 2026-11-17. The first version gave one placeholder number per phase (HB-258 to HB-267); phases were split into API and screen tickets as they ran, so those numbers went to other work. Linear tickets later moved from team HB to team HYB with the same numbers (HYB-n is HB-n); records written before the move keep the HB prefix.

A phase that turns out too large is split into more tickets in the same order. Each binding ticket:
- records which screen fields come from which module read or command, extending `docs/ui/screen-binding-register.md`;
- adds the handlers, commands and seed data it needs in `apps/api`, without changing module rules unless a module change is reviewed separately;
- runs the console's existing screen QA (`npm run qa:*`) against the real service, and adds API tests, including the gate's refusals.
- cites the requirement and acceptance IDs from its screen spec's trace section (HB, IAM, AUD, RET) in its plan, and its verification record has a planning cross-check section with a requirement trace (covered, deferred, not connected, or unclassified for owner review) and any deviations from the spec.

**Phase-end audit.** At the end of each phase, a read-only reviewer separate from the writers checks the phase's tickets against this plan and the screen specs, and records the result under `docs/reviews/`. [HYB-291](../../reviews/HYB-291-planning-cross-check/report.md) is the first such audit, covering phases 1–5.

**Pre-launch** is complete for the owner's review when:
- the 35 Go-bound screens read and change state through the Go service;
- the 9 FOCUS screens read through the Go session gate;
- the whole flow runs from one start command.

## Verification for every phase

- Go: `gofmt`, `vet`, `go test -race ./...`, and full statement coverage for new `apps/api` packages, with mutation testing on their decision logic.
- Console: `npm run typecheck`, `npm test`, `npm run build`, and the relevant browser QA in EN, KO and AR at the documented widths.
- Reviews follow the rules:
  - an independent design review before code and a change review after it;
  - a security-focused review for phases 1–3;
  - a DBA-focused review for phases 6–7.

## Out of scope

- Durable storage and a database adapter, which wait for the owner's database decision.
- Production deployment, HA, TLS certificates, packaging and installers for customers.
- Real CSP or ERP connections, payments, email or messages, and real customer data.
- SSO, LDAP and MFA verifiers; self-service recovery.
- Owner decisions: the database, retention periods, money precision and the support matrix. They stay provisional and are listed per ticket.

## Residuals

- State and the audit journal reset on every restart. The pre-launch notice says so.
- The Python FOCUS endpoints are not yet ported to Go.
- System-level acceptance cases remain NOT_RUN in production form. Phase 10 runs them locally against the pre-launch build only.
- Screen QA used per-ticket browser QA scripts instead of `npm run qa:*` from HB-260 onward; no record said so before [HYB-291](../../reviews/HYB-291-planning-cross-check/report.md). Whether `npm run qa:*` must run against the service is open for phase 10 or the owner.
- The audit journal keeps 2 hours (`apps/api/server/server.go:121`) and its test drives 5,400 commands, not the 28,800-record bound above; fix ticket HYB-292.

# Verification plan

This ticket is documentation: an independent architecture review of this plan, then OKF and link checks.

# Review history

- Round 1 (independent architecture reviewer, read-only): FAIL. Findings and what changed:
  - H1–H4, M1, M2. A generic command journal replayed through the gate cannot be made deterministic: sessions and Answers are bound to the process, reads extend sessions, generated values are random, and apply order is not append order. It was dropped. State is in memory with a synthetic seed, and durable storage waits for the owner's database decision.
  - M3. Passwords get a new reviewed module, `modules/passwords`, with a stated hash format.
  - M4. The completion criteria are narrowed to the 35 Go-bound screens, and the Python endpoints are put behind the Go session in phase 2.
  - M5. A seed loader and workers under named system actors.
  - M6. Phase 1 assembles identity and the bootstrap.
  - L1–L7. The bootstrap verifier is generated and written to a 0600 file; dummy-hash timing and a login limiter; Host and Origin checks; login CSRF; same-origin cookies through the proxy. Storage lows are moot without a journal. Untracked planning files are marked as such.
- Round 2 (same reviewer): FAIL. Findings and what changed:
  - N1. One installation-wide mutation lock and counter cover every mutation path. The counter is every target's revision, and module versions are read under the lock.
  - N2. Installation user operations name a gate action and kind, are limited to administrators, take the operator from the Answer and get a security review.
  - N3. The activation link goes only to a 0600 file, which is deleted after use.
  - L-a to L-g. The seed's password-set links; the password triple; a shared secret for the Python services and the FOCUS read permission; a global limiter without `X-Forwarded-For`; the proxy host and a session endpoint for CSRF; audit limits sized for review. The owner is told that changes reset on restart.
- Round 3 (same reviewer): FAIL. Findings and what changed:
  - R1. The lock is held for the whole `Execute` call, reads take their data and the counter together, and handlers never take the lock.
  - R2. Domain changes are defined, and session refresh, limiter, idempotency and audit bookkeeping are excluded.
  - R3. `view` replaces `read`. Installation user operations wait for a separately reviewed `modules/auth` change that defines an installation administrator (an administrator of the bootstrap workspace).
  - L1–L4. Per-link invalidation; how the shared secret is generated and refreshed; numeric audit limits with a test; the module version check is not a second guard.
- Round 4 (same reviewer): PASS. Low items folded in:
  - M-a. Workers bump only on reported module changes, and both directions are tested.
  - M-b. Hashing runs outside the lock; logins cause conflicts in open forms, stated.
  - M-c. The idempotency store and rate limits are sized and tested.
  - M-d. The bootstrap workspace ID is checked explicitly.
  - M-e. The secret file is re-read only when its modification time changes.

