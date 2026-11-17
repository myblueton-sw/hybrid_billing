---
type: VerificationRecord
title: HB-261 collected FOCUS reader gate verification
description: Actual checks run for phase 2c of the pre-launch plan (the session gate in front of the collected FOCUS readers), plus review outcomes and residuals.
status: final
---

# Scope

[HB-261](https://linear.app/hybrid-billing/issue/HB-261) puts the six collected FOCUS reader paths behind the Go session, following [plan.md](plan.md).

**`apps/api`** forwards GET requests only after these checks pass:
- the session;
- identity access;
- a `view` grant on `focus_data`.

Forwarding rules:
- an allowlist of request headers, plus the reader's local-client header and a shared secret;
- `Set-Cookie` and CORS headers dropped from responses;
- a 25-second response-header timeout.

The secret is written to an exclusive 0600 file at every start.

**The reader** can require the secret (`--gate-secret-file`) and compares it in constant time.

**The console** routes the reader paths to `apps/api` when `HB_GATED=1`.

Base: main `84685ef`. Machine: mac_mini.

# Process note

The design review took place after implementation, combined with the change review, instead of before it. This departs from the design-first order in the repository rules, and is recorded here rather than presented as compliant.

# Checks run (2026-11-16, this machine)

| Check | Result |
| --- | --- |
| `gofmt -l`, `go vet ./apps/api/...` | clean |
| `go test -race -count=5 -short ./apps/api/...` | pass |
| `go test -count=20 -short ./apps/api/server` | pass |
| `go test -race -short ./...` (as part of the suite) | pass |
| Statement coverage, `apps/api/server` | 100.0% |
| Python `pytest tests/focus_overview` | 51 passed, including the secret cases: required, wrong, short, rotated, missing and duplicated, plus the ungated reader |
| Console `npm test` | 326 of 326 pass |

**Manual end to end, against the real reader code:**
- `hb-api --reader /api/focus-overview=http://127.0.0.1:8836` sat in front of the worktree's `overview_http.py`, loaded into the primary checkout's package on port 8836 with its local configuration and the secret file.
- An anonymous request got 401.
- A request after signing in through redeem and login was forwarded. The reader returned its own `503 collected_unavailable`, because its data stack (PostgreSQL 25432, ClickHouse 28123, SeaweedFS 28333) was not running. The owner's long-running readers on 8806 and 8810 returned the same 503 for every path at the same time.
- A direct request to the gated reader without the secret got 403.
- Through the Vite dev proxy with `HB_GATED=1`, an anonymous request with `X-HB-Local-Client: 1` got the gate's 401.

No reader returned data during this ticket. The containers were not started.

# Review

- Separate independent reviewer (AI, read-only), security focus, design and change together: PASS with nine lows. All were folded in or recorded as residuals; see plan.md.

# Residuals

See plan.md:
- the gate holds only with both opt-in switches, until the phase 10 start script;
- the same-user trust boundary for the secret file;
- no workspace binding in the reader (tenant security);
- the widened cookie path reaches other local listeners;
- the onboarding service on 8793 is outside the gate;
- `--seed=none` has no reader grants.

# Planning cross-check (HYB-291)

Added by the [HYB-291 planning cross-check](../../reviews/HYB-291-planning-cross-check/report.md) on 2026-11-17. A record added by HYB-291; it changes no result above except where marked. The findings come from a read-only auditor; the HYB-291 writer agent (delegated by root) wrote this trace, the residual notes and the copy changes; the HYB-291 change review is their first independent check. Rows marked "(classified by HYB-301)" replace rows that HYB-291 left unclassified; the [HYB-301 trace classification](../../reviews/HYB-301-trace-classification/report.md) gives the evidence.

## Requirement trace

Source: No screen spec. IDs derived from the phase 2 gate scope in `docs/features/HB-257/plan.md` (Existing Python endpoints). Status meanings are in the [cross-check report](../../reviews/HYB-291-planning-cross-check/report.md#status-meanings). No acceptance test was run.

| ID | Status | Reason | Record |
| --- | --- | --- | --- |
| HB-03 | deferred | Addressed within pre-launch scope for the six reader paths, forwarded only after the session and the `view` gate action are checked; registry-wide and leakage acceptance not built (no workspace binding in the reader). | HB-261/plan.md |
| HB-36 | deferred | The gate holds only with both opt-in switches until the phase 10 start script; the onboarding service on 8793 is outside it. | HB-261/plan.md:24, :92, :93 |
| HB-07 | not applicable | (classified by HYB-301) HB-261 forwards the reader's response body unchanged (only request and response headers are filtered, `apps/api/server/readers.go`) and adds no mapping, conversion or display, so it cannot change FOCUS data meaning; HB-261 has no spec, and its ticket scope is the gate. HB-07 for the nine FOCUS reader screens themselves (received and target FOCUS versions kept apart, cost columns not mixed, null, missing and zero distinct, decimals preserved) is unassessed: it lives in the Python FOCUS import and reader screens built before HB-257, which phases 1–5 do not change. | [HYB-301 report](../../reviews/HYB-301-trace-classification/report.md) |
