---
type: VerificationRecord
title: Collected reader slot wait verification
description: Records what was executed and reviewed for the collected reader's bounded slot wait and abandoned-caller check, with results, limits and accepted residuals.
status: draft
---

# Delivered behavior

[HB-162](https://linear.app/hybrid-billing/issue/HB-162/collected-reader-wait-for-a-slot-and-retry-reads-instead-of-surfacing) changes `modules/focus_import/overview_http.py` from main `225e6a5` by the owner's decision of 2026-11-12. Root (Claude Code) owns planning, implementation, QA and integration; a separate read-only reviewer reviewed the plan before implementation and the change afterwards. Machine: mac_mini. See the [plan](plan.md).

After transport validation a read takes one of eight admission places or is answered `503 source_busy` at once. An admitted read runs at once when one of the two slots is free; otherwise it waits up to 20 seconds and is answered `source_busy` if no slot becomes free. A read that had to wait and whose caller has closed the connection is dropped without running its query. At most two queries run at a time, as before. Authorization, selection, snapshot and paging behavior, the error codes and all console code are unchanged.

# Executed checks

| Check | Actual outcome |
| --- | --- |
| `tests/focus_overview` | PASS 43, of which 7 new in `test_http.py`; repeated three times with the same result, and again after the review corrections |
| New tests on the unchanged reader | 5 of the 7 FAIL in the run before the review corrections, as intended: wait and serve, full waiting places, abandoned caller, release after a failed waited query, admission balance. The transport-invalid test describes behavior the unchanged reader also has and is a regression guard; the busy-after-the-wait test passed on the unchanged reader and now also asserts that a wait took place, which was not re-run against the unchanged reader |
| `tests/focus_connections`, `tests/focus_collection_status`, `tests/focus_receipts`, `tests/focus_marketplace`, `tests/focus_commitments` | PASS 97, 84, 77, 40, 34. The slot tests in all five existing suites use a shortened wait; two of them were missed at first and waited the full 20 seconds each, found by the change review. Each directory is run on its own because three share the file name `test_http.py` |
| Six concurrent reads of `/api/focus-overview?provider=AWS` on the retained sample | Unchanged reader: two `200` (1.7 and 3.1 seconds) and four `503` within 3 milliseconds. Changed reader: six `200` between 1.8 and 9.2 seconds |
| U01 browser suite against the changed reader | PASS twice, with 0 recoveries from `source_busy` each time (machine load average 12.5 and 5.5). On the unchanged reader the same suite needed 1 to 2 recoveries or failed during HB-161 |
| S04, B07 and B06 browser suites against the changed reader | PASS. B07 and B06 route their own one-time `source_busy` answer and still see the error state, because the console does not retry |
| Retained source, control and prior-result files | PASS 93 file hashes equal to the HB-150 preservation manifest after all reads |
| Proxy behavior on a client abort | Observed once with a Vite 8 preview proxy and a socket sink: client abort at 1.5 seconds, upstream end of stream at 1.54 seconds |

The changed reader was run from the ticket's module on a separate port with the existing trusted local configuration and the same retained store, read-only; the running reader on its usual port was not touched. No reset, ingestion or benchmark was run. The console build served to the suites is main's console; no console file changed.

The new tests are synchronized by a semaphore subclass that signals when a handler starts a timed wait and when a slot or an admission place is released. The only timed step is a 0.2 second pause in the abandoned-caller test for the close to reach the server on loopback; that delivery is prompt but not guaranteed.

Host: macOS arm64, Python 3.11.7 from the prepared environment, Node 26.7.0, Playwright 1.56.1 with Chromium headless shell build 1243. The non-blocking peek uses only portable socket calls, but Linux and Windows were not exercised and remain unverified targets.

# Limits

- A passing browser suite shows no regression at the latency of that run. It is not evidence about capacity, and no latency bound is asserted anywhere.
- A read can now take up to 20 seconds longer before it is served or refused. Screens show their loading state during that time without a hint.
- A running query is not cancelled when its caller leaves; only waiting callers are skipped.
- A client that closes its sending side after the request and then has to wait is treated as gone. No known client of this reader does that.
- Waiters are not served in a guaranteed order. A waiter whose caller has left keeps its waiting place until it reaches a slot or its wait ends, so about seven aborted reads within 20 seconds behind two slow queries can still fill the waiting places and produce an immediate `source_busy`.
- A failed write to a caller that has gone can print a traceback to the reader's standard error through the standard library's default handler, as before this change; overriding that handler is a follow-up.
- `docs/features/HB-156/closing-target-contract.md` cites line numbers of `overview_http.py` that have moved; it is outside this change.
- The U01 browser suite's recovery from `source_busy` is now mostly redundant and could be tightened to zero in a follow-up.
- The fix takes effect only in a reader process started from the changed code.

# Independent review

Reviewer in each round: the repository's read-only `independent-reviewer` agent role (AI), 2026-11-12. It read files and executed nothing.

| Round | Result |
| --- | --- |
| Design review of the plan, before implementation | FAIL without blocker: 3 HIGH, 3 MEDIUM, 1 LOW. The planned console retry would have broken one-time busy cases in two browser suites and did not reach three endpoints; it was dropped for the reader-only design the reviewer named. The validation order, the abandoned-caller check and the wait's rationale were restated as in the plan's resolution table |
| Change review of the implemented diff | PASS without blocker: 1 MEDIUM, 5 LOW. MEDIUM: two further slot tests waited the full 20 seconds, corrected with the plan's file list. LOW corrected: the admission-balance test waits on a release signal instead of racing the handler; the busy-after-the-wait test proves a wait. LOW recorded as limits: waiting places held by callers that left, the standard-error traceback, stale line citations in HB-156. No second review round ran on these test-only corrections |
