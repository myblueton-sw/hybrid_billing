---
type: Verification
title: HYB-316 M04 held credential issue survives a layout change
description: Root cause, fixes and observed results for the M04 credential issue retry that sometimes sent no second request after a lost response (panel unmounted on a width change across 1280 px), plus the review findings on an attempt renewed by a reload during the request and on Cancel at narrow width.
status: stable
tags: [HYB-316, M04, service-accounts, idempotency, verification]
sources:
  - resource: ../HYB-293/verification.md
---

# HYB-316 verification (mac_mini, 2026-11-19, base main e5f7751)

All browser runs used synthetic data, a fresh API process per run with `--seed sample`, a new mode-0700 data directory deleted afterwards, Playwright 1.56.1 with chrome-headless-shell 1243 and the console `dist` built from the worktree.

## Reproduction

- `qa269` (M04 runner) unchanged on main: 3 of 3 passed in this session, so the full runner alone does not give a rate here.
- The runner trimmed to the held-retry step (`qa316cap`, same steps as `qa269` up to line 164, including the full-page screenshot between "Try again" appearing and the retry click), with event tracing: 2 of 30 failed, then 0 of 25 with a heavier DOM observer, then 1 of 30 and 2 of 30 with component tracing. In every failure the button read "Issue credential", not "Try again", the reason field was empty, the click submitted the form and the client validation summary took focus, so no request was sent.

## Root cause

Component tracing in the two failing runs of the last batch shows the same order:

1. The aborted POST makes `useCommand` hold the attempt (`holding=true`, same `request_id` and `idempotency_key` kept in the `IssueForm` refs) and reload `GET /identity/services` (200).
2. During the full-page screenshot the `(min-width: 1280px)` media query reported `false` with `innerWidth` 1 for 7 ms and 14 ms in the two failing runs, then `true` again. `useWide` followed it.
3. At narrow width `LiveServices` shows the panel only when the mode is not `form` or `open` is set. Neither held, so the panel, and with it `IssueForm`, unmounted. When the width returned, a new `IssueForm` mounted with empty state: default days, empty reason, a new attempt and `holding=false`.
4. The retry click therefore submitted an empty new form.

In the 30 traced runs before the fix, 26 recorded a narrow flip at some point, and the flip fell inside the held-retry window in exactly the 2 failing runs. A flip with no screenshot can also be produced by any real width change.

The held attempt lives only in a component that a pure layout change can unmount. That is the defect; the screenshot only triggers it.

## Can a real user hit it

Yes, with a real width change across 1280 CSS px while the attempt is held or in flight, for example resizing or snapping the window, changing the browser zoom, or rotating a tablet. Deterministic check `qa316check` on the unfixed code (1440 px → 1000 px → 1440 px after the reload; the 20 `back` runs used the pre-fix bundle with trace logging added, the single `narrow` and `redo` runs the plain pre-fix bundle):

| Mode | Observed |
| --- | --- |
| `back` (retry after returning to 1440 px) | 20 of 20 failed: form shows "Issue credential" with days 30 and empty reason; no second POST |
| `narrow` (retry at 1000 px) | 1 of 1 failed: no issue form on screen |
| `redo` (user re-enters 1 day and the reason and submits) | 1 of 1: second POST with a new `idempotency_key`; two active credentials; only the second secret shown |

The `redo` run shows the user-visible consequence: the first credential, committed by the server when the response was lost, stays active and its secret is never shown, and it was not added to the "Secrets not confirmed" reminder.

## Review findings applied (round 2)

Root's change review passed the scoped fix and asked for three more changes in this ticket:

- **N1, a second key for the same intent.** `useCommand` renewed the attempt whenever the page revision changed and no attempt was held. A reload that lands while an issue POST is in flight changed the revision, so when that POST's response was then lost, the held attempt was the renewed one and "Try again" sent a new `request_id` and `idempotency_key`.
- **N3, a committed check of the `Live.tsx` wiring** across a width change.
- **N4, Cancel on the revoke step at narrow width.** With the panel open, Cancel returned to the issue form instead of the list, and the requested focus target (the credential row) was not on screen, so focus was lost.

## Fix

- `live-model.ts`:
  - `panelShown(wide, form, open)` holds the existing panel rule; `pinIssue(service)` returns `{ chosen: service, open: true }`.
  - `attemptState(fresh)` holds the command attempt. A revision change renews it only when nothing is held and no request is in flight; a held or unknown outcome keeps the attempt that was sent; the next send after a success uses a fresh attempt unless the result is then held.
- `Live.tsx`:
  - Submitting an issue pins both the service and the open panel, so the panel stays mounted at narrow width and `IssueForm` keeps its held attempt and values across a breakpoint change. Changing the service in the selector clears the pin. `showPanel` uses `panelShown`.
  - `useCommand` uses `attemptState` instead of its own refs.
  - Cancel on the revoke step also clears the open panel, so a narrow layout returns to the credential list and focus reaches the credential row.
- `tools/service-accounts-qa.mjs` (new, committed) and a README line: a browser check against a real API with `--seed sample`, with four scenarios:
  - **flip:** a held issue across 1440 → 1000 → 1440 px.
  - **narrow:** a held issue retried at 1000 px.
  - **inflight:** Reload while the issue POST is in flight, then the response is lost.
  - **cancel:** Cancel on the revoke step at 390 px.
- No change to the command client, the API or the copy.

The tool generates a random password at run time for the activation it performs and the sign-in; it contains no password literal. It is not part of `npm test` and must be run by hand (see the console README).

The project has no DOM library (no jsdom, happy-dom or testing-library), so a node-run component test of `LiveServices` is not possible without adding a dependency. The `Live.tsx` wiring is therefore locked by the committed browser check above. The attempt rule is locked by node unit tests on `attemptState`.

## Checks

| Check | Result |
| --- | --- |
| Unit test "a submitted issue stays on screen when the layout narrows" | failed with the pre-fix behaviour (`pinIssue` returning `open: false`): 6 pass, 1 fail; passes after the fix |
| Unit test "a reload while an issue is in flight keeps its attempt for the retry" | failed with `attemptState` renewing during a request (the rule `useCommand` had): 8 pass, 1 fail; passes after the fix: 9 pass |
| `tsc --noEmit`, `npm test`, `npm run build` | clean; 796 pass, 0 fail; ok |
| `service-accounts-qa.mjs` on pre-fix main `Live.tsx` | 2 pass, 11 fail: flip and narrow lose the form (`30\|\|Issue credential`, no form), inflight sends a new key, cancel leaves the list hidden and focus off the row |
| `service-accounts-qa.mjs` after round 1 only | 9 pass, 4 fail: inflight (new key, no secret) and cancel (list hidden, focus off the row) |
| `service-accounts-qa.mjs` after the final fix | 20 of 20 runs, 13 pass, 0 fail each |
| `service-accounts-qa.mjs` after switching to a generated password | 1 run, all four scenarios, 13 pass, 0 fail |
| `qa316check` after the final fix | `back` 20 of 20, `narrow` 20 of 20, `redo` 20 of 20 passed |
| Held-retry step with the screenshot (`qa316cap2`) after the final fix | 30 of 30 passed |
| Round 1 only, same step | 30 of 30 passed; 30 of 30 with tracing, 3 of which had the narrow flip inside the held-retry window and kept the same `IssueForm` |
| `qa269` M04 runner after the final fix | 3 of 3 runs, 49 pass, 0 fail each (round 1: 5 of 5) |
| HYB-293 runner for current main (`qa293on307`, groups B C D A) after the final fix | B 32/0, C 27/0, D 54/0, A 21/0 (pass/fail) |
| Older HYB-293 runner (`qa293on297`), round 1 | B 28/2, C 22/1, D 54/0, A 21/0; the three failures are timeouts clicking the pre-HYB-307 "End other sessions" button in M05, which HYB-307 replaced with a confirmation dialog; `qa293on307` differs from it only in those M05 steps |

## Step-up refusal and the idempotency journal

Root checked the reviewer's hypothesis in `apps/api/server/commands.go:237-241`. When step-up is required, the server's command hook returns `stepUpRefusal` before `s.commands.Execute(call)` runs. So a `reauthentication_required` answer reaches neither the command engine nor its idempotency journal, and never stores the key. Resending the same attempt after reauthentication is therefore a first execution, not a replay.

## Residuals

- [HYB-333](https://linear.app/hybrid-billing/issue/HYB-333) (M04 unknown-outcome credential issue leaves an unseen secret without a truthful trace) carries two review findings:
  - **N2:** an issue whose outcome stays unknown can leave a committed credential whose secret was never shown and which is not in the "Secrets not confirmed" reminder (seen in the pre-fix `redo` run).
  - **N5:** `CreateForm` does not hold an unknown outcome (`holdUnknown` is off).
  - Cancel on the revoke step opened from the "Secrets not confirmed" reminder sets focus to the credential row, falling back to `#svc-list-title`. That heading does not exist in the credentials view, so if the row is not on screen, focus is lost.
- During a transient or real narrow flip, the stepped panel's focus rule moves focus to the issue heading. This was seen in 3 traced runs in round 1 and is the existing stepped-panel behaviour.
- `CreateForm` follows the same panel rule and loses typed values on a breakpoint change. It is not changed here.
- Leaving the stepped panel with Back at narrow width still discards a held attempt, as before; this is an explicit user action.
- `service-accounts-qa.mjs` needs a fresh API data directory per run. It was run only on mac_mini with chrome-headless-shell 1243.
- Root review passed round 1 and round 2; root recorded the step-up finding above.

## Root final verification (2026-11-19)

- Rebased on current main without conflicts; tsc clean, npm test 801/801, build ok, OKF status test ok; no password literal in the committed tool.
