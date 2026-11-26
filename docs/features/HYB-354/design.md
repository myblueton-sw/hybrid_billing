---
type: Design
title: Telling a refusal for a missing second factor apart from a domain prerequisite refusal
description: Facts about the step-up refusal, two ways to make it distinguishable, a recommendation, and what must be decided and verified before code changes.
status: draft
---

# Outcome and ownership

HYB-354, machine:mac_mini, 2026-11-24. This document proposes a change to the public error contract of the command endpoint and to how consoles word one refusal. It changes no code. Because it touches authentication behaviour and every command's error contract, it needs the owner's choice between the two options before implementation.

The refusal it concerns cannot be reached through the product today (fact 7), so the change is a preparation for the time a second factor exists, not the correction of something a person can meet now.

# The problem

When the authentication policy demands a second factor for an action, the service refuses the action with the same answer it gives for a failed domain prerequisite. A screen cannot tell them apart and explains the refusal with its own domain causes, which are not the cause. On the O03 rules step, for example, a person reads that rules may already exist or the settings may no longer be confirmed, and that the service does not say which.

# Facts read in main 2e99c7a

1. `stepUpCheck` returns `prerequisite_failed` when the terms for the action demand a second factor, before it looks at the age of the authentication (`apps/api/server/stepup.go:78-91`). The demand comes from a step-up rule of the active or candidate policy (`stepUpTerms`, `tighten`, `:51-76`).
2. The command endpoint turns that into a refusal with status 422, next action `correct_request`, no retry and no fields (`stepUpRefusal`, `refusalOf`, `:112-127`; applied to every command at `apps/api/server/commands.go:239-243`). `stepUpRoute` gives the same answer on the route that ends sessions (`:130-139`, `selfend.go:292`). These are the only two producers. The refusal is built by hand and not through the command layer's error table, and `stepUpRoute` writes through a helper that has no parameter for fields (`errors.go:75-77`).
3. Four reads call the same check before a write. Two of them name the cause distinctly, in a field of their own answer and not as an error: the O03 settings action read and the issuance draft context report `step_up: authentication_policy_unsatisfied` (`financialissuancesetupaction.go:61-63`, `financialissuancedraftcontext.go:36-37`). The two financial prepare reads pass the outcome through as `prerequisite_failed` (`financialprepare.go:77-78`, `financialfinalprepare.go:41-42`), so they do not tell it apart either.
4. The error body of a command allows `fields`, an array of strings, for any code and without a constraint on its items (`modules/commands/openapi.go:131-133`). It already carries names that are not fields of the body: `request_id`, `target_id`, `rationale`, `not_recorded` (`execute.go:55-80`) and `after` on read errors (`apps/api/server/openapi.go:79-80`). The command layer drops the fields of every handler refusal except `invalid_request` (`execute.go:408-420`), so a domain `prerequisite_failed` cannot carry a marker, which makes one on the step-up refusal unambiguous.
5. The set of error codes and their status, retry flag and next action is a fixed table (`modules/commands/types.go:58-79`).
6. Consoles validate a refusal strictly before they treat it as clean. Seven transports do so (`setup/mutation.ts:16-18`, `result-authority/mutations.ts:26-27`, `approval-inbox/final-request/client.ts:27`, `approval-inbox/drafts/mutation.ts:19`, `approval-inbox/results/mutations.ts:21-22`, `approval-inbox/documents/transport.ts:20-21`, `approval-inbox/decision-api.ts:89`), and the general client maps codes to failure kinds (`service/admin.ts:863-867`). A code, a next action or a property a table does not know turns the answer into an unknown outcome, not a refusal. The design review read all seven: each accepts `prerequisite_failed` at 422 with `correct_request` and a `fields` array of strings. They accept the body but do not keep `fields`: their error classes carry only the code, the effect and the operation (`setup/context.ts:4-6`).
7. The refusal is not reachable through the product today. The policy module accepts a step-up rule that demands a second factor (`modules/authpolicy/types.go:108`, `evaluate.go:55-56`), but the only proposal path builds its candidate without a second-factor requirement and with a step-up rule that does not demand one (`apps/api/server/trust.go:106-120`), and a policy is published only from there, from ledger replay or from the seed (`trust.go:301`, `trustledger.go:231`, `:249`). Such a policy exists only in tests or in a ledger record written outside the product.

# Options

## A. Keep the code, name the cause in `fields`

The refusal stays `prerequisite_failed` with its status and next action, and carries `fields: ["authentication_policy"]`.

- Compatible: a console that ignores `fields` behaves exactly as today, and all seven strict transports still read the refusal as clean (fact 6).
- Cost on the server: small. `fields` already carries names that are not request fields (fact 4); the contract text names the marker; `stepUpRoute` needs a way to write fields.
- Cost on the console: a screen that wants the specific sentence must first be able to see the marker, and the strict transports drop `fields` today. Each transport that should show it needs a change to carry the marker to its screen. On the O03 settings form no such change is needed, because its action read already returns the cause; the O03 step writes have no such read.

## B. A code of its own

The refusal becomes `authentication_policy_unsatisfied`, a name the product already uses in the four reads of fact 3, with status 403, no retry and next action `none`.

- Clear: the code says what happened, and the next action no longer tells the caller to correct a request that is not wrong.
- Cost: a new member of the public error enum. Every strict console transport must learn it in the same change, or a clean refusal is shown as an unknown outcome and, on the O03 stores, blocks further writes until settled. The general client's mapping and every screen's wording need an entry.
- Further places the design review found for this option: `refusalOf` (`stepup.go:114-122`), where an unmapped code falls through to 401 and "obtain a new access answer"; the four reads of fact 3, which compare against the present code; the console's list of decision outcomes (`approval-inbox/decision-api.ts:5`), which would reject a new value in the prepare reads; `my-security/session-intent.ts:37` and `service/admin.ts:865`, where an unknown 403 becomes "unavailable"; and tests that pin the present shape (`stepup_test.go:391`, `:398`). If the code were added to the command layer's own table, the tests that pin its nineteen codes and thirteen handler codes and the OpenAPI golden file change too; declaring it among the server's boundary codes, as `reauthentication_required` is (`errors.go:25-29`), avoids that.
- One console helper already treats this name as "needs authentication" and offers the identity action (`setup/write-copy.ts`, `needsAuthentication`). Signing in again does not supply a second factor, so that helper must stop treating the two alike.

# Recommendation

Option A, for the first step. It removes the false explanation with the smallest change to a contract every command shares, and it cannot turn a clean refusal into an unknown outcome on a screen that was not updated. Option B is the cleaner contract and can follow when the product has a second factor to offer, at which point the next action could become meaningful ("authenticate with a second factor").

With either option, the sentence a screen shows should say what was observed and what a person can do: the authentication policy requires a second factor for this action; this installation cannot provide one, so the action was refused and nothing was recorded; the policy is managed on the authentication screen.

Two other shapes were considered in the design review and are less compatible: a new next action fails five of the strict transports, and a new property such as a reason fails the list of allowed keys in six; either would turn the refusal into an unknown outcome on a console that was not updated.

# What must be decided

1. Whether to make this preparation now or when a second factor is introduced, since the refusal cannot be reached through the product today.
2. Option A or B.
3. When a second factor is introduced: whether a policy whose step-up rule demands one may be activated before any person can supply it. Not a question today, because the proposal path cannot express such a rule.

# Verification planned for the implementation

- Server tests: with a policy whose step-up rule demands a second factor, a command is refused with the chosen shape; without such a rule the refusal for an old authentication is unchanged (`reauthentication_required`); a domain `prerequisite_failed` carries no marker.
- The contract document and the generated OpenAPI state the marker or the code.
- Console tests per strict table: the refusal is read as clean; the O03 step stores keep it as a refusal and not as an unknown outcome.
- Screens: the specific sentence on the O03 step writes first, since that is where the false list of causes was found; a survey of the other screens that show a domain explanation for `prerequisite_failed`.
- A real-server browser run with such a policy active. Because the product cannot produce such a policy, the run needs a seeded ledger record or a test hook.
- Independent change review, because the change touches authentication behaviour and a public contract.

# Assumptions and unknowns

- Not read by the design review: the gate in `organizations/drafts-model.ts:140` that decides whether error fields mark form inputs; console tests and any server test asserting that a step-up refusal has no fields; the contract documents under `docs/`.
- Not surveyed: which screens besides O03 show a domain explanation for this refusal.
- Not read: whether the route that ends sessions shows the refusal sensibly; it shares the answer.
- Everything above rests on reading; nothing was executed.

# Design review

Independent design review on 2026-11-24 (independent-reviewer role, AI, by reading; nothing executed): approve with conditions. It confirmed that option A is compatible with all seven strict console transports and that the recommendation is sound, and required four corrections, all made above: two of the four reads pass the outcome through and do not name the cause; the strict transports accept the marker but drop it, so adopting the sentence is a change per transport; there are seven strict transports and not five; and the refusal is not reachable through the product today. The corrected text has not been reviewed again.
