---
type: VerificationRecord
title: O03 issuance setup browser test against the dev server
description: Why TestIssuanceSetupActualScreenBrowser failed against the dev server and passed against the built console, the correction, and the runs that show it.
status: stable
---

# Outcome and ownership

HYB-351, machine:mac_mini, 2026-11-24. The failure was not a product defect. The test server ran out of its per-person command allowance. The test now starts its server with a larger allowance, and the whole server-backed browser suite passes against one served console. No product code changes.

# Cause

The command layer allows each person a burst of 60 commands and refills one per second (`CommandLimits`, `apps/api/server/server.go`). The test server runs on a test clock that moves only when the test moves it, so during one phase of the check nothing is refilled and each person has 60 commands in total.

Reads on the O03 documents view are commands. Against the built console the busiest person of the changes phase sends 48. The dev server runs React's development behaviour, in which effects run twice on mount, so each read is sent twice: the same person had sent 67 when the run stopped. The refused reads (`429 rate_limited`, seven of them in the evidence of the failing run) left the documents view without its field cards, which is what the check was waiting for.

Counts measured on the English run, commands sent per person in the changes phase:

| Console served as | Requester | Verifier | Result |
| --- | --- | --- | --- |
| built preview, allowance 60 | 48 | 48 | passed |
| dev server, allowance 60 | 67 when it stopped | 56 | failed at `change-request-2` |
| dev server, allowance 600 | 84 | 84 | passed |
| built preview, allowance 600 | 48 | 48 | passed |

# Correction

- `apps/api/server/issuancesetupscreen_browser_test.go` starts its server with ten times the default burst. The limit itself is not what this test checks; it has its own tests.
- `tools/o03-setup-live-qa.mjs` records in its evidence how many commands each person sent (`sent`), so the margin is visible in every run.

# Runs

Isolated in-process server, headless Chromium, synthetic data, no real account or customer data; not an owner acceptance of any screen.

1. `TestIssuanceSetupActualScreenBrowser`, en, ko and ar, against the dev server: passed.
2. The same against the built preview: passed.
3. The whole suite (`go test ./server/ -run 'Browser|GrantScreens'`) against the dev server: 17 passed, 1 skipped (`TestFinancialDraftLiveBrowserJourney`, which needs its own runner settings), 0 failed.

The whole suite therefore passes against one served console, the dev server. Two of its tests import console source and cannot run against a built preview (see `docs/features/HYB-350/browser-test-failures.md`).

# Observations left with the owner

- Against the built console the changes phase used 48 of 60 commands for one person before this change. That is the production allowance: a person who works through several protected changes in quick succession on the O03 documents view comes within a dozen reads of being refused for a moment. With the real clock one command is refilled each second, so the margin recovers quickly, and no refusal was observed. Whether the documents view should send fewer reads, or the allowance should be an installation setting, is a product question this record does not settle.
- A refused read on that view is not silent. Read afterwards in the code (`setup/Rules.tsx`, `PartFeedback`): the section shows an alert with the general sentence for a failed read and the code `rate_limited` beneath it, and the field cards are absent because the read gave nothing to show. The sentence does not say that the cause is the number of requests or that waiting a moment is enough; the code is the only indication. Whether a rate-limited read deserves its own sentence is a copy question for the owner's screen review. An earlier version of this record said there was no message; that was written before the code was read and was wrong.
