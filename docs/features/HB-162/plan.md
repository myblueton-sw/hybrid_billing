---
type: ImplementationPlan
title: Collected reader slot wait
description: Defines how the local collected reader waits a bounded time for a request slot and skips callers that have left, so that quick selection changes during slow reads no longer require a manual refresh.
status: reviewed
---

# Scope and ownership

[HB-162](https://linear.app/hybrid-billing/issue/HB-162/collected-reader-wait-for-a-slot-and-retry-reads-instead-of-surfacing) follows the owner's decision of 2026-11-12 to improve the `source_busy` behavior recorded in HB-145, HB-151 and HB-161. Base: main `225e6a5`. Root (Claude Code) owns planning, implementation, QA and integration. A separate read-only reviewer reviewed this plan before implementation and reviews the actual change afterwards, because the change touches a local service's request handling. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-12 from the code and from the HB-161 runs:

- `modules/focus_import/overview_http.py` serves `/api/focus-overview`, `/api/focus-commitments`, `/api/focus-marketplace`, `/api/focus-receipts`, `/api/focus-connections` and `/api/focus-collection-status` from one `ThreadingHTTPServer` bound to `127.0.0.1`. Transport validation (host, origin, headers, path, query shape) precedes the slot. A request then takes one of two slots (`threading.BoundedSemaphore(2)`) without blocking; when both are taken it answers `503 source_busy` at once. Selection validation and authorization run inside the query, under the slot, and their errors are mapped afterwards. The slot is released when the query returns. The 15 second socket timeout applies to socket operations, not to a thread waiting for a slot.
- A selection change in the console aborts the browser request. The reader does not check for a closed connection and finishes the query, holding its slot.
- The console shows the error with a refresh action. Nothing waits or retries.
- During HB-161 the machine ran unrelated workloads and reads took 2 to 25 seconds; three selection changes in a row then produced `source_busy` in more than ten suite runs.
- Reader HTTP tests in `tests/focus_connections`, `tests/focus_collection_status`, `tests/focus_receipts`, `tests/focus_marketplace` and `tests/focus_commitments` hold both slots and expect `source_busy` without a query. `tests/focus_overview/test_http.py` had no slot test.
- Observed on 2026-11-12 with a Vite 8 preview proxy and a socket sink: when the client aborts, the proxy closes its connection to the reader at once (client abort at 1.5 seconds, upstream end of stream seen at 1.54 seconds).

Not known: read latency and the number of concurrent readers in a production deployment. This change makes no capacity claim.

# Decision

Reader only. The endpoints, the error code, the two-slot limit on concurrent queries, authorization, selection, snapshot and paging behavior and all console code are unchanged.

| Area | Decision |
| --- | --- |
| Admission | After transport validation a request takes one of eight admission places (two running plus `MAX_WAITERS` = 6) without blocking; with none free it is answered `503 source_busy` at once, as today |
| Slot wait | An admitted request takes a slot at once if one is free; otherwise it waits up to `slot_wait` seconds (default 20) and is answered `source_busy` if none becomes free. The default is a judgement from the observed 2 to 25 second reads on a loaded machine, not a measured target |
| Abandoned callers | A request that had to wait checks, when it gets its slot, whether its caller has closed the connection: a non-blocking one-byte peek that returns end of stream, or a connection error, means the caller left. The reader then releases the slot without running the query and without writing a response. A peek that would block or returns data means the caller is there. A request that never waited is not checked; a running query is not cancelled |
| Release | The slot and the admission place are each released exactly once on every path, including a failed query and an abandoned caller |
| Configuration | `slot_wait` is a server attribute set in the constructor, so tests can lower it. No new command-line option |
| Console | No change. A screen stays in its loading state while its read waits and shows the existing error with its refresh action if the reader stays busy for the whole wait |

Limits stated on purpose: a client that sends its request and then closes only its sending side looks like a caller that left and would be dropped if it had to wait; browsers, the Vite proxy, Playwright and Python's HTTP client do not do this. A caller that leaves just after the check still gets its query run, as today. Waiters are not served in a guaranteed order. Worst case for one read: 20 seconds of waiting plus its own query time, or the existing error after 20 seconds.

# Allowed files

`modules/focus_import/overview_http.py`; `tests/focus_overview/test_http.py`; `tests/focus_connections/test_http.py`, `tests/focus_collection_status/test_http.py`, `tests/focus_receipts/test_receipt_http.py`, `tests/focus_marketplace/test_marketplace.py`, `tests/focus_commitments/test_commitments.py` (a shortened wait in the existing slot test only); `apps/console/README.md`; `docs/features/HB-162/*`.

# Verification

- Reader tests in `tests/focus_overview/test_http.py`, synchronized by recording semaphores that signal when a handler starts waiting and when a slot or an admission place is released, not by timing: a request that arrives while both slots are held is served after one is released and its query runs once; with a shortened wait it gets `source_busy` and no query runs; with the waiting places full a further request is answered `source_busy` while the six waiters are still waiting, and all six are served afterwards; a waiting caller that closes its connection causes no query and leaves the slot free (the only timed step is a short pause for the close to reach the server); a query that fails after waiting releases its slot; transport-invalid requests are rejected without waiting while slots are held; after ten timed-out waits every admission place is free. The new tests must fail on the unchanged reader.
- The five existing slot tests now assert `source_busy` after a shortened wait instead of at once.
- All reader backend suites. Console unit suite, build and build verifier unchanged.
- On the retained sample, without reset, ingestion or benchmark: six concurrent reads against the unchanged reader and against the changed reader; the U01, S04, B07 and B06 browser suites against the changed reader, with the U01 suite's count of `source_busy` recoveries recorded. A passing suite shows no regression at the recorded latency; it is not evidence about capacity.
- Retained file hashes against the HB-150 manifest.
- Independent review of the actual change.

# Deferred

Cancelling a running query when its caller disconnects; a `Retry-After` header; per-endpoint slot limits; production capacity and latency targets; a "still waiting" hint on screens; an automatic retry in the console, dropped after the design review.

# Design review resolution

The independent design review of 2026-11-12 (read-only, nothing executed) returned FAIL without a blocker: 3 HIGH, 3 MEDIUM, 1 LOW. Resolution:

| Finding | Resolution |
| --- | --- |
| F1 console retry breaks one-shot busy cases in the B07 and B08 suites | The console retry is dropped; the minimal reader-only design the reviewer named is adopted |
| F2 "a rejected request never waits" was false | Restated: transport validation precedes the wait; selection validation and authorization stay inside the query under the slot, unchanged. Not hoisted |
| F3 abandoned-caller check under-specified | Non-blocking peek as specified by the reviewer, limits stated above; proxy behavior observed and recorded in the baseline. Linux and Windows remain unverified targets |
| F4 rationale for the wait | The socket timeout is no longer cited; the wait is 20 seconds with its basis stated, and both worst cases are given |
| F5 retry did not reach three reader endpoints | Moot: the reader change applies to all six endpoints |
| F6 verification | Recording semaphore, the additional cases, and the limits of what a passing browser suite shows |
| F7 retry details | Moot |
