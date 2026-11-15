---
type: ImplementationPlan
title: Assistant work entry point with dual-controlled tool exposure from the command registry, server-side approval cards confirmed by eligible humans, a job ledger separate from conversations, per-job budgets, replay-safe calls and bounded tool results
description: Defines the first increment of work item W-32, a register that exposes command-registry commands to the assistant by stage under dual control and never exposes approval or protected-field commands, runs read, draft and simulation tools through the same command executor and permissions as the admin UI on behalf of the job owner, turns requests into approval cards that an eligible human confirms through the same executor, keeps conversations with their own retention apart from a job ledger with fixed job identifiers, enforces per-job call, target and duration budgets, replays resent calls and passes only bounded results to the model.
status: draft
---

# Scope and ownership

[HB-236](https://linear.app/hybrid-billing/issue/HB-236) starts work item W-32 (requirement HB-32; HB-32-D/I/V; the assistant parts of AR-05, QA-01 and QA-10) after W-31. Base: main `5311c40`. Root (Claude Code) owns planning, implementation and tests; a separate read-only reviewer reviews the plan and the change, because the register decides what an AI may do in the billing system. Machine: mac_mini.

Out of scope, recorded as residuals: human presence at confirmation (transport and screen), step-up answers, gate permissions for tool administration, the assistant client's grant configuration, the handler contract that treats `Context.Client` as an assistant call, cost views and downloads at read stage, the model connection, prompts and the chat user interface, storage of conversation text and attachments (this register keeps only conversation identifiers, owners and retention), real business commands (none is registered yet; tests use synthetic handlers), cost accounting in currency, demo-environment isolation and credential refusal of QA-10 (W-38 simulator), AR-05 source-field exclusion (delivered by HB-186/HB-188/HB-196), persistence and screens (P04, P06, A01).

# Observed baseline

- HB-32-D: the assistant uses the same API as the admin to read, draft, simulate, request approval, and lets eligible users approve, issue or correct; the LLM is neither an amount calculator nor an approver; export, conversation retention, model connection and write-tool exposure are staged.
- HB-32-I: restricted tools based on the CommandRegistry and server-side approval cards; conversation and job ledgers are separate and a fixed job identifier resumes work from a screen or a new conversation; no arbitrary SQL, HTTP, code or infrastructure tools.
- HB-32-V: the same server controls hold for malicious source text, another customer's identifiers, missing explicit approval, resent commands and model failure; the LLM gets only in-scope aggregates, pages and job states, never 100,000 rows; budgets for call counts, cost, targets and execution time.
- QA-01: an LLM never generates or replaces payment information. QA-10: assistant conversations, attachments and tool-execution references have their own retention class, separate from billing work, approvals and audit records; deleting a conversation never removes business evidence.
- `modules/commands` (HB-176) registers definitions with one permission, approval, mutation and rationale flags and typed fields, and executes a call only through the gate with the caller's identity answer, an idempotency key and an expected revision for mutations; its catalogue lists supported commands.

# Decision

A new package `modules/assistant` follows the sibling conventions: injected clock, one mutex, workspace isolation, `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (`Refusal{Reason}`), `ErrLimit`; check order form, lookup, repeat, rule, room. `NewRegistry(Config{Commands, Client, Protected}, Limits{Workspaces, Tools, Jobs, Conversations, Cards, Calls, Targets, Duration, Retention, Fields, Bytes}, clock)`. `Commands` has the `Catalogue` and `Execute` methods of `modules/commands`; `Client` is the client identifier under which assistant calls run, so the command gate intersects the client's grants with the user's; `Protected` is a non-empty list of permission kinds (payment accounts, document recipients, tax identifiers, templates) whose commands are never exposed. The register's mutex is held across `Commands.Execute`, which never calls back, so budgets cannot be overshot by concurrent calls.

## Tools

`ProposeTool(workspace, Tool{Command, Stage, Results}, actor)` by a human and `ApproveTool(workspace, command, actor)` by another human expose one catalogued, supported command at one stage, with the stage checked against the command's own definition, not the proposer's word:

- `read`: a non-mutating command whose permission action is `view` (cost views and downloads are not exposed in this increment, so export stays staged);
- `draft` and `simulate`: a mutating, non-approval command whose permission action is `view` or `request_approval`;
- `request`: any other catalogued command, including approvals, issuing, delivery, downloads and policy changes; the assistant can only prepare cards for it.

Commands whose permission kind is protected are never exposed (`protected_command`), at any stage. `Results` lists the result fields the model may see (they cannot be checked against the catalogue, which describes input fields only). `WithdrawTool(workspace, command, actor)` by a human ends an exposure and expires its pending cards; each exposure has a revision, and proposing a change to an exposed command suspends the current exposure and expires its pending cards until another human approves the new revision. There is no tool for SQL, HTTP, code or infrastructure because only catalogued commands can be exposed. Granting tool administration through a gate permission is a residual; so is configuring the assistant client's grants so that they exclude issue, delivery and approval, which the gate then enforces independently of this register.

## Conversations, jobs and retention

`OpenConversation(workspace, Conversation{ID, Owner, Expires}, actor)` by the human owner records a conversation with its own retention (`Expires` after now and at most 365 days ahead; the cap is a provisional assumption, conversation retention is an open decision of HB-32). `CreateJob(workspace, Job{ID, Purpose}, actor)` by a human creates a job owned by them with a fixed identifier. `Attach(workspace, job, conversation, actor)` by the owner links a job to an unexpired conversation of theirs; a job can be resumed from any number of conversations. Expired conversations cannot be attached or used for calls or cards (`conversation_expired`). `DeleteConversation(workspace, id, actor)` by the owner, or by a system actor after expiry, removes the conversation record and its links. The job ledger is its own assistant retention class: it keeps job, tool, target, card, operation and outcome references, never conversation text, field values or answers, and `Purge(workspace, actor)` by a system actor removes ledgers of jobs closed longer than `Limits.Retention` ago. Business evidence lives in the `modules/commands` journal and in the owning modules, linked by operation identifier, and is never removed here. `CloseJob` ends a job and expires its pending cards; a job past its duration budget counts as closed for `Purge`.

## Tool calls

`Invoke(workspace, job, Invocation{Conversation, Tool, Target, Expected, Input, Answer}, actor)` by an assistant actor, inside an unexpired conversation attached to the job. The answer must be the job owner's identity answer for this workspace (`not_owner`). The register then:

- refuses tools that are not exposed (`not_exposed`), `request` tools (`request_only`), protected kinds and closed jobs (`job_closed`);
- refuses any decimal input value (`model_amount`): amounts are never supplied by the model; they come from server-side simulations or targets referenced by identifier; commands must declare money fields as decimals (handler contract, residual), since amounts sent as integers or text cannot be recognised here;
- enforces the job's budgets before executing: calls (every attempt that reaches the executor counts, including errors and replays), distinct targets and duration since the job was created, each `budget_exhausted`;
- builds the command call with the configured client, a request identifier and, for mutating tools, an idempotency key: the hex SHA-256 of a domain tag and the length-prefixed workspace, job, tool, target, expected revision and sorted input values, so a resent identical call replays the earlier operation and keys never collide across workspaces;
- executes it through `Commands.Execute`, so the gate, permissions, scope, revision and idempotency rules are those of the admin path; errors are returned with the command error code;
- records the call in the job ledger and returns to the model only the allowed result fields, sorted, at most `Limits.Fields` of them, each cut on a UTF-8 boundary to `Limits.Bytes` with a truncation flag, plus the job state.

Malicious text in results is data: the register never interprets result values.

## Approval cards

`Prepare(workspace, job, Card{ID, Tool, Target, Expected, Input, Note}, actor)` by an assistant actor inside an unexpired attached conversation creates a pending card for a `request` tool (at most `Limits.Cards` per job; decimal inputs are refused as above), bound to a digest over the workspace, card, job, tool, exposure revision, target, expected revision and sorted inputs. `Note` is the assistant's explanation, kept with assistant provenance in the assistant retention class (removed by `Purge`), never passed as the approval rationale and never entering the commands journal. A card executes nothing.

`Confirm(workspace, card, Confirmation{Digest, Rationale, Answer}, actor)` by a human: the digest must be the one shown (`digest_changed`), the answer must be human, for this workspace and of the confirming person (`not_confirmer`), the exposure must still be current, and the rationale, when the command needs one, is the human's own. The card's command runs through `Commands.Execute` with that human's answer and no client (the human admin path; approval commands refuse a client anyway), with an idempotency key derived from the card and the confirming person. Outcomes: success (completed or accepted) confirms the card; an error that carries an operation (an `internal_error` whose effect is unknown) moves the card to `uncertain`, where only the same confirmer may repeat (replaying the stored result) and everybody else gets `outcome_uncertain` until the operation is inspected outside this register; any other error (gate, shape or revision refusals, errors without an operation) leaves the card pending and records the attempt; the same person confirming a confirmed card gets the stored result; anybody else gets `already_decided`. Cards can no longer be confirmed once the job's duration budget has passed (`card_expired`). `Decline(workspace, card, reason, actor)` by a human ends a pending card, or an uncertain card once its operation has been inspected outside this register.

The register cannot prove that a human, rather than the assistant runtime using the owner's credentials, called `Confirm`: human presence at confirmation must be enforced by the transport and screen (A01) and by keeping `Confirm` off the assistant's tool router. This is a stated trust assumption and residual; a step-up answer issued after the card would need an identity change.

# Verification plan

Literal tests for every refusal and repeat; every one of the configured protected kinds refused (the kind identifiers depend on the gate catalogue); an uncertain confirmation blocking other confirmers; stage checks against command actions (an issuing command never runs at draft); decimal inputs refused; confirmation by an assistant actor or with another person's or workspace's answer refused; withdrawn tools' and closed jobs' cards not confirmable; keys that do not collide across workspaces; expired conversations unusable; purging closed jobs keeping the commands journal; synthetic commands through the real `modules/commands` registry and gate; another customer's target refused by the gate; protected and approval commands refused as tools; resent calls replaying; cards confirmed only by humans with their own answers; budgets; deleting conversations keeping the ledger; model failure modelled as an interrupted job resumed from a new conversation; bounded results; a property test of random calls and cards against an independent model of budgets and ledger entries; `-race -count=10`, a 386 build, gofmt, vet, the whole repository with `-race`, 100% coverage and mutation testing.

# Review history

- Design review round 1: FAIL (3 high, 7 medium, 3 low). Folded in: stages checked against the command's permission action (1); the human-presence trust assumption stated and `Confirm` bound to the confirmer's own answer (2); decimal inputs refused (3); confirmation without a client (4); card outcomes and replays (5); digests with exposure revision, re-checks and expiry (6); human rationale and assistant notes (7); domain-tagged keys over the workspace (8); exposure administration and staged exports (9); ledger retention and expired conversations (10); budgets and locking (11); bounded results (12); verification cases (13).
- Design review round 2: FAIL (1 medium, 4 low). Folded in: `uncertain` cards after errors with an operation (R2-1); money fields must be decimals, a residual (R2-2); notes in the assistant retention class (R2-3); card expiry with the job duration and purging of expired jobs (R2-4); a test for every protected kind (R2-5).
- Design review round 3: PASS. Folded in: uncertain cards can be declined by a human after inspection (R3-1).

# Change review resolution

- Change review round 1: FAIL (2 medium, 9 low). Resolved:
  - M1: every job and card gets a registry-wide serial number at creation. Call keys include the job serial; card digests include the card serial; confirmation keys use the card serial, digest and confirming person. Identifiers reused after `Purge` therefore never replay earlier operations. A test confirms and invokes after a purge and a reuse.
  - M2: request identifiers are built from serial numbers (`assistant-<job serial>-<call>`, `card-<card serial>-<attempt>`), so 128-byte job and card identifiers work. A test covers maximum-length identifiers.
  - L1: an identical proposal of an active tool returns the current exposure instead of suspending it.
  - L3: pending cards of a job past its duration are reported and handled as `expired`.
  - L4: declining an expired card gives `card_expired`.
  - L5: at most 10 confirmation attempts per card, counted across all confirmers (`budget_exhausted`); a card that reaches the limit is declined and prepared again.
  - L6: the ledger records the fixed outcome `declined`; the human's reason identifier stays on the card and is purged with it.
  - L7: replies carry the remaining call budget.
  - L8: repeated cards are compared with sorted inputs.
  - L9: a second property test drives random confirmations and declines against an independent card model.
  - Not changed: L2 (approval naming a revision) stays a hardening residual; a retried approval after a new proposal approves the new revision, which the approver can see in the returned view. Serial numbers live in memory, like the rest of this register; persistence is a residual.
- Change review round 2: PASS. Recorded as residuals: one human whose confirmations are refused can use up a card's shared attempt limit (N1, recovery by decline and a new card); the shared count is stated above (N2).
