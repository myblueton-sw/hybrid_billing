---
type: VerificationRecord
title: Current approval prerequisite verification
description: Records non-consuming current approval checks and the remaining W-23 integration work.
status: draft
---

# Scope and result

[HYB-225](https://linear.app/myblueton/issue/HYB-225) adds `Registry.CheckCurrent` as the first prerequisite in the [plan](./plan.md). This verifies the current shared approval rules without consuming an execution key or replaying historical success. It does not authenticate a caller, reserve an execution, prove downstream readiness or complete W-23 integration. The registry's monotonic clock watermark may advance.

# Executed checks

Executed on macOS arm64 with Go 1.26.4 in the isolated ticket worktree. Other operating systems remain unverified.

| Check | Result |
| --- | --- |
| `go test -race ./modules/approvals -count=1` | PASS, 1.846 seconds |
| Same module race run with coverage | PASS, 2.122 seconds; module statement coverage 99.7%; CheckCurrent 100% |
| `go vet ./modules/approvals` | PASS |
| Disposable current-check mutation baseline | PASS |
| Four targeted decision/state mutations | All four killed by intended assertions; no build failures counted as kills |

The tests exercise repeated and concurrent non-consumption, identifiers already used with different execution content, historical success after authority revocation, exact expiry and UTC authority date boundaries, clock rollback, expected subject dimensions, missing and foreign requests, invalid forms, pending/rejected/withdrawn/superseded states, item exclusion and concurrent revocation. Existing Execute still enforces consumption and historical idempotent replay independently.

The portable Python runner copies only required source and Go manifests into a temporary directory. The four mutations skip current checks, write an execution key during a check, bypass checks after historical success and skip workspace/actor validation. Root inspected each assertion failure: eligibility/expiry refusal bypassed, changed records, revoked authority accepted and invalid input mishandled respectively. These four mutations are focused evidence, not exhaustive mutation coverage.

Local executed logs are `/private/tmp/hyb225-check-tests.log`, `/private/tmp/hyb225-check-final.log`, `/private/tmp/hyb225-check-coverage.out` and `/private/tmp/hyb225-current-mutants/`. They are local verification artifacts, not committed product files.

# Independent reviews

The read-only reviewer `/root/refusal_review` independently reviewed the public contract before implementation and returned scoped PASS. The reviewer required ID shape validation without reservation, explicit authentication and downstream-execution limits, and no historical-success bypass.

After implementation the same independent reviewer inspected code and tests and returned scoped PASS with no blocker. The reviewer inspected the final coverage log but did not execute tests; the executed results above were run by root. A further independent read of the mutation runner and verification record found no blocking verification flaw; root corrected the reported mutation-to-failure ordering. The verdict covers this prerequisite only, not HTTP, producer integration, persistence, production policy or complete ticket acceptance.

# Remaining work

Service-owned immutable producer binding, authenticated grants, stage prerequisites and execution boundaries remain open. Adjustments, protected changes, recognition attribution, screens, persistence decisions and independent system QA remain in the original ticket scope. The follow-on integration design is under independent review. No external effect, deployment, merge or owner screen acceptance occurred.
