---
type: ImplementationRecord
title: Protected retained receipt metadata reader
description: Records the read-only collection receipt projection, its ownership rules and isolated backend verification.
status: draft
---

# Scope

HB-145 adds fixed GET `/api/focus-receipts` to the existing trusted-local `OverviewServer`. The isolated implementer owns only the receipt projection/helper, GET dispatcher, receipt tests and this report. Root owns frontend/configuration/retained-runtime integration; a separate reviewer owns independent review. No migrations, uploads, source imports, resets, seeds, financial commands, real-account operations or retained-data mutations were performed by this implementation.

`receipt_projection.receipts` uses `collection.access(actor, 'query')`, which checks the control journal under its existing lock, followed by the current authority's query transaction. Delivery occurs inside both guards. Operator-only contexts cannot project the journal. Existing Host/Origin/FetchSite/custom-header/concurrency/no-store protections and all older GET routes are preserved. Query has only view, q, after and snapshot/permission pins; actor/company/tenant selectors are rejected.

# Projection and privacy

The envelope and exact22 nullable record keys match the frozen design. Pages contain at most25 records, counts are exact within a maximum10000-record local inspection window, and the projected/wire body is limited to2MiB. Success omits transient request IDs; new-path errors include a safe code and request ID. Prefix search applies only to authorized IDs/connection tokens. Pagination pins bind the complete filtered result, view, q and current authorization, while excluding after so a single evidence pin remains usable across pages. The response echoes the exact after offset.

Current active tenant bindings and the journal's verified configuration must agree with the fixed reader's source/config binding. An explicit connection is visible only when all registered domains resolve to currently permitted members; a successful manifest's positive slices must also remain inside those domains. Legacy successful deliveries require an exact immutable manifest/descriptor and every actual positive-row tenant to be authorized. Zero-row placeholders, missing batches and unregistered legacy receipts do not prove ownership. Ambiguous/unattributable/foreign objects contribute no records, counts, search hits or public pins.

Admitted quarantine additionally requires exact current connection ownership, matching admitted tenant keys and the immutable bare-descriptor digest. Current admission validates the descriptor shape, revision, row count, time range, file hash and connection version before producing admitted=True. The reader rejects an admitted owned record if its recorded file hash differs from its descriptor, its version differs from the current connection, or its declaration contradicts these invariants. Pre-admission hash/version/scalar failures are unadmitted rejected receipts and remain excluded; nullable output fields are not a licence to display contradictory admitted evidence. Lineage references require independently authorized records in the same connection/window; the attempt counter belongs to the attributable root, not a child or duplicate metric.

Delivery stages are registered/verified/sealed. Quarantine stages are quarantined/replacement-requested/escalated/resolved/deleted. These are collection observations, not job success or financial eligibility. Delivery received_at is the window lineage origin; quarantine received_at is that receipt timestamp. All delivery/quarantine completeness/publication fields remain null.

Completeness reuses only fully owned registered calendars/domains. It aggregates one row per connection/window and distinguishes latest collection stage from the current published descriptor. Publication, finalized and known-zero states are separate; no publication leaves the latter values null. Known zero requires explicit current published attestations for every registered domain. Unsafe or unavailable completeness returns total=null, not zero. Extraction has no producer and always returns an empty record list with total=null and capability unconnected. No originals, CSV/Parquet rows, warehouse facts or source payloads are loaded by this projection; the existing readiness guard still reads its protected control metadata/mirrors.

Public authorization/snapshot hashes contain only current authorized bindings, relevant owned policy/connection information and permitted returned facts/counts/capabilities. They exclude the whole journal/configuration, global control head/sequence and foreign activity. Source namespace is part of auth_scope. The projection never serializes object keys, filenames, URLs, bodies, configurations, plans, holds, tenant-slice maps or credentials.

# Verification

Tests use in-memory synthetic journal metadata and ephemeral HTTP handlers with a mocked consumer. They do not connect to retained services, read credentials, initialize a profile or run a mutation integration suite. Coverage includes exact keys/null semantics, source/member bindings, full versus mixed ownership, unadmitted rejection exclusion, independent lineage, zero attestations, latest versus published coverage, prefix filtering, page/pin consistency, changed facts/permission, foreign-activity hash invariance, invalid metadata/query, response bounds, guarded callback scope and sanitized errors.

Two actual request-guard defects were reproduced with raw HTTP: a malformed absolute URI could close the request instead of returning a safe error, and duplicate Content-Length was accepted. The handler now returns a safe invalid-request result for malformed URLs, duplicate lengths and transfer encoding; before-failure evidence was preserved outside Git. New-path denied methods include a request ID without changing older denied-method body shapes.

The initial combined pytest command hit the repository's unpackaged test-basename collision. New receipt test modules were given unique basenames, and the final combined regression uses `--import-mode=importlib`. An initial malformed-URL test incorrectly let the HTTP client parse the URL; the corrected test uses a raw request line before checking the actual server behavior. Harness failures are separate from product failures.

Initial implementation verification PASS: `python -m pytest -q --import-mode=importlib tests/focus_receipts tests/focus_overview/test_http.py tests/focus_overview/test_projection.py tests/focus_overview/test_ports.py tests/focus_commitments/test_commitments.py tests/focus_marketplace/test_marketplace.py` returned178 passed. Python compilation and scoped whitespace checks also passed. The combined suite covers receipt, overview HTTP/projection/port, commitment and Marketplace tests. Bytecode and pytest caches remain unstaged. Each maintained source/test file is below1500 physical lines; no explanatory comments/docstrings were added.

# Limits

Independent design PASS is not an implementation/runtime approval. Root must run actual retained read-only integration, frontend schema/race/locale/layout verification, independent source review and handoff. This narrow phase can exclude legacy or mixed-scope records that lack complete ownership proof; it does not present a whole-CSP inventory or upstream completeness guarantee. Metadata does not verify current physical source/warehouse availability, worker approval or financial readiness. Scale, native-language, assistive-technology and other OS/browser acceptance remain unverified.

The inherited `control` guard imported POSIX fcntl unconditionally. Root subsequently added a separate native lock adapter and delegated the same existing sidecar lock to it. The isolated backend writer did not modify control.py or claim Windows execution. POSIX old/new process interoperability and mocked Windows API tests are recorded in the integrated verification; native Windows and existing write-side directory-sync portability remain unverified.

# Independent-review correction

A separate actual-source reviewer found that the initial metadata fixture deliberately used a receipt hash different from its descriptor, and the reader allowed that inconsistency. Reviewing `connections.admit_source` established that file-hash equality, connection-version equality and valid descriptor scalars precede admitted=True. Fixtures now preserve those invariants and use a post-admission numeric-validation reason. Seven negative variants with internally recomputed metadata identities reproduced the prior acceptance gap before correction; the reader now fails closed instead of forwarding impossible admitted declarations. This correction changes no response keys or live data.

Correction verification PASS: the same isolated combined regression returned 187 passed in 31.08 seconds, including seven projection rejection variants and two HTTP cases for contradictory admitted hashes/versions. HTTP returns 503 collected_unavailable with only error and request_id; no contradictory receipt is serialized. Retained-runtime and independent re-review remain root responsibilities.
