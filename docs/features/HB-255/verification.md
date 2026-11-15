---
type: VerificationRecord
title: HB-255 authentication policy verification
description: Actual checks run for the W-42 increment modules/authpolicy, plus review outcomes and residuals.
status: final
---

# Scope

[HB-255](https://linear.app/hybrid-billing/issue/HB-255) adds `modules/authpolicy`, following [plan.md](plan.md):
- **Policy.** Per-workspace revisions with allowed methods, an optional issuer list, an MFA scope, a maximum authentication age and step-up rules.
- **Evaluation.** Forms first, then refusals in a fixed order. Each decision carries the revision and a security version, and `Current` supports the recheck obligation.
- **Actors.** Every proposal, approval and withdrawal is made by a workspace administrator whose authentication passes the current revision. A revision's approver must also pass the new revision. Before revision 1, only revision actions are allowed. Every applied change bumps the version and drops the workspace's pending proposals.
- **Bulk revocation.** The whole scope is blocked at apply. Plan IDs are generated per workspace. Chunk reports are ordered, idempotent and resumable. Lifts are dual-controlled by people outside the scope.

Base: main `9b95ea7`. Machine: mac_mini.

# Checks run (2026-11-16, this machine)

| Check | Result |
| --- | --- |
| `gofmt -l`, `go vet ./modules/authpolicy/` | clean |
| `go test -race -count=10 ./modules/authpolicy/` | pass, including concurrent evaluations, chunk reports and proposals |
| `GOOS=linux GOARCH=386 go test -c -o /dev/null ./modules/authpolicy/` | pass |
| `go test -race ./...` | every package passes |
| Statement coverage | 100.0% |
| Property test | 150 seeds of random revisions, plans, lifts, chunk reports, clock moves and evaluations against an independent decision, block, version and progress model |

# Mutation testing

| Run | Mutants | Did not build | Survived |
| --- | --- | --- | --- |
| First (before the change-review lows) | 368 | 24 | 14 |
| Final code | 370 | 24 | 8 |

These were killed by literal boundary tests:
- step-up copying;
- the exact global entry bound;
- the applied-proposal window;
- the lifted-plan window;
- duplicate step-up actions.

The remaining 8 are equivalent:
- 4 changes to the internal proposal encoding, which change the deduplication key consistently;
- `Round(0)` → `Round(1)`;
- 2 changes to `cut < 0` that only affect an empty workspace part, which `validID` refuses anyway;
- `n > 0` → `n >= 0`, where zero is already refused by the leading-zero rule.

# Reviews

- **Design review** (independent reviewer, read-only): rounds 1–4 FAIL, round 5 PASS with lows folded in. Details are in plan.md.
- **Change review** (a separate independent reviewer, read-only): PASS, with lows L1–L6 folded in.

# Residuals and exposure blockers

- Wiring into `modules/identity` (`Access`, `Recheck`, session opening) and `modules/auth`. Blocking downloads, jobs and tokens already in progress (HB-42-V) depends on that wiring and is NOT_RUN.
- Not built here:
  - protocol adapters, verifiers, password storage, MFA enrolment;
  - recovery;
  - bulk invitation;
  - persistence, audit records and screens.
- Owner decisions listed in plan.md, all provisional.
- SEC01, AUD-AR-09, AUD-DB-08 and AUD-QA-04 are NOT_RUN at system level.
