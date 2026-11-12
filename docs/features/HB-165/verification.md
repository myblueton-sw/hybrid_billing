---
type: VerificationRecord
title: B08 browser suite comparison case correction
description: Records why the B08 browser suite's lower-total comparison case failed on slow runs, the correction to the suite, and what was executed.
status: draft
---

# Finding

[HB-165](https://linear.app/hybrid-billing/issue/HB-165/b08-browser-suite-the-lower-total-comparison-case-reads-the-screen) was opened during HB-164 because `apps/console/tools/billing-runs-qa.mjs` failed at `deltas.includes("USD -1.00")` with no difference cells, on main's build and on the HB-164 build. Root (Claude Code) investigated and corrected it on 2026-11-12. Machine: mac_mini.

The ticket's first guess, that the suite read the screen before the new answer arrived, was wrong: an isolated reproduction showed the screen in its loading state immediately after the navigation and after "New query".

Cause, established with a diagnostic copy of the suite that printed the screen state and the reader traffic at the failure: the screen shows comparison rows only when the current read's `observed_at` is later than the kept baseline's (`ViewPanels.tsx`, the `fresh` condition). The suite built its routed "lower" answer from the very first response of the run and set its time to that response's time plus 60 seconds. The baseline, however, is kept from a later live read. When the steps before this case took longer than 60 seconds, the routed answer was older than the baseline, the screen correctly showed "Read the source again with New query before comparing", and no difference cell existed. In the captured failure the baseline was kept, the routed answer was applied with status 200, and the difference cells numbered zero before and after it.

The screen is not defective. The suite's other cases that reuse the first response either predate the baseline on purpose or do not depend on the order of times.

# Correction

The case records the answer of the read from which the baseline is kept and derives the routed "lower" answer from that read, 60 seconds later. Suite only; no screen code changes.

# Executed checks

| Check | Actual outcome |
| --- | --- |
| Unchanged suite, as observed during HB-164 | FAIL at this assertion in six of seven runs across main's build and the HB-164 build; the passing run was the fastest |
| Unchanged suite with a 65 second pause inserted after its first read (temporary copy, removed) | FAIL at the same assertion, as predicted |
| Corrected suite with the same 65 second pause (temporary copy, removed) | PASS, 36 captures |
| Corrected suite without a pause, on the retained sample | PASS twice, 36 captures, no page error, no non-GET request; the runs took 135 and 140 seconds, both longer than the 60 seconds that broke the unchanged suite |

All runs used main's console build and the reader restarted from main, read-only, under unrelated machine load (load average 10 to 16). No reset, ingestion or benchmark. Host: macOS arm64, Node 26.7.0, Playwright 1.56.1 with Chromium headless shell build 1243.

# Limits

- No independent review was run on this six-line suite change; root checked it directly with the before-and-after runs above.
- The suite still reuses its first response in other cases; they were read and judged unaffected, not rewritten.
- The suite's long duration under load is unchanged.
