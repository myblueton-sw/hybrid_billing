---
type: ImplementationPlan
title: Collected reader waiting places and error output
description: Defines how a waiting read whose caller has left frees its waiting place early and how the reader's unhandled-error output is reduced to one fixed line.
status: reviewed
---

# Scope and ownership

[HB-163](https://linear.app/hybrid-billing/issue/HB-163/collected-reader-free-the-waiting-place-of-a-caller-that-left-and-keep) is the owner's chosen follow-up to HB-162 (2026-11-12). Base: main `b96a213`. Root (Claude Code) owns planning, implementation, QA and integration. A separate read-only reviewer reviews this plan before implementation and the actual change afterwards. Machine: mac_mini.

# Observed baseline

Facts read on 2026-11-12 from `modules/focus_import/overview_http.py` at `b96a213` and from the HB-162 change review:

- A validated read takes one of eight admission places without blocking. If no slot is free it makes one timed wait of up to `slot_wait` seconds (20) on the slot semaphore. Only after it gets a slot does it check whether its caller has closed the connection (`abandoned()`, a non-blocking one-byte peek).
- A waiter whose caller has left therefore keeps its admission place until it reaches a slot or its wait ends. Behind two slow queries, about seven aborted reads within 20 seconds fill the six waiting places and the next read is answered `source_busy` at once.
- A waiter whose wait ends writes its `source_busy` answer whether or not the caller is still there.
- `log_message` is silenced, but the server keeps the standard library's default `handle_error`, which prints a traceback to standard error when a handler raises, for example on a write to a closed connection. If that write happens while an exception from a query is being handled, the chained traceback includes that exception's message.
- The HB-162 tests use a semaphore subclass that signals each timed acquire and each release.

# Decision

Reader only; endpoints, error codes, the two-query limit, the 20 second wait, the six waiting places, authorization and all console code are unchanged.

| Area | Decision |
| --- | --- |
| Waiting in steps | A read that has to wait computes one deadline from a monotonic clock and makes timed waits of `min(slot_step, time left)` seconds (default step 0.25) until it gets a slot or the deadline has passed. After each step that ends without a slot it checks `abandoned()` before looking at the deadline; if the caller has left it returns without a query and without writing, and its admission place is released by the existing `finally`. The loop sits where the single wait was, inside the admission block and before the slot block, so an exit without a slot releases only the admission place and an exit with a slot always passes the slot's release |
| After the slot | The existing check after a waited read gets its slot stays, so a caller that left during the last step is still skipped and the slot is released |
| End of the wait | Unchanged: `503 source_busy`. Because of the step checks the caller was present at most one step earlier |
| Error output | `OverviewServer.handle_error` is overridden to write one fixed line, "Collected FOCUS local reader: a request ended without a complete response.", to standard error in a single write, looked up at call time, with no exception type, message, traceback, address or request data. Request logging and `send_error` output are already silenced through `log_message`. The response paths and the mapping of query errors are unchanged |
| Configuration | `slot_step` is a server attribute set in the constructor next to `slot_wait`. No new command-line option |

Cost: a waiting read wakes four times a second for a non-blocking peek; at most six reads wait at once.

Limits. A client that closes only its sending side and then has to wait is treated as gone, now within one step instead of at its slot, and no longer receives `source_busy` at the deadline. A running query is not cancelled. Wait order changes: with one long wait the semaphore served waiters close to arrival order; with steps every waiter rejoins the queue four times a second, so a freed slot goes to whichever waiter's current step began first. Later arrivals cannot overtake without end because of the 20 second deadline, but when the six places stay full for the whole 20 seconds an early live caller can be refused while later ones are served (the reviewer's rough estimate is a 1 to 9 percent chance in that state). This is accepted for a local single-operator reader in preference to an arrival-order queue, by the simplest-design rule; it is the owner's to revisit if the reader gets concurrent users.

# Allowed files

`modules/focus_import/overview_http.py`; `tests/focus_overview/test_http.py`; `apps/console/README.md` (only if the reader sentence added by HB-162 needs a change); `docs/features/HB-163/*`.

# Verification

- Reader tests in `tests/focus_overview/test_http.py`. The recording semaphore signals a waiter once per handler thread, tracked with a thread-local flag rather than thread identifiers (which are reused), and counts timed acquires, so the existing HB-162 tests keep their counts with stepped waits. The existing abandoned-caller test sets one step as long as the whole wait, so that it keeps testing the check after the slot.
  - New: a waiting caller that closes its connection frees its admission place while both slots are still held, no query runs and no slot is taken; after eight such callers one after another a further read is still admitted, waits and is served once a slot is released (the places are never full at the same time in this test; that case is covered by the retained-sample script); a waiter whose total wait ends gets `source_busy` no earlier than `slot_wait` and after at least two steps; the error output is exactly the fixed line, both when the handler is called directly with an exception active and when a query fails and the write of its error answer fails too (the write is made to fail by replacing the reply method, not by a real closed connection).
  - All five new tests must fail on the reader at `b96a213`.
  - Existing HB-162 tests still pass.
- All six reader suites, each directory on its own.
- On the retained sample, without reset, ingestion or benchmark, one raw-socket script run against a reader at `b96a213` and against the changed reader on spare ports: two reads are started, eight more are sent and closed at once after 0.2 seconds, and a final read is sent 0.6 seconds later; a run counts only if the final read started before the first of the two finished. Expected: `503` within milliseconds on `b96a213`, `200` on the changed reader. Then the U01 browser suite with its count of `source_busy` recoveries.
- Retained file hashes against the HB-150 manifest.
- Independent review of the actual change.

# Deferred

A "still waiting" hint on screens (next ticket); an arrival-order queue for waiters; the stale line citations of `overview_http.py` in `docs/features/HB-156/closing-target-contract.md`; per-endpoint slot limits; Linux and Windows remain unverified; cancelling a running query; a `Retry-After` header; production capacity and latency targets; tightening the U01 suite's recovery bound to zero.

# Design review resolution

The independent design review of 2026-11-12 (read-only, nothing executed) returned FAIL without a blocker: 1 HIGH, 4 MEDIUM, 5 LOW. Resolution:

| Finding | Resolution |
| --- | --- |
| F1 the existing abandoned-caller test would race the step check | That test pins a single step; the step path has its own tests |
| F2 wait order becomes near-random | Accepted and stated as a limit above |
| F3 once-per-thread signal keyed on reused identifiers | Thread-local flag; timed acquires counted |
| F4 retained-sample check only discriminates in a window | Raw-socket script with timestamps against both readers; a run counts only inside the window |
| F5 error-output test should prove the wiring | Direct call and a failed write after a failed query |
| F6 loop details | Adopted as written in the decision |
| F7 half-closing client | Consequence stated |
| F8, F9 error line | One fixed line in a single write; no guard for a closed standard error |
| F10 scope and documents | Deferred list extended; HB-162's record is left as history |
