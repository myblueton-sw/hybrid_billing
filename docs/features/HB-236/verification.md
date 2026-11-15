---
type: VerificationRecord
title: Assistant tools, approval cards and job ledger verification
description: Records what was executed and reviewed for the first increment of work item W-32, a register that exposes command-registry commands to the assistant by stage under dual control, runs read, draft and simulation tools through the command executor on behalf of the job owner, turns requests into approval cards that eligible humans confirm with their own answers, keeps a job ledger apart from conversations, enforces per-job budgets and bounds results. The record lists results, the mutation check, provisional choices and what remains unverified.
status: draft
---

# Delivered behavior

[HB-236](https://linear.app/hybrid-billing/issue/HB-236) adds the Go package `modules/assistant` from main `5311c40`.

- **Tools.** A human proposes and another human approves the exposure of one catalogued, supported command at one stage. The stage is checked against the command's own definition: `read` needs a non-mutating `view` command; `draft` and `simulate` need a mutating, non-approval `view` or `request_approval` command; everything else is request-only. Commands of protected permission kinds are never exposed. Withdrawing or changing an exposure expires its pending cards. Exposures carry revisions.
- **Conversations and jobs.** Conversations have their own owner and expiry, at most 365 days ahead. Jobs have fixed identifiers, owners, a duration and serial numbers. Owners attach jobs to unexpired conversations and can resume from new ones. Deleting a conversation keeps the job ledger and the commands journal. A system actor purges the ledgers of jobs closed or expired longer than the retention ago.
- **Calls.** An assistant actor calls inside an attached, unexpired conversation with the job owner's identity answer. Request tools and decimal inputs are refused. Calls, distinct targets and duration are budgeted, and the mutex is held across execution. Calls run through `commands.Execute` with the configured client. Mutating calls carry a domain-tagged key over workspace, job, job serial, tool, target, expected revision and sorted inputs, so resent calls replay. Results are reduced to allowed fields, at most a configured number of them, each cut on UTF-8 boundaries with a truncation flag.
- **Cards.** An assistant prepares cards for request tools. A card is bound to a digest over workspace, card, card serial, job, tool, exposure revision, target, expected revision and sorted inputs, and executes nothing.
- **Confirmation.** A human confirms a card with their own answer and no client, and the rationale is passed only when the command needs one.
  - Success confirms the card.
  - An error that carries an operation makes the card uncertain. Only the same confirmer may repeat, and others are refused.
  - Any other error leaves the card pending.
  - At most 10 attempts are allowed per card. A job past its duration expires its pending cards.
  - Humans can decline pending and uncertain cards. The ledger records the fixed outcome `declined`.

# Commands and results

All runs were made in the worktree `/private/tmp/hb236` on 2026-11-15 with Go 1.25.7 on darwin/arm64.

| Check | Result |
| --- | --- |
| `gofmt -l modules/` | no output |
| `go vet ./modules/assistant` | pass |
| `go test -race -count=10 ./modules/assistant` | pass |
| `GOOS=linux GOARCH=386 go test -c ./modules/assistant` | builds |
| `go test -race -count=1 ./...` | 49 packages, all pass |
| Coverage | `modules/assistant` 100.0% |
| Mutation testing | 470 mutants of `registry.go` and `calls.go` on the final code: 435 killed, 23 did not build, 12 survived |

The 12 survivors were read and judged equivalent:

- Five mutants change the number base of serial numbers to 11. The serials are identical below 10 and stay injective above.
- Two mutants swap domain tags between the card digest and the confirmation key. These keys have different arities and never meet.
- Two mutants use `Round(1)` instead of `Round(0)`; the effect is the same.
- One mutant uses a `[][3]` limits array.
- One mutant uses `<=` when sorting inputs whose names are unique.
- One mutant lets the UTF-8 cut reach index -1, which cannot happen for valid UTF-8.

Tests use the real `modules/identity`, `modules/gate` and `modules/commands` registries with synthetic handlers. They show:

- another customer's target is refused by the gate;
- the assistant client is recorded on calls and absent on confirmations;
- the author confirming an approval card is refused, and an independent approver succeeds;
- resent drafts replay;
- reused job and card identifiers after a purge execute afresh;
- 128-byte identifiers work.

Two property tests run 200 seeds each:

- random calls against an independent model of call and target budgets, duration, closure and ledger entries;
- random confirmations and declines by two people against an independent card state model, with refused, uncertain and successful outcomes and expiry.

# Reviews

- Design review: three rounds by a separate read-only reviewer: FAIL, FAIL, PASS ([plan](plan.md)).
- Change review: two rounds by a fresh read-only reviewer: FAIL (identifier reuse after purge, request identifier length), PASS.

# Provisional choices for the owner

- Conversations expire at most 365 days ahead.
- 10 confirmation attempts per card, shared by all confirmers.
- `request_approval` commands may run at draft stage.
- Changing an exposure suspends it until it is approved again.

# Not verified and residuals

- **Human presence and identity.**
  - Human presence at confirmation must come from the transport and screen.
  - Tool administration, Decline and Prepare trust the caller's actor rather than a gate answer.
  - There are no step-up answers.
- **Grants and handlers.**
  - The assistant client's grant configuration is not set up here.
  - The handler contract requires money fields to be decimals.
  - Approvals do not name the revision they approve.
  - One refused confirmer can use up a card's shared attempt limit.
- **Exposure scope.**
  - Cost views and downloads at read stage are not exposed.
  - Costs in currency are not tracked.
- **Not built in this increment.**
  - The model connection, prompts and the chat interface.
  - Conversation text and attachments.
  - Demo isolation (W-38).
  - Persistence, including serial numbers.
  - Screens.
