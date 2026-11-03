---
type: Implementation
title: Local source ingestion service integration
description: Connects S03 to actual local FOCUS ingestion and durable receipt inspection with scoped boundaries.
---

# Scope and design

HB-141 follows the owner's request to develop executable integrations and discover issues during use. Root owns implementation/QA; a separate read-only architect reviews authority and API boundaries. Branch starts from main 36394ed and incorporates pending HB74 commit 618987e as an explicit frontend dependency; neither main nor PR68 is merged by this work. PR targets the dependency branch until authorized integration.

The first vertical flow is S03 file reception, actual validation/storage, receipt inspection and reloading persisted state. It uses the existing `focus_local` coordinator and PostgreSQL, ClickHouse and S3 implementations. This is real service execution using the existing synthetic-local-v1 capability; it is not production identity, arbitrary-provider support or completed S03/S06. No confirmation, publication, approval or external delivery endpoint is exposed. The full 49-screen implementation remains incremental.

A loopback-only same-origin server serves the manifest-owned console build and `/api/local-imports`. Startup explicitly opts into a local operator profile and fixes one loader actor; client parameters never select actors or scopes. Exact Host/Origin and a required custom API header reject cross-origin browser use. This is a trusted local operator boundary, not an authenticated multi-user deployment. No CORS or public bind exists. Source CSV is byte-bounded and restricted to the selected registry's headers before persistence. File names and source bodies are not logged.

List/detail reads use scoped authority transactions with `allow_unfinished=True` so unknown operations remain inspectable. Mutations retain the existing default unfinished-operation guard. Allowlisted metadata includes operation state: PostgreSQL receipt existence alone does not prove object/warehouse success. Unknown outcomes require inspection and cannot be represented as successful uploads. Source revision plus file/registry digest retains existing replay/conflict behavior. Monetary values are not recalculated by the browser.

The frontend preserves the established full-width design and owner-only review control. Only S03's explicit local-service mode calls this API; other candidates remain identified as previews. Server-derived scope replaces synthetic scope controls for the connected flow. Request cancellation and request identity prevent stale responses updating the current screen. Errors retain request IDs; refreshing never silently resubmits a write.

# Verification plan

Use a dedicated `hb141-integration` local verification project with new credentials and volumes, on the currently unused existing development ports. Existing deployments/volumes are not initialized or reset. Test actual HTTP→PostgreSQL/ClickHouse/S3 ingestion, reload, duplicate replay, changed-payload conflict, foreign-scope rejection, quarantine projection, unknown operations, cross-origin/invalid host/body-size rejection and real browser interactions. Other OS execution and production authentication remain unverified.

# API and operation

| Method/path | Contract |
| --- | --- |
| GET `/api/local-imports/profile` | Server-selected local scope, supported registry versions and byte limit. |
| POST `/api/local-imports` | Raw `text/csv`; `X-Focus-Version`, positive signed-64-bit decimal-string `X-Source-Revision`; same-origin `Origin` and `X-HB-Local-Client: 1`. No actor, workspace or tenant parameter. |
| GET `/api/local-imports` | Latest 25 releases and quarantine receipts, separate `more` flags and actual operation states. No raw object paths/CSV. |
| GET `/api/local-imports/{release-id}` | Exact scoped release evidence, generic denial for unavailable/foreign IDs. |

All API calls require the custom client header. Successful responses include a request ID for transport tracing; the frontend excludes it from stable review snapshots. Errors retain it. Revision values are strings through PostgreSQL projection, JSON and React; the browser never converts them to Number. The source-revision/digest duplicate check is the existing local contract, not a newly invented global idempotency scheme.

Install Python 3.11+ dependencies with `python -m pip install -r tools/console-service-requirements.txt`; the versions match the existing engine-probe pins. Configure `deploy/local/verification/.env` using its example without committing secrets. Existing local service endpoints are fixed at PostgreSQL 15432, ClickHouse 18123 and S3 18333. A fresh optional container setup is `docker compose -p hb141-integration -f deploy/local/verification/compose.yaml up -d postgres clickhouse seaweedfs`. Do not use this profile to initialize an existing customer deployment or assume the project name redirects these fixed ports. Native provisioned compatible services can supply the same local endpoints; Docker is a development provisioner, not a product-runtime OS choice.

Initialize this isolated profile with `python tools/local_focus.py init`. Build the console using `npm ci --prefix apps/console` and `npm --prefix apps/console run build`. Start with `python tools/console_service.py --local-operator`. The default URL is `http://127.0.0.1:8793/#/imports?lang=ko&service=local`. Stop with Ctrl+C. Rebuild then restart the server after frontend changes because its static allowlist is loaded at startup. The existing `tests/fixtures/focus/synthetic-1.3.csv` is an allowed test input; use a new source revision for changed bytes. Other screen candidates remain previews.

# Verification and independent review

Root executed five HTTP integration tests against a fresh dedicated HB141 database/warehouse/object-store project: actual validated ingestion, identical replay, changed-payload conflict, cross-scope read/upload rejection, restarted-server reread, quarantine with completed storage, unavailable-service rejection, an injected storage failure leaving a real unknown operation that remains inspectable, and hostile origin/host/header/oversize/actor-query rejection. Tests live in `tests/console_service/test_http.py`. They deliberately leave an unresolved operation in test workspace B; workspace A remains usable. These tests mutate their isolated profile and must not be run against an existing deployment. The failure injection is explicit and is not described as a real infrastructure outage.

The first initialization failed because two required local secret entries were absent; after supplying them in the excluded environment file, initialization succeeded. The first browser language-transition checks encountered retained transient state and the unsaved-input guard. The connected screen now remounts and re-queries on accepted locale changes, aborts prior requests, and clears discarded dirty state. The QA driver explicitly accepts its own discard prompt. Mobile rendering exposed horizontal overflow from the incorrect table wrapper; it was corrected and subsequently refined into labelled stacked rows.

Final actual browser run: file upload, validation/persistence, selected evidence, refresh/reload, duplicate replay and changed-payload error PASS; KO/EN/AR at 1440/390 PASS with no document horizontal overflow. Unit suite: 58 PASS, including stable success metadata and error request identity. TypeScript/build PASS; 242 artifact hashes including seven license notices verified. Existing planning semantic smoke PASS, without repeating the earlier 294-view matrix. The main JS size warning remains non-blocking.

Independent static review initially failed review identity because operation/quarantine evidence was omitted, then caught transient success request IDs contaminating the replacement identity. Both were corrected and re-review passed. It also passed the scoped locale-reset/dirty-navigation changes. The reviewer did not run mutation tests or claim runtime verification. The independent review covers this local adapter, not production authentication or all billing policy contracts.

Actual host: macOS arm64, Python 3.11.7, Node 26.7.0, Chromium headless shell build 1243. Portable entry points avoid required OS shell commands; other OS/browser executions remain unverified. No owner FIX, production readiness, main merge or completion of all 49 business screens is claimed.

The integration test entry point is `python tests/console_service/test_http.py --isolated-profile` on a fresh isolated profile. Default discovery skips these mutation tests. The explicit flag enables them; it does not reset or recover an existing unresolved operation. Browser QA is `npm --prefix apps/console run qa:service -- --playwright-module <module> --chromium <executable>` and also writes test receipts, so use the isolated service only.

Visual review identified hidden mobile status, mixed RTL values, oversized form controls, missing disabled-state explanation and weak receipt hierarchy. Root applied labelled mobile rows, individually isolated timestamps/IDs, bounded desktop fields, localized states with raw codes, accessible styled file selection and disabled reasons. Review of selected/conflict screenshots additionally caught a genuine conflict-recovery dead end: the screen cleared its receipts while asking the operator to inspect them. On revision conflict it now fetches current profile/receipts without resubmitting, preserving the error; failed rereads clear data. Independent source re-review passed that recovery. Fresh evidence captures start at scroll position zero so sticky navigation is not misrepresented.

Final actual Claude CLI visual re-review: PASS for corrected representative layouts and conflict/selected states, with unverified empty-file font in the last two images and native-language acceptance still pending. A full-page screenshot crops the independently scrolling sidebar; root separately verified the active link can scroll fully into view and the native file input exposes a visible focus-within outline. No CLI runtime testing was claimed.

Three actual CLI calls reported input_tokens: 14, cache_creation_input_tokens: 34,189, cache_read_input_tokens: 36,077, output_tokens: 10,155. Whole Codex/team/JEV usage is unavailable. No no-skill/no-cache control run was performed, so no token savings percentage is claimed.
