---
type: VerificationRecord
title: Approval subjects, decisions and execution gate verification
description: Records what was executed and reviewed for the first increment of work item W-23, the approval register with two-administrator governance, digest-bound subjects, human decisions with authorities and limits, and an execution gate that re-checks and consumes once, with results, the mutation check, provisional choices and what remains unverified.
status: draft
---

# Delivered behavior

[HB-221](https://linear.app/hybrid-billing/issue/HB-221) adds the Go package `modules/approvals` from main `3aee5bd`. It uses the standard library and the primitives `decimal` and `exact`, and imports no other module. A registry with an injected clock and one mutex keeps, per workspace:

- **Governance.**
  - A workspace is opened once with two or more human administrators.
  - Policies, grants and administrator changes are proposed by one administrator and confirmed by another, with re-checks at confirmation. A grantee can be neither proposer nor confirmer. There are always at least two administrators.
  - Revocation is immediate, by a human administrator or by the grantee.
- **Policies per stage.** The stages are `charge_approval`, `erp_final_approval`, `issue` and `deliver`. Each policy sets:
  - the number of approvers (1 to 3);
  - the required digest kinds, which must include `result`;
  - a validity.

  A new revision supersedes the stage's pending and approved requests.
- **Subjects.** A subject has a stage and a target, the author, the digests, an amount, which may be negative, and an optional bulk manifest of `{item, scope, amount, digest}` entries. A new request for the same stage and target supersedes earlier ones.
- **Decisions.**
  - Only humans decide, and the author and the requester are refused.
  - Approval needs an active authority for the stage and currency whose limit covers the covered amount. That is |amount| for a single subject, and for a bulk subject the gross sum of the absolute amounts of the entries left after all exclusions so far.
  - The union of exclusions may never leave nothing. A reject ends the request.
- **Execution gate.**
  - The checks run in a fixed order: not approved, superseded, withdrawn, policy changed, expired, subject changed (live digests, amounts and item digests), authority lost, item ineligible, already executed.
  - Consumption is per stage and key across all requests.
  - Replays re-check, except after a reported success. Execution identifiers are unique.
- **Reports.** A report is `succeeded` or `failed` per execution. A `failed` may be followed once by a `succeeded` from a replay, and the earlier outcome is kept.

Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviewed the plan twice and the change once (see the plan). Machine: mac_mini. Nothing calls the package.

# Executed checks

All checks ran on 2026-11-14 on the mac_mini (darwin/arm64, Go 1.25.7) in the ticket worktree.

| Check | Result |
| --- | --- |
| `go test ./modules/approvals` | ok |
| `go test -race -count=10 ./modules/approvals` | ok |
| `GOOS=linux GOARCH=386 go test -c ./modules/approvals` | builds |
| `gofmt -l modules`, `go vet ./modules/approvals` | clean |
| `go test -race -count=1 ./...` | 35 packages ok, no failures |
| Statement coverage of `modules/approvals` | 99.6% |
| Source scan (no binary fractions, no comments, imports only the standard library and the primitives, under 1,500 lines) | ok |

The tests include:

- **The HB-23-V scenarios.** After approval, each of the following blocks execution:
  - a concurrent new request;
  - a withdrawal;
  - an ERP draft digest change with no new request;
  - another amount, target or stage, or a missing digest;
  - a revoked authority;
  - a policy change;
  - expiry, with the last valid instant allowed.
- **Separation of duties.**
  - Self-approval by the requester or the author, and decisions by an assistant or a system, are refused.
  - Two approvers are needed when the policy says so.
  - A late reject is refused.
- **Governance.** A single administrator acting alone is refused, as are a grantee confirming its own grant, a removed administrator's proposal, a skipped revision and too few administrators.
- **Bulk subjects.**
  - Limits are checked against gross absolute sums, exclusions combine as a union, and excluding the last entry is refused.
  - Excluded or unknown items, and changed item amounts or digests, are refused.
  - A failed act allows only a replay. A single subject over an executed bulk item is refused.
  - Withdrawal and supersede of a partly executed bulk request are covered.
- **Execution and reports.** The check order, replays, reports, concurrent executions consuming once, every form and room check, and a pinned subject digest.

A property test of 300 seeds × 80 calls checks request states, decision and execution refusals, and consumption against an independent model.

Mutation check (`mutate221.py` on the final production code, with survivors rerun after the last tests): 650 mutants, 610 killed and 30 did not build. 10 survived, all classified as equivalent:

- 6 bounds that parsing already enforces, or `Round` and absolute-value forms that give the same results;
- 1 repeat comparison where approve and reject can never share a reason;
- 1 covered-sum accumulation, which is compared by absolute value either way;
- 2 date bounds already enforced by the date check.

# Acceptance status

HB-23-V is covered at package level. NOT_RUN as a system test. AUD-AR-06 (delivered with HB-208), AUD-AR-07 and AUD-QA-01 are not covered by this ticket.

# Provisional choices awaiting the owner

- Approver counts, required digest kinds, validity, and limits per approver, currency and dates.
- Assistants and systems may request and execute, but never administer, decide or revoke.
- Every administrative change needs two administrators; revocation needs one.
- Withdrawal is allowed until execution. A stage and key are never re-run.
- Any human other than the author and the requester may reject.
- The register's administrators and authorities are separate from `modules/auth` until integration.

# Not done or unverified

- The change-review fixes were not reviewed again.
- Independent DBA and QA verification were not done.
- The caller obligations are UNVERIFIED:
  - opening a workspace under controls;
  - actor authentication and kinds;
  - the author being a human;
  - live digests;
  - execution identifiers as idempotency keys;
  - one identifier space for targets and items.
- Not built: stage prerequisites, AR-07 adjustment drafts, QA-01 protected changes, the ERP recognition cadence (ERPC-01 to ERPC-08), the `modules/auth` link, persistence, screens and external approval methods.
