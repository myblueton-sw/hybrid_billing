---
type: VerificationRecord
title: Billing findings and billing cases verification
description: Records what was executed and reviewed for the second increment of work item W-22, the billing finding register and the billing case register, with results, the mutation check, provisional choices and what remains undecided or unverified.
status: draft
---

# Delivered behavior

[HB-217](https://linear.app/hybrid-billing/issue/HB-217) adds the Go package `modules/cases` from main `bdd5915`. It uses the standard library only and imports no other module. A registry with an injected clock and one mutex keeps, per workspace:

- **Billing findings (AR-08).** Each finding records:
  - one of seven kinds;
  - a target of customer, contract, document and line, and resource (customer optional, forbidden for `unattributed`);
  - a period, an amount meaning (`cost`, `sale`, `difference`, `none`), and an amount and currency (signed only for a difference);
  - evidence, severity, owner, and a manual or named-detector source.

  Findings move through `in_review`, `confirmed_normal`, `adjustment_needed` and `resolved`. Every step carries a reason code or a remedy reference, an evidence digest, the actor, the time and a digest. The recorder and the latest reopener cannot confirm normal or require adjustment. Reopen reasons depend on the source state. Findings are operator-only, and a customer is refused before any lookup.
- **Billing cases (HB-22-I).** Inquiries and disputes on documents and lines, identified per customer. Each case keeps:
  - an owner and its history;
  - a response due time with original value, history and clearing;
  - public messages and answers, and internal notes in a separate identifier space;
  - closings with outcome and remedy reference, and reopens.

  Every transition has a client step identifier and a digest. Identical repeats change nothing.
- **Customer visibility (HB-22-V).**
  - A customer actor reaches only its own customer's cases. Any other address is `ErrNotFound`.
  - Customer views omit notes, the owner and owner history, the closer, the original due time, the due history, closings and the case digest. Operator entries keep their body and blank their actor and record digest. A remedy is shown only for an issued correction or credit document.
- **Dispute axis (PT-07 input).** `Disputes` lists, for operators only, the open or answered disputes of one customer naming one document, with a cursor. A case never changes a document or an obligation.

Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviewed the plan three times and the change once (see Independent review). Machine: mac_mini. Nothing calls the package, and the registry authenticates no actor; the caller must.

# Executed checks

All checks ran on 2026-11-14 on the mac_mini (darwin/arm64, Go 1.25.7) in the ticket worktree.

| Check | Result |
| --- | --- |
| `go test ./modules/cases` | ok |
| `go test -race -count=10 ./modules/cases` | ok |
| `GOOS=linux GOARCH=386 go test -c ./modules/cases` | builds |
| `gofmt -l modules`, `go vet ./modules/cases` | clean |
| `go test -race -count=1 ./...` | 32 packages ok, no failures |
| Statement coverage of `modules/cases` | 100.0% |
| Source scan (no binary fractions, no comments, no import of another module, under 1,500 lines) | ok |

The tests include:

- literal tests for every transition, refusal and visibility rule;
- the full case state table, and a check that refused calls leave the operator view unchanged in every state;
- a property test of 300 seeds × 120 calls of cases against an independent model, checking errors, state, due time, overdue, entries and customer views after every call;
- a property test of 300 seeds × 80 calls of findings against independent transition and reopen tables;
- concurrency tests under the race detector.

Mutation check (`mutate217.py`, second run on the final code), 700 mutants:

- 637 killed and 53 did not build.
- 10 survived, all classified as equivalent:
  - 4 page clamps `>`→`>=`, where truncation at equal length changes nothing;
  - a sort `<`→`<=` over unique keys;
  - 2 `Round(0)`→`Round(1)`, where both strip the monotonic reading of nanosecond times;
  - 2 date bounds and a parse check, where `time.Parse` and the format comparison already reject the same inputs;
  - a decimal-point scan from index 1, where a leading point fails the digit check either way.

# Acceptance status

- AUD-AR-08 at package level: a new-charge finding with meaning `none` and no difference is recorded and reviewed. Confirmed normal and adjustment needed are separate states. A resolution keeps its remedy kind, reference and evidence. NOT_RUN as a system test.
- HB-22-V inquiry permissions at package level: covered by literal and property tests. NOT_RUN as a system test.
- PT-07 dispute axis: input only (`Disputes`); the receivables display is not built.

# Provisional choices awaiting the owner

- The reason, kind, topic, outcome and remedy lists.
- Separation of duties limited to ConfirmNormal and RequireAdjustment (recorder and latest reopener).
- Customers may reopen closed cases without a time limit, so a dispute marker can be kept on a document indefinitely.
- The response SLA is set by hand with no default; a customer message after an answer and a reopen clear the current due time.
- Customer views hide the operator identities and the owner.
- A customer whose open hits the workspace case room gets `ErrLimit`, revealing only that the workspace is full.
- Internal notes are allowed on closed and withdrawn cases.

# Not done or unverified

- The change-review fixes (the error kinds, subject room, history digests, owner history and dispute cursor) were not reviewed again.
- Independent QA and DBA verification were not done.
- Persistence, screens (U04, U03, S06, B11), notifications, detector thresholds and system acceptance are not done.
- The remaining W-22 scope (AR-10, DB-10, payment terms PT-01 to PT-08, refunds, external authority locks) is not in this ticket.
