---
type: VerificationRecord
title: Collected reader waiting places and error output verification
description: Records what was executed and reviewed for the stepped slot wait with early release of a departed caller's waiting place and for the fixed error line, with results, limits and accepted residuals.
status: draft
---

# Delivered behavior

[HB-163](https://linear.app/hybrid-billing/issue/HB-163/collected-reader-free-the-waiting-place-of-a-caller-that-left-and-keep) changes `modules/focus_import/overview_http.py` from main `b96a213` as the owner's chosen follow-up to HB-162. Root (Claude Code) owns planning, implementation, QA and integration; a separate read-only reviewer reviewed the plan before implementation and the change afterwards. Machine: mac_mini. See the [plan](plan.md).

A read that has to wait for a slot now waits in steps of at most 0.25 seconds up to the unchanged 20 seconds. After each step without a slot it checks whether its caller has closed the connection and, if so, returns without a query or a response and frees its waiting place. The check after a waited read gets its slot stays. An exception that escapes a request handler prints one fixed line without any detail. Endpoints, error codes, the two-query limit, the six waiting places, authorization and all console code are unchanged. This closes two residuals recorded in HB-162: waiting places held by callers that left, and tracebacks on standard error.

# Executed checks

| Check | Actual outcome |
| --- | --- |
| `tests/focus_overview` | PASS 48, of which 5 new in `test_http.py`; the HTTP file was run three times with the same result |
| New tests on the reader at `b96a213` | All 5 FAIL, as intended: place freed before a slot, eight callers leaving one after another, total wait across steps, fixed error line by direct call, fixed error line after a failed write |
| `tests/focus_connections`, `tests/focus_collection_status`, `tests/focus_receipts`, `tests/focus_marketplace`, `tests/focus_commitments` | PASS 97, 84, 77, 40, 34, each directory on its own |
| Departed callers on the retained sample, three runs per reader | Reader at `b96a213`: the final read was refused with `503` within 10 milliseconds in all three runs. Changed reader: the final read was answered `200` in all three runs, after 9.7, 10.5 and 19.8 seconds. In every run the final read started (0.81 to 0.84 seconds) before the first of the two running reads finished (3.0 seconds at the earliest), so every run counts |
| U01 browser suite against the changed reader | PASS with 0 recoveries from `source_busy`, at a machine load average of 25.9 |
| Retained source, control and prior-result files | PASS 93 file hashes equal to the HB-150 preservation manifest after all reads |

The script for the departed-caller check uses raw sockets: two reads of `/api/focus-overview?provider=AWS`, then after 0.2 seconds eight requests sent and closed at once, then after 0.6 seconds a final read, with monotonic timestamps. Both readers ran on spare ports with the existing trusted local configuration and the same retained store, read-only; the changed reader was loaded from the ticket's module. The readers on their usual ports were not touched. The machine was under unrelated load during these runs (load average 13 to 26), which is why the reads took 3 to 20 seconds. No reset, ingestion or benchmark was run. No console file changed; the suite used main's console build.

Not run in this ticket: the S04, B07 and B06 browser suites and the six-concurrent-reads comparison of HB-162. The reader's behavior for a caller that stays is the same as in HB-162 apart from wait order.

Host: macOS arm64, Python 3.11.7 from the prepared environment, Node 26.7.0, Playwright 1.56.1 with Chromium headless shell build 1243. Linux and Windows were not exercised and remain unverified targets.

# Limits

- No capacity or latency claim follows from these runs; in one run a served read took 19.8 seconds, just inside the 20 second wait.
- `source_busy` can still occur: when eight live reads are admitted, or when a read waits 20 seconds.
- Wait order is no longer close to arrival order; see the plan's limits. An early live caller can be refused while later ones are served when the waiting places stay full for the whole wait.
- A running query is not cancelled when its caller leaves.
- A client that closes only its sending side and then has to wait is dropped within one step and gets no `source_busy` answer.
- No unit test fills the waiting places with departed callers at the same time; only the retained-sample script does.
- A step of zero or less would make a waiter spin until its deadline; the step is a constant and has no option.
- If standard error is unusable the error handler itself fails; there is no guard.
- The fixed error line removes the traceback as a diagnostic: a programming error in the handler looks the same as a caller that left, and its cause must be reproduced to be seen.
- The fix takes effect only in a reader process started from the changed code.

# Independent review

Reviewer in each round: the repository's read-only `independent-reviewer` agent role (AI), 2026-11-12. It read files and executed nothing.

| Round | Result |
| --- | --- |
| Design review of the plan, before implementation | FAIL without blocker: 1 HIGH, 4 MEDIUM, 5 LOW. Resolved as in the plan's resolution table; the change in wait order is accepted as a stated limit rather than designed away |
| Change review of the implemented diff | PASS without blocker: 5 LOW and one correction to this record (the review rounds listed here did not yet include this one). Corrected: the two departed-caller tests also assert that nothing was written to standard error, which excludes an exception path; one test renamed to say what it covers; the half-closing client's missing answer stated. LOW recorded as limits above. No second round ran on these test and record corrections |
