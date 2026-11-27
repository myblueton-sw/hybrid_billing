---
type: Runbook
title: HYB-384 — how to run the M04 rendered checks
description: The service settings and fixture the M04 rendered checks need, what was wrong on main, and which checks still cannot run.
status: draft
---

# Scope and ownership

[HYB-384](https://linear.app/myblueton/issue/HYB-384), machine:mac_mini, root (Claude). Written 2026-11-27 from main `518e9916`.

# Setup

Start an owned disposable service with the sample seed and these settings, then run each check from `apps/console` with `--url`, `--activation` and the Playwright module and Chromium paths:

```
hb-api --listen 127.0.0.1:8820 --data <fresh directory> --seed sample \
  --console-origin http://127.0.0.1:8831 \
  --service-catalog apps/console/tools/fixtures/m04-service-catalog.json \
  --service-declaration-versions 20 \
  --service-query-scan 100
```

- `tools/fixtures/m04-service-catalog.json`: synthetic catalog for `sample-workspace` and `workspace`. Audience `test-service-api` with scopes `read` and `read-01` to `read-12`, and audiences `test-z-api-01` to `test-z-api-11`. The pickers need more than one page of each, as the checks expect.
- `--service-declaration-versions 20`: the declaration-history check reads eleven retained versions; the default keeps five.
- `--service-query-scan 100`: the scope-search check expects the scan to stop at 96 work units; the default budget is 1000.
- Use a fresh data directory for every run; the activation link is single use.

# What was wrong on main

- None of the settings above were documented, so every check failed at the Audience picker.
- `m04-completion-qa.mjs` issued 27 credentials to one service account. The service refuses more than ten active credentials per service account (`CredentialsPerServiceUser`, `apps/api/server/server.go`). The setup now spreads them over three accounts (10, 10 and 7). The global "awaiting collection" list still holds 27, and the primary account's list holds 10.
- `m04-completion-navigation.mjs` and `m04-completion-declarations.mjs` used the list structure from before the shared list pattern (HYB-376 L7, PR 508). They now use `.svc-list-body` and `.list-pager-nav`.
- On a refused setup command the runner now prints the service's answer, and on failure the stack.

# Result with this setup (2026-11-27)

| Check | Result |
| --- | --- |
| `m04-completion-qa.mjs --group navigation` | pass (4 scenarios) |
| `m04-completion-qa.mjs --group remaining` | pass (7) |
| `m04-completion-qa.mjs --group declaration` | pass (3) |
| `m04-consumer-qa.mjs`, `m04-errors-qa.mjs` | pass |
| `m04-lists-qa.mjs` | 109 passed |
| `m04-completion-qa.mjs --group actions` | pass (3 scenarios) since HYB-386: the revoke step-up journey routes the credential detail read to report reauthentication required until the real password confirmation, instead of the clock control that was never in the repository. The routed answer also reports rotation as needing reauthentication, as a real stale session would; the return to ready comes from removing the route, not from the reauthentication, so this check proves the screen's behaviour, not the server's expiry. The server side (a step-up older than the default 15-minute window becoming `reauthentication_required`, with revoke unavailable) is covered by `apps/api/server/servicequery_credential_controls_test.go` and `servicequery_detail_test.go`; configured windows by `stepup_test.go` (`TestStepUpWindows`). |
| `service-accounts-qa.mjs` | cannot run: written on 2026-11-19 for the earlier create and issue controls; it covers held-issue, unknown-outcome and retry journeys no newer check covers, so it is to be updated, not retired (HYB-387). |

The actions group was fixed in HYB-386; the service-accounts check is HYB-387.
