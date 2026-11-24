---
type: Design
title: Settling an unknown O03 setup write whose exact retry is refused
description: Facts about the command layer's replay order, the rule proposed for marking an unknown issuance setup write as not recorded, its limits, and what is left for the owner.
status: draft
---

# Outcome and ownership

HYB-353, machine:mac_mini, 2026-11-24. This document proposes one narrow rule for the console and asks for independent design review before any code changes. It changes no server behaviour and no public contract. The writes concerned are issuance setup writes of the O03 documents view (settings proposals and confirmations, rules, channels, protected changes, terms and the statements behind them), so the rule must never report a write as not recorded when it may have taken effect.

# The problem

The console keeps a write whose outcome it does not know as "unknown" and refuses other writes until it is settled (`setup/write-store.ts`, `setup/step-store.ts`). Two ways of settling exist: "retry", which resends the identical request so that the server replays its stored result, and "recover", which reads the current records and completes a write that the module shows as half done.

A retry keeps the phase "unknown" whatever the answer (`write-store.ts`, `run`; `step-store.ts`, `run` with `repeated`). So when the original request never reached the server there is nothing to replay and nothing to recover, and once anything else in the installation has changed the retry is answered `revision_conflict` every time. The person can retry forever, and in the settings store every further proposal and confirmation of the session is refused. In the step store the same happens to a recovery attempt that itself ended unknown, and it also disables recovery of the original.

# Facts read in the server (main be0c58a)

1. For a mutating command the command layer looks up the stored operation of the idempotency key before it compares the expected revision (`modules/commands/execute.go:344-361`).
2. The lookup has these answers (`modules/commands/store.go:387-407`): a different caller or body under the key is `idempotency_conflict`; an operation still running is `dependency_unavailable`; a stored operation older than the replay window is `idempotency_key_expired`; a stored operation is replayed; a full store is `dependency_unavailable`; otherwise a new operation is reserved and execution continues to the revision comparison.
3. For a mutating command `revision_conflict` is produced at that comparison (`execute.go:359-361`) or later by a handler. It is never produced before the lookup. The refusals produced before the lookup are `authority_unavailable`, `unauthenticated`, `rate_limited`, `unsupported_command`, `unsupported_capability`, `invalid_request`, `permission_denied`, `self_approval_denied`, `dependency_unavailable` and `internal_error` (`execute.go:195-341`).
4. When a handler has started applying and fails, the operation is stored as an internal error with an operation identifier (`execute.go:255-259`, `:426-431`), and the exact retry replays it. This is what the real-server run of HYB-341 observed for the staged fault after a channel admission.
5. A refusal is not stored: the reserved operation is abandoned (`execute.go:260-261`).
6. Stored operations are swept after the retention period; the lookup sweeps first (`store.go:390`). The installation defaults are a replay window of one hour and a retention of eight hours (`apps/api/server/server.go`, `CommandLimits`). A full store answers `dependency_unavailable`; nothing is evicted early for capacity (`store.go:401`). The limits accept a retention as short as one hour (`modules/commands/types.go:263`, `:290`).
7. Before the command layer runs, the server's own boundaries can refuse with `evidence_stale`, `prerequisite_failed` and `reauthentication_required` (`apps/api/server/commands.go:228-273`, `financialissuancesetupcontext.go:20-95`). None of them produces `revision_conflict`; its only other producer serves contract lifecycle commands (`contracts.go:597`).
8. Handler discipline: the issuance handlers return a named refusal only before they change anything, and an error after an effect is not a named refusal, so it is stored (read in `financialissuanceconfigurationeffect.go:39-42`, `financialissuanceprotectedwrites.go:375-389`, `financialissuancetermwrites.go:205-229`, `financialissuanceattestation.go:216-221`, `:252-256`). The module states `proposal_unknown`, `confirmation_unknown` and `admission_unknown` arise this way, so their exact retry replays or expires and never reaches the revision comparison. The rule depends on this discipline.
9. The installation revision only increases (`apps/api/server/commands.go:225`, `:276`). So a delayed original that arrives after its retry was refused is refused as well.
10. Stored operations live in memory only (`modules/commands/registry.go:42`, `:65`). Issuance state is built in memory (`apps/api/server/approvalseed.go:101-105`), and no persistence of issuance state was found under the installation data directory. Inferred from this, not verified: a restart resets both together, so it does not leave an applied write without its operation.
11. The idempotency key is scoped by installation, client, command and key, and the client is one constant for the console (`apps/api/server/commands.go:210`). A retry reuses the stored intent, so it cannot miss a stored operation by a changed scope; another person under the same key is answered `idempotency_conflict`.

# What follows

An exact retry that is answered `revision_conflict` passed the lookup without finding an operation under its key: not a stored result, not one in flight, not an expired one. Within the retention period that means the original request was never admitted as an operation, so it applied nothing. After the retention period the same answer proves nothing, because a stored operation would have been swept.

No other refusal supports this conclusion. Those produced before the lookup say nothing about the key; `dependency_unavailable` may mean the original is still running; `idempotency_key_expired` means an operation exists whose result can no longer be replayed; a replayed internal error means the effect may have happened.

# Proposed rule

The console settles an unknown attempt as "not recorded" only when all of these hold:

- its exact retry is answered `revision_conflict`, as a clean refusal without an operation identifier;
- the attempt has never carried an operation identifier, in any earlier answer;
- less than thirty minutes have passed since the attempt was first sent, measured by wall-clock time kept with the entry, and the measured time is not negative.

Then:

- the entry leaves the unknown phase and no longer blocks other writes;
- it stays in the list with a sentence of its own, distinct from an ordinary refusal: the service has no record of this request; the settings have changed since it was prepared; check the current records before preparing it again. The sentence states what was observed and does not assert more;
- nothing is resent automatically, and the kept draft is not sent again without a new review step.

Thirty minutes is half the shortest retention the limits accept and a sixteenth of the default. Wall-clock time is used, and not a timer that stops while the machine sleeps, because a sleep longer than the retention would otherwise go unnoticed; a clock that was set back shows as a negative time and keeps the attempt unknown. An attempt older than the bound stays unknown.

In every other case the attempt stays unknown, exactly as today.

In the step store the rule applies to a recovery attempt as well. Once a recovery attempt is settled as not recorded, the original is again the only unknown entry, and its "recover" becomes available with a freshly read revision.

# What the rule does not settle

- An attempt whose retries keep failing before the lookup (a lasting permission or authentication refusal), one answered `idempotency_key_expired`, and one older than the bound remain unknown and keep blocking until the page is reloaded. A way for a person to set such an attempt aside after reading the current records would end that, but it lets an issuance setup write stay unconfirmed by a person's choice. That is a product decision for the owner and is not proposed here.
- The lower findings of the ticket (draft not cleared after a recorded settings write, the wording of a step-up refusal, entries never pruned, the store key after signing in again) are separate changes.

# Verification planned for the implementation

- Unit tests of both stores: `revision_conflict` on a retry inside the bound settles as not recorded and unblocks; the same answer outside the bound, and every other answer, stays unknown; a first send answered `revision_conflict` is an ordinary refusal as today; in the step store a settled recovery attempt re-enables recovery of the original.
- Server tests that pin the facts the rule rests on: an exact retry of a request that was never admitted, after the revision moved, is answered `revision_conflict` and stores nothing (facts 1 and 5); an exact retry of a stored operation is replayed although the revision moved; with the staged faults after an effect, the exact retry replays the stored internal error and is not answered `revision_conflict` (fact 8); the default retention is at least twice the bound (fact 6); the revision does not decrease (fact 9).
- A real-server browser run that stages a request lost before it reaches the server, changes the installation, and shows the retry settling.
- Copy review of the new sentence in three languages.

# Residual risk

- A server clock that jumps forward within the thirty minutes, unseen by the console, by more than the retention less the time already elapsed sweeps a stored operation; a retry after that would be read as not recorded. At the default retention that is a step of about seven and a half hours; at the one-hour minimum it is a step of just over thirty minutes. No bound removes this. The server follows forward wall-clock steps of up to twenty-four hours (`apps/api/server/clock.go:76-77`).
- `flight.applying` is cleared one statement before the operation is completed (`modules/commands/execute.go:386`, `:389`). A panic between the two would abandon an applied write. No source of such a panic was found. Moving the clearing after the completion is a small server follow-up, outside this design.
- Fact 8 is a discipline of the handlers, not an enforced property. A future handler that returns a named refusal after an effect would break the rule; the server tests above guard the present handlers only.

# Design review

Independent design review on 2026-11-24 (independent-reviewer role, AI, by reading; nothing executed): approve with conditions. No path was found under the default limits in which the rule is wrong. A bounded re-review of the incorporated text gave approval, with two corrections of wording that are made above (fact 10 marked as partly inferred; the clock residual stated with its real trigger). Its five conditions are incorporated above: settle only on a clean refusal for an attempt that never carried an operation identifier; measure by wall clock and keep a negative time unknown; a bound below the shortest accepted retention; the added facts 7 to 11 with tests that pin them; and the residual risks stated. The wording was reduced to what was observed, as the review suggested.

# Assumptions and unknowns

- Whether the replay and retention periods are meant to become installation settings is not known to this document. The limits do not accept a retention below one hour, which the bound respects.
- Not verified: whether sessions survive a service restart. If they do not, the console is signed out and its entries are no longer reachable.
- Not read: `financialissuanceprotectedrules.go` and `financialissuancetermbinding.go`, so fact 8 is read for the wrappers that call them and not for those two files. The implementation must read them before relying on fact 8.
- Not read: whether any issuance setup handler can itself return `revision_conflict`. If one does, it does so after the lookup, so the conclusion is unchanged.
- Not verified by execution: everything above rests on reading.
