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

# Server facts pinned by tests (2026-11-24)

`apps/api/server/issuancesetupexactretry_test.go`, run and passing with the whole `apps/api/server` package:

- `TestIssuanceSetupExactRetryOfUnadmittedRequestIsRevisionConflict`: for a channel registration and a change request, a request that was never sent, and one that was refused before any effect, is answered `409 revision_conflict` on its exact retry after the revision moved, twice in a row, without an operation identifier and without any change of state or revision; the same write prepared again from the current revision then succeeds. This executes facts 1, 3 and 5.
- `TestIssuanceSetupExactRetryOfStoredEffectIsReplayedAfterRevisionMoved`: for the same two writes, a fault after the effect is answered with an internal error carrying an operation identifier, and after the revision moved the exact retry replays that stored answer and is not answered `revision_conflict`. This executes facts 4 and 8 for these two handlers. The existing `TestFinancialProtectedWritesFaultOutcomes` covers the replay for every protected write and every staged fault point.
- `TestCommandRetentionCoversTheNotRecordedBound`: the default retention is at least twice the thirty-minute bound and the replay window at least the bound, so a change of the defaults that undercuts the rule fails a test.

Not pinned by a test: the settings proposal and confirmation handlers and the term handlers (the property tested belongs to the command layer, which they share, but their fault points were not staged here); fact 9, that the revision only increases; fact 10, the reset on restart.

# Implementation, step store (2026-11-24)

The rule is implemented in `setup/step-store.ts`, which carries the writes of rules, channels, protected changes, terms and statements. The settings store (`write-store.ts`, settings proposals and confirmations) is not changed yet.

What was built, and where it is stricter than the proposal above:

- A settled attempt becomes a refused entry marked `noRecord`. Its lines in the results list are "The retry was refused.", then "The service has no record of this request. Records changed after this request was reviewed. Check the current records, then review and send it again if it is still needed.", then the code. It does not show the ordinary line "Refused. This request recorded nothing.", which claims more than was observed.
- The thirty minutes are required on two clocks at once: wall-clock time since the first send, and a monotonic timer started at the first send. Either one alone past the bound, a negative reading, or a missing start keeps the attempt unknown.
- A settled entry of another contract stays listed with its sentence, like an unknown one.

Fact 8, completed by the change review: `financialissuanceprotectedrules.go` and `financialissuancetermbinding.go` do return named refusals after an effect, but their wrappers compare the state before and after and turn such a case into an uncertain outcome that is stored with an operation identifier (`financialissuanceprotectedwrites.go:375-383`, `financialissuancetermwrites.go:205-223`). So the fact holds through the wrappers, not in those two files by themselves. All three kinds of step store send mutating `issuance_setup.*` commands with an expected revision and an idempotency key.

Checks: `tsc` clean; console tests 1353, pass 1350, fail 0, skipped 3, with three new tests of the store (every clause of the rule, both clocks, an operation learnt on a later retry, recovery of the original available again after a settled recovery attempt). Rendered, against a console built from the change: five O03 write checks, exit code 0 each, and `TestIssuanceSetupActualScreenBrowser` on a real server in en, ko and ar. None of them stages a request lost before it reaches the server, so the settled state itself has no rendered or real-server run yet.

Reviews (AI, by reading; nothing executed by the reviewers): independent change review, pass with one condition (a settled entry of another contract vanished from the list), corrected; bounded re-review, pass. Copy review, ui-designer role: fail with two findings (the sentence said the settings had changed although the revision is installation-wide, and used a verb the screen does not use), corrected with its wording; re-review, pass.

Added residual risk: on a platform whose monotonic timer pauses while the machine sleeps, a sleep longer than the retention together with a wall clock set back into the window would still read as fresh.

Open observation from the change review, not changed: when the original attempt is retried and settled while its recovery attempt is still unknown, the original shows the note that its statement was not used, although the statement is withheld from reuse until the recovery settles.

Still to do after this step: see the next section.

# Implementation, settings store (2026-11-24)

The same rule is implemented in `setup/write-store.ts`, which carries settings proposals and confirmations. This is the store in which one unknown attempt refuses every further proposal and confirmation of the session, so a settled attempt now ends that block. The condition, the two clocks and the sentences are the same as in the step store; the entry keeps the time of its first send, which this store did not record before.

Fact 8 for these two commands, from the change review: every state change of the proposal and confirmation handlers is marked before it is made, any error after a mark becomes an uncertain outcome stored with an operation identifier, and the module states `proposal_unknown` and `confirmation_unknown` are written only after such a mark (`financialissuanceconfigurationwrite.go:125-128`, `:141-147`, `:161`, `:219-229`, `:238`; `financialissuanceconfigurationeffect.go:39-42`). Their exact retry therefore replays or expires and does not reach the revision comparison.

Checks: `tsc` clean; console tests 1354, pass 1351, fail 0, skipped 3, with one new test of the store: it settles inside the bound and another request is then admitted, an unsettled entry still answers that an operation is unresolved, eleven cases stay unknown, and a first send is an ordinary refusal. Rendered, against a console built from the change: three O03 setup checks, exit code 0 each, and `TestIssuanceSetupActualScreenBrowser` on a real server. As before, none stages a request lost before it reaches the server.

Review: independent change review (AI, by reading; nothing executed): pass, no blocking finding. The sentences are the ones reviewed for the step store; they had no separate copy review here.

With both stores done, items 1 and 2 of the ticket are implemented. Still to do under this ticket: a real-server run that stages a request lost before it reaches the server and shows the settled state on the screen; the lower findings (the settings draft kept after a recorded write, the wording of a step-up refusal, entries never pruned, the store key after signing in again); and the owner's decision on setting aside an attempt that this rule cannot settle.

# Real-server run with a lost request (2026-11-24)

`TestIssuanceSetupActualScreenBrowser` now begins with a phase that stages the case this ticket is about, in en, ko and ar, on an isolated in-process server with a console built from main f9e27d5 (headless Chromium, synthetic data; not an owner acceptance):

1. The first governor fills and sends a proposal of the protected settings rules. The check drops that one request in the browser before it is forwarded, so the service never receives it. The screen shows the outcome as unknown with "Retry exact request", and the rules form says that further writes are blocked.
2. The test moves the installation revision by an unrelated change.
3. The governor presses "Retry exact request". The service answers `409 revision_conflict`; this is the only refusal of the phase. The entry then reads "The retry was refused.", "The service has no record of this request. …" and the code; the retry and recover buttons are gone; the form is no longer blocked and offers the proposal again.
4. The server holds no rules and no statement afterwards.

The later phases of the same test then propose and confirm the rules normally, so the settled attempt left nothing behind that disturbs them.

This run exercises the step store. The settings store has the same rule and its unit tests but no real-server run with a lost request.

# Item 3, the settings draft after a recorded write (2026-11-24)

Confirmed by reading: `SetupEditor.submit` never reset its draft, so after a recorded proposal or confirmation the reason, the sample choice and the languages stayed, the unload guard of the settings form stayed armed, and the draft kept the identifier of the configuration just recorded for the next "New proposal".

Now, when the write that was just sent is recorded, the editor starts a fresh draft with a new identifier. A refused write and one whose outcome is unknown keep the draft, so nothing a person entered is lost before it took effect.

Checks: `tsc` clean; console tests 1355, pass 1352, fail 0, skipped 3, with one new test of the editor (cleared and renewed after a recorded write; kept after a refusal and after an unknown outcome). Rendered: `tools/o03-setup-parts-qa.mjs` and `o03-trust-states-qa.mjs` against a console built from the change, exit code 0 each; neither sends a settings write, so the cleared form itself has no rendered run. No review of this change.

Not decided here: a write settled as "no record" keeps the draft, like any refusal, so the person can review and send it again.

# Item 4, a refusal for a missing second factor (2026-11-24)

Confirmed by reading, and moved to HYB-354 because its remedy is a change to the error contract of every command and to authentication behaviour, not to the O03 write machinery.

When the active authentication policy demands a second factor for an action, the command endpoint refuses the command with `prerequisite_failed`, without fields, in the same shape as a domain refusal for a failed prerequisite (`apps/api/server/stepup.go:78-91`, `:112-127`; `commands.go:240-242`). The console cannot tell them apart. For an O03 step write it therefore shows the domain's list of possible causes, none of which is the real one. The settings form does learn the cause, because its action read reports `authentication_policy_unsatisfied` before the write is sent; the step writes have no such read.

Offering the identity action for it, as the finding first suggested, would not help: signing in again does not supply a second factor, and the pre-launch service has none to offer.

Nothing is changed for this item under HYB-353.

# Item 5, a request left unresolved before signing in again (2026-11-24)

Confirmed by reading: when a session ends the console shows the sign-in form on the same page and sets the new session without reloading (`apps/console/src/service/Service.tsx:497`), and a new sign-in has a new CSRF token. The step stores are keyed by person, workspace and that token, so after signing in again a request left pending or unknown was listed nowhere, while the unload prompt still fired for it. It cannot be retried either: its request is bound to the old session.

Corrected for the step store: the results list of a store also shows the unresolved requests of stores of the same kind, person and workspace from an earlier sign-in, read-only, with the action, the target, the contract, the settings, the sent time and this sentence: "Sent before you signed in again. Its outcome is not known here, and it can no longer be retried or recovered from this page. Check the current records. This entry, and the prompt when you leave the page, stay until the page is reloaded." Another person who signs in on the same page does not see them.

Checks: `tsc` clean; console tests 1356, pass 1353, fail 0, skipped 3, with one new store test (listed for the same person and kind, not for another kind or person, and not retryable from the new store). Rendered, against a console built from the change: five O03 write checks, exit code 0 each, and the real-server issuance setup test. None of them ends a session and signs in again, so the listed entry itself has no rendered run. Copy and state review, ui-designer role (AI, by reading): fail with one finding (the line above tells the person to settle the request before leaving, which is impossible here, and the persisting prompt was unexplained), corrected with its wording; re-review pass. No independent change review.

Left open: the settings store has the same keying and is not changed, so an unresolved settings proposal or confirmation of an earlier sign-in is still not listed; a request that was still pending when the session ended and settles afterwards in the old store is not re-rendered; making such a request retryable after signing in again would need the write machinery to carry a request across sessions and is not proposed.

# Item 5 for the settings store (2026-11-24)

The settings store lists the same way: an unresolved settings proposal or confirmation sent before the same person signed in again is shown read-only in the results of the settings form. Its sentence differs from the step store's in two points that are true only here. The settings form's unload guard looks at the current session's store only, so the entry does not keep the leave prompt armed and the sentence does not say so. And an attempt that the service recorded with an unknown outcome appears in the form's "Start from" list, where the same person can recover it in the new session, so the sentence points there: "Sent before you signed in again. Its outcome is not known here, and this request can no longer be retried from this page. Check “Start from” above: if the service lists these settings with an unknown outcome, select them there to recover the recorded attempt. This entry stays until the page is reloaded."

Checks: `tsc` clean; console tests 1357, pass 1354, fail 0, skipped 3, with one new store test. No rendered run of the listed entry. Copy review, ui-designer role (AI, by reading): fail with one finding (the control was not named, the next step was missing, and the Arabic verb differed from the form's), corrected with its wording, which was adopted without a further review. No independent change review.

Not checked: whether such a row can sit on a page of the "Start from" list that is not loaded.
